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
	Hash    string `toml:"hash"`
	Managed bool   `toml:"managed"`
}

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

// SetFile records a file in the manifest.
func (m *Manifest) SetFile(relPath, hash string) {
	m.Files[relPath] = ManifestFile{
		Hash:    hash,
		Managed: true,
	}
}
