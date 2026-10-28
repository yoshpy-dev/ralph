package cli

import (
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"charm.land/huh/v2"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// testAvailablePacks is what scaffold.AvailablePacks returns under
// setupTestEmbedFS (cli_test.go): the packs of its templates/packs/.
var testAvailablePacks = []string{"golang", "typescript"}

const testAvailablePacksText = "available: golang, typescript"

// runInitCmd runs `ralph init <args...>` through newInitCmd and returns what
// it printed on stdout (executeInit prints with fmt.Printf) and Execute's
// error.
func runInitCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newInitCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	var out string
	err := captureStdoutText(t, &out, cmd.Execute)
	return out, err
}

func TestParsePackFlag_ReturnsThePacksToInstall(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   []string
	}{
		{name: "no flag", values: nil, want: nil},
		{name: "one name", values: []string{"golang"}, want: []string{"golang"}},
		{name: "comma separated", values: []string{"golang,typescript"}, want: []string{"golang", "typescript"}},
		{name: "repeated flag", values: []string{"typescript", "golang"}, want: []string{"typescript", "golang"}},
		{name: "same name twice", values: []string{"golang", "golang"}, want: []string{"golang"}},
		{name: "duplicate keeps first-seen order", values: []string{"typescript", "golang,typescript"}, want: []string{"typescript", "golang"}},
		{name: "spaces around names", values: []string{"golang , typescript"}, want: []string{"golang", "typescript"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePackFlag(tt.values, testAvailablePacks)
			if err != nil {
				t.Fatalf("parsePackFlag(%q): %v", tt.values, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parsePackFlag(%q) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

func TestParsePackFlag_RefusesEmptyAndUnknownNames(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		wantErr string
	}{
		{name: "empty value", values: []string{""}, wantErr: `empty language pack name in --pack ""`},
		{name: "empty value after a name", values: []string{"golang", ""}, wantErr: `empty language pack name in --pack ""`},
		{name: "empty value before a name", values: []string{"", "golang"}, wantErr: `empty language pack name in --pack ""`},
		{name: "trailing comma", values: []string{"golang,"}, wantErr: `empty language pack name in --pack "golang,"`},
		{name: "leading comma", values: []string{",golang"}, wantErr: `empty language pack name in --pack ",golang"`},
		{name: "blank name", values: []string{"golang, "}, wantErr: `empty language pack name in --pack "golang, "`},
		{name: "unknown name", values: []string{"nosuch"}, wantErr: `unknown language pack "nosuch" in --pack`},
		{name: "unknown name after a known one", values: []string{"golang,nosuch"}, wantErr: `unknown language pack "nosuch" in --pack`},
		{name: "skip is not a pack", values: []string{"skip"}, wantErr: `unknown language pack "skip" in --pack`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePackFlag(tt.values, testAvailablePacks)
			if err == nil {
				t.Fatalf("parsePackFlag(%q) = %q, want an error", tt.values, got)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), testAvailablePacksText) {
				t.Errorf("error = %q, want it to list the packs (%q)", err, testAvailablePacksText)
			}
		})
	}
}

// assertInstalledPacks checks the scaffold in target holds exactly the packs
// in want: in the manifest's Meta.Packs, under packs/languages/, and as
// .claude/rules/ralph/<lang>.md.
func assertInstalledPacks(t *testing.T, target string, want []string) {
	t.Helper()
	m, err := scaffold.ReadManifest(filepath.Join(target, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if !slices.Equal(m.Meta.Packs, want) {
		t.Errorf("manifest Meta.Packs = %q, want %q", m.Meta.Packs, want)
	}
	for _, p := range testAvailablePacks {
		for _, rel := range []string{packRelDir(p), packRuleRelPath(p)} {
			_, err := os.Stat(filepath.Join(target, rel))
			switch installed := slices.Contains(want, p); {
			case installed && err != nil:
				t.Errorf("pack %s: %s should exist: %v", p, rel, err)
			case !installed && !os.IsNotExist(err):
				t.Errorf("pack %s: %s should not exist; stat err = %v", p, rel, err)
			}
		}
	}
	if len(want) == 0 {
		if _, err := os.Stat(filepath.Join(target, "packs", "languages")); !os.IsNotExist(err) {
			t.Errorf("packs/languages/ should not exist with no pack installed; stat err = %v", err)
		}
	}
}

func TestInitCmd_YesAloneInstallsNoPackAndPrintsTheHint(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	target := filepath.Join(t.TempDir(), "project")

	out, err := runInitCmd(t, "--yes", target)
	if err != nil {
		t.Fatalf("init --yes: %v\noutput:\n%s", err, out)
	}

	assertInstalledPacks(t, target, nil)
	wantHint := "  ralph pack add <language> to add a language pack later (" + testAvailablePacksText + ")\n"
	if !strings.Contains(out, wantHint) {
		t.Errorf("output should contain the hint %q; got:\n%s", wantHint, out)
	}
	if hint, doctor := strings.Index(out, wantHint), strings.Index(out, "  ralph doctor to verify setup"); hint > doctor {
		t.Errorf("the hint should come before the ralph doctor line in Next steps; got:\n%s", out)
	}
}

func TestInitCmd_YesWithPackInstallsExactlyThosePacks(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "one pack", args: []string{"--pack", "golang"}, want: []string{"golang"}},
		{name: "comma separated", args: []string{"--pack", "golang,typescript"}, want: []string{"golang", "typescript"}},
		{name: "repeated flag in flag order", args: []string{"--pack", "typescript", "--pack", "golang"}, want: []string{"typescript", "golang"}},
		{name: "same pack twice", args: []string{"--pack", "golang", "--pack", "golang"}, want: []string{"golang"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestEmbedFS(t)
			Version = "0.1.0-test"
			target := filepath.Join(t.TempDir(), "project")

			args := append(append([]string{"--yes"}, tt.args...), target)
			out, err := runInitCmd(t, args...)
			if err != nil {
				t.Fatalf("init %q: %v\noutput:\n%s", args, err, out)
			}

			assertInstalledPacks(t, target, tt.want)
			if strings.Contains(out, "ralph pack add") {
				t.Errorf("output should not carry the ralph pack add hint when a pack was installed; got:\n%s", out)
			}
		})
	}
}

// TestInitCmd_BadPackNameStopsBeforeAnyWrite checks that an unknown or empty
// --pack name ends init with an error before the form and before anything is
// written, with and without --yes.
func TestInitCmd_BadPackNameStopsBeforeAnyWrite(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	// An existing v2 project: without --yes, init on it returns nil from
	// runInitInteractive's existing-project check before the form. A run
	// there that does not fail shows the --pack check did not run, and it
	// cannot wait on a form for a terminal.
	existingV2 := filepath.Join(t.TempDir(), "existing")
	if err := captureStdout(t, func() error {
		return executeInit(existingV2, initConfig{ProjectName: "existing"}, false)
	}); err != nil {
		t.Fatalf("seed init: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "unknown name", args: []string{"--pack", "nosuch"}, wantErr: `unknown language pack "nosuch" in --pack`},
		{name: "skip is not a pack", args: []string{"--pack", "skip"}, wantErr: `unknown language pack "skip" in --pack`},
		{name: "empty value", args: []string{"--pack", ""}, wantErr: `empty language pack name in --pack ""`},
		{name: "empty value after a name", args: []string{"--pack", "golang", "--pack", ""}, wantErr: `empty language pack name in --pack ""`},
		{name: "empty value before a name", args: []string{"--pack", "", "--pack", "golang"}, wantErr: `empty language pack name in --pack ""`},
		{name: "trailing comma", args: []string{"--pack", "golang,"}, wantErr: `empty language pack name in --pack "golang,"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkErr := func(args []string, err error) {
				t.Helper()
				if !strings.Contains(err.Error(), tt.wantErr) || !strings.Contains(err.Error(), testAvailablePacksText) {
					t.Errorf("init %q: error = %q, want it to contain %q and %q", args, err, tt.wantErr, testAvailablePacksText)
				}
			}

			// Without --yes, on the existing project first: if the check is
			// missing this returns nil instead of reaching the form.
			args := append(append([]string{}, tt.args...), existingV2)
			if _, err := runInitCmd(t, args...); err == nil {
				t.Fatalf("init %q on an existing v2 project: got nil, which runInitInteractive returns there before its form; want the --pack check in RunE to refuse first (this run comes before the new-directory runs so that a missing check fails here instead of waiting on the form)", args)
			} else {
				checkErr(args, err)
			}

			for _, yes := range [][]string{{"--yes"}, nil} {
				target := filepath.Join(t.TempDir(), "project")
				args := append(append(append([]string{}, yes...), tt.args...), target)
				out, err := runInitCmd(t, args...)
				if err == nil {
					t.Fatalf("init %q: got nil error, want the --pack check to refuse\noutput:\n%s", args, out)
				}
				checkErr(args, err)
				if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
					t.Errorf("init %q: target %s should not be created; stat err = %v", args, target, statErr)
				}
				if out != "" {
					t.Errorf("init %q: should print nothing before the error; got:\n%s", args, out)
				}
			}
		})
	}
}

// repositoryPacks is every pack in the repository's templates/packs/.
var repositoryPacks = []string{"dart", "golang", "python", "rust", "terraform", "typescript"}

// useRepositoryTemplates points scaffold.EmbeddedFS at the repository's
// templates/ (what cmd/ralph embeds) for the rest of the test.
func useRepositoryTemplates(t *testing.T) {
	t.Helper()
	saved := scaffold.EmbeddedFS
	t.Cleanup(func() { scaffold.EmbeddedFS = saved })
	scaffold.EmbeddedFS = os.DirFS(filepath.Join("..", ".."))
}

// TestSkipPackValue_IsNoPackName checks skipPackValue holds a "/". That is
// enough: scaffold.AvailablePacks lists the directory names of
// templates/packs/, and no directory name holds a "/".
func TestSkipPackValue_IsNoPackName(t *testing.T) {
	if !strings.Contains(skipPackValue, "/") {
		t.Errorf("skipPackValue = %q, want a \"/\" in it so that no pack directory can have its name", skipPackValue)
	}
}

func TestPackMarkers_NameEveryRepositoryPack(t *testing.T) {
	useRepositoryTemplates(t)
	packs, err := scaffold.AvailablePacks()
	if err != nil {
		t.Fatalf("AvailablePacks: %v", err)
	}
	if !slices.Equal(packs, repositoryPacks) {
		t.Errorf("templates/packs/ holds %q, want %q: give a new pack its markers in packMarkers (init.go) and a row in TestRecommendedPacks_FindsEachMarker", packs, repositoryPacks)
	}
	keys := slices.Sorted(maps.Keys(packMarkers))
	if !slices.Equal(keys, repositoryPacks) {
		t.Errorf("packMarkers names %q, want the repository packs %q", keys, repositoryPacks)
	}
}

func TestRecommendedPacks_FindsEachMarker(t *testing.T) {
	// One row per marker of packMarkers (the table of issue #275 and the
	// shell detection); a pattern is given as a name it matches.
	tests := []struct {
		file string
		pack string
	}{
		{file: "go.mod", pack: "golang"},
		{file: "package.json", pack: "typescript"},
		{file: "package-lock.json", pack: "typescript"},
		{file: "pnpm-lock.yaml", pack: "typescript"},
		{file: "yarn.lock", pack: "typescript"},
		{file: "tsconfig.json", pack: "typescript"},
		{file: "tsconfig.build.json", pack: "typescript"},
		{file: "pyproject.toml", pack: "python"},
		{file: "requirements.txt", pack: "python"},
		{file: "requirements-dev.txt", pack: "python"},
		{file: "setup.py", pack: "python"},
		{file: "setup.cfg", pack: "python"},
		{file: "tox.ini", pack: "python"},
		{file: "Cargo.toml", pack: "rust"},
		{file: "pubspec.yaml", pack: "dart"},
		{file: "main.tf", pack: "terraform"},
		{file: "main.tofu", pack: "terraform"},
		{file: "main.tftest.hcl", pack: "terraform"},
		{file: ".terraform.lock.hcl", pack: "terraform"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			dir := t.TempDir()
			writeMarkerFile(t, filepath.Join(dir, tt.file))
			if got := recommendedPacks(dir, repositoryPacks); !slices.Equal(got, []string{tt.pack}) {
				t.Errorf("recommendedPacks with %s = %q, want [%q]", tt.file, got, tt.pack)
			}
		})
	}
}

func TestRecommendedPacks_LooksOnlyAtFilesDirectlyInTheTarget(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string)
		available []string
		want      []string
	}{
		{name: "empty directory", setup: func(*testing.T, string) {}},
		{name: "marker only in a subdirectory", setup: func(t *testing.T, dir string) {
			writeMarkerFile(t, filepath.Join(dir, "sub", "go.mod"))
		}},
		{name: "directory named go.mod", setup: func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, "go.mod"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "names that only start like a marker", setup: func(t *testing.T, dir string) {
			for _, name := range []string{"go.mod.bak", "tsconfig.json.orig", "requirements.in", "vars.tfvars"} {
				writeMarkerFile(t, filepath.Join(dir, name))
			}
		}},
		{name: "symlink to a marker file", want: []string{"golang"}, setup: func(t *testing.T, dir string) {
			file := filepath.Join(t.TempDir(), "elsewhere.txt")
			writeMarkerFile(t, file)
			symlinkMarker(t, file, filepath.Join(dir, "go.mod"))
		}},
		{name: "symlink to a directory", setup: func(t *testing.T, dir string) {
			symlinkMarker(t, t.TempDir(), filepath.Join(dir, "go.mod"))
		}},
		{name: "dangling symlink", setup: func(t *testing.T, dir string) {
			symlinkMarker(t, filepath.Join(dir, "missing"), filepath.Join(dir, "go.mod"))
		}},
		{name: "several packs in available's order", available: []string{"typescript", "golang"}, want: []string{"typescript", "golang"}, setup: func(t *testing.T, dir string) {
			writeMarkerFile(t, filepath.Join(dir, "go.mod"))
			writeMarkerFile(t, filepath.Join(dir, "package.json"))
		}},
		{name: "marker of a pack that is not available", available: testAvailablePacks, setup: func(t *testing.T, dir string) {
			writeMarkerFile(t, filepath.Join(dir, "Cargo.toml"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			available := tt.available
			if available == nil {
				available = repositoryPacks
			}
			if got := recommendedPacks(dir, available); !slices.Equal(got, tt.want) {
				t.Errorf("recommendedPacks = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("target directory does not exist yet", func(t *testing.T) {
		if got := recommendedPacks(filepath.Join(t.TempDir(), "project"), repositoryPacks); got != nil {
			t.Errorf("recommendedPacks = %q, want none", got)
		}
	})
}

func TestPackOptions_LabelsSelectionAndSkip(t *testing.T) {
	skip := huh.NewOption("skip (no language pack)", skipPackValue)
	golang := huh.NewOption("golang", "golang")
	typescript := huh.NewOption("typescript", "typescript")
	tests := []struct {
		name        string
		recommended []string
		initial     []string
		want        []huh.Option[string]
	}{
		{name: "no --pack selects nothing", want: []huh.Option[string]{golang, typescript, skip}},
		{name: "--pack golang selects only golang", initial: []string{"golang"}, want: []huh.Option[string]{golang.Selected(true), typescript, skip}},
		{
			name:        "recommended label keeps the bare value",
			recommended: []string{"typescript"},
			want:        []huh.Option[string]{golang, huh.NewOption("typescript (Recommended)", "typescript"), skip},
		},
		{
			name:        "recommended and selected",
			recommended: []string{"golang", "typescript"},
			initial:     []string{"typescript"},
			want: []huh.Option[string]{
				huh.NewOption("golang (Recommended)", "golang"),
				huh.NewOption("typescript (Recommended)", "typescript").Selected(true),
				skip,
			},
		},
		{name: "skip in the initial selection", initial: []string{skipPackValue}, want: []huh.Option[string]{golang, typescript, skip.Selected(true)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := packOptions(testAvailablePacks, tt.recommended, tt.initial)
			if !slices.Equal(got, tt.want) {
				t.Errorf("packOptions(%q, recommended %q, initial %q) =\n  %+v\nwant\n  %+v", testAvailablePacks, tt.recommended, tt.initial, got, tt.want)
			}
		})
	}
}

const (
	errChooseSomething = "choose at least one language pack, or skip"
	errSkipAlone       = "choose skip alone, or language packs without skip"
)

func TestValidatePackSelection(t *testing.T) {
	tests := []struct {
		name      string
		selection []string
		wantErr   string
	}{
		{name: "skip alone", selection: []string{skipPackValue}},
		{name: "one pack", selection: []string{"golang"}},
		{name: "two packs", selection: []string{"golang", "typescript"}},
		{name: "skip with a pack", selection: []string{"golang", skipPackValue}, wantErr: errSkipAlone},
		{name: "skip before a pack", selection: []string{skipPackValue, "golang"}, wantErr: errSkipAlone},
		{name: "nothing (nil)", selection: nil, wantErr: errChooseSomething},
		{name: "nothing (empty)", selection: []string{}, wantErr: errChooseSomething},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePackSelection(tt.selection)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("validatePackSelection(%q) = %v, want nil", tt.selection, err)
			case tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr):
				t.Errorf("validatePackSelection(%q) = %v, want %q", tt.selection, err, tt.wantErr)
			}
		})
	}
}

func TestPacksFromSelection_DropsSkip(t *testing.T) {
	tests := []struct {
		name      string
		selection []string
		want      []string
	}{
		{name: "skip alone", selection: []string{skipPackValue}, want: nil},
		{name: "one pack", selection: []string{"golang"}, want: []string{"golang"}},
		{name: "packs in selection order", selection: []string{"typescript", "golang"}, want: []string{"typescript", "golang"}},
		{name: "skip with a pack, which validatePackSelection refuses", selection: []string{"golang", skipPackValue}, want: []string{"golang"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := packsFromSelection(tt.selection)
			if !slices.Equal(got, tt.want) {
				t.Errorf("packsFromSelection(%q) = %q, want %q", tt.selection, got, tt.want)
			}
			if slices.Contains(got, skipPackValue) {
				t.Errorf("packsFromSelection(%q) = %q passes skip's value on to executeInit", tt.selection, got)
			}
		})
	}
}

// TestNewPackQuestion_ValidatesTheSelectionOnBlur builds the question with
// each selection already made (newPackQuestion selects the values it is
// given) and blurs it: Blur reads the selection back and runs the question's
// Validate, as the Next and Submit keys do before leaving the question.
func TestNewPackQuestion_ValidatesTheSelectionOnBlur(t *testing.T) {
	tests := []struct {
		name      string
		selection []string
		wantErr   string
	}{
		{name: "skip alone", selection: []string{skipPackValue}},
		{name: "packs", selection: []string{"golang", "typescript"}},
		{name: "skip with a pack", selection: []string{"golang", skipPackValue}, wantErr: errSkipAlone},
		{name: "nothing", selection: nil, wantErr: errChooseSomething},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selection := slices.Clone(tt.selection)
			question := newPackQuestion(testAvailablePacks, nil, &selection)
			question.Blur()

			err := question.Error()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("Error() after Blur = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr):
				t.Errorf("Error() after Blur = %v, want %q", err, tt.wantErr)
			}
			if !slices.Equal(selection, tt.selection) {
				t.Errorf("selection after Blur = %q, want %q (the question's value should be the selection it started with)", selection, tt.selection)
			}
		})
	}
}

// TestNewPackQuestion_ShowsEveryOptionWithItsLabel checks the question shows
// its description and every option, skip included, without scrolling.
func TestNewPackQuestion_ShowsEveryOptionWithItsLabel(t *testing.T) {
	var selection []string
	view := newPackQuestion(testAvailablePacks, []string{"typescript"}, &selection).View()
	wants := append([]string{"Language packs", "typescript (Recommended)", "skip (no language pack)"}, strings.Split(packQuestionDescription, "\n")...)
	for _, want := range wants {
		if !strings.Contains(view, want) {
			t.Errorf("question view should show %q; got:\n%s", want, view)
		}
	}
	if !strings.Contains(view, "golang") || strings.Contains(view, "golang (Recommended)") {
		t.Errorf("question view should show golang without (Recommended); got:\n%s", view)
	}

	// The question's height counts one line per description line, so a line
	// that wraps would push the last option (skip) out of view.
	for _, line := range strings.Split(packQuestionDescription, "\n") {
		if len(line) > 76 {
			t.Errorf("description line %q is %d characters, want at most 76 so it does not wrap in an 80-column terminal", line, len(line))
		}
	}
}

// heldStdinWriters keeps the write end of a stdin pipe that a form still
// waits on after its test failed. Once unreachable, the garbage collector
// would close it, and at the end of stdin the form asks without end.
var heldStdinWriters []*os.File

// TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm checks
// init does not ask when huh would ask in accessible mode (TERM=dumb) on a
// stdin that is not a terminal: at the end of such a stdin huh asks the pack
// question again without end. The run gets a bounded wait, and stdin stays
// open while it runs, so a missing check fails here instead of looping.
func TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	t.Setenv("TERM", "dumb")
	stdinRead, stdinWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdin
	os.Stdin = stdinRead
	t.Cleanup(func() { os.Stdin = saved })
	target := filepath.Join(t.TempDir(), "project")

	done := make(chan error, 1)
	cfg := initConfig{ProjectName: "project", AvailablePacks: testAvailablePacks}
	go func() { done <- runInitInteractive(target, cfg, false) }()
	select {
	case err := <-done:
		_ = stdinWrite.Close()
		_ = stdinRead.Close()
		if err == nil || !strings.Contains(err.Error(), "ralph init --yes") {
			t.Errorf("runInitInteractive with TERM=dumb and a pipe for stdin = %v, want an error that points at ralph init --yes", err)
		}
	case <-time.After(10 * time.Second):
		heldStdinWriters = append(heldStdinWriters, stdinWrite)
		t.Fatal("runInitInteractive with TERM=dumb and a pipe for stdin did not return in 10s: it reached the form, which waits on stdin")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("target %s should not be created; stat err = %v", target, err)
	}
}

func TestExecuteInit_SkipAloneInstallsNoPackAndPrintsTheHint(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	target := filepath.Join(t.TempDir(), "project")

	cfg := initConfig{ProjectName: "project", Packs: packsFromSelection([]string{skipPackValue}), AvailablePacks: testAvailablePacks}
	var out string
	if err := captureStdoutText(t, &out, func() error { return executeInit(target, cfg, false) }); err != nil {
		t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
	}

	assertInstalledPacks(t, target, nil)
	wantHint := "  ralph pack add <language> to add a language pack later (" + testAvailablePacksText + ")\n"
	if !strings.Contains(out, wantHint) {
		t.Errorf("output should contain the hint %q; got:\n%s", wantHint, out)
	}
	if warning := "⚠ pack " + skipPackValue; strings.Contains(out, warning) {
		t.Errorf("output should not carry %q: skip's value reached executeInit as a pack; got:\n%s", warning, out)
	}
}

func TestExecuteInit_HintWithNoAvailablePacksLeavesOutTheList(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	target := filepath.Join(t.TempDir(), "project")

	var out string
	if err := captureStdoutText(t, &out, func() error { return executeInit(target, initConfig{ProjectName: "project"}, false) }); err != nil {
		t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
	}

	wantHint := "  ralph pack add <language> to add a language pack later\n"
	if !strings.Contains(out, wantHint) {
		t.Errorf("output should contain the hint %q with no list; got:\n%s", wantHint, out)
	}
}

// writeMarkerFile writes a small file at path, creating its directory.
func writeMarkerFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symlinkMarker(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}
