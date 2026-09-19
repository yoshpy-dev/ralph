package org

// TestSendDefaults_DocsMatchConstant pins DefaultSendEnterDelayMS to the two
// hand-written doc families that state its value in prose (self-review
// LOW-7, docs/reports/self-review-2026-09-19-org-send-enter-timing.md): the
// /org skill and the codex-seat-permissions recipe. Neither surface derives
// its number from Go code (they are markdown prose, not templated), so
// nothing else catches drift if DefaultSendEnterDelayMS ever changes --
// this test is that catch. It follows the same repo-root-via-runtime.Caller
// pattern as internal/config/defaults_sync_test.go's TestDefaultsLockStep.
//
// If this test fails, update whichever of these six files still shows the
// old value, next to its own --enter-delay-ms mention:
//   - .claude/skills/org/SKILL.md                    ("<N>ms", no space)
//   - .agents/skills/org/SKILL.md                     (mirror)
//   - templates/base/.claude/skills/org/SKILL.md      (mirror)
//   - templates/base/.agents/skills/org/SKILL.md      (mirror)
//   - docs/recipes/codex-seat-permissions.md          ("<N> ms", with a space)
//   - templates/base/docs/recipes/codex-seat-permissions.md (template copy)
//
// The four skill mirrors are kept byte-identical by
// scripts/check-skill-sync.sh, and the two recipe copies by
// scripts/check-sync.sh -- so in practice only one root file needs a hand
// edit plus a sync-script run; this test is what keeps the *value* in that
// one root file in lock-step with the Go constant, which those sync scripts
// do not check.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sendDefaultsRepoRoot resolves the repository root from this source
// file's location. internal/org/send_defaults_sync_test.go -> repo root is
// ../../ (same depth as internal/config/defaults_sync_test.go's repoRoot).
func sendDefaultsRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func TestSendDefaults_DocsMatchConstant(t *testing.T) {
	root := sendDefaultsRepoRoot(t)

	type docSurface struct {
		relPath string
		marker  string // this doc's exact "<N><unit>" spelling of the default
	}
	surfaces := []docSurface{
		{filepath.Join(".claude", "skills", "org", "SKILL.md"), fmt.Sprintf("%dms", DefaultSendEnterDelayMS)},
		{filepath.Join(".agents", "skills", "org", "SKILL.md"), fmt.Sprintf("%dms", DefaultSendEnterDelayMS)},
		{filepath.Join("templates", "base", ".claude", "skills", "org", "SKILL.md"), fmt.Sprintf("%dms", DefaultSendEnterDelayMS)},
		{filepath.Join("templates", "base", ".agents", "skills", "org", "SKILL.md"), fmt.Sprintf("%dms", DefaultSendEnterDelayMS)},
		{filepath.Join("docs", "recipes", "codex-seat-permissions.md"), fmt.Sprintf("%d ms", DefaultSendEnterDelayMS)},
		{filepath.Join("templates", "base", "docs", "recipes", "codex-seat-permissions.md"), fmt.Sprintf("%d ms", DefaultSendEnterDelayMS)},
	}

	for _, s := range surfaces {
		path := filepath.Join(root, s.relPath)
		data, err := os.ReadFile(path)
		if err != nil {
			// A downstream repo that vendors internal/org without the
			// meta-repo's docs/skills tree should not fail this test --
			// see TestDefaultsLockStep's identical rationale. t.Skipf halts
			// the whole test here, so a missing first surface means the
			// remaining surfaces below are not checked either.
			t.Skipf("%s not found (%v) -- skipping doc/constant lock-step check (vendored repo?)", s.relPath, err)
		}
		text := string(data)
		if !strings.Contains(text, "--enter-delay-ms") {
			t.Errorf("%s: expected a --enter-delay-ms mention, found none -- has this doc's send section been removed or renamed?", s.relPath)
			continue
		}
		if !strings.Contains(text, s.marker) {
			t.Errorf("%s: expected to document the current default as %q (next to its --enter-delay-ms mention) -- "+
				"if DefaultSendEnterDelayMS changed, update this file (see this test's doc comment for the full list of six files to keep in lock-step)",
				s.relPath, s.marker)
		}
	}
}
