# Verify report: dispatch-test-private-tmpdir

- Date: 2026-09-29
- Plan: docs/plans/active/2026-09-28-dispatch-test-private-tmpdir.md (issue #182)
- Verifier: verifier subagent (Claude Code)
- Scope: `git diff main...HEAD` at HEAD 95934b5 (`tests/test-ralph-dispatch.sh`, `docs/tech-debt/README.md`, plan, self-review report, insight events). Spec compliance (AC-1..AC-7) + static analysis + doc drift. No behavioral test execution (tester's responsibility).

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `bash -n tests/test-ralph-dispatch.sh` | PASS | No syntax errors |
| `shellcheck -S warning tests/test-ralph-dispatch.sh` | PASS | Exit 0, no findings at warning level or above |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS | `check-sync.sh` (159 IDENTICAL / 0 DRIFTED), `check-pipeline-sync.sh`, `check-skill-sync.sh` (13 in lock-step), `check-template-purity.sh`, golang verifier (`gofmt: ok`, 0 issues), `secret-scan-branch.sh` (21321cc..95934b5 against origin/main: clean). Evidence: `docs/evidence/verify-2026-09-29-031731.log` |
| `git diff main...HEAD -- docs/tech-debt/README.md` | PASS | Exactly one row removed (the case I flake row), no other change |
| `grep -n "test-ralph-dispatch\|pgrep\|pkill\|10-slow\|10-concurrent" docs/tech-debt/README.md` | PASS (no match) | No dangling tech-debt row for case I or the pgrep/pkill non-goal (Slice C fixed it in-cycle instead of leaving debt) |
| `grep -n "i_tmpdir_base\|i_leaked_tmp\|i_tmp_before\|i_tmp_after" tests/test-ralph-dispatch.sh` | PASS (no match) | Old shared-dir snapshot/diff variables fully removed |
| `grep -n 'TMPDIR:-/tmp' tests/test-ralph-dispatch.sh` | PASS | Only hit is line 85 (`workdir=$(mktemp -d "${TMPDIR:-/tmp}/...")`), the sole host-shared-dir use |
| find/rm target audit (all `find`/`rm` call sites) | PASS | Every target resolves under `$workdir` (`$fixture`, `$shared_tmp`, `$case_i_tmpdir`, or `$workdir/...` files) |
| `grep -n "AC-" tests/test-ralph-dispatch.sh` | PASS (1 pre-existing hit) | Line 21 `(AC-5)` predates this branch (`git show main:tests/test-ralph-dispatch.sh` has it; introduced in `24a611a`, case H's header line) — not a comment this plan touched. All other AC references were replaced with `#182` |

## Observational checks (self-review fix cross-check)

The 2026-09-29 self-review (cycle 1, `b45326e`) found 0 CRITICAL/HIGH, 1 MEDIUM, 5 LOW. Cross-checked each against current code:

| Finding | Status | Evidence |
| --- | --- | --- |
| MEDIUM: `pkill -f "Stop.d/10-concurrent-slow.sh"` cross-run collision | Fixed | Hook renamed to `10-concurrent-slow-$$.sh` (`tests/test-ralph-dispatch.sh:431`); both `pkill` calls against the concurrent fixture removed — stop is `kill -TERM "$concurrent_pid"` + `wait` only (`:572-575`); cleanup comment at `:93-95` states "No pkill safety net is needed" |
| LOW: `sleep 30` orphan | Fixed | Hook ends in `exec sleep 30` (`:435`), so `kill_child`'s TERM lands on the sleep itself |
| LOW: no baseline check before fixture start; find used `$TMPDIR` not `$shared_tmp`; concurrent subshell inherited TMPDIR implicitly | Fixed | Baseline assertion added (`:438-449`); all three `find` calls use `$shared_tmp` explicitly (`:444`, `:475`, `:592`); concurrent subshell sets `TMPDIR="$shared_tmp"` explicitly (`:453-454`), mirroring case I |
| LOW: private-dir leak check filtered by `-name 'ralph-dispatch-*'` | Fixed | Now `find "$case_i_tmpdir" -mindepth 1 -maxdepth 1` (unfiltered, `:541`), plus explicit fail when the directory itself is missing (`:538-539`) |
| LOW: PASS line embedded absolute temp paths | Fixed | Presence PASS now prints a count only (`:476-480`); paths remain only on FAIL branches |
| LOW: 4 stale comments (AC references, suite-wide export reach, "reproduces the real flake" overstatement, `concurrent_pid` lifetime description) | Fixed | AC refs → `#182`; shared-dir comment now states cases A-H and case H's run-verify.sh/run-test.sh land there too (`:104-115`); wording is now "models the condition behind the flake in #182" (`:406`); cleanup comment now describes `concurrent_pid` as "set right after the fixture is launched... stays set until case I's own explicit stop clears it" (`:90-95`) |

All six self-review findings are resolved in the code at HEAD.

## AC-by-AC

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 (private TMPDIR for case I's target, no shared-dir snapshot/rm) | PASS | Target dispatcher subshell sets `TMPDIR="$case_i_tmpdir"` before `exec` (`:495-500`); leak check lists only `$case_i_tmpdir` (`:538-547`); no `rm -f $i_leaked_tmp` or shared-dir `comm -13` diff anywhere in the file |
| AC-2 (concurrent fixture, started-marker gate, presence assertion before case I runs) | PASS | Second dispatcher under `$concurrent_fixture` (`Stop.d/10-concurrent-slow-$$.sh`, different event/hook) starts at `:451-456`; started-marker poll with explicit FAIL text if absent within 5s (`:458-467`); presence assertion (`:469-480`) runs before case I's own dispatcher launch (`:495`) |
| AC-3 (red evidence recorded, consistent with efd4ec1) | PASS (documentary) | Plan Deviation notes (Slice A) record 25 PASS / 1 FAIL with message "left stray ralph-dispatch-* temp files: …/shared/ralph-dispatch-{merged,out,stdin}.*" from a scratch copy of `efd4ec1` with the fixture inserted after the pre-snapshot. Read `git show efd4ec1:tests/test-ralph-dispatch.sh` directly: it takes `i_tmpdir_base="${TMPDIR:-/tmp}"`, snapshots via `find ... \| sort` before/after, diffs with `comm -13`, and fails with exactly that message shape (`rm -f $i_leaked_tmp` present) — consistent with the claimed reproduction. Not independently re-run here (scratch-only per plan, not committed); tester is the AC's execution-evidence owner |
| AC-4 (concurrent fixture stopped at case I's end + EXIT trap safety net) | PASS | Explicit stop after case I's checks: `kill -TERM "$concurrent_pid"` + `wait`, `concurrent_pid=""`, then absence assertion (`:572-597`); EXIT trap `cleanup()` also TERMs+waits on `$concurrent_pid` if still set (`:87-101`) |
| AC-5 (A-I all pass, shellcheck clean, run-verify.sh green) | Static half PASS, behavioral half deferred to tester | shellcheck/`bash -n`/`run-static-verify.sh` all pass (table above). 5-consecutive-run and pass-count evidence is recorded in the plan's Deviation notes (29/0, 5 runs); this verifier did not re-execute the suite per the assigned scope |
| AC-6 (header describes new checks; tech-debt row removed) | PASS | Header `i.` block (`:22-38`) describes the private TMPDIR check, the concurrent fixture, and the per-run hook naming; `docs/tech-debt/README.md` diff shows exactly the case I row deleted, no dangling reference elsewhere in that file |
| AC-7 (no find/rm outside $workdir; static + tester's dynamic canary) | Static half PASS | Full find/rm target audit above — every target is under `$workdir`. Host-shared canary (`ralph-dispatch-canary.*` surviving the suite run) is tester's dynamic check per the plan's test plan, not re-run here |

Also confirmed: the plan's Non-goals second-bullet revision (2026-09-29, pgrep/pkill self-collision brought into scope via Slice C) is recorded with a reason in the plan body; test comments cite `#182` rather than plan AC numbers (one pre-existing `(AC-5)` exception, not introduced by this diff, see table above); `scripts/verify.local.sh` discovers the suite via a `tests/test-*.sh` glob (`:213`), so no runner change was needed and none was made.

## Coverage gaps

- Behavioral execution of the suite (pass counts, 5-consecutive-run stability, AC-3's red reproduction) was not re-run by this verifier — that is `/test`'s responsibility per the assigned scope, and per `.claude/rules/ralph/post-implementation-pipeline.md`'s step separation (`verifier`: static + spec compliance; `tester`: behavioral).
- AC-3's red evidence was cross-checked only by reading `efd4ec1`'s code shape and comparing it against the plan's recorded failure message; the scratch-copy reproduction itself was not independently re-executed.
- AC-7's dynamic canary check (host-shared-dir file surviving a full suite run) was not performed here; it is explicitly assigned to `/test` in the plan.

## Verdict

- Verified: AC-1, AC-2, AC-4, AC-6, AC-7 (static half); static analysis (`bash -n`, `shellcheck -S warning`, `run-static-verify.sh` full scope incl. `check-sync.sh`, `check-pipeline-sync.sh`, `check-skill-sync.sh`, `check-template-purity.sh`, golang verifier, `secret-scan-branch.sh`); all six 2026-09-29 self-review findings resolved in code; doc drift (tech-debt row removed cleanly, header comment in sync, no stray references).
- Partially verified: AC-3 (documentary cross-check against `efd4ec1` and the plan's recorded evidence, not independently re-executed), AC-5 (static half only; pass-count/5-run evidence taken from the plan's Deviation notes).
- Not verified (deferred to `/test` by design): AC-5's and AC-7's behavioral/dynamic halves — full-suite pass count, 5-consecutive-run stability, and the host-shared-dir canary survival check.
- Overall verdict: **PASS**. No spec-compliance gaps, no static-analysis failures, no unresolved self-review findings, no documentation drift found within this verifier's scope.
