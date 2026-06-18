package cli

import (
	"github.com/spf13/cobra"

	"github.com/Gerry3010/pipepush/internal/config"
	"github.com/Gerry3010/pipepush/internal/tui"
)

// version is set at build time via -ldflags.
var version = "dev"

// Execute runs the root command.
func Execute(v string) error {
	if v != "" {
		version = v
	}
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "pipepush",
		Short:   "pipepush — get notified when your CI/CD pipelines finish",
		Version: version,
		Long: `pipepush sends end-to-end encrypted notifications when your CI/CD
pipelines succeed or fail — across any provider.

Run 'pipepush' with no arguments to launch the interactive TUI, or use the
subcommands below for scripting and CI/CD.`,
		// With no subcommand, launch the TUI.
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadClientConfig()
			if err != nil {
				return err
			}
			return tui.Run(cfg)
		},
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newWhoamiCmd(),
		newServerCmd(),
		newProjectsCmd(),
		newPipelinesCmd(),
		newTokensCmd(),
		newRunsCmd(),
		newSendCmd(),
	)

	return root
}
