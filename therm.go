package therm

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
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
		_, err = fmt.Fprintf(w, "%s", line)
		if err != nil {
			fmt.Println("write error:", err)
			return
		}
		flusher.Flush()
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("got indexHandler call for:", r.URL.Path, r.RemoteAddr, r.UserAgent())
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	fmt.Fprint(w, page)
}

func startWeb(sensorhost string, sensorport int) {
	s_host = sensorhost
	s_port = sensorport
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ir", thermHandler)
	fmt.Println("about to ListenAndServe on http://0.0.0.0:8080")
	err := http.ListenAndServe("0.0.0.0:8080", nil)
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

  .videocanvas {
            border: 1px solid black;
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
  .hidden {
            display: none;
  }
</style>
</head>
<body>
<canvas id="heatmap" width="100%" height="100%"></canvas>
<canvas id="videoCanvas" class="videocanvas" width="640" height="480"></canvas>
<!--<img class="fullscreen" src="hike.jpg" alt="">-->
<video id="video1" class="hidden" autoplay muted playsinline webkit-playsinline controls></video>
<!-- Include chroma.js for color scaling -->
<script src="https://cdn.jsdelivr.net/npm/chroma-js@2.4.2/chroma.min.js"></script>
<script>
    // https://jsfiddle.net/jib1/yxbLvjm6/
  const pc1 = new RTCPeerConnection();
  const video = document.getElementById('video1');
  const videocanvas = document.getElementById('videoCanvas');
  const canvas = document.getElementById('heatmap');
  const ctx = canvas.getContext('2d');
  const videoctx = videoCanvas.getContext('2d');
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
  let chunks = [];
  let thermChunks = [];
  var mediaRecorder;
  var recordingStart = null;
  var thermStart = null;
  var downloadPending = false;

  video.addEventListener('canplaythrough', () => {
      video.play();
  });

  video.addEventListener('play', () => {
    function drawVideoFrame() {
        // Draw the current frame of the video onto the canvas
        videoctx.drawImage(video, 0, 0, videoCanvas.width, videoCanvas.height);

        // Request the next frame to continue the animation loop
        requestAnimationFrame(drawVideoFrame);
    }

    // Start the initial animation frame
    requestAnimationFrame(drawVideoFrame);
  });

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
  thermStart = new Date();

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
    var out = chunk.substring(startIndex, result.index) + "\n";
    thermChunks.push(out);
    yield out;
    startIndex = re.lastIndex;
  }
  if (startIndex < chunk.length) {
    // last line didn't end in a newline char
    var out = chunk.substring(startIndex) + "\n";
    thermChunks.push(out);
    yield out;
  }
}

async function startStream() {
    for await (let line of makeTextFileLineIterator(document.URL + "ir")) {
        drawHeatmap(JSON.parse(line));
    }
}

function startVideoRecording() {
  // https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement/captureStream#browser_compatibility
  mediaRecorder = new MediaRecorder(videoCanvas.captureStream());
  mediaRecorder.ondataavailable = event => {
            console.log("got MediaRecorder data available");
            chunks.push(event.data);
            if (downloadPending) {
              doDownload();
            }
        };
  recordingStart = new Date();
  mediaRecorder.start();
}

function doDownload() {
  const blob = new Blob(chunks, { type: 'video/webm' });
  const videoURL = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = videoURL;
  a.download = recordingStart.toISOString() + '_recording.webm';
  a.click();
}

function downloadVideoRecording() {
  console.log("stopping MediaRecorder");
  downloadPending = true;
  mediaRecorder.stop();
}

function downloadTherm() {
  const blob = new Blob(thermChunks, { type: 'text/plain' });
  const textURL = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = textURL;
  a.download = thermStart.toISOString() + '_therm_recording.txt';
  a.click();
}

window.addEventListener('orientationchange', () => setTimeout(switchToRearCamera, 100));
</script>
</body>
</html>
`
