package upgrade

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

func mapFS(files map[string]string) fstest.MapFS {
	m := fstest.MapFS{}
	for path, content := range files {
		m[path] = &fstest.MapFile{Data: []byte(content)}
	}
	return m
}

func writeDiskFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func hashOf(content string) string {
	return scaffold.HashBytes([]byte(content))
}

func findOp(t *testing.T, plan ReplacePlan, path string) FileOp {
	t.Helper()
	for _, op := range plan.Ops {
		if op.Path == path {
			return op
		}
	}
	t.Fatalf("no op found for path %q; plan.Ops=%+v", path, plan.Ops)
	return FileOp{}
}

func assertNoOpFor(t *testing.T, plan ReplacePlan, path string) {
	t.Helper()
	for _, op := range plan.Ops {
		if op.Path == path {
			t.Fatalf("unexpected op for path %q: %+v", path, op)
		}
	}
}

// --- owner=core ---

func TestPlanCoreReplace_CoreCreateWhenDiskMissing(t *testing.T) {
	dir := t.TempDir()
	tmpl := mapFS(map[string]string{"AGENTS.md": "new content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("AGENTS.md", scaffold.OwnerCore, hashOf("old content"), hashOf("old content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	op := findOp(t, plan, "AGENTS.md")
	if op.Kind != OpCreate {
		t.Errorf("Kind = %v, want OpCreate", op.Kind)
	}
	if string(op.Content) != "new content" || op.NewHash != hashOf("new content") {
		t.Errorf("op = %+v, want content %q", op, "new content")
	}
	if len(plan.Drift) != 0 || len(plan.Advisories) != 0 {
		t.Errorf("unexpected drift/advisories: %+v / %+v", plan.Drift, plan.Advisories)
	}
}

func TestPlanCoreReplace_CoreUpdateWhenDiskMatchesRecordedAndTemplateChanged(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "AGENTS.md", "old content")
	tmpl := mapFS(map[string]string{"AGENTS.md": "new content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("AGENTS.md", scaffold.OwnerCore, hashOf("old content"), hashOf("old content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	op := findOp(t, plan, "AGENTS.md")
	if op.Kind != OpUpdate || string(op.Content) != "new content" {
		t.Errorf("op = %+v, want OpUpdate with new content", op)
	}
	if len(plan.Drift) != 0 {
		t.Errorf("unexpected drift: %+v", plan.Drift)
	}
}

func TestPlanCoreReplace_CoreRecordedHashFallsBackToLegacyHashField(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "AGENTS.md", "old content")
	tmpl := mapFS(map[string]string{"AGENTS.md": "new content"})
	m := scaffold.NewManifest("0.1.0")
	// Simulate a manifest entry with no DiskHash set (only the legacy Hash
	// field), exercising the "DiskHash fallback Hash" rule.
	m.Files["AGENTS.md"] = scaffold.ManifestFile{
		Hash:    hashOf("old content"),
		Managed: true,
		Owner:   scaffold.OwnerCore,
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	op := findOp(t, plan, "AGENTS.md")
	if op.Kind != OpUpdate {
		t.Errorf("Kind = %v, want OpUpdate (recorded hash should fall back to Hash field)", op.Kind)
	}
}

func TestPlanCoreReplace_CoreNoopWhenTemplateUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "AGENTS.md", "same content")
	tmpl := mapFS(map[string]string{"AGENTS.md": "same content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("AGENTS.md", scaffold.OwnerCore, hashOf("same content"), hashOf("same content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "AGENTS.md")
	if len(plan.ManifestRefresh) != 0 {
		t.Errorf("unexpected manifest refresh: %+v", plan.ManifestRefresh)
	}
	if len(plan.Drift) != 0 {
		t.Errorf("unexpected drift: %+v", plan.Drift)
	}
}

func TestPlanCoreReplace_CoreManifestRefreshWhenDiskAlreadyMatchesNewTemplate(t *testing.T) {
	dir := t.TempDir()
	// Disk already has the *new* template content (e.g. left over from a
	// prior interrupted apply), but the manifest still records the old hash.
	writeDiskFile(t, dir, "AGENTS.md", "new content")
	tmpl := mapFS(map[string]string{"AGENTS.md": "new content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("AGENTS.md", scaffold.OwnerCore, hashOf("old content"), hashOf("old content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "AGENTS.md")
	if len(plan.ManifestRefresh) != 1 || plan.ManifestRefresh[0].Path != "AGENTS.md" || plan.ManifestRefresh[0].Hash != hashOf("new content") {
		t.Fatalf("ManifestRefresh = %+v, want single entry for AGENTS.md with new hash", plan.ManifestRefresh)
	}
	if len(plan.Drift) != 0 {
		t.Errorf("unexpected drift: %+v", plan.Drift)
	}
}

func TestPlanCoreReplace_CoreUnresolvedDriftDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "AGENTS.md", "user-modified content")
	tmpl := mapFS(map[string]string{"AGENTS.md": "new content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("AGENTS.md", scaffold.OwnerCore, hashOf("old content"), hashOf("old content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "AGENTS.md")
	if len(plan.Drift) != 1 {
		t.Fatalf("Drift = %+v, want 1 entry", plan.Drift)
	}
	d := plan.Drift[0]
	if d.Path != "AGENTS.md" || d.RecordedHash != hashOf("old content") || d.DiskHash != hashOf("user-modified content") || d.NewHash != hashOf("new content") {
		t.Errorf("drift entry = %+v", d)
	}
}

func TestPlanCoreReplace_CoreDeleteWhenTemplateRemovesFileAndDiskUnmodified(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "old-file.md", "gone content")
	tmpl := mapFS(map[string]string{}) // template no longer ships old-file.md
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("old-file.md", scaffold.OwnerCore, hashOf("gone content"), hashOf("gone content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	op := findOp(t, plan, "old-file.md")
	if op.Kind != OpDelete {
		t.Errorf("Kind = %v, want OpDelete", op.Kind)
	}
	if len(plan.Drift) != 0 {
		t.Errorf("unexpected drift: %+v", plan.Drift)
	}
	if len(plan.ManifestRemove) != 1 || plan.ManifestRemove[0] != "old-file.md" {
		t.Errorf("ManifestRemove = %+v, want [old-file.md] alongside the OpDelete", plan.ManifestRemove)
	}
}

// TestPlanCoreReplace_CoreManifestRemoveWhenDiskAlreadyAbsent proves the
// second template-removed-core case: disk already lacks the file (nothing to
// delete), but the manifest entry is still stale and must be signaled for
// removal so a caller doesn't accumulate manifest entries for paths that
// exist nowhere.
func TestPlanCoreReplace_CoreManifestRemoveWhenDiskAlreadyAbsent(t *testing.T) {
	dir := t.TempDir()
	tmpl := mapFS(map[string]string{}) // template no longer ships old-file.md
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("old-file.md", scaffold.OwnerCore, hashOf("gone content"), hashOf("gone content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "old-file.md")
	if len(plan.Drift) != 0 {
		t.Errorf("unexpected drift: %+v", plan.Drift)
	}
	if len(plan.ManifestRemove) != 1 || plan.ManifestRemove[0] != "old-file.md" {
		t.Errorf("ManifestRemove = %+v, want [old-file.md] even with no op (disk already absent)", plan.ManifestRemove)
	}
}

func TestPlanCoreReplace_CoreTemplateRemovalOfModifiedFileIsDriftNotDelete(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "old-file.md", "user-modified content")
	tmpl := mapFS(map[string]string{}) // template no longer ships old-file.md
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("old-file.md", scaffold.OwnerCore, hashOf("original content"), hashOf("original content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "old-file.md")
	if len(plan.Drift) != 1 {
		t.Fatalf("Drift = %+v, want 1 entry (never delete modified files)", plan.Drift)
	}
	d := plan.Drift[0]
	if d.Path != "old-file.md" || d.NewHash != "" {
		t.Errorf("drift entry = %+v, want empty NewHash (template no longer has file)", d)
	}
	if len(plan.ManifestRemove) != 0 {
		t.Errorf("ManifestRemove = %+v, want none for a drifted path (manifest must not be advanced)", plan.ManifestRemove)
	}
}

// --- owner=fork ---

func TestPlanCoreReplace_ForkSuppressedFromOpsAndCollectedAsAdvisory(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, ".claude/skills/custom/SKILL.md", "forked content")
	tmpl := mapFS(map[string]string{".claude/skills/custom/SKILL.md": "new core content"})
	m := scaffold.NewManifest("0.1.0")
	m.SetFileFork(".claude/skills/custom/SKILL.md", hashOf("forked content"), "0.3.0")

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, ".claude/skills/custom/SKILL.md")
	if len(plan.Drift) != 0 {
		t.Errorf("fork paths must never be classified as drift: %+v", plan.Drift)
	}
	if len(plan.Advisories) != 1 {
		t.Fatalf("Advisories = %+v, want 1 entry", plan.Advisories)
	}
	adv := plan.Advisories[0]
	if adv.Path != ".claude/skills/custom/SKILL.md" || adv.Owner != scaffold.OwnerFork {
		t.Errorf("advisory = %+v", adv)
	}
	if adv.DiskHash != hashOf("forked content") || adv.NewHash != hashOf("new core content") {
		t.Errorf("advisory hashes = %+v", adv)
	}
}

// --- owner=seed ---

func TestPlanCoreReplace_SeedCreateWhenDiskMissing(t *testing.T) {
	dir := t.TempDir()
	tmpl := mapFS(map[string]string{"ralph.toml": "seed content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("ralph.toml", scaffold.OwnerSeed, hashOf("seed content"), ""); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	op := findOp(t, plan, "ralph.toml")
	if op.Kind != OpCreate || string(op.Content) != "seed content" {
		t.Errorf("op = %+v, want OpCreate with seed content", op)
	}
}

func TestPlanCoreReplace_SeedAdvisoryWhenTemplateChangedAndDiskPresent(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "ralph.toml", "user-edited seed")
	tmpl := mapFS(map[string]string{"ralph.toml": "new seed content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("ralph.toml", scaffold.OwnerSeed, hashOf("old seed content"), hashOf("user-edited seed")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "ralph.toml") // seed is advisory-only, never auto-applied
	if len(plan.Advisories) != 1 {
		t.Fatalf("Advisories = %+v, want 1 entry", plan.Advisories)
	}
	adv := plan.Advisories[0]
	if adv.Path != "ralph.toml" || adv.Owner != scaffold.OwnerSeed {
		t.Errorf("advisory = %+v", adv)
	}
	if adv.DiskHash != hashOf("user-edited seed") || adv.NewHash != hashOf("new seed content") {
		t.Errorf("advisory hashes = %+v", adv)
	}
}

func TestPlanCoreReplace_SeedNoopWhenTemplateUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "ralph.toml", "whatever is on disk")
	tmpl := mapFS(map[string]string{"ralph.toml": "seed content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("ralph.toml", scaffold.OwnerSeed, hashOf("seed content"), hashOf("whatever is on disk")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "ralph.toml")
	if len(plan.Advisories) != 0 {
		t.Errorf("unexpected advisories: %+v", plan.Advisories)
	}
}

// --- owner=block / legacy ---

func TestPlanCoreReplace_BlockOwnerSkipped(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "AGENTS.md.block", "block disk content")
	tmpl := mapFS(map[string]string{"AGENTS.md.block": "block new content"})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("AGENTS.md.block", scaffold.OwnerBlock, hashOf("block old content"), hashOf("block disk content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "AGENTS.md.block")
	if len(plan.Drift) != 0 || len(plan.Advisories) != 0 {
		t.Errorf("block owner must produce no drift/advisory: drift=%+v advisories=%+v", plan.Drift, plan.Advisories)
	}
	if len(plan.LegacySkipped) != 1 || plan.LegacySkipped[0] != "AGENTS.md.block" {
		t.Errorf("LegacySkipped = %+v, want [AGENTS.md.block]", plan.LegacySkipped)
	}
}

func TestPlanCoreReplace_LegacyOwnerSkipped(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "AGENTS.md", "legacy disk content")
	tmpl := mapFS(map[string]string{"AGENTS.md": "legacy new content"})
	m := scaffold.NewManifest("0.1.0")
	// SetFile is the legacy (v1) setter: Owner stays unset.
	m.SetFile("AGENTS.md", hashOf("legacy old content"))

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	assertNoOpFor(t, plan, "AGENTS.md")
	if len(plan.Drift) != 0 || len(plan.Advisories) != 0 {
		t.Errorf("legacy owner must produce no drift/advisory: drift=%+v advisories=%+v", plan.Drift, plan.Advisories)
	}
	if len(plan.LegacySkipped) != 1 || plan.LegacySkipped[0] != "AGENTS.md" {
		t.Errorf("LegacySkipped = %+v, want [AGENTS.md]", plan.LegacySkipped)
	}
}

// --- untracked (no manifest entry) ---

func TestPlanCoreReplace_UntrackedTemplateFile(t *testing.T) {
	tmpl := mapFS(map[string]string{
		"create-me.md":  "content A",
		"refresh-me.md": "content B",
		"drift-me.md":   "content C new",
	})
	dir := t.TempDir()
	writeDiskFile(t, dir, "refresh-me.md", "content B") // already matches template
	writeDiskFile(t, dir, "drift-me.md", "content C different")
	m := scaffold.NewManifest("0.1.0") // no entries at all

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}

	createOp := findOp(t, plan, "create-me.md")
	if createOp.Kind != OpCreate || string(createOp.Content) != "content A" {
		t.Errorf("create-me.md op = %+v", createOp)
	}

	assertNoOpFor(t, plan, "refresh-me.md")
	if len(plan.ManifestRefresh) != 1 || plan.ManifestRefresh[0].Path != "refresh-me.md" || plan.ManifestRefresh[0].Hash != hashOf("content B") {
		t.Fatalf("ManifestRefresh = %+v, want single entry for refresh-me.md", plan.ManifestRefresh)
	}

	assertNoOpFor(t, plan, "drift-me.md")
	if len(plan.Drift) != 1 {
		t.Fatalf("Drift = %+v, want 1 entry", plan.Drift)
	}
	d := plan.Drift[0]
	if d.Path != "drift-me.md" || d.RecordedHash != "" || d.DiskHash != hashOf("content C different") || d.NewHash != hashOf("content C new") {
		t.Errorf("drift entry = %+v", d)
	}
}

// --- path validation (AC-9) ---

func TestPlanCoreReplace_RejectsNonLocalManifestPath(t *testing.T) {
	dir := t.TempDir()
	tmpl := mapFS(map[string]string{})
	m := scaffold.NewManifest("0.1.0")
	m.Files["../escape.md"] = scaffold.ManifestFile{Hash: "sha256:x", Managed: true, Owner: scaffold.OwnerCore}

	_, err := PlanCoreReplace(m, dir, tmpl)
	if err == nil {
		t.Fatal("expected error for manifest path containing \"..\"")
	}
}

func TestPlanCoreReplace_RejectsAbsoluteManifestPath(t *testing.T) {
	dir := t.TempDir()
	tmpl := mapFS(map[string]string{})
	m := scaffold.NewManifest("0.1.0")
	m.Files["/etc/passwd"] = scaffold.ManifestFile{Hash: "sha256:x", Managed: true, Owner: scaffold.OwnerCore}

	_, err := PlanCoreReplace(m, dir, tmpl)
	if err == nil {
		t.Fatal("expected error for absolute manifest path")
	}
}

// --- deterministic op ordering ---

func TestPlanCoreReplace_DeterministicOpOrdering(t *testing.T) {
	dir := t.TempDir()
	// Two deletes, two creates, two updates, deliberately unsorted by name.
	writeDiskFile(t, dir, "zzz-delete.md", "gone")
	writeDiskFile(t, dir, "aaa-delete.md", "gone")
	writeDiskFile(t, dir, "zzz-update.md", "old")
	writeDiskFile(t, dir, "aaa-update.md", "old")

	tmpl := mapFS(map[string]string{
		"zzz-create.md": "c1",
		"aaa-create.md": "c2",
		"zzz-update.md": "new",
		"aaa-update.md": "new",
	})

	m := scaffold.NewManifest("0.1.0")
	for _, p := range []string{"zzz-delete.md", "aaa-delete.md"} {
		if err := m.SetFileOwned(p, scaffold.OwnerCore, hashOf("gone"), hashOf("gone")); err != nil {
			t.Fatalf("SetFileOwned(%s): %v", p, err)
		}
	}
	for _, p := range []string{"zzz-update.md", "aaa-update.md"} {
		if err := m.SetFileOwned(p, scaffold.OwnerCore, hashOf("old"), hashOf("old")); err != nil {
			t.Fatalf("SetFileOwned(%s): %v", p, err)
		}
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	if len(plan.Ops) != 6 {
		t.Fatalf("Ops = %+v, want 6 entries", plan.Ops)
	}
	var gotKinds []OpKind
	var gotPaths []string
	for _, op := range plan.Ops {
		gotKinds = append(gotKinds, op.Kind)
		gotPaths = append(gotPaths, op.Path)
	}
	wantPaths := []string{
		"aaa-delete.md", "zzz-delete.md", // deletes, sorted
		"aaa-create.md", "zzz-create.md", // creates, sorted
		"aaa-update.md", "zzz-update.md", // updates, sorted
	}
	for i, p := range wantPaths {
		if gotPaths[i] != p {
			t.Errorf("Ops[%d].Path = %q, want %q (full order: %v)", i, gotPaths[i], p, gotPaths)
		}
	}
	wantKinds := []OpKind{OpDelete, OpDelete, OpCreate, OpCreate, OpUpdate, OpUpdate}
	for i, k := range wantKinds {
		if gotKinds[i] != k {
			t.Errorf("Ops[%d].Kind = %v, want %v (full order: %v)", i, gotKinds[i], k, gotKinds)
		}
	}
}

// --- ApplyOps ---

func TestApplyOps_WritesCreateAndUpdateAndDelete(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "to-delete.md", "bye")
	writeDiskFile(t, dir, "to-update.md", "old")

	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpDelete, Path: "to-delete.md"},
			{Kind: OpCreate, Path: "nested/new.md", Content: []byte("fresh")},
			{Kind: OpUpdate, Path: "to-update.md", Content: []byte("new")},
		},
	}
	if err := ApplyOps(dir, plan); err != nil {
		t.Fatalf("ApplyOps: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "to-delete.md")); !os.IsNotExist(err) {
		t.Errorf("to-delete.md should be removed, stat err = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "nested/new.md"))
	if err != nil || string(got) != "fresh" {
		t.Errorf("nested/new.md = %q, err %v, want %q", got, err, "fresh")
	}
	got, err = os.ReadFile(filepath.Join(dir, "to-update.md"))
	if err != nil || string(got) != "new" {
		t.Errorf("to-update.md = %q, err %v, want %q", got, err, "new")
	}
}

// TestApplyOps_PartialFailureStopsSubsequentOps proves the commit-barrier
// contract: when an op mid-plan fails, ApplyOps returns an error identifying
// the failed op and does not attempt ops that come after it.
//
// The failure is injected via chmod 0o444 on an existing regular file
// rather than a directory-in-the-way: since ApplyOps now Lstat-preflights
// every op target (a directory target fails preflight before any op
// executes, see TestApplyOps_LstatPreflightRejectsDirectoryTarget), a
// directory-collision failure would no longer land mid-plan and this test
// would no longer exercise the "ops before the failure already landed"
// case. chmod is root-defeated (root can write a 0o444 file), so this test
// skips under root instead of hard-failing there.
func TestApplyOps_PartialFailureStopsSubsequentOps(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0o444 does not deny writes to root; this failure-injection technique cannot run as root")
	}
	dir := t.TempDir()
	writeDiskFile(t, dir, "blocked.md", "old")
	blockedPath := filepath.Join(dir, "blocked.md")
	if err := os.Chmod(blockedPath, 0o444); err != nil {
		t.Fatalf("chmod blocked.md read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedPath, 0o644) })

	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpCreate, Path: "first.md", Content: []byte("first")},
			{Kind: OpUpdate, Path: "blocked.md", Content: []byte("should fail")},
			{Kind: OpCreate, Path: "third.md", Content: []byte("never written")},
		},
	}

	err := ApplyOps(dir, plan)
	if err == nil {
		t.Fatal("expected ApplyOps to return an error")
	}
	if !strings.Contains(err.Error(), "update") || !strings.Contains(err.Error(), "blocked.md") {
		t.Errorf("error %q does not identify the failed op", err.Error())
	}

	// The op before the failure was applied.
	if _, statErr := os.Stat(filepath.Join(dir, "first.md")); statErr != nil {
		t.Errorf("first.md should have been written before the failure: %v", statErr)
	}
	// The op after the failure was never attempted.
	if _, statErr := os.Stat(filepath.Join(dir, "third.md")); !os.IsNotExist(statErr) {
		t.Errorf("third.md should not exist (op after failure must not be attempted): stat err = %v", statErr)
	}
}

// TestPlanCoreReplace_ReplanAfterPartialFailureIsStable proves that
// re-planning over a tree left partially applied by a failed ApplyOps run
// completes the remaining work without treating the already-applied paths
// as drift, and without the manifest having advanced (the test manifest is
// reused unmodified across both planning calls, simulating "caller never
// reached the commit barrier").
//
// The failure is injected via chmod 0o444 rather than the directory-in-the-way
// technique used by TestApplyOps_PartialFailureStopsSubsequentOps: this test
// needs beta.md to plan as an OpUpdate (a regular file readable by
// PlanCoreReplace) both before and after the injected failure, and a
// directory at that path would make PlanCoreReplace itself error out on the
// unreadable path instead of reaching the two-op plan this test depends on.
// chmod is root-defeated (root can write a 0o444 file), so this test skips
// under root instead of hard-failing there.
func TestPlanCoreReplace_ReplanAfterPartialFailureIsStable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0o444 does not deny writes to root; this failure-injection technique cannot run as root")
	}
	dir := t.TempDir()
	writeDiskFile(t, dir, "alpha.md", "alpha old")
	writeDiskFile(t, dir, "beta.md", "beta old")

	tmpl := mapFS(map[string]string{
		"alpha.md": "alpha new",
		"beta.md":  "beta new",
	})
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("alpha.md", scaffold.OwnerCore, hashOf("alpha old"), hashOf("alpha old")); err != nil {
		t.Fatalf("SetFileOwned(alpha.md): %v", err)
	}
	if err := m.SetFileOwned("beta.md", scaffold.OwnerCore, hashOf("beta old"), hashOf("beta old")); err != nil {
		t.Fatalf("SetFileOwned(beta.md): %v", err)
	}

	plan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace (first): %v", err)
	}
	if len(plan.Ops) != 2 {
		t.Fatalf("expected 2 update ops, got %+v", plan.Ops)
	}

	// Revoke write permission on beta.md only, after planning but before
	// applying: alpha.md (sorted first) will be written successfully, then
	// beta.md's write fails, leaving its content untouched on disk (still
	// readable, so a subsequent plan can classify it normally instead of
	// erroring on an unreadable/unexpected path).
	betaPath := filepath.Join(dir, "beta.md")
	if err := os.Chmod(betaPath, 0o444); err != nil {
		t.Fatalf("chmod beta.md read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(betaPath, 0o644) })

	applyErr := ApplyOps(dir, plan)
	if applyErr == nil {
		t.Fatal("expected ApplyOps to fail writing beta.md (read-only)")
	}

	// alpha.md was written (it's ordered before beta.md alphabetically);
	// beta.md's write never happened.
	got, readErr := os.ReadFile(filepath.Join(dir, "alpha.md"))
	if readErr != nil || string(got) != "alpha new" {
		t.Fatalf("alpha.md = %q, err %v, want %q (should have been applied before the failure)", got, readErr, "alpha new")
	}
	got, readErr = os.ReadFile(betaPath)
	if readErr != nil || string(got) != "beta old" {
		t.Fatalf("beta.md = %q, err %v, want %q (write should have failed)", got, readErr, "beta old")
	}

	// Re-plan with the SAME manifest (caller never reached the commit
	// barrier, so recorded hashes are still stale for alpha.md).
	replan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace (replan): %v", err)
	}

	// alpha.md must not be re-planned as an update, and must not be
	// classified as drift — it should be recognized as already settled via
	// ManifestRefresh.
	assertNoOpFor(t, replan, "alpha.md")
	for _, d := range replan.Drift {
		if d.Path == "alpha.md" {
			t.Fatalf("alpha.md must not be classified as drift after a successful partial apply: %+v", d)
		}
	}
	foundRefresh := false
	for _, r := range replan.ManifestRefresh {
		if r.Path == "alpha.md" && r.Hash == hashOf("alpha new") {
			foundRefresh = true
		}
	}
	if !foundRefresh {
		t.Errorf("expected ManifestRefresh entry for alpha.md, got %+v", replan.ManifestRefresh)
	}

	// beta.md's write never happened (its content is still the old
	// content), so the remaining work (an update op) must still be present
	// to complete the interrupted run.
	betaOp := findOp(t, replan, "beta.md")
	if betaOp.Kind != OpUpdate {
		t.Errorf("beta.md op = %+v, want OpUpdate (remaining work must still be planned)", betaOp)
	}
}

// TestApplyOps_RejectsInvalidOpPathBeforeWritingAnything proves ApplyOps
// self-validates every op path before performing any filesystem operation:
// a hand-built plan (bypassing PlanCoreReplace) with a path that escapes
// targetDir must fail closed, writing nothing at all — not even the ops
// that would have succeeded and come before the invalid one in the list.
func TestApplyOps_RejectsInvalidOpPathBeforeWritingAnything(t *testing.T) {
	dir := t.TempDir()

	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpCreate, Path: "first.md", Content: []byte("should not be written")},
			{Kind: OpCreate, Path: "../outside/escape.md", Content: []byte("should not escape")},
		},
	}

	err := ApplyOps(dir, plan)
	if err == nil {
		t.Fatal("expected ApplyOps to reject a plan containing an invalid op path")
	}
	if !strings.Contains(err.Error(), "../outside/escape.md") {
		t.Errorf("error %q does not name the invalid path", err.Error())
	}

	if _, statErr := os.Stat(filepath.Join(dir, "first.md")); !os.IsNotExist(statErr) {
		t.Errorf("first.md should not have been written (validate-all-upfront): stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "outside", "escape.md")); !os.IsNotExist(statErr) {
		t.Errorf("escape.md should not exist outside targetDir: stat err = %v", statErr)
	}
}

// --- desired-state map input (PlanCoreReplaceDesired) ---

// TestPlanCoreReplaceDesired_EquivalentToFSAdapter proves PlanCoreReplace is
// a thin adapter: walking the same content through an fs.FS and through a
// direct path→content map must produce byte-for-byte identical plans.
func TestPlanCoreReplaceDesired_EquivalentToFSAdapter(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "existing.md", "old content")

	files := map[string]string{
		"existing.md":  "new content",
		"brand-new.md": "brand new content",
	}
	tmpl := mapFS(files)
	desired := make(map[string][]byte, len(files))
	for path, content := range files {
		desired[path] = []byte(content)
	}

	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("existing.md", scaffold.OwnerCore, hashOf("old content"), hashOf("old content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	fsPlan, err := PlanCoreReplace(m, dir, tmpl)
	if err != nil {
		t.Fatalf("PlanCoreReplace: %v", err)
	}
	mapPlan, err := PlanCoreReplaceDesired(m, dir, desired, ReplaceOptions{})
	if err != nil {
		t.Fatalf("PlanCoreReplaceDesired: %v", err)
	}
	if !reflect.DeepEqual(fsPlan, mapPlan) {
		t.Errorf("desired-map plan differs from fs.FS adapter plan:\nfs:  %+v\nmap: %+v", fsPlan, mapPlan)
	}
}

// TestPlanCoreReplaceDesired_SkipPaths_ExcludedFromEveryList proves a path
// listed in ReplaceOptions.SkipPaths never appears in any output list (Ops,
// ManifestRefresh, Drift, Advisories, LegacySkipped, ManifestRemove,
// Preserved), even when template, manifest, and disk are all present and
// would otherwise classify normally (here: disk content differs from both
// the recorded and template hashes, which would ordinarily be Drift).
func TestPlanCoreReplaceDesired_SkipPaths_ExcludedFromEveryList(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, ".claude/settings.json", "disk content")

	desired := map[string][]byte{".claude/settings.json": []byte("template content")}
	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned(".claude/settings.json", scaffold.OwnerCore, hashOf("recorded content"), hashOf("recorded content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplaceDesired(m, dir, desired, ReplaceOptions{
		SkipPaths: map[string]bool{".claude/settings.json": true},
	})
	if err != nil {
		t.Fatalf("PlanCoreReplaceDesired: %v", err)
	}

	assertNoOpFor(t, plan, ".claude/settings.json")
	for _, r := range plan.ManifestRefresh {
		if r.Path == ".claude/settings.json" {
			t.Errorf("skipped path must not appear in ManifestRefresh: %+v", r)
		}
	}
	for _, d := range plan.Drift {
		if d.Path == ".claude/settings.json" {
			t.Errorf("skipped path must not appear in Drift: %+v", d)
		}
	}
	for _, a := range plan.Advisories {
		if a.Path == ".claude/settings.json" {
			t.Errorf("skipped path must not appear in Advisories: %+v", a)
		}
	}
	for _, p := range plan.LegacySkipped {
		if p == ".claude/settings.json" {
			t.Error("skipped path must not appear in LegacySkipped")
		}
	}
	for _, p := range plan.ManifestRemove {
		if p == ".claude/settings.json" {
			t.Error("skipped path must not appear in ManifestRemove")
		}
	}
	for _, p := range plan.Preserved {
		if p == ".claude/settings.json" {
			t.Error("skipped path must not appear in Preserved")
		}
	}
}

// TestPlanCoreReplaceDesired_PreservePrefixes_ManifestEntryWithNoTemplate
// proves a manifest-tracked owner=core path under a preserve prefix whose
// desired-state map has no content for it (e.g. an uninstalled language
// pack) produces no delete op, no ManifestRemove entry, and no Drift entry.
// The path is reported in ReplacePlan.Preserved instead, and ApplyOps
// leaves its disk content untouched (AC-9).
func TestPlanCoreReplaceDesired_PreservePrefixes_ManifestEntryWithNoTemplate(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "packs/languages/golang/rule.md", "golang rule content")

	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("packs/languages/golang/rule.md", scaffold.OwnerCore, hashOf("golang rule content"), hashOf("golang rule content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	plan, err := PlanCoreReplaceDesired(m, dir, map[string][]byte{}, ReplaceOptions{
		PreservePrefixes: []string{"packs/languages/golang/"},
	})
	if err != nil {
		t.Fatalf("PlanCoreReplaceDesired: %v", err)
	}

	assertNoOpFor(t, plan, "packs/languages/golang/rule.md")
	for _, d := range plan.Drift {
		if d.Path == "packs/languages/golang/rule.md" {
			t.Errorf("preserved path must not be classified as drift: %+v", d)
		}
	}
	for _, p := range plan.ManifestRemove {
		if p == "packs/languages/golang/rule.md" {
			t.Error("preserved path must not appear in ManifestRemove")
		}
	}
	if len(plan.Preserved) != 1 || plan.Preserved[0] != "packs/languages/golang/rule.md" {
		t.Errorf("Preserved = %+v, want [packs/languages/golang/rule.md]", plan.Preserved)
	}

	if err := ApplyOps(dir, plan); err != nil {
		t.Fatalf("ApplyOps: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "packs", "languages", "golang", "rule.md"))
	if err != nil || string(got) != "golang rule content" {
		t.Errorf("preserved file must be untouched by ApplyOps: got %q, err %v", got, err)
	}
}

// TestPlanCoreReplaceDesired_PreservePrefixes_TemplatePresentClassifiesNormally
// proves the preserve mechanism only suppresses the "template absent"
// outcome: a path under a preserve prefix that DOES have desired-state
// content (the pack is still installed) classifies exactly as it would
// without the prefix.
func TestPlanCoreReplaceDesired_PreservePrefixes_TemplatePresentClassifiesNormally(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "packs/languages/golang/rule.md", "old content")

	m := scaffold.NewManifest("0.1.0")
	if err := m.SetFileOwned("packs/languages/golang/rule.md", scaffold.OwnerCore, hashOf("old content"), hashOf("old content")); err != nil {
		t.Fatalf("SetFileOwned: %v", err)
	}

	desired := map[string][]byte{"packs/languages/golang/rule.md": []byte("new content")}
	plan, err := PlanCoreReplaceDesired(m, dir, desired, ReplaceOptions{
		PreservePrefixes: []string{"packs/languages/golang/"},
	})
	if err != nil {
		t.Fatalf("PlanCoreReplaceDesired: %v", err)
	}

	op := findOp(t, plan, "packs/languages/golang/rule.md")
	if op.Kind != OpUpdate {
		t.Errorf("Kind = %v, want OpUpdate (an actively desired path under a preserve prefix must classify normally)", op.Kind)
	}
	if len(plan.Preserved) != 0 {
		t.Errorf("Preserved = %+v, want empty (path had desired-state content)", plan.Preserved)
	}
}

// --- ApplyOps Lstat preflight (AC-4b) ---

// TestApplyOps_LstatPreflightRejectsSymlinkTarget_ZeroWrites proves the
// Lstat preflight runs before any op executes: a symlinked op target
// anywhere in the plan makes ApplyOps fail with zero filesystem changes,
// including ops ordered before it that would otherwise have succeeded.
func TestApplyOps_LstatPreflightRejectsSymlinkTarget_ZeroWrites(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "external.md", "outside content")
	symlinkPath := filepath.Join(dir, "linked.md")
	if err := os.Symlink(filepath.Join(dir, "external.md"), symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpCreate, Path: "first.md", Content: []byte("would succeed")},
			{Kind: OpUpdate, Path: "linked.md", Content: []byte("should not be written")},
		},
	}

	err := ApplyOps(dir, plan)
	if err == nil {
		t.Fatal("expected ApplyOps to reject a symlinked op target")
	}
	if !strings.Contains(err.Error(), "linked.md") {
		t.Errorf("error %q does not name the symlinked path", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(dir, "first.md")); !os.IsNotExist(statErr) {
		t.Errorf("first.md should not have been written (preflight runs before any op executes): stat err = %v", statErr)
	}
	linkInfo, lerr := os.Lstat(symlinkPath)
	if lerr != nil {
		t.Fatalf("Lstat(linked.md): %v", lerr)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Error("linked.md should still be a symlink")
	}
	got, rerr := os.ReadFile(filepath.Join(dir, "external.md"))
	if rerr != nil || string(got) != "outside content" {
		t.Errorf("external.md = %q, err %v, want unchanged %q", got, rerr, "outside content")
	}
}

// TestApplyOps_LstatPreflightRejectsSymlinkDeleteTarget proves an OpDelete
// targeting a symlink is also rejected by the preflight rather than being
// followed or unlinked.
func TestApplyOps_LstatPreflightRejectsSymlinkDeleteTarget(t *testing.T) {
	dir := t.TempDir()
	writeDiskFile(t, dir, "external.md", "outside content")
	symlinkPath := filepath.Join(dir, "linked.md")
	if err := os.Symlink(filepath.Join(dir, "external.md"), symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpDelete, Path: "linked.md"},
		},
	}

	err := ApplyOps(dir, plan)
	if err == nil {
		t.Fatal("expected ApplyOps to reject a symlinked delete target")
	}
	if _, lerr := os.Lstat(symlinkPath); lerr != nil {
		t.Errorf("linked.md should still exist as a symlink: %v", lerr)
	}
}

// TestApplyOps_LstatPreflightRejectsDirectoryTarget proves a directory
// occupying an op's target path is rejected by the preflight before any
// write is attempted.
func TestApplyOps_LstatPreflightRejectsDirectoryTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "blocked.md"), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpCreate, Path: "blocked.md", Content: []byte("should not be written")},
		},
	}

	err := ApplyOps(dir, plan)
	if err == nil {
		t.Fatal("expected ApplyOps to reject a directory op target")
	}
	if !strings.Contains(err.Error(), "blocked.md") {
		t.Errorf("error %q does not name the directory path", err.Error())
	}
	info, statErr := os.Stat(filepath.Join(dir, "blocked.md"))
	if statErr != nil || !info.IsDir() {
		t.Errorf("blocked.md should still be a directory: info=%+v err=%v", info, statErr)
	}
}

// TestApplyOps_LstatPreflightRejectsDanglingSymlinkCreateTarget proves a
// dangling symlink at a create target is rejected: os.Lstat succeeds on a
// dangling symlink (it reports the symlink's own mode without following
// it), so the preflight must not mistake it for "missing" and proceed to
// overwrite through the link.
func TestApplyOps_LstatPreflightRejectsDanglingSymlinkCreateTarget(t *testing.T) {
	dir := t.TempDir()
	symlinkPath := filepath.Join(dir, "dangling.md")
	if err := os.Symlink(filepath.Join(dir, "does-not-exist.md"), symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpCreate, Path: "dangling.md", Content: []byte("should not be written")},
		},
	}

	err := ApplyOps(dir, plan)
	if err == nil {
		t.Fatal("expected ApplyOps to reject a dangling-symlink create target")
	}
	linkInfo, lerr := os.Lstat(symlinkPath)
	if lerr != nil {
		t.Fatalf("Lstat(dangling.md): %v", lerr)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Error("dangling.md should still be a symlink")
	}
}

// TestApplyOps_LstatPreflightAllowsMissingCreateTarget proves the ordinary
// create case (no entry at the target path at all) still succeeds: the
// preflight must treat os.ErrNotExist as "fine", not as a rejection.
func TestApplyOps_LstatPreflightAllowsMissingCreateTarget(t *testing.T) {
	dir := t.TempDir()
	plan := ReplacePlan{
		Ops: []FileOp{
			{Kind: OpCreate, Path: "new.md", Content: []byte("fresh")},
		},
	}
	if err := ApplyOps(dir, plan); err != nil {
		t.Fatalf("ApplyOps: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "new.md"))
	if err != nil || string(got) != "fresh" {
		t.Errorf("new.md = %q, err %v, want %q", got, err, "fresh")
	}
}
