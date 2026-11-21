package org

import (
	"slices"
	"strings"
	"testing"
)

// sharedSplitPlan is a split plan whose header holds header after its
// `- Status:` line, so header starts on line 3, and whose `## Features`
// section holds features after a blank line. With a one-line header, the
// features start on line 7.
func sharedSplitPlan(header, features string) string {
	return "# t\n- Status: Draft\n" + header + "\n## Features\n\n" + features
}

// sharedTestFeature is a feature whose reservation overlaps none of the
// shared paths of the tests below.
const sharedTestFeature = "### a\n- Reserve: a/\n"

// TestLoadSplitPlan_Shared covers the accepted forms of the header's
// `- Shared:` line (AC1 of plan 2026-11-22-split-shared-paths): its paths
// normalized by the reservation rules, sorted and deduplicated, into
// SplitPlan.Shared, and nil without the line. A `- Shared:` line in a fenced
// code block (of the header, the preamble or a feature), an indented one, and
// a line of the preamble or a feature that only mentions shared paths in
// prose are text, not the field (TestLoadSplitPlan_SharedRejects refuses an
// unfenced `- Shared:` line after the header).
func TestLoadSplitPlan_Shared(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		want     []string
		preamble string // the plan's Preamble
		body     string // the first feature's Body
	}{
		{
			name:    "two paths",
			content: sharedSplitPlan("- Shared: pnpm-lock.yaml, tsconfig.json\n", sharedTestFeature),
			want:    []string{"pnpm-lock.yaml", "tsconfig.json"},
		},
		{
			name:    "one path",
			content: sharedSplitPlan("- Shared: go.sum\n", sharedTestFeature),
			want:    []string{"go.sum"},
		},
		{
			name:    "spaces around the commas and the value, sorted",
			content: sharedSplitPlan("- Shared:   tsconfig.json ,pnpm-lock.yaml   \n", sharedTestFeature),
			want:    []string{"pnpm-lock.yaml", "tsconfig.json"},
		},
		{
			name:    "normalized and deduplicated",
			content: sharedSplitPlan("- Shared: ./packages//shared/., go.sum, ./go.sum\n", sharedTestFeature),
			want:    []string{"go.sum", "packages/shared/"},
		},
		{
			name:    "a directory",
			content: sharedSplitPlan("- Shared: packages/shared/\n", sharedTestFeature),
			want:    []string{"packages/shared/"},
		},
		{
			name:    "a directory next to a reserved one with the same prefix",
			content: sharedSplitPlan("- Shared: internal/authz/\n", "### a\n- Reserve: internal/auth/\n"),
			want:    []string{"internal/authz/"},
		},
		{
			name:    "a file next to a reserved file",
			content: sharedSplitPlan("- Shared: go.sum\n", "### a\n- Reserve: go.mod\n"),
			want:    []string{"go.sum"},
		},
		{
			name:    "no Shared line",
			content: sharedSplitPlan("", sharedTestFeature),
		},
		{
			name:    "a Shared line in a fence in the header is text",
			content: sharedSplitPlan("```\n- Shared: go.sum\n```\n", sharedTestFeature),
		},
		{
			name:    "a Shared line in a fence, then one outside it, is no second line",
			content: sharedSplitPlan("~~~\n- Shared: x.json\n~~~\n- Shared: go.sum\n", sharedTestFeature),
			want:    []string{"go.sum"},
		},
		{
			name:    "an indented Shared line is free text",
			content: sharedSplitPlan("  - Shared: go.sum\n", sharedTestFeature),
		},
		{
			name:     "a Shared line in a fence in the preamble is text",
			content:  "# t\n\n## Rules\n```\n- Shared: go.sum\n```\n\n## Features\n\n" + sharedTestFeature,
			preamble: "## Rules\n```\n- Shared: go.sum\n```",
		},
		{
			name:    "a Shared line in a fence in a feature is body",
			content: minimalSplitPlan(sharedTestFeature + "~~~\n- Shared: go.sum\n~~~\n"),
			body:    "~~~\n- Shared: go.sum\n~~~",
		},
		{
			name:    "an indented Shared line in a feature is body",
			content: minimalSplitPlan(sharedTestFeature + "  - Shared: go.sum\n"),
			body:    "  - Shared: go.sum",
		},
		{
			name: "prose about shared paths in the preamble and a feature is text",
			content: "# t\n\n## Rules\nShared: go.sum is regenerated, never merged by hand.\n- Shared code lives in packages/shared/.\n" +
				"\n## Features\n\n" + sharedTestFeature + "The - Shared: line of the header lists go.sum.\n- Shared files: none.\n",
			preamble: "## Rules\nShared: go.sum is regenerated, never merged by hand.\n- Shared code lives in packages/shared/.",
			body:     "The - Shared: line of the header lists go.sum.\n- Shared files: none.",
		},
		{
			name:    "CRLF lines",
			content: strings.ReplaceAll(sharedSplitPlan("- Shared: go.sum, tsconfig.json\n", sharedTestFeature), "\n", "\r\n"),
			want:    []string{"go.sum", "tsconfig.json"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, _, err := loadSplitPlanContent(t, tt.content)
			if err != nil {
				t.Fatalf("LoadSplitPlan: %v", err)
			}
			if !slices.Equal(plan.Shared, tt.want) || (plan.Shared == nil) != (tt.want == nil) {
				t.Errorf("Shared = %#v, want %#v", plan.Shared, tt.want)
			}
			if plan.Preamble != tt.preamble {
				t.Errorf("Preamble = %q, want %q", plan.Preamble, tt.preamble)
			}
			if len(plan.Features) == 0 || plan.Features[0].Body != tt.body {
				t.Errorf("Features = %+v, want a first feature with Body %q", plan.Features, tt.body)
			}
		})
	}
}

// TestLoadSplitPlan_SharedRejects covers the refusals of the `- Shared:` line
// (AC1 of plan 2026-11-22-split-shared-paths): a second line, an empty value
// or entry, the whole repo, a path the reservation rules refuse, a shared
// path that overlaps a feature's `- Reserve:` path, which names the feature
// and both paths, and an unfenced `- Shared:` line after the header (in the
// preamble, in `## Features` before the first feature, in a feature, or in a
// section after `## Features`), which says to move it into the header. Each
// error starts with the plan's path and names the line.
func TestLoadSplitPlan_SharedRejects(t *testing.T) {
	const misplaced = " a - Shared: line after the header: the shared paths are every feature's and are read from the header only, " +
		"so here the line would share nothing; move it into the header, before the first ## heading, " +
		"or put it in a fenced code block to keep it as text"
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"a second Shared line", sharedSplitPlan("- Shared: go.sum\n- Shared: tsconfig.json\n", sharedTestFeature),
			"line 4: a second - Shared: line (the first is on line 3): list every shared path on one line"},
		{"a second Shared line after a fenced one", sharedSplitPlan("- Shared: go.sum\n```\n- Shared: x.json\n```\n- Shared: go.sum\n", sharedTestFeature),
			"line 7: a second - Shared: line (the first is on line 3)"},
		{"an empty value", sharedSplitPlan("- Shared:\n", sharedTestFeature), "line 3: - Shared: is empty"},
		{"a value of spaces", sharedSplitPlan("- Shared:   \n", sharedTestFeature), "line 3: - Shared: is empty"},
		{"an empty entry", sharedSplitPlan("- Shared: go.sum,,x.json\n", sharedTestFeature), "line 3: - Shared: has an empty entry"},
		{"a trailing comma", sharedSplitPlan("- Shared: go.sum,\n", sharedTestFeature), "line 3: - Shared: has an empty entry"},
		{"the whole repo", sharedSplitPlan("- Shared: .\n", sharedTestFeature), `line 3: - Shared: "." (the whole repo) cannot be shared`},
		{"the whole repo written ./", sharedSplitPlan("- Shared: go.sum, ./\n", sharedTestFeature), `line 3: - Shared: "." (the whole repo) cannot be shared`},
		{"an absolute path", sharedSplitPlan("- Shared: /etc/go.sum\n", sharedTestFeature),
			`line 3: - Shared: path "/etc/go.sum" is absolute: write it relative to the repo root`},
		{"a .. segment", sharedSplitPlan("- Shared: go.sum, ../x.json\n", sharedTestFeature), `line 3: - Shared: path "../x.json" has a .. segment`},
		{"whitespace inside a path", sharedSplitPlan("- Shared: a b.json\n", sharedTestFeature),
			`line 3: - Shared: path "a b.json" has a comma, whitespace or a control character`},
		{"a control character", sharedSplitPlan("- Shared: a\x01b.json\n", sharedTestFeature),
			`line 3: - Shared: path "a\x01b.json" has a comma, whitespace or a control character`},
		// A shared path that overlaps a feature's `- Reserve:` path.
		{"the same file as a feature's Reserve", sharedSplitPlan("- Shared: go.sum\n", "### a\n- Reserve: go.sum, a/\n"),
			"line 8: feature a: - Reserve: overlaps the - Shared: line on line 3 (go.sum overlaps go.sum): " +
				"a path is either reserved by one feature or shared by every feature, so remove it from one of the two lines"},
		{"a shared directory over a reserved file", sharedSplitPlan("- Shared: packages/\n", "### a\n- Reserve: packages/a.json\n"),
			"line 8: feature a: - Reserve: overlaps the - Shared: line on line 3 (packages/a.json overlaps packages/)"},
		{"a shared file under a reserved directory", sharedSplitPlan("- Shared: internal/auth/x.json\n", "### a\n- Reserve: internal/auth/\n"),
			"line 8: feature a: - Reserve: overlaps the - Shared: line on line 3 (internal/auth/ overlaps internal/auth/x.json)"},
		{"a shared directory inside a reserved directory", sharedSplitPlan("- Shared: internal/auth/x/\n", "### a\n- Reserve: internal/\n"),
			"line 8: feature a: - Reserve: overlaps the - Shared: line on line 3 (internal/ overlaps internal/auth/x/)"},
		{"a feature reserving the whole repo", sharedSplitPlan("- Shared: go.sum\n", "### a\n- Reserve: .\n"),
			"line 8: feature a: - Reserve: overlaps the - Shared: line on line 3 (. overlaps go.sum)"},
		{"the second feature overlaps", sharedSplitPlan("- Shared: go.sum\n", "### a\n- Reserve: a/\n### b\n- Type: fix\n- Reserve: b/, go.sum\n"),
			"line 11: feature b: - Reserve: overlaps the - Shared: line on line 3 (go.sum overlaps go.sum)"},
		{"every overlapping pair listed", sharedSplitPlan("- Shared: go.sum, a/x.json\n", "### a\n- Reserve: a/, go.sum\n"),
			"line 8: feature a: - Reserve: overlaps the - Shared: line on line 3 (a/ overlaps a/x.json, go.sum overlaps go.sum)"},
		// A `- Shared:` line after the header, outside a fence.
		{"a Shared line in the preamble", "# t\n\n## Rules\n- Shared: go.sum\n\n## Features\n\n" + sharedTestFeature,
			"line 4:" + misplaced},
		{"a Shared line in a feature", minimalSplitPlan(sharedTestFeature + "- Shared: go.sum\n"), "line 9:" + misplaced},
		{"a Shared line in a feature, with one in the header", sharedSplitPlan("- Shared: go.sum\n", sharedTestFeature+"- Shared: x.json\n"),
			"line 9:" + misplaced},
		{"an empty Shared line in a feature", minimalSplitPlan(sharedTestFeature + "- Shared:\n"), "line 9:" + misplaced},
		{"a Shared line in Features before the first feature", "# t\n\n## Features\n- Shared: go.sum\n" + sharedTestFeature,
			"line 4:" + misplaced},
		{"a Shared line in a section after Features", minimalSplitPlan(sharedTestFeature + "\n## Notes\n- Shared: go.sum\n"),
			"line 11:" + misplaced},
		{"a Shared line after a closed fence in a feature", minimalSplitPlan(sharedTestFeature + "```\nx\n```\n- Shared: go.sum\n"),
			"line 12:" + misplaced},
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

// TestFeatureDigest_WithoutSharedIsUnchanged is AC2 of plan
// 2026-11-22-split-shared-paths: the features of a plan with no `- Shared:`
// line keep the digests ralph gave them before that line existed (the values
// below, computed at commit 1a478b6b), so an approved plan and a stopped
// feature org start as before. A `- Shared:` line in a fenced code block of
// the header is text and changes no feature digest either.
func TestFeatureDigest_WithoutSharedIsUnchanged(t *testing.T) {
	fenced := strings.Replace(splitPlanFixture, "- Owner: someone\n", "- Owner: someone\n\n```\n- Shared: go.sum\n```\n", 1)
	if fenced == splitPlanFixture {
		t.Fatal("the fixture has no - Owner: line to add the fence after")
	}
	tests := []struct {
		name    string
		content string
		want    map[string]string // feature slug -> digest
	}{
		{"the split plan fixture", splitPlanFixture, map[string]string{"auth-core": "de0b5c3b804f", "auth-docs": "5a33b3d6707c"}},
		{"a fenced Shared line in the header", fenced, map[string]string{"auth-core": "de0b5c3b804f", "auth-docs": "5a33b3d6707c"}},
		{"no preamble and an empty body", minimalSplitPlan("### a\n- Reserve: x/\n"), map[string]string{"a": "1f1742b5c65c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, _, err := loadSplitPlanContent(t, tt.content)
			if err != nil {
				t.Fatalf("LoadSplitPlan: %v", err)
			}
			if plan.Shared != nil {
				t.Fatalf("Shared = %#v, want nil", plan.Shared)
			}
			for slug, want := range tt.want {
				if got := loadFeatureDigest(t, tt.content, slug); got != want {
					t.Errorf("feature %s: Digest = %s, want %s, the digest it had before the - Shared: line existed", slug, got, want)
				}
			}
		})
	}
}

// TestFeatureDigest_Shared: the shared paths go into the digest of every
// feature (AC2 of plan 2026-11-22-split-shared-paths), since they are paths
// its seats may change, so adding the `- Shared:` line, removing it, or
// changing its paths changes each feature's digest, and the same paths
// written another way leave it as it is. The line is a header line the
// approval digest reads, so editing it changes the plan's digest too.
func TestFeatureDigest_Shared(t *testing.T) {
	withShared := func(value string) string {
		return strings.Replace(splitPlanFixture, "- Owner: someone\n", "- Owner: someone\n- Shared: "+value+"\n", 1)
	}
	base := withShared("pnpm-lock.yaml, tsconfig.json")
	if base == splitPlanFixture {
		t.Fatal("the fixture has no - Owner: line to add the Shared line after")
	}
	plan, _, err := loadSplitPlanContent(t, base)
	if err != nil {
		t.Fatalf("LoadSplitPlan: %v", err)
	}
	if want := []string{"pnpm-lock.yaml", "tsconfig.json"}; !slices.Equal(plan.Shared, want) {
		t.Fatalf("Shared = %#v, want %#v", plan.Shared, want)
	}
	for _, f := range plan.Features {
		if want := featureDigest(plan.Preamble, plan.Shared, f); f.Digest != want {
			t.Errorf("feature %s: Digest = %s, want featureDigest of the preamble, the shared paths and the feature %s", f.Slug, f.Digest, want)
		}
	}

	tests := []struct {
		name          string
		before, after string
		changes       bool
	}{
		{"a Shared line added", splitPlanFixture, base, true},
		{"the Shared line removed", base, splitPlanFixture, true},
		{"a shared path added", base, withShared("pnpm-lock.yaml, tsconfig.json, go.sum"), true},
		{"a shared path changed", base, withShared("pnpm-lock.yaml, tsconfig.base.json"), true},
		{"a file shared as a directory", base, withShared("pnpm-lock.yaml, tsconfig.json/"), true},
		{"the same shared paths written another way", base, withShared(" ./tsconfig.json,pnpm-lock.yaml , pnpm-lock.yaml"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, slug := range []string{"auth-core", "auth-docs"} {
				was, now := loadFeatureDigest(t, tt.before, slug), loadFeatureDigest(t, tt.after, slug)
				if changed := was != now; changed != tt.changes {
					t.Errorf("feature %s: digest %s before the edit, %s after: changed = %v, want %v", slug, was, now, changed, tt.changes)
				}
			}
			if PlanDigest([]byte(tt.before)) == PlanDigest([]byte(tt.after)) {
				t.Errorf("the plan digest did not change with the edit of the - Shared: line")
			}
		})
	}
}
