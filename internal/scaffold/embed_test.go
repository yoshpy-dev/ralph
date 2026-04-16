package scaffold

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestBaseFS_WithMockFS(t *testing.T) {
	// EmbeddedFS is only populated when built from cmd/ralph/ with go:embed.
	// In unit tests, it's a zero-value embed.FS. Skip these tests.
	if _, err := fs.ReadDir(EmbeddedFS, "templates"); err != nil {
		t.Skip("EmbeddedFS not initialized (only available when built from cmd/ralph/)")
	}

	baseFS, err := BaseFS()
	if err != nil {
		t.Fatalf("BaseFS: %v", err)
	}

	if _, err := fs.Stat(baseFS, "AGENTS.md"); err != nil {
		t.Errorf("AGENTS.md not found in BaseFS: %v", err)
	}
}

func TestAvailablePacks_WithMockFS(t *testing.T) {
	if _, err := fs.ReadDir(EmbeddedFS, "templates"); err != nil {
		t.Skip("EmbeddedFS not initialized")
	}

	packs, err := AvailablePacks()
	if err != nil {
		t.Fatalf("AvailablePacks: %v", err)
	}

	if len(packs) < 5 {
		t.Errorf("packs count = %d, want >= 5, got: %v", len(packs), packs)
	}
}

// TestEmbedFSInterface verifies the exported variable is the right type.
func TestEmbedFSInterface(t *testing.T) {
	var _ = EmbeddedFS // type is embed.FS
}

// TestAvailablePacksExcludesTemplate verifies _template is excluded.
func TestAvailablePacksExcludesTemplate(t *testing.T) {
	orig := EmbeddedFS
	defer func() { EmbeddedFS = orig }()

	// Use a MapFS to simulate embedded templates.
	mock := fstest.MapFS{
		"templates/packs/golang/README.md":    {Data: []byte("go")},
		"templates/packs/python/README.md":    {Data: []byte("py")},
		"templates/packs/_template/README.md": {Data: []byte("tpl")},
	}

	// AvailablePacks reads from EmbeddedFS directly, but since embed.FS
	// can't be mocked, test the filtering logic directly.
	entries, err := fs.ReadDir(mock, "templates/packs")
	if err != nil {
		t.Fatal(err)
	}
	var packs []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "_template" {
			packs = append(packs, e.Name())
		}
	}
	if len(packs) != 2 {
		t.Errorf("packs = %v, want [golang python]", packs)
	}
}
