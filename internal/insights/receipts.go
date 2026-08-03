package insights

import (
	"github.com/yoshpy-dev/ralph/internal/org"
)

// Receipt is the org runtime's model-commanding receipt record
// (internal/org/receipts.go). `ralph insights` reads the exact schema the
// org runtime writes to <state-dir>/model-receipts.jsonl: Honored is a
// tri-state string ("true" | "false" | "unknown"), never collapsed to a
// bool. The retired pipeline-schema Receipt/ReadReceipts (which read
// .harness/state/pipeline/model-receipts.jsonl, written by the now-removed
// Ralph Loop scripts) has been deleted -- that writer no longer exists, so
// there is nothing left to read under the old schema.
type Receipt = org.Receipt

// ReceiptStats holds counters from a ReadReceipts call.
type ReceiptStats struct {
	// LinesRead is the total number of non-empty lines scanned (valid +
	// corrupt).
	LinesRead int
	// SkippedLines is the count of lines that failed JSON unmarshaling.
	SkippedLines int
}

// ReadReceipts reads org runtime model receipts from path, delegating to
// org.NewReceiptStoreAtPath so insights and the org runtime never drift on
// what counts as a valid receipt line. A missing file returns empty
// receipts with no error (graceful degradation, mirroring
// org.ReceiptStore.Read). Corrupt lines are counted in stats.SkippedLines
// and skipped rather than aborting the read.
func ReadReceipts(path string) ([]Receipt, ReceiptStats, error) {
	store := org.NewReceiptStoreAtPath(path)
	result, err := store.Read()
	if err != nil {
		return nil, ReceiptStats{}, err
	}

	stats := ReceiptStats{
		LinesRead:    len(result.Receipts) + result.CorruptLines,
		SkippedLines: result.CorruptLines,
	}
	return result.Receipts, stats, nil
}
