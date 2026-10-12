package org

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Approve at the org layer (plan
// docs/plans/active/2026-10-12-org-plan-approval.md, AC2, AC3 and AC4). The
// fixtures it shares with the tests of the check (approvalFixture, writePlan,
// planQuestion, ...) are in approval_test.go.

// approveSetup is an approval fixture with org-a bound in per-plan mode to a
// worktree holding the draft plan, and the leader's QUESTION for it.
type approveSetup struct {
	*approvalFixture
	worktree string
	f        *FeatureBinding
	ref      PlanRef
	id       string
}

func newApproveSetup(t *testing.T) *approveSetup {
	t.Helper()
	s := &approveSetup{approvalFixture: newApprovalFixture(t), worktree: t.TempDir()}
	s.f = s.bind(t, "org-a", s.worktree, config.ApprovalModePerPlan)
	data := writePlan(t, s.worktree, testPlanRel, testPlanDraft)
	s.ref = PlanRef{Path: testPlanRel, Digest: PlanDigest(data)}
	s.id = s.ask(t, "org-a", "QUESTION", planQuestion(s.ref))
	return s
}

// TestOrgApprove_RecordsResolvesAndLeavesThePlan covers AC2 at the org
// layer: one plan_approved line with every Details key, the item resolved
// with its note, the plan file untouched, and the check passing once the
// leader writes the two header lines.
func TestOrgApprove_RecordsResolvesAndLeavesThePlan(t *testing.T) {
	s := newApproveSetup(t)
	planPath := filepath.Join(s.worktree, filepath.FromSlash(testPlanRel))
	before := fileOrAbsent(t, planPath)
	eventsBefore := len(mustReadEvents(t, s.o))
	s.env[herdrPaneIDEnv] = "p-human"

	res := s.o.Approve(ApproveParams{IDs: []string{s.id}, Cwd: "/work dir"})
	if res.Err != nil || len(res.Refused) != 0 {
		t.Fatalf("Approve: %+v", res)
	}
	want := ApprovedPlan{ID: s.id, OrgID: "org-a", Plan: testPlanRel, Digest: s.ref.Digest, RequiresApproval: true, Resolved: true}
	if len(res.Approved) != 1 || res.Approved[0] != want {
		t.Fatalf("Approved = %+v, want [%+v]", res.Approved, want)
	}
	events := mustReadEvents(t, s.o)
	if len(events) != eventsBefore+1 {
		t.Fatalf("expected exactly one new manifest line, got %d: %+v", len(events)-eventsBefore, events[eventsBefore:])
	}
	ev := events[len(events)-1]
	wantDetails := "plan=" + testPlanRel + " digest=" + s.ref.Digest + " content=" + PlanContentHash([]byte(testPlanDraft)) +
		" inbox=" + s.id + ` by=alice herdr_pane=set cwd="/work dir"`
	if ev.Event != EventPlanApproved || ev.OrgID != "org-a" || ev.SeatID != "" || ev.DryRun || ev.Details != wantDetails ||
		ev.TS != "2026-10-12T12:00:00Z" {
		t.Fatalf("plan_approved = %+v, want Details %q", ev, wantDetails)
	}
	item := s.item(t, s.id)
	if item.State != InboxStateResolved || item.Note != "approved "+testPlanRel+" sha256:"+s.ref.Digest {
		t.Fatalf("inbox item = %+v, want resolved with the approval note", item)
	}
	if after := fileOrAbsent(t, planPath); after != before {
		t.Fatalf("Approve changed the plan file:\n%s", after)
	}

	err := checkPlanApproved(events, "org-a", s.f, s.ref)
	if err == nil || !strings.Contains(err.Error(), "is not marked approved") {
		t.Fatalf("before the leader writes the two lines, checkPlanApproved = %v, want the Status refusal", err)
	}
	writePlan(t, s.worktree, testPlanRel, approvedPlanText(testPlanDraft, s.ref.Digest))
	if err := checkPlanApproved(events, "org-a", s.f, s.ref); err != nil {
		t.Fatalf("after the leader writes the two lines, checkPlanApproved: %v", err)
	}
	if err := checkAnyPlanApproved(events, "org-a", s.f); err != nil {
		t.Fatalf("checkAnyPlanApproved: %v", err)
	}
}

// TestOrgApprove_Batch records every item of a batch, in argument order, an
// item of an org in split mode or started by an older start included and
// marked, and an acked item too.
func TestOrgApprove_Batch(t *testing.T) {
	s := newApproveSetup(t)
	ids := []string{s.id}
	modes := map[string]string{"org-b": config.ApprovalModeSplit, "org-c": ""}
	for _, orgID := range []string{"org-b", "org-c"} {
		wt := t.TempDir()
		s.bind(t, orgID, wt, modes[orgID])
		data := writePlan(t, wt, "plan.md", "# "+orgID+"\n")
		ids = append(ids, s.ask(t, orgID, "QUESTION", planQuestion(PlanRef{"plan.md", PlanDigest(data)})))
	}
	if _, err := s.o.Inbox.Ack(ids[1]); err != nil {
		t.Fatal(err)
	}
	ids[0], ids[2] = ids[2], ids[0]

	res := s.o.Approve(ApproveParams{IDs: ids, Cwd: "/w"})
	if res.Err != nil {
		t.Fatalf("Approve: %v", res.Err)
	}
	if len(res.Approved) != 3 {
		t.Fatalf("Approved = %+v, want 3", res.Approved)
	}
	for i, wantOrg := range []string{"org-c", "org-b", "org-a"} {
		a := res.Approved[i]
		if a.ID != ids[i] || a.OrgID != wantOrg || !a.Resolved || a.RequiresApproval != (wantOrg == "org-a") {
			t.Fatalf("Approved[%d] = %+v, want item %s of %s resolved, RequiresApproval only for org-a", i, a, ids[i], wantOrg)
		}
	}
	got := planApprovedEvents(t, s.o)
	if len(got) != 3 {
		t.Fatalf("expected 3 plan_approved lines, got %+v", got)
	}
	for i, ev := range got {
		if ev.OrgID != res.Approved[i].OrgID || !strings.Contains(ev.Details, " inbox="+ids[i]+" ") {
			t.Fatalf("plan_approved %d = %+v, want org %s item %s", i, ev, res.Approved[i].OrgID, ids[i])
		}
	}
}

// TestOrgApprove_RecordsTheFileAsItIsThen covers the A9 note: a line added
// where the digest does not look, between the QUESTION and approve, is part
// of what approve records, so the check passes on that file and refuses the
// one the leader asked about.
func TestOrgApprove_RecordsTheFileAsItIsThen(t *testing.T) {
	s := newApproveSetup(t)
	edited := testPlanDraft + "- [ ] a line added before approve\n"
	writePlan(t, s.worktree, testPlanRel, edited)
	if PlanDigest([]byte(edited)) != s.ref.Digest {
		t.Fatal("the edit must be one the digest does not show")
	}
	if res := s.o.Approve(ApproveParams{IDs: []string{s.id}, Cwd: "/w"}); res.Err != nil {
		t.Fatalf("Approve: %v", res.Err)
	}
	if got := planApprovedEvents(t, s.o); len(got) != 1 || !strings.Contains(got[0].Details, " content="+PlanContentHash([]byte(edited))+" ") {
		t.Fatalf("plan_approved = %+v, want the content of the file approve read", got)
	}
	writePlan(t, s.worktree, testPlanRel, approvedPlanText(edited, s.ref.Digest))
	if err := checkPlanApproved(mustReadEvents(t, s.o), "org-a", s.f, s.ref); err != nil {
		t.Fatalf("checkPlanApproved on the file approve recorded: %v", err)
	}
	writePlan(t, s.worktree, testPlanRel, approvedPlanText(testPlanDraft, s.ref.Digest))
	if err := checkPlanApproved(mustReadEvents(t, s.o), "org-a", s.f, s.ref); err == nil || !strings.Contains(err.Error(), "outside its - Status: and - Approved: lines") {
		t.Fatalf("checkPlanApproved on the file as asked = %v, want the content refusal", err)
	}
}

// TestOrgApprove_Refusals covers AC3: each refusal names the item and its
// reason, and nothing is written to the manifest or the inbox.
func TestOrgApprove_Refusals(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T, s *approveSetup) string // returns the ID to approve
		wantErr string
		wantIs  error
	}{
		{"an unknown ID", func(t *testing.T, s *approveSetup) string { return "e99" }, "approve e99", ErrInboxUnknownID},
		{"an ID not of the form e<N>", func(t *testing.T, s *approveSetup) string { return "E1" }, "approve E1", ErrInboxUnknownID},
		{"a resolved item", func(t *testing.T, s *approveSetup) string {
			if err := s.o.Inbox.Resolve(s.id, "done elsewhere"); err != nil {
				t.Fatal(err)
			}
			return s.id
		}, "", ErrInboxResolved},
		{"a BLOCKED item", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-a", "BLOCKED", "TYPE: BLOCKED\nTASK_ID: org-a\nPLAN: "+s.ref.String()+"\n\nstuck")
		}, "the item is a BLOCKED, not a QUESTION", nil},
		{"no PLAN header", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-a", "QUESTION", "TYPE: QUESTION\n\nPLAN: "+s.ref.String())
		}, "no PLAN: header", nil},
		{"two PLAN headers", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-a", "QUESTION", "TYPE: QUESTION\nPLAN: "+s.ref.String()+"\nPLAN: "+s.ref.String()+"\n\nok?")
		}, "2 PLAN: headers", nil},
		{"a PLAN of the wrong shape", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-a", "QUESTION", "TYPE: QUESTION\nPLAN: "+testPlanRel+"\n\nok?")
		}, "not the two words", nil},
		{"an absolute path", func(t *testing.T, s *approveSetup) string {
			abs := filepath.Join(s.worktree, filepath.FromSlash(testPlanRel))
			return s.ask(t, "org-a", "QUESTION", planQuestion(PlanRef{abs, s.ref.Digest}))
		}, "is absolute", nil},
		{"a .. path", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-a", "QUESTION", planQuestion(PlanRef{"../org-a/" + testPlanRel, s.ref.Digest}))
		}, "has a .. element", nil},
		{"an org with no binding", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-x", "QUESTION", planQuestion(s.ref))
		}, `org_id "org-x" holds no split plan feature binding`, nil},
		{"an org with a plain reservation", func(t *testing.T, s *approveSetup) string {
			s.appendEvents(t, reserveEvent("org-p", "docs/"))
			return s.ask(t, "org-p", "QUESTION", planQuestion(s.ref))
		}, `org_id "org-p" holds no split plan feature binding`, nil},
		{"a disbanded org", func(t *testing.T, s *approveSetup) string {
			s.appendEvents(t, ManifestEvent{OrgID: "org-a", Event: EventDisbanded})
			return s.id
		}, `org_id "org-a" holds no split plan feature binding`, nil},
		{"an incomplete binding", func(t *testing.T, s *approveSetup) string {
			g := *s.f
			g.Worktree = ""
			s.appendEvents(t, boundReserveEvent("org-i", &g, "internal/org-i/"))
			return s.ask(t, "org-i", "QUESTION", planQuestion(s.ref))
		}, "read from an incomplete record", nil},
		{"a symlink outside the worktree", func(t *testing.T, s *approveSetup) string {
			outside := t.TempDir()
			writePlan(t, outside, "plan.md", testPlanDraft)
			if err := os.Symlink(filepath.Join(outside, "plan.md"), filepath.Join(s.worktree, "linked.md")); err != nil {
				t.Fatal(err)
			}
			return s.ask(t, "org-a", "QUESTION", planQuestion(PlanRef{"linked.md", s.ref.Digest}))
		}, "outside the feature worktree", nil},
		{"a missing file", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-a", "QUESTION", planQuestion(PlanRef{"docs/missing.md", s.ref.Digest}))
		}, "no such file or directory", nil},
		{"a directory", func(t *testing.T, s *approveSetup) string {
			return s.ask(t, "org-a", "QUESTION", planQuestion(PlanRef{"docs", s.ref.Digest}))
		}, "is not a regular file", nil},
		{"the plan changed after the QUESTION", func(t *testing.T, s *approveSetup) string {
			writePlan(t, s.worktree, testPlanRel, strings.Replace(testPlanDraft, "Add the login form.", "Add the signup form.", 1))
			return s.id
		}, "the plan changed after the leader asked", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newApproveSetup(t)
			id := tc.setup(t, s)
			before := s.ledgerBytes(t)

			res := s.o.Approve(ApproveParams{IDs: []string{id}, Cwd: "/w"})
			if len(res.Refused) != 1 || res.Refused[0].ID != id || len(res.Approved) != 0 || res.Err == nil {
				t.Fatalf("Approve = %+v, want one refusal of %s and nothing approved", res, id)
			}
			for _, err := range []error{res.Refused[0].Err, res.Err} {
				if !strings.Contains(err.Error(), tc.wantErr) || (tc.wantIs != nil && !errors.Is(err, tc.wantIs)) {
					t.Fatalf("error %q: want it to contain %q (and wrap %v)", err, tc.wantErr, tc.wantIs)
				}
			}
			if !strings.HasPrefix(res.Err.Error(), "org: approve: nothing recorded, since 1 of the 1 inbox items cannot be approved\n") {
				t.Fatalf("Err = %q, want the nothing-recorded summary first", res.Err)
			}
			if after := s.ledgerBytes(t); after != before {
				t.Fatal("a refused approve wrote to the manifest or the inbox")
			}
		})
	}
}

// TestOrgApprove_AllOrNothing covers A4: when some items of a batch fail,
// none is recorded or resolved, and every failure is listed in argument
// order.
func TestOrgApprove_AllOrNothing(t *testing.T) {
	s := newApproveSetup(t)
	wt := t.TempDir()
	s.bind(t, "org-b", wt, config.ApprovalModePerPlan)
	data := writePlan(t, wt, "plan.md", "# b\n")
	good := s.ask(t, "org-b", "QUESTION", planQuestion(PlanRef{"plan.md", PlanDigest(data)}))
	bad := s.ask(t, "org-b", "QUESTION", planQuestion(PlanRef{"plan.md", "ffffffffffff"}))
	before := s.ledgerBytes(t)

	t.Run("one of three", func(t *testing.T) {
		res := s.o.Approve(ApproveParams{IDs: []string{s.id, bad, good}})
		if len(res.Approved) != 0 || len(res.Refused) != 1 || res.Refused[0].ID != bad || res.Err == nil ||
			!strings.Contains(res.Err.Error(), "1 of the 3 inbox items cannot be approved") {
			t.Fatalf("Approve = %+v, want only %s refused and nothing recorded", res, bad)
		}
		if s.ledgerBytes(t) != before {
			t.Fatal("a batch with a refused item wrote to the manifest or the inbox")
		}
	})
	t.Run("two of three, both listed in order", func(t *testing.T) {
		res := s.o.Approve(ApproveParams{IDs: []string{"e77", s.id, bad}})
		if len(res.Approved) != 0 || len(res.Refused) != 2 || res.Refused[0].ID != "e77" || res.Refused[1].ID != bad {
			t.Fatalf("Approve = %+v, want e77 and %s refused", res, bad)
		}
		msg := res.Err.Error()
		if i, j := strings.Index(msg, "approve e77:"), strings.Index(msg, "approve "+bad+":"); i < 0 || j < i {
			t.Fatalf("Err = %q, want both refusals listed in order", msg)
		}
		if s.ledgerBytes(t) != before {
			t.Fatal("a batch with refused items wrote to the manifest or the inbox")
		}
	})
}

func TestOrgApprove_IDList(t *testing.T) {
	s := newApproveSetup(t)
	before := s.ledgerBytes(t)
	for _, tc := range []struct {
		name string
		ids  []string
		want string
	}{
		{"no ID", nil, "name at least one inbox item ID"},
		{"an ID given twice", []string{s.id, s.id}, "inbox item " + s.id + " is given twice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := s.o.Approve(ApproveParams{IDs: tc.ids})
			if res.Err == nil || !strings.Contains(res.Err.Error(), tc.want) || len(res.Approved)+len(res.Refused) != 0 {
				t.Fatalf("Approve = %+v, want an error containing %q", res, tc.want)
			}
			if s.ledgerBytes(t) != before {
				t.Fatal("the refused ID list wrote to the manifest or the inbox")
			}
		})
	}
}

// TestOrgApprove_Tripwire covers AC4: HERDR_PANE_ID equal to an active real
// seat's pane refuses before the inbox is read (the inbox here cannot be
// read at all), writing nothing; unset, another pane, a stopped seat's pane
// and a dry-run seat's pane pass.
func TestOrgApprove_Tripwire(t *testing.T) {
	seat := func(orgID, seatID, event, pane string, dryRun bool) ManifestEvent {
		return ManifestEvent{OrgID: orgID, SeatID: seatID, Event: event, Role: "worker", PaneID: pane, DryRun: dryRun}
	}
	t.Run("an active seat's pane refuses first", func(t *testing.T) {
		s := newApproveSetup(t)
		s.appendEvents(t, seat("org-b", "seat-1", EventSpawned, "p-7", false))
		inboxPath := s.o.Inbox.Path()
		if err := os.Rename(inboxPath, inboxPath+".bak"); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(inboxPath, 0o755); err != nil {
			t.Fatal(err)
		}
		before := fileOrAbsent(t, s.o.Manifest.Path())
		s.env[herdrPaneIDEnv] = "p-7"
		res := s.o.Approve(ApproveParams{IDs: []string{s.id}})
		if res.Err == nil || !strings.Contains(res.Err.Error(), `HERDR_PANE_ID "p-7" is the pane of the active seat "seat-1" of org_id "org-b"`) ||
			!strings.Contains(res.Err.Error(), "a terminal outside the seats' panes") || len(res.Approved)+len(res.Refused) != 0 {
			t.Fatalf("Approve = %+v, want the tripwire refusal", res)
		}
		if fileOrAbsent(t, s.o.Manifest.Path()) != before {
			t.Fatal("the tripwire refusal wrote to the manifest")
		}
		delete(s.env, herdrPaneIDEnv)
		if res := s.o.Approve(ApproveParams{IDs: []string{s.id}}); res.Err == nil || !strings.Contains(res.Err.Error(), "read") {
			t.Fatalf("with no pane the unreadable inbox must be read and fail, got %+v", res)
		}
	})
	for _, tc := range []struct {
		name string
		pane string
		evs  []ManifestEvent
		want string // the herdr_pane value recorded
	}{
		{"HERDR_PANE_ID unset", "", []ManifestEvent{seat("org-b", "seat-1", EventSpawned, "p-7", false)}, "unset"},
		{"another pane", "p-8", []ManifestEvent{seat("org-b", "seat-1", EventSpawned, "p-7", false)}, "set"},
		{"a stopped seat's pane", "p-7", []ManifestEvent{seat("org-b", "seat-1", EventSpawned, "p-7", false), seat("org-b", "seat-1", EventStopped, "p-7", false)}, "set"},
		{"a dry-run seat's pane", "p-7", []ManifestEvent{seat("org-b", "seat-1", EventSpawned, "p-7", true)}, "set"},
		{"a seat of a disbanded org", "p-7", []ManifestEvent{seat("org-b", "seat-1", EventSpawned, "p-7", false), {OrgID: "org-b", Event: EventDisbanded}}, "set"},
	} {
		t.Run(tc.name+" passes", func(t *testing.T) {
			s := newApproveSetup(t)
			s.appendEvents(t, tc.evs...)
			if tc.pane != "" {
				s.env[herdrPaneIDEnv] = tc.pane
			}
			res := s.o.Approve(ApproveParams{IDs: []string{s.id}, Cwd: "/w"})
			if res.Err != nil || len(res.Approved) != 1 {
				t.Fatalf("Approve = %+v, want it recorded", res)
			}
			if got := planApprovedEvents(t, s.o); len(got) != 1 || !strings.Contains(got[0].Details, " herdr_pane="+tc.want+" ") {
				t.Fatalf("plan_approved = %+v, want herdr_pane=%s", got, tc.want)
			}
		})
	}
}

// TestOrgApprove_ByFromUser records USER, and unknown when it is unset.
func TestOrgApprove_ByFromUser(t *testing.T) {
	for _, tc := range []struct{ user, want string }{{"bob", "by=bob"}, {"", "by=unknown"}} {
		s := newApproveSetup(t)
		s.env["USER"] = tc.user
		if res := s.o.Approve(ApproveParams{IDs: []string{s.id}, Cwd: "/w"}); res.Err != nil {
			t.Fatalf("Approve: %v", res.Err)
		}
		if got := planApprovedEvents(t, s.o); len(got) != 1 || !strings.Contains(got[0].Details, " "+tc.want+" ") {
			t.Fatalf("USER %q: plan_approved = %+v, want %s", tc.user, got, tc.want)
		}
	}
}

// TestOrgApprove_ResolveFailureKeepsTheRecord: when the inbox item cannot be
// resolved after the record, the record stays and the error names the item
// and the resolve command that finishes it, with --state-dir when the ledger
// was named with it.
func TestOrgApprove_ResolveFailureKeepsTheRecord(t *testing.T) {
	s := newApproveSetup(t)
	s.o.InboxHintStateDir = s.stateDir
	lock := filepath.Join(s.stateDir, inboxLockFile)
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}

	res := s.o.Approve(ApproveParams{IDs: []string{s.id}, Cwd: "/w"})
	if len(res.Approved) != 1 || res.Approved[0].Resolved || res.Err == nil {
		t.Fatalf("Approve = %+v, want the item recorded, not resolved, and an error", res)
	}
	wantCmd := "`ralph org inbox resolve " + s.id + " --state-dir " + shellQuote(s.stateDir) +
		" --note 'approved " + testPlanRel + " sha256:" + s.ref.Digest + "'`"
	if msg := res.Err.Error(); !strings.Contains(msg, "the approval is recorded, but the inbox item could not be resolved: "+s.id+" (") ||
		!strings.Contains(msg, "run "+wantCmd+" to finish") {
		t.Fatalf("Err = %q, want the item and %s", msg, wantCmd)
	}
	if got := planApprovedEvents(t, s.o); len(got) != 1 {
		t.Fatalf("expected the record to stay, got %+v", got)
	}
	if item := s.item(t, s.id); item.State != InboxStateOpen {
		t.Fatalf("item = %+v, want it still open", item)
	}
}

// TestOrgApprove_ResolvedBySomeoneElse covers self-review L3 of
// org-plan-approval: when someone else resolves an item between approve's
// read of the inbox and its own resolve (Inbox.Resolve wraps
// ErrInboxResolved), the record stays, the error says the item was already
// resolved by someone else and the approval is recorded all the same, and it
// prints no `ralph org inbox resolve` command, which would fail. The other
// item of the batch is resolved as usual.
func TestOrgApprove_ResolvedBySomeoneElse(t *testing.T) {
	s := newApproveSetup(t)
	s.o.InboxHintStateDir = s.stateDir
	wt := t.TempDir()
	s.bind(t, "org-b", wt, config.ApprovalModePerPlan)
	data := writePlan(t, wt, "plan.md", "# b\n")
	secondRef := PlanRef{"plan.md", PlanDigest(data)}
	second := s.ask(t, "org-b", "QUESTION", planQuestion(secondRef))
	// Org.Now is first read for the first record's TS: after approve read
	// the inbox, before it resolves. A human's `ralph org inbox resolve`
	// lands there.
	clock, raced := s.o.Now, false
	s.o.Now = func() time.Time {
		if !raced {
			raced = true
			if err := s.o.Inbox.Resolve(s.id, "handled by hand"); err != nil {
				t.Errorf("the racing resolve: %v", err)
			}
		}
		return clock()
	}

	res := s.o.Approve(ApproveParams{IDs: []string{s.id, second}, Cwd: "/w"})
	if !raced {
		t.Fatal("Org.Now was never read between the inbox read and the resolve, so nothing raced approve")
	}
	if len(res.Approved) != 2 || res.Approved[0].Resolved || !res.Approved[1].Resolved || res.Err == nil {
		t.Fatalf("Approve = %+v, want both recorded, only %s resolved by approve, and an error", res, second)
	}
	msg := res.Err.Error()
	if want := "the approval is recorded all the same, but the inbox item was already resolved by someone else " +
		"after approve read the inbox, so nothing is left to resolve: " + s.id; !strings.Contains(msg, want) {
		t.Fatalf("Err = %q, want it to contain %q", msg, want)
	}
	if strings.Contains(msg, "ralph org inbox resolve") || strings.Contains(msg, "--note") || strings.Contains(msg, second) {
		t.Fatalf("Err = %q, want no resolve command and no word of the item approve resolved", msg)
	}
	if got := planApprovedEvents(t, s.o); len(got) != 2 {
		t.Fatalf("expected both records to stay, got %+v", got)
	}
	if item := s.item(t, s.id); item.State != InboxStateResolved || item.Note != "handled by hand" {
		t.Fatalf("item %s = %+v, want it resolved with the other resolve's note", s.id, item)
	}
	if item := s.item(t, second); item.State != InboxStateResolved || item.Note != approveNote(secondRef) {
		t.Fatalf("item %s = %+v, want it resolved with approve's note", second, item)
	}
}

// TestOrgApprove_AppendFailureRecordsNothing: when the first plan_approved
// cannot be appended, nothing is recorded or resolved, and the error names
// every item and the approve command to run again.
func TestOrgApprove_AppendFailureRecordsNothing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes a read-only file")
	}
	s := newApproveSetup(t)
	wt := t.TempDir()
	s.bind(t, "org-b", wt, config.ApprovalModePerPlan)
	data := writePlan(t, wt, "plan.md", "# b\n")
	second := s.ask(t, "org-b", "QUESTION", planQuestion(PlanRef{"plan.md", PlanDigest(data)}))
	s.o.InboxHintStateDir = s.stateDir
	if err := os.Chmod(s.o.Manifest.Path(), 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(s.o.Manifest.Path(), 0o644) })

	res := s.o.Approve(ApproveParams{IDs: []string{s.id, second}, Cwd: "/w"})
	if len(res.Approved) != 0 || res.Err == nil {
		t.Fatalf("Approve = %+v, want nothing approved and an error", res)
	}
	msg := res.Err.Error()
	for _, want := range []string{
		"recording the approval of " + s.id + " failed (",
		"nothing was recorded; not recorded: " + s.id + ", " + second,
		"run `ralph org approve " + s.id + " " + second + " --state-dir " + shellQuote(s.stateDir) + "` again",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Err = %q, want it to contain %q", msg, want)
		}
	}
	for _, id := range []string{s.id, second} {
		if item := s.item(t, id); item.State != InboxStateOpen {
			t.Fatalf("item %s = %+v, want it still open", id, item)
		}
	}
}
