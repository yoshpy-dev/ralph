package insights

// Event represents one schema-v1 insight event line from a JSONL file.
// Unknown extra fields are tolerated by the JSON decoder (forward-compatible).
// Optional fields are represented as pointers so absence is distinguishable
// from zero-value.
type Event struct {
	Schema         int      `json:"schema"`
	TS             string   `json:"ts"`
	RunID          string   `json:"run_id"`
	Slug           string   `json:"slug"`
	Flow           string   `json:"flow"`
	Phase          string   `json:"phase"`
	Cycle          int      `json:"cycle"`
	Verdict        string   `json:"verdict"`
	Findings       Findings `json:"findings"`
	Triage         Triage   `json:"triage"`
	Driver         string   `json:"driver"`
	RequestedModel string   `json:"requested_model"`
	EffectiveModel string   `json:"effective_model"`
	Honored        bool     `json:"honored"`
	Source         string   `json:"source"`

	// SourceReportPath is a backfill-only field (not in the pipeline-emitted
	// schema). It is documented in docs/insights/README.md backfill section.
	SourceReportPath string `json:"source_report_path,omitempty"`
}

// Findings holds per-severity finding counts for a pipeline phase.
type Findings struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// Total returns the sum of all finding counts.
func (f Findings) Total() int {
	return f.Critical + f.High + f.Medium + f.Low
}

// Triage holds cross-review triage outcome counts.
type Triage struct {
	ActionRequired   int `json:"action_required"`
	WorthConsidering int `json:"worth_considering"`
	Dismissed        int `json:"dismissed"`
}
