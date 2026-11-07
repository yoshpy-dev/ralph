package org

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Send's last approval check of a TASK, under the pane lock (cross-review
// cycle 2 #1 of org-seat-resume; verbs.go and pane_lock.go). The approval
// fixtures (enforceFixture, planTask, assertTaskRefused) live in
// approval_enforce_test.go, and the pane lock helpers (paneLockPath,
// mustPaneLockFree, waitForResumeTest) in pane_lock_test.go. The test names
// hold Send so the slice's `-run 'Send'` runs them.

// underLockEditHerdr is fakeHerdr whose second AgentWait, Send's idle/done
// wait under the pane lock, runs onWait before it returns, and records whether
// the pane lock at path was held then.
type underLockEditHerdr struct {
	*fakeHerdr
	path   string
	onWait func()
	waits  int
	held   bool
}

func (h *underLockEditHerdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	out, err := h.fakeHerdr.AgentWait(ctx, target, until, timeoutMS)
	if h.waits++; h.waits == 2 {
		held, lockErr := fileLockHeld(h.path)
		h.held = lockErr == nil && held
		h.onWait()
	}
	return out, err
}

// TestOrgSend_ApprovalCheck_AgainUnderThePaneLock (cross-review cycle 2 #1 of
// org-seat-resume): holding the pane lock, after its idle/done wait under it
// and right before it types, Send checks a TASK's plan approval a last time,
// on a fresh read of the manifest. A plan that changes during that wait (in
// its digest, or only where the digest does not show), or a revision a human
// approves during it, makes Send refuse the TASK with ErrPlanNotApproved,
// typed and raw alike, with nothing typed, nothing recorded, and the lock
// released; the refusal after an approved revision names its digest. The
// controls: a wait during which the plan does not change, or only a checkbox
// is ticked, sends the TASK.
func TestOrgSend_ApprovalCheck_AgainUnderThePaneLock(t *testing.T) {
	revised := replaceOnce("Add the login form.", "Add the signup form.")(testPlanDraft)
	revisedRef := PlanRef{Path: testPlanRel, Digest: PlanDigest([]byte(revised))}
	cases := []struct {
		name    string
		during  func(t *testing.T, fx *enforceFixture) // what happens during the wait under the lock
		refused string                                 // what the refusal contains; "": sent
	}{
		{"the plan changed", func(t *testing.T, fx *enforceFixture) {
			fx.mark(t, replaceOnce("Add the login form.", "Add the signup form."))
		}, "changed after its approval: approved digest "},
		{"a Progress checklist line added", func(t *testing.T, fx *enforceFixture) {
			fx.mark(t, replaceOnce("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n"))
		}, "changed after its approval outside its - Status: and - Approved: lines and checkbox marks"},
		{"a revision approved", func(t *testing.T, fx *enforceFixture) {
			if err := fx.o.Manifest.Append(approvedEvent("org-a", "e2", revisedRef, PlanContentHash([]byte(revised)))); err != nil {
				t.Fatal(err)
			}
			writePlan(t, fx.worktree, testPlanRel, approvedPlanText(revised, revisedRef.Digest))
		}, "the plan's current approved digest is sha256:" + revisedRef.Digest},
		{"nothing changed", func(*testing.T, *enforceFixture) {}, ""},
		{"a checkbox ticked", func(t *testing.T, fx *enforceFixture) { fx.mark(t, replaceOnce("- [ ] AC1", "- [x] AC1")) }, ""},
	}
	for _, tc := range cases {
		for _, raw := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s raw=%v", tc.name, raw), func(t *testing.T) {
				fx := newEnforceFixture(t, config.ApprovalModePerPlan)
				seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
				fx.record(t)
				fx.mark(t, nil)
				var manifestAfterWait string
				h := &underLockEditHerdr{fakeHerdr: fx.h, path: paneLockPath(fx.o, "pane-2"), onWait: func() {
					tc.during(t, fx)
					manifestAfterWait = fileOrAbsent(t, fx.o.Manifest.Path())
				}}
				fx.o.Herdr = h
				calls, receipts := len(fx.h.calls), fileOrAbsent(t, fx.o.Receipts.Path())

				r := fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: planTask(fx.ref), Raw: raw, TimeoutMS: 10000})
				if h.waits < 2 || !h.held {
					t.Fatalf("send = %+v after %d idle/done waits (the lock held at the second: %v); want the wait under the lock first", r, h.waits, h.held)
				}
				mustPaneLockFree(t, fx.o, "pane-2")
				if tc.refused == "" {
					if r.Err != nil || r.Progress != SendProgressEnterPressed {
						t.Fatalf("send = %+v; want it sent", r)
					}
					if last := lastEvent(t, fx.o); last.Event != EventSent || last.DryRun || last.SeatID != implementerSeatID {
						t.Fatalf("last event = %+v; want sent to the implementer", last)
					}
					return
				}
				assertTaskRefused(t, r, tc.refused)
				if got := fx.h.calls[calls:]; !slices.Equal(got, []string{"agent_wait", "agent_wait"}) {
					t.Fatalf("herdr calls = %q; want the two idle/done waits and nothing typed", got)
				}
				if fileOrAbsent(t, fx.o.Manifest.Path()) != manifestAfterWait || fileOrAbsent(t, fx.o.Receipts.Path()) != receipts {
					t.Fatal("a TASK refused under the pane lock changed the manifest or the receipts")
				}
			})
		}
	}
}

// paneLockOpens counts this process's open files on the pane lock file of
// paneID: the descriptors /dev/fd lists whose fstat has the lock file's
// device and inode (a stat of /dev/fd/<n> itself does not, on macOS). A
// holder's is one, and a Send that waits for the lock has opened another. The
// test is skipped where /dev/fd cannot be read.
func paneLockOpens(t *testing.T, o *Org, paneID string) int {
	t.Helper()
	fi, err := os.Stat(paneLockPath(o, paneID))
	if err != nil {
		t.Fatalf("stat the pane lock of %s: %v", paneID, err)
	}
	lockFile, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skipf("no device and inode of the pane lock file to compare open files with (%T)", fi.Sys())
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

// TestOrgSend_ApprovalCheck_PlanChangedWhileSendWaitsForThePaneLock
// (cross-review cycle 2 #1 of org-seat-resume): a Send of a TASK whose
// approval checks before the pane lock passed waits for the lock another send
// or resume holds. The plan changes while it waits; once the lock is free,
// Send waits for the seat under it and then refuses the TASK with
// ErrPlanNotApproved, typed and raw alike, with nothing typed, nothing
// recorded, and the lock released. The control: with no change while it
// waits, the TASK is sent once the lock is free.
func TestOrgSend_ApprovalCheck_PlanChangedWhileSendWaitsForThePaneLock(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edit    func(string) string // the edit of the plan while Send waits for the lock; nil: none
		refused string              // what the refusal contains; "": sent
	}{
		{"the plan changed", replaceOnce("Add the login form.", "Add the signup form."), "changed after its approval: approved digest "},
		{"nothing changed", nil, ""},
	} {
		for _, raw := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s raw=%v", tc.name, raw), func(t *testing.T) {
				fx := newEnforceFixture(t, config.ApprovalModePerPlan)
				seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
				fx.record(t)
				fx.mark(t, nil)
				holder, err := fx.o.acquirePaneLock("pane-2", 0)
				if err != nil {
					t.Fatalf("hold the pane lock of pane-2: %v", err)
				}
				release := sync.OnceFunc(func() { _ = holder.release() })
				t.Cleanup(release)
				if n := paneLockOpens(t, fx.o, "pane-2"); n != 1 {
					t.Fatalf("open files on the pane lock = %d with the holder's only; want 1", n)
				}
				calls := len(fx.h.calls)
				manifest, receipts := fileOrAbsent(t, fx.o.Manifest.Path()), fileOrAbsent(t, fx.o.Receipts.Path())

				sent := make(chan SendResult, 1)
				go func() {
					sent <- fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: planTask(fx.ref), Raw: raw, TimeoutMS: 10000})
				}()
				waitForResumeTest(t, "the send's wait for the pane lock", func() bool { return paneLockOpens(t, fx.o, "pane-2") >= 2 })
				if tc.edit != nil {
					fx.mark(t, tc.edit)
				}
				release()
				r := <-sent
				mustPaneLockFree(t, fx.o, "pane-2")
				if tc.refused == "" {
					if r.Err != nil || r.Progress != SendProgressEnterPressed {
						t.Fatalf("send = %+v; want it sent once the lock was free", r)
					}
					return
				}
				assertTaskRefused(t, r, tc.refused)
				if got := fx.h.calls[calls:]; !slices.Equal(got, []string{"agent_wait", "agent_wait"}) {
					t.Fatalf("herdr calls = %q; want the two idle/done waits and nothing typed", got)
				}
				if fileOrAbsent(t, fx.o.Manifest.Path()) != manifest || fileOrAbsent(t, fx.o.Receipts.Path()) != receipts {
					t.Fatal("a TASK refused under the pane lock changed the manifest or the receipts")
				}
			})
		}
	}
}
