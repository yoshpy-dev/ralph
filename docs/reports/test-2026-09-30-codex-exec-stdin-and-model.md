# Test report: codex-exec-stdin-and-model

- Date: 2026-09-30
- Plan: docs/plans/active/2026-09-29-codex-exec-stdin-and-model.md (issue #184)
- Tester: tester subagent (Claude Code), cycle 1
- Scope: `git diff main...HEAD` at HEAD `a21882e` on `fix/codex-exec-stdin-and-model`. Behavioral tests only (per `.claude/agents/tester.md`): `./scripts/run-test.sh` (full scope), the two named shell suites, the two named Go checks, live codex round-trips for AC-6 (success and timeout paths) and the `/cross-review` reviewer form, 6 scratch-copy mutations, and a 2-shell portability check. No static analysis, linting, or drift checks were run (that is `/verify`'s job; see `docs/reports/verify-2026-09-30-codex-exec-stdin-and-model.md`, PASS).
- Evidence: `docs/evidence/verify-2026-09-30-020341.log` (gitignored; local copy only), plus live-run artifacts under this session's scratchpad (not committed — see the live-run and mutation tables below for exact commands and observed values).

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 32 shell suites + 8 Go packages | all | 0 | 0 | evidence log has 1533 lines; exit 0 |
| `sh tests/test-codex-exec-invocation.sh` (unmodified) | 96 | 96 | 0 | 0 | < 1s |
| `sh tests/test-ralph-config.sh` | 19 | 19 | 0 | 0 | < 1s |
| `go test ./internal/config/... -count=1 -v -run TestDefaultsLockStep` | 1 | 1 | 0 | 0 | 0.46s |
| `TMPDIR=/tmp go test ./internal/config/... -count=1` | package | ok | 0 | 0 | 0.245s |

The full-scope run's 32 shell suites and 8 Go packages (`internal/cli`, `internal/config`, `internal/insights`, `internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`, `internal/upgrade`) all reported `PASS`/`ok`; a line-by-line grep of the 1533-line evidence log for `FAIL:\s*[1-9]` and any bare `FAIL` outside a `FAIL: 0` context found no hits. `RALPH_VERIFY_SCOPE=full` was used per the assignment (issue #190 — the default scope skips language packs on a pushed branch).

## Live-run checks (AC-6, timeout path, cross-review form, portability)

All live runs used `. ./scripts/ralph-config.sh` sourced in the same Bash call as the launch, exactly as both skills document (`RALPH_CODEX_REVIEWER_MODEL=gpt-6-astra`, `RALPH_CODEX_REASONING_EFFORT=xhigh`).

| Check | Form | `codex rc=` | Elapsed | `-o` file | Leftover process | git status (worktree + main) |
| --- | --- | --- | --- | --- | --- | --- |
| AC-6 success (`/plan` step 11.c, `sleep 1200`, prompt replaced with `Reply with the single word ok`) | `--sandbox read-only -o <file>` | 0 | ~15s | non-empty, content trimmed = `ok` | none | clean / clean |
| Timeout path (same form, `sleep 1200` → `sleep 3`, prompt = "Think carefully, then list the first 300 prime numbers one per line") | same | 0 | ~3.2s | missing (no such file) | none (a `sleep 3` process visible mid-check belonged to an unrelated polling loop in this shared session — confirmed gone on re-inspection, unrelated to our run) | clean / clean |
| `/cross-review` step 4 form (`exec review --base main`, `sleep 1200`) | `exec review --base "$BASE" -o <file>` | 0 | ~291s (4m51s) | non-empty (190 bytes); first line: "No actionable regressions were found. Targeted shell and Go tests, skill/template synchronization checks, and shortened watchdog probes passed; live model-backed review calls were not rerun." (not triaged, per instructions) | none | clean / clean |
| Portability — `dash` (AC-6-style form, `sleep 3`, ok prompt) | same as AC-6 but `sleep 1200`→`sleep 3` | 0 | ~3.1s | (not inspected; script parsed and completed cleanly, which is what step 8 checks) | none | clean / clean |
| Portability — `bash --posix` (same form) | same | 0 | ~3.1s | (same) | none | clean / clean |

Findings:
- AC-6 success matches the plan's design contract exactly: `rc=0` and a non-empty, trimmed-`ok` `-o` file, well under the 60s ceiling this assignment set (and under the skill's own 20-minute watchdog).
- The timeout path confirms the plan's own recorded finding: a `sleep 1200`-style watchdog TERM against codex still reports `codex rc=0` (graceful shutdown on signal), so the skill's documented contract — "the `-o` file check, not `rc`, is what actually catches a timeout" — is exactly right and was directly observed here: `rc=0` alone would have wrongly signaled success, but the `-o` file was absent.
- The `/cross-review` step 4 form completed with `rc=0` and a real, non-empty finding file in about 4m51s, consistent with the plan's own recorded 282s/290s live runs (Deviation notes, 2026-09-30 work).
- Both `dash` and `bash --posix` parsed and ran the extracted one-liner to completion (~3.1s each, `rc=0`), confirming the watchdog form (background siblings, `kill -0` polling, `wait`) is POSIX-portable and not bash-specific despite living inside a bash-run agent tool.
- After every codex call (5 total across this cycle plus the earlier 96/96 unmodified run), `git status --porcelain` was empty in both the task worktree and the main checkout — no `.codex/config.toml` rewrite was observed this cycle (`git diff --stat .codex/config.toml` empty in both checkouts).
- No leftover `codex` CLI process or watchdog `sleep`/subshell was found after any run (`ps aux` filtered for ` codex .*exec`, excluding the unrelated ChatGPT desktop app processes that also happen to contain "codex" in their path).

## Mutation checks (scratch copies only, tracked files untouched throughout)

A scratch mutation root was built at a private scratchpad path mirroring the project layout the test's `PROJECT_ROOT` resolution expects (`tests/`, `scripts/ralph-config.sh`, and the four skill faces under `.claude/skills`, `.agents/skills`, `templates/base/.claude/skills`, `templates/base/.agents/skills`), with a pristine backup restored before each mutation. No file inside the actual worktree or the main checkout was ever edited; `git status --porcelain` was empty in both checkouts throughout this section.

| # | Mutation | Predicted | Observed | Match |
| --- | --- | --- | --- | --- |
| (a) | Split the cross-review step 4 invocation line with a backslash into two lines (`.claude/skills/cross-review/SKILL.md`) | invocation-count assertion fails | RED, but via a different assertion: the split still yields exactly 2 detected invocation lines (the grep detector requires both `codex` and `' exec '` on the same physical line; the second half of the split line still contains both — `codex` from the trailing `echo "codex rc=$rc"` and `exec` from `exec review`), so the count assertion (`has exactly 2 ... invocation line(s)`) still passes. What actually fails are 3 per-line content checks on that now-incomplete second-half line (`missing 'command codex '`, `missing -m ...`, `missing model_reasoning_effort=...`). Net: 93/96, 3 FAILED | Partial — same net result (red), different assertion path than predicted |
| (b) | Remove `command ` from the `/plan` invocation line (`.claude/skills/plan/SKILL.md`) | fails | 95/96, 1 FAILED — `missing 'command codex '` on `.claude/skills/plan/SKILL.md:71` | Yes |
| (c) | Remove `</dev/null` from one face (`.agents/skills/cross-review/SKILL.md`, step-4 line) | fails | 95/96, 1 FAILED — `missing </dev/null` on `.agents/skills/cross-review/SKILL.md:57` | Yes |
| (d) | Remove ` -o <...>` from one face (`templates/base/.claude/skills/plan/SKILL.md`) | fails | 95/96, 1 FAILED — `missing -o / --output-last-message` on `templates/base/.claude/skills/plan/SKILL.md:71` | Yes |
| (e) | Change `sleep 1200` to `sleep 12000` (`.claude/skills/plan/SKILL.md`, both occurrences) | information, not a defect | 96/96, 0 FAILED — no assertion inspects the watchdog sleep duration; the test's grep-based shape checks never touch that literal | Yes (correctly not caught; documented gap, not a defect — matches AC-3's own scope, which is invocation-line shape, not watchdog timing) |
| (f) | Change `RALPH_CODEX_REASONING_EFFORT` default in `scripts/ralph-config.sh` from `xhigh` to `high` | fails | 84/96, 12 FAILED — every `RALPH_CODEX_REASONING_EFFORT fallback matches ...` assertion across all 4 faces now compares against the mutated `high` default and fails, since the skill bodies still say `xhigh` | Yes (matches the verify report's own mutation (iv), also 12 FAILED) |

Mutation (a) is a genuine, if minor, finding: the task description's stated prediction ("the invocation-count assertion fails") does not match what the test actually does when this specific split shape is applied — the detector's own combined `codex` + `' exec '` same-line requirement happens to still find exactly one matching line in the split's second half, so the count stays correct and different per-line content assertions catch the mutation instead. The net effect is the same (the mutation is still caught, red as required), so this is not a coverage gap — just a difference in *which* assertion fires, worth noting for anyone relying on the specific assertion name in future debugging.

## Coverage

- Statement/branch/function: not applicable — `tests/test-codex-exec-invocation.sh` and `tests/test-ralph-config.sh` are POSIX-shell grep/case-based regression guards with no instrumented coverage tool (consistent with this repo's existing shell-test coverage model). `internal/config`'s `TestDefaultsLockStep` is a single deterministic Go test with no partial-coverage concern (it either finds all 5 skill/var fallback pairs correctly or fails).
- Notes: the 6 mutations above (5 requested defect-shaped + 1 informational) exercise all 5 assertion kinds the invocation test performs per line (`command codex `, `</dev/null`, `-m` fallback, `model_reasoning_effort=` fallback, `-o`/`--output-last-message`) plus the fallback-value cross-check against `scripts/ralph-config.sh`'s live defaults. The one assertion kind not independently mutated this cycle is the AC-8 `Reviewer status: incomplete` text check (verify report's mutation (ii) already covered this red path in cycle 1's verify pass, 95/96 1 FAILED) — not re-run here since it was outside this cycle's assigned mutation list.

## Failure analysis

No failures in any unmodified test run this cycle. All 6 mutation-induced failures (a-f) were on scratch copies and matched a genuine, understood code path (see mutation table above); none indicate a defect in the tracked skill bodies, `scripts/ralph-config.sh`, or the tests themselves.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `codex exec` hangs waiting on stdin ("Reading additional input from stdin...", #153) | Fixed and regression-guarded | All 5 live codex round-trips this cycle used `</dev/null` and none hung; `tests/test-codex-exec-invocation.sh` asserts `</dev/null` on every invocation line across all 4 faces (96/96); mutation (c) confirms the guard is red-sensitive |
| Shell-alias `-m` conflict causing a 400 (#162) | Fixed and regression-guarded | Every live run used `command codex -m ... -c model_reasoning_effort=...` and returned `rc=0` with no API error; `tests/test-codex-exec-invocation.sh` asserts `command codex ` and the explicit `-m`/effort flags on every invocation line; mutation (b) confirms `command` removal is red-sensitive |
| A hung/incomplete codex background call silently reported as "no findings" | Fixed and regression-guarded (AC-8) | The timeout-path live run this cycle directly reproduced the exact failure shape the fix targets: `rc=0` with a missing `-o` file — the skill's documented "`-o` file check, not `rc`" contract is what would correctly classify this as incomplete rather than "no findings" |

## Test gaps

- The `-o`/`--output-last-message` mutation (removing the flag from a *cross-review* face, as opposed to the `/plan` face tested here as mutation (d)) was not independently re-run this cycle; the identical code path (`case ... *' -o '*|*'--output-last-message'*`) was already exercised by mutation (d) on the `/plan` face and by the verify report's own equivalent mutation. Low risk given the shared code path.
- Mutation (a)'s actual discriminating mechanism (per-line content checks, not the invocation-count assertion) is not itself covered by a dedicated test that asserts *which* assertion catches a backslash-split line — a future refactor of the detector's `grep 'codex' | grep ' exec '` same-line requirement could silently stop catching this exact split shape without any test failing to alert on it, since no test currently pins the detector's line-locating regex behavior in isolation. Not an AC gap (AC-3 only requires the mutation be caught, which it is), but worth a follow-up unit test on the detector itself if this area gets touched again.
- No coverage of a genuinely malformed `-o` file (e.g. a file that exists but was last written by a stale prior run and is non-empty but stale) was exercised live this cycle; the plan's own AC-2/AC-8 contract addresses this via `rm -f` before every launch (verified present in all 5 live-run invocation lines used), which is the intended mitigation rather than a runtime staleness check.

## Verdict

- Pass: `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` (32 shell suites + 8 Go packages, 0 failures); `tests/test-codex-exec-invocation.sh` (96/96); `tests/test-ralph-config.sh` (19/19); `go test ./internal/config/... -run TestDefaultsLockStep` and `TMPDIR=/tmp go test ./internal/config/...` (both green); AC-6 live success path (rc=0, `-o` file = `ok`, ~15s); timeout path (rc=0, `-o` file absent — the documented completion signal, ~3.2s, no leftover process); `/cross-review` step 4 live form (rc=0, non-empty finding file, ~291s, no leftover process); dash and bash --posix portability (both parse and complete, ~3.1s each); 5 of 6 mutations (b, c, d, f discriminate exactly as predicted; e correctly not caught, informational) plus mutation (a) discriminates via a different-than-predicted assertion but is still correctly red.
- Fail: none.
- Blocked: none. No `.codex/config.toml` rewrite, no leftover codex/sleep/watchdog process, and clean `git status --porcelain` in both the task worktree and the main checkout after every one of the 7 live codex invocations run this cycle (1 unmodified AC-6, 1 timeout, 1 cross-review, 2 portability, plus the earlier plan-time and Slice B live runs already recorded in the plan's own deviation notes).

Overall verdict: **PASS**. Safe to proceed to `/pr`.
