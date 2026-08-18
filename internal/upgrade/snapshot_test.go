package upgrade

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// repoRoot resolves the repository root from this source file's location:
// internal/upgrade/snapshot_test.go → repo root is ../../. Mirrors
// internal/config/defaults_sync_test.go's repoRoot helper.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// TestSettingsSnapshotTemplate_MatchesSettingsJSON pins the settings.json
// snapshot template to stay byte-identical to templates/base/.claude/settings.json.
// The snapshot exists so a later upgrade's 3-way settings merge has an
// oldOwned side to diff against offline; since the whole settings.json file
// is ralph-shipped (owner=core), the "ralph-owned subset" is the entire
// file, so the two must never drift apart. If a file is absent (e.g. the
// package is vendored without the template tree), the test is skipped.
func TestSettingsSnapshotTemplate_MatchesSettingsJSON(t *testing.T) {
	root := repoRoot(t)
	settingsPath := filepath.Join(root, "templates", "base", ".claude", "settings.json")
	snapshotPath := filepath.Join(root, "templates", "base", ".ralph", "core", "settings.ralph.json")

	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("settings.json template not found at %s, skipping (vendored package?)", settingsPath)
		}
		t.Fatalf("reading %s: %v", settingsPath, err)
	}
	snapshotData, err := os.ReadFile(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("settings.ralph.json snapshot not found at %s, skipping (vendored package?)", snapshotPath)
		}
		t.Fatalf("reading %s: %v", snapshotPath, err)
	}

	if !bytes.Equal(settingsData, snapshotData) {
		t.Errorf("templates/base/.ralph/core/settings.ralph.json has drifted from templates/base/.claude/settings.json; they must be kept byte-identical (see internal/upgrade/snapshot.go doc comment)")
	}
}

func TestLoadSettingsSnapshot(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		dir := t.TempDir()
		want := []byte(`{"env":{"FOO":"bar"}}` + "\n")
		writeDiskFile(t, dir, SettingsSnapshotRelPath, string(want))

		got, found, err := LoadSettingsSnapshot(dir)
		if err != nil {
			t.Fatalf("LoadSettingsSnapshot: %v", err)
		}
		if !found {
			t.Fatal("found = false, want true")
		}
		if !bytes.Equal(got, want) {
			t.Errorf("content = %q, want %q", got, want)
		}
	})

	t.Run("absent", func(t *testing.T) {
		dir := t.TempDir()

		got, found, err := LoadSettingsSnapshot(dir)
		if err != nil {
			t.Fatalf("LoadSettingsSnapshot: %v", err)
		}
		if found {
			t.Fatal("found = true, want false")
		}
		if got != nil {
			t.Errorf("content = %q, want nil", got)
		}
	})
}
