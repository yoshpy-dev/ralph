package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// The merge guarantee at the org layer (plan
// docs/plans/active/2026-10-14-org-merge-guarantee.md, the org-layer half of
// AC1 to AC5 and AC10, and the cache half of AC6 and AC7):
// CheckMergeGuarantee with a fake MergeOrderExec standing in for git and gh
// and a fake LookPath, and the cache through SaveMergeGuarantee and
// ReadMergeGuarantee in temp dirs. No test runs gh or reaches a network.

// mgClock is the fixtures' clock; mgCheckedAt is it as CheckedAt.
var mgClock = time.Date(2026, 10, 14, 5, 18, 40, 0, time.UTC)

const mgCheckedAt = "2026-10-14T05:18:40Z"

// mgOriginURL is origin's URL by default: github.com, acme/app.
const mgOriginURL = "git@github.com:acme/app.git"

// The gh api endpoints of acme/app with the default branch main.
const (
	mgRepoEP   = "repos/acme/app"
	mgRulesEP  = "repos/acme/app/rules/branches/main"
	mgBranchEP = "repos/acme/app/branches/main"
	mgStrictEP = "repos/acme/app/branches/main/protection/required_status_checks"
)

// mgFix is the end of every pr_checks_only Detail.
const mgFix = "; set Require branches to be up to date before merging with at least one required status check, " +
	"in a ruleset or the branch protection of the branch, or add a merge queue with at least one required " +
	"status check to a ruleset"

// mgSubject names main of acme/app in a Detail.
const mgSubject = "main, the default branch of origin's repository acme/app"

// mgExec is a fake MergeOrderExec. `git remote get-url origin` answers
// originErr, else origin. gh answers, per endpoint (the last argument), the
// reply in replies; an endpoint with no reply is an error. The command whose
// key (git, or a gh endpoint) is block waits for ctx instead.
type mgExec struct {
	mu        sync.Mutex
	calls     []mgCall
	origin    string
	originErr error
	replies   map[string]mgReply
	block     string
}

type mgReply struct {
	out string
	err error
}

type mgCall struct {
	dir      string
	env      []string
	name     string
	args     []string
	deadline time.Time // ctx's, zero when it has none
}

// newMGExec is an mgExec for acme/app on github.com whose default branch
// main has no ruleset and no branch protection.
func newMGExec() *mgExec {
	return &mgExec{origin: mgOriginURL + "\n", replies: map[string]mgReply{
		mgRepoEP:   {out: mgRepoJSON("main", "acme/app")},
		mgRulesEP:  {out: "[]"},
		mgBranchEP: {out: mgClassicJSON(false, "off", nil, nil)},
	}}
}

func (e *mgExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	deadline, _ := ctx.Deadline()
	e.mu.Lock()
	e.calls = append(e.calls, mgCall{dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args), deadline: deadline})
	e.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	key := name
	switch name {
	case "git":
		if !slices.Equal(args, []string{"remote", "get-url", "origin"}) {
			return "", fmt.Errorf("mgExec: unexpected git %q", args)
		}
	case "gh":
		key = args[len(args)-1]
	default:
		return "", fmt.Errorf("mgExec: unexpected command %s", name)
	}
	if e.block == key {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if name == "git" {
		if e.originErr != nil {
			return "", e.originErr
		}
		return e.origin, nil
	}
	r, ok := e.replies[key]
	if !ok {
		return "", fmt.Errorf("mgExec: unexpected gh api %s", key)
	}
	return r.out, r.err
}

// callsOf returns the calls of the command name.
func (e *mgExec) callsOf(name string) []mgCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []mgCall
	for _, c := range e.calls {
		if c.name == name {
			out = append(out, c)
		}
	}
	return out
}

// endpoints are the gh api endpoints called, in order.
func (e *mgExec) endpoints() []string {
	var out []string
	for _, c := range e.callsOf("gh") {
		out = append(out, c.args[len(c.args)-1])
	}
	return out
}

// assertGHCalls fails the test unless every gh call is `gh api --hostname
// <host> [--paginate] <endpoint>` in /repo with GH_PROMPT_DISABLED=1, with
// --paginate on the rules endpoint only.
func (e *mgExec) assertGHCalls(t *testing.T, host string) {
	t.Helper()
	for _, c := range e.callsOf("gh") {
		ep := c.args[len(c.args)-1]
		want := []string{"api", "--hostname", host, ep}
		if strings.Contains(ep, "/rules/branches/") {
			want = []string{"api", "--hostname", host, "--paginate", ep}
		}
		if !slices.Equal(c.args, want) || c.dir != "/repo" || !slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) {
			t.Errorf("gh call = %q in %q with %q, want %q in /repo with GH_PROMPT_DISABLED=1", c.args, c.dir, c.env, want)
		}
	}
}

func mgFound(string) (string, error) { return "/usr/local/bin/gh", nil }

func mgNotFound(file string) (string, error) {
	return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
}

func mgParams(e *mgExec) MergeGuaranteeParams {
	return MergeGuaranteeParams{Dir: "/repo", Exec: e.run, LookPath: mgFound, Now: func() time.Time { return mgClock }}
}

// mgCheck is CheckMergeGuarantee with mgParams(e).
func mgCheck(e *mgExec) MergeGuaranteeCheck {
	return CheckMergeGuarantee(context.Background(), mgParams(e))
}

func mgJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func mgRepoJSON(def, fullName string) string {
	return mgJSON(map[string]any{"id": 1, "full_name": fullName, "default_branch": def, "private": false})
}

// mgChecksRule is a required_status_checks rule of the rules/branches output.
func mgChecksRule(strict bool, contexts ...string) string {
	checks := []map[string]any{}
	for _, c := range contexts {
		checks = append(checks, map[string]any{"context": c, "integration_id": 15368})
	}
	return mgJSON(map[string]any{
		"type": "required_status_checks", "ruleset_source_type": "Repository", "ruleset_source": "acme/app", "ruleset_id": 1,
		"parameters": map[string]any{
			"strict_required_status_checks_policy": strict, "do_not_enforce_on_create": false, "required_status_checks": checks,
		},
	})
}

const (
	mgMergeQueueRule = `{"type":"merge_queue","ruleset_id":2,"parameters":{"merge_method":"SQUASH","grouping_strategy":"ALLGREEN"}}`
	mgDeletionRule   = `{"type":"deletion","ruleset_id":3}`
	mgReviewRule     = `{"type":"pull_request","ruleset_id":4,"parameters":{"required_approving_review_count":1}}`
)

func mgRules(rules ...string) string { return "[" + strings.Join(rules, ",") + "]" }

// mgClassicJSON is a branches/<branch> answer whose protection has enabled,
// the required status checks' enforcement level, contexts and checks.
func mgClassicJSON(enabled bool, level string, contexts, checks []string) string {
	cs := []map[string]any{}
	for _, c := range checks {
		cs = append(cs, map[string]any{"context": c, "app_id": nil})
	}
	return mgJSON(map[string]any{
		"name": "main", "protected": enabled,
		"protection": map[string]any{
			"enabled": enabled,
			"required_status_checks": map[string]any{
				"enforcement_level": level, "contexts": append([]string{}, contexts...), "checks": cs,
			},
		},
	})
}

// mgClassicRequires is a branch protection that requires contexts, as both
// contexts and checks.
func mgClassicRequires(contexts ...string) string {
	return mgClassicJSON(true, "non_admins", contexts, contexts)
}

func mgStrictJSON(strict bool) string {
	return fmt.Sprintf(`{"url":"https://api.github.com/x","strict":%t,"contexts":["ci"],"checks":[{"context":"ci","app_id":null}]}`, strict)
}

// mgAPIError is how gh api fails on an HTTP error: the JSON body on stdout,
// `gh: <message> (HTTP <status>)` on stderr, exit 1.
func mgAPIError(message, status string) mgReply {
	return mgReply{
		out: fmt.Sprintf(`{"message":%q,"documentation_url":"https://docs.github.com/rest","status":%q}`, message, status),
		err: fmt.Errorf("exit status 1: gh: %s (HTTP %s)", message, status),
	}
}

// mgAPIErrorReply is mgAPIError as a pointer, for a table's optional reply.
func mgAPIErrorReply(message, status string) *mgReply {
	r := mgAPIError(message, status)
	return &r
}

// TestMergeGuarantee_RulesetsDecideTheKind: an up-to-date rule with a check
// of its own, or a merge queue with ruleset checks, is github_protection
// without reading the branch protection; an up-to-date rule or a merge queue
// with no check, checks that are not up to date, and no rule at all are
// pr_checks_only, and say what to set (AC1, AC3).
func TestMergeGuarantee_RulesetsDecideTheKind(t *testing.T) {
	both := []string{mgRepoEP, mgRulesEP}
	three := []string{mgRepoEP, mgRulesEP, mgBranchEP}
	for _, tc := range []struct {
		name      string
		rules     string
		kind      string
		detail    string // the exact Detail, or a part of it with partial
		partial   bool
		endpoints []string
	}{
		{"up-to-date rule with checks", mgRules(mgChecksRule(true, "ci", "lint")), MergeGuaranteeGitHubProtection,
			mgSubject + ", requires its 2 required status checks to pass on a head with the latest main before a PR merges: a " +
				"ruleset requires branches to be up to date before merging", false, both},
		{"the up-to-date rule with the most checks counts", mgRules(mgChecksRule(true, "a"), mgChecksRule(true, "b", "c")),
			MergeGuaranteeGitHubProtection, "requires its 2 required status checks", true, both},
		{"merge queue with ruleset checks", mgRules(mgMergeQueueRule, mgChecksRule(false, "ci", "ci", "lint")),
			MergeGuaranteeGitHubProtection, mgSubject + ", requires its 2 required status checks to pass on a head with the " +
				"latest main before a PR merges: a ruleset has a merge queue, and rulesets require the checks", false, both},
		{"up-to-date rule with no check", mgRules(mgChecksRule(true)), MergeGuaranteePRChecksOnly,
			mgSubject + ", does not make a PR's required status checks pass on a head with the latest main before it merges: " +
				"a ruleset requires branches to be up to date before merging, but no status check is required, so the branch " +
				"becomes up to date but no check runs on the updated head" + mgFix, false, three},
		{"merge queue alone", mgRules(mgMergeQueueRule), MergeGuaranteePRChecksOnly,
			"a ruleset has a merge queue, but no status check is required, so the branch becomes up to date but no check runs " +
				"on the updated head" + mgFix, true, three},
		{"up-to-date rule and merge queue with no check", mgRules(mgChecksRule(true), mgMergeQueueRule), MergeGuaranteePRChecksOnly,
			"a ruleset requires branches to be up to date before merging and a ruleset has a merge queue, but no status check " +
				"is required", true, three},
		{"required checks with no up-to-date rule", mgRules(mgChecksRule(false, "ci")), MergeGuaranteePRChecksOnly,
			mgSubject + ", does not make a PR's required status checks pass on a head with the latest main before it merges: " +
				"rulesets require 1 required status check, but nothing requires branches to be up to date before merging with " +
				"them" + mgFix, false, three},
		{"an up-to-date rule with no check beside another rule's checks", mgRules(mgChecksRule(true), mgChecksRule(false, "ci")),
			MergeGuaranteePRChecksOnly, "rulesets require 1 required status check, but nothing requires branches to be up to " +
				"date before merging with them (the ruleset that requires branches to be up to date requires no status check " +
				"itself)" + mgFix, true, three},
		{"no rule", "[]", MergeGuaranteePRChecksOnly,
			mgSubject + ", does not make a PR's required status checks pass on a head with the latest main before it merges: " +
				"no status check is required and nothing requires branches to be up to date before merging" + mgFix, false, three},
		{"other rule types only", mgRules(mgDeletionRule, mgReviewRule), MergeGuaranteePRChecksOnly,
			"no status check is required and nothing requires branches to be up to date before merging" + mgFix, true, three},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMGExec()
			e.replies[mgRulesEP] = mgReply{out: tc.rules}
			c := mgCheck(e)
			if c.Kind != tc.kind {
				t.Fatalf("kind = %s (%s), want %s", c.Kind, c.Detail, tc.kind)
			}
			if tc.partial && !strings.Contains(c.Detail, tc.detail) || !tc.partial && c.Detail != tc.detail {
				t.Errorf("detail = %q\nwant (partial %v) %q", c.Detail, tc.partial, tc.detail)
			}
			if got := e.endpoints(); !slices.Equal(got, tc.endpoints) {
				t.Errorf("gh endpoints = %q, want %q", got, tc.endpoints)
			}
			e.assertGHCalls(t, "github.com")
		})
	}
}

// TestMergeGuarantee_ClassicProtectionDecidesTheKind: the branch
// protection's required checks with strict true are github_protection, with
// strict false or off pr_checks_only; a strict that needs admin access (404
// Not Found, 403) is unknown, a 404 that says the branch or its checks are
// not protected is strict false, any other failure unknown; strict is read
// only when the branch protection requires a check and nothing decided
// before it (AC2, AC3).
func TestMergeGuarantee_ClassicProtectionDecidesTheKind(t *testing.T) {
	all := []string{mgRepoEP, mgRulesEP, mgBranchEP, mgStrictEP}
	three := []string{mgRepoEP, mgRulesEP, mgBranchEP}
	admin := func(answer string) string {
		return "the branch protection of " + mgSubject + ", requires 1 required status check, but whether it requires " +
			"branches to be up to date before merging can be read only with admin access to the repository (gh api " +
			"--hostname github.com " + mgStrictEP + " failed: " + answer + ")"
	}
	for _, tc := range []struct {
		name      string
		rules     string // "" means "[]"
		branch    string
		strict    *mgReply
		kind      string
		detail    string
		partial   bool
		endpoints []string
	}{
		{"strict with a required check", "", mgClassicRequires("ci"), &mgReply{out: mgStrictJSON(true)},
			MergeGuaranteeGitHubProtection, mgSubject + ", requires its 1 required status check to pass on a head with the " +
				"latest main before a PR merges: the branch protection requires branches to be up to date before merging",
			false, all},
		{"not strict", "", mgClassicRequires("ci"), &mgReply{out: mgStrictJSON(false)}, MergeGuaranteePRChecksOnly,
			mgSubject + ", does not make a PR's required status checks pass on a head with the latest main before it merges: " +
				"the branch protection requires 1 required status check, but nothing requires branches to be up to date " +
				"before merging with them" + mgFix, false, all},
		{"required checks off", "", mgClassicJSON(true, "off", []string{"ci"}, []string{"ci"}), nil, MergeGuaranteePRChecksOnly,
			"no status check is required and nothing requires branches to be up to date before merging" + mgFix, true, three},
		{"protection not enabled", "", mgClassicJSON(false, "non_admins", []string{"ci"}, nil), nil, MergeGuaranteePRChecksOnly,
			"no status check is required", true, three},
		{"protection with no required check", "", mgClassicRequires(), nil, MergeGuaranteePRChecksOnly,
			"no status check is required", true, three},
		{"strict needs admin access: 404 Not Found", "", mgClassicRequires("ci"), mgAPIErrorReply("Not Found", "404"),
			MergeGuaranteeUnknown, admin("exit status 1: gh: Not Found (HTTP 404)"), false, all},
		{"strict needs admin access: 403 in the body", "", mgClassicRequires("ci"),
			mgAPIErrorReply("Must have admin rights to Repository.", "403"), MergeGuaranteeUnknown,
			admin("exit status 1: gh: Must have admin rights to Repository. (HTTP 403)"), false, all},
		{"strict needs admin access: 403 on stderr only", "", mgClassicRequires("ci"),
			&mgReply{err: errors.New("exit status 1: gh: Forbidden (HTTP 403)")}, MergeGuaranteeUnknown,
			admin("exit status 1: gh: Forbidden (HTTP 403)"), false, all},
		{"strict answers Branch not protected", "", mgClassicRequires("ci"), mgAPIErrorReply("Branch not protected", "404"),
			MergeGuaranteePRChecksOnly, "the branch protection requires 1 required status check, but nothing requires", true, all},
		{"strict answers Required status checks not enabled", "", mgClassicRequires("ci"),
			mgAPIErrorReply("Required status checks not enabled", "404"), MergeGuaranteePRChecksOnly,
			"the branch protection requires 1 required status check, but nothing requires", true, all},
		{"strict fails another way", "", mgClassicRequires("ci"), mgAPIErrorReply("Server Error", "500"), MergeGuaranteeUnknown,
			"gh api --hostname github.com " + mgStrictEP + " failed (exit status 1: gh: Server Error (HTTP 500)), so whether " +
				"the branch protection of " + mgSubject + ", requires branches to be up to date before merging is unknown",
			false, all},
		{"strict answer with no strict", "", mgClassicRequires("ci"), &mgReply{out: `{"url":"x","contexts":["ci"]}`},
			MergeGuaranteeUnknown, "printed output with no strict, so whether the branch protection of " + mgSubject +
				", requires branches to be up to date before merging is unknown", true, all},
		{"strict answer that is not JSON", "", mgClassicRequires("ci"), &mgReply{out: "<html>"}, MergeGuaranteeUnknown,
			"printed output with no strict", true, all},
		{"merge queue with the branch protection's checks", mgRules(mgMergeQueueRule), mgClassicRequires("ci", "lint"), nil,
			MergeGuaranteeGitHubProtection, mgSubject + ", requires its 2 required status checks to pass on a head with the " +
				"latest main before a PR merges: a ruleset has a merge queue, and the branch protection requires the checks",
			false, three},
		{"contexts and checks count once each", "", mgClassicJSON(true, "everyone", []string{"ci", "lint", ""}, []string{"ci", "e2e"}),
			&mgReply{out: mgStrictJSON(false)}, MergeGuaranteePRChecksOnly,
			"the branch protection requires 3 required status checks, but nothing requires", true, all},
		{"ruleset and branch protection checks, neither up to date", mgRules(mgChecksRule(false, "ci")),
			mgClassicRequires("lint"), &mgReply{out: mgStrictJSON(false)}, MergeGuaranteePRChecksOnly,
			"rulesets require 1 required status check and the branch protection requires 1 required status check, but " +
				"nothing requires branches to be up to date before merging with them" + mgFix, true, all},
		{"ruleset checks and a strict that needs admin access", mgRules(mgChecksRule(false, "ci")), mgClassicRequires("ci"),
			mgAPIErrorReply("Not Found", "404"), MergeGuaranteeUnknown, "can be read only with admin access", true, all},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMGExec()
			if tc.rules != "" {
				e.replies[mgRulesEP] = mgReply{out: tc.rules}
			}
			e.replies[mgBranchEP] = mgReply{out: tc.branch}
			if tc.strict != nil {
				e.replies[mgStrictEP] = *tc.strict
			}
			c := mgCheck(e)
			if c.Kind != tc.kind {
				t.Fatalf("kind = %s (%s), want %s", c.Kind, c.Detail, tc.kind)
			}
			if tc.partial && !strings.Contains(c.Detail, tc.detail) || !tc.partial && c.Detail != tc.detail {
				t.Errorf("detail = %q\nwant (partial %v) %q", c.Detail, tc.partial, tc.detail)
			}
			if got := e.endpoints(); !slices.Equal(got, tc.endpoints) {
				t.Errorf("gh endpoints = %q, want %q", got, tc.endpoints)
			}
			e.assertGHCalls(t, "github.com")
		})
	}
}

// TestMergeGuarantee_FailuresAreUnknownWithAReason: origin missing, an
// origin URL with no host or owner/repo, gh not on PATH (gh never runs), a
// gh that fails (on an ssh alias host too), and output that is not the JSON
// expected are each unknown with a Detail that says why, on one line (AC4).
func TestMergeGuarantee_FailuresAreUnknownWithAReason(t *testing.T) {
	alias := "exit status 1: error connecting to github.com.emu\ncheck your internet connection or https://githubstatus.com"
	for _, tc := range []struct {
		name      string
		setup     func(e *mgExec, p *MergeGuaranteeParams)
		detail    string // a part of the Detail
		endpoints []string
		host      string // the Host recorded, "" for none
	}{
		{"origin is missing", func(e *mgExec, _ *MergeGuaranteeParams) {
			e.originErr = errors.New("exit status 2: error: No such remote 'origin'")
		}, "git remote get-url origin failed (exit status 2: error: No such remote 'origin'), so origin's repository is " +
			"unknown", nil, ""},
		{"origin is a local path", func(e *mgExec, _ *MergeGuaranteeParams) { e.origin = "/srv/git/app.git\n" },
			"origin's URL names no host and owner/repo (a local path, file://, or a scheme other than https, http, ssh and " +
				"git), so there is no GitHub repository to read", nil, ""},
		{"origin is a file URL", func(e *mgExec, _ *MergeGuaranteeParams) { e.origin = "file:///srv/acme/app.git\n" },
			"origin's URL names no host and owner/repo", nil, ""},
		{"origin names no owner/repo", func(e *mgExec, _ *MergeGuaranteeParams) { e.origin = "https://github.com/acme\n" },
			"origin's URL names no host and owner/repo", nil, ""},
		{"get-url prints nothing", func(e *mgExec, _ *MergeGuaranteeParams) { e.origin = "" },
			"origin's URL names no host and owner/repo", nil, ""},
		{"gh is not on PATH", func(_ *mgExec, p *MergeGuaranteeParams) { p.LookPath = mgNotFound },
			`gh is not on PATH (exec: "gh": executable file not found in $PATH), so the protection of the default branch of ` +
				"origin's repository acme/app was not read", nil, "github.com"},
		{"gh fails on an ssh alias host", func(e *mgExec, _ *MergeGuaranteeParams) {
			e.origin = "git@github.com.emu:acme/app.git\n"
			e.replies[mgRepoEP] = mgReply{err: errors.New(alias)}
		}, "gh api --hostname github.com.emu repos/acme/app failed (exit status 1: error connecting to github.com.emu check " +
			"your internet connection or https://githubstatus.com): check that gh can reach github.com.emu and is logged in " +
			"to it (gh auth status --hostname github.com.emu); if github.com.emu is an ssh alias (a Host in ~/.ssh/config), " +
			"doctor cannot resolve it: point origin at the real host", []string{mgRepoEP}, "github.com.emu"},
		{"gh fails on github.com", func(e *mgExec, _ *MergeGuaranteeParams) { e.replies[mgRepoEP] = mgAPIError("Not Found", "404") },
			"gh api --hostname github.com repos/acme/app failed (exit status 1: gh: Not Found (HTTP 404)): check that gh can " +
				"reach github.com and is logged in to it (gh auth status --hostname github.com)", []string{mgRepoEP}, "github.com"},
		{"repository answer is not JSON", func(e *mgExec, _ *MergeGuaranteeParams) { e.replies[mgRepoEP] = mgReply{out: "<html>"} },
			"gh api --hostname github.com repos/acme/app printed output that is not a repository's JSON", []string{mgRepoEP}, "github.com"},
		{"repository answer has no default_branch", func(e *mgExec, _ *MergeGuaranteeParams) {
			e.replies[mgRepoEP] = mgReply{out: `{"full_name":"acme/app"}`}
		}, "gh api --hostname github.com repos/acme/app returned no default_branch, so the default branch of origin's " +
			"repository acme/app is unknown", []string{mgRepoEP}, "github.com"},
		{"rules call fails", func(e *mgExec, _ *MergeGuaranteeParams) { e.replies[mgRulesEP] = mgAPIError("Server Error", "500") },
			"gh api --hostname github.com --paginate " + mgRulesEP + " failed (exit status 1: gh: Server Error (HTTP 500)), " +
				"so the rulesets of " + mgSubject + ", are unknown", []string{mgRepoEP, mgRulesEP}, "github.com"},
		{"rules answer is not a list", func(e *mgExec, _ *MergeGuaranteeParams) {
			e.replies[mgRulesEP] = mgReply{out: `{"message":"x"}`}
		}, "--paginate " + mgRulesEP + " printed output that is not JSON lists of rules", []string{mgRepoEP, mgRulesEP}, "github.com"},
		{"rules answer is empty", func(e *mgExec, _ *MergeGuaranteeParams) { e.replies[mgRulesEP] = mgReply{out: " \n"} },
			"--paginate " + mgRulesEP + " printed nothing, so the rulesets of " + mgSubject + ", are unknown",
			[]string{mgRepoEP, mgRulesEP}, "github.com"},
		{"branch call fails", func(e *mgExec, _ *MergeGuaranteeParams) { e.replies[mgBranchEP] = mgAPIError("Server Error", "502") },
			"gh api --hostname github.com " + mgBranchEP + " failed (exit status 1: gh: Server Error (HTTP 502)), so the " +
				"protection of " + mgSubject + ", is unknown", []string{mgRepoEP, mgRulesEP, mgBranchEP}, "github.com"},
		{"branch answer is not JSON", func(e *mgExec, _ *MergeGuaranteeParams) { e.replies[mgBranchEP] = mgReply{out: "[1]"} },
			mgBranchEP + " printed output that is not a branch's JSON", []string{mgRepoEP, mgRulesEP, mgBranchEP}, "github.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMGExec()
			p := mgParams(e)
			tc.setup(e, &p)
			looked := 0
			lookPath := p.LookPath
			p.LookPath = func(file string) (string, error) {
				looked++
				if file != "gh" {
					t.Errorf("LookPath(%q), want gh", file)
				}
				return lookPath(file)
			}
			c := CheckMergeGuarantee(context.Background(), p)
			if c.Kind != MergeGuaranteeUnknown || !strings.Contains(c.Detail, tc.detail) {
				t.Fatalf("check = %s %q\nwant unknown with %q", c.Kind, c.Detail, tc.detail)
			}
			if strings.ContainsAny(c.Detail, "\n\r") {
				t.Errorf("detail %q is not one line", c.Detail)
			}
			if got := e.endpoints(); !slices.Equal(got, tc.endpoints) {
				t.Errorf("gh endpoints = %q, want %q", got, tc.endpoints)
			}
			if wantLooked := map[bool]int{true: 1, false: 0}[tc.host != ""]; looked != wantLooked {
				t.Errorf("LookPath ran %d times, want %d", looked, wantLooked)
			}
			if c.Host != tc.host || tc.host != "" && c.Repo != "acme/app" || tc.host == "" && c.Repo != "" {
				t.Errorf("host, repo = %q, %q, want %q and acme/app when the host is read", c.Host, c.Repo, tc.host)
			}
			if c.Version != MergeGuaranteeVersion || c.CheckedAt != mgCheckedAt {
				t.Errorf("version, checked_at = %d, %q", c.Version, c.CheckedAt)
			}
			e.assertGHCalls(t, cmpHost(tc.host))
		})
	}
}

// cmpHost is the --hostname of a test's gh calls: host, or github.com when
// the test reads none (and so runs no gh).
func cmpHost(host string) string {
	if host == "" {
		return "github.com"
	}
	return host
}

// TestMergeGuarantee_LongErrorIsCutToOneLine: a command's error is quoted on
// one line and cut on a rune boundary.
func TestMergeGuarantee_LongErrorIsCutToOneLine(t *testing.T) {
	long := "exit status 1:\n" + strings.Repeat("é", 400)
	got := mergeGuaranteeErrText(errors.New(long))
	if strings.Contains(got, "\n") || !strings.HasSuffix(got, "...") || len(got) > mergeGuaranteeMaxErrText+3 {
		t.Errorf("text = %q (%d bytes)", got, len(got))
	}
	if !strings.HasPrefix(got, "exit status 1: é") || !utf8.ValidString(got) {
		t.Errorf("text is not cut on a rune boundary: %q", got)
	}
	if got := mergeGuaranteeErrText(errors.New("  a\tb  ")); got != "a b" {
		t.Errorf("short text = %q, want %q", got, "a b")
	}
}

// mgFullExec is an mgExec whose check runs all four gh calls and ends in
// github_protection (a strict branch protection with a check).
func mgFullExec() *mgExec {
	e := newMGExec()
	e.replies[mgBranchEP] = mgReply{out: mgClassicRequires("ci")}
	e.replies[mgStrictEP] = mgReply{out: mgStrictJSON(true)}
	return e
}

// TestMergeGuarantee_OneDeadlineCoversEveryCall: git and every gh call run
// under the same deadline, p.Timeout from the start, 20 s when it is zero
// (C1).
func TestMergeGuarantee_OneDeadlineCoversEveryCall(t *testing.T) {
	for _, tc := range []struct {
		timeout, want time.Duration
	}{{5 * time.Second, 5 * time.Second}, {0, 20 * time.Second}, {-time.Second, 20 * time.Second}} {
		e := mgFullExec()
		p := mgParams(e)
		p.Timeout = tc.timeout
		start := time.Now()
		if c := CheckMergeGuarantee(context.Background(), p); c.Kind != MergeGuaranteeGitHubProtection {
			t.Fatalf("timeout %s: kind = %s (%s)", tc.timeout, c.Kind, c.Detail)
		}
		end := time.Now()
		e.mu.Lock()
		calls := slices.Clone(e.calls)
		e.mu.Unlock()
		if len(calls) != 5 {
			t.Fatalf("timeout %s: %d calls, want git and 4 gh", tc.timeout, len(calls))
		}
		for _, c := range calls {
			if !c.deadline.Equal(calls[0].deadline) {
				t.Errorf("timeout %s: %s %q has deadline %v, git had %v", tc.timeout, c.name, c.args, c.deadline, calls[0].deadline)
			}
		}
		if d := calls[0].deadline; d.Before(start.Add(tc.want)) || d.After(end.Add(tc.want)) {
			t.Errorf("timeout %s: deadline %v is not %s after the start (%v..%v)", tc.timeout, d, tc.want, start, end)
		}
	}
}

// TestMergeGuarantee_DeadlineExceededIsUnknown: a call that outlasts the
// deadline makes the check unknown, naming what ran, and the check returns
// then (AC4).
func TestMergeGuarantee_DeadlineExceededIsUnknown(t *testing.T) {
	for block, ran := range map[string]string{
		"git":      "git remote get-url origin",
		mgRepoEP:   "gh api --hostname github.com repos/acme/app",
		mgRulesEP:  "gh api --hostname github.com --paginate " + mgRulesEP,
		mgBranchEP: "gh api --hostname github.com " + mgBranchEP,
		mgStrictEP: "gh api --hostname github.com " + mgStrictEP,
	} {
		t.Run(block, func(t *testing.T) {
			e := mgFullExec()
			e.block = block
			p := mgParams(e)
			p.Timeout = 30 * time.Millisecond
			start := time.Now()
			c := CheckMergeGuarantee(context.Background(), p)
			want := "the check timed out after 30ms while " + ran + " ran, so the protection of origin's repository's " +
				"default branch is unknown: check the network and gh's access"
			if c.Kind != MergeGuaranteeUnknown || c.Detail != want {
				t.Errorf("check = %s %q\nwant unknown %q", c.Kind, c.Detail, want)
			}
			if elapsed := time.Since(start); elapsed > 10*time.Second {
				t.Errorf("the check took %s", elapsed)
			}
		})
	}
}

// TestMergeGuarantee_CallerContextDoneIsUnknown: the caller's ctx done
// before or while a step runs is unknown, with its cause.
func TestMergeGuarantee_CallerContextDoneIsUnknown(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("doctor stopped"))
	e := newMGExec()
	c := CheckMergeGuarantee(ctx, mgParams(e))
	want := "the check was interrupted (doctor stopped) while git remote get-url origin ran, so the protection of origin's " +
		"repository's default branch is unknown"
	if c.Kind != MergeGuaranteeUnknown || c.Detail != want {
		t.Errorf("check = %s %q\nwant unknown %q", c.Kind, c.Detail, want)
	}
	if len(e.callsOf("gh")) != 0 {
		t.Errorf("gh ran: %q", e.endpoints())
	}

	ctx, cancelRules := context.WithCancel(context.Background())
	defer cancelRules()
	e = newMGExec()
	e.block = mgRulesEP
	go func() {
		for len(e.endpoints()) < 2 {
			time.Sleep(time.Millisecond)
		}
		cancelRules()
	}()
	c = CheckMergeGuarantee(ctx, mgParams(e))
	if c.Kind != MergeGuaranteeUnknown || !strings.HasPrefix(c.Detail, "the check was interrupted (context canceled) while gh "+
		"api --hostname github.com --paginate "+mgRulesEP+" ran") {
		t.Errorf("check = %s %q", c.Kind, c.Detail)
	}
}

// TestMergeGuarantee_HostComesFromOriginsURL: the Host recorded is the URL's
// host as written, lowercased, without the user, the password and the port,
// and every gh call carries it as --hostname, ssh.github.com as github.com
// (C7, AC4).
func TestMergeGuarantee_HostComesFromOriginsURL(t *testing.T) {
	for _, tc := range []struct {
		origin, host, apiHost string
	}{
		{"https://GitHub.com/acme/app.git", "github.com", "github.com"},
		{"ssh://git@ghe.example.com:2222/acme/app.git", "ghe.example.com", "ghe.example.com"},
		{"git@ghe.example.com:acme/app.git", "ghe.example.com", "ghe.example.com"},
		{"ssh://git@ssh.github.com:443/acme/app.git", "ssh.github.com", "github.com"},
		{"git@SSH.GitHub.com:acme/app.git", "ssh.github.com", "github.com"},
		{"https://token:x-oauth-basic@GHE.example.com:8443/acme/app", "ghe.example.com", "ghe.example.com"},
		{"git://example.org/acme/app.git", "example.org", "example.org"},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			e := mgFullExec()
			e.origin = tc.origin + "\n"
			c := mgCheck(e)
			if c.Kind != MergeGuaranteeGitHubProtection || c.Host != tc.host || c.Repo != "acme/app" {
				t.Fatalf("check = %s host %q repo %q (%s), want github_protection host %q repo acme/app", c.Kind, c.Host, c.Repo,
					c.Detail, tc.host)
			}
			if n := len(e.callsOf("gh")); n != 4 {
				t.Errorf("%d gh calls, want 4", n)
			}
			e.assertGHCalls(t, tc.apiHost)
			git := e.callsOf("git")
			if len(git) != 1 || git[0].dir != "/repo" || !slices.Equal(git[0].env, []string{"GIT_TERMINAL_PROMPT=0"}) {
				t.Errorf("git calls = %+v, want one in /repo with GIT_TERMINAL_PROMPT=0", git)
			}
			if js := mgJSON(c); strings.Contains(js, "token") || strings.Contains(js, "oauth") {
				t.Errorf("the check holds origin's credentials: %s", js)
			}
		})
	}
}

// TestMergeGuarantee_ReadsTheHostWithTheOwnerAndRepo: mergeOriginFromURL
// reads the host, and nothing from a URL mergeRepoFromOriginURL rejects.
func TestMergeGuarantee_ReadsTheHostWithTheOwnerAndRepo(t *testing.T) {
	for in, want := range map[string]string{
		"git@github.com:acme/app.git":             "github.com acme/app",
		"ssh://git@[::1]:22/acme/app.git":         "::1 acme/app",
		"HTTPS://user@Example.COM:8443/acme/app/": "example.com acme/app",
		"https://github.com/acme":                 "",
		"git@:acme/app":                           "",
		"/abs/acme/app.git":                       "",
		"file:///acme/app":                        "",
		"git+ssh://git@github.com/acme/app.git":   "",
	} {
		host, repo, ok := mergeOriginFromURL(in)
		got := ""
		if ok {
			got = host + " " + repo.String()
		} else if host != "" || repo != (mergeRepo{}) {
			t.Errorf("%q: not ok but host %q repo %q", in, host, repo)
		}
		if got != want {
			t.Errorf("%q = %q, want %q", in, got, want)
		}
	}
}

// TestMergeGuarantee_PaginatedRulesAreAllRead: --paginate output is read as
// one array or as one array per page, and the rules of every page count.
func TestMergeGuarantee_PaginatedRulesAreAllRead(t *testing.T) {
	for _, tc := range []struct {
		name, rules, kind string
	}{
		{"one array", mgRules(mgDeletionRule, mgChecksRule(true, "ci")), MergeGuaranteeGitHubProtection},
		{"one array per page", mgRules(mgDeletionRule) + mgRules(mgChecksRule(true, "ci")), MergeGuaranteeGitHubProtection},
		{"pages on their own lines", mgRules(mgReviewRule) + "\n" + mgRules(mgChecksRule(true, "ci")) + "\n", MergeGuaranteeGitHubProtection},
		{"merge queue and checks on two pages", mgRules(mgMergeQueueRule) + mgRules(mgChecksRule(false, "ci")), MergeGuaranteeGitHubProtection},
		{"an empty page first", "[]" + mgRules(mgChecksRule(true, "ci")), MergeGuaranteeGitHubProtection},
		{"a later page that is not a list", mgRules(mgReviewRule) + `{"message":"x"}`, MergeGuaranteeUnknown},
		{"a cut page", mgRules(mgReviewRule) + `[{"type":`, MergeGuaranteeUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMGExec()
			e.replies[mgRulesEP] = mgReply{out: tc.rules}
			if c := mgCheck(e); c.Kind != tc.kind {
				t.Errorf("kind = %s (%s), want %s", c.Kind, c.Detail, tc.kind)
			}
		})
	}
}

// TestMergeGuarantee_RenamedRepositoryStillDecides: a full_name other than
// origin's owner/repo is recorded as APIRepo and named in the Detail, and the
// check still decides, reading origin's owner/repo.
func TestMergeGuarantee_RenamedRepositoryStillDecides(t *testing.T) {
	e := newMGExec()
	e.replies[mgRepoEP] = mgReply{out: mgRepoJSON("main", "acme/core")}
	e.replies[mgRulesEP] = mgReply{out: mgRules(mgChecksRule(true, "ci"))}
	c := mgCheck(e)
	if c.Kind != MergeGuaranteeGitHubProtection || c.Repo != "acme/app" || c.APIRepo != "acme/core" {
		t.Fatalf("check = %s repo %q api_repo %q (%s)", c.Kind, c.Repo, c.APIRepo, c.Detail)
	}
	if !strings.HasPrefix(c.Detail, mgSubject+" (GitHub now names it acme/core), requires its 1 required status check") {
		t.Errorf("detail = %q", c.Detail)
	}
	if got := e.endpoints(); !slices.Equal(got, []string{mgRepoEP, mgRulesEP}) {
		t.Errorf("gh endpoints = %q", got)
	}

	e = newMGExec()
	e.replies[mgRepoEP] = mgReply{out: mgRepoJSON("main", "Acme/App")}
	if c := mgCheck(e); c.APIRepo != "Acme/App" || strings.Contains(c.Detail, "GitHub now names it") {
		t.Errorf("a full_name that differs only in case: api_repo %q, detail %q", c.APIRepo, c.Detail)
	}
}

// TestMergeGuarantee_EscapesTheBranchAndRepositoryInAPIPaths: the default
// branch, owner and repository are path-escaped in every gh path, so a "/"
// in a branch stays in it and gh fills no {owner} placeholder.
func TestMergeGuarantee_EscapesTheBranchAndRepositoryInAPIPaths(t *testing.T) {
	e := newMGExec()
	e.origin = "https://ghe.example.com/{owner}/app\n"
	const repo = "repos/%7Bowner%7D/app"
	e.replies = map[string]mgReply{}
	e.replies[repo] = mgReply{out: mgRepoJSON("release/v1", "{owner}/app")}
	e.replies[repo+"/rules/branches/release%2Fv1"] = mgReply{out: "[]"}
	e.replies[repo+"/branches/release%2Fv1"] = mgReply{out: mgClassicRequires("ci")}
	e.replies[repo+"/branches/release%2Fv1/protection/required_status_checks"] = mgReply{out: mgStrictJSON(true)}
	c := mgCheck(e)
	if c.Kind != MergeGuaranteeGitHubProtection || c.DefaultBranch != "release/v1" || c.Repo != "{owner}/app" {
		t.Fatalf("check = %s branch %q repo %q (%s)", c.Kind, c.DefaultBranch, c.Repo, c.Detail)
	}
	if n := len(e.callsOf("gh")); n != 4 {
		t.Errorf("%d gh calls, want 4: %q", n, e.endpoints())
	}
	e.assertGHCalls(t, "ghe.example.com")
}

// mgStateDir is an existing state dir under a temp dir.
func mgStateDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "state")
	mkdirAll(t, dir)
	return dir
}

// mgReadParams reads with e's git and the clock at now.
func mgReadParams(e *mgExec, now time.Time) MergeGuaranteeReadParams {
	return MergeGuaranteeReadParams{Dir: "/repo", Exec: e.run, Now: func() time.Time { return now }}
}

// mgProtected is a github_protection check of acme/app's main, checked at
// mgClock.
func mgProtected() MergeGuaranteeCheck {
	e := newMGExec()
	e.replies[mgRulesEP] = mgReply{out: mgRules(mgChecksRule(true, "ci"))}
	return mgCheck(e)
}

// mgSave saves c into stateDir, failing the test unless it was written.
func mgSave(t *testing.T, stateDir string, c MergeGuaranteeCheck) {
	t.Helper()
	if s := SaveMergeGuarantee(stateDir, c); !s.Written || s.WriteErr != nil || s.NoLedger {
		t.Fatalf("save = %+v", s)
	}
}

// mgDirNames are the names in dir.
func mgDirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// TestMergeGuarantee_SaveThenReadRoundTrips: a saved check reads back with
// its kind and fields through a temporary file that does not remain, and
// git runs once, in Dir; a second save replaces the first (AC5, AC6).
func TestMergeGuarantee_SaveThenReadRoundTrips(t *testing.T) {
	stateDir := mgStateDir(t)
	e := newMGExec()
	e.replies[mgRulesEP] = mgReply{out: mgRules(mgChecksRule(true, "ci"))}
	p := mgParams(e)
	p.Now = func() time.Time { return mgClock.In(time.FixedZone("JST", 9*3600)) }
	c := CheckMergeGuarantee(context.Background(), p)
	if c.CheckedAt != mgCheckedAt {
		t.Fatalf("checked_at = %q, want UTC %q", c.CheckedAt, mgCheckedAt)
	}
	s := SaveMergeGuarantee(stateDir, c)
	if want := (MergeGuaranteeSave{Path: filepath.Join(stateDir, "merge-guarantee.json"), Written: true}); s != want {
		t.Fatalf("save = %+v, want %+v", s, want)
	}
	if names := mgDirNames(t, stateDir); !slices.Equal(names, []string{"merge-guarantee.json"}) {
		t.Errorf("state dir holds %q, want the cache only", names)
	}
	data, err := os.ReadFile(MergeGuaranteePathIn(stateDir))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"version": float64(1), "kind": "github_protection", "detail": c.Detail, "host": "github.com", "repo": "acme/app",
		"api_repo": "acme/app", "default_branch": "main", "checked_at": mgCheckedAt,
	}
	if mgJSON(raw) != mgJSON(want) {
		t.Errorf("cache = %s\nwant %s", mgJSON(raw), mgJSON(want))
	}

	read := newMGExec()
	read.replies = nil
	g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(read, mgClock.Add(time.Hour)))
	wantG := MergeGuarantee{
		Kind: MergeGuaranteeGitHubProtection, Detail: c.Detail, Host: "github.com", Repo: "acme/app", APIRepo: "acme/app",
		DefaultBranch: "main", CheckedAt: mgCheckedAt,
	}
	if g != wantG {
		t.Fatalf("read = %+v\nwant %+v", g, wantG)
	}
	if text := g.Text(); text != "github_protection (main of github.com/acme/app, checked 2026-10-14T05:18:40Z): "+c.Detail {
		t.Errorf("text = %q", text)
	}
	git := read.callsOf("git")
	if len(git) != 1 || git[0].dir != "/repo" || !slices.Equal(git[0].env, []string{"GIT_TERMINAL_PROMPT=0"}) || len(read.callsOf("gh")) != 0 {
		t.Errorf("read ran %+v, want one git remote get-url origin in /repo and no gh", read.calls)
	}

	e.replies[mgRulesEP] = mgReply{out: "[]"}
	mgSave(t, stateDir, mgCheck(e))
	if g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock)); g.Kind != MergeGuaranteePRChecksOnly {
		t.Errorf("after the second save, kind = %s", g.Kind)
	}
}

// TestMergeGuarantee_SaveWritesAnUnknownToo: an unknown check is written,
// and reads back as the cache's unknown with its Detail; one that read no
// origin reads as origin_changed once origin names a repository (C5).
func TestMergeGuarantee_SaveWritesAnUnknownToo(t *testing.T) {
	stateDir := mgStateDir(t)
	p := mgParams(newMGExec())
	p.LookPath = mgNotFound
	c := CheckMergeGuarantee(context.Background(), p)
	mgSave(t, stateDir, c)
	g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock))
	if g.Kind != MergeGuaranteeUnknown || g.Reason != "" || g.Detail != c.Detail || g.LastKind != "" {
		t.Fatalf("read = %+v", g)
	}
	if want := "unknown (github.com/acme/app, checked 2026-10-14T05:18:40Z): " + c.Detail; g.Text() != want {
		t.Errorf("text = %q\nwant %q", g.Text(), want)
	}

	e := newMGExec()
	e.originErr = errors.New("exit status 2: error: No such remote 'origin'")
	mgSave(t, stateDir, CheckMergeGuarantee(context.Background(), mgParams(e)))
	g = ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock))
	if want := "unknown (doctor checked no readable origin; origin is now github.com/acme/app)"; g.Reason != MergeGuaranteeReasonOriginChanged || g.Text() != want {
		t.Errorf("read = %+v, text %q, want origin_changed %q", g, g.Text(), want)
	}
}

// TestMergeGuarantee_SaveWithoutALedgerDirCreatesNothing: a state dir that
// does not exist, or is not a directory, gets no cache and is not created
// (C2, AC5).
func TestMergeGuarantee_SaveWithoutALedgerDirCreatesNothing(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "org", "state")
	s := SaveMergeGuarantee(missing, mgProtected())
	if want := (MergeGuaranteeSave{Path: filepath.Join(missing, "merge-guarantee.json"), NoLedger: true}); s != want {
		t.Errorf("save = %+v, want %+v", s, want)
	}
	if _, err := os.Stat(filepath.Join(root, "org")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the save created %s: %v", filepath.Join(root, "org"), err)
	}

	file := filepath.Join(root, "state-file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if s := SaveMergeGuarantee(file, mgProtected()); !s.NoLedger || s.Written || s.WriteErr != nil {
		t.Errorf("save into a file = %+v, want no ledger", s)
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != "x" {
		t.Errorf("the file became %q (%v)", data, err)
	}
}

// TestMergeGuarantee_WriteFailureRemovesTheOldCache: when the write fails,
// the old cache is removed, so it no longer reads as github_protection; with
// no old cache nothing is removed (C5, AC5, AC10).
func TestMergeGuarantee_WriteFailureRemovesTheOldCache(t *testing.T) {
	stateDir := mgStateDir(t)
	mgSave(t, stateDir, mgProtected())
	writeErr := errors.New("org: write merge-guarantee.json: no space left on device")
	failing := func(string, any) error { return writeErr }
	s := saveMergeGuarantee(stateDir, mgProtected(), failing)
	if !errors.Is(s.WriteErr, writeErr) || !s.Removed || s.RemoveErr != nil || s.Written || s.NoLedger {
		t.Fatalf("save = %+v, want the write error and the old cache removed", s)
	}
	if _, err := os.Stat(MergeGuaranteePathIn(stateDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the old cache remains: %v", err)
	}
	if g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock)); g.Reason != MergeGuaranteeReasonNotChecked {
		t.Errorf("read after the failure = %+v, want not_checked", g)
	}

	s = saveMergeGuarantee(stateDir, mgProtected(), failing)
	if !errors.Is(s.WriteErr, writeErr) || s.Removed || s.RemoveErr != nil {
		t.Errorf("save with no old cache = %+v, want the write error and nothing removed", s)
	}

	// The real write fails on a directory at the cache's path; an empty one
	// is removed.
	mkdirAll(t, MergeGuaranteePathIn(stateDir))
	s = SaveMergeGuarantee(stateDir, mgProtected())
	if s.WriteErr == nil || !s.Removed || s.RemoveErr != nil || s.Written {
		t.Errorf("save over an empty directory = %+v, want the write error and the directory removed", s)
	}
	if names := mgDirNames(t, stateDir); len(names) != 0 {
		t.Errorf("state dir holds %q after the failed write, want nothing", names)
	}
}

// TestMergeGuarantee_RemovalFailureIsReported: when the write fails and the
// old cache cannot be removed either, the save says so, and the old cache
// still reads (doctor warns about it).
func TestMergeGuarantee_RemovalFailureIsReported(t *testing.T) {
	stateDir := mgStateDir(t)
	cache := MergeGuaranteePathIn(stateDir)
	mkdirAll(t, filepath.Join(cache, "held"))
	s := SaveMergeGuarantee(stateDir, mgProtected())
	if s.WriteErr == nil || s.RemoveErr == nil || s.Removed || s.Written {
		t.Fatalf("save over a non-empty directory = %+v, want a write and a remove error", s)
	}
	if !strings.Contains(s.RemoveErr.Error(), "org: remove "+cache) {
		t.Errorf("remove error = %v", s.RemoveErr)
	}
	if names := mgDirNames(t, stateDir); !slices.Equal(names, []string{"merge-guarantee.json"}) {
		t.Errorf("state dir holds %q, want no temporary file", names)
	}

	if os.Geteuid() == 0 {
		t.Skip("root writes into a read-only directory")
	}
	stateDir = mgStateDir(t)
	mgSave(t, stateDir, mgProtected())
	if err := os.Chmod(stateDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(stateDir, 0o755) })
	s = SaveMergeGuarantee(stateDir, mgProtected())
	if s.WriteErr == nil || s.RemoveErr == nil || s.Removed {
		t.Fatalf("save into a read-only state dir = %+v, want a write and a remove error", s)
	}
	if g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock)); g.Kind != MergeGuaranteeGitHubProtection {
		t.Errorf("the old cache reads as %+v", g)
	}
}

// TestMergeGuarantee_ReadWithoutAUsableCacheIsUnknown: no state dir, no
// cache, and a cache that cannot be read, does not parse, or has another
// version or kind are unknown with their reason, and git does not run
// (AC6).
func TestMergeGuarantee_ReadWithoutAUsableCacheIsUnknown(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, root string) string // returns the state dir
		reason string
		text   string
	}{
		{"no state dir", func(_ *testing.T, root string) string { return filepath.Join(root, "missing") },
			MergeGuaranteeReasonNoLedger, "unknown (no org ledger yet)"},
		{"state dir is a file", func(t *testing.T, root string) string {
			f := filepath.Join(root, "file")
			if err := os.WriteFile(f, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			return f
		}, MergeGuaranteeReasonNoLedger, "unknown (no org ledger yet)"},
		{"no cache", func(t *testing.T, root string) string { return root }, MergeGuaranteeReasonNotChecked,
			"unknown (not checked: run ralph doctor)"},
		{"corrupt cache", mgWriteRawCache(`{"version":1,"kind":`), MergeGuaranteeReasonUnreadable,
			"unknown (cache unreadable: run ralph doctor)"},
		{"another version", mgWriteRawCache(`{"version":2,"kind":"github_protection","checked_at":"` + mgCheckedAt + `"}`),
			MergeGuaranteeReasonUnreadable, "unknown (cache unreadable: run ralph doctor)"},
		{"no version", mgWriteRawCache(`{"kind":"github_protection","checked_at":"` + mgCheckedAt + `"}`),
			MergeGuaranteeReasonUnreadable, "unknown (cache unreadable: run ralph doctor)"},
		{"another kind", mgWriteRawCache(`{"version":1,"kind":"protected","checked_at":"` + mgCheckedAt + `"}`),
			MergeGuaranteeReasonUnreadable, "unknown (cache unreadable: run ralph doctor)"},
		{"cache is a directory", func(t *testing.T, root string) string {
			mkdirAll(t, MergeGuaranteePathIn(root))
			return root
		}, MergeGuaranteeReasonUnreadable, "unknown (cache unreadable: run ralph doctor)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMGExec()
			g := ReadMergeGuarantee(context.Background(), tc.setup(t, t.TempDir()), mgReadParams(e, mgClock))
			if want := (MergeGuarantee{Kind: MergeGuaranteeUnknown, Reason: tc.reason, Detail: strings.TrimSuffix(
				strings.TrimPrefix(tc.text, "unknown ("), ")")}); g != want {
				t.Errorf("read = %+v, want %+v", g, want)
			}
			if g.Text() != tc.text {
				t.Errorf("text = %q, want %q", g.Text(), tc.text)
			}
			if len(e.calls) != 0 {
				t.Errorf("read ran %+v", e.calls)
			}
		})
	}
}

// mgWriteRawCache is a setup that writes data as the cache in root.
func mgWriteRawCache(data string) func(t *testing.T, root string) string {
	return func(t *testing.T, root string) string {
		t.Helper()
		if err := os.WriteFile(MergeGuaranteePathIn(root), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}
}

// TestMergeGuarantee_ReadMakesAnOldCacheStale: a cache checked more than
// MergeGuaranteeMaxAge before now (or after it), or whose checked_at does
// not parse, is unknown (stale) with its last kind, and git does not run; at
// MergeGuaranteeMaxAge or less it is used (C5, AC10).
func TestMergeGuarantee_ReadMakesAnOldCacheStale(t *testing.T) {
	staleText := "unknown (stale: last github_protection at 2026-10-14T05:18:40Z; run ralph doctor)"
	for _, tc := range []struct {
		name      string
		now       time.Time
		checkedAt string // "" keeps mgCheckedAt
		stale     bool
		text      string
	}{
		{"just under the max age", mgClock.Add(MergeGuaranteeMaxAge - time.Second), "", false, ""},
		{"at the max age", mgClock.Add(MergeGuaranteeMaxAge), "", false, ""},
		{"just over the max age", mgClock.Add(MergeGuaranteeMaxAge + time.Second), "", true, staleText},
		{"days later", mgClock.Add(72 * time.Hour), "", true, staleText},
		{"checked over the max age after now", mgClock.Add(-MergeGuaranteeMaxAge - time.Second), "", true, staleText},
		{"checked_at does not parse", mgClock, "yesterday", true,
			"unknown (stale: last github_protection at yesterday; run ralph doctor)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := mgStateDir(t)
			c := mgProtected()
			if tc.checkedAt != "" {
				c.CheckedAt = tc.checkedAt
			}
			mgSave(t, stateDir, c)
			e := newMGExec()
			g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(e, tc.now))
			if !tc.stale {
				if g.Kind != MergeGuaranteeGitHubProtection || g.Reason != "" || len(e.callsOf("git")) != 1 {
					t.Errorf("read = %+v with %d git calls, want the cache's kind after one", g, len(e.callsOf("git")))
				}
				return
			}
			if g.Kind != MergeGuaranteeUnknown || g.Reason != MergeGuaranteeReasonStale || g.LastKind != MergeGuaranteeGitHubProtection ||
				g.CheckedAt != c.CheckedAt || g.DefaultBranch != "main" || g.Repo != "acme/app" {
				t.Errorf("read = %+v, want stale with the cache's fields", g)
			}
			if g.Text() != tc.text {
				t.Errorf("text = %q, want %q", g.Text(), tc.text)
			}
			if len(e.calls) != 0 {
				t.Errorf("a stale cache ran %+v", e.calls)
			}
		})
	}
}

// TestMergeGuarantee_ReadComparesOriginWithTheCache: origin's host and
// owner/repo now are compared with the cache's without case; another host or
// repository is origin_changed, an origin that cannot be read here
// origin_unreadable, both with the last kind (C7, AC6, AC10).
func TestMergeGuarantee_ReadComparesOriginWithTheCache(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(e *mgExec, p *MergeGuaranteeReadParams, cancel context.CancelFunc)
		reason string
		text   string
	}{
		{"same origin in another case", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.origin = "git@GitHub.com:Acme/App.git\n"
		}, "", ""},
		{"same origin over https", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.origin = "https://github.com/acme/app\n"
		}, "", ""},
		{"another repository with the same branch", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.origin = "git@github.com:acme/other.git\n"
		}, MergeGuaranteeReasonOriginChanged, "unknown (doctor checked github.com/acme/app; origin is now github.com/acme/other)"},
		{"another owner", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.origin = "git@github.com:fork/app.git\n"
		}, MergeGuaranteeReasonOriginChanged, "unknown (doctor checked github.com/acme/app; origin is now github.com/fork/app)"},
		{"another host", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.origin = "git@ghe.example.com:acme/app.git\n"
		}, MergeGuaranteeReasonOriginChanged, "unknown (doctor checked github.com/acme/app; origin is now ghe.example.com/acme/app)"},
		{"ssh.github.com is another host as written", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.origin = "ssh://git@ssh.github.com:443/acme/app.git\n"
		}, MergeGuaranteeReasonOriginChanged, "unknown (doctor checked github.com/acme/app; origin is now ssh.github.com/acme/app)"},
		{"get-url fails", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.originErr = errors.New("exit status 128: fatal: not a git repository")
		}, MergeGuaranteeReasonOriginUnreadable, "unknown (origin cannot be read here (git remote get-url origin failed: exit " +
			"status 128: fatal: not a git repository); doctor checked github.com/acme/app)"},
		{"origin is a local path", func(e *mgExec, _ *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.origin = "/srv/git/app.git\n"
		}, MergeGuaranteeReasonOriginUnreadable, "unknown (origin cannot be read here (its URL names no host and owner/repo); " +
			"doctor checked github.com/acme/app)"},
		{"get-url times out", func(e *mgExec, p *MergeGuaranteeReadParams, _ context.CancelFunc) {
			e.block = "git"
			p.GitTimeout = 20 * time.Millisecond
		}, MergeGuaranteeReasonOriginUnreadable, "unknown (origin cannot be read here (git remote get-url origin timed out " +
			"after 20ms); doctor checked github.com/acme/app)"},
		{"caller's ctx done", func(_ *mgExec, _ *MergeGuaranteeReadParams, cancel context.CancelFunc) { cancel() },
			MergeGuaranteeReasonOriginUnreadable, "unknown (origin cannot be read here (interrupted (context canceled) while " +
				"git remote get-url origin ran); doctor checked github.com/acme/app)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := mgStateDir(t)
			mgSave(t, stateDir, mgProtected())
			e := newMGExec()
			p := mgReadParams(e, mgClock)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tc.setup(e, &p, cancel)
			g := ReadMergeGuarantee(ctx, stateDir, p)
			if tc.reason == "" {
				if g.Kind != MergeGuaranteeGitHubProtection || g.Reason != "" {
					t.Errorf("read = %+v, want the cache's kind", g)
				}
				return
			}
			if g.Kind != MergeGuaranteeUnknown || g.Reason != tc.reason || g.LastKind != MergeGuaranteeGitHubProtection ||
				g.Host != "github.com" || g.Repo != "acme/app" {
				t.Errorf("read = %+v, want %s with the cache's fields", g, tc.reason)
			}
			if g.Text() != tc.text {
				t.Errorf("text = %q\nwant %q", g.Text(), tc.text)
			}
		})
	}
}

// TestMergeGuarantee_ReadBoundsGitWithItsTimeout: the read's git runs with
// GitTimeout, 10 s when it is zero.
func TestMergeGuarantee_ReadBoundsGitWithItsTimeout(t *testing.T) {
	stateDir := mgStateDir(t)
	mgSave(t, stateDir, mgProtected())
	for timeout, want := range map[time.Duration]time.Duration{3 * time.Second: 3 * time.Second, 0: 10 * time.Second} {
		e := newMGExec()
		p := mgReadParams(e, mgClock)
		p.GitTimeout = timeout
		start := time.Now()
		ReadMergeGuarantee(context.Background(), stateDir, p)
		end := time.Now()
		git := e.callsOf("git")
		if len(git) != 1 {
			t.Fatalf("timeout %s: git calls = %+v", timeout, git)
		}
		if d := git[0].deadline; d.Before(start.Add(want)) || d.After(end.Add(want)) {
			t.Errorf("timeout %s: deadline %v is not %s after the start", timeout, d, want)
		}
	}
}

// TestMergeGuarantee_ForDefaultBranch: the cache's kind for another default
// branch than the one doctor checked is unknown (default_branch_changed);
// the same branch, an empty one, an unknown with a reason, and a cache with
// no branch are kept (AC7).
func TestMergeGuarantee_ForDefaultBranch(t *testing.T) {
	stateDir := mgStateDir(t)
	mgSave(t, stateDir, mgProtected())
	g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock))
	if got := g.ForDefaultBranch("main"); got != g {
		t.Errorf("same branch: %+v", got)
	}
	if got := g.ForDefaultBranch(""); got != g {
		t.Errorf("no branch: %+v", got)
	}
	got := g.ForDefaultBranch("trunk")
	if got.Kind != MergeGuaranteeUnknown || got.Reason != MergeGuaranteeReasonDefaultBranchChanged ||
		got.LastKind != MergeGuaranteeGitHubProtection || got.DefaultBranch != "main" {
		t.Errorf("another branch: %+v", got)
	}
	if want := "unknown (doctor checked main; origin's default branch is now trunk)"; got.Text() != want {
		t.Errorf("text = %q, want %q", got.Text(), want)
	}
	stale := g.unknownFor(MergeGuaranteeReasonStale, "stale: x")
	if again := stale.ForDefaultBranch("trunk"); again != stale {
		t.Errorf("an unknown with a reason: %+v", again)
	}
	noBranch := MergeGuarantee{Kind: MergeGuaranteeUnknown, Detail: "gh is not on PATH", Host: "github.com", Repo: "acme/app"}
	if again := noBranch.ForDefaultBranch("trunk"); again != noBranch {
		t.Errorf("a cache with no branch: %+v", again)
	}
}

// TestMergeGuarantee_DisplayJSONShape: the display's JSON keys, with the
// empty ones left out.
func TestMergeGuarantee_DisplayJSONShape(t *testing.T) {
	keys := func(g MergeGuarantee) []string {
		var m map[string]any
		if err := json.Unmarshal([]byte(mgJSON(g)), &m); err != nil {
			t.Fatal(err)
		}
		var out []string
		for k := range m {
			out = append(out, k)
		}
		slices.Sort(out)
		return out
	}
	stateDir := mgStateDir(t)
	mgSave(t, stateDir, mgProtected())
	g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock))
	if got := keys(g); !slices.Equal(got, []string{"api_repo", "checked_at", "default_branch", "detail", "host", "kind", "repo"}) {
		t.Errorf("cached kind keys = %q", got)
	}
	if got := keys(g.ForDefaultBranch("trunk")); !slices.Equal(got, []string{"api_repo", "checked_at", "default_branch",
		"detail", "host", "kind", "last_kind", "reason", "repo"}) {
		t.Errorf("unknown from the cache keys = %q", got)
	}
	none := ReadMergeGuarantee(context.Background(), filepath.Join(stateDir, "missing"), mgReadParams(newMGExec(), mgClock))
	if got := keys(none); !slices.Equal(got, []string{"detail", "kind", "reason"}) {
		t.Errorf("no ledger keys = %q", got)
	}
}
