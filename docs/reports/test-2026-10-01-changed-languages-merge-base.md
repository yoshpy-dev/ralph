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

---

## Cycle 1 addendum (Slices C and D)

- Date: 2026-10-01
- Scope: delta test of `git diff 725e7a9b..HEAD -- scripts/ tests/ templates/` at HEAD `02f7c78e` (Slice C `7f5c6235`: closes G-1 to G-4 and O-1; Slice D `54ab517c`: the reason shows only `remote_label`, never the raw `branch.<b>.remote`). Three files changed (+202 / -23): the detector, its template copy (`cmp`-identical to the root script), and `tests/test-detect-changed-languages.sh` (cases 26 to 33). Behavioral tests only; same isolation as above (`HOME=<scratch>`, `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`). No credential-shaped literal was written anywhere; where a probe needed a URL carrying a userinfo component, it was assembled at runtime from a generated token and is described in words here.
- Pipeline cycle: still 1 (this is a delta of the same cycle, not a re-run of the pipeline).

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` (foreground) | full repo test scope | all green, exit 0 | 0 | 0 | 3:51 |
| `tests/test-detect-changed-languages.sh` (inside the full run) | 76 (was 60) | 76 | 0 | 0 | 12.1 s (dash, timed once) |
| `tests/test-run-verify-scope.sh` (inside the full run) | 19 | 19 | 0 | 0 | ~8 s |
| `sh` x3 and `dash` x3 of the detector suite | 76 each | 76 each (6 runs) | 0 | 0 | - |
| `sh` x3 and `dash` x3 of the scope suite | 19 each | 19 each (6 runs) | 0 | 0 | - |
| `go test ./...` (inside the full run) | 8 test-bearing packages | 8 `ok` | 0 | 0 | org 9.1 s, rest cached |

The other shell suites in the full run report the same totals as in the first run (44, 8, 29, 64, 36, 11, 59 passed, each with `FAIL: 0`). All 12 repeat runs were identical (no flakiness). The 16 new assertions are cases 26 (2), 27 (2), 28 (2), 29 (1), 30 (1), 31 (2), 32 (2) and 33 (4: `scope`, `reason`, and two `assert_output_lacks` checks for the scheme and the path).

### Mutation table (HEAD `02f7c78e`)

Harness as in the first run (`git archive HEAD` mirror, edits applied to the mirror's root script only, both suites run under `sh`, restore plus `cmp` of root and template after each). Baseline on the mirror: detector 76/76, scope 19/19. Anchors were re-derived for the changed code (the guard now tests `configured_remote`; `i` is redefined as "drop the `.` branch of the label `case`", which makes `.` fall into the configured-remote branch and be labelled `url`).

28 mutations: 24 caught, 4 survived (1 equivalent, 1 cosmetic, 2 real closable gaps).

Previously surviving, now caught (the lead's list):

| # | Mutation | Caught by (detector suite; scope suite survives, as expected) |
| --- | --- | --- |
| g | tracked remote before origin | case 26 "origin default branch wins over the tracked fork (exact languages)" (got `golang python`) |
| h | drop the `refs/remotes/<remote>/` prefix check | case 31 "origin/HEAD outside the remote falls through to origin/main" |
| i | `.` no longer special (labelled `url`) | case 32 "main tracking a local branch records no_remote_default" (got `no_remote_default:url`) |
| j1 | drop the remote `master` fallback | case 27, both assertions |
| j2 | drop the local `master` fallback | case 28, both assertions |
| p | remote `master` before `main` | case 29 (got `golang python`) |
| q | local `master` before `main` | case 30 (got `golang python`) |
| o1 | O-1 revert (`.` treated as no remote for the guard) | case 32, both assertions (got `no_changes`) |
| A1 | reason prints the raw `branch.<b>.remote` | case 33: the `no_remote_default:url` reason assertion and both `assert_output_lacks` checks (scheme and path) |

Carried over, still caught with the same or a larger set of assertions: a (9 detector assertions plus the scope suite's pushed-branch assertion), b, c, d (both suites), e, f (8 assertions now, including cases 32 and 33), k, n, u, r, s.

New variants for Slice D:

| # | Mutation | Result | Catcher |
| --- | --- | --- | --- |
| A1b | non-remote value labelled with itself instead of `url` | caught | case 33, reason plus both lacks checks |
| A1c | label for a non-remote value empty | caught | case 33, reason (got `no_remote_default:`) |
| A2 | drop the configured-remote check (every non-`.` value is a remote name) | caught | case 33 (raw value reaches the reason) |
| A2c | configured-remote check inverted | caught | cases 17, 23, 24, 33 and the scope suite's `central` case |
| A2b | step 2 consults the raw value even when it is not a configured remote (label stays `url`) | SURVIVED both suites | G-5 |
| A2d | configured-remote check keyed on `remote.<name>.fetch` instead of `.url` | SURVIVED both suites | G-6 |
| A3 | configured-remote key built from the first dot or slash segment of the name | SURVIVED both suites | G-7 |
| t | tracked-remote step no longer skips `origin` | SURVIVED, equivalent | `remote_default_ref origin` just runs twice with an identical empty result |

All first-run required mutations (a to f) still go red on the assertions named in the first run's table.

### Probes (real wrappers, hermetic scratch fixtures)

The fixtures reuse the first run's pattern: the real `run-verify.sh`, `run-static-verify.sh`, `run-test.sh` plus the HEAD detector, stub golang and python packs, a stub `detect-languages.sh` (prints both) and a stub `verify.local.sh`, with no `RALPH_VERIFY_SCOPE` set.

A-1, URL-valued `branch.main.remote` (main has no remote, a Go commit sits on main, so the L-1 guard fires). Eight runtime-built value shapes: a `file://` URL to a scratch bare repo, an scp-style `host:path`, an `ssh://` URL, a relative path, `-x`, `r*`, `a b`, and an `ssh://` URL whose authority carried a userinfo component with a generated token.

| Observation | Result for all 8 shapes |
| --- | --- |
| Detector output | `scope=full reason=no_remote_default:url` |
| `./scripts/run-test.sh` and `./scripts/run-static-verify.sh` (default scope) | both exit 0 and print `==> Language scope: full fallback (no_remote_default:url)`; the golang and python stub packs both run with no project roots (full fallback) |
| `.harness/state/verify-scope` | `reason=no_remote_default:url` |
| Leak scan of the wrappers' stdout and stderr, the stub call logs, `.harness/state/verify-scope` and both `docs/evidence/verify-*.log` files (7 files per shape) for the scheme, host, path, the raw value and, for the userinfo shape, the user name and the token | 0 hits for every shape. The raw value exists only in the fixture's `.git/config`, where the user put it. The short values (`-x`, `r*`, `a b`) were checked against the `Language scope` and `reason=` lines only, since they match unrelated text elsewhere (the copied wrapper script itself contains `-x`); no hit |

The scan was sanity-checked: each scanned file does contain the `no_remote_default:url` label, and the extracted token was non-empty (12 characters).

Configured remote names containing a dot or a slash (`my.remote`, `team/central`, `a.b/c.d`), each added with `git remote add`, pushed with `-u`, then an unpushed Go commit on main:

| Name | Fetched remote, unpushed Go commit | Configured but never fetched |
| --- | --- | --- |
| `my.remote` | detector `golang`; both wrappers run `golang:*:changed:service` only | `full no_remote_default:my.remote` |
| `team/central` | same | `full no_remote_default:team/central` |
| `a.b/c.d` | same | `full no_remote_default:a.b/c.d` |

So step 2 consults such remotes (otherwise the local main would equal HEAD and the guard would have forced `full`), and the reason carries the name. The suite does not pin these (G-7).

O-1 under the real wrappers (main tracks a local branch with no remote, Go commit on main): detector `full no_remote_default:.`; both wrappers print `==> Language scope: full fallback (no_remote_default:.)` and run the golang pack (and the stub python pack) with no project roots. Before Slice C this shape reported `no_changes` and ran nothing; the old `main` detector reported `golang` through its `@{upstream}` base. The first run's probe set re-run on the HEAD detector is otherwise unchanged (P-g, P-h, P-j1, P-j2, P-p, P-q, E1 to E9 give the same results as before; only P-i changed, to the O-1 result above).

Survivor probes:

| Probe | Fixture | Baseline | Mutant |
| --- | --- | --- | --- |
| ghost | `branch.main.remote` names a remote whose `remote.<name>` section was removed while `refs/remotes/<name>/main` remains | `full no_remote_default:url` | A2b: `changed golang` |
| fetchless | `remote.<name>.url` set, no `fetch` refspec, main tracks it | `full no_remote_default:<name>` | A2d: `full no_remote_default:url` |
| dotted names | the three fetched-remote fixtures above | `changed golang` | A3: `full no_remote_default:url` |

### Remaining gaps

None blocks the verdict. Counted as LOW findings.

- G-5 (LOW): the Slice D rule "step 2 skips a value that is not a configured remote name" is not observable in case 33, because a URL cannot name a valid `refs/remotes/<x>/` namespace. Only a stale namespace (config section removed, refs kept: the ghost fixture) tells the two apart. Mutation A2b survives; the ghost fixture kills it. Defensive code for a rare shape (`git remote remove` deletes the refs too).
- G-6 (cosmetic, not counted): A2d changes only the reason label for a remote that has a `url` but no `fetch` refspec. Such a remote has no fetch refspec, so it normally holds no remote-tracking refs and step 2 finds nothing either way. Equivalent in effect except for the label text; a `git config remote.<name>.url` only fixture would pin it if wanted.
- G-7 (LOW): no case uses a remote name with a dot or slash, so a naive-split bug in the name handling (A3) would pass both suites. The probes show the current code is correct for `my.remote`, `team/central` and `a.b/c.d`; adding one fetched-remote case with such a name closes it.
- G-1 to G-4 from the first run are closed (mutations g, h, i, j1, j2, p, q caught).
- Unchanged from the first run: busybox sh not available on this machine.
- Observation O-1 from the first run is resolved by Slice C: a default branch tracking a local branch now falls back to full instead of `no_changes`. The remote-less V-1 limitation (no tracked remote at all) is unchanged.
- Flakiness: none observed across 12 repeat runs plus one full-scope run in this delta.

### Verdict (addendum)

- Pass: yes. Full-scope `run-test.sh` exits 0; detector suite 76/76 and scope suite 19/19, identical under sh and dash on 3 runs each; g, h, i, j1, j2, p, q, o1 and the A-1 revert are all caught by the new cases; A-1 holds end to end (8 URL-valued shapes give `no_remote_default:url` and no wrapper output, state file or evidence log contains the value); dotted and slashed remote names are consulted and named in the reason; O-1 runs the golang pack under the real wrappers.
- Fail: no.
- Blocked: no.

Proceeding to `/sync-docs` is appropriate. Two further LOW test gaps (G-5, G-7) are recorded for the operator to decide on; neither is a defect in the shipped detector.
