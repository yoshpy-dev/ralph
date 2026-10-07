package org

import (
	"fmt"
	"slices"
)

// OrgSeat names one seat across org_ids, for StopAll and DisbandAll.
type OrgSeat struct {
	OrgID  string
	SeatID string
}

// OrgSeatFailure is a SeatFailure from StopAll or DisbandAll, with the
// org_id the seat belongs to. Forced means the same as in SeatFailure.
type OrgSeatFailure struct {
	OrgID string
	SeatFailure
}

// OrgWorkspace names one recorded herdr workspace across org_ids.
type OrgWorkspace struct {
	OrgID       string
	WorkspaceID string
}

// OrgWorkspaceFailure is a WorkspaceFailure from DisbandAll, with the org_id
// the workspace belongs to. Forced means the same as in WorkspaceFailure.
type OrgWorkspaceFailure struct {
	OrgID string
	WorkspaceFailure
}

// allOwnLastWhy is the reason StopAll and DisbandAll give for leaving the
// caller's own seat or org untouched: closing it would end the caller before
// it could report the failure (see Disband's doc comment).
const allOwnLastWhy = "the caller's own pane and workspace are handled last, after every other seat and org, and one failed"

// StopAllParams describes one `ralph org stop --all` invocation.
type StopAllParams struct {
	DryRun bool
	// Force is passed to every seat's Stop (StopParams.Force).
	Force bool
}

// StopAllResult is StopAll's return value. After a manifest read that
// succeeded, each seat that was active is in exactly one of StoppedSeats and
// FailedSeats.
type StopAllResult struct {
	// StoppedSeats are the seats Stop recorded `stopped` without a close
	// failure, in the order StopAll stopped them.
	StoppedSeats []OrgSeat
	// FailedSeats are the seats not stopped cleanly, with the reason. A
	// Forced entry was recorded `stopped` anyway (a warning); any other
	// entry is still active, so the next StopAll stops it again.
	FailedSeats []OrgSeatFailure
	// DeferredSelfPaneID is the caller's own pane when StopAll recorded the
	// seat in it `stopped` but left the pane open (StopResult's field of the
	// same name). The caller must finish all output and then close it with
	// CloseDeferredSelfPane as its very last action.
	DeferredSelfPaneID string
	// ModelReceipts are the receipts the seats' Stops appended
	// (StopResult.ModelReceipt), each naming its org_id and seat_id, so the
	// CLI can print the codex model-mismatch warning per seat.
	ModelReceipts []Receipt
	// Errs is every error that makes this stop fail: a manifest read failure
	// (then nothing was stopped), or the Err of each FailedSeats entry whose
	// Forced is false, prefixed with `<org_id>/<seat_id>: `. Empty when every
	// active seat was recorded `stopped`.
	Errs []error
}

func (r *StopAllResult) recordStop(seat OrgSeat, res StopResult) {
	switch {
	case res.Err != nil:
		r.failSeat(seat, res.Err)
	case res.CloseErr != nil:
		r.FailedSeats = append(r.FailedSeats, OrgSeatFailure{OrgID: seat.OrgID, SeatFailure: SeatFailure{SeatID: seat.SeatID, Err: res.CloseErr, Forced: true}})
	default:
		r.StoppedSeats = append(r.StoppedSeats, seat)
	}
	if res.DeferredSelfPaneID != "" {
		r.DeferredSelfPaneID = res.DeferredSelfPaneID
	}
	if res.ModelReceipt != (Receipt{}) {
		r.ModelReceipts = append(r.ModelReceipts, res.ModelReceipt)
	}
}

func (r *StopAllResult) failSeat(seat OrgSeat, err error) {
	r.FailedSeats = append(r.FailedSeats, OrgSeatFailure{OrgID: seat.OrgID, SeatFailure: SeatFailure{SeatID: seat.SeatID, Err: err}})
	r.Errs = append(r.Errs, fmt.Errorf("%s/%s: %w", seat.OrgID, seat.SeatID, err))
}

// StopAll stops every active seat of every org_id with Stop, passing DryRun
// and Force through, in org_id then seat_id order (Roster's order). The
// seats come from one manifest read, real seats only (Roster without
// dry-run events); each Stop reads the manifest again. A seat Stop could not
// stop does not stop the rest: it stays active (stop_failed) and is in
// FailedSeats, and the next StopAll picks it up again. Every driver call
// Stop makes has its own driverCallTimeout deadline, so a herdr or agmsg
// that never answers costs one seat a few deadlines, not the whole run.
//
// A real run stops the seat in the caller's own pane (HERDR_PANE_ID, read
// via o.Getenv) last, and only when every other seat was recorded `stopped`
// (forced ones included): Stop then records it without closing the pane,
// and the pane comes back in DeferredSelfPaneID. When another seat failed,
// the own seat is left running and listed in FailedSeats, so a caller that
// ran stop --all in its own pane stays alive to see the failures and retry
// (the same rule Disband uses).
//
// A manifest read failure stops nothing and is the only error.
func (o *Org) StopAll(p StopAllParams) StopAllResult {
	var result StopAllResult
	rr, err := o.Manifest.Read()
	if err != nil {
		result.Errs = append(result.Errs, fmt.Errorf("org: stop all: read manifest: %w", err))
		return result
	}

	var ownPane string
	if !p.DryRun {
		ownPane = o.getenv(herdrPaneIDEnv)
	}
	var ownSeats []OrgSeat
	for _, s := range Roster(rr.Events, RosterOptions{}) {
		if !s.Active {
			continue
		}
		seat := OrgSeat{OrgID: s.OrgID, SeatID: s.SeatID}
		if ownPane != "" && s.PaneID == ownPane {
			ownSeats = append(ownSeats, seat)
			continue
		}
		result.recordStop(seat, o.Stop(StopParams{OrgID: s.OrgID, Seat: s.SeatID, DryRun: p.DryRun, Force: p.Force}))
	}
	for _, seat := range ownSeats {
		if len(result.Errs) > 0 {
			result.failSeat(seat, fmt.Errorf("org: stop all: seat %q in org_id %q left running in the caller's own pane: %s", seat.SeatID, seat.OrgID, allOwnLastWhy))
			continue
		}
		result.recordStop(seat, o.Stop(StopParams{OrgID: seat.OrgID, Seat: seat.SeatID, Force: p.Force}))
	}
	return result
}

// DisbandAllParams describes one `ralph org disband --all` invocation.
type DisbandAllParams struct {
	DryRun bool
	// Force is passed to every org's Disband (DisbandParams.Force).
	Force bool
}

// DisbandAllResult is DisbandAll's return value: the DisbandResult of every
// org it targeted, merged, each entry tagged with its org_id.
type DisbandAllResult struct {
	// Orgs are the org_ids DisbandAll targeted, in the order it handled
	// them, and DisbandedOrgs those that got `disbanded`. An org in Orgs but
	// not in DisbandedOrgs failed, and the next DisbandAll targets it again.
	Orgs          []string
	DisbandedOrgs []string
	// StoppedSeats, FailedSeats, ClosedWorkspaces and FailedWorkspaces are
	// the per-org DisbandResult fields of the same names, with Forced
	// meaning the same.
	StoppedSeats     []OrgSeat
	FailedSeats      []OrgSeatFailure
	ClosedWorkspaces []OrgWorkspace
	FailedWorkspaces []OrgWorkspaceFailure
	// DeferredSelfPaneID and DeferredSelfWorkspaceID are the caller's own
	// pane and workspace that one org's Disband recorded closed but left
	// open (DisbandResult's fields of the same names). The caller must
	// finish all output and then close them as its very last action. The
	// own pane lies in the own workspace, so when DeferredSelfWorkspaceID is
	// set, DeferredSelfPaneID is empty, even when the two came from
	// different orgs.
	DeferredSelfPaneID      string
	DeferredSelfWorkspaceID string
	// Errs is every error that makes this disband fail: a manifest read
	// failure (then nothing was done), every per-org Disband error prefixed
	// with `org_id "<org_id>": `, and one error per org left untouched
	// because it holds the caller's own pane or workspace (see DisbandAll).
	// After a manifest read that succeeded, it is empty exactly when every
	// org in Orgs is in DisbandedOrgs.
	Errs []error
}

func (r *DisbandAllResult) add(orgID string, d DisbandResult) {
	r.Orgs = append(r.Orgs, orgID)
	if d.Disbanded {
		r.DisbandedOrgs = append(r.DisbandedOrgs, orgID)
	}
	for _, seatID := range d.StoppedSeats {
		r.StoppedSeats = append(r.StoppedSeats, OrgSeat{OrgID: orgID, SeatID: seatID})
	}
	for _, f := range d.FailedSeats {
		r.FailedSeats = append(r.FailedSeats, OrgSeatFailure{OrgID: orgID, SeatFailure: f})
	}
	for _, ws := range d.ClosedWorkspaces {
		r.ClosedWorkspaces = append(r.ClosedWorkspaces, OrgWorkspace{OrgID: orgID, WorkspaceID: ws})
	}
	for _, f := range d.FailedWorkspaces {
		r.FailedWorkspaces = append(r.FailedWorkspaces, OrgWorkspaceFailure{OrgID: orgID, WorkspaceFailure: f})
	}
	if d.DeferredSelfPaneID != "" {
		r.DeferredSelfPaneID = d.DeferredSelfPaneID
	}
	if d.DeferredSelfWorkspaceID != "" {
		r.DeferredSelfWorkspaceID = d.DeferredSelfWorkspaceID
	}
	if r.DeferredSelfWorkspaceID != "" {
		r.DeferredSelfPaneID = ""
	}
	for _, err := range d.Errs {
		r.Errs = append(r.Errs, fmt.Errorf("org_id %q: %w", orgID, err))
	}
}

// leaveOwnOrg records orgID, which holds the caller's own pane or workspace,
// as not disbanded without touching it: every active seat and open
// workspace it has is a failure with the same reason.
func (r *DisbandAllResult) leaveOwnOrg(events []ManifestEvent, roster []SeatStatus, orgID string) {
	err := fmt.Errorf("org: disband all: org_id %q left as is: it holds the caller's own pane or workspace, and %s", orgID, allOwnLastWhy)
	r.Orgs = append(r.Orgs, orgID)
	for _, s := range roster {
		if s.OrgID == orgID && s.Active {
			r.FailedSeats = append(r.FailedSeats, OrgSeatFailure{OrgID: orgID, SeatFailure: SeatFailure{SeatID: s.SeatID, Err: err}})
		}
	}
	for _, ws := range openOrgWorkspaces(events, orgID) {
		r.FailedWorkspaces = append(r.FailedWorkspaces, OrgWorkspaceFailure{OrgID: orgID, WorkspaceFailure: WorkspaceFailure{WorkspaceID: ws, Err: err}})
	}
	r.Errs = append(r.Errs, err)
}

// DisbandAll disbands, with Disband, every org_id that still needs
// disbanding (orgsToDisband), passing DryRun and Force through, in org_id
// order. The targets come from one manifest read; each Disband reads the
// manifest again. An org whose Disband fails does not stop the rest: it gets
// no `disbanded`, its failures are in the result, and the next DisbandAll
// targets it again. Every driver call has its own driverCallTimeout
// deadline, so a herdr or agmsg that never answers for one org costs that
// org a few deadlines, not the whole run.
//
// A real run disbands the orgs holding the caller's own pane (an active
// seat whose pane_id is HERDR_PANE_ID) or own workspace (an open workspace
// that is HERDR_WORKSPACE_ID) last, and only when every other org was
// disbanded: their Disband records the own pane and workspace without
// closing them, and they come back in DeferredSelfPaneID /
// DeferredSelfWorkspaceID. When another org failed, such an org is left
// untouched and listed as failed (its active seats and open workspaces in
// FailedSeats / FailedWorkspaces), so a caller that ran disband --all in its
// own pane stays alive to see the failures and retry (the same rule Disband
// uses within one org).
//
// A dry run targets the orgs a real run would, makes no driver call, and
// appends one dry-run `disbanded` per org (Disband's DryRun).
//
// A manifest read failure does nothing and is the only error.
func (o *Org) DisbandAll(p DisbandAllParams) DisbandAllResult {
	var result DisbandAllResult
	rr, err := o.Manifest.Read()
	if err != nil {
		result.Errs = append(result.Errs, fmt.Errorf("org: disband all: read manifest: %w", err))
		return result
	}

	roster := Roster(rr.Events, RosterOptions{})
	orgs := orgsToDisband(rr.Events)
	var ownOrgs []string
	if !p.DryRun {
		orgs, ownOrgs = splitOwnOrgs(rr.Events, roster, orgs, o.getenv(herdrPaneIDEnv), o.getenv(herdrWorkspaceIDEnv))
	}
	for _, orgID := range orgs {
		result.add(orgID, o.Disband(DisbandParams{OrgID: orgID, DryRun: p.DryRun, Force: p.Force}))
	}
	for _, orgID := range ownOrgs {
		if len(result.Errs) > 0 {
			result.leaveOwnOrg(rr.Events, roster, orgID)
			continue
		}
		result.add(orgID, o.Disband(DisbandParams{OrgID: orgID, Force: p.Force}))
	}
	return result
}

// orgsToDisband returns, sorted, the org_ids DisbandAll targets. Only real
// (non-dry-run) events count. An org needs disbanding when either holds:
//   - it has a seat state event (isStateEvent: spawn_started through
//     stopped) or an org_workspace_created / org_workspace_closed after its
//     latest org-level `disbanded`, or it has such an event and no
//     `disbanded` at all. This covers every org with an active seat: a
//     seat is active only while its latest state event comes after the
//     org's latest `disbanded` (see Roster);
//   - the manifest still records one of its workspaces open
//     (openOrgWorkspaces), including one that an older ralph's disband,
//     which did not close workspaces, left open.
//
// So an org whose latest such record is its `disbanded`, with no workspace
// open, is skipped: disbanding it again would only repeat the record.
// Non-state seat events (`sent`, `stop_failed`) do not make an org a target
// by themselves; a seat left active by stop_failed already does.
func orgsToDisband(events []ManifestEvent) []string {
	lastDisbanded := make(map[string]int)
	lastRecord := make(map[string]int)
	for i, ev := range events {
		if ev.DryRun || ev.OrgID == "" {
			continue
		}
		switch {
		case ev.SeatID == "" && ev.Event == EventDisbanded:
			lastDisbanded[ev.OrgID] = i
		case ev.SeatID == "" && (ev.Event == EventOrgWorkspaceCreated || ev.Event == EventOrgWorkspaceClosed),
			ev.SeatID != "" && isStateEvent(ev.Event):
			lastRecord[ev.OrgID] = i
		}
	}

	var orgs []string
	for orgID, i := range lastRecord {
		if d, ok := lastDisbanded[orgID]; !ok || i > d || len(openOrgWorkspaces(events, orgID)) > 0 {
			orgs = append(orgs, orgID)
		}
	}
	slices.Sort(orgs)
	return orgs
}

// splitOwnOrgs splits orgs, keeping their order, into those that hold the
// caller's own pane (an active seat whose pane_id is ownPane) or own
// workspace (an open workspace that is ownWorkspace) and the others.
// DisbandAll disbands the own ones last. An empty ownPane / ownWorkspace
// matches nothing.
func splitOwnOrgs(events []ManifestEvent, roster []SeatStatus, orgs []string, ownPane, ownWorkspace string) (others, own []string) {
	for _, orgID := range orgs {
		if holdsCaller(events, roster, orgID, ownPane, ownWorkspace) {
			own = append(own, orgID)
		} else {
			others = append(others, orgID)
		}
	}
	return others, own
}

func holdsCaller(events []ManifestEvent, roster []SeatStatus, orgID, ownPane, ownWorkspace string) bool {
	if ownPane != "" {
		for _, s := range roster {
			if s.OrgID == orgID && s.Active && s.PaneID == ownPane {
				return true
			}
		}
	}
	return ownWorkspace != "" && slices.Contains(openOrgWorkspaces(events, orgID), ownWorkspace)
}
