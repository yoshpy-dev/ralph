package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// `ralph org resume` (plan 2026-11-07-org-seat-resume, S5; spec
// docs/specs/2026-11-07-org-seat-resume.md, FR-1 to FR-7 and FR-5a). Resume
// finds the seats, and the headless directors in herdr, whose CLI ended its
// turn on an API error (a network failure or a usage limit), and types the
// resume sentence into each, or presses Enter where the CLI asks for it after
// a usage limit reset. It decides with herdr's agent status and the screen
// (ClassifyResumeScreen, resume_screens.go) only, never with a model, and
// types nothing into a screen it does not know.

// ResumeText is the sentence Resume types into a stopped seat's input box
// (spec FR-5). It names where it comes from, since a seat's role prompt
// treats plain text from anywhere but its leader as data; the role prompts
// tell the seats to take this sentence as an instruction to go on (FR-10).
const ResumeText = "`ralph org resume` からの続行の指示です。直前の API エラー(通信断か利用上限)で作業が止まっていました。止まったところから続けてください。"

// The kinds of target Resume reports (ResumeTargetResult.Kind).
const (
	// ResumeTargetSeat is an org seat of the manifest.
	ResumeTargetSeat = "seat"
	// ResumeTargetDirector is an open headless director registration of
	// director.jsonl; its ID is the registration's d<N>.
	ResumeTargetDirector = "director"
)

// ResumeOutcome is what Resume did with one target.
type ResumeOutcome string

const (
	// ResumeOutcomeResumed: the screen showed a stop, Resume typed
	// ResumeText and pressed Enter once.
	ResumeOutcomeResumed ResumeOutcome = "resumed"
	// ResumeOutcomeEntered: the screen asked for Enter after a usage limit
	// reset, and Resume pressed Enter once and typed nothing.
	ResumeOutcomeEntered ResumeOutcome = "entered"
	// ResumeOutcomeWaiting: herdr reports the target working, or its screen
	// shows the CLI retrying or waiting on its own. Nothing was typed.
	ResumeOutcomeWaiting ResumeOutcome = "waiting"
	// ResumeOutcomeSkipped: anything else that makes Resume type nothing
	// (Reason says what).
	ResumeOutcomeSkipped ResumeOutcome = "skipped"
	// ResumeOutcomeFailed: a herdr call or the check before Enter failed
	// after Resume took the pane's lock, or the `resumed` event could not be
	// recorded. Reason says what is known of what reached the pane.
	ResumeOutcomeFailed ResumeOutcome = "failed"
	// ResumeOutcomeWouldResume and ResumeOutcomeWouldEnter are DryRun's
	// ResumeOutcomeResumed and ResumeOutcomeEntered: nothing was typed.
	ResumeOutcomeWouldResume ResumeOutcome = "would_resume"
	ResumeOutcomeWouldEnter  ResumeOutcome = "would_enter"
)

// The reasons Resume gives (ResumeTargetResult.Reason) for a target it
// types nothing into without a herdr error to quote.
const (
	resumeReasonOwnPane       = "the caller's own pane"
	resumeReasonNotInHerdr    = "not in herdr"
	resumeReasonNoPane        = "no pane_id recorded"
	resumeReasonNoAgentGet    = "the herdr client cannot run herdr agent get"
	resumeReasonHerdrWorking  = "herdr reports working"
	resumeReasonHerdrBlocked  = "herdr reports blocked"
	resumeReasonPaneBusy      = "another send or resume to this pane is in progress"
	resumeReasonScreenChanged = "the screen changed"
)

// resumeReadLines is how many lines of a pane Resume reads (plan D5): the
// 30 lines of the evidence's screen (docs/evidence/
// org-seat-resume-probe-2026-11-07.md, section 2) with room for the waiting
// line and a long error sentence wrapped over several lines.
const resumeReadLines = 60

// defaultResumePaneLockWait is how long Resume waits for a pane's lock when
// ResumeParams.LockWait is unset. A Send holds it for its pause before Enter
// and its submit confirmation (seconds); a lock held longer than this is a
// send or resume that is stuck, and Resume skips the target.
const defaultResumePaneLockWait = 30 * time.Second

// errResumeTextNotInInput is the check before Enter refusing: the pane read
// after the typing shows no input box holding the start of ResumeText.
var errResumeTextNotInInput = errors.New("typed text not found in the input box; Enter not pressed")

// ResumeParams describes one `ralph org resume` invocation.
type ResumeParams struct {
	// OrgID, when set, limits Resume to that org's seats, and no director
	// is a target. Empty means the seats of every org and the ledger's
	// open headless directors.
	OrgID string
	// DryRun looks at every target as a real run does (herdr's agent
	// status and the screen) and reports ResumeOutcomeWouldResume or
	// ResumeOutcomeWouldEnter where a real run would type, but takes no
	// lock, types nothing, and records nothing.
	DryRun bool
	// LockWait is how long Resume waits for each pane's lock. Zero means
	// defaultResumePaneLockWait; tests set a short one.
	LockWait time.Duration
}

// ResumeTargetResult is what Resume did with one target.
type ResumeTargetResult struct {
	// Kind is ResumeTargetSeat or ResumeTargetDirector.
	Kind string
	// OrgID is the seat's org_id; empty for a director.
	OrgID string
	// ID is the seat id, or the director registration's d<N>.
	ID string
	// Role is the seat's role (the latest state event that records one),
	// or DirectorRole for a director.
	Role string
	// Driver is the seat's or the director's driver (claude or codex).
	Driver string
	// PaneID is the herdr pane Resume looked at (empty for a director with
	// no pane recorded).
	PaneID  string
	Outcome ResumeOutcome
	// Reason says why for ResumeOutcomeWaiting, ResumeOutcomeSkipped, and
	// ResumeOutcomeFailed, and for a ResumeOutcomeResumed or
	// ResumeOutcomeEntered seat whose `resumed` event could not be recorded.
	// It is empty otherwise.
	Reason string
	// Line is the screen line ClassifyResumeScreen matched (its wrapped
	// lines joined), printed as it is: a usage limit's reset time in it is
	// not read (FR-7). Empty when Resume did not get as far as the screen,
	// or the screen matched nothing.
	Line string
	// SubmitConfirmed reports, for ResumeOutcomeResumed and
	// ResumeOutcomeEntered, whether herdr reported the target working or
	// blocked after Enter (confirmSubmitted; the same proxy for "submitted"
	// as Send's). false asks the caller to check the pane.
	SubmitConfirmed bool
	// Recorded reports whether Resume appended a `resumed` event for the
	// target. A director is not a seat of the manifest, so it is always
	// false for one.
	Recorded bool
}

// ResumeResult is Resume's return value.
type ResumeResult struct {
	// Targets are the targets in the order Resume handled them (FR-4):
	// every seat that is not a leader, then the leaders, then the
	// directors, each group in Roster's order (org_id, then seat_id) and the
	// directors by their d<N>.
	Targets []ResumeTargetResult
	// Errs is every error that makes the resume fail (FR-7): a manifest read
	// failure (then nothing was looked at), or the error of each
	// ResumeOutcomeFailed target, prefixed with the target
	// (`<org_id>/<seat_id>: ` or `director <id>: `). A seat whose `resumed`
	// event could not be recorded is in it too. Empty when every target was
	// typed into, waiting, or skipped.
	Errs []error
	// DirectorReadErr is why director.jsonl could not be read when OrgID is
	// empty. The seats are resumed all the same, and it is not in Errs.
	DirectorReadErr error
}

// resumeTarget is one target Resume looks at.
type resumeTarget struct {
	kind, orgID, id, role, driver, paneID string
	// agentName is the herdr agent name of the target's agent.
	agentName string
	// skip, when set, is why the target is listed without a look at herdr:
	// a director with launch cmd (not in herdr), or one in herdr with no
	// pane recorded yet.
	skip string
}

// label names the target in an error of ResumeResult.Errs.
func (t resumeTarget) label() string {
	if t.kind == ResumeTargetDirector {
		return "director " + t.id
	}
	return t.orgID + "/" + t.id
}

// Resume looks at every active seat with a pane (OrgID's only, when set)
// and, when OrgID is empty, every open headless director, and types into
// each whose CLI stopped on an API error (spec FR-1 to FR-7). The seats come
// from one manifest read (Roster, real seats only); the directors from one
// read of the director.jsonl next to the manifest, whose failure is only
// recorded in DirectorReadErr.
//
// For each target, in the order of FR-4 (ResumeResult.Targets), and with
// each failure confined to that target:
//
//  1. A director with launch cmd, or with no pane recorded, is skipped. A
//     target in the caller's own pane (HERDR_PANE_ID, read via o.Getenv) is
//     skipped.
//  2. `herdr agent get` must report the agent idle or done: working is
//     waiting, blocked or any other status is skipped, and so is a failed
//     call or a reply with no status.
//  3. `herdr pane read` of resumeReadLines lines, classified with
//     ClassifyResumeScreen for the target's driver: waiting is waiting,
//     unknown is skipped, and stopped and enter_only go on. DryRun stops
//     here with ResumeOutcomeWouldResume or ResumeOutcomeWouldEnter.
//  4. Resume takes the pane's lock (pane_lock.go), the one Send takes,
//     waiting up to LockWait; a lock still held then is skipped. Holding
//     it, it runs 2 and 3 again, and a target that is no longer idle or
//     done, or whose screen is no longer of the kind 3 found, is skipped as
//     changed, with nothing typed.
//  5. A stopped screen gets ResumeText, the pause Send waits before Enter
//     (sendEnterDelay), and then a pane read whose input box must hold the
//     start of ResumeText (ResumeInputHolds); only then Enter, once. An
//     enter_only screen gets Enter once and no text. Then confirmSubmitted,
//     as Send does, and the lock is released. typeAndSubmit, which Send
//     uses too, runs these steps on one deadline (resumeSubmitBudget).
//  6. A seat gets a `resumed` event (EventResumed) once Enter was pressed;
//     a director gets none, as it is not a seat of the manifest.
//
// A failed herdr call in 5, the check before Enter refusing, and a failed
// `resumed` append make the target ResumeOutcomeFailed and add an error to
// Errs; Resume goes on with the next target. A herdr call in 2 and 3 runs
// with driverCallTimeout as its deadline.
func (o *Org) Resume(p ResumeParams) ResumeResult {
	var result ResumeResult
	rr, err := o.Manifest.Read()
	if err != nil {
		result.Errs = append(result.Errs, fmt.Errorf("org: resume: read manifest: %w", err))
		return result
	}
	targets := resumeSeatTargets(rr.Events, p.OrgID)
	if p.OrgID == "" {
		regs, err := o.openHeadlessDirectors()
		result.DirectorReadErr = err
		targets = append(targets, resumeDirectorTargets(regs)...)
	}
	ownPane := o.getenv(herdrPaneIDEnv)
	for _, t := range targets {
		res, err := o.resumeOne(t, p, ownPane)
		result.Targets = append(result.Targets, res)
		if err != nil {
			result.Errs = append(result.Errs, fmt.Errorf("%s: %w", t.label(), err))
		}
	}
	return result
}

// resumeSeatTargets is the seats of events Resume looks at: active real
// seats (Roster without dry-run events) with a pane, of orgID when it is
// set, every seat that is not a leader first and then the leaders (FR-4: a
// leader that goes on first could send a seat still stopped a TASK built on
// what it knew before), each group in Roster's order. A seat is a leader
// when its role or its seat id is LeaderIdentity. A seat in the middle of
// its spawn has a latest state event (spawn_step) with no role or driver,
// so those come from the latest state event that records one.
func resumeSeatTargets(events []ManifestEvent, orgID string) []resumeTarget {
	var seats, leaders []resumeTarget
	for _, s := range Roster(events, RosterOptions{}) {
		if !s.Active || s.PaneID == "" || (orgID != "" && s.OrgID != orgID) {
			continue
		}
		t := resumeTarget{
			kind: ResumeTargetSeat, orgID: s.OrgID, id: s.SeatID, role: s.Role, driver: s.Driver,
			paneID: s.PaneID, agentName: resolvedHerdrAgentName(s),
		}
		if t.role == "" {
			t.role = seatRoleFromEvents(events, s.OrgID, s.SeatID)
		}
		if t.driver == "" {
			t.driver = seatDriverFromEvents(events, s.OrgID, s.SeatID)
		}
		if t.role == LeaderIdentity || t.id == LeaderIdentity {
			leaders = append(leaders, t)
		} else {
			seats = append(seats, t)
		}
	}
	return append(seats, leaders...)
}

// seatDriverFromEvents is the driver of seatID in orgID: the Driver of the
// seat's latest real state event that records one (see seatRoleFromEvents).
func seatDriverFromEvents(events []ManifestEvent, orgID, seatID string) string {
	driver := ""
	for _, ev := range events {
		if !ev.DryRun && ev.OrgID == orgID && ev.SeatID == seatID && isStateEvent(ev.Event) && ev.Driver != "" {
			driver = ev.Driver
		}
	}
	return driver
}

// resumeDirectorTargets is the directors Resume looks at: every open
// headless registration of regs, in their order. One with launch cmd runs
// in a terminal outside herdr (the spec's out of scope), and one in herdr
// whose pane is not recorded yet has nothing to type into; both are listed
// as skipped.
func resumeDirectorTargets(regs []DirectorRegistration) []resumeTarget {
	targets := make([]resumeTarget, 0, len(regs))
	for _, r := range regs {
		t := resumeTarget{
			kind: ResumeTargetDirector, id: r.ID, role: DirectorRole, driver: r.Driver,
			paneID: r.PaneID, agentName: DirectorAgentName(r.Token),
		}
		switch {
		case !r.HerdrLaunch():
			t.skip = resumeReasonNotInHerdr
		case r.PaneID == "":
			t.skip = resumeReasonNoPane
		}
		targets = append(targets, t)
	}
	return targets
}

// resumeOne is steps 1 to 6 of Resume's doc comment for one target. The
// error is the target's ResumeOutcomeFailed error (or the failed append of
// its `resumed` event), nil otherwise.
func (o *Org) resumeOne(t resumeTarget, p ResumeParams, ownPane string) (ResumeTargetResult, error) {
	res := ResumeTargetResult{Kind: t.kind, OrgID: t.orgID, ID: t.id, Role: t.role, Driver: t.driver, PaneID: t.paneID}
	skip := func(reason string) (ResumeTargetResult, error) {
		res.Outcome, res.Reason = ResumeOutcomeSkipped, reason
		return res, nil
	}
	if t.skip != "" {
		return skip(t.skip)
	}
	if ownPane != "" && t.paneID == ownPane {
		return skip(resumeReasonOwnPane)
	}
	probe, ok := o.Herdr.(watchHerdrProbe)
	if !ok {
		return skip(resumeReasonNoAgentGet)
	}

	screen, outcome, reason := o.resumeLook(probe, t)
	res.Line = screen.Line
	if outcome != "" {
		res.Outcome, res.Reason = outcome, reason
		return res, nil
	}
	if p.DryRun {
		res.Outcome = ResumeOutcomeWouldResume
		if screen.Kind == ResumeScreenEnterOnly {
			res.Outcome = ResumeOutcomeWouldEnter
		}
		return res, nil
	}

	wait := p.LockWait
	if wait <= 0 {
		wait = defaultResumePaneLockWait
	}
	lock, err := o.acquirePaneLock(t.paneID, wait)
	if err != nil {
		if isPaneLockHeld(err) {
			return skip(resumeReasonPaneBusy)
		}
		err = fmt.Errorf("take the lock of pane %s; nothing was typed: %w", t.paneID, err)
		res.Outcome, res.Reason = ResumeOutcomeFailed, err.Error()
		return res, err
	}
	unlock := sync.OnceFunc(func() { _ = lock.release() })
	defer unlock()

	// The look again, holding the lock: a Send or another resume that held
	// it may have typed, or the screen changed since the first look.
	again, outcome, reason := o.resumeLook(probe, t)
	res.Line = again.Line
	if outcome != "" || again.Kind != screen.Kind {
		if reason == "" {
			reason = "now " + string(again.Kind)
		}
		return skip(resumeReasonScreenChanged + " (" + reason + ")")
	}

	confirmed, err := o.resumeSubmit(t, again.Kind)
	unlock()
	if err != nil {
		res.Outcome, res.Reason = ResumeOutcomeFailed, err.Error()
		return res, err
	}
	res.Outcome, res.SubmitConfirmed = ResumeOutcomeResumed, confirmed
	if again.Kind == ResumeScreenEnterOnly {
		res.Outcome = ResumeOutcomeEntered
	}
	if t.kind != ResumeTargetSeat {
		return res, nil
	}
	details := "kind=" + string(again.Kind) + " "
	if !confirmed {
		details += "submit_unconfirmed=true "
	}
	details += "line=" + truncateForDetails(again.Line)
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: t.orgID, SeatID: t.id, Event: EventResumed,
		PaneID: t.paneID, Details: details,
	}); err != nil {
		err = fmt.Errorf("the resumed event could not be recorded after Enter was pressed: %w", err)
		res.Reason = err.Error()
		return res, err
	}
	res.Recorded = true
	return res, nil
}

// resumeLook is steps 2 and 3 of Resume's doc comment: herdr's agent status
// and the screen's classification. outcome is empty when the screen is
// stopped or enter_only on an idle or done agent (go on); otherwise it is
// ResumeOutcomeWaiting or ResumeOutcomeSkipped, with reason.
func (o *Org) resumeLook(probe watchHerdrProbe, t resumeTarget) (screen ResumeScreen, outcome ResumeOutcome, reason string) {
	status, err := o.resumeAgentStatus(probe, t.agentName)
	if err != nil {
		return ResumeScreen{}, ResumeOutcomeSkipped, err.Error()
	}
	switch status {
	case "idle", "done":
	case "working":
		return ResumeScreen{}, ResumeOutcomeWaiting, resumeReasonHerdrWorking
	case "blocked":
		return ResumeScreen{}, ResumeOutcomeSkipped, resumeReasonHerdrBlocked
	default:
		return ResumeScreen{}, ResumeOutcomeSkipped, "herdr reports " + strconv.Quote(status)
	}

	var text string
	err = o.callWithTimeout(func(ctx context.Context) error {
		var err error
		text, err = o.Herdr.PaneRead(ctx, t.paneID, resumeReadLines)
		return err
	})
	if err != nil {
		return ResumeScreen{}, ResumeOutcomeSkipped, "herdr pane read " + t.paneID + " failed: " + err.Error()
	}
	screen = ClassifyResumeScreen(t.driver, text)
	switch screen.Kind {
	case ResumeScreenStopped, ResumeScreenEnterOnly:
		return screen, "", ""
	case ResumeScreenWaiting:
		return screen, ResumeOutcomeWaiting, screen.Reason
	default:
		return screen, ResumeOutcomeSkipped, "unknown screen: " + screen.Reason
	}
}

// resumeAgentStatus runs `herdr agent get` for name, with driverCallTimeout
// as its deadline, and returns the agent's status (herdrAgentStatus).
func (o *Org) resumeAgentStatus(probe watchHerdrProbe, name string) (string, error) {
	var out string
	err := o.callWithTimeout(func(ctx context.Context) error {
		var err error
		out, err = probe.AgentGet(ctx, name)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("herdr agent get %s failed: %w", name, err)
	}
	status, err := herdrAgentStatus(out)
	if err != nil {
		return "", fmt.Errorf("herdr agent get %s: %w", name, err)
	}
	return status, nil
}

// herdrAgentStatus reads the agent's status from the reply of
// `herdr agent get` (herdr 0.7.5): a JSON envelope whose
// result.agent.agent_status is the status (idle, working, blocked, done,
// unknown), as docs/evidence/org-recheck-2026-10-24/ recorded it, for
// example
// {"id":"cli:agent:get","result":{"agent":{"agent_status":"done",...},"type":"agent_info"}}.
// A reply that is not that JSON, or has no status, is an error.
func herdrAgentStatus(out string) (string, error) {
	var reply struct {
		Result struct {
			Agent struct {
				AgentStatus string `json:"agent_status"`
			} `json:"agent"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &reply); err != nil {
		return "", fmt.Errorf("the reply is not herdr's JSON: %w", err)
	}
	if reply.Result.Agent.AgentStatus == "" {
		return "", errors.New("the reply has no result.agent.agent_status")
	}
	return reply.Result.Agent.AgentStatus, nil
}

// resumeSubmit is step 5 of Resume's doc comment for a target whose screen,
// read again under the pane's lock, is kind (stopped or enter_only), and
// reports whether the submit was confirmed. The error says what is known of
// what reached the pane.
func (o *Org) resumeSubmit(t resumeTarget, kind ResumeScreenKind) (bool, error) {
	enterDelay := o.sendEnterDelay()
	ctx, cancel := context.WithTimeout(context.Background(), o.resumeSubmitBudget(enterDelay))
	defer cancel()

	enterOnly := kind == ResumeScreenEnterOnly
	checked := false
	check := func(ctx context.Context) error {
		checked = true
		text, err := o.Herdr.PaneRead(ctx, t.paneID, resumeReadLines)
		if err != nil {
			return fmt.Errorf("herdr pane read %s after typing failed: %w; the text was typed; Enter not pressed", t.paneID, err)
		}
		if !ResumeInputHolds(t.driver, text, ResumeText) {
			return errResumeTextNotInInput
		}
		return nil
	}
	progress, confirmed, err := o.typeAndSubmit(ctx, t.paneID, t.agentName, ResumeText, enterDelay, enterOnly, check)
	switch progress {
	case SendProgressTextUnacknowledged:
		// The call may have been killed on its deadline after herdr pasted
		// the text (see SendProgress).
		return false, fmt.Errorf("herdr pane send-text %s failed: %w; the text may or may not have reached the pane; Enter not pressed", t.paneID, err)
	case SendProgressTextTyped:
		if checked {
			return false, err
		}
		return false, fmt.Errorf("the pause before Enter ended: %w; the text was typed; Enter not pressed", err)
	case SendProgressEnterUnacknowledged:
		what := "Enter"
		if !enterOnly {
			what = "the text was typed and Enter"
		}
		return false, fmt.Errorf("herdr pane send-keys %s Enter failed: %w; %s may or may not have reached the pane", t.paneID, err, what)
	}
	return confirmed, nil
}

// resumeSubmitBudget is the deadline of resumeSubmit's typing: a
// driverCallTimeout each for the send-text, the pane read before Enter, and
// the send-keys, plus the pause before Enter and the submit confirmation, so
// no herdr call of it runs unbounded.
func (o *Org) resumeSubmitBudget(enterDelay time.Duration) time.Duration {
	return 3*o.driverCallTimeout() + enterDelay + o.sendSubmitConfirmTimeout()
}
