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
//   - 3: `ralph upgrade` finished but left paths in unresolved drift
//     (ErrUpgradeDriftRemaining, docs/specs/2026-08-17-overlay-scaffold-v2.md
//     FR-4).
//   - 4: `ralph org wait --inbox --director` ended because a cross-org watch
//     that accepted work no longer does (org.ErrDirectorWaitNoWatch).
//   - 5: `ralph org wait --inbox --director` was refused or ended because its
//     registration is not the current one (org.ErrDirectorWaitNotCurrent).
//
// nil is 0 and any other error 1. Each sentinel is matched with errors.Is, so
// a wrapped one counts too.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrUpgradeDriftRemaining):
		return 3
	case errors.Is(err, org.ErrDirectorWaitNoWatch):
		return 4
	case errors.Is(err, org.ErrDirectorWaitNotCurrent):
		return 5
	default:
		return 1
	}
}
