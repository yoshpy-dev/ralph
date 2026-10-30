package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckLegacyAgmsgHomeEnv pins AC9 of plan 2026-10-29-local-config-org
// at the check (checkRetiredEnv's RALPH_ORG_AGMSG_HOME row): no line for an
// empty or unset RALPH_ORG_AGMSG_HOME, and one warn otherwise that names the
// value, says ralph no longer reads it, and says where the value goes now
// and in which forms.
func TestCheckLegacyAgmsgHomeEnv(t *testing.T) {
	agmsgHomeOnly := func(value string) func(string) string {
		return func(name string) string {
			if name == legacyAgmsgHomeEnv {
				return value
			}
			return ""
		}
	}
	if got := checkRetiredEnv(agmsgHomeOnly("")); len(got) != 0 {
		t.Errorf("checkRetiredEnv with RALPH_ORG_AGMSG_HOME empty = %+v; want no line", got)
	}
	got := checkRetiredEnv(agmsgHomeOnly("rel/agmsg"))
	want := checkResult{
		Name:   "RALPH_ORG_AGMSG_HOME",
		Status: "warn",
		Detail: `set to "rel/agmsg", but ralph no longer reads it; write the value in [org].agmsg_home of ralph.local.toml ` +
			"(next to ralph.toml in the main worktree root, ignored by git), as an absolute path or one that starts with ~/, " +
			"and unset RALPH_ORG_AGMSG_HOME",
	}
	if len(got) != 1 || got[0] != want {
		t.Errorf("checkRetiredEnv with RALPH_ORG_AGMSG_HOME=%q = %+v; want [%+v]", "rel/agmsg", got, want)
	}
}

// TestRunDoctorOpts_LegacyAgmsgHomeEnv pins AC9 (and AC6 for doctor) of
// plan 2026-10-29-local-config-org through runDoctorOpts on an initialized
// project with claude, codex, and go stubbed: with RALPH_ORG_AGMSG_HOME set
// to a directory that holds an agmsg install, doctor prints the warn line
// for it and still exits 0, and its agmsg check looks at [org].agmsg_home of
// ralph.toml (a directory with no agmsg), not at the variable's directory.
// With the variable unset, no line names it.
func TestRunDoctorOpts_LegacyAgmsgHomeEnv(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	if err := executeInit(dir, initConfig{ProjectName: "test", Packs: []string{"golang"}}, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	tomlHome := filepath.Join(dir, "no-such-agmsg-home")
	tomlPath := filepath.Join(dir, "ralph.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "[org]") {
		t.Fatalf("the test scaffold's ralph.toml has an [org] table already:\n%s", data)
	}
	writeTestFile(t, tomlPath, string(data)+fmt.Sprintf("\n[org]\nagmsg_home = %q\n", tomlHome))

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	setAgmsgHomeOverride(t, "")
	envHome := filepath.Join(t.TempDir(), "env-agmsg")
	writeAgmsgHome(t, envHome, agmsgTestedVersion)

	runDoctor := func() (string, error) {
		t.Helper()
		origStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w
		runErr := runDoctorOpts(dir, false)
		_ = w.Close()
		os.Stdout = origStdout
		out, _ := io.ReadAll(r)
		return string(out), runErr
	}

	t.Run("set", func(t *testing.T) {
		t.Setenv(legacyAgmsgHomeEnv, envHome)
		out, err := runDoctor()
		if err != nil {
			t.Fatalf("a RALPH_ORG_AGMSG_HOME warning must not fail doctor: %v\noutput:\n%s", err, out)
		}
		if want := fmt.Sprintf("⚠ RALPH_ORG_AGMSG_HOME: warn — set to %q, but ralph no longer reads it; write the value in [org].agmsg_home of ralph.local.toml", envHome); !strings.Contains(out, want) {
			t.Errorf("doctor output does not contain %q:\n%s", want, out)
		}
		if want := "agmsg: warn — agmsg not installed at " + tomlHome; !strings.Contains(out, want) {
			t.Errorf("doctor output does not contain %q (the agmsg check must use [org].agmsg_home, not the variable):\n%s", want, out)
		}
		if strings.Contains(out, "available at "+envHome) {
			t.Errorf("doctor's agmsg check used the variable's directory %s:\n%s", envHome, out)
		}
	})
	t.Run("unset", func(t *testing.T) {
		setOrUnsetEnv(t, legacyAgmsgHomeEnv, "", true)
		out, err := runDoctor()
		if err != nil {
			t.Fatalf("doctor: %v\noutput:\n%s", err, out)
		}
		if strings.Contains(out, legacyAgmsgHomeEnv) {
			t.Errorf("doctor output names %s while it is unset:\n%s", legacyAgmsgHomeEnv, out)
		}
	})
	t.Run("empty", func(t *testing.T) {
		t.Setenv(legacyAgmsgHomeEnv, "")
		out, err := runDoctor()
		if err != nil {
			t.Fatalf("doctor: %v\noutput:\n%s", err, out)
		}
		if strings.Contains(out, legacyAgmsgHomeEnv) {
			t.Errorf("doctor output names %s while it is empty:\n%s", legacyAgmsgHomeEnv, out)
		}
	})
}
