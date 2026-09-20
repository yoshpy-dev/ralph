package org

import (
	"context"
	"fmt"
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
	for _, s := range Roster(rr.Events, RosterOptions{}) {
		if s.OrgID == orgID && s.SeatID == seatID {
			return s, true, nil
		}
	}
	return SeatStatus{}, false, nil
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

// StopParams describes one `ralph org stop` invocation.
type StopParams struct {
	OrgID  string
	Seat   string
	DryRun bool
}

// StopResult is Stop's return value.
type StopResult struct {
	Err error
	// ModelReceipt is the Receipt this call appended, same contract as
	// SpawnResult.ModelReceipt: set when Stop's own observation
	// (observeStopModelReceipt, AC-5) finds a record and appends a
	// receipt, the zero Receipt on every other path -- not-found,
	// ambiguous, an observer error, an already-observed spawn attempt, a
	// claude seat, a dry-run stop, a seat with no role-prompt file, or a
	// receipts-append failure. The CLI layer reads this the same way it
	// reads SpawnResult.ModelReceipt, to decide whether to print the codex
	// model-mismatch warning (AC-6) without re-reading the receipts file.
	ModelReceipt Receipt
}

// Stop sends a best-effort C-c to Seat's pane and a best-effort agmsg
// Leave (real invocations only), then appends a `stopped` state event
// recording both outcomes. DryRun appends the event without attempting
// either real driver call.
//
// AC-10 (existing-seat precondition): Stop resolves Seat from the manifest
// roster *first*, for both real and dry-run invocations. A seat that was
// never spawned (never appears in the roster at all) returns an error and
// appends NO manifest event -- this is what prevents `stop --seat <unknown>`
// from fabricating a phantom `stopped` state event for a seat that never
// existed.
func (o *Org) Stop(p StopParams) StopResult {
	seat, ok, err := o.findSeat(p.OrgID, p.Seat)
	if err != nil {
		return StopResult{Err: fmt.Errorf("org: stop: read manifest: %w", err)}
	}
	if !ok {
		return StopResult{Err: fmt.Errorf("org: stop: seat %q not found in org_id %q", p.Seat, p.OrgID)}
	}

	paneID := seat.PaneID
	team := seat.AgmsgTeam
	var details string

	if p.DryRun {
		details = "dry-run: no driver call"
	} else {
		var paneNote string
		if paneID == "" {
			paneNote = "pane=no pane_id on record"
		} else if err := o.Herdr.PaneSendKeys(context.Background(), paneID, "C-c"); err != nil {
			paneNote = fmt.Sprintf("pane=failed: %v", err)
		} else {
			paneNote = "pane=ok"
		}

		var leaveNote string
		if team == "" {
			leaveNote = "leave=skipped: no agmsg_team on record"
		} else if err := o.Agmsg.Leave(context.Background(), team, p.Seat); err != nil {
			leaveNote = fmt.Sprintf("leave=failed: %v", err)
		} else {
			leaveNote = "leave=ok"
		}

		details = paneNote + " " + leaveNote
	}

	// The codex model observation runs after the driver calls above and
	// before the `stopped` event append below, so its outcome can ride
	// along on that same event's Details (AC-5) instead of needing a
	// second manifest write. Only for a real (non-dry-run) codex seat --
	// see observeStopModelReceipt's own doc comment for when it actually
	// observes versus reports "none" without calling the observer at all.
	var modelReceipt Receipt
	if !p.DryRun && seat.Driver == "codex" {
		token := "none"
		if receipt, observed := o.observeStopModelReceipt(seat, p.Seat); observed {
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

	err = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.Seat, Event: EventStopped,
		Role: seat.Role, Driver: seat.Driver, Model: seat.Model, Worktree: seat.Worktree,
		PaneID: paneID, AgmsgTeam: team, DryRun: p.DryRun, Details: details,
	})
	return StopResult{Err: err, ModelReceipt: modelReceipt}
}

// codexSpawnCorrelation resolves what Stop needs to correlate a codex
// seat with its codex session record: the TS of the seat's current (most
// recent) spawn_started event for (orgID, seatID), parsed as a time.Time,
// alongside that same TS string (hasObservedCodexReceiptAfter compares
// receipt TS strings lexicographically, the same trick latestSeatEventTS
// -- watch.go -- already relies on for RFC3339 UTC timestamps), and the
// role-prompt file path recorded on that same spawn attempt's
// agent_started spawn_step (spawn.go's codexPromptFileDetailsPrefix).
// ok is false only when there is no spawn_started for (orgID, seatID) at
// all, or its TS fails to parse -- promptPath legitimately comes back ""
// (with ok=true) for a seat whose initial prompt was passed inline or was
// empty; the caller (observeStopModelReceipt) treats both "not ok" and an
// empty promptPath as "nothing to correlate".
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
func codexSpawnCorrelation(events []ManifestEvent, orgID, seatID string) (spawnStartedAt time.Time, spawnStartedTS, promptPath string, ok bool) {
	startedIdx := -1
	for i, ev := range events {
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID != seatID || ev.Event != EventSpawnStarted {
			continue
		}
		startedIdx = i
	}
	if startedIdx == -1 {
		return time.Time{}, "", "", false
	}
	tsStr := events[startedIdx].TS
	t, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return time.Time{}, "", "", false
	}
	for i := startedIdx; i < len(events); i++ {
		ev := events[i]
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID != seatID || ev.Event != EventSpawnStep {
			continue
		}
		if path, found := promptPathFromAgentStartedDetails(ev.Details); found {
			promptPath = path
		}
	}
	return t, tsStr, promptPath, true
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
// what a lead does when it replaces a seat, and what a scripted flow can
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
// Stop performs (AC-5). It returns observed=false -- doing nothing else --
// unless every one of these holds: seat.Driver is "codex"; seatID has a
// spawn_started event with a resolvable TS and a recorded role-prompt file
// (codexSpawnCorrelation); and no receipt strictly after that spawn_started
// already carries a reported model (hasObservedCodexReceiptAfter). Only
// then does it call ObserveCodexEffectiveModel once, with no retry and no
// wait -- unlike Spawn's own poll, Stop has already waited as long as the
// seat itself ran. A not-found, ambiguous, or observer-error result also
// returns observed=false: Stop appends nothing in any of those cases (see
// Stop's own doc comment).
func (o *Org) observeStopModelReceipt(seat SeatStatus, seatID string) (Receipt, bool) {
	rr, err := o.Manifest.Read()
	if err != nil {
		return Receipt{}, false
	}
	spawnStartedAt, spawnStartedTS, promptPath, ok := codexSpawnCorrelation(rr.Events, seat.OrgID, seatID)
	if !ok || promptPath == "" {
		return Receipt{}, false
	}

	rec, err := o.Receipts.Read()
	if err != nil {
		return Receipt{}, false
	}
	if hasObservedCodexReceiptAfter(rec.Receipts, seat.OrgID, seatID, spawnStartedTS) {
		return Receipt{}, false
	}

	// until is Stop's own current instant, via the same injectable clock
	// spawn_started itself is read through (o.nowTime()) -- Stop can run
	// long after the spawn, so the date-directory walk needs to reach a
	// session that started well after spawnStartedAt (e.g. a codex
	// model-retirement dialog answered days later; see
	// codexSessionDateDirs's own doc comment).
	obs, _ := ObserveCodexEffectiveModel(o.codexSessionsDir(), promptPath, spawnStartedAt, o.nowTime())
	if obs.Status != CodexObservationFound {
		return Receipt{}, false
	}
	base := Receipt{
		TS: o.now(), OrgID: seat.OrgID, SeatID: seatID, Role: seat.Role, Driver: seat.Driver,
		CommandedModel: seat.Model,
	}
	receipt := codexFoundReceipt(base, seat.Model, obs.Model)
	receipt.Reason += " (observed at stop)"
	return receipt, true
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
}

// DisbandResult is Disband's return value: which seats were stopped (best
// effort -- Disband continues past individual Stop errors) and any errors
// encountered along the way, including from the final disbanded event.
type DisbandResult struct {
	StoppedSeats []string
	Errs         []error
}

// Disband best-effort-stops every currently active seat in OrgID (each via
// Stop, so pane C-c and agmsg Leave are both attempted per seat -- AC-5),
// then appends an org-level `disbanded` event (SeatID empty) that marks
// every seat in that org_id inactive from that point forward (see Roster).
// DryRun skips stopping real seats and only appends the disbanded event.
//
// AC-10: the seats iterated here come from Roster, which by construction
// only contains seats that actually have a recorded state event -- so
// Disband inherently only ever processes existing (never phantom/unknown)
// active seats.
func (o *Org) Disband(p DisbandParams) DisbandResult {
	var result DisbandResult

	if !p.DryRun {
		rr, err := o.Manifest.Read()
		if err != nil {
			result.Errs = append(result.Errs, fmt.Errorf("org: disband: read manifest: %w", err))
		} else {
			for _, s := range Roster(rr.Events, RosterOptions{}) {
				if s.OrgID != p.OrgID || !s.Active {
					continue
				}
				if res := o.Stop(StopParams{OrgID: p.OrgID, Seat: s.SeatID}); res.Err != nil {
					result.Errs = append(result.Errs, res.Err)
				}
				result.StoppedSeats = append(result.StoppedSeats, s.SeatID)
			}
		}
	}

	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: "", Event: EventDisbanded, DryRun: p.DryRun,
	}); err != nil {
		result.Errs = append(result.Errs, err)
	}
	return result
}
