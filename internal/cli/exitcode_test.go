package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// ExitCode (plan 2026-10-19-org-director-wait, D4 and AC6): nil is 0, the
// upgrade drift 3, the director wait's watch change 4 and its registration
// that is not current 5, a ci regate's checks that are not done 6 (plan
// 2026-10-29-regate-ci-checks, D4), each also when wrapped, and any other
// error 1.
func TestExitCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"upgrade drift", ErrUpgradeDriftRemaining, 3},
		{"upgrade drift wrapped", fmt.Errorf("%w (2 path(s)); see docs/reports/upgrade-x.md", ErrUpgradeDriftRemaining), 3},
		{"no watch accepts work wrapped", fmt.Errorf("org: wait --inbox --director: %w: none does now", org.ErrDirectorWaitNoWatch), 4},
		{"not the current registration wrapped", fmt.Errorf("org: wait --inbox --director: d1 is %w: no registration is current", org.ErrDirectorWaitNotCurrent), 5},
		{"regate checks pending", ErrRegateChecksPending, 6},
		{"regate checks pending wrapped", fmt.Errorf("org: regate: %w (PR #7), so nothing was recorded: run it again", ErrRegateChecksPending), 6},
		{"a plain error", errors.New("org: wait --inbox: no open inbox item arrived within 300 ms"), 1},
	} {
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("%s: ExitCode(%v) = %d; want %d", tc.name, tc.err, got, tc.want)
		}
	}
}
