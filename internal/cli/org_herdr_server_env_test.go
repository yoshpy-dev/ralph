package cli

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// Tests for the environment of the herdr server (plan
// 2026-11-20-org-reseat-restart-herdr, S1: AC5 for the environment of the
// server the gate starts, AC6 for the read of the server's
// CLAUDE_CONFIG_DIR). The read's tests give it a fake probe
// (herdrServerEnvFake), so no lsof or ps reaches the machine's herdr server.
// None of these tests runs in parallel: they swap package-level vars.

// claudeConfigDirEntries is the CLAUDE_CONFIG_DIR entries of env, in order,
// and others the rest.
func claudeConfigDirEntries(env []string) (entries, others []string) {
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); name == "CLAUDE_CONFIG_DIR" {
			entries = append(entries, kv)
		} else {
			others = append(others, kv)
		}
	}
	return entries, others
}

// AC5, the override itself: not given, it returns env as it is; a value
// takes the place of every CLAUDE_CONFIG_DIR entry, after the others; ""
// removes them. The other entries, names that only contain
// CLAUDE_CONFIG_DIR included, keep their order, and env itself is not
// changed.
func TestOrgHerdrServerEnvOverride_ApplySetsReplacesOrRemovesClaudeConfigDir(t *testing.T) {
	env := []string{
		"PATH=/usr/bin", "CLAUDE_CONFIG_DIR=/old", "HOME=/h", "CLAUDE_CONFIG_DIR=/dup",
		"CLAUDE_CONFIG_DIRX=keep", "MY_CLAUDE_CONFIG_DIR=keep", "CLAUDE_CONFIG_DIR",
	}
	others := []string{"PATH=/usr/bin", "HOME=/h", "CLAUDE_CONFIG_DIRX=keep", "MY_CLAUDE_CONFIG_DIR=keep"}
	for _, tc := range []struct {
		name     string
		override orgHerdrServerEnvOverride
		env      []string
		want     []string
	}{
		{"not given", orgHerdrServerEnvOverride{}, env, env},
		{"not given, with a value left in the struct", orgHerdrServerEnvOverride{claudeConfigDir: "/ignored"}, env, env},
		{"a value", orgHerdrServerEnvOverride{setClaudeConfigDir: true, claudeConfigDir: "/new dir"}, env,
			append(slices.Clone(others), "CLAUDE_CONFIG_DIR=/new dir")},
		{"a value where there was none", orgHerdrServerEnvOverride{setClaudeConfigDir: true, claudeConfigDir: "/new"}, others,
			append(slices.Clone(others), "CLAUDE_CONFIG_DIR=/new")},
		{"empty", orgHerdrServerEnvOverride{setClaudeConfigDir: true}, env, others},
		{"empty where there was none", orgHerdrServerEnvOverride{setClaudeConfigDir: true}, others, others},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := slices.Clone(tc.env)
			if got := tc.override.apply(tc.env); !slices.Equal(got, tc.want) {
				t.Errorf("apply =\n%q\nwant\n%q", got, tc.want)
			}
			if !slices.Equal(tc.env, before) {
				t.Errorf("apply changed its argument to %q", tc.env)
			}
		})
	}
}

// AC5 through the gate: a stopped server is started with this process's
// CLAUDE_CONFIG_DIR when no override is given (ensureOrgHerdrServer, as
// before), with the override's value in its place, or with none for "".
// Every other entry of the server's environment is what it was without the
// override, the herdr pane and Claude Code variables still dropped and PWD
// still the home directory.
func TestOrgHerdrServerGate_StartsTheServerWithTheClaudeConfigDirOverride(t *testing.T) {
	for _, tc := range []struct {
		name     string
		override *orgHerdrServerEnvOverride
		want     []string
	}{
		{"not given", nil, []string{"CLAUDE_CONFIG_DIR=/old"}},
		{"a value", &orgHerdrServerEnvOverride{setClaudeConfigDir: true, claudeConfigDir: "/new"}, []string{"CLAUDE_CONFIG_DIR=/new"}},
		{"empty", &orgHerdrServerEnvOverride{setClaudeConfigDir: true}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			home := t.TempDir()
			t.Setenv("HOME", home)
			for name, value := range map[string]string{
				"CLAUDE_CONFIG_DIR": "/old", "CLAUDECODE": "1", "CLAUDE_CODE_ENTRYPOINT": "cli",
				"HERDR_PANE_ID": "p-9", "XDG_CONFIG_HOME": "/iso/config", "CODEX_HOME": "/iso/codex",
			} {
				t.Setenv(name, value)
			}
			proc, _ := fakeHerdrServerProcess(4701, false)
			f := &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning, herdrRunning}, lookPath: "/opt/herdr/bin/herdr", proc: proc}
			setOrgHerdrServerSeams(t, f)
			var stderr strings.Builder

			var err error
			if tc.override == nil {
				err = ensureOrgHerdrServer("reseat", t.TempDir(), &stderr)
			} else {
				err = ensureOrgHerdrServerWithEnv("reseat", t.TempDir(), &stderr, *tc.override)
			}
			if err != nil {
				t.Fatalf("gate with a stopped server: %v", err)
			}
			if len(f.launches) != 1 {
				t.Fatalf("launches = %+v; want one", f.launches)
			}
			got, gotOthers := claudeConfigDirEntries(f.launches[0].req.env)
			if !slices.Equal(got, tc.want) {
				t.Errorf("CLAUDE_CONFIG_DIR entries of the server's environment = %q; want %q", got, tc.want)
			}
			_, wantOthers := claudeConfigDirEntries(orgHerdrServerEnv(os.Environ(), home))
			if !slices.Equal(gotOthers, wantOthers) {
				t.Errorf("the other entries of the server's environment =\n%q\nwant\n%q", gotOthers, wantOthers)
			}
			for _, kv := range []string{"XDG_CONFIG_HOME=/iso/config", "CODEX_HOME=/iso/codex", "PWD=" + home} {
				if !slices.Contains(gotOthers, kv) {
					t.Errorf("the server's environment lacks %s", kv)
				}
			}
			for _, dropped := range []string{"CLAUDECODE=", "CLAUDE_CODE_ENTRYPOINT=", "HERDR_PANE_ID="} {
				if slices.ContainsFunc(gotOthers, func(kv string) bool { return strings.HasPrefix(kv, dropped) }) {
					t.Errorf("the server's environment has %s; want it dropped", dropped)
				}
			}
		})
	}
}

// AC5 through the wiring: the gate wireOrgHerdrServerGateWithEnv sets
// starts the server with the override, writing its line on the command's
// stderr in the verb's words; wireOrgHerdrServerGate's gate starts it with
// this process's CLAUDE_CONFIG_DIR; and with orgHerdrServerGateOverride set,
// that gate replaces the whole gate, override included, and nothing starts.
func TestOrgHerdrServerGate_TheWiringCarriesTheClaudeConfigDirOverride(t *testing.T) {
	override := orgHerdrServerEnvOverride{setClaudeConfigDir: true, claudeConfigDir: "/new"}
	for _, tc := range []struct {
		name string
		wire func(cmd *cobra.Command, rt *org.Org, stateDir string)
		want []string
	}{
		{"with the override", func(cmd *cobra.Command, rt *org.Org, stateDir string) {
			wireOrgHerdrServerGateWithEnv(cmd, rt, stateDir, "reseat", override)
		}, []string{"CLAUDE_CONFIG_DIR=/new"}},
		{"without one", func(cmd *cobra.Command, rt *org.Org, stateDir string) {
			wireOrgHerdrServerGate(cmd, rt, stateDir, "reseat")
		}, []string{"CLAUDE_CONFIG_DIR=/old"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setOrgHerdrServerGateOverride(t, nil)
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			t.Setenv("CLAUDE_CONFIG_DIR", "/old")
			proc, _ := fakeHerdrServerProcess(4701, false)
			f := &herdrGateFake{answers: []herdrStatusAnswer{herdrNotRunning, herdrRunning}, lookPath: "/opt/herdr/bin/herdr", proc: proc}
			setOrgHerdrServerSeams(t, f)
			cmd := &cobra.Command{}
			var stderr strings.Builder
			cmd.SetErr(&stderr)
			rt := &org.Org{}

			tc.wire(cmd, rt, t.TempDir())
			if rt.HerdrServerGate == nil {
				t.Fatal("HerdrServerGate is nil")
			}
			if err := rt.HerdrServerGate(); err != nil {
				t.Fatalf("HerdrServerGate: %v", err)
			}
			if len(f.launches) != 1 {
				t.Fatalf("launches = %+v; want one", f.launches)
			}
			if got, _ := claudeConfigDirEntries(f.launches[0].req.env); !slices.Equal(got, tc.want) {
				t.Errorf("CLAUDE_CONFIG_DIR entries of the server's environment = %q; want %q", got, tc.want)
			}
			if !strings.HasPrefix(stderr.String(), "ralph org reseat: started the herdr server (pid 4701,") {
				t.Errorf("stderr = %q; want the gate's started line for reseat", stderr.String())
			}
		})
	}
	t.Run("orgHerdrServerGateOverride replaces the gate", func(t *testing.T) {
		gate, calls := refusingHerdrGate(errors.New("stand-in"))
		setOrgHerdrServerGateOverride(t, gate)
		f := &herdrGateFake{}
		setOrgHerdrServerSeams(t, f)
		rt := &org.Org{}

		wireOrgHerdrServerGateWithEnv(&cobra.Command{}, rt, t.TempDir(), "reseat", override)
		if err := rt.HerdrServerGate(); err == nil || err.Error() != "stand-in" || *calls != 1 {
			t.Fatalf("HerdrServerGate = %v after %d calls; want the override's error once", err, *calls)
		}
		if f.reads != 0 || len(f.launches) != 0 {
			t.Errorf("status reads = %d, launches = %+v; want none", f.reads, f.launches)
		}
	})
}

// AC5, the check of a --claude-config-dir value: "" passes as "", an
// existing directory passes made absolute (a relative one against the
// working directory, a symlink to a directory as the symlink's path), and a
// file or a path that does not exist is refused, naming the value.
func TestOrgHerdrServerCheckClaudeConfigDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "claude b")
	file := filepath.Join(root, "file")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	// t.Chdir sets PWD, and filepath.Abs reads os.Getwd, which may spell
	// root another way (macOS's /private/var for /var).
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ name, value, want string }{
		{"empty", "", ""},
		{"an absolute directory", dir, dir},
		{"a relative directory", "claude b", filepath.Join(wd, "claude b")},
		{"a directory with dots", "./x/../claude b/", filepath.Join(wd, "claude b")},
		{"a symlink to a directory", link, link},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := checkOrgClaudeConfigDir(tc.value); err != nil || got != tc.want {
				t.Errorf("checkOrgClaudeConfigDir(%q) = %q, %v; want %q", tc.value, got, err, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		name, value, want string
		wantIs            error
	}{
		{"a file", file, "--claude-config-dir \"" + file + "\" is not a directory", nil},
		{"a path that does not exist", filepath.Join(root, "none"),
			"--claude-config-dir \"" + filepath.Join(root, "none") + "\" is not an existing directory: ", fs.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := checkOrgClaudeConfigDir(tc.value)
			if err == nil || got != "" {
				t.Fatalf("checkOrgClaudeConfigDir(%q) = %q, %v; want a refusal", tc.value, got, err)
			}
			if !strings.HasPrefix(err.Error(), tc.want) {
				t.Errorf("error = %q; want it to start %q", err.Error(), tc.want)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Errorf("errors.Is(%v, %v) = false", err, tc.wantIs)
			}
		})
	}
}

// herdrServerEnvAnswer is what the fake probe answers one call with.
type herdrServerEnvAnswer struct {
	out string
	err error
}

// herdrServerEnvCall is one call the fake probe got: the command line (the
// name and the arguments joined by spaces, "herdr status server --json"
// for the status read, "read <path>" for a file read), and the time its
// context had left (0 for a file read, which takes no context).
type herdrServerEnvCall struct {
	line string
	left time.Duration
}

// herdrServerEnvFake is the probe of one read: the status read answers st
// and statusErr, a command line answers runs[line], a file read answers
// files[path], and goos is the system ("darwin" when empty). A call it has
// no answer for fails the test. It records every call.
type herdrServerEnvFake struct {
	st        driver.HerdrServerStatus
	statusErr error
	runs      map[string]herdrServerEnvAnswer
	files     map[string]herdrServerEnvAnswer
	goos      string

	calls []herdrServerEnvCall
}

func (f *herdrServerEnvFake) record(ctx context.Context, line string) {
	var left time.Duration
	if deadline, ok := ctx.Deadline(); ok {
		left = time.Until(deadline)
	}
	f.calls = append(f.calls, herdrServerEnvCall{line: line, left: left})
}

func (f *herdrServerEnvFake) probe(t *testing.T) orgHerdrServerEnvProbe {
	t.Helper()
	goos := f.goos
	if goos == "" {
		goos = "darwin"
	}
	return orgHerdrServerEnvProbe{
		status: func(ctx context.Context) (driver.HerdrServerStatus, error) {
			f.record(ctx, "herdr status server --json")
			return f.st, f.statusErr
		},
		run: func(ctx context.Context, name string, args ...string) (string, error) {
			line := strings.Join(append([]string{name}, args...), " ")
			f.record(ctx, line)
			a, ok := f.runs[line]
			if !ok {
				t.Errorf("the read ran %q, which the test gave no answer for", line)
				return "", errors.New("no answer")
			}
			return a.out, a.err
		},
		readFile: func(name string) ([]byte, error) {
			f.calls = append(f.calls, herdrServerEnvCall{line: "read " + name})
			a, ok := f.files[name]
			if !ok {
				t.Errorf("the read read %s, which the test gave no answer for", name)
				return nil, errors.New("no answer")
			}
			return []byte(a.out), a.err
		},
		goos: goos,
	}
}

// lines is the command lines of the calls, in order.
func (f *herdrServerEnvFake) lines() []string {
	lines := make([]string, len(f.calls))
	for i, c := range f.calls {
		lines[i] = c.line
	}
	return lines
}

// herdrServerEnvRunning is a running server's status with the socket the
// fake lsof answers for.
var herdrServerEnvRunning = driver.HerdrServerStatus{Running: true, Socket: "/iso/herdr.sock"}

// herdrServerPSLine is what `ps eww -o command=` prints for the server
// (pid 4701): its command and an environment holding env between two other
// entries, the second with a space in its value.
func herdrServerPSLine(env string) string {
	return "/opt/homebrew/bin/herdr server PATH=/usr/bin:/bin " + env + " TERM_PROGRAM=Apple Terminal\n"
}

// checkHerdrServerEnvRead checks one read's result.
func checkHerdrServerEnvRead(t *testing.T, got, want orgHerdrServerClaudeConfigDir) {
	t.Helper()
	if got != want {
		t.Errorf("read = %+v; want %+v", got, want)
	}
}

// AC6, one pid: lsof lists the server's pid (once, or twice, which counts
// once), no ps reads its command, and ps eww reads the value. Each call is
// bounded: the status read by orgHerdrStatusTimeout, lsof and ps by
// orgHerdrServerEnvCallTimeout.
func TestOrgHerdrServerReadClaudeConfigDir_OnePIDFromLsof(t *testing.T) {
	for _, lsof := range []string{"4701\n", "4701\n4701\n"} {
		t.Run(strings.ReplaceAll(lsof, "\n", " "), func(t *testing.T) {
			setOrgWatchDuration(t, &orgHerdrServerEnvCallTimeout, 3*time.Second)
			f := &herdrServerEnvFake{st: herdrServerEnvRunning, runs: map[string]herdrServerEnvAnswer{
				"lsof -t /iso/herdr.sock":    {out: lsof},
				"ps eww -o command= -p 4701": {out: herdrServerPSLine("CLAUDE_CONFIG_DIR=/Users/u/.claude-b")},
			}}

			got := readOrgHerdrServerClaudeConfigDir(context.Background(), f.probe(t))
			checkHerdrServerEnvRead(t, got, orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/Users/u/.claude-b"})
			want := []string{"herdr status server --json", "lsof -t /iso/herdr.sock", "ps eww -o command= -p 4701"}
			if !slices.Equal(f.lines(), want) {
				t.Fatalf("calls = %q; want %q", f.lines(), want)
			}
			if left := f.calls[0].left; left <= 4*time.Second || left > 5*time.Second {
				t.Errorf("time left on the status read = %v; want it bounded by the 5s of orgHerdrStatusTimeout", left)
			}
			for _, c := range f.calls[1:] {
				if c.left <= 2*time.Second || c.left > 3*time.Second {
					t.Errorf("time left on %q = %v; want it bounded by the 3s of orgHerdrServerEnvCallTimeout", c.line, c.left)
				}
			}
		})
	}
}

// AC6, several pids: ps reads the command of each, the pid whose command is
// herdr (the last element of macOS's path, or Linux's name) is the server's,
// and a command that only ends in herdr does not count.
func TestOrgHerdrServerReadClaudeConfigDir_SeveralPIDsAreNarrowedToHerdr(t *testing.T) {
	for _, tc := range []struct{ name, herdrComm, otherComm string }{
		{"macOS paths", "/opt/homebrew/bin/herdr\n", "/Applications/Ghostty.app/Contents/MacOS/ghostty\n"},
		{"Linux names", "herdr\n", "notherdr\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrServerEnvFake{st: herdrServerEnvRunning, runs: map[string]herdrServerEnvAnswer{
				"lsof -t /iso/herdr.sock":    {out: "812\n4701\n"},
				"ps -o comm= -p 812":         {out: tc.otherComm},
				"ps -o comm= -p 4701":        {out: tc.herdrComm},
				"ps eww -o command= -p 4701": {out: herdrServerPSLine("CLAUDE_CONFIG_DIR=/Users/u/.claude-b")},
			}}

			got := readOrgHerdrServerClaudeConfigDir(context.Background(), f.probe(t))
			checkHerdrServerEnvRead(t, got, orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/Users/u/.claude-b"})
			want := []string{"herdr status server --json", "lsof -t /iso/herdr.sock", "ps -o comm= -p 812", "ps -o comm= -p 4701", "ps eww -o command= -p 4701"}
			if !slices.Equal(f.lines(), want) {
				t.Errorf("calls = %q; want %q", f.lines(), want)
			}
		})
	}
}

// AC6, no pid: a status that cannot be read, a server that does not run, a
// status with no socket, an lsof that fails, lists nothing, or prints what is
// not a pid, a ps of a command that fails, and pids of which none or two
// are herdr (a herdr client holding the socket too) cannot be read, with
// the reason, and the read stops there: no environment is read.
func TestOrgHerdrServerReadClaudeConfigDir_NoPIDCannotBeRead(t *testing.T) {
	statusErr := errors.New("herdr status server: herdr: exit status 1")
	lsofErr := errors.New("lsof: exit status 1: lsof: status error on /iso/herdr.sock: No such file or directory")
	psErr := errors.New("ps: exit status 1")
	twoPIDs := map[string]herdrServerEnvAnswer{"lsof -t /iso/herdr.sock": {out: "4701\n812\n"}}
	with := func(base map[string]herdrServerEnvAnswer, line string, a herdrServerEnvAnswer) map[string]herdrServerEnvAnswer {
		m := map[string]herdrServerEnvAnswer{line: a}
		for k, v := range base {
			m[k] = v
		}
		return m
	}
	for _, tc := range []struct {
		name      string
		st        driver.HerdrServerStatus
		statusErr error
		runs      map[string]herdrServerEnvAnswer
		calls     []string
		reason    string
	}{
		{name: "the status cannot be read", statusErr: statusErr,
			calls:  []string{"herdr status server --json"},
			reason: "could not read the herdr server's status: herdr status server: herdr: exit status 1"},
		{name: "no server runs", st: driver.HerdrServerStatus{Running: false, Socket: "/iso/herdr.sock"},
			calls:  []string{"herdr status server --json"},
			reason: "no herdr server is running"},
		{name: "no socket", st: driver.HerdrServerStatus{Running: true},
			calls:  []string{"herdr status server --json"},
			reason: "'herdr status server --json' reported no socket"},
		{name: "lsof fails", st: herdrServerEnvRunning,
			runs:   map[string]herdrServerEnvAnswer{"lsof -t /iso/herdr.sock": {err: lsofErr}},
			calls:  []string{"herdr status server --json", "lsof -t /iso/herdr.sock"},
			reason: `could not find the herdr server's pid: lsof -t on the socket "/iso/herdr.sock" failed: ` + lsofErr.Error()},
		{name: "lsof lists nothing", st: herdrServerEnvRunning,
			runs:   map[string]herdrServerEnvAnswer{"lsof -t /iso/herdr.sock": {out: "\n"}},
			calls:  []string{"herdr status server --json", "lsof -t /iso/herdr.sock"},
			reason: `could not find the herdr server's pid: lsof -t on the socket "/iso/herdr.sock" listed no process`},
		{name: "lsof prints what is not a pid", st: herdrServerEnvRunning,
			runs:   map[string]herdrServerEnvAnswer{"lsof -t /iso/herdr.sock": {out: "4701\np812\n"}},
			calls:  []string{"herdr status server --json", "lsof -t /iso/herdr.sock"},
			reason: `could not find the herdr server's pid: lsof -t on the socket "/iso/herdr.sock" printed "p812", which is not a pid`},
		{name: "a ps of a command fails", st: herdrServerEnvRunning,
			runs: with(with(twoPIDs, "ps -o comm= -p 4701", herdrServerEnvAnswer{out: "/opt/homebrew/bin/herdr\n"}),
				"ps -o comm= -p 812", herdrServerEnvAnswer{err: psErr}),
			calls: []string{"herdr status server --json", "lsof -t /iso/herdr.sock", "ps -o comm= -p 4701", "ps -o comm= -p 812"},
			reason: `could not find the herdr server's pid: lsof -t on the socket "/iso/herdr.sock" listed pids 4701, 812, ` +
				"and ps -o comm= -p 812 failed: ps: exit status 1"},
		{name: "no pid is herdr", st: herdrServerEnvRunning,
			runs: with(with(twoPIDs, "ps -o comm= -p 4701", herdrServerEnvAnswer{out: "/bin/zsh\n"}),
				"ps -o comm= -p 812", herdrServerEnvAnswer{out: "\n"}),
			calls: []string{"herdr status server --json", "lsof -t /iso/herdr.sock", "ps -o comm= -p 4701", "ps -o comm= -p 812"},
			reason: `could not find the herdr server's pid: lsof -t on the socket "/iso/herdr.sock" listed pids 4701, 812, ` +
				"and 0 of them are herdr, not one"},
		{name: "two pids are herdr", st: herdrServerEnvRunning,
			runs: with(with(twoPIDs, "ps -o comm= -p 4701", herdrServerEnvAnswer{out: "/opt/homebrew/bin/herdr\n"}),
				"ps -o comm= -p 812", herdrServerEnvAnswer{out: "herdr\n"}),
			calls: []string{"herdr status server --json", "lsof -t /iso/herdr.sock", "ps -o comm= -p 4701", "ps -o comm= -p 812"},
			reason: `could not find the herdr server's pid: lsof -t on the socket "/iso/herdr.sock" listed pids 4701, 812, ` +
				"and 2 of them are herdr, not one"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrServerEnvFake{st: tc.st, statusErr: tc.statusErr, runs: tc.runs}

			got := readOrgHerdrServerClaudeConfigDir(context.Background(), f.probe(t))
			checkHerdrServerEnvRead(t, got, orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnreadable, reason: tc.reason})
			if !slices.Equal(f.lines(), tc.calls) {
				t.Errorf("calls = %q; want %q", f.lines(), tc.calls)
			}
		})
	}
}

// AC6, the value ps eww prints (spec FR-7): it runs to the next " KEY=" or
// to the end of the line, so a value with spaces is read whole; an empty
// value is set to ""; the first entry counts; and an environment whose
// entries are only names that contain CLAUDE_CONFIG_DIR, or text where it
// does not start an entry, has none (unset).
func TestOrgHerdrServerReadClaudeConfigDir_ThePSValueRunsToTheNextKey(t *testing.T) {
	set := func(v string) orgHerdrServerClaudeConfigDir {
		return orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: v}
	}
	unset := orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnset}
	for _, tc := range []struct {
		name string
		out  string
		want orgHerdrServerClaudeConfigDir
	}{
		{"a value with spaces followed by another KEY=", herdrServerPSLine("CLAUDE_CONFIG_DIR=/Users/u/Claude B  two"), set("/Users/u/Claude B  two")},
		{"a value with an = that does not start an entry", herdrServerPSLine("CLAUDE_CONFIG_DIR=/a=b,c=d"), set("/a=b,c=d")},
		{"the last entry", "/opt/homebrew/bin/herdr server HOME=/Users/u CLAUDE_CONFIG_DIR=/Users/u/Claude B\n", set("/Users/u/Claude B")},
		{"the last entry, with no newline", "/opt/homebrew/bin/herdr server HOME=/Users/u CLAUDE_CONFIG_DIR=/x y", set("/x y")},
		{"an empty value", herdrServerPSLine("CLAUDE_CONFIG_DIR="), set("")},
		{"the first of two entries", herdrServerPSLine("CLAUDE_CONFIG_DIR=/first CLAUDE_CONFIG_DIR=/second"), set("/first")},
		{"no such entry", herdrServerPSLine("HOME=/Users/u"), unset},
		{"names that contain it", herdrServerPSLine("MY_CLAUDE_CONFIG_DIR=/no CLAUDE_CONFIG_DIRS=/no"), unset},
		{"text where it starts no entry", herdrServerPSLine("X=a,CLAUDE_CONFIG_DIR=/no"), unset},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrServerEnvFake{st: herdrServerEnvRunning, runs: map[string]herdrServerEnvAnswer{
				"lsof -t /iso/herdr.sock":    {out: "4701\n"},
				"ps eww -o command= -p 4701": {out: tc.out},
			}}
			checkHerdrServerEnvRead(t, readOrgHerdrServerClaudeConfigDir(context.Background(), f.probe(t)), tc.want)
		})
	}
}

// AC6, an environment ps cannot give: a ps eww that fails, and one that
// prints the command with no environment (as it does for a process whose
// environment it may not read), cannot be read; neither is unset, and the
// reason holds nothing ps printed.
func TestOrgHerdrServerReadClaudeConfigDir_PSWithoutAnEnvironmentCannotBeRead(t *testing.T) {
	for _, tc := range []struct {
		name   string
		answer herdrServerEnvAnswer
		reason string
	}{
		{"ps fails", herdrServerEnvAnswer{out: herdrServerPSLine("PS_ONLY_ENTRY=from-ps"), err: errors.New("ps: exit status 1")},
			"could not read the environment of the herdr server (pid 4701) with ps: ps: exit status 1"},
		{"ps prints no environment", herdrServerEnvAnswer{out: "/opt/homebrew/bin/herdr server\n"},
			"the herdr server's environment (pid 4701) showed no variable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrServerEnvFake{st: herdrServerEnvRunning, runs: map[string]herdrServerEnvAnswer{
				"lsof -t /iso/herdr.sock":    {out: "4701\n"},
				"ps eww -o command= -p 4701": tc.answer,
			}}
			got := readOrgHerdrServerClaudeConfigDir(context.Background(), f.probe(t))
			checkHerdrServerEnvRead(t, got, orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnreadable, reason: tc.reason})
		})
	}
}

// orgPSEnvValue on what ps eww prints: the value of CLAUDE_CONFIG_DIR ends
// at the next " <word>=", whatever the word, so an entry after it whose name
// is not a C identifier (a hyphen, a dot, bash's BASH_FUNC_x%%, a leading
// digit) never shows in the value; a space with no "=" after the word stays
// in the value, and a value that itself holds " <word>=" is cut there. The
// first entry named CLAUDE_CONFIG_DIR counts, a name that only starts with it
// is another entry, and a line with no entry at all has anyEnv false.
func TestOrgPSEnvValue_TheValueEndsAtTheNextEntryOfAnyName(t *testing.T) {
	const cmd = "/opt/homebrew/bin/herdr server"
	for _, tc := range []struct {
		name          string
		out           string
		value         string
		found, anyEnv bool
	}{
		{"a hyphen in the next name", cmd + " PATH=/usr/bin CLAUDE_CONFIG_DIR=/Users/x/.claude-b x-y=SECRET1 PATH=/bin\n",
			"/Users/x/.claude-b", true, true},
		{"a dot in the next name", cmd + " CLAUDE_CONFIG_DIR=/Users/x/.claude-b foo.bar=SECRET2 HOME=/Users/x\n",
			"/Users/x/.claude-b", true, true},
		{"a bash function after it", cmd + " CLAUDE_CONFIG_DIR=/Users/x/.claude-b BASH_FUNC_ml%%=() { SECRET3; } PATH=/bin\n",
			"/Users/x/.claude-b", true, true},
		{"a next name that starts with a digit", cmd + " CLAUDE_CONFIG_DIR=/Users/x/.claude-b 9LIVES=SECRET4\n",
			"/Users/x/.claude-b", true, true},
		{"an empty value before a name with a hyphen", cmd + " CLAUDE_CONFIG_DIR= x-y=SECRET5\n", "", true, true},
		{"a value with a space and no = after it", cmd + " CLAUDE_CONFIG_DIR=/Users/x/My Dir HOME=/Users/x\n",
			"/Users/x/My Dir", true, true},
		{"the last entry", cmd + " HOME=/Users/x CLAUDE_CONFIG_DIR=/Users/x/My Dir\n", "/Users/x/My Dir", true, true},
		{"a value that holds a space and a word with =", cmd + " CLAUDE_CONFIG_DIR=/Users/x/a b=c HOME=/Users/x\n",
			"/Users/x/a", true, true},
		{"no entry named it", cmd + " PATH=/bin x-y=SECRET6 HOME=/Users/x\n", "", false, true},
		{"no environment at all", cmd + "\n", "", false, false},
		{"the first of two entries", cmd + " CLAUDE_CONFIG_DIR=/first x-y=SECRET7 CLAUDE_CONFIG_DIR=/second\n", "/first", true, true},
		{"a name that only starts with it", cmd + " CLAUDE_CONFIG_DIRX=/no HOME=/Users/x\n", "", false, true},
		{"a name that only starts with it, then the name", cmd + " CLAUDE_CONFIG_DIRX=/no CLAUDE_CONFIG_DIR=/yes\n", "/yes", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, found, anyEnv := orgPSEnvValue(tc.out, orgClaudeConfigDirName)
			if value != tc.value || found != tc.found || anyEnv != tc.anyEnv {
				t.Errorf("orgPSEnvValue = (%q, %v, %v); want (%q, %v, %v)", value, found, anyEnv, tc.value, tc.found, tc.anyEnv)
			}
			if strings.Contains(value, "SECRET") {
				t.Errorf("orgPSEnvValue = %q, which holds the value of another entry", value)
			}
		})
	}
}

// AC6 on Linux: the environment is /proc/<pid>/environ, NUL-separated, read
// instead of ps eww: a value with spaces whole, an entry whose name only
// contains CLAUDE_CONFIG_DIR not counted, none (unset), an empty file and a
// read that fails (cannot be read). lsof still finds the pid.
func TestOrgHerdrServerReadClaudeConfigDir_LinuxReadsProcEnviron(t *testing.T) {
	readErr := &fs.PathError{Op: "open", Path: "/proc/4701/environ", Err: fs.ErrPermission}
	for _, tc := range []struct {
		name   string
		answer herdrServerEnvAnswer
		want   orgHerdrServerClaudeConfigDir
	}{
		{"a value with spaces", herdrServerEnvAnswer{out: "PATH=/usr/bin\x00MY_CLAUDE_CONFIG_DIR=/no\x00CLAUDE_CONFIG_DIR=/home/u/claude b\x00HOME=/home/u\x00"},
			orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/home/u/claude b"}},
		{"the last entry with no NUL after it", herdrServerEnvAnswer{out: "PATH=/usr/bin\x00CLAUDE_CONFIG_DIR=/home/u/c"},
			orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/home/u/c"}},
		{"no such entry", herdrServerEnvAnswer{out: "PATH=/usr/bin\x00MY_CLAUDE_CONFIG_DIR=/no\x00HOME=/home/u\x00"},
			orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnset}},
		{"an empty file", herdrServerEnvAnswer{out: ""},
			orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnreadable, reason: "the herdr server's environment (pid 4701) showed no variable"}},
		{"a read that fails", herdrServerEnvAnswer{err: readErr},
			orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnreadable,
				reason: "could not read the environment of the herdr server (pid 4701): open /proc/4701/environ: permission denied"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &herdrServerEnvFake{st: herdrServerEnvRunning, goos: "linux",
				runs:  map[string]herdrServerEnvAnswer{"lsof -t /iso/herdr.sock": {out: "4701\n"}},
				files: map[string]herdrServerEnvAnswer{"/proc/4701/environ": tc.answer}}

			checkHerdrServerEnvRead(t, readOrgHerdrServerClaudeConfigDir(context.Background(), f.probe(t)), tc.want)
			if want := []string{"herdr status server --json", "lsof -t /iso/herdr.sock", "read /proc/4701/environ"}; !slices.Equal(f.lines(), want) {
				t.Errorf("calls = %q; want %q", f.lines(), want)
			}
		})
	}
}

// The zero result is one that cannot be read, never unset.
func TestOrgHerdrServerReadClaudeConfigDir_TheZeroResultCannotBeRead(t *testing.T) {
	if got := (orgHerdrServerClaudeConfigDir{}).state; got != orgHerdrServerEnvUnreadable {
		t.Errorf("state of the zero result = %v; want orgHerdrServerEnvUnreadable", got)
	}
}

// writeHerdrServerEnvCommand writes an executable sh script named name with
// body into a new directory at the front of PATH, for the rest of the test.
func writeHerdrServerEnvCommand(t *testing.T, name, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// NFR-3 and the environment of lsof and ps: the production runner runs a
// command with PATH and LC_ALL only (no HOME, no PS_FORMAT or locale of the
// caller's) and returns its stdout as it printed it. LC_ALL is C for lsof,
// ps -o comm=, and any other command, and en_US.UTF-8 for ps eww, whose
// output holds the value the read returns.
func TestOrgHerdrServerEnvCommand_RunsWithPATHAndLCALLOnly(t *testing.T) {
	t.Setenv("PS_FORMAT", "pid")
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	probe := `printf '%s|%s|%s|%s|%s\n' "$*" "$LC_ALL" "${HOME-unset}" "${PS_FORMAT-unset}" "$PATH"` + "\n"
	for _, name := range []string{"lsof", "ps", "ralph-env-probe"} {
		writeHerdrServerEnvCommand(t, name, probe)
	}
	for _, tc := range []struct {
		command []string
		locale  string
	}{
		{[]string{"lsof", "-t", "/iso/herdr.sock"}, "C"},
		{[]string{"ps", "-o", "comm=", "-p", "4701"}, "C"},
		{[]string{"ps", "eww", "-o", "command=", "-p", "4701"}, "en_US.UTF-8"},
		{[]string{"ps", "-o", "command=", "eww"}, "C"},
		{[]string{"ps"}, "C"},
		{[]string{"ralph-env-probe", "eww"}, "C"},
	} {
		name, args := tc.command[0], tc.command[1:]
		want := []string{"PATH=" + os.Getenv("PATH"), "LC_ALL=" + tc.locale}
		if got := orgHerdrServerEnvCommandEnv(name, args...); !slices.Equal(got, want) {
			t.Errorf("orgHerdrServerEnvCommandEnv(%q) = %q; want %q", tc.command, got, want)
		}
		out, err := runOrgHerdrServerEnvCommand(context.Background(), name, args...)
		if want := strings.Join(args, " ") + "|" + tc.locale + "|unset|unset|" + os.Getenv("PATH") + "\n"; err != nil || out != want {
			t.Errorf("runOrgHerdrServerEnvCommand(%q) = %q, %v; want %q", tc.command, out, err, want)
		}
	}
}

// AC6, a non-ASCII value: through the production runner, the read runs lsof
// under C and ps eww under en_US.UTF-8, so a non-ASCII CLAUDE_CONFIG_DIR
// reads as the path it is. The fake ps answers as macOS ps does: the value's
// bytes under en_US.UTF-8, escapes under C.
func TestOrgHerdrServerReadClaudeConfigDir_PSReadsANonASCIIValueUnderUTF8(t *testing.T) {
	const value = "/Users/x/日本語 dir/.claude"
	raw := herdrServerPSLine("CLAUDE_CONFIG_DIR=" + value)
	escaped := herdrServerPSLine(`CLAUDE_CONFIG_DIR=/Users/x/M-fM^WM-%M-fM^\M-,M-hM-*M^^ dir/.claude`)
	writeHerdrServerEnvCommand(t, "lsof", "[ \"$LC_ALL $*\" = 'C -t /iso/herdr.sock' ] || exit 3\necho 4701\n")
	writeHerdrServerEnvCommand(t, "ps", "case \"$LC_ALL $*\" in\n"+
		"'en_US.UTF-8 eww -o command= -p 4701') printf %s '"+raw+"' ;;\n"+
		"'C eww -o command= -p 4701') printf %s '"+escaped+"' ;;\n"+
		"*) exit 2 ;;\nesac\n")
	p := orgHerdrServerEnvProbe{
		status: func(context.Context) (driver.HerdrServerStatus, error) { return herdrServerEnvRunning, nil },
		run:    runOrgHerdrServerEnvCommand,
		goos:   "darwin",
	}

	got := readOrgHerdrServerClaudeConfigDir(context.Background(), p)
	checkHerdrServerEnvRead(t, got, orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: value})
}

// NFR-3: a command that fails is an error with its exit and its stderr
// folded to one line, and never its stdout; one that does not answer ends
// when the context does, with an error that says so and wraps the cause.
func TestOrgHerdrServerEnvCommand_FailuresAndTheDeadline(t *testing.T) {
	t.Run("a failure", func(t *testing.T) {
		writeHerdrServerEnvCommand(t, "ralph-env-probe",
			"echo 'CLAUDE_CONFIG_DIR=/leak'\nprintf 'lsof: status error\\n  on /iso/herdr.sock\\n' >&2\nexit 1\n")
		out, err := runOrgHerdrServerEnvCommand(context.Background(), "ralph-env-probe")
		if want := "ralph-env-probe: exit status 1: lsof: status error on /iso/herdr.sock"; err == nil || err.Error() != want || out != "" {
			t.Errorf("runOrgHerdrServerEnvCommand = %q, %v; want no output and %q", out, err, want)
		}
	})
	t.Run("the deadline", func(t *testing.T) {
		writeHerdrServerEnvCommand(t, "ralph-env-probe", "exec sleep 5\n")
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		start := time.Now()
		out, err := runOrgHerdrServerEnvCommand(ctx, "ralph-env-probe")
		if err == nil || !strings.HasPrefix(err.Error(), "ralph-env-probe did not answer: ") || !errors.Is(err, context.DeadlineExceeded) || out != "" {
			t.Errorf("runOrgHerdrServerEnvCommand = %q, %v; want no output and an error that it did not answer", out, err)
		}
		if took := time.Since(start); took > 3*time.Second {
			t.Errorf("the run took %v; want it to end soon after the 200ms deadline", took)
		}
	})
}
