package org

import (
	"context"
	"fmt"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// EventSent is a non-state event (see seat.go stateEvents) recorded by the
// send verb. It never affects seat activity derivation -- it is a pure
// history/audit entry.
const EventSent = "sent"

const (
	defaultSendTimeoutMS = 30000
	defaultReadLines     = 50
	sendDetailsMaxLen    = 200
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

// SendParams describes one `ralph org send` invocation.
type SendParams struct {
	OrgID     string
	To        string
	Text      string
	TimeoutMS int
	DryRun    bool
	// Raw bypasses AC-11 typed-protocol validation entirely, for the rare
	// case that genuinely needs to send free-form text. See
	// .claude/rules/agent-messaging.md's "Size cap" section for the
	// intended use (e.g. relaying an external tool's raw output). A
	// bypassed send still records raw=true on the `sent` event's Details,
	// so it stays traceable after the fact.
	Raw bool
}

// SendResult is Send's return value.
type SendResult struct {
	Err error
}

// Send validates Text against the typed message protocol (unless Raw),
// waits for the target seat to go idle, then types Text into its pane and
// presses Enter. A non-state `sent` event is appended for history. DryRun
// skips the seat lookup and driver calls entirely and only appends the
// (dry_run: true) history event -- but protocol validation still runs
// first for DryRun too, since it is a pure, side-effect-free check.
//
// AC-11: an invalid message (unless Raw) is rejected before any manifest
// event is appended and before any driver call is attempted -- Send fails
// closed, not open.
func (o *Org) Send(p SendParams) SendResult {
	if !p.Raw {
		if err := protocol.ValidateText(p.Text, protocol.DefaultMaxBodyChars); err != nil {
			return SendResult{Err: fmt.Errorf("org: send: message rejected by protocol validation (use --raw to bypass): %w", err)}
		}
	}

	details := truncateForDetails(p.Text)
	if p.Raw {
		details = "raw=true " + details
	}

	if p.DryRun {
		err := o.appendEvent(ManifestEvent{
			TS: o.now(), OrgID: p.OrgID, SeatID: p.To, Event: EventSent,
			DryRun: true, Details: details,
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

	if _, err := o.Herdr.AgentWait(ctx, herdrAgentName(p.OrgID, p.To), []string{"idle"}, timeoutMS); err != nil {
		return SendResult{Err: fmt.Errorf("org: send: wait for seat %q idle: %w", p.To, err)}
	}
	if err := o.Herdr.PaneSendText(ctx, seat.PaneID, p.Text); err != nil {
		return SendResult{Err: fmt.Errorf("org: send: send text to seat %q: %w", p.To, err)}
	}
	if err := o.Herdr.PaneSendKeys(ctx, seat.PaneID, "Enter"); err != nil {
		return SendResult{Err: fmt.Errorf("org: send: send Enter to seat %q: %w", p.To, err)}
	}

	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.To, Event: EventSent,
		PaneID: seat.PaneID, Details: details,
	}); err != nil {
		return SendResult{Err: err}
	}
	return SendResult{}
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
// passthrough to Herdr.AgentWait -- it never touches the manifest. OrgID is
// required: it namespaces the herdr agent name (see herdrAgentName) so wait
// targets the seat spawned within this org_id, not any same-named seat in a
// different org_id.
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

	err = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.Seat, Event: EventStopped,
		Role: seat.Role, Driver: seat.Driver, Model: seat.Model, Worktree: seat.Worktree,
		PaneID: paneID, AgmsgTeam: team, DryRun: p.DryRun, Details: details,
	})
	return StopResult{Err: err}
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
