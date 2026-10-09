package org

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Split plans (plan docs/plans/active/2026-10-09-org-feature-worktree.md,
// spec FR-4). A split plan divides one piece of work into features, and
// `ralph org start --plan <path> --feature <slug>` starts one org per feature
// from it. It is a Markdown file `<id>.md` directly in the org state dir's
// `splits/` directory (outside git):
//
//	# <title>
//
//	- Status: Approved
//	- Approved: <date> sha256:<the value scripts/plan-visual.sh digest prints>
//
//	## Features
//
//	### <slug>
//
//	- Type: feat
//	- Reserve: internal/auth/, docs/auth.md
//	- Depends on: none
//
//	<the feature's body: its objective and acceptance criteria>
//
// The approval digest is the one scripts/plan-visual.sh digest prints for any
// plan (PlanDigest). That digest skips some lines, so a split plan must not
// hold them anywhere except the one `- Status:` and one `- Approved:` line of
// its header: every line that could change after the approval is then
// covered by the digest.

// splitIDPattern is the shape of a split plan id, the file name without
// `.md`. The id goes into the org's reservation record, so it cannot hold
// whitespace or a comma.
var splitIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// approvedDigestPattern is the shape of the value after `sha256:` on the
// `- Approved:` line, as scripts/plan-visual.sh digest prints it.
var approvedDigestPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

// splitFeatureTypes are the values a feature's `- Type:` may take: the branch
// types /plan offers (.claude/skills/plan/SKILL.md step 4). The feature's
// branch is `<type>/<slug>`.
var splitFeatureTypes = []string{"feat", "fix", "docs", "chore", "refactor", "test", "ci", "build", "perf", "release", "security"}

const (
	defaultSplitFeatureType = "feat"
	splitFeaturesHeading    = "## Features"
	planProgressHeading     = "## Progress checklist"
	splitDependsOnNone      = "none"

	splitFieldType      = "Type"
	splitFieldReserve   = "Reserve"
	splitFieldDependsOn = "Depends on"
)

// splitFieldKeys are the field lines of a feature section, `- <key>: <value>`.
var splitFieldKeys = []string{splitFieldType, splitFieldReserve, splitFieldDependsOn}

// digestSkipsReason ends the error for a line a split plan must not hold.
const digestSkipsReason = "scripts/plan-visual.sh digest skips it, so it could change after the approval without changing the digest"

// SplitPlan is a split plan read by LoadSplitPlan.
type SplitPlan struct {
	ID             string // file name without .md
	Path           string // absolute path
	Status         string // value of the header `- Status:` line, trimmed ("" if absent)
	ApprovedDigest string // the 12 hex digits after `sha256:` on the header `- Approved:` line ("" if absent or not 12 hex digits)
	Digest         string // PlanDigest of the file bytes
	Features       []SplitFeature
}

// SplitFeature is one `### <slug>` section under a split plan's `## Features`.
type SplitFeature struct {
	Slug      string
	Type      string   // default "feat"
	Reserve   []string // NormalizeReservePaths output
	DependsOn []string // slugs of other features of the same plan, nil for none
	Body      string   // the section without its heading and field lines, leading and trailing blank lines trimmed
}

// SplitPlansDirIn returns the directory that holds the split plans of an
// already-resolved org state directory (see ManifestPathIn).
func SplitPlansDirIn(stateDir string) string {
	return filepath.Join(stateDir, "splits")
}

// PlanDigest returns the first 12 hex digits of the SHA-256 that
// scripts/plan-visual.sh digest computes (its digest_body awk). The script
// reads the file as "\n"-separated lines and prints each kept line followed
// by "\n", so a missing final newline counts as present and an empty file
// has no lines. It leaves out the `## Progress checklist` section (from a
// line equal to that heading up to the next line starting `## `) and every
// line starting `- Status:`, `- Approved:` or `- Branch:`, and reads a line
// starting, after whitespace, `- [x]` or `- [X]` as `- [ ]`. Lines compare as
// bytes, so a CRLF line keeps its "\r" and `## Progress checklist\r` is not
// that heading. TestPlanDigest_MatchesScript runs the script on the same
// inputs to keep the two the same.
func PlanDigest(data []byte) string {
	h := sha256.New()
	inProgress := false
	for _, line := range planLines(data) {
		if strings.HasPrefix(line, "## ") {
			inProgress = line == planProgressHeading
		}
		if inProgress || digestSkipsLine(line) {
			continue
		}
		if i := checkedBoxIndex(line); i >= 0 {
			line = line[:i] + " " + line[i+1:]
		}
		_, _ = io.WriteString(h, line+"\n")
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// planLines splits a plan file into its "\n"-separated lines, the way awk
// reads records: a final "\n" ends the last line instead of starting another,
// and an empty file has none.
func planLines(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

// digestSkipsLine reports whether PlanDigest leaves line out wherever it
// stands (the awk pattern `^- (Status|Approved|Branch):`).
func digestSkipsLine(line string) bool {
	return isStatusOrApprovedLine(line) || strings.HasPrefix(line, "- Branch:")
}

func isStatusOrApprovedLine(line string) bool {
	return strings.HasPrefix(line, "- Status:") || strings.HasPrefix(line, "- Approved:")
}

// checkedBoxIndex returns the index of the x in a line that starts, after
// whitespace, with `- [x]` or `- [X]` (the awk pattern
// `^[[:space:]]*- \[[xX]\]` in the C locale), and -1 for any other line.
func checkedBoxIndex(line string) int {
	rest := strings.TrimLeft(line, " \t\v\f\r")
	if strings.HasPrefix(rest, "- [x]") || strings.HasPrefix(rest, "- [X]") {
		return len(line) - len(rest) + len("- [")
	}
	return -1
}

// ResolveSplitPlanPath returns the absolute path, with symlinks resolved, and
// the id of the split plan at path, which must be a regular file `<id>.md`
// directly in SplitPlansDirIn(stateDir). Both sides are compared after
// filepath.Abs and filepath.EvalSymlinks. Anything else is an error that
// names the expected directory.
func ResolveSplitPlanPath(stateDir, path string) (abs, id string, err error) {
	dir := resolvedOrClean(mustAbs(SplitPlansDirIn(stateDir)))
	expect := "a split plan is a file <id>.md directly in " + dir
	given, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("org: split plan %s: %w", path, err)
	}
	resolved, err := filepath.EvalSymlinks(given)
	if err != nil {
		return "", "", fmt.Errorf("org: split plan %s: %w; %s", given, pathErrorCause(err), expect)
	}
	if filepath.Dir(resolved) != dir {
		where := given
		if resolved != given {
			where = fmt.Sprintf("%s (resolved to %s)", given, resolved)
		}
		return "", "", fmt.Errorf("org: split plan %s: not directly in %s: %s", where, dir, expect)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", "", fmt.Errorf("org: split plan %s: %w; %s", resolved, pathErrorCause(err), expect)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("org: split plan %s: not a regular file: %s", resolved, expect)
	}
	if id, err = splitIDFromPath(resolved); err != nil {
		return "", "", fmt.Errorf("%w; %s", err, expect)
	}
	return resolved, id, nil
}

// splitIDFromPath returns the split plan id of path, its file name without
// `.md`.
func splitIDFromPath(path string) (string, error) {
	id, ok := strings.CutSuffix(filepath.Base(path), ".md")
	if !ok {
		return "", fmt.Errorf("org: split plan %s: the file name must be <id>.md", path)
	}
	if !splitIDPattern.MatchString(id) {
		return "", fmt.Errorf("org: split plan %s: id %q (the file name without .md) must match %s", path, id, splitIDPattern)
	}
	return id, nil
}

// pathErrorCause drops the operation and path a *fs.PathError repeats, since
// the caller's message already names the path.
func pathErrorCause(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// LoadSplitPlan reads the split plan at path and checks its format: the file
// name, the header, the `## Features` section and its features, and the lines
// the approval digest skips (see the comment at the top of this file). It
// does not check the approval; see CheckApproved.
func LoadSplitPlan(path string) (*SplitPlan, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("org: split plan %s: %w", path, err)
	}
	id, err := splitIDFromPath(abs)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("org: split plan %s: %w", abs, pathErrorCause(err))
	}
	p, err := parseSplitPlan(data)
	if err != nil {
		return nil, fmt.Errorf("org: split plan %s: %w", abs, err)
	}
	p.ID, p.Path, p.Digest = id, abs, PlanDigest(data)
	return p, nil
}

// CheckApproved returns nil when the plan's `- Status:` is `Approved` and the
// digest on its `- Approved:` line equals the digest of its current bytes.
func (p *SplitPlan) CheckApproved() error {
	switch {
	case p.Status == "":
		return fmt.Errorf("org: split plan %s: not approved: it has no - Status: line in its header", p.Path)
	case p.Status != "Approved":
		return fmt.Errorf("org: split plan %s: not approved: - Status: is %q, want Approved", p.Path, p.Status)
	case p.ApprovedDigest == "":
		return fmt.Errorf("org: split plan %s: no approval digest: write - Approved: <date> sha256:<digest> in its header "+
			"with the 12 hex digits scripts/plan-visual.sh digest %s prints", p.Path, p.Path)
	case p.ApprovedDigest != p.Digest:
		return fmt.Errorf("org: split plan %s: changed after its approval: approved digest %s, current digest %s; "+
			"review the plan and record the approval again with scripts/plan-visual.sh digest %s",
			p.Path, p.ApprovedDigest, p.Digest, p.Path)
	}
	return nil
}

// Feature returns the feature named slug.
func (p *SplitPlan) Feature(slug string) (SplitFeature, bool) {
	for _, f := range p.Features {
		if f.Slug == slug {
			return f, true
		}
	}
	return SplitFeature{}, false
}

// splitFeatureDraft is a feature section while parseSplitPlan reads it.
type splitFeatureDraft struct {
	SplitFeature
	line        int            // line of the `### <slug>` heading
	fieldLine   map[string]int // line of each field line seen
	bodyLines   []string
	dependsLine int
}

// splitHeader is the header (the lines before the first `## ` line) while
// parseSplitPlan reads it.
type splitHeader struct {
	status, approvedDigest   string
	statusLine, approvedLine int
}

// parseSplitPlan parses a split plan's bytes. Lines are split on "\n" and
// lose one trailing "\r". Errors name the line and are wrapped by
// LoadSplitPlan with the plan's path.
func parseSplitPlan(data []byte) (*SplitPlan, error) {
	var (
		header       splitHeader
		inHeader     = true
		featuresLine int
		inFeatures   bool
		features     []*splitFeatureDraft
		cur          *splitFeatureDraft
	)
	for i, raw := range planLines(data) {
		n, line := i+1, strings.TrimSuffix(raw, "\r")
		if err := rejectDigestSkippedLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		if strings.HasPrefix(line, "## ") {
			inHeader, cur = false, nil
			inFeatures = strings.TrimRight(line, " \t") == splitFeaturesHeading
			if inFeatures && featuresLine != 0 {
				return nil, fmt.Errorf("line %d: a second %s heading (the first is on line %d)", n, splitFeaturesHeading, featuresLine)
			}
			if inFeatures {
				featuresLine = n
			}
			continue
		}
		if inHeader {
			if err := header.read(line, n); err != nil {
				return nil, err
			}
			continue
		}
		if isStatusOrApprovedLine(line) {
			return nil, fmt.Errorf("line %d: a - Status: or - Approved: line after the header: %s; "+
				"keep one of each in the header, before the first ## heading", n, digestSkipsReason)
		}
		if !inFeatures {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "### "); ok {
			f, err := newSplitFeatureDraft(strings.TrimSpace(rest), n, features)
			if err != nil {
				return nil, err
			}
			features, cur = append(features, f), f
			continue
		}
		if cur == nil {
			if key, _, ok := splitFieldLine(line); ok {
				return nil, fmt.Errorf("line %d: a - %s: line before the first ### <slug> heading of %s", n, key, splitFeaturesHeading)
			}
			continue
		}
		if err := cur.read(line, n); err != nil {
			return nil, err
		}
	}
	return finishSplitPlan(header, featuresLine, features)
}

// rejectDigestSkippedLine refuses the lines PlanDigest skips or rewrites
// wherever they stand: a `- Branch:` line, the `## Progress checklist`
// heading, and a checked box. The `- Status:` and `- Approved:` lines are
// refused by position in parseSplitPlan.
func rejectDigestSkippedLine(line string) error {
	switch {
	case strings.HasPrefix(line, "- Branch:"):
		return fmt.Errorf("a - Branch: line is not allowed in a split plan: %s (the branch of a feature is <type>/<slug>)", digestSkipsReason)
	case line == planProgressHeading:
		return fmt.Errorf("a %s section is not allowed in a split plan: %s", planProgressHeading, digestSkipsReason)
	case checkedBoxIndex(line) >= 0:
		return errors.New("a checked box (- [x] or - [X]) is not allowed in a split plan: scripts/plan-visual.sh digest reads it as - [ ], " +
			"so it could change after the approval without changing the digest; write - [ ] or plain text")
	}
	return nil
}

// read takes one header line: at most one `- Status:` and one `- Approved:`
// line; anything else is free text.
func (h *splitHeader) read(line string, n int) error {
	if value, ok := strings.CutPrefix(line, "- Status:"); ok {
		if h.statusLine != 0 {
			return fmt.Errorf("line %d: a second - Status: line (the first is on line %d): %s", n, h.statusLine, digestSkipsReason)
		}
		h.status, h.statusLine = strings.TrimSpace(value), n
	}
	if value, ok := strings.CutPrefix(line, "- Approved:"); ok {
		if h.approvedLine != 0 {
			return fmt.Errorf("line %d: a second - Approved: line (the first is on line %d): %s", n, h.approvedLine, digestSkipsReason)
		}
		h.approvedDigest, h.approvedLine = approvedDigestFrom(value), n
	}
	return nil
}

// approvedDigestFrom returns the 12 hex digits after `sha256:` in an
// `- Approved:` value, and "" when there are none.
func approvedDigestFrom(value string) string {
	_, after, ok := strings.Cut(value, "sha256:")
	if !ok {
		return ""
	}
	fields := strings.Fields(after)
	if len(fields) == 0 || !approvedDigestPattern.MatchString(fields[0]) {
		return ""
	}
	return fields[0]
}

// newSplitFeatureDraft starts the feature section headed `### <slug>` on
// line n. The slug becomes the default org_id, so it must be a valid
// identifier of at most maxFeatureOrgIDLen characters, and it must be new in
// the plan.
func newSplitFeatureDraft(slug string, n int, seen []*splitFeatureDraft) (*splitFeatureDraft, error) {
	if err := ValidateIdentifier("feature slug", slug); err != nil {
		return nil, fmt.Errorf("line %d: feature slug %q: it becomes the default org_id, so it must match %s", n, slug, identifierPattern)
	}
	if len(slug) > maxFeatureOrgIDLen {
		return nil, fmt.Errorf("line %d: feature slug %q is %d characters: it becomes the default org_id, and %s; use a shorter slug",
			n, slug, len(slug), featureOrgIDLimit())
	}
	if slug == splitDependsOnNone {
		return nil, fmt.Errorf("line %d: feature slug %q is reserved for - Depends on: %s", n, slug, splitDependsOnNone)
	}
	for _, f := range seen {
		if f.Slug == slug {
			return nil, fmt.Errorf("line %d: feature %s is listed twice (the first is on line %d)", n, slug, f.line)
		}
	}
	return &splitFeatureDraft{SplitFeature: SplitFeature{Slug: slug}, line: n, fieldLine: map[string]int{}}, nil
}

// splitFieldLine splits a feature field line `- <key>: <value>` (unindented)
// into its key and trimmed value.
func splitFieldLine(line string) (key, value string, ok bool) {
	for _, k := range splitFieldKeys {
		if v, found := strings.CutPrefix(line, "- "+k+":"); found {
			return k, strings.TrimSpace(v), true
		}
	}
	return "", "", false
}

// read takes one line of the feature's section: a field line sets the field
// (once), any other line is body.
func (f *splitFeatureDraft) read(line string, n int) error {
	key, value, ok := splitFieldLine(line)
	if !ok {
		f.bodyLines = append(f.bodyLines, line)
		return nil
	}
	if first, seen := f.fieldLine[key]; seen {
		return fmt.Errorf("line %d: feature %s: a second - %s: line (the first is on line %d)", n, f.Slug, key, first)
	}
	f.fieldLine[key] = n
	var err error
	switch key {
	case splitFieldType:
		if slices.Contains(splitFeatureTypes, value) {
			f.Type = value
		} else {
			err = fmt.Errorf("unknown type %q: use one of %s", value, strings.Join(splitFeatureTypes, ", "))
		}
	case splitFieldReserve:
		f.Reserve, err = parseSplitReserve(value)
	case splitFieldDependsOn:
		f.DependsOn, err = parseSplitDependsOn(value)
		f.dependsLine = n
	}
	if err != nil {
		return fmt.Errorf("line %d: feature %s: - %s: %w", n, f.Slug, key, err)
	}
	return nil
}

// parseSplitReserve reads a `- Reserve:` value: comma-separated paths under
// the reservation rules of NormalizeReservePaths.
func parseSplitReserve(value string) ([]string, error) {
	if value == "" {
		return nil, errors.New(`is empty: list the paths the feature reserves, relative to the repo root ("." for the whole repo)`)
	}
	raw := strings.Split(value, ",")
	for i := range raw {
		raw[i] = strings.TrimSpace(raw[i])
	}
	paths, err := NormalizeReservePaths(raw)
	if err != nil {
		return nil, errors.New(strings.TrimPrefix(err.Error(), "org: "))
	}
	return paths, nil
}

// parseSplitDependsOn reads a `- Depends on:` value: `none` or empty for no
// dependency, else comma-separated slugs, each once. Whether each slug names
// another feature of the plan is checked in finishSplitPlan.
func parseSplitDependsOn(value string) ([]string, error) {
	if value == "" || value == splitDependsOnNone {
		return nil, nil
	}
	var deps []string
	for _, d := range strings.Split(value, ",") {
		d = strings.TrimSpace(d)
		switch {
		case d == "":
			return nil, fmt.Errorf("has an empty entry: list feature slugs separated by commas, or write %s", splitDependsOnNone)
		case d == splitDependsOnNone:
			return nil, fmt.Errorf("%s cannot be listed with feature slugs", splitDependsOnNone)
		case slices.Contains(deps, d):
			return nil, fmt.Errorf("lists %s twice", d)
		}
		deps = append(deps, d)
	}
	return deps, nil
}

// finishSplitPlan checks what needs the whole file (the `## Features`
// section, its features, each feature's `- Reserve:` and the slugs its
// `- Depends on:` names) and builds the SplitPlan.
func finishSplitPlan(header splitHeader, featuresLine int, drafts []*splitFeatureDraft) (*SplitPlan, error) {
	if featuresLine == 0 {
		return nil, fmt.Errorf("no %s section: list the features under it as ### <slug> headings", splitFeaturesHeading)
	}
	if len(drafts) == 0 {
		return nil, fmt.Errorf("line %d: %s has no ### <slug> feature", featuresLine, splitFeaturesHeading)
	}
	p := &SplitPlan{Status: header.status, ApprovedDigest: header.approvedDigest}
	for _, f := range drafts {
		if f.Reserve == nil {
			return nil, fmt.Errorf("line %d: feature %s has no - %s: line: list the paths it reserves", f.line, f.Slug, splitFieldReserve)
		}
		for _, d := range f.DependsOn {
			if d == f.Slug {
				return nil, fmt.Errorf("line %d: feature %s: - %s: names the feature itself", f.dependsLine, f.Slug, splitFieldDependsOn)
			}
			if !slices.ContainsFunc(drafts, func(o *splitFeatureDraft) bool { return o.Slug == d }) {
				return nil, fmt.Errorf("line %d: feature %s: - %s: names %s, which is not a feature of this plan", f.dependsLine, f.Slug, splitFieldDependsOn, d)
			}
		}
		if f.Type == "" {
			f.Type = defaultSplitFeatureType
		}
		f.Body = trimBlankLines(f.bodyLines)
		p.Features = append(p.Features, f.SplitFeature)
	}
	return p, nil
}

// trimBlankLines joins lines with "\n" after dropping the blank lines at the
// start and the end.
func trimBlankLines(lines []string) string {
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}
