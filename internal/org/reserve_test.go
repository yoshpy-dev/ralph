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
		details := scopeReservedDetails(paths, note)
		got, ok := reservedPathsFromDetails(details)
		if !ok || !slices.Equal(got, paths) {
			t.Fatalf("reservedPathsFromDetails(%q) = %q, %v; want %q", details, got, ok, paths)
		}
	}
	if d := scopeReservedDetails(paths, ""); d != "paths=docs/,internal/auth/token.go" {
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
	return scopeReservedEvent("", orgID, paths, "", false)
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
