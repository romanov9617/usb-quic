// Command usb-quic provides a USB/IP-compatible CLI.
package main

import (
	"fmt"
	"os"

	"usb-quic/internal/adapter/delivery/cli"
)

func main() {
	err := cli.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
