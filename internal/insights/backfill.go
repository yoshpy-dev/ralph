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

// ParseReport parses one docs/reports/*.md file into a BackfillEvent.
// The phase is inferred from the filename prefix.
// ts is set to the file's mtime (not now) so history ordering is meaningful.
// source_report_path is embedded in the event for dedup.
// Returns (nil, nil) when the file is not a recognised report type.
func ParseReport(path string) (*BackfillEvent, error) {
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

	slug, cycle := slugAndCycleFromFilename(base, phase)

	ev := &BackfillEvent{}
	ev.Schema = 1
	ev.TS = mtime.Format(time.RFC3339)
	ev.Slug = slug
	ev.Phase = phase
	ev.Cycle = cycle
	ev.Source = "backfill"
	ev.SourceReportPath = path

	// Flow is best-effort: we can't reliably detect standard vs loop from
	// report content in all cases, so we omit it (empty string → omitted from JSON).
	ev.Flow = detectFlow(path)

	switch phase {
	case "self_review":
		findings, miss, reason := parseSelfReviewFindings(path)
		if miss {
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
		} else {
			ev.Findings = findings
			ev.Verdict = selfReviewVerdict(findings)
		}
	case "verify":
		verdict, miss, reason := parseVerifyVerdict(path)
		if miss {
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
		} else {
			ev.Verdict = verdict
		}
	case "test":
		verdict, miss, reason := parseTestVerdict(path)
		if miss {
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
		} else {
			ev.Verdict = verdict
		}
	case "cross_review":
		triage, miss, reason := parseCrossReviewTriage(path)
		if miss {
			ev.ParseMiss = true
			ev.ParseMissReason = reason
			ev.Verdict = "n/a"
		} else {
			ev.Triage = triage
			ev.Verdict = crossReviewVerdict(triage)
		}
	}

	return ev, nil
}

// slugAndCycleFromFilename extracts the task slug and cycle from a report filename.
// Naming conventions handled:
//
//	self-review-<date>-<slug>.md   → slug=<slug>, cycle=1
//	self-review-<slug>.md          → slug=<slug>, cycle=1
//	verify-<date>-<slug>.md        → slug=<slug>, cycle=1
//	cross-review-triage-<slug>.md  → slug=<slug>, cycle=1
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

	// Cycle is always 1 for backfill from a single report file.
	// Multi-cycle addenda (cycle-2 markers in the report body) are not
	// a pattern seen in this repo's reports.
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

// parseCrossReviewTriage extracts triage counts from a cross-review report.
// It looks for "After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N".
func parseCrossReviewTriage(path string) (Triage, bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return Triage{}, true, fmt.Sprintf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	// Match: "After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N"
	triageRe := regexp.MustCompile(`(?i)After triage:\s*ACTION_REQUIRED=(\d+),\s*WORTH_CONSIDERING=(\d+),\s*DISMISSED=(\d+)`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		m := triageRe.FindStringSubmatch(scanner.Text())
		if m != nil {
			ar, _ := strconv.Atoi(m[1])
			wc, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[3])
			return Triage{
				ActionRequired:   ar,
				WorthConsidering: wc,
				Dismissed:        d,
			}, false, ""
		}
	}
	if err := scanner.Err(); err != nil {
		return Triage{}, true, fmt.Sprintf("scan: %v", err)
	}

	return Triage{}, true, "no 'After triage:' line found"
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
		bev, err := ParseReport(f)
		if err != nil {
			// Non-fatal: count as parse miss.
			stats.ParseMiss++
			continue
		}
		if bev == nil {
			// Unrecognised file type — skip silently.
			continue
		}
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

	return stats, nil
}
