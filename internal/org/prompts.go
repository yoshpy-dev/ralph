package org

import (
	"embed"
	"strings"
)

// promptFS embeds every role prompt template under internal/org/prompts/.
// Embedding (rather than reading from disk at runtime) keeps the binary and
// the prompt templates version-locked -- a `ralph` binary always ships with
// the exact prompt text it was built with, per the plan's "go:embed only"
// design decision (see docs/plans/active/2026-08-02-org-runtime-seats.md,
// "役割プロンプトの配置").
//
//go:embed prompts/*.md
var promptFS embed.FS

// RolePromptVars holds the values substituted into a role prompt template.
// Substitution is plain strings.ReplaceAll on "{{NAME}}" placeholders (no
// templating engine, no control flow) -- kept deliberately small and boring
// per .claude/rules/architecture.md.
type RolePromptVars struct {
	OrgID    string
	SeatID   string
	Team     string
	Role     string
	Scope    string
	PlanPath string
}

// RenderRolePrompt returns the rendered template for role with vars
// substituted, and true, if an embedded template exists for role. If no
// template exists for role, it returns ("", false, nil) -- the caller falls
// back to whatever --prompt was given verbatim, with no error: an unknown
// role is not a failure, just "no template available".
//
// Placeholders that are not among the known {{...}} substitutions listed
// below are left in the rendered text unchanged (documented, not an error)
// -- the template author is expected to only use the known variable names.
func RenderRolePrompt(role string, vars RolePromptVars) (string, bool, error) {
	data, err := promptFS.ReadFile("prompts/" + role + ".md")
	if err != nil {
		return "", false, nil
	}
	text := string(data)
	replacer := strings.NewReplacer(
		"{{ORG_ID}}", vars.OrgID,
		"{{SEAT_ID}}", vars.SeatID,
		"{{TEAM}}", vars.Team,
		"{{ROLE}}", vars.Role,
		"{{SCOPE}}", vars.Scope,
		"{{PLAN_PATH}}", vars.PlanPath,
	)
	return replacer.Replace(text), true, nil
}
