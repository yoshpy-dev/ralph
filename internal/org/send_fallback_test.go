package org

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// The tests of Send's screen fallback (send_fallback.go; plan
// docs/plans/active/2026-11-12-send-screen-fallback.md, AC1 to AC5, AC10,
// and AC11).

// sendFallbackHerdr is fakeResumeHerdr (resume_test.go) with the idle/done
// wait of a seat herdr keeps reporting working: an AgentWait for idle and
// done returns "done" once doneAfter has passed since the first AgentWait
// began, and until then waits for ctx (doneAfter 0: never), as a herdr whose
// call ctx kills does. An AgentWait for any other states (herdr's path's
// check after Enter) returns at once. fakeResumeHerdr logs every call
// ("wait <name> <until>", "get <name>", "read <pane> <lines>", ...);
// waitTimeouts are the timeouts AgentWait was given, in order. failRead,
// when > 0, makes that PaneRead (counting from 1) fail.
type sendFallbackHerdr struct {
	*fakeResumeHerdr
	doneAfter time.Duration
	failRead  int

	waitMu   sync.Mutex
	start    time.Time
	timeouts []int
}

func (h *sendFallbackHerdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	_, _ = h.fakeResumeHerdr.AgentWait(ctx, target, until, timeoutMS)
	h.waitMu.Lock()
	h.timeouts = append(h.timeouts, timeoutMS)
	if h.start.IsZero() {
		h.start = time.Now()
	}
	start := h.start
	h.waitMu.Unlock()
	if !slices.Equal(until, []string{"idle", "done"}) {
		return "working", nil
	}
	var done <-chan time.Time
	if h.doneAfter > 0 {
		done = time.After(time.Until(start.Add(h.doneAfter)))
	}
	select {
	case <-done:
		return "done", nil
	case <-ctx.Done():
		return "", fmt.Errorf("herdr: timed out: %w", ctx.Err())
	}
}

func (h *sendFallbackHerdr) PaneRead(ctx context.Context, paneID string, lines int) (string, error) {
	screen, err := h.fakeResumeHerdr.PaneRead(ctx, paneID, lines)
	if h.failRead > 0 && h.readCount() == h.failRead {
		return "", errors.New("stub: pane read failed")
	}
	return screen, err
}

func (h *sendFallbackHerdr) waitTimeouts() []int {
	h.waitMu.Lock()
	defer h.waitMu.Unlock()
	return slices.Clone(h.timeouts)
}

// sendFallbackTimeoutMS is the --timeout-ms of these tests: with
// sendFallbackOrg's reserve of 300 ms, herdr's first wait gets 600 ms.
const sendFallbackTimeoutMS = 900

// sendFallbackSlack is how much longer than --timeout-ms a Send may take in
// these tests: the deadline bounds every call, and the slack is only for the
// test process being scheduled late under load.
const sendFallbackSlack = 300 * time.Millisecond

// sendFallbackMsg is the message these tests send: a typed message of three
// lines, the second blank.
const sendFallbackMsg = "TYPE: QUESTION\n\nwhich plan?"

// sendFallbackOrg is testOrg with a sendFallbackHerdr and a seat seat-1 of
// org-a with driver in pane-1, which herdr reports working, and short
// durations for the screen fallback: a read gap of 20 ms, a reserve of
// 300 ms, a pause before Enter of 1 ms (testOrg's), and a 60 ms window for
// the check after Enter. It returns the seat's herdr agent name.
func sendFallbackOrg(t *testing.T, driver string) (*Org, *sendFallbackHerdr, string) {
	t.Helper()
	o, h := resumeTestOrg(t)
	fh := &sendFallbackHerdr{fakeResumeHerdr: h}
	o.Herdr = fh
	o.SendScreenReadGap = 20 * time.Millisecond
	o.SendScreenReserve = 300 * time.Millisecond
	o.SendSubmitConfirmTimeout = 60 * time.Millisecond
	name := seedResumeSeat(t, o, "org-a", "seat-1", "worker", driver, "pane-1")
	h.setStatus(name, "working")
	return o, fh, name
}

// sendTypedScreen is idle, a fixture whose input box is empty, with text
// typed into that box the way both CLIs show a message that fits in the box
// (evidence 2026-11-12 section 10): the first line after the marker, the
// other lines indented by two spaces.
func sendTypedScreen(t *testing.T, driver, idle, text string) string {
	t.Helper()
	empty, marker := "\n❯\n", "❯"
	if driver == "codex" {
		empty, marker = "\n› Ask Codex to do anything\n", "›"
	}
	if strings.Count(idle, empty) != 1 {
		t.Fatalf("fixture has %d empty input boxes %q; want one", strings.Count(idle, empty), empty)
	}
	typed := "\n" + marker + " " + strings.Join(strings.Split(text, "\n"), "\n  ") + "\n"
	return strings.Replace(idle, empty, typed, 1)
}

// sendFallbackNextSteps is the end of the error of a send whose screen check
// did not find the seat waiting for input (sendScreenNextSteps).
func sendFallbackNextSteps(name string) string {
	return "read the pane with `ralph org read --org-id org-a --seat seat-1`, " +
		"compare herdr's state with the screen with `ralph org status --org-id org-a --live`, " +
		"and see the rule behind herdr's state with `herdr agent explain " + name + "`"
}

// lastSentDetails is the Details of the one event o's manifest gained after
// before, which must be a `sent` event.
func lastSentDetails(t *testing.T, o *Org, before int) string {
	t.Helper()
	events := mustReadEvents(t, o)[before:]
	if len(events) != 1 || events[0].Event != EventSent || events[0].PaneID != "pane-1" {
		t.Fatalf("new events = %+v; want one sent event for pane-1", events)
	}
	return events[0].Details
}

// TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen covers
// AC1: herdr reports the seat working for all of herdr's first wait, which
// gets --timeout-ms less the reserve; the screen shows the seat waiting for
// input on two reads with the same text, and again under the pane lock, so
// Send types, finds the text in the input box, presses Enter once, confirms
// the submit on the screen (not with herdr), and records
// `via=screen herdr_state=working`, all within --timeout-ms.
func TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen(t *testing.T) {
	for _, tc := range []struct{ driver, idle, submitted string }{
		{"claude", "claude-idle-after-reply.txt", "claude-submitted-just-now.txt"},
		{"codex", "codex-idle-after-reply.txt", "codex-submitted-just-now.txt"},
	} {
		t.Run(tc.driver, func(t *testing.T) {
			o, h, name := sendFallbackOrg(t, tc.driver)
			idle := readResumeScreen(t, tc.idle)
			h.setScreens("pane-1", idle, idle, idle, sendTypedScreen(t, tc.driver, idle, sendFallbackMsg), readResumeScreen(t, tc.submitted))
			before := len(mustReadEvents(t, o))

			start := time.Now()
			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
			elapsed := time.Since(start)
			if res.Err != nil || res.Progress != SendProgressEnterPressed || !res.SubmitConfirmed || res.PaneID != "pane-1" {
				t.Fatalf("Send = %+v; want the message sent and the submit confirmed", res)
			}
			want := []string{
				"wait " + name + " idle,done",                     // herdr's first wait, which runs out
				"get " + name, "read pane-1 60", "read pane-1 60", // the screen check
				"read pane-1 60",                // again, holding the pane lock
				"text pane-1", "read pane-1 60", // the typing and the check before Enter
				"keys pane-1 Enter", "read pane-1 60", // Enter, and the check after it on the screen
			}
			if got := h.calls(); !slices.Equal(got, want) {
				t.Fatalf("herdr calls = %q\nwant %q", got, want)
			}
			if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: sendFallbackMsg}}) {
				t.Fatalf("typed = %+v; want the message once into pane-1", got)
			}
			if got := h.waitTimeouts(); !slices.Equal(got, []int{sendFallbackTimeoutMS - 300}) {
				t.Fatalf("AgentWait timeouts = %v; want only herdr's first wait, with --timeout-ms less the reserve", got)
			}
			if got, want := lastSentDetails(t, o, before), "via=screen herdr_state=working "+sendFallbackMsg; got != want {
				t.Fatalf("sent Details = %q, want %q", got, want)
			}
			if limit := sendFallbackTimeoutMS*time.Millisecond + sendFallbackSlack; elapsed > limit {
				t.Fatalf("Send took %v; want it within --timeout-ms (%d ms)", elapsed, sendFallbackTimeoutMS)
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}

// TestOrgSendScreen_ScreenNotWaiting_TypesNothingAndSaysWhy covers AC2: when
// the screen check does not find the seat waiting for input (the reads
// differ, a busy line, a choice dialog, no turn-end line, a read that
// fails), Send types nothing and waits on herdr for the rest of
// --timeout-ms; when that runs out too, the error keeps the first wait's
// text and hint and adds herdr's state, what the screen showed, and where
// to look. herdr's state is "unknown" when `herdr agent get` fails.
func TestOrgSendScreen_ScreenNotWaiting_TypesNothingAndSaysWhy(t *testing.T) {
	idle := readResumeScreen(t, "claude-idle-after-reply.txt")
	changed := strings.Replace(idle, "Cooked for 3s", "Cooked for 4s", 1)
	busy := readResumeScreen(t, "claude-working-tool.txt")
	const busyVerdict = `the screen was busy (a busy line is on the screen: "✶ Deliberating… (6s · ↓ 100 tokens)")`
	for _, tc := range []struct {
		name     string
		screens  []string
		failRead int
		getFails bool
		reads    int
		state    string
		verdict  string
	}{
		{name: "the two reads differ", screens: []string{idle, changed}, reads: 2, state: "working",
			verdict: "the screen changed between two reads 20ms apart"},
		{name: "a busy line", screens: []string{busy}, reads: 1, state: "working", verdict: busyVerdict},
		{name: "the second read is busy", screens: []string{idle, busy}, reads: 2, state: "working", verdict: busyVerdict},
		{name: "a choice dialog", screens: []string{readResumeScreen(t, "claude-folder-trust-unnumbered.txt")}, reads: 1, state: "working",
			verdict: `the screen was choice (choice dialog: "Enter to confirm · Esc to cancel")`},
		{name: "no turn-end line", screens: []string{readResumeScreen(t, "claude-prompt-quotes-error-interrupted.txt")}, reads: 1, state: "working",
			verdict: `the screen was unknown (no turn-end line after the last prompt: "⎿  Interrupted · What should Claude do instead?")`},
		{name: "the read fails", screens: []string{idle}, failRead: 1, reads: 1, state: "working",
			verdict: "the screen could not be read (herdr pane read pane-1 failed: stub: pane read failed)"},
		{name: "herdr agent get fails", screens: []string{busy}, getFails: true, reads: 1, state: "unknown", verdict: busyVerdict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, name := sendFallbackOrg(t, "claude")
			h.setScreens("pane-1", tc.screens...)
			h.failRead = tc.failRead
			if tc.getFails {
				h.getErr[name] = errors.New("stub: agent get failed")
			}
			before := len(mustReadEvents(t, o))

			start := time.Now()
			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
			elapsed := time.Since(start)
			want := `org: send: wait for seat "seat-1" idle/done: herdr: timed out: context deadline exceeded` +
				sendWaitDeadlineHint(sendFallbackTimeoutMS) +
				"; screen check: herdr state " + tc.state + ", " + tc.verdict + "; " + sendFallbackNextSteps(name)
			if res.Err == nil || res.Err.Error() != want {
				t.Fatalf("Send error =\n%v\nwant\n%s", res.Err, want)
			}
			if !errors.Is(res.Err, context.DeadlineExceeded) || !strings.Contains(res.Err.Error(), directorSkillSendWaitTimeoutPhrase) {
				t.Fatalf("Send error %v; want the deadline wrapped and the /director skill's phrase kept", res.Err)
			}
			if res.Progress != SendProgressNothingSent || res.PaneID != "" || res.SubmitConfirmed {
				t.Fatalf("Send = %+v; want nothing sent", res)
			}
			wantCalls := []string{"wait " + name + " idle,done", "get " + name}
			for range tc.reads {
				wantCalls = append(wantCalls, "read pane-1 60")
			}
			wantCalls = append(wantCalls, "wait "+name+" idle,done") // herdr's wait for the rest
			if got := h.calls(); !slices.Equal(got, wantCalls) {
				t.Fatalf("herdr calls = %q\nwant %q", got, wantCalls)
			}
			if got := h.waitTimeouts(); len(got) != 2 || got[0] != sendFallbackTimeoutMS-300 || got[1] < 1 || got[1] > 300 {
				t.Fatalf("AgentWait timeouts = %v; want %d, then at most the 300 ms left", got, sendFallbackTimeoutMS-300)
			}
			if got := len(mustReadEvents(t, o)); got != before {
				t.Fatalf("events %d -> %d; want none recorded", before, got)
			}
			if limit := sendFallbackTimeoutMS*time.Millisecond + sendFallbackSlack; elapsed > limit {
				t.Fatalf("Send took %v; want it within --timeout-ms (%d ms)", elapsed, sendFallbackTimeoutMS)
			}
		})
	}
}

// TestOrgSendScreen_ScreenChangedUnderThePaneLock_TypesNothing covers AC3:
// holding the pane lock, the screen path reads the screen once more in place
// of herdr's wait; a screen that is no longer idle, or a read that fails,
// types nothing, records nothing, and releases the lock.
func TestOrgSendScreen_ScreenChangedUnderThePaneLock_TypesNothing(t *testing.T) {
	idle := readResumeScreen(t, "claude-idle-after-reply.txt")
	for _, tc := range []struct {
		name     string
		screens  []string
		failRead int
		want     string
	}{
		{name: "the screen is busy now", screens: []string{idle, idle, readResumeScreen(t, "claude-submitted-just-now.txt")},
			want: `org: send: the screen of seat "seat-1" changed under the lock of its pane (pane-1) and is now busy (a busy line is on the screen: "✽ Bunning…"); nothing was typed`},
		{name: "the read fails", screens: []string{idle}, failRead: 3,
			want: `org: send: read the screen of seat "seat-1" again under the lock of its pane (pane-1) failed; nothing was typed: stub: pane read failed`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, name := sendFallbackOrg(t, "claude")
			h.setScreens("pane-1", tc.screens...)
			h.failRead = tc.failRead
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
			if res.Err == nil || res.Err.Error() != tc.want {
				t.Fatalf("Send error = %v\nwant %s", res.Err, tc.want)
			}
			if res.Progress != SendProgressNothingSent || res.PaneID != "" {
				t.Fatalf("Send = %+v; want nothing sent", res)
			}
			want := []string{"wait " + name + " idle,done", "get " + name, "read pane-1 60", "read pane-1 60", "read pane-1 60"}
			if got := h.calls(); !slices.Equal(got, want) {
				t.Fatalf("herdr calls = %q\nwant %q", got, want)
			}
			if got := len(mustReadEvents(t, o)); got != before {
				t.Fatalf("events %d -> %d; want none recorded", before, got)
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}

// TestOrgSendScreen_TextNotInTheInputBox_EnterNotPressed covers AC4: on the
// screen path, the pane read after the typing must show the message in the
// input box; when it does not, or the read fails, Enter is not pressed, Send
// returns SendProgressTextTyped with the pane, nothing is recorded, and the
// lock is released.
func TestOrgSendScreen_TextNotInTheInputBox_EnterNotPressed(t *testing.T) {
	idle := readResumeScreen(t, "claude-idle-after-reply.txt")
	for _, tc := range []struct {
		name     string
		failRead int
		want     string
		is       error
	}{
		{name: "the input box is empty", want: `org: send: check before Enter for seat "seat-1": the typed text was not found in the input box, so Enter was not pressed (text typed but not submitted)`,
			is: errSendTextNotInInput},
		{name: "the read fails", failRead: 4,
			want: `org: send: check before Enter for seat "seat-1": herdr pane read pane-1 after typing failed: stub: pane read failed, so Enter was not pressed (text typed but not submitted)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, name := sendFallbackOrg(t, "claude")
			h.setScreens("pane-1", idle)
			h.failRead = tc.failRead
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
			if res.Err == nil || res.Err.Error() != tc.want || (tc.is != nil && !errors.Is(res.Err, tc.is)) {
				t.Fatalf("Send error = %v\nwant %s", res.Err, tc.want)
			}
			if res.Progress != SendProgressTextTyped || res.PaneID != "pane-1" || res.SubmitConfirmed {
				t.Fatalf("Send = %+v; want SendProgressTextTyped with pane-1", res)
			}
			want := []string{
				"wait " + name + " idle,done", "get " + name, "read pane-1 60", "read pane-1 60",
				"read pane-1 60", "text pane-1", "read pane-1 60",
			}
			if got := h.calls(); !slices.Equal(got, want) {
				t.Fatalf("herdr calls = %q\nwant %q (no Enter)", got, want)
			}
			if got := len(mustReadEvents(t, o)); got != before {
				t.Fatalf("events %d -> %d; want none recorded", before, got)
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}

// TestOrgSendScreen_AfterEnter_ConfirmedOnTheScreenOnly covers AC10: on the
// screen path herdr's state is not asked after Enter (it stays working); the
// submit is confirmed only by a read of the pane in the confirmation window
// that shows the message out of the input box. A box that still holds the
// message for the whole window leaves it unconfirmed, with
// submit_unconfirmed=true after via=screen (and after raw=true), and Enter is
// pressed once all the same. A box that empties on a later read confirms it.
func TestOrgSendScreen_AfterEnter_ConfirmedOnTheScreenOnly(t *testing.T) {
	idle := readResumeScreen(t, "claude-idle-after-reply.txt")
	typed := sendTypedScreen(t, "claude", idle, sendFallbackMsg)
	submitted := readResumeScreen(t, "claude-submitted-just-now.txt")
	for _, tc := range []struct {
		name      string
		screens   []string
		raw       bool
		window    time.Duration
		confirmed bool
		minReads  int // reads after Enter
		details   string
	}{
		{name: "the message stays in the box", screens: []string{idle, idle, idle, typed}, window: 60 * time.Millisecond,
			minReads: 1, details: "via=screen herdr_state=working submit_unconfirmed=true " + sendFallbackMsg},
		{name: "raw, the message stays in the box", screens: []string{idle, idle, idle, typed}, raw: true, window: 60 * time.Millisecond,
			minReads: 1, details: "raw=true via=screen herdr_state=working submit_unconfirmed=true " + sendFallbackMsg},
		{name: "the box empties on the second read", screens: []string{idle, idle, idle, typed, typed, submitted}, window: 200 * time.Millisecond,
			confirmed: true, minReads: 2, details: "via=screen herdr_state=working " + sendFallbackMsg},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, name := sendFallbackOrg(t, "claude")
			o.SendSubmitConfirmTimeout = tc.window
			h.setScreens("pane-1", tc.screens...)
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, Raw: tc.raw, TimeoutMS: sendFallbackTimeoutMS})
			if res.Err != nil || res.Progress != SendProgressEnterPressed || res.SubmitConfirmed != tc.confirmed {
				t.Fatalf("Send = %+v; want Enter pressed, confirmed %v", res, tc.confirmed)
			}
			calls := h.calls()
			if got := resumeSends(calls); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter"}) {
				t.Fatalf("sends = %q; want the text and Enter once", got)
			}
			after := calls[slices.Index(calls, "keys pane-1 Enter")+1:]
			for _, c := range after {
				if c != "read pane-1 60" {
					t.Fatalf("calls after Enter = %q; want pane reads only (no herdr wait)", after)
				}
			}
			if len(after) < tc.minReads {
				t.Fatalf("calls after Enter = %q; want at least %d reads", after, tc.minReads)
			}
			if got := lastSentDetails(t, o, before); got != tc.details {
				t.Fatalf("sent Details = %q, want %q", got, tc.details)
			}
			for _, c := range calls {
				if strings.HasPrefix(c, "wait "+name+" working") {
					t.Fatalf("herdr calls = %q; want no wait for working on the screen path", calls)
				}
			}
		})
	}
}

// TestOrgSendScreen_ScreenBusyThenHerdrDone_SendsByHerdrsPath covers AC11:
// herdr's first wait runs out and the screen is busy, so Send waits on herdr
// again for the rest of --timeout-ms; herdr reports done within it, and Send
// types on herdr's path as before the fallback (the wait under the lock, the
// confirmation by herdr's state), with Details that carry no via=screen.
// The scale is the plan's example shrunk: working at the reserve point,
// done 50 ms later, within --timeout-ms.
func TestOrgSendScreen_ScreenBusyThenHerdrDone_SendsByHerdrsPath(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "claude")
	h.doneAfter = time.Duration(sendFallbackTimeoutMS-300+50) * time.Millisecond
	h.setScreens("pane-1", readResumeScreen(t, "claude-working-tool.txt"))
	before := len(mustReadEvents(t, o))

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
	if res.Err != nil || res.Progress != SendProgressEnterPressed || !res.SubmitConfirmed {
		t.Fatalf("Send = %+v; want the message sent on herdr's path", res)
	}
	want := []string{
		"wait " + name + " idle,done", "get " + name, "read pane-1 60", // the first wait runs out; the screen is busy
		"wait " + name + " idle,done", // herdr's wait for the rest, which reports done
		"wait " + name + " idle,done", // again, holding the pane lock
		"text pane-1", "keys pane-1 Enter", "wait " + name + " working,blocked",
	}
	if got := h.calls(); !slices.Equal(got, want) {
		t.Fatalf("herdr calls = %q\nwant %q", got, want)
	}
	if got, want := lastSentDetails(t, o, before), sendFallbackMsg; got != want {
		t.Fatalf("sent Details = %q, want %q (herdr's path)", got, want)
	}
}

// TestOrgSendScreen_HerdrDoneAtTheCheck_ScreenNotRead: a seat herdr reports
// done by the time of the screen check is left to herdr's wait, which then
// returns, and its screen is never read.
func TestOrgSendScreen_HerdrDoneAtTheCheck_ScreenNotRead(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "claude")
	h.setStatus(name, "done")
	h.doneAfter = time.Duration(sendFallbackTimeoutMS-300+50) * time.Millisecond

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
	if res.Err != nil || res.Progress != SendProgressEnterPressed {
		t.Fatalf("Send = %+v; want the message sent on herdr's path", res)
	}
	if n := h.readCount(); n != 0 {
		t.Fatalf("pane reads = %d (calls %q); want none", n, h.calls())
	}
}

// TestOrgSendScreen_NoScreenFallback_HerdrOnly covers AC5: with
// NoScreenFallback, herdr's wait gets all of --timeout-ms, the pane is never
// read, and the error is the one from before the fallback.
func TestOrgSendScreen_NoScreenFallback_HerdrOnly(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "claude")
	idle := readResumeScreen(t, "claude-idle-after-reply.txt")
	h.setScreens("pane-1", idle)

	start := time.Now()
	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS, NoScreenFallback: true})
	elapsed := time.Since(start)
	want := `org: send: wait for seat "seat-1" idle/done: herdr: timed out: context deadline exceeded` + sendWaitDeadlineHint(sendFallbackTimeoutMS)
	if res.Err == nil || res.Err.Error() != want {
		t.Fatalf("Send error = %v\nwant %s", res.Err, want)
	}
	if got := h.calls(); !slices.Equal(got, []string{"wait " + name + " idle,done"}) {
		t.Fatalf("herdr calls = %q; want herdr's wait only", got)
	}
	if got := h.waitTimeouts(); !slices.Equal(got, []int{sendFallbackTimeoutMS}) {
		t.Fatalf("AgentWait timeouts = %v; want all of --timeout-ms", got)
	}
	if elapsed < sendFallbackTimeoutMS*time.Millisecond {
		t.Fatalf("Send took %v; want herdr's wait to run all of --timeout-ms", elapsed)
	}
}

// TestOrgSendScreen_HerdrIdleAtOnce_ScreenNotRead covers the other half of
// AC5: when herdr reports the seat idle or done within its first wait, Send
// goes on as before the fallback and never reads the pane; only the first
// wait's timeout is shorter, by the reserve.
func TestOrgSendScreen_HerdrIdleAtOnce_ScreenNotRead(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "claude")
	h.doneAfter = time.Nanosecond
	before := len(mustReadEvents(t, o))

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
	if res.Err != nil || res.Progress != SendProgressEnterPressed || !res.SubmitConfirmed {
		t.Fatalf("Send = %+v; want the message sent", res)
	}
	want := []string{
		"wait " + name + " idle,done", "wait " + name + " idle,done",
		"text pane-1", "keys pane-1 Enter", "wait " + name + " working,blocked",
	}
	if got := h.calls(); !slices.Equal(got, want) {
		t.Fatalf("herdr calls = %q\nwant %q", got, want)
	}
	if got := h.waitTimeouts(); len(got) != 3 || got[0] != sendFallbackTimeoutMS-300 {
		t.Fatalf("AgentWait timeouts = %v; want the first to be --timeout-ms less the reserve", got)
	}
	if got := lastSentDetails(t, o, before); got != sendFallbackMsg {
		t.Fatalf("sent Details = %q, want %q", got, sendFallbackMsg)
	}
}

// TestOrgSendScreen_ReserveTooShortForTheCheck_HerdrOnly: when the reserve
// cannot fund the check (two read gaps and the pause before Enter), Send
// keeps none, waits on herdr for all of --timeout-ms, never reads the pane,
// and fails as before the fallback.
func TestOrgSendScreen_ReserveTooShortForTheCheck_HerdrOnly(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "claude")
	o.SendScreenReadGap = 200 * time.Millisecond // 2 x 200 + 1 > the reserve of 300

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendFallbackMsg, TimeoutMS: sendFallbackTimeoutMS})
	want := `org: send: wait for seat "seat-1" idle/done: herdr: timed out: context deadline exceeded` + sendWaitDeadlineHint(sendFallbackTimeoutMS)
	if res.Err == nil || res.Err.Error() != want {
		t.Fatalf("Send error = %v\nwant %s", res.Err, want)
	}
	if got := h.calls(); !slices.Equal(got, []string{"wait " + name + " idle,done"}) {
		t.Fatalf("herdr calls = %q; want herdr's wait only", got)
	}
	if got := h.waitTimeouts(); !slices.Equal(got, []int{sendFallbackTimeoutMS}) {
		t.Fatalf("AgentWait timeouts = %v; want all of --timeout-ms", got)
	}
}

// TestSendScreenReserve_Table pins D1 at the built-in durations: 10 s of
// --timeout-ms, at most a third of it, and none when that is less than two
// read gaps (6 s) and the pause before Enter, when the fallback is off, or
// for a driver other than claude and codex.
func TestSendScreenReserve_Table(t *testing.T) {
	o := &Org{}
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	enter := ms(DefaultSendEnterDelayMS)
	for _, tc := range []struct {
		name    string
		p       SendParams
		driver  string
		timeout time.Duration
		enter   time.Duration
		want    time.Duration
	}{
		{"the default --timeout-ms", SendParams{}, "claude", ms(30000), enter, 10 * time.Second},
		{"codex, a long --timeout-ms", SendParams{}, "codex", ms(120000), enter, 10 * time.Second},
		{"a third, just enough", SendParams{}, "claude", ms(20250), enter, ms(6750)},
		{"a third, just short", SendParams{}, "claude", ms(20249), enter, 0},
		{"a long pause before Enter", SendParams{}, "claude", ms(30000), 5 * time.Second, 0},
		{"no driver recorded", SendParams{}, "", ms(30000), enter, 0},
		{"another driver", SendParams{}, "gemini", ms(30000), enter, 0},
		{"turned off", SendParams{NoScreenFallback: true}, "claude", ms(30000), enter, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := o.sendScreenReserve(tc.p, tc.driver, tc.timeout, tc.enter); got != tc.want {
				t.Fatalf("sendScreenReserve(%v, %q, %v, %v) = %v, want %v", tc.p.NoScreenFallback, tc.driver, tc.timeout, tc.enter, got, tc.want)
			}
		})
	}
}

// TestSendScreen_InputHoldsAMultiLineProtocolMessage: the check before Enter
// (sendInputHolds) finds the first 12 runes of a message that fits in the
// box, white space dropped, in the box's joined lines, so a typed message of
// several lines, whose first line is its TYPE header, is found; an empty box
// is not. The real screens of messages taller than the box are the fixtures
// of TestSendInputHolds_TypedMultiLineFixtures.
func TestSendScreen_InputHoldsAMultiLineProtocolMessage(t *testing.T) {
	for _, tc := range []struct{ driver, idle string }{
		{"claude", "claude-idle-after-reply.txt"},
		{"codex", "codex-idle-after-reply.txt"},
	} {
		idle := readResumeScreen(t, tc.idle)
		for _, msg := range []string{sendFallbackMsg, "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "TYPE: STOP"} {
			if !sendInputHolds(tc.driver, sendTypedScreen(t, tc.driver, idle, msg), msg) {
				t.Errorf("%s: sendInputHolds of %q typed into the box = false, want true", tc.driver, msg)
			}
			if sendInputHolds(tc.driver, idle, msg) {
				t.Errorf("%s: sendInputHolds of %q with an empty box = true, want false", tc.driver, msg)
			}
		}
	}
}

// TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed covers
// AC4 for the messages a leader sends: a TASK of many lines whose start is
// not in the input box once typed (the box shows its end, or claude's paste
// placeholders; evidence 2026-11-12 section 10) passes the check before
// Enter, so Send presses Enter once, confirms the submit on the screen, and
// records via=screen.
func TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed(t *testing.T) {
	for _, tc := range []struct{ name, driver, idle, typed, submitted, msg string }{
		{"claude, the end of the message in the box", "claude",
			"claude-idle-after-reply.txt", "claude-input-typed-multiline-medium.txt", "claude-submitted-multiline-long.txt", sendProbeMediumMsg},
		{"claude, paste placeholders in the box", "claude",
			"claude-idle-after-reply.txt", "claude-input-typed-multiline-long.txt", "claude-submitted-multiline-long.txt", sendProbeLongMsg},
		{"codex, the end of the message under the scroll arrow", "codex",
			"codex-idle-after-reply.txt", "codex-input-typed-multiline-long.txt", "codex-submitted-multiline-long.txt", sendProbeLongMsg},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := sendFallbackOrg(t, tc.driver)
			idle := readResumeScreen(t, tc.idle)
			h.setScreens("pane-1", idle, idle, idle, readResumeScreen(t, tc.typed), readResumeScreen(t, tc.submitted))
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: tc.msg, TimeoutMS: sendFallbackTimeoutMS})
			if res.Err != nil || res.Progress != SendProgressEnterPressed || !res.SubmitConfirmed || res.PaneID != "pane-1" {
				t.Fatalf("Send = %+v; want the message sent and the submit confirmed", res)
			}
			if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter"}) {
				t.Fatalf("sends = %q; want the text and Enter once", got)
			}
			if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: tc.msg}}) {
				t.Fatalf("typed = %+v; want the message once into pane-1", got)
			}
			if got := lastSentDetails(t, o, before); !strings.HasPrefix(got, "via=screen herdr_state=working ") || strings.Contains(got, "submit_unconfirmed") {
				t.Fatalf("sent Details = %q; want via=screen and the submit confirmed", got)
			}
		})
	}
}

// TestSendScreenNextSteps_RepeatsTheStateDir: the commands the error names
// carry --state-dir when the CLI was given one, quoted as one shell word.
func TestSendScreenNextSteps_RepeatsTheStateDir(t *testing.T) {
	got := sendScreenNextSteps(SendParams{OrgID: "org-a", To: "seat-1", HintStateDir: "/tmp/a b"}, "org-a_seat-1")
	want := "read the pane with `ralph org read --org-id org-a --seat seat-1 --state-dir '/tmp/a b'`, " +
		"compare herdr's state with the screen with `ralph org status --org-id org-a --live --state-dir '/tmp/a b'`, " +
		"and see the rule behind herdr's state with `herdr agent explain org-a_seat-1`"
	if got != want {
		t.Fatalf("sendScreenNextSteps =\n%s\nwant\n%s", got, want)
	}
}

// TestSendScreenText_CutsALongLine: a screen line quoted in an error is cut
// to sendScreenLineMaxRunes runes and ends with an ellipsis.
func TestSendScreenText_CutsALongLine(t *testing.T) {
	long := strings.Repeat("あ", sendScreenLineMaxRunes+30)
	got := sendScreenText(SendScreen{Kind: SendScreenUnknown, Reason: "no turn-end line after the last prompt", Line: long})
	want := `unknown (no turn-end line after the last prompt: "` + strings.Repeat("あ", sendScreenLineMaxRunes) + `…")`
	if got != want {
		t.Fatalf("sendScreenText = %q, want %q", got, want)
	}
	if got := sendScreenText(SendScreen{Kind: SendScreenUnknown, Reason: "no input box"}); got != "unknown (no input box)" {
		t.Fatalf("sendScreenText with no line = %q", got)
	}
}
