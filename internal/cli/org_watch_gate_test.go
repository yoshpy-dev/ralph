package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for the watch gate on the CLI (plan 2026-10-12-org-watch-gate, AC1,
// AC2, AC7 and AC8 on the CLI side). Each test sets TestMain's gate that
// always passes back to nil (useRealOrgWatchGate), so spawn, start, and send
// go through org.NewWatchGate on a ledger with a real watch.lock and
// liveness mark; holdWatchLock and runningMark come from org_watch_test.go,
// writeWatchMark from status_test.go. A refusal is the command's error, which
// main prints on stderr before it exits 1. None of these tests runs in
// parallel: they swap package-level vars.

// useRealOrgWatchGate sets orgWatchGateOverride to nil for the rest of the
// test, so wireOrgWatchGate builds the real gate, and restores TestMain's
// gate at cleanup.
func useRealOrgWatchGate(t *testing.T) {
	t.Helper()
	orig := orgWatchGateOverride
	orgWatchGateOverride = nil
	t.Cleanup(func() { orgWatchGateOverride = orig })
}

// stoppingWatchMark is the liveness mark of a watch with pid that has run its
// last cycle and drains, written at ts.
func stoppingWatchMark(pid int, ts time.Time) *org.WatchAlive {
	mark := runningMark(pid, ts)
	mark.State = org.WatchAliveStopping
	return mark
}

// onceWatchMark is the running mark of a --once watch with pid.
func onceWatchMark(pid int, ts time.Time) *org.WatchAlive {
	mark := runningMark(pid, ts)
	mark.Once = true
	return mark
}

// gateRefusal is a watch the gate refuses: how to set it up on a ledger, and
// what the refusal says.
type gateRefusal struct {
	name  string
	setup func(t *testing.T, stateDir string)
	want  []string
}

// gateRefusals are the watches that accept no work: none, a killed watch
// that left its running mark, a stale watch with an old mark and with none,
// a stopping watch, and a --once watch.
func gateRefusals() []gateRefusal {
	return []gateRefusal{
		{
			name:  "no watch",
			setup: func(*testing.T, string) {},
			want:  []string{"no ralph org watch --all is running", "start one with ralph org watch --all --detach"},
		},
		{
			name:  "the running mark of a killed watch",
			setup: func(t *testing.T, stateDir string) { writeWatchMark(t, stateDir, runningMark(4301, time.Now())) },
			want:  []string{"no ralph org watch --all is running", "start one with ralph org watch --all --detach"},
		},
		{
			name: "stale with an old mark",
			setup: func(t *testing.T, stateDir string) {
				holdWatchLock(t, stateDir, runningMark(4302, time.Now().Add(-time.Hour)))
			},
			want: []string{"has not refreshed its liveness mark (pid 4302)", "stop it with ralph org watch --stop",
				"kill 4302 (the pid in " + org.WatchAlivePathIn(stateDirPlaceholder) + "; check that it is still a ralph org watch first)"},
		},
		{
			name:  "stale with no mark",
			setup: func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, nil) },
			want: []string{"has not refreshed its liveness mark (pid unknown)", "stop it with ralph org watch --stop",
				"find the process that holds " + org.WatchLockPathIn(stateDirPlaceholder) + " (" + org.WatchAlivePathIn(stateDirPlaceholder) + " names no pid) and kill it"},
		},
		{
			name:  "stopping",
			setup: func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, stoppingWatchMark(4303, time.Now())) },
			want:  []string{"(pid 4303) is stopping", "run this again once it has ended"},
		},
		{
			name:  "running --once",
			setup: func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, onceWatchMark(4304, time.Now())) },
			want:  []string{"(pid 4304) is a --once run", "run this again once it has ended"},
		},
	}
}

// stateDirPlaceholder stands for the ledger in a string a gateRefusal wants,
// which checkGateRefusal replaces with the state dir of the test.
const stateDirPlaceholder = "{state dir}"

// checkGateRefusal checks that err is a refusal of the gate on stateDir that
// says every string of want.
func checkGateRefusal(t *testing.T, err error, stateDir string, want ...string) {
	t.Helper()
	if !errors.Is(err, org.ErrNoRunningWatch) {
		t.Fatalf("error = %v; want a refusal of the watch gate (org.ErrNoRunningWatch)", err)
	}
	for _, w := range append([]string{"org state dir " + stateDir}, want...) {
		w = strings.ReplaceAll(w, stateDirPlaceholder, stateDir)
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error %q does not say %q", err, w)
		}
	}
}

// checkNothingStarted checks that the ledger at stateDir has no manifest
// event and no receipt, and that the herdr and agmsg stubs were not called.
func checkNothingStarted(t *testing.T, stateDir, herdrLog, agmsgLog string) {
	t.Helper()
	if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
		t.Errorf("manifest events = %v; want none (not even rejected)", eventTypes(events))
	}
	if _, err := os.Stat(org.ReceiptsPathIn(stateDir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stat %s = %v; want no receipt file", org.ReceiptsPathIn(stateDir), err)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
	if lines := readLogLines(t, agmsgLog); len(lines) != 0 {
		t.Errorf("agmsg calls = %v; want none", lines)
	}
}

// gateSpawnArgs is `ralph org spawn` of org-a/seat-1 into the ledger at
// stateDir, with extra flags after it.
func gateSpawnArgs(t *testing.T, stateDir string, extra ...string) []string {
	t.Helper()
	return append([]string{
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(), "--scope", "test-scope",
		"--state-dir", stateDir,
	}, extra...)
}

// AC1: with no watch that accepts work, spawn, real or --dry-run, is refused
// with the next step for the watch's state, prints only the rejection, and
// writes no manifest event, no receipt, and calls neither herdr nor agmsg.
func TestOrgSpawn_WatchGate_RefusesAWatchThatAcceptsNoWork(t *testing.T) {
	for _, tc := range gateRefusals() {
		for _, dryRun := range []bool{false, true} {
			name := tc.name
			var extra []string
			if dryRun {
				name += " --dry-run"
				extra = []string{"--dry-run"}
			}
			t.Run(name, func(t *testing.T) {
				useRealOrgWatchGate(t)
				herdrLog, agmsgLog := setupOrgStubPATH(t)
				stateDir := t.TempDir()
				tc.setup(t, stateDir)

				stdout, _, err := runOrgCmdStreams(t, gateSpawnArgs(t, stateDir, extra...)...)
				checkGateRefusal(t, err, stateDir, tc.want...)
				if want := "rejected: " + err.Error() + "\n"; stdout != want {
					t.Errorf("stdout = %q; want only %q", stdout, want)
				}
				checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
			})
		}
	}
}

// AC1: a running watch lets spawn go on as before, real and --dry-run.
func TestOrgSpawn_WatchGate_ARunningWatchPasses(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		name := "real"
		var extra []string
		if dryRun {
			name = "--dry-run"
			extra = []string{"--dry-run"}
		}
		t.Run(name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			setupOrgStubPATH(t)
			stateDir := t.TempDir()
			holdWatchLock(t, stateDir, runningMark(4310, time.Now()))

			stdout, stderr, err := runOrgCmdStreams(t, gateSpawnArgs(t, stateDir, extra...)...)
			if err != nil {
				t.Fatalf("spawn with a running watch: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if want := `spawned seat "seat-1" (org_id=org-a `; !strings.HasPrefix(stdout, want) {
				t.Errorf("stdout = %q; want it to start %q", stdout, want)
			}
			ev := lastSeatEvent(t, stateDir, "org-a", "seat-1")
			if ev.Event != org.EventSpawned || ev.DryRun != dryRun {
				t.Errorf("last event = %s (dry_run=%t); want %s (dry_run=%t)", ev.Event, ev.DryRun, org.EventSpawned, dryRun)
			}
		})
	}
}

// Plan F10 and AC15 on the CLI: a watch that holds watch.lock and writes its
// first running mark within the settle is waited for, and spawn goes on.
func TestOrgSpawn_WatchGate_AStaleWatchThatTurnsRunningWithinTheSettlePasses(t *testing.T) {
	useRealOrgWatchGate(t)
	setOrgWatchDuration(t, &orgWatchGateSettleOverride, 5*time.Second)
	setupOrgStubPATH(t)
	stateDir := t.TempDir()
	holdWatchLock(t, stateDir, nil)
	written := make(chan error, 1)
	go func() {
		time.Sleep(200 * time.Millisecond)
		data, err := json.Marshal(runningMark(4311, time.Now()))
		if err == nil {
			tmp := org.WatchAlivePathIn(stateDir) + ".tmp"
			if err = os.WriteFile(tmp, data, 0o644); err == nil {
				err = os.Rename(tmp, org.WatchAlivePathIn(stateDir))
			}
		}
		written <- err
	}()

	start := time.Now()
	stdout, stderr, err := runOrgCmdStreams(t, gateSpawnArgs(t, stateDir, "--dry-run")...)
	elapsed := time.Since(start)
	if werr := <-written; werr != nil {
		t.Fatal(werr)
	}
	if err != nil {
		t.Fatalf("spawn --dry-run while the watch writes its first mark: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if elapsed < 150*time.Millisecond || elapsed >= 5*time.Second {
		t.Errorf("spawn returned after %v; want it to wait for the mark written after 200ms, and not the whole settle of 5s", elapsed)
	}
}

// AC7 at the gate: a mark that carries the watch's own deadman_minutes is
// judged by it, and a mark without one by the caller's config. Each mark is
// 20 minutes old and holds watch.lock.
func TestOrgSpawn_WatchGate_JudgesTheMarkByTheWatchsDeadmanMinutes(t *testing.T) {
	ten, thirty := 10, 30
	cases := []struct {
		name     string
		callerDM string
		markDM   *int
		wantPass bool
	}{
		{name: "caller 10, watch 30", callerDM: "10", markDM: &thirty, wantPass: true},
		{name: "caller 10, watch without deadman_minutes", callerDM: "10"},
		{name: "caller 30, watch without deadman_minutes", callerDM: "30", wantPass: true},
		{name: "caller 30, watch 10", callerDM: "30", markDM: &ten},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			stateDir := t.TempDir()
			configPath := filepath.Join(t.TempDir(), "ralph.toml")
			writeTestFile(t, configPath, "[org]\ndeadman_minutes = "+tc.callerDM+"\n")
			mark := runningMark(4312, time.Now().Add(-20*time.Minute))
			mark.DeadmanMinutes = tc.markDM
			holdWatchLock(t, stateDir, mark)

			_, _, err := runOrgCmdStreams(t, gateSpawnArgs(t, stateDir, "--dry-run", "--config", configPath)...)
			if tc.wantPass {
				if err != nil {
					t.Fatalf("spawn --dry-run: %v; want the 20-minute-old mark read as running", err)
				}
				return
			}
			checkGateRefusal(t, err, stateDir, "has not refreshed its liveness mark (pid 4312)")
			checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
		})
	}
}

// AC2: with no watch, send refuses every TASK, typed or --raw whose first
// line reads as TYPE: TASK, real or --dry-run, before it records or calls
// anything. Another TYPE, and --raw text whose first line is no TYPE header,
// are sent as before.
func TestOrgSend_WatchGate_RefusesOnlyATask(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantSent bool
	}{
		{name: "typed TASK", args: []string{"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"}},
		{name: "typed TASK --dry-run", args: []string{"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--dry-run"}},
		{name: "--raw TYPE: TASK", args: []string{"--raw", "--text", "TYPE:  TASK \nfree text"}},
		{name: "--raw TYPE: TASK --dry-run", args: []string{"--raw", "--text", "TYPE: TASK\nfree text", "--dry-run"}},
		{name: "QUESTION --dry-run", args: []string{"--text", "TYPE: QUESTION\n\nwhich plan?", "--dry-run"}, wantSent: true},
		{name: "--raw without a TYPE line --dry-run", args: []string{"--raw", "--text", "please do the thing", "--dry-run"}, wantSent: true},
		{name: "--raw lower-case type: task --dry-run", args: []string{"--raw", "--text", "type: task\nfree text", "--dry-run"}, wantSent: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			stateDir := t.TempDir()
			seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
			seeded := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))

			args := append([]string{"send", "--org-id", "org-a", "--to", "seat-1", "--enter-delay-ms", "1", "--state-dir", stateDir}, tc.args...)
			stdout, stderr, err := runOrgCmdStreams(t, args...)
			events := readManifestEvents(t, org.ManifestPathIn(stateDir))
			if tc.wantSent {
				if err != nil {
					t.Fatalf("send: %v (stdout: %s stderr: %s); want it sent with no watch", err, stdout, stderr)
				}
				if want := "sent message to seat \"seat-1\"\n"; stdout != want {
					t.Errorf("stdout = %q; want %q", stdout, want)
				}
				if len(events) != seeded+1 || events[seeded].Event != org.EventSent || !events[seeded].DryRun {
					t.Errorf("manifest events after the seed = %v; want one dry-run sent", eventTypes(events[seeded:]))
				}
				return
			}
			checkGateRefusal(t, err, stateDir, "org: send: TASK not sent: no ralph org watch --all is running", "start one with ralph org watch --all --detach")
			if !strings.HasPrefix(err.Error(), "org: send: TASK not sent: ") || strings.Count(err.Error(), "org:") != 1 {
				t.Errorf("error %q; want it to start with \"org: send: TASK not sent: \" and say \"org:\" only there", err)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if len(events) != seeded {
				t.Errorf("manifest events after the seed = %v; want none", eventTypes(events[seeded:]))
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
			if lines := readLogLines(t, agmsgLog); len(lines) != 0 {
				t.Errorf("agmsg calls = %v; want none", lines)
			}
		})
	}
}

// A send error that does not start with "org: send: " in the org layer (the
// dry-run event's append, here to a manifest path that is a directory) gets
// that prefix from the CLI, once, and keeps its cause.
func TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := t.TempDir()
	if err := os.Mkdir(org.ManifestPathIn(stateDir), 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := runOrgCmdStreams(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: QUESTION\n\nwhich plan?", "--dry-run", "--state-dir", stateDir)
	want := "org: send: org: open " + org.ManifestPathIn(stateDir) + ": "
	if err == nil || !strings.HasPrefix(err.Error(), want) || strings.Count(err.Error(), "org: send:") != 1 {
		t.Fatalf("dry-run send with a manifest that cannot be opened = %v; want it to start with %q", err, want)
	}
	if !errors.Is(err, syscall.EISDIR) {
		t.Errorf("error %v does not wrap EISDIR; want the append's cause kept", err)
	}
}

// AC2: a running watch lets a TASK go out as before.
func TestOrgSend_WatchGate_ATaskPassesARunningWatch(t *testing.T) {
	useRealOrgWatchGate(t)
	setupOrgStubPATH(t)
	stateDir := t.TempDir()
	holdWatchLock(t, stateDir, runningMark(4320, time.Now()))

	stdout, stderr, err := runOrgCmdStreams(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--dry-run", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("send a TASK with a running watch: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if ev := lastSeatEvent(t, stateDir, "org-a", "seat-1"); ev.Event != org.EventSent || !ev.DryRun {
		t.Errorf("last event = %s (dry_run=%t); want a dry-run sent", ev.Event, ev.DryRun)
	}
}

// shortenOrgWatchDeadline makes the wait of ensureOrgWatch (start and
// --detach) for a stale watch, and for a launched one, last 300ms, probing
// every 10ms.
func shortenOrgWatchDeadline(t *testing.T) {
	t.Helper()
	setOrgWatchDuration(t, &orgWatchEnsureDeadline, 300*time.Millisecond)
	setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
}

// AC1 and AC4 for `ralph org start <task>`: the leader's spawn is refused
// while the watch that holds watch.lock is stale for the whole deadline
// (shortened), no watch is launched, and nothing is recorded or called.
func TestOrgStart_WatchGate_RefusesAStaleWatch(t *testing.T) {
	useRealOrgWatchGate(t)
	refuseOrgWatchLaunch(t)
	shortenOrgWatchDeadline(t)
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := t.TempDir()
	holdWatchLock(t, stateDir, runningMark(4330, time.Now().Add(-time.Hour)))

	start := time.Now()
	stdout, _, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide", "--state-dir", stateDir, "do the task")
	if elapsed := time.Since(start); elapsed < 300*time.Millisecond {
		t.Errorf("start refused the stale watch after %v; want it to wait the 300ms deadline first", elapsed)
	}
	checkGateRefusal(t, err, stateDir, "has not refreshed its liveness mark (pid 4330)", "stop it with ralph org watch --stop")
	if want := "rejected: " + err.Error() + "\n"; stdout != want {
		t.Errorf("stdout = %q; want only %q", stdout, want)
	}
	checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
}

// AC1 and AC4 for `ralph org start --plan`: with a watch stale for the whole
// deadline (shortened) the start is refused before the feature worktree is
// made, so there is no launched watch, no worktree, no branch, no manifest
// event, and no herdr or agmsg call.
func TestOrgStartPlan_WatchGate_RefusesAStaleWatchBeforeTheWorktree(t *testing.T) {
	useRealOrgWatchGate(t)
	refuseOrgWatchLaunch(t)
	shortenOrgWatchDeadline(t)
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(tmp, "state")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	holdWatchLock(t, stateDir, runningMark(4331, time.Now().Add(-time.Hour)))

	stdout, _, err := startOrgFeature(t, plan, "a", "--state-dir", stateDir)
	checkGateRefusal(t, err, stateDir, "has not refreshed its liveness mark (pid 4331)", "stop it with ralph org watch --stop")
	if want := "rejected: " + err.Error() + "\n"; stdout != want {
		t.Errorf("stdout = %q; want only %q", stdout, want)
	}
	wt := filepath.Join(root, ".claude", "worktrees", "org-a")
	if _, statErr := os.Stat(wt); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("stat %s = %v; want no worktree", wt, statErr)
	}
	out, gitErr := exec.Command("git", "-C", root, "branch", "--list", "feat/a").Output()
	if gitErr != nil {
		t.Fatalf("git branch --list: %v", gitErr)
	}
	if branches := strings.TrimSpace(string(out)); branches != "" {
		t.Errorf("branches named feat/a = %q; want none", branches)
	}
	checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
}

// The `ralph org watch` commands of a refusal repeat --state-dir when it was
// given, by orgInboxHintStateDir's rule, and leave it out when the ledger was
// resolved another way (here RALPH_ORG_STATE_DIR), for spawn and send alike.
func TestOrgWatchGate_TheCommandsRepeatAnExplicitStateDir(t *testing.T) {
	type verb struct {
		name string
		args []string
	}
	verbs := []verb{
		{name: "spawn", args: []string{"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
			"--driver", "claude", "--model", "sonnet", "--cwd", ".", "--scope", "test-scope", "--dry-run"}},
		{name: "send", args: []string{"send", "--org-id", "org-a", "--to", "seat-1", "--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo it", "--dry-run"}},
	}
	for _, v := range verbs {
		t.Run(v.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "led ger")
			quoted := "'" + stateDir + "'"

			t.Setenv(org.EnvOrgStateDir, "")
			_, _, err := runOrgCmdStreams(t, append(slices.Clone(v.args), "--state-dir", stateDir)...)
			checkGateRefusal(t, err, stateDir, "start one with ralph org watch --all --detach --state-dir "+quoted+", ")

			holdWatchLock(t, stateDir, runningMark(4340, time.Now().Add(-time.Hour)))
			_, _, err = runOrgCmdStreams(t, append(slices.Clone(v.args), "--state-dir", stateDir)...)
			checkGateRefusal(t, err, stateDir, "stop it with ralph org watch --stop --state-dir "+quoted+", ")

			t.Setenv(org.EnvOrgStateDir, stateDir)
			_, _, err = runOrgCmdStreams(t, v.args...)
			checkGateRefusal(t, err, stateDir, "stop it with ralph org watch --stop, ")
			if strings.Contains(err.Error(), "--state-dir") {
				t.Errorf("error with the ledger from %s = %q; want no --state-dir", org.EnvOrgStateDir, err)
			}
		})
	}
}

// AC8: with no watch and the real gate, the verbs that start no work behave
// as before on a seeded ledger: `ralph status`, org status, read, report,
// escalate, inbox, wait --inbox, stop, disband, stop --all, and disband
// --all. None of them starts a watch.
func TestOrgVerbsThatStartNoWork_WorkWithNoWatch(t *testing.T) {
	useRealOrgWatchGate(t)
	setInboxPollIntervalOverride(t, 20*time.Millisecond)
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := t.TempDir()
	for _, o := range []string{"org-a", "org-b", "org-c"} {
		seedWorkspace(t, stateDir, o, "ws-"+o)
		seedSeat(t, stateDir, o, "seat-1", "p-"+o+"-1")
	}
	seedSeat(t, stateDir, "org-a", "seat-2", "p-org-a-2")

	ok := func(args ...string) string {
		t.Helper()
		stdout, stderr, err := runOrgCmdStreams(t, args...)
		if err != nil {
			t.Fatalf("%v with no watch: %v (stdout: %s stderr: %s)", args, err, stdout, stderr)
		}
		return stdout
	}

	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("ralph status: %v (output: %s)", err, out)
	}
	if got := statusOrgWatchLine(t, out); got != "org watch: not running" {
		t.Errorf("org watch line = %q; want %q", got, "org watch: not running")
	}
	if out := ok("status", "--org-id", "org-a", "--state-dir", stateDir); !strings.Contains(out, "seat-2\t") {
		t.Errorf("org status of org-a does not list seat-2:\n%s", out)
	}
	if out := ok("read", "--org-id", "org-a", "--seat", "seat-1", "--lines", "10", "--state-dir", stateDir); !strings.Contains(out, "pane output") {
		t.Errorf("read printed %q; want the pane output", out)
	}
	if out := ok("report", "--org-id", "org-a", "--state-dir", stateDir, "--out", t.TempDir()); !strings.Contains(out, "wrote ") {
		t.Errorf("report printed %q; want the written path", out)
	}

	if id := escalateOK(t, stateDir, "org-a", inboxQuestion("which plan?")); id != "e1" {
		t.Errorf("escalated item = %s; want e1", id)
	}
	if rows := inboxTableRows(t, ok("inbox", "--state-dir", stateDir)); len(rows) != 1 || rows[0][0] != "e1" || rows[0][1] != "open" {
		t.Errorf("inbox rows = %q; want e1 open", rows)
	}
	stdout, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, "wait", "--inbox", "--timeout-ms", "300", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("wait --inbox with no watch: %v", err)
	}
	if want := "e1\torg-a\tQUESTION\twhich plan?\n"; stdout != want {
		t.Errorf("wait --inbox printed %q; want %q", stdout, want)
	}

	if out := ok("stop", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", stateDir); out != "stopped seat \"seat-1\"\n" {
		t.Errorf("stop printed %q; want the stopped line", out)
	}
	want := []string{`stopped seat "seat-2"`, `disbanded org "org-a"`}
	if got := outputLines(ok("disband", "--org-id", "org-a", "--state-dir", stateDir)); !slices.Equal(got, want) {
		t.Errorf("disband org-a printed %q; want %q", got, want)
	}
	want = []string{"stopped seat org-b/seat-1", "stopped seat org-c/seat-1"}
	if got := outputLines(ok("stop", "--all", "--state-dir", stateDir)); !slices.Equal(got, want) {
		t.Errorf("stop --all printed %q; want %q", got, want)
	}
	want = []string{"disbanded org org-b", "disbanded org org-c"}
	if got := outputLines(ok("disband", "--all", "--state-dir", stateDir)); !slices.Equal(got, want) {
		t.Errorf("disband --all printed %q; want %q", got, want)
	}
	if got := realDisbandedOrgs(t, stateDir); !slices.Equal(got, []string{"org-a", "org-b", "org-c"}) {
		t.Errorf("disbanded orgs = %v; want org-a, org-b, org-c", got)
	}
	if !containsLine(readLogLines(t, herdrLog), "workspace close ws-org-c") {
		t.Errorf("herdr log has no workspace close of ws-org-c: %v", readLogLines(t, herdrLog))
	}

	if probe := org.ProbeWatch(stateDir, time.Now(), 10); probe.State != org.WatchLivenessNotRunning || probe.Mark != nil {
		t.Errorf("watch after the verbs = %+v; want not running with no mark", probe)
	}
}
