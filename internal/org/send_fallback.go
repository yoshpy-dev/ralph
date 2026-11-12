package org

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// The screen fallback of `ralph org send` (plan
// docs/plans/active/2026-11-12-send-screen-fallback.md, D1 to D5 and D8;
// issue #308): when herdr keeps reporting a seat as working, Send reads the
// seat's screen and types only when ClassifySendScreen (send_screen.go) says
// the CLI waits for input. Send's doc comment (verbs.go) is the contract;
// this file holds its steps.

const (
	// defaultSendScreenReadGap is how long the screen check waits between
	// its two reads of the pane (D1). In the probe a running turn's screen
	// always changed over 3 s (its seconds, its spinner, a tool's elapsed
	// time), and a waiting one stayed the same over 10 to 40 s
	// (docs/evidence/org-send-screen-probe-2026-11-12.md). Overridable via
	// Org.SendScreenReadGap.
	defaultSendScreenReadGap = 3 * time.Second
	// defaultSendScreenReserve is the most of --timeout-ms that Send keeps
	// from herdr's first idle/done wait for the screen check (D1); it is
	// never more than a third of --timeout-ms either. Overridable via
	// Org.SendScreenReserve.
	defaultSendScreenReserve = 10 * time.Second
	// sendScreenConfirmInterval is how often the check after Enter on the
	// screen path reads the pane (D8), when the confirmation window is at
	// least four times as long.
	sendScreenConfirmInterval = 250 * time.Millisecond
	// sendScreenLineMaxRunes caps a screen line quoted in an error.
	sendScreenLineMaxRunes = 120
	// sendHerdrStateUnknown is the herdr state the screen check reports when
	// `herdr agent get` cannot tell it.
	sendHerdrStateUnknown = "unknown"
)

// errSendTextNotInInput is the screen path's check before Enter refusing:
// the pane read after the typing shows no input box holding the start of the
// message (ResumeInputHolds).
var errSendTextNotInInput = errors.New("the typed text was not found in the input box")

// sendScreenPath is how Send goes on once its screen check found the seat
// waiting for input (D2, D3, D8). herdrState is what herdr reported at the
// check, which the `sent` event records; checked is set once the check
// before Enter ran, so Send can tell its refusal from the pause before Enter
// running out.
type sendScreenPath struct {
	herdrState string
	checked    bool
}

// details is the start of the `sent` event's Details for a send by the
// screen, `via=screen herdr_state=<state> `, and "" for a send after
// herdr's wait (sp nil).
func (sp *sendScreenPath) details() string {
	if sp == nil {
		return ""
	}
	return "via=screen herdr_state=" + sp.herdrState + " "
}

// sendScreenCheck is what the screen check found: herdr's state at the
// check, whether the screen showed the seat waiting for input on both reads
// with the same text, and, when it did not, verdict, the words for the error
// after "screen check: herdr state <state>, ".
type sendScreenCheck struct {
	herdrState string
	idle       bool
	verdict    string
}

func (o *Org) sendScreenReadGap() time.Duration {
	if o.SendScreenReadGap > 0 {
		return o.SendScreenReadGap
	}
	return defaultSendScreenReadGap
}

func (o *Org) sendScreenReserveMax() time.Duration {
	if o.SendScreenReserve > 0 {
		return o.SendScreenReserve
	}
	return defaultSendScreenReserve
}

// sendScreenReserve is how much of timeout Send keeps from herdr's first
// idle/done wait for the screen check (D1): o's reserve (10 s by default),
// at most a third of timeout. It is 0, and Send waits on herdr alone for
// all of timeout as it did before the fallback, when p turns the fallback
// off, when driver is not claude or codex (a seat with no driver recorded
// included), and when the reserve is shorter than the least the check needs:
// two read gaps and the pause before Enter. One gap is between the two
// reads; the other stands for the rest, the herdr calls (agent get, four
// pane reads, send-text, send-keys) and the check after Enter. At the
// defaults that is 6.75 s, so a --timeout-ms under 20.25 s gets no check.
func (o *Org) sendScreenReserve(p SendParams, driver string, timeout, enterDelay time.Duration) time.Duration {
	if p.NoScreenFallback {
		return 0
	}
	if _, ok := resumeScreenSentences[driver]; !ok {
		return 0
	}
	reserve := min(o.sendScreenReserveMax(), timeout/3)
	if reserve < 2*o.sendScreenReadGap()+enterDelay {
		return 0
	}
	return reserve
}

// sendWaitForSeat is Send's wait for the seat before the pane lock (D1).
// With no reserve (sendScreenReserve), it is herdr's idle/done wait for all
// of timeoutMS, as before the fallback. With one, herdr's first wait gets
// timeoutMS less the reserve, and only when it runs out of that time (not
// when herdr fails otherwise) does Send run the screen check: a seat waiting
// for input on the screen goes on by the screen (a non-nil *sendScreenPath),
// and any other seat gets herdr's wait again for what is left of ctx. It
// returns nil and no error when herdr reported the seat idle or done, and
// the error of the wait that failed otherwise.
func (o *Org) sendWaitForSeat(ctx context.Context, p SendParams, seat SeatStatus, name string, timeoutMS int, enterDelay time.Duration) (*sendScreenPath, error) {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	reserve := o.sendScreenReserve(p, seat.Driver, timeout, enterDelay)
	if reserve == 0 {
		// Wait for "idle" OR "done": live-probed herdr (v0.7.5) reports an
		// interactive agent resting at its input prompt as "done" (turn
		// finished), not "idle" -- waiting on "idle" alone times out against
		// a perfectly receptive seat (found by the PR③ live smoke).
		if _, err := o.Herdr.AgentWait(ctx, name, []string{"idle", "done"}, timeoutMS); err != nil {
			return nil, sendWaitErr(ctx, p.To, err, timeoutMS)
		}
		return nil, nil
	}
	ranOut, err := o.sendFirstWait(ctx, name, timeout-reserve)
	if err == nil {
		return nil, nil
	}
	if !ranOut {
		return nil, sendWaitErr(ctx, p.To, err, timeoutMS)
	}
	check := o.checkSendScreen(ctx, seat, name)
	if check.idle {
		return &sendScreenPath{herdrState: check.herdrState}, nil
	}
	// Back to herdr's wait for what is left (AC11): a seat that ends its turn
	// after the first wait ran out but within --timeout-ms still gets the
	// message on herdr's path.
	if _, err := o.Herdr.AgentWait(ctx, name, []string{"idle", "done"}, capMSToContext(ctx, timeout)); err != nil {
		return nil, sendScreenWaitErr(ctx, p, name, timeoutMS, err, check)
	}
	return nil, nil
}

// sendFirstWait is herdr's first idle/done wait when Send keeps a reserve:
// wait long, on a deadline of its own inside ctx. ranOut reports whether it
// ended because that time ran out, read from the deadline, not from the
// error's text, as for the wait with no reserve: herdr's own --timeout gets
// the same ms but starts after the deadline.
func (o *Org) sendFirstWait(ctx context.Context, name string, wait time.Duration) (ranOut bool, err error) {
	waitCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	_, err = o.Herdr.AgentWait(waitCtx, name, []string{"idle", "done"}, max(int(wait/time.Millisecond), 1))
	return err != nil && errors.Is(waitCtx.Err(), context.DeadlineExceeded), err
}

// checkSendScreen is the screen check of D1 and D4, on ctx. It asks herdr
// for the seat's state (sendHerdrState); a seat herdr reports idle or done
// by now is left to herdr's wait, which then returns at once, and its screen
// is not read. Otherwise it reads the pane (sendReadIdleScreen). Only an
// idle screen is read again, after the read gap, and the seat waits for
// input only when that read is idle too and its text is the same as the
// first one's, byte for byte. A failed read, or the time running out, is not
// idle.
func (o *Org) checkSendScreen(ctx context.Context, seat SeatStatus, name string) sendScreenCheck {
	check := sendScreenCheck{herdrState: o.sendHerdrState(ctx, name)}
	if check.herdrState == "idle" || check.herdrState == "done" {
		check.verdict = "the screen was not read"
		return check
	}
	first, verdict := o.sendReadIdleScreen(ctx, seat)
	if verdict != "" {
		check.verdict = verdict
		return check
	}
	gap := o.sendScreenReadGap()
	if err := waitOrCtxDone(ctx, gap); err != nil {
		check.verdict = "the screen check ran out of --timeout-ms between its two reads"
		return check
	}
	second, verdict := o.sendReadIdleScreen(ctx, seat)
	switch {
	case verdict != "":
		check.verdict = verdict
	case second != first:
		check.verdict = "the screen changed between two reads " + gap.String() + " apart"
	default:
		check.idle = true
	}
	return check
}

// sendReadIdleScreen reads seat's pane (resumeReadLines lines, as Resume
// does) and classifies it with ClassifySendScreen for the seat's driver. It
// returns the screen's text when it is idle, and otherwise the verdict for
// checkSendScreen: what the screen was, or that it could not be read.
func (o *Org) sendReadIdleScreen(ctx context.Context, seat SeatStatus) (screen, verdict string) {
	screen, err := o.Herdr.PaneRead(ctx, seat.PaneID, resumeReadLines)
	if err != nil {
		return "", fmt.Sprintf("the screen could not be read (herdr pane read %s failed: %v)", seat.PaneID, err)
	}
	if s := ClassifySendScreen(seat.Driver, screen); s.Kind != SendScreenIdle {
		return "", "the screen was " + sendScreenText(s)
	}
	return screen, ""
}

// sendHerdrState is the seat's agent_status by `herdr agent get` on ctx
// (herdrAgentStatus, as Resume reads it), or sendHerdrStateUnknown when the
// herdr client cannot run it, the call fails, or the reply has no status.
func (o *Org) sendHerdrState(ctx context.Context, name string) string {
	probe, ok := o.Herdr.(watchHerdrProbe)
	if !ok {
		return sendHerdrStateUnknown
	}
	out, err := probe.AgentGet(ctx, name)
	if err != nil {
		return sendHerdrStateUnknown
	}
	status, err := herdrAgentStatus(out)
	if err != nil {
		return sendHerdrStateUnknown
	}
	return status
}

// sendScreenUnderLock is the screen path's check holding the pane lock, in
// place of herdr's wait under it (D2, AC3): herdr still reports the seat
// working, so that wait would only run out. The screen is read once more and
// must still be idle; otherwise nothing is typed, and the caller's deferred
// unlock releases the lock.
func (o *Org) sendScreenUnderLock(ctx context.Context, p SendParams, seat SeatStatus) error {
	screen, err := o.Herdr.PaneRead(ctx, seat.PaneID, resumeReadLines)
	if err != nil {
		return fmt.Errorf("org: send: read the screen of seat %q again under the lock of its pane (%s) failed; nothing was typed: %w", p.To, seat.PaneID, err)
	}
	if s := ClassifySendScreen(seat.Driver, screen); s.Kind != SendScreenIdle {
		return fmt.Errorf("org: send: the screen of seat %q changed under the lock of its pane (%s) and is now %s; nothing was typed", p.To, seat.PaneID, sendScreenText(s))
	}
	return nil
}

// sendScreenSubmitChecks returns the check before Enter and the check after
// it that typeAndSubmit runs on the screen path, and nil, nil for a send
// after herdr's wait (sp nil), which keeps typeAndSubmit's own.
//
// Before Enter (D3, AC4), the pane is read and its input box must hold the
// start of text (ResumeInputHolds, which drops white space and compares the
// first 12 runes, so a message of several lines matches the box's joined
// lines); otherwise Enter is not pressed. After Enter (D8, AC10), herdr's
// state says nothing, since herdr kept reporting working: the pane is read
// at once and then every sendScreenConfirmInterval (a quarter of the window
// when that is shorter), for no longer than the submit confirmation window
// (sendSubmitConfirmTimeout, 3 s by default) and what is left of ctx, and
// the submit is confirmed only when SendScreenSubmitted holds. A read that
// fails counts as not confirmed. Neither presses a key.
func (o *Org) sendScreenSubmitChecks(sp *sendScreenPath, seat SeatStatus, text string) (func(context.Context) error, func(context.Context) bool) {
	if sp == nil {
		return nil, nil
	}
	beforeEnter := func(ctx context.Context) error {
		sp.checked = true
		screen, err := o.Herdr.PaneRead(ctx, seat.PaneID, resumeReadLines)
		if err != nil {
			return fmt.Errorf("herdr pane read %s after typing failed: %w", seat.PaneID, err)
		}
		if !ResumeInputHolds(seat.Driver, screen, text) {
			return errSendTextNotInInput
		}
		return nil
	}
	confirm := func(ctx context.Context) bool {
		window := o.sendSubmitConfirmTimeout()
		interval := min(sendScreenConfirmInterval, window/4)
		ctx, cancel := context.WithTimeout(ctx, window)
		defer cancel()
		for {
			screen, err := o.Herdr.PaneRead(ctx, seat.PaneID, resumeReadLines)
			if err == nil && SendScreenSubmitted(seat.Driver, screen, text) {
				return true
			}
			if waitOrCtxDone(ctx, interval) != nil {
				return false
			}
		}
	}
	return beforeEnter, confirm
}

// sendWaitErr is the error of a failed idle/done wait with no screen check:
// "org: send: wait for seat %q idle/done: " and the cause, followed by
// sendWaitDeadlineHint when ctx's deadline ran out (read from ctx, not from
// err's text: herdr's own --timeout gets the same ms but starts after ctx,
// so ctx has expired by the time either one ends the wait).
func sendWaitErr(ctx context.Context, to string, err error, timeoutMS int) error {
	hint := ""
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		hint = sendWaitDeadlineHint(timeoutMS)
	}
	return fmt.Errorf("org: send: wait for seat %q idle/done: %w%s", to, err, hint)
}

// sendScreenWaitErr is the error when herdr's wait after a screen check that
// did not find the seat waiting for input fails too (AC2): sendWaitErr's
// text, then "; screen check: herdr state <state>, " and check's verdict,
// then where to look (sendScreenNextSteps).
func sendScreenWaitErr(ctx context.Context, p SendParams, name string, timeoutMS int, err error, check sendScreenCheck) error {
	return fmt.Errorf("%w; screen check: herdr state %s, %s; %s",
		sendWaitErr(ctx, p.To, err, timeoutMS), check.herdrState, check.verdict, sendScreenNextSteps(p, name))
}

// sendScreenNextSteps names the commands that show why the seat got nothing:
// `ralph org read` for its pane, `ralph org status --live` for herdr's state
// next to the screen's, and `herdr agent explain` for the rule herdr took
// its state from. The ralph commands repeat p.HintStateDir.
func sendScreenNextSteps(p SendParams, name string) string {
	read := VerbCommand("read --org-id "+ShellWord(p.OrgID)+" --seat "+ShellWord(p.To), p.HintStateDir)
	status := VerbCommand("status --org-id "+ShellWord(p.OrgID)+" --live", p.HintStateDir)
	return fmt.Sprintf("read the pane with `%s`, compare herdr's state with the screen with `%s`, and see the rule behind herdr's state with `herdr agent explain %s`",
		read, status, ShellWord(name))
}

// sendScreenText is s for an error: its kind, then its reason and, when it
// has one, its line, quoted and cut to sendScreenLineMaxRunes runes, in
// parentheses: `busy (a busy line is on the screen: "✶ Deliberating…")`.
func sendScreenText(s SendScreen) string {
	text := string(s.Kind) + " (" + s.Reason
	if s.Line != "" {
		line := []rune(s.Line)
		if len(line) > sendScreenLineMaxRunes {
			line = append(line[:sendScreenLineMaxRunes], '…')
		}
		text += ": " + strconv.Quote(string(line))
	}
	return text + ")"
}
