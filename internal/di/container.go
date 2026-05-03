// Package di assembles application entrypoints.
package di

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
	"usb-quic/internal/config"
	runtimedaemon "usb-quic/internal/daemon"
)

// Streams contains process streams used by commands.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// ExecuteCLI builds and executes the usb-quic operator CLI.
func ExecuteCLI(stdin io.Reader, stdout, stderr io.Writer) error {
	const name = "cli"

	return execute(
		stdin,
		stdout,
		stderr,
		newRootCommand,
		name,
	)
}

// ExecuteDaemon builds and executes the daemon CLI.
func ExecuteDaemon(stdin io.Reader, stdout, stderr io.Writer) error {
	const name = "daemon"

	return execute(
		stdin,
		stdout,
		stderr,
		newDaemonCommand,
		name,
	)
}

func execute(
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	commandProvider any,
	name string,
) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
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
		fx.Provide(newLogger),
		fx.Provide(newDaemonRun),
		fx.Provide(commandProvider),
		fx.Populate(&command),
	)

	err := application.Err()
	if err != nil {
		return fmt.Errorf("build %s container: %w", name, err)
	}

	command.SetContext(ctx)

	err = command.Execute()
	if err != nil {
		return fmt.Errorf("execute %s: %w", name, err)
	}

	return nil
}

func newLogLevel() *logging.LevelVar {
	return logging.NewVerboseLevel(false)
}

func newLogger(streams Streams, level *logging.LevelVar) *logging.Logger {
	return logging.NewTextLogger(streams.Stderr, level)
}

func newRootCommand(streams Streams, level *logging.LevelVar) *cobra.Command {
	runtime := cli.Runtime{
		LogLevel: level,
	}

	return cli.NewRootCommandWithRuntime(
		streams.Stdin,
		streams.Stdout,
		streams.Stderr,
		runtime,
	)
}

func newDaemonRun(logger *logging.Logger) daemon.RunFunc {
	return func(ctx context.Context, cfg config.Daemon) error {
		return runtimedaemon.Run(ctx, cfg, logger)
	}
}

func newDaemonCommand(
	streams Streams,
	level *logging.LevelVar,
	run daemon.RunFunc,
) *cobra.Command {
	runtime := daemon.Runtime{
		LogLevel: level,
		Run:      run,
	}

	return daemon.NewRootCommandWithRuntime(
		streams.Stdin,
		streams.Stdout,
		streams.Stderr,
		runtime,
	)
}
