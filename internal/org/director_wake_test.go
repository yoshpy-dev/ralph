package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// Tests for the wake of a director's registration (plan
// 2026-10-18-org-director-wake, stage 8b-1: D3, D4, AC2, and AC3 in the org
// layer): the wake and wake_target an attached records, the shape of a wake
// target, attach's check that a wake can be sent, the target a registration
// resolves to, and the wake start records.
// Every test runs on a store under t.TempDir() and the fake herdr, never
// herdr, a terminal, or a launch_cmd.

// wakeTestLaunchCmd gives f's Org a valid [org.director].launch_cmd and
// lookups of the agent and of ralph that fail the test when they are asked,
// so a start refused before checkDirectorLaunchCmd proves it asked neither.
func wakeTestLaunchCmd(t *testing.T, f *dirStartFixture) {
	t.Helper()
	f.o.Config.Director.LaunchCmd = []string{"sh", "-c", "exit 0", "sh", "{script}"}
	f.o.DirectorAgentLookPath = func(file string) (string, error) {
		t.Errorf("the %s agent was looked up", file)
		return "", errors.New("not looked up in this test")
	}
	f.o.DirectorRalphExecutable = func() (string, error) {
		t.Error("the ralph executable was looked up")
		return "", errors.New("not looked up in this test")
	}
}

// ValidateDirectorWakeTarget (D4): a target is not empty, is valid UTF-8
// with no control character, has at most 256 characters (runes, not bytes),
// and does not start with "-". Each refusal wraps
// ErrDirectorWakeTargetInvalid, says why, and quotes the value.
func TestValidateDirectorWakeTarget(t *testing.T) {
	for _, target := range []string{
		"x", "x-", "%orca-1", "w1:p2", "director_0123456789ab", "a b", "a{target}",
		strings.Repeat("a", MaxDirectorWakeTargetRunes),
		strings.Repeat("あ", MaxDirectorWakeTargetRunes), // 768 bytes, 256 characters
	} {
		if err := ValidateDirectorWakeTarget(target); err != nil {
			t.Errorf("ValidateDirectorWakeTarget(%q) = %v, want nil", target, err)
		}
	}
	for _, tc := range []struct {
		name, target, reason string
	}{
		{"empty", "", "it is empty"},
		{"a bell", "a\x07b", `"a\ab" has a control character`},
		{"a newline", "a\nb", `"a\nb" has a control character`},
		{"a tab", "a\tb", `"a\tb" has a control character`},
		{"an escape", "\x1b[2J", `"\x1b[2J" has a control character`},
		{"DEL", "a\x7f", `"a\x7f" has a control character`},
		{"a C1 control", "a\u0085", `"a\u0085" has a control character`},
		{"invalid UTF-8", "a\xffb", `"a\xffb" is not valid UTF-8`},
		{"257 characters", strings.Repeat("a", MaxDirectorWakeTargetRunes+1), "it is 257 characters long, more than 256"},
		{"257 multibyte characters", strings.Repeat("あ", MaxDirectorWakeTargetRunes+1), "it is 257 characters long, more than 256"},
		{"a leading dash", "-x", `"-x" starts with "-", which the command [org.director].wake_cmd runs could read as an option`},
		{"a flag", "--terminal", `"--terminal" starts with "-"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDirectorWakeTarget(tc.target)
			if !errors.Is(err, ErrDirectorWakeTargetInvalid) || !strings.HasPrefix(err.Error(), "the wake target is not usable: "+tc.reason) {
				t.Fatalf("ValidateDirectorWakeTarget(%q) = %v, want ErrDirectorWakeTargetInvalid with %q", tc.target, err, tc.reason)
			}
		})
	}
}

// CheckDirectorAttachWake (D4): a wake with push needs a wake_cmd, and a
// wake_cmd with {target} needs a target; a target needs push and a usable
// shape. pull alone, and push with a wake_cmd whose arguments do not use
// {target}, need no target.
func TestCheckDirectorAttachWake(t *testing.T) {
	var (
		pull     = []string{DirectorWakePull}
		push     = []string{DirectorWakePush}
		pullPush = []string{DirectorWakePull, DirectorWakePush}
		pushPull = []string{DirectorWakePush, DirectorWakePull}
		orca     = []string{"orca", "terminal", "send", "--terminal", "{target}", "--text", "{text}", "--enter"}
		inside   = []string{"wake", "--to={target}", "--line={text}"}
		noTarget = []string{"./wake.sh", "{text}"}
	)
	for _, tc := range []struct {
		name    string
		wake    []string
		target  string
		wakeCmd []string
		is      error // nil: accepted
	}{
		{"pull, no target", pull, "", nil, nil},
		{"pull, no target, a wake_cmd", pull, "", orca, nil},
		{"pull with a target", pull, "%1", orca, ErrDirectorWakeTargetWithoutPush},
		{"pull with a target and no wake_cmd", pull, "%1", nil, ErrDirectorWakeTargetWithoutPush},
		{"pull with a bad target", pull, "-x", orca, ErrDirectorWakeTargetWithoutPush},
		{"push, no wake_cmd", push, "", nil, ErrDirectorWakeCmdNeeded},
		{"push, no wake_cmd, a target", push, "%1", nil, ErrDirectorWakeCmdNeeded},
		{"push, {target}, no target", push, "", orca, ErrDirectorWakeTargetNeeded},
		{"push, {target} inside an element, no target", push, "", inside, ErrDirectorWakeTargetNeeded},
		{"push, a wake_cmd without {target}, no target", push, "", noTarget, nil},
		{"push, a target, {target}", push, "%1", orca, nil},
		{"push, a target, a wake_cmd without {target}", push, "%1", noTarget, nil},
		{"push, a target that starts with -", push, "-x", orca, ErrDirectorWakeTargetInvalid},
		{"push, a target with a tab", push, "a\tb", orca, ErrDirectorWakeTargetInvalid},
		{"pull,push, no wake_cmd", pullPush, "", nil, ErrDirectorWakeCmdNeeded},
		{"push,pull, {target}, no target", pushPull, "", orca, ErrDirectorWakeTargetNeeded},
		{"pull,push, a target", pullPush, "%1", orca, nil},
		{"push,pull, a wake_cmd without {target}", pushPull, "", noTarget, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckDirectorAttachWake(tc.wake, tc.target, tc.wakeCmd)
			if tc.is == nil {
				if err != nil {
					t.Fatalf("CheckDirectorAttachWake(%q, %q, %q) = %v, want nil", tc.wake, tc.target, tc.wakeCmd, err)
				}
				return
			}
			if !errors.Is(err, tc.is) {
				t.Fatalf("CheckDirectorAttachWake(%q, %q, %q) = %v, want it to wrap %v", tc.wake, tc.target, tc.wakeCmd, err, tc.is)
			}
			if strings.HasPrefix(err.Error(), "org: ") {
				t.Errorf("error %q starts with org:; the CLI adds its verb", err)
			}
		})
	}

	for _, tc := range []struct {
		wake    []string
		target  string
		wakeCmd []string
		want    string
	}{
		{pullPush, "", nil, "the wake has push, but [org.director].wake_cmd is empty (wake pull,push): a session attached with ralph org director attach " +
			"has no default way to be sent the wake line (only a headless director in herdr has one, its herdr pane), so ralph org watch --all " +
			"could not wake it; set [org.director].wake_cmd, or attach with --wake pull"},
		{push, "", orca, "[org.director].wake_cmd has {target}, but no --wake-target was given (wake push): pass --wake-target with where " +
			"[org.director].wake_cmd sends the wake line"},
		{pull, "%1", orca, "--wake-target is given, but the wake has no push (wake pull): the watch sends to the wake target only to push; " +
			"drop --wake-target, or attach with --wake push or --wake pull,push"},
		{push, "-x", orca, `--wake-target: the wake target is not usable: "-x" starts with "-", which the command [org.director].wake_cmd runs could read as an option`},
	} {
		if err := CheckDirectorAttachWake(tc.wake, tc.target, tc.wakeCmd); err == nil || err.Error() != tc.want {
			t.Errorf("CheckDirectorAttachWake(%q, %q, %q) =\n%v\nwant\n%s", tc.wake, tc.target, tc.wakeCmd, err, tc.want)
		}
	}
}

// WakesByPush and ResolvedWakeTarget (D3): push in the wake; the target is a
// promoted session's --wake-target, a launch herdr's pane (none before its
// pane_created), and a launch cmd's terminal name director_<token>. A launch
// this binary does not know counts as a promoted session's.
func TestDirectorRegistration_WakesByPushAndResolvedWakeTarget(t *testing.T) {
	for _, tc := range []struct {
		wake []string
		want bool
	}{
		{nil, false}, {[]string{"pull"}, false}, {[]string{"push"}, true}, {[]string{"pull", "push"}, true}, {[]string{"push", "pull"}, true},
	} {
		if got := (DirectorRegistration{Wake: tc.wake}).WakesByPush(); got != tc.want {
			t.Errorf("WakesByPush with wake %q = %v, want %v", tc.wake, got, tc.want)
		}
	}
	for _, tc := range []struct {
		name string
		reg  DirectorRegistration
		want string
	}{
		{"a promoted session with a target", DirectorRegistration{WakeTarget: "%orca-1", PaneID: "w9:p9"}, "%orca-1"},
		{"a promoted session without one", DirectorRegistration{PaneID: "w9:p9"}, ""},
		{"a launch herdr before pane_created", DirectorRegistration{Launch: DirectorLaunchHerdr, Token: directorTestToken}, ""},
		{"a launch herdr with its pane", DirectorRegistration{Launch: DirectorLaunchHerdr, Token: directorTestToken, PaneID: "w7:p1", WakeTarget: "x"}, "w7:p1"},
		{"a launch cmd", DirectorRegistration{Launch: DirectorLaunchCmd, Token: directorTestToken, WakeTarget: "x"}, "director_" + directorTestToken},
		{"a later binary's launch", DirectorRegistration{Launch: "terminal", WakeTarget: "t1"}, "t1"},
	} {
		if got := tc.reg.ResolvedWakeTarget(); got != tc.want {
			t.Errorf("%s: ResolvedWakeTarget = %q, want %q", tc.name, got, tc.want)
		}
	}

	// Through the store: a herdr registration's target appears with its
	// pane_created.
	s, _, _ := newTestHeadlessStore(t)
	h := testHeadlessAttach()
	h.Wake = []string{DirectorWakePush}
	reg, err := s.AttachHeadless(h)
	if err != nil {
		t.Fatalf("AttachHeadless: %v", err)
	}
	if got := reg.ResolvedWakeTarget(); got != "" || !reg.WakesByPush() {
		t.Fatalf("a starting launch herdr: ResolvedWakeTarget %q, WakesByPush %v; want none and true", got, reg.WakesByPush())
	}
	if err := s.RecordPane(reg.ID, "w7:p1", testDirectorWho); err != nil {
		t.Fatalf("RecordPane: %v", err)
	}
	if got := mustReadDirector(t, s).Current.ResolvedWakeTarget(); got != "w7:p1" {
		t.Fatalf("after pane_created: ResolvedWakeTarget = %q, want w7:p1", got)
	}
}

// AC2 in the store: attach writes the wake and the wake target it is given
// into the attached, the fold reads both back, and the registration's JSON
// carries wake_target right after wake. A replace with no target writes no
// wake_target key, and the wake recorded is a copy of the caller's slice.
func TestDirectorAttach_RecordsTheWakeAndTheWakeTarget(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	wake := []string{DirectorWakePull, DirectorWakePush}
	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude, Wake: wake, WakeTarget: "%orca-1", User: "alice", Cwd: "/work/a"})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	wake[0] = "changed"
	want := DirectorRegistration{
		ID: "d1", Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull, DirectorWakePush}, WakeTarget: "%orca-1",
		AttachedAt: directorTestNow, User: "alice", Cwd: "/work/a",
	}
	if !reflect.DeepEqual(reg, want) || replaced != nil {
		t.Fatalf("Attach = %+v, replaced %v; want %+v and nil", reg, replaced, want)
	}
	wantLine := `{"ts":"2026-10-15T09:00:00Z","id":"d1","event":"attached","driver":"claude","wake":["pull","push"],"wake_target":"%orca-1",` +
		`"user":"alice","cwd":"/work/a","herdr_pane":false}`
	if got := readDirectorFile(t, s); got != wantLine+"\n" {
		t.Fatalf("director.jsonl = %q, want %q", got, wantLine+"\n")
	}
	res := mustReadDirector(t, s)
	if res.Current == nil || !reflect.DeepEqual(*res.Current, want) || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want %+v", res, res.Current, want)
	}
	raw, err := json.Marshal(*res.Current)
	if err != nil {
		t.Fatal(err)
	}
	if wantJSON := `{"id":"d1","driver":"claude","wake":["pull","push"],"wake_target":"%orca-1","attached_at":"2026-10-15T09:00:00Z",` +
		`"user":"alice","cwd":"/work/a","herdr_pane":false}`; string(raw) != wantJSON {
		t.Fatalf("JSON = %s, want %s", raw, wantJSON)
	}

	reg, replaced, err = s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, Wake: []string{DirectorWakePush}, User: "bob"})
	if err != nil {
		t.Fatalf("Attach --replace: %v", err)
	}
	if reg.ID != "d2" || !slices.Equal(reg.Wake, []string{DirectorWakePush}) || reg.WakeTarget != "" || replaced == nil || replaced.WakeTarget != "%orca-1" {
		t.Fatalf("Attach --replace = %+v, replaced %+v; want d2 with wake push and no target, replacing d1 with its target", reg, replaced)
	}
	lines := directorFileLines(t, s)
	if last := lines[len(lines)-1]; strings.Contains(last, "wake_target") || !strings.Contains(last, `"wake":["push"]`) {
		t.Fatalf("the replace's attached = %s; want wake push and no wake_target key", last)
	}
}

// An empty wake records pull, as every Attach did before stage 8b: the CLI
// always passes the wake it resolved, and callers written before the wake
// existed keep recording pull.
func TestDirectorAttach_AnEmptyWakeRecordsPull(t *testing.T) {
	for _, wake := range [][]string{nil, {}} {
		s, _, _ := newTestDirectorStore(t)
		reg := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, Wake: wake})
		if !slices.Equal(reg.Wake, []string{DirectorWakePull}) || !strings.Contains(readDirectorFile(t, s), `"wake":["pull"]`) {
			t.Errorf("Attach with wake %#v recorded %q (%s), want pull", wake, reg.Wake, readDirectorFile(t, s))
		}
	}
}

// AC2's defense in depth in the store: a wake that config.ValidateDirectorWake
// refuses and a wake target that ValidateDirectorWakeTarget refuses are
// refused before anything is read or written: a missing state dir is not
// created, and with a registration current, also with Replace, the file's
// bytes stay as they were and the registration stays current.
func TestDirectorAttach_RefusesABadWakeOrTargetWithNothingWritten(t *testing.T) {
	for _, tc := range []struct {
		name   string
		wake   []string
		target string
		want   string
	}{
		{"an unknown wake", []string{"poll"}, "", `org: director: attach: [org.director].wake: element 0 "poll" is not a wake value`},
		{"an empty wake value", []string{"pull", ""}, "", `org: director: attach: [org.director].wake: element 1 "" is not a wake value`},
		{"a wake listed twice", []string{"push", "push"}, "x", `org: director: attach: [org.director].wake: element 1 "push" repeats element 0`},
		{"a target that starts with -", []string{"push"}, "-x", `org: director: attach: the wake target is not usable: "-x" starts with "-"`},
		{"a target with a newline", []string{"push"}, "a\nb", `org: director: attach: the wake target is not usable: "a\nb" has a control character`},
		{"a target that is not UTF-8", []string{"push"}, "\xff", `org: director: attach: the wake target is not usable: "\xff" is not valid UTF-8`},
		{"a target of 257 characters", []string{"push"}, strings.Repeat("t", 257), "org: director: attach: the wake target is not usable: it is 257 characters long"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := filepath.Join(t.TempDir(), "absent", "org")
			a := DirectorAttach{Driver: DirectorDriverClaude, Wake: tc.wake, WakeTarget: tc.target}
			if _, _, err := NewDirectorStore(state).Attach(a); err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("Attach = %v, want an error starting with %q", err, tc.want)
			}
			if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("a refused Attach created the state dir (stat err = %v)", err)
			}

			s, _, _ := newTestDirectorStore(t)
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverCodex, User: "alice"})
			before := readDirectorFile(t, s)
			for _, replace := range []bool{false, true} {
				a.Replace = replace
				if _, _, err := s.Attach(a); err == nil || !strings.HasPrefix(err.Error(), tc.want) {
					t.Fatalf("Attach (replace %v) = %v, want an error starting with %q", replace, err, tc.want)
				}
				if after := readDirectorFile(t, s); after != before {
					t.Fatalf("a refused Attach (replace %v) changed director.jsonl from %q to %q", replace, before, after)
				}
			}
			if got := currentDirectorID(mustReadDirector(t, s)); got != "d1" {
				t.Fatalf("current registration = %q, want d1", got)
			}
		})
	}
}

// D3 in the fold: a wake_target is read for a promoted session only. A
// headless attached never carries one (start writes none), so a hand-written
// one is not read, as its pane_id is not, and its target is its pane. The
// fold stays lenient about the wake: an attached whose wake has a value this
// binary does not know still registers, nothing ignored.
func TestDirectorRead_AWakeTargetIsReadForAPromotedSessionOnly(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, strings.Join([]string{
		`{"ts":"2026-10-15T00:00:00Z","id":"d1","event":"attached","driver":"claude","wake":["push"],"wake_target":"%orca-1","user":"u","cwd":"/w","herdr_pane":false}`,
	}, "\n")+"\n")
	res := mustReadDirector(t, s)
	if res.Current == nil || res.Current.WakeTarget != "%orca-1" || res.Current.ResolvedWakeTarget() != "%orca-1" || !res.Current.WakesByPush() || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want d1 with wake push to %%orca-1", res, res.Current)
	}

	writeDirectorFile(t, s, strings.Join([]string{
		directorAttachedLine("d1", "claude"),
		directorDetachedLine("d1"),
		`{"ts":"2026-10-15T00:00:00Z","id":"d2","event":"attached","driver":"codex","wake":["push"],"wake_target":"%forged","pane_id":"w0:p0",` +
			`"launch":"herdr","token":"0123456789ab","model":"m","permission_mode":"guarded","user":"u","cwd":"/w","herdr_pane":false}`,
		directorPaneLine("d2", "w7:p1"),
	}, "\n")+"\n")
	res = mustReadDirector(t, s)
	if cur := res.Current; cur == nil || cur.ID != "d2" || cur.WakeTarget != "" || cur.PaneID != "w7:p1" || cur.ResolvedWakeTarget() != "w7:p1" || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want the headless d2 with no wake target, its pane w7:p1 as the target", res, res.Current)
	}
	raw, err := json.Marshal(*res.Current)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "wake_target") {
		t.Fatalf("the headless registration's JSON = %s; want no wake_target", raw)
	}

	writeDirectorFile(t, s, `{"ts":"2026-10-15T00:00:00Z","id":"d1","event":"attached","driver":"claude","wake":["poll","push"],"user":"u","cwd":"/w","herdr_pane":false}`+"\n")
	res = mustReadDirector(t, s)
	if res.Current == nil || !slices.Equal(res.Current.Wake, []string{"poll", "push"}) || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want d1 registered with its wake as written and nothing ignored", res, res.Current)
	}
}

// AC3 in the store: AttachHeadless records the wake it is given and no
// wake_target.
func TestDirectorAttachHeadless_RecordsTheWake(t *testing.T) {
	s, _, _ := newTestHeadlessStore(t)
	h := testHeadlessAttach()
	h.Wake = []string{DirectorWakePush, DirectorWakePull}
	reg, err := s.AttachHeadless(h)
	if err != nil {
		t.Fatalf("AttachHeadless: %v", err)
	}
	h.Wake[0] = "changed"
	if !slices.Equal(reg.Wake, []string{DirectorWakePush, DirectorWakePull}) || reg.WakeTarget != "" {
		t.Fatalf("AttachHeadless = %+v, want wake push,pull and no target", reg)
	}
	line := readDirectorFile(t, s)
	if !strings.Contains(line, `"wake":["push","pull"],"launch":"herdr"`) || strings.Contains(line, "wake_target") {
		t.Fatalf("director.jsonl = %s, want wake push,pull and no wake_target", line)
	}
}

// AC3: start records the wake of [org.director].wake in its attached, push
// included with an empty wake_cmd for a start in herdr, whose pane is then
// the registration's target.
func TestStartDirector_RecordsTheWakeOfTheConfig(t *testing.T) {
	for _, wake := range [][]string{{DirectorWakePull}, {DirectorWakePush}, {DirectorWakePull, DirectorWakePush}} {
		t.Run(strings.Join(wake, ","), func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.o.Config.Director.Wake = wake
			res := f.mustStart(t, DirectorDriverClaude, "opus")
			if !slices.Equal(res.Registration.Wake, wake) || res.Registration.WakeTarget != "" {
				t.Fatalf("StartDirector registration = %+v, want wake %q and no wake target", res.Registration, wake)
			}
			if got := res.Registration.ResolvedWakeTarget(); got != res.Registration.PaneID || got == "" {
				t.Fatalf("ResolvedWakeTarget = %q, want the pane %q", got, res.Registration.PaneID)
			}
			lines := directorFileLines(t, f.store)
			var att DirectorEvent
			if err := json.Unmarshal([]byte(lines[0]), &att); err != nil {
				t.Fatal(err)
			}
			if att.Event != DirectorEventAttached || !slices.Equal(att.Wake, wake) || att.WakeTarget != "" {
				t.Fatalf("attached = %+v, want wake %q and no wake target", att, wake)
			}
		})
	}
}

// AC3: checkDirectorStart lets a launch_cmd start with push through when
// [org.director].wake_cmd is set, and with pull alone when it is not; the
// refusals are in TestStartDirector_RefusesBeforeWritingOrCallingHerdr.
func TestStartDirector_ALaunchCmdStartThatPushesPassesWithAWakeCmd(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wake    []string
		wakeCmd []string
	}{
		{"push with a wake_cmd", []string{DirectorWakePush}, []string{"tmux", "send-keys", "-t", "{target}", "-l", "{text}"}},
		{"pull,push with a wake_cmd", []string{DirectorWakePull, DirectorWakePush}, []string{"./wake.sh", "{text}"}},
		{"pull with no wake_cmd", []string{DirectorWakePull}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.o.Config.Director.LaunchCmd = []string{"sh", "-c", "exit 0", "sh", "{script}"}
			f.o.Config.Director.Wake, f.o.Config.Director.WakeCmd = tc.wake, tc.wakeCmd
			f.o.DirectorAgentLookPath = func(string) (string, error) { return "/bin/sh", nil }
			f.o.DirectorRalphExecutable = func() (string, error) { return "/bin/sh", nil }
			p := f.params(DirectorDriverClaude, "opus")
			c, err := f.o.checkDirectorStart(&p)
			if err != nil || !c.cmdLaunch {
				t.Fatalf("checkDirectorStart = %+v, %v; want the launch_cmd path and no error", c, err)
			}
		})
	}
}

// The tests below are for the push wake of `ralph org watch --all` (plan
// 2026-10-18-org-director-wake, D5 to D8, AC5, AC6, and AC7):
// WakeDirector, its text, the items it wakes for, and its memory. The
// wake_cmd they run is a #!/bin/sh script under t.TempDir() that records its
// arguments and its working directory (writeWakeScript); herdr is the fake of
// the start and stop tests. No test changes the working directory of this
// process.

// wakeTestNow is the clock of a wakeFixture's inbox and WakeDirector calls.
var wakeTestNow = time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)

// wakeRun is one run of a script writeWakeScript wrote: its arguments and
// its working directory as `pwd -P` prints it.
type wakeRun struct {
	args []string
	pwd  string
}

// writeWakeScript writes path, mode 0755, a #!/bin/sh stand-in for the
// program [org.director].wake_cmd names: it appends one run to mark (a
// "run" line, an "arg=<value>" line per argument, and "pwd=<pwd -P>"), then
// runs tail, such as "exit 1". The watch runs it without a shell; the script
// itself is a shell script.
func writeWakeScript(t *testing.T, path, mark, tail string) {
	t.Helper()
	writeTestExecutable(t, path, "#!/bin/sh\n"+
		"{ printf 'run\\n'; for a in \"$@\"; do printf 'arg=%s\\n' \"$a\"; done; printf 'pwd=%s\\n' \"$(pwd -P)\"; } >> "+shellQuote(mark)+"\n"+
		tail+"\n")
}

// readWakeRuns returns the runs a writeWakeScript script recorded in mark,
// nil when it never ran.
func readWakeRuns(t *testing.T, mark string) []wakeRun {
	t.Helper()
	raw, err := os.ReadFile(mark)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read %s: %v", mark, err)
	}
	var runs []wakeRun
	for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		switch {
		case line == "run":
			runs = append(runs, wakeRun{})
		case len(runs) == 0:
			t.Fatalf("%s: %q before the first run line", mark, line)
		case strings.HasPrefix(line, "arg="):
			runs[len(runs)-1].args = append(runs[len(runs)-1].args, strings.TrimPrefix(line, "arg="))
		case strings.HasPrefix(line, "pwd="):
			runs[len(runs)-1].pwd = strings.TrimPrefix(line, "pwd=")
		default:
			t.Fatalf("%s: unexpected line %q", mark, line)
		}
	}
	return runs
}

// wakeRealPath is path with its symlinks resolved, as `pwd -P` prints it.
func wakeRealPath(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// wakeIDs returns the item ids e1 to en.
func wakeIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("e%d", i+1)
	}
	return ids
}

// writeDirectorEvents writes evs, one JSON line each, as s's director.jsonl:
// a hand-edited file.
func writeDirectorEvents(t *testing.T, s *DirectorStore, evs ...DirectorEvent) {
	t.Helper()
	var b strings.Builder
	for _, ev := range evs {
		raw, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	writeDirectorFile(t, s, b.String())
}

// wakeTextHerdr is a directorHerdr that also records the pane and the text
// of every PaneSendText, which fakeHerdr records by name only, and when each
// PaneSendText and PaneSendKeys was called.
type wakeTextHerdr struct {
	*directorHerdr
	texts          []paneSendTextCall
	textAt, keysAt []time.Time
}

func (h *wakeTextHerdr) PaneSendText(ctx context.Context, paneID, text string) error {
	h.texts = append(h.texts, paneSendTextCall{paneID: paneID, text: text})
	h.textAt = append(h.textAt, time.Now())
	return h.directorHerdr.PaneSendText(ctx, paneID, text)
}

func (h *wakeTextHerdr) PaneSendKeys(ctx context.Context, paneID string, keys ...string) error {
	h.keysAt = append(h.keysAt, time.Now())
	return h.directorHerdr.PaneSendKeys(ctx, paneID, keys...)
}

// wakeFixture is a dirStartFixture with an inbox in the same ledger, a
// wake_cmd of ["./wake.sh", "{target}", "{text}"] whose wake.sh in cmdDir
// records its runs in mark and exits 0, and a DirectorWakeMemory, so
// WakeDirector can be called cycle after cycle at f.now.
type wakeFixture struct {
	*dirStartFixture
	inbox  *InboxStore
	cmdDir string
	mark   string
	banner bytes.Buffer
	mem    DirectorWakeMemory
	now    time.Time
}

func newWakeFixture(t *testing.T) *wakeFixture {
	t.Helper()
	f := &wakeFixture{dirStartFixture: newDirStartFixture(t, directorTestToken), cmdDir: t.TempDir(), now: wakeTestNow}
	f.mark = filepath.Join(t.TempDir(), "runs")
	f.inbox = NewInboxStore(f.dir)
	f.inbox.Now = func() time.Time { return f.now }
	f.o.Inbox = f.inbox
	f.o.Config.Director.WakeCmd = []string{"./wake.sh", "{target}", "{text}"}
	f.script(t, "exit 0")
	return f
}

// script rewrites cmdDir/wake.sh with tail after its record of the run.
func (f *wakeFixture) script(t *testing.T, tail string) {
	t.Helper()
	writeWakeScript(t, filepath.Join(f.cmdDir, "wake.sh"), f.mark, tail)
}

func (f *wakeFixture) wake() DirectorWakeResult {
	return f.o.WakeDirector(f.now, f.dir, f.cmdDir, &f.mem, &f.banner)
}

// attachPush attaches a promoted session with wake pull,push and target.
func (f *wakeFixture) attachPush(t *testing.T, target string) DirectorRegistration {
	t.Helper()
	return mustAttachDirector(t, f.store, DirectorAttach{
		Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull, DirectorWakePush}, WakeTarget: target, User: "alice", Cwd: "/work/a",
	})
}

// escalate records a QUESTION with body, left to the deadline, and returns
// its id.
func (f *wakeFixture) escalate(t *testing.T, body string) string {
	t.Helper()
	id, err := f.inbox.Escalate(InboxEscalation{OrgID: "org-a", Type: "QUESTION", Body: "TYPE: QUESTION\n\n" + body, Delivery: InboxDeliveryDeadline})
	if err != nil {
		t.Fatalf("Escalate: %v", err)
	}
	return id
}

func (f *wakeFixture) runs(t *testing.T) []wakeRun {
	t.Helper()
	return readWakeRuns(t, f.mark)
}

func (f *wakeFixture) item(t *testing.T, id string) InboxItem {
	t.Helper()
	return inboxItemByID(t, mustReadInbox(t, f.inbox), id)
}

// bannerLines returns the lines written to the banner so far.
func (f *wakeFixture) bannerLines() []string {
	if f.banner.Len() == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(f.banner.String(), "\n"), "\n")
}

// wokenIDs returns the ids of the items of the inbox with a woken event.
func (f *wakeFixture) wokenIDs(t *testing.T) []string {
	t.Helper()
	var ids []string
	for _, item := range mustReadInbox(t, f.inbox).Items {
		if item.Woken {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

// D7: the wake line names up to 20 ids in the order given and counts the
// rest, holds none of the shell characters D7 lists, and does not start with
// a character an agent reads as a command.
func TestDirectorWakeText(t *testing.T) {
	const tail = ". Read them with ralph org inbox and handle each as the director skill says."
	const twenty = "e1 e2 e3 e4 e5 e6 e7 e8 e9 e10 e11 e12 e13 e14 e15 e16 e17 e18 e19 e20"
	for _, tc := range []struct {
		ids  []string
		want string
	}{
		{[]string{"e1"}, "New org inbox items for the director: e1" + tail},
		{[]string{"e3", "e5"}, "New org inbox items for the director: e3 e5" + tail},
		{[]string{"e10", "e2"}, "New org inbox items for the director: e10 e2" + tail},
		{wakeIDs(20), "New org inbox items for the director: " + twenty + tail},
		{wakeIDs(21), "New org inbox items for the director: " + twenty + " plus 1 more" + tail},
		{wakeIDs(25), "New org inbox items for the director: " + twenty + " plus 5 more" + tail},
	} {
		got := DirectorWakeText(tc.ids)
		if got != tc.want {
			t.Errorf("DirectorWakeText(%d ids) =\n%q\nwant\n%q", len(tc.ids), got, tc.want)
		}
		if strings.ContainsAny(got, ";|&$`<>()*?'\"") || strings.ContainsAny(got[:1], "/!#") || strings.ContainsAny(got, "\n\r") {
			t.Errorf("DirectorWakeText(%d ids) = %q holds a shell character, a line break, or starts with /, !, or #", len(tc.ids), got)
		}
	}
}

// D5: a wake is for the items that are open and have no woken event, in the
// inbox's order, whatever their delivery.
func TestDirectorWakeItems(t *testing.T) {
	var items []InboxItem
	var want []string
	n := 0
	for _, state := range []string{InboxStateOpen, InboxStateAcked, InboxStateResolved} {
		for _, woken := range []bool{false, true} {
			for _, delivery := range []string{InboxDeliveryNow, InboxDeliveryDeadline} {
				n++
				id := fmt.Sprintf("e%d", n)
				items = append(items, InboxItem{ID: id, State: state, Woken: woken, Delivery: delivery})
				if state == InboxStateOpen && !woken {
					want = append(want, id)
				}
			}
		}
	}
	if got := DirectorWakeItems(items); !slices.Equal(got, want) || !slices.Equal(want, []string{"e1", "e2"}) {
		t.Fatalf("DirectorWakeItems = %q, want %q (the open items with no woken event, both deliveries)", got, want)
	}
	if got := DirectorWakeItems(nil); got != nil {
		t.Fatalf("DirectorWakeItems(nil) = %q, want none", got)
	}
}

// AC5: a promoted session attached with wake pull,push and a target is woken
// for an open item by one run of wake_cmd, without a shell (the target's
// space, "$" and ";" reach the script as written), in the wake_cmd
// directory, with {target} and {text} replaced; the text names the item and
// holds nothing of its body. The item then has a woken event with the
// registration's id and via wake_cmd, and the next call runs nothing.
func TestWakeDirector_RunsTheWakeCmdOnceAndRecordsWoken(t *testing.T) {
	f := newWakeFixture(t)
	const target = "%orca 1;$HOME"
	reg := f.attachPush(t, target)
	const marker = "MARKER-body-7f3a"
	e1 := f.escalate(t, marker)

	res := f.wake()
	text := DirectorWakeText([]string{e1})
	want := DirectorWakeResult{RegistrationID: reg.ID, IDs: []string{e1}, Sent: true, Via: InboxWokenViaWakeCmd, Recorded: true}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("WakeDirector = %+v, want %+v", res, want)
	}
	runs := f.runs(t)
	if len(runs) != 1 || !slices.Equal(runs[0].args, []string{target, text}) || runs[0].pwd != wakeRealPath(t, f.cmdDir) {
		t.Fatalf("wake_cmd runs = %+v, want one with the args %q in %s", runs, []string{target, text}, f.cmdDir)
	}
	if strings.Contains(strings.Join(runs[0].args, "\n"), marker) || !strings.Contains(runs[0].args[1], e1) {
		t.Fatalf("the wake line %q holds the body or misses %s", runs[0].args[1], e1)
	}
	item := f.item(t, e1)
	last := item.Events[len(item.Events)-1]
	if !item.Woken || item.WokenAt != wakeTestNow.Format(time.RFC3339) || item.State != InboxStateOpen ||
		last.Event != InboxEventWoken || last.DirectorID != reg.ID || last.Via != InboxWokenViaWakeCmd {
		t.Fatalf("item %s = %+v, want it still open with a woken event by %s via wake_cmd", e1, item, reg.ID)
	}
	if f.banner.Len() != 0 {
		t.Fatalf("banner = %q, want nothing for a wake that worked", f.banner.String())
	}

	res = f.wake()
	if res.Skipped != DirectorWakeSkippedNothingToWake || res.Sent || len(f.runs(t)) != 1 {
		t.Fatalf("the next WakeDirector = %+v with %d runs, want nothing to wake and still one run", res, len(f.runs(t)))
	}
}

// AC5: the items of one call share one run: two items are named together,
// and of 21 the line names 20 and counts one more, while all 21 get woken.
func TestWakeDirector_OneRunNamesEveryNewItem(t *testing.T) {
	for _, n := range []int{2, 21} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f := newWakeFixture(t)
			f.attachPush(t, "term-1")
			for range n {
				f.escalate(t, "which one?")
			}
			res := f.wake()
			ids := wakeIDs(n)
			if !res.Sent || !res.Recorded || !slices.Equal(res.IDs, ids) {
				t.Fatalf("WakeDirector = %+v, want %d items sent and recorded", res, n)
			}
			runs := f.runs(t)
			if len(runs) != 1 || runs[0].args[1] != DirectorWakeText(ids) {
				t.Fatalf("wake_cmd runs = %+v, want one with the line for %d items", runs, n)
			}
			if n == 21 && !strings.Contains(runs[0].args[1], "e20 plus 1 more.") {
				t.Fatalf("the line for 21 items = %q, want 20 ids and plus 1 more", runs[0].args[1])
			}
			if got := f.wokenIDs(t); !slices.Equal(got, ids) {
				t.Fatalf("woken items = %q, want all %d", got, n)
			}
		})
	}
}

// AC5: nothing runs and nothing is written when the items are acked,
// resolved, or already woken, when no registration is current (none, or
// detached), and when the registration's wake has no push.
func TestWakeDirector_RunsNothingWithoutAnItemOrAPushRegistration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T, f *wakeFixture)
		skipped string
	}{
		{"an acked item", func(t *testing.T, f *wakeFixture) {
			f.attachPush(t, "term-1")
			if _, err := f.inbox.Ack(f.escalate(t, "x")); err != nil {
				t.Fatal(err)
			}
		}, DirectorWakeSkippedNothingToWake},
		{"a resolved item", func(t *testing.T, f *wakeFixture) {
			f.attachPush(t, "term-1")
			if err := f.inbox.Resolve(f.escalate(t, "x"), "done"); err != nil {
				t.Fatal(err)
			}
		}, DirectorWakeSkippedNothingToWake},
		{"an item already woken", func(t *testing.T, f *wakeFixture) {
			f.attachPush(t, "term-1")
			if err := f.inbox.AppendWoken([]string{f.escalate(t, "x")}, "d9", InboxWokenViaHerdr); err != nil {
				t.Fatal(err)
			}
		}, DirectorWakeSkippedNothingToWake},
		{"no item", func(t *testing.T, f *wakeFixture) { f.attachPush(t, "term-1") }, DirectorWakeSkippedNothingToWake},
		{"no registration", func(t *testing.T, f *wakeFixture) { f.escalate(t, "x") }, DirectorWakeSkippedNoRegistration},
		{"a detached registration", func(t *testing.T, f *wakeFixture) {
			reg := f.attachPush(t, "term-1")
			if _, err := f.store.Detach(DirectorDetach{ID: reg.ID}); err != nil {
				t.Fatal(err)
			}
			f.escalate(t, "x")
		}, DirectorWakeSkippedNoRegistration},
		{"wake pull", func(t *testing.T, f *wakeFixture) {
			mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull}})
			f.escalate(t, "x")
		}, DirectorWakeSkippedNoPush},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWakeFixture(t)
			tc.setup(t, f)
			before := fileOrAbsent(t, f.inbox.Path())
			res := f.wake()
			if res.Skipped != tc.skipped || res.Sent || res.Failure != nil || res.Err != nil || res.HeldBack {
				t.Fatalf("WakeDirector = %+v, want skipped %s", res, tc.skipped)
			}
			if runs := f.runs(t); runs != nil {
				t.Fatalf("wake_cmd ran %+v", runs)
			}
			if after := fileOrAbsent(t, f.inbox.Path()); after != before || f.banner.Len() != 0 {
				t.Fatalf("inbox.jsonl %q -> %q, banner %q; want both unchanged", before, after, f.banner.String())
			}
		})
	}
}

// AC5 and D6: a headless director is woken only while running. Started in
// herdr with push, it is not woken while starting; once its pane is recorded
// and it runs, wake_cmd runs with {target} its pane. A hand-edited stopped
// line without the detached that MarkStopped writes leaves it current but
// stopped, and it is not woken.
func TestWakeDirector_AHeadlessDirectorIsWokenOnlyWhileRunning(t *testing.T) {
	f := newWakeFixture(t)
	h := testHeadlessAttach()
	h.Wake = []string{DirectorWakePush}
	reg, err := f.store.AttachHeadless(h)
	if err != nil {
		t.Fatal(err)
	}
	e1 := f.escalate(t, "x")
	if res := f.wake(); res.Skipped != DirectorWakeSkippedNotRunning || res.RegistrationID != reg.ID || f.runs(t) != nil {
		t.Fatalf("WakeDirector while starting = %+v (runs %+v), want not running and no run", res, f.runs(t))
	}
	if err := f.store.RecordWorkspace(reg.ID, "w7", testDirectorWho); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RecordPane(reg.ID, "w7:p1", testDirectorWho); err != nil {
		t.Fatal(err)
	}
	if res := f.wake(); res.Skipped != DirectorWakeSkippedNotRunning || f.runs(t) != nil {
		t.Fatalf("WakeDirector with its pane, still starting = %+v, want not running", res)
	}
	if _, err := f.store.MarkLaunched(reg.ID, testDirectorWho); err != nil {
		t.Fatal(err)
	}
	res := f.wake()
	runs := f.runs(t)
	if !res.Sent || !res.Recorded || len(runs) != 1 || !slices.Equal(runs[0].args, []string{"w7:p1", DirectorWakeText([]string{e1})}) {
		t.Fatalf("WakeDirector while running = %+v (runs %+v), want one run with the pane as {target}", res, runs)
	}

	writeDirectorEvents(t, f.store,
		DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverCodex, Wake: []string{DirectorWakePush},
			Launch: DirectorLaunchHerdr, Token: directorTestToken, Model: "m", PermissionMode: PermissionModeGuarded, User: "u", Cwd: "/w"},
		DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventPaneCreated, PaneID: "w7:p1", User: "u", Cwd: "/w"},
		DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventLaunched, User: "u", Cwd: "/w"},
		DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventStopped, User: "u", Cwd: "/w"},
	)
	f.escalate(t, "y")
	if res := f.wake(); res.Skipped != DirectorWakeSkippedNotRunning || res.RegistrationID != "d1" || len(f.runs(t)) != 1 {
		t.Fatalf("WakeDirector of a current stopped registration = %+v (%d runs), want not running and no new run", res, len(f.runs(t)))
	}
}

// AC5 for a headless director in a terminal launch_cmd opened: its target is
// the terminal's name director_<token>.
func TestWakeDirector_ALaunchCmdDirectorIsWokenAtItsTerminalName(t *testing.T) {
	f := newWakeFixture(t)
	writeDirectorEvents(t, f.store,
		DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverCodex, Wake: []string{DirectorWakePush},
			Launch: DirectorLaunchCmd, Token: directorTestToken, Model: "m", PermissionMode: PermissionModeGuarded, User: "u", Cwd: "/w"},
		DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventLaunched, PID: 4242, StartTime: "Thu Oct 15 21:51:09 2026", User: "u", Cwd: "/w"},
	)
	e1 := f.escalate(t, "x")
	res := f.wake()
	runs := f.runs(t)
	if !res.Sent || len(runs) != 1 || !slices.Equal(runs[0].args, []string{"director_" + directorTestToken, DirectorWakeText([]string{e1})}) {
		t.Fatalf("WakeDirector = %+v (runs %+v), want one run with director_%s as {target}", res, runs, directorTestToken)
	}
}

// D9: a relative program is resolved from the wake_cmd directory, never from
// another one: of two directories, each with a wake.sh of its own, only the
// one WakeDirector is given runs, with that directory as its working
// directory. This process's working directory has neither.
func TestWakeDirector_ARelativeProgramIsResolvedFromTheWakeCmdDir(t *testing.T) {
	f := newWakeFixture(t)
	f.o.Config.Director.WakeCmd = []string{"./wake.sh", "{text}"}
	other, otherMark := t.TempDir(), filepath.Join(t.TempDir(), "other-runs")
	writeWakeScript(t, filepath.Join(other, "wake.sh"), otherMark, "exit 0")
	f.attachPush(t, "")
	e1 := f.escalate(t, "x")

	if res := f.wake(); !res.Sent {
		t.Fatalf("WakeDirector in %s = %+v, want sent", f.cmdDir, res)
	}
	if runs := f.runs(t); len(runs) != 1 || runs[0].pwd != wakeRealPath(t, f.cmdDir) || !slices.Equal(runs[0].args, []string{DirectorWakeText([]string{e1})}) {
		t.Fatalf("runs of %s/wake.sh = %+v, want one in that directory", f.cmdDir, runs)
	}
	if runs := readWakeRuns(t, otherMark); runs != nil {
		t.Fatalf("the other wake.sh ran: %+v", runs)
	}

	e2 := f.escalate(t, "y")
	if res := f.o.WakeDirector(f.now, f.dir, other, &f.mem, &f.banner); !res.Sent {
		t.Fatalf("WakeDirector in %s = %+v, want sent", other, res)
	}
	if runs := readWakeRuns(t, otherMark); len(runs) != 1 || runs[0].pwd != wakeRealPath(t, other) || !slices.Equal(runs[0].args, []string{DirectorWakeText([]string{e2})}) {
		t.Fatalf("runs of %s/wake.sh = %+v, want one in that directory for %s", other, runs, e2)
	}
	if n := len(f.runs(t)); n != 1 {
		t.Fatalf("%s/wake.sh ran %d times, want still once", f.cmdDir, n)
	}
}

// D9 and AC4: a program whose name has no "/" is looked up in PATH, not
// resolved from the wake_cmd directory. With wake_cmd ["wakeprobe",
// "{text}"], the wakeprobe of a directory first on PATH runs, with the
// wake_cmd directory as its working directory, and the wakeprobe in the
// wake_cmd directory itself does not run.
func TestWakeDirector_ABareProgramNameIsLookedUpInPath(t *testing.T) {
	f := newWakeFixture(t)
	bin, cmdDirMark := t.TempDir(), filepath.Join(t.TempDir(), "cmd-dir-runs")
	writeWakeScript(t, filepath.Join(bin, "wakeprobe"), f.mark, "exit 0")
	writeWakeScript(t, filepath.Join(f.cmdDir, "wakeprobe"), cmdDirMark, "exit 0")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	f.o.Config.Director.WakeCmd = []string{"wakeprobe", "{text}"}
	f.attachPush(t, "")
	e1 := f.escalate(t, "x")

	if res := f.wake(); !res.Sent || !res.Recorded {
		t.Fatalf("WakeDirector = %+v (banner %q), want sent and recorded", res, f.banner.String())
	}
	if runs := f.runs(t); len(runs) != 1 || runs[0].pwd != wakeRealPath(t, f.cmdDir) || !slices.Equal(runs[0].args, []string{DirectorWakeText([]string{e1})}) {
		t.Fatalf("runs of %s/wakeprobe = %+v, want one in %s", bin, runs, f.cmdDir)
	}
	if runs := readWakeRuns(t, cmdDirMark); runs != nil {
		t.Fatalf("the wakeprobe of the wake_cmd directory ran: %+v", runs)
	}
}

// AC7: a wake_cmd that exits non-zero, or does not return within
// DriverCallTimeout, writes no woken event and one line, and the
// registration is not tried again until 5 minutes have passed on the clock;
// then it is, and a wake that works records the items.
func TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tail    func(t *testing.T) string
		timeout time.Duration
		reason  string
	}{
		{"a non-zero exit", func(*testing.T) string { return "exit 1" }, 0, "failed: exit status 1"},
		// 3 s, not 1 s: the script records its run before it sleeps, and
		// under load sh can take more than a second to get there.
		{"no return in time", func(t *testing.T) string { return "exec " + shellQuote(sleepPath(t)) + " 30" }, 3 * time.Second,
			"did not return in time and was killed (context deadline exceeded)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWakeFixture(t)
			f.o.DriverCallTimeout = tc.timeout
			f.script(t, tc.tail(t))
			f.attachPush(t, "term-1")
			e1 := f.escalate(t, "x")
			wantLine := "watchdog: director d1 not woken for " + e1 + ": wake_cmd " + filepath.Join(f.cmdDir, "wake.sh") + " " + tc.reason

			res := f.wake()
			if res.Sent || res.Recorded || res.Failure == nil || !strings.Contains(res.Failure.Error(), tc.reason) {
				t.Fatalf("WakeDirector = %+v, want a failure that says %q", res, tc.reason)
			}
			if lines := f.bannerLines(); len(lines) != 1 || lines[0] != wantLine {
				t.Fatalf("banner lines = %q, want [%q]", lines, wantLine)
			}
			if woken := f.wokenIDs(t); woken != nil || len(f.runs(t)) != 1 {
				t.Fatalf("woken %q after %d runs, want none after one", woken, len(f.runs(t)))
			}

			f.now = wakeTestNow.Add(5*time.Minute - time.Second)
			if res := f.wake(); !res.HeldBack || res.Sent || res.Failure != nil {
				t.Fatalf("WakeDirector 4m59s later = %+v, want held back", res)
			}
			if n, lines := len(f.runs(t)), f.bannerLines(); n != 1 || len(lines) != 1 {
				t.Fatalf("4m59s later: %d runs and lines %q, want still one of each", n, lines)
			}

			f.now = wakeTestNow.Add(5 * time.Minute)
			if res := f.wake(); res.HeldBack || res.Failure == nil {
				t.Fatalf("WakeDirector 5m later = %+v, want tried again and failed", res)
			}
			if n, lines := len(f.runs(t)), f.bannerLines(); n != 2 || len(lines) != 2 || lines[1] != wantLine {
				t.Fatalf("5m later: %d runs and lines %q, want two of each", n, lines)
			}

			f.script(t, "exit 0")
			f.now = wakeTestNow.Add(10 * time.Minute)
			if res := f.wake(); !res.Sent || !res.Recorded {
				t.Fatalf("WakeDirector 10m later with a wake_cmd that works = %+v, want sent and recorded", res)
			}
			if woken := f.wokenIDs(t); !slices.Equal(woken, []string{e1}) {
				t.Fatalf("woken = %q, want %s", woken, e1)
			}
		})
	}
}

// D8: the failure line keeps only the end of wake_cmd's output, at most 512
// bytes of it, escaped, on one line.
func TestWakeDirector_AFailureLineKeepsTheEndOfTheOutputEscaped(t *testing.T) {
	f := newWakeFixture(t)
	f.script(t, "head -c 2000 /dev/zero | tr '\\0' a; printf '\\033[2J the-end\\n' >&2; exit 3")
	f.attachPush(t, "term-1")
	f.escalate(t, "x")
	f.wake()
	lines := f.bannerLines()
	if len(lines) != 1 || !strings.HasSuffix(lines[0], `failed: exit status 3; its output ended with: `+strings.Repeat("a", 512-len("\x1b[2J the-end\n"))+`\x1b[2J the-end`) {
		t.Fatalf("banner lines = %q, want one ending with the last 512 bytes of the output, escaped", lines)
	}
	if strings.Contains(f.banner.String(), "\x1b") {
		t.Fatalf("banner %q holds a raw escape", f.banner.String())
	}
}

// AC7 and D8: a target that is not usable (a hand-edited director.jsonl
// line) or missing while wake_cmd has {target} is said once for the
// registration, over cycles, and nothing runs.
func TestWakeDirector_AnUnusableTargetIsSaidOnceAndNothingRuns(t *testing.T) {
	for _, tc := range []struct {
		name, target, line string
	}{
		{"a control character", "t\x1b[2J", `watchdog: director d1 is not woken: [org.director].wake_cmd has {target}, but the wake target is not usable: ` +
			`"t\x1b[2J" has a control character (said once)`},
		{"a leading dash", "-x", `watchdog: director d1 is not woken: [org.director].wake_cmd has {target}, but the wake target is not usable: ` +
			`"-x" starts with "-", which the command [org.director].wake_cmd runs could read as an option (said once)`},
		{"no target", "", `watchdog: director d1 is not woken: [org.director].wake_cmd has {target}, but the wake target is not usable: it is empty (said once)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWakeFixture(t)
			writeDirectorEvents(t, f.store, DirectorEvent{
				TS: directorTestNow, ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverClaude, Wake: []string{DirectorWakePush},
				WakeTarget: tc.target, User: "u", Cwd: "/w",
			})
			f.escalate(t, "x")
			for i := range 3 {
				f.now = wakeTestNow.Add(time.Duration(i) * 10 * time.Minute)
				if res := f.wake(); res.Skipped != DirectorWakeSkippedTargetInvalid || res.Sent || res.Failure != nil {
					t.Fatalf("WakeDirector %d = %+v, want skipped for the target", i, res)
				}
			}
			if lines := f.bannerLines(); len(lines) != 1 || lines[0] != tc.line {
				t.Fatalf("banner lines = %q, want [%q]", lines, tc.line)
			}
			if runs := f.runs(t); runs != nil || f.wokenIDs(t) != nil {
				t.Fatalf("runs %+v, woken %q; want neither", runs, f.wokenIDs(t))
			}
		})
	}
}

// AC7 and D8: with an empty wake_cmd, nothing can type into a promoted
// session or into the terminal launch_cmd opened: nothing runs, and one line
// per registration says so; a registration that replaces it is said once
// too. A wake_cmd that is not valid (an Org built without config.Load) is
// said once and nothing runs.
func TestWakeDirector_NoWayToSendIsSaidOncePerRegistration(t *testing.T) {
	noSender := func(id, what string) string {
		return "watchdog: director " + id + " is not woken: its wake has push, but [org.director].wake_cmd is empty, and nothing else can type into " + what +
			"; its inbox items reach the human at the ack deadline (said once)"
	}
	const promoted = "a session attached with ralph org director attach"
	for _, tc := range []struct {
		name    string
		wakeCmd []string
		setup   func(t *testing.T, f *wakeFixture)
		skipped string
		lines   []string
	}{
		{"a promoted session", nil, func(t *testing.T, f *wakeFixture) { f.attachPush(t, "") }, DirectorWakeSkippedNoSender,
			[]string{noSender("d1", promoted)}},
		{"a launch_cmd director", nil, func(t *testing.T, f *wakeFixture) {
			writeDirectorEvents(t, f.store,
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverCodex, Wake: []string{DirectorWakePush},
					Launch: DirectorLaunchCmd, Token: directorTestToken, Model: "m", PermissionMode: PermissionModeGuarded, User: "u", Cwd: "/w"},
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventLaunched, PID: 4242, StartTime: "Thu Oct 15 21:51:09 2026", User: "u", Cwd: "/w"},
			)
		}, DirectorWakeSkippedNoSender, []string{noSender("d1", "a headless director in the terminal [org.director].launch_cmd opened")}},
		{"a wake_cmd whose program is a placeholder", []string{"{text}"}, func(t *testing.T, f *wakeFixture) { f.attachPush(t, "") }, DirectorWakeSkippedWakeCmdInvalid,
			[]string{`watchdog: director d1 is not woken: [org.director].wake_cmd: the first element is the command to run and must not contain a placeholder ` +
				`(a director's registration must never choose the program), got "{text}"; fix it and restart ralph org watch --all (said once)`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWakeFixture(t)
			f.o.Config.Director.WakeCmd = tc.wakeCmd
			tc.setup(t, f)
			f.escalate(t, "x")
			for i := range 3 {
				f.now = wakeTestNow.Add(time.Duration(i) * 10 * time.Minute)
				if res := f.wake(); res.Skipped != tc.skipped || res.Sent || res.Failure != nil || res.RegistrationID != "d1" {
					t.Fatalf("WakeDirector %d = %+v, want skipped %s for d1", i, res, tc.skipped)
				}
			}
			if lines := f.bannerLines(); !slices.Equal(lines, tc.lines) {
				t.Fatalf("banner lines =\n%q\nwant\n%q", lines, tc.lines)
			}
			if runs := f.runs(t); runs != nil || f.wokenIDs(t) != nil {
				t.Fatalf("runs %+v, woken %q; want neither", runs, f.wokenIDs(t))
			}
		})
	}

	f := newWakeFixture(t)
	f.o.Config.Director.WakeCmd = nil
	f.attachPush(t, "")
	f.escalate(t, "x")
	f.wake()
	f.wake()
	if _, _, err := f.store.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, Wake: []string{DirectorWakePush}}); err != nil {
		t.Fatal(err)
	}
	f.wake()
	f.wake()
	if lines, want := f.bannerLines(), []string{noSender("d1", promoted), noSender("d2", promoted)}; !slices.Equal(lines, want) {
		t.Fatalf("banner lines across a replace =\n%q\nwant\n%q", lines, want)
	}
}

// D9: with no wake_cmd directory, or a relative one, nothing runs and the
// wake fails.
func TestWakeDirector_WithoutAnAbsoluteDirectoryTheWakeCmdFails(t *testing.T) {
	for _, tc := range []struct {
		dir, reason string
	}{
		{"", "wake_cmd: no directory to run it in was given (the directory of the ralph.toml it was read from)"},
		{"cmd", `wake_cmd: the directory to run it in must be absolute (the directory of the ralph.toml it was read from), got "cmd"`},
	} {
		f := newWakeFixture(t)
		f.attachPush(t, "term-1")
		e1 := f.escalate(t, "x")
		res := f.o.WakeDirector(f.now, f.dir, tc.dir, &f.mem, &f.banner)
		want := "watchdog: director d1 not woken for " + e1 + ": " + tc.reason
		if res.Failure == nil || res.Sent || !slices.Equal(f.bannerLines(), []string{want}) || f.runs(t) != nil {
			t.Fatalf("WakeDirector with directory %q = %+v, banner %q, runs %+v; want the failure %q and no run", tc.dir, res, f.bannerLines(), f.runs(t), want)
		}
	}
}

// D6: when the woken events cannot be written after a send, that is said
// once and counts as a failure: the registration is held back for 5
// minutes, so a broken inbox is not woken every cycle, and then woken again.
func TestWakeDirector_AWokenWriteThatFailsHoldsTheRegistrationBack(t *testing.T) {
	f := newWakeFixture(t)
	f.attachPush(t, "term-1")
	e1 := f.escalate(t, "x")
	saved := readInboxFile(t, f.inbox)
	f.script(t, ": > "+shellQuote(f.inbox.Path()))

	res := f.wake()
	if !res.Sent || res.Recorded || !errors.Is(res.Failure, ErrInboxUnknownID) {
		t.Fatalf("WakeDirector = %+v, want sent, not recorded, with the inbox's unknown-id error", res)
	}
	want := "watchdog: director d1 was woken for " + e1 + ", but the woken events were not recorded, so it is not woken again for 5m0s: " +
		"org: inbox: wake " + e1 + ": no such inbox item"
	if lines := f.bannerLines(); !slices.Equal(lines, []string{want}) {
		t.Fatalf("banner lines = %q, want [%q]", lines, want)
	}

	writeInboxFile(t, f.inbox, saved)
	f.script(t, "exit 0")
	f.now = wakeTestNow.Add(time.Minute)
	if res := f.wake(); !res.HeldBack || len(f.runs(t)) != 1 {
		t.Fatalf("WakeDirector 1m later = %+v (%d runs), want held back", res, len(f.runs(t)))
	}
	f.now = wakeTestNow.Add(5 * time.Minute)
	if res := f.wake(); !res.Sent || !res.Recorded || len(f.runs(t)) != 2 {
		t.Fatalf("WakeDirector 5m later = %+v (%d runs), want sent and recorded", res, len(f.runs(t)))
	}
}

// D5: a registration that replaces one whose wake failed is not held back by
// that failure, since it has a new id, and items already woken are not woken
// again for the next one.
func TestWakeDirector_AReplacedRegistrationIsNotHeldBackAndDoesNotWakeAgain(t *testing.T) {
	f := newWakeFixture(t)
	f.script(t, "exit 1")
	f.attachPush(t, "term-1")
	e1 := f.escalate(t, "x")
	if res := f.wake(); res.Failure == nil {
		t.Fatalf("WakeDirector of d1 = %+v, want a failure", res)
	}
	replace := func(target string) {
		t.Helper()
		if _, _, err := f.store.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, Wake: []string{DirectorWakePush}, WakeTarget: target}); err != nil {
			t.Fatal(err)
		}
	}
	replace("term-2")
	f.script(t, "exit 0")
	f.now = wakeTestNow.Add(time.Minute)
	res := f.wake()
	if res.RegistrationID != "d2" || !res.Sent || !res.Recorded {
		t.Fatalf("WakeDirector of d2 a minute after d1 failed = %+v, want d2 sent and recorded", res)
	}
	if runs := f.runs(t); len(runs) != 2 || runs[1].args[0] != "term-2" {
		t.Fatalf("runs = %+v, want d1's and then d2's to term-2", runs)
	}
	item := f.item(t, e1)
	if last := item.Events[len(item.Events)-1]; last.DirectorID != "d2" {
		t.Fatalf("the woken event of %s = %+v, want d2's", e1, last)
	}
	replace("term-3")
	if res := f.wake(); res.RegistrationID != "d3" || res.Skipped != DirectorWakeSkippedNothingToWake || len(f.runs(t)) != 2 {
		t.Fatalf("WakeDirector of d3 = %+v, want nothing to wake: %s was woken", res, e1)
	}
}

// An item acked while the wake runs is still recorded as woken: the woken
// event records that the wake happened, in any state.
func TestWakeDirector_AnItemAckedDuringTheWakeIsStillRecordedAsWoken(t *testing.T) {
	f := newWakeFixture(t)
	f.attachPush(t, "term-1")
	e1 := f.escalate(t, "x")
	acked := fmt.Sprintf(`{"ts":%q,"id":%q,"event":"acked"}`, wakeTestNow.Format(time.RFC3339), e1)
	f.script(t, "printf '%s\\n' "+shellQuote(acked)+" >> "+shellQuote(f.inbox.Path()))
	if res := f.wake(); !res.Sent || !res.Recorded {
		t.Fatalf("WakeDirector = %+v, want sent and recorded", res)
	}
	if item := f.item(t, e1); item.State != InboxStateAcked || !item.Woken {
		t.Fatalf("item %s = %+v, want acked and woken", e1, item)
	}
}

// AC6: with no wake_cmd, a headless director in herdr is woken by input to
// its pane: the pane is confirmed by its tab's and its workspace's labels as
// stop confirms it, then the line is typed and Enter pressed, in that order,
// SendEnterDelay apart, and the items are woken via herdr. A label that
// differs, a pane herdr no longer has, and a send-text or an Enter that fails
// are each a failure with no woken event, and nothing is typed into a pane
// that is not confirmed; herdr's error text reaches the line escaped. With a
// wake_cmd, that runs with the pane as {target} and herdr is not called.
func TestWakeDirector_TypesIntoTheHerdrPaneOfAHeadlessDirector(t *testing.T) {
	start := func(t *testing.T) (*wakeFixture, *wakeTextHerdr, DirectorRegistration) {
		t.Helper()
		f := newWakeFixture(t)
		f.o.Config.Director.Wake = []string{DirectorWakePush}
		f.o.Config.Director.WakeCmd = nil
		reg := f.mustStart(t, DirectorDriverClaude, "opus").Registration
		h := &wakeTextHerdr{directorHerdr: f.h}
		f.o.Herdr = h
		f.h.calls, f.h.sendKeysCalls, f.h.sendKeysKeys = nil, nil, nil
		return f, h, reg
	}

	f, h, reg := start(t)
	const enterDelay = 60 * time.Millisecond
	f.o.SendEnterDelay = enterDelay
	e1 := f.escalate(t, "x")
	res := f.wake()
	if want := (DirectorWakeResult{RegistrationID: reg.ID, IDs: []string{e1}, Sent: true, Via: InboxWokenViaHerdr, Recorded: true}); !reflect.DeepEqual(res, want) {
		t.Fatalf("WakeDirector = %+v, want %+v", res, want)
	}
	if len(h.textAt) != 1 || len(h.keysAt) != 1 || h.keysAt[0].Sub(h.textAt[0]) < enterDelay {
		t.Fatalf("send-text at %v, send-keys at %v; want Enter at least %v after the line", h.textAt, h.keysAt, enterDelay)
	}
	if want := []string{"pane_get", "tab_get", "workspace_get", "pane_send_text", "pane_send_keys"}; !slices.Equal(f.h.calls, want) {
		t.Fatalf("herdr calls = %q, want %q", f.h.calls, want)
	}
	if want := []paneSendTextCall{{paneID: reg.PaneID, text: DirectorWakeText([]string{e1})}}; !slices.Equal(h.texts, want) {
		t.Fatalf("send-text = %+v, want %+v", h.texts, want)
	}
	if !slices.Equal(f.h.sendKeysCalls, []string{reg.PaneID}) || len(f.h.sendKeysKeys) != 1 || !slices.Equal(f.h.sendKeysKeys[0], []string{"Enter"}) {
		t.Fatalf("send-keys = %q %q, want Enter once to %s", f.h.sendKeysCalls, f.h.sendKeysKeys, reg.PaneID)
	}
	if item := f.item(t, e1); !item.Woken || item.Events[len(item.Events)-1].Via != InboxWokenViaHerdr || f.runs(t) != nil {
		t.Fatalf("item %s = %+v (runs %+v), want woken via herdr and no wake_cmd run", e1, item, f.runs(t))
	}

	for _, tc := range []struct {
		name   string
		break_ func(f *wakeFixture, reg DirectorRegistration)
		calls  []string
		reason string
	}{
		{"a tab label that differs", func(f *wakeFixture, reg DirectorRegistration) { f.h.tabLabels[fakeTabID(reg.PaneID)] = "seat-x" },
			[]string{"pane_get", "tab_get"}, `pane "pane-1" is not confirmed as d1's`},
		{"a workspace label that differs", func(f *wakeFixture, reg DirectorRegistration) { f.h.workspaceLabels[reg.WorkspaceID] = "org-x" },
			[]string{"pane_get", "tab_get", "workspace_get"}, `pane "pane-1" is not confirmed as d1's`},
		{"a pane herdr no longer has", func(f *wakeFixture, reg DirectorRegistration) { delete(f.h.paneTabs, reg.PaneID) },
			[]string{"pane_get"}, `herdr has no pane "pane-1", so nothing was typed into it`},
		{"a send-text that fails", func(f *wakeFixture, _ DirectorRegistration) {
			f.h.paneSendTextErr = errors.New("herdr: send-text failed \x1b[2J")
		},
			[]string{"pane_get", "tab_get", "workspace_get", "pane_send_text"}, `herdr pane send-text "pane-1": herdr: send-text failed`},
		{"an Enter that fails", func(f *wakeFixture, _ DirectorRegistration) {
			f.h.paneSendKeysErr = errors.New("herdr: send-keys failed")
		},
			[]string{"pane_get", "tab_get", "workspace_get", "pane_send_text", "pane_send_keys"},
			`herdr pane send-keys "pane-1" Enter, after the line was typed: herdr: send-keys failed`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, h, reg := start(t)
			tc.break_(f, reg)
			e1 := f.escalate(t, "x")
			res := f.wake()
			if res.Sent || res.Failure == nil || !strings.Contains(res.Failure.Error(), tc.reason) {
				t.Fatalf("WakeDirector = %+v, want a failure with %q", res, tc.reason)
			}
			if !slices.Equal(f.h.calls, tc.calls) {
				t.Fatalf("herdr calls = %q, want %q", f.h.calls, tc.calls)
			}
			if typed := slices.Contains(tc.calls, "pane_send_text"); typed != (len(h.texts) == 1) {
				t.Fatalf("send-text calls %+v; want one only when the pane was confirmed", h.texts)
			}
			lines := f.bannerLines()
			if len(lines) != 1 || !strings.HasPrefix(lines[0], "watchdog: director d1 not woken for "+e1+": ") || !strings.Contains(lines[0], tc.reason) {
				t.Fatalf("banner lines = %q, want one failure line with %q", lines, tc.reason)
			}
			if strings.Contains(lines[0], "\x1b") || (tc.name == "a send-text that fails" && !strings.HasSuffix(lines[0], `send-text failed \x1b[2J`)) {
				t.Fatalf("banner line %q holds a raw escape, or misses herdr's error escaped", lines[0])
			}
			if woken := f.wokenIDs(t); woken != nil {
				t.Fatalf("woken = %q, want none", woken)
			}
		})
	}

	t.Run("with a wake_cmd", func(t *testing.T) {
		f, h, reg := start(t)
		f.o.Config.Director.WakeCmd = []string{"./wake.sh", "{target}", "{text}"}
		e1 := f.escalate(t, "x")
		res := f.wake()
		runs := f.runs(t)
		if !res.Sent || res.Via != InboxWokenViaWakeCmd || len(runs) != 1 || !slices.Equal(runs[0].args, []string{reg.PaneID, DirectorWakeText([]string{e1})}) {
			t.Fatalf("WakeDirector = %+v (runs %+v), want wake_cmd run with the pane %s as {target}", res, runs, reg.PaneID)
		}
		if len(f.h.calls) != 0 || len(h.texts) != 0 {
			t.Fatalf("herdr calls = %q, send-text %+v; want none", f.h.calls, h.texts)
		}
	})
}

// AC6 and D8: with no wake_cmd, a headless director in herdr whose recorded
// pane id is not one to type into (a hand-edited pane_created: one that
// starts with "-", one with an escape sequence) is skipped and said once
// over cycles, and herdr is not called, not even to confirm the pane.
func TestWakeDirector_AnUnusableHerdrPaneIsSkippedWithoutCallingHerdr(t *testing.T) {
	for _, tc := range []struct {
		name, pane string
	}{
		{"a leading dash", "-x"},
		{"a control character", "p\x1b[2J"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWakeFixture(t)
			f.o.Config.Director.WakeCmd = nil
			writeDirectorEvents(t, f.store,
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverCodex, Wake: []string{DirectorWakePush},
					Launch: DirectorLaunchHerdr, Token: directorTestToken, Model: "m", PermissionMode: PermissionModeGuarded, User: "u", Cwd: "/w"},
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventPaneCreated, PaneID: tc.pane, User: "u", Cwd: "/w"},
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventLaunched, User: "u", Cwd: "/w"},
			)
			if cur := mustReadDirector(t, f.store).Current; cur == nil || cur.State != DirectorStateRunning || cur.PaneID != tc.pane {
				t.Fatalf("registration = %+v, want d1 running with pane %q", cur, tc.pane)
			}
			f.escalate(t, "x")
			f.h.calls = nil
			for i := range 2 {
				f.now = wakeTestNow.Add(time.Duration(i) * 10 * time.Minute)
				if res := f.wake(); res.Skipped != DirectorWakeSkippedTargetInvalid || res.Sent || res.Failure != nil || res.RegistrationID != "d1" {
					t.Fatalf("WakeDirector %d = %+v, want skipped %s for d1", i, res, DirectorWakeSkippedTargetInvalid)
				}
			}
			if len(f.h.calls) != 0 {
				t.Fatalf("herdr calls = %q, want none", f.h.calls)
			}
			lines := f.bannerLines()
			if len(lines) != 1 || !strings.HasPrefix(lines[0], "watchdog: director d1 is not woken: its herdr pane is not one to type into: ") ||
				!strings.HasSuffix(lines[0], " (said once)") {
				t.Fatalf("banner lines = %q, want one line for the pane, said once", lines)
			}
			if strings.Contains(f.banner.String(), "\x1b") {
				t.Fatalf("banner %q holds a raw escape", f.banner.String())
			}
			if runs := f.runs(t); runs != nil || f.wokenIDs(t) != nil {
				t.Fatalf("runs %+v, woken %q; want neither", runs, f.wokenIDs(t))
			}
		})
	}
}
