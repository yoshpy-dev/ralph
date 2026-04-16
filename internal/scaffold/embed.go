package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
)

// EmbeddedFS holds the embedded template filesystem.
// It is set from cmd/ralph/main.go where the go:embed directive lives
// (embed directives can only reference files in the same package's directory tree).
var EmbeddedFS embed.FS

// BaseFS returns the filesystem rooted at templates/base/.
func BaseFS() (fs.FS, error) {
	return fs.Sub(EmbeddedFS, "templates/base")
}

// PackFS returns the filesystem for a specific language pack.
// The lang parameter is validated against the known pack list.
func PackFS(lang string) (fs.FS, error) {
	packs, err := AvailablePacks()
	if err != nil {
		return nil, err
	}
	if !slices.Contains(packs, lang) {
		return nil, fmt.Errorf("unknown language pack: %q", lang)
	}
	return fs.Sub(EmbeddedFS, "templates/packs/"+lang)
}

// PromptsFS returns the filesystem rooted at templates/prompts/.
func PromptsFS() (fs.FS, error) {
	return fs.Sub(EmbeddedFS, "templates/prompts")
}

// AvailablePacks lists all language packs in templates/packs/.
func AvailablePacks() ([]string, error) {
	entries, err := fs.ReadDir(EmbeddedFS, "templates/packs")
	if err != nil {
		return nil, err
	}
	var packs []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "_template" {
			packs = append(packs, e.Name())
		}
	}
	return packs, nil
}
