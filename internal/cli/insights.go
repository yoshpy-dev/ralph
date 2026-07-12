package cli

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/insights"
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
		Long: `Aggregate insight events from docs/insights/events/ and model receipts
from .harness/state/pipeline/model-receipts.jsonl into a pipeline summary.

Sections:
  Events     — per-phase table: phase / events / verdicts / findings / triage
  Escalation — slugs that reached cycle >= 2 with cycle-1 vs final outcomes
  Routing    — honored-rate per phase (from event routing fields)
  Local receipts — supplementary machine-local diagnostics (not joined to events)

Use --json for machine-readable output of the full aggregate.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if eventsDir == "" {
				eventsDir = "docs/insights/events"
			}
			if receiptsPath == "" {
				receiptsPath = ".harness/state/pipeline/model-receipts.jsonl"
			}
			return runInsights(eventsDir, receiptsPath, jsonMode, cmd)
		},
	}

	cmd.Flags().StringVar(&eventsDir, "events-dir", "", "directory containing insight event JSONL files (default: docs/insights/events)")
	cmd.Flags().StringVar(&receiptsPath, "receipts", "", "path to model-receipts.jsonl (default: .harness/state/pipeline/model-receipts.jsonl)")
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

	// Attach receipt diagnostics (supplementary; absent receipts → Present=false).
	receipts, rStats, err := insights.ReadReceipts(receiptsPath)
	if err != nil {
		return fmt.Errorf("reading receipts: %w", err)
	}
	if len(receipts) > 0 || rStats.LinesRead > 0 {
		insights.AggregateWithReceipts(agg, receipts, rStats)
	}

	// Zero-data early return.
	if agg.TotalEvents == 0 && !agg.Receipts.Present {
		fmt.Fprintln(cmd.OutOrStdout(), "No insight data yet.")
		fmt.Fprintf(cmd.OutOrStdout(), "  Expected events:   %s\n", eventsDir)
		fmt.Fprintf(cmd.OutOrStdout(), "  Expected receipts: %s\n", receiptsPath)
		return nil
	}

	if jsonMode {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(agg)
	}

	return printInsightsHuman(agg, cmd)
}

func printInsightsHuman(agg *insights.AggregateResult, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "\n=== Events (%d total, %d lines skipped) ===\n\n", agg.TotalEvents, agg.SkippedLines)

	if len(agg.PerPhase) == 0 {
		fmt.Fprintln(out, "  (no events)")
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

		fmt.Fprintf(out, "%-14s %6s %6s %6s %6s %6s  %3s %3s %3s %3s  %3s %3s %3s\n",
			"Phase", "Events", "Pass", "Fail", "AR", "NA",
			"C", "H", "M", "L",
			"AR", "WC", "D")
		fmt.Fprintf(out, "%-14s %6s %6s %6s %6s %6s  %3s %3s %3s %3s  %3s %3s %3s\n",
			"---", "---", "---", "---", "---", "---",
			"---", "---", "---", "---",
			"---", "---", "---")

		for _, phase := range orderedPhases {
			ps, ok := agg.PerPhase[phase]
			if !ok {
				continue
			}
			fmt.Fprintf(out, "%-14s %6d %6d %6d %6d %6d  %3d %3d %3d %3d  %3d %3d %3d\n",
				phase,
				ps.Events,
				ps.Verdicts.Pass, ps.Verdicts.Fail, ps.Verdicts.ActionRequired, ps.Verdicts.NA,
				ps.Findings.Critical, ps.Findings.High, ps.Findings.Medium, ps.Findings.Low,
				ps.Triage.ActionRequired, ps.Triage.WorthConsidering, ps.Triage.Dismissed,
			)
		}
	}

	// Routing section (from event routing fields).
	fmt.Fprintf(out, "\n=== Routing (honored-rate per phase from events) ===\n\n")
	if len(agg.PerPhase) == 0 {
		fmt.Fprintln(out, "  (no events with routing fields)")
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
			fmt.Fprintf(out, "  %-14s honored-rate: %.0f%%\n", phase, ps.HonoredRate*100)
		}
		if !anyRouting {
			fmt.Fprintln(out, "  (no events with routing fields)")
		}
	}

	// Escalation section.
	fmt.Fprintf(out, "\n=== Escalation (slugs with cycle >= 2) ===\n\n")
	if len(agg.Escalations) == 0 {
		fmt.Fprintln(out, "  none observed")
	} else {
		for _, esc := range agg.Escalations {
			fmt.Fprintf(out, "  %s (max cycle: %d)\n", esc.Slug, esc.MaxCycle)
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
				fmt.Fprintf(out, "    %-14s cycle1=%-16s final=%s\n", phase, c1, cf)
			}
		}
	}

	// Local receipts section (supplementary / machine-local).
	fmt.Fprintf(out, "\n=== Local receipts (machine-local supplementary diagnostics) ===\n\n")
	diag := agg.Receipts
	if !diag.Present {
		fmt.Fprintln(out, "  not present — run a ralph pipeline loop to generate receipts")
		fmt.Fprintln(out, "  (receipts are machine-local state, not committed to the repo)")
	} else {
		fmt.Fprintf(out, "  total: %d  skipped: %d  honored-rate: %.0f%%\n",
			diag.TotalCount, diag.SkippedLines, diag.HonoredRate*100)
		if len(diag.PerPhase) > 0 {
			fmt.Fprintln(out, "  per phase:")
			phaseOrder := []string{"implement", "self_review", "verify", "test", "sync_docs", "cross_review", "pr"}
			seen := make(map[string]bool)
			for _, p := range phaseOrder {
				seen[p] = true
			}
			var extras []string
			for p := range diag.PerPhase {
				if !seen[p] {
					extras = append(extras, p)
				}
			}
			sort.Strings(extras)
			for _, phase := range append(phaseOrder, extras...) {
				rate, ok := diag.PerPhase[phase]
				if !ok {
					continue
				}
				fmt.Fprintf(out, "    %-14s %.0f%%\n", phase, rate*100)
			}
		}
	}
	fmt.Fprintln(out)
	return nil
}

// newInsightsBackfillCmd returns the "ralph insights backfill" subcommand.
// Full implementation is in insights_backfill.go (Slice 5); this stub
// makes the Slice 4 build self-contained until Slice 5 replaces it.
func newInsightsBackfillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backfill",
		Short: "Parse existing docs/reports/ and emit backfill insight events",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("backfill not yet implemented")
		},
	}
}
