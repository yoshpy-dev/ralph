// Package protocol implements the typed inter-seat message protocol
// described in .claude/rules/ralph/agent-messaging.md: a small header block
// ("KEY: value" lines) followed by an optional blank-line-separated body,
// with a required TYPE enum and a per-TYPE TASK_ID requirement. This
// package is the enforcing implementation the rule doc refers to -- keep
// both in sync when either changes.
//
// The protocol is intentionally boring: no nested structures, no escaping
// rules beyond "one header per line". EVIDENCE-as-pointers (commit SHAs,
// file:line, report paths -- never raw code/log dumps) is a body-content
// convention this package cannot enforce directly; the body size cap below
// is the mechanical proxy for it.
package protocol

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Message TYPE enum. Every typed message exchanged between org seats must
// use exactly one of these values (case-sensitive, upper-case).
const (
	TypeTask      = "TASK"
	TypeResult    = "RESULT"
	TypeQuestion  = "QUESTION"
	TypeReview    = "REVIEW"
	TypeDecision  = "DECISION"
	TypeBlocked   = "BLOCKED"
	TypeContract  = "CONTRACT"
	TypeHeartbeat = "HEARTBEAT"
	TypeStop      = "STOP"
	TypeHello     = "HELLO"
	TypeAlert     = "ALERT"
)

// DefaultMaxBodyChars is the body size cap applied when a caller passes
// maxBodyChars <= 0. EVIDENCE must be pointers (commit SHA, file:line,
// report path), never inline code/log dumps -- this cap is the mechanical
// floor for that principle, not a substitute for it.
const DefaultMaxBodyChars = 2000

// validTypes is the TYPE enum as a set, for O(1) membership checks.
var validTypes = map[string]bool{
	TypeTask: true, TypeResult: true, TypeQuestion: true, TypeReview: true,
	TypeDecision: true, TypeBlocked: true, TypeContract: true,
	TypeHeartbeat: true, TypeStop: true, TypeHello: true,
	TypeAlert: true,
}

// taskIDRequiredTypes is the subset of the TYPE enum that must carry a
// non-blank TASK_ID header. QUESTION/DECISION/HEARTBEAT/STOP/HELLO/ALERT are
// exempt: they are not necessarily scoped to a single task.
var taskIDRequiredTypes = map[string]bool{
	TypeTask: true, TypeResult: true, TypeReview: true,
	TypeBlocked: true, TypeContract: true,
}

// Sentinel errors, wrapped with fmt.Errorf so callers can errors.Is against
// them while still getting a message-specific detail string.
var (
	ErrMissingType   = errors.New("protocol: message missing TYPE header")
	ErrUnknownType   = errors.New("protocol: unknown TYPE")
	ErrMissingTaskID = errors.New("protocol: TASK_ID required for this TYPE")
	ErrBodyTooLarge  = errors.New("protocol: body exceeds max size")
)

// Message is a parsed typed protocol message.
type Message struct {
	// Type is the TYPE header's value, verbatim (not validated against the
	// enum by Parse -- that is Validate's job).
	Type string
	// TaskID is the TASK_ID header's value, or "" if absent.
	TaskID string
	// Fields holds every other "KEY: value" header line (TYPE and TASK_ID
	// are promoted to their own struct fields above and are not duplicated
	// here).
	Fields map[string]string
	// Body is everything after the header block, verbatim (no trimming
	// beyond the header/body split itself).
	Body string
}

// Parse splits text into a header block and a body. The header block is
// read line by line starting at line 1: the first line must be a
// "TYPE: value" header (TYPE is mandatory and must come first). Header
// parsing then continues, one "KEY: value" line at a time, until either a
// blank line is reached (the body starts on the line after it) or a line
// that does not match "KEY: value" is reached (the body starts on that
// line itself, inclusive) -- whichever comes first. A message with no body
// at all (header-only) is valid; Body is "".
//
// Because header/body detection is line-shape based (not an explicit
// blank-line requirement), a body whose first line happens to look like
// "Key: value" will be misparsed as an additional header -- callers that
// want an unambiguous split should always separate the header block from
// the body with a blank line.
func Parse(text string) (Message, error) {
	lines := strings.Split(text, "\n")

	firstKey, firstVal, ok := splitHeaderLine(lines[0])
	if !ok || firstKey != "TYPE" {
		return Message{}, fmt.Errorf("%w: first line must be \"TYPE: <value>\", got %q", ErrMissingType, lines[0])
	}

	m := Message{Type: firstVal, Fields: map[string]string{}}

	i := 1
	for ; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			i++ // body starts after the blank separator line
			break
		}
		key, val, ok := splitHeaderLine(line)
		if !ok {
			break // non-header line: body starts here, inclusive
		}
		if key == "TASK_ID" {
			m.TaskID = val
			continue
		}
		m.Fields[key] = val
	}
	m.Body = strings.Join(lines[i:], "\n")

	return m, nil
}

// splitHeaderLine reports whether line is a well-formed "KEY: value"
// header: a non-blank key (no leading/trailing whitespace once trimmed,
// and no internal whitespace) followed by a colon.
func splitHeaderLine(line string) (key, val string, ok bool) {
	before, after, found := strings.Cut(line, ":")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(before)
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	val = strings.TrimSpace(after)
	return key, val, true
}

// Validate checks m against the protocol rules: TYPE must be a member of
// the enum, TASK_ID must be present (non-blank) for the TYPEs that require
// it, and the Body must not exceed maxBodyChars runes. maxBodyChars <= 0
// falls back to DefaultMaxBodyChars.
func Validate(m Message, maxBodyChars int) error {
	if maxBodyChars <= 0 {
		maxBodyChars = DefaultMaxBodyChars
	}

	if strings.TrimSpace(m.Type) == "" {
		return ErrMissingType
	}
	if !validTypes[m.Type] {
		return fmt.Errorf("%w: %q (must be one of: %s)", ErrUnknownType, m.Type, strings.Join(sortedTypeNames(), ", "))
	}
	if taskIDRequiredTypes[m.Type] && strings.TrimSpace(m.TaskID) == "" {
		return fmt.Errorf("%w: TYPE %s requires a non-blank TASK_ID header", ErrMissingTaskID, m.Type)
	}
	if n := utf8.RuneCountInString(m.Body); n > maxBodyChars {
		return fmt.Errorf("%w: body is %d chars, max is %d", ErrBodyTooLarge, n, maxBodyChars)
	}
	return nil
}

// ValidateText is the Parse+Validate convenience most callers want: parse
// text into a Message, then validate it against maxBodyChars.
func ValidateText(text string, maxBodyChars int) error {
	m, err := Parse(text)
	if err != nil {
		return err
	}
	return Validate(m, maxBodyChars)
}

// sortedTypeNames returns the TYPE enum in a stable (sorted) order, for
// deterministic error messages.
func sortedTypeNames() []string {
	names := make([]string, 0, len(validTypes))
	for t := range validTypes {
		names = append(names, t)
	}
	slices.Sort(names)
	return names
}
