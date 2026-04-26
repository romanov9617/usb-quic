// Package app wires application components.
package app

import (
	"context"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"usb-quic/internal/adapter/delivery/cli"
	"usb-quic/internal/adapter/logging"
	"usb-quic/internal/adapter/usbip"
)

// Role identifies the binary role assembled by the container.
type Role string

const (
	// RoleClient configures client commands.
	RoleClient Role = "client"
	// RoleServer configures server commands.
	RoleServer Role = "server"
)

// Streams contains process streams used by CLI commands.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type runtimeConfig struct {
	Role Role
}

// ExecuteCLI builds the CLI through the DI container and executes it.
func ExecuteCLI(role Role, stdin io.Reader, stdout, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	streams := Streams{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	config := runtimeConfig{
		Role: role,
	}

	var command *cobra.Command

	application := fx.New(
		fx.NopLogger,
		fx.Supply(streams),
		fx.Supply(config),
		fx.Provide(newLogLevel),
		fx.Provide(newLogger),
		fx.Provide(newUSBIPTransport),
		fx.Provide(newRootCommand),
		fx.Populate(&command),
	)

	err := application.Err()
	if err != nil {
		return fmt.Errorf("build app container: %w", err)
	}

	command.SetContext(ctx)

	err = command.Execute()
	if err != nil {
		return fmt.Errorf("execute cli: %w", err)
	}

	return nil
}

func newLogLevel() *logging.LevelVar {
	return logging.NewVerboseLevel(false)
}

func newLogger(streams Streams, level *logging.LevelVar) *logging.Logger {
	return logging.NewTextLogger(streams.Stderr, level)
}

func newUSBIPTransport(logger *logging.Logger) *usbip.Transport {
	return usbip.NewTransport(usbip.WithLogger(logger))
}

func newRootCommand(
	streams Streams,
	config runtimeConfig,
	level *logging.LevelVar,
	transport *usbip.Transport,
) *cobra.Command {
	runtime := cli.Runtime{
		Role:           cli.Role(config.Role),
		LogLevel:       level,
		USBIPTransport: transport,
	}

	return cli.NewRootCommandWithRuntime(streams.Stdin, streams.Stdout, streams.Stderr, runtime)
}
