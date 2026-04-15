package action

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Executor runs ralph CLI commands. All operations go through scripts/ralph
// to preserve the orchestrator's control plane.
type Executor struct {
	repoRoot  string
	ralphPath string
}

// NewExecutor creates an Executor that runs commands through the ralph CLI.
// repoRoot must be the repository root containing scripts/ralph.
func NewExecutor(repoRoot string) (*Executor, error) {
	ralphPath := filepath.Join(repoRoot, "scripts", "ralph")
	if _, err := os.Stat(ralphPath); err != nil {
		return nil, fmt.Errorf("ralph CLI not found at %s: %w", ralphPath, err)
	}
	return &Executor{
		repoRoot:  repoRoot,
		ralphPath: ralphPath,
	}, nil
}

// RalphPath returns the resolved path to the ralph CLI script.
func (e *Executor) RalphPath() string {
	return e.ralphPath
}

// RepoRoot returns the repository root directory.
func (e *Executor) RepoRoot() string {
	return e.repoRoot
}

// ValidateSliceName ensures the slice name is safe for use as a CLI argument.
// Rejects empty names, path traversal attempts, and shell metacharacters.
func ValidateSliceName(name string) error {
	if name == "" {
		return fmt.Errorf("slice name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\;|&$`\"'<>(){}[]!#~\t\n\r ") {
		return fmt.Errorf("slice name contains invalid characters: %q", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("slice name contains path traversal: %q", name)
	}
	return nil
}

// BuildCommand creates an exec.Cmd for the given ralph subcommand and args.
// Arguments are passed directly to exec.Command (no shell interpretation).
func (e *Executor) BuildCommand(args ...string) *exec.Cmd {
	cmd := exec.Command(e.ralphPath, args...)
	cmd.Dir = e.repoRoot
	return cmd
}

// RunAsync executes a ralph command asynchronously and returns the result as a tea.Msg.
func (e *Executor) RunAsync(msgBuilder func(output string, err error) tea.Msg, args ...string) tea.Cmd {
	ralphPath := e.ralphPath
	repoRoot := e.repoRoot
	return func() tea.Msg {
		cmd := exec.Command(ralphPath, args...)
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		return msgBuilder(string(output), err)
	}
}
