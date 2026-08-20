package cli

import (
	"errors"
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
	// AC-11: `ralph init` writes no baseline — the mechanism was removed in
	// Phase 3 (docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md,
	// FR-13). BaselineStatus/BaselinePath are legacy-read-only fields; a
	// fresh v2 init must leave them at SetFile's default ("missing" / empty)
	// and must not create the .ralph/baseline/ directory at all.
	agentsEntry := m.Files["AGENTS.md"]
	if agentsEntry.BaselineStatus != scaffold.BaselineStatusMissing {
		t.Errorf("AGENTS.md baseline_status = %q, want %q (init writes no baseline)", agentsEntry.BaselineStatus, scaffold.BaselineStatusMissing)
	}
	if agentsEntry.BaselinePath != "" {
		t.Errorf("AGENTS.md baseline_path = %q, want empty (init writes no baseline)", agentsEntry.BaselinePath)
	}
	if _, err := os.Stat(filepath.Join(target, ".ralph", "baseline")); !os.IsNotExist(err) {
		t.Errorf(".ralph/baseline must not exist after init; stat err = %v", err)
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

// validHooksJSON is the shipped-form fixture: a schema-valid hooks.json
// whose PostToolUse handler routes through ralph-dispatch.sh.
const validHooksJSON = `{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write|MultiEdit|apply_patch",
        "hooks": [
          {
            "type": "command",
            "command": "\"$(git rev-parse --show-toplevel)/.claude/hooks/ralph-dispatch.sh\" PostToolUse"
          }
        ]
      }
    ]
  }
}`

func writeCodexConfigToml(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".codex", "config.toml"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeCodexHooksJSON(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".codex", "hooks.json"), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckCodexEffectiveConfig_MissingConfigToml_Warns asserts that we
// degrade to a warning (not a fail) when the project has no
// .codex/config.toml — the .codex/ tree is template-driven, so a missing
// file just means the user has not run `ralph init` or `ralph upgrade`
// against this revision yet.
func TestCheckCodexEffectiveConfig_MissingConfigToml_Warns(t *testing.T) {
	dir := t.TempDir()
	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "not found") {
		t.Errorf("detail = %q, want substring 'not found'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_HooksFeatureAbsent_Lenient covers the Slice 1
// open-questions resolution: the official hooks doc does not specify a
// default when `[features] hooks` is omitted, so doctor must not warn about
// an absent key — only an explicit `false` is a finding.
func TestCheckCodexEffectiveConfig_HooksFeatureAbsent_Lenient(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"
`)
	writeCodexHooksJSON(t, dir, validHooksJSON)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_HooksFeatureExplicitFalse_Warns asserts that
// an explicit `[features] hooks = false` is surfaced as a warn — Codex
// project hooks are disabled outright in that case.
func TestCheckCodexEffectiveConfig_HooksFeatureExplicitFalse_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"

[features]
hooks = false
`)
	writeCodexHooksJSON(t, dir, validHooksJSON)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "hooks = false") {
		t.Errorf("detail = %q, want substring 'hooks = false'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_DeprecatedFeatureFlagKey_TreatedAsAbsent
// ensures the old `[features] codex_hooks` flag does not satisfy the real
// `hooks` key — but because an absent key is lenient (see
// TestCheckCodexEffectiveConfig_HooksFeatureAbsent_Lenient), this must not
// produce a warn either; codex_hooks is simply ignored.
func TestCheckCodexEffectiveConfig_DeprecatedFeatureFlagKey_TreatedAsAbsent(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"

[features]
codex_hooks = true
`)
	writeCodexHooksJSON(t, dir, validHooksJSON)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_HooksJSONMissing_Warns ensures a
// structurally fine config.toml with no .codex/hooks.json is flagged —
// hooks.json is the source of truth, so its absence means Codex hooks are
// not wired at all.
func TestCheckCodexEffectiveConfig_HooksJSONMissing_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"

[features]
hooks = true
`)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "hooks.json") || !strings.Contains(r.Detail, "not found") {
		t.Errorf("detail = %q, want substrings 'hooks.json' and 'not found'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_FullyWired_Pass exercises the success path:
// a schema-valid hooks.json with a PostToolUse handler routed through
// ralph-dispatch.sh, and no stale config.toml [hooks] table. The detail
// must remind the operator that effective loading still requires
// `codex trust .` because we cannot probe Codex's trust state from Go.
func TestCheckCodexEffectiveConfig_FullyWired_Pass(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"

[features]
hooks = true
`)
	writeCodexHooksJSON(t, dir, validHooksJSON)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "codex trust") {
		t.Errorf("detail must mention `codex trust .` reminder, got %q", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_DualRepresentation_Warns mirrors the Codex
// startup warning for projects that carry both hook representations at
// once. hooks.json is the source of truth now, so a surviving config.toml
// [hooks] table is stale duplication doctor must flag.
func TestCheckCodexEffectiveConfig_DualRepresentation_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"

[features]
hooks = true

[[hooks.PostToolUse]]
command = ["./scripts/check_mojibake.sh"]
`)
	writeCodexHooksJSON(t, dir, validHooksJSON)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "hooks.json") || !strings.Contains(r.Detail, "source of truth") {
		t.Errorf("detail = %q, want substrings 'hooks.json' and 'source of truth'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_InvalidTOML_Fails proves the doctor surfaces
// a fail (not warn) when the TOML cannot be parsed — silently warning would
// hide a configuration error from the operator.
func TestCheckCodexEffectiveConfig_InvalidTOML_Fails(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, "not [valid toml==")

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "fail" {
		t.Errorf("status = %q, want fail", r.Status)
	}
}

// TestCheckCodexEffectiveConfig_HooksJSONTopLevelEventKey_Warns covers AC-3b
// schema-defect case 1: the file has the event name directly at the top
// level instead of nested under a "hooks" wrapper key.
func TestCheckCodexEffectiveConfig_HooksJSONTopLevelEventKey_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"
`)
	writeCodexHooksJSON(t, dir, `{"PostToolUse":[{"matcher":"apply_patch","hooks":[{"type":"command","command":"./.claude/hooks/ralph-dispatch.sh PostToolUse"}]}]}`)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, `"hooks"`) {
		t.Errorf("detail = %q, want substring about the missing \"hooks\" wrapper key", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_HooksJSONHooksKeyMissing_Warns covers AC-3b
// schema-defect case 2: valid JSON, but the top-level "hooks" key is simply
// absent (as opposed to case 1's misplaced sibling key).
func TestCheckCodexEffectiveConfig_HooksJSONHooksKeyMissing_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"
`)
	writeCodexHooksJSON(t, dir, `{"description":"no hooks key here"}`)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, `"hooks"`) {
		t.Errorf("detail = %q, want substring about the missing \"hooks\" key", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_HooksJSONHandlerMissingType_Warns covers
// AC-3b schema-defect case 3: a handler object with no "type" field.
func TestCheckCodexEffectiveConfig_HooksJSONHandlerMissingType_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"
`)
	writeCodexHooksJSON(t, dir, `{"hooks":{"PostToolUse":[{"matcher":"apply_patch","hooks":[{"command":"./.claude/hooks/ralph-dispatch.sh PostToolUse"}]}]}}`)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, `"type"`) {
		t.Errorf("detail = %q, want substring about the missing \"type\" field", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_HooksJSONCommandAsArray_Warns covers AC-3b
// schema-defect case 4: "command" given as an array instead of a single
// shell-evaluated string.
func TestCheckCodexEffectiveConfig_HooksJSONCommandAsArray_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"
`)
	writeCodexHooksJSON(t, dir, `{"hooks":{"PostToolUse":[{"matcher":"apply_patch","hooks":[{"type":"command","command":["./.claude/hooks/ralph-dispatch.sh","PostToolUse"]}]}]}}`)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "command") || !strings.Contains(r.Detail, "array") {
		t.Errorf("detail = %q, want substrings 'command' and 'array'", r.Detail)
	}
}

// TestCheckCodexEffectiveConfig_DispatcherRoutingMissing_Warns proves a
// schema-valid hooks.json whose PostToolUse handler does NOT route through
// ralph-dispatch.sh is still flagged — schema validity alone is not enough;
// the 3-layer .d dispatcher contract requires the dispatcher script itself.
func TestCheckCodexEffectiveConfig_DispatcherRoutingMissing_Warns(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, `model = "gpt-5.5"
`)
	writeCodexHooksJSON(t, dir, `{"hooks":{"PostToolUse":[{"matcher":"apply_patch","hooks":[{"type":"command","command":"./scripts/direct-call.sh"}]}]}}`)

	r := checkCodexEffectiveConfig(dir)
	if r.Status != "warn" {
		t.Errorf("status = %q, want warn", r.Status)
	}
	if !strings.Contains(r.Detail, "ralph-dispatch.sh") {
		t.Errorf("detail = %q, want substring 'ralph-dispatch.sh'", r.Detail)
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

// TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed pins the
// cycle-3 cross-review fix: renderMappedFile (the pack rule.md ->
// .claude/rules/ralph/<lang>.md control-file path) must apply the same
// Lstat-not-Stat guard as scaffold.RenderFS (see
// TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed). Before the fix,
// os.Stat on a dangling symlink at the rule path returns ErrNotExist,
// renderPackInto classifies it as a create, and os.WriteFile writes the pack
// rule content straight through the link to wherever it resolves --
// escaping targetDir even though the earlier isInsideDir boundary check on
// the symlink's own (in-bounds) path passed.
func TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	outsideDir := t.TempDir()
	externalTarget := filepath.Join(outsideDir, "OUTSIDE.md")

	lang := "golang"
	rulePath := filepath.Join(dir, ".claude", "rules", "ralph", lang+".md")
	if err := os.MkdirAll(filepath.Dir(rulePath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Symlink(externalTarget, rulePath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	pr, err := renderPackInto(dir, lang, false /* non-force */)
	if err != nil {
		t.Fatalf("renderPackInto: %v", err)
	}

	ruleKey := filepath.Join(".claude", "rules", "ralph", lang+".md")
	found := false
	for _, skipped := range pr.result.Skipped {
		if skipped == ruleKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("result.Skipped = %v, want %q present", pr.result.Skipped, ruleKey)
	}
	for _, created := range pr.result.Created {
		if created == ruleKey {
			t.Errorf("result.Created contains %q, want it skipped (dangling symlink must not be followed)", ruleKey)
		}
	}

	// The symlink itself must be untouched.
	info, err := os.Lstat(rulePath)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s must remain a symlink, got mode %v", ruleKey, info.Mode())
	}

	// Nothing must have been written at the external (dangling) link target
	// -- this is the containment failure the fix closes.
	if _, err := os.Stat(externalTarget); !os.IsNotExist(err) {
		t.Errorf("dangling symlink target must not be created; stat err = %v", err)
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

// TestAddPack_LegacyManifest_FailsClosedZeroWrites is AC-7 coverage for
// `ralph pack add`: on a legacy (pre-v2) manifest — Meta.Layout unset —
// addPack must refuse with errLegacyLayoutFailClosed and write nothing at
// all (no pack files rendered, manifest untouched). The legacy manifest's
// ownership model was removed alongside the legacy upgrade engine in Phase 3
// (docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md, FR-13); the
// automated migration to v2 arrives in a later ralph release (Phase 4).
func TestAddPack_LegacyManifest_FailsClosedZeroWrites(t *testing.T) {
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
	beforeManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest before addPack: %v", err)
	}

	lang := "golang"
	err = addPack(dir, lang)
	if err == nil {
		t.Fatal("addPack on a legacy manifest: expected an error, got nil")
	}
	if !errors.Is(err, errLegacyLayoutFailClosed) {
		t.Errorf("err = %v, want errors.Is(err, errLegacyLayoutFailClosed)", err)
	}

	afterManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after addPack: %v", err)
	}
	if string(afterManifest) != string(beforeManifest) {
		t.Errorf("addPack must not touch the manifest on a legacy-layout refusal\nbefore:\n%s\nafter:\n%s", beforeManifest, afterManifest)
	}

	packDir := filepath.Join(dir, "packs", "languages", lang)
	if _, statErr := os.Stat(packDir); !os.IsNotExist(statErr) {
		t.Errorf("%s must not exist after a legacy-layout refusal; stat err = %v", packDir, statErr)
	}
	ruleFile := filepath.Join(dir, ".claude", "rules", "ralph", lang+".md")
	if _, statErr := os.Stat(ruleFile); !os.IsNotExist(statErr) {
		t.Errorf("%s must not exist after a legacy-layout refusal; stat err = %v", ruleFile, statErr)
	}
}
