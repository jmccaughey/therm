package therm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var s_host = ""
var s_port = 0

func Therm(name string) string {
	message := fmt.Sprintf("Hi, %v. Welcome from therm!", name)
	return message
}

func thermHandler(w http.ResponseWriter, r *http.Request) {
	address := fmt.Sprint(s_host, ":", s_port)
	fmt.Println("got call for", r.URL.Path, "connecting to", address)
	conn, err := net.DialTimeout("tcp4", address, 20*time.Second)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("connected to", address)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	timeout := 20 * time.Second

	reader := bufio.NewReader(conn)

	for {
		conn.SetReadDeadline(time.Now().Add(timeout))

		line, err := reader.ReadString('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("Read timeout — waiting for more data")
				return
			}
			fmt.Println("Socket closed:", err)
			return
		}
		// matrix, err := parse2dArray(line)
		// if err != nil {
		// 	panic(err)
		// }
		// var scaled [][]float64 = ScaleBicubicNFNT(matrix, 10)
		// data, err := json.Marshal(scaled)
		// if err != nil {
		// 	panic(err)
		// }
		_, err = fmt.Fprintf(w, "%s", line)
		if err != nil {
			fmt.Println("write error:", err)
			return
		}
		flusher.Flush()
	}
}

func parse2dArray(jsonStr string) ([][]float64, error) {
	var matrix [][]float64
	err := json.Unmarshal([]byte(jsonStr), &matrix)
	if err != nil {
		return nil, err
	}
	return matrix, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got indexHandler call for:", r.URL.Path, r.RemoteAddr, r.UserAgent())
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	fmt.Fprint(w, page)
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got static call for:", r.URL.Path, r.RemoteAddr, r.UserAgent())

}

func startWeb(sensorhost string, sensorport int) {
	s_host = sensorhost
	s_port = sensorport
	targetURL, err := url.Parse("http://192.168.1.92:8000") // Replace with your target host and port
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}
	// Create a new reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	http.HandleFunc("/static", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying request from %s to %s%s", r.RemoteAddr, targetURL.Host, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ir", thermHandler)
	fmt.Println("about to ListenAndServe on http://0.0.0.0:8080")
	err = http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func StartWeb(sensorhost string, sensorport int) {
	fmt.Println("got StartWeb call sensor host:", sensorhost, "port:", sensorport)
	go startWeb(sensorhost, sensorport)
	fmt.Println("...web started")
}

var page = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>IR Sensor Heatmap</title>
<style>
  /* Make page and canvas fill the viewport, no scrollbars or gaps */
  html, body {
    height: 100%;
    margin: 0;
  }

  /* Use the viewport size — avoids issues with body margins/padding */
  canvas {
    position: absolute;
    width: 100vw;                /* fill viewport width */
    height: 100vh;               /* fill viewport height */
    touch-action: none;          /* optional: prevents double-tap zoom on mobile */
  }
  video.fullscreen {
      width: 400;
      height: 300;
      object-fit: cover; /* crop to fill */
    }
</style>
</head>
<body>
<canvas id="heatmap" width="100%" height="100%"></canvas>
<!--<img class="fullscreen" src="hike.jpg" alt="">-->
<video id="video1" class="fullscreen" autoplay muted playsinline webkit-playsinline controls></video>
<!-- Include chroma.js for color scaling -->
<script src="https://cdn.jsdelivr.net/npm/chroma-js@2.4.2/chroma.min.js"></script>
<script>
    // https://jsfiddle.net/jib1/yxbLvjm6/
  const pc1 = new RTCPeerConnection();

  const canvas = document.getElementById('heatmap');
  const ctx = canvas.getContext('2d');
  const width = canvas.width;
  const height = canvas.height;

  // Thermal gradient (cold → hot)
  const colorScale = chroma.scale([
    '#00008B10', // Dark blue (cold)
    '#0000FF24',
    '#00FFFF38',
    '#FFFF004C',
    '#FF450060',
    '#FF000088', // Red (hot)
    '#FFFFFF88'  // White (max)
  ]).domain([14, 26]); // IR intensity

  
  var alreadyUsingRearCamera = false;

  function drawHeatmap(data) {
      const rows = data.length
      const columns = data[0].length    
    ctx.clearRect(0,0,canvas.width,canvas.height);  
    for (let y = 0; y < rows; y++) {
      for (let x = 0; x < columns; x++) {
        const temp = data[y][x];
        ctx.fillStyle = colorScale(temp).hex();
          const cellWidth = width / columns;
          const cellHeight = height / rows;
        ctx.fillRect(x * cellWidth, y * cellHeight, cellWidth, cellHeight);
      }
    }
  }
    
    async function findRearCamera() {
        videoId = "";
        await navigator.mediaDevices
            .enumerateDevices()
            .then((devices) => {
              devices.forEach((device) => {
                if (device.kind.includes("ideoinput") && device.label.includes("ack")) {
                    videoId = device.deviceId;
                }
              });
            })
            .catch((err) => {
              console.error(err.name + " " + err.message);
            });
            return videoId;
    }
    
    async function switchToRearCamera() {
        // only possible after default camera has started (after user grants perms)
        // get ID of rear camera
        if (alreadyUsingRearCamera) {
          return;
        }
        rearCameraId = await findRearCamera();
        if (rearCameraId.length > 0) {
            console.log("switching to rear camera: " + rearCameraId)
            rearConstraints = { video: { deviceId: { exact: rearCameraId }}};
            rearStream = await navigator.mediaDevices.getUserMedia(rearConstraints);
            console.log("...got rear stream: " + rearStream)
            rearVideo = await rearStream.getVideoTracks()[0];
            console.log("...got rear video: " + rearVideo + "...")
            result = await pc1.getSenders()[0].replaceTrack(rearVideo);
            console.log("...got result " + result)
            video1.srcObject = rearStream;
            alreadyUsingRearCamera = true;
            startStream();
        }
    }
    
    (async () => {
      try {
        userMediaParams = {video: true, audio: false};
        const stream = await navigator.mediaDevices.getUserMedia(userMediaParams);
        for (const track of stream.getTracks()) {
          await pc1.addTrack(track, stream);
        }
        video1.srcObject = stream;
      } catch (e) {
        console.log(e);
      }
    })();

async function* makeTextFileLineIterator(fileURL) {
  const utf8Decoder = new TextDecoder("utf-8");
  console.log("fetching " + fileURL);
  let response = await fetch(fileURL);
  console.log("...fetched " + fileURL);
  let reader = response.body.getReader();
  let { value: chunk, done: readerDone } = await reader.read();
  chunk = chunk ? utf8Decoder.decode(chunk, { stream: true }) : "";
  console.log("got chunk: " + chunk);
  let re = /\r?\n/g;
  let startIndex = 0;

  for (;;) {
    let result = re.exec(chunk);
    if (!result) {
      if (readerDone) {
        break;
      }
      let remainder = chunk.substring(startIndex);
      ({ value: chunk, done: readerDone } = await reader.read());
      chunk =
        remainder + (chunk ? utf8Decoder.decode(chunk, { stream: true }) : "");
      startIndex = re.lastIndex = 0;
      continue;
    }
    yield chunk.substring(startIndex, result.index);
    startIndex = re.lastIndex;
  }
  if (startIndex < chunk.length) {
    // last line didn't end in a newline char
    yield chunk.substring(startIndex);
  }
}

async function startStream() {
    for await (let line of makeTextFileLineIterator(document.URL + "ir")) {
        drawHeatmap(JSON.parse(line));
    }
}

window.addEventListener('orientationchange', () => setTimeout(switchToRearCamera, 100));
</script>
</body>
</html>
`
