package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// --- fixture helpers -------------------------------------------------
//
// These build minimal lines shaped like real codex-cli 0.154.0 rollout
// records (session_meta / turn_context / response_item), per the plan's
// Assumptions section. Every helper goes through encoding/json so fixture
// content (including promptPath, which contains '/') is always correctly
// escaped -- no hand-rolled JSON string concatenation.

// rfc3339Milli formats t the way real records do: UTC, millisecond
// precision, "Z" suffix (e.g. "2026-09-18T07:13:57.831Z").
func rfc3339Milli(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// fixtureDateDir returns the YYYY/MM/DD directory under sessionsDir that
// codexSessionDateDirs would use for at, mirroring codex's own layout.
func fixtureDateDir(sessionsDir string, at time.Time) string {
	return filepath.Join(sessionsDir, at.Local().Format(codexSessionDateLayout))
}

// writeRolloutFile joins lines with newlines (plus a trailing newline) and
// writes them to dir/name, creating dir as needed.
func writeRolloutFile(t *testing.T, dir, name string, lines []string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, name)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
	return path
}

type fixtureLine struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   any    `json:"payload"`
	Ordinal   int    `json:"ordinal"`
}

func marshalFixtureLine(t *testing.T, ts, typ string, payload any, ordinal int) string {
	t.Helper()
	b, err := json.Marshal(fixtureLine{Timestamp: ts, Type: typ, Payload: payload, Ordinal: ordinal})
	if err != nil {
		t.Fatalf("marshal fixture line: %v", err)
	}
	return string(b)
}

func sessionMetaLine(t *testing.T, ts string) string {
	return marshalFixtureLine(t, ts, codexLineTypeSessionMeta, map[string]string{"timestamp": ts}, 0)
}

func turnContextLine(t *testing.T, ts, model string) string {
	return marshalFixtureLine(t, ts, codexLineTypeTurnContext, map[string]string{"model": model, "effort": "medium"}, 1)
}

func userMessageLine(t *testing.T, ts, text string) string {
	payload := map[string]any{
		"type": "message",
		"role": "user",
		"content": []map[string]string{
			{"type": "input_text", "text": text},
		},
	}
	return marshalFixtureLine(t, ts, codexLineTypeResponseItem, payload, 2)
}

func assistantMessageLine(t *testing.T, ts, text string) string {
	payload := map[string]any{
		"type": "message",
		"role": "assistant",
		"content": []map[string]string{
			{"type": "output_text", "text": text},
		},
	}
	return marshalFixtureLine(t, ts, codexLineTypeResponseItem, payload, 2)
}

// --- countingCtx: deterministic, call-count-based fake context.Context ---
//
// ObserveCodexEffectiveModel checks ctx.Err() at specific, documented
// points inside its own pass (candidate collection, before
// opening each candidate, once per line while reading one). Proving those
// checks land exactly where documented -- not one call early, not one call
// late -- needs a ctx whose "done" transition is pinned to an exact call
// count, not to wall-clock timing (which cannot deterministically land a
// cutoff between two specific internal steps that both complete in
// microseconds). countingCtx's Err() returns nil for the first n calls,
// then context.DeadlineExceeded for every call after; Done() closes at the
// same moment Err() first reports non-nil, so anything that selects on
// Done() (none of codex_session.go's own checks do, but the interface
// contract still holds) sees the same transition.
type countingCtx struct {
	n     int
	calls int
	done  chan struct{}
}

func newCountingCtx(n int) *countingCtx {
	return &countingCtx{n: n, done: make(chan struct{})}
}

func (c *countingCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *countingCtx) Done() <-chan struct{}       { return c.done }
func (c *countingCtx) Value(any) any               { return nil }
func (c *countingCtx) Err() error {
	c.calls++
	if c.calls > c.n {
		select {
		case <-c.done:
		default:
			close(c.done)
		}
		return context.DeadlineExceeded
	}
	return nil
}

// TestObserveCodexEffectiveModel_AlreadyDoneCtx_NotFoundDespiteMatchingRecord
// is AC-3: a ctx that is already done reports not-found -- alongside ctx's
// own error -- even when a fully qualifying record sits on disk, proving
// the pass never actually scanned anything (the observer's own first
// ctx.Err() check, inside candidate collection, is what stops it).
func TestObserveCodexEffectiveModel_AlreadyDoneCtx_NotFoundDespiteMatchingRecord(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-matching.jsonl", lines)

	ctx := newCountingCtx(0) // done on the very first Err() call
	obs, err := ObserveCodexEffectiveModel(ctx, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error, got %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (an already-done ctx must never report found, even with a matching record on disk)", obs)
	}
}

// TestObserveCodexEffectiveModel_AlreadyDoneCtx_ZeroCandidates_ReturnsCtxError
// is a /test cycle-1 addition (issue #173 handoff item 10: "a ctx whose
// deadline is already past but with zero candidates"). The test above
// proves an already-done ctx wins even when a matching record sits on
// disk; this one proves the SAME ctx error -- never a plain nil "clean
// not-found" -- comes back when sessionsDir exists but is genuinely empty
// (no rollout files anywhere), so a caller can never mistake "ctx cut this
// short" for "genuinely nothing here" by checking err alone. This is
// reachable because codexRolloutCandidates' per-date-directory ctx.Err()
// check runs on the very first iteration of codexSessionDateDirs' own
// (always non-empty) list, before it ever tries to read a directory that
// may not exist on disk.
func TestObserveCodexEffectiveModel_AlreadyDoneCtx_ZeroCandidates_ReturnsCtxError(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessionsDir: %v", err)
	}
	// sessionsDir exists (passes the earlier os.Stat check) but holds no
	// date directories and no rollout files at all.

	ctx := newCountingCtx(0) // done on the very first Err() call
	obs, err := ObserveCodexEffectiveModel(ctx, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error even with zero candidates, got %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found", obs)
	}
}

// TestObserveCodexEffectiveModel_CtxDoneBeforeSecondCandidate_SecondFileNeverOpened
// is AC-4: two candidates exist, sorted newest-first; the newer one
// matches. ctx becomes done at the exact moment the older (second)
// candidate would be opened -- calibrated against real calls (collection
// count via codexRolloutCandidates directly, plus the newer candidate's
// own scanRolloutRecord call count), not a hardcoded magic number, so this
// stays correct even if the exact checkpoints inside a pass shift, as long
// as candidate collection still completes before any candidate is opened.
// The older candidate's own content would ALSO qualify if actually read --
// proving via the final status (not-found, never ambiguous) that it was
// never opened at all, not merely that its match didn't end up counting.
func TestObserveCodexEffectiveModel_CtxDoneBeforeSecondCandidate_SecondFileNeverOpened(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)

	newer := spawnStarted.Add(2 * time.Second)
	newerLines := []string{
		sessionMetaLine(t, rfc3339Milli(newer)),
		userMessageLine(t, rfc3339Milli(newer.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(newer.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	newerPath := writeRolloutFile(t, fixtureDateDir(sessionsDir, newer), "rollout-newer.jsonl", newerLines)
	newerModTime := newer.Add(10 * time.Second)
	if err := os.Chtimes(newerPath, newerModTime, newerModTime); err != nil {
		t.Fatalf("chtimes newer: %v", err)
	}

	older := spawnStarted.Add(1 * time.Second)
	olderLines := []string{
		sessionMetaLine(t, rfc3339Milli(older)),
		userMessageLine(t, rfc3339Milli(older.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(older.Add(300*time.Millisecond)), "gpt-9.9-would-make-this-ambiguous"),
	}
	olderPath := writeRolloutFile(t, fixtureDateDir(sessionsDir, older), "rollout-older.jsonl", olderLines)
	olderModTime := older.Add(5 * time.Second)
	if err := os.Chtimes(olderPath, olderModTime, olderModTime); err != nil {
		t.Fatalf("chtimes older: %v", err)
	}

	// Calibrate: collection sees both files (both must exist on disk for
	// the directory-entry count to match the real run below); the newer
	// one must sort first.
	collectCtx := newCountingCtx(1 << 30)
	candidates, err := codexRolloutCandidates(collectCtx, sessionsDir, spawnStarted, spawnStarted)
	if err != nil || len(candidates) != 2 {
		t.Fatalf("test setup invariant broken: candidates=%+v err=%v", candidates, err)
	}
	if candidates[0].path != newerPath {
		t.Fatalf("test setup invariant broken: expected the newer file to sort first, got %q", candidates[0].path)
	}
	collectionCalls := collectCtx.calls

	// Calibrate: how many ctx.Err() calls a full, successful scan of the
	// newer candidate alone makes.
	scanCtx := newCountingCtx(1 << 30)
	matched, model, err := scanRolloutRecord(scanCtx, newerPath, promptPath, spawnStarted.Truncate(time.Second))
	if err != nil || !matched || model != "gpt-5.6-sol" {
		t.Fatalf("test setup invariant broken: matched=%v model=%q err=%v", matched, model, err)
	}
	scanCalls := scanCtx.calls

	// The real run: every call up through finishing the newer candidate's
	// scan succeeds; the very next call -- scanRolloutRecord's own
	// before-open check for the older candidate -- is the first to fail.
	ctx := newCountingCtx(collectionCalls + scanCalls)
	obs, err := ObserveCodexEffectiveModel(ctx, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error, got %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (cut short before the older candidate could be opened -- ambiguous would mean it was read after all)", obs)
	}
}

// TestObserveCodexEffectiveModel_CtxDoneDuringCandidateCollection_NotFound
// is AC-4b's first clause: SOME ctx.Err() check inside candidate
// collection -- not a check made while opening or reading any file's
// content -- is enough to stop the whole pass early, returning not-found
// and ctx's own error. It does not by itself prove WHICH checkpoint
// (per-directory or per-entry) is the one doing the work: with n=3, the
// cut lands on the second per-entry check inside the "day" directory
// (day-1's own per-directory check succeeds since that directory doesn't
// exist and is skipped; day's own per-directory check succeeds; the first
// entry's per-entry check succeeds; the second entry's fails) -- day+1 is
// never reached at all, since the walk is ascending (day-1, day, day+1)
// and the cutoff happens before it. Placement itself is pinned by
// TestCodexRolloutCandidates_PerDateDirectoryCheckIsPinned_AlreadyDoneCtx
// and TestCodexRolloutCandidates_PerEntryCheckIsPinned_CtxDoneAtLastDirectory
// below, each of which fails if its own specific check is removed.
func TestObserveCodexEffectiveModel_CtxDoneDuringCandidateCollection_NotFound(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	// Many entries -- content is irrelevant, since candidate collection
	// only Lstats each entry; it never opens or reads file content.
	dateDir := fixtureDateDir(sessionsDir, start)
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("rollout-filler-%02d.jsonl", i)
		if err := os.WriteFile(filepath.Join(dateDir, name), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write filler %d: %v", i, err)
		}
	}

	ctx := newCountingCtx(3)
	obs, err := ObserveCodexEffectiveModel(ctx, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error, got %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (cut short mid candidate-collection)", obs)
	}
}

// TestCodexRolloutCandidates_PerDateDirectoryCheckIsPinned_AlreadyDoneCtx
// is M2's first placement-pinning test: with an already-done ctx and a
// sessions directory whose date directories don't even exist (nothing to
// Lstat, no entries loop ever reached), codexRolloutCandidates must still
// report ctx's own error. Without the per-directory ctx.Err() check, this
// call has nothing else to trip on -- every os.ReadDir fails silently and
// is skipped (a missing date directory is not an error, per this
// function's own doc comment) -- so it would walk straight through the
// whole span and return (nil, nil), never reporting that it was actually
// cut short. Deleting the per-directory check reproduces exactly that
// (verified: temporarily removed the check, this test failed with
// err=<nil>, restored it, green again).
func TestCodexRolloutCandidates_PerDateDirectoryCheckIsPinned_AlreadyDoneCtx(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions") // never created: no date directory exists
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)

	ctx := newCountingCtx(0) // already done on the very first Err() call
	candidates, err := codexRolloutCandidates(ctx, sessionsDir, spawnStarted, spawnStarted)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error, got %v", err)
	}
	if candidates != nil {
		t.Fatalf("expected no candidates on a cut-short walk, got %+v", candidates)
	}
}

// TestCodexRolloutCandidates_PerEntryCheckIsPinned_CtxDoneAtLastDirectory
// is M2's second placement-pinning test: entries exist ONLY in the LAST
// date directory the walk reaches (codexSessionDateDirs' own ascending
// order: spawnStarted's date minus one day, ..., until's date plus one
// day) -- every EARLIER directory doesn't exist, so its own
// per-directory check is the only ctx.Err() call it costs (os.ReadDir
// fails and is skipped, per-entry never runs for a directory with no
// entries loop reached). N is derived from codexSessionDateDirs itself,
// not hard-coded, so this test survives a change to how many directories
// the window spans: with N = len(dirs) calls already spent (one per
// directory), the very next call is the FIRST per-entry check, inside the
// last directory's own entries loop. Without that check,
// codexRolloutCandidates would finish listing the last directory and
// return its entries with err=nil (verified: temporarily removed the
// check, this test failed with err=<nil> and a non-nil candidates slice,
// restored it, green again).
func TestCodexRolloutCandidates_PerEntryCheckIsPinned_CtxDoneAtLastDirectory(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	until := spawnStarted

	dirs := codexSessionDateDirs(spawnStarted, until)
	lastDir := filepath.Join(sessionsDir, dirs[len(dirs)-1])
	if err := os.MkdirAll(lastDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("rollout-filler-%02d.jsonl", i)
		if err := os.WriteFile(filepath.Join(lastDir, name), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write filler %d: %v", i, err)
		}
	}

	ctx := newCountingCtx(len(dirs))
	candidates, err := codexRolloutCandidates(ctx, sessionsDir, spawnStarted, until)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error, got %v", err)
	}
	if candidates != nil {
		t.Fatalf("expected no candidates on a cut-short walk, got %+v", candidates)
	}
}

// TestScanRolloutRecord_BeforeOpenCheckIsPinned_AlreadyDoneCtx is M2's
// third placement-pinning test: an already-done ctx plus a path that does
// NOT exist. With the before-open check, scanRolloutRecord reports ctx's
// own error. Without it, os.Open's own ErrNotExist is what this function
// reaches next, mapped (deliberately, for a real missing-file race) to
// matched=false, err=nil -- observably identical to "nothing found", so a
// removed check would silently disappear here rather than fail loudly
// (verified: temporarily removed the check, this test failed with
// err=<nil>, restored it, green again).
func TestScanRolloutRecord_BeforeOpenCheckIsPinned_AlreadyDoneCtx(t *testing.T) {
	dir := t.TempDir()
	nonexistentPath := filepath.Join(dir, "does-not-exist.jsonl")

	ctx := newCountingCtx(0) // already done on the very first Err() call
	matched, model, err := scanRolloutRecord(ctx, nonexistentPath, "/prompts/x.md", time.Now())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error, got %v", err)
	}
	if matched || model != "" {
		t.Fatalf("expected matched=false, model=\"\", got matched=%v model=%q", matched, model)
	}
}

// TestObserveCodexEffectiveModel_CtxDoneMidFileRead_NotFoundEvenThoughRecordWouldMatch
// is AC-4b's second clause: the ONLY candidate would fully qualify, but
// ctx becomes done partway through reading its lines -- well before the
// matching content near the end -- so found is never reported.
func TestObserveCodexEffectiveModel_CtxDoneMidFileRead_NotFoundEvenThoughRecordWouldMatch(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{sessionMetaLine(t, rfc3339Milli(start))}
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf(`{"type":"noise","payload":{"i":%d}}`, i))
	}
	lines = append(lines,
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-should-not-be-reported"),
	)
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-midfile.jsonl", lines)

	collectCtx := newCountingCtx(1 << 30)
	candidates, err := codexRolloutCandidates(collectCtx, sessionsDir, spawnStarted, spawnStarted)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("test setup invariant broken: candidates=%+v err=%v", candidates, err)
	}
	collectionCalls := collectCtx.calls

	// +1 for the before-open check, +5 lines in -- well short of the ~52
	// lines needed to reach the matching content near the end.
	ctx := newCountingCtx(collectionCalls + 1 + 5)
	obs, err := ObserveCodexEffectiveModel(ctx, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected ctx's own error, got %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (cut short mid-read, even though this record would otherwise match)", obs)
	}
}

// TestObserveCodexEffectiveModel_CtxExhaustedExactlyAsPassCompletes_FoundStillReturned
// is AC-4b's third clause: ctx's budget is calibrated to exactly the
// number of calls a full, successful pass makes -- every call the pass
// itself performs still succeeds, so it reaches its own natural end and
// returns the real (found) result, even though the very next call to
// ctx.Err() (one the pass never makes) would already report done.
func TestObserveCodexEffectiveModel_CtxExhaustedExactlyAsPassCompletes_FoundStillReturned(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-exact.jsonl", lines)

	calib := newCountingCtx(1 << 30)
	obs, err := ObserveCodexEffectiveModel(calib, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil || obs.Status != CodexObservationFound {
		t.Fatalf("calibration run: got %+v, err=%v, want found", obs, err)
	}

	ctx := newCountingCtx(calib.calls)
	obs2, err2 := ObserveCodexEffectiveModel(ctx, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if obs2.Status != CodexObservationFound || obs2.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (a pass that reaches its own natural end must return its real result)", obs2)
	}
	if err := ctx.Err(); err == nil {
		t.Fatalf("test setup invariant broken: expected ctx's budget to already be exactly exhausted after the pass")
	}
}

// TestObserveCodexEffectiveModel_NoCtxCheckAfterCandidateLoopCompletes is a
// /test cycle-1 addition (issue #173 handoff item 9f): a /test-cycle
// mutation run found that
// TestObserveCodexEffectiveModel_CtxExhaustedExactlyAsPassCompletes_FoundStillReturned
// above does NOT fail when a ctx.Err() check is added right after the
// candidate loop (before the matchCount switch) -- because that test
// self-calibrates its budget by running ObserveCodexEffectiveModel itself
// once (calib.calls), so any extra real call the mutated function makes is
// silently absorbed into the calibration rather than exceeding it. This
// test instead derives its budget by calling codexRolloutCandidates and
// scanRolloutRecord DIRECTLY and summing their own call counts, never
// running ObserveCodexEffectiveModel to calibrate -- so a ctx.Err() call
// anywhere in ObserveCodexEffectiveModel's own body that neither of those
// two helpers made is not part of the budget and flips the result to
// not-found, which this test catches as a failure. Confirmed to fail (red)
// against the "add a ctx check after the candidate loop" mutation, and to
// pass (green) at HEAD.
func TestObserveCodexEffectiveModel_NoCtxCheckAfterCandidateLoopCompletes(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-exact.jsonl", lines)

	// Calibrate collection and the one candidate's scan independently of
	// ObserveCodexEffectiveModel itself.
	collectCtx := newCountingCtx(1 << 30)
	candidates, err := codexRolloutCandidates(collectCtx, sessionsDir, spawnStarted, spawnStarted)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("test setup invariant broken: candidates=%+v err=%v", candidates, err)
	}
	collectionCalls := collectCtx.calls

	scanCtx := newCountingCtx(1 << 30)
	matched, model, err := scanRolloutRecord(scanCtx, candidates[0].path, promptPath, spawnStarted.Truncate(time.Second))
	if err != nil || !matched || model != "gpt-5.6-sol" {
		t.Fatalf("test setup invariant broken: matched=%v model=%q err=%v", matched, model, err)
	}
	scanCalls := scanCtx.calls

	ctx := newCountingCtx(collectionCalls + scanCalls)
	obs, err := ObserveCodexEffectiveModel(ctx, sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v (ObserveCodexEffectiveModel must make no ctx.Err() call beyond exactly what candidate collection plus the one candidate's scan need)", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol", obs)
	}
}

// --- tests -------------------------------------------------------------

func TestObserveCodexEffectiveModel_Found(t *testing.T) {
	cases := []struct {
		name           string
		turnBeforeUser bool
	}{
		{"turn_context before prompt (real record order)", true},
		{"turn_context after prompt", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			sessionsDir := filepath.Join(dir, "sessions")
			promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
			spawnStarted := time.Date(2026, 9, 18, 7, 13, 57, 0, time.UTC)
			base := spawnStarted.Add(500 * time.Millisecond)

			meta := sessionMetaLine(t, rfc3339Milli(base))
			turn := turnContextLine(t, rfc3339Milli(base.Add(2*time.Second)), "gpt-5.6-sol")
			user := userMessageLine(t, rfc3339Milli(base.Add(2300*time.Millisecond)), PromptFilePointer(promptPath))

			var lines []string
			if tc.turnBeforeUser {
				lines = []string{meta, turn, user}
			} else {
				lines = []string{meta, user, turn}
			}
			writeRolloutFile(t, fixtureDateDir(sessionsDir, base), "rollout-found.jsonl", lines)

			obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
				t.Fatalf("got %+v, want found/gpt-5.6-sol", obs)
			}
		})
	}
}

func TestObserveCodexEffectiveModel_PromptPathAbsent(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "environment context, unrelated to any prompt file"),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-noprompt.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound || obs.Model != "" {
		t.Fatalf("got %+v, want not-found/empty model", obs)
	}
}

func TestObserveCodexEffectiveModel_PromptPathOnlyInAssistantMessageDoesNotCount(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		assistantMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "I have read "+promptPath+" and will follow it"),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-assistantonly.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (an assistant message mentioning the path must not count)", obs)
	}
}

// TestObserveCodexEffectiveModel_PathOnlyQuoteInUserMessageDoesNotMatch: a
// user message that merely quotes the bare prompt path -- without the
// full pointer sentence ralph itself writes, PromptFilePointer(promptPath)
// -- must not count as evidence this session is the seat's own. This is
// the scenario the plan itself named: a TASK text relayed to a different
// seat that happens to mention this seat's prompt path.
func TestObserveCodexEffectiveModel_PathOnlyQuoteInUserMessageDoesNotMatch(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "TASK_ID: t-1\n\nsee "+promptPath+" for context"),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-pathonly.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (a bare path quote must not count as the role-prompt pointer)", obs)
	}
}

// TestObserveCodexEffectiveModel_PointerSentenceMatches is the positive
// counterpart: the exact literal ralph writes as the seat's initial prompt
// argument, PromptFilePointer(promptPath), does match.
func TestObserveCodexEffectiveModel_PointerSentenceMatches(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-pointerexact.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (the exact pointer sentence must match)", obs)
	}
}

// TestObserveCodexEffectiveModel_PointerSentenceInsideLongerTextMatches
// proves the match stays Contains, not equality: codex may wrap the text
// (e.g. inside its own turn-formatting), so the pointer sentence appearing
// as a substring of a longer message must still match.
func TestObserveCodexEffectiveModel_PointerSentenceInsideLongerTextMatches(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	wrapped := "<environment_context>\n" + PromptFilePointer(promptPath) + "\n</environment_context>"
	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), wrapped),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-pointerwrapped.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (the pointer sentence wrapped inside more text must still match)", obs)
	}
}

// TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked is AC-2b:
// a session that actually started before spawnStarted must not be picked
// just because its file was touched again (e.g. a still-running old
// process) after spawnStarted -- the age decision comes from the record's
// own session_meta timestamp, not the file's ModTime.
func TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)

	oldStart := spawnStarted.Add(-10 * time.Second)
	oldLines := []string{
		sessionMetaLine(t, rfc3339Milli(oldStart)),
		userMessageLine(t, rfc3339Milli(oldStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(oldStart.Add(300*time.Millisecond)), "gpt-5.5-old"),
	}
	oldPath := writeRolloutFile(t, fixtureDateDir(sessionsDir, oldStart), "rollout-old.jsonl", oldLines)
	touchedAt := spawnStarted.Add(5 * time.Second)
	if err := os.Chtimes(oldPath, touchedAt, touchedAt); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	newStart := spawnStarted.Add(1 * time.Second)
	newLines := []string{
		sessionMetaLine(t, rfc3339Milli(newStart)),
		userMessageLine(t, rfc3339Milli(newStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(newStart.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, newStart), "rollout-new.jsonl", newLines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (the old session must not be picked)", obs)
	}
}

func TestObserveCodexEffectiveModel_TwoQualifyingRecordsAmbiguous(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)

	for i, model := range []string{"gpt-5.6-sol", "gpt-5.7-preview"} {
		start := spawnStarted.Add(time.Duration(i+1) * time.Second)
		lines := []string{
			sessionMetaLine(t, rfc3339Milli(start)),
			userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
			turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), model),
		}
		writeRolloutFile(t, fixtureDateDir(sessionsDir, start), fmt.Sprintf("rollout-dup-%d.jsonl", i), lines)
	}

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationAmbiguous || obs.Model != "" {
		t.Fatalf("got %+v, want ambiguous/empty model", obs)
	}
}

// TestObserveCodexEffectiveModel_SameSecondAsSpawnStartedQualifies covers
// the truncated-cutoff edge case: a session that started a fraction of a
// second before spawnStarted, but within the same whole second, must still
// qualify once spawnStarted is truncated to seconds (the precision the
// manifest actually stores).
func TestObserveCodexEffectiveModel_SameSecondAsSpawnStartedQualifies(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")

	spawnStarted := time.Date(2026, 9, 18, 7, 13, 57, 831000000, time.UTC)
	sessionStart := spawnStarted.Add(-400 * time.Millisecond)
	if sessionStart.Truncate(time.Second) != spawnStarted.Truncate(time.Second) {
		t.Fatalf("test setup invariant broken: sessionStart and spawnStarted must share a whole second")
	}
	if !sessionStart.Before(spawnStarted) {
		t.Fatalf("test setup invariant broken: sessionStart must be before spawnStarted")
	}

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(200*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, sessionStart), "rollout-samesecond.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol", obs)
	}
}

func TestObserveCodexEffectiveModel_SymlinkIgnored(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	dateDir := fixtureDateDir(sessionsDir, start)
	realPath := writeRolloutFile(t, dateDir, "rollout-real.jsonl", lines)

	linkPath := filepath.Join(dateDir, "rollout-link.jsonl")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// If the symlink were followed and scanned as its own candidate, this
	// would come back ambiguous (two matches) instead of found.
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (symlink must not count as a second candidate)", obs)
	}
}

// TestObserveCodexEffectiveModel_FIFODoesNotHang: package org already
// builds unix-only without a build tag (lockfile.go uses syscall.Flock
// unconditionally), so this test needs no //go:build constraint either.
func TestObserveCodexEffectiveModel_FIFODoesNotHang(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	dateDir := fixtureDateDir(sessionsDir, spawnStarted.Add(1*time.Second))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fifoPath := filepath.Join(dateDir, "rollout-fifo.jsonl")
	if err := syscall.Mkfifo(fifoPath, 0o644); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_, _ = ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ObserveCodexEffectiveModel hung on a FIFO left in the sessions directory")
	}
}

func TestObserveCodexEffectiveModel_MalformedLineSkipped(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		"not even json",
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		`{"type": "turn_context", "payload": {`, // truncated/invalid JSON
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-malformed.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol despite the garbage lines", obs)
	}
}

func TestObserveCodexEffectiveModel_OversizedLineSkipped(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	oversizedTurn := marshalFixtureLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), codexLineTypeTurnContext, map[string]string{
		"model":   "gpt-should-not-be-seen",
		"effort":  "medium",
		"padding": strings.Repeat("x", codexObserveMaxLineBytes),
	}, 1)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		oversizedTurn,
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-oversized.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (the only turn_context line exceeds the per-line cap)", obs)
	}
}

func TestObserveCodexEffectiveModel_SessionMetaWithoutTurnContext(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-noturn.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (no turn_context yet)", obs)
	}
}

func TestObserveCodexEffectiveModel_FileByteCapStopsBeforeMatch(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	filler := `{"type":"noise","payload":{"blob":"` + strings.Repeat("x", codexObserveMaxFileBytes+100*1024) + `"}}`
	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		filler,
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-should-not-be-seen"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-bytecap.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (turn_context only appears after the per-file byte cap)", obs)
	}
}

// TestObserveCodexEffectiveModel_YesterdaysDateDirectoryIsWalked places a
// fixture in *yesterday's* date directory (relative to spawnStarted) with a
// session_meta timestamp that still passes the age gate -- this is the
// local/UTC day-boundary mismatch codexSessionDateDirs' slack exists to
// absorb (a directory named by one clock, a session_meta timestamp
// recorded by another, landing a day apart near midnight).
func TestObserveCodexEffectiveModel_YesterdaysDateDirectoryIsWalked(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Now()
	sessionStart := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-yesterday-dir"),
	}
	yesterdayDir := filepath.Join(sessionsDir, spawnStarted.Local().AddDate(0, 0, -1).Format(codexSessionDateLayout))
	writeRolloutFile(t, yesterdayDir, "rollout-yesterday-dir.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-yesterday-dir" {
		t.Fatalf("got %+v, want found/gpt-yesterday-dir (yesterday's date directory must be walked)", obs)
	}
}

// TestObserveCodexEffectiveModel_ThreeDaysOldDateDirectoryNotWalked is the
// mirror of the yesterday test: a directory three days before spawnStarted
// is outside the walked window even though the record inside it would
// otherwise fully qualify.
func TestObserveCodexEffectiveModel_ThreeDaysOldDateDirectoryNotWalked(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Now()
	sessionStart := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-3days-dir"),
	}
	threeDaysDir := filepath.Join(sessionsDir, spawnStarted.Local().AddDate(0, 0, -3).Format(codexSessionDateLayout))
	path := writeRolloutFile(t, threeDaysDir, "rollout-3days-dir.jsonl", lines)
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (a date directory three days old must not be walked)", obs)
	}
}

// TestObserveCodexEffectiveModel_DirectoryThreeDaysAfterSpawnDateNotWalked
// pins that codexSessionDateDirs has no unbounded "through today" upper
// bound, so a directory three days
// AFTER spawnStarted's own date must not be walked either -- this is the
// direction the old "today+1" bound could never have caught (a directory
// well after spawnStarted's date used to be reachable whenever the real
// wall clock, at observation time, was far enough past spawnStarted -- the
// exact shape Stop's own "well after spawnStarted" call pattern could hit
// in production). spawnStarted is a fixed past date, not time.Now(), so
// this test cannot depend on the wall clock at all -- codexSessionDateDirs
// takes none anymore.
func TestObserveCodexEffectiveModel_DirectoryThreeDaysAfterSpawnDateNotWalked(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	sessionStart := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-3days-after-dir"),
	}
	threeDaysAfterDir := filepath.Join(sessionsDir, spawnStarted.Local().AddDate(0, 0, 3).Format(codexSessionDateLayout))
	writeRolloutFile(t, threeDaysAfterDir, "rollout-3days-after-dir.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (a date directory three days after the spawn date must not be walked)", obs)
	}
}

// TestObserveCodexEffectiveModel_SpawnDateDirectory_ModTimeManyDaysLaterStillFound
// is the other direction from the test above: the record's directory
// matches the day the session STARTED (spawnStarted's own date), but the
// file's ModTime is many days
// later -- codex kept writing to it long after the session began. Real
// file-metadata evidence (docs/evidence/codex-effective-model-receipt-2026-09-20.md's
// follow-up check): of 1,013 real records, 57 were last modified on a
// later calendar day than their directory date, up to 17 days later, and
// none had moved directories. This must still be found: the fixed
// 3-directory window is keyed to spawnStarted's date (not the file's
// ModTime), and the ModTime-slack pre-filter only ever excludes a file
// modified too early, never one modified late.
func TestObserveCodexEffectiveModel_SpawnDateDirectory_ModTimeManyDaysLaterStillFound(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	sessionStart := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-long-lived-session"),
	}
	spawnDateDir := filepath.Join(sessionsDir, spawnStarted.Local().Format(codexSessionDateLayout))
	path := writeRolloutFile(t, spawnDateDir, "rollout-long-lived.jsonl", lines)
	touchedAt := spawnStarted.AddDate(0, 0, 17)
	if err := os.Chtimes(path, touchedAt, touchedAt); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-long-lived-session" {
		t.Fatalf("got %+v, want found/gpt-long-lived-session (a session touched 17 days later must still be found in its own start-date directory)", obs)
	}
}

// TestObserveCodexEffectiveModel_UntilReachesSessionStartedDaysAfterSpawn
// covers cycle-2 self-review C2-1
// (docs/reports/self-review-2026-09-20-codex-effective-model-receipt.md):
// a codex model-retirement dialog left open for three days before being
// answered means the session record itself does not start until day 3 --
// until must be able to reach
// that day's directory for Stop's second-chance observation to find it.
func TestObserveCodexEffectiveModel_UntilReachesSessionStartedDaysAfterSpawn(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	sessionStart := spawnStarted.AddDate(0, 0, 3)
	until := spawnStarted.AddDate(0, 0, 4)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, sessionStart), "rollout-late-start.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, until)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (until must reach a session that started days after the spawn)", obs)
	}
}

// TestObserveCodexEffectiveModel_UntilEqualsSpawnStarted_LateSessionNotFound
// is the same fixture as above with until pinned back to spawnStarted --
// documenting Spawn's own narrow window (it always passes spawnStarted for
// until, since its poll runs moments after the spawn): a session that only
// starts three days later is out of reach for Spawn's own poll, exactly as
// before this fix.
func TestObserveCodexEffectiveModel_UntilEqualsSpawnStarted_LateSessionNotFound(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	sessionStart := spawnStarted.AddDate(0, 0, 3)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, sessionStart), "rollout-late-start.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (until == spawnStarted must not reach a session that starts 3 days later)", obs)
	}
}

// TestObserveCodexEffectiveModel_UntilBeforeSpawnStarted_TreatedAsSpawnStarted
// pins that an until earlier than spawnStarted never shrinks the window
// below the original three directories -- a record in spawnStarted's own
// date is still found.
func TestObserveCodexEffectiveModel_UntilBeforeSpawnStarted_TreatedAsSpawnStarted(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	sessionStart := spawnStarted.Add(1 * time.Second)

	lines := []string{
		sessionMetaLine(t, rfc3339Milli(sessionStart)),
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, sessionStart), "rollout-same-day.jsonl", lines)

	until := spawnStarted.Add(-10 * time.Hour)
	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, until)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (until before spawnStarted must behave like until == spawnStarted, not shrink the window)", obs)
	}
}

// TestCodexSessionDateDirs_UntilBeforeSpawnStarted is the pure-function
// counterpart of the test above: an until earlier than spawnStarted must
// produce the exact same directory list as until == spawnStarted.
func TestCodexSessionDateDirs_UntilBeforeSpawnStarted(t *testing.T) {
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	until := spawnStarted.Add(-10 * time.Hour)
	got := codexSessionDateDirs(spawnStarted, until)
	want := codexSessionDateDirs(spawnStarted, spawnStarted)
	if !slices.Equal(got, want) {
		t.Fatalf("codexSessionDateDirs(spawnStarted, until-before-spawnStarted) = %v, want %v (same as until == spawnStarted)", got, want)
	}
}

// TestCodexSessionDateDirs_CapKeepsEarliestDirectories is the self-review
// cycle-2 fix's own cap test: a 60-day span between spawnStarted and until
// yields exactly codexObserveMaxDateDirs directories, and they are the
// EARLIEST consecutive days in that span (starting at spawnStarted's local
// date minus one day), not an arbitrary or a latest-days subset.
func TestCodexSessionDateDirs_CapKeepsEarliestDirectories(t *testing.T) {
	spawnStarted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	until := spawnStarted.AddDate(0, 0, 60)

	got := codexSessionDateDirs(spawnStarted, until)
	if len(got) != codexObserveMaxDateDirs {
		t.Fatalf("len(got) = %d, want %d", len(got), codexObserveMaxDateDirs)
	}
	startDay := truncateToLocalDay(spawnStarted).AddDate(0, 0, -1)
	for i, dir := range got {
		want := startDay.AddDate(0, 0, i).Format(codexSessionDateLayout)
		if dir != want {
			t.Fatalf("got[%d] = %q, want %q (consecutive earliest days starting at spawnStarted's date minus one)", i, dir, want)
		}
	}
}

// TestCodexSessionDateDirs_UntilZeroValue_TreatedAsSpawnStarted is a /test
// cycle-2 addition pinning ObserveCodexEffectiveModel's doc comment
// literally ("until earlier than spawnStarted, INCLUDING THE ZERO VALUE, is
// treated as spawnStarted") against the actual zero time.Time{} value, not
// just an arbitrary earlier instant (TestCodexSessionDateDirs_UntilBeforeSpawnStarted
// above already covers "earlier", but never the literal zero value the doc
// comment calls out by name).
func TestCodexSessionDateDirs_UntilZeroValue_TreatedAsSpawnStarted(t *testing.T) {
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	got := codexSessionDateDirs(spawnStarted, time.Time{})
	want := codexSessionDateDirs(spawnStarted, spawnStarted)
	if !slices.Equal(got, want) {
		t.Fatalf("codexSessionDateDirs(spawnStarted, zero-value until) = %v, want %v (same as until == spawnStarted)", got, want)
	}
}

// TestCodexSessionDateDirs_UntilYearsLater_CapHoldsAndReturnsQuickly is a
// /test cycle-2 addition: TestCodexSessionDateDirs_CapKeepsEarliestDirectories
// already proves the cap's CONTENT is correct for a 60-day span, but never
// exercises a span so wide it would otherwise mean tens of thousands of
// loop iterations (a multi-year until, e.g. a stop called long after an
// abandoned seat) -- proving both that the cap still holds at that scale
// and that codexSessionDateDirs returns near-instantly rather than
// building and discarding a huge slice first.
func TestCodexSessionDateDirs_UntilYearsLater_CapHoldsAndReturnsQuickly(t *testing.T) {
	spawnStarted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	until := spawnStarted.AddDate(3, 0, 0) // 3 years later

	start := time.Now()
	got := codexSessionDateDirs(spawnStarted, until)
	elapsed := time.Since(start)

	if len(got) != codexObserveMaxDateDirs {
		t.Fatalf("len(got) = %d, want %d (the cap must hold for a multi-year span too)", len(got), codexObserveMaxDateDirs)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("codexSessionDateDirs took %s for a 3-year until span -- the cap should keep this near-instant, not proportional to the span", elapsed)
	}
}

// TestObserveCodexEffectiveModel_WidenedWindowStillExcludesStaleRecord is a
// /test cycle-2 addition probing the interaction the hand-off asked about
// between C2-1 (the until widening) and AC-2b (the session_meta age
// exclusion). Structurally, codexSessionDateDirs's startDay is fixed at
// spawnStarted's own local date minus one day, independent of until (see
// its own doc comment) -- so widening until can only add directories on
// the FUTURE side of spawnStarted, never reach further into the past. A
// record whose session_meta genuinely predates spawnStarted therefore
// cannot become newly reachable through this widening under codex's own
// "a record lives in the directory of the day it started" convention. This
// test still proves the underlying safety property directly rather than by
// argument alone: even a record deliberately misfiled into a
// later-dated directory (a shape real codex never produces, but the only
// way to make "reachable only via the wider window" and "stale" true at
// the same time in a fixture) is excluded by its own session_meta
// timestamp, not read as qualifying just because its directory is now in
// reach -- matching codexRolloutCandidates' own doc comment ("This is a
// pre-filter only -- it never decides which record belongs to which
// seat").
func TestObserveCodexEffectiveModel_WidenedWindowStillExcludesStaleRecord(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)

	// staleStart predates spawnStarted (AC-2b's own disqualifying
	// condition), but is deliberately filed under a directory 5 days AFTER
	// spawnStarted's own day -- only reachable because until (below)
	// widens the window that far forward.
	staleStart := spawnStarted.Add(-1 * time.Hour)
	staleDir := fixtureDateDir(sessionsDir, spawnStarted.AddDate(0, 0, 5))
	staleLines := []string{
		sessionMetaLine(t, rfc3339Milli(staleStart)),
		userMessageLine(t, rfc3339Milli(staleStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(staleStart.Add(300*time.Millisecond)), "gpt-5.5-stale"),
	}
	staleFile := writeRolloutFile(t, staleDir, "rollout-misplaced-stale.jsonl", staleLines)
	// The ModTime pre-filter is anchored to spawnStarted only (never to
	// until or to the file's own directory) -- touch it forward so it
	// survives that pre-filter and actually reaches scanRolloutRecord,
	// where the session_meta check is what this test is proving.
	touchedAt := spawnStarted.Add(1 * time.Minute)
	if err := os.Chtimes(staleFile, touchedAt, touchedAt); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// validStart is the seat's own, real, in-window session, correctly
	// filed under spawnStarted's own day.
	validStart := spawnStarted.Add(1 * time.Second)
	validLines := []string{
		sessionMetaLine(t, rfc3339Milli(validStart)),
		userMessageLine(t, rfc3339Milli(validStart.Add(400*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(validStart.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, validStart), "rollout-valid.jsonl", validLines)

	// until reaches well past the misplaced stale record's own directory,
	// simulating a Stop call running days after a slow-starting seat --
	// exactly the scenario C2-1's widening exists for.
	until := spawnStarted.AddDate(0, 0, 10)
	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, until)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationFound || obs.Model != "gpt-5.6-sol" {
		t.Fatalf("got %+v, want found/gpt-5.6-sol (the widened window reaching the misplaced stale record's directory must not change AC-2b's session_meta-based exclusion)", obs)
	}
}

func TestObserveCodexEffectiveModel_SessionsDirMissing(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "does-not-exist")
	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, filepath.Join(dir, "prompts", "org1_seat1.md"), time.Now(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found", obs)
	}
}

func TestObserveCodexEffectiveModel_EmptyInputsReturnNotFoundImmediately(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name        string
		sessionsDir string
		promptPath  string
	}{
		{"empty sessionsDir", "", filepath.Join(dir, "prompts", "org1_seat1.md")},
		{"empty promptPath", filepath.Join(dir, "sessions"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs, err := ObserveCodexEffectiveModel(context.Background(), tc.sessionsDir, tc.promptPath, time.Now(), time.Now())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if obs.Status != CodexObservationNotFound || obs.Model != "" {
				t.Fatalf("got %+v, want not-found/empty model", obs)
			}
		})
	}
}

// TestObserveCodexEffectiveModel_PermissionDeniedDoesNotLeakContent is the
// one realistic, triggerable error path in this observer: a candidate file
// that Lstat already confirmed is a regular, name-matching, recently
// modified file, but that os.Open then fails to read. The returned error
// must name only the operation and the path -- never any line content,
// even though this exact file's body does contain some.
func TestObserveCodexEffectiveModel_PermissionDeniedDoesNotLeakContent(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: chmod-based permission denial is not enforced")
	}
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptPath := filepath.Join(dir, "state", "prompts", "org1_seat1.md")
	spawnStarted := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	start := spawnStarted.Add(1 * time.Second)

	const sentinel = "sentinel-fixture-body-must-not-appear-in-any-error"
	lines := []string{
		sessionMetaLine(t, rfc3339Milli(start)),
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), PromptFilePointer(promptPath)+" "+sentinel),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	path := writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-noperm.jsonl", lines)
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	obs, err := ObserveCodexEffectiveModel(context.Background(), sessionsDir, promptPath, spawnStarted, spawnStarted)
	if err == nil {
		t.Fatalf("expected a non-nil error for a permission-denied candidate file, got obs=%+v", obs)
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("error text leaked fixture body content: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got status %v alongside the error, want not-found", obs.Status)
	}
}

func TestCodexSessionsDir(t *testing.T) {
	cases := []struct {
		name      string
		codexHome string
		home      string
		want      string
	}{
		{"codexHome set", "/custom/codex-home", "/home/user", filepath.Join("/custom/codex-home", "sessions")},
		{"codexHome empty", "", "/home/user", filepath.Join("/home/user", ".codex", "sessions")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CodexSessionsDir(tc.codexHome, tc.home)
			if got != tc.want {
				t.Fatalf("CodexSessionsDir(%q, %q) = %q, want %q", tc.codexHome, tc.home, got, tc.want)
			}
		})
	}
}
