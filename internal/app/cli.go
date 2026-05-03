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
	"usb-quic/internal/adapter/delivery/daemon"
	"usb-quic/internal/adapter/logging"
)

// Streams contains process streams used by CLI commands.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// ExecuteCLI builds the CLI through the DI container and executes it.
func ExecuteCLI(stdin io.Reader, stdout, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	streams := Streams{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}

	var command *cobra.Command

	application := fx.New(
		fx.NopLogger,
		fx.Supply(streams),
		fx.Provide(newLogLevel),
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

// ExecuteDaemon builds the daemon CLI through the DI container and executes it.
func ExecuteDaemon(stdin io.Reader, stdout, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	streams := Streams{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}

	var command *cobra.Command

	application := fx.New(
		fx.NopLogger,
		fx.Supply(streams),
		fx.Provide(newLogLevel),
		fx.Provide(newDaemonCommand),
		fx.Populate(&command),
	)

	err := application.Err()
	if err != nil {
		return fmt.Errorf("build daemon container: %w", err)
	}

	command.SetContext(ctx)

	err = command.Execute()
	if err != nil {
		return fmt.Errorf("execute daemon: %w", err)
	}

	return nil
}

func newLogLevel() *logging.LevelVar {
	return logging.NewVerboseLevel(false)
}

func newRootCommand(
	streams Streams,
	level *logging.LevelVar,
) *cobra.Command {
	runtime := cli.Runtime{
		LogLevel: level,
	}

	return cli.NewRootCommandWithRuntime(streams.Stdin, streams.Stdout, streams.Stderr, runtime)
}

func newDaemonCommand(streams Streams, level *logging.LevelVar) *cobra.Command {
	runtime := daemon.Runtime{
		LogLevel: level,
	}

	return daemon.NewRootCommandWithRuntime(streams.Stdin, streams.Stdout, streams.Stderr, runtime)
}
