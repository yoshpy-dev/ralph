package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The approval check in Send and Spawn (plan
// docs/plans/active/2026-10-12-org-plan-approval.md, AC5 to AC9, the org
// layer of AC13, and AC14): in an org that `ralph org start --plan` bound in
// per-plan mode, a TASK to an implementer seat and an implementer spawn need a
// plan a human approved, and in any bound org no seat but the leader is
// spawned while the leader is not running. approveFeaturePlan, testPlanDraft
// and the other plan helpers live in approval_test.go.

// enforceFixture is an Org on fake drivers whose org-a is bound in the given
// approval mode to a feature worktree holding testPlanDraft, with its leader
// spawned (pane-1 in ws-a).
type enforceFixture struct {
	o        *Org
	h        *fakeHerdr
	a        *fakeAgmsg
	worktree string
	f        *FeatureBinding
	ref      PlanRef // testPlanDraft's ref, the plan the leader asks to approve
}

func newEnforceFixture(t *testing.T, mode string) *enforceFixture {
	t.Helper()
	o, h, a := testOrg(t)
	fx := &enforceFixture{o: o, h: h, a: a, worktree: t.TempDir()}
	fx.f = testFeature("split-1", "auth", "0123456789ab")
	fx.f.Worktree, fx.f.Approval = fx.worktree, mode
	spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", fx.f, "internal/auth/")
	fx.ref = PlanRef{Path: testPlanRel, Digest: PlanDigest(writePlan(t, fx.worktree, testPlanRel, testPlanDraft))}
	return fx
}

// record appends the approval of testPlanDraft as Approve writes it.
func (fx *enforceFixture) record(t *testing.T) {
	t.Helper()
	if err := fx.o.Manifest.Append(approvedEvent("org-a", "e1", fx.ref, PlanContentHash([]byte(testPlanDraft)))); err != nil {
		t.Fatal(err)
	}
}

// mark writes the plan with the two header lines the leader writes after the
// approval of fx.ref, then edit (nil: none).
func (fx *enforceFixture) mark(t *testing.T, edit func(string) string) {
	t.Helper()
	plan := approvedPlanText(testPlanDraft, fx.ref.Digest)
	if edit != nil {
		plan = edit(plan)
	}
	writePlan(t, fx.worktree, testPlanRel, plan)
}

// replaceOnce returns an edit that replaces from with to once, and panics
// when the text has no from, so a case cannot pass on an edit that did
// nothing. The plan edits of the check tests (TestCheckPlanApproved) use it
// too.
func replaceOnce(from, to string) func(string) string {
	return func(s string) string {
		if !strings.Contains(s, from) {
			panic(fmt.Sprintf("the plan has no %q", from))
		}
		return strings.Replace(s, from, to, 1)
	}
}

// seedSeat records seatID with role in orgID as spawned on pane, the way a
// spawn that went through leaves it, without running Spawn and its checks:
// the Send tests use it for a seat whose approval state they set afterwards.
func seedSeat(t *testing.T, o *Org, orgID, seatID, role, pane string) {
	t.Helper()
	if err := o.Manifest.Append(ManifestEvent{
		TS: o.now(), OrgID: orgID, SeatID: seatID, Event: EventSpawned, Role: role, Driver: "claude", Model: "sonnet",
		PaneID: pane, HerdrAgentName: herdrAgentName(orgID, seatID),
	}); err != nil {
		t.Fatalf("seed %s/%s: %v", orgID, seatID, err)
	}
}

// implementerParams is mustSpawnParams for orgID's implementer seat.
func implementerParams(orgID string) SpawnParams {
	p := mustSpawnParams(orgID, implementerSeatID)
	p.Role = implementerRole
	return p
}

// reviewerParams is mustSpawnParams for orgID's reviewer seat.
func reviewerParams(orgID string) SpawnParams {
	p := mustSpawnParams(orgID, "reviewer")
	p.Role = "reviewer"
	return p
}

// planTask is a TASK with one PLAN: header per ref.
func planTask(refs ...PlanRef) string {
	text := "TYPE: TASK\nTASK_ID: t-1\n"
	for _, ref := range refs {
		text += "PLAN: " + ref.String() + "\n"
	}
	return text + "\nimplement the plan"
}

// ledgerSnapshot is what a refused send or spawn must leave as it was: the
// manifest's and the receipts' bytes and the fake drivers' call counts.
type ledgerSnapshot struct {
	manifest, receipts string
	herdr, agmsg       int
}

func takeLedgerSnapshot(t *testing.T, o *Org, h *fakeHerdr, a *fakeAgmsg) ledgerSnapshot {
	t.Helper()
	return ledgerSnapshot{fileOrAbsent(t, o.Manifest.Path()), fileOrAbsent(t, o.Receipts.Path()), len(h.calls), len(a.calls)}
}

// assertTaskRefused checks that r is the approval check's refusal of a TASK
// containing want: errors.Is ErrPlanNotApproved, "org: send: TASK not sent: "
// first, "org:" once, the marker and the next step, what a leader never does
// (--raw included) last, and nothing sent.
func assertTaskRefused(t *testing.T, r SendResult, want string) {
	t.Helper()
	if r.Err == nil || !errors.Is(r.Err, ErrPlanNotApproved) {
		t.Fatalf("send = %v; want a refusal that is ErrPlanNotApproved", r.Err)
	}
	msg := r.Err.Error()
	if !strings.HasPrefix(msg, "org: send: TASK not sent: ") || !strings.Contains(msg, want) ||
		!strings.Contains(msg, "; "+planNotApprovedMarker+": ") || !strings.HasSuffix(msg, "; "+planNotApprovedNever+planNotApprovedNeverRaw) {
		t.Fatalf("send = %q; want it to start with \"org: send: TASK not sent: \", contain %q and the marker, and end with what a leader never does", msg, want)
	}
	if n := strings.Count(msg, "org:"); n != 1 {
		t.Errorf("send = %q says \"org:\" %d times; want once", msg, n)
	}
	if errors.Is(r.Err, ErrNoRunningWatch) {
		t.Errorf("send = %v is the watch gate's refusal too; want the approval check's only", r.Err)
	}
	if r.Progress != SendProgressNothingSent || r.PaneID != "" {
		t.Errorf("send = %+v; want nothing sent", r)
	}
}

// sendVariants are the four ways the approval check must answer the same:
// typed, Raw (the same text, unvalidated), and either one in dry-run.
var sendVariants = []struct {
	name        string
	raw, dryRun bool
}{
	{"typed", false, false},
	{"raw", true, false},
	{"dry-run", false, true},
	{"raw dry-run", true, true},
}

// TestOrgSend_ApprovalCheck_Refusals covers AC5's refusals, AC9 and AC14 at
// the org layer: in an org bound in per-plan mode, a TASK to the implementer
// seat is refused when it names no plan or two, when no approval of the plan
// is recorded in the org's current life, and when the plan file does not
// pass (changed, not marked approved, another digest on its - Approved:
// line, or a change the digest does not show). Each refusal, typed, raw and
// dry-run alike, leaves the manifest and the receipts byte for byte as they
// were and calls neither herdr nor agmsg.
func TestOrgSend_ApprovalCheck_Refusals(t *testing.T) {
	recordAndMark := func(edit func(string) string) func(*testing.T, *enforceFixture) {
		return func(t *testing.T, fx *enforceFixture) {
			fx.record(t)
			fx.mark(t, edit)
		}
	}
	task := func(fx *enforceFixture) string { return planTask(fx.ref) }
	cases := []struct {
		name  string
		setup func(*testing.T, *enforceFixture)
		text  func(*enforceFixture) string
		want  string
	}{
		{"no PLAN header", recordAndMark(nil), func(*enforceFixture) string { return "TYPE: TASK\nTASK_ID: t-1\n\nimplement the plan" },
			"TASK not sent: no PLAN: header"},
		{"PLAN twice", recordAndMark(nil), func(fx *enforceFixture) string { return planTask(fx.ref, fx.ref) },
			"TASK not sent: 2 PLAN: headers"},
		{"no approval recorded", func(t *testing.T, fx *enforceFixture) { fx.mark(t, nil) }, task,
			`org_id "org-a" has no approval recorded for plan ` + testPlanRel},
		{"the plan changed after the approval", recordAndMark(replaceOnce("Add the login form.", "Add the signup form.")), task,
			"changed after its approval: approved digest"},
		{"Status not Approved", func(t *testing.T, fx *enforceFixture) { fx.record(t) }, task,
			`is not marked approved: its - Status: is "Draft"`},
		{"the Approved line names another digest", func(t *testing.T, fx *enforceFixture) {
			fx.record(t)
			fx.mark(t, replaceOnce("sha256:"+fx.ref.Digest, "sha256:ffffffffffff"))
		}, task, "its - Approved: line names sha256:ffffffffffff"},
		{"a Progress checklist line added after the approval (AC14)",
			recordAndMark(replaceOnce("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n")), task,
			"changed after its approval outside its - Status: and - Approved: lines and checkbox marks"},
		{"the Branch line changed after the approval (AC14)", recordAndMark(replaceOnce("- Branch: feat/auth", "- Branch: main")), task,
			"outside its - Status: and - Approved: lines"},
		{"an approval of the previous life (AC9)", func(t *testing.T, fx *enforceFixture) {
			fx.record(t)
			fx.mark(t, nil)
			// What Disband records for seats whose panes the fake never made:
			// each seat stopped, then the org disbanded.
			for _, seat := range []string{LeaderIdentity, implementerSeatID} {
				if err := fx.o.Manifest.Append(ManifestEvent{TS: fx.o.now(), OrgID: "org-a", SeatID: seat, Event: EventStopped}); err != nil {
					t.Fatal(err)
				}
			}
			if err := fx.o.Manifest.Append(ManifestEvent{TS: fx.o.now(), OrgID: "org-a", Event: EventDisbanded}); err != nil {
				t.Fatal(err)
			}
			spawnFeatureLeaderIn(t, fx.o, fx.h, "org-a", "ws-a2", "pane-4", fx.f, "internal/auth/")
			seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-5")
		}, task, "has no approval recorded for plan " + testPlanRel},
	}
	for _, tc := range cases {
		for _, v := range sendVariants {
			t.Run(tc.name+" "+v.name, func(t *testing.T) {
				fx := newEnforceFixture(t, config.ApprovalModePerPlan)
				seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
				tc.setup(t, fx)
				before := takeLedgerSnapshot(t, fx.o, fx.h, fx.a)
				r := fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: tc.text(fx), Raw: v.raw, DryRun: v.dryRun})
				assertTaskRefused(t, r, tc.want)
				if after := takeLedgerSnapshot(t, fx.o, fx.h, fx.a); after != before {
					t.Fatalf("a refused TASK changed the ledger or called a driver:\nbefore %+v\nafter  %+v", before, after)
				}
			})
		}
	}
}

// TestOrgSend_ApprovalCheck_PassesAndApprovedAgain covers AC5's passing
// half: with the approval recorded and the two header lines written, a TASK
// naming the plan is sent as before, typed, raw and in dry-run, and ticking
// a checkbox keeps it passing. Once the plan is revised, the old PLAN: is
// refused; after the revision is approved, a TASK naming the new digest is
// sent and one naming the old digest is still refused, with the new digest
// to name (self-review N1 of org-plan-approval).
func TestOrgSend_ApprovalCheck_PassesAndApprovedAgain(t *testing.T) {
	fx := newEnforceFixture(t, config.ApprovalModePerPlan)
	seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
	ref := approveFeaturePlan(t, fx.o, "org-a", fx.worktree)
	if ref != fx.ref {
		t.Fatalf("approveFeaturePlan = %s, want %s", ref, fx.ref)
	}
	send := func(ref PlanRef, raw, dryRun bool) SendResult {
		return fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: planTask(ref), Raw: raw, DryRun: dryRun})
	}
	for _, v := range sendVariants {
		r := send(fx.ref, v.raw, v.dryRun)
		if r.Err != nil {
			t.Fatalf("%s: send of an approved plan's TASK = %v; want it sent", v.name, r.Err)
		}
		if last := lastEvent(t, fx.o); last.Event != EventSent || last.DryRun != v.dryRun || last.SeatID != implementerSeatID {
			t.Fatalf("%s: last event = %+v; want sent to the implementer, dry_run %t", v.name, last, v.dryRun)
		}
		if !v.dryRun && r.Progress != SendProgressEnterPressed {
			t.Fatalf("%s: send = %+v; want Enter pressed", v.name, r)
		}
	}
	fx.mark(t, replaceOnce("- [ ] AC1", "- [x] AC1"))
	if r := send(fx.ref, false, false); r.Err != nil {
		t.Fatalf("send after a checkbox was ticked = %v; want it sent", r.Err)
	}

	revised := strings.Replace(testPlanDraft, "Add the login form.", "Add the signup form.", 1)
	writePlan(t, fx.worktree, testPlanRel, revised)
	assertTaskRefused(t, send(fx.ref, false, false), "changed after its approval: approved digest "+fx.ref.Digest)

	newRef := PlanRef{Path: testPlanRel, Digest: PlanDigest([]byte(revised))}
	if newRef.Digest == fx.ref.Digest {
		t.Fatal("the revision did not change the digest")
	}
	if err := fx.o.Manifest.Append(approvedEvent("org-a", "e2", newRef, PlanContentHash([]byte(revised)))); err != nil {
		t.Fatal(err)
	}
	writePlan(t, fx.worktree, testPlanRel, approvedPlanText(revised, newRef.Digest))
	if r := send(newRef, false, false); r.Err != nil {
		t.Fatalf("send naming the approved revision = %v; want it sent", r.Err)
	}
	stale := send(fx.ref, false, false)
	assertTaskRefused(t, stale, "TASK not sent: the TASK's PLAN: header names sha256:"+fx.ref.Digest+" of plan "+testPlanRel+
		` of org_id "org-a", and the plan's current approved digest is sha256:`+newRef.Digest+"; ")
	assertTaskRefused(t, stale, planNotApprovedMarker+": the leader changes the TASK's PLAN: header to PLAN: "+newRef.String()+
		" and sends the TASK again; ")
}

// planEditingHerdr is fakeHerdr whose first AgentWait, Send's wait for the
// seat to go idle, runs onWait before it returns: the seat at work edits the
// plan while Send waits for it.
type planEditingHerdr struct {
	*fakeHerdr
	onWait func()
}

func (h *planEditingHerdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	out, err := h.fakeHerdr.AgentWait(ctx, target, until, timeoutMS)
	if h.onWait != nil {
		onWait := h.onWait
		h.onWait = nil
		onWait()
	}
	return out, err
}

// TestOrgSend_ApprovalCheck_AgainAfterTheWait covers cross-review #1 of
// org-plan-approval: Send runs the approval check again after it waits for
// the seat to go idle, so a TASK whose plan changed while Send waited (in
// its digest, or only where the digest does not show) is refused, typed and
// raw alike, with nothing typed (no PaneSendText, no PaneSendKeys) and the
// manifest and the receipts byte for byte as they were. The controls: a wait
// during which the plan does not change, or only a checkbox is ticked, sends
// the TASK. TestOrgSend_ApprovalCheck_PassesAndApprovedAgain sends with the
// plain fake.
func TestOrgSend_ApprovalCheck_AgainAfterTheWait(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(string) string // the edit of the plan during the wait; nil: none
		refused string              // what the refusal contains; "": sent
	}{
		{"the plan changed while Send waited", replaceOnce("Add the login form.", "Add the signup form."),
			"changed after its approval: approved digest "},
		{"a Progress checklist line added while Send waited",
			replaceOnce("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n"),
			"changed after its approval outside its - Status: and - Approved: lines and checkbox marks"},
		{"nothing changed while Send waited", nil, ""},
		{"a checkbox ticked while Send waited", replaceOnce("- [ ] AC1", "- [x] AC1"), ""},
	}
	for _, tc := range cases {
		for _, raw := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s raw=%v", tc.name, raw), func(t *testing.T) {
				fx := newEnforceFixture(t, config.ApprovalModePerPlan)
				seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
				fx.record(t)
				fx.mark(t, nil)
				waited := false
				fx.o.Herdr = &planEditingHerdr{fakeHerdr: fx.h, onWait: func() {
					waited = true
					if tc.edit != nil {
						fx.mark(t, tc.edit)
					}
				}}
				manifest, receipts := fileOrAbsent(t, fx.o.Manifest.Path()), fileOrAbsent(t, fx.o.Receipts.Path())
				r := fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: planTask(fx.ref), Raw: raw})
				if !waited {
					t.Fatalf("send = %+v without waiting for the seat; want the wait first", r)
				}
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
				for _, call := range fx.h.calls {
					if call == "pane_send_text" || call == "pane_send_keys" {
						t.Fatalf("herdr calls %v; want nothing typed after the second approval check refused", fx.h.calls)
					}
				}
				if fileOrAbsent(t, fx.o.Manifest.Path()) != manifest || fileOrAbsent(t, fx.o.Receipts.Path()) != receipts {
					t.Fatal("a TASK refused after the wait changed the manifest or the receipts")
				}
			})
		}
	}
}

// TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest covers self-review N1
// of org-plan-approval through Send: a TASK to the implementer whose PLAN:
// names a digest the plan does not have now, while its current digest is
// approved, is refused with that current digest and the step to change the
// header to it, typed, raw and dry-run alike, leaving the ledger and the
// drivers as they were: (a) the digest of the first approval after the plan
// was approved again, (b) a mistyped digest. (c) A revision with no approval
// keeps the plan-changed refusal, and (d) a TASK with no PLAN: header keeps
// the step to put one on it. (e) After the plan was approved again, while
// its two header lines do not name the new digest yet (still Draft and TBD,
// or still the first approval's lines), a TASK naming the first approval's
// digest gets the step to write the lines naming the new digest; (e2) once
// they are written, the same TASK gets the step to change its PLAN: header to
// the new digest, and a TASK naming it is sent.
func TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest(t *testing.T) {
	revised := replaceOnce("Add the login form.", "Add the signup form.")(testPlanDraft)
	draftRef := PlanRef{Path: testPlanRel, Digest: PlanDigest([]byte(testPlanDraft))}
	revisedRef := PlanRef{Path: testPlanRel, Digest: PlanDigest([]byte(revised))}
	if draftRef.Digest == revisedRef.Digest || mistypedDigest == draftRef.Digest {
		t.Fatal("the revision or the mistyped digest is the draft's digest")
	}
	// approveRevisionAs records the approvals of the draft and the revision,
	// and leaves plan, the revision with some header lines, as the plan file.
	approveRevisionAs := func(plan string) func(*testing.T, *enforceFixture) {
		return func(t *testing.T, fx *enforceFixture) {
			t.Helper()
			if PlanDigest([]byte(plan)) != revisedRef.Digest {
				t.Fatal("the plan is not the approved revision")
			}
			fx.record(t)
			if err := fx.o.Manifest.Append(approvedEvent("org-a", "e2", revisedRef, PlanContentHash([]byte(revised)))); err != nil {
				t.Fatal(err)
			}
			writePlan(t, fx.worktree, testPlanRel, plan)
		}
	}
	approveRevision := approveRevisionAs(approvedPlanText(revised, revisedRef.Digest))
	recordAndMark := func(edit func(string) string) func(*testing.T, *enforceFixture) {
		return func(t *testing.T, fx *enforceFixture) {
			fx.record(t)
			fx.mark(t, edit)
		}
	}
	restated := func(named string, current PlanRef) []string {
		return []string{
			"TASK not sent: the TASK's PLAN: header names sha256:" + named + " of plan " + testPlanRel +
				` of org_id "org-a", and the plan's current approved digest is sha256:` + current.Digest + "; ",
			planNotApprovedMarker + ": the leader changes the TASK's PLAN: header to PLAN: " + current.String() + " and sends the TASK again; ",
		}
	}
	writeLines := func(approved PlanRef) string {
		return planNotApprovedMarker + ": the leader writes - Status: Approved and - Approved: <YYYY-MM-DD> sha256:" + approved.Digest +
			" before the plan's first ## heading, commits them, and tries again; "
	}
	notRestatedOrUndone := []string{"changes the TASK's PLAN: header", "changed after its approval", "undoes the change"}
	cases := []struct {
		name          string
		setup         func(*testing.T, *enforceFixture)
		text          string
		want, notWant []string
	}{
		{"(a) approved again, the PLAN digest of the first approval", approveRevision, planTask(draftRef),
			restated(draftRef.Digest, revisedRef), []string{"on the line right after TASK_ID:", "changed after its approval"}},
		{"(b) a mistyped PLAN digest", recordAndMark(nil), planTask(PlanRef{Path: testPlanRel, Digest: mistypedDigest}),
			restated(mistypedDigest, draftRef), []string{"on the line right after TASK_ID:", "has no approval recorded"}},
		{"(c) a revision with no approval", recordAndMark(replaceOnce("Add the login form.", "Add the signup form.")), planTask(draftRef),
			[]string{
				"TASK not sent: plan " + testPlanRel + ` of org_id "org-a" changed after its approval: approved digest ` + draftRef.Digest +
					", current digest " + revisedRef.Digest + "; ",
				planNotApprovedMarker + ": the leader undoes the change",
			}, []string{"changes the TASK's PLAN: header"}},
		{"(d) no PLAN header", approveRevision, "TYPE: TASK\nTASK_ID: t-1\n\nimplement the plan",
			[]string{"TASK not sent: no PLAN: header", "on the line right after TASK_ID: and sends the TASK again; "},
			[]string{"changes the TASK's PLAN: header"}},
		{"(e) approved again, the lines still Draft and TBD", approveRevisionAs(revised), planTask(draftRef),
			[]string{
				"TASK not sent: plan " + testPlanRel + ` of org_id "org-a" is not marked approved: its - Status: is "Draft", want Approved; `,
				writeLines(revisedRef),
			}, notRestatedOrUndone},
		{"(e) approved again, the first approval's lines still there", approveRevisionAs(approvedPlanText(revised, draftRef.Digest)),
			planTask(draftRef),
			[]string{
				"TASK not sent: plan " + testPlanRel + ` of org_id "org-a": its - Approved: line names sha256:` + draftRef.Digest +
					", not the approved sha256:" + revisedRef.Digest + "; ",
				writeLines(revisedRef),
			}, notRestatedOrUndone},
	}
	for _, tc := range cases {
		for _, v := range sendVariants {
			t.Run(tc.name+" "+v.name, func(t *testing.T) {
				fx := newEnforceFixture(t, config.ApprovalModePerPlan)
				seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
				tc.setup(t, fx)
				before := takeLedgerSnapshot(t, fx.o, fx.h, fx.a)
				r := fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: tc.text, Raw: v.raw, DryRun: v.dryRun})
				for _, want := range tc.want {
					assertTaskRefused(t, r, want)
				}
				for _, notWant := range tc.notWant {
					if strings.Contains(r.Err.Error(), notWant) {
						t.Errorf("send = %q; want no %q", r.Err, notWant)
					}
				}
				if after := takeLedgerSnapshot(t, fx.o, fx.h, fx.a); after != before {
					t.Fatalf("a refused TASK changed the ledger or called a driver:\nbefore %+v\nafter  %+v", before, after)
				}
			})
		}
	}

	for _, v := range sendVariants {
		t.Run("(e2) approved again, the lines written after the refusal "+v.name, func(t *testing.T) {
			fx := newEnforceFixture(t, config.ApprovalModePerPlan)
			seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
			approveRevisionAs(revised)(t, fx)
			send := func(ref PlanRef) SendResult {
				return fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: planTask(ref), Raw: v.raw, DryRun: v.dryRun})
			}
			before := takeLedgerSnapshot(t, fx.o, fx.h, fx.a)
			assertTaskRefused(t, send(draftRef), writeLines(revisedRef))
			writePlan(t, fx.worktree, testPlanRel, approvedPlanText(revised, revisedRef.Digest))
			stale := send(draftRef)
			for _, want := range restated(draftRef.Digest, revisedRef) {
				assertTaskRefused(t, stale, want)
			}
			if after := takeLedgerSnapshot(t, fx.o, fx.h, fx.a); after != before {
				t.Fatalf("a refused TASK changed the ledger or called a driver:\nbefore %+v\nafter  %+v", before, after)
			}
			if r := send(revisedRef); r.Err != nil {
				t.Fatalf("send naming the digest the refusal names = %v; want it sent", r.Err)
			}
			if last := lastEvent(t, fx.o); last.Event != EventSent || last.DryRun != v.dryRun || last.SeatID != implementerSeatID {
				t.Fatalf("last event = %+v; want sent to the implementer, dry_run %t", last, v.dryRun)
			}
		})
	}
}

// TestOrgSend_ApprovalCheck_NotApplied covers AC6: with no approval at all,
// a TASK to the reviewer of an org bound in per-plan mode, a QUESTION or a
// DECISION to its implementer, and a TASK to the implementer of an org bound
// in split mode, of an org an older start bound (no approval= token), and of
// an org with no binding are sent as before, typed and in dry-run.
func TestOrgSend_ApprovalCheck_NotApplied(t *testing.T) {
	perPlan := func(t *testing.T) *Org {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
		seedSeat(t, fx.o, "org-a", "reviewer", "reviewer", "pane-3")
		return fx.o
	}
	cases := []struct {
		name string
		org  func(t *testing.T) *Org
		to   string
		text string
	}{
		{"a TASK to the reviewer, per-plan", perPlan, "reviewer", "TYPE: TASK\nTASK_ID: t-1\n\nreview the plan"},
		{"a QUESTION to the implementer, per-plan", perPlan, implementerSeatID, "TYPE: QUESTION\n\nwhich file?"},
		{"a DECISION to the implementer, per-plan", perPlan, implementerSeatID, "TYPE: DECISION\n\nuse the token store"},
		{"a TASK to the implementer, split", func(t *testing.T) *Org {
			fx := newEnforceFixture(t, config.ApprovalModeSplit)
			seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
			return fx.o
		}, implementerSeatID, "TYPE: TASK\nTASK_ID: t-1\n\nimplement it"},
		{"a TASK to the implementer, a binding with no approval mode", func(t *testing.T) *Org {
			o, _, _ := testOrg(t)
			f := testFeature("split-1", "auth", "0123456789ab")
			f.Approval = ""
			ev := boundReserveEvent("org-a", f, "internal/auth/")
			if strings.Contains(ev.Details, "approval=") {
				t.Fatalf("expected a record with no approval= token, got %q", ev.Details)
			}
			if err := o.Manifest.Append(ev); err != nil {
				t.Fatal(err)
			}
			seedSeat(t, o, "org-a", LeaderIdentity, LeaderIdentity, "pane-1")
			seedSeat(t, o, "org-a", implementerSeatID, implementerRole, "pane-2")
			return o
		}, implementerSeatID, "TYPE: TASK\nTASK_ID: t-1\n\nimplement it"},
		{"a TASK to the implementer, no binding", func(t *testing.T) *Org {
			o, h, _ := testOrg(t)
			spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", "internal/auth/")
			seedSeat(t, o, "org-a", implementerSeatID, implementerRole, "pane-2")
			return o
		}, implementerSeatID, "TYPE: TASK\nTASK_ID: t-1\n\nimplement it"},
	}
	for _, tc := range cases {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s dry-run=%v", tc.name, dryRun), func(t *testing.T) {
				o := tc.org(t)
				r := o.Send(SendParams{OrgID: "org-a", To: tc.to, Text: tc.text, DryRun: dryRun})
				if r.Err != nil {
					t.Fatalf("send = %v; want it sent with no approval", r.Err)
				}
				if last := lastEvent(t, o); last.Event != EventSent || last.DryRun != dryRun || last.SeatID != tc.to {
					t.Fatalf("last event = %+v; want sent to %s, dry_run %t", last, tc.to, dryRun)
				}
			})
		}
	}
}

// TestOrgSend_ApprovalCheck_SeatMidSpawn: a seat whose latest state event is
// a spawn_step (in the middle of its spawn, or left there by a spawn that
// never finished, its agent started in a pane) has no role in the roster,
// since a spawn_step records none. The check takes the role its spawn_started
// recorded, so a TASK to such an implementer is refused and one to such a
// reviewer is sent.
func TestOrgSend_ApprovalCheck_SeatMidSpawn(t *testing.T) {
	for _, tc := range []struct {
		seatID, role string
		refused      bool
	}{
		{implementerSeatID, implementerRole, true},
		{"reviewer", "reviewer", false},
	} {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s dry-run=%v", tc.seatID, dryRun), func(t *testing.T) {
				fx := newEnforceFixture(t, config.ApprovalModePerPlan)
				for _, ev := range []ManifestEvent{
					{OrgID: "org-a", SeatID: tc.seatID, Event: EventSpawnStarted, Role: tc.role, Driver: "claude", Model: "sonnet"},
					{OrgID: "org-a", SeatID: tc.seatID, Event: EventSpawnStep, PaneID: "pane-2", Details: "agent_started"},
				} {
					if err := fx.o.Manifest.Append(ev); err != nil {
						t.Fatal(err)
					}
				}
				if seat, ok := seatFromEvents(mustReadEvents(t, fx.o), "org-a", tc.seatID); !ok || seat.Role != "" || !seat.Active {
					t.Fatalf("expected the roster to give the seat mid-spawn active with no role, got %+v (found=%v)", seat, ok)
				}
				before := takeLedgerSnapshot(t, fx.o, fx.h, fx.a)
				r := fx.o.Send(SendParams{OrgID: "org-a", To: tc.seatID, Text: "TYPE: TASK\nTASK_ID: t-1\n\ndo it", DryRun: dryRun})
				if !tc.refused {
					if r.Err != nil {
						t.Fatalf("send = %v; want it sent", r.Err)
					}
					return
				}
				assertTaskRefused(t, r, "no PLAN: header")
				if after := takeLedgerSnapshot(t, fx.o, fx.h, fx.a); after != before {
					t.Fatalf("a refused TASK changed the ledger or called a driver: %+v -> %+v", before, after)
				}
			})
		}
	}
}

// TestOrgSend_ApprovalCheck_UnknownSeat: in an org bound in per-plan mode the
// approval check looks the destination up before the dry-run branch, so a
// TASK to a seat the manifest does not have is refused with the real path's
// "seat not found" error in dry-run too, with nothing recorded; it is not an
// approval refusal. In an org with no binding a dry-run TASK still skips the
// lookup.
func TestOrgSend_ApprovalCheck_UnknownSeat(t *testing.T) {
	const want = `org: send: seat "ghost" not found in org_id "org-a"`
	for _, dryRun := range []bool{false, true} {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		before := takeLedgerSnapshot(t, fx.o, fx.h, fx.a)
		r := fx.o.Send(SendParams{OrgID: "org-a", To: "ghost", Text: planTask(fx.ref), DryRun: dryRun})
		if r.Err == nil || r.Err.Error() != want || errors.Is(r.Err, ErrPlanNotApproved) {
			t.Fatalf("dry-run=%v: send = %v; want exactly %q", dryRun, r.Err, want)
		}
		if after := takeLedgerSnapshot(t, fx.o, fx.h, fx.a); after != before {
			t.Fatalf("dry-run=%v: a TASK to an unknown seat changed the ledger or called a driver: %+v -> %+v", dryRun, before, after)
		}
	}
	o, h, _ := testOrg(t)
	spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1")
	if r := o.Send(SendParams{OrgID: "org-a", To: "ghost", Text: "TYPE: TASK\nTASK_ID: t-1\n\nx"}); r.Err == nil || r.Err.Error() != want {
		t.Fatalf("send to an unknown seat of an org with no binding = %v; want %q", r.Err, want)
	}
	if r := o.Send(SendParams{OrgID: "org-a", To: "ghost", Text: "TYPE: TASK\nTASK_ID: t-1\n\nx", DryRun: true}); r.Err != nil {
		t.Fatalf("dry-run send to an unknown seat of an org with no binding = %v; want it recorded as before", r.Err)
	}
}

// TestOrgSend_ApprovalCheck_ModeFromTheRecord covers AC13 at the org layer:
// the check reads the approval mode start recorded in the binding, never
// o.Config. An org recorded per-plan refuses an unapproved TASK while the
// config says split, one recorded split sends it while the config says
// per-plan, and one whose approval= token cannot be read (given twice)
// requires the approval.
func TestOrgSend_ApprovalCheck_ModeFromTheRecord(t *testing.T) {
	task := "TYPE: TASK\nTASK_ID: t-1\n\nimplement it"
	t.Run("recorded per-plan, config split", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		fx.o.Config.Approval.Mode = config.ApprovalModeSplit
		seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
		assertTaskRefused(t, fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: task}), "no PLAN: header")
	})
	t.Run("recorded split, config per-plan", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModeSplit)
		fx.o.Config.Approval.Mode = config.ApprovalModePerPlan
		seedSeat(t, fx.o, "org-a", implementerSeatID, implementerRole, "pane-2")
		if r := fx.o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: task}); r.Err != nil {
			t.Fatalf("send = %v; want it sent in an org recorded split", r.Err)
		}
	})
	t.Run("an approval= token given twice", func(t *testing.T) {
		o, _, _ := testOrg(t)
		o.Config.Approval.Mode = config.ApprovalModeSplit
		f := testFeature("split-1", "auth", "0123456789ab")
		f.Approval = config.ApprovalModeSplit
		ev := boundReserveEvent("org-a", f, "internal/auth/")
		ev.Details += " approval=split"
		if err := o.Manifest.Append(ev); err != nil {
			t.Fatal(err)
		}
		if got := ActiveFeature(mustReadEvents(t, o), "org-a"); got == nil || got.Approval != approvalModeUnreadable {
			t.Fatalf("ActiveFeature = %+v; want the approval mode unreadable", got)
		}
		seedSeat(t, o, "org-a", LeaderIdentity, LeaderIdentity, "pane-1")
		seedSeat(t, o, "org-a", implementerSeatID, implementerRole, "pane-2")
		assertTaskRefused(t, o.Send(SendParams{OrgID: "org-a", To: implementerSeatID, Text: task}), "no PLAN: header")
	})
}

// assertSpawnApprovalRefused checks that r is the approval check's refusal of
// a spawn, recorded through reject(), containing want.
func assertSpawnApprovalRefused(t *testing.T, o *Org, r SpawnResult, seatID, want string) {
	t.Helper()
	assertRejectedRecorded(t, o, r, "org-a", seatID, want)
	msg := r.Err.Error()
	if !errors.Is(r.Err, ErrPlanNotApproved) || !strings.Contains(msg, "; "+planNotApprovedMarker+": ") ||
		!strings.HasSuffix(msg, "; "+planNotApprovedNever) || strings.Contains(msg, "--raw") {
		t.Fatalf("spawn = %v; want ErrPlanNotApproved with the marker, ending with what a leader never does and no --raw", r.Err)
	}
}

// TestOrgSpawn_ApprovalCheck_Implementer covers AC7, AC9 and AC14 for a
// spawn, and AC13's org layer: in an org bound in per-plan mode, an
// implementer spawn with no approval that passes is refused with a
// `rejected` and a receipt, and its dry run predicts the same refusal as a
// dry-run `rejected`; with one it is spawned. A reviewer is spawned before
// any approval, and an implementer of an org bound in split mode needs none.
// An approval of the org's previous life, and an approved plan changed where
// its digest does not show, do not pass. The mode is the recorded one, never
// o.Config's.
func TestOrgSpawn_ApprovalCheck_Implementer(t *testing.T) {
	const noApproval = `no plan of org_id "org-a" has been approved since its last disband`
	t.Run("per-plan, no approval: rejected, and the dry run predicts it", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		dry := implementerParams("org-a")
		dry.DryRun = true
		dr := fx.o.Spawn(dry)
		if dr.Outcome != SpawnOutcomeRejected || !errors.Is(dr.Err, ErrPlanNotApproved) || !strings.Contains(dr.Err.Error(), noApproval) {
			t.Fatalf("dry-run spawn = %+v; want the approval refusal", dr)
		}
		if ev := lastEvent(t, fx.o); ev.Event != EventRejected || !ev.DryRun || ev.SeatID != implementerSeatID || ev.Details != dr.Err.Error() {
			t.Fatalf("expected a dry-run rejected for the implementer, got %+v", ev)
		}
		if dr.ModelReceipt.Honored != HonoredFalse || dr.ModelReceipt.Reason != dr.Err.Error() {
			t.Fatalf("expected the dry run's honored=false receipt, got %+v", dr.ModelReceipt)
		}
		r := fx.o.Spawn(implementerParams("org-a"))
		assertSpawnApprovalRefused(t, fx.o, r, implementerSeatID, noApproval)
		if r.Err.Error() != dr.Err.Error() {
			t.Fatalf("spawn = %q; want the dry run's %q", r.Err, dr.Err)
		}
		if len(fx.h.tabCreateCwds) != 1 {
			t.Fatalf("expected no herdr tab beyond the leader's, got %v", fx.h.tabCreateCwds)
		}
	})
	t.Run("per-plan, approved: spawned", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		approveFeaturePlan(t, fx.o, "org-a", fx.worktree)
		fx.h.paneID = "pane-2"
		if r := fx.o.Spawn(implementerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn = %+v; want spawned", r)
		}
	})
	t.Run("per-plan, a reviewer before any approval: spawned", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		fx.h.paneID = "pane-2"
		if r := fx.o.Spawn(reviewerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn = %+v; want spawned", r)
		}
	})
	t.Run("split, no approval: spawned", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModeSplit)
		fx.h.paneID = "pane-2"
		if r := fx.o.Spawn(implementerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn = %+v; want spawned", r)
		}
	})
	t.Run("an approval of the previous life (AC9)", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		approveFeaturePlan(t, fx.o, "org-a", fx.worktree)
		if d := fx.o.Disband(DisbandParams{OrgID: "org-a"}); len(d.Errs) != 0 || !d.Disbanded {
			t.Fatalf("disband org-a: %+v", d)
		}
		spawnFeatureLeaderIn(t, fx.o, fx.h, "org-a", "ws-a2", "pane-2", fx.f, "internal/auth/")
		fx.h.paneID = "pane-3"
		assertSpawnApprovalRefused(t, fx.o, fx.o.Spawn(implementerParams("org-a")), implementerSeatID, noApproval)
	})
	t.Run("a Progress checklist line added after the approval (AC14)", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		fx.record(t)
		fx.mark(t, replaceOnce("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n"))
		assertSpawnApprovalRefused(t, fx.o, fx.o.Spawn(implementerParams("org-a")), implementerSeatID,
			"none of the 1 plan approvals of org_id \"org-a\" since its last disband passes the check")
		fx.mark(t, replaceOnce("- [ ] Implementation started", "- [x] Implementation started"))
		fx.h.paneID = "pane-2"
		if r := fx.o.Spawn(implementerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn after only a checkbox was ticked = %+v; want spawned", r)
		}
	})
	t.Run("recorded per-plan, config split: rejected", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModePerPlan)
		fx.o.Config.Approval.Mode = config.ApprovalModeSplit
		assertSpawnApprovalRefused(t, fx.o, fx.o.Spawn(implementerParams("org-a")), implementerSeatID, noApproval)
	})
	t.Run("recorded split, config per-plan: spawned", func(t *testing.T) {
		fx := newEnforceFixture(t, config.ApprovalModeSplit)
		fx.o.Config.Approval.Mode = config.ApprovalModePerPlan
		fx.h.paneID = "pane-2"
		if r := fx.o.Spawn(implementerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn = %+v; want spawned", r)
		}
	})
}

// TestOrgSpawn_FeatureOrgNeedsItsLeader covers AC8: in an org bound in
// either mode, while its leader seat is not running, a spawn of any other
// seat (the implementer of an approved plan included, and a seat given the
// leader's seat id or role but not both) is refused with a `rejected` and a
// receipt, and its dry run predicts it; once start spawns the leader again,
// the seat is spawned. A seat of an org with no binding is spawned with no
// leader, as before.
func TestOrgSpawn_FeatureOrgNeedsItsLeader(t *testing.T) {
	const noLeader = `org_id "org-a" was started with ralph org start --plan and its leader seat is not running`
	for _, mode := range []string{config.ApprovalModePerPlan, config.ApprovalModeSplit} {
		t.Run(mode, func(t *testing.T) {
			fx := newEnforceFixture(t, mode)
			approveFeaturePlan(t, fx.o, "org-a", fx.worktree)
			if r := fx.o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
				t.Fatalf("stop the leader: %v", r.Err)
			}
			leaderID := mustSpawnParams("org-a", LeaderIdentity)
			leaderID.Role = implementerRole
			leaderRole := mustSpawnParams("org-a", "boss")
			leaderRole.Role = LeaderIdentity
			for _, p := range []SpawnParams{reviewerParams("org-a"), implementerParams("org-a"), leaderID, leaderRole} {
				dry := p
				dry.DryRun = true
				if dr := fx.o.Spawn(dry); dr.Outcome != SpawnOutcomeRejected || dr.Err == nil || !strings.Contains(dr.Err.Error(), noLeader) {
					t.Fatalf("dry-run spawn of %s (role %s) = %+v; want the leader refusal", p.SeatID, p.Role, dr)
				}
				r := fx.o.Spawn(p)
				assertRejectedRecorded(t, fx.o, r, "org-a", p.SeatID, noLeader, fmt.Sprintf("so seat_id %q is not spawned", p.SeatID),
					"start the org again with ralph org start --plan")
				if errors.Is(r.Err, ErrPlanNotApproved) {
					t.Fatalf("spawn of %s = %v is an approval refusal; want the leader refusal", p.SeatID, r.Err)
				}
			}
			spawnFeatureLeaderIn(t, fx.o, fx.h, "org-a", "ws-a", "pane-2", fx.f, "internal/auth/")
			fx.h.paneID = "pane-3"
			if r := fx.o.Spawn(implementerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn of the implementer with the leader running again = %+v; want spawned", r)
			}
		})
	}
	t.Run("no binding", func(t *testing.T) {
		o, h, _ := testOrg(t)
		spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-1")
		h.paneID = "pane-2"
		if r := o.Spawn(implementerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn of an implementer in an org with no binding and no leader = %+v; want spawned", r)
		}
	})
}

// TestOrgSpawn_ApprovalCheck_IdempotentRespawn covers A6: a spawn of a seat
// that is already active returns it before either check, so a respawn of a
// running implementer whose plan no longer passes, or whose leader has
// stopped, writes nothing (no `rejected` over the active seat) and calls no
// driver.
func TestOrgSpawn_ApprovalCheck_IdempotentRespawn(t *testing.T) {
	fx := newEnforceFixture(t, config.ApprovalModePerPlan)
	approveFeaturePlan(t, fx.o, "org-a", fx.worktree)
	fx.h.paneID = "pane-2"
	if r := fx.o.Spawn(implementerParams("org-a")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn = %+v; want spawned", r)
	}
	respawn := func(why string) {
		t.Helper()
		before := takeLedgerSnapshot(t, fx.o, fx.h, fx.a)
		r := fx.o.Spawn(implementerParams("org-a"))
		if r.Outcome != SpawnOutcomeIdempotent || r.Err != nil || r.Seat.SeatID != implementerSeatID {
			t.Fatalf("respawn %s = %+v; want the active implementer returned", why, r)
		}
		if after := takeLedgerSnapshot(t, fx.o, fx.h, fx.a); after != before {
			t.Fatalf("respawn %s recorded or called something: %+v -> %+v", why, before, after)
		}
		assertSeatUnchanged(t, fx.o, "org-a", implementerSeatID)
	}
	fx.mark(t, replaceOnce("Add the login form.", "Add the signup form."))
	respawn("after the plan changed")
	if r := fx.o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
		t.Fatalf("stop the leader: %v", r.Err)
	}
	respawn("after the leader stopped")
}
