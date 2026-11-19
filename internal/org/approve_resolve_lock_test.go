package org

// The lock-order tests of ResolveInboxItem (plan
// docs/plans/active/2026-11-19-approve-resolve-decision.md, AC-10), with the
// helpers only they use. The fixture and the helpers the other
// ResolveInboxItem tests share with them (newResolveFixture, ask,
// appendEvents, item, assertRefused) stay in approve_resolve_test.go.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// lockFileOpens counts the open files of this process on the lock file at
// path, by the device and inode of /dev/fd/<n> (a stat of /dev/fd/<n> itself
// does not match, on macOS): one for a holder, and one more for each caller
// waiting for the lock. The test is skipped where /dev/fd cannot be read.
func lockFileOpens(t *testing.T, path string) int {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat the lock %s: %v", path, err)
	}
	lockFile, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skipf("no device and inode of the lock file to compare open files with (%T)", fi.Sys())
	}
	fds, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skipf("cannot count open files without /dev/fd: %v", err)
	}
	n := 0
	for _, fd := range fds {
		num, err := strconv.Atoi(fd.Name())
		var st syscall.Stat_t
		if err == nil && syscall.Fstat(num, &st) == nil && st.Dev == lockFile.Dev && st.Ino == lockFile.Ino {
			n++
		}
	}
	return n
}

// assertLockOpenOnce checks that the lock file at path is open once, by its
// holder, before the test starts a caller that waits for it, so a probe that
// counts nothing fails here instead of passing waitForLockWaiter by chance.
func assertLockOpenOnce(t *testing.T, path string) {
	t.Helper()
	if n := lockFileOpens(t, path); n != 1 {
		t.Fatalf("the lock %s is open %d times with only its holder; want 1", path, n)
	}
}

// waitForLockWaiter waits until a second caller has opened the lock file at
// path, which one holder already has open (assertLockOpenOnce), and fails the
// test after 3 s (the waiter gives up after manifestLockTimeout, 5 s).
func waitForLockWaiter(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for lockFileOpens(t, path) < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("no second caller waits for the lock %s", path)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// resolveTestWait bounds each wait of the lock tests below for a goroutine's
// result, so a change that never sends it fails the test instead of hanging
// it until go test's own timeout.
const resolveTestWait = 30 * time.Second

// receiveWithin returns the value received from ch, and fails the test when
// none comes within resolveTestWait; what names the value in the failure.
func receiveWithin[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(resolveTestWait):
		t.Fatalf("no %s within %s", what, resolveTestWait)
		var zero T
		return zero
	}
}

// AC-10, resolve first: a resolve of an item with no approval yet holds the
// manifest lock from its check to its write, so an approve of the item that
// starts meanwhile records nothing until the resolve is done, then records
// the approval and says the item was resolved by someone else.
func TestResolveInboxItem_ApproveWaitsForAResolveThatHoldsTheLock(t *testing.T) {
	rf := newResolveFixture(t)
	paused, release := make(chan struct{}), make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	resolveAfterCheckHook = func() {
		close(paused)
		<-release
	}
	t.Cleanup(func() { resolveAfterCheckHook = nil })
	// A test that fails while the resolve is paused lets it go on, so it
	// does not keep the manifest lock after the test.
	t.Cleanup(releaseOnce)

	resolveDone := make(chan error, 1)
	go func() {
		_, err := rf.o.ResolveInboxItem(rf.id, "changes asked for: split AC1", false)
		resolveDone <- err
	}()
	select {
	case <-paused:
	case err := <-resolveDone:
		t.Fatalf("resolve returned %v before it reached its write; want it paused there, holding the manifest lock", err)
	case <-time.After(resolveTestWait):
		t.Fatalf("the resolve did not reach its write within %s", resolveTestWait)
	}
	lockPath := filepath.Join(filepath.Dir(rf.o.Manifest.Path()), manifestLockFile)
	assertLockOpenOnce(t, lockPath)
	approveDone := make(chan ApproveResult, 1)
	go func() { approveDone <- rf.o.Approve(ApproveParams{IDs: []string{rf.id}, Cwd: "/w"}) }()
	waitForLockWaiter(t, lockPath)
	approvalsWhilePaused := len(planApprovedEvents(t, rf.o))
	releaseOnce()

	if err := receiveWithin(t, resolveDone, "result of the resolve"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	res := receiveWithin(t, approveDone, "result of the approve")
	if approvalsWhilePaused != 0 {
		t.Errorf("approve recorded %d approvals while the resolve held the lock; want none", approvalsWhilePaused)
	}
	if len(res.Approved) != 1 || !res.Approved[0].ResolvedElsewhere || res.Err == nil ||
		!strings.Contains(res.Err.Error(), "the inbox item was resolved by someone else after approve read the inbox") {
		t.Fatalf("approve = %+v; want the approval recorded and the item reported resolved by someone else", res)
	}
	if got := rf.item(t, rf.id); got.State != InboxStateResolved || got.Note != "changes asked for: split AC1" {
		t.Errorf("item = %+v; want it resolved by the resolve", got)
	}
}

// AC-10, approve first: a resolve that starts while the manifest lock is held
// by an approve recording the approval waits for it, then reads the approval
// and refuses the item.
func TestResolveInboxItem_ResolveWaitsForAnApproveThatHoldsTheLock(t *testing.T) {
	rf := newResolveFixture(t)
	before := fileOrAbsent(t, rf.o.Inbox.Path())
	resolveDone := make(chan error, 1)
	lockPath := filepath.Join(filepath.Dir(rf.o.Manifest.Path()), manifestLockFile)
	if err := withManifestLock(filepath.Dir(rf.o.Manifest.Path()), func() error {
		assertLockOpenOnce(t, lockPath)
		go func() {
			_, err := rf.o.ResolveInboxItem(rf.id, "approved", false)
			resolveDone <- err
		}()
		waitForLockWaiter(t, lockPath)
		select {
		case err := <-resolveDone:
			t.Fatalf("resolve returned %v while the lock was held; want it to wait", err)
		default:
		}
		// What Approve appends under the lock.
		rf.appendEvents(t, approvedEvent("org-a", rf.id, rf.ref, PlanContentHash([]byte(testPlanDraft))))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rf.assertRefused(t, receiveWithin(t, resolveDone, "result of the resolve"), rf.id, rf.ref, before, InboxStateOpen)
}

// Self-review L1: an item that is not a plan QUESTION (a QUESTION with no
// PLAN: header, and a BLOCKED and a RESULT item with one) is resolved
// without the manifest lock, so its resolve neither waits for a holder of
// that lock nor fails when the lock is held past manifestLockTimeout. The
// control: the resolve of the fixture's plan QUESTION, started under the
// same hold, does wait for the lock (waitForLockWaiter) and resolves once it
// is released.
func TestResolveInboxItem_ItemsThatAreNotPlanQuestionsDoNotWaitForTheManifestLock(t *testing.T) {
	rf := newResolveFixture(t)
	ids := []string{
		rf.ask(t, "QUESTION", "TYPE: QUESTION\n\nwhich base branch?"),
		rf.ask(t, "BLOCKED", "TYPE: BLOCKED\nTASK_ID: org-a\nPLAN: "+rf.ref.String()+"\n\nstuck"),
		rf.ask(t, "RESULT", "TYPE: RESULT\nTASK_ID: org-a\nPLAN: "+rf.ref.String()+"\n\ndone"),
	}
	dir := filepath.Dir(rf.o.Manifest.Path())
	lockPath := filepath.Join(dir, manifestLockFile)
	planDone := make(chan error, 1)
	if err := withManifestLock(dir, func() error {
		assertLockOpenOnce(t, lockPath)
		for _, id := range ids {
			start := time.Now()
			res, err := rf.o.ResolveInboxItem(id, "docs/reports/x.md", false)
			if err != nil || res != (InboxResolveResult{}) {
				t.Errorf("resolve %s while the manifest lock is held = %+v, %v; want it resolved with an empty result", id, res, err)
			}
			if took := time.Since(start); took >= manifestLockTimeout {
				t.Errorf("resolve %s took %s while the manifest lock was held; want no wait for the lock", id, took)
			}
			if got := rf.item(t, id); got.State != InboxStateResolved || got.Note != "docs/reports/x.md" {
				t.Errorf("item = %+v; want it resolved", got)
			}
		}
		go func() {
			_, err := rf.o.ResolveInboxItem(rf.id, "docs/reports/x.md", false)
			planDone <- err
		}()
		waitForLockWaiter(t, lockPath)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := receiveWithin(t, planDone, "result of the plan QUESTION's resolve"); err != nil {
		t.Fatalf("resolve of the plan QUESTION after the lock was released: %v", err)
	}
	if got := rf.item(t, rf.id); got.State != InboxStateResolved {
		t.Errorf("plan QUESTION = %+v; want it resolved", got)
	}
}

// AC-10: the resolve of an item with no approval yet writes the inbox while it
// still holds the manifest lock, not after it. The test holds the inbox lock,
// so the resolve stops at its write (InboxStore.Resolve waits for that lock),
// and checks from there that the manifest lock is still held by the resolve
// (open once) and that an approve of the item, started now, waits for it and
// records nothing. Once the inbox lock is released, the resolve writes, and the
// approve records its approval and says the item was resolved by someone else.
// With the write after the manifest lock, the lock is free at that point and
// the approve could record the approval before the item is resolved.
func TestResolveInboxItem_WritesTheInboxBeforeItReleasesTheManifestLock(t *testing.T) {
	rf := newResolveFixture(t)
	dir := filepath.Dir(rf.o.Manifest.Path())
	manifestLock, inboxLock := filepath.Join(dir, manifestLockFile), filepath.Join(dir, inboxLockFile)
	resolveDone := make(chan error, 1)
	approveDone := make(chan ApproveResult, 1)
	approvalsWhileTheWriteWaits := -1
	if err := withFileLock(dir, inboxLockFile, "inbox", func() error {
		assertLockOpenOnce(t, inboxLock)
		go func() {
			_, err := rf.o.ResolveInboxItem(rf.id, "changes asked for: split AC1", false)
			resolveDone <- err
		}()
		// The resolve reads the inbox without its lock, so the only code of the
		// resolve that opens the inbox lock file is InboxStore.Resolve: the
		// second open is the resolve at its write.
		waitForLockWaiter(t, inboxLock)
		if n := lockFileOpens(t, manifestLock); n != 1 {
			t.Fatalf("the manifest lock is open %d times while the resolve waits to write the inbox; want 1 (the resolve holds it until the write is done)", n)
		}
		go func() { approveDone <- rf.o.Approve(ApproveParams{IDs: []string{rf.id}, Cwd: "/w"}) }()
		waitForLockWaiter(t, manifestLock)
		approvalsWhileTheWriteWaits = len(planApprovedEvents(t, rf.o))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := receiveWithin(t, resolveDone, "result of the resolve"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	res := receiveWithin(t, approveDone, "result of the approve")
	if approvalsWhileTheWriteWaits != 0 {
		t.Errorf("approve recorded %d approvals while the resolve waited to write the inbox; want none", approvalsWhileTheWriteWaits)
	}
	if len(res.Approved) != 1 || !res.Approved[0].ResolvedElsewhere || res.Err == nil ||
		!strings.Contains(res.Err.Error(), "the inbox item was resolved by someone else after approve read the inbox") {
		t.Fatalf("approve = %+v; want the approval recorded and the item reported resolved by someone else", res)
	}
	if got := rf.item(t, rf.id); got.State != InboxStateResolved || got.Note != "changes asked for: split AC1" {
		t.Errorf("item = %+v; want it resolved by the resolve", got)
	}
}

// An ID that the first read of the inbox does not find is resolved under the
// manifest lock too, since the item can be raised and approved after that read.
// The test holds the lock and starts the resolve of e2, which the inbox does not
// hold yet; once the resolve waits for the lock, the test raises e2 as a plan
// QUESTION and records its approval. The resolve then reads both under the
// lock and refuses e2, with nothing written. With no lock for such an ID, the
// resolve ends at once with the unknown-ID error.
func TestResolveInboxItem_AnIDMissingAtTheFirstReadIsCheckedUnderTheManifestLock(t *testing.T) {
	rf := newResolveFixture(t)
	const laterID = "e2" // the ID the inbox gives the item raised after the fixture's e1
	dir := filepath.Dir(rf.o.Manifest.Path())
	lockPath := filepath.Join(dir, manifestLockFile)
	resolveDone := make(chan error, 1)
	var before string
	if err := withManifestLock(dir, func() error {
		assertLockOpenOnce(t, lockPath)
		go func() {
			_, err := rf.o.ResolveInboxItem(laterID, approveNote(rf.ref), false)
			resolveDone <- err
		}()
		waitForLockWaiter(t, lockPath)
		if id := rf.ask(t, "QUESTION", planQuestion(rf.ref)); id != laterID {
			t.Fatalf("the plan QUESTION raised got ID %s; the test needs %s", id, laterID)
		}
		// What Approve appends under the lock.
		rf.appendEvents(t, approvedEvent("org-a", laterID, rf.ref, PlanContentHash([]byte(testPlanDraft))))
		before = fileOrAbsent(t, rf.o.Inbox.Path())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rf.assertRefused(t, receiveWithin(t, resolveDone, "result of the resolve"), laterID, rf.ref, before, InboxStateOpen)
}
