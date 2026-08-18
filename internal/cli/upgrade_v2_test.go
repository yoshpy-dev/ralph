package cli

import (
	"bytes"
	"errors"
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
	includeGolang    bool
	golangVerifySh   []byte
	golangReadme     []byte
	golangRuleMd     []byte
}

func (g v2FixtureGen) build() fstest.MapFS {
	m := fstest.MapFS{
		"templates/base/AGENTS.md":                       {Data: v2AgentsMD(g.agentsManaged)},
		"templates/base/.gitignore":                      {Data: v2Gitignore(g.gitignoreManaged)},
		"templates/base/CLAUDE.md":                       {Data: []byte("@AGENTS.md\n\n# Claude Code\n")},
		"templates/base/ralph.toml":                      {Data: []byte("[pipeline]\nmodel = \"test\"\n[doctor]\nrequire_codex_cli = false\n")},
		"templates/base/.claude/settings.json":           {Data: g.settingsJSON},
		"templates/base/.ralph/core/settings.ralph.json": {Data: g.settingsJSON},
		"templates/base/.ralph/core/AGENTS.core.md":      {Data: g.agentsManaged},
		"templates/base/scripts/run-verify.sh":           {Data: g.runVerifySh},
		"templates/base/.ralph/local/verify.d/.gitkeep":  {Data: []byte("")},
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
		includeGolang:    true,
		golangVerifySh:   []byte(v1GolangVerify),
		golangReadme:     []byte(v1GolangReadme),
		golangRuleMd:     []byte(v1GolangRule),
	}
}

func gen2() v2FixtureGen {
	return v2FixtureGen{
		agentsManaged:    []byte(v2AgentsManaged),
		gitignoreManaged: []byte(v2GitignoreManaged),
		settingsJSON:     v2Settings(),
		runVerifySh:      []byte(v2RunVerify),
		// oldToolSh omitted: removed upstream between gen1 and gen2.
		docsNotes:      []byte(v2DocsNotes),
		docsNewNote:    []byte(v2DocsNewNote),
		includeGolang:  true,
		golangVerifySh: []byte(v2GolangVerify),
		golangReadme:   []byte(v2GolangReadme),
		golangRuleMd:   []byte(v2GolangRule),
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
