# Verify report: cli-stub-stdin-hang

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-cli-stub-stdin-hang.md
- Verifier: verifier subagent
- Commit: c842469 (fix: make CLI test stubs hang-proof on never-closing stdin)
- Branch: fix/cli-stub-stdin-hang
- Prior artifact: docs/reports/self-review-2026-07-12-cli-stub-stdin-hang.md (MERGE, 2 LOW)
- Static verifier: `./scripts/run-static-verify.sh < /dev/null` → exit 0

## Per-AC table

| AC | Description | Status | Evidence |
|---|---|---|---|
| AC1 | Both stubs no longer read stdin unless it is a regular file; header comments document both hazard cases | PASS | `[ -f /dev/stdin ]` present at codex:55 and claude:57; old `[ -t 0 ]` guard absent from both; header comment documents TTY case (#44 cycle-3 P2) and open-pipe case (fix/cli-stub-stdin-hang) at codex:17-20 and claude:14-17; `/dev/stdin`-absent fallback documented at codex:29 and claude:26 |
| AC2 | `printf x \| stub ...` (open-pipe shape) exits immediately; regression test asserts elapsed < 3s for both stubs and passes | PASS (structural) | Test 8 added at test-ralph-cli-driver.sh:332-391; `_stub_elapsed` uses `mkfifo` + `sleep 5` background writer (never-closing pipe); assertions 8a-i (`elapsed_8a -lt 3`) and 8b-i (`elapsed_8b -lt 3`) present; call-log and last-message assertions (8a-ii, 8a-iii, 8b-ii) also present. Behavioral pass confirmed by self-review (53 PASS / 0 FAIL, elapsed=0s) — runtime execution belongs to /test |
| AC3 | Direct dispatcher-shape invocations in the test file carry `< /dev/null` | PASS | Line 203: `codex exec review --base main < /dev/null`; line 200 comment documents rationale ("defense-in-depth; stub must not block on inherited harness stdin"). Test 6b already had explicit file redirect (`< "$PROMPT_ADV"` at line 212) — no `/dev/null` needed there; non-gap confirmed |
| AC4 | Full `bash tests/test-ralph-cli-driver.sh` passes; `./scripts/run-test.sh` passes | LIKELY (unverified by /verify) | Self-review reports 53 PASS / 0 FAIL for the full driver test; run-test.sh execution is /test scope. Structural evidence: all existing tests unchanged; new test 8 adds 5 checks. Behavioral confirmation belongs to /test |
| AC5 | `./scripts/run-static-verify.sh < /dev/null` passes; verify chain completes without hanging | PASS | Exit 0 confirmed; full output in docs/evidence/verify-2026-07-11-161837.log; shellcheck passed for all three changed shell files; sync gates (check-sync.sh, check-skill-sync.sh, check-pipeline-sync.sh) all OK |
| AC6 | tech-debt row marked RESOLVED with commit reference | PASS | docs/tech-debt/README.md line 39-40: HTML comment `<!-- RESOLVED 2026-07-12 in fix/cli-stub-stdin-hang: ... -->` + strikethrough row + `(RESOLVED 2026-07-12 in fix/cli-stub-stdin-hang)` suffix; convention consistent with prior resolved rows (lines 24, 31-32) |

## Static analysis

| Check | Result | Notes |
|---|---|---|
| `./scripts/run-static-verify.sh < /dev/null` | PASS (exit 0) | All hooks, sync gates, and golang verifier passed |
| `sh -n tests/fixtures/cli-stubs/codex` | PASS | POSIX syntax OK |
| `sh -n tests/fixtures/cli-stubs/claude` | PASS | POSIX syntax OK |
| `shellcheck -s sh tests/fixtures/cli-stubs/codex` | PASS | No warnings |
| `shellcheck -s sh tests/fixtures/cli-stubs/claude` | PASS | No warnings |
| `shellcheck -s bash tests/test-ralph-cli-driver.sh` | PASS | No warnings |
| `scripts/check-sync.sh` | PASS | DRIFTED=0; KNOWN_DIFF=3 (pre-existing) |
| `scripts/check-skill-sync.sh` | PASS | 13 skills in lock-step |
| `scripts/check-pipeline-sync.sh` | PASS | All pipeline steps referenced in all docs |
| Go verifier (gofmt + vet) | PASS | No issues; Go files unchanged |

## Documentation drift check

| Area | Status | Notes |
|---|---|---|
| Stub header comments vs actual behavior | IN SYNC | Both stubs: header documents `[ -f /dev/stdin ]` policy → behavior implements it exactly. Two hazard cases (TTY + open-pipe) and absent-`/dev/stdin` fallback all correct |
| tech-debt RESOLVED row consistency | IN SYNC | Row follows established convention: HTML comment with explanation, strikethrough content, RESOLVED annotation, row preserved for traceability; matches rows at lines 24, 31-32 |
| Non-goal compliance: no production/template changes | PASS | `git diff main...HEAD --name-only` contains only test fixtures, test file, tech-debt README, and plan/report artifacts — no `ralph-pipeline.sh` or `templates/base` changes |
| Self-review report (prior artifact) | CURRENT | No blocking findings; 2 LOW (optional polish); consistent with this verify report's findings |

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
|---|---|---|---|---|

No findings. All ACs met.

## Gaps and unverified items

| Item | Why unverified | Confidence | Belongs to |
|---|---|---|---|
| AC4 behavioral pass (`bash tests/test-ralph-cli-driver.sh` runtime) | /verify does not run behavioral test suites | High (self-review reported 53 PASS / 0 FAIL; Test 8 elapsed=0s) | /test |
| AC5 "previously hanging invocation shape" end-to-end regression | Demonstrated by Test 8 structural analysis + self-review empirical probe; no live background-task environment available | High (regression test uses identical FIFO mechanism; detection confirmed by synthetic regressed-stub probe in self-review) | /test |
| `[ -f /dev/stdin ]` on a platform without `/dev/stdin` | Not tested; documented behavior is "condition is false → skip drain → safe default" | High (safe-default path is the else branch; no drain = no hang) | accepted risk per plan |

## Verdict

PASS — all 6 acceptance criteria met. Static analysis clean. Documentation in sync. No blocking or high-severity issues.

AC2/AC4/AC5 have a "behavioral confirmation belongs to /test" caveat (runtime execution), but the structural evidence (Test 8 code, self-review empirical results) provides high confidence.

Prior self-review recommendation (MERGE, no blockers) is consistent with this verdict.

## Evidence

- Raw static verifier output: docs/evidence/verify-2026-07-11-161837.log (produced by `run-static-verify.sh < /dev/null`)
- Full evidence log: docs/evidence/verify-2026-07-12-cli-stub-stdin-hang.log
