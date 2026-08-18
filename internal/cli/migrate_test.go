package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

func migrateHash(content string) string {
	return scaffold.HashBytes([]byte(content))
}

func writeMigrationDiskFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func findMigrationEntry(t *testing.T, plan MigrationPlan, oldPath string) MigrationEntry {
	t.Helper()
	for _, e := range plan.Entries {
		if e.OldPath == oldPath {
			return e
		}
	}
	t.Fatalf("no migration entry found for %q; entries=%+v", oldPath, plan.Entries)
	return MigrationEntry{}
}

func assertNoMigrationEntry(t *testing.T, plan MigrationPlan, oldPath string) {
	t.Helper()
	for _, e := range plan.Entries {
		if e.OldPath == oldPath {
			t.Fatalf("unexpected migration entry for %q: %+v", oldPath, e)
		}
	}
}

// --- LegacyEntryState contract ---

func TestClassifyLegacyEntryState(t *testing.T) {
	tests := []struct {
		name     string
		entry    scaffold.ManifestFile
		diskHash string
		hasDisk  bool
		want     LegacyEntryState
	}{
		{
			name:     "managed unmodified via disk_hash",
			entry:    scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged, Hash: "sha256:stale", DiskHash: "sha256:cur"},
			diskHash: "sha256:cur",
			hasDisk:  true,
			want:     LegacyUnmodified,
		},
		{
			name:     "managed unmodified via hash fallback (v1)",
			entry:    scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged, Hash: "sha256:cur"},
			diskHash: "sha256:cur",
			hasDisk:  true,
			want:     LegacyUnmodified,
		},
		{
			name:     "managed modified: disk_hash differs from disk",
			entry:    scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged, Hash: "sha256:old", DiskHash: "sha256:old"},
			diskHash: "sha256:new",
			hasDisk:  true,
			want:     LegacyModified,
		},
		{
			name:     "managed=false is unmanaged unconditionally, even with matching hash",
			entry:    scaffold.ManifestFile{Managed: false, State: scaffold.FileStateUnmanaged, Hash: "sha256:cur", DiskHash: "sha256:cur"},
			diskHash: "sha256:cur",
			hasDisk:  true,
			want:     LegacyUnmanaged,
		},
		{
			name:     "state=partial is modified unconditionally, even with matching hash",
			entry:    scaffold.ManifestFile{Managed: true, State: legacyStatePartial, Hash: "sha256:cur", DiskHash: "sha256:cur"},
			diskHash: "sha256:cur",
			hasDisk:  true,
			want:     LegacyModified,
		},
		{
			name:     "empty hash (v1 heal target) is modified even though disk matches empty",
			entry:    scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged},
			diskHash: "",
			hasDisk:  true,
			want:     LegacyModified,
		},
		{
			name:     "missing disk file is modified, not unmodified",
			entry:    scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged, Hash: "sha256:cur"},
			diskHash: "",
			hasDisk:  false,
			want:     LegacyModified,
		},
		{
			name:     "disk_hash takes precedence over stale hash even when hash would match disk",
			entry:    scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged, Hash: "sha256:disk-content", DiskHash: "sha256:recorded-edit"},
			diskHash: "sha256:disk-content",
			hasDisk:  true,
			want:     LegacyModified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLegacyEntryState(tt.entry, tt.diskHash, tt.hasDisk)
			if got != tt.want {
				t.Errorf("classifyLegacyEntryState() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Generic relocation / retirement / keep-in-place ---

func TestClassifyMigration_UnmodifiedRelocatedRule_DeletesOldPath(t *testing.T) {
	dir := t.TempDir()
	content := "rule body"
	writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", content)

	m := scaffold.NewManifest("0.9.0")
	m.SetFile(".claude/rules/architecture.md", migrateHash(content))

	desired := map[string][]byte{
		".claude/rules/ralph/architecture.md": []byte("new core content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, ".claude/rules/architecture.md")
	if e.Kind != OpDeleteOldPath {
		t.Errorf("Kind = %v, want OpDeleteOldPath", e.Kind)
	}
	if e.NewPath != ".claude/rules/ralph/architecture.md" {
		t.Errorf("NewPath = %q, want relocated path", e.NewPath)
	}
	if e.State != LegacyUnmodified {
		t.Errorf("State = %v, want LegacyUnmodified", e.State)
	}
}

func TestClassifyMigration_ModifiedRelocatedRule_ForkRelocates(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", "user edited content")

	m := scaffold.NewManifest("0.9.0")
	m.SetFile(".claude/rules/architecture.md", migrateHash("original template content"))

	desired := map[string][]byte{
		".claude/rules/ralph/architecture.md": []byte("new core content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, ".claude/rules/architecture.md")
	if e.Kind != OpForkRelocate {
		t.Errorf("Kind = %v, want OpForkRelocate", e.Kind)
	}
	if e.NewPath != ".claude/rules/ralph/architecture.md" {
		t.Errorf("NewPath = %q, want relocated path", e.NewPath)
	}
	if e.Owner != scaffold.OwnerFork {
		t.Errorf("Owner = %q, want fork", e.Owner)
	}
	if e.ForkedFromVersion != "0.9.0" {
		t.Errorf("ForkedFromVersion = %q, want 0.9.0", e.ForkedFromVersion)
	}
}

func TestClassifyMigration_UnmodifiedRetiredPath_Deletes(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, "old-retired-file.md", "content")

	m := scaffold.NewManifest("0.9.0")
	m.SetFile("old-retired-file.md", migrateHash("content"))

	desired := map[string][]byte{} // path no longer shipped at all

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, "old-retired-file.md")
	if e.Kind != OpDeleteOldPath {
		t.Errorf("Kind = %v, want OpDeleteOldPath", e.Kind)
	}
	if e.NewPath != "" {
		t.Errorf("NewPath = %q, want empty for a retired path", e.NewPath)
	}
}

func TestClassifyMigration_SamePathUnmodified_KeepsInPlace(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, "scripts/run-verify.sh", "content")

	m := scaffold.NewManifest("0.9.0")
	m.SetFile("scripts/run-verify.sh", migrateHash("content"))

	desired := map[string][]byte{
		"scripts/run-verify.sh": []byte("new template content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, "scripts/run-verify.sh")
	if e.Kind != OpKeepInPlace {
		t.Errorf("Kind = %v, want OpKeepInPlace", e.Kind)
	}
	if e.Owner != scaffold.OwnerCore {
		t.Errorf("Owner = %q, want core", e.Owner)
	}
}

func TestClassifyMigration_SamePathModified_ForksInPlace(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, "scripts/run-verify.sh", "user edited")

	m := scaffold.NewManifest("0.9.0")
	m.SetFile("scripts/run-verify.sh", migrateHash("original"))

	desired := map[string][]byte{
		"scripts/run-verify.sh": []byte("new template content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, "scripts/run-verify.sh")
	if e.Kind != OpForkInPlace {
		t.Errorf("Kind = %v, want OpForkInPlace", e.Kind)
	}
	if e.NewPath != "scripts/run-verify.sh" {
		t.Errorf("NewPath = %q, want same path", e.NewPath)
	}
	if e.Owner != scaffold.OwnerFork {
		t.Errorf("Owner = %q, want fork", e.Owner)
	}
}

func TestClassifyMigration_UnmanagedSamePath_ForksInPlace(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, "docs/notes.md", "user content")

	m := scaffold.NewManifest("0.9.0")
	m.Files["docs/notes.md"] = scaffold.ManifestFile{Managed: false, State: scaffold.FileStateUnmanaged, Hash: migrateHash("user content"), DiskHash: migrateHash("user content")}

	desired := map[string][]byte{
		"docs/notes.md": []byte("template content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, "docs/notes.md")
	if e.Kind != OpForkInPlace {
		t.Errorf("Kind = %v, want OpForkInPlace", e.Kind)
	}
	if e.State != LegacyUnmanaged {
		t.Errorf("State = %v, want LegacyUnmanaged", e.State)
	}
	if e.Owner != scaffold.OwnerFork {
		t.Errorf("Owner = %q, want fork", e.Owner)
	}
}

func TestClassifyMigration_UnmanagedRelocated_ForkRelocates(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, ".claude/rules/custom.md", "user content")

	m := scaffold.NewManifest("0.9.0")
	m.Files[".claude/rules/custom.md"] = scaffold.ManifestFile{Managed: false, State: scaffold.FileStateUnmanaged, Hash: migrateHash("user content"), DiskHash: migrateHash("user content")}

	desired := map[string][]byte{
		".claude/rules/ralph/custom.md": []byte("template content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, ".claude/rules/custom.md")
	if e.Kind != OpForkRelocate {
		t.Errorf("Kind = %v, want OpForkRelocate", e.Kind)
	}
	if e.NewPath != ".claude/rules/ralph/custom.md" {
		t.Errorf("NewPath = %q, want relocated path", e.NewPath)
	}
}

// --- Pack rules ---

func TestClassifyMigration_PackRuleWithInstalledPack(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, ".claude/rules/golang.md", "go rule content")

	m := scaffold.NewManifest("0.9.0")
	m.Meta.Packs = []string{"golang"}
	m.SetFile(".claude/rules/golang.md", migrateHash("go rule content"))

	desired := map[string][]byte{
		".claude/rules/ralph/golang.md": []byte("new pack rule content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, ".claude/rules/golang.md")
	if e.Kind != OpDeleteOldPath {
		t.Errorf("Kind = %v, want OpDeleteOldPath", e.Kind)
	}
	if e.NewPath != ".claude/rules/ralph/golang.md" {
		t.Errorf("NewPath = %q, want relocated pack rule path", e.NewPath)
	}
	if !reflect.DeepEqual(plan.Packs, []string{"golang"}) {
		t.Errorf("Packs = %v, want [golang]", plan.Packs)
	}
}

func TestClassifyMigration_PackRuleModified_ForkRelocates(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, ".claude/rules/golang.md", "user edited go rule")

	m := scaffold.NewManifest("0.9.0")
	m.Meta.Packs = []string{"golang"}
	m.SetFile(".claude/rules/golang.md", migrateHash("original go rule"))

	desired := map[string][]byte{
		".claude/rules/ralph/golang.md": []byte("new pack rule content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	e := findMigrationEntry(t, plan, ".claude/rules/golang.md")
	if e.Kind != OpForkRelocate {
		t.Errorf("Kind = %v, want OpForkRelocate", e.Kind)
	}
}

// --- Special faces (FR-8) ---

func TestClassifyMigration_SpecialFaces(t *testing.T) {
	t.Run("CLAUDE.md unmodified replaced with seed", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, "CLAUDE.md", "old seed")
		m := scaffold.NewManifest("0.9.0")
		m.SetFile("CLAUDE.md", migrateHash("old seed"))
		desired := map[string][]byte{"CLAUDE.md": []byte("new seed")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, "CLAUDE.md")
		if e.Kind != OpReplaceWithTemplate {
			t.Errorf("Kind = %v, want OpReplaceWithTemplate", e.Kind)
		}
		if e.Owner != scaffold.OwnerSeed {
			t.Errorf("Owner = %q, want seed", e.Owner)
		}
	})

	t.Run("CLAUDE.md modified is byte-untouched", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, "CLAUDE.md", "user edited")
		m := scaffold.NewManifest("0.9.0")
		m.SetFile("CLAUDE.md", migrateHash("old seed"))
		desired := map[string][]byte{"CLAUDE.md": []byte("new seed")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, "CLAUDE.md")
		if e.Kind != OpUntouched {
			t.Errorf("Kind = %v, want OpUntouched", e.Kind)
		}
	})

	for _, p := range []string{"AGENTS.md", ".gitignore"} {
		t.Run(p+" unmodified replaced with block template", func(t *testing.T) {
			dir := t.TempDir()
			writeMigrationDiskFile(t, dir, p, "old content")
			m := scaffold.NewManifest("0.9.0")
			m.SetFile(p, migrateHash("old content"))
			desired := map[string][]byte{p: []byte("new block content")}

			plan, err := ClassifyMigration(m, dir, desired)
			if err != nil {
				t.Fatalf("ClassifyMigration: %v", err)
			}
			e := findMigrationEntry(t, plan, p)
			if e.Kind != OpReplaceWithTemplate {
				t.Errorf("Kind = %v, want OpReplaceWithTemplate", e.Kind)
			}
			if e.Owner != scaffold.OwnerBlock {
				t.Errorf("Owner = %q, want block", e.Owner)
			}
		})

		t.Run(p+" modified stays in place for block append", func(t *testing.T) {
			dir := t.TempDir()
			writeMigrationDiskFile(t, dir, p, "user edited")
			m := scaffold.NewManifest("0.9.0")
			m.SetFile(p, migrateHash("old content"))
			desired := map[string][]byte{p: []byte("new block content")}

			plan, err := ClassifyMigration(m, dir, desired)
			if err != nil {
				t.Fatalf("ClassifyMigration: %v", err)
			}
			e := findMigrationEntry(t, plan, p)
			if e.Kind != OpUntouched {
				t.Errorf("Kind = %v, want OpUntouched", e.Kind)
			}
		})
	}

	t.Run("settings.json unmodified replaced with template + snapshot marker", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, pathSettings, "old settings")
		m := scaffold.NewManifest("0.9.0")
		m.SetFile(pathSettings, migrateHash("old settings"))
		desired := map[string][]byte{pathSettings: []byte("new settings")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, pathSettings)
		if e.Kind != OpReplaceWithTemplate {
			t.Errorf("Kind = %v, want OpReplaceWithTemplate", e.Kind)
		}
		if !e.SnapshotCreate {
			t.Errorf("SnapshotCreate = false, want true")
		}
	})

	t.Run("settings.json modified prunes known legacy hook commands", func(t *testing.T) {
		dir := t.TempDir()
		legacySettings := `{
  "hooks": {
    "PreToolUse": [{"hooks": [{"type": "command", "command": "./.claude/hooks/pre_bash_guard.sh"}]}],
    "SessionStart": [{"hooks": [{"type": "command", "command": "./.claude/hooks/session_start_context.sh"}]}]
  },
  "permissions": {"allow": ["Bash(git status:*)", "Bash(my-custom-tool:*)"]}
}`
		writeMigrationDiskFile(t, dir, pathSettings, legacySettings)
		m := scaffold.NewManifest("0.9.0")
		m.SetFile(pathSettings, migrateHash("original template settings"))
		desired := map[string][]byte{pathSettings: []byte("new settings")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, pathSettings)
		if e.Kind != OpSettingsPrune {
			t.Errorf("Kind = %v, want OpSettingsPrune", e.Kind)
		}
		want := []string{"./.claude/hooks/pre_bash_guard.sh", "./.claude/hooks/session_start_context.sh"}
		if !reflect.DeepEqual(e.PrunedHookCommands, want) {
			t.Errorf("PrunedHookCommands = %v, want %v", e.PrunedHookCommands, want)
		}
	})

	t.Run("settings.json modified with no legacy hooks present prunes nothing", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, pathSettings, `{"hooks": {}}`)
		m := scaffold.NewManifest("0.9.0")
		m.SetFile(pathSettings, migrateHash("original template settings"))
		desired := map[string][]byte{pathSettings: []byte("new settings")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, pathSettings)
		if len(e.PrunedHookCommands) != 0 {
			t.Errorf("PrunedHookCommands = %v, want empty", e.PrunedHookCommands)
		}
	})

	t.Run("codex override always untouched with owner=seed, even modified", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, pathCodexOverride, "user customization")
		m := scaffold.NewManifest("0.9.0")
		m.SetFile(pathCodexOverride, migrateHash("original"))
		desired := map[string][]byte{pathCodexOverride: []byte("template")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, pathCodexOverride)
		if e.Kind != OpUntouched {
			t.Errorf("Kind = %v, want OpUntouched", e.Kind)
		}
		if e.Owner != scaffold.OwnerSeed {
			t.Errorf("Owner = %q, want seed", e.Owner)
		}
	})

	t.Run("codex override untouched with owner=seed when unmodified too", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, pathCodexOverride, "original")
		m := scaffold.NewManifest("0.9.0")
		m.SetFile(pathCodexOverride, migrateHash("original"))
		desired := map[string][]byte{pathCodexOverride: []byte("template")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, pathCodexOverride)
		if e.Kind != OpUntouched {
			t.Errorf("Kind = %v, want OpUntouched", e.Kind)
		}
		if e.Owner != scaffold.OwnerSeed {
			t.Errorf("Owner = %q, want seed", e.Owner)
		}
	})
}

// --- .ralph/baseline/ removal marker ---

func TestClassifyMigration_BaselineDirMarker(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ralph", "baseline"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeMigrationDiskFile(t, dir, ".ralph/baseline/AGENTS.md", "cached template")

	m := scaffold.NewManifest("0.9.0")

	plan, err := ClassifyMigration(m, dir, map[string][]byte{})
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	e := findMigrationEntry(t, plan, pathBaselineDir)
	if e.Kind != OpDeleteDir {
		t.Errorf("Kind = %v, want OpDeleteDir", e.Kind)
	}
}

func TestClassifyMigration_NoBaselineDir_NoMarker(t *testing.T) {
	dir := t.TempDir()
	m := scaffold.NewManifest("0.9.0")

	plan, err := ClassifyMigration(m, dir, map[string][]byte{})
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	assertNoMigrationEntry(t, plan, pathBaselineDir)
}

// --- Collision matrix (Codex advisory 3) ---

func TestClassifyMigration_CollisionMatrix(t *testing.T) {
	t.Run("(a) unmodified relocate, dest matches source content -> delete-old only", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", "same content")
		writeMigrationDiskFile(t, dir, ".claude/rules/ralph/architecture.md", "same content")

		m := scaffold.NewManifest("0.9.0")
		m.SetFile(".claude/rules/architecture.md", migrateHash("same content"))
		desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("new core content")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		if len(plan.Collisions) != 0 {
			t.Fatalf("Collisions = %+v, want none", plan.Collisions)
		}
		e := findMigrationEntry(t, plan, ".claude/rules/architecture.md")
		if e.Kind != OpDeleteOldPath {
			t.Errorf("Kind = %v, want OpDeleteOldPath", e.Kind)
		}
	})

	t.Run("(a) modified fork-relocate, dest matches source content -> delete-old, dest adopted as fork", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", "already forked content")
		writeMigrationDiskFile(t, dir, ".claude/rules/ralph/architecture.md", "already forked content")

		m := scaffold.NewManifest("0.9.0")
		m.SetFile(".claude/rules/architecture.md", migrateHash("original template"))
		desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("new core content")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		if len(plan.Collisions) != 0 {
			t.Fatalf("Collisions = %+v, want none", plan.Collisions)
		}
		e := findMigrationEntry(t, plan, ".claude/rules/architecture.md")
		if e.Kind != OpDeleteOldPathAdoptFork {
			t.Errorf("Kind = %v, want OpDeleteOldPathAdoptFork", e.Kind)
		}
		// Fork attribution must survive an already-resolved relocation of
		// modified content: silently falling through to owner=core would
		// permanently lose it (self-review HIGH-1).
		if e.Owner != scaffold.OwnerFork {
			t.Errorf("Owner = %q, want fork", e.Owner)
		}
		if e.ForkedFromVersion != "0.9.0" {
			t.Errorf("ForkedFromVersion = %q, want %q", e.ForkedFromVersion, "0.9.0")
		}
	})

	t.Run("(b) unmodified relocate, dest matches new template -> delete-old only", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", "unmodified original")
		writeMigrationDiskFile(t, dir, ".claude/rules/ralph/architecture.md", "the new template content")

		m := scaffold.NewManifest("0.9.0")
		m.SetFile(".claude/rules/architecture.md", migrateHash("unmodified original"))
		desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("the new template content")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		if len(plan.Collisions) != 0 {
			t.Fatalf("Collisions = %+v, want none", plan.Collisions)
		}
		e := findMigrationEntry(t, plan, ".claude/rules/architecture.md")
		if e.Kind != OpDeleteOldPath {
			t.Errorf("Kind = %v, want OpDeleteOldPath", e.Kind)
		}
	})

	t.Run("(b) does not apply to a modified source even if dest matches template", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", "user edited")
		writeMigrationDiskFile(t, dir, ".claude/rules/ralph/architecture.md", "the new template content")

		m := scaffold.NewManifest("0.9.0")
		m.SetFile(".claude/rules/architecture.md", migrateHash("original template"))
		desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("the new template content")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		if len(plan.Collisions) != 1 {
			t.Fatalf("Collisions = %+v, want exactly one", plan.Collisions)
		}
		assertNoMigrationEntry(t, plan, ".claude/rules/architecture.md")
	})

	t.Run("(c) divergent destination -> collision, zero-write", func(t *testing.T) {
		dir := t.TempDir()
		writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", "user edited")
		writeMigrationDiskFile(t, dir, ".claude/rules/ralph/architecture.md", "totally unrelated divergent content")

		m := scaffold.NewManifest("0.9.0")
		m.SetFile(".claude/rules/architecture.md", migrateHash("original template"))
		desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("the new template content")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		if len(plan.Collisions) != 1 {
			t.Fatalf("Collisions = %+v, want exactly one", plan.Collisions)
		}
		c := plan.Collisions[0]
		if c.OldPath != ".claude/rules/architecture.md" || c.NewPath != ".claude/rules/ralph/architecture.md" {
			t.Errorf("collision = %+v, want old/new architecture.md paths", c)
		}
		assertNoMigrationEntry(t, plan, ".claude/rules/architecture.md")
	})
}

// --- Rerun stability ---

func TestClassifyMigration_RerunStability_AlreadyRelocatedOldPathAbsent(t *testing.T) {
	dir := t.TempDir()
	// Old path already gone (deleted by a prior interrupted migration
	// run); new path already exists (created by that same run).
	writeMigrationDiskFile(t, dir, ".claude/rules/ralph/architecture.md", "new core content")

	m := scaffold.NewManifest("0.9.0")
	m.SetFile(".claude/rules/architecture.md", migrateHash("original template"))
	desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("new core content")}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	// Nothing left to plan for this path: no entry, no collision.
	assertNoMigrationEntry(t, plan, ".claude/rules/architecture.md")
	if len(plan.Collisions) != 0 {
		t.Errorf("Collisions = %+v, want none", plan.Collisions)
	}
}

func TestClassifyMigration_RerunStability_PartiallyMigratedTreeCompletesRemainingWork(t *testing.T) {
	dir := t.TempDir()
	// architecture.md already relocated (old gone, new present).
	writeMigrationDiskFile(t, dir, ".claude/rules/ralph/architecture.md", "new core content")
	// testing.md still needs to move.
	writeMigrationDiskFile(t, dir, ".claude/rules/testing.md", "unmodified testing rule")

	m := scaffold.NewManifest("0.9.0")
	m.SetFile(".claude/rules/architecture.md", migrateHash("original architecture template"))
	m.SetFile(".claude/rules/testing.md", migrateHash("unmodified testing rule"))
	desired := map[string][]byte{
		".claude/rules/ralph/architecture.md": []byte("new core content"),
		".claude/rules/ralph/testing.md":      []byte("new testing content"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	if len(plan.Entries) != 1 {
		t.Fatalf("Entries = %+v, want exactly one (only the remaining work)", plan.Entries)
	}
	e := plan.Entries[0]
	if e.OldPath != ".claude/rules/testing.md" || e.Kind != OpDeleteOldPath {
		t.Errorf("entry = %+v, want testing.md DeleteOldPath", e)
	}
}

// --- Packs / nil manifest / preview rendering ---

func TestClassifyMigration_NilManifest(t *testing.T) {
	if _, err := ClassifyMigration(nil, t.TempDir(), map[string][]byte{}); err == nil {
		t.Fatal("expected error for nil manifest")
	}
}

func TestClassifyMigration_PacksSortedAndCarried(t *testing.T) {
	dir := t.TempDir()
	m := scaffold.NewManifest("0.9.0")
	m.Meta.Packs = []string{"typescript", "golang"}

	plan, err := ClassifyMigration(m, dir, map[string][]byte{})
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	if !reflect.DeepEqual(plan.Packs, []string{"golang", "typescript"}) {
		t.Errorf("Packs = %v, want sorted [golang typescript]", plan.Packs)
	}
}

func TestRenderMigrationPreview_CountsMatchAndDeterministic(t *testing.T) {
	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", "unmodified")
	writeMigrationDiskFile(t, dir, ".claude/rules/testing.md", "user edited")
	writeMigrationDiskFile(t, dir, "scripts/run-verify.sh", "unmodified")
	writeMigrationDiskFile(t, dir, "CLAUDE.md", "unmodified seed")

	m := scaffold.NewManifest("0.9.0")
	m.SetFile(".claude/rules/architecture.md", migrateHash("unmodified"))
	m.SetFile(".claude/rules/testing.md", migrateHash("original"))
	m.SetFile("scripts/run-verify.sh", migrateHash("unmodified"))
	m.SetFile("CLAUDE.md", migrateHash("unmodified seed"))

	desired := map[string][]byte{
		".claude/rules/ralph/architecture.md": []byte("new"),
		".claude/rules/ralph/testing.md":      []byte("new"),
		"scripts/run-verify.sh":               []byte("new"),
		"CLAUDE.md":                           []byte("new seed"),
	}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}

	out1 := RenderMigrationPreview(plan)
	out2 := RenderMigrationPreview(plan)
	if out1 != out2 {
		t.Errorf("RenderMigrationPreview is not deterministic:\n%s\n---\n%s", out1, out2)
	}

	counts := map[MigrationOpKind]int{}
	for _, e := range plan.Entries {
		counts[e.Kind]++
	}
	if counts[OpDeleteOldPath] != 1 {
		t.Errorf("DeleteOldPath count = %d, want 1", counts[OpDeleteOldPath])
	}
	if counts[OpForkRelocate] != 1 {
		t.Errorf("ForkRelocate count = %d, want 1", counts[OpForkRelocate])
	}
	if counts[OpKeepInPlace] != 1 {
		t.Errorf("KeepInPlace count = %d, want 1", counts[OpKeepInPlace])
	}
	if counts[OpReplaceWithTemplate] != 1 {
		t.Errorf("ReplaceWithTemplate count = %d, want 1", counts[OpReplaceWithTemplate])
	}
}

// ===================================================================
// Migration executor e2e tests (plan Scope C, slice 3).
//
// legacyFixtureNewTemplateFS builds the "new" v2 template set migration
// converts into, and buildLegacyProject builds a hand-crafted legacy (v1/v2,
// non-overlay) on-disk project + manifest — the LEGACY equivalent of
// upgrade_v2_test.go's v2FixtureGen/initV2Project pair, since executeInit
// only ever produces v2-layout manifests now and cannot be reused to
// generate a genuinely legacy fixture.
// ===================================================================

const (
	legacyOldClaudeMD          = "# OLD CLAUDE.md\n\nlegacy content.\n"
	legacyNewClaudeMD          = "# NEW CLAUDE.md seed\n\n@AGENTS.md\n"
	legacyOldAgentsMD          = "# AGENTS.md (legacy, pre-block-engine)\n\nHand-written project notes.\n"
	legacyOldGitignore         = "# Project ignores (legacy)\nnode_modules/\n"
	legacyOldArchitectureRule  = "# Architecture (legacy)\n\nunmodified rule body.\n"
	legacyNewArchitectureRule  = "# Architecture (new core)\n\nnew rule body.\n"
	legacyOldTestingRuleEdited = "# Testing (user edited)\n\nthe user's own testing guidance.\n"
	legacyNewTestingRule       = "# Testing (new core)\n\nnew rule body.\n"
	legacyOldGolangRule        = "---\npaths:\n  - \"**/*.go\"\n---\n# Go rules (legacy, unmodified)\n"
	legacyNewGolangRule        = "---\npaths:\n  - \"**/*.go\"\n---\n# Go rules (new)\n"
	legacyOldRunVerify         = "#!/bin/sh\necho legacy-v1\n"
	legacyNewRunVerify         = "#!/bin/sh\necho v2\n"
	legacyUserNotes            = "# User notes\n\nkept exactly as the user left them.\n"
	legacyOldCodexOverride     = "# Codex agent overrides (legacy, user customized)\n"
	legacyNewCodexOverride     = "# Codex agent overrides (new template)\n"
	legacyNewMissionManaged    = "## Mission\n\nNew mission text.\n"
	legacyNewGitignoreManaged  = ".ralph/local/\nnew-ignore/\n"
)

// legacyOldSettingsJSON ships two direct-invocation legacy hook commands
// (superseded by the dispatcher — see legacyRalphHookCommands) alongside a
// user-added permission and a top-level key ralph never owned, so
// TestRunMigrateLegacy_HappyPath_Yes can assert AC-13 (prune + preserve) end
// to end.
const legacyOldSettingsJSON = `{
  "customUserSetting": true,
  "permissions": {
    "allow": ["Bash(git status:*)", "Bash(user-added:*)"]
  },
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "./.claude/hooks/pre_bash_guard.sh"}]}
    ],
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "./.claude/hooks/session_start_context.sh"}]}
    ]
  }
}
`

const legacyNewSettingsJSON = `{
  "permissions": {
    "allow": ["Bash(git status:*)"]
  },
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "./.claude/hooks/ralph-dispatch.sh PreToolUse"}]}
    ]
  }
}
`

func legacyNewAgentsMD() []byte {
	return []byte("# AGENTS.md\n\nProject notes go here.\n\n" +
		upgrade.BeginMarker("agents-md") + "\n" +
		legacyNewMissionManaged +
		upgrade.EndMarker + "\n")
}

func legacyNewGitignore() []byte {
	return []byte("# Project ignores\n" +
		upgrade.BeginMarkerStyled("gitignore", upgrade.BlockMarkerHash) + "\n" +
		legacyNewGitignoreManaged +
		upgrade.EndMarkerStyled(upgrade.BlockMarkerHash) + "\n")
}

// legacyFixtureNewTemplateFS builds the embedded-templates fstest.MapFS for
// the "new" v2 layout a legacy project migrates into. docs/notes.md is
// deliberately NOT shipped, so the fixture's unmanaged docs/notes.md legacy
// entry exercises buildMigratedManifest's "fork with no template
// counterpart" path.
func legacyFixtureNewTemplateFS() fstest.MapFS {
	return fstest.MapFS{
		"templates/base/AGENTS.md":                           {Data: legacyNewAgentsMD()},
		"templates/base/.gitignore":                          {Data: legacyNewGitignore()},
		"templates/base/CLAUDE.md":                           {Data: []byte(legacyNewClaudeMD)},
		"templates/base/ralph.toml":                          {Data: []byte("[pipeline]\nmodel = \"test\"\n[doctor]\nrequire_codex_cli = false\n")},
		"templates/base/.claude/settings.json":               {Data: []byte(legacyNewSettingsJSON)},
		"templates/base/.ralph/core/settings.ralph.json":     {Data: []byte(legacyNewSettingsJSON)},
		"templates/base/.ralph/core/AGENTS.core.md":          {Data: []byte(legacyNewMissionManaged)},
		"templates/base/scripts/run-verify.sh":               {Data: []byte(legacyNewRunVerify)},
		"templates/base/.ralph/local/verify.d/.gitkeep":      {Data: []byte("")},
		"templates/base/.claude/rules/ralph/architecture.md": {Data: []byte(legacyNewArchitectureRule)},
		"templates/base/.claude/rules/ralph/testing.md":      {Data: []byte(legacyNewTestingRule)},
		"templates/base/.codex/AGENTS.override.md":           {Data: []byte(legacyNewCodexOverride)},
		"templates/packs/golang/verify.sh":                   {Data: []byte("#!/bin/sh\necho golang-new\n")},
		"templates/packs/golang/README.md":                   {Data: []byte("# Go new\n")},
		"templates/packs/golang/rule.md":                     {Data: []byte(legacyNewGolangRule)},
	}
}

// runMigrationTestGit runs git with the given args in dir, failing the test
// on error. isolateGitConfig (called by buildLegacyProject) keeps this from
// touching the real user/CI gitconfig.
func runMigrationTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// buildLegacyProject writes a hand-crafted legacy (v1/v2, non-overlay)
// project to a fresh temp dir and commits it (git init + commit), so the
// migration's git preflight passes by default. Every legacy manifest entry
// is deliberately shaped to exercise a distinct classification branch — see
// the inline comments below and TestRunMigrateLegacy_HappyPath_Yes, which
// asserts on all of them.
func buildLegacyProject(t *testing.T) string {
	t.Helper()
	isolateGitConfig(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	writeMigrationDiskFile(t, target, "CLAUDE.md", legacyOldClaudeMD)
	writeMigrationDiskFile(t, target, "AGENTS.md", legacyOldAgentsMD)
	writeMigrationDiskFile(t, target, ".gitignore", legacyOldGitignore)
	writeMigrationDiskFile(t, target, pathSettings, legacyOldSettingsJSON)
	writeMigrationDiskFile(t, target, ".claude/rules/architecture.md", legacyOldArchitectureRule)
	writeMigrationDiskFile(t, target, ".claude/rules/testing.md", legacyOldTestingRuleEdited)
	writeMigrationDiskFile(t, target, ".claude/rules/golang.md", legacyOldGolangRule)
	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", legacyOldRunVerify)
	writeMigrationDiskFile(t, target, "docs/notes.md", legacyUserNotes)
	writeMigrationDiskFile(t, target, pathCodexOverride, legacyOldCodexOverride)
	if err := os.MkdirAll(filepath.Join(target, ".ralph", "baseline"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph/baseline: %v", err)
	}
	writeMigrationDiskFile(t, target, ".ralph/baseline/AGENTS.md", "cached template")

	m := scaffold.NewManifest("0.9.0-test")
	m.Meta.Packs = []string{"golang"}
	m.SetFile("CLAUDE.md", migrateHash(legacyOldClaudeMD))                             // unmodified -> OpReplaceWithTemplate
	m.SetFile("AGENTS.md", migrateHash("stale-agents-hash"))                           // modified -> OpUntouched (block-appended by chained upgrade)
	m.SetFile(".gitignore", migrateHash(legacyOldGitignore))                           // unmodified -> OpReplaceWithTemplate
	m.SetFile(pathSettings, migrateHash("stale-settings-hash"))                        // modified -> OpSettingsPrune
	m.SetFile(".claude/rules/architecture.md", migrateHash(legacyOldArchitectureRule)) // unmodified -> relocate
	m.SetFile(".claude/rules/testing.md", migrateHash("stale-testing-hash"))           // modified -> fork-relocate
	m.SetFile(".claude/rules/golang.md", migrateHash(legacyOldGolangRule))             // unmodified pack rule -> relocate
	m.SetFile("scripts/run-verify.sh", migrateHash(legacyOldRunVerify))                // unmodified, same path -> OpKeepInPlace
	m.Files["docs/notes.md"] = scaffold.ManifestFile{
		Managed: false, State: scaffold.FileStateUnmanaged,
		Hash: migrateHash(legacyUserNotes), DiskHash: migrateHash(legacyUserNotes),
	} // legacy skip -> OpForkInPlace, no template counterpart
	m.SetFile(pathCodexOverride, migrateHash("stale-codex-hash")) // always OpUntouched, owner=seed

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}

	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "legacy project")

	return target
}

func TestRunMigrateLegacy_HappyPath_Yes(t *testing.T) {
	target := buildLegacyProject(t)
	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions (migration): %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	// -- AC-4: unmodified relocated rule (old deleted, new created by the
	// chained v2 upgrade) --
	if _, statErr := os.Stat(filepath.Join(target, ".claude", "rules", "architecture.md")); !os.IsNotExist(statErr) {
		t.Errorf(".claude/rules/architecture.md must be deleted; stat err = %v", statErr)
	}
	gotArch := mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", "architecture.md"))
	if string(gotArch) != legacyNewArchitectureRule {
		t.Errorf(".claude/rules/ralph/architecture.md = %q, want new core content %q", gotArch, legacyNewArchitectureRule)
	}

	// -- AC-4/AC-8: modified relocated rule forked to the new path,
	// byte-preserved --
	if _, statErr := os.Stat(filepath.Join(target, ".claude", "rules", "testing.md")); !os.IsNotExist(statErr) {
		t.Errorf(".claude/rules/testing.md must be deleted; stat err = %v", statErr)
	}
	gotTesting := mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", "testing.md"))
	if string(gotTesting) != legacyOldTestingRuleEdited {
		t.Errorf(".claude/rules/ralph/testing.md = %q, want preserved user content %q", gotTesting, legacyOldTestingRuleEdited)
	}

	// -- AC-11: pack rule relocated, Meta.Packs carried over --
	gotGolang := mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", "golang.md"))
	if string(gotGolang) != legacyNewGolangRule {
		t.Errorf(".claude/rules/ralph/golang.md = %q, want new pack rule content %q", gotGolang, legacyNewGolangRule)
	}

	// -- AC-4/AC-7: unmodified, same-path core file converges via the
	// chained v2 upgrade --
	gotRunVerify := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotRunVerify) != legacyNewRunVerify {
		t.Errorf("scripts/run-verify.sh = %q, want new core content %q", gotRunVerify, legacyNewRunVerify)
	}

	// -- AC-4: unmanaged (legacy skip) entry forked in place, byte-preserved,
	// no template counterpart --
	gotNotes := mustReadFile(t, filepath.Join(target, "docs", "notes.md"))
	if string(gotNotes) != legacyUserNotes {
		t.Errorf("docs/notes.md = %q, want preserved user content %q", gotNotes, legacyUserNotes)
	}

	// -- AC-5: unmodified CLAUDE.md replaced with the new seed --
	gotClaude := mustReadFile(t, filepath.Join(target, "CLAUDE.md"))
	if string(gotClaude) != legacyNewClaudeMD {
		t.Errorf("CLAUDE.md = %q, want new seed %q", gotClaude, legacyNewClaudeMD)
	}

	// -- AC-5: unmodified .gitignore replaced with the block-carrying
	// template, and the chained upgrade's block engine leaves it unchanged
	// (already exactly matches) --
	gotGitignore := mustReadFile(t, filepath.Join(target, ".gitignore"))
	if string(gotGitignore) != string(legacyNewGitignore()) {
		t.Errorf(".gitignore = %q, want new block template %q", gotGitignore, legacyNewGitignore())
	}

	// -- AC-5: modified AGENTS.md left in place, block appended by the
	// chained upgrade, original content preserved outside the block --
	gotAgents := string(mustReadFile(t, filepath.Join(target, "AGENTS.md")))
	if !strings.Contains(gotAgents, legacyOldAgentsMD) {
		t.Errorf("AGENTS.md must preserve the original legacy content outside the block:\n%s", gotAgents)
	}
	if !strings.Contains(gotAgents, legacyNewMissionManaged) {
		t.Errorf("AGENTS.md must have the new managed block appended:\n%s", gotAgents)
	}

	// -- AC-6: codex override always byte-untouched, regardless of modified
	// state --
	gotCodex := mustReadFile(t, filepath.Join(target, ".codex", "AGENTS.override.md"))
	if string(gotCodex) != legacyOldCodexOverride {
		t.Errorf(".codex/AGENTS.override.md = %q, want byte-untouched legacy content %q", gotCodex, legacyOldCodexOverride)
	}

	// -- AC-13: settings prune removes the legacy direct-invocation hook,
	// the chained 3-way merge adds the dispatcher entry, user permission and
	// unrelated top-level key are preserved --
	gotSettings := string(mustReadFile(t, filepath.Join(target, ".claude", "settings.json")))
	if strings.Contains(gotSettings, "pre_bash_guard.sh") {
		t.Errorf("settings.json must not retain the pruned legacy hook command:\n%s", gotSettings)
	}
	if !strings.Contains(gotSettings, "ralph-dispatch.sh") {
		t.Errorf("settings.json must gain the new dispatcher hook command:\n%s", gotSettings)
	}
	if !strings.Contains(gotSettings, "user-added") {
		t.Errorf("settings.json must preserve the user-added permission entry:\n%s", gotSettings)
	}
	if !strings.Contains(gotSettings, "customUserSetting") {
		t.Errorf("settings.json must preserve the unrelated top-level user key:\n%s", gotSettings)
	}

	// -- .ralph/baseline/ removed --
	if _, statErr := os.Stat(filepath.Join(target, ".ralph", "baseline")); !os.IsNotExist(statErr) {
		t.Errorf(".ralph/baseline must be removed; stat err = %v", statErr)
	}

	// -- AC-7: v3 manifest is layout=v2, Packs carried over, fork entries
	// recorded --
	m := readManifestV2(t, target)
	if m.Meta.Layout != scaffold.LayoutV2 {
		t.Errorf("Meta.Layout = %q, want %q", m.Meta.Layout, scaffold.LayoutV2)
	}
	if !reflect.DeepEqual(m.Meta.Packs, []string{"golang"}) {
		t.Errorf("Meta.Packs = %v, want [golang]", m.Meta.Packs)
	}
	forkEntry, ok := m.Files[".claude/rules/ralph/testing.md"]
	if !ok || forkEntry.Owner != scaffold.OwnerFork || forkEntry.ForkedFromVersion != "0.9.0-test" {
		t.Errorf("fork entry for .claude/rules/ralph/testing.md = %+v, want Owner=fork ForkedFromVersion=0.9.0-test", forkEntry)
	}
	notesEntry, ok := m.Files["docs/notes.md"]
	if !ok || notesEntry.Owner != scaffold.OwnerFork {
		t.Errorf("fork entry for docs/notes.md = %+v, want Owner=fork", notesEntry)
	}
	codexEntry, ok := m.Files[pathCodexOverride]
	if !ok || codexEntry.Owner != scaffold.OwnerSeed {
		t.Errorf("owner for %s = %+v, want Owner=seed", pathCodexOverride, codexEntry)
	}

	// -- AC-8: migration report written with a fork diff and the prune
	// listing --
	reportPaths, globErr := filepath.Glob(filepath.Join(target, "docs", "reports", "ralph-migration-*.md"))
	if globErr != nil || len(reportPaths) != 1 {
		t.Fatalf("expected exactly one ralph-migration-*.md report, got %v (err=%v)", reportPaths, globErr)
	}
	report := string(mustReadFile(t, reportPaths[0]))
	if !strings.Contains(report, ".claude/rules/ralph/testing.md") {
		t.Errorf("report must list the forked testing.md path:\n%s", report)
	}
	if !strings.Contains(report, "pre_bash_guard.sh") {
		t.Errorf("report must list the pruned legacy hook command:\n%s", report)
	}

	// -- AC-7: immediate re-upgrade is a clean run through the v2 path --
	var out2, errOut2 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false); err != nil {
		t.Fatalf("re-running ralph upgrade after migration must succeed: %v\nstderr:\n%s", err, errOut2.String())
	}
}

func TestRunMigrateLegacy_ConfirmUX(t *testing.T) {
	t.Run("no_aborts_zero_writes", func(t *testing.T) {
		target := buildLegacyProject(t)
		scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
		Version = "2.0.0-test"

		before := snapshotDirHashes(t, target)
		var out, errOut bytes.Buffer
		err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader("n\n"), &out, &errOut, false)
		if err != nil {
			t.Fatalf("declining the migration prompt must not error: %v", err)
		}
		after := snapshotDirHashes(t, target)
		if !slicesEqualStrings(before, after) {
			t.Errorf("declining the migration prompt must write zero files; before=%v after=%v", before, after)
		}
	})

	t.Run("eof_aborts_zero_writes", func(t *testing.T) {
		target := buildLegacyProject(t)
		scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
		Version = "2.0.0-test"

		before := snapshotDirHashes(t, target)
		var out, errOut bytes.Buffer
		err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
		if err != nil {
			t.Fatalf("a non-interactive EOF must abort safely, not error: %v", err)
		}
		after := snapshotDirHashes(t, target)
		if !slicesEqualStrings(before, after) {
			t.Errorf("an EOF confirmation must write zero files; before=%v after=%v", before, after)
		}
	})

	t.Run("dry_run_zero_writes", func(t *testing.T) {
		target := buildLegacyProject(t)
		scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
		Version = "2.0.0-test"

		before := snapshotDirHashes(t, target)
		var out, errOut bytes.Buffer
		err := runUpgradeIOWithOptions(target, upgradeOptions{DryRun: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
		if err != nil {
			t.Fatalf("--dry-run must succeed: %v", err)
		}
		if !strings.Contains(out.String(), "Legacy -> v2 migration preview") {
			t.Errorf("--dry-run must render the migration preview:\n%s", out.String())
		}
		after := snapshotDirHashes(t, target)
		if !slicesEqualStrings(before, after) {
			t.Errorf("--dry-run must write zero files; before=%v after=%v", before, after)
		}
	})
}

func TestRunMigrateLegacy_DirtyGitTree_ZeroWrites(t *testing.T) {
	target := buildLegacyProject(t)
	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	// Dirty the tree after the fixture's own commit.
	writeMigrationDiskFile(t, target, "CLAUDE.md", "uncommitted local edit")

	before := snapshotDirHashes(t, target)
	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("migration on a dirty git work tree: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "clean git work tree") {
		t.Errorf("err = %v, want a clean-git-work-tree preflight error", err)
	}
	after := snapshotDirHashes(t, target)
	if !slicesEqualStrings(before, after) {
		t.Errorf("a dirty-tree refusal must write zero further files; before=%v after=%v", before, after)
	}
}

func TestRunMigrateLegacy_Collision_ZeroWrites(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}
	writeMigrationDiskFile(t, target, ".claude/rules/testing.md", legacyOldTestingRuleEdited)
	// Pre-existing, divergent destination content: collision-matrix case (c).
	writeMigrationDiskFile(t, target, ".claude/rules/ralph/testing.md", "totally unrelated divergent content")

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/testing.md", migrateHash("stale-hash"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "collision fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	before := snapshotDirHashes(t, target)
	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("migration with a relocation collision: expected an error, got nil")
	}
	if !strings.Contains(out.String(), "Collisions (1)") {
		t.Errorf("preview must list the collision:\n%s", out.String())
	}
	after := snapshotDirHashes(t, target)
	if !slicesEqualStrings(before, after) {
		t.Errorf("a collision refusal must write zero files; before=%v after=%v", before, after)
	}
}

func TestRunMigrateLegacy_SymlinkedRelocationDestParent_ZeroWrites(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}
	writeMigrationDiskFile(t, target, ".claude/rules/testing.md", legacyOldTestingRuleEdited)

	// .claude/rules/ralph is a symlink pointing outside target: the
	// relocation destination's parent chain must be refused before any
	// write (AC-16).
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(target, ".claude", "rules", "ralph")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/testing.md", migrateHash("stale-hash"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "symlink fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	// snapshotDirHashes cannot walk through the symlinked directory itself
	// (os.ReadFile on a symlinked directory errors "is a directory"), so
	// this test asserts zero-writes directly instead: the legacy manifest
	// bytes are unchanged, the symlink itself is untouched, and nothing was
	// written into the external directory it points at.
	manifestBefore := mustReadFile(t, manifestPath)

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("migration through a symlinked relocation-destination parent: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "preflight") {
		t.Errorf("err = %v, want a preflight-check error naming the unsafe path", err)
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Errorf("a preflight refusal must not touch the legacy manifest:\nbefore: %s\nafter:  %s", manifestBefore, manifestAfter)
	}
	info, lerr := os.Lstat(filepath.Join(target, ".claude", "rules", "ralph"))
	if lerr != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf(".claude/rules/ralph must remain the untouched symlink: info=%+v err=%v", info, lerr)
	}
	externalEntries, rerr := os.ReadDir(external)
	if rerr != nil {
		t.Fatalf("reading external dir: %v", rerr)
	}
	if len(externalEntries) != 0 {
		t.Errorf("a preflight refusal must not write through the symlink into the external dir; entries=%v", externalEntries)
	}
}

// TestRunMigrateLegacy_PartialFailure_ManifestNotAdvanced_ResumeCompletes is
// AC-14 coverage: a failure injected partway through execution (CLAUDE.md
// made read-only, so its OpReplaceWithTemplate write fails) must leave the
// legacy manifest unadvanced and every entry sorting after CLAUDE.md
// unattempted. Restoring write access, committing the partially-applied
// tree (satisfying the migration's own git-clean precondition again — see
// runMigrateLegacy's partial-failure error message), and re-running must
// then complete via ClassifyMigration's rerun-stability classification.
func TestRunMigrateLegacy_PartialFailure_ManifestNotAdvanced_ResumeCompletes(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0o444 does not deny writes to root; this failure-injection technique cannot run as root")
	}

	target := buildLegacyProject(t)
	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	manifestBefore := mustReadFile(t, manifestPath)

	claudePath := filepath.Join(target, "CLAUDE.md")
	if err := os.Chmod(claudePath, 0o444); err != nil {
		t.Fatalf("chmod CLAUDE.md read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(claudePath, 0o644) })

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("expected the injected CLAUDE.md write failure to surface as an error")
	}

	manifestAfterFailure := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfterFailure) {
		t.Errorf("manifest must not advance on a partial migration failure:\nbefore: %s\nafter:  %s", manifestBefore, manifestAfterFailure)
	}

	// CLAUDE.md sorts last among write-performing entries in this fixture,
	// so everything else should already have landed.
	if _, statErr := os.Stat(filepath.Join(target, ".claude", "rules", "architecture.md")); !os.IsNotExist(statErr) {
		t.Errorf(".claude/rules/architecture.md should already be relocated before the CLAUDE.md failure; stat err = %v", statErr)
	}
	gotClaudeStillLegacy := mustReadFile(t, claudePath)
	if string(gotClaudeStillLegacy) != legacyOldClaudeMD {
		t.Errorf("CLAUDE.md must still hold its pre-migration content after the injected failure: got %q", gotClaudeStillLegacy)
	}

	// Restore write access and commit the partially-applied tree (the
	// migration's git-clean precondition), then re-run.
	if err := os.Chmod(claudePath, 0o644); err != nil {
		t.Fatalf("chmod CLAUDE.md writable: %v", err)
	}
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "partial migration checkpoint")

	var out2, errOut2 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false); err != nil {
		t.Fatalf("retry after restoring permissions must succeed: %v\nstderr:\n%s", err, errOut2.String())
	}

	gotClaudeAfterRetry := mustReadFile(t, claudePath)
	if string(gotClaudeAfterRetry) != legacyNewClaudeMD {
		t.Errorf("CLAUDE.md after retry = %q, want new seed %q", gotClaudeAfterRetry, legacyNewClaudeMD)
	}
	m := readManifestV2(t, target)
	if m.Meta.Layout != scaffold.LayoutV2 {
		t.Errorf("Meta.Layout after retry = %q, want %q", m.Meta.Layout, scaffold.LayoutV2)
	}
}

// ===================================================================
// Slice 4: remaining edges (docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md,
// Implementation outline step 4). Slice 3 already covered the happy path,
// confirm UX, dirty git, collision case (c), a symlinked destination
// parent, and partial-failure/rerun. This section covers what is left:
// an all-modified fixture, unmanaged carry-over, legacy manifest oddities
// (empty hash, state=partial, missing-from-disk), collision cases (a)/(b)
// end to end, --yes never touching stdin, and a post-migration invariants
// sweep including no-churn stability across repeated `ralph upgrade` runs.
// ===================================================================

// legacyAllModifiedStaleHash is an intentionally mismatched recorded hash.
// Every SetFile call in buildAllModifiedLegacyProject uses it instead of the
// file's real content hash, so every managed entry classifies as
// LegacyModified (see classifyLegacyEntryState) regardless of what content
// it actually holds — letting buildAllModifiedLegacyProject reuse
// buildLegacyProject's own disk fixtures verbatim for the "every tracked
// file is user-modified" edge, which slice 3's mixed fixture did not
// exercise on its own.
const legacyAllModifiedStaleHash = "sha256:all-modified-stale-marker"

// buildAllModifiedLegacyProject mirrors buildLegacyProject (same disk
// content, same paths, same installed pack), but every managed manifest
// entry is recorded with legacyAllModifiedStaleHash instead of its real
// content hash, so every single one classifies as LegacyModified: relocated
// rules and the same-path core file all become forks instead of
// deletes/keeps, and the special faces all take their "modified" branch
// instead of their "unmodified" one.
func buildAllModifiedLegacyProject(t *testing.T) string {
	t.Helper()
	isolateGitConfig(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	writeMigrationDiskFile(t, target, "CLAUDE.md", legacyOldClaudeMD)
	writeMigrationDiskFile(t, target, "AGENTS.md", legacyOldAgentsMD)
	writeMigrationDiskFile(t, target, ".gitignore", legacyOldGitignore)
	writeMigrationDiskFile(t, target, pathSettings, legacyOldSettingsJSON)
	writeMigrationDiskFile(t, target, ".claude/rules/architecture.md", legacyOldArchitectureRule)
	writeMigrationDiskFile(t, target, ".claude/rules/testing.md", legacyOldTestingRuleEdited)
	writeMigrationDiskFile(t, target, ".claude/rules/golang.md", legacyOldGolangRule)
	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", legacyOldRunVerify)
	writeMigrationDiskFile(t, target, "docs/notes.md", legacyUserNotes)
	writeMigrationDiskFile(t, target, pathCodexOverride, legacyOldCodexOverride)
	if err := os.MkdirAll(filepath.Join(target, ".ralph", "baseline"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph/baseline: %v", err)
	}
	writeMigrationDiskFile(t, target, ".ralph/baseline/AGENTS.md", "cached template")

	m := scaffold.NewManifest("0.9.0-test")
	m.Meta.Packs = []string{"golang"}
	m.SetFile("CLAUDE.md", legacyAllModifiedStaleHash)
	m.SetFile("AGENTS.md", legacyAllModifiedStaleHash)
	m.SetFile(".gitignore", legacyAllModifiedStaleHash)
	m.SetFile(pathSettings, legacyAllModifiedStaleHash)
	m.SetFile(".claude/rules/architecture.md", legacyAllModifiedStaleHash)
	m.SetFile(".claude/rules/testing.md", legacyAllModifiedStaleHash)
	m.SetFile(".claude/rules/golang.md", legacyAllModifiedStaleHash)
	m.SetFile("scripts/run-verify.sh", legacyAllModifiedStaleHash)
	m.Files["docs/notes.md"] = scaffold.ManifestFile{
		Managed: false, State: scaffold.FileStateUnmanaged,
		Hash: migrateHash(legacyUserNotes), DiskHash: migrateHash(legacyUserNotes),
	} // legacy skip -> always a fork, regardless of hash
	m.SetFile(pathCodexOverride, legacyAllModifiedStaleHash)

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}

	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "all-modified legacy project")

	return target
}

// TestRunMigrateLegacy_AllModifiedFixture_EveryTrackedFileForked is slice 4
// work item 1 (every tracked file is user-modified) plus work item 6 (the
// post-migration invariants sweep, run against this same fixture).
func TestRunMigrateLegacy_AllModifiedFixture_EveryTrackedFileForked(t *testing.T) {
	target := buildAllModifiedLegacyProject(t)
	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions (all-modified migration): %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	// -- every relocatable rule fork-relocated, byte-preserved, old path gone --
	for _, tc := range []struct{ name, want string }{
		{"architecture.md", legacyOldArchitectureRule},
		{"testing.md", legacyOldTestingRuleEdited},
		{"golang.md", legacyOldGolangRule},
	} {
		oldPath := filepath.Join(target, ".claude", "rules", tc.name)
		if _, statErr := os.Stat(oldPath); !os.IsNotExist(statErr) {
			t.Errorf(".claude/rules/%s must be deleted after fork-relocation; stat err = %v", tc.name, statErr)
		}
		got := mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", tc.name))
		if string(got) != tc.want {
			t.Errorf(".claude/rules/ralph/%s = %q, want preserved user content %q", tc.name, got, tc.want)
		}
	}

	// -- same-path core file forked in place, byte-preserved --
	gotRunVerify := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotRunVerify) != legacyOldRunVerify {
		t.Errorf("scripts/run-verify.sh = %q, want preserved user content %q (fork in place)", gotRunVerify, legacyOldRunVerify)
	}

	// -- special face: modified CLAUDE.md byte-untouched (FR-8) --
	gotClaude := mustReadFile(t, filepath.Join(target, "CLAUDE.md"))
	if string(gotClaude) != legacyOldClaudeMD {
		t.Errorf("CLAUDE.md = %q, want byte-untouched legacy content %q", gotClaude, legacyOldClaudeMD)
	}

	// -- special faces: modified AGENTS.md/.gitignore left in place, chained
	// upgrade's block engine appends its managed block, original content
	// preserved outside the block --
	gotAgents := string(mustReadFile(t, filepath.Join(target, "AGENTS.md")))
	if !strings.Contains(gotAgents, legacyOldAgentsMD) {
		t.Errorf("AGENTS.md must preserve original legacy content outside the block:\n%s", gotAgents)
	}
	if !strings.Contains(gotAgents, legacyNewMissionManaged) {
		t.Errorf("AGENTS.md must have the new managed block appended:\n%s", gotAgents)
	}
	gotGitignore := string(mustReadFile(t, filepath.Join(target, ".gitignore")))
	if !strings.Contains(gotGitignore, legacyOldGitignore) {
		t.Errorf(".gitignore must preserve original legacy content outside the block:\n%s", gotGitignore)
	}
	if !strings.Contains(gotGitignore, legacyNewGitignoreManaged) {
		t.Errorf(".gitignore must have the new managed block appended:\n%s", gotGitignore)
	}

	// -- special face: modified settings.json pruned of known legacy hooks,
	// chained 3-way merge adds the dispatcher entry, user content preserved --
	gotSettings := string(mustReadFile(t, filepath.Join(target, ".claude", "settings.json")))
	if strings.Contains(gotSettings, "pre_bash_guard.sh") || strings.Contains(gotSettings, "session_start_context.sh") {
		t.Errorf("settings.json must not retain any pruned legacy hook command:\n%s", gotSettings)
	}
	if !strings.Contains(gotSettings, "ralph-dispatch.sh") {
		t.Errorf("settings.json must gain the new dispatcher hook command:\n%s", gotSettings)
	}
	if !strings.Contains(gotSettings, "user-added") || !strings.Contains(gotSettings, "customUserSetting") {
		t.Errorf("settings.json must preserve user-added content:\n%s", gotSettings)
	}

	// -- special face: codex override always byte-untouched, owner=seed --
	gotCodex := mustReadFile(t, filepath.Join(target, ".codex", "AGENTS.override.md"))
	if string(gotCodex) != legacyOldCodexOverride {
		t.Errorf(".codex/AGENTS.override.md = %q, want byte-untouched legacy content %q", gotCodex, legacyOldCodexOverride)
	}

	// -- report lists every fork with diffs --
	reportPaths, globErr := filepath.Glob(filepath.Join(target, "docs", "reports", "ralph-migration-*.md"))
	if globErr != nil || len(reportPaths) != 1 {
		t.Fatalf("expected exactly one ralph-migration-*.md report, got %v (err=%v)", reportPaths, globErr)
	}
	report := string(mustReadFile(t, reportPaths[0]))
	for _, want := range []string{
		".claude/rules/ralph/architecture.md",
		".claude/rules/ralph/testing.md",
		".claude/rules/ralph/golang.md",
		"scripts/run-verify.sh",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report must list the forked path %q:\n%s", want, report)
		}
	}

	// -- work item 6: post-migration invariants sweep --

	if _, statErr := os.Stat(filepath.Join(target, ".ralph", "baseline")); !os.IsNotExist(statErr) {
		t.Errorf(".ralph/baseline must be absent after migration; stat err = %v", statErr)
	}

	m := readManifestV2(t, target)
	if m.Meta.Layout != scaffold.LayoutV2 {
		t.Errorf("Meta.Layout = %q, want %q", m.Meta.Layout, scaffold.LayoutV2)
	}
	if !reflect.DeepEqual(m.Meta.Packs, []string{"golang"}) {
		t.Errorf("Meta.Packs = %v, want [golang]", m.Meta.Packs)
	}
	if len(m.Files) == 0 {
		t.Fatal("manifest has no files after migration")
	}
	for path, entry := range m.Files {
		if entry.Owner == "" {
			t.Errorf("manifest entry %q has no Owner attribute; every v3 entry must be owned after migration", path)
		}
	}

	for _, name := range []string{"architecture.md", "testing.md", "golang.md"} {
		legacyPath := filepath.Join(target, ".claude", "rules", name)
		if _, statErr := os.Stat(legacyPath); !os.IsNotExist(statErr) {
			t.Errorf("legacy shipped-rule path %q must not remain on disk after migration; stat err = %v", legacyPath, statErr)
		}
	}

	// Second and third `ralph upgrade` runs (fresh calls, not the chained
	// call inside the migration itself) must be exit-behavior-stable and
	// write nothing: the tree already converged during the migration's own
	// chained v2 upgrade.
	before2 := snapshotTreeHashesExcluding(t, target)
	var out2, errOut2 bytes.Buffer
	err2 := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false)
	after2 := snapshotTreeHashesExcluding(t, target)
	if !slicesEqualStrings(before2, after2) {
		t.Errorf("second post-migration `ralph upgrade` must write nothing (already converged); before=%v after=%v", before2, after2)
	}

	before3 := snapshotTreeHashesExcluding(t, target)
	var out3, errOut3 bytes.Buffer
	err3 := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out3, &errOut3, false)
	after3 := snapshotTreeHashesExcluding(t, target)
	if !slicesEqualStrings(before3, after3) {
		t.Errorf("third post-migration `ralph upgrade` must write nothing (already converged); before=%v after=%v", before3, after3)
	}
	if !slicesEqualStrings(after2, after3) {
		t.Errorf("repeated post-migration `ralph upgrade` runs must leave an identical tree; run2=%v run3=%v", after2, after3)
	}
	if (err2 == nil) != (err3 == nil) {
		t.Errorf("repeated post-migration `ralph upgrade` runs must be exit-behavior-stable: err2=%v err3=%v", err2, err3)
	}
	if err2 != nil && !errors.Is(err2, ErrUpgradeDriftRemaining) {
		t.Errorf("if a post-migration `ralph upgrade` run errors, it must only be the drift sentinel: err2=%v", err2)
	}
	if err3 != nil && !errors.Is(err3, ErrUpgradeDriftRemaining) {
		t.Errorf("if a post-migration `ralph upgrade` run errors, it must only be the drift sentinel: err3=%v", err3)
	}
}

// TestRunMigrateLegacy_UnmanagedCarryOver_BothRelocatableAndSamePath is slice
// 4 work item 2: a legacy manifest whose only entries are managed=false
// ("skip") records, one at a relocatable seed path and one at a same-path
// (non-relocatable) location. Both must become forks, and both files must
// be byte-preserved, untouched by the migration itself.
func TestRunMigrateLegacy_UnmanagedCarryOver_BothRelocatableAndSamePath(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	const unmanagedRelocatableContent = "# custom architecture note (never ralph's, legacy skip)\n\nuser wrote this.\n"
	const unmanagedSamePathContent = "totally unmanaged notes, never templated, legacy skip.\n"

	// .claude/rules/architecture.md is a relocatable path (its basename
	// matches a rule the new template ships under .claude/rules/ralph/);
	// docs/legacy-skip.md is not under .claude/rules/ at all, so it can
	// never relocate regardless of desired's contents.
	writeMigrationDiskFile(t, target, ".claude/rules/architecture.md", unmanagedRelocatableContent)
	writeMigrationDiskFile(t, target, "docs/legacy-skip.md", unmanagedSamePathContent)

	m := scaffold.NewManifest("0.9.0-test")
	m.Files[".claude/rules/architecture.md"] = scaffold.ManifestFile{
		Managed: false, State: scaffold.FileStateUnmanaged,
		Hash: migrateHash(unmanagedRelocatableContent), DiskHash: migrateHash(unmanagedRelocatableContent),
	}
	m.Files["docs/legacy-skip.md"] = scaffold.ManifestFile{
		Managed: false, State: scaffold.FileStateUnmanaged,
		Hash: migrateHash(unmanagedSamePathContent), DiskHash: migrateHash(unmanagedSamePathContent),
	}
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "unmanaged carry-over fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions (unmanaged carry-over): %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	// -- relocatable unmanaged entry: relocated to the new path,
	// byte-preserved, old path gone --
	if _, statErr := os.Stat(filepath.Join(target, ".claude", "rules", "architecture.md")); !os.IsNotExist(statErr) {
		t.Errorf(".claude/rules/architecture.md must be deleted after fork-relocation; stat err = %v", statErr)
	}
	gotArch := mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", "architecture.md"))
	if string(gotArch) != unmanagedRelocatableContent {
		t.Errorf(".claude/rules/ralph/architecture.md = %q, want preserved unmanaged content %q", gotArch, unmanagedRelocatableContent)
	}

	// -- same-path unmanaged entry: forked in place, byte-preserved --
	gotSkip := mustReadFile(t, filepath.Join(target, "docs", "legacy-skip.md"))
	if string(gotSkip) != unmanagedSamePathContent {
		t.Errorf("docs/legacy-skip.md = %q, want preserved unmanaged content %q", gotSkip, unmanagedSamePathContent)
	}

	m2 := readManifestV2(t, target)
	archEntry, ok := m2.Files[".claude/rules/ralph/architecture.md"]
	if !ok || archEntry.Owner != scaffold.OwnerFork {
		t.Errorf("manifest entry for .claude/rules/ralph/architecture.md = %+v, want Owner=fork", archEntry)
	}
	skipEntry, ok := m2.Files["docs/legacy-skip.md"]
	if !ok || skipEntry.Owner != scaffold.OwnerFork {
		t.Errorf("manifest entry for docs/legacy-skip.md = %+v, want Owner=fork", skipEntry)
	}
}

// --- Slice 4 work item 3: legacy manifest oddities (empty hash "heal"
// targets, state=partial, and entries whose file is missing from disk).
// These exercise ClassifyMigration end to end (not just
// classifyLegacyEntryState in isolation), confirming the resulting
// MigrationEntry.Kind, not only the intermediate LegacyEntryState.

func TestClassifyMigration_EmptyHashHealTarget_RelocatablePath_ForksNeverDeleted(t *testing.T) {
	dir := t.TempDir()
	content := "content the v1 heal mechanism never recorded a hash for"
	writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", content)

	m := scaffold.NewManifest("0.9.0")
	m.Files[".claude/rules/architecture.md"] = scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged}
	desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("new core content")}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	e := findMigrationEntry(t, plan, ".claude/rules/architecture.md")
	if e.Kind != OpForkRelocate {
		t.Errorf("Kind = %v, want OpForkRelocate (an empty recorded hash must never be treated as unmodified, so it is never deleted)", e.Kind)
	}
	if e.State != LegacyModified {
		t.Errorf("State = %v, want LegacyModified", e.State)
	}
}

func TestClassifyMigration_EmptyHashHealTarget_SamePath_ForksNeverDeleted(t *testing.T) {
	dir := t.TempDir()
	content := "unhashed content, same path"
	writeMigrationDiskFile(t, dir, "scripts/run-verify.sh", content)

	m := scaffold.NewManifest("0.9.0")
	m.Files["scripts/run-verify.sh"] = scaffold.ManifestFile{Managed: true, State: scaffold.FileStateManaged}
	desired := map[string][]byte{"scripts/run-verify.sh": []byte("new template content")}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	e := findMigrationEntry(t, plan, "scripts/run-verify.sh")
	if e.Kind != OpForkInPlace {
		t.Errorf("Kind = %v, want OpForkInPlace (an empty recorded hash must never be treated as unmodified, so it is never deleted)", e.Kind)
	}
}

func TestClassifyMigration_PartialState_RelocatablePath_Forks(t *testing.T) {
	dir := t.TempDir()
	content := "content the user chose to keep during a v1-era interactive conflict resolution"
	writeMigrationDiskFile(t, dir, ".claude/rules/architecture.md", content)

	m := scaffold.NewManifest("0.9.0")
	m.Files[".claude/rules/architecture.md"] = scaffold.ManifestFile{
		Managed: true, State: legacyStatePartial,
		// Hash/DiskHash deliberately match disk: state=partial must force
		// LegacyModified regardless, per the LegacyEntryState contract.
		Hash: migrateHash(content), DiskHash: migrateHash(content),
	}
	desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("new core content")}

	plan, err := ClassifyMigration(m, dir, desired)
	if err != nil {
		t.Fatalf("ClassifyMigration: %v", err)
	}
	e := findMigrationEntry(t, plan, ".claude/rules/architecture.md")
	if e.Kind != OpForkRelocate {
		t.Errorf("Kind = %v, want OpForkRelocate (state=partial must always fork, even with a matching hash)", e.Kind)
	}
}

// TestClassifyMigration_MissingFromDisk_NoPhantomOps covers a manifest entry
// whose file is missing from disk at both a relocatable and a
// non-relocatable path. These hit two different short-circuits:
// ClassifyMigration's own rerun-stability check (relocatable — the path is
// treated as already migrated, no entry at all) and
// classifyForkCandidate's own !hasDisk branch (non-relocatable — an
// OpUntouched no-op entry). Both must produce no error and no collision.
func TestClassifyMigration_MissingFromDisk_NoPhantomOps(t *testing.T) {
	t.Run("relocatable path missing from disk: rerun-stability short-circuit, no entry at all", func(t *testing.T) {
		dir := t.TempDir()
		m := scaffold.NewManifest("0.9.0")
		m.SetFile(".claude/rules/architecture.md", migrateHash("content that was never written to this fixture's disk"))
		desired := map[string][]byte{".claude/rules/ralph/architecture.md": []byte("new core content")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		assertNoMigrationEntry(t, plan, ".claude/rules/architecture.md")
		if len(plan.Collisions) != 0 {
			t.Errorf("Collisions = %+v, want none", plan.Collisions)
		}
	})

	t.Run("non-relocatable path missing from disk: classifyForkCandidate short-circuit, untouched no-op entry", func(t *testing.T) {
		dir := t.TempDir()
		m := scaffold.NewManifest("0.9.0")
		m.SetFile("docs/gone.md", migrateHash("content that was never written to this fixture's disk"))
		desired := map[string][]byte{"docs/gone.md": []byte("new template content")}

		plan, err := ClassifyMigration(m, dir, desired)
		if err != nil {
			t.Fatalf("ClassifyMigration: %v", err)
		}
		e := findMigrationEntry(t, plan, "docs/gone.md")
		if e.Kind != OpUntouched {
			t.Errorf("Kind = %v, want OpUntouched", e.Kind)
		}
		if e.NewPath != e.OldPath {
			t.Errorf("NewPath = %q, want it left at OldPath %q (no relocation is ever attempted for a missing file)", e.NewPath, e.OldPath)
		}
		if len(plan.Collisions) != 0 {
			t.Errorf("Collisions = %+v, want none", plan.Collisions)
		}
	})
}

// TestRunMigrateLegacy_CollisionMatrixA_HalfMigrated_RerunDeletesOldOnly is
// slice 4 work item 4, collision-matrix case (a): the relocation
// destination already holds content identical to the source (as if a prior
// migration run relocated the file but was interrupted before deleting the
// old path). A rerun must complete with a delete-old-only operation: no
// collision, and no duplicate write to the destination.
func TestRunMigrateLegacy_CollisionMatrixA_HalfMigrated_RerunDeletesOldOnly(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	forkedContent := legacyOldTestingRuleEdited
	writeMigrationDiskFile(t, target, ".claude/rules/testing.md", forkedContent)
	writeMigrationDiskFile(t, target, ".claude/rules/ralph/testing.md", forkedContent)

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/testing.md", migrateHash("stale-hash-forces-modified"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "collision-a fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions (collision case a): %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}
	if strings.Contains(out.String(), "Collisions (1)") {
		t.Errorf("preview must report zero collisions for case (a):\n%s", out.String())
	}

	if _, statErr := os.Stat(filepath.Join(target, ".claude", "rules", "testing.md")); !os.IsNotExist(statErr) {
		t.Errorf(".claude/rules/testing.md must be deleted; stat err = %v", statErr)
	}

	destPath := filepath.Join(target, ".claude", "rules", "ralph", "testing.md")
	gotDest := mustReadFile(t, destPath)
	if string(gotDest) != forkedContent {
		t.Errorf("relocation destination content changed unexpectedly: got %q, want unchanged %q", gotDest, forkedContent)
	}

	// Case (a) resolves as "already relocated" for the delete, but the
	// content itself is modified (an unmanaged, user-supplied fixture): the
	// destination must be recorded as a fork (Owner=fork), not silently
	// adopted as owner=core, which would permanently lose the fork
	// attribution and leave the path in unresolved drift on the very next
	// upgrade (self-review HIGH-1,
	// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md).
	m2 := readManifestV2(t, target)
	entry, ok := m2.Files[".claude/rules/ralph/testing.md"]
	if !ok || entry.Owner != scaffold.OwnerFork {
		t.Fatalf("manifest entry for .claude/rules/ralph/testing.md = %+v, want Owner=fork (already-resolved relocation of modified content must keep its fork attribution)", entry)
	}
	if entry.ForkedFromVersion != "0.9.0-test" {
		t.Errorf("ForkedFromVersion = %q, want %q", entry.ForkedFromVersion, "0.9.0-test")
	}
}

// TestRunMigrateLegacy_CollisionMatrixB_DestMatchesTemplate_CoreAdopted is
// slice 4 work item 4, collision-matrix case (b): the source is unmodified,
// and the relocation destination already holds exactly the new template's
// content (rather than the source's own content). The old path is deleted,
// the destination is adopted as core, and no collision is reported.
func TestRunMigrateLegacy_CollisionMatrixB_DestMatchesTemplate_CoreAdopted(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	writeMigrationDiskFile(t, target, ".claude/rules/architecture.md", legacyOldArchitectureRule)
	writeMigrationDiskFile(t, target, ".claude/rules/ralph/architecture.md", legacyNewArchitectureRule)

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/architecture.md", migrateHash(legacyOldArchitectureRule)) // unmodified
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "collision-b fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions (collision case b): %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}
	if strings.Contains(out.String(), "Collisions (1)") {
		t.Errorf("preview must report zero collisions for case (b):\n%s", out.String())
	}

	if _, statErr := os.Stat(filepath.Join(target, ".claude", "rules", "architecture.md")); !os.IsNotExist(statErr) {
		t.Errorf(".claude/rules/architecture.md must be deleted; stat err = %v", statErr)
	}
	gotDest := mustReadFile(t, filepath.Join(target, ".claude", "rules", "ralph", "architecture.md"))
	if string(gotDest) != legacyNewArchitectureRule {
		t.Errorf(".claude/rules/ralph/architecture.md = %q, want the new core content it already held %q", gotDest, legacyNewArchitectureRule)
	}

	m2 := readManifestV2(t, target)
	entry, ok := m2.Files[".claude/rules/ralph/architecture.md"]
	if !ok || entry.Owner != scaffold.OwnerCore {
		t.Errorf("manifest entry for .claude/rules/ralph/architecture.md = %+v, want Owner=core (adopted as core, not left a fork)", entry)
	}
}

// poisonedReader panics if ever read from. Used by
// TestRunMigrateLegacy_YesFlag_NeverReadsStdin to prove --yes
// (opts.Yes=true) skips the confirmation prompt without touching stdin at
// all, rather than merely reading an empty/EOF line.
type poisonedReader struct{}

func (poisonedReader) Read([]byte) (int, error) {
	panic("stdin must not be read when --yes (opts.Yes) is set")
}

// TestRunMigrateLegacy_YesFlag_NeverReadsStdin is slice 4 work item 5.
func TestRunMigrateLegacy_YesFlag_NeverReadsStdin(t *testing.T) {
	target := buildLegacyProject(t)
	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, poisonedReader{}, &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions (--yes): %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Migration complete") {
		t.Errorf("stdout must announce migration completion:\n%s", out.String())
	}
}

// ===================================================================
// Self-review fixes (docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md).
// ===================================================================

// TestRunMigrateLegacy_RerunAfterPartialRelocation_ModifiedSource_DestAdoptsFork
// covers self-review HIGH-1 item (1): a prior interrupted migration run
// already fully relocated a MODIFIED source (relocateMigrationFile's
// write-then-delete completed for this entry) before a later entry's write
// failed and aborted the overall migration before the manifest barrier. On
// rerun, the legacy manifest still records the old (pre-relocation) entry,
// but the old path is now gone from disk while the relocation destination
// already holds the user's forked content. ClassifyMigration must recognize
// this and record the destination as owner=fork -- not silently adopt it as
// owner=core, which would leave it permanently drifted on the very next
// upgrade.
func TestRunMigrateLegacy_RerunAfterPartialRelocation_ModifiedSource_DestAdoptsFork(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	// Old path already gone; the relocation destination already holds the
	// user's forked content (as if a prior interrupted run already wrote
	// the destination and deleted the source, per relocateMigrationFile's
	// write-then-delete ordering).
	forkedContent := legacyOldTestingRuleEdited
	writeMigrationDiskFile(t, target, ".claude/rules/ralph/testing.md", forkedContent)

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/testing.md", migrateHash("stale-hash-forces-modified"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "rerun-after-partial-relocation fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err != nil {
		t.Fatalf("runUpgradeIOWithOptions (rerun after partial relocation): %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	destPath := filepath.Join(target, ".claude", "rules", "ralph", "testing.md")
	gotDest := mustReadFile(t, destPath)
	if string(gotDest) != forkedContent {
		t.Errorf("relocation destination content changed unexpectedly: got %q, want unchanged %q", gotDest, forkedContent)
	}

	m2 := readManifestV2(t, target)
	entry, ok := m2.Files[".claude/rules/ralph/testing.md"]
	if !ok || entry.Owner != scaffold.OwnerFork {
		t.Fatalf("manifest entry for .claude/rules/ralph/testing.md = %+v, want Owner=fork (fork attribution must survive a rerun after partial relocation)", entry)
	}
	if entry.ForkedFromVersion != "0.9.0-test" {
		t.Errorf("ForkedFromVersion = %q, want %q", entry.ForkedFromVersion, "0.9.0-test")
	}

	// No drift on a subsequent upgrade: a forked path is never core-checked
	// against the template, so a second run must succeed cleanly.
	var out2, errOut2 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false); err != nil {
		t.Fatalf("second runUpgradeIOWithOptions (drift check): %v\nstdout:\n%s\nstderr:\n%s", err, out2.String(), errOut2.String())
	}
}

// TestRunMigrateLegacy_ChainedDriftSentinel_SurvivesAsExitSentinel covers
// self-review MEDIUM-1: the chained v2 upgrade's ErrUpgradeDriftRemaining
// sentinel must propagate out of runMigrateLegacy rather than being
// swallowed as a warning, since cmd/ralph/main.go maps it to a dedicated
// exit code (3) distinct from a genuine execution error.
//
// scripts/run-verify.sh is a core-owned template path the legacy manifest
// never tracks at all, with disk content that diverges from the new
// template: buildMigratedManifest's generic desired-state sweep
// optimistically records DiskHash=templateHash for it (untracked paths are
// never disk-compared for owner=core -- see that function's own doc
// comment), so the chained v2 upgrade's classifyCore has no way to tell
// this apart from genuine drift and correctly reports it as such.
func TestRunMigrateLegacy_ChainedDriftSentinel_SurvivesAsExitSentinel(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", "#!/bin/sh\necho pre-existing-divergent-content\n")
	writeMigrationDiskFile(t, target, ".claude/rules/testing.md", "unmodified testing rule")

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/testing.md", migrateHash("unmodified testing rule"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "drift-sentinel fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("expected ErrUpgradeDriftRemaining to survive the chained v2 upgrade, got nil")
	}
	if !errors.Is(err, ErrUpgradeDriftRemaining) {
		t.Errorf("err = %v, want errors.Is(err, ErrUpgradeDriftRemaining)", err)
	}
	if strings.Contains(errOut.String(), "Warning: migration completed, but the chained v2 upgrade reported an issue") {
		t.Errorf("drift must not be reported through the generic warning path:\n%s", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Unresolved drift") {
		t.Errorf("stderr must report the unresolved drift path:\n%s", errOut.String())
	}

	// The migration barrier itself must still have committed: it succeeded
	// before the chained call ran, so the drift sentinel from that later,
	// independent step must not roll it back.
	m2 := readManifestV2(t, target)
	if m2.Meta.Layout != scaffold.LayoutV2 {
		t.Errorf("Meta.Layout after migration = %q, want %q (the migration barrier must commit even though the chained upgrade reports drift)", m2.Meta.Layout, scaffold.LayoutV2)
	}
}

// TestRunMigrateLegacy_SymlinkedDeleteParent_ZeroWrites covers self-review
// MEDIUM-2: a delete-kind migration op (OpDeleteOldPath) must validate its
// own parent chain before any write, the same as a relocation's write
// target already does (see TestRunMigrateLegacy_SymlinkedRelocationDestParent_ZeroWrites).
func TestRunMigrateLegacy_SymlinkedDeleteParent_ZeroWrites(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	// legacy/retired.md is unmodified and absent from the new template set,
	// so ClassifyMigration marks it OpDeleteOldPath. Its parent directory
	// is a symlink pointing outside target: the delete's own parent-chain
	// must be refused before any write.
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(target, "legacy")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if err := os.WriteFile(filepath.Join(external, "retired.md"), []byte("retired content"), 0644); err != nil {
		t.Fatalf("WriteFile through symlink for fixture setup: %v", err)
	}

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile("legacy/retired.md", migrateHash("retired content"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "symlinked-delete-parent fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	manifestBefore := mustReadFile(t, manifestPath)

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("migration deleting a path through a symlinked parent: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "preflight") {
		t.Errorf("err = %v, want a preflight-check error naming the unsafe path", err)
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Errorf("a preflight refusal must not touch the legacy manifest:\nbefore: %s\nafter:  %s", manifestBefore, manifestAfter)
	}
	info, lerr := os.Lstat(filepath.Join(target, "legacy"))
	if lerr != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("legacy must remain the untouched symlink: info=%+v err=%v", info, lerr)
	}
	externalEntries, rerr := os.ReadDir(external)
	if rerr != nil {
		t.Fatalf("reading external dir: %v", rerr)
	}
	if len(externalEntries) != 1 {
		t.Errorf("a preflight refusal must not delete through the symlink into the external dir; entries=%v", externalEntries)
	}
}

// TestPruneLegacySettingsHooks_ArgumentVariant_LeftInPlaceAsNearMiss covers
// self-review MEDIUM-4: an argument-carrying variant of a known legacy hook
// command (e.g. "--verbose") must survive pruning (exact match only, per
// design decision -- argument-modified commands are treated as a deliberate
// user customization) and must be reported as a near-miss, not silently
// counted among the removed commands.
func TestPruneLegacySettingsHooks_ArgumentVariant_LeftInPlaceAsNearMiss(t *testing.T) {
	content := []byte(`{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [
        {"type": "command", "command": "./.claude/hooks/pre_bash_guard.sh --verbose"},
        {"type": "command", "command": "./.claude/hooks/session_start_context.sh"}
      ]}
    ]
  }
}
`)
	candidates := prunedLegacyHookCommands(content)

	pruned, removed, nearMisses, err := pruneLegacySettingsHooks(content, candidates)
	if err != nil {
		t.Fatalf("pruneLegacySettingsHooks: %v", err)
	}

	if len(removed) != 1 || removed[0] != "./.claude/hooks/session_start_context.sh" {
		t.Errorf("removed = %v, want exactly [\"./.claude/hooks/session_start_context.sh\"]", removed)
	}
	if len(nearMisses) != 1 || nearMisses[0] != "./.claude/hooks/pre_bash_guard.sh --verbose" {
		t.Errorf("nearMisses = %v, want exactly the argument-carrying variant", nearMisses)
	}
	if !bytes.Contains(pruned, []byte("./.claude/hooks/pre_bash_guard.sh --verbose")) {
		t.Errorf("pruned settings.json must still contain the argument-carrying variant untouched:\n%s", pruned)
	}
	if bytes.Contains(pruned, []byte("./.claude/hooks/session_start_context.sh\"")) {
		t.Errorf("pruned settings.json must not still contain the exact-match command:\n%s", pruned)
	}
}

// TestRunMigrateLegacy_SettingsPruneReport_NearMissNotClaimedAsRemoved is the
// end-to-end counterpart of the unit test above: the migration report must
// not claim an argument-carrying variant was removed when it was actually
// left in place (self-review MEDIUM-4's core complaint -- the report and
// the actual pruning behavior must agree).
func TestRunMigrateLegacy_SettingsPruneReport_NearMissNotClaimedAsRemoved(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	settingsWithVariant := `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "./.claude/hooks/pre_bash_guard.sh --verbose"}]}
    ]
  }
}
`
	writeMigrationDiskFile(t, target, pathSettings, settingsWithVariant)

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(pathSettings, migrateHash("stale-settings-hash"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "settings near-miss fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	reportPaths, globErr := filepath.Glob(filepath.Join(target, "docs", "reports", "ralph-migration-*.md"))
	if globErr != nil || len(reportPaths) != 1 {
		t.Fatalf("expected exactly one ralph-migration-*.md report, got %v (err=%v)", reportPaths, globErr)
	}
	report := string(mustReadFile(t, reportPaths[0]))
	if !strings.Contains(report, "No known legacy direct-invocation hook commands were removed") {
		t.Errorf("migration report must not claim any commands were removed (the only settings.json entry is an argument-carrying variant, never an exact match):\n%s", report)
	}
	if !strings.Contains(report, "left in place because their arguments differ") || !strings.Contains(report, "./.claude/hooks/pre_bash_guard.sh --verbose") {
		t.Errorf("migration report must surface the near-miss so an operator does not mistake its survival for an oversight:\n%s", report)
	}
}

// TestRunMigrateLegacy_SymlinkedAdoptForkDestParent_ZeroWrites covers AR#1
// (docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md): an
// OpDeleteOldPathAdoptFork entry's NewPath (the relocation destination whose
// content is trusted and adopted as the fork record — see
// buildMigratedManifest's forkByPath handling) must be preflight-validated
// the same way a write target is, even though executeMigrationEntries never
// actually writes to it. Collision-matrix case (a) with a modified source
// (see TestClassifyMigration_CollisionMatrix) puts NewPath's content on disk
// through a symlinked parent directory; validateMigrationOp must refuse this
// before OldPath is deleted.
func TestRunMigrateLegacy_SymlinkedAdoptForkDestParent_ZeroWrites(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	const forkedContent = "# Architecture (user edited)\n\nadopted fork content.\n"

	// Old path still holds the modified content on disk (must survive a
	// preflight refusal). The relocation destination's directory
	// (.claude/rules/ralph) is a symlink pointing outside target, holding
	// content matching the source — collision-matrix case (a) — so
	// ClassifyMigration plans OpDeleteOldPathAdoptFork without the AR#1 fix
	// ever inspecting NewPath's parent chain.
	writeMigrationDiskFile(t, target, ".claude/rules/architecture.md", forkedContent)

	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(target, ".claude", "rules", "ralph")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if err := os.WriteFile(filepath.Join(external, "architecture.md"), []byte(forkedContent), 0644); err != nil {
		t.Fatalf("WriteFile through symlink for fixture setup: %v", err)
	}

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/architecture.md", migrateHash("stale-hash-forces-modified"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "symlinked adopt-fork dest fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	manifestBefore := mustReadFile(t, manifestPath)

	var out, errOut bytes.Buffer
	err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false)
	if err == nil {
		t.Fatal("migration adopting a fork through a symlinked destination parent: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "preflight") {
		t.Errorf("err = %v, want a preflight-check error naming the unsafe path", err)
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Errorf("a preflight refusal must not touch the legacy manifest:\nbefore: %s\nafter:  %s", manifestBefore, manifestAfter)
	}

	gotOld := mustReadFile(t, filepath.Join(target, ".claude", "rules", "architecture.md"))
	if string(gotOld) != forkedContent {
		t.Errorf("a preflight refusal must leave OldPath surviving with its original content: got %q, want %q", gotOld, forkedContent)
	}
	info, lerr := os.Lstat(filepath.Join(target, ".claude", "rules", "ralph"))
	if lerr != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf(".claude/rules/ralph must remain the untouched symlink: info=%+v err=%v", info, lerr)
	}
	externalEntries, rerr := os.ReadDir(external)
	if rerr != nil {
		t.Fatalf("reading external dir: %v", rerr)
	}
	if len(externalEntries) != 1 {
		t.Errorf("a preflight refusal must not write through the symlink into the external dir; entries=%v", externalEntries)
	}
}

// TestRunMigrateLegacy_UntrackedSeedCollision_AdvisorySurvivesChainedUpgrade
// covers AR#2 (docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md):
// a path with no legacy manifest entry at all ("untracked"), whose resolved
// v3 owner is seed, and whose pre-existing disk content diverges from the
// new template, must not be recorded into the migrated v3 manifest with
// TemplateHash=current. Doing so would make the chained v2 upgrade's
// classifySeed see "template unchanged since last recorded application" and
// silently no-op, swallowing the seed advisory that
// internal/upgrade/replaceplan.go's classifyUntracked would otherwise raise
// for exactly this shape.
func TestRunMigrateLegacy_UntrackedSeedCollision_AdvisorySurvivesChainedUpgrade(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	const localRalphToml = "[pipeline]\nmodel = \"pre-existing-local-choice\"\n"
	writeMigrationDiskFile(t, target, "ralph.toml", localRalphToml)

	// A minimal legacy manifest that never tracked ralph.toml at all (an old
	// enough legacy version, or a file the user created by hand before the
	// v2 template started shipping a seed there).
	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/testing.md", migrateHash("unmodified testing rule"))
	writeMigrationDiskFile(t, target, ".claude/rules/testing.md", "unmodified testing rule")
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "untracked seed collision fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	// -- the pre-existing local file is left untouched --
	gotRalphToml := mustReadFile(t, filepath.Join(target, "ralph.toml"))
	if string(gotRalphToml) != localRalphToml {
		t.Errorf("ralph.toml = %q, want untouched local content %q", gotRalphToml, localRalphToml)
	}

	// -- the v3 manifest records owner=seed with DiskHash matching the local
	// content, not the template --
	m2 := readManifestV2(t, target)
	entry, ok := m2.Files["ralph.toml"]
	if !ok {
		t.Fatalf("ralph.toml must be tracked in the v3 manifest after the chained upgrade, got no entry")
	}
	if entry.Owner != scaffold.OwnerSeed {
		t.Errorf("ralph.toml Owner = %q, want %q", entry.Owner, scaffold.OwnerSeed)
	}
	if entry.DiskHash != migrateHash(localRalphToml) {
		t.Errorf("ralph.toml DiskHash = %q, want the local content's hash %q", entry.DiskHash, migrateHash(localRalphToml))
	}

	// -- the advisory is present in the chained upgrade's own report --
	upgradeReportPaths, globErr := filepath.Glob(filepath.Join(target, "docs", "reports", "upgrade-*.md"))
	if globErr != nil || len(upgradeReportPaths) != 1 {
		t.Fatalf("expected exactly one upgrade-*.md report, got %v (err=%v)", upgradeReportPaths, globErr)
	}
	upgradeReport := string(mustReadFile(t, upgradeReportPaths[0]))
	if !strings.Contains(upgradeReport, "## Advisories") {
		t.Errorf("upgrade report must contain an Advisories section:\n%s", upgradeReport)
	}
	if !strings.Contains(upgradeReport, "`ralph.toml` (owner: seed)") {
		t.Errorf("upgrade report must surface the ralph.toml seed advisory:\n%s", upgradeReport)
	}

	// -- a subsequent run is a no-op: the advisory is one-shot, cleared by
	// the manifest rebuild that just ran --
	before := snapshotTreeHashesExcluding(t, target)
	var out2, errOut2 bytes.Buffer
	err2 := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out2, &errOut2, false)
	after := snapshotTreeHashesExcluding(t, target)
	if err2 != nil {
		t.Errorf("subsequent run must succeed as a no-op: %v\nstdout:\n%s\nstderr:\n%s", err2, out2.String(), errOut2.String())
	}
	if !slicesEqualStrings(before, after) {
		t.Errorf("subsequent run must write nothing (advisory already cleared); before=%v after=%v", before, after)
	}
}

// TestRunMigrateLegacy_AdoptedForkDiff_IncludedInReport covers AR#3
// (docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md): the
// migration report's fork-diff section must include OpDeleteOldPathAdoptFork
// entries, not just OpForkRelocate/OpForkInPlace — collision-matrix case (a)
// with a modified source (see TestClassifyMigration_CollisionMatrix)
// produces exactly this kind, and its adopted content is otherwise never
// surfaced anywhere in the report.
func TestRunMigrateLegacy_AdoptedForkDiff_IncludedInReport(t *testing.T) {
	isolateGitConfig(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "project")
	if err := os.MkdirAll(filepath.Join(target, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}

	const adoptedContent = "# Architecture (user edited)\n\nadopted fork content.\n"

	// Old path and the relocation destination already hold identical
	// modified content — collision-matrix case (a) — so ClassifyMigration
	// adopts the destination as a fork (OpDeleteOldPathAdoptFork) rather
	// than treating it as a plain relocation.
	writeMigrationDiskFile(t, target, ".claude/rules/architecture.md", adoptedContent)
	writeMigrationDiskFile(t, target, ".claude/rules/ralph/architecture.md", adoptedContent)

	m := scaffold.NewManifest("0.9.0-test")
	m.SetFile(".claude/rules/architecture.md", migrateHash("stale-hash-forces-modified"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	runMigrationTestGit(t, target, "init")
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "adopt-fork report fixture")

	scaffold.EmbeddedFS = legacyFixtureNewTemplateFS()
	Version = "2.0.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Yes: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}

	reportPaths, globErr := filepath.Glob(filepath.Join(target, "docs", "reports", "ralph-migration-*.md"))
	if globErr != nil || len(reportPaths) != 1 {
		t.Fatalf("expected exactly one ralph-migration-*.md report, got %v (err=%v)", reportPaths, globErr)
	}
	report := string(mustReadFile(t, reportPaths[0]))
	if !strings.Contains(report, "## Fork diffs") {
		t.Errorf("report must contain a Fork diffs section:\n%s", report)
	}
	if !strings.Contains(report, "### `.claude/rules/ralph/architecture.md`") {
		t.Errorf("report must include the adopted fork's path:\n%s", report)
	}
	if strings.Contains(report, "### `.claude/rules/ralph/architecture.md`\n\n_No differences") {
		t.Errorf("the adopted fork diverges from the new core content; report must show a diff, not \"no differences\":\n%s", report)
	}
}
