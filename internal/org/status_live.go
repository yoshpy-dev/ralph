package org

import (
	"context"
	"strings"
)

// `ralph org status --live` (plan
// docs/plans/active/2026-11-12-send-screen-fallback.md, D6 and AC6; issue
// #308): next to the roster Status derives from the manifest alone,
// LiveSeatStates asks herdr for each seat's agent_status and reads its pane,
// so a seat herdr reports working while its screen waits for input shows up.
// Status itself stays manifest-only (AC-4); only --live calls this. It
// starts no herdr server: with herdr not running, every call fails and each
// seat says so.

const (
	// LiveScreenUnreadable is LiveSeatState.Screen when `herdr pane read`
	// failed, so the screen could not be classified.
	LiveScreenUnreadable = "unreadable"
	// liveScreenLineMaxRunes caps the screen line LiveSeatState.ScreenText
	// quotes. It is shorter than the cap of send's errors
	// (sendScreenLineMaxRunes), since the text goes into a table row.
	liveScreenLineMaxRunes = 60
	// liveErrMaxRunes caps the herdr failure LiveSeatState keeps in HerdrErr
	// and, for an unreadable screen, in ScreenReason.
	liveErrMaxRunes = 100
)

// LiveSeatState is what LiveSeatStates found for one seat.
type LiveSeatState struct {
	// Read is false for a seat LiveSeatStates did not look at: one that is
	// not active, a dry run, or has no pane. Its other fields are empty.
	Read bool
	// Agent is the seat's herdr agent name (resolvedHerdrAgentName), the
	// name `herdr agent explain` takes.
	Agent string
	// HerdrState is the agent_status `herdr agent get` reported (idle,
	// working, blocked, done, or herdr's own unknown), or "unknown" when the
	// call failed or its reply had no status; HerdrErr then says why, in one
	// line of at most liveErrMaxRunes runes.
	HerdrState string
	HerdrErr   string
	// Screen is the kind ClassifySendScreen gave the pane for the seat's
	// driver (idle, busy, choice, unknown), or LiveScreenUnreadable when the
	// pane read failed. ScreenReason is the classifier's reason, or for an
	// unreadable screen the failure (cut like HerdrErr), and ScreenLine the
	// line that decided the kind, when there is one.
	Screen       string
	ScreenReason string
	ScreenLine   string
	// Mismatch is true when herdr reports the seat working while the screen
	// is idle, waiting for input: the seat of issue #308, which `ralph org
	// send` types into only after its screen check.
	Mismatch bool
}

// LiveSeatStates looks at each seat of seats (Status's roster) that is
// active, not a dry run, and has a pane, one after another, and returns one
// LiveSeatState per seat, in the order of seats. For each it runs `herdr
// agent get` for the seat's agent and `herdr pane read` of its pane
// (resumeReadLines lines, as send's screen check reads it), each with
// driverCallTimeout as its deadline, and classifies the screen with
// ClassifySendScreen. The pane of a seat whose driver has no screen rules
// (neither claude nor codex, or none recorded) is not read: its screen is
// unknown, for an unknown driver. It changes nothing: no manifest event, no
// pane lock, no key typed, and no herdr server started.
func (o *Org) LiveSeatStates(seats []SeatStatus) []LiveSeatState {
	states := make([]LiveSeatState, len(seats))
	for i, s := range seats {
		if !s.Active || s.DryRun || s.PaneID == "" {
			continue
		}
		states[i] = o.liveSeatState(s)
	}
	return states
}

// liveSeatState is LiveSeatStates for one seat it looks at.
func (o *Org) liveSeatState(s SeatStatus) LiveSeatState {
	st := LiveSeatState{Read: true, Agent: resolvedHerdrAgentName(s)}
	st.HerdrState, st.HerdrErr = o.liveHerdrState(st.Agent)
	st.Screen, st.ScreenReason, st.ScreenLine = o.liveScreen(s)
	st.Mismatch = st.HerdrState == "working" && st.Screen == string(SendScreenIdle)
	return st
}

// liveHerdrState is the agent_status of name (resumeAgentStatus, with
// driverCallTimeout as its deadline), or sendHerdrStateUnknown and why when
// the herdr client cannot run `herdr agent get`, the call fails, or the
// reply has no status.
func (o *Org) liveHerdrState(name string) (state, errText string) {
	probe, ok := o.Herdr.(watchHerdrProbe)
	if !ok {
		return sendHerdrStateUnknown, resumeReasonNoAgentGet
	}
	status, err := o.resumeAgentStatus(probe, name)
	if err != nil {
		return sendHerdrStateUnknown, liveShortText(err.Error())
	}
	return status, ""
}

// liveScreen reads s's pane, with driverCallTimeout as its deadline, and
// returns ClassifySendScreen's kind, reason, and line for s's driver, or
// LiveScreenUnreadable and the failure when the read fails. For a driver
// with no screen rules it reads nothing.
func (o *Org) liveScreen(s SeatStatus) (kind, reason, line string) {
	if _, ok := resumeScreenSentences[s.Driver]; !ok {
		return string(SendScreenUnknown), sendScreenReasonUnknownDriver, ""
	}
	var text string
	err := o.callWithTimeout(func(ctx context.Context) error {
		var err error
		text, err = o.Herdr.PaneRead(ctx, s.PaneID, resumeReadLines)
		return err
	})
	if err != nil {
		return LiveScreenUnreadable, liveShortText("herdr pane read " + s.PaneID + " failed: " + err.Error()), ""
	}
	screen := ClassifySendScreen(s.Driver, text)
	return string(screen.Kind), screen.Reason, screen.Line
}

// liveShortText is text on one line (each run of white space, newlines
// included, made one space) and cut to liveErrMaxRunes runes, with "…"
// where it was cut.
func liveShortText(text string) string {
	runes := []rune(strings.Join(strings.Fields(text), " "))
	if len(runes) > liveErrMaxRunes {
		runes = append(runes[:liveErrMaxRunes], '…')
	}
	return string(runes)
}

// HerdrText is the herdr column of `ralph org status --live`: HerdrState,
// followed by HerdrErr in parentheses when there is one.
func (s LiveSeatState) HerdrText() string {
	if s.HerdrErr == "" {
		return s.HerdrState
	}
	return s.HerdrState + " (" + s.HerdrErr + ")"
}

// ScreenText is the screen column of `ralph org status --live`, written as
// send's errors write a screen (sendScreenText): Screen, then in
// parentheses its reason and, when there is one, its line, quoted and cut
// to liveScreenLineMaxRunes runes: `idle (the turn ended: "✻ Cooked for 3s
// · done 6:09 PM")`.
func (s LiveSeatState) ScreenText() string {
	line := []rune(s.ScreenLine)
	if len(line) > liveScreenLineMaxRunes {
		line = append(line[:liveScreenLineMaxRunes], '…')
	}
	return sendScreenText(SendScreen{Kind: SendScreenKind(s.Screen), Reason: s.ScreenReason, Line: string(line)})
}
