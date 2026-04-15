package state

import "time"

// SliceStatus represents the execution status of a slice.
type SliceStatus string

const (
	StatusComplete    SliceStatus = "complete"
	StatusRunning     SliceStatus = "running"
	StatusPending     SliceStatus = "pending"
	StatusFailed      SliceStatus = "failed"
	StatusStuck       SliceStatus = "stuck"
	StatusAborted     SliceStatus = "aborted"
	StatusRepairLimit SliceStatus = "repair_limit"
	StatusConfigError SliceStatus = "config_error"
	StatusMaxRetries  SliceStatus = "max_retries"
)

// SliceState holds the current state of a single slice.
type SliceState struct {
	Name       string      `json:"name"`
	Status     SliceStatus `json:"status"`
	Phase      string      `json:"phase"`
	Cycle      int         `json:"cycle"`
	MaxCycles  int         `json:"max_cycles"`
	Elapsed    int         `json:"elapsed"`
	TestResult string      `json:"test_result"`
	PRURL      string      `json:"pr_url"`
	PID        int         `json:"pid"`
	StartedAt  *time.Time  `json:"started_at,omitempty"`
}

// SliceDependency represents a dependency between two slices.
type SliceDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// OrchestratorState holds the top-level orchestrator state.
type OrchestratorState struct {
	Plan              string       `json:"plan"`
	Status            string       `json:"status"`
	StartedAt         string       `json:"started"`
	EndedAt           string       `json:"ended"`
	Slices            []SliceState `json:"slices"`
	IntegrationBranch string       `json:"integration_branch"`
}

// PipelineCheckpoint holds the pipeline checkpoint state for a slice.
type PipelineCheckpoint struct {
	Phase            string `json:"phase"`
	Status           string `json:"status"`
	InnerCycle       int    `json:"inner_cycle"`
	OuterCycle       int    `json:"outer_cycle"`
	Iteration        int    `json:"iteration"`
	SelfReviewResult string `json:"self_review_result"`
	VerifyResult     string `json:"verify_result"`
	LastTestResult   string `json:"last_test_result"`
}

// FullStatus combines orchestrator state with per-slice pipeline details.
type FullStatus struct {
	Orchestrator OrchestratorState             `json:"orchestrator"`
	Checkpoints  map[string]PipelineCheckpoint `json:"checkpoints"`
	Dependencies []SliceDependency             `json:"dependencies"`
}
