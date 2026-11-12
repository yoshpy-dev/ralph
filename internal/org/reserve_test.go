package org

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestNormalizeReservePaths covers plan AC6's path rules: a trailing `/` (or
// `/.`) is a directory, anything else a file, `.` / `./` the whole repo;
// redundant `./` and repeated slashes go away, and the list comes back sorted
// without duplicates so the same set compares equal however it was written.
func TestNormalizeReservePaths(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
		want []string
	}{
		{"directory kept", []string{"internal/auth/"}, []string{"internal/auth/"}},
		{"file kept", []string{"internal/auth/token.go"}, []string{"internal/auth/token.go"}},
		{"dot is the whole repo", []string{"."}, []string{"."}},
		{"dot slash is the whole repo", []string{"./"}, []string{"."}},
		{"leading dot slash dropped", []string{"./README.md"}, []string{"README.md"}},
		{"repeated slashes and inner dot dropped", []string{"internal//./auth//"}, []string{"internal/auth/"}},
		{"trailing slash dot is a directory", []string{"internal/."}, []string{"internal/"}},
		{"sorted and deduplicated", []string{"b.go", "a/", "./b.go", "a//"}, []string{"a/", "b.go"}},
		{"a file and a directory of the same name stay apart", []string{"docs", "docs/"}, []string{"docs", "docs/"}},
		{"no paths is no reservation", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeReservePaths(tt.raw)
			if err != nil {
				t.Fatalf("NormalizeReservePaths(%q): %v", tt.raw, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("NormalizeReservePaths(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// TestNormalizeReservePaths_Rejects covers the rejected inputs of plan AC6:
// an empty path, an absolute path, any `..` segment, and the characters the
// manifest record cannot hold. One bad path rejects the whole list.
func TestNormalizeReservePaths_Rejects(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
		want string
	}{
		{"empty", []string{""}, "is empty"},
		{"empty among good ones", []string{"internal/", ""}, "is empty"},
		{"absolute", []string{"/etc/passwd"}, "is absolute"},
		{"absolute directory", []string{"//"}, "is absolute"},
		{"leading dot dot", []string{"../outside"}, "has a .. segment"},
		{"inner dot dot", []string{"internal/../cmd/"}, "has a .. segment"},
		{"trailing dot dot", []string{"internal/.."}, "has a .. segment"},
		{"bare dot dot", []string{".."}, "has a .. segment"},
		{"comma", []string{"a,b.go"}, "has a comma"},
		{"space", []string{"my file.go"}, "has a comma, whitespace"},
		{"tab", []string{"a\tb"}, "has a comma, whitespace"},
		{"control character", []string{"a\x01b"}, "control character"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeReservePaths(tt.raw)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("NormalizeReservePaths(%q) = %q, %v; want an error containing %q", tt.raw, got, err, tt.want)
			}
		})
	}
}

// TestReservePathsOverlap covers plan AC5 and AC6's overlap rules, including
// the AC5 examples against internal/auth/, in both argument orders.
func TestReservePathsOverlap(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"internal/auth/", "internal/auth/token.go", true},
		{"internal/auth/", "internal/", true},
		{"internal/auth/", ".", true},
		{"internal/auth/", "internal/authz/", false},
		{"internal/auth/", "internal/auth/", true},
		{"internal/auth/", "internal/auth/sub/", true},
		{"internal/auth/", "internal/auth", false},
		{"internal/auth/", "internal/authz.go", false},
		{"internal/auth/token.go", "internal/auth/token.go", true},
		{"internal/auth/token.go", "internal/auth/token_test.go", false},
		{"internal/auth/token.go", "internal/auth/token.go.bak", false},
		{".", ".", true},
		{".", "README.md", true},
		{"docs/", "internal/", false},
	}
	for _, tt := range tests {
		for _, pair := range [][2]string{{tt.a, tt.b}, {tt.b, tt.a}} {
			if got := reservePathsOverlap(pair[0], pair[1]); got != tt.want {
				t.Errorf("reservePathsOverlap(%q, %q) = %v, want %v", pair[0], pair[1], got, tt.want)
			}
		}
	}
}

func TestScopeReservedDetails_RoundTrip(t *testing.T) {
	paths := []string{"docs/", "internal/auth/token.go"}
	for _, note := range []string{"", "restored: self workspace close failed: herdr: refused"} {
		details := scopeReservedDetails(Reservation{Paths: paths}, note)
		got, ok := reservedPathsFromDetails(details)
		if !ok || !slices.Equal(got, paths) {
			t.Fatalf("reservedPathsFromDetails(%q) = %q, %v; want %q", details, got, ok, paths)
		}
	}
	if d := scopeReservedDetails(Reservation{Paths: paths}, ""); d != "paths=docs/,internal/auth/token.go" {
		t.Fatalf("Details = %q, want paths=docs/,internal/auth/token.go", d)
	}
	for _, bad := range []string{"", "paths=", "scope=x", "paths= x", "paths=/abs"} {
		if got, ok := reservedPathsFromDetails(bad); ok {
			t.Errorf("reservedPathsFromDetails(%q) = %q, true; want not readable", bad, got)
		}
	}
}

// reserveEvent builds a real EventScopeReserved for orgID holding paths.
func reserveEvent(orgID string, paths ...string) ManifestEvent {
	return scopeReservedEvent("", orgID, Reservation{Paths: paths}, "", false)
}

// TestRunningOrgs covers plan AC3: an org runs when, after its latest real
// disbanded, it has an active seat, a workspace created and not closed, or a
// reservation, and in no other case.
func TestRunningOrgs(t *testing.T) {
	seat := func(orgID, seatID, event string) ManifestEvent {
		return ManifestEvent{OrgID: orgID, SeatID: seatID, Event: event}
	}
	org := func(orgID, event, workspace string) ManifestEvent {
		return ManifestEvent{OrgID: orgID, Event: event, PaneID: workspace}
	}
	dry := func(ev ManifestEvent) ManifestEvent {
		ev.DryRun = true
		return ev
	}
	tests := []struct {
		name   string
		events []ManifestEvent
		want   []string
	}{
		{"active seat", []ManifestEvent{seat("x", "s1", EventSpawned)}, []string{"x"}},
		{"in-flight seat", []ManifestEvent{seat("x", "s1", EventSpawnStarted)}, []string{"x"}},
		{"rejected only", []ManifestEvent{seat("x", "s1", EventRejected), seat("x", "s2", EventRejected)}, nil},
		{"seat stopped, nothing open", []ManifestEvent{seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped)}, nil},
		{"spawn failed before any workspace", []ManifestEvent{seat("x", "s1", EventSpawnStarted), seat("x", "s1", EventSpawnFailed)}, nil},
		{"seats stopped, workspace still open", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped),
		}, []string{"x"}},
		{"workspace created then closed", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), org("x", EventOrgWorkspaceClosed, "ws"),
		}, nil},
		{"workspace event with no id", []ManifestEvent{org("x", EventOrgWorkspaceCreated, "")}, nil},
		{"reservation only", []ManifestEvent{reserveEvent("x", "internal/")}, []string{"x"}},
		{"reservation then disbanded", []ManifestEvent{reserveEvent("x", "internal/"), org("x", EventDisbanded, "")}, nil},
		{"cleanly disbanded", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), seat("x", "s1", EventSpawned), reserveEvent("x", "internal/"),
			seat("x", "s1", EventStopped), org("x", EventOrgWorkspaceClosed, "ws"), org("x", EventDisbanded, ""),
		}, nil},
		{"workspace an older ralph's disband left open", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), seat("x", "s1", EventSpawned), org("x", EventDisbanded, ""),
		}, nil},
		{"workspace reopened after disbanded", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), org("x", EventOrgWorkspaceClosed, "ws"), org("x", EventDisbanded, ""),
			org("x", EventOrgWorkspaceCreated, "ws"),
		}, []string{"x"}},
		{"rejected after disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned), org("x", EventDisbanded, ""), seat("x", "s2", EventRejected),
		}, nil},
		{"spawned again after disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned), org("x", EventDisbanded, ""), seat("x", "s1", EventSpawned),
		}, []string{"x"}},
		{"dry-run only", []ManifestEvent{
			dry(seat("x", "s1", EventSpawned)), dry(org("x", EventOrgWorkspaceCreated, "ws")), dry(reserveEvent("x", ".")),
		}, nil},
		{"a dry-run disbanded does not end a real life", []ManifestEvent{
			reserveEvent("x", "internal/"), dry(org("x", EventDisbanded, "")),
		}, []string{"x"}},
		{"several orgs come back sorted", []ManifestEvent{
			seat("org-c", "s1", EventSpawned), reserveEvent("org-a", "docs/"), seat("org-b", "s1", EventRejected),
			org("org-d", EventOrgWorkspaceCreated, "ws-d"),
		}, []string{"org-a", "org-c", "org-d"}},
		{"empty manifest", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RunningOrgs(tt.events)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("RunningOrgs = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestActiveReservation: the latest real reservation after the org's latest
// real disbanded counts; one before it, a dry-run one, or another org's does
// not; a record that cannot be read counts as the whole repo.
func TestActiveReservation(t *testing.T) {
	disbanded := ManifestEvent{OrgID: "x", Event: EventDisbanded}
	dryReserve := reserveEvent("x", "dry/")
	dryReserve.DryRun = true
	damaged := ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: "garbage"}
	tests := []struct {
		name   string
		events []ManifestEvent
		want   []string
	}{
		{"none", nil, nil},
		{"one", []ManifestEvent{reserveEvent("x", "docs/", "internal/")}, []string{"docs/", "internal/"}},
		{"the latest wins", []ManifestEvent{reserveEvent("x", "docs/"), reserveEvent("x", "internal/")}, []string{"internal/"}},
		{"released by disbanded", []ManifestEvent{reserveEvent("x", "docs/"), disbanded}, nil},
		{"reserved again after disbanded", []ManifestEvent{reserveEvent("x", "docs/"), disbanded, reserveEvent("x", "cmd/")}, []string{"cmd/"}},
		{"dry-run ignored", []ManifestEvent{dryReserve}, nil},
		{"another org's ignored", []ManifestEvent{reserveEvent("y", "docs/")}, nil},
		{"unreadable is the whole repo", []ManifestEvent{damaged}, []string{"."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActiveReservation(tt.events, "x"); !slices.Equal(got, tt.want) {
				t.Fatalf("ActiveReservation = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTotalActiveSeats(t *testing.T) {
	events := []ManifestEvent{
		{OrgID: "a", SeatID: "s1", Event: EventSpawned},
		{OrgID: "a", SeatID: "s2", Event: EventSpawnStarted},
		{OrgID: "a", SeatID: "s3", Event: EventSpawned},
		{OrgID: "a", SeatID: "s3", Event: EventStopped},
		{OrgID: "b", SeatID: "s1", Event: EventSpawned},
		{OrgID: "b", SeatID: "s2", Event: EventRejected},
		{OrgID: "c", SeatID: "s1", Event: EventSpawned},
		{OrgID: "c", Event: EventDisbanded},
		{OrgID: "d", SeatID: "s1", Event: EventSpawned, DryRun: true},
	}
	if got := TotalActiveSeats(events); got != 3 {
		t.Fatalf("TotalActiveSeats = %d, want 3 (a/s1, a/s2, b/s1)", got)
	}
}

// TestReservationDecision covers plan AC5 and AC7 at the decision level: the
// org's own reservation passes for the same set and refuses a different
// one; otherwise another running org's overlapping reservation refuses,
// naming that org and the paths; an org without a reservation, or one that
// was disbanded, blocks nothing.
func TestReservationDecision(t *testing.T) {
	events := []ManifestEvent{
		reserveEvent("org-a", "internal/auth/"),
		reserveEvent("org-old", "."),
		{OrgID: "org-old", Event: EventDisbanded},
		{OrgID: "org-free", SeatID: "s1", Event: EventSpawned},
	}
	tests := []struct {
		name       string
		orgID      string
		want       []string
		wantRecord bool
		wantError  []string
	}{
		{"the same set passes without a record", "org-a", []string{"internal/auth/"}, false, nil},
		{"a different set is refused", "org-a", []string{"docs/"}, false, []string{`org_id "org-a" already reserves internal/auth/`, "(docs/)", "ralph org disband --org-id org-a"}},
		{"a file under it overlaps", "org-b", []string{"internal/auth/token.go"}, false, []string{`running org_id "org-a"`, "internal/auth/token.go overlaps internal/auth/"}},
		{"a parent directory overlaps", "org-b", []string{"internal/"}, false, []string{`running org_id "org-a"`, "internal/ overlaps internal/auth/"}},
		{"the whole repo overlaps", "org-b", []string{"."}, false, []string{`running org_id "org-a"`, ". overlaps internal/auth/"}},
		{"a sibling with a shared prefix does not", "org-b", []string{"internal/authz/"}, true, nil},
		{"only the overlapping paths are named", "org-b", []string{"docs/", "internal/auth/x.go"}, false, []string{"internal/auth/x.go overlaps internal/auth/"}},
		{"an org without a reservation blocks nothing", "org-c", []string{"cmd/"}, true, nil},
		{"an org with seats but no reservation still checks the others", "org-free", []string{"."}, false, []string{`running org_id "org-a"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := reservationDecision(events, tt.orgID, Reservation{Paths: tt.want})
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
			if strings.Contains(err.Error(), "docs/ overlaps") {
				t.Errorf("error %q names a path that overlaps nothing", err)
			}
		})
	}
}

// TestReservationBeforeLastDisband: the reservation the org held right
// before its latest disbanded, not one from an earlier life, and nothing for
// an org that has no disbanded.
func TestReservationBeforeLastDisband(t *testing.T) {
	disbanded := ManifestEvent{OrgID: "x", Event: EventDisbanded}
	tests := []struct {
		name   string
		events []ManifestEvent
		want   []string
	}{
		{"never disbanded", []ManifestEvent{reserveEvent("x", "docs/")}, nil},
		{"held one", []ManifestEvent{reserveEvent("x", "docs/"), disbanded}, []string{"docs/"}},
		{"held none in its last life", []ManifestEvent{reserveEvent("x", "docs/"), disbanded, disbanded}, nil},
		{"held one in its last life", []ManifestEvent{reserveEvent("x", "docs/"), disbanded, reserveEvent("x", "cmd/"), disbanded}, []string{"cmd/"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reservationBeforeLastDisband(tt.events, "x"); !slices.Equal(got, tt.want) {
				t.Fatalf("reservationBeforeLastDisband = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- split plan feature bindings (plan 2026-10-09-org-feature-worktree) ---

// testFeatureDigest is the feature digest of every testFeature binding.
const testFeatureDigest = "fedcba987654"

// testFeature is a complete binding to feature slug of split plan split,
// approved with digest, on branch feat/<slug> in a worktree named after it,
// with the default approval mode per-plan and the feature digest
// testFeatureDigest (validateFeatureBinding requires a mode and a feature
// digest of every binding Spawn records).
func testFeature(split, slug, digest string) *FeatureBinding {
	return &FeatureBinding{
		Split: split, Feature: slug, Digest: digest,
		Branch: "feat/" + slug, Worktree: "/repo/.claude/worktrees/org-" + slug,
		Approval: config.ApprovalModePerPlan, FeatureDigest: testFeatureDigest,
	}
}

// boundReserveEvent is reserveEvent for a reservation bound to f.
func boundReserveEvent(orgID string, f *FeatureBinding, paths ...string) ManifestEvent {
	return scopeReservedEvent("", orgID, Reservation{Paths: paths, Feature: f}, "", false)
}

// TestScopeReservedDetails_FeatureRoundTrip covers the record of a bound
// reservation: the binding's tokens follow the paths in a fixed order, the
// worktree goes into the event's Worktree field, a note after them does not
// change what is read, and the paths still read back with
// reservedPathsFromDetails, which cuts at the first space as older binaries
// do. Records without binding tokens and worktree (an older binary's, with
// or without the compensation's note) are unbound.
func TestScopeReservedDetails_FeatureRoundTrip(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	paths := []string{"docs/", "internal/auth/"}
	const binding = "paths=docs/,internal/auth/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth approval=per-plan feature_digest=" + testFeatureDigest
	for _, note := range []string{"", "restored: self workspace close failed: herdr: refused"} {
		ev := scopeReservedEvent("", "org-a", Reservation{Paths: paths, Feature: f}, note, false)
		want := binding
		if note != "" {
			want += " " + note
		}
		if ev.Details != want || ev.Worktree != f.Worktree || ev.SeatID != "" || ev.Event != EventScopeReserved {
			t.Fatalf("note %q: event = %+v, want Details %q and Worktree %q", note, ev, want, f.Worktree)
		}
		got := reservationFromEvent(ev)
		if !slices.Equal(got.Paths, paths) || got.Feature == nil || *got.Feature != *f {
			t.Fatalf("note %q: reservationFromEvent = %+v (feature %+v), want %q bound to %+v", note, got, got.Feature, paths, f)
		}
		if old, ok := reservedPathsFromDetails(ev.Details); !ok || !slices.Equal(old, paths) {
			t.Fatalf("note %q: reservedPathsFromDetails(%q) = %q, %v; want %q", note, ev.Details, old, ok, paths)
		}
	}
	split := *f
	split.Approval = config.ApprovalModeSplit
	ev := scopeReservedEvent("", "org-a", Reservation{Paths: paths, Feature: &split}, "", false)
	if want := strings.Replace(binding, "approval=per-plan", "approval=split", 1); ev.Details != want {
		t.Fatalf("split binding: Details = %q, want %q", ev.Details, want)
	}
	if got := reservationFromEvent(ev).Feature; got == nil || *got != split || got.RequiresApproval() {
		t.Fatalf("split binding: read back %+v, want %+v needing no approval", got, split)
	}
	for _, details := range []string{
		"paths=docs/,internal/auth/",
		"paths=docs/,internal/auth/ restored: self pane close failed: herdr: refused",
		"paths=docs/,internal/auth/ restored: split=split-1 feature=auth digest=0123456789ab branch=feat/auth",
	} {
		got := reservationFromEvent(ManifestEvent{OrgID: "org-a", Event: EventScopeReserved, Details: details})
		if !slices.Equal(got.Paths, paths) || got.Feature != nil {
			t.Errorf("reservationFromEvent(%q) = %q bound to %+v, want %q unbound", details, got.Paths, got.Feature, paths)
		}
	}
}

// oldBindingTokens is the binding as an older binary that knows only keys
// reads it (its featureBindingFromTokens): those tokens up to the first
// other token, the rest being the note.
func oldBindingTokens(details string, keys ...string) (read map[string]string, note string) {
	read = make(map[string]string)
	fields := strings.Fields(details)[1:]
	for i, token := range fields {
		key, value, ok := strings.Cut(token, "=")
		if !ok || !slices.Contains(keys, key) {
			return read, strings.Join(fields[i:], " ")
		}
		read[key] = value
	}
	return read, ""
}

// The keys a binary from before the approval mode reads, and those one from
// before the feature digest reads.
var (
	keysBeforeApprovalMode  = []string{"split", "feature", "digest", "branch"}
	keysBeforeFeatureDigest = []string{"split", "feature", "digest", "branch", "approval"}
)

// TestScopeReservedDetails_ApprovalTokenAfterBranch: the approval token is
// written after the branch token and before the feature digest's token and
// the note, so a binary from before the approval mode, which stops at the
// first key it does not know, still reads the split, feature, digest and
// branch of a new record, and takes `approval=...` and what follows it as
// the note. A binding with no mode (a record read back from an older start)
// writes no approval token.
func TestScopeReservedDetails_ApprovalTokenAfterBranch(t *testing.T) {
	const note = "restored: self pane close failed: herdr: refused"
	for _, mode := range []string{config.ApprovalModePerPlan, config.ApprovalModeSplit} {
		f := testFeature("split-1", "auth", "0123456789ab")
		f.Approval = mode
		details := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: f}, note)
		branchAt, approvalAt, noteAt := strings.Index(details, " branch="), strings.Index(details, " approval="+mode), strings.Index(details, " "+note)
		if branchAt < 0 || approvalAt <= branchAt || noteAt <= approvalAt {
			t.Fatalf("%s: Details %q, want branch=, then approval=%s, then the note", mode, details, mode)
		}
		read, oldNote := oldBindingTokens(details, keysBeforeApprovalMode...)
		want := map[string]string{"split": "split-1", "feature": "auth", "digest": "0123456789ab", "branch": "feat/auth"}
		wantNote := "approval=" + mode + " feature_digest=" + testFeatureDigest + " " + note
		if !maps.Equal(read, want) || oldNote != wantNote {
			t.Fatalf("%s: an older binary reads %v with note %q, want %v with note %q", mode, read, oldNote, want, wantNote)
		}
	}
	legacy := testFeature("split-1", "auth", "0123456789ab")
	legacy.Approval = ""
	if details := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: legacy}, ""); strings.Contains(details, "approval") {
		t.Fatalf("a binding with no mode wrote %q, want no approval token", details)
	}
}

// TestScopeReservedDetails_FeatureDigestTokenLast covers AC2 of plan
// 2026-11-12-split-feature-digest: the feature digest's token is the last
// binding token, after approval= or, for a binding with no mode, after
// branch=, and before the note, which still reads back as the note (the
// binding is read the same with and without it). A binding with no feature
// digest writes no token, and either way the binding reads back as written.
func TestScopeReservedDetails_FeatureDigestTokenLast(t *testing.T) {
	const note = "restored: self pane close failed: herdr: refused"
	const token = " feature_digest=" + testFeatureDigest
	for _, tc := range []struct {
		name, approval, after string
	}{
		{"after approval=", config.ApprovalModePerPlan, " approval=per-plan"},
		{"after branch= with no mode", "", " branch=feat/auth"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := testFeature("split-1", "auth", "0123456789ab")
			f.Approval = tc.approval
			for _, n := range []string{"", note} {
				ev := scopeReservedEvent("", "org-a", Reservation{Paths: []string{"docs/"}, Feature: f}, n, false)
				want := "paths=docs/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth"
				if tc.approval != "" {
					want += " approval=" + tc.approval
				}
				want += token
				if n != "" {
					want += " " + n
				}
				if ev.Details != want || !strings.Contains(ev.Details, tc.after+token) {
					t.Fatalf("note %q: Details = %q, want %q (the token right after %q)", n, ev.Details, want, tc.after)
				}
				if got := reservationFromEvent(ev).Feature; got == nil || *got != *f || !got.Complete() {
					t.Fatalf("note %q: read back %+v, want %+v, complete", n, got, f)
				}
			}
		})
	}
	none := testFeature("split-1", "auth", "0123456789ab")
	none.FeatureDigest = ""
	ev := scopeReservedEvent("", "org-a", Reservation{Paths: []string{"docs/"}, Feature: none}, note, false)
	if want := "paths=docs/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth approval=per-plan " + note; ev.Details != want {
		t.Fatalf("a binding with no feature digest wrote %q, want %q", ev.Details, want)
	}
	if got := reservationFromEvent(ev).Feature; got == nil || *got != *none || !got.Complete() {
		t.Fatalf("read back %+v, want %+v, complete", got, none)
	}
}

// TestScopeReservedDetails_OlderBinaryReadsFiveTokens covers AC8 of plan
// 2026-11-12-split-feature-digest: a binary from before the feature digest,
// which reads split, feature, digest, branch and approval and stops at the
// first other key, reads a new record's five tokens as before and takes
// `feature_digest=...` and the note after it as the note.
func TestScopeReservedDetails_OlderBinaryReadsFiveTokens(t *testing.T) {
	for _, n := range []string{"", "restored: self pane close failed: herdr: refused"} {
		for _, mode := range []string{config.ApprovalModePerPlan, config.ApprovalModeSplit} {
			f := testFeature("split-1", "auth", "0123456789ab")
			f.Approval = mode
			details := scopeReservedDetails(Reservation{Paths: []string{"docs/"}, Feature: f}, n)
			read, oldNote := oldBindingTokens(details, keysBeforeFeatureDigest...)
			want := map[string]string{"split": "split-1", "feature": "auth", "digest": "0123456789ab", "branch": "feat/auth", "approval": mode}
			wantNote := strings.TrimSpace("feature_digest=" + testFeatureDigest + " " + n)
			if !maps.Equal(read, want) || oldNote != wantNote {
				t.Fatalf("%s, note %q: an older binary reads %v with note %q from %q, want %v with note %q", mode, n, read, oldNote, details, want, wantNote)
			}
		}
	}
}

// TestFeatureBinding_FeatureDigestRead covers AC2 and D6 of plan
// 2026-11-12-split-feature-digest: the feature digest reads back as written;
// no token, the token given twice (the same value or not), an empty value,
// and a token after the note all read as "", with no sentinel. In every case
// the binding stays complete (D4) and keeps its approval mode.
func TestFeatureBinding_FeatureDigestRead(t *testing.T) {
	const wt = "/repo/.claude/worktrees/org-auth"
	const base = "paths=docs/ split=s feature=auth digest=0123456789ab branch=feat/auth approval=split"
	tests := []struct {
		name, details, want string
	}{
		{"no token (an older start)", base, ""},
		{"the token", base + " feature_digest=" + testFeatureDigest, testFeatureDigest},
		{"the token and a note", base + " feature_digest=" + testFeatureDigest + " restored: why", testFeatureDigest},
		{"right after branch=", "paths=docs/ split=s feature=auth digest=0123456789ab branch=feat/auth feature_digest=" + testFeatureDigest + " approval=split", testFeatureDigest},
		{"given twice, the same", base + " feature_digest=" + testFeatureDigest + " feature_digest=" + testFeatureDigest, ""},
		{"given twice, different", base + " feature_digest=" + testFeatureDigest + " feature_digest=0a1b2c3d4e5f", ""},
		{"an empty value", base + " feature_digest=", ""},
		{"after the note", base + " restored: feature_digest=" + testFeatureDigest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: tt.details, Worktree: wt}).Feature
			if got == nil {
				t.Fatal("expected a binding, got none")
			}
			if got.FeatureDigest != tt.want {
				t.Fatalf("FeatureDigest = %q, want %q", got.FeatureDigest, tt.want)
			}
			if !got.Complete() || got.Approval != config.ApprovalModeSplit || !sameFeature(got, got) {
				t.Fatalf("expected %+v to be complete, approval split, and equal to itself", got)
			}
			if tt.want == "" && !strings.Contains(got.String(), `(digest "0123456789ab", no feature digest, branch "feat/auth"`) {
				t.Fatalf("String() = %q, want it to say there is no feature digest", got.String())
			}
		})
	}
}

// TestFeatureBinding_OlderRecordStillBound covers D4 of plan
// 2026-11-12-split-feature-digest: a scope_reserved an older start wrote,
// with no feature_digest= token, still reads as a complete binding, so its
// org is still a gate's feature org (gateOrgLife) and its branch a
// merge-order target (collectMergeTargets).
func TestFeatureBinding_OlderRecordStillBound(t *testing.T) {
	old := testFeature("split-1", "auth", "0123456789ab")
	old.FeatureDigest = ""
	events := []ManifestEvent{boundReserveEvent("org-a", old, "internal/auth/")}
	if strings.Contains(events[0].Details, "feature_digest") {
		t.Fatalf("expected a record with no feature digest, got %q", events[0].Details)
	}
	got := ActiveFeature(events, "org-a")
	if got == nil || !got.Complete() || *got != *old {
		t.Fatalf("ActiveFeature = %+v, want %+v, complete", got, old)
	}
	if life, refused := gateOrgLife(events, "org-a"); refused != "" || life.binding == nil || *life.binding != *old {
		t.Fatalf("gateOrgLife = %+v, %q; want the binding %+v", life, refused, old)
	}
	targets, unattached := collectMergeTargets(mergeLedger{events: events})
	if len(targets) != 1 || unattached != 0 || targets[0].branch != "feat/auth" || targets[0].binding == nil ||
		targets[0].binding.OrgID != "org-a" || targets[0].binding.Feature != "auth" {
		t.Fatalf("collectMergeTargets = %d targets (%d unattached), want one for feat/auth bound to org-a", len(targets), unattached)
	}
}

// TestFeatureBinding_ApprovalModeRead covers the approval mode a record
// reads back as, and what RequiresApproval says of it: a record from before
// the approval mode (no token) reads as "" and is complete and needs no
// approval (read as split, also in String); per-plan needs it and split does
// not; the token given twice, given with an empty value, or with a value
// that is neither mode reads as a mode that needs the approval, while the
// binding stays complete. A token after the note is not read.
func TestFeatureBinding_ApprovalModeRead(t *testing.T) {
	const wt = "/repo/.claude/worktrees/org-auth"
	const base = "paths=docs/ split=s feature=auth digest=d branch=feat/auth"
	tests := []struct {
		name, details    string
		wantApproval     string
		requiresApproval bool
	}{
		{"no token (an older start)", base, "", false},
		{"per-plan", base + " approval=per-plan", config.ApprovalModePerPlan, true},
		{"split", base + " approval=split", config.ApprovalModeSplit, false},
		{"split with a note", base + " approval=split restored: why", config.ApprovalModeSplit, false},
		{"given twice, the same", base + " approval=split approval=split", approvalModeUnreadable, true},
		{"given twice, different", base + " approval=split approval=per-plan", approvalModeUnreadable, true},
		{"an empty value", base + " approval=", approvalModeUnreadable, true},
		{"an unknown value", base + " approval=auto", "auto", true},
		{"another case", base + " approval=Split", "Split", true},
		{"before the other tokens", "paths=docs/ approval=split split=s feature=auth digest=d branch=feat/auth", config.ApprovalModeSplit, false},
		{"after the note", base + " restored: approval=per-plan", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: tt.details, Worktree: wt}).Feature
			if got == nil {
				t.Fatal("expected a binding, got none")
			}
			if got.Approval != tt.wantApproval {
				t.Fatalf("Approval = %q, want %q", got.Approval, tt.wantApproval)
			}
			if got.RequiresApproval() != tt.requiresApproval {
				t.Fatalf("RequiresApproval() = %v, want %v", got.RequiresApproval(), tt.requiresApproval)
			}
			if !got.Complete() || !sameFeature(got, got) {
				t.Fatalf("expected %+v to be complete and equal to itself", got)
			}
			if tt.wantApproval == "" && !strings.Contains(got.String(), "no approval mode, read as split") {
				t.Fatalf("String() = %q, want it to say the mode reads as split", got.String())
			}
		})
	}
	if approvalModeUnreadable == config.ApprovalModePerPlan || approvalModeUnreadable == config.ApprovalModeSplit || approvalModeUnreadable == "" ||
		strings.ContainsAny(approvalModeUnreadable, " \t=,") {
		t.Fatalf("approvalModeUnreadable = %q must be neither mode, non-empty, and fit a Details token", approvalModeUnreadable)
	}
}

// TestReservationFromEvent_DamagedBindingMatchesNothing: a record with some
// binding fields but not all of them (a missing or empty key, a key given
// twice, no worktree, or a worktree with no keys) is read as a binding that
// sameFeature takes as equal to nothing, not as an unbound reservation.
// Tokens after the first one that is not a binding key are not read.
func TestReservationFromEvent_DamagedBindingMatchesNothing(t *testing.T) {
	const wt = "/repo/.claude/worktrees/org-auth"
	tests := []struct {
		name, details, worktree string
		want                    FeatureBinding
		wantPaths               []string
	}{
		{"a missing key", "paths=docs/ split=s feature=auth digest=d", wt,
			FeatureBinding{Split: "s", Feature: "auth", Digest: "d", Worktree: wt}, []string{"docs/"}},
		{"an empty value", "paths=docs/ split= feature=auth digest=d branch=feat/auth", wt,
			FeatureBinding{Feature: "auth", Digest: "d", Branch: "feat/auth", Worktree: wt}, []string{"docs/"}},
		{"a key given twice", "paths=docs/ split=s split=t feature=auth digest=d branch=feat/auth", wt,
			FeatureBinding{Feature: "auth", Digest: "d", Branch: "feat/auth", Worktree: wt}, []string{"docs/"}},
		{"keys but no worktree", "paths=docs/ split=s feature=auth digest=d branch=feat/auth", "",
			FeatureBinding{Split: "s", Feature: "auth", Digest: "d", Branch: "feat/auth"}, []string{"docs/"}},
		{"a worktree but no keys", "paths=docs/", wt,
			FeatureBinding{Worktree: wt}, []string{"docs/"}},
		{"reading stops at the note", "paths=docs/ split=s feature=auth restored: digest=d branch=feat/auth", wt,
			FeatureBinding{Split: "s", Feature: "auth", Worktree: wt}, []string{"docs/"}},
		{"unreadable paths with a partial binding", "paths=/abs split=s", "",
			FeatureBinding{Split: "s"}, []string{"."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reservationFromEvent(ManifestEvent{OrgID: "x", Event: EventScopeReserved, Details: tt.details, Worktree: tt.worktree})
			if !slices.Equal(got.Paths, tt.wantPaths) {
				t.Fatalf("paths = %q, want %q", got.Paths, tt.wantPaths)
			}
			if got.Feature == nil || *got.Feature != tt.want {
				t.Fatalf("feature = %+v, want %+v", got.Feature, tt.want)
			}
			if got.Feature.Complete() || sameFeature(got.Feature, got.Feature) || sameFeature(got.Feature, nil) {
				t.Fatalf("expected %+v to be incomplete and equal to nothing", got.Feature)
			}
			if !strings.HasSuffix(got.Feature.String(), " read from an incomplete record") {
				t.Fatalf("String() = %q, want it to say the record is incomplete", got.Feature.String())
			}
		})
	}
}

// TestSameFeature covers AC3 and D5 of plan 2026-11-12-split-feature-digest:
// two bindings are the same only when both are nil or both are complete with
// the same split plan, feature, branch, worktree and approval mode (a record
// with no mode differs from one with per-plan or split), and the same
// feature digest when both have one, the whole digest then not compared, or
// else (either has none) the same whole digest. An incomplete binding equals
// nothing, not even itself, also when its feature digest is the other's.
func TestSameFeature(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	g := *f
	if !sameFeature(nil, nil) || !sameFeature(f, &g) {
		t.Fatal("expected nil/nil and two equal complete bindings to be the same")
	}
	if sameFeature(f, nil) || sameFeature(nil, f) {
		t.Fatal("expected a binding and none to differ")
	}
	// Each change keeps the feature digest of f, so these differ even though
	// their feature digests are equal.
	for name, change := range map[string]func(*FeatureBinding){
		"split":    func(b *FeatureBinding) { b.Split = "split-2" },
		"feature":  func(b *FeatureBinding) { b.Feature = "api" },
		"branch":   func(b *FeatureBinding) { b.Branch = "fix/auth" },
		"worktree": func(b *FeatureBinding) { b.Worktree = "/elsewhere" },
		"approval": func(b *FeatureBinding) { b.Approval = config.ApprovalModeSplit },
		"no mode":  func(b *FeatureBinding) { b.Approval = "" },
	} {
		for _, digest := range []string{f.Digest, "ba9876543210"} {
			other := *f
			change(&other)
			other.Digest = digest
			if sameFeature(f, &other) || sameFeature(&other, f) {
				t.Errorf("expected bindings with another %s (digest %s, the same feature digest) to differ", name, digest)
			}
		}
	}

	const d1, d2, fd1, fd2 = "0123456789ab", "ba9876543210", testFeatureDigest, "0a1b2c3d4e5f"
	withDigests := func(digest, featureDigest string) *FeatureBinding {
		b := testFeature("split-1", "auth", digest)
		b.FeatureDigest = featureDigest
		return b
	}
	for _, tc := range []struct {
		name string
		a, b *FeatureBinding
		want bool
	}{
		{"both have the same feature digest, another whole digest", withDigests(d1, fd1), withDigests(d2, fd1), true},
		{"both have the same feature digest and whole digest", withDigests(d1, fd1), withDigests(d1, fd1), true},
		{"both have another feature digest, the same whole digest", withDigests(d1, fd1), withDigests(d1, fd2), false},
		{"both have another feature digest and whole digest", withDigests(d1, fd1), withDigests(d2, fd2), false},
		{"one has a feature digest, the same whole digest", withDigests(d1, fd1), withDigests(d1, ""), true},
		{"one has a feature digest, another whole digest", withDigests(d1, fd1), withDigests(d2, ""), false},
		{"neither has a feature digest, the same whole digest", withDigests(d1, ""), withDigests(d1, ""), true},
		{"neither has a feature digest, another whole digest", withDigests(d1, ""), withDigests(d2, ""), false},
	} {
		if got, back := sameFeature(tc.a, tc.b), sameFeature(tc.b, tc.a); got != tc.want || back != tc.want {
			t.Errorf("%s: sameFeature = %v, %v the other way; want %v", tc.name, got, back, tc.want)
		}
	}

	for name, change := range map[string]func(*FeatureBinding){
		"no split":    func(b *FeatureBinding) { b.Split = "" },
		"no feature":  func(b *FeatureBinding) { b.Feature = "" },
		"no digest":   func(b *FeatureBinding) { b.Digest = "" },
		"no branch":   func(b *FeatureBinding) { b.Branch = "" },
		"no worktree": func(b *FeatureBinding) { b.Worktree = "" },
	} {
		incomplete := *f
		change(&incomplete)
		same := incomplete
		if sameFeature(&incomplete, &same) || sameFeature(f, &incomplete) || sameFeature(&incomplete, f) {
			t.Errorf("expected a binding with %s and the feature digest of f to equal nothing", name)
		}
	}
}

// TestValidateFeatureBinding: every field is required, split, feature,
// digest and branch cannot hold what the Details tokens cannot (whitespace,
// `=`, `,`, control characters), the worktree must be absolute, the
// approval mode must be per-plan or split (no mode, as an older record
// reads, approvalModeUnreadable and any other value are refused), and the
// feature digest must be 12 lowercase hex digits (D3 of plan
// 2026-11-12-split-feature-digest: none, as an older record reads, is
// refused).
func TestValidateFeatureBinding(t *testing.T) {
	for _, mode := range []string{config.ApprovalModePerPlan, config.ApprovalModeSplit} {
		f := testFeature("split-1", "auth", "0123456789ab")
		f.Approval = mode
		if err := validateFeatureBinding(*f); err != nil {
			t.Fatalf("validateFeatureBinding(complete binding, approval %s) = %v, want nil", mode, err)
		}
	}
	tests := []struct {
		name   string
		change func(*FeatureBinding)
		want   string
	}{
		{"empty split", func(b *FeatureBinding) { b.Split = "" }, "feature's split is empty"},
		{"empty feature", func(b *FeatureBinding) { b.Feature = "" }, "feature's feature is empty"},
		{"empty digest", func(b *FeatureBinding) { b.Digest = "" }, "feature's digest is empty"},
		{"empty branch", func(b *FeatureBinding) { b.Branch = "" }, "feature's branch is empty"},
		{"space in split", func(b *FeatureBinding) { b.Split = "split 1" }, `split "split 1" has whitespace`},
		{"tab in digest", func(b *FeatureBinding) { b.Digest = "01\t23" }, "digest"},
		{"equals in branch", func(b *FeatureBinding) { b.Branch = "feat/a=b" }, `branch "feat/a=b" has whitespace, a comma, an =`},
		{"comma in feature", func(b *FeatureBinding) { b.Feature = "a,b" }, `feature "a,b" has whitespace, a comma`},
		{"control character in split", func(b *FeatureBinding) { b.Split = "a\x01b" }, "control character"},
		{"empty worktree", func(b *FeatureBinding) { b.Worktree = "" }, `worktree "" is not an absolute path`},
		{"relative worktree", func(b *FeatureBinding) { b.Worktree = ".claude/worktrees/org-auth" }, "is not an absolute path"},
		{"no approval mode", func(b *FeatureBinding) { b.Approval = "" }, `feature's approval mode "" is not "per-plan" or "split"`},
		{"unknown approval mode", func(b *FeatureBinding) { b.Approval = "auto" }, `approval mode "auto" is not "per-plan" or "split"`},
		{"unreadable approval mode", func(b *FeatureBinding) { b.Approval = approvalModeUnreadable }, `approval mode "unreadable"`},
		{"approval mode in another case", func(b *FeatureBinding) { b.Approval = "Per-Plan" }, `approval mode "Per-Plan"`},
		{"no feature digest", func(b *FeatureBinding) { b.FeatureDigest = "" }, `feature's feature_digest "" is not 12 lowercase hex digits`},
		{"a short feature digest", func(b *FeatureBinding) { b.FeatureDigest = "fedcba98765" }, `feature_digest "fedcba98765" is not 12 lowercase hex digits`},
		{"a long feature digest", func(b *FeatureBinding) { b.FeatureDigest = "fedcba9876543" }, `feature_digest "fedcba9876543"`},
		{"an uppercase feature digest", func(b *FeatureBinding) { b.FeatureDigest = "FEDCBA987654" }, `feature_digest "FEDCBA987654"`},
		{"a feature digest that is not hex", func(b *FeatureBinding) { b.FeatureDigest = "fedcba98765g" }, `feature_digest "fedcba98765g"`},
		{"a space in the feature digest", func(b *FeatureBinding) { b.FeatureDigest = "fedcba 87654" }, `feature_digest "fedcba 87654"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testFeature("split-1", "auth", "0123456789ab")
			tt.change(f)
			if err := validateFeatureBinding(*f); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateFeatureBinding(%+v) = %v, want an error containing %q", f, err, tt.want)
			}
		})
	}
}

// TestReservationDecision_Feature covers plan AC6 and AC7 at the decision
// level, and AC4 to AC6 of plan 2026-11-12-split-feature-digest. An org
// bound to a feature passes only the same paths with the same binding (with
// another whole digest when the feature digest is the same), and refuses
// another feature, feature digest, branch, worktree or approval mode (an org
// bound before the approval mode existed refuses one with a mode), other
// paths, and a reservation without a binding even for the same paths; a
// refusal for the same feature with only another digest says which digest
// changed, and an org whose record has no feature digest refuses another
// whole digest, saying so. An org with a plain reservation refuses a
// binding. An org that is running without a reservation (seats of a
// promoted session's leader, a leader from `ralph org start <task>`, an open
// workspace) refuses a binding but still takes a plain reservation. A new or
// disbanded org takes a binding unless its paths overlap another running
// org's, and an org whose record is damaged refuses everything.
func TestReservationDecision_Feature(t *testing.T) {
	fAuth := testFeature("split-1", "auth", "0123456789ab")
	withField := func(change func(*FeatureBinding)) *FeatureBinding {
		f := *fAuth
		change(&f)
		return &f
	}
	fOld := withField(func(f *FeatureBinding) { f.FeatureDigest = "" })
	events := []ManifestEvent{
		boundReserveEvent("org-bound", fAuth, "internal/auth/"),
		reserveEvent("org-plain", "docs/"),
		{OrgID: "org-promoted", SeatID: "seat-1", Event: EventSpawned},
		{OrgID: "org-task", SeatID: LeaderIdentity, Event: EventSpawned},
		{OrgID: "org-ws", Event: EventOrgWorkspaceCreated, PaneID: "ws-1"},
		boundReserveEvent("org-gone", fAuth, "cmd/"),
		{OrgID: "org-gone", Event: EventDisbanded},
		{OrgID: "org-damaged", Event: EventScopeReserved, Details: "paths=damaged/ split=split-1 feature=auth", Worktree: fAuth.Worktree},
		{OrgID: "org-legacy", Event: EventScopeReserved, Details: "paths=legacy/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth", Worktree: fAuth.Worktree},
		boundReserveEvent("org-old", fOld, "old/"),
	}
	const boundTo = `org_id "org-bound" is bound to feature "auth" of split plan "split-1" (digest "0123456789ab", feature digest "fedcba987654", ` +
		`branch "feat/auth", worktree "/repo/.claude/worktrees/org-auth", approval "per-plan") and reserves internal/auth/`
	const disbandBound = "ralph org disband --org-id org-bound first, or use another org_id"
	const sectionChanged = `is refused, since the feature's section (### auth) or the preamble of split plan "split-1" changed after org_id "org-bound" was started ` +
		`(feature digest "fedcba987654", now "0a1b2c3d4e5f"): ` + disbandBound
	const olderRecord = `is refused, and since its record has no feature digest (an older ralph started it), every change to the split plan is refused; ` +
		"disbanding the org and starting it again records one: ralph org disband --org-id org-old first, or use another org_id"
	noDigestClause := []string{"since the feature's section", "older ralph"}
	tests := []struct {
		name       string
		orgID      string
		want       Reservation
		wantRecord bool
		wantError  []string
		wantNot    []string // in no error of a refusal
	}{
		{"bound: the same paths and binding pass without a record", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(*FeatureBinding) {})}, false, nil, nil},
		{"bound: only another whole digest passes without a record", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Digest = "ba9876543210" })}, false, nil, nil},
		{"bound: another feature is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Feature = "api" })}, false,
			[]string{boundTo, `so the reservation internal/auth/ for feature "api" of split plan "split-1"`, disbandBound}, noDigestClause},
		{"bound: another feature digest is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.FeatureDigest = "0a1b2c3d4e5f" })}, false,
			[]string{boundTo, `(digest "0123456789ab", feature digest "0a1b2c3d4e5f"`, sectionChanged}, nil},
		{"bound: another feature digest and whole digest are refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Digest, f.FeatureDigest = "ba9876543210", "0a1b2c3d4e5f" })}, false,
			[]string{boundTo, `(digest "ba9876543210", feature digest "0a1b2c3d4e5f"`, sectionChanged}, nil},
		{"an older record with no feature digest: another whole digest is refused", "org-old",
			Reservation{Paths: []string{"old/"}, Feature: withField(func(f *FeatureBinding) { f.Digest = "ba9876543210" })}, false,
			[]string{`org_id "org-old" is bound to feature "auth" of split plan "split-1" (digest "0123456789ab", no feature digest, branch "feat/auth"`,
				`(digest "ba9876543210", feature digest "fedcba987654"`, olderRecord}, nil},
		{"an older record with no feature digest: the same whole digest passes without a record", "org-old",
			Reservation{Paths: []string{"old/"}, Feature: withField(func(*FeatureBinding) {})}, false, nil, nil},
		{"an older record with no feature digest: another approval mode is refused as before", "org-old",
			Reservation{Paths: []string{"old/"}, Feature: withField(func(f *FeatureBinding) { f.Digest, f.Approval = "ba9876543210", config.ApprovalModeSplit })}, false,
			[]string{`org_id "org-old" is bound to`, `approval "split")`}, noDigestClause},
		{"bound: another split plan is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Split = "split-2" })}, false,
			[]string{boundTo, `split plan "split-2"`}, noDigestClause},
		{"bound: another branch is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Branch = "fix/auth" })}, false,
			[]string{boundTo}, noDigestClause},
		{"bound: another worktree is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Worktree = "/elsewhere" })}, false,
			[]string{boundTo}, noDigestClause},
		{"bound: another approval mode is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Approval = config.ApprovalModeSplit })}, false,
			[]string{boundTo, `worktree "/repo/.claude/worktrees/org-auth", approval "split")`, disbandBound}, noDigestClause},
		{"bound before the approval mode: a mode is refused", "org-legacy",
			Reservation{Paths: []string{"legacy/"}, Feature: withField(func(*FeatureBinding) {})}, false,
			[]string{`org_id "org-legacy" is bound to feature "auth" of split plan "split-1"`, "no approval mode, read as split) and reserves legacy/",
				`approval "per-plan")`}, noDigestClause},
		{"bound before the approval mode: the same binding with no mode passes", "org-legacy",
			Reservation{Paths: []string{"legacy/"}, Feature: withField(func(f *FeatureBinding) { f.Approval = "" })}, false, nil, nil},
		{"bound: the same binding with other paths is refused", "org-bound",
			Reservation{Paths: []string{"internal/"}, Feature: withField(func(*FeatureBinding) {})}, false,
			[]string{boundTo, "so the reservation internal/ for"}, noDigestClause},
		{"bound: the same paths without a binding are refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}}, false,
			[]string{boundTo, "so a reservation without a split plan (internal/auth/) is refused", disbandBound}, noDigestClause},
		{"plain: a binding for the same paths is refused", "org-plain",
			Reservation{Paths: []string{"docs/"}, Feature: fAuth}, false,
			[]string{`org_id "org-plain" was started without a split plan (it reserves docs/ with a plain --reserve)`,
				`so the reservation docs/ for feature "auth"`, "use another org_id, or ralph org disband --org-id org-plain first"}, nil},
		{"promoted leader's org: a binding is refused", "org-promoted",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, false,
			[]string{`org_id "org-promoted" is running without a split plan`, `feature "auth"`, "ralph org disband --org-id org-promoted first"}, nil},
		{"ralph org start <task>'s org: a binding is refused", "org-task",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, false,
			[]string{`org_id "org-task" is running without a split plan`}, nil},
		{"an org with only an open workspace: a binding is refused", "org-ws",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, false,
			[]string{`org_id "org-ws" is running without a split plan`}, nil},
		{"promoted leader's org: a plain reservation is still taken", "org-promoted",
			Reservation{Paths: []string{"api/"}}, true, nil, nil},
		{"a new org takes a binding", "org-new",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, true, nil, nil},
		{"a disbanded bound org takes another binding", "org-gone",
			Reservation{Paths: []string{"cmd/"}, Feature: withField(func(f *FeatureBinding) { f.Digest, f.FeatureDigest = "ba9876543210", "0a1b2c3d4e5f" })}, true, nil, nil},
		{"a binding overlapping another running org is refused", "org-new",
			Reservation{Paths: []string{"internal/"}, Feature: fAuth}, false,
			[]string{`running org_id "org-bound": internal/ overlaps internal/auth/`}, nil},
		{"damaged: the binding as read is refused", "org-damaged",
			Reservation{Paths: []string{"damaged/"}, Feature: &FeatureBinding{Split: "split-1", Feature: "auth", Worktree: fAuth.Worktree}}, false,
			[]string{`org_id "org-damaged" is bound to feature "auth" of split plan "split-1"`, "read from an incomplete record"}, noDigestClause},
		{"damaged: the same paths without a binding are refused", "org-damaged",
			Reservation{Paths: []string{"damaged/"}}, false,
			[]string{`org_id "org-damaged" is bound to`, "so a reservation without a split plan (damaged/) is refused"}, noDigestClause},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := reservationDecision(events, tt.orgID, tt.want)
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
			for _, w := range tt.wantNot {
				if strings.Contains(err.Error(), w) {
					t.Errorf("error %q contains %q", err, w)
				}
			}
		})
	}
}

// TestActiveFeature: the binding of the org's current reservation; none for a
// plain reservation, after a disbanded, or for another org.
func TestActiveFeature(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	disbanded := ManifestEvent{OrgID: "x", Event: EventDisbanded}
	tests := []struct {
		name   string
		events []ManifestEvent
		want   *FeatureBinding
	}{
		{"none", nil, nil},
		{"plain", []ManifestEvent{reserveEvent("x", "docs/")}, nil},
		{"bound", []ManifestEvent{boundReserveEvent("x", f, "docs/")}, f},
		{"released by disbanded", []ManifestEvent{boundReserveEvent("x", f, "docs/"), disbanded}, nil},
		{"a plain run after a bound one", []ManifestEvent{boundReserveEvent("x", f, "docs/"), disbanded, reserveEvent("x", "docs/")}, nil},
		{"another org's", []ManifestEvent{boundReserveEvent("y", f, "docs/")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ActiveFeature(tt.events, "x")
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Fatalf("ActiveFeature = %+v, want %+v", got, tt.want)
			}
		})
	}
}
