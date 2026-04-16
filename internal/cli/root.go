package cli

import (
	"github.com/spf13/cobra"
)

// Version, GitCommit, and BuildDate are set via ldflags at build time.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ralph",
		Short: "Harness engineering scaffold and autonomous pipeline CLI",
		Long: `ralph is a CLI tool for harness engineering.
It scaffolds projects with best-practice Claude Code configurations,
manages template updates, and runs autonomous development pipelines.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newInitCmd(),
		newVersionCmd(),
		newStatusCmd(),
	)

	return root
}
