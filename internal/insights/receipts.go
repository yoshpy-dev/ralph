package insights

import (
	"bufio"
	"encoding/json"
	"os"
)

// Receipt represents one line from .harness/state/pipeline/model-receipts.jsonl.
// Fields match the write contract in scripts/ralph-cli-driver.sh write_model_receipt.
// Unknown extra fields are tolerated.
type Receipt struct {
	TS             string `json:"ts"`
	Phase          string `json:"phase"`
	Cycle          int    `json:"cycle"`
	Driver         string `json:"driver"`
	RequestedModel string `json:"requested_model"`
	EffectiveModel string `json:"effective_model"`
	Honored        bool   `json:"honored"`
	Effort         string `json:"effort"`
	Reason         string `json:"reason"`
}

// ReceiptStats holds counters from a ReadReceipts call.
type ReceiptStats struct {
	// LinesRead is the total number of non-empty lines scanned.
	LinesRead int
	// SkippedLines is the count of lines that failed JSON unmarshaling.
	SkippedLines int
}

// ReadReceipts reads model receipts from path.
// A missing file returns empty receipts with no error (graceful degradation).
// Corrupt lines are counted in stats.SkippedLines and skipped (non-fatal).
func ReadReceipts(path string) ([]Receipt, ReceiptStats, error) {
	var stats ReceiptStats

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, stats, nil
		}
		return nil, stats, err
	}
	defer func() { _ = f.Close() }()

	var receipts []Receipt
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		stats.LinesRead++

		var r Receipt
		if err := json.Unmarshal(line, &r); err != nil {
			stats.SkippedLines++
			continue
		}
		receipts = append(receipts, r)
	}

	if err := scanner.Err(); err != nil {
		return nil, stats, err
	}

	return receipts, stats, nil
}
