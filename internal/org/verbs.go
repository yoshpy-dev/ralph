package org

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// EventSent is a non-state event (see seat.go stateEvents) recorded by the
// send verb. It never affects seat activity derivation -- it is a pure
// history/audit entry.
const EventSent = "sent"

// DefaultSendEnterDelayMS is defaultSendEnterDelay expressed in whole
// milliseconds, exported so the two hand-written doc families that state
// this value in prose -- the /org skill (.claude/skills/org/SKILL.md and
// its 3 mirrors) and the codex-seat-permissions recipe
// (docs/recipes/codex-seat-permissions.md and its template copy) -- can be
// pinned to it by send_defaults_sync_test.go instead of silently drifting
// if this constant ever changes (self-review LOW-7,
// docs/reports/self-review-2026-09-19-org-send-enter-timing.md). The CLI's
// --enter-delay-ms flag help (internal/cli/org.go) is built from this
// constant directly, so it never needs a matching update.
const DefaultSendEnterDelayMS = 750

const (
	defaultSendTimeoutMS = 30000
	defaultReadLines     = 50
	sendDetailsMaxLen    = 200
	// defaultSendEnterDelay is how long Send waits between typing the
	// message (PaneSendText) and pressing Enter (PaneSendKeys). A freshly
	// pasted multi-line message can have an immediately-following Enter
	// swallowed by the TUI as part of the paste rather than read as a
	// submit keystroke -- the #155 live evidence
	// (docs/evidence/codex-seat-permissions-2026-09-18.md, P2) saw exactly
	// this against an idle codex seat: the text landed in the input box but
	// was not submitted until an Enter sent roughly 3s later. Overridable
	// via Org.SendEnterDelay (tests: a tiny value) or
	// SendParams.EnterDelayMS (CLI: --enter-delay-ms).
	defaultSendEnterDelay = DefaultSendEnterDelayMS * time.Millisecond
	// defaultSendSubmitConfirmTimeout bounds how long Send waits, after
	// Enter, for the target seat to leave idle/done (into working or
	// blocked) -- the best available proxy for "the Enter submitted the
	// message", though not proof of it (see confirmSubmitted's doc comment
	// for the false-positive case this cannot rule out). Overridable via
	// Org.SendSubmitConfirmTimeout.
	defaultSendSubmitConfirmTimeout = 3 * time.Second
)

// findSeat returns the current derived status of (orgID, seatID) from the
// manifest, ignoring dry-run events (real seats only -- send/wait/read/stop
// act on real driver-backed seats).
func (o *Org) findSeat(orgID, seatID string) (SeatStatus, bool, error) {
	rr, err := o.Manifest.Read()
	if err != nil {
		return SeatStatus{}, false, err
	}
	seat, ok := seatFromEvents(rr.Events, orgID, seatID)
	return seat, ok, nil
}

// seatFromEvents is the pure (orgID, seatID) lookup both findSeat and Stop
// use: findSeat wraps it around its own o.Manifest.Read(); Stop reads the
// manifest once itself and passes the same events here and to codex-spawn
// correlation, rather than reading the manifest a second time (issue
// #173).
func seatFromEvents(events []ManifestEvent, orgID, seatID string) (SeatStatus, bool) {
	for _, s := range Roster(events, RosterOptions{}) {
		if s.OrgID == orgID && s.SeatID == seatID {
			return s, true
		}
	}
	return SeatStatus{}, false
}

// resolvedHerdrAgentName returns seat's persisted HerdrAgentName (recorded
// on the `spawned` event since the AC-8 tech-debt fix) if present, falling
// back to the derived name via herdrAgentName for legacy manifest events
// recorded before that field existed -- so a seat spawned by an older
// `ralph` build stays reachable by name-based verbs (compat). Every verb
// that already resolves a seat from the manifest before targeting a herdr
// agent (Send today) should call this instead of herdrAgentName directly.
func resolvedHerdrAgentName(seat SeatStatus) string {
	if seat.HerdrAgentName != "" {
		return seat.HerdrAgentName
	}
	return herdrAgentName(seat.OrgID, seat.SeatID)
}

// SendProgress reports how far Send got before returning. Two of its five
// states are deliberately "call attempted, outcome unknown": herdr calls
// run through driver.ExecRunner.Run's exec.CommandContext
// (internal/org/driver/driver.go), so a ctx deadline firing while `herdr
// pane send-text` or `herdr pane send-keys` is still in flight kills the
// herdr process and returns an error whether or not herdr had already
// delivered the paste or the keystroke. Reporting a flat "no" for those
// calls would tell an operator "not submitted, retype it" on a seat that
// may have already submitted and reached an approval dialog -- exactly the
// blind-Enter hazard this whole change exists to prevent -- so Send's
// error text and the CLI's operator notes both say only what was actually
// observed. See cross-review AR-1
// (docs/reports/cross-review-triage-org-send-enter-timing.md) for the
// incident that prompted this design.
type SendProgress int

const (
	// SendProgressNothingSent is the zero value: no pane call was ever
	// attempted. Set on protocol-validation rejection, DryRun, every
	// seat-lookup error, the idle/done-wait failure, and the fail-closed
	// --timeout-ms budget check -- every return before PaneSendText is
	// even called.
	SendProgressNothingSent SendProgress = iota
	// SendProgressTextUnacknowledged means PaneSendText was called and
	// returned an error. The text may or may not have reached the pane:
	// a ctx deadline firing mid-call kills the herdr process and reports
	// an error even if herdr had already delivered the paste.
	SendProgressTextUnacknowledged
	// SendProgressTextTyped means PaneSendText succeeded and Enter was
	// never attempted. Set only when the pre-Enter pause is cut short by
	// ctx expiring inside waitOrCtxDone -- the only way to reach this
	// state, since PaneSendKeys is called unconditionally right after a
	// successful wait. The text IS known to be sitting in the pane's
	// input box, unsubmitted.
	SendProgressTextTyped
	// SendProgressEnterUnacknowledged means PaneSendKeys("Enter") was
	// called and returned an error. Enter may or may not have been
	// delivered, for the same exec.CommandContext reason as
	// SendProgressTextUnacknowledged -- the message may or may not have
	// been submitted.
	SendProgressEnterUnacknowledged
	// SendProgressEnterPressed means PaneSendKeys("Enter") succeeded. Set
	// on exactly two returns: the appendEvent-failure error (Enter
	// succeeded and confirmSubmitted already ran; only the `sent` history
	// event could not be recorded) and the final success return.
	SendProgressEnterPressed
)

// String returns a grep-able, lower-case, hyphenated name for p. Used in
// test failure messages; production code never prints a SendProgress value
// directly -- the CLI switches on it to choose a full operator-facing
// sentence instead (see internal/cli/org.go's newOrgSendCmd).
func (p SendProgress) String() string {
	switch p {
	case SendProgressNothingSent:
		return "nothing-sent"
	case SendProgressTextUnacknowledged:
		return "text-unacknowledged"
	case SendProgressTextTyped:
		return "text-typed"
	case SendProgressEnterUnacknowledged:
		return "enter-unacknowledged"
	case SendProgressEnterPressed:
		return "enter-pressed"
	default:
		return fmt.Sprintf("SendProgress(%d)", int(p))
	}
}

// SendParams describes one `ralph org send` invocation.
type SendParams struct {
	OrgID     string
	To        string
	Text      string
	TimeoutMS int
	DryRun    bool
	// Raw bypasses AC-11 typed-protocol validation entirely, for the rare
	// case that genuinely needs to send free-form text. See
	// .claude/rules/ralph/agent-messaging.md's "Size cap" section for the
	// intended use (e.g. relaying an external tool's raw output). A
	// bypassed send still records raw=true on the `sent` event's Details,
	// so it stays traceable after the fact.
	Raw bool
	// EnterDelayMS, when > 0, overrides both defaultSendEnterDelay and
	// Org.SendEnterDelay for this one Send call (the CLI's
	// --enter-delay-ms flag). <= 0 means "not specified" -- Send falls
	// back to the Org-level setting.
	EnterDelayMS int
}

// SendResult is Send's return value.
type SendResult struct {
	Err error
	// SubmitConfirmed reports whether Send observed the target seat leave
	// idle/done (into working or blocked) within the confirmation window
	// after Enter was pressed. false does not mean the message failed to
	// send -- a very short agent turn can go working -> done before Send's
	// confirmation wait even starts, and herdr's own state reporting can
	// lag. Conversely, true does not prove OUR Enter caused the
	// transition -- see confirmSubmitted's doc comment for that caveat. A
	// false here means only that Send could not positively confirm the
	// submit, so the sent event's Details carries submit_unconfirmed=true
	// and the caller (the CLI) should tell the operator to check the pane.
	// Always false for DryRun (which never runs the confirmation) and for
	// any error return.
	SubmitConfirmed bool
	// Progress reports how far Send got before returning -- see
	// SendProgress's own doc comment for the five states and exactly which
	// return sets each one. A caller (the CLI) switches on this instead of
	// asserting a single "typed but not submitted" story for every
	// failure: two of the five states are deliberately "call attempted,
	// outcome unknown", because Send genuinely cannot tell whether a
	// ctx-cancelled herdr call reached the pane before it was killed.
	Progress SendProgress
	// PaneID is the target seat's pane id. Set whenever Progress !=
	// SendProgressNothingSent -- i.e. from the PaneSendText call onward,
	// including that call's own error return -- so a caller can still name
	// the pane to check even when Send does not know whether that call
	// succeeded. Empty for DryRun and for any SendProgressNothingSent
	// return.
	PaneID string
}

// Send validates Text against the typed message protocol (unless Raw),
// waits for the target seat to go idle, then types Text into its pane,
// waits briefly, and presses Enter exactly once. It then tries to confirm
// the submit (see confirmSubmitted) before appending a non-state `sent`
// event for history. DryRun skips the seat lookup and driver calls
// entirely and only appends the (dry_run: true) history event -- but
// protocol validation still runs first for DryRun too, since it is a
// pure, side-effect-free check; DryRun never waits or confirms, so
// SendResult.SubmitConfirmed is always false for it.
//
// AC-11: an invalid message (unless Raw) is rejected before any manifest
// event is appended and before any driver call is attempted -- Send fails
// closed, not open.
//
// No-resend design decision: Send presses Enter at most once, ever, per
// call -- see confirmSubmitted's doc comment for why a second Enter is
// unsafe.
//
// Four failure paths land on four different SendResult.Progress states
// (see SendProgress's own doc comment for the full five-state contract),
// and each needs its own operator instruction because Send genuinely knows
// different things in each case:
//
//   - PaneSendText itself failing: SendProgressTextUnacknowledged. herdr's
//     CLI can be killed by ctx expiry mid-call, so an error here does NOT
//     mean the text failed to reach the pane -- it means Send does not
//     know. The operator must check the pane before assuming anything.
//   - ctx expiring during the pre-Enter wait: SendProgressTextTyped. This
//     one IS known for certain: PaneSendText already returned success, and
//     Enter was never attempted. The message text is sitting unsubmitted
//     in the pane's input box; a retry would type a second copy on top of
//     it.
//   - PaneSendKeys itself failing: SendProgressEnterUnacknowledged. Same
//     uncertainty as the PaneSendText case, one step later: Enter may or
//     may not have reached the pane.
//   - appendEvent failing after a successful PaneSendKeys:
//     SendProgressEnterPressed. Enter WAS delivered and confirmSubmitted
//     has already run -- the message was very likely submitted, only the
//     `sent` history record was lost. Resending here risks a duplicate
//     submit or, worse, a blind Enter landing on an approval dialog the
//     submitted seat has since reached (the exact hazard confirmSubmitted's
//     doc comment explains) -- so this case must NOT be told to retype or
//     press Enter.
//
// A --timeout-ms budget too small to even fund the pre-Enter wait is
// instead caught before PaneSendText is ever called (see the check right
// after the idle/done wait below), so that case never types anything and
// leaves Progress at its SendProgressNothingSent zero value.
func (o *Org) Send(p SendParams) SendResult {
	if !p.Raw {
		if err := protocol.ValidateText(p.Text, protocol.DefaultMaxBodyChars); err != nil {
			return SendResult{Err: fmt.Errorf("org: send: message rejected by protocol validation (use --raw to bypass): %w", err)}
		}
	}

	rawPrefix := ""
	if p.Raw {
		rawPrefix = "raw=true "
	}

	if p.DryRun {
		err := o.appendEvent(ManifestEvent{
			TS: o.now(), OrgID: p.OrgID, SeatID: p.To, Event: EventSent,
			DryRun: true, Details: rawPrefix + truncateForDetails(p.Text),
		})
		return SendResult{Err: err}
	}

	seat, ok, err := o.findSeat(p.OrgID, p.To)
	if err != nil {
		return SendResult{Err: fmt.Errorf("org: send: read manifest: %w", err)}
	}
	if !ok {
		return SendResult{Err: fmt.Errorf("org: send: seat %q not found in org_id %q", p.To, p.OrgID)}
	}
	if seat.PaneID == "" {
		return SendResult{Err: fmt.Errorf("org: send: seat %q has no pane_id recorded", p.To)}
	}

	timeoutMS := p.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultSendTimeoutMS
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	name := resolvedHerdrAgentName(seat)

	// Resolved before the idle/done wait (it depends only on p/o, not on
	// the wait's outcome) so the fail-closed budget check right after that
	// wait can compare it against the ctx time actually left.
	enterDelay := o.sendEnterDelay()
	if p.EnterDelayMS > 0 {
		enterDelay = time.Duration(p.EnterDelayMS) * time.Millisecond
	}

	// Wait for "idle" OR "done": live-probed herdr (v0.7.5) reports an
	// interactive agent resting at its input prompt as "done" (turn
	// finished), not "idle" -- waiting on "idle" alone times out against a
	// perfectly receptive seat (found by the PR③ live smoke).
	if _, err := o.Herdr.AgentWait(ctx, name, []string{"idle", "done"}, timeoutMS); err != nil {
		return SendResult{Err: fmt.Errorf("org: send: wait for seat %q idle/done: %w", p.To, err)}
	}

	// Fail closed *before* typing anything: Send always builds ctx via
	// context.WithTimeout above, so a deadline is present here today -- the
	// ok-check is defence in depth in case that ever stops being true (e.g.
	// a future 0-means-unbounded --timeout-ms path on Send, mirroring
	// Wait's existing one), so a deadline-less ctx skips this check rather
	// than refusing every call with a nonsense negative figure. When a
	// deadline IS present and what remains of the --timeout-ms budget
	// cannot even fund the pre-Enter pause, typing now would strand the
	// message in the seat's composer with no way for this call to finish
	// it -- and a later retry would type a second copy on top of that
	// residue, so the seat could receive two concatenated protocol
	// messages as one (self-review MEDIUM-2,
	// docs/reports/self-review-2026-09-19-org-send-enter-timing.md).
	// Refusing here, before PaneSendText, means that case never types
	// anything at all.
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= enterDelay {
			remainingMS := int(remaining / time.Millisecond)
			if remainingMS < 0 {
				remainingMS = 0
			}
			enterDelayMS := int(enterDelay / time.Millisecond)
			return SendResult{Err: fmt.Errorf(
				"org: send: %dms of --timeout-ms left after waiting for seat %q, but the pause before Enter needs %dms; nothing was typed (raise --timeout-ms or lower --enter-delay-ms)",
				remainingMS, p.To, enterDelayMS)}
		}
	}

	if err := o.Herdr.PaneSendText(ctx, seat.PaneID, p.Text); err != nil {
		// An error here does not mean the text failed to reach the pane --
		// exec.CommandContext can kill herdr's CLI mid-call on ctx expiry,
		// so the paste may have already landed before the error came back
		// (cross-review AR-1). Progress and PaneID are still set so the
		// operator has somewhere to check instead of a flat "it failed".
		return SendResult{
			Err:      fmt.Errorf("org: send: send text to seat %q: %w (the text may or may not have reached the pane)", p.To, err),
			PaneID:   seat.PaneID,
			Progress: SendProgressTextUnacknowledged,
		}
	}

	if err := waitOrCtxDone(ctx, enterDelay); err != nil {
		// Unlike the two "unacknowledged" cases, this one IS certain:
		// PaneSendText already returned success above, and Enter is never
		// attempted below when this branch is taken.
		return SendResult{
			Err:      fmt.Errorf("org: send: waiting before Enter for seat %q: %w (text typed but not submitted)", p.To, err),
			PaneID:   seat.PaneID,
			Progress: SendProgressTextTyped,
		}
	}

	if err := o.Herdr.PaneSendKeys(ctx, seat.PaneID, "Enter"); err != nil {
		// Same exec.CommandContext uncertainty as the PaneSendText error
		// above, one step later: Enter may or may not have reached the
		// pane, so the message may or may not have been submitted.
		return SendResult{
			Err:      fmt.Errorf("org: send: send Enter to seat %q: %w (text typed; Enter was sent but not acknowledged, so the message may or may not have been submitted)", p.To, err),
			PaneID:   seat.PaneID,
			Progress: SendProgressEnterUnacknowledged,
		}
	}

	submitConfirmed := o.confirmSubmitted(ctx, name)
	details := rawPrefix
	if !submitConfirmed {
		details += "submit_unconfirmed=true "
	}
	details += truncateForDetails(p.Text)

	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.To, Event: EventSent,
		PaneID: seat.PaneID, Details: details,
	}); err != nil {
		// Enter already succeeded (and confirmSubmitted already ran) by the
		// time appendEvent runs -- the message was very likely delivered,
		// only this history record was lost. The wrapped error states only
		// what Send observed ("Enter was pressed", not "submitted": the
		// confirmation's outcome is not carried on an error return), and
		// SendProgressEnterPressed is how the CLI tells this case apart
		// from the two typed-but-unsubmitted/unacknowledged residues above:
		// it must not suggest retyping or pressing Enter.
		return SendResult{
			Err:      fmt.Errorf("org: send: Enter was pressed for seat %q but the sent event could not be recorded: %w", p.To, err),
			PaneID:   seat.PaneID,
			Progress: SendProgressEnterPressed,
		}
	}
	return SendResult{SubmitConfirmed: submitConfirmed, PaneID: seat.PaneID, Progress: SendProgressEnterPressed}
}

// sendEnterDelay returns o.SendEnterDelay, falling back to
// defaultSendEnterDelay when unset (the Org zero value) -- mirrors
// agentStartRetryInterval's pattern in spawn.go.
func (o *Org) sendEnterDelay() time.Duration {
	if o.SendEnterDelay > 0 {
		return o.SendEnterDelay
	}
	return defaultSendEnterDelay
}

// sendSubmitConfirmTimeout returns o.SendSubmitConfirmTimeout, falling back
// to defaultSendSubmitConfirmTimeout when unset.
func (o *Org) sendSubmitConfirmTimeout() time.Duration {
	if o.SendSubmitConfirmTimeout > 0 {
		return o.SendSubmitConfirmTimeout
	}
	return defaultSendSubmitConfirmTimeout
}

// waitOrCtxDone blocks for delay, honoring ctx: if ctx is done first, it
// returns ctx's error instead of waiting out the full delay. Two callers
// share this: Send's pre-Enter pause (delay is SendEnterDelay or
// SendParams.EnterDelayMS) and Spawn's codex model-observation poll
// (observeCodexSpawnReceipt, spawn.go; delay is one poll interval, clamped
// to the remaining budget). Both need the same ctx-vs-timer race -- the
// mechanism that lets --timeout-ms bound the wait too -- so it is factored
// out once, readable and independently testable rather than duplicated.
func waitOrCtxDone(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// confirmSubmitted waits, after Enter, for the target seat to leave
// idle/done (into "working" or "blocked"), and reports whether that
// happened. Both "working" and "blocked" count as "the message was
// submitted" -- "blocked" covers a seat that submit-triggered straight into
// an approval-dialog wait (see #155 evidence: a codex seat showing "Yes,
// proceed" reports blocked, not idle/done).
//
// What this does NOT prove: only that the seat left idle/done within the
// window, not that OUR Enter caused it. Something else transitioning the
// seat inside the same window -- a queued follow-up firing, a concurrent
// `ralph org send`, a background hook -- reads as confirmed even if this
// call's text is still sitting untouched in the composer. This is the
// false-positive counterpart to the false-negative case documented below
// (a very short turn finishing before this wait even starts); both mean
// SubmitConfirmed is the best available proxy for "submitted", not proof
// of it.
//
// Deliberate no-resend: this function never presses another key, no matter
// what it observes. The design it replaces was "resend Enter once if
// working is not confirmed", but that is unsafe: if the first Enter DID
// submit and the seat has already moved into an approval dialog (a
// "blocked" state) before this wait starts, a blind second Enter would
// approve that dialog -- an operation Send has no authorization to take.
// An unsent message is recoverable by resending; an approval taken without
// a human's consent is not. So a confirmation failure here (error, or the
// confirm timeout elapsing) is reported back to the caller via the bool
// return only -- Send surfaces it as submit_unconfirmed=true on the `sent`
// event and never fails the call because of it, since a very short agent
// turn can legitimately go working -> done before this wait even begins.
func (o *Org) confirmSubmitted(ctx context.Context, name string) bool {
	confirmMS := capMSToContext(ctx, o.sendSubmitConfirmTimeout())
	_, err := o.Herdr.AgentWait(ctx, name, []string{"working", "blocked"}, confirmMS)
	return err == nil
}

// capMSToContext converts want to whole milliseconds, capped at ctx's
// remaining deadline (if any) so a confirmation wait can never itself
// outlive the caller's own --timeout-ms budget. Always returns at least 1:
// driver.Herdr.AgentWait omits herdr's own `--timeout` flag entirely when
// its timeoutMS argument is <= 0, and herdr (v0.7.5, `agent wait --help`)
// documents that "without --timeout, waits indefinitely" -- 0 does not mean
// "no wait", it means "unbounded wait", which is exactly what a
// confirmation call must never risk even when the remaining ctx budget
// rounds down to zero. (Wait, verbs.go's other AgentWait caller, forwards a
// caller-supplied 0 through unchanged on purpose -- `ralph org wait
// --timeout-ms 0` is documented as "wait unbounded" -- so 0 is a valid
// choice there; it is only unsafe for this clamp.)
func capMSToContext(ctx context.Context, want time.Duration) int {
	ms := int(want / time.Millisecond)
	if ms < 1 {
		ms = 1
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := int(time.Until(deadline) / time.Millisecond)
		if remaining < 1 {
			remaining = 1
		}
		if remaining < ms {
			ms = remaining
		}
	}
	return ms
}

// truncateForDetails clips text for the manifest Details field so a long
// message body never inflates the JSONL manifest unreasonably.
func truncateForDetails(text string) string {
	if len(text) <= sendDetailsMaxLen {
		return text
	}
	return text[:sendDetailsMaxLen] + "...(truncated)"
}

// WaitParams describes one `ralph org wait` invocation. Wait is a pure
// passthrough to Herdr.AgentWait -- it never touches the manifest (see
// TestOrgWait_UnknownSeat_StillDrivesHerdr_NoManifestCheck, which pins this
// as deliberate). OrgID is required: it namespaces the herdr agent name (see
// herdrAgentName) so wait targets the seat spawned within this org_id, not
// any same-named seat in a different org_id.
//
// Deliberate scope note (AC-8/herdr_agent_name persistence, see
// resolvedHerdrAgentName): unlike Send, Wait always derives the herdr agent
// name via herdrAgentName rather than preferring a seat's persisted
// HerdrAgentName -- doing the latter would require a manifest read Wait's
// own contract explicitly forbids. herdrAgentName's naming convention has
// not changed since HerdrAgentName started being persisted (both use the
// same `<org_id>_<seat_id>` join), so this is not currently a live gap; it
// becomes one only if that convention changes again, at which point Wait
// would need to either gain a manifest lookup (breaking its no-manifest-
// touch contract) or accept staleness for respawned/renamed seats.
type WaitParams struct {
	OrgID     string
	Seat      string
	Until     []string
	TimeoutMS int
}

// WaitResult is Wait's return value.
type WaitResult struct {
	Output string
	Err    error
}

// Wait blocks until Seat reaches one of the Until states (or TimeoutMS
// elapses), returning herdr's raw output. No manifest write.
func (o *Org) Wait(p WaitParams) WaitResult {
	ctx := context.Background()
	var cancel context.CancelFunc
	if p.TimeoutMS > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(p.TimeoutMS)*time.Millisecond)
		defer cancel()
	}
	out, err := o.Herdr.AgentWait(ctx, herdrAgentName(p.OrgID, p.Seat), p.Until, p.TimeoutMS)
	return WaitResult{Output: out, Err: err}
}

// ReadParams describes one `ralph org read` invocation.
type ReadParams struct {
	OrgID string
	Seat  string
	Lines int
}

// ReadResult is Read's return value.
type ReadResult struct {
	Output string
	Err    error
}

// Read returns the last Lines of Seat's pane output. No manifest write.
func (o *Org) Read(p ReadParams) ReadResult {
	lines := p.Lines
	if lines <= 0 {
		lines = defaultReadLines
	}
	seat, ok, err := o.findSeat(p.OrgID, p.Seat)
	if err != nil {
		return ReadResult{Err: fmt.Errorf("org: read: read manifest: %w", err)}
	}
	if !ok {
		return ReadResult{Err: fmt.Errorf("org: read: seat %q not found in org_id %q", p.Seat, p.OrgID)}
	}
	if seat.PaneID == "" {
		return ReadResult{Err: fmt.Errorf("org: read: seat %q has no pane_id recorded", p.Seat)}
	}
	out, err := o.Herdr.PaneRead(context.Background(), seat.PaneID, lines)
	return ReadResult{Output: out, Err: err}
}

// defaultDriverCallTimeout bounds each herdr / agmsg call Stop and Disband
// make when Org.DriverCallTimeout is unset (zero value). A call that has not
// answered by then counts as failed, so one unresponsive herdr or agmsg can
// never hold a stop (or the stops after it) forever.
const defaultDriverCallTimeout = 10 * time.Second

// herdrPaneIDEnv is the variable herdr sets inside every pane to that pane's
// own id (alongside HERDR_ENV=1, HERDR_TAB_ID and HERDR_WORKSPACE_ID; plan
// Assumptions, confirmed live 2026-10-07). Stop compares it with the seat's
// recorded pane_id to recognise the caller's own pane.
const herdrPaneIDEnv = "HERDR_PANE_ID"

// herdrWorkspaceIDEnv is the variable herdr sets inside every pane to the id
// of the workspace that holds it (see herdrPaneIDEnv). Disband compares it
// with the org's recorded workspace ids to recognise the caller's own
// workspace.
const herdrWorkspaceIDEnv = "HERDR_WORKSPACE_ID"

// driverCallTimeout returns o.DriverCallTimeout, falling back to
// defaultDriverCallTimeout when unset (the Org zero value) -- mirrors
// sendEnterDelay's pattern.
func (o *Org) driverCallTimeout() time.Duration {
	if o.DriverCallTimeout > 0 {
		return o.DriverCallTimeout
	}
	return defaultDriverCallTimeout
}

// callWithTimeout runs one external call under its own context with
// driverCallTimeout as the deadline. The driver adapters honour ctx (the
// real Runner is exec.CommandContext), so an unanswered call comes back as
// a context error, which callers treat like any other failure.
func (o *Org) callWithTimeout(call func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), o.driverCallTimeout())
	defer cancel()
	return call(ctx)
}

// getenv reads key through o.Getenv, falling back to os.Getenv when unset.
func (o *Org) getenv(key string) string {
	if o.Getenv != nil {
		return o.Getenv(key)
	}
	return os.Getenv(key)
}

// notFoundError is how this package recognises a herdr "already closed"
// reply without importing internal/org/driver: the driver's error type has a
// NotFound method that is true for exactly the codes driver.IsNotFound
// matches (pane_not_found, tab_not_found, workspace_not_found).
type notFoundError interface{ NotFound() bool }

// isNotFound reports whether err, or an error it wraps, is a herdr not-found
// reply. Any other error, and nil, is false.
func isNotFound(err error) bool {
	var nf notFoundError
	return errors.As(err, &nf) && nf.NotFound()
}

// confirmSeatPane checks, before anything sends C-c to or closes paneID,
// that herdr still has it as seatID's pane in orgID's workspace: the tab
// holding it is labelled seatID and its workspace orgID, the labels spawn
// gave them (TabCreate with the seat id, WorkspaceCreate with the org_id).
// herdr numbers its ids from w1 again when it loses its session state, so a
// recorded id can come to name an unrelated pane (plan AC14). gone is true
// when herdr does not know the pane (already closed). A non-nil err means the
// pane is not confirmed and must be left alone: a label differs, or a get
// call failed or timed out, including a tab or workspace herdr no longer
// knows although it had the pane a moment before. Each of the three get
// calls runs under its own driverCallTimeout deadline.
func (o *Org) confirmSeatPane(orgID, seatID, paneID string) (gone bool, err error) {
	var tabID, workspaceID string
	err = o.callWithTimeout(func(ctx context.Context) error {
		var getErr error
		tabID, workspaceID, getErr = o.Herdr.PaneGet(ctx, paneID)
		return getErr
	})
	switch {
	case isNotFound(err):
		return true, nil
	case err != nil:
		return false, fmt.Errorf("check pane %q: %w", paneID, err)
	}

	var tabLabel string
	err = o.callWithTimeout(func(ctx context.Context) error {
		var getErr error
		tabLabel, getErr = o.Herdr.TabGet(ctx, tabID)
		return getErr
	})
	switch {
	case isNotFound(err):
		// Not wrapped: this is a failure, and the not-found code must not let
		// a later isNotFound read it as an already closed pane.
		return false, fmt.Errorf("check pane %q: herdr has no tab %q although it has the pane (%v)", paneID, tabID, err)
	case err != nil:
		return false, fmt.Errorf("check pane %q: %w", paneID, err)
	}
	if tabLabel != seatID {
		return false, fmt.Errorf("pane %q is not seat %q's: herdr has it in tab %q labelled %q", paneID, seatID, tabID, tabLabel)
	}

	switch wsGone, wsErr := o.confirmOrgWorkspace(orgID, workspaceID); {
	case wsErr != nil:
		return false, fmt.Errorf("pane %q is not confirmed in org_id %q's workspace: %w", paneID, orgID, wsErr)
	case wsGone:
		return false, fmt.Errorf("pane %q is not confirmed in org_id %q's workspace: herdr has no workspace %q", paneID, orgID, workspaceID)
	}
	return false, nil
}

// confirmOrgWorkspace is confirmSeatPane for a workspace: herdr's label for
// workspaceID must be orgID. gone is true when herdr does not know the
// workspace (already closed); a non-nil err (a different label, or a failed
// or timed-out get) means it must be left alone. The get runs under its own
// driverCallTimeout deadline.
func (o *Org) confirmOrgWorkspace(orgID, workspaceID string) (gone bool, err error) {
	var label string
	err = o.callWithTimeout(func(ctx context.Context) error {
		var getErr error
		label, getErr = o.Herdr.WorkspaceGet(ctx, workspaceID)
		return getErr
	})
	switch {
	case isNotFound(err):
		return true, nil
	case err != nil:
		return false, fmt.Errorf("check workspace %q: %w", workspaceID, err)
	case label != orgID:
		return false, fmt.Errorf("workspace %q is not org_id %q's: herdr labels it %q", workspaceID, orgID, label)
	}
	return false, nil
}

// notClosedNote formats a Details note for a pane or workspace left open:
// what (e.g. "pane=close failed"), " (forced)" when Force recorded it
// anyway, then the reason.
func notClosedNote(what string, err error, force bool) string {
	if force {
		return fmt.Sprintf("%s (forced): %v", what, err)
	}
	return fmt.Sprintf("%s: %v", what, err)
}

// StopParams describes one `ralph org stop` invocation.
type StopParams struct {
	OrgID  string
	Seat   string
	DryRun bool
	// Force records `stopped` even when the seat's pane could not be closed
	// (the CLI's --force, for cleaning up while herdr is unreachable). The
	// close failure is then reported through StopResult.CloseErr instead of
	// StopResult.Err.
	Force bool
}

// StopResult is Stop's return value.
type StopResult struct {
	Err error
	// CloseErr is the reason the seat's pane was not closed, nil when it was
	// (or needed no close): the ownership check did not confirm the pane as
	// the seat's (a label differs, or a get call failed or timed out; then
	// Stop sent no C-c either), or the close failed (a herdr error other
	// than not-found, or a timeout). Without Force, Err wraps the same
	// failure and the seat stays active; with Force, Err is nil and this is
	// the only place the failure shows, so the CLI prints it as a warning.
	CloseErr error
	// DeferredSelfPaneID is the seat's pane_id when that pane is the
	// caller's own (HERDR_PANE_ID) and the ownership check confirmed it as
	// the seat's: Stop sent no C-c and did not close it,
	// so the caller can finish its own output first and then close it with
	// CloseDeferredSelfPane as its very last action. Empty on every other
	// path, including a self-pane stop whose `stopped` append failed (the
	// seat is then still active in the manifest, so the pane stays open).
	DeferredSelfPaneID string
	// ModelReceipt is the Receipt this call appended, same contract as
	// SpawnResult.ModelReceipt: set when Stop's own observation
	// (observeStopModelReceipt, AC-5) finds a record and appends a
	// receipt, the zero Receipt on every other path -- no correlated
	// spawn_started at all, a seat whose correlated spawn was not a codex
	// spawn, a seat with no role-prompt file to correlate, no commanded
	// model recorded on the correlated spawn_started, a receipts-file read
	// failure, an already-observed spawn attempt, not-found, ambiguous, an
	// observation cut short by its own deadline, a dry-run stop, or a
	// receipts-append failure. The CLI layer reads this the same way it
	// reads SpawnResult.ModelReceipt, to decide whether to print the codex
	// model-mismatch warning (AC-6) without re-reading the receipts file.
	ModelReceipt Receipt
}

// Stop ends Seat's process and records it, in this order (real invocations
// only): the ownership check on the seat's pane (confirmSeatPane: its tab is
// labelled with the seat id and its workspace with the org_id), a
// best-effort C-c to the pane, `herdr pane close` on it, a best-effort agmsg
// Leave, then the manifest event. Each external call runs under its own
// driverCallTimeout deadline. Details record every outcome: `ctrl_c=...`,
// `pane=...` (closed, already closed -- herdr's not-found reply to the
// check or the close -- or no pane_id on record), and `leave=...`. A failed
// C-c alone does not fail the stop when the close succeeds. DryRun appends a
// `stopped` event without any driver call.
//
// When the pane is not confirmed as the seat's (a label differs, or a get
// call failed or timed out), Stop sends no C-c and does not close it, since
// it may be another process's pane. That, and a failed close (any herdr
// error other than not-found, including a timeout), means the pane was not
// closed: Stop does not append `stopped`, skips the Leave, so the
// still-running seat stays reachable over agmsg, appends the non-state
// EventStopFailed with the same seat fields (details `pane=not closed: ...`
// or `pane=close failed: ...`), and returns an error naming the seat and the
// reason. The seat stays active in the roster, so stopping it again once
// herdr answers closes it and records `stopped` (plan AC2). With Force, the
// seat is recorded `stopped` anyway (the same note with `(forced)`, Leave
// attempted, still no C-c or close for an unconfirmed pane) and the reason
// is reported through StopResult.CloseErr with a nil Err.
//
// The caller's own pane (the seat's pane_id equals HERDR_PANE_ID, read via
// o.Getenv, and the ownership check confirmed it) is never interrupted or
// closed here, since that would end the caller before it records anything:
// Stop skips the C-c and the close, does the Leave, appends `stopped` with
// `pane=self, closed last`, and returns the pane id in
// StopResult.DeferredSelfPaneID for CloseDeferredSelfPane (which records the
// seat active again if that close fails). A recorded pane id
// that equals HERDR_PANE_ID but fails the check is not the seat's: it is
// neither deferred nor closed, and the stop fails as above.
//
// Stop only ever closes the pane_id the manifest recorded for this seat.
//
// AC-10 (existing-seat precondition): Stop resolves Seat from the manifest
// roster *first*, for both real and dry-run invocations. A seat that was
// never spawned (never appears in the roster at all) returns an error and
// appends NO manifest event -- this is what prevents `stop --seat <unknown>`
// from fabricating a phantom `stopped` state event for a seat that never
// existed.
func (o *Org) Stop(p StopParams) StopResult {
	// One manifest read for the whole call: the seat lookup below and the
	// codex-spawn correlation inside observeStopModelReceipt both need the
	// same events, so there is no window in which the two could disagree,
	// and every seat's stop -- codex or claude -- pays for exactly one
	// read (issue #173).
	rr, err := o.Manifest.Read()
	if err != nil {
		return StopResult{Err: fmt.Errorf("org: stop: read manifest: %w", err)}
	}
	seat, ok := seatFromEvents(rr.Events, p.OrgID, p.Seat)
	if !ok {
		return StopResult{Err: fmt.Errorf("org: stop: seat %q not found in org_id %q", p.Seat, p.OrgID)}
	}

	paneID := seat.PaneID
	team := seat.AgmsgTeam
	var details string
	var closeErr error
	selfPane := false

	if p.DryRun {
		details = "dry-run: no driver call"
	} else {
		var ctrlCNote, paneNote string
		ctrlCNote, paneNote, selfPane, closeErr = o.stopSeatPane(p, paneID)

		var leaveNote string
		switch {
		case closeErr != nil && !p.Force:
			// The seat keeps running and stays active in the roster, so it
			// must stay reachable over agmsg until a later stop closes it.
			leaveNote = "leave=skipped: pane not closed"
		case team == "":
			leaveNote = "leave=skipped: no agmsg_team on record"
		default:
			if err := o.callWithTimeout(func(ctx context.Context) error {
				return o.Agmsg.Leave(ctx, team, p.Seat)
			}); err != nil {
				leaveNote = fmt.Sprintf("leave=failed: %v", err)
			} else {
				leaveNote = "leave=ok"
			}
		}

		details = ctrlCNote + " " + paneNote + " " + leaveNote
	}
	stopFailed := closeErr != nil && !p.Force

	// The codex model observation runs after the driver calls above and
	// before the `stopped` (or `stop_failed`) event append below, so its
	// outcome can ride along on that same event's Details (AC-5) instead of
	// needing a second manifest write. Only for a real (non-dry-run) stop whose
	// CORRELATED SPAWN -- the spawn that actually launched this seat, per
	// codexSpawnCorrelation, not the roster's latest state -- was a codex
	// spawn: a later rejected retry for a different driver or model must
	// never decide whether Stop even attempts an observation. See
	// observeStopModelReceipt's own doc comment for when it actually
	// observes versus reports "none" without calling the observer at all.
	var modelReceipt Receipt
	if !p.DryRun {
		if receipt, observed, isCodexSpawn := o.observeStopModelReceipt(rr.Events, p.OrgID, p.Seat); isCodexSpawn {
			token := "none"
			if observed {
				if appendErr := o.Receipts.Append(receipt); appendErr == nil {
					modelReceipt = receipt
					token = receipt.Honored // "true" or "false" -- observeStopModelReceipt never returns observed=true for an unknown outcome
				}
				// A receipts append failure leaves token at "none": nothing was
				// actually persisted, so Stop must not claim otherwise, but the
				// failure itself must never turn a successful stop into an
				// error (same contract as every other best-effort step above).
			}
			details = details + " model_observed=" + token
		}
	}

	// Role/Driver/Model here are the ROSTER's view (seat, the derived
	// SeatStatus), deliberately not the correlated spawn the observation
	// above used: the `stopped` event mirrors the roster entry it closes,
	// same as every other state event in this package, and changing that
	// contract is out of scope (plan non-goal). This means the receipt
	// appended just above -- keyed to the correlated spawn -- can legally
	// name a different driver or model than this event does, when the
	// roster's latest state is a rejected retry (issue #173). A stop_failed
	// event carries the same fields so the failure stays attributable.
	event := EventStopped
	if stopFailed {
		event = EventStopFailed
	}
	err = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.Seat, Event: event,
		Role: seat.Role, Driver: seat.Driver, Model: seat.Model, Worktree: seat.Worktree,
		PaneID: paneID, AgmsgTeam: team, DryRun: p.DryRun, Details: details,
	})
	result := StopResult{ModelReceipt: modelReceipt, CloseErr: closeErr}
	if stopFailed {
		result.Err = fmt.Errorf("org: stop: seat %q in org_id %q: pane %q not closed, seat left active: %w", p.Seat, p.OrgID, paneID, closeErr)
		if err != nil {
			result.Err = errors.Join(result.Err, fmt.Errorf("org: stop: record %s: %w", EventStopFailed, err))
		}
		return result
	}
	result.Err = err
	if selfPane && err == nil {
		result.DeferredSelfPaneID = paneID
	}
	return result
}

// stopSeatPane runs the pane half of a real Stop: the ownership check
// (confirmSeatPane), then the C-c and the close, each call under its own
// driverCallTimeout deadline. It returns the details notes for the C-c and
// the pane, whether the pane is the caller's own (confirmed, and equal to
// HERDR_PANE_ID: no C-c, no close, see Stop's doc comment), and the reason
// the pane was not closed, which is nil when it closed, herdr reported it
// not found (already closed), it is the caller's own, or no pane_id is on
// record. An unconfirmed pane gets no C-c and no close.
func (o *Org) stopSeatPane(p StopParams, paneID string) (ctrlCNote, paneNote string, selfPane bool, closeErr error) {
	if paneID == "" {
		return "ctrl_c=skipped", "pane=no pane_id on record", false, nil
	}
	gone, err := o.confirmSeatPane(p.OrgID, p.Seat, paneID)
	switch {
	case err != nil:
		return "ctrl_c=skipped: pane not confirmed", notClosedNote("pane=not closed", err, p.Force), false, err
	case gone:
		return "ctrl_c=skipped", "pane=already closed", false, nil
	case paneID == o.getenv(herdrPaneIDEnv):
		return "ctrl_c=skipped: self pane", "pane=self, closed last", true, nil
	}

	ctrlCNote = "ctrl_c=ok"
	if err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.PaneSendKeys(ctx, paneID, "C-c")
	}); err != nil {
		ctrlCNote = fmt.Sprintf("ctrl_c=failed: %v", err)
	}

	err = o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.PaneClose(ctx, paneID)
	})
	switch {
	case err == nil:
		return ctrlCNote, "pane=closed", false, nil
	case isNotFound(err):
		return ctrlCNote, "pane=already closed", false, nil
	default:
		return ctrlCNote, notClosedNote("pane=close failed", err, p.Force), false, err
	}
}

// CloseDeferredSelfPane closes the caller's own herdr pane that Stop left
// open and returned in StopResult.DeferredSelfPaneID. Closing that pane ends
// its process, which is the process calling this method (and the agent that
// ran the command), so the caller must make this its very last action, after
// every manifest write and all output; a nil return may never be seen. An
// empty paneID is a no-op.
//
// It closes only a pane the manifest recorded for a seat, and only after the
// same ownership check Stop made (confirmSeatPane against the latest real
// seat event that recorded paneID, the `stopped` Stop just wrote), so a pane
// herdr has since given to something else stays open. Each call (the gets,
// then the close) runs under its own driverCallTimeout deadline. A pane herdr
// reports not found counts as closed (nil). Any other outcome -- no seat
// recorded on paneID, a manifest read failure, an unconfirmed pane, a failed
// close -- is an error naming the pane, and the pane stays open.
//
// A pane left open means the caller is still running while the manifest
// says its seat is stopped, so a retry of the same stop, or stop --all,
// would skip it. After an unconfirmed pane or a failed close, unless force,
// it therefore appends a compensating `spawned` for that seat
// (reactivateSeat), and the error says the seat is recorded active again.
// It decides that from the manifest read again under the manifest lock
// (compensateUnderLock), not from the copy it read before the herdr calls:
// when the seat has a newer state event by then (it was spawned again in
// another pane, say), it appends nothing, and the error says the newer
// record stays. With force (the CLI's --force) it appends nothing and the
// seat stays recorded stopped. The error ends with the herdr command that
// closes the pane by hand whenever a retry would not find it: with force,
// after a manifest read failure or with no seat recorded on paneID, when a
// newer record stays, and when the lock, the second read or the
// compensating append fails.
func (o *Org) CloseDeferredSelfPane(paneID string, force bool) error {
	if paneID == "" {
		return nil
	}
	byHand := "close it by hand: herdr pane close " + paneID
	rr, err := o.Manifest.Read()
	if err != nil {
		return fmt.Errorf("org: close own pane %q: read manifest: %w; %s", paneID, err, byHand)
	}
	orgID, seatID, ok := lastSeatOnPane(rr.Events, paneID)
	if !ok {
		return fmt.Errorf("org: close own pane %q: no seat recorded on it, left open; %s", paneID, byHand)
	}
	failure := o.closeSelfPane(orgID, seatID, paneID)
	if failure == nil {
		return nil
	}
	err = fmt.Errorf("org: close own pane %q: %w", paneID, failure)
	if force {
		return fmt.Errorf("%w; seat %q of org_id %q stays recorded stopped (--force); %s", err, seatID, orgID, byHand)
	}
	why := "self pane close failed: " + failure.Error()
	var c selfCompensation
	o.compensateUnderLock(&c, func(now []ManifestEvent) {
		c.add(o.reactivateSeat(rr.Events, now, orgID, seatID, paneID, why))
	})
	return c.errorFor(err, byHand)
}

// closeSelfPane is CloseDeferredSelfPane's ownership check and close: nil
// when the pane closed or herdr does not know it, otherwise why it is still
// open.
func (o *Org) closeSelfPane(orgID, seatID, paneID string) error {
	switch gone, err := o.confirmSeatPane(orgID, seatID, paneID); {
	case err != nil:
		return fmt.Errorf("left open: %w", err)
	case gone:
		return nil
	}
	err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.PaneClose(ctx, paneID)
	})
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

// CloseDeferredSelfWorkspace closes the caller's own herdr workspace that
// Disband recorded as closed but left open, returned in
// DisbandResult.DeferredSelfWorkspaceID. Closing it closes every pane in it,
// including the caller's, so the same rule as CloseDeferredSelfPane applies:
// make it the very last action, after all output; a nil return may never be
// seen. An empty workspaceID is a no-op.
//
// It closes only a workspace the manifest recorded for an org, and only
// after the same ownership check Disband made (confirmOrgWorkspace against
// the org of the latest real workspace event for workspaceID, the
// org_workspace_closed Disband just wrote). Each call runs under its own
// driverCallTimeout deadline. A workspace herdr reports not found counts as
// closed (nil).
//
// When the workspace stays open (unconfirmed, or the close failed), the
// caller and its pane are still running, while the manifest has the
// workspace closed, the caller's seat stopped and the org disbanded. Unless
// force, it then appends a compensating org_workspace_created for the
// workspace (reopenWorkspace) and, when HERDR_PANE_ID (read via o.Getenv)
// is a seat of the same org stopped in that pane, a compensating `spawned`
// for it (reactivateSeat), so the same disband or disband --all retries
// both. When the org held a reservation before its `disbanded`, it also
// appends that reservation again (reserveAgain), so the org that is recorded
// running again holds its range again. As in CloseDeferredSelfPane, the
// three decide from the manifest read again under the manifest lock, and
// each appends nothing when that read has a newer record than the copy read
// before the herdr calls: a newer workspace event for workspaceID, a newer
// state event for the seat, or, for the reservation, one the org holds (it
// was started again with --reserve) or a different one before its latest
// `disbanded`. The error then names what it did not record again. The rest
// follows CloseDeferredSelfPane: with force it appends nothing, and the
// error names `herdr workspace close` (to run if the workspace is the
// org's) in the same cases.
func (o *Org) CloseDeferredSelfWorkspace(workspaceID string, force bool) error {
	if workspaceID == "" {
		return nil
	}
	byHand := "if it is the org's, close it by hand: herdr workspace close " + workspaceID
	rr, err := o.Manifest.Read()
	if err != nil {
		return fmt.Errorf("org: close own workspace %q: read manifest: %w; %s", workspaceID, err, byHand)
	}
	last, ok := lastWorkspaceEvent(rr.Events, workspaceID)
	if !ok {
		return fmt.Errorf("org: close own workspace %q: no org recorded on it, left open; %s", workspaceID, byHand)
	}
	failure := o.closeSelfWorkspace(last.OrgID, workspaceID)
	if failure == nil {
		return nil
	}
	err = fmt.Errorf("org: close own workspace %q: %w", workspaceID, failure)
	if force {
		return fmt.Errorf("%w; it stays recorded closed (--force); %s", err, byHand)
	}
	why := "self workspace close failed: " + failure.Error()
	var c selfCompensation
	o.compensateUnderLock(&c, func(now []ManifestEvent) {
		c.add(o.reopenWorkspace(now, last, why))
		if ownPane := o.getenv(herdrPaneIDEnv); ownPane != "" {
			if orgID, seatID, ok := lastSeatOnPane(rr.Events, ownPane); ok && orgID == last.OrgID {
				c.add(o.reactivateSeat(rr.Events, now, orgID, seatID, ownPane, why))
			}
		}
		c.add(o.reserveAgain(rr.Events, now, last.OrgID, why))
	})
	return c.errorFor(err, byHand)
}

// closeSelfWorkspace is closeSelfPane for CloseDeferredSelfWorkspace.
func (o *Org) closeSelfWorkspace(orgID, workspaceID string) error {
	switch gone, err := o.confirmOrgWorkspace(orgID, workspaceID); {
	case err != nil:
		return fmt.Errorf("left open: %w", err)
	case gone:
		return nil
	}
	err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.WorkspaceClose(ctx, workspaceID)
	})
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

// selfCompensation collects what a failed deferred self close restored in
// the manifest, what it left alone because of a newer record, and the
// appends that failed, for the error it returns.
type selfCompensation struct {
	restored []string
	skipped  []string
	failed   []error
}

// add records one compensation step's outcome: restored names what it
// recorded, skipped what it did not record because the manifest has a newer
// record for it (both "" when the copy read before the close needed
// nothing), err a failed append.
func (c *selfCompensation) add(restored, skipped string, err error) {
	if restored != "" {
		c.restored = append(c.restored, restored)
	}
	if skipped != "" {
		c.skipped = append(c.skipped, skipped)
	}
	if err != nil {
		c.failed = append(c.failed, err)
	}
}

// errorFor returns err, which names the pane or workspace left open and
// why, followed by what was recorded again, what was not because newer
// records stay, and the appends that failed. When something was not
// recorded or failed, a retry may not find what is open, so it ends with
// byHand, the herdr command that closes it by hand. With nothing restored,
// skipped or failed, the copy read before the close needed nothing, and err
// is returned as is.
func (c selfCompensation) errorFor(err error, byHand string) error {
	if len(c.restored) > 0 {
		err = fmt.Errorf("%w; recorded %s again, so running the command again retries the close", err, strings.Join(c.restored, " and "))
	}
	if len(c.skipped) > 0 {
		err = fmt.Errorf("%w; did not record %s again: newer records written to the manifest while the close waited stay as they are", err, strings.Join(c.skipped, " and "))
	}
	for _, failed := range c.failed {
		err = fmt.Errorf("%w; %w", err, failed)
	}
	if len(c.failed) > 0 || len(c.skipped) > 0 {
		err = fmt.Errorf("%w; %s", err, byHand)
	}
	return err
}

// compensateUnderLock runs compensate under the manifest lock
// (withManifestLock) with the manifest as read again there. The copy a
// deferred self close read before its herdr calls can be several
// driverCallTimeout deadlines old by the time the close has failed, and
// another command (a spawn of the same org, say) may have written newer
// records since; each compensation step compares that copy with this read
// and appends only what is still missing. The lock keeps a spawn's locked
// section (its checks, scope_reserved and spawn_started) from running
// between this read and the appends. When the lock or the read fails,
// nothing is appended, and c records the failure.
func (o *Org) compensateUnderLock(c *selfCompensation, compensate func(now []ManifestEvent)) {
	err := withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
		rr, err := o.Manifest.Read()
		if err != nil {
			return fmt.Errorf("read manifest: %w", err)
		}
		compensate(rr.Events)
		return nil
	})
	if err != nil {
		c.add("", "", fmt.Errorf("nothing recorded again: %w", err))
	}
}

// reactivateSeat appends the compensating `spawned` for seatID of orgID
// when before, the manifest copy read before the close, has the seat
// stopped in paneID (its latest real state event is `stopped` with that
// pane_id), after a deferred close left paneID open and the seat's process
// running. It appends only when now, the manifest read under the lock, still
// has that `stopped` as the seat's latest state event. The event copies the
// role, driver, model, worktree, pane_id and agmsg team of that `stopped`,
// and the herdr agent name of the latest event that recorded one (`stopped`
// does not), so Roster shows the seat as it was, active again, also after a
// later `disbanded` of the org. Details are `reactivated: <why>`. It does
// not rejoin agmsg: the record is there so a later stop closes the pane, not
// to give the seat more work. It returns, for the error text, what it
// recorded, or what it did not record because now has a newer state event
// for the seat (a spawn in another pane, for one); both are "" when the seat
// was not stopped in paneID in before.
func (o *Org) reactivateSeat(before, now []ManifestEvent, orgID, seatID, paneID, why string) (restored, skipped string, err error) {
	seat, ok := seatFromEvents(before, orgID, seatID)
	if !ok || seat.Event != EventStopped || seat.PaneID != paneID {
		return "", "", nil
	}
	what := fmt.Sprintf("seat %q of org_id %q active", seatID, orgID)
	if current, _ := seatFromEvents(now, orgID, seatID); current != seat {
		return "", what, nil
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: orgID, SeatID: seatID, Event: EventSpawned,
		Role: seat.Role, Driver: seat.Driver, Model: seat.Model, Worktree: seat.Worktree,
		PaneID: paneID, AgmsgTeam: seat.AgmsgTeam, HerdrAgentName: lastHerdrAgentName(now, orgID, seatID),
		Details: "reactivated: " + why,
	}); err != nil {
		return "", "", fmt.Errorf("record seat %q of org_id %q active again: %w", seatID, orgID, err)
	}
	return what, "", nil
}

// reopenWorkspace appends the compensating org_workspace_created for the
// workspace of last, its latest real workspace event in the manifest copy
// read before the close, when that is org_workspace_closed, after a deferred
// close left the workspace open. It appends only when now, the manifest read
// under the lock, still has last as the workspace's latest real workspace
// event. openOrgWorkspaces then lists it again, so Disband and
// orgsToDisband target the org again (also after its `disbanded`) and
// resolveWorkspace reuses it. Details are `reopened: <why>`. It returns, for
// the error text, what it recorded, or what it did not record because now
// has a newer workspace event for the workspace (herdr gave the id to a new
// workspace, for one); both are "" when last is not org_workspace_closed.
func (o *Org) reopenWorkspace(now []ManifestEvent, last ManifestEvent, why string) (restored, skipped string, err error) {
	if last.Event != EventOrgWorkspaceClosed {
		return "", "", nil
	}
	what := fmt.Sprintf("workspace %q of org_id %q open", last.PaneID, last.OrgID)
	if current, _ := lastWorkspaceEvent(now, last.PaneID); current != last {
		return "", what, nil
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: last.OrgID, SeatID: "", Event: EventOrgWorkspaceCreated,
		PaneID: last.PaneID, Details: "reopened: " + why,
	}); err != nil {
		return "", "", fmt.Errorf("record workspace %q of org_id %q open again: %w", last.PaneID, last.OrgID, err)
	}
	return what, "", nil
}

// reserveAgain appends the compensating scope_reserved for orgID after a
// deferred close left its workspace open: the reservation the org held right
// before its latest real `disbanded` in before, the manifest copy read
// before the close (reservationBeforeLastDisband), with the same paths and
// Details `paths=<paths> restored: <why>`. It appends only when now, the
// manifest read under the lock, has no reservation for the org and the same
// reservation before its latest `disbanded`; otherwise the org was started
// again (with --reserve) or disbanded again since, and the newer records
// stay. It does not check the other orgs' reservations or the org-wide
// limits: another org may have taken the range or the last org slot in the
// window since `disbanded`, and then they overlap, or max_orgs is exceeded
// by one, until this org is disbanded again (plan
// 2026-10-08-org-limits-reserve, Risks). It returns, for the error text,
// what it recorded or what it did not record; both are "" when the org held
// no reservation before its `disbanded` in before.
func (o *Org) reserveAgain(before, now []ManifestEvent, orgID, why string) (restored, skipped string, err error) {
	paths := reservationBeforeLastDisband(before, orgID)
	if paths == nil {
		return "", "", nil
	}
	what := fmt.Sprintf("the reservation %s of org_id %q", strings.Join(paths, ","), orgID)
	if ActiveReservation(now, orgID) != nil || !slices.Equal(reservationBeforeLastDisband(now, orgID), paths) {
		return "", what, nil
	}
	if err := o.appendEvent(scopeReservedEvent(o.now(), orgID, paths, "restored: "+why, false)); err != nil {
		return "", "", fmt.Errorf("record the reservation of org_id %q again: %w", orgID, err)
	}
	return what, "", nil
}

// lastSeatOnPane returns the org_id and seat_id of the latest real
// (non-dry-run) seat event that recorded paneID.
func lastSeatOnPane(events []ManifestEvent, paneID string) (orgID, seatID string, ok bool) {
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if !ev.DryRun && ev.SeatID != "" && ev.PaneID == paneID {
			return ev.OrgID, ev.SeatID, true
		}
	}
	return "", "", false
}

// lastHerdrAgentName returns the herdr agent name recorded by the latest
// real (non-dry-run) event of seatID in orgID that has one (the `spawned`
// that started it), "" when none has.
func lastHerdrAgentName(events []ManifestEvent, orgID, seatID string) string {
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if !ev.DryRun && ev.OrgID == orgID && ev.SeatID == seatID && ev.HerdrAgentName != "" {
			return ev.HerdrAgentName
		}
	}
	return ""
}

// lastWorkspaceEvent returns the latest real (non-dry-run) org-level
// workspace event (org_workspace_created or org_workspace_closed) for
// workspaceID.
func lastWorkspaceEvent(events []ManifestEvent, workspaceID string) (ManifestEvent, bool) {
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if !ev.DryRun && ev.SeatID == "" && ev.PaneID == workspaceID &&
			(ev.Event == EventOrgWorkspaceCreated || ev.Event == EventOrgWorkspaceClosed) {
			return ev, true
		}
	}
	return ManifestEvent{}, false
}

// codexSpawnInfo is codexSpawnCorrelation's structured result: everything
// Stop needs to know about the spawn that actually launched a seat --
// StartedAt, StartedTS, and PromptPath (identifying the spawn attempt and
// its role-prompt file), plus Model, Driver, and Role, all taken from that
// SAME spawn_started event. Every field here must come from this one
// event, never from the roster (Roster's derived SeatStatus): a
// `rejected` state event for a later retry of the same seat_id -- a
// different model, sometimes a different driver -- becomes the roster's
// latest entry, but it describes the REJECTED request, not the spawn that
// actually launched the seat (issue #173). A stop-time observation that
// compared against the roster's Model instead could report a false
// honored=false and a false mismatch warning for a seat that was never
// actually given that model.
type codexSpawnInfo struct {
	StartedAt  time.Time
	StartedTS  string
	PromptPath string
	Model      string
	Driver     string
	Role       string
}

// codexSpawnCorrelation resolves what Stop needs to correlate a seat with
// its codex session record, and what Stop needs to decide whether that
// seat is a codex seat at all: the seat's current (most recent)
// spawn_started event for (orgID, seatID), never the roster (see
// codexSpawnInfo's own doc comment for why the two can disagree).
// StartedTS is kept alongside StartedAt as the raw string because
// hasObservedCodexReceiptAfter compares receipt TS strings
// lexicographically, the same trick latestSeatEventTS (watch.go) already
// relies on for RFC3339 UTC timestamps. ok is false only when there is no
// spawn_started for (orgID, seatID) at all, or its TS fails to parse --
// PromptPath legitimately comes back "" (with ok=true) for a seat whose
// initial prompt was passed inline or was empty, and Model can
// legitimately come back "" for a spawn_started event recorded before
// that field existed (an old manifest); the caller (observeStopModelReceipt)
// treats "not ok", a non-codex Driver, an empty PromptPath, and an empty
// Model all as "nothing to observe here".
//
// events are assumed to be in manifest (append/chronological) order, same
// as every other roster/event-scan helper in this package -- so "the
// latest spawn_started" is simply the last matching entry seen while
// scanning forward, and only spawn_step events at or after that entry's
// index can belong to the same spawn attempt (an older attempt's
// spawn_step, from a prior stale/failed saga for the same seat, can only
// have been appended before the current attempt's own spawn_started --
// Spawn's idempotent/stale-in-flight checks are what guarantee a seat
// never has two unresolved spawn_started entries at once).
//
// Dry-run events are skipped entirely, in both loops: a `ralph org spawn
// --dry-run` for an already-spawned seat appends its own spawn_started/
// spawn_step trail with DryRun: true (dryRunSpawn, spawn.go), and package
// invariant (manifest.go's Roster doc comment) is that a dry-run event for
// a seat_id never becomes or clears a real seat's state. Without this
// filter, a later dry-run's spawn_started would look like "the latest" and
// displace the real one, so Stop would correlate against the dry-run's
// timestamp instead and silently find nothing to observe.
func codexSpawnCorrelation(events []ManifestEvent, orgID, seatID string) (codexSpawnInfo, bool) {
	startedIdx := -1
	for i, ev := range events {
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID != seatID || ev.Event != EventSpawnStarted {
			continue
		}
		startedIdx = i
	}
	if startedIdx == -1 {
		return codexSpawnInfo{}, false
	}
	spawnEv := events[startedIdx]
	t, err := time.Parse(time.RFC3339, spawnEv.TS)
	if err != nil {
		return codexSpawnInfo{}, false
	}
	info := codexSpawnInfo{
		StartedAt: t,
		StartedTS: spawnEv.TS,
		Model:     spawnEv.Model,
		Driver:    spawnEv.Driver,
		Role:      spawnEv.Role,
	}
	for i := startedIdx; i < len(events); i++ {
		ev := events[i]
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID != seatID || ev.Event != EventSpawnStep {
			continue
		}
		if path, found := promptPathFromAgentStartedDetails(ev.Details); found {
			info.PromptPath = path
		}
	}
	return info, true
}

// promptPathFromAgentStartedDetails extracts the role-prompt file path
// from one agent_started spawn_step event's Details, stripping a trailing
// " agent_start_retries=<digits>" suffix if AgentStart needed a retry
// (spawn.go's agentStartedDetails: "agent_started prompt_file=<path>[
// agent_start_retries=<N>]"). ok is false only when details does not
// start with codexPromptFileDetailsPrefix at all; a prefix with an empty
// remaining path returns ok=true, path="" -- codexSpawnCorrelation still
// assigns it, the same "nothing to correlate" signal an inline prompt
// already produces.
//
// The suffix is stripped from the END of the string only: strings.LastIndex
// finds the rightmost " agent_start_retries=" substring, and everything
// after it must be non-empty and all-digit for
// it to count as the real suffix -- so a path that legitimately contains
// spaces, or even the literal text "agent_start_retries=" in the middle
// of itself (state dirs can live under arbitrary directory names), is
// never mistaken for the suffix and survives untouched.
func promptPathFromAgentStartedDetails(details string) (path string, ok bool) {
	rest, found := strings.CutPrefix(details, codexPromptFileDetailsPrefix)
	if !found {
		return "", false
	}
	marker := " " + codexAgentStartRetriesDetailsSuffixKey
	idx := strings.LastIndex(rest, marker)
	if idx == -1 {
		return rest, true
	}
	digits := rest[idx+len(marker):]
	if digits == "" || !isAllDigits(digits) {
		return rest, true
	}
	return rest[:idx], true
}

// isAllDigits reports whether every byte of s is an ASCII digit. s is
// never empty when called from promptPathFromAgentStartedDetails (checked
// by the caller).
func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// hasObservedCodexReceiptAfter reports whether any receipt for (orgID,
// seatID), with TS strictly after spawnStartedTS, already carries a
// non-empty ReportedEffectiveModel -- i.e. an observation (Honored true or
// false, from Spawn's own poll or an earlier Stop call) already happened
// for this spawn attempt. A rejection or dry-run receipt never carries a
// reported model, so neither ever suppresses a Stop-time observation; a
// spawn attempt with no receipt at all (e.g. interrupted mid-poll) is
// likewise not suppressed, since there is nothing to compare against
// (AC-5).
//
// The comparison is strict, not "at or after": every
// timestamp in this package is RFC3339 at whole seconds (o.now()), so a
// stop followed by a re-spawn of the same seat within one real second --
// what a leader does when it replaces a seat, and what a scripted flow can
// do in milliseconds -- can leave the PREVIOUS spawn's stop-time receipt
// carrying the exact same second-string as the NEW spawn's own
// spawn_started. Under an inclusive "at or after" comparison, that old
// receipt would be mistaken for having already observed the new spawn,
// silently suppressing the new spawn's own stop-time observation. On a
// real codex seat, a receipt that genuinely belongs to THIS spawn cannot
// land in the same second as its own spawn_started -- the session record
// only appears some seconds later (the plan's own real-record
// measurement) -- so requiring strictly-after costs nothing there. If it
// ever did land in the same second, the cost of this stricter rule is one
// duplicate observed receipt at stop, not a silently missed mismatch --
// the safer direction to err in.
func hasObservedCodexReceiptAfter(receipts []Receipt, orgID, seatID, spawnStartedTS string) bool {
	for _, r := range receipts {
		if r.OrgID != orgID || r.SeatID != seatID {
			continue
		}
		if r.TS <= spawnStartedTS {
			continue
		}
		if r.ReportedEffectiveModel != "" {
			return true
		}
	}
	return false
}

// observeStopModelReceipt runs the single, non-waiting codex observation
// Stop performs (AC-5). events is the manifest read Stop already did for
// its own seat lookup (issue #173): correlation and the seat lookup must
// see the same snapshot, and re-reading the manifest here would cost every
// non-dry-run stop -- claude seats included -- a second full read.
// isCodexSpawn reports whether the spawn that actually launched (orgID,
// seatID) -- per codexSpawnCorrelation, never the roster's latest state --
// was a codex spawn; Stop only writes a model_observed= token on the
// stopped event's Details when this is true, and it is always false when
// there is no correlated spawn_started at all (a seat rejected before ever
// reaching a real spawn attempt has nothing to have been codex-shaped).
// The roster can reflect a later, unrelated rejected request for a
// different driver or model than the spawn that is actually running, and
// must never decide this (codexSpawnInfo's own doc comment).
//
// observed reports whether receipt was actually built from a found
// observation and should be appended by the caller -- always false when
// isCodexSpawn is false, and it can also be false when isCodexSpawn is
// true: no role-prompt file to correlate, no commanded model on the
// correlated spawn_started (an old manifest event recorded before that
// field existed -- nothing to compare against), a receipts-file read
// failure, an already-observed spawn attempt (hasObservedCodexReceiptAfter),
// or a not-found/ambiguous/observer-error result. Stop appends nothing in
// any of those cases (see Stop's own doc comment). When isCodexSpawn is
// true, the observation itself -- when it runs at all -- has no retry and
// no wait, unlike Spawn's own poll: Stop has already waited as long as
// the seat itself ran.
func (o *Org) observeStopModelReceipt(events []ManifestEvent, orgID, seatID string) (receipt Receipt, observed bool, isCodexSpawn bool) {
	corr, ok := codexSpawnCorrelation(events, orgID, seatID)
	if !ok || corr.Driver != "codex" {
		return Receipt{}, false, false
	}
	if corr.PromptPath == "" || corr.Model == "" {
		return Receipt{}, false, true
	}

	rec, err := o.Receipts.Read()
	if err != nil {
		return Receipt{}, false, true
	}
	if hasObservedCodexReceiptAfter(rec.Receipts, orgID, seatID, corr.StartedTS) {
		return Receipt{}, false, true
	}

	// until is Stop's own current instant, via the same injectable clock
	// spawn_started itself is read through (o.nowTime()) -- Stop can run
	// long after the spawn, so the date-directory walk needs to reach a
	// session that started well after corr.StartedAt (e.g. a codex
	// model-retirement dialog answered days later; see
	// codexSessionDateDirs's own doc comment).
	//
	// The single observation is bounded by the same budget Spawn's own
	// poll uses (o.codexModelObserveTimeout()) -- Stop has no ctx of
	// its own (every other driver call in Stop uses context.Background()
	// too), so a sessions directory large enough to make one pass slow
	// cannot make Stop itself hang: a cut-short pass here returns
	// not-found, same as a genuine miss (see ObserveCodexEffectiveModel's
	// own doc comment) -- observed stays false, and the caller (Stop)
	// appends nothing.
	obsCtx, cancel := context.WithTimeout(context.Background(), o.codexModelObserveTimeout())
	defer cancel()
	obs, _ := ObserveCodexEffectiveModel(obsCtx, o.codexSessionsDir(), corr.PromptPath, corr.StartedAt, o.nowTime())
	if obs.Status != CodexObservationFound {
		return Receipt{}, false, true
	}
	base := Receipt{
		TS: o.now(), OrgID: orgID, SeatID: seatID, Role: corr.Role, Driver: corr.Driver,
		CommandedModel: corr.Model,
	}
	rcpt := codexFoundReceipt(base, corr.Model, obs.Model)
	rcpt.Reason += " (observed at stop)"
	return rcpt, true, true
}

// StatusResult is Status's return value: the derived roster for one org_id,
// plus the count of corrupt manifest lines encountered while reading (so the
// CLI can surface it as a warning without failing the command -- AC-4).
type StatusResult struct {
	Seats        []SeatStatus
	CorruptLines int
}

// Status derives the roster for orgID from the manifest alone -- no herdr,
// no agmsg, no processes required (AC-4). all controls whether dry-run
// seats/events are included (default excludes them, per the dry-run audit
// separation design decision).
func (o *Org) Status(orgID string, all bool) (StatusResult, error) {
	rr, err := o.Manifest.Read()
	if err != nil {
		return StatusResult{}, err
	}
	full := Roster(rr.Events, RosterOptions{IncludeDryRun: all})
	seats := make([]SeatStatus, 0, len(full))
	for _, s := range full {
		if s.OrgID == orgID {
			seats = append(seats, s)
		}
	}
	return StatusResult{Seats: seats, CorruptLines: rr.CorruptLines}, nil
}

// DisbandParams describes one `ralph org disband` invocation.
type DisbandParams struct {
	OrgID  string
	DryRun bool
	// Force is the CLI's --force: every seat's Stop gets StopParams.Force,
	// a workspace whose close failed is recorded org_workspace_closed
	// anyway (Details `workspace=close failed (forced): ...`), and
	// `disbanded` is appended despite those failures. They still come back
	// in FailedSeats / FailedWorkspaces with Forced set, as warnings.
	Force bool
}

// SeatFailure is one seat Disband did not stop cleanly, and why.
type SeatFailure struct {
	SeatID string
	Err    error
	// Forced is true when the seat's pane could not be closed but Force
	// recorded it `stopped` anyway: the seat is no longer active, and Err is
	// the close failure, a warning. When false the seat is still active
	// (Stop recorded stop_failed, or Disband left it running; see Disband),
	// so the next disband stops it again, and Err is also in
	// DisbandResult.Errs.
	Forced bool
}

// WorkspaceFailure is one recorded org workspace Disband did not close, and
// why. Forced means the same as in SeatFailure: true when Force recorded
// org_workspace_closed anyway, false when the manifest still has the
// workspace open, so the next disband closes it again.
type WorkspaceFailure struct {
	WorkspaceID string
	Err         error
	Forced      bool
}

// DisbandResult is Disband's return value. After a real Disband that could
// read the manifest, each seat that was active is in exactly one of
// StoppedSeats and FailedSeats, and each workspace the manifest had open
// for the org is in exactly one of ClosedWorkspaces, FailedWorkspaces, and
// DeferredSelfWorkspaceID.
type DisbandResult struct {
	// StoppedSeats are the seats Stop recorded `stopped` with the pane
	// closed, already gone, never recorded, or left to the caller (its own).
	StoppedSeats []string
	FailedSeats  []SeatFailure
	// ClosedWorkspaces are the workspaces closed (or not found) and
	// recorded org_workspace_closed.
	ClosedWorkspaces []string
	FailedWorkspaces []WorkspaceFailure
	// Disbanded is true once the `disbanded` event was appended.
	Disbanded bool
	// DeferredSelfPaneID is the caller's own pane when Disband recorded the
	// seat in it `stopped` but left the pane open (StopResult's field of the
	// same name), and DeferredSelfWorkspaceID the caller's own workspace when
	// Disband confirmed it as the org's, recorded it org_workspace_closed,
	// and left it open. The caller
	// must finish all output and then close them with CloseDeferredSelfPane
	// / CloseDeferredSelfWorkspace as its very last action, since either
	// close ends the calling process. The own pane lies in the own
	// workspace, so when DeferredSelfWorkspaceID is set, closing the
	// workspace covers the pane and DeferredSelfPaneID is empty. Each is set
	// whenever its record was written, even if a later step failed.
	DeferredSelfPaneID      string
	DeferredSelfWorkspaceID string
	// Errs is every error that makes this disband fail: the Err of each
	// FailedSeats / FailedWorkspaces entry whose Forced is false, and
	// manifest read / append failures. It is empty exactly when Disbanded
	// is true.
	Errs []error
}

func (r *DisbandResult) recordStop(seatID string, res StopResult) {
	switch {
	case res.Err != nil:
		r.failSeat(seatID, res.Err)
	case res.CloseErr != nil:
		r.FailedSeats = append(r.FailedSeats, SeatFailure{SeatID: seatID, Err: res.CloseErr, Forced: true})
	default:
		r.StoppedSeats = append(r.StoppedSeats, seatID)
	}
	if res.DeferredSelfPaneID != "" {
		r.DeferredSelfPaneID = res.DeferredSelfPaneID
	}
}

func (r *DisbandResult) failSeat(seatID string, err error) {
	r.FailedSeats = append(r.FailedSeats, SeatFailure{SeatID: seatID, Err: err})
	r.Errs = append(r.Errs, err)
}

func (r *DisbandResult) failWorkspace(workspaceID string, err error, forced bool) {
	r.FailedWorkspaces = append(r.FailedWorkspaces, WorkspaceFailure{WorkspaceID: workspaceID, Err: err, Forced: forced})
	if !forced {
		r.Errs = append(r.Errs, err)
	}
}

// forcedNote is the `disbanded` event's Details when Force recorded past a
// close failure, naming what was not closed; empty otherwise.
func (r *DisbandResult) forcedNote() string {
	var parts []string
	for _, f := range r.FailedSeats {
		if f.Forced {
			parts = append(parts, "seat "+f.SeatID)
		}
	}
	for _, f := range r.FailedWorkspaces {
		if f.Forced {
			parts = append(parts, "workspace "+f.WorkspaceID)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "forced: not closed: " + strings.Join(parts, ", ")
}

// Disband stops every active seat of OrgID with Stop (ownership check, C-c,
// pane close, agmsg Leave, `stopped`), then closes each herdr workspace the
// manifest has open for the org (openOrgWorkspaces) and records
// org_workspace_closed for it, then appends the org-level `disbanded` event
// (SeatID empty) that marks every seat in that org_id inactive from that
// point forward (see Roster). Before closing a workspace it checks that
// herdr still labels it with the org_id (confirmOrgWorkspace), and leaves a
// workspace that fails the check open, as a workspace failure. Each
// workspace get and close runs under its own driverCallTimeout deadline,
// and a workspace herdr reports not found counts as closed.
//
// `disbanded` is appended only when nothing failed: a seat Stop could not
// close stays active (stop_failed) and is in FailedSeats, a workspace that
// was not closed stays open in the manifest and is in FailedWorkspaces, and
// the result's Errs make the CLI exit 1. Disband never
// skips a seat because an earlier one failed, but if any seat failed it
// closes no workspace, since closing one ends whatever still runs in it;
// those workspaces are listed in FailedWorkspaces. Disbanding again later
// retries what is left. With Force, close failures are recorded anyway
// (`stopped (forced)`, org_workspace_closed) and reported as Forced
// warnings, and `disbanded` is appended with a `forced: not closed: ...`
// note.
//
// The caller's own pane (HERDR_PANE_ID) and workspace (HERDR_WORKSPACE_ID),
// read via o.Getenv, are handled last, after every other close succeeded:
// the seat in the own pane is stopped only then (Stop records it without
// closing the pane), and the own workspace is only recorded
// org_workspace_closed (`workspace=self, closed last`), before `disbanded`.
// Both go through the same ownership check first: a recorded id that equals
// the caller's but fails it is neither deferred nor closed, and fails like
// any other.
// The record is written before the close because the close ends the
// calling process, which could not write anything afterwards; the caller
// closes them via DeferredSelfPaneID / DeferredSelfWorkspaceID. If that
// close fails, CloseDeferredSelfPane / CloseDeferredSelfWorkspace record
// the own seat active and the own workspace open again, so the org is a
// disband target again, and the caller shows the error. When an earlier
// close failed, the own seat and workspace are left untouched (and listed
// as failures), so a leader that ran disband in its own pane stays alive to
// see the error and retry. Telling the own pane's workspace apart relies on
// herdr setting HERDR_WORKSPACE_ID next to HERDR_PANE_ID.
//
// DryRun makes no driver call and only appends a dry-run `disbanded`.
//
// AC-10 / plan AC8: the seats come from Roster and the workspaces from
// org_workspace_created events of this org_id, so Disband only ever closes
// panes and workspaces this org's manifest recorded.
func (o *Org) Disband(p DisbandParams) DisbandResult {
	var result DisbandResult
	if p.DryRun {
		o.appendDisbanded(p, "", &result)
		return result
	}
	rr, err := o.Manifest.Read()
	if err != nil {
		result.Errs = append(result.Errs, fmt.Errorf("org: disband: read manifest: %w", err))
		return result
	}

	ownPane := o.getenv(herdrPaneIDEnv)
	var ownSeats []string
	for _, s := range Roster(rr.Events, RosterOptions{}) {
		if s.OrgID != p.OrgID || !s.Active {
			continue
		}
		if ownPane != "" && s.PaneID == ownPane {
			ownSeats = append(ownSeats, s.SeatID)
			continue
		}
		result.recordStop(s.SeatID, o.Stop(StopParams{OrgID: p.OrgID, Seat: s.SeatID, Force: p.Force}))
	}

	ownWorkspaceEnv := o.getenv(herdrWorkspaceIDEnv)
	seatsFailed := len(result.Errs) > 0
	var ownWorkspace string
	for _, ws := range openOrgWorkspaces(rr.Events, p.OrgID) {
		switch {
		case ownWorkspaceEnv != "" && ws == ownWorkspaceEnv:
			ownWorkspace = ws
		case seatsFailed:
			result.failWorkspace(ws, fmt.Errorf("org: disband: workspace %q of org_id %q left open: a seat in the org could not be stopped", ws, p.OrgID), false)
		default:
			o.closeOrgWorkspace(p, ws, false, &result)
		}
	}

	o.disbandOwnLast(p, ownSeats, ownWorkspace, &result)
	if len(result.Errs) > 0 {
		return result
	}
	o.appendDisbanded(p, result.forcedNote(), &result)
	return result
}

// disbandOwnLast is Disband's last step before `disbanded`: the seats in the
// caller's own pane and the caller's own workspace (empty: none). When
// nothing failed so far it stops those seats and, once the ownership check
// confirms the workspace, records it org_workspace_closed without closing it
// (closeOrgWorkspace with self); otherwise it leaves both as they are and
// lists them as failures. See Disband's doc comment.
func (o *Org) disbandOwnLast(p DisbandParams, ownSeats []string, ownWorkspace string, result *DisbandResult) {
	const why = "disband closes it last, after every other seat and workspace closed, and one did not"
	for _, seat := range ownSeats {
		if len(result.Errs) > 0 {
			result.failSeat(seat, fmt.Errorf("org: disband: seat %q in org_id %q left running in the caller's own pane: %s", seat, p.OrgID, why))
			continue
		}
		result.recordStop(seat, o.Stop(StopParams{OrgID: p.OrgID, Seat: seat, Force: p.Force}))
	}
	if ownWorkspace == "" {
		return
	}
	if len(result.Errs) > 0 {
		result.failWorkspace(ownWorkspace, fmt.Errorf("org: disband: the caller's own workspace %q of org_id %q left open: %s", ownWorkspace, p.OrgID, why), false)
		return
	}
	o.closeOrgWorkspace(p, ownWorkspace, true, result)
}

// closeOrgWorkspace handles one recorded workspace of p.OrgID and records
// the outcome on result. It first runs the ownership check
// (confirmOrgWorkspace: herdr's label for it is the org_id), then closes it
// -- or, for self (the caller's own workspace), only records it and hands it
// back in DeferredSelfWorkspaceID for CloseDeferredSelfWorkspace, dropping
// DeferredSelfPaneID since the own pane lies inside it. Each call runs under
// its own driverCallTimeout deadline. Closed, not found (by the check or the
// close), or deferred appends org_workspace_closed. A workspace that is not
// confirmed (a different label, or a failed or timed-out get) is neither
// closed nor deferred, and that or a failed close appends nothing and is a
// workspace failure -- unless p.Force, which appends org_workspace_closed
// with the reason in Details (`workspace=not closed (forced): ...` or
// `workspace=close failed (forced): ...`) and reports it as a Forced
// failure, still without closing an unconfirmed workspace.
func (o *Org) closeOrgWorkspace(p DisbandParams, workspaceID string, self bool, result *DisbandResult) {
	var note, failed string
	gone, closeErr := o.confirmOrgWorkspace(p.OrgID, workspaceID)
	switch {
	case closeErr != nil:
		failed = "workspace=not closed"
	case gone:
		note = "workspace=already closed"
	case self:
		note = "workspace=self, closed last"
	default:
		closeErr = o.callWithTimeout(func(ctx context.Context) error {
			return o.Herdr.WorkspaceClose(ctx, workspaceID)
		})
		switch {
		case closeErr == nil:
			note = "workspace=closed"
		case isNotFound(closeErr):
			note, closeErr = "workspace=already closed", nil
		default:
			failed = "workspace=close failed"
		}
	}
	if closeErr != nil {
		if !p.Force {
			result.failWorkspace(workspaceID, fmt.Errorf("org: disband: workspace %q of org_id %q not closed: %w", workspaceID, p.OrgID, closeErr), false)
			return
		}
		note = notClosedNote(failed, closeErr, true)
	}
	if err := o.recordWorkspaceClosed(p.OrgID, workspaceID, note); err != nil {
		result.failWorkspace(workspaceID, err, false)
		return
	}
	switch {
	case closeErr != nil:
		result.failWorkspace(workspaceID, closeErr, true)
	case self && !gone:
		result.DeferredSelfWorkspaceID = workspaceID
		result.DeferredSelfPaneID = ""
	default:
		result.ClosedWorkspaces = append(result.ClosedWorkspaces, workspaceID)
	}
}

// recordWorkspaceClosed appends the org-level org_workspace_closed event for
// workspaceID with details. A failed append leaves the workspace open in the
// manifest, so the next disband closes it again (not found then counts as
// closed).
func (o *Org) recordWorkspaceClosed(orgID, workspaceID, details string) error {
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: orgID, SeatID: "", Event: EventOrgWorkspaceClosed,
		PaneID: workspaceID, Details: details,
	}); err != nil {
		return fmt.Errorf("org: disband: record %s for workspace %q of org_id %q: %w", EventOrgWorkspaceClosed, workspaceID, orgID, err)
	}
	return nil
}

func (o *Org) appendDisbanded(p DisbandParams, details string, result *DisbandResult) {
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: "", Event: EventDisbanded, DryRun: p.DryRun, Details: details,
	}); err != nil {
		result.Errs = append(result.Errs, fmt.Errorf("org: disband: record %s: %w", EventDisbanded, err))
		return
	}
	result.Disbanded = true
}
