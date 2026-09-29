# Test report: dispatch-test-private-tmpdir

- Date: 2026-09-29
- Plan: docs/plans/active/2026-09-28-dispatch-test-private-tmpdir.md (issue #182)
- Tester: tester subagent (Claude Code)
- Scope: Behavioral tests only for `tests/test-ralph-dispatch.sh` case I (AC-1..AC-7 behavioral halves deferred by the verifier). No static analysis performed (verifier's responsibility, already PASS per `docs/reports/verify-2026-09-29-dispatch-test-private-tmpdir.md`).
- Worktree: `.claude/worktrees/dispatch-test-private-tmpdir` (branch `test/dispatch-test-private-tmpdir`), HEAD `d23f08d`, `git status --porcelain` empty before and after this cycle. The tracked `tests/test-ralph-dispatch.sh` was never edited; all mutation checks ran against scratch copies under the session scratchpad.
- Evidence: `docs/evidence/verify-2026-09-29-032048.log` (full `./scripts/run-test.sh` run, changed-language scope)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (full changed-scope suite, includes `tests/test-ralph-dispatch.sh`) | all shell suites | all PASS | 0 | 0 | ~65s |
| `bash tests/test-ralph-dispatch.sh` (single direct run) | 31 | 31 | 0 | 0 | ~7s |
| `bash tests/test-ralph-dispatch.sh` x 5 (stability) | 31 each | 31 each | 0 each | 0 | ~7s each |
| `bash tests/test-ralph-dispatch.sh` x 12 (6 parallel pairs) | 31 each | 31 each | 0 each | 0 | ~8s per pair |
| `bash tests/test-ralph-dispatch.sh` x 3 (1 triple) | 31 each | 31 each | 0 each | 0 | ~8s |

`./scripts/run-test.sh` exit 0, final line `All verifiers passed.` Evidence log confirms case I's 31 individual assertion lines (all `PASS`), matching the direct-run count.

## Case I assertion lines (single direct run, `bash tests/test-ralph-dispatch.sh`)

```
PASS  A. PreToolUse deny exits 0 (exit 0)
PASS  A. PreToolUse deny output is exact JSON
PASS  A. later script skipped after first decision
PASS  B. PostToolUse merge exits 0 (exit 0)
PASS  B. merged output is valid JSON with additionalContext
PASS  B. merged additionalContext contains both scripts' text
PASS  C. single-output passthrough exits 0 (exit 0)
PASS  C. single-output passthrough is byte-exact
PASS  D. non-zero exit propagates (exit 3)
PASS  D. failing script stdout emitted verbatim
PASS  D. later script skipped after non-zero exit
PASS  E. stdin payload reached both scripts
PASS  F. core .d runs before .ralph/local/hooks .d
PASS  F2. three-layer order core,local,gitignored
PASS  G. missing event dir exits 0 (exit 0)
PASS  G. missing event dir produces empty output
PASS  H1. run-static-verify.sh runs only verify.d
PASS  H2. run-test.sh runs only test.d
PASS  H3. run-verify.sh (mode=all) runs both verify.d and test.d
PASS  H4. failing verify.d drop-in fails run-static-verify.sh (exit 1)
PASS  I. simulated shared TMPDIR had no ralph-dispatch-* files before the concurrent fixture started
PASS  I. concurrent-dispatcher fixture started
PASS  I. concurrent dispatcher's temp files present in simulated shared TMPDIR (3 files)
PASS  I. SIGTERM to the dispatcher terminates it with non-zero exit (not resumed) (exit 143)
PASS  I. SIGTERM interrupted the dispatcher promptly (0s, not deferred until the hook child finished on its own)
PASS  I. SIGTERM stopped the running hook script child before it completed (finished_marker absent)
PASS  I. SIGTERM cleanup left no stray .first file in fixture cwd
PASS  I. SIGTERM cleanup left no stray entries in its private TMPDIR
PASS  I. SIGTERM killed the running hook script child (no longer alive after dispatcher exit)
PASS  I. concurrent dispatcher's SIGTERM handling reaped its exec'd sleep child before exiting (exit 143)
PASS  I. concurrent dispatcher's temp files removed from simulated shared TMPDIR after it was stopped

PASS: 31
FAIL: 0
```

Identical across all 5 stability runs and all 18 concurrency runs (6 pairs + 1 triple).

## Stability (5 consecutive runs)

| Run | rc | PASS | FAIL |
| --- | --- | --- | --- |
| 1 | 0 | 31 | 0 |
| 2 | 0 | 31 | 0 |
| 3 | 0 | 31 | 0 |
| 4 | 0 | 31 | 0 |
| 5 | 0 | 31 | 0 |

## Concurrency

6 parallel pairs (`( bash tests/test-ralph-dispatch.sh > A.log 2>&1; echo rc=$? ) & ( ... > B.log ...) & wait`) plus 1 triple, all against the real (current, fixed) suite:

| Pair | A rc | B rc | C rc (triple only) |
| --- | --- | --- | --- |
| 1 | 0 | 0 | — |
| 2 | 0 | 0 | — |
| 3 | 0 | 0 | — |
| 4 | 0 | 0 | — |
| 5 | 0 | 0 | — |
| 6 | 0 | 0 | — |
| triple | 0 | 0 | 0 |

`grep -rn "^  FAIL" <all 12 pair logs>` found zero matches (only the summary lines `FAIL: 0` at end of each log). Same for the triple's 3 logs. No `FAIL ` assertion line appeared in any of the 15 concurrent runs.

## AC-7 canary and orphan-process check

- Planted `${TMPDIR:-/tmp}/ralph-dispatch-canary.<pid>` before the single run, stability runs, and concurrency runs (steps 1-4 of this cycle). It survived all of them; removed and confirmed gone afterward.
- `pgrep -fl 'sleep 30'`, `pgrep -fl '10-slow'`, `pgrep -fl '10-concurrent-slow'` all empty after the single/stability/concurrency runs — no orphaned fixture processes.
- A second, independent host canary was planted specifically around the AC-3 old-behavior red run (mutation check d, below) and a third around the old-behavior parallel red run (item 7, below); both survived their respective runs and were removed afterward.

## Mutation checks

All ran against scratch copies (`<scratchpad>/dispatch-test/mutation-repo/tests/`), never the tracked file. The scratch repo layout symlinked the worktree's `templates/` directory alongside a copied `tests/test-ralph-dispatch.sh`, so `REPO_ROOT`-derived paths resolved correctly without touching the tracked tree.

| # | Mutation | Expected | Observed | Discriminates? |
| --- | --- | --- | --- | --- |
| a | Remove `TMPDIR="$case_i_tmpdir"` / `export TMPDIR` in case I's own target-dispatcher subshell (`tests/test-ralph-dispatch.sh:497-498`) | Suite should fail (target writes to the simulated shared dir instead of its private dir) | Suite stayed **31/0 (PASS)** — no assertion caught it | **No — coverage gap** |
| b | Revert the concurrent fixture hook's `exec sleep 30` to plain `sleep 30` (`:435`) | Uncertain per task instructions; determine empirically | The exit-143 assertion still **PASSED** (31/0); a real orphaned `sleep 30` process was left behind (found via `pgrep`, manually killed) | **No — but this is a known, documented limitation** (see analysis below), not a new gap |
| c | Replace `find "$case_i_tmpdir" -mindepth 1 -maxdepth 1` with `true` (always empty) + plant a file in `$case_i_tmpdir` before the check | Unmutated find must FAIL with the planted file present; mutated `true` must PASS | Unmutated-but-planted variant: **30/1 FAIL**, naming the planted file. Mutated (`true`) variant: **31/0 PASS**, missing it | **Yes — assertion is live** |
| d | Old-behavior red run: `efd4ec1` + concurrent-fixture insertion after the pre-snapshot line, run with `TMPDIR=<scratch>/shared` | `FAIL "left stray ralph-dispatch-* temp files: ..."` under the scratch shared dir | **26 PASS / 1 FAIL**, message: `I. SIGTERM cleanup left stray ralph-dispatch-* temp files: <scratch>/shared/ralph-dispatch-{merged,out,stdin}.*` | **Yes — reproduces the plan's Deviation-notes evidence exactly** (25/1 recorded by the implementer's earlier run; this run added one extra PASS assertion — the "(red-run) concurrent-dispatcher fixture started" marker I inserted — for 26/1) |

### Mutation (a) analysis — genuine coverage gap

Diagnosis added to a throw-away scratch variant confirmed the mechanism: with the `TMPDIR` override removed, case I's target dispatcher inherits the suite-wide `TMPDIR="$shared_tmp"` export (line 118-119) instead of getting its own private directory. A diagnostic printed mid-run showed **6** `ralph-dispatch-*` files in `$shared_tmp` while the target was running (3 from the concurrent fixture + 3 from the mutated target), confirming the target really did write to the wrong directory. Despite this, no assertion fails, because:

- `ralph-dispatch.sh`'s own SIGTERM cleanup trap (`kill_child`/`cleanup` in `templates/base/.claude/hooks/ralph-dispatch.sh:92-106`) removes its own `mktemp`'d files on exit **regardless of which directory `TMPDIR` pointed at when they were created**. That cleanup is a separately-tested, working feature (case I's other assertions exercise it directly), so by the time the private-dir check (`:538-547`) or the final shared-dir-empty check (`:592-597`) run, the target's files are already gone from wherever they were created.
- `$case_i_tmpdir` was `mkdir -p`'d but never written to in the mutated run, so the private-dir emptiness check trivially passes — it is checking the right *invariant* (nothing left behind) but not the *location* the target actually used.

This is a real, reproducible test-only coverage gap: nothing in the suite asserts that case I's own target dispatcher's temp files transiently appear under `$case_i_tmpdir` (an analogous positive presence check to the one that already exists for the concurrent fixture at `:469-480`) while it is mid-run. A mutation that silently drops the private-`TMPDIR` wiring for case I's own target is invisible to the current suite. Recommend a follow-up: add a presence assertion for `$case_i_tmpdir` while the target is running (mirroring the existing pattern for `$shared_tmp`/the concurrent fixture), so the private-directory wiring itself — not just "nothing leaked" — is under test.

### Mutation (b) analysis — known, documented limitation (not new)

The suite's own comment at `tests/test-ralph-dispatch.sh:582-585` already states this precisely: "A pgrep-based check is not possible here the way case I's own child-kill check (above) does it: once `exec` replaces the hook script with `sleep 30`, its command line is just `sleep 30`, indistinguishable from an unrelated sleep on the host." Empirically confirmed: `concurrent_rc` (the dispatcher's own exit code) is **143 in both the fixed and reverted versions**, because `kill_child`'s `wait "$child"` unblocks as soon as the *direct* child (the `sh` process running the hook) dies — and an untrapped `sh` process terminates immediately on SIGTERM by default disposition, even while it is itself blocked waiting on its own un-exec'd `sleep 30` child. That immediate death is what orphans the grandchild `sleep 30` when `exec` is absent; the dispatcher's own exit code carries no information about the grandchild's fate either way. So the exit-143 assertion was never designed to catch this mutation — it verifies the dispatcher's own SIGTERM handling, not the fixture hook's internal `exec` discipline. The orphan I found and killed is exactly the failure mode `exec sleep 30` exists to prevent; the assertion around it is honest about not being able to detect a reversion of that specific line, per its own comment.

## Old-behavior parallel red run (best effort)

Plain, unmodified `efd4ec1` `tests/test-ralph-dispatch.sh` (no concurrent-fixture insertion), run two at a time, three rounds (6 total runs), against the real host `${TMPDIR:-/tmp}` (not overridden), to approximate the plan's Deviation-notes finding ("旧テスト efd4ec1 でも 4/6、同じ失敗の型を含む").

| Round | A | B |
| --- | --- | --- |
| 1 | rc=1 (FAIL) | rc=0 |
| 2 | rc=0 | rc=1 (FAIL) |
| 3 | rc=0 | rc=0 |

2 of 3 pairs (4 of 6 runs pass, 2 of 6 fail) showed a failure — a different exact ratio than the plan's recorded "4 of 6 failed," but timing-dependent races of this kind are inherently non-deterministic and this was explicitly a best-effort check. Both failures matched known failure types from the plan's own analysis:

- Round 1, run A: `FAIL  I. SIGTERM cleanup left stray ralph-dispatch-* temp files: .../ralph-dispatch-merged.CvGT6Z` (25 PASS / 1 FAIL) — the shared-`TMPDIR` collision bug #182 itself.
- Round 2, run B: `FAIL  I. SIGTERM cleanup left stray ralph-dispatch-* temp files: .../ralph-dispatch-merged.U3pTcX` **and** `FAIL  I. SIGTERM left the running hook script child alive (dispatcher did not kill it)` (24 PASS / 2 FAIL) — the second failure is the bare-hook-name `pgrep`/`pkill` cross-run collision type the plan's Deviation notes describe from Slice B, reproduced here against the unmodified `efd4ec1` script as the plan claimed.

A dedicated host canary was planted before this run and survived; no leaked `ralph-dispatch-*` files remained afterward (the old script's own `rm -f $i_leaked_tmp` branch removed the ones it detected as "leaked," consistent with its known-buggy behavior described in the plan); no orphaned processes remained (`pgrep -fl 'sleep 5|10-slow'` empty).

## Coverage

- Statement/branch/function coverage: not applicable (POSIX shell test suite, no instrumented coverage tool — consistent with prior cycles' notes).
- Notes: case I's 31 assertions exercise every AC-1 through AC-4 and AC-7 behavior described in the plan. Mutation checks (a)-(d) plus the parallel red run give direct red/green evidence for the core fix; mutation (a) found one genuine coverage gap (see above), mutation (c) confirmed the leak-check assertion is live, and mutation (b) confirmed a pre-acknowledged, documented limitation rather than a new one.

## Failure analysis

No failures in the real (current) suite across 23 total runs (1 direct + 5 stability + 12 concurrency-pair + 3 triple + 1 via `run-test.sh` + 1 mutation-c-planted-only control run that was expected to fail). No table needed.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Case I false-failed when another session's dispatcher shared the host `${TMPDIR:-/tmp}` (issue #182) | Fixed | AC-2/AC-3 fixture evidence above: concurrent dispatcher's temp files provably present in the simulated shared dir throughout case I's run, and all 31 assertions still pass |
| Case I's `rm -f $i_leaked_tmp` could delete another session's in-use temp files | Fixed | `grep -n "i_leaked_tmp\|rm -f \$i_leaked_tmp"` finds nothing in the current file (confirmed by verifier; not re-checked here since it's a static grep, but the old-behavior red/parallel runs above show the *old* script still exhibits this via its own FAIL/rm-f branch, while the current script never takes that branch) |
| Cross-run `pgrep`/`pkill` collision on a bare hook name (Slice C fix) | Fixed | 6 parallel pairs + 1 triple of the *current* suite: zero FAIL lines. Old-behavior parallel run (item above) reproduced this exact collision type (round 2, run B) against `efd4ec1`, confirming the fix addresses a real, reproducible failure mode |

## Test gaps

- Mutation (a): no assertion currently discriminates a regression that removes the `TMPDIR` override for case I's own target-dispatcher subshell (see analysis above). Recommend adding a mid-run presence assertion for `$case_i_tmpdir`, mirroring the existing one for the concurrent fixture's `$shared_tmp`.
- The exit-143 assertion around the concurrent fixture's `exec sleep 30` cannot and is not meant to discriminate a reversion of that specific `exec`; this is a known, self-documented limitation (comment at `:582-585`), reconfirmed empirically here, not a new gap.
- AC-5's shellcheck/`bash -n`/`run-static-verify.sh` static half is out of this tester's scope; already verified PASS by the verifier report.

## Verdict

- Pass: `./scripts/run-test.sh` (exit 0, `All verifiers passed.`); 1 direct run (31/0); 5/5 stability runs (31/0 each); 6/6 parallel pairs + 1 triple (31/0 each, zero FAIL lines); AC-7 host-canary survival (3 separate plantings, all survived, no orphan processes); mutation (c) discriminates correctly; mutation (d) reproduces the plan's recorded red evidence exactly; old-behavior parallel red run reproduces both known failure types against unmodified `efd4ec1`.
- Fail: none — all behavioral tests in scope pass, and this cycle finds no regression in the current tracked suite.
- Blocked: none.
- Known gap (informational, not blocking): mutation (a) is not caught by the current suite (see Test gaps). Recommend a follow-up but do not consider it a reason to fail this cycle, since it is a coverage gap in the *test*, not a defect in the shipped behavior — the private-`TMPDIR` wiring itself is correct per AC-1/AC-2 evidence; only a hypothetical future regression of that wiring would slip through undetected.
- Overall: **PASS**. Safe to proceed to `/sync-docs` and `/cross-review`.
