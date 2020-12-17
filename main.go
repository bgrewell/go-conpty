package main

import (
	"fmt"
	"github.com/bgrewell/go-conpty/libconpty"
	"log"
)

// main is a simple test application to check the functionality of libconpty. This is not an
// automated test, it is simply a way for a developer to manually test functionality.
func main() {

	pty, err := libconpty.NewConPty("cmd.exe", 90, 60)
	if err != nil {
		log.Fatalf("failed to get new ConPty: %v\n", err)
	}
	data := make([]byte, 1024)
	n, err := pty.Read(data)
	fmt.Printf("bytes read: %d\n", n)
	if err != nil {
		log.Fatalf("failed to read from ConPty: %v\n", err)
	} else {
		fmt.Println(string(data[:n]))
	}

	n, err = pty.Write([]byte("whoami\r\n"))
	fmt.Printf("bytes written: %d\n", n)
	if err != nil {
		log.Fatalf("failed to write to ConPty: %v\n", err)
	}

	n, err = pty.Read(data)
	fmt.Printf("bytes read: %d\n", n)
	if err != nil {
		log.Fatalf("failed to read from ConPty: %v\n", err)
	} else {
		fmt.Println(string(data[:n]))
	}

	pty.Close()
}
