package cli

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeAliasRc writes content to path, creating parent directories as
// needed, so fixtures can target nested rc locations like
// ~/.config/zsh/.zshrc or a ZDOTDIR-relative .zshrc.
func writeAliasRc(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setShellAliasTestHome pins HOME (and clears ZDOTDIR by default) to an
// isolated temp directory so checkShellAliases never reads the real
// machine's rc files.
func setShellAliasTestHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("ZDOTDIR", "")
	return dir
}

func TestCheckShellAliases_NoRcFiles_Pass(t *testing.T) {
	setShellAliasTestHome(t)

	r := checkShellAliases(true)
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "0 shell rc file(s)") {
		t.Errorf("expected detail to mention 0 shell rc file(s), got: %s", r.Detail)
	}
}

func TestCheckShellAliases_ZshrcModelFlag_Warn(t *testing.T) {
	home := setShellAliasTestHome(t)
	writeAliasRc(t, filepath.Join(home, ".zshrc"),
		"alias codex=\"codex -m gpt-6-astra -c 'model_reasoning_effort=\\\"xhigh\\\"'\"\n")

	r := checkShellAliases(true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{"~" + string(filepath.Separator) + ".zshrc:1", "--model (-m)", "docs/recipes/codex-seat-permissions.md"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
}

func TestCheckShellAliases_ModelFlagForms_Warn(t *testing.T) {
	cases := []struct {
		name  string
		alias string
		want  string
	}{
		{"concatenated short flag", `alias codex='codex -mgpt-5.5'`, "--model (-m)"},
		{"long flag with equals", `alias codex="codex --model=gpt-5.5"`, "--model"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := setShellAliasTestHome(t)
			writeAliasRc(t, filepath.Join(home, ".zshrc"), tc.alias+"\n")

			r := checkShellAliases(true)
			if r.Status != "warn" {
				t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
			}
			if !strings.Contains(r.Detail, tc.want) {
				t.Errorf("expected detail to contain %q, got: %s", tc.want, r.Detail)
			}
		})
	}
}

func TestCheckShellAliases_ClaudeModelFlag_Warn(t *testing.T) {
	home := setShellAliasTestHome(t)
	writeAliasRc(t, filepath.Join(home, ".zshrc"),
		`alias claude="claude --model fable --effort xhigh --enable-auto-mode"`+"\n")

	r := checkShellAliases(true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{"alias claude", "--model"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
}

func TestCheckShellAliases_HarmlessAliases_Pass(t *testing.T) {
	home := setShellAliasTestHome(t)
	writeAliasRc(t, filepath.Join(home, ".zshrc"),
		"alias codex=\"codex\"\nalias claude='claude --effort xhigh'\n")

	r := checkShellAliases(true)
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "alias line(s) seen") {
		t.Errorf("expected detail to mention alias line(s) seen, got: %s", r.Detail)
	}
}

func TestCheckShellAliases_SymlinkedRc_DedupedToOneFinding(t *testing.T) {
	home := setShellAliasTestHome(t)
	real := filepath.Join(home, ".config", "zsh", ".zshrc")
	writeAliasRc(t, real, "alias codex=\"codex -m x\"\n")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}

	r := checkShellAliases(true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	if n := strings.Count(r.Detail, "alias codex in"); n != 1 {
		t.Errorf("expected exactly one finding, got %d occurrences in: %s", n, r.Detail)
	}
}

func TestCheckShellAliases_Zdotdir_Detected(t *testing.T) {
	home := setShellAliasTestHome(t)
	zdotdir := filepath.Join(home, "zdot")
	t.Setenv("ZDOTDIR", zdotdir)
	writeAliasRc(t, filepath.Join(zdotdir, ".zshrc"), "alias codex=\"codex -m x\"\n")

	r := checkShellAliases(true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckShellAliases_FishAlias_Warn(t *testing.T) {
	home := setShellAliasTestHome(t)
	writeAliasRc(t, filepath.Join(home, ".config", "fish", "config.fish"), "alias codex \"codex -m x\"\n")

	r := checkShellAliases(true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckShellAliases_HerdrAbsent_ConflictDowngradesToInfo(t *testing.T) {
	home := setShellAliasTestHome(t)
	writeAliasRc(t, filepath.Join(home, ".zshrc"),
		"alias codex=\"codex -m gpt-6-astra -c 'model_reasoning_effort=\\\"xhigh\\\"'\"\n")

	r := checkShellAliases(false)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "herdr not installed") {
		t.Errorf("expected detail to mention herdr not installed, got: %s", r.Detail)
	}
}

func TestCheckShellAliases_HomeUnresolvable_Info(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.UserHomeDir on windows does not key off HOME")
	}
	t.Setenv("HOME", "")
	t.Setenv("ZDOTDIR", "")

	r := checkShellAliases(true)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "could not resolve home directory") {
		t.Errorf("expected detail to mention could not resolve home directory, got: %s", r.Detail)
	}
}

func TestCheckShellAliases_CommentedAliasLine_Pass(t *testing.T) {
	home := setShellAliasTestHome(t)
	writeAliasRc(t, filepath.Join(home, ".zshrc"), "# alias codex=\"codex -m x\"\n")

	r := checkShellAliases(true)
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
}

func TestConflictingFlag(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"codex -m x", "--model (-m)"},
		{"codex -mx", "--model (-m)"},
		{"codex --model=x", "--model"},
		{"codex --model x", "--model"},
		{"claude --permission-mode bypassPermissions", "--permission-mode"},
		{"codex --sandbox workspace-write", "--sandbox"},
		{"codex --ask-for-approval never", "--ask-for-approval"},
		{"codex -c a=b --effort xhigh", ""},
		{"codex --models-dir x", ""},
		{"codex -m", "--model (-m)"},
		{"codex --m", ""},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			if got := conflictingFlag(tc.value); got != tc.want {
				t.Errorf("conflictingFlag(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

// TestRunDoctorOpts_ShellAliasesCheck_AppearsInOutput is the integration pin
// for AC-1: the "Shell aliases (codex/claude)" check runs as part of
// runDoctorOpts and its output line is present. HOME is pinned to an
// alias-free temp directory so the result is deterministically "pass" and
// does not affect the exit code (modeled on
// TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected).
func TestRunDoctorOpts_ShellAliasesCheck_AppearsInOutput(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(dir, "no-such-agmsg-home"))

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("ZDOTDIR", "")

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runDoctorOpts(dir, false)
	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("runDoctorOpts returned error with an alias-free HOME: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(string(out), "Shell aliases (codex/claude):") {
		t.Errorf("expected Shell aliases (codex/claude) line in output:\n%s", out)
	}
}
