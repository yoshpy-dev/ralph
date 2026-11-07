package org

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// fakeResumeHerdr is the herdr of the Resume tests: a HerdrClient whose
// methods Resume never calls come from the embedded fakeHerdr, plus the
// AgentGet of watchHerdrProbe. The agent status of each herdr agent name and
// the screen of each pane are scripted as queues (one entry per call; the
// last one repeats), and every call Resume makes is logged in order:
// "get <name>", "read <pane> <lines>", "text <pane>", "keys <pane> <keys>",
// and "wait <name> <until>". onSendText and onSendKeys, when set, run after
// a call that succeeded, without mu held, so a test can move the pane on as
// a real seat would (the typed text in the input box, the seat at work).
type fakeResumeHerdr struct {
	*fakeHerdr

	mu    sync.Mutex
	log   []string
	texts []paneSendTextCall
	reads int

	status   map[string][]string // herdr agent name -> agent_status per AgentGet
	getErr   map[string]error    // herdr agent name -> AgentGet's error
	getReply map[string]string   // herdr agent name -> AgentGet's raw reply
	screens  map[string][]string // pane id -> screen per PaneRead
	readErr  map[string]error    // pane id -> PaneRead's error

	sendTextErr map[string]error // pane id -> PaneSendText's error
	sendKeysErr map[string]error // pane id -> PaneSendKeys' error
	onSendText  func(paneID string)
	onSendKeys  func(paneID string)
}

func newFakeResumeHerdr() *fakeResumeHerdr {
	return &fakeResumeHerdr{
		fakeHerdr: &fakeHerdr{},
		status:    map[string][]string{}, getErr: map[string]error{}, getReply: map[string]string{},
		screens: map[string][]string{}, readErr: map[string]error{},
		sendTextErr: map[string]error{}, sendKeysErr: map[string]error{},
	}
}

// resumeScriptNext is the next entry of key's queue in m: the first one,
// dropped from the queue unless it is the last. Called with mu held.
func resumeScriptNext(m map[string][]string, key string) (string, bool) {
	q := m[key]
	if len(q) == 0 {
		return "", false
	}
	if len(q) > 1 {
		m[key] = q[1:]
	}
	return q[0], true
}

// resumeAgentGetReply is herdr 0.7.5's reply to `herdr agent get <name>`
// for an agent with status, in the shape
// docs/evidence/org-recheck-2026-10-24/watch-status-hooks-matcher.json
// recorded.
func resumeAgentGetReply(name, status string) string {
	return fmt.Sprintf(`{"id":"cli:agent:get","result":{"agent":{"agent":"claude","agent_status":%q,"cwd":"/w","interactive_ready":true,"name":%q,"pane_id":"pane-x","revision":698,"state_change_seq":48},"type":"agent_info"}}`, status, name)
}

func (f *fakeResumeHerdr) AgentGet(_ context.Context, target string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "get "+target)
	if err := f.getErr[target]; err != nil {
		return "", err
	}
	if raw, ok := f.getReply[target]; ok {
		return raw, nil
	}
	status, ok := resumeScriptNext(f.status, target)
	if !ok {
		return "", fmt.Errorf("fakeResumeHerdr: no agent %s", target)
	}
	return resumeAgentGetReply(target, status), nil
}

func (f *fakeResumeHerdr) PaneRead(_ context.Context, paneID string, lines int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, fmt.Sprintf("read %s %d", paneID, lines))
	f.reads++
	if err := f.readErr[paneID]; err != nil {
		return "", err
	}
	screen, _ := resumeScriptNext(f.screens, paneID)
	return screen, nil
}

func (f *fakeResumeHerdr) PaneSendText(_ context.Context, paneID, text string) error {
	f.mu.Lock()
	f.log = append(f.log, "text "+paneID)
	f.texts = append(f.texts, paneSendTextCall{paneID: paneID, text: text})
	err, hook := f.sendTextErr[paneID], f.onSendText
	f.mu.Unlock()
	if err == nil && hook != nil {
		hook(paneID)
	}
	return err
}

func (f *fakeResumeHerdr) PaneSendKeys(_ context.Context, paneID string, keys ...string) error {
	f.mu.Lock()
	f.log = append(f.log, "keys "+paneID+" "+strings.Join(keys, " "))
	err, hook := f.sendKeysErr[paneID], f.onSendKeys
	f.mu.Unlock()
	if err == nil && hook != nil {
		hook(paneID)
	}
	return err
}

func (f *fakeResumeHerdr) AgentWait(_ context.Context, target string, until []string, _ int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, "wait "+target+" "+strings.Join(until, ","))
	return "", nil
}

func (f *fakeResumeHerdr) setStatus(name string, statuses ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status[name] = statuses
}

func (f *fakeResumeHerdr) setScreens(paneID string, screens ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.screens[paneID] = screens
}

func (f *fakeResumeHerdr) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.log)
}

func (f *fakeResumeHerdr) typed() []paneSendTextCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.texts)
}

func (f *fakeResumeHerdr) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

// resumeSends is the "text" and "keys" entries of calls, in order: what
// reached the panes.
func resumeSends(calls []string) []string {
	var out []string
	for _, c := range calls {
		if strings.HasPrefix(c, "text ") || strings.HasPrefix(c, "keys ") {
			out = append(out, c)
		}
	}
	return out
}

// resumeTestOrg is testOrg with a fakeResumeHerdr as its herdr.
func resumeTestOrg(t *testing.T) (*Org, *fakeResumeHerdr) {
	t.Helper()
	o, _, _ := testOrg(t)
	h := newFakeResumeHerdr()
	o.Herdr = h
	return o, h
}

// seedResumeSeat records seatID of orgID spawned with role and driver in
// pane and returns its herdr agent name.
func seedResumeSeat(t *testing.T, o *Org, orgID, seatID, role, driver, pane string) string {
	t.Helper()
	name := herdrAgentName(orgID, seatID)
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: orgID, SeatID: seatID, Event: EventSpawned, Role: role, Driver: driver,
		PaneID: pane, HerdrAgentName: name,
	}); err != nil {
		t.Fatalf("seed seat %s/%s: %v", orgID, seatID, err)
	}
	return name
}

// seedStoppedClaudeSeat is seedResumeSeat of a claude seat whose agent is
// idle and whose pane shows the ENOTFOUND stop (evidence section 2) for the
// look and the look under the lock, and then the resume sentence in the
// input box, as a real pane does once it is typed.
func seedStoppedClaudeSeat(t *testing.T, o *Org, h *fakeResumeHerdr, orgID, seatID, role, pane string) string {
	t.Helper()
	name := seedResumeSeat(t, o, orgID, seatID, role, "claude", pane)
	stopped := readResumeScreen(t, "claude-enotfound.txt")
	h.setStatus(name, "idle")
	h.setScreens(pane, stopped, stopped, readResumeScreen(t, "claude-input-typed-wrapped.txt"))
	return name
}

// resumeTestDirector is the director.jsonl events of id, a headless director
// with launch and driver claude, launched with token; in herdr, with pane
// when it is set. detached detaches it, so another registration can be
// attached after it. writeResumeDirectors writes them.
func resumeTestDirector(id, token, launch, pane string, detached bool) []DirectorEvent {
	ev := func(kind string, set func(*DirectorEvent)) DirectorEvent {
		e := DirectorEvent{TS: "2026-11-07T00:00:01Z", ID: id, Event: kind, User: "u", Cwd: "/w"}
		if set != nil {
			set(&e)
		}
		return e
	}
	evs := []DirectorEvent{ev(DirectorEventAttached, func(e *DirectorEvent) {
		e.TS, e.Driver, e.Wake, e.Launch, e.Token = "2026-11-07T00:00:00Z", DirectorDriverClaude, []string{DirectorWakePull}, launch, token
		e.Model, e.PermissionMode = "m", PermissionModeGuarded
	})}
	if launch == DirectorLaunchHerdr {
		evs = append(evs, ev(DirectorEventWorkspaceCreated, func(e *DirectorEvent) { e.WorkspaceID = "ws-" + id }))
		if pane != "" {
			evs = append(evs, ev(DirectorEventPaneCreated, func(e *DirectorEvent) { e.PaneID = pane }))
		}
		evs = append(evs, ev(DirectorEventLaunched, nil))
	} else {
		evs = append(evs, ev(DirectorEventLaunched, func(e *DirectorEvent) { e.PID, e.StartTime = 4242, "Sat Nov  7 00:00:00 2026" }))
	}
	if detached {
		evs = append(evs, ev(DirectorEventDetached, nil))
	}
	return evs
}

// writeResumeDirectors writes evs as the director.jsonl next to o's
// manifest.
func writeResumeDirectors(t *testing.T, o *Org, evs ...DirectorEvent) {
	t.Helper()
	writeDirectorEvents(t, NewDirectorStore(filepath.Dir(o.Manifest.Path())), evs...)
}

// claudeENOTFOUNDLine is the line ClassifyResumeScreen matches on the
// ENOTFOUND stop of claude-enotfound.txt: its two wrapped lines joined.
const claudeENOTFOUNDLine = "⏺ API Error: Can't reach the API server — check your internet or DNS (ENOTFOUND)"

func TestResumeText_IsTheSentenceOfTheSpec(t *testing.T) {
	if ResumeText != resumeTestSentence {
		t.Fatalf("ResumeText = %q, want the spec's FR-5 sentence %q", ResumeText, resumeTestSentence)
	}
}

// TestOrgResume_StoppedClaudeSeat_TypesTheSentenceAndEnterOnce covers AC-1:
// a claude seat stopped on ENOTFOUND (evidence section 2) that herdr reports
// idle gets the resume sentence, the check that it is in the input box, and
// Enter exactly once, all after the look again under the pane lock, and one
// `resumed` event.
func TestOrgResume_StoppedClaudeSeat_TypesTheSentenceAndEnterOnce(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedStoppedClaudeSeat(t, o, h, "org-a", "implementer", "implementer", "pane-1")
	before := len(mustReadEvents(t, o))

	res := o.Resume(ResumeParams{})
	if len(res.Errs) != 0 || res.DirectorReadErr != nil {
		t.Fatalf("Errs %v, DirectorReadErr %v; want none", res.Errs, res.DirectorReadErr)
	}
	want := ResumeTargetResult{
		Kind: ResumeTargetSeat, OrgID: "org-a", ID: "implementer", Role: "implementer", Driver: "claude", PaneID: "pane-1",
		Outcome: ResumeOutcomeResumed, Line: claudeENOTFOUNDLine, SubmitConfirmed: true, Recorded: true,
	}
	if !slices.Equal(res.Targets, []ResumeTargetResult{want}) {
		t.Fatalf("Targets = %+v, want %+v", res.Targets, want)
	}
	wantCalls := []string{
		"get " + name, "read pane-1 60", // the look
		"get " + name, "read pane-1 60", // the look again, holding the pane lock
		"text pane-1", "read pane-1 60", // the sentence, and the check of the input box
		"keys pane-1 Enter", "wait " + name + " working,blocked",
	}
	if got := h.calls(); !slices.Equal(got, wantCalls) {
		t.Fatalf("herdr calls = %q, want %q", got, wantCalls)
	}
	if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: ResumeText}}) {
		t.Fatalf("typed = %+v, want ResumeText once into pane-1", got)
	}
	events := mustReadEvents(t, o)[before:]
	wantEvent := ManifestEvent{OrgID: "org-a", SeatID: "implementer", Event: EventResumed, PaneID: "pane-1", Details: "kind=stopped line=" + claudeENOTFOUNDLine}
	if len(events) != 1 {
		t.Fatalf("new events = %+v, want one resumed event", events)
	}
	got := events[0]
	got.TS = ""
	if got != wantEvent {
		t.Fatalf("resumed event = %+v, want %+v", got, wantEvent)
	}
	if seat, ok := seatFromEvents(mustReadEvents(t, o), "org-a", "implementer"); !ok || !seat.Active || seat.Event != EventSpawned {
		t.Fatalf("seat after resume = %+v (found %v); want it still active with spawned as its state event", seat, ok)
	}
}

// A codex seat stopped on a usage limit (codex-usage-limit.txt) is resumed
// the same way, with the codex composer as the input box.
func TestOrgResume_StoppedCodexSeat_TypesTheSentence(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedResumeSeat(t, o, "org-a", "reviewer", "reviewer", "codex", "pane-1")
	stopped := readResumeScreen(t, "codex-usage-limit.txt")
	h.setStatus(name, "done")
	h.setScreens("pane-1", stopped, stopped, readResumeScreen(t, "codex-input-typed.txt"))

	res := o.Resume(ResumeParams{})
	if len(res.Errs) != 0 || len(res.Targets) != 1 || res.Targets[0].Outcome != ResumeOutcomeResumed || !res.Targets[0].Recorded {
		t.Fatalf("Resume = %+v; want the codex seat resumed and recorded", res)
	}
	if want := ClassifyResumeScreen("codex", stopped).Line; res.Targets[0].Line != want || !strings.Contains(want, "You've hit your usage limit") {
		t.Fatalf("Line = %q, want %q", res.Targets[0].Line, want)
	}
	if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter"}) {
		t.Fatalf("sends = %q, want the sentence and one Enter", got)
	}
}

// TestOrgResume_RetryingOrWorking_Waits covers AC-2: a seat herdr reports
// working is waiting without a look at its screen, and an idle seat whose
// screen shows claude retrying (evidence section 3) is waiting too; neither
// gets anything typed or recorded.
func TestOrgResume_RetryingOrWorking_Waits(t *testing.T) {
	retrying := "✻ Can't reach the… · Retrying in 4s · attempt 13/300"
	for _, tc := range []struct {
		name, status string
		wantCalls    func(name string) []string
		wantReason   string
		wantLine     string
	}{
		{"working", "working", func(n string) []string { return []string{"get " + n} }, resumeReasonHerdrWorking, ""},
		{"idle on the retry screen", "idle", func(n string) []string { return []string{"get " + n, "read pane-1 60"} }, resumeReasonWaiting, retrying},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h := resumeTestOrg(t)
			name := seedResumeSeat(t, o, "org-a", "implementer", "implementer", "claude", "pane-1")
			h.setStatus(name, tc.status)
			h.setScreens("pane-1", readResumeScreen(t, "claude-retrying.txt"))
			before := len(mustReadEvents(t, o))

			res := o.Resume(ResumeParams{})
			if len(res.Errs) != 0 || len(res.Targets) != 1 {
				t.Fatalf("Resume = %+v; want one target and no error", res)
			}
			if got := res.Targets[0]; got.Outcome != ResumeOutcomeWaiting || got.Reason != tc.wantReason || got.Line != tc.wantLine || got.Recorded {
				t.Fatalf("target = %+v; want waiting, reason %q, line %q", got, tc.wantReason, tc.wantLine)
			}
			if got := h.calls(); !slices.Equal(got, tc.wantCalls(name)) {
				t.Fatalf("herdr calls = %q, want %q", got, tc.wantCalls(name))
			}
			if got := len(mustReadEvents(t, o)); got != before {
				t.Fatalf("events %d -> %d; want none recorded", before, got)
			}
		})
	}
}

// TestOrgResume_CodexHooksReviewDialog_Skipped covers AC-3: codex's hook
// review (evidence section 4), which herdr reports idle, is not a known
// stop, so nothing is typed and the seat is skipped.
func TestOrgResume_CodexHooksReviewDialog_Skipped(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedResumeSeat(t, o, "org-a", "reviewer", "reviewer", "codex", "pane-1")
	h.setStatus(name, "idle")
	h.setScreens("pane-1", readResumeScreen(t, "codex-hooks-review.txt"))

	res := o.Resume(ResumeParams{})
	if len(res.Errs) != 0 || len(res.Targets) != 1 {
		t.Fatalf("Resume = %+v; want one target and no error", res)
	}
	if got := res.Targets[0]; got.Outcome != ResumeOutcomeSkipped || got.Reason != "unknown screen: "+resumeReasonChoiceDialog || got.Line != "" {
		t.Fatalf("target = %+v; want skipped as an unknown screen (choice dialog)", got)
	}
	if got := h.calls(); !slices.Equal(got, []string{"get " + name, "read pane-1 60"}) {
		t.Fatalf("herdr calls = %q; want only the look", got)
	}
}

// TestOrgResume_PressEnterToContinue_SendsEnterOnly covers AC-4: claude's
// `Press enter to continue` after a usage limit reset gets Enter once and no
// text, with no check of the input box, and an enter_only `resumed` event.
func TestOrgResume_PressEnterToContinue_SendsEnterOnly(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedResumeSeat(t, o, "org-a", "implementer", "implementer", "claude", "pane-1")
	h.setStatus(name, "idle")
	h.setScreens("pane-1", readResumeScreen(t, "claude-press-enter.txt"))
	before := len(mustReadEvents(t, o))

	res := o.Resume(ResumeParams{})
	const line = "Your usage limit has reset Press enter to continue"
	if len(res.Errs) != 0 || len(res.Targets) != 1 {
		t.Fatalf("Resume = %+v; want one target and no error", res)
	}
	if got := res.Targets[0]; got.Outcome != ResumeOutcomeEntered || got.Line != line || !got.SubmitConfirmed || !got.Recorded {
		t.Fatalf("target = %+v; want entered with line %q, confirmed and recorded", got, line)
	}
	wantCalls := []string{"get " + name, "read pane-1 60", "get " + name, "read pane-1 60", "keys pane-1 Enter", "wait " + name + " working,blocked"}
	if got := h.calls(); !slices.Equal(got, wantCalls) {
		t.Fatalf("herdr calls = %q, want %q", got, wantCalls)
	}
	events := mustReadEvents(t, o)[before:]
	if len(events) != 1 || events[0].Event != EventResumed || events[0].Details != "kind=enter_only line="+line {
		t.Fatalf("new events = %+v; want one resumed event with kind=enter_only", events)
	}
}

// TestOrgResume_CallersOwnPane_Skipped covers AC-5: the seat in the pane of
// the process that runs resume (HERDR_PANE_ID) is skipped without a herdr
// call, and the other seats are resumed.
func TestOrgResume_CallersOwnPane_Skipped(t *testing.T) {
	o, h := resumeTestOrg(t)
	own := seedStoppedClaudeSeat(t, o, h, "org-a", "leader", LeaderIdentity, "pane-own")
	seedStoppedClaudeSeat(t, o, h, "org-a", "implementer", "implementer", "pane-2")
	o.Getenv = func(key string) string {
		if key == herdrPaneIDEnv {
			return "pane-own"
		}
		return ""
	}

	res := o.Resume(ResumeParams{})
	if len(res.Errs) != 0 || len(res.Targets) != 2 {
		t.Fatalf("Resume = %+v; want two targets and no error", res)
	}
	if got := res.Targets[0]; got.ID != "implementer" || got.Outcome != ResumeOutcomeResumed {
		t.Fatalf("Targets[0] = %+v; want the implementer resumed", got)
	}
	if got := res.Targets[1]; got.ID != "leader" || got.Outcome != ResumeOutcomeSkipped || got.Reason != resumeReasonOwnPane {
		t.Fatalf("Targets[1] = %+v; want the leader skipped as the caller's own pane", got)
	}
	for _, c := range h.calls() {
		if strings.Contains(c, own) || strings.Contains(c, "pane-own") {
			t.Fatalf("herdr call %q for the caller's own pane; want none", c)
		}
	}
}

// TestOrgResume_DryRun_TypesAndRecordsNothing covers AC-6: a dry run looks
// at every target as a real run does and reports would_resume or
// would_enter, but takes no lock, calls neither send-text nor send-keys,
// waits for nothing, and appends no event.
func TestOrgResume_DryRun_TypesAndRecordsNothing(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedStoppedClaudeSeat(t, o, h, "org-a", "implementer", "implementer", "pane-1")
	enter := seedResumeSeat(t, o, "org-a", "reviewer", "reviewer", "claude", "pane-2")
	h.setStatus(enter, "idle")
	h.setScreens("pane-2", readResumeScreen(t, "claude-press-enter.txt"))
	retry := seedResumeSeat(t, o, "org-a", "worker", "worker", "claude", "pane-3")
	h.setStatus(retry, "idle")
	h.setScreens("pane-3", readResumeScreen(t, "claude-retrying.txt"))
	writeResumeDirectors(t, o, resumeTestDirector("d1", directorTestToken, DirectorLaunchHerdr, "pane-d1", false)...)
	h.setStatus(DirectorAgentName(directorTestToken), "idle")
	h.setScreens("pane-d1", readResumeScreen(t, "claude-enotfound.txt"))
	before := mustReadEvents(t, o)

	res := o.Resume(ResumeParams{DryRun: true})
	if len(res.Errs) != 0 || res.DirectorReadErr != nil {
		t.Fatalf("Errs %v, DirectorReadErr %v; want none", res.Errs, res.DirectorReadErr)
	}
	var got []string
	for _, r := range res.Targets {
		got = append(got, r.ID+"="+string(r.Outcome))
	}
	want := []string{"implementer=would_resume", "reviewer=would_enter", "worker=waiting", "d1=would_resume"}
	if !slices.Equal(got, want) {
		t.Fatalf("outcomes = %q, want %q", got, want)
	}
	if res.Targets[0].Line != claudeENOTFOUNDLine || res.Targets[0].Recorded || res.Targets[3].Recorded {
		t.Fatalf("Targets = %+v; want the matched line and nothing recorded", res.Targets)
	}
	for _, c := range h.calls() {
		if !strings.HasPrefix(c, "get ") && !strings.HasPrefix(c, "read ") {
			t.Fatalf("herdr call %q in a dry run; want only gets and reads", c)
		}
	}
	if after := mustReadEvents(t, o); !slices.Equal(after, before) {
		t.Fatalf("a dry run appended %+v", after[len(before):])
	}
	locks, err := filepath.Glob(filepath.Join(filepath.Dir(o.Manifest.Path()), "pane-*.lock"))
	if err != nil || len(locks) != 0 {
		t.Fatalf("pane lock files after a dry run = %v (%v); want none", locks, err)
	}
}

// TestOrgResume_OrderIsSeatsThenLeadersThenDirectors covers AC-7 and FR-4:
// every seat that is not a leader is typed into before the leader, and the
// director last. The reviewer's seat id sorts after "leader", so the order
// comes from the groups, not from Roster's sort. The director is resumed
// but, as it is no seat of the manifest, gets no `resumed` event.
func TestOrgResume_OrderIsSeatsThenLeadersThenDirectors(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedStoppedClaudeSeat(t, o, h, "org-a", LeaderIdentity, LeaderIdentity, "pane-leader")
	seedStoppedClaudeSeat(t, o, h, "org-a", "implementer", "implementer", "pane-impl")
	seedStoppedClaudeSeat(t, o, h, "org-a", "reviewer", "reviewer", "pane-rev")
	writeResumeDirectors(t, o, resumeTestDirector("d1", directorTestToken, DirectorLaunchHerdr, "pane-d1", false)...)
	stopped := readResumeScreen(t, "claude-enotfound.txt")
	h.setStatus(DirectorAgentName(directorTestToken), "idle")
	h.setScreens("pane-d1", stopped, stopped, readResumeScreen(t, "claude-input-typed-wrapped.txt"))
	before := len(mustReadEvents(t, o))

	res := o.Resume(ResumeParams{})
	if len(res.Errs) != 0 {
		t.Fatalf("Errs = %v; want none", res.Errs)
	}
	var ids []string
	for _, r := range res.Targets {
		if r.Outcome != ResumeOutcomeResumed {
			t.Fatalf("target %+v; want every target resumed", r)
		}
		ids = append(ids, r.ID)
	}
	if want := []string{"implementer", "reviewer", LeaderIdentity, "d1"}; !slices.Equal(ids, want) {
		t.Fatalf("target order = %q, want %q", ids, want)
	}
	var panes []string
	for _, c := range h.typed() {
		panes = append(panes, c.paneID)
	}
	if want := []string{"pane-impl", "pane-rev", "pane-leader", "pane-d1"}; !slices.Equal(panes, want) {
		t.Fatalf("typed into %q, want %q", panes, want)
	}
	d := res.Targets[3]
	if d.Kind != ResumeTargetDirector || d.OrgID != "" || d.Role != DirectorRole || d.Driver != DirectorDriverClaude || d.PaneID != "pane-d1" || d.Recorded {
		t.Fatalf("director target = %+v; want kind director, role director, pane-d1, not recorded", d)
	}
	var recorded []string
	for _, ev := range mustReadEvents(t, o)[before:] {
		recorded = append(recorded, ev.Event+" "+ev.SeatID)
	}
	if want := []string{"resumed implementer", "resumed reviewer", "resumed leader"}; !slices.Equal(recorded, want) {
		t.Fatalf("new events = %q, want %q", recorded, want)
	}
}

// With OrgID set, Resume looks only at that org's seats and at no director;
// without it, at every org's seats and the directors.
func TestOrgResume_OrgIDFilter_TwoOrgsAndTheDirector(t *testing.T) {
	setup := func(t *testing.T) (*Org, *fakeResumeHerdr) {
		o, h := resumeTestOrg(t)
		seedStoppedClaudeSeat(t, o, h, "org-a", "seat-1", "worker", "pane-a")
		seedStoppedClaudeSeat(t, o, h, "org-b", "seat-1", "worker", "pane-b")
		writeResumeDirectors(t, o, resumeTestDirector("d1", directorTestToken, DirectorLaunchHerdr, "pane-d1", false)...)
		stopped := readResumeScreen(t, "claude-enotfound.txt")
		h.setStatus(DirectorAgentName(directorTestToken), "idle")
		h.setScreens("pane-d1", stopped, stopped, readResumeScreen(t, "claude-input-typed-wrapped.txt"))
		return o, h
	}
	for _, tc := range []struct {
		orgID string
		want  []string
	}{
		{"org-b", []string{"org-b/seat-1"}},
		{"", []string{"org-a/seat-1", "org-b/seat-1", "/d1"}},
	} {
		t.Run("org_id="+tc.orgID, func(t *testing.T) {
			o, h := setup(t)
			res := o.Resume(ResumeParams{OrgID: tc.orgID})
			if len(res.Errs) != 0 {
				t.Fatalf("Errs = %v; want none", res.Errs)
			}
			var got []string
			for _, r := range res.Targets {
				if r.Outcome != ResumeOutcomeResumed {
					t.Fatalf("target %+v; want resumed", r)
				}
				got = append(got, r.OrgID+"/"+r.ID)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("targets = %q, want %q", got, tc.want)
			}
			directorLooked := slices.Contains(h.calls(), "get "+DirectorAgentName(directorTestToken))
			if directorLooked != (tc.orgID == "") {
				t.Fatalf("director looked at = %v with org_id %q; want it only without org_id", directorLooked, tc.orgID)
			}
		})
	}
}

// A director started with launch_cmd runs outside herdr, and a director in
// herdr whose pane is not recorded yet has no pane: both are listed as
// skipped with no herdr call.
func TestOrgResume_DirectorsNotInHerdrOrWithoutAPane_Skipped(t *testing.T) {
	o, h := resumeTestOrg(t)
	evs := resumeTestDirector("d1", "111111111111", DirectorLaunchHerdr, "", true)
	evs = append(evs, resumeTestDirector("d2", "222222222222", DirectorLaunchCmd, "", false)...)
	writeResumeDirectors(t, o, evs...)

	res := o.Resume(ResumeParams{})
	if len(res.Errs) != 0 || res.DirectorReadErr != nil {
		t.Fatalf("Errs %v, DirectorReadErr %v; want none", res.Errs, res.DirectorReadErr)
	}
	want := []ResumeTargetResult{
		{Kind: ResumeTargetDirector, ID: "d1", Role: DirectorRole, Driver: DirectorDriverClaude, Outcome: ResumeOutcomeSkipped, Reason: resumeReasonNoPane},
		{Kind: ResumeTargetDirector, ID: "d2", Role: DirectorRole, Driver: DirectorDriverClaude, Outcome: ResumeOutcomeSkipped, Reason: resumeReasonNotInHerdr},
	}
	if !slices.Equal(res.Targets, want) {
		t.Fatalf("Targets = %+v, want %+v", res.Targets, want)
	}
	if got := h.calls(); len(got) != 0 {
		t.Fatalf("herdr calls = %q; want none", got)
	}
}

// What herdr answers before the screen is classified, and a herdr client
// with no agent get, skip the seat with the reason and type nothing; none of
// them is an error of the resume.
func TestOrgResume_AgentGetAndPaneReadFailures_Skipped(t *testing.T) {
	name := herdrAgentName("org-a", "seat-1")
	for _, tc := range []struct {
		name   string
		setup  func(o *Org, h *fakeResumeHerdr)
		reason string
	}{
		{"agent get fails", func(_ *Org, h *fakeResumeHerdr) { h.getErr[name] = errors.New("boom") }, "herdr agent get " + name + " failed: boom"},
		{"reply not JSON", func(_ *Org, h *fakeResumeHerdr) { h.getReply[name] = "not json" }, "herdr agent get " + name + ": the reply is not herdr's JSON"},
		{"reply without status", func(_ *Org, h *fakeResumeHerdr) {
			h.getReply[name] = `{"id":"cli:agent:get","result":{"agent":{"name":"x"},"type":"agent_info"}}`
		}, "herdr agent get " + name + ": the reply has no result.agent.agent_status"},
		{"blocked", func(_ *Org, h *fakeResumeHerdr) { h.setStatus(name, "blocked") }, resumeReasonHerdrBlocked},
		{"unknown status", func(_ *Org, h *fakeResumeHerdr) { h.setStatus(name, "unknown") }, `herdr reports "unknown"`},
		{"pane read fails", func(_ *Org, h *fakeResumeHerdr) { h.readErr["pane-1"] = errors.New("gone") }, "herdr pane read pane-1 failed: gone"},
		{"no agent get", func(o *Org, _ *fakeResumeHerdr) { o.Herdr = &fakeHerdr{} }, resumeReasonNoAgentGet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h := resumeTestOrg(t)
			seedStoppedClaudeSeat(t, o, h, "org-a", "seat-1", "worker", "pane-1")
			tc.setup(o, h)
			res := o.Resume(ResumeParams{})
			if len(res.Errs) != 0 || len(res.Targets) != 1 {
				t.Fatalf("Resume = %+v; want one target and no error", res)
			}
			if got := res.Targets[0]; got.Outcome != ResumeOutcomeSkipped || !strings.HasPrefix(got.Reason, tc.reason) {
				t.Fatalf("target = %+v; want skipped with a reason starting %q", got, tc.reason)
			}
			if got := resumeSends(h.calls()); len(got) != 0 {
				t.Fatalf("sends = %q; want none", got)
			}
		})
	}
}

// One target's failed send makes it failed and puts its error in Errs, and
// Resume goes on with the next target: a failed send-text presses no Enter,
// a failed send-keys is reported as unknown, and the third seat is resumed
// and recorded.
func TestOrgResume_OneTargetFails_TheNextIsStillResumed(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedStoppedClaudeSeat(t, o, h, "org-a", "seat-1", "worker", "pane-1")
	seedStoppedClaudeSeat(t, o, h, "org-a", "seat-2", "worker", "pane-2")
	seedStoppedClaudeSeat(t, o, h, "org-a", "seat-3", "worker", "pane-3")
	textErr, keysErr := errors.New("send-text: no answer"), errors.New("send-keys: no answer")
	h.sendTextErr["pane-1"], h.sendKeysErr["pane-2"] = textErr, keysErr
	before := len(mustReadEvents(t, o))

	res := o.Resume(ResumeParams{})
	var got []string
	for _, r := range res.Targets {
		got = append(got, r.ID+"="+string(r.Outcome))
	}
	if want := []string{"seat-1=failed", "seat-2=failed", "seat-3=resumed"}; !slices.Equal(got, want) {
		t.Fatalf("outcomes = %q, want %q", got, want)
	}
	if len(res.Errs) != 2 || !errors.Is(res.Errs[0], textErr) || !errors.Is(res.Errs[1], keysErr) ||
		!strings.HasPrefix(res.Errs[0].Error(), "org-a/seat-1: ") || !strings.HasPrefix(res.Errs[1].Error(), "org-a/seat-2: ") {
		t.Fatalf("Errs = %v; want seat-1's send-text and seat-2's send-keys errors, each prefixed with its seat", res.Errs)
	}
	if r := res.Targets[0].Reason; !strings.Contains(r, "the text may or may not have reached the pane; Enter not pressed") {
		t.Fatalf("seat-1 reason = %q; want it to say the text may or may not have reached the pane", r)
	}
	if r := res.Targets[1].Reason; !strings.Contains(r, "the text was typed and Enter may or may not have reached the pane") {
		t.Fatalf("seat-2 reason = %q; want it to say Enter may or may not have reached the pane", r)
	}
	if want := []string{"text pane-1", "text pane-2", "keys pane-2 Enter", "text pane-3", "keys pane-3 Enter"}; !slices.Equal(resumeSends(h.calls()), want) {
		t.Fatalf("sends = %q, want %q", resumeSends(h.calls()), want)
	}
	events := mustReadEvents(t, o)[before:]
	if len(events) != 1 || events[0].SeatID != "seat-3" || events[0].Event != EventResumed {
		t.Fatalf("new events = %+v; want only seat-3's resumed", events)
	}
}

// A manifest that cannot be read is the only error: no target is looked at.
func TestOrgResume_ManifestUnreadable_IsTheOnlyError(t *testing.T) {
	o, h := resumeTestOrg(t)
	o.Manifest = NewManifestStoreAtPath(t.TempDir())

	res := o.Resume(ResumeParams{})
	if len(res.Errs) != 1 || !strings.HasPrefix(res.Errs[0].Error(), "org: resume: read manifest: ") || len(res.Targets) != 0 {
		t.Fatalf("Resume = %+v; want the manifest read error alone", res)
	}
	if got := h.calls(); len(got) != 0 {
		t.Fatalf("herdr calls = %q; want none", got)
	}
}

// A director.jsonl that cannot be read is recorded in DirectorReadErr only:
// the seats are resumed and Errs stays empty.
func TestOrgResume_DirectorFileUnreadable_SeatsStillResumed(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedStoppedClaudeSeat(t, o, h, "org-a", "seat-1", "worker", "pane-1")
	if err := os.MkdirAll(DirectorPathIn(filepath.Dir(o.Manifest.Path())), 0o755); err != nil {
		t.Fatalf("make director.jsonl a directory: %v", err)
	}

	res := o.Resume(ResumeParams{})
	if res.DirectorReadErr == nil || len(res.Errs) != 0 || len(res.Targets) != 1 || res.Targets[0].Outcome != ResumeOutcomeResumed {
		t.Fatalf("Resume = %+v; want the seat resumed, DirectorReadErr set, and no error", res)
	}
}

// When Enter was pressed but the `resumed` event cannot be appended, the
// seat is still resumed (Enter is never pressed again), not recorded, and
// the append's error is in Errs.
func TestOrgResume_ResumedEventNotRecorded_ReportedAsAnError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	o, h := resumeTestOrg(t)
	seedStoppedClaudeSeat(t, o, h, "org-a", "seat-1", "worker", "pane-1")
	path := o.Manifest.Path()
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatalf("chmod manifest read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	res := o.Resume(ResumeParams{})
	if len(res.Targets) != 1 || len(res.Errs) != 1 {
		t.Fatalf("Resume = %+v; want one target and one error", res)
	}
	got := res.Targets[0]
	if got.Outcome != ResumeOutcomeResumed || got.Recorded || !strings.HasPrefix(got.Reason, "the resumed event could not be recorded after Enter was pressed: ") ||
		!errors.Is(res.Errs[0], os.ErrPermission) {
		t.Fatalf("target = %+v, Errs = %v; want resumed, not recorded, and the append's permission error", got, res.Errs)
	}
	if want := []string{"text pane-1", "keys pane-1 Enter"}; !slices.Equal(resumeSends(h.calls()), want) {
		t.Fatalf("sends = %q, want %q", resumeSends(h.calls()), want)
	}
}

// resumeSeatTargets takes active real seats with a pane, of the org when
// one is named, non-leaders first and then leaders (by role or seat id),
// each group in Roster's order, and takes the role and driver of a seat in
// the middle of its spawn from the state event that recorded them.
func TestOrgResume_SeatTargets_GroupsAndFields(t *testing.T) {
	ts := "2026-11-07T00:00:00Z"
	events := []ManifestEvent{
		{TS: ts, OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned, Role: LeaderIdentity, Driver: "claude", PaneID: "p-l"},
		{TS: ts, OrgID: "org-a", SeatID: "w1", Event: EventSpawnStarted, Role: "worker", Driver: "codex"},
		{TS: ts, OrgID: "org-a", SeatID: "w1", Event: EventSpawnStep, PaneID: "p-w1"},
		{TS: ts, OrgID: "org-a", SeatID: "a-lead", Event: EventSpawned, Role: LeaderIdentity, Driver: "claude", PaneID: "p-al"},
		{TS: ts, OrgID: "org-a", SeatID: "gone", Event: EventSpawned, Role: "worker", Driver: "claude", PaneID: "p-g"},
		{TS: ts, OrgID: "org-a", SeatID: "gone", Event: EventStopped},
		{TS: ts, OrgID: "org-a", SeatID: "nopane", Event: EventSpawned, Role: "worker", Driver: "claude"},
		{TS: ts, OrgID: "org-a", SeatID: "dry", Event: EventSpawned, Role: "worker", Driver: "claude", PaneID: "p-d", DryRun: true},
		{TS: ts, OrgID: "org-b", SeatID: "x", Event: EventSpawned, Role: "worker", Driver: "claude", PaneID: "p-x", HerdrAgentName: "custom-name"},
		{TS: ts, OrgID: "org-a", SeatID: "w1", Event: EventSent},
	}
	summary := func(ts []resumeTarget) []string {
		var out []string
		for _, t := range ts {
			out = append(out, fmt.Sprintf("%s/%s %s %s %s %s", t.orgID, t.id, t.role, t.driver, t.paneID, t.agentName))
		}
		return out
	}
	all := []string{
		"org-a/w1 worker codex p-w1 " + herdrAgentName("org-a", "w1"),
		"org-b/x worker claude p-x custom-name",
		"org-a/a-lead leader claude p-al " + herdrAgentName("org-a", "a-lead"),
		"org-a/leader leader claude p-l " + herdrAgentName("org-a", LeaderIdentity),
	}
	if got := summary(resumeSeatTargets(events, "")); !slices.Equal(got, all) {
		t.Fatalf("every org: %q, want %q", got, all)
	}
	if got, want := summary(resumeSeatTargets(events, "org-a")), []string{all[0], all[2], all[3]}; !slices.Equal(got, want) {
		t.Fatalf("org-a: %q, want %q", got, want)
	}
}

func TestOrgResume_HerdrAgentStatus_ReadsTheEnvelope(t *testing.T) {
	// The reply docs/evidence/org-recheck-2026-10-24/watch-status-hooks-matcher.json
	// recorded for a seat at rest.
	real := `{"id":"cli:agent:get","result":{"agent":{"agent":"claude","agent_session":{"agent":"claude","kind":"id","source":"herdr:claude","value":"session-4"},"agent_status":"done","cwd":"/w","focused":false,"interactive_ready":true,"name":"hooks-matcher_implementer","pane_id":"pane-3","revision":698,"state_change_seq":48,"tab_id":"tab-3","workspace_id":"ws-2"},"type":"agent_info"}}` + "\n"
	if got, err := herdrAgentStatus(real); err != nil || got != "done" {
		t.Fatalf("herdrAgentStatus(real reply) = %q, %v; want done", got, err)
	}
	for _, bad := range []string{"", "idle", `{"result":{}}`, `{"result":{"agent":{"agent_status":""}}}`, `[1]`} {
		if got, err := herdrAgentStatus(bad); err == nil {
			t.Errorf("herdrAgentStatus(%q) = %q, nil; want an error", bad, got)
		}
	}
}
