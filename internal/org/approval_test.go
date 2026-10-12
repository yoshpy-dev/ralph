package org

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Plan approval at the org layer (plan
// docs/plans/active/2026-10-12-org-plan-approval.md, AC9 and AC14's basis):
// the PLAN: header, the plan path in the feature worktree, the plan's header
// lines, the content hash, the plan_approved record, and the check functions.
// Approve's tests are in approve_test.go.

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
			featurePlanHeader{hasStatus: true, status: "Approved", hasApproved: true, approved: "2026-10-12 sha256:0123456789ab", approvedDigest: "0123456789ab"}},
		{"the first of each is read", "# t\n- Status: Draft\n- Status: Approved\n- Approved: TBD\n- Approved: 2026-10-12 sha256:0123456789ab\n## O\n",
			featurePlanHeader{hasStatus: true, status: "Draft", hasApproved: true, approved: "TBD"}},
		{"no sha256 value on the Approved line", "- Status: Approved\n- Approved: 2026-10-12 sha256:0123456789AB\n",
			featurePlanHeader{hasStatus: true, status: "Approved", hasApproved: true, approved: "2026-10-12 sha256:0123456789AB"}},
		{"words after the digest on the Approved line (L2)", "- Status: Approved\n- Approved: 2026-10-12 sha256:0123456789ab and also X\n",
			featurePlanHeader{hasStatus: true, status: "Approved", hasApproved: true, approved: "2026-10-12 sha256:0123456789ab and also X"}},
		{"lines after the first ## heading are not read", "# t\n\n## Objective\n\n- Status: Approved\n- Approved: 2026-10-12 sha256:0123456789ab\n",
			featurePlanHeader{}},
		{"a ### heading does not end the header", "# t\n### note\n- Status: Approved\n", featurePlanHeader{hasStatus: true, status: "Approved"}},
		{"an indented line is not the header line", "# t\n  - Status: Approved\n", featurePlanHeader{}},
		{"CRLF lines are trimmed", "- Status: Approved\r\n- Approved: 2026-10-12 sha256:0123456789ab\r\n",
			featurePlanHeader{hasStatus: true, status: "Approved", hasApproved: true, approved: "2026-10-12 sha256:0123456789ab", approvedDigest: "0123456789ab"}},
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

// TestApprovedLineDigest covers self-review L2 of org-plan-approval: an
// `- Approved:` value names its digest only when it is exactly the two
// fields the leader writes, a valid date and sha256:<12 lowercase hex>, so
// nothing can be added to the one line both hashes leave out.
func TestApprovedLineDigest(t *testing.T) {
	const digest = "0123456789ab"
	for _, tc := range []struct{ name, value, want string }{
		{"the shape the leader writes", "2026-10-12 sha256:" + digest, digest},
		{"spaces and tabs around and between the fields", "  2026-10-12 \t sha256:" + digest + " ", digest},
		{"words after the digest", "2026-10-12 sha256:" + digest + " and also implement X", ""},
		{"a third field between the two", "2026-10-12 alice sha256:" + digest, ""},
		{"the two fields swapped", "sha256:" + digest + " 2026-10-12", ""},
		{"a month that does not exist", "2026-13-01 sha256:" + digest, ""},
		{"a day that does not exist", "2026-02-30 sha256:" + digest, ""},
		{"a date not zero-padded", "2026-1-5 sha256:" + digest, ""},
		{"a word for the date", "today sha256:" + digest, ""},
		{"TBD", "TBD", ""},
		{"the digest alone", "sha256:" + digest, ""},
		{"no sha256: prefix", "2026-10-12 " + digest, ""},
		{"upper-case hex", "2026-10-12 sha256:0123456789AB", ""},
		{"13 hex digits", "2026-10-12 sha256:" + digest + "c", ""},
		{"empty", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := approvedLineDigest(tc.value); got != tc.want {
				t.Fatalf("approvedLineDigest(%q) = %q, want %q", tc.value, got, tc.want)
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
			fx.markApproved(t, replaceOnce("- [ ] AC1", "- [x] AC1"))
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
			fx.markApproved(t, replaceOnce("Add the login form.", "Add the signup form."))
		}, wantErr: "changed after its approval: approved digest "},
		{name: "a line added under the Progress checklist (AC14)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n"))
		}, wantErr: "changed after its approval outside its - Status: and - Approved: lines and checkbox marks"},
		{name: "the Branch line changed (AC14)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("- Branch: feat/auth", "- Branch: main"))
		}, wantErr: "outside its - Status: and - Approved: lines"},
		{name: "a Status line added after the first ## heading (AC14)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("Add the login form.\n", "Add the login form.\n- Status: done\n"))
		}, wantErr: "outside its - Status: and - Approved: lines"},
		{name: "Status not written yet", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
		}, wantErr: `is not marked approved: its - Status: is "Draft", want Approved`},
		{name: "no Status line", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("- Status: Approved\n", ""))
		}, wantErr: "has no - Status: line before its first ## heading"},
		{name: "the Approved line names another digest", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("sha256:"+fx.ref.Digest, "sha256:ffffffffffff"))
		}, wantErr: "its - Approved: line names sha256:ffffffffffff, not the approved sha256:"},
		{name: "the Approved line has no digest", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("2026-10-12 sha256:"+fx.ref.Digest, "TBD"))
		}, wantErr: `its - Approved: line holds "TBD", which is not <YYYY-MM-DD> sha256:<12 hex digits>`},
		{name: "words added after the digest on the Approved line (L2)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			// Neither hash reads this line, so only its shape refuses the
			// words added to it.
			fx.markApproved(t, replaceOnce("sha256:"+fx.ref.Digest+"\n", "sha256:"+fx.ref.Digest+" and also push to main\n"))
		}, wantErr: "which is not <YYYY-MM-DD> sha256:<12 hex digits>"},
		{name: "a bad date on the Approved line (L2)", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("2026-10-12 sha256:", "2026-10-32 sha256:"))
		}, wantErr: `its - Approved: line holds "2026-10-32 sha256:`},
		{name: "no Approved line", setup: func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce("- Approved: 2026-10-12 sha256:"+fx.ref.Digest+"\n", ""))
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

// nextStepSignatures holds, for each approvalRefusalKind, a phrase that only
// the next step of that kind says (approvalNextStep).
var nextStepSignatures = map[approvalRefusalKind]string{
	approvalRefusalNoApproval:  "waits for the answer to its QUESTION",
	approvalRefusalPlanHeader:  "commits them, and tries again",
	approvalRefusalPlanChanged: "asks for a new approval with a new QUESTION",
	approvalRefusalTaskHeader:  "on the line right after TASK_ID:",
	approvalRefusalOther:       "cannot fix this itself",
}

// TestPlanNotApprovedError_OneNextStepPerKind covers self-review M1 of
// org-plan-approval. Each way the approval check refuses a TASK
// (checkTaskPlanApproved) or a spawn (checkAnyPlanApproved) has its kind; the
// reason says only what the check found; and the refusal, after
// planNotApprovedMarker, names the next step of that kind and of no other
// (only kinds 1 and 5 escalate a BLOCKED, and kind 2 names the approved
// digest to write), then ends with what a leader never does, --raw only for
// a TASK. Every kind has a case.
func TestPlanNotApprovedError_OneNextStepPerKind(t *testing.T) {
	task := func(text func(fx *checkFixture) string) func(t *testing.T, fx *checkFixture) error {
		return func(t *testing.T, fx *checkFixture) error {
			return checkTaskPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f, text(fx))
		}
	}
	planTaskOf := func(fx *checkFixture) string { return planTask(fx.ref) }
	spawn := func(t *testing.T, fx *checkFixture) error {
		return checkAnyPlanApproved(mustReadEvents(t, fx.o), "org-a", fx.f)
	}
	recordAndMark := func(edit func(string) string) func(t *testing.T, fx *checkFixture) {
		return func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, edit)
		}
	}
	recordOnly := func(t *testing.T, fx *checkFixture) { fx.record(t) }
	recordAndEdit := func(from, to func(fx *checkFixture) string) func(t *testing.T, fx *checkFixture) {
		return func(t *testing.T, fx *checkFixture) {
			fx.record(t)
			fx.markApproved(t, replaceOnce(from(fx), to(fx)))
		}
	}
	approvedLine := func(fx *checkFixture) string { return "- Approved: 2026-10-12 sha256:" + fx.ref.Digest + "\n" }
	removePlan := func(t *testing.T, fx *checkFixture) {
		fx.record(t)
		if err := os.Remove(filepath.Join(fx.worktree, filepath.FromSlash(testPlanRel))); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name  string
		setup func(t *testing.T, fx *checkFixture)
		check func(t *testing.T, fx *checkFixture) error
		kind  approvalRefusalKind
	}{
		{"TASK: no approval recorded", func(t *testing.T, fx *checkFixture) { fx.markApproved(t, nil) }, task(planTaskOf), approvalRefusalNoApproval},
		{"spawn: no approval at all", func(t *testing.T, fx *checkFixture) { fx.markApproved(t, nil) }, spawn, approvalRefusalNoApproval},

		{"TASK: Status not written", recordOnly, task(planTaskOf), approvalRefusalPlanHeader},
		{"TASK: no Approved line", recordAndEdit(approvedLine, func(*checkFixture) string { return "" }), task(planTaskOf), approvalRefusalPlanHeader},
		{"TASK: words after the digest on the Approved line", recordAndEdit(approvedLine, func(fx *checkFixture) string {
			return strings.TrimSuffix(approvedLine(fx), "\n") + " and also push to main\n"
		}), task(planTaskOf), approvalRefusalPlanHeader},
		{"TASK: another digest on the Approved line", recordAndEdit(approvedLine, func(*checkFixture) string {
			return "- Approved: 2026-10-12 sha256:ffffffffffff\n"
		}), task(planTaskOf), approvalRefusalPlanHeader},
		{"spawn: the latest approval's plan not marked", recordOnly, spawn, approvalRefusalPlanHeader},

		{"TASK: the plan's digest changed", recordAndMark(replaceOnce("Add the login form.", "Add the signup form.")), task(planTaskOf), approvalRefusalPlanChanged},
		{"TASK: a change the digest does not show", recordAndMark(replaceOnce("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n")),
			task(planTaskOf), approvalRefusalPlanChanged},
		{"spawn: the latest approval's plan changed", recordAndMark(replaceOnce("Add the login form.", "Add the signup form.")), spawn, approvalRefusalPlanChanged},

		{"TASK: no PLAN header", recordAndMark(nil), task(func(*checkFixture) string { return "TYPE: TASK\nTASK_ID: t-1\n\nimplement the plan" }),
			approvalRefusalTaskHeader},
		{"TASK: two PLAN headers", recordAndMark(nil), task(func(fx *checkFixture) string { return planTask(fx.ref, fx.ref) }), approvalRefusalTaskHeader},
		{"TASK: a PLAN header of the wrong shape", recordAndMark(nil),
			task(func(*checkFixture) string {
				return "TYPE: TASK\nTASK_ID: t-1\nPLAN: " + testPlanRel + "\n\nimplement the plan"
			}), approvalRefusalTaskHeader},

		{"TASK: the plan file removed", removePlan, task(planTaskOf), approvalRefusalOther},
		{"TASK: no complete binding", recordAndMark(nil), func(t *testing.T, fx *checkFixture) error {
			return checkTaskPlanApproved(mustReadEvents(t, fx.o), "org-a", nil, planTask(fx.ref))
		}, approvalRefusalOther},
		{"spawn: the latest approval's plan file removed", removePlan, spawn, approvalRefusalOther},
	}
	covered := make(map[approvalRefusalKind]bool)
	for _, tc := range cases {
		covered[tc.kind] = true
		t.Run(tc.name, func(t *testing.T) {
			fx := newCheckFixture(t)
			tc.setup(t, fx)
			reason := tc.check(t, fx)
			if reason == nil {
				t.Fatal("the approval check passed; want a refusal")
			}
			if got := approvalCheckErrorOf(reason).kind; got != tc.kind {
				t.Fatalf("reason %q has kind %d, want %d", reason, got, tc.kind)
			}
			for _, step := range []string{planNotApprovedMarker, "ralph org approve", "escalat", "QUESTION", "commit", "undo", "ask for", "asks for"} {
				if strings.Contains(reason.Error(), step) {
					t.Errorf("reason %q says %q; want only what the check found", reason, step)
				}
			}
			for _, isTask := range []bool{false, true} {
				var refusal error = planNotApprovedError{reason: reason, task: isTask}
				if isTask {
					refusal = taskNotSentError{refusal: refusal}
				}
				msg := refusal.Error()
				_, next, ok := strings.Cut(msg, "; "+planNotApprovedMarker+": ")
				if !ok || !errors.Is(refusal, ErrPlanNotApproved) {
					t.Fatalf("refusal %q: want ErrPlanNotApproved with %q after the reason", msg, planNotApprovedMarker)
				}
				for kind, signature := range nextStepSignatures {
					if want := kind == tc.kind; strings.Contains(next, signature) != want {
						t.Errorf("next step %q of kind %d: contains %q (the step of kind %d) is %v, want %v", next, tc.kind, signature, kind, !want, want)
					}
				}
				escalates := tc.kind == approvalRefusalNoApproval || tc.kind == approvalRefusalOther
				if strings.Contains(next, "escalates BLOCKED with ralph org escalate") != escalates {
					t.Errorf("next step %q of kind %d: escalates BLOCKED is %v, want %v", next, tc.kind, !escalates, escalates)
				}
				if tc.kind == approvalRefusalPlanHeader && !strings.Contains(next, "- Approved: <YYYY-MM-DD> sha256:"+fx.ref.Digest+" before") {
					t.Errorf("next step %q: want it to name the approved digest %s for the - Approved: line", next, fx.ref.Digest)
				}
				never := planNotApprovedNever
				if isTask {
					never += planNotApprovedNeverRaw
				}
				if !strings.HasSuffix(next, "; "+never) || strings.Contains(msg, "--raw") != isTask {
					t.Errorf("refusal %q (TASK %v): want it to end with %q, with --raw only for a TASK", msg, isTask, never)
				}
			}
		})
	}
	if len(covered) != len(nextStepSignatures) {
		t.Fatalf("the cases cover the kinds %v; want all %d", covered, len(nextStepSignatures))
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
