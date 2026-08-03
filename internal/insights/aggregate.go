package insights

import (
	"sort"

	"github.com/yoshpy-dev/ralph/internal/org"
)

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

// ReceiptSeatStats aggregates tri-state honored counts for one seat within
// one org, plus the distinct set of commanded models observed for that
// seat. honored-rate is computed by HonoredRate: true/(true+false), with
// unknown excluded from the denominator (AC-3 output contract).
type ReceiptSeatStats struct {
	SeatID string `json:"seat_id"`
	// CommandedModels is the sorted, de-duplicated set of commanded_model
	// values seen for this seat.
	CommandedModels []string `json:"commanded_models"`
	HonoredTrue     int      `json:"honored_true"`
	HonoredFalse    int      `json:"honored_false"`
	HonoredUnknown  int      `json:"honored_unknown"`
}

// HonoredRate returns true/(true+false); unknown is excluded from the
// denominator. ok is false when there is no true/false data to rate (every
// receipt for this seat was unknown), in which case rate should be
// displayed as "n/a" rather than a misleading 0%.
func (s ReceiptSeatStats) HonoredRate() (rate float64, ok bool) {
	denom := s.HonoredTrue + s.HonoredFalse
	if denom == 0 {
		return 0, false
	}
	return float64(s.HonoredTrue) / float64(denom), true
}

// ReceiptOrgStats groups ReceiptSeatStats under one org_id. Seats is sorted
// by seat_id for deterministic output.
type ReceiptOrgStats struct {
	OrgID string             `json:"org_id"`
	Seats []ReceiptSeatStats `json:"seats"`
}

// ReceiptsSummary is the org-runtime receipts section of AggregateResult:
// receipts grouped by org_id x seat_id (Orgs sorted by org_id). This is
// supplementary machine-local diagnostics and is never joined against
// events. Orgs is always a non-nil (possibly empty) slice so the JSON shape
// is identical whether or not any receipts were found (status --json
// convention).
type ReceiptsSummary struct {
	// Path is the receipts file that was read, echoed back so a consumer
	// can tell an empty result from a genuinely different source.
	Path         string            `json:"path"`
	Orgs         []ReceiptOrgStats `json:"orgs"`
	SkippedLines int               `json:"skipped_lines"`
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
	// Receipts holds org runtime model receipts grouped by org_id x seat_id
	// (machine-local supplementary diagnostics, not joined against events).
	Receipts ReceiptsSummary `json:"receipts"`
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

// AggregateReceipts groups receipts by org_id x seat_id (Orgs sorted by
// org_id, Seats within an org sorted by seat_id -- deterministic output).
// path is echoed onto the result so callers building the human/JSON
// receipts section never need to thread it separately. This always
// produces a non-nil (possibly empty) Orgs slice, matching Read-Receipts'
// "missing file -> empty, not error" contract and keeping the JSON shape
// identical whether or not any receipts were found.
func AggregateReceipts(receipts []Receipt, stats ReceiptStats, path string) ReceiptsSummary {
	summary := ReceiptsSummary{
		Path:         path,
		Orgs:         []ReceiptOrgStats{},
		SkippedLines: stats.SkippedLines,
	}

	type seatKey struct{ orgID, seatID string }
	type seatAccum struct {
		commandedModels map[string]bool
		trueCount       int
		falseCount      int
		unknownCount    int
	}

	bySeat := make(map[seatKey]*seatAccum)
	seenOrg := make(map[string]bool)
	var orgIDs []string
	seenSeatByOrg := make(map[string]map[string]bool)

	for _, r := range receipts {
		key := seatKey{orgID: r.OrgID, seatID: r.SeatID}
		acc, ok := bySeat[key]
		if !ok {
			acc = &seatAccum{commandedModels: make(map[string]bool)}
			bySeat[key] = acc
		}
		if r.CommandedModel != "" {
			acc.commandedModels[r.CommandedModel] = true
		}
		switch r.Honored {
		case org.HonoredTrue:
			acc.trueCount++
		case org.HonoredFalse:
			acc.falseCount++
		default:
			// org.HonoredUnknown and any unrecognised value both count as
			// unknown -- tri-state display never silently drops a receipt.
			acc.unknownCount++
		}

		if !seenOrg[r.OrgID] {
			seenOrg[r.OrgID] = true
			orgIDs = append(orgIDs, r.OrgID)
		}
		if seenSeatByOrg[r.OrgID] == nil {
			seenSeatByOrg[r.OrgID] = make(map[string]bool)
		}
		seenSeatByOrg[r.OrgID][r.SeatID] = true
	}

	sort.Strings(orgIDs)
	for _, orgID := range orgIDs {
		seatIDs := make([]string, 0, len(seenSeatByOrg[orgID]))
		for seatID := range seenSeatByOrg[orgID] {
			seatIDs = append(seatIDs, seatID)
		}
		sort.Strings(seatIDs)

		seats := make([]ReceiptSeatStats, 0, len(seatIDs))
		for _, seatID := range seatIDs {
			acc := bySeat[seatKey{orgID: orgID, seatID: seatID}]
			models := make([]string, 0, len(acc.commandedModels))
			for m := range acc.commandedModels {
				models = append(models, m)
			}
			sort.Strings(models)
			seats = append(seats, ReceiptSeatStats{
				SeatID:          seatID,
				CommandedModels: models,
				HonoredTrue:     acc.trueCount,
				HonoredFalse:    acc.falseCount,
				HonoredUnknown:  acc.unknownCount,
			})
		}
		summary.Orgs = append(summary.Orgs, ReceiptOrgStats{OrgID: orgID, Seats: seats})
	}

	return summary
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
