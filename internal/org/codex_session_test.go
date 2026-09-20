package org

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
			user := userMessageLine(t, rfc3339Milli(base.Add(2300*time.Millisecond)), "role prompt: "+promptPath)

			var lines []string
			if tc.turnBeforeUser {
				lines = []string{meta, turn, user}
			} else {
				lines = []string{meta, user, turn}
			}
			writeRolloutFile(t, fixtureDateDir(sessionsDir, base), "rollout-found.jsonl", lines)

			obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (an assistant message mentioning the path must not count)", obs)
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
		userMessageLine(t, rfc3339Milli(oldStart.Add(400*time.Millisecond)), "role prompt: "+promptPath),
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
		userMessageLine(t, rfc3339Milli(newStart.Add(400*time.Millisecond)), "role prompt: "+promptPath),
		turnContextLine(t, rfc3339Milli(newStart.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, newStart), "rollout-new.jsonl", newLines)

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
			userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "role prompt: "+promptPath),
			turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), model),
		}
		writeRolloutFile(t, fixtureDateDir(sessionsDir, start), fmt.Sprintf("rollout-dup-%d.jsonl", i), lines)
	}

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "role prompt: "+promptPath),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(200*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, sessionStart), "rollout-samesecond.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "role prompt: "+promptPath),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	dateDir := fixtureDateDir(sessionsDir, start)
	realPath := writeRolloutFile(t, dateDir, "rollout-real.jsonl", lines)

	linkPath := filepath.Join(dateDir, "rollout-link.jsonl")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		_, _ = ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "role prompt: "+promptPath),
		`{"type": "turn_context", "payload": {`, // truncated/invalid JSON
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-malformed.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "role prompt: "+promptPath),
		oversizedTurn,
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-oversized.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "role prompt: "+promptPath),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-noturn.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "role prompt: "+promptPath),
		filler,
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-should-not-be-seen"),
	}
	writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-bytecap.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), "role prompt: "+promptPath),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-yesterday-dir"),
	}
	yesterdayDir := filepath.Join(sessionsDir, spawnStarted.Local().AddDate(0, 0, -1).Format(codexSessionDateLayout))
	writeRolloutFile(t, yesterdayDir, "rollout-yesterday-dir.jsonl", lines)

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
		userMessageLine(t, rfc3339Milli(sessionStart.Add(400*time.Millisecond)), "role prompt: "+promptPath),
		turnContextLine(t, rfc3339Milli(sessionStart.Add(300*time.Millisecond)), "gpt-3days-dir"),
	}
	threeDaysDir := filepath.Join(sessionsDir, spawnStarted.Local().AddDate(0, 0, -3).Format(codexSessionDateLayout))
	path := writeRolloutFile(t, threeDaysDir, "rollout-3days-dir.jsonl", lines)
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.Status != CodexObservationNotFound {
		t.Fatalf("got %+v, want not-found (a date directory three days old must not be walked)", obs)
	}
}

func TestObserveCodexEffectiveModel_SessionsDirMissing(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "does-not-exist")
	obs, err := ObserveCodexEffectiveModel(sessionsDir, filepath.Join(dir, "prompts", "org1_seat1.md"), time.Now())
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
			obs, err := ObserveCodexEffectiveModel(tc.sessionsDir, tc.promptPath, time.Now())
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
		userMessageLine(t, rfc3339Milli(start.Add(400*time.Millisecond)), "role prompt: "+promptPath+" "+sentinel),
		turnContextLine(t, rfc3339Milli(start.Add(300*time.Millisecond)), "gpt-5.6-sol"),
	}
	path := writeRolloutFile(t, fixtureDateDir(sessionsDir, start), "rollout-noperm.jsonl", lines)
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted)
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
