package org

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// ResolveInboxItem and the text_sha256 of Send's `sent` event (plan
// docs/plans/active/2026-11-19-approve-resolve-decision.md, AC-1 to AC-5 and
// AC-8 to AC-10 at the org layer). The fixture is enforceFixture
// (approval_enforce_test.go): org-a bound in per-plan mode, its leader
// running on the fake herdr, with an inbox added, so Approve and Send run as
// they do in the CLI. The test names hold Resolve, Inbox, or Send so the
// slice's -run filter runs them.

// resolveFixture is an enforceFixture with an inbox in the manifest's state
// dir and the leader's QUESTION for fx.ref in it.
type resolveFixture struct {
	*enforceFixture
	id string
}

func newResolveFixture(t *testing.T) *resolveFixture {
	t.Helper()
	fx := newEnforceFixture(t, config.ApprovalModePerPlan)
	fx.o.Inbox = NewInboxStore(filepath.Dir(fx.o.Manifest.Path()))
	rf := &resolveFixture{enforceFixture: fx}
	rf.id = rf.ask(t, "QUESTION", planQuestion(fx.ref))
	return rf
}

// ask escalates text as an org-a inbox item and returns its ID.
func (rf *resolveFixture) ask(t *testing.T, typ, text string) string {
	t.Helper()
	id, err := rf.o.Inbox.Escalate(InboxEscalation{OrgID: "org-a", Type: typ, Body: text})
	if err != nil {
		t.Fatalf("escalate: %v", err)
	}
	return id
}

// appendEvents appends evs to the fixture's manifest.
func (rf *resolveFixture) appendEvents(t *testing.T, evs ...ManifestEvent) {
	t.Helper()
	for _, ev := range evs {
		if err := rf.o.Manifest.Append(ev); err != nil {
			t.Fatalf("append %+v: %v", ev, err)
		}
	}
}

// approve runs Approve for the fixture's QUESTION and checks it recorded it.
func (rf *resolveFixture) approve(t *testing.T) {
	t.Helper()
	if res := rf.o.Approve(ApproveParams{IDs: []string{rf.id}, Cwd: "/w"}); res.Err != nil || len(res.Approved) != 1 {
		t.Fatalf("Approve: %+v", res)
	}
}

// send sends text to the org-a seat named by to through Send and checks it
// went through.
func (rf *resolveFixture) send(t *testing.T, to, text string) SendResult {
	t.Helper()
	r := rf.o.Send(SendParams{OrgID: "org-a", To: to, Text: text})
	if r.Err != nil || r.Progress != SendProgressEnterPressed {
		t.Fatalf("send to %s: %+v", to, r)
	}
	return r
}

// item returns inbox item id.
func (rf *resolveFixture) item(t *testing.T, id string) InboxItem {
	t.Helper()
	rr, err := rf.o.Inbox.Read()
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}
	for _, item := range rr.Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no inbox item %s", id)
	return InboxItem{}
}

// assertRefused checks that err is ResolveInboxItem's refusal of id for the
// approval of ref: ErrApprovalDecisionNotSent, the send command of the
// DECISION on its own lines, the resolve again, when --without-decision is
// for, and the resolve approve prints with --without-decision added, on its
// own line at the end, with the inbox as it was (inboxBefore) and the item
// still in state.
func (rf *resolveFixture) assertRefused(t *testing.T, err error, id string, ref PlanRef, inboxBefore, state string) {
	t.Helper()
	if !errors.Is(err, ErrApprovalDecisionNotSent) {
		t.Fatalf("resolve %s = %v; want ErrApprovalDecisionNotSent", id, err)
	}
	msg := err.Error()
	for _, want := range []string{
		"org: inbox: resolve " + id + ": refused, with nothing written: " + ErrApprovalDecisionNotSent.Error() +
			` (org_id "org-a", plan ` + ref.String() + "); tell the leader with:\n",
		"\n" + ApprovalDecisionCommand("org-a", ref, "") + "\nand once that send succeeded, run the resolve again. ",
		"--without-decision resolves the item on purpose with no counted DECISION: after you told the leader another way, " +
			"or when a ralph from before this check sent the DECISION (its sent event has no text_sha256, so it is not counted). ",
		"Keep approve's note on that resolve, since a leader woken while it waits reads the note as the approval:\n",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal =\n%s\nwant it to contain %q", msg, want)
		}
	}
	if want := "\n" + ApprovalResolveCommand(id, ref, "") + " --without-decision"; !strings.HasSuffix(msg, want) {
		t.Errorf("refusal =\n%s\nwant it to end with %q", msg, want)
	}
	if after := fileOrAbsent(t, rf.o.Inbox.Path()); after != inboxBefore {
		t.Errorf("a refused resolve wrote to the inbox:\nbefore: %s\nafter:  %s", inboxBefore, after)
	}
	if got := rf.item(t, id); got.State != state || got.Note != "" {
		t.Errorf("item %s = %+v; want it still %s with no note", id, got, state)
	}
}

// sentEvent is the `sent` event a real Send of text to seat of orgID records
// after herdr's path confirmed the submit.
func sentEvent(orgID, seat, text string) ManifestEvent {
	return ManifestEvent{
		TS: "2026-10-12T12:00:00Z", OrgID: orgID, SeatID: seat, Event: EventSent, PaneID: "pane-1",
		Details: truncateForDetails(text), TextSHA256: sentTextSHA256(text),
	}
}

// AC-1, AC-2, AC-8, and AC-9: an approved item whose DECISION was not sent is
// refused, open and acked alike, with nothing written; once the DECISION
// approve prints is sent to the leader through Send, the item resolves with
// its note, and the result names its org and plan.
func TestResolveInboxItem_RefusedUntilTheApprovalDecisionIsSent(t *testing.T) {
	rf := newResolveFixture(t)
	rf.approve(t)
	note := approveNote(rf.ref)

	before := fileOrAbsent(t, rf.o.Inbox.Path())
	_, err := rf.o.ResolveInboxItem(rf.id, note, false)
	rf.assertRefused(t, err, rf.id, rf.ref, before, InboxStateOpen)

	if _, err := rf.o.Inbox.Ack(rf.id); err != nil {
		t.Fatal(err)
	}
	before = fileOrAbsent(t, rf.o.Inbox.Path())
	_, err = rf.o.ResolveInboxItem(rf.id, note, false)
	rf.assertRefused(t, err, rf.id, rf.ref, before, InboxStateAcked)

	text := ApprovalDecisionText(rf.ref)
	if r := rf.send(t, LeaderIdentity, text); !r.SubmitConfirmed {
		t.Fatalf("send of the DECISION = %+v; want its submit confirmed", r)
	}
	events := mustReadEvents(t, rf.o)
	sum := sha256.Sum256([]byte(text))
	if last := events[len(events)-1]; last.Event != EventSent || last.SeatID != LeaderIdentity || last.TextSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("last event = %+v; want the sent to the leader with text_sha256 %x", last, sum)
	}

	res, err := rf.o.ResolveInboxItem(rf.id, note, false)
	if err != nil {
		t.Fatalf("resolve after the DECISION was sent: %v", err)
	}
	if want := (InboxResolveResult{OrgID: "org-a", Plan: rf.ref}); res != want {
		t.Errorf("result = %+v; want %+v", res, want)
	}
	if got := rf.item(t, rf.id); got.State != InboxStateResolved || got.Note != note {
		t.Errorf("item = %+v; want it resolved with the note %q", got, note)
	}
}

// AC-1: the refusal repeats --state-dir InboxHintStateDir in the send command
// when it is set.
func TestResolveInboxItem_RefusalRepeatsTheStateDir(t *testing.T) {
	rf := newResolveFixture(t)
	rf.approve(t)
	rf.o.InboxHintStateDir = "/ledger dir"
	_, err := rf.o.ResolveInboxItem(rf.id, "approved", false)
	want := "\n" + ApprovalDecisionCommand("org-a", rf.ref, "/ledger dir") + "\n"
	if !errors.Is(err, ErrApprovalDecisionNotSent) || !strings.Contains(err.Error(), want) || !strings.Contains(want, "--state-dir '/ledger dir'") {
		t.Fatalf("refusal = %v; want it to contain %q", err, want)
	}
}

// AC-3 and AC-8: --without-decision resolves an approved item whose DECISION
// was not sent, with the note approve prints for the resolve, and the result
// says no counted DECISION was found.
func TestResolveInboxItem_WithoutDecisionResolvesAndSaysNoDecisionWasFound(t *testing.T) {
	rf := newResolveFixture(t)
	rf.approve(t)
	note := approveNote(rf.ref)
	res, err := rf.o.ResolveInboxItem(rf.id, note, true)
	if err != nil {
		t.Fatalf("resolve --without-decision: %v", err)
	}
	if want := (InboxResolveResult{OrgID: "org-a", Plan: rf.ref, NoDecisionFound: true}); res != want {
		t.Errorf("result = %+v; want %+v", res, want)
	}
	if got := rf.item(t, rf.id); got.State != InboxStateResolved || got.Note != note {
		t.Errorf("item = %+v; want it resolved with the note %q", got, note)
	}
}

// --without-decision changes nothing for an item whose DECISION was sent or
// whose approval is not recorded: the result says nothing about the leader.
func TestResolveInboxItem_WithoutDecisionOnItemsThatNeedNoneSaysNothing(t *testing.T) {
	rf := newResolveFixture(t)
	other := rf.ask(t, "QUESTION", "TYPE: QUESTION\n\nwhich base branch?")
	rf.approve(t)
	rf.send(t, LeaderIdentity, ApprovalDecisionText(rf.ref))
	for _, id := range []string{rf.id, other} {
		res, err := rf.o.ResolveInboxItem(id, "done", true)
		if err != nil || res.NoDecisionFound || res.SubmitUnconfirmed {
			t.Errorf("resolve %s --without-decision = %+v, %v; want resolved with no note", id, res, err)
		}
	}
}

// AC-4: a `sent` that is not the approval DECISION approve prints, sent to the
// org's leader after the approval in a real send, is not counted, and the
// item is refused. Each case appends the one `sent` as Send records it
// (sentEvent), changed in the one way the case names.
func TestResolveInboxItem_SentsThatAreNotCounted(t *testing.T) {
	decision := func(rf *resolveFixture) string { return ApprovalDecisionText(rf.ref) }
	cases := []struct {
		name   string
		before bool // appended before the approval
		event  func(rf *resolveFixture) ManifestEvent
	}{
		{"a dry run", false, func(rf *resolveFixture) ManifestEvent {
			ev := sentEvent("org-a", LeaderIdentity, decision(rf))
			ev.DryRun = true
			return ev
		}},
		{"a send before the approval", true, func(rf *resolveFixture) ManifestEvent {
			return sentEvent("org-a", LeaderIdentity, decision(rf))
		}},
		{"a send to a seat that is not the leader", false, func(rf *resolveFixture) ManifestEvent {
			return sentEvent("org-a", "reviewer", decision(rf))
		}},
		{"a send in another org", false, func(rf *resolveFixture) ManifestEvent {
			return sentEvent("org-b", LeaderIdentity, decision(rf))
		}},
		{"the DECISION of another plan", false, func(rf *resolveFixture) ManifestEvent {
			return sentEvent("org-a", LeaderIdentity, ApprovalDecisionText(PlanRef{Path: "docs/plans/active/other.md", Digest: rf.ref.Digest}))
		}},
		{"a DECISION of the same plan that asks for changes", false, func(rf *resolveFixture) ManifestEvent {
			return sentEvent("org-a", LeaderIdentity, "TYPE: DECISION\nPLAN: "+rf.ref.String()+"\n\nnot approved yet: split AC1 in two")
		}},
		{"another TYPE", false, func(rf *resolveFixture) ManifestEvent {
			return sentEvent("org-a", LeaderIdentity, "TYPE: QUESTION\nPLAN: "+rf.ref.String()+"\n\napproved")
		}},
		{"a send an older ralph recorded, with no text_sha256", false, func(rf *resolveFixture) ManifestEvent {
			ev := sentEvent("org-a", LeaderIdentity, decision(rf))
			ev.TextSHA256 = ""
			if ev.Details != decision(rf) {
				panic("the DECISION must fit in Details whole for this case")
			}
			return ev
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rf := newResolveFixture(t)
			if tc.before {
				rf.appendEvents(t, tc.event(rf))
				rf.approve(t)
			} else {
				rf.approve(t)
				rf.appendEvents(t, tc.event(rf))
			}
			before := fileOrAbsent(t, rf.o.Inbox.Path())
			_, err := rf.o.ResolveInboxItem(rf.id, "approved", false)
			rf.assertRefused(t, err, rf.id, rf.ref, before, InboxStateOpen)

			// The control: the same fixture with the approval DECISION
			// recorded as Send records it resolves.
			rf.appendEvents(t, sentEvent("org-a", LeaderIdentity, decision(rf)))
			if _, err := rf.o.ResolveInboxItem(rf.id, "approved", false); err != nil {
				t.Fatalf("resolve after the approval DECISION: %v", err)
			}
		})
	}
}

// AC-4 with plan paths of 160 and 180 bytes, where the `sent` Details, cut at
// 200 bytes, are the same for the approval DECISION, a DECISION that asks for
// changes, and the DECISION of another digest: the two others, sent through
// Send, are refused, and the approval DECISION resolves.
func TestResolveInboxItem_LongPlanPathsAreToldApartByTheWholeText(t *testing.T) {
	for _, n := range []int{160, 180} {
		t.Run(strconv.Itoa(n)+" bytes", func(t *testing.T) {
			rf := newResolveFixture(t)
			prefix := "docs/plans/active/"
			rel := prefix + strings.Repeat("p", n-len(prefix)-len(".md")) + ".md"
			if len(rel) != n {
				t.Fatalf("plan path is %d bytes; want %d", len(rel), n)
			}
			ref := PlanRef{Path: rel, Digest: PlanDigest(writePlan(t, rf.worktree, rel, testPlanDraft))}
			rf.id = rf.ask(t, "QUESTION", planQuestion(ref))
			rf.approve(t)

			last := ref.Digest[len(ref.Digest)-1:]
			flipped := "0"
			if last == "0" {
				flipped = "1"
			}
			other := PlanRef{Path: rel, Digest: ref.Digest[:len(ref.Digest)-1] + flipped}
			approval := ApprovalDecisionText(ref)
			for _, text := range []string{"TYPE: DECISION\nPLAN: " + ref.String() + "\n\nnot approved yet: split AC1 in two", ApprovalDecisionText(other)} {
				if truncateForDetails(text) != truncateForDetails(approval) {
					t.Fatalf("Details of %q differ from the approval's; the case needs them the same", text)
				}
				rf.send(t, LeaderIdentity, text)
				before := fileOrAbsent(t, rf.o.Inbox.Path())
				_, err := rf.o.ResolveInboxItem(rf.id, "approved", false)
				rf.assertRefused(t, err, rf.id, ref, before, InboxStateOpen)
			}

			rf.send(t, LeaderIdentity, approval)
			if _, err := rf.o.ResolveInboxItem(rf.id, "approved", false); err != nil {
				t.Fatalf("resolve after the approval DECISION: %v", err)
			}
		})
	}
}

// AC-4: an approval DECISION whose send could not confirm the submit
// (submit_unconfirmed=true) counts, and the result says so; once a send of it
// confirmed the submit, it no longer does.
func TestResolveInboxItem_SubmitUnconfirmedCountsAndIsReported(t *testing.T) {
	for _, confirmedToo := range []bool{false, true} {
		t.Run("confirmed too: "+strconv.FormatBool(confirmedToo), func(t *testing.T) {
			rf := newResolveFixture(t)
			rf.approve(t)
			// The two idle/done waits pass; the confirm wait after Enter fails.
			rf.h.agentWaitErrs = []error{nil, nil, errors.New("stub: confirm wait failed")}
			if r := rf.send(t, LeaderIdentity, ApprovalDecisionText(rf.ref)); r.SubmitConfirmed {
				t.Fatalf("send = %+v; want the submit unconfirmed", r)
			}
			if confirmedToo {
				rf.send(t, LeaderIdentity, ApprovalDecisionText(rf.ref))
			}
			res, err := rf.o.ResolveInboxItem(rf.id, "approved", false)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if want := (InboxResolveResult{OrgID: "org-a", Plan: rf.ref, SubmitUnconfirmed: !confirmedToo}); res != want {
				t.Errorf("result = %+v; want %+v", res, want)
			}
		})
	}
}

// sentSubmitUnconfirmed reads submit_unconfirmed=true only among the words
// Send writes before the text.
func TestResolveInboxItem_SentSubmitUnconfirmedReadsTheLeadingWords(t *testing.T) {
	text := "TYPE: DECISION\nPLAN: docs/p.md sha256:0123456789ab\n\napproved"
	for _, tc := range []struct {
		details string
		want    bool
	}{
		{"submit_unconfirmed=true " + text, true},
		{"raw=true via=screen herdr_state=working submit_unconfirmed=true " + text, true},
		{"via=screen herdr_state=working " + text, false},
		{text, false},
		{"TYPE: DECISION\n\nsubmit_unconfirmed=true submit_unconfirmed=true", false},
		{"", false},
	} {
		if got := sentSubmitUnconfirmed(tc.details); got != tc.want {
			t.Errorf("sentSubmitUnconfirmed(%q) = %v; want %v", tc.details, got, tc.want)
		}
	}
}

// AC-5: items the check leaves alone resolve as before, with an empty result:
// a QUESTION with no PLAN: header, a plan QUESTION whose approval is not
// recorded, an approval from before the org's latest disband, and, with an
// approval record naming them (only a hand-edited ledger has one), a BLOCKED
// item and a QUESTION with no PLAN: header.
func TestResolveInboxItem_ItemsWithNoApprovalResolveAsBefore(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, rf *resolveFixture) string
	}{
		{"a QUESTION with no PLAN: header", func(t *testing.T, rf *resolveFixture) string {
			return rf.ask(t, "QUESTION", "TYPE: QUESTION\n\nwhich base branch?")
		}},
		{"a plan QUESTION whose approval is not recorded", func(t *testing.T, rf *resolveFixture) string {
			return rf.id
		}},
		{"an approval from before the org's latest disband", func(t *testing.T, rf *resolveFixture) string {
			rf.approve(t)
			rf.appendEvents(t, ManifestEvent{OrgID: "org-a", Event: EventDisbanded})
			return rf.id
		}},
		{"a BLOCKED item with an approval record", func(t *testing.T, rf *resolveFixture) string {
			id := rf.ask(t, "BLOCKED", "TYPE: BLOCKED\nTASK_ID: org-a\nPLAN: "+rf.ref.String()+"\n\nstuck")
			rf.appendEvents(t, approvedEvent("org-a", id, rf.ref, PlanContentHash([]byte(testPlanDraft))))
			return id
		}},
		{"a QUESTION with no PLAN: header and an approval record", func(t *testing.T, rf *resolveFixture) string {
			id := rf.ask(t, "QUESTION", "TYPE: QUESTION\n\nPLAN: "+rf.ref.String())
			rf.appendEvents(t, approvedEvent("org-a", id, rf.ref, PlanContentHash([]byte(testPlanDraft))))
			return id
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rf := newResolveFixture(t)
			id := tc.setup(t, rf)
			res, err := rf.o.ResolveInboxItem(id, "docs/reports/x.md", false)
			if err != nil {
				t.Fatalf("resolve %s: %v", id, err)
			}
			if res != (InboxResolveResult{}) {
				t.Errorf("result = %+v; want an empty one", res)
			}
			if got := rf.item(t, id); got.State != InboxStateResolved || got.Note != "docs/reports/x.md" {
				t.Errorf("item = %+v; want it resolved", got)
			}
		})
	}
}

// An unknown ID, a resolved item, and a note that does not pass keep the
// errors InboxStore.Resolve gives, and write nothing; a note is checked before
// the approval, and with no inbox file the ID is unknown and nothing is
// created, not even the state dir.
func TestResolveInboxItem_UnknownResolvedAndBadNoteKeepTheirErrors(t *testing.T) {
	rf := newResolveFixture(t)
	resolved := rf.ask(t, "QUESTION", "TYPE: QUESTION\n\nwhich?")
	if _, err := rf.o.ResolveInboxItem(resolved, "done", false); err != nil {
		t.Fatal(err)
	}
	rf.approve(t)
	before := fileOrAbsent(t, rf.o.Inbox.Path())
	for _, tc := range []struct {
		id, note string
		want     string
		is       error
	}{
		{"e99", "done", "org: inbox: resolve e99: no such inbox item", ErrInboxUnknownID},
		{resolved, "done", "org: inbox: resolve " + resolved + ": the item is already resolved", ErrInboxResolved},
		{rf.id, "", "org: inbox: the resolve note is required: give a pointer such as a report path or a PR URL", nil},
		{rf.id, "two\nlines", "org: inbox: the resolve note must be a single line", nil},
	} {
		_, err := rf.o.ResolveInboxItem(tc.id, tc.note, false)
		storeErr := rf.o.Inbox.Resolve(tc.id, tc.note)
		if err == nil || err.Error() != tc.want || storeErr == nil || storeErr.Error() != tc.want || (tc.is != nil && !errors.Is(err, tc.is)) {
			t.Errorf("resolve %s with note %q = %v (InboxStore.Resolve: %v); want %q", tc.id, tc.note, err, storeErr, tc.want)
		}
	}
	if after := fileOrAbsent(t, rf.o.Inbox.Path()); after != before {
		t.Errorf("a refused resolve wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
	}

	missing := filepath.Join(t.TempDir(), "no-ledger")
	o := &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(missing)), Inbox: NewInboxStore(missing)}
	if _, err := o.ResolveInboxItem("e1", "done", false); !errors.Is(err, ErrInboxUnknownID) || err.Error() != "org: inbox: resolve e1: no such inbox item" {
		t.Errorf("resolve with no inbox file = %v; want ErrInboxUnknownID", err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("resolve with no inbox file created %s (stat: %v)", missing, err)
	}
}

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

// AC-9: the `sent` event of a real send carries the sha256 of the whole text
// as written, also when Details cut it, and without the bracketed paste
// markers typed around it for the claude leader; a dry run's has none, and
// the field is left out of a line that has none.
func TestOrgSend_SentEventRecordsTheTextSHA256(t *testing.T) {
	rf := newResolveFixture(t)
	text := "TYPE: QUESTION\n\n" + strings.Repeat("which base branch? ", 20)
	rf.send(t, LeaderIdentity, text)
	if r := rf.o.Send(SendParams{OrgID: "org-a", To: LeaderIdentity, Text: text, DryRun: true}); r.Err != nil {
		t.Fatalf("dry-run send: %v", r.Err)
	}
	events := mustReadEvents(t, rf.o)
	sent, dry := events[len(events)-2], events[len(events)-1]
	sum := sha256.Sum256([]byte(text))
	if sent.Event != EventSent || sent.DryRun || sent.TextSHA256 != hex.EncodeToString(sum[:]) || !strings.HasSuffix(sent.Details, "...(truncated)") {
		t.Errorf("sent = %+v; want text_sha256 %x and Details cut", sent, sum)
	}
	if dry.Event != EventSent || !dry.DryRun || dry.TextSHA256 != "" {
		t.Errorf("dry-run sent = %+v; want no text_sha256", dry)
	}
	raw, err := os.ReadFile(rf.o.Manifest.Path())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if got := lines[len(lines)-2]; !strings.Contains(got, `"text_sha256":"`+hex.EncodeToString(sum[:])+`"`) {
		t.Errorf("sent line = %s; want its text_sha256", got)
	}
	if got := lines[len(lines)-1]; strings.Contains(got, "text_sha256") {
		t.Errorf("dry-run sent line = %s; want no text_sha256 key", got)
	}
}
