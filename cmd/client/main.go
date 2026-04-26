// Command client provides a USB/IP-compatible client CLI.
package main

import (
	"log/slog"
	"os"

	"usb-quic/internal/app"
)

func main() {
	err := app.ExecuteCLI(app.RoleClient, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		logger().Error("command failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelInfo,
		ReplaceAttr: nil,
	}))
}
