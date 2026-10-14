# Verify evidence: ralph org merge-order, cycle 3 (org-merge-order)

- Date: 2026-10-14
- Plan: `docs/plans/active/2026-10-14-org-merge-order.md` (Progress, last bullet: the cycle-3 decision on gh's PR limit)
- Report: `docs/reports/verify-2026-10-14-org-merge-order.md` (`## Cycle 3`)
- HEAD: 48f0aed7 (cycle-3 delta: `git diff e463eb9d..HEAD`)
- How: two binaries were built with `GOTOOLCHAIN=go1.25.8` before the probe ran: `<bin>/ralph-head` from the worktree at HEAD, and `<bin>/ralph-e463eb9d` from `git archive e463eb9d` (the commit before the cycle-3 fix, as a negative control). A scratch bash script (its text is in the last section) makes a bare origin and a work repo under a `mktemp -d` dir, seeds `<tmp>/state/manifest.jsonl`, and runs `ralph org merge-order` under `env -i` with HOME set to a temp dir and PATH set to one temp bin dir: `<tmp>/bin` (a git wrapper only) or `<tmp>/bin-gh` (the wrapper plus a fake `gh`). The git wrapper logs each call and answers only `git remote get-url origin` (with `git@github.com:acme/app.git`); everything else goes to the real git. The fake `gh` logs its argv, reads `--limit` (30 when absent, gh's documented default) and `--head`, and prints, newest first, a configured number of fork PRs (`isCrossRepository: true`, numbers 101 and up) followed by the branch's own PR, cut at the limit. Nothing runs the real `gh`, herdr, or osascript, and nothing reaches the network.
- gh's ordering: the `PullRequestList` GraphQL query below is read from the strings of the local gh binary (2.102.0) with `strings`; no network.
- Redaction: the temp dir is `<tmp>`, the session scratchpad `$SP`, and `$HOME` is `~`. SHAs and times are from the run. Below this header, everything is the commands' output as printed (checked with `cmp` against the scratch copy), except the `##` headings and code fences that wrap it, and the two `jq -e .` dumps of the settings files in the static run (148 lines each, replaced by one bracketed line; the full run is the gitignored `docs/evidence/verify-2026-10-14-011142.log`).
- Note: a first run of the probe was discarded: its fake `gh` called `awk` by name, which is not on the one-dir PATH, so every gh call exited 127. The script now calls `/usr/bin/awk`; the run below is the second one, in a fresh temp dir.

## HEAD, plan digest, cycle counter, cycle-3 delta

```
$ git rev-parse HEAD
48f0aed7164736696491aa0f04e76559087d5770
$ ./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-order.md
c0b1e9230999
$ command cat .harness/state/standard-pipeline/cycle-count.json
{"plan_path": "~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-merge-order/docs/plans/active/2026-10-14-org-merge-order.md", "cycle": 2}

$ git log --oneline e463eb9d..HEAD
48f0aed7 docs: record the cycle-3 self-review insight for org-merge-order
a8fd9992 docs: add cycle-3 self-review for org-merge-order
c885fcab fix: name the gate record's org in merge-order's gate command
d7599d82 fix: read enough PRs in merge-order before leaving out forks
984a163e docs: record the cycle-3 PR limit decision in the org-merge-order plan
$ git diff --stat e463eb9d..HEAD | tail -1
 11 files changed, 340 insertions(+), 46 deletions(-)
$ git diff e463eb9d..HEAD -U0 -- internal/org/mergeorder.go | grep "^@@"
@@ -85 +85,2 @@ const (
@@ -95,0 +97,7 @@ const (
@@ -100 +108,2 @@ const (
@@ -123 +132,2 @@ type MergeOrderParams struct {
@@ -210,2 +220,2 @@ type MergeOrderBinding struct {
@@ -471,4 +481,6 @@ type ghPullRequest struct {
@@ -480,0 +493,4 @@ func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec,
@@ -483 +499 @@ func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec,
@@ -488 +504 @@ func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec,
@@ -491,0 +508,3 @@ func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec,
@@ -503 +522 @@ func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec,
@@ -507 +526 @@ func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec,
@@ -539,4 +557,0 @@ func (f mergeGHFailures) notes() []string {
@@ -544 +559 @@ func (f mergeGHFailures) notes() []string {
@@ -548,0 +564,27 @@ func (f mergeGHFailures) notes() []string {
@@ -682,4 +724,7 @@ func mergeRepoFromPRURL(s string) (repo mergeRepo, ok bool) {
@@ -689 +734 @@ func lookupMergePR(ctx context.Context, dir string, run MergeOrderExec, timeout
@@ -692 +737 @@ func lookupMergePR(ctx context.Context, dir string, run MergeOrderExec, timeout
@@ -694 +739 @@ func lookupMergePR(ctx context.Context, dir string, run MergeOrderExec, timeout
@@ -698 +743 @@ func lookupMergePR(ctx context.Context, dir string, run MergeOrderExec, timeout
@@ -700 +745,2 @@ func lookupMergePR(ctx context.Context, dir string, run MergeOrderExec, timeout
@@ -795 +841,4 @@ func classifyMergeTarget(t *mergeTarget, remote mergeRemote) (class, detail stri
@@ -797,2 +846,5 @@ func mergeGateCommand(t *mergeTarget) string {
@@ -800 +852 @@ func mergeGateCommand(t *mergeTarget) string {
```

## Go build and vet (go1.25.8 and go1.26.0), gofmt, skill sync, skill copies, diff --check

```
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
rc=0
$ GOTOOLCHAIN=go1.25.8 go build ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go vet ./...
rc=0
$ go version
go version go1.26.0 darwin/arm64
$ go vet ./...
rc=0
$ gofmt -l internal/ cmd/
rc=0
$ scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
rc=0
$ scripts/check-sync.sh | tail -8
=== Sync Summary ===
  IDENTICAL:      167
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
rc=0
$ cmp the four /org skill copies
same: .agents/skills/org/SKILL.md
same: templates/base/.claude/skills/org/SKILL.md
same: templates/base/.agents/skills/org/SKILL.md
$ git diff --check ac34ccf3...HEAD
rc=0
$ git diff --check e463eb9d..HEAD
rc=0
```

## ./scripts/run-static-verify.sh (changed scope)

```
$ ./scripts/run-static-verify.sh
# Verification run
- Timestamp: 2026-10-14T01:11:42Z
- Mode: static
- Requested scope: changed

==> Running local verifier
==> shellcheck hook + verify scripts
    OK
==> sh -n .claude/hooks/check_mojibake.sh
    OK
==> sh -n .claude/hooks/lib_json.sh
    OK
==> sh -n .claude/hooks/post_edit_verify.sh
    OK
==> sh -n .claude/hooks/post_tool_failure_feedback.sh
    OK
==> sh -n .claude/hooks/pre_bash_guard.sh
    OK
==> sh -n .claude/hooks/precompact_checkpoint.sh
    OK
==> sh -n .claude/hooks/prompt_gate.sh
    OK
==> sh -n .claude/hooks/ralph-dispatch.sh
    OK
==> sh -n .claude/hooks/session_end_summary.sh
    OK
==> sh -n .claude/hooks/session_start_context.sh
    OK
==> sh -n templates/base/.claude/hooks/check_mojibake.sh
    OK
==> sh -n templates/base/.claude/hooks/lib_json.sh
    OK
==> sh -n templates/base/.claude/hooks/post_edit_verify.sh
    OK
==> sh -n templates/base/.claude/hooks/post_tool_failure_feedback.sh
    OK
==> sh -n templates/base/.claude/hooks/pre_bash_guard.sh
    OK
==> sh -n templates/base/.claude/hooks/precompact_checkpoint.sh
    OK
==> sh -n templates/base/.claude/hooks/prompt_gate.sh
    OK
==> sh -n templates/base/.claude/hooks/ralph-dispatch.sh
    OK
==> sh -n templates/base/.claude/hooks/session_end_summary.sh
    OK
==> sh -n templates/base/.claude/hooks/session_start_context.sh
    OK
==> awk parse of the guard in .claude/hooks
    OK
==> awk parse of the guard in templates/base/.claude/hooks
    OK
==> jq -e . .claude/settings.json
[148 lines of jq -e . output (the settings file) omitted; full run in docs/evidence/verify-2026-10-14-011142.log]
    OK
==> jq -e . templates/base/.claude/settings.json
[148 lines of jq -e . output (the settings file) omitted; full run in docs/evidence/verify-2026-10-14-011142.log]
    OK
==> Codex hook single-source guard
    OK
==> Codex inline hook detector smoke test
    OK
==> Codex PR provenance policy guard
    OK
==> scripts/check-sync.sh
=== Checking templates/base/ <-> root ===
  TEMPLATE_ONLY  .agents/skills/.gitkeep
  KNOWN_DIFF     .claude/rules/ralph/model-routing.md
  KNOWN_DIFF     .github/workflows/verify.yml
  TEMPLATE_ONLY  .ralph/core/settings.ralph.json
  KNOWN_DIFF     CLAUDE.md
  TEMPLATE_ONLY  docs/architecture/.gitkeep
  TEMPLATE_ONLY  docs/evidence/.gitkeep
  KNOWN_DIFF     docs/quality/quality-gates.md
  TEMPLATE_ONLY  docs/recipes/.gitkeep
  KNOWN_DIFF     docs/recipes/adding-a-language-pack.md
  TEMPLATE_ONLY  docs/references/.gitkeep
  TEMPLATE_ONLY  docs/reports/.gitkeep
  TEMPLATE_ONLY  docs/research/.gitkeep
  TEMPLATE_ONLY  docs/roadmap/.gitkeep
  TEMPLATE_ONLY  docs/tech-debt/.gitkeep
  TEMPLATE_ONLY  ralph.toml

=== Checking AGENTS.md managed block ===

=== Checking templates/packs/ <-> packs/languages/ ===

=== Checking for root-only files not in templates/base/ ===

=== Checking for packs/languages/ files not in templates/packs/ ===

=== Sync Summary ===
  IDENTICAL:      167
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
    OK
==> scripts/check-pipeline-sync.sh
[ok] Canonical source valid
[ok] .claude/skills/implement/SKILL.md: all pipeline steps referenced
[ok] .claude/skills/cross-review/SKILL.md: all pipeline steps referenced
[ok] .claude/rules/ralph/subagent-policy.md: all pipeline steps referenced
[ok] docs/quality/definition-of-done.md: all pipeline steps referenced
[ok] README.md: all pipeline steps referenced
[ok] AGENTS.md: all pipeline steps referenced
    OK
==> scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
    OK
==> scripts/check-template-purity.sh
=== Checking templates for meta-repo-specific references ===

PASS: no meta-repo-specific references found in templates.
    OK
==> tech-debt README plan references
    OK
==> Language scope: changed (changed_languages)
==> Language packs selected: golang
==> Running golang verifier
==> golang project roots selected: .
==> Go project root: .
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned ac34ccf3..48f0aed7 against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-14-011142.log
rc=0
```

## gh: the --limit default and the PullRequestList query (local binary, no network)

```
$ GH_CONFIG_DIR=$SP/c3/ghconf /opt/homebrew/bin/gh --version | head -1
gh version 2.102.0 (2026-09-30)
$ GH_CONFIG_DIR=$SP/c3/ghconf /opt/homebrew/bin/gh pr list --help | grep -n -- "--limit\|--head\|--state"
24:  -H, --head string       Filter by head branch ("<owner>:<branch>" syntax not supported)
28:  -L, --limit int         Maximum number of items to fetch (default 30)
30:  -s, --state string      Filter by state: {open|closed|merged|all} (default "open")
56:  $ gh pr list --head "typo"
65:  $ gh pr list --search "<SHA>" --state merged
$ strings -n 8 /opt/homebrew/Cellar/gh/2.102.0/bin/gh | sed -n "/^query PullRequestList(/,/^endCursor$/p"
query PullRequestList(
$owner: String!,
$repo: String!,
$limit: Int!,
$endCursor: String,
$baseBranch: String,
$headBranch: String,
$state: [PullRequestState!] = OPEN
repository(owner: $owner, name: $repo) {
pullRequests(
states: $state,
baseRefName: $baseBranch,
headRefName: $headBranch,
first: $limit,
after: $endCursor,
orderBy: {field: CREATED_AT, direction: DESC}
totalCount
pageInfo {
hasNextPage
endCursor
$ strings -n 8 /opt/homebrew/Cellar/gh/2.102.0/bin/gh | grep -c "^query PullRequestList("
1
```

## Probe setup

```
TMP=<tmp>
worktree HEAD: 48f0aed7164736696491aa0f04e76559087d5770
binaries: GOTOOLCHAIN=go1.25.8 go build -o <bin>/ralph-head ./cmd/ralph (HEAD); the same from git archive e463eb9d -> <bin>/ralph-e463eb9d (pre-fix)
main     7f5248c494683dbb91637a676c6a5e872483a70b
feat/m   3c66b2083823646eb7f85ada79b7a4be2bf315f4 (gated, never pushed)
feat/k   06ee209fa7ec066ea8fb2909501229265361b171 (gated, pushed)
feat/r   14bddcff1ae75adff1680bc10f825030f8b6e889 (gated, pushed)
feat/w   6a8f815b3ecbf357c063668f6c3db6489ec919fb (gated, pushed)
feat/c   9b63b9236588789d267d09a876a9f2edf3c90100 (gated, pushed)
feat/j   b37bef40a56ab86fe07d30d9697da559507df937 (gated, pushed)
feat/x   gated 94221ac749becee808b6ad10b292f17d6bb1dce5 by org-q (no binding for feat/x), pushed after the gate: a3ef23a8c0b95ebf7ab343ee9a0b5d708cfb01cf
feat/n   bound, no gate, never pushed
feat/q   bound by org-q, no gate, never pushed
$ git -C <tmp>/work ls-remote --symref origin HEAD 'refs/heads/*'
ref: refs/heads/main	HEAD
7f5248c494683dbb91637a676c6a5e872483a70b	HEAD
9b63b9236588789d267d09a876a9f2edf3c90100	refs/heads/feat/c
b37bef40a56ab86fe07d30d9697da559507df937	refs/heads/feat/j
06ee209fa7ec066ea8fb2909501229265361b171	refs/heads/feat/k
14bddcff1ae75adff1680bc10f825030f8b6e889	refs/heads/feat/r
6a8f815b3ecbf357c063668f6c3db6489ec919fb	refs/heads/feat/w
a3ef23a8c0b95ebf7ab343ee9a0b5d708cfb01cf	refs/heads/feat/x
7f5248c494683dbb91637a676c6a5e872483a70b	refs/heads/main
Seeded ledger <tmp>/state/manifest.jsonl:
{"ts":"2026-10-14T01:00:01Z","org_id":"org-m","seat_id":"","event":"scope_reserved","worktree":"/wt/m","details":"paths=internal/org-m/ split=demo feature=m digest=0123456789ab branch=feat/m approval=per-plan"}
{"ts":"2026-10-14T01:00:02Z","org_id":"org-k","seat_id":"","event":"scope_reserved","worktree":"/wt/k","details":"paths=internal/org-k/ split=demo feature=k digest=0123456789ab branch=feat/k approval=per-plan"}
{"ts":"2026-10-14T01:00:03Z","org_id":"org-r","seat_id":"","event":"scope_reserved","worktree":"/wt/r","details":"paths=internal/org-r/ split=demo feature=r digest=0123456789ab branch=feat/r approval=per-plan"}
{"ts":"2026-10-14T01:00:04Z","org_id":"org-w","seat_id":"","event":"scope_reserved","worktree":"/wt/w","details":"paths=internal/org-w/ split=demo feature=w digest=0123456789ab branch=feat/w approval=per-plan"}
{"ts":"2026-10-14T01:00:05Z","org_id":"org-c","seat_id":"","event":"scope_reserved","worktree":"/wt/c","details":"paths=internal/org-c/ split=demo feature=c digest=0123456789ab branch=feat/c approval=per-plan"}
{"ts":"2026-10-14T01:00:06Z","org_id":"org-j","seat_id":"","event":"scope_reserved","worktree":"/wt/j","details":"paths=internal/org-j/ split=demo feature=j digest=0123456789ab branch=feat/j approval=per-plan"}
{"ts":"2026-10-14T01:00:07Z","org_id":"org-n","seat_id":"","event":"scope_reserved","worktree":"/wt/n","details":"paths=internal/org-n/ split=demo feature=n digest=0123456789ab branch=feat/n approval=per-plan"}
{"ts":"2026-10-14T01:00:08Z","org_id":"org-q","seat_id":"","event":"scope_reserved","worktree":"/wt/q","details":"paths=internal/org-q/ split=demo feature=q digest=0123456789ab branch=feat/q approval=per-plan"}
{"ts":"2026-10-14T01:00:11Z","org_id":"org-m","seat_id":"","event":"gate_passed","details":"head=3c66b2083823646eb7f85ada79b7a4be2bf315f4 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/m by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:12Z","org_id":"org-k","seat_id":"","event":"gate_passed","details":"head=06ee209fa7ec066ea8fb2909501229265361b171 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/k by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:13Z","org_id":"org-r","seat_id":"","event":"gate_passed","details":"head=14bddcff1ae75adff1680bc10f825030f8b6e889 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/r by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:14Z","org_id":"org-w","seat_id":"","event":"gate_passed","details":"head=6a8f815b3ecbf357c063668f6c3db6489ec919fb base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/w by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:15Z","org_id":"org-c","seat_id":"","event":"gate_passed","details":"head=9b63b9236588789d267d09a876a9f2edf3c90100 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/c by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:16Z","org_id":"org-j","seat_id":"","event":"gate_passed","details":"head=b37bef40a56ab86fe07d30d9697da559507df937 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/j by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:17Z","org_id":"org-q","seat_id":"","event":"gate_passed","details":"head=94221ac749becee808b6ad10b292f17d6bb1dce5 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/x by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
$ command cat <tmp>/git-wrapper
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/git.log"
if [ $# -eq 3 ] && [ "$1 $2 $3" = "remote get-url origin" ] && [ -n "${PROBE_ORIGIN_URL-}" ]; then
  printf '%s\n' "$PROBE_ORIGIN_URL"; exit 0
fi
exec /opt/homebrew/bin/git "$@"
$ command cat <tmp>/gh-fake
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/gh.log"
head=""; limit=30
while [ $# -gt 0 ]; do
  case "$1" in --head) head="$2"; shift 2 ;; --limit) limit="$2"; shift 2 ;; *) shift ;; esac
done
exec /usr/bin/awk -v head="$head" -v limit="$limit" -v U="$PROBE_PR_BASE" -v OU="${PROBE_OWN_BASE:-$PROBE_PR_BASE}" '
$1 == head { forks = $2; own = $3; st = $4; base = $5; found = 1 }
END {
  if (forks == "FAIL") { print "To get started with GitHub CLI, please run:  gh auth login" > "/dev/stderr"; exit 4 }
  n = 0; printf "["
  for (i = forks + 0; i >= 1 && n < limit; i--) {
    if (n) printf ","
    printf "{\"number\":%d,\"state\":\"OPEN\",\"url\":\"%s/%d\",\"headRefOid\":\"1111111111111111111111111111111111111111\",\"baseRefName\":\"main\",\"isCrossRepository\":true}", 100 + i, U, 100 + i
    n++
  }
  if (found && own != "-" && n < limit) {
    if (n) printf ","
    printf "{\"number\":%d,\"state\":\"%s\",\"url\":\"%s/%d\",\"headRefOid\":\"0000000000000000000000000000000000000000\",\"baseRefName\":\"%s\",\"isCrossRepository\":false}", own, st, OU, own, base
    n++
  }
  printf "]\n"
}' "$PROBE_GH_CONF"
$ command cat <tmp>/conf-main
feat/m 1000 19 MERGED main
feat/k 1000 22 OPEN main
feat/r 0 21 OPEN main
feat/w 30 23 OPEN release
feat/c 40 20 CLOSED main
feat/j 999 24 OPEN main
$ command cat <tmp>/conf-one
feat/m 1000 19 MERGED main
feat/k 0 22 OPEN main
feat/r 0 21 OPEN main
feat/w 30 23 OPEN release
feat/c 40 20 CLOSED main
feat/j 999 24 OPEN main
$ command cat <tmp>/conf-fail
feat/m 1000 19 MERGED main
feat/k 1000 22 OPEN main
feat/r 0 21 OPEN main
feat/w 30 23 OPEN release
feat/c 40 20 CLOSED main
feat/j FAIL - - -
Fake gh output sizes for conf-main with --limit 1000 (PR count per branch):
  feat/m: --limit 1000 -> total/own 1000 0; no --limit -> total/own 30 0
  feat/k: --limit 1000 -> total/own 1000 0; no --limit -> total/own 30 0
  feat/r: --limit 1000 -> total/own 1 1; no --limit -> total/own 1 1
  feat/w: --limit 1000 -> total/own 31 1; no --limit -> total/own 30 0
  feat/c: --limit 1000 -> total/own 41 1; no --limit -> total/own 30 0
  feat/j: --limit 1000 -> total/own 1000 1; no --limit -> total/own 30 0
  feat/n: --limit 1000 -> total/own 0 0; no --limit -> total/own 0 0
Ledger and refs (before the runs):
f9f7a96f84d8ae7a7654aa726100554a0b20c1dcb2fd039e3efe228836519c3e  ./manifest.jsonl
      15    3697 manifest.jsonl
manifest.jsonl
0461e7dd0cfdd75d7b436ab969e3f6c570741ec41a85105873bfed22f3c98fef  -
d4a9dfc2c0895dccf5d7a63fa5d25ec02d35bccb9ed6bd530a8677eb6e53d25d  -
```

## Probe runs

### Q1 HEAD: 30/40 fork PRs newer than the own PR, 1000 fork PRs (feat/m, feat/k), 999 forks + own (feat/j)

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-main PATH=<tmp>/bin-gh ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T01:15:58Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/k	-	06ee209fa7ec	7f5248c49468	org-k/k	the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/r	#21 https://github.com/acme/app/pull/21	14bddcff1ae7	7f5248c49468	org-r/r	the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/j	#24 https://github.com/acme/app/pull/24	b37bef40a56a	7f5248c49468	org-j/j	the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	6a8f815b3ecb	7f5248c49468	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
5	regate	feat/x	-	94221ac749be	7f5248c49468	-	pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate --org-id org-q again (the gate runs only while the feature org runs)
6	no_remote	feat/m	-	3c66b2083823	7f5248c49468	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
7	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
8	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh returned 1000 PRs, its limit, for each of 2 branches (feat/m, feat/k), all from forks, and their own PRs may be past them: no PR is used for them, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 1 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T01:15:58Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/m --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/k --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/r --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/j --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### Q2 HEAD: same, --json

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-main PATH=<tmp>/bin-gh ralph-head org merge-order --json --state-dir <tmp>/state
--- stdout
{
  "default_branch": "main",
  "default_head": "7f5248c494683dbb91637a676c6a5e872483a70b",
  "listed_at": "2026-10-14T01:15:59Z",
  "gh_available": true,
  "entries": [
    {
      "rank": 1,
      "class": "ready",
      "branch": "feat/k",
      "remote_head": "06ee209fa7ec066ea8fb2909501229265361b171",
      "detail": "the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "06ee209fa7ec066ea8fb2909501229265361b171",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:12Z"
      },
      "binding": {
        "org_id": "org-k",
        "feature": "k",
        "split": "demo",
        "ts": "2026-10-14T01:00:02Z"
      }
    },
    {
      "rank": 2,
      "class": "ready",
      "branch": "feat/r",
      "remote_head": "14bddcff1ae75adff1680bc10f825030f8b6e889",
      "detail": "the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip",
      "pr": {
        "number": 21,
        "state": "OPEN",
        "url": "https://github.com/acme/app/pull/21",
        "head_ref_oid": "0000000000000000000000000000000000000000",
        "base_ref_name": "main"
      },
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "14bddcff1ae75adff1680bc10f825030f8b6e889",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:13Z"
      },
      "binding": {
        "org_id": "org-r",
        "feature": "r",
        "split": "demo",
        "ts": "2026-10-14T01:00:03Z"
      }
    },
    {
      "rank": 3,
      "class": "ready",
      "branch": "feat/j",
      "remote_head": "b37bef40a56ab86fe07d30d9697da559507df937",
      "detail": "the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip",
      "pr": {
        "number": 24,
        "state": "OPEN",
        "url": "https://github.com/acme/app/pull/24",
        "head_ref_oid": "0000000000000000000000000000000000000000",
        "base_ref_name": "main"
      },
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "b37bef40a56ab86fe07d30d9697da559507df937",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:16Z"
      },
      "binding": {
        "org_id": "org-j",
        "feature": "j",
        "split": "demo",
        "ts": "2026-10-14T01:00:06Z"
      }
    },
    {
      "rank": 4,
      "class": "wrong_base",
      "branch": "feat/w",
      "remote_head": "6a8f815b3ecbf357c063668f6c3db6489ec919fb",
      "detail": "PR #23 merges into release, not origin's default branch main: change the PR's base to main",
      "pr": {
        "number": 23,
        "state": "OPEN",
        "url": "https://github.com/acme/app/pull/23",
        "head_ref_oid": "0000000000000000000000000000000000000000",
        "base_ref_name": "release"
      },
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "6a8f815b3ecbf357c063668f6c3db6489ec919fb",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:14Z"
      },
      "binding": {
        "org_id": "org-w",
        "feature": "w",
        "split": "demo",
        "ts": "2026-10-14T01:00:04Z"
      }
    },
    {
      "rank": 5,
      "class": "regate",
      "branch": "feat/x",
      "remote_head": "a3ef23a8c0b95ebf7ab343ee9a0b5d708cfb01cf",
      "detail": "pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate --org-id org-q again (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "94221ac749becee808b6ad10b292f17d6bb1dce5",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:17Z"
      }
    },
    {
      "rank": 6,
      "class": "no_remote",
      "branch": "feat/m",
      "remote_head": "",
      "detail": "origin has no branch feat/m: it was not pushed, or was deleted after a merge",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "3c66b2083823646eb7f85ada79b7a4be2bf315f4",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:11Z"
      },
      "binding": {
        "org_id": "org-m",
        "feature": "m",
        "split": "demo",
        "ts": "2026-10-14T01:00:01Z"
      }
    },
    {
      "rank": 7,
      "class": "no_gate",
      "branch": "feat/n",
      "remote_head": "",
      "detail": "no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)",
      "binding": {
        "org_id": "org-n",
        "feature": "n",
        "split": "demo",
        "ts": "2026-10-14T01:00:07Z"
      }
    },
    {
      "rank": 8,
      "class": "no_gate",
      "branch": "feat/q",
      "remote_head": "",
      "detail": "no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)",
      "binding": {
        "org_id": "org-q",
        "feature": "q",
        "split": "demo",
        "ts": "2026-10-14T01:00:08Z"
      }
    }
  ],
  "notes": [
    "gh returned 1000 PRs, its limit, for each of 2 branches (feat/m, feat/k), all from forks, and their own PRs may be past them: no PR is used for them, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked",
    "1 branch(es) whose PR is merged or closed are not shown (--all shows them)",
    "3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)",
    "the order is as of the ls-remote at 2026-10-14T01:15:59Z and is not a guarantee: origin/main can move before a merge"
  ]
}
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/m --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/k --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/r --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/j --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### Q3 HEAD: same, --all

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-main PATH=<tmp>/bin-gh ralph-head org merge-order --all --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T01:16:00Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/k	-	06ee209fa7ec	7f5248c49468	org-k/k	the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/r	#21 https://github.com/acme/app/pull/21	14bddcff1ae7	7f5248c49468	org-r/r	the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/j	#24 https://github.com/acme/app/pull/24	b37bef40a56a	7f5248c49468	org-j/j	the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	6a8f815b3ecb	7f5248c49468	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
5	regate	feat/x	-	94221ac749be	7f5248c49468	-	pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate --org-id org-q again (the gate runs only while the feature org runs)
6	no_remote	feat/m	-	3c66b2083823	7f5248c49468	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
7	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
8	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	closed	feat/c	#20 https://github.com/acme/app/pull/20	9b63b9236588	7f5248c49468	org-c/c	PR #20 was closed without a merge
note: gh returned 1000 PRs, its limit, for each of 2 branches (feat/m, feat/k), all from forks, and their own PRs may be past them: no PR is used for them, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T01:16:00Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/m --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/k --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/r --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/j --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### Q4 HEAD: only feat/m has 1000 fork PRs (one branch: the singular note)

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-one PATH=<tmp>/bin-gh ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T01:16:00Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/k	#22 https://github.com/acme/app/pull/22	06ee209fa7ec	7f5248c49468	org-k/k	the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/r	#21 https://github.com/acme/app/pull/21	14bddcff1ae7	7f5248c49468	org-r/r	the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/j	#24 https://github.com/acme/app/pull/24	b37bef40a56a	7f5248c49468	org-j/j	the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	6a8f815b3ecb	7f5248c49468	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
5	regate	feat/x	-	94221ac749be	7f5248c49468	-	pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate --org-id org-q again (the gate runs only while the feature org runs)
6	no_remote	feat/m	-	3c66b2083823	7f5248c49468	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
7	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
8	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh returned 1000 PRs for feat/m, its limit, all from forks, and the branch's own PR may be past them: no PR is used for it, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 1 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T01:16:00Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/m --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/k --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/r --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/j --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### Q5 HEAD: gh fails for feat/j, 1000 fork PRs for feat/m and feat/k (note order)

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-fail PATH=<tmp>/bin-gh ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T01:16:01Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/k	-	06ee209fa7ec	7f5248c49468	org-k/k	the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/r	#21 https://github.com/acme/app/pull/21	14bddcff1ae7	7f5248c49468	org-r/r	the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/j	-	b37bef40a56a	7f5248c49468	org-j/j	the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	6a8f815b3ecb	7f5248c49468	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
5	regate	feat/x	-	94221ac749be	7f5248c49468	-	pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate --org-id org-q again (the gate runs only while the feature org runs)
6	no_remote	feat/m	-	3c66b2083823	7f5248c49468	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
7	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
8	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh pr list --head feat/j failed (exit status 4: To get started with GitHub CLI, please run:  gh auth login): its PR is unknown, so merged, closed and the PR base are not told apart for it
note: gh returned 1000 PRs, its limit, for each of 2 branches (feat/m, feat/k), all from forks, and their own PRs may be past them: no PR is used for them, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 1 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T01:16:01Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/m --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/k --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/r --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/j --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### Q6 HEAD: the own PRs' url names upstream-org/app (repository check stops after two limit-full branches)

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-main PATH=<tmp>/bin-gh PROBE_OWN_BASE=https://github.com/upstream-org/app/pull ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T01:16:01Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/k	-	06ee209fa7ec	7f5248c49468	org-k/k	the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/r	-	14bddcff1ae7	7f5248c49468	org-r/r	the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/w	-	6a8f815b3ecb	7f5248c49468	org-w/w	the gate passed on 6a8f815b3ecb, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	ready	feat/c	-	9b63b9236588	7f5248c49468	org-c/c	the gate passed on 9b63b9236588, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
5	ready	feat/j	-	b37bef40a56a	7f5248c49468	org-j/j	the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
6	regate	feat/x	-	94221ac749be	7f5248c49468	-	pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate --org-id org-q again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	3c66b2083823	7f5248c49468	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh returned 1000 PRs, its limit, for each of 2 branches (feat/m, feat/k), all from forks, and their own PRs may be past them: no PR is used for them, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: gh reads upstream-org/app, not origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked; run gh repo set-default acme/app in this repository
note: 5 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T01:16:01Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 3
    gh pr list --head feat/m --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/k --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/r --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### Q7 HEAD: no gh on PATH

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-main PATH=<tmp>/bin ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T01:16:01Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/k	-	06ee209fa7ec	7f5248c49468	org-k/k	the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/r	-	14bddcff1ae7	7f5248c49468	org-r/r	the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/w	-	6a8f815b3ecb	7f5248c49468	org-w/w	the gate passed on 6a8f815b3ecb, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	ready	feat/c	-	9b63b9236588	7f5248c49468	org-c/c	the gate passed on 9b63b9236588, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
5	ready	feat/j	-	b37bef40a56a	7f5248c49468	org-j/j	the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
6	regate	feat/x	-	94221ac749be	7f5248c49468	-	pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate --org-id org-q again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	3c66b2083823	7f5248c49468	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 5 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T01:16:01Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 1 (remote get-url origin: 0, ls-remote: 1)
--- gh calls: 0
```

### Q8 pre-fix e463eb9d (negative control): same as Q1

```
$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/conf-main PATH=<tmp>/bin-gh ralph-e463eb9d org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T01:16:02Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/k	-	06ee209fa7ec	7f5248c49468	org-k/k	the gate passed on 06ee209fa7ec, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/r	#21 https://github.com/acme/app/pull/21	14bddcff1ae7	7f5248c49468	org-r/r	the gate passed on 14bddcff1ae7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/w	-	6a8f815b3ecb	7f5248c49468	org-w/w	the gate passed on 6a8f815b3ecb, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	ready	feat/c	-	9b63b9236588	7f5248c49468	org-c/c	the gate passed on 9b63b9236588, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
5	ready	feat/j	-	b37bef40a56a	7f5248c49468	org-j/j	the gate passed on b37bef40a56a, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
6	regate	feat/x	-	94221ac749be	7f5248c49468	-	pushed after the gate: origin has a3ef23a8c0b9, the gate checked 94221ac749be: run ralph org gate again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	3c66b2083823	7f5248c49468	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: 5 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T01:16:02Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/k --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/j --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### Ledger, refs and files after Q1-Q8 (AC9)

```
Ledger and refs (after the runs):
f9f7a96f84d8ae7a7654aa726100554a0b20c1dcb2fd039e3efe228836519c3e  ./manifest.jsonl
      15    3697 manifest.jsonl
manifest.jsonl
0461e7dd0cfdd75d7b436ab969e3f6c570741ec41a85105873bfed22f3c98fef  -
d4a9dfc2c0895dccf5d7a63fa5d25ec02d35bccb9ed6bd530a8677eb6e53d25d  -
Files newer than the marker under state, work, origin.git, home:
(end of list)
```

## Probe script

```bash
#!/bin/bash
# Cycle-3 probe of ralph org merge-order (org-merge-order verify).
# Scratch only: bare origin + work repo in a temp dir, a fake gh that honours
# --limit (default 30, like gh), and a git wrapper. No herdr, no network.
set -u
SP=$SP
HEADBIN=$SP/c3/bin/ralph-head
PREBIN=$SP/c3/bin/ralph-e463eb9d
REALGIT=/opt/homebrew/bin/git
T=$(mktemp -d "$SP/c3/probe.XXXXXX")
mkdir -p "$T/home" "$T/logs" "$T/state" "$T/bin" "$T/bin-gh"
printf '[user]\n\tname = probe\n\temail = probe@example.invalid\n[init]\n\tdefaultBranch = main\n' > "$T/gitconfig"
export GIT_CONFIG_GLOBAL="$T/gitconfig" GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_DATE='2026-10-14T00:00:00Z' GIT_COMMITTER_DATE='2026-10-14T00:00:00Z'
g() { "$REALGIT" "$@"; }

echo "TMP=<tmp>"
echo "worktree HEAD: $(g -C ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-merge-order rev-parse HEAD)"
echo "binaries: GOTOOLCHAIN=go1.25.8 go build -o <bin>/ralph-head ./cmd/ralph (HEAD); the same from git archive e463eb9d -> <bin>/ralph-e463eb9d (pre-fix)"

g init -q --bare "$T/origin.git"
g init -q -b main "$T/work"
cd "$T/work" || exit 1
g remote add origin "$T/origin.git"
EMPTY=$(g mktree </dev/null)
C0=$(g commit-tree -m main "$EMPTY")
g update-ref refs/heads/main "$C0"
mk() { g commit-tree -p "$1" -m "$2" "$1^{tree}"; }
R=$(mk "$C0" r); W=$(mk "$C0" w); C=$(mk "$C0" c); M=$(mk "$C0" m)
K=$(mk "$C0" k); J=$(mk "$C0" j); X0=$(mk "$C0" x0); X1=$(mk "$X0" x1)
g push -q origin "$C0:refs/heads/main" "$R:refs/heads/feat/r" "$W:refs/heads/feat/w" \
  "$C:refs/heads/feat/c" "$K:refs/heads/feat/k" "$J:refs/heads/feat/j" "$X1:refs/heads/feat/x"
echo "main     $C0"
echo "feat/m   $M (gated, never pushed)"
echo "feat/k   $K (gated, pushed)"
echo "feat/r   $R (gated, pushed)"
echo "feat/w   $W (gated, pushed)"
echo "feat/c   $C (gated, pushed)"
echo "feat/j   $J (gated, pushed)"
echo "feat/x   gated $X0 by org-q (no binding for feat/x), pushed after the gate: $X1"
echo "feat/n   bound, no gate, never pushed"
echo "feat/q   bound by org-q, no gate, never pushed"
echo "\$ git -C <tmp>/work ls-remote --symref origin HEAD 'refs/heads/*'"
g ls-remote --symref origin HEAD 'refs/heads/*' | sed "s|$T|<tmp>|g"

bind() { # ts org feature branch
  printf '{"ts":"%s","org_id":"%s","seat_id":"","event":"scope_reserved","worktree":"/wt/%s","details":"paths=internal/%s/ split=demo feature=%s digest=0123456789ab branch=%s approval=per-plan"}\n' "$1" "$2" "$3" "$2" "$3" "$4"
}
gate() { # ts org head branch
  printf '{"ts":"%s","org_id":"%s","seat_id":"","event":"gate_passed","details":"head=%s base=%s base_ref=refs/remotes/origin/main branch=%s by=probe herdr_pane=unset log=\\"/state/gates/x.log\\""}\n' "$1" "$2" "$3" "$C0" "$4"
}
{
  bind 2026-10-14T01:00:01Z org-m m feat/m
  bind 2026-10-14T01:00:02Z org-k k feat/k
  bind 2026-10-14T01:00:03Z org-r r feat/r
  bind 2026-10-14T01:00:04Z org-w w feat/w
  bind 2026-10-14T01:00:05Z org-c c feat/c
  bind 2026-10-14T01:00:06Z org-j j feat/j
  bind 2026-10-14T01:00:07Z org-n n feat/n
  bind 2026-10-14T01:00:08Z org-q q feat/q
  gate 2026-10-14T01:00:11Z org-m "$M" feat/m
  gate 2026-10-14T01:00:12Z org-k "$K" feat/k
  gate 2026-10-14T01:00:13Z org-r "$R" feat/r
  gate 2026-10-14T01:00:14Z org-w "$W" feat/w
  gate 2026-10-14T01:00:15Z org-c "$C" feat/c
  gate 2026-10-14T01:00:16Z org-j "$J" feat/j
  gate 2026-10-14T01:00:17Z org-q "$X0" feat/x
} > "$T/state/manifest.jsonl"
echo "Seeded ledger <tmp>/state/manifest.jsonl:"
command cat "$T/state/manifest.jsonl"

# git wrapper: logs every call; answers only `remote get-url origin` from PROBE_ORIGIN_URL.
cat > "$T/git-wrapper" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/git.log"
if [ $# -eq 3 ] && [ "$1 $2 $3" = "remote get-url origin" ] && [ -n "${PROBE_ORIGIN_URL-}" ]; then
  printf '%s\n' "$PROBE_ORIGIN_URL"; exit 0
fi
exec /opt/homebrew/bin/git "$@"
EOF
# fake gh: --limit defaults to 30 (gh's documented default). Per branch, from
# PROBE_GH_CONF: <branch> <fork PRs newer than its own|FAIL> <own number|-> <state> <base>.
# Output is newest first: the fork PRs (numbers 100+i), then the branch's own
# PR, cut at the limit.
cat > "$T/gh-fake" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/gh.log"
head=""; limit=30
while [ $# -gt 0 ]; do
  case "$1" in --head) head="$2"; shift 2 ;; --limit) limit="$2"; shift 2 ;; *) shift ;; esac
done
exec /usr/bin/awk -v head="$head" -v limit="$limit" -v U="$PROBE_PR_BASE" -v OU="${PROBE_OWN_BASE:-$PROBE_PR_BASE}" '
$1 == head { forks = $2; own = $3; st = $4; base = $5; found = 1 }
END {
  if (forks == "FAIL") { print "To get started with GitHub CLI, please run:  gh auth login" > "/dev/stderr"; exit 4 }
  n = 0; printf "["
  for (i = forks + 0; i >= 1 && n < limit; i--) {
    if (n) printf ","
    printf "{\"number\":%d,\"state\":\"OPEN\",\"url\":\"%s/%d\",\"headRefOid\":\"1111111111111111111111111111111111111111\",\"baseRefName\":\"main\",\"isCrossRepository\":true}", 100 + i, U, 100 + i
    n++
  }
  if (found && own != "-" && n < limit) {
    if (n) printf ","
    printf "{\"number\":%d,\"state\":\"%s\",\"url\":\"%s/%d\",\"headRefOid\":\"0000000000000000000000000000000000000000\",\"baseRefName\":\"%s\",\"isCrossRepository\":false}", own, st, OU, own, base
    n++
  }
  printf "]\n"
}' "$PROBE_GH_CONF"
EOF
chmod +x "$T/git-wrapper" "$T/gh-fake"
cp "$T/git-wrapper" "$T/bin/git"; cp "$T/git-wrapper" "$T/bin-gh/git"; cp "$T/gh-fake" "$T/bin-gh/gh"
echo "\$ command cat <tmp>/git-wrapper"; command cat "$T/git-wrapper"
echo "\$ command cat <tmp>/gh-fake"; command cat "$T/gh-fake"

cat > "$T/conf-main" <<'EOF'
feat/m 1000 19 MERGED main
feat/k 1000 22 OPEN main
feat/r 0 21 OPEN main
feat/w 30 23 OPEN release
feat/c 40 20 CLOSED main
feat/j 999 24 OPEN main
EOF
sed 's/^feat\/k .*/feat\/k 0 22 OPEN main/' "$T/conf-main" > "$T/conf-one"
sed 's/^feat\/j .*/feat\/j FAIL - - -/' "$T/conf-main" > "$T/conf-fail"
for f in conf-main conf-one conf-fail; do echo "\$ command cat <tmp>/$f"; command cat "$T/$f"; done
echo "Fake gh output sizes for conf-main with --limit 1000 (PR count per branch):"
for b in feat/m feat/k feat/r feat/w feat/c feat/j feat/n; do
  n=$(PROBE_LOG_DIR=$T/logs PROBE_GH_CONF=$T/conf-main PROBE_PR_BASE=https://github.com/acme/app/pull "$T/gh-fake" pr list --head "$b" --limit 1000 | python3 -c 'import json,sys; p=json.load(sys.stdin); print(len(p), sum(1 for x in p if not x["isCrossRepository"]))')
  n30=$(PROBE_LOG_DIR=$T/logs PROBE_GH_CONF=$T/conf-main PROBE_PR_BASE=https://github.com/acme/app/pull "$T/gh-fake" pr list --head "$b" | python3 -c 'import json,sys; p=json.load(sys.stdin); print(len(p), sum(1 for x in p if not x["isCrossRepository"]))')
  echo "  $b: --limit 1000 -> total/own $n; no --limit -> total/own $n30"
done
rm -f "$T/logs/gh.log"

snap() {
  echo "Ledger and refs ($1):"
  (cd "$T/state" && shasum -a 256 ./manifest.jsonl && wc -l -c manifest.jsonl && command ls -A)
  g -C "$T/work" for-each-ref --format='%(objectname) %(refname)' | shasum -a 256
  g -C "$T/origin.git" for-each-ref --format='%(objectname) %(refname)' | shasum -a 256
}
snap "before the runs"
touch "$T/marker"; sleep 1

run() { # title bin conf extra-env... -- args...
  title=$1; bin=$2; conf=$3; shift 3
  envs=()
  while [ "$1" != "--" ]; do envs+=("$1"); shift; done
  shift
  : > "$T/logs/gh.log"; : > "$T/logs/git.log"
  echo
  echo "### $title"
  echo
  echo '```'
  shown_env=""
  for e in "${envs[@]}"; do shown_env="$shown_env $e"; done
  echo "\$ cd <tmp>/work && PROBE_GH_CONF=<tmp>/$conf$shown_env $(basename "$bin") org merge-order $* --state-dir <tmp>/state" | sed "s|$T|<tmp>|g"
  (cd "$T/work" && env -i HOME="$T/home" GIT_CONFIG_GLOBAL="$T/gitconfig" GIT_CONFIG_NOSYSTEM=1 \
     PROBE_LOG_DIR="$T/logs" PROBE_GH_CONF="$T/$conf" PROBE_ORIGIN_URL=git@github.com:acme/app.git \
     PROBE_PR_BASE=https://github.com/acme/app/pull "${envs[@]}" \
     "$bin" org merge-order "$@" --state-dir "$T/state" >"$T/out" 2>"$T/err")
  rc=$?
  echo "--- stdout"; sed "s|$T|<tmp>|g" "$T/out"
  echo "--- stderr"; sed "s|$T|<tmp>|g" "$T/err"
  echo "--- exit=$rc"
  echo "--- git calls: $(wc -l < "$T/logs/git.log" | tr -d ' ') (remote get-url origin: $(grep -c '^remote get-url origin$' "$T/logs/git.log"), ls-remote: $(grep -c '^ls-remote' "$T/logs/git.log"))"
  echo "--- gh calls: $(wc -l < "$T/logs/gh.log" | tr -d ' ')"
  sed 's/^/    gh /' "$T/logs/gh.log"
  echo '```'
}

run "Q1 HEAD: 30/40 fork PRs newer than the own PR, 1000 fork PRs (feat/m, feat/k), 999 forks + own (feat/j)" "$HEADBIN" conf-main PATH="$T/bin-gh" --
run "Q2 HEAD: same, --json" "$HEADBIN" conf-main PATH="$T/bin-gh" -- --json
run "Q3 HEAD: same, --all" "$HEADBIN" conf-main PATH="$T/bin-gh" -- --all
run "Q4 HEAD: only feat/m has 1000 fork PRs (one branch: the singular note)" "$HEADBIN" conf-one PATH="$T/bin-gh" --
run "Q5 HEAD: gh fails for feat/j, 1000 fork PRs for feat/m and feat/k (note order)" "$HEADBIN" conf-fail PATH="$T/bin-gh" --
run "Q6 HEAD: the own PRs' url names upstream-org/app (repository check stops after two limit-full branches)" "$HEADBIN" conf-main PATH="$T/bin-gh" PROBE_OWN_BASE=https://github.com/upstream-org/app/pull --
run "Q7 HEAD: no gh on PATH" "$HEADBIN" conf-main PATH="$T/bin" --
run "Q8 pre-fix e463eb9d (negative control): same as Q1" "$PREBIN" conf-main PATH="$T/bin-gh" --

echo
echo "### Ledger, refs and files after Q1-Q8 (AC9)"
echo
echo '```'
snap "after the runs"
echo "Files newer than the marker under state, work, origin.git, home:"
find "$T/state" "$T/work" "$T/origin.git" "$T/home" -newer "$T/marker" | sed "s|$T|<tmp>|g"
echo "(end of list)"
echo '```'
echo "PROBE_TMP=$T" >&2
```
