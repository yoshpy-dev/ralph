package org

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"
)

// The wait of `ralph org merge --wait-checks <d>`, S7 of plan
// docs/plans/active/2026-10-17-org-auto-merge.md (D13, AC13). A leader opens
// the PR right after its gate and disbands, and regate's push makes CI run
// again, so a branch merge-order lists as ready often has checks that still
// run, and a director that waits on the inbox has no turn that comes back to
// run merge again. So the verb waits: when checkMerge (merge.go) refuses only
// because the PR's checks are pending or none is reported yet
// (mergeChecksKind.waits, decided on the kind and never on the refusal's
// text), Merge (merge_run.go) releases regate.lock, so that a ralph org regate
// or another merge can run meanwhile, and reads only gh pr checks of the PR,
// every mergeWaitInterval, with readChecks, the read of step 10 of
// checkMerge. Once the checks are neither pending nor none, gh pr checks
// fails, the time is up, or ctx is done, the wait ends; Merge then takes
// regate.lock again as it took it first and runs every check of checkMerge
// from the start, so a head or a base that moved meanwhile is refused as it
// is without a wait. A refusal of any other kind is never waited out.

const (
	// maxMergeWaitChecks is the longest --wait-checks Merge takes
	// (MergeWaitChecksRefusal).
	maxMergeWaitChecks = 6 * time.Hour
	// mergeWaitInterval is how long a wait sleeps before each read of gh pr
	// checks.
	mergeWaitInterval = 30 * time.Second
)

// MergeWaitChecksRefusal is the refusal of a --wait-checks of d below 0 or
// above maxMergeWaitChecks, and "" for one from 0 to maxMergeWaitChecks.
// Merge refuses such a value before it takes a lock or runs a command, and
// ralph org merge before it reads the ledger.
func MergeWaitChecksRefusal(d time.Duration) string {
	if d >= 0 && d <= maxMergeWaitChecks {
		return ""
	}
	return fmt.Sprintf(mergePrefix+"--wait-checks %s is out of range, so nothing was merged: give a duration from 0 (no wait, "+
		"the default) to %s, such as 60m", d, maxMergeWaitChecks)
}

// mergeWaitTest replaces, in a test, what a wait takes from the real clock
// (newMergeWait): interval, when above 0, replaces mergeWaitInterval, and
// now and sleep, when set, replace time.Now and sleepContext.
type mergeWaitTest struct {
	interval time.Duration
	now      func() time.Time
	sleep    func(ctx context.Context, d time.Duration) error
}

// mergeWait is the waiting of one Merge: how long --wait-checks lets it wait
// in all (budget), how often it reads the checks, where its lines go, and,
// once its first wait started, when that was and when the budget is used up.
type mergeWait struct {
	budget, interval  time.Duration
	progress          io.Writer
	now               func() time.Time
	sleep             func(ctx context.Context, d time.Duration) error
	started, deadline time.Time
}

// mergeWaitFor is what a wait reads the checks of: what checkMerge read
// before it refused (r), and the kind of its refusal.
type mergeWaitFor struct {
	r    mergeReady
	kind mergeChecksKind
}

func newMergeWait(p MergeParams) *mergeWait {
	w := &mergeWait{budget: p.WaitChecks, interval: mergeWaitInterval, progress: p.Progress, now: time.Now, sleep: sleepContext}
	if p.waitTest.interval > 0 {
		w.interval = p.waitTest.interval
	}
	if p.waitTest.now != nil {
		w.now = p.waitTest.now
	}
	if p.waitTest.sleep != nil {
		w.sleep = p.waitTest.sleep
	}
	if w.progress == nil {
		w.progress = io.Discard
	}
	return w
}

// left reports whether w may still wait: its budget is above 0 and, counted
// from the start of its first wait, not used up.
func (w *mergeWait) left() bool {
	return w.budget > 0 && (w.started.IsZero() || w.now().Before(w.deadline))
}

// waited is what Merge adds to a refusal of checks still pending or none
// once w's budget is used up: how long w waited from the start of its first
// wait. It is "" when w never waited.
func (w *mergeWait) waited() string {
	if w.started.IsZero() {
		return ""
	}
	return fmt.Sprintf("; ralph org merge waited %s for the checks (--wait-checks %s)", mergeWaitDuration(w.now().Sub(w.started)),
		w.budget)
}

// wait is one wait for the checks of f.r's PR, with regate.lock released. It
// writes one line to w.progress when it starts; then, again and again, it
// sleeps w.interval, or what is left of the budget when that is less, and
// reads the checks with m.readChecks, until they are neither pending nor none,
// gh pr checks fails, the budget is used up, or ctx is done. Every read comes
// after a sleep. When it ends, it writes one line that says why and how long
// it waited. The error is ctx done: nothing was merged, and Merge returns it.
func (w *mergeWait) wait(ctx context.Context, m *mergeCheck, f mergeWaitFor) error {
	start := w.now()
	if w.started.IsZero() {
		w.started, w.deadline = start, start.Add(w.budget)
	}
	cmd := "gh pr checks " + strconv.Itoa(f.r.prNumber) + " -R " + f.r.repoArg
	why := fmt.Sprintf("PR #%d has checks pending", f.r.prNumber)
	if f.kind == mergeChecksZero {
		why = fmt.Sprintf("PR #%d has no checks reported yet", f.r.prNumber)
	}
	w.line(fmt.Sprintf("%s, so ralph org merge waits for them for up to %s (--wait-checks %s), with regate.lock released, "+
		"reading %s every %s", why, mergeWaitDuration(max(w.deadline.Sub(start), 0)), w.budget, cmd, w.interval))
	ended, interrupted := w.poll(ctx, m, f.r, cmd)
	end := fmt.Sprintf("stopped waiting for the checks of PR #%d after %s: %s", f.r.prNumber, mergeWaitDuration(w.now().Sub(start)),
		ended)
	if interrupted {
		w.line(end + "; nothing was merged")
		return mergeInterrupted(ctx, fmt.Sprintf("waiting for the checks of PR #%d, before gh pr merge ran", f.r.prNumber))
	}
	w.line(end + "; running every check again")
	return nil
}

// poll is the loop of wait: it returns why the wait ended, and interrupted
// when that is ctx done.
func (w *mergeWait) poll(ctx context.Context, m *mergeCheck, r mergeReady, cmd string) (ended string, interrupted bool) {
	for {
		left := w.deadline.Sub(w.now())
		if left <= 0 {
			return "--wait-checks " + w.budget.String() + " is used up", false
		}
		if err := w.sleep(ctx, min(w.interval, left)); err != nil || ctx.Err() != nil {
			return fmt.Sprintf("interrupted (%v)", context.Cause(ctx)), true
		}
		refused, kind, err := m.readChecks(ctx, r)
		switch {
		case err != nil:
			// readChecks errs only when ctx is done.
			return fmt.Sprintf("interrupted (%v)", context.Cause(ctx)), true
		case kind.waits():
			// Still pending, or none reported yet: the wait goes on.
		case refused == "":
			return "the checks passed", false
		case kind == mergeChecksUnread:
			return cmd + " failed", false
		default:
			return "a check did not pass", false
		}
	}
}

// line writes text after mergePrefix, escaped (PrintableInboxText), as one
// line of w.progress.
func (w *mergeWait) line(text string) {
	_, _ = fmt.Fprintln(w.progress, PrintableInboxText(mergePrefix+text, false))
}

// mergeWaitDuration is d for a text: to the second from a second on, and to
// the millisecond below it.
func mergeWaitDuration(d time.Duration) time.Duration {
	if d >= time.Second {
		return d.Round(time.Second)
	}
	return d.Round(time.Millisecond)
}

// sleepContext sleeps d, or until ctx is done, and then returns ctx's error.
func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
