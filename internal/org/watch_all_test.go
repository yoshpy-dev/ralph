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
	"sync/atomic"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
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

// Tech-debt 204 (e) and AC9 (plan 2026-10-12-org-watch-gate): RunWatchAll
// calls OnLocked once, while it holds watch.lock and before it writes its
// first mark or runs a cycle, and a watch refused the lock does not call it.
func TestRunWatchAll_CallsOnLockedOnceItHoldsTheLock(t *testing.T) {
	f := newWatchAllFixture(t)
	calls := 0
	p := f.params(1)
	p.OnLocked = func() {
		calls++
		if lock, err := AcquireWatchLock(f.dir, 20*time.Millisecond); !errors.Is(err, ErrWatchRunning) {
			t.Errorf("AcquireWatchLock in OnLocked = %v; want ErrWatchRunning", err)
			if err == nil {
				_ = lock.Release()
			}
		}
		if _, err := ReadWatchAlive(f.dir); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("ReadWatchAlive in OnLocked = %v; want no mark yet", err)
		}
	}
	err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{OnCycle: func(int, time.Time) {
		if calls != 1 {
			t.Errorf("OnLocked was called %d times before the first cycle; want once", calls)
		}
	}})
	if err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if calls != 1 {
		t.Errorf("OnLocked was called %d times; want once", calls)
	}

	lock, err := AcquireWatchLock(f.dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()
	refused := f.params(1)
	refused.OnLocked = func() { t.Error("a watch refused the lock called OnLocked") }
	if err := runWatchAll(t, f.o, context.Background(), refused, WatchHooks{}); !errors.Is(err, ErrWatchRunning) {
		t.Fatalf("RunWatchAll while another holds the lock = %v; want ErrWatchRunning", err)
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

// Plan 2026-10-12-org-watch-gate, AC7 and AC14 at the watch: the mark
// carries the watch's own deadman_minutes as configured (0 included) and
// once for a run with Cycles > 0, from the first mark to the stopped one.
// ProbeWatch then judges the mark by that deadman_minutes, not by its
// caller's: 30 minutes after a cycle, a watch with 60 is running for a
// caller with 10, and one with 0 (the default 10) is stale for a caller
// with 60. The fixture's 1 ms interval makes interval_seconds 0, so the
// deadman alone sets the window.
func TestRunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce(t *testing.T) {
	cases := []struct {
		name      string
		deadman   int
		cycles    int
		wantOnce  bool
		callerDM  int
		wantAfter string
	}{
		{name: "deadman 60, Cycles 2", deadman: 60, cycles: 2, wantOnce: true, callerDM: 10, wantAfter: WatchLivenessRunning},
		{name: "deadman 0, long-running", deadman: 0, cycles: 0, callerDM: 60, wantAfter: WatchLivenessStale},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			f.o.Config.DeadmanMinutes = tc.deadman
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			check := func(what string, mark WatchAlive) {
				t.Helper()
				if mark.DeadmanMinutes == nil || *mark.DeadmanMinutes != tc.deadman || mark.Once != tc.wantOnce {
					t.Errorf("%s = %+v (deadman_minutes %v); want deadman_minutes %d, once %t", what, mark, mark.DeadmanMinutes, tc.deadman, tc.wantOnce)
				}
			}
			hooks := WatchHooks{OnCycle: func(n int, ts time.Time) {
				check(fmt.Sprintf("mark after cycle %d", n), f.mark(t))
				if got := ProbeWatch(f.dir, ts.Add(30*time.Minute), tc.callerDM).State; got != tc.wantAfter {
					t.Errorf("ProbeWatch 30 minutes after cycle %d with the caller's deadman %d = %q; want %q", n, tc.callerDM, got, tc.wantAfter)
				}
				if tc.cycles == 0 && n == 2 {
					cancel()
				}
			}}
			if err := runWatchAll(t, f.o, ctx, f.params(tc.cycles), hooks); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}
			mark := f.mark(t)
			if mark.State != WatchAliveStopped {
				t.Errorf("mark at the end = %+v; want stopped", mark)
			}
			check("the stopped mark", mark)
		})
	}
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
// earlier watch. It is removed at the start and does not stop this one, as
// is one that does not parse; a request later than the start that is
// already there at the start is followed after the first cycle. The file
// the start moves the request to (takeWatchStopRequest) is not left behind.
func TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart(t *testing.T) {
	cases := []struct {
		name       string
		at         time.Duration // requested_at minus the start
		raw        string        // written as watch-stop.json instead of a request at at
		wantCycles int
		wantRemove bool // removed before the first cycle ends
	}{
		{name: "older than the start", at: -time.Minute, wantCycles: 3, wantRemove: true},
		{name: "at the same instant as the start", at: 0, wantCycles: 3, wantRemove: true},
		{name: "later than the start", at: time.Nanosecond, wantCycles: 1},
		{name: "not JSON", raw: "{not json", wantCycles: 3, wantRemove: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			if tc.raw != "" {
				if err := os.WriteFile(WatchStopPathIn(f.dir), []byte(tc.raw), 0o644); err != nil {
					t.Fatal(err)
				}
			} else if err := WriteWatchStopRequest(f.dir, f.clk.Now().Add(tc.at)); err != nil {
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
			if names := dirEntries(t, f.dir); slices.Contains(names, watchStopTakenFileName) {
				t.Errorf("state dir = %v; want no %s left", names, watchStopTakenFileName)
			}
			if got := f.stderr.String(); got != "" {
				t.Errorf("stderr = %q; want nothing", got)
			}
		})
	}
}

// setAfterStopRequestTaken sets the afterStopRequestTaken seam for the rest
// of the test.
func setAfterStopRequestTaken(t *testing.T, fn func()) {
	t.Helper()
	orig := afterStopRequestTaken
	afterStopRequestTaken = fn
	t.Cleanup(func() { afterStopRequestTaken = orig })
}

// Cross-review cycle 1 #2: the start takes an old stop request with one
// rename, so a request a concurrent --stop writes after the take is not
// removed with the old one. The seam writes the new request in that
// window: the old request is gone, and the new one, later than the start,
// is still there after the first cycle and ends the run then.
func TestRunWatchAll_AStopRequestWrittenWhileTheStartTakesAnOldOneIsKept(t *testing.T) {
	f := newWatchAllFixture(t)
	if err := WriteWatchStopRequest(f.dir, f.clk.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	newAt := f.clk.Now().Add(time.Second)
	var seamCalls int
	setAfterStopRequestTaken(t, func() {
		seamCalls++
		if err := WriteWatchStopRequest(f.dir, newAt); err != nil {
			t.Error(err)
		}
	})
	var cycles int
	var atFirstCycle *WatchStopRequest
	hooks := WatchHooks{OnCycle: func(n int, _ time.Time) {
		cycles = n
		if n == 1 {
			atFirstCycle, _ = ReadWatchStopRequest(f.dir)
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), f.params(3), hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if seamCalls != 1 {
		t.Fatalf("the seam ran %d times; want once, after the old request was taken", seamCalls)
	}
	if want := newAt.Format(time.RFC3339Nano); atFirstCycle == nil || atFirstCycle.RequestedAt != want {
		t.Errorf("stop request after the first cycle = %+v; want the one written during the take, requested_at %s", atFirstCycle, want)
	}
	if cycles != 1 {
		t.Errorf("ran %d cycles; want 1 (the request written during the take is followed)", cycles)
	}
	if _, err := ReadWatchStopRequest(f.dir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stop request after the watch ended: %v; want it removed", err)
	}
	if names := dirEntries(t, f.dir); slices.Contains(names, watchStopTakenFileName) {
		t.Errorf("state dir = %v; want no %s left", names, watchStopTakenFileName)
	}
}

// A request later than the start that cannot be put back after the take
// (here: a directory now stands at watch-stop.json) is still followed after
// the first cycle (stopTaken), with one line on stderr for the put-back.
func TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed(t *testing.T) {
	f := newWatchAllFixture(t)
	if err := WriteWatchStopRequest(f.dir, f.clk.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	setAfterStopRequestTaken(t, func() {
		if err := os.MkdirAll(filepath.Join(WatchStopPathIn(f.dir), "child"), 0o755); err != nil {
			t.Error(err)
		}
	})
	var cycles int
	hooks := WatchHooks{OnCycle: func(n int, _ time.Time) { cycles = n }}
	if err := runWatchAll(t, f.o, context.Background(), f.params(3), hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if cycles != 1 {
		t.Errorf("ran %d cycles; want 1 (the request taken at the start is followed)", cycles)
	}
	if want := "watchdog: put the stop request back: "; strings.Count(f.stderr.String(), want) != 1 {
		t.Errorf("stderr = %q; want one line %q", f.stderr.String(), want)
	}
	if names := dirEntries(t, f.dir); slices.Contains(names, watchStopTakenFileName) {
		t.Errorf("state dir = %v; want no %s left", names, watchStopTakenFileName)
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

// Cross-review cycle 1 #1: however the run ends after its first cycle,
// hooks.BeforeStop runs while the watch still holds watch.lock. Plan
// 2026-10-12-org-watch-gate (F9, AC13) changed what the mark says then: it
// is stopping already when BeforeStop is called, not running. While
// BeforeStop blocks, ProbeWatch says stopping, so WatchAcceptsWork is false,
// a second AcquireWatchLock and a second RunWatchAll are refused (the
// refused one calls no BeforeStop), and the stopping mark's ts is rewritten
// every interval. Once BeforeStop returns, the mark is stopped and the lock
// free.
func TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysStopping(t *testing.T) {
	cases := []struct {
		name string
		// end ends a run with no Cycles after its first cycle; nil runs
		// Cycles 1.
		end func(f *watchAllFixture, cancel context.CancelFunc)
	}{
		{name: "Cycles reached"},
		{name: "context cancelled", end: func(_ *watchAllFixture, cancel context.CancelFunc) { cancel() }},
		{name: "stop request", end: func(f *watchAllFixture, _ context.CancelFunc) {
			_ = WriteWatchStopRequest(f.dir, f.clk.Now().Add(time.Second))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := f.params(1)
			if tc.end != nil {
				p.Cycles = 0
			}
			entered := make(chan struct{})
			release := make(chan struct{})
			var beforeStopCalls, cycles int
			// atEntry is the mark BeforeStop finds when it is called; the
			// close of entered publishes it to the test goroutine.
			var atEntry *WatchAlive
			var atEntryErr error
			hooks := WatchHooks{
				OnCycle: func(n int, _ time.Time) {
					cycles = n
					if n == 1 && tc.end != nil {
						tc.end(f, cancel)
					}
				},
				BeforeStop: func() {
					beforeStopCalls++
					atEntry, atEntryErr = ReadWatchAlive(f.dir)
					close(entered)
					<-release
				},
			}
			done := make(chan error, 1)
			go func() { done <- f.o.RunWatchAll(ctx, p, hooks) }()
			select {
			case <-entered:
			case err := <-done:
				t.Fatalf("RunWatchAll returned %v without calling BeforeStop", err)
			case <-time.After(10 * time.Second):
				t.Fatal("RunWatchAll did not call BeforeStop within 10s")
			}

			if atEntryErr != nil || atEntry.State != WatchAliveStopping || atEntry.Once != (p.Cycles > 0) {
				t.Errorf("mark when BeforeStop is called = %+v (%v); want stopping, once %t", atEntry, atEntryErr, p.Cycles > 0)
			}
			probe := ProbeWatch(f.dir, f.clk.Now(), f.o.Config.DeadmanMinutes)
			if probe.State != WatchLivenessStopping {
				t.Errorf("ProbeWatch during BeforeStop = %+v; want stopping", probe)
			}
			if WatchAcceptsWork(probe) {
				t.Errorf("WatchAcceptsWork during BeforeStop = true for %+v; want false", probe)
			}
			if _, err := AcquireWatchLock(f.dir, 20*time.Millisecond); !errors.Is(err, ErrWatchRunning) {
				t.Errorf("AcquireWatchLock during BeforeStop = %v; want ErrWatchRunning", err)
			}
			err := f.o.RunWatchAll(context.Background(), f.params(1), WatchHooks{BeforeStop: func() {
				t.Error("the refused watch called BeforeStop")
			}})
			if !errors.Is(err, ErrWatchRunning) {
				t.Errorf("a second RunWatchAll during BeforeStop = %v; want ErrWatchRunning", err)
			}
			f.clk.Advance(time.Minute)
			refreshed := f.clk.Now().Format(time.RFC3339)
			for deadline := time.Now().Add(10 * time.Second); ; {
				mark := f.mark(t)
				if mark.State == WatchAliveStopping && mark.TS == refreshed {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("mark during BeforeStop = %+v; want stopping with ts rewritten to %s", mark, refreshed)
				}
				time.Sleep(time.Millisecond)
			}

			close(release)
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("RunWatchAll: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("RunWatchAll did not return after BeforeStop returned")
			}
			if beforeStopCalls != 1 || cycles != 1 {
				t.Errorf("BeforeStop called %d times after %d cycles; want once after 1", beforeStopCalls, cycles)
			}
			if mark := f.mark(t); mark.State != WatchAliveStopped || mark.StoppedAt != refreshed {
				t.Errorf("mark after BeforeStop = %+v; want stopped with stopped_at %s", mark, refreshed)
			}
			if got := ProbeWatch(f.dir, f.clk.Now(), f.o.Config.DeadmanMinutes).State; got != WatchLivenessNotRunning {
				t.Errorf("ProbeWatch after BeforeStop = %q; want not_running", got)
			}
		})
	}
}

// A panic in BeforeStop still ends the run the way every end does: the mark
// is stopped and the lock free, and the panic goes on out of RunWatchAll.
func TestRunWatchAll_APanicInBeforeStopStillWritesTheStoppedMarkAndReleasesTheLock(t *testing.T) {
	f := newWatchAllFixture(t)
	hooks := WatchHooks{BeforeStop: func() { panic("before stop") }}
	func() {
		defer func() {
			if r := recover(); r != "before stop" {
				t.Errorf("recovered %v; want the panic of BeforeStop", r)
			}
		}()
		_ = runWatchAll(t, f.o, context.Background(), f.params(1), hooks)
		t.Error("RunWatchAll returned; want the panic of BeforeStop to go on")
	}()
	if mark := f.mark(t); mark.State != WatchAliveStopped || mark.StoppedAt == "" {
		t.Errorf("mark after the panic = %+v; want stopped", mark)
	}
	if got := ProbeWatch(f.dir, f.clk.Now(), f.o.Config.DeadmanMinutes).State; got != WatchLivenessNotRunning {
		t.Errorf("ProbeWatch after the panic = %q; want not_running", got)
	}
}

// The mark refresher of the drain ends before the lock is released, so no
// rewrite of the old watch lands after it. A new watch's running mark is
// written right after RunWatchAll returns (as if a new watch took the lock
// at once); it must still be there 500 ms later. The clock seam holds the
// refresher's first rewrite after BeforeStop starts for 300 ms, so a
// refresher that is not waited for writes after the return. Without -race
// this is the only test that sees a drain that does not wait for it.
func TestRunWatchAll_NoMarkRewriteOfTheDrainLandsAfterTheLockIsReleased(t *testing.T) {
	f := newWatchAllFixture(t)
	var armed atomic.Bool
	held := make(chan struct{})
	var once sync.Once
	f.o.Now = func() time.Time {
		if armed.Load() {
			first := false
			once.Do(func() { first = true })
			if first {
				close(held)
				time.Sleep(300 * time.Millisecond)
			}
		}
		return f.clk.Now()
	}
	hooks := WatchHooks{BeforeStop: func() {
		armed.Store(true)
		select {
		case <-held:
		case <-time.After(10 * time.Second):
			t.Error("the refresher did not rewrite the mark within 10s of BeforeStop")
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	next := WatchAlive{State: WatchAliveRunning, PID: 4343, StartedAt: f.clk.Now().UTC().Format(time.RFC3339Nano), TS: "2099-01-01T00:00:00Z", IntervalSeconds: 30}
	if err := writeWatchAlive(f.dir, next); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	got := f.mark(t)
	if got.PID != 4343 || got.State != WatchAliveRunning || got.TS != next.TS {
		data, _ := os.ReadFile(WatchAlivePathIn(f.dir))
		t.Errorf("mark 500ms after the next watch wrote its own = %s; want the next watch's (pid 4343): the old watch rewrote it after releasing the lock", data)
	}
}

// A take of the stop request that fails for a reason other than "no
// request" says so on stderr once and the run goes on. A non-empty directory
// at .watch-stop.json.taken makes the rename of an old watch-stop.json fail
// (EISDIR on darwin, ENOTEMPTY or EISDIR on Linux). The old request stays
// where it was and is not followed.
func TestRunWatchAll_ATakeOfTheStopRequestThatFailsIsSaidOnStderrAndTheRunGoesOn(t *testing.T) {
	f := newWatchAllFixture(t)
	if err := os.MkdirAll(filepath.Join(f.dir, watchStopTakenFileName, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteWatchStopRequest(f.dir, f.clk.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	var cycles int
	hooks := WatchHooks{OnCycle: func(n int, _ time.Time) { cycles = n }}
	if err := runWatchAll(t, f.o, context.Background(), f.params(3), hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if cycles != 3 {
		t.Errorf("ran %d cycles; want 3 (the old request is not followed)", cycles)
	}
	if want := "watchdog: org: watch: take the stop request: "; strings.Count(f.stderr.String(), want) != 1 {
		t.Errorf("stderr = %q; want one line %q", f.stderr.String(), want)
	}
	if _, err := ReadWatchStopRequest(f.dir); err != nil {
		t.Errorf("the old request after the run: %v; want it left in place (the take failed)", err)
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

// The tests below are for step 3 of RunWatchAll's cycle, the push wake of
// the director (plan 2026-10-18-org-director-wake, D6, D10, AC5, and
// AC7). The wake_cmd is a writeWakeScript script in its own directory, given
// to the watch as DirectorWakeDir.

// useWakeCmd sets f's wake_cmd to ["./wake.sh", args...], writes cmdDir's
// wake.sh with tail after its record of the run, and returns the params of
// cycles cycles with cmdDir as DirectorWakeDir, and the mark the runs are
// recorded in.
func (f *watchAllFixture) useWakeCmd(t *testing.T, cycles int, tail string, args ...string) (WatchAllParams, string) {
	t.Helper()
	cmdDir := t.TempDir()
	mark := filepath.Join(t.TempDir(), "runs")
	writeWakeScript(t, filepath.Join(cmdDir, "wake.sh"), mark, tail)
	f.o.Config.Director.WakeCmd = append([]string{"./wake.sh"}, args...)
	p := f.params(cycles)
	p.DirectorWakeDir = cmdDir
	return p, mark
}

// directorStore is the director store of f's ledger, on f's clock.
func (f *watchAllFixture) directorStore() *DirectorStore {
	s := NewDirectorStore(f.dir)
	s.Now = f.clk.Now
	return s
}

// D6: in a cycle the inbox deadlines are swept before the director is
// woken, and the director is woken before any org's pulse: when wake_cmd
// runs, the item past its ack deadline is already in escalations.jsonl, and
// at the first herdr call of the pulse wake_cmd has run. The deadline send
// does not keep the item from the wake: it is still open and not woken.
func TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	snapshot := filepath.Join(t.TempDir(), "escalations-at-the-wake")
	p, mark := f.useWakeCmd(t, 1, "cat "+shellQuote(f.o.EscalationsPath)+" > "+shellQuote(snapshot), "{text}")
	mustAttachDirector(t, f.directorStore(), DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull, DirectorWakePush}})
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now().Add(-11*time.Minute), "org-a")
	var herdrCalls, runsAtFirstHerdrCall int
	f.o.Herdr = &beforeAgentGetHerdr{fakeWatchHerdr: f.h, before: func() {
		herdrCalls++
		if herdrCalls == 1 {
			runsAtFirstHerdrCall = len(readWakeRuns(t, mark))
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if want := []string{e1 + " " + InboxReasonAckDeadline}; !slices.Equal(escalationReasons(t, snapshot), want) {
		t.Fatalf("escalations.jsonl when wake_cmd ran = %v; want %v already sent", escalationReasons(t, snapshot), want)
	}
	if herdrCalls == 0 || runsAtFirstHerdrCall != 1 {
		t.Fatalf("wake_cmd runs at the first of %d herdr calls = %d; want 1", herdrCalls, runsAtFirstHerdrCall)
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), e1); !item.Notified || !item.Woken {
		t.Fatalf("item %s = %+v; want notified and woken", e1, item)
	}
}

// AC5: a registration whose wake has no push is not woken, by wake_cmd or by
// herdr, cycle after cycle, and nothing is said about it; the same ledger
// with a push registration is woken (the control).
func TestRunWatchAll_DoesNotWakeAPullDirector(t *testing.T) {
	t.Run("a promoted session", func(t *testing.T) {
		f := newWatchAllFixture(t)
		p, mark := f.useWakeCmd(t, 3, "exit 0", "{text}")
		s := f.directorStore()
		mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull}})
		e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-a")
		if err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{}); err != nil {
			t.Fatalf("RunWatchAll: %v", err)
		}
		if runs := readWakeRuns(t, mark); runs != nil || f.stderr.String() != "" {
			t.Fatalf("a pull director: wake_cmd runs %+v, stderr %q; want neither", runs, f.stderr.String())
		}
		if _, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude, Replace: true, Wake: []string{DirectorWakePush}}); err != nil {
			t.Fatal(err)
		}
		p.Cycles = 1
		if err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{}); err != nil {
			t.Fatalf("RunWatchAll: %v", err)
		}
		if runs := readWakeRuns(t, mark); len(runs) != 1 || !slices.Equal(runs[0].args, []string{DirectorWakeText([]string{e1})}) {
			t.Fatalf("a push director: wake_cmd runs %+v; want one for %s", runs, e1)
		}
	})

	for _, wake := range []string{DirectorWakePull, DirectorWakePush} {
		t.Run("a headless director in herdr with wake "+wake, func(t *testing.T) {
			f := newWatchAllFixture(t)
			s := f.directorStore()
			h := testHeadlessAttach()
			h.Wake = []string{wake}
			reg, err := s.AttachHeadless(h)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.RecordWorkspace(reg.ID, "w7", testDirectorWho); err != nil {
				t.Fatal(err)
			}
			if err := s.RecordPane(reg.ID, "w7:p1", testDirectorWho); err != nil {
				t.Fatal(err)
			}
			if _, err := s.MarkLaunched(reg.ID, testDirectorWho); err != nil {
				t.Fatal(err)
			}
			e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-a")
			if err := runWatchAll(t, f.o, context.Background(), f.params(2), WatchHooks{}); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}
			f.h.mu.Lock()
			texts, keys := slices.Clone(f.h.paneSendTextCall), slices.Clone(f.h.sendKeysCalls)
			f.h.mu.Unlock()
			if len(texts) != 0 || len(keys) != 0 {
				t.Fatalf("herdr send-text %+v, send-keys %+v; want none (this fake has no pane to confirm)", texts, keys)
			}
			// fakeWatchHerdr has no pane: a push wake looks for it and fails.
			tried := strings.Contains(f.stderr.String(), "watchdog: director d1 not woken for "+e1+`: herdr has no pane "w7:p1", so nothing was typed into it`)
			if tried != (wake == DirectorWakePush) {
				t.Fatalf("wake %s: stderr = %q; want the herdr wake tried only with push", wake, f.stderr.String())
			}
			if wake == DirectorWakePull && f.stderr.String() != "" {
				t.Fatalf("wake pull: stderr = %q; want nothing", f.stderr.String())
			}
		})
	}
}

// A director.jsonl that cannot be read (here: a directory in its place)
// skips the director check and the wake with one line each per cycle, and
// no director-alive.json is written; the deadlines and the pulse still run.
func TestRunWatchAll_ADirectorFileThatCannotBeReadSkipsOnlyTheWake(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	p, mark := f.useWakeCmd(t, 2, "exit 0", "{text}")
	if err := os.Mkdir(f.directorStore().Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now().Add(-11*time.Minute), "org-a")
	if err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if n := strings.Count(f.stderr.String(), "watchdog: director wake skipped this cycle: org: director wake: "); n != 2 {
		t.Fatalf("stderr = %q; want the wake line once per cycle", f.stderr.String())
	}
	if n := strings.Count(f.stderr.String(), "watchdog: director check skipped this cycle: org: director: read "); n != 2 {
		t.Fatalf("stderr = %q; want the check line once per cycle", f.stderr.String())
	}
	if got := fileOrAbsent(t, DirectorAlivePathIn(f.dir)); got != "<absent>" {
		t.Fatalf("director-alive.json = %q; want none", got)
	}
	if runs := readWakeRuns(t, mark); runs != nil {
		t.Fatalf("wake_cmd ran %+v", runs)
	}
	if got, want := escalationReasons(t, f.o.EscalationsPath), []string{e1 + " " + InboxReasonAckDeadline}; !slices.Equal(got, want) {
		t.Fatalf("escalations.jsonl = %v; want %v", got, want)
	}
	if s := readWatchStatusFile(t, f.dir, "org-a"); s.Cycles != 2 {
		t.Fatalf("org-a status = %+v; want 2 cycles", s)
	}
}

// AC7 and D10: a push wake does not stop the ack deadline. An item the
// director was woken for and one whose wake failed both reach the human at
// their ack deadlines when neither is acked.
func TestRunWatchAll_WokenAndNotWokenItemsBothReachTheHumanAtTheAckDeadline(t *testing.T) {
	f := newWatchAllFixture(t)
	p, mark := f.useWakeCmd(t, 3, "exit 0", "{text}")
	mustAttachDirector(t, f.directorStore(), DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePush}})
	t0 := f.clk.Now()
	e1 := escalateInboxAt(t, f.o.Inbox, t0, "org-a")
	var e2 string
	hooks := WatchHooks{OnCycle: func(n int, _ time.Time) {
		switch n {
		case 1:
			writeWakeScript(t, filepath.Join(p.DirectorWakeDir, "wake.sh"), mark, "exit 1")
			f.clk.Advance(time.Minute)
			e2 = escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-b")
		case 2:
			f.clk.Advance(11 * time.Minute)
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), p, hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	if a, b := inboxItemByID(t, inbox, e1), inboxItemByID(t, inbox, e2); !a.Woken || b.Woken {
		t.Fatalf("woken: %s %v, %s %v; want only %s", e1, a.Woken, e2, b.Woken, e1)
	}
	want := []string{e1 + " " + InboxReasonAckDeadline, e2 + " " + InboxReasonAckDeadline}
	if got := escalationReasons(t, f.o.EscalationsPath); !slices.Equal(got, want) {
		t.Fatalf("escalations.jsonl = %v; want both at the ack deadline %v", got, want)
	}
	if n := len(f.notifications()); n != 2 {
		t.Fatalf("desktop notified %d times; want 2", n)
	}
	// e2's wake failed at t0+1m and was tried again at t0+12m.
	if runs := readWakeRuns(t, mark); len(runs) != 3 {
		t.Fatalf("wake_cmd ran %d times; want 3 (e1, then e2 twice)", len(runs))
	}
	if n := strings.Count(f.stderr.String(), "watchdog: director d1 not woken for "+e2+": "); n != 2 {
		t.Fatalf("stderr = %q; want the failure line for %s twice", f.stderr.String(), e2)
	}
}

// AC7 across cycles: RunWatchAll keeps one DirectorWakeMemory for the whole
// run, so a wake_cmd that fails runs once in 3 cycles a minute apart, not
// once per cycle, and its failure line is written once.
func TestRunWatchAll_HoldsAFailedWakeBackAcrossCycles(t *testing.T) {
	f := newWatchAllFixture(t)
	p, mark := f.useWakeCmd(t, 3, "exit 1", "{text}")
	mustAttachDirector(t, f.directorStore(), DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePush}})
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-a")
	hooks := WatchHooks{OnCycle: func(int, time.Time) { f.clk.Advance(time.Minute) }}
	if err := runWatchAll(t, f.o, context.Background(), p, hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if runs := readWakeRuns(t, mark); len(runs) != 1 {
		t.Fatalf("wake_cmd ran %d times in 3 cycles a minute apart; want 1", len(runs))
	}
	if n := strings.Count(f.stderr.String(), "watchdog: director d1 not woken for "+e1+": "); n != 1 {
		t.Fatalf("stderr = %q; want one failure line", f.stderr.String())
	}
}

// AC7 across cycles: the line for a push registration with no way to send
// (a promoted session while wake_cmd is empty) is written once over the 3
// cycles of one watch run, not once per cycle.
func TestRunWatchAll_SaysTheNoSenderLineOnceAcrossCycles(t *testing.T) {
	f := newWatchAllFixture(t)
	p := f.params(3)
	p.DirectorWakeDir = t.TempDir()
	mustAttachDirector(t, f.directorStore(), DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePush}})
	escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-a")
	if err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if n := strings.Count(f.stderr.String(), "watchdog: director d1 is not woken: its wake has push, but [org.director].wake_cmd is empty"); n != 1 {
		t.Fatalf("stderr = %q; want the no-sender line once", f.stderr.String())
	}
}

// AC5, end to end in the org layer: with a director attached with wake
// pull,push and a watch that accepts work, an escalate is left to the
// director, `wait --inbox` returns it, the next cycle runs wake_cmd once for
// it, and the cycle after that runs nothing.
func TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem(t *testing.T) {
	f := newWatchAllFixture(t)
	f.o.Config.Director.Enabled = true
	f.o.EscalateWatch = acceptingWatch
	p, mark := f.useWakeCmd(t, 2, "exit 0", "{target}", "{text}")
	reg := mustAttachDirector(t, f.directorStore(), DirectorAttach{
		Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull, DirectorWakePush}, WakeTarget: "orca-term-1",
	})
	var banner bytes.Buffer
	res := f.o.Escalate(EscalateParams{OrgID: "org-a", Text: "TYPE: QUESTION\n\nMARKER-body-51c2 which base?", Banner: &banner})
	if res.Err != nil || !res.Deferred {
		t.Fatalf("Escalate = %+v; want the item left to the director", res)
	}
	f.o.InboxPollInterval = time.Millisecond
	open, err := f.o.WaitInbox(time.Second)
	if err != nil || len(open) != 1 || open[0].ID != res.ID {
		t.Fatalf("WaitInbox = %+v, %v; want %s", open, err, res.ID)
	}
	var runsAfterCycle []int
	hooks := WatchHooks{OnCycle: func(int, time.Time) { runsAfterCycle = append(runsAfterCycle, len(readWakeRuns(t, mark))) }}
	if err := runWatchAll(t, f.o, context.Background(), p, hooks); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if !slices.Equal(runsAfterCycle, []int{1, 1}) {
		t.Fatalf("wake_cmd runs after each cycle = %v; want [1 1]", runsAfterCycle)
	}
	runs := readWakeRuns(t, mark)
	if want := []string{"orca-term-1", DirectorWakeText([]string{res.ID})}; !slices.Equal(runs[0].args, want) || strings.Contains(runs[0].args[1], "MARKER") {
		t.Fatalf("wake_cmd args = %q; want %q, with nothing of the body", runs[0].args, want)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), res.ID)
	if last := item.Events[len(item.Events)-1]; !item.Woken || last.DirectorID != reg.ID || last.Via != InboxWokenViaWakeCmd || item.State != InboxStateOpen {
		t.Fatalf("item %s = %+v; want open and woken by %s via wake_cmd", res.ID, item, reg.ID)
	}
	if f.stderr.String() != "" || len(f.notifications()) != 0 {
		t.Fatalf("stderr %q, notifications %q; want neither before the ack deadline", f.stderr.String(), f.notifications())
	}
}

// The tests below are for step 2 of RunWatchAll's cycle, the director check
// and its mark (plan 2026-10-18-org-director-liveness, D4, D5, AC4, and
// AC5). herdr is a cycleHerdr; a ps read, when a test needs one, is the
// Org's DirectorProcessReader, never ps.

// cycleHerdr is a fakeHerdr that also runs herdr agent get and logs every
// call in order in calls: an agent get as "agent_get <target>", next to the
// fakeHerdr's own calls (the gets, the closes, send-text, and send-keys). An
// agent get answers agentGetErr[target], or "ok" (herdr has the agent).
// before, when set, runs ahead of each agent get and send-text with the
// name the call is logged under.
type cycleHerdr struct {
	*fakeHerdr
	agentGetErr map[string]error
	before      func(call string)
}

func (h *cycleHerdr) AgentGet(_ context.Context, target string) (string, error) {
	call := "agent_get " + target
	if h.before != nil {
		h.before(call)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, call)
	if err := h.agentGetErr[target]; err != nil {
		return "", err
	}
	return "ok", nil
}

func (h *cycleHerdr) PaneSendText(ctx context.Context, paneID, text string) error {
	if h.before != nil {
		h.before("pane_send_text")
	}
	return h.fakeHerdr.PaneSendText(ctx, paneID, text)
}

func (h *cycleHerdr) log() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.calls)
}

// headlessInHerdr records d1 in f's director.jsonl, a headless director in
// herdr with wake wake, running in workspace w7 and pane w7:p1, and gives f
// a cycleHerdr that holds that pane in a tab and a workspace labelled for
// its token, so the herdr wake can type into it. Spawn the org's seats
// before: their calls would go to the new herdr. It returns the herdr and
// the registration as recorded.
func (f *watchAllFixture) headlessInHerdr(t *testing.T, wake string) (*cycleHerdr, DirectorRegistration) {
	t.Helper()
	s := f.directorStore()
	a := testHeadlessAttach()
	a.Wake = []string{wake}
	reg, err := s.AttachHeadless(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordWorkspace(reg.ID, "w7", testDirectorWho); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordPane(reg.ID, "w7:p1", testDirectorWho); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MarkLaunched(reg.ID, testDirectorWho); err != nil {
		t.Fatal(err)
	}
	h := &cycleHerdr{fakeHerdr: &fakeHerdr{}, agentGetErr: map[string]error{}}
	setLabel(&h.paneTabs, "w7:p1", fakeTabID("w7:p1"))
	setLabel(&h.paneWorkspaces, "w7:p1", "w7")
	setLabel(&h.tabLabels, fakeTabID("w7:p1"), DirectorTabLabel(reg.Token))
	setLabel(&h.workspaceLabels, "w7", DirectorWorkspaceLabel(reg.Token))
	f.o.Herdr = h
	f.o.SendEnterDelay = time.Millisecond
	cur := mustReadDirector(t, s).Current
	if cur == nil || cur.State != DirectorStateRunning || cur.PaneID != "w7:p1" {
		t.Fatalf("registration = %+v, want d1 running in w7:p1", cur)
	}
	return h, *cur
}

// directorAgentNotFound is the error the driver's AgentGet returns for
// herdr's agent_not_found reply about the agent of token (see
// checkAgentNotFound).
func directorAgentNotFound(token string) error {
	return fmt.Errorf("%w (%w)", errors.New("herdr: exit status 1"), driver.NewHerdrError(driver.HerdrCodeAgentNotFound, "agent "+DirectorAgentName(token)+" not found"))
}

// readDirectorAliveMark reads f's director-alive.json, which must be there.
func (f *watchAllFixture) readDirectorAliveMark(t *testing.T) DirectorAlive {
	t.Helper()
	mark, err := ReadDirectorAlive(f.dir)
	if err != nil || mark == nil {
		t.Fatalf("ReadDirectorAlive = %+v, %v; want a mark", mark, err)
	}
	return *mark
}

// AC4 and D4: one cycle runs the inbox deadlines, then the director check,
// then the wake, then the pulse. When the check's herdr agent get runs, the
// item past its ack deadline is already in escalations.jsonl; when the wake
// types its line, director-alive.json already holds the check; the seat's
// agent get of the pulse comes after the wake's send-keys. The mark holds the
// registration's id and attached_at, the state and the how, and the cycle's
// time; director.jsonl is byte for byte as it was.
func TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	h, reg := f.headlessInHerdr(t, DirectorWakePush)
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now().Add(-11*time.Minute), "org-a")
	directorBefore := directorFileState(t, f.directorStore())
	checkCall := "agent_get " + DirectorAgentName(reg.Token)
	snapshot := filepath.Join(t.TempDir(), "escalations-at-the-check")
	var markAtSend *DirectorAlive
	h.before = func(call string) {
		switch call {
		case checkCall:
			if err := os.WriteFile(snapshot, []byte(fileOrAbsent(t, f.o.EscalationsPath)), 0o644); err != nil {
				t.Error(err)
			}
		case "pane_send_text":
			var err error
			if markAtSend, err = ReadDirectorAlive(f.dir); err != nil {
				t.Error(err)
			}
		}
	}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}

	log := h.log()
	wantFirst := []string{checkCall, "pane_get", "tab_get", "workspace_get", "pane_send_text", "pane_send_keys"}
	if len(log) <= len(wantFirst) || !slices.Equal(log[:len(wantFirst)], wantFirst) {
		t.Fatalf("herdr calls = %q; want %q first, then the pulse", log, wantFirst)
	}
	pulse := log[len(wantFirst):]
	if !slices.Contains(pulse, "agent_get "+herdrAgentName("org-a", "seat-a")) {
		t.Fatalf("herdr calls after the wake = %q; want the seat's agent get", pulse)
	}
	for _, call := range pulse {
		if !strings.HasPrefix(call, "agent_get ") || call == checkCall {
			t.Fatalf("herdr calls after the wake = %q; want only the pulse's agent gets", pulse)
		}
	}
	if want := []string{e1 + " " + InboxReasonAckDeadline}; !slices.Equal(escalationReasons(t, snapshot), want) {
		t.Fatalf("escalations.jsonl at the check = %v; want %v already sent", escalationReasons(t, snapshot), want)
	}
	want := DirectorAlive{ID: reg.ID, AttachedAt: reg.AttachedAt, State: DirectorCheckAlive, How: DirectorCheckHowAgentFound,
		CheckedAt: f.clk.Now().UTC().Format(time.RFC3339)}
	if markAtSend == nil || *markAtSend != want {
		t.Fatalf("director-alive.json when the wake typed = %+v; want %+v", markAtSend, want)
	}
	if got := f.readDirectorAliveMark(t); got != want || reg.AttachedAt == "" {
		t.Fatalf("director-alive.json = %+v; want %+v", got, want)
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), e1); !item.Woken || item.Events[len(item.Events)-1].Via != InboxWokenViaHerdr {
		t.Fatalf("item %s = %+v; want woken via herdr", e1, item)
	}
	if after := directorFileState(t, f.directorStore()); after != directorBefore {
		t.Fatalf("director.jsonl changed from %q to %q", directorBefore, after)
	}
	if s := readWatchStatusFile(t, f.dir, "org-a"); s.Cycles != 1 {
		t.Fatalf("org-a status = %+v; want 1 cycle", s)
	}
}

// AC4 and AC5 over 3 cycles: the check writes nothing but its mark. Whatever
// it finds, director.jsonl is byte for byte as it was, and herdr is only
// asked to get (one agent get per cycle, and the label gets of an agent herdr
// does not have): no close, no send-keys, no send-text. The mark holds the
// last cycle's check. An ended registration with wake push is not woken
// (D5): its open item gets no woken event, and the watch says so once over
// the 3 cycles.
func TestRunWatchAll_TheDirectorCheckWritesOnlyItsMark(t *testing.T) {
	for _, tc := range []struct {
		name       string
		wake       string
		herdr      func(h *cycleHerdr, reg DirectorRegistration)
		state, how string
		detail     string
	}{
		{"alive", DirectorWakePull, func(*cycleHerdr, DirectorRegistration) {}, DirectorCheckAlive, DirectorCheckHowAgentFound, ""},
		{"ended, its agent gone from its pane, with push", DirectorWakePush, func(h *cycleHerdr, reg DirectorRegistration) {
			h.agentGetErr[DirectorAgentName(reg.Token)] = directorAgentNotFound(reg.Token)
		}, DirectorCheckEnded, DirectorCheckHowAgentEnded, ""},
		{"ended, its pane closed", DirectorWakePull, func(h *cycleHerdr, reg DirectorRegistration) {
			h.agentGetErr[DirectorAgentName(reg.Token)] = directorAgentNotFound(reg.Token)
			delete(h.paneTabs, "w7:p1")
		}, DirectorCheckEnded, DirectorCheckHowPaneClosed, ""},
		{"unknown, herdr does not answer", DirectorWakePull, func(h *cycleHerdr, reg DirectorRegistration) {
			h.agentGetErr[DirectorAgentName(reg.Token)] = errors.New("herdr: no server is running")
		}, DirectorCheckUnknown, DirectorCheckHowAgentGetFailed, "herdr: no server is running"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			h, reg := f.headlessInHerdr(t, tc.wake)
			tc.herdr(h, reg)
			e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-a")
			directorBefore := directorFileState(t, f.directorStore())
			hooks := WatchHooks{OnCycle: func(n int, _ time.Time) {
				if n < 3 {
					f.clk.Advance(time.Minute)
				}
			}}
			if err := runWatchAll(t, f.o, context.Background(), f.params(3), hooks); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}
			if after := directorFileState(t, f.directorStore()); after != directorBefore {
				t.Fatalf("director.jsonl changed from %q to %q", directorBefore, after)
			}
			log := h.log()
			agentGets := 0
			for _, call := range log {
				switch call {
				case "agent_get " + DirectorAgentName(reg.Token):
					agentGets++
				case "pane_get", "tab_get", "workspace_get":
				default:
					t.Fatalf("herdr calls = %q; want only gets", log)
				}
			}
			if agentGets != 3 {
				t.Fatalf("herdr calls = %q; want one agent get per cycle", log)
			}
			mark := f.readDirectorAliveMark(t)
			want := DirectorAlive{ID: reg.ID, AttachedAt: reg.AttachedAt, State: tc.state, How: tc.how, CheckedAt: f.clk.Now().UTC().Format(time.RFC3339), Detail: mark.Detail}
			if mark != want || !strings.Contains(mark.Detail, tc.detail) || (tc.detail == "") != (mark.Detail == "") {
				t.Fatalf("director-alive.json = %+v; want %+v with a detail holding %q", mark, want, tc.detail)
			}
			if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), e1); item.Woken {
				t.Fatalf("item %s = %+v; want it not woken", e1, item)
			}
			endedLine := "watchdog: director d1 is not woken: the watch's check found it ended (" + tc.how + ") (said once)"
			wantLines := 0
			if tc.wake == DirectorWakePush {
				wantLines = 1
			}
			if n := strings.Count(f.stderr.String(), endedLine); n != wantLines || strings.Count(f.stderr.String(), "watchdog: director") != wantLines {
				t.Fatalf("stderr = %q; want the ended line %d time(s) and no other director line", f.stderr.String(), wantLines)
			}
		})
	}
}

// AC4: with no current headless registration (no director.jsonl, a promoted
// session's, one stopped by stop, and one whose stopped was written without
// its detached) the check runs nothing and writes no mark: none appears, and
// a mark an earlier registration left stays byte for byte as it was.
func TestRunWatchAll_WritesNoDirectorMarkWithoutACurrentHeadlessRegistration(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, f *watchAllFixture)
	}{
		{"no director.jsonl", func(*testing.T, *watchAllFixture) {}},
		{"a promoted session", func(t *testing.T, f *watchAllFixture) {
			mustAttachDirector(t, f.directorStore(), DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull}})
		}},
		{"a headless registration stopped", func(t *testing.T, f *watchAllFixture) {
			_, reg := f.headlessInHerdr(t, DirectorWakePull)
			if _, err := f.directorStore().MarkStopped(reg.ID, testDirectorWho); err != nil {
				t.Fatal(err)
			}
		}},
		{"a current headless registration stopped", func(t *testing.T, f *watchAllFixture) {
			writeDirectorEvents(t, f.directorStore(), append(checkHerdrRunning(), checkEvent("d1", DirectorEventStopped, nil))...)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			h := &cycleHerdr{fakeHerdr: &fakeHerdr{}, agentGetErr: map[string]error{}}
			f.o.Herdr = h
			tc.setup(t, f)
			h, _ = f.o.Herdr.(*cycleHerdr)
			h.mu.Lock()
			h.calls = nil
			h.mu.Unlock()
			if err := runWatchAll(t, f.o, context.Background(), f.params(2), WatchHooks{}); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}
			if got := fileOrAbsent(t, DirectorAlivePathIn(f.dir)); got != "<absent>" {
				t.Fatalf("director-alive.json = %q; want none", got)
			}
			prior := DirectorAlive{ID: "d9", AttachedAt: "2026-10-01T00:00:00Z", State: DirectorCheckEnded, How: DirectorCheckHowPaneClosed, CheckedAt: "2026-10-01T00:01:00Z"}
			if err := writeDirectorAlive(f.dir, prior); err != nil {
				t.Fatal(err)
			}
			before := fileOrAbsent(t, DirectorAlivePathIn(f.dir))
			if err := runWatchAll(t, f.o, context.Background(), f.params(2), WatchHooks{}); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}
			if after := fileOrAbsent(t, DirectorAlivePathIn(f.dir)); after != before {
				t.Fatalf("director-alive.json changed from %q to %q", before, after)
			}
			if log := h.log(); len(log) != 0 {
				t.Fatalf("herdr calls = %q; want none", log)
			}
			if strings.Contains(f.stderr.String(), "director check") {
				t.Fatalf("stderr = %q; want no check line", f.stderr.String())
			}
		})
	}
}

// AC4: a check that fails (here: director.lock cannot be opened, so the read
// under it fails, while director.jsonl itself reads) is one line per cycle
// and writes no mark, and the wake and the pulse go on: wake_cmd runs once
// for the open item, and the org's pulse runs every cycle.
func TestRunWatchAll_ADirectorCheckThatFailsIsOneLineAndTheWakeAndThePulseGoOn(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	p, mark := f.useWakeCmd(t, 2, "exit 0", "{target}", "{text}")
	attached := checkAttachedAs("d1", directorTestToken, DirectorLaunchCmd, true)
	attached.Wake = []string{DirectorWakePush}
	writeDirectorEvents(t, f.directorStore(), attached,
		checkEvent("d1", DirectorEventLaunched, func(ev *DirectorEvent) { ev.PID, ev.StartTime = checkAgentPID, checkAgentStart }))
	f.o.DirectorProcessReader = func(_ context.Context, pid int) (DirectorProcess, error) {
		if pid != checkAgentPID {
			t.Errorf("ps read pid %d", pid)
		}
		return DirectorProcess{State: DirectorProcessAlive, StartTime: checkAgentStart}, nil
	}
	if err := os.Mkdir(filepath.Join(f.dir, directorLockFile), 0o755); err != nil {
		t.Fatal(err)
	}
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-a")
	if err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if n := strings.Count(f.stderr.String(), "watchdog: director check skipped this cycle: "); n != 2 || strings.Count(f.stderr.String(), "\n") != 2 {
		t.Fatalf("stderr = %q; want the check line once per cycle and nothing else", f.stderr.String())
	}
	if got := fileOrAbsent(t, DirectorAlivePathIn(f.dir)); got != "<absent>" {
		t.Fatalf("director-alive.json = %q; want none", got)
	}
	runs := readWakeRuns(t, mark)
	if len(runs) != 1 || !slices.Equal(runs[0].args, []string{"director_" + directorTestToken, DirectorWakeText([]string{e1})}) {
		t.Fatalf("wake_cmd runs = %+v; want one for %s", runs, e1)
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), e1); !item.Woken {
		t.Fatalf("item %s = %+v; want woken", e1, item)
	}
	if s := readWatchStatusFile(t, f.dir, "org-a"); s.Cycles != 2 {
		t.Fatalf("org-a status = %+v; want 2 cycles", s)
	}
}

// AC4 and AC5: a mark that cannot be written (here: a non-empty directory
// where it goes) is one line per cycle; the wake still goes by the check, so
// an ended registration with push is still not woken, and the pulse runs
// every cycle.
func TestRunWatchAll_ADirectorMarkThatCannotBeWrittenIsOneLineAndTheCycleGoesOn(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	h, reg := f.headlessInHerdr(t, DirectorWakePush)
	h.agentGetErr[DirectorAgentName(reg.Token)] = directorAgentNotFound(reg.Token)
	if err := os.MkdirAll(filepath.Join(DirectorAlivePathIn(f.dir), "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	e1 := escalateInboxAt(t, f.o.Inbox, f.clk.Now(), "org-a")
	if err := runWatchAll(t, f.o, context.Background(), f.params(2), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	stderr := f.stderr.String()
	if n := strings.Count(stderr, "watchdog: director check mark not written: org: write "+DirectorAlivePathIn(f.dir)+": "); n != 2 {
		t.Fatalf("stderr = %q; want the mark line once per cycle", stderr)
	}
	if n := strings.Count(stderr, "watchdog: director d1 is not woken: the watch's check found it ended (agent_ended) (said once)"); n != 1 {
		t.Fatalf("stderr = %q; want the ended line once", stderr)
	}
	if slices.Contains(h.log(), "pane_send_text") {
		t.Fatalf("herdr calls = %q; want nothing typed", h.log())
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), e1); item.Woken {
		t.Fatalf("item %s = %+v; want it not woken", e1, item)
	}
	if s := readWatchStatusFile(t, f.dir, "org-a"); s.Cycles != 2 {
		t.Fatalf("org-a status = %+v; want 2 cycles", s)
	}
}

// The plan's edge case: the start of a starting registration lands its
// launched while the watch checks it, after the read, and then ends. The
// mark is not start_ended: the check reads the registration again, finds it
// running, and checks its agent.
func TestRunWatchAll_ALaunchedThatLandsDuringTheCheckIsNotMarkedStartEnded(t *testing.T) {
	f := newWatchAllFixture(t)
	h := &cycleHerdr{fakeHerdr: &fakeHerdr{}, agentGetErr: map[string]error{}}
	f.o.Herdr = h
	s := f.directorStore()
	writeDirectorEvents(t, s, checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkPane("d1", "pane-1"))
	starterAlive := true
	f.o.DirectorProcessReader = func(_ context.Context, pid int) (DirectorProcess, error) {
		if pid != checkStarterPID {
			t.Errorf("ps read pid %d", pid)
		}
		if starterAlive {
			return DirectorProcess{State: DirectorProcessAlive, StartTime: checkStarterStart}, nil
		}
		return DirectorProcess{State: DirectorProcessGone}, nil
	}
	var tries []int
	f.o.directorCheckAfterRead = func(attempt int) {
		tries = append(tries, attempt)
		if attempt == 1 {
			if _, err := s.MarkLaunched("d1", testDirectorWho); err != nil {
				t.Errorf("MarkLaunched: %v", err)
			}
			starterAlive = false
		}
	}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	got := f.readDirectorAliveMark(t)
	if got.State != DirectorCheckAlive || got.How != DirectorCheckHowAgentFound || got.ID != "d1" || !slices.Equal(tries, []int{1, 2}) {
		t.Fatalf("director-alive.json = %+v after tries %v; want alive agent_found from the second try, not start_ended", got, tries)
	}
	if log := h.log(); !slices.Equal(log, []string{"agent_get " + DirectorAgentName(directorTestToken)}) {
		t.Fatalf("herdr calls = %q; want the second try's agent get only", log)
	}
}
