package org

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// Tests of the claude retry entries of a headless director (spec
// docs/specs/2026-11-07-org-seat-resume.md, FR-8 and FR-9; plan
// 2026-11-07-org-seat-resume, AC-8 and S3): its herdr pane and the launch
// script of a launch_cmd start. testOrgConfig sets no
// [org].api_wait_minutes, so each test sets it.

// directorPermsFor is a permission config under which a director of
// driverName starts: a codex director runs guarded only.
func directorPermsFor(driverName string) config.OrgPermissionsConfig {
	if driverName == DirectorDriverCodex {
		return config.OrgPermissionsConfig{CodexVerified: true, Roles: map[string]string{DirectorRole: PermissionModeGuarded}}
	}
	return config.OrgPermissionsConfig{}
}

// directorModelFor is the model the tests start a director of driverName
// with.
func directorModelFor(driverName string) string {
	if driverName == DirectorDriverCodex {
		return "gpt-5-codex"
	}
	return "opus"
}

// TestDirectorStart_ClaudeRetryEnvFollowsTheDirectorsDriver pins AC-8 for a
// headless director in herdr: with [org].api_wait_minutes 30 the
// WorkspaceCreate that makes a claude director's pane gets the three claude
// retry entries as its extraEnv, and no RALPH_PRIMARY_CLI; a codex
// director's gets none, and so does a claude director's with the setting 0.
func TestDirectorStart_ClaudeRetryEnvFollowsTheDirectorsDriver(t *testing.T) {
	for _, tc := range []struct {
		name, driver string
		minutes      int
		want         []string
	}{
		{"claude with 30 minutes", DirectorDriverClaude, 30, claudeRetryEnv30},
		{"codex with 30 minutes", DirectorDriverCodex, 30, nil},
		{"claude with the setting 0", DirectorDriverClaude, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.o.Config.Permissions = directorPermsFor(tc.driver)
			f.o.Config.APIWaitMinutes = tc.minutes
			f.mustStart(t, tc.driver, directorModelFor(tc.driver))

			got := f.h.workspaceCreateEnvs
			if len(got) != 1 || !slices.Equal(got[0], tc.want) || (got[0] == nil) != (tc.want == nil) {
				t.Fatalf("WorkspaceCreate extraEnv per call = %q; want one call with %q", got, tc.want)
			}
		})
	}
}

// claudeRetryExports is what the launch script writes for the entries of
// driver.ClaudeRetryEnv(30), each value single-quoted and then exported.
const claudeRetryExports = "CLAUDE_CODE_RETRY_WATCHDOG='1'\nexport CLAUDE_CODE_RETRY_WATCHDOG\n" +
	"CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS='1800000'\nexport CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS\n" +
	"CLAUDE_CODE_MAX_RETRIES='55'\nexport CLAUDE_CODE_MAX_RETRIES\n"

// TestDirectorLaunchScript_ExportsTheClaudeRetryEnvAfterSeatPaneEnv pins
// AC-8 for the launch script's text: its RetryEnv entries are exported, in
// their order, right after the SeatPaneEnv entries and before the
// check-in; with no RetryEnv the script names no CLAUDE_CODE_ variable.
func TestDirectorLaunchScript_ExportsTheClaudeRetryEnvAfterSeatPaneEnv(t *testing.T) {
	s := testLaunchScript("/b")
	s.RetryEnv = driver.ClaudeRetryEnv(30)
	text, err := s.text()
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	if want := "\nRALPH_WIP_CHECKPOINT='off'\nexport RALPH_WIP_CHECKPOINT\n" + claudeRetryExports +
		"'/b/bin dir'\\''s/ralph' org director launched "; !strings.Contains(text, want) {
		t.Fatalf("launch script =\n%s\nwant it to hold\n%s", text, want)
	}

	s.RetryEnv = nil
	if text, err = s.text(); err != nil {
		t.Fatalf("text: %v", err)
	}
	if strings.Contains(text, "CLAUDE_CODE_") {
		t.Fatalf("a launch script with no RetryEnv names a CLAUDE_CODE_ variable:\n%s", text)
	}
}

// TestDirectorLaunchScript_TheAgentGetsTheClaudeRetryEnv runs the script
// through sh: the agent it execs sees the values the script exports, over
// other values of the same variables in the terminal's environment, and
// with none there.
func TestDirectorLaunchScript_TheAgentGetsTheClaudeRetryEnv(t *testing.T) {
	for name, env := range map[string][]string{
		"the terminal passes none": {"PATH=/usr/bin:/bin"},
		"the terminal passes other values": {"PATH=/usr/bin:/bin", "CLAUDE_CODE_RETRY_WATCHDOG=0",
			"CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1", "CLAUDE_CODE_MAX_RETRIES=2"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			s, _, _, _, agentOut := launchScriptFakes(t, dir, 0)
			s.RetryEnv = driver.ClaudeRetryEnv(30)
			writeTestExecutable(t, s.AgentPath, "#!/bin/sh\nprintf '%s\\n' "+
				"\"CLAUDE_CODE_RETRY_WATCHDOG=${CLAUDE_CODE_RETRY_WATCHDOG-<unset>}\" "+
				"\"CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=${CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS-<unset>}\" "+
				"\"CLAUDE_CODE_MAX_RETRIES=${CLAUDE_CODE_MAX_RETRIES-<unset>}\" > "+shellQuote(agentOut)+"\n")
			path, err := writeDirectorLaunchScript(s)
			if err != nil {
				t.Fatalf("writeDirectorLaunchScript: %v", err)
			}
			if code := runLaunchScript(t, path, env); code != 0 {
				t.Fatalf("the launch script exited %d, want 0", code)
			}
			if got := readLines(t, agentOut); !slices.Equal(got, claudeRetryEnv30) {
				t.Fatalf("the agent got %q, want %q", got, claudeRetryEnv30)
			}
		})
	}
}

// TestStartDirector_LaunchCmdScriptCarriesTheClaudeRetryEnvOfItsDriver pins
// AC-8 for a launch_cmd start: the launch script start writes exports the
// three claude retry entries for a claude director with
// [org].api_wait_minutes 30, and names no CLAUDE_CODE_ variable for a codex
// director or with the setting 0. launch_cmd fails here before any terminal
// runs the script, which start leaves on disk
// (TestStartDirector_ALaunchCmdThatFailsBeforeACheckInIsRecordedStopped).
func TestStartDirector_LaunchCmdScriptCarriesTheClaudeRetryEnvOfItsDriver(t *testing.T) {
	for _, tc := range []struct {
		name, driver string
		minutes      int
		wantExports  bool
	}{
		{"claude with 30 minutes", DirectorDriverClaude, 30, true},
		{"codex with 30 minutes", DirectorDriverCodex, 30, false},
		{"claude with the setting 0", DirectorDriverClaude, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCmdStartFixture(t, []string{"sh", "-c", "exit 3", "sh", "{script}"}, cmdStartOptions{})
			f.o.Config.Permissions = directorPermsFor(tc.driver)
			f.o.Config.APIWaitMinutes = tc.minutes
			p := f.params(tc.driver, directorModelFor(tc.driver))
			p.TimeoutMS = 30000
			if _, err := f.o.StartDirector(p); err == nil || !strings.Contains(err.Error(), "exit status 3") {
				t.Fatalf("StartDirector = %v, want the failure of launch_cmd", err)
			}
			raw, err := os.ReadFile(DirectorLaunchScriptPath(f.dir, directorTestToken))
			if err != nil {
				t.Fatalf("read the launch script: %v", err)
			}
			script := string(raw)
			if !tc.wantExports {
				if strings.Contains(script, "CLAUDE_CODE_") {
					t.Fatalf("the launch script names a CLAUDE_CODE_ variable, want none:\n%s", script)
				}
				return
			}
			at := strings.Index(script, "\nexport RALPH_WIP_CHECKPOINT\n"+claudeRetryExports)
			checkIn := strings.Index(script, " org director launched ")
			if at < 0 || checkIn < at {
				t.Fatalf("the launch script does not export the claude retry entries right after RALPH_WIP_CHECKPOINT and before the check-in:\n%s", script)
			}
		})
	}
}
