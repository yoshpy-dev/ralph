package org

import (
	"regexp"
	"strconv"
	"strings"
)

// The gate record (gateRecord): the org-level gate_passed or gate_failed
// event that ralph org gate (recordGate in gate.go) and ralph org regate
// (recordRegate in regate.go) append, or the gate_ci_passed or
// gate_ci_failed event a regate under [org.gate].regate_checks = "ci" appends
// from the checks of the PR of the branch's head on origin (via=regate_ci),
// its Details as they are written (gateRecord.details) and read back
// (gateRecordFromDetails), and the shape of the commit SHAs in it
// (gitObjectPattern). ralph org merge-order (mergeorder.go) reads the
// records.

// gitObjectPattern is the shape of a commit SHA the gate records: 40
// lowercase hex digits, or 64 in a SHA-256 repository.
var gitObjectPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// The keys of a gate record's Details (EventGatePassed, EventGateFailed,
// EventGateCIPassed, EventGateCIFailed), written in this order (reason only
// on a failure, via only on a record ralph org regate wrote), each as one
// space-separated `key=value` token. log comes last because its value is
// strconv.Quote'd and may hold spaces, the rule of plan_approved's cwd
// (planApproval.details): gateRecordFromDetails reads everything after
// " log=" as the log, so a key written after it would be lost.
const (
	gateRecordReasonKey    = "reason"
	gateRecordHeadKey      = "head"
	gateRecordBaseKey      = "base"
	gateRecordBaseRefKey   = "base_ref"
	gateRecordBranchKey    = "branch"
	gateRecordByKey        = "by"
	gateRecordHerdrPaneKey = "herdr_pane"
	gateRecordViaKey       = "via"
	gateRecordLogKey       = "log"
)

// The via= of a gate record ralph org regate wrote (regate.go): gateViaRegate
// when it ran the gate scripts, gateViaRegateCI when it read the checks of the
// PR of the branch's head on origin instead ([org.gate].regate_checks = "ci").
// A record ralph org gate writes has no via=.
const (
	gateViaRegate   = "regate"
	gateViaRegateCI = "regate_ci"
)

// gateRecord is one gate record (EventGatePassed, EventGateFailed,
// EventGateCIPassed, EventGateCIFailed): what the gate ran on and who ran it.
type gateRecord struct {
	Reason  string // GateReason*; "" on a pass
	Head    string // HEAD before the scripts ran; the head on origin, for via=regate_ci
	Base    string // the base ref's tip
	BaseRef string // refs/remotes/origin/<name> or refs/heads/<name>
	Branch  string // the feature branch
	By      string // USER, or approverUnknown (approverName)
	// HerdrPane: HERDR_PANE_ID was set in the caller's environment.
	HerdrPane bool
	// Via is the verb that ran the gate when it is not ralph org gate
	// (gateViaRegate, gateViaRegateCI), "" for ralph org gate.
	Via string
	Log string // the absolute path of the gate log
}

// event is the manifest event r is recorded as: for via=regate_ci,
// gate_ci_passed when Reason is empty and gate_ci_failed otherwise; for any
// other record, gate_passed and gate_failed the same way. A reader pairs it
// with the event's name, so a CI record under a gate_passed or gate_failed
// name, or another record under a gate_ci_* name, does not read.
func (r gateRecord) event() string {
	ci := r.Via == gateViaRegateCI
	switch {
	case r.Reason == "" && ci:
		return EventGateCIPassed
	case r.Reason == "":
		return EventGatePassed
	case ci:
		return EventGateCIFailed
	}
	return EventGateFailed
}

// details is the Details of r's event (see the gateRecord*Key constants).
func (r gateRecord) details() string {
	pane := "unset"
	if r.HerdrPane {
		pane = "set"
	}
	var d string
	if r.Reason != "" {
		d = gateRecordReasonKey + "=" + r.Reason + " "
	}
	d += gateRecordHeadKey + "=" + r.Head +
		" " + gateRecordBaseKey + "=" + r.Base +
		" " + gateRecordBaseRefKey + "=" + r.BaseRef +
		" " + gateRecordBranchKey + "=" + r.Branch +
		" " + gateRecordByKey + "=" + r.By +
		" " + gateRecordHerdrPaneKey + "=" + pane
	if r.Via != "" {
		d += " " + gateRecordViaKey + "=" + r.Via
	}
	return d + " " + gateRecordLogKey + "=" + strconv.Quote(r.Log)
}

// gateRecordFromDetails reads a gate event's Details back (stage 7c reads
// the records with it). ok is false when a key before log is given twice or a
// token is not `key=value`, when head or base is not a commit SHA
// (gitObjectPattern), base_ref is not under refs/, branch is empty, or reason
// is set to anything but a GateReason* value. Keys it does not know are
// skipped. by, herdr_pane, via and log are read on a best-effort basis: they
// are for an audit. A record whose Reason is empty is a pass, and one whose
// Via is regate_ci a CI record (gateRecord.event); a reader checks that
// against the event's name.
func gateRecordFromDetails(details string) (r gateRecord, ok bool) {
	head, logValue, hasLog := strings.Cut(details, " "+gateRecordLogKey+"=")
	values := make(map[string]string)
	for _, token := range strings.Fields(head) {
		key, value, found := strings.Cut(token, "=")
		if _, dup := values[key]; !found || dup {
			return gateRecord{}, false
		}
		values[key] = value
	}
	r = gateRecord{
		Reason: values[gateRecordReasonKey], Head: values[gateRecordHeadKey], Base: values[gateRecordBaseKey],
		BaseRef: values[gateRecordBaseRefKey], Branch: values[gateRecordBranchKey], By: values[gateRecordByKey],
		HerdrPane: values[gateRecordHerdrPaneKey] == "set", Via: values[gateRecordViaKey],
	}
	if _, hasReason := values[gateRecordReasonKey]; hasReason && !isGateReason(r.Reason) {
		return gateRecord{}, false
	}
	if !gitObjectPattern.MatchString(r.Head) || !gitObjectPattern.MatchString(r.Base) ||
		!strings.HasPrefix(r.BaseRef, "refs/") || r.Branch == "" {
		return gateRecord{}, false
	}
	if hasLog {
		if s, err := strconv.Unquote(logValue); err == nil {
			r.Log = s
		}
	}
	return r, true
}

// GateReasonCIFailed is the reason of a gate_ci_failed: a check of the PR of
// the branch's head on origin, which ralph org regate pushed or found there,
// failed, with none pending (plan
// docs/plans/active/2026-10-29-regate-ci-checks.md, D4). The other
// GateReason* values are in gate.go.
const GateReasonCIFailed = "ci_failed"

// isGateReason reports whether reason is one of the GateReason* values.
func isGateReason(reason string) bool {
	switch reason {
	case GateReasonStaticFailed, GateReasonTestFailed, GateReasonHeadMoved, GateReasonTreeChanged, GateReasonCIFailed:
		return true
	}
	return false
}
