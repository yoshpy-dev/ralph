package org

import (
	"encoding/json"
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

			obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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
		_, _ = ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, until)
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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
	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, until)
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

func TestObserveCodexEffectiveModel_SessionsDirMissing(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "does-not-exist")
	obs, err := ObserveCodexEffectiveModel(sessionsDir, filepath.Join(dir, "prompts", "org1_seat1.md"), time.Now(), time.Now())
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
			obs, err := ObserveCodexEffectiveModel(tc.sessionsDir, tc.promptPath, time.Now(), time.Now())
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

	obs, err := ObserveCodexEffectiveModel(sessionsDir, promptPath, spawnStarted, spawnStarted)
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
