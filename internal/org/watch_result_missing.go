package org

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The result_missing condition of the cross-org watch (plan
// 2026-11-22-org-result-missing, S2; issue #361): a seat other than the
// leader that received a TASK and sits waiting for input without having sent
// the leader the RESULT (or REVIEW) of that TASK gets an ALERT to the leader
// that names the seat and the TASK_ID. It reads three sources: the ledger's
// `sent` events (the TASK, a STOP after it, and the leader's answer to a
// QUESTION or BLOCKED), herdr's agent status (watchSeatSnapshot.IdleSince),
// and the seat's agmsg history (the lines the seat sent the leader). What it
// cannot read is not raised. The seat's stall is held while the condition's
// clock runs and while the condition holds (resultMissingVerdict), so the
// leader gets one ALERT for a seat that sits idle without its result, not a
// stall before the result_missing. A rise for a TASK other than the one of
// the condition's last ALERT is a new ALERT, never a recurrence of that
// ALERT or of its inbox item (raiseResultMissing; the plan's self-review
// M1).

// resultMissingVerdict is what resultMissing found for a seat in a cycle.
type resultMissingVerdict int

const (
	// resultMissingNo: the condition does not hold, and the seat's stall is
	// decided as before. This is also the verdict once the clock has run out
	// and the history says the TASK is done, that the seat waits for the
	// leader, or nothing (it cannot be read or does not reach back). For a
	// TASK whose history did not reach back with the last limit
	// (snap.ResultOutOfReach), it is the verdict in every later cycle too,
	// from the start: the clock does not run again for that TASK, so the
	// seat's stall is not held, even after the seat works and waits again.
	// It is the verdict, too, while the clock started by the leader's answer
	// to a QUESTION or BLOCKED runs: that answer is a `sent` event of the
	// seat, so the seat's last event is no older than it and its stall does
	// not hold before that clock runs out.
	resultMissingNo resultMissingVerdict = iota
	// resultMissingClockRuns: the seat waits for input with a last TASK not
	// known to be done (no STOP after it, not in snap.ResultSeen) and not
	// known to be out of the history's reach (snap.ResultOutOfReach), and
	// stall_minutes have not passed since the later of the TASK's time and
	// snap.IdleSince. The condition may hold once they have, so the seat
	// gets no stall in this cycle. It is read from the ledger and the
	// snapshot only, with no history read. This clock starts at the TASK's
	// time, never later than the condition's own clock, which after a
	// QUESTION or BLOCKED starts at the leader's answer, so the stall is
	// never held longer than that clock: the hold ends in the first cycle
	// resultMissing reads the history in.
	resultMissingClockRuns
	// resultMissingHolds: the condition holds; its ALERT is sent, and the
	// seat gets no stall in this cycle.
	resultMissingHolds
)

// resultMissingHistoryLimits are the limits of the reads of a seat's agmsg
// history, in order: when a read does not reach back to resultMissingSentLag
// before the TASK's time (historyReachesBack), the next limit is tried, and
// after the last one the read is unknown and the condition is not raised.
// Such a TASK is remembered (watchSeatSnapshot.ResultOutOfReach) and its
// history is not read again: the seat's lines after the TASK only grow, so a
// later read does not reach back either.
var resultMissingHistoryLimits = []int{50, 200, 1000}

// resultMissingSentLag bounds how much later than the Enter that delivered a
// message the ledger's `sent` event of it is timed, and result_missing widens
// by it its two comparisons of a seat's lines with `sent` times (cross-review
// cycle 1 of plan 2026-11-22-org-result-missing): the seat's lines to the
// leader count as replies to the TASK from this much before the TASK's time
// on, and the history is read back to then; and a `sent` to the seat after
// its QUESTION or BLOCKED is taken for the leader's answer only when it is
// timed more than this much after that line (firstSentAfter).
//
// Send (verbs.go) takes the event's time (o.now()) only once typeAndSubmit
// has returned and the pane lock is released. After Enter, typeAndSubmit
// runs the submit check: on herdr's path herdr's agent wait
// (confirmSubmitted), which herdr ends at its --timeout of
// defaultSendSubmitConfirmTimeout (3 s); on the screen path the pane reads
// of sendScreenSubmitChecks, which stop at that same window, and a read still
// running then is killed and waited for up to the driver's exec WaitDelay
// (2 s, execWaitDelay in package driver). A seat with a short turn can send
// its RESULT, QUESTION, or BLOCKED in that time, so the line is timed before
// the TASK's `sent` event. 10 s is those 5 s and 5 s more for the herdr
// calls to start and return (PaneSendKeys returns after the key reached the
// pane) on a loaded machine. It is whole seconds: the ledger and agmsg both
// time to the second, so a line sent at or after the Enter is never timed
// before the TASK's time less this bound.
//
// A change that makes that time longer (a longer submit check, another step
// between Enter and the record) must raise this bound with it
// (TestResultMissing_SentLagCoversSendsSubmitCheck holds it at or above the
// submit check's window, the WaitDelay, and the 5 s for the herdr calls, and
// TestAgentMessagingRule_ResultMissingAndTheSeatsSendCheck has the protocol
// rule say it in seconds). With the bound too short, a reply the seat sent in
// the time beyond it is not counted, and the condition may rise for a seat
// that answered; and a `sent` typed before the seat's QUESTION but recorded
// more than the bound after it is taken for its answer, so the clock starts
// there and the condition may rise while the seat waits for the leader. The
// widening errs the other way: a RESULT of an earlier TASK with the same
// TASK_ID, sent within the bound before the new TASK's time, counts for the
// new TASK, and an answer recorded within the bound after the seat's line is
// not taken for it (a later `sent` is), so the condition does not rise for
// them.
const resultMissingSentLag = 10 * time.Second

// agmsgNoHistory is the whole output of the agmsg skill's history.sh for a
// team or an agent that has no message yet.
const agmsgNoHistory = "No message history."

// historyEscapes undoes the escaping history.sh does to a message body to
// keep it on one line: it writes a newline as `\n` and a tab as `\t`. It
// leaves a backslash as it is, so the undo is not exact: the two characters
// `\n` or `\t` written in a body (in a path or a pattern, say) also come
// back as a newline or a tab. A header line cut that way ends the header
// block there (protocol.Parse), and a TASK_ID after it is read as body;
// the TYPE on the first line is read as sent unless its own value holds
// them.
var historyEscapes = strings.NewReplacer(`\n`, "\n", `\t`, "\t")

// resultMissingTask is the last TASK the leader sent to a seat in the seat's
// current life (seatLastTask): its TASK_ID, the time of its `sent` event as
// the ledger has it (TS) and read to the second (At), and the event's index
// in the cycle's events.
type resultMissingTask struct {
	TaskID string
	TS     string
	At     time.Time
	Index  int
}

// resultMissingTaskKey is the watchSeatSnapshot.ResultSeen,
// ResultOutOfReach, and ResultMissingTask value of task: the time of its
// `sent` event and its TASK_ID, so a later TASK with the same TASK_ID is a
// different one.
func resultMissingTaskKey(task resultMissingTask) string {
	return task.TS + " " + task.TaskID
}

// resultMissingFinding is what resultMissing found for a seat in a cycle:
// the verdict, and, when it is resultMissingHolds, the TASK the condition
// holds for and the ALERT to send the leader.
type resultMissingFinding struct {
	Verdict resultMissingVerdict
	Task    resultMissingTask
	Message string
}

// agmsgHistoryRecord is one line of the output of the agmsg skill's
// history.sh: `<status> [<ts>] <from> → <to>: <body>`, with the body's
// escapes undone (historyEscapes) and the time read to the second.
type agmsgHistoryRecord struct {
	At             time.Time
	From, To, Body string
}

// resultMissingLine is a line a seat sent the leader, as result_missing reads
// it: its time, and the TYPE and TASK_ID of its body (both "" for a body that
// is not a typed message).
type resultMissingLine struct {
	At           time.Time
	Type, TaskID string
}

// updateIdleSince records in snap the first cycle of the run of cycles in
// which out, the seat's herdr agent get reply of this cycle, says the seat is
// not working: it sets IdleSince to now when it holds no time, and clears it
// for a reply that says working or that does not parse as herdr's
// (herdrAgentStatus), which does not show the seat waiting for input.
func updateIdleSince(snap *watchSeatSnapshot, out string, now time.Time) {
	agentStatus, err := herdrAgentStatus(out)
	if err != nil || agentStatus == "working" {
		snap.IdleSince = ""
		return
	}
	if _, err := time.Parse(time.RFC3339, snap.IdleSince); err != nil {
		snap.IdleSince = now.UTC().Format(time.RFC3339)
	}
}

// resultMissing reports whether the result_missing condition holds this
// cycle for seatID of orgID, a seat other than the leader (verdict
// resultMissingHolds), and returns, when it does, the TASK it holds for and
// the ALERT to send. For a seat that meets the first two conditions below
// and waits for input, it returns resultMissingClockRuns, without reading
// the history, until stall_minutes have passed since the later of the
// TASK's time and IdleSince, and the seat's stall is held. It holds when all
// of these do:
//
//   - the seat's current life has a last TASK (seatLastTask) whose TASK_ID is
//     known and that no STOP to the seat followed;
//   - its RESULT was not already found for that TASK (snap.ResultSeen), and
//     the history was not already found out of reach for it
//     (snap.ResultOutOfReach);
//   - the seat has been waiting for input since snap.IdleSince
//     (updateIdleSince), and [org.watchdog].stall_minutes have passed since
//     the later of IdleSince and the start of the clock;
//   - the seat's agmsg history, read back to resultMissingSentLag before the
//     TASK's time (seatLinesToLeader), holds no RESULT or REVIEW with that
//     TASK_ID among the lines the seat sent the leader from then on, to the
//     second: the TASK's `sent` event is timed after the Enter that delivered
//     it, so the seat's reply may be timed before it. One that does is
//     recorded in snap.ResultSeen.
//
// The clock starts at the TASK's time, unless the last of those lines is a
// QUESTION (with any TASK_ID or none) or a BLOCKED of that TASK_ID: the seat
// then waits for the leader, and the clock starts at the first `sent` to the
// seat timed more than resultMissingSentLag after that line's second
// (firstSentAfter), since one timed closer to it may have been typed before
// the line came; with none, the condition does not hold. A reply the leader
// sends after a RESULT does not start it again, as the RESULT ends the TASK.
// The ALERT says where the clock started (resultMissingAlertMessage).
//
// The history is read only once the time has passed since the TASK's time,
// the earliest the clock can start, so a seat that cannot be raised costs no
// agmsg call. A history that cannot be read, that does not reach back to
// resultMissingSentLag before the TASK's time, or that has a line it cannot
// parse does not raise the condition; one that does not reach back with the
// last limit is recorded in snap.ResultOutOfReach, so it is not read for that
// TASK again.
func (w *watchRun) resultMissing(orgID, seatID string, snap *watchSeatSnapshot, now time.Time, events []ManifestEvent) resultMissingFinding {
	stall := time.Duration(w.cfg.Watchdog.StallMinutes) * time.Minute
	if stall <= 0 || snap.IdleSince == "" {
		return resultMissingFinding{}
	}
	idleSince, err := time.Parse(time.RFC3339, snap.IdleSince)
	if err != nil {
		return resultMissingFinding{}
	}
	task, ok := seatLastTask(events, orgID, seatID)
	if !ok {
		return resultMissingFinding{}
	}
	taskKey := resultMissingTaskKey(task)
	if snap.ResultSeen == taskKey || snap.ResultOutOfReach == taskKey {
		return resultMissingFinding{}
	}
	if !waitedFor(now, stall, task.At, idleSince) {
		return resultMissingFinding{Verdict: resultMissingClockRuns}
	}
	repliesFrom := task.At.Add(-resultMissingSentLag)
	lines, ok, outOfReach := w.seatLinesToLeader(orgID, seatID, repliesFrom)
	if outOfReach {
		snap.ResultOutOfReach = taskKey
	}
	if !ok {
		return resultMissingFinding{}
	}
	var last *resultMissingLine
	for i, l := range lines {
		if l.At.Before(repliesFrom) {
			continue
		}
		if (l.Type == protocol.TypeResult || l.Type == protocol.TypeReview) && l.TaskID == task.TaskID {
			snap.ResultSeen = taskKey
			return resultMissingFinding{}
		}
		last = &lines[i]
	}
	start, waitedOn := task.At, ""
	if last != nil && (last.Type == protocol.TypeQuestion || (last.Type == protocol.TypeBlocked && last.TaskID == task.TaskID)) {
		answered, ok := firstSentAfter(events, orgID, seatID, task.Index, last.At.Add(resultMissingSentLag))
		if !ok {
			return resultMissingFinding{}
		}
		start, waitedOn = answered, last.Type
	}
	if !waitedFor(now, stall, start, idleSince) {
		return resultMissingFinding{}
	}
	return resultMissingFinding{
		Verdict: resultMissingHolds,
		Task:    task,
		Message: resultMissingAlertMessage(orgID, seatID, task, waitedOn, start, w.cfg.Watchdog.StallMinutes),
	}
}

// resultMissingAlertMessage is the ALERT result_missing sends the leader
// about task of seatID. It does not say the seat failed: it says what the
// watch saw, that the seat sent the leader no RESULT or REVIEW of task since
// task was sent (the watch counts the seat's lines from resultMissingSentLag
// before task's time on, as the seat may reply before the `sent` event of
// task is recorded), and where the clock started. With waitedOn "" it started
// at task. With waitedOn QUESTION or BLOCKED (the TYPE of the last of the
// seat's lines to the leader from resultMissingSentLag before task on) it
// started at answeredAt, the first `sent` to the seat timed more than
// resultMissingSentLag after that line (firstSentAfter), so the ALERT names
// that line and that time and does not claim the seat sent no BLOCKED. It
// asks the leader to read the seat's screen.
func resultMissingAlertMessage(orgID, seatID string, task resultMissingTask, waitedOn string, answeredAt time.Time, stallMinutes int) string {
	saw := fmt.Sprintf("seat %s has sent the leader no RESULT or REVIEW for TASK %s since that TASK was sent at %s, "+
		"and has been waiting for input for %dm or more since the TASK.",
		seatID, task.TaskID, task.At.UTC().Format(time.RFC3339), stallMinutes)
	if waitedOn != "" {
		line := "a QUESTION"
		if waitedOn == protocol.TypeBlocked {
			line = "a BLOCKED of that TASK"
		}
		saw = fmt.Sprintf("seat %s has sent the leader no RESULT or REVIEW for TASK %s since that TASK was sent at %s; "+
			"its last line to the leader since then is %s, a message was sent to the seat after it at %s, "+
			"and it has been waiting for input for %dm or more since that message.",
			seatID, task.TaskID, task.At.UTC().Format(time.RFC3339), line, answeredAt.UTC().Format(time.RFC3339), stallMinutes)
	}
	return fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\nTASK_ID: %s\n\n%s "+
		"Read its screen with `ralph org read` (a SEND FAILED: line, or a result it wrote but did not send); if it is waiting for a background command, wait.",
		orgID, seatID, condResultMissing, task.TaskID, saw)
}

// isResultMissingAlert reports whether alertID is the alert_id of an ALERT
// of result_missing (alertIDConditionType).
func isResultMissingAlert(alertID string) bool {
	return alertIDConditionType(alertID) == condResultMissing
}

// raiseResultMissing is raiseOrClear for result_missing, with the TASK the
// condition holds for (found.Task; the plan's self-review M1). The rising
// edge and the dedupe go by the TASK as well as by the key (org, seat,
// result_missing), so a rise for another TASK is not lost to an earlier
// TASK's ALERT:
//
//   - a cycle in which the condition does not hold clears the key's record,
//     as raiseOrClear does;
//   - while the key is Active for the TASK of the last ALERT
//     (snap.ResultMissingTask), nothing is sent: dedupe;
//   - a rise for that TASK (the key was cleared since, say by a cycle in
//     which the seat worked) is a recurrence of that ALERT's episode when
//     there is one (recordResultMissingRecurrence), as raiseOrClear's
//     recurrence; otherwise it sends a new ALERT;
//   - a rise for another TASK, Active or not, always sends a new ALERT: it
//     is not folded into the pending alert or the inbox item of an earlier
//     TASK, which the leader or the director may have taken for that TASK
//     alone. Its deadman escalation is its own item too: escalateToInbox
//     records no ALERT of result_missing as a recurrence of an earlier item.
//
// A new ALERT is recorded in snap (ResultMissingTask and ResultMissingAlert,
// its alert_id) and fires WatchHooks.OnSemanticTrigger, as raiseOrClear's
// ALERT with semantic true does. When snap names another TASK or none (a
// status file the watch lost, or one written before these two fields
// existed), the TASK's latest ALERT is looked for by the time in its
// alert_id (resultMissingAlertOfTask; the plan's self-review R2-L2) and,
// when there is one, recorded in snap as if the watch had kept it: a rise
// after a lost status file is still a recurrence of that ALERT's episode, not
// a second item. A rise whose recurrence cannot be recorded (the manifest or
// the inbox cannot be read, or the append fails) is a new ALERT, and its
// deadman escalation a new item even beside an open item of the same TASK:
// a second item, not a silent rise.
func (w *watchRun) raiseResultMissing(status *watchStatusFile, orgID, seatID string, snap *watchSeatSnapshot, found resultMissingFinding, now time.Time) {
	key := conditionKey(orgID, seatID, condResultMissing)
	rec := status.Conditions[key]
	if found.Verdict != resultMissingHolds {
		if rec != nil && rec.Active {
			rec.Active = false // recovered: clears, a future re-occurrence is a new rising edge
		}
		return
	}
	taskKey := resultMissingTaskKey(found.Task)
	if snap.ResultMissingTask != taskKey {
		if alertID := w.resultMissingAlertOfTask(status, orgID, seatID, found.Task); alertID != "" {
			snap.ResultMissingTask, snap.ResultMissingAlert = taskKey, alertID
		}
	}
	sameTask := snap.ResultMissingTask == taskKey
	if rec != nil && rec.Active && sameTask {
		return // already alerted for this TASK and still holds: dedupe
	}
	status.Conditions[key] = &watchConditionRecord{Active: true, FirstTS: now.UTC().Format(time.RFC3339)}
	if sameTask && w.recordResultMissingRecurrence(status, orgID, seatID, snap.ResultMissingAlert, now) {
		return
	}
	pending := w.sendAlert(status, orgID, seatID, condResultMissing, found.Message, now)
	snap.ResultMissingTask, snap.ResultMissingAlert = taskKey, pending.AlertID
	if w.hooks.OnSemanticTrigger != nil {
		w.hooks.OnSemanticTrigger(orgID, seatID, condResultMissing, found.Message)
	}
}

// recordResultMissingRecurrence is recordRecurrence (watch_recurrence.go)
// for a rise of result_missing for the TASK of alertID, the seat's last
// ALERT of the condition (snap.ResultMissingAlert), limited to that ALERT's
// own episode, and reports whether it recorded one:
//
//   - the pending alert alertID, while the leader has not acted since it
//     (leaderActedSinceAlert, against the manifest read again here, as
//     foldIntoPendingAlert reads it), gets one more Recurrences; one the
//     leader has acted on is deleted, its episode ended, as checkDeadman
//     would delete it later in the cycle;
//   - otherwise the inbox item whose alert_id is alertID, when it is not
//     resolved, gets a recurred event with a new alert_id of the firing
//     (watchAlertID) and a count of 1, and one stderr line names the item.
//
// It never takes another pending alert or item of the key, which may be of
// another TASK. It reports false, and the caller sends a new ALERT, when
// alertID is "" or has neither, when the manifest (one stderr line) or the
// inbox cannot be read, and when the append fails (one stderr line, unless
// the director resolved the item meanwhile).
func (w *watchRun) recordResultMissingRecurrence(status *watchStatusFile, orgID, seatID, alertID string, now time.Time) bool {
	if alertID == "" {
		return false
	}
	org, seat := PrintableInboxText(orgID, false), PrintableInboxText(seatID, false)
	if pending := status.PendingAlerts[alertID]; pending != nil {
		rr, err := w.org.Manifest.Read()
		if err != nil {
			_, _ = fmt.Fprintf(w.stderr, "watchdog: org=%s seat=%s condition=%s fired again for the TASK of its ALERT %s, which is pending, "+
				"but the manifest could not be read to tell whether the leader acted since (%v); sending a new ALERT\n",
				org, seat, condResultMissing, PrintableInboxText(alertID, false), err)
			return false
		}
		if !w.leaderActedSinceAlert(orgID, pending, rr.Events) {
			pending.Recurrences++
			return true
		}
		delete(status.PendingAlerts, alertID) // the leader answered it: the episode has ended
	}
	if w.org.Inbox == nil {
		return false
	}
	inbox, err := w.org.Inbox.Read()
	if err != nil {
		return false
	}
	i := slices.IndexFunc(inbox.Items, func(item InboxItem) bool {
		return item.AlertID == alertID && item.State != InboxStateResolved
	})
	if i < 0 {
		return false
	}
	item := inbox.Items[i]
	beforeAppendRecurred()
	if err := w.org.Inbox.AppendRecurred(item.ID, watchAlertID(conditionKey(orgID, seatID, condResultMissing), now), "", 1); err != nil {
		if !errors.Is(err, ErrInboxResolved) {
			_, _ = fmt.Fprintf(w.stderr, "watchdog: org=%s seat=%s condition=%s fired again for the TASK of inbox item %s, which is not resolved, "+
				"but the recurrence was not recorded (%v); sending a new ALERT\n", org, seat, condResultMissing, item.ID, err)
		}
		return false
	}
	_, _ = fmt.Fprintf(w.stderr, "watchdog: org=%s seat=%s condition=%s fired again for the same TASK; recorded as a recurrence on inbox item %s (recurrences=%d)\n",
		org, seat, condResultMissing, item.ID, item.Recurrences+1)
	return true
}

// resultMissingAlertOfTask returns the alert_id of the latest ALERT of
// result_missing of seatID of orgID that rose for task, for raiseResultMissing
// when snap.ResultMissingTask names another TASK or none (the plan's
// self-review R2-L2), or "" when there is none. It reads the time in the
// alert_id (escalatedTimestamp) of each pending alert of the key and of each
// of orgID's ALERT items of the key, in any state, and takes the latest whose
// time, to the second, is after task's time. Such an ALERT rose for task: the
// condition holds for a TASK only once stall_minutes have passed since it,
// while an ALERT of an earlier TASK rose in a cycle whose ledger read, made
// after the cycle's time, did not hold task, so its time, to the second, is
// not after task's. An item resolved since is taken too, as snap would name
// it, so the rise is a new ALERT then (recordResultMissingRecurrence). With
// no inbox store, or an inbox that cannot be read, it reads the pending
// alerts only.
func (w *watchRun) resultMissingAlertOfTask(status *watchStatusFile, orgID, seatID string, task resultMissingTask) string {
	key := conditionKey(orgID, seatID, condResultMissing)
	latest := ""
	consider := func(alertID string) {
		at := escalatedTimestamp(alertID)
		if alertIDKey(alertID) != key || !time.Unix(0, at).UTC().Truncate(time.Second).After(task.At) {
			return
		}
		if prev := escalatedTimestamp(latest); latest == "" || at > prev || (at == prev && alertID > latest) {
			latest = alertID
		}
	}
	for alertID, pending := range status.PendingAlerts {
		if pending != nil {
			consider(alertID)
		}
	}
	if w.org.Inbox != nil {
		if inbox, err := w.org.Inbox.Read(); err == nil {
			for _, item := range inbox.Items {
				if item.Type == protocol.TypeAlert && item.OrgID == orgID {
					consider(item.AlertID)
				}
			}
		}
	}
	return latest
}

// waitedFor reports whether wait has passed at now since the later of a and
// b.
func waitedFor(now time.Time, wait time.Duration, a, b time.Time) bool {
	since := a
	if b.After(a) {
		since = b
	}
	return now.Sub(since) >= wait
}

// seatLastTask returns the last TASK the leader sent to seatID of orgID in
// the seat's current life: the real `sent` events to the seat after its
// latest real spawn_started or spawned, so a TASK of an earlier life (before
// a reseat or a disband) does not count. The TASK is read with
// reseatProgressSentTask. ok is false when there is none, when a STOP was
// sent to the seat after it, and when its TASK_ID is unknown (the ledger's
// text was cut there), empty, or holds a control character, or its time
// does not parse: what result_missing cannot name or time is not raised.
func seatLastTask(events []ManifestEvent, orgID, seatID string) (task resultMissingTask, ok bool) {
	start := 0
	for i, ev := range events {
		if !ev.DryRun && ev.OrgID == orgID && ev.SeatID == seatID && (ev.Event == EventSpawnStarted || ev.Event == EventSpawned) {
			start = i + 1
		}
	}
	found := false
	for i := start; i < len(events); i++ {
		ev := events[i]
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID != seatID || ev.Event != EventSent {
			continue
		}
		if taskID, isTask, unknown := reseatProgressSentTask(ev.Details); isTask {
			at, err := time.Parse(time.RFC3339, ev.TS)
			found = true
			ok = !unknown && taskID != "" && !strings.ContainsFunc(taskID, unicode.IsControl) && err == nil
			task = resultMissingTask{TaskID: taskID, TS: ev.TS, At: at.UTC().Truncate(time.Second), Index: i}
			continue
		}
		if m, err := protocol.Parse(sentDetailsText(ev.Details)); found && err == nil && m.Type == protocol.TypeStop {
			ok = false
		}
	}
	return task, found && ok
}

// firstSentAfter returns the time of the first real `sent` event to seatID of
// orgID after events[taskIndex] whose time, to the second, is after at.
// resultMissing passes the time of the seat's QUESTION or BLOCKED plus
// resultMissingSentLag, so the event is the leader's answer to that line,
// typed after it. An event whose time does not parse is passed over. ok is
// false when there is none.
func firstSentAfter(events []ManifestEvent, orgID, seatID string, taskIndex int, at time.Time) (time.Time, bool) {
	for _, ev := range events[taskIndex+1:] {
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID != seatID || ev.Event != EventSent {
			continue
		}
		ts, err := time.Parse(time.RFC3339, ev.TS)
		if err != nil {
			continue
		}
		if ts = ts.UTC().Truncate(time.Second); ts.After(at) {
			return ts, true
		}
	}
	return time.Time{}, false
}

// seatLinesToLeader reads seatID's agmsg history in orgID's team
// (`history.sh <team> <seat> <limit>`, which returns the latest lines the
// seat sent or received, in time order) back to since, and returns the lines
// the seat sent the leader, in that order. Each read runs under its own
// callWithTimeout deadline. A read that does not reach back to since
// (historyReachesBack) is made again with the next limit of
// resultMissingHistoryLimits. ok is false, and the condition is not raised,
// when the agmsg client has no history read, a read fails, a line does not
// parse (parseAgmsgHistory), or the last limit does not reach back either;
// outOfReach is true only in that last case, which a later read cannot
// change (resultMissingHistoryLimits), while the others may pass in a later
// cycle.
func (w *watchRun) seatLinesToLeader(orgID, seatID string, since time.Time) (lines []resultMissingLine, ok, outOfReach bool) {
	probe, isProbe := w.org.Agmsg.(watchAgmsgHistory)
	if !isProbe {
		return nil, false, false
	}
	for _, limit := range resultMissingHistoryLimits {
		var out string
		if err := w.org.callWithTimeout(func(ctx context.Context) error {
			var err error
			out, err = probe.History(ctx, agmsgTeam(orgID), seatID, limit)
			return err
		}); err != nil {
			return nil, false, false
		}
		records, parsed := parseAgmsgHistory(out)
		if !parsed {
			return nil, false, false
		}
		if !historyReachesBack(records, limit, since) {
			continue
		}
		for _, r := range records {
			if r.From == seatID && r.To == LeaderIdentity {
				lines = append(lines, resultMissingLineOf(r))
			}
		}
		return lines, true, false
	}
	return nil, false, true
}

// historyReachesBack reports whether records, read with limit, hold every
// line at or after since: they are fewer than limit (the read returned all
// there is), or their oldest line is before since (to the second), so no line
// at or after since was left out.
//
// "Fewer than limit" trusts history.sh to print every record its read
// returned. It prints none for a record with a field it cannot read (its
// `[ -n "$ts$from$to$body" ] || continue`: in agmsg 1.5.1 one missing field
// makes the whole row empty), so a store that holds such records gives fewer
// lines than limit with older lines left out, and a RESULT among them would
// be missed. agmsg's send.sh writes every field, so the watch takes fewer
// lines as all of them; a rule that also asked for a line before since
// would leave unknown the common case of a seat whose first line came after
// its first TASK.
func historyReachesBack(records []agmsgHistoryRecord, limit int, since time.Time) bool {
	if len(records) < limit {
		return true
	}
	for _, r := range records {
		if r.At.Before(since) {
			return true
		}
	}
	return false
}

// resultMissingLineOf reads the body of r with protocol.Parse: its TYPE and
// TASK_ID, or neither for a body that is not a typed message. Blank lines at
// the start of the body are passed over first, as `"$(cat <file>)"` sends a
// file's leading blank lines and protocol.Parse reads only a first line that
// is `TYPE:`: a seat's RESULT whose file starts with a blank line still
// counts (the plan's self-review L5). protocol.Parse reads the TASK_ID only
// among the header lines, up to the first blank line or line that is not
// `KEY: value` (a `\n` written in a header's value ends them here, as
// historyEscapes says). The seat templates have the file start with the
// `TYPE:` line and the `TASK_ID:` line second, and their history check look
// for both right after `leader: `, so the watch reads the TYPE and the
// TASK_ID of a send the seat counts as made from the lines that check looked
// at (the plan's self-review R2-L4).
func resultMissingLineOf(r agmsgHistoryRecord) resultMissingLine {
	l := resultMissingLine{At: r.At}
	body := r.Body
	for {
		first, rest, found := strings.Cut(body, "\n")
		if !found || strings.TrimSpace(first) != "" {
			break
		}
		body = rest
	}
	if m, err := protocol.Parse(body); err == nil {
		l.Type, l.TaskID = m.Type, m.TaskID
	}
	return l
}

// parseAgmsgHistory reads the output of the agmsg skill's history.sh
// (parseAgmsgHistoryLine for each line). An empty output and agmsgNoHistory
// are no line, and a blank line is passed over. ok is false when any other
// line does not parse, so that a count of lines and their times can be
// trusted.
func parseAgmsgHistory(out string) (records []agmsgHistoryRecord, ok bool) {
	text := strings.TrimSpace(out)
	if text == "" || text == agmsgNoHistory {
		return nil, true
	}
	for line := range strings.SplitSeq(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		r, parsed := parseAgmsgHistoryLine(line)
		if !parsed {
			return nil, false
		}
		records = append(records, r)
	}
	return records, true
}

// parseAgmsgHistoryLine reads one line history.sh prints,
// `  <status> [<ts>] <from> → <to>: <body>` (its `echo` of each row; the
// driver's output is trimmed, so the first line has no leading spaces and a
// last line with an empty body no space after the colon). The ts is the
// message's `at`, which agmsg's storage drivers write in UTC to the second
// (`date -u +%Y-%m-%dT%H:%M:%SZ` in drivers/storage/sqlite.sh and jsonl.sh);
// it is read as RFC 3339 and cut to the second. The status is the text before
// the first `[`, from runs to the first ` → `, and to to the first `:` after
// it. history.sh writes each newline of the body as `\n` and each tab as
// `\t`, which are undone (historyEscapes, which also turns a `\n` written in
// the body into a newline). ok is false for a line without that shape or
// with an empty from or to, or whose ts does not parse.
//
// leaderHistoryFromField (watch.go), the deadman's count of the leader's
// lines, reads the from of the same line on its own: it cuts at the first
// `] ` and needs neither a ts that parses nor a to, so the two do not share
// a parser without changing which lines that count takes. A change to the
// shape of history.sh's line changes both.
func parseAgmsgHistoryLine(line string) (r agmsgHistoryRecord, ok bool) {
	_, rest, found := strings.Cut(line, "[")
	if !found {
		return r, false
	}
	ts, rest, found := strings.Cut(rest, "] ")
	if !found {
		return r, false
	}
	at, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return r, false
	}
	from, rest, found := strings.Cut(rest, " → ")
	if !found {
		return r, false
	}
	to, body, found := strings.Cut(rest, ":")
	from, to = strings.TrimSpace(from), strings.TrimSpace(to)
	if !found || from == "" || to == "" {
		return r, false
	}
	return agmsgHistoryRecord{
		At:   at.UTC().Truncate(time.Second),
		From: from, To: to,
		Body: historyEscapes.Replace(strings.TrimPrefix(body, " ")),
	}, true
}
