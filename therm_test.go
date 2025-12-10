package therm

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"testing"
)

func TestTherm(t *testing.T) {
	StartWeb("127.0.0.1:9090")
	fmt.Println("Enter text:")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("You entered: %s", line)
}
