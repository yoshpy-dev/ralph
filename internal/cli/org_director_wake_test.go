package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for how a director is woken, through the CLI (plan
// docs/plans/active/2026-10-18-org-director-wake.md, D3, D4, D9, AC2, AC3,
// and AC4 for attach and start): the wake and the wake target attach and
// start record, the attach and start refusals of a wake that cannot be
// sent, the main worktree's ralph.toml as the source of wake and wake_cmd,
// and where status says the wake line goes. Every ledger is under
// t.TempDir() (named with --state-dir, or found in a temp repository by
// newLegacyLedgerRepo). start runs the stub herdr or a launch_cmd that only
// creates a file, and no test runs herdr, a terminal, or osascript; nothing
// runs a wake_cmd here (the watch does, in stage 8b-1's S3).

// The wake_cmd values the tests write: one whose {target} needs a
// --wake-target, and one that needs none. Nothing runs them.
const (
	directorWakeCmdWithTarget    = `["notify", "{target}", "{text}"]`
	directorWakeCmdWithoutTarget = `["notify", "{text}"]`
)

// directorWakeToml is a ralph.toml whose [org.director] has wake and
// wake_cmd set to the TOML arrays given, each left out when "".
func directorWakeToml(wake, wakeCmd string) string {
	s := "[org.director]\n"
	if wake != "" {
		s += "wake = " + wake + "\n"
	}
	if wakeCmd != "" {
		s += "wake_cmd = " + wakeCmd + "\n"
	}
	return s
}

// directorAttachedTail is what follows the parentheses of an attached
// director: line with [org.director].auto_merge false.
const directorAttachedTail = " -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline" +
	directorAutoMergeOff

// directorWakeCmdNeededText is the reason of ErrDirectorWakeCmdNeeded at
// attach, after "(wake <wake>)".
const directorWakeCmdNeededText = ": a session attached with ralph org director attach has no default way to be sent the wake line " +
	"(only a headless director in herdr has one, its herdr pane), so ralph org watch --all could not wake it; " +
	"set [org.director].wake_cmd, or attach with --wake pull"

// AC2 and D4: attach refuses, with exit 1, nothing on stdout, and
// director.jsonl as it was, a wake that has push while wake_cmd is empty
// (from --wake or from [org.director].wake), a wake_cmd with {target} and no
// --wake-target, a --wake-target with a wake that has no push, a
// --wake-target that is empty, has a control character, is not UTF-8, is
// longer than 256 characters, or starts with "-", and a --wake with a value
// other than pull and push, an empty part, or one twice. A bad flag is
// refused before any ralph.toml is read, so a file that does not load does
// not hide it. With --replace over d1 the refusal ends nothing: d1 stays the
// current registration. The refusals that need wake_cmd name the file it
// was read from.
func TestOrgDirectorAttach_RefusesAWakeItCannotSendWithNothingWritten(t *testing.T) {
	const broken = "[org]\nmax_orgs = 0\n"
	readFrom := func(what string) func(string) string {
		return func(configPath string) string { return "; " + what + " the --config file " + configPath }
	}
	cmdRead, bothRead := readFrom("[org.director].wake_cmd is read from"), readFrom("[org.director].wake and wake_cmd are read from")
	none := func(string) string { return "" }
	targetErr := "org: director attach: --wake-target: the wake target is not usable: "
	wakeErr := "org: director attach: --wake: "
	cases := []struct {
		name string
		toml string   // the --config file's content
		args []string // attach's wake flags
		is   error    // a sentinel the error wraps (nil: none)
		want string   // the error, before the file it names
		file func(configPath string) string
	}{
		{"--wake push with no wake_cmd", "", []string{"--wake", "push"}, org.ErrDirectorWakeCmdNeeded,
			"org: director attach: the wake has push, but [org.director].wake_cmd is empty (wake push)" + directorWakeCmdNeededText, cmdRead},
		{"the config's pull,push with no wake_cmd", directorWakeToml(`["pull", "push"]`, ""), nil, org.ErrDirectorWakeCmdNeeded,
			"org: director attach: the wake has push, but [org.director].wake_cmd is empty (wake pull,push)" + directorWakeCmdNeededText, bothRead},
		{"a wake_cmd with {target} and no --wake-target", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push"},
			org.ErrDirectorWakeTargetNeeded,
			"org: director attach: [org.director].wake_cmd has {target}, but no --wake-target was given (wake push): " +
				"pass --wake-target with where [org.director].wake_cmd sends the wake line", cmdRead},
		{"the config's push, a wake_cmd with {target}, and no --wake-target", directorWakeToml(`["push"]`, directorWakeCmdWithTarget), nil,
			org.ErrDirectorWakeTargetNeeded,
			"org: director attach: [org.director].wake_cmd has {target}, but no --wake-target was given (wake push): " +
				"pass --wake-target with where [org.director].wake_cmd sends the wake line", bothRead},
		{"--wake pull with a --wake-target", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "pull", "--wake-target", "term-1"},
			org.ErrDirectorWakeTargetWithoutPush,
			"org: director attach: --wake-target is given, but the wake has no push (wake pull): the watch sends to the wake target only to push; " +
				"drop --wake-target, or attach with --wake push or --wake pull,push", cmdRead},
		{"the config's pull with a --wake-target", "", []string{"--wake-target", "term-1"}, org.ErrDirectorWakeTargetWithoutPush,
			"org: director attach: --wake-target is given, but the wake has no push (wake pull): the watch sends to the wake target only to push; " +
				"drop --wake-target, or attach with --wake push or --wake pull,push", bothRead},
		{"an empty --wake-target", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push", "--wake-target", ""},
			org.ErrDirectorWakeTargetInvalid, targetErr + "it is empty", none},
		{"a --wake-target with a bell", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push", "--wake-target", "a\x07b"},
			org.ErrDirectorWakeTargetInvalid, targetErr + `"a\ab" has a control character`, none},
		{"a --wake-target with a line break", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push", "--wake-target", "a\nb"},
			org.ErrDirectorWakeTargetInvalid, targetErr + `"a\nb" has a control character`, none},
		{"a --wake-target that is not UTF-8", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push", "--wake-target", "a\xffb"},
			org.ErrDirectorWakeTargetInvalid, targetErr + `"a\xffb" is not valid UTF-8`, none},
		{"a --wake-target of 257 characters", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push", "--wake-target", strings.Repeat("é", 257)},
			org.ErrDirectorWakeTargetInvalid, targetErr + "it is 257 characters long, more than 256", none},
		{"a --wake-target that starts with -", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push", "--wake-target", "-x"},
			org.ErrDirectorWakeTargetInvalid, targetErr + `"-x" starts with "-", which the command [org.director].wake_cmd runs could read as an option`, none},
		{"--wake poll", "", []string{"--wake", "poll"}, nil, wakeErr + `element 0 "poll" is not a wake value; the values are "pull" and "push"`, none},
		{"--wake pull,pull", "", []string{"--wake", "pull,pull"}, nil, wakeErr + `element 1 "pull" repeats element 0; list each value once`, none},
		{"an empty --wake", "", []string{"--wake", ""}, nil, wakeErr + `element 0 "" is not a wake value; the values are "pull" and "push"`, none},
		{"--wake with a space after the comma", "", []string{"--wake", "pull, push"}, nil,
			wakeErr + `element 1 " push" is not a wake value; the values are "pull" and "push"`, none},
		{"a bad --wake before a ralph.toml that does not load", broken, []string{"--wake", "push,push"}, nil,
			wakeErr + `element 1 "push" repeats element 0; list each value once`, none},
		{"a bad --wake-target before a ralph.toml that does not load", broken, []string{"--wake", "push", "--wake-target", "-x"},
			org.ErrDirectorWakeTargetInvalid, targetErr + `"-x" starts with "-", which the command [org.director].wake_cmd runs could read as an option`, none},
	}
	for _, tc := range cases {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s, replace %v", tc.name, replace), func(t *testing.T) {
				setEscalateWatch(t, acceptingWatchLiveness())
				t.Chdir(t.TempDir())
				stateDir := t.TempDir()
				configPath := writeDirectorConfig(t, tc.toml)
				args := []string{"director", "attach", "--driver", "claude", "--state-dir", stateDir, "--config", configPath}
				if replace {
					runInboxOK(t, "director", "attach", "--driver", "codex", "--state-dir", stateDir)
					args = append(args, "--replace")
				}
				before := directorFileOrAbsent(t, stateDir)

				stdout, _, err := runOrgCmdStreams(t, append(args, tc.args...)...)
				if want := tc.want + tc.file(configPath); err == nil || err.Error() != want {
					t.Fatalf("attach error =\n%v\nwant\n%s", err, want)
				}
				if tc.is != nil && !errors.Is(err, tc.is) {
					t.Errorf("attach error = %v; want it to wrap %v", err, tc.is)
				}
				if stdout != "" {
					t.Errorf("refused attach stdout = %q; want nothing", stdout)
				}
				if after := directorFileOrAbsent(t, stateDir); after != before {
					t.Errorf("a refused attach changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
				}
				if replace {
					if got := directorStatusLine(t, "--state-dir", stateDir); !strings.HasPrefix(got, "director: attached d1 (driver=codex, ") {
						t.Errorf("director line after a refused --replace = %q; want d1 still attached", got)
					}
				}
			})
		}
	}
}

// AC2: attach records the --wake it was given, else [org.director].wake, in
// the attached line's wake, and the --wake-target in its wake_target (left
// out of the line when not given). The success line is that of stage 8a for
// a wake without push and adds wake= and, with a target, wake_target= when
// the wake has push. A target of 256 characters (multibyte ones included)
// passes, and a pull --wake needs no wake_cmd whatever the config's wake
// says.
func TestOrgDirectorAttach_RecordsTheWakeAndTheWakeTarget(t *testing.T) {
	long := strings.Repeat("é", 255) + "x"
	cases := []struct {
		name       string
		toml       string
		args       []string
		wantWake   []string
		wantTarget string
		wantLine   string
	}{
		{"--wake push and a --wake-target", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push", "--wake-target", "term 1"},
			[]string{"push"}, "term 1", "attached d1 (driver=claude, wake=push, wake_target=term 1)"},
		{"--wake push,pull and a target of 256 characters", directorWakeToml("", directorWakeCmdWithTarget), []string{"--wake", "push,pull", "--wake-target", long},
			[]string{"push", "pull"}, long, "attached d1 (driver=claude, wake=push,pull, wake_target=" + long + ")"},
		{"--wake push with a wake_cmd that needs no target", directorWakeToml("", directorWakeCmdWithoutTarget), []string{"--wake", "push"},
			[]string{"push"}, "", "attached d1 (driver=claude, wake=push)"},
		{"no --wake: the config's pull,push", directorWakeToml(`["pull", "push"]`, directorWakeCmdWithoutTarget), nil,
			[]string{"pull", "push"}, "", "attached d1 (driver=claude, wake=pull,push)"},
		{"no --wake: the config's push with a target", directorWakeToml(`["push"]`, directorWakeCmdWithTarget), []string{"--wake-target", "term-2"},
			[]string{"push"}, "term-2", "attached d1 (driver=claude, wake=push, wake_target=term-2)"},
		{"--wake pull over the config's push with no wake_cmd", directorWakeToml(`["push"]`, ""), []string{"--wake", "pull"},
			[]string{"pull"}, "", "attached d1 (driver=claude)"},
		{"no --wake and no wake in the config", "", nil, []string{"pull"}, "", "attached d1 (driver=claude)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEscalateWatch(t, acceptingWatchLiveness())
			t.Chdir(t.TempDir())
			stateDir := t.TempDir()
			configPath := writeDirectorConfig(t, tc.toml)

			stdout, _ := runInboxOK(t, append([]string{"director", "attach", "--driver", "claude", "--state-dir", stateDir, "--config", configPath}, tc.args...)...)
			if want := tc.wantLine + "\ndetach it with: ralph org director detach --id d1 --state-dir '" + stateDir + "'\n"; stdout != want {
				t.Errorf("attach stdout =\n%q\nwant\n%q", stdout, want)
			}
			events := directorEventsIn(t, stateDir)
			if len(events) != 1 || events[0].Event != org.DirectorEventAttached || !slices.Equal(events[0].Wake, tc.wantWake) || events[0].WakeTarget != tc.wantTarget {
				t.Fatalf("director.jsonl = %+v; want one attached line with wake %v and wake_target %q", events, tc.wantWake, tc.wantTarget)
			}
			if raw := directorFileOrAbsent(t, stateDir); tc.wantTarget == "" && strings.Contains(raw, "wake_target") {
				t.Errorf("director.jsonl names wake_target with no --wake-target:\n%s", raw)
			}
		})
	}
}

// AC2: the replaced part of attach's success line shows the replaced
// registration's wake like the new one's, escaped, since a human can edit
// director.jsonl: a hand-edited d1 with a control character in its wake and
// its wake target prints them escaped on the one line.
func TestOrgDirectorAttach_TheReplacedPartShowsTheWakeEscaped(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	t.Chdir(t.TempDir())
	stateDir := t.TempDir()
	const ts = "2026-10-18T09:00:00Z"
	writeDirectorLines(t, stateDir, directorEventLine(t, org.DirectorEvent{
		TS: ts, ID: "d1", Event: org.DirectorEventAttached, Driver: "claude", Wake: []string{"push", "bell\a"}, WakeTarget: "t\x1b[2J\ndirector: forged",
	}))

	stdout, _ := runInboxOK(t, "director", "attach", "--driver", "codex", "--replace", "--wake", "pull", "--state-dir", stateDir)
	line, _, _ := strings.Cut(stdout, "\n")
	want := `attached d2 (driver=codex), replacing d1 (driver=claude, attached at ` + ts + `, wake=push,bell\a, wake_target=t\x1b[2J\ndirector: forged)`
	if line != want {
		t.Errorf("attach --replace line:\n got %q\nwant %q", line, want)
	}
	if strings.ContainsAny(stdout, "\a\x1b") || strings.Count(stdout, "\n") != 2 {
		t.Errorf("attach --replace stdout holds a raw control character or a forged line: %q", stdout)
	}
}

// AC4 and D9 for attach: without --config, on the main worktree's ledger
// found from a linked worktree, attach takes [org.director].wake and
// wake_cmd from the main worktree's ralph.toml, never the linked
// worktree's, both for the wake it records and for the wake_cmd its check
// needs, and a refusal names the main worktree's file. --config names the
// file to read instead, and with a --state-dir naming another ledger the
// values come from ./ralph.toml.
func TestOrgDirectorAttach_ReadsTheWakeFromTheMainWorktree(t *testing.T) {
	cases := []struct {
		name             string
		mainToml, wtToml string // "" writes no file
		// extra returns attach's extra flags and the ledger it writes ("" for
		// the shared one).
		extra    func(t *testing.T, r legacyLedgerRepo) (args []string, ledger string)
		wantWake []string                        // nil: refused
		is       error                           // the refusal's sentinel
		wantErr  func(r legacyLedgerRepo) string // the whole refusal
	}{
		{name: "no --config: the main worktree's wake and wake_cmd",
			mainToml: directorWakeToml(`["push"]`, directorWakeCmdWithoutTarget), wtToml: directorWakeToml(`["pull"]`, ""),
			extra:    func(*testing.T, legacyLedgerRepo) ([]string, string) { return nil, "" },
			wantWake: []string{"push"}},
		{name: "no --config: no main ralph.toml, the worktree pushes",
			wtToml:   directorWakeToml(`["push"]`, directorWakeCmdWithoutTarget),
			extra:    func(*testing.T, legacyLedgerRepo) ([]string, string) { return nil, "" },
			wantWake: []string{"pull"}},
		{name: "no --config: --wake push and the main worktree's empty wake_cmd",
			mainToml: directorWakeToml(`["pull"]`, ""), wtToml: directorWakeToml(`["push"]`, directorWakeCmdWithoutTarget),
			extra: func(*testing.T, legacyLedgerRepo) ([]string, string) { return []string{"--wake", "push"}, "" },
			is:    org.ErrDirectorWakeCmdNeeded,
			wantErr: func(r legacyLedgerRepo) string {
				return "org: director attach: the wake has push, but [org.director].wake_cmd is empty (wake push)" + directorWakeCmdNeededText +
					"; [org.director].wake_cmd is read from the main worktree's " + filepath.Join(r.mainRoot, "ralph.toml")
			}},
		{name: "no --config: the main worktree's {target} and no --wake-target",
			mainToml: directorWakeToml(`["push"]`, directorWakeCmdWithTarget), wtToml: directorWakeToml(`["push"]`, directorWakeCmdWithoutTarget),
			extra: func(*testing.T, legacyLedgerRepo) ([]string, string) { return nil, "" },
			is:    org.ErrDirectorWakeTargetNeeded,
			wantErr: func(r legacyLedgerRepo) string {
				return "org: director attach: [org.director].wake_cmd has {target}, but no --wake-target was given (wake push): " +
					"pass --wake-target with where [org.director].wake_cmd sends the wake line; " +
					"[org.director].wake and wake_cmd are read from the main worktree's " + filepath.Join(r.mainRoot, "ralph.toml")
			}},
		{name: "--config names the worktree's ralph.toml",
			mainToml: directorWakeToml(`["pull"]`, ""), wtToml: directorWakeToml(`["push"]`, directorWakeCmdWithoutTarget),
			extra:    func(*testing.T, legacyLedgerRepo) ([]string, string) { return []string{"--config", "ralph.toml"}, "" },
			wantWake: []string{"push"}},
		{name: "--state-dir names another ledger: ./ralph.toml",
			mainToml: directorWakeToml(`["push"]`, ""), wtToml: directorWakeToml(`["pull", "push"]`, directorWakeCmdWithoutTarget),
			extra: func(t *testing.T, _ legacyLedgerRepo) ([]string, string) {
				other := filepath.Join(t.TempDir(), "org")
				return []string{"--state-dir", other}, other
			},
			wantWake: []string{"pull", "push"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEscalateWatch(t, acceptingWatchLiveness())
			r := newLegacyLedgerRepo(t)
			if tc.mainToml != "" {
				writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), tc.mainToml)
			}
			if tc.wtToml != "" {
				writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), tc.wtToml)
			}
			extra, ledger := tc.extra(t, r)
			if ledger == "" {
				ledger = r.sharedDir
			}

			stdout, _, err := runOrgCmdStreams(t, append([]string{"director", "attach", "--driver", "claude"}, extra...)...)
			if tc.wantWake == nil {
				if want := tc.wantErr(r); err == nil || err.Error() != want {
					t.Fatalf("attach error =\n%v\nwant\n%s", err, want)
				}
				if !errors.Is(err, tc.is) {
					t.Errorf("attach error = %v; want it to wrap %v", err, tc.is)
				}
				if stdout != "" {
					t.Errorf("refused attach stdout = %q; want nothing", stdout)
				}
				if got := directorFileOrAbsent(t, ledger); got != "<absent>" {
					t.Errorf("a refused attach wrote director.jsonl in %s: %s", ledger, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("attach: %v (stdout %s)", err, stdout)
			}
			events := directorEventsIn(t, ledger)
			if len(events) != 1 || !slices.Equal(events[0].Wake, tc.wantWake) {
				t.Errorf("director.jsonl in %s = %+v; want d1 with wake %v", ledger, events, tc.wantWake)
			}
		})
	}
}

// withMainWorktreeDirectorWakeConfig takes Director.Wake and
// Director.WakeCmd, and nothing else, from the main worktree's ralph.toml
// when --config is not given and the ledger is the main worktree's;
// otherwise it returns cfg as it is. A main worktree ralph.toml that does
// not load is an error naming the file and the two keys. attach loads that
// file for enabled first, so the CLI tests cannot reach this error; it is
// tested here.
func TestWithMainWorktreeDirectorWakeConfig(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	mainToml := filepath.Join(r.mainRoot, "ralph.toml")
	shared, sharedSource := org.ResolveOrgStateDir("", false)
	if shared != r.sharedDir {
		t.Fatalf("the resolved ledger = %s; want the main worktree's %s", shared, r.sharedDir)
	}
	in := config.OrgConfig{MaxOrgs: 3, Director: config.OrgDirectorConfig{
		Enabled: true, AutoMerge: true, LaunchCmd: []string{"open", "{script}"}, Wake: []string{"pull"},
	}}

	writeTestFile(t, mainToml, "[org.director]\nenabled = false\nwake = [\"pull\", \"push\"]\nwake_cmd = [\"notify\", \"{target}\", \"{text}\"]\n")
	got, err := withMainWorktreeDirectorWakeConfig(in, "", shared, sharedSource)
	if err != nil {
		t.Fatalf("withMainWorktreeDirectorWakeConfig: %v", err)
	}
	want := in
	want.Director.Wake = []string{"pull", "push"}
	want.Director.WakeCmd = []string{"notify", "{target}", "{text}"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("withMainWorktreeDirectorWakeConfig =\n%+v\nwant only wake and wake_cmd taken\n%+v", got, want)
	}

	other, otherSource := org.ResolveOrgStateDir(filepath.Join(t.TempDir(), "org"), true)
	for _, c := range []struct {
		name, configPath, stateDir, source string
	}{
		{"--config", "ralph.toml", shared, sharedSource},
		{"another ledger", "", other, otherSource},
	} {
		if got, err := withMainWorktreeDirectorWakeConfig(in, c.configPath, c.stateDir, c.source); err != nil || !reflect.DeepEqual(got, in) {
			t.Errorf("%s: withMainWorktreeDirectorWakeConfig = %+v, %v; want cfg unchanged", c.name, got, err)
		}
	}

	writeTestFile(t, mainToml, "[org]\nmax_orgs = 0\n")
	_, err = withMainWorktreeDirectorWakeConfig(in, "", shared, sharedSource)
	if err == nil || !strings.HasPrefix(err.Error(), "org: load [org.director].wake and wake_cmd from the main worktree's "+mainToml+": ") ||
		!strings.Contains(err.Error(), "[org].max_orgs must be >= 1, got 0") {
		t.Errorf("withMainWorktreeDirectorWakeConfig on a broken main ralph.toml = %v; want the load error naming the file", err)
	}
}

// AC3 through the CLI: a herdr start records [org.director].wake in the
// attached line, and a wake with push passes with an empty wake_cmd there,
// since the watch types the wake line into the director's pane; status then
// says the wake goes to that pane, on the director: line of `ralph org
// director status` and of `ralph status`, while the fields show the wake
// and no wake_target line, and the JSON has no wake_target.
func TestOrgDirectorStart_RecordsTheWakeAndPushesInHerdrWithNoWakeCmd(t *testing.T) {
	stateDir, _, _ := directorStartLedger(t)
	t.Setenv("USER", "alice")
	configPath := writeDirectorConfig(t, directorWakeToml(`["pull", "push"]`, ""))

	stdout, _ := runInboxOK(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", configPath)
	if !strings.HasPrefix(stdout, "started d1 (driver=claude, model=opus, permission=autonomous, pane=pane-stub-1, workspace=ws-stub-1)\n") {
		t.Errorf("start stdout = %q; want d1 started in pane-stub-1", stdout)
	}
	events := directorEventsIn(t, stateDir)
	if len(events) != 4 || !slices.Equal(events[0].Wake, []string{"pull", "push"}) || events[0].WakeTarget != "" {
		t.Fatalf("director.jsonl = %+v; want attached d1 with wake pull,push and no wake_target, then three more", events)
	}
	wantLine := "director: attached d1 (driver=claude, headless running, model=opus, pane pane-stub-1, workspace ws-stub-1, attached at " + events[0].TS +
		", by alice, wake=pull,push to pane-stub-1)" + directorAttachedTail

	statusOut, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
	if line, _, _ := strings.Cut(statusOut, "\n"); line != wantLine {
		t.Errorf("director line:\n got %q\nwant %q", line, wantLine)
	}
	if !strings.Contains(statusOut, "\nwake: pull,push\nattached_at: ") || strings.Contains(statusOut, "wake_target:") ||
		!strings.Contains(statusOut, "\npane_id: pane-stub-1\n") {
		t.Errorf("director status =\n%s\nwant wake: pull,push, no wake_target:, and pane_id: pane-stub-1", statusOut)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if reg := directorReportOf(t, jsonOut).Registration; reg == nil || !slices.Equal(reg.Wake, []string{"pull", "push"}) || reg.WakeTarget != "" ||
		strings.Contains(jsonOut, "wake_target") {
		t.Errorf("director status --json = %s; want wake pull,push and no wake_target", jsonOut)
	}
	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("ralph status: %v (output %s)", err, out)
	}
	if got := statusDirectorLine(t, out); got != wantLine {
		t.Errorf("ralph status director line:\n got %q\nwant %q", got, wantLine)
	}
}

// AC3 and D4 through the CLI: a start with [org.director].launch_cmd whose
// wake has push and whose wake_cmd is empty is refused, with exit 1,
// nothing on stdout or stderr, nothing in the ledger, no agent lookup, no
// launch_cmd run, and no herdr call. With a wake_cmd, or a pull-only wake,
// the same start goes on to run launch_cmd (here one that fails at once)
// and records the config's wake.
func TestOrgDirectorStart_ALaunchCmdThatPushesNeedsAWakeCmd(t *testing.T) {
	for _, tc := range []struct {
		name, wake, wakeCmd string
		refused             bool
	}{
		{"push, no wake_cmd", `["push"]`, "", true},
		{"pull,push, no wake_cmd", `["pull", "push"]`, "", true},
		{"push with a wake_cmd", `["push"]`, directorWakeCmdWithTarget, false},
		{"pull, no wake_cmd", `["pull"]`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			fakes := setDirectorLaunchSeams(t)
			ran := filepath.Join(t.TempDir(), "launch_cmd ran")
			toml := directorTouchingLaunchCmdToml(ran) + "wake = " + tc.wake + "\n"
			if tc.wakeCmd != "" {
				toml += "wake_cmd = " + tc.wakeCmd + "\n"
			}
			configPath := writeDirectorConfig(t, toml)
			wantWake := strings.Split(strings.NewReplacer(`[`, "", `]`, "", `"`, "", " ", "").Replace(tc.wake), ",")

			stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir, "--config", configPath)
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if !tc.refused {
				if err == nil || !strings.Contains(err.Error(), "launch_cmd sh failed: exit status 3; d1 is recorded stopped") {
					t.Fatalf("director start = %v; want it to get past the wake check to launch_cmd's failure", err)
				}
				if events := directorEventsIn(t, stateDir); len(events) == 0 || !slices.Equal(events[0].Wake, wantWake) {
					t.Errorf("director.jsonl = %+v; want attached d1 with wake %v", events, wantWake)
				}
				if _, err := os.Stat(ran); err != nil {
					t.Errorf("launch_cmd did not run: %v", err)
				}
				return
			}
			want := "org: director: start: the wake has push, but [org.director].wake_cmd is empty (wake " + strings.Join(wantWake, ",") + "), " +
				"and [org.director].launch_cmd is set: launch_cmd opens a terminal ralph cannot type into, so ralph org watch --all could not wake the director; " +
				`set [org.director].wake_cmd (its {target} is director_<token>, the name launch_cmd gives the terminal) or use wake = ["pull"]`
			if err == nil || err.Error() != want {
				t.Fatalf("director start error =\n%v\nwant\n%s", err, want)
			}
			if !errors.Is(err, org.ErrDirectorWakeCmdNeeded) {
				t.Errorf("director start error = %v; want it to wrap ErrDirectorWakeCmdNeeded", err)
			}
			if stderr != "" {
				t.Errorf("stderr = %q; want nothing", stderr)
			}
			if got := dirNames(t, stateDir); len(got) != 0 {
				t.Errorf("ledger entries = %v; want none", got)
			}
			if _, err := os.Stat(ran); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("a refused start ran launch_cmd (stat %v)", err)
			}
			if len(fakes.lookedUp) != 0 {
				t.Errorf("a refused start looked up the agent %q; want no lookup", fakes.lookedUp)
			}
		})
	}
}

// AC4 and D9 for start: without --config, on the main worktree's ledger
// found from a linked worktree, start takes [org.director].wake and
// wake_cmd from the main worktree's ralph.toml with its launch_cmd: the
// main worktree's wake is recorded, and its empty wake_cmd refuses a start
// that pushes although the linked worktree's ralph.toml pulls; a wake_cmd
// there lets a start through that the linked worktree's file would refuse.
// --config names the file to read instead.
func TestOrgDirectorStart_ReadsTheWakeFromTheMainWorktree(t *testing.T) {
	for _, tc := range []struct {
		name              string
		mainWake, mainCmd string
		mainLaunch        bool   // the main worktree's ralph.toml has a launch_cmd
		wtWake            string // the linked worktree's ralph.toml always has a launch_cmd
		config            bool   // --config ralph.toml (the worktree's)
		wantRan           string // "main", "worktree", or "" (none ran)
		wantWake          []string
		refused           bool
	}{
		{"no --config: the main worktree pushes with a wake_cmd, the worktree without one",
			`["pull", "push"]`, directorWakeCmdWithoutTarget, true, `["push"]`, false, "main", []string{"pull", "push"}, false},
		{"no --config: the main worktree pushes without a wake_cmd, the worktree pulls",
			`["push"]`, "", true, `["pull"]`, false, "", nil, true},
		{"no --config: herdr records the main worktree's wake",
			`["pull", "push"]`, "", false, `["pull"]`, false, "", []string{"pull", "push"}, false},
		{"--config names the worktree's ralph.toml",
			`["push"]`, "", true, `["pull"]`, true, "worktree", []string{"pull"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			unsetOrgAgmsgHomeEnv(t)
			setDirectorLaunchSeams(t)
			r := newLegacyLedgerRepo(t)
			marks := t.TempDir()
			ranMain, ranWorktree := filepath.Join(marks, "main"), filepath.Join(marks, "worktree")
			mainToml := directorWakeToml(tc.mainWake, tc.mainCmd)
			if tc.mainLaunch {
				mainToml = directorTouchingLaunchCmdToml(ranMain) + strings.TrimPrefix(mainToml, "[org.director]\n")
			}
			writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), mainToml)
			writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), directorTouchingLaunchCmdToml(ranWorktree)+"wake = "+tc.wtWake+"\n")
			args := []string{"director", "start", "--driver", "claude", "--model", "opus"}
			if tc.config {
				args = append(args, "--config", "ralph.toml")
			}

			stdout, stderr, err := runOrgCmdStreams(t, args...)
			for _, m := range []struct{ name, path string }{{"main", ranMain}, {"worktree", ranWorktree}} {
				_, statErr := os.Stat(m.path)
				if ran := statErr == nil; ran != (m.name == tc.wantRan) {
					t.Errorf("the %s worktree's launch_cmd ran: %v; want %v", m.name, ran, m.name == tc.wantRan)
				}
			}
			switch {
			case tc.refused:
				if !errors.Is(err, org.ErrDirectorWakeCmdNeeded) {
					t.Fatalf("director start = %v (stdout: %s stderr: %s); want the wake_cmd refusal", err, stdout, stderr)
				}
				if got := directorFileOrAbsent(t, r.sharedDir); got != "<absent>" {
					t.Errorf("a refused start wrote director.jsonl: %s", got)
				}
				if lines := readLogLines(t, herdrLog); len(lines) != 0 {
					t.Errorf("herdr calls = %v; want none", lines)
				}
				return
			case tc.wantRan == "":
				if err != nil || !strings.HasPrefix(stdout, "started d1 (driver=claude, model=opus, permission=autonomous, pane=pane-stub-1, ") {
					t.Fatalf("director start = %v (stdout: %s stderr: %s); want d1 started in herdr", err, stdout, stderr)
				}
			default:
				if err == nil || !strings.Contains(err.Error(), "launch_cmd sh failed: exit status 3; d1 is recorded stopped") {
					t.Fatalf("director start = %v (stdout: %s stderr: %s); want launch_cmd's failure", err, stdout, stderr)
				}
			}
			if events := directorEventsIn(t, r.sharedDir); len(events) == 0 || !slices.Equal(events[0].Wake, tc.wantWake) {
				t.Errorf("director.jsonl in the main worktree's ledger = %+v; want attached d1 with wake %v", events, tc.wantWake)
			}
		})
	}
}

// AC2 for status: a registration whose wake has push shows where `ralph
// org watch --all` sends the wake line at the end of the director: line's
// details: a promoted session's --wake-target (with a wake_target: line
// right after wake: in the fields and wake_target in the JSON), nothing
// more for a promoted session with no target, a herdr director's pane once
// pane_created is recorded (nothing before it), and a launch cmd's terminal
// name. A pull-only registration's line is that of stage 8a.
func TestOrgDirectorStatus_ShowsWhereAPushWakeIsSent(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	t.Chdir(t.TempDir())
	setOrUnsetEnv(t, "HERDR_PANE_ID", "", true)
	t.Setenv("USER", "alice")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	attach := func(t *testing.T, wakeCmd string, args ...string) (stateDir string, ev org.DirectorEvent) {
		t.Helper()
		stateDir = t.TempDir()
		configPath := writeDirectorConfig(t, directorWakeToml("", wakeCmd))
		runInboxOK(t, append([]string{"director", "attach", "--driver", "claude", "--state-dir", stateDir, "--config", configPath}, args...)...)
		return stateDir, directorEventsIn(t, stateDir)[0]
	}

	t.Run("a promoted session with a wake target", func(t *testing.T) {
		stateDir, ev := attach(t, directorWakeCmdWithTarget, "--wake", "pull,push", "--wake-target", "term-1")
		wantLine := "director: attached d1 (driver=claude, attached at " + ev.TS + ", by alice, wake=pull,push to term-1)" + directorAttachedTail
		stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
		want := wantLine + "\nenabled: true\nauto_merge: false\nid: d1\ndriver: claude\nwake: pull,push\nwake_target: term-1\nattached_at: " + ev.TS +
			"\nuser: alice\ncwd: " + cwd + "\nherdr_pane: no\n"
		if stdout != want {
			t.Errorf("director status =\n%s\nwant\n%s", stdout, want)
		}
		jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
		if reg := directorReportOf(t, jsonOut).Registration; reg == nil || !slices.Equal(reg.Wake, []string{"pull", "push"}) || reg.WakeTarget != "term-1" ||
			!strings.Contains(jsonOut, `"wake_target": "term-1"`) {
			t.Errorf("director status --json = %s; want wake pull,push and wake_target term-1", jsonOut)
		}
		out, err := runStatusCmd(t, "--state-dir", stateDir)
		if err != nil {
			t.Fatalf("ralph status: %v (output %s)", err, out)
		}
		if got := statusDirectorLine(t, out); got != wantLine {
			t.Errorf("ralph status director line:\n got %q\nwant %q", got, wantLine)
		}
	})

	t.Run("a promoted session with no wake target", func(t *testing.T) {
		stateDir, ev := attach(t, directorWakeCmdWithoutTarget, "--wake", "push")
		want := "director: attached d1 (driver=claude, attached at " + ev.TS + ", by alice, wake=push)" + directorAttachedTail
		stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
		if line, _, _ := strings.Cut(stdout, "\n"); line != want {
			t.Errorf("director line:\n got %q\nwant %q", line, want)
		}
		if strings.Contains(stdout, "wake_target:") || !strings.Contains(stdout, "\nwake: push\nattached_at: ") {
			t.Errorf("director status =\n%s\nwant wake: push and no wake_target:", stdout)
		}
	})

	t.Run("a promoted session that pulls", func(t *testing.T) {
		stateDir, ev := attach(t, "")
		if got, want := directorStatusLine(t, "--state-dir", stateDir), directorAttachedLine("d1", "claude", ev.TS, "alice"); got != want {
			t.Errorf("director line:\n got %q\nwant %q", got, want)
		}
	})

	t.Run("a herdr director before and after its pane", func(t *testing.T) {
		stateDir := t.TempDir()
		store := org.NewDirectorStore(stateDir)
		who := org.DirectorWho{User: "alice", Cwd: "/work"}
		reg, err := store.AttachHeadless(org.DirectorHeadlessAttach{
			Driver: "codex", Model: "gpt-6-astra", PermissionMode: org.PermissionModeGuarded, Wake: []string{"push"}, User: "alice", Cwd: "/work",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.RecordWorkspace(reg.ID, "w-1", who); err != nil {
			t.Fatal(err)
		}
		want := "director: attached d1 (driver=codex, headless starting, model=gpt-6-astra, workspace w-1, attached at " + reg.AttachedAt + ", by alice, wake=push)" +
			directorAttachedTail
		if got := directorStatusLine(t, "--state-dir", stateDir); got != want {
			t.Errorf("director line before pane_created:\n got %q\nwant %q", got, want)
		}
		if err := store.RecordPane(reg.ID, "p-1", who); err != nil {
			t.Fatal(err)
		}
		if ok, err := store.MarkLaunched(reg.ID, who); err != nil || !ok {
			t.Fatalf("mark d1 launched: %v, %v", ok, err)
		}
		want = "director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane p-1, workspace w-1, attached at " + reg.AttachedAt +
			", by alice, wake=push to p-1)" + directorAttachedTail
		stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
		if line, _, _ := strings.Cut(stdout, "\n"); line != want {
			t.Errorf("director line after pane_created:\n got %q\nwant %q", line, want)
		}
		if strings.Contains(stdout, "wake_target:") || !strings.Contains(stdout, "\nwake: push\n") || !strings.Contains(stdout, "\npane_id: p-1\n") {
			t.Errorf("director status =\n%s\nwant wake: push, pane_id: p-1, and no wake_target:", stdout)
		}
	})

	t.Run("a launch cmd director", func(t *testing.T) {
		stateDir := t.TempDir()
		reg, err := org.NewDirectorStore(stateDir).AttachHeadless(org.DirectorHeadlessAttach{
			Launch: org.DirectorLaunchCmd, Driver: "claude", Model: "opus", PermissionMode: org.PermissionModeAutonomous, Wake: []string{"pull", "push"},
			User: "alice", Cwd: "/work",
		})
		if err != nil {
			t.Fatal(err)
		}
		name := "director_" + reg.Token
		want := "director: attached d1 (driver=claude, headless starting, model=opus, launch cmd, name " + name + ", attached at " + reg.AttachedAt +
			", by alice, wake=pull,push to " + name + ")" + directorAttachedTail
		stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
		if line, _, _ := strings.Cut(stdout, "\n"); line != want {
			t.Errorf("director line:\n got %q\nwant %q", line, want)
		}
		if strings.Contains(stdout, "wake_target:") || !strings.Contains(stdout, "\nterminal_name: "+name+"\n") {
			t.Errorf("director status =\n%s\nwant terminal_name: %s and no wake_target:", stdout, name)
		}
	})
}

// AC2 for status: the wake and the wake target of a hand-edited
// director.jsonl print with their control characters escaped, on the
// director: line and on the wake: and wake_target: lines, so neither can
// forge a line or drive the terminal; the JSON carries them as recorded.
func TestOrgDirectorStatus_EscapesTheWakeAndTheWakeTarget(t *testing.T) {
	t.Chdir(t.TempDir())
	stateDir := t.TempDir()
	const ts = "2026-10-18T09:00:00Z"
	writeDirectorLines(t, stateDir, directorEventLine(t, org.DirectorEvent{
		TS: ts, ID: "d1", Event: org.DirectorEventAttached, Driver: "claude", Wake: []string{"push", "bell\a"}, WakeTarget: "t\x1b[2J\ndirector: forged",
	}))

	stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
	lines := strings.Split(stdout, "\n")
	wantLine := `director: attached d1 (driver=claude, attached at ` + ts + `, wake=push,bell\a to t\x1b[2J\ndirector: forged)` + directorAttachedTail
	if lines[0] != wantLine {
		t.Errorf("director line:\n got %q\nwant %q", lines[0], wantLine)
	}
	if want := "\n" + `wake: push,bell\a` + "\n" + `wake_target: t\x1b[2J\ndirector: forged` + "\n"; !strings.Contains(stdout, want) {
		t.Errorf("director status does not contain %q:\n%s", want, stdout)
	}
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "director:") {
			t.Errorf("the wake target forged a second director: line %q:\n%s", line, stdout)
		}
	}
	if strings.ContainsAny(stdout, "\a\x1b") {
		t.Errorf("stdout holds a raw control character: %q", stdout)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if reg := directorReportOf(t, jsonOut).Registration; reg == nil || reg.WakeTarget != "t\x1b[2J\ndirector: forged" || !slices.Equal(reg.Wake, []string{"push", "bell\a"}) {
		t.Errorf("director status --json = %s; want the wake and wake target as recorded", jsonOut)
	}
}

// AC2 and AC3 for the Go help: attach describes --wake and --wake-target
// and their refusals, start says it records [org.director].wake and when a
// launch_cmd start that pushes is refused, and status says where the
// director: line says the wake line goes.
func TestOrgDirectorHelp_DescribesTheWake(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"director", "attach", "--help"}, []string{
			"--wake is how the director learns of a new open inbox item: pull (it waits on the inbox with `ralph org wait --inbox`), push, or both (pull,push).",
			"With push, `ralph org watch --all` wakes it by running [org.director].wake_cmd with {target} replaced by --wake-target and {text} by one fixed line that names the new items.",
			"Without --wake, attach records [org.director].wake.",
			"attach is refused, and writes nothing (with --replace, the current registration stays), when --wake has a value other than pull and push or one twice, " +
				"when the wake has push and wake_cmd is empty, when wake_cmd has {target} and no --wake-target is given, " +
				"when --wake-target is given and the wake has no push, and when --wake-target is empty, has a control character, " +
				`is longer than 256 characters, or starts with "-".`,
			"Without --config, wake and wake_cmd are read from the main worktree's ralph.toml by the rule of enabled.",
			"attached d1 (driver=claude, wake=push, wake_target=...)",
			"how the director learns of a new inbox item: pull, push, or pull,push (default: [org.director].wake); " +
				"push has ralph org watch --all wake it by running [org.director].wake_cmd with {target} = --wake-target",
			"where [org.director].wake_cmd sends the wake line, its {target} (such as a terminal's handle); only with a wake that has push",
		}},
		{[]string{"director", "start", "--help"}, []string{
			"So are [org.director].wake, which start records as the director's wake (see ralph org director attach --help), and wake_cmd.",
			"It is refused too when [org.director].wake has push and wake_cmd is empty: the terminal launch_cmd opens is one ralph cannot type into, " +
				`so set wake_cmd (its {target} is director_<token>) or use wake = ["pull"].`,
			"In herdr, push needs no wake_cmd: the watch types the wake line into the director's pane.",
		}},
		{[]string{"director", "status", "--help"}, []string{
			"The fields show the registration's wake, and wake_target when attach was given --wake-target.",
			"For a registration whose wake has push, the director: line adds wake= and where `ralph org watch --all` sends the wake line: " +
				"the --wake-target, the herdr pane, or the terminal name.",
		}},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
	}
}
