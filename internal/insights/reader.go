package insights

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

// ReadStats holds counters from a ReadEvents call.
type ReadStats struct {
	// FilesRead is the number of JSONL files processed.
	FilesRead int
	// LinesRead is the total number of non-empty lines scanned.
	LinesRead int
	// SkippedLines is the count of lines that failed JSON unmarshaling.
	SkippedLines int
}

// ReadEvents reads all *.jsonl files from dir and returns their events.
// Corrupt lines are counted in stats.SkippedLines and skipped (non-fatal).
// A missing or empty dir returns an empty slice with no error.
func ReadEvents(dir string) ([]Event, ReadStats, error) {
	var stats ReadStats

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		// Missing dir is graceful degradation: no events, no error.
		return nil, stats, nil
	}

	pattern := filepath.Join(dir, "*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, stats, err
	}

	var events []Event
	for _, f := range files {
		evs, s, err := readJSONLFile(f)
		if err != nil {
			return nil, stats, err
		}
		stats.FilesRead++
		stats.LinesRead += s.LinesRead
		stats.SkippedLines += s.SkippedLines
		events = append(events, evs...)
	}

	return events, stats, nil
}

// readJSONLFile reads one JSONL file, skipping corrupt lines.
func readJSONLFile(path string) ([]Event, ReadStats, error) {
	var stats ReadStats

	f, err := os.Open(path)
	if err != nil {
		return nil, stats, err
	}
	defer func() { _ = f.Close() }()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		stats.LinesRead++

		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			stats.SkippedLines++
			continue
		}
		events = append(events, ev)
	}

	if err := scanner.Err(); err != nil {
		return nil, stats, err
	}

	return events, stats, nil
}
