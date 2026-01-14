package therm

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"
)

//go:embed docs
var docs embed.FS

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

func startWeb(sensorhost string, sensorport int) {
	s_host = sensorhost
	s_port = sensorport
	docsFS, err := fs.Sub(docs, "docs")
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/ir", thermHandler)
	http.Handle("/", http.FileServer(http.FS(docsFS)))
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
