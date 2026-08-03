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
		Short: "Harness engineering scaffold and org-runtime CLI",
		Long: `ralph is a CLI tool for harness engineering.
It scaffolds projects with best-practice Claude Code configurations,
manages template updates, and coordinates autonomous multi-seat
org-runtime execution (ralph org).`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newInitCmd(),
		newUpgradeCmd(),
		newDoctorCmd(),
		newInsightsCmd(),
		newPackCmd(),
		newVersionCmd(),
		newStatusCmd(),
		newOrgCmd(),
	)

	return root
}
