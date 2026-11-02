package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// The lines executeInit prints for ralph.toml when it has answers.
const (
	initSettingsWroteLine      = "  ✓ ralph.toml (wrote your answers)\n"
	initSettingsNotWrittenLine = "  ⚠ ralph.toml: already exists, so your answers were not written\n"
)

// setupInitSettingsEmbedFS is setupTestEmbedFS with the real
// templates/base/ralph.toml in place of the fake one, which has none of the
// keys the answers write. It returns the template's bytes.
func setupInitSettingsEmbedFS(t *testing.T) []byte {
	t.Helper()
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	template := readInitSettingsTemplate(t)
	embedded, ok := scaffold.EmbeddedFS.(fstest.MapFS)
	if !ok {
		t.Fatalf("setupTestEmbedFS set scaffold.EmbeddedFS to a %T, not an fstest.MapFS", scaffold.EmbeddedFS)
	}
	embedded["templates/base/ralph.toml"] = &fstest.MapFile{Data: template}
	return template
}

// executeInitOutput runs executeInit with stdout captured and returns what
// it printed and its error.
func executeInitOutput(t *testing.T, target string, cfg initConfig, force bool) (string, error) {
	t.Helper()
	var out string
	err := captureStdoutText(t, &out, func() error { return executeInit(target, cfg, force) })
	return out, err
}

// readInitSettingsFile returns the ralph.toml in dir.
func readInitSettingsFile(t *testing.T, dir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "ralph.toml"))
	if err != nil {
		t.Fatalf("reading ralph.toml: %v", err)
	}
	return data
}

// readInitSettingsManifestEntry returns the manifest entry of ralph.toml in
// dir's .ralph/manifest.toml.
func readInitSettingsManifestEntry(t *testing.T, dir string) scaffold.ManifestFile {
	t.Helper()
	m, err := scaffold.ReadManifest(filepath.Join(dir, ".ralph", "manifest.toml"))
	if err != nil {
		t.Fatalf("reading the manifest: %v", err)
	}
	entry, ok := m.Files["ralph.toml"]
	if !ok {
		t.Fatalf("the manifest has no ralph.toml entry: %v", m.Files)
	}
	return entry
}

// checkInitSettingsManifestEntry checks that the manifest records ralph.toml
// as seed with the template's hash as its Hash and TemplateHash, and
// diskHash ("" for none) as its DiskHash.
func checkInitSettingsManifestEntry(t *testing.T, dir string, template []byte, diskHash string) {
	t.Helper()
	entry := readInitSettingsManifestEntry(t, dir)
	templateHash := scaffold.HashBytes(template)
	if entry.TemplateHash != templateHash || entry.Hash != templateHash {
		t.Errorf("manifest ralph.toml: TemplateHash %s, Hash %s, want the template's hash %s in both", entry.TemplateHash, entry.Hash, templateHash)
	}
	if entry.DiskHash != diskHash {
		t.Errorf("manifest ralph.toml: DiskHash %q, want %q", entry.DiskHash, diskHash)
	}
	if entry.Owner != scaffold.OwnerSeed {
		t.Errorf("manifest ralph.toml: owner %q, want %q", entry.Owner, scaffold.OwnerSeed)
	}
}

// TestInitSettings_AnsweredFSServesTheAnswersForRalphTOMLOnly pins the FS
// executeInit renders when it has answers: fs.ReadFile reads the answers for
// ralph.toml and the base FS's bytes for every other file, fs.WalkDir walks
// the base FS's tree, and scaffold.RenderFS writes and hashes the answers as
// ralph.toml.
func TestInitSettings_AnsweredFSServesTheAnswersForRalphTOMLOnly(t *testing.T) {
	base, err := fs.Sub(fstest.MapFS{
		"templates/base/ralph.toml": {Data: []byte("template = true\n")},
		"templates/base/docs/a.md":  {Data: []byte("# a\n")},
		"templates/base/scripts/x":  {Data: []byte("x\n")},
	}, "templates/base")
	if err != nil {
		t.Fatal(err)
	}
	answered := []byte("answered = true\n")
	fsys := initAnsweredFS{FS: base, answered: answered}

	got, err := fs.ReadFile(fsys, "ralph.toml")
	if err != nil || !bytes.Equal(got, answered) {
		t.Fatalf("fs.ReadFile(ralph.toml) = %q, %v; want %q", got, err, answered)
	}
	got[0] = 'X'
	if again, _ := fs.ReadFile(fsys, "ralph.toml"); !bytes.Equal(again, []byte("answered = true\n")) {
		t.Errorf("changing the bytes ReadFile returned changed the next read: %q", again)
	}
	if got, err := fs.ReadFile(fsys, "docs/a.md"); err != nil || string(got) != "# a\n" {
		t.Errorf("fs.ReadFile(docs/a.md) = %q, %v; want the base FS's %q", got, err, "# a\n")
	}
	if _, err := fs.ReadFile(fsys, "missing.md"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("fs.ReadFile(missing.md) error = %v, want fs.ErrNotExist", err)
	}

	walk := func(fsys fs.FS) []string {
		var paths []string
		if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			paths = append(paths, path)
			return err
		}); err != nil {
			t.Fatalf("fs.WalkDir: %v", err)
		}
		return paths
	}
	if got, want := walk(fsys), walk(base); !slices.Equal(got, want) {
		t.Errorf("fs.WalkDir walks %v, want the base FS's %v", got, want)
	}

	target := t.TempDir()
	result, hashes, err := scaffold.RenderFS(fsys, scaffold.RenderOptions{TargetDir: target})
	if err != nil {
		t.Fatalf("RenderFS: %v", err)
	}
	if !slices.Contains(result.Created, "ralph.toml") {
		t.Errorf("RenderFS created %v, want ralph.toml among them", result.Created)
	}
	if got := readInitSettingsFile(t, target); !bytes.Equal(got, answered) {
		t.Errorf("RenderFS wrote ralph.toml %q, want the answers %q", got, answered)
	}
	if got, err := os.ReadFile(filepath.Join(target, "docs", "a.md")); err != nil || string(got) != "# a\n" {
		t.Errorf("RenderFS wrote docs/a.md %q, %v; want %q", got, err, "# a\n")
	}
	if want := scaffold.HashBytes(answered); hashes["ralph.toml"] != want {
		t.Errorf("RenderFS hashed ralph.toml %s, want the answers' hash %s", hashes["ralph.toml"], want)
	}
}

// TestExecuteInit_SettingsNilWritesTheTemplateRalphTOML is AC4: without
// answers (the --yes path) ralph.toml is the template's bytes, and the
// manifest records it as before: the template's hash and no DiskHash.
func TestExecuteInit_SettingsNilWritesTheTemplateRalphTOML(t *testing.T) {
	template := setupInitSettingsEmbedFS(t)
	target := filepath.Join(t.TempDir(), "new-project")

	out, err := executeInitOutput(t, target, initConfig{ProjectName: "new-project"}, false)
	if err != nil {
		t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
	}
	if got := readInitSettingsFile(t, target); !bytes.Equal(got, template) {
		t.Errorf("ralph.toml is not the template's bytes:\n%s", got)
	}
	checkInitSettingsManifestEntry(t, target, template, "")
	if strings.Contains(out, "your answers") {
		t.Errorf("init without answers printed a line about them:\n%s", out)
	}
}

// TestExecuteInit_SettingsWriteTheAnswersOnTheFirstWrite checks that with
// answers ralph.toml is written as applyInitSettings returns it, with the
// line that says so, and that the manifest keeps the template's hash as
// TemplateHash and records the written content's hash as DiskHash.
func TestExecuteInit_SettingsWriteTheAnswersOnTheFirstWrite(t *testing.T) {
	template := setupInitSettingsEmbedFS(t)
	target := filepath.Join(t.TempDir(), "new-project")
	s := initSettings{CLI: initCLIClaude, CI: true, AutoMerge: false}
	want, err := applyInitSettings(template, s)
	if err != nil {
		t.Fatalf("applyInitSettings: %v", err)
	}
	if bytes.Equal(want, template) {
		t.Fatal("the answers leave the template unchanged, so this test cannot tell them apart")
	}

	out, err := executeInitOutput(t, target, initConfig{ProjectName: "new-project", Settings: &s}, false)
	if err != nil {
		t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
	}
	got := readInitSettingsFile(t, target)
	if !bytes.Equal(got, want) {
		t.Errorf("ralph.toml is not the template with the answers:\n%s", got)
	}
	checkInitSettingsManifestEntry(t, target, template, scaffold.HashBytes(got))
	if !strings.Contains(out, initSettingsWroteLine) {
		t.Errorf("output has no line %q:\n%s", initSettingsWroteLine, out)
	}
	if strings.Contains(out, initSettingsNotWrittenLine) {
		t.Errorf("output says the answers were not written:\n%s", out)
	}

	cfg, err := config.Load(filepath.Join(target, "ralph.toml"))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.Org.Director.AutoMerge || cfg.Org.Gate.RegateChecks != config.RegateChecksCI || !slices.Equal(cfg.Org.DriverPool, []string{initCLIClaude}) {
		t.Errorf("ralph.toml loads auto_merge %t, regate_checks %q, driver_pool %v; want false, %q, [claude]",
			cfg.Org.Director.AutoMerge, cfg.Org.Gate.RegateChecks, cfg.Org.DriverPool, config.RegateChecksCI)
	}
}

// TestExecuteInit_SettingsAndAnExistingRalphTOML checks a ralph.toml that is
// already in the target directory: without --force init leaves it, says the
// answers were not written, and records the template's hash with no
// DiskHash; with --force it overwrites it with the answers.
func TestExecuteInit_SettingsAndAnExistingRalphTOML(t *testing.T) {
	existing := []byte("# my settings\n[standard]\ncross_review = true\n")
	s := initSettings{CLI: initCLIBoth, PlanReview: true, AutoMerge: false}

	t.Run("without force", func(t *testing.T) {
		template := setupInitSettingsEmbedFS(t)
		target := t.TempDir()
		if err := os.WriteFile(filepath.Join(target, "ralph.toml"), existing, 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := executeInitOutput(t, target, initConfig{ProjectName: "p", Settings: &s}, false)
		if err != nil {
			t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
		}
		if got := readInitSettingsFile(t, target); !bytes.Equal(got, existing) {
			t.Errorf("the existing ralph.toml changed:\n%s", got)
		}
		if !strings.Contains(out, initSettingsNotWrittenLine) {
			t.Errorf("output has no line %q:\n%s", initSettingsNotWrittenLine, out)
		}
		if strings.Contains(out, initSettingsWroteLine) {
			t.Errorf("output says the answers were written:\n%s", out)
		}
		checkInitSettingsManifestEntry(t, target, template, "")
	})

	t.Run("with force", func(t *testing.T) {
		template := setupInitSettingsEmbedFS(t)
		want, err := applyInitSettings(template, s)
		if err != nil {
			t.Fatalf("applyInitSettings: %v", err)
		}
		target := t.TempDir()
		if err := os.WriteFile(filepath.Join(target, "ralph.toml"), existing, 0o644); err != nil {
			t.Fatal(err)
		}
		out, err := executeInitOutput(t, target, initConfig{ProjectName: "p", Settings: &s}, true)
		if err != nil {
			t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
		}
		if got := readInitSettingsFile(t, target); !bytes.Equal(got, want) {
			t.Errorf("ralph.toml is not the template with the answers:\n%s", got)
		}
		if !strings.Contains(out, initSettingsWroteLine) {
			t.Errorf("output has no line %q:\n%s", initSettingsWroteLine, out)
		}
		checkInitSettingsManifestEntry(t, target, template, scaffold.HashBytes(want))
	})
}

// TestExecuteInit_SettingsTheTemplateDoesNotFitWriteNothing is AC8: when the
// template ralph.toml lacks a key the answers write (the fake ralph.toml of
// setupTestEmbedFS), init fails with an error naming the key before it
// writes anything: no target directory, and no file in one that exists.
func TestExecuteInit_SettingsTheTemplateDoesNotFitWriteNothing(t *testing.T) {
	s := initSettings{CLI: initCLIBoth, AutoMerge: true}
	wantPrefix := "writing the answers into ralph.toml: template ralph.toml has no ["

	t.Run("new directory", func(t *testing.T) {
		setupTestEmbedFS(t)
		Version = "0.1.0-test"
		target := filepath.Join(t.TempDir(), "new-project")
		out, err := executeInitOutput(t, target, initConfig{ProjectName: "new-project", Settings: &s}, false)
		if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
			t.Fatalf("executeInit error = %v, want one starting with %q", err, wantPrefix)
		}
		if _, statErr := os.Lstat(target); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("the target directory was created (Lstat error %v)", statErr)
		}
		if out != "" {
			t.Errorf("executeInit printed before it failed:\n%s", out)
		}
	})

	t.Run("existing empty directory", func(t *testing.T) {
		setupTestEmbedFS(t)
		Version = "0.1.0-test"
		target := t.TempDir()
		_, err := executeInitOutput(t, target, initConfig{ProjectName: "p", Settings: &s}, false)
		if err == nil || !strings.HasPrefix(err.Error(), wantPrefix) {
			t.Fatalf("executeInit error = %v, want one starting with %q", err, wantPrefix)
		}
		entries, err := os.ReadDir(target)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("executeInit wrote %v before it failed", entries)
		}
	})
}

// initSettingsUpgradeDryRun runs ralph upgrade --dry-run --diff on dir and
// returns what it printed. --diff names each advisory's path.
func initSettingsUpgradeDryRun(t *testing.T, dir string) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(dir, upgradeOptions{DryRun: true, DiffPreview: true, Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("upgrade --dry-run: %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errOut.String())
	}
	return out.String()
}

// TestExecuteInit_SettingsThenUpgradeDryRunHasNoRalphTOMLAdvisory is AC7:
// right after an init that wrote the answers, ralph upgrade --dry-run has no
// seed advisory for ralph.toml, because the manifest's TemplateHash is the
// template's hash. The sub-test shows the check would catch a TemplateHash
// that is the answers' hash.
func TestExecuteInit_SettingsThenUpgradeDryRunHasNoRalphTOMLAdvisory(t *testing.T) {
	setupInitSettingsEmbedFS(t)
	dir := t.TempDir()
	s := initSettings{CLI: initCLICodex, CI: true, AutoMerge: false}
	if out, err := executeInitOutput(t, dir, initConfig{ProjectName: "p", Settings: &s}, false); err != nil {
		t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
	}

	const noAdvisory = "  advisories:        0 files\n"
	const ralphTOMLAdvisory = "--- ralph.toml (owner: seed) ---"
	out := initSettingsUpgradeDryRun(t, dir)
	if !strings.Contains(out, noAdvisory) || strings.Contains(out, ralphTOMLAdvisory) {
		t.Errorf("upgrade --dry-run right after init has an advisory (want the line %q and no %q):\n%s", noAdvisory, ralphTOMLAdvisory, out)
	}

	t.Run("a TemplateHash of the answers raises the advisory", func(t *testing.T) {
		manifestPath := filepath.Join(dir, ".ralph", "manifest.toml")
		m, err := scaffold.ReadManifest(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		entry := m.Files["ralph.toml"]
		entry.TemplateHash = scaffold.HashBytes(readInitSettingsFile(t, dir))
		entry.Hash = entry.TemplateHash
		m.Files["ralph.toml"] = entry
		if err := m.Write(manifestPath); err != nil {
			t.Fatal(err)
		}
		out := initSettingsUpgradeDryRun(t, dir)
		if !strings.Contains(out, "  advisories:        1 files\n") || !strings.Contains(out, ralphTOMLAdvisory) {
			t.Errorf("upgrade --dry-run has no ralph.toml advisory for a TemplateHash of the answers:\n%s", out)
		}
	})
}

// TestRunDoctorOpts_InitSettingsForEachCLIPass is the doctor part of AC5:
// for each CLI answer, runDoctorOpts on the project init wrote reads
// ralph.toml with no warn line, passes with only that CLI's binary (and go)
// on PATH, and counts the model_pool entries of that CLI's drivers.
func TestRunDoctorOpts_InitSettingsForEachCLIPass(t *testing.T) {
	tmplCfg := loadInitSettingsTOML(t, readInitSettingsTemplate(t))
	cases := []struct {
		cli  string
		bins []string
	}{
		{initCLIClaude, []string{"claude", "go"}},
		{initCLICodex, []string{"codex", "go"}},
		{initCLIBoth, []string{"claude", "codex", "go"}},
	}
	for _, tc := range cases {
		t.Run(tc.cli, func(t *testing.T) {
			setupInitSettingsEmbedFS(t)
			dir := t.TempDir()
			s := initSettings{CLI: tc.cli, AutoMerge: true}
			if out, err := executeInitOutput(t, dir, initConfig{ProjectName: "p", Settings: &s}, false); err != nil {
				t.Fatalf("executeInit: %v\noutput:\n%s", err, out)
			}

			binDir := t.TempDir()
			for _, bin := range tc.bins {
				writeStubBin(t, binDir, bin, "")
			}
			t.Setenv("PATH", binDir)
			t.Setenv("HOME", t.TempDir())
			setAgmsgHomeOverride(t, "")

			var out string
			if err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) }); err != nil {
				t.Fatalf("doctor failed with only %v on PATH: %v\noutput:\n%s", tc.bins, err, out)
			}
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "  ⚠ ralph.toml: warn") {
					t.Errorf("doctor warns about the ralph.toml init wrote: %s", line)
				}
			}
			entries := 0
			for _, e := range tmplCfg.Org.ModelPool {
				if tc.cli == initCLIBoth || e.Driver == tc.cli {
					entries++
				}
			}
			if entries == 0 {
				t.Fatalf("the template's model_pool has no %s entry", tc.cli)
			}
			if want := fmt.Sprintf("Org envelope: info — model_pool: %d entries, max_seats: %d\n", entries, tmplCfg.Org.MaxSeats); !strings.Contains(out, want) {
				t.Errorf("doctor output has no line %q:\n%s", want, out)
			}
		})
	}
}

// TestExecuteInit_SettingsSurviveAFailedRenderAndARerunWithout is AC10: an
// init whose rendering fails after it wrote ralph.toml (the target's
// scripts is a regular file) leaves the answers in ralph.toml, and the
// re-run without answers (the form does not ask when ralph.toml exists)
// keeps them and records the template's hash as TemplateHash.
func TestExecuteInit_SettingsSurviveAFailedRenderAndARerunWithout(t *testing.T) {
	template := setupInitSettingsEmbedFS(t)
	// The re-run skips the AGENTS.md the first run wrote, and init reads the
	// managed block of a skipped AGENTS.md from the template, so the
	// template needs the block's markers.
	scaffold.EmbeddedFS.(fstest.MapFS)["templates/base/AGENTS.md"] = &fstest.MapFile{Data: testAgentsMDContent()}
	target := t.TempDir()
	scripts := filepath.Join(target, "scripts")
	if err := os.WriteFile(scripts, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := initSettings{CLI: initCLIBoth, ChangeReview: true, AutoMerge: false}
	want, err := applyInitSettings(template, s)
	if err != nil {
		t.Fatalf("applyInitSettings: %v", err)
	}

	out, err := executeInitOutput(t, target, initConfig{ProjectName: "p", Settings: &s}, false)
	if err == nil {
		t.Fatalf("executeInit with a regular file at scripts succeeded:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(target, "ralph.toml")); statErr != nil {
		t.Fatalf("the failed run wrote no ralph.toml (RenderFS walks ralph.toml before scripts/, so it should have): %v\nerror: %v", statErr, err)
	}
	if got := readInitSettingsFile(t, target); !bytes.Equal(got, want) {
		t.Errorf("the failed run left a ralph.toml without the answers:\n%s", got)
	}
	cfg, err := config.Load(filepath.Join(target, "ralph.toml"))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.Org.Director.AutoMerge {
		t.Error("the failed run left [org.director].auto_merge = true, not the answer false")
	}

	if err := os.Remove(scripts); err != nil {
		t.Fatal(err)
	}
	out, err = executeInitOutput(t, target, initConfig{ProjectName: "p"}, false)
	if err != nil {
		t.Fatalf("the re-run without answers: %v\noutput:\n%s", err, out)
	}
	if got := readInitSettingsFile(t, target); !bytes.Equal(got, want) {
		t.Errorf("the re-run changed the answered ralph.toml:\n%s", got)
	}
	checkInitSettingsManifestEntry(t, target, template, "")
}
