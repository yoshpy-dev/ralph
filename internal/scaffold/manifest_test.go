package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")

	m := NewManifest("0.1.0")
	m.SetFile("AGENTS.md", "sha256:abc123")
	m.SetFile(".claude/rules/ralph/testing.md", "sha256:def456")

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
	if f, ok := got.Files[".claude/rules/ralph/testing.md"]; !ok || !f.Managed {
		t.Errorf(".claude/rules/ralph/testing.md managed = %v, want true", f.Managed)
	}
}

// TestManifestRoundTripLegacyBaselineFields pins read compatibility for the
// baseline_status/baseline_path TOML fields written by pre-Phase-3 `ralph`
// versions (internal/scaffold/baseline.go, removed in Phase 3 — see
// docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md). The current
// codebase never writes these fields (no setter constructs them anymore), so
// the entries below are built directly rather than through a setter, purely
// to prove ReadManifest/Write still round-trip a manifest a legacy `ralph`
// left behind.
func TestManifestRoundTripLegacyBaselineFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")

	m := NewManifest("0.3.0")
	m.Files["AGENTS.md"] = ManifestFile{
		Hash:           "sha256:abc123",
		Managed:        true,
		State:          FileStateManaged,
		TemplateHash:   "sha256:abc123",
		BaselineStatus: "available",
		BaselinePath:   ".ralph/baseline/AGENTS.md",
	}
	m.Files["partial.md"] = ManifestFile{
		Hash:           "sha256:template",
		Managed:        true,
		State:          "partial", // legacy pre-Phase-3 State value; no constant, see manifest.go
		TemplateHash:   "sha256:template",
		DiskHash:       "sha256:disk",
		BaselineStatus: "available",
		BaselinePath:   ".ralph/baseline/partial.md",
	}
	m.Files["CLAUDE.md"] = ManifestFile{
		Hash:           "sha256:local",
		Managed:        false,
		State:          FileStateUnmanaged,
		DiskHash:       "sha256:local",
		BaselineStatus: BaselineStatusMissing,
	}

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
	if agents.BaselineStatus != "available" {
		t.Errorf("BaselineStatus = %q, want available", agents.BaselineStatus)
	}
	if agents.BaselinePath != ".ralph/baseline/AGENTS.md" {
		t.Errorf("BaselinePath = %q", agents.BaselinePath)
	}

	partial := got.Files["partial.md"]
	if !partial.Managed || partial.State != "partial" {
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
}

func TestManifestRoundTripV3OwnershipFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")

	m := NewManifest("0.4.0")
	m.SetLayoutV2()
	if err := m.SetFileOwned("AGENTS.md", OwnerCore, "sha256:core", "sha256:coredisk"); err != nil {
		t.Fatalf("SetFileOwned(core): %v", err)
	}
	if err := m.SetFileOwned("ralph.toml", OwnerSeed, "sha256:seed", "sha256:seeddisk"); err != nil {
		t.Fatalf("SetFileOwned(seed): %v", err)
	}
	if err := m.SetFileOwned("AGENTS.md.block", OwnerBlock, "sha256:block", "sha256:blockdisk"); err != nil {
		t.Fatalf("SetFileOwned(block): %v", err)
	}
	m.SetFileFork(".claude/skills/custom/SKILL.md", "sha256:forkdisk", "0.3.0")

	if err := m.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if got.Meta.Layout != LayoutV2 {
		t.Errorf("Meta.Layout = %q, want %q", got.Meta.Layout, LayoutV2)
	}

	core := got.Files["AGENTS.md"]
	if core.Owner != OwnerCore || !core.Managed || core.IsLegacyOwner() {
		t.Errorf("core entry = %+v, want owner=core managed=true", core)
	}

	seed := got.Files["ralph.toml"]
	if seed.Owner != OwnerSeed || !seed.Managed {
		t.Errorf("seed entry = %+v, want owner=seed managed=true", seed)
	}

	block := got.Files["AGENTS.md.block"]
	if block.Owner != OwnerBlock || !block.Managed {
		t.Errorf("block entry = %+v, want owner=block managed=true", block)
	}

	fork := got.Files[".claude/skills/custom/SKILL.md"]
	if fork.Owner != OwnerFork || fork.Managed || fork.ForkedFromVersion != "0.3.0" {
		t.Errorf("fork entry = %+v, want owner=fork managed=false forked_from_version=0.3.0", fork)
	}
	if fork.IsLegacyOwner() {
		t.Error("fork entry must not report IsLegacyOwner")
	}
}

func TestSetFileOwned_RejectsUnknownOwner(t *testing.T) {
	m := NewManifest("0.4.0")
	if err := m.SetFileOwned("x.md", "not-a-real-owner", "sha256:a", "sha256:b"); err == nil {
		t.Fatal("SetFileOwned accepted unknown owner")
	}
	if _, ok := m.Files["x.md"]; ok {
		t.Error("SetFileOwned must not record an entry when owner is invalid")
	}
}

func TestSetFileOwned_RejectsFork(t *testing.T) {
	m := NewManifest("0.4.0")
	if err := m.SetFileOwned("x.md", OwnerFork, "sha256:a", "sha256:b"); err == nil {
		t.Fatal("SetFileOwned accepted OwnerFork; forks must go through SetFileFork")
	}
	if _, ok := m.Files["x.md"]; ok {
		t.Error("SetFileOwned must not record an entry for OwnerFork")
	}
}

func TestReadManifestV1V2_OwnerStaysLegacy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.toml")
	data := []byte(`[meta]
version = "0.2.0"
created = "2026-05-18T00:00:00Z"
updated = "2026-05-18T00:00:00Z"

[files."AGENTS.md"]
hash = "sha256:abc123"
managed = true
state = "managed"
template_hash = "sha256:abc123"
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got.Meta.Layout != "" {
		t.Errorf("Meta.Layout = %q, want empty for legacy manifest", got.Meta.Layout)
	}
	entry := got.Files["AGENTS.md"]
	if !entry.IsLegacyOwner() {
		t.Fatal("v1/v2 manifest entry must be legacy owner (unset)")
	}
	if entry.Owner != "" || entry.ForkedFromVersion != "" {
		t.Fatalf("reading must not assign ownership: %+v", entry)
	}
}

// TestExistingConstructorsWriteNoV3Fields is AC-8: manifests written through
// the existing (non-opt-in) constructors/setters must contain no v3 fields
// (layout, owner, forked_from_version) in the marshaled TOML bytes.
func TestExistingConstructorsWriteNoV3Fields(t *testing.T) {
	m := NewManifest("0.3.0")
	m.SetFile("AGENTS.md", "sha256:a")
	m.Files["CLAUDE.md"] = ManifestFile{
		Hash:           "sha256:b",
		Managed:        true,
		State:          FileStateManaged,
		TemplateHash:   "sha256:b",
		BaselineStatus: "available",
		BaselinePath:   ".ralph/baseline/CLAUDE.md",
	}
	m.Files["partial.md"] = ManifestFile{
		Hash:           "sha256:c",
		Managed:        true,
		State:          "partial",
		TemplateHash:   "sha256:c",
		DiskHash:       "sha256:d",
		BaselineStatus: "available",
		BaselinePath:   ".ralph/baseline/partial.md",
	}
	m.Files["local.md"] = ManifestFile{
		Hash:           "sha256:e",
		Managed:        false,
		State:          FileStateUnmanaged,
		DiskHash:       "sha256:e",
		BaselineStatus: BaselineStatusMissing,
	}

	data, err := toml.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)
	for _, forbidden := range []string{"layout", "owner", "forked_from_version"} {
		if strings.Contains(s, forbidden) {
			t.Errorf("marshaled manifest from legacy setters contains v3 field %q:\n%s", forbidden, s)
		}
	}
}

func TestSetOwner_SetsOnExistingEntry(t *testing.T) {
	m := NewManifest("0.4.0")
	m.Files["AGENTS.md"] = ManifestFile{
		Hash:           "sha256:tmpl",
		Managed:        true,
		State:          FileStateManaged,
		TemplateHash:   "sha256:tmpl",
		BaselineStatus: "available",
		BaselinePath:   ".ralph/baseline/AGENTS.md",
	}

	if err := m.SetOwner("AGENTS.md", OwnerBlock); err != nil {
		t.Fatalf("SetOwner: %v", err)
	}

	entry := m.Files["AGENTS.md"]
	if entry.Owner != OwnerBlock {
		t.Errorf("Owner = %q, want %q", entry.Owner, OwnerBlock)
	}
	// Other v1/v2 fields must be untouched.
	if entry.Hash != "sha256:tmpl" || entry.TemplateHash != "sha256:tmpl" {
		t.Errorf("Hash/TemplateHash changed unexpectedly: %+v", entry)
	}
	if !entry.Managed || entry.State != FileStateManaged {
		t.Errorf("Managed/State changed unexpectedly: %+v", entry)
	}
	if entry.BaselineStatus != "available" || entry.BaselinePath != ".ralph/baseline/AGENTS.md" {
		t.Errorf("baseline fields changed unexpectedly: %+v", entry)
	}
}

func TestSetOwner_RejectsFork(t *testing.T) {
	m := NewManifest("0.4.0")
	m.SetFile("x.md", "sha256:a")

	if err := m.SetOwner("x.md", OwnerFork); err == nil {
		t.Fatal("SetOwner accepted OwnerFork; forks must go through SetFileFork")
	}
	if got := m.Files["x.md"].Owner; got != "" {
		t.Errorf("Owner = %q after rejected SetOwner, want unchanged empty", got)
	}
}

func TestSetOwner_RejectsUnknownOwner(t *testing.T) {
	m := NewManifest("0.4.0")
	m.SetFile("x.md", "sha256:a")

	if err := m.SetOwner("x.md", "not-a-real-owner"); err == nil {
		t.Fatal("SetOwner accepted unknown owner")
	}
	if got := m.Files["x.md"].Owner; got != "" {
		t.Errorf("Owner = %q after rejected SetOwner, want unchanged empty", got)
	}
}

func TestSetOwner_RejectsMissingEntry(t *testing.T) {
	m := NewManifest("0.4.0")

	if err := m.SetOwner("missing.md", OwnerCore); err == nil {
		t.Fatal("SetOwner accepted a path with no existing manifest entry")
	}
	if _, ok := m.Files["missing.md"]; ok {
		t.Error("SetOwner must not create a new entry for a missing path")
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
