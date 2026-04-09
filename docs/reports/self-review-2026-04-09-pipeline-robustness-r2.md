# Self-review report: Pipeline robustness improvements (r2 re-run)

- Date: 2026-04-09
- Plan: docs/plans/active/2026-04-09-pipeline-robustness.md
- Reviewer: reviewer subagent (claude-sonnet-4-6)
- Scope: Diff quality only — focus on changes introduced in commit `6e49d6a` (Codex WORTH_CONSIDERING fix-pass). Scope: COMPLETE gating (return 6), pipeline-outer.md scope restriction, dry-run COMPLETE simulation, jq --arg for safe JSON updates, phase transition ordering fix.
- Prior report: docs/reports/self-review-2026-04-09-pipeline-robustness.md (r1)

## Evidence reviewed

- `git diff main...HEAD` with emphasis on commit `6e49d6a` (`git show 6e49d6a`)
- `scripts/ralph-pipeline.sh` (current, 892 lines)
- `.claude/skills/loop/prompts/pipeline-outer.md` (current)
- `docs/tech-debt/README.md`
- `.claude/agent-memory/reviewer/MEMORY.md`

---

## Summary of changes reviewed

The r2 commit (`6e49d6a`) addresses Codex WORTH_CONSIDERING findings:

1. **COMPLETE gating**: `run_inner_loop()` now returns 6 when tests pass but COMPLETE is not signalled. The main loop adds a `case 6` handler that increments `_inner_cycle` and continues. Previously, tests passing was sufficient to advance to Outer Loop.
2. **pipeline-outer.md scope restriction**: Removed codex review and PR creation instructions. The prompt now covers docs sync only.
3. **dry-run COMPLETE simulation**: After `run_claude()` in dry-run mode, writes `COMPLETE` to `.agent-signal` so the pipeline can proceed through all phases in dry-run.
4. **jq --arg for safe JSON updates**: `_new_session` (line 428) and `_pr_url` (line 705) both use `ckpt_update --arg <name> "$val"` pattern — confirmed fixed.
5. **Phase transition ordering fix**: `_prev_phase` is read before `ckpt_update` at lines 349-351 — confirmed fixed.

---

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| HIGH | security | `report_event "pr-created"` at line 706 still embeds `_pr_url` via string concatenation into a raw JSON string. The `ckpt_update` caller at line 705 was fixed to use `--arg`, but the adjacent `report_event` call was not. A URL containing `"` or `\` will corrupt the JSONL event log. | `scripts/ralph-pipeline.sh:706`: `report_event "pr-created" "{\"cycle\":${_cycle},\"url\":\"${_pr_url}\"}"` — this was reported as HIGH in the r1 report and marked for resolution, but was not included in the r2 fix-pass. | Build the JSON string safely before passing: `_ev="$(jq -n --argjson c "$_cycle" --arg u "$_pr_url" '{"cycle":$c,"url":$u}')" && report_event "pr-created" "$_ev"`. |
| MEDIUM | readability | The dry-run COMPLETE simulation comment says "after first cycle" but the code fires on every dry-run cycle. The `.agent-signal` file is cleared at the top of each cycle (line 354) before `run_claude` is called, and then written again by the simulation block (line 417). This means every dry-run cycle will detect COMPLETE — which is the correct behaviour for test compatibility — but the comment is misleading. | `ralph-pipeline.sh:415`: `# In dry-run mode, simulate COMPLETE signal after first cycle` vs actual behaviour: fires on every cycle. | Change comment to: `# In dry-run mode, always simulate COMPLETE signal so the pipeline advances to Outer Loop`. |
| MEDIUM | maintainability | `pipeline-outer.md` was correctly restricted to docs-sync only, but the inline fallback docs prompt inside `ralph-pipeline.sh:595-600` (used when neither `${PIPELINE_DIR}/pipeline-outer.md` nor the skill file is found) was also updated in this commit. The fallback is now consistent with the prompt. However, the PR prompt at lines 659-671 contains a hardcoded example URL `echo "https://github.com/..." > .harness/state/pipeline/.pr-url`. This was flagged as MEDIUM in r1 and remains unfixed. An agent may copy the example literally. | `ralph-pipeline.sh:670`: `echo "https://github.com/..." > .harness/state/pipeline/.pr-url` | Replace the example with a command that writes the actual PR URL: `gh pr view --json url --jq '.url' > .harness/state/pipeline/.pr-url`. This was in r1 MEDIUM follow-ups and is still present. |
| LOW | readability | The fallback inline review prompt (lines 482-492) now correctly writes to `.harness/state/pipeline/` per the r1 MEDIUM finding. However the fallback omits the verification fallback note. `pipeline-review.md:30` shows `(or HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh)` as an option. The inline fallback at line 490 only says `Write findings to .harness/state/pipeline/ following the self-review template.` — the static verify fallback command is not mentioned. This is a minor discrepancy if `run-static-verify.sh` is absent and the fallback prompt is used. | `ralph-pipeline.sh:482-492` inline REVIEW heredoc vs `pipeline-review.md:30`. | Low impact. Add `(or HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh if run-static-verify.sh is not available)` to the inline fallback. |

---

## Confirmation of r1 HIGH fixes

| r1 Finding | Status | Evidence |
|------------|--------|----------|
| `_new_session` embedded in jq filter string without `--arg` (line 422) | **RESOLVED** | `ralph-pipeline.sh:428`: `ckpt_update --arg sid "$_new_session" '.session_id = $sid'` |
| `_pr_url` embedded in `ckpt_update` filter string without `--arg` (line 699) | **RESOLVED** | `ralph-pipeline.sh:705`: `ckpt_update --arg url "$_pr_url" '.pr_created = true \| .pr_url = $url \| .status = "complete"'` |
| `_pr_url` embedded in `report_event` JSON string (line 700) | **NOT RESOLVED** | `ralph-pipeline.sh:706`: still uses string interpolation `\"${_pr_url}\"` |
| Phase `ckpt_update` before `ckpt_read` for transition record (line 349-350) | **RESOLVED** | `ralph-pipeline.sh:349-351`: `_prev_phase` read before `ckpt_update`, passed to `ckpt_transition` |

## Confirmation of Codex WORTH_CONSIDERING fixes

| Codex Finding | Status | Evidence |
|---------------|--------|----------|
| COMPLETE gating: tests pass without COMPLETE advances to Outer Loop | **RESOLVED** | `ralph-pipeline.sh:572`: `return 6` added; `ralph-pipeline.sh:800-804`: `case 6` handler continues inner loop |
| `pipeline-outer.md` scope duplicates script-level codex/PR phases | **RESOLVED** | `pipeline-outer.md` now contains docs-sync only with explicit `Do NOT create pull requests or run codex review` instruction |

---

## Positive notes

- The `ckpt_update --arg` pattern is now consistently applied to the two most security-sensitive fields (`_new_session`, `_pr_url` in ckpt_update). This addresses the primary injection vector identified in r1.
- Phase transition ordering (`_prev_phase` read before `ckpt_update`) is correctly implemented. This recurring pattern from the reviewer memory is now fixed.
- The COMPLETE gating logic (return 6) is clean and the case statement in the main loop handles it without ambiguity. The comment in `case 0` was also updated to clarify the exit condition.
- `pipeline-outer.md` scope restriction is well-executed. The prompt now explicitly states its scope and repeats the safety rule at the bottom.
- Dry-run COMPLETE simulation correctly uses the sidecar file mechanism (same path the real agent uses), making the dry-run path realistic.

---

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `report_event "pr-created"` at line 706 still embeds `_pr_url` via string concatenation. The adjacent `ckpt_update` at line 705 was fixed, but the JSONL event writer was not (HIGH: security). | Malformed JSONL if PR URL contains `"` or `\`. GitHub URLs are typically safe, but structurally inconsistent with the fix applied to `ckpt_update`. | r2 fix-pass targeted `ckpt_update` callers only. `report_event` uses a different interface (raw JSON string). | Next time `report_event` is refactored to accept structured args, or when PR URL sources are broadened beyond gh CLI output. | docs/reports/self-review-2026-04-09-pipeline-robustness-r2.md |

_(Above entry to be appended to `docs/tech-debt/README.md`.)_

---

## Recommendation

- **Merge: CONDITIONAL** — one HIGH finding remains from r1 that was not resolved in r2.
- The unfixed HIGH is `report_event "pr-created"` at line 706 (string interpolation of `_pr_url` in a raw JSON string). The practical risk is low since GitHub PR URLs do not contain `"` or `\`, but it is structurally inconsistent with the `ckpt_update` fix directly above it on line 705.
- If the team accepts the residual risk (GitHub URLs are well-formed), this can be recorded in tech-debt and the branch can merge. If not, apply the `jq -n --arg` pattern to `report_event` at line 706 before merging.
- MEDIUM and LOW findings are non-blocking: the hardcoded example URL in the PR prompt (line 670) and the misleading dry-run comment (line 415) are safe to defer.
- Follow-ups:
  - Fix `report_event "pr-created"` at line 706 to use `jq -n --argjson c "$_cycle" --arg u "$_pr_url"` pattern, or accept and record in tech-debt.
  - Fix dry-run COMPLETE comment at line 415 from "after first cycle" to "on every cycle".
  - Fix hardcoded example URL at line 670 to use `gh pr view --json url --jq '.url'`.
  - Update `docs/tech-debt/README.md` with the `report_event` injection debt entry.
