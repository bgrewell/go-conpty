package main

import (
	"fmt"

	"github.com/bgrewell/go-conpty/libconpty"
)

// main is a simple test application to check the functionality of libconpty. This is not an
// automated test, it is simply a way for a developer to manually test functionality.
func main() {

	pty, err := libconpty.NewConPty("powershell.exe", 90, 60)
	if err != nil {
		fmt.Errorf("failed to get new ConPty: %v", err)
	}
	data, err := pty.Read()
	if err != nil {
		fmt.Errorf("failed to read from ConPty: %v", err)
	} else {
		fmt.Println(string(data))
	}

	_, err = pty.Write([]byte("whoami\r\n"))
	if err != nil {
		fmt.Errorf("failed to write to ConPty: %v", err)
	}

	data, err = pty.Read()
	if err != nil {
		fmt.Errorf("failed to read from ConPty: %v", err)
	} else {
		fmt.Println(string(data))
	}

	pty.Close()
}
