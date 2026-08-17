package upgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func writeAdvisoryFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestRenderAdvisoryDiffs_IdenticalContents(t *testing.T) {
	dir := t.TempDir()
	writeAdvisoryFile(t, dir, "AGENTS.md", "same\n")
	tmplFS := fstest.MapFS{
		"AGENTS.md": {Data: []byte("same\n")},
	}

	got, err := RenderAdvisoryDiffs(dir, tmplFS, "v2.0.0", []AdvisoryEntry{
		{Path: "AGENTS.md", Owner: "fork"},
	})
	if err != nil {
		t.Fatalf("RenderAdvisoryDiffs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].Diff != "" {
		t.Errorf("expected empty diff for identical contents, got:\n%s", got[0].Diff)
	}
	if got[0].Skipped {
		t.Errorf("expected Skipped=false for identical contents")
	}
}

func TestRenderAdvisoryDiffs_ChangedContents(t *testing.T) {
	dir := t.TempDir()
	writeAdvisoryFile(t, dir, "AGENTS.md", "local version\n")
	tmplFS := fstest.MapFS{
		"AGENTS.md": {Data: []byte("template version\n")},
	}

	got, err := RenderAdvisoryDiffs(dir, tmplFS, "v2.0.0", []AdvisoryEntry{
		{Path: "AGENTS.md", Owner: "fork"},
	})
	if err != nil {
		t.Fatalf("RenderAdvisoryDiffs: %v", err)
	}
	diff := got[0].Diff
	if !strings.Contains(diff, "--- local\n") {
		t.Errorf("expected local label, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++ template (v2.0.0)\n") {
		t.Errorf("expected template version label, got:\n%s", diff)
	}
	if !strings.Contains(diff, "-local version\n") {
		t.Errorf("expected removed local line, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+template version\n") {
		t.Errorf("expected added template line, got:\n%s", diff)
	}
}

func TestRenderAdvisoryDiffs_MissingDiskFile(t *testing.T) {
	dir := t.TempDir()
	tmplFS := fstest.MapFS{
		"seed/new-file.md": {Data: []byte("brand new\ncontent\n")},
	}

	got, err := RenderAdvisoryDiffs(dir, tmplFS, "v2.0.0", []AdvisoryEntry{
		{Path: "seed/new-file.md", Owner: "seed"},
	})
	if err != nil {
		t.Fatalf("RenderAdvisoryDiffs: %v", err)
	}
	diff := got[0].Diff
	if got[0].Skipped {
		t.Errorf("expected Skipped=false for missing disk file")
	}
	if strings.Contains(diff, "-brand new") || strings.Contains(diff, "-content") {
		t.Errorf("expected no deletions for full-addition diff, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+brand new\n") || !strings.Contains(diff, "+content\n") {
		t.Errorf("expected full-addition diff content, got:\n%s", diff)
	}
	if !strings.Contains(diff, "@@ 旧 (空)") {
		t.Errorf("expected empty-old-side range header, got:\n%s", diff)
	}
}

func TestRenderAdvisoryDiffs_MissingTemplateFile(t *testing.T) {
	dir := t.TempDir()
	writeAdvisoryFile(t, dir, "fork/removed.md", "still here on disk\n")
	tmplFS := fstest.MapFS{
		"other.md": {Data: []byte("unrelated\n")},
	}

	got, err := RenderAdvisoryDiffs(dir, tmplFS, "v2.0.0", []AdvisoryEntry{
		{Path: "fork/removed.md", Owner: "fork"},
	})
	if err != nil {
		t.Fatalf("RenderAdvisoryDiffs: %v", err)
	}
	if !got[0].Skipped {
		t.Errorf("expected Skipped=true when template no longer has the path")
	}
	if got[0].Diff != "" {
		t.Errorf("expected empty diff when skipped, got:\n%s", got[0].Diff)
	}
	if got[0].Note == "" {
		t.Errorf("expected a non-empty note explaining the skip")
	}
}

func TestRenderAdvisoryDiffs_InvalidPath(t *testing.T) {
	dir := t.TempDir()
	tmplFS := fstest.MapFS{}

	_, err := RenderAdvisoryDiffs(dir, tmplFS, "v2.0.0", []AdvisoryEntry{
		{Path: "../escape.md", Owner: "fork"},
	})
	if err == nil {
		t.Fatalf("expected an error for a path escaping the local tree")
	}
}
