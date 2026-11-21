package org

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The record of the issue a feature org was started for (plan
// 2026-11-22-org-start-issue, AC1, AC2, AC5 and AC6 at the record and
// decision level). testFeature, boundReserveEvent and oldBindingTokens live
// in reserve_test.go, sharedFeature and keysBeforeShared in
// reserve_shared_test.go.

// keysBeforeIssue are the binding keys a binary from before the issue reads.
var keysBeforeIssue = []string{"split", "feature", "digest", "branch", "approval", "feature_digest", "shared"}

// issueFeature is testFeature with the issue issue.
func issueFeature(split, slug, digest, issue string) *FeatureBinding {
	f := testFeature(split, slug, digest)
	f.Issue = issue
	return f
}

// TestScopeReservedDetails_IssueRoundTrip covers AC1 at the record level: a
// binding with an issue writes it as the last binding token, `issue=<n>`,
// right after `shared=` when there are shared paths and after
// `feature_digest=` when there are none, before the note, and reads back as
// written; the reservation's paths still read back, also with
// reservedPathsFromDetails, which cuts at the first space as older binaries
// do.
func TestScopeReservedDetails_IssueRoundTrip(t *testing.T) {
	const note = "restored: self pane close failed: herdr: refused"
	paths := []string{"docs/", "internal/auth/"}
	const head = "paths=docs/,internal/auth/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth approval=per-plan" +
		" feature_digest=" + testFeatureDigest
	withShared := issueFeature("split-1", "auth", "0123456789ab", "12")
	withShared.Shared = "go.sum,packages/shared/"
	for _, tc := range []struct {
		name    string
		f       *FeatureBinding
		binding string
	}{
		{"no shared paths", issueFeature("split-1", "auth", "0123456789ab", "12"), head + " issue=12"},
		{"shared paths", withShared, head + " shared=go.sum,packages/shared/ issue=12"},
		{"ten digits", issueFeature("split-1", "auth", "0123456789ab", "9999999999"), head + " issue=9999999999"},
	} {
		for _, n := range []string{"", note} {
			ev := scopeReservedEvent("", "org-a", Reservation{Paths: paths, Feature: tc.f}, n, false)
			want := tc.binding
			if n != "" {
				want += " " + n
			}
			if ev.Details != want || ev.Worktree != tc.f.Worktree {
				t.Fatalf("%s, note %q: Details = %q (Worktree %q), want %q (Worktree %q)", tc.name, n, ev.Details, ev.Worktree, want, tc.f.Worktree)
			}
			got := reservationFromEvent(ev)
			if !slices.Equal(got.Paths, paths) || got.Feature == nil || *got.Feature != *tc.f || !got.Feature.Complete() {
				t.Fatalf("%s, note %q: reservationFromEvent = %q bound to %+v, want %q bound to %+v", tc.name, n, got.Paths, got.Feature, paths, tc.f)
			}
			if old, ok := reservedPathsFromDetails(ev.Details); !ok || !slices.Equal(old, paths) {
				t.Fatalf("%s, note %q: reservedPathsFromDetails(%q) = %q, %v; want %q", tc.name, n, ev.Details, old, ok, paths)
			}
		}
	}
}

// TestScopeReservedDetails_NoIssueIsTheRecordOfBefore covers AC2 at the
// record level: a binding with no issue writes no `issue=` token, and its
// Details are byte for byte those this function wrote before the issue
// (printed by scopeReservedDetails at 81530fb0, the commit before plan
// 2026-11-22-org-start-issue).
func TestScopeReservedDetails_NoIssueIsTheRecordOfBefore(t *testing.T) {
	b := &FeatureBinding{Split: "split-1", Feature: "auth", Digest: "0123456789ab", Branch: "feat/auth", Worktree: "/wt",
		Approval: "per-plan", FeatureDigest: "fedcba987654", Shared: "go.sum"}
	const before = "paths=docs/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth approval=per-plan " +
		"feature_digest=fedcba987654 shared=go.sum restored: why"
	if got := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: b}, "restored: why"); got != before {
		t.Fatalf("Details = %q, want the record of before %q", got, before)
	}
	if got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: before, Worktree: "/wt"}).Feature; got == nil || *got != *b || got.Issue != "" {
		t.Fatalf("read back %+v, want %+v with no issue", got, b)
	}
}

// TestScopeReservedDetails_IssueTokenLast: the issue token follows
// `shared=`, `feature_digest=`, `approval=` or `branch=`, whichever is the
// last token the binding has before it, so it is always the last binding
// token.
func TestScopeReservedDetails_IssueTokenLast(t *testing.T) {
	for _, tc := range []struct {
		name, approval, featureDigest, shared, after string
	}{
		{"after shared=", config.ApprovalModeSplit, testFeatureDigest, "go.sum", " shared=go.sum"},
		{"after feature_digest=", config.ApprovalModeSplit, testFeatureDigest, "", " feature_digest=" + testFeatureDigest},
		{"after approval= with no feature digest", config.ApprovalModeSplit, "", "", " approval=split"},
		{"after branch= with no mode and no feature digest", "", "", "", " branch=feat/auth"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := issueFeature("split-1", "auth", "0123456789ab", "365")
			f.Approval, f.FeatureDigest, f.Shared = tc.approval, tc.featureDigest, tc.shared
			details := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: f}, "restored: why")
			if !strings.HasSuffix(details, tc.after+" issue=365 restored: why") {
				t.Fatalf("Details = %q, want the issue token right after %q and before the note", details, tc.after)
			}
			if got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: details, Worktree: f.Worktree}).Feature; got == nil || *got != *f {
				t.Fatalf("read back %+v, want %+v", got, f)
			}
		})
	}
}

// TestScopeReservedDetails_OlderBinaryReadsSevenTokens covers the Risks
// entry of the plan on older binaries: one from before the issue, which
// reads split, feature, digest, branch, approval, feature_digest and shared
// and stops at the first other key, reads a new record's seven tokens as
// before and takes `issue=...` and the note after it as the note; one from
// before the shared paths reads the six tokens of a record with an issue
// and no shared paths the same way.
func TestScopeReservedDetails_OlderBinaryReadsSevenTokens(t *testing.T) {
	want := map[string]string{
		"split": "split-1", "feature": "auth", "digest": "0123456789ab", "branch": "feat/auth",
		"approval": config.ApprovalModePerPlan, "feature_digest": testFeatureDigest,
	}
	for _, n := range []string{"", "restored: self pane close failed: herdr: refused"} {
		f := issueFeature("split-1", "auth", "0123456789ab", "12")
		f.Shared = "go.sum"
		details := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: f}, n)
		read, oldNote := oldBindingTokens(details, keysBeforeIssue...)
		wantSeven := maps.Clone(want)
		wantSeven["shared"] = "go.sum"
		if wantNote := strings.TrimSpace("issue=12 " + n); !maps.Equal(read, wantSeven) || oldNote != wantNote {
			t.Fatalf("note %q: a binary from before the issue reads %v with note %q from %q, want %v with note %q",
				n, read, oldNote, details, wantSeven, wantNote)
		}
		if old, ok := reservedPathsFromDetails(details); !ok || !slices.Equal(old, []string{"docs/"}) {
			t.Fatalf("note %q: an older binary reads the paths %q, %v from %q, want docs/", n, old, ok, details)
		}

		noShared := issueFeature("split-1", "auth", "0123456789ab", "12")
		details = scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: noShared}, n)
		read, oldNote = oldBindingTokens(details, keysBeforeShared...)
		if wantNote := strings.TrimSpace("issue=12 " + n); !maps.Equal(read, want) || oldNote != wantNote {
			t.Fatalf("note %q: a binary from before the shared paths reads %v with note %q from %q, want %v with note %q",
				n, read, oldNote, details, want, wantNote)
		}
	}
}

// TestFeatureBinding_IssueRead covers AC6: the issue reads back as written,
// also when a hand-written record puts `issue=` before `shared=`; no token
// (an older start, or a start with no --issue), the token given twice, an
// empty value, a value that is not an issue number (zero, a leading zero, a
// `#`, a sign, a letter, a space-free non-ASCII digit, eleven digits) and a
// token after the note all read as no issue. In every case the binding
// stays complete and keeps its other fields, the shared paths included.
func TestFeatureBinding_IssueRead(t *testing.T) {
	const wt = "/repo/.claude/worktrees/org-auth"
	const base = "paths=docs/ split=s feature=auth digest=0123456789ab branch=feat/auth approval=split feature_digest=" + testFeatureDigest
	tests := []struct {
		name, details, want, wantShared string
	}{
		{"no token (an older start)", base, "", ""},
		{"no token, with shared paths", base + " shared=go.sum", "", "go.sum"},
		{"one digit", base + " issue=7", "7", ""},
		{"ten digits", base + " issue=1234567890", "1234567890", ""},
		{"after shared= and before a note", base + " shared=go.sum issue=365 restored: why", "365", "go.sum"},
		{"before shared= (hand-written)", base + " issue=365 shared=go.sum", "365", "go.sum"},
		{"given twice, the same", base + " issue=12 issue=12", "", ""},
		{"given twice, different", base + " issue=12 issue=13", "", ""},
		{"an empty value", base + " issue=", "", ""},
		{"zero", base + " issue=0", "", ""},
		{"a leading zero", base + " issue=012", "", ""},
		{"a #", base + " issue=#12", "", ""},
		{"a sign", base + " issue=+12", "", ""},
		{"a negative number", base + " issue=-12", "", ""},
		{"a letter", base + " issue=12a", "", ""},
		{"not a number", base + " issue=abc", "", ""},
		{"full-width digits", base + " issue=１２", "", ""},
		{"eleven digits", base + " issue=12345678901", "", ""},
		{"after the note", base + " restored: issue=12", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: tt.details, Worktree: wt}).Feature
			if got == nil {
				t.Fatal("expected a binding, got none")
			}
			if got.Issue != tt.want || got.Shared != tt.wantShared {
				t.Fatalf("Issue = %q, Shared = %q; want %q, %q", got.Issue, got.Shared, tt.want, tt.wantShared)
			}
			if !got.Complete() || got.Approval != config.ApprovalModeSplit || got.FeatureDigest != testFeatureDigest || !sameFeature(got, got) {
				t.Fatalf("expected %+v to be complete, approval split, with its feature digest, and equal to itself", got)
			}
		})
	}
}

// TestSameFeature_Issue covers AC5 at the binding level: two bindings with
// the same issue, or both with none, are the same; another issue, and an
// issue on one side only, either way round, make them differ, also when
// every other field is the same.
func TestSameFeature_Issue(t *testing.T) {
	for _, tc := range []struct {
		name, a, b string
		want       bool
	}{
		{"both with no issue", "", "", true},
		{"the same issue", "12", "12", true},
		{"another issue", "12", "13", false},
		{"an issue on one side only", "12", "", false},
	} {
		a, b := issueFeature("split-1", "auth", "0123456789ab", tc.a), issueFeature("split-1", "auth", "0123456789ab", tc.b)
		if got, back := sameFeature(a, b), sameFeature(b, a); got != tc.want || back != tc.want {
			t.Errorf("%s: sameFeature = %v, %v the other way; want %v", tc.name, got, back, tc.want)
		}
	}
}

// TestFeatureBinding_StringNamesTheIssue: the binding's text names its issue
// when it has one, after the shared paths, and says nothing of an issue
// otherwise.
func TestFeatureBinding_StringNamesTheIssue(t *testing.T) {
	f := issueFeature("split-1", "auth", "0123456789ab", "12")
	if s := f.String(); !strings.HasSuffix(s, `approval "per-plan", issue #12)`) {
		t.Fatalf("String() = %q, want it to end with the issue", s)
	}
	f.Shared = "go.sum"
	if s := f.String(); !strings.HasSuffix(s, `approval "per-plan", shared paths "go.sum", issue #12)`) {
		t.Fatalf("String() = %q, want it to end with the shared paths and then the issue", s)
	}
	if s := testFeature("split-1", "auth", "0123456789ab").String(); strings.Contains(s, "issue") {
		t.Fatalf("String() = %q, want no issue named", s)
	}
}

// TestValidateFeatureBinding_Issue: Spawn records an issue only in the form
// start gives it, 1 to 10 decimal digits with no leading zero and no `#`;
// no issue passes too.
func TestValidateFeatureBinding_Issue(t *testing.T) {
	for _, ok := range []string{"", "1", "12", "365", "9999999999"} {
		if err := validateFeatureBinding(*issueFeature("split-1", "auth", "0123456789ab", ok)); err != nil {
			t.Errorf("issue %q: %v, want it accepted", ok, err)
		}
	}
	for _, bad := range []string{"0", "012", "#12", "12a", "abc", " 12", "12 ", "+12", "-12", "1.0", "１２", "12345678901", "1 2"} {
		err := validateFeatureBinding(*issueFeature("split-1", "auth", "0123456789ab", bad))
		if err == nil || !strings.Contains(err.Error(), `issue "`+bad+`" is not an issue number (1 to 10 decimal digits with no leading zero and no #)`) {
			t.Errorf("issue %q: %v, want a refusal naming it and the form", bad, err)
		}
	}
}

// TestReservationDecision_IssueOwnOrg covers AC5 at the decision level: an
// org whose record holds an issue passes, with nothing to record, the same
// binding again, and so does one with no issue given none; one bound with
// another issue, with none when it has one, or with one when it has none
// is refused, naming the issue of each side and the --issue that starts it
// as it was. When the feature digest changed as well, the refusal says both.
func TestReservationDecision_IssueOwnOrg(t *testing.T) {
	paths := []string{"internal/auth/"}
	for _, issue := range []string{"", "12"} {
		events := []ManifestEvent{boundReserveEvent("org-a", issueFeature("split-1", "auth", "0123456789ab", issue), paths...)}
		want := Reservation{Paths: paths, Feature: issueFeature("split-1", "auth", "0123456789ab", issue)}
		if record, err := reservationDecision(events, "org-a", want); err != nil || record {
			t.Fatalf("issue %q, the same binding again: %v, %v; want false, nil", issue, record, err)
		}
	}
	for _, tc := range []struct {
		name, held, want, clause string
	}{
		{"another issue", "12", "13",
			`is refused, since org_id "org-a" was started with issue #12 and this start has issue #13 ` +
				`(a start of the org that is not disbanded names the issue it was started with: run it with --issue 12): `},
		{"no issue now", "12", "",
			`is refused, since org_id "org-a" was started with issue #12 and this start has no issue ` +
				`(a start of the org that is not disbanded names the issue it was started with: run it with --issue 12): `},
		{"no issue before", "", "13",
			`is refused, since org_id "org-a" was started with no issue and this start has issue #13 ` +
				`(a start of the org that is not disbanded names the issue it was started with: run it with no --issue): `},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := []ManifestEvent{boundReserveEvent("org-a", issueFeature("split-1", "auth", "0123456789ab", tc.held), paths...)}
			record, err := reservationDecision(events, "org-a", Reservation{Paths: paths, Feature: issueFeature("split-1", "auth", "0123456789ab", tc.want)})
			wantErr := tc.clause + "ralph org disband --org-id org-a first, or use another org_id"
			if err == nil || record || !strings.Contains(err.Error(), wantErr) {
				t.Fatalf("reservationDecision = %v, %v; want a refusal containing %q", record, err, wantErr)
			}
			if strings.Contains(err.Error(), "feature digest \""+testFeatureDigest+"\", now") {
				t.Fatalf("error %q says the feature digest changed, want only the issue", err)
			}
		})
	}

	t.Run("the feature digest and the issue", func(t *testing.T) {
		events := []ManifestEvent{boundReserveEvent("org-a", issueFeature("split-1", "auth", "0123456789ab", "12"), paths...)}
		want := issueFeature("split-1", "auth", "0123456789ab", "13")
		want.FeatureDigest = "0a1b2c3d4e5f"
		_, err := reservationDecision(events, "org-a", Reservation{Paths: paths, Feature: want})
		wantErr := `is refused, since the feature's section (### auth) or the preamble of split plan "split-1" changed after org_id "org-a" ` +
			`was started (feature digest "` + testFeatureDigest + `", now "0a1b2c3d4e5f"), and since org_id "org-a" was started with ` +
			`issue #12 and this start has issue #13 (a start of the org that is not disbanded names the issue it was started with: ` +
			`run it with --issue 12): ralph org disband --org-id org-a first, or use another org_id`
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("reservationDecision = %v; want a refusal containing %q", err, wantErr)
		}
	})

	t.Run("another feature names no issue clause", func(t *testing.T) {
		events := []ManifestEvent{boundReserveEvent("org-a", issueFeature("split-1", "auth", "0123456789ab", "12"), paths...)}
		other := issueFeature("split-1", "api", "0123456789ab", "13")
		_, err := reservationDecision(events, "org-a", Reservation{Paths: paths, Feature: other})
		if err == nil || strings.Contains(err.Error(), "was started with issue") {
			t.Fatalf("reservationDecision = %v; want a refusal that names the bindings only", err)
		}
	})
}
