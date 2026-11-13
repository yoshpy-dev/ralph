package upgrade

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

// embeddedBehaviorNotes holds internal/upgrade/notes/: one Markdown file per
// release, named <X.Y.Z>.md, that lists what changes in the behavior of the
// scaffold in that release, and the README.md that tells a maintainer how to
// write one (issue #317; plan 2026-11-13-upgrade-behavior-notes, D2 and D4).
// The README also keeps the pattern matching while no release has a note,
// so the build does not fail on an empty pattern.
//
//go:embed notes/*.md
var embeddedBehaviorNotes embed.FS

// BehaviorNotesFS is the directory BehaviorNotesBetween reads the notes
// from: notes/ of embeddedBehaviorNotes. It is declared as fs.FS so the tests
// of this package and of internal/cli can put an fstest.MapFS of notes in its
// place (and put the embedded one back with t.Cleanup); nothing else sets it.
var BehaviorNotesFS fs.FS = embeddedBehaviorNotesDir()

// embeddedBehaviorNotesDir returns notes/ of embeddedBehaviorNotes as an
// fs.FS rooted at that directory.
func embeddedBehaviorNotesDir() fs.FS {
	dir, err := fs.Sub(embeddedBehaviorNotes, "notes")
	if err != nil {
		// fs.Sub fails only for a path that is not valid, and "notes" is.
		panic(fmt.Sprintf("upgrade: behavior notes directory: %v", err))
	}
	return dir
}

// BehaviorNote is the behavior note of one release (issue #317).
type BehaviorNote struct {
	// Version is the release's version as the file name gives it, with no
	// leading "v" ("8.2.0", "8.2.0-rc.1").
	Version string
	// Body is the file's content with surrounding white space trimmed: a
	// Markdown bullet list with no line that starts with "#".
	Body string
}

// BehaviorNotesBetween returns the notes in BehaviorNotesFS of the releases
// after from up to and including to (from < version <= to), oldest first, for
// the upgrade report (issue #317; plan 2026-11-13-upgrade-behavior-notes,
// D3). from, to, and the notes' versions are compared as semver with
// golang.org/x/mod/semver, a leading "v" added where it is missing: "8.1.0"
// and "v8.1.0" are the same version, a prerelease comes before its release
// (a to of "8.2.0-rc.1" leaves out the note of 8.2.0), and build metadata
// ("+abc") is ignored. It returns nothing when from or to is not a valid
// semver ("dev", ""), or when from is not older than to.
//
// A file is a note only when behaviorNoteVersion accepts its name; any other
// file (README.md, a name with a leading "v") is skipped silently. A note in
// the range that cannot be used (it cannot be read, its body is empty, or a
// line starts with "#") is skipped too, and problems gets one line about it
// for the report's Notes section (D5).
func BehaviorNotesBetween(from, to string) (notes []BehaviorNote, problems []string) {
	fromV, toV := SemverWithV(from), SemverWithV(to)
	if !semver.IsValid(fromV) || !semver.IsValid(toV) || semver.Compare(fromV, toV) >= 0 {
		return nil, nil
	}
	entries, err := fs.ReadDir(BehaviorNotesFS, ".")
	if err != nil {
		return nil, []string{fmt.Sprintf("behavior notes not read: %v", err)}
	}

	type candidate struct {
		version string
		entry   fs.DirEntry
	}
	var inRange []candidate
	for _, e := range entries {
		version, ok := behaviorNoteVersion(e.Name())
		if !ok {
			continue
		}
		v := "v" + version
		if semver.Compare(fromV, v) >= 0 || semver.Compare(v, toV) > 0 {
			continue
		}
		inRange = append(inRange, candidate{version: version, entry: e})
	}
	sort.Slice(inRange, func(i, j int) bool {
		return semver.Compare("v"+inRange[i].version, "v"+inRange[j].version) < 0
	})

	for _, c := range inRange {
		body, problem := readBehaviorNote(BehaviorNotesFS, c.entry)
		if problem != "" {
			problems = append(problems, fmt.Sprintf("behavior note for %s skipped: %s", c.version, problem))
			continue
		}
		notes = append(notes, BehaviorNote{Version: c.version, Body: body})
	}
	return notes, problems
}

// ValidateBehaviorNotes checks the shape of the notes in fsys, a directory
// laid out as internal/upgrade/notes/ (issue #317; plan
// 2026-11-13-upgrade-behavior-notes, D5 and AC6). Every .md file other than
// README.md must have a name behaviorNoteVersion accepts, be a file that can
// be read, have a body that is not empty once white space is trimmed, and
// have no line that starts with "#", indented or not. It returns every
// problem found, joined, or nil. Files whose names do not end in ".md" are
// not checked, since //go:embed notes/*.md leaves them out of the binary.
func ValidateBehaviorNotes(fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("behavior notes: %w", err)
	}
	var errs []error
	for _, e := range entries {
		name := e.Name()
		if name == "README.md" || !strings.HasSuffix(name, ".md") {
			continue
		}
		if _, ok := behaviorNoteVersion(name); !ok {
			errs = append(errs, fmt.Errorf("behavior note %s: name it <X.Y.Z>.md, a semver written in full with no leading v and no build metadata", name))
			continue
		}
		if _, problem := readBehaviorNote(fsys, e); problem != "" {
			errs = append(errs, fmt.Errorf("behavior note %s: %s", name, problem))
		}
	}
	return errors.Join(errs...)
}

// behaviorNoteVersion returns name without ".md" and true when name is a
// note's file name: <X.Y.Z>.md or <X.Y.Z>-<prerelease>.md, the version a
// semver written in full ("8.2", which golang.org/x/mod/semver also accepts,
// is not), with no leading "v" and no build metadata. For any other name,
// README.md included, it returns "" and false.
func behaviorNoteVersion(name string) (string, bool) {
	version, ok := strings.CutSuffix(name, ".md")
	if !ok || strings.HasPrefix(version, "v") {
		return "", false
	}
	if semver.Canonical("v"+version) != "v"+version {
		return "", false
	}
	return version, true
}

// readBehaviorNote reads the note e in fsys and returns its body with
// surrounding white space trimmed, or, when the note cannot be used, a short
// reason (problem) and no body.
func readBehaviorNote(fsys fs.FS, e fs.DirEntry) (body, problem string) {
	if e.IsDir() {
		return "", "it is a directory, not a file"
	}
	content, err := fs.ReadFile(fsys, e.Name())
	if err != nil {
		return "", fmt.Sprintf("it cannot be read (%v)", err)
	}
	body = strings.TrimSpace(string(content))
	if body == "" {
		return "", "its body is empty"
	}
	for i, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			return "", fmt.Sprintf("line %d starts with \"#\"", i+1)
		}
	}
	return body, ""
}

// SemverWithV returns v trimmed of surrounding spaces, with the leading "v"
// golang.org/x/mod/semver requires added when it is missing. ralph writes
// meta.version and Version without it ("8.1.0", goreleaser's {{.Version}}).
// BehaviorNotesBetween reads its versions with it, and so does
// scaffoldVersionOlder in internal/cli, so the upgrade report's behavior
// notes and the note that a scaffold is older than the CLI read a version
// the same way.
func SemverWithV(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}
