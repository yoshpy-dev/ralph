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

func TestRenderFS_SkipsConfiguredPaths(t *testing.T) {
	src := fstest.MapFS{
		"README.md": {Data: []byte("# Pack")},
		"rule.md":   {Data: []byte("# Rule")},
	}

	dir := t.TempDir()
	result, hashes, err := RenderFS(src, RenderOptions{
		TargetDir: dir,
		SkipPaths: map[string]bool{
			"rule.md": true,
		},
	})
	if err != nil {
		t.Fatalf("RenderFS: %v", err)
	}

	if len(result.Created) != 1 || result.Created[0] != "README.md" {
		t.Errorf("created = %v, want [README.md]", result.Created)
	}
	if _, ok := hashes["rule.md"]; ok {
		t.Fatal("skipped rule.md should not be hashed")
	}
	if _, err := os.Stat(filepath.Join(dir, "rule.md")); !os.IsNotExist(err) {
		t.Errorf("skipped rule.md should not exist; stat err = %v", err)
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

// TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed pins the C3-1
// self-review fix: a *dangling* symlink at a rendered path (target does not
// exist) must be classified as "exists" (skipped) rather than "absent"
// (created). Before the fix, os.Stat on a dangling symlink returns
// ErrNotExist, RenderFS treated the path as a create, and os.WriteFile wrote
// the scaffold content straight through the link to wherever it resolved --
// landing outside TargetDir regardless of the boundary check above.
func TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed(t *testing.T) {
	src := fstest.MapFS{
		"AGENTS.md": {Data: []byte("scaffold content")},
	}

	dir := t.TempDir()
	outsideDir := t.TempDir()
	externalTarget := filepath.Join(outsideDir, "OUTSIDE.md")

	linkPath := filepath.Join(dir, "AGENTS.md")
	if err := os.Symlink(externalTarget, linkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	result, _, err := RenderFS(src, RenderOptions{TargetDir: dir, Overwrite: false})
	if err != nil {
		t.Fatalf("RenderFS: %v", err)
	}

	if len(result.Skipped) != 1 || result.Skipped[0] != "AGENTS.md" {
		t.Errorf("skipped = %v, want [AGENTS.md]", result.Skipped)
	}
	if len(result.Created) != 0 {
		t.Errorf("created = %v, want none (dangling symlink must not be followed)", result.Created)
	}

	// The symlink itself must be untouched: still a symlink, still pointing
	// at the same (nonexistent) external target.
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("AGENTS.md must remain a symlink, got mode %v", info.Mode())
	}
	resolved, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if resolved != externalTarget {
		t.Errorf("symlink target = %q, want %q", resolved, externalTarget)
	}

	// Nothing must have been written at the external (dangling) link target
	// -- this is the containment failure the fix closes.
	if _, err := os.Stat(externalTarget); !os.IsNotExist(err) {
		t.Errorf("dangling symlink target must not be created; stat err = %v", err)
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
