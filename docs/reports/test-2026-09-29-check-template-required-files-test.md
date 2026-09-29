# Test report: check-template-required-files-test

- Date: 2026-09-29
- Plan: docs/plans/active/2026-09-29-check-template-required-files-test.md (issue #183)
- Tester: tester subagent (Claude Code), cycle 1
- Scope: behavioral tests only, at HEAD `e90ffa3` (worktree `.claude/worktrees/check-template-required-files-test`, branch `test/check-template-required-files-test`). No static analysis (verifier's job; see `docs/reports/verify-2026-09-29-check-template-required-files-test.md`, PASS).
- Evidence: `docs/evidence/verify-2026-09-29-080926.log` (default-scope `./scripts/run-test.sh`), `docs/evidence/verify-2026-09-29-081229.log` (`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration | Exit |
| --- | --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (default scope) | 22 shell suites (incl. `tests/test-check-template.sh`) | all | 0 | golang skipped (scope=`changed (no_changes)`) | ~2 min | 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 22 shell suites + golang verifier | all | 0 | — | ~2.5 min | 0 |
| `bash tests/test-check-template.sh` (direct) | 32 | 32 | 0 | 0 | <1s | 0 |
| `go test ./internal/scaffold/... -run TestTemplateBase -v` | 4 | 4 | 0 | 0 | 0.25s | 0 |
| `TMPDIR=/tmp go test ./internal/scaffold/... -count=1` | package | ok | 0 | 0 | 0.27s | 0 |
| Stability: `bash tests/test-check-template.sh` x3 | 32 each | 32 each | 0 each | 0 | ~2s each | 0 each |

Default-scope `./scripts/run-test.sh`'s language-scope detector reported `Language scope: changed (no_changes)` and ran zero golang-specific checks in this pushed task worktree (same root cause the verifier already recorded: `origin/<branch>` already points at HEAD, so the diff base is empty). `tests/test-check-template.sh` itself is a plain shell suite and ran unconditionally in both invocations (`run_hook_tests` enumerates `tests/test-*.sh` regardless of language scope), so its 32/0 result is scope-independent. The full-scope invocation confirms the golang verifier separately: `go test ./...` reported `ok` for every test-bearing package, including `internal/scaffold` (0.748s, not cached — ran fresh).

### `bash tests/test-check-template.sh` — all 32 assertion names (run 1, identical across all 3 stability runs)

```
PASS: A. required_files block is non-empty
PASS: A. required_files matches the golden list (28 entries, in order)
PASS: B. clean fixture passes check-template.sh
PASS: B. removing README.md is detected
PASS: B. removing AGENTS.md is detected
PASS: B. removing CLAUDE.md is detected
PASS: B. removing .claude/settings.json is detected
PASS: B. removing docs/research/approach-comparison.md is detected
PASS: B. removing docs/roadmap/harness-maturity-model.md is detected
PASS: B. removing scripts/run-verify.sh is detected
PASS: B. removing scripts/run-static-verify.sh is detected
PASS: B. removing scripts/run-test.sh is detected
PASS: B. removing scripts/detect-changed-languages.sh is detected
PASS: B. removing scripts/detect-languages.sh is detected
PASS: B. removing scripts/archive-plan.sh is detected
PASS: B. removing scripts/branch-name.sh is detected
PASS: B. removing scripts/ensure-pr-ready.sh is detected
PASS: B. removing scripts/ensure-pr-title-prefix.sh is detected
PASS: B. removing scripts/new-feature-plan.sh is detected
PASS: B. removing scripts/codex-check.sh is detected
PASS: B. removing scripts/ralph-config.sh is detected
PASS: B. removing scripts/ralph-worktree.sh is detected
PASS: B. removing scripts/xreview-helpers.sh is detected
PASS: B. removing scripts/secret-scan.sh is detected
PASS: B. removing scripts/secret-scan-branch.sh is detected
PASS: B. removing scripts/pre-commit-secret-guard.sh is detected
PASS: B. removing scripts/commit-msg-guard.sh is detected
PASS: B. removing scripts/prepare-commit-msg-secret-guard.sh is detected
PASS: B. removing scripts/pre-merge-commit-secret-guard.sh is detected
PASS: B. removing scripts/check-template.sh is detected
PASS: B. removing scripts/check-skill-sync.sh is detected
PASS: C. all 28 golden entries exist at the repo root
```

## Coverage

- Statement/branch/function: no instrumented Go coverage tool run in this cycle (not requested by the plan's Test plan; `internal/scaffold` carries broader package coverage tracked in prior cycles, unaffected by this diff's pure test-file addition).
- Notes: `tests/test-check-template.sh` exercises 3 independent detection paths against the real `required_files` lists (golden comparison, per-entry fixture removal, root-existence sanity), giving each of the 28 golden entries its own individually-attributed assertion — a real regression on any single entry produces a named failure, not just an aggregate count change.

## Stability and interruption checks

- **3 consecutive direct runs** of `tests/test-check-template.sh`: 32 passed / 0 failed every time, no flakiness observed.
- **Leftover fixture check**: `find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'check-template-fixture.*'` returned nothing after all direct/stability runs.
- **Interrupt test**: started the suite in the background, sent `SIGTERM` ~1s in (landed 9 entries into case B's 28-entry loop, confirmed by the captured last lines of output). Result: exit `143` (128+SIGTERM), and the leftover-fixture check immediately after found **no** `check-template-fixture.*` directory — the `trap 'rm -rf "$fixture"' EXIT` set in `run_case_b()` (`tests/test-check-template.sh:147`) fires correctly on signal-induced exit, confirming the self-review's LOW-severity fix (Slice B, `6f3d48a`) holds under a real interrupt, not just the normal-completion path.

## Mutation table

All mutations ran against scratch copies (repo-wide `rsync` copy under the tester's own scratchpad) or, where the task explicitly named the real files, against the real worktree with an immediate `git checkout --` restore — confirmed via `git status --porcelain` empty after every worktree-touching mutation.

| # | Mutation | Expected | Observed | Match |
| --- | --- | --- | --- | --- |
| (a) 1–28 | Remove each of the 28 `required_files` entries, one at a time, from a scratch copy of `scripts/check-template.sh`; run the scratch copy of `tests/test-check-template.sh` against it | Cases A and B (that one entry) fail for every entry | **All 28/28** entries: exit 1, `30 passed, 2 failed` (case A `required_files differs from the golden list` + case B `removing <entry> was NOT detected (exit 0)` for that exact entry), zero exceptions. Scratch `scripts/check-template.sh` restored and `diff`-confirmed byte-identical to the original after the sweep. | Exact |
| (b) | Remove `"xreview-helpers.sh"` from `requiredTemplateScripts` in `internal/scaffold/embed_test.go` (real worktree file); run `go test ./internal/scaffold/... -run TestTemplateBaseScriptsMatch -count=1 -v` | Fail, naming `xreview-helpers.sh` | `FAIL` — `only in check-template.sh: [xreview-helpers.sh]`, `only in the Go list: []`. Restored via `git checkout -- internal/scaffold/embed_test.go`; `git status --porcelain` empty afterward. | Exact |
| (c) | Remove the `scripts/xreview-helpers.sh` line from `templates/base/scripts/check-template.sh` (real worktree file); run the same Go test | Fail, naming `xreview-helpers.sh` | `FAIL` — `only in check-template.sh: []`, `only in the Go list: [xreview-helpers.sh]`. Restored via `git checkout -- templates/base/scripts/check-template.sh`; `git status --porcelain` empty afterward. | Exact |
| (d) shell | In a scratch copy, change `required_files="` to `required_files='`; run the scratch shell test | Case A fails with the empty-block message | Case A: `FAIL: A. required_files block is non-empty (extraction format of <path> may have changed)`. Case B also cascaded to failure (`unexpected EOF while looking for matching \`''`, exit 2) because the single-quote edit leaves the script itself syntactically broken past that point — a natural side effect of this specific mutation, not a discrepancy from the task's stated expectation (which named only case A's message). Scratch file restored and `diff`-confirmed identical. | Exact (case A); case B behavior additional but consistent |
| (d) Go | Same quote-style edit, applied temporarily to `templates/base/scripts/check-template.sh` (real worktree file); run the Go equivalence test | `Fatalf` about zero entries | `FAIL` — `required_files block in <path> yielded 0 entries; the extraction format (required_files="...") may have changed`. Restored via `git checkout -- templates/base/scripts/check-template.sh`; `git status --porcelain` empty afterward. | Exact |
| (e) 1 | `git clone --no-hardlinks` the worktree into scratch; add a bogus `GOLDEN_ENTRIES` entry (`scripts/bogus-not-in-script.sh`) to break the golden list; run `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` | Non-zero, naming `tests/test-check-template.sh` | Exit `1`. Log shows `==> tests/test-check-template.sh` ... `test-check-template: 30 passed, 3 failed` ... `FAIL` (case A diff `28a29 > scripts/bogus-not-in-script.sh`, case B `removing scripts/bogus-not-in-script.sh was NOT detected (exit 0)`, case C `scripts/bogus-not-in-script.sh missing at repo root`). | Exact |
| (e) 2 | `chmod -x tests/test-check-template.sh` in the same clone; rerun `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` | Exit 0, silent skip | Exit `0`. Full log contains **zero** occurrences of `test-check-template` anywhere — confirming `run_hook_tests`'s `[ -x "$f" ] || continue` gate (`scripts/verify.local.sh:213`) skips the non-executable suite without any trace in the output, which is exactly AC-7's stated rationale for requiring `100755`. | Exact |

All 6 mutation categories (32 individual mutation runs total: 28 in (a) + 1 each in (b)/(c)/(d)-Go + 2 in (e), plus (d)-shell) discriminated exactly as the plan and self-review predicted. No mutation went undetected.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #169 mutation 6f: dropping `scripts/xreview-helpers.sh` from `required_files` went undetected by any existing test | Fixed | Mutation (a) entry 20 and mutation (b)/(c) above all detect this exact entry's removal from either list |
| Self-review LOW: fixture directory could leak on interrupted run (no `EXIT` trap) | Fixed, reconfirmed under a real signal | Interrupt test above: `SIGTERM` mid-case-B, no leftover fixture |
| AC-7 rationale: `verify.local.sh`'s `run_hook_tests` silently skips non-executable `tests/test-*.sh` files | Confirmed as designed, `100755` mode present | Mutation (e) both halves; `git ls-files -s tests/test-check-template.sh` → `100755` |

## Test gaps

- **Fresh-scaffold full-suite pass**: not attempted (explicitly out of scope per the plan's Non-goals — `check-template.sh`'s other checks and the meta-repo-only 3-entry gap are #189's scope). See "Fresh scaffold" below for the bounded check this plan's AC-3 does require.
- **Concurrent/parallel invocation** of `tests/test-check-template.sh` (e.g. two sessions running it at once against the same repo root) was not tested; the suite only reads files under the repo root in cases A/C and writes exclusively to its own `mktemp -d` fixture in case B, so no shared-state race is expected, but this was not empirically verified in this cycle.
- **Instrumented Go coverage** (`-coverprofile`) for `internal/scaffold` was not generated in this cycle; the plan's Test plan did not call for it and the new Go test is a small, fully-deterministic text-parsing comparison (no branches beyond the two direction checks, both mutation-tested above).
- Mutation (d)-shell's case B cascade (syntax-broken script) was not independently minimized to isolate case-A-only behavior; not necessary since case A's message matched the task's stated expectation exactly and case B's cascade is an inherent, non-surprising consequence of the same single edit.

## Fresh scaffold

`go run ./cmd/ralph init --yes <scratch>/fresh` succeeded (`EXIT=0`): 147 base files + 3 files each for 6 language packs, git initialized, all 4 git hooks installed.

- All 22 `scripts/` entries named in `requiredTemplateScripts` / the golden list's `scripts/` half are present in the fresh scaffold's `scripts/` directory (28 total `.sh` files there; no `MISSING:` lines from an explicit per-name check).
- `CI=true sh scripts/check-template.sh` inside the fresh scaffold: exit `1`, exactly 3 `Missing required file` lines — `README.md`, `docs/research/approach-comparison.md`, `docs/roadmap/harness-maturity-model.md` — matching AC-3's stated expectation precisely (all three are #189-scoped, meta-repo-only entries). 7 additional `FAIL:` lines are the pre-existing hook-reference false positives (also #189 scope; `.claude/hooks/ralph-dispatch.sh <Event>` greps as a bare path). Total 10 `FAIL:` lines.
- Note for the record: this fresh-scaffold run exits `1`, not `0`. This does not contradict the plan's documented root-repo quirk ("`CI=true ./scripts/check-template.sh` exits 0" at the meta-repo root, `docs/tech-debt/README.md:135`): that fail-open behavior is specific to the hook-reference check's `| while read` subshell dropping its own `status=1`. The `Missing required file` checks are a separate, non-subshelled code path and correctly propagate `status=1` to the parent shell — the root repo never observes this because it has zero missing-required-file failures to propagate, while the fresh scaffold has three real ones. Both observations are consistent with the same underlying mechanism; no new defect found.

## Verdict

- Pass: `./scripts/run-test.sh` (both default and `RALPH_VERIFY_SCOPE=full` invocations), `tests/test-check-template.sh` (direct run x1 + 3 stability runs, 32/0 every time), `go test ./internal/scaffold/... -run TestTemplateBase` (both plain and `TMPDIR=/tmp`), interrupt-safety check, all 6 mutation categories (32 total mutation runs) discriminated exactly as predicted, fresh-scaffold AC-3 bound (10 FAIL lines, exactly the 3 #189-scoped `Missing required file` entries plus 7 pre-existing #189-scoped hook-reference false positives, nothing else).
- Fail: none.
- Blocked: none.
- **Overall verdict: PASS.** No behavioral test failures, no flakiness across 3 stability runs plus a real-signal interrupt test, every mutation this plan calls for was independently reproduced and discriminated as expected, no regressions in the previously-passing suites (`run-test.sh` both scopes stayed at 0 failures across all ~22 shell suites and all Go packages). Recommend proceeding to `/sync-docs`.
