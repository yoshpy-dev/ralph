package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// lockedBuffer is a bytes.Buffer safe to write from the watch loop's
// goroutine while the test reads it.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// hangingWatchHerdr is a fakeWatchHerdr whose AgentGet for the targets in
// hang does not return until ctx is done: one org's herdr never answering
// (plan AC7b).
type hangingWatchHerdr struct {
	*fakeWatchHerdr
	hang map[string]bool
}

func (h *hangingWatchHerdr) AgentGet(ctx context.Context, target string) (string, error) {
	if h.hang[target] {
		<-ctx.Done()
		return "", ctx.Err()
	}
	return h.fakeWatchHerdr.AgentGet(ctx, target)
}

// hangingWatchAgmsg is a fakeWatchAgmsg whose Send, Join, and History on
// the teams in hang record the call and then do not return until ctx is
// done.
type hangingWatchAgmsg struct {
	*fakeWatchAgmsg
	hang map[string]bool
}

func (a *hangingWatchAgmsg) Send(ctx context.Context, team, from, to, message string) error {
	err := a.fakeWatchAgmsg.Send(ctx, team, from, to, message)
	if a.hang[team] {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (a *hangingWatchAgmsg) Join(ctx context.Context, team, agentID, agmsgType, projectPath string) error {
	err := a.fakeWatchAgmsg.Join(ctx, team, agentID, agmsgType, projectPath)
	if a.hang[team] {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (a *hangingWatchAgmsg) History(ctx context.Context, team, agentID string, limit int) (string, error) {
	out, err := a.fakeWatchAgmsg.History(ctx, team, agentID, limit)
	if a.hang[team] {
		<-ctx.Done()
		return "", ctx.Err()
	}
	return out, err
}

// beforeAgentGetHerdr is a fakeWatchHerdr that calls before ahead of each
// AgentGet, the only herdr call of a pulse cycle.
type beforeAgentGetHerdr struct {
	*fakeWatchHerdr
	before func()
}

func (h *beforeAgentGetHerdr) AgentGet(ctx context.Context, target string) (string, error) {
	h.before()
	return h.fakeWatchHerdr.AgentGet(ctx, target)
}

// watchAllFixture is one ledger for RunWatchAll: the watch test fakes, a
// manifest, and an inbox with the default deadlines (ack 10 min, resolve
// 60 min), all in dir, on a frozen clock.
type watchAllFixture struct {
	o      *Org
	h      *fakeWatchHerdr
	a      *fakeWatchAgmsg
	clk    *fakeClock
	dir    string
	stderr *lockedBuffer

	mu       sync.Mutex
	notified []string
	// gitCalls counts the git status calls per worktree.
	gitCalls map[string]int
}

func newWatchAllFixture(t *testing.T) *watchAllFixture {
	t.Helper()
	dir := t.TempDir()
	f := &watchAllFixture{
		h:        &fakeWatchHerdr{AgentGetSeq: map[string][]string{}, AgentGetErrSeq: map[string][]error{}},
		a:        &fakeWatchAgmsg{HistorySeq: map[string][]string{}, JoinErrSeq: map[string][]error{}},
		clk:      newFakeClock(),
		dir:      dir,
		stderr:   &lockedBuffer{},
		gitCalls: map[string]int{},
	}
	cfg := watchTestConfig()
	cfg.Director = config.OrgDirectorConfig{AckMinutes: 10, ResolveMinutes: 60}
	inbox := NewInboxStore(dir)
	inbox.Now = f.clk.Now
	f.o = &Org{
		Config:          cfg,
		Manifest:        NewManifestStoreAtPath(ManifestPathIn(dir)),
		Receipts:        NewReceiptStoreAtPath(filepath.Join(dir, "receipts.jsonl")),
		Herdr:           f.h,
		Agmsg:           f.a,
		Now:             f.clk.Now,
		Inbox:           inbox,
		EscalationsPath: EscalationsPathIn(dir),
		DesktopNotify: func(_ context.Context, message string) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.notified = append(f.notified, message)
			return nil
		},
	}
	return f
}

// params runs cycles cycles, 1 ms apart, with fast lock and stop waits, a
// git status stub, and no desktop escalation.
func (f *watchAllFixture) params(cycles int) WatchAllParams {
	return WatchAllParams{
		StatusDir: f.dir, Interval: time.Millisecond, Cycles: cycles,
		GitStatus: f.gitStatus, Escalate: func(context.Context, string) error { return nil },
		Stderr: f.stderr, LockWait: 50 * time.Millisecond, StopPollInterval: time.Millisecond,
	}
}

// gitStatus answers a different porcelain line on each call for a
// worktree, so every seat with a worktree changes scope from its second
// cycle on.
func (f *watchAllFixture) gitStatus(cwd string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gitCalls[cwd]++
	return fmt.Sprintf(" M file-%d\n", f.gitCalls[cwd]), nil
}

func (f *watchAllFixture) notifications() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.notified)
}

func (f *watchAllFixture) spawn(t *testing.T, orgID, seatID string) {
	t.Helper()
	if r := f.o.Spawn(watchSpawnParams(orgID, seatID, "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn %s/%s failed: %+v", orgID, seatID, r)
	}
}

func (f *watchAllFixture) mark(t *testing.T) WatchAlive {
	t.Helper()
	mark, err := ReadWatchAlive(f.dir)
	if err != nil {
		t.Fatalf("ReadWatchAlive: %v", err)
	}
	return *mark
}

// runWatchAll runs RunWatchAll on the test goroutine, so its hooks may
// call t.Fatal, and crashes the test binary when it has not returned
// within 10 s: a hung call that was not cut off would otherwise hold the
// test until go test's -timeout.
func runWatchAll(t *testing.T, o *Org, ctx context.Context, p WatchAllParams, hooks WatchHooks) error {
	t.Helper()
	guard := time.AfterFunc(10*time.Second, func() {
		panic(t.Name() + ": RunWatchAll did not return within 10s")
	})
	defer guard.Stop()
	return o.RunWatchAll(ctx, p, hooks)
}

func readWatchStatusFile(t *testing.T, dir, orgID string) watchStatusFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, WatchStatusFileName(orgID)))
	if err != nil {
		t.Fatalf("read the status file of %s: %v", orgID, err)
	}
	var s watchStatusFile
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("parse the status file of %s: %v", orgID, err)
	}
	return s
}

// alertsByTeam returns the watchdog ALERT messages of a, grouped by agmsg
// team.
func alertsByTeam(a *fakeWatchAgmsg) map[string][]string {
	out := map[string][]string{}
	for _, c := range watchdogAlerts(a) {
		out[c.team] = append(out[c.team], c.message)
	}
	return out
}

// runSingleOrgCycles runs orgID's pulse cycle the way the retired
// single-org RunWatch did: its status file loaded once, then evaluateCycle
// cycles times, with f's git status stub and stderr and no desktop
// escalation. It is the baseline RunWatchAll is compared with (AC7).
func (f *watchAllFixture) runSingleOrgCycles(t *testing.T, orgID string, cycles int) {
	t.Helper()
	statusPath := filepath.Join(f.dir, WatchStatusFileName(orgID))
	status, err := loadWatchStatus(statusPath, orgID)
	if err != nil {
		t.Fatalf("load the status file of %s: %v", orgID, err)
	}
	run := &watchRun{
		org: f.o, cfg: f.o.Config,
		gitStatus: f.gitStatus, escalateFn: func(context.Context, string) error { return nil }, stderr: f.stderr,
		statusPath: statusPath, escalationsPath: EscalationsPathIn(f.dir),
	}
	for n := 1; n <= cycles; n++ {
		if err := run.evaluateCycle(orgID, status); err != nil {
			t.Fatalf("cycle %d of %s: %v", n, orgID, err)
		}
	}
}

// AC7: one RunWatchAll gives each running org what the single-org cycle of
// that org gives (the same status file and the same ALERTs), and does not
// look at a disbanded org.
func TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated(t *testing.T) {
	setup := func(t *testing.T) *watchAllFixture {
		f := newWatchAllFixture(t)
		f.spawn(t, "org-a", "seat-a")
		f.spawn(t, "org-b", "seat-b")
		f.spawn(t, "org-c", "seat-c")
		if r := f.o.Disband(DisbandParams{OrgID: "org-c"}); len(r.Errs) != 0 || !r.Disbanded {
			t.Fatalf("disband org-c: %+v", r)
		}
		// org-a's seat fails liveness; both seats change scope in cycle 2
		// (gitStatus).
		f.h.AgentGetErrSeq[herdrAgentName("org-a", "seat-a")] = []error{errors.New("herdr: agent not found")}
		return f
	}

	all := setup(t)
	if err := runWatchAll(t, all.o, context.Background(), all.params(2), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}

	single := setup(t)
	for _, orgID := range []string{"org-a", "org-b"} {
		single.runSingleOrgCycles(t, orgID, 2)
	}

	for _, orgID := range []string{"org-a", "org-b"} {
		got, err := os.ReadFile(filepath.Join(all.dir, WatchStatusFileName(orgID)))
		if err != nil {
			t.Fatalf("watch --all wrote no status file for %s: %v", orgID, err)
		}
		want, err := os.ReadFile(filepath.Join(single.dir, WatchStatusFileName(orgID)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("status file of %s differs from the single-org watch's:\n got %s\nwant %s", orgID, got, want)
		}
	}
	if s := readWatchStatusFile(t, all.dir, "org-a"); s.Cycles != 2 || len(s.Conditions) == 0 {
		t.Errorf("org-a status = %+v; want 2 cycles and the liveness condition", s)
	}

	gotAlerts, wantAlerts := alertsByTeam(all.a), alertsByTeam(single.a)
	for _, team := range []string{agmsgTeam("org-a"), agmsgTeam("org-b")} {
		if len(gotAlerts[team]) == 0 || !slices.Equal(gotAlerts[team], wantAlerts[team]) {
			t.Errorf("ALERTs to %s = %q; want the single-org watch's ALERTs %q", team, gotAlerts[team], wantAlerts[team])
		}
	}
	if !strings.Contains(strings.Join(gotAlerts[agmsgTeam("org-a")], ""), "CONDITION: liveness") ||
		!strings.Contains(strings.Join(gotAlerts[agmsgTeam("org-b")], ""), "CONDITION: scope_change") {
		t.Errorf("ALERTs = %q; want org-a liveness and org-b scope_change", gotAlerts)
	}

	if _, err := os.Stat(filepath.Join(all.dir, WatchStatusFileName("org-c"))); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the disbanded org-c has a status file (%v); want it not evaluated", err)
	}
	if len(gotAlerts[agmsgTeam("org-c")]) != 0 {
		t.Errorf("ALERTs to the disbanded org-c: %q", gotAlerts[agmsgTeam("org-c")])
	}
	all.h.mu.Lock()
	calls := slices.Clone(all.h.agentGetCalls)
	all.h.mu.Unlock()
	for _, target := range calls {
		if strings.HasPrefix(target, "org-c_") {
			t.Errorf("herdr agent get for the disbanded org-c: %s", target)
		}
	}
	if got := all.stderr.String(); got != "" {
		t.Errorf("stderr = %q; want nothing for a clean run", got)
	}
}

// AC5: an item past its ack deadline that was never notified is sent in the
// first cycle of a watch; an item already notified is not, and the next
// cycle and a restarted watch do not send the first again.
func TestRunWatchAll_SendsAnInboxItemPastItsDeadlineInTheFirstCycleOnce(t *testing.T) {
	f := newWatchAllFixture(t)
	t0 := f.clk.Now()
	e1 := escalateInboxAt(t, f.o.Inbox, t0.Add(-11*time.Minute), "org-a")
	e2 := escalateInboxAt(t, f.o.Inbox, t0.Add(-11*time.Minute), "org-b")
	if err := f.o.Inbox.AppendNotified(e2, InboxReasonAckDeadline, "ok"); err != nil {
		t.Fatal(err)
	}

	var firstCycle []string
	hooks := WatchHooks{OnCycle: func(n int, _ time.Time) {
		if n == 1 {
			firstCycle = escalationReasons(t, f.o.EscalationsPath)
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), f.params(2), hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	want := []string{e1 + " " + InboxReasonAckDeadline}
	if !slices.Equal(firstCycle, want) {
		t.Fatalf("escalations.jsonl after the first cycle = %v; want %v", firstCycle, want)
	}
	if !strings.Contains(f.stderr.String(), "ORG INBOX DEADLINE: org=org-a item="+e1) {
		t.Errorf("stderr = %q; want the deadline banner for %s", f.stderr.String(), e1)
	}

	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("restarted RunWatchAll: %v", err)
	}
	if got := escalationReasons(t, f.o.EscalationsPath); !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl after a second cycle and a restart = %v; want still %v", got, want)
	}
	if n := len(f.notifications()); n != 1 {
		t.Errorf("desktop notified %d times; want 1", n)
	}
}

// The inbox deadlines are swept before any org's pulse (step 1 of
// RunWatchAll's cycle; mutation survivor A6), so an org whose herdr does
// not answer cannot hold them back by the length of its cut-off calls: at
// the first herdr call of the cycle, the item past its deadline is already
// in escalations.jsonl.
func TestRunWatchAll_SweepsTheInboxBeforeThePulse(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now().Add(-11*time.Minute), "org-a")
	var herdrCalls int
	var atFirstHerdrCall []string
	f.o.Herdr = &beforeAgentGetHerdr{fakeWatchHerdr: f.h, before: func() {
		herdrCalls++
		if herdrCalls == 1 {
			atFirstHerdrCall = escalationReasons(t, f.o.EscalationsPath)
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if herdrCalls == 0 {
		t.Fatal("the cycle made no herdr call")
	}
	want := []string{e1 + " " + InboxReasonAckDeadline}
	if !slices.Equal(atFirstHerdrCall, want) {
		t.Errorf("escalations.jsonl at the first herdr call = %v; want %v already sent", atFirstHerdrCall, want)
	}
}

// AC7b: the first org's herdr agent get never answers. It is cut off by
// DriverCallTimeout and reads as a liveness failure, and in the same cycle
// the inbox item is sent, the second org's cycle runs, and the mark is
// written.
func TestRunWatchAll_AHungHerdrCallOfTheFirstOrgIsCutOffAndTheCycleGoesOn(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	f.spawn(t, "org-b", "seat-b")
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now().Add(-11*time.Minute), "org-a")
	f.o.Herdr = &hangingWatchHerdr{fakeWatchHerdr: f.h, hang: map[string]bool{herdrAgentName("org-a", "seat-a"): true}}
	f.o.DriverCallTimeout = 20 * time.Millisecond

	var escalations []string
	var statusB watchStatusFile
	var mark WatchAlive
	hooks := WatchHooks{OnCycle: func(int, time.Time) {
		escalations = escalationReasons(t, f.o.EscalationsPath)
		statusB = readWatchStatusFile(t, f.dir, "org-b")
		mark = f.mark(t)
	}}
	start := time.Now()
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("the cycle took %v; want the hung call cut off at DriverCallTimeout", elapsed)
	}

	alerts := alertsByTeam(f.a)[agmsgTeam("org-a")]
	if len(alerts) != 1 || !strings.Contains(alerts[0], "CONDITION: liveness") || !strings.Contains(alerts[0], context.DeadlineExceeded.Error()) {
		t.Errorf("org-a ALERTs = %q; want one liveness ALERT for the cut-off call", alerts)
	}
	if want := []string{e1 + " " + InboxReasonAckDeadline}; !slices.Equal(escalations, want) {
		t.Errorf("escalations.jsonl = %v; want %v sent in the same cycle", escalations, want)
	}
	if statusB.Cycles != 1 {
		t.Errorf("org-b status = %+v; want its cycle run", statusB)
	}
	if mark.State != WatchAliveRunning || mark.TS != f.clk.Now().Format(time.RFC3339) {
		t.Errorf("mark after the cycle = %+v; want running with the cycle's ts", mark)
	}
}

// AC7b for every driver call of the pulse: the first org's herdr agent
// gets (its seat's and its leader's) and its agmsg Join, Send, and History
// never answer. Each is cut off and reads as that call's failure (Join not
// recorded, the ALERT logged as not sent, the deadman baselines at their
// probe-unavailable values), and the second org's cycle runs.
func TestRunWatchAll_EveryHungDriverCallOfTheFirstOrgIsCutOff(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	f.spawn(t, "org-b", "seat-b")
	f.o.Herdr = &hangingWatchHerdr{fakeWatchHerdr: f.h, hang: map[string]bool{
		herdrAgentName("org-a", "seat-a"):       true,
		herdrAgentName("org-a", LeaderIdentity): true,
	}}
	f.o.Agmsg = &hangingWatchAgmsg{fakeWatchAgmsg: f.a, hang: map[string]bool{agmsgTeam("org-a"): true}}
	f.o.DriverCallTimeout = 20 * time.Millisecond

	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}

	statusA := readWatchStatusFile(t, f.dir, "org-a")
	if statusA.WatchdogJoined {
		t.Error("org-a status records the watchdog joined; want the cut-off Join not recorded")
	}
	if len(statusA.PendingAlerts) != 1 {
		t.Fatalf("org-a pending alerts = %+v; want the liveness ALERT's one", statusA.PendingAlerts)
	}
	for _, pending := range statusA.PendingAlerts {
		if pending.LeaderAgentGet != "" || pending.HistoryLeaderLines != -1 {
			t.Errorf("pending alert = %+v; want the cut-off leader probe and history at \"\" and -1", pending)
		}
	}
	if want := `watchdog: failed to ALERT leader for org "org-a" condition "liveness": ` + context.DeadlineExceeded.Error(); !strings.Contains(f.stderr.String(), want) {
		t.Errorf("stderr = %q; want %q", f.stderr.String(), want)
	}
	if s := readWatchStatusFile(t, f.dir, "org-b"); s.Cycles != 1 {
		t.Errorf("org-b status = %+v; want its cycle run", s)
	}
	if mark := f.mark(t); mark.TS != f.clk.Now().Format(time.RFC3339) {
		t.Errorf("mark = %+v; want the cycle's ts", mark)
	}
}

// AC7b: the first org's status file cannot be read or written. That org is
// skipped with one line on stderr (the load's own line, or the line of a
// skipped cycle), and in the same cycle the inbox item is sent, the second
// org's cycle runs, and the mark is written.
func TestRunWatchAll_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg(t *testing.T) {
	cases := []struct {
		name string
		// blockAt is the cycle before which watch-status-org-a.json becomes
		// a directory: 1 makes the load fail, 2 makes the save fail.
		blockAt  int
		wantLine string
	}{
		{name: "a directory at the status file from the start", blockAt: 1, wantLine: `watchdog: org "org-a" is skipped every cycle until its watch-status file can be read (said once):`},
		{name: "a directory put at the status file after the first cycle", blockAt: 2, wantLine: `watchdog: org "org-a" skipped this cycle:`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			f.spawn(t, "org-a", "seat-a")
			f.spawn(t, "org-b", "seat-b")
			// e1 becomes due in the cycle that blocks: 11 min old at the start
			// when blockAt is 1; escalated at the start, with the clock moved
			// 11 min after cycle 1, when blockAt is 2.
			escalatedAt := f.clk.Now()
			if tc.blockAt == 1 {
				escalatedAt = escalatedAt.Add(-11 * time.Minute)
			}
			e1 := escalateInboxAt(t, f.o.Inbox, escalatedAt, "org-a")
			statusA := filepath.Join(f.dir, WatchStatusFileName("org-a"))
			block := func() {
				if err := os.Remove(statusA); err != nil && !errors.Is(err, fs.ErrNotExist) {
					t.Fatal(err)
				}
				if err := os.Mkdir(statusA, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if tc.blockAt == 1 {
				block()
			}

			var escalations []string
			var statusB watchStatusFile
			var mark WatchAlive
			hooks := WatchHooks{OnCycle: func(n int, _ time.Time) {
				if n == tc.blockAt {
					escalations = escalationReasons(t, f.o.EscalationsPath)
					statusB = readWatchStatusFile(t, f.dir, "org-b")
					mark = f.mark(t)
				}
				if n == 1 && tc.blockAt == 2 {
					if got := escalationReasons(t, f.o.EscalationsPath); len(got) != 0 {
						t.Errorf("escalations.jsonl after cycle 1 = %v; want nothing due yet", got)
					}
					block()
					f.clk.Advance(11 * time.Minute)
				}
			}}
			if err := runWatchAll(t, f.o, context.Background(), f.params(tc.blockAt), hooks); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}

			if lines := strings.Count(f.stderr.String(), `watchdog: org "org-a"`); lines != 1 || !strings.Contains(f.stderr.String(), tc.wantLine) {
				t.Errorf("stderr = %q; want one line saying org-a was skipped, %q", f.stderr.String(), tc.wantLine)
			}
			if want := []string{e1 + " " + InboxReasonAckDeadline}; !slices.Equal(escalations, want) {
				t.Errorf("escalations.jsonl = %v; want %v sent in the same cycle", escalations, want)
			}
			if statusB.Cycles != tc.blockAt {
				t.Errorf("org-b status = %+v; want %d cycles", statusB, tc.blockAt)
			}
			if mark.State != WatchAliveRunning || mark.TS != f.clk.Now().Format(time.RFC3339) {
				t.Errorf("mark = %+v; want running with the blocked cycle's ts", mark)
			}
		})
	}
}

// Self-review L3: a status file that cannot be read (here: not JSON) is said
// on stderr once in the run, not every cycle. The org is skipped and its
// file left as it is, and the load is tried again each cycle, so once the
// file is gone the next cycle watches the org again. Status files are
// written through a rename, so no temporary file is left behind.
func TestRunWatchAll_AStatusFileThatCannotBeReadIsSaidOncePerOrg(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	f.spawn(t, "org-b", "seat-b")
	statusA := filepath.Join(f.dir, WatchStatusFileName("org-a"))
	if err := os.WriteFile(statusA, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := WatchHooks{OnCycle: func(n int, _ time.Time) {
		if n != 2 {
			return
		}
		if data, err := os.ReadFile(statusA); err != nil || string(data) != "{not json" {
			t.Errorf("org-a status file after 2 cycles = %q (%v); want it left as it was", data, err)
		}
		if err := os.Remove(statusA); err != nil {
			t.Fatal(err)
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), f.params(3), hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}

	want := `watchdog: org "org-a" is skipped every cycle until its watch-status file can be read (said once): org: watch: parse ` + statusA + ": "
	if got := f.stderr.String(); strings.Count(got, `watchdog: org "org-a"`) != 1 || !strings.Contains(got, want) {
		t.Errorf("stderr = %q; want one line with %q", got, want)
	}
	if s := readWatchStatusFile(t, f.dir, "org-a"); s.Cycles != 1 {
		t.Errorf("org-a status = %+v; want the one cycle run after the file was removed", s)
	}
	if s := readWatchStatusFile(t, f.dir, "org-b"); s.Cycles != 3 {
		t.Errorf("org-b status = %+v; want 3 cycles", s)
	}
	for _, name := range dirEntries(t, f.dir) {
		if strings.HasSuffix(name, ".tmp") {
			t.Errorf("temporary file %s left in the state dir", name)
		}
	}
}

// Self-review L1: SendWatchdogAlertBounded, the on-demand watcher's ALERT
// send, is cut off at DriverCallTimeout when agmsg does not answer, and at
// once when its context is already done.
func TestSendWatchdogAlertBounded_IsCutOffAtTheDriverCallTimeoutOrByItsContext(t *testing.T) {
	o, _, a, _ := testWatchOrg(t)
	o.Agmsg = &hangingWatchAgmsg{fakeWatchAgmsg: a, hang: map[string]bool{agmsgTeam("org-a"): true}}
	o.DriverCallTimeout = 20 * time.Millisecond
	const msg = "TYPE: ALERT\nORG_ID: org-a\nSEAT: seat-1\nCONDITION: watcher_stall\n\nwatcher verdict=circular reason=x"

	start := time.Now()
	err := o.SendWatchdogAlertBounded(context.Background(), "org-a", msg)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("SendWatchdogAlertBounded with a hung agmsg = %v; want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("the send took %v; want it cut off at the 20ms DriverCallTimeout", elapsed)
	}

	o.DriverCallTimeout = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start = time.Now()
	if err := o.SendWatchdogAlertBounded(ctx, "org-a", msg); !errors.Is(err, context.Canceled) {
		t.Errorf("SendWatchdogAlertBounded with a cancelled context = %v; want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("the send took %v with a cancelled context; want it to return at once", elapsed)
	}
	if n := len(watchdogAlerts(a)); n != 2 {
		t.Errorf("agmsg got %d watchdog sends; want 2", n)
	}
}

// A manifest that cannot be read skips the pulse of the cycle, with one
// line on stderr; the inbox deadlines are still sent and the mark written.
func TestRunWatchAll_AManifestThatCannotBeReadSkipsOnlyThePulse(t *testing.T) {
	f := newWatchAllFixture(t)
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now().Add(-11*time.Minute), "org-a")
	if err := os.Mkdir(f.o.Manifest.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if !strings.Contains(f.stderr.String(), "watchdog: pulse skipped this cycle: read manifest:") {
		t.Errorf("stderr = %q; want the pulse-skipped line", f.stderr.String())
	}
	if got, want := escalationReasons(t, f.o.EscalationsPath), []string{e1 + " " + InboxReasonAckDeadline}; !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want %v", got, want)
	}
	if mark := f.mark(t); mark.State != WatchAliveStopped || mark.TS == "" {
		t.Errorf("mark = %+v; want a cycle ts and stopped at the end", mark)
	}
}

// An inbox that cannot be swept (here: an Org with no inbox store) is one
// line on stderr per cycle, and the pulse still runs.
func TestRunWatchAll_AnInboxThatCannotBeSweptDoesNotStopThePulse(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	f.o.Inbox = nil
	if err := runWatchAll(t, f.o, context.Background(), f.params(2), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if n := strings.Count(f.stderr.String(), "watchdog: inbox deadlines skipped this cycle: org: inbox deadlines: no inbox store is configured\n"); n != 2 {
		t.Errorf("stderr = %q; want the inbox line once per cycle", f.stderr.String())
	}
	if s := readWatchStatusFile(t, f.dir, "org-a"); s.Cycles != 2 {
		t.Errorf("org-a status = %+v; want 2 cycles", s)
	}
}

// AC9: a second RunWatchAll on the same state dir is refused, naming the
// first one's pid, and writes nothing; once the first returns, a new one
// starts.
func TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds(t *testing.T) {
	f := newWatchAllFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := f.params(0)
	first.Interval = time.Hour
	firstCycle := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- f.o.RunWatchAll(ctx, first, WatchHooks{OnCycle: func(n int, _ time.Time) {
			if n == 1 {
				close(firstCycle)
			}
		}})
	}()
	select {
	case <-firstCycle:
	case <-time.After(10 * time.Second):
		t.Fatal("the first watch ran no cycle")
	}
	markBefore, err := os.ReadFile(WatchAlivePathIn(f.dir))
	if err != nil {
		t.Fatal(err)
	}

	err = f.o.RunWatchAll(context.Background(), f.params(1), WatchHooks{OnCycle: func(int, time.Time) {
		t.Error("the refused watch ran a cycle")
	}})
	if !errors.Is(err, ErrWatchRunning) || !strings.Contains(err.Error(), fmt.Sprintf("pid %d", os.Getpid())) {
		t.Fatalf("second RunWatchAll = %v; want ErrWatchRunning naming pid %d", err, os.Getpid())
	}
	if markAfter, err := os.ReadFile(WatchAlivePathIn(f.dir)); err != nil || !bytes.Equal(markAfter, markBefore) {
		t.Errorf("the refused watch changed the mark: %s -> %s (%v)", markBefore, markAfter, err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("first RunWatchAll: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the first watch did not end after its context was cancelled")
	}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll after the first ended: %v", err)
	}
}

// AC10: the mark is written running (own pid, started_at, ts) before the
// first cycle and after each cycle, ProbeWatch reads it as running, and
// when the watch ends (its context cancelled, or its Cycles reached) the
// mark is stopped with stopped_at and the lock is free.
func TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd(t *testing.T) {
	cases := []struct {
		name   string
		cancel bool
	}{
		{name: "Cycles reached"},
		{name: "context cancelled", cancel: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			f.spawn(t, "org-a", "seat-a")
			t0 := f.clk.Now()
			wantStarted := t0.Format(time.RFC3339Nano)

			// The first git status call is inside the first cycle, before
			// that cycle writes the mark.
			var beforeFirstCycle WatchAlive
			p := f.params(3)
			p.GitStatus = func(cwd string) (string, error) {
				if f.gitCallCount(cwd) == 0 {
					beforeFirstCycle = f.mark(t)
				}
				return f.gitStatus(cwd)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				p.Cycles = 0 // the cancel after cycle 3 ends it
			}
			var cycles int
			hooks := WatchHooks{OnCycle: func(n int, ts time.Time) {
				cycles = n
				mark := f.mark(t)
				if mark.State != WatchAliveRunning || mark.PID != os.Getpid() || mark.StartedAt != wantStarted || mark.TS != ts.Format(time.RFC3339) || mark.StoppedAt != "" {
					t.Errorf("mark after cycle %d = %+v; want running, pid %d, started_at %s, ts %s", n, mark, os.Getpid(), wantStarted, ts.Format(time.RFC3339))
				}
				if got := ProbeWatch(f.dir, ts, f.o.Config.DeadmanMinutes).State; got != WatchLivenessRunning {
					t.Errorf("ProbeWatch after cycle %d = %q; want running", n, got)
				}
				f.clk.Advance(time.Minute)
				if tc.cancel && n == 3 {
					cancel()
				}
			}}
			if err := runWatchAll(t, f.o, ctx, p, hooks); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}

			if beforeFirstCycle.State != WatchAliveRunning || beforeFirstCycle.PID != os.Getpid() ||
				beforeFirstCycle.StartedAt != wantStarted || beforeFirstCycle.TS != t0.Format(time.RFC3339) {
				t.Errorf("mark before the first cycle = %+v; want running, pid %d, started_at and ts at the start", beforeFirstCycle, os.Getpid())
			}
			if cycles != 3 {
				t.Errorf("ran %d cycles; want 3", cycles)
			}
			mark := f.mark(t)
			if mark.State != WatchAliveStopped || mark.StoppedAt != f.clk.Now().Format(time.RFC3339) || mark.StartedAt != wantStarted || mark.PID != os.Getpid() {
				t.Errorf("mark at the end = %+v; want stopped with stopped_at %s", mark, f.clk.Now().Format(time.RFC3339))
			}
			if got := ProbeWatch(f.dir, f.clk.Now(), f.o.Config.DeadmanMinutes).State; got != WatchLivenessNotRunning {
				t.Errorf("ProbeWatch at the end = %q; want not_running", got)
			}
		})
	}
}

func (f *watchAllFixture) gitCallCount(cwd string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gitCalls[cwd]
}

// A stop request written while the watch waits between cycles ends it
// within about a tick: the mark is stopped, the request removed, and the
// lock free.
func TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt(t *testing.T) {
	f := newWatchAllFixture(t)
	p := f.params(0)
	p.Interval = time.Hour
	p.StopPollInterval = 5 * time.Millisecond
	firstCycle := make(chan struct{})
	var cycles int
	done := make(chan error, 1)
	go func() {
		done <- f.o.RunWatchAll(context.Background(), p, WatchHooks{OnCycle: func(n int, _ time.Time) {
			cycles = n
			if n == 1 {
				close(firstCycle)
			}
		}})
	}()
	select {
	case <-firstCycle:
	case <-time.After(10 * time.Second):
		t.Fatal("the watch ran no cycle")
	}
	if err := WriteWatchStopRequest(f.dir, f.clk.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunWatchAll: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the watch did not end after a stop request")
	}
	if cycles != 1 {
		t.Errorf("ran %d cycles; want 1", cycles)
	}
	if mark := f.mark(t); mark.State != WatchAliveStopped || mark.StoppedAt == "" {
		t.Errorf("mark = %+v; want stopped", mark)
	}
	if _, err := ReadWatchStopRequest(f.dir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stop request after the watch ended: %v; want it removed", err)
	}
	if got := ProbeWatch(f.dir, f.clk.Now(), f.o.Config.DeadmanMinutes).State; got != WatchLivenessNotRunning {
		t.Errorf("ProbeWatch = %q; want not_running", got)
	}
}

// AC13b: a stop request not later than the watch's start was left for an
// earlier watch. It is removed at the start and does not stop this one; a
// request later than the start that is already there at the start is
// followed after the first cycle.
func TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart(t *testing.T) {
	cases := []struct {
		name       string
		at         time.Duration // requested_at minus the start
		wantCycles int
		wantRemove bool // removed before the first cycle ends
	}{
		{name: "older than the start", at: -time.Minute, wantCycles: 3, wantRemove: true},
		{name: "at the same instant as the start", at: 0, wantCycles: 3, wantRemove: true},
		{name: "later than the start", at: time.Nanosecond, wantCycles: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			if err := WriteWatchStopRequest(f.dir, f.clk.Now().Add(tc.at)); err != nil {
				t.Fatal(err)
			}
			var cycles int
			var removedAtFirstCycle bool
			hooks := WatchHooks{OnCycle: func(n int, _ time.Time) {
				cycles = n
				if n == 1 {
					_, err := ReadWatchStopRequest(f.dir)
					removedAtFirstCycle = errors.Is(err, fs.ErrNotExist)
				}
			}}
			if err := runWatchAll(t, f.o, context.Background(), f.params(3), hooks); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}
			if cycles != tc.wantCycles {
				t.Errorf("ran %d cycles; want %d", cycles, tc.wantCycles)
			}
			if removedAtFirstCycle != tc.wantRemove {
				t.Errorf("request removed by the end of the first cycle: %t; want %t", removedAtFirstCycle, tc.wantRemove)
			}
			if _, err := ReadWatchStopRequest(f.dir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("stop request after the watch ended: %v; want it removed", err)
			}
		})
	}
}

// The start is read before the lock wait (mutation survivor S6): a stop
// request written while this watch waits for watch.lock is later than its
// started_at, so it is followed after the first cycle, not removed as one
// left for an earlier watch. The test holds the lock, waits for RunWatchAll's
// first Org.Now call (its start), then moves the clock on, writes the
// request between the start and the new time, and only then releases the
// lock. A start read after the lock would see the new time.
func TestRunWatchAll_StartedAtIsTakenBeforeWaitingForTheLock(t *testing.T) {
	f := newWatchAllFixture(t)
	holder := mustAcquireWatchLock(t, f.dir)
	t0 := f.clk.Now()
	startRead := make(chan struct{})
	var once sync.Once
	f.o.Now = func() time.Time {
		now := f.clk.Now()
		once.Do(func() { close(startRead) })
		return now
	}
	p := f.params(3)
	p.LockWait = 10 * time.Second
	var cycles int
	done := make(chan error, 1)
	go func() {
		done <- f.o.RunWatchAll(context.Background(), p, WatchHooks{OnCycle: func(n int, _ time.Time) { cycles = n }})
	}()
	select {
	case <-startRead:
	case err := <-done:
		t.Fatalf("RunWatchAll returned %v and read no clock while the test held watch.lock; want its start read before the lock wait", err)
	case <-time.After(30 * time.Second):
		t.Fatal("RunWatchAll read no clock within 30s")
	}
	f.clk.Advance(time.Minute)
	if err := WriteWatchStopRequest(f.dir, t0.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := holder.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunWatchAll: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunWatchAll did not return after the lock was released")
	}
	if cycles != 1 {
		t.Errorf("ran %d cycles; want 1 (the request written during the lock wait is followed)", cycles)
	}
	if mark := f.mark(t); mark.StartedAt != t0.Format(time.RFC3339Nano) {
		t.Errorf("started_at = %s; want %s, the clock before the lock wait", mark.StartedAt, t0.Format(time.RFC3339Nano))
	}
	if _, err := ReadWatchStopRequest(f.dir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stop request after the watch ended: %v; want it removed", err)
	}
}

// finish writes the stopped mark before it releases watch.lock (mutation
// survivor L6): whoever sees the lock free, a --stop or a new watch that
// takes it, also sees the stopped mark, and the old watch never writes over
// the running mark of a new one. From the stop request on, the test polls
// the lock with no pause and reads the mark as soon as the lock is free. A
// release before the mark shows only when a poll falls between the two, so
// the test runs 20 watches.
func TestRunWatchAll_WritesTheStoppedMarkBeforeReleasingTheLock(t *testing.T) {
	for run := 1; run <= 20; run++ {
		f := newWatchAllFixture(t)
		p := f.params(0)
		p.Interval = time.Hour
		firstCycle := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- f.o.RunWatchAll(context.Background(), p, WatchHooks{OnCycle: func(n int, _ time.Time) {
				if n == 1 {
					close(firstCycle)
				}
			}})
		}()
		select {
		case <-firstCycle:
		case <-time.After(10 * time.Second):
			t.Fatalf("run %d: the watch ran no cycle", run)
		}
		if err := WriteWatchStopRequest(f.dir, f.clk.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			held, err := fileLockHeld(WatchLockPathIn(f.dir))
			if err != nil {
				t.Fatalf("run %d: check watch.lock: %v", run, err)
			}
			if !held {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("run %d: watch.lock still held 10s after the stop request", run)
			}
		}
		mark, err := ReadWatchAlive(f.dir)
		if err != nil || mark.State != WatchAliveStopped || mark.StoppedAt == "" {
			t.Fatalf("run %d: mark when watch.lock was free = %+v (%v); want the stopped mark already written", run, mark, err)
		}
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("run %d: RunWatchAll: %v", run, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("run %d: RunWatchAll did not return after the lock was free", run)
		}
	}
}

// An org_id in the manifest that is not an identifier (a hand-edited line)
// never names a status file: the org is skipped with one line.
func TestRunWatchAll_AnOrgIDThatIsNotAnIdentifierIsSkipped(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	// watch-status-x/../../escape.json would be <state dir>/../escape.json.
	const badOrgID = "x/../../escape"
	ev := ManifestEvent{TS: f.clk.Now().Format(time.RFC3339), OrgID: badOrgID, SeatID: "seat-x", Event: EventSpawned, Role: "worker"}
	if err := f.o.Manifest.Append(ev); err != nil {
		t.Fatal(err)
	}
	if got := RunningOrgs(mustReadEvents(t, f.o)); !slices.Contains(got, badOrgID) {
		t.Fatalf("RunningOrgs = %v; the test needs the bad org_id to run", got)
	}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if !strings.Contains(f.stderr.String(), `watchdog: org "`+badOrgID+`" skipped this cycle: org: invalid org_id`) {
		t.Errorf("stderr = %q; want the bad org_id skipped", f.stderr.String())
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(f.dir), "escape.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a status file was written outside the state dir: %v", err)
	}
	if s := readWatchStatusFile(t, f.dir, "org-a"); s.Cycles != 1 {
		t.Errorf("org-a status = %+v; want its cycle run", s)
	}
}

// The real git status is cut off at DriverCallTimeout, and an exit 0
// whose output pipe a child of git still holds is a success.
func TestRealGitStatus_IsCutOffAtTheDriverCallTimeout(t *testing.T) {
	stubDir := t.TempDir()
	writeGit := func(script string) {
		if err := os.WriteFile(filepath.Join(stubDir, "git"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cwd := t.TempDir()

	writeGit("#!/bin/sh\necho ' M a.go'\n")
	o := &Org{DriverCallTimeout: 5 * time.Second}
	if out, err := o.realGitStatus(cwd); err != nil || out != " M a.go\n" {
		t.Fatalf("realGitStatus = %q, %v; want the stub's porcelain line", out, err)
	}

	// A child that keeps the pipe after git exits 0.
	writeGit("#!/bin/sh\nsleep 3 &\necho ' M b.go'\n")
	if out, err := o.realGitStatus(cwd); err != nil || out != " M b.go\n" {
		t.Fatalf("realGitStatus with a child holding the pipe = %q, %v; want a success", out, err)
	}

	// A git that does not return; its sleep child holds the pipe past the kill.
	writeGit("#!/bin/sh\nsleep 30\n")
	o.DriverCallTimeout = 50 * time.Millisecond
	start := time.Now()
	_, err := o.realGitStatus(cwd)
	if err == nil {
		t.Fatal("realGitStatus of a git that does not return succeeded")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("realGitStatus took %v; want it cut off at the timeout plus the wait delay", elapsed)
	}
}
