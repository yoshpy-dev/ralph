package driver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestExpandTilde_AgreesWithConfigExpandHomePath covers plan
// 2026-11-01-org-ledger-gaps AC2 (D2) for the agmsg home: expandTilde gives
// what config.ExpandHomePath gives for the six values the checks of
// ralph.local.toml are tested with (~/x, ~, ~user/x, a relative path, an
// absolute one, and the empty string), and only ~/x changes.
func TestExpandTilde_AgreesWithConfigExpandHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	abs := filepath.Join(t.TempDir(), "abs")

	for _, tc := range []struct{ value, want string }{
		{"~/x", filepath.Join(home, "x")},
		{"~", "~"},
		{"~user/x", "~user/x"},
		{"rel/x", "rel/x"},
		{abs, abs},
		{"", ""},
	} {
		expanded, err := config.ExpandHomePath(tc.value)
		if err != nil {
			t.Fatalf("config.ExpandHomePath(%q): %v", tc.value, err)
		}
		if got := expandTilde(tc.value); got != expanded || got != tc.want {
			t.Errorf("expandTilde(%q) = %q, want %q (config.ExpandHomePath gives %q)", tc.value, got, tc.want, expanded)
		}
	}
}

// TestExpandTilde_KeepsTheValueWhenUserHomeDirFails covers the other half of
// AC2 (D2): when os.UserHomeDir fails, config.ExpandHomePath returns an
// error, and expandTilde returns its argument unchanged, as it did before
// the expansion moved to internal/config; so does ResolveAgmsgHome.
func TestExpandTilde_KeepsTheValueWhenUserHomeDirFails(t *testing.T) {
	t.Setenv("HOME", "")
	if _, err := os.UserHomeDir(); err == nil {
		t.Skip("os.UserHomeDir does not fail with HOME empty on this platform")
	}
	if _, err := config.ExpandHomePath("~/x"); err == nil {
		t.Fatalf("config.ExpandHomePath(%q): nil error, want the os.UserHomeDir failure", "~/x")
	}
	if got := expandTilde("~/x"); got != "~/x" {
		t.Errorf("expandTilde(%q) = %q, want it unchanged", "~/x", got)
	}
	if got := ResolveAgmsgHome("~/x"); got != "~/x" {
		t.Errorf("ResolveAgmsgHome(%q) = %q, want it unchanged", "~/x", got)
	}
}
