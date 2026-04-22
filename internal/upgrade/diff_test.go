package upgrade

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
)

func setupTestProject(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()

	// Create a file on disk.
	if err := os.WriteFile(filepath.Join(dir, "file.md"), []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create manifest.
	m := scaffold.NewManifest("0.1.0")
	m.SetFile("file.md", scaffold.HashBytes([]byte("original")))

	manifestDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(manifestDir, "manifest.toml")
	if err := m.Write(manifestPath); err != nil {
		t.Fatal(err)
	}

	return dir, manifestPath
}

func TestComputeDiffs_AutoUpdate(t *testing.T) {
	dir, manifestPath := setupTestProject(t)

	// New template with updated content.
	newFS := fstest.MapFS{
		"file.md": {Data: []byte("updated")},
	}

	diffs, err := ComputeDiffs(manifestPath, dir, newFS)
	if err != nil {
		t.Fatal(err)
	}

	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionAutoUpdate {
		t.Errorf("action = %d, want ActionAutoUpdate (%d)", diffs[0].Action, ActionAutoUpdate)
	}
}

func TestComputeDiffs_Conflict(t *testing.T) {
	dir, manifestPath := setupTestProject(t)

	// User edits the file.
	if err := os.WriteFile(filepath.Join(dir, "file.md"), []byte("user edit"), 0644); err != nil {
		t.Fatal(err)
	}

	// New template also changes.
	newFS := fstest.MapFS{
		"file.md": {Data: []byte("updated")},
	}

	diffs, err := ComputeDiffs(manifestPath, dir, newFS)
	if err != nil {
		t.Fatal(err)
	}

	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionConflict {
		t.Errorf("action = %d, want ActionConflict (%d)", diffs[0].Action, ActionConflict)
	}
}

func TestComputeDiffs_AddNewFile(t *testing.T) {
	dir, manifestPath := setupTestProject(t)

	newFS := fstest.MapFS{
		"file.md":     {Data: []byte("original")},
		"new-file.md": {Data: []byte("new content")},
	}

	diffs, err := ComputeDiffs(manifestPath, dir, newFS)
	if err != nil {
		t.Fatal(err)
	}

	var addCount int
	for _, d := range diffs {
		if d.Action == ActionAdd {
			addCount++
		}
	}
	if addCount != 1 {
		t.Errorf("add count = %d, want 1", addCount)
	}
}

func TestComputeDiffs_RemoveFile(t *testing.T) {
	dir, manifestPath := setupTestProject(t)

	// New template doesn't have file.md anymore.
	newFS := fstest.MapFS{
		"other.md": {Data: []byte("other")},
	}

	diffs, err := ComputeDiffs(manifestPath, dir, newFS)
	if err != nil {
		t.Fatal(err)
	}

	var removeCount int
	for _, d := range diffs {
		if d.Action == ActionRemove {
			removeCount++
		}
	}
	if removeCount != 1 {
		t.Errorf("remove count = %d, want 1", removeCount)
	}
}

// Regression: ActionSkip must carry NewHash so callers can rewrite the
// manifest with a real hash instead of an empty string.
func TestComputeDiffs_Skip_PreservesHash(t *testing.T) {
	dir, manifestPath := setupTestProject(t)

	// Same content in newFS → template unchanged → ActionSkip.
	content := []byte("original")
	newFS := fstest.MapFS{
		"file.md": {Data: content},
	}

	diffs, err := ComputeDiffs(manifestPath, dir, newFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionSkip {
		t.Fatalf("action = %d, want ActionSkip", diffs[0].Action)
	}
	want := scaffold.HashBytes(content)
	if diffs[0].NewHash != want {
		t.Errorf("NewHash = %q, want %q", diffs[0].NewHash, want)
	}
}

// Regression: a pack-scoped manifest subset (keys stripped of the pack prefix)
// should resolve pack FS paths as ActionSkip when disk + template match,
// not mis-classify them as ActionAdd.
func TestComputeDiffsWithManifest_PackPrefixedSubset(t *testing.T) {
	dir := t.TempDir()

	// Simulate pack files rendered to disk under packs/languages/golang/.
	packDir := filepath.Join(dir, "packs", "languages", "golang")
	if err := os.MkdirAll(packDir, 0755); err != nil {
		t.Fatal(err)
	}
	readme := []byte("pack readme")
	if err := os.WriteFile(filepath.Join(packDir, "README.md"), readme, 0644); err != nil {
		t.Fatal(err)
	}

	// Full manifest keys are namespaced under packs/languages/golang/,
	// but we pass a scoped subset with the prefix stripped.
	packManifest := scaffold.NewManifest("0.1.0")
	packManifest.SetFile("README.md", scaffold.HashBytes(readme))

	packFS := fstest.MapFS{
		"README.md": {Data: readme},
	}

	diffs, err := ComputeDiffsWithManifest(packManifest, packDir, packFS, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionSkip {
		t.Fatalf("action = %d, want ActionSkip (got Add=%d)", diffs[0].Action, ActionAdd)
	}
}

// Regression: an empty-hash manifest entry (caused by the prior bug) should
// self-heal when the on-disk content matches the template — no conflict, no
// user prompt, just a hash repair via ActionSkip.
func TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate(t *testing.T) {
	dir := t.TempDir()
	content := []byte("pristine")
	if err := os.WriteFile(filepath.Join(dir, "file.md"), content, 0644); err != nil {
		t.Fatal(err)
	}

	// Simulate corrupted manifest (hash = "").
	m := scaffold.NewManifest("0.1.0")
	m.SetFile("file.md", "")

	newFS := fstest.MapFS{
		"file.md": {Data: content},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionSkip {
		t.Errorf("action = %d, want ActionSkip", diffs[0].Action)
	}
	if diffs[0].NewHash == "" {
		t.Errorf("NewHash should be populated for heal")
	}
}

// Empty-hash entries where disk differs from template must still surface as
// conflicts so the user is asked instead of silently overwriting edits.
// Additionally, the conflict must carry OldHash=newHash so that a non-
// interactive "skip" resolution rewrites the manifest with a real hash
// and ends the perpetual-conflict loop.
func TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.md"), []byte("user edit"), 0644); err != nil {
		t.Fatal(err)
	}
	m := scaffold.NewManifest("0.1.0")
	m.SetFile("file.md", "")

	template := []byte("template")
	newFS := fstest.MapFS{
		"file.md": {Data: template},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionConflict {
		t.Fatalf("action = %d, want ActionConflict", diffs[0].Action)
	}
	wantHash := scaffold.HashBytes(template)
	if diffs[0].OldHash != wantHash {
		t.Errorf("OldHash = %q, want newHash %q (heal contract)", diffs[0].OldHash, wantHash)
	}
}

// Regression: a file absent from the manifest but present on disk with
// content that differs from the template must surface as ActionConflict,
// not ActionAdd. Prior behavior would silently overwrite the user's file
// when a later template release reintroduced a previously-removed path.
func TestComputeDiffs_AddBecomesConflictWhenDiskDiffers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "returning.md"), []byte("user kept this"), 0644); err != nil {
		t.Fatal(err)
	}
	m := scaffold.NewManifest("0.1.0") // no entry for returning.md

	newFS := fstest.MapFS{
		"returning.md": {Data: []byte("new template version")},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionConflict {
		t.Fatalf("action = %d, want ActionConflict (reintroduction safeguard)", diffs[0].Action)
	}
	wantHash := scaffold.HashBytes([]byte("new template version"))
	if diffs[0].OldHash != wantHash {
		t.Errorf("OldHash = %q, want newHash %q (one-run convergence contract)", diffs[0].OldHash, wantHash)
	}
	if diffs[0].NewHash != wantHash {
		t.Errorf("NewHash = %q, want %q", diffs[0].NewHash, wantHash)
	}
}

// Template unchanged but disk drifted from the recorded hash: must surface as
// a conflict so the user is asked to keep the local variant, overwrite, or
// view a diff. Prior behavior silently ActionSkip'd and never noticed the
// local edit.
func TestComputeDiffs_LocalEditWithUnchangedTemplate(t *testing.T) {
	dir, manifestPath := setupTestProject(t)

	// User edits the file locally; template does NOT change.
	if err := os.WriteFile(filepath.Join(dir, "file.md"), []byte("user edit"), 0644); err != nil {
		t.Fatal(err)
	}
	newFS := fstest.MapFS{
		"file.md": {Data: []byte("original")}, // same as manifest hash
	}

	diffs, err := ComputeDiffs(manifestPath, dir, newFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionConflict {
		t.Errorf("action = %d, want ActionConflict (template unchanged + local edit)", diffs[0].Action)
	}
	if diffs[0].NewContent == nil {
		t.Errorf("NewContent must be populated so the diff viewer can render template side")
	}
	if diffs[0].DiskHash == diffs[0].OldHash {
		t.Errorf("DiskHash %q should differ from OldHash %q (disk drifted)", diffs[0].DiskHash, diffs[0].OldHash)
	}
}

// Manifest entries marked Managed=false represent files the user has taken
// ownership of via a prior skip resolution. They must silent-skip regardless
// of template changes or disk drift until an explicit resync brings them back
// under template management.
func TestComputeDiffs_Unmanaged_IsSilentSkip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "owned.md"), []byte("user owned"), 0644); err != nil {
		t.Fatal(err)
	}

	m := scaffold.NewManifest("0.1.0")
	m.SetFileUnmanaged("owned.md", scaffold.HashBytes([]byte("user owned")))

	// Even when the template diverges, the unmanaged entry must not
	// surface as auto-update or conflict.
	newFS := fstest.MapFS{
		"owned.md": {Data: []byte("template wants to change this")},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionSkip {
		t.Errorf("action = %d, want ActionSkip (unmanaged → silent)", diffs[0].Action)
	}
}

// Unmanaged entries must survive template-side removal. Prior behavior emitted
// ActionRemove which dropped the manifest entry, breaking the "user owns this
// forever until --resync" contract when the template later reintroduced the
// same path (the reintroduction became an add/conflict instead of silent skip).
func TestComputeDiffs_Unmanaged_SurvivesTemplateRemoval(t *testing.T) {
	dir := t.TempDir()
	m := scaffold.NewManifest("0.1.0")
	m.SetFileUnmanaged("kept-by-user.md", "sha256:userhash")

	// Template no longer ships kept-by-user.md.
	newFS := fstest.MapFS{
		"other.md": {Data: []byte("other")},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	var skipFound bool
	for _, d := range diffs {
		if d.Path == "kept-by-user.md" {
			if d.Action != ActionSkip {
				t.Errorf("action = %d, want ActionSkip (unmanaged must survive removal)", d.Action)
			}
			skipFound = true
		}
		if d.Path == "kept-by-user.md" && d.Action == ActionRemove {
			t.Error("unmanaged entry must not surface as ActionRemove")
		}
	}
	if !skipFound {
		t.Error("unmanaged entry missing from diffs — it was silently dropped")
	}
}

// Unmanaged-skip entries must carry NewContent so the caller can re-adopt
// them under `--force`. Without this, the force path cannot restore template
// coverage in one invocation.
func TestComputeDiffs_Unmanaged_CarriesNewContentForForceReadoption(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "owned.md"), []byte("local"), 0644); err != nil {
		t.Fatal(err)
	}
	m := scaffold.NewManifest("0.1.0")
	m.SetFileUnmanaged("owned.md", scaffold.HashBytes([]byte("local")))

	template := []byte("template content")
	newFS := fstest.MapFS{
		"owned.md": {Data: template},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].NewContent == nil {
		t.Error("unmanaged skip must carry NewContent so --force can re-adopt")
	}
	if string(diffs[0].NewContent) != string(template) {
		t.Errorf("NewContent = %q, want template bytes", diffs[0].NewContent)
	}
}

// Unmanaged entries where the user later deletes the file on disk must also
// silent-skip: the entry exists to block re-adoption, not to enforce
// presence. This prevents a surprise re-add if the template still ships the
// file.
func TestComputeDiffs_Unmanaged_SilentSkipWhenDiskMissing(t *testing.T) {
	dir := t.TempDir()
	m := scaffold.NewManifest("0.1.0")
	m.SetFileUnmanaged("gone.md", "deadbeef")

	newFS := fstest.MapFS{
		"gone.md": {Data: []byte("template content")},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("diffs = %d, want 1", len(diffs))
	}
	if diffs[0].Action != ActionSkip {
		t.Errorf("action = %d, want ActionSkip (unmanaged + missing disk → silent)", diffs[0].Action)
	}
}

// Safe-add: if the disk file already matches the new template, ActionAdd is safe
// (no conflict prompt needed). This covers the no-op re-add case.
func TestComputeDiffs_AddStaysAddWhenDiskMatchesTemplate(t *testing.T) {
	dir := t.TempDir()
	content := []byte("identical")
	if err := os.WriteFile(filepath.Join(dir, "same.md"), content, 0644); err != nil {
		t.Fatal(err)
	}
	m := scaffold.NewManifest("0.1.0")

	newFS := fstest.MapFS{
		"same.md": {Data: content},
	}

	diffs, err := ComputeDiffsWithManifest(m, dir, newFS, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 || diffs[0].Action != ActionAdd {
		t.Fatalf("action = %v, want ActionAdd", diffs)
	}
}
