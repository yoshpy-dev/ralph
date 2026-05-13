# Test report: fix-codex-review-wording-residues

- Date: 2026-05-13
- Plan: docs/plans/active/2026-05-13-fix-codex-review-wording-residues.md
- Tester: Claude Code (`tester` subagent)
- Scope: Run the existing test suite to confirm no regression from the 6-file string-only rename ("codex review" / "codex ACTION_REQUIRED" → "cross-review" / "cross-review ACTION_REQUIRED"). The plan's test plan explicitly says "Unit tests: 既存テストの green 維持確認（ロジック変更なし）"; no new tests are required for a documentation-only diff.
- Evidence: `docs/evidence/test-2026-05-13-fix-codex-review-wording-residues.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (orchestrator wrapper, `HARNESS_VERIFY_MODE=test`) | — | — | 0 | — | — |
| ├─ `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | — |
| ├─ `tests/test-check-skill-sync.sh` | 6 | 6 | 0 | 0 | — |
| ├─ `tests/test-ralph-cli-driver.sh` | 48 | 48 | 0 | 0 | — |
| └─ Go: `go test ./...` (cached, run via packs/languages/go/verify.sh) | — | all `ok` | 0 | 0 | cached |
| `go test ./... -count=1` (forced fresh run, no cache) | — | all `ok` | 0 | 0 | ~5.2s max pkg |
| `tests/test-ralph-config.sh` (explicit) | 27 | 27 | 0 | 0 | — |
| `tests/test-ralph-signals.sh` (explicit) | 3 | 3 | 0 | 0 | — |
| `tests/test-ralph-status.sh` (explicit) | 40 | 40 | 0 | 0 | — |
| `tests/test-xreview-gate-regression.sh` (explicit) | 21 | 21 | 0 | 0 | — |
| `tests/test-xreview-prompt-render.sh` (explicit) | 54 | 54 | 0 | 0 | — |
| **Shell test total** | **210** | **210** | **0** | **0** | — |
| **Go packages with tests** | 9 packages (action, cli, config, scaffold, state, ui, ui/panes, upgrade, watcher) | all `ok` | 0 | 0 (no SKIPs visible at package summary level) | ~5s per pkg |

`./scripts/run-test.sh` exit code: **0**. `go test ./... -count=1` exit code: **0**.

Note: `scripts/verify.local.sh` in `test` mode wires only 3 of the 8 shell suites (mojibake, skill-sync, cli-driver). The other 5 (`config`, `signals`, `status`, `xreview-gate-regression`, `xreview-prompt-render`) are not orchestrated by `run-test.sh` today. They were executed explicitly here to confirm the rename did not break suites the orchestrator does not see. (This is a pre-existing coverage-orchestration gap, not a regression introduced by this diff — flagged below under Test gaps.)

## Coverage

- Statement: not measured (Go `-coverprofile` not enabled in `run-test.sh`); shell suites have no instrumented coverage tool.
- Branch: not measured.
- Function: not measured.
- Notes: For a string-only documentation diff with **Non-goals: 挙動変更は一切なし**, instrumented coverage does not move and is not a meaningful gate. The relevant question — "does any existing test assert on the literal strings we changed?" — was answered by a targeted grep (see Regression checks).

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures. Pure green.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| String literal "codex review" / "codex ACTION_REQUIRED" used in test assertions | **Not used in any test file** | `git grep -nE 'codex[- ]review\|codex ACTION_REQUIRED' -- 'tests/' '*_test.go' '*.test.sh'` returned empty. The same grep over `internal/`, `cmd/`, `scripts/` also returned empty. No assertion depends on the literals that were renamed. |
| Template mirror parity (top-level ↔ `templates/base/`) | Holds | `scripts/check-sync.sh` runs as part of `verify.local.sh static` (already exercised in `/verify`) — confirmed PASS in verify report. No template-mirror test exists in `tests/` to re-execute here. |
| Cross-review pipeline parsing of triage report sections | Green | `tests/test-xreview-gate-regression.sh` (21/21) and `tests/test-xreview-prompt-render.sh` (54/54) continue to pass. These suites are the canonical guard for `/cross-review` behavior and would surface any prompt-text accident. |
| Ralph-Loop pipeline orchestration & driver dispatch | Green | `tests/test-ralph-cli-driver.sh` (48/48) — covers `pick_reviewer` inversion, `count_triage_findings`, dry-run paths. Unaffected. |

## Test gaps

1. **Orchestration coverage hole (pre-existing, not caused by this diff).** `scripts/verify.local.sh` only routes 3 of 8 shell test files in `test` mode. The 5 unwired suites (`test-ralph-config.sh`, `test-ralph-signals.sh`, `test-ralph-status.sh`, `test-xreview-gate-regression.sh`, `test-xreview-prompt-render.sh` — 145 test cases total) only run when invoked directly. A future change touching their subjects could pass `./scripts/run-test.sh` while silently regressing 145 cases. This already exists in `tester` memory under `coverage_gaps.md` and is out of scope for this PR.
2. **No dedicated "wording residue / glossary" gate.** The grep that drives AC-5 (allowlist-aware sweep for `codex[- ]review|codex ACTION_REQUIRED`) is documented in the plan but not codified as a repeatable test. If wording drifts back in a future commit, no automated suite will catch it; only manual `/verify` will. Worth considering a tiny `tests/test-glossary-residues.sh` in a follow-up, but not required for this scope.
3. **No template-mirror byte-identity test in `tests/`.** The check lives in `scripts/check-sync.sh` and is only exercised via `verify.local.sh`. This is consistent with how other mirror checks are structured; not a new gap.

## Verdict

- Pass: **YES** — 210/210 shell tests green, all Go packages `ok` (both cached and `-count=1` fresh), exit code 0 from `./scripts/run-test.sh`.
- Fail: none.
- Blocked: none.

**Scope decision matches the plan's Non-goals.** This is a string-only documentation diff with explicit "挙動変更は一切なし". No new tests are required and none were added; the existing 210 shell tests + 9 Go packages remain uniformly green, confirming "既存テストの green 維持確認（ロジック変更なし）" as the test plan specifies. **Proceed to `/sync-docs` and `/cross-review`.**
