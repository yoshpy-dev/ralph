package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The merge guarantee, stage 7c-2 of plan
// docs/plans/active/2026-10-14-org-merge-guarantee.md (FR-7 of
// docs/specs/2026-10-07-org-multi-org-director.md): what makes sure, when a
// PR merges into origin's default branch, that its required status checks
// passed on a head with the latest default branch. CheckMergeGuarantee
// decides the kind (`ralph doctor` runs it) by reading the default branch's
// rulesets and branch protection with git and gh through MergeOrderExec;
// SaveMergeGuarantee writes the result, the cache merge-guarantee.json, into
// an org state dir that already exists (C2); ReadMergeGuarantee reads the
// cache back for `ralph status` and `ralph org merge-order`, which never run
// gh.
//
// The repository is origin's: the host and owner/repo of `git remote get-url
// origin` (C6). For a fork's origin it is the fork, not the repository its
// PRs merge into. Every gh call names origin's host with --hostname, so
// GH_HOST and gh's default host do not apply; ssh.github.com is read as
// github.com, and an ssh alias (a Host in ~/.ssh/config) is not resolved, so
// gh fails on it and the kind is unknown (C7). What cannot be read is unknown,
// never github_protection (C4).

// The kinds of a merge guarantee (MergeGuaranteeCheck.Kind,
// MergeGuarantee.Kind).
const (
	// MergeGuaranteeGitHubProtection: GitHub makes the required status checks
	// of a PR pass on a head with the latest default branch before the merge
	// (an up-to-date rule or a merge queue, with at least one required check).
	MergeGuaranteeGitHubProtection = "github_protection"
	// MergeGuaranteePRChecksOnly: GitHub does not, so the guarantee is only
	// the gate and the CI that ran when the PR was made.
	MergeGuaranteePRChecksOnly = "pr_checks_only"
	// MergeGuaranteeUnknown: the kind cannot be read or decided.
	MergeGuaranteeUnknown = "unknown"
)

// The reasons ReadMergeGuarantee and ForDefaultBranch give for an unknown
// they make themselves (MergeGuarantee.Reason). A cached kind, unknown
// included, has no reason.
const (
	// MergeGuaranteeReasonNoLedger: the org state dir does not exist, so
	// doctor writes no cache there.
	MergeGuaranteeReasonNoLedger = "no_ledger"
	// MergeGuaranteeReasonNotChecked: the state dir has no cache.
	MergeGuaranteeReasonNotChecked = "not_checked"
	// MergeGuaranteeReasonUnreadable: the cache cannot be read, does not
	// parse, or has another version or kind.
	MergeGuaranteeReasonUnreadable = "unreadable"
	// MergeGuaranteeReasonStale: the cache's checked_at is more than
	// MergeGuaranteeMaxAge away from now, or does not parse (C5).
	MergeGuaranteeReasonStale = "stale"
	// MergeGuaranteeReasonOriginChanged: origin's host or owner/repo is not
	// the one the cache was checked for (C7).
	MergeGuaranteeReasonOriginChanged = "origin_changed"
	// MergeGuaranteeReasonOriginUnreadable: origin's URL cannot be read
	// here, or names no host and owner/repo.
	MergeGuaranteeReasonOriginUnreadable = "origin_unreadable"
	// MergeGuaranteeReasonDefaultBranchChanged: origin's default branch, as
	// merge-order read it, is not the one the cache was checked for.
	MergeGuaranteeReasonDefaultBranchChanged = "default_branch_changed"
)

const (
	// MergeGuaranteeFileName is the cache, within the org state dir.
	MergeGuaranteeFileName = "merge-guarantee.json"
	// MergeGuaranteeVersion is the cache's version; ReadMergeGuarantee reads
	// no other.
	MergeGuaranteeVersion = 1
	// MergeGuaranteeMaxAge is how long the cache is used after its
	// checked_at (C5).
	MergeGuaranteeMaxAge = 24 * time.Hour
	// defaultMergeGuaranteeTimeout bounds the whole of CheckMergeGuarantee,
	// git and every gh call together (C1), when MergeGuaranteeParams leaves
	// Timeout zero.
	defaultMergeGuaranteeTimeout = 20 * time.Second
	// defaultMergeGuaranteeGitTimeout bounds ReadMergeGuarantee's `git remote
	// get-url origin` when MergeGuaranteeReadParams leaves GitTimeout zero.
	defaultMergeGuaranteeGitTimeout = 10 * time.Second
	// mergeGuaranteeMaxErrText is how many bytes of a command's error a
	// Detail quotes, counted after escaping (mergeGuaranteeErrText).
	mergeGuaranteeMaxErrText = 300
	// mergeGuaranteeFix ends every pr_checks_only Detail: what to set. It names
	// only what the check reads: a merge queue counts from a ruleset only, never
	// from the branch protection (decide).
	mergeGuaranteeFix = "set Require branches to be up to date before merging with at least one required status check, " +
		"in a ruleset or the branch protection of the branch, or add a merge queue with at least one required " +
		"status check to a ruleset"
	// mergeGuaranteeGetURL names the git command in Details.
	mergeGuaranteeGetURL = "git remote get-url origin"
)

// MergeGuaranteePathIn is the cache's path within stateDir.
func MergeGuaranteePathIn(stateDir string) string {
	return filepath.Join(stateDir, MergeGuaranteeFileName)
}

// MergeGuaranteeCheck is what CheckMergeGuarantee found, and the cache's
// JSON shape as SaveMergeGuarantee writes it. Host, Repo, APIRepo and
// DefaultBranch are empty when the check stopped before reading them.
type MergeGuaranteeCheck struct {
	Version int    `json:"version"` // MergeGuaranteeVersion
	Kind    string `json:"kind"`    // MergeGuarantee*
	// Detail says in one sentence which setting decided the kind, or why it
	// is unknown.
	Detail string `json:"detail"`
	// Host is origin's URL's host as written, lowercased, without the user
	// and the port; Repo is origin's owner/repo.
	Host string `json:"host"`
	Repo string `json:"repo"`
	// APIRepo is the full_name gh returned for Repo: another name when the
	// repository was renamed or moved.
	APIRepo       string `json:"api_repo"`
	DefaultBranch string `json:"default_branch"`
	// CheckedAt is when the check started, UTC RFC3339.
	CheckedAt string `json:"checked_at"`
}

// MergeGuaranteeParams describes one CheckMergeGuarantee call.
type MergeGuaranteeParams struct {
	// Dir is the repository directory git and gh run in; empty means the
	// current directory.
	Dir string
	// Exec runs git and gh; nil means realMergeOrderExec.
	Exec MergeOrderExec
	// LookPath finds gh before any gh call; nil means exec.LookPath.
	LookPath func(file string) (string, error)
	// Timeout bounds the whole check; zero or less means 20 s.
	Timeout time.Duration
	// Now is the clock CheckedAt reads; nil means time.Now.
	Now func() time.Time
}

// CheckMergeGuarantee decides the merge guarantee of origin's repository's
// default branch. Every step runs under one deadline, p.Timeout (C1):
//
//  1. `git remote get-url origin` in p.Dir with GIT_TERMINAL_PROMPT=0, read
//     by mergeOriginFromURL (host and owner/repo).
//  2. gh is looked up with p.LookPath, and not run when it is not found.
//  3. `gh api repos/<o>/<r>`: default_branch and full_name.
//  4. `gh api --paginate repos/<o>/<r>/rules/branches/<branch>`: the rules
//     of the active rulesets (readMergeRules).
//  5. `gh api repos/<o>/<r>/branches/<branch>`: the classic branch
//     protection's required status checks (readMergeClassic).
//  6. Only when the branch protection requires a check:
//     `gh api repos/<o>/<r>/branches/<branch>/protection/required_status_checks`
//     for its strict, which needs admin access (readMergeClassicStrict).
//
// Each gh call carries --hostname <origin's host> (ssh.github.com is
// github.com) and GH_PROMPT_DISABLED=1, and the path escapes owner, repo
// and branch. It stops at the first step that decides (C4): github_protection
// when a ruleset's up-to-date rule has a required check of its own, when a
// ruleset has a merge queue and rulesets or the branch protection require a
// check, or when the branch protection is strict and requires a check;
// unknown when the branch protection requires a check and its strict cannot
// be read; pr_checks_only otherwise. Any failure, ctx done included, is
// unknown with a Detail that says why, so there is no error.
func CheckMergeGuarantee(ctx context.Context, p MergeGuaranteeParams) MergeGuaranteeCheck {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	c := MergeGuaranteeCheck{Version: MergeGuaranteeVersion, Kind: MergeGuaranteeUnknown, CheckedAt: now().UTC().Format(time.RFC3339)}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultMergeGuaranteeTimeout
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	g := &mergeGuaranteeRun{parent: ctx, ctx: checkCtx, timeout: timeout, dir: p.Dir, run: p.Exec, lookPath: p.LookPath, c: &c}
	if g.run == nil {
		g.run = realMergeOrderExec
	}
	if g.lookPath == nil {
		g.lookPath = exec.LookPath
	}
	c.Kind, c.Detail = g.decide()
	return c
}

// mergeGuaranteeRun is one CheckMergeGuarantee while it runs: ctx carries
// the deadline, parent is the caller's ctx, and c is filled in as the steps
// read origin and the repository.
type mergeGuaranteeRun struct {
	parent, ctx context.Context
	timeout     time.Duration
	dir         string
	run         MergeOrderExec
	lookPath    func(string) (string, error)
	c           *MergeGuaranteeCheck
	apiHost     string // the --hostname of every gh call
	repo        mergeRepo
}

// decide runs the steps of CheckMergeGuarantee and returns the kind and its
// Detail.
func (g *mergeGuaranteeRun) decide() (kind, detail string) {
	if why := g.readOrigin(); why != "" {
		return MergeGuaranteeUnknown, why
	}
	if _, err := g.lookPath("gh"); err != nil {
		return MergeGuaranteeUnknown, fmt.Sprintf("gh is not on PATH (%s), so the protection of the default branch of "+
			"origin's repository %s was not read", mergeGuaranteeErrText(err), g.repo)
	}
	if why := g.readRepository(); why != "" {
		return MergeGuaranteeUnknown, why
	}
	rules, why := g.readMergeRules()
	switch {
	case why != "":
		return MergeGuaranteeUnknown, why
	case rules.strictChecks > 0:
		return MergeGuaranteeGitHubProtection, g.protected(rules.strictChecks,
			"a ruleset requires branches to be up to date before merging")
	case rules.mergeQueue && rules.checks > 0:
		return MergeGuaranteeGitHubProtection, g.protected(rules.checks,
			"a ruleset has a merge queue, and rulesets require the checks")
	}
	classic, why := g.readMergeClassic()
	switch {
	case why != "":
		return MergeGuaranteeUnknown, why
	case rules.mergeQueue && classic > 0:
		return MergeGuaranteeGitHubProtection, g.protected(classic,
			"a ruleset has a merge queue, and the branch protection requires the checks")
	case classic > 0:
		strict, why := g.readMergeClassicStrict(classic)
		switch {
		case why != "":
			return MergeGuaranteeUnknown, why
		case strict:
			return MergeGuaranteeGitHubProtection, g.protected(classic,
				"the branch protection requires branches to be up to date before merging")
		}
	}
	return MergeGuaranteePRChecksOnly, g.checksOnly(rules, classic)
}

// readOrigin is step 1: it sets c.Host and c.Repo, or returns why it cannot.
func (g *mergeGuaranteeRun) readOrigin() (why string) {
	line, err := getMergeOriginURL(g.ctx, g.dir, g.run)
	if why := g.stopped(mergeGuaranteeGetURL); why != "" {
		return why
	}
	if err != nil {
		return fmt.Sprintf("%s failed (%s), so origin's repository is unknown", mergeGuaranteeGetURL, mergeGuaranteeErrText(err))
	}
	host, repo, ok := mergeOriginFromURL(line)
	if !ok {
		return "origin's URL names no host and owner/repo (a local path, file://, or a scheme other than https, http, ssh " +
			"and git), so there is no GitHub repository to read"
	}
	g.c.Host, g.c.Repo, g.repo = host, repo.String(), repo
	g.apiHost = mergeGuaranteeAPIHost(host)
	return ""
}

// mergeGuaranteeAPIHost is the --hostname gh reads origin's host with:
// host itself, except ssh.github.com (GitHub's ssh over port 443), which is
// github.com.
func mergeGuaranteeAPIHost(host string) string {
	if host == "ssh.github.com" {
		return "github.com"
	}
	return host
}

// readRepository is step 3: it sets c.DefaultBranch and c.APIRepo, or
// returns why it cannot. A gh that fails is told how to check gh's access
// to the host, and, for a host other than github.com, that an ssh alias is
// not resolved.
func (g *mergeGuaranteeRun) readRepository() (why string) {
	path := g.repoPath()
	out, err := g.gh(false, path)
	cmd := g.command(false, path)
	if why := g.stopped(cmd); why != "" {
		return why
	}
	if err != nil {
		why = fmt.Sprintf("%s failed (%s): check that gh can reach %s and is logged in to it (gh auth status --hostname %s)",
			cmd, mergeGuaranteeErrText(err), g.apiHost, g.apiHost)
		if g.apiHost != "github.com" {
			why += fmt.Sprintf("; if %s is an ssh alias (a Host in ~/.ssh/config), doctor cannot resolve it: point origin "+
				"at the real host", g.apiHost)
		}
		return why
	}
	var repo struct {
		DefaultBranch string `json:"default_branch"`
		FullName      string `json:"full_name"`
	}
	if err := json.Unmarshal([]byte(out), &repo); err != nil {
		return fmt.Sprintf("%s printed output that is not a repository's JSON (%s)", cmd, mergeGuaranteeErrText(err))
	}
	if repo.DefaultBranch == "" {
		return fmt.Sprintf("%s returned no default_branch, so the default branch of origin's repository %s is unknown", cmd, g.repo)
	}
	g.c.DefaultBranch, g.c.APIRepo = repo.DefaultBranch, repo.FullName
	return ""
}

// mergeRules is what the rules of the active rulesets say about the default
// branch. A required_status_checks rule counts its distinct check contexts.
type mergeRules struct {
	// strictChecks is the most checks one up-to-date rule
	// (strict_required_status_checks_policy) requires itself.
	strictChecks int
	// strictNoChecks: an up-to-date rule requires no check.
	strictNoChecks bool
	// checks is the distinct checks every required_status_checks rule
	// requires together.
	checks     int
	mergeQueue bool // a merge_queue rule
	// strictRulesets are the rulesets of the up-to-date rules that require
	// checks of their own, each once, in the order the output names them:
	// what CheckMergeEnforcement reads the caller's bypass of.
	// CheckMergeGuarantee does not read it.
	strictRulesets []mergeStrictRuleset
}

// mergeStrictRuleset is a ruleset whose up-to-date rule requires checks of
// its own: its id (0 when the rule names none) and the most checks one of its
// rules requires.
type mergeStrictRuleset struct {
	id, checks int
}

// ghBranchRule is one element of the rules/branches output.
type ghBranchRule struct {
	Type       string `json:"type"`
	RulesetID  int    `json:"ruleset_id"`
	Parameters struct {
		Strict bool `json:"strict_required_status_checks_policy"`
		Checks []struct {
			Context string `json:"context"`
		} `json:"required_status_checks"`
	} `json:"parameters"`
}

// readMergeRules is step 4. gh --paginate prints the pages as one JSON array
// or as one array per page, one after the other (it depends on gh's
// version), so it reads every array in the output; none is an error.
func (g *mergeGuaranteeRun) readMergeRules() (rules mergeRules, why string) {
	path := g.repoPath("rules", "branches", url.PathEscape(g.c.DefaultBranch))
	out, err := g.gh(true, path)
	cmd := g.command(true, path)
	if why := g.stopped(cmd); why != "" {
		return mergeRules{}, why
	}
	if err != nil {
		return mergeRules{}, fmt.Sprintf("%s failed (%s), so the rulesets of %s, are unknown", cmd, mergeGuaranteeErrText(err), g.subject())
	}
	var all []ghBranchRule
	pages := 0
	for dec := json.NewDecoder(strings.NewReader(out)); ; pages++ {
		var page []ghBranchRule
		if err := dec.Decode(&page); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return mergeRules{}, fmt.Sprintf("%s printed output that is not JSON lists of rules (%s)", cmd, mergeGuaranteeErrText(err))
		}
		all = append(all, page...)
	}
	if pages == 0 {
		return mergeRules{}, fmt.Sprintf("%s printed nothing, so the rulesets of %s, are unknown", cmd, g.subject())
	}
	required := make(map[string]bool)
	for _, r := range all {
		switch r.Type {
		case "merge_queue":
			rules.mergeQueue = true
		case "required_status_checks":
			own := make(map[string]bool)
			for _, c := range r.Parameters.Checks {
				if c.Context != "" {
					own[c.Context], required[c.Context] = true, true
				}
			}
			if r.Parameters.Strict {
				rules.strictChecks = max(rules.strictChecks, len(own))
				rules.strictNoChecks = rules.strictNoChecks || len(own) == 0
				if len(own) > 0 {
					rules.addStrictRuleset(r.RulesetID, len(own))
				}
			}
		}
	}
	rules.checks = len(required)
	return rules, ""
}

// addStrictRuleset adds the ruleset id, whose up-to-date rule requires checks
// checks of its own, to r.strictRulesets, or raises its count when it is
// there already.
func (r *mergeRules) addStrictRuleset(id, checks int) {
	for i := range r.strictRulesets {
		if r.strictRulesets[i].id == id {
			r.strictRulesets[i].checks = max(r.strictRulesets[i].checks, checks)
			return
		}
	}
	r.strictRulesets = append(r.strictRulesets, mergeStrictRuleset{id: id, checks: checks})
}

// readMergeClassic is step 5: the distinct checks the classic branch
// protection requires (contexts and checks together), 0 when it is not
// enabled or its required status checks are off.
func (g *mergeGuaranteeRun) readMergeClassic() (checks int, why string) {
	var branch struct {
		Protection struct {
			Enabled              bool `json:"enabled"`
			RequiredStatusChecks struct {
				EnforcementLevel string   `json:"enforcement_level"`
				Contexts         []string `json:"contexts"`
				Checks           []struct {
					Context string `json:"context"`
				} `json:"checks"`
			} `json:"required_status_checks"`
		} `json:"protection"`
	}
	if why := g.ghJSON(g.repoPath("branches", url.PathEscape(g.c.DefaultBranch)), "a branch's JSON", &branch); why != "" {
		return 0, why
	}
	p := branch.Protection
	if !p.Enabled || p.RequiredStatusChecks.EnforcementLevel == "off" {
		return 0, ""
	}
	required := make(map[string]bool)
	for _, c := range p.RequiredStatusChecks.Contexts {
		required[c] = true
	}
	for _, c := range p.RequiredStatusChecks.Checks {
		required[c.Context] = true
	}
	delete(required, "")
	return len(required), ""
}

// mergeClassicNotStrict are the messages of the 404 the strict endpoint
// answers when the branch protection has no required status checks, read as
// strict false.
var mergeClassicNotStrict = []string{"Branch not protected", "Required status checks not enabled"}

// readMergeClassicStrict is step 6, for a branch protection that requires
// checks. The endpoint needs admin access: GitHub answers others with a 404
// Not Found, and the strict is then unknown, with a Detail that says it
// needs admin access. A 403 leaves it unknown too, with a Detail that names
// admin access as only one possible cause: GitHub answers 403 without admin
// access and for other causes, such as a rate limit or an SSO authorization
// the token lacks. The failure's JSON body, which gh prints on stdout, gives
// the message and the status.
func (g *mergeGuaranteeRun) readMergeClassicStrict(checks int) (strict bool, why string) {
	path := g.repoPath("branches", url.PathEscape(g.c.DefaultBranch), "protection", "required_status_checks")
	out, err := g.gh(false, path)
	cmd := g.command(false, path)
	if why := g.stopped(cmd); why != "" {
		return false, why
	}
	if err == nil {
		var body struct {
			Strict *bool `json:"strict"`
		}
		if err := json.Unmarshal([]byte(out), &body); err != nil || body.Strict == nil {
			return false, fmt.Sprintf("%s printed output with no strict, so whether the branch protection of %s, requires "+
				"branches to be up to date before merging is unknown", cmd, g.subject())
		}
		return *body.Strict, ""
	}
	var body struct {
		Message string `json:"message"`
		Status  any    `json:"status"`
	}
	_ = json.Unmarshal([]byte(out), &body) // the body may be missing; the message then matches nothing
	switch {
	case slices.Contains(mergeClassicNotStrict, body.Message):
		return false, ""
	case body.Message == "Not Found":
		return false, fmt.Sprintf("the branch protection of %s, requires %s, but whether it requires branches to be up to date "+
			"before merging can be read only with admin access to the repository (%s failed: %s)", g.subject(),
			mergeChecksText(checks), cmd, mergeGuaranteeErrText(err))
	case fmt.Sprint(body.Status) == "403" || strings.Contains(err.Error(), "(HTTP 403)"):
		return false, fmt.Sprintf("the branch protection of %s, requires %s, but whether it requires branches to be up to date "+
			"before merging is unknown: GitHub answered 403, which can come from missing admin access to the repository or "+
			"from another cause, such as a rate limit or an SSO authorization the token lacks (%s failed: %s)", g.subject(),
			mergeChecksText(checks), cmd, mergeGuaranteeErrText(err))
	}
	return false, fmt.Sprintf("%s failed (%s), so whether the branch protection of %s, requires branches to be up to date "+
		"before merging is unknown", cmd, mergeGuaranteeErrText(err), g.subject())
}

// protected is a github_protection Detail: checks required checks, and how
// GitHub makes them pass on the latest default branch.
func (g *mergeGuaranteeRun) protected(checks int, how string) string {
	return fmt.Sprintf("%s, requires its %s to pass on a head with the latest %s before a PR merges: %s", g.subject(),
		mergeChecksText(checks), g.c.DefaultBranch, how)
}

// checksOnly is the pr_checks_only Detail: what the rulesets and the branch
// protection (classic checks) set, and what to set instead. An up-to-date
// rule or a merge queue with no check at all makes a branch up to date, but
// runs no check on it.
func (g *mergeGuaranteeRun) checksOnly(rules mergeRules, classic int) string {
	var why string
	switch {
	case rules.checks+classic == 0 && (rules.strictNoChecks || rules.mergeQueue):
		var set []string
		if rules.strictNoChecks {
			set = append(set, "a ruleset requires branches to be up to date before merging")
		}
		if rules.mergeQueue {
			set = append(set, "a ruleset has a merge queue")
		}
		why = strings.Join(set, " and ") + ", but no status check is required, so the branch becomes up to date but no " +
			"check runs on the updated head"
	case rules.checks+classic == 0:
		why = "no status check is required and nothing requires branches to be up to date before merging"
	default:
		var set []string
		if rules.checks > 0 {
			set = append(set, "rulesets require "+mergeChecksText(rules.checks))
		}
		if classic > 0 {
			set = append(set, "the branch protection requires "+mergeChecksText(classic))
		}
		why = strings.Join(set, " and ") + ", but nothing requires branches to be up to date before merging with them"
		if rules.strictNoChecks {
			why += " (the ruleset that requires branches to be up to date requires no status check itself)"
		}
	}
	return fmt.Sprintf("%s, does not make a PR's required status checks pass on a head with the latest %s before it merges: "+
		"%s; %s", g.subject(), g.c.DefaultBranch, why, mergeGuaranteeFix)
}

// subject names the default branch in a Detail, with the name GitHub
// returned when it is another than origin's.
func (g *mergeGuaranteeRun) subject() string {
	s := fmt.Sprintf("%s, the default branch of origin's repository %s", g.c.DefaultBranch, g.repo)
	if g.c.APIRepo != "" && !strings.EqualFold(g.c.APIRepo, g.repo.String()) {
		s += fmt.Sprintf(" (GitHub now names it %s)", g.c.APIRepo)
	}
	return s
}

// mergeChecksText is "1 required status check" or "<n> required status
// checks".
func mergeChecksText(n int) string {
	if n == 1 {
		return "1 required status check"
	}
	return fmt.Sprintf("%d required status checks", n)
}

// repoPath is the API path of origin's repository, then rest, which the
// caller escapes; owner and repository are escaped, so gh replaces no
// {owner} or {repo} placeholder in them.
func (g *mergeGuaranteeRun) repoPath(rest ...string) string {
	return strings.Join(append([]string{"repos", url.PathEscape(g.repo.owner), url.PathEscape(g.repo.name)}, rest...), "/")
}

// ghArgs are the arguments of `gh api` for path.
func (g *mergeGuaranteeRun) ghArgs(paginate bool, path string) []string {
	args := []string{"api", "--hostname", g.apiHost}
	if paginate {
		args = append(args, "--paginate")
	}
	return append(args, path)
}

// gh runs `gh api` for path in dir with GH_PROMPT_DISABLED=1 under the
// check's deadline.
func (g *mergeGuaranteeRun) gh(paginate bool, path string) (string, error) {
	return g.run(g.ctx, g.dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", g.ghArgs(paginate, path)...)
}

// command names the gh call for path in Details.
func (g *mergeGuaranteeRun) command(paginate bool, path string) string {
	return "gh " + strings.Join(g.ghArgs(paginate, path), " ")
}

// ghJSON runs gh for path and reads its output into v; why says what went
// wrong, what naming the JSON expected.
func (g *mergeGuaranteeRun) ghJSON(path, what string, v any) (why string) {
	out, err := g.gh(false, path)
	cmd := g.command(false, path)
	if why := g.stopped(cmd); why != "" {
		return why
	}
	if err != nil {
		return fmt.Sprintf("%s failed (%s), so the protection of %s, is unknown", cmd, mergeGuaranteeErrText(err), g.subject())
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return fmt.Sprintf("%s printed output that is not %s (%s)", cmd, what, mergeGuaranteeErrText(err))
	}
	return ""
}

// stopped is the Detail when the check's ctx is done after step ran: the
// caller's ctx was done, or the deadline passed. It is "" while neither.
func (g *mergeGuaranteeRun) stopped(step string) string {
	switch {
	case g.parent.Err() != nil:
		return fmt.Sprintf("the check was interrupted (%v) while %s ran, so the protection of origin's repository's default "+
			"branch is unknown", context.Cause(g.parent), step)
	case g.ctx.Err() != nil:
		return fmt.Sprintf("the check timed out after %s while %s ran, so the protection of origin's repository's default "+
			"branch is unknown: check the network and gh's access", g.timeout, step)
	}
	return ""
}

// getMergeOriginURL runs `git remote get-url origin` in dir with
// GIT_TERMINAL_PROMPT=0 and returns the first line it prints, trimmed.
func getMergeOriginURL(ctx context.Context, dir string, run MergeOrderExec) (string, error) {
	out, err := run(ctx, dir, []string{"GIT_TERMINAL_PROMPT=0"}, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	line, _, _ := strings.Cut(out, "\n")
	return strings.TrimSpace(line), nil
}

// mergeGuaranteeErrText is err's text for a Detail: each run of white space
// (newlines included) becomes one space, then each control character left,
// such as ESC, is written as its Go escape (\x1b) by PrintableInboxText,
// and then the text is cut to mergeGuaranteeMaxErrText bytes with "..."
// after it. The cut comes after the escaping, so the escapes count toward
// the limit and none is cut in two. doctor prints a Detail as it is, and
// status and merge-order pass it through PrintableInboxText again, which
// leaves an escaped text as it is.
func mergeGuaranteeErrText(err error) string {
	var b strings.Builder
	for _, r := range strings.Join(strings.Fields(err.Error()), " ") {
		printable := PrintableInboxText(string(r), false)
		if b.Len()+len(printable) > mergeGuaranteeMaxErrText {
			return b.String() + "..."
		}
		b.WriteString(printable)
	}
	return b.String()
}

// mergeEnforceFix is what the not-ok Details of CheckMergeEnforcement about a
// setting tell the reader to set: the only protection it reads that a caller
// without admin access can read, an up-to-date rule of a ruleset.
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
//  1. `gh api repos/<o>/<r>`: its default_branch must be p.DefaultBranch;
//     permissions.admin tells whether the caller is an admin, unknown when it
//     is missing (readEnforcementRepository).
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
//     needs admin access. Strict with checks is ok when the caller is not an
//     admin, or when `gh api repos/<o>/<r>/branches/<branch>/protection/enforce_admins`
//     is enabled (enforceClassic).
//  6. Anything else is not ok: nothing makes the PR up to date with its
//     required checks, ralph doctor's pr_checks_only (notEnforced).
//
// These fields were read live on 2026-10-17 (the plan's Assumptions): the
// ruleset_id of a rules/branches element, a ruleset's current_user_can_bypass
// (never), repository.mergeQueue (null with no queue), and permissions.admin.
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
	ok, detail := g.enforce()
	return MergeEnforcement{OK: ok, Detail: detail}
}

// enforce runs the steps of CheckMergeEnforcement and returns whether the
// merge may go on, and its Detail.
func (g *mergeGuaranteeRun) enforce() (ok bool, detail string) {
	admin, why := g.readEnforcementRepository()
	if why != "" {
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
			return g.enforceClassic(classic, admin)
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
// c.APIRepo and returns permissions.admin (nil when the answer has none), or
// why the check cannot go on.
func (g *mergeGuaranteeRun) readEnforcementRepository() (admin *bool, why string) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
		FullName      string `json:"full_name"`
		Permissions   *struct {
			Admin *bool `json:"admin"`
		} `json:"permissions"`
	}
	path := g.repoPath()
	if why := g.ghJSON(path, "a repository's JSON", &repo); why != "" {
		return nil, g.enforceUnread(why)
	}
	g.c.APIRepo = repo.FullName
	if repo.DefaultBranch != g.c.DefaultBranch {
		return nil, fmt.Sprintf("%s returned the default branch %q, but origin's HEAD names %s, so which branch's protection "+
			"to read is unknown: run ralph org merge-order again", g.command(false, path), repo.DefaultBranch, g.c.DefaultBranch)
	}
	if repo.Permissions != nil {
		admin = repo.Permissions.Admin
	}
	return admin, ""
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
// that requires checks checks and branches to be up to date: ok when the
// caller is not an admin of the repository (admin false), or is one and
// enforce_admins is enabled; not ok when admin is unknown (nil), or
// enforce_admins is disabled or cannot be read.
func (g *mergeGuaranteeRun) enforceClassic(checks int, admin *bool) (ok bool, detail string) {
	what := fmt.Sprintf("the branch protection of %s, requires branches to be up to date before merging with its %s",
		g.subject(), mergeChecksText(checks))
	switch {
	case admin == nil:
		return false, fmt.Sprintf("%s, but %s returned no permissions.admin, so whether it applies to you is unknown: %s instead",
			what, g.command(false, g.repoPath()), mergeEnforceFix)
	case !*admin:
		return true, what + ", and you are not an admin of the repository, so it applies to you"
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	path := g.repoPath("branches", url.PathEscape(g.c.DefaultBranch), "protection", "enforce_admins")
	if why := g.ghJSON(path, "the enforce_admins JSON", &body); why != "" {
		return false, g.enforceUnread(why)
	}
	switch {
	case body.Enabled == nil:
		return false, fmt.Sprintf("%s, and you are an admin of the repository, but %s printed no enabled, so whether it applies to "+
			"admins is unknown: %s instead", what, g.command(false, path), mergeEnforceFix)
	case *body.Enabled:
		return true, what + ", and it applies to admins too (enforce_admins is enabled), so to you"
	}
	return false, fmt.Sprintf("%s, but you are an admin of the repository and enforce_admins is disabled, so GitHub lets you merge "+
		"a head that is not up to date: enable Do not allow bypassing the above settings in the branch protection, or %s",
		what, mergeEnforceFix)
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

// MergeGuaranteeSave is what SaveMergeGuarantee did.
type MergeGuaranteeSave struct {
	Path string // the cache's path
	// NoLedger: the state dir does not exist (or is not a directory), so
	// nothing was written or created.
	NoLedger bool
	Written  bool
	// WriteErr is why the cache could not be written. The old cache was then
	// removed (Removed, false when there was none), unless RemoveErr says
	// why not: the old cache remains, possibly with an older kind.
	WriteErr  error
	Removed   bool
	RemoveErr error
}

// SaveMergeGuarantee writes c, unknown included (C5), as the cache in
// stateDir, through a temporary file renamed over it (writeJSONFileAtomic),
// only when stateDir already exists (C2). When the write fails it removes
// the old cache, so that no older kind outlives the failure; the result
// says whether that failed too.
func SaveMergeGuarantee(stateDir string, c MergeGuaranteeCheck) MergeGuaranteeSave {
	return saveMergeGuarantee(stateDir, c, writeJSONFileAtomic)
}

// saveMergeGuarantee is SaveMergeGuarantee with write in place of
// writeJSONFileAtomic.
func saveMergeGuarantee(stateDir string, c MergeGuaranteeCheck, write func(path string, v any) error) MergeGuaranteeSave {
	s := MergeGuaranteeSave{Path: MergeGuaranteePathIn(stateDir)}
	info, err := os.Stat(stateDir)
	switch {
	case errors.Is(err, fs.ErrNotExist) || err == nil && !info.IsDir():
		s.NoLedger = true
		return s
	case err != nil:
		s.WriteErr = fmt.Errorf("org: write %s: %w", s.Path, err)
	default:
		c.Version = MergeGuaranteeVersion
		s.WriteErr = write(s.Path, c)
	}
	if s.WriteErr == nil {
		s.Written = true
		return s
	}
	switch err := os.Remove(s.Path); {
	case err == nil:
		s.Removed = true
	case !errors.Is(err, fs.ErrNotExist):
		s.RemoveErr = fmt.Errorf("org: remove %s: %w", s.Path, err)
	}
	return s
}

// MergeGuarantee is the merge guarantee as `ralph status` and `ralph org
// merge-order` show it (ReadMergeGuarantee): the cache's kind and fields, or
// unknown with a Reason. The CLI prints Text, or marshals it as it is for
// --json.
type MergeGuarantee struct {
	Kind string `json:"kind"` // MergeGuarantee*
	// Reason is the MergeGuaranteeReason* of an unknown the reader made
	// itself, "" when Kind is the cache's.
	Reason string `json:"reason,omitempty"`
	// Detail is the cache's Detail when Reason is "", else what Text shows
	// within the parentheses: why the kind is unknown.
	Detail string `json:"detail,omitempty"`
	// Host, Repo, APIRepo, DefaultBranch and CheckedAt are the cache's, when
	// it was read.
	Host          string `json:"host,omitempty"`
	Repo          string `json:"repo,omitempty"`
	APIRepo       string `json:"api_repo,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
	CheckedAt     string `json:"checked_at,omitempty"`
	// LastKind is the cache's kind when the reader made Kind unknown.
	LastKind string `json:"last_kind,omitempty"`
}

// MergeGuaranteeReadParams describes one ReadMergeGuarantee call.
type MergeGuaranteeReadParams struct {
	// Dir is the repository directory git runs in; empty means the current
	// directory.
	Dir string
	// Exec runs git; nil means realMergeOrderExec.
	Exec MergeOrderExec
	// Now is the clock the cache's age is read against; nil means time.Now.
	Now func() time.Time
	// GitTimeout bounds `git remote get-url origin`; zero or less means 10 s.
	GitTimeout time.Duration
}

// ReadMergeGuarantee reads the cache in stateDir. The kind is unknown, with
// its Reason, when stateDir does not exist (no_ledger), the cache does not
// exist (not_checked), cannot be read, does not parse, or has another
// version or kind (unreadable), or its checked_at is more than
// MergeGuaranteeMaxAge before or after now, or does not parse (stale, C5).
// Only then, past all of these, it runs `git remote get-url origin` in p.Dir
// (local, never the network) and compares origin's host and owner/repo with
// the cache's, without case (C7): another is origin_changed, a git that
// fails or a URL that names no host and owner/repo origin_unreadable.
// Otherwise the kind is the cache's, unknown included.
func ReadMergeGuarantee(ctx context.Context, stateDir string, p MergeGuaranteeReadParams) MergeGuarantee {
	unknown := func(reason, detail string) MergeGuarantee {
		return MergeGuarantee{Kind: MergeGuaranteeUnknown, Reason: reason, Detail: detail}
	}
	notReadable := unknown(MergeGuaranteeReasonUnreadable, "cache unreadable: run ralph doctor")
	info, err := os.Stat(stateDir)
	switch {
	case errors.Is(err, fs.ErrNotExist) || err == nil && !info.IsDir():
		return unknown(MergeGuaranteeReasonNoLedger, "no org ledger yet")
	case err != nil:
		return notReadable
	}
	data, err := os.ReadFile(MergeGuaranteePathIn(stateDir))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return unknown(MergeGuaranteeReasonNotChecked, "not checked: run ralph doctor")
	case err != nil:
		return notReadable
	}
	var c MergeGuaranteeCheck
	if err := json.Unmarshal(data, &c); err != nil || c.Version != MergeGuaranteeVersion ||
		!slices.Contains([]string{MergeGuaranteeGitHubProtection, MergeGuaranteePRChecksOnly, MergeGuaranteeUnknown}, c.Kind) {
		return notReadable
	}
	g := MergeGuarantee{
		Kind: c.Kind, Detail: c.Detail, Host: c.Host, Repo: c.Repo, APIRepo: c.APIRepo,
		DefaultBranch: c.DefaultBranch, CheckedAt: c.CheckedAt,
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	if at, err := time.Parse(time.RFC3339, c.CheckedAt); err != nil || now().Sub(at).Abs() > MergeGuaranteeMaxAge {
		return g.unknownFor(MergeGuaranteeReasonStale, fmt.Sprintf("stale: last %s at %s; run ralph doctor", c.Kind, c.CheckedAt))
	}
	host, repo, why := readMergeGuaranteeOrigin(ctx, p)
	checked := mergeOriginName(c.Host, c.Repo)
	switch {
	case why != "":
		return g.unknownFor(MergeGuaranteeReasonOriginUnreadable, fmt.Sprintf("origin cannot be read here (%s); doctor "+
			"checked %s", why, checked))
	case !strings.EqualFold(host, c.Host) || !strings.EqualFold(repo, c.Repo):
		return g.unknownFor(MergeGuaranteeReasonOriginChanged, fmt.Sprintf("doctor checked %s; origin is now %s", checked,
			mergeOriginName(host, repo)))
	}
	return g
}

// readMergeGuaranteeOrigin is origin's host and owner/repo for
// ReadMergeGuarantee, or why they cannot be read.
func readMergeGuaranteeOrigin(ctx context.Context, p MergeGuaranteeReadParams) (host, repo, why string) {
	run := p.Exec
	if run == nil {
		run = realMergeOrderExec
	}
	timeout := p.GitTimeout
	if timeout <= 0 {
		timeout = defaultMergeGuaranteeGitTimeout
	}
	getCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	line, err := getMergeOriginURL(getCtx, p.Dir, run)
	switch {
	case ctx.Err() != nil:
		return "", "", fmt.Sprintf("interrupted (%v) while %s ran", context.Cause(ctx), mergeGuaranteeGetURL)
	case err != nil && getCtx.Err() != nil:
		return "", "", fmt.Sprintf("%s timed out after %s", mergeGuaranteeGetURL, timeout)
	case err != nil:
		return "", "", fmt.Sprintf("%s failed: %s", mergeGuaranteeGetURL, mergeGuaranteeErrText(err))
	}
	h, r, ok := mergeOriginFromURL(line)
	if !ok {
		return "", "", "its URL names no host and owner/repo"
	}
	return h, r.String(), ""
}

// mergeOriginName is host/owner/repo, or "no readable origin" for a cache
// whose check could not read origin (host and repo both empty).
func mergeOriginName(host, repo string) string {
	if host == "" && repo == "" {
		return "no readable origin"
	}
	return host + "/" + repo
}

// unknownFor is g made unknown for reason, with detail, keeping the cache's
// fields and its kind as LastKind.
func (g MergeGuarantee) unknownFor(reason, detail string) MergeGuarantee {
	g.LastKind, g.Kind, g.Reason, g.Detail = g.Kind, MergeGuaranteeUnknown, reason, detail
	return g
}

// ForDefaultBranch is g for origin's default branch branch, as merge-order
// read it: unknown (default_branch_changed) when g is the cache's kind and
// the cache was checked for another branch. A g that is unknown with a
// Reason, or whose cache names no branch or branch is "", is returned as it
// is.
func (g MergeGuarantee) ForDefaultBranch(branch string) MergeGuarantee {
	if g.Reason != "" || g.DefaultBranch == "" || branch == "" || g.DefaultBranch == branch {
		return g
	}
	return g.unknownFor(MergeGuaranteeReasonDefaultBranchChanged, fmt.Sprintf("doctor checked %s; origin's default branch "+
		"is now %s", g.DefaultBranch, branch))
}

// Text is g on one line, what follows "merge guarantee: " in the CLI's
// output: `<kind> (<branch> of <host>/<repo>, checked <checked_at>):
// <detail>` for the cache's kind, `unknown (<detail>)` for an unknown with a
// Reason.
func (g MergeGuarantee) Text() string {
	if g.Reason != "" {
		return MergeGuaranteeUnknown + " (" + g.Detail + ")"
	}
	var of string
	switch {
	case g.DefaultBranch != "":
		of = g.DefaultBranch + " of " + mergeOriginName(g.Host, g.Repo) + ", "
	case g.Host != "" || g.Repo != "":
		of = mergeOriginName(g.Host, g.Repo) + ", "
	}
	return fmt.Sprintf("%s (%schecked %s): %s", g.Kind, of, g.CheckedAt, g.Detail)
}
