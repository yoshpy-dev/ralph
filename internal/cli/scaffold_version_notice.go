package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/yoshpy-dev/ralph/internal/gitroot"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// scaffoldVersionOlder reports whether manifestVersion, the meta.version of a
// scaffold's .ralph/manifest.toml, is an older version than cliVersion, the
// CLI's Version (issue #313; plan 2026-11-12-scaffold-version-notice, D1).
// Both are trimmed and given a leading "v" when they have none, so "8.1.0"
// and "v8.1.0" are the same version, and compared with
// golang.org/x/mod/semver: a prerelease is older than its release
// ("8.2.0-rc.1" < "8.2.0"), each number compares as a number
// ("8.1.0" < "8.10.0"), and build metadata ("+abc") is ignored. It is false
// when either is not a valid semver, such as the "dev" of a build with no
// ldflags or an empty meta.version, so there is nothing to report then.
func scaffoldVersionOlder(manifestVersion, cliVersion string) bool {
	m, c := semverWithV(manifestVersion), semverWithV(cliVersion)
	if !semver.IsValid(m) || !semver.IsValid(c) {
		return false
	}
	return semver.Compare(m, c) < 0
}

// semverWithV returns v trimmed of surrounding spaces, with the leading "v"
// golang.org/x/mod/semver requires added when it is missing. ralph writes
// meta.version and Version without it ("8.1.0", goreleaser's {{.Version}}).
func semverWithV(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// scaffoldVersionNotice returns the one line (with no trailing newline) that
// says the scaffold in root is older than the CLI and names
// `ralph upgrade` (issue #313; plan 2026-11-12-scaffold-version-notice, D2
// and D4). It reads meta.version from root's .ralph/manifest.toml and
// compares it with cliVersion by scaffoldVersionOlder. It returns "" when
// that manifest is missing or cannot be read or parsed (`ralph status` and
// `ralph doctor` report that already), and when the scaffold is not older:
// the same version, a newer one, or a version either side has that is not
// semver ("dev", ""). The line names verb, root as an absolute path (root
// itself when filepath.Abs fails), and both versions as written, trimmed of
// surrounding spaces.
func scaffoldVersionNotice(verb, root, cliVersion string) string {
	m, err := scaffold.ReadManifest(filepath.Join(root, ".ralph", "manifest.toml"))
	if err != nil {
		return ""
	}
	manifestVersion := strings.TrimSpace(m.Meta.Version)
	cliVersion = strings.TrimSpace(cliVersion)
	if !scaffoldVersionOlder(manifestVersion, cliVersion) {
		return ""
	}
	dir, err := filepath.Abs(root)
	if err != nil {
		dir = root
	}
	return fmt.Sprintf("note: %s: the scaffold in %s (.ralph/manifest.toml) is ralph %s, older than this CLI (%s), "+
		"so its skills and rules may not match this CLI; run 'ralph upgrade' in %s",
		verb, dir, manifestVersion, cliVersion, dir)
}

// orgScaffoldRoot returns the directory whose .ralph/manifest.toml an org
// verb reads for scaffoldVersionNotice, for the ledger ResolveOrgStateDir
// returned as stateDir and source (issue #313; plan
// 2026-11-12-scaffold-version-notice, D3): the main worktree root read back
// from the ledger's path (org.MainWorktreeRoot, which runs no git), else the
// main worktree of the repository containing the current working directory
// (gitroot.MainWorktree), else that working tree's toplevel
// (gitroot.Toplevel), else ".". It tries them in the order
// org.FeatureRepoRoot does, so a ledger named by --state-dir still points
// at the current repository's scaffold, and a linked worktree's copy of the
// manifest, which may be stale, is read only when git names no main
// worktree. Unlike org.FeatureRepoRoot it never fails.
func orgScaffoldRoot(stateDir, source string) string {
	if root, ok := org.MainWorktreeRoot(stateDir, source); ok {
		return root
	}
	if root, ok := gitroot.MainWorktree(""); ok {
		return root
	}
	if top, err := gitroot.Toplevel(""); err == nil && top != "" {
		return top
	}
	return "."
}

// printScaffoldVersionNotice writes scaffoldVersionNotice(verb, root,
// Version) and a newline to w when the scaffold in root is older than this
// CLI, and nothing otherwise (issue #313; plan
// 2026-11-12-scaffold-version-notice, D5). Like printLegacyModelPoolNotice
// it ignores the write error: the line is a note, and the verb goes on
// either way. Callers pass cmd.ErrOrStderr() so tests can capture the line,
// and nothing goes to stdout.
func printScaffoldVersionNotice(w io.Writer, verb, root string) {
	if notice := scaffoldVersionNotice(verb, root, Version); notice != "" {
		_, _ = fmt.Fprintln(w, notice)
	}
}
