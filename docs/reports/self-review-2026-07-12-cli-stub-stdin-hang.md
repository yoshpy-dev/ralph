# Self-review report: cli-stub-stdin-hang

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-cli-stub-stdin-hang.md
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: `git diff main...HEAD` — 4 changed files (commit c842469): `tests/fixtures/cli-stubs/claude`, `tests/fixtures/cli-stubs/codex`, `tests/test-ralph-cli-driver.sh`, `docs/tech-debt/README.md`, plus the plan artifact. Diff quality only — no spec/test/doc-drift verdict (those belong to /verify and /test).

## Evidence reviewed

- Full diff of both stubs: TTY-only guard (`[ -t 0 ]`) replaced by positive condition `[ -f /dev/stdin ]`; the drain and no-drain branches are swapped so drain now happens in the `if` (regular file) branch and `stdin_buf=""` in the `else`. Header comments rewritten to document both hazard cases (TTY typing wait #44 cycle-3 P2; open-pipe EOF wait, this fix).
- Full diff of `tests/test-ralph-cli-driver.sh`: `< /dev/null` added to the test 6a `codex exec review` direct invocation; new Test 8 (`_stub_elapsed` helper + 8a/8b) using `mkfifo` + a backgrounded `sleep 5 > "$fifo"` writer to reproduce the never-closing-pipe hang shape and assert elapsed `< 3s`.
- Full diff of `docs/tech-debt/README.md`: the stub-stdin row marked RESOLVED.
- Portability probe under `/bin/sh` (macOS): `[ -f /dev/stdin ]` → true for a regular-file redirect, false for `/dev/null` and false for a pipe. Matches the documented policy exactly.
- Regression-detection probe: a synthetic *regressed* stub (old `[ -t 0 ]` guard) fed by the same FIFO blocks ~5s, which would fail the `< 3s` assertion — confirming Test 8 has real detection value, not a tautological pass.
- Full suite run with `< /dev/null` (per the pre-fix hang shape): `bash tests/test-ralph-cli-driver.sh` → 53 PASS / 0 FAIL in <1s; Test 8 reports `elapsed=0s` for both stubs. No orphan `sleep 5` writers left behind (`pgrep` clean).
- `shellcheck -s sh` on both stubs and `shellcheck -s bash` on the test file: all clean, no warnings.
- Call-log correctness: both stubs write the call log *after* the stdin branch unconditionally (codex lines 61-76, claude equivalent), so the "still wrote call log despite no-drain" assertions (8a-ii, 8a-iii, 8b-ii) are structurally sound.
- Tech-debt RESOLVED convention cross-checked against the two prior resolved rows (README.md lines 24, 31-32): HTML comment + strike-through + `(RESOLVED <date> in <branch>)` — new row is consistent.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `kill "$writer_pid"` targets the subshell PID of `( sleep 5 > "$fifo" )`, not necessarily the `sleep` process directly. On most shells killing the subshell reaps the child, and `sleep 5` is self-terminating within the 5s budget regardless, so no orphan can outlive the test. The probe confirmed zero orphan writers after a full run. Purely a defense-robustness note — not a leak in practice. | `tests/test-ralph-cli-driver.sh:354-355,371-372`; `pgrep -fl 'sleep 5'` → none after suite | Optional: no change needed. If ever tightened, `sleep 5 & wpid=$!; exec >"$fifo"` style or a trap-based reaper would kill the leaf directly. Not worth the churn for a bounded 5s helper. |
| LOW | readability | `date +%s` gives whole-second granularity, so `elapsed` is a coarse integer (0/1/2…). The `< 3s` threshold against a 5s writer leaves a ≥2s margin and the assertion direction (must be FAST) cannot false-pass when the bug is present — a regressed stub always reads ≥5s. Sound, but the second-resolution timer is worth a one-line note for a future maintainer tempted to tighten the threshold. | `tests/test-ralph-cli-driver.sh:357,370,374`; observed `elapsed=0s` | Optional: leave as-is; the plan's risk register already documents the 2s margin. Do not lower the threshold below 3 without switching to sub-second timing. |

No CRITICAL, HIGH, or MEDIUM findings.

## Positive notes

- **Correct swap of branch logic.** The condition inversion (negative `skip iff TTY` → positive `drain iff regular file`) is implemented cleanly: unlisted descriptor types now default to the safe no-drain branch, which is exactly the plan's design decision #1. Verified by probe that pipe/`/dev/null`/absent-`/dev/stdin` all fall through to `stdin_buf=""`.
- **Regression test has genuine detection value.** Confirmed empirically that a reverted stub blocks ~5s and fails the `< 3s` assertion — this is not a test that passes regardless of the fix.
- **No orphan processes / clean teardown.** `_stub_elapsed` reaps its writer (`kill` + `wait`) and removes its own mktemp dir; the suite-level `trap 'rm -rf "$WORK_DIR"' EXIT` covers the rest. `pgrep` confirmed no leaked writers.
- **Comment accuracy.** Both stub headers document the two distinct hazard cases with correct attribution (#44 cycle-3 for TTY, this fix for open-pipe) and correctly state the `/dev/stdin`-absent fallback is the safe branch. The inline comment at the condition matches the header.
- **shellcheck-clean** across all three shell files, both `sh` and `bash` dialects.
- **Defense in depth as planned.** Root-cause stub fix + call-site `< /dev/null` (6a) + elapsed-time regression test — the three-layer approach the plan committed to.
- **Tech-debt row** follows the established RESOLVED convention byte-for-byte in structure.
- **Non-goal discipline.** No production-script (`ralph-pipeline.sh`) or `templates/base` changes — matches the plan's non-goals; no scope creep in the diff.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_No new tech debt. The diff itself closes an existing tech-debt row (README.md line 40) and adds no deferred work. The two LOW findings above are optional-polish notes, not deferred obligations._

## Recommendation

- Merge: Yes (no blocking findings). The two LOW findings are optional and need not block. Diff is focused, well-commented, shellcheck-clean, and the regression test provably catches the bug it targets.
- Follow-ups: None required. If a future maintainer tightens the Test 8 threshold, switch to sub-second timing rather than lowering below `3` (finding #2).
