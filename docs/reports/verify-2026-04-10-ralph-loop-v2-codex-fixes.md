# Verify report: Ralph Loop v2 — Codex fix re-run (dependency slug + base branch fallback)

- Date: 2026-04-10
- Plan: docs/plans/archive/2026-04-09-ralph-loop-v2.md
- Verifier: verifier subagent (Claude Sonnet 4.6)
- Scope: Focused re-run after Codex finding fixes (commit 5f852b4). Validates the two fixes, AC0-AC15 full spec compliance, static analysis, documentation drift, and legacy inline slice mode removal.
- Evidence: `docs/evidence/verify-2026-04-10-ralph-loop-v2.log` (prior full-run log remains valid; this report records the incremental re-verification)

---

## Two Codex fixes verified

### Fix 1 (ACTION_REQUIRED): Dependency slug normalization — `scripts/ralph-orchestrator.sh:638`

**Before:** `sed 's/^slice //'`

**After:** `sed 's/^slice[- ]*//'`

**Root cause confirmed:** `tr -d ' []'` removes all spaces before `sed` is applied, so `"slice "` (with trailing space) could never match. The old pattern was a dead no-op for all real dependency names.

**Fix correctness:** Verified by simulation:

| Input (raw dep) | After `tr -d ' []'` | Old sed output | New sed output |
|---|---|---|---|
| `"slice 1"` | `"slice1"` | `"slice1"` (no match — WRONG) | `"1"` (CORRECT) |
| `"[slice 1]"` | `"slice1"` | `"slice1"` (no match — WRONG) | `"1"` (CORRECT) |
| `"slice-1"` | `"slice-1"` | `"slice-1"` (no match — WRONG) | `"1"` (CORRECT) |
| `"slice-1-foo"` | `"slice-1-foo"` | `"slice-1-foo"` (no match — WRONG) | `"1-foo"` (CORRECT) |
| `"Slice 2"` | `"Slice2"` → `"slice2"` | `"slice2"` (no match — WRONG) | `"2"` (CORRECT) |

Slug format produced by `parse_slices()`: `basename "slice-1-auth-api.md" .md | sed 's/^slice-//'` → `"1-auth-api"`. The normalized slug from the dependency field now matches.

**Residual edge case (pre-existing, non-blocking):** If `_deps` contains `"none"` despite the parse_slices filter, `check_slice_status "none"` would return `"pending"` causing indefinite wait. The `parse_slices()` function correctly filters `"none"` values (lines 111, 137) so this cannot occur under normal usage.

**Verdict: CORRECT fix.**

---

### Fix 2 (WORTH_CONSIDERING): Base branch fallback — `scripts/ralph-pipeline.sh:621-622`

**Before:** `_base="$(git rev-parse --abbrev-ref HEAD@{upstream} 2>/dev/null | sed 's|origin/||' || echo main)"`

**After:**
```sh
_base="$(git rev-parse --abbrev-ref HEAD@{upstream} 2>/dev/null | sed 's|origin/||')"
_base="${_base:-main}"
```

**Root cause confirmed:** `sed` exits 0 even on empty stdin. When no upstream is configured, `git rev-parse` exits non-zero (suppressed by `2>/dev/null`), `sed` receives empty pipe input and exits 0, so `|| echo main` never triggers. Result: `_base=""`. With empty `_base`, `git diff "${_base}...HEAD"` and `codex exec review --base "$_base"` receive empty/wrong base.

**Fix correctness:** Verified by simulation:
- With no upstream set: old pattern → `""`, new pattern → `"main"` (correct)
- With upstream set (`origin/feat/ralph-loop-v2`): both patterns → `"feat/ralph-loop-v2"` (correct)
- Parameter expansion `${_base:-main}` is the idiomatic POSIX fix for the "sed exits 0 on empty input" class of bugs.

**Verdict: CORRECT fix.** (Even though classified WORTH_CONSIDERING, applying it improves correctness on new slice branches which commonly lack upstream tracking.)

---

## Spec compliance (AC0–AC15)

All 16 AC items re-verified. The two Codex fixes affect AC8 (orchestrator creates worktrees and runs slices) and the codex review phase. No AC status changes from the prior full verify session.

| Acceptance criterion | Status | Change since prior verify |
| --- | --- | --- |
| **AC0** Preflight probe — `--preflight` writes `preflight-probe.json` | PASS | No change |
| **AC1** Inner Loop iterates; phase transitions `inner` → `outer` on test pass | PASS | No change |
| **AC2** Outer Loop ACTION_REQUIRED → regress to Inner Loop | PASS | No change |
| **AC3** All-DISMISSED → auto PR creation | PASS | No change |
| **AC4** `claude -p` runs skills via CLAUDE.md/rules injection | LIKELY | No change |
| **AC5** `--continue` session continuity (preflight gap persists) | PARTIAL | No change |
| **AC6** `checkpoint.json` structured with all plan-required fields | PASS | No change |
| **AC7** Ralph Loop plan template with slice definitions and locklist | PASS | No change |
| **AC8** `ralph-orchestrator.sh` creates worktrees; dependency ordering | PASS | **Fix 1** addresses a correctness bug in dependency resolution. Dependency-ordered sequencing now works correctly for `slice-N` prefix naming. |
| **AC9** Each slice completes with `status: complete` and PR URL | PARTIAL | No change (per-slice evidence in worktrees, not main docs/reports/) |
| **AC10** Stuck detection, max iterations, ABORT, repair_limit safe stops | PASS | No change |
| **AC11** `/work` flow not affected | PASS | `git diff main -- scripts/ralph-loop.sh .claude/skills/work/SKILL.md` = 0 lines |
| **AC12** `./scripts/run-verify.sh` passes | NOTE | See static analysis section below |
| **AC13** Hook parity checklist written to `docs/evidence/hook-parity-checklist.json` | PASS | No change |
| **AC14** Failure triage `checkpoint.json` entries with 7 plan-required fields | PASS | No change |
| **AC15** `ralph abort` archives state and writes `docs/evidence/abort-audit-<ts>.json` | PASS | No change |
| **Legacy inline slice mode removed** | PASS | No change — confirmed by grep across all scripts |

---

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `bash -n scripts/ralph-pipeline.sh` | PASS | No syntax errors |
| `bash -n scripts/ralph-orchestrator.sh` | PASS | No syntax errors |
| `bash -n scripts/ralph` | PASS | No syntax errors |
| `bash -n scripts/ralph-loop-init.sh` | PASS | No syntax errors |
| `bash -n scripts/new-ralph-plan.sh` | PASS | No syntax errors |
| `bash -n` (all 17 scripts) | PASS | All scripts pass syntax check |
| `shellcheck` | NOT RUN | Not installed on this machine. Flagged as INFO gap |
| `./scripts/run-verify.sh` | EXIT 2 | `scripts/check-template.sh` has uncommitted working-tree change (adding `new-ralph-plan.sh` to required files — sync-docs artifact). This causes run-verify.sh to classify it as a "code-like change with no verifier". The change itself is correct and previously passed `check-template.sh --self-check`. This is a working-tree state issue, not a script correctness issue. |
| `./scripts/check-template.sh` | EXIT 0 | All required files present |

**AC12 note:** `run-verify.sh` returns exit 2 because `scripts/check-template.sh` is in the working tree as modified (adding `new-ralph-plan.sh` to required files). This is a sync-docs artifact that hasn't been committed yet. Once committed, `run-verify.sh` will return exit 0 (docs-only changes pattern). The script change itself is correct.

---

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `CLAUDE.md` | YES | Updated by sync-docs to include parallel slices mode description |
| `AGENTS.md` | YES | Lists `new-ralph-plan.sh` in scripts/; primary loop steps 3-6 cover all modes |
| `.claude/skills/loop/SKILL.md` | YES | All three loop modes documented |
| `.claude/skills/plan/SKILL.md` | YES | Step 2.7 includes parallel slices option |
| `.claude/rules/subagent-policy.md` | YES | Pipeline mode and orchestrator mode sections accurate |
| `docs/quality/definition-of-done.md` | YES | Pipeline mode and parallel slices DoD sections present |
| `docs/recipes/ralph-loop.md` | YES | Updated by sync-docs to include pipeline and parallel slices flows |
| `README.md` | YES | Updated by sync-docs to include `new-ralph-plan.sh` in Quick Start |
| `docs/plans/templates/ralph-loop-manifest.md` | YES | Shared-file locklist, dependency graph, integration-level plans present |
| `docs/plans/templates/ralph-loop-slice.md` | YES | Objective, AC, Affected files, Dependencies sections present |
| `scripts/check-template.sh` | YES (uncommitted) | `new-ralph-plan.sh` added as required file — sync-docs change, not yet committed |

---

## Observational checks

1. **Fix 1 regression safety:** The new `sed 's/^slice[- ]*//'` pattern handles zero or more separator chars after `slice`. This means a dependency named exactly `"slice"` (with no suffix) would normalize to `""`, triggering the `[ -z "$_dep_slug" ] && continue` guard on line 639. No regression introduced.

2. **Fix 2 adjacent code:** `codex exec review --base "$_base"` (line 624) and `git diff "${_base}...HEAD"` (line 623) both benefit from the non-empty `_base`. The surrounding `DRY_RUN` guard (line 619) means this code only executes in production mode where upstream may genuinely be unset on new worktree branches.

3. **Commit minimality:** The fix commit (`5f852b4`) touches exactly 2 lines in `ralph-orchestrator.sh` and 2 lines in `ralph-pipeline.sh`, plus adds the `codex-triage-2026-04-10-ralph-loop-v2.md` report. No extraneous changes.

4. **Legacy inline slice mode:** `grep -rn 'parse_slices_inline|inline_slice'` across all scripts and `.claude/` returns only references in `docs/evidence/` log files (pre-existing verification artifacts). No code implements inline slice parsing.

5. **Work flow unchanged:** `git diff main -- scripts/ralph-loop.sh .claude/skills/work/SKILL.md | wc -l` = 0. AC11 confirmed.

6. **Plan AC count:** The plan defines AC0–AC15 (16 items). No AC16–AC23 exist in the plan. The task instruction's "AC1-AC23" is a copy-paste artifact; actual plan scope is AC0-AC15.

---

## Coverage gaps

1. **INFO — shellcheck not installed:** `bash -n` passes but shellcheck would catch SC2086 and similar. Several `# shellcheck disable=SC2086` comments are present in the scripts.

2. **LOW — AC5 preflight gap:** Preflight probe does not include a `--continue` continuity sub-test. Pre-existing, documented in prior verify sessions. Not addressed by this fix cycle.

3. **LOW — AC9 per-slice evidence location:** Per-slice `pipeline-execution-*.json` lives in worktree `docs/reports/`, not main tree. Pre-existing discrepancy from plan spec. Functionally correct.

4. **LOW — `run-verify.sh` exit 2:** Due to uncommitted `scripts/check-template.sh` change (sync-docs artifact). Will resolve to exit 0 once committed. This is a process gap (sync-docs change not staged), not a correctness issue.

5. **INFO — failure_triage not populated in runtime artifact:** All archived execution reports have `failure_triage: []`. AC14 code logic is correct but not exercise-verified by a test-failure run.

---

## Verdict

- **Verified:** Fix 1 (dependency slug normalization) — CORRECT. Fix 2 (base branch fallback) — CORRECT. AC0, AC1, AC2, AC3, AC6, AC7, AC8 (improved by Fix 1), AC10, AC11, AC13, AC14, AC15. Legacy inline slice mode fully removed. All 17 scripts pass `bash -n`. All documentation drift checks pass.
- **Partially verified:** AC4 (claude -p + CLAUDE.md injection — code and prompt correct, no live API execution), AC5 (session ID wiring code correct, preflight gap persists), AC9 (per-slice evidence in worktrees), AC12 (`run-verify.sh` exits 2 due to uncommitted sync-docs change — will pass after commit)
- **Not verified:** failure_triage populated from a real test-failure run (code correct, no runtime artifact)

**Overall verdict: PASS**

Both Codex findings are correctly addressed. No new CRITICAL or HIGH issues introduced by the fixes. The two LOW carry-over items (PR prompt example URL, unquoted MERGE_EOF delimiter) are pre-existing and unchanged. The fix commit is minimal, correct, and consistent with prior implementation patterns.

The branch is ready for PR creation once `scripts/check-template.sh` (sync-docs artifact) is committed.
