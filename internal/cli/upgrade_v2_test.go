package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

// ---- shared v2-upgrade fixture builders ----
//
// These mirror setupTestEmbedFSV2 (init_v2_test.go) but are parameterized so
// a test can render an "old" generation with executeInit, then swap in a
// "new" generation and drive runUpgradeIOWithOptions against it — simulating
// template evolution across a `ralph upgrade` the way a real release bump
// would.

func v2AgentsMD(managed []byte) []byte {
	return []byte("# AGENTS.md\n\nProject notes go here.\n\n" +
		upgrade.BeginMarker("agents-md") + "\n" +
		string(managed) +
		upgrade.EndMarker + "\n")
}

func v2Gitignore(managed []byte) []byte {
	return []byte("# Project ignores\n" +
		upgrade.BeginMarkerStyled("gitignore", upgrade.BlockMarkerHash) + "\n" +
		string(managed) +
		upgrade.EndMarkerStyled(upgrade.BlockMarkerHash) + "\n")
}

// v2FixtureGen composes an embedded-templates fstest.MapFS for one
// "generation" (a specific template version's content) of a v2-layout
// project. Every field is independently overridable so each test only needs
// to specify what actually changes between generations.
type v2FixtureGen struct {
	agentsManaged    []byte
	gitignoreManaged []byte
	settingsJSON     []byte
	runVerifySh      []byte
	oldToolSh        []byte // included only when non-nil; absence across generations exercises core removal
	docsNotes        []byte // included only when non-nil
	docsNewNote      []byte // included only when non-nil; new-in-this-generation seed file
	docsGuide        []byte // included only when non-nil; identical across gen1/gen2 by default (v2DocsGuide) — a stable seed used to exercise "seed missing, recreated" without also triggering a template-changed advisory
	includeGolang    bool
	golangVerifySh   []byte
	golangReadme     []byte
	golangRuleMd     []byte
	preCommitGuardSh []byte // rendered to scripts/pre-commit-secret-guard.sh; the only one of the four managed git-hook guard scripts that varies across gen1/gen2 in these fixtures
}

func (g v2FixtureGen) build() fstest.MapFS {
	m := fstest.MapFS{
		"templates/base/AGENTS.md":                                  {Data: v2AgentsMD(g.agentsManaged)},
		"templates/base/.gitignore":                                 {Data: v2Gitignore(g.gitignoreManaged)},
		"templates/base/CLAUDE.md":                                  {Data: []byte("@AGENTS.md\n\n# Claude Code\n")},
		"templates/base/ralph.toml":                                 {Data: []byte("[pipeline]\nmodel = \"test\"\n[doctor]\nrequire_codex_cli = false\n")},
		"templates/base/.claude/settings.json":                      {Data: g.settingsJSON},
		"templates/base/.ralph/core/settings.ralph.json":            {Data: g.settingsJSON},
		"templates/base/.ralph/core/AGENTS.core.md":                 {Data: g.agentsManaged},
		"templates/base/scripts/run-verify.sh":                      {Data: g.runVerifySh},
		"templates/base/.ralph/local/verify.d/.gitkeep":             {Data: []byte("")},
		"templates/base/scripts/pre-commit-secret-guard.sh":         {Data: g.preCommitGuardSh},
		"templates/base/scripts/commit-msg-guard.sh":                {Data: []byte(testCommitMsgGuard)},
		"templates/base/scripts/prepare-commit-msg-secret-guard.sh": {Data: []byte(testPrepareCommitMsgGuard)},
		"templates/base/scripts/pre-merge-commit-secret-guard.sh":   {Data: []byte(testPreMergeCommitGuard)},
	}
	if g.oldToolSh != nil {
		m["templates/base/scripts/old-tool.sh"] = &fstest.MapFile{Data: g.oldToolSh}
	}
	if g.docsNotes != nil {
		m["templates/base/docs/notes.md"] = &fstest.MapFile{Data: g.docsNotes}
	}
	if g.docsNewNote != nil {
		m["templates/base/docs/newnote.md"] = &fstest.MapFile{Data: g.docsNewNote}
	}
	if g.docsGuide != nil {
		m["templates/base/docs/guide.md"] = &fstest.MapFile{Data: g.docsGuide}
	}
	if g.includeGolang {
		m["templates/packs/golang/verify.sh"] = &fstest.MapFile{Data: g.golangVerifySh}
		m["templates/packs/golang/README.md"] = &fstest.MapFile{Data: g.golangReadme}
		m["templates/packs/golang/rule.md"] = &fstest.MapFile{Data: g.golangRuleMd}
	}
	return m
}

const (
	v1AgentsManaged    = "## Mission\n\nOld mission text.\n"
	v2AgentsManaged    = "## Mission\n\nNew mission text.\n"
	v1GitignoreManaged = ".ralph/local/\nold-ignore/\n"
	v2GitignoreManaged = ".ralph/local/\nnew-ignore/\n"
	v1RunVerify        = "#!/bin/sh\necho v1\n"
	v2RunVerify        = "#!/bin/sh\necho v2\n"
	v1OldTool          = "#!/bin/sh\necho old-tool\n"
	v1DocsNotes        = "# Notes\n\nv1 seed content.\n"
	v2DocsNotes        = "# Notes\n\nv2 seed content (changed upstream).\n"
	v2DocsNewNote      = "# New note\n\nadded in v2.\n"
	v1GolangVerify     = "#!/bin/sh\necho golang-v1\n"
	v2GolangVerify     = "#!/bin/sh\necho golang-v2\n"
	v1GolangReadme     = "# Go v1\n"
	v2GolangReadme     = "# Go v2\n"
	v1GolangRule       = "---\npaths:\n  - \"**/*.go\"\n---\n# Go rules v1\n"
	v2GolangRule       = "---\npaths:\n  - \"**/*.go\"\n---\n# Go rules v2\n"
	v1PreCommitGuard   = testPreCommitGuard
	v2PreCommitGuard   = "#!/usr/bin/env sh\n# pre-commit-secret-guard v2\nexit 0\n"
	v2DocsGuide        = "# Guide\n\nStable guide content, unchanged across upgrades.\n"
)

func v1Settings() []byte {
	return []byte(`{
  "customUserSetting": true,
  "env": {
    "FOO": "v1"
  },
  "permissions": {
    "allow": [
      "Bash(git status:*)",
      "Bash(old-owned:*)"
    ]
  }
}
`)
}

func v2Settings() []byte {
	return []byte(`{
  "customUserSetting": true,
  "env": {
    "FOO": "v2"
  },
  "permissions": {
    "allow": [
      "Bash(git status:*)",
      "Bash(new-owned:*)"
    ]
  }
}
`)
}

func gen1() v2FixtureGen {
	return v2FixtureGen{
		agentsManaged:    []byte(v1AgentsManaged),
		gitignoreManaged: []byte(v1GitignoreManaged),
		settingsJSON:     v1Settings(),
		runVerifySh:      []byte(v1RunVerify),
		oldToolSh:        []byte(v1OldTool),
		docsNotes:        []byte(v1DocsNotes),
		docsGuide:        []byte(v2DocsGuide),
		includeGolang:    true,
		golangVerifySh:   []byte(v1GolangVerify),
		golangReadme:     []byte(v1GolangReadme),
		golangRuleMd:     []byte(v1GolangRule),
		preCommitGuardSh: []byte(v1PreCommitGuard),
	}
}

func gen2() v2FixtureGen {
	return v2FixtureGen{
		agentsManaged:    []byte(v2AgentsManaged),
		gitignoreManaged: []byte(v2GitignoreManaged),
		settingsJSON:     v2Settings(),
		runVerifySh:      []byte(v2RunVerify),
		// oldToolSh omitted: removed upstream between gen1 and gen2.
		docsNotes:        []byte(v2DocsNotes),
		docsNewNote:      []byte(v2DocsNewNote),
		docsGuide:        []byte(v2DocsGuide), // identical to gen1: a stable seed
		includeGolang:    true,
		golangVerifySh:   []byte(v2GolangVerify),
		golangReadme:     []byte(v2GolangReadme),
		golangRuleMd:     []byte(v2GolangRule),
		preCommitGuardSh: []byte(v2PreCommitGuard),
	}
}

// initV2Project runs executeInit against gen's fixture and returns the
// project directory.
func initV2Project(t *testing.T, gen v2FixtureGen, version string) string {
	t.Helper()
	isolateGitConfig(t)
	scaffold.EmbeddedFS = gen.build()
	Version = version

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	cfg := initConfig{ProjectName: "project", Packs: []string{"golang"}}
	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}
	return target
}

func readManifestV2(t *testing.T, target string) *scaffold.Manifest {
	t.Helper()
	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	return m
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return data
}

// snapshotTreeHashesExcluding mirrors snapshotDirHashes (init_v2_test.go)
// but skips paths under any of the given prefixes — used to compare a tree
// before/after an idempotent re-run while excluding bookkeeping paths that
// are expected to always change (manifest.toml's Updated timestamp,
// dated report filenames).
func snapshotTreeHashesExcluding(t *testing.T, dir string, excludePrefixes ...string) []string {
	t.Helper()
	var entries []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		relSlash := filepath.ToSlash(rel)
		for _, prefix := range excludePrefixes {
			if strings.HasPrefix(relSlash, prefix) {
				return nil
			}
		}
		hash, hashErr := scaffold.HashFile(path)
		if hashErr != nil {
			return hashErr
		}
		entries = append(entries, rel+":"+hash)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotTreeHashesExcluding(%s): %v", dir, err)
	}
	sort.Strings(entries)
	return entries
}

// TestRunUpgradeV2_FullPass covers AC-1 end to end: core file update, pack
// payload+rule update, core file removal (ManifestRemove), managed-block
// interior update with out-of-block user content preserved, settings.json
// 3-way merge (stale owned entry pruned, new owned entry added, user entry
// and unrelated user key preserved — also AC-3), a changed seed file
// (advisory only, disk untouched), a newly introduced seed file (created),
// a report file written, and a rebuilt manifest with correct owners/hashes
// and the removed core path dropped.
func TestRunUpgradeV2_FullPass(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	// Simulate a user edit: text added outside AGENTS.md's managed block.
	agentsBefore := mustReadFile(t, filepath.Join(target, "AGENTS.md"))
	agentsWithUserSection := bytes.Replace(
		agentsBefore,
		[]byte("Project notes go here.\n\n"),
		[]byte("Project notes go here.\n\n## My section\n\nHand-written, keep me.\n\n"),
		1,
	)
	if bytes.Equal(agentsWithUserSection, agentsBefore) {
		t.Fatal("test setup: AGENTS.md user-edit replacement did not match")
	}
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), agentsWithUserSection, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	// Simulate a user edit to settings.json: an appended never-owned
	// permission, plus a brand new top-level key ralph has never shipped.
	userSettings := []byte(`{
  "customUserSetting": true,
  "extraUserKey": "please keep me",
  "env": {
    "FOO": "v1"
  },
  "permissions": {
    "allow": [
      "Bash(git status:*)",
      "Bash(old-owned:*)",
      "Bash(user-added:*)"
    ]
  }
}
`)
	if err := os.WriteFile(filepath.Join(target, ".claude", "settings.json"), userSettings, 0644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstderr:\n%s", err, errOut.String())
	}

	// -- core file update --
	gotRunVerify := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotRunVerify) != v2RunVerify {
		t.Errorf("scripts/run-verify.sh = %q, want %q", gotRunVerify, v2RunVerify)
	}

	// -- core file removal --
	if _, err := os.Stat(filepath.Join(target, "scripts", "old-tool.sh")); !os.IsNotExist(err) {
		t.Errorf("scripts/old-tool.sh must be deleted (removed from template, unmodified by user); stat err = %v", err)
	}

	// -- pack payload + rule update --
	gotGoVerify := mustReadFile(t, filepath.Join(target, "packs", "languages", "golang", "verify.sh"))
	if string(gotGoVerify) != v2GolangVerify {
		t.Errorf("pack verify.sh = %q, want %q", gotGoVerify, v2GolangVerify)
	}
	gotGoRule := mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", "golang.md"))
	if string(gotGoRule) != v2GolangRule {
		t.Errorf("pack rule.md = %q, want %q", gotGoRule, v2GolangRule)
	}

	// -- managed block update, out-of-block bytes preserved --
	gotAgents := mustReadFile(t, filepath.Join(target, "AGENTS.md"))
	if !bytes.Contains(gotAgents, []byte("## My section\n\nHand-written, keep me.\n")) {
		t.Errorf("AGENTS.md must preserve the user's out-of-block section:\n%s", gotAgents)
	}
	if !bytes.Contains(gotAgents, []byte(v2AgentsManaged)) {
		t.Errorf("AGENTS.md must contain the new managed block interior:\n%s", gotAgents)
	}
	if bytes.Contains(gotAgents, []byte(v1AgentsManaged)) {
		t.Errorf("AGENTS.md must not still contain the old managed block interior:\n%s", gotAgents)
	}

	// -- settings.json 3-way merge (AC-1 + AC-3) --
	gotSettings := mustReadFile(t, filepath.Join(target, ".claude", "settings.json"))
	var settingsStr = string(gotSettings)
	for _, want := range []string{`"customUserSetting": true`, `"extraUserKey": "please keep me"`, `"FOO": "v2"`, `"Bash(git status:*)"`, `"Bash(new-owned:*)"`, `"Bash(user-added:*)"`} {
		if !strings.Contains(settingsStr, want) {
			t.Errorf("merged settings.json missing %q:\n%s", want, settingsStr)
		}
	}
	if strings.Contains(settingsStr, `"Bash(old-owned:*)"`) {
		t.Errorf("merged settings.json must have pruned the stale ralph-owned entry:\n%s", settingsStr)
	}

	// -- seed: changed template, advisory only, disk untouched --
	gotNotes := mustReadFile(t, filepath.Join(target, "docs", "notes.md"))
	if string(gotNotes) != v1DocsNotes {
		t.Errorf("docs/notes.md (seed) must stay untouched on disk: got %q, want %q", gotNotes, v1DocsNotes)
	}

	// -- seed: newly introduced path, created --
	gotNewNote := mustReadFile(t, filepath.Join(target, "docs", "newnote.md"))
	if string(gotNewNote) != v2DocsNewNote {
		t.Errorf("docs/newnote.md (new seed) = %q, want %q", gotNewNote, v2DocsNewNote)
	}

	// -- report written --
	reportPath := upgrade.UpgradeReportRelPath(Version, time.Now().UTC().Format("2006-01-02"))
	reportContent := mustReadFile(t, filepath.Join(target, reportPath))
	if !strings.Contains(string(reportContent), "docs/notes.md") {
		t.Errorf("upgrade report must mention the changed seed advisory docs/notes.md:\n%s", reportContent)
	}

	// -- manifest rebuild --
	m := readManifestV2(t, target)
	if m.Meta.Layout != scaffold.LayoutV2 {
		t.Errorf("Meta.Layout = %q, want v2", m.Meta.Layout)
	}
	if len(m.Meta.Packs) != 1 || m.Meta.Packs[0] != "golang" {
		t.Errorf("Meta.Packs = %v, want [golang]", m.Meta.Packs)
	}
	if _, ok := m.Files["scripts/old-tool.sh"]; ok {
		t.Error("manifest must drop the removed scripts/old-tool.sh entry")
	}
	if entry, ok := m.Files["scripts/run-verify.sh"]; !ok || entry.Owner != scaffold.OwnerCore || entry.DiskHash != scaffold.HashBytes([]byte(v2RunVerify)) {
		t.Errorf("scripts/run-verify.sh manifest entry = %+v, want owner=core, DiskHash=hash(v2 content)", entry)
	}
	if entry, ok := m.Files["docs/notes.md"]; !ok || entry.Owner != scaffold.OwnerSeed || entry.TemplateHash != scaffold.HashBytes([]byte(v2DocsNotes)) || entry.DiskHash != scaffold.HashBytes([]byte(v1DocsNotes)) {
		t.Errorf("docs/notes.md manifest entry = %+v, want owner=seed, TemplateHash=hash(v2), DiskHash=hash(v1, unchanged)", entry)
	}
	if entry, ok := m.Files["docs/newnote.md"]; !ok || entry.Owner != scaffold.OwnerSeed || entry.TemplateHash != entry.DiskHash {
		t.Errorf("docs/newnote.md manifest entry = %+v, want owner=seed, TemplateHash==DiskHash", entry)
	}
	if entry, ok := m.Files["AGENTS.md"]; !ok || entry.Owner != scaffold.OwnerBlock || entry.DiskHash != scaffold.HashBytes(gotAgents) {
		t.Errorf("AGENTS.md manifest entry = %+v, want owner=block, DiskHash=hash(final disk content)", entry)
	}
	if entry, ok := m.Files[".claude/settings.json"]; !ok || entry.Owner != scaffold.OwnerCore || entry.DiskHash != scaffold.HashBytes(gotSettings) {
		t.Errorf(".claude/settings.json manifest entry = %+v, want owner=core, DiskHash=hash(final merged content)", entry)
	}
	if entry, ok := m.Files["packs/languages/golang/verify.sh"]; !ok || entry.Owner != scaffold.OwnerCore {
		t.Errorf("pack verify.sh manifest entry = %+v, want owner=core", entry)
	}
}

// TestRunUpgradeV2_UnresolvedDrift_ExitSentinelAndStderr covers AC-4: a
// path whose disk content diverges from both the recorded and new template
// hash is left completely untouched, listed on stderr, and the flow returns
// a wrapped ErrUpgradeDriftRemaining (mapped to exit 3 by cmd/ralph/main.go)
// while everything else still completes (report + manifest still written).
func TestRunUpgradeV2_UnresolvedDrift_ExitSentinelAndStderr(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	driftedContent := "#!/bin/sh\necho totally-unexpected-content\n"
	if err := os.WriteFile(filepath.Join(target, "scripts", "run-verify.sh"), []byte(driftedContent), 0644); err != nil {
		t.Fatalf("write drifted run-verify.sh: %v", err)
	}

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("expected an error when unresolved drift remains, got nil")
	}
	if !errors.Is(err, ErrUpgradeDriftRemaining) {
		t.Errorf("err = %v, want errors.Is(err, ErrUpgradeDriftRemaining)", err)
	}
	if !strings.Contains(errOut.String(), "scripts/run-verify.sh") {
		t.Errorf("stderr must list the drifted path:\n%s", errOut.String())
	}

	gotContent := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotContent) != driftedContent {
		t.Errorf("drifted file must not be touched: got %q, want %q", gotContent, driftedContent)
	}

	// Everything else still completes: manifest and report exist, and a
	// second (unrelated) core update still landed.
	m := readManifestV2(t, target)
	if m.Meta.Layout != scaffold.LayoutV2 {
		t.Errorf("manifest must still be rebuilt on a drift-only failure: Layout = %q", m.Meta.Layout)
	}
	entry, ok := m.Files["scripts/run-verify.sh"]
	if !ok {
		t.Fatal("manifest must still carry an entry for the drifted path")
	}
	if entry.DiskHash == scaffold.HashBytes([]byte(v2RunVerify)) {
		t.Error("drifted path's manifest entry must not be advanced to the new template hash")
	}

	reportPath := upgrade.UpgradeReportRelPath(Version, time.Now().UTC().Format("2006-01-02"))
	if _, statErr := os.Stat(filepath.Join(target, reportPath)); statErr != nil {
		t.Errorf("upgrade report must still be written on a drift-only failure: %v", statErr)
	}
}

// TestRunUpgradeV2_SettingsSnapshotMissing_FallbackDegradesNonDestructively
// covers AC-4c's fallback path (AC-11(a)/(b)): a project scaffolded before
// the settings snapshot existed (Phase 2 init generation) has no
// .ralph/core/settings.ralph.json. The upgrade must still succeed, treat
// oldOwned as "{}" (so a stale ralph-owned entry is NOT pruned — the
// documented non-destructive degradation), record the fallback usage in the
// report, and create the snapshot going forward.
func TestRunUpgradeV2_SettingsSnapshotMissing_FallbackDegradesNonDestructively(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	snapshotPath := filepath.Join(target, ".ralph", "core", "settings.ralph.json")
	if err := os.Remove(snapshotPath); err != nil {
		t.Fatalf("removing settings snapshot to simulate a pre-Phase-3 init: %v", err)
	}
	m := readManifestV2(t, target)
	delete(m.Files, ".ralph/core/settings.ralph.json")
	if err := m.Write(filepath.Join(target, ".ralph", "manifest.toml")); err != nil {
		t.Fatalf("rewriting manifest without the snapshot entry: %v", err)
	}

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstderr:\n%s", err, errOut.String())
	}

	gotSettings := string(mustReadFile(t, filepath.Join(target, ".claude", "settings.json")))
	if !strings.Contains(gotSettings, `"Bash(old-owned:*)"`) {
		t.Errorf("with oldOwned falling back to {}, the stale owned entry must NOT be pruned:\n%s", gotSettings)
	}
	if !strings.Contains(gotSettings, `"Bash(new-owned:*)"`) {
		t.Errorf("the new owned entry must still be added:\n%s", gotSettings)
	}

	reportPath := upgrade.UpgradeReportRelPath(Version, time.Now().UTC().Format("2006-01-02"))
	reportContent := string(mustReadFile(t, filepath.Join(target, reportPath)))
	if !strings.Contains(reportContent, "fallback") {
		t.Errorf("report must record that the settings snapshot fallback was used:\n%s", reportContent)
	}

	if _, statErr := os.Stat(snapshotPath); statErr != nil {
		t.Errorf("settings snapshot must be (re)created going forward: %v", statErr)
	}
}

// TestRunUpgradeV2_SettingsWriteFailure_SnapshotNotAdvanced pins the 2-phase
// settings snapshot update ordering (AC-4c): if the settings.json write
// itself fails, the snapshot on disk must not have been advanced to the new
// template content yet — the oldOwned side of the 3-way merge is never lost,
// so a subsequent successful run can still correctly prune stale entries.
func TestRunUpgradeV2_SettingsWriteFailure_SnapshotNotAdvanced(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	settingsPath := filepath.Join(target, ".claude", "settings.json")
	if err := os.Chmod(settingsPath, 0444); err != nil {
		t.Fatalf("chmod settings.json read-only: %v", err)
	}
	if err := os.Chmod(filepath.Join(target, ".claude"), 0555); err != nil {
		t.Fatalf("chmod .claude read-only: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(target, ".claude"), 0755)
		_ = os.Chmod(settingsPath, 0644)
	})

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Skip("environment allows writes through a read-only directory (e.g. running as root) — cannot exercise this failure injection")
	}

	snapshotContent := string(mustReadFile(t, filepath.Join(target, ".ralph", "core", "settings.ralph.json")))
	if snapshotContent != string(v1Settings()) {
		t.Errorf("snapshot must not advance when the settings.json write fails first:\ngot  %q\nwant %q (unchanged v1)", snapshotContent, v1Settings())
	}

	// Restore write access and retry: the run must now fully succeed and
	// the snapshot must advance to v2.
	if err := os.Chmod(filepath.Join(target, ".claude"), 0755); err != nil {
		t.Fatalf("chmod .claude writable: %v", err)
	}
	if err := os.Chmod(settingsPath, 0644); err != nil {
		t.Fatalf("chmod settings.json writable: %v", err)
	}

	var out2, errOut2 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false); err != nil {
		t.Fatalf("retry after restoring permissions must succeed: %v\nstderr:\n%s", err, errOut2.String())
	}
	snapshotAfterRetry := string(mustReadFile(t, filepath.Join(target, ".ralph", "core", "settings.ralph.json")))
	if snapshotAfterRetry != string(v2Settings()) {
		t.Errorf("snapshot must advance to v2 after a successful retry:\ngot  %q\nwant %q", snapshotAfterRetry, v2Settings())
	}
}

// TestRunUpgradeV2_Idempotent_SecondRunIsNoOp covers AC-5: running the same
// target version twice in a row produces zero further scaffold-content
// writes on the second run (manifest.toml's Updated timestamp and the dated
// report file are bookkeeping and expected to change every run, so they are
// excluded from the comparison).
func TestRunUpgradeV2_Idempotent_SecondRunIsNoOp(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out1, errOut1 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out1, &errOut1, false); err != nil {
		t.Fatalf("first upgrade run: %v\nstderr:\n%s", err, errOut1.String())
	}

	before := snapshotTreeHashesExcluding(t, target, ".ralph/manifest.toml", "docs/reports/")

	var out2, errOut2 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false); err != nil {
		t.Fatalf("second upgrade run: %v\nstderr:\n%s", err, errOut2.String())
	}

	after := snapshotTreeHashesExcluding(t, target, ".ralph/manifest.toml", "docs/reports/")
	if !slicesEqualStrings(before, after) {
		t.Errorf("second run must be a no-op on scaffold content; before=%v after=%v", before, after)
	}

	reportPath := upgrade.UpgradeReportRelPath(Version, time.Now().UTC().Format("2006-01-02"))
	reportContent := string(mustReadFile(t, filepath.Join(target, reportPath)))
	if !strings.Contains(reportContent, "| Deleted | 0 |") || !strings.Contains(reportContent, "| Created | 0 |") || !strings.Contains(reportContent, "| Updated | 0 |") {
		t.Errorf("second run's report must record zero ops:\n%s", reportContent)
	}
}

// TestRunUpgradeV2_UnavailablePack_FullyPreserved covers AC-9: a pack that
// was installed but is no longer available in the embedded templates is
// left completely untouched on disk and in the manifest (no delete, no
// drift classification), with a warning surfaced to stderr.
func TestRunUpgradeV2_UnavailablePack_FullyPreserved(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	before := snapshotTreeHashesExcluding(t, target)
	beforeManifest := readManifestV2(t, target)

	genNoPack := gen2()
	genNoPack.includeGolang = false
	scaffold.EmbeddedFS = genNoPack.build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstderr:\n%s", err, errOut.String())
	}
	if !strings.Contains(errOut.String(), "golang") {
		t.Errorf("stderr must warn about the unavailable golang pack:\n%s", errOut.String())
	}

	after := snapshotTreeHashesExcluding(t, target, ".ralph/manifest.toml", "docs/reports/", "scripts/run-verify.sh", "docs/notes.md", "docs/newnote.md", "AGENTS.md", ".gitignore", ".claude/settings.json", ".ralph/core/settings.ralph.json", "scripts/old-tool.sh")
	before = filterPrefixed(before, "packs/languages/golang/", ".claude/rules/ralph/golang.md")
	after = filterPrefixed(after, "packs/languages/golang/", ".claude/rules/ralph/golang.md")
	if !slicesEqualStrings(before, after) {
		t.Errorf("golang pack files must be byte-for-byte untouched; before=%v after=%v", before, after)
	}

	afterManifest := readManifestV2(t, target)
	if afterManifest.Meta.Packs == nil || afterManifest.Meta.Packs[0] != "golang" {
		t.Errorf("Meta.Packs must still retain golang, got %v", afterManifest.Meta.Packs)
	}
	for path, beforeEntry := range beforeManifest.Files {
		if !strings.HasPrefix(path, "packs/languages/golang/") && path != ".claude/rules/ralph/golang.md" {
			continue
		}
		afterEntry, ok := afterManifest.Files[path]
		if !ok {
			t.Errorf("manifest entry for %s must be preserved, but is missing", path)
			continue
		}
		if afterEntry != beforeEntry {
			t.Errorf("manifest entry for %s changed: before=%+v after=%+v", path, beforeEntry, afterEntry)
		}
	}
}

// TestRunUpgradeV2_MalformedBlock_CompletesWithReport covers AC-12: a
// managed block that has been hand-edited into a malformed shape (missing
// END marker) is left byte-for-byte untouched, a warning is recorded, and
// the upgrade still completes successfully overall.
func TestRunUpgradeV2_MalformedBlock_CompletesWithReport(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	malformed := []byte("# AGENTS.md\n\n" + upgrade.BeginMarker("agents-md") + "\nstray content, no end marker\n")
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), malformed, 0644); err != nil {
		t.Fatalf("write malformed AGENTS.md: %v", err)
	}

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("a malformed block must not fail the whole upgrade: %v\nstderr:\n%s", err, errOut.String())
	}
	if !strings.Contains(errOut.String(), "malformed") {
		t.Errorf("stderr must warn about the malformed block:\n%s", errOut.String())
	}

	got := mustReadFile(t, filepath.Join(target, "AGENTS.md"))
	if !bytes.Equal(got, malformed) {
		t.Errorf("malformed AGENTS.md must be left byte-for-byte untouched:\ngot  %q\nwant %q", got, malformed)
	}

	reportPath := upgrade.UpgradeReportRelPath(Version, time.Now().UTC().Format("2006-01-02"))
	reportContent := string(mustReadFile(t, filepath.Join(target, reportPath)))
	if !strings.Contains(reportContent, "malformed") {
		t.Errorf("report must record the malformed block:\n%s", reportContent)
	}
}

// TestRunUpgradeIOWithOptions_LegacyManifest_FailsClosedZeroWrites is AC-7
// coverage for `ralph upgrade`: a manifest with no meta.layout (a genuine
// pre-v2 project) is rejected fail-closed with zero writes to the tree. The
// legacy interactive upgrade engine was removed in Phase 3 (docs/plans
// /active/2026-08-18-overlay-scaffold-v2-p3.md, FR-13); the automated
// migration to v2 arrives in a later ralph release (Phase 4).
func TestRunUpgradeIOWithOptions_LegacyManifest_FailsClosedZeroWrites(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "2.0.0-test"

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}
	legacy := scaffold.NewManifest("1.0.0-test")
	legacy.SetFile("AGENTS.md", "sha256:legacy")
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	if err := legacy.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# legacy AGENTS\n"), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	before := snapshotDirHashes(t, dir)

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(dir, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("upgrade on a legacy manifest: expected an error, got nil")
	}
	if !errors.Is(err, errLegacyLayoutFailClosed) {
		t.Errorf("err = %v, want errors.Is(err, errLegacyLayoutFailClosed)", err)
	}

	after := snapshotDirHashes(t, dir)
	if !slicesEqualStrings(before, after) {
		t.Errorf("legacy-manifest upgrade must write zero files; before=%v after=%v", before, after)
	}
}

// TestRunUpgradeV2_RemovesLegacyBaselineDirectory is AC-8 coverage: a
// successful v2 upgrade deletes a leftover .ralph/baseline/ directory from a
// project that predates Phase 3's baseline-mechanism removal (docs/plans
// /active/2026-08-18-overlay-scaffold-v2-p3.md). The directory is simulated
// directly (rather than produced by a real legacy write path, since that
// path no longer exists in this codebase) — see the `## 後始末` step of the
// plan's Scope section.
func TestRunUpgradeV2_RemovesLegacyBaselineDirectory(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	baselineDir := filepath.Join(target, ".ralph", "baseline")
	if err := os.MkdirAll(baselineDir, 0755); err != nil {
		t.Fatalf("MkdirAll .ralph/baseline: %v", err)
	}
	if err := os.WriteFile(filepath.Join(baselineDir, "AGENTS.md"), []byte("stale baseline\n"), 0644); err != nil {
		t.Fatalf("seed stale baseline file: %v", err)
	}

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstderr:\n%s", err, errOut.String())
	}

	if _, statErr := os.Stat(baselineDir); !os.IsNotExist(statErr) {
		t.Errorf(".ralph/baseline must be removed after a successful v2 upgrade; stat err = %v", statErr)
	}
}

// TestRunUpgradeV2_ReinstallsGitHooks covers the git-hooks gap recorded in
// the plan's Deviations section: slice 3's rewrite of the v2 flow dropped
// the installManagedGitHooks call the legacy engine used to make. A v2
// upgrade must (a) refresh an already-installed managed hook whose
// underlying guard script changed upstream, and (b) reinstall a hook that
// is missing entirely (e.g. deleted by hand, or a fresh clone that never
// ran `ralph init` with the current guard scripts).
func TestRunUpgradeV2_ReinstallsGitHooks(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	preCommitHook := filepath.Join(target, ".git", "hooks", "pre-commit")
	before := mustReadFile(t, preCommitHook)
	if string(before) != v1PreCommitGuard {
		t.Fatalf("pre-commit hook content after init = %q, want %q", before, v1PreCommitGuard)
	}

	commitMsgHook := filepath.Join(target, ".git", "hooks", "commit-msg")
	if err := os.Remove(commitMsgHook); err != nil {
		t.Fatalf("removing commit-msg hook to simulate a missing hook: %v", err)
	}

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstderr:\n%s", err, errOut.String())
	}

	// -- refreshed: the pre-commit hook's content tracks the new upstream
	// guard script, not the one recorded at init time.
	afterPreCommit := mustReadFile(t, preCommitHook)
	if string(afterPreCommit) != v2PreCommitGuard {
		t.Errorf("pre-commit hook content after upgrade = %q, want refreshed %q", afterPreCommit, v2PreCommitGuard)
	}

	// -- reinstalled: a hook removed out-of-band is put back, executable.
	afterCommitMsg := mustReadFile(t, commitMsgHook)
	if string(afterCommitMsg) != testCommitMsgGuard {
		t.Errorf("commit-msg hook content = %q, want %q", afterCommitMsg, testCommitMsgGuard)
	}
	info, err := os.Stat(commitMsgHook)
	if err != nil {
		t.Fatalf("stat commit-msg hook: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("commit-msg hook is not executable: mode %v", info.Mode().Perm())
	}
}

// TestRunUpgradeV2_PartialFailure_ManifestNotAdvanced_ResumeCompletes covers
// AC-6: a failure injected mid-ApplyOps (chmod 0o444 on one of several core
// files scheduled for update, the same technique as
// TestApplyOps_PartialFailureStopsSubsequentOps in
// internal/upgrade/replaceplan_test.go) must leave the manifest byte-for-byte
// unadvanced and must not reach the settings merge or settings-snapshot
// write (both happen after ApplyOps in runUpgradeV2) — proving the 2-phase
// settings-snapshot guarantee end to end, not just at the settings-write
// boundary already covered by
// TestRunUpgradeV2_SettingsWriteFailure_SnapshotNotAdvanced. Ops before the
// failed path may have already landed; ops after it must not have been
// attempted. Restoring write access and re-running must then complete fully
// with a stable classification (no drift misclassification of the
// already-applied ops) and advance the manifest and settings snapshot.
func TestRunUpgradeV2_PartialFailure_ManifestNotAdvanced_ResumeCompletes(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0o444 does not deny writes to root; this failure-injection technique cannot run as root")
	}

	target := initV2Project(t, gen1(), "1.0.0-test")
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	manifestBefore := mustReadFile(t, manifestPath)
	settingsSnapshotPath := filepath.Join(target, ".ralph", "core", "settings.ralph.json")
	snapshotBefore := mustReadFile(t, settingsSnapshotPath)
	settingsPath := filepath.Join(target, ".claude", "settings.json")
	settingsBefore := mustReadFile(t, settingsPath)

	// packs/languages/golang/verify.sh sorts between two other update ops
	// this generation bump produces (.ralph/core/AGENTS.core.md and
	// packs/languages/golang/README.md sort before it; scripts/run-verify.sh
	// sorts after it — see PlanCoreReplaceDesired's per-op-kind, sorted-path
	// ordering), so blocking it here exercises both "already landed" and
	// "never attempted" update ops in the same run.
	blockedPath := filepath.Join(target, "packs", "languages", "golang", "verify.sh")
	if err := os.Chmod(blockedPath, 0o444); err != nil {
		t.Fatalf("chmod blocked file read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedPath, 0o644) })

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("expected the injected mid-ApplyOps failure to surface as an error")
	}

	// -- manifest unchanged: the commit barrier was never reached --
	manifestAfterFailure := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfterFailure) {
		t.Errorf("manifest must not advance on a partial ApplyOps failure:\nbefore: %s\nafter:  %s", manifestBefore, manifestAfterFailure)
	}

	// -- settings merge and snapshot write never reached (both run after
	// ApplyOps in runUpgradeV2) --
	settingsAfterFailure := mustReadFile(t, settingsPath)
	if !bytes.Equal(settingsBefore, settingsAfterFailure) {
		t.Errorf("settings.json must be untouched when ApplyOps fails before the settings merge step:\nbefore: %s\nafter:  %s", settingsBefore, settingsAfterFailure)
	}
	snapshotAfterFailure := mustReadFile(t, settingsSnapshotPath)
	if !bytes.Equal(snapshotBefore, snapshotAfterFailure) {
		t.Errorf("settings snapshot must be untouched when ApplyOps fails before the settings-merge/snapshot steps:\nbefore: %s\nafter:  %s", snapshotBefore, snapshotAfterFailure)
	}

	// -- earlier ops (sorted before the blocked path) landed --
	gotAgentsCore := mustReadFile(t, filepath.Join(target, ".ralph", "core", "AGENTS.core.md"))
	if string(gotAgentsCore) != v2AgentsManaged {
		t.Errorf(".ralph/core/AGENTS.core.md = %q, want landed v2 content %q (this op sorts before the blocked path)", gotAgentsCore, v2AgentsManaged)
	}
	gotGoReadme := mustReadFile(t, filepath.Join(target, "packs", "languages", "golang", "README.md"))
	if string(gotGoReadme) != v2GolangReadme {
		t.Errorf("packs/languages/golang/README.md = %q, want landed v2 content %q (this op sorts before the blocked path)", gotGoReadme, v2GolangReadme)
	}

	// -- the blocked op itself did not land --
	gotBlocked := mustReadFile(t, blockedPath)
	if string(gotBlocked) != v1GolangVerify {
		t.Errorf("packs/languages/golang/verify.sh must still hold its pre-upgrade (v1) content: got %q", gotBlocked)
	}

	// -- later ops (sorted after the blocked path) were never attempted --
	gotRunVerify := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotRunVerify) != v1RunVerify {
		t.Errorf("scripts/run-verify.sh must not have been touched (sorts after the blocked path): got %q, want v1 %q", gotRunVerify, v1RunVerify)
	}

	// -- restore write access and re-run: must now fully succeed --
	if err := os.Chmod(blockedPath, 0o644); err != nil {
		t.Fatalf("chmod blocked file writable: %v", err)
	}

	var out2, errOut2 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false); err != nil {
		t.Fatalf("retry after restoring permissions must succeed: %v\nstderr:\n%s", err, errOut2.String())
	}

	m := readManifestV2(t, target)
	if entry, ok := m.Files["packs/languages/golang/verify.sh"]; !ok || entry.DiskHash != scaffold.HashBytes([]byte(v2GolangVerify)) {
		t.Errorf("packs/languages/golang/verify.sh manifest entry after retry = %+v, want DiskHash=hash(v2 content)", entry)
	}
	gotRunVerifyAfterRetry := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotRunVerifyAfterRetry) != v2RunVerify {
		t.Errorf("scripts/run-verify.sh after retry = %q, want v2 %q", gotRunVerifyAfterRetry, v2RunVerify)
	}
	snapshotAfterRetry := mustReadFile(t, settingsSnapshotPath)
	if string(snapshotAfterRetry) != string(v2Settings()) {
		t.Errorf("settings snapshot after successful retry = %q, want v2 %q", snapshotAfterRetry, v2Settings())
	}

	// -- final tree matches a clean (non-failure-injected) run byte for
	// byte, aside from bookkeeping paths that always change (manifest.toml's
	// Updated timestamp, the dated report file, and .git/ — a second,
	// independently git-inited comparison tree is not byte-comparable there).
	clean := initV2Project(t, gen1(), "1.0.0-test")
	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"
	var outClean, errOutClean bytes.Buffer
	if err := runUpgradeIOWithOptions(clean, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &outClean, &errOutClean, false); err != nil {
		t.Fatalf("clean-run comparison upgrade: %v\nstderr:\n%s", err, errOutClean.String())
	}
	resumed := snapshotTreeHashesExcluding(t, target, ".ralph/manifest.toml", "docs/reports/", ".git/")
	cleanTree := snapshotTreeHashesExcluding(t, clean, ".ralph/manifest.toml", "docs/reports/", ".git/")
	if !slicesEqualStrings(resumed, cleanTree) {
		t.Errorf("resumed tree must match a clean run byte for byte; resumed=%v clean=%v", resumed, cleanTree)
	}
}

// setupMixedScenarioProject builds a v2 project on gen1, mutates disk and
// manifest to exercise every op/advisory class the v2 engine supports in a
// single tree, and swaps the embedded FS to gen2 (Version bumped) without
// running the upgrade. Shared by TestRunUpgradeV2_MixedScenario_* and
// TestRunUpgradeV2_DryRunPreview_MatchesMixedScenarioCounts so a real run
// and its --dry-run preview can never drift out of sync with each other.
//
// Classes exercised (expected plan shape on the gen1→gen2 upgrade this
// produces): 1 delete (scripts/old-tool.sh, ManifestRemove), 2 creates
// (docs/newnote.md — new upstream seed; docs/guide.md — existing seed
// missing from disk, recreated), 4 updates (.ralph/core/AGENTS.core.md,
// scripts/run-verify.sh, packs/languages/golang/verify.sh,
// .claude/rules/ralph/golang.md), 1 manifest refresh
// (scripts/pre-commit-secret-guard.sh — disk already matches the new
// template, only the recorded hash is stale), 1 drift
// (packs/languages/golang/README.md — disk matches neither the recorded nor
// the new template hash), 2 advisories (docs/notes.md — seed, user-edited
// disk + template changed upstream; scripts/my-fork.sh — fork entry, no
// template at all), 1 legacy-skipped (legacy-notes.txt — no ownership
// attribute recorded), 0 preserved packs (golang stays available in gen2).
// Plus the two block surfaces (AGENTS.md user content outside the block,
// .gitignore) and a settings.json 3-way merge with a user-added permission
// and a brand new top-level key.
func setupMixedScenarioProject(t *testing.T) string {
	t.Helper()
	target := initV2Project(t, gen1(), "1.0.0-test")

	// -- user content outside AGENTS.md's managed block --
	agentsBefore := mustReadFile(t, filepath.Join(target, "AGENTS.md"))
	agentsWithUserSection := bytes.Replace(
		agentsBefore,
		[]byte("Project notes go here.\n\n"),
		[]byte("Project notes go here.\n\n## My section\n\nHand-written, keep me.\n\n"),
		1,
	)
	if bytes.Equal(agentsWithUserSection, agentsBefore) {
		t.Fatal("test setup: AGENTS.md user-edit replacement did not match")
	}
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), agentsWithUserSection, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	// -- user edits to settings.json: a never-owned appended permission plus
	// a brand new top-level key ralph has never shipped --
	userSettings := []byte(`{
  "customUserSetting": true,
  "extraUserKey": "please keep me",
  "env": {
    "FOO": "v1"
  },
  "permissions": {
    "allow": [
      "Bash(git status:*)",
      "Bash(old-owned:*)",
      "Bash(user-added:*)"
    ]
  }
}
`)
	if err := os.WriteFile(filepath.Join(target, ".claude", "settings.json"), userSettings, 0644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	m := readManifestV2(t, target)

	// -- fork entry: untouched on disk, surfaced only as a (skipped) advisory --
	forkContent := []byte("#!/bin/sh\necho fork, hand-maintained\n")
	if err := os.WriteFile(filepath.Join(target, "scripts", "my-fork.sh"), forkContent, 0755); err != nil {
		t.Fatalf("write fork file: %v", err)
	}
	m.SetFileFork("scripts/my-fork.sh", scaffold.HashBytes(forkContent), "1.0.0-test")

	// -- legacy (no ownership attribute) entry: left completely alone --
	legacyContent := []byte("pre-v3 manifest entry, never attributed\n")
	if err := os.WriteFile(filepath.Join(target, "legacy-notes.txt"), legacyContent, 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}
	m.SetFile("legacy-notes.txt", scaffold.HashBytes(legacyContent))

	// -- seed missing: user (or tooling) deleted docs/guide.md; template
	// still ships it, so it must be recreated --
	if err := os.Remove(filepath.Join(target, "docs", "guide.md")); err != nil {
		t.Fatalf("removing docs/guide.md to simulate a missing seed: %v", err)
	}

	// -- seed modified-by-user + template-changed: user hand-edits
	// docs/notes.md; upstream also changes it between gen1 and gen2 — must
	// surface as an advisory only, disk stays exactly as the user left it --
	userNotes := []byte("# Notes\n\nHand-edited by the user, keep me exactly as-is.\n")
	if err := os.WriteFile(filepath.Join(target, "docs", "notes.md"), userNotes, 0644); err != nil {
		t.Fatalf("write user-edited docs/notes.md: %v", err)
	}

	// -- drift: a core file whose disk content matches neither the recorded
	// hash nor the new template hash --
	driftedReadme := []byte("# totally unexpected drifted content\n")
	if err := os.WriteFile(filepath.Join(target, "packs", "languages", "golang", "README.md"), driftedReadme, 0644); err != nil {
		t.Fatalf("write drifted golang README: %v", err)
	}

	// -- manifest refresh: disk already matches the *new* (gen2) template
	// content (as if the user hand-synced it ahead of time), but the
	// recorded hash still reflects gen1 — a hash-only bookkeeping update,
	// no file write --
	if err := os.WriteFile(filepath.Join(target, "scripts", "pre-commit-secret-guard.sh"), []byte(v2PreCommitGuard), 0644); err != nil {
		t.Fatalf("write pre-synced pre-commit guard: %v", err)
	}

	if err := m.Write(filepath.Join(target, ".ralph", "manifest.toml")); err != nil {
		t.Fatalf("writing mutated manifest: %v", err)
	}

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	return target
}

// TestRunUpgradeV2_MixedScenario_AllClassesLandCorrectly runs a single
// upgrade over setupMixedScenarioProject's fixture and asserts every op and
// advisory class lands correctly in the same run (a v2 upgrade is one plan,
// one commit barrier — these classes are never mutually exclusive in
// practice), then re-runs and confirms convergence: the drift path persists
// (exit sentinel stays ErrUpgradeDriftRemaining) but the run produces zero
// further writes.
func TestRunUpgradeV2_MixedScenario_AllClassesLandCorrectly(t *testing.T) {
	target := setupMixedScenarioProject(t)

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if !errors.Is(err, ErrUpgradeDriftRemaining) {
		t.Fatalf("err = %v, want errors.Is(err, ErrUpgradeDriftRemaining) (the drift path must not block the rest of the plan)", err)
	}

	// -- core update --
	if got := string(mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))); got != v2RunVerify {
		t.Errorf("scripts/run-verify.sh = %q, want %q", got, v2RunVerify)
	}
	if got := string(mustReadFile(t, filepath.Join(target, "packs", "languages", "golang", "verify.sh"))); got != v2GolangVerify {
		t.Errorf("pack verify.sh = %q, want %q", got, v2GolangVerify)
	}
	if got := string(mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", "golang.md"))); got != v2GolangRule {
		t.Errorf("pack rule.md = %q, want %q", got, v2GolangRule)
	}

	// -- core removal (ManifestRemove) --
	if _, err := os.Stat(filepath.Join(target, "scripts", "old-tool.sh")); !os.IsNotExist(err) {
		t.Errorf("scripts/old-tool.sh must be deleted; stat err = %v", err)
	}

	// -- fork entry: untouched + advisory --
	forkPath := filepath.Join(target, "scripts", "my-fork.sh")
	forkContentAfter := mustReadFile(t, forkPath)
	if string(forkContentAfter) != "#!/bin/sh\necho fork, hand-maintained\n" {
		t.Errorf("fork file must be byte-for-byte untouched, got %q", forkContentAfter)
	}

	// -- seed missing, recreated --
	if got := string(mustReadFile(t, filepath.Join(target, "docs", "guide.md"))); got != v2DocsGuide {
		t.Errorf("docs/guide.md must be recreated from the template, got %q, want %q", got, v2DocsGuide)
	}

	// -- seed modified-by-user + template-changed: advisory only, disk
	// stays exactly as the user left it --
	wantUserNotes := "# Notes\n\nHand-edited by the user, keep me exactly as-is.\n"
	if got := string(mustReadFile(t, filepath.Join(target, "docs", "notes.md"))); got != wantUserNotes {
		t.Errorf("docs/notes.md must stay exactly as the user left it, got %q, want %q", got, wantUserNotes)
	}

	// -- user content outside AGENTS.md block, and the block itself updated --
	gotAgents := mustReadFile(t, filepath.Join(target, "AGENTS.md"))
	if !bytes.Contains(gotAgents, []byte("## My section\n\nHand-written, keep me.\n")) {
		t.Errorf("AGENTS.md must preserve the user's out-of-block section:\n%s", gotAgents)
	}
	if !bytes.Contains(gotAgents, []byte(v2AgentsManaged)) {
		t.Errorf("AGENTS.md must contain the new managed block interior:\n%s", gotAgents)
	}

	// -- user permission in settings: preserved, plus stale pruning + new entry --
	gotSettings := string(mustReadFile(t, filepath.Join(target, ".claude", "settings.json")))
	for _, want := range []string{`"extraUserKey": "please keep me"`, `"Bash(user-added:*)"`, `"Bash(new-owned:*)"`} {
		if !strings.Contains(gotSettings, want) {
			t.Errorf("merged settings.json missing %q:\n%s", want, gotSettings)
		}
	}
	if strings.Contains(gotSettings, `"Bash(old-owned:*)"`) {
		t.Errorf("merged settings.json must have pruned the stale ralph-owned entry:\n%s", gotSettings)
	}

	// -- drift path: untouched, listed on stderr --
	driftPath := filepath.Join(target, "packs", "languages", "golang", "README.md")
	wantDrift := "# totally unexpected drifted content\n"
	if got := string(mustReadFile(t, driftPath)); got != wantDrift {
		t.Errorf("drifted packs/languages/golang/README.md must be untouched, got %q, want %q", got, wantDrift)
	}
	if !strings.Contains(errOut.String(), "packs/languages/golang/README.md") {
		t.Errorf("stderr must list the drifted path:\n%s", errOut.String())
	}

	// -- manifest: legacy entry preserved unchanged, fork entry preserved
	// unchanged, drift path unchanged, manifest-refresh path advanced --
	m := readManifestV2(t, target)
	legacyEntry, ok := m.Files["legacy-notes.txt"]
	if !ok || !legacyEntry.IsLegacyOwner() {
		t.Errorf("legacy-notes.txt manifest entry = %+v, want a preserved legacy (unattributed-owner) entry", legacyEntry)
	}
	forkEntry, ok := m.Files["scripts/my-fork.sh"]
	if !ok || forkEntry.Owner != scaffold.OwnerFork {
		t.Errorf("scripts/my-fork.sh manifest entry = %+v, want owner=fork, preserved", forkEntry)
	}
	if entry, ok := m.Files["scripts/pre-commit-secret-guard.sh"]; !ok || entry.DiskHash != scaffold.HashBytes([]byte(v2PreCommitGuard)) {
		t.Errorf("scripts/pre-commit-secret-guard.sh manifest entry = %+v, want DiskHash=hash(v2 content) (manifest-refresh, no file write)", entry)
	}
	if entry, ok := m.Files["docs/notes.md"]; !ok || entry.TemplateHash != scaffold.HashBytes([]byte(v2DocsNotes)) || entry.DiskHash != scaffold.HashBytes([]byte(wantUserNotes)) {
		t.Errorf("docs/notes.md manifest entry = %+v, want TemplateHash=hash(v2 template), DiskHash=hash(user content)", entry)
	}

	// -- report contains every relevant section --
	reportPath := upgrade.UpgradeReportRelPath(Version, time.Now().UTC().Format("2006-01-02"))
	reportContent := string(mustReadFile(t, filepath.Join(target, reportPath)))
	for _, want := range []string{
		"## Summary",
		"## Applied",
		"### Deleted",
		"### Created",
		"### Updated",
		"## Manifest refresh",
		"## Unresolved drift",
		"## Advisories",
		"## Legacy skipped",
		"scripts/old-tool.sh",
		"docs/newnote.md",
		"docs/guide.md",
		"scripts/run-verify.sh",
		"scripts/pre-commit-secret-guard.sh",
		"packs/languages/golang/README.md",
		"docs/notes.md",
		"scripts/my-fork.sh",
		"legacy-notes.txt",
	} {
		if !strings.Contains(reportContent, want) {
			t.Errorf("upgrade report must contain %q:\n%s", want, reportContent)
		}
	}

	// -- second run: drift persists (still exit sentinel), zero new writes --
	before := snapshotTreeHashesExcluding(t, target, ".ralph/manifest.toml", "docs/reports/")

	var out2, errOut2 bytes.Buffer
	err2 := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false)
	if !errors.Is(err2, ErrUpgradeDriftRemaining) {
		t.Errorf("second run err = %v, want errors.Is(err, ErrUpgradeDriftRemaining) (drift path was never fixed)", err2)
	}

	after := snapshotTreeHashesExcluding(t, target, ".ralph/manifest.toml", "docs/reports/")
	if !slicesEqualStrings(before, after) {
		t.Errorf("second run must produce zero further scaffold-content writes; before=%v after=%v", before, after)
	}
}

// TestRunUpgradeV2_DryRunPreview_MatchesMixedScenarioCounts covers "Dry-run
// completeness": --dry-run over setupMixedScenarioProject's fixture (the
// same shared setup TestRunUpgradeV2_MixedScenario_AllClassesLandCorrectly
// uses for a real run) must report the same op/advisory/preserved counts a
// real run would produce, and must write zero bytes to the tree.
func TestRunUpgradeV2_DryRunPreview_MatchesMixedScenarioCounts(t *testing.T) {
	target := setupMixedScenarioProject(t)

	before := snapshotTreeHashesExcluding(t, target)

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{DryRun: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("--dry-run must not itself error on a plan containing drift (drift is only reported, not fatal, in preview mode): %v\nstderr:\n%s", err, errOut.String())
	}

	after := snapshotTreeHashesExcluding(t, target)
	if !slicesEqualStrings(before, after) {
		t.Errorf("--dry-run must write zero files; before=%v after=%v", before, after)
	}

	preview := out.String()
	for _, want := range []string{
		fmt.Sprintf("  delete:            %d files\n", 1),
		fmt.Sprintf("  create:            %d files\n", 2),
		fmt.Sprintf("  update:            %d files\n", 4),
		fmt.Sprintf("  manifest refresh:  %d files\n", 1),
		fmt.Sprintf("  drift (untouched): %d files\n", 1),
		fmt.Sprintf("  advisories:        %d files\n", 2),
		fmt.Sprintf("  preserved packs:   %d files\n", 0),
	} {
		if !strings.Contains(preview, want) {
			t.Errorf("dry-run preview missing %q; full preview:\n%s", want, preview)
		}
	}
	if !strings.Contains(errOut.String(), "packs/languages/golang/README.md") {
		t.Errorf("dry-run preview stderr must list the drifted path:\n%s", errOut.String())
	}
}

func slicesEqualStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func filterPrefixed(entries []string, prefixes ...string) []string {
	var out []string
	for _, e := range entries {
		for _, p := range prefixes {
			if strings.HasPrefix(e, p) {
				out = append(out, e)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}
