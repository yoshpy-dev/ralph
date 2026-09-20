package org

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// This file is the observer only: it reads codex session records and
// reports a model, nothing else -- it never reads any other part of the
// process environment, and it never writes anything. Two callers wire it
// in: Spawn's own poll after a codex seat reaches `spawned`
// (spawn.go's observeCodexSpawnReceipt) and Stop's single, non-waiting
// check (verbs.go's observeStopModelReceipt). See issue #165 and
// docs/evidence/codex-effective-model-receipt-2026-09-20.md for the
// real-record measurements this file's constants are built from.

// codexObserveMaxFiles caps how many rollout-*.jsonl files ObserveCodexEffectiveModel
// actually opens and scans per call, newest (by ModTime) first. Real codex
// session directories are small in practice; this is a hard backstop against
// a sessions directory that has grown unexpectedly large.
const codexObserveMaxFiles = 200

// codexObserveMaxFileBytes caps how many bytes are read from a single
// candidate file (via io.LimitReader). The plan's real-record measurements
// (2026-09-18, codex-cli 0.154.0) put the qualifying content -- turn_context
// and the prompt-path user message -- within the first ~10 lines, so 4 MiB
// is generous headroom, not a tight fit. A qualifying match that would only
// appear after this many bytes is treated the same as no match: see
// scanRolloutRecord.
const codexObserveMaxFileBytes = 4 * 1024 * 1024

// codexObserveMaxLineBytes bounds the length of a single JSONL line this
// scanner will decode. The largest real line observed in the same sample
// was about 44 KiB; this constant leaves generous headroom above that while
// still bounding per-line memory use. A line longer than this is skipped
// (never decoded), not fatal -- see scanRolloutRecord.
const codexObserveMaxLineBytes = 256 * 1024

// codexObserveModTimeSlack is the cheap pre-filter applied to a candidate
// file's ModTime before it is even opened: a file last modified more than
// this long before spawnStarted cannot contain a session that started at or
// after spawnStarted, so it is skipped without reading. This is a pre-filter
// only -- it never decides which record belongs to which seat; that
// decision is made from the session_meta timestamp inside the file (see
// scanRolloutRecord and AC-2b: a stale session that gets touched again
// after spawnStarted must still be excluded by its own recorded start time,
// not picked because its ModTime looks recent).
const codexObserveModTimeSlack = 2 * time.Second

// codexRolloutFilePrefix and codexRolloutFileSuffix identify codex's own
// per-session record filename shape: rollout-<local timestamp>-<uuid>.jsonl.
const (
	codexRolloutFilePrefix = "rollout-"
	codexRolloutFileSuffix = ".jsonl"
)

// codexSessionDateLayout mirrors codex's own YYYY/MM/DD date-directory
// layout under sessions/.
const codexSessionDateLayout = "2006/01/02"

// CodexSessionsDir resolves where codex keeps its session records:
// $CODEX_HOME/sessions when codexHome is non-empty, else <home>/.codex/sessions.
// Pure: it takes both values as parameters and reads no environment itself.
// Callers resolve os.Getenv("CODEX_HOME") / os.UserHomeDir() the same way
// codexModelsCachePath (internal/cli/doctor_codex_models.go) already does,
// and pass the results in here -- keeping this function a deterministic,
// easily test-seamed boundary rather than one more place in the org package
// that reads the process environment directly.
func CodexSessionsDir(codexHome, home string) string {
	if codexHome != "" {
		return filepath.Join(codexHome, "sessions")
	}
	return filepath.Join(home, ".codex", "sessions")
}

// CodexObservationStatus is the tri-state outcome of ObserveCodexEffectiveModel.
type CodexObservationStatus string

const (
	// CodexObservationFound means exactly one qualifying session record was
	// found and its first turn_context reported a model.
	CodexObservationFound CodexObservationStatus = "found"
	// CodexObservationNotFound means no qualifying session record was found
	// (or sessionsDir/promptPath was empty, or the observation window
	// contained nothing readable).
	CodexObservationNotFound CodexObservationStatus = "not-found"
	// CodexObservationAmbiguous means two or more session records qualified
	// -- the caller must never guess which one is authoritative.
	CodexObservationAmbiguous CodexObservationStatus = "ambiguous"
)

// CodexModelObservation is the result of one ObserveCodexEffectiveModel
// call. Model is non-empty only when Status is CodexObservationFound --
// deliberately: an ambiguous or not-found result must never carry a guessed
// model. Nothing on this type (or anywhere in this file) ever carries
// session record content -- only the single model string a qualifying
// turn_context reported.
type CodexModelObservation struct {
	Model  string
	Status CodexObservationStatus
}

// ObserveCodexEffectiveModel looks for the codex session record of one seat
// spawn under sessionsDir, identifying it by promptPath -- the absolute
// path to the seat's role-prompt file (see spawn.go's promptFilePath). A
// record qualifies only when a user message's content contains the full
// pointer sentence ralph itself passed the seat, promptFilePointer(promptPath)
// (see codexResponseItemMentionsPrompt), not merely the bare path.
// spawnStarted is the spawn's own start time; only a session that began at
// or after spawnStarted (truncated to whole seconds, since that is the
// precision the manifest stores -- see AC-2b) can belong to this spawn,
// which is what lets a re-spawn of the same org_id/seat_id (same
// promptPath) tell its own session apart from an older one that happens to
// still be running or gets touched again later.
//
// This function is pure: no environment reads, no globals mutated, no
// goroutines, no writes. It never returns partial or guessed content -- see
// CodexModelObservation's doc comment. The non-nil error return is reserved
// for a specific, content-free signal a caller might want to log (a
// candidate file that exists but could not be opened, e.g. permission
// denied); every other failure mode -- a missing sessionsDir, an unreadable
// date directory, a missing/unreadable candidate file, a malformed or
// oversized line -- degrades silently to CodexObservationNotFound with a
// nil error. Neither caller (Spawn's poll, Stop's single check) ever lets a
// codex CLI record-shape change or a permissions quirk fail the seat's own
// spawn or stop.
func ObserveCodexEffectiveModel(sessionsDir, promptPath string, spawnStarted time.Time) (CodexModelObservation, error) {
	if sessionsDir == "" || promptPath == "" {
		return CodexModelObservation{Status: CodexObservationNotFound}, nil
	}
	if _, err := os.Stat(sessionsDir); err != nil {
		// Missing or unreadable sessionsDir: not an error (codex may simply
		// not be installed, or CODEX_HOME points somewhere ralph can't see).
		return CodexModelObservation{Status: CodexObservationNotFound}, nil
	}

	cutoff := spawnStarted.Truncate(time.Second)
	candidates := codexRolloutCandidates(sessionsDir, spawnStarted)

	var (
		matchedModel string
		matchCount   int
		firstErr     error
	)
	for _, c := range candidates {
		matched, model, err := scanRolloutRecord(c.path, promptPath, cutoff)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if !matched {
			continue
		}
		matchCount++
		if matchCount == 1 {
			matchedModel = model
		}
		if matchCount >= 2 {
			// Already ambiguous; no later candidate can change that verdict.
			break
		}
	}

	switch matchCount {
	case 0:
		return CodexModelObservation{Status: CodexObservationNotFound}, firstErr
	case 1:
		return CodexModelObservation{Model: matchedModel, Status: CodexObservationFound}, firstErr
	default:
		return CodexModelObservation{Status: CodexObservationAmbiguous}, firstErr
	}
}

// codexRolloutCandidate is one file gathered by codexRolloutCandidates,
// carrying just enough to sort and open it.
type codexRolloutCandidate struct {
	path    string
	modTime time.Time
}

// codexRolloutCandidates walks the date-directory window codexSessionDateDirs
// returns for spawnStarted, collects every regular rollout-*.jsonl file
// whose ModTime is not older than spawnStarted minus codexObserveModTimeSlack,
// and returns them newest-first, capped at codexObserveMaxFiles. A symlink,
// FIFO, device, or directory that happens to match the name pattern is
// identified via os.Lstat (which never follows a symlink) and dropped --
// never opened -- so a FIFO left behind in the tree cannot hang this call.
// A missing or unreadable date directory is silently skipped (per
// ObserveCodexEffectiveModel's doc comment, not an error).
func codexRolloutCandidates(sessionsDir string, spawnStarted time.Time) []codexRolloutCandidate {
	minModTime := spawnStarted.Add(-codexObserveModTimeSlack)

	var candidates []codexRolloutCandidate
	for _, dateDir := range codexSessionDateDirs(spawnStarted) {
		dirPath := filepath.Join(sessionsDir, dateDir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if !isCodexRolloutFileName(name) {
				continue
			}
			fullPath := filepath.Join(dirPath, name)
			info, err := os.Lstat(fullPath)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			if info.ModTime().Before(minModTime) {
				continue
			}
			candidates = append(candidates, codexRolloutCandidate{path: fullPath, modTime: info.ModTime()})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	if len(candidates) > codexObserveMaxFiles {
		candidates = candidates[:codexObserveMaxFiles]
	}
	return candidates
}

// isCodexRolloutFileName reports whether name matches codex's rollout
// record filename shape (rollout-*.jsonl).
func isCodexRolloutFileName(name string) bool {
	return strings.HasPrefix(name, codexRolloutFilePrefix) && strings.HasSuffix(name, codexRolloutFileSuffix)
}

// codexSessionDateDirs returns exactly three YYYY/MM/DD date directories
// (codex's own layout) to walk: spawnStarted's local date, minus one day
// and plus one day. A session's record lives in the directory of the day
// the session STARTED, no matter how much longer codex keeps writing to it
// afterward -- confirmed against this machine's real sessions directory
// (file metadata only): of 1,013 records, 57 were last modified on a later
// calendar day than their directory date (up to 17 days later), none had
// moved to a different directory, and every file name's embedded date
// matched its directory date exactly. So this walk never needs "today" (or
// any other wall-clock read) -- it only needs the single day spawnStarted
// itself falls on. The one-day pad on each side absorbs a UTC/local
// day-boundary mismatch between whatever clock named the directory (codex
// documents it as a "local timestamp") and spawnStarted's own Location --
// narrowing this to exactly spawnStarted's date would risk silently
// missing a real file created just across a midnight boundary. This was
// also the only place in this package that read the live wall clock;
// removing that read here means ObserveCodexEffectiveModel never touches
// it at all, only the caller-supplied spawnStarted.
func codexSessionDateDirs(spawnStarted time.Time) []string {
	day := truncateToLocalDay(spawnStarted)
	return []string{
		day.AddDate(0, 0, -1).Format(codexSessionDateLayout),
		day.Format(codexSessionDateLayout),
		day.AddDate(0, 0, 1).Format(codexSessionDateLayout),
	}
}

// truncateToLocalDay returns t converted to its local time zone with the
// time-of-day component zeroed, for date-directory arithmetic.
func truncateToLocalDay(t time.Time) time.Time {
	lt := t.Local()
	return time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, lt.Location())
}

// codexSessionLine is the common envelope every line of a rollout file
// shares: a type discriminator and an opaque payload decoded only for the
// three types this scanner cares about (see codexSessionMetaQualifies,
// codexTurnContextModel, codexResponseItemMentionsPrompt). Every other
// field in a real record -- and every other line type -- is never decoded,
// by design: this scanner has no reason to ever hold a whole payload's
// conversation content in memory.
type codexSessionLine struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

const (
	codexLineTypeSessionMeta  = "session_meta"
	codexLineTypeTurnContext  = "turn_context"
	codexLineTypeResponseItem = "response_item"
)

// scanRolloutRecord decides whether the rollout file at path qualifies as
// the session for promptPath at or after cutoff, and if so, what model its
// first turn_context line reported. It reads at most
// codexObserveMaxFileBytes from path and never decodes a line longer than
// codexObserveMaxLineBytes (see those constants' doc comments); either
// limit being hit simply means whatever hasn't been seen yet counts as not
// seen -- scanning stops and matched is false unless everything needed was
// already found.
//
// err is non-nil only when path exists (it was already Lstat'd as a
// regular, name-matching, recently-modified file by the caller) but could
// not be opened -- e.g. permission denied. That error is a plain wrapped
// os.PathError: it names the operation and the path, never any line
// content. Every other failure -- the file having vanished in the race
// between Lstat and Open, a malformed line, an unparsable timestamp --
// degrades to matched=false, err=nil.
func scanRolloutRecord(path, promptPath string, cutoff time.Time) (matched bool, model string, err error) {
	f, openErr := os.Open(path)
	if openErr != nil {
		if errors.Is(openErr, fs.ErrNotExist) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("org: open codex session record %s: %w", path, openErr)
	}
	defer func() { _ = f.Close() }()

	reader := bufio.NewReaderSize(io.LimitReader(f, codexObserveMaxFileBytes), 64*1024)

	var (
		metaSeen     bool
		ageQualifies bool
		promptSeen   bool
		turnModel    string
	)
	decided := func() bool {
		return metaSeen && ageQualifies && promptSeen && turnModel != ""
	}

	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 && len(line) <= codexObserveMaxLineBytes {
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) > 0 {
				var env codexSessionLine
				if json.Unmarshal(trimmed, &env) == nil {
					switch env.Type {
					case codexLineTypeSessionMeta:
						if !metaSeen {
							metaSeen = true
							ageQualifies = codexSessionMetaQualifies(env.Payload, cutoff)
						}
					case codexLineTypeTurnContext:
						if turnModel == "" {
							turnModel = codexTurnContextModel(env.Payload)
						}
					case codexLineTypeResponseItem:
						if !promptSeen {
							promptSeen = codexResponseItemMentionsPrompt(env.Payload, promptPath)
						}
					}
				}
			}
		}
		if metaSeen && !ageQualifies {
			// The record's own start time disqualifies it; nothing later in
			// the file can change that (AC-2b).
			return false, "", nil
		}
		if decided() {
			return true, turnModel, nil
		}
		if readErr != nil {
			// io.EOF (a real end of file, or codexObserveMaxFileBytes being
			// exhausted) or a genuine read error: either way, nothing more
			// to learn from this file.
			break
		}
	}
	return false, "", nil
}

// codexSessionMetaQualifies reports whether a session_meta line's
// payload.timestamp parses and is not before cutoff. An unparsable or
// missing timestamp cannot be shown to satisfy "not before cutoff", so it
// is treated as disqualifying rather than guessed at.
func codexSessionMetaQualifies(payload json.RawMessage, cutoff time.Time) bool {
	var p struct {
		Timestamp string `json:"timestamp"`
	}
	if json.Unmarshal(payload, &p) != nil {
		return false
	}
	t, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		return false
	}
	return !t.Before(cutoff)
}

// codexTurnContextModel returns a turn_context line's payload.model, or ""
// if the payload doesn't decode or the field is empty.
func codexTurnContextModel(payload json.RawMessage) string {
	var p struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(payload, &p) != nil {
		return ""
	}
	return p.Model
}

// codexResponseItemMentionsPrompt reports whether a response_item line's
// payload is a user message whose content contains the full pointer
// sentence ralph passes the seat as its role-prompt argument --
// promptFilePointer(promptPath), spawn.go, e.g. "役割指示を読み込んで従っ
// てください: <path>" -- as a substring (Contains, not equality: codex may
// wrap the text). Matching on the bare path alone would also match any
// other message that merely quotes it, such as a TASK text relayed to a
// different seat; the full sentence is the exact literal ralph itself
// writes, so only a record whose seat was actually handed this pointer can
// match. Only role=="user" messages count -- an assistant message that
// happens to echo the same text (e.g. quoting it back) is not evidence the
// seat's own initial prompt was this one.
func codexResponseItemMentionsPrompt(payload json.RawMessage, promptPath string) bool {
	var p struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(payload, &p) != nil {
		return false
	}
	if p.Type != "message" || p.Role != "user" {
		return false
	}
	pointer := promptFilePointer(promptPath)
	for _, part := range p.Content {
		if strings.Contains(part.Text, pointer) {
			return true
		}
	}
	return false
}
