package org

import (
	"strings"
	"testing"
)

func testRolePromptVars() RolePromptVars {
	return RolePromptVars{
		OrgID: "org-a", SeatID: "reviewer-1", Team: "ralph-org-a",
		Role: "reviewer", Scope: "internal/org/**", PlanPath: "docs/plans/active/2026-08-02-org-runtime-seats.md",
	}
}

func TestRenderRolePrompt_Reviewer_AllKnownVarsSubstituted(t *testing.T) {
	text, ok, err := RenderRolePrompt("reviewer", testRolePromptVars())
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in reviewer template")
	}
	for _, want := range []string{"org-a", "reviewer-1", "ralph-org-a", "reviewer", "internal/org/**", "docs/plans/active/2026-08-02-org-runtime-seats.md"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected rendered reviewer prompt to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "{{") {
		t.Errorf("expected no unsubstituted {{...}} placeholders for known vars, got:\n%s", text)
	}
	if !strings.Contains(text, ".claude/rules/agent-messaging.md") {
		t.Errorf("expected reviewer template to reference the protocol rule doc, got:\n%s", text)
	}
}

func TestRenderRolePrompt_QA_AllKnownVarsSubstituted(t *testing.T) {
	vars := testRolePromptVars()
	vars.Role = "qa"
	vars.SeatID = "qa-1"
	text, ok, err := RenderRolePrompt("qa", vars)
	if err != nil {
		t.Fatalf("RenderRolePrompt: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for the built-in qa template")
	}
	for _, want := range []string{"org-a", "qa-1", "ralph-org-a", "qa", "internal/org/**"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected rendered qa prompt to contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "{{") {
		t.Errorf("expected no unsubstituted {{...}} placeholders for known vars, got:\n%s", text)
	}
	if !strings.Contains(text, ".claude/rules/agent-messaging.md") {
		t.Errorf("expected qa template to reference the protocol rule doc, got:\n%s", text)
	}
	if !strings.Contains(text, "run-static-verify.sh") || !strings.Contains(text, "run-test.sh") {
		t.Errorf("expected qa template to reference the deterministic gate scripts, got:\n%s", text)
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
	// in the actual templates would be left as-is (only the 6 known
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
