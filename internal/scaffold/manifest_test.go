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
