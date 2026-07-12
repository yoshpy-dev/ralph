package insights

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BackfillEvent is a parsed result from one report file, ready to become an Event.
type BackfillEvent struct {
	Event
	// ParseMiss is true when the parser could not extract meaningful values.
	ParseMiss bool
	// ParseMissReason describes why parsing yielded no useful data.
	ParseMissReason string
}

// BackfillStats tracks parse outcomes across a backfill run.
type BackfillStats struct {
	Parsed    int
	ParseMiss int
	Duplicate int
	Written   int
}

// ParseReport parses one docs/reports/*.md file into zero or more BackfillEvents.
// The phase is inferred from the filename prefix.
// ts is set to the file's mtime (not now) so history ordering is meaningful.
// source_report_path is embedded in each event for dedup.
//
// For cross-review-triage reports, one event is emitted per pipeline cycle found
// in the file. Cycle detection strategy:
//   - Each occurrence of "After triage: ACTION_REQUIRED=N, ..." is assigned to a
//     cycle in order: 1st occurrence = cycle 1, 2nd = cycle 2, etc.
//   - When an explicit "## Cycle N" section heading appears before an "After triage:"
//     line, the heading's cycle number takes precedence over the occurrence counter.
//     This matches the real report format in docs/reports/cross-review-triage-*.md
//     where the initial triage result appears in the file header (cycle 1) and
//     subsequent pipeline cycles are introduced by "## Cycle N (date)" headings.
//   - Explicit markers are preferred over occurrence counting when present, because
//     they are author-stated and survive reordering or future format changes.
//
// For self-review, verify, and test reports, a single event is emitted.
// Cycle-2 addenda sections are not present in this repo's current report format
// for these types — each phase gets a separate report file per pipeline run, so
// multi-cycle output is represented as multiple files rather than addenda in one.
//
// Returns (nil, nil) when the file is not a recognised report type.
func ParseReport(path string) ([]BackfillEvent, error) {
	// Normalize path to absolute so dedupe keys are stable regardless of
	// whether the caller passes a relative or absolute path. Fix 3: filepath.Abs
	// before storing to prevent rel-vs-abs duplicate events.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs %s: %w", path, err)
	}
	path = absPath

	base := filepath.Base(path)

	// Determine phase from filename prefix.
	var phase string
	switch {
	case strings.HasPrefix(base, "self-review-"):
		phase = "self_review"
	case strings.HasPrefix(base, "verify-"):
		phase = "verify"
	case strings.HasPrefix(base, "test-"):
		phase = "test"
	case strings.HasPrefix(base, "cross-review-triage-"):
		phase = "cross_review"
	default:
		// Not a recognised report type (walkthrough, sync-docs, codex-triage, etc.)
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	mtime := info.ModTime().UTC()

	slug, _ := slugAndCycleFromFilename(base, phase)
	flow := detectFlow(path)

	makeBase := func(cycle int) BackfillEvent {
		ev := BackfillEvent{}
		ev.Schema = 1
		ev.TS = mtime.Format(time.RFC3339)
		ev.Slug = slug
		ev.Phase = phase
		ev.Cycle = cycle
		ev.Source = "backfill"
		ev.SourceReportPath = path
		// Flow is best-effort: we can't reliably detect standard vs loop from
		// report content in all cases, so we omit it (empty string → omitted from JSON).
		ev.Flow = flow
		return ev
	}

	switch phase {
	case "self_review":
		ev := makeBase(1)
		findings, miss, reason := parseSelfReviewFindings(path)
		if miss {
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
		} else {
			ev.Findings = findings
			ev.Verdict = selfReviewVerdict(findings)
		}
		return []BackfillEvent{ev}, nil

	case "verify":
		ev := makeBase(1)
		verdict, miss, reason := parseVerifyVerdict(path)
		if miss {
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
		} else {
			ev.Verdict = verdict
		}
		return []BackfillEvent{ev}, nil

	case "test":
		ev := makeBase(1)
		verdict, miss, reason := parseTestVerdict(path)
		if miss {
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
		} else {
			ev.Verdict = verdict
		}
		return []BackfillEvent{ev}, nil

	case "cross_review":
		// parseCrossReviewAllCycles returns one entry per cycle found in the file.
		entries, miss, reason := parseCrossReviewAllCycles(path)
		if miss {
			ev := makeBase(1)
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
			return []BackfillEvent{ev}, nil
		}
		var evs []BackfillEvent
		for _, entry := range entries {
			ev := makeBase(entry.cycle)
			ev.Triage = entry.triage
			ev.Verdict = crossReviewVerdict(entry.triage)
			evs = append(evs, ev)
		}
		return evs, nil
	}

	return nil, nil
}

// slugAndCycleFromFilename extracts the task slug from a report filename.
// The cycle return value is always 1 — callers that need per-body-cycle
// splitting (cross_review) use parseCrossReviewAllCycles instead.
//
// Naming conventions handled:
//
//	self-review-<date>-<slug>.md   → slug=<slug>
//	self-review-<slug>.md          → slug=<slug>
//	verify-<date>-<slug>.md        → slug=<slug>
//	cross-review-triage-<slug>.md  → slug=<slug>
func slugAndCycleFromFilename(base, phase string) (slug string, cycle int) {
	// Strip phase prefix and .md suffix.
	var prefix string
	switch phase {
	case "self_review":
		prefix = "self-review-"
	case "verify":
		prefix = "verify-"
	case "test":
		prefix = "test-"
	case "cross_review":
		prefix = "cross-review-triage-"
	}
	name := strings.TrimPrefix(base, prefix)
	name = strings.TrimSuffix(name, ".md")

	// Try stripping a leading YYYY-MM-DD- date.
	dateRe := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)$`)
	if m := dateRe.FindStringSubmatch(name); m != nil {
		slug = m[1]
	} else {
		slug = name
	}

	// Cycle is always 1 for the filename-level assignment; body-level
	// cycle splitting is handled by parseCrossReviewAllCycles for cross_review.
	cycle = 1
	return slug, cycle
}

// detectFlow tries to infer "standard" or "loop" from report content.
// Returns "" when not derivable — the caller omits the field.
func detectFlow(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for i := 0; scanner.Scan() && i < 20; i++ {
		line := strings.ToLower(scanner.Text())
		if strings.Contains(line, "ralph-pipeline") || strings.Contains(line, "loop") {
			return "loop"
		}
		if strings.Contains(line, "standard flow") || strings.Contains(line, "/work") {
			return "standard"
		}
	}
	return ""
}

// parseSelfReviewFindings counts finding severities from a self-review report.
// It looks for table rows whose first column is a severity keyword.
func parseSelfReviewFindings(path string) (Findings, bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return Findings{}, true, fmt.Sprintf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	var findings Findings
	found := false

	// Match table rows: | SEVERITY | ... |
	// The severity must be the first pipe-delimited cell (case-insensitive).
	severityRe := regexp.MustCompile(`^\|\s*(CRITICAL|HIGH|MEDIUM|LOW)\s*\|`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		m := severityRe.FindStringSubmatch(strings.ToUpper(line))
		if m == nil {
			continue
		}
		found = true
		switch strings.ToUpper(m[1]) {
		case "CRITICAL":
			findings.Critical++
		case "HIGH":
			findings.High++
		case "MEDIUM":
			findings.Medium++
		case "LOW":
			findings.Low++
		}
	}

	if err := scanner.Err(); err != nil {
		return Findings{}, true, fmt.Sprintf("scan: %v", err)
	}

	// A self-review with no findings table rows is valid (no issues found).
	// ParseMiss only when the file is completely unreadable.
	_ = found
	return findings, false, ""
}

// selfReviewVerdict returns "pass" when no CRITICAL/HIGH findings, else "fail".
func selfReviewVerdict(f Findings) string {
	if f.Critical > 0 || f.High > 0 {
		return "fail"
	}
	return "pass"
}

// parseVerifyVerdict extracts the PASS/FAIL verdict from a verify report.
// It looks for "## Verdict: PASS" or "## Verdict: FAIL" (case-insensitive).
func parseVerifyVerdict(path string) (string, bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return "", true, fmt.Sprintf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Match: ## Verdict: PASS or ## Verdict: FAIL (with optional trailing text)
	verdictRe := regexp.MustCompile(`(?i)^##\s+Verdict[:\s]+(PASS|FAIL)`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		m := verdictRe.FindStringSubmatch(scanner.Text())
		if m != nil {
			return strings.ToLower(m[1]), false, ""
		}
	}
	if err := scanner.Err(); err != nil {
		return "", true, fmt.Sprintf("scan: %v", err)
	}

	return "", true, "no verdict line found (## Verdict: PASS|FAIL)"
}

// parseTestVerdict extracts the verdict from a test report.
// It looks for:
//  1. "## Verdict" followed by "- Pass:" / "- Fail:" lines
//  2. A "Fail: 0" line (pass) or "Fail: N" line with N>0 (fail)
func parseTestVerdict(path string) (string, bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return "", true, fmt.Sprintf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Pattern 1: "- Fail: N" line in the Verdict section.
	failLineRe := regexp.MustCompile(`(?i)^-\s+Fail:\s*(\d+)`)
	// Pattern 2: "- Pass: N/M" or "- Pass: N" in the Verdict section.
	passLineRe := regexp.MustCompile(`(?i)^-\s+Pass:\s*\d+`)

	inVerdict := false
	var failCount int
	var hasFailLine, hasPassLine bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if regexp.MustCompile(`(?i)^##\s+Verdict`).MatchString(trimmed) {
			inVerdict = true
			continue
		}
		if inVerdict && strings.HasPrefix(trimmed, "## ") {
			break // left Verdict section
		}
		if !inVerdict {
			continue
		}

		if m := failLineRe.FindStringSubmatch(trimmed); m != nil {
			hasFailLine = true
			n, _ := strconv.Atoi(m[1])
			failCount = n
		}
		if passLineRe.MatchString(trimmed) {
			hasPassLine = true
		}
	}
	if err := scanner.Err(); err != nil {
		return "", true, fmt.Sprintf("scan: %v", err)
	}

	if hasFailLine {
		if failCount == 0 {
			return "pass", false, ""
		}
		return "fail", false, ""
	}
	if hasPassLine {
		// Pass line found but no Fail line — treat as pass.
		return "pass", false, ""
	}

	return "", true, "no verdict section with Fail: N found"
}

// crossReviewCycleEntry holds one cycle's triage result from a cross-review report.
type crossReviewCycleEntry struct {
	cycle  int
	triage Triage
}

// parseCrossReviewAllCycles extracts triage counts for every pipeline cycle
// present in a cross-review-triage report. It looks for two kinds of markers:
//
//  1. Explicit section headings of the form "## Cycle N" or "## Cycle N (date)".
//     When such a heading appears before an "After triage:" line, the heading's N
//     is used as the cycle number, overriding the occurrence counter. This matches
//     the real report format in docs/reports/cross-review-triage-*.md where cycle-1
//     data appears in the file header and subsequent cycles start a new "## Cycle N"
//     section (e.g. "## Cycle 2 (2026-07-11)" at line 51 of loop-model-routing).
//
//  2. "After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N" lines.
//     Each occurrence produces one event. When no explicit "## Cycle N" heading has
//     been seen since the previous triage line, the occurrence counter (starting at 1)
//     determines the cycle number.
//
// Explicit markers are preferred over occurrence counting because they are
// author-stated and survive future format changes or section reordering.
func parseCrossReviewAllCycles(path string) ([]crossReviewCycleEntry, bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, true, fmt.Sprintf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Match: "After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N"
	triageRe := regexp.MustCompile(`(?i)After triage:\s*ACTION_REQUIRED=(\d+),\s*WORTH_CONSIDERING=(\d+),\s*DISMISSED=(\d+)`)
	// Match: "## Cycle N" or "## Cycle N (date)" — explicit cycle section headings.
	cycleHeadingRe := regexp.MustCompile(`(?i)^##\s+Cycle\s+(\d+)`)

	var entries []crossReviewCycleEntry
	occurrenceCount := 0
	// pendingCycle tracks the cycle number from the most recent "## Cycle N"
	// heading seen since the last "After triage:" line. 0 means no explicit
	// heading has been seen; in that case occurrence counting is used.
	pendingCycle := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for an explicit "## Cycle N" heading.
		if m := cycleHeadingRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			pendingCycle = n
			continue
		}

		// Check for a triage summary line.
		if m := triageRe.FindStringSubmatch(line); m != nil {
			occurrenceCount++
			ar, _ := strconv.Atoi(m[1])
			wc, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[3])

			cycle := occurrenceCount
			if pendingCycle > 0 {
				// Explicit heading takes precedence over occurrence counter.
				cycle = pendingCycle
			}
			// Reset pending heading after consuming it.
			pendingCycle = 0

			entries = append(entries, crossReviewCycleEntry{
				cycle: cycle,
				triage: Triage{
					ActionRequired:   ar,
					WorthConsidering: wc,
					Dismissed:        d,
				},
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, true, fmt.Sprintf("scan: %v", err)
	}

	if len(entries) == 0 {
		return nil, true, "no 'After triage:' line found"
	}
	return entries, false, ""
}

// crossReviewVerdict returns the verdict based on triage counts.
func crossReviewVerdict(t Triage) string {
	if t.ActionRequired > 0 {
		return "action_required"
	}
	return "pass"
}

// DedupeKey returns the deduplication key for a backfill event.
// key = source_report_path + ":" + phase + ":" + cycle
func DedupeKey(ev Event) string {
	return fmt.Sprintf("%s:%s:%d", ev.SourceReportPath, ev.Phase, ev.Cycle)
}

// LoadExistingDedupeKeys reads all existing event files in eventsDir and
// returns a set of dedup keys for events that were written by backfill
// (source=="backfill") and carry source_report_path.
func LoadExistingDedupeKeys(eventsDir string) (map[string]bool, error) {
	events, _, err := ReadEvents(eventsDir)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]bool)
	for _, ev := range events {
		if ev.Source == "backfill" && ev.SourceReportPath != "" {
			keys[DedupeKey(ev)] = true
		}
	}
	return keys, nil
}

// AppendBackfillEvent appends one backfill event to the appropriate JSONL file
// in eventsDir. File name: <YYYY-MM-DD>-<slug>.jsonl (date from event TS).
func AppendBackfillEvent(eventsDir string, ev Event) error {
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", eventsDir, err)
	}

	// Derive date from event TS.
	date := "0000-00-00"
	if t, err := time.Parse(time.RFC3339, ev.TS); err == nil {
		date = t.Format("2006-01-02")
	}

	filename := fmt.Sprintf("%s-%s.jsonl", date, ev.Slug)
	path := filepath.Join(eventsDir, filename)

	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

// RunBackfill scans reportsDir for recognised report files, parses each one,
// deduplicates against existing events in eventsDir, and either prints (dry-run)
// or appends (apply) the new events.
// Returns BackfillStats describing the outcome.
func RunBackfill(reportsDir, eventsDir string, apply bool) (BackfillStats, error) {
	var stats BackfillStats

	// Load existing dedup keys.
	existing, err := LoadExistingDedupeKeys(eventsDir)
	if err != nil {
		return stats, fmt.Errorf("loading existing events: %w", err)
	}

	// Glob report files.
	pattern := filepath.Join(reportsDir, "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return stats, fmt.Errorf("globbing %s: %w", reportsDir, err)
	}

	for _, f := range files {
		bevs, err := ParseReport(f)
		if err != nil {
			// Non-fatal: count as parse miss.
			stats.ParseMiss++
			continue
		}
		if bevs == nil {
			// Unrecognised file type — skip silently.
			continue
		}

		for i := range bevs {
			bev := &bevs[i]
			if bev.ParseMiss {
				stats.ParseMiss++
				continue
			}

			key := DedupeKey(bev.Event)
			if existing[key] {
				stats.Duplicate++
				continue
			}

			stats.Parsed++

			if apply {
				if err := AppendBackfillEvent(eventsDir, bev.Event); err != nil {
					return stats, fmt.Errorf("appending event for %s: %w", f, err)
				}
				// Mark as existing so a second call in the same run dedupes correctly.
				existing[key] = true
				stats.Written++
			}
		}
	}

	return stats, nil
}
