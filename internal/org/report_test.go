package org

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedClock returns an org.Clock that always returns the parsed RFC3339
// instant ts -- used so Report's date derivation (o.now()'s date portion)
// is deterministic in tests, the same way every other org verb test injects
// a fixed Clock.
func fixedClock(t *testing.T, ts string) Clock {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatalf("fixedClock: parse %q: %v", ts, err)
	}
	return func() time.Time { return parsed }
}

func fixtureReportEvents() []ManifestEvent {
	return []ManifestEvent{
		{TS: "2026-08-02T10:00:00Z", OrgID: "org-a", SeatID: "", Event: EventOrgWorkspaceCreated, PaneID: "ws-1"},
		{TS: "2026-08-02T10:00:01Z", OrgID: "org-a", SeatID: "lead", Event: EventSpawnStarted, Role: "lead", Driver: "claude", Model: "opus"},
		{TS: "2026-08-02T10:00:05Z", OrgID: "org-a", SeatID: "lead", Event: EventSpawned, Role: "lead", Driver: "claude", Model: "opus", PaneID: "pane-1", AgmsgTeam: "ralph-org-a", HerdrAgentName: "org-a_lead", Details: "scope=repo permission_mode=autonomous"},
		{TS: "2026-08-02T10:01:00Z", OrgID: "org-a", SeatID: "reviewer-1", Event: EventSpawnStarted, Role: "reviewer", Driver: "claude", Model: "sonnet"},
		{TS: "2026-08-02T10:01:05Z", OrgID: "org-a", SeatID: "reviewer-1", Event: EventSpawned, Role: "reviewer", Driver: "claude", Model: "sonnet", PaneID: "pane-2", AgmsgTeam: "ralph-org-a", HerdrAgentName: "org-a_reviewer-1", Details: "scope=internal/org/** permission_mode=autonomous"},
		{TS: "2026-08-02T10:05:00Z", OrgID: "org-a", SeatID: "reviewer-1", Event: EventStopped, Role: "reviewer", Driver: "claude", Model: "sonnet", PaneID: "pane-2", AgmsgTeam: "ralph-org-a", Details: "pane=ok leave=ok"},
		{TS: "2026-08-02T10:02:00Z", OrgID: "org-b", SeatID: "seat-x", Event: EventSpawned, Role: "worker", Driver: "claude", Model: "sonnet", Details: "permission_mode=guarded"},
	}
}

func fixtureReportReceipts() []Receipt {
	return []Receipt{
		{TS: "2026-08-02T10:00:05Z", OrgID: "org-a", SeatID: "lead", Role: "lead", Driver: "claude", CommandedModel: "opus", Honored: HonoredUnknown, Reason: "interactive session; effective model not yet observable"},
		{TS: "2026-08-02T10:01:05Z", OrgID: "org-a", SeatID: "reviewer-1", Role: "reviewer", Driver: "claude", CommandedModel: "sonnet", Honored: HonoredUnknown},
		{TS: "2026-08-02T10:02:00Z", OrgID: "org-b", SeatID: "seat-x", Role: "worker", Driver: "claude", CommandedModel: "sonnet", Honored: HonoredTrue, ReportedEffectiveModel: "sonnet"},
	}
}

func TestBuildOrgReport_RosterTimelineReceiptsAndResiduals(t *testing.T) {
	events := filterEventsByOrg(fixtureReportEvents(), "org-a")
	receipts := filterReceiptsByOrg(fixtureReportReceipts(), "org-a")

	got := BuildOrgReport(events, receipts, "org-a", "2026-08-02", 2)

	if !strings.Contains(got, "# org report: org-a") {
		t.Errorf("expected a title naming the org_id, got:\n%s", got)
	}
	if !strings.Contains(got, "generated: 2026-08-02") {
		t.Errorf("expected the generated date to appear, got:\n%s", got)
	}

	// Roster: org-b's seat-x must never leak into org-a's report.
	if strings.Contains(got, "seat-x") {
		t.Errorf("expected org-b's seat-x excluded from org-a's report, got:\n%s", got)
	}
	for _, want := range []string{"## Roster", "lead", "reviewer-1", "autonomous"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected roster section to contain %q, got:\n%s", want, got)
		}
	}
	// reviewer-1 was stopped -- its state must not be reported "active".
	if !strings.Contains(got, "stopped") {
		t.Errorf("expected reviewer-1's derived state 'stopped' to appear, got:\n%s", got)
	}

	// Timeline: every org-a event, including the org-level workspace_created
	// event (seat_id blank -> "-").
	for _, want := range []string{"## Event timeline", "org_workspace_created", "spawn_started", "spawned", "stopped"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected event timeline section to contain %q, got:\n%s", want, got)
		}
	}

	// Receipts.
	for _, want := range []string{"## Model receipts", "commanded_model", "opus", "unknown"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected model receipts section to contain %q, got:\n%s", want, got)
		}
	}

	// Known residuals: only lead is still active (reviewer-1 was stopped).
	if !strings.Contains(got, "## Known residuals") || !strings.Contains(got, "active seats: 1") {
		t.Errorf("expected known residuals section reporting 1 active seat, got:\n%s", got)
	}
	if !strings.Contains(got, "corrupt manifest lines: 2") {
		t.Errorf("expected known residuals section to report the passed-in corrupt line count, got:\n%s", got)
	}
}

func TestBuildOrgReport_EmptyOrg_NoEventsNote(t *testing.T) {
	got := BuildOrgReport(nil, nil, "org-empty", "2026-08-02", 0)

	if !strings.Contains(got, "# org report: org-empty") {
		t.Errorf("expected the title even for an empty org, got:\n%s", got)
	}
	if !strings.Contains(got, "no events recorded for this org") {
		t.Errorf("expected an explicit no-events note for an org with zero events, got:\n%s", got)
	}
	if !strings.Contains(got, "active seats: 0") {
		t.Errorf("expected 0 active seats for an empty org, got:\n%s", got)
	}
	if !strings.Contains(got, "(no seats)") {
		t.Errorf("expected a (no seats) marker in the roster section, got:\n%s", got)
	}
	if !strings.Contains(got, "(no receipts recorded for this org)") {
		t.Errorf("expected a no-receipts marker in the receipts section, got:\n%s", got)
	}
}

func TestPermissionModeFromDetails_ExtractsFragmentOrEmpty(t *testing.T) {
	cases := []struct {
		details string
		want    string
	}{
		{"scope=repo permission_mode=autonomous", "autonomous"},
		{"permission_mode=guarded", "guarded"},
		{"scope=repo allow_unscoped=true permission_mode=edits", "edits"},
		{"", ""},
		{"pane=ok leave=ok", ""},
	}
	for _, c := range cases {
		if got := permissionModeFromDetails(c.details); got != c.want {
			t.Errorf("permissionModeFromDetails(%q) = %q, want %q", c.details, got, c.want)
		}
	}
}

func TestOrgReport_WritesFileFilteredByOrgAndUsesInjectedClock(t *testing.T) {
	dir := t.TempDir()
	o := &Org{
		Manifest: NewManifestStoreAtPath(ManifestPathIn(dir)),
		Receipts: NewReceiptStoreAtPath(filepath.Join(dir, "receipts.jsonl")),
		Now:      fixedClock(t, "2026-08-02T12:00:00Z"),
	}

	for _, ev := range fixtureReportEvents() {
		if err := o.Manifest.Append(ev); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}
	for _, r := range fixtureReportReceipts() {
		if err := o.Receipts.Append(r); err != nil {
			t.Fatalf("append receipt: %v", err)
		}
	}

	outDir := filepath.Join(dir, "reports")
	result := o.Report(ReportParams{OrgID: "org-a", OutDir: outDir})
	if result.Err != nil {
		t.Fatalf("Report: unexpected error: %v", result.Err)
	}
	wantPath := filepath.Join(outDir, "org-manifest-org-a-2026-08-02.md")
	if result.Path != wantPath {
		t.Fatalf("Report: Path = %q, want %q", result.Path, wantPath)
	}

	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("expected the report file to exist at %q: %v", wantPath, err)
	}
	content := string(data)
	if strings.Contains(content, "seat-x") {
		t.Errorf("expected org-b's seat-x excluded from the written report, got:\n%s", content)
	}
	if !strings.Contains(content, "reviewer-1") {
		t.Errorf("expected org-a's reviewer-1 in the written report, got:\n%s", content)
	}
}

func TestOrgReport_DefaultOutDir_WritesUnderDocsReports(t *testing.T) {
	dir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	o := &Org{
		Manifest: NewManifestStoreAtPath(filepath.Join(dir, "state", "manifest.jsonl")),
		Receipts: NewReceiptStoreAtPath(filepath.Join(dir, "state", "receipts.jsonl")),
		Now:      fixedClock(t, "2026-08-02T00:00:00Z"),
	}
	if err := o.Manifest.Append(ManifestEvent{TS: "2026-08-02T00:00:00Z", OrgID: "org-a", SeatID: "lead", Event: EventSpawned, Role: "lead"}); err != nil {
		t.Fatalf("append event: %v", err)
	}

	result := o.Report(ReportParams{OrgID: "org-a"})
	if result.Err != nil {
		t.Fatalf("Report: unexpected error: %v", result.Err)
	}
	// Report's default OutDir (defaultReportDir) is the plain relative
	// "docs/reports", resolved against the process cwd (chdir'd to dir
	// above) -- it is not made absolute.
	wantPath := filepath.Join("docs", "reports", "org-manifest-org-a-2026-08-02.md")
	if result.Path != wantPath {
		t.Fatalf("Report: Path = %q, want %q (default OutDir must be docs/reports)", result.Path, wantPath)
	}
	absPath := filepath.Join(dir, "docs", "reports", "org-manifest-org-a-2026-08-02.md")
	if _, err := os.Stat(absPath); err != nil {
		t.Fatalf("expected the report file at the default path: %v", err)
	}
}

func TestOrgReport_EmptyOrg_StillWritesReportWithNoEventsNote(t *testing.T) {
	dir := t.TempDir()
	o := &Org{
		Manifest: NewManifestStoreAtPath(ManifestPathIn(dir)),
		Receipts: NewReceiptStoreAtPath(filepath.Join(dir, "receipts.jsonl")),
		Now:      fixedClock(t, "2026-08-02T00:00:00Z"),
	}
	result := o.Report(ReportParams{OrgID: "org-empty", OutDir: filepath.Join(dir, "reports")})
	if result.Err != nil {
		t.Fatalf("Report: unexpected error for an org with zero events: %v", result.Err)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("expected the report file to exist: %v", err)
	}
	if !strings.Contains(string(data), "no events recorded for this org") {
		t.Errorf("expected the empty-org report to contain the no-events note, got:\n%s", string(data))
	}
}
