package org

import (
	"bufio"
	"bytes"
	"context"
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
// pointer sentence ralph itself passed the seat, PromptFilePointer(promptPath)
// (see codexResponseItemMentionsPrompt), not merely the bare path.
// spawnStarted is the spawn's own start time; only a session that began at
// or after spawnStarted (truncated to whole seconds, since that is the
// precision the manifest stores -- see AC-2b) can belong to this spawn,
// which is what lets a re-spawn of the same org_id/seat_id (same
// promptPath) tell its own session apart from an older one that happens to
// still be running or gets touched again later.
//
// until is how far past spawnStarted the date-directory walk reaches
// (codexSessionDateDirs) -- a session can start well after the spawn when
// codex's own model-retirement dialog is left open, since the record is
// only written once the dialog is answered and the first turn begins.
// Spawn's own poll passes spawnStarted for until (the observation happens
// moments after the spawn, so there is nothing later to reach); Stop
// passes its own current instant, since it can run long after the spawn.
// until earlier than spawnStarted, including the zero value, is treated
// as spawnStarted.
//
// This function is pure: no environment reads, no globals mutated, no
// goroutines, no writes. It never returns partial or guessed content -- see
// CodexModelObservation's doc comment. The non-nil error return is reserved
// for two content-free signals a caller might want to log: a candidate file
// that exists but could not be opened (e.g. permission denied), or ctx
// being done (see below); every other failure mode -- a missing
// sessionsDir, an unreadable date directory, a missing/unreadable candidate
// file, a malformed or oversized line -- degrades silently to
// CodexObservationNotFound with a nil error. Neither caller (Spawn's poll,
// Stop's single check) ever lets a codex CLI record-shape change or a
// permissions quirk fail the seat's own spawn or stop.
//
// ctx cancellation is cooperative and checked at three points: before each
// date directory and each directory entry considered while gathering
// candidates (codexRolloutCandidates), before opening each candidate file,
// and once per line while reading a candidate (scanRolloutRecord) -- so a
// caller-supplied deadline (e.g. observeCodexSpawnReceipt's own observation
// budget) bounds a single call, not just the gap between repeated calls.
// A pass cut short by ctx at any of those points returns
// CodexObservationNotFound alongside ctx's own error and NEVER
// CodexObservationFound, even if a match was already seen on an earlier
// candidate -- an unexamined later candidate could still have made the
// result ambiguous, so "cut short" can never be distinguished from
// "genuinely not found" well enough to report found. A pass that reaches
// its natural end (every candidate examined) returns its real result even
// if ctx expired right as it finished -- the identification is already
// sound at that point. Because the checks are cooperative, not preemptive,
// synchronous file I/O itself cannot be interrupted: this call can overrun
// ctx's deadline by the time of one read (at most codexObserveMaxLineBytes)
// or one syscall (ReadDir/Lstat/Open) already in flight when ctx becomes
// done.
func ObserveCodexEffectiveModel(ctx context.Context, sessionsDir, promptPath string, spawnStarted, until time.Time) (CodexModelObservation, error) {
	if sessionsDir == "" || promptPath == "" {
		return CodexModelObservation{Status: CodexObservationNotFound}, nil
	}
	if _, err := os.Stat(sessionsDir); err != nil {
		// Missing or unreadable sessionsDir: not an error (codex may simply
		// not be installed, or CODEX_HOME points somewhere ralph can't see).
		return CodexModelObservation{Status: CodexObservationNotFound}, nil
	}

	cutoff := spawnStarted.Truncate(time.Second)
	candidates, err := codexRolloutCandidates(ctx, sessionsDir, spawnStarted, until)
	if err != nil {
		// Cut short while still gathering candidates: the pass never even
		// finished deciding what to look at, let alone which one qualifies.
		return CodexModelObservation{Status: CodexObservationNotFound}, err
	}

	var (
		matchedModel string
		matchCount   int
		firstErr     error
	)
	for _, c := range candidates {
		matched, model, scanErr := scanRolloutRecord(ctx, c.path, promptPath, cutoff)
		if isCtxDoneErr(scanErr) {
			// Cut short before this candidate could even be opened, or
			// partway through reading it: whatever matched on an earlier
			// candidate cannot be trusted as the final answer (see this
			// function's own doc comment) -- unknown, never found.
			return CodexModelObservation{Status: CodexObservationNotFound}, scanErr
		}
		if scanErr != nil && firstErr == nil {
			firstErr = scanErr
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

// isCtxDoneErr reports whether err is exactly the ctx-cancellation signal
// scanRolloutRecord (or codexRolloutCandidates) returns when a pass was cut
// short -- context.Canceled or context.DeadlineExceeded, ctx's own Err() --
// as opposed to a genuine read error (a wrapped os.PathError; see
// scanRolloutRecord's own doc comment). Only a ctx-cut-short error ends the
// whole ObserveCodexEffectiveModel pass early; a genuine read error is
// recorded (firstErr) and scanning continues to the next candidate, exactly
// as before ctx was threaded through this file.
func isCtxDoneErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// codexRolloutCandidate is one file gathered by codexRolloutCandidates,
// carrying just enough to sort and open it.
type codexRolloutCandidate struct {
	path    string
	modTime time.Time
}

// codexRolloutCandidates walks the date-directory window codexSessionDateDirs
// returns for spawnStarted/until, collects every regular rollout-*.jsonl
// file whose ModTime is not older than spawnStarted minus
// codexObserveModTimeSlack, and returns them newest-first, capped at
// codexObserveMaxFiles. A symlink, FIFO, device, or directory that happens
// to match the name pattern is identified via os.Lstat (which never
// follows a symlink) and dropped -- never opened -- so a FIFO left behind
// in the tree cannot hang this call. A missing or unreadable date
// directory is silently skipped (per ObserveCodexEffectiveModel's doc
// comment, not an error). The ModTime pre-filter is anchored to
// spawnStarted only, not until -- it stays a cheap "cannot possibly be
// this spawn" filter, never a decision about how late a legitimate record
// may have been touched (see the constant's own doc comment).
//
// ctx.Err() is checked before each date directory's own ReadDir and before
// each of its entries' Lstat -- listing, stat'ing, and sorting are
// otherwise unbounded work that would run to completion regardless of a
// caller's deadline (ObserveCodexEffectiveModel's own doc comment). A
// non-nil error return means the walk was cut short: it is always exactly
// ctx.Err() (context.Canceled or context.DeadlineExceeded), never a
// directory-read error (those are skipped silently, as before), and the
// returned candidate slice is nil in that case -- a partial candidate list
// is never useful to the caller, which discards the whole pass on this
// error regardless of what had already been gathered.
func codexRolloutCandidates(ctx context.Context, sessionsDir string, spawnStarted, until time.Time) ([]codexRolloutCandidate, error) {
	minModTime := spawnStarted.Add(-codexObserveModTimeSlack)

	var candidates []codexRolloutCandidate
	for _, dateDir := range codexSessionDateDirs(spawnStarted, until) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		dirPath := filepath.Join(sessionsDir, dateDir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
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
	return candidates, nil
}

// isCodexRolloutFileName reports whether name matches codex's rollout
// record filename shape (rollout-*.jsonl).
func isCodexRolloutFileName(name string) bool {
	return strings.HasPrefix(name, codexRolloutFilePrefix) && strings.HasSuffix(name, codexRolloutFileSuffix)
}

// codexObserveMaxDateDirs caps how many date directories codexSessionDateDirs
// returns, keeping the EARLIEST ones when the [spawnStarted-1d, until+1d]
// span would exceed it. A seat left running -- or parked on a startup
// dialog -- for more than a month before Stop's second-chance observation
// runs is out of scope: 32 days of headroom is generous for the
// documented "codex's model-retirement dialog answered more than a day
// later" case this bound exists for, while still keeping the walk bounded
// no matter how stale a caller's until ends up being. Keeping the
// earliest days, not the latest, is deliberate: the session that answers
// a startup dialog still started on or soon after spawnStarted, so if a
// span ever needed trimming, the directories nearest the actual spawn are
// the ones most likely to hold the real record.
const codexObserveMaxDateDirs = 32

// codexSessionDateDirs returns the YYYY/MM/DD date directories (codex's
// own layout) to walk: from spawnStarted's local date minus one day,
// through until's local date plus one day, capped at
// codexObserveMaxDateDirs (keeping the earliest days -- see its own doc
// comment). until earlier than spawnStarted, including the zero value, is
// treated as spawnStarted, so a caller that only has the spawn time still
// gets the original three-directory window.
//
// A session's record lives in the directory of the day the session
// STARTED, no matter how much longer codex keeps writing to it afterward
// -- confirmed against this machine's real sessions directory (file
// metadata only): of 1,013 records, 57 were last modified on a later
// calendar day than their directory date (up to 17 days later), none had
// moved to a different directory, and every file name's embedded date
// matched its directory date exactly. But the session can also START
// later than the spawn: when codex's model-retirement dialog is open, the
// record is not written until the dialog is answered and the first turn
// begins -- the one real record that had the dialog shows the same
// ~2.5s gap between session_meta and turn_context as every dialog-free
// record, meaning the record was created at first-turn time, not at
// process launch. A seat left on that dialog for two days needs until to
// reach that day's directory, which is why this function takes it as a
// second, caller-supplied endpoint rather than assuming spawnStarted
// alone always covers the session's real start. The one-day pad on each
// side of the resulting span absorbs a UTC/local day-boundary mismatch
// between whatever clock named the directory (codex documents it as a
// "local timestamp") and either endpoint's own Location -- narrowing
// either side to exactly its own date would risk silently missing a real
// file created just across a midnight boundary.
//
// This function still never reads the live wall clock itself -- both
// endpoints are caller-supplied, so ObserveCodexEffectiveModel as a whole
// still touches no clock but the ones its caller already had.
func codexSessionDateDirs(spawnStarted, until time.Time) []string {
	if until.Before(spawnStarted) {
		until = spawnStarted
	}
	startDay := truncateToLocalDay(spawnStarted).AddDate(0, 0, -1)
	endDay := truncateToLocalDay(until).AddDate(0, 0, 1)

	var dirs []string
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		dirs = append(dirs, d.Format(codexSessionDateLayout))
		if len(dirs) >= codexObserveMaxDateDirs {
			break
		}
	}
	return dirs
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
// err is non-nil in two cases: path exists (it was already Lstat'd as a
// regular, name-matching, recently-modified file by the caller) but could
// not be opened -- e.g. permission denied -- which returns a plain wrapped
// os.PathError naming the operation and the path, never any line content;
// or ctx became done, checked once before opening path and once per line
// read thereafter, which returns ctx.Err() itself (see isCtxDoneErr, the
// caller's way to tell the two apart). Every other failure -- the file
// having vanished in the race between Lstat and Open, a malformed line, an
// unparsable timestamp -- degrades to matched=false, err=nil.
func scanRolloutRecord(ctx context.Context, path, promptPath string, cutoff time.Time) (matched bool, model string, err error) {
	if err := ctx.Err(); err != nil {
		return false, "", err
	}
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
		if err := ctx.Err(); err != nil {
			return false, "", err
		}
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
// PromptFilePointer(promptPath), spawn.go, e.g. "役割指示を読み込んで従っ
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
	pointer := PromptFilePointer(promptPath)
	for _, part := range p.Content {
		if strings.Contains(part.Text, pointer) {
			return true
		}
	}
	return false
}
