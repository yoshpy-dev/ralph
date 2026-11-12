package upgrade

import (
	"io/fs"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// useBehaviorNotes puts fsys in the place of BehaviorNotesFS for the test and
// puts the previous one back when it ends.
func useBehaviorNotes(t *testing.T, fsys fs.FS) {
	t.Helper()
	saved := BehaviorNotesFS
	BehaviorNotesFS = fsys
	t.Cleanup(func() { BehaviorNotesFS = saved })
}

func noteFile(body string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(body)}
}

func noteVersions(notes []BehaviorNote) []string {
	var versions []string
	for _, n := range notes {
		versions = append(versions, n.Version)
	}
	return versions
}

// behaviorNotesFixture holds the notes of 8.1.0, 8.2.0, and 8.3.0, plus two
// .md files that are not notes.
func behaviorNotesFixture() fstest.MapFS {
	return fstest.MapFS{
		"README.md":    noteFile("# Release behavior notes\n\nOne file per release.\n"),
		"changelog.md": noteFile("- not a note\n"),
		"8.1.0.md":     noteFile("- note of 8.1.0\n"),
		"8.2.0.md":     noteFile("- note of 8.2.0\n- second item of 8.2.0\n"),
		"8.3.0.md":     noteFile("\n\n- note of 8.3.0\n\n"),
	}
}

func TestBehaviorNotesBetween_SelectsTheReleasesAfterFromUpToTo(t *testing.T) {
	useBehaviorNotes(t, behaviorNotesFixture())

	cases := []struct {
		name     string
		from, to string
		want     []string
	}{
		{"two releases", "8.1.0", "8.3.0", []string{"8.2.0", "8.3.0"}},
		{"only the upper end", "8.2.0", "8.3.0", []string{"8.3.0"}},
		{"from before every note", "8.0.0", "8.3.0", []string{"8.1.0", "8.2.0", "8.3.0"}},
		{"to between two notes", "8.0.0", "8.2.5", []string{"8.1.0", "8.2.0"}},
		{"to past the last note", "8.2.0", "9.0.0", []string{"8.3.0"}},
		{"leading v on both", "v8.1.0", "v8.3.0", []string{"8.2.0", "8.3.0"}},
		{"leading v on to only", "8.1.0", "v8.3.0", []string{"8.2.0", "8.3.0"}},
		{"surrounding white space", " 8.1.0 ", "8.3.0\n", []string{"8.2.0", "8.3.0"}},
		{"build metadata on both is ignored", "8.1.0+abc", "8.3.0+def", []string{"8.2.0", "8.3.0"}},
		{"build metadata on from still leaves from's note out", "8.2.0+abc", "8.3.0", []string{"8.3.0"}},
		{"same version", "8.3.0", "8.3.0", nil},
		{"same version with other build metadata", "8.3.0+a", "8.3.0+b", nil},
		{"downgrade", "8.3.0", "8.1.0", nil},
		{"from is dev", "dev", "8.3.0", nil},
		{"from is empty", "", "8.3.0", nil},
		{"to is dev", "8.1.0", "dev", nil},
		{"to is empty", "8.1.0", "", nil},
		{"no note in the range", "9.0.0", "9.1.0", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notes, problems := BehaviorNotesBetween(tc.from, tc.to)
			if got := noteVersions(notes); !slices.Equal(got, tc.want) {
				t.Errorf("BehaviorNotesBetween(%q, %q) versions = %q, want %q", tc.from, tc.to, got, tc.want)
			}
			if len(problems) != 0 {
				t.Errorf("BehaviorNotesBetween(%q, %q) problems = %q, want none", tc.from, tc.to, problems)
			}
		})
	}
}

func TestBehaviorNotesBetween_BodyIsTheFileTrimmed(t *testing.T) {
	useBehaviorNotes(t, behaviorNotesFixture())

	notes, problems := BehaviorNotesBetween("8.1.0", "8.3.0")
	want := []BehaviorNote{
		{Version: "8.2.0", Body: "- note of 8.2.0\n- second item of 8.2.0"},
		{Version: "8.3.0", Body: "- note of 8.3.0"},
	}
	if !reflect.DeepEqual(notes, want) {
		t.Errorf("notes = %#v, want %#v", notes, want)
	}
	if len(problems) != 0 {
		t.Errorf("problems = %q, want none", problems)
	}
}

func TestBehaviorNotesBetween_PrereleaseComesBeforeItsRelease(t *testing.T) {
	useBehaviorNotes(t, fstest.MapFS{
		"8.3.0.md":      noteFile("- note of 8.3.0\n"),
		"8.4.0-rc.1.md": noteFile("- note of 8.4.0-rc.1\n"),
		"8.4.0.md":      noteFile("- note of 8.4.0\n"),
	})

	cases := []struct {
		name     string
		from, to string
		want     []string
	}{
		{"to a prerelease leaves its release out", "8.3.0", "8.4.0-rc.1", []string{"8.4.0-rc.1"}},
		{"to the release takes its prerelease too", "8.3.0", "8.4.0", []string{"8.4.0-rc.1", "8.4.0"}},
		{"from the prerelease to its release", "8.4.0-rc.1", "8.4.0", []string{"8.4.0"}},
		{"to an earlier prerelease", "8.3.0", "8.4.0-beta.1", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notes, problems := BehaviorNotesBetween(tc.from, tc.to)
			if got := noteVersions(notes); !slices.Equal(got, tc.want) {
				t.Errorf("BehaviorNotesBetween(%q, %q) versions = %q, want %q", tc.from, tc.to, got, tc.want)
			}
			if len(problems) != 0 {
				t.Errorf("problems = %q, want none", problems)
			}
		})
	}
}

// The directory lists names in byte order ("10.0.0.md" < "8.10.0.md" <
// "8.2.0.md" < "8.9.0.md"), which is not the order of the versions.
func TestBehaviorNotesBetween_OrdersByVersionNotByName(t *testing.T) {
	useBehaviorNotes(t, fstest.MapFS{
		"8.2.0.md":  noteFile("- a\n"),
		"8.9.0.md":  noteFile("- b\n"),
		"8.10.0.md": noteFile("- c\n"),
		"10.0.0.md": noteFile("- d\n"),
	})

	notes, _ := BehaviorNotesBetween("8.1.0", "10.0.0")
	want := []string{"8.2.0", "8.9.0", "8.10.0", "10.0.0"}
	if got := noteVersions(notes); !slices.Equal(got, want) {
		t.Errorf("versions = %q, want %q", got, want)
	}
}

// Each of these names would fall in the range if its version were read
// loosely ("8.2" and "8.2.0+abc" are valid to golang.org/x/mod/semver), so
// the test fails if one of them is taken for a note.
func TestBehaviorNotesBetween_SkipsFilesThatAreNotNotesSilently(t *testing.T) {
	useBehaviorNotes(t, fstest.MapFS{
		"README.md":       noteFile("- looks like a note\n"),
		"changelog.md":    noteFile("- looks like a note\n"),
		"v8.2.0.md":       noteFile("- looks like a note\n"),
		"8.2.md":          noteFile("- looks like a note\n"),
		"8.2.0+abc.md":    noteFile("- looks like a note\n"),
		"08.2.0.md":       noteFile("- looks like a note\n"),
		"8.2.0.txt":       noteFile("- looks like a note\n"),
		"8.2.0":           noteFile("- looks like a note\n"),
		"drafts/8.2.0.md": noteFile("- looks like a note\n"),
		"8.3.0.md":        noteFile("- note of 8.3.0\n"),
	})

	notes, problems := BehaviorNotesBetween("8.1.0", "9.0.0")
	if got, want := noteVersions(notes), []string{"8.3.0"}; !slices.Equal(got, want) {
		t.Errorf("versions = %q, want %q", got, want)
	}
	if len(problems) != 0 {
		t.Errorf("problems = %q, want none", problems)
	}
}

func TestBehaviorNotesBetween_SkipsBadNotesInTheRangeWithAProblemLine(t *testing.T) {
	useBehaviorNotes(t, fstest.MapFS{
		"8.2.0.md":          noteFile(" \n\t\n"),
		"8.3.0.md":          noteFile("# 8.3.0\n- note\n"),
		"8.4.0.md":          noteFile("- good note\n"),
		"8.5.0.md":          noteFile("- note\n  ## More\n- more\n"),
		"8.6.0.md/inner.md": noteFile("- a directory, not a file\n"),
		"9.0.0.md":          noteFile(""),
	})

	notes, problems := BehaviorNotesBetween("8.1.0", "8.6.0")
	if got, want := noteVersions(notes), []string{"8.4.0"}; !slices.Equal(got, want) {
		t.Errorf("versions = %q, want %q", got, want)
	}
	wantProblems := []string{
		"behavior note for 8.2.0 skipped: its body is empty",
		`behavior note for 8.3.0 skipped: line 1 starts with "#"`,
		`behavior note for 8.5.0 skipped: line 2 starts with "#"`,
		"behavior note for 8.6.0 skipped: it is a directory, not a file",
	}
	// 9.0.0 is out of the range, so its empty body is not reported.
	if !slices.Equal(problems, wantProblems) {
		t.Errorf("problems = %q, want %q", problems, wantProblems)
	}
}

// unreadableFS serves files from files, except that opening the file named
// failing fails. It has only Open, so fs.ReadFile and fs.ReadDir go through
// it.
type unreadableFS struct {
	files   fstest.MapFS
	failing string
}

func (u unreadableFS) Open(name string) (fs.File, error) {
	if name == u.failing {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return u.files.Open(name)
}

func TestBehaviorNotesBetween_UnreadableNoteIsAProblemLine(t *testing.T) {
	useBehaviorNotes(t, unreadableFS{
		files: fstest.MapFS{
			"8.2.0.md": noteFile("- note of 8.2.0\n"),
			"8.3.0.md": noteFile("- note of 8.3.0\n"),
		},
		failing: "8.2.0.md",
	})

	notes, problems := BehaviorNotesBetween("8.1.0", "8.3.0")
	if got, want := noteVersions(notes), []string{"8.3.0"}; !slices.Equal(got, want) {
		t.Errorf("versions = %q, want %q", got, want)
	}
	if len(problems) != 1 ||
		!strings.HasPrefix(problems[0], "behavior note for 8.2.0 skipped: it cannot be read (") ||
		!strings.Contains(problems[0], "permission denied") {
		t.Errorf("problems = %q, want one line saying 8.2.0 cannot be read (permission denied)", problems)
	}
}

func TestBehaviorNotesBetween_UnreadableDirectoryIsAProblemLine(t *testing.T) {
	useBehaviorNotes(t, unreadableFS{
		files:   fstest.MapFS{"8.2.0.md": noteFile("- note of 8.2.0\n")},
		failing: ".",
	})

	notes, problems := BehaviorNotesBetween("8.1.0", "8.3.0")
	if len(notes) != 0 {
		t.Errorf("notes = %#v, want none", notes)
	}
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "behavior notes not read: ") {
		t.Errorf("problems = %q, want one line starting %q", problems, "behavior notes not read: ")
	}

	// With no range to look at, the directory is not read and there is
	// nothing to report.
	if notes, problems := BehaviorNotesBetween("dev", "8.3.0"); len(notes) != 0 || len(problems) != 0 {
		t.Errorf("BehaviorNotesBetween(dev, 8.3.0) = %#v, %q, want nothing", notes, problems)
	}
}

func TestValidateBehaviorNotes_EmbeddedNotesPass(t *testing.T) {
	dir := embeddedBehaviorNotesDir()
	// README.md is there, so this checks internal/upgrade/notes/ and not an
	// empty directory where every check passes.
	if _, err := fs.Stat(dir, "README.md"); err != nil {
		t.Fatalf("embedded notes directory has no README.md: %v", err)
	}
	if err := ValidateBehaviorNotes(dir); err != nil {
		t.Errorf("ValidateBehaviorNotes(embedded notes) = %v, want nil; see internal/upgrade/notes/README.md", err)
	}
}

func TestValidateBehaviorNotes_AcceptsWellFormedNotes(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":     noteFile("# Release behavior notes\n\n## One file per release\n"),
		"8.2.0.md":      noteFile("- a step of /implement now runs first\n- a hook now blocks `rm -rf`\n"),
		"8.3.0-rc.1.md": noteFile("- note of a prerelease\n  continued, with #317 inside the line\n"),
		".gitkeep":      noteFile(""),
		"draft.txt":     noteFile("# not embedded, so not checked\n"),
	}
	if err := ValidateBehaviorNotes(fsys); err != nil {
		t.Errorf("ValidateBehaviorNotes = %v, want nil", err)
	}
}

func TestValidateBehaviorNotes_RefusesBadNotes(t *testing.T) {
	const badName = "name it <X.Y.Z>.md"
	cases := []struct {
		name    string
		file    string
		body    string
		wantErr string
	}{
		{"name is not a version", "changelog.md", "- x\n", "behavior note changelog.md: " + badName},
		{"leading v", "v8.2.0.md", "- x\n", "behavior note v8.2.0.md: " + badName},
		{"shorthand version", "8.2.md", "- x\n", "behavior note 8.2.md: " + badName},
		{"build metadata", "8.2.0+abc.md", "- x\n", "behavior note 8.2.0+abc.md: " + badName},
		{"leading zero", "08.2.0.md", "- x\n", "behavior note 08.2.0.md: " + badName},
		{"empty body", "8.2.0.md", "", "behavior note 8.2.0.md: its body is empty"},
		{"white space only", "8.2.0.md", " \n\t\n", "behavior note 8.2.0.md: its body is empty"},
		{"heading first", "8.2.0.md", "# 8.2.0\n- x\n", `behavior note 8.2.0.md: line 1 starts with "#"`},
		{"heading after the list", "8.2.0.md", "- x\n\n## More\n- y\n", `behavior note 8.2.0.md: line 3 starts with "#"`},
		{"indented heading", "8.2.0.md", "- x\n    # y\n", `behavior note 8.2.0.md: line 2 starts with "#"`},
		{"directory", "8.2.0.md/inner.md", "- x\n", "behavior note 8.2.0.md: it is a directory, not a file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				"README.md": noteFile("# Release behavior notes\n"),
				"8.1.0.md":  noteFile("- a good note\n"),
				tc.file:     noteFile(tc.body),
			}
			err := ValidateBehaviorNotes(fsys)
			if err == nil {
				t.Fatalf("ValidateBehaviorNotes = nil, want an error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("ValidateBehaviorNotes = %q, want it to contain %q", err, tc.wantErr)
			}
			if strings.Contains(err.Error(), "8.1.0.md") {
				t.Errorf("ValidateBehaviorNotes = %q, names the good note 8.1.0.md", err)
			}
		})
	}
}

func TestValidateBehaviorNotes_ReportsEveryBadNote(t *testing.T) {
	err := ValidateBehaviorNotes(fstest.MapFS{
		"v8.2.0.md": noteFile("- x\n"),
		"8.3.0.md":  noteFile(""),
	})
	if err == nil {
		t.Fatal("ValidateBehaviorNotes = nil, want an error naming both notes")
	}
	for _, want := range []string{"behavior note v8.2.0.md: ", "behavior note 8.3.0.md: "} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ValidateBehaviorNotes = %q, want it to contain %q", err, want)
		}
	}
}
