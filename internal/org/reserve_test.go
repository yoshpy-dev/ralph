package org

import (
	"slices"
	"strings"
	"testing"
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

// testFeature is a complete binding to feature slug of split plan split,
// approved with digest, on branch feat/<slug> in a worktree named after it.
func testFeature(split, slug, digest string) *FeatureBinding {
	return &FeatureBinding{
		Split: split, Feature: slug, Digest: digest,
		Branch: "feat/" + slug, Worktree: "/repo/.claude/worktrees/org-" + slug,
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
	const binding = "paths=docs/,internal/auth/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth"
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
			if got.Feature.complete() || sameFeature(got.Feature, got.Feature) || sameFeature(got.Feature, nil) {
				t.Fatalf("expected %+v to be incomplete and equal to nothing", got.Feature)
			}
			if !strings.HasSuffix(got.Feature.String(), " read from an incomplete record") {
				t.Fatalf("String() = %q, want it to say the record is incomplete", got.Feature.String())
			}
		})
	}
}

// TestSameFeature: two bindings are the same only when both are nil or both
// are complete with all five fields equal.
func TestSameFeature(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	g := *f
	if !sameFeature(nil, nil) || !sameFeature(f, &g) {
		t.Fatal("expected nil/nil and two equal complete bindings to be the same")
	}
	if sameFeature(f, nil) || sameFeature(nil, f) {
		t.Fatal("expected a binding and none to differ")
	}
	for name, change := range map[string]func(*FeatureBinding){
		"split":    func(b *FeatureBinding) { b.Split = "split-2" },
		"feature":  func(b *FeatureBinding) { b.Feature = "api" },
		"digest":   func(b *FeatureBinding) { b.Digest = "ba9876543210" },
		"branch":   func(b *FeatureBinding) { b.Branch = "fix/auth" },
		"worktree": func(b *FeatureBinding) { b.Worktree = "/elsewhere" },
	} {
		other := *f
		change(&other)
		if sameFeature(f, &other) || sameFeature(&other, f) {
			t.Errorf("expected bindings with another %s to differ", name)
		}
	}
}

// TestValidateFeatureBinding: every field is required, split, feature,
// digest and branch cannot hold what the Details tokens cannot (whitespace,
// `=`, `,`, control characters), and the worktree must be absolute.
func TestValidateFeatureBinding(t *testing.T) {
	if err := validateFeatureBinding(*testFeature("split-1", "auth", "0123456789ab")); err != nil {
		t.Fatalf("validateFeatureBinding(complete binding) = %v, want nil", err)
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
// level. An org bound to a feature passes only the same paths with the same
// binding, and refuses another feature, digest, branch or worktree, other
// paths, and a reservation without a binding even for the same paths. An org
// with a plain reservation refuses a binding. An org that is running without
// a reservation (seats of a promoted session's leader, a leader from `ralph
// org start <task>`, an open workspace) refuses a binding but still takes a
// plain reservation. A new or disbanded org takes a binding unless its paths
// overlap another running org's, and an org whose record is damaged refuses
// everything.
func TestReservationDecision_Feature(t *testing.T) {
	fAuth := testFeature("split-1", "auth", "0123456789ab")
	withField := func(change func(*FeatureBinding)) *FeatureBinding {
		f := *fAuth
		change(&f)
		return &f
	}
	events := []ManifestEvent{
		boundReserveEvent("org-bound", fAuth, "internal/auth/"),
		reserveEvent("org-plain", "docs/"),
		{OrgID: "org-promoted", SeatID: "seat-1", Event: EventSpawned},
		{OrgID: "org-task", SeatID: LeaderIdentity, Event: EventSpawned},
		{OrgID: "org-ws", Event: EventOrgWorkspaceCreated, PaneID: "ws-1"},
		boundReserveEvent("org-gone", fAuth, "cmd/"),
		{OrgID: "org-gone", Event: EventDisbanded},
		{OrgID: "org-damaged", Event: EventScopeReserved, Details: "paths=damaged/ split=split-1 feature=auth", Worktree: fAuth.Worktree},
	}
	const boundTo = `org_id "org-bound" is bound to feature "auth" of split plan "split-1" (digest "0123456789ab", branch "feat/auth", worktree "/repo/.claude/worktrees/org-auth") and reserves internal/auth/`
	const disbandBound = "ralph org disband --org-id org-bound first, or use another org_id"
	tests := []struct {
		name       string
		orgID      string
		want       Reservation
		wantRecord bool
		wantError  []string
	}{
		{"bound: the same paths and binding pass without a record", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(*FeatureBinding) {})}, false, nil},
		{"bound: another feature is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Feature = "api" })}, false,
			[]string{boundTo, `so the reservation internal/auth/ for feature "api" of split plan "split-1"`, disbandBound}},
		{"bound: another digest is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Digest = "ba9876543210" })}, false,
			[]string{boundTo, `(digest "ba9876543210"`, disbandBound}},
		{"bound: another split plan is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Split = "split-2" })}, false,
			[]string{boundTo, `split plan "split-2"`}},
		{"bound: another branch is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Branch = "fix/auth" })}, false,
			[]string{boundTo}},
		{"bound: another worktree is refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}, Feature: withField(func(f *FeatureBinding) { f.Worktree = "/elsewhere" })}, false,
			[]string{boundTo}},
		{"bound: the same binding with other paths is refused", "org-bound",
			Reservation{Paths: []string{"internal/"}, Feature: withField(func(*FeatureBinding) {})}, false,
			[]string{boundTo, "so the reservation internal/ for"}},
		{"bound: the same paths without a binding are refused", "org-bound",
			Reservation{Paths: []string{"internal/auth/"}}, false,
			[]string{boundTo, "so a reservation without a split plan (internal/auth/) is refused", disbandBound}},
		{"plain: a binding for the same paths is refused", "org-plain",
			Reservation{Paths: []string{"docs/"}, Feature: fAuth}, false,
			[]string{`org_id "org-plain" was started without a split plan (it reserves docs/ with a plain --reserve)`,
				`so the reservation docs/ for feature "auth"`, "use another org_id, or ralph org disband --org-id org-plain first"}},
		{"promoted leader's org: a binding is refused", "org-promoted",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, false,
			[]string{`org_id "org-promoted" is running without a split plan`, `feature "auth"`, "ralph org disband --org-id org-promoted first"}},
		{"ralph org start <task>'s org: a binding is refused", "org-task",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, false,
			[]string{`org_id "org-task" is running without a split plan`}},
		{"an org with only an open workspace: a binding is refused", "org-ws",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, false,
			[]string{`org_id "org-ws" is running without a split plan`}},
		{"promoted leader's org: a plain reservation is still taken", "org-promoted",
			Reservation{Paths: []string{"api/"}}, true, nil},
		{"a new org takes a binding", "org-new",
			Reservation{Paths: []string{"api/"}, Feature: fAuth}, true, nil},
		{"a disbanded bound org takes another binding", "org-gone",
			Reservation{Paths: []string{"cmd/"}, Feature: withField(func(f *FeatureBinding) { f.Digest = "ba9876543210" })}, true, nil},
		{"a binding overlapping another running org is refused", "org-new",
			Reservation{Paths: []string{"internal/"}, Feature: fAuth}, false,
			[]string{`running org_id "org-bound": internal/ overlaps internal/auth/`}},
		{"damaged: the binding as read is refused", "org-damaged",
			Reservation{Paths: []string{"damaged/"}, Feature: &FeatureBinding{Split: "split-1", Feature: "auth", Worktree: fAuth.Worktree}}, false,
			[]string{`org_id "org-damaged" is bound to feature "auth" of split plan "split-1"`, "read from an incomplete record"}},
		{"damaged: the same paths without a binding are refused", "org-damaged",
			Reservation{Paths: []string{"damaged/"}}, false,
			[]string{`org_id "org-damaged" is bound to`, "so a reservation without a split plan (damaged/) is refused"}},
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
