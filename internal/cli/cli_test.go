package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

const testCommitMsgGuard = "#!/usr/bin/env sh\n# commit-msg-guard\nexit 0\n"
const testPreCommitGuard = "#!/usr/bin/env sh\n# pre-commit-secret-guard\nexit 0\n"
const testPrepareCommitMsgGuard = "#!/usr/bin/env sh\n# prepare-commit-msg-secret-guard\nexit 0\n"
const testPreMergeCommitGuard = "#!/usr/bin/env sh\n# pre-merge-commit-secret-guard\nexit 0\n"
const testGoRule = "---\npaths:\n  - \"**/*.go\"\n---\n# Go rules\n"
const testTypescriptRule = "---\npaths:\n  - \"**/*.ts\"\n---\n# TS rules\n"

// setupTestEmbedFS injects a minimal mock FS into scaffold.EmbeddedFS for testing.
// Includes the Codex parity tree (.codex/ + .agents/skills/) so tests can
// assert all three CLI surfaces are rendered by `ralph init`.
func setupTestEmbedFS(t *testing.T) {
	t.Helper()
	isolateGitConfig(t)
	setupTestEmbedFSWithAgentsAndCommitGuard(t, []byte("# AGENTS\n"), []byte(testCommitMsgGuard))
}

func setupTestEmbedFSWithCommitGuard(t *testing.T, commitMsgGuard []byte) {
	t.Helper()
	isolateGitConfig(t)
	setupTestEmbedFSWithAgentsAndCommitGuard(t, []byte("# AGENTS\n"), commitMsgGuard)
}

func setupTestEmbedFSWithAgents(t *testing.T, agents []byte) {
	t.Helper()
	isolateGitConfig(t)
	setupTestEmbedFSWithAgentsAndCommitGuard(t, agents, []byte(testCommitMsgGuard))
}

func setupTestEmbedFSWithAgentsAndCommitGuard(t *testing.T, agents, commitMsgGuard []byte) {
	t.Helper()
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/AGENTS.md":                                  {Data: agents},
		"templates/base/CLAUDE.md":                                  {Data: []byte("# CLAUDE\n")},
		"templates/base/ralph.toml":                                 {Data: []byte("[pipeline]\nmodel = \"test\"\n[doctor]\nrequire_codex_cli = false\n")},
		"templates/base/.claude/settings.json":                      {Data: []byte("{}\n")},
		"templates/base/.codex/config.toml":                         {Data: []byte("model = \"gpt-5.5\"\n[features]\nhooks = true\n")},
		"templates/base/.codex/AGENTS.override.md":                  {Data: []byte("# codex overrides\n")},
		"templates/base/.codex/README.md":                           {Data: []byte("# codex setup\n")},
		"templates/base/.codex/agents/doc-maintainer.toml":          {Data: []byte("name = \"doc-maintainer\"\n")},
		"templates/base/.codex/agents/reviewer.toml":                {Data: []byte("name = \"reviewer\"\n")},
		"templates/base/.codex/agents/tester.toml":                  {Data: []byte("name = \"tester\"\n")},
		"templates/base/.codex/agents/verifier.toml":                {Data: []byte("name = \"verifier\"\n")},
		"templates/base/.agents/skills/.gitkeep":                    {Data: []byte("")},
		"templates/base/.agents/skills/spec/SKILL.md":               {Data: []byte("---\nname: spec\ndescription: refine\n---\nbody\n")},
		"templates/base/scripts/pre-commit-secret-guard.sh":         {Data: []byte(testPreCommitGuard)},
		"templates/base/scripts/commit-msg-guard.sh":                {Data: commitMsgGuard},
		"templates/base/scripts/prepare-commit-msg-secret-guard.sh": {Data: []byte(testPrepareCommitMsgGuard)},
		"templates/base/scripts/pre-merge-commit-secret-guard.sh":   {Data: []byte(testPreMergeCommitGuard)},
		"templates/packs/golang/verify.sh":                          {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/golang/README.md":                          {Data: []byte("# Go\n")},
		"templates/packs/golang/rule.md":                            {Data: []byte(testGoRule)},
		"templates/packs/typescript/verify.sh":                      {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/typescript/README.md":                      {Data: []byte("# TS\n")},
		"templates/packs/typescript/rule.md":                        {Data: []byte(testTypescriptRule)},
	}
}

// clearManifestLayoutV2 rewrites the project manifest at dir/.ralph/manifest.toml
// with an empty Meta.Layout, simulating a genuine pre-v2 (legacy) project.
// executeInit unconditionally writes Meta.Layout = scaffold.LayoutV2 (see
// docs/specs 2026-08-17-overlay-scaffold-v2.md, AC-10), so tests that exercise
// the legacy upgrade engine's diff/migration behavior must downgrade the
// fixture back to a layout the legacy engine is still allowed to run against
// — otherwise they only exercise the new fail-closed guard, not the migration
// logic under test.
func clearManifestLayoutV2(t *testing.T, dir string) {
	t.Helper()
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("clearManifestLayoutV2: ReadManifest: %v", err)
	}
	m.Meta.Layout = ""
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("clearManifestLayoutV2: Write: %v", err)
	}
}

func isolateGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func TestExecuteInit_NewProject(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "new-project")

	cfg := initConfig{
		ProjectName: "new-project",
		Packs:       []string{"golang"},
	}

	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	// Check files created.
	for _, f := range []string{"AGENTS.md", "CLAUDE.md", "ralph.toml", ".ralph/manifest.toml", "packs/languages/golang/verify.sh", ".claude/rules/ralph/golang.md"} {
		if _, err := os.Stat(filepath.Join(target, f)); err != nil {
			t.Errorf("expected %s to exist: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "packs", "languages", "golang", "rule.md")); !os.IsNotExist(err) {
		t.Errorf("pack control rule.md should not render under packs/languages/golang; stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, ".claude", "rules", "ralph", "typescript.md")); !os.IsNotExist(err) {
		t.Errorf("unselected typescript rule should not be rendered; stat err = %v", err)
	}

	// Check git init happened.
	if _, err := os.Stat(filepath.Join(target, ".git")); err != nil {
		t.Errorf("expected .git to exist: %v", err)
	}

	// Check manifest has files.
	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if _, ok := m.Files["AGENTS.md"]; !ok {
		t.Error("manifest missing AGENTS.md")
	}
	if _, ok := m.Files[filepath.Join(".claude", "rules", "ralph", "golang.md")]; !ok {
		t.Error("manifest missing selected pack rule .claude/rules/ralph/golang.md")
	}
	if _, ok := m.Files[filepath.Join("packs", "languages", "golang", "rule.md")]; ok {
		t.Error("manifest should not track packs/languages/golang/rule.md")
	}
	agentsEntry := m.Files["AGENTS.md"]
	if agentsEntry.BaselineStatus != scaffold.BaselineStatusAvailable {
		t.Errorf("AGENTS.md baseline_status = %q, want available", agentsEntry.BaselineStatus)
	}
	baselineBytes, err := scaffold.ReadBaseline(target, agentsEntry)
	if err != nil {
		t.Fatalf("ReadBaseline(AGENTS.md): %v", err)
	}
	if string(baselineBytes) != "# AGENTS\n" {
		t.Errorf("AGENTS.md baseline = %q, want template content", baselineBytes)
	}
	if m.Meta.Version != "0.1.0-test" {
		t.Errorf("manifest version = %q, want 0.1.0-test", m.Meta.Version)
	}
}

func TestExecuteInit_InstallsManagedGitHooks(t *testing.T) {
	isolateGitConfig(t)
	setupTestEmbedFSWithCommitGuard(t, []byte(testCommitMsgGuard))
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	expected := map[string]string{
		"pre-commit":         testPreCommitGuard,
		"commit-msg":         testCommitMsgGuard,
		"prepare-commit-msg": testPrepareCommitMsgGuard,
		"pre-merge-commit":   testPreMergeCommitGuard,
	}
	for name, want := range expected {
		hookPath := filepath.Join(dir, ".git", "hooks", name)
		got, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("%s hook not installed: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s hook content = %q, want %q", name, got, want)
		}
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("stat %s hook: %v", name, err)
		}
		if info.Mode().Perm()&0100 == 0 {
			t.Errorf("%s hook is not executable: mode %v", name, info.Mode().Perm())
		}
	}
}

// TestExecuteInit_RendersCodexSurfaces enforces AC-1 of the Codex parity
// spec: a fresh `ralph init` must scaffold .claude/, .codex/, AND
// .agents/skills/ in lock-step. Without this gate the embed FS could quietly
// drop the Codex tree (e.g. stale go:embed pattern) and produce projects that
// look fine to Claude users but break for Codex users.
func TestExecuteInit_RendersCodexSurfaces(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	target := filepath.Join(dir, "codex-parity-project")

	cfg := initConfig{ProjectName: "codex-parity-project", Packs: []string{"golang"}}
	if err := executeInit(target, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	required := []string{
		// Claude surfaces (pre-existing).
		".claude/settings.json",
		// Codex project surfaces.
		".codex/config.toml",
		".codex/AGENTS.override.md",
		".codex/README.md",
		".codex/agents/doc-maintainer.toml",
		".codex/agents/reviewer.toml",
		".codex/agents/tester.toml",
		".codex/agents/verifier.toml",
		// Codex skill surface.
		".agents/skills/spec/SKILL.md",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(target, rel)); err != nil {
			t.Errorf("Codex parity: %s missing after init: %v", rel, err)
		}
	}

	// Manifest should track every Codex-side path so future `ralph upgrade`
	// runs can detect drift on these files.
	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	for _, rel := range []string{
		".codex/config.toml",
		".codex/agents/reviewer.toml",
		".agents/skills/spec/SKILL.md",
	} {
		if _, ok := m.Files[rel]; !ok {
			t.Errorf("manifest missing Codex-side path %q", rel)
		}
	}
}

func TestExecuteInit_ExistingProject_DelegatesToUpgrade(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()

	// First init.
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("first init: %v", err)
	}

	// Add a user-owned file.
	userFile := filepath.Join(dir, "my-custom.md")
	if err := os.WriteFile(userFile, []byte("user content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Re-init (should delegate to upgrade, preserving user files).
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("re-init: %v", err)
	}

	// User file should still exist.
	content, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatalf("user file missing: %v", err)
	}
	if string(content) != "user content" {
		t.Errorf("user file content = %q, want %q", content, "user content")
	}
}

// Regression: runInitInteractive must short-circuit to upgrade BEFORE the
// huh form runs when a manifest already exists. We can't drive the TTY form
// in tests, but we can verify the early-return path completes without error
// (form.Run() would block on stdin in a non-tty environment) and that the
// existing project files remain intact.
func TestRunInitInteractive_ExistingProjectSkipsForm(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()

	// Seed an initialized project.
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("seed init: %v", err)
	}

	userFile := filepath.Join(dir, "user-edit.md")
	if err := os.WriteFile(userFile, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runInitInteractive(dir, false); err != nil {
		t.Fatalf("runInitInteractive on existing project: %v", err)
	}

	content, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatalf("user file missing after re-init: %v", err)
	}
	if string(content) != "keep me" {
		t.Errorf("user file content = %q, want %q", content, "keep me")
	}
}

func TestExecuteInit_GitSkippedIfExists(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	// Pre-create .git directory.
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	// .git should still exist (not re-initialized).
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Error(".git should still exist")
	}
}

func TestRunUpgrade_AutoUpdate(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.2.0-test"

	dir := t.TempDir()

	// Create initial state with old version.
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	Version = "0.1.0-test"
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	// Bump version and run upgrade.
	Version = "0.2.0-test"
	if err := runUpgrade(dir, true); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// Manifest should have new version.
	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if m.Meta.Version != "0.2.0-test" {
		t.Errorf("manifest version = %q, want 0.2.0-test", m.Meta.Version)
	}
}

func TestRunUpgrade_UpdatesManagedCommitMsgHook(t *testing.T) {
	isolateGitConfig(t)
	v1 := "#!/usr/bin/env sh\n# commit-msg-guard\nexit 0\n"
	v2 := "#!/usr/bin/env sh\n# commit-msg-guard\nprintf '%s\\n' upgraded\n"
	setupTestEmbedFSWithCommitGuard(t, []byte(v1))
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	setupTestEmbedFSWithCommitGuard(t, []byte(v2))
	Version = "2.0.0-test"
	if err := runUpgrade(dir, true); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read commit-msg hook: %v", err)
	}
	if string(got) != v2 {
		t.Errorf("commit-msg hook was not updated from template script; got %q want %q", got, v2)
	}
}

func TestRunUpgrade_ChainsUserCommitMsgHook(t *testing.T) {
	isolateGitConfig(t)
	setupTestEmbedFSWithCommitGuard(t, []byte(testCommitMsgGuard))
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	hookPath := filepath.Join(dir, ".git", "hooks", "commit-msg")
	customHook := "#!/usr/bin/env sh\n# custom hook\nexit 0\n"
	if err := os.WriteFile(hookPath, []byte(customHook), 0755); err != nil {
		t.Fatalf("write custom hook: %v", err)
	}

	updatedGuard := "#!/usr/bin/env sh\n# commit-msg-guard\nprintf '%s\\n' upgraded\n"
	setupTestEmbedFSWithCommitGuard(t, []byte(updatedGuard))
	Version = "2.0.0-test"
	if err := runUpgrade(dir, true); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read commit-msg hook: %v", err)
	}
	if !strings.Contains(string(got), "ralph git hook wrapper") {
		t.Errorf("commit-msg hook was not replaced with ralph wrapper; got %q", got)
	}

	originalPath := filepath.Join(dir, ".git", "hooks", "commit-msg.ralph-original")
	original, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("read preserved original hook: %v", err)
	}
	if string(original) != customHook {
		t.Errorf("original commit-msg hook was not preserved; got %q want %q", original, customHook)
	}

	guardPath := filepath.Join(dir, ".git", "hooks", "commit-msg.ralph-guard")
	guard, err := os.ReadFile(guardPath)
	if err != nil {
		t.Fatalf("read ralph guard hook: %v", err)
	}
	if string(guard) != updatedGuard {
		t.Errorf("ralph guard hook was not updated; got %q want %q", guard, updatedGuard)
	}
}

func TestRunUpgrade_ChainsUserPreCommitHook(t *testing.T) {
	isolateGitConfig(t)
	setupTestEmbedFSWithCommitGuard(t, []byte(testCommitMsgGuard))
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	customHook := "#!/usr/bin/env sh\n# custom pre-commit\nexit 0\n"
	if err := os.WriteFile(hookPath, []byte(customHook), 0755); err != nil {
		t.Fatalf("write custom hook: %v", err)
	}

	Version = "2.0.0-test"
	if err := runUpgrade(dir, true); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read pre-commit hook: %v", err)
	}
	if !strings.Contains(string(got), "ralph git hook wrapper") {
		t.Errorf("pre-commit hook was not replaced with ralph wrapper; got %q", got)
	}

	originalPath := filepath.Join(dir, ".git", "hooks", "pre-commit.ralph-original")
	original, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("read preserved original hook: %v", err)
	}
	if string(original) != customHook {
		t.Errorf("original pre-commit hook was not preserved; got %q want %q", original, customHook)
	}

	guardPath := filepath.Join(dir, ".git", "hooks", "pre-commit.ralph-guard")
	guard, err := os.ReadFile(guardPath)
	if err != nil {
		t.Fatalf("read ralph guard hook: %v", err)
	}
	if string(guard) != testPreCommitGuard {
		t.Errorf("ralph pre-commit guard hook = %q, want %q", guard, testPreCommitGuard)
	}
}

// Regression: upgrading across the same version twice must not drift the
// manifest into empty-hash entries or re-prompt the user for unchanged files.
func TestRunUpgrade_SameVersionIsIdempotent(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	// Same-version upgrade twice.
	if err := runUpgrade(dir, false); err != nil {
		t.Fatalf("first upgrade: %v", err)
	}
	if err := runUpgrade(dir, false); err != nil {
		t.Fatalf("second upgrade: %v", err)
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	for k, v := range m.Files {
		if v.Hash == "" {
			t.Errorf("manifest entry %q has empty hash after upgrade", k)
		}
	}
	// Pack files must be tracked under the namespaced key exactly once.
	packReadme := filepath.Join("packs", "languages", "golang", "README.md")
	if _, ok := m.Files[packReadme]; !ok {
		t.Errorf("manifest missing %s", packReadme)
	}
	if _, ok := m.Files["README.md"]; ok {
		t.Error("manifest has unprefixed README.md (pack namespace leak)")
	}
}

// Heal path: if a manifest already contains empty-hash entries (bug state),
// a single same-version upgrade should repair them without prompting the
// user for files whose disk content matches the template.
func TestRunUpgrade_HealsCorruptedManifest(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	// Corrupt the manifest: wipe all base-file hashes.
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	for k, v := range m.Files {
		if filepath.Base(k) == "AGENTS.md" || filepath.Base(k) == "CLAUDE.md" {
			v.Hash = ""
			m.Files[k] = v
		}
	}
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("Write manifest: %v", err)
	}

	// Upgrade without --force: since disk == template, heal must run without
	// prompting (stdin is a closed pipe inside tests).
	if err := runUpgrade(dir, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	m2, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest after heal: %v", err)
	}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		if m2.Files[name].Hash == "" {
			t.Errorf("%s still has empty hash after heal", name)
		}
	}
}

func TestRunUpgrade_DryRunDiff_DoesNotMutateFilesOrManifest(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	local := []byte("# local edit\n")
	if err := os.WriteFile(agents, local, 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	beforeManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var out, errOut bytes.Buffer
	err = runUpgradeIOWithOptions(dir, upgradeOptions{
		DryRun:      true,
		DiffPreview: true,
		Pager:       pagerNever,
	}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("dry-run upgrade: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != string(local) {
		t.Errorf("dry run mutated AGENTS.md: got %q want %q", got, local)
	}
	afterManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after dry run: %v", err)
	}
	if string(afterManifest) != string(beforeManifest) {
		t.Errorf("dry run mutated manifest\nbefore:\n%s\nafter:\n%s", beforeManifest, afterManifest)
	}
	if !strings.Contains(out.String(), "Upgrade preview (dry run)") {
		t.Errorf("dry run preview missing; out:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "│ -# local edit") || !strings.Contains(out.String(), "│ +# AGENTS") {
		t.Errorf("dry run diff missing expected lines; out:\n%s", out.String())
	}
}

func TestRunUpgrade_DryRunDiff_HonorsPagerAlwaysFallback(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# local edit\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PAGER", "false")
	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(dir, upgradeOptions{
		DryRun:      true,
		DiffPreview: true,
		Pager:       pagerAlways,
	}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("dry-run upgrade: %v", err)
	}

	if !strings.Contains(errOut.String(), "pager failed") {
		t.Errorf("dry-run diff should still honor --pager=always and report fallback; errOut:\n%s", errOut.String())
	}
	if !strings.Contains(out.String(), "--- local") || !strings.Contains(out.String(), "│ -# local edit") {
		t.Errorf("dry-run pager fallback should still write the diff to stdout; out:\n%s", out.String())
	}
}

func TestRunUpgrade_DoesNotAddUnselectedPackRules(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	if err := runUpgrade(dir, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	for _, path := range []string{
		filepath.Join(".claude", "rules", "ralph", "golang.md"),
		filepath.Join(".claude", "rules", "ralph", "typescript.md"),
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Errorf("unselected pack rule %s should not be rendered; stat err = %v", path, err)
		}
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	for _, path := range []string{
		filepath.Join(".claude", "rules", "ralph", "golang.md"),
		filepath.Join(".claude", "rules", "ralph", "typescript.md"),
	} {
		if _, ok := m.Files[path]; ok {
			t.Errorf("manifest should not track unselected pack rule %s", path)
		}
	}
}

// TestRunUpgrade_RelocatesInstalledPackRuleFromLegacyBasePath exercises the
// legacy upgrade engine's behavior when a pack's rule.md control file used
// to live directly at .claude/rules/<lang>.md (base-template-owned, no
// rule.md in the pack itself yet) and a later template gains a real pack
// rule at the new .claude/rules/ralph/<lang>.md location. Because the two
// keys differ, the diff engine has no rename detection: it flags the old
// base-level file "removed from template" and adds the new pack-namespaced
// rule fresh. This is expected, not a regression — see docs/specs
// 2026-08-17-overlay-scaffold-v2.md's "旧レイアウト下流の移行ロジック
// (Phase 4)" non-goal for where smarter relocation logic belongs.
func TestRunUpgrade_RelocatesInstalledPackRuleFromLegacyBasePath(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/AGENTS.md":                   {Data: []byte("# AGENTS\n")},
		"templates/base/CLAUDE.md":                   {Data: []byte("# CLAUDE\n")},
		"templates/base/ralph.toml":                  {Data: []byte("[pipeline]\nmodel = \"test\"\n")},
		"templates/base/.claude/settings.json":       {Data: []byte("{}\n")},
		"templates/base/.claude/rules/golang.md":     {Data: []byte("# Old Go rule\n")},
		"templates/packs/golang/verify.sh":           {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/golang/README.md":           {Data: []byte("# Go\n")},
		"templates/packs/typescript/verify.sh":       {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/typescript/README.md":       {Data: []byte("# TS\n")},
		"templates/packs/typescript/rule.md":         {Data: []byte(testTypescriptRule)},
		"templates/base/scripts/commit-msg-guard.sh": {Data: []byte(testCommitMsgGuard)},
	}
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("legacy init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	setupTestEmbedFS(t)
	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !strings.Contains(out.String(), ".claude/rules/golang.md") || !strings.Contains(out.String(), "removed from template") {
		t.Errorf("stale legacy base rule should be reported removed; out:\n%s", out.String())
	}

	got, err := os.ReadFile(filepath.Join(dir, ".claude", "rules", "ralph", "golang.md"))
	if err != nil {
		t.Fatalf("read relocated rule: %v", err)
	}
	if string(got) != testGoRule {
		t.Errorf("relocated golang rule = %q, want pack rule", got)
	}
	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	newRulePath := filepath.Join(".claude", "rules", "ralph", "golang.md")
	if _, ok := m.Files[newRulePath]; !ok {
		t.Fatalf("manifest missing relocated rule %s", newRulePath)
	}
	if _, ok := m.Files[filepath.Join(".claude", "rules", "golang.md")]; ok {
		t.Fatal("manifest should not keep tracking the stale legacy base rule path")
	}
	if _, ok := m.Files[filepath.Join("packs", "languages", "golang", "rule.md")]; ok {
		t.Fatal("manifest should not track packs/languages/golang/rule.md after relocation")
	}
}

func TestRunUpgrade_DropsLegacyUninstalledBaseRule(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/AGENTS.md":               {Data: []byte("# AGENTS\n")},
		"templates/base/CLAUDE.md":               {Data: []byte("# CLAUDE\n")},
		"templates/base/ralph.toml":              {Data: []byte("[pipeline]\nmodel = \"test\"\n")},
		"templates/base/.claude/settings.json":   {Data: []byte("{}\n")},
		"templates/base/.claude/rules/golang.md": {Data: []byte("# Old Go rule\n")},
		"templates/packs/golang/verify.sh":       {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/golang/README.md":       {Data: []byte("# Go\n")},
		"templates/packs/golang/rule.md":         {Data: []byte(testGoRule)},
	}
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("legacy init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	setupTestEmbedFS(t)
	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !strings.Contains(out.String(), ".claude/rules/golang.md") || !strings.Contains(out.String(), "removed from template") {
		t.Errorf("uninstalled legacy base rule should be reported removed once; out:\n%s", out.String())
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if _, ok := m.Files[filepath.Join(".claude", "rules", "golang.md")]; ok {
		t.Fatal("legacy uninstalled base rule should be dropped from manifest")
	}
}

// Regression: when a pack was removed/renamed in a later release
// (scaffold.AvailablePacks no longer contains it), upgrade must drop the
// manifest tracking and the Meta.Packs entry rather than carrying a stale
// pack forward forever.
func TestRunUpgrade_DropsPacksRemovedFromTemplates(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	// Inject a pack that was once installed but no longer exists in templates.
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	m.Meta.Packs = []string{"golang", "ghostpack"}
	ghostEntry := filepath.Join("packs", "languages", "ghostpack", "verify.sh")
	golangEntry := filepath.Join("packs", "languages", "golang", "README.md")
	m.SetFile(ghostEntry, "sha256:deadbeef")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("Write manifest: %v", err)
	}

	if err := runUpgrade(dir, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	m2, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if _, ok := m2.Files[ghostEntry]; ok {
		t.Errorf("%s should be dropped when pack is absent from templates", ghostEntry)
	}
	if _, ok := m2.Files[golangEntry]; !ok {
		t.Errorf("%s was dropped — only the removed pack should drop", golangEntry)
	}
	ghostFound := false
	golangFound := false
	for _, p := range m2.Meta.Packs {
		if p == "ghostpack" {
			ghostFound = true
		}
		if p == "golang" {
			golangFound = true
		}
	}
	if ghostFound {
		t.Error("ghostpack should be removed from Meta.Packs")
	}
	if !golangFound {
		t.Error("golang should be retained in Meta.Packs")
	}
}

// Regression: a file dropped from a pack template but still tracked in the
// manifest must surface as ActionRemove (namespaced pack path) on the first
// upgrade, and the entry must be dropped from the manifest afterwards so a
// second same-version upgrade does NOT re-emit the notice.
func TestRunUpgrade_ReportsDeletedPackFileOnceThenDrops(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	deprecatedEntry := filepath.Join("packs", "languages", "golang", "deprecated.sh")
	m.SetFile(deprecatedEntry, "sha256:cafef00d")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("Write manifest: %v", err)
	}

	// Capture stdout of the first upgrade to assert the user-facing notice.
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	upgradeErr := runUpgrade(dir, false)
	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)
	if upgradeErr != nil {
		t.Fatalf("first upgrade: %v", upgradeErr)
	}
	if !strings.Contains(string(out), deprecatedEntry) {
		t.Errorf("first upgrade stdout missing pack-scoped remove notice for %s; got:\n%s", deprecatedEntry, out)
	}

	m2, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest after first upgrade: %v", err)
	}
	if _, ok := m2.Files[deprecatedEntry]; ok {
		t.Errorf("%s should be dropped from manifest after ActionRemove (idempotency)", deprecatedEntry)
	}

	// Second same-version upgrade must NOT re-emit the notice.
	r2, w2, _ := os.Pipe()
	os.Stdout = w2
	err = runUpgrade(dir, false)
	_ = w2.Close()
	os.Stdout = origStdout
	out2, _ := io.ReadAll(r2)
	if err != nil {
		t.Fatalf("second upgrade: %v", err)
	}
	if strings.Contains(string(out2), "removed from template") {
		t.Errorf("second same-version upgrade re-emitted removal notice; got:\n%s", out2)
	}
}

// Regression (round 3 codex): if scaffold.AvailablePacks() fails (e.g. the
// embedded template FS has no templates/packs directory), runUpgrade must
// still complete for base files and preserve installed pack manifest
// entries, not abort with an error.
func TestRunUpgrade_SurvivesAvailablePacksFailure(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	// Swap embedded FS to one that has no templates/packs directory at all —
	// AvailablePacks() will error on ReadDir.
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/AGENTS.md":             {Data: []byte("# AGENTS\n")},
		"templates/base/CLAUDE.md":             {Data: []byte("# CLAUDE\n")},
		"templates/base/ralph.toml":            {Data: []byte("[pipeline]\nmodel = \"test\"\n")},
		"templates/base/.claude/settings.json": {Data: []byte("{}\n")},
	}
	t.Cleanup(func() { setupTestEmbedFS(t) })

	if err := runUpgrade(dir, false); err != nil {
		t.Fatalf("upgrade should not abort on AvailablePacks failure: %v", err)
	}

	// Manifest must still track golang pack entries (preservation path).
	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	golangReadme := filepath.Join("packs", "languages", "golang", "README.md")
	if _, ok := m.Files[golangReadme]; !ok {
		t.Errorf("pack entry %s dropped after AvailablePacks failure — expected preservation", golangReadme)
	}
	found := false
	for _, p := range m.Meta.Packs {
		if p == "golang" {
			found = true
		}
	}
	if !found {
		t.Error("golang missing from Meta.Packs after AvailablePacks failure")
	}
}

// Force flag must overwrite local edits without prompting. Verifies the
// non-interactive regression path for users who explicitly opt in to
// template-wins behavior.
func TestRunUpgrade_ForceOverwritesLocalEdit(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	// User edits a managed file.
	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# local edit\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runUpgrade(dir, true); err != nil {
		t.Fatalf("upgrade --force: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != "# AGENTS\n" {
		t.Errorf("AGENTS.md = %q, want template content restored", got)
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if !m.Files["AGENTS.md"].Managed {
		t.Errorf("force overwrite should keep AGENTS.md Managed=true")
	}
}

// Interactive "overwrite" path: disk returns to template content and the
// manifest stays Managed=true so subsequent template changes auto-update.
func TestRunUpgrade_InteractiveOverwrite_WritesManaged(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# local edit\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("a\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != "# AGENTS\n" {
		t.Errorf("AGENTS.md = %q, want template content", got)
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := m.Files["AGENTS.md"]
	if !entry.Managed {
		t.Errorf("AGENTS.md.Managed = false after overwrite, want true")
	}
	if entry.Hash != scaffold.HashBytes([]byte("# AGENTS\n")) {
		t.Errorf("AGENTS.md hash not updated to template hash: got %q", entry.Hash)
	}
	if entry.BaselineStatus != scaffold.BaselineStatusAvailable {
		t.Errorf("AGENTS.md baseline_status = %q, want available", entry.BaselineStatus)
	}
}

// Interactive "keep local file" path: disk is left as-is and the manifest
// records the file as a managed partial resolution against the new baseline.
func TestRunUpgrade_InteractiveKeep_RecordsPartialManaged(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	local := []byte("# local edit\n")
	if err := os.WriteFile(agents, local, 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("k\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !strings.Contains(out.String(), "Keep local:     1 files") {
		t.Errorf("keep-local summary missing; got:\n%s", out.String())
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != string(local) {
		t.Errorf("AGENTS.md = %q, want local content preserved", got)
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := m.Files["AGENTS.md"]
	if !entry.Managed || entry.State != scaffold.FileStatePartial {
		t.Errorf("AGENTS.md state = managed:%v state:%q, want managed partial", entry.Managed, entry.State)
	}
	if entry.Hash != scaffold.HashBytes([]byte("# AGENTS\n")) || entry.TemplateHash != scaffold.HashBytes([]byte("# AGENTS\n")) {
		t.Errorf("template hash fields = hash:%q template:%q, want template hash", entry.Hash, entry.TemplateHash)
	}
	if entry.DiskHash != scaffold.HashBytes(local) {
		t.Errorf("AGENTS.md disk_hash = %q, want disk hash %q", entry.DiskHash, scaffold.HashBytes(local))
	}
}

// Interactive "diff" path: the prompt renders a unified diff, then continues
// to ask until the user picks overwrite or skip. Verifies both the diff
// contents and the re-prompt behavior.
func TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# my agents\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("k\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	combined := out.String()
	for _, want := range []string{
		"--- local",
		"+++ template (1.0.0-test)",
		"│ -# my agents",
		"│ +# AGENTS",
	} {
		if !strings.Contains(combined, want) {
			t.Errorf("diff output missing %q; got:\n%s", want, combined)
		}
	}
	diffIdx := strings.Index(combined, "--- local")
	promptIdx := strings.Index(combined, "[a]pply template file")
	if diffIdx < 0 || promptIdx < 0 || diffIdx > promptIdx {
		t.Errorf("file diff should be visible before the command prompt; got:\n%s", combined)
	}
	for _, notWant := range []string{
		"@@ ",
		"template hash:",
		"local hash:",
	} {
		if strings.Contains(combined, notWant) {
			t.Errorf("diff output should not include %q; got:\n%s", notWant, combined)
		}
	}
	// Non-TTY destination (bytes.Buffer) and colorize=false must not emit ANSI.
	if strings.Contains(combined, "\x1b[") {
		t.Errorf("ANSI escape leaked into non-TTY output:\n%q", combined)
	}
}

// When colorize is true the diff render must wrap recognized lines in ANSI
// escapes so terminal users get a readable color-coded view.
func TestRunUpgrade_InteractiveDiff_ColorizesWhenEnabled(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# my agents\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("k\ny\n"), &out, &errOut, true); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "\x1b[1;31m--- local") {
		t.Errorf("expected bold-red --- header; got:\n%q", got)
	}
	if !strings.Contains(got, "\x1b[1;32m+++ template (1.0.0-test)") {
		t.Errorf("expected bold-green +++ header; got:\n%q", got)
	}
	if strings.Contains(got, "\x1b[36m@@ ") || strings.Contains(got, "@@ ") {
		t.Errorf("upgrade UI should omit range headers; got:\n%q", got)
	}
	if !strings.Contains(got, "\x1b[31m") || !strings.Contains(got, "\x1b[32m") {
		t.Errorf("expected colored removal/addition lines; got:\n%q", got)
	}
}

func TestRunUpgrade_InteractiveDiff_IgnoresPagerAlways(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# pager should not run\n"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PAGER", "false")
	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(dir, upgradeOptions{
		Pager: pagerAlways,
	}, strings.NewReader("k\ny\n"), &out, &errOut, false)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	combined := out.String()
	if !strings.Contains(combined, "--- local") || !strings.Contains(combined, "[a]pply template file") {
		t.Errorf("interactive output should include both diff and prompt; got:\n%s", combined)
	}
	if strings.Contains(errOut.String(), "pager failed") {
		t.Errorf("interactive conflict resolution should not invoke pager; errOut:\n%s", errOut.String())
	}
}

func TestRunUpgrade_V1ManifestConflict_UsesLegacyPrompt(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := m.Files["AGENTS.md"]
	entry.State = ""
	entry.TemplateHash = ""
	entry.BaselineStatus = ""
	entry.BaselinePath = ""
	m.Files["AGENTS.md"] = entry
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("Write manifest: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# v1 local\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("s\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "[o]verwrite / [s]kip / [d]iff") {
		t.Errorf("v1 manifest should use legacy prompt; got:\n%s", got)
	}
	diffIdx := strings.Index(got, "--- local")
	promptIdx := strings.Index(got, "[o]verwrite / [s]kip / [d]iff")
	if diffIdx < 0 || promptIdx < 0 || diffIdx > promptIdx {
		t.Errorf("v1 manifest should render the diff before the command prompt; got:\n%s", got)
	}
	if strings.Contains(got, "[a]pply template file") {
		t.Errorf("v1 manifest without baseline must not enter baseline-backed prompt; got:\n%s", got)
	}
}

// Invalid baseline-backed prompt input must re-prompt without terminating.
// `edit` is currently acknowledged as a reserved option and re-prompts without
// writing.
func TestRunUpgrade_InteractiveDiff_RepromptsOnInvalid(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# drift\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	// garbage -> edit with missing editor -> keep local file
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	input := strings.NewReader("xyz\ne\nk\ny\n")
	if err := runUpgradeIO(dir, false, input, &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got := out.String()
	if strings.Count(got, "[a]pply template file / [k]eep local file / [e]dit file ?") < 3 {
		t.Errorf("expected prompt to re-render on invalid and edit inputs; got:\n%s", got)
	}
	if strings.Contains(got, "[s]kip") || strings.Contains(got, "[n]ext") || strings.Contains(got, "[q]uit") {
		t.Errorf("baseline-backed prompt must not offer skip/next/quit; got:\n%s", got)
	}
}

func TestRunUpgrade_FileApply_WritesTemplateManaged(t *testing.T) {
	initial := []byte("alpha\nbase-local\nmiddle\nbase-template\nomega\n")
	local := []byte("alpha\nLOCAL\nmiddle\nbase-template\nomega\n")
	template := []byte("alpha\nTEMPLATE\nmiddle\nTEMPLATE-ONLY\nomega\n")

	setupTestEmbedFSWithAgents(t, initial)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, local, 0644); err != nil {
		t.Fatal(err)
	}

	setupTestEmbedFSWithAgents(t, template)
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("a\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != string(template) {
		t.Fatalf("AGENTS.md = %q, want template content %q", got, template)
	}

	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := m.Files["AGENTS.md"]
	if !entry.Managed || entry.State != scaffold.FileStateManaged {
		t.Fatalf("AGENTS.md state = managed:%v state:%q, want managed", entry.Managed, entry.State)
	}
	if entry.Hash != scaffold.HashBytes(template) || entry.TemplateHash != scaffold.HashBytes(template) {
		t.Errorf("template hash fields = hash:%q template:%q, want %q", entry.Hash, entry.TemplateHash, scaffold.HashBytes(template))
	}
	if entry.DiskHash != scaffold.HashBytes(template) {
		t.Errorf("DiskHash = %q, want template hash %q", entry.DiskHash, scaffold.HashBytes(template))
	}
	baseline, err := scaffold.ReadBaseline(dir, entry)
	if err != nil {
		t.Fatalf("ReadBaseline: %v", err)
	}
	if string(baseline) != string(template) {
		t.Errorf("baseline = %q, want new template", baseline)
	}

	combined := out.String()
	for _, wantText := range []string{
		"Apply summary",
		"Apply template: 1 files",
	} {
		if !strings.Contains(combined, wantText) {
			t.Errorf("output missing %q; got:\n%s", wantText, combined)
		}
	}
	for _, notWant := range []string{"[s]kip", "files /", "[n]ext", "[q]uit", "template hash:", "local hash:"} {
		if strings.Contains(combined, notWant) {
			t.Errorf("output should not include %q; got:\n%s", notWant, combined)
		}
	}

	out.Reset()
	errOut.Reset()
	if err := runUpgradeIO(dir, false, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("second upgrade: %v", err)
	}
	if strings.Contains(out.String(), "modified locally") {
		t.Errorf("partial managed file re-prompted when template and disk hashes were unchanged; got:\n%s", out.String())
	}
	if strings.Contains(errOut.String(), "non-interactive input detected") {
		t.Errorf("second upgrade should not require input; errOut:\n%s", errOut.String())
	}
}

func TestRunUpgrade_FileSummaryNo_DoesNotWrite(t *testing.T) {
	initial := []byte("base\n")
	local := []byte("LOCAL\n")
	template := []byte("TEMPLATE\n")

	setupTestEmbedFSWithAgents(t, initial)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	beforeManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest before upgrade: %v", err)
	}
	beforeEntry, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest before upgrade: %v", err)
	}
	beforeBaseline, err := scaffold.ReadBaseline(dir, beforeEntry.Files["AGENTS.md"])
	if err != nil {
		t.Fatalf("ReadBaseline before upgrade: %v", err)
	}

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, local, 0644); err != nil {
		t.Fatal(err)
	}

	setupTestEmbedFSWithAgents(t, template)
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("a\nn\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != string(local) {
		t.Errorf("AGENTS.md = %q, want local content untouched", got)
	}
	afterManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after upgrade: %v", err)
	}
	if string(afterManifest) != string(beforeManifest) {
		t.Errorf("manifest changed despite summary rejection\nbefore:\n%s\nafter:\n%s", beforeManifest, afterManifest)
	}
	afterEntry, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest after upgrade: %v", err)
	}
	afterBaseline, err := scaffold.ReadBaseline(dir, afterEntry.Files["AGENTS.md"])
	if err != nil {
		t.Fatalf("ReadBaseline after upgrade: %v", err)
	}
	if string(afterBaseline) != string(beforeBaseline) {
		t.Errorf("baseline changed despite summary rejection: got %q want %q", afterBaseline, beforeBaseline)
	}
	if !strings.Contains(out.String(), "No changes applied.") {
		t.Errorf("output missing cancellation notice; got:\n%s", out.String())
	}
}

func TestRunUpgrade_FileEdit_UsesEditor(t *testing.T) {
	initial := []byte("base\n")
	local := []byte("LOCAL\n")
	template := []byte("TEMPLATE\n")
	edited := []byte("EDITED\n")

	setupTestEmbedFSWithAgents(t, initial)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, local, 0644); err != nil {
		t.Fatal(err)
	}

	setupTestEmbedFSWithAgents(t, template)
	Version = "2.0.0-test"

	editorDir := t.TempDir()
	editor := filepath.Join(editorDir, "editor.sh")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf '%s\\n' 'EDITED' > \"$1\"\n"), 0755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	t.Setenv("VISUAL", editor)
	t.Setenv("EDITOR", "")

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("e\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != string(edited) {
		t.Errorf("AGENTS.md = %q, want edited file %q", got, edited)
	}
	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := m.Files["AGENTS.md"]
	if entry.State != scaffold.FileStatePartial || entry.DiskHash != scaffold.HashBytes(edited) {
		t.Errorf("entry = %+v, want partial with edited disk hash", entry)
	}
	if !strings.Contains(out.String(), "Edited:         1 files") {
		t.Errorf("summary missing edited file count; got:\n%s", out.String())
	}
}

func TestRunUpgrade_FileEdit_SeedsConflictMarkersWhenLocalSideEmpty(t *testing.T) {
	initial := []byte("alpha\nremove-me\nomega\n")
	local := []byte("alpha\nomega\n")
	template := []byte("alpha\ntemplate replacement\nomega\n")
	edited := []byte("EDITED\n")

	setupTestEmbedFSWithAgents(t, initial)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, local, 0644); err != nil {
		t.Fatal(err)
	}

	setupTestEmbedFSWithAgents(t, template)
	Version = "2.0.0-test"

	editorDir := t.TempDir()
	capturedSeed := filepath.Join(editorDir, "seed.txt")
	editor := filepath.Join(editorDir, "editor.sh")
	editorScript := "#!/bin/sh\ncat \"$1\" > \"$RALPH_CAPTURE_EDIT_SEED\"\nprintf '%s\\n' 'EDITED' > \"$1\"\n"
	if err := os.WriteFile(editor, []byte(editorScript), 0755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	t.Setenv("RALPH_CAPTURE_EDIT_SEED", capturedSeed)
	t.Setenv("VISUAL", editor)
	t.Setenv("EDITOR", "")

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("e\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	seed, err := os.ReadFile(capturedSeed)
	if err != nil {
		t.Fatalf("read captured editor seed: %v", err)
	}
	wantSeed := "alpha\n<<<<<<< local\n=======\ntemplate replacement\n>>>>>>> template (2.0.0-test)\nomega\n"
	if string(seed) != wantSeed {
		t.Fatalf("editor seed = %q, want full-file conflict marker block %q", seed, wantSeed)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != string(edited) {
		t.Errorf("AGENTS.md = %q, want edited file %q", got, edited)
	}
}

func TestRunUpgrade_FileEdit_RejectsUnresolvedConflictMarkers(t *testing.T) {
	initial := []byte("base\n")
	local := []byte("LOCAL\n")
	template := []byte("TEMPLATE\n")
	edited := []byte("EDITED\n")

	setupTestEmbedFSWithAgents(t, initial)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, local, 0644); err != nil {
		t.Fatal(err)
	}

	setupTestEmbedFSWithAgents(t, template)
	Version = "2.0.0-test"

	editorDir := t.TempDir()
	countFile := filepath.Join(editorDir, "count")
	editor := filepath.Join(editorDir, "editor.sh")
	editorScript := "#!/bin/sh\nif [ ! -f \"$RALPH_EDIT_COUNT\" ]; then touch \"$RALPH_EDIT_COUNT\"; exit 0; fi\nprintf '%s\\n' 'EDITED' > \"$1\"\n"
	if err := os.WriteFile(editor, []byte(editorScript), 0755); err != nil {
		t.Fatalf("write editor script: %v", err)
	}
	t.Setenv("RALPH_EDIT_COUNT", countFile)
	t.Setenv("VISUAL", editor)
	t.Setenv("EDITOR", "")

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("e\ne\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != string(edited) {
		t.Errorf("AGENTS.md = %q, want edited file %q", got, edited)
	}
	if !strings.Contains(errOut.String(), "unresolved conflict markers remain") {
		t.Errorf("missing unresolved marker warning; errOut:\n%s", errOut.String())
	}
	if strings.Count(out.String(), "[a]pply template file / [k]eep local file / [e]dit file ?") < 2 {
		t.Errorf("expected edit prompt to reappear after unresolved markers; got:\n%s", out.String())
	}
}

// --force must re-adopt files previously marked Managed=false.
// Otherwise the flag's "overwrite all files without prompting" contract is
// broken: the user has no single-command path to restore template coverage.
func TestRunUpgrade_ForceReadoptsUnmanaged(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# local edit\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Setup: simulate a user-owned file from a legacy skip path.
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m1, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	m1.SetFileUnmanaged("AGENTS.md", scaffold.HashBytes([]byte("# local edit\n")))
	if err := m1.Write(manifestPath); err != nil {
		t.Fatalf("write unmanaged manifest: %v", err)
	}
	if m1.Files["AGENTS.md"].Managed {
		t.Fatalf("setup: expected AGENTS.md to be unmanaged")
	}

	// Second upgrade with --force must overwrite and re-manage.
	if err := runUpgrade(dir, true); err != nil {
		t.Fatalf("force upgrade: %v", err)
	}

	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(got) != "# AGENTS\n" {
		t.Errorf("AGENTS.md = %q, want template content restored by --force", got)
	}

	m2, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := m2.Files["AGENTS.md"]
	if !entry.Managed {
		t.Errorf("AGENTS.md.Managed = false after --force, want true (re-adopted)")
	}
	if entry.Hash != scaffold.HashBytes([]byte("# AGENTS\n")) {
		t.Errorf("AGENTS.md hash not restored to template hash: got %q", entry.Hash)
	}
}

// When a file the user owns (Managed=false) is deleted from the template,
// the manifest must keep the entry so a later reintroduction of the same
// path still silent-skips — not re-add or re-conflict.
func TestRunUpgrade_UnmanagedSurvivesTemplateRemovalAcrossRuns(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# my variant\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Simulate a user-owned file from a legacy skip path.
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	m.SetFileUnmanaged("AGENTS.md", scaffold.HashBytes([]byte("# my variant\n")))
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("write unmanaged manifest: %v", err)
	}

	// Simulate a later release that no longer ships AGENTS.md.
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/CLAUDE.md":             {Data: []byte("# CLAUDE\n")},
		"templates/base/ralph.toml":            {Data: []byte("[pipeline]\nmodel = \"test\"\n")},
		"templates/base/.claude/settings.json": {Data: []byte("{}\n")},
		"templates/packs/golang/verify.sh":     {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/golang/README.md":     {Data: []byte("# Go\n")},
		"templates/packs/typescript/verify.sh": {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/typescript/README.md": {Data: []byte("# TS\n")},
	}
	t.Cleanup(func() { setupTestEmbedFS(t) })

	var out, errOut bytes.Buffer
	out.Reset()
	errOut.Reset()
	if err := runUpgradeIO(dir, false, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade after removal: %v", err)
	}

	// AGENTS.md must NOT be reported as removed — it is user-owned.
	if strings.Contains(out.String(), "AGENTS.md") && strings.Contains(out.String(), "removed from template") {
		t.Errorf("unmanaged entry surfaced as ActionRemove; out:\n%s", out.String())
	}

	m2, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry, ok := m2.Files["AGENTS.md"]
	if !ok {
		t.Fatal("unmanaged entry dropped when template removed the path")
	}
	if entry.Managed {
		t.Errorf("unmanaged entry flipped to Managed=true across template removal")
	}
}

// Convergence: after keeping a local file, running the same upgrade again must
// not re-prompt because the local disk hash is recorded as a partial managed
// resolution.
func TestRunUpgrade_NextRunAfterKeep_IsSilent(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# local edit\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, strings.NewReader("k\ny\n"), &out, &errOut, false); err != nil {
		t.Fatalf("first upgrade: %v", err)
	}

	out.Reset()
	errOut.Reset()

	// Empty stdin: if the second run re-prompts, the EOF branch would print a
	// non-interactive warning. No prompt means no such output.
	if err := runUpgradeIO(dir, false, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("second upgrade: %v", err)
	}

	if strings.Contains(out.String(), "modified locally") {
		t.Errorf("second upgrade re-prompted for skipped file; got:\n%s", out.String())
	}
	if strings.Contains(errOut.String(), "non-interactive input detected") {
		t.Errorf("second upgrade hit the non-interactive branch; got:\n%s", errOut.String())
	}
}

// If the local file vanishes between diff computation and the prompt render,
// showDiff must fall back to a warning and let the user continue choosing
// rather than abort the whole upgrade.
func TestRunUpgrade_DiskReadFailure_FallsBackToWarning(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "1.0.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	clearManifestLayoutV2(t, dir)

	agents := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("# will be removed mid-run\n"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := m.Files["AGENTS.md"]
	entry.BaselineStatus = ""
	entry.BaselinePath = ""
	m.Files["AGENTS.md"] = entry
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("Write manifest: %v", err)
	}

	// removingReader simulates the file being deleted after diff computation
	// but before the user's `d` input reaches the prompt handler.
	reader := &removingReader{
		script: []string{"d\n", "s\n"},
		onFirst: func() {
			_ = os.Remove(agents)
		},
	}

	var out, errOut bytes.Buffer
	if err := runUpgradeIO(dir, false, reader, &out, &errOut, false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if !strings.Contains(errOut.String(), "could not read") {
		t.Errorf("expected disk-read fallback warning; errOut:\n%s", errOut.String())
	}
	if strings.Contains(out.String(), "template hash:") || strings.Contains(out.String(), "local hash:") {
		t.Errorf("hash fallback summary should stay out of normal UI; out:\n%s", out.String())
	}
}

// removingReader yields one scripted input line per Read call and fires the
// onFirst hook before the first line, letting tests inject mid-prompt
// filesystem changes.
type removingReader struct {
	script  []string
	onFirst func()
	called  bool
	buf     []byte
}

func (r *removingReader) Read(p []byte) (int, error) {
	if !r.called && r.onFirst != nil {
		r.onFirst()
	}
	r.called = true
	if len(r.buf) == 0 {
		if len(r.script) == 0 {
			return 0, io.EOF
		}
		r.buf = []byte(r.script[0])
		r.script = r.script[1:]
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

func TestRunDoctor_Passes(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()

	// Init a project first.
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		path := filepath.Join(binDir, bin)
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho '"+bin+" test-version'\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", binDir)

	// Doctor should not error fatally (it may warn about missing claude CLI).
	// We just verify it doesn't panic.
	_ = runDoctor(dir)
}

// TestProbeBinary_BrokenShimFails covers the codex-cross-review finding that
// `LookPath("codex")` is not enough — a broken shim on PATH lets `ralph
// doctor` falsely report `pass` while every subsequent /work or /cross-review
// invocation crashes. probeBinary must run `<bin> --version` and surface the
// failure so doctor can warn or fail.
func TestProbeBinary_BrokenShimFails(t *testing.T) {
	dir := t.TempDir()
	shim := filepath.Join(dir, "claude")
	// Shim that exits non-zero on any invocation, simulating a stale entry
	// script that lost its target.
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho 'shim broken' >&2\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	// Point PATH at this directory only so the probe finds the shim.
	t.Setenv("PATH", dir)

	if _, err := probeBinary("claude"); err == nil {
		t.Fatal("probeBinary returned nil error for a broken shim — should have surfaced --version failure")
	}
}

// TestProbeBinary_MissingBinary distinguishes "not on PATH" from "shim
// broken". Both should be errors but for different reasons; the test pins the
// LookPath branch so a future refactor cannot silently swallow it.
func TestProbeBinary_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	if _, err := probeBinary("claude"); err == nil {
		t.Fatal("probeBinary returned nil error for missing binary")
	}
}

// TestCheckCodexEffectiveConfig_MissingFile asserts that we degrade to a
// warning (not a fail) when the project has no .codex/config.toml — the
// .codex/ tree is template-driven, so a missing file just means the user has
// not run `ralph init` or `ralph upgrade` against this revision yet.
func TestCheckCodexEffectiveConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "not found") {
		t.Errorf("detail = %q, want substring 'not found'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_MissingFeatureFlag_Warns covers the failure
// mode Codex documents explicitly: project [hooks] are silently ignored unless
// `[features] hooks = true` is set. Doctor must surface this as a warn
// even when [hooks] are otherwise wired up.
func TestCheckCodexEffectiveConfig_MissingFeatureFlag_Warns(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	contents := `model = "gpt-5.5"

[hooks]
[[hooks.PostToolUse]]
command = ["./scripts/hello.sh"]
`
	if err := os.WriteFile(filepath.Join(dir, ".codex", "config.toml"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "[features] hooks = true") {
		t.Errorf("detail = %q, want substring '[features] hooks = true'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_DeprecatedFeatureFlag_Warns ensures the old
// `[features] codex_hooks` flag does not satisfy the current Codex config
// contract.
func TestCheckCodexEffectiveConfig_DeprecatedFeatureFlag_Warns(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	contents := `model = "gpt-5.5"

[features]
codex_hooks = true

[[hooks.PostToolUse]]
command = ["./scripts/check_mojibake.sh"]
`
	if err := os.WriteFile(filepath.Join(dir, ".codex", "config.toml"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "[features] hooks = true") {
		t.Errorf("detail = %q, want substring '[features] hooks = true'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_NoHooks_Warns ensures the check distinguishes
// "feature flag missing" from "no hooks wired" so the operator gets a precise
// remediation hint.
func TestCheckCodexEffectiveConfig_NoHooks_Warns(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	contents := `model = "gpt-5.5"

[features]
hooks = true
`
	if err := os.WriteFile(filepath.Join(dir, ".codex", "config.toml"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "codex trust") {
		t.Errorf("detail = %q, want substring 'codex trust'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_FullyWired exercises the success path: feature
// flag on AND at least one hook entry. The detail must remind the operator
// that effective loading still requires `codex trust .` because we cannot
// probe Codex's trust state from Go.
func TestCheckCodexEffectiveConfig_FullyWired(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	contents := `model = "gpt-5.5"

[features]
hooks = true

[[hooks.PostToolUse]]
command = ["./scripts/check_mojibake.sh"]
`
	if err := os.WriteFile(filepath.Join(dir, ".codex", "config.toml"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	r := checkCodexEffectiveConfig(dir)
	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "codex trust") {
		t.Errorf("detail must mention `codex trust .` reminder, got %q", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_DuplicateHooksJSON_Fails mirrors the Codex
// startup warning for projects that carry both hook representations in the
// same .codex layer. ralph standardizes on inline config.toml hooks, so doctor
// must catch a stray hooks.json before the next Codex launch does.
func TestCheckCodexEffectiveConfig_DuplicateHooksJSON_Fails(t *testing.T) {
	dir := t.TempDir()
	codexDir := filepath.Join(dir, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatal(err)
	}
	contents := `model = "gpt-5.5"

[features]
hooks = true

[[hooks.PostToolUse]]
command = ["./scripts/check_mojibake.sh"]
`
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "hooks.json"), []byte(`{"PostToolUse":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "fail" {
		t.Errorf("status = %q, want fail (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "hooks.json") {
		t.Errorf("detail = %q, want substring 'hooks.json'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_InvalidTOML_Fails proves the doctor surfaces
// a fail (not warn) when the TOML cannot be parsed — silently warning would
// hide a configuration error from the operator.
func TestCheckCodexEffectiveConfig_InvalidTOML_Fails(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".codex", "config.toml"), []byte("not [valid toml=="), 0644); err != nil {
		t.Fatal(err)
	}
	r := checkCodexEffectiveConfig(dir)
	if r.Status != "fail" {
		t.Errorf("status = %q, want fail", r.Status)
	}
}

// shouldColorize must respect NO_COLOR (any non-empty value disables) and
// must return false when out is nil or not a terminal. Pipes / regular files
// (the typical test path) are not terminals.
func TestShouldColorize_HonorsNoColorAndTTY(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if shouldColorize(nil) {
		t.Errorf("nil out should disable color")
	}

	// A regular temp file is not a terminal.
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if shouldColorize(f) {
		t.Errorf("regular file should not be classified as terminal")
	}

	// NO_COLOR=1 must short-circuit even when destination would otherwise be
	// eligible.
	t.Setenv("NO_COLOR", "1")
	if shouldColorize(f) {
		t.Errorf("NO_COLOR=1 must disable color regardless of destination")
	}
}

func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	root := NewRootCmd()
	expected := []string{"init", "upgrade", "doctor", "pack", "version", "status", "org"}
	for _, name := range expected {
		found := false
		for _, cmd := range root.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// TestAddPack_RendersIntoPackSubdir verifies AC1 and AC2 of the fix-known-breakage
// plan: ralph pack add <lang> must write files under packs/languages/<lang>/,
// never at the project root, map rule.md to .claude/rules/ralph/<lang>.md, and
// record namespaced manifest keys with Meta.Packs updated.
func TestAddPack_RendersIntoPackSubdir(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	// Create a project with a minimal manifest so addPack can read/update it.
	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("executeInit (seed): %v", err)
	}

	// Choose a pack that is available in the test FS ("golang").
	lang := "golang"

	if err := addPack(dir, lang); err != nil {
		t.Fatalf("addPack(%q): %v", lang, err)
	}

	// AC1a: pack payload file must exist under packs/languages/<lang>/.
	packVerify := filepath.Join(dir, "packs", "languages", lang, "verify.sh")
	if _, err := os.Stat(packVerify); err != nil {
		t.Errorf("pack payload packs/languages/%s/verify.sh missing: %v", lang, err)
	}

	// AC1b: pack payload file must NOT exist at the project root.
	rootVerify := filepath.Join(dir, "verify.sh")
	if _, err := os.Stat(rootVerify); err == nil {
		t.Errorf("verify.sh was written at project root — pack dir layout is wrong")
	}

	// AC1c: rule.md control file must render to .claude/rules/ralph/<lang>.md.
	ruleFile := filepath.Join(dir, ".claude", "rules", "ralph", lang+".md")
	if _, err := os.Stat(ruleFile); err != nil {
		t.Errorf(".claude/rules/ralph/%s.md missing: %v", lang, err)
	}

	// AC1d: rule.md must NOT appear as packs/languages/<lang>/rule.md.
	packRule := filepath.Join(dir, "packs", "languages", lang, "rule.md")
	if _, err := os.Stat(packRule); err == nil {
		t.Errorf("packs/languages/%s/rule.md should not exist (it is a control file)", lang)
	}

	// AC2a: manifest keys must be namespaced under packs/languages/<lang>/.
	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	packVerifyKey := filepath.Join("packs", "languages", lang, "verify.sh")
	if _, ok := m.Files[packVerifyKey]; !ok {
		t.Errorf("manifest missing namespaced key %q", packVerifyKey)
	}
	// Ensure no un-namespaced pack key leaks into the manifest root.
	if _, ok := m.Files["verify.sh"]; ok {
		t.Error("manifest has un-namespaced key 'verify.sh' (pack namespace leak)")
	}
	// rule.md must be tracked under .claude/rules/ralph/<lang>.md (not the pack dir).
	ruleKey := filepath.Join(".claude", "rules", "ralph", lang+".md")
	if _, ok := m.Files[ruleKey]; !ok {
		t.Errorf("manifest missing rule key %q", ruleKey)
	}
	if _, ok := m.Files[filepath.Join("packs", "languages", lang, "rule.md")]; ok {
		t.Error("manifest must not track packs/languages/<lang>/rule.md")
	}

	// AC2b: Meta.Packs must include the added lang.
	found := false
	for _, p := range m.Meta.Packs {
		if p == lang {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Meta.Packs = %v, want %q to be present", m.Meta.Packs, lang)
	}
}

// TestAddPack_MetaPacksNotDuplicated verifies that running addPack twice on
// the same lang does not duplicate the entry in Meta.Packs.
func TestAddPack_MetaPacksNotDuplicated(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	lang := "golang"
	if err := addPack(dir, lang); err != nil {
		t.Fatalf("first addPack: %v", err)
	}
	if err := addPack(dir, lang); err != nil {
		t.Fatalf("second addPack: %v", err)
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	count := 0
	for _, p := range m.Meta.Packs {
		if p == lang {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Meta.Packs has %d occurrences of %q, want 1: %v", count, lang, m.Meta.Packs)
	}
}

// TestAddPack_UnknownLangErrors verifies that addPack returns an error for an
// unrecognised language pack name (no side effects on the project directory).
func TestAddPack_UnknownLangErrors(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	err := addPack(dir, "nonexistent-lang")
	if err == nil {
		t.Fatal("addPack with unknown lang: expected error, got nil")
	}
}

// TestAddPack_V2Manifest_AssignsOwnerCore verifies that on a v2-layout
// project (Meta.Layout == scaffold.LayoutV2, the case executeInit always
// produces), addPack assigns owner=core to every manifest entry it adds:
// the pack payload files and the rule.md → .claude/rules/ralph/<lang>.md
// mapping. Without this, the Phase 3 replace planner would treat pack-added
// paths as legacy-skipped (ownerless) instead of core-managed.
func TestAddPack_V2Manifest_AssignsOwnerCore(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: nil}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}

	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest before addPack: %v", err)
	}
	if m.Meta.Layout != scaffold.LayoutV2 {
		t.Fatalf("precondition failed: Meta.Layout = %q, want %q", m.Meta.Layout, scaffold.LayoutV2)
	}

	lang := "golang"
	if err := addPack(dir, lang); err != nil {
		t.Fatalf("addPack(%q): %v", lang, err)
	}

	m, err = scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest after addPack: %v", err)
	}

	packVerifyKey := filepath.Join("packs", "languages", lang, "verify.sh")
	packReadmeKey := filepath.Join("packs", "languages", lang, "README.md")
	ruleKey := filepath.Join(".claude", "rules", "ralph", lang+".md")
	for _, key := range []string{packVerifyKey, packReadmeKey, ruleKey} {
		entry, ok := m.Files[key]
		if !ok {
			t.Errorf("manifest missing entry for %s", key)
			continue
		}
		if entry.Owner != scaffold.OwnerCore {
			t.Errorf("owner[%s] = %q, want %q", key, entry.Owner, scaffold.OwnerCore)
		}
	}
}

// TestAddPack_LegacyManifest_StaysOwnerless verifies that on a legacy
// (pre-v2) manifest — Meta.Layout unset — addPack does not write ownership
// metadata: legacy manifests have no ownership model, so entries added by
// addPack must stay ownerless, matching the entries already present from
// before the v2 layout existed.
func TestAddPack_LegacyManifest_StaysOwnerless(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}
	legacy := scaffold.NewManifest(Version)
	legacy.SetFile("AGENTS.md", "sha256:legacy")
	manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
	if err := legacy.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}

	lang := "golang"
	if err := addPack(dir, lang); err != nil {
		t.Fatalf("addPack(%q): %v", lang, err)
	}

	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifest after addPack: %v", err)
	}
	if m.Meta.Layout == scaffold.LayoutV2 {
		t.Fatalf("precondition failed: addPack must not upgrade Meta.Layout to v2 on its own")
	}

	packVerifyKey := filepath.Join("packs", "languages", lang, "verify.sh")
	ruleKey := filepath.Join(".claude", "rules", "ralph", lang+".md")
	for _, key := range []string{packVerifyKey, ruleKey} {
		entry, ok := m.Files[key]
		if !ok {
			t.Errorf("manifest missing entry for %s", key)
			continue
		}
		if entry.Owner != "" {
			t.Errorf("owner[%s] = %q, want empty (legacy manifest must stay ownerless)", key, entry.Owner)
		}
	}
}
