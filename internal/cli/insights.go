package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/insights"
	"github.com/yoshpy-dev/ralph/internal/org"
)

func newInsightsCmd() *cobra.Command {
	var (
		eventsDir    string
		receiptsPath string
		jsonMode     bool
	)

	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Show aggregated pipeline insights",
		Long: `Aggregate insight events from docs/insights/events/ and org runtime model
receipts (<org-state-dir>/model-receipts.jsonl) into a pipeline summary.

The org state dir is resolved with the same precedence "ralph org" verbs
use (env RALPH_ORG_STATE_DIR, git toplevel, then cwd) -- see
internal/org/statedir.go's ResolveOrgStateDir. Pass --receipts to point at
an explicit file instead.

Sections:
  Events     — per-phase table: phase / events / verdicts / findings / triage
  Escalation — slugs that reached cycle >= 2 with cycle-1 vs final outcomes
  Routing    — honored-rate per phase (from event routing fields)
  Receipts   — org runtime model-commanding receipts grouped by org_id x
               seat_id, tri-state honored (true/false/unknown); rate is
               true/(true+false) with unknown excluded from the denominator

Use --json for machine-readable output of the full aggregate.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if eventsDir == "" {
				eventsDir = "docs/insights/events"
			}
			if receiptsPath == "" {
				stateDir, _ := org.ResolveOrgStateDir("", false)
				receiptsPath = org.ReceiptsPathIn(stateDir)
			}
			return runInsights(eventsDir, receiptsPath, jsonMode, cmd)
		},
	}

	cmd.Flags().StringVar(&eventsDir, "events-dir", "", "directory containing insight event JSONL files (default: docs/insights/events)")
	cmd.Flags().StringVar(&receiptsPath, "receipts", "", "path to the org runtime's model-receipts.jsonl (default: resolved via the org state-dir precedence -- env RALPH_ORG_STATE_DIR, git toplevel, or cwd)")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "emit the aggregate as JSON")

	cmd.AddCommand(newInsightsBackfillCmd())

	return cmd
}

func runInsights(eventsDir, receiptsPath string, jsonMode bool, cmd *cobra.Command) error {
	events, stats, err := insights.ReadEvents(eventsDir)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}

	agg := insights.Aggregate(events, stats)

	// Attach org runtime receipt diagnostics (supplementary; grouped by
	// org_id x seat_id). AggregateReceipts always returns a valid,
	// non-nil-Orgs summary regardless of presence, so the JSON shape stays
	// identical whether or not the receipts file exists or has any lines.
	receipts, rStats, err := insights.ReadReceipts(receiptsPath)
	if err != nil {
		return fmt.Errorf("reading receipts: %w", err)
	}
	receiptsPresent := len(receipts) > 0 || rStats.LinesRead > 0
	agg.Receipts = insights.AggregateReceipts(receipts, rStats, receiptsPath)

	if jsonMode {
		// JSON mode always emits valid JSON — zero data is represented as an
		// empty aggregate object, not a human-readable message.
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(agg)
	}

	// Zero-data early return — human mode only.
	if agg.TotalEvents == 0 && !receiptsPresent {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No insight data yet.")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Expected events:   %s\n", eventsDir)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Expected receipts: %s\n", receiptsPath)
		return nil
	}

	return printInsightsHuman(agg, cmd)
}

func printInsightsHuman(agg *insights.AggregateResult, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	_, _ = fmt.Fprintf(out, "\n=== Events (%d total, %d lines skipped) ===\n\n", agg.TotalEvents, agg.SkippedLines)

	if len(agg.PerPhase) == 0 {
		_, _ = fmt.Fprintln(out, "  (no events)")
	} else {
		// Canonical phase order.
		phaseOrder := []string{"implement", "self_review", "verify", "test", "sync_docs", "cross_review", "pr"}
		// Collect any extra phases not in the canonical list.
		seen := make(map[string]bool)
		for _, p := range phaseOrder {
			seen[p] = true
		}
		var extras []string
		for p := range agg.PerPhase {
			if !seen[p] {
				extras = append(extras, p)
			}
		}
		sort.Strings(extras)
		orderedPhases := append(phaseOrder, extras...)

		_, _ = fmt.Fprintf(out, "%-14s %6s %6s %6s %6s %6s  %3s %3s %3s %3s  %3s %3s %3s\n",
			"Phase", "Events", "Pass", "Fail", "AR", "NA",
			"C", "H", "M", "L",
			"AR", "WC", "D")
		_, _ = fmt.Fprintf(out, "%-14s %6s %6s %6s %6s %6s  %3s %3s %3s %3s  %3s %3s %3s\n",
			"---", "---", "---", "---", "---", "---",
			"---", "---", "---", "---",
			"---", "---", "---")

		for _, phase := range orderedPhases {
			ps, ok := agg.PerPhase[phase]
			if !ok {
				continue
			}
			_, _ = fmt.Fprintf(out, "%-14s %6d %6d %6d %6d %6d  %3d %3d %3d %3d  %3d %3d %3d\n",
				phase,
				ps.Events,
				ps.Verdicts.Pass, ps.Verdicts.Fail, ps.Verdicts.ActionRequired, ps.Verdicts.NA,
				ps.Findings.Critical, ps.Findings.High, ps.Findings.Medium, ps.Findings.Low,
				ps.Triage.ActionRequired, ps.Triage.WorthConsidering, ps.Triage.Dismissed,
			)
		}
	}

	// Routing section (from event routing fields).
	_, _ = fmt.Fprintf(out, "\n=== Routing (honored-rate per phase from events) ===\n\n")
	if len(agg.PerPhase) == 0 {
		_, _ = fmt.Fprintln(out, "  (no events with routing fields)")
	} else {
		anyRouting := false
		phaseOrder := []string{"implement", "self_review", "verify", "test", "sync_docs", "cross_review", "pr"}
		seen := make(map[string]bool)
		for _, p := range phaseOrder {
			seen[p] = true
		}
		var extras []string
		for p := range agg.PerPhase {
			if !seen[p] {
				extras = append(extras, p)
			}
		}
		sort.Strings(extras)
		ordered := append(phaseOrder, extras...)

		for _, phase := range ordered {
			ps, ok := agg.PerPhase[phase]
			if !ok {
				continue
			}
			if ps.HonoredRate < 0 {
				continue // no routing data for this phase
			}
			anyRouting = true
			_, _ = fmt.Fprintf(out, "  %-14s honored-rate: %.0f%%\n", phase, ps.HonoredRate*100)
		}
		if !anyRouting {
			_, _ = fmt.Fprintln(out, "  (no events with routing fields)")
		}
	}

	// Escalation section.
	_, _ = fmt.Fprintf(out, "\n=== Escalation (slugs with cycle >= 2) ===\n\n")
	if len(agg.Escalations) == 0 {
		_, _ = fmt.Fprintln(out, "  none observed")
	} else {
		for _, esc := range agg.Escalations {
			_, _ = fmt.Fprintf(out, "  %s (max cycle: %d)\n", esc.Slug, esc.MaxCycle)
			// Canonical phases for display.
			for _, phase := range []string{"implement", "self_review", "verify", "test", "sync_docs", "cross_review", "pr"} {
				c1, hasC1 := esc.Cycle1Verdicts[phase]
				cf, hasCF := esc.FinalVerdicts[phase]
				if !hasC1 && !hasCF {
					continue
				}
				if !hasC1 {
					c1 = "-"
				}
				if !hasCF {
					cf = "-"
				}
				_, _ = fmt.Fprintf(out, "    %-14s cycle1=%-16s final=%s\n", phase, c1, cf)
			}
		}
	}

	// Receipts section (org runtime model-commanding receipts, grouped by
	// org_id x seat_id; supplementary machine-local diagnostics, never
	// joined against events).
	_, _ = fmt.Fprintf(out, "\n=== Receipts (org runtime model-receipts: %s) ===\n\n", agg.Receipts.Path)
	if len(agg.Receipts.Orgs) == 0 {
		_, _ = fmt.Fprintf(out, "  no org receipts found (%s)\n", agg.Receipts.Path)
	} else {
		for _, o := range agg.Receipts.Orgs {
			for _, s := range o.Seats {
				commanded := strings.Join(s.CommandedModels, ",")
				rateStr := "n/a"
				if rate, ok := s.HonoredRate(); ok {
					rateStr = fmt.Sprintf("%.0f%%", rate*100)
				}
				_, _ = fmt.Fprintf(out, "  ORG %s  SEAT %s  commanded=%s  honored: true=%d false=%d unknown=%d  rate=%s (unknown %d excluded)\n",
					o.OrgID, s.SeatID, commanded, s.HonoredTrue, s.HonoredFalse, s.HonoredUnknown, rateStr, s.HonoredUnknown)
			}
		}
	}
	if agg.Receipts.SkippedLines > 0 {
		_, _ = fmt.Fprintf(out, "  (%d corrupt line(s) skipped)\n", agg.Receipts.SkippedLines)
	}
	_, _ = fmt.Fprintln(out)
	return nil
}

// newInsightsBackfillCmd returns the "ralph insights backfill" subcommand.
func newInsightsBackfillCmd() *cobra.Command {
	var (
		reportsDir string
		eventsDir  string
		apply      bool
	)

	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Parse existing docs/reports/ and emit backfill insight events",
		Long: `Parse historical pipeline reports (self-review, verify, test, cross-review-triage)
from docs/reports/ and derive insight events with source:"backfill".

By default this is a dry-run: events are printed but not written.
Pass --apply to write them to the events directory.

Deduplication key: source_report_path + phase + cycle — running --apply
twice produces zero new events (idempotent). Multi-cycle addenda produce
distinct events per cycle.

Event timestamps are set to the report file's mtime, not the current time,
so historical ordering is preserved.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportsDir == "" {
				reportsDir = "docs/reports"
			}
			if eventsDir == "" {
				eventsDir = "docs/insights/events"
			}
			return runInsightsBackfill(reportsDir, eventsDir, apply, cmd)
		},
	}

	cmd.Flags().StringVar(&reportsDir, "reports-dir", "", "directory containing report .md files (default: docs/reports)")
	cmd.Flags().StringVar(&eventsDir, "events-dir", "", "directory to write insight events (default: docs/insights/events)")
	cmd.Flags().BoolVar(&apply, "apply", false, "write events; default is dry-run (print only)")

	return cmd
}

func runInsightsBackfill(reportsDir, eventsDir string, apply bool, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	// Collect parse misses for summary.
	pattern := reportsDir + "/*.md"
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("globbing %s: %w", reportsDir, err)
	}

	// Load existing dedup keys first.
	existing, err := insights.LoadExistingDedupeKeys(eventsDir)
	if err != nil {
		return fmt.Errorf("loading existing events: %w", err)
	}

	type entry struct {
		ev     *insights.BackfillEvent
		key    string
		isDupe bool
	}

	var entries []entry
	var parseMiss int

	for _, f := range files {
		bevs, err := insights.ParseReport(f)
		if err != nil {
			parseMiss++
			continue
		}
		if bevs == nil {
			continue // unrecognised file type
		}
		for i := range bevs {
			bev := &bevs[i]
			if bev.ParseMiss {
				parseMiss++
				continue
			}
			key := insights.DedupeKey(bev.Event)
			entries = append(entries, entry{ev: bev, key: key, isDupe: existing[key]})
		}
	}

	newCount := 0
	dupeCount := 0
	for _, e := range entries {
		if e.isDupe {
			dupeCount++
		} else {
			newCount++
		}
	}

	if !apply {
		_, _ = fmt.Fprintf(out, "Backfill dry-run: %d reports scanned, %d derivable events, %d duplicates, %d parse misses\n\n",
			len(files), newCount+dupeCount, dupeCount, parseMiss)
		for _, e := range entries {
			dupeMark := ""
			if e.isDupe {
				dupeMark = " [duplicate — would skip]"
			}
			_, _ = fmt.Fprintf(out, "  %s  phase=%-12s cycle=%d  verdict=%-16s%s\n",
				e.ev.Slug, e.ev.Phase, e.ev.Cycle, e.ev.Verdict, dupeMark)
		}
		if parseMiss > 0 {
			_, _ = fmt.Fprintf(out, "\n  parse misses: %d (unrecognised format or missing verdict)\n", parseMiss)
		}
		return nil
	}

	// Apply mode: write new events. Re-check `existing` inside the loop so
	// two same-batch entries sharing a DedupeKey cannot both be written
	// (isDupe only reflects the state at scan time).
	written := 0
	for _, e := range entries {
		if e.isDupe || existing[e.key] {
			if !e.isDupe {
				// Same-batch collision skipped here — keep the summary count
				// consistent with what was actually skipped.
				dupeCount++
			}
			continue
		}
		if err := insights.AppendBackfillEvent(eventsDir, e.ev.Event); err != nil {
			return fmt.Errorf("writing event for %s: %w", e.ev.SourceReportPath, err)
		}
		existing[e.key] = true
		written++
	}

	_, _ = fmt.Fprintf(out, "Backfill applied: %d new events written, %d duplicates skipped, %d parse misses\n",
		written, dupeCount, parseMiss)
	return nil
}
