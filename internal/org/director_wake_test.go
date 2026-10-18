package org

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Tests for the wake of a director's registration (plan
// docs/plans/active/2026-10-18-org-director-wake.md, stage 8b-1: D3, D4,
// AC2, and AC3 in the org layer): the wake and wake_target an attached
// records, the shape of a wake target, attach's check that a wake can be
// sent, the target a registration resolves to, and the wake start records.
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
