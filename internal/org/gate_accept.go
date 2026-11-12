package org

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The `ralph org gate accept` verb (GateAccept), S3 of plan
// docs/plans/active/2026-11-12-gate-lock-accept.md (AC7, D3, D5): it records a
// human's acceptance of the failure that a branch's latest gate record holds,
// as the org-level event gate_accepted (EventGateAccepted), so that ralph org
// merge-order lists the branch as accepted where a pass would be ready, and
// ralph org merge merges it as it merges a ready one (collectMergeTargets and
// classifyAccepted in mergeorder.go, checkMergeBranch in merge.go). The
// acceptance names the failure by its gate record (head, base, base_ref,
// branch, event, reason, ts, and log, GateAcceptance.accepts), so any gate
// record after it voids it. A human runs the verb, or a director the human
// told to; it refuses a seat's pane (seatPaneTripwire), as ralph org approve
// does. It reads the ledger and appends under one hold of the manifest lock,
// so two accepts of the same failure record one.

const (
	// gateAcceptPrefix starts every refusal and error of the verb.
	gateAcceptPrefix = "org: gate accept: "
	// GateAcceptReasonMaxRunes is the most characters --reason may have once
	// trimmed.
	GateAcceptReasonMaxRunes = 500
)

// The keys of a gate_accepted's Details that a gate record's Details do not
// have (the others are the gateRecord*Key constants: head, base, base_ref,
// branch, by, herdr_pane). GateAcceptance.details writes failed_log and note
// last, in that order, since their values are strconv.Quote'd and may hold
// spaces; gateAcceptanceFromDetails reads them back with strconv.QuotedPrefix.
const (
	gateAcceptedFailedEventKey  = "failed_event"
	gateAcceptedFailedReasonKey = "failed_reason"
	gateAcceptedFailedTSKey     = "failed_ts"
	gateAcceptedFailedLogKey    = "failed_log"
	gateAcceptedNoteKey         = "note"
)

// GateAcceptParams describes one `ralph org gate accept` call.
type GateAcceptParams struct {
	// Branch is the feature branch whose latest gate failure is accepted.
	Branch string
	// Log is the gate log of that failure, as its gate record holds it: the
	// LOG: line ralph org gate printed, or the gate's log in ralph org
	// merge-order --json. Each gate run writes a log of its own, so it names
	// the failure the human read.
	Log string
	// Reason is why the human accepts the failure: one line, trimmed, of at
	// most GateAcceptReasonMaxRunes characters.
	Reason string
}

// GateAcceptance is one gate_accepted record: the failure it accepts, as
// that failure's gate record holds it, and who accepted it and why.
type GateAcceptance struct {
	// TS is when the acceptance was recorded (the event's ts), and OrgID the
	// org of the failure's gate record, under which it is recorded. Neither
	// is in Details.
	TS, OrgID string
	// Head, Base, BaseRef, and Branch are the failure's gate record's.
	Head, Base, BaseRef, Branch string
	// FailedEvent (gate_failed or gate_ci_failed), FailedReason
	// (static_failed, test_failed, or ci_failed), FailedTS, and FailedLog are
	// the failure's event name, reason, ts, and log.
	FailedEvent, FailedReason, FailedTS, FailedLog string
	// By is USER, or approverUnknown (approverName); HerdrPane reports that
	// HERDR_PANE_ID was set, to no active seat's pane.
	By        string
	HerdrPane bool
	// Note is why the human accepted the failure (--reason).
	Note string
}

// GateAcceptResult is GateAccept's outcome: Accepted, the acceptance it
// recorded, or Refused, why it recorded nothing, escaped
// (PrintableInboxText).
type GateAcceptResult struct {
	Accepted *GateAcceptance
	Refused  string
}

// details is the Details of a's gate_accepted (EventGateAccepted).
func (a GateAcceptance) details() string {
	pane := "unset"
	if a.HerdrPane {
		pane = "set"
	}
	return gateRecordHeadKey + "=" + a.Head +
		" " + gateRecordBaseKey + "=" + a.Base +
		" " + gateRecordBaseRefKey + "=" + a.BaseRef +
		" " + gateRecordBranchKey + "=" + a.Branch +
		" " + gateAcceptedFailedEventKey + "=" + a.FailedEvent +
		" " + gateAcceptedFailedReasonKey + "=" + a.FailedReason +
		" " + gateAcceptedFailedTSKey + "=" + a.FailedTS +
		" " + gateRecordByKey + "=" + a.By +
		" " + gateRecordHerdrPaneKey + "=" + pane +
		" " + gateAcceptedFailedLogKey + "=" + strconv.Quote(a.FailedLog) +
		" " + gateAcceptedNoteKey + "=" + strconv.Quote(a.Note)
}

// gateAcceptanceFromDetails reads a gate_accepted's Details back; TS and
// OrgID are left for the caller to set from the event. ok is false for
// anything malformed: no ` failed_log=`, a token before it that is not
// `key=value` or a key given twice, failed_log and note that are not two
// double-quoted strings, in that order, with nothing after note, and any of
// head, base, base_ref, branch, failed_event, failed_reason, failed_ts, by,
// herdr_pane, failed_log, and note missing or out of shape: head and base
// commit SHAs (gitObjectPattern), base_ref under refs/, failed_event
// gate_failed or gate_ci_failed, failed_reason one gateAcceptableReason
// accepts, herdr_pane set or unset, branch, failed_ts, by, and failed_log not
// empty, and note a line gateAcceptNoteProblem passes. Keys it does not know
// are skipped, as gateRecordFromDetails skips them.
func gateAcceptanceFromDetails(details string) (a GateAcceptance, ok bool) {
	tokens, quoted, found := strings.Cut(details, " "+gateAcceptedFailedLogKey+"=")
	if !found {
		return GateAcceptance{}, false
	}
	values := make(map[string]string)
	for _, token := range strings.Split(tokens, " ") {
		key, value, isPair := strings.Cut(token, "=")
		if _, dup := values[key]; !isPair || key == "" || dup {
			return GateAcceptance{}, false
		}
		values[key] = value
	}
	logQuoted, err := strconv.QuotedPrefix(quoted)
	if err != nil || !strings.HasPrefix(logQuoted, `"`) {
		return GateAcceptance{}, false
	}
	rest, isNote := strings.CutPrefix(quoted[len(logQuoted):], " "+gateAcceptedNoteKey+"=")
	if !isNote {
		return GateAcceptance{}, false
	}
	noteQuoted, err := strconv.QuotedPrefix(rest)
	if err != nil || noteQuoted != rest || !strings.HasPrefix(noteQuoted, `"`) {
		return GateAcceptance{}, false
	}
	// QuotedPrefix returned both, so both unquote.
	a.FailedLog, _ = strconv.Unquote(logQuoted)
	a.Note, _ = strconv.Unquote(noteQuoted)
	a.Head, a.Base, a.BaseRef, a.Branch = values[gateRecordHeadKey], values[gateRecordBaseKey], values[gateRecordBaseRefKey],
		values[gateRecordBranchKey]
	a.FailedEvent, a.FailedReason, a.FailedTS = values[gateAcceptedFailedEventKey], values[gateAcceptedFailedReasonKey],
		values[gateAcceptedFailedTSKey]
	a.By = values[gateRecordByKey]
	switch values[gateRecordHerdrPaneKey] {
	case "set":
		a.HerdrPane = true
	case "unset":
	default:
		return GateAcceptance{}, false
	}
	if !gitObjectPattern.MatchString(a.Head) || !gitObjectPattern.MatchString(a.Base) || !strings.HasPrefix(a.BaseRef, "refs/") ||
		a.Branch == "" || (a.FailedEvent != EventGateFailed && a.FailedEvent != EventGateCIFailed) ||
		!gateAcceptableReason(a.FailedReason) || a.FailedTS == "" || a.By == "" || a.FailedLog == "" ||
		gateAcceptNoteProblem(a.Note) != "" {
		return GateAcceptance{}, false
	}
	return a, true
}

// accepts reports whether a is the acceptance of g, a gate record attached
// to a branch: g is a readable failure (a reason set) recorded under a's
// org, and a names its head, base, base_ref, branch, event, reason, ts, and
// log.
func (a GateAcceptance) accepts(g *mergeGateRecord) bool {
	r := g.record
	return g.readable && r.Reason != "" && a.OrgID == g.ev.OrgID && a.Head == r.Head && a.Base == r.Base &&
		a.BaseRef == r.BaseRef && a.Branch == r.Branch && a.FailedEvent == g.ev.Event && a.FailedReason == r.Reason &&
		a.FailedTS == g.ev.TS && a.FailedLog == r.Log
}

// gateAcceptableReason reports whether a gate failure with reason may be
// accepted: static_failed, test_failed, and ci_failed, where a script or a
// check failed. head_moved and tree_changed are failures of scripts that
// passed, so the gate is run again instead (D5).
func gateAcceptableReason(reason string) bool {
	switch reason {
	case GateReasonStaticFailed, GateReasonTestFailed, GateReasonCIFailed:
		return true
	}
	return false
}

// gateAcceptNoteProblem says what is wrong with note, the trimmed --reason or
// a note read back from a gate_accepted, "" when nothing is: it is empty,
// spans lines (isInboxLineBreak), is not valid UTF-8, has more than
// GateAcceptReasonMaxRunes characters, or holds a control character.
func gateAcceptNoteProblem(note string) string {
	switch {
	case note == "":
		return "is required: say in one line why the human accepts the failure"
	case strings.IndexFunc(note, isInboxLineBreak) >= 0:
		return "must be a single line"
	case !utf8.ValidString(note):
		return "is not valid UTF-8"
	}
	if n := utf8.RuneCountInString(note); n > GateAcceptReasonMaxRunes {
		return fmt.Sprintf("is %d characters; at most %d", n, GateAcceptReasonMaxRunes)
	}
	if i := strings.IndexFunc(note, unicode.IsControl); i >= 0 {
		r, _ := utf8.DecodeRuneInString(note[i:])
		return fmt.Sprintf("contains a control character (%U)", r)
	}
	return ""
}

// GateAcceptParamsRefusal is the refusal of p's flags, escaped, "" when they
// pass: a --branch or --log that is empty or blank, and a --reason that
// gateAcceptNoteProblem refuses once trimmed. The CLI calls it before it
// reads the ledger, and GateAccept calls it again.
func GateAcceptParamsRefusal(p GateAcceptParams) string {
	switch {
	case strings.TrimSpace(p.Branch) == "":
		return gateAcceptPrefix + "--branch is required: name the feature branch whose latest gate failure the human accepts " +
			"(ralph org merge-order lists the ledger's branches)"
	case strings.TrimSpace(p.Log) == "":
		return gateAcceptPrefix + "--log is required: name the gate log of the failure the human read and accepts (the LOG: " +
			"line of ralph org gate, or the gate's log in ralph org merge-order --json)"
	}
	if why := gateAcceptNoteProblem(strings.TrimSpace(p.Reason)); why != "" {
		return PrintableInboxText(gateAcceptPrefix+"--reason "+why, false)
	}
	return ""
}

// GateAccept runs `ralph org gate accept` (AC7 of plan
// docs/plans/active/2026-11-12-gate-lock-accept.md). It refuses p's flags
// first (GateAcceptParamsRefusal). Then, under the manifest lock, it reads
// the ledger (readMergeLedger), refuses when HERDR_PANE_ID is the pane of an
// active seat (seatPaneTripwire), and finds p.Branch's latest gate record as
// ralph org merge-order does (collectMergeTargets): it refuses when no gate
// record names the branch, the latest cannot be read or a corrupt line
// follows it, it is a pass, its reason is head_moved or tree_changed (the
// scripts passed: run the gate again), its log cannot be read or is not
// p.Log (a newer failure replaced the one the human read: read the latest
// one's log and ask again), or an acceptance of that failure is already
// recorded. Otherwise it appends one gate_accepted under the org of that
// gate record, in the same lock hold, with Details the failure's record and
// by (USER, approverName), herdr_pane, and the trimmed p.Reason
// (GateAcceptance.details). A refusal records nothing; every refusal is
// escaped (PrintableInboxText). The error is for a manifest that cannot be
// read, a lock that cannot be taken, and an append that fails; nothing is
// recorded then either.
func (o *Org) GateAccept(p GateAcceptParams) (GateAcceptResult, error) {
	if refused := GateAcceptParamsRefusal(p); refused != "" {
		return GateAcceptResult{Refused: refused}, nil
	}
	pane := o.getenv(herdrPaneIDEnv)
	var res GateAcceptResult
	var recordErr error
	lockErr := withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
		ledger, err := readMergeLedger(o.Manifest.Path())
		if err != nil {
			recordErr = fmt.Errorf(gateAcceptPrefix+"read manifest: %w", err)
			return nil
		}
		if err := seatPaneTripwire(ledger.events, pane, "gate accept", "a human's acceptance of a gate failure"); err != nil {
			res.Refused = PrintableInboxText(err.Error(), false)
			return nil
		}
		g, refused := gateAcceptFailure(ledger, p)
		if refused != "" {
			res.Refused = PrintableInboxText(refused, false)
			return nil
		}
		a := GateAcceptance{
			TS: o.now(), OrgID: g.ev.OrgID, Head: g.record.Head, Base: g.record.Base, BaseRef: g.record.BaseRef,
			Branch: g.record.Branch, FailedEvent: g.ev.Event, FailedReason: g.record.Reason, FailedTS: g.ev.TS,
			FailedLog: g.record.Log, By: approverName(o.getenv("USER")), HerdrPane: pane != "", Note: strings.TrimSpace(p.Reason),
		}
		if err := o.appendEvent(ManifestEvent{TS: a.TS, OrgID: a.OrgID, SeatID: "", Event: EventGateAccepted, Details: a.details()}); err != nil {
			recordErr = fmt.Errorf(gateAcceptPrefix+"nothing was accepted, since %s could not be recorded: %w", EventGateAccepted, err)
			return nil
		}
		res.Accepted = &a
		return nil
	})
	switch {
	case lockErr != nil:
		return GateAcceptResult{}, fmt.Errorf(gateAcceptPrefix+"nothing was accepted, since the manifest lock could not be taken: %w", lockErr)
	case recordErr != nil:
		return GateAcceptResult{}, recordErr
	}
	return res, nil
}

// gateAcceptFailure is the gate record GateAccept accepts for p.Branch in
// ledger, read under the manifest lock: the branch's latest gate record as
// collectMergeTargets attaches it, or the refusal that says why there is
// none to accept.
func gateAcceptFailure(ledger mergeLedger, p GateAcceptParams) (*mergeGateRecord, string) {
	targets, _ := collectMergeTargets(ledger)
	i := slices.IndexFunc(targets, func(t *mergeTarget) bool { return t.branch == p.Branch })
	if i < 0 || targets[i].gate == nil {
		return nil, fmt.Sprintf(gateAcceptPrefix+"no gate record in the ledger names the branch %s, so there is no gate failure "+
			"to accept and nothing was recorded: ralph org merge-order lists the ledger's branches and their gate records", p.Branch)
	}
	t := targets[i]
	g, r := t.gate, t.gate.record
	latest := fmt.Sprintf("the latest gate record of the branch %s (%s at %s)", p.Branch, g.ev.Event, g.ev.TS)
	const again = "follow the branch's Detail in ralph org merge-order to run the gate again"
	switch {
	case !g.readable:
		return nil, fmt.Sprintf(gateAcceptPrefix+"%s cannot be read, so which failure it holds is unknown and nothing was "+
			"recorded: %s", latest, again)
	case t.corruptAfterGate:
		return nil, fmt.Sprintf(gateAcceptPrefix+"a corrupt manifest line follows %s and may be a newer one, so nothing was "+
			"recorded: %s", latest, again)
	case r.Reason == "":
		return nil, fmt.Sprintf(gateAcceptPrefix+"%s is a pass, not a failure, so there is nothing to accept and nothing was "+
			"recorded: ralph org merge-order lists the branch by that pass", latest)
	case r.Reason == GateReasonHeadMoved || r.Reason == GateReasonTreeChanged:
		return nil, fmt.Sprintf(gateAcceptPrefix+"%s failed with reason %s: the scripts passed, so there is nothing to accept and "+
			"nothing was recorded: %s", latest, r.Reason, again)
	case !gateAcceptableReason(r.Reason):
		return nil, fmt.Sprintf(gateAcceptPrefix+"%s failed with reason %s, which cannot be accepted, so nothing was recorded: %s",
			latest, r.Reason, again)
	case r.Log == "":
		return nil, fmt.Sprintf(gateAcceptPrefix+"%s failed with reason %s, but its log cannot be read from the record, so the "+
			"failure --log names cannot be checked against it and nothing was recorded: %s", latest, r.Reason, again)
	case r.Log != p.Log:
		return nil, fmt.Sprintf(gateAcceptPrefix+"%s failed with reason %s and its log is %s, not %s, so nothing was recorded: a "+
			"newer gate may have replaced the failure that was read; read %s and ask the human again before accepting that "+
			"failure", latest, r.Reason, r.Log, p.Log, r.Log)
	case t.accepted != nil:
		return nil, fmt.Sprintf(gateAcceptPrefix+"%s failed with reason %s, and %s accepted that failure at %s already, so "+
			"nothing more was recorded: ralph org merge-order lists the branch by that acceptance", latest, r.Reason,
			t.accepted.By, t.accepted.TS)
	}
	return g, ""
}
