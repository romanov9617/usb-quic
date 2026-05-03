// Command daemon provides a usbipd-like service entrypoint.
package main

import (
	"os"

	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/app"
)

func main() {
	err := app.ExecuteDaemon(os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		logging.NewDefaultLogger(os.Stderr).Error("daemon failed", "error", err)
		os.Exit(1)
	}
}
