package upgrade

import "strings"

// ConflictRegion is a line-range conflict between local disk and new template,
// expressed relative to the old template baseline.
type ConflictRegion struct {
	Index         int
	BaseStart     int
	BaseEnd       int
	LocalLines    []string
	TemplateLines []string
}

// NeedsResolution reports whether local and template differ for this region.
func (r ConflictRegion) NeedsResolution() bool {
	return !equalSlices(r.LocalLines, r.TemplateLines)
}

// MergePlan contains the line ranges needed to build a file-level conflict
// marker view from old template, local disk, and new template content.
type MergePlan struct {
	BaseLines               []string
	LocalLines              []string
	TemplateLines           []string
	LocalTrailingNewline    bool
	TemplateTrailingNewline bool
	Regions                 []ConflictRegion
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

	regions := make([]ConflictRegion, 0, len(groups))
	for _, group := range groups {
		localReplacement := replacementForSpan(baseLines, localLines, localEdits, group.start, group.end)
		templateReplacement := replacementForSpan(baseLines, templateLines, templateEdits, group.start, group.end)
		regions = append(regions, ConflictRegion{
			Index:         len(regions),
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
		Regions:                 regions,
	}
}

// ConflictCount returns the number of regions that require file-level conflict
// resolution.
func (p MergePlan) ConflictCount() int {
	count := 0
	for _, region := range p.Regions {
		if region.NeedsResolution() {
			count++
		}
	}
	return count
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
