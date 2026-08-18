package main

import (
	"errors"
	"fmt"
	"os"

	ralph "github.com/yoshpy-dev/ralph"
	"github.com/yoshpy-dev/ralph/internal/cli"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

func main() {
	// Inject build-time variables and embedded templates.
	cli.Version = Version
	cli.GitCommit = GitCommit
	cli.BuildDate = BuildDate
	scaffold.EmbeddedFS = ralph.TemplatesFS

	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		// `ralph upgrade` on a v2 layout can complete successfully while
		// leaving unresolved drift paths untouched (non-destructive by
		// design). That state is distinguishable from a genuine execution
		// error via a dedicated exit code so it stays machine-detectable
		// ahead of `doctor --strict` (Phase 5). See
		// docs/specs/2026-08-17-overlay-scaffold-v2.md FR-4.
		if errors.Is(err, cli.ErrUpgradeDriftRemaining) {
			os.Exit(3)
		}
		os.Exit(1)
	}
}
