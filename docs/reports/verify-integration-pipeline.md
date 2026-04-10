# Verify Report — Integration Pipeline

**Date:** 2026-04-10
**Branch:** feat/ralph-loop-v2
**Verifier:** verifier subagent
**Plan:** Integration pipeline feature (inline ACs from user prompt)

---

## Verdict: PASS

All 7 acceptance criteria met. Both target scripts pass `bash -n` syntax check. Documentation is consistent with implementation.

---

## Acceptance Criteria Tracking

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| AC1 | `ralph-pipeline.sh` has `--skip-pr` flag that skips PR creation in Outer Loop | PASS | Lines 21, 38, 53, 735–739, 813 |
| AC2a | `ralph-pipeline.sh` has `--fix-all` flag: overrides COMPLETE when ANY self-review findings exist | PASS | Lines 22, 39, 54, 532–539, 814 |
| AC2b | `ralph-pipeline.sh` has `--fix-all` flag: treats WORTH_CONSIDERING as ACTION_REQUIRED | PASS | Lines 724–728 |
| AC3 | `ralph-orchestrator.sh` has `run_integration_pipeline()` that sets up state and runs `ralph-pipeline.sh --skip-pr --fix-all` | PASS | Lines 528–596 |
| AC4 | `ralph-orchestrator.sh` main() calls integration pipeline after successful merge, only creates PR if pipeline passes | PASS | Lines 786–815 |
| AC5 | `create_unified_pr()` PR body includes integration pipeline checkmarks | PASS | Lines 503–509 |
| AC6 | `.claude/rules/subagent-policy.md` updated with integration pipeline description | PASS | Lines 51–53 |
| AC7 | `.claude/skills/loop/SKILL.md` updated with integration pipeline step | PASS | Lines 93, 97 |

---

## Spec Compliance Detail

### AC1 — `--skip-pr` flag

```
scripts/ralph-pipeline.sh:
  Line 21:  SKIP_PR=0
  Line 38:  --skip-pr  (in usage())
  Line 53:  --skip-pr) SKIP_PR=1 ;;
  Line 735: if [ "$SKIP_PR" -eq 1 ]; then
  Line 737:   ckpt_update '.status = "complete"'
  Line 739:   return 0
  Line 813: log "Skip PR: ${SKIP_PR}"
```

Flag is declared, documented in usage(), parsed from CLI args, and correctly skips PR creation in `run_outer_loop()` with `status = "complete"` to allow calling scripts to detect success.

### AC2 — `--fix-all` flag

**Part (a) — override COMPLETE on any self-review findings:**
```
scripts/ralph-pipeline.sh lines 532–539:
  if [ "$FIX_ALL" -eq 1 ]; then
    _sr_total=$((_sr_critical + _sr_high + _sr_medium + _sr_low))
    if [ "$_sr_total" -gt 0 ]; then
      log "fix-all: ${_sr_total} self-review finding(s) — overriding COMPLETE"
      _agent_complete=0
    fi
  fi
```
Counts all four severity levels (critical + high + medium + low). Any nonzero total clears `_agent_complete`.

**Part (b) — WORTH_CONSIDERING treated as ACTION_REQUIRED:**
```
scripts/ralph-pipeline.sh lines 724–728:
  if [ "$FIX_ALL" -eq 1 ] && [ "$_worth_considering" -gt 0 ]; then
    log "fix-all: ${_worth_considering} WORTH_CONSIDERING finding(s) — regressing to Inner Loop"
    return 1
  fi
```
Returns 1 (regress to Inner Loop) — the same code path as `ACTION_REQUIRED`.

### AC3 — `run_integration_pipeline()` in ralph-orchestrator.sh

Function at lines 528–596 of `scripts/ralph-orchestrator.sh`:

1. **State setup** (lines 537–569): Creates `.harness/state/loop/PROMPT.md`, `task.json`, and `progress.log`. The PROMPT.md includes integration-specific objective (review cross-module issues, fix ALL findings, signal COMPLETE only when clean).
2. **Pipeline prompt adaptation** (lines 572–578): Copies `pipeline-*.md` prompts from `.claude/skills/loop/prompts/` to `.harness/state/pipeline/`, with sed substitution to use `git diff <base>...HEAD` for integration context.
3. **Pipeline invocation** (lines 581–585): Calls `ralph-pipeline.sh --skip-pr --fix-all --max-iterations 10 --max-inner-cycles 5 --max-outer-cycles 2`. Both flags confirmed present.
4. **Terminal status check** (lines 587–594): Reads checkpoint.json status. Returns 0 only if `complete`, returns 1 otherwise.

### AC4 — main() orchestration flow

```
scripts/ralph-orchestrator.sh lines 786–815:
  if [ "$_completed" -gt 0 ] && [ "$_failed" -eq 0 ]; then
    if integration_merge ...; then
      # Run integration pipeline on the merged branch
      if run_integration_pipeline ...; then
        _merge_status="pipeline_passed"
        if [ "$UNIFIED_PR" -eq 1 ]; then
          _pr_url="$(create_unified_pr ...)"
        fi
      else
        _merge_status="pipeline_failed"
        log_error "Integration pipeline failed. PR not created."
      fi
    fi
  fi
```

Logic is correct:
- Integration pipeline runs only after successful merge (all slices complete + merge conflict-free)
- PR is created only when `run_integration_pipeline` returns 0
- On pipeline failure: `_merge_status="pipeline_failed"`, PR not created

### AC5 — PR body integration pipeline checkmarks

`create_unified_pr()` in `scripts/ralph-orchestrator.sh` lines 503–509:

```
## Test plan

- [x] All slice pipelines passed (self-review, verify, test)
- [x] Integration pipeline passed (self-review, verify, test, sync-docs, codex-review)
- [x] All self-review findings fixed (--fix-all)
- [x] Integration merge passed without conflicts
- [ ] CI checks pass on this PR
```

Three integration-specific checkmarks present:
- Integration pipeline passed (confirmed)
- All self-review findings fixed via --fix-all (confirmed)
- Integration merge passed without conflicts (confirmed)

### AC6 — subagent-policy.md update

`.claude/rules/subagent-policy.md` lines 51–53:

```
After all slices are merged into the integration branch, `ralph-orchestrator.sh` runs
`ralph-pipeline.sh --skip-pr --fix-all` on the integration branch as a unified quality gate.
This catches cross-module issues and fixes ALL findings (including MEDIUM/LOW and
WORTH_CONSIDERING) before unified PR creation.
```

This accurately describes the implementation. Flags, scope (MEDIUM/LOW + WORTH_CONSIDERING), and purpose are all present.

### AC7 — SKILL.md update

`.claude/skills/loop/SKILL.md` lines 93–98 ("After the loop" section):

```
The orchestrator handles everything autonomously (parallel pipeline per slice →
integration merge → **integration pipeline** (`--skip-pr --fix-all`) → unified PR).
The integration pipeline runs `ralph-pipeline.sh` on the merged branch to catch
cross-module issues and fix ALL findings before PR creation.
```

Integration pipeline is named, flags are documented, purpose is explained. No additional table row needed (SKILL.md describes it inline in the After-the-loop narrative).

---

## Static Analysis

```
bash -n scripts/ralph-pipeline.sh   → PASS
bash -n scripts/ralph-orchestrator.sh → PASS
bash -n scripts/*.sh (all 16 scripts) → PASS (no failures)
```

INFO: `shellcheck` is not installed in this environment. Patterns that `bash -n` cannot detect (e.g., unquoted variable expansion, array misuse, word-split bugs) remain unverified by static analysis.

---

## Documentation Drift Check

| Location | Expected | Actual | Status |
|----------|----------|--------|--------|
| `subagent-policy.md` — /loop section | integration pipeline description with --skip-pr --fix-all | Present (lines 51–53) | No drift |
| `SKILL.md` — After the loop | integration pipeline mentioned with flags | Present (lines 93, 97) | No drift |
| `ralph-pipeline.sh` usage() | --skip-pr and --fix-all documented | Lines 38–40 | No drift |
| `ralph-orchestrator.sh` usage() | No integration pipeline flags (internal) | N/A — function is internal | No drift |

No documentation drift detected.

---

## Known Gaps (not blocking)

1. **shellcheck not available**: `bash -n` only validates syntax. Semantic shell bugs (unquoted variables in word-split contexts, SC2086 class issues) are not caught. This is a recurring environment gap.

2. **run_integration_pipeline does not reset checkpoint.json before running pipeline**: If a previous pipeline run left stale checkpoint state in `.harness/state/pipeline/checkpoint.json`, the integration pipeline may resume from an unexpected state. The `--resume` flag is not passed, but stale checkpoint files are not explicitly cleaned before the call. This is LOW severity — the pipeline overwrites checkpoint.json on fresh start (lines 838–863 of ralph-pipeline.sh), so the risk is minimal.

3. **AC2b coverage gap**: `--fix-all` WORTH_CONSIDERING logic (line 725) is placed after the `ACTION_REQUIRED` check (line 719). If `_action_required > 0` AND `_worth_considering > 0`, the function already returns 1 from the ACTION_REQUIRED path. The --fix-all path for WORTH_CONSIDERING is only exercised when `_action_required == 0`. This is the correct semantics and not a bug, but worth noting.

4. **Integration pipeline uses main repo state, not worktree**: `run_integration_pipeline()` runs in the main repo directory (after `git checkout "$INTEGRATION_BRANCH"`), not in a separate worktree. This means if the main repo has uncommitted changes, they could interfere. The function does not call `check_uncommitted()` first.

---

## Unverified

- Runtime behavior of `--skip-pr` and `--fix-all` in an actual pipeline execution (test agent's responsibility)
- Whether the pipeline prompt adaptation (sed substitution in `run_integration_pipeline()` line 576) produces a valid prompt that the agent correctly interprets
- Whether `create_unified_pr()` succeeds when called from a non-worktree context (requires live `gh` CLI)

---

## What Would Increase Confidence Most

The single highest-value additional check would be a **dry-run integration test**:

```sh
./scripts/ralph run \
  --plan docs/plans/active/<any-existing-plan>/ \
  --unified-pr \
  --dry-run
```

This would verify that:
1. `run_integration_pipeline()` is reached in the dry-run flow
2. The `--skip-pr --fix-all` flags are echoed in the dry-run log
3. The PR body generation does not fail with shell errors

This is a tester-scope item, not verifier-scope.
