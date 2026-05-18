package upgrade

import "strings"

// MergeChoice identifies how a single merge hunk should be resolved.
type MergeChoice int

const (
	MergeUseLocal MergeChoice = iota
	MergeUseTemplate
	MergeUseEdited
)

// MergeDecision stores the chosen replacement for one merge hunk.
type MergeDecision struct {
	Choice      MergeChoice
	EditedLines []string
}

// MergeHunk is a line-range conflict between local disk and new template,
// expressed relative to the old template baseline.
type MergeHunk struct {
	Index         int
	BaseStart     int
	BaseEnd       int
	LocalLines    []string
	TemplateLines []string
}

// NeedsDecision reports whether local and template differ for this hunk.
func (h MergeHunk) NeedsDecision() bool {
	return !equalSlices(h.LocalLines, h.TemplateLines)
}

// MergePlan contains all hunk-level replacements needed to transform the old
// template baseline into a resolved file.
type MergePlan struct {
	BaseLines               []string
	LocalLines              []string
	TemplateLines           []string
	LocalTrailingNewline    bool
	TemplateTrailingNewline bool
	Hunks                   []MergeHunk
}

// PlanMerge builds a line-based 3-way merge plan from old template, local disk,
// and new template content.
func PlanMerge(baseText, localText, templateText []byte) MergePlan {
	baseLines, _ := splitLines(baseText)
	localLines, localNL := splitLines(localText)
	templateLines, templateNL := splitLines(templateText)

	localEdits := diffEdits(baseLines, localLines)
	templateEdits := diffEdits(baseLines, templateLines)
	groups := mergeEditGroups(localEdits, templateEdits)

	hunks := make([]MergeHunk, 0, len(groups))
	for _, group := range groups {
		localReplacement := replacementForSpan(baseLines, localLines, localEdits, group.start, group.end)
		templateReplacement := replacementForSpan(baseLines, templateLines, templateEdits, group.start, group.end)
		hunks = append(hunks, MergeHunk{
			Index:         len(hunks),
			BaseStart:     group.start,
			BaseEnd:       group.end,
			LocalLines:    localReplacement,
			TemplateLines: templateReplacement,
		})
	}

	return MergePlan{
		BaseLines:               baseLines,
		LocalLines:              localLines,
		TemplateLines:           templateLines,
		LocalTrailingNewline:    localNL,
		TemplateTrailingNewline: templateNL,
		Hunks:                   hunks,
	}
}

// DecisionCount returns the number of hunks that require a user decision.
func (p MergePlan) DecisionCount() int {
	count := 0
	for _, h := range p.Hunks {
		if h.NeedsDecision() {
			count++
		}
	}
	return count
}

// Render applies decisions to the plan and returns resolved file content.
// Missing decisions default to keeping local content for safety.
func (p MergePlan) Render(decisions map[int]MergeDecision) []byte {
	resolved := make([]string, 0, len(p.TemplateLines))
	cursor := 0
	for _, h := range p.Hunks {
		if h.BaseStart > cursor {
			resolved = append(resolved, p.BaseLines[cursor:h.BaseStart]...)
		}
		decision, ok := decisions[h.Index]
		if !ok && !h.NeedsDecision() {
			decision = MergeDecision{Choice: MergeUseTemplate}
			ok = true
		}
		if !ok {
			decision = MergeDecision{Choice: MergeUseLocal}
		}
		switch decision.Choice {
		case MergeUseTemplate:
			resolved = append(resolved, h.TemplateLines...)
		case MergeUseEdited:
			resolved = append(resolved, decision.EditedLines...)
		default:
			resolved = append(resolved, h.LocalLines...)
		}
		cursor = h.BaseEnd
	}
	if cursor < len(p.BaseLines) {
		resolved = append(resolved, p.BaseLines[cursor:]...)
	}

	trailingNewline := p.TemplateTrailingNewline
	switch {
	case equalSlices(resolved, p.LocalLines):
		trailingNewline = p.LocalTrailingNewline
	case equalSlices(resolved, p.TemplateLines):
		trailingNewline = p.TemplateTrailingNewline
	case p.LocalTrailingNewline:
		trailingNewline = true
	}
	return JoinLines(resolved, trailingNewline)
}

// JoinLines serializes splitLines-style lines back to bytes.
func JoinLines(lines []string, trailingNewline bool) []byte {
	if len(lines) == 0 {
		if trailingNewline {
			return []byte("\n")
		}
		return nil
	}
	out := strings.Join(lines, "\n")
	if trailingNewline {
		out += "\n"
	}
	return []byte(out)
}

type mergeEdit struct {
	baseStart, baseEnd int
	newStart, newEnd   int
}

func diffEdits(base, changed []string) []mergeEdit {
	ops := lcsDiff(base, changed)
	var edits []mergeEdit
	baseIdx, newIdx := 0, 0
	for i := 0; i < len(ops); {
		if ops[i].kind == opEqual {
			baseIdx++
			newIdx++
			i++
			continue
		}
		edit := mergeEdit{baseStart: baseIdx, newStart: newIdx}
		for i < len(ops) && ops[i].kind != opEqual {
			switch ops[i].kind {
			case opDel:
				baseIdx++
			case opAdd:
				newIdx++
			}
			i++
		}
		edit.baseEnd = baseIdx
		edit.newEnd = newIdx
		edits = append(edits, edit)
	}
	return edits
}

type editGroup struct {
	start, end int
}

func mergeEditGroups(a, b []mergeEdit) []editGroup {
	edits := make([]mergeEdit, 0, len(a)+len(b))
	edits = append(edits, a...)
	edits = append(edits, b...)
	if len(edits) == 0 {
		return nil
	}
	sortMergeEdits(edits)

	groups := []editGroup{{start: edits[0].baseStart, end: edits[0].baseEnd}}
	for _, edit := range edits[1:] {
		last := &groups[len(groups)-1]
		if editsOverlapOrTouch(last.start, last.end, edit.baseStart, edit.baseEnd) {
			if edit.baseStart < last.start {
				last.start = edit.baseStart
			}
			if edit.baseEnd > last.end {
				last.end = edit.baseEnd
			}
			continue
		}
		groups = append(groups, editGroup{start: edit.baseStart, end: edit.baseEnd})
	}
	return groups
}

func sortMergeEdits(edits []mergeEdit) {
	for i := 1; i < len(edits); i++ {
		for j := i; j > 0; j-- {
			left, right := edits[j-1], edits[j]
			if left.baseStart < right.baseStart ||
				(left.baseStart == right.baseStart && left.baseEnd <= right.baseEnd) {
				break
			}
			edits[j-1], edits[j] = edits[j], edits[j-1]
		}
	}
}

func editsOverlapOrTouch(aStart, aEnd, bStart, bEnd int) bool {
	if aStart == aEnd && bStart == bEnd {
		return aStart == bStart
	}
	return bStart <= aEnd && bEnd >= aStart
}

func replacementForSpan(base, changed []string, edits []mergeEdit, start, end int) []string {
	out := make([]string, 0, end-start)
	cursor := start
	for _, edit := range edits {
		if edit.baseEnd < start || edit.baseStart > end {
			continue
		}
		if edit.baseStart < cursor {
			continue
		}
		if cursor < edit.baseStart {
			out = append(out, base[cursor:edit.baseStart]...)
		}
		out = append(out, changed[edit.newStart:edit.newEnd]...)
		cursor = edit.baseEnd
	}
	if cursor < end {
		out = append(out, base[cursor:end]...)
	}
	return out
}
