package upgrade

import (
	"fmt"
	"strings"
)

// UnifiedDiff returns a minimal unified-diff representation of oldText → newText
// at line granularity. Labels are rendered in the `--- oldLabel` / `+++ newLabel`
// header. The algorithm is LCS-based with 3 lines of context. Output is a
// display artifact — callers must not parse it.
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

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", oldLabel)
	fmt.Fprintf(&b, "+++ %s\n", newLabel)

	for _, h := range hunks {
		oldStart := h.oldStart + 1
		newStart := h.newStart + 1
		if h.oldCount == 0 {
			oldStart = h.oldStart
		}
		if h.newCount == 0 {
			newStart = h.newStart
		}
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart, h.oldCount, newStart, h.newCount)
		for _, op := range h.ops {
			switch op.kind {
			case opEqual:
				b.WriteString(" ")
			case opDel:
				b.WriteString("-")
			case opAdd:
				b.WriteString("+")
			}
			b.WriteString(op.line)
			b.WriteString("\n")
		}
	}

	if !oldNL || !newNL {
		b.WriteString("\\ No newline at end of file\n")
	}

	return b.String()
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
