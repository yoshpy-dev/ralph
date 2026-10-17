package org

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// CheckMergeEnforcement (plan docs/plans/active/2026-10-17-org-auto-merge.md,
// D11 and AC5) with mgExec, the fake exec of mergeguarantee_test.go: a GraphQL
// call is keyed by its endpoint, graphql, its last argument as in every gh api
// call. No test runs gh or reaches a network.

// The endpoints CheckMergeEnforcement reads beside mgRepoEP, mgRulesEP,
// mgBranchEP and mgStrictEP.
const (
	meQueueEP    = "graphql"
	meRuleset1EP = "repos/acme/app/rulesets/1"
	meRuleset2EP = "repos/acme/app/rulesets/2"
	meEnforceEP  = "repos/acme/app/branches/main/protection/enforce_admins"
)

// meFix is mergeEnforceFix as the tests expect it; meAccess ends the Detail
// of a read that failed.
const (
	meFix = "add a ruleset to the branch with Require branches to be up to date before merging and at least one required " +
		"status check, with no bypass for whoever runs ralph org merge"
	meAccess = ": check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com)"
)

// meRepoJSON is a repos/<o>/<r> answer with def, fullName, and the value
// admin as permissions.admin (left out when nil; the permissions object too
// when noPermissions).
func meRepoJSON(def, fullName string, admin any, noPermissions bool) string {
	repo := map[string]any{"id": 1, "full_name": fullName, "default_branch": def}
	if !noPermissions {
		perms := map[string]any{"pull": true, "push": true}
		if admin != nil {
			perms["admin"] = admin
		}
		repo["permissions"] = perms
	}
	return mgJSON(repo)
}

// meChecksRule is mgChecksRule of the ruleset id (0 leaves ruleset_id out).
func meChecksRule(id int, strict bool, contexts ...string) string {
	var rule map[string]any
	if err := json.Unmarshal([]byte(mgChecksRule(strict, contexts...)), &rule); err != nil {
		panic(err)
	}
	delete(rule, "ruleset_id")
	if id != 0 {
		rule["ruleset_id"] = id
	}
	return mgJSON(rule)
}

// meRuleset is a rulesets/<id> answer named name whose
// current_user_can_bypass is bypass (left out when nil).
func meRuleset(id int, name string, bypass any) string {
	rs := map[string]any{"id": id, "name": name, "target": "branch", "source_type": "Repository", "source": "acme/app",
		"enforcement": "active"}
	if bypass != nil {
		rs["current_user_can_bypass"] = bypass
	}
	return mgJSON(rs)
}

// meQueueJSON is a GraphQL answer whose repository.mergeQueue is queue, as
// JSON.
func meQueueJSON(queue string) string {
	return `{"data":{"repository":{"mergeQueue":` + queue + `}}}`
}

// newMEExec is an mgExec for acme/app on github.com whose default branch main
// has no merge queue and one ruleset, id 1, with an up-to-date rule requiring
// ci that the caller, no admin, cannot bypass.
func newMEExec() *mgExec {
	e := newMGExec()
	e.replies[mgRepoEP] = mgReply{out: meRepoJSON("main", "acme/app", false, false)}
	e.replies[meQueueEP] = mgReply{out: meQueueJSON("null")}
	e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(1, true, "ci"))}
	e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", "never")}
	return e
}

// meClassicExec is newMEExec with no rule, and a strict branch protection
// requiring ci, which the caller reads as an admin (admin) or not.
func meClassicExec(admin bool) *mgExec {
	e := newMEExec()
	e.replies[mgRepoEP] = mgReply{out: meRepoJSON("main", "acme/app", admin, false)}
	e.replies[mgRulesEP] = mgReply{out: "[]"}
	e.replies[mgBranchEP] = mgReply{out: mgClassicRequires("ci")}
	e.replies[mgStrictEP] = mgReply{out: mgStrictJSON(true)}
	return e
}

// meBool is b's address, for a table's optional bool (go.mod's Go has no
// new(expr)).
func meBool(b bool) *bool { return &b }

func meParams(e *mgExec) MergeEnforcementParams {
	return MergeEnforcementParams{Dir: "/repo", Exec: e.run, Host: "github.com", Owner: "acme", Name: "app", DefaultBranch: "main"}
}

// meAssertGHCalls fails the test unless every gh call is the exact argv
// CheckMergeEnforcement gives its endpoint, in /repo with
// GH_PROMPT_DISABLED=1: `api --hostname <host> [--paginate] <path>` (paginate
// on the rules only), and for graphql the query and the variables owner,
// name and branch, each with -f, before the endpoint.
func meAssertGHCalls(t *testing.T, e *mgExec, host, owner, name, branch string) {
	t.Helper()
	for _, c := range e.callsOf("gh") {
		ep := c.args[len(c.args)-1]
		want := []string{"api", "--hostname", host, ep}
		switch {
		case ep == meQueueEP:
			want = []string{"api", "--hostname", host, "-f", "query=" + mergeQueueQuery, "-f", "owner=" + owner, "-f", "name=" + name,
				"-f", "branch=" + branch, "graphql"}
		case strings.Contains(ep, "/rules/branches/"):
			want = []string{"api", "--hostname", host, "--paginate", ep}
		}
		if !slices.Equal(c.args, want) || c.dir != "/repo" || !slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) {
			t.Errorf("gh call = %q in %q with %q, want %q in /repo with GH_PROMPT_DISABLED=1", c.args, c.dir, c.env, want)
		}
	}
	if git := e.callsOf("git"); len(git) != 0 {
		t.Errorf("git ran: %+v", git)
	}
}

// TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn covers D11 and AC5: it
// is ok only when an up-to-date rule with checks of its own belongs to a
// ruleset the caller cannot bypass (never), or when a strict branch
// protection with checks applies to everyone (enforce_admins enabled,
// TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller); a merge
// queue (GraphQL or rules), a bypass of another value or none, a ruleset or
// other read that fails, and nothing up to date are not ok, each with a
// Detail that says why and what to do, and with the reads after the deciding
// one not run.
func TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn(t *testing.T) {
	ruleset := []string{mgRepoEP, meQueueEP, mgRulesEP, meRuleset1EP}
	classic := []string{mgRepoEP, meQueueEP, mgRulesEP, mgBranchEP, mgStrictEP}
	never := `the ruleset "protect main" (id 1) of ` + mgSubject + `, requires branches to be up to date before merging with its ` +
		`1 required status check, and you cannot bypass it (current_user_can_bypass: never)`
	bypassed := func(values string) string {
		return "every ruleset of " + mgSubject + ", that requires branches to be up to date before merging with required status " +
			"checks of its own can be bypassed by you (current_user_can_bypass of " + values + "), so GitHub would let you merge " +
			"a head that is not up to date: take whoever runs ralph org merge off the bypass list of one of them, or merge the PR " +
			"yourself"
	}
	notEnforced := func(note string) string {
		return mgSubject + ", does not make a PR's required status checks pass on a head with the latest main before it " +
			"merges: nothing requires branches to be up to date before merging with required status checks" + note + "; " + meFix +
			", or merge the PR yourself"
	}
	queue := func(how string) string {
		return mgSubject + ", has a merge queue (" + how + "): ralph org merge does not merge into a branch with a merge queue, so " +
			"merge the PR through the queue yourself"
	}
	queueUnknown := ", so whether " + mgSubject + ", has a merge queue is unknown" + meAccess
	const queueCmd = "gh api --hostname github.com graphql (mergeQueue of main)"
	for _, tc := range []struct {
		name      string
		classic   *bool // nil: newMEExec; else meClassicExec(*classic)
		setup     func(e *mgExec)
		ok        bool
		detail    string // the exact Detail, or a part of it with partial
		partial   bool
		endpoints []string
	}{
		{"ruleset that cannot be bypassed", nil, func(*mgExec) {}, true, never, false, ruleset},
		{"ruleset bypass always", nil, func(e *mgExec) { e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", "always")} },
			false, bypassed(`"protect main" (id 1): "always"`), false, ruleset},
		{"ruleset bypass pull_requests_only", nil, func(e *mgExec) {
			e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", "pull_requests_only")}
		}, false, bypassed(`"protect main" (id 1): "pull_requests_only"`), false, ruleset},
		{"ruleset bypass exempt", nil, func(e *mgExec) { e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", "exempt")} },
			false, bypassed(`"protect main" (id 1): "exempt"`), false, ruleset},
		{"ruleset bypass field missing", nil, func(e *mgExec) { e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", nil)} },
			false, bypassed(`"protect main" (id 1): "missing"`), false, ruleset},
		{"ruleset bypass null", nil, func(e *mgExec) {
			e.replies[meRuleset1EP] = mgReply{out: `{"id":1,"name":"protect main","current_user_can_bypass":null}`}
		}, false, bypassed(`"protect main" (id 1): "missing"`), false, ruleset},
		{"ruleset bypass never in capitals", nil, func(e *mgExec) { e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", "NEVER")} },
			false, bypassed(`"protect main" (id 1): "NEVER"`), false, ruleset},
		{"ruleset read fails", nil, func(e *mgExec) { e.replies[meRuleset1EP] = mgAPIError("Server Error", "500") }, false,
			"gh api --hostname github.com " + meRuleset1EP + " failed (exit status 1: gh: Server Error (HTTP 500)), so the protection " +
				"of " + mgSubject + ", is unknown" + meAccess, false, ruleset},
		{"ruleset answer is not JSON", nil, func(e *mgExec) { e.replies[meRuleset1EP] = mgReply{out: "<html>"} }, false,
			"gh api --hostname github.com " + meRuleset1EP + " printed output that is not a ruleset's JSON", true, ruleset},
		{"two strict rulesets, the second cannot be bypassed", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(1, true, "ci"), meChecksRule(2, true, "ci", "lint"))}
			e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", "always")}
			e.replies[meRuleset2EP] = mgReply{out: meRuleset(2, "release", "never")}
		}, true, `the ruleset "release" (id 2) of ` + mgSubject + `, requires branches to be up to date before merging with its 2 ` +
			`required status checks, and you cannot bypass it (current_user_can_bypass: never)`, false, append(slices.Clone(ruleset), meRuleset2EP)},
		{"two strict rulesets, both bypassed", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(1, true, "ci"), meChecksRule(2, true, "lint"))}
			e.replies[meRuleset1EP] = mgReply{out: meRuleset(1, "protect main", "always")}
			e.replies[meRuleset2EP] = mgReply{out: meRuleset(2, "release", "pull_requests_only")}
		}, false, bypassed(`"protect main" (id 1): "always", "release" (id 2): "pull_requests_only"`), false,
			append(slices.Clone(ruleset), meRuleset2EP)},
		{"two strict rulesets, one never and one that cannot be read", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(1, true, "ci"), meChecksRule(2, true, "lint"))}
			e.replies[meRuleset2EP] = mgAPIError("Not Found", "404")
		}, false, "gh api --hostname github.com " + meRuleset2EP + " failed (exit status 1: gh: Not Found (HTTP 404))", true,
			append(slices.Clone(ruleset), meRuleset2EP)},
		{"two up-to-date rules of one ruleset are read once", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(1, true, "ci"), meChecksRule(1, true, "ci", "lint"))}
		}, true, "requires branches to be up to date before merging with its 2 required status checks, and you cannot bypass it",
			true, ruleset},
		{"a ruleset that is not up to date does not count", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(2, false, "ci", "lint"), meChecksRule(1, true, "ci"))}
		}, true, never, false, ruleset},
		{"strict rule with no check of its own only", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(1, true), meChecksRule(2, false, "ci"))}
		}, false, notEnforced(" (the ruleset that requires branches to be up to date requires no status check itself)"), false,
			[]string{mgRepoEP, meQueueEP, mgRulesEP, mgBranchEP}},
		{"strict rule with checks and no ruleset_id", nil, func(e *mgExec) { e.replies[mgRulesEP] = mgReply{out: mgRules(meChecksRule(0, true, "ci"))} },
			false, "an up-to-date rule of " + mgSubject + ", with 1 required status check of its own names no ruleset_id, so whether " +
				"you can bypass it is unknown: " + meFix, false, []string{mgRepoEP, meQueueEP, mgRulesEP}},
		{"merge queue in GraphQL only", nil, func(e *mgExec) { e.replies[meQueueEP] = mgReply{out: meQueueJSON(`{"id":"MQ_kwDOA"}`)} },
			false, queue("GraphQL mergeQueue is set"), false, []string{mgRepoEP, meQueueEP}},
		{"merge queue in the rules only", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(mgMergeQueueRule, meChecksRule(1, true, "ci"))}
		}, false, queue("a ruleset's merge_queue rule"), false, []string{mgRepoEP, meQueueEP, mgRulesEP}},
		{"GraphQL fails", nil, func(e *mgExec) {
			e.replies[meQueueEP] = mgReply{out: `{"data":{"repository":null},"errors":[{"type":"NOT_FOUND"}]}`,
				err: errors.New("exit status 1: gh: Could not resolve to a Repository with the name 'acme/app'.")}
		}, false, queueCmd + " failed (exit status 1: gh: Could not resolve to a Repository with the name 'acme/app'.)" + queueUnknown,
			false, []string{mgRepoEP, meQueueEP}},
		{"GraphQL answers with errors", nil, func(e *mgExec) {
			e.replies[meQueueEP] = mgReply{out: `{"data":{"repository":{"mergeQueue":null}},"errors":[{"message":"x"}]}`}
		}, false, queueCmd + ` answered with errors ([{"message":"x"}])` + queueUnknown, false, []string{mgRepoEP, meQueueEP}},
		{"GraphQL answers with errors null", nil, func(e *mgExec) {
			e.replies[meQueueEP] = mgReply{out: `{"data":{"repository":{"mergeQueue":null}},"errors":null}`}
		}, true, never, false, ruleset},
		{"GraphQL answer is not JSON", nil, func(e *mgExec) { e.replies[meQueueEP] = mgReply{out: "<html>"} }, false,
			queueCmd + " printed output that is not GraphQL JSON (invalid character '<' looking for beginning of value)" + queueUnknown,
			false, []string{mgRepoEP, meQueueEP}},
		{"GraphQL answer has no repository", nil, func(e *mgExec) { e.replies[meQueueEP] = mgReply{out: `{"data":{"repository":null}}`} },
			false, queueCmd + " returned no repository.mergeQueue" + queueUnknown, false, []string{mgRepoEP, meQueueEP}},
		{"GraphQL answer has no mergeQueue", nil, func(e *mgExec) { e.replies[meQueueEP] = mgReply{out: `{"data":{"repository":{}}}`} },
			false, queueCmd + " returned no repository.mergeQueue" + queueUnknown, false, []string{mgRepoEP, meQueueEP}},
		// The strict classic protection's enforce_admins, for every caller:
		// TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller.
		{"classic strict that needs admin access (404)", meBool(false), func(e *mgExec) { e.replies[mgStrictEP] = mgAPIError("Not Found", "404") },
			false, "the branch protection of " + mgSubject + ", requires 1 required status check, but whether it requires branches to " +
				"be up to date before merging can be read only with admin access to the repository (gh api --hostname github.com " +
				mgStrictEP + " failed: exit status 1: gh: Not Found (HTTP 404)): " + meFix + " instead, which ralph org merge reads " +
				"without admin access", false, classic},
		{"classic strict answers 403", meBool(false), func(e *mgExec) { e.replies[mgStrictEP] = mgAPIError("Forbidden", "403") },
			false, "GitHub answered 403", true, classic},
		{"classic not strict", meBool(false), func(e *mgExec) { e.replies[mgStrictEP] = mgReply{out: mgStrictJSON(false)} }, false,
			notEnforced(" (the branch protection requires 1 required status check, but not up to date)"), false, classic},
		{"classic with no required check", meBool(false), func(e *mgExec) { e.replies[mgBranchEP] = mgReply{out: mgClassicRequires()} },
			false, notEnforced(""), false, []string{mgRepoEP, meQueueEP, mgRulesEP, mgBranchEP}},
		{"no rule and no branch protection", nil, func(e *mgExec) { e.replies[mgRulesEP] = mgReply{out: "[]"} }, false,
			notEnforced(""), false, []string{mgRepoEP, meQueueEP, mgRulesEP, mgBranchEP}},
		{"API default branch differs", nil, func(e *mgExec) {
			e.replies[mgRepoEP] = mgReply{out: meRepoJSON("trunk", "acme/app", false, false)}
		}, false, `gh api --hostname github.com repos/acme/app returned the default branch "trunk", but origin's HEAD names main, ` +
			"so which branch's protection to read is unknown: run ralph org merge-order again", false, []string{mgRepoEP}},
		{"repository read fails", nil, func(e *mgExec) { e.replies[mgRepoEP] = mgAPIError("Not Found", "404") }, false,
			"gh api --hostname github.com repos/acme/app failed (exit status 1: gh: Not Found (HTTP 404)), so the protection of " +
				mgSubject + ", is unknown" + meAccess, false, []string{mgRepoEP}},
		{"rules read fails", nil, func(e *mgExec) { e.replies[mgRulesEP] = mgAPIError("Server Error", "500") }, false,
			"gh api --hostname github.com --paginate " + mgRulesEP + " failed (exit status 1: gh: Server Error (HTTP 500)), so the " +
				"rulesets of " + mgSubject + ", are unknown" + meAccess, false, []string{mgRepoEP, meQueueEP, mgRulesEP}},
		{"branch read fails", nil, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: "[]"}
			e.replies[mgBranchEP] = mgAPIError("Server Error", "502")
		}, false, "gh api --hostname github.com " + mgBranchEP + " failed (exit status 1: gh: Server Error (HTTP 502)), so the " +
			"protection of " + mgSubject + ", is unknown" + meAccess, false, []string{mgRepoEP, meQueueEP, mgRulesEP, mgBranchEP}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMEExec()
			if tc.classic != nil {
				e = meClassicExec(*tc.classic)
			}
			tc.setup(e)
			got := CheckMergeEnforcement(context.Background(), meParams(e))
			if got.OK != tc.ok {
				t.Fatalf("ok = %v (%s), want %v", got.OK, got.Detail, tc.ok)
			}
			if tc.partial && !strings.Contains(got.Detail, tc.detail) || !tc.partial && got.Detail != tc.detail {
				t.Errorf("detail = %q\nwant (partial %v) %q", got.Detail, tc.partial, tc.detail)
			}
			if strings.ContainsAny(got.Detail, "\n\r") {
				t.Errorf("detail %q is not one line", got.Detail)
			}
			if eps := e.endpoints(); !slices.Equal(eps, tc.endpoints) {
				t.Errorf("gh endpoints = %q, want %q", eps, tc.endpoints)
			}
			meAssertGHCalls(t, e, "github.com", "acme", "app", "main")
		})
	}
}

// TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller covers D11 and
// AC5 for a strict branch protection with checks (cross-review #1 of the
// plan): GitHub applies it by default neither to admins nor to custom roles
// with the bypass branch protections permission, which permissions.admin does
// not show, so it is ok only when enforce_admins is enabled, and not ok when
// that is disabled, cannot be read, or has no enabled, whether
// permissions.admin is true, false, or missing, or the answer has no
// permissions at all; enforce_admins is read every time, after the strict.
func TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller(t *testing.T) {
	what := "the branch protection of " + mgSubject + ", requires branches to be up to date before merging with its 1 " +
		"required status check"
	const enforceCmd = "gh api --hostname github.com " + meEnforceEP
	endpoints := []string{mgRepoEP, meQueueEP, mgRulesEP, mgBranchEP, mgStrictEP, meEnforceEP}
	callers := []struct{ name, repo string }{
		{"an admin", meRepoJSON("main", "acme/app", true, false)},
		{"not an admin", meRepoJSON("main", "acme/app", false, false)},
		{"permissions with no admin", meRepoJSON("main", "acme/app", nil, false)},
		{"no permissions", meRepoJSON("main", "acme/app", nil, true)},
	}
	for _, tc := range []struct {
		name   string
		reply  mgReply
		ok     bool
		detail string
	}{
		{"enforce_admins enabled", mgReply{out: `{"url":"https://api.github.com/x","enabled":true}`}, true, what + ", and it " +
			"applies to admins and to custom roles that may bypass branch protections too (enforce_admins is enabled), so to you"},
		{"enforce_admins disabled", mgReply{out: `{"url":"https://api.github.com/x","enabled":false}`}, false, what + ", but " +
			"enforce_admins is disabled, so it does not apply to admins of the repository or to custom roles with the bypass branch " +
			"protections permission, which you may hold though you are not an admin, and GitHub may let you merge a head that is " +
			"not up to date: enable Do not allow bypassing the above settings in the branch protection, or " + meFix},
		{"enforce_admins cannot be read", mgAPIError("Not Found", "404"), false, enforceCmd + " failed (exit status 1: gh: Not " +
			"Found (HTTP 404)), so the protection of " + mgSubject + ", is unknown" + meAccess},
		{"enforce_admins answer has no enabled", mgReply{out: `{"url":"https://api.github.com/x"}`}, false, what + ", but " +
			enforceCmd + " printed no enabled, so whether it applies to admins and to custom roles that may bypass branch " +
			"protections is unknown: " + meFix + " instead"},
	} {
		for _, c := range callers {
			t.Run(tc.name+", "+c.name, func(t *testing.T) {
				e := meClassicExec(false)
				e.replies[mgRepoEP] = mgReply{out: c.repo}
				e.replies[meEnforceEP] = tc.reply
				got := CheckMergeEnforcement(context.Background(), meParams(e))
				if got.OK != tc.ok || got.Detail != tc.detail {
					t.Errorf("enforcement = %+v\nwant ok %v with %q", got, tc.ok, tc.detail)
				}
				if eps := e.endpoints(); !slices.Equal(eps, endpoints) {
					t.Errorf("gh endpoints = %q, want %q", eps, endpoints)
				}
				meAssertGHCalls(t, e, "github.com", "acme", "app", "main")
			})
		}
	}
}

// TestMergeEnforcement_PinsTheHostAndPassesTheVariablesRaw: every gh call
// carries --hostname, origin's host lowercased with ssh.github.com read as
// github.com; the REST paths escape owner, repository and branch, and the
// GraphQL variables pass them as they are, with -f.
func TestMergeEnforcement_PinsTheHostAndPassesTheVariablesRaw(t *testing.T) {
	for _, tc := range []struct{ host, apiHost string }{
		{"github.com", "github.com"}, {"ssh.github.com", "github.com"}, {"GHE.Example.com", "ghe.example.com"},
	} {
		t.Run(tc.host, func(t *testing.T) {
			e := newMEExec()
			const repo = "repos/%7Bowner%7D/app"
			e.replies = map[string]mgReply{
				repo:                                  {out: meRepoJSON("release/v1", "{owner}/app", false, false)},
				meQueueEP:                             {out: meQueueJSON("null")},
				repo + "/rules/branches/release%2Fv1": {out: mgRules(meChecksRule(5, true, "ci"))},
				repo + "/rulesets/5":                  {out: meRuleset(5, "rel", "never")},
			}
			p := meParams(e)
			p.Host, p.Owner, p.DefaultBranch = tc.host, "{owner}", "release/v1"
			if got := CheckMergeEnforcement(context.Background(), p); !got.OK {
				t.Fatalf("not ok: %s", got.Detail)
			}
			if n := len(e.callsOf("gh")); n != 4 {
				t.Errorf("%d gh calls, want 4: %q", n, e.endpoints())
			}
			meAssertGHCalls(t, e, tc.apiHost, "{owner}", "app", "release/v1")
		})
	}
}

// TestMergeEnforcement_OneDeadlineCoversEveryCall: every gh call runs under
// the same deadline, p.Timeout from the start, 20 s when it is zero or less.
func TestMergeEnforcement_OneDeadlineCoversEveryCall(t *testing.T) {
	for _, tc := range []struct {
		timeout, want time.Duration
	}{{5 * time.Second, 5 * time.Second}, {0, 20 * time.Second}, {-time.Second, 20 * time.Second}} {
		e := meClassicExec(true)
		e.replies[meEnforceEP] = mgReply{out: `{"enabled":true}`}
		p := meParams(e)
		p.Timeout = tc.timeout
		start := time.Now()
		if got := CheckMergeEnforcement(context.Background(), p); !got.OK {
			t.Fatalf("timeout %s: not ok: %s", tc.timeout, got.Detail)
		}
		end := time.Now()
		calls := e.callsOf("gh")
		if len(calls) != 6 {
			t.Fatalf("timeout %s: %d gh calls, want 6", tc.timeout, len(calls))
		}
		for _, c := range calls {
			if !c.deadline.Equal(calls[0].deadline) {
				t.Errorf("timeout %s: %q has deadline %v, the first call had %v", tc.timeout, c.args, c.deadline, calls[0].deadline)
			}
		}
		if d := calls[0].deadline; d.Before(start.Add(tc.want)) || d.After(end.Add(tc.want)) {
			t.Errorf("timeout %s: deadline %v is not %s after the start (%v..%v)", tc.timeout, d, tc.want, start, end)
		}
	}
}

// TestMergeEnforcement_TimeoutAndCallerContextAreNotOK: a call that outlasts
// the deadline, at any step, is not ok and names what ran; the caller's ctx
// done before or during the check is not ok with its cause.
func TestMergeEnforcement_TimeoutAndCallerContextAreNotOK(t *testing.T) {
	for _, tc := range []struct {
		block, ran string
		classic    bool
	}{
		{mgRepoEP, "gh api --hostname github.com repos/acme/app", false},
		{meQueueEP, "gh api --hostname github.com graphql (mergeQueue of main)", false},
		{mgRulesEP, "gh api --hostname github.com --paginate " + mgRulesEP, false},
		{meRuleset1EP, "gh api --hostname github.com " + meRuleset1EP, false},
		{mgBranchEP, "gh api --hostname github.com " + mgBranchEP, true},
		{mgStrictEP, "gh api --hostname github.com " + mgStrictEP, true},
		{meEnforceEP, "gh api --hostname github.com " + meEnforceEP, true},
	} {
		t.Run(tc.block, func(t *testing.T) {
			e := newMEExec()
			if tc.classic {
				e = meClassicExec(true)
			}
			e.block = tc.block
			p := meParams(e)
			p.Timeout = 30 * time.Millisecond
			start := time.Now()
			got := CheckMergeEnforcement(context.Background(), p)
			want := "the check timed out after 30ms while " + tc.ran + " ran, so the protection of origin's repository's default " +
				"branch is unknown: check the network and gh's access"
			if got.OK || got.Detail != want {
				t.Errorf("enforcement = %+v\nwant not ok with %q", got, want)
			}
			if elapsed := time.Since(start); elapsed > 10*time.Second {
				t.Errorf("the check took %s", elapsed)
			}
		})
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("merge stopped"))
	e := newMEExec()
	got := CheckMergeEnforcement(ctx, meParams(e))
	want := "the check was interrupted (merge stopped) while gh api --hostname github.com repos/acme/app ran, so the protection " +
		"of origin's repository's default branch is unknown"
	if got.OK || got.Detail != want {
		t.Errorf("enforcement = %+v\nwant not ok with %q", got, want)
	}

	ctx, cancelQueue := context.WithCancel(context.Background())
	defer cancelQueue()
	e = newMEExec()
	e.block = meQueueEP
	go func() {
		for len(e.endpoints()) < 2 {
			time.Sleep(time.Millisecond)
		}
		cancelQueue()
	}()
	got = CheckMergeEnforcement(ctx, meParams(e))
	if got.OK || !strings.HasPrefix(got.Detail, "the check was interrupted (context canceled) while gh api --hostname github.com "+
		"graphql (mergeQueue of main) ran") {
		t.Errorf("enforcement = %+v", got)
	}
}
