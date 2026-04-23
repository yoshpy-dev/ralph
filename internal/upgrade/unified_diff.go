package upgrade

import (
	"fmt"
	"strings"
)

// diffSeparator is the visual separator between the line-number gutter and the
// diff prefix. Multi-byte (`│` is U+2502, three bytes in UTF-8) — code that
// scans for this marker (e.g. Colorize) must use byte length.
const diffSeparator = " │ "

// UnifiedDiff returns a human-readable line-numbered diff representation of
// oldText → newText at line granularity. Labels are rendered in the
// `--- oldLabel` / `+++ newLabel` header. The algorithm is LCS-based with 3
// lines of context. Output is a display artifact — callers must not parse it.
//
// Each emitted change line carries two gutter columns showing the 1-based
// line numbers in the old and new files (blank when the line does not exist
// on that side). The hunk header replaces the cryptic `@@ -a,b +c,d @@`
// notation with a human-friendly range summary.
//
// When either side lacks a trailing newline, a single
// `\ No newline at end of file` marker is appended after the diff so the
// ambiguity is visible to a human reviewer.
func UnifiedDiff(oldText, newText []byte, oldLabel, newLabel string) string {
	oldLines, oldNL := splitLines(oldText)
	newLines, newNL := splitLines(newText)

	if equalSlices(oldLines, newLines) && oldNL == newNL {
		return ""
	}

	ops := lcsDiff(oldLines, newLines)
	hunks := groupHunks(ops, 3)

	width := gutterWidth(hunks)
	blank := strings.Repeat(" ", width)

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", oldLabel)
	fmt.Fprintf(&b, "+++ %s\n", newLabel)

	for _, h := range hunks {
		fmt.Fprintf(&b, "@@ 旧 %s  →  新 %s @@\n",
			formatRange(h.oldStart, h.oldCount),
			formatRange(h.newStart, h.newCount))

		oldNo := h.oldStart // 0-based; pre-increment before emit makes it 1-based.
		newNo := h.newStart
		for _, op := range h.ops {
			oldCol := blank
			newCol := blank
			var prefix byte
			switch op.kind {
			case opEqual:
				oldNo++
				newNo++
				oldCol = fmt.Sprintf("%*d", width, oldNo)
				newCol = fmt.Sprintf("%*d", width, newNo)
				prefix = ' '
			case opDel:
				oldNo++
				oldCol = fmt.Sprintf("%*d", width, oldNo)
				prefix = '-'
			case opAdd:
				newNo++
				newCol = fmt.Sprintf("%*d", width, newNo)
				prefix = '+'
			}
			fmt.Fprintf(&b, "%s %s%s%c%s\n", oldCol, newCol, diffSeparator, prefix, op.line)
		}
	}

	if !oldNL || !newNL {
		b.WriteString("\\ No newline at end of file\n")
	}

	return b.String()
}

// formatRange renders a hunk's line range. Empty sides (count == 0) collapse
// to "(空)" so the user can immediately tell that the file was created from
// nothing or fully deleted on one side.
func formatRange(start, count int) string {
	if count == 0 {
		return "(空)"
	}
	if count == 1 {
		return fmt.Sprintf("L%d", start+1)
	}
	return fmt.Sprintf("L%d–%d", start+1, start+count)
}

// gutterWidth returns the column width needed to right-align every line
// number that may appear across all hunks. Floors at 2 so single-digit files
// still produce a stable two-column gutter.
func gutterWidth(hunks []hunk) int {
	maxLine := 0
	for _, h := range hunks {
		if v := h.oldStart + h.oldCount; v > maxLine {
			maxLine = v
		}
		if v := h.newStart + h.newCount; v > maxLine {
			maxLine = v
		}
	}
	w := 1
	for n := maxLine; n >= 10; n /= 10 {
		w++
	}
	if w < 2 {
		w = 2
	}
	return w
}

// splitLines splits text on '\n'. The returned slice never has a trailing
// empty line caused by a final newline — instead, hasTrailingNewline reports
// whether the input ended with '\n'. Empty input yields a nil slice with
// hasTrailingNewline=false.
func splitLines(text []byte) (lines []string, hasTrailingNewline bool) {
	if len(text) == 0 {
		return nil, false
	}
	s := string(text)
	hasTrailingNewline = strings.HasSuffix(s, "\n")
	if hasTrailingNewline {
		s = s[:len(s)-1]
	}
	return strings.Split(s, "\n"), hasTrailingNewline
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type opKind int

const (
	opEqual opKind = iota
	opDel
	opAdd
)

type diffOp struct {
	kind opKind
	line string
}

// lcsDiff returns a sequence of equal/del/add ops transforming old into new.
// Classic LCS DP + backtrack. O(m*n) time and space, acceptable for scaffold
// files (tens to a few thousand lines).
func lcsDiff(oldLines, newLines []string) []diffOp {
	m, n := len(oldLines), len(newLines)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	ops := make([]diffOp, 0, m+n)
	i, j := 0, 0
	for i < m && j < n {
		if oldLines[i] == newLines[j] {
			ops = append(ops, diffOp{kind: opEqual, line: oldLines[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{kind: opDel, line: oldLines[i]})
			i++
		} else {
			ops = append(ops, diffOp{kind: opAdd, line: newLines[j]})
			j++
		}
	}
	for ; i < m; i++ {
		ops = append(ops, diffOp{kind: opDel, line: oldLines[i]})
	}
	for ; j < n; j++ {
		ops = append(ops, diffOp{kind: opAdd, line: newLines[j]})
	}
	return ops
}

type hunk struct {
	oldStart, newStart int
	oldCount, newCount int
	ops                []diffOp
}

// groupHunks collects runs of non-equal ops surrounded by up to `context` lines
// of equality on each side. Adjacent changes whose context windows overlap
// (distance ≤ 2*context) are merged into one hunk.
func groupHunks(ops []diffOp, context int) []hunk {
	type indexed struct {
		op     diffOp
		oldIdx int
		newIdx int
	}
	idx := make([]indexed, len(ops))
	oi, ni := 0, 0
	for k, op := range ops {
		idx[k] = indexed{op: op, oldIdx: oi, newIdx: ni}
		switch op.kind {
		case opEqual:
			oi++
			ni++
		case opDel:
			oi++
		case opAdd:
			ni++
		}
	}

	var hunks []hunk
	i := 0
	for i < len(idx) {
		if idx[i].op.kind == opEqual {
			i++
			continue
		}
		start := i
		for start > 0 && idx[start-1].op.kind == opEqual && i-start < context {
			start--
		}

		end := i
		for end < len(idx) {
			if idx[end].op.kind != opEqual {
				end++
				continue
			}
			look := end
			for look < len(idx) && idx[look].op.kind == opEqual && look-end < 2*context {
				look++
			}
			if look < len(idx) && idx[look].op.kind != opEqual {
				end = look
				continue
			}
			trailing := context
			if end+trailing > len(idx) {
				trailing = len(idx) - end
			}
			end += trailing
			break
		}

		hOps := make([]diffOp, 0, end-start)
		for k := start; k < end; k++ {
			hOps = append(hOps, idx[k].op)
		}
		var oldCount, newCount int
		for _, op := range hOps {
			switch op.kind {
			case opEqual:
				oldCount++
				newCount++
			case opDel:
				oldCount++
			case opAdd:
				newCount++
			}
		}
		hunks = append(hunks, hunk{
			oldStart: idx[start].oldIdx,
			newStart: idx[start].newIdx,
			oldCount: oldCount,
			newCount: newCount,
			ops:      hOps,
		})
		i = end
	}
	return hunks
}
