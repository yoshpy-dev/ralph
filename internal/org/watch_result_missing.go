package org

import (
	"context"
	"fmt"
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
// cannot read is not raised.

// resultMissingHistoryLimits are the limits of the reads of a seat's agmsg
// history, in order: when a read does not reach back to the TASK's time
// (historyReachesBack), the next limit is tried, and after the last one the
// read is unknown and the condition is not raised.
var resultMissingHistoryLimits = []int{50, 200, 1000}

// agmsgNoHistory is the whole output of the agmsg skill's history.sh for a
// team or an agent that has no message yet.
const agmsgNoHistory = "No message history."

// historyEscapes undoes the escaping history.sh does to a message body to
// keep it on one line: a newline is written as `\n` and a tab as `\t`.
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

// resultMissingTaskKey is the watchSeatSnapshot.ResultSeen value of task:
// the time of its `sent` event and its TASK_ID, so a later TASK with the
// same TASK_ID is a different one.
func resultMissingTaskKey(task resultMissingTask) string {
	return task.TS + " " + task.TaskID
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
// cycle for seatID of orgID, a seat other than the leader, and returns the
// ALERT to send when it does. It holds when all of these do:
//
//   - the seat's current life has a last TASK (seatLastTask) whose TASK_ID is
//     known and that no STOP to the seat followed;
//   - its RESULT was not already found for that TASK (snap.ResultSeen);
//   - the seat has been waiting for input since snap.IdleSince
//     (updateIdleSince), and [org.watchdog].stall_minutes have passed since
//     the later of IdleSince and the start of the clock;
//   - the seat's agmsg history, read back to the TASK's time
//     (seatLinesToLeader), holds no RESULT or REVIEW with that TASK_ID among
//     the lines the seat sent the leader at or after the TASK's time, to the
//     second. One that does is recorded in snap.ResultSeen.
//
// The clock starts at the TASK's time, unless the last line the seat sent the
// leader since then is a QUESTION (with any TASK_ID or none) or a BLOCKED of
// that TASK_ID: the seat then waits for the leader, and the clock starts at
// the first `sent` to the seat after that line's second (firstSentAfter); with
// none, the condition does not hold. A reply the leader sends after a RESULT
// does not start it again, as the RESULT ends the TASK.
//
// The history is read only once the time has passed since the TASK's time,
// the earliest the clock can start, so a seat that cannot be raised costs no
// agmsg call. A history that cannot be read, that does not reach back to the
// TASK's time, or that has a line it cannot parse does not raise the
// condition.
func (w *watchRun) resultMissing(orgID, seatID string, snap *watchSeatSnapshot, now time.Time, events []ManifestEvent) (bool, string) {
	stall := time.Duration(w.cfg.Watchdog.StallMinutes) * time.Minute
	if stall <= 0 || snap.IdleSince == "" {
		return false, ""
	}
	idleSince, err := time.Parse(time.RFC3339, snap.IdleSince)
	if err != nil {
		return false, ""
	}
	task, ok := seatLastTask(events, orgID, seatID)
	if !ok || snap.ResultSeen == resultMissingTaskKey(task) || !waitedFor(now, stall, task.At, idleSince) {
		return false, ""
	}
	lines, ok := w.seatLinesToLeader(orgID, seatID, task.At)
	if !ok {
		return false, ""
	}
	var last *resultMissingLine
	for i, l := range lines {
		if l.At.Before(task.At) {
			continue
		}
		if (l.Type == protocol.TypeResult || l.Type == protocol.TypeReview) && l.TaskID == task.TaskID {
			snap.ResultSeen = resultMissingTaskKey(task)
			return false, ""
		}
		last = &lines[i]
	}
	start := task.At
	if last != nil && (last.Type == protocol.TypeQuestion || (last.Type == protocol.TypeBlocked && last.TaskID == task.TaskID)) {
		answered, ok := firstSentAfter(events, orgID, seatID, task.Index, last.At)
		if !ok {
			return false, ""
		}
		start = answered
	}
	if !waitedFor(now, stall, start, idleSince) {
		return false, ""
	}
	return true, resultMissingAlertMessage(orgID, seatID, task.TaskID, w.cfg.Watchdog.StallMinutes)
}

// resultMissingAlertMessage is the ALERT result_missing sends the leader. It
// does not say the seat failed: it says what the watch saw and asks the
// leader to read the seat's screen.
func resultMissingAlertMessage(orgID, seatID, taskID string, stallMinutes int) string {
	return fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\nTASK_ID: %s\n\n"+
		"seat %s received TASK %s and has been waiting for input for %dm or more without sending the leader a RESULT or BLOCKED for it. "+
		"Read its screen with `ralph org read` (a SEND FAILED: line, or a result it wrote but did not send); if it is waiting for a background command, wait.",
		orgID, seatID, condResultMissing, taskID, seatID, taskID, stallMinutes)
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
// orgID after events[taskIndex] whose time, to the second, is after at: the
// leader's answer to the QUESTION or BLOCKED the seat sent at at. An event
// whose time does not parse is passed over. ok is false when there is none.
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
// parse (parseAgmsgHistory), or the last limit does not reach back either.
func (w *watchRun) seatLinesToLeader(orgID, seatID string, since time.Time) (lines []resultMissingLine, ok bool) {
	probe, isProbe := w.org.Agmsg.(watchAgmsgHistory)
	if !isProbe {
		return nil, false
	}
	for _, limit := range resultMissingHistoryLimits {
		var out string
		if err := w.org.callWithTimeout(func(ctx context.Context) error {
			var err error
			out, err = probe.History(ctx, agmsgTeam(orgID), seatID, limit)
			return err
		}); err != nil {
			return nil, false
		}
		records, parsed := parseAgmsgHistory(out)
		if !parsed {
			return nil, false
		}
		if !historyReachesBack(records, limit, since) {
			continue
		}
		for _, r := range records {
			if r.From == seatID && r.To == LeaderIdentity {
				lines = append(lines, resultMissingLineOf(r))
			}
		}
		return lines, true
	}
	return nil, false
}

// historyReachesBack reports whether records, read with limit, hold every
// line at or after since: they are fewer than limit (the read returned all
// there is), or their oldest line is before since (to the second), so no line
// at or after since was left out.
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
// TASK_ID, or neither for a body that is not a typed message.
func resultMissingLineOf(r agmsgHistoryRecord) resultMissingLine {
	l := resultMissingLine{At: r.At}
	if m, err := protocol.Parse(r.Body); err == nil {
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
// `\t`, which are undone. ok is false for a line without that shape or with
// an empty from or to, or whose ts does not parse.
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
