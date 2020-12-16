package main

import (
	"fmt"
	"github.com/bgrewell/go-conpty/libconpty"
	"log"
)

// main is a simple test application to check the functionality of libconpty. This is not an
// automated test, it is simply a way for a developer to manually test functionality.
func main() {

	pty, err := libconpty.NewConPty("powershell.exe", 90, 60)
	if err != nil {
		log.Fatalf("failed to get new ConPty: %v\n", err)
	}
	data := make([]byte, 1024)
	n, err := pty.Read(data)
	if err != nil {
		log.Fatalf("failed to read from ConPty: %v\n", err)
	} else {
		fmt.Println(string(data[:n]))
	}

	_, err = pty.Write([]byte("whoami\r\n"))
	if err != nil {
		log.Fatalf("failed to write to ConPty: %v\n", err)
	}

	n, err = pty.Read(data)
	if err != nil {
		log.Fatalf("failed to read from ConPty: %v\n", err)
	} else {
		fmt.Println(string(data[:n]))
	}

	pty.Close()
}
