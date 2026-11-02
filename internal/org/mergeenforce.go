package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The protection check of `ralph org merge`, CheckMergeEnforcement (D11 of
// plan docs/plans/active/2026-10-17-org-auto-merge.md): whether GitHub keeps
// the merge of a PR into origin's default branch up to date for the caller,
// with no merge queue in the way, read live. It runs on mergeGuaranteeRun and
// shares its readers of the rules and the branch protection (readMergeRules,
// readMergeClassic and readMergeClassicStrict) with CheckMergeGuarantee in
// mergeguarantee.go, whose kinds, Details and cache it leaves as they are.

// mergeEnforceFix is what the not-ok Details of CheckMergeEnforcement about a
// setting tell the reader to set: the only protection it reads that a caller
// without admin access can read and that says whether that caller can bypass
// it (current_user_can_bypass), an up-to-date rule of a ruleset.
const mergeEnforceFix = "add a ruleset to the branch with Require branches to be up to date before merging and at least one " +
	"required status check, with no bypass for whoever runs ralph org merge"

// mergeQueueQuery is the GraphQL query CheckMergeEnforcement reads the
// default branch's merge queue with; mergeQueue is null when it has none.
const mergeQueueQuery = "query($owner: String!, $name: String!, $branch: String!) " +
	"{ repository(owner: $owner, name: $name) { mergeQueue(branch: $branch) { id } } }"

// MergeEnforcementParams describes one CheckMergeEnforcement call: the
// repository `ralph org merge` read from origin's URL and the default branch
// origin's HEAD names.
type MergeEnforcementParams struct {
	// Dir is the repository directory gh runs in; empty means the current
	// directory.
	Dir string
	// Exec runs gh; nil means realMergeOrderExec.
	Exec MergeOrderExec
	// Timeout bounds the whole check; zero or less means 20 s.
	Timeout time.Duration
	// Host is origin's host, the --hostname of every gh call (ssh.github.com
	// is read as github.com); Owner and Name are origin's repository, and
	// DefaultBranch the branch whose protection is read.
	Host, Owner, Name, DefaultBranch string
	// HintStateDir is MergeParams.HintStateDir: the --state-dir the ralph org
	// merge-order command of a Detail repeats (VerbCommand).
	HintStateDir string
}

// MergeEnforcement is what CheckMergeEnforcement found.
type MergeEnforcement struct {
	// OK reports that GitHub makes a PR's required status checks pass on a
	// head with the latest default branch before it merges, for the caller
	// too, and that the branch has no merge queue.
	OK bool
	// Detail says in one sentence which setting makes it so, or why it is not
	// ok and what to do.
	Detail string
}

// CheckMergeEnforcement decides whether `ralph org merge` may merge a PR into
// p.DefaultBranch of the repository p names (D11 of plan
// docs/plans/active/2026-10-17-org-auto-merge.md): whether GitHub makes the
// PR's required status checks pass on a head with the latest default branch,
// for the caller too, with no merge queue in the way. It reads live, never the
// cache CheckMergeGuarantee's result is saved in, shares CheckMergeGuarantee's
// readers of the rules and the branch protection, and leaves
// CheckMergeGuarantee, its kinds and its Details as they are. Every step runs
// under one deadline, p.Timeout, with --hostname and GH_PROMPT_DISABLED=1:
//
//  1. `gh api repos/<o>/<r>`: its default_branch must be p.DefaultBranch
//     (readEnforcementRepository).
//  2. `gh api graphql` with repository(owner, name).mergeQueue(branch), which
//     also sees a queue the branch protection sets: a queue, an errors list, a
//     gh that fails, or an answer that is not that JSON is not ok
//     (readMergeQueue).
//  3. `gh api --paginate repos/<o>/<r>/rules/branches/<branch>`
//     (readMergeRules): a merge_queue rule is not ok.
//  4. When up-to-date rules (strict_required_status_checks_policy) require
//     checks of their own: `gh api repos/<o>/<r>/rulesets/<id>` for each of
//     their rulesets; ok when the current_user_can_bypass of one is never, not
//     ok when none is or when a read fails (enforceRulesets).
//  5. Otherwise the branch protection: its required checks (readMergeClassic)
//     and, when it requires some, its strict (readMergeClassicStrict), which
//     needs admin access. Strict with checks is ok only when
//     `gh api repos/<o>/<r>/branches/<branch>/protection/enforce_admins` is
//     enabled, whether the caller is an admin or not (enforceClassic).
//  6. Anything else is not ok: nothing makes the PR up to date with its
//     required checks, ralph doctor's pr_checks_only (notEnforced).
//
// These fields were read live on 2026-10-17 (the plan's Assumptions): the
// ruleset_id of a rules/branches element, a ruleset's current_user_can_bypass
// (never), and repository.mergeQueue (null with no queue).
// Not seen live: the other values of current_user_can_bypass (always,
// pull_requests_only, exempt) and a mergeQueue that is set. So every value but
// never and null is not ok, as is every read that fails, ctx done included;
// the Detail says why.
func CheckMergeEnforcement(ctx context.Context, p MergeEnforcementParams) MergeEnforcement {
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultMergeGuaranteeTimeout
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	repo := mergeRepo{owner: p.Owner, name: p.Name}
	c := MergeGuaranteeCheck{Host: strings.ToLower(p.Host), Repo: repo.String(), DefaultBranch: p.DefaultBranch}
	g := &mergeGuaranteeRun{parent: ctx, ctx: checkCtx, timeout: timeout, dir: p.Dir, run: p.Exec, c: &c,
		apiHost: mergeGuaranteeAPIHost(c.Host), repo: repo}
	if g.run == nil {
		g.run = realMergeOrderExec
	}
	ok, detail := g.enforce(p.HintStateDir)
	return MergeEnforcement{OK: ok, Detail: detail}
}

// enforce runs the steps of CheckMergeEnforcement and returns whether the
// merge may go on, and its Detail. hintStateDir is
// MergeEnforcementParams.HintStateDir.
func (g *mergeGuaranteeRun) enforce(hintStateDir string) (ok bool, detail string) {
	if why := g.readEnforcementRepository(hintStateDir); why != "" {
		return false, why
	}
	if why := g.readMergeQueue(); why != "" {
		return false, why
	}
	rules, why := g.readMergeRules()
	switch {
	case why != "":
		return false, g.enforceUnread(why)
	case rules.mergeQueue:
		return false, fmt.Sprintf("%s, has a merge queue (a ruleset's merge_queue rule): ralph org merge does not merge into "+
			"a branch with a merge queue, so merge the PR through the queue yourself", g.subject())
	case len(rules.strictRulesets) > 0:
		return g.enforceRulesets(rules.strictRulesets)
	}
	classic, why := g.readMergeClassic()
	if why != "" {
		return false, g.enforceUnread(why)
	}
	if classic > 0 {
		strict, why := g.readMergeClassicStrict(classic)
		switch {
		case why != "" && g.ctx.Err() != nil:
			return false, why
		case why != "":
			return false, why + ": " + mergeEnforceFix + " instead, which ralph org merge reads without admin access"
		case strict:
			return g.enforceClassic(classic)
		}
	}
	return false, g.notEnforced(rules, classic)
}

// enforceUnread is the Detail of a read CheckMergeEnforcement failed on: why,
// and the next step, which stopped (ctx done) has given already.
func (g *mergeGuaranteeRun) enforceUnread(why string) string {
	if g.ctx.Err() != nil {
		return why
	}
	return fmt.Sprintf("%s: check that gh can reach %s and is logged in to it (gh auth status --hostname %s)", why, g.apiHost, g.apiHost)
}

// readEnforcementRepository is step 1 of CheckMergeEnforcement: it sets
// c.APIRepo, or returns why the check cannot go on, which names ralph org
// merge-order with hintStateDir (VerbCommand) when the default branches
// differ. permissions.admin is not read: whether the caller is an admin
// decides nothing (enforceClassic).
func (g *mergeGuaranteeRun) readEnforcementRepository(hintStateDir string) (why string) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
		FullName      string `json:"full_name"`
	}
	path := g.repoPath()
	if why := g.ghJSON(path, "a repository's JSON", &repo); why != "" {
		return g.enforceUnread(why)
	}
	g.c.APIRepo = repo.FullName
	if repo.DefaultBranch != g.c.DefaultBranch {
		return fmt.Sprintf("%s returned the default branch %q, but origin's HEAD names %s, so which branch's protection "+
			"to read is unknown: run %s again", g.command(false, path), repo.DefaultBranch, g.c.DefaultBranch,
			VerbCommand("merge-order", hintStateDir))
	}
	return ""
}

// mergeQueueArgs are the arguments of the gh api graphql call of
// readMergeQueue. The variables go with -f, which gh passes as strings: -F
// would read a value such as 123 or true as a number or a boolean, and one
// that starts with @ as a file. The endpoint comes last, as in ghArgs.
func (g *mergeGuaranteeRun) mergeQueueArgs() []string {
	return []string{"api", "--hostname", g.apiHost, "-f", "query=" + mergeQueueQuery, "-f", "owner=" + g.repo.owner,
		"-f", "name=" + g.repo.name, "-f", "branch=" + g.c.DefaultBranch, "graphql"}
}

// readMergeQueue is step 2 of CheckMergeEnforcement: "" when the GraphQL
// answer says the default branch has no merge queue (mergeQueue null), else
// why the check cannot go on.
func (g *mergeGuaranteeRun) readMergeQueue() (why string) {
	out, err := g.run(g.ctx, g.dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", g.mergeQueueArgs()...)
	cmd := fmt.Sprintf("gh api --hostname %s graphql (mergeQueue of %s)", g.apiHost, g.c.DefaultBranch)
	if why := g.stopped(cmd); why != "" {
		return why
	}
	unknown := fmt.Sprintf("so whether %s, has a merge queue is unknown", g.subject())
	if err != nil {
		return g.enforceUnread(fmt.Sprintf("%s failed (%s), %s", cmd, mergeGuaranteeErrText(err), unknown))
	}
	var body struct {
		Data *struct {
			Repository *struct {
				MergeQueue json.RawMessage `json:"mergeQueue"`
			} `json:"repository"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		return g.enforceUnread(fmt.Sprintf("%s printed output that is not GraphQL JSON (%s), %s", cmd, mergeGuaranteeErrText(err), unknown))
	}
	if errs := bytes.TrimSpace(body.Errors); len(errs) > 0 && string(errs) != "null" {
		return g.enforceUnread(fmt.Sprintf("%s answered with errors (%s), %s", cmd, mergeGuaranteeErrText(errors.New(string(errs))), unknown))
	}
	if body.Data == nil || body.Data.Repository == nil || body.Data.Repository.MergeQueue == nil {
		return g.enforceUnread(fmt.Sprintf("%s returned no repository.mergeQueue, %s", cmd, unknown))
	}
	if string(bytes.TrimSpace(body.Data.Repository.MergeQueue)) != "null" {
		return fmt.Sprintf("%s, has a merge queue (GraphQL mergeQueue is set): ralph org merge does not merge into a branch "+
			"with a merge queue, so merge the PR through the queue yourself", g.subject())
	}
	return ""
}

// enforceRulesets is step 4 of CheckMergeEnforcement, for the rulesets of the
// up-to-date rules that require checks of their own: it reads each, and is ok
// when one's current_user_can_bypass is never and every read passed. A rule
// with no ruleset_id, a read that fails, and any other value (always,
// pull_requests_only, exempt, one not known, a missing field) are not ok.
func (g *mergeGuaranteeRun) enforceRulesets(strict []mergeStrictRuleset) (ok bool, detail string) {
	var never string
	var bypassed []string
	for _, s := range strict {
		if s.id <= 0 {
			return false, fmt.Sprintf("an up-to-date rule of %s, with %s of its own names no ruleset_id, so whether you can "+
				"bypass it is unknown: %s", g.subject(), mergeChecksText(s.checks), mergeEnforceFix)
		}
		var rs struct {
			Name   string  `json:"name"`
			Bypass *string `json:"current_user_can_bypass"`
		}
		if why := g.ghJSON(g.repoPath("rulesets", strconv.Itoa(s.id)), "a ruleset's JSON", &rs); why != "" {
			return false, g.enforceUnread(why)
		}
		bypass := "missing"
		if rs.Bypass != nil {
			bypass = *rs.Bypass
		}
		switch {
		case bypass != "never":
			bypassed = append(bypassed, fmt.Sprintf("%q (id %d): %q", rs.Name, s.id, bypass))
		case never == "":
			never = fmt.Sprintf("the ruleset %q (id %d) of %s, requires branches to be up to date before merging with its %s, "+
				"and you cannot bypass it (current_user_can_bypass: never)", rs.Name, s.id, g.subject(), mergeChecksText(s.checks))
		}
	}
	if never != "" {
		return true, never
	}
	return false, fmt.Sprintf("every ruleset of %s, that requires branches to be up to date before merging with required status "+
		"checks of its own can be bypassed by you (current_user_can_bypass of %s), so GitHub would let you merge a head that is "+
		"not up to date: take whoever runs ralph org merge off the bypass list of one of them, or merge the PR yourself",
		g.subject(), mergeNamedBranches(bypassed))
}

// enforceClassic is step 5 of CheckMergeEnforcement, for a branch protection
// that requires branches to be up to date, with checks the number of its
// required status checks: ok only when enforce_admins is enabled, whether the
// caller is an admin of the repository or not; not ok when it is disabled or
// cannot be read. By default such a protection applies neither to admins nor
// to custom roles with the bypass branch protections permission, and
// enforce_admins (Do not allow bypassing the above settings) applies it to
// both (GitHub's About protected branches). permissions.admin does not show
// such a role, so a caller who is not an admin may bypass the protection all
// the same (cross-review #1 of the plan).
func (g *mergeGuaranteeRun) enforceClassic(checks int) (ok bool, detail string) {
	what := fmt.Sprintf("the branch protection of %s, requires branches to be up to date before merging with its %s",
		g.subject(), mergeChecksText(checks))
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	path := g.repoPath("branches", url.PathEscape(g.c.DefaultBranch), "protection", "enforce_admins")
	if why := g.ghJSON(path, "the enforce_admins JSON", &body); why != "" {
		return false, g.enforceUnread(why)
	}
	switch {
	case body.Enabled == nil:
		return false, fmt.Sprintf("%s, but %s printed no enabled, so whether it applies to admins and to custom roles that may "+
			"bypass branch protections is unknown: %s instead", what, g.command(false, path), mergeEnforceFix)
	case *body.Enabled:
		return true, what + ", and it applies to admins and to custom roles that may bypass branch protections too " +
			"(enforce_admins is enabled), so to you"
	}
	return false, fmt.Sprintf("%s, but enforce_admins is disabled, so it does not apply to admins of the repository or to "+
		"custom roles with the bypass branch protections permission, which you may hold though you are not an admin, and GitHub "+
		"may let you merge a head that is not up to date: enable Do not allow bypassing the above settings in the branch "+
		"protection, or %s", what, mergeEnforceFix)
}

// notEnforced is the Detail of step 6 of CheckMergeEnforcement: nothing makes
// the PR's required checks pass on an up-to-date head, with what rules and the
// branch protection (classic checks) set instead.
func (g *mergeGuaranteeRun) notEnforced(rules mergeRules, classic int) string {
	why := "nothing requires branches to be up to date before merging with required status checks"
	switch {
	case rules.strictNoChecks:
		why += " (the ruleset that requires branches to be up to date requires no status check itself)"
	case classic > 0:
		why += fmt.Sprintf(" (the branch protection requires %s, but not up to date)", mergeChecksText(classic))
	}
	return fmt.Sprintf("%s, does not make a PR's required status checks pass on a head with the latest %s before it merges: %s; "+
		"%s, or merge the PR yourself", g.subject(), g.c.DefaultBranch, why, mergeEnforceFix)
}
