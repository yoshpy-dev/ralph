package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// homePathCase is one of the six values plan 2026-11-01-org-ledger-gaps
// (AC2) runs through every check and expansion of "absolute or ~/", with
// what IsAbsOrHomePath and ExpandHomePath give for it.
type homePathCase struct {
	name, value string
	absOrHome   bool
	expanded    string
}

// homePathCases returns the six values with home as the home directory and
// abs as an absolute, clean path.
func homePathCases(home, abs string) []homePathCase {
	return []homePathCase{
		{name: "under home", value: "~/x", absOrHome: true, expanded: filepath.Join(home, "x")},
		{name: "a bare tilde", value: "~", expanded: "~"},
		{name: "another user's home", value: "~user/x", expanded: "~user/x"},
		{name: "relative", value: "rel/x", expanded: "rel/x"},
		{name: "absolute", value: abs, absOrHome: true, expanded: abs},
		{name: "empty", value: "", expanded: ""},
	}
}

// TestHomePath_ChecksAndExpansionsAgree covers AC2 (D2): for each of the six
// values, IsAbsOrHomePath and ExpandHomePath give the expected results, and
// the two checks of the personal file that call them agree with
// IsAbsOrHomePath: LocalOrgStateDir and Load (its [org].agmsg_home check)
// accept a value exactly when it is absolute or starts with ~/, or is empty
// (unset). LocalOrgStateDir returns the expansion ExpandHomePath gives, and
// Load keeps an accepted agmsg_home as written (the default for an empty one).
func TestHomePath_ChecksAndExpansionsAgree(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	abs := filepath.Join(t.TempDir(), "abs")

	for _, tc := range homePathCases(home, abs) {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsAbsOrHomePath(tc.value); got != tc.absOrHome {
				t.Errorf("IsAbsOrHomePath(%q) = %v, want %v", tc.value, got, tc.absOrHome)
			}
			if got, err := ExpandHomePath(tc.value); err != nil || got != tc.expanded {
				t.Errorf("ExpandHomePath(%q) = (%q, %v), want (%q, nil)", tc.value, got, err, tc.expanded)
			}
			accepted := tc.value == "" || tc.absOrHome

			configPath, localPath := localTestFiles(t, "", "[org]\nstate_dir = \""+tc.value+"\"\n")
			got, path, err := LocalOrgStateDir(filepath.Dir(configPath))
			if accepted && (err != nil || got != tc.expanded || path != localPath) {
				t.Errorf("LocalOrgStateDir with state_dir %q = (%q, %q, %v), want (%q, %q, nil)", tc.value, got, path, err, tc.expanded, localPath)
			}
			if !accepted && (err == nil || got != "") {
				t.Errorf("LocalOrgStateDir with state_dir %q = (%q, %v), want a refusal", tc.value, got, err)
			}

			configPath, _ = localTestFiles(t, "[org]\nmax_seats = 3\n", "[org]\nagmsg_home = \""+tc.value+"\"\n")
			cfg, err := Load(configPath)
			wantHome := tc.value
			if wantHome == "" {
				wantHome = Default().Org.AgmsgHome
			}
			if accepted && (err != nil || cfg.Org.AgmsgHome != wantHome) {
				t.Errorf("Load with agmsg_home %q = (agmsg_home %q, %v), want (%q, nil)", tc.value, cfg.Org.AgmsgHome, err, wantHome)
			}
			if !accepted && err == nil {
				t.Errorf("Load with agmsg_home %q: nil error, want a refusal", tc.value)
			}
		})
	}
}

// TestExpandHomePath_UserHomeDirFails covers AC2's failure half (D2): when
// os.UserHomeDir fails, ExpandHomePath returns its error for a value that
// starts with ~/, and LocalOrgStateDir reports that error in the text it
// had before the expansion moved here; every other value is returned
// unchanged with a nil error, since it needs no home directory.
func TestExpandHomePath_UserHomeDirFails(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	t.Setenv("HOME", "")
	_, homeErr := os.UserHomeDir()
	if homeErr == nil {
		t.Skip("os.UserHomeDir does not fail with HOME empty on this platform")
	}
	abs := filepath.Join(t.TempDir(), "abs")

	got, err := ExpandHomePath("~/x")
	if err == nil || err.Error() != homeErr.Error() || got != "" {
		t.Errorf("ExpandHomePath(%q) = (%q, %v), want (\"\", %v)", "~/x", got, err, homeErr)
	}
	for _, tc := range homePathCases("", abs) {
		if strings.HasPrefix(tc.value, "~/") {
			continue
		}
		if got, err := ExpandHomePath(tc.value); err != nil || got != tc.value {
			t.Errorf("ExpandHomePath(%q) = (%q, %v), want (%q, nil)", tc.value, got, err, tc.value)
		}
	}

	configPath, localPath := localTestFiles(t, "", "[org]\nstate_dir = \"~/x\"\n")
	value, path, err := LocalOrgStateDir(filepath.Dir(configPath))
	want := fmt.Sprintf("%s: [org].state_dir %q: %s", localPath, "~/x", homeErr)
	if err == nil || err.Error() != want || value != "" || path != localPath {
		t.Errorf("LocalOrgStateDir = (%q, %q, %v), want (\"\", %q, %q)", value, path, err, localPath, want)
	}
}
