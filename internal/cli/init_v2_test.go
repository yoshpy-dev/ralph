package cli

import (
	"bytes"
	"io"
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
const testSettingsSnapshotContent = "{\n  \"env\": {}\n}\n"

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
		"templates/base/.ralph/local/verify.d/.gitkeep":     {Data: []byte("")},
		"templates/base/.ralph/core/settings.ralph.json":    {Data: []byte(testSettingsSnapshotContent)},
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
		filepath.Join(".ralph", "local", "verify.d", ".gitkeep"):  scaffold.OwnerSeed,
		filepath.Join(".ralph", "core", "settings.ralph.json"):    scaffold.OwnerCore,
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

	// The settings.json snapshot renders via the normal owner=core catch-all
	// (no init-specific special-casing needed) and LoadSettingsSnapshot must
	// find it immediately after a fresh init.
	snapshotOnDisk, found, err := upgrade.LoadSettingsSnapshot(target)
	if err != nil {
		t.Fatalf("LoadSettingsSnapshot: %v", err)
	}
	if !found {
		t.Fatal("LoadSettingsSnapshot found = false after fresh v2 init, want true")
	}
	if string(snapshotOnDisk) != testSettingsSnapshotContent {
		t.Errorf("settings.ralph.json on disk = %q, want %q", snapshotOnDisk, testSettingsSnapshotContent)
	}
}

// TestExecuteInit_V2_FreshInit_DoctorHooksIntegrityPasses pins the cycle-2
// cross-review fix end to end: a fresh `ralph init` (v2 layout) followed by
// `ralph doctor` must report hooks integrity as passing. Before the fix,
// settings.json hook commands shaped as "executable arg" strings (the
// dispatcher pattern every v2 scaffold ships, see
// templates/base/.claude/settings.json) made checkHooks stat the whole
// command string instead of just the executable token, so every fresh v2
// init falsely reported "hook script(s) missing" — the reviewer's repro.
func TestExecuteInit_V2_FreshInit_DoctorHooksIntegrityPasses(t *testing.T) {
	isolateGitConfig(t)
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/AGENTS.md":                          {Data: testAgentsMDContent()},
		"templates/base/.gitignore":                         {Data: testGitignoreContent()},
		"templates/base/CLAUDE.md":                          {Data: []byte("@AGENTS.md\n\n# Claude Code\n")},
		"templates/base/ralph.toml":                         {Data: []byte("[pipeline]\nmodel = \"test\"\n[doctor]\nrequire_codex_cli = false\n")},
		"templates/base/docs/quality/definition-of-done.md": {Data: []byte("# Definition of done\n")},
		"templates/base/.github/workflows/verify.yml":       {Data: []byte("name: verify\n")},
		"templates/base/.claude/skills/work/SKILL.md":       {Data: []byte("---\nname: work\ndescription: work\n---\nbody\n")},
		// Dispatcher-shaped hook command, matching production
		// templates/base/.claude/settings.json.
		"templates/base/.claude/settings.json": {Data: []byte(`{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear|compact",
        "hooks": [
          {
            "type": "command",
            "command": "./.claude/hooks/ralph-dispatch.sh SessionStart"
          }
        ]
      }
    ]
  }
}
`)},
		"templates/base/.claude/hooks/ralph-dispatch.sh": {Data: []byte("#!/bin/sh\nexit 0\n")},
		"templates/base/scripts/run-verify.sh":           {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/base/.ralph/local/verify.d/.gitkeep":  {Data: []byte("")},
	}
	Version = "0.5.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	cfg := initConfig{ProjectName: "project", Packs: nil}
	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	r := checkHooks(target)
	if r.Status != "pass" {
		t.Fatalf("checkHooks against a fresh v2 init = status %q, detail %q, want pass", r.Status, r.Detail)
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

// TestExecuteInit_V2_SymlinkedBlockSurface_LeftUntouchedWithWarning pins the
// cycle-2 cross-review fix: init's block-append step must not follow a
// symlinked block surface (AGENTS.md/.gitignore), since writing through it
// could land outside targetDir. Symlink and its external target must be
// left byte-for-byte unchanged, a warning must be printed, init must still
// succeed, and the manifest must record ownership without a DiskHash (same
// shape as the malformed-block case) since nothing was written to disk.
func TestExecuteInit_V2_SymlinkedBlockSurface_LeftUntouchedWithWarning(t *testing.T) {
	setupTestEmbedFSV2(t)
	Version = "0.5.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// The symlink target lives outside targetDir entirely.
	externalContent := []byte("# external AGENTS.md\n\nnot managed by ralph.\n")
	externalPath := filepath.Join(dir, "external-AGENTS.md")
	if err := os.WriteFile(externalPath, externalContent, 0644); err != nil {
		t.Fatalf("write external file: %v", err)
	}

	linkPath := filepath.Join(target, "AGENTS.md")
	if err := os.Symlink(externalPath, linkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := initConfig{ProjectName: "project", Packs: nil}
	err := executeInit(target, cfg, false)

	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("executeInit must still succeed when a block surface is a symlink: %v", err)
	}
	if !strings.Contains(string(out), "AGENTS.md: is a symlink; left untouched") {
		t.Errorf("expected symlink warning in output:\n%s", out)
	}

	// The symlink itself must be untouched: still a symlink, still pointing
	// at the same external target.
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat AGENTS.md: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("AGENTS.md must remain a symlink, got mode %v", info.Mode())
	}
	resolved, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink AGENTS.md: %v", err)
	}
	if resolved != externalPath {
		t.Errorf("AGENTS.md symlink target = %q, want %q", resolved, externalPath)
	}

	// The external target's content must be unchanged (never written to).
	gotExternal, err := os.ReadFile(externalPath)
	if err != nil {
		t.Fatalf("reading external target: %v", err)
	}
	if string(gotExternal) != string(externalContent) {
		t.Errorf("external symlink target must be left untouched:\ngot  %q\nwant %q", gotExternal, externalContent)
	}

	// The manifest must still record ownership for AGENTS.md, but without a
	// DiskHash: nothing was actually written to disk, so SetOwner (not
	// SetFileOwned) is the code path taken, mirroring the malformed-block
	// case.
	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry, ok := m.Files["AGENTS.md"]
	if !ok {
		t.Fatalf("manifest missing entry for AGENTS.md")
	}
	if entry.Owner != scaffold.OwnerBlock {
		t.Errorf("AGENTS.md owner = %q, want %q", entry.Owner, scaffold.OwnerBlock)
	}
	if entry.DiskHash != "" {
		t.Errorf("AGENTS.md DiskHash = %q, want empty (nothing written to disk)", entry.DiskHash)
	}
}

// TestExecuteInit_V2_DanglingSymlinkBlockSurface_LeftUntouchedWithWarning
// pins the C3-1 self-review fix: RenderFS's existence check must use Lstat,
// not Stat, so a *dangling* symlinked block surface (the symlink's target
// does not exist anywhere) is classified as "exists" (skipped) rather than
// "absent" (created) and never gets written through by os.WriteFile. Before
// the fix, os.Stat on a dangling symlink returns ErrNotExist, RenderFS
// treated AGENTS.md as a create, and the scaffold content landed at the
// symlink's external target -- outside targetDir entirely.
func TestExecuteInit_V2_DanglingSymlinkBlockSurface_LeftUntouchedWithWarning(t *testing.T) {
	setupTestEmbedFSV2(t)
	Version = "0.5.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// The symlink target does not exist anywhere -- a dangling symlink.
	externalPath := filepath.Join(dir, "external-AGENTS.md")

	linkPath := filepath.Join(target, "AGENTS.md")
	if err := os.Symlink(externalPath, linkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := initConfig{ProjectName: "project", Packs: nil}
	err := executeInit(target, cfg, false)

	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("executeInit must still succeed when a block surface is a dangling symlink: %v", err)
	}
	if !strings.Contains(string(out), "AGENTS.md: is a symlink; left untouched") {
		t.Errorf("expected symlink warning in output:\n%s", out)
	}

	// The symlink itself must be untouched: still a symlink, still pointing
	// at the same (nonexistent) external target.
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat AGENTS.md: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("AGENTS.md must remain a symlink, got mode %v", info.Mode())
	}
	resolved, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink AGENTS.md: %v", err)
	}
	if resolved != externalPath {
		t.Errorf("AGENTS.md symlink target = %q, want %q", resolved, externalPath)
	}

	// Nothing must have been created at the external (dangling) target --
	// this is the containment failure the fix closes.
	if _, err := os.Stat(externalPath); !os.IsNotExist(err) {
		t.Errorf("scaffold content must not be written through a dangling symlink; stat err = %v", err)
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

// TestRunUpgradeIO_V2Layout_DispatchesToV2Engine pins the Phase 3 slice-2
// replacement of the Phase 2 fail-closed guard: a manifest whose
// meta.layout is "v2" no longer refuses to run at all — it dispatches to
// the non-interactive v2 upgrade flow (internal/cli/upgrade_v2.go) instead
// of the legacy diff/conflict engine. --force remains rejected outright on
// v2 layouts (zero writes, error names "--force" and "v2"; fork
// re-adoption is Phase 5's `ralph adopt`, not this flag). --dry-run now
// succeeds (it runs the v2 preview path) with zero writes, since dry-run
// must never touch disk regardless of engine.
func TestRunUpgradeIO_V2Layout_DispatchesToV2Engine(t *testing.T) {
	setupTestEmbed := func(t *testing.T) {
		t.Helper()
		setupTestEmbedFS(t)
	}

	t.Run("force_rejected_zero_writes", func(t *testing.T) {
		setupTestEmbed(t)
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
		err := runUpgradeIOWithOptions(dir, upgradeOptions{Force: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
		if err == nil {
			t.Fatal("expected an error for --force on a v2-layout manifest, got nil")
		}
		if !strings.Contains(err.Error(), "--force") {
			t.Errorf("error = %q, want it to mention --force", err.Error())
		}
		if !strings.Contains(err.Error(), "v2") {
			t.Errorf("error = %q, want it to mention the v2 layout", err.Error())
		}

		after := snapshotDirHashes(t, dir)
		if !slices.Equal(before, after) {
			t.Errorf("--force rejection must write zero files; before=%v after=%v", before, after)
		}
	})

	t.Run("dry_run_succeeds_zero_writes", func(t *testing.T) {
		setupTestEmbed(t)
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
		err := runUpgradeIOWithOptions(dir, upgradeOptions{DryRun: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
		if err != nil {
			t.Fatalf("--dry-run on a v2-layout manifest must succeed (previews the v2 plan): %v", err)
		}

		after := snapshotDirHashes(t, dir)
		if !slices.Equal(before, after) {
			t.Errorf("--dry-run must write zero files; before=%v after=%v", before, after)
		}
	})
}
