package org

import (
	"encoding/json"
	"testing"
)

// The issue of a branch's binding in `ralph org merge-order` (plan
// 2026-11-22-org-start-issue, AC10 at the org layer). The moFixture and its
// helpers live in mergeorder_test.go.

// moBindIssue is moBind for an org started for issue ("" for none).
func moBindIssue(orgID, slug, issue string) ManifestEvent {
	f := testFeature("split-1", slug, "0123456789ab")
	f.Issue = issue
	return boundReserveEvent(orgID, f, "internal/"+slug+"/")
}

// moBindIssueTo is moBindIssue on branch.
func moBindIssueTo(orgID, slug, branch, issue string) ManifestEvent {
	f := testFeature("split-1", slug, "0123456789ab")
	f.Branch, f.Issue = branch, issue
	return boundReserveEvent(orgID, f, "internal/"+slug+"/")
}

// TestMergeOrder_BindingIssue covers AC10: a branch's binding holds the
// issue of the binding it was started for, after its org was disbanded
// (feat/one, feat/solo) as while it runs (feat/two); an org_id started
// again for another feature keeps each branch's own issue (feat/one and
// feat/two); a branch two bindings name has the latest one's (feat/auth,
// whose latest binding has none); and a binding with no issue has none
// (feat/three). In --json, issue is a number, and left out for none.
func TestMergeOrder_BindingIssue(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBindIssue("org-a", "one", "12"), moDisbanded("org-a"), moBindIssue("org-a", "two", "34"),
		moBindIssue("org-b", "three", ""), moDisbanded("org-b"),
		moBindIssue("org-c", "solo", "9999999999"), moDisbanded("org-c"),
		moBindIssueTo("org-d", "auth", "feat/auth", "7"), moDisbanded("org-d"), moBindIssueTo("org-e", "auth2", "feat/auth", ""),
		moPassed("org-g", "feat/orphan", moHeadA, moMain),
	)
	fx.exec.lsOut = moListing("main", "main", moMain)
	res := fx.run(t, true)
	want := map[string]int64{"feat/one": 12, "feat/two": 34, "feat/three": 0, "feat/solo": 9999999999, "feat/auth": 0}
	for branch, issue := range want {
		if e := moEntry(t, res, branch); e.Binding == nil || e.Binding.Issue != issue {
			t.Errorf("%s binding = %+v, want issue %d", branch, e.Binding, issue)
		}
	}
	if e := moEntry(t, res, "feat/auth"); e.Binding == nil || e.Binding.OrgID != "org-e" {
		t.Errorf("feat/auth binding = %+v, want org-e's, the latest", e.Binding)
	}

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Entries []struct {
			Branch  string                     `json:"branch"`
			Binding map[string]json.RawMessage `json:"binding"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	wantJSON := map[string]string{"feat/one": "12", "feat/two": "34", "feat/solo": "9999999999"}
	seen := 0
	for _, e := range got.Entries {
		raw, ok := e.Binding["issue"]
		switch w, has := wantJSON[e.Branch]; {
		case has:
			seen++
			if string(raw) != w {
				t.Errorf("%s issue = %s (present %v), want the number %s", e.Branch, raw, ok, w)
			}
		case ok:
			t.Errorf("%s has issue %s, want the key left out", e.Branch, raw)
		}
	}
	if seen != len(wantJSON) {
		t.Fatalf("saw %d of the branches with an issue in %s, want %d", seen, b, len(wantJSON))
	}
}

// TestMergeOrderIssue: an issue number reads as its number, and no issue as
// 0.
func TestMergeOrderIssue(t *testing.T) {
	for in, want := range map[string]int64{"": 0, "1": 1, "365": 365, "9999999999": 9999999999} {
		if got := mergeOrderIssue(in); got != want {
			t.Errorf("mergeOrderIssue(%q) = %d, want %d", in, got, want)
		}
	}
}
