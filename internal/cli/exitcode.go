package cli

import (
	"errors"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// ExitCode returns the exit code of ralph for err, the error the root
// command's Execute returned; cmd/ralph/main.go exits with it after it prints
// err. Each code other than 0 and 1 marks an outcome a caller can tell from a
// failure without reading the message:
//
//   - 2: `ralph config get` was given a key that is not a key of one value
//     in ralph.toml (ErrConfigKey).
//   - 3: `ralph upgrade` finished but left paths in unresolved drift
//     (ErrUpgradeDriftRemaining, docs/specs/2026-08-17-overlay-scaffold-v2.md
//     FR-4); `ralph config get` could not read ralph.toml or found it invalid
//     (ErrConfigLoad).
//   - 4: `ralph org wait --inbox --director` ended because a cross-org watch
//     that accepted work no longer does (org.ErrDirectorWaitNoWatch).
//   - 5: `ralph org wait --inbox --director` was refused or ended because its
//     registration is not the current one (org.ErrDirectorWaitNotCurrent).
//   - 6: `ralph org regate` under [org.gate].regate_checks = "ci" found the
//     checks of the branch's PR not done, also once --wait-checks ran out,
//     and recorded nothing (ErrRegateChecksPending, plan
//     docs/plans/active/2026-10-29-regate-ci-checks.md, D4).
//
// nil is 0 and any other error 1. Each sentinel is matched with errors.Is, so
// a wrapped one counts too.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrConfigKey):
		return 2
	case errors.Is(err, ErrUpgradeDriftRemaining), errors.Is(err, ErrConfigLoad):
		return 3
	case errors.Is(err, org.ErrDirectorWaitNoWatch):
		return 4
	case errors.Is(err, org.ErrDirectorWaitNotCurrent):
		return 5
	case errors.Is(err, ErrRegateChecksPending):
		return 6
	default:
		return 1
	}
}
