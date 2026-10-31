package cli

import (
	"fmt"
	"strings"
	"testing"
)

// Tests for doctor's warn about the five variables the project's scripts
// read before the local-only keys of [plan_visual], [standard], and
// [verify] took their place (checkRetiredEnv), plan
// 2026-10-30-script-local-config, D6 and AC5. TestMain unsets them, so each
// test sets them with t.Setenv or through the getenv it passes.

// retiredScriptEnvCase is one of the five variables: a non-empty value, the
// key whose place it took, and the script that read it.
type retiredScriptEnvCase struct {
	name, value, key, script string
}

var retiredScriptEnvCases = []retiredScriptEnvCase{
	{"RALPH_PLAN_VISUAL_BROWSER", "/opt/chromium/chrome", "[plan_visual].browser", "plan-visual.sh"},
	{"RALPH_PLAN_VISUAL_OPENER", "xdg-open", "[plan_visual].opener", "plan-visual.sh"},
	{"RALPH_XREVIEW_BASE", "develop", "[standard].base_branch", "xreview-helpers.sh"},
	{"RALPH_VERIFY_BASE", "HEAD~1", "[verify].base", "detect-changed-languages.sh"},
	{"RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN", "1", "[verify].skip_branch_secret_scan", "run-verify.sh"},
}

// retiredScriptEnvDetail is the Detail of the warn about c: its value, that
// ralph no longer reads it once `ralph upgrade` has replaced
// scripts/ralph-config.sh and the script that reads it until then, its key
// in ralph.local.toml only (the keys are local-only), for the scan skip that
// the value to write is true for 1 and nothing for any other, and the unset.
func retiredScriptEnvDetail(c retiredScriptEnvCase) string {
	more := ""
	if c.name == "RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN" {
		more = ", as true only if it is 1; any other value did not skip the scan, so write nothing (the key takes only true or false)"
	}
	return fmt.Sprintf("set to %q, but ralph no longer reads it, once `ralph upgrade` has replaced the project's "+
		"scripts/ralph-config.sh and scripts/%s, which reads it until then; write the value in %s of ralph.local.toml "+
		"(next to ralph.toml in the main worktree root, ignored by git)%s, and unset %s",
		c.value, c.script, c.key, more, c.name)
}

// TestCheckRetiredEnv_ScriptVariables covers AC5 at the check: each of the
// five variables set to a non-empty value gives one warn that names the
// value, the script that reads it until the upgrade, and the key of
// ralph.local.toml that took its place, never ralph.toml; empty or unset,
// it gives none.
func TestCheckRetiredEnv_ScriptVariables(t *testing.T) {
	for _, c := range retiredScriptEnvCases {
		t.Run(c.name, func(t *testing.T) {
			only := func(value string) func(string) string {
				return func(name string) string {
					if name == c.name {
						return value
					}
					return ""
				}
			}
			want := checkResult{Name: c.name, Status: "warn", Detail: retiredScriptEnvDetail(c)}
			got := checkRetiredEnv(only(c.value))
			if len(got) != 1 || got[0] != want {
				t.Errorf("checkRetiredEnv with %s=%q = %+v\nwant [%+v]", c.name, c.value, got, want)
			}
			if len(got) == 1 && strings.Contains(got[0].Detail, "of ralph.toml") {
				t.Errorf("the warn about %s points at ralph.toml, where Load refuses %s:\n%s", c.name, c.key, got[0].Detail)
			}
			if got := checkRetiredEnv(only("")); len(got) != 0 {
				t.Errorf("checkRetiredEnv with %s empty = %+v; want no line", c.name, got)
			}
		})
	}
}

// TestCheckRetiredEnv_ScanSkipOtherThanOne covers self-review L3 of plan
// 2026-10-30-script-local-config: RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN
// skipped the scan only for the value 1, and [verify].skip_branch_secret_scan
// takes only true or false. For any other value the warn must not have the
// value written in the key (= 0 makes config.Load refuse the file, and so
// every key fails), so it says to write true only for 1 and nothing
// otherwise.
func TestCheckRetiredEnv_ScanSkipOtherThanOne(t *testing.T) {
	for _, value := range []string{"0", "yes", "true"} {
		t.Run(value, func(t *testing.T) {
			c := retiredScriptEnvCase{"RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN", value, "[verify].skip_branch_secret_scan", "run-verify.sh"}
			got := checkRetiredEnv(func(name string) string {
				if name == c.name {
					return value
				}
				return ""
			})
			want := checkResult{Name: c.name, Status: "warn", Detail: retiredScriptEnvDetail(c)}
			if len(got) != 1 || got[0] != want {
				t.Fatalf("checkRetiredEnv with %s=%q = %+v\nwant [%+v]", c.name, value, got, want)
			}
			for _, phrase := range []string{
				"as true only if it is 1;",
				"any other value did not skip the scan, so write nothing (the key takes only true or false)",
			} {
				if !strings.Contains(got[0].Detail, phrase) {
					t.Errorf("the warn about %s=%q does not limit true to 1 (missing %q):\n%s", c.name, value, phrase, got[0].Detail)
				}
			}
		})
	}
}

// TestCheckRetiredEnv_AllElevenInTableOrder covers the order of every
// retired variable: with all eleven set, there is one warn for each, the
// two org ones first, then the four standard-flow ones (plan
// 2026-10-30-standard-flow-config, AC6), then the five script ones (AC5),
// in retiredEnvs' order.
func TestCheckRetiredEnv_AllElevenInTableOrder(t *testing.T) {
	env := map[string]string{
		legacyAgmsgHomeEnv:    "/agmsg",
		retiredOrgStateDirEnv: "/old ledger",
	}
	for _, c := range retiredStandardEnvCases {
		env[c.name] = c.value
	}
	for _, c := range retiredScriptEnvCases {
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
		"RALPH_CLAUDE_REVIEWER_MODEL", "RALPH_CODEX_REVIEWER_MODEL", "RALPH_CODEX_REASONING_EFFORT",
		"RALPH_PLAN_VISUAL_BROWSER", "RALPH_PLAN_VISUAL_OPENER", "RALPH_XREVIEW_BASE", "RALPH_VERIFY_BASE",
		"RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("rows = %q; want %q", names, want)
	}
}

// TestRunDoctorOpts_RetiredScriptEnv covers AC5 through runDoctorOpts on an
// initialized project with claude, codex, and go stubbed (doctorMGProject):
// with the five variables set, doctor prints a warn line for each and still
// exits 0; unset or empty, no line names any of them.
func TestRunDoctorOpts_RetiredScriptEnv(t *testing.T) {
	dir, _ := doctorMGProject(t)

	t.Run("set", func(t *testing.T) {
		for _, c := range retiredScriptEnvCases {
			t.Setenv(c.name, c.value)
		}
		var out string
		if err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) }); err != nil {
			t.Fatalf("a retired script variable must not fail doctor: %v\noutput:\n%s", err, out)
		}
		for _, c := range retiredScriptEnvCases {
			if want := "⚠ " + c.name + ": warn — " + retiredScriptEnvDetail(c); !strings.Contains(out, want) {
				t.Errorf("doctor output does not contain %q:\n%s", want, out)
			}
		}
	})
	for _, tc := range []struct {
		name  string
		unset bool
	}{{"unset", true}, {"empty", false}} {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range retiredScriptEnvCases {
				setOrUnsetEnv(t, c.name, "", tc.unset)
			}
			var out string
			if err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) }); err != nil {
				t.Fatalf("doctor: %v\noutput:\n%s", err, out)
			}
			for _, c := range retiredScriptEnvCases {
				if strings.Contains(out, c.name) {
					t.Errorf("doctor output names %s while it is %s:\n%s", c.name, tc.name, out)
				}
			}
		})
	}
}
