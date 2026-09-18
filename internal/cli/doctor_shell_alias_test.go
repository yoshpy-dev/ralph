package cli

import (
	"errors"
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

// shellAliasTestEnv returns a resolveEnv closure fixed to home, with no
// ZDOTDIR, for tests that do not want to mutate process environment
// variables at all.
func shellAliasTestEnv(home string) func() (shellAliasEnv, error) {
	return func() (shellAliasEnv, error) { return shellAliasEnv{Home: home}, nil }
}

// shellAliasTestEnvWithZdotdir is shellAliasTestEnv plus a ZDOTDIR.
func shellAliasTestEnvWithZdotdir(home, zdotdir string) func() (shellAliasEnv, error) {
	return func() (shellAliasEnv, error) { return shellAliasEnv{Home: home, Zdotdir: zdotdir}, nil }
}

func TestCheckShellAliases_NoRcFiles_Pass(t *testing.T) {
	dir := t.TempDir()

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "no shell rc file found to scan") {
		t.Errorf("expected detail to mention no shell rc file found to scan, got: %s", r.Detail)
	}
}

func TestCheckShellAliases_CodexModelFlag_Warn(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"),
		`alias codex="codex -m gpt-6-astra -c 'model_reasoning_effort=\"xhigh\"'"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
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
			dir := t.TempDir()
			writeAliasRc(t, filepath.Join(dir, ".zshrc"), tc.alias+"\n")

			r := checkShellAliases(shellAliasTestEnv(dir), true)
			if r.Status != "warn" {
				t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
			}
			if !strings.Contains(r.Detail, tc.want) {
				t.Errorf("expected detail to contain %q, got: %s", tc.want, r.Detail)
			}
		})
	}
}

// TestCheckShellAliases_ClaudeModelFlag_Info pins the measured CLI behaviour
// (Design decisions): claude accepts a repeated --model without erroring, so
// a claude-only finding is informational, never a spawn blocker.
func TestCheckShellAliases_ClaudeModelFlag_Info(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"),
		`alias claude="claude --model fable --effort xhigh --enable-auto-mode"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{"alias claude in", "--model", "the seat still starts"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
}

func TestCheckShellAliases_MixedCodexAndClaude_WarnWithCodexSentenceFirst(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"),
		"alias codex=\"codex -m x\"\nalias claude=\"claude --model y\"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	codexIdx := strings.Index(r.Detail, "alias codex in")
	claudeIdx := strings.Index(r.Detail, "alias claude in")
	if codexIdx == -1 || claudeIdx == -1 {
		t.Fatalf("expected both codex and claude sentences, got: %s", r.Detail)
	}
	if codexIdx > claudeIdx {
		t.Errorf("expected the codex sentence before the claude sentence, got: %s", r.Detail)
	}
	if !strings.Contains(r.Detail, "cannot be used multiple times") {
		t.Errorf("expected the codex sentence's content, got: %s", r.Detail)
	}
	if !strings.Contains(r.Detail, "the seat still starts") {
		t.Errorf("expected the claude sentence's content, got: %s", r.Detail)
	}
}

func TestCheckShellAliases_HarmlessAlias_Pass(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias codex="codex"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	want := "alias codex in ~" + string(filepath.Separator) + ".zshrc:1 has no conflicting flags"
	if !strings.Contains(r.Detail, want) {
		t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
	}
}

func TestCheckShellAliases_SymlinkedRc_DedupedToOneFinding(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, ".config", "zsh", ".zshrc")
	writeAliasRc(t, real, `alias codex="codex -m x"`+"\n")
	if err := os.Symlink(real, filepath.Join(dir, ".zshrc")); err != nil {
		t.Fatal(err)
	}

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	if n := strings.Count(r.Detail, "alias codex in"); n != 1 {
		t.Errorf("expected exactly one finding, got %d occurrences in: %s", n, r.Detail)
	}
}

func TestCheckShellAliases_Zdotdir_Detected(t *testing.T) {
	dir := t.TempDir()
	zdotdir := filepath.Join(dir, "zdot")
	writeAliasRc(t, filepath.Join(zdotdir, ".zshrc"), `alias codex="codex -m x"`+"\n")

	r := checkShellAliases(shellAliasTestEnvWithZdotdir(dir, zdotdir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckShellAliases_FishAlias_Warn(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".config", "fish", "config.fish"), `alias codex "codex -m x"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckShellAliases_HerdrAbsent_CodexConflictDowngradesToInfo(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias codex="codex -m x"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), false)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "herdr not installed") {
		t.Errorf("expected detail to mention herdr not installed, got: %s", r.Detail)
	}
}

func TestCheckShellAliases_EnvResolutionError_Info(t *testing.T) {
	wantErr := errors.New("boom")
	r := checkShellAliases(func() (shellAliasEnv, error) { return shellAliasEnv{}, wantErr }, true)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "could not resolve home directory") {
		t.Errorf("expected detail to mention could not resolve home directory, got: %s", r.Detail)
	}
}

func TestShellAliasEnvFromOS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.UserHomeDir on windows does not key off HOME")
	}

	t.Run("HOME empty returns an error", func(t *testing.T) {
		t.Setenv("HOME", "")
		if _, err := shellAliasEnvFromOS(); err == nil {
			t.Fatal("expected an error when HOME is empty")
		}
	})

	t.Run("resolves Home and Zdotdir from the process environment", func(t *testing.T) {
		dir := t.TempDir()
		zdotdir := filepath.Join(dir, "zdot")
		t.Setenv("HOME", dir)
		t.Setenv("ZDOTDIR", zdotdir)

		env, err := shellAliasEnvFromOS()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if env.Home != dir {
			t.Errorf("expected Home %q, got %q", dir, env.Home)
		}
		if env.Zdotdir != zdotdir {
			t.Errorf("expected Zdotdir %q, got %q", zdotdir, env.Zdotdir)
		}
	})
}

// TestCheckShellAliases_CodexShortFlags_Warn is the H1 fix: codex's short
// flags -s/-a are clap aliases for --sandbox/--ask-for-approval and collide
// exactly like -m does (self-review H1).
func TestCheckShellAliases_CodexShortFlags_Warn(t *testing.T) {
	cases := []struct {
		name  string
		alias string
		want  string
	}{
		{"sandbox short flag with space", `alias codex='codex -s danger-full-access'`, "--sandbox (-s)"},
		{"sandbox short flag concatenated", `alias codex='codex -sread-only'`, "--sandbox (-s)"},
		{"ask-for-approval short flag", `alias codex='codex -a never'`, "--ask-for-approval (-a)"},
		{"ask-for-approval long flag with equals", `alias codex="codex --ask-for-approval=never"`, "--ask-for-approval"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeAliasRc(t, filepath.Join(dir, ".zshrc"), tc.alias+"\n")

			r := checkShellAliases(shellAliasTestEnv(dir), true)
			if r.Status != "warn" {
				t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
			}
			if !strings.Contains(r.Detail, tc.want) {
				t.Errorf("expected detail to contain %q, got: %s", tc.want, r.Detail)
			}
		})
	}
}

func TestCheckShellAliases_MultipleCodexFlags_ListedInOrder(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias codex='codex -m x -s y'`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "--model (-m), --sandbox (-s)") {
		t.Errorf("expected detail to list both flags in order of appearance, got: %s", r.Detail)
	}
}

// TestCheckShellAliases_TrailingComment_NotMisreadAsFlag is the M4 fix: a
// trailing `# ...` comment on the same line must not be tokenized into the
// alias value.
func TestCheckShellAliases_TrailingComment_NotMisreadAsFlag(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias codex='codex'  # never add --model here`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "pass" {
		t.Fatalf("expected pass (trailing comment must not be tokenized), got %s (%s)", r.Status, r.Detail)
	}
}

// TestCheckShellAliases_TwoAliasesOnOneStatement_EachAttributedCorrectly is
// the other half of M4: several NAME=VALUE words in one alias statement are
// each examined and attributed to their own name.
func TestCheckShellAliases_TwoAliasesOnOneStatement_EachAttributedCorrectly(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias codex='codex' claude='claude --model x'`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "info" {
		t.Fatalf("expected info (only claude carries a flag, codex is harmless), got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "alias claude in") || !strings.Contains(r.Detail, "--model") {
		t.Errorf("expected detail to name claude with --model, got: %s", r.Detail)
	}
}

func TestCheckShellAliases_AliasOption_StillDetected(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias -g codex='codex -m x'`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn (a leading -g option must not block detection), got %s (%s)", r.Status, r.Detail)
	}
}

// TestCheckShellAliases_SecondStatementOnSameLine_NotDetected pins the
// documented limitation that only one alias statement per line is read: the
// codex alias here follows an unquoted ';' and is never examined.
func TestCheckShellAliases_SecondStatementOnSameLine_NotDetected(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias ll='ls -l'; alias codex='codex -m x'`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "pass" {
		t.Fatalf("expected pass (documented limitation: a second statement on the line is not read), got %s (%s)", r.Status, r.Detail)
	}
}

func TestCheckShellAliases_EscapedQuotesInValue_Warn(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"),
		`alias codex="codex -m gpt -c 'model_reasoning_effort=\"xhigh\"'"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "--model (-m)") {
		t.Errorf("expected detail to contain --model (-m), got: %s", r.Detail)
	}
}

// TestCheckShellAliases_PassDetail_NamesScannedFiles is the M2 fix: even a
// clean pass names which rc files were actually opened, not just a count.
func TestCheckShellAliases_PassDetail_NamesScannedFiles(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias ls='ls -la'`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "pass" {
		t.Fatalf("expected pass, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{"~" + string(filepath.Separator) + ".zshrc", "files they source are not followed"} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
}

// TestCheckShellAliases_UnreadableRc_Info is the M3 fix: an rc that cannot
// be opened is named in the Detail with its reason, and the check reports
// info rather than a bare pass.
func TestCheckShellAliases_UnreadableRc_Info(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0o000 is not meaningful on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root ignores file permissions")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, ".zshrc")
	writeAliasRc(t, path, `alias codex="codex -m x"`+"\n")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	want := "could not read: ~" + string(filepath.Separator) + ".zshrc (permission denied)"
	if !strings.Contains(r.Detail, want) {
		t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
	}
}

// TestCheckShellAliases_LongPrecedingLine_StillFindsLaterAlias proves the 1
// MiB scanner buffer (M3): a line far longer than bufio.Scanner's 64 KiB
// default must not abort the whole file before the alias line is reached.
func TestCheckShellAliases_LongPrecedingLine_StillFindsLaterAlias(t *testing.T) {
	dir := t.TempDir()
	longLine := "export SOME_VAR=" + strings.Repeat("x", 70*1024)
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), longLine+"\n"+`alias codex="codex -m x"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn (the 1 MiB buffer must absorb the long preceding line), got %s (%s)", r.Status, r.Detail)
	}
}

// TestCheckShellAliases_OverLongLine_KeepsPartialFindingsAndReportsUnreadable
// is the other half of M3: a line beyond even the 1 MiB cap still fails the
// scan, but findings collected before the failure are kept, not discarded.
func TestCheckShellAliases_OverLongLine_KeepsPartialFindingsAndReportsUnreadable(t *testing.T) {
	dir := t.TempDir()
	firstLine := `alias codex="codex -m x"`
	overLong := strings.Repeat("y", 2*1024*1024)
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), firstLine+"\n"+overLong+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn (findings collected before the scan error must be kept), got %s (%s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "could not read:") {
		t.Errorf("expected detail to report the scan failure, got: %s", r.Detail)
	}
}

// TestCheckShellAliases_AliasValueNeverAppearsInDetail guards against an
// alias value (which can hold secrets, e.g. `-c api_key=...`) leaking into
// doctor output -- only the fixed flag labels are ever rendered.
func TestCheckShellAliases_AliasValueNeverAppearsInDetail(t *testing.T) {
	dir := t.TempDir()
	writeAliasRc(t, filepath.Join(dir, ".zshrc"), `alias codex="codex -m x -c api_key=SECRET123"`+"\n")

	r := checkShellAliases(shellAliasTestEnv(dir), true)
	if strings.Contains(r.Detail, "SECRET123") {
		t.Errorf("alias value must never appear in Detail, got: %s", r.Detail)
	}
}

func TestShellAliasWords(t *testing.T) {
	cases := []struct {
		name string
		rest string
		want []string
	}{
		{"simple unquoted", " codex x", []string{"codex", "x"}},
		{"single-quoted with spaces", ` codex='codex -m x'`, []string{"codex=codex -m x"}},
		{"double-quoted with escaped quote", ` codex="codex -m \"x\""`, []string{`codex=codex -m "x"`}},
		{"single-quote concatenation", ` codex='a'\''b'`, []string{"codex=a'b"}},
		{"comment ends statement", " codex=codex # trailing comment", []string{"codex=codex"}},
		{"semicolon ends statement", " codex=codex; claude=claude", []string{"codex=codex"}},
		{"pipe ends statement", " codex=codex | cat", []string{"codex=codex"}},
		{"ampersand ends statement", " codex=codex & echo hi", []string{"codex=codex"}},
		{"unterminated single quote runs to EOL", ` codex='codex -m x`, []string{"codex=codex -m x"}},
		{"unterminated double quote runs to EOL", ` codex="codex -m x`, []string{"codex=codex -m x"}},
		{"hash mid-word is not a comment", " codex=codex#not-a-comment", []string{"codex=codex#not-a-comment"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shellAliasWords(tc.rest)
			if len(got) != len(tc.want) {
				t.Fatalf("shellAliasWords(%q) = %#v, want %#v", tc.rest, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("shellAliasWords(%q)[%d] = %q, want %q", tc.rest, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestShellAliasConflictingFlags(t *testing.T) {
	cases := []struct {
		name  string
		alias string
		value string
		want  []string
	}{
		{"codex short model with space", "codex", "codex -m x", []string{"--model (-m)"}},
		{"codex short model concatenated", "codex", "codex -mx", []string{"--model (-m)"}},
		{"codex long model with equals", "codex", "codex --model=x", []string{"--model"}},
		{"codex long model with space", "codex", "codex --model x", []string{"--model"}},
		{"claude permission-mode", "claude", "claude --permission-mode bypassPermissions", []string{"--permission-mode"}},
		{"codex long sandbox", "codex", "codex --sandbox workspace-write", []string{"--sandbox"}},
		{"codex long ask-for-approval", "codex", "codex --ask-for-approval never", []string{"--ask-for-approval"}},
		{"unrelated -c flag", "codex", "codex -c a=b --effort xhigh", nil},
		{"models-dir must not match --model prefix", "codex", "codex --models-dir x", nil},
		{"bare -m", "codex", "codex -m", []string{"--model (-m)"}},
		{"double-dash-m must not match", "codex", "codex --m", nil},
		{"claude has no -m short flag", "claude", "claude -m x", nil},
		{"claude has no --sandbox flag", "claude", "claude --sandbox x", nil},
		{"codex has no --permission-mode flag", "codex", "codex --permission-mode x", nil},
		{"multiple flags in order", "codex", "codex -m x -s y", []string{"--model (-m)", "--sandbox (-s)"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shellAliasConflictingFlags(tc.alias, tc.value)
			if len(got) != len(tc.want) {
				t.Fatalf("shellAliasConflictingFlags(%q, %q) = %v, want %v", tc.alias, tc.value, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("shellAliasConflictingFlags(%q, %q)[%d] = %q, want %q", tc.alias, tc.value, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault proves the TestMain
// seam (main_test.go) works end to end: without any test-local override,
// checkShellAliases resolves through doctorShellAliasEnv, which TestMain has
// pinned to a non-existent home directory -- so the check reads "pass" and
// "no shell rc file found to scan" regardless of the developer's real rc
// files (self-review L7).
func TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault(t *testing.T) {
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

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runDoctorOpts(dir, false)
	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("runDoctorOpts returned error: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(string(out), "Shell aliases (codex/claude): pass") {
		t.Errorf("expected a pass-level Shell aliases (codex/claude) line in output:\n%s", out)
	}
	if !strings.Contains(string(out), "no shell rc file found to scan") {
		t.Errorf("expected the pinned no-rc-file detail in output:\n%s", out)
	}
}

// TestRunDoctorOpts_ShellAliasesCheck_WarnsThroughTheSeam overrides the
// doctorShellAliasEnv seam (not HOME/ZDOTDIR) to prove runDoctorOpts wires
// checkShellAliases' result into its printed output end to end, and that a
// warn-level finding does not flip the exit code.
func TestRunDoctorOpts_ShellAliasesCheck_WarnsThroughTheSeam(t *testing.T) {
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
	// herdr present, so a codex finding must warn.
	for _, bin := range []string{"claude", "codex", "go", "herdr"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(dir, "no-such-agmsg-home"))

	homeDir := t.TempDir()
	writeAliasRc(t, filepath.Join(homeDir, ".zshrc"), `alias codex="codex -m x"`+"\n")

	orig := doctorShellAliasEnv
	doctorShellAliasEnv = shellAliasTestEnv(homeDir)
	t.Cleanup(func() { doctorShellAliasEnv = orig })

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runDoctorOpts(dir, false)
	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("runDoctorOpts returned error with only a warn-level shell-alias finding: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(string(out), "Shell aliases (codex/claude): warn") {
		t.Errorf("expected a warn-level Shell aliases (codex/claude) line in output:\n%s", out)
	}
}
