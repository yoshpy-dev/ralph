package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ReadOrchestratorState reads and parses orchestrator.json from the given
// orchestrator state directory (typically .harness/state/orchestrator/).
func ReadOrchestratorState(orchDir string) (*OrchestratorState, error) {
	path := filepath.Join(orchDir, "orchestrator.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading orchestrator state: %w", err)
	}
	var st OrchestratorState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parsing orchestrator.json: %w", err)
	}
	return &st, nil
}

// ReadSliceStatuses reads all slice-*.status files from the orchestrator
// directory and returns a map of slice name to raw status string.
func ReadSliceStatuses(orchDir string) (map[string]string, error) {
	pattern := filepath.Join(orchDir, "slice-*.status")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing slice status files: %w", err)
	}
	result := make(map[string]string, len(matches))
	for _, m := range matches {
		name := extractSliceName(filepath.Base(m))
		data, err := os.ReadFile(m)
		if err != nil {
			return nil, fmt.Errorf("reading slice status %s: %w", name, err)
		}
		result[name] = strings.TrimSpace(string(data))
	}
	return result, nil
}

// ReadPipelineCheckpoint reads checkpoint.json from a slice's worktree.
// worktreeBase is the parent directory containing all worktrees,
// sliceName is the name of the slice (used as subdirectory name).
func ReadPipelineCheckpoint(worktreeBase, sliceName string) (*PipelineCheckpoint, error) {
	path := filepath.Join(worktreeBase, sliceName, ".harness", "state", "pipeline", "checkpoint.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading checkpoint for slice %s: %w", sliceName, err)
	}
	var cp PipelineCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("parsing checkpoint.json for slice %s: %w", sliceName, err)
	}
	return &cp, nil
}

// sliceDepRe matches dependency lines in the manifest like:
//
//	slice-1 (foundation) ──→ slice-2 (watcher)
var sliceDepRe = regexp.MustCompile(`^slice-(\d+)[^─]*──→\s*slice-(\d+)`)

// ReadSliceDependencies parses the dependency graph from the plan manifest.
// It looks for lines matching "slice-N ... ──→ slice-M" in the file.
func ReadSliceDependencies(planDir string) ([]SliceDependency, error) {
	manifest := filepath.Join(planDir, "_manifest.md")
	data, err := os.ReadFile(manifest)
	if err != nil {
		return nil, fmt.Errorf("reading plan manifest: %w", err)
	}

	var deps []SliceDependency
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		// Split on comma to handle "slice-2, slice-4, slice-5 ──→ slice-6"
		parts := splitDependencyLine(line)
		for _, p := range parts {
			matches := sliceDepRe.FindStringSubmatch(p)
			if matches != nil {
				deps = append(deps, SliceDependency{
					From: matches[1],
					To:   matches[2],
				})
			}
		}
	}
	return deps, nil
}

// splitDependencyLine handles lines with multiple sources like
// "slice-2, slice-4, slice-5 ──→ slice-6 (integration)".
// It returns individual "slice-N ──→ slice-M" strings.
func splitDependencyLine(line string) []string {
	// Find the arrow and target
	arrowIdx := strings.Index(line, "──→")
	if arrowIdx < 0 {
		return []string{line}
	}
	target := strings.TrimSpace(line[arrowIdx:])
	sources := strings.TrimSpace(line[:arrowIdx])

	// Split sources by comma
	parts := strings.Split(sources, ",")
	if len(parts) <= 1 {
		return []string{line}
	}

	var result []string
	for _, src := range parts {
		src = strings.TrimSpace(src)
		if src != "" {
			result = append(result, src+" "+target)
		}
	}
	return result
}

// ReadFullStatus assembles the complete pipeline status from all sources.
func ReadFullStatus(orchDir, worktreeBase, planDir string) (*FullStatus, error) {
	orch, err := ReadOrchestratorState(orchDir)
	if err != nil {
		return nil, err
	}

	statuses, err := ReadSliceStatuses(orchDir)
	if err != nil {
		return nil, err
	}

	deps, err := ReadSliceDependencies(planDir)
	if err != nil {
		// Dependencies are optional; don't fail the whole read
		deps = nil
	}

	now := time.Now()
	var elapsed int64
	if orch.Started != "" {
		if t, err := time.Parse(time.RFC3339, orch.Started); err == nil {
			elapsed = int64(now.Sub(t).Seconds())
		}
	}

	completed := 0
	total := len(statuses)
	slices := make([]SliceState, 0, total)
	checkpoints := make(map[string]PipelineCheckpoint, total)

	for name, status := range statuses {
		ss := SliceState{
			Name:   name,
			Status: status,
			Phase:  "unknown",
		}

		if cp, err := ReadPipelineCheckpoint(worktreeBase, name); err == nil {
			ss.Phase = cp.Phase
			ss.Cycle = cp.InnerCycle
			if cp.LastTestResult != nil {
				ss.TestResult = *cp.LastTestResult
			}
			if cp.PRUrl != nil {
				ss.PRUrl = *cp.PRUrl
			}
			// Calculate slice elapsed from first phase transition
			if len(cp.PhaseTransitions) > 0 && cp.PhaseTransitions[0].Timestamp != "" {
				if t, err := time.Parse(time.RFC3339, cp.PhaseTransitions[0].Timestamp); err == nil {
					ss.ElapsedSeconds = int64(now.Sub(t).Seconds())
				}
			}
			checkpoints[name] = *cp
		}

		if status == "complete" {
			completed++
		}
		slices = append(slices, ss)
	}

	pct := 0
	if total > 0 {
		pct = (completed * 100) / total
	}

	return &FullStatus{
		Plan:           orch.Plan,
		Status:         orch.Status,
		ElapsedSeconds: elapsed,
		Slices:         slices,
		Progress: Progress{
			Completed: completed,
			Total:     total,
			Percent:   pct,
		},
		Checkpoints:  checkpoints,
		Dependencies: deps,
	}, nil
}

// extractSliceName extracts the slice name from a filename like "slice-1-foo.status".
// It strips the "slice-" prefix and ".status" suffix.
func extractSliceName(filename string) string {
	name := strings.TrimPrefix(filename, "slice-")
	name = strings.TrimSuffix(name, ".status")
	return name
}
