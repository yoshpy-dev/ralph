package scaffold

import (
	"embed"
	"io/fs"
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
func PackFS(lang string) (fs.FS, error) {
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
