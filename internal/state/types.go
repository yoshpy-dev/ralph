package state

import "time"

// SliceStatus represents the execution status of a slice.
type SliceStatus = string

const (
	StatusComplete      SliceStatus = "complete"
	StatusRunning       SliceStatus = "running"
	StatusPending       SliceStatus = "pending"
	StatusFailed        SliceStatus = "failed"
	StatusStuck         SliceStatus = "stuck"
	StatusAborted       SliceStatus = "aborted"
	StatusRepairLimit   SliceStatus = "repair_limit"
	StatusConfigError   SliceStatus = "config_error"
	StatusMaxRetries    SliceStatus = "max_retries"
	StatusTimeout       SliceStatus = "timeout"
	StatusGhUnavailable SliceStatus = "gh_unavailable"
)

// OrchestratorState represents the orchestrator.json file written by ralph-orchestrator.sh.
type OrchestratorState struct {
	Plan              string `json:"plan"`
	Started           string `json:"started"`
	Ended             string `json:"ended,omitempty"`
	MaxParallel       int    `json:"max_parallel"`
	MaxIterations     int    `json:"max_iterations"`
	UnifiedPR         bool   `json:"unified_pr"`
	Status            string `json:"status"`
	PRUrl             string `json:"pr_url,omitempty"`
	IntegrationBranch string `json:"integration_branch,omitempty"`
}

// SliceState represents the status of a single slice, read from slice-<name>.status files
// and enriched with checkpoint data.
type SliceState struct {
	Name         string              `json:"name"`
	Status       SliceStatus         `json:"status"`
	Phase        string              `json:"phase"`
	Cycle        int                 `json:"cycle"`
	MaxCycles    int                 `json:"max_cycles"`
	Elapsed      int                 `json:"elapsed"`
	TestResult   string              `json:"test_result"`
	PRURL        string              `json:"pr_url"`
	PID          int                 `json:"pid,omitempty"`
	StartedAt    *time.Time          `json:"started_at,omitempty"`
	LogPath      string              `json:"log_path,omitempty"`
	WorktreePath string              `json:"worktree_path,omitempty"`
	Checkpoint   *PipelineCheckpoint `json:"checkpoint,omitempty"`
}

// CanRetry returns true if the slice is in a retryable status.
func (s *SliceState) CanRetry() bool {
	switch s.Status {
	case StatusFailed, StatusStuck, StatusRepairLimit, StatusMaxRetries, StatusTimeout, StatusGhUnavailable:
		return true
	}
	return false
}

// CanAbort returns true if the slice is in an abortable status.
func (s *SliceState) CanAbort() bool {
	return s.Status == StatusRunning
}

// HasLogs returns true if the slice has a log path.
func (s *SliceState) HasLogs() bool {
	return s.LogPath != ""
}

// HasWorktree returns true if the slice has a worktree path.
func (s *SliceState) HasWorktree() bool {
	return s.WorktreePath != ""
}

// PipelineCheckpoint represents .harness/state/pipeline/checkpoint.json.
type PipelineCheckpoint struct {
	SchemaVersion               int               `json:"schema_version"`
	Iteration                   int               `json:"iteration"`
	Phase                       string            `json:"phase"`
	Status                      string            `json:"status"`
	InnerCycle                  int               `json:"inner_cycle"`
	OuterCycle                  int               `json:"outer_cycle"`
	StuckCount                  int               `json:"stuck_count"`
	LastTestResult              *string           `json:"last_test_result"`
	TestFailures                []string          `json:"test_failures"`
	FailureTriage               []FailureEntry    `json:"failure_triage"`
	SelfReviewResult            *string           `json:"self_review_result"`
	VerifyResult                *string           `json:"verify_result"`
	ReviewFindings              []string          `json:"review_findings"`
	CrossReviewTriage           CrossReviewTriage `json:"cross_review_triage"`
	AcceptanceCriteriaMet       []string          `json:"acceptance_criteria_met"`
	AcceptanceCriteriaRemaining []string          `json:"acceptance_criteria_remaining"`
	SessionID                   *string           `json:"session_id"`
	PRCreated                   bool              `json:"pr_created"`
	PRUrl                       *string           `json:"pr_url"`
	PhaseTransitions            []PhaseTransition `json:"phase_transitions"`
}

// FailureEntry represents an entry in the failure_triage array.
type FailureEntry struct {
	Iteration  int    `json:"iteration"`
	Hypothesis string `json:"hypothesis"`
	Action     string `json:"action"`
	Result     string `json:"result"`
}

// CrossReviewTriage represents the cross_review_triage object in checkpoint.json.
type CrossReviewTriage struct {
	ActionRequired   int `json:"action_required"`
	WorthConsidering int `json:"worth_considering"`
	Dismissed        int `json:"dismissed"`
}

// PhaseTransition represents a phase transition entry.
type PhaseTransition struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Timestamp string `json:"timestamp"`
	Reason    string `json:"reason,omitempty"`
}

// SliceDependency represents a dependency edge between two slices.
type SliceDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// FullStatus aggregates the orchestrator state, all slice states,
// and dependency edges into a single view.
type FullStatus struct {
	Orchestrator *OrchestratorState `json:"orchestrator"`
	Slices       []SliceState       `json:"slices"`
	Dependencies []SliceDependency  `json:"dependencies"`
	Progress     Progress           `json:"progress"`
}

// Progress summarizes overall completion.
type Progress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
	Percent   int `json:"percent"`
}

// StartedTime parses the Started field of OrchestratorState.
func (o *OrchestratorState) StartedTime() (time.Time, error) {
	return parseTimestamp(o.Started)
}

// EndedTime parses the Ended field of OrchestratorState.
func (o *OrchestratorState) EndedTime() (time.Time, error) {
	return parseTimestamp(o.Ended)
}

// FirstTransitionTime returns the timestamp of the first phase transition.
func (c *PipelineCheckpoint) FirstTransitionTime() (time.Time, error) {
	if len(c.PhaseTransitions) == 0 {
		return time.Time{}, nil
	}
	return parseTimestamp(c.PhaseTransitions[0].Timestamp)
}

// parseTimestamp handles ISO 8601 timestamps with or without trailing Z.
func parseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	// Try RFC3339 first (2006-01-02T15:04:05Z or 2006-01-02T15:04:05+00:00)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try without timezone (assume UTC)
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, &time.ParseError{Value: s, Message: "unrecognized timestamp format"}
}
