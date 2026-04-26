// Command server provides a USB/IP-compatible server CLI.
package main

import (
	"fmt"
	"os"

	"usb-quic/internal/app"
)

func main() {
	err := app.ExecuteCLI(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
