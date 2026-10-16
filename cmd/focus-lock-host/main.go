//go:build windows

// The browser starts this small console program, never the GUI. Stdout is
// reserved exclusively for framed Native Messaging messages.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Karench0/focus-lock/internal/bridge"
	"github.com/Microsoft/go-winio"
)

func run() error {
	if len(os.Args) < 2 || strings.TrimSuffix(os.Args[1], "/") != "chrome-extension://"+bridge.ExtensionID {
		return fmt.Errorf("untrusted extension origin")
	}
	name, err := bridge.PipeName()
	if err != nil {
		return err
	}
	timeout := 2 * time.Second
	c, err := winio.DialPipe(name, &timeout)
	if err != nil {
		return err
	}
	defer c.Close()
	// Closing this process's pipe on EOF also disconnects the browser port.
	go func() {
		reader := bufio.NewScanner(c)
		reader.Buffer(make([]byte, 4096), bridge.MaxMessage)
		for reader.Scan() {
			if bridge.WriteFrame(os.Stdout, reader.Bytes()) != nil {
				os.Exit(0)
			}
		}
		os.Exit(0)
	}()
	for {
		b, err := bridge.ReadFrame(os.Stdin)
		if err != nil {
			return err
		}
		_ = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
		if _, err = c.Write(append(b, '\n')); err != nil {
			return err
		}
	}
}
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}
