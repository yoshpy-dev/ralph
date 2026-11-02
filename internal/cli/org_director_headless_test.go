package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org director start` and `stop`, the headless director in
// director status, and the spawn refusal of role director (plan
// docs/plans/active/2026-10-15-org-director-start.md, AC1 to AC3, AC5 to
// AC11, AC13, and AC14 through the CLI). Every test runs the stub herdr
// (setupOrgStubPATH, which also pins HERDR_PANE_ID and HERDR_WORKSPACE_ID
// unset), never herdr itself, on a ledger under t.TempDir().

// directorStartLedger is a ledger for director start under t.TempDir(), with
// the stub herdr on PATH and a working directory of its own with no
// ralph.toml, which the director runs in (the ledger is not a main
// worktree's). It returns the state dir, the herdr stub's log, and that
// directory as os.Getwd spells it.
func directorStartLedger(t *testing.T) (stateDir, herdrLog, cwd string) {
	t.Helper()
	herdrLog, _ = setupOrgStubPATH(t)
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return t.TempDir(), herdrLog, cwd
}

// writeDirectorConfig writes content as a ralph.toml under t.TempDir() and
// returns its path, for --config.
func writeDirectorConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, path, content)
	return path
}

// startDirectorOK runs director start with args on the ledger stateDir,
// fails the test when it fails, and returns the token of the last attached
// event, the registration start made.
func startDirectorOK(t *testing.T, stateDir string, args ...string) (token string) {
	t.Helper()
	runInboxOK(t, append([]string{"director", "start", "--state-dir", stateDir}, args...)...)
	for _, ev := range directorEventsIn(t, stateDir) {
		if ev.Event == org.DirectorEventAttached {
			token = ev.Token
		}
	}
	if token == "" {
		t.Fatalf("no attached event with a token in %s", stateDir)
	}
	return token
}

// seedHeadless records a headless registration in the ledger stateDir
// through the store, as start does around its herdr calls: attached (by
// alice in /work, permission autonomous), then the workspace and the pane
// when given, then launched when launched is set. It runs no herdr and
// returns the registration as attached (with its token).
func seedHeadless(t *testing.T, stateDir, driverName, model, workspaceID, paneID string, launched bool) org.DirectorRegistration {
	t.Helper()
	store := org.NewDirectorStore(stateDir)
	reg, err := store.AttachHeadless(org.DirectorHeadlessAttach{
		Driver: driverName, Model: model, PermissionMode: org.PermissionModeAutonomous, Wake: []string{org.DirectorWakePull}, User: "alice", Cwd: "/work",
	})
	if err != nil {
		t.Fatalf("seed the headless registration: %v", err)
	}
	who := org.DirectorWho{User: "alice", Cwd: "/work"}
	if workspaceID != "" {
		if err := store.RecordWorkspace(reg.ID, workspaceID, who); err != nil {
			t.Fatalf("seed the workspace: %v", err)
		}
	}
	if paneID != "" {
		if err := store.RecordPane(reg.ID, paneID, who); err != nil {
			t.Fatalf("seed the pane: %v", err)
		}
	}
	if launched {
		if ok, err := store.MarkLaunched(reg.ID, who); err != nil || !ok {
			t.Fatalf("seed launched: %v, %v", ok, err)
		}
	}
	return reg
}

// herdrActions returns the lines of the herdr stub's log that send a key or
// close something: what a stop that refuses must not run.
func herdrActions(t *testing.T, herdrLog string) []string {
	t.Helper()
	var actions []string
	for _, line := range readLogLines(t, herdrLog) {
		if strings.HasPrefix(line, "pane send-keys ") || strings.HasPrefix(line, "pane close ") || strings.HasPrefix(line, "workspace close ") {
			actions = append(actions, line)
		}
	}
	return actions
}

// resetStubHerdrState empties the herdr stub's state, as a herdr that lost
// its state does: its next workspace, tab, and pane get the ids it handed
// out first again (ws-stub-1, pane-stub-1).
func resetStubHerdrState(t *testing.T) {
	t.Helper()
	dir := os.Getenv("ORG_STUB_HERDR_STATE")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
}

// AC1 and AC8 through the CLI: director start, against the stub herdr,
// records attached (launch herdr, a token, the model, the permission mode,
// and who ran it), workspace_created, pane_created, and launched, runs
// herdr's workspace create, tab create, and agent start with the token's
// names, the permission flags, --model, and the prompt pointer in that
// order, writes the prompt file, and prints the registration, the pane, and
// the stop command with --state-dir. A codex director runs guarded only
// (D12): no permission flag and no --add-dir, though the ledger is outside
// the cwd. Nothing goes to the manifest or the model receipts. director
// status then shows it running with its pane, workspace, and model, and
// stop closes the pane and the workspace and ends the registration.
func TestOrgDirectorStart_LaunchesRegistersAndStops(t *testing.T) {
	for _, tc := range []struct {
		driver, model string
		config        string // the ralph.toml --config names ("" for none)
		mode          string
		permArgs      string // the permission flags, each followed by a space
	}{
		{"claude", "opus", "", org.PermissionModeAutonomous, "--permission-mode bypassPermissions "},
		{"codex", "gpt-6-astra", "[org.permissions]\ncodex_verified = true\n\n[org.permissions.roles]\ndirector = \"guarded\"\n", org.PermissionModeGuarded, ""},
	} {
		t.Run(tc.driver, func(t *testing.T) {
			stateDir, herdrLog, cwd := directorStartLedger(t)
			t.Setenv("USER", "alice")
			args := []string{"director", "start", "--driver", tc.driver, "--model", tc.model, "--state-dir", stateDir}
			if tc.config != "" {
				args = append(args, "--config", writeDirectorConfig(t, tc.config))
			}
			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if err != nil {
				t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			events := directorEventsIn(t, stateDir)
			if got, want := directorEventKinds(events), []string{"attached d1", "workspace_created d1", "pane_created d1", "launched d1"}; !slices.Equal(got, want) {
				t.Fatalf("director.jsonl = %v; want %v", got, want)
			}
			att := events[0]
			if att.Launch != org.DirectorLaunchHerdr || len(att.Token) != 12 || att.Model != tc.model || att.PermissionMode != tc.mode ||
				att.Driver != tc.driver || att.User != "alice" || att.Cwd != cwd || att.HerdrPane || att.PaneID != "" {
				t.Errorf("attached = %+v; want launch herdr, a token, %s, %s, by alice in %s, no pane", att, tc.model, tc.mode, cwd)
			}
			if events[1].WorkspaceID != "ws-stub-1" || events[2].PaneID != "pane-stub-1" {
				t.Errorf("workspace_created and pane_created = %+v, %+v; want ws-stub-1 and pane-stub-1", events[1], events[2])
			}
			tok := att.Token
			suffix := " --state-dir '" + stateDir + "'"
			wantOut := "started d1 (driver=" + tc.driver + ", model=" + tc.model + ", permission=" + tc.mode + ", pane=pane-stub-1, workspace=ws-stub-1)\n" +
				"talk to the director in herdr: pane pane-stub-1, in the tab director_" + tok + " of the workspace labelled ralph_director_" + tok + "\n" +
				"stop it with: ralph org director stop --id d1" + suffix + "\n"
			if stdout != wantOut {
				t.Errorf("stdout =\n%s\nwant\n%s", stdout, wantOut)
			}
			if stderr != "" {
				t.Errorf("stderr = %q; want nothing with --model given", stderr)
			}
			prompt := filepath.Join(stateDir, "prompts", "director_"+tok+".md")
			if data, err := os.ReadFile(prompt); err != nil || !strings.Contains(string(data), "/director") {
				t.Errorf("prompt file %s = %v; want the director prompt", prompt, err)
			}
			wantHerdr := []string{
				"workspace create --cwd " + cwd + " --label ralph_director_" + tok,
				"tab create --workspace ws-stub-1 --cwd " + cwd + " --label director_" + tok + " --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off",
				"agent start director_" + tok + " --kind " + tc.driver + " --pane pane-stub-1 --timeout 60000 -- " + tc.permArgs +
					"--model " + tc.model + " " + org.PromptFilePointer(prompt),
			}
			if got := readLogLines(t, herdrLog); !slices.Equal(got, wantHerdr) {
				t.Errorf("herdr calls =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantHerdr, "\n"))
			}
			for _, path := range []string{org.ManifestPathIn(stateDir), org.ReceiptsPathIn(stateDir)} {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("stat %s = %v; want nothing written to it", path, err)
				}
			}

			jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
			report := directorReportOf(t, jsonOut)
			if reg := report.Registration; report.State != "attached" || reg == nil || reg.ID != "d1" || reg.Launch != org.DirectorLaunchHerdr ||
				reg.State != org.DirectorStateRunning || reg.PaneID != "pane-stub-1" || reg.WorkspaceID != "ws-stub-1" || reg.Model != tc.model ||
				reg.Token != tok || reg.PermissionMode != tc.mode || len(report.OpenHeadless) != 0 {
				t.Errorf("director status --json = %s; want d1 running in pane-stub-1 and ws-stub-1 with %s", jsonOut, tc.model)
			}

			stdout, _ = runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
			if want := "stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)\nthe director is no longer attached\n"; stdout != want {
				t.Errorf("director stop stdout = %q; want %q", stdout, want)
			}
			if got, want := herdrActions(t, herdrLog), []string{"pane send-keys pane-stub-1 C-c", "pane close pane-stub-1", "workspace close ws-stub-1"}; !slices.Equal(got, want) {
				t.Errorf("herdr actions of stop = %v; want %v", got, want)
			}
			if got := directorEventKinds(directorEventsIn(t, stateDir)); !slices.Equal(got[4:], []string{"detached d1", "stopped d1"}) {
				t.Errorf("director.jsonl after stop = %v; want detached d1 and stopped d1 appended", got)
			}
			if got := directorStatusLine(t, "--state-dir", stateDir); got != directorNotAttachedLine {
				t.Errorf("director line after stop = %q; want %q", got, directorNotAttachedLine)
			}
		})
	}
}

// AC2 and AC6 through the CLI: director start refuses, with exit 1, nothing
// on stdout, director.jsonl as it was, and no herdr call, when
// [org.director].enabled is false (naming the file), for a --model
// [org.roles].director does not allow, for codex autonomous and edits
// whatever codex_verified says (D12), while a promoted session is attached
// (naming detach), while a headless registration is still starting (naming
// stop), and inside a herdr pane.
func TestOrgDirectorStart_RefusedWithNothingWritten(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, stateDir string) (args []string)
		is    error // a sentinel the error wraps (nil: none checked)
		wants func(stateDir string) []string
	}{
		{"enabled false", func(t *testing.T, _ string) []string {
			return []string{"--driver", "claude", "--model", "opus", "--config", writeDirectorConfig(t, "[org.director]\nenabled = false\n")}
		}, nil, func(string) []string {
			return []string{"org: director start: [org.director].enabled is false in the --config file ",
				", so ralph org escalate already sends every item to the human at once and leaves nothing to a director, and no director is started; " +
					"to use one, set [org.director].enabled = true there"}
		}},
		{"a model [org.roles].director does not allow", func(t *testing.T, _ string) []string {
			return []string{"--driver", "claude", "--model", "opus", "--config", writeDirectorConfig(t, "[org.roles]\ndirector = [\"sonnet\", \"haiku\"]\n")}
		}, nil, func(string) []string {
			return []string{`org: director start: model "opus" is not permitted for role director by [org.roles].director in the --config file `, "(it allows sonnet, haiku)"}
		}},
		{"codex autonomous before codex_verified", func(*testing.T, string) []string {
			return []string{"--driver", "codex", "--model", "gpt-6-astra"}
		}, org.ErrDirectorCodexWorkspaceWrite, func(string) []string {
			return []string{`org: director: start: permission mode "autonomous" of role director ([org.permissions.roles].director, else [org.permissions].default): ` +
				"a codex director runs only in permission mode guarded, whatever [org.permissions].codex_verified says: " +
				"autonomous and edits run codex under --sandbox workspace-write, which keeps .git read-only, " +
				"so the director could not start a feature org (ralph org start --plan runs git worktree add); " +
				`set [org.permissions.roles].director = "guarded" and let your codex config (~/.codex/config.toml) allow writing .git and reaching the network, ` +
				`or start a claude director (--driver claude); a guarded codex director follows that config, so under sandbox_mode = "workspace-write" it meets the same wall`}
		}},
		{"codex edits with codex_verified", func(t *testing.T, _ string) []string {
			return []string{"--driver", "codex", "--model", "gpt-6-astra", "--config",
				writeDirectorConfig(t, "[org.permissions]\ndefault = \"edits\"\ncodex_verified = true\n")}
		}, org.ErrDirectorCodexWorkspaceWrite, func(string) []string {
			return []string{`org: director: start: permission mode "edits" of role director ([org.permissions.roles].director, else [org.permissions].default): ` +
				"a codex director runs only in permission mode guarded, whatever [org.permissions].codex_verified says: "}
		}},
		{"a promoted session is attached", func(t *testing.T, stateDir string) []string {
			setEscalateWatch(t, acceptingWatchLiveness())
			runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
			return []string{"--driver", "claude", "--model", "opus"}
		}, org.ErrDirectorAttached, func(stateDir string) []string {
			return []string{"a director is already attached: d1 (driver claude, attached ",
				"): end it with ralph org director detach --id d1 --state-dir '" + stateDir + "' from the session that attached it " +
					"(or with --force in place of --id, when that session is gone), then run start again (start has no --replace)"}
		}},
		{"a headless director is still starting", func(t *testing.T, stateDir string) []string {
			seedHeadless(t, stateDir, "codex", "gpt-6-astra", "", "", false)
			return []string{"--driver", "claude", "--model", "opus"}
		}, org.ErrDirectorAttached, func(stateDir string) []string {
			return []string{"a director is already attached: d1 (driver codex, attached ",
				", headless starting): stop it with ralph org director stop --id d1 --state-dir '" + stateDir + "' from a terminal outside herdr, " +
					"then run start again (start has no --replace)"}
		}},
		{"inside a herdr pane", func(t *testing.T, _ string) []string {
			t.Setenv("HERDR_PANE_ID", "p-9")
			return []string{"--driver", "claude", "--model", "opus"}
		}, org.ErrDirectorStartInsideHerdr, func(string) []string {
			return []string{"HERDR_PANE_ID is set, so this runs inside a herdr pane: run ralph org director start from a terminal outside herdr"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			args := tc.setup(t, stateDir)
			before := directorFileOrAbsent(t, stateDir)

			stdout, _, err := runOrgCmdStreams(t, append([]string{"director", "start", "--state-dir", stateDir}, args...)...)
			if err == nil {
				t.Fatalf("director start succeeded; want refused (stdout %s)", stdout)
			}
			if tc.is != nil && !errors.Is(err, tc.is) {
				t.Errorf("error = %v; want it to wrap %v", err, tc.is)
			}
			for _, want := range tc.wants(stateDir) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused start changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
			if _, err := os.Stat(filepath.Join(stateDir, "prompts")); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("stat prompts/ = %v; want no prompt file written", err)
			}
		})
	}
}

// Self-review L1 of org-director-start through the CLI, with TestMain's gate
// that always passes set back to nil and no watch running: start refuses an
// attached director before its watch gate, so it launches no watch, prints
// nothing, and leaves the ledger as it was (no watch.log, watch.lock, or
// liveness mark, director.jsonl unchanged), whether the director is a
// promoted session or a headless one still starting.
func TestOrgDirectorStart_AnAttachedDirectorIsRefusedBeforeTheWatchGate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, stateDir string)
	}{
		{"a promoted session", func(t *testing.T, stateDir string) {
			runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
		}},
		{"a headless director still starting", func(t *testing.T, stateDir string) {
			seedHeadless(t, stateDir, "codex", "gpt-6-astra", "", "", false)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			launches := 0
			setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
				launches++
				return nil, errors.New("no launch in this test")
			})
			stateDir, herdrLog, _ := directorStartLedger(t)
			tc.setup(t, stateDir)
			before, entries := directorFileOrAbsent(t, stateDir), dirNames(t, stateDir)

			stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
			if !errors.Is(err, org.ErrDirectorAttached) {
				t.Errorf("director start over an attached director with no watch = %v; want ErrDirectorAttached", err)
			}
			if launches != 0 {
				t.Errorf("the watch launcher was called %d times; want none", launches)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused start changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
			}
			if got := dirNames(t, stateDir); !slices.Equal(got, entries) {
				t.Errorf("ledger entries = %v; want %v as before (no watch files)", got, entries)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
		})
	}
}

// start's refusal from AttachHeadless when the registration it refused on is
// gone, or unreadable, by the time start reads it again (Current nil) names
// status with the --state-dir of the call when one was given, as the other
// commands the director verbs print do. A CLI run would need a race, so this
// tests the helper, as TestOrgDirector_NotRestoredErrorNamesTheStatusCommand
// does (a refusal that named status without the call's --state-dir would
// read another ledger's registration).
func TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus(t *testing.T) {
	refusal := fmt.Errorf("org: director: start: %w: d1 (driver claude, attached 2026-10-15T09:00:00Z)", org.ErrDirectorAttached)
	const rest = ", and run start again once none is attached (start has no --replace)"
	for _, tc := range []struct {
		name, hintStateDir, want string
	}{
		{"without --state-dir", "", refusal.Error() + ": read the current registration with ralph org director status" + rest},
		{"with --state-dir", "/work/the ledger", refusal.Error() + ": read the current registration with ralph org director status --state-dir '/work/the ledger'" + rest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &orgDirectorRuntime{hintStateDir: tc.hintStateDir}
			err := rt.startAttachedError(&org.DirectorAttachedError{Err: refusal})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("startAttachedError = %v; want %q", err, tc.want)
			}
			if !errors.Is(err, org.ErrDirectorAttached) {
				t.Errorf("startAttachedError = %v; want it to still wrap ErrDirectorAttached", err)
			}
		})
	}
}

// dirNames lists the names in dir, sorted.
func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// AC6 through the CLI (D6): without --config, on the main worktree's ledger
// found from a linked worktree, start takes [org.director].enabled,
// [org.permissions], [org.roles], and model_pool from the main worktree's
// ralph.toml, not the linked worktree's, and the director runs in the main
// worktree root; --config names the file to read instead. A main worktree
// ralph.toml that does not load is an error. The ralph org commands the
// prompt file has the director run carry a relative --config made absolute
// (in the main worktree root, where the director runs, it would name the
// main worktree's file), and without --config carry none; its stop line
// carries none either way, as stop reads no ralph.toml (AC11).
func TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml(t *testing.T) {
	const mainToml = "[org]\nmodel_pool = [{ driver = \"claude\", model = \"haiku\" }, { driver = \"claude\", model = \"opus\" }]\n\n" +
		"[org.permissions]\ndefault = \"guarded\"\n\n[org.roles]\ndirector = [\"haiku\"]\n"
	const worktreeToml = "[org.permissions]\ndefault = \"autonomous\"\n\n[org.roles]\ndirector = [\"opus\"]\n"
	cases := []struct {
		name               string
		mainToml, wtToml   string
		args               []string
		wantErr            func(r legacyLedgerRepo) []string // nil: start succeeds
		wantStderr         string
		wantPerm, wantArgs string // the started line's permission, and agent start's flags before the pointer
		wantModel          string
		wantConfig         func(r legacyLedgerRepo) string // the --config of the ralph org commands the prompt has the director run (nil: none)
	}{
		{name: "no --config: the main worktree's settings", mainToml: mainToml, wtToml: worktreeToml,
			args:       []string{"--driver", "claude"},
			wantStderr: "org: --model omitted; falling back to first [org].model_pool entry permitted for role director on claude: haiku (pass --model explicitly)\n",
			wantPerm:   "guarded", wantArgs: "--model haiku", wantModel: "haiku"},
		{name: "no --config: a model only the worktree allows", mainToml: mainToml, wtToml: worktreeToml,
			args: []string{"--driver", "claude", "--model", "opus"},
			wantErr: func(r legacyLedgerRepo) []string {
				return []string{`org: director start: model "opus" is not permitted for role director by [org.roles].director in the main worktree's ` +
					filepath.Join(r.mainRoot, "ralph.toml") + " (it allows haiku)"}
			}},
		{name: "--config names the worktree's ralph.toml", mainToml: mainToml, wtToml: worktreeToml,
			args:     []string{"--driver", "claude", "--model", "opus", "--config", "ralph.toml"},
			wantPerm: "autonomous", wantArgs: "--permission-mode bypassPermissions --model opus", wantModel: "opus",
			wantConfig: func(r legacyLedgerRepo) string { return filepath.Join(r.worktree, "ralph.toml") }},
		{name: "the main worktree says enabled false, the worktree true", mainToml: "[org.director]\nenabled = false\n", wtToml: "[org.director]\nenabled = true\n",
			args: []string{"--driver", "claude", "--model", "opus"},
			wantErr: func(r legacyLedgerRepo) []string {
				return []string{"org: director start: [org.director].enabled is false in the main worktree's " + filepath.Join(r.mainRoot, "ralph.toml") + ", "}
			}},
		{name: "a main worktree ralph.toml that does not load", mainToml: "[org]\nmax_orgs = 0\n",
			args: []string{"--driver", "claude", "--model", "opus"},
			wantErr: func(r legacyLedgerRepo) []string {
				return []string{"org: load [org.director].enabled, launch_cmd, wake and wake_cmd, [org.permissions], [org.roles] and model_pool from the main worktree's " +
					filepath.Join(r.mainRoot, "ralph.toml"), "[org].max_orgs must be >= 1, got 0"}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			r := newLegacyLedgerRepo(t)
			writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), tc.mainToml)
			if tc.wtToml != "" {
				writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), tc.wtToml)
			}

			stdout, stderr, err := runOrgCmdStreams(t, append([]string{"director", "start"}, tc.args...)...)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("director start succeeded; want refused (stdout %s)", stdout)
				}
				for _, want := range tc.wantErr(r) {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not contain %q", err, want)
					}
				}
				if got := directorFileOrAbsent(t, r.sharedDir); got != "<absent>" {
					t.Errorf("a refused start wrote director.jsonl: %s", got)
				}
				if lines := readLogLines(t, herdrLog); len(lines) != 0 {
					t.Errorf("herdr calls = %v; want none", lines)
				}
				return
			}
			if err != nil {
				t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if stderr != tc.wantStderr {
				t.Errorf("stderr = %q; want %q", stderr, tc.wantStderr)
			}
			events := directorEventsIn(t, r.sharedDir)
			if len(events) != 4 || events[0].PermissionMode != tc.wantPerm || events[0].Model != tc.wantModel {
				t.Fatalf("director.jsonl in the main worktree's ledger = %+v; want d1 with %s and %s", events, tc.wantPerm, tc.wantModel)
			}
			tok := events[0].Token
			if !strings.HasPrefix(stdout, "started d1 (driver=claude, model="+tc.wantModel+", permission="+tc.wantPerm+", ") {
				t.Errorf("stdout = %q; want d1 started with %s and %s", stdout, tc.wantModel, tc.wantPerm)
			}
			herdrLines := readLogLines(t, herdrLog)
			if len(herdrLines) != 3 || herdrLines[0] != "workspace create --cwd "+r.mainRoot+" --label ralph_director_"+tok {
				t.Fatalf("herdr calls = %v; want the workspace made in the main worktree root %s", herdrLines, r.mainRoot)
			}
			prompt := filepath.Join(r.sharedDir, "prompts", "director_"+tok+".md")
			if want := " -- " + tc.wantArgs + " " + org.PromptFilePointer(prompt); !strings.HasSuffix(herdrLines[2], want) {
				t.Errorf("agent start = %q; want it to end with %q", herdrLines[2], want)
			}
			data, err := os.ReadFile(prompt)
			if err != nil {
				t.Fatalf("read the prompt file: %v", err)
			}
			stateDirFlag := " --state-dir '" + r.sharedDir + "'"
			flags := stateDirFlag
			if tc.wantConfig != nil {
				flags += " --config '" + tc.wantConfig(r) + "'"
			}
			for _, want := range []string{"\nralph org director status" + flags + "\n", "\n  ralph org director stop --id d1" + stateDirFlag + "\n"} {
				if !strings.Contains(string(data), want) {
					t.Errorf("prompt file %s lacks %q:\n%s", prompt, want, data)
				}
			}
			// " --config", with the space: the t.TempDir paths in the prompt
			// hold the subtest's name, which can say --config.
			if tc.wantConfig == nil && strings.Contains(string(data), " --config") {
				t.Errorf("prompt file %s says --config for a start without one:\n%s", prompt, data)
			}
		})
	}
}

// AC15 through the CLI (D13): a --config whose [org].agmsg_home is a
// relative path (a bare "~" included: only "~/" is expanded) is refused with
// exit 1, naming the value and the file, before the --model fallback and the
// watch gate, which is the real one here: nothing on stdout or stderr, no
// director.jsonl, no prompt file, no herdr call, and no watch launched. An
// absolute value, one that starts with "~/", and an empty one (config.Load
// gives it the default) pass.
func TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name, home string // home is agmsg_home's value; "<abs>" is an absolute path
		refused    bool
	}{
		{"a relative path", "agmsg/home", true},
		{"a ./ path", "./agmsg", true},
		{"a bare ~", "~", true},
		{"an absolute path", "<abs>", false},
		{"a ~/ path", "~/.agents/skills/agmsg", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.refused {
				useRealOrgWatchGate(t)
			}
			launches := 0
			setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
				launches++
				return nil, errors.New("no launch in this test")
			})
			stateDir, herdrLog, _ := directorStartLedger(t)
			home := tc.home
			if home == "<abs>" {
				home = filepath.Join(t.TempDir(), "agmsg")
			}
			configPath := writeDirectorConfig(t, fmt.Sprintf("[org]\nagmsg_home = %q\n", home))

			stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--state-dir", stateDir, "--config", configPath)
			if !tc.refused {
				if err != nil {
					t.Fatalf("director start with agmsg_home %q: %v (stdout: %s stderr: %s)", home, err, stdout, stderr)
				}
				if got := directorEventKinds(directorEventsIn(t, stateDir)); len(got) != 4 || got[3] != "launched d1" {
					t.Errorf("director.jsonl = %v; want d1 launched", got)
				}
				return
			}
			if want := directorRelativeAgmsgHomeRefusal(home, configPath); err == nil || err.Error() != want {
				t.Fatalf("director start with agmsg_home %q = %v; want %q", home, err, want)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing (no --model fallback warning)", stdout, stderr)
			}
			if launches != 0 {
				t.Errorf("the watch launcher was called %d times; want none", launches)
			}
			if got := dirNames(t, stateDir); len(got) != 0 {
				t.Errorf("ledger entries = %v; want none (no director.jsonl, prompt file, or watch file)", got)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
		})
	}
}

// relativeAgmsgHomeRefusal is the whole error, after prefix, for the
// relative [org].agmsg_home home of the --config file configPath: director
// start's (D13; runs "the director and the watch run") and the watch
// launch's of --detach and `ralph org start` (plan
// 2026-10-29-local-config-org, D7; runs "the watch runs"), one sentence for
// both.
func relativeAgmsgHomeRefusal(prefix, runs, home, configPath string) string {
	return fmt.Sprintf("%s[org].agmsg_home %q in the --config file %s is a relative path: "+
		"%s in the ledger's main worktree root when the ledger has one, not in this directory, "+
		"so the path may name another directory there than here, and so another agmsg install; it is refused wherever ralph runs; "+
		"give an absolute path (or one that starts with ~/)", prefix, home, configPath, runs)
}

// directorRelativeAgmsgHomeRefusal is director start's whole error for the
// relative [org].agmsg_home home of the --config file configPath (D13).
func directorRelativeAgmsgHomeRefusal(home, configPath string) string {
	return relativeAgmsgHomeRefusal("org: director start: ", "the director and the watch run", home, configPath)
}

// AC15 through the CLI (D13): the refusal of a relative [org].agmsg_home in
// a --config does not depend on where start runs, because the rule compares
// no directories. It is refused in the ledger's main worktree root, where
// the director runs in start's directory; in a linked worktree, where the
// director runs in the main worktree root instead; and on a ledger with no
// main worktree (--state-dir outside a repository), where the director again
// runs in start's directory. Each refusal writes no director.jsonl and runs
// no herdr. The same start with an absolute agmsg_home then succeeds, and
// its workspace's --cwd shows where the director runs.
func TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWhereverStartRuns(t *testing.T) {
	for _, tc := range []struct {
		name          string
		directorInCwd bool // the director runs in start's directory
	}{
		{"in the main worktree root", true},
		{"in a linked worktree", false},
		{"on a ledger with no main worktree", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stateDir, herdrLog, directorCwd string
			args := []string{"director", "start", "--driver", "claude", "--model", "opus"}
			if tc.name == "on a ledger with no main worktree" {
				stateDir, herdrLog, directorCwd = directorStartLedger(t)
				if root, ok := org.LedgerMainWorktreeRoot(stateDir, "flag"); ok {
					t.Fatalf("the ledger %s has the main worktree %s; want none", stateDir, root)
				}
				args = append(args, "--state-dir", stateDir)
			} else {
				herdrLog, _ = setupOrgStubPATH(t)
				r := newLegacyLedgerRepo(t)
				if tc.directorInCwd {
					t.Chdir(r.mainRoot)
				}
				stateDir, directorCwd = r.sharedDir, r.mainRoot
			}
			startCwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if got := startCwd == directorCwd; got != tc.directorInCwd {
				t.Fatalf("start runs in %s and the director in %s; want the same directory: %v", startCwd, directorCwd, tc.directorInCwd)
			}

			relative := writeDirectorConfig(t, "[org]\nagmsg_home = \"agmsg/home\"\n")
			stdout, stderr, err := runOrgCmdStreams(t, slices.Concat(args, []string{"--config", relative})...)
			if want := directorRelativeAgmsgHomeRefusal("agmsg/home", relative); err == nil || err.Error() != want {
				t.Fatalf("director start with a relative agmsg_home = %v (stdout: %s stderr: %s); want %q", err, stdout, stderr, want)
			}
			if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
				t.Errorf("a refused start wrote director.jsonl: %s", got)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}

			absolute := writeDirectorConfig(t, fmt.Sprintf("[org]\nagmsg_home = %q\n", filepath.Join(t.TempDir(), "agmsg")))
			if stdout, stderr, err = runOrgCmdStreams(t, slices.Concat(args, []string{"--config", absolute})...); err != nil {
				t.Fatalf("director start with an absolute agmsg_home: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if lines := readLogLines(t, herdrLog); len(lines) == 0 || !strings.HasPrefix(lines[0], "workspace create --cwd "+directorCwd+" --label ") {
				t.Errorf("herdr calls = %v; want the workspace made in %s", lines, directorCwd)
			}
		})
	}
}

// AC15 through the CLI (D13): without --config, a relative [org].agmsg_home
// in the main worktree's ralph.toml is not refused, since the director and
// the watch both read that file and resolve it from that root.
func TestOrgDirectorStart_ARelativeAgmsgHomeWithoutConfigIsNotRefused(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), "[org]\nagmsg_home = \"agmsg/home\"\n")

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus")
	if err != nil {
		t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if !strings.HasPrefix(stdout, "started d1 (driver=claude, ") || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q; want d1 started and nothing on stderr", stdout, stderr)
	}
}

// AC8 of plan 2026-10-29-local-config-org through the CLI: with
// RALPH_ORG_AGMSG_HOME set, which ralph no longer reads, start with herdr
// warns about nothing (the D13 warning that the director's pane does not
// get the variable is gone) and starts the director.
func TestOrgDirectorStart_TheAgmsgHomeVariableGetsNoWarning(t *testing.T) {
	stateDir, _, _ := directorStartLedger(t)
	t.Setenv(legacyAgmsgHomeEnv, filepath.Join(t.TempDir(), "agmsg"))

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing", stderr)
	}
	if !strings.HasPrefix(stdout, "started d1 (driver=claude, ") {
		t.Errorf("stdout = %q; want d1 started", stdout)
	}
}

// The help of director start states D12 (a codex director runs guarded
// only, why, the two ways out, and the guarded caveat) and D13 (the
// relative agmsg_home refusal, wherever start runs, since the director and
// the watch run in the ledger's main worktree root), and no longer says that
// codex_verified unlocks autonomous and edits, nor that the director and
// the watch always use different agmsg installs. "Without --config, those
// settings" follows the sentence on the permission mode and the model it
// refers to. It says nothing of RALPH_ORG_AGMSG_HOME, which ralph no longer
// reads (plan 2026-10-29-local-config-org, AC8): neither the herdr warning
// nor the launch script's export.
func TestOrgDirectorStart_HelpStatesTheCodexAndAgmsgHomeRules(t *testing.T) {
	out, err := runOrgCmd(t, "director", "start", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"A codex director runs only in guarded: start refuses autonomous and edits whatever [org.permissions].codex_verified says, " +
			"because they run codex under --sandbox workspace-write, which keeps .git read-only, so the director could not start a feature org " +
			"(ralph org start --plan runs git worktree add).",
		`Set [org.permissions.roles].director = "guarded" and let your codex config (~/.codex/config.toml) write .git and reach the network, or use --driver claude;`,
		`a guarded codex director follows that config, so under sandbox_mode = "workspace-write" it meets the same wall.`,
		"exports RALPH_DIRECTOR_TOKEN, checks in, and then execs the agent start found on its PATH",
		"for a codex director in autonomous or edits, when the --config file's [org].agmsg_home is a relative path (neither absolute nor starting with ~/",
		"(neither absolute nor starting with ~/: the director and the watch run in the ledger's main worktree root when the ledger has one, " +
			"not where start runs, so the path may name another directory there than here and another agmsg install, " +
			"and start refuses it wherever it runs; give an absolute path)",
		"with the first model_pool entry for --driver that it allows, and a warning. Without --config, those settings and [org.director].enabled are read",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("director start --help does not say %q:\n%s", want, out)
		}
	}
	for _, stale := range []string{"need [org.permissions].codex_verified", "would use different agmsg installs", "RALPH_ORG_AGMSG_HOME",
		"the watch resolves it from where start runs"} {
		if strings.Contains(help, stale) {
			t.Errorf("director start --help still says %q:\n%s", stale, out)
		}
	}
}

// AC7 through the CLI (D5), with TestMain's gate that always passes set back
// to nil: start refuses a watch that holds the lock and is stopping or a
// --once run at once, and one whose liveness mark is an hour old once it has
// stayed stale for the whole wait of ensureOrgWatch (shortened), as `ralph
// org start` refuses it (TestOrgStart_WatchGate_RefusesAStaleWatch); each
// time it launches nothing, writes nothing (director.jsonl, a prompt file, or
// a watch file), and runs no herdr. With no watch it launches one as `ralph
// org start` does, says so on stderr with its own verb, and starts the
// director; a launch that fails is refused with start's prefix, which says
// no director was started.
func TestOrgDirectorStart_WatchGate(t *testing.T) {
	for _, w := range []struct {
		name  string
		mark  func() *org.WatchAlive
		want  []string
		stale bool // stale for the whole wait, which start waits out before it refuses
	}{
		{"stopping", func() *org.WatchAlive { return stoppingWatchMark(4601, time.Now()) },
			[]string{"(pid 4601) is stopping", "run this again once it has ended"}, false},
		{"running --once", func() *org.WatchAlive { return onceWatchMark(4602, time.Now()) },
			[]string{"(pid 4602) is a --once run", "run this again once it has ended"}, false},
		{"stale with an old mark", func() *org.WatchAlive { return runningMark(4604, time.Now().Add(-time.Hour)) },
			[]string{"has not refreshed its liveness mark (pid 4604)", "stop it with ralph org watch --stop"}, true},
	} {
		t.Run(w.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			refuseOrgWatchLaunch(t)
			if w.stale {
				shortenOrgWatchDeadline(t)
			}
			stateDir, herdrLog, _ := directorStartLedger(t)
			holdWatchLock(t, stateDir, w.mark())
			entries := dirNames(t, stateDir)

			start := time.Now()
			stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
			elapsed := time.Since(start)
			checkGateRefusal(t, err, stateDir, w.want...)
			if w.stale && elapsed < 300*time.Millisecond {
				t.Errorf("start refused the stale watch after %v; want it to wait the 300ms deadline first", elapsed)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
				t.Errorf("director.jsonl = %s; want nothing written", got)
			}
			if got := dirNames(t, stateDir); !slices.Equal(got, entries) {
				t.Errorf("ledger entries = %v; want %v as before (nothing written)", got, entries)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
		})
	}

	t.Run("no watch: start launches one", func(t *testing.T) {
		useRealOrgWatchGate(t)
		stateDir, _, _ := directorStartLedger(t)
		var got []orgWatchLaunchRequest
		setOrgWatchLaunch(t, lateMarkLauncher(t, stateDir, 4603, 50*time.Millisecond, &got))

		stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		if err != nil {
			t.Fatalf("director start with no watch: %v (stdout: %s stderr: %s)", err, stdout, stderr)
		}
		if len(got) != 1 || !slices.Equal(got[0].args, []string{"org", "watch", "--all", "--state-dir", stateDir}) {
			t.Errorf("launches = %+v; want one foreground watch of the ledger", got)
		}
		if want := fmt.Sprintf("ralph org director start: started the cross-org watch (pid 4603); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stderr != want {
			t.Errorf("stderr = %q; want %q", stderr, want)
		}
		if !strings.HasPrefix(stdout, "started d1 (driver=claude, ") {
			t.Errorf("stdout = %q; want d1 started", stdout)
		}
	})

	t.Run("a launch that fails", func(t *testing.T) {
		useRealOrgWatchGate(t)
		failOrgWatchLaunch(t)
		stateDir, herdrLog, _ := directorStartLedger(t)

		_, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		if want := "org: director start: the cross-org watch did not start, so no director was started: start /x/ralph: boom"; err == nil || err.Error() != want {
			t.Fatalf("director start with a launch that fails = %v; want %q", err, want)
		}
		if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
			t.Errorf("director.jsonl = %s; want nothing written", got)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})
}

// AC3 through the CLI: when herdr's agent start fails, start closes the pane
// and the workspace it made, records d1 stopped (with detached, as it was
// current), and exits 1 saying so. When the workspace cannot be closed it
// records detached only, so d1 stays open, and names the herdr command and
// the stop command with this invocation's --state-dir; status then lists d1
// as open.
func TestOrgDirectorStart_AFailedAgentStartIsCleanedUp(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		stateDir, herdrLog, _ := directorStartLedger(t)
		t.Setenv("ORG_STUB_FAIL", "agent:start")

		stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		if err == nil || !strings.Contains(err.Error(), "herdr agent start director_") ||
			!strings.HasSuffix(err.Error(), "; start closed what it had made in herdr, and d1 is recorded stopped") {
			t.Fatalf("director start with a failing agent start = %v; want the failure and the cleanup", err)
		}
		if stdout != "" {
			t.Errorf("stdout = %q; want nothing", stdout)
		}
		if got, want := directorEventKinds(directorEventsIn(t, stateDir)),
			[]string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1", "stopped d1"}; !slices.Equal(got, want) {
			t.Errorf("director.jsonl = %v; want %v", got, want)
		}
		if got, want := herdrActions(t, herdrLog), []string{"pane send-keys pane-stub-1 C-c", "pane close pane-stub-1", "workspace close ws-stub-1"}; !slices.Equal(got, want) {
			t.Errorf("herdr actions = %v; want %v", got, want)
		}
		if got := directorStatusLine(t, "--state-dir", stateDir); got != directorNotAttachedLine {
			t.Errorf("director line = %q; want %q", got, directorNotAttachedLine)
		}
	})

	t.Run("not closed", func(t *testing.T) {
		stateDir, _, _ := directorStartLedger(t)
		t.Setenv("ORG_STUB_FAIL", "agent:start")
		t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "ws-stub-1")
		suffix := " --state-dir '" + stateDir + "'"

		_, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		var cleanup *org.DirectorCleanupError
		if !errors.As(err, &cleanup) || !cleanup.Detached {
			t.Fatalf("director start = %v; want a *DirectorCleanupError with d1 detached", err)
		}
		for _, want := range []string{
			"herdr agent start director_", `its cleanup did not finish: close workspace "ws-stub-1"`,
			"d1 was recorded detached only, so it is no longer the current registration but stays open",
			"close it by hand: herdr workspace close ws-stub-1; afterwards record the stop with ralph org director stop --id d1" + suffix,
		} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
		}
		if got, want := directorEventKinds(directorEventsIn(t, stateDir)),
			[]string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1"}; !slices.Equal(got, want) {
			t.Errorf("director.jsonl = %v; want %v", got, want)
		}
		want := directorNotAttachedLine + "; the headless director d1 (starting, pane pane-stub-1, workspace ws-stub-1) is still open but not attached: " +
			"stop it with ralph org director stop --id d1" + suffix
		if got := directorStatusLine(t, "--state-dir", stateDir); got != want {
			t.Errorf("director line:\n got %q\nwant %q", got, want)
		}
	})
}

// AC8 through the CLI: a headless director detached by detach stays open:
// status names it with the stop command, and stop closes its pane and
// workspace and records stopped alone (no detached: it is not current), after
// which status names it no more.
func TestOrgDirectorStop_ADetachedHeadlessDirector(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
	runInboxOK(t, "director", "detach", "--id", "d1", "--state-dir", stateDir)

	want := directorNotAttachedLine + "; the headless director d1 (running, pane pane-stub-1, workspace ws-stub-1) is still open but not attached: " +
		"stop it with ralph org director stop --id d1 --state-dir '" + stateDir + "'"
	if got := directorStatusLine(t, "--state-dir", stateDir); got != want {
		t.Errorf("director line:\n got %q\nwant %q", got, want)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if report := directorReportOf(t, jsonOut); report.Registration != nil || len(report.OpenHeadless) != 1 ||
		report.OpenHeadless[0].ID != "d1" || report.OpenHeadless[0].State != org.DirectorStateRunning {
		t.Errorf("director status --json = %s; want no registration and d1 in open_headless", jsonOut)
	}

	stdout, _ := runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
	if want := "stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)\n"; stdout != want {
		t.Errorf("stop stdout = %q; want %q", stdout, want)
	}
	if got := directorEventKinds(directorEventsIn(t, stateDir)); !slices.Equal(got[len(got)-2:], []string{"detached d1", "stopped d1"}) || len(got) != 6 {
		t.Errorf("director.jsonl = %v; want the detach's detached d1, then stopped d1 alone", got)
	}
	if got := herdrActions(t, herdrLog); len(got) != 3 {
		t.Errorf("herdr actions = %v; want the C-c and the two closes", got)
	}
	if got := directorStatusLine(t, "--state-dir", stateDir); got != directorNotAttachedLine {
		t.Errorf("director line after stop = %q; want %q", got, directorNotAttachedLine)
	}
	jsonOut, _ = runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if strings.Contains(jsonOut, "open_headless") {
		t.Errorf("director status --json = %s; want no open_headless once d1 is stopped", jsonOut)
	}
}

// AC5, AC9, and AC14 through the CLI: two ledgers share one stub herdr, and
// herdr loses its state between their starts, so B's director gets the
// pane and workspace ids A recorded. A's stop finds the labels of B's token
// on them: it sends no C-c and closes nothing, writes nothing, exits 1, and
// names the label search and stop --force. A's stop --force then records d1
// stopped with force (and detached, as it is current), still closing
// nothing, and warns how to find A's workspace by its label. B's director is
// untouched, and its own stop closes it.
func TestOrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced(t *testing.T) {
	stateA, herdrLog, _ := directorStartLedger(t)
	stateB := t.TempDir()
	tokA := startDirectorOK(t, stateA, "--driver", "claude", "--model", "opus")
	resetStubHerdrState(t)
	tokB := startDirectorOK(t, stateB, "--driver", "codex", "--model", "gpt-6-astra", "--config", writeDirectorConfig(t, "[org.permissions.roles]\ndirector = \"guarded\"\n"))
	if tokA == tokB {
		t.Fatalf("both starts have token %s; want one per start", tokA)
	}
	for _, s := range []string{stateA, stateB} {
		ev := directorEventsIn(t, s)
		if ev[1].WorkspaceID != "ws-stub-1" || ev[2].PaneID != "pane-stub-1" {
			t.Fatalf("%s recorded %s and %s; want ws-stub-1 and pane-stub-1 for both ledgers", s, ev[1].WorkspaceID, ev[2].PaneID)
		}
	}
	suffix := " --state-dir '" + stateA + "'"
	labelSearch := "herdr workspace list, then herdr workspace close <the id of the workspace labelled ralph_director_" + tokA + ">"
	const unconfirmed = "ctrl_c=skipped: not confirmed, pane=not closed: not confirmed, workspace=not closed: not confirmed"
	before := directorFileOrAbsent(t, stateA)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateA)
	var cleanup *org.DirectorCleanupError
	if !errors.As(err, &cleanup) || !cleanup.Unconfirmed {
		t.Fatalf("A's stop = %v; want an unconfirmed *DirectorCleanupError", err)
	}
	for _, want := range []string{
		`pane "pane-stub-1" is not confirmed as d1's (a tab labelled director_` + tokA + " in a workspace labelled ralph_director_" + tokA + ")",
		"; nothing was recorded; close it by hand: " + labelSearch + "; afterwards record the stop with ralph org director stop --id d1 --force" + suffix,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("A's stop error %q does not contain %q", err, want)
		}
	}
	if stdout != "" || stderr != "stop d1: "+unconfirmed+"\n" {
		t.Errorf("A's stop stdout %q, stderr %q; want nothing, and the notes on stderr", stdout, stderr)
	}
	if after := directorFileOrAbsent(t, stateA); after != before {
		t.Errorf("A's refused stop changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}
	if got := herdrActions(t, herdrLog); len(got) != 0 {
		t.Errorf("herdr actions = %v; want no C-c and no close", got)
	}

	stdout, stderr = runInboxOK(t, "director", "stop", "--id", "d1", "--force", "--state-dir", stateA)
	if want := "stopped d1 (forced; " + unconfirmed + ")\nthe director is no longer attached\n"; stdout != want {
		t.Errorf("A's stop --force stdout = %q; want %q", stdout, want)
	}
	if !strings.HasPrefix(stderr, "warning: director stop --force: d1 is recorded stopped, but its herdr pane or workspace may still be open (") ||
		!strings.HasSuffix(stderr, "); find and close them by hand: "+labelSearch+"\n") {
		t.Errorf("A's stop --force stderr = %q; want the warning with the label search", stderr)
	}
	events := directorEventsIn(t, stateA)
	if got := directorEventKinds(events); !slices.Equal(got[4:], []string{"detached d1", "stopped d1"}) || !events[5].Force || events[4].Force {
		t.Errorf("A's director.jsonl = %+v; want detached d1, then stopped d1 with force", events)
	}
	if got := herdrActions(t, herdrLog); len(got) != 0 {
		t.Errorf("herdr actions after --force = %v; want still no C-c and no close", got)
	}

	if line := directorStatusLine(t, "--state-dir", stateB); !strings.HasPrefix(line, "director: attached d1 (driver=codex, headless running, ") {
		t.Errorf("B's director line = %q; want d1 still running", line)
	}
	stdout, _ = runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateB)
	if want := "stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)\nthe director is no longer attached\n"; stdout != want {
		t.Errorf("B's stop stdout = %q; want %q", stdout, want)
	}
}

// AC9 and AC14 through the CLI: stop refuses, with or without --force,
// writing nothing and sending no C-c or close, when it runs inside the
// director's pane (HERDR_PANE_ID) or inside its workspace (HERDR_WORKSPACE_ID
// on the workspace herdr's label confirmed).
func TestOrgDirectorStop_RefusedInsideTheDirectorsPaneOrWorkspace(t *testing.T) {
	for _, tc := range []struct {
		env, value string
		is         error
	}{
		{"HERDR_PANE_ID", "pane-stub-1", org.ErrDirectorStopSelfPane},
		{"HERDR_WORKSPACE_ID", "ws-stub-1", org.ErrDirectorStopSelfWorkspace},
	} {
		t.Run(tc.env, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
			t.Setenv(tc.env, tc.value)
			before := directorFileOrAbsent(t, stateDir)
			for _, extra := range [][]string{nil, {"--force"}} {
				stdout, _, err := runOrgCmdStreams(t, append([]string{"director", "stop", "--id", "d1", "--state-dir", stateDir}, extra...)...)
				if !errors.Is(err, tc.is) || !strings.Contains(err.Error(), tc.is.Error()) {
					t.Errorf("stop %v inside the director's %s = %v; want %v", extra, tc.env, err, tc.is)
				}
				if stdout != "" {
					t.Errorf("stop %v stdout = %q; want nothing", extra, stdout)
				}
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused stop changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
			}
			if got := herdrActions(t, herdrLog); len(got) != 0 {
				t.Errorf("herdr actions = %v; want no C-c and no close", got)
			}
		})
	}
}

// AC9 and AC14 through the CLI: a workspace close that fails leaves stop
// refused with nothing written: it says what was closed (on stderr), the
// herdr command for what is left, the stop to run afterwards, and stop
// --force, since a close failure's own stop command does not carry it.
// stop --force then records the stop with the workspace left open.
func TestOrgDirectorStop_ACloseThatFailsNamesForce(t *testing.T) {
	stateDir, _, _ := directorStartLedger(t)
	startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "ws-stub-1")
	suffix := " --state-dir '" + stateDir + "'"
	before := directorFileOrAbsent(t, stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
	var cleanup *org.DirectorCleanupError
	if !errors.As(err, &cleanup) || cleanup.Unconfirmed {
		t.Fatalf("stop with a failing workspace close = %v; want a confirmed *DirectorCleanupError", err)
	}
	if want := "; nothing was recorded; close it by hand: herdr workspace close ws-stub-1; afterwards record the stop with ralph org director stop --id d1" + suffix +
		"; or record the stop and leave them open with ralph org director stop --id d1 --force" + suffix; !strings.HasSuffix(err.Error(), want) {
		t.Errorf("stop error %q does not end with %q", err, want)
	}
	if stdout != "" || !strings.HasPrefix(stderr, "stop d1: ctrl_c=ok, pane=closed, workspace=close failed: ") {
		t.Errorf("stop stdout %q, stderr %q; want nothing, and the notes on stderr", stdout, stderr)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused stop changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}

	stdout, _ = runInboxOK(t, "director", "stop", "--id", "d1", "--force", "--state-dir", stateDir)
	if !strings.HasPrefix(stdout, "stopped d1 (forced; ctrl_c=ok, pane=closed, workspace=close failed: ") {
		t.Errorf("stop --force stdout = %q; want d1 stopped, forced, with the workspace left open", stdout)
	}
	if ev := directorEventsIn(t, stateDir); ev[len(ev)-1].Event != org.DirectorEventStopped || !ev[len(ev)-1].Force {
		t.Errorf("last event = %+v; want stopped d1 with force", ev[len(ev)-1])
	}
}

// AC9's last sentence (plan D14) in the CLI's words: a stop that found a
// workspace or pane a start recorded after stop read the registration writes
// what it closed on stderr, as for the other stops that record nothing, and
// only there: its error says what changed, that nothing was recorded, and
// names the stop to run again with the call's --state-dir and no --force,
// with or without --force given, with "org: director: " once and none of the
// notes. A CLI run would need a start that records while stop runs, so this
// tests the helper with the error StopDirector returns (internal/org's
// TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns pins how that
// error is made). The stop help says so too.
func TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain(t *testing.T) {
	const notes = "ctrl_c=skipped, pane=none recorded, workspace=closed"
	refusal := fmt.Errorf("%w: d1 had workspace \"ws-7\" and no pane when stop read it, and has workspace \"ws-7\" and pane \"pane-1\" now",
		org.ErrDirectorChangedDuringStop)
	res := org.DirectorStopResult{
		Registration: org.DirectorRegistration{ID: "d1", WorkspaceID: "ws-7"},
		CtrlCNote:    "ctrl_c=skipped", PaneNote: "pane=none recorded", WorkspaceNote: "workspace=closed",
	}
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force %v", force), func(t *testing.T) {
			cleanup := &org.DirectorCleanupError{
				Verb: "stop", ID: "d1", WorkspaceID: "ws-7",
				CleanupErr:  fmt.Errorf("record d1 stopped: %w; this stop closed only what it had read", refusal),
				StopCommand: "ralph org director stop --id d1",
			}
			rt := &orgDirectorRuntime{hintStateDir: "/work/the ledger"}
			var stderr strings.Builder
			err := rt.stopError(&stderr, "d1", force, res, cleanup)
			want := "org: director: stop d1: record d1 stopped: " + refusal.Error() + "; this stop closed only what it had read; " +
				"nothing was recorded; run ralph org director stop --id d1 --state-dir '/work/the ledger' again to close what start recorded"
			if err == nil || err.Error() != want {
				t.Fatalf("stopError =\n%v\nwant\n%s", err, want)
			}
			if !errors.Is(err, org.ErrDirectorChangedDuringStop) {
				t.Errorf("stopError = %v; want it to still wrap ErrDirectorChangedDuringStop", err)
			}
			if got := stderr.String(); got != "stop d1: "+notes+"\n" {
				t.Errorf("stderr = %q; want the notes", got)
			}
			if n := strings.Count(stderr.String()+err.Error(), notes); n != 1 {
				t.Errorf("the notes appear %d times in stderr and the error; want once, on stderr", n)
			}
		})
	}

	out, err := runOrgCmd(t, "director", "stop", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if want := "When a start that is still running records a workspace or pane after stop read the registration, stop has not closed it, " +
		"so it records nothing and exits 1 with what it closed: run it again to close what start recorded."; !strings.Contains(strings.Join(strings.Fields(out), " "), want) {
		t.Errorf("director stop --help does not say %q:\n%s", want, out)
	}
}

// AC8's last sentence in the CLI's words: when start's cleanup or another
// stop recorded the stop first (AlreadyStopped), stop exits 0. On stdout it
// says that d1 was already recorded stopped, with what this stop did to the
// pane and the workspace, and that this stop recorded nothing more. It says
// nothing on stderr (no --force warning, also after a forced stop whose
// close failed), does not say the director is no longer attached, and names
// no stop to run, since one would be refused as already stopped. A CLI run
// would need a start that records while stop runs, so this tests
// printOrgDirectorStop with the result StopDirector returns (internal/org's
// TestStopDirector_SucceedsWhenTheStopWasRecordedFirst pins how that result
// is made). The stop help says so too.
func TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun(t *testing.T) {
	for _, tc := range []struct {
		name, workspaceNote string
	}{
		{"the workspace closed", "workspace=closed"},
		{"a forced stop whose workspace close failed", "workspace=close failed: herdr: connection refused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := org.DirectorStopResult{
				Registration: org.DirectorRegistration{ID: "d1", WorkspaceID: "ws-7"},
				CtrlCNote:    "ctrl_c=skipped", PaneNote: "pane=none recorded", WorkspaceNote: tc.workspaceNote,
				AlreadyStopped: true,
			}
			cmd := NewRootCmd()
			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			printOrgDirectorStop(cmd, res)
			want := "d1 was already recorded stopped (ctrl_c=skipped, pane=none recorded, " + tc.workspaceNote + ")\n" +
				"start's cleanup or another stop recorded d1 stopped after this stop read it, so this stop recorded nothing more\n"
			if got := stdout.String(); got != want {
				t.Errorf("stdout =\n%q\nwant\n%q", got, want)
			}
			if got := stderr.String(); got != "" {
				t.Errorf("stderr = %q; want nothing", got)
			}
			if strings.Contains(stdout.String(), "ralph org director stop") {
				t.Errorf("stdout %q names a stop to run; want none", stdout.String())
			}
		})
	}

	out, err := runOrgCmd(t, "director", "stop", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if want := "When start's cleanup (after a step of start failed) or another stop records the director stopped while stop closes it, " +
		"stop records nothing more and exits 0 with what it closed and that the director was already recorded stopped."; !strings.Contains(strings.Join(strings.Fields(out), " "), want) {
		t.Errorf("director stop --help does not say %q:\n%s", want, out)
	}
}

// AC9 through the CLI: stop refuses, writing nothing and running no herdr,
// without --id, for an id no registration has (naming status, which lists
// the open headless directors), for a registration a promoted session
// attached (naming detach while it is current, and saying it has ended once
// it is not), and for one already stopped.
func TestOrgDirectorStop_Refusals(t *testing.T) {
	t.Run("no --id", func(t *testing.T) {
		herdrLog, _ := setupOrgStubPATH(t)
		stateDir := filepath.Join(t.TempDir(), "state")
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--state-dir", stateDir)
		if want := "org: director stop: give --id <id>, the id start printed (ralph org director status lists the open headless directors)"; err == nil || err.Error() != want {
			t.Errorf("stop without --id = %v; want %q", err, want)
		}
		if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stat state dir = %v; want it not created", err)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})

	t.Run("an unknown id", func(t *testing.T) {
		herdrLog, _ := setupOrgStubPATH(t)
		stateDir := t.TempDir()
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--id", "d9", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorNoOpenHeadless) ||
			!strings.HasSuffix(err.Error(), "no registration d9 is recorded; ralph org director status --state-dir '"+stateDir+"' lists the open headless directors") {
			t.Errorf("stop --id d9 = %v; want no registration d9, naming status", err)
		}
		if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
			t.Errorf("director.jsonl = %s; want nothing written", got)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})

	t.Run("a promoted session's registration", func(t *testing.T) {
		herdrLog, _ := setupOrgStubPATH(t)
		setEscalateWatch(t, acceptingWatchLiveness())
		stateDir := t.TempDir()
		runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
		before := directorFileOrAbsent(t, stateDir)
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorNotHeadless) ||
			!strings.HasSuffix(err.Error(), ": end it with ralph org director detach --id d1 --state-dir '"+stateDir+"' from the session that attached it "+
				"(or with --force in place of --id, when that session is gone)") {
			t.Errorf("stop of a promoted registration = %v; want it to name detach", err)
		}
		if after := directorFileOrAbsent(t, stateDir); after != before {
			t.Errorf("a refused stop changed director.jsonl")
		}
		runInboxOK(t, "director", "detach", "--id", "d1", "--state-dir", stateDir)
		_, _, err = runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorNotHeadless) || !strings.HasSuffix(err.Error(), ": it is not the current registration, so it has already ended and there is nothing to stop") {
			t.Errorf("stop of a detached promoted registration = %v; want it to say it has ended", err)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})

	t.Run("already stopped", func(t *testing.T) {
		stateDir, herdrLog, _ := directorStartLedger(t)
		startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
		runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		before, calls := directorFileOrAbsent(t, stateDir), len(readLogLines(t, herdrLog))
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorStopped) || strings.Contains(err.Error(), "lists the open headless directors") {
			t.Errorf("a second stop = %v; want ErrDirectorStopped alone", err)
		}
		if after := directorFileOrAbsent(t, stateDir); after != before {
			t.Errorf("a refused stop changed director.jsonl")
		}
		if got := len(readLogLines(t, herdrLog)); got != calls {
			t.Errorf("herdr was called %d time(s) by the refused stop; want 0", got-calls)
		}
	})
}

// AC8 through the CLI: stop reads no ralph.toml, so a ./ralph.toml and a
// --config that do not load block it no more than they block detach. It
// also stops a registration still starting: with nothing recorded it runs
// no herdr and names the workspace label to look for; with only the
// workspace recorded it closes that workspace.
func TestOrgDirectorStop_AStartingRegistrationAndABrokenRalphToml(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	dir := t.TempDir()
	broken := filepath.Join(dir, "ralph.toml")
	writeTestFile(t, broken, "[org\n")
	t.Chdir(dir)
	stateDir := t.TempDir()

	reg := seedHeadless(t, stateDir, "claude", "opus", "", "", false)
	stdout, _ := runInboxOK(t, "director", "stop", "--id", reg.ID, "--state-dir", stateDir, "--config", broken)
	want := "stopped d1 (ctrl_c=skipped, pane=none recorded, workspace=none recorded)\nthe director is no longer attached\n" +
		"no workspace id was recorded for d1: herdr may still have a workspace labelled ralph_director_" + reg.Token +
		" that start made before it could record it; look for it with herdr workspace list\n"
	if stdout != want {
		t.Errorf("stop of d1 with nothing recorded =\n%s\nwant\n%s", stdout, want)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none with nothing recorded", lines)
	}

	reg = seedHeadless(t, stateDir, "claude", "opus", "ws-9", "", false)
	recordStubHerdr(t, "workspace-ws-9", "ralph_director_"+reg.Token)
	stdout, _ = runInboxOK(t, "director", "stop", "--id", reg.ID, "--state-dir", stateDir)
	if want := "stopped d2 (ctrl_c=skipped, pane=none recorded, workspace=closed)\nthe director is no longer attached\n"; stdout != want {
		t.Errorf("stop of d2 with only its workspace = %q; want %q", stdout, want)
	}
	if got := herdrActions(t, herdrLog); !slices.Equal(got, []string{"workspace close ws-9"}) {
		t.Errorf("herdr actions = %v; want only the workspace close", got)
	}
}

// AC2 for attach: a headless registration, even one still starting, is ended
// with director stop, not detach, so attach's refusal names stop (and
// --replace, which leaves its pane open) and writes nothing.
func TestOrgDirectorAttach_RefusedByAHeadlessDirectorNamesStop(t *testing.T) {
	stateDir := t.TempDir()
	seedHeadless(t, stateDir, "codex", "gpt-6-astra", "", "", false)
	before := directorFileOrAbsent(t, stateDir)
	suffix := " --state-dir '" + stateDir + "'"

	stdout, _, err := runOrgCmdStreams(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	if !errors.Is(err, org.ErrDirectorAttached) {
		t.Fatalf("attach over a starting headless director = %v; want ErrDirectorAttached", err)
	}
	if want := ", headless starting): stop it with ralph org director stop --id d1" + suffix + " from a terminal outside herdr, which closes its herdr pane, " +
		"or replace it with ralph org director attach --driver claude --replace" + suffix + ", which leaves that pane open until such a stop"; !strings.HasSuffix(err.Error(), want) {
		t.Errorf("attach error %q does not end with %q", err, want)
	}
	if strings.Contains(err.Error(), "detach") {
		t.Errorf("attach error %q names detach; want stop for a headless director", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused attach changed director.jsonl")
	}
}

// Self-review L2 of org-director-start: detach --id and detach --force of a
// headless director that is not stopped only record (D4), so after their
// detached line they print one more that says its herdr pane and workspace
// stay open and names the stop that closes them, with --state-dir; they
// write detached only and run no herdr. A headless registration already
// stopped (a stopped with no detached, which a cut-back that failed can
// leave) gets no such line. A promoted session's detach prints none either
// (TestOrgDirector_AttachReplaceAndDetachByID pins its stdout whole).
func TestOrgDirectorDetach_AHeadlessDirectorNamesItsStop(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		detached string // the first stdout line, after the id
	}{
		{"--id", []string{"--id", "d1"}, ""},
		{"--force", []string{"--force"}, " (forced; driver=codex, attached at "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			reg := seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
			want := "detached d1" + tc.detached
			if tc.detached != "" {
				want += reg.AttachedAt + ")"
			}
			want += "\nthe herdr pane and workspace of the headless director d1 stay open; close them with: ralph org director stop --id d1 --state-dir '" +
				stateDir + "'\n"

			stdout, _ := runInboxOK(t, append([]string{"director", "detach", "--state-dir", stateDir}, tc.args...)...)
			if stdout != want {
				t.Errorf("detach %v stdout =\n%s\nwant\n%s", tc.args, stdout, want)
			}
			if got := directorEventKinds(directorEventsIn(t, stateDir)); got[len(got)-1] != "detached d1" || len(got) != 5 {
				t.Errorf("director.jsonl = %v; want the seeded four events and detached d1", got)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
		})
	}

	t.Run("already stopped", func(t *testing.T) {
		stateDir, _, _ := directorStartLedger(t)
		seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
		f, err := os.OpenFile(org.DirectorPathIn(stateDir), os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(`{"ts":"2026-10-15T09:00:00Z","id":"d1","event":"stopped","user":"alice","cwd":"/work","herdr_pane":false}` + "\n"); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		if res, err := org.NewDirectorStore(stateDir).Read(); err != nil || res.Current == nil || res.Current.State != org.DirectorStateStopped {
			t.Fatalf("seeded registration = %+v, %v; want d1 current and stopped", res.Current, err)
		}

		stdout, _ := runInboxOK(t, "director", "detach", "--id", "d1", "--state-dir", stateDir)
		if stdout != "detached d1\n" {
			t.Errorf("detach of a stopped headless director stdout = %q; want only the detached line", stdout)
		}
	})
}

// Self-review L2 of org-director-start for attach --replace: replacing a
// headless director leaves its herdr pane and workspace open, so after the
// attached and detach lines attach prints the line that says so and names
// the stop that closes the replaced one, with --state-dir.
func TestOrgDirectorAttachReplace_AHeadlessDirectorNamesItsStop(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	stateDir, herdrLog, _ := directorStartLedger(t)
	reg := seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
	suffix := " --state-dir '" + stateDir + "'"

	stdout, _ := runInboxOK(t, "director", "attach", "--driver", "claude", "--replace", "--state-dir", stateDir)
	want := "attached d2 (driver=claude), replacing d1 (driver=codex, attached at " + reg.AttachedAt + ")\n" +
		"detach it with: ralph org director detach --id d2" + suffix + "\n" +
		"the herdr pane and workspace of the headless director d1 stay open; close them with: ralph org director stop --id d1" + suffix + "\n"
	if stdout != want {
		t.Errorf("attach --replace stdout =\n%s\nwant\n%s", stdout, want)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
}

// AC10 for director status: a headless registration shows its state, model,
// pane, and workspace on the director: line, and its fields in the text
// form; the open headless registrations that are not current are named at
// the end of the line, one or several, and listed as open_headless lines and
// in JSON. A promoted session's lines keep their 8a-1 form (also
// TestOrgDirector_AttachReplaceAndDetachByID). No watch has checked d1, so
// its line says not checked yet (plan 2026-10-18-org-director-liveness, D8).
func TestOrgDirectorStatus_ShowsHeadlessDirectors(t *testing.T) {
	const tail = directorNotCheckedTail
	stateDir := t.TempDir()
	store := org.NewDirectorStore(stateDir)
	who := org.DirectorWho{User: "alice", Cwd: "/work"}
	d1 := seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "", false)
	ts := d1.AttachedAt

	stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
	want := "director: attached d1 (driver=codex, headless starting, model=gpt-6-astra, workspace w-1, attached at " + ts + ", by alice)" + tail + "\n" +
		"enabled: true\nauto_merge: true\nid: d1\ndriver: codex\nwake: pull\nattached_at: " + ts + "\nuser: alice\ncwd: /work\nherdr_pane: no\n" +
		"launch: herdr\nstate: starting\nmodel: gpt-6-astra\npermission_mode: autonomous\ntoken: " + d1.Token + "\nworkspace_id: w-1\n" +
		directorNoWaitMarkLine + "\n"
	if stdout != want {
		t.Errorf("status of a starting d1 =\n%s\nwant\n%s", stdout, want)
	}

	if err := store.RecordPane("d1", "p-1", who); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.MarkLaunched("d1", who); err != nil || !ok {
		t.Fatalf("mark d1 launched: %v, %v", ok, err)
	}
	running := "d1 (running, pane p-1, workspace w-1)"
	if got, want := directorStatusLine(t, "--state-dir", stateDir),
		"director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane p-1, workspace w-1, attached at "+ts+", by alice)"+tail; got != want {
		t.Errorf("director line of a running d1:\n got %q\nwant %q", got, want)
	}

	d2, _, err := store.Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, Replace: true, User: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	stdout, _ = runInboxOK(t, "director", "status", "--state-dir", stateDir)
	suffix := " --state-dir '" + stateDir + "'"
	wantLine := directorAttachedLine("d2", "claude", d2.AttachedAt, "bob") + "; the headless director " + running +
		" is still open but not attached: stop it with ralph org director stop --id d1" + suffix
	if lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); lines[0] != wantLine || lines[len(lines)-1] != "open_headless: "+running ||
		strings.Contains(stdout, "launch:") {
		t.Errorf("status with d2 promoted over d1 =\n%s\nwant the line %q, d2's 8a-1 fields, and open_headless: %s", stdout, wantLine, running)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	report := directorReportOf(t, jsonOut)
	if report.Registration == nil || report.Registration.ID != "d2" || report.Registration.Launch != "" || len(report.OpenHeadless) != 1 ||
		report.OpenHeadless[0].ID != "d1" || report.OpenHeadless[0].PaneID != "p-1" || report.OpenHeadless[0].WorkspaceID != "w-1" {
		t.Errorf("director status --json = %s; want d2 current and d1 in open_headless", jsonOut)
	}

	if _, err := store.Detach(org.DirectorDetach{ID: "d2", User: "bob"}); err != nil {
		t.Fatal(err)
	}
	seedHeadless(t, stateDir, "claude", "opus", "", "", false)
	if _, err := store.Detach(org.DirectorDetach{ID: "d3", User: "bob"}); err != nil {
		t.Fatal(err)
	}
	wantLine = directorNotAttachedLine + "; the headless directors " + running + ", d3 (starting) are still open but not attached: " +
		"stop each with ralph org director stop --id <id>" + suffix
	if got := directorStatusLine(t, "--state-dir", stateDir); got != wantLine {
		t.Errorf("director line with two open headless directors:\n got %q\nwant %q", got, wantLine)
	}
}

// D7 through the CLI: `ralph org spawn --role director` is refused before
// the --model fallback, so with no --model it prints no fallback warning,
// prints the refusal as a rejected spawn, exits 1, and writes nothing and
// runs neither herdr nor agmsg.
func TestOrgSpawn_RoleDirectorIsRefusedBeforeTheModelFallback(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := t.TempDir()
	stdout, stderr, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "director",
		"--driver", "claude", "--cwd", t.TempDir(), "--state-dir", stateDir)
	want := `org: role "director" is not a seat role: the director is not an org seat; launch a headless director with ralph org director start, ` +
		"or promote an interactive session with ralph org director attach"
	if err == nil || err.Error() != want {
		t.Fatalf("spawn --role director = %v; want %q", err, want)
	}
	if stdout != "rejected: "+want+"\n" {
		t.Errorf("stdout = %q; want the rejected line", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want no --model fallback warning", stderr)
	}
	checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
}

// The check-in `ralph org director launched` of a launch cmd's script (plan
// 2026-10-16-org-director-launch-cmd, AC3 through the CLI). The tests run no
// launch script: the check-in's parent is the test binary's own, or a
// process the test started and gives through orgDirectorCheckInParentPID.

// seedCmdHeadless records a launch cmd registration in the ledger stateDir
// through the store, as start does before it runs launch_cmd, and returns it
// (with its token).
func seedCmdHeadless(t *testing.T, stateDir string) org.DirectorRegistration {
	t.Helper()
	reg, err := org.NewDirectorStore(stateDir).AttachHeadless(org.DirectorHeadlessAttach{
		Launch: org.DirectorLaunchCmd, Driver: "claude", Model: "opus", PermissionMode: org.PermissionModeAutonomous, Wake: []string{org.DirectorWakePull},
		User: "alice", Cwd: "/work",
	})
	if err != nil {
		t.Fatalf("seed the launch cmd registration: %v", err)
	}
	return reg
}

// setCheckInSeams points the check-in at parent as its parent pid and, when
// read is non-nil, at read for the start time, until the test ends.
func setCheckInSeams(t *testing.T, parent int, read func(context.Context, int) (org.DirectorProcess, error)) {
	t.Helper()
	oldParent, oldRead := orgDirectorCheckInParentPID, orgDirectorCheckInReadProcess
	t.Cleanup(func() { orgDirectorCheckInParentPID, orgDirectorCheckInReadProcess = oldParent, oldRead })
	orgDirectorCheckInParentPID = func() int { return parent }
	if read != nil {
		orgDirectorCheckInReadProcess = read
	}
}

// launchedOf returns the launched events in stateDir's director.jsonl.
func launchedOf(t *testing.T, stateDir string) []org.DirectorEvent {
	t.Helper()
	var launched []org.DirectorEvent
	for _, ev := range directorEventsIn(t, stateDir) {
		if ev.Event == org.DirectorEventLaunched {
			launched = append(launched, ev)
		}
	}
	return launched
}

// The check-in is for the launch script alone, so neither `ralph org
// director --help` nor `ralph org --help` lists it; its own help still
// says what it does.
func TestOrgDirectorLaunched_IsHidden(t *testing.T) {
	for _, args := range [][]string{{"director", "--help"}, {"--help"}, {"director"}} {
		out, err := runOrgCmd(t, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		for _, line := range strings.Split(out, "\n") {
			if fields := strings.Fields(line); len(fields) > 0 && fields[0] == "launched" {
				t.Errorf("%v lists the hidden check-in: %q", args, line)
			}
		}
		if strings.Contains(out, "Check in the launch script") {
			t.Errorf("%v shows the check-in's short help:\n%s", args, out)
		}
	}
	out, err := runOrgCmd(t, "director", "launched", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"the launch script that `ralph org director start` writes when [org.director].launch_cmd is set",
		`(--pid "$$") before it execs the agent, and execs the agent only when it exits 0`,
		"It is refused, writes nothing, and exits 1 with the reason on stderr, so the agent does not start",
		// Plan 2026-10-18-org-director-liveness, D2.
		"It is refused the same way when the `ralph org director start` that recorded the registration has ended",
		"a ps that cannot read that pid does not refuse it.",
		"It reads no ralph.toml.", "--id", "--token", "--pid",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("director launched --help does not say %q:\n%s", want, out)
		}
	}
}

// AC3: the check-in records launched with --pid, the pid of the process
// that ran it (here the test binary's parent, read with the real os.Getppid
// and ps), and that pid's start time as org.ReadDirectorProcess reads it,
// and prints nothing. It reads no ralph.toml, so a ./ralph.toml and a
// --config that do not load do not block it.
func TestOrgDirectorLaunched_RecordsThePIDAndItsStartTime(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "ralph.toml")
	writeTestFile(t, broken, "[org\n")
	t.Chdir(dir)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	stateDir := t.TempDir()
	reg := seedCmdHeadless(t, stateDir)
	ppid := os.Getppid()
	stdout, stderr, err := runOrgCmdStreams(t, "director", "launched", "--id", reg.ID, "--token", reg.Token, "--pid", fmt.Sprint(ppid),
		"--state-dir", stateDir, "--config", broken)
	if err != nil || stdout != "" || stderr != "" {
		t.Fatalf("director launched = %v, stdout %q, stderr %q; want nil and nothing printed", err, stdout, stderr)
	}
	proc, err := org.ReadDirectorProcess(context.Background(), ppid)
	if err != nil || proc.State != org.DirectorProcessAlive || proc.StartTime == "" {
		t.Fatalf("ReadDirectorProcess(%d) = %+v, %v", ppid, proc, err)
	}
	launched := launchedOf(t, stateDir)
	if len(launched) != 1 || launched[0].ID != reg.ID || launched[0].PID != ppid || launched[0].StartTime != proc.StartTime ||
		launched[0].User != os.Getenv("USER") || launched[0].Cwd != cwd {
		t.Fatalf("launched = %+v; want one for %s with pid %d, start time %q, user %q, cwd %q", launched, reg.ID, ppid, proc.StartTime, os.Getenv("USER"), cwd)
	}
	res, err := org.NewDirectorStore(stateDir).Read()
	if err != nil || res.Current == nil || res.Current.State != org.DirectorStateRunning || res.Current.PID != ppid {
		t.Fatalf("Read = %+v, %v; want %s running with pid %d", res, err, reg.ID, ppid)
	}
}

// The start time recorded is --pid's, not the check-in's own parent's: a
// process the test started, given as the parent through the seam.
func TestOrgDirectorLaunched_RecordsTheStartTimeOfTheGivenPID(t *testing.T) {
	sleep := exec.Command("sleep", "30")
	if err := sleep.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sleep.Process.Kill()
		_ = sleep.Wait()
	})
	pid := sleep.Process.Pid
	setCheckInSeams(t, pid, nil)
	stateDir := t.TempDir()
	reg := seedCmdHeadless(t, stateDir)
	runInboxOK(t, "director", "launched", "--id", reg.ID, "--token", reg.Token, "--pid", fmt.Sprint(pid), "--state-dir", stateDir)
	proc, err := org.ReadDirectorProcess(context.Background(), pid)
	if err != nil || proc.State != org.DirectorProcessAlive {
		t.Fatalf("ReadDirectorProcess(%d) = %+v, %v", pid, proc, err)
	}
	if launched := launchedOf(t, stateDir); len(launched) != 1 || launched[0].PID != pid || launched[0].StartTime != proc.StartTime {
		t.Fatalf("launched = %+v; want pid %d started %q", launched, pid, proc.StartTime)
	}
}

// AC3: every other check-in is refused with one line, writes nothing, and
// prints nothing on stdout, so the script exits without its agent: missing
// flags (before the state dir is made), a --pid that is not the parent, a
// pid whose start time ps does not read or that is not running, and what
// the store refuses (another token, a stopped, herdr, or unknown
// registration, a second check-in).
func TestOrgDirectorLaunched_Refusals(t *testing.T) {
	t.Run("missing flags", func(t *testing.T) {
		stateDir := filepath.Join(t.TempDir(), "state")
		const want = "org: director launched: give --id, --token, and --pid; the launch script that ralph org director start writes runs this check-in itself"
		for _, args := range [][]string{
			{"--token", "0123456789ab", "--pid", "1"},
			{"--id", "d1", "--pid", "1"},
			{"--id", "d1", "--token", "0123456789ab"},
			{},
		} {
			stdout, _, err := runOrgCmdStreams(t, append(append([]string{"director", "launched"}, args...), "--state-dir", stateDir)...)
			if err == nil || err.Error() != want || stdout != "" {
				t.Errorf("launched %v = %v, stdout %q; want %q", args, err, stdout, want)
			}
		}
		if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("a refused check-in created the state dir (stat: %v)", err)
		}
	})

	gone := func(context.Context, int) (org.DirectorProcess, error) {
		return org.DirectorProcess{State: org.DirectorProcessGone}, nil
	}
	zombie := func(context.Context, int) (org.DirectorProcess, error) {
		return org.DirectorProcess{State: org.DirectorProcessZombie, StartTime: "Thu Oct 15 21:51:09 2026"}, nil
	}
	unreadable := func(context.Context, int) (org.DirectorProcess, error) {
		return org.DirectorProcess{}, errors.New("read process 4242: ps: exit status 2: bad\nline")
	}
	alive := func(context.Context, int) (org.DirectorProcess, error) {
		return org.DirectorProcess{State: org.DirectorProcessAlive, StartTime: "Thu Oct 15 21:51:09 2026"}, nil
	}
	for _, tc := range []struct {
		name   string
		seed   func(t *testing.T, stateDir string) (id, token string)
		parent int // 0 for the real os.Getppid
		pid    int // 0 for the parent
		read   func(context.Context, int) (org.DirectorProcess, error)
		want   string // the whole error, with %s for the registration's attached ts
	}{
		{name: "a --pid that is not the parent", pid: os.Getpid(),
			want: "org: director launched d1: the check-in of the launch script is refused, so its agent must not start: --pid " + fmt.Sprint(os.Getpid()) +
				" is not the pid of the process that ran this check-in (" + fmt.Sprint(os.Getppid()) + `); the launch script passes its own ("$$")`},
		{name: "a ps that cannot be read", parent: 4242, read: unreadable,
			want: `org: director launched d1: the check-in of the launch script is refused, so its agent must not start: the start time of pid 4242 cannot be read: read process 4242: ps: exit status 2: bad\nline`},
		{name: "a pid that is gone", parent: 4242, read: gone,
			want: "org: director launched d1: the check-in of the launch script is refused, so its agent must not start: pid 4242 is gone, not a running process"},
		{name: "a zombie", parent: 4242, read: zombie,
			want: "org: director launched d1: the check-in of the launch script is refused, so its agent must not start: pid 4242 is zombie, not a running process"},
		{name: "another token", parent: 4242, read: alive,
			seed: func(t *testing.T, stateDir string) (string, string) {
				reg := seedCmdHeadless(t, stateDir)
				return reg.ID, "aaaaaaaaaaaa"
			},
			want: "org: director: check in d1: the check-in of the launch script is refused, so its agent must not start: the token is not the one d1 was started with"},
		{name: "a stopped registration", parent: 4242, read: alive,
			seed: func(t *testing.T, stateDir string) (string, string) {
				reg := seedCmdHeadless(t, stateDir)
				if _, err := org.NewDirectorStore(stateDir).MarkStopped(reg.ID, org.DirectorWho{User: "alice"}); err != nil {
					t.Fatal(err)
				}
				return reg.ID, reg.Token
			},
			want: "org: director: check in d1: the check-in of the launch script is refused, so its agent must not start: " +
				"d1 (driver claude, attached %s, headless stopped, launch cmd) is already recorded stopped (start's cleanup or a stop came first)"},
		{name: "a launch herdr registration", parent: 4242, read: alive,
			seed: func(t *testing.T, stateDir string) (string, string) {
				reg := seedHeadless(t, stateDir, "claude", "opus", "", "", false)
				return reg.ID, reg.Token
			},
			want: "org: director: check in d1: the check-in of the launch script is refused, so its agent must not start: " +
				"d1 (driver claude, attached %s, headless starting) was not started with [org.director].launch_cmd"},
		{name: "a second check-in", parent: 4242, read: alive,
			seed: func(t *testing.T, stateDir string) (string, string) {
				reg := seedCmdHeadless(t, stateDir)
				if err := org.NewDirectorStore(stateDir).MarkCmdLaunched(org.DirectorCmdCheckIn{ID: reg.ID, Token: reg.Token, PID: 99, StartTime: "Wed Oct 14 00:00:00 2026"}); err != nil {
					t.Fatal(err)
				}
				return reg.ID, reg.Token
			},
			want: "org: director: check in d1: the check-in of the launch script is refused, so its agent must not start: " +
				"d1 (driver claude, attached %s, headless running, launch cmd, pid 99) has already checked in"},
		{name: "an unknown registration", parent: 4242, read: alive,
			seed: func(*testing.T, string) (string, string) { return "d9", "0123456789ab" },
			want: "org: director: check in d9: the check-in of the launch script is refused, so its agent must not start: no registration d9 is recorded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			seed := tc.seed
			if seed == nil {
				seed = func(t *testing.T, stateDir string) (string, string) {
					reg := seedCmdHeadless(t, stateDir)
					return reg.ID, reg.Token
				}
			}
			id, token := seed(t, stateDir)
			if tc.parent != 0 {
				setCheckInSeams(t, tc.parent, tc.read)
			}
			pid := tc.pid
			if pid == 0 {
				pid = orgDirectorCheckInParentPID()
			}
			before := directorFileOrAbsent(t, stateDir)
			want := tc.want
			if strings.Contains(want, "%s") {
				want = fmt.Sprintf(want, directorEventsIn(t, stateDir)[0].TS)
			}
			stdout, _, err := runOrgCmdStreams(t, "director", "launched", "--id", id, "--token", token, "--pid", fmt.Sprint(pid), "--state-dir", stateDir)
			if !errors.Is(err, org.ErrDirectorCheckInRefused) || err.Error() != want || strings.Contains(err.Error(), "\n") {
				t.Fatalf("director launched err =\n%v\nwant\n%s", err, want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused check-in changed director.jsonl from %q to %q", before, after)
			}
		})
	}
}

// Plan 2026-10-18-org-director-liveness, AC2 through the CLI: the check-in
// reads the start process that recorded the registration with the same seam
// it reads --pid with (orgDirectorCheckInReadProcess). A starter read gone
// refuses it with one line and exit 1, writing nothing and printing nothing
// on stdout; a starter still running with its recorded start time lets the
// check-in record launched as before.
func TestOrgDirectorLaunched_RefusesOnceTheStartThatRecordedItHasEnded(t *testing.T) {
	const (
		scriptPID  = 4242
		starterPID = 7777
		scriptTime = "Thu Oct 15 21:51:09 2026"
		started    = "Wed Oct 14 00:00:00 2026"
	)
	for _, tc := range []struct {
		name    string
		starter org.DirectorProcess
		want    string // the whole refusal, with %s for the attached ts; "" when the check-in is recorded
	}{
		{"the start has ended", org.DirectorProcess{State: org.DirectorProcessGone},
			"org: director: check in d1: the check-in of the launch script is refused, so its agent must not start: " +
				"the ralph org director start that recorded d1 (driver claude, attached %s, headless starting, launch cmd) is no longer running " +
				"(no process has pid 7777), so no start waits for this check-in: run ralph org director start again"},
		{"the start still runs", org.DirectorProcess{State: org.DirectorProcessAlive, StartTime: started}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			reg, err := org.NewDirectorStore(stateDir).AttachHeadless(org.DirectorHeadlessAttach{
				Launch: org.DirectorLaunchCmd, Driver: "claude", Model: "opus", PermissionMode: org.PermissionModeAutonomous, Wake: []string{org.DirectorWakePull},
				StarterPID: starterPID, StarterStartTime: started, User: "alice", Cwd: "/work",
			})
			if err != nil {
				t.Fatalf("seed the launch cmd registration: %v", err)
			}
			var reads []int
			setCheckInSeams(t, scriptPID, func(_ context.Context, pid int) (org.DirectorProcess, error) {
				reads = append(reads, pid)
				switch pid {
				case scriptPID:
					return org.DirectorProcess{State: org.DirectorProcessAlive, StartTime: scriptTime}, nil
				case starterPID:
					return tc.starter, nil
				}
				t.Errorf("the check-in read pid %d", pid)
				return org.DirectorProcess{}, errors.New("not a pid of the test's")
			})
			before := directorFileOrAbsent(t, stateDir)
			stdout, _, err := runOrgCmdStreams(t, "director", "launched", "--id", reg.ID, "--token", reg.Token, "--pid", fmt.Sprint(scriptPID), "--state-dir", stateDir)
			if !slices.Equal(reads, []int{scriptPID, starterPID}) {
				t.Errorf("the check-in read the pids %v, want --pid %d and then the starter %d", reads, scriptPID, starterPID)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if tc.want != "" {
				want := fmt.Sprintf(tc.want, directorEventsIn(t, stateDir)[0].TS)
				if !errors.Is(err, org.ErrDirectorCheckInRefused) || err.Error() != want || strings.Contains(err.Error(), "\n") {
					t.Fatalf("director launched err =\n%v\nwant\n%s", err, want)
				}
				if after := directorFileOrAbsent(t, stateDir); after != before {
					t.Fatalf("a refused check-in changed director.jsonl from %q to %q", before, after)
				}
				return
			}
			if err != nil {
				t.Fatalf("director launched: %v", err)
			}
			if launched := launchedOf(t, stateDir); len(launched) != 1 || launched[0].PID != scriptPID || launched[0].StartTime != scriptTime {
				t.Fatalf("launched = %+v; want one with pid %d started %q", launched, scriptPID, scriptTime)
			}
		})
	}
}

// The tests of director start with [org.director].launch_cmd through the
// CLI (plan 2026-10-16-org-director-launch-cmd, AC2, AC4, AC5, and D10). The
// launch script start writes runs for real: its
// check-in is this test binary run as `ralph org director launched`
// (TestOrgDirectorCheckInHelperProcess, through a ralph wrapper), and its
// agent is a fake that records what it got and becomes a sleep, which the
// test kills in its cleanup (killDirectorLaunchedProcesses). The stub herdr
// stays on PATH and must log no call.

// orgDirectorCheckInHelperEnv makes this test binary, run by the ralph
// wrapper of setDirectorLaunchSeams, the check-in of a launch script.
const orgDirectorCheckInHelperEnv = "RALPH_CLI_TEST_DIRECTOR_CHECKIN_HELPER"

// TestOrgDirectorCheckInHelperProcess is not a test of its own: with
// orgDirectorCheckInHelperEnv set, it runs the root command with the
// arguments after "--" (`org director launched ...`, as the launch script
// gives them) and exits 0 or, writing the error on stderr, 1. The wrapper
// execs this binary, so its parent is the launch script's shell, which the
// real check-in reads (os.Getppid and ps).
func TestOrgDirectorCheckInHelperProcess(t *testing.T) {
	if os.Getenv(orgDirectorCheckInHelperEnv) != "1" {
		return
	}
	root := NewRootCmd()
	root.SetArgs(flag.Args())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

// directorLaunchFakes are what setDirectorLaunchSeams set up: the file the
// fake agent writes once it runs (its pid, RALPH_ORG_AGMSG_HOME or
// "<unset>", and each argument), and the names the agent lookup was asked
// for.
type directorLaunchFakes struct {
	agentMarker string
	lookedUp    []string
}

// setDirectorLaunchSeams points director start's agent lookup at a fake
// agent and its ralph executable at a wrapper that runs
// TestOrgDirectorCheckInHelperProcess, until the test ends.
func setDirectorLaunchSeams(t *testing.T) *directorLaunchFakes {
	t.Helper()
	base := t.TempDir()
	f := &directorLaunchFakes{agentMarker: filepath.Join(base, "agent.out")}
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Fatal(err)
	}
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	agent, ralph := filepath.Join(base, "agent"), filepath.Join(base, "ralph")
	marker := org.ShellWord(f.agentMarker)
	writeExecutable := func(path, content string) {
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeExecutable(agent, "#!/bin/sh\n{\nprintf 'pid=%s\\n' \"$$\"\nprintf 'agmsg=%s\\n' \"${RALPH_ORG_AGMSG_HOME-<unset>}\"\n"+
		"for a in \"$@\"; do printf 'arg=%s\\n' \"$a\"; done\n} > "+marker+".tmp && mv "+marker+".tmp "+marker+"\nexec "+org.ShellWord(sleep)+" 600\n")
	writeExecutable(ralph, "#!/bin/sh\n"+orgDirectorCheckInHelperEnv+"=1\nexport "+orgDirectorCheckInHelperEnv+"\nexec "+org.ShellWord(testBinary)+
		" -test.run='^TestOrgDirectorCheckInHelperProcess$' -- \"$@\"\n")
	oldLookPath, oldExecutable := orgDirectorAgentLookPath, orgDirectorRalphExecutable
	t.Cleanup(func() { orgDirectorAgentLookPath, orgDirectorRalphExecutable = oldLookPath, oldExecutable })
	orgDirectorAgentLookPath = func(file string) (string, error) {
		f.lookedUp = append(f.lookedUp, file)
		return agent, nil
	}
	orgDirectorRalphExecutable = func() (string, error) { return ralph, nil }
	return f
}

// killDirectorLaunchedProcesses is t's cleanup for every process a launch
// script of the ledger stateDir checked in with: each pid of a launched
// event that ps still reads alive with the start time that event recorded
// gets SIGKILL, and the cleanup waits (up to 5s) for it to be gone. No
// other process is signalled.
func killDirectorLaunchedProcesses(t *testing.T, stateDir string) {
	t.Cleanup(func() {
		raw, err := os.ReadFile(org.DirectorPathIn(stateDir))
		if err != nil {
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			var ev org.DirectorEvent
			if json.Unmarshal([]byte(line), &ev) != nil || ev.Event != org.DirectorEventLaunched || ev.PID <= 0 {
				continue
			}
			same := func() bool {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				proc, err := org.ReadDirectorProcess(ctx, ev.PID)
				return err == nil && proc.State == org.DirectorProcessAlive && proc.StartTime == ev.StartTime
			}
			if !same() {
				continue
			}
			_ = syscall.Kill(ev.PID, syscall.SIGKILL)
			for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && same(); time.Sleep(20 * time.Millisecond) {
			}
		}
	})
}

// waitForDirectorAgentMarker returns the lines the fake agent wrote, once it
// has run (up to 30s).
func waitForDirectorAgentMarker(t *testing.T, path string) []string {
	t.Helper()
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(path); err == nil {
			return readLogLines(t, path)
		}
		if time.Now().After(deadline) {
			t.Fatalf("the fake agent did not write %s within 30s", path)
		}
	}
}

// directorLaunchCmdToml is a ralph.toml whose launch_cmd runs the launch
// script in the background, as a terminal would, and returns at once. Like
// a terminal, which has its own environment, the script does not get
// RALPH_ORG_AGMSG_HOME from start's, so the agent has it only when the
// script exports it, which it must not (plan 2026-10-29-local-config-org,
// AC8).
const directorLaunchCmdToml = "[org.director]\nlaunch_cmd = [\"sh\", \"-c\", 'env -u RALPH_ORG_AGMSG_HOME nohup sh \"$1\" >/dev/null 2>&1 &', \"sh\", \"{script}\"]\n"

// directorTouchingLaunchCmdToml is a ralph.toml whose launch_cmd runs no
// launch script: it creates path (so a test sees which one ran) and exits 3.
func directorTouchingLaunchCmdToml(path string) string {
	return fmt.Sprintf("[org.director]\nlaunch_cmd = [\"sh\", \"-c\", ': > \"$2\"; exit 3', \"sh\", \"{script}\", %q]\n", path)
}

// AC2 through the CLI: with launch_cmd, director start opens the terminal
// with it, waits for the launch script's check-in, and prints the
// registration with launch=cmd, the terminal's name, and the pid the script
// checked in with, where to talk to the director, and the stop command with
// --state-dir. RALPH_ORG_AGMSG_HOME set here gets no warning, and the launch
// script does not export it (plan 2026-10-29-local-config-org, AC8), so the
// agent runs without it, with the permission flags, --model, and the prompt
// pointer. No herdr is called.
func TestOrgDirectorStart_LaunchCmdOpensTheTerminal(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	t.Setenv(legacyAgmsgHomeEnv, "/agmsg home/it's")
	fakes := setDirectorLaunchSeams(t)
	killDirectorLaunchedProcesses(t, stateDir)
	configPath := writeDirectorConfig(t, directorLaunchCmdToml)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", configPath)
	if err != nil {
		t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	events := directorEventsIn(t, stateDir)
	if got, want := directorEventKinds(events), []string{"attached d1", "launched d1"}; !slices.Equal(got, want) {
		t.Fatalf("director.jsonl = %v; want %v", got, want)
	}
	att, launched := events[0], events[1]
	if att.Launch != org.DirectorLaunchCmd || len(att.Token) != 12 || att.Model != "opus" || att.PermissionMode != org.PermissionModeAutonomous || launched.PID <= 0 {
		t.Fatalf("attached = %+v, launched = %+v; want launch cmd, a token, opus, autonomous, and a pid", att, launched)
	}
	tok := att.Token
	wantOut := fmt.Sprintf("started d1 (driver=claude, model=opus, permission=autonomous, launch=cmd, name=director_%s, pid=%d)\n", tok, launched.PID) +
		"talk to the director in the terminal that [org.director].launch_cmd opened (name director_" + tok + ")\n" +
		"stop it with: ralph org director stop --id d1 --state-dir '" + stateDir + "'\n"
	if stdout != wantOut {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, wantOut)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing (no RALPH_ORG_AGMSG_HOME warning)", stderr)
	}
	prompt := filepath.Join(stateDir, "prompts", "director_"+tok+".md")
	want := []string{"pid=" + fmt.Sprint(launched.PID), "agmsg=<unset>",
		"arg=--permission-mode", "arg=bypassPermissions", "arg=--model", "arg=opus", "arg=" + org.PromptFilePointer(prompt)}
	if got := waitForDirectorAgentMarker(t, fakes.agentMarker); !slices.Equal(got, want) {
		t.Errorf("the agent recorded\n%q\nwant\n%q", got, want)
	}
	if !slices.Equal(fakes.lookedUp, []string{"claude"}) {
		t.Errorf("the agent lookup was asked for %q; want [claude]", fakes.lookedUp)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none with launch_cmd", lines)
	}
}

// AC4 through the CLI: a launch_cmd that exits non-zero ends start with exit
// 1 and nothing on stdout; the error says what failed, that the
// registration is recorded stopped, and that the terminal it may have
// opened is left for the human to close while a late check-in from it is
// refused. No herdr is called.
func TestOrgDirectorStart_AFailedLaunchCmdLeavesTheTerminalToTheHuman(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	setDirectorLaunchSeams(t)
	ran := filepath.Join(t.TempDir(), "launch_cmd ran")
	configPath := writeDirectorConfig(t, directorTouchingLaunchCmdToml(ran))

	stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", configPath)
	events := directorEventsIn(t, stateDir)
	if got, want := directorEventKinds(events), []string{"attached d1", "detached d1", "stopped d1"}; !slices.Equal(got, want) {
		t.Fatalf("director.jsonl = %v; want %v", got, want)
	}
	want := "org: director: start d1: launch_cmd sh failed: exit status 3; d1 is recorded stopped; the terminal that [org.director].launch_cmd may have opened " +
		"(name director_" + events[0].Token + ") is left open for you to close, and a launch script that checks in from it now is refused and ends without starting the agent"
	if err == nil || err.Error() != want {
		t.Fatalf("director start err =\n%v\nwant\n%s", err, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if _, err := os.Stat(ran); err != nil {
		t.Errorf("launch_cmd did not run: %v", err)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
}

// startError on a cleanup of a launch cmd that did not finish: the stop
// command it names gets this invocation's --state-dir, and the error adds
// that the terminal is to be closed once that stop is recorded. Nothing in it
// names herdr.
func TestOrgDirectorStart_ALaunchCmdCleanupErrorNamesTheStopAndTheTerminal(t *testing.T) {
	rt := &orgDirectorRuntime{hintStateDir: "/work/the ledger"}
	cleanup := &org.DirectorCleanupError{
		Verb: "start", ID: "d4", Launch: org.DirectorLaunchCmd, TerminalName: "director_0123456789ab", PID: 77, Detached: true,
		Err:         errors.New("org: director: start d4: launch_cmd sh failed: exit status 1"),
		CleanupErr:  errors.New("d4 is not recorded stopped, since the end of the process that checked in is not confirmed"),
		StopCommand: "ralph org director stop --id d4",
	}
	err := rt.startError(cleanup)
	want := "org: director: start d4: launch_cmd sh failed: exit status 1; its cleanup did not finish: d4 is not recorded stopped, since the end of the process " +
		"that checked in is not confirmed; d4 was recorded detached only, so it is no longer the current registration but stays open; " +
		"record the stop with ralph org director stop --id d4 --state-dir '/work/the ledger'; " +
		"the terminal that [org.director].launch_cmd may have opened (name director_0123456789ab) is left open: close it once the stop is recorded"
	if err == nil || err.Error() != want {
		t.Fatalf("startError =\n%v\nwant\n%s", err, want)
	}
	var ce *org.DirectorCleanupError
	if !errors.As(err, &ce) || ce != cleanup {
		t.Errorf("startError = %v; want it to still wrap the cleanup error", err)
	}
}

// start's refusal while a launch cmd registration is current names its stop
// from a terminal other than the director's, not one outside herdr.
func TestOrgDirectorStart_AnAttachedLaunchCmdRefusalNamesItsStop(t *testing.T) {
	refusal := fmt.Errorf("org: director: start: %w: d2 (driver claude, attached 2026-10-15T09:00:00Z)", org.ErrDirectorAttached)
	rt := &orgDirectorRuntime{hintStateDir: "/work/the ledger"}
	err := rt.startAttachedError(&org.DirectorAttachedError{Err: refusal, Current: &org.DirectorRegistration{
		ID: "d2", Driver: "claude", Launch: org.DirectorLaunchCmd, State: org.DirectorStateRunning, PID: 77,
	}})
	want := refusal.Error() + ": stop it with ralph org director stop --id d2 --state-dir '/work/the ledger' from a terminal other than the director's, " +
		"then run start again (start has no --replace)"
	if err == nil || err.Error() != want {
		t.Fatalf("startAttachedError =\n%v\nwant\n%s", err, want)
	}
	if !errors.Is(err, org.ErrDirectorAttached) {
		t.Errorf("startAttachedError = %v; want it to still wrap ErrDirectorAttached", err)
	}
}

// AC5 through the CLI (D10): a relative [org].agmsg_home of a --config is
// refused with launch_cmd too, before anything is written or run.
func TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWithALaunchCmd(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	setDirectorLaunchSeams(t)
	ran := filepath.Join(t.TempDir(), "launch_cmd ran")
	configPath := writeDirectorConfig(t, "[org]\nagmsg_home = \"agmsg/home\"\n\n"+directorTouchingLaunchCmdToml(ran))

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", configPath)
	if want := directorRelativeAgmsgHomeRefusal("agmsg/home", configPath); err == nil || err.Error() != want {
		t.Fatalf("director start = %v; want %q", err, want)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
	}
	if got := dirNames(t, stateDir); len(got) != 0 {
		t.Errorf("ledger entries = %v; want none", got)
	}
	if _, err := os.Stat(ran); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused start ran launch_cmd (stat %v)", err)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
}

// AC8 of plan 2026-10-29-local-config-org through the CLI: a relative
// RALPH_ORG_AGMSG_HOME set where start runs (here a linked worktree), which
// start used to make absolute and pass on, no longer reaches the director
// launched with launch_cmd: the launch script names no RALPH_ORG_AGMSG_HOME,
// the agent has none, and start says nothing on stderr.
func TestOrgDirectorStart_LaunchCmdPassesTheDirectorNoAgmsgHomeVariable(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	t.Setenv(legacyAgmsgHomeEnv, "agmsg/home")
	fakes := setDirectorLaunchSeams(t)
	killDirectorLaunchedProcesses(t, r.sharedDir)
	writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), directorLaunchCmdToml)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus")
	if err != nil || !strings.HasPrefix(stdout, "started d1 (driver=claude, model=opus, permission=autonomous, launch=cmd, ") || stderr != "" {
		t.Fatalf("director start = %v (stdout: %s stderr: %s); want d1 started with launch_cmd and nothing on stderr", err, stdout, stderr)
	}
	if _, err := os.Stat(org.DirectorPathIn(r.sharedDir)); err != nil {
		t.Fatalf("the main worktree's ledger has no director.jsonl (%v), so the cleanup would not find the agent", err)
	}
	if got := waitForDirectorAgentMarker(t, fakes.agentMarker); len(got) < 2 || got[1] != "agmsg=<unset>" {
		t.Fatalf("the agent recorded %q; want agmsg=<unset>", got)
	}
	tok := directorEventsIn(t, r.sharedDir)[0].Token
	script, err := os.ReadFile(filepath.Join(r.sharedDir, "launch", "director_"+tok+".sh"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(script), legacyAgmsgHomeEnv) {
		t.Errorf("the launch script names %s; want none:\n%s", legacyAgmsgHomeEnv, script)
	}
}

// D10 through the CLI: without --config, on the main worktree's ledger
// found from a linked worktree, start takes launch_cmd from the main
// worktree's ralph.toml, never the linked worktree's: the main worktree's
// launch_cmd runs, and with none there start uses herdr although the
// worktree has one. --config names the file to read instead, so its
// launch_cmd runs.
func TestOrgDirectorStart_ReadsLaunchCmdFromTheMainWorktree(t *testing.T) {
	for _, tc := range []struct {
		name          string
		mainHas       bool
		config        bool // --config ralph.toml (the worktree's)
		wantRan       string
		wantHerdrUsed bool
	}{
		{"no --config: the main worktree's launch_cmd", true, false, "main", false},
		{"no --config: the main worktree has none", false, false, "", true},
		{"--config names the worktree's ralph.toml", true, true, "worktree", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			setDirectorLaunchSeams(t)
			r := newLegacyLedgerRepo(t)
			marks := t.TempDir()
			ranMain, ranWorktree := filepath.Join(marks, "main"), filepath.Join(marks, "worktree")
			mainToml := ""
			if tc.mainHas {
				mainToml = directorTouchingLaunchCmdToml(ranMain)
			}
			writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), mainToml)
			writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), directorTouchingLaunchCmdToml(ranWorktree))
			args := []string{"director", "start", "--driver", "claude", "--model", "opus"}
			if tc.config {
				args = append(args, "--config", "ralph.toml")
			}

			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if tc.wantHerdrUsed {
				if err != nil || !strings.HasPrefix(stdout, "started d1 (driver=claude, model=opus, permission=autonomous, pane=pane-stub-1, ") {
					t.Fatalf("director start = %v (stdout: %s stderr: %s); want d1 started in herdr", err, stdout, stderr)
				}
				if lines := readLogLines(t, herdrLog); len(lines) != 3 {
					t.Errorf("herdr calls = %v; want the three of a herdr start", lines)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), "launch_cmd sh failed: exit status 3; d1 is recorded stopped") {
					t.Fatalf("director start = %v (stdout: %s stderr: %s); want launch_cmd's failure", err, stdout, stderr)
				}
				if lines := readLogLines(t, herdrLog); len(lines) != 0 {
					t.Errorf("herdr calls = %v; want none", lines)
				}
			}
			for _, m := range []struct{ name, path string }{{"main", ranMain}, {"worktree", ranWorktree}} {
				_, err := os.Stat(m.path)
				if ran := err == nil; ran != (m.name == tc.wantRan) {
					t.Errorf("the %s worktree's launch_cmd ran: %v; want %v", m.name, ran, m.name == tc.wantRan)
				}
			}
		})
	}
}

// The tests of director stop, attach, detach, and status on a launch cmd's
// registration through the CLI (plan 2026-10-16-org-director-launch-cmd, D7,
// D8, AC6 to AC8). A stop signals only a process the test started itself:
// the launch script of a real start (killDirectorLaunchedProcesses cleans
// up after it), or startDirectorTestProcess's. A registration that no stop
// runs on may carry a made-up pid, since nothing then reads or signals it.

// directorTestStartTime is the start time the tests record with a made-up
// pid.
const directorTestStartTime = "Thu Oct 15 21:51:09 2026"

// directorTestProcess is a process a test started (startDirectorTestProcess).
type directorTestProcess struct {
	cmd       *exec.Cmd
	startTime string // as ps reads it once the process runs its sleep
	waited    bool
}

// startDirectorTestProcess starts what a launch script leaves running, a
// shell that writes a marker and becomes a sleep of 600 s, ignoring SIGTERM
// when ignoreTerm (an ignored signal stays ignored across the exec), and
// returns it once it runs the sleep, with its start time. It is killed
// (SIGKILL) and waited for in t's cleanup unless the test waited for it;
// until then its pid stays its own, so the kill reaches no other process.
func startDirectorTestProcess(t *testing.T, ignoreTerm bool) *directorTestProcess {
	t.Helper()
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "ran")
	trap := ""
	if ignoreTerm {
		trap = "trap '' TERM; "
	}
	cmd := exec.Command("/bin/sh", "-c", trap+": > "+org.ShellWord(marker)+"; exec "+org.ShellWord(sleep)+" 600")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p := &directorTestProcess{cmd: cmd}
	t.Cleanup(func() {
		if !p.waited {
			_ = cmd.Process.Kill()
			p.wait()
		}
	})
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the test process did not start within 10s")
		}
	}
	time.Sleep(100 * time.Millisecond)
	p.startTime = p.read(t).StartTime
	return p
}

func (p *directorTestProcess) pid() int { return p.cmd.Process.Pid }

func (p *directorTestProcess) wait() {
	if !p.waited {
		_ = p.cmd.Wait()
		p.waited = true
	}
}

// read is ps's reading of p.
func (p *directorTestProcess) read(t *testing.T) org.DirectorProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	proc, err := org.ReadDirectorProcess(ctx, p.pid())
	if err != nil {
		t.Fatalf("read pid %d: %v", p.pid(), err)
	}
	return proc
}

// wantRunning fails t unless p, after a moment a SIGTERM would need to end
// it, still runs with its start time: nothing was sent to it, or it ignored
// what was.
func (p *directorTestProcess) wantRunning(t *testing.T, what string) {
	t.Helper()
	time.Sleep(200 * time.Millisecond)
	if proc := p.read(t); proc.State != org.DirectorProcessAlive || proc.StartTime != p.startTime {
		t.Fatalf("pid %d reads %+v, want it still running: %s", p.pid(), proc, what)
	}
}

// endedBySIGTERM reports whether p, waited for, was ended by SIGTERM.
func (p *directorTestProcess) endedBySIGTERM() bool {
	p.wait()
	ws, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
	return ok && ws.Signaled() && ws.Signal() == syscall.SIGTERM
}

// checkInDirector records the check-in of reg's launch script in the ledger
// stateDir with pid and startTime, as `ralph org director launched` does.
func checkInDirector(t *testing.T, stateDir string, reg org.DirectorRegistration, pid int, startTime string) {
	t.Helper()
	if err := org.NewDirectorStore(stateDir).MarkCmdLaunched(org.DirectorCmdCheckIn{
		ID: reg.ID, Token: reg.Token, PID: pid, StartTime: startTime, Who: org.DirectorWho{User: "alice", Cwd: "/work"},
	}); err != nil {
		t.Fatalf("check in %s with pid %d: %v", reg.ID, pid, err)
	}
}

// setOrgDirectorStopWait makes director stop wait d for a process after its
// SIGTERM, until the test ends.
func setOrgDirectorStopWait(t *testing.T, d time.Duration) {
	t.Helper()
	old := orgDirectorStopWait
	t.Cleanup(func() { orgDirectorStopWait = old })
	orgDirectorStopWait = d
}

// unsetDirectorTokenEnv unsets RALPH_DIRECTOR_TOKEN for the test, so a run
// inside a director's terminal does not change what stop does.
func unsetDirectorTokenEnv(t *testing.T) {
	t.Helper()
	setOrUnsetEnv(t, "RALPH_DIRECTOR_TOKEN", "", true)
}

// AC6 and AC8 end to end: a director started with launch_cmd (its launch
// script and the fake agent it execs run for real) shows launch cmd, its
// terminal name, and its pid on the director: line of `ralph status` and in
// its JSON; stop then sends SIGTERM to that pid, sees it end, records
// detached and stopped, says that the terminal is left for the human, and
// calls no herdr.
func TestOrgDirectorStop_LaunchCmdEndsTheProcessAndLeavesTheTerminal(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	unsetDirectorTokenEnv(t)
	fakes := setDirectorLaunchSeams(t)
	killDirectorLaunchedProcesses(t, stateDir)
	configPath := writeDirectorConfig(t, directorLaunchCmdToml)
	runInboxOK(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", configPath)
	waitForDirectorAgentMarker(t, fakes.agentMarker)
	events := directorEventsIn(t, stateDir)
	att, launched := events[0], launchedOf(t, stateDir)[0]
	name := "director_" + att.Token
	by := ""
	if user := os.Getenv("USER"); user != "" {
		by = ", by " + user
	}

	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	wantLine := fmt.Sprintf("director: attached d1 (driver=claude, headless running, model=opus, launch cmd, name %s, pid %d, attached at %s%s)"+directorNotCheckedTail,
		name, launched.PID, att.TS, by)
	if got := statusDirectorLine(t, out); got != wantLine {
		t.Errorf("director line:\n got %q\nwant %q", got, wantLine)
	}
	jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	if reg := statusDirectorJSONOf(t, jsonOut).Registration; reg == nil || reg.Launch != org.DirectorLaunchCmd || reg.State != org.DirectorStateRunning ||
		reg.TerminalName != name || reg.PID != launched.PID || reg.StartTime != launched.StartTime || reg.Model != "opus" {
		t.Errorf("status --json registration = %+v; want launch cmd, running, %s, pid %d, opus", reg, name, launched.PID)
	}

	stdout, stderr, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director stop: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	want := "stopped d1 (process=ended after SIGTERM)\nthe director is no longer attached\n" +
		"the terminal that [org.director].launch_cmd opened (name " + name + ") is left open for you to close\n"
	if stdout != want || stderr != "" {
		t.Errorf("stop stdout =\n%s\nwant\n%s\nstderr = %q, want nothing", stdout, want, stderr)
	}
	if got, want := directorEventKinds(directorEventsIn(t, stateDir)), []string{"attached d1", "launched d1", "detached d1", "stopped d1"}; !slices.Equal(got, want) {
		t.Errorf("director.jsonl = %v; want %v", got, want)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if proc, err := org.ReadDirectorProcess(ctx, launched.PID); err != nil || (proc.State == org.DirectorProcessAlive && proc.StartTime == launched.StartTime) {
		t.Errorf("pid %d reads %+v, %v after the stop; want the director's process ended", launched.PID, proc, err)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none with launch_cmd", lines)
	}
}

// AC6: a launch cmd's registration still starting has no process recorded,
// so stop sends nothing, records it stopped, and says that the terminal
// launch_cmd may have opened is left for the human and that its check-in is
// refused from now on, which it is.
func TestOrgDirectorStop_ALaunchCmdStillStartingIsRecordedStopped(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	unsetDirectorTokenEnv(t)
	reg := seedCmdHeadless(t, stateDir)
	stdout, stderr := runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
	want := "stopped d1 (process=none recorded)\nthe director is no longer attached\n" +
		"the terminal that [org.director].launch_cmd may have opened (name director_" + reg.Token + ") is left open for you to close, " +
		"and a launch script that checks in from it now is refused and ends without starting the agent\n"
	if stdout != want || stderr != "" {
		t.Errorf("stop stdout =\n%s\nwant\n%s\nstderr = %q, want nothing", stdout, want, stderr)
	}
	err := org.NewDirectorStore(stateDir).MarkCmdLaunched(org.DirectorCmdCheckIn{ID: reg.ID, Token: reg.Token, PID: 4242, StartTime: directorTestStartTime})
	if !errors.Is(err, org.ErrDirectorCheckInRefused) {
		t.Errorf("a check-in after the stop = %v; want it refused", err)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
}

// AC7: a process that ignores its SIGTERM outlives the wait (shortened to
// 300ms), so stop without --force records nothing and exits 1: stderr says
// what was done to the process, and the error that the pid may still be
// running, the ps that shows it, and the two ways on, with --state-dir.
// --force records the stop and warns that the pid may still be running.
// Neither sends SIGKILL: the process still runs after both.
func TestOrgDirectorStop_ALaunchCmdProcessThatOutlivesItsSIGTERM(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	unsetDirectorTokenEnv(t)
	setOrgDirectorStopWait(t, 300*time.Millisecond)
	reg := seedCmdHeadless(t, stateDir)
	p := startDirectorTestProcess(t, true)
	checkInDirector(t, stateDir, reg, p.pid(), p.startTime)
	pid, suffix, name := p.pid(), " --state-dir '"+stateDir+"'", "director_"+reg.Token
	before := directorFileOrAbsent(t, stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
	want := fmt.Sprintf("org: director: stop d1: the end of the process the launch script checked in with is not confirmed: pid %d did not end within 300ms of its SIGTERM; "+
		"nothing was recorded; run ralph org director stop --id d1%s again once pid %d has ended; pid %d may still be running: look with ps -o pid,lstart,command -p %d; "+
		"or record the stop and leave the process as it is with ralph org director stop --id d1 --force%s", pid, suffix, pid, pid, pid, suffix)
	if err == nil || err.Error() != want {
		t.Fatalf("director stop err =\n%v\nwant\n%s", err, want)
	}
	if stdout != "" || stderr != "stop d1: process=still running after SIGTERM\n" {
		t.Errorf("stdout = %q, stderr = %q; want nothing and the process note", stdout, stderr)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused stop changed director.jsonl from %q to %q", before, after)
	}
	p.wantRunning(t, "a stop that was not forced sent more than its SIGTERM")

	stdout, stderr, err = runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--force", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director stop --force: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	want = "stopped d1 (forced; process=still running after SIGTERM)\nthe director is no longer attached\n" +
		"the terminal that [org.director].launch_cmd opened (name " + name + ") is left open for you to close\n"
	if stdout != want {
		t.Errorf("stop --force stdout =\n%s\nwant\n%s", stdout, want)
	}
	wantWarning := fmt.Sprintf("warning: director stop --force: d1 is recorded stopped, but pid %d, which its launch script checked in with, may still be running "+
		"(pid %d did not end within 300ms of its SIGTERM); look with ps -o pid,lstart,command -p %d\n", pid, pid, pid)
	if stderr != wantWarning {
		t.Errorf("stop --force stderr =\n%s\nwant\n%s", stderr, wantWarning)
	}
	events := directorEventsIn(t, stateDir)
	if got, want := directorEventKinds(events), []string{"attached d1", "launched d1", "detached d1", "stopped d1"}; !slices.Equal(got, want) || !events[3].Force {
		t.Errorf("director.jsonl = %v (stopped %+v); want %v with the stop forced", got, events[len(events)-1], want)
	}
	p.wantRunning(t, "a forced stop sent SIGKILL")
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
}

// D8 through the CLI: with RALPH_DIRECTOR_TOKEN set to the registration's
// token, stop runs inside the director, so it is refused with and without
// --force, writes nothing, and sends nothing: the process still runs. With
// another launch's token in the variable, stop ends it.
func TestOrgDirectorStop_ALaunchCmdIsRefusedInsideTheDirector(t *testing.T) {
	stateDir, _, _ := directorStartLedger(t)
	reg := seedCmdHeadless(t, stateDir)
	p := startDirectorTestProcess(t, false)
	checkInDirector(t, stateDir, reg, p.pid(), p.startTime)
	before := directorFileOrAbsent(t, stateDir)
	t.Setenv("RALPH_DIRECTOR_TOKEN", reg.Token)
	want := "org: director: stop d1: RALPH_DIRECTOR_TOKEN is its launch token, so this runs inside the director, whose launch script exported it, " +
		"and stopping the director's process would end this command before it records the stop: run ralph org director stop from a terminal other than the director's"
	for _, args := range [][]string{{"--id", "d1"}, {"--id", "d1", "--force"}} {
		stdout, stderr, err := runOrgCmdStreams(t, append(append([]string{"director", "stop"}, args...), "--state-dir", stateDir)...)
		if !errors.Is(err, org.ErrDirectorStopSelfProcess) || err.Error() != want {
			t.Fatalf("stop %v inside the director =\n%v\nwant\n%s", args, err, want)
		}
		if stdout != "" || stderr != "" {
			t.Errorf("stop %v: stdout = %q, stderr = %q; want nothing", args, stdout, stderr)
		}
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused stop changed director.jsonl from %q to %q", before, after)
	}
	p.wantRunning(t, "a refused stop sent a signal")

	t.Setenv("RALPH_DIRECTOR_TOKEN", "ffffffffffff")
	if stdout, _ := runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir); !strings.HasPrefix(stdout, "stopped d1 (process=ended after SIGTERM)\n") {
		t.Errorf("stop with another launch's token: stdout = %q; want d1's process stopped", stdout)
	}
	if !p.endedBySIGTERM() {
		t.Errorf("the process ended with %v; want SIGTERM", p.cmd.ProcessState)
	}
}

// stopError and printOrgDirectorStop on a launch cmd's results that the
// tests above do not reach through a real process: a check-in after stop
// read the registration names no --force and no ps (it sent that process
// nothing), a forced stop whose record failed names the ps (unless ps could
// not be read) but not --force again (its stop command carries --force
// already), a stop or a forced stop whose ps could not be read names no ps
// to look with, and a stop that start's cleanup recorded first says so and
// names the terminal.
func TestOrgDirectorStop_LaunchCmdErrorsAndAlreadyStopped(t *testing.T) {
	rt := &orgDirectorRuntime{hintStateDir: "/work/the ledger"}
	reg := org.DirectorRegistration{ID: "d1", Driver: "claude", Launch: org.DirectorLaunchCmd, State: org.DirectorStateStarting, Token: "0123456789ab"}

	var w strings.Builder
	checkedIn := &org.DirectorCleanupError{
		Verb: "stop", ID: "d1", Launch: org.DirectorLaunchCmd, TerminalName: "director_0123456789ab",
		CleanupErr: fmt.Errorf("record d1 stopped: %w: d1 had no pid when stop read it", org.ErrDirectorCheckedInDuringStop), StopCommand: "ralph org director stop --id d1",
	}
	err := rt.stopError(&w, "d1", false, org.DirectorStopResult{Registration: reg, ProcessNote: "process=none recorded"}, checkedIn)
	want := "org: director: stop d1: record d1 stopped: the launch script checked in after stop read the registration: d1 had no pid when stop read it; " +
		"nothing was recorded; run ralph org director stop --id d1 --state-dir '/work/the ledger' again to stop the process that checked in"
	if err == nil || err.Error() != want || !errors.Is(err, org.ErrDirectorCheckedInDuringStop) {
		t.Errorf("stopError =\n%v\nwant\n%s", err, want)
	}
	if w.String() != "stop d1: process=none recorded\n" {
		t.Errorf("stopError wrote %q; want the process note", w.String())
	}

	running := reg
	running.State, running.PID = org.DirectorStateRunning, 77
	for _, tc := range []struct {
		name, note string
		notEnded   error
		wantSuffix string
	}{
		{"ps could not be read", "process=not signalled: ps could not be read",
			fmt.Errorf("%w: %w: ps is broken", org.ErrDirectorProcessNotEnded, org.ErrDirectorProcessUnreadable), "; pid 77 may still be running"},
		{"the process outlived its SIGTERM", "process=still running after SIGTERM",
			fmt.Errorf("%w: pid 77 did not end within 10s of its SIGTERM", org.ErrDirectorProcessNotEnded),
			"; pid 77 may still be running: look with ps -o pid,lstart,command -p 77"},
	} {
		w.Reset()
		forcedRecord := &org.DirectorCleanupError{
			Verb: "stop", ID: "d1", Launch: org.DirectorLaunchCmd, TerminalName: "director_0123456789ab", PID: 77,
			CleanupErr:  errors.Join(tc.notEnded, errors.New("record d1 stopped: no space left on device")),
			StopCommand: "ralph org director stop --id d1 --force",
		}
		err = rt.stopError(&w, "d1", true, org.DirectorStopResult{Registration: running, ProcessNote: tc.note}, forcedRecord)
		if err == nil || !strings.Contains(err.Error(), "ralph org director stop --id d1 --force --state-dir '/work/the ledger'") ||
			!strings.HasSuffix(err.Error(), tc.wantSuffix) || strings.Contains(err.Error(), "or record the stop and leave") {
			t.Errorf("%s: stopError of a forced stop whose record failed = %v; want it to end with %q", tc.name, err, tc.wantSuffix)
		}
		if w.String() != "stop d1: "+tc.note+"\n" {
			t.Errorf("%s: stopError wrote %q; want the process note", tc.name, w.String())
		}
	}

	// A stop that recorded nothing because ps could not be read names the
	// stop to run again once ps can read the pid and stop --force, and no ps
	// to look with.
	w.Reset()
	unreadable := &org.DirectorCleanupError{
		Verb: "stop", ID: "d1", Launch: org.DirectorLaunchCmd, TerminalName: "director_0123456789ab", PID: 77,
		CleanupErr:  fmt.Errorf("%w: %w: ps is broken", org.ErrDirectorProcessNotEnded, org.ErrDirectorProcessUnreadable),
		StopCommand: "ralph org director stop --id d1",
	}
	err = rt.stopError(&w, "d1", false, org.DirectorStopResult{Registration: running, ProcessNote: "process=not signalled: ps could not be read"}, unreadable)
	want = "org: director: stop d1: the end of the process the launch script checked in with is not confirmed: ps could not be read: ps is broken; " +
		"nothing was recorded; run ralph org director stop --id d1 --state-dir '/work/the ledger' again once ps can read pid 77; pid 77 may still be running; " +
		"or record the stop and leave the process as it is with ralph org director stop --id d1 --force --state-dir '/work/the ledger'"
	if err == nil || err.Error() != want {
		t.Errorf("stopError of a stop whose ps could not be read =\n%v\nwant\n%s", err, want)
	}

	// A forced stop whose ps could not be read warns that the pid may still
	// be running, with why and no ps to look with.
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	printOrgDirectorStop(cmd, org.DirectorStopResult{Registration: running, Forced: true, WasCurrent: true, ProcessNote: "process=not signalled: ps could not be read",
		TerminalName: "director_0123456789ab", CloseErr: fmt.Errorf("%w: ps is broken", org.ErrDirectorProcessUnreadable)})
	wantErr := "warning: director stop --force: d1 is recorded stopped, but pid 77, which its launch script checked in with, may still be running " +
		"(ps could not be read: ps is broken)\n"
	if stderr.String() != wantErr {
		t.Errorf("printOrgDirectorStop of a forced stop whose ps could not be read: stderr =\n%s\nwant\n%s", stderr.String(), wantErr)
	}

	stdout.Reset()
	stderr.Reset()
	printOrgDirectorStop(cmd, org.DirectorStopResult{Registration: running, AlreadyStopped: true, ProcessNote: "process=ended after SIGTERM", TerminalName: "director_0123456789ab"})
	wantOut := "d1 was already recorded stopped (process=ended after SIGTERM)\n" +
		"start's cleanup or another stop recorded d1 stopped after this stop read it, so this stop recorded nothing more\n" +
		"the terminal that [org.director].launch_cmd opened (name director_0123456789ab) is left open for you to close\n"
	if stdout.String() != wantOut || stderr.String() != "" {
		t.Errorf("printOrgDirectorStop stdout =\n%s\nwant\n%s\nstderr = %q, want nothing", stdout.String(), wantOut, stderr.String())
	}
}

// attach's refusal while a launch cmd's registration is current names the
// stop from a terminal other than the director's, which stops its process,
// and --replace, which leaves it running; it writes nothing. The pid is
// made up: nothing reads or signals it.
func TestOrgDirectorAttach_RefusedByALaunchCmdDirectorNamesItsStop(t *testing.T) {
	stateDir := t.TempDir()
	reg := seedCmdHeadless(t, stateDir)
	checkInDirector(t, stateDir, reg, 4242, directorTestStartTime)
	before := directorFileOrAbsent(t, stateDir)
	suffix := " --state-dir '" + stateDir + "'"

	stdout, _, err := runOrgCmdStreams(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	if !errors.Is(err, org.ErrDirectorAttached) {
		t.Fatalf("attach over a launch cmd director = %v; want ErrDirectorAttached", err)
	}
	if want := ", headless running, launch cmd, pid 4242): stop it with ralph org director stop --id d1" + suffix + " from a terminal other than the director's, " +
		"which stops its process, or replace it with ralph org director attach --driver claude --replace" + suffix + ", which leaves that process running until such a stop"; !strings.HasSuffix(err.Error(), want) {
		t.Errorf("attach error %q does not end with %q", err, want)
	}
	if strings.Contains(err.Error(), "herdr") || strings.Contains(err.Error(), "detach") {
		t.Errorf("attach error %q names herdr or detach; want the stop of a launch cmd", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused attach changed director.jsonl")
	}
}

// detach --id, detach --force, and attach --replace of a launch cmd's
// registration only record, so they say that its process keeps running and
// name the stop that ends it, with --state-dir; with no check-in recorded
// yet they say that a check-in is refused now (it is) and name the stop that
// records it stopped. The pid is made up: nothing reads or signals it.
func TestOrgDirectorDetach_ALaunchCmdDirectorNamesItsStop(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	for _, tc := range []struct {
		name     string
		args     []string
		pid      int
		wantHead func(reg org.DirectorRegistration) string
	}{
		{"detach --id", []string{"director", "detach", "--id", "d1"}, 4242,
			func(org.DirectorRegistration) string { return "detached d1\n" }},
		{"detach --force, no check-in yet", []string{"director", "detach", "--force"}, 0,
			func(reg org.DirectorRegistration) string {
				return "detached d1 (forced; driver=claude, attached at " + reg.AttachedAt + ")\n"
			}},
		{"attach --replace", []string{"director", "attach", "--driver", "codex", "--replace"}, 4242,
			func(reg org.DirectorRegistration) string {
				return "attached d2 (driver=codex), replacing d1 (driver=claude, attached at " + reg.AttachedAt + ")\n"
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			reg := seedCmdHeadless(t, stateDir)
			if tc.pid > 0 {
				checkInDirector(t, stateDir, reg, tc.pid, directorTestStartTime)
			}
			suffix := " --state-dir '" + stateDir + "'"
			want := tc.wantHead(reg)
			if strings.HasPrefix(want, "attached d2") {
				want += "detach it with: ralph org director detach --id d2" + suffix + "\n"
			}
			if tc.pid > 0 {
				want += "the process of the headless director d1 (pid 4242) keeps running; stop it with: ralph org director stop --id d1" + suffix + "\n"
			} else {
				want += "the headless director d1 has no pid yet: a check-in of its launch script is refused now, so it starts no agent; " +
					"record its stop with: ralph org director stop --id d1" + suffix + "\n"
			}

			stdout, _ := runInboxOK(t, append(tc.args, "--state-dir", stateDir)...)
			if stdout != want {
				t.Errorf("%v stdout =\n%s\nwant\n%s", tc.args, stdout, want)
			}
			if tc.pid == 0 {
				err := org.NewDirectorStore(stateDir).MarkCmdLaunched(org.DirectorCmdCheckIn{ID: reg.ID, Token: reg.Token, PID: 4242, StartTime: directorTestStartTime})
				if !errors.Is(err, org.ErrDirectorCheckInRefused) {
					t.Errorf("a check-in after the detach = %v; want it refused", err)
				}
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
		})
	}
}

// AC8 for director status: a launch cmd's registration shows launch cmd,
// its terminal name, and its pid (once checked in) on the director: line,
// pid, start_time, and terminal_name in the text form, and the same in the
// JSON; an open one that is not current is named so at the end of the line
// and on its open_headless line. The pid is made up: nothing reads or
// signals it. No watch has checked d1, so its line says not checked yet
// (plan 2026-10-18-org-director-liveness, D8).
func TestOrgDirectorStatus_ShowsLaunchCmdDirectors(t *testing.T) {
	const tail = directorNotCheckedTail
	stateDir := t.TempDir()
	d1 := seedCmdHeadless(t, stateDir)
	name, ts := "director_"+d1.Token, d1.AttachedAt

	if got, want := directorStatusLine(t, "--state-dir", stateDir),
		"director: attached d1 (driver=claude, headless starting, model=opus, launch cmd, name "+name+", attached at "+ts+", by alice)"+tail; got != want {
		t.Errorf("director line of a starting d1:\n got %q\nwant %q", got, want)
	}
	checkInDirector(t, stateDir, d1, 4242, directorTestStartTime)
	stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
	want := "director: attached d1 (driver=claude, headless running, model=opus, launch cmd, name " + name + ", pid 4242, attached at " + ts + ", by alice)" + tail + "\n" +
		"enabled: true\nauto_merge: true\nid: d1\ndriver: claude\nwake: pull\nattached_at: " + ts + "\nuser: alice\ncwd: /work\nherdr_pane: no\n" +
		"launch: cmd\nstate: running\nmodel: opus\npermission_mode: autonomous\ntoken: " + d1.Token + "\npid: 4242\nstart_time: " + directorTestStartTime +
		"\nterminal_name: " + name + "\n" + directorNoWaitMarkLine + "\n"
	if stdout != want {
		t.Errorf("status of a running d1 =\n%s\nwant\n%s", stdout, want)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if reg := directorReportOf(t, jsonOut).Registration; reg == nil || reg.Launch != org.DirectorLaunchCmd || reg.State != org.DirectorStateRunning ||
		reg.TerminalName != name || reg.PID != 4242 || reg.StartTime != directorTestStartTime || reg.Model != "opus" || reg.WorkspaceID != "" || reg.PaneID != "" {
		t.Errorf("director status --json = %s; want d1 with launch cmd, running, %s, pid 4242", jsonOut, name)
	}
	for _, key := range []string{`"launch": "cmd"`, `"terminal_name": "` + name + `"`, `"pid": 4242`, `"start_time": "` + directorTestStartTime + `"`} {
		if !strings.Contains(jsonOut, key) {
			t.Errorf("director status --json does not carry %s:\n%s", key, jsonOut)
		}
	}

	d2, _, err := org.NewDirectorStore(stateDir).Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, Replace: true, User: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	stdout, _ = runInboxOK(t, "director", "status", "--state-dir", stateDir)
	open := "d1 (running, launch cmd, name " + name + ", pid 4242)"
	wantLine := directorAttachedLine("d2", "claude", d2.AttachedAt, "bob") + "; the headless director " + open +
		" is still open but not attached: stop it with ralph org director stop --id d1 --state-dir '" + stateDir + "'"
	if lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); lines[0] != wantLine || lines[len(lines)-1] != "open_headless: "+open ||
		strings.Contains(stdout, "terminal_name:") {
		t.Errorf("status with d2 promoted over d1 =\n%s\nwant the line %q, d2's 8a-1 fields, and open_headless: %s", stdout, wantLine, open)
	}
	jsonOut, _ = runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	report := directorReportOf(t, jsonOut)
	if report.Registration == nil || report.Registration.ID != "d2" || report.Registration.TerminalName != "" || len(report.OpenHeadless) != 1 ||
		report.OpenHeadless[0].ID != "d1" || report.OpenHeadless[0].TerminalName != name || report.OpenHeadless[0].PID != 4242 {
		t.Errorf("director status --json = %s; want d2 current with no terminal_name and d1 in open_headless with %s", jsonOut, name)
	}
}

// AC8's last sentence: terminal_name is added for a launch cmd only, so the
// JSON of a herdr or a promoted registration in status is the
// org.DirectorRegistration's, byte for byte, and a launch cmd's is that
// with terminal_name added.
func TestOrgDirectorReportRegistration_AddsTheTerminalNameForALaunchCmdOnly(t *testing.T) {
	for _, reg := range []org.DirectorRegistration{
		{ID: "d1", Driver: "codex", Wake: []string{"pull"}, Launch: org.DirectorLaunchHerdr, State: org.DirectorStateRunning, Token: "0123456789ab",
			Model: "gpt-6-astra", PermissionMode: "guarded", WorkspaceID: "w-1", PaneID: "p-1", AttachedAt: "2026-10-16T09:00:00Z", User: "alice", Cwd: "/work"},
		{ID: "d2", Driver: "claude", Wake: []string{"pull"}, PaneID: "%7", AttachedAt: "2026-10-16T09:00:00Z", User: "bob", Cwd: "/w", HerdrPane: true},
	} {
		got, err := json.Marshal(newOrgDirectorReportRegistration(reg))
		if err != nil {
			t.Fatal(err)
		}
		want, err := json.Marshal(reg)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("the status JSON of %s =\n%s\nwant the registration's\n%s", reg.ID, got, want)
		}
	}
	cmdReg := org.DirectorRegistration{ID: "d3", Driver: "claude", Wake: []string{"pull"}, Launch: org.DirectorLaunchCmd, State: org.DirectorStateRunning,
		Token: "0123456789ab", Model: "opus", PermissionMode: "autonomous", PID: 4242, StartTime: directorTestStartTime, AttachedAt: "2026-10-16T09:00:00Z"}
	got, err := json.Marshal(newOrgDirectorReportRegistration(cmdReg))
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(cmdReg)
	if err != nil {
		t.Fatal(err)
	}
	if wantStr := strings.TrimSuffix(string(want), "}") + `,"terminal_name":"director_0123456789ab"}`; string(got) != wantStr {
		t.Errorf("the status JSON of a launch cmd =\n%s\nwant\n%s", got, wantStr)
	}
}
