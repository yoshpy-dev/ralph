package org

import (
	"testing"
)

func newTestReceiptStore(t *testing.T) *ReceiptStore {
	t.Helper()
	dir := t.TempDir()
	return NewReceiptStoreAtPath(ReceiptsPathIn(dir))
}

func TestReceiptStore_AppendReadRoundTrip(t *testing.T) {
	store := newTestReceiptStore(t)

	receipts := []Receipt{
		{
			TS:                     "2026-08-01T00:00:00Z",
			OrgID:                  "org-a",
			SeatID:                 "seat-1",
			Driver:                 "claude",
			CommandedModel:         "sonnet",
			Honored:                HonoredTrue,
			ReportedEffectiveModel: "sonnet",
		},
		{
			TS:             "2026-08-01T00:00:01Z",
			OrgID:          "org-a",
			SeatID:         "seat-2",
			Driver:         "claude",
			CommandedModel: "opus",
			Honored:        HonoredFalse,
			Reason:         "model_pool rejected before spawn",
		},
		{
			TS:             "2026-08-01T00:00:02Z",
			OrgID:          "org-a",
			SeatID:         "seat-3",
			Driver:         "codex",
			CommandedModel: "gpt-5-codex",
			Honored:        HonoredUnknown,
			Reason:         "no driver-observed confirmation available",
		},
	}

	for _, r := range receipts {
		if err := store.Append(r); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	result, err := store.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result.CorruptLines != 0 {
		t.Fatalf("expected 0 corrupt lines, got %d", result.CorruptLines)
	}
	if len(result.Receipts) != len(receipts) {
		t.Fatalf("expected %d receipts, got %d", len(receipts), len(result.Receipts))
	}
	for i, r := range receipts {
		if result.Receipts[i] != r {
			t.Errorf("receipt %d mismatch: want %+v, got %+v", i, r, result.Receipts[i])
		}
	}
}

func TestReceiptStore_TriStateHonoredValues(t *testing.T) {
	store := newTestReceiptStore(t)

	for _, honored := range []string{HonoredTrue, HonoredFalse, HonoredUnknown} {
		if err := store.Append(Receipt{
			TS:             "2026-08-01T00:00:00Z",
			OrgID:          "org-a",
			SeatID:         "seat-1",
			Driver:         "claude",
			CommandedModel: "sonnet",
			Honored:        honored,
		}); err != nil {
			t.Fatalf("Append failed for honored=%q: %v", honored, err)
		}
	}

	result, err := store.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if len(result.Receipts) != 3 {
		t.Fatalf("expected 3 receipts, got %d", len(result.Receipts))
	}
	want := []string{HonoredTrue, HonoredFalse, HonoredUnknown}
	for i, r := range result.Receipts {
		if r.Honored != want[i] {
			t.Errorf("receipt %d: expected honored=%q, got %q", i, want[i], r.Honored)
		}
	}
}

func TestReceiptStore_Read_MissingFileIsEmpty(t *testing.T) {
	store := newTestReceiptStore(t)
	result, err := store.Read()
	if err != nil {
		t.Fatalf("expected no error reading missing receipts file, got: %v", err)
	}
	if len(result.Receipts) != 0 || result.CorruptLines != 0 {
		t.Fatalf("expected empty result for missing receipts file, got: %+v", result)
	}
}
