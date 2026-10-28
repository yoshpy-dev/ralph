package main

import (
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
		// A few outcomes get a dedicated exit code so they stay
		// machine-detectable apart from a genuine execution error
		// (cli.ExitCode). `ralph upgrade` on a v2 layout can complete
		// successfully while leaving unresolved drift paths untouched
		// (non-destructive by design), which exits 3 ahead of `doctor
		// --strict` (Phase 5; docs/specs/2026-08-17-overlay-scaffold-v2.md
		// FR-4). `ralph org wait --inbox --director` exits 4 when the
		// cross-org watch stopped accepting work and 5 when its registration
		// is not the current one, so the director can tell both from a
		// failed wait (plan docs/plans/active/2026-10-19-org-director-wait.md,
		// D4). `ralph config get` exits 2 for a key that is not a key of one
		// value in ralph.toml and 3 when ralph.toml cannot be read or is
		// invalid, so a skill can tell both from an older ralph without the
		// verb (plan 2026-10-29-review-switches-config, D6). Any other error
		// exits 1.
		os.Exit(cli.ExitCode(err))
	}
}
