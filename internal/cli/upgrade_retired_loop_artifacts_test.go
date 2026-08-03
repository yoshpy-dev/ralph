package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ralph "github.com/yoshpy-dev/ralph"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// This file is a single-purpose regression test proving that `ralph
// upgrade`'s remove-path retires Ralph Loop scaffold artifacts that a
// pre-retirement `ralph` tracked. It necessarily names those retired
// artifacts' historical filenames as literal fixture strings, so it is
// listed alongside internal/cli/insights_test.go in
// tests/test-no-loop-references.sh's EXCLUDE_REGEX (same rationale: a test
// that proves retirement, not a live reference to the retired surface). Kept
// in its own file, rather than folded into the much larger cli_test.go, so
// that exclusion stays narrowly scoped to this one test instead of
// exempting an entire general-purpose test file from the guard.

// TestRunUpgrade_RemovesRetiredLoopArtifactsFromManifest is the automated
// smoke test for the "does `ralph upgrade` actually retire deleted Ralph
// Loop templates downstream" gap flagged as a known-gap by
// docs/reports/self-review-2026-08-03-org-runtime-retire-loop.md ("whether
// the `ralph upgrade` remove-path actually retires the deleted templates
// downstream (plan risk row 3) was not exercised").
//
// It deliberately uses the REAL embedded templates (github.com/yoshpy-dev/ralph
// root package's go:embed `TemplatesFS`, the same value cmd/ralph/main.go
// injects at runtime) instead of the fstest.MapFS mock the rest of this
// package's tests use, so the assertion is against the actual current
// template set — not a hand-picked fixture that could drift from reality.
//
// Scenario: a project scaffolded by a pre-retirement `ralph` tracks three
// Ralph Loop artifacts in its manifest (the orchestrator shell script, the
// per-slice pipeline shell script, and the loop skill body) that the current
// templates/base/ tree no longer contains (retired in
// refactor/org-runtime-retire-loop). Running the current `ralph upgrade`
// against that manifest must report all three as removed-from-template and
// drop their entries from .ralph/manifest.toml.
//
// Scope note: `ralph upgrade`'s ActionRemove handling is notify-only by
// design (see runUpgradeIOWithOptions's `case upgrade.ActionRemove` in
// upgrade.go) — it prints "review and delete manually" and drops the
// manifest entry, it does not delete the file from disk. This test proves
// that real, current contract (manifest-level retirement + user-facing
// notice), not silent disk deletion; the on-disk file is asserted to survive
// untouched to make that contract explicit rather than assumed.
func TestRunUpgrade_RemovesRetiredLoopArtifactsFromManifest(t *testing.T) {
	isolateGitConfig(t)
	scaffold.EmbeddedFS = ralph.TemplatesFS
	Version = "test-real-templates"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("executeInit (real templates): %v", err)
	}

	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	// Loop-era artifacts a pre-retirement `ralph init`/`ralph upgrade` would
	// have written and tracked, but which the current templates/base/ tree
	// no longer contains.
	loopArtifacts := []string{
		filepath.Join("scripts", "ralph-orchestrator.sh"),
		filepath.Join("scripts", "ralph-pipeline.sh"),
		filepath.Join(".claude", "skills", "loop", "SKILL.md"),
	}
	const loopArtifactContent = "#!/usr/bin/env sh\n# retired Ralph Loop artifact\n"

	for _, relPath := range loopArtifacts {
		diskPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(diskPath), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", relPath, err)
		}
		if err := os.WriteFile(diskPath, []byte(loopArtifactContent), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", relPath, err)
		}
		hash, err := scaffold.HashFile(diskPath)
		if err != nil {
			t.Fatalf("HashFile %s: %v", relPath, err)
		}
		m.SetFile(relPath, hash)
	}
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("Write manifest: %v", err)
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	for _, relPath := range loopArtifacts {
		slashPath := filepath.ToSlash(relPath)
		if !strings.Contains(out.String(), slashPath) {
			t.Errorf("upgrade output missing removal notice for %s; got:\n%s", slashPath, out.String())
		}
	}
	if !strings.Contains(out.String(), "removed from template") {
		t.Errorf("upgrade output missing 'removed from template' notice; got:\n%s", out.String())
	}

	m2, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest after upgrade: %v", err)
	}
	for _, relPath := range loopArtifacts {
		if _, ok := m2.Files[relPath]; ok {
			t.Errorf("%s should be dropped from manifest after upgrade removes it from the template", relPath)
		}
		// ActionRemove is notify-only: the file itself must survive on disk
		// untouched (see the ActionRemove scope note above) — `ralph upgrade`
		// never silently deletes user-visible files.
		if _, err := os.Stat(filepath.Join(dir, relPath)); err != nil {
			t.Errorf("%s should still exist on disk (ActionRemove does not delete files): %v", relPath, err)
		}
	}
}
