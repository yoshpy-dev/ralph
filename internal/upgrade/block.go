package upgrade

import (
	"bytes"
	"fmt"
	"strings"
)

// BlockOutcome classifies the result of UpdateManagedBlock.
type BlockOutcome int

const (
	// BlockUnchanged means the existing block already equals the desired
	// managed content byte-for-byte; the caller should skip writing.
	BlockUnchanged BlockOutcome = iota
	// BlockUpdated means a well-formed existing block's interior was
	// replaced with the desired managed content.
	BlockUpdated
	// BlockAppended means no ralph markers were found in the file and a new
	// block was appended at EOF.
	BlockAppended
	// BlockMalformed means the marker pair is broken (unmatched, duplicate,
	// out of order, or belongs to a different surface); no content is
	// produced and the caller must leave the file untouched.
	BlockMalformed
)

// BlockResult is the outcome of UpdateManagedBlock.
type BlockResult struct {
	Outcome BlockOutcome
	Content []byte
	// Reason is populated only when Outcome == BlockMalformed.
	Reason string
}

// EndMarker is the exact line that closes a ralph managed block.
const EndMarker = "<!-- END RALPH MANAGED -->"

const (
	beginMarkerPrefix = "<!-- BEGIN RALPH MANAGED (ralph:"
	beginMarkerSuffix = ") -->"
)

// BeginMarker returns the exact BEGIN marker line for a given surface token
// (e.g. "agents-md", "gitignore").
func BeginMarker(surface string) string {
	return beginMarkerPrefix + surface + beginMarkerSuffix
}

// UpdateManagedBlock updates the ralph-owned block delimited by BeginMarker /
// EndMarker inside current, replacing only the interior with managed.
//
// It is pure bytes-in/bytes-out: it never touches the filesystem. Every byte
// outside the marker pair — including the markers themselves — is preserved
// exactly. Callers own file I/O and must not write Content when Outcome is
// BlockMalformed.
func UpdateManagedBlock(current []byte, surface string, managed []byte) BlockResult {
	lines := splitRawLines(current)

	var beginIdxs, endIdxs []int
	var beginSurfaces []string
	for i, l := range lines {
		if s, ok := isRalphBeginMarker(l.text); ok {
			beginIdxs = append(beginIdxs, i)
			beginSurfaces = append(beginSurfaces, s)
		}
		if l.text == EndMarker {
			endIdxs = append(endIdxs, i)
		}
	}

	reason := classifyMarkers(beginIdxs, endIdxs, beginSurfaces, surface)
	if reason != "" {
		return BlockResult{Outcome: BlockMalformed, Reason: reason}
	}

	dominantTerm := computeDominantTerm(lines)
	interior := managedInteriorLines(managed, dominantTerm)

	if len(beginIdxs) == 0 && len(endIdxs) == 0 {
		newLines := appendBlockLines(lines, surface, interior, dominantTerm)
		content := joinRawLines(newLines)
		return BlockResult{Outcome: BlockAppended, Content: content}
	}

	beginIdx, endIdx := beginIdxs[0], endIdxs[0]
	newLines := make([]rawLine, 0, len(lines)-(endIdx-beginIdx-1)+len(interior))
	newLines = append(newLines, lines[:beginIdx+1]...)
	newLines = append(newLines, interior...)
	newLines = append(newLines, lines[endIdx:]...)
	content := joinRawLines(newLines)

	if bytes.Equal(content, current) {
		return BlockResult{Outcome: BlockUnchanged, Content: content}
	}
	return BlockResult{Outcome: BlockUpdated, Content: content}
}

// classifyMarkers returns a non-empty malformed reason if the discovered
// marker indices/surfaces do not form exactly one well-formed block (or the
// complete absence of markers, which is not malformed — that's the append
// case handled by the caller).
func classifyMarkers(beginIdxs, endIdxs []int, beginSurfaces []string, wantSurface string) string {
	switch {
	case len(beginIdxs) > 1 && len(endIdxs) > 1:
		return "multiple BEGIN and END markers found"
	case len(beginIdxs) > 1:
		return "duplicate BEGIN marker found"
	case len(endIdxs) > 1:
		return "duplicate END marker found"
	case len(beginIdxs) == 1 && len(endIdxs) == 0:
		return "BEGIN marker found without a matching END marker"
	case len(beginIdxs) == 0 && len(endIdxs) == 1:
		return "END marker found without a matching BEGIN marker"
	case len(beginIdxs) == 1 && len(endIdxs) == 1 && endIdxs[0] < beginIdxs[0]:
		return "END marker appears before BEGIN marker"
	case len(beginIdxs) == 1 && len(endIdxs) == 1 && beginSurfaces[0] != wantSurface:
		return fmt.Sprintf("existing managed block has surface %q, expected %q", beginSurfaces[0], wantSurface)
	default:
		return ""
	}
}

// isRalphBeginMarker reports whether line is a well-formed ralph BEGIN
// marker for any surface, returning the surface token it names.
func isRalphBeginMarker(line string) (surface string, ok bool) {
	if !strings.HasPrefix(line, beginMarkerPrefix) || !strings.HasSuffix(line, beginMarkerSuffix) {
		return "", false
	}
	surface = strings.TrimSuffix(strings.TrimPrefix(line, beginMarkerPrefix), beginMarkerSuffix)
	if surface == "" {
		return "", false
	}
	return surface, true
}

// rawLine is one line of a file, split so that reconstruction (text + term
// for every line, concatenated in order) reproduces the original bytes
// exactly. term is "\n", "\r\n", or "" (only for a final line with no
// trailing newline).
type rawLine struct {
	text string
	term string
}

func splitRawLines(data []byte) []rawLine {
	if len(data) == 0 {
		return nil
	}
	s := string(data)
	var lines []rawLine
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\n' {
			continue
		}
		end := i
		term := "\n"
		if end > start && s[end-1] == '\r' {
			end--
			term = "\r\n"
		}
		lines = append(lines, rawLine{text: s[start:end], term: term})
		start = i + 1
	}
	if start < len(s) {
		lines = append(lines, rawLine{text: s[start:], term: ""})
	}
	return lines
}

func joinRawLines(lines []rawLine) []byte {
	var b bytes.Buffer
	for _, l := range lines {
		b.WriteString(l.text)
		b.WriteString(l.term)
	}
	return b.Bytes()
}

// computeDominantTerm picks the majority line terminator among lines that
// actually have one, defaulting to "\n" (including on ties or no data).
func computeDominantTerm(lines []rawLine) string {
	crlf, lf := 0, 0
	for _, l := range lines {
		switch l.term {
		case "\r\n":
			crlf++
		case "\n":
			lf++
		}
	}
	if crlf > lf {
		return "\r\n"
	}
	return "\n"
}

// managedInteriorLines splits managed content into rawLines terminated with
// term, suitable for insertion strictly between BEGIN and END markers. Empty
// managed content yields no lines (an empty block).
func managedInteriorLines(managed []byte, term string) []rawLine {
	if len(managed) == 0 {
		return nil
	}
	normalized := strings.ReplaceAll(string(managed), "\r\n", "\n")
	parts := strings.Split(normalized, "\n")
	if strings.HasSuffix(normalized, "\n") {
		parts = parts[:len(parts)-1]
	}
	lines := make([]rawLine, 0, len(parts))
	for _, p := range parts {
		lines = append(lines, rawLine{text: p, term: term})
	}
	return lines
}

// appendBlockLines returns current plus a newly appended managed block,
// separated from existing content by exactly one blank line (unless current
// is empty, or already ends with a blank line).
func appendBlockLines(current []rawLine, surface string, interior []rawLine, term string) []rawLine {
	lines := make([]rawLine, len(current))
	copy(lines, current)

	if len(lines) > 0 {
		last := &lines[len(lines)-1]
		if last.term == "" {
			last.term = term
		}
		if lines[len(lines)-1].text != "" {
			lines = append(lines, rawLine{text: "", term: term})
		}
	}

	lines = append(lines, rawLine{text: BeginMarker(surface), term: term})
	lines = append(lines, interior...)
	lines = append(lines, rawLine{text: EndMarker, term: term})
	return lines
}
