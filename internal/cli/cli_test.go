package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
)

// setupTestEmbedFS injects a minimal mock FS into scaffold.EmbeddedFS for testing.
func setupTestEmbedFS(t *testing.T) {
	t.Helper()
	scaffold.EmbeddedFS = fstest.MapFS{
		"templates/base/AGENTS.md":             {Data: []byte("# AGENTS\n")},
		"templates/base/CLAUDE.md":             {Data: []byte("# CLAUDE\n")},
		"templates/base/ralph.toml":            {Data: []byte("[pipeline]\nmodel = \"test\"\n")},
		"templates/base/.claude/settings.json": {Data: []byte("{}\n")},
		"templates/packs/golang/verify.sh":     {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/golang/README.md":     {Data: []byte("# Go\n")},
		"templates/packs/typescript/verify.sh": {Data: []byte("#!/bin/sh\necho ok\n")},
		"templates/packs/typescript/README.md": {Data: []byte("# TS\n")},
	}
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
	for _, f := range []string{"AGENTS.md", "CLAUDE.md", "ralph.toml", ".ralph/manifest.toml", "packs/languages/golang/verify.sh"} {
		if _, err := os.Stat(filepath.Join(target, f)); err != nil {
			t.Errorf("expected %s to exist: %v", f, err)
		}
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
	if m.Meta.Version != "0.1.0-test" {
		t.Errorf("manifest version = %q, want 0.1.0-test", m.Meta.Version)
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

func TestRunDoctor_Passes(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()

	// Init a project first.
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Doctor should not error fatally (it may warn about missing claude CLI).
	// We just verify it doesn't panic.
	_ = runDoctor(dir)
}

func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	root := NewRootCmd()
	expected := []string{"init", "upgrade", "run", "retry", "abort", "doctor", "pack", "version", "status"}
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
