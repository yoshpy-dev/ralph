# Self-review report: xreview-base-detection

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-xreview-base-detection.md
- Reviewer: reviewer subagent (Claude Code, /self-review)
- Scope: Diff quality only (`git diff main...HEAD`). No spec-compliance, coverage, or doc-drift evaluation (those belong to /verify and /test). Test 14 was executed once, strictly to confirm the gate-proof assertions genuinely bite — not for coverage assessment.

## Evidence reviewed

Changed files (15; +407/-11):

- `scripts/ralph-cli-driver.sh` (+ `templates/base/` mirror) — new `detect_base_branch` helper
- `scripts/ralph-orchestrator.sh` (+ mirror) — `export RALPH_XREVIEW_BASE="$_base_branch"` at L1297
- `scripts/ralph-pipeline.sh` (+ mirror) — cross-review gate now calls `detect_base_branch`
- `.claude/skills/cross-review/SKILL.md` (+ `.agents/` + 2 `templates/base/` copies) — documented BASE command
- `AGENTS.md` (+ mirror) — driver function list gains `detect_base_branch`
- `docs/tech-debt/README.md` — base-detection row marked RESOLVED
- `tests/test-ralph-cli-driver.sh` — Test 14 (a–e) added

Priority checks performed (per task brief):

- **(a) `detect_base_branch` correctness** — resolution order matches plan Scope 1a/1b/1c: `RALPH_XREVIEW_BASE` (non-empty) → `git symbolic-ref --quiet --short refs/remotes/origin/HEAD` with leading `origin/` stripped → `main` if `refs/heads/main` else `master`. `${_dbb_remote_head#origin/}` is prefix-only strip (correct; `release/1.0` under `origin/` survives — verified by test 14c-edge). POSIX-sh clean: `#!/usr/bin/env sh`, no bashisms, unique `_dbb_`-prefixed locals, guarded `2>/dev/null || true`. The `RALPH_XREVIEW_BASE` path validating nothing is deliberate (plan lines 34-36, Non-goal): an invalid explicit base makes `git diff base...HEAD` fail, which the pipeline gate treats as "has changes" → review runs (fail-open-to-review). This is documented in the helper's own comment block (lines "Note: the pipeline gate treats a failing diff...").
- **(b) orchestrator export placement** — `export RALPH_XREVIEW_BASE="$_base_branch"` at L1297 precedes BOTH `run_slice` (L1526) and `run_integration_pipeline` (L1597). `export` snapshots the *value*, so the subsequent clobbering of the global `_base_branch` by `create_worktree` (L1450, `_base_branch="${INTEGRATION_BRANCH:-...}"` — no `local` in this POSIX-sh script) does NOT corrupt the exported env var. Verified safe. See Positive notes.
- **(c) Test 14 bites** — executed once: 103 PASS / 0 FAIL. 14a is a genuine end-to-end gate proof on a single fixture: asserts the OLD `HEAD@{upstream}` expression yields the feature branch (14a-i) AND an empty `git diff feature...HEAD` (14a-ii), then `detect_base_branch` yields `main` (14a-iii) AND a non-empty `git diff main...HEAD` (14a-iv). Override (14b), non-main default (14c), `/`-containing default (14c-edge), main/master fallback (14d-i/ii), and worktree common-dir resolution (14e) all present and passing.
- **(d) SKILL x4 wording vs helper semantics** — the documented resolution order `(1) $RALPH_XREVIEW_BASE ... (2) symbolic-ref origin/HEAD ... (3) main if refs/heads/main exists, else master` matches the helper exactly. Sourcing `. scripts/ralph-cli-driver.sh` is safe: the file is pure function definitions with zero top-level execution (verified). Pipeline obtains the helper via its existing `. "${SCRIPT_DIR}/ralph-cli-driver.sh"` at L15.
- **(e) mirror byte-identity** — all synced pairs `cmp`-identical AND same git blob hash: cli-driver, pipeline, orchestrator, both SKILL surfaces (root vs template, .claude vs .agents), AGENTS.md. All three scripts carry mode `100755` on both sides. `grep 'HEAD@{upstream}'` over all production sites → NONE (remaining hits are only inside the test file, deliberately proving the OLD behavior).

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | typo/comment-accuracy | Helper comment claims it mirrors "the exact semantics of `default_branch()` in `scripts/ralph` **and** `ralph-worktree.sh`", but those two existing helpers themselves diverge in the final fallback: `scripts/ralph` prints `master` unconditionally, while `ralph-worktree.sh` verifies `refs/heads/master` and `die`s if absent. `detect_base_branch` follows the `scripts/ralph` form (unconditional `master`), not `ralph-worktree.sh`. | `scripts/ralph-cli-driver.sh` comment (Scope 1c note) vs `scripts/ralph` `default_branch()` (unconditional `master`) vs `scripts/ralph-worktree.sh` `default_branch()` (verify + `die`) | Optional: narrow the comment to "mirrors `scripts/ralph`'s `default_branch()`". The unconditional-`master` behavior is itself correct and intentional here — a nonexistent base makes the diff fail → gate fires (fail-open-to-review), which is the desired direction. No code change needed. |

No CRITICAL, HIGH, or MEDIUM findings. No secrets, no debug code, no swallowed errors (all `2>/dev/null` are on read-only git queries with intended fallbacks), no injection surface (helper only reads git refs; no user-string interpolation into commands), no unnecessary/unrelated changes.

## Positive notes

- **Export-by-value robustness**: the `export RALPH_XREVIEW_BASE="$_base_branch"` placement is correct precisely because the global `_base_branch` is later clobbered by `create_worktree` (L1450) in this no-`local` POSIX-sh script. Snapshotting the value at export time (L1297, before any worktree creation) is the right call and sidesteps the clobber. Worth an explicit note because a naive "re-read `_base_branch` at the call site" alternative would have silently used the integration branch as the review base.
- **Gate-proof test design**: 14a asserts both sides of the fix on one fixture (old→empty, new→non-empty), which is exactly the Codex-advisory finding-3 requirement and is the difference between "tests the return value" and "tests the gate".
- **Grep-ability & precedent**: the helper sits next to `pick_reviewer`/`resolve_phase_model`, is announced in the file header and AGENTS.md function list, and reuses the established `default_branch()` shape — consistent with `.claude/rules/architecture.md`.
- **tech-debt hygiene**: the resolved row is struck through (`~~...~~`) with a RESOLVED HTML comment and preserved for traceability, matching the file's existing convention (cli-stub-stdin-hang precedent directly above).

## Tech debt identified

None. (The single LOW finding is a one-line comment nicety, not deferred work; no new tech-debt row warranted.)

## Recommendation

- Merge: Yes (no blocking findings; the one LOW is optional and behavior is correct as written).
- Follow-ups: Optionally tighten the helper comment (LOW above) at next touch. No standalone follow-up task needed.

---

## Cycle 2 addendum

- Date: 2026-07-12
- Scope: diff quality of the two new commits since cycle-1 MERGE.
  - `4de8dc9` docs: sync docs — plan ticks + `RALPH_XREVIEW_BASE` recipe env-table row (root + `templates/base` mirror)
  - `c3d89f2` fix: preserve operator-supplied `RALPH_XREVIEW_BASE` in orchestrator (both copies) + cross-review triage report
- Method: `git show` on both commits, targeted file reads. No tests/static/spec/doc-drift. `< /dev/null` throughout.

### Checks performed

- **(a) param-expansion correctness under `set -u`** — orchestrator declares `set -eu` (L2). The new `RALPH_XREVIEW_BASE="${RALPH_XREVIEW_BASE:-$_base_branch}"` uses the `${VAR:-default}` form, which is exactly the `set -u`-safe idiom: an unset/empty `RALPH_XREVIEW_BASE` is tolerated (no unbound-variable abort) and falls back to `$_base_branch`. The fallback operand `$_base_branch` is unconditionally assigned five lines above at L1292 (`_base_branch="$(git rev-parse --abbrev-ref HEAD)"`) in the same `main()` scope with no early-return in between, so it is guaranteed set. Splitting assignment from `export` (vs. the old `export RALPH_XREVIEW_BASE="..."` one-liner) is behavior-neutral and correct. No word-splitting/globbing risk: the RHS is fully double-quoted.
- **(b) precedence matches detection** — `detect_base_branch()` (`ralph-cli-driver.sh`) checks `[ -n "${RALPH_XREVIEW_BASE:-}" ]` first, so preserving an operator-set value at export time makes the override actually reach the pipelines. The fix closes exactly the clobber the cross-review finding identified. Behavior confirmed consistent with the priority-1 branch of the helper.
- **(c) comment accuracy** — the amended comment ("An operator-supplied `RALPH_XREVIEW_BASE` takes precedence and is preserved as-is") accurately describes the `${:-}` semantics and the downstream `detect_base_branch` priority. No drift between comment and code.
- **(d) recipe row wording vs behavior** — the env-table row states the variable "takes priority over `refs/remotes/origin/HEAD` detection" (matches helper priority 1 > 2) and is "Exported automatically by `ralph-orchestrator.sh` from the launch branch so Loop runs from `develop`/`release/*` diff against the true merge target." Accurate: for a Loop launched from `develop`, the launch branch *is* the correct diff base, and the export sets `RALPH_XREVIEW_BASE=develop`. "Can also be set by the operator for one-off overrides" now matches the preserved-value behavior introduced by `c3d89f2`. No overclaim.
- **(e) plan ticks** — the three referenced report paths (`self-review`, `verify`, `test`) all exist on disk; the sync-docs tick describes the mirror-identical recipe row, which holds. Ticks are accurate.
- **(f) mirror identity** — `scripts/ralph-orchestrator.sh` vs `templates/base/` mirror: `cmp`-identical AND identical git blob hash `3f51770`, mode `100755` both sides. `docs/recipes/ralph-loop.md` vs `templates/base/` mirror: `cmp`-identical AND identical blob `97bccb7`. `check-sync.sh` PASS under its declared `bash` interpreter.
- **triage report** (`cross-review-triage-xreview-base-detection.md`) — new doc-only artifact; content matches the fix applied. No code impact.

### Findings

None. No CRITICAL/HIGH/MEDIUM/LOW. No secrets, debug code, swallowed errors, or unnecessary changes in either commit. The cycle-1 LOW (helper-comment `default_branch()` phrasing) is untouched by these commits and remains optional.

### Recommendation

- Merge: Yes. Both commits are correct, minimal, and mirror-clean. `set -u` safety and precedence semantics verified. Cycle-2 verdict unchanged from cycle 1: MERGE.
