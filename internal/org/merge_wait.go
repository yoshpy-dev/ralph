package org

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The wait of `ralph org merge --wait-checks <d>`, S7 of plan
// docs/plans/active/2026-10-17-org-auto-merge.md (D13, AC13), and its wait
// for an org to be disbanded, S8c (D5, AC11). A leader opens the PR right
// after its gate and disbands, and regate's push makes CI run again, so a
// branch merge-order lists as ready often has checks that still run, and a
// director that waits on the inbox has no turn that comes back to run merge
// again. So the verb waits: when checkMerge (merge.go) refuses only because
// the PR's checks are pending or none is reported yet (mergeRefusalKind.waits,
// decided on the kind and never on the refusal's text), Merge (merge_run.go)
// releases regate.lock, so that a ralph org regate or another merge can run
// meanwhile, and reads only gh pr checks of the PR, every mergeWaitInterval,
// with readChecks, the read of step 10 of checkMerge. A director that merges
// once the leader's RESULT reached the inbox finds the org still running, too,
// since the leader disbands it after its RESULT: then Merge waits the same
// way, but reads only the ledger, as step 8 reads it, with no git and no gh
// (orgWaitEnd). The waits of one Merge share its budget, its interval, its
// lines, and its handling of ctx. Once what the refusal waits for is gone, the
// read fails, the time is up, or ctx is done, the wait ends; Merge then takes
// regate.lock again as it took it first and runs every check of checkMerge
// from the start, so a head or a base that moved meanwhile is refused as it
// is without a wait. A refusal of any other kind, an org that runs another
// feature now included, is never waited out.

const (
	// maxMergeWaitChecks is the longest --wait-checks Merge takes
	// (MergeWaitChecksRefusal).
	maxMergeWaitChecks = 6 * time.Hour
	// mergeWaitInterval is how long a wait sleeps before each read of gh pr
	// checks or of the ledger.
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
// in all (budget), how often it reads the checks or the ledger, where its
// lines go, and, once its first wait started, when that was, when the budget
// is used up, and what its waits waited for (waitedFor: "the checks", or
// org_id "<org>" to be disbanded), each once, in the order they began.
type mergeWait struct {
	budget, interval  time.Duration
	progress          io.Writer
	now               func() time.Time
	sleep             func(ctx context.Context, d time.Duration) error
	started, deadline time.Time
	waitedFor         []string
}

// mergeWaitFor is what a wait waits for: what checkMerge read before it
// refused (r), and the kind of its refusal, which waits.
type mergeWaitFor struct {
	r    mergeReady
	kind mergeRefusalKind
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

// waited is what Merge adds to a refusal whose kind waits once w's budget is
// used up: how long w waited from the start of its first wait, and for what
// (waitedFor). It is "" when w never waited.
func (w *mergeWait) waited() string {
	if w.started.IsZero() {
		return ""
	}
	return fmt.Sprintf("; ralph org merge waited %s for %s (--wait-checks %s)", mergeWaitDuration(w.now().Sub(w.started)),
		strings.Join(w.waitedFor, " and for "), w.budget)
}

// waitedOut is Merge's refusal once w's budget is used up and checkMerge
// still refuses with kind, which waits: refused, with what waited adds, and,
// for r's org that still runs, where to see what it does, since whether to
// disband it is a human's call. With no wait before it, it is refused as it
// is.
func (w *mergeWait) waitedOut(refused string, r mergeReady, kind mergeRefusalKind) string {
	waited := w.waited()
	if waited == "" || kind != mergeRefusalOrgRunning {
		return refused + waited
	}
	return PrintableInboxText(fmt.Sprintf("%s%s; the org still runs: see what it does with ralph org status --org-id %s, and a "+
		"human decides whether to disband it (ralph org disband --org-id %s)", refused, waited, r.orgID, r.orgID), false)
}

// wait is one wait, with regate.lock released, for what f's refusal waits
// for: f.r's org to be disbanded, read in the ledger at manifest
// (orgWaitEnd), or the checks of f.r's PR (checksWaitEnd). It writes one line
// to w.progress when it starts; then, again and again, it sleeps w.interval,
// or what is left of the budget when that is less, and reads, until what the
// refusal waits for is gone, the read fails, the budget is used up, or ctx is
// done. Every read comes after a sleep. When it ends, it writes one line that
// says why and how long it waited. The error is ctx done: nothing was merged,
// and Merge returns it.
func (w *mergeWait) wait(ctx context.Context, m *mergeCheck, manifest string, f mergeWaitFor) error {
	start := w.now()
	if w.started.IsZero() {
		w.started, w.deadline = start, start.Add(w.budget)
	}
	left := mergeWaitDuration(max(w.deadline.Sub(start), 0))
	what, forWhat := fmt.Sprintf("the checks of PR #%d", f.r.prNumber), "the checks"
	var read func() (ended string, interrupted bool)
	if f.kind == mergeRefusalOrgRunning {
		what = fmt.Sprintf("org_id %q to be disbanded", f.r.orgID)
		forWhat = what
		w.line(fmt.Sprintf("org_id %q of the branch is still running (its leader disbands it after its RESULT), so ralph org "+
			"merge waits for it to be disbanded for up to %s (--wait-checks %s), with regate.lock released, reading the ledger "+
			"every %s", f.r.orgID, left, w.budget, w.interval))
		read = func() (string, bool) { return orgWaitEnd(manifest, f.r), false }
	} else {
		cmd := "gh pr checks " + strconv.Itoa(f.r.prNumber) + " -R " + f.r.repoArg
		why := fmt.Sprintf("PR #%d has checks pending", f.r.prNumber)
		if f.kind == mergeRefusalChecksZero {
			why = fmt.Sprintf("PR #%d has no checks reported yet", f.r.prNumber)
		}
		w.line(fmt.Sprintf("%s, so ralph org merge waits for them for up to %s (--wait-checks %s), with regate.lock released, "+
			"reading %s every %s", why, left, w.budget, cmd, w.interval))
		read = func() (string, bool) { return m.checksWaitEnd(ctx, f.r, cmd) }
	}
	if !slices.Contains(w.waitedFor, forWhat) {
		w.waitedFor = append(w.waitedFor, forWhat)
	}
	ended, interrupted := w.poll(ctx, read)
	end := fmt.Sprintf("stopped waiting for %s after %s: %s", what, mergeWaitDuration(w.now().Sub(start)), ended)
	if interrupted {
		w.line(end + "; nothing was merged")
		return mergeInterrupted(ctx, "waiting for "+what+", before gh pr merge ran")
	}
	w.line(end + "; running every check again")
	return nil
}

// poll is the loop of wait: it sleeps, then reads with read, which says why
// the wait ends ("" while it goes on) or that ctx is done, and returns why the
// wait ended, and interrupted when that is ctx done.
func (w *mergeWait) poll(ctx context.Context, read func() (ended string, interrupted bool)) (string, bool) {
	for {
		left := w.deadline.Sub(w.now())
		if left <= 0 {
			return "--wait-checks " + w.budget.String() + " is used up", false
		}
		if err := w.sleep(ctx, min(w.interval, left)); err != nil || ctx.Err() != nil {
			return fmt.Sprintf("interrupted (%v)", context.Cause(ctx)), true
		}
		switch ended, interrupted := read(); {
		case interrupted:
			return fmt.Sprintf("interrupted (%v)", context.Cause(ctx)), true
		case ended != "":
			return ended, false
		}
	}
}

// checksWaitEnd is what a wait for the checks of r's PR reads after each
// sleep: the checks, with m.readChecks. It says why the wait ends, or ""
// while they are still pending or none, and interrupted for ctx done.
func (m *mergeCheck) checksWaitEnd(ctx context.Context, r mergeReady, cmd string) (ended string, interrupted bool) {
	refused, kind, err := m.readChecks(ctx, r)
	switch {
	case err != nil:
		// readChecks errs only when ctx is done.
		return "", true
	case kind == mergeRefusalChecksPending || kind == mergeRefusalChecksZero:
		// Still pending, or none reported yet: the wait goes on.
		return "", false
	case refused == "":
		return "the checks passed", false
	case kind == mergeRefusalChecksUnread:
		return cmd + " failed", false
	default:
		return "a check did not pass", false
	}
}

// orgWaitEnd is what a wait for r's org reads after each sleep: the ledger at
// manifest, without the manifest lock, as checkMerge reads it
// (readMergeLedger), with no git and no gh. It says why the wait ends: the
// ledger cannot be read, r.orgID is not in RunningOrgs any more, or the org
// runs another feature now (mergeTarget.orgRunsOn of r.branch), which step 8
// refuses without a wait; and "" while the org still runs.
func orgWaitEnd(manifest string, r mergeReady) string {
	ledger, err := readMergeLedger(manifest)
	if err != nil {
		return fmt.Sprintf("the ledger could not be read (%v)", err)
	}
	if !slices.Contains(RunningOrgs(ledger.events), r.orgID) {
		return "the org was disbanded"
	}
	targets, _ := collectMergeTargets(ledger)
	if i := slices.IndexFunc(targets, func(t *mergeTarget) bool { return t.branch == r.branch }); i >= 0 &&
		targets[i].orgRunsOn != "" {
		return "the org runs another feature now"
	}
	return ""
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
