package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type attachOptions struct {
	remote string
	busid  string
}

type detachOptions struct {
	port string
}

type listOptions struct {
	local    bool
	remote   string
	parsable bool
}

type deviceOptions struct {
	busid string
}

func newAttachCommand(runtime Runtime, rootOpts *rootOptions) *cobra.Command {
	opts := attachOptions{
		remote: "",
		busid:  "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "attach -r HOST -b BUSID",
		Short: "Attach a remote USB device",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd
			_ = runtime
			_ = rootOpts

			return notImplemented("attach")
		},
	}

	cmd.Flags().StringVarP(&opts.remote, "remote", "r", "", "remote host")
	cmd.Flags().StringVarP(&opts.busid, "busid", "b", "", "remote USB device bus ID")

	return cmd
}

func newDetachCommand() *cobra.Command {
	opts := detachOptions{
		port: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "detach -p PORT",
		Short: "Detach a remote USB device",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented("detach")
		},
	}

	cmd.Flags().StringVarP(&opts.port, "port", "p", "", "imported USB device port")

	return cmd
}

func newListCommand(runtime Runtime, rootOpts *rootOptions) *cobra.Command {
	opts := listOptions{
		local:    false,
		remote:   "",
		parsable: false,
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "list [-l] [-r HOST]",
		Short: "List exportable or local USB devices",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd
			_ = runtime
			_ = rootOpts

			return notImplemented("list")
		},
	}

	cmd.Flags().BoolVarP(&opts.local, "local", "l", false, "list local USB devices")
	cmd.Flags().StringVarP(&opts.remote, "remote", "r", "", "list exportable devices on remote host")
	cmd.Flags().BoolVarP(&opts.parsable, "parsable", "p", false, "print parsable output")

	return cmd
}

func newBindCommand() *cobra.Command {
	opts := deviceOptions{
		busid: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "bind -b BUSID",
		Short: "Bind device to usbip-host.ko",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented("bind")
		},
	}

	cmd.Flags().StringVarP(&opts.busid, "busid", "b", "", "local USB device bus ID")

	return cmd
}

func newUnbindCommand() *cobra.Command {
	opts := deviceOptions{
		busid: "",
	}

	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	cmd := &cobra.Command{
		Use:   "unbind -b BUSID",
		Short: "Unbind device from usbip-host.ko",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented("unbind")
		},
	}

	cmd.Flags().StringVarP(&opts.busid, "busid", "b", "", "local USB device bus ID")

	return cmd
}

func newPortCommand() *cobra.Command {
	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	return &cobra.Command{
		Use:   portCommand,
		Short: "Show imported USB devices",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return notImplemented(portCommand)
		},
	}
}

func newVersionCommand() *cobra.Command {
	//nolint:exhaustruct // Cobra commands intentionally set only behavior relevant to this CLI.
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}
