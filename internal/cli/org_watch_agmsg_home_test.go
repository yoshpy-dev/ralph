package cli

import (
	"path/filepath"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestAgmsgHomeIsRelative_ComplementsConfigIsAbsOrHomePath covers plan
// 2026-11-01-org-ledger-gaps AC2 (D2) for the CLI: for the six values the
// checks of ralph.local.toml are tested with, agmsgHomeIsRelative is true
// exactly for a value that is not empty and that config.IsAbsOrHomePath
// refuses, so the --config refusal of the watch and of director start and
// config.Load's refusal in ralph.local.toml agree.
func TestAgmsgHomeIsRelative_ComplementsConfigIsAbsOrHomePath(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "abs")
	for _, tc := range []struct {
		value    string
		relative bool
	}{
		{"~/x", false},
		{"~", true},
		{"~user/x", true},
		{"rel/x", true},
		{abs, false},
		{"", false},
	} {
		got := agmsgHomeIsRelative(tc.value)
		if got != tc.relative {
			t.Errorf("agmsgHomeIsRelative(%q) = %v, want %v", tc.value, got, tc.relative)
		}
		if complement := tc.value != "" && !config.IsAbsOrHomePath(tc.value); got != complement {
			t.Errorf("agmsgHomeIsRelative(%q) = %v, but config.IsAbsOrHomePath gives %v", tc.value, got, config.IsAbsOrHomePath(tc.value))
		}
	}
}
