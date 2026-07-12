package insights

// PhaseStats holds aggregated metrics for one pipeline phase.
type PhaseStats struct {
	Phase    string        `json:"phase"`
	Events   int           `json:"events"`
	Verdicts VerdictCounts `json:"verdicts"`
	Findings FindingTotals `json:"findings"`
	Triage   TriageTotals  `json:"triage"`
	// HonoredRate is the fraction of events where honored==true.
	// -1 when no events have routing fields set (driver non-empty).
	HonoredRate float64 `json:"honored_rate"`
}

// VerdictCounts tallies per-verdict event counts for a phase.
type VerdictCounts struct {
	Pass           int `json:"pass"`
	Fail           int `json:"fail"`
	Complete       int `json:"complete"`
	ActionRequired int `json:"action_required"`
	NA             int `json:"na"`
	Other          int `json:"other"`
}

// FindingTotals sums findings across all events in a phase.
type FindingTotals struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// TriageTotals sums triage counts across all events in a phase.
type TriageTotals struct {
	ActionRequired   int `json:"action_required"`
	WorthConsidering int `json:"worth_considering"`
	Dismissed        int `json:"dismissed"`
}

// EscalationOutcome describes the pipeline outcome for one slug that had
// events at cycle >= 2 (i.e. outer-loop escalation was triggered).
type EscalationOutcome struct {
	Slug string `json:"slug"`
	// Cycle1Verdicts holds per-phase verdicts from cycle 1.
	Cycle1Verdicts map[string]string `json:"cycle1_verdicts"`
	// FinalVerdicts holds per-phase verdicts from the highest cycle seen.
	FinalVerdicts map[string]string `json:"final_verdicts"`
	// MaxCycle is the highest cycle number observed for this slug.
	MaxCycle int `json:"max_cycle"`
}

// ReceiptDiagnostics summarises the local model-receipts.jsonl file.
// This is supplementary local state and is never joined against events.
type ReceiptDiagnostics struct {
	// Present is false when the receipts file was absent.
	Present      bool    `json:"present"`
	TotalCount   int     `json:"total_count"`
	SkippedLines int     `json:"skipped_lines"`
	HonoredRate  float64 `json:"honored_rate"`
	// PerPhase maps phase name to honored-rate for that phase.
	PerPhase map[string]float64 `json:"per_phase"`
}

// AggregateResult holds all aggregated metrics derived from insight events.
type AggregateResult struct {
	// TotalEvents is the number of events read.
	TotalEvents int `json:"total_events"`
	// SkippedLines is the count of corrupt lines that were skipped.
	SkippedLines int `json:"skipped_lines"`
	// PerPhase maps phase name to its aggregated stats.
	PerPhase map[string]*PhaseStats `json:"per_phase"`
	// Escalations holds slugs that had events at cycle >= 2.
	Escalations []EscalationOutcome `json:"escalations"`
	// Receipts holds supplementary local-diagnostics (machine-local, not joined).
	Receipts ReceiptDiagnostics `json:"receipts"`
}

// Aggregate computes an AggregateResult from a slice of events.
// stats carries the read counters from ReadEvents.
func Aggregate(events []Event, stats ReadStats) *AggregateResult {
	agg := &AggregateResult{
		TotalEvents:  len(events),
		SkippedLines: stats.SkippedLines,
		PerPhase:     make(map[string]*PhaseStats),
	}

	// Per-phase accumulation.
	// Also track slug→cycle→phase→verdict for escalation view.
	// slugPhaseVerdict[slug][cycle][phase] = last verdict
	slugPhaseVerdict := make(map[string]map[int]map[string]string)

	// For honored-rate: track routed counts per phase.
	type routingAccum struct {
		total   int
		honored int
	}
	phaseRouting := make(map[string]*routingAccum)

	for _, ev := range events {
		ps, ok := agg.PerPhase[ev.Phase]
		if !ok {
			ps = &PhaseStats{
				Phase:       ev.Phase,
				HonoredRate: -1,
			}
			agg.PerPhase[ev.Phase] = ps
		}
		ps.Events++

		switch ev.Verdict {
		case "pass":
			ps.Verdicts.Pass++
		case "fail":
			ps.Verdicts.Fail++
		case "complete":
			ps.Verdicts.Complete++
		case "action_required":
			ps.Verdicts.ActionRequired++
		case "n/a":
			ps.Verdicts.NA++
		default:
			ps.Verdicts.Other++
		}

		ps.Findings.Critical += ev.Findings.Critical
		ps.Findings.High += ev.Findings.High
		ps.Findings.Medium += ev.Findings.Medium
		ps.Findings.Low += ev.Findings.Low

		ps.Triage.ActionRequired += ev.Triage.ActionRequired
		ps.Triage.WorthConsidering += ev.Triage.WorthConsidering
		ps.Triage.Dismissed += ev.Triage.Dismissed

		// Routing: only count events that have driver set (routing fields present).
		if ev.Driver != "" {
			if phaseRouting[ev.Phase] == nil {
				phaseRouting[ev.Phase] = &routingAccum{}
			}
			phaseRouting[ev.Phase].total++
			if ev.Honored {
				phaseRouting[ev.Phase].honored++
			}
		}

		// Escalation tracking: record per-slug per-cycle per-phase verdicts.
		if _, ok := slugPhaseVerdict[ev.Slug]; !ok {
			slugPhaseVerdict[ev.Slug] = make(map[int]map[string]string)
		}
		if _, ok := slugPhaseVerdict[ev.Slug][ev.Cycle]; !ok {
			slugPhaseVerdict[ev.Slug][ev.Cycle] = make(map[string]string)
		}
		slugPhaseVerdict[ev.Slug][ev.Cycle][ev.Phase] = ev.Verdict
	}

	// Compute per-phase honored-rate from routing accumulator.
	for phase, ra := range phaseRouting {
		if ra.total > 0 {
			agg.PerPhase[phase].HonoredRate = float64(ra.honored) / float64(ra.total)
		}
	}

	// Build escalation outcomes: slugs with events at cycle >= 2.
	for slug, cycleMap := range slugPhaseVerdict {
		maxCycle := 0
		for c := range cycleMap {
			if c > maxCycle {
				maxCycle = c
			}
		}
		if maxCycle < 2 {
			continue
		}
		outcome := EscalationOutcome{
			Slug:           slug,
			MaxCycle:       maxCycle,
			Cycle1Verdicts: copyVerdictMap(cycleMap[1]),
			FinalVerdicts:  copyVerdictMap(cycleMap[maxCycle]),
		}
		agg.Escalations = append(agg.Escalations, outcome)
	}

	return agg
}

// AggregateWithReceipts attaches receipt diagnostics to an existing AggregateResult.
// receipts and rStats come from ReadReceipts; this is always a supplementary
// local-diagnostics section and never joined against events.
func AggregateWithReceipts(agg *AggregateResult, receipts []Receipt, rStats ReceiptStats) {
	diag := ReceiptDiagnostics{
		Present:      true,
		TotalCount:   len(receipts),
		SkippedLines: rStats.SkippedLines,
		PerPhase:     make(map[string]float64),
	}

	type ra struct{ total, honored int }
	byPhase := make(map[string]*ra)

	var totalHonored int
	for _, r := range receipts {
		if r.Honored {
			totalHonored++
		}
		if r.Phase == "" {
			continue
		}
		if byPhase[r.Phase] == nil {
			byPhase[r.Phase] = &ra{}
		}
		byPhase[r.Phase].total++
		if r.Honored {
			byPhase[r.Phase].honored++
		}
	}

	if len(receipts) > 0 {
		diag.HonoredRate = float64(totalHonored) / float64(len(receipts))
	}
	for phase, a := range byPhase {
		if a.total > 0 {
			diag.PerPhase[phase] = float64(a.honored) / float64(a.total)
		}
	}

	agg.Receipts = diag
}

func copyVerdictMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
