package org

import (
	"context"
	"fmt"
	"time"
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
}

// SendResult is Send's return value.
type SendResult struct {
	Err error
}

// Send waits for the target seat to go idle, then types Text into its pane
// and presses Enter. A non-state `sent` event is appended for history.
// DryRun skips the seat lookup and driver calls entirely and only appends
// the (dry_run: true) history event.
func (o *Org) Send(p SendParams) SendResult {
	details := truncateForDetails(p.Text)

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

// Stop sends a best-effort C-c to Seat's pane (real invocations only), then
// appends a `stopped` state event recording the method used and any error.
// DryRun appends the event without attempting a real stop signal.
func (o *Org) Stop(p StopParams) StopResult {
	var paneID, details string

	if p.DryRun {
		details = "dry-run: no driver call"
	} else {
		seat, ok, err := o.findSeat(p.OrgID, p.Seat)
		if err != nil {
			return StopResult{Err: fmt.Errorf("org: stop: read manifest: %w", err)}
		}
		if ok {
			paneID = seat.PaneID
		}
		if paneID == "" {
			details = "method=C-c no pane_id on record"
		} else if err := o.Herdr.PaneSendKeys(context.Background(), paneID, "C-c"); err != nil {
			details = fmt.Sprintf("method=C-c error=%v", err)
		} else {
			details = "method=C-c"
		}
	}

	err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.Seat, Event: EventStopped,
		PaneID: paneID, DryRun: p.DryRun, Details: details,
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

// Disband best-effort-stops every currently active seat in OrgID, then
// appends an org-level `disbanded` event (SeatID empty) that marks every
// seat in that org_id inactive from that point forward (see Roster). DryRun
// skips stopping real seats and only appends the disbanded event.
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
