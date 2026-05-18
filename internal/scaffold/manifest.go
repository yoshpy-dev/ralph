package scaffold

import (
	"os"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// Manifest tracks which files ralph manages in a project.
type Manifest struct {
	Meta  ManifestMeta            `toml:"meta"`
	Files map[string]ManifestFile `toml:"files"`
}

// ManifestMeta holds manifest-level metadata.
type ManifestMeta struct {
	Version string   `toml:"version"`
	Created string   `toml:"created"`
	Updated string   `toml:"updated"`
	Packs   []string `toml:"packs,omitempty"`
}

// ManifestFile tracks a single managed file.
type ManifestFile struct {
	// v1 compatibility fields. Keep writing these until every supported
	// consumer understands the v2 metadata below.
	Hash    string `toml:"hash"`
	Managed bool   `toml:"managed"`

	// v2 metadata. These fields are additive and must not be required when
	// reading existing manifests.
	State          string `toml:"state,omitempty"`
	TemplateHash   string `toml:"template_hash,omitempty"`
	DiskHash       string `toml:"disk_hash,omitempty"`
	BaselineStatus string `toml:"baseline_status,omitempty"`
	BaselinePath   string `toml:"baseline_path,omitempty"`
}

const (
	FileStateManaged   = "managed"
	FileStateUnmanaged = "unmanaged"
	FileStatePartial   = "partial"

	BaselineStatusMissing   = "missing"
	BaselineStatusAvailable = "available"
)

// NewManifest creates a new manifest with the given version.
func NewManifest(version string) *Manifest {
	now := time.Now().UTC().Format(time.RFC3339)
	return &Manifest{
		Meta: ManifestMeta{
			Version: version,
			Created: now,
			Updated: now,
		},
		Files: make(map[string]ManifestFile),
	}
}

// ReadManifest reads a manifest from a TOML file.
func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Files == nil {
		m.Files = make(map[string]ManifestFile)
	}
	return &m, nil
}

// Write saves the manifest to the given path.
func (m *Manifest) Write(path string) error {
	m.Meta.Updated = time.Now().UTC().Format(time.RFC3339)
	data, err := toml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SetFile records a file in the manifest as managed (i.e. tracked against the
// template for future upgrades).
func (m *Manifest) SetFile(relPath, hash string) {
	m.Files[relPath] = ManifestFile{
		Hash:           hash,
		Managed:        true,
		State:          FileStateManaged,
		TemplateHash:   hash,
		BaselineStatus: BaselineStatusMissing,
	}
}

// SetFileWithBaseline records a managed file whose template content is
// available in the local baseline cache. The legacy Hash field remains the
// template hash so v1 readers continue to compare the same value.
func (m *Manifest) SetFileWithBaseline(relPath, hash, baselinePath string) {
	m.Files[relPath] = ManifestFile{
		Hash:           hash,
		Managed:        true,
		State:          FileStateManaged,
		TemplateHash:   hash,
		BaselineStatus: BaselineStatusAvailable,
		BaselinePath:   baselinePath,
	}
}

// SetFileResolvedWithBaseline records a managed file resolved from template
// metadata and local choices. Hash remains the template hash for v1-compatible
// upgrade comparisons; DiskHash records the actual resolved content.
func (m *Manifest) SetFileResolvedWithBaseline(relPath, templateHash, diskHash, state, baselinePath string) {
	m.Files[relPath] = ManifestFile{
		Hash:           templateHash,
		Managed:        true,
		State:          state,
		TemplateHash:   templateHash,
		DiskHash:       diskHash,
		BaselineStatus: BaselineStatusAvailable,
		BaselinePath:   baselinePath,
	}
}

// SetFileUnmanaged records a file in the manifest as user-owned. The ralph
// upgrade flow uses this when the user chooses to keep a local variant over
// the template: the entry is preserved (so the path is not mistaken for a
// "new file" on later upgrades) but marked so future runs skip it silently
// instead of re-prompting.
func (m *Manifest) SetFileUnmanaged(relPath, hash string) {
	m.Files[relPath] = ManifestFile{
		Hash:           hash,
		Managed:        false,
		State:          FileStateUnmanaged,
		DiskHash:       hash,
		BaselineStatus: BaselineStatusMissing,
	}
}

// IsBaselineAvailable reports whether the manifest entry has usable baseline
// metadata. Callers must still verify that BaselinePath exists before using it.
func (f ManifestFile) IsBaselineAvailable() bool {
	return f.BaselineStatus == BaselineStatusAvailable && f.BaselinePath != ""
}

// WithTemplateHash returns a copy with v2 template metadata filled in while
// preserving any existing baseline metadata.
func (f ManifestFile) WithTemplateHash(hash string) ManifestFile {
	f.Hash = hash
	f.Managed = true
	if f.State == "" {
		f.State = FileStateManaged
	}
	if f.TemplateHash == "" {
		f.TemplateHash = hash
	}
	if f.BaselineStatus == "" {
		f.BaselineStatus = BaselineStatusMissing
	}
	return f
}
