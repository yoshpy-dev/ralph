package org

import (
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
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
	// PlanPath is not wired into any embedded template today -- none of the
	// three templates (leader.md, implementer.md, reviewer.md)
	// reference {{PLAN_PATH}} (removed: no production caller populated it,
	// so every rendered prompt shipped a literal "- plan: " with nothing
	// after it -- self-review finding M5). The field is kept so a future
	// `--plan` flag can be wired through to a template substitution without
	// another RolePromptVars schema change.
	PlanPath string
	// Task is the task text substituted for {{TASK}} -- currently only
	// prompts/leader.md references it. `ralph org start`'s positional task
	// argument (internal/cli/org.go's newOrgStartCmd) flows through
	// SpawnParams.Task (spawn.go) into this field. Every other embedded role
	// template ignores it.
	Task string
	// Envelope is a one-line summary of the org's [org] envelope
	// (EnvelopeSummary, envelope_summary.go) substituted for {{ENVELOPE}} --
	// currently only prompts/leader.md references it.
	Envelope string
}

// RenderRolePrompt returns the rendered template for role with vars
// substituted, and true, if an embedded template exists for role. If no
// template exists for role, it returns ("", false, nil) -- the caller falls
// back to whatever --prompt was given verbatim, with no error: an unknown
// role is not a failure, just "no template available".
//
// DirectorRole is not a seat role: its template, prompts/director.md, is
// rendered only by RenderDirectorPrompt, so for it RenderRolePrompt returns
// ("", false, nil) as for a role with no template. Spawn refuses the role
// before it gets here (DirectorRoleInputErr, spawn.go).
//
// Placeholders that are not among the known {{...}} substitutions listed
// below are left in the rendered text unchanged (documented, not an error)
// -- the template author is expected to only use the known variable names.
func RenderRolePrompt(role string, vars RolePromptVars) (string, bool, error) {
	if role == DirectorRole {
		return "", false, nil
	}
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
		// PLAN_PATH has no consumer among the embedded templates today. The
		// replacer entry stays so a template that re-adds {{PLAN_PATH}} can
		// never ship the literal placeholder to a seat.
		"{{PLAN_PATH}}", vars.PlanPath,
		"{{TASK}}", vars.Task,
		"{{ENVELOPE}}", vars.Envelope,
	)
	return replacer.Replace(text), true, nil
}

// DirectorRole is the role name of the director: the key of its
// [org.roles] and [org.permissions.roles] entries and the name of its
// template, prompts/director.md. The director is not an org seat (plan
// docs/plans/active/2026-10-15-org-director-start.md, D2 and D7): `ralph org
// director start` launches it and RenderDirectorPrompt renders its template,
// and `ralph org spawn --role director` is refused.
const DirectorRole = "director"

// DirectorPromptVars holds the values substituted into prompts/director.md
// for the headless director `ralph org director start` launches: ID, its
// registration's d<N>; Driver, claude or codex; StateDir, the org state
// directory of the ledger, which every ralph org command it runs is given
// with --state-dir; ConfigPath, the absolute path of the ralph.toml start was
// given with --config, which every ralph org command it runs is given with
// --config too, so it reads the settings start read ("" when start had no
// --config); and Cwd, the directory it runs in.
type DirectorPromptVars struct {
	ID         string
	Driver     string
	StateDir   string
	ConfigPath string
	Cwd        string
}

// directorPromptNoConfigText substitutes for {{CONFIG}} when
// DirectorPromptVars.ConfigPath is empty: the file the director's ralph org
// commands read without --config. The director runs in the main worktree root
// when the ledger is the main worktree's, so both the 8a-1 rule
// (withMainWorktreeDirectorEnabled in internal/cli) and ./ralph.toml name
// that file; otherwise it runs where start ran, and ./ralph.toml is the one
// start read.
const directorPromptNoConfigText = "main worktree の ralph.toml(台帳が main worktree のものでなければ、cwd の ralph.toml)"

// RenderDirectorPrompt returns prompts/director.md with vars substituted:
// {{ID}}, {{DRIVER}}, {{CWD}}; {{STATE_DIR}}, which is vars.StateDir as one
// shell word (shellQuote) so the commands in the template run as written;
// {{ORG_FLAGS}}, the flags of every ralph org command in the template:
// --state-dir with {{STATE_DIR}}, followed by --config with vars.ConfigPath
// as one shell word when it is set; and {{CONFIG}}, the header's name of the
// config: vars.ConfigPath as one shell word, else
// directorPromptNoConfigText. Without ConfigPath the text holds no
// "--config". Unlike RenderRolePrompt it refuses to leave a placeholder: an
// id not of the form d<N>, a driver other than claude and codex, an empty
// StateDir or Cwd, a ConfigPath that is not absolute (the director runs in
// another directory, where a relative path names another file), and a
// rendered text that still holds "{{" (a placeholder it does not know, or a
// value that holds one) are errors.
func RenderDirectorPrompt(vars DirectorPromptVars) (string, error) {
	if _, ok := directorIDNumber(vars.ID); !ok {
		return "", fmt.Errorf("org: director prompt: %q is not a registration id (d<N>, such as d1)", vars.ID)
	}
	if vars.Driver != DirectorDriverClaude && vars.Driver != DirectorDriverCodex {
		return "", fmt.Errorf("org: director prompt: unknown driver %q: use %s or %s", vars.Driver, DirectorDriverClaude, DirectorDriverCodex)
	}
	if vars.StateDir == "" || vars.Cwd == "" {
		return "", fmt.Errorf("org: director prompt: the state dir (%q) and the cwd (%q) must both be set", vars.StateDir, vars.Cwd)
	}
	if vars.ConfigPath != "" && !filepath.IsAbs(vars.ConfigPath) {
		return "", fmt.Errorf("org: director prompt: the config path %q is not absolute (the director runs in another directory, where it would name another file)", vars.ConfigPath)
	}
	data, err := promptFS.ReadFile("prompts/" + DirectorRole + ".md")
	if err != nil {
		return "", fmt.Errorf("org: director prompt: read the embedded template: %w", err)
	}
	stateDir := shellQuote(vars.StateDir)
	orgFlags, config := "--state-dir "+stateDir, directorPromptNoConfigText
	if vars.ConfigPath != "" {
		config = shellQuote(vars.ConfigPath)
		orgFlags += " --config " + config
		config += "(start の --config)"
	}
	text := strings.NewReplacer(
		"{{ID}}", vars.ID,
		"{{DRIVER}}", vars.Driver,
		"{{STATE_DIR}}", stateDir,
		"{{ORG_FLAGS}}", orgFlags,
		"{{CONFIG}}", config,
		"{{CWD}}", vars.Cwd,
	).Replace(string(data))
	if i := strings.Index(text, "{{"); i >= 0 {
		rest, _, _ := strings.Cut(text[i:], "\n")
		return "", fmt.Errorf("org: director prompt: a placeholder is left after substitution: %q", rest)
	}
	return text, nil
}

// retiredRoleKind says how a retired role name left the vocabulary.
type retiredRoleKind string

// retiredRoleRenamed marks a name that still exists under a new name: the
// old spelling is rejected wherever it is *used* (spawn --role / --id, and
// a ralph.toml key that would otherwise be silently ignored), and the
// message points at the successor.
const retiredRoleRenamed retiredRoleKind = "renamed"

// retiredRoleRemoved marks a name that was removed outright: its template is
// gone and the successor is where its duty went, not a new spelling of the
// same role. Only `spawn --role <name>` without a --prompt is rejected (the
// seat would start with no purpose at all, and the message points at the
// successor and at --prompt). With a --prompt the name is an ordinary custom
// role, so it stays a valid seat id and a valid [org.roles] /
// [org.permissions.roles] key.
const retiredRoleRemoved retiredRoleKind = "removed"

// retiredRole is one entry of retiredRoles: what replaced the name and how.
type retiredRole struct {
	Successor string
	Kind      retiredRoleKind
}

// retiredRoles is the single place that lists role names the runtime no
// longer accepts, keyed by the old name (exact, case-sensitive match, like
// the rest of the role handling). Spawn's guard (RetiredRoleInputErr and
// retiredRoleConfigErr, spawn.go), the ralph.toml key scan
// (RetiredRoleConfigKeys), and `ralph doctor` all read this table.
//
// The guard's rejection messages and its seat-id rule are written for the
// two current entries, not derived from the table: a renamed name is refused
// as --id because the coordinator's old name was its agmsg identity (renamed
// to leader), and a removed name's message says its deterministic-gate
// re-run moved to the reviewer role (the removed seat's duty). Adding an
// entry therefore means reviewing those messages and that rule as well.
var retiredRoles = map[string]retiredRole{
	"lead": {Successor: LeaderIdentity, Kind: retiredRoleRenamed},
	"qa":   {Successor: "reviewer", Kind: retiredRoleRemoved},
}

// RetiredRoleConfigKey names one ralph.toml key spelled with a renamed role
// and the spelling it should have instead, e.g. Key
// "[org.permissions.roles].lead" and RenameTo
// "[org.permissions.roles].leader".
type RetiredRoleConfigKey struct {
	Key      string
	RenameTo string
}

// RetiredRoleConfigKeys lists the [org.roles] and [org.permissions.roles]
// keys in cfg that use a renamed role's old name, ordered by old name, then
// [org.roles] before [org.permissions.roles]. Only renamed roles count: a
// role that was removed outright may still be a legitimate custom role name
// for a seat that brings its own --prompt.
//
// Why config keys matter at all: config.Load does not validate role names,
// so a key under the old name is accepted and then never consulted -- a
// `lead = "guarded"` that no longer applies would silently run the leader
// with the full model_pool and with [org.permissions].default (autonomous
// unless the operator changed it) instead of the mode the key asked for.
// Spawn therefore rejects such a config rather than ignoring it; config.Load
// itself stays quiet so the read-only verbs and stop / disband can still
// clean up an old org.
func RetiredRoleConfigKeys(cfg config.OrgConfig) []RetiredRoleConfigKey {
	names := make([]string, 0, len(retiredRoles))
	for name, r := range retiredRoles {
		if r.Kind == retiredRoleRenamed {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var keys []RetiredRoleConfigKey
	for _, name := range names {
		successor := retiredRoles[name].Successor
		if _, ok := cfg.Roles[name]; ok {
			keys = append(keys, RetiredRoleConfigKey{
				Key:      "[org.roles]." + name,
				RenameTo: "[org.roles]." + successor,
			})
		}
		if _, ok := cfg.Permissions.Roles[name]; ok {
			keys = append(keys, RetiredRoleConfigKey{
				Key:      "[org.permissions.roles]." + name,
				RenameTo: "[org.permissions.roles]." + successor,
			})
		}
	}
	return keys
}
