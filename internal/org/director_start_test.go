package org

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// directorHerdr is a fakeHerdr whose AgentStart also records the kind, the
// pane, herdr's timeout, and the time left on its context, and runs
// agentStartHook (when set) before it answers, so a test can change
// director.jsonl while start waits on herdr (AC4).
type directorHerdr struct {
	*fakeHerdr
	agentStartKinds    []string
	agentStartPanes    []string
	agentStartTimeouts []int
	agentStartLeft     []time.Duration // -1 when the context had no deadline
	agentStartHook     func()
}

func (h *directorHerdr) AgentStart(ctx context.Context, name, kind, paneID string, timeoutMS int, agentArgs []string) (string, error) {
	left := time.Duration(-1)
	if d, ok := ctx.Deadline(); ok {
		left = time.Until(d)
	}
	h.agentStartKinds = append(h.agentStartKinds, kind)
	h.agentStartPanes = append(h.agentStartPanes, paneID)
	h.agentStartTimeouts = append(h.agentStartTimeouts, timeoutMS)
	h.agentStartLeft = append(h.agentStartLeft, left)
	if h.agentStartHook != nil {
		h.agentStartHook()
	}
	return h.fakeHerdr.AgentStart(ctx, name, kind, paneID, timeoutMS, agentArgs)
}

// dirStartFixture is one ledger for the StartDirector and StopDirector
// tests: an Org from testOrg with [org.director].enabled set, the director
// store in the manifest's directory, and a pane cwd outside that directory.
type dirStartFixture struct {
	o     *Org
	h     *directorHerdr
	store *DirectorStore
	dir   string            // the ledger: where the manifest and director.jsonl live
	cwd   string            // the pane's cwd, outside the ledger
	env   map[string]string // what o.Getenv reads; empty means not in herdr
}

// newDirStartFixture builds a fixture whose AttachHeadless hands out tokens
// in order; with none, the store makes real random tokens.
func newDirStartFixture(t *testing.T, tokens ...string) *dirStartFixture {
	t.Helper()
	o, fh, _ := testOrg(t)
	f := &dirStartFixture{o: o, h: &directorHerdr{fakeHerdr: fh}, dir: filepath.Dir(o.Manifest.Path()), cwd: t.TempDir(), env: map[string]string{}}
	o.Herdr = f.h
	o.Config.Director.Enabled = true
	o.Getenv = func(key string) string { return f.env[key] }
	o.AgentStartRetryInterval = time.Millisecond
	f.store = NewDirectorStore(f.dir)
	clock := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	f.store.Now = func() time.Time { return clock }
	if len(tokens) > 0 {
		f.store.token = func() (string, error) {
			if len(tokens) == 0 {
				t.Fatal("AttachHeadless asked for more launch tokens than the test gave")
			}
			token := tokens[0]
			tokens = tokens[1:]
			return token, nil
		}
	}
	return f
}

func (f *dirStartFixture) params(driver, model string) DirectorStartParams {
	return DirectorStartParams{Store: f.store, Driver: driver, Model: model, Cwd: f.cwd, TimeoutMS: 5000, Who: testDirectorWho}
}

func (f *dirStartFixture) mustStart(t *testing.T, driver, model string) DirectorStartResult {
	t.Helper()
	res, err := f.o.StartDirector(f.params(driver, model))
	if err != nil {
		t.Fatalf("StartDirector(%s, %s): %v", driver, model, err)
	}
	return res
}

func (f *dirStartFixture) stopParams(id string) DirectorStopParams {
	return DirectorStopParams{Store: f.store, ID: id, Who: testDirectorWho}
}

func (f *dirStartFixture) stop(id string) (DirectorStopResult, error) {
	return f.o.StopDirector(f.stopParams(id))
}

// directorTrail is director.jsonl as "<event> <id>" per line ("+force" for a
// forced detached); nil when the file is missing.
func directorTrail(t *testing.T, s *DirectorStore) []string {
	t.Helper()
	raw, err := os.ReadFile(s.Path())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read director.jsonl: %v", err)
	}
	var trail []string
	for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		var ev DirectorEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("director.jsonl line %q: %v", line, err)
		}
		entry := ev.Event + " " + ev.ID
		if ev.Force {
			entry += "+force"
		}
		trail = append(trail, entry)
	}
	return trail
}

// directorFileState is director.jsonl's content, or "<missing>".
func directorFileState(t *testing.T, s *DirectorStore) string {
	t.Helper()
	raw, err := os.ReadFile(s.Path())
	if errors.Is(err, fs.ErrNotExist) {
		return "<missing>"
	}
	if err != nil {
		t.Fatalf("read director.jsonl: %v", err)
	}
	return string(raw)
}

func wantTrail(t *testing.T, s *DirectorStore, want ...string) {
	t.Helper()
	if got := directorTrail(t, s); !slices.Equal(got, want) {
		t.Fatalf("director.jsonl events = %q, want %q", got, want)
	}
}

// wantNoDirectorOpen asserts no registration is current and none is open.
func wantNoDirectorOpen(t *testing.T, s *DirectorStore) {
	t.Helper()
	res := mustReadDirector(t, s)
	if res.Current != nil || len(res.OpenHeadless) != 0 {
		t.Fatalf("expected no current and no open headless registration, got current %+v, open %q", res.Current, openHeadlessSummary(res))
	}
}

func herdrCallsSince(h *directorHerdr, n int) []string {
	return slices.Clone(h.calls[n:])
}

// directorCloseCalls are the herdr calls a full close of a confirmed pane
// and workspace makes: the checks of both, then C-c and the two closes.
var directorCloseCalls = []string{"pane_get", "tab_get", "workspace_get", "workspace_get", "pane_send_keys", "pane_close", "workspace_close"}

// TestStartDirector_LaunchesInAWorkspaceOfItsOwn pins AC1 for both drivers:
// the registration comes first, then the workspace and the tab labelled with
// the launch token, each recorded as herdr hands it out, then the agent
// start with the permission flags, codex's --add-dir for a ledger outside
// the cwd, --model, and the prompt file's pointer in that order, then
// launched. Nothing but director.jsonl, its lock, and the prompt file is
// written: no manifest, no model receipt.
func TestStartDirector_LaunchesInAWorkspaceOfItsOwn(t *testing.T) {
	const token = directorTestToken
	cases := []struct {
		driver, model string
		verified      bool
		wantPerm      func(f *dirStartFixture) []string
	}{
		{DirectorDriverClaude, "opus", false, func(*dirStartFixture) []string { return []string{"--permission-mode", "bypassPermissions"} }},
		{DirectorDriverCodex, "gpt-5-codex", true, func(f *dirStartFixture) []string {
			return append(slices.Clone(codexAutonomousArgs), "--add-dir", f.dir)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.driver, func(t *testing.T) {
			f := newDirStartFixture(t, token)
			f.o.Config.Permissions.CodexVerified = tc.verified
			res := f.mustStart(t, tc.driver, tc.model)

			reg := res.Registration
			if reg.ID != "d1" || reg.State != DirectorStateRunning || reg.Token != token || reg.Driver != tc.driver || reg.Model != tc.model ||
				reg.PermissionMode != PermissionModeAutonomous || reg.WorkspaceID != "ws-1" || reg.PaneID != "pane-1" || !reg.Headless() {
				t.Fatalf("Registration = %+v, want d1 running with ws-1, pane-1, the token, model, and autonomous", reg)
			}
			wantPrompt := filepath.Join(f.dir, "prompts", "director_"+token+".md")
			if res.AgentName != "director_"+token || res.TabLabel != "director_"+token || res.WorkspaceLabel != "ralph_director_"+token ||
				res.PromptPath != wantPrompt || res.AgentStartRetries != 0 {
				t.Fatalf("result = %+v, want the token's names and prompt %s", res, wantPrompt)
			}
			wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1")
			if cur := mustReadDirector(t, f.store).Current; cur == nil || directorRegSummary(*cur) != "d1 running ws=ws-1 pane=pane-1" {
				t.Fatalf("current registration = %+v, want d1 running ws-1 pane-1", cur)
			}

			h := f.h
			if !slices.Equal(h.calls, []string{"workspace_create", "tab_create", "agent_start"}) {
				t.Fatalf("herdr calls = %v, want workspace_create, tab_create, agent_start", h.calls)
			}
			if h.workspaceLabels["ws-1"] != "ralph_director_"+token || h.tabLabels[fakeTabID("pane-1")] != "director_"+token {
				t.Fatalf("labels: workspace %q, tab %q, want the token's", h.workspaceLabels["ws-1"], h.tabLabels[fakeTabID("pane-1")])
			}
			if !slices.Equal(h.workspaceCreateCwds, []string{f.cwd}) || !slices.Equal(h.tabCreateCwds, []string{f.cwd}) || !slices.Equal(h.tabCreateWorkspaces, []string{"ws-1"}) {
				t.Fatalf("workspace cwds %v, tab cwds %v, tab workspaces %v, want the pane cwd and ws-1", h.workspaceCreateCwds, h.tabCreateCwds, h.tabCreateWorkspaces)
			}
			if !slices.Equal(h.agentStartNames, []string{"director_" + token}) || !slices.Equal(h.agentStartKinds, []string{tc.driver}) ||
				!slices.Equal(h.agentStartPanes, []string{"pane-1"}) || !slices.Equal(h.agentStartTimeouts, []int{5000}) {
				t.Fatalf("agent start: names %v kinds %v panes %v timeouts %v", h.agentStartNames, h.agentStartKinds, h.agentStartPanes, h.agentStartTimeouts)
			}
			if left := h.agentStartLeft[0]; left <= 0 || left > 5*time.Second {
				t.Fatalf("agent start context had %v left, want a deadline within the 5s timeout", left)
			}
			wantArgs := append(tc.wantPerm(f), "--model", tc.model, PromptFilePointer(wantPrompt))
			if !slices.Equal(h.agentStartArgs[0], wantArgs) {
				t.Fatalf("agent args = %q, want %q", h.agentStartArgs[0], wantArgs)
			}

			prompt, err := os.ReadFile(wantPrompt)
			if err != nil {
				t.Fatalf("read the prompt file: %v", err)
			}
			for _, want := range []string{"ralph org director status --state-dir " + shellQuote(f.dir), "ralph org director stop --id d1", "driver: " + tc.driver} {
				if !strings.Contains(string(prompt), want) {
					t.Fatalf("prompt file lacks %q:\n%s", want, prompt)
				}
			}
			if strings.Contains(string(prompt), "{{") {
				t.Fatalf("prompt file keeps a placeholder:\n%s", prompt)
			}
			var attached DirectorEvent
			if err := json.Unmarshal([]byte(directorFileLines(t, f.store)[0]), &attached); err != nil ||
				attached.Launch != DirectorLaunchHerdr || attached.Token != token || attached.Model != tc.model ||
				attached.PermissionMode != PermissionModeAutonomous || attached.PaneID != "" || attached.User != testDirectorWho.User {
				t.Fatalf("attached = %+v (%v), want launch herdr, the token, model, permission mode, and no pane", attached, err)
			}
			if got := dirEntryNames(t, f.dir); !slices.Equal(got, []string{"director.jsonl", "director.lock", "prompts"}) {
				t.Fatalf("ledger entries = %v, want director.jsonl, director.lock, and prompts only (no manifest, no receipts)", got)
			}
		})
	}
}

// TestStartDirector_PermissionModeOfRoleDirector pins AC6: roles.director
// wins over the default, each mode maps to the driver's flags as for a seat,
// and codex refuses autonomous and edits until codex_verified, with nothing
// written and no herdr call. A codex director gets --add-dir only when the
// ledger is outside its cwd.
func TestStartDirector_PermissionModeOfRoleDirector(t *testing.T) {
	type perms = config.OrgPermissionsConfig
	cases := []struct {
		name     string
		driver   string
		perms    perms
		inside   bool // the cwd holds the ledger
		wantMode string
		wantPerm []string // with --add-dir; nil and refused for a refusal
		refused  bool
	}{
		{"roles.director wins over the default", DirectorDriverClaude, perms{Default: PermissionModeGuarded, Roles: map[string]string{DirectorRole: PermissionModeEdits}}, false,
			PermissionModeEdits, []string{"--permission-mode", "acceptEdits"}, false},
		{"another role's entry does not apply", DirectorDriverClaude, perms{Default: PermissionModeGuarded, Roles: map[string]string{LeaderIdentity: PermissionModeAutonomous}}, false,
			PermissionModeGuarded, nil, false},
		{"claude autonomous", DirectorDriverClaude, perms{Default: PermissionModeAutonomous}, false, PermissionModeAutonomous, []string{"--permission-mode", "bypassPermissions"}, false},
		{"claude edits", DirectorDriverClaude, perms{Roles: map[string]string{DirectorRole: PermissionModeEdits}}, false, PermissionModeEdits, []string{"--permission-mode", "acceptEdits"}, false},
		{"claude guarded", DirectorDriverClaude, perms{Default: PermissionModeGuarded}, false, PermissionModeGuarded, nil, false},
		{"codex autonomous before codex_verified", DirectorDriverCodex, perms{Default: PermissionModeAutonomous}, false, "", nil, true},
		{"codex edits before codex_verified", DirectorDriverCodex, perms{Roles: map[string]string{DirectorRole: PermissionModeEdits}}, false, "", nil, true},
		{"codex guarded before codex_verified", DirectorDriverCodex, perms{Default: PermissionModeGuarded}, false, PermissionModeGuarded, nil, false},
		{"codex edits, ledger outside the cwd", DirectorDriverCodex, perms{Default: PermissionModeEdits, CodexVerified: true}, false,
			PermissionModeEdits, append(slices.Clone(codexEditsArgs), "--add-dir", "<ledger>"), false},
		{"codex autonomous, ledger inside the cwd", DirectorDriverCodex, perms{Default: PermissionModeAutonomous, CodexVerified: true}, true,
			PermissionModeAutonomous, slices.Clone(codexAutonomousArgs), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.o.Config.Permissions = tc.perms
			if tc.inside {
				f.cwd = filepath.Dir(f.dir)
			}
			res, err := f.o.StartDirector(f.params(tc.driver, "m1"))
			if tc.refused {
				if err == nil || !strings.Contains(err.Error(), "codex_verified") || !strings.Contains(err.Error(), "role director") {
					t.Fatalf("err = %v, want the codex_verified refusal for role director", err)
				}
				if len(f.h.calls) != 0 || directorFileState(t, f.store) != "<missing>" {
					t.Fatalf("a refusal called herdr %v or wrote director.jsonl", f.h.calls)
				}
				return
			}
			if err != nil {
				t.Fatalf("StartDirector: %v", err)
			}
			if res.Registration.PermissionMode != tc.wantMode {
				t.Fatalf("permission mode = %q, want %q", res.Registration.PermissionMode, tc.wantMode)
			}
			want := slices.Clone(tc.wantPerm)
			for i, a := range want {
				if a == "<ledger>" {
					want[i] = f.dir
				}
			}
			want = append(want, "--model", "m1", PromptFilePointer(res.PromptPath))
			if got := f.h.agentStartArgs[0]; !slices.Equal(got, want) {
				t.Fatalf("agent args = %q, want %q", got, want)
			}
		})
	}
}

// TestStartDirector_RefusesBeforeWritingOrCallingHerdr pins AC2's org-layer
// refusals: each writes nothing (director.jsonl as it was, no prompt file)
// and calls no herdr, and the input checks run before the watch gate. A
// current registration, a headless one still starting included, is refused
// with a *DirectorAttachedError that says whether it is headless, also
// before the watch gate, so with no watch running it is still the attached
// refusal and no watch is launched (self-review L1 of org-director-start).
func TestStartDirector_RefusesBeforeWritingOrCallingHerdr(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T, f *dirStartFixture, p *DirectorStartParams)
		wantGate  bool // the watch gate was asked
		wantErrIs error
		check     func(t *testing.T, err error)
	}{
		{"enabled is false", func(_ *testing.T, f *dirStartFixture, _ *DirectorStartParams) { f.o.Config.Director.Enabled = false }, false, ErrDirectorDisabled, nil},
		{"inside a herdr pane", func(_ *testing.T, f *dirStartFixture, _ *DirectorStartParams) { f.env[herdrPaneIDEnv] = "pane-9" }, false, ErrDirectorStartInsideHerdr,
			func(t *testing.T, err error) {
				if !strings.Contains(err.Error(), "outside herdr") {
					t.Fatalf("err = %v, want it to say to run it outside herdr", err)
				}
			}},
		{"unknown driver", func(_ *testing.T, _ *dirStartFixture, p *DirectorStartParams) { p.Driver = "gemini" }, false, nil, nil},
		{"empty model", func(_ *testing.T, _ *dirStartFixture, p *DirectorStartParams) { p.Model = "" }, false, nil, nil},
		{"empty cwd", func(_ *testing.T, _ *dirStartFixture, p *DirectorStartParams) { p.Cwd = "" }, false, nil, nil},
		{"no store", func(_ *testing.T, _ *dirStartFixture, p *DirectorStartParams) { p.Store = nil }, false, nil, nil},
		{"permission rule", func(_ *testing.T, f *dirStartFixture, p *DirectorStartParams) {
			p.Driver, f.o.Config.Permissions.Default = DirectorDriverCodex, PermissionModeAutonomous
		}, false, nil, nil},
		{"watch gate", func(_ *testing.T, f *dirStartFixture, _ *DirectorStartParams) {
			f.o.WatchGate = func() error {
				return &WatchGateError{StateDir: f.dir, Liveness: WatchLiveness{State: WatchLivenessNotRunning}}
			}
		}, true, ErrNoRunningWatch, nil},
		{"a promoted session is attached", func(t *testing.T, f *dirStartFixture, _ *DirectorStartParams) {
			mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverClaude, User: "u", Cwd: "/w"})
		}, false, ErrDirectorAttached, func(t *testing.T, err error) {
			var ae *DirectorAttachedError
			if !errors.As(err, &ae) || ae.Current == nil || ae.Current.ID != "d1" || ae.Current.Headless() {
				t.Fatalf("err = %#v, want a *DirectorAttachedError naming the promoted d1", err)
			}
		}},
		{"a headless registration is still starting", func(t *testing.T, f *dirStartFixture, _ *DirectorStartParams) {
			if _, err := f.store.AttachHeadless(testHeadlessAttach()); err != nil {
				t.Fatalf("AttachHeadless: %v", err)
			}
		}, false, ErrDirectorAttached, func(t *testing.T, err error) {
			var ae *DirectorAttachedError
			if !errors.As(err, &ae) || ae.Current == nil || !ae.Current.Headless() || ae.Current.State != DirectorStateStarting ||
				!strings.Contains(err.Error(), "headless starting") {
				t.Fatalf("err = %v, want a *DirectorAttachedError naming the starting headless d1", err)
			}
		}},
		{"a director is attached and no watch runs", func(t *testing.T, f *dirStartFixture, _ *DirectorStartParams) {
			f.o.WatchGate = func() error {
				return &WatchGateError{StateDir: f.dir, Liveness: WatchLiveness{State: WatchLivenessNotRunning}}
			}
			mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverCodex, User: "u", Cwd: "/w"})
		}, false, ErrDirectorAttached, func(t *testing.T, err error) {
			var ae *DirectorAttachedError
			if !errors.As(err, &ae) || ae.Current == nil || ae.Current.ID != "d1" || errors.Is(err, ErrNoRunningWatch) ||
				err.Error() != "org: director: start: a director is already attached: "+ae.Current.describe() {
				t.Fatalf("err = %v, want only AttachHeadless's refusal naming d1, not the watch gate's", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken, "ba0000000000")
			p := f.params(DirectorDriverClaude, "opus")
			tc.setup(t, f, &p)
			// The gate passes unless the case set one; either way the
			// wrapper counts that it was asked.
			gateCalls, inner := 0, f.o.WatchGate
			f.o.WatchGate = func() error {
				gateCalls++
				if inner != nil {
					return inner()
				}
				return nil
			}
			before := directorFileState(t, f.store)

			_, err := f.o.StartDirector(p)
			if err == nil {
				t.Fatal("StartDirector succeeded, want a refusal")
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("err = %v, want it to wrap %v", err, tc.wantErrIs)
			}
			if tc.check != nil {
				tc.check(t, err)
			}
			if (gateCalls > 0) != tc.wantGate {
				t.Fatalf("watch gate asked %d times, want asked = %v", gateCalls, tc.wantGate)
			}
			if len(f.h.calls) != 0 {
				t.Fatalf("a refusal called herdr: %v", f.h.calls)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed from %q to %q", before, after)
			}
			if _, err := os.Stat(filepath.Join(f.dir, "prompts")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("a refusal made the prompts dir (stat err %v)", err)
			}
		})
	}
}

// TestStartDirector_ARegistrationAfterTheFirstLookIsRefusedUnderTheLock pins
// that the first look at a current registration only refuses early: an
// attach that lands after it (here, while the watch gate runs) is still
// refused by AttachHeadless under the lock, with the registration read
// again, and start writes nothing and calls no herdr.
func TestStartDirector_ARegistrationAfterTheFirstLookIsRefusedUnderTheLock(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	f.o.WatchGate = func() error {
		mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverClaude, User: "u", Cwd: "/w"})
		return nil
	}
	_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
	var ae *DirectorAttachedError
	if !errors.As(err, &ae) || !errors.Is(err, ErrDirectorAttached) || ae.Current == nil || ae.Current.ID != "d1" || ae.Current.Headless() {
		t.Fatalf("err = %#v, want AttachHeadless's *DirectorAttachedError naming the promoted d1", err)
	}
	wantTrail(t, f.store, "attached d1")
	if len(f.h.calls) != 0 {
		t.Fatalf("herdr calls = %v, want none", f.h.calls)
	}
}

// TestStartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop pins AC3:
// when the workspace create, the tab create, the prompt write, or the agent
// start fails, start closes what it made (C-c only once there is a pane),
// writes stopped with detached, and returns the step's error. The ledger is
// free for the next start.
func TestStartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop(t *testing.T) {
	boom := errors.New("herdr: connection refused")
	cases := []struct {
		name      string
		inject    func(t *testing.T, f *dirStartFixture)
		wantBoom  bool // the error wraps boom
		wantCalls []string
		wantTrail []string
		wantText  string
	}{
		{"workspace create", func(_ *testing.T, f *dirStartFixture) { f.h.workspaceCreateErr = boom }, true,
			[]string{"workspace_create"}, []string{"attached d1", "detached d1", "stopped d1"}, "labelled ralph_director_" + directorTestToken},
		{"tab create", func(_ *testing.T, f *dirStartFixture) { f.h.tabCreateErr = boom }, true,
			[]string{"workspace_create", "tab_create", "workspace_get", "workspace_close"},
			[]string{"attached d1", "workspace_created d1", "detached d1", "stopped d1"}, "create the herdr tab"},
		{"prompt write", func(t *testing.T, f *dirStartFixture) {
			// A file where the prompts dir goes makes writePromptFile fail.
			if err := os.WriteFile(filepath.Join(f.dir, "prompts"), []byte("not a dir"), 0o644); err != nil {
				t.Fatalf("write the blocking file: %v", err)
			}
		}, false, append([]string{"workspace_create", "tab_create"}, directorCloseCalls...),
			[]string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1", "stopped d1"}, "prompt file"},
		{"agent start", func(_ *testing.T, f *dirStartFixture) { f.h.agentStartErr = boom }, true,
			append([]string{"workspace_create", "tab_create", "agent_start"}, directorCloseCalls...),
			[]string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1", "stopped d1"}, "herdr agent start director_" + directorTestToken},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken, "ba0000000000")
			tc.inject(t, f)
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			if err == nil {
				t.Fatal("StartDirector succeeded, want the step's failure")
			}
			if tc.wantBoom && !errors.Is(err, boom) {
				t.Fatalf("err = %v, want it to wrap the step's error", err)
			}
			var ce *DirectorCleanupError
			if errors.As(err, &ce) {
				t.Fatalf("err = %v, want a cleanup that finished", err)
			}
			for _, want := range []string{tc.wantText, "d1 is recorded stopped"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("err = %v, want it to say %q", err, want)
				}
			}
			if !slices.Equal(f.h.calls, tc.wantCalls) {
				t.Fatalf("herdr calls = %v, want %v", f.h.calls, tc.wantCalls)
			}
			wantTrail(t, f.store, tc.wantTrail...)
			wantNoDirectorOpen(t, f.store)

			// The ledger is free again: a new start registers d2.
			f.h.workspaceCreateErr, f.h.tabCreateErr, f.h.agentStartErr = nil, nil, nil
			_ = os.Remove(filepath.Join(f.dir, "prompts"))
			if res := f.mustStart(t, DirectorDriverClaude, "opus"); res.Registration.ID != "d2" {
				t.Fatalf("the next start registered %s, want d2", res.Registration.ID)
			}
		})
	}
}

// TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade pins the
// cleanup of an id herdr handed out that start could not record: it closes
// the workspace (and the pane) by the id it holds and records the stop.
func TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade(t *testing.T) {
	cases := []struct {
		name      string
		failWrite int // the write that fails: 1 attached, 2 workspace_created, 3 pane_created
		wantCalls []string
		wantTrail []string
	}{
		{"workspace_created", 2, []string{"workspace_create", "workspace_get", "workspace_close"}, []string{"attached d1", "detached d1", "stopped d1"}},
		{"pane_created", 3, append([]string{"workspace_create", "tab_create"}, directorCloseCalls...),
			[]string{"attached d1", "workspace_created d1", "detached d1", "stopped d1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			writes := 0
			f.store.write = func(file *os.File, b []byte) (int, error) {
				writes++
				if writes == tc.failWrite {
					return 0, syscall.ENOSPC
				}
				return file.Write(b)
			}
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			if !errors.Is(err, syscall.ENOSPC) || !strings.Contains(err.Error(), "d1 is recorded stopped") {
				t.Fatalf("err = %v, want the failed record and a finished cleanup", err)
			}
			if !slices.Equal(f.h.calls, tc.wantCalls) {
				t.Fatalf("herdr calls = %v, want %v", f.h.calls, tc.wantCalls)
			}
			if !slices.Equal(f.h.workspaceCloseCalls, []string{"ws-1"}) {
				t.Fatalf("workspace closes = %v, want ws-1, the id herdr handed out", f.h.workspaceCloseCalls)
			}
			wantTrail(t, f.store, tc.wantTrail...)
			wantNoDirectorOpen(t, f.store)
		})
	}
}

// TestStartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly pins AC3's
// second half: when start cannot close what it made, it writes detached only
// and returns a *DirectorCleanupError with the herdr commands and the stop
// command. A closed id is never offered as a herdr command when its labels
// were not confirmed. Once herdr answers, stop closes it.
func TestStartDirector_ACleanupThatCannotCloseLeavesItDetachedOnly(t *testing.T) {
	agentErr := errors.New("agent start: exited")
	t.Run("the close fails", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		closeErr := errors.New("herdr pane close: connection refused")
		f.h.agentStartErr = agentErr
		f.h.paneCloseErrs = map[string]error{"pane-1": closeErr}
		_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
		var ce *DirectorCleanupError
		if !errors.As(err, &ce) || !errors.Is(err, agentErr) || !errors.Is(err, closeErr) {
			t.Fatalf("err = %v, want a *DirectorCleanupError wrapping the agent start and the close failure", err)
		}
		if ce.Verb != "start" || ce.ID != "d1" || !ce.Detached || ce.Unconfirmed || ce.PaneID != "pane-1" || ce.WorkspaceID != "ws-1" ||
			ce.WorkspaceLabel != "ralph_director_"+directorTestToken || ce.TabLabel != "director_"+directorTestToken ||
			!slices.Equal(ce.HerdrCommands, []string{"herdr pane close pane-1", "herdr workspace close ws-1"}) ||
			ce.StopCommand != "ralph org director stop --id d1" {
			t.Fatalf("cleanup error = %+v", ce)
		}
		for _, want := range []string{"herdr pane close pane-1, then herdr workspace close ws-1", "detached only", "ralph org director stop --id d1"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("err = %v, want it to say %q", err, want)
			}
		}
		if len(f.h.workspaceCloseCalls) != 0 {
			t.Fatalf("the workspace was closed (%v) although its pane was left open", f.h.workspaceCloseCalls)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "detached d1")
		res := mustReadDirector(t, f.store)
		if res.Current != nil || !slices.Equal(openHeadlessSummary(res), []string{"d1 starting ws=ws-1 pane=pane-1"}) {
			t.Fatalf("current %+v, open %q, want none current and d1 open", res.Current, openHeadlessSummary(res))
		}

		delete(f.h.paneCloseErrs, "pane-1")
		stopped, err := f.stop("d1")
		if err != nil || stopped.WasCurrent {
			t.Fatalf("stop after herdr came back: %+v, %v; want a stop of a registration no longer current", stopped, err)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "detached d1", "stopped d1")
	})
	t.Run("the labels are not this launch's", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		f.h.agentStartErr = agentErr
		f.h.agentStartHook = func() { setLabel(&f.h.tabLabels, fakeTabID("pane-1"), "someone_else") }
		_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
		var ce *DirectorCleanupError
		if !errors.As(err, &ce) || !ce.Unconfirmed || !ce.Detached ||
			!slices.Equal(ce.HerdrCommands, []string{"herdr workspace list", "herdr workspace close <the id of the workspace labelled ralph_director_" + directorTestToken + ">"}) ||
			ce.StopCommand != "ralph org director stop --id d1 --force" {
			t.Fatalf("err = %#v, want an unconfirmed cleanup that finds the workspace by its label and records the stop with --force", err)
		}
		if len(f.h.sendKeysCalls) != 0 || len(f.h.paneCloseCalls) != 0 || len(f.h.workspaceCloseCalls) != 0 {
			t.Fatalf("an unconfirmed pane got C-c %v, pane closes %v, workspace closes %v", f.h.sendKeysCalls, f.h.paneCloseCalls, f.h.workspaceCloseCalls)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "detached d1")

		// The stop it names records d1 with force and still sends nothing.
		stopped, err := stopWithForce(f, "d1")
		if err != nil || !stopped.Forced || stopped.WasCurrent {
			t.Fatalf("stop --force: %+v, %v; want a forced stop of a registration no longer current", stopped, err)
		}
		if len(f.h.sendKeysCalls) != 0 || len(f.h.paneCloseCalls) != 0 || len(f.h.workspaceCloseCalls) != 0 {
			t.Fatalf("stop --force sent C-c %v, pane closes %v, workspace closes %v", f.h.sendKeysCalls, f.h.paneCloseCalls, f.h.workspaceCloseCalls)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "detached d1", "stopped d1+force")
	})
}

// TestStartDirector_AReplaceOrDetachWhileTheAgentStartsEndsItsOwnLaunch
// pins AC4: an attach --replace or a detach --force that lands while start
// waits on the agent start makes start close its own pane and workspace,
// write stopped and no launched, and fail with
// ErrDirectorReplacedDuringStart; what came in stays as it is.
func TestStartDirector_AReplaceOrDetachWhileTheAgentStartsEndsItsOwnLaunch(t *testing.T) {
	cases := []struct {
		name        string
		cut         func(t *testing.T, s *DirectorStore)
		wantTrail   []string
		wantCurrent string
		wantText    string
	}{
		{"attach --replace", func(t *testing.T, s *DirectorStore) {
			if _, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude, Replace: true, User: "carol", Cwd: "/c"}); err != nil {
				t.Errorf("Attach --replace: %v", err)
			}
		}, []string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1", "attached d2", "stopped d1"}, "d2", "the current registration is d2"},
		{"detach --force", func(t *testing.T, s *DirectorStore) {
			if _, err := s.Detach(DirectorDetach{Force: true, User: "carol", Cwd: "/c"}); err != nil {
				t.Errorf("Detach --force: %v", err)
			}
		}, []string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1+force", "stopped d1"}, "", "no registration is current"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.h.agentStartHook = func() { tc.cut(t, f.store) }
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			if !errors.Is(err, ErrDirectorReplacedDuringStart) || !strings.Contains(err.Error(), tc.wantText) || !strings.Contains(err.Error(), "d1 is recorded stopped") {
				t.Fatalf("err = %v, want ErrDirectorReplacedDuringStart saying %q and a finished cleanup", err, tc.wantText)
			}
			wantCalls := append([]string{"workspace_create", "tab_create", "agent_start"}, directorCloseCalls...)
			if !slices.Equal(f.h.calls, wantCalls) {
				t.Fatalf("herdr calls = %v, want %v", f.h.calls, wantCalls)
			}
			wantTrail(t, f.store, tc.wantTrail...)
			res := mustReadDirector(t, f.store)
			if currentDirectorID(res) != tc.wantCurrent || len(res.OpenHeadless) != 0 {
				t.Fatalf("current %q, open %q, want current %q and nothing open", currentDirectorID(res), openHeadlessSummary(res), tc.wantCurrent)
			}
		})
	}
}

// TestDirector_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration
// pins the crash after step 1: the registration reads starting, start and
// attach refuse on it, and stop closes only what was recorded (nothing, or
// the workspace alone) and writes stopped and detached.
func TestDirector_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration(t *testing.T) {
	t.Run("nothing recorded", func(t *testing.T) {
		// The refused start takes the second token before it reads the
		// file; the last start takes the third.
		f := newDirStartFixture(t, directorTestToken, "ba0000000000", "cb0000000000")
		if _, err := f.store.AttachHeadless(testHeadlessAttach()); err != nil {
			t.Fatalf("AttachHeadless: %v", err)
		}
		_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
		var ae *DirectorAttachedError
		if !errors.As(err, &ae) || ae.Current == nil || ae.Current.State != DirectorStateStarting {
			t.Fatalf("second start: %v, want a refusal naming the starting d1", err)
		}
		if _, _, err := f.store.Attach(DirectorAttach{Driver: DirectorDriverClaude}); !errors.Is(err, ErrDirectorAttached) {
			t.Fatalf("attach: %v, want ErrDirectorAttached", err)
		}
		res, err := f.stop("d1")
		if err != nil {
			t.Fatalf("stop: %v", err)
		}
		if !res.WasCurrent || res.UnrecordedWorkspaceLabel != "ralph_director_"+directorTestToken ||
			res.CtrlCNote != "ctrl_c=skipped" || res.PaneNote != "pane=none recorded" || res.WorkspaceNote != "workspace=none recorded" {
			t.Fatalf("stop result = %+v", res)
		}
		if len(f.h.calls) != 0 {
			t.Fatalf("herdr calls = %v, want none", f.h.calls)
		}
		wantTrail(t, f.store, "attached d1", "detached d1", "stopped d1")
		if next := f.mustStart(t, DirectorDriverClaude, "opus"); next.Registration.ID != "d2" {
			t.Fatalf("next start registered %s, want d2", next.Registration.ID)
		}
	})
	t.Run("only the workspace recorded", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		if _, err := f.store.AttachHeadless(testHeadlessAttach()); err != nil {
			t.Fatalf("AttachHeadless: %v", err)
		}
		if err := f.store.RecordWorkspace("d1", "ws-7", testDirectorWho); err != nil {
			t.Fatalf("RecordWorkspace: %v", err)
		}
		setLabel(&f.h.workspaceLabels, "ws-7", DirectorWorkspaceLabel(directorTestToken))
		res, err := f.stop("d1")
		if err != nil || !res.WasCurrent || res.UnrecordedWorkspaceLabel != "" || res.WorkspaceNote != "workspace=closed" {
			t.Fatalf("stop = %+v, %v; want the recorded workspace closed", res, err)
		}
		if !slices.Equal(f.h.calls, []string{"workspace_get", "workspace_close"}) || !slices.Equal(f.h.workspaceCloseCalls, []string{"ws-7"}) {
			t.Fatalf("herdr calls = %v (closes %v), want the check and the close of ws-7", f.h.calls, f.h.workspaceCloseCalls)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "detached d1", "stopped d1")
	})
}

// TestStartDirector_BoundsTheAgentStart pins how the agent start is bounded:
// TimeoutMS 0 means defaultSpawnTimeoutMS for herdr's --timeout and the
// context, and an agent_pane_busy reply is retried and counted.
func TestStartDirector_BoundsTheAgentStart(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	f.h.agentStartErrs = []error{errors.New("herdr: " + agentPaneBusyMarker), nil}
	p := f.params(DirectorDriverClaude, "opus")
	p.TimeoutMS = 0
	res, err := f.o.StartDirector(p)
	if err != nil {
		t.Fatalf("StartDirector: %v", err)
	}
	if res.AgentStartRetries != 1 || !slices.Equal(f.h.agentStartTimeouts, []int{defaultSpawnTimeoutMS, defaultSpawnTimeoutMS}) {
		t.Fatalf("retries %d, timeouts %v, want 1 retry at the default timeout", res.AgentStartRetries, f.h.agentStartTimeouts)
	}
	if left := f.h.agentStartLeft[0]; left <= 5*time.Second || left > time.Duration(defaultSpawnTimeoutMS)*time.Millisecond {
		t.Fatalf("agent start context had %v left, want the default timeout's deadline", left)
	}
}
