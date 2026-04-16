package cli

import (
	"os"
	"path/filepath"
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
		"templates/prompts/self-review.md":     {Data: []byte("review prompt\n")},
		"templates/prompts/verify.md":          {Data: []byte("verify prompt\n")},
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
