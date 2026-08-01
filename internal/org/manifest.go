package org

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// ManifestEvent is a single JSONL record appended to the org manifest. One
// event = one line. This shape is a contract: later slices (driver
// adapters, `ralph org` verbs) depend on these exact field names and JSON
// tags, so changes here must be deliberate and lock-stepped across slices.
type ManifestEvent struct {
	TS        string `json:"ts"`      // UTC RFC3339
	OrgID     string `json:"org_id"`  // execution namespace, required
	SeatID    string `json:"seat_id"` // required except org-level events (disbanded)
	Event     string `json:"event"`
	Role      string `json:"role,omitempty"`
	Driver    string `json:"driver,omitempty"`
	Model     string `json:"model,omitempty"`
	Worktree  string `json:"worktree,omitempty"`
	PaneID    string `json:"pane_id,omitempty"` // herdr external id, persisted as soon as known
	AgmsgTeam string `json:"agmsg_team,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
	Details   string `json:"details,omitempty"` // free text: rejection reason, compensation result, etc.
}

// ManifestRelPath is the default state-root-relative path for the org
// manifest JSONL store.
const ManifestRelPath = ".harness/state/org/manifest.jsonl"

// ManifestStore appends to and reads the org manifest JSONL file.
type ManifestStore struct {
	path string
}

// NewManifestStore returns a ManifestStore rooted at root (typically the
// worktree/state root), using the default ManifestRelPath fragment.
func NewManifestStore(root string) *ManifestStore {
	return &ManifestStore{path: filepath.Join(root, ManifestRelPath)}
}

// NewManifestStoreAtPath returns a ManifestStore backed by an explicit file
// path, bypassing the root-relative default. Primarily useful for tests.
func NewManifestStoreAtPath(path string) *ManifestStore {
	return &ManifestStore{path: path}
}

// Path returns the on-disk path this store reads from and appends to.
func (s *ManifestStore) Path() string {
	return s.path
}

// Append writes ev as a single JSON line, creating the parent directory and
// file as needed. Uses O_APPEND|O_CREATE with a single write() call per
// event so that concurrent appenders (goroutines or processes) do not
// interleave partial lines.
func (s *ManifestStore) Append(ev ManifestEvent) error {
	return appendJSONLine(s.path, ev)
}

// ManifestReadResult is the result of reading the org manifest: the
// successfully parsed events in file order, plus a count of lines that
// failed to parse (skipped rather than aborting the whole read, so a single
// corrupt line cannot take down `status`).
type ManifestReadResult struct {
	Events       []ManifestEvent
	CorruptLines int
}

// Read reads every line of the manifest file, skipping and counting corrupt
// lines. A missing manifest file is not an error -- it reads as empty,
// since `ralph org status` must work before any seat has ever spawned.
func (s *ManifestStore) Read() (ManifestReadResult, error) {
	var result ManifestReadResult
	corrupt, err := readJSONLines(s.path, func(line []byte) error {
		var ev ManifestEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return err
		}
		result.Events = append(result.Events, ev)
		return nil
	})
	result.CorruptLines = corrupt
	if err != nil {
		return result, err
	}
	return result, nil
}

// RosterOptions controls roster/status derivation from manifest events.
type RosterOptions struct {
	// IncludeDryRun, when true, includes seats whose latest applicable
	// event was recorded during a --dry-run invocation. The default
	// (false) excludes dry-run events from both the roster and
	// max_seats/ActiveSeatCount accounting -- dry-run audit trail stays
	// visible only via `status --all` (AC-8, dry-run audit separation).
	IncludeDryRun bool
}

// seatKey identifies a seat within the org_id namespace it was spawned in.
// The same seat_id in two different org_id namespaces is never conflated
// (AC-2).
type seatKey struct {
	OrgID  string
	SeatID string
}

// Roster derives per-(org_id, seat_id) SeatStatus from a sequence of
// manifest events, purely in memory (no I/O). Events are assumed to be in
// file/append order (oldest first); "latest" is determined by that order,
// not by parsing the TS field, so ties at second-level RFC3339 resolution
// do not produce ambiguous results.
//
// An org-level `disbanded` event (SeatID == "") marks every seat in that
// org_id inactive from that point forward, regardless of the seat's own
// latest saga event.
func Roster(events []ManifestEvent, opts RosterOptions) []SeatStatus {
	type seatEntry struct {
		ev  ManifestEvent
		idx int
	}
	latest := make(map[seatKey]seatEntry)
	disbandedIdx := make(map[string]int)

	for i, ev := range events {
		if !opts.IncludeDryRun && ev.DryRun {
			continue
		}
		if ev.SeatID == "" {
			if ev.Event == EventDisbanded {
				disbandedIdx[ev.OrgID] = i
			}
			continue
		}
		// Only STATE events drive seat status derivation (see seat.go
		// stateEvents). A non-state event (e.g. `sent`) must never
		// overwrite or clear a seat's latest derived status -- it is
		// still fully readable via ManifestStore.Read for history, just
		// not a candidate for "latest applicable event" here.
		if !isStateEvent(ev.Event) {
			continue
		}
		latest[seatKey{OrgID: ev.OrgID, SeatID: ev.SeatID}] = seatEntry{ev: ev, idx: i}
	}

	result := make([]SeatStatus, 0, len(latest))
	for key, entry := range latest {
		active := isActiveEvent(entry.ev.Event)
		if dIdx, ok := disbandedIdx[key.OrgID]; ok && dIdx > entry.idx {
			active = false
		}
		result = append(result, SeatStatus{
			OrgID:     entry.ev.OrgID,
			SeatID:    entry.ev.SeatID,
			Role:      entry.ev.Role,
			Driver:    entry.ev.Driver,
			Model:     entry.ev.Model,
			Worktree:  entry.ev.Worktree,
			PaneID:    entry.ev.PaneID,
			AgmsgTeam: entry.ev.AgmsgTeam,
			Event:     entry.ev.Event,
			Active:    active,
			DryRun:    entry.ev.DryRun,
			Details:   entry.ev.Details,
			TS:        entry.ev.TS,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].OrgID != result[j].OrgID {
			return result[i].OrgID < result[j].OrgID
		}
		return result[i].SeatID < result[j].SeatID
	})
	return result
}

// ActiveSeatCount returns the number of seats within orgID whose derived
// status is Active, honoring opts.IncludeDryRun. This is a pure function of
// already-read events; see (*ManifestStore).ActiveSeatCount for the
// convenience method that reads the store first.
func ActiveSeatCount(events []ManifestEvent, orgID string, opts RosterOptions) int {
	count := 0
	for _, s := range Roster(events, opts) {
		if s.OrgID == orgID && s.Active {
			count++
		}
	}
	return count
}

// Roster reads the manifest and derives SeatStatus for every seat, per the
// package-level Roster function.
func (s *ManifestStore) Roster(opts RosterOptions) ([]SeatStatus, error) {
	rr, err := s.Read()
	if err != nil {
		return nil, err
	}
	return Roster(rr.Events, opts), nil
}

// ActiveSeatCount reads the manifest and returns the count of active seats
// in orgID, honoring opts.IncludeDryRun. Callers computing the activeSeats
// argument for ValidateSpawn should use this method.
func (s *ManifestStore) ActiveSeatCount(orgID string, opts RosterOptions) (int, error) {
	rr, err := s.Read()
	if err != nil {
		return 0, err
	}
	return ActiveSeatCount(rr.Events, orgID, opts), nil
}

// appendJSONLine appends v as a single JSON line to the file at path,
// creating parent directories and the file as needed.
func appendJSONLine(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("org: create dir for %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("org: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	line, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("org: marshal %T: %w", v, err)
	}
	line = append(line, '\n')
	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("org: append to %s: %w", path, err)
	}
	return nil
}

// readJSONLines reads path line by line, calling unmarshal for each
// non-blank line. unmarshal should decode the line and return an error only
// on parse failure; readJSONLines counts and skips lines that fail to
// unmarshal instead of aborting the whole read, so one corrupt line cannot
// take down status/roster derivation. A missing file reads as zero lines,
// not an error.
func readJSONLines(path string, unmarshal func(line []byte) error) (corrupt int, err error) {
	f, openErr := os.Open(path)
	if openErr != nil {
		if errors.Is(openErr, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("org: open %s: %w", path, openErr)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		if uErr := unmarshal(line); uErr != nil {
			corrupt++
			continue
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return corrupt, fmt.Errorf("org: scan %s: %w", path, scanErr)
	}
	return corrupt, nil
}
