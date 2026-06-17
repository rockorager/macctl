package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type options struct {
	scope string
}

func NewRootCommand() *cobra.Command {
	opts := &options{scope: "user"}
	cmd := &cobra.Command{
		Use:   "macctl",
		Short: "systemctl-style launchd manager for macOS",
		Long:  "macctl provides systemctl-style commands for macOS launchd services, timers, and environment.",
	}

	cmd.PersistentFlags().StringVar(&opts.scope, "scope", "user", "target launchd scope: user or system")
	cmd.PersistentFlags().BoolFunc("user", "target the current user's launchd domain", func(string) error {
		opts.scope = "user"
		return nil
	})
	cmd.PersistentFlags().BoolFunc("system", "target the system launchd domain", func(string) error {
		opts.scope = "system"
		return nil
	})

	cmd.AddCommand(
		startCommand(opts),
		stopCommand(opts),
		restartCommand(opts),
		statusCommand(opts),
		enableCommand(opts),
		disableCommand(opts),
		listUnitsCommand(opts),
		daemonReloadCommand(opts),
		setEnvironmentCommand(opts),
		unsetEnvironmentCommand(opts),
		showEnvironmentCommand(opts),
		importEnvironmentCommand(opts),
	)
	return cmd
}

func requireArgs(name string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s requires at least one argument", name)
	}
	return nil
}
