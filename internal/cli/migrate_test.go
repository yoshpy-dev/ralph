package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
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

	t.Run("(a) modified fork-relocate, dest matches source content -> delete-old only, no fork", func(t *testing.T) {
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
		if e.Kind != OpDeleteOldPath {
			t.Errorf("Kind = %v, want OpDeleteOldPath", e.Kind)
		}
		if e.Owner == scaffold.OwnerFork {
			t.Errorf("Owner should not be fork once the relocation already resolved")
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
