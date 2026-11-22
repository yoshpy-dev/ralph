package org

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// resultMissingTestSeat is the seat other than the leader that the
// result_missing tests send TASKs to.
const resultMissingTestSeat = "implementer"

// resultMissingRead is one History call with an agent: the agent and the
// limit asked for.
type resultMissingRead struct {
	agent string
	limit int
}

// resultMissingAgmsg is fakeWatchAgmsg with a history that answers a read
// with an agent the way the agmsg skill's history.sh does: the latest limit
// records the agent sent or received, in time order, one line each
// (`  ○ [<ts>] <from> → <to>: <body>`, the body's newlines and tabs escaped),
// "No message history." for none, and the output trimmed as the driver trims
// it. A read with no agent (the deadman's) is fakeWatchAgmsg's. err, when
// set, fails every read with an agent; raw, when set, is the output of every
// such read.
type resultMissingAgmsg struct {
	*fakeWatchAgmsg

	histMu  sync.Mutex
	records []agmsgHistoryRecord
	raw     *string
	err     error
	reads   []resultMissingRead
}

func (f *resultMissingAgmsg) History(ctx context.Context, team, agentID string, limit int) (string, error) {
	if agentID == "" {
		return f.fakeWatchAgmsg.History(ctx, team, agentID, limit)
	}
	f.histMu.Lock()
	defer f.histMu.Unlock()
	f.reads = append(f.reads, resultMissingRead{agent: agentID, limit: limit})
	if f.err != nil {
		return "", f.err
	}
	if f.raw != nil {
		return *f.raw, nil
	}
	var mine []agmsgHistoryRecord
	for _, r := range f.records {
		if r.From == agentID || r.To == agentID {
			mine = append(mine, r)
		}
	}
	slices.SortStableFunc(mine, func(a, b agmsgHistoryRecord) int { return a.At.Compare(b.At) })
	if limit > 0 && len(mine) > limit {
		mine = mine[len(mine)-limit:]
	}
	if len(mine) == 0 {
		return agmsgNoHistory, nil
	}
	escape := strings.NewReplacer("\n", `\n`, "\t", `\t`)
	lines := make([]string, len(mine))
	for i, r := range mine {
		lines[i] = fmt.Sprintf("  ○ [%s] %s → %s: %s", r.At.UTC().Format(time.RFC3339), r.From, r.To, escape.Replace(r.Body))
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

// resultMissingFixture is org-a with the leader and resultMissingTestSeat
// spawned, stall_minutes 5, deadman_minutes 10, and an agmsg whose history
// is resultMissingAgmsg's. Each cycle answers the leader's agent get with a
// text that changes every cycle (leaderReply, when set, in every cycle
// instead), so neither the leader's stall nor the deadman's probe of it is
// in the way unless a test asks for it.
type resultMissingFixture struct {
	o      *Org
	h      *fakeWatchHerdr
	a      *fakeWatchAgmsg
	hist   *resultMissingAgmsg
	clk    *fakeClock
	run    *watchRun
	status *watchStatusFile
	stderr bytes.Buffer
	cycles int

	leaderReply string
}

func newResultMissingFixture(t *testing.T) *resultMissingFixture {
	t.Helper()
	o, h, a, clk := testWatchOrg(t)
	o.Config.Watchdog.StallMinutes = 5
	hist := &resultMissingAgmsg{fakeWatchAgmsg: a}
	o.Agmsg = hist
	for _, seat := range []string{LeaderIdentity, resultMissingTestSeat} {
		if r := o.Spawn(watchSpawnParams("org-a", seat, seat)); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn %s failed: %+v", seat, r)
		}
	}
	dir := t.TempDir()
	f := &resultMissingFixture{o: o, h: h, a: a, hist: hist, clk: clk, status: &watchStatusFile{OrgID: "org-a"}}
	f.run = &watchRun{
		org: o, cfg: o.Config,
		gitStatus:       func(string) (string, error) { return "", nil },
		escalateFn:      func(context.Context, string) error { return nil },
		stderr:          &f.stderr,
		statusPath:      filepath.Join(dir, WatchStatusFileName("org-a")),
		escalationsPath: filepath.Join(dir, EscalationsRelName),
	}
	return f
}

// sentDetails records a `sent` event to seat at the clock's time with
// details as its Details.
func (f *resultMissingFixture) sentDetails(t *testing.T, seat, details string, dryRun bool) {
	t.Helper()
	if err := f.o.Manifest.Append(ManifestEvent{
		TS: f.clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: seat, Event: EventSent,
		Details: details, DryRun: dryRun,
	}); err != nil {
		t.Fatal(err)
	}
}

// sent records the `sent` event of a real send of text to seat, its Details
// cut as send cuts them.
func (f *resultMissingFixture) sent(t *testing.T, seat, text string) {
	t.Helper()
	f.sentDetails(t, seat, truncateForDetails(text), false)
}

// task records a TASK with taskID sent to resultMissingTestSeat.
func (f *resultMissingFixture) task(t *testing.T, taskID string) {
	t.Helper()
	f.sent(t, resultMissingTestSeat, "TYPE: TASK\nTASK_ID: "+taskID+"\n\nimplement the slice")
}

// lineAt adds a line from from to to with body at at to the agmsg history.
func (f *resultMissingFixture) lineAt(at time.Time, from, to, body string) {
	f.hist.histMu.Lock()
	defer f.hist.histMu.Unlock()
	f.hist.records = append(f.hist.records, agmsgHistoryRecord{At: at, From: from, To: to, Body: body})
}

// seatSays adds a line resultMissingTestSeat sent the leader at the clock's
// time.
func (f *resultMissingFixture) seatSays(body string) {
	f.lineAt(f.clk.Now(), resultMissingTestSeat, LeaderIdentity, body)
}

// advanceTo moves the clock to at.
func (f *resultMissingFixture) advanceTo(at time.Time) {
	f.clk.Advance(at.Sub(f.clk.Now()))
}

// idle and working are agent get replies of resultMissingTestSeat that differ
// from every other cycle's, so the seat's stall does not hold.
func (f *resultMissingFixture) idle() string    { return watchAgentReply("idle", f.cycles+1) }
func (f *resultMissingFixture) working() string { return watchAgentReply("working", f.cycles+1) }

// cycle runs one cycle of org-a at the clock's time, with the agent get of
// resultMissingTestSeat answering seatReply.
func (f *resultMissingFixture) cycle(t *testing.T, seatReply string) {
	t.Helper()
	f.cycles++
	leader := f.leaderReply
	if leader == "" {
		leader = fmt.Sprintf("leader-%d", f.cycles)
	}
	f.h.mu.Lock()
	f.h.AgentGetSeq[herdrAgentName("org-a", LeaderIdentity)] = []string{leader}
	f.h.AgentGetSeq[herdrAgentName("org-a", resultMissingTestSeat)] = []string{seatReply}
	f.h.mu.Unlock()
	if err := f.run.evaluateCycle("org-a", f.status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}
}

// alerts returns the messages of the watch's ALERTs to the leader about seat
// whose CONDITION is cond.
func (f *resultMissingFixture) alerts(seat, cond string) []string {
	var out []string
	for _, c := range watchdogAlerts(f.a) {
		if strings.Contains(c.message, "\nSEAT: "+seat+"\nCONDITION: "+cond+"\n") {
			out = append(out, c.message)
		}
	}
	return out
}

// active reports whether cond of seat is recorded as active.
func (f *resultMissingFixture) active(seat, cond string) bool {
	rec := f.status.Conditions[conditionKey("org-a", seat, cond)]
	return rec != nil && rec.Active
}

// reads returns the History calls with an agent so far.
func (f *resultMissingFixture) reads() []resultMissingRead {
	f.hist.histMu.Lock()
	defer f.hist.histMu.Unlock()
	return slices.Clone(f.hist.reads)
}

// assertResultMissingAlert fails the test unless msg is a typed ALERT that
// passes the protocol check, with the headers ORG_ID org-a, SEAT
// resultMissingTestSeat, CONDITION result_missing, and TASK_ID taskID, and a
// body that says what the watch saw and asks the leader to read the seat's
// screen, without saying the seat failed.
func assertResultMissingAlert(t *testing.T, msg, taskID string) {
	t.Helper()
	m, err := protocol.Parse(msg)
	if err == nil {
		err = protocol.Validate(m, protocol.DefaultMaxBodyChars)
	}
	if err != nil {
		t.Fatalf("ALERT %q fails the protocol check: %v", msg, err)
	}
	want := map[string]string{"ORG_ID": "org-a", "SEAT": resultMissingTestSeat, "CONDITION": condResultMissing}
	if m.Type != protocol.TypeAlert || m.TaskID != taskID || !reflect.DeepEqual(m.Fields, want) {
		t.Errorf("ALERT parses as TYPE %q, TASK_ID %q, headers %v; want ALERT, %q, %v", m.Type, m.TaskID, m.Fields, taskID, want)
	}
	head := fmt.Sprintf("seat %s received TASK %s and has been waiting for input for 5m or more without sending the leader a RESULT or BLOCKED for it. ", resultMissingTestSeat, taskID)
	if !strings.HasPrefix(m.Body, head) || !strings.Contains(m.Body, "`ralph org read`") || !strings.Contains(m.Body, "if it is waiting for a background command, wait.") {
		t.Errorf("ALERT body %q; want it to start %q, name `ralph org read`, and say to wait for a background command", m.Body, head)
	}
}

// AC3 and AC4: a seat that received a TASK and has been waiting for input
// for stall_minutes without a line to the leader gets one result_missing
// ALERT naming it and the TASK_ID, in the first cycle the time has passed
// and not one second before, and the agmsg history is read only then, as
// the seat, with the first limit. While the condition holds no second ALERT
// is sent. Once the seat's RESULT of the TASK is in the history, the
// condition clears, the TASK is remembered, and later cycles read no
// history for it.
func TestWatch_ResultMissing_RaisesForAnIdleSeatWithoutTheResultAndClearsOnIt(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.task(t, "t1")
	f.cycle(t, f.idle())
	f.advanceTo(t0.Add(5*time.Minute - time.Second))
	f.cycle(t, f.idle())
	if got := f.alerts(resultMissingTestSeat, condResultMissing); len(got) != 0 || f.active(resultMissingTestSeat, condResultMissing) {
		t.Fatalf("ALERTs %q before stall_minutes passed; want none", got)
	}
	if reads := f.reads(); len(reads) != 0 {
		t.Fatalf("history reads %v before stall_minutes passed; want none", reads)
	}

	f.advanceTo(t0.Add(5 * time.Minute))
	f.cycle(t, f.idle())
	got := f.alerts(resultMissingTestSeat, condResultMissing)
	if len(got) != 1 || !f.active(resultMissingTestSeat, condResultMissing) {
		t.Fatalf("ALERTs %q, active %t once stall_minutes passed; want one, active", got, f.active(resultMissingTestSeat, condResultMissing))
	}
	assertResultMissingAlert(t, got[0], "t1")
	if reads := f.reads(); !reflect.DeepEqual(reads, []resultMissingRead{{agent: resultMissingTestSeat, limit: 50}}) {
		t.Errorf("history reads %v; want one of the seat with limit 50", reads)
	}
	if n := len(f.alerts(resultMissingTestSeat, condStall)); n != 0 {
		t.Errorf("%d stall ALERTs about the seat; want none", n)
	}

	f.advanceTo(t0.Add(6 * time.Minute))
	f.cycle(t, f.idle())
	if n := len(f.alerts(resultMissingTestSeat, condResultMissing)); n != 1 {
		t.Fatalf("%d ALERTs while the condition still holds; want the one", n)
	}

	f.seatSays("TYPE: RESULT\nTASK_ID: t1\n\ndone: abc1234")
	f.advanceTo(t0.Add(7 * time.Minute))
	f.cycle(t, f.idle())
	if f.active(resultMissingTestSeat, condResultMissing) {
		t.Errorf("result_missing still active after the seat's RESULT; want it cleared")
	}
	wantSeen := t0.UTC().Format(time.RFC3339) + " t1"
	if got := f.status.SeatSnapshots[resultMissingTestSeat].ResultSeen; got != wantSeen {
		t.Errorf("ResultSeen = %q; want %q", got, wantSeen)
	}
	readsAfterResult := len(f.reads())
	for m := 8; m <= 12; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.idle())
	}
	if n := len(f.reads()); n != readsAfterResult {
		t.Errorf("history read %d more times after the RESULT was found; want none", n-readsAfterResult)
	}
	if n := len(f.alerts(resultMissingTestSeat, condResultMissing)); n != 1 {
		t.Errorf("%d result_missing ALERTs in all; want the one", n)
	}
}

// AC4 and AC6 (f), (i): the condition is not raised, cycle after cycle, when
// the seat sent the RESULT or REVIEW of the TASK (in the TASK's own second
// too), when the seat is working or herdr's reply does not show it waiting,
// when the history cannot be read or has a line it cannot parse, when there
// is no TASK the watch can name (none, a dry run, a TASK_ID the ledger's cut
// lost, a raw TASK with no TASK_ID, a TASK of the seat's earlier life), when
// a STOP followed the TASK, and when stall_minutes is 0. Where the watch has
// nothing to read the history for, it reads none.
func TestWatch_ResultMissing_IsNotRaisedWhenTheSeatAnsweredOrItCannotBeTold(t *testing.T) {
	longHeader := "TYPE: TASK\nNOTE: " + strings.Repeat("x", 220) + "\nTASK_ID: t1\n\nwork"
	for _, tc := range []struct {
		name       string
		noTask     bool
		setup      func(t *testing.T, f *resultMissingFixture, t0 time.Time)
		reply      func(f *resultMissingFixture) string
		wantNoRead bool
	}{
		{name: "the seat sent the RESULT of the TASK", setup: func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0.Add(time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: RESULT\nTASK_ID: t1\n\ndone")
		}},
		{name: "the seat sent a REVIEW of the TASK", setup: func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0.Add(time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: REVIEW\nTASK_ID: t1\n\nno findings")
		}},
		{name: "the RESULT came in the TASK's own second", setup: func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0, resultMissingTestSeat, LeaderIdentity, "TYPE: RESULT\nTASK_ID: t1\n\ndone")
		}},
		{name: "a RESULT whose body has escaped newlines and tabs", setup: func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0.Add(time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: RESULT\nTASK_ID: t1\n\n- commit:\tabc1234\n- report: docs/reports/x.md")
		}},
		{name: "a RESULT the leader answered with a DECISION", setup: func(t *testing.T, f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0.Add(time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: RESULT\nTASK_ID: t1\n\ndone")
			f.lineAt(t0.Add(2*time.Minute), LeaderIdentity, resultMissingTestSeat, "TYPE: DECISION\n\nthanks, wait for the next TASK")
			f.sent(t, resultMissingTestSeat, "TYPE: DECISION\n\nthanks, wait for the next TASK")
		}},
		{name: "the seat is working", reply: (*resultMissingFixture).working, wantNoRead: true},
		{name: "herdr's reply is not its JSON", reply: func(*resultMissingFixture) string { return "ok" }, wantNoRead: true},
		{name: "the history cannot be read", setup: func(_ *testing.T, f *resultMissingFixture, _ time.Time) {
			f.hist.err = errors.New("history.sh: exit status 1")
		}},
		{name: "a line of the history is cut", setup: func(_ *testing.T, f *resultMissingFixture, _ time.Time) {
			raw := "○ [2026-01-01T00:00:30Z] leader → implementer: TYPE: DECISION\n  ○ [2026-01-01T00:0"
			f.hist.raw = &raw
		}},
		{name: "a line of the history has a time that does not parse", setup: func(_ *testing.T, f *resultMissingFixture, _ time.Time) {
			raw := "○ [yesterday] leader → implementer: TYPE: DECISION"
			f.hist.raw = &raw
		}},
		{name: "no TASK was sent", noTask: true, wantNoRead: true},
		{name: "the TASK was a dry run", noTask: true, wantNoRead: true, setup: func(t *testing.T, f *resultMissingFixture, _ time.Time) {
			f.sentDetails(t, resultMissingTestSeat, "TYPE: TASK\nTASK_ID: t1\n\nwork", true)
		}},
		{name: "the ledger's cut lost the TASK_ID", noTask: true, wantNoRead: true, setup: func(t *testing.T, f *resultMissingFixture, _ time.Time) {
			f.sent(t, resultMissingTestSeat, longHeader)
		}},
		{name: "a raw TASK with no TASK_ID", noTask: true, wantNoRead: true, setup: func(t *testing.T, f *resultMissingFixture, _ time.Time) {
			f.sentDetails(t, resultMissingTestSeat, "raw=true TYPE: TASK\n\nwork with no id", false)
		}},
		{name: "a STOP followed the TASK", wantNoRead: true, setup: func(t *testing.T, f *resultMissingFixture, _ time.Time) {
			f.sent(t, resultMissingTestSeat, "TYPE: STOP\n\nstop here")
		}},
		{name: "the TASK was sent in the seat's earlier life", wantNoRead: true, setup: func(t *testing.T, f *resultMissingFixture, _ time.Time) {
			if r := f.o.Stop(StopParams{OrgID: "org-a", Seat: resultMissingTestSeat}); r.Err != nil {
				t.Fatalf("stop: %v", r.Err)
			}
			if r := f.o.Spawn(watchSpawnParams("org-a", resultMissingTestSeat, resultMissingTestSeat)); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn again failed: %+v", r)
			}
		}},
		{name: "stall_minutes is 0", wantNoRead: true, setup: func(_ *testing.T, f *resultMissingFixture, _ time.Time) {
			f.run.cfg.Watchdog.StallMinutes = 0
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newResultMissingFixture(t)
			t0 := f.clk.Now()
			if !tc.noTask {
				f.task(t, "t1")
			}
			if tc.setup != nil {
				tc.setup(t, f, t0)
			}
			reply := tc.reply
			if reply == nil {
				reply = (*resultMissingFixture).idle
			}
			for m := 0; m <= 12; m++ {
				f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
				f.cycle(t, reply(f))
			}
			if got := f.alerts(resultMissingTestSeat, condResultMissing); len(got) != 0 || f.active(resultMissingTestSeat, condResultMissing) {
				t.Errorf("ALERTs %q, active %t; want none", got, f.active(resultMissingTestSeat, condResultMissing))
			}
			if reads := f.reads(); tc.wantNoRead && len(reads) != 0 {
				t.Errorf("history reads %v; want none", reads)
			}
		})
	}
}

// AC3 and AC6 (a), (b), (c), (e), and the edge cases of the plan's test
// plan: when the condition is first raised, for lines the seat sent and
// received after a TASK sent at t0. The seat works at t0 and waits for
// input from t0+1m on, so with no line from it the clock starts at the TASK
// and the condition is raised at t0+6m (c). Lines the seat only received
// (e), a RESULT of another TASK, an old RESULT of the same TASK_ID from
// before the TASK, and a RESULT with the TASK_ID between the seat and
// another seat do not count. After a QUESTION (with or without the TASK_ID)
// or a BLOCKED of the TASK, the clock starts at the leader's first send to
// the seat after it, at t0+3m, so the condition is raised at t0+8m (a);
// with no such send it is never raised (b), and a send before the QUESTION
// or in its second is not one. A BLOCKED of another TASK does not wait for
// the leader.
func TestWatch_ResultMissing_TheClockStartsAtTheTaskOrAtTheLeadersAnswer(t *testing.T) {
	seatSaysAt := func(d time.Duration, body string) func(*testing.T, *resultMissingFixture, time.Time) {
		return func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0.Add(d), resultMissingTestSeat, LeaderIdentity, body)
		}
	}
	leaderSendsAt := func(d time.Duration) func(*testing.T, *resultMissingFixture, time.Time) {
		return func(t *testing.T, f *resultMissingFixture, t0 time.Time) {
			now := f.clk.Now()
			f.advanceTo(t0.Add(d))
			f.sent(t, resultMissingTestSeat, "TYPE: DECISION\n\nuse the plan's wording")
			f.lineAt(t0.Add(d), LeaderIdentity, resultMissingTestSeat, "TYPE: DECISION\n\nuse the plan's wording")
			f.advanceTo(now)
		}
	}
	const never = 0
	for _, tc := range []struct {
		name    string
		steps   []func(*testing.T, *resultMissingFixture, time.Time)
		raiseAt time.Duration
	}{
		{name: "no line from the seat (c)", raiseAt: 6 * time.Minute},
		{name: "only lines the seat received (e)", raiseAt: 6 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
				f.lineAt(t0, LeaderIdentity, resultMissingTestSeat, "TYPE: TASK\nTASK_ID: t1\n\nimplement the slice")
				f.lineAt(t0.Add(time.Minute), LeaderIdentity, resultMissingTestSeat, "TYPE: DECISION\nTASK_ID: t1\n\nkeep going")
			},
		}},
		{name: "a RESULT of another TASK", raiseAt: 6 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(time.Minute, "TYPE: RESULT\nTASK_ID: t0\n\nlate result of the previous TASK"),
		}},
		{name: "an old RESULT of the same TASK_ID from before the TASK", raiseAt: 6 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(-time.Second, "TYPE: RESULT\nTASK_ID: t1\n\nold result"),
		}},
		{name: "a RESULT with the TASK_ID the seat sent another seat", raiseAt: 6 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
				f.lineAt(t0.Add(time.Minute), resultMissingTestSeat, "reviewer", "TYPE: RESULT\nTASK_ID: t1\n\nwrong address")
			},
		}},
		{name: "a RESULT with the TASK_ID another seat sent the seat", raiseAt: 6 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
				f.lineAt(t0.Add(time.Minute), "reviewer", resultMissingTestSeat, "TYPE: RESULT\nTASK_ID: t1\n\nnot the seat's")
			},
		}},
		{name: "a QUESTION with the TASK_ID the leader answered (a)", raiseAt: 8 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(time.Minute, "TYPE: QUESTION\nTASK_ID: t1\n\nwhich wording?"), leaderSendsAt(3 * time.Minute),
		}},
		{name: "a QUESTION with no TASK_ID the leader answered (a)", raiseAt: 8 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(time.Minute, "TYPE: QUESTION\n\nwhich wording?"), leaderSendsAt(3 * time.Minute),
		}},
		{name: "a BLOCKED of the TASK the leader answered (a)", raiseAt: 8 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(time.Minute, "TYPE: BLOCKED\nTASK_ID: t1\n\nthe plan is not approved"), leaderSendsAt(3 * time.Minute),
		}},
		{name: "a QUESTION the leader has not answered (b)", raiseAt: never, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(time.Minute, "TYPE: QUESTION\nTASK_ID: t1\n\nwhich wording?"),
		}},
		{name: "a BLOCKED of the TASK the leader has not answered (b)", raiseAt: never, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(time.Minute, "TYPE: BLOCKED\nTASK_ID: t1\n\nthe plan is not approved"),
		}},
		{name: "a QUESTION after the leader's last send (b)", raiseAt: never, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			leaderSendsAt(time.Minute), seatSaysAt(2*time.Minute, "TYPE: QUESTION\n\nwhich wording?"),
		}},
		{name: "a QUESTION in the second of the leader's last send (b)", raiseAt: never, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			leaderSendsAt(time.Minute), seatSaysAt(time.Minute, "TYPE: QUESTION\n\nwhich wording?"),
		}},
		{name: "a BLOCKED of another TASK", raiseAt: 6 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			seatSaysAt(time.Minute, "TYPE: BLOCKED\nTASK_ID: t0\n\nthe previous TASK is stuck"),
		}},
		{name: "a STOP before a later TASK", raiseAt: 7 * time.Minute, steps: []func(*testing.T, *resultMissingFixture, time.Time){
			func(t *testing.T, f *resultMissingFixture, t0 time.Time) {
				f.sent(t, resultMissingTestSeat, "TYPE: STOP\n\nstop t1")
				f.advanceTo(t0.Add(2 * time.Minute))
				f.task(t, "t2")
				f.advanceTo(t0)
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newResultMissingFixture(t)
			t0 := f.clk.Now()
			f.task(t, "t1")
			for _, step := range tc.steps {
				step(t, f, t0)
			}
			f.cycle(t, f.working())
			var raisedAt time.Duration
			for m := 1; m <= 15 && raisedAt == never; m++ {
				f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
				f.cycle(t, f.idle())
				if len(f.alerts(resultMissingTestSeat, condResultMissing)) > 0 {
					raisedAt = time.Duration(m) * time.Minute
				}
			}
			if raisedAt != tc.raiseAt {
				t.Fatalf("raised at t0+%v; want t0+%v (0 is never)", raisedAt, tc.raiseAt)
			}
			if tc.raiseAt != never {
				wantID := "t1"
				if strings.Contains(tc.name, "later TASK") {
					wantID = "t2"
				}
				assertResultMissingAlert(t, f.alerts(resultMissingTestSeat, condResultMissing)[0], wantID)
			}
		})
	}
}

// AC6 (h): a seat that worked for ten minutes after its TASK and then waits
// for input is raised only once it has waited stall_minutes, not in the
// first cycle the TASK is that old.
func TestWatch_ResultMissing_WaitsForTheSeatToStayWaitingForStallMinutes(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.task(t, "t1")
	for m := 0; m < 10; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.working())
	}
	for m := 10; m < 15; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.idle())
		if got := f.alerts(resultMissingTestSeat, condResultMissing); len(got) != 0 {
			t.Fatalf("raised at t0+%dm, %dm after the seat stopped working; want stall_minutes (5m) first", m, m-10)
		}
	}
	f.advanceTo(t0.Add(15 * time.Minute))
	f.cycle(t, f.idle())
	if got := f.alerts(resultMissingTestSeat, condResultMissing); len(got) != 1 {
		t.Fatalf("ALERTs %q 5m after the seat stopped working; want one", got)
	}

	// A cycle that reports the seat working again clears the condition and
	// the time it started waiting.
	f.advanceTo(t0.Add(16 * time.Minute))
	f.cycle(t, f.working())
	if f.active(resultMissingTestSeat, condResultMissing) || f.status.SeatSnapshots[resultMissingTestSeat].IdleSince != "" {
		t.Errorf("active %t, IdleSince %q after the seat worked again; want cleared", f.active(resultMissingTestSeat, condResultMissing), f.status.SeatSnapshots[resultMissingTestSeat].IdleSince)
	}
}

// AC6 (d): a RESULT the first limit leaves out (the latest 50 lines all
// came after the TASK) is found by the read with the next limit, and once
// found the TASK is remembered, so the condition is not raised even when
// the RESULT later falls out of every limit, and no history is read for it
// again.
func TestWatch_ResultMissing_ReadsFurtherBackForAResultTheFirstLimitLeftOut(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.task(t, "t1")
	f.lineAt(t0.Add(time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: RESULT\nTASK_ID: t1\n\ndone")
	for i := range 60 {
		f.lineAt(t0.Add(2*time.Minute), LeaderIdentity, resultMissingTestSeat, fmt.Sprintf("TYPE: HEARTBEAT\n\n%d", i))
	}
	f.cycle(t, f.working())
	for m := 1; m <= 6; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.idle())
	}
	want := []resultMissingRead{{agent: resultMissingTestSeat, limit: 50}, {agent: resultMissingTestSeat, limit: 200}}
	if reads := f.reads(); !reflect.DeepEqual(reads, want) {
		t.Fatalf("history reads %v; want %v", reads, want)
	}
	if got := f.alerts(resultMissingTestSeat, condResultMissing); len(got) != 0 {
		t.Fatalf("ALERTs %q; want none, the RESULT is in the second read", got)
	}

	for i := range 2000 {
		f.lineAt(t0.Add(7*time.Minute), LeaderIdentity, resultMissingTestSeat, fmt.Sprintf("TYPE: HEARTBEAT\n\nlater %d", i))
	}
	for m := 7; m <= 12; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.idle())
	}
	if got := f.alerts(resultMissingTestSeat, condResultMissing); len(got) != 0 {
		t.Errorf("ALERTs %q after the RESULT fell out of every limit; want none, it was found before", got)
	}
	if reads := f.reads(); len(reads) != len(want) {
		t.Errorf("history reads %v; want no read after the RESULT was found", reads[len(want):])
	}
}

// AC6 (d): when even the last limit does not reach back to the TASK (the
// latest 1000 lines all came after it), the history cannot tell, and the
// condition is not raised; the reads are made with each limit in turn. With
// one line fewer, the last read reaches the seat's line from before the
// TASK, and the condition is raised.
func TestWatch_ResultMissing_IsNotRaisedWhenTheHistoryDoesNotReachBackToTheTask(t *testing.T) {
	for _, tc := range []struct {
		lines int
		raise bool
	}{{lines: 1000, raise: false}, {lines: 999, raise: true}} {
		t.Run(fmt.Sprintf("%d lines after the TASK", tc.lines), func(t *testing.T) {
			f := newResultMissingFixture(t)
			t0 := f.clk.Now()
			f.lineAt(t0.Add(-time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: HELLO\n\nready")
			f.task(t, "t1")
			for i := range tc.lines {
				f.lineAt(t0.Add(time.Minute), LeaderIdentity, resultMissingTestSeat, fmt.Sprintf("TYPE: HEARTBEAT\n\n%d", i))
			}
			f.cycle(t, f.working())
			for m := 1; m <= 6; m++ {
				f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
				f.cycle(t, f.idle())
			}
			want := []resultMissingRead{{resultMissingTestSeat, 50}, {resultMissingTestSeat, 200}, {resultMissingTestSeat, 1000}}
			if reads := f.reads(); !reflect.DeepEqual(reads, want) {
				t.Errorf("history reads %v; want %v", reads, want)
			}
			if raised := len(f.alerts(resultMissingTestSeat, condResultMissing)) == 1; raised != tc.raise {
				t.Errorf("raised %t; want %t", raised, tc.raise)
			}
		})
	}
}

// AC4: while result_missing holds for a seat, its stall is not raised, even
// though its agent get answers the same text and its last event is older
// than stall_minutes; once the seat's RESULT clears result_missing, the
// stall is raised as before.
func TestWatch_ResultMissing_TheSeatGetsNoStallWhileItHolds(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.task(t, "t1")
	frozen := watchAgentReply("idle", 0)
	for m := 0; m <= 10; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, frozen)
	}
	if n := len(f.alerts(resultMissingTestSeat, condResultMissing)); n != 1 {
		t.Fatalf("%d result_missing ALERTs; want one", n)
	}
	if n := len(f.alerts(resultMissingTestSeat, condStall)); n != 0 || f.active(resultMissingTestSeat, condStall) {
		t.Fatalf("%d stall ALERTs, stall active %t while result_missing holds; want none", n, f.active(resultMissingTestSeat, condStall))
	}

	f.seatSays("TYPE: RESULT\nTASK_ID: t1\n\ndone")
	f.advanceTo(t0.Add(11 * time.Minute))
	f.cycle(t, frozen)
	if f.active(resultMissingTestSeat, condResultMissing) {
		t.Errorf("result_missing still active after the RESULT")
	}
	if n := len(f.alerts(resultMissingTestSeat, condStall)); n != 1 {
		t.Errorf("%d stall ALERTs once result_missing cleared; want the stall raised as before", n)
	}
}

// alertMinutes runs one cycle a minute from t0+from to t0+to (minutes), the
// agent get of resultMissingTestSeat answering reply(m) in the cycle at
// t0+m, and returns, by condition (stall and result_missing), the minutes of
// the cycles that sent an ALERT about the seat.
func (f *resultMissingFixture) alertMinutes(t *testing.T, t0 time.Time, from, to int, reply func(m int) string) map[string][]int {
	t.Helper()
	conds := []string{condStall, condResultMissing}
	sent := map[string][]int{}
	for m := from; m <= to; m++ {
		before := map[string]int{}
		for _, cond := range conds {
			before[cond] = len(f.alerts(resultMissingTestSeat, cond))
		}
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, reply(m))
		for _, cond := range conds {
			if len(f.alerts(resultMissingTestSeat, cond)) > before[cond] {
				sent[cond] = append(sent[cond], m)
			}
		}
	}
	return sent
}

// worksThenWaits is the agent get reply of resultMissingTestSeat in the
// cycle at t0+m for a seat that works on its TASK for ten minutes, longer
// than stall_minutes (5), and then waits for input: a reply that changes
// every cycle before t0+10m, and the same one in every cycle from t0+10m on.
// With its TASK at t0 as its last event, the seat's stall holds by the
// stall's own rule from t0+11m on, and its result_missing clock runs out at
// t0+15m.
func (f *resultMissingFixture) worksThenWaits(m int) string {
	if m < 10 {
		return f.working()
	}
	return watchAgentReply("idle", 0)
}

// AC4: a seat that works on its TASK longer than stall_minutes and then
// waits for input without sending the leader its RESULT gets no stall while
// its result_missing clock runs (from t0+11m, where the stall's own rule
// holds, to t0+14m), and no history is read for it then. At t0+15m it gets
// the one result_missing ALERT, and no stall after it: one ALERT to the
// leader, not two.
func TestWatch_ResultMissing_HoldsTheSeatsStallWhileItsClockRuns(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.task(t, "t1")
	if got := f.alertMinutes(t, t0, 0, 14, f.worksThenWaits); len(got) != 0 || f.active(resultMissingTestSeat, condStall) {
		t.Fatalf("ALERTs about the seat at minutes %v, stall active %t while its result_missing clock runs; want none", got, f.active(resultMissingTestSeat, condStall))
	}
	if reads := f.reads(); len(reads) != 0 {
		t.Fatalf("history reads %v while the clock runs; want none", reads)
	}
	got := f.alertMinutes(t, t0, 15, 20, f.worksThenWaits)
	if want := map[string][]int{condResultMissing: {15}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ALERTs about the seat at minutes %v; want %v", got, want)
	}
	assertResultMissingAlert(t, f.alerts(resultMissingTestSeat, condResultMissing)[0], "t1")
	if f.active(resultMissingTestSeat, condStall) {
		t.Errorf("stall active while result_missing holds; want it not raised")
	}
}

// AC4: once the clock has run out at t0+15m and result_missing does not
// hold, because the history has the seat's RESULT of the TASK, shows the
// seat waiting for the leader's answer to its QUESTION, cannot be read, or
// does not reach back to the TASK, the seat's stall is decided by its own
// rule, as before: it holds, so the stall is raised in that cycle, once.
// It is held until then, as the watch cannot tell these from the
// condition before it reads the history.
func TestWatch_ResultMissing_TheStallIsDecidedAsBeforeOnceTheClockRunsOutWithoutIt(t *testing.T) {
	for _, tc := range []struct {
		name     string
		setup    func(f *resultMissingFixture, t0 time.Time)
		wantSeen bool
	}{
		{name: "the seat sent the RESULT of the TASK", wantSeen: true, setup: func(f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0.Add(9*time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: RESULT\nTASK_ID: t1\n\ndone")
		}},
		{name: "the seat waits for the leader's answer to its QUESTION", setup: func(f *resultMissingFixture, t0 time.Time) {
			f.lineAt(t0.Add(9*time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: QUESTION\nTASK_ID: t1\n\nwhich wording?")
		}},
		{name: "the history cannot be read", setup: func(f *resultMissingFixture, _ time.Time) {
			f.hist.err = errors.New("history.sh: exit status 1")
		}},
		{name: "the history does not reach back to the TASK", setup: func(f *resultMissingFixture, t0 time.Time) {
			for i := range 1000 {
				f.lineAt(t0.Add(time.Minute), LeaderIdentity, resultMissingTestSeat, fmt.Sprintf("TYPE: HEARTBEAT\n\n%d", i))
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newResultMissingFixture(t)
			t0 := f.clk.Now()
			f.task(t, "t1")
			tc.setup(f, t0)
			got := f.alertMinutes(t, t0, 0, 20, f.worksThenWaits)
			if want := map[string][]int{condStall: {15}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("ALERTs about the seat at minutes %v; want %v", got, want)
			}
			if seen := f.status.SeatSnapshots[resultMissingTestSeat].ResultSeen != ""; seen != tc.wantSeen {
				t.Errorf("TASK remembered as done %t; want %t", seen, tc.wantSeen)
			}
		})
	}
}

// AC4: the stall of a seat whose result_missing clock does not run is not
// held, and is raised by the stall's own rule as before: a seat whose TASK
// is remembered as done (its RESULT found at t0+6m) and that works and then
// waits for input from t0+10m gets its stall at t0+11m; so does one whose
// TASK a STOP at t0+1m followed; and a seat herdr reports working with the
// same reply in every cycle, which is not waiting for input, gets its stall
// at t0+6m.
func TestWatch_ResultMissing_DoesNotHoldTheStallWhenItsClockDoesNotRun(t *testing.T) {
	frozenIdle, frozenWorking := watchAgentReply("idle", 0), watchAgentReply("working", 0)
	for _, tc := range []struct {
		name      string
		setup     func(t *testing.T, f *resultMissingFixture, t0 time.Time)
		reply     func(f *resultMissingFixture, m int) string
		wantStall []int
	}{
		{name: "the TASK is remembered as done", wantStall: []int{11},
			setup: func(_ *testing.T, f *resultMissingFixture, t0 time.Time) {
				f.lineAt(t0.Add(time.Minute), resultMissingTestSeat, LeaderIdentity, "TYPE: RESULT\nTASK_ID: t1\n\ndone")
			},
			reply: func(f *resultMissingFixture, m int) string {
				switch {
				case m >= 1 && m <= 6:
					return f.idle()
				case m < 10:
					return f.working()
				}
				return frozenIdle
			}},
		{name: "a STOP followed the TASK", wantStall: []int{11},
			setup: func(t *testing.T, f *resultMissingFixture, t0 time.Time) {
				f.advanceTo(t0.Add(time.Minute))
				f.sent(t, resultMissingTestSeat, "TYPE: STOP\n\nstop here")
				f.advanceTo(t0)
			},
			reply: (*resultMissingFixture).worksThenWaits},
		{name: "herdr reports the seat working with the same reply", wantStall: []int{6},
			reply: func(*resultMissingFixture, int) string { return frozenWorking }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newResultMissingFixture(t)
			t0 := f.clk.Now()
			f.task(t, "t1")
			if tc.setup != nil {
				tc.setup(t, f, t0)
			}
			got := f.alertMinutes(t, t0, 0, 14, func(m int) string { return tc.reply(f, m) })
			if want := map[string][]int{condStall: tc.wantStall}; !reflect.DeepEqual(got, want) {
				t.Errorf("ALERTs about the seat at minutes %v; want %v", got, want)
			}
		})
	}
}

// AC4: the leader's stall is not held: with a TASK to it at t0, working for
// ten minutes and then waiting for input with the same reply, the leader
// gets its stall at t0+11m by the stall's own rule, as before, and no
// result_missing.
func TestWatch_ResultMissing_DoesNotHoldTheLeadersStall(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.sent(t, LeaderIdentity, "TYPE: TASK\nTASK_ID: t1\n\nlead the org")
	var stallAt []int
	for m := 0; m <= 14; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.leaderReply = watchAgentReply("working", m)
		if m >= 10 {
			f.leaderReply = watchAgentReply("idle", 0)
		}
		before := len(f.alerts(LeaderIdentity, condStall))
		f.cycle(t, f.idle())
		if len(f.alerts(LeaderIdentity, condStall)) > before {
			stallAt = append(stallAt, m)
		}
	}
	if !reflect.DeepEqual(stallAt, []int{11}) {
		t.Errorf("the leader's stall ALERTs at minutes %v; want [11]", stallAt)
	}
	if got := f.alerts(LeaderIdentity, condResultMissing); len(got) != 0 {
		t.Errorf("result_missing ALERTs about the leader %q; want none", got)
	}
}

// AC4: a new TASK clears the condition, and the new TASK is watched on its
// own: raised stall_minutes after it, naming its TASK_ID.
func TestWatch_ResultMissing_ANewTaskClearsItAndIsWatchedOnItsOwn(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.task(t, "t1")
	for m := 0; m <= 5; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.idle())
	}
	if n := len(f.alerts(resultMissingTestSeat, condResultMissing)); n != 1 {
		t.Fatalf("%d ALERTs for t1; want one", n)
	}
	f.advanceTo(t0.Add(6 * time.Minute))
	f.task(t, "t2")
	f.cycle(t, f.idle())
	if f.active(resultMissingTestSeat, condResultMissing) {
		t.Fatalf("result_missing still active after a new TASK; want it cleared")
	}
	for m := 7; m <= 11; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.idle())
		if n := len(f.alerts(resultMissingTestSeat, condResultMissing)); m < 11 && n != 1 {
			t.Fatalf("%d ALERTs at t0+%dm; want t2 raised only at t0+11m", n, m)
		}
	}
	got := f.alerts(resultMissingTestSeat, condResultMissing)
	if len(got) != 2 {
		t.Fatalf("ALERTs %q; want a second one for t2", got)
	}
	assertResultMissingAlert(t, got[1], "t2")
}

// AC4: the leader is never watched for result_missing, even when the ledger
// holds a TASK sent to it and it waits for input; its history is not read.
func TestWatch_ResultMissing_IsNotRaisedForTheLeader(t *testing.T) {
	f := newResultMissingFixture(t)
	t0 := f.clk.Now()
	f.sent(t, LeaderIdentity, "TYPE: TASK\nTASK_ID: t1\n\nlead the org")
	for m := 0; m <= 12; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.leaderReply = watchAgentReply("idle", m)
		f.cycle(t, f.working())
	}
	if got := f.alerts(LeaderIdentity, condResultMissing); len(got) != 0 {
		t.Errorf("result_missing ALERTs about the leader %q; want none", got)
	}
	if reads := f.reads(); len(reads) != 0 {
		t.Errorf("history reads %v; want none", reads)
	}
	if snap := f.status.SeatSnapshots[LeaderIdentity]; snap == nil || snap.IdleSince != "" {
		t.Errorf("the leader's snapshot %+v; want no IdleSince", snap)
	}
}

// A seat stopped on a usage limit is reported by the org's usage_limited and
// gets no result_missing, though it waits for input past stall_minutes
// without the result of its TASK.
func TestWatch_ResultMissing_ASeatStoppedOnAUsageLimitGetsNone(t *testing.T) {
	f := newUsageLimitFixture(t, usageLimitSeat{LeaderIdentity, "codex"}, usageLimitSeat{resultMissingTestSeat, "claude"})
	f.screen(t, resultMissingTestSeat, "claude-session-limit.txt")
	t0 := f.clk.Now()
	if err := f.o.Manifest.Append(ManifestEvent{
		TS: t0.UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: resultMissingTestSeat, Event: EventSent,
		Details: "TYPE: TASK\nTASK_ID: t1\n\nimplement the slice",
	}); err != nil {
		t.Fatal(err)
	}
	for m := 0; m <= 12; m++ {
		f.reply(LeaderIdentity, fmt.Sprintf("leader-%d", m))
		reply := watchAgentReply("working", m)
		if m >= 3 {
			reply = watchAgentReply("idle", 0)
		}
		f.reply(resultMissingTestSeat, reply)
		f.cycle(t)
	}
	if got := f.alerts(condResultMissing); len(got) != 0 {
		t.Errorf("result_missing ALERTs %q; want none for a seat stopped on a usage limit", got)
	}
	if got := f.alerts(condUsageLimited); len(got) != 1 {
		t.Errorf("usage_limited ALERTs %q; want one", got)
	}
}

// The deadman escalates a result_missing ALERT the leader did not act on as
// an ALERT item of the inbox, like any seat's ALERT: CONDITION
// result_missing, reason deadman_timeout, and a body that passes the
// protocol check.
func TestWatch_ResultMissing_EscalatesAsAnAlertItemOfTheInbox(t *testing.T) {
	f := newResultMissingFixture(t)
	f.leaderReply = "leader-unchanged"
	dir := t.TempDir()
	inbox := NewInboxStore(dir)
	inbox.Now = f.clk.Now
	f.o.Inbox = inbox
	f.o.EscalationsPath = EscalationsPathIn(dir)
	f.o.DesktopNotify = func(context.Context, string) error { return nil }
	f.run.escalationsPath = f.o.EscalationsPath
	t0 := f.clk.Now()
	f.task(t, "t1")
	for m := 0; m <= 16; m++ {
		f.advanceTo(t0.Add(time.Duration(m) * time.Minute))
		f.cycle(t, f.idle())
	}
	key := conditionKey("org-a", resultMissingTestSeat, condResultMissing)
	var items []InboxItem
	for _, item := range mustReadInbox(t, inbox).Items {
		if item.Type == protocol.TypeAlert && alertIDKey(item.AlertID) == key {
			items = append(items, item)
		}
	}
	if len(items) != 1 {
		t.Fatalf("result_missing ALERT items %+v; want one", items)
	}
	assertAlertItemBody(t, items[0].Body, map[string]string{
		"ORG_ID": "org-a", "SEAT": resultMissingTestSeat, "CONDITION": condResultMissing,
		"REASON": "deadman_timeout", "ALERT_ID": items[0].AlertID,
		"ALERT_TS": t0.Add(5 * time.Minute).UTC().Format(time.RFC3339),
	})
}

// A history line is read in the shape history.sh prints it, the first line
// without its leading spaces and a last line with an empty body without the
// space after the colon, as the driver trims its output: the time to the
// second, from and to, and the body with its escapes undone. A line of any
// other shape, or whose time does not parse, is not read.
func TestResultMissing_ParseAgmsgHistoryLine(t *testing.T) {
	at := time.Date(2026, 11, 22, 1, 2, 3, 0, time.UTC)
	for _, tc := range []struct {
		line string
		want agmsgHistoryRecord
	}{
		{`  ○ [2026-11-22T01:02:03Z] implementer → leader: TYPE: RESULT\nTASK_ID: t1\n\nok`,
			agmsgHistoryRecord{At: at, From: "implementer", To: "leader", Body: "TYPE: RESULT\nTASK_ID: t1\n\nok"}},
		{`● [2026-11-22T01:02:03Z] leader → implementer: TYPE: TASK`,
			agmsgHistoryRecord{At: at, From: "leader", To: "implementer", Body: "TYPE: TASK"}},
		{`  ○ [2026-11-22T01:02:03Z] implementer → leader:`,
			agmsgHistoryRecord{At: at, From: "implementer", To: "leader", Body: ""}},
		{`  ○ [2026-11-22T01:02:03.456Z] a → b: x`,
			agmsgHistoryRecord{At: at, From: "a", To: "b", Body: "x"}},
		{`  ○ [2026-11-22T10:02:03+09:00] a → b: x`,
			agmsgHistoryRecord{At: at, From: "a", To: "b", Body: "x"}},
		{`  ○ [2026-11-22T01:02:03Z] a → b: see [x] a → b: y\tz`,
			agmsgHistoryRecord{At: at, From: "a", To: "b", Body: "see [x] a → b: y\tz"}},
	} {
		got, ok := parseAgmsgHistoryLine(tc.line)
		if !ok || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseAgmsgHistoryLine(%q) = %+v, %t; want %+v, true", tc.line, got, ok, tc.want)
		}
	}
	for _, line := range []string{
		`  ○ [2026-11-22T01:0`,
		`  ○ [yesterday] a → b: x`,
		`  ○ [2026-11-22T01:02:03Z] a b: x`,
		`  ○ [2026-11-22T01:02:03Z] a → b x`,
		`  ○ [2026-11-22T01:02:03Z]  → b: x`,
		`  ○ [2026-11-22T01:02:03Z] a → : x`,
		`No message history yet`,
	} {
		if got, ok := parseAgmsgHistoryLine(line); ok {
			t.Errorf("parseAgmsgHistoryLine(%q) = %+v, true; want it not read", line, got)
		}
	}
}

// The whole output of history.sh is read line by line: "No message
// history." and an empty output are no line, blank lines are passed over,
// and one line that does not parse makes the whole read unknown.
func TestResultMissing_ParseAgmsgHistory(t *testing.T) {
	for _, out := range []string{"", agmsgNoHistory, "\n" + agmsgNoHistory + "\n"} {
		if records, ok := parseAgmsgHistory(out); !ok || len(records) != 0 {
			t.Errorf("parseAgmsgHistory(%q) = %v, %t; want no line, true", out, records, ok)
		}
	}
	two := "○ [2026-11-22T01:02:03Z] a → b: x\n\n  ● [2026-11-22T01:02:04Z] b → a: y"
	if records, ok := parseAgmsgHistory(two); !ok || len(records) != 2 || records[1].Body != "y" {
		t.Errorf("parseAgmsgHistory(%q) = %+v, %t; want two lines", two, records, ok)
	}
	if records, ok := parseAgmsgHistory(two + "\n  ○ [2026-11-22T01:0"); ok {
		t.Errorf("parseAgmsgHistory with a cut line = %+v, true; want unknown", records)
	}
}

// seatLastTask reads the seat's last TASK of its current life from the
// ledger: a later TASK replaces an earlier one, a STOP only stops the TASK
// before it, another seat's, another org's, and a dry run's events do not
// count, and a TASK before the seat's latest spawn is of an earlier life.
func TestResultMissing_SeatLastTask(t *testing.T) {
	sent := func(ts, org, seat, details string, dry bool) ManifestEvent {
		return ManifestEvent{TS: ts, OrgID: org, SeatID: seat, Event: EventSent, Details: details, DryRun: dry}
	}
	spawned := func(ts string) ManifestEvent {
		return ManifestEvent{TS: ts, OrgID: "org-a", SeatID: "s", Event: EventSpawned}
	}
	task := func(id string) string { return "TYPE: TASK\nTASK_ID: " + id + "\n\nwork" }
	const ts1, ts2, ts3 = "2026-11-22T01:00:00Z", "2026-11-22T01:01:00Z", "2026-11-22T01:02:00Z"
	for _, tc := range []struct {
		name   string
		events []ManifestEvent
		wantID string
		wantOK bool
	}{
		{"the later TASK", []ManifestEvent{spawned(ts1), sent(ts1, "org-a", "s", task("t1"), false), sent(ts2, "org-a", "s", task("t2"), false)}, "t2", true},
		{"a STOP after the TASK", []ManifestEvent{spawned(ts1), sent(ts1, "org-a", "s", task("t1"), false), sent(ts2, "org-a", "s", "TYPE: STOP\n\nstop", false)}, "", false},
		{"a raw STOP after the TASK", []ManifestEvent{spawned(ts1), sent(ts1, "org-a", "s", task("t1"), false), sent(ts2, "org-a", "s", "raw=true TYPE: STOP\n\nstop", false)}, "", false},
		{"a STOP before the TASK", []ManifestEvent{spawned(ts1), sent(ts1, "org-a", "s", "TYPE: STOP\n\nstop", false), sent(ts2, "org-a", "s", task("t2"), false)}, "t2", true},
		{"a DECISION after the TASK", []ManifestEvent{spawned(ts1), sent(ts1, "org-a", "s", task("t1"), false), sent(ts2, "org-a", "s", "TYPE: DECISION\n\ngo on", false)}, "t1", true},
		{"other seats, orgs, and dry runs", []ManifestEvent{spawned(ts1), sent(ts1, "org-a", "s", task("t1"), false),
			sent(ts2, "org-a", "s2", task("x"), false), sent(ts2, "org-b", "s", task("y"), false), sent(ts3, "org-a", "s", task("z"), true),
			sent(ts3, "org-a", "s2", "TYPE: STOP\n\nstop", false)}, "t1", true},
		{"a TASK of an earlier life", []ManifestEvent{spawned(ts1), sent(ts2, "org-a", "s", task("t1"), false), spawned(ts3)}, "", false},
		{"a TASK whose time does not parse", []ManifestEvent{spawned(ts1), sent("not a time", "org-a", "s", task("t1"), false)}, "t1", false},
		{"a TASK_ID with a control character", []ManifestEvent{spawned(ts1), sent(ts1, "org-a", "s", "TYPE: TASK\nTASK_ID: t\x1b1\n\nwork", false)}, "t\x1b1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := seatLastTask(tc.events, "org-a", "s")
			if ok != tc.wantOK || (tc.wantID != "" && got.TaskID != tc.wantID) {
				t.Errorf("seatLastTask = %+v, %t; want TASK_ID %q, %t", got, ok, tc.wantID, tc.wantOK)
			}
		})
	}
}
