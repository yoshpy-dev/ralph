package prompt

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
)

// Resolve returns the content of a prompt template.
// It first checks the project-local directory, then falls back to the embedded default.
func Resolve(promptName string, projectPromptDir string) ([]byte, error) {
	// Step 1: Check project-local prompt.
	localPath := filepath.Join(projectPromptDir, promptName+".md")
	if data, err := os.ReadFile(localPath); err == nil {
		return data, nil
	}

	// Step 2: Fall back to embedded default.
	promptsFS, err := scaffold.PromptsFS()
	if err != nil {
		return nil, err
	}

	return fs.ReadFile(promptsFS, promptName+".md")
}

// Available lists all prompt template names from the embedded defaults.
func Available() ([]string, error) {
	promptsFS, err := scaffold.PromptsFS()
	if err != nil {
		return nil, err
	}

	var names []string
	entries, err := fs.ReadDir(promptsFS, ".")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			name := e.Name()
			if len(name) > 3 && name[len(name)-3:] == ".md" {
				names = append(names, name[:len(name)-3])
			}
		}
	}
	return names, nil
}
