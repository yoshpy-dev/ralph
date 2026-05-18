package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")

	m := NewManifest("0.1.0")
	m.SetFile("AGENTS.md", "sha256:abc123")
	m.SetFile(".claude/rules/testing.md", "sha256:def456")

	if err := m.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if got.Meta.Version != "0.1.0" {
		t.Errorf("version = %q, want %q", got.Meta.Version, "0.1.0")
	}
	if len(got.Files) != 2 {
		t.Errorf("files count = %d, want 2", len(got.Files))
	}
	if f, ok := got.Files["AGENTS.md"]; !ok || f.Hash != "sha256:abc123" {
		t.Errorf("AGENTS.md file = %+v, want hash sha256:abc123", f)
	}
	if f, ok := got.Files[".claude/rules/testing.md"]; !ok || !f.Managed {
		t.Errorf(".claude/rules/testing.md managed = %v, want true", f.Managed)
	}
}

func TestManifestRoundTripV2BaselineFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")

	m := NewManifest("0.3.0")
	m.SetFileWithBaseline("AGENTS.md", "sha256:abc123", ".ralph/baseline/AGENTS.md")
	m.SetFileResolvedWithBaseline("partial.md", "sha256:template", "sha256:disk", FileStatePartial, ".ralph/baseline/partial.md")
	m.SetFileUnmanaged("CLAUDE.md", "sha256:local")

	if err := m.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	agents := got.Files["AGENTS.md"]
	if !agents.Managed || agents.State != FileStateManaged {
		t.Fatalf("AGENTS.md state = managed:%v state:%q, want managed", agents.Managed, agents.State)
	}
	if agents.TemplateHash != "sha256:abc123" {
		t.Errorf("TemplateHash = %q, want sha256:abc123", agents.TemplateHash)
	}
	if agents.BaselineStatus != BaselineStatusAvailable {
		t.Errorf("BaselineStatus = %q, want available", agents.BaselineStatus)
	}
	if agents.BaselinePath != ".ralph/baseline/AGENTS.md" {
		t.Errorf("BaselinePath = %q", agents.BaselinePath)
	}

	partial := got.Files["partial.md"]
	if !partial.Managed || partial.State != FileStatePartial {
		t.Fatalf("partial.md state = managed:%v state:%q, want partial managed", partial.Managed, partial.State)
	}
	if partial.Hash != "sha256:template" || partial.TemplateHash != "sha256:template" {
		t.Errorf("partial.md template hash fields = hash:%q template:%q", partial.Hash, partial.TemplateHash)
	}
	if partial.DiskHash != "sha256:disk" {
		t.Errorf("partial.md DiskHash = %q, want sha256:disk", partial.DiskHash)
	}

	claude := got.Files["CLAUDE.md"]
	if claude.Managed || claude.State != FileStateUnmanaged {
		t.Fatalf("CLAUDE.md state = managed:%v state:%q, want unmanaged", claude.Managed, claude.State)
	}
	if claude.DiskHash != "sha256:local" {
		t.Errorf("DiskHash = %q, want sha256:local", claude.DiskHash)
	}
}

func TestReadManifestV1Compatibility(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")
	data := []byte(`[meta]
version = "0.1.0"
created = "2026-05-18T00:00:00Z"
updated = "2026-05-18T00:00:00Z"

[files."AGENTS.md"]
hash = "sha256:abc123"
managed = true
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	entry := got.Files["AGENTS.md"]
	if !entry.Managed || entry.Hash != "sha256:abc123" {
		t.Fatalf("v1 fields not preserved: %+v", entry)
	}
	if entry.BaselineStatus != "" || entry.BaselinePath != "" {
		t.Fatalf("v1 manifest should not gain baseline metadata on read: %+v", entry)
	}
	if entry.IsBaselineAvailable() {
		t.Fatal("v1 manifest entry must not be baseline-available")
	}
}

func TestReadManifestNotFound(t *testing.T) {
	_, err := ReadManifest("/nonexistent/manifest.toml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestManifestWriteCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".ralph", "manifest.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}

	m := NewManifest("0.2.0")
	if err := m.Write(path); err != nil {
		t.Fatalf("Write to nested path: %v", err)
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got.Meta.Version != "0.2.0" {
		t.Errorf("version = %q, want %q", got.Meta.Version, "0.2.0")
	}
}
