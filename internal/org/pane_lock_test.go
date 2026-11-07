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
	"time"
)

// The pane lock that Send and Resume share (pane_lock.go; plan
// 2026-11-07-org-seat-resume, D8 and AC-14). The test names hold Send or
// Resume so the slice's `-run 'Resume|Send'` runs them.

func TestSendResumePaneLockName_MapsUnsafeBytes(t *testing.T) {
	for _, tc := range []struct{ paneID, want string }{
		{"pane-3", "pane-pane-3.lock"},
		{"w1:p1", "pane-w1_p1.lock"},
		{"a/b", "pane-a_b.lock"},
		{"../x", "pane-.._x.lock"},
		{"A.b_c-9", "pane-A.b_c-9.lock"},
		{"ペ", "pane-___.lock"},
	} {
		got := paneLockName(tc.paneID)
		if got != tc.want || filepath.Base(got) != got {
			t.Errorf("paneLockName(%q) = %q, want %q (one name, no directory)", tc.paneID, got, tc.want)
		}
	}
}

// paneLockPath is the path of paneID's lock file next to o's manifest.
func paneLockPath(o *Org, paneID string) string {
	return filepath.Join(filepath.Dir(o.Manifest.Path()), paneLockName(paneID))
}

// holdPaneLock takes paneID's pane lock as another holder would (a Send or
// a resume in another process) and releases it when the test ends.
func holdPaneLock(t *testing.T, o *Org, paneID string) {
	t.Helper()
	lock, err := o.acquirePaneLock(paneID, 0)
	if err != nil {
		t.Fatalf("hold the pane lock of %s: %v", paneID, err)
	}
	t.Cleanup(func() { _ = lock.release() })
}

func mustPaneLockFree(t *testing.T, o *Org, paneID string) {
	t.Helper()
	held, err := fileLockHeld(paneLockPath(o, paneID))
	if err != nil || held {
		t.Fatalf("pane lock of %s held = %v (%v); want it free", paneID, held, err)
	}
}

// lockProbeHerdr is fakeHerdr that also records, at each AgentWait,
// PaneSendText, and PaneSendKeys, whether the pane lock at path was held.
type lockProbeHerdr struct {
	*fakeHerdr
	path string
	held []string
}

func (h *lockProbeHerdr) probe(call string) {
	held, err := fileLockHeld(h.path)
	if err != nil {
		h.held = append(h.held, call+"=error:"+err.Error())
		return
	}
	if held {
		h.held = append(h.held, call+"=held")
	} else {
		h.held = append(h.held, call+"=free")
	}
}

func (h *lockProbeHerdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	h.probe("wait " + strings.Join(until, ","))
	return h.fakeHerdr.AgentWait(ctx, target, until, timeoutMS)
}

func (h *lockProbeHerdr) PaneSendText(ctx context.Context, paneID, text string) error {
	h.probe("text")
	return h.fakeHerdr.PaneSendText(ctx, paneID, text)
}

func (h *lockProbeHerdr) PaneSendKeys(ctx context.Context, paneID string, keys ...string) error {
	h.probe("keys")
	return h.fakeHerdr.PaneSendKeys(ctx, paneID, keys...)
}

// TestOrgSend_HoldsThePaneLockWhileTypingAndReleasesIt: Send takes the pane
// lock after its idle/done wait and holds it through the idle/done wait
// again (cross-review cycle 1 #1), send-text, Enter, and the submit
// confirmation; it releases it before it returns, so a second Send to the
// same pane after the first returns takes it and succeeds. The pane id holds
// a ':', which the lock file's name replaces.
func TestOrgSend_HoldsThePaneLockWhileTypingAndReleasesIt(t *testing.T) {
	o, fh, _ := testOrg(t)
	seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "w1:p1")
	h := &lockProbeHerdr{fakeHerdr: fh, path: paneLockPath(o, "w1:p1")}
	o.Herdr = h

	msg := "TYPE: QUESTION\n\nwhich plan?"
	for i := range 2 {
		res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
		if res.Err != nil || res.Progress != SendProgressEnterPressed {
			t.Fatalf("send %d = %+v; want it to succeed", i+1, res)
		}
		mustPaneLockFree(t, o, "w1:p1")
	}
	want := []string{"wait idle,done=free", "wait idle,done=held", "text=held", "keys=held", "wait working,blocked=held"}
	if wantBoth := append(slices.Clone(want), want...); !slices.Equal(h.held, wantBoth) {
		t.Fatalf("lock at each call = %q, want %q", h.held, wantBoth)
	}
	if filepath.Base(h.path) != "pane-w1_p1.lock" {
		t.Fatalf("lock path = %s; want pane-w1_p1.lock", h.path)
	}
	if _, err := os.Stat(h.path); err != nil {
		t.Fatalf("lock file: %v; want it next to the manifest", err)
	}
}

// TestOrgSend_PaneLockHeld_RefusesWithNothingTyped (AC-14): while another
// send or resume holds the pane's lock, Send waits for it no longer than its
// --timeout-ms deadline and then refuses with nothing typed, nothing
// recorded, and Progress SendProgressNothingSent.
func TestOrgSend_PaneLockHeld_RefusesWithNothingTyped(t *testing.T) {
	o, h, _ := testOrg(t)
	seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
	holdPaneLock(t, o, "pane-1")
	before := len(mustReadEvents(t, o))

	start := time.Now()
	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "TYPE: QUESTION\n\nwhich plan?", TimeoutMS: 300})
	elapsed := time.Since(start)
	if res.Err == nil || !errors.Is(res.Err, errFileLockHeld) ||
		!strings.HasPrefix(res.Err.Error(), `org: send: another send or resume to the pane of seat "seat-1" (pane-1) is in progress`) ||
		!strings.Contains(res.Err.Error(), "nothing was typed") {
		t.Fatalf("Send = %v; want the pane-lock refusal saying nothing was typed", res.Err)
	}
	if res.Progress != SendProgressNothingSent || res.PaneID != "" || res.SubmitConfirmed {
		t.Fatalf("Send = %+v; want SendProgressNothingSent and no pane", res)
	}
	if !slices.Equal(h.calls, []string{"agent_wait"}) {
		t.Fatalf("herdr calls = %q; want only the idle/done wait", h.calls)
	}
	if got := len(mustReadEvents(t, o)); got != before {
		t.Fatalf("events %d -> %d; want none recorded", before, got)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("Send took %v; want it bounded by its --timeout-ms deadline", elapsed)
	}
}

// A Send the budget check refuses after it took the pane lock releases the
// lock all the same.
func TestOrgSend_BudgetRefusal_ReleasesThePaneLock(t *testing.T) {
	o, _, _ := testOrg(t)
	o.SendEnterDelay = 200 * time.Millisecond
	seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "TYPE: QUESTION\n\nwhich plan?", TimeoutMS: 5})
	if res.Err == nil || !strings.Contains(res.Err.Error(), "the pause before Enter needs") {
		t.Fatalf("Send = %v; want the budget refusal", res.Err)
	}
	mustPaneLockFree(t, o, "pane-1")
}

// underLockWaitHerdr is fakeHerdr whose first AgentWait, Send's idle/done
// wait before the pane lock, takes firstDelay, and whose second, the
// idle/done wait under the lock, does not return until ctx is done and then
// returns ctx's error as the driver wraps it: a seat that stays at work for
// the rest of the --timeout-ms deadline.
type underLockWaitHerdr struct {
	*fakeHerdr
	firstDelay time.Duration
	waits      int
}

func (h *underLockWaitHerdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	out, err := h.fakeHerdr.AgentWait(ctx, target, until, timeoutMS)
	h.waits++
	switch h.waits {
	case 1:
		time.Sleep(h.firstDelay)
	case 2:
		<-ctx.Done()
		return "", fmt.Errorf("herdr: timed out: %w", ctx.Err())
	}
	return out, err
}

// TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing (cross-review
// cycle 1 #1 of org-seat-resume): holding the pane lock, Send waits for
// idle/done again, with the same until states and no longer than what is
// left of its --timeout-ms deadline. When that wait fails, or the deadline
// runs out in it, Send types nothing, records nothing, returns
// SendProgressNothingSent, and releases the lock. The error (cycle 2
// self-review C2-2 of org-seat-resume) says the wait under the lock of the
// pane failed and nothing was typed when herdr's call fails, and when the
// deadline runs out in the wait, says another send or resume may have
// started a turn and ends with sendWaitDeadlineHint, as the first wait's
// error does, so the /director skill's rule for sending again applies.
func TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing(t *testing.T) {
	const head = `org: send: wait for seat "seat-1" idle/done again under the lock of its pane (pane-1)`
	for _, tc := range []struct {
		name      string
		timeoutMS int
		herdr     func(fh *fakeHerdr) HerdrClient
		want      string
		deadline  bool // errors.Is(err, context.DeadlineExceeded)
		// maxSecondMS is the most the wait under the lock may be given: what
		// is left of the deadline after the first wait.
		maxSecondMS int
	}{
		{"the wait fails", 10000, func(fh *fakeHerdr) HerdrClient {
			fh.agentWaitErrs = []error{nil, errors.New("stub: wait under the lock failed")}
			return fh
		}, head + " failed; nothing was typed: stub: wait under the lock failed", false, 10000},
		{"the deadline runs out in the wait", 300, func(fh *fakeHerdr) HerdrClient {
			return &underLockWaitHerdr{fakeHerdr: fh, firstDelay: 100 * time.Millisecond}
		}, head + "; another send or resume may have started a turn in it: herdr: timed out: context deadline exceeded" +
			sendWaitDeadlineHint(300), true, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, fh, _ := testOrg(t)
			o.Herdr = tc.herdr(fh)
			seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "TYPE: QUESTION\n\nwhich plan?", TimeoutMS: tc.timeoutMS})
			if res.Err == nil || res.Err.Error() != tc.want || errors.Is(res.Err, context.DeadlineExceeded) != tc.deadline {
				t.Fatalf("Send = %v; want %q (deadline exceeded %v)", res.Err, tc.want, tc.deadline)
			}
			if hint := strings.Contains(res.Err.Error(), directorSkillSendWaitTimeoutPhrase); hint != tc.deadline {
				t.Fatalf("Send = %v; holds the /director skill's phrase %q: %v, want %v", res.Err, directorSkillSendWaitTimeoutPhrase, hint, tc.deadline)
			}
			if res.Progress != SendProgressNothingSent || res.PaneID != "" || res.SubmitConfirmed {
				t.Fatalf("Send = %+v; want SendProgressNothingSent and no pane", res)
			}
			if !slices.Equal(fh.calls, []string{"agent_wait", "agent_wait"}) {
				t.Fatalf("herdr calls = %q; want the two idle/done waits and nothing typed", fh.calls)
			}
			if len(fh.agentWaitUntil) != 2 || !slices.Equal(fh.agentWaitUntil[1], []string{"idle", "done"}) {
				t.Fatalf("until states = %q; want idle/done again under the lock", fh.agentWaitUntil)
			}
			if got := fh.agentWaitTimeouts; len(got) != 2 || got[0] != tc.timeoutMS || got[1] < 1 || got[1] > tc.maxSecondMS {
				t.Fatalf("wait timeouts = %v; want %d, then at most the %d ms left", got, tc.timeoutMS, tc.maxSecondMS)
			}
			if got := len(mustReadEvents(t, o)); got != before {
				t.Fatalf("events %d -> %d; want none recorded", before, got)
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}

// A lock file that cannot be opened (a directory in its place) is not a
// lock another holder keeps: Send refuses with nothing typed, and Resume
// reports the seat failed, with nothing typed, as an error of the resume.
func TestOrgSendResume_PaneLockCannotBeOpened(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedStoppedClaudeSeat(t, o, h, "org-a", "seat-1", "worker", "pane-1")
	if err := os.Mkdir(paneLockPath(o, "pane-1"), 0o755); err != nil {
		t.Fatalf("make the lock file a directory: %v", err)
	}

	send := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "TYPE: QUESTION\n\nwhich plan?"})
	if send.Err == nil || errors.Is(send.Err, errFileLockHeld) || send.Progress != SendProgressNothingSent ||
		!strings.HasPrefix(send.Err.Error(), `org: send: take the lock of the pane of seat "seat-1" (pane-1); nothing was typed: `) {
		t.Fatalf("Send = %+v; want the lock-open refusal with nothing typed", send)
	}

	res := o.Resume(ResumeParams{})
	if len(res.Targets) != 1 || len(res.Errs) != 1 {
		t.Fatalf("Resume = %+v; want one target and one error", res)
	}
	if got := res.Targets[0]; got.Outcome != ResumeOutcomeFailed || !strings.HasPrefix(got.Reason, "take the lock of pane pane-1; nothing was typed: ") {
		t.Fatalf("target = %+v; want failed, nothing typed", got)
	}
	if got := resumeSends(h.calls()); len(got) != 0 {
		t.Fatalf("sends = %q; want none", got)
	}
}

// TestOrgResume_PaneLockHeld_SkipsAndTypesNothing covers AC-14 (i): while
// another send or resume holds the pane's lock, Resume waits LockWait and
// skips the seat with nothing typed; a Send to the same seat also refuses
// after its wait. Neither is recorded.
func TestOrgResume_PaneLockHeld_SkipsAndTypesNothing(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedStoppedClaudeSeat(t, o, h, "org-a", "seat-1", "worker", "pane-1")
	holdPaneLock(t, o, "pane-1")
	before := len(mustReadEvents(t, o))

	res := o.Resume(ResumeParams{LockWait: 50 * time.Millisecond})
	if len(res.Errs) != 0 || len(res.Targets) != 1 {
		t.Fatalf("Resume = %+v; want one target and no error", res)
	}
	if got := res.Targets[0]; got.Outcome != ResumeOutcomeSkipped || got.Reason != resumeReasonPaneBusy || got.Line != claudeENOTFOUNDLine {
		t.Fatalf("target = %+v; want skipped: %s", got, resumeReasonPaneBusy)
	}
	if got := h.calls(); !slices.Equal(got, []string{"get " + name, "read pane-1 60"}) {
		t.Fatalf("herdr calls = %q; want only the first look", got)
	}

	send := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "TYPE: QUESTION\n\nwhich plan?", TimeoutMS: 200})
	if send.Err == nil || !errors.Is(send.Err, errFileLockHeld) || send.Progress != SendProgressNothingSent {
		t.Fatalf("Send = %+v; want the pane-lock refusal", send)
	}
	if got := resumeSends(h.calls()); len(got) != 0 {
		t.Fatalf("sends = %q; want none", got)
	}
	if got := len(mustReadEvents(t, o)); got != before {
		t.Fatalf("events %d -> %d; want none recorded", before, got)
	}
}

// TestOrgResume_ScreenChangedUnderTheLock_TypesNothing covers AC-14 (ii):
// the look again under the pane lock must find the agent idle or done and a
// screen of the same kind; a dialog that came up after the first look (the
// folder trust dialog), another status, or another kind of screen makes
// Resume type neither the sentence nor Enter.
func TestOrgResume_ScreenChangedUnderTheLock_TypesNothing(t *testing.T) {
	for _, tc := range []struct {
		name     string
		statuses []string
		screens  []string
		reason   string
	}{
		{"an approval dialog", []string{"idle"}, []string{"claude-enotfound.txt", "claude-folder-trust.txt"}, "unknown screen: " + resumeReasonChoiceDialog},
		{"blocked", []string{"idle", "blocked"}, []string{"claude-enotfound.txt"}, resumeReasonHerdrBlocked},
		{"working", []string{"idle", "working"}, []string{"claude-enotfound.txt"}, resumeReasonHerdrWorking},
		{"enter only now", []string{"idle"}, []string{"claude-enotfound.txt", "claude-press-enter.txt"}, "now " + string(ResumeScreenEnterOnly)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h := resumeTestOrg(t)
			name := seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
			h.setStatus(name, tc.statuses...)
			var screens []string
			for _, f := range tc.screens {
				screens = append(screens, readResumeScreen(t, f))
			}
			h.setScreens("pane-1", screens...)
			before := len(mustReadEvents(t, o))

			res := o.Resume(ResumeParams{})
			if len(res.Errs) != 0 || len(res.Targets) != 1 {
				t.Fatalf("Resume = %+v; want one target and no error", res)
			}
			want := resumeReasonScreenChanged + " (" + tc.reason + ")"
			if got := res.Targets[0]; got.Outcome != ResumeOutcomeSkipped || got.Reason != want {
				t.Fatalf("target = %+v; want skipped: %s", got, want)
			}
			if got := resumeSends(h.calls()); len(got) != 0 {
				t.Fatalf("sends = %q; want neither the sentence nor Enter", got)
			}
			if got := len(mustReadEvents(t, o)); got != before {
				t.Fatalf("events %d -> %d; want none recorded", before, got)
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}

// TestOrgResume_TypedTextNotInTheInputBox_NoEnter covers AC-14 (iii): when
// the pane read after typing does not show the sentence in the input box,
// or cannot be read, Enter is not pressed and the seat is failed.
func TestOrgResume_TypedTextNotInTheInputBox_NoEnter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(h *fakeResumeHerdr, stopped string)
		reason string
		is     error
	}{
		{"input box still empty", func(h *fakeResumeHerdr, stopped string) { h.setScreens("pane-1", stopped) }, errResumeTextNotInInput.Error(), errResumeTextNotInInput},
		{"pane read fails after typing", func(h *fakeResumeHerdr, stopped string) {
			h.setScreens("pane-1", stopped)
			h.onSendText = func(paneID string) {
				h.mu.Lock()
				defer h.mu.Unlock()
				h.readErr[paneID] = errors.New("gone")
			}
		}, "herdr pane read pane-1 after typing failed: gone; the text was typed; Enter not pressed", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h := resumeTestOrg(t)
			name := seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
			h.setStatus(name, "idle")
			tc.setup(h, readResumeScreen(t, "claude-enotfound.txt"))
			before := len(mustReadEvents(t, o))

			res := o.Resume(ResumeParams{})
			if len(res.Targets) != 1 || len(res.Errs) != 1 || (tc.is != nil && !errors.Is(res.Errs[0], tc.is)) {
				t.Fatalf("Resume = %+v; want one failed target and its error", res)
			}
			if got := res.Targets[0]; got.Outcome != ResumeOutcomeFailed || got.Reason != tc.reason || got.Recorded {
				t.Fatalf("target = %+v; want failed: %s", got, tc.reason)
			}
			if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1"}) {
				t.Fatalf("sends = %q; want the sentence and no Enter", got)
			}
			if got := len(mustReadEvents(t, o)); got != before {
				t.Fatalf("events %d -> %d; want none recorded", before, got)
			}
		})
	}
}

// waitForResumeTest polls cond until it holds, or fails the test after 10s.
func waitForResumeTest(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// blockFirstSendText makes h's first PaneSendText wait, once it has
// recorded the call, until release is called, and put typed in the pane's
// input box when it goes on. typing is closed when that first call arrives.
// Later calls do neither.
func blockFirstSendText(t *testing.T, h *fakeResumeHerdr, typed string) (typing <-chan struct{}, release func()) {
	t.Helper()
	arrived, gate := make(chan struct{}), make(chan struct{})
	var first, open sync.Once
	release = func() { open.Do(func() { close(gate) }) }
	t.Cleanup(release)
	h.onSendText = func(paneID string) {
		first.Do(func() {
			close(arrived)
			<-gate
			h.setScreens(paneID, typed)
		})
	}
	return arrived, release
}

// TestOrgResume_TwoResumesOfOnePane_TheSecondReadsAgainAndTypesNothing
// covers AC-14: two resumes of one pane type one at a time. The first holds
// the pane lock while it types; the second looks while the first is still
// typing (the seat still idle at the stop), waits for the lock, and, holding
// it, looks again: the first resume's Enter set the seat to work, so the
// second types nothing.
func TestOrgResume_TwoResumesOfOnePane_TheSecondReadsAgainAndTypesNothing(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
	stopped := readResumeScreen(t, "claude-enotfound.txt")
	h.setStatus(name, "idle")
	h.setScreens("pane-1", stopped)
	typing, release := blockFirstSendText(t, h, readResumeScreen(t, "claude-input-typed-wrapped.txt"))
	h.onSendKeys = func(string) { h.setStatus(name, "working") }

	first := make(chan ResumeResult, 1)
	go func() { first <- o.Resume(ResumeParams{}) }()
	<-typing
	second := make(chan ResumeResult, 1)
	go func() { second <- o.Resume(ResumeParams{LockWait: 10 * time.Second}) }()
	// The first resume read the pane twice (its look, and its look under the
	// lock); the third read is the second resume's look.
	waitForResumeTest(t, "the second resume's look", func() bool { return h.readCount() >= 3 })
	release()

	a, b := <-first, <-second
	if len(a.Errs) != 0 || len(a.Targets) != 1 || a.Targets[0].Outcome != ResumeOutcomeResumed {
		t.Fatalf("first resume = %+v; want the seat resumed", a)
	}
	want := resumeReasonScreenChanged + " (" + resumeReasonHerdrWorking + ")"
	if len(b.Errs) != 0 || len(b.Targets) != 1 || b.Targets[0].Outcome != ResumeOutcomeSkipped || b.Targets[0].Reason != want {
		t.Fatalf("second resume = %+v; want the seat skipped: %s", b, want)
	}
	if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter"}) {
		t.Fatalf("sends = %q; want one sentence and one Enter", got)
	}
}

// TestOrgResume_SendWaitsForAResumeHoldingThePaneLock covers AC-14 for a
// resume and a Send: a Send to a seat a resume is typing into waits for the
// pane lock and types only after the resume pressed Enter, so the two texts
// never mix in the input box.
func TestOrgResume_SendWaitsForAResumeHoldingThePaneLock(t *testing.T) {
	o, h := resumeTestOrg(t)
	name := seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
	stopped := readResumeScreen(t, "claude-enotfound.txt")
	h.setStatus(name, "idle")
	h.setScreens("pane-1", stopped)
	typing, release := blockFirstSendText(t, h, readResumeScreen(t, "claude-input-typed-wrapped.txt"))

	resumed := make(chan ResumeResult, 1)
	go func() { resumed <- o.Resume(ResumeParams{}) }()
	<-typing
	msg := "TYPE: QUESTION\n\nwhich plan?"
	sent := make(chan SendResult, 1)
	go func() { sent <- o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: 10000}) }()
	waitForResumeTest(t, "the send's idle/done wait", func() bool {
		return slices.Contains(h.calls(), "wait "+name+" idle,done")
	})
	release()

	r, s := <-resumed, <-sent
	if len(r.Errs) != 0 || len(r.Targets) != 1 || r.Targets[0].Outcome != ResumeOutcomeResumed {
		t.Fatalf("resume = %+v; want the seat resumed", r)
	}
	if s.Err != nil || s.Progress != SendProgressEnterPressed {
		t.Fatalf("send = %+v; want it to succeed after the resume", s)
	}
	if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter", "text pane-1", "keys pane-1 Enter"}) {
		t.Fatalf("sends = %q; want the resume's sentence and Enter, then the send's", got)
	}
	if got := h.typed(); len(got) != 2 || got[0].text != ResumeText || got[1].text != msg {
		t.Fatalf("typed = %+v; want ResumeText, then the send's message", got)
	}
}

// turnWaitHerdr is fakeResumeHerdr whose AgentWait answers from the agent's
// scripted status as herdr does: at once when the status (the current entry
// of its queue, which the wait does not consume) is one of until, and
// otherwise only when ranOut is closed (herdr's own --timeout) or ctx is
// done.
type turnWaitHerdr struct {
	*fakeResumeHerdr
	ranOut chan struct{}
}

func (h *turnWaitHerdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	_, _ = h.fakeResumeHerdr.AgentWait(ctx, target, until, timeoutMS)
	h.mu.Lock()
	status := ""
	if q := h.status[target]; len(q) > 0 {
		status = q[0]
	}
	h.mu.Unlock()
	if slices.Contains(until, status) {
		return status, nil
	}
	select {
	case <-h.ranOut:
		return "", errors.New("stub: herdr agent wait --timeout ran out")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing
// (cross-review cycle 1 #1 of org-seat-resume): a Send whose idle/done wait
// passed while a resume held the pane lock waits for the lock; the resume
// presses Enter and the seat goes to work, so the Send's idle/done wait under
// the lock does not pass, and the Send types nothing, records nothing, and
// releases the lock, while the resume's sentence and Enter are all that reach
// the pane. Once the seat is idle again, a Send whose wait under the lock
// passes at once types and presses Enter as before.
func TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing(t *testing.T) {
	o, base := resumeTestOrg(t)
	h := &turnWaitHerdr{fakeResumeHerdr: base, ranOut: make(chan struct{})}
	runOut := sync.OnceFunc(func() { close(h.ranOut) })
	t.Cleanup(runOut)
	o.Herdr = h
	name := seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
	base.setStatus(name, "idle")
	base.setScreens("pane-1", readResumeScreen(t, "claude-enotfound.txt"))
	typing, release := blockFirstSendText(t, base, readResumeScreen(t, "claude-input-typed-wrapped.txt"))
	base.onSendKeys = func(string) { base.setStatus(name, "working") }
	idleDone := "wait " + name + " idle,done"

	resumed := make(chan ResumeResult, 1)
	go func() { resumed <- o.Resume(ResumeParams{}) }()
	<-typing
	msg := "TYPE: QUESTION\n\nwhich plan?"
	sent := make(chan SendResult, 1)
	go func() { sent <- o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: 10000}) }()
	waitForResumeTest(t, "the send's idle/done wait", func() bool { return countCalls(base.calls(), idleDone) >= 1 })
	release()
	if r := <-resumed; len(r.Errs) != 0 || len(r.Targets) != 1 || r.Targets[0].Outcome != ResumeOutcomeResumed {
		t.Fatalf("resume = %+v; want the seat resumed", r)
	}
	// A Send that does not wait again under the lock returns without it.
	waitForResumeTest(t, "the send's idle/done wait under the pane lock", func() bool {
		return countCalls(base.calls(), idleDone) >= 2 || len(sent) == 1
	})
	runOut()

	// The stub's --timeout ends the wait while ctx has time left, so the
	// error is the one of a failed wait, with no deadline hint.
	s := <-sent
	want := `org: send: wait for seat "seat-1" idle/done again under the lock of its pane (pane-1) failed; ` +
		`nothing was typed: stub: herdr agent wait --timeout ran out`
	if s.Err == nil || s.Err.Error() != want || s.Progress != SendProgressNothingSent || s.PaneID != "" || s.SubmitConfirmed {
		t.Fatalf("send = %+v; want %q with nothing sent", s, want)
	}
	if got := resumeSends(base.calls()); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter"}) {
		t.Fatalf("sends = %q; want only the resume's sentence and Enter", got)
	}
	if got := base.typed(); len(got) != 1 || got[0].text != ResumeText {
		t.Fatalf("typed = %+v; want ResumeText only", got)
	}
	for _, ev := range mustReadEvents(t, o) {
		if ev.Event == EventSent {
			t.Fatalf("event %+v; want no sent event", ev)
		}
	}
	mustPaneLockFree(t, o, "pane-1")

	base.setStatus(name, "idle")
	if s := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: 10000}); s.Err != nil || s.Progress != SendProgressEnterPressed {
		t.Fatalf("send once the seat is idle = %+v; want it to succeed", s)
	}
	if got := base.typed(); len(got) != 2 || got[1].text != msg {
		t.Fatalf("typed = %+v; want the send's message after ResumeText", got)
	}
	mustPaneLockFree(t, o, "pane-1")
}
