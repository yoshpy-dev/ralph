// Package state reads and models the runtime state of a Ralph pipeline
// from .harness/state/ files (orchestrator.json, checkpoint.json, slice-*.status).
package state

import "time"

// OrchestratorState maps to .harness/state/orchestrator/orchestrator.json.
type OrchestratorState struct {
	Plan          string `json:"plan"`
	Status        string `json:"status"`         // "running" | "complete" | "partial"
	Started       string `json:"started"`         // ISO 8601 timestamp
	Ended         string `json:"ended,omitempty"` // set on completion
	MaxParallel   int    `json:"max_parallel"`
	MaxIterations int    `json:"max_iterations"`
	UnifiedPR     bool   `json:"unified_pr"`
	PRUrl         string `json:"pr_url,omitempty"`
}

// SliceState represents a single slice's runtime status as tracked by the
// orchestrator (slice-<name>.status files + checkpoint.json from the worktree).
type SliceState struct {
	Name           string `json:"name"`
	Status         string `json:"status"` // raw content of slice-*.status file
	Phase          string `json:"phase"`
	Cycle          int    `json:"cycle"`
	ElapsedSeconds int64  `json:"elapsed_seconds"`
	TestResult     string `json:"test_result"`
	PRUrl          string `json:"pr_url"`
}

// PhaseTransition records a phase change in the pipeline checkpoint.
type PhaseTransition struct {
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// CodexTriage holds Codex review statistics from the pipeline checkpoint.
type CodexTriage struct {
	ActionRequired   int `json:"action_required"`
	WorthConsidering int `json:"worth_considering"`
	Dismissed        int `json:"dismissed"`
}

// PipelineCheckpoint maps to .harness/state/pipeline/checkpoint.json
// inside each worktree.
type PipelineCheckpoint struct {
	SchemaVersion int    `json:"schema_version"`
	Iteration     int    `json:"iteration"`
	Phase         string `json:"phase"`   // "preflight" | "inner" | "outer" | "done"
	Status        string `json:"status"`  // "running" | "complete" | "failed"
	InnerCycle    int    `json:"inner_cycle"`
	OuterCycle    int    `json:"outer_cycle"`
	StuckCount    int    `json:"stuck_count"`

	LastTestResult *string  `json:"last_test_result"` // nullable
	TestFailures   []string `json:"test_failures"`
	FailureTriage  []string `json:"failure_triage"`

	SelfReviewResult *string `json:"self_review_result"` // nullable
	VerifyResult     *string `json:"verify_result"`      // nullable
	ReviewFindings   []string `json:"review_findings"`

	CodexTriage CodexTriage `json:"codex_triage"`

	AcceptanceCriteriaMet       []string `json:"acceptance_criteria_met"`
	AcceptanceCriteriaRemaining []string `json:"acceptance_criteria_remaining"`

	SessionID *string `json:"session_id"` // nullable
	PRCreated bool    `json:"pr_created"`
	PRUrl     *string `json:"pr_url"` // nullable

	PhaseTransitions []PhaseTransition `json:"phase_transitions"`
}

// SliceDependency represents a dependency edge between two slices.
type SliceDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Progress holds completion statistics for the orchestrator.
type Progress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
	Percent   int `json:"percent"`
}

// FullStatus combines all pipeline state into one view.
type FullStatus struct {
	Plan           string              `json:"plan"`
	Status         string              `json:"status"`
	ElapsedSeconds int64               `json:"elapsed_seconds"`
	Slices         []SliceState        `json:"slices"`
	Progress       Progress            `json:"progress"`
	Checkpoints    map[string]PipelineCheckpoint `json:"checkpoints"`
	Dependencies   []SliceDependency   `json:"dependencies"`
}

// Elapsed returns the elapsed duration since the orchestrator started.
// Returns zero duration if the timestamp is empty or unparseable.
func (o *OrchestratorState) Elapsed() time.Duration {
	if o.Started == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, o.Started)
	if err != nil {
		return 0
	}
	return time.Since(t)
}
