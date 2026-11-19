package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
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

	// v3 metadata. Additive; absent on v1/v2 manifests and on manifests
	// written through the legacy (non-opt-in) constructors/setters below.
	Layout string `toml:"layout,omitempty"`
}

// ManifestFile tracks a single managed file.
type ManifestFile struct {
	// v1 compatibility fields. Keep writing these until every supported
	// consumer understands the v2 metadata below.
	Hash    string `toml:"hash"`
	Managed bool   `toml:"managed"`

	// v2 metadata. These fields are additive and must not be required when
	// reading existing manifests.
	State        string `toml:"state,omitempty"`
	TemplateHash string `toml:"template_hash,omitempty"`
	DiskHash     string `toml:"disk_hash,omitempty"`

	// BaselineStatus and BaselinePath are legacy-read-only: the baseline
	// mechanism (internal/scaffold/baseline.go) was removed in Phase 3
	// (docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md, FR-13). These
	// fields are kept solely so manifests written by pre-Phase-3 `ralph`
	// versions (State may be "partial", BaselineStatus may be "available")
	// still parse without error; nothing in the current codebase writes
	// those values anymore, so no named constant exists for them.
	BaselineStatus string `toml:"baseline_status,omitempty"`
	BaselinePath   string `toml:"baseline_path,omitempty"`

	// v3 metadata (ownership). Additive; absent (legacy/unset) unless
	// written through the opt-in v3 setters below. Reading a v1/v2 manifest
	// must never assign these fields.
	Owner             string `toml:"owner,omitempty"`
	ForkedFromVersion string `toml:"forked_from_version,omitempty"`
}

const (
	FileStateManaged   = "managed"
	FileStateUnmanaged = "unmanaged"

	BaselineStatusMissing = "missing"

	// LayoutV2 identifies the overlay-scaffold layout (manifest v3, ownership
	// attributes). Set only through the opt-in SetLayoutV2 API.
	LayoutV2 = "v2"

	// Ownership attributes for manifest v3 entries. See docs/specs
	// 2026-08-17-overlay-scaffold-v2.md, section "層モデル".
	OwnerCore  = "core"
	OwnerFork  = "fork"
	OwnerSeed  = "seed"
	OwnerBlock = "block"
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

// renameManifestFile is os.Rename. Tests replace it to make the last step of
// Write fail.
var renameManifestFile = os.Rename

// Write saves the manifest to path and replaces the file there atomically:
// it writes the TOML to a temporary file in path's directory and renames it
// over path, so a reader sees the old content or the new, never a part.
// When any step fails, Write removes the temporary file and leaves the file
// at path as it was. The directory of path must exist; Write does not create
// it. The new file's mode is 0644, and a symlink at path is replaced by a
// regular file, not followed.
func (m *Manifest) Write(path string) error {
	m.Meta.Updated = time.Now().UTC().Format(time.RFC3339)
	data, err := toml.Marshal(m)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = renameManifestFile(tmpPath, path)
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
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

// SetLayoutV2 marks the manifest as belonging to the overlay-scaffold (v2)
// layout. Opt-in only: existing constructors/setters never set this.
func (m *Manifest) SetLayoutV2() {
	m.Meta.Layout = LayoutV2
}

// SetFileOwned records a manifest v3 entry with an explicit ownership
// attribute (core/seed/block). Fork entries are rejected: SetFileFork is
// the single way to record a fork, because a fork additionally needs
// ForkedFromVersion and Managed=false semantics.
func (m *Manifest) SetFileOwned(relPath, owner, templateHash, diskHash string) error {
	if err := validateManagedOwner(owner); err != nil {
		return err
	}
	m.Files[relPath] = ManifestFile{
		Hash:           templateHash,
		Managed:        true,
		State:          FileStateManaged,
		TemplateHash:   templateHash,
		DiskHash:       diskHash,
		BaselineStatus: BaselineStatusMissing,
		Owner:          owner,
	}
	return nil
}

// SetFileFork records a manifest v3 entry for a file the user has ejected
// from core ownership into a fork. Managed is false: upgrade must not
// replace fork content.
func (m *Manifest) SetFileFork(relPath, diskHash, forkedFromVersion string) {
	m.Files[relPath] = ManifestFile{
		Hash:              diskHash,
		Managed:           false,
		State:             FileStateUnmanaged,
		DiskHash:          diskHash,
		BaselineStatus:    BaselineStatusMissing,
		Owner:             OwnerFork,
		ForkedFromVersion: forkedFromVersion,
	}
}

// SetOwner mutates an existing manifest entry's Owner attribute in place.
// Only core/seed/block are accepted here — a fork must be recorded via
// SetFileFork, since a fork additionally needs ForkedFromVersion and
// Managed=false semantics that SetOwner does not touch. All other fields on
// the existing entry (Hash, Managed, State, TemplateHash, DiskHash,
// BaselineStatus, BaselinePath) are left untouched.
func (m *Manifest) SetOwner(relPath, owner string) error {
	if err := validateManagedOwner(owner); err != nil {
		return err
	}
	entry, ok := m.Files[relPath]
	if !ok {
		return fmt.Errorf("no manifest entry for %q", relPath)
	}
	entry.Owner = owner
	m.Files[relPath] = entry
	return nil
}

func validateManagedOwner(owner string) error {
	switch owner {
	case OwnerCore, OwnerSeed, OwnerBlock:
		return nil
	case OwnerFork:
		return fmt.Errorf("owner %q must be recorded via SetFileFork", owner)
	default:
		return fmt.Errorf("unknown owner %q", owner)
	}
}

// IsLegacyOwner reports whether the entry predates manifest v3 ownership
// attribution (owner unset). Reading a v1/v2 manifest must never assign
// Owner, so this is the correct way to detect "no ownership decided yet".
func (f ManifestFile) IsLegacyOwner() bool {
	return f.Owner == ""
}
