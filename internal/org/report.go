package org

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultReportDir is the default output directory for `ralph org report`
// (AC-4, FR-9 後半 -- see docs/plans/active/2026-08-02-org-runtime-lead.md).
const defaultReportDir = "docs/reports"

// ReportParams describes one `ralph org report` invocation.
type ReportParams struct {
	OrgID string
	// OutDir overrides defaultReportDir (the CLI's --out flag). Empty means
	// "use defaultReportDir".
	OutDir string
}

// ReportResult is Report's return value: the path written, or Err on
// failure.
type ReportResult struct {
	Path string
	Err  error
}

// Report reads the manifest and model receipts, filters both to OrgID, and
// writes a docs/reports/org-manifest-<org_id>-<date>.md artifact built by
// BuildOrgReport. date is derived from o.now() (UTC, YYYY-MM-DD) rather than
// calling time.Now() directly, so tests can inject a deterministic Org.Clock
// the same way every other org verb does, and so two `ralph org report`
// calls made within the same UTC day overwrite the same file rather than
// accumulating duplicates.
func (o *Org) Report(p ReportParams) ReportResult {
	rr, err := o.Manifest.Read()
	if err != nil {
		return ReportResult{Err: fmt.Errorf("org: report: read manifest: %w", err)}
	}
	receiptsResult, err := o.Receipts.Read()
	if err != nil {
		return ReportResult{Err: fmt.Errorf("org: report: read receipts: %w", err)}
	}

	events := filterEventsByOrg(rr.Events, p.OrgID)
	receipts := filterReceiptsByOrg(receiptsResult.Receipts, p.OrgID)
	date := reportDate(o.now())
	content := BuildOrgReport(events, receipts, p.OrgID, date, rr.CorruptLines)

	outDir := p.OutDir
	if outDir == "" {
		outDir = defaultReportDir
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ReportResult{Err: fmt.Errorf("org: report: create output dir %s: %w", outDir, err)}
	}
	path := filepath.Join(outDir, fmt.Sprintf("org-manifest-%s-%s.md", p.OrgID, date))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ReportResult{Err: fmt.Errorf("org: report: write %s: %w", path, err)}
	}
	return ReportResult{Path: path}
}

// reportDate extracts the YYYY-MM-DD prefix from an RFC3339 timestamp string
// (the shape o.now() always returns). A malformed/short input (defensive
// only -- o.now() never actually produces one) falls back to the input
// as-is rather than panicking on a slice-out-of-range.
func reportDate(ts string) string {
	if len(ts) < 10 {
		return ts
	}
	return ts[:10]
}

func filterEventsByOrg(events []ManifestEvent, orgID string) []ManifestEvent {
	out := make([]ManifestEvent, 0, len(events))
	for _, ev := range events {
		if ev.OrgID == orgID {
			out = append(out, ev)
		}
	}
	return out
}

func filterReceiptsByOrg(receipts []Receipt, orgID string) []Receipt {
	out := make([]Receipt, 0, len(receipts))
	for _, r := range receipts {
		if r.OrgID == orgID {
			out = append(out, r)
		}
	}
	return out
}

// permissionModeFromDetails extracts the "permission_mode=<value>" fragment
// spawnedEventDetails records on a `spawned` event's free-text Details field
// (spawn.go), returning "" when no such fragment is present (e.g. a
// `rejected`/`spawn_failed` event, or a `spawned` event recorded before
// AC-2b started stamping permission_mode on every seat).
func permissionModeFromDetails(details string) string {
	for field := range strings.FieldsSeq(details) {
		if v, ok := strings.CutPrefix(field, "permission_mode="); ok {
			return v
		}
	}
	return ""
}

// BuildOrgReport renders a `docs/reports/org-manifest-<org_id>-<date>.md`
// markdown document from already-org-filtered manifest events and receipts
// (see (*Org).Report, the only production caller). It is a pure function of
// its inputs -- no I/O, no Clock -- so it is directly and deterministically
// unit-testable against fixture events/receipts (AC-4's "スタブデータの
// ユニットテスト").
//
// Sections, in order: a roster summary (one row per seat, including dry-run
// seats, from Roster), the full event timeline (every event in file/append
// order, org-level events included), the model-receipts table, and a "known
// residuals" summary (active real-seat count, corrupt manifest line count).
// An org with zero events still produces a complete report with an explicit
// "no events recorded" note in place of the roster/timeline tables (edge
// case: "report 対象 org が空", plan Test plan).
func BuildOrgReport(events []ManifestEvent, receipts []Receipt, orgID, date string, corruptLines int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# org report: %s\n\n", orgID)
	fmt.Fprintf(&b, "- generated: %s\n", date)
	fmt.Fprintf(&b, "- org_id: %s\n\n", orgID)

	if len(events) == 0 {
		b.WriteString("(no events recorded for this org)\n\n")
	}

	b.WriteString("## Roster\n\n")
	roster := Roster(events, RosterOptions{IncludeDryRun: true})
	if len(roster) == 0 {
		b.WriteString("(no seats)\n\n")
	} else {
		b.WriteString("| seat_id | role | driver | model | state | permission | herdr_agent |\n")
		b.WriteString("|---|---|---|---|---|---|---|\n")
		for _, s := range roster {
			state := s.Event
			if s.Active {
				state += " (active)"
			}
			if s.DryRun {
				state += " [dry-run]"
			}
			perm := permissionModeFromDetails(s.Details)
			if perm == "" {
				perm = "-"
			}
			herdrAgent := s.HerdrAgentName
			if herdrAgent == "" {
				herdrAgent = "-"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
				s.SeatID, s.Role, s.Driver, s.Model, state, perm, herdrAgent)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Event timeline\n\n")
	if len(events) == 0 {
		b.WriteString("(no events recorded for this org)\n\n")
	} else {
		b.WriteString("| ts | seat_id | event | details |\n")
		b.WriteString("|---|---|---|---|\n")
		for _, ev := range events {
			seatID := ev.SeatID
			if seatID == "" {
				seatID = "-"
			}
			details := ev.Details
			if details == "" {
				details = "-"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", ev.TS, seatID, ev.Event, details)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Model receipts\n\n")
	if len(receipts) == 0 {
		b.WriteString("(no receipts recorded for this org)\n\n")
	} else {
		b.WriteString("| seat_id | role | driver | commanded_model | effective_model | honored |\n")
		b.WriteString("|---|---|---|---|---|---|\n")
		for _, r := range receipts {
			effective := r.ReportedEffectiveModel
			if effective == "" {
				effective = "-"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
				r.SeatID, r.Role, r.Driver, r.CommandedModel, effective, r.Honored)
		}
		b.WriteString("\n")
	}

	activeSeats := ActiveSeatCount(events, orgID, RosterOptions{})
	b.WriteString("## Known residuals\n\n")
	fmt.Fprintf(&b, "- active seats: %d\n", activeSeats)
	fmt.Fprintf(&b, "- corrupt manifest lines: %d\n", corruptLines)

	return b.String()
}
