package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"example.com/therm"
)

func main() {
	host := *flag.String("host", "127.0.0.1", "network host with sensor tcp/ip socket server")
	port := *flag.Int("port", 9090, "network port with sensor tcp/ip socket server")
	flag.Parse()
	fmt.Println("host:", host, "port:", port)
	therm.StartWeb(host, port)
	sigs := make(chan os.Signal, 1)

	// Register the channel to receive notifications for SIGINT (Ctrl+C) and SIGTERM.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to signal when the program should exit.
	done := make(chan bool, 1)

	// Start a goroutine to listen for signals.
	go func() {
		sig := <-sigs // Block until a signal is received
		fmt.Println("\nReceived signal:", sig)
		done <- true // Signal the main goroutine to exit
	}()

	fmt.Println("Awaiting signal (e.g., press Ctrl+C to interrupt)...")
	<-done // Block the main goroutine until a signal is processed
	fmt.Println("Exiting.")
}
