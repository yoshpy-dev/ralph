package org

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Plan approval at the org layer (plan
// docs/plans/active/2026-10-12-org-plan-approval.md, AC2, AC3, AC4, AC9 and
// AC14's basis): the PLAN: header, the plan path in the feature worktree, the
// plan's header lines, the content hash, the plan_approved record, the check
// functions, and Approve.

// testPlanRel is the plan path the tests' leader writes in its worktree.
const testPlanRel = "docs/plans/active/2026-10-12-auth.md"

// testPlanDraft is a feature plan as the leader asks for its approval: the
// header lines not yet approved, a - Branch: line, and a Progress checklist.
const testPlanDraft = `# auth

- Status: Draft
- Approved: TBD
- Owner: leader
- Branch: feat/auth

## Objective

Add the login form.

## Acceptance criteria

- [ ] AC1: the form posts

## Progress checklist

- [ ] Plan reviewed
- [ ] Implementation started
`

// approvedPlanText is plan with its two header lines written the way the
// leader writes them after the approval of digest.
func approvedPlanText(plan, digest string) string {
	plan = strings.Replace(plan, "- Status: Draft", "- Status: Approved", 1)
	return strings.Replace(plan, "- Approved: TBD", "- Approved: 2026-10-12 sha256:"+digest, 1)
}

// approvalFixture is an Org whose manifest and inbox share one scratch state
// dir, with a fixed clock and an environment that Getenv reads from env
// (USER=alice, nothing else, unless a test sets more).
type approvalFixture struct {
	o        *Org
	stateDir string
	env      map[string]string
}

func newApprovalFixture(t *testing.T) *approvalFixture {
	t.Helper()
	stateDir := t.TempDir()
	clock := func() time.Time { return time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC) }
	inbox := NewInboxStore(stateDir)
	inbox.Now = clock
	fx := &approvalFixture{stateDir: stateDir, env: map[string]string{"USER": "alice"}}
	fx.o = &Org{
		Manifest: NewManifestStoreAtPath(ManifestPathIn(stateDir)),
		Inbox:    inbox,
		Now:      clock,
		Getenv:   func(key string) string { return fx.env[key] },
	}
	return fx
}

// appendEvents appends evs to the fixture's manifest.
func (fx *approvalFixture) appendEvents(t *testing.T, evs ...ManifestEvent) {
	t.Helper()
	for _, ev := range evs {
		if err := fx.o.Manifest.Append(ev); err != nil {
			t.Fatalf("append %+v: %v", ev, err)
		}
	}
}

// bind records orgID as started for a split plan feature in mode, with
// worktree as its feature worktree, and returns the binding.
func (fx *approvalFixture) bind(t *testing.T, orgID, worktree, mode string) *FeatureBinding {
	t.Helper()
	f := testFeature("split-1", orgID, "0123456789ab")
	f.Worktree, f.Approval = worktree, mode
	fx.appendEvents(t, boundReserveEvent(orgID, f, "internal/"+orgID+"/"))
	return f
}

// ask escalates text as orgID's inbox item and returns its ID.
func (fx *approvalFixture) ask(t *testing.T, orgID, typ, text string) string {
	t.Helper()
	id, err := fx.o.Inbox.Escalate(InboxEscalation{OrgID: orgID, Type: typ, Body: text})
	if err != nil {
		t.Fatalf("escalate: %v", err)
	}
	return id
}

// item returns inbox item id.
func (fx *approvalFixture) item(t *testing.T, id string) InboxItem {
	t.Helper()
	rr, err := fx.o.Inbox.Read()
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

// ledgerBytes returns the manifest and inbox files as they are.
func (fx *approvalFixture) ledgerBytes(t *testing.T) string {
	t.Helper()
	return fileOrAbsent(t, fx.o.Manifest.Path()) + "\x00" + fileOrAbsent(t, fx.o.Inbox.Path())
}

// planApprovedEvents returns the manifest's plan_approved events.
func planApprovedEvents(t *testing.T, o *Org) []ManifestEvent {
	t.Helper()
	var out []ManifestEvent
	for _, ev := range mustReadEvents(t, o) {
		if ev.Event == EventPlanApproved {
			out = append(out, ev)
		}
	}
	return out
}

// writePlan writes content as the plan at rel in worktree and returns it.
func writePlan(t *testing.T, worktree, rel, content string) []byte {
	t.Helper()
	path := filepath.Join(worktree, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return []byte(content)
}

// planQuestion is the QUESTION a leader escalates for ref.
func planQuestion(ref PlanRef) string {
	return "TYPE: QUESTION\nPLAN: " + ref.String() + "\nVISUAL: .harness/state/plan-visual/auth.html\n\nPlease approve the plan."
}

// approvedEvent is the plan_approved record Approve writes for ref with
// content, for org orgID and item inboxID.
func approvedEvent(orgID, inboxID string, ref PlanRef, content string) ManifestEvent {
	a := planApproval{Plan: ref.Path, Digest: ref.Digest, Content: content, InboxID: inboxID, By: "alice", Cwd: "/work"}
	return ManifestEvent{OrgID: orgID, Event: EventPlanApproved, Details: a.details()}
}

// approveFeaturePlan is a human's approval of testPlanDraft for orgID, whose
// feature worktree is worktree, as the check then finds it: the record
// Approve writes for the draft (approvedEvent, with its PlanContentHash), and
// the plan at testPlanRel with the two header lines the leader writes after
// it. It returns the plan's ref, for a TASK's PLAN: header. A test whose org
// is bound in per-plan mode calls it before it spawns an implementer or sends
// one a TASK.
func approveFeaturePlan(t *testing.T, o *Org, orgID, worktree string) PlanRef {
	t.Helper()
	draft := []byte(testPlanDraft)
	ref := PlanRef{Path: testPlanRel, Digest: PlanDigest(draft)}
	if err := o.Manifest.Append(approvedEvent(orgID, "e1", ref, PlanContentHash(draft))); err != nil {
		t.Fatalf("append the approval of %s: %v", ref, err)
	}
	writePlan(t, worktree, testPlanRel, approvedPlanText(testPlanDraft, ref.Digest))
	return ref
}

func TestParsePlanHeader(t *testing.T) {
	const digest = "0123456789ab"
	msg := func(planLines ...string) string {
		text := "TYPE: TASK\nTASK_ID: t-1\n"
		for _, l := range planLines {
			text += "PLAN: " + l + "\n"
		}
		return text + "\nimplement it"
	}
	cases := []struct {
		name    string
		text    string
		want    PlanRef
		wantErr string
	}{
		{"one header", msg("docs/a.md sha256:" + digest), PlanRef{"docs/a.md", digest}, ""},
		{"a path cleaned", msg("./docs//plans/./a.md sha256:" + digest), PlanRef{"docs/plans/a.md", digest}, ""},
		{"extra spaces around the words", "TYPE: QUESTION\nPLAN:   docs/a.md   sha256:" + digest + "  \n\nok?", PlanRef{"docs/a.md", digest}, ""},
		{"no PLAN header", msg(), PlanRef{}, "no PLAN: header"},
		{"a PLAN line in the body only", "TYPE: TASK\nTASK_ID: t-1\n\nPLAN: docs/a.md sha256:" + digest, PlanRef{}, "no PLAN: header"},
		{"two headers", msg("docs/a.md sha256:"+digest, "docs/b.md sha256:"+digest), PlanRef{}, "2 PLAN: headers"},
		{"the same header twice", msg("docs/a.md sha256:"+digest, "docs/a.md sha256:"+digest), PlanRef{}, "2 PLAN: headers"},
		{"one word", msg("docs/a.md"), PlanRef{}, "not the two words"},
		{"three words", msg("docs/my plan.md sha256:" + digest), PlanRef{}, "not the two words"},
		{"an empty value", msg(""), PlanRef{}, "not the two words"},
		{"no sha256: prefix", msg("docs/a.md " + digest), PlanRef{}, "is not sha256: and the 12 lowercase hex digits"},
		{"another hash name", msg("docs/a.md md5:" + digest), PlanRef{}, "is not sha256:"},
		{"upper-case hex", msg("docs/a.md sha256:0123456789AB"), PlanRef{}, "is not sha256:"},
		{"11 hex digits", msg("docs/a.md sha256:0123456789a"), PlanRef{}, "is not sha256:"},
		{"13 hex digits", msg("docs/a.md sha256:0123456789abc"), PlanRef{}, "is not sha256:"},
		{"an absolute path", msg("/repo/docs/a.md sha256:" + digest), PlanRef{}, "is absolute"},
		{"a leading ..", msg("../a.md sha256:" + digest), PlanRef{}, "has a .. element"},
		{"a .. that stays inside", msg("./a/../b.md sha256:" + digest), PlanRef{}, "has a .. element"},
		{"a trailing ..", msg("docs/.. sha256:" + digest), PlanRef{}, "has a .. element"},
		{"the worktree itself", msg(". sha256:" + digest), PlanRef{}, "names the feature worktree itself"},
		{"the worktree itself with slashes", msg(".//./ sha256:" + digest), PlanRef{}, "names the feature worktree itself"},
		{"a control character", msg("docs/\x1b[2Ja.md sha256:" + digest), PlanRef{}, "has a control character"},
		{"no TYPE first line", "PLAN: docs/a.md sha256:" + digest + "\nTYPE: TASK", PlanRef{}, "no PLAN: header"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePlanHeader(tc.text)
			if tc.wantErr == "" {
				if err != nil || got != tc.want {
					t.Fatalf("ParsePlanHeader = %+v, %v; want %+v", got, err, tc.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.HasPrefix(err.Error(), "org: ") {
				t.Fatalf("ParsePlanHeader = %+v, %v; want an org: error containing %q", got, err, tc.wantErr)
			}
			if got != (PlanRef{}) {
				t.Fatalf("expected the zero PlanRef with the error, got %+v", got)
			}
		})
	}
}

func TestResolvePlanFile(t *testing.T) {
	worktree, outside := t.TempDir(), t.TempDir()
	writePlan(t, worktree, "docs/a.md", "# a\n")
	writePlan(t, outside, "secret.md", "# secret\n")
	for _, link := range []struct{ name, target string }{
		{"docs/inside.md", "a.md"},
		{"docs/out.md", filepath.Join(outside, "secret.md")},
		{"ext", outside},
		{"docs/dangling.md", "missing.md"},
	} {
		if err := os.Symlink(link.target, filepath.Join(worktree, filepath.FromSlash(link.name))); err != nil {
			t.Fatal(err)
		}
	}
	linkedWorktree := filepath.Join(t.TempDir(), "wt")
	if err := os.Symlink(worktree, linkedWorktree); err != nil {
		t.Fatal(err)
	}
	realWorktree, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		t.Fatal(err)
	}
	wantA := filepath.Join(realWorktree, "docs", "a.md")
	cases := []struct {
		name     string
		worktree string
		rel      string
		want     string
		wantErr  string
	}{
		{"a relative path", worktree, "docs/a.md", wantA, ""},
		{"a symlink inside the worktree", worktree, "docs/inside.md", wantA, ""},
		{"the worktree given through a symlink", linkedWorktree, "docs/a.md", wantA, ""},
		{"a symlink to a file outside", worktree, "docs/out.md", "", "outside the feature worktree"},
		{"a file under a symlinked directory outside", worktree, "ext/secret.md", "", "outside the feature worktree"},
		{"a missing file", worktree, "docs/missing.md", "", "no such file or directory"},
		{"a dangling symlink", worktree, "docs/dangling.md", "", "no such file or directory"},
		{"a directory", worktree, "docs", "", "is not a regular file"},
		{"a missing worktree", filepath.Join(worktree, "gone"), "docs/a.md", "", "cannot be read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePlanFile(tc.worktree, tc.rel)
			if tc.wantErr == "" {
				if err != nil || got != tc.want {
					t.Fatalf("resolvePlanFile = %q, %v; want %q", got, err, tc.want)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), tc.worktree) {
				t.Fatalf("resolvePlanFile = %q, %v; want an error containing %q that names the worktree %s", got, err, tc.wantErr, tc.worktree)
			}
		})
	}
}

func TestReadFeaturePlanHeader(t *testing.T) {
	cases := []struct {
		name string
		plan string
		want featurePlanHeader
	}{
		{"both before the first ## heading", "# t\n\n- Status: Approved \n- Approved: 2026-10-12 sha256:0123456789ab\n\n## Objective\n",
			featurePlanHeader{hasStatus: true, status: "Approved", hasApproved: true, approvedDigest: "0123456789ab"}},
		{"the first of each is read", "# t\n- Status: Draft\n- Status: Approved\n- Approved: TBD\n- Approved: x sha256:0123456789ab\n## O\n",
			featurePlanHeader{hasStatus: true, status: "Draft", hasApproved: true}},
		{"no sha256 value on the Approved line", "- Status: Approved\n- Approved: 2026-10-12 sha256:0123456789AB\n",
			featurePlanHeader{hasStatus: true, status: "Approved", hasApproved: true}},
		{"lines after the first ## heading are not read", "# t\n\n## Objective\n\n- Status: Approved\n- Approved: x sha256:0123456789ab\n",
			featurePlanHeader{}},
		{"a ### heading does not end the header", "# t\n### note\n- Status: Approved\n", featurePlanHeader{hasStatus: true, status: "Approved"}},
		{"an indented line is not the header line", "# t\n  - Status: Approved\n", featurePlanHeader{}},
		{"CRLF lines are trimmed", "- Status: Approved\r\n- Approved: x sha256:0123456789ab\r\n",
			featurePlanHeader{hasStatus: true, status: "Approved", hasApproved: true, approvedDigest: "0123456789ab"}},
		{"neither line", "# t\n\n## Objective\n", featurePlanHeader{}},
		{"an empty file", "", featurePlanHeader{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := readFeaturePlanHeader([]byte(tc.plan)); got != tc.want {
				t.Fatalf("readFeaturePlanHeader = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestPlanContentHash covers A9 and AC14's basis: the hash leaves out only
// the first - Status: and - Approved: lines before the first ## heading and
// the checkbox marks, so writing those lines or ticking a box keeps it, while
// a line the digest skips anywhere else changes it with the digest left as
// it was.
func TestPlanContentHash(t *testing.T) {
	base := testPlanDraft
	hash, digest := PlanContentHash([]byte(base)), PlanDigest([]byte(base))
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(hash) {
		t.Fatalf("PlanContentHash = %q, want 64 lowercase hex digits", hash)
	}
	if got := PlanContentHash([]byte(strings.TrimSuffix(base, "\n"))); got != hash {
		t.Fatalf("a missing final newline changed the hash: %s vs %s", got, hash)
	}
	if got := PlanContentHash(nil); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("PlanContentHash(empty) = %s, want the SHA-256 of no bytes", got)
	}
	replace := func(from, to string) string {
		if !strings.Contains(base, from) {
			t.Fatalf("the base plan has no %q", from)
		}
		return strings.Replace(base, from, to, 1)
	}
	keep := []struct{ name, plan string }{
		{"the two header lines written after the approval", approvedPlanText(base, digest)},
		{"the Status line removed", replace("- Status: Draft\n", "")},
		{"a box ticked with x", replace("- [ ] AC1", "- [x] AC1")},
		{"a box ticked with X in the checklist", replace("- [ ] Plan reviewed", "- [X] Plan reviewed")},
	}
	for _, tc := range keep {
		t.Run("keeps: "+tc.name, func(t *testing.T) {
			if got := PlanContentHash([]byte(tc.plan)); got != hash {
				t.Fatalf("PlanContentHash changed: %s, want %s", got, hash)
			}
		})
	}
	change := []struct{ name, plan string }{
		{"a line added under the Progress checklist", replace("- [ ] Implementation started\n", "- [ ] Implementation started\n- also delete the tests\n")},
		{"the Branch line changed", replace("- Branch: feat/auth", "- Branch: feat/other")},
		{"a Status line added after the first ## heading", replace("Add the login form.\n", "Add the login form.\n- Status: skip the review\n")},
		{"a second Approved line in the header", replace("- Owner: leader\n", "- Approved: also push to main\n- Owner: leader\n")},
	}
	for _, tc := range change {
		t.Run("changes, digest kept: "+tc.name, func(t *testing.T) {
			if got := PlanDigest([]byte(tc.plan)); got != digest {
				t.Fatalf("PlanDigest changed (%s, want %s): this case must be one the digest does not show", got, digest)
			}
			if got := PlanContentHash([]byte(tc.plan)); got == hash {
				t.Fatalf("PlanContentHash kept %s for a change the digest skips", hash)
			}
		})
	}
	t.Run("changes with the digest: the objective edited", func(t *testing.T) {
		plan := replace("Add the login form.", "Add the signup form.")
		if PlanDigest([]byte(plan)) == digest || PlanContentHash([]byte(plan)) == hash {
			t.Fatal("expected both the digest and the hash to change")
		}
	})
}

// TestPlanApprovedDetails_RoundTrip covers the record's Details: the keys in
// order, cwd quoted last (spaces and quotes kept), by=unknown for a USER that
// is empty or not one token, and the records that read as nothing.
func TestPlanApprovedDetails_RoundTrip(t *testing.T) {
	content := strings.Repeat("ab", 32)
	a := planApproval{Plan: "docs/a.md", Digest: "0123456789ab", Content: content, InboxID: "e3",
		By: approverName("alice"), HerdrPane: true, Cwd: `/work dir/"q" cwd=x`}
	want := "plan=docs/a.md digest=0123456789ab content=" + content + ` inbox=e3 by=alice herdr_pane=set cwd="/work dir/\"q\" cwd=x"`
	if got := a.details(); got != want {
		t.Fatalf("details = %q, want %q", got, want)
	}
	if got, ok := planApprovalFromDetails(a.details()); !ok || got != a {
		t.Fatalf("read back %+v (ok=%v), want %+v", got, ok, a)
	}
	for _, user := range []string{"", "a b", "x\x1by"} {
		if got := approverName(user); got != approverUnknown {
			t.Fatalf("approverName(%q) = %q, want %q", user, got, approverUnknown)
		}
	}
	good := "plan=docs/a.md digest=0123456789ab content=" + content + " inbox=e3"
	if _, ok := planApprovalFromDetails(good + " future=1"); !ok {
		t.Fatal("expected a record with only the four checked keys and an unknown one to read")
	}
	for _, bad := range []string{
		"",
		"plan=docs/a.md digest=0123456789ab content=" + content,
		"plan=docs/a.md digest=0123456789ab inbox=e3",
		strings.Replace(good, "docs/a.md", "../a.md", 1),
		strings.Replace(good, "docs/a.md", "./docs/a.md", 1),
		strings.Replace(good, "digest=0123456789ab", "digest=0123456789AB", 1),
		strings.Replace(good, "content="+content, "content="+content[:63], 1),
		good + " plan=docs/b.md",
		good + " stray",
	} {
		if got, ok := planApprovalFromDetails(bad); ok {
			t.Fatalf("expected %q not to read, got %+v", bad, got)
		}
	}
}

// checkFixture is a fixture with org-a bound in per-plan mode to a worktree
// holding the approved plan, and the record of its approval.
type checkFixture struct {
	*approvalFixture
	worktree string
	f        *FeatureBinding
	ref      PlanRef
	approved []byte // the plan's bytes when approved, before the leader writes the two lines
}

func newCheckFixture(t *testing.T) *checkFixture {
	t.Helper()
	fx := &checkFixture{approvalFixture: newApprovalFixture(t), worktree: t.TempDir()}
	fx.f = fx.bind(t, "org-a", fx.worktree, config.ApprovalModePerPlan)
	fx.approved = writePlan(t, fx.worktree, testPlanRel, testPlanDraft)
	fx.ref = PlanRef{Path: testPlanRel, Digest: PlanDigest(fx.approved)}
	return fx
}

// record appends the approval of fx.ref as Approve would have.
func (fx *checkFixture) record(t *testing.T) {
	t.Helper()
	fx.appendEvents(t, approvedEvent("org-a", "e1", fx.ref, PlanContentHash(fx.approved)))
}

// markApproved writes the plan with the two header lines, plus edit.
func (fx *checkFixture) markApproved(t *testing.T, edit func(string) string) {
	t.Helper()
	plan := approvedPlanText(testPlanDraft, fx.ref.Digest)
	if edit != nil {
		plan = edit(plan)
	}
	writePlan(t, fx.worktree, testPlanRel, plan)
}

func TestCheckPlanApproved(t *testing.T) {
	replace := func(from, to string) func(string) string {
		return func(s string) string {
			if !strings.Contains(s, from) {
				panic("the plan has no " + strconv.Quote(from))
			}
			return strings.Replace(s, from, to, 1)
		}
	}
	cases := []struct {
		name    string
		setup   func(t *testing.T, fx *checkFixture)
		ref     func(fx *checkFixture) PlanRef
		wantErr string // "" passes
	}{
		{name: "approved, both lines written", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, nil)
		}},
		{name: "boxes ticked after the approval", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("- [ ] AC1", "- [x] AC1"))
		}},
		{name: "the PLAN path given uncleaned is the same plan", ref: func(fx *checkFixture) PlanRef {
			ref, err := ParsePlanHeader("TYPE: TASK\nTASK_ID: t\nPLAN: ./docs//plans/active/2026-10-12-auth.md sha256:" + fx.ref.Digest)
			if err != nil {
				panic(err)
			}
			return ref
		}, setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, nil)
		}},
		{name: "several records, only an older one's content matches", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.appendEvents(t, approvedEvent("org-a", "e2", fx.ref, strings.Repeat("0", 64)))
			fx.markApproved(t, nil)
		}},
		{name: "no record", setup: func(t *testing.T, fx *checkFixture) {
			fx.markApproved(t, nil)
		}, wantErr: `org_id "org-a" has no approval recorded for plan ` + testPlanRel + " sha256:"},
		{name: "only a record of another digest", setup: func(t *testing.T, fx *checkFixture) {
			fx.appendEvents(t, approvedEvent("org-a", "e1", PlanRef{testPlanRel, "ffffffffffff"}, PlanContentHash(fx.approved)))
			fx.markApproved(t, nil)
		}, wantErr: "its latest approval is of sha256:ffffffffffff"},
		{name: "only a record of another path", setup: func(t *testing.T, fx *checkFixture) {
			fx.appendEvents(t, approvedEvent("org-a", "e1", PlanRef{"docs/other.md", fx.ref.Digest}, PlanContentHash(fx.approved)))
			fx.markApproved(t, nil)
		}, wantErr: "has no approval recorded for plan"},
		{name: "a record of another org", setup: func(t *testing.T, fx *checkFixture) {
			fx.appendEvents(t, approvedEvent("org-b", "e1", fx.ref, PlanContentHash(fx.approved)))
			fx.markApproved(t, nil)
		}, wantErr: "has no approval recorded for plan"},
		{name: "a dry-run record", setup: func(t *testing.T, fx *checkFixture) {
			ev := approvedEvent("org-a", "e1", fx.ref, PlanContentHash(fx.approved))
			ev.DryRun = true
			fx.appendEvents(t, ev)
			fx.markApproved(t, nil)
		}, wantErr: "has no approval recorded for plan"},
		{name: "a record a seat wrote", setup: func(t *testing.T, fx *checkFixture) {
			ev := approvedEvent("org-a", "e1", fx.ref, PlanContentHash(fx.approved))
			ev.SeatID = LeaderIdentity
			fx.appendEvents(t, ev)
			fx.markApproved(t, nil)
		}, wantErr: "has no approval recorded for plan"},
		{name: "a record whose details do not read", setup: func(t *testing.T, fx *checkFixture) {
			fx.appendEvents(t, ManifestEvent{OrgID: "org-a", Event: EventPlanApproved,
				Details: "plan=" + testPlanRel + " digest=" + fx.ref.Digest + " inbox=e1"})
			fx.markApproved(t, nil)
		}, wantErr: "has no approval recorded for plan"},
		{name: "a record from the previous life (AC9)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.appendEvents(t, ManifestEvent{OrgID: "org-a", Event: EventDisbanded}, boundReserveEvent("org-a", fx.f, "internal/org-a/"))
			fx.markApproved(t, nil)
		}, wantErr: "has no approval recorded for plan"},
		{name: "a dry-run disband does not end the life", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.appendEvents(t, ManifestEvent{OrgID: "org-a", Event: EventDisbanded, DryRun: true})
			fx.markApproved(t, nil)
		}},
		{name: "the plan changed after the approval", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("Add the login form.", "Add the signup form."))
		}, wantErr: "changed after its approval: approved digest "},
		{name: "a line added under the Progress checklist (AC14)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n"))
		}, wantErr: "changed after its approval outside its - Status: and - Approved: lines and checkbox marks"},
		{name: "the Branch line changed (AC14)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("- Branch: feat/auth", "- Branch: main"))
		}, wantErr: "outside its - Status: and - Approved: lines"},
		{name: "a Status line added after the first ## heading (AC14)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("Add the login form.\n", "Add the login form.\n- Status: done\n"))
		}, wantErr: "outside its - Status: and - Approved: lines"},
		{name: "Status not written yet", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
		}, wantErr: `is not marked approved: its - Status: is "Draft", want Approved`},
		{name: "no Status line", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("- Status: Approved\n", ""))
		}, wantErr: "has no - Status: line before its first ## heading"},
		{name: "the Approved line names another digest", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("sha256:"+fx.ref.Digest, "sha256:ffffffffffff"))
		}, wantErr: "its - Approved: line names sha256:ffffffffffff, not the approved sha256:"},
		{name: "the Approved line has no digest", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("2026-10-12 sha256:"+fx.ref.Digest, "TBD"))
		}, wantErr: "its - Approved: line names no sha256:<12 hex digits>"},
		{name: "no Approved line", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replace("- Approved: 2026-10-12 sha256:"+fx.ref.Digest+"\n", ""))
		}, wantErr: "has no - Approved: line before its first ## heading"},
		{name: "the plan file removed", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			if err := os.Remove(filepath.Join(fx.worktree, filepath.FromSlash(testPlanRel))); err != nil {
				t.Fatal(err)
			}
		}, wantErr: "no such file or directory"},
		{name: "the plan replaced by a symlink outside the worktree", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			outside := t.TempDir()
			writePlan(t, outside, "plan.md", approvedPlanText(testPlanDraft, fx.ref.Digest))
			path := filepath.Join(fx.worktree, filepath.FromSlash(testPlanRel))
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(outside, "plan.md"), path); err != nil {
				t.Fatal(err)
			}
		}, wantErr: "outside the feature worktree"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newCheckFixture(t)
			tc.setup(t, fx)
			ref := fx.ref
			if tc.ref != nil {
				ref = tc.ref(fx)
			}
			err := checkPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f, ref)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("checkPlanApproved: %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.HasPrefix(err.Error(), "org: ") {
				t.Fatalf("checkPlanApproved = %v, want an org: error containing %q", err, tc.wantErr)
			}
		})
	}
	t.Run("no complete binding", func(t *testing.T) {
		fx := newCheckFixture(t)
		fx.record(t)
		fx.markApproved(t, nil)
		incomplete := *fx.f
		incomplete.Worktree = ""
		for _, f := range []*FeatureBinding{nil, &incomplete} {
			if err := checkPlanApproved(mustReadEvents(t, fx.o), "org-a", f, fx.ref); err == nil ||
				!strings.Contains(err.Error(), "no complete split plan feature binding") {
				t.Fatalf("binding %+v: checkPlanApproved = %v, want the binding refusal", f, err)
			}
		}
	})
}

func TestCheckAnyPlanApproved(t *testing.T) {
	t.Run("no approval", func(t *testing.T) {
		fx := newCheckFixture(t)
		fx.markApproved(t, nil)
		err := checkAnyPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f)
		if err == nil || !strings.Contains(err.Error(), `no plan of org_id "org-a" has been approved since its last disband`) {
			t.Fatalf("checkAnyPlanApproved = %v, want the no-approval refusal", err)
		}
	})
	t.Run("an approval from the previous life only (AC9)", func(t *testing.T) {
		fx := newCheckFixture(t)
		fx.record(t)
		fx.appendEvents(t, ManifestEvent{OrgID: "org-a", Event: EventDisbanded}, boundReserveEvent("org-a", fx.f, "internal/org-a/"))
		fx.markApproved(t, nil)
		err := checkAnyPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f)
		if err == nil || !strings.Contains(err.Error(), "has been approved since its last disband") {
			t.Fatalf("checkAnyPlanApproved = %v, want the no-approval refusal", err)
		}
	})
	t.Run("one approval passes", func(t *testing.T) {
		fx := newCheckFixture(t)
		fx.record(t)
		fx.markApproved(t, nil)
		if err := checkAnyPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f); err != nil {
			t.Fatalf("checkAnyPlanApproved: %v", err)
		}
	})
	t.Run("an older approval of another plan passes when the latest does not", func(t *testing.T) {
		fx := newCheckFixture(t)
		fx.record(t)
		fx.markApproved(t, nil)
		fx.appendEvents(t, approvedEvent("org-a", "e2", PlanRef{"docs/next.md", "ffffffffffff"}, strings.Repeat("0", 64)))
		if err := checkAnyPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f); err != nil {
			t.Fatalf("checkAnyPlanApproved: %v", err)
		}
	})
	t.Run("none passes: the latest approval's reason", func(t *testing.T) {
		fx := newCheckFixture(t)
		fx.record(t)
		writePlan(t, fx.worktree, "docs/next.md", testPlanDraft)
		fx.appendEvents(t, approvedEvent("org-a", "e2", PlanRef{"docs/next.md", PlanDigest([]byte(testPlanDraft))}, PlanContentHash([]byte(testPlanDraft))))
		// Neither plan has the two header lines written.
		err := checkAnyPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f)
		if err == nil || !strings.Contains(err.Error(), `none of the 2 plan approvals of org_id "org-a" since its last disband passes the check; the latest: plan docs/next.md`) ||
			!strings.Contains(err.Error(), "is not marked approved") || strings.Contains(err.Error(), "org: plan") {
			t.Fatalf("checkAnyPlanApproved = %v, want the latest approval's reason", err)
		}
	})
}

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

// TestPlanApprovedEvent_IsNotASeat: a manifest with a plan_approved line
// reads as before for the roster, status, report, the running orgs, the
// leader activity count of the watch, and disband --all's targets.
func TestPlanApprovedEvent_IsNotASeat(t *testing.T) {
	approved := approvedEvent("org-a", "e1", PlanRef{testPlanRel, "0123456789ab"}, strings.Repeat("0", 64))
	approved.TS = "2026-10-12T12:00:00Z"
	leader := ManifestEvent{TS: "2026-10-12T11:00:00Z", OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned, Role: "leader", PaneID: "p-1"}
	withLine := []ManifestEvent{leader, approved}

	roster := Roster(withLine, RosterOptions{IncludeDryRun: true})
	if len(roster) != 1 || roster[0].SeatID != LeaderIdentity || !roster[0].Active || roster[0].Event != EventSpawned {
		t.Fatalf("Roster = %+v, want only the active leader", roster)
	}
	if got := leaderActivityEventCount(withLine, "org-a"); got != leaderActivityEventCount([]ManifestEvent{leader}, "org-a") {
		t.Fatalf("leaderActivityEventCount counts plan_approved: %d", got)
	}
	if got := RunningOrgs([]ManifestEvent{approved}); len(got) != 0 {
		t.Fatalf("RunningOrgs = %q, want a plan_approved alone not to make an org run", got)
	}
	if got := orgsToDisband([]ManifestEvent{approved}); len(got) != 0 {
		t.Fatalf("orgsToDisband = %q, want a plan_approved alone not to make a target", got)
	}

	report := BuildOrgReport(withLine, nil, "org-a", "2026-10-12", 0)
	rosterSection := report[strings.Index(report, "## Roster"):strings.Index(report, "## Event timeline")]
	if rows := strings.Count(rosterSection, "\n| ") - 1; rows != 1 || !strings.Contains(rosterSection, "| "+LeaderIdentity+" |") {
		t.Fatalf("report roster has %d seat rows, want only the leader:\n%s", rows, rosterSection)
	}
	if !strings.Contains(report, "| 2026-10-12T12:00:00Z | - | plan_approved | "+approved.Details+" |") {
		t.Fatalf("report timeline lacks the plan_approved row:\n%s", report)
	}
	if !strings.Contains(report, "- active seats: 1\n") {
		t.Fatalf("report counts a phantom seat:\n%s", report)
	}

	fx := newApprovalFixture(t)
	fx.appendEvents(t, withLine...)
	st, err := fx.o.Status("org-a", true)
	if err != nil || len(st.Seats) != 1 || st.Seats[0].SeatID != LeaderIdentity || st.CorruptLines != 0 {
		t.Fatalf("Status = %+v, %v; want only the leader", st, err)
	}
}
