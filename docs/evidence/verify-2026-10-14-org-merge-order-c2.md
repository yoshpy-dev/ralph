# Verify evidence: ralph org merge-order, cycle 2 (org-merge-order)

- Date: 2026-10-14
- Plan: `docs/plans/active/2026-10-14-org-merge-order.md` (Progress, last bullet: the cycle-2 repository check)
- Report: `docs/reports/verify-2026-10-14-org-merge-order.md` (`## Cycle 2`)
- HEAD: fdb457a8 (cycle-2 delta: `git diff cab2fd50..HEAD`)
- How: scratch bash scripts (not committed) build the HEAD binary with `GOTOOLCHAIN=go1.25.8`, make a bare origin and a work repo under a `mktemp -d` dir, seed `<tmp>/state/manifest.jsonl`, and run `ralph org merge-order` under `env -i` with PATH set to one temp bin dir: `<tmp>/bin` (a git wrapper only), `<tmp>/bin-gh` (the wrapper plus a fake `gh` that prints canned JSON per `--head`, its PR url base from `PROBE_PR_BASE`), or `<tmp>/bin-ghfail` (the wrapper plus a `gh` that exits 4). The git wrapper logs each call and, when `PROBE_ORIGIN_URL` is set, prints it for `git remote get-url origin` (exits 2 for `FAIL`); every other git call, ls-remote included, goes to the real git and the local bare origin. With `PROBE_ORIGIN_URL` empty (K11), get-url is the real git too. No network, no real gh; HOME is an empty `<tmp>/home`, and git reads no global or system config.
- Redaction: the temp dir is `<tmp>`, the session scratchpad `$SP`, and `$HOME` is `~`. SHAs and times are from the run. Below this header, everything is the commands' output as printed (checked with `cmp` against the scratch copy), except the `##` headings and code fences that wrap it, and the two `jq -e .` dumps of the settings files in the static run (148 lines each, replaced by one bracketed line; the full run is in the gitignored `docs/evidence/verify-2026-10-13-223656.log`, which `diff` shows equal to the captured output apart from the last three lines it does not hold).
- Note: the feat/f `gate_failed` record uses `reason=static_failed`. A first attempt with `reason=lint_failed` (not a reason `gateRecordFromDetails` reads) made feat/f `unreadable`, as AC4 says; that run was discarded and is not below.

## Go build and vet, gofmt, skill copies, diff --check, plan digest

```
$ git rev-parse HEAD
fdb457a8e2a1841066e9ef3c2ba32a72c9262b37
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
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
$ for f in .agents/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md; do cmp .claude/skills/org/SKILL.md "$f" && echo "same: $f"; done
same: .agents/skills/org/SKILL.md
same: templates/base/.claude/skills/org/SKILL.md
same: templates/base/.agents/skills/org/SKILL.md
$ git diff --check ac34ccf3...HEAD
rc=0
$ ./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-order.md
c0b1e9230999
$ git remote get-url origin
ssh://git@github.com/yoshpy-dev/ralph.git
```

## ./scripts/run-static-verify.sh (changed scope)

```
$ ./scripts/run-static-verify.sh
# Verification run
- Timestamp: 2026-10-13T22:36:56Z
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
[148 lines elided: jq pretty-printed the file; full output in docs/evidence/verify-2026-10-13-223656.log, gitignored]
    OK
==> jq -e . templates/base/.claude/settings.json
[148 lines elided: jq pretty-printed the file; full output in docs/evidence/verify-2026-10-13-223656.log, gitignored]
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
secret-scan-branch: scanned ac34ccf3..fdb457a8 against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-13-223656.log
rc=0
```

## Probe setup

```
TMP=<tmp>
worktree HEAD: fdb457a8e2a1841066e9ef3c2ba32a72c9262b37
binary: GOTOOLCHAIN=go1.25.8 go build -o <tmp>/ralph ./cmd/ralph
main     6321b372a1b777a24292ac21efd40459637d46a3
feat/r   8ecb5ba2e10c84d31ffab369753e06e556eeb7ae (gated, pushed)
feat/w   bfbb6c5645ba759bab9399b8b83acd4d8d54263a (gated, pushed; fake PR #23 into release)
feat/m   da25a99f4cf94cbf450b5cd7b79918fc74d9a161 (gated, never pushed; fake PR #19 MERGED)
feat/c   659aad92f456d754828f27dd78a66076876bb232 (gated, pushed; fake PR #20 CLOSED)
feat/g   gated 313a0dc4944cabb83179f5450afa0d38516393a9, pushed after the gate: 8d152e9a26268895a96f458ec1dfdd9d468664f0
feat/f   f251a842d4db913b0bbfc8aa58829e8f6cd51c83 (gate_failed static_failed)
feat/x   gated 48971dad8ba6c28f3e67ef73da935f4a3bddcf7f by org-q (no binding for feat/x), pushed after the gate: 84e3c10fbcbc8c57ed5e23c2f0f84650e27f6f26
feat/n   bound, no gate, never pushed
feat/q   bound by org-q, no gate, never pushed
$ git -C <tmp>/work ls-remote --symref origin HEAD 'refs/heads/*'
ref: refs/heads/main	HEAD
6321b372a1b777a24292ac21efd40459637d46a3	HEAD
659aad92f456d754828f27dd78a66076876bb232	refs/heads/feat/c
f251a842d4db913b0bbfc8aa58829e8f6cd51c83	refs/heads/feat/f
8d152e9a26268895a96f458ec1dfdd9d468664f0	refs/heads/feat/g
8ecb5ba2e10c84d31ffab369753e06e556eeb7ae	refs/heads/feat/r
bfbb6c5645ba759bab9399b8b83acd4d8d54263a	refs/heads/feat/w
84e3c10fbcbc8c57ed5e23c2f0f84650e27f6f26	refs/heads/feat/x
6321b372a1b777a24292ac21efd40459637d46a3	refs/heads/main
Seeded ledger <tmp>/state/manifest.jsonl:
{"ts":"2026-10-14T01:00:01Z","org_id":"org-r","seat_id":"","event":"scope_reserved","worktree":"/wt/r","details":"paths=internal/org-r/ split=demo feature=r digest=0123456789ab branch=feat/r approval=per-plan"}
{"ts":"2026-10-14T01:00:02Z","org_id":"org-w","seat_id":"","event":"scope_reserved","worktree":"/wt/w","details":"paths=internal/org-w/ split=demo feature=w digest=0123456789ab branch=feat/w approval=per-plan"}
{"ts":"2026-10-14T01:00:03Z","org_id":"org-m","seat_id":"","event":"scope_reserved","worktree":"/wt/m","details":"paths=internal/org-m/ split=demo feature=m digest=0123456789ab branch=feat/m approval=per-plan"}
{"ts":"2026-10-14T01:00:04Z","org_id":"org-c","seat_id":"","event":"scope_reserved","worktree":"/wt/c","details":"paths=internal/org-c/ split=demo feature=c digest=0123456789ab branch=feat/c approval=per-plan"}
{"ts":"2026-10-14T01:00:05Z","org_id":"org-n","seat_id":"","event":"scope_reserved","worktree":"/wt/n","details":"paths=internal/org-n/ split=demo feature=n digest=0123456789ab branch=feat/n approval=per-plan"}
{"ts":"2026-10-14T01:00:06Z","org_id":"org-g","seat_id":"","event":"scope_reserved","worktree":"/wt/g","details":"paths=internal/org-g/ split=demo feature=g digest=0123456789ab branch=feat/g approval=per-plan"}
{"ts":"2026-10-14T01:00:07Z","org_id":"org-f","seat_id":"","event":"scope_reserved","worktree":"/wt/f","details":"paths=internal/org-f/ split=demo feature=f digest=0123456789ab branch=feat/f approval=per-plan"}
{"ts":"2026-10-14T01:00:08Z","org_id":"org-q","seat_id":"","event":"scope_reserved","worktree":"/wt/q","details":"paths=internal/org-q/ split=demo feature=q digest=0123456789ab branch=feat/q approval=per-plan"}
{"ts":"2026-10-14T01:00:11Z","org_id":"org-r","seat_id":"","event":"gate_passed","details":"head=8ecb5ba2e10c84d31ffab369753e06e556eeb7ae base=6321b372a1b777a24292ac21efd40459637d46a3 base_ref=refs/remotes/origin/main branch=feat/r by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:12Z","org_id":"org-w","seat_id":"","event":"gate_passed","details":"head=bfbb6c5645ba759bab9399b8b83acd4d8d54263a base=6321b372a1b777a24292ac21efd40459637d46a3 base_ref=refs/remotes/origin/main branch=feat/w by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:13Z","org_id":"org-m","seat_id":"","event":"gate_passed","details":"head=da25a99f4cf94cbf450b5cd7b79918fc74d9a161 base=6321b372a1b777a24292ac21efd40459637d46a3 base_ref=refs/remotes/origin/main branch=feat/m by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:14Z","org_id":"org-c","seat_id":"","event":"gate_passed","details":"head=659aad92f456d754828f27dd78a66076876bb232 base=6321b372a1b777a24292ac21efd40459637d46a3 base_ref=refs/remotes/origin/main branch=feat/c by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:15Z","org_id":"org-g","seat_id":"","event":"gate_passed","details":"head=313a0dc4944cabb83179f5450afa0d38516393a9 base=6321b372a1b777a24292ac21efd40459637d46a3 base_ref=refs/remotes/origin/main branch=feat/g by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:16Z","org_id":"org-f","seat_id":"","event":"gate_failed","details":"reason=static_failed head=f251a842d4db913b0bbfc8aa58829e8f6cd51c83 base=6321b372a1b777a24292ac21efd40459637d46a3 base_ref=refs/remotes/origin/main branch=feat/f by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
{"ts":"2026-10-14T01:00:17Z","org_id":"org-q","seat_id":"","event":"gate_passed","details":"head=48971dad8ba6c28f3e67ef73da935f4a3bddcf7f base=6321b372a1b777a24292ac21efd40459637d46a3 base_ref=refs/remotes/origin/main branch=feat/x by=probe herdr_pane=unset log=\"/state/gates/x.log\""}
$ command cat <tmp>/git-wrapper
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/git.log"
if [ $# -eq 3 ] && [ "$1 $2 $3" = "remote get-url origin" ]; then
  case "${PROBE_ORIGIN_URL-}" in
    "") ;;
    FAIL) echo "fatal: probe: cannot read origin" >&2; exit 2 ;;
    *) printf '%s\n' "$PROBE_ORIGIN_URL"; exit 0 ;;
  esac
fi
exec /opt/homebrew/bin/git "$@"
$ command cat <tmp>/gh-fake
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/gh.log"
head=""
while [ $# -gt 0 ]; do
  case "$1" in --head) head="$2"; shift 2 ;; *) shift ;; esac
done
U="$PROBE_PR_BASE"
pr() { printf '[{"number":%s,"state":"%s","url":"%s/%s","headRefOid":"0000000000000000000000000000000000000000","baseRefName":"%s","isCrossRepository":false}]\n' "$1" "$2" "${3:-$U}" "$1" "$4"; }
case "$head" in
  feat/r) pr 21 OPEN "" main ;;
  feat/w) pr 23 OPEN "" release ;;
  feat/m) pr 19 MERGED "" main ;;
  feat/c) pr 20 CLOSED "${PROBE_PR_BASE_C:-$U}" main ;;
  *) printf '[]\n' ;;
esac
$ command cat <tmp>/gh-fail
#!/bin/sh
printf '%s\n' "$*" >> "$PROBE_LOG_DIR/gh.log"
echo "To get started with GitHub CLI, please run:  gh auth login" >&2
exit 4
Ledger and refs before the runs (AC9):
2f9a699addb3ed6566b926ba023eb2f3e49c657b870b2acb0c5f6d4d8ec71fcb  ./manifest.jsonl
      15    3718 manifest.jsonl
manifest.jsonl
55e343c56d236e3d200df07f3e961cfc7fdeb170ede0c52260e7e0fcbae4f3f2  -
402370edaaa6c73d28bc5298aaa3a9b7b03fb88bf05e5c8c6068fbaf63151604  -
```

### K1 origin acme/app (scp-like), gh PRs from acme/app: PRs used

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/acme/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:41Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	#21 https://github.com/acme/app/pull/21	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	bfbb6c5645ba	6321b372a1b7	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
3	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
4	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
6	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
7	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: 2 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: the order is as of the ls-remote at 2026-10-13T22:41:41Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K2 same, --all

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/acme/app/pull ralph org merge-order --all --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:42Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	#21 https://github.com/acme/app/pull/21	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	bfbb6c5645ba	6321b372a1b7	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
3	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
4	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
6	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
7	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
8	merged	feat/m	#19 https://github.com/acme/app/pull/19	da25a99f4cf9	6321b372a1b7	org-m/m	PR #19 is merged
9	closed	feat/c	#20 https://github.com/acme/app/pull/20	659aad92f456	6321b372a1b7	org-c/c	PR #20 was closed without a merge
note: the order is as of the ls-remote at 2026-10-13T22:41:42Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K3 origin with other case and https: PRs used (owner/repo compared without case)

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=https://GitHub.com/ACME/App.git PROBE_PR_BASE=https://github.com/acme/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:43Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	#21 https://github.com/acme/app/pull/21	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	bfbb6c5645ba	6321b372a1b7	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
3	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
4	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
6	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
7	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: 2 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: the order is as of the ls-remote at 2026-10-13T22:41:43Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K4 gh reads upstream-org/app: no PR used, one note, gh not run again

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/upstream-org/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:44Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	ready	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	the gate passed on bfbb6c5645ba, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
3	ready	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	the gate passed on 659aad92f456, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh reads upstream-org/app, not origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked; run gh repo set-default acme/app in this repository
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T22:41:44Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 1
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K5 same, --json

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/upstream-org/app/pull ralph org merge-order --json --state-dir <tmp>/state
--- stdout
{
  "default_branch": "main",
  "default_head": "6321b372a1b777a24292ac21efd40459637d46a3",
  "listed_at": "2026-10-13T22:41:44Z",
  "gh_available": true,
  "entries": [
    {
      "rank": 1,
      "class": "ready",
      "branch": "feat/r",
      "remote_head": "8ecb5ba2e10c84d31ffab369753e06e556eeb7ae",
      "detail": "the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "8ecb5ba2e10c84d31ffab369753e06e556eeb7ae",
        "base": "6321b372a1b777a24292ac21efd40459637d46a3",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:11Z"
      },
      "binding": {
        "org_id": "org-r",
        "feature": "r",
        "split": "demo",
        "ts": "2026-10-14T01:00:01Z"
      }
    },
    {
      "rank": 2,
      "class": "ready",
      "branch": "feat/w",
      "remote_head": "bfbb6c5645ba759bab9399b8b83acd4d8d54263a",
      "detail": "the gate passed on bfbb6c5645ba, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "bfbb6c5645ba759bab9399b8b83acd4d8d54263a",
        "base": "6321b372a1b777a24292ac21efd40459637d46a3",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:12Z"
      },
      "binding": {
        "org_id": "org-w",
        "feature": "w",
        "split": "demo",
        "ts": "2026-10-14T01:00:02Z"
      }
    },
    {
      "rank": 3,
      "class": "ready",
      "branch": "feat/c",
      "remote_head": "659aad92f456d754828f27dd78a66076876bb232",
      "detail": "the gate passed on 659aad92f456, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "659aad92f456d754828f27dd78a66076876bb232",
        "base": "6321b372a1b777a24292ac21efd40459637d46a3",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:14Z"
      },
      "binding": {
        "org_id": "org-c",
        "feature": "c",
        "split": "demo",
        "ts": "2026-10-14T01:00:04Z"
      }
    },
    {
      "rank": 4,
      "class": "regate",
      "branch": "feat/g",
      "remote_head": "8d152e9a26268895a96f458ec1dfdd9d468664f0",
      "detail": "pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "313a0dc4944cabb83179f5450afa0d38516393a9",
        "base": "6321b372a1b777a24292ac21efd40459637d46a3",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:15Z"
      },
      "binding": {
        "org_id": "org-g",
        "feature": "g",
        "split": "demo",
        "ts": "2026-10-14T01:00:06Z"
      }
    },
    {
      "rank": 5,
      "class": "regate",
      "branch": "feat/x",
      "remote_head": "84e3c10fbcbc8c57ed5e23c2f0f84650e27f6f26",
      "detail": "pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "48971dad8ba6c28f3e67ef73da935f4a3bddcf7f",
        "base": "6321b372a1b777a24292ac21efd40459637d46a3",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:17Z"
      }
    },
    {
      "rank": 6,
      "class": "failed",
      "branch": "feat/f",
      "remote_head": "f251a842d4db913b0bbfc8aa58829e8f6cd51c83",
      "detail": "the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)",
      "gate": {
        "event": "gate_failed",
        "readable": true,
        "head": "f251a842d4db913b0bbfc8aa58829e8f6cd51c83",
        "base": "6321b372a1b777a24292ac21efd40459637d46a3",
        "base_ref": "refs/remotes/origin/main",
        "reason": "static_failed",
        "ts": "2026-10-14T01:00:16Z"
      },
      "binding": {
        "org_id": "org-f",
        "feature": "f",
        "split": "demo",
        "ts": "2026-10-14T01:00:07Z"
      }
    },
    {
      "rank": 7,
      "class": "no_remote",
      "branch": "feat/m",
      "remote_head": "",
      "detail": "origin has no branch feat/m: it was not pushed, or was deleted after a merge",
      "gate": {
        "event": "gate_passed",
        "readable": true,
        "head": "da25a99f4cf94cbf450b5cd7b79918fc74d9a161",
        "base": "6321b372a1b777a24292ac21efd40459637d46a3",
        "base_ref": "refs/remotes/origin/main",
        "reason": "",
        "ts": "2026-10-14T01:00:13Z"
      },
      "binding": {
        "org_id": "org-m",
        "feature": "m",
        "split": "demo",
        "ts": "2026-10-14T01:00:03Z"
      }
    },
    {
      "rank": 8,
      "class": "no_gate",
      "branch": "feat/n",
      "remote_head": "",
      "detail": "no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)",
      "binding": {
        "org_id": "org-n",
        "feature": "n",
        "split": "demo",
        "ts": "2026-10-14T01:00:05Z"
      }
    },
    {
      "rank": 9,
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
    "gh reads upstream-org/app, not origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked; run gh repo set-default acme/app in this repository",
    "3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)",
    "the order is as of the ls-remote at 2026-10-13T22:41:44Z and is not a guarantee: origin/main can move before a merge"
  ]
}
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 1
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K6 first PRs from acme/app, feat/c's from upstream-org/app: every PR dropped

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/acme/app/pull PROBE_PR_BASE_C=https://github.com/upstream-org/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:45Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	ready	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	the gate passed on bfbb6c5645ba, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
3	ready	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	the gate passed on 659aad92f456, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh reads upstream-org/app, not origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked; run gh repo set-default acme/app in this repository
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T22:41:45Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 4
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K7 PR url with no owner/repo (cycle-1 fake shape): no PR used

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://example.invalid ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:45Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	ready	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	the gate passed on bfbb6c5645ba, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
3	ready	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	the gate passed on 659aad92f456, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh returned PR #21 for feat/r with the url "https://example.invalid/21", which names no repository to check against origin's acme/app: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T22:41:45Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 1
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K8 git remote get-url origin fails: no PR used

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=FAIL PROBE_PR_BASE=https://github.com/acme/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:45Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	ready	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	the gate passed on bfbb6c5645ba, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
3	ready	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	the gate passed on 659aad92f456, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: git remote get-url origin failed (exit status 2: fatal: probe: cannot read origin), so the repository gh read cannot be checked against origin's: no PR is used, so merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T22:41:45Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 1
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K9 ssh alias host (github.com.emu): hosts not compared, PRs used

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com.emu:acme/app.git PROBE_PR_BASE=https://github.com/acme/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:46Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	#21 https://github.com/acme/app/pull/21	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	bfbb6c5645ba	6321b372a1b7	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
3	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
4	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
6	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
7	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: 2 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: the order is as of the ls-remote at 2026-10-13T22:41:46Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K10 another host, same owner/repo (tech-debt row): PRs used

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=https://ghe.example.invalid/acme/app.git PROBE_PR_BASE=https://github.com/acme/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:46Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	#21 https://github.com/acme/app/pull/21	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	wrong_base	feat/w	#23 https://github.com/acme/app/pull/23	bfbb6c5645ba	6321b372a1b7	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
3	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
4	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
6	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
7	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: 2 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: the order is as of the ls-remote at 2026-10-13T22:41:46Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K11 real origin URL is a local path (no fake): no check, upstream PRs used as is (C2-3)

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_PR_BASE=https://github.com/upstream-org/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:46Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	#21 https://github.com/upstream-org/app/pull/21	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	wrong_base	feat/w	#23 https://github.com/upstream-org/app/pull/23	bfbb6c5645ba	6321b372a1b7	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
3	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
4	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
6	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
7	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: 2 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: the order is as of the ls-remote at 2026-10-13T22:41:46Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K12 origin git+ssh:// (scheme not read): no check, upstream PRs used as is (C2-3)

```
$ cd <tmp>/work && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git+ssh://git@github.com/acme/app.git PROBE_PR_BASE=https://github.com/upstream-org/app/pull ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:47Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	#21 https://github.com/upstream-org/app/pull/21	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	wrong_base	feat/w	#23 https://github.com/upstream-org/app/pull/23	bfbb6c5645ba	6321b372a1b7	org-w/w	PR #23 merges into release, not origin's default branch main: change the PR's base to main
3	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
4	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
5	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
6	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
7	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: 2 branch(es) whose PR is merged or closed are not shown (--all shows them)
note: the order is as of the ls-remote at 2026-10-13T22:41:47Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 2 (remote get-url origin: 1, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K13 gh fails for every branch with the same error: one grouped note (L2)

```
$ cd <tmp>/work && PATH=<tmp>/bin-ghfail PROBE_ORIGIN_URL=git@github.com:acme/app.git ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:47Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	ready	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	the gate passed on bfbb6c5645ba, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
3	ready	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	the gate passed on 659aad92f456, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh pr list failed for 9 branches (feat/r, feat/w, feat/m, feat/c, feat/n and 4 more) with the same error (exit status 4: To get started with GitHub CLI, please run:  gh auth login): their PRs are unknown, so merged, closed and the PR base are not told apart for them
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T22:41:47Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 1 (remote get-url origin: 0, ls-remote: 1)
--- gh calls: 9
    gh pr list --head feat/r --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/w --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/m --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/c --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/n --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/g --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/f --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/q --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
    gh pr list --head feat/x --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository
```

### K14 no gh on PATH: get-url not run

```
$ cd <tmp>/work && PATH=<tmp>/bin PROBE_ORIGIN_URL=git@github.com:acme/app.git ralph org merge-order  --state-dir <tmp>/state
--- stdout
origin/main 6321b372a1b7 (listed 2026-10-13T22:41:47Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	ready	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate passed on 8ecb5ba2e10c, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
2	ready	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	the gate passed on bfbb6c5645ba, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
3	ready	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	the gate passed on 659aad92f456, origin's tip of the branch, with base 6321b372a1b7, origin/main's tip
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: 3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)
note: the order is as of the ls-remote at 2026-10-13T22:41:47Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
--- git calls: 1 (remote get-url origin: 0, ls-remote: 1)
--- gh calls: 0
```

### Ledger, refs and files after K1-K14 (AC9)

```
2f9a699addb3ed6566b926ba023eb2f3e49c657b870b2acb0c5f6d4d8ec71fcb  ./manifest.jsonl
      15    3718 manifest.jsonl
manifest.jsonl
55e343c56d236e3d200df07f3e961cfc7fdeb170ede0c52260e7e0fcbae4f3f2  -
402370edaaa6c73d28bc5298aaa3a9b7b03fb88bf05e5c8c6068fbaf63151604  -
find <tmp>/state <tmp>/work <tmp>/origin.git <tmp>/home -newer <tmp>/marker | wc -l
       0
```

### real git: remote get-url expands insteadOf and reads no network

```
$ git -C <tmp>/alias config --get-regexp "^(remote\.origin\.url|url\.)"
remote.origin.url https://example.invalid/acme/app.git
url.git@alias.invalid:.insteadof https://example.invalid/
$ time git -C <tmp>/alias remote get-url origin
git@alias.invalid:acme/app.git
rc=0
elapsed 0.043s
$ git -C <tmp>/work remote get-url origin   (K11 origin, real)
<tmp>/origin.git
```
PROBE_DONE T=<tmp>

### K15 ls-remote fails (repository with no origin) with gh on PATH: exit 1, gh and get-url not run (AC7)

```
$ cd <tmp>/noorigin && PATH=<tmp>/bin-gh PROBE_ORIGIN_URL=git@github.com:acme/app.git PROBE_PR_BASE=https://github.com/acme/app/pull ralph org merge-order --state-dir <tmp>/state
--- stdout
--- stderr
org: merge-order: git ls-remote --symref origin HEAD 'refs/heads/*' in . failed (exit status 128: fatal: 'origin' does not appear to be a git repository
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.): origin's state is unknown, so nothing was classified; merge-order reads origin, so run it in a repository whose origin answers
--- exit=1
--- git calls: 1 (remote get-url origin: 0)
--- gh calls: 0
```

### K16 the K1 Detail for feat/x says `ralph org gate` without --org-id: the CLI refuses that form (self-review C2-1)

```
$ cd <tmp>/work && PATH=<tmp>/bin ralph org gate --state-dir <tmp>/state
--- stdout
--- stderr
org: --org-id is required
--- exit=1
```

```
$ cmp <tmp>/manifest.before <tmp>/state/manifest.jsonl && /bin/ls -A <tmp>/state
cmp: same
manifest.jsonl
```

### K17 setup: <tmp>/origin2.git is a bare clone of <tmp>/origin.git; its main moved from 6321b372a1b777a24292ac21efd40459637d46a3 to f889f74955607e57af628dc7d257b1cf92f761c6

### K17 origin's main moved after the gates (AC1 rebase; Detail names --org-id)

```
$ cd <tmp>/work2 && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
origin/main f889f7495560 (listed 2026-10-13T22:47:42Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	rebase	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	origin/main moved: gate base 6321b372a1b7, now f889f7495560: rebase on origin/main and run ralph org gate --org-id org-r again (the gate runs only while the feature org runs)
2	rebase	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	origin/main moved: gate base 6321b372a1b7, now f889f7495560: rebase on origin/main and run ralph org gate --org-id org-w again (the gate runs only while the feature org runs)
3	rebase	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	origin/main moved: gate base 6321b372a1b7, now f889f7495560: rebase on origin/main and run ralph org gate --org-id org-c again (the gate runs only while the feature org runs)
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: the order is as of the ls-remote at 2026-10-13T22:47:42Z and is not a guarantee: origin/main can move before a merge
--- stderr
--- exit=0
```

### K18 setup: origin2's HEAD -> trunk (f889f74955607e57af628dc7d257b1cf92f761c6), main back at 6321b372a1b777a24292ac21efd40459637d46a3

### K18 origin's default branch is now trunk, old main kept (AC11 rebase)

```
$ cd <tmp>/work2 && PATH=<tmp>/bin ralph org merge-order --state-dir <tmp>/state
--- stdout
origin/trunk f889f7495560 (listed 2026-10-13T22:47:42Z)
RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
1	rebase	feat/r	-	8ecb5ba2e10c	6321b372a1b7	org-r/r	the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk: rebase on origin/trunk and run ralph org gate --org-id org-r again (the gate runs only while the feature org runs)
2	rebase	feat/w	-	bfbb6c5645ba	6321b372a1b7	org-w/w	the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk: rebase on origin/trunk and run ralph org gate --org-id org-w again (the gate runs only while the feature org runs)
3	rebase	feat/c	-	659aad92f456	6321b372a1b7	org-c/c	the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk: rebase on origin/trunk and run ralph org gate --org-id org-c again (the gate runs only while the feature org runs)
4	regate	feat/g	-	313a0dc4944c	6321b372a1b7	org-g/g	pushed after the gate: origin has 8d152e9a2626, the gate checked 313a0dc4944c: run ralph org gate --org-id org-g again (the gate runs only while the feature org runs)
5	regate	feat/x	-	48971dad8ba6	6321b372a1b7	-	pushed after the gate: origin has 84e3c10fbcbc, the gate checked 48971dad8ba6: run ralph org gate again (the gate runs only while the feature org runs)
6	failed	feat/f	-	f251a842d4db	6321b372a1b7	org-f/f	the gate failed (static_failed): fix it and run ralph org gate --org-id org-f again (the gate runs only while the feature org runs)
7	no_remote	feat/m	-	da25a99f4cf9	6321b372a1b7	org-m/m	origin has no branch feat/m: it was not pushed, or was deleted after a merge
8	no_gate	feat/n	-	-	-	org-n/n	no gate record: run ralph org gate --org-id org-n in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
9	no_gate	feat/q	-	-	-	org-q/q	no gate record: run ralph org gate --org-id org-q in the feature worktree before the PR is merged (the gate runs only while the feature org runs)
note: gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR bases are not checked
note: the order is as of the ls-remote at 2026-10-13T22:47:42Z and is not a guarantee: origin/trunk can move before a merge
--- stderr
--- exit=0
```

```
$ cmp <tmp>/manifest.before <tmp>/state/manifest.jsonl && /bin/ls -A <tmp>/state
cmp: same
manifest.jsonl
```
