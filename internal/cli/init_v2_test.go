package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

const testAgentsCoreManaged = "## Mission\n\nManaged agents guidance.\n"
const testGitignoreManaged = ".ralph/local/\nnode_modules/\n"

func testAgentsMDContent() []byte {
	return []byte("# AGENTS.md\n\nProject notes go here.\n\n" +
		upgrade.BeginMarker("agents-md") + "\n" +
		testAgentsCoreManaged +
		upgrade.EndMarker + "\n")
}

func testGitignoreContent() []byte {
	return []byte("# Project ignores\n" +
		upgrade.BeginMarkerStyled("gitignore", upgrade.BlockMarkerHash) + "\n" +
		testGitignoreManaged +
		upgrade.EndMarkerStyled(upgrade.BlockMarkerHash) + "\n")
}

// setupTestEmbedFSV2 injects a mock FS shaped like the v2 overlay layout:
// AGENTS.md and .gitignore carry real ralph managed blocks, and a
// representative seed/core file is present at each owner-mapped path so
// ownerForScaffoldPath can be exercised end to end.
func setupTestEmbedFSV2(t *testing.T) {
	t.Helper()
	isolateGitConfig(t)
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/AGENTS.md":                          {Data: testAgentsMDContent()},
		"templates/base/.gitignore":                         {Data: testGitignoreContent()},
		"templates/base/CLAUDE.md":                          {Data: []byte("@AGENTS.md\n\n# Claude Code\n")},
		"templates/base/ralph.toml":                         {Data: []byte("[pipeline]\nmodel = \"test\"\n[doctor]\nrequire_codex_cli = false\n")},
		"templates/base/docs/quality/definition-of-done.md": {Data: []byte("# Definition of done\n")},
		"templates/base/.github/workflows/verify.yml":       {Data: []byte("name: verify\n")},
		"templates/base/.claude/skills/work/SKILL.md":       {Data: []byte("---\nname: work\ndescription: work\n---\nbody\n")},
		"templates/base/.claude/settings.json":              {Data: []byte("{}\n")},
		"templates/base/scripts/run-verify.sh":              {Data: []byte("#!/bin/sh\necho ok\n")},
	}
}

func TestExecuteInit_V2_FreshInit_LayoutAndOwners(t *testing.T) {
	setupTestEmbedFSV2(t)
	Version = "0.5.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	cfg := initConfig{ProjectName: "project", Packs: nil}

	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if m.Meta.Layout != scaffold.LayoutV2 {
		t.Fatalf("Meta.Layout = %q, want %q", m.Meta.Layout, scaffold.LayoutV2)
	}

	wantOwners := map[string]string{
		"AGENTS.md":  scaffold.OwnerBlock,
		".gitignore": scaffold.OwnerBlock,
		"CLAUDE.md":  scaffold.OwnerSeed,
		"ralph.toml": scaffold.OwnerSeed,
		filepath.Join("docs", "quality", "definition-of-done.md"): scaffold.OwnerSeed,
		filepath.Join(".github", "workflows", "verify.yml"):       scaffold.OwnerSeed,
		filepath.Join(".claude", "skills", "work", "SKILL.md"):    scaffold.OwnerCore,
		filepath.Join("scripts", "run-verify.sh"):                 scaffold.OwnerCore,
	}
	for path, wantOwner := range wantOwners {
		entry, ok := m.Files[path]
		if !ok {
			t.Errorf("manifest missing entry for %s", path)
			continue
		}
		if entry.Owner != wantOwner {
			t.Errorf("owner[%s] = %q, want %q", path, entry.Owner, wantOwner)
		}
	}

	// AGENTS.md on disk must contain exactly one block whose interior
	// equals the managed content shipped in the template (here standing in
	// for .ralph/core/AGENTS.core.md).
	agentsOnDisk, err := os.ReadFile(filepath.Join(target, "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}
	result := upgrade.UpdateManagedBlock(agentsOnDisk, "agents-md", []byte(testAgentsCoreManaged))
	if result.Outcome != upgrade.BlockUnchanged {
		t.Fatalf("AGENTS.md block interior mismatch or malformed: outcome=%v reason=%q", result.Outcome, result.Reason)
	}
}

func TestExecuteInit_V2_PreExistingFiles_AppendsBlockPreservesSeed(t *testing.T) {
	setupTestEmbedFSV2(t)
	Version = "0.5.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	customClaude := []byte("@AGENTS.md\n\n# My Claude notes\n\nDo not touch.\n")
	customAgentsPrefix := "# AGENTS.md\n\nMy own project notes, hand-written.\n\n"
	customGitignorePrefix := "# my ignores\ncustom-dir/\n"

	if err := os.WriteFile(filepath.Join(target, "CLAUDE.md"), customClaude, 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), []byte(customAgentsPrefix), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, ".gitignore"), []byte(customGitignorePrefix), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	cfg := initConfig{ProjectName: "project", Packs: nil}
	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	// CLAUDE.md is seed-once: byte-identical after init.
	gotClaude, err := os.ReadFile(filepath.Join(target, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	if string(gotClaude) != string(customClaude) {
		t.Errorf("CLAUDE.md changed by init:\ngot  %q\nwant %q", gotClaude, customClaude)
	}

	// AGENTS.md = original content + appended HTML block.
	gotAgents, err := os.ReadFile(filepath.Join(target, "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}
	// customAgentsPrefix already ends with a blank line, so the block
	// append does not insert an additional separator (see
	// upgrade.UpdateManagedBlock's "already ends with a blank line" case).
	wantAgents := customAgentsPrefix +
		upgrade.BeginMarker("agents-md") + "\n" +
		testAgentsCoreManaged +
		upgrade.EndMarker + "\n"
	if string(gotAgents) != wantAgents {
		t.Errorf("AGENTS.md after init:\ngot  %q\nwant %q", gotAgents, wantAgents)
	}
	if !hasPrefixBytes(gotAgents, []byte(customAgentsPrefix)) {
		t.Error("AGENTS.md must preserve original bytes outside the block")
	}

	// .gitignore = original + appended hash block.
	gotGitignore, err := os.ReadFile(filepath.Join(target, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	wantGitignore := customGitignorePrefix + "\n" +
		upgrade.BeginMarkerStyled("gitignore", upgrade.BlockMarkerHash) + "\n" +
		testGitignoreManaged +
		upgrade.EndMarkerStyled(upgrade.BlockMarkerHash) + "\n"
	if string(gotGitignore) != wantGitignore {
		t.Errorf(".gitignore after init:\ngot  %q\nwant %q", gotGitignore, wantGitignore)
	}
	if !hasPrefixBytes(gotGitignore, []byte(customGitignorePrefix)) {
		t.Error(".gitignore must preserve original bytes outside the block")
	}

	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if entry := m.Files["AGENTS.md"]; entry.Owner != scaffold.OwnerBlock {
		t.Errorf("AGENTS.md owner = %q, want block", entry.Owner)
	}
	if entry := m.Files[".gitignore"]; entry.Owner != scaffold.OwnerBlock {
		t.Errorf(".gitignore owner = %q, want block", entry.Owner)
	}
	if entry := m.Files["CLAUDE.md"]; entry.Owner != scaffold.OwnerSeed {
		t.Errorf("CLAUDE.md owner = %q, want seed", entry.Owner)
	}
}

func TestExecuteInit_V2_MalformedBlock_LeftUntouchedWithWarning(t *testing.T) {
	setupTestEmbedFSV2(t)
	Version = "0.5.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// BEGIN marker only, no matching END: malformed.
	malformed := []byte("# AGENTS.md\n\n" + upgrade.BeginMarker("agents-md") + "\nstray content\n")
	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), malformed, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	cfg := initConfig{ProjectName: "project", Packs: nil}
	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit must still succeed on a malformed block: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(target, "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading AGENTS.md: %v", err)
	}
	if string(got) != string(malformed) {
		t.Errorf("AGENTS.md with a malformed block must be left untouched:\ngot  %q\nwant %q", got, malformed)
	}
}

func hasPrefixBytes(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	return string(b[:len(prefix)]) == string(prefix)
}

// TestExecuteInit_V2_LanguagePack_RuleUnderRalphSubdir verifies that a
// language pack's rule.md control file renders under .claude/rules/ralph/
// (not directly under .claude/rules/), is tracked in the manifest with
// owner=core (via ownerForScaffoldPath's catch-all), and that doctor's
// installed-pack check still passes against the rendered layout. See
// docs/specs 2026-08-17-overlay-scaffold-v2.md, AC-6.
func TestExecuteInit_V2_LanguagePack_RuleUnderRalphSubdir(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.5.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	cfg := initConfig{ProjectName: "project", Packs: []string{"golang"}}

	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	rulePath := filepath.Join(".claude", "rules", "ralph", "golang.md")
	if _, err := os.Stat(filepath.Join(target, rulePath)); err != nil {
		t.Fatalf("expected pack rule to exist at %s: %v", rulePath, err)
	}
	if _, err := os.Stat(filepath.Join(target, ".claude", "rules", "golang.md")); !os.IsNotExist(err) {
		t.Errorf("pack rule must not render directly under .claude/rules/; stat err = %v", err)
	}

	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry, ok := m.Files[rulePath]
	if !ok {
		t.Fatalf("manifest missing pack rule entry for %s", rulePath)
	}
	if entry.Owner != scaffold.OwnerCore {
		t.Errorf("pack rule owner[%s] = %q, want %q", rulePath, entry.Owner, scaffold.OwnerCore)
	}

	for _, r := range checkInstalledPacks(target) {
		if r.Status == "fail" {
			t.Errorf("doctor pack check failed after moving rule under rules/ralph: %+v", r)
		}
	}
}

// snapshotDirHashes returns a sorted "relpath:sha256" list for every regular
// file under dir. Used to assert that a fail-closed guard produced zero
// writes to the project tree.
func snapshotDirHashes(t *testing.T, dir string) []string {
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
		hash, hashErr := scaffold.HashFile(path)
		if hashErr != nil {
			return hashErr
		}
		entries = append(entries, rel+":"+hash)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotDirHashes(%s): %v", dir, err)
	}
	sort.Strings(entries)
	return entries
}

// TestRunUpgradeIO_V2Layout_FailsClosedWithoutWrites verifies AC-10: the
// legacy upgrade engine refuses to run against a manifest whose
// meta.layout is "v2", writes zero files, and names the v2/Phase-3 reason
// in its error. Exercised for both the interactive/apply path (force=true)
// and --dry-run, since the guard must short-circuit before either branch
// touches the diff engine.
func TestRunUpgradeIO_V2Layout_FailsClosedWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts upgradeOptions
	}{
		{name: "force", opts: upgradeOptions{Force: true, Pager: pagerNever}},
		{name: "dry-run", opts: upgradeOptions{DryRun: true, Pager: pagerNever}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupTestEmbedFS(t)
			Version = "2.0.0-test"

			dir := t.TempDir()
			agentsContent := []byte("# AGENTS\n")
			if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), agentsContent, 0644); err != nil {
				t.Fatalf("write AGENTS.md: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(dir, ".ralph"), 0755); err != nil {
				t.Fatalf("MkdirAll .ralph: %v", err)
			}
			m := scaffold.NewManifest("1.0.0-test")
			m.SetLayoutV2()
			if setErr := m.SetFileOwned("AGENTS.md", scaffold.OwnerBlock, scaffold.HashBytes(agentsContent), scaffold.HashBytes(agentsContent)); setErr != nil {
				t.Fatalf("SetFileOwned: %v", setErr)
			}
			manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
			if err := m.Write(manifestPath); err != nil {
				t.Fatalf("write manifest: %v", err)
			}

			before := snapshotDirHashes(t, dir)

			var out, errOut bytes.Buffer
			err := runUpgradeIOWithOptions(dir, tc.opts, strings.NewReader(""), &out, &errOut, false)
			if err == nil {
				t.Fatal("expected an error for a v2-layout manifest, got nil")
			}
			if !strings.Contains(err.Error(), "v2") {
				t.Errorf("error = %q, want it to mention the v2 layout", err.Error())
			}
			if !strings.Contains(err.Error(), "Phase 3") {
				t.Errorf("error = %q, want it to mention Phase 3", err.Error())
			}

			after := snapshotDirHashes(t, dir)
			if !slices.Equal(before, after) {
				t.Errorf("guard must write zero files; before=%v after=%v", before, after)
			}
		})
	}
}
