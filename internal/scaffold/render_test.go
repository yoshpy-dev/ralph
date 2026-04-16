package scaffold

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestRenderFS_CreatesFiles(t *testing.T) {
	src := fstest.MapFS{
		"foo.md":     {Data: []byte("# Foo")},
		"dir/bar.md": {Data: []byte("# Bar")},
	}

	dir := t.TempDir()
	result, hashes, err := RenderFS(src, RenderOptions{TargetDir: dir})
	if err != nil {
		t.Fatalf("RenderFS: %v", err)
	}

	if len(result.Created) != 2 {
		t.Errorf("created = %d, want 2", len(result.Created))
	}
	if len(hashes) != 2 {
		t.Errorf("hashes = %d, want 2", len(hashes))
	}

	// Verify files exist on disk.
	content, err := os.ReadFile(filepath.Join(dir, "foo.md"))
	if err != nil {
		t.Fatalf("reading foo.md: %v", err)
	}
	if string(content) != "# Foo" {
		t.Errorf("foo.md content = %q, want %q", content, "# Foo")
	}

	content, err = os.ReadFile(filepath.Join(dir, "dir", "bar.md"))
	if err != nil {
		t.Fatalf("reading dir/bar.md: %v", err)
	}
	if string(content) != "# Bar" {
		t.Errorf("dir/bar.md content = %q, want %q", content, "# Bar")
	}
}

func TestRenderFS_SkipsExisting(t *testing.T) {
	src := fstest.MapFS{
		"existing.md": {Data: []byte("new content")},
	}

	dir := t.TempDir()
	existingPath := filepath.Join(dir, "existing.md")
	if err := os.WriteFile(existingPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := RenderFS(src, RenderOptions{TargetDir: dir, Overwrite: false})
	if err != nil {
		t.Fatalf("RenderFS: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Errorf("skipped = %d, want 1", len(result.Skipped))
	}

	// Verify original content preserved.
	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "old content" {
		t.Errorf("content = %q, want %q", content, "old content")
	}
}

func TestRenderFS_OverwritesExisting(t *testing.T) {
	src := fstest.MapFS{
		"existing.md": {Data: []byte("new content")},
	}

	dir := t.TempDir()
	existingPath := filepath.Join(dir, "existing.md")
	if err := os.WriteFile(existingPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := RenderFS(src, RenderOptions{TargetDir: dir, Overwrite: true})
	if err != nil {
		t.Fatalf("RenderFS: %v", err)
	}

	if len(result.Overwritten) != 1 {
		t.Errorf("overwritten = %d, want 1", len(result.Overwritten))
	}

	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new content" {
		t.Errorf("content = %q, want %q", content, "new content")
	}
}

func TestHashBytes(t *testing.T) {
	hash := HashBytes([]byte("hello"))
	if hash[:7] != "sha256:" {
		t.Errorf("hash prefix = %q, want %q", hash[:7], "sha256:")
	}
	// SHA256 hex is 64 chars.
	if len(hash) != 7+64 {
		t.Errorf("hash length = %d, want %d", len(hash), 7+64)
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	fileHash, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	bytesHash := HashBytes([]byte("hello"))
	if fileHash != bytesHash {
		t.Errorf("HashFile = %q, HashBytes = %q, want equal", fileHash, bytesHash)
	}
}
