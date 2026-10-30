package cli

import (
	"fmt"
	"strings"
	"testing"
)

// Tests for doctor's warn about the four variables the standard flow's
// skills read before [standard] and [reviewer] took their place
// (checkRetiredEnv), plan 2026-10-30-standard-flow-config, D7 and AC6.
// TestMain unsets them, so each test sets them with t.Setenv or through
// the getenv it passes.

// retiredStandardEnvCase is one of the four variables: a non-empty value
// and the key whose place it took.
type retiredStandardEnvCase struct {
	name, value, key string
}

var retiredStandardEnvCases = []retiredStandardEnvCase{
	{"RALPH_STANDARD_MAX_PIPELINE_CYCLES", "3", "[standard].max_pipeline_cycles"},
	{"RALPH_CLAUDE_REVIEWER_MODEL", "opus", "[reviewer].claude_model"},
	{"RALPH_CODEX_REVIEWER_MODEL", "gpt-6-astra", "[reviewer].codex_model"},
	{"RALPH_CODEX_REASONING_EFFORT", "high", "[reviewer].codex_effort"},
}

// retiredStandardEnvDetail is the Detail of the warn about c: its value,
// that ralph no longer reads it, its key in ralph.toml or ralph.local.toml,
// for the cap the way to raise it for one task, and the unset.
func retiredStandardEnvDetail(c retiredStandardEnvCase) string {
	more := ""
	if c.name == "RALPH_STANDARD_MAX_PIPELINE_CYCLES" {
		more = ", or, to raise the cap of one task only, have /cross-review write cap in that task's " +
			".harness/state/standard-pipeline/cycle-count.json when it reaches the cap"
	}
	return fmt.Sprintf("set to %q, but ralph no longer reads it; write the value in %s of ralph.toml, "+
		"or of ralph.local.toml (next to ralph.toml in the main worktree root, ignored by git) for this machine only%s, "+
		"and unset %s", c.value, c.key, more, c.name)
}

// TestCheckRetiredEnv_StandardFlowVariables covers AC6 at the check: each
// of the four variables set to a non-empty value gives one warn that names
// the value and the key that took its place, and the cap's also names
// cycle-count.json's cap; empty or unset, it gives none.
func TestCheckRetiredEnv_StandardFlowVariables(t *testing.T) {
	for _, c := range retiredStandardEnvCases {
		t.Run(c.name, func(t *testing.T) {
			only := func(value string) func(string) string {
				return func(name string) string {
					if name == c.name {
						return value
					}
					return ""
				}
			}
			want := checkResult{Name: c.name, Status: "warn", Detail: retiredStandardEnvDetail(c)}
			if got := checkRetiredEnv(only(c.value)); len(got) != 1 || got[0] != want {
				t.Errorf("checkRetiredEnv with %s=%q = %+v\nwant [%+v]", c.name, c.value, got, want)
			}
			if got := checkRetiredEnv(only("")); len(got) != 0 {
				t.Errorf("checkRetiredEnv with %s empty = %+v; want no line", c.name, got)
			}
		})
	}
	if got := checkRetiredEnv(func(string) string { return "" }); len(got) != 0 {
		t.Errorf("checkRetiredEnv with nothing set = %+v; want no line", got)
	}
}

// TestCheckRetiredEnv_AllSixInTableOrder covers AC6's order: with all six
// retired variables set, there is one warn for each, the two org ones
// first and the four standard-flow ones after them, in retiredEnvs' order.
func TestCheckRetiredEnv_AllSixInTableOrder(t *testing.T) {
	env := map[string]string{
		legacyAgmsgHomeEnv:    "/agmsg",
		retiredOrgStateDirEnv: "/old ledger",
	}
	for _, c := range retiredStandardEnvCases {
		env[c.name] = c.value
	}
	got := checkRetiredEnv(func(name string) string { return env[name] })
	var names []string
	for _, r := range got {
		names = append(names, r.Name)
		if r.Status != "warn" {
			t.Errorf("%s row status = %q; want warn", r.Name, r.Status)
		}
	}
	want := []string{"RALPH_ORG_AGMSG_HOME", "RALPH_ORG_STATE_DIR", "RALPH_STANDARD_MAX_PIPELINE_CYCLES",
		"RALPH_CLAUDE_REVIEWER_MODEL", "RALPH_CODEX_REVIEWER_MODEL", "RALPH_CODEX_REASONING_EFFORT"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("rows = %q; want %q", names, want)
	}
}

// TestRunDoctorOpts_RetiredStandardFlowEnv covers AC6 through runDoctorOpts
// on an initialized project with claude, codex, and go stubbed
// (doctorMGProject): with the four variables set, doctor prints a warn line
// for each and still exits 0; unset or empty, no line names any of them.
func TestRunDoctorOpts_RetiredStandardFlowEnv(t *testing.T) {
	dir, _ := doctorMGProject(t)

	t.Run("set", func(t *testing.T) {
		for _, c := range retiredStandardEnvCases {
			t.Setenv(c.name, c.value)
		}
		var out string
		if err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) }); err != nil {
			t.Fatalf("a retired standard-flow variable must not fail doctor: %v\noutput:\n%s", err, out)
		}
		for _, c := range retiredStandardEnvCases {
			if want := "⚠ " + c.name + ": warn — " + retiredStandardEnvDetail(c); !strings.Contains(out, want) {
				t.Errorf("doctor output does not contain %q:\n%s", want, out)
			}
		}
	})
	for _, tc := range []struct {
		name  string
		unset bool
	}{{"unset", true}, {"empty", false}} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range retiredStandardEnvCases {
				setOrUnsetEnv(t, c.name, "", tc.unset)
			}
			var out string
			if err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) }); err != nil {
				t.Fatalf("doctor: %v\noutput:\n%s", err, out)
			}
			for _, c := range retiredStandardEnvCases {
				if strings.Contains(out, c.name) {
					t.Errorf("doctor output names %s while it is %s:\n%s", c.name, tc.name, out)
				}
			}
		})
	}
}
