package org

import (
	"encoding/json"
	"path/filepath"
)

// Honored tri-state values for Receipt.Honored (Codex advisory finding 5:
// receipts must be honest, not optimistic). "true" is reserved for cases
// where a driver-observed confirmation of the effective model actually
// exists; "unknown" is the correct default whenever no such observation is
// possible yet -- this package only stores the value; spawn.go's Spawn (and
// its dry-run path) is the current caller deciding which value applies.
const (
	HonoredTrue    = "true"
	HonoredFalse   = "false"
	HonoredUnknown = "unknown"
)

// Receipt records one org seat's model-commanding outcome: what model was
// requested (CommandedModel) versus what the driver reported back
// (ReportedEffectiveModel), and whether the command was verifiably honored.
type Receipt struct {
	TS                     string `json:"ts"`
	OrgID                  string `json:"org_id"`
	SeatID                 string `json:"seat_id"`
	Role                   string `json:"role,omitempty"`
	Driver                 string `json:"driver"`
	CommandedModel         string `json:"commanded_model"`
	ReportedEffectiveModel string `json:"reported_effective_model,omitempty"`
	Honored                string `json:"honored"` // "true" | "false" | "unknown"
	Reason                 string `json:"reason,omitempty"`
}

// ReceiptsRelPath is the default state-root-relative path for the org model
// receipts JSONL store, kept alongside ManifestRelPath's directory.
const ReceiptsRelPath = ".harness/state/org/model-receipts.jsonl"

// ReceiptStore appends to and reads the org model receipts JSONL file.
type ReceiptStore struct {
	path string
}

// NewReceiptStore returns a ReceiptStore rooted at root (typically the
// worktree/state root), using the default ReceiptsRelPath fragment.
func NewReceiptStore(root string) *ReceiptStore {
	return &ReceiptStore{path: filepath.Join(root, ReceiptsRelPath)}
}

// NewReceiptStoreAtPath returns a ReceiptStore backed by an explicit file
// path, bypassing the root-relative default. Primarily useful for tests.
func NewReceiptStoreAtPath(path string) *ReceiptStore {
	return &ReceiptStore{path: path}
}

// Path returns the on-disk path this store reads from and appends to.
func (s *ReceiptStore) Path() string {
	return s.path
}

// Append writes r as a single JSON line, creating the parent directory and
// file as needed (same O_APPEND single-write contract as ManifestStore).
func (s *ReceiptStore) Append(r Receipt) error {
	return appendJSONLine(s.path, r)
}

// ReceiptReadResult is the result of reading the receipts file: the
// successfully parsed receipts in file order, plus a count of lines that
// failed to parse.
type ReceiptReadResult struct {
	Receipts     []Receipt
	CorruptLines int
}

// Read reads every line of the receipts file, skipping and counting corrupt
// lines. A missing receipts file is not an error -- it reads as empty.
func (s *ReceiptStore) Read() (ReceiptReadResult, error) {
	var result ReceiptReadResult
	corrupt, err := readJSONLines(s.path, func(line []byte) error {
		var r Receipt
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		result.Receipts = append(result.Receipts, r)
		return nil
	})
	result.CorruptLines = corrupt
	if err != nil {
		return result, err
	}
	return result, nil
}
