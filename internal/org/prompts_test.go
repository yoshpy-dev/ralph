package org

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

func testRolePromptVars() RolePromptVars {
	return RolePromptVars{
		OrgID: "org-a", SeatID: "reviewer-1", Team: "ralph-org-a",
		Role: "reviewer", Scope: "internal/org/**", PlanPath: "docs/plans/active/2026-08-02-org-runtime-seats.md",
	}
}

// markdownSection returns the body of the level-2 markdown section whose
// header occupies a whole line (e.g. a line that is exactly "## 座席内
// fan-out"): the text after that header line up to, but not including, the
// next line that starts with "## ", or the end of text. found is false when
// header is absent. It lets tests assert on one section without a match
// elsewhere in the template masking a regression in that section.
func markdownSection(text, header string) (body string, found bool) {
	// Anchor header to a whole line ("\n<header>\n" in a text padded with a
	// leading newline) so a demoted "### <header>" line or a mid-line
	// mention of the header text cannot satisfy the lookup.
	padded := "\n" + text
	needle := "\n" + header + "\n"
	start := strings.Index(padded, needle)
	if start < 0 {
		return "", false
	}
	// Keep the header line's trailing newline at the head of body so that an
	// empty section (header immediately followed by the next "## " line)
	// yields "" rather than leaking the following section's text.
	body = padded[start+len(needle)-1:]
	if end := strings.Index(body, "\n## "); end >= 0 {
		body = body[:end]
	}
	return body, true
}

func TestMarkdownSection_AnchorsHeaderAndBoundsBody(t *testing.T) {
	// "see ## D" is a deliberate mid-line mention: the unanchored lookup this
	// helper replaced would have matched it.
	const doc = "intro\n## A\nbody a, see ## D for details\n### A sub\nsub text\n## B\n## C\nbody c"
	cases := []struct {
		name, header, wantBody string
		wantFound              bool
	}{
		{"normal section stops at the next level-2 header", "## A", "\nbody a, see ## D for details\n### A sub\nsub text", true},
		{"empty section returns an empty body, not the next section", "## B", "", true},
		{"section at end of text runs to EOF", "## C", "\nbody c", true},
		{"a level-3 header does not satisfy a level-2 lookup", "## A sub", "", false},
		{"a mid-line mention does not satisfy the lookup", "## D", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, found := markdownSection(doc, tc.header)
			if found != tc.wantFound || body != tc.wantBody {
				t.Errorf("markdownSection(doc, %q) = (%q, %v), want (%q, %v)", tc.header, body, found, tc.wantBody, tc.wantFound)
			}
		})
	}
}

// numberedItemStart matches the start of a numbered list item ("1. ", "12. ").
var numberedItemStart = regexp.MustCompile(`^\d+\. `)

// markdownItem returns the list item of section that contains marker: the text
// from the first line containing marker up to, but not including, the next
// line whose trimmed form starts with "- " or with a number and ". " (the next
// bullet or numbered item), or the end of section. found is false when marker
// is absent. It lets a test assert that one instruction carries its own
// wording, so a match in a neighbouring item cannot mask a regression.
func markdownItem(section, marker string) (item string, found bool) {
	lines := strings.Split(section, "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, marker) {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "- ") || numberedItemStart.MatchString(trimmed) {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n"), true
}

func TestMarkdownItem_BoundsItemAtNextBulletOrNumberedItem(t *testing.T) {
	const doc = "intro\n" +
		"1. first item\n" +
		"   continues here\n" +
		"2. second item mentions MARK\n" +
		"   - nested bullet\n" +
		"3. third item\n" +
		"- bullet with MARK2\n" +
		"  wrapped line\n" +
		"- last bullet\n" +
		"tail MARK3\n" +
		"more tail"
	cases := []struct {
		name, marker, wantItem string
		wantFound              bool
	}{
		{"numbered item stops before its nested bullet", "MARK", "2. second item mentions MARK", true},
		{"bullet keeps its wrapped continuation line", "MARK2", "- bullet with MARK2\n  wrapped line", true},
		{"a final item runs to the end of the text", "MARK3", "tail MARK3\nmore tail", true},
		{"an absent marker is not found", "NOPE", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item, found := markdownItem(doc, tc.marker)
			if found != tc.wantFound || item != tc.wantItem {
				t.Errorf("markdownItem(doc, %q) = (%q, %v), want (%q, %v)", tc.marker, item, found, tc.wantItem, tc.wantFound)
			}
		})
	}
}

// renderSeatPrompt renders the built-in template for role with the shared test
// vars and fails the test when no template is embedded.
func renderSeatPrompt(t *testing.T, role string) string {
	t.Helper()
	vars := testRolePromptVars()
	vars.Role = role
	vars.SeatID = role
	text, ok, err := RenderRolePrompt(role, vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt(%q): unexpected error: %v", role, err)
	}
	if !ok {
		t.Fatalf("expected ok=true for the built-in %s template", role)
	}
	return text
}

func TestRenderRolePrompt_Reviewer_AllKnownVarsSubstituted(t *testing.T) {
	text, ok, err := RenderRolePrompt("reviewer", testRolePromptVars())
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in reviewer template")
	}
	for _, want := range []string{"org-a", "reviewer-1", "ralph-org-a", "reviewer", "internal/org/**"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected rendered reviewer prompt to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "PlanPath") || strings.Contains(text, "{{PLAN_PATH}}") {
		t.Errorf("expected no {{PLAN_PATH}} placeholder or leftover text in the rendered reviewer prompt, got:\n%s", text)
	}
	if strings.Contains(text, "{{") {
		t.Errorf("expected no unsubstituted {{...}} placeholders for known vars, got:\n%s", text)
	}
	if !strings.Contains(text, ".claude/rules/ralph/agent-messaging.md") {
		t.Errorf("expected reviewer template to reference the protocol rule doc, got:\n%s", text)
	}
}

func TestRenderRolePrompt_Implementer_AllKnownVarsSubstituted(t *testing.T) {
	vars := testRolePromptVars()
	vars.Role = "implementer"
	vars.SeatID = "implementer-1"
	text, ok, err := RenderRolePrompt("implementer", vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in implementer template")
	}
	for _, want := range []string{"org-a", "implementer-1", "ralph-org-a", "implementer", "internal/org/**"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected rendered implementer prompt to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "{{") {
		t.Errorf("expected no unsubstituted {{...}} placeholders for known vars, got:\n%s", text)
	}
	if !strings.Contains(text, ".claude/rules/ralph/agent-messaging.md") {
		t.Errorf("expected implementer template to reference the protocol rule doc, got:\n%s", text)
	}
}

func TestRenderRolePrompt_QA_NoTemplate(t *testing.T) {
	// The qa seat template was retired: its deterministic-gate re-run moved
	// into the reviewer template, so "qa" is an ordinary role with no template.
	vars := testRolePromptVars()
	vars.Role = removedRoleName
	vars.SeatID = removedRoleName + "-1"
	text, ok, err := RenderRolePrompt(removedRoleName, vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: expected no error for the retired qa role, got %v", err)
	}
	if ok {
		t.Fatal("expected ok=false: the qa seat template no longer exists")
	}
	if text != "" {
		t.Fatalf("expected empty text for the retired qa role, got %q", text)
	}
}

func TestRenderRolePrompt_Leader_AllKnownVarsSubstituted(t *testing.T) {
	vars := testRolePromptVars()
	vars.Role = "leader"
	vars.SeatID = "leader"
	vars.Task = "dry-run 座席を1つ spawn し、typed message を送り、status を確認して disband せよ"
	// Derive the envelope from the shipped default pool (EnvelopeSummary is
	// what `ralph org start` renders) so this fixture never goes stale when
	// the default model_pool changes; defaults_sync_test.go locks that
	// default separately.
	vars.Envelope = EnvelopeSummary(config.Default().Org)
	text, ok, err := RenderRolePrompt("leader", vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in leader template")
	}
	for _, want := range []string{"org-a", "leader", "ralph-org-a", vars.Task, vars.Envelope} {
		if !strings.Contains(text, want) {
			t.Errorf("expected rendered leader prompt to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "{{") {
		t.Errorf("expected no unsubstituted {{...}} placeholders for known vars, got:\n%s", text)
	}
	if !strings.Contains(text, ".claude/rules/ralph/agent-messaging.md") {
		t.Errorf("expected leader template to reference the protocol rule doc, got:\n%s", text)
	}
	if !strings.Contains(text, "/org") {
		t.Errorf("expected leader template to reference the /org skill (its full operating manual), got:\n%s", text)
	}
	if !strings.Contains(text, "ralph org report") {
		t.Errorf("expected leader template to instruct the leader to run `ralph org report` before finishing, got:\n%s", text)
	}
}

func TestRenderRolePrompt_Leader_EmptyTaskAndEnvelope_NoLeftoverPlaceholders(t *testing.T) {
	vars := testRolePromptVars()
	vars.Role = "leader"
	vars.SeatID = "leader"
	vars.Task = ""
	vars.Envelope = ""
	text, ok, err := RenderRolePrompt("leader", vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in leader template")
	}
	if strings.Contains(text, "{{TASK}}") || strings.Contains(text, "{{ENVELOPE}}") {
		t.Errorf("expected no leftover {{TASK}}/{{ENVELOPE}} placeholders even when both vars are empty, got:\n%s", text)
	}
}

func TestRolePrompts_SeatTemplatesContainFanOutSection(t *testing.T) {
	for _, role := range []string{"implementer", "reviewer"} {
		t.Run(role, func(t *testing.T) {
			vars := testRolePromptVars()
			vars.Role = role
			text, ok, err := RenderRolePrompt(role, vars)
			if err != nil {
				t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
			}
			if !ok {
				t.Fatalf("expected ok=true for the built-in %s template", role)
			}
			section, found := markdownSection(text, "## 座席内 fan-out")
			if !found {
				t.Fatalf("expected %s template to contain a '## 座席内 fan-out' section, got:\n%s", role, text)
			}
			if !strings.Contains(section, "max_seats") {
				t.Errorf("expected %s template's fan-out section to mention max_seats, got section:\n%s", role, section)
			}
			if !strings.Contains(section, "leader") || !strings.Contains(section, "送ることは絶対に") {
				t.Errorf("expected %s template's fan-out section to prohibit sub-agents from sending to leader, got section:\n%s", role, section)
			}
		})
	}
}

func TestRenderRolePrompt_Leader_DelegatesToImplementer(t *testing.T) {
	vars := testRolePromptVars()
	vars.Role = "leader"
	vars.SeatID = "leader"
	text, ok, err := RenderRolePrompt("leader", vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in leader template")
	}
	if !strings.Contains(text, "implementer") {
		t.Errorf("expected leader template to delegate implementation to implementer seats, got:\n%s", text)
	}
	if strings.Contains(text, "budget") {
		t.Errorf("expected leader template to no longer reference budget, got:\n%s", text)
	}
}

func TestRenderRolePrompt_Reviewer_MissionRunsGateFirstAndBlocksWithoutReviewing(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## ミッション' section, got:\n%s", text)
	}

	// The default gate commands, used when the leader's TASK names none.
	for _, want := range []string{"./scripts/run-static-verify.sh", "./scripts/run-test.sh"} {
		if !strings.Contains(mission, want) {
			t.Errorf("expected the reviewer mission to name the gate script %q, got section:\n%s", want, mission)
		}
	}

	// A failing gate and an unrunnable gate each return BLOCKED and stop short
	// of the diff review. Each is checked inside its own list item so the
	// wording of one cannot be satisfied by the other.
	for _, marker := range []string{"GATE: fail", "GATE: unrunnable"} {
		item, ok := markdownItem(mission, marker)
		if !ok {
			t.Errorf("expected the reviewer mission to have an item for %q, got section:\n%s", marker, mission)
			continue
		}
		for _, want := range []string{"BLOCKED", "差分レビューに進まない"} {
			if !strings.Contains(item, want) {
				t.Errorf("expected the %q item of the reviewer mission to contain %q, got item:\n%s", marker, want, item)
			}
		}
	}

	// A passing gate leads to the diff review and is reported in the RESULT.
	if !strings.Contains(mission, "GATE: pass") {
		t.Errorf("expected the reviewer mission to report GATE: pass on a passing gate, got section:\n%s", mission)
	}

	// The retired qa seat is not a collaborator any more.
	for _, banned := range []string{"QA 座席", "qa 座席"} {
		if strings.Contains(text, banned) {
			t.Errorf("expected the reviewer template not to mention %q, got:\n%s", banned, text)
		}
	}
}

func TestRenderRolePrompt_Leader_MissionRoutesGateBlocked(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the leader template to contain a '## ミッション' section, got:\n%s", text)
	}

	// Review and verification (the gate re-run included) go to the reviewer.
	if !strings.Contains(mission, "reviewer 座席へ委譲") {
		t.Errorf("expected the leader mission to delegate review and verification to the reviewer seat, got section:\n%s", mission)
	}

	// GATE: fail goes back to the implementer; GATE: unrunnable does not.
	failItem, ok := markdownItem(mission, "GATE: fail")
	if !ok {
		t.Errorf("expected the leader mission to handle GATE: fail, got section:\n%s", mission)
	} else if !strings.Contains(failItem, "implementer 座席に差し戻") {
		t.Errorf("expected the GATE: fail item to send the work back to the implementer seat, got item:\n%s", failItem)
	}
	unrunnableItem, ok := markdownItem(mission, "GATE: unrunnable")
	if !ok {
		t.Errorf("expected the leader mission to handle GATE: unrunnable, got section:\n%s", mission)
	} else {
		for _, want := range []string{"implementer には戻さず", "人に上げる"} {
			if !strings.Contains(unrunnableItem, want) {
				t.Errorf("expected the GATE: unrunnable item to contain %q, got item:\n%s", want, unrunnableItem)
			}
		}
	}

	// The retired qa seat is not a delegation target any more.
	for _, banned := range []string{"qa 座席", "QA 座席"} {
		if strings.Contains(text, banned) {
			t.Errorf("expected the leader template not to mention %q, got:\n%s", banned, text)
		}
	}
}

func TestRenderRolePrompt_UnknownRole_NoTemplate(t *testing.T) {
	text, ok, err := RenderRolePrompt("unknown-role", testRolePromptVars())
	if err != nil {
		t.Fatalf("RenderRolePrompt: expected no error for an unknown role, got %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for a role with no embedded template")
	}
	if text != "" {
		t.Fatalf("expected empty text for an unknown role, got %q", text)
	}
}

func TestRenderRolePrompt_UnknownPlaceholder_PassesThroughUnchanged(t *testing.T) {
	// Documents the contract: an unknown {{PLACEHOLDER}} that never appears
	// in the actual templates would be left as-is (only the 5 known
	// variable names are substituted). We can't inject a fake template
	// through the embedded FS, so this test asserts the substitution
	// behavior directly against RenderRolePrompt's replacer semantics by
	// checking a known template has no accidental unknown-placeholder
	// leftovers, and by exercising the same strings.Replacer logic used
	// internally would require exporting it -- instead we assert on the
	// documented contract via a table of vars containing a value that
	// itself looks like a placeholder, verifying it is inserted verbatim
	// (not recursively substituted).
	vars := testRolePromptVars()
	vars.Scope = "{{NOT_A_KNOWN_VAR}}"
	text, ok, err := RenderRolePrompt("reviewer", vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in reviewer template")
	}
	if !strings.Contains(text, "{{NOT_A_KNOWN_VAR}}") {
		t.Errorf("expected a value that itself looks like a placeholder to be inserted verbatim, not recursively substituted, got:\n%s", text)
	}
}

func TestRenderRolePrompt_EmptyScope_SubstitutesDefaultText(t *testing.T) {
	// self-review finding M5: an empty --scope must not render as a bare
	// "scope: " with nothing after the colon -- RenderRolePrompt substitutes
	// an explicit "not specified" default instead.
	vars := testRolePromptVars()
	vars.Scope = ""
	text, ok, err := RenderRolePrompt("reviewer", vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in reviewer template")
	}
	if strings.Contains(text, "scope: \n") || strings.Contains(text, "scope:  ") {
		t.Errorf("expected no bare empty scope in the rendered prompt, got:\n%s", text)
	}
	if !strings.Contains(text, defaultScopeText) {
		t.Errorf("expected the rendered prompt to contain the default scope text %q, got:\n%s", defaultScopeText, text)
	}
}

// oldLeaderName is the coordinator's retired identifier. Tests refer to it
// through this constant instead of repeating the literal.
const oldLeaderName = "lead"

// removedRoleName is the seat template that was removed outright (its
// deterministic-gate re-run moved to the reviewer role). Tests refer to it
// through this constant instead of repeating the literal.
const removedRoleName = "qa"

func TestRetiredRoles_RemovedRoleNamesTheReviewerAsSuccessor(t *testing.T) {
	r, ok := retiredRoles[removedRoleName]
	if !ok {
		t.Fatalf("retiredRoles has no entry for %q", removedRoleName)
	}
	if r.Successor != "reviewer" {
		t.Errorf("Successor = %q, want %q", r.Successor, "reviewer")
	}
	if r.Kind != retiredRoleRemoved {
		t.Errorf("Kind = %q, want %q", r.Kind, retiredRoleRemoved)
	}
}

func TestRetiredRoles_OldLeaderNameIsRenamedToLeaderIdentity(t *testing.T) {
	r, ok := retiredRoles[oldLeaderName]
	if !ok {
		t.Fatalf("retiredRoles has no entry for %q", oldLeaderName)
	}
	if r.Successor != LeaderIdentity {
		t.Errorf("Successor = %q, want %q (LeaderIdentity)", r.Successor, LeaderIdentity)
	}
	if r.Kind != retiredRoleRenamed {
		t.Errorf("Kind = %q, want %q", r.Kind, retiredRoleRenamed)
	}
}

func TestRetiredRoles_SuccessorsAreLiveRoles(t *testing.T) {
	// A successor that is itself retired would send the operator in a circle.
	for name, r := range retiredRoles {
		if _, retired := retiredRoles[r.Successor]; retired {
			t.Errorf("retiredRoles[%q].Successor %q is itself retired", name, r.Successor)
		}
	}
}

func TestRetiredRoleConfigKeys(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.OrgConfig
		want []RetiredRoleConfigKey
	}{
		{
			name: "neither table has the old name",
			cfg: config.OrgConfig{
				Roles:       map[string][]string{"worker": {"sonnet"}},
				Permissions: config.OrgPermissionsConfig{Roles: map[string]string{"reviewer": "guarded"}},
			},
			want: nil,
		},
		{
			name: "nil maps",
			cfg:  config.OrgConfig{},
			want: nil,
		},
		{
			name: "org.roles only",
			cfg:  config.OrgConfig{Roles: map[string][]string{oldLeaderName: {"sonnet"}}},
			want: []RetiredRoleConfigKey{{Key: "[org.roles]." + oldLeaderName, RenameTo: "[org.roles]." + LeaderIdentity}},
		},
		{
			name: "an empty model list is still a present key",
			cfg:  config.OrgConfig{Roles: map[string][]string{oldLeaderName: {}}},
			want: []RetiredRoleConfigKey{{Key: "[org.roles]." + oldLeaderName, RenameTo: "[org.roles]." + LeaderIdentity}},
		},
		{
			name: "org.permissions.roles only",
			cfg:  config.OrgConfig{Permissions: config.OrgPermissionsConfig{Roles: map[string]string{oldLeaderName: "guarded"}}},
			want: []RetiredRoleConfigKey{{Key: "[org.permissions.roles]." + oldLeaderName, RenameTo: "[org.permissions.roles]." + LeaderIdentity}},
		},
		{
			name: "both tables, org.roles first",
			cfg: config.OrgConfig{
				Roles:       map[string][]string{oldLeaderName: {"sonnet"}},
				Permissions: config.OrgPermissionsConfig{Roles: map[string]string{oldLeaderName: "guarded"}},
			},
			want: []RetiredRoleConfigKey{
				{Key: "[org.roles]." + oldLeaderName, RenameTo: "[org.roles]." + LeaderIdentity},
				{Key: "[org.permissions.roles]." + oldLeaderName, RenameTo: "[org.permissions.roles]." + LeaderIdentity},
			},
		},
		{
			name: "matching is case-sensitive",
			cfg:  config.OrgConfig{Roles: map[string][]string{strings.ToUpper(oldLeaderName): {"sonnet"}}},
			want: nil,
		},
		{
			// A removed role's name is a legitimate custom-role key: a seat
			// that brings its own --prompt can still be configured under it.
			name: "removed role keys in both tables are not reported",
			cfg: config.OrgConfig{
				Roles:       map[string][]string{removedRoleName: {"sonnet"}},
				Permissions: config.OrgPermissionsConfig{Roles: map[string]string{removedRoleName: "guarded"}},
			},
			want: nil,
		},
		{
			name: "removed role keys do not hide renamed role keys",
			cfg: config.OrgConfig{
				Roles: map[string][]string{removedRoleName: {"sonnet"}, oldLeaderName: {"sonnet"}},
				Permissions: config.OrgPermissionsConfig{Roles: map[string]string{
					removedRoleName: "guarded", oldLeaderName: "guarded",
				}},
			},
			want: []RetiredRoleConfigKey{
				{Key: "[org.roles]." + oldLeaderName, RenameTo: "[org.roles]." + LeaderIdentity},
				{Key: "[org.permissions.roles]." + oldLeaderName, RenameTo: "[org.permissions.roles]." + LeaderIdentity},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RetiredRoleConfigKeys(tc.cfg)
			if len(got) != len(tc.want) {
				t.Fatalf("RetiredRoleConfigKeys = %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("RetiredRoleConfigKeys[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
