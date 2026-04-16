package prompt

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
)

// Resolve returns the content of a prompt template.
// It first checks the project-local directory, then falls back to the embedded default.
func Resolve(promptName string, projectPromptDir string) ([]byte, error) {
	// Validate promptName: must be a simple filename component.
	if strings.ContainsAny(promptName, "/\\") || strings.Contains(promptName, "..") {
		return nil, fmt.Errorf("invalid prompt name: %q", promptName)
	}

	// Step 1: Check project-local prompt.
	localPath := filepath.Join(projectPromptDir, promptName+".md")
	absBase, err := filepath.Abs(projectPromptDir)
	if err != nil {
		return nil, fmt.Errorf("resolving prompt dir: %w", err)
	}
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return nil, fmt.Errorf("resolving prompt path: %w", err)
	}
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) {
		return nil, fmt.Errorf("path traversal detected for prompt: %q", promptName)
	}
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
			if strings.HasSuffix(name, ".md") {
				names = append(names, strings.TrimSuffix(name, ".md"))
			}
		}
	}
	return names, nil
}
