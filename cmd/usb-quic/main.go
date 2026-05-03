// Command usb-quic provides a USB/IP-compatible operator CLI.
package main

import (
	"os"

	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/di"
)

func main() {
	err := di.ExecuteCLI(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		logging.NewDefaultLogger(os.Stderr).Error("command failed", "error", err)
		os.Exit(1)
	}
}
