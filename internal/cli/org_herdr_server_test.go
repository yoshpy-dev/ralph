package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// Tests for the herdr server gate of the CLI (plan
// 2026-11-06-org-herdr-autostart, S3: AC1, AC2, AC4, AC5, AC7, and AC8 on
// the CLI side). The gate's own tests drive ensureOrgHerdrServer through
// its seams (setOrgHerdrServerSeams), so no herdr runs and no process
// starts; the wiring tests run the verbs with orgHerdrServerGateOverride
// swapped. None of these tests runs in parallel: they swap package-level
// vars.

// herdrStatusAnswer is one answer of the fake status read.
type herdrStatusAnswer struct {
	st  driver.HerdrServerStatus
	err error
}

// herdrLaunchCall is one start of `herdr server` the gate asked for.
type herdrLaunchCall struct {
	path string
	req  orgWatchLaunchRequest
}

// herdrGateFake is the gate's seams for one test: the status reads answer
// answers in order, the last one again once they run out; the lookup of
// herdr returns lookPath and lookErr; a launch returns proc and launchErr.
// It records the reads, the time each read's context had left, the
// lookups, and the launches.
type herdrGateFake struct {
	answers   []herdrStatusAnswer
	lookPath  string
	lookErr   error
	proc      *orgWatchProcess
	launchErr error

	reads    int
	timeLeft []time.Duration
	lookups  int
	launches []herdrLaunchCall
}

// setOrgHerdrServerSeams points the gate's status read, lookup, and launch
// at f for the rest of the test, and restores them at cleanup.
func setOrgHerdrServerSeams(t *testing.T, f *herdrGateFake) {
	t.Helper()
	origStatus, origLook, origLaunch := orgHerdrServerStatus, orgHerdrLookPath, orgHerdrServerLaunch
	t.Cleanup(func() {
		orgHerdrServerStatus, orgHerdrLookPath, orgHerdrServerLaunch = origStatus, origLook, origLaunch
	})
	orgHerdrServerStatus = func(ctx context.Context) (driver.HerdrServerStatus, error) {
		if deadline, ok := ctx.Deadline(); ok {
			f.timeLeft = append(f.timeLeft, time.Until(deadline))
		}
		if len(f.answers) == 0 {
			t.Fatal("the gate read the status, and the test gave no answer")
		}
		a := f.answers[min(f.reads, len(f.answers)-1)]
		f.reads++
		return a.st, a.err
	}
	orgHerdrLookPath = func(string) (string, error) {
		f.lookups++
		return f.lookPath, f.lookErr
	}
	orgHerdrServerLaunch = func(path string, req orgWatchLaunchRequest) (*orgWatchProcess, error) {
		f.launches = append(f.launches, herdrLaunchCall{path: path, req: req})
		return f.proc, f.launchErr
	}
}

// useHerdrStatusRunner makes the gate read the status with the real
// driver.Herdr.ServerStatus on r, for the rest of the test (call it after
// setOrgHerdrServerSeams, whose cleanup restores the production read).
func useHerdrStatusRunner(r driver.Runner) {
	orgHerdrServerStatus = func(ctx context.Context) (driver.HerdrServerStatus, error) {
		return driver.Herdr{R: r}.ServerStatus(ctx)
	}
}

// herdrStatusRunner is a driver.Runner that answers `herdr status server
// --json` with out and err, or, with block set, waits for the context to end
// and returns the error driver.ExecRunner returns then.
type herdrStatusRunner struct {
	out   string
	err   error
	block bool
}

func (r herdrStatusRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if r.block {
		<-ctx.Done()
		return "", fmt.Errorf("%s: timed out: %w", name, ctx.Err())
	}
	return r.out, r.err
}

// fakeHerdrServerProcess is a started `herdr server` with pid, already
// exited when exited is set. stopped reports whether the gate stopped it.
func fakeHerdrServerProcess(pid int, exited bool) (proc *orgWatchProcess, stopped func() bool) {
	ch := make(chan struct{})
	if exited {
		close(ch)
	}
	var stops int
	return &orgWatchProcess{pid: pid, exited: ch, stop: func() { stops++ }}, func() bool { return stops > 0 }
}

var (
	herdrNotRunning = herdrStatusAnswer{st: driver.HerdrServerStatus{Running: false, Socket: "/iso/herdr.sock"}}
	herdrRunning    = herdrStatusAnswer{st: driver.HerdrServerStatus{Running: true, Socket: "/iso/herdr.sock"}}
)

// herdrNext is how the gate's errors for verb end.
func herdrNext(verb string) string {
	return "start herdr in another terminal with 'herdr', then run ralph org " + verb + " again"
}

// checkNoHerdrLaunch checks that the gate neither looked herdr up nor
// started it.
func checkNoHerdrLaunch(t *testing.T, f *herdrGateFake) {
	t.Helper()
	if f.lookups != 0 || len(f.launches) != 0 {
		t.Errorf("lookups = %d, launches = %+v; want none", f.lookups, f.launches)
	}
}

// AC3 on the CLI side: a running server passes with one read of the status,
// whatever compatible says, and nothing is looked up, started, or written.
// The read is bounded by the 5s of orgHerdrStatusTimeout.
func TestOrgHerdrServerGate_ARunningServerStartsNothing(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name       string
		compatible *bool
	}{{"compatible", &yes}, {"not compatible", &no}, {"compatible unknown", nil}} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrGateFake{answers: []herdrStatusAnswer{{st: driver.HerdrServerStatus{Running: true, Compatible: tc.compatible, Socket: "/s"}}}}
			setOrgHerdrServerSeams(t, f)
			var stderr strings.Builder

			if err := ensureOrgHerdrServer("spawn", t.TempDir(), &stderr); err != nil {
				t.Fatalf("gate with a running server: %v", err)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q; want nothing", stderr.String())
			}
			if f.reads != 1 {
				t.Errorf("status reads = %d; want 1", f.reads)
			}
			if len(f.timeLeft) != 1 || f.timeLeft[0] > 5*time.Second || f.timeLeft[0] < 4*time.Second {
				t.Errorf("time left on the read's context = %v; want one read bounded by 5s", f.timeLeft)
			}
			checkNoHerdrLaunch(t, f)
		})
	}
}

// AC1 and AC2: with no server running, the gate starts `herdr server` from
// the PATH lookup, in HOME, with this process's environment less the herdr
// pane variables and Claude Code's session variables and with PWD set to
// HOME, logging to herdr-launch.log of the ledger; it waits until the status
// says running, and writes one stderr line with the pid, the socket, the
// log, and what to run, in the words of each verb. The server is not
// stopped.
func TestOrgHerdrServerGate_AStoppedServerIsStartedAndAnnounced(t *testing.T) {
	for _, tc := range []struct{ verb, sees string }{
		{"spawn", "the seats' panes and to answer the first seat's prompts"},
		{"start", "the seats' panes and to answer the first seat's prompts"},
		{"director start", "the director's pane and to answer its first prompts"},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("PWD", "/where/ralph/ran")
			for name, value := range map[string]string{
				"CLAUDECODE": "1", "CLAUDE_CODE_MESSAGING_TOKEN": "token", "CLAUDE_CODE_ENTRYPOINT": "cli",
				"HERDR_PANE_ID": "p-9", "HERDR_TAB_ID": "t-9", "HERDR_WORKSPACE_ID": "w-9", "HERDR_ENV": "1",
				"XDG_CONFIG_HOME": "/iso/config", "XDG_STATE_HOME": "/iso/state", "HERDR_SOCKET_PATH": "/iso/herdr.sock",
				"CODEX_HOME": "/iso/codex", "CLAUDE_CONFIG_DIR": "/iso/claude",
			} {
				t.Setenv(name, value)
			}
			proc, stopped := fakeHerdrServerProcess(4701, false)
			f := &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning, herdrNotRunning, herdrRunning}, lookPath: "/opt/herdr/bin/herdr", proc: proc}
			setOrgHerdrServerSeams(t, f)
			stateDir := t.TempDir()
			var stderr strings.Builder

			if err := ensureOrgHerdrServer(tc.verb, stateDir, &stderr); err != nil {
				t.Fatalf("gate with a stopped server: %v", err)
			}
			logPath := filepath.Join(stateDir, "herdr-launch.log")
			want := fmt.Sprintf("ralph org %s: started the herdr server (pid 4701, socket /iso/herdr.sock); run 'herdr' in another terminal to see %s; log: %s\n",
				tc.verb, tc.sees, logPath)
			if stderr.String() != want {
				t.Errorf("stderr = %q; want %q", stderr.String(), want)
			}
			if len(f.launches) != 1 {
				t.Fatalf("launches = %+v; want one", f.launches)
			}
			got := f.launches[0]
			if got.path != "/opt/herdr/bin/herdr" || !slices.Equal(got.req.args, []string{"server"}) || got.req.dir != home || got.req.logPath != logPath {
				t.Errorf("launch = %s %q in %q, log %q; want /opt/herdr/bin/herdr [server] in %q, log %q",
					got.path, got.req.args, got.req.dir, got.req.logPath, home, logPath)
			}
			for _, dropped := range []string{"CLAUDECODE", "CLAUDE_CODE_MESSAGING_TOKEN", "CLAUDE_CODE_ENTRYPOINT",
				"HERDR_PANE_ID", "HERDR_TAB_ID", "HERDR_WORKSPACE_ID", "HERDR_ENV"} {
				if slices.ContainsFunc(got.req.env, func(kv string) bool { return strings.HasPrefix(kv, dropped+"=") }) {
					t.Errorf("the server's environment has %s; want it dropped", dropped)
				}
			}
			for _, kept := range []string{"XDG_CONFIG_HOME=/iso/config", "XDG_STATE_HOME=/iso/state", "HERDR_SOCKET_PATH=/iso/herdr.sock",
				"CODEX_HOME=/iso/codex", "CLAUDE_CONFIG_DIR=/iso/claude", "PATH=" + os.Getenv("PATH"), "HOME=" + home} {
				if !slices.Contains(got.req.env, kept) {
					t.Errorf("the server's environment lacks %s", kept)
				}
			}
			if pwds := slices.DeleteFunc(slices.Clone(got.req.env), func(kv string) bool { return !strings.HasPrefix(kv, "PWD=") }); !slices.Equal(pwds, []string{"PWD=" + home}) {
				t.Errorf("PWD entries = %q; want only PWD=%s", pwds, home)
			}
			if f.reads != 3 || f.lookups != 1 {
				t.Errorf("status reads = %d, lookups = %d; want 3 and 1", f.reads, f.lookups)
			}
			if stopped() {
				t.Error("the gate stopped the server it started")
			}
		})
	}
}

// AC5 (spec FR-A2): a launch that fails passes when the status read right
// after it says running (another ralph started the server), with no stderr
// line; when it does not say running, the gate fails with the launch's
// error, and with the read's error when that read failed too.
func TestOrgHerdrServerGate_ALaunchThatFails(t *testing.T) {
	launchErr := errors.New("start /opt/herdr/bin/herdr: permission denied")
	for _, tc := range []struct {
		name  string
		after herdrStatusAnswer
		want  string
	}{
		{"another server answers", herdrRunning, ""},
		{"no server", herdrNotRunning,
			"org: start: could not start the herdr server (start /opt/herdr/bin/herdr: permission denied), so nothing was made; " + herdrNext("start")},
		{"the status cannot be read after it", herdrStatusAnswer{err: errors.New("herdr status server: herdr: exit status 1")},
			"org: start: could not start the herdr server (start /opt/herdr/bin/herdr: permission denied), " +
				"and its status could not be read after that (herdr status server: herdr: exit status 1), so nothing was made; " + herdrNext("start")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning, tc.after}, lookPath: "/opt/herdr/bin/herdr", launchErr: launchErr}
			setOrgHerdrServerSeams(t, f)
			var stderr strings.Builder

			err := ensureOrgHerdrServer("start", t.TempDir(), &stderr)
			if tc.want == "" && err != nil {
				t.Fatalf("gate after a failed launch with a server running: %v", err)
			}
			if tc.want != "" && (err == nil || err.Error() != tc.want) {
				t.Fatalf("gate after a failed launch = %v; want %q", err, tc.want)
			}
			if tc.want != "" && !errors.Is(err, launchErr) {
				t.Errorf("error %v does not wrap the launch's error", err)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q; want nothing", stderr.String())
			}
			if f.reads != 2 || len(f.launches) != 1 {
				t.Errorf("status reads = %d, launches = %d; want 2 and 1", f.reads, len(f.launches))
			}
		})
	}
}

// AC5 and plan D4: when the server the gate started exits before the status
// says running, the gate reads the status once more. Running passes with no
// stderr line (another ralph's server won, and herdr ended this one with
// "already running"); anything else fails with the pid and the log.
func TestOrgHerdrServerGate_AServerThatExitsFirst(t *testing.T) {
	stateDir := t.TempDir()
	logPath := filepath.Join(stateDir, "herdr-launch.log")
	for _, tc := range []struct {
		name    string
		answers []herdrStatusAnswer
		want    string
	}{
		{"another server answers", []herdrStatusAnswer{herdrNotRunning, herdrNotRunning, herdrRunning}, ""},
		{"another server answers on the first read", []herdrStatusAnswer{herdrNotRunning, herdrRunning}, ""},
		{"no server", []herdrStatusAnswer{herdrNotRunning},
			"org: spawn: the herdr server (pid 4702) exited before it answered, so nothing was made; see " + logPath + ", or " + herdrNext("spawn")},
		{"the status cannot be read after it", []herdrStatusAnswer{herdrNotRunning, herdrNotRunning, {err: errors.New("herdr status server: herdr: exit status 1")}},
			"org: spawn: the herdr server (pid 4702) exited before it answered, and its status could not be read after that " +
				"(herdr status server: herdr: exit status 1), so nothing was made; see " + logPath + ", or " + herdrNext("spawn")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proc, _ := fakeHerdrServerProcess(4702, true)
			f := &herdrGateFake{answers: tc.answers, lookPath: "/opt/herdr/bin/herdr", proc: proc}
			setOrgHerdrServerSeams(t, f)
			var stderr strings.Builder

			err := ensureOrgHerdrServer("spawn", stateDir, &stderr)
			if tc.want == "" && err != nil {
				t.Fatalf("gate whose server exited while another answers: %v", err)
			}
			if tc.want != "" && (err == nil || err.Error() != tc.want) {
				t.Fatalf("gate whose server exited = %v; want %q", err, tc.want)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q; want nothing (this ralph's server is not the one that answers)", stderr.String())
			}
		})
	}
}

// AC4: a server that does not answer within the deadline (shortened to
// 200ms) fails the gate with its pid, the deadline, what the last read
// found, and the log; the server is not stopped, and no read's context
// reaches past the deadline.
func TestOrgHerdrServerGate_NoAnswerWithinTheDeadline(t *testing.T) {
	for _, tc := range []struct {
		name string
		last herdrStatusAnswer
		says string
	}{
		{"not running", herdrNotRunning, "the last read of its status said it was not running"},
		{"a read that fails", herdrStatusAnswer{err: errors.New("herdr status server: herdr: timed out: context deadline exceeded")},
			"the last read of its status failed: herdr status server: herdr: timed out: context deadline exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setOrgWatchDuration(t, &orgHerdrServerWaitDeadline, 200*time.Millisecond)
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			proc, stopped := fakeHerdrServerProcess(4703, false)
			f := &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning, herdrNotRunning, tc.last}, lookPath: "/opt/herdr/bin/herdr", proc: proc}
			setOrgHerdrServerSeams(t, f)
			stateDir := t.TempDir()
			var stderr strings.Builder

			start := time.Now()
			err := ensureOrgHerdrServer("director start", stateDir, &stderr)
			elapsed := time.Since(start)
			want := "org: director start: the herdr server (pid 4703) did not answer within 200ms (" + tc.says + "), so nothing was made; see " +
				filepath.Join(stateDir, "herdr-launch.log") + ", or " + herdrNext("director start")
			if err == nil || err.Error() != want {
				t.Fatalf("gate past the deadline = %v; want %q", err, want)
			}
			if elapsed < 200*time.Millisecond {
				t.Errorf("the gate failed after %v; want it to wait the 200ms deadline first", elapsed)
			}
			if stopped() {
				t.Error("the gate stopped the server at the deadline; want it left running")
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q; want nothing", stderr.String())
			}
			for i, left := range f.timeLeft[1:] { // the first read, before the launch, has its own 5s
				if left > 200*time.Millisecond {
					t.Errorf("read %d of the wait had %v left; want no more than the 200ms deadline", i+1, left)
				}
			}
		})
	}
}

// AC4: with herdr not on PATH the gate fails, says so, and starts nothing,
// whether the lookup after a status that says not running fails or the
// status read itself finds no herdr to run.
func TestOrgHerdrServerGate_HerdrNotOnPATH(t *testing.T) {
	notFound := &exec.Error{Name: "herdr", Err: exec.ErrNotFound}
	for _, tc := range []struct {
		name  string
		f     *herdrGateFake
		cause string
	}{
		{"the lookup fails", &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning}, lookErr: notFound}, notFound.Error()},
		{"the status read finds no herdr", &herdrGateFake{answers: []herdrStatusAnswer{{err: fmt.Errorf("herdr status server: herdr: %w", notFound)}}},
			"herdr status server: herdr: " + notFound.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setOrgHerdrServerSeams(t, tc.f)
			var stderr strings.Builder

			err := ensureOrgHerdrServer("spawn", t.TempDir(), &stderr)
			want := "org: spawn: herdr is not on PATH (" + tc.cause + "), so ralph cannot start the herdr server, and nothing was made; " +
				"install herdr, or start it in another terminal with 'herdr', then run ralph org spawn again"
			if err == nil || err.Error() != want {
				t.Fatalf("gate with no herdr = %v; want %q", err, want)
			}
			if len(tc.f.launches) != 0 {
				t.Errorf("launches = %+v; want none", tc.f.launches)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q; want nothing", stderr.String())
			}
		})
	}
}

// AC4 and plan D3: a status that cannot be read (a runner error, output that
// is not JSON, JSON with no running, a read that runs out its 5s, shortened
// here) fails the gate with the cause, through the real ServerStatus, and
// nothing is looked up, started, or written.
func TestOrgHerdrServerGate_AStatusThatCannotBeReadFails(t *testing.T) {
	setOrgWatchDuration(t, &orgHerdrStatusTimeout, 50*time.Millisecond)
	for _, tc := range []struct {
		name  string
		r     herdrStatusRunner
		cause string
	}{
		{"runner error", herdrStatusRunner{err: errors.New("herdr: exit status 1: protocol mismatch")}, "herdr status server: herdr: exit status 1: protocol mismatch"},
		{"not JSON", herdrStatusRunner{out: "server is up"}, "herdr status server: cannot read the output as its JSON object: "},
		{"no running", herdrStatusRunner{out: `{"status":"running"}`}, `herdr status server: "running" is missing or null in the output: `},
		{"timeout", herdrStatusRunner{block: true}, "herdr status server: herdr: timed out: context deadline exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrGateFake{}
			setOrgHerdrServerSeams(t, f)
			useHerdrStatusRunner(tc.r)
			var stderr strings.Builder

			start := time.Now()
			err := ensureOrgHerdrServer("spawn", t.TempDir(), &stderr)
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Errorf("the gate took %v; want the read bounded by its 50ms", elapsed)
			}
			const prefix = "org: spawn: could not read the herdr server's status ("
			if err == nil || !strings.HasPrefix(err.Error(), prefix+tc.cause) ||
				!strings.HasSuffix(err.Error(), "), so nothing was made; "+herdrNext("spawn")) {
				t.Fatalf("gate with an unreadable status = %v; want %q...%q", err, prefix+tc.cause, "), so nothing was made; "+herdrNext("spawn"))
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q; want nothing", stderr.String())
			}
			checkNoHerdrLaunch(t, f)
		})
	}
}

// Plan D3 and R5: a herdr that answers `unknown option: --json`, on stdout
// or in the runner's error, gets one warning line on stderr and passes, and
// nothing is looked up or started.
func TestOrgHerdrServerGate_AHerdrWithoutStatusJSONWarnsAndPasses(t *testing.T) {
	for _, tc := range []struct {
		name string
		r    herdrStatusRunner
	}{
		{"in the runner's error", herdrStatusRunner{err: errors.New("herdr: exit status 2: unknown option: --json")}},
		{"on stdout", herdrStatusRunner{out: "unknown option: --json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrGateFake{}
			setOrgHerdrServerSeams(t, f)
			useHerdrStatusRunner(tc.r)
			var stderr strings.Builder

			if err := ensureOrgHerdrServer("start", t.TempDir(), &stderr); err != nil {
				t.Fatalf("gate with an old herdr: %v", err)
			}
			const want = "warning: this herdr has no 'herdr status server --json' (herdr 0.7.5 has it), " +
				"so ralph org start cannot check or start the herdr server and goes on as before\n"
			if stderr.String() != want {
				t.Errorf("stderr = %q; want the one line %q", stderr.String(), want)
			}
			checkNoHerdrLaunch(t, f)
		})
	}
}

// AC2 (spec FR-A3), the names pinned: the server's environment drops
// exactly HERDR_PANE_ID, HERDR_TAB_ID, HERDR_WORKSPACE_ID, HERDR_ENV,
// CLAUDECODE, and every name that starts with CLAUDE_CODE_, and keeps the
// rest in order; PWD is replaced by the server's directory, and kept as it
// is when there is none.
func TestOrgHerdrServerEnv_DropsThePaneAndClaudeCodeVariablesOnly(t *testing.T) {
	environ := []string{
		"PATH=/usr/bin", "HOME=/home/u", "PWD=/work",
		"HERDR_PANE_ID=p", "HERDR_TAB_ID=t", "HERDR_WORKSPACE_ID=w", "HERDR_ENV=1", "HERDR_SOCKET_PATH=/iso/herdr.sock",
		"CLAUDECODE=1", "CLAUDE_CODE_MESSAGING_TOKEN=tok", "CLAUDE_CODE_SSE_PORT=1234", "CLAUDE_CODE_=x",
		"CLAUDE_CONFIG_DIR=/iso/claude", "CLAUDE_CODE=keep", "CLAUDECODE_X=keep",
		"XDG_CONFIG_HOME=/iso/config", "XDG_STATE_HOME=/iso/state", "CODEX_HOME=/iso/codex", "CODEX_SANDBOX=seatbelt",
		"NOEQUALS",
	}
	kept := []string{
		"PATH=/usr/bin", "HOME=/home/u", "HERDR_SOCKET_PATH=/iso/herdr.sock",
		"CLAUDE_CONFIG_DIR=/iso/claude", "CLAUDE_CODE=keep", "CLAUDECODE_X=keep",
		"XDG_CONFIG_HOME=/iso/config", "XDG_STATE_HOME=/iso/state", "CODEX_HOME=/iso/codex", "CODEX_SANDBOX=seatbelt",
		"NOEQUALS",
	}
	if got, want := orgHerdrServerEnv(environ, "/home/u"), append(slices.Clone(kept), "PWD=/home/u"); !slices.Equal(got, want) {
		t.Errorf("env with a directory =\n%q\nwant\n%q", got, want)
	}
	withPWD := slices.Insert(slices.Clone(kept), 2, "PWD=/work")
	if got := orgHerdrServerEnv(environ, ""); !slices.Equal(got, withPWD) {
		t.Errorf("env with no directory =\n%q\nwant\n%q", got, withPWD)
	}
}

// AC8: in a test, starting a herdr server panics until the test sets a fake,
// and the gate the verbs get is TestMain's, which passes.
func TestOrgHerdrServerLaunch_TestMainMakesARealLaunchPanic(t *testing.T) {
	if orgHerdrServerGateOverride == nil || orgHerdrServerGateOverride() != nil {
		t.Error("orgHerdrServerGateOverride is not TestMain's gate that passes")
	}
	defer func() {
		r := recover()
		if msg := fmt.Sprint(r); !strings.Contains(msg, "a test tried to start a real herdr server") ||
			!strings.Contains(msg, "set orgHerdrServerLaunch to a fake") {
			t.Errorf("recover() = %v; want TestMain's panic", r)
		}
	}()
	_, _ = orgHerdrServerLaunch("/opt/herdr/bin/herdr", orgWatchLaunchRequest{args: []string{"server"}, logPath: "/nowhere/herdr-launch.log"})
	t.Error("orgHerdrServerLaunch returned; want a panic")
}

// setOrgHerdrServerGateOverride sets the gate the verbs get to gate for the
// rest of the test, and restores TestMain's at cleanup.
func setOrgHerdrServerGateOverride(t *testing.T, gate func() error) {
	t.Helper()
	orig := orgHerdrServerGateOverride
	orgHerdrServerGateOverride = gate
	t.Cleanup(func() { orgHerdrServerGateOverride = orig })
}

// refusingHerdrGate is a gate that refuses with err and counts its calls.
func refusingHerdrGate(err error) (gate func() error, calls *int) {
	calls = new(int)
	return func() error {
		*calls++
		return err
	}, calls
}

// AC4 on the CLI: a real spawn whose herdr server gate refuses fails with
// the gate's error as it is, prints it as a failed spawn, and records
// nothing and calls neither herdr nor agmsg; a --dry-run does not call the
// gate and spawns as before.
func TestOrgSpawn_HerdrServerGate(t *testing.T) {
	gateErr := errors.New("org: spawn: herdr is not on PATH (stand-in), so ralph cannot start the herdr server, and nothing was made")

	t.Run("a refusal records nothing", func(t *testing.T) {
		gate, calls := refusingHerdrGate(gateErr)
		setOrgHerdrServerGateOverride(t, gate)
		herdrLog, agmsgLog := setupOrgStubPATH(t)
		stateDir := t.TempDir()

		stdout, _, err := runOrgCmdStreams(t, gateSpawnArgs(t, stateDir)...)
		if err == nil || err.Error() != gateErr.Error() {
			t.Fatalf("spawn with a refusing gate = %v; want %v", err, gateErr)
		}
		if want := "spawn failed: " + gateErr.Error() + "\n"; stdout != want {
			t.Errorf("stdout = %q; want only %q", stdout, want)
		}
		if *calls != 1 {
			t.Errorf("gate calls = %d; want 1", *calls)
		}
		checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
	})

	t.Run("a --dry-run does not call it", func(t *testing.T) {
		gate, calls := refusingHerdrGate(gateErr)
		setOrgHerdrServerGateOverride(t, gate)
		setupOrgStubPATH(t)
		stateDir := t.TempDir()

		stdout, stderr, err := runOrgCmdStreams(t, gateSpawnArgs(t, stateDir, "--dry-run")...)
		if err != nil {
			t.Fatalf("spawn --dry-run: %v (stdout: %s stderr: %s)", err, stdout, stderr)
		}
		if *calls != 0 {
			t.Errorf("gate calls = %d; want none for a --dry-run", *calls)
		}
		if ev := lastSeatEvent(t, stateDir, "org-a", "seat-1"); ev.Event != org.EventSpawned || !ev.DryRun {
			t.Errorf("last event = %s (dry_run=%t); want a dry-run spawned", ev.Event, ev.DryRun)
		}
	})
}

// AC4 on the CLI for `ralph org start <task>`: a refusal of the gate fails
// the start with the gate's error, and nothing is recorded or called.
func TestOrgStart_HerdrServerGate_ARefusalRecordsNothing(t *testing.T) {
	gateErr := errors.New("org: start: could not read the herdr server's status (stand-in), so nothing was made")
	gate, calls := refusingHerdrGate(gateErr)
	setOrgHerdrServerGateOverride(t, gate)
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := t.TempDir()

	stdout, _, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide", "--state-dir", stateDir, "do the task")
	if err == nil || err.Error() != gateErr.Error() {
		t.Fatalf("start with a refusing gate = %v; want %v", err, gateErr)
	}
	if want := "spawn failed: " + gateErr.Error() + "\n"; stdout != want {
		t.Errorf("stdout = %q; want only %q", stdout, want)
	}
	if *calls != 1 {
		t.Errorf("gate calls = %d; want 1", *calls)
	}
	checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
}

// AC4 on the CLI for `ralph org start --plan`: a refusal of the gate comes
// before the feature worktree, so there is no worktree, no branch, no
// manifest event, and no herdr or agmsg call.
func TestOrgStartPlan_HerdrServerGate_ARefusalMakesNoWorktree(t *testing.T) {
	gateErr := errors.New("org: start: the herdr server (pid 1) did not answer within 10s, so nothing was made")
	gate, calls := refusingHerdrGate(gateErr)
	setOrgHerdrServerGateOverride(t, gate)
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(tmp, "state")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))

	_, _, err = startOrgFeature(t, plan, "a", "--state-dir", stateDir)
	if err == nil || err.Error() != gateErr.Error() {
		t.Fatalf("start --plan with a refusing gate = %v; want %v", err, gateErr)
	}
	if *calls != 1 {
		t.Errorf("gate calls = %d; want 1", *calls)
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

// AC4 on the CLI for `ralph org director start` in herdr: a refusal of the
// gate fails the start with its error, writes no registration, and runs no
// herdr.
func TestOrgDirectorStart_HerdrServerGate_ARefusalWritesNothing(t *testing.T) {
	gateErr := errors.New("org: director start: herdr is not on PATH (stand-in), so ralph cannot start the herdr server, and nothing was made")
	gate, calls := refusingHerdrGate(gateErr)
	setOrgHerdrServerGateOverride(t, gate)
	stateDir, herdrLog, _ := directorStartLedger(t)

	stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
	if err == nil || err.Error() != gateErr.Error() {
		t.Fatalf("director start with a refusing gate = %v; want %v", err, gateErr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if *calls != 1 {
		t.Errorf("gate calls = %d; want 1", *calls)
	}
	if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
		t.Errorf("director.jsonl = %s; want nothing written", got)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
}

// AC1 through the verbs, with the real gate wired (orgHerdrServerGateOverride
// nil) on the fake seams: spawn, start, and director start each start the
// stopped server once, say so on stderr in their own words, and go on to
// make what they make with the stub herdr.
func TestOrgHerdrServerGate_EachVerbStartsAStoppedServerAndGoesOn(t *testing.T) {
	for _, tc := range []struct {
		verb    string
		args    func(t *testing.T, stateDir string) []string
		started string
	}{
		{"spawn", func(t *testing.T, stateDir string) []string { return gateSpawnArgs(t, stateDir) }, `spawned seat "seat-1" (org_id=org-a `},
		{"start", func(t *testing.T, stateDir string) []string {
			return []string{"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
				"--cwd", t.TempDir(), "--scope", "org-wide", "--state-dir", stateDir, "do the task"}
		}, `spawned seat "leader" (org_id=org-a `},
		{"director start", func(t *testing.T, stateDir string) []string {
			return []string{"director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir}
		}, "started d1 (driver=claude, "},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			setOrgHerdrServerGateOverride(t, nil)
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			proc, _ := fakeHerdrServerProcess(4720, false)
			f := &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning, herdrRunning}, lookPath: "/opt/herdr/bin/herdr", proc: proc}
			setOrgHerdrServerSeams(t, f)
			stateDir, _, _ := directorStartLedger(t)

			stdout, stderr, err := runOrgCmdStreams(t, tc.args(t, stateDir)...)
			if err != nil {
				t.Fatalf("%s with a stopped server: %v (stdout: %s stderr: %s)", tc.verb, err, stdout, stderr)
			}
			want := "ralph org " + tc.verb + ": started the herdr server (pid 4720, socket /iso/herdr.sock); "
			if strings.Count(stderr, "started the herdr server") != 1 || !strings.Contains(stderr, want) {
				t.Errorf("stderr = %q; want one line with %q", stderr, want)
			}
			if !strings.HasPrefix(stdout, tc.started) {
				t.Errorf("stdout = %q; want it to start %q", stdout, tc.started)
			}
			if len(f.launches) != 1 {
				t.Errorf("launches = %d; want 1", len(f.launches))
			}
		})
	}
}

// AC1 for `ralph org start --plan` (spec AC-A1), with the real gate wired on
// the fake seams: start --plan starts the stopped server once and says so in
// one line in start's words, and the Spawn inside StartFeature, which asks
// the gate again (plan D1), reads the status once more and starts nothing: 1
// launch and 3 status reads (2 by StartFeature's gate, 1 by Spawn's).
func TestOrgStartPlan_HerdrServerGate_StartsAStoppedServerOnceInStartsWords(t *testing.T) {
	setOrgHerdrServerGateOverride(t, nil)
	setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
	proc, _ := fakeHerdrServerProcess(4721, false)
	f := &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning, herdrRunning}, lookPath: "/opt/herdr/bin/herdr", proc: proc}
	setOrgHerdrServerSeams(t, f)
	setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(tmp, "state")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))

	stdout, stderr, err := startOrgFeature(t, plan, "a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("start --plan with a stopped server: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	want := "ralph org start: started the herdr server (pid 4721, socket /iso/herdr.sock); run 'herdr' in another terminal to see the seats' panes and to answer the first seat's prompts; log: " +
		filepath.Join(stateDir, "herdr-launch.log") + "\n"
	if strings.Count(stderr, "started the herdr server") != 1 || !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q; want the one line %q", stderr, want)
	}
	if !strings.HasPrefix(stdout, `spawned seat "leader" (org_id=a `) {
		t.Errorf("stdout = %q; want the leader spawned", stdout)
	}
	if len(f.launches) != 1 || f.reads != 3 {
		t.Errorf("launches = %d, status reads = %d; want 1 launch and 3 reads (StartFeature 2, Spawn 1)", len(f.launches), f.reads)
	}
}

// AC7: the --help of spawn, start, and director start says when it makes
// sure the herdr server runs, and how it starts one.
func TestOrgHerdrServerGate_TheHelpOfEachVerbSaysItStartsTheServer(t *testing.T) {
	shared := []string{
		"When `herdr status server --json` says the server is not running, ralph starts `herdr server` in the background, in your home directory,",
		"less the herdr pane variables (HERDR_PANE_ID, HERDR_TAB_ID, HERDR_WORKSPACE_ID, HERDR_ENV) and Claude Code's session variables (CLAUDECODE, CLAUDE_CODE_*), " +
			"waits up to 10s for it to answer,",
		"with its pid, its socket, and <state dir>/herdr-launch.log, where its output goes; run `herdr` in another terminal",
		"the command fails with nothing made: start herdr in another terminal with `herdr`, then run the command again.",
		"A herdr that has no `herdr status server --json` (herdr 0.7.5 has it) gets one warning line on stderr and is neither checked nor started.",
	}
	for _, tc := range []struct {
		args []string
		lead string
	}{
		{[]string{"spawn", "--help"}, "A spawn that is not a --dry-run then makes sure the herdr server runs."},
		{[]string{"start", "--help"}, "Both forms then make sure the herdr server runs, the --plan form before it makes the worktree."},
		{[]string{"director", "start", "--help"}, "A start in herdr (not one with launch_cmd) first makes sure the herdr server runs, before it looks for an attached director (see below)."},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range append([]string{tc.lead}, shared...) {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
	}
}
