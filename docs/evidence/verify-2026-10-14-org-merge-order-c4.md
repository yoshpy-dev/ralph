# Verify evidence: ralph org merge-order, cycle 4 (org-merge-order)

- Date: 2026-10-14
- Plan: `docs/plans/active/2026-10-14-org-merge-order.md` (Progress, last bullet: the cycle-4 decisions, including the deviation that widens Scope/C8)
- Report: `docs/reports/verify-2026-10-14-org-merge-order.md` (`## Cycle 4`)
- HEAD: a7cbdd87 (cycle-4 delta: `git diff 999e0394..HEAD`)
- How: two binaries were built with `GOTOOLCHAIN=go1.25.8` before the probe ran: `ralph-head` from the worktree at HEAD, and `ralph-pre` from `git archive 999e0394` (the commit before the cycle-4 fixes, as a negative control). A scratch bash script (its text is in the last section) makes a bare origin and a work repo under a `mktemp -d` dir, seeds `<tmp>/state/manifest.jsonl`, and runs `ralph org merge-order` under `env -i` with HOME set to a temp dir and PATH set to one temp bin dir: `<tmp>/bin` (a git wrapper only) or `<tmp>/bin-gh` (the wrapper plus a fake `gh`). The git wrapper logs each call and answers only `git remote get-url origin` (from `PROBE_ORIGIN_URL`); everything else goes to the real git (`/opt/homebrew/bin/git`, 2.49.0). The fake `gh` logs its argv and prints, for `--head feat/a` only, one non-fork PR whose url starts with `PROBE_PR_BASE`; for any other branch it prints `[]`. V8 runs `ralph org gate` on a copy of the ledger; it refuses before any script runs. Nothing runs the real `gh`, herdr, or osascript, and nothing reaches the network.
- Redaction: the temp dir is `<tmp>`, the session scratchpad `$SP`, and `$HOME` is `~`. SHAs and times are from the run. Below this header, everything is the commands' output as printed (checked with `cmp` against the scratch copy), except the `##` headings and code fences that wrap it, and the two `jq -e .` dumps of the settings files in the static run (148 lines each, replaced by one bracketed line; the full run is the gitignored `docs/evidence/verify-2026-10-14-033350.log`). In the first section, the `### exit=0` after `./scripts/check-sync.sh | tail -8` is check-sync.sh's own exit (zsh `pipestatus[1]`).

## HEAD, plan digest, cycle counter, cycle-4 delta, skill copies and sync

```
$ git rev-parse HEAD
a7cbdd8770f4e23a4fcaaa09e5e8d835ce168c83
$ git status --porcelain
$ ./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-order.md
c0b1e9230999
$ grep -n "^- Status\|^- Approved" docs/plans/active/2026-10-14-org-merge-order.md
3:- Status: Approved
4:- Approved: 2026-10-14 sha256:c0b1e9230999
$ command cat .harness/state/standard-pipeline/cycle-count.json
{"plan_path": "~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-merge-order/docs/plans/active/2026-10-14-org-merge-order.md", "cycle": 2}

$ git log --oneline 999e0394..HEAD
a7cbdd87 docs: record the cycle-4 self-review insight for org-merge-order
2e3d272a docs: add cycle-4 self-review for org-merge-order
1052a12a fix: attach an unreadable gate record to its org's gated branch
813d6d2d fix: say how to fix origin's URL when gh reads a renamed repository
fd1f3f30 docs: record the cycle-4 decisions in the org-merge-order plan
546e9f66 docs: add cross-review triage for org-merge-order (cycle 3)
$ git diff --stat 999e0394..HEAD | tail -1
 14 files changed, 338 insertions(+), 59 deletions(-)
$ git diff 999e0394..HEAD -U0 -- internal/org/mergeorder.go internal/cli/org_merge_order.go | grep "^@@"
@@ -51 +51,5 @@ func newOrgMergeOrderCmd(stateDir, configPath *string) *cobra.Command {
@@ -318,3 +318,6 @@ func (t *mergeTarget) pos() int {
@@ -333,0 +337 @@ func collectMergeTargets(l mergeLedger) (targets []*mergeTarget, unattached int)
@@ -353 +357,3 @@ func collectMergeTargets(l mergeLedger) (targets []*mergeTarget, unattached int)
@@ -355,2 +361,4 @@ func collectMergeTargets(l mergeLedger) (targets []*mergeTarget, unattached int)
@@ -382,2 +390,2 @@ func mergeLedgerNotes(corrupt, unattached int) []string {
@@ -616,2 +624,6 @@ type mergeOriginRepo struct {
@@ -640,2 +652,3 @@ func (o *mergeOriginRepo) check(ctx context.Context, dir string, run MergeOrderE
$ cmp .claude/skills/org/SKILL.md <each of the other 3 copies>
.agents/skills/org/SKILL.md: cmp exit=0
templates/base/.claude/skills/org/SKILL.md: cmp exit=0
templates/base/.agents/skills/org/SKILL.md: cmp exit=0
$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
### exit=0
$ ./scripts/check-sync.sh | tail -8
=== Sync Summary ===
  IDENTICAL:      167
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
### exit=0
```

## Go build and vet (go1.25.8), go1.26.0 version, gofmt, diff --check

```
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
$ GOTOOLCHAIN=go1.25.8 go build ./...
### exit=0
$ GOTOOLCHAIN=go1.25.8 go vet ./...
### exit=0
$ go version
go version go1.26.0 darwin/arm64
$ gofmt -l internal/ cmd/
### exit=0
$ git diff --check ac34ccf3...HEAD
### exit=0
$ git diff --check 999e0394..HEAD
### exit=0
```

## ./scripts/run-static-verify.sh (changed scope)

```
# Verification run
- Timestamp: 2026-10-14T03:33:50Z
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
[148 lines of the pretty-printed settings file omitted; the full run is the gitignored docs/evidence/verify-2026-10-14-033350.log]
    OK
==> jq -e . templates/base/.claude/settings.json
[148 lines of the pretty-printed settings file omitted; the full run is the gitignored docs/evidence/verify-2026-10-14-033350.log]
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
secret-scan-branch: scanned ac34ccf3..a7cbdd87 against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-14-033350.log
### exit=0
```

## Probe setup

```
TMP=<tmp>
worktree HEAD: a7cbdd8770f4e23a4fcaaa09e5e8d835ce168c83
binaries: GOTOOLCHAIN=go1.25.8 go build -o <bin>/ralph-head ./cmd/ralph (HEAD); the same from git archive 999e0394 -> <bin>/ralph-pre (pre-fix)
sha256 of the binaries:
ab8f44997fdf9c58cfd9462b5c374ba2a2c20bfb3148f7de00bb553a16aecf8c  ralph-head
b97b743546cd005753566e8f408c5636b6846246295e5ec3952641ad0d9120f4  ralph-pre
main     7f5248c494683dbb91637a676c6a5e872483a70b
feat/a   b2ce2e50ac80794601668d242167dd4294b6ee10 (org-a: bound, gated, pushed)
feat/c   9b63b9236588789d267d09a876a9f2edf3c90100 (org-c: no binding; readable gate_passed, then a gate_failed whose head= is not a SHA)
feat/e   023a929f92d686c606f95a2a73884f0f0582356a (org-e: no binding; readable gate_passed)
feat/e2  7e3e980d64303d8a0ec4bddf00c7c114ab9caed4 (org-e: readable gate_passed after feat/e's, then a gate_failed whose head= is not a SHA)
feat/f   f43295ef98ade7affc8a7662c7e80cdda4277e3f (org-f: no binding; readable gate_passed, then a gate_failed event whose details have no reason=)
feat/x   4c1ee9d851c7d75f02fee6271be2ebe963203ff7 (org-b: readable gate_passed for feat/x before org-b's binding)
feat/b   d484dbc968caa08adcf4d98efdae50ea7c13b57b (org-b: bound to feat/b after the feat/x gate, then a gate_failed whose head= is not a SHA)
org-d    only a gate_passed whose details cannot be read (no binding, no earlier readable record)
$ git -C <tmp>/work ls-remote --symref origin HEAD 'refs/heads/*'
ref: refs/heads/main	HEAD
7f5248c494683dbb91637a676c6a5e872483a70b	HEAD
b2ce2e50ac80794601668d242167dd4294b6ee10	refs/heads/feat/a
d484dbc968caa08adcf4d98efdae50ea7c13b57b	refs/heads/feat/b
9b63b9236588789d267d09a876a9f2edf3c90100	refs/heads/feat/c
023a929f92d686c606f95a2a73884f0f0582356a	refs/heads/feat/e
7e3e980d64303d8a0ec4bddf00c7c114ab9caed4	refs/heads/feat/e2
f43295ef98ade7affc8a7662c7e80cdda4277e3f	refs/heads/feat/f
4c1ee9d851c7d75f02fee6271be2ebe963203ff7	refs/heads/feat/x
7f5248c494683dbb91637a676c6a5e872483a70b	refs/heads/main
Seeded ledger <tmp>/state/manifest.jsonl:
{"ts":"2026-10-14T01:00:01Z","org_id":"org-a","seat_id":"","event":"scope_reserved","worktree":"/wt/a","details":"paths=internal/org-a/ split=demo feature=a digest=0123456789ab branch=feat/a approval=per-plan"}
{"ts":"2026-10-14T01:00:11Z","org_id":"org-a","seat_id":"","event":"gate_passed","details":"head=b2ce2e50ac80794601668d242167dd4294b6ee10 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/a by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:12Z","org_id":"org-c","seat_id":"","event":"gate_passed","details":"head=9b63b9236588789d267d09a876a9f2edf3c90100 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/c by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:13Z","org_id":"org-c","seat_id":"","event":"gate_failed","details":"reason=static_failed head=not-a-sha base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/c by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:14Z","org_id":"org-d","seat_id":"","event":"gate_passed","details":"head=not-a-sha garbled"}
{"ts":"2026-10-14T01:00:15Z","org_id":"org-e","seat_id":"","event":"gate_passed","details":"head=023a929f92d686c606f95a2a73884f0f0582356a base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/e by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:16Z","org_id":"org-e","seat_id":"","event":"gate_passed","details":"head=7e3e980d64303d8a0ec4bddf00c7c114ab9caed4 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/e2 by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:17Z","org_id":"org-e","seat_id":"","event":"gate_failed","details":"reason=static_failed head=not-a-sha base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/e2 by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:18Z","org_id":"org-f","seat_id":"","event":"gate_passed","details":"head=f43295ef98ade7affc8a7662c7e80cdda4277e3f base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/f by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:19Z","org_id":"org-f","seat_id":"","event":"gate_failed","details":"head=f43295ef98ade7affc8a7662c7e80cdda4277e3f base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/f by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:20Z","org_id":"org-b","seat_id":"","event":"gate_passed","details":"head=4c1ee9d851c7d75f02fee6271be2ebe963203ff7 base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/x by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:21Z","org_id":"org-b","seat_id":"","event":"scope_reserved","worktree":"/wt/b","details":"paths=internal/org-b/ split=demo feature=b digest=0123456789ab branch=feat/b approval=per-plan"}
{"ts":"2026-10-14T01:00:22Z","org_id":"org-b","seat_id":"","event":"gate_failed","details":"reason=static_failed head=not-a-sha base=7f5248c494683dbb91637a676c6a5e872483a70b base_ref=refs/remotes/origin/main branch=feat/b by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
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
head=""
while [ $# -gt 0 ]; do
  case "$1" in --head) head="$2"; shift 2 ;; *) shift ;; esac
done
exec /usr/bin/awk -v head="$head" -v U="$PROBE_PR_BASE" '
$1 == head { n = $2; st = $3; base = $4; found = 1 }
END {
  if (found) printf "[{\"number\":%d,\"state\":\"%s\",\"url\":\"%s/%d\",\"headRefOid\":\"0000000000000000000000000000000000000000\",\"baseRefName\":\"%s\",\"isCrossRepository\":false}]\n", n, st, U, n, base
  else print "[]"
}' "$PROBE_GH_CONF"
$ command cat <tmp>/conf
feat/a 7 MERGED main
Ledger and refs (before the runs):
7a19259295370e04982cc5d94e3cf3fe0ee49f2c84372e47c7d5f442e7d7a82a  ./manifest.jsonl
      13    3381 manifest.jsonl
manifest.jsonl
7061cbf11b74ca99015091c5706c5c180a6735380f4b88406089b78fe6bb3776  -
272845dfacadccc3616e640b8c9e1873f081fe5b2c3770fa5f71b90d9bd91010  -
```

## Probe runs

### V1 HEAD: gh's PR url names acme/app-v2 (origin's repository renamed on GitHub)

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/acme/app-v2/pull ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T03:36:32Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/a	-	b2ce2e50ac80	7f5248c49468	org-a/a	the gate passed on b2ce2e50ac80, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/e	-	023a929f92d6	7f5248c49468	-	the gate passed on 023a929f92d6, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/x	-	4c1ee9d851c7	7f5248c49468	-	the gate passed on 4c1ee9d851c7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	unreadable	feat/c	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:13Z) cannot be read: run ralph org gate --org-id org-c again in the feature worktree (the gate runs only while the feature org runs)
5	unreadable	feat/e2	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:17Z) cannot be read: run ralph org gate --org-id org-e again in the feature worktree (the gate runs only while the feature org runs)
6	unreadable	feat/f	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:19Z) cannot be read: run ralph org gate --org-id org-f again in the feature worktree (the gate runs only while the feature org runs)
7	unreadable	feat/b	-	-	-	org-b/b	the latest gate record (gate_failed at 2026-10-14T01:00:22Z) cannot be read: run ralph org gate --org-id org-b again in the feature worktree (the gate runs only while the feature org runs)
note: gh reads acme/app-v2, not origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked; if acme/app was renamed to acme/app-v2, point origin at it (git remote set-url origin <its URL>); otherwise run gh repo set-default acme/app in this repository
note: 1 gate record(s) cannot be read and their org had neither a feature binding nor a readable gate record before them, so they are attached to no branch
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T03:36:32Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 1
    gh pr list --head feat/a --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### V2 HEAD: gh's PR url names upstream-org/app (another repository)

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/upstream-org/app/pull ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T03:36:33Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/a	-	b2ce2e50ac80	7f5248c49468	org-a/a	the gate passed on b2ce2e50ac80, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/e	-	023a929f92d6	7f5248c49468	-	the gate passed on 023a929f92d6, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/x	-	4c1ee9d851c7	7f5248c49468	-	the gate passed on 4c1ee9d851c7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	unreadable	feat/c	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:13Z) cannot be read: run ralph org gate --org-id org-c again in the feature worktree (the gate runs only while the feature org runs)
5	unreadable	feat/e2	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:17Z) cannot be read: run ralph org gate --org-id org-e again in the feature worktree (the gate runs only while the feature org runs)
6	unreadable	feat/f	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:19Z) cannot be read: run ralph org gate --org-id org-f again in the feature worktree (the gate runs only while the feature org runs)
7	unreadable	feat/b	-	-	-	org-b/b	the latest gate record (gate_failed at 2026-10-14T01:00:22Z) cannot be read: run ralph org gate --org-id org-b again in the feature worktree (the gate runs only while the feature org runs)
note: gh reads upstream-org/app, not origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked; if acme/app was renamed to upstream-org/app, point origin at it (git remote set-url origin <its URL>); otherwise run gh repo set-default acme/app in this repository
note: 1 gate record(s) cannot be read and their org had neither a feature binding nor a readable gate record before them, so they are attached to no branch
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T03:36:33Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 1
    gh pr list --head feat/a --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### V3 HEAD: origin pointed at acme/app-v2 (the note's git remote set-url origin fix), PR url acme/app-v2

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app-v2.git PROBE_PR_BASE=https://github.com/acme/app-v2/pull ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T03:36:33Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/e	-	023a929f92d6	7f5248c49468	-	the gate passed on 023a929f92d6, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/x	-	4c1ee9d851c7	7f5248c49468	-	the gate passed on 4c1ee9d851c7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	unreadable	feat/c	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:13Z) cannot be read: run ralph org gate --org-id org-c again in the feature worktree (the gate runs only while the feature org runs)
4	unreadable	feat/e2	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:17Z) cannot be read: run ralph org gate --org-id org-e again in the feature worktree (the gate runs only while the feature org runs)
5	unreadable	feat/f	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:19Z) cannot be read: run ralph org gate --org-id org-f again in the feature worktree (the gate runs only while the feature org runs)
6	unreadable	feat/b	-	-	-	org-b/b	the latest gate record (gate_failed at 2026-10-14T01:00:22Z) cannot be read: run ralph org gate --org-id org-b again in the feature worktree (the gate runs only while the feature org runs)
note: 1 gate record(s) cannot be read and their org had neither a feature binding nor a readable gate record before them, so they are attached to no branch
note: 1 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: 2 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T03:36:33Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 7
    gh pr list --head feat/a --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/e --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/e2 --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/b --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### V4 pre-fix 999e0394 (negative control): same as V1

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/acme/app-v2/pull ralph-pre org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T03:36:34Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/a	-	b2ce2e50ac80	7f5248c49468	org-a/a	the gate passed on b2ce2e50ac80, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/c	-	9b63b9236588	7f5248c49468	-	the gate passed on 9b63b9236588, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/e	-	023a929f92d6	7f5248c49468	-	the gate passed on 023a929f92d6, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	ready	feat/e2	-	7e3e980d6430	7f5248c49468	-	the gate passed on 7e3e980d6430, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
5	ready	feat/f	-	f43295ef98ad	7f5248c49468	-	the gate passed on f43295ef98ad, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
6	ready	feat/x	-	4c1ee9d851c7	7f5248c49468	-	the gate passed on 4c1ee9d851c7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
7	unreadable	feat/b	-	-	-	org-b/b	the latest gate record (gate_failed at 2026-10-14T01:00:22Z) cannot be read: run ralph org gate --org-id org-b again in the feature worktree (the gate runs only while the feature org runs)
note: gh reads acme/app-v2, not origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked; run gh repo set-default acme/app in this repository
note: 4 gate record(s) cannot be read and their org had no feature binding before them, so they are attached to no branch
note: 6 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T03:36:34Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 1
    gh pr list --head feat/a --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### V5 HEAD: no gh on PATH (unreadable records of orgs with no binding)

```
$ cd <tmp>/work && PATH=<tmp>/bin PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=- ralph-head org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T03:36:34Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/a	-	b2ce2e50ac80	7f5248c49468	org-a/a	the gate passed on b2ce2e50ac80, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/e	-	023a929f92d6	7f5248c49468	-	the gate passed on 023a929f92d6, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/x	-	4c1ee9d851c7	7f5248c49468	-	the gate passed on 4c1ee9d851c7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	unreadable	feat/c	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:13Z) cannot be read: run ralph org gate --org-id org-c again in the feature worktree (the gate runs only while the feature org runs)
5	unreadable	feat/e2	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:17Z) cannot be read: run ralph org gate --org-id org-e again in the feature worktree (the gate runs only while the feature org runs)
6	unreadable	feat/f	-	-	-	-	the latest gate record (gate_failed at 2026-10-14T01:00:19Z) cannot be read: run ralph org gate --org-id org-f again in the feature worktree (the gate runs only while the feature org runs)
7	unreadable	feat/b	-	-	-	org-b/b	the latest gate record (gate_failed at 2026-10-14T01:00:22Z) cannot be read: run ralph org gate --org-id org-b again in the feature worktree (the gate runs only while the feature org runs)
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 1 gate record(s) cannot be read and their org had neither a feature binding nor a readable gate record before them, so they are attached to no branch
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T03:36:34Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 1 (remote get-url origin: 0, ls-remote: 1)
--- gh calls: 0
```

### V6 HEAD: same, --json

```
$ cd <tmp>/work && PATH=<tmp>/bin PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=- ralph-head org merge-order --json --state-dir <tmp>/state
--- stdout
{
  "default_branch": "main",
  "default_head": "7f5248c494683dbb91637a676c6a5e872483a70b",
  "listed_at": "2026-10-14T03:36:35Z",
  "gh_available": false,
  "entries": [
    {
      "rank": 1,
      "class": "ready",
      "branch": "feat/a",
      "remote_head": "b2ce2e50ac80794601668d242167dd4294b6ee10",
      "detail": "the gate passed on b2ce2e50ac80, origin's tip of the branch, with base 7f5248c49468, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "b2ce2e50ac80794601668d242167dd4294b6ee10",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:11Z"
      },
      "binding": {
        "org_id": "org-a",
        "feature": "a",
        "split": "demo",
        "ts": "2026-10-14T01:00:01Z"
      }
    },
    {
      "rank": 2,
      "class": "ready",
      "branch": "feat/e",
      "remote_head": "023a929f92d686c606f95a2a73884f0f0582356a",
      "detail": "the gate passed on 023a929f92d6, origin's tip of the branch, with base 7f5248c49468, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "023a929f92d686c606f95a2a73884f0f0582356a",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:15Z"
      }
    },
    {
      "rank": 3,
      "class": "ready",
      "branch": "feat/x",
      "remote_head": "4c1ee9d851c7d75f02fee6271be2ebe963203ff7",
      "detail": "the gate passed on 4c1ee9d851c7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "4c1ee9d851c7d75f02fee6271be2ebe963203ff7",
        "base": "7f5248c494683dbb91637a676c6a5e872483a70b",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:20Z"
      }
    },
    {
      "rank": 4,
      "class": "unreadable",
      "branch": "feat/c",
      "remote_head": "9b63b9236588789d267d09a876a9f2edf3c90100",
      "detail": "the latest gate record (gate_failed at 2026-10-14T01:00:13Z) cannot be read: run ralph org gate --org-id org-c again in the feature worktree (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_failed",
        "readable": false,
        "head": "",
        "base": "",
        "base_ref": "",
        "reason": "",
        "ts": "2026-10-14T01:00:13Z"
      }
    },
    {
      "rank": 5,
      "class": "unreadable",
      "branch": "feat/e2",
      "remote_head": "7e3e980d64303d8a0ec4bddf00c7c114ab9caed4",
      "detail": "the latest gate record (gate_failed at 2026-10-14T01:00:17Z) cannot be read: run ralph org gate --org-id org-e again in the feature worktree (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_failed",
        "readable": false,
        "head": "",
        "base": "",
        "base_ref": "",
        "reason": "",
        "ts": "2026-10-14T01:00:17Z"
      }
    },
    {
      "rank": 6,
      "class": "unreadable",
      "branch": "feat/f",
      "remote_head": "f43295ef98ade7affc8a7662c7e80cdda4277e3f",
      "detail": "the latest gate record (gate_failed at 2026-10-14T01:00:19Z) cannot be read: run ralph org gate --org-id org-f again in the feature worktree (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_failed",
        "readable": false,
        "head": "",
        "base": "",
        "base_ref": "",
        "reason": "",
        "ts": "2026-10-14T01:00:19Z"
      }
    },
    {
      "rank": 7,
      "class": "unreadable",
      "branch": "feat/b",
      "remote_head": "d484dbc968caa08adcf4d98efdae50ea7c13b57b",
      "detail": "the latest gate record (gate_failed at 2026-10-14T01:00:22Z) cannot be read: run ralph org gate --org-id org-b again in the feature worktree (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_failed",
        "readable": false,
        "head": "",
        "base": "",
        "base_ref": "",
        "reason": "",
        "ts": "2026-10-14T01:00:22Z"
      },
      "binding": {
        "org_id": "org-b",
        "feature": "b",
        "split": "demo",
        "ts": "2026-10-14T01:00:21Z"
      }
    }
  ],
  "notes": [
    "gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked",
    "1 gate record(s) cannot be read and their org had neither a feature binding nor a readable gate record before them, so they are attached to no branch",
    "3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)",
    "the order is as of the ls-remote at 2026-10-14T03:36:35Z and is not a guarantee: origin/main can move before a merge"
  ]
}
--- stderr
--- exit=0
--- git calls: 1 (remote get-url origin: 0, ls-remote: 1)
--- gh calls: 0
```

### V7 pre-fix 999e0394 (negative control): same as V5

```
$ cd <tmp>/work && PATH=<tmp>/bin PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=- ralph-pre org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 7f5248c49468 (listed 2026-10-14T03:36:35Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/a	-	b2ce2e50ac80	7f5248c49468	org-a/a	the gate passed on b2ce2e50ac80, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
2	ready	feat/c	-	9b63b9236588	7f5248c49468	-	the gate passed on 9b63b9236588, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
3	ready	feat/e	-	023a929f92d6	7f5248c49468	-	the gate passed on 023a929f92d6, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
4	ready	feat/e2	-	7e3e980d6430	7f5248c49468	-	the gate passed on 7e3e980d6430, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
5	ready	feat/f	-	f43295ef98ad	7f5248c49468	-	the gate passed on f43295ef98ad, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
6	ready	feat/x	-	4c1ee9d851c7	7f5248c49468	-	the gate passed on 4c1ee9d851c7, origin's tip of the branch, with base 7f5248c49468, origin/main's tip
7	unreadable	feat/b	-	-	-	org-b/b	the latest gate record (gate_failed at 2026-10-14T01:00:22Z) cannot be read: run ralph org gate --org-id org-b again in the feature worktree (the gate runs only while the feature org runs)
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 4 gate record(s) cannot be read and their org had no feature binding before them, so they are attached to no branch
note: 6 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-14T03:36:35Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 1 (remote get-url origin: 0, ls-remote: 1)
--- gh calls: 0
```

### Ledger, refs and files after V1-V7 (AC9)

```
Ledger and refs (after V1-V7):
7a19259295370e04982cc5d94e3cf3fe0ee49f2c84372e47c7d5f442e7d7a82a  ./manifest.jsonl
      13    3381 manifest.jsonl
manifest.jsonl
7061cbf11b74ca99015091c5706c5c180a6735380f4b88406089b78fe6bb3776  -
272845dfacadccc3616e640b8c9e1873f081fe5b2c3770fa5f71b90d9bd91010  -
files newer than the marker under state, work, origin.git, home:
(end of list)
```

### V8 HEAD: the Detail's command for feat/c, ralph org gate --org-id org-c, on a copy of the ledger (self-review C4-1)

```
7a19259295370e04982cc5d94e3cf3fe0ee49f2c84372e47c7d5f442e7d7a82a  manifest.jsonl
manifest.jsonl
$ cd <tmp>/work && PATH=<tmp>/bin ralph-head org gate --org-id org-c --state-dir <tmp>/state-gate
--- stdout
--- stderr
org: gate: org_id "org-c" is not running and holds no split plan feature binding: name the org_id of a feature org that ralph org start --plan started (ralph org status lists them)
--- exit=1
7a19259295370e04982cc5d94e3cf3fe0ee49f2c84372e47c7d5f442e7d7a82a  manifest.jsonl
manifest.jsonl
```

### V9 merge-order --help: HEAD vs pre-fix 999e0394

```
$ diff <(ralph-pre org merge-order --help) <(ralph-head org merge-order --help)
21c21,25
< gh repo set-default. The check has three gaps: when origin's URL names
---
> gh repo set-default. A repository renamed on GitHub while origin keeps
> the old URL looks the same (git reaches it through GitHub's redirect,
> gh's PR urls carry the new name), so the note also says to point
> origin at the new name with git remote set-url origin if it was
> renamed. The check has three gaps: when origin's URL names
--- diff exit=1
```

## Probe script

```bash
#!/bin/bash
# Cycle-4 probe of ralph org merge-order (org-merge-order verify).
# Scratch only: bare origin + work repo in a temp dir, a fake gh, and a git
# wrapper. No herdr, no osascript, no network, no real gh.
set -u
SP=$SP
HEADBIN=$SP/c4/ralph-head
PREBIN=$SP/c4/ralph-pre
REALGIT=/opt/homebrew/bin/git
T=$(mktemp -d "$SP/c4/probe.XXXXXX")
mkdir -p "$T/home" "$T/logs" "$T/state" "$T/bin" "$T/bin-gh"
printf '[user]\n\tname = probe\n\temail = probe@example.invalid\n[init]\n\tdefaultBranch = main\n' > "$T/gitconfig"
export GIT_CONFIG_GLOBAL="$T/gitconfig" GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_DATE='2026-10-14T00:00:00Z' GIT_COMMITTER_DATE='2026-10-14T00:00:00Z'
g() { "$REALGIT" "$@"; }

echo "TMP=<tmp>"
echo "worktree HEAD: $(g -C ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-merge-order rev-parse HEAD)"
echo "binaries: GOTOOLCHAIN=go1.25.8 go build -o <bin>/ralph-head ./cmd/ralph (HEAD); the same from git archive 999e0394 -> <bin>/ralph-pre (pre-fix)"
echo "sha256 of the binaries:"
(cd "$SP/c4" && shasum -a 256 ralph-head ralph-pre)

g init -q --bare "$T/origin.git"
g init -q -b main "$T/work"
cd "$T/work" || exit 1
g remote add origin "$T/origin.git"
EMPTY=$(g mktree </dev/null)
C0=$(g commit-tree -m main "$EMPTY")
g update-ref refs/heads/main "$C0"
mk() { g commit-tree -p "$1" -m "$2" "$1^{tree}"; }
A=$(mk "$C0" a); C=$(mk "$C0" c); E=$(mk "$C0" e); E2=$(mk "$C0" e2)
F=$(mk "$C0" f); X=$(mk "$C0" x); B=$(mk "$C0" b)
g push -q origin "$C0:refs/heads/main" "$A:refs/heads/feat/a" "$C:refs/heads/feat/c" \
  "$E:refs/heads/feat/e" "$E2:refs/heads/feat/e2" "$F:refs/heads/feat/f" "$X:refs/heads/feat/x" "$B:refs/heads/feat/b"
echo "main     $C0"
echo "feat/a   $A (org-a: bound, gated, pushed)"
echo "feat/c   $C (org-c: no binding; readable gate_passed, then a gate_failed whose head= is not a SHA)"
echo "feat/e   $E (org-e: no binding; readable gate_passed)"
echo "feat/e2  $E2 (org-e: readable gate_passed after feat/e's, then a gate_failed whose head= is not a SHA)"
echo "feat/f   $F (org-f: no binding; readable gate_passed, then a gate_failed event whose details have no reason=)"
echo "feat/x   $X (org-b: readable gate_passed for feat/x before org-b's binding)"
echo "feat/b   $B (org-b: bound to feat/b after the feat/x gate, then a gate_failed whose head= is not a SHA)"
echo "org-d    only a gate_passed whose details cannot be read (no binding, no earlier readable record)"
echo "\$ git -C <tmp>/work ls-remote --symref origin HEAD 'refs/heads/*'"
g ls-remote --symref origin HEAD 'refs/heads/*' | sed "s|$T|<tmp>|g"

bind() { # ts org feature branch
  printf '{"ts":"%s","org_id":"%s","seat_id":"","event":"scope_reserved","worktree":"/wt/%s","details":"paths=internal/%s/ split=demo feature=%s digest=0123456789ab branch=%s approval=per-plan"}\n' "$1" "$2" "$3" "$2" "$3" "$4"
}
passed() { # ts org head branch
  printf '{"ts":"%s","org_id":"%s","seat_id":"","event":"gate_passed","details":"head=%s base=%s base_ref=refs/remotes/origin/main branch=%s by=probe herdr_pane=unset log=\\"/state/gates/x.log\\""}\n' "$1" "$2" "$3" "$C0" "$4"
}
badfailed() { # ts org branch: a gate_failed whose head= is not a SHA
  printf '{"ts":"%s","org_id":"%s","seat_id":"","event":"gate_failed","details":"reason=static_failed head=not-a-sha base=%s base_ref=refs/remotes/origin/main branch=%s by=probe herdr_pane=unset log=\\"/state/gates/x.log\\""}\n' "$1" "$2" "$C0" "$3"
}
mismatch() { # ts org head branch: a gate_failed event whose details are a pass (no reason=)
  printf '{"ts":"%s","org_id":"%s","seat_id":"","event":"gate_failed","details":"head=%s base=%s base_ref=refs/remotes/origin/main branch=%s by=probe herdr_pane=unset log=\\"/state/gates/x.log\\""}\n' "$1" "$2" "$3" "$C0" "$4"
}
{
  bind      2026-10-14T01:00:01Z org-a a feat/a
  passed    2026-10-14T01:00:11Z org-a "$A" feat/a
  passed    2026-10-14T01:00:12Z org-c "$C" feat/c
  badfailed 2026-10-14T01:00:13Z org-c feat/c
  printf '{"ts":"2026-10-14T01:00:14Z","org_id":"org-d","seat_id":"","event":"gate_passed","details":"head=not-a-sha garbled"}\n'
  passed    2026-10-14T01:00:15Z org-e "$E" feat/e
  passed    2026-10-14T01:00:16Z org-e "$E2" feat/e2
  badfailed 2026-10-14T01:00:17Z org-e feat/e2
  passed    2026-10-14T01:00:18Z org-f "$F" feat/f
  mismatch  2026-10-14T01:00:19Z org-f "$F" feat/f
  passed    2026-10-14T01:00:20Z org-b "$X" feat/x
  bind      2026-10-14T01:00:21Z org-b b feat/b
  badfailed 2026-10-14T01:00:22Z org-b feat/b
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
# fake gh: logs its argv; per branch, from PROBE_GH_CONF: <branch> <number> <state> <base>.
# Prints that one PR (url $PROBE_PR_BASE/<number>, not from a fork) or [].
cat > "$T/gh-fake" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/gh.log"
head=""
while [ $# -gt 0 ]; do
  case "$1" in --head) head="$2"; shift 2 ;; *) shift ;; esac
done
exec /usr/bin/awk -v head="$head" -v U="$PROBE_PR_BASE" '
$1 == head { n = $2; st = $3; base = $4; found = 1 }
END {
  if (found) printf "[{\"number\":%d,\"state\":\"%s\",\"url\":\"%s/%d\",\"headRefOid\":\"0000000000000000000000000000000000000000\",\"baseRefName\":\"%s\",\"isCrossRepository\":false}]\n", n, st, U, n, base
  else print "[]"
}' "$PROBE_GH_CONF"
EOF
chmod +x "$T/git-wrapper" "$T/gh-fake"
cp "$T/git-wrapper" "$T/bin/git"; cp "$T/git-wrapper" "$T/bin-gh/git"; cp "$T/gh-fake" "$T/bin-gh/gh"
echo "\$ command cat <tmp>/git-wrapper"; command cat "$T/git-wrapper"
echo "\$ command cat <tmp>/gh-fake"; command cat "$T/gh-fake"
printf 'feat/a 7 MERGED main\n' > "$T/conf"
echo "\$ command cat <tmp>/conf"; command cat "$T/conf"

snap() {
  echo "Ledger and refs ($1):"
  (cd "$T/state" && shasum -a 256 ./manifest.jsonl && wc -l -c manifest.jsonl && command ls -A)
  g -C "$T/work" for-each-ref --format='%(objectname) %(refname)' | shasum -a 256
  g -C "$T/origin.git" for-each-ref --format='%(objectname) %(refname)' | shasum -a 256
}
snap "before the runs"
touch "$T/marker"; sleep 1

run() { # title bin path origin-url pr-base -- args...
  title=$1; bin=$2; path=$3; ourl=$4; prbase=$5; shift 5
  [ "$1" = "--" ] && shift
  : > "$T/logs/gh.log"; : > "$T/logs/git.log"
  echo
  echo "### $title"
  echo
  echo '```'
  echo "\$ cd <tmp>/work && PATH=$path PROBE_ORIGIN_URL=$ourl PROBE_PR_BASE=$prbase $(basename "$bin") org merge-order $* --state-dir <tmp>/state" | sed "s|$T|<tmp>|g"
  (cd "$T/work" && env -i HOME="$T/home" GIT_CONFIG_GLOBAL="$T/gitconfig" GIT_CONFIG_NOSYSTEM=1 \
     PATH="$path" PROBE_LOG_DIR="$T/logs" PROBE_GH_CONF="$T/conf" PROBE_ORIGIN_URL="$ourl" PROBE_PR_BASE="$prbase" \
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

OURL=git@github.com:acme/app.git
run "V1 HEAD: gh's PR url names acme/app-v2 (origin's repository renamed on GitHub)" "$HEADBIN" "$T/bin-gh" "$OURL" https://github.com/acme/app-v2/pull --
run "V2 HEAD: gh's PR url names upstream-org/app (another repository)" "$HEADBIN" "$T/bin-gh" "$OURL" https://github.com/upstream-org/app/pull --
run "V3 HEAD: origin pointed at acme/app-v2 (the note's git remote set-url origin fix), PR url acme/app-v2" "$HEADBIN" "$T/bin-gh" git@github.com:acme/app-v2.git https://github.com/acme/app-v2/pull --
run "V4 pre-fix 999e0394 (negative control): same as V1" "$PREBIN" "$T/bin-gh" "$OURL" https://github.com/acme/app-v2/pull --
run "V5 HEAD: no gh on PATH (unreadable records of orgs with no binding)" "$HEADBIN" "$T/bin" "$OURL" - --
run "V6 HEAD: same, --json" "$HEADBIN" "$T/bin" "$OURL" - -- --json
run "V7 pre-fix 999e0394 (negative control): same as V5" "$PREBIN" "$T/bin" "$OURL" - --

echo
echo "### Ledger, refs and files after V1-V7 (AC9)"
echo
echo '```'
snap "after V1-V7"
echo "files newer than the marker under state, work, origin.git, home:"
find "$T/state" "$T/work" "$T/origin.git" "$T/home" -newer "$T/marker" | sed "s|$T|<tmp>|g"
echo "(end of list)"
echo '```'

echo
echo "### V8 HEAD: the Detail's command for feat/c, ralph org gate --org-id org-c, on a copy of the ledger (self-review C4-1)"
echo
echo '```'
cp -R "$T/state" "$T/state-gate"
(cd "$T/state-gate" && shasum -a 256 manifest.jsonl && command ls -A)
echo "\$ cd <tmp>/work && PATH=<tmp>/bin ralph-head org gate --org-id org-c --state-dir <tmp>/state-gate"
(cd "$T/work" && env -i HOME="$T/home" GIT_CONFIG_GLOBAL="$T/gitconfig" GIT_CONFIG_NOSYSTEM=1 \
   PATH="$T/bin" PROBE_LOG_DIR="$T/logs" "$HEADBIN" org gate --org-id org-c --state-dir "$T/state-gate" >"$T/out" 2>"$T/err")
rc=$?
echo "--- stdout"; sed "s|$T|<tmp>|g" "$T/out"
echo "--- stderr"; sed "s|$T|<tmp>|g" "$T/err"
echo "--- exit=$rc"
(cd "$T/state-gate" && shasum -a 256 manifest.jsonl && command ls -A)
echo '```'

echo
echo "### V9 merge-order --help: HEAD vs pre-fix 999e0394"
echo
echo '```'
echo "\$ diff <(ralph-pre org merge-order --help) <(ralph-head org merge-order --help)"
diff <(env -i HOME="$T/home" PATH="$T/bin" "$PREBIN" org merge-order --help 2>&1) <(env -i HOME="$T/home" PATH="$T/bin" "$HEADBIN" org merge-order --help 2>&1)
echo "--- diff exit=$?"
echo '```'
```
