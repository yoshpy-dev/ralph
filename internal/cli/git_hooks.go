package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	hookWrapperMarker = "ralph git hook wrapper"
)

type gitHookSpec struct {
	Name      string
	SourceRel string
	Marker    string
}

var managedGitHooks = []gitHookSpec{
	{Name: "pre-commit", SourceRel: "scripts/pre-commit-secret-guard.sh", Marker: "pre-commit-secret-guard"},
	{Name: "commit-msg", SourceRel: "scripts/commit-msg-guard.sh", Marker: "commit-msg-guard"},
	{Name: "prepare-commit-msg", SourceRel: "scripts/prepare-commit-msg-secret-guard.sh", Marker: "prepare-commit-msg-secret-guard"},
	{Name: "pre-merge-commit", SourceRel: "scripts/pre-merge-commit-secret-guard.sh", Marker: "pre-merge-commit-secret-guard"},
}

func installManagedGitHooks(targetDir string, out, errOut io.Writer) {
	hookDir, skipReason, err := gitHooksDir(targetDir)
	if err != nil {
		writef(errOut, "  ⚠ git hooks skipped: %v\n", err)
		return
	}
	if skipReason != "" {
		writef(errOut, "  ⚠ git hooks skipped: %s\n", skipReason)
		return
	}

	for _, spec := range managedGitHooks {
		installGitHook(targetDir, hookDir, spec, out, errOut)
	}
}

func installGitHook(targetDir, hookDir string, spec gitHookSpec, out, errOut io.Writer) {
	sourcePath := filepath.Join(targetDir, spec.SourceRel)
	source, err := os.ReadFile(sourcePath)
	if errors.Is(err, os.ErrNotExist) {
		writef(errOut, "  ⚠ %s hook skipped: %s not found\n", spec.Name, spec.SourceRel)
		return
	}
	if err != nil {
		writef(errOut, "  ⚠ %s hook skipped: could not read %s: %v\n", spec.Name, spec.SourceRel, err)
		return
	}

	hookPath := filepath.Join(hookDir, spec.Name)
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		writef(errOut, "  ⚠ %s hook skipped: could not create hook dir: %v\n", spec.Name, err)
		return
	}

	info, statErr := os.Lstat(hookPath)
	switch {
	case statErr == nil && info.Mode()&os.ModeSymlink != 0:
		writef(errOut, "  ⚠ %s hook skipped: %s is a symlink; not rewriting\n", spec.Name, hookPath)
		return
	case statErr == nil:
		existing, readErr := os.ReadFile(hookPath)
		if readErr != nil {
			writef(errOut, "  ⚠ %s hook skipped: could not inspect existing hook: %v\n", spec.Name, readErr)
			return
		}

		if bytes.Contains(existing, []byte(hookWrapperMarker)) {
			changed, err := writeWrappedGitHook(hookPath, spec.Name, source)
			if err != nil {
				writef(errOut, "  ⚠ %s hook skipped: %v\n", spec.Name, err)
				return
			}
			if changed {
				writef(out, "  ✓ %s hook updated\n", spec.Name)
			} else {
				writef(out, "  ✓ %s hook current\n", spec.Name)
			}
			return
		}

		if !bytes.Contains(existing, []byte(spec.Marker)) {
			originalPath := filepath.Join(filepath.Dir(hookPath), spec.Name+".ralph-original")
			if _, err := os.Lstat(originalPath); err == nil {
				writef(errOut, "  ⚠ %s hook skipped: existing hook is custom and %s already exists\n", spec.Name, originalPath)
				return
			} else if !errors.Is(err, os.ErrNotExist) {
				writef(errOut, "  ⚠ %s hook skipped: could not inspect %s: %v\n", spec.Name, originalPath, err)
				return
			}
			if err := os.Rename(hookPath, originalPath); err != nil {
				writef(errOut, "  ⚠ %s hook skipped: could not preserve existing hook: %v\n", spec.Name, err)
				return
			}
			if _, err := writeWrappedGitHook(hookPath, spec.Name, source); err != nil {
				writef(errOut, "  ⚠ %s hook skipped: %v\n", spec.Name, err)
				return
			}
			writef(out, "  ✓ %s hook installed (chained existing hook)\n", spec.Name)
			return
		}

		if bytes.Equal(existing, source) && info.Mode().Perm()&0100 != 0 {
			writef(out, "  ✓ %s hook current\n", spec.Name)
			return
		}
	case !errors.Is(statErr, os.ErrNotExist):
		writef(errOut, "  ⚠ %s hook skipped: could not inspect hook path: %v\n", spec.Name, statErr)
		return
	}

	if err := os.WriteFile(hookPath, source, 0755); err != nil {
		writef(errOut, "  ⚠ %s hook skipped: could not write %s: %v\n", spec.Name, hookPath, err)
		return
	}
	if err := os.Chmod(hookPath, 0755); err != nil {
		writef(errOut, "  ⚠ %s hook written but chmod failed: %v\n", spec.Name, err)
		return
	}

	if statErr == nil {
		writef(out, "  ✓ %s hook updated\n", spec.Name)
	} else {
		writef(out, "  ✓ %s hook installed\n", spec.Name)
	}
}

func writeWrappedGitHook(hookPath, hookName string, guard []byte) (bool, error) {
	hookDir := filepath.Dir(hookPath)
	guardPath := filepath.Join(hookDir, hookName+".ralph-guard")
	wrapper := gitHookWrapper(hookName)

	changed := true
	if existingGuard, guardErr := os.ReadFile(guardPath); guardErr == nil {
		if existingWrapper, wrapperErr := os.ReadFile(hookPath); wrapperErr == nil {
			changed = !bytes.Equal(existingGuard, guard) || !bytes.Equal(existingWrapper, wrapper)
		}
	}

	if err := os.WriteFile(guardPath, guard, 0755); err != nil {
		return false, fmt.Errorf("could not write %s: %w", guardPath, err)
	}
	if err := os.Chmod(guardPath, 0755); err != nil {
		return false, fmt.Errorf("could not chmod %s: %w", guardPath, err)
	}
	if err := os.WriteFile(hookPath, wrapper, 0755); err != nil {
		return false, fmt.Errorf("could not write %s: %w", hookPath, err)
	}
	if err := os.Chmod(hookPath, 0755); err != nil {
		return false, fmt.Errorf("could not chmod %s: %w", hookPath, err)
	}
	return changed, nil
}

func gitHookWrapper(hookName string) []byte {
	return []byte(`#!/usr/bin/env sh
# ralph git hook wrapper - runs the Ralph guard before any pre-existing hook.
set -eu

HOOK_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
RALPH_GUARD="$HOOK_DIR/` + hookName + `.ralph-guard"
ORIGINAL_HOOK="$HOOK_DIR/` + hookName + `.ralph-original"

"$RALPH_GUARD" "$@"
if [ -x "$ORIGINAL_HOOK" ]; then
  "$ORIGINAL_HOOK" "$@"
fi
`)
}

func gitHooksDir(targetDir string) (path string, skipReason string, err error) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		return "", "git executable not found", nil
	}

	if _, err := gitOutput(gitBin, targetDir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", "not a git repository", nil
	}

	hooksPath, err := gitOutput(gitBin, targetDir, "config", "--get", "core.hooksPath")
	if err == nil && hooksPath != "" {
		hookDir := hooksPath
		if !filepath.IsAbs(hookDir) {
			hookDir = filepath.Join(targetDir, hookDir)
		}
		hookDir, err = filepath.Abs(hookDir)
		if err != nil {
			return "", "", fmt.Errorf("resolving core.hooksPath: %w", err)
		}
		inside, err := pathInside(targetDir, hookDir)
		if err != nil {
			return "", "", err
		}
		if !inside {
			return "", fmt.Sprintf("core.hooksPath points outside this project (%s)", hookDir), nil
		}
		return hookDir, "", nil
	}
	if err != nil && !isGitConfigUnset(err) {
		return "", "", fmt.Errorf("reading core.hooksPath: %w", err)
	}

	hookPath, err := gitOutput(gitBin, targetDir, "rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err != nil {
		return "", "", fmt.Errorf("resolving git hooks path: %w", err)
	}
	if hookPath == "" {
		return "", "", errors.New("git returned an empty hooks path")
	}
	return hookPath, "", nil
}

func gitOutput(gitBin, targetDir string, args ...string) (string, error) {
	cmd := exec.Command(gitBin, append([]string{"-C", targetDir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func isGitConfigUnset(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == 1
}

func pathInside(root, candidate string) (bool, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, fmt.Errorf("resolving project root: %w", err)
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return false, fmt.Errorf("resolving hook path: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absCandidate)
	if err != nil {
		return false, fmt.Errorf("checking hook path: %w", err)
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))), nil
}
