package cli

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
	existing := filepath.Join(t.TempDir(), "existing")
	if err := captureStdout(t, func() error {
		return executeInit(existing, initConfig{ProjectName: "existing"}, false)
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
			args := append(append([]string{}, tt.args...), existing)
			if _, err := runInitCmd(t, args...); err == nil {
				t.Fatalf("init %q: got nil error, want the --pack check to refuse before the form", args)
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
