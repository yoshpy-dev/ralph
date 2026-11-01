package config

import (
	"os"
	"path/filepath"
	"strings"
)

// IsAbsOrHomePath reports whether path is an absolute path or starts with
// "~/", the two forms that name the same directory wherever the process
// that reads them runs (plan 2026-11-01-org-ledger-gaps, D2). It is false
// for "", a bare "~", "~user/...", and every relative path. The checks of
// the personal file (checkLocalAgmsgHome, LocalOrgStateDir) accept these
// forms, and the CLI's refusal of a relative agmsg home
// (agmsgHomeIsRelative) is its complement for a value that is not empty.
func IsAbsOrHomePath(path string) bool {
	return filepath.IsAbs(path) || strings.HasPrefix(path, "~/")
}

// ExpandHomePath returns path with a leading "~/" replaced by the current
// user's home directory (os.UserHomeDir), joined with filepath.Join, so the
// result is cleaned. Any other path, a bare "~" and "~user/..." included,
// is returned unchanged with a nil error. When os.UserHomeDir fails for a
// path that starts with "~/", the error is that of os.UserHomeDir, returned
// as is, and the path is "". LocalOrgStateDir reports that error;
// driver.ResolveAgmsgHome (expandTilde) keeps the value unchanged instead.
func ExpandHomePath(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
