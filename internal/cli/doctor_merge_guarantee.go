package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The doctor check "Merge guarantee" (Check 11c), stage 7c-2 of plan
// docs/plans/active/2026-10-14-org-merge-guarantee.md (FR-7 of
// docs/specs/2026-10-07-org-multi-org-director.md): what makes the required
// status checks of a PR pass on a head with the latest default branch when
// it merges into origin's default branch. org.CheckMergeGuarantee decides
// the kind with git and gh; this check maps the kind to doctor's status (C3)
// and caches the result with org.SaveMergeGuarantee for `ralph status` and
// `ralph org merge-order`, which never run gh (C2, C5).

// mergeGuaranteeCheckName is the check's name in doctor's report.
const mergeGuaranteeCheckName = "Merge guarantee"

// orgLedgerCheckName is the name of the fail line checkMergeGuarantee adds
// when the org ledger cannot be resolved.
const orgLedgerCheckName = "Org ledger"

// mergeGuaranteeEnvFromOS is the production resolver for
// checkMergeGuarantee: real git and gh (nil Exec and LookPath) run in
// targetDir, and the cache goes into the org state dir
// org.ResolveOrgStateDir("", false) names (RALPH_ORG_STATE_DIR, else the
// personal file's [org].state_dir, else the main worktree's
// .harness/state/org, found with git rev-parse). That state dir is resolved
// from the process's working directory, not from targetDir; the CLI always
// passes ".", so both name the same repository. err is
// org.ResolveOrgStateDir's (a personal file that cannot be read, or whose
// state_dir is not absolute), and stateDir is "" with it.
func mergeGuaranteeEnvFromOS(targetDir string) (params org.MergeGuaranteeParams, stateDir string, err error) {
	stateDir, _, err = org.ResolveOrgStateDir("", false)
	return org.MergeGuaranteeParams{Dir: targetDir}, stateDir, err
}

// doctorMergeGuaranteeEnv is the resolver runDoctorFull hands to
// checkMergeGuarantee. It is a package variable (rather than a direct call
// to mergeGuaranteeEnvFromOS) so internal/cli's TestMain (main_test.go) can
// pin it to an Exec and a gh lookup that run nothing and a state dir that
// does not exist, keeping every runDoctor*-based test from running the real
// git and gh against GitHub and from writing merge-guarantee.json into the
// repository's own org ledger (same seam shape as doctorShellAliasEnv).
var doctorMergeGuaranteeEnv = mergeGuaranteeEnvFromOS

// checkMergeGuarantee runs org.CheckMergeGuarantee with the params
// resolveEnv gives for targetDir, saves the result into the state dir
// resolveEnv gives (org.SaveMergeGuarantee writes, unknown included, only
// when that dir already exists, and never creates it), and reports both
// through mergeGuaranteeResult. The check bounds itself (20 s for git and
// every gh call together, no gh call when gh is not on PATH), and an
// unknown kind is never "fail", so doctor's exit code never changes because
// of it.
//
// When resolveEnv cannot resolve the org ledger, the check still runs and
// its line says the result was not cached, after a fail line
// (orgLedgerCheckName) with the error, which names the personal file: the
// verbs that resolve the ledger exit 1 on the same error, and doctor, which
// reports such breakage, goes on with its other checks (plan
// 2026-10-30-org-state-dir-local, D4).
func checkMergeGuarantee(ctx context.Context, targetDir string, resolveEnv func(targetDir string) (org.MergeGuaranteeParams, string, error)) []checkResult {
	params, stateDir, err := resolveEnv(targetDir)
	c := org.CheckMergeGuarantee(ctx, params)
	if err != nil {
		ledger := checkResult{Name: orgLedgerCheckName, Status: "fail",
			Detail: org.PrintableInboxText("cannot resolve the org ledger: "+err.Error(), false)}
		r := checkResult{Name: mergeGuaranteeCheckName, Status: mergeGuaranteeStatus(c.Kind),
			Detail: org.PrintableInboxText(c.Kind+": "+c.Detail+"; not cached: the org ledger could not be resolved", false)}
		return []checkResult{ledger, r}
	}
	return []checkResult{mergeGuaranteeResult(c, org.SaveMergeGuarantee(stateDir, c))}
}

// mergeGuaranteeStatus is doctor's status for the kind (C3): pass for
// github_protection, warn for pr_checks_only, info for unknown.
func mergeGuaranteeStatus(kind string) string {
	switch kind {
	case org.MergeGuaranteeGitHubProtection:
		return "pass"
	case org.MergeGuaranteePRChecksOnly:
		return "warn"
	default:
		return "info"
	}
}

// mergeGuaranteeResult is doctor's result for the check c and its save s.
// The status follows the kind (mergeGuaranteeStatus): pass for
// github_protection, warn for pr_checks_only (c.Detail says what to set),
// info for unknown (c.Detail says why). A save that failed and could not
// remove the old cache either is warn whatever the kind, since `ralph
// status` and `ralph org merge-order` then read that stale cache (C5). The
// Detail is one line: the kind, c.Detail, then where the result was cached
// or why it was not.
func mergeGuaranteeResult(c org.MergeGuaranteeCheck, s org.MergeGuaranteeSave) checkResult {
	r := checkResult{Name: mergeGuaranteeCheckName, Status: mergeGuaranteeStatus(c.Kind)}
	var saved string
	switch {
	case s.NoLedger:
		saved = fmt.Sprintf("not cached: there is no org ledger yet at %s", filepath.Dir(s.Path))
	case s.Written:
		saved = "cached in " + s.Path
	case s.RemoveErr != nil:
		r.Status = "warn"
		saved = fmt.Sprintf("not cached (%v), and the old cache could not be removed (%v): a stale cache remains, and ralph "+
			"status and ralph org merge-order may show an older kind from it", s.WriteErr, s.RemoveErr)
	case s.Removed:
		saved = fmt.Sprintf("not cached (%v): the old cache was removed", s.WriteErr)
	default:
		saved = fmt.Sprintf("not cached (%v): there was no old cache", s.WriteErr)
	}
	r.Detail = org.PrintableInboxText(c.Kind+": "+c.Detail+"; "+saved, false)
	return r
}
