package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteBaseline_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# AGENTS\n")
	baselinePath, err := WriteBaseline(dir, "AGENTS.md", content)
	if err != nil {
		t.Fatalf("WriteBaseline: %v", err)
	}
	if baselinePath != ".ralph/baseline/AGENTS.md" {
		t.Fatalf("baselinePath = %q, want .ralph/baseline/AGENTS.md", baselinePath)
	}

	got, err := ReadBaseline(dir, ManifestFile{
		BaselineStatus: BaselineStatusAvailable,
		BaselinePath:   baselinePath,
	})
	if err != nil {
		t.Fatalf("ReadBaseline: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("baseline content = %q, want %q", got, content)
	}
}

func TestWriteBaseline_RejectsEscapingPath(t *testing.T) {
	dir := t.TempDir()
	if _, err := WriteBaseline(dir, "../escape.md", []byte("x")); err == nil {
		t.Fatal("WriteBaseline accepted escaping path")
	}
	if _, err := os.Stat(filepath.Join(dir, "escape.md")); !os.IsNotExist(err) {
		t.Fatalf("escape.md should not be written, stat err=%v", err)
	}
}

func TestReadBaseline_RejectsOutsideBaselineRoot(t *testing.T) {
	dir := t.TempDir()
	entry := ManifestFile{
		BaselineStatus: BaselineStatusAvailable,
		BaselinePath:   "AGENTS.md",
	}
	if _, err := ReadBaseline(dir, entry); err == nil {
		t.Fatal("ReadBaseline accepted path outside .ralph/baseline")
	}
}
