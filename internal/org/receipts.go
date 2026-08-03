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

// ReceiptStore appends to and reads the org model receipts JSONL file.
type ReceiptStore struct {
	path string
}

// ReceiptsPathIn returns the model-receipts.jsonl path within an
// already-resolved org state directory, mirroring ManifestPathIn. This is
// THE single derivation of a receipts path from a resolved state dir --
// every caller (internal/cli/org.go's write path and tests) must go through
// this function instead of re-deriving the join themselves. The
// root-relative constructor this package used to export (joining a
// caller-supplied root against a package-level relative-path constant) was
// removed for the same reason as the manifest-side one: passing
// an already-resolved state dir into a root-relative constructor
// double-joins the relative fragment, the exact class of bug behind AR-1
// (docs/reports/cross-review-triage-org-runtime-retire-loop.md, manifest
// side). Centralizing the join here removes the ambiguity for receipts too.
func ReceiptsPathIn(stateDir string) string {
	return filepath.Join(stateDir, "model-receipts.jsonl")
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
