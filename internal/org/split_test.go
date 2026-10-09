package org

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// splitPlanFixture is a valid split plan with two features. Its approval
// digest is a placeholder; approveSplitPlan writes the real one.
const splitPlanFixture = `# 認証の分割

- Status: Approved
- Approved: 2026-10-09 sha256:000000000000
- Owner: someone

## Overview

認証まわりを 2 つの機能に分ける。

## Features

Each feature below becomes one org.

### auth-core

- Type: fix
- Reserve: internal/auth/, ./docs//auth.md
- Depends on: none

Objective: add the token store.

- [ ] AC1: tokens persist

### auth-docs

- Reserve: docs/auth/
- Depends on: auth-core

Objective: document it.

## Notes

Other sections are ignored.
`

// approveSplitPlan returns content with the placeholder approval digest
// replaced by its real digest. The `- Approved:` line is one the digest
// skips, so the result has the same digest as content.
func approveSplitPlan(t *testing.T, content string) string {
	t.Helper()
	if !strings.Contains(content, "sha256:000000000000") {
		t.Fatal("approveSplitPlan: content has no placeholder digest")
	}
	return strings.Replace(content, "sha256:000000000000", "sha256:"+PlanDigest([]byte(content)), 1)
}

// minimalSplitPlan is a split plan whose `## Features` heading is line 5,
// so features starts on line 7.
func minimalSplitPlan(features string) string {
	return "# t\n\n- Status: Draft\n\n## Features\n\n" + features
}

// writeSplitPlan writes content to dir/name and returns the path.
func writeSplitPlan(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func loadSplitPlanContent(t *testing.T, content string) (*SplitPlan, string, error) {
	t.Helper()
	p := writeSplitPlan(t, t.TempDir(), "auth-split.md", content)
	plan, err := LoadSplitPlan(p)
	return plan, p, err
}

// TestLoadSplitPlan_ParsesFeatures covers plan AC1's accepted shape: the
// header's Status and approval digest, features in file order, `- Type:`
// defaulting to feat, `- Reserve:` normalized by the reservation rules,
// `- Depends on:` (none and a slug), the body without the field lines, and
// other `## ` sections and the prose before the first feature ignored.
func TestLoadSplitPlan_ParsesFeatures(t *testing.T) {
	plan, path, err := loadSplitPlanContent(t, splitPlanFixture)
	if err != nil {
		t.Fatalf("LoadSplitPlan: %v", err)
	}
	if plan.ID != "auth-split" || plan.Path != path {
		t.Errorf("ID, Path = %q, %q, want auth-split, %q", plan.ID, plan.Path, path)
	}
	if plan.Status != "Approved" || plan.ApprovedDigest != "000000000000" {
		t.Errorf("Status, ApprovedDigest = %q, %q, want Approved, 000000000000", plan.Status, plan.ApprovedDigest)
	}
	if want := PlanDigest([]byte(splitPlanFixture)); plan.Digest != want {
		t.Errorf("Digest = %q, want PlanDigest of the file %q", plan.Digest, want)
	}
	want := []SplitFeature{
		{
			Slug: "auth-core", Type: "fix", Reserve: []string{"docs/auth.md", "internal/auth/"},
			Body: "Objective: add the token store.\n\n- [ ] AC1: tokens persist",
		},
		{
			Slug: "auth-docs", Type: "feat", Reserve: []string{"docs/auth/"}, DependsOn: []string{"auth-core"},
			Body: "Objective: document it.",
		},
	}
	assertSplitFeatures(t, plan.Features, want)
}

func assertSplitFeatures(t *testing.T, got, want []SplitFeature) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d features %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Slug != w.Slug || g.Type != w.Type || g.Body != w.Body ||
			!slices.Equal(g.Reserve, w.Reserve) || !slices.Equal(g.DependsOn, w.DependsOn) ||
			(g.DependsOn == nil) != (w.DependsOn == nil) {
			t.Errorf("feature %d = %+v\nwant %+v", i, g, w)
		}
	}
}

// TestLoadSplitPlan_CRLF: a split plan with CRLF lines parses to the same
// features as its LF form; no "\r" reaches a value or a body.
func TestLoadSplitPlan_CRLF(t *testing.T) {
	lf, _, err := loadSplitPlanContent(t, splitPlanFixture)
	if err != nil {
		t.Fatalf("LoadSplitPlan (LF): %v", err)
	}
	crlf, _, err := loadSplitPlanContent(t, strings.ReplaceAll(splitPlanFixture, "\n", "\r\n"))
	if err != nil {
		t.Fatalf("LoadSplitPlan (CRLF): %v", err)
	}
	if crlf.Status != lf.Status || crlf.ApprovedDigest != lf.ApprovedDigest {
		t.Errorf("CRLF Status, ApprovedDigest = %q, %q, want %q, %q", crlf.Status, crlf.ApprovedDigest, lf.Status, lf.ApprovedDigest)
	}
	assertSplitFeatures(t, crlf.Features, lf.Features)
}

// TestLoadSplitPlan_Fields covers each field's accepted forms on its own.
func TestLoadSplitPlan_Fields(t *testing.T) {
	tests := []struct {
		name     string
		features string
		want     []SplitFeature
	}{
		{
			name:     "type defaults to feat",
			features: "### a\n- Reserve: a/\n",
			want:     []SplitFeature{{Slug: "a", Type: "feat", Reserve: []string{"a/"}}},
		},
		{
			name:     "every branch type",
			features: "### a\n- Type: security\n- Reserve: a/\n### b\n- Type: release\n- Reserve: b/\n",
			want: []SplitFeature{
				{Slug: "a", Type: "security", Reserve: []string{"a/"}},
				{Slug: "b", Type: "release", Reserve: []string{"b/"}},
			},
		},
		{
			name:     "reserve normalized, sorted and deduplicated",
			features: "### a\n- Reserve:  ./internal//auth/ ,docs/a.md, internal/auth/. , .\n",
			want:     []SplitFeature{{Slug: "a", Type: "feat", Reserve: []string{".", "docs/a.md", "internal/auth/"}}},
		},
		{
			name:     "fields anywhere in the section, after the body",
			features: "### a\n\nbody first\n\n- Depends on: b\n- Reserve: a/\n### b\n- Reserve: b/\n",
			want: []SplitFeature{
				{Slug: "a", Type: "feat", Reserve: []string{"a/"}, DependsOn: []string{"b"}, Body: "body first"},
				{Slug: "b", Type: "feat", Reserve: []string{"b/"}},
			},
		},
		{
			name:     "depends on empty is none",
			features: "### a\n- Reserve: a/\n- Depends on:\n",
			want:     []SplitFeature{{Slug: "a", Type: "feat", Reserve: []string{"a/"}}},
		},
		{
			name:     "depends on a list, in the order written",
			features: "### a\n- Reserve: a/\n- Depends on: c ,b\n### b\n- Reserve: b/\n### c\n- Reserve: c/\n",
			want: []SplitFeature{
				{Slug: "a", Type: "feat", Reserve: []string{"a/"}, DependsOn: []string{"c", "b"}},
				{Slug: "b", Type: "feat", Reserve: []string{"b/"}},
				{Slug: "c", Type: "feat", Reserve: []string{"c/"}},
			},
		},
		{
			name:     "heading with trailing spaces",
			features: "### a  \n- Reserve: a/\n",
			want:     []SplitFeature{{Slug: "a", Type: "feat", Reserve: []string{"a/"}}},
		},
		{
			name:     "sub-headings and indented field-like lines stay in the body",
			features: "### a\n- Reserve: a/\n\n#### Acceptance\n\n  - Reserve: not a field\n\n\n",
			want:     []SplitFeature{{Slug: "a", Type: "feat", Reserve: []string{"a/"}, Body: "#### Acceptance\n\n  - Reserve: not a field"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, _, err := loadSplitPlanContent(t, minimalSplitPlan(tt.features))
			if err != nil {
				t.Fatalf("LoadSplitPlan: %v", err)
			}
			assertSplitFeatures(t, plan.Features, tt.want)
		})
	}
}

// TestLoadSplitPlan_Rejects covers plan AC1's and AC2's rejections. Each
// error starts with the plan's path and names the line where it can.
func TestLoadSplitPlan_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		// The `## Features` section and its features.
		{"no Features section", "# t\n\n- Status: Draft\n\n## Other\n\n### a\n- Reserve: a/\n", "no ## Features section"},
		{"no heading at all", "# t\n- Status: Draft\n", "no ## Features section"},
		{"empty file", "", "no ## Features section"},
		{"no feature", minimalSplitPlan("Prose only.\n"), "line 5: ## Features has no ### <slug> feature"},
		{"feature under another section", "# t\n\n## Features\n\nprose\n\n## Other\n\n### a\n- Reserve: a/\n", "line 3: ## Features has no ### <slug> feature"},
		{"second Features heading", minimalSplitPlan("### a\n- Reserve: a/\n\n## Features\n"), "line 10: a second ## Features heading (the first is on line 5)"},
		// Slugs.
		{"uppercase slug", minimalSplitPlan("### Auth\n- Reserve: a/\n"), `line 7: feature slug "Auth"`},
		{"slug with underscore", minimalSplitPlan("### a_b\n- Reserve: a/\n"), `line 7: feature slug "a_b"`},
		{"slug starting with a digit", minimalSplitPlan("### 1a\n- Reserve: a/\n"), `line 7: feature slug "1a"`},
		{"slug of 31 characters", minimalSplitPlan("### a" + strings.Repeat("b", 30) + "\n- Reserve: a/\n"), "line 7: feature slug"},
		{"empty slug", minimalSplitPlan("### \n- Reserve: a/\n"), `line 7: feature slug ""`},
		{"slug none", minimalSplitPlan("### none\n- Reserve: a/\n"), `line 7: feature slug "none" is reserved`},
		{"duplicate slug", minimalSplitPlan("### a\n- Reserve: a/\n### a\n- Reserve: b/\n"), "line 9: feature a is listed twice (the first is on line 7)"},
		// Type.
		{"unknown type", minimalSplitPlan("### a\n- Type: feature\n- Reserve: a/\n"), `line 8: feature a: - Type: unknown type "feature"`},
		{"type in capitals", minimalSplitPlan("### a\n- Type: Feat\n- Reserve: a/\n"), `line 8: feature a: - Type: unknown type "Feat"`},
		{"empty type", minimalSplitPlan("### a\n- Type:\n- Reserve: a/\n"), `line 8: feature a: - Type: unknown type ""`},
		// Reserve.
		{"no reserve", minimalSplitPlan("### a\n- Type: fix\n"), "line 7: feature a has no - Reserve: line"},
		{"no reserve in the second feature", minimalSplitPlan("### a\n- Reserve: a/\n### b\nbody\n"), "line 9: feature b has no - Reserve: line"},
		{"empty reserve", minimalSplitPlan("### a\n- Reserve:\n"), "line 8: feature a: - Reserve: is empty"},
		{"reserve of spaces", minimalSplitPlan("### a\n- Reserve:   \n"), "line 8: feature a: - Reserve: is empty"},
		{"absolute reserve", minimalSplitPlan("### a\n- Reserve: /etc/\n"), `line 8: feature a: - Reserve: reserve path "/etc/" is absolute`},
		{"dot dot reserve", minimalSplitPlan("### a\n- Reserve: a/, ../b\n"), `reserve path "../b" has a .. segment`},
		{"trailing comma in reserve", minimalSplitPlan("### a\n- Reserve: a/,\n"), "line 8: feature a: - Reserve: reserve path is empty"},
		{"space inside a reserve path", minimalSplitPlan("### a\n- Reserve: a b/\n"), "has a comma, whitespace or a control character"},
		// Depends on.
		{"depends on an unknown slug", minimalSplitPlan("### a\n- Reserve: a/\n- Depends on: b\n"), "line 9: feature a: - Depends on: names b, which is not a feature of this plan"},
		{"depends on itself", minimalSplitPlan("### a\n- Reserve: a/\n- Depends on: a\n"), "line 9: feature a: - Depends on: names the feature itself"},
		{"depends on an empty entry", minimalSplitPlan("### a\n- Reserve: a/\n- Depends on: b,,c\n"), "line 9: feature a: - Depends on: has an empty entry"},
		{"depends on none and a slug", minimalSplitPlan("### a\n- Reserve: a/\n- Depends on: none, b\n"), "none cannot be listed with feature slugs"},
		{"depends on a slug twice", minimalSplitPlan("### a\n- Reserve: a/\n- Depends on: b, b\n### b\n- Reserve: b/\n"), "lists b twice"},
		// A field twice, or outside a feature.
		{"type twice", minimalSplitPlan("### a\n- Type: fix\n- Type: fix\n- Reserve: a/\n"), "line 9: feature a: a second - Type: line (the first is on line 8)"},
		{"reserve twice", minimalSplitPlan("### a\n- Reserve: a/\n- Reserve: a/\n"), "line 9: feature a: a second - Reserve: line (the first is on line 8)"},
		{"depends on twice", minimalSplitPlan("### a\n- Reserve: a/\n- Depends on: none\n- Depends on: none\n"), "line 10: feature a: a second - Depends on: line (the first is on line 9)"},
		{"field before the first feature", minimalSplitPlan("- Reserve: a/\n### a\n- Reserve: a/\n"), "line 7: a - Reserve: line before the first ### <slug> heading"},
		// The lines the approval digest skips (AC2).
		{"branch in the header", "# t\n- Branch: feat/x\n\n## Features\n\n### a\n- Reserve: a/\n", "line 2: a - Branch: line is not allowed"},
		{"branch in a feature", minimalSplitPlan("### a\n- Reserve: a/\n- Branch: feat/b\n"), "line 9: a - Branch: line is not allowed"},
		{"branch in another section", minimalSplitPlan("### a\n- Reserve: a/\n## Notes\n- Branch: x\n"), "line 10: a - Branch: line is not allowed"},
		{"status in a feature", minimalSplitPlan("### a\n- Reserve: a/\n- Status: Approved\n"), "line 9: a - Status: or - Approved: line after the header"},
		{"approved in another section", minimalSplitPlan("### a\n- Reserve: a/\n\n## Notes\n- Approved: x\n"), "line 11: a - Status: or - Approved: line after the header"},
		{"second status in the header", "# t\n- Status: Draft\n- Status: Approved\n\n## Features\n### a\n- Reserve: a/\n", "line 3: a second - Status: line (the first is on line 2)"},
		{"second approved in the header", "# t\n- Approved: x\n- Approved: y\n\n## Features\n### a\n- Reserve: a/\n", "line 3: a second - Approved: line (the first is on line 2)"},
		{"progress checklist at the end", minimalSplitPlan("### a\n- Reserve: a/\n\n## Progress checklist\n\n- [ ] done\n"), "line 10: a ## Progress checklist section is not allowed"},
		{"progress checklist before the features", "# t\n\n## Progress checklist\n\n## Features\n### a\n- Reserve: a/\n", "line 3: a ## Progress checklist section is not allowed"},
		{"progress checklist with CRLF", strings.ReplaceAll(minimalSplitPlan("### a\n- Reserve: a/\n## Progress checklist\n"), "\n", "\r\n"), "line 9: a ## Progress checklist section is not allowed"},
		{"checked box", minimalSplitPlan("### a\n- Reserve: a/\n- [x] done\n"), "line 9: a checked box"},
		{"indented capital checked box", minimalSplitPlan("### a\n- Reserve: a/\n  - [X] done\n"), "line 9: a checked box"},
		{"tab-indented checked box in the header", "# t\n\t- [x] note\n## Features\n### a\n- Reserve: a/\n", "line 2: a checked box"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, path, err := loadSplitPlanContent(t, tt.content)
			if err == nil {
				t.Fatalf("LoadSplitPlan accepted:\n%s", tt.content)
			}
			if prefix := "org: split plan " + path + ": "; !strings.HasPrefix(err.Error(), prefix) {
				t.Errorf("error %q does not start with %q", err, prefix)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

// TestLoadSplitPlan_AcceptsLinesTheDigestReads: the rejections of AC2 match
// the digest's own patterns exactly, so a line the digest reads (an indented
// `- Status:` or `- Branch:`, an unchecked box, a box written `-  [x]`) is
// body text, and changing it changes the digest.
func TestLoadSplitPlan_AcceptsLinesTheDigestReads(t *testing.T) {
	body := "  - Status: indented\n  - Branch: indented\n- [ ] open box\n-  [x] two spaces"
	content := minimalSplitPlan("### a\n- Reserve: a/\n" + body + "\n")
	plan, _, err := loadSplitPlanContent(t, content)
	if err != nil {
		t.Fatalf("LoadSplitPlan: %v", err)
	}
	if got := plan.Features[0].Body; got != body {
		t.Errorf("Body = %q, want %q", got, body)
	}
	for _, line := range strings.Split(body, "\n") {
		edited := strings.Replace(content, line, line+" edited", 1)
		if PlanDigest([]byte(edited)) == PlanDigest([]byte(content)) {
			t.Errorf("editing %q did not change the digest", line)
		}
	}
}

// TestLoadSplitPlan_RejectsEditsTheDigestSkips is plan AC2's and AC4's
// reason for the rejections: each edit below leaves the approval digest
// unchanged, so CheckApproved alone would pass it, and LoadSplitPlan refuses
// the edited plan instead.
func TestLoadSplitPlan_RejectsEditsTheDigestSkips(t *testing.T) {
	approved := approveSplitPlan(t, splitPlanFixture)
	plan, _, err := loadSplitPlanContent(t, approved)
	if err != nil {
		t.Fatalf("LoadSplitPlan (approved): %v", err)
	}
	if err := plan.CheckApproved(); err != nil {
		t.Fatalf("CheckApproved (approved): %v", err)
	}
	const anchor = "Objective: add the token store.\n"
	tests := []struct {
		name   string
		edited string
		want   string
	}{
		{"branch line added to a feature", strings.Replace(approved, anchor, anchor+"- Branch: feat/other\n", 1), "a - Branch: line"},
		{"branch line added to the header", strings.Replace(approved, "- Owner:", "- Branch: feat/x\n- Owner:", 1), "a - Branch: line"},
		{"status line added to a feature", strings.Replace(approved, anchor, anchor+"- Status: Approved\n", 1), "line after the header"},
		{"approved line added to a feature", strings.Replace(approved, anchor, anchor+"- Approved: 2026-10-10 sha256:111111111111\n", 1), "line after the header"},
		{"second status line in the header", strings.Replace(approved, "- Status: Approved\n", "- Status: Approved\n- Status: Approved\n", 1), "a second - Status: line"},
		{"second approved line in the header", strings.Replace(approved, "- Owner:", "- Approved: later\n- Owner:", 1), "a second - Approved: line"},
		{"box ticked", strings.Replace(approved, "- [ ] AC1", "- [x] AC1", 1), "a checked box"},
		{"box ticked with a capital", strings.Replace(approved, "- [ ] AC1", "- [X] AC1", 1), "a checked box"},
		{"progress checklist appended", approved + "## Progress checklist\n\n- Reserve: ./\n", "a ## Progress checklist section"},
		{"progress checklist inserted", strings.Replace(approved, "## Notes\n", "## Progress checklist\n\nAlso edit cmd/.\n\n## Notes\n", 1), "a ## Progress checklist section"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.edited == approved {
				t.Fatal("the edit did not change the plan")
			}
			if PlanDigest([]byte(tt.edited)) != PlanDigest([]byte(approved)) {
				t.Fatal("the edit changed the digest, so it does not show a line the digest skips")
			}
			_, _, err := loadSplitPlanContent(t, tt.edited)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadSplitPlan error = %v, want one containing %q", err, tt.want)
			}
		})
	}
}

// TestSplitPlanCheckApproved covers the approval check of plan AC4: the
// Status must be Approved and the `- Approved:` digest must equal the
// current one.
func TestSplitPlanCheckApproved(t *testing.T) {
	approved := approveSplitPlan(t, splitPlanFixture)
	tests := []struct {
		name    string
		content string
		want    []string // nil: approved
	}{
		{"approved", approved, nil},
		{"draft", strings.Replace(approved, "- Status: Approved", "- Status: Draft", 1), []string{`not approved: - Status: is "Draft", want Approved`}},
		{"status in lower case", strings.Replace(approved, "- Status: Approved", "- Status: approved", 1), []string{`- Status: is "approved"`}},
		{"no status line", strings.Replace(approved, "- Status: Approved\n", "", 1), []string{"not approved: it has no - Status: line"}},
		{"no approved line", strings.Replace(splitPlanFixture, "- Approved: 2026-10-09 sha256:000000000000\n", "", 1), []string{"no approval digest"}},
		{"approved line without a digest", strings.Replace(splitPlanFixture, "2026-10-09 sha256:000000000000", "TBD", 1), []string{"no approval digest"}},
		{"digest in capitals", strings.Replace(splitPlanFixture, "sha256:000000000000", "sha256:ABCDEF000000", 1), []string{"no approval digest"}},
		{"digest too short", strings.Replace(splitPlanFixture, "sha256:000000000000", "sha256:abc", 1), []string{"no approval digest"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, path, err := loadSplitPlanContent(t, tt.content)
			if err != nil {
				t.Fatalf("LoadSplitPlan: %v", err)
			}
			err = plan.CheckApproved()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("CheckApproved: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("CheckApproved accepted the plan")
			}
			for _, w := range append(tt.want, "org: split plan "+path+": ") {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
		})
	}

	t.Run("body changed after the approval", func(t *testing.T) {
		edited := strings.Replace(approved, "Objective: document it.", "Objective: document all of it.", 1)
		plan, path, err := loadSplitPlanContent(t, edited)
		if err != nil {
			t.Fatalf("LoadSplitPlan: %v", err)
		}
		err = plan.CheckApproved()
		if err == nil {
			t.Fatal("CheckApproved accepted a plan changed after its approval")
		}
		approvedDigest := PlanDigest([]byte(approved))
		for _, w := range []string{
			"org: split plan " + path + ": changed after its approval",
			"approved digest " + approvedDigest,
			"current digest " + PlanDigest([]byte(edited)),
			"scripts/plan-visual.sh digest " + path,
		} {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("error %q does not contain %q", err, w)
			}
		}
	})
}

func TestSplitPlanFeature(t *testing.T) {
	plan, _, err := loadSplitPlanContent(t, splitPlanFixture)
	if err != nil {
		t.Fatalf("LoadSplitPlan: %v", err)
	}
	f, ok := plan.Feature("auth-docs")
	if !ok || f.Slug != "auth-docs" || !slices.Equal(f.DependsOn, []string{"auth-core"}) {
		t.Errorf("Feature(auth-docs) = %+v, %v", f, ok)
	}
	if f, ok := plan.Feature("auth"); ok {
		t.Errorf("Feature(auth) = %+v, true; want false", f)
	}
}

// TestLoadSplitPlan_FileName: the id is the file name without .md, in the
// shape the reservation record can hold.
func TestLoadSplitPlan_FileName(t *testing.T) {
	dir := t.TempDir()
	content := minimalSplitPlan("### a\n- Reserve: a/\n")
	for _, name := range []string{"a.md", "0-split.md", "a" + strings.Repeat("b", 63) + ".md"} {
		plan, err := LoadSplitPlan(writeSplitPlan(t, dir, name, content))
		if err != nil {
			t.Errorf("LoadSplitPlan(%s): %v", name, err)
			continue
		}
		if want := strings.TrimSuffix(name, ".md"); plan.ID != want {
			t.Errorf("LoadSplitPlan(%s).ID = %q, want %q", name, plan.ID, want)
		}
	}
	for name, want := range map[string]string{
		"a.txt":                               "the file name must be <id>.md",
		"Auth.md":                             `id "Auth"`,
		"-a.md":                               `id "-a"`,
		"a_b.md":                              `id "a_b"`,
		"a" + strings.Repeat("b", 64) + ".md": "must match",
	} {
		if _, err := LoadSplitPlan(writeSplitPlan(t, dir, name, content)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("LoadSplitPlan(%s) error = %v, want one containing %q", name, err, want)
		}
	}
	missing := filepath.Join(dir, "missing.md")
	_, err := LoadSplitPlan(missing)
	if err == nil || !errors.Is(err, os.ErrNotExist) || !strings.HasPrefix(err.Error(), "org: split plan "+missing+": ") {
		t.Errorf("LoadSplitPlan(missing) error = %v, want a not-exist error naming %s", err, missing)
	}
}

// TestResolveSplitPlanPath: a split plan must be a regular file <id>.md
// directly in the state dir's splits/, compared with symlinks resolved on
// both sides; every refusal names that directory.
func TestResolveSplitPlanPath(t *testing.T) {
	state := t.TempDir()
	splits := SplitPlansDirIn(state)
	if err := os.MkdirAll(filepath.Join(splits, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(splits, "d.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := minimalSplitPlan("### a\n- Reserve: a/\n")
	plan := writeSplitPlan(t, splits, "auth-split.md", content)
	resolvedSplits, err := filepath.EvalSymlinks(splits)
	if err != nil {
		t.Fatal(err)
	}
	wantAbs := filepath.Join(resolvedSplits, "auth-split.md")
	elsewhere := t.TempDir()
	link := filepath.Join(t.TempDir(), "state-link")
	if err := os.Symlink(state, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(writeSplitPlan(t, elsewhere, "x.md", content), filepath.Join(splits, "outside.md")); err != nil {
		t.Fatal(err)
	}

	for name, tc := range map[string]struct{ stateDir, path string }{
		"plan in splits":                    {state, plan},
		"redundant segments":                {state, filepath.Join(splits, "sub", "..", ".", "auth-split.md")},
		"state dir through a symlink":       {link, plan},
		"plan path through a symlink":       {state, filepath.Join(link, "splits", "auth-split.md")},
		"both through a symlink":            {link, filepath.Join(link, "splits", "auth-split.md")},
		"state dir given with a trailing /": {state + "/", plan},
	} {
		abs, id, err := ResolveSplitPlanPath(tc.stateDir, tc.path)
		if err != nil || abs != wantAbs || id != "auth-split" {
			t.Errorf("%s: ResolveSplitPlanPath = %q, %q, %v; want %q, auth-split", name, abs, id, err, wantAbs)
		}
	}

	t.Run("relative path", func(t *testing.T) {
		t.Chdir(splits)
		abs, id, err := ResolveSplitPlanPath(state, "auth-split.md")
		if err != nil || abs != wantAbs || id != "auth-split" {
			t.Errorf("ResolveSplitPlanPath = %q, %q, %v; want %q, auth-split", abs, id, err, wantAbs)
		}
	})

	for name, tc := range map[string]struct{ path, want string }{
		"in the state dir":          {writeSplitPlan(t, state, "a.md", content), ": not directly in"},
		"in a subdirectory":         {writeSplitPlan(t, filepath.Join(splits, "sub"), "a.md", content), ": not directly in"},
		"in another directory":      {writeSplitPlan(t, elsewhere, "a.md", content), ": not directly in"},
		"symlink to a file outside": {filepath.Join(splits, "outside.md"), "resolved to"},
		"wrong extension":           {writeSplitPlan(t, splits, "a.txt", content), "the file name must be <id>.md"},
		"bad id":                    {writeSplitPlan(t, splits, "A.md", content), `id "A"`},
		"missing file":              {filepath.Join(splits, "missing.md"), "no such file or directory"},
		"directory":                 {filepath.Join(splits, "d.md"), ": not a regular file"},
		"splits itself":             {splits, ": not directly in"},
	} {
		_, _, err := ResolveSplitPlanPath(state, tc.path)
		if err == nil {
			t.Errorf("%s: ResolveSplitPlanPath(%s) accepted it", name, tc.path)
			continue
		}
		if !strings.HasPrefix(err.Error(), "org: split plan ") {
			t.Errorf("%s: error %q does not start with %q", name, err, "org: split plan ")
		}
		for _, w := range []string{tc.want, "a split plan is a file <id>.md directly in " + resolvedSplits} {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("%s: error %q does not contain %q", name, err, w)
			}
		}
	}

	t.Run("no splits directory", func(t *testing.T) {
		bare := t.TempDir()
		_, _, err := ResolveSplitPlanPath(bare, plan)
		if err == nil || !strings.Contains(err.Error(), ": not directly in "+SplitPlansDirIn(bare)) {
			t.Errorf("error = %v, want one naming %s", err, SplitPlansDirIn(bare))
		}
	})
}

// TestPlanDigest_KnownValues pins values that do not need the script: an
// empty file hashes nothing, a missing final newline counts as present, and a
// CRLF heading is not the progress checklist.
func TestPlanDigest_KnownValues(t *testing.T) {
	if got := PlanDigest(nil); got != "e3b0c44298fc" {
		t.Errorf("PlanDigest(empty) = %q, want e3b0c44298fc (SHA-256 of nothing)", got)
	}
	if got := PlanDigest([]byte("\n")); got != "01ba4719c80b" {
		t.Errorf(`PlanDigest("\n") = %q, want 01ba4719c80b (SHA-256 of "\n")`, got)
	}
	if PlanDigest([]byte("a")) != PlanDigest([]byte("a\n")) {
		t.Error("a missing final newline changed the digest")
	}
	if PlanDigest([]byte("## Progress checklist\r\nx\r\n")) == PlanDigest(nil) {
		t.Error("a CRLF ## Progress checklist heading was skipped like the LF one")
	}
	if PlanDigest([]byte("- Status: Draft\n- [x] a\n")) != PlanDigest([]byte("- [ ] a\n")) {
		t.Error("the digest read a - Status: line or a checked box as written")
	}
}

// TestPlanDigest_MatchesScript is plan AC2: PlanDigest returns what
// scripts/plan-visual.sh digest prints, on generated inputs for every rule of
// its awk and on the plan files of this repo.
func TestPlanDigest_MatchesScript(t *testing.T) {
	root := splitTestRepoRoot(t)
	script := filepath.Join(root, "scripts", "plan-visual.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("scripts/plan-visual.sh is needed to cross-check PlanDigest: %v", err)
	}
	dir := t.TempDir()
	generated := map[string]string{
		"header lines":                     "# t\n\n- Status: Approved\n- Approved: 2026-10-09 sha256:abcdefabcdef\n- Branch: feat/x\n- Owner: me\n\n## Objective\n\n本文\n",
		"progress checklist at the end":    "# t\n\n## A\n\na\n\n## Progress checklist\n\n- [x] Plan reviewed\n- [ ] PR created\n",
		"progress checklist in the middle": "## A\n\na\n\n## Progress checklist\n\n- [x] done\n### sub\nskipped\n\n## B\n\n- [X] b\n",
		"repeated progress checklist":      "## Progress checklist\na\n## Progress checklist\nb\n## Progress checklist \nc\n## Progress checklists\nd\n",
		"checked boxes":                    "- [x] a\n  - [X] b\n\t- [x] c\n\f- [x] d\n\v- [X] e\n- [ ] f\n- [xx] g\n-  [x] h\n- [x\n- [x]\n",
		"header-like lines not skipped":    "  - Status: kept\n - Branch: kept\n-Status: kept\n- status: kept\n- Approved kept\n",
		"no final newline":                 "# t\n\n- Status: Draft\n\n## A\n\n- [x] last",
		"crlf":                             "# t\r\n\r\n- Status: Approved\r\n- Branch: b\r\n\r\n## Progress checklist\r\n\r\n- [x] a\r\n\r\n## B\r\n  - [X] b\r\n",
		"crlf without final newline":       "a\r\n- [x] b\r",
		"empty":                            "",
		"only a newline":                   "\n",
		"blank lines":                      "\n\n\na\n\n",
		"split plan fixture":               splitPlanFixture,
	}
	files := map[string]string{}
	for name, content := range generated {
		files[name] = writeSplitPlan(t, dir, strings.ReplaceAll(name, " ", "-")+".md", content)
	}
	files["plan template"] = filepath.Join(root, "docs", "plans", "templates", "feature-plan.md")
	for _, glob := range []string{"active", "archive"} {
		plans, err := filepath.Glob(filepath.Join(root, "docs", "plans", glob, "*.md"))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range plans {
			files[glob+"/"+filepath.Base(p)] = p
		}
	}
	for name, file := range files {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command("bash", script, "digest", file).Output()
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					t.Fatalf("plan-visual.sh digest %s: %v\n%s", file, err, exitErr.Stderr)
				}
				t.Fatalf("plan-visual.sh digest %s: %v", file, err)
			}
			if want, got := strings.TrimSpace(string(out)), PlanDigest(data); got != want {
				t.Errorf("PlanDigest = %s, plan-visual.sh digest = %s", got, want)
			}
		})
	}
}

// splitTestRepoRoot resolves the repository root from this source file's
// location: internal/org/ -> ../../ (as sendDefaultsRepoRoot does).
func splitTestRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}
