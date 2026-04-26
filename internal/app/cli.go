// Package app wires application components.
package app

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"usb-quic/internal/adapter/delivery/cli"
)

// Streams contains process streams used by CLI commands.
type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// ExecuteCLI builds the CLI through the DI container and executes it.
func ExecuteCLI(stdin io.Reader, stdout, stderr io.Writer) error {
	streams := Streams{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}

	var command *cobra.Command

	application := fx.New(
		fx.NopLogger,
		fx.Supply(streams),
		fx.Provide(newRootCommand),
		fx.Populate(&command),
	)

	err := application.Err()
	if err != nil {
		return fmt.Errorf("build app container: %w", err)
	}

	err = command.Execute()
	if err != nil {
		return fmt.Errorf("execute cli: %w", err)
	}

	return nil
}

func newRootCommand(streams Streams) *cobra.Command {
	return cli.NewRootCommand(streams.Stdin, streams.Stdout, streams.Stderr)
}
