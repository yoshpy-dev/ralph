package org

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The record and the admission of a split plan's shared paths (plan
// 2026-11-22-split-shared-paths, AC3, AC5 and AC7 at
// the decision level). testFeature, boundReserveEvent, reserveEvent and
// oldBindingTokens live in reserve_test.go.

// keysBeforeShared are the binding keys a binary from before the shared
// paths reads.
var keysBeforeShared = []string{"split", "feature", "digest", "branch", "approval", "feature_digest"}

// sharedFeature is testFeature with the shared paths shared (the form
// FeatureBinding.Shared holds: normalized and joined with commas).
func sharedFeature(split, slug, digest, shared string) *FeatureBinding {
	f := testFeature(split, slug, digest)
	f.Shared = shared
	return f
}

// TestScopeReservedDetails_SharedRoundTrip covers AC3: a binding with shared
// paths writes them as the last binding token, `shared=<paths>`, right after
// `feature_digest=` and before the note, and reads back as written, with
// SharedPaths giving the list; the reservation's paths still read back, also
// with reservedPathsFromDetails, which cuts at the first space as older
// binaries do. A binding with no shared paths writes no token.
func TestScopeReservedDetails_SharedRoundTrip(t *testing.T) {
	const note = "restored: self pane close failed: herdr: refused"
	paths := []string{"docs/", "internal/auth/"}
	f := sharedFeature("split-1", "auth", "0123456789ab", "packages/shared/,pnpm-lock.yaml")
	const binding = "paths=docs/,internal/auth/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth approval=per-plan" +
		" feature_digest=" + testFeatureDigest + " shared=packages/shared/,pnpm-lock.yaml"
	for _, n := range []string{"", note} {
		ev := scopeReservedEvent("", "org-a", Reservation{Paths: paths, Feature: f}, n, false)
		want := binding
		if n != "" {
			want += " " + n
		}
		if ev.Details != want || ev.Worktree != f.Worktree {
			t.Fatalf("note %q: Details = %q (Worktree %q), want %q (Worktree %q)", n, ev.Details, ev.Worktree, want, f.Worktree)
		}
		got := reservationFromEvent(ev)
		if !slices.Equal(got.Paths, paths) || got.Feature == nil || *got.Feature != *f || !got.Feature.Complete() {
			t.Fatalf("note %q: reservationFromEvent = %q bound to %+v, want %q bound to %+v", n, got.Paths, got.Feature, paths, f)
		}
		if shared := got.Feature.SharedPaths(); !slices.Equal(shared, []string{"packages/shared/", "pnpm-lock.yaml"}) {
			t.Fatalf("note %q: SharedPaths = %q, want packages/shared/ and pnpm-lock.yaml", n, shared)
		}
		if old, ok := reservedPathsFromDetails(ev.Details); !ok || !slices.Equal(old, paths) {
			t.Fatalf("note %q: reservedPathsFromDetails(%q) = %q, %v; want %q", n, ev.Details, old, ok, paths)
		}
	}

	none := testFeature("split-1", "auth", "0123456789ab")
	ev := scopeReservedEvent("", "org-a", Reservation{Paths: paths, Feature: none}, note, false)
	if strings.Contains(ev.Details, "shared") {
		t.Fatalf("a binding with no shared paths wrote %q, want no shared token", ev.Details)
	}
	if got := reservationFromEvent(ev).Feature; got == nil || *got != *none || got.SharedPaths() != nil {
		t.Fatalf("read back %+v, want %+v with no shared paths", got, none)
	}
}

// TestScopeReservedDetails_SharedTokenLast covers the order AC3 asks for: the
// shared token follows `feature_digest=`, or `approval=` and `branch=` when
// the binding has no feature digest or no mode, so it is always the last
// binding token.
func TestScopeReservedDetails_SharedTokenLast(t *testing.T) {
	for _, tc := range []struct {
		name, approval, featureDigest, after string
	}{
		{"after feature_digest=", config.ApprovalModeSplit, testFeatureDigest, " feature_digest=" + testFeatureDigest},
		{"after approval= with no feature digest", config.ApprovalModeSplit, "", " approval=split"},
		{"after branch= with no mode and no feature digest", "", "", " branch=feat/auth"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := sharedFeature("split-1", "auth", "0123456789ab", "go.sum")
			f.Approval, f.FeatureDigest = tc.approval, tc.featureDigest
			details := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: f}, "restored: why")
			if !strings.Contains(details, tc.after+" shared=go.sum restored: why") {
				t.Fatalf("Details = %q, want the shared token right after %q and before the note", details, tc.after)
			}
			if got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: details, Worktree: f.Worktree}).Feature; got == nil || *got != *f {
				t.Fatalf("read back %+v, want %+v", got, f)
			}
		})
	}
}

// TestScopeReservedDetails_OlderBinaryReadsSixTokens covers the Risks entry
// of the plan on older binaries: one from before the shared paths, which
// reads split, feature, digest, branch, approval and feature_digest and
// stops at the first other key, reads a new record's six tokens as before
// and takes `shared=...` and the note after it as the note.
func TestScopeReservedDetails_OlderBinaryReadsSixTokens(t *testing.T) {
	for _, n := range []string{"", "restored: self pane close failed: herdr: refused"} {
		f := sharedFeature("split-1", "auth", "0123456789ab", "go.sum,pnpm-lock.yaml")
		details := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: f}, n)
		read, oldNote := oldBindingTokens(details, keysBeforeShared...)
		want := map[string]string{
			"split": "split-1", "feature": "auth", "digest": "0123456789ab", "branch": "feat/auth",
			"approval": config.ApprovalModePerPlan, "feature_digest": testFeatureDigest,
		}
		wantNote := strings.TrimSpace("shared=go.sum,pnpm-lock.yaml " + n)
		if !maps.Equal(read, want) || oldNote != wantNote {
			t.Fatalf("note %q: an older binary reads %v with note %q from %q, want %v with note %q", n, read, oldNote, details, want, wantNote)
		}
		if old, ok := reservedPathsFromDetails(details); !ok || !slices.Equal(old, []string{"docs/"}) {
			t.Fatalf("note %q: an older binary reads the paths %q, %v from %q, want docs/", n, old, ok, details)
		}
	}
}

// TestFeatureBinding_SharedRead covers AC3's older record and the reading of
// a damaged `shared=` value: the shared paths read back as written, and
// normalized again when they are not in the written form; no token (an
// older start, or a plan with no `- Shared:` line), the token given twice,
// an empty value, a value with a path the reservation rules refuse or with
// `.`, and a token after the note all read as no shared paths, never as the
// whole repo. In every case the binding stays complete and keeps its other
// fields.
func TestFeatureBinding_SharedRead(t *testing.T) {
	const wt = "/repo/.claude/worktrees/org-auth"
	const base = "paths=docs/ split=s feature=auth digest=0123456789ab branch=feat/auth approval=split feature_digest=" + testFeatureDigest
	tests := []struct {
		name, details, want string
	}{
		{"no token (an older start)", base, ""},
		{"no token, no feature digest (a start before both)", "paths=docs/ split=s feature=auth digest=0123456789ab branch=feat/auth approval=split", ""},
		{"one path", base + " shared=go.sum", "go.sum"},
		{"two paths and a note", base + " shared=go.sum,packages/shared/ restored: why", "go.sum,packages/shared/"},
		{"normalized again", base + " shared=./pnpm-lock.yaml,go.sum,packages//shared/.,go.sum", "go.sum,packages/shared/,pnpm-lock.yaml"},
		{"given twice, the same", base + " shared=go.sum shared=go.sum", ""},
		{"given twice, different", base + " shared=go.sum shared=pnpm-lock.yaml", ""},
		{"an empty value", base + " shared=", ""},
		{"an empty entry", base + " shared=go.sum,,pnpm-lock.yaml", ""},
		{"the whole repo", base + " shared=.", ""},
		{"the whole repo among others", base + " shared=go.sum,./", ""},
		{"an absolute path", base + " shared=/etc/passwd", ""},
		{"a .. segment", base + " shared=../go.sum", ""},
		{"after the note", base + " restored: shared=go.sum", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: tt.details, Worktree: wt}).Feature
			if got == nil {
				t.Fatal("expected a binding, got none")
			}
			if got.Shared != tt.want {
				t.Fatalf("Shared = %q, want %q", got.Shared, tt.want)
			}
			if tt.want == "" && got.SharedPaths() != nil {
				t.Fatalf("SharedPaths = %q, want none", got.SharedPaths())
			}
			if !got.Complete() || got.Approval != config.ApprovalModeSplit || got.Split != "s" || !sameFeature(got, got) {
				t.Fatalf("expected %+v to be complete, approval split, and equal to itself", got)
			}
		})
	}
}

// TestFeatureBinding_SharedPaths: a nil binding and one with no shared paths
// have none, and the list comes back in the order Shared holds it.
func TestFeatureBinding_SharedPaths(t *testing.T) {
	var unbound *FeatureBinding
	if got := unbound.SharedPaths(); got != nil {
		t.Fatalf("nil binding: SharedPaths = %q, want nil", got)
	}
	if got := (&FeatureBinding{}).SharedPaths(); got != nil {
		t.Fatalf("no shared paths: SharedPaths = %q, want nil", got)
	}
	if got := (&FeatureBinding{Shared: "go.sum,packages/shared/"}).SharedPaths(); !slices.Equal(got, []string{"go.sum", "packages/shared/"}) {
		t.Fatalf("SharedPaths = %q, want go.sum and packages/shared/", got)
	}
}

// TestFeatureBinding_StringNamesSharedPaths: the binding's text names its
// shared paths when it has any, and says nothing of them otherwise.
func TestFeatureBinding_StringNamesSharedPaths(t *testing.T) {
	if s := sharedFeature("split-1", "auth", "0123456789ab", "go.sum").String(); !strings.HasSuffix(s, `approval "per-plan", shared paths "go.sum")`) {
		t.Fatalf("String() = %q, want it to end with the shared paths", s)
	}
	if s := testFeature("split-1", "auth", "0123456789ab").String(); strings.Contains(s, "shared") {
		t.Fatalf("String() = %q, want no shared paths named", s)
	}
}

// TestValidateFeatureBinding_Shared: Spawn records shared paths only in the
// form SplitPlan.Shared gives them, normalized, sorted, without duplicates
// and without `.`, joined with commas; no shared paths pass too.
func TestValidateFeatureBinding_Shared(t *testing.T) {
	for _, ok := range []string{"", "go.sum", "packages/shared/,pnpm-lock.yaml", "a=b.json"} {
		if err := validateFeatureBinding(*sharedFeature("split-1", "auth", "0123456789ab", ok)); err != nil {
			t.Errorf("shared %q: %v, want it accepted", ok, err)
		}
	}
	for _, bad := range []string{".", "go.sum,.", "pnpm-lock.yaml,go.sum", "go.sum,go.sum", "./go.sum", "packages//shared/", "go.sum, pnpm-lock.yaml", "/go.sum", "../go.sum", ","} {
		err := validateFeatureBinding(*sharedFeature("split-1", "auth", "0123456789ab", bad))
		if err == nil || !strings.Contains(err.Error(), `shared paths "`+bad+`"`) {
			t.Errorf("shared %q: %v, want a refusal naming it", bad, err)
		}
	}
}

// TestReservationDecision_Shared covers AC5 and AC7 at the decision level.
// org-plain holds internal/auth/ with a plain --reserve, and org-shared,
// bound to a feature of split plan split-s, holds feat-a/ and shares
// packages/shared/ and pnpm-lock.yaml. A reservation whose paths overlap
// org-shared's shared paths is refused (the shared org started first), and
// so is a bound one whose shared paths overlap org-plain's or org-shared's
// paths (the reserving org started first), file against file, file against
// directory and directory against directory, each naming the other org, the
// overlapping pairs and which side shares the path. Shared paths that
// overlap the other org's shared paths pass, paths that overlap another
// org's paths are refused as before, and a sibling with a common prefix is
// no overlap.
func TestReservationDecision_Shared(t *testing.T) {
	shared := sharedFeature("split-s", "feat-a", "0123456789ab", "packages/shared/,pnpm-lock.yaml")
	events := []ManifestEvent{
		reserveEvent("org-plain", "internal/auth/"),
		boundReserveEvent("org-shared", shared, "feat-a/"),
	}
	const sharedThere = `reservation overlaps the shared paths of running org_id "org-shared" ` +
		`(each pair names the reserved path first, then the path split plan "split-s" shares, which every org of it may change): `
	const sharedThereFix = `: a path another org shares cannot be reserved; reserve other paths, or wait until org_id "org-shared" is disbanded`
	sharedHere := func(other string) string {
		return `the shared paths of split plan "split-new" overlap the reservation of running org_id "` + other + `" ` +
			`(each pair names the shared path first, then the reserved one): `
	}
	sharedHereFix := func(other string) string {
		return `: a path one org reserves cannot be shared by another; take it out of the split plan's - Shared: line, or wait until org_id "` +
			other + `" is disbanded`
	}
	bound := func(shared string, paths ...string) Reservation {
		f := sharedFeature("split-new", "new", "0a1b2c3d4e5f", shared)
		f.Worktree = "/repo/.claude/worktrees/org-new"
		return Reservation{Paths: paths, Feature: f}
	}
	tests := []struct {
		name       string
		want       Reservation
		wantRecord bool
		wantError  []string
	}{
		// The shared org started first: paths that overlap its shared paths.
		{"a file that is a shared file", Reservation{Paths: []string{"pnpm-lock.yaml"}}, false,
			[]string{sharedThere + "pnpm-lock.yaml overlaps pnpm-lock.yaml" + sharedThereFix}},
		{"a file under a shared directory", Reservation{Paths: []string{"packages/shared/index.ts"}}, false,
			[]string{sharedThere + "packages/shared/index.ts overlaps packages/shared/" + sharedThereFix}},
		{"a directory holding a shared directory", Reservation{Paths: []string{"packages/"}}, false,
			[]string{sharedThere + "packages/ overlaps packages/shared/" + sharedThereFix}},
		{"every overlapping pair is named", Reservation{Paths: []string{"docs/", "packages/", "pnpm-lock.yaml"}}, false,
			[]string{sharedThere + "packages/ overlaps packages/shared/, pnpm-lock.yaml overlaps pnpm-lock.yaml" + sharedThereFix}},
		{"a sibling of a shared directory", Reservation{Paths: []string{"packages/sharedx/"}}, true, nil},
		// The reserving org started first: shared paths that overlap its paths.
		{"a shared file under a reserved directory", bound("internal/auth/token.go", "new/"), false,
			[]string{sharedHere("org-plain") + "internal/auth/token.go overlaps internal/auth/" + sharedHereFix("org-plain")}},
		{"a shared directory holding a reserved directory", bound("internal/", "new/"), false,
			[]string{sharedHere("org-plain") + "internal/ overlaps internal/auth/" + sharedHereFix("org-plain")}},
		{"a shared directory under a bound org's reserved directory", bound("feat-a/gen/", "new/"), false,
			[]string{sharedHere("org-shared") + "feat-a/gen/ overlaps feat-a/" + sharedHereFix("org-shared")}},
		{"a shared directory next to a reserved one with the same prefix", bound("internal/authz/", "new/"), true, nil},
		// Shared against shared passes; paths against paths as before.
		{"the same shared paths", bound("packages/shared/,pnpm-lock.yaml", "new/"), true, nil},
		{"a shared directory holding another org's shared directory", bound("packages/", "new/"), true, nil},
		{"paths that overlap a reservation", Reservation{Paths: []string{"feat-a/x.go"}}, false,
			[]string{`org: reservation overlaps the reservation of running org_id "org-shared": feat-a/x.go overlaps feat-a/`}},
		{"bound paths that overlap a reservation", bound("go.sum", "internal/auth/x.go"), false,
			[]string{`org: reservation overlaps the reservation of running org_id "org-plain": internal/auth/x.go overlaps internal/auth/`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := reservationDecision(events, "org-new", tt.want)
			if len(tt.wantError) == 0 {
				if err != nil || record != tt.wantRecord {
					t.Fatalf("reservationDecision = %v, %v; want %v, nil", record, err, tt.wantRecord)
				}
				return
			}
			if err == nil || record {
				t.Fatalf("reservationDecision = %v, %v; want a refusal", record, err)
			}
			for _, w := range tt.wantError {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
			if strings.Contains(err.Error(), "docs/ overlaps") || strings.Contains(err.Error(), "new/ overlaps") {
				t.Errorf("error %q names a path that overlaps nothing", err)
			}
		})
	}
}

// TestReservationDecision_SharedOwnOrg covers AC3's restart at the decision
// level: an org whose record holds shared paths passes, with nothing to
// record, the same binding again, and is refused one whose shared paths
// changed (and so its feature digest), naming both lists and both feature
// digests; an org with no shared paths that gets some is refused the same
// way. Another org's damaged `shared=` value holds nothing off.
func TestReservationDecision_SharedOwnOrg(t *testing.T) {
	held := sharedFeature("split-1", "auth", "0123456789ab", "go.sum")
	events := []ManifestEvent{boundReserveEvent("org-a", held, "internal/auth/")}
	if record, err := reservationDecision(events, "org-a", Reservation{Paths: []string{"internal/auth/"}, Feature: sharedFeature("split-1", "auth", "0123456789ab", "go.sum")}); err != nil || record {
		t.Fatalf("the same binding again: %v, %v; want false, nil", record, err)
	}
	for _, tc := range []struct {
		name, heldShared, wantShared, wantClause string
	}{
		{"another list", "go.sum", "go.sum,pnpm-lock.yaml", `("go.sum", now "go.sum,pnpm-lock.yaml"; feature digest "` + testFeatureDigest + `", now "0a1b2c3d4e5f")`},
		{"none before", "", "go.sum", `(none, now "go.sum"; feature digest "` + testFeatureDigest + `", now "0a1b2c3d4e5f")`},
		{"none now", "go.sum", "", `("go.sum", now none; feature digest "` + testFeatureDigest + `", now "0a1b2c3d4e5f")`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := []ManifestEvent{boundReserveEvent("org-a", sharedFeature("split-1", "auth", "0123456789ab", tc.heldShared), "internal/auth/")}
			want := sharedFeature("split-1", "auth", "0123456789ab", tc.wantShared)
			want.FeatureDigest = "0a1b2c3d4e5f"
			record, err := reservationDecision(events, "org-a", Reservation{Paths: []string{"internal/auth/"}, Feature: want})
			wantErr := `is refused, since the shared paths of split plan "split-1" (its - Shared: line) changed after org_id "org-a" was started ` +
				tc.wantClause + ": ralph org disband --org-id org-a first, or use another org_id"
			if err == nil || record || !strings.Contains(err.Error(), wantErr) {
				t.Fatalf("reservationDecision = %v, %v; want a refusal containing %q", record, err, wantErr)
			}
			if strings.Contains(err.Error(), "### auth") {
				t.Fatalf("error %q says the section or preamble changed, want only the shared paths", err)
			}
		})
	}

	damaged := []ManifestEvent{{
		OrgID: "org-b", Event: EventScopeReserved, Worktree: "/repo/.claude/worktrees/org-b",
		Details: "paths=b/ split=split-1 feature=b digest=0123456789ab branch=feat/b approval=per-plan feature_digest=" + testFeatureDigest + " shared=.",
	}}
	if record, err := reservationDecision(damaged, "org-c", Reservation{Paths: []string{"go.sum"}}); err != nil || !record {
		t.Fatalf("a damaged shared= value: %v, %v; want true, nil (read as no shared paths)", record, err)
	}
}
