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

// defaultScopeText substitutes for {{SCOPE}} when RolePromptVars.Scope is
// empty (SpawnParams.Scope is an optional --scope flag), so a rendered
// prompt never reads "scope:  の範囲外は..." with nothing after the colon --
// an instruction referencing an empty scope is worse than an explicit
// "not specified" (self-review finding M5).
const defaultScopeText = "未指定(読み取り中心で、リポジトリ規約に従うこと)"

// RolePromptVars holds the values substituted into a role prompt template.
// Substitution is plain strings.ReplaceAll on "{{NAME}}" placeholders (no
// templating engine, no control flow) -- kept deliberately small and boring
// per .claude/rules/ralph/architecture.md.
type RolePromptVars struct {
	OrgID  string
	SeatID string
	Team   string
	Role   string
	Scope  string
	// PlanPath is not wired into any embedded template today -- reviewer.md
	// and qa.md do not reference {{PLAN_PATH}} (removed: no production
	// caller populated it, so every rendered prompt shipped a literal
	// "- plan: " with nothing after it -- self-review finding M5). The field
	// is kept so PR③ (Lead 自律編成) can wire a `--plan` flag through to a
	// template substitution without another RolePromptVars schema change.
	PlanPath string
	// Task is the task text substituted for {{TASK}} -- currently only
	// prompts/lead.md references it. `ralph org start`'s positional task
	// argument (internal/cli/org.go's newOrgStartCmd) flows through
	// SpawnParams.Task (spawn.go) into this field. Every other embedded role
	// template ignores it.
	Task string
	// Envelope is a one-line summary of the org's [org] envelope
	// (EnvelopeSummary, envelope_summary.go) substituted for {{ENVELOPE}} --
	// currently only prompts/lead.md references it.
	Envelope string
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
	scope := vars.Scope
	if scope == "" {
		scope = defaultScopeText
	}
	text := string(data)
	replacer := strings.NewReplacer(
		"{{ORG_ID}}", vars.OrgID,
		"{{SEAT_ID}}", vars.SeatID,
		"{{TEAM}}", vars.Team,
		"{{ROLE}}", vars.Role,
		"{{SCOPE}}", scope,
		// PLAN_PATH is reserved for PR③ (no template uses it today). The
		// replacer entry stays so a template that re-adds {{PLAN_PATH}} can
		// never ship the literal placeholder to a seat.
		"{{PLAN_PATH}}", vars.PlanPath,
		"{{TASK}}", vars.Task,
		"{{ENVELOPE}}", vars.Envelope,
	)
	return replacer.Replace(text), true, nil
}
