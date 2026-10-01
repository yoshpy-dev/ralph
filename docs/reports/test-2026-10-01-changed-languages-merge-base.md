# Test report: changed-languages-merge-base

- Date: 2026-10-01
- Plan: docs/plans/active/2026-10-01-changed-languages-merge-base.md (issue #190)
- Tester: tester subagent (Claude Code), cycle 1
- Scope: behavioral tests only (`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`, `tests/test-detect-changed-languages.sh`, `tests/test-run-verify-scope.sh`), plus sh/dash repeat runs, mutation re-testing in a scratch mirror, an end-to-end run of the real wrappers against a pushed-branch fixture, and edge-case probes. No static analysis (that is `/verify`'s job; see `docs/reports/verify-2026-10-01-changed-languages-merge-base.md`, PASS).
- Evidence: this report's own command output (inline below). The full-scope run wrote `docs/evidence/verify-2026-10-01-123444.log` and the default-scope run wrote `docs/evidence/verify-2026-10-01-123912.log` (both gitignored, inside the worktree). Scratch harnesses (mutation runner, fixture probes, e2e) live in the session scratchpad and are not committed.

Worktree: `.claude/worktrees/changed-languages-merge-base` (branch `fix/changed-languages-merge-base`), HEAD `725e7a9b` confirmed at start and end of this cycle, `git status --porcelain` empty throughout. Every scratch probe ran with `HOME=<scratch>`, `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`.

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` (all shell suites via `verify.local.sh` + Go packages) | full repo test scope | all green, exit 0 | 0 | 0 (one in-suite SKIP: real git < 2.41 case, stubbed variant covers it) | 3:50 |
| `./scripts/run-test.sh` (default scope, no env) | same scope, selected by the new detector | all green, exit 0 | 0 | 0 | 3:35 |
| `tests/test-detect-changed-languages.sh` (inside the full run) | 60 | 60 | 0 | 0 | ~8 s |
| `tests/test-run-verify-scope.sh` (inside the full run) | 19 | 19 | 0 | 0 | ~8 s |
| `sh` x3 and `dash` x3 of `test-detect-changed-languages.sh` | 60 each | 60 each (6 runs) | 0 | 0 | 8.3 s (dash, timed once) |
| `sh` x3 and `dash` x3 of `test-run-verify-scope.sh` | 19 each | 19 each (6 runs) | 0 | 0 | 7.6 s (dash, timed once) |
| `go test ./...` (inside the full run) | 8 test-bearing packages | 8 `ok` (cli, config, insights, org, org/driver, org/protocol, scaffold, upgrade) | 0 | 0 | org 10.8 s, rest cached |

The 22 `FAIL` summary lines in the full-scope log are all `FAIL: 0`; the other `FAIL` substrings are PASS case descriptions (for example "does not cause a FAIL"). No non-zero failure count anywhere.

`sh` on this machine is bash 3.2.57 in POSIX mode, `dash` is `/bin/dash`; both gave identical output on all 12 runs (no flakiness, no ordering or timing sensitivity). The suites use only local bare remotes and private temp dirs, so there is no network or clock dependence.

### What the default `./scripts/run-test.sh` selects on this branch

```
==> Language scope: full fallback (shared:scripts/detect-changed-languages.sh)
==> Language packs selected: golang
```

This is the expected result: the branch changes the detector itself, which `is_shared_full_file` classifies as shared. The upstream of the branch equals HEAD (`725e7a9b`) and `origin/HEAD` points to `origin/main`. The old detector (`git show main:scripts/detect-changed-languages.sh`) run in the same checkout returns `scope=changed reason=no_changes docs_only=true languages=` - the exact #190 symptom, which this branch no longer produces.

## Coverage

- Statement: not measured (POSIX shell, no instrumentation tool in this repo).
- Branch: not measured; assessed by case reach instead (below).
- Function: not measured.
- Notes: the new base-selection code is `remote_default_ref` plus the cascade after it. Case reach against the baseline detector:
  - `remote_default_ref`: HEAD target valid (13), HEAD target dangling (14), no HEAD, falls to `main` (10, 16, 17), `master` fallback (NOT reached by any case, see gaps), HEAD target outside `refs/remotes/<remote>/` (NOT reached), none found (23-25).
  - Base cascade: origin step (10, 13, 14, 16), tracked-remote step (17; 25 reaches it and finds nothing), local `main` (7, 18, 19, 22, 25), local `master` (NOT reached), `no_remote_default` guard (23, 24; not-fired side 25), `no_diff_base` (20), `no_merge_base` (12), detached HEAD (18), remote `.` (19, but see gap G-4).
  - Wrapper level (`run-static-verify.sh`): pushed branch, `central`-only remote. `run-test.sh` wrappers are covered with the local-main fixture and, in this cycle, by the manual e2e below.

## Mutation table

All mutations were applied only to the root `scripts/detect-changed-languages.sh` inside a `git archive HEAD` mirror (`<scratch>/mut`). Both suites were run (under `sh`) after each one. The template copy in the mirror was left untouched and `cmp`-checked equal to the baseline after every restore; neither suite reads the template. Baseline on the mirror: detector suite 60/60, scope suite 19/19. After each mutation the root script was restored from a baseline copy and `cmp`-verified.

Required by the lead (a-f), all caught by the predicted assertion:

| # | Mutation | Detector suite | Scope suite |
| --- | --- | --- | --- |
| a | `@{upstream}` consulted first (old behaviour) | 4 FAIL: "pushed branch is not docs-only", "pushed branch selects golang" (AC-1), "trunk default branch selects golang", "dangling origin/HEAD falls through to origin/main" (the last two fail only because their fixtures are pushed branches too) | 1 FAIL: "pushed branch static wrapper runs golang pack for committed change" (AC-5) |
| b | drop the `refs/remotes/<r>/HEAD` step | 2 FAIL: "trunk default branch uses changed scope" / "selects golang" (AC-3) | survives (no trunk fixture at wrapper level; not expected to catch) |
| c | drop the HEAD-target existence check | 2 FAIL: "dangling origin/HEAD uses changed scope" / "falls through to origin/main" (AC-3) | survives (not expected to catch) |
| d | drop the tracked-remote step | 2 FAIL: "central-only remote uses changed scope" / "selects golang for unpushed commit" (AC-8) | 2 FAIL: "central-only remote static wrapper runs golang pack for unpushed commit", "... skips unrelated python pack" (AC-8) |
| e | short names to `merge-base` (strip `refs/remotes/`, `refs/heads/`) | 2 FAIL: "branch named origin/main does not hide golang" (case 21), "tag named main does not hide golang" (case 22) | survives (not expected to catch) |
| f | drop the `no_remote_default` guard | 4 FAIL: cases 23 and 24, scope and reason assertions each | survives (not expected to catch) |

Additional mutations chosen by the tester:

| # | Mutation | Result | Catcher |
| --- | --- | --- | --- |
| k | drop the whole origin step | caught (detector 5 FAIL) | "trunk default branch ...", "clean default branch reports no_changes" (got `no_remote_default:origin`), "unpushed commit on default branch selects golang" |
| n | L-1 guard fires whenever a remote is tracked (drop the `local_default = current_ref` condition) | caught (detector 2 FAIL) | case 25, "unfetched remote on a feature branch uses changed scope" / "diffs against local main" |
| u | L-1 guard compares with literal `refs/heads/main` instead of the current branch | caught (detector 2 FAIL) | same two case-25 assertions |
| r | drop `\|\| true` on the `branch.<b>.remote` config read | caught: detector suite aborts on its first case (set -e), scope suite 4 FAIL | whole suite |
| s | drop `\|\| true` on the current-branch `symbolic-ref` (detached HEAD) | caught: detector suite aborts at case 18 after "central-only remote selects golang..." | case 18 (detached HEAD) |
| g | swap order: tracked remote before origin | SURVIVED both suites | see G-1 |
| h | drop the `refs/remotes/<remote>/` prefix check on the HEAD target | SURVIVED both suites | see G-3 |
| i | treat remote `.` as a real remote (drop the `.` -> empty case) | SURVIVED both suites | see G-4 |
| j1 | drop the `master` fallback on remotes | SURVIVED both suites | see G-2 |
| j2 | drop the local `master` fallback | SURVIVED both suites | see G-2 |
| p | swap remote `main`/`master` order | SURVIVED both suites | see G-2 |
| q | swap local `main`/`master` order | SURVIVED both suites | see G-2 |
| t | tracked-remote step no longer skips `origin` | SURVIVED, equivalent mutant | `remote_default_ref origin` just runs twice with an identical empty result; no observable difference possible |

Tally: 19 mutations, 11 caught, 8 survived (7 closable test gaps, 1 equivalent). All six AC-6 mutations are caught by exactly the assertions the plan names.

### Survivors are closable, not equivalent

Each survivor except `t` was re-probed with a purpose-built fixture, run against the baseline detector and the mutant (outputs: `scope reason languages`):

| Probe | Fixture | Baseline | Mutant | Old detector (main) |
| --- | --- | --- | --- | --- |
| P-g | `origin` has a python commit on main; stale `fork` remote (main = init commit); feature branch carrying a Go commit tracks `fork/feature` | `changed changed_languages golang` | g: `golang python` | `changed no_changes` |
| P-h | `origin/HEAD` -> `refs/heads/main` (outside the remote); local main ahead of `origin/main` by a python commit; feature with a Go commit | `golang python` | h: `golang` | `golang python` |
| P-i | `main` tracks local `develop` (remote `.`), no remotes, Go commit on main | `changed no_changes` | i: `full no_remote_default:.` | `changed golang` (old base was the `develop` upstream) |
| P-j1 | remote default is `master` only (local `master` deleted), pushed feature with a Go commit | `golang` | j1: `full no_diff_base` | `no_changes` |
| P-j2 | only local `master`, no remote | `golang` | j2: `full no_diff_base` | `golang` |
| P-p | remote has stale `master` and current `main`, no `origin/HEAD` | `golang` | p: `golang python` | `no_changes` |
| P-q | local stale `master` and current `main`, no remote | `golang` | q: `golang python` | `golang` |

Each baseline/mutant pair differs, so a case with the probe's fixture and an exact `languages` or `scope` assertion would kill the mutant.

## End-to-end (real wrappers, pushed feature branch)

A scratch fixture was built with the real `run-verify.sh`, `run-static-verify.sh`, `run-test.sh` (wrappers are byte-identical between main and this branch) plus the detector under test, stub golang and python packs, a stub `detect-languages.sh` (prints both) and a stub `verify.local.sh`, the same pattern as `tests/test-run-verify-scope.sh`. `main` was pushed to a bare `origin`, then a `feature` branch with a Go commit under `service/` was pushed with `-u` (HEAD == `@{upstream}`, clean tree). The wrappers were then run with no `RALPH_VERIFY_SCOPE` set.

| Detector | `./scripts/run-test.sh` | `./scripts/run-static-verify.sh` |
| --- | --- | --- |
| new (this branch) | `Language scope: changed (changed_languages)`, `packs selected: golang`, calls `local:test:changed`, `golang:test:changed:service`; python not called | `Language scope: changed (changed_languages)`, `packs selected: golang`, calls `local:static:changed`, `golang:static:changed:service` |
| old (`git show main:scripts/detect-changed-languages.sh`) | `Language scope: changed (no_changes)`, `packs selected: none`, only `local:test:changed` | `Language scope: changed (docs_only)`, `packs selected: none`, only `local:static:changed` |

Result: pass. The new detector makes the default `/test` and `/verify` wrappers run the golang pack for a pushed branch with a Go change (narrowed to root `service`); the old detector reproduces the #190 symptom (no pack runs, wrappers still exit 0). The `run-test.sh` side is not covered by `tests/test-run-verify-scope.sh` for the pushed-branch case (that suite uses `run-static-verify.sh` there); it works, as shown above.

## Edge cases beyond the suites

Probed with hermetic fixtures against the new detector (old detector in the last column for reference):

| Edge | New | Old (main) |
| --- | --- | --- |
| E1 shallow clone (`--depth 1 --no-single-branch`) of a pushed feature: merge-base cannot be computed | `full no_merge_base:refs/remotes/origin/main` (conservative direction) | `no_changes` |
| E2 remote named with a slash (`team/central`), unpushed Go commit on main | `golang` (`branch.main.remote` = `team/central`) | `golang` |
| E3 branch named `feature/x.1` pushed to `central` only | `golang` | `no_changes` |
| E4 origin default branch `release/1.0` (slash) via `origin/HEAD`, no main/master anywhere | `golang` | `no_changes` |
| E5 linked worktree of a pushed feature branch | `golang` | `no_changes` |
| E6 `origin/HEAD` is a direct, non-symbolic ref | `golang` (falls through to `origin/main`) | `no_changes` |
| E7 pushed docs-only branch | `changed docs_only` | `no_changes` |
| E8 pushed branch touching `scripts/run-test.sh` | `full shared:scripts/run-test.sh` | `no_changes` |
| E9 pushed feature with no local `main` (CI-style) | `golang` (via `origin/main`) | `no_changes` |

All new-detector results match the plan's intent. The plan's own edge list (no upstream, detached HEAD, remote `.`, no origin, no main/master, nonexistent explicit base) is covered by cases 7, 18, 19, 20 and 12; only the shallow-clone and slash-in-remote-name edges, listed above, were not covered by the suites.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| none | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #190: pushed branch (upstream == HEAD) with a Go commit reports `no_changes` and the default wrappers run no language pack | fixed | cases 10 (detector) and the pushed-branch case (scope suite), e2e table above, mutation (a) turns them red again |
| Explicit `RALPH_VERIFY_BASE` keeps precedence | unchanged | cases 11 and 12 (HEAD-equal explicit base -> `no_changes`; nonexistent -> `no_merge_base:nope`) |
| Original 9 cases (uncommitted change, multi-language, docs-only, shared, unclassified, no git, branch vs main, JVM, nested roots) | green | cases 1-9 pass under sh and dash x3 |
| Default branch, clean, nothing unpushed | unchanged (`no_changes`) | case 15 |
| Tag or branch sharing a short name cannot become the base | held | cases 21, 22; mutation (e) red |

Behavioral difference vs the old detector, not a failure (observation O-1): a `main` that tracks a local branch with no remotes (probe P-i) used to report `golang` because `@{upstream}` was that local branch; it now reports `no_changes`, the same V-1 limitation the verify report already notes for remote-less repos (a commit on the local default branch cannot be seen). The scenario is rare and the plan accepts the V-1 limitation; no test failure.

## Test gaps

None blocks the verdict; all are closable with the fixtures listed above. Counted as LOW findings.

- G-1 (LOW): the documented design decision "origin is consulted before the tracked remote" has no case. Mutation g survives both suites. A fixture with `origin` plus a stale tracked `fork` remote and an exact `languages=golang` assertion (probe P-g) kills it.
- G-2 (LOW): no fixture uses `master` as the default branch. The remote and local `master` fallbacks (j1, j2) and both `main`-before-`master` orderings (p, q) are untested; probes P-j1, P-j2, P-p, P-q each distinguish baseline from mutant.
- G-3 (LOW): a `refs/remotes/<remote>/HEAD` that points outside `refs/remotes/<remote>/` is never exercised, so the prefix check (h) is untested; probe P-h kills it. Git itself only creates the symbolic ref inside the remote's namespace, so this is defensive code.
- G-4 (LOW): case 19 (feature tracking local `main`, remote `.`) gives the same result whether or not `.` is normalised to "no remote", so mutation i survives. Probe P-i (default branch tracking a local branch) is the discriminating shape; if added, assert only that `reason` is not `no_remote_default:.`, so the V-1 `no_changes` limitation is not pinned as desired behavior.
- Not tested and not applicable here: busybox sh (not installed on this machine), Windows/Git-Bash, and the real `ralph init` scaffold path for the detector (the root and template detector copies are byte-identical per the verify report, and `go test ./internal/scaffold/...` passed inside the full run).
- Flakiness: none observed across 12 suite runs under two shells plus two full-scope runs.

## Verdict

- Pass: yes. Full-scope and default-scope `run-test.sh` exit 0; detector suite 60/60 and scope suite 19/19, identical under sh and dash on 3 repeat runs each; all six AC-6 mutations caught by the assertions the plan names; end-to-end default wrappers run the golang pack on a pushed branch where the old detector ran nothing; 9 extra edge cases behave as intended.
- Fail: no.
- Blocked: no.

Proceeding to `/sync-docs` is appropriate. Four LOW test gaps (G-1 to G-4) are recorded for the operator to decide on; none is a defect in the shipped detector.
