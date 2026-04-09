# Self-review report: Ralph Loop v2 — final review before merge

- Date: 2026-04-10
- Plan: docs/plans/archive/2026-04-09-ralph-loop-v2.md
- Reviewer: reviewer subagent (claude-sonnet-4-6)
- Scope: Diff quality only (naming, readability, null safety, debug code, secrets, exception handling, security, maintainability). Does not evaluate spec compliance, test coverage, or documentation drift.

## Evidence reviewed

- `git diff main...feat/ralph-loop-v2 --stat` — 63 files changed, 6566 insertions (+65 deletions)
- Scripts reviewed in full: `scripts/ralph`, `scripts/ralph-pipeline.sh`, `scripts/ralph-orchestrator.sh`, `scripts/ralph-loop-init.sh`, `scripts/new-ralph-plan.sh`, `scripts/archive-plan.sh`
- Prompt templates: `.claude/skills/loop/prompts/pipeline-inner.md`, `pipeline-outer.md`, `pipeline-review.md`
- Skill files: `.claude/skills/loop/SKILL.md`, `.claude/skills/plan/SKILL.md`
- Previous review: `docs/reports/self-review-2026-04-09-ralph-loop-v2-r2.md`
- Tech-debt register: `docs/tech-debt/README.md`
- MEMORY.md checked for recurring patterns

---

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | `detect_shared_files()` in `scripts/ralph-orchestrator.sh` declares `_all_files=""` but never uses it. The function works correctly via pipe + `uniq -d`, but the unused variable is misleading. | `scripts/ralph-orchestrator.sh:203–204` | Remove the unused `_all_files=""` line. |
| MEDIUM | maintainability | `wait_for_slice()` is defined in `scripts/ralph-orchestrator.sh` but never called. The dependency-wait logic is handled instead by the `sleep 5` polling loop. Dead code that creates false impression of a blocking-wait primitive. | `scripts/ralph-orchestrator.sh:394–407`; `grep -n "wait_for_slice" scripts/ralph-orchestrator.sh` returns only line 394 (definition) | Either call `wait_for_slice` to replace the sleep-poll approach, or remove the function and update the comment about "wait for a specific slice". |
| MEDIUM | maintainability | `integration_merge()` in `scripts/ralph-orchestrator.sh` declares and increments `_conflicts` (line 415, line 447) but uses it only in a `log_error` message that is immediately followed by `return 1`, after which the caller ignores the conflict count. The variable adds visual complexity with no semantic benefit. | `scripts/ralph-orchestrator.sh:415,447,450` | Remove `_conflicts` variable; fold the count directly into the log message if needed, or drop it entirely. |
| MEDIUM | maintainability | Auto-detection of the newest directory-based plan (`scripts/ralph:103`) uses `find ... | while read -r d; do ... done | head -1` without sorting by modification time. Single-file plan fallback uses `ls -t` (sorted by mtime). This inconsistency means directory-based plans are selected in filesystem order, not recency order. With multiple active directory plans this selects unpredictably. | `scripts/ralph:103–105` vs `scripts/ralph:108` | Sort the `find` output by modification time: `find docs/plans/active -maxdepth 1 -type d ! -name active -newer /dev/null | xargs -I{} stat -f "%m %N" {} | sort -rn | head -1 | awk '{print $2}'` or equivalent for macOS/Linux portability. |
| MEDIUM | security | `ckpt_transition()` in `scripts/ralph-pipeline.sh` builds a JSON entry via raw string concatenation (lines 90, 92): `_entry="{\"from\": \"${_from}\", ...}"`. The `_from` and `_to` values are always internal constants, but `_reason` is passed from `_context`, which in principle could contain user-supplied text. Currently all `_context` values are hardcoded shell strings, so this is a latent risk rather than an active vulnerability. | `scripts/ralph-pipeline.sh:86–95`; callers at lines 351, 583, 777, 856 | Replace string concatenation with `jq -n --arg from "$_from" --arg to "$_to" --arg reason "$_reason" ...`. Pre-existing finding from r1 review; recorded in tech-debt. |
| MEDIUM | maintainability | `_total_iteration` is incremented in the outer loop (line 784) and again inside the inner loop for cases 1 and 6 (lines 798, 803). When the inner loop fails repeatedly, `_total_iteration` grows faster than once per inner iteration, causing `MAX_ITERATIONS` to be reached earlier than the user intends. | `scripts/ralph-pipeline.sh:784,798,803` | Pre-existing finding from r2 review. Decide whether `_total_iteration` should count outer-loop passes only, or all inner iterations. If the latter, remove the outer-loop increment and use only the inner increments. |
| LOW | null-safety | `run_outer_loop()` hardcodes `ckpt_transition "inner" "outer" "tests passed"` (line 583), bypassing `_prev_phase`. If `run_outer_loop` is ever called from a state other than "inner" (e.g., resumed from checkpoint in "outer" phase), the transition record will be incorrect. | `scripts/ralph-pipeline.sh:582–583` | Read `_prev_phase` before updating, as done in `run_inner_loop`. Pre-existing structural inconsistency. |
| LOW | readability | `scripts/ralph-pipeline.sh:670` PR prompt instructs the agent to write the PR URL with a hardcoded example: `echo "https://github.com/..." > .harness/state/pipeline/.pr-url`. An agent may copy the example literally. Layer 1 (`gh pr list`) and Layer 3 (log grep) provide fallback, so practical impact is LOW. | `scripts/ralph-pipeline.sh:659–671` | Replace the example with `gh pr view --json url --jq '.url' > .harness/state/pipeline/.pr-url`. Pre-existing finding recorded in tech-debt. |
| LOW | null-safety | `ralph-loop-init.sh` uses `archive_ts="${archive_ts:-$(date -u '+%Y%m%d-%H%M%S')}` for the pipeline state archive (line 87). This reuses the loop archive timestamp if the loop state was archived in the same second. In practice this produces a deterministic path collision only if both archives fire in the same second, which is unlikely but possible when re-initializing rapidly. | `scripts/ralph-loop-init.sh:77–88` | Assign a dedicated timestamp for the pipeline archive rather than reusing the loop's. |
| LOW | readability | `scripts/ralph-orchestrator.sh:441–444` uses `cat <<MERGE_EOF` inside `git merge -m "$(...)"`for the commit message. The MERGE_EOF is unquoted, so variable expansions inside would be interpreted — harmless here since only `$_slice_branch` and `$_int_branch` are referenced and both are safe, but it is an unexpected deviation from the repo's policy of using single-quoted HEREDOC delimiters for commit messages (`.claude/rules/git-commit-strategy.md`). | `scripts/ralph-orchestrator.sh:441–443` | Use `'MERGE_EOF'` (single-quoted) to match the repo convention. |

---

## Positive notes

- The `log_error` HIGH defect from the r2 review has been fixed: `log_error()` is now defined at `scripts/ralph:23` alongside `log()`.
- The pipe-subshell fixes from prior reviews are all carried forward correctly. The `_slices_file` temp-file pattern (line 605) avoids the POSIX sh pipe scope issue for all main iteration variables (`_completed`, `_failed`, `_running`).
- The phase transition ordering bug (from r1 and r2) is confirmed fixed in `run_inner_loop` (line 349: `_prev_phase` read before `ckpt_update`).
- `check_locklist_conflict()` and `detect_shared_files()` are correctly implemented as pure output functions that return values via command substitution — not subject to pipe-subshell scope loss.
- `ckpt_update()` correctly uses `--arg` for all user-controlled values (`_new_session`, `_pr_url`), consistent with the pipeline-robustness fixes.
- The abort command in `scripts/ralph` uses a temp file (`$_wt_tmp`) to iterate worktrees without a pipe subshell (line 484), correctly addressing the partially-resolved tech-debt item.
- Preflight probe design is clean: each probe fails fast, sets `_all_pass=false`, and the JSON report is built with `jq --argjson` rather than string concatenation.
- `new-ralph-plan.sh` correctly validates `slice_count` as a positive integer before use (line 110).

---

## Tech debt identified

No new deferred items beyond those already in `docs/tech-debt/README.md`. The MEDIUM findings above (unused variable, dead function, `_conflicts` counter, `_total_iteration` double-increment, `ckpt_transition` string concatenation) are pre-existing or low-impact. The prompt example URL (LOW) is already recorded.

The partially-resolved tech-debt item for `scripts/ralph abort` worktree-list pipe-subshell has been fully resolved by the temp-file approach in this diff.

---

## Recommendation

- **Merge: YES** — No CRITICAL or HIGH findings. The one HIGH finding from the previous review cycle (`log_error` undefined) has been resolved. The remaining findings are MEDIUM or LOW.

- **Fix before first production autonomous pipeline run (non-blocking for merge):**
  - Remove `wait_for_slice` dead function or connect it to actual slice-wait logic (MEDIUM, maintainability)
  - Fix directory-based plan auto-detection sort order (MEDIUM, silent incorrect behaviour with multiple active directory plans)

- **Follow-up (post-merge):**
  - Remove `_all_files` and `_conflicts` dead variables (MEDIUM, readability)
  - Address `_total_iteration` double-increment semantics (MEDIUM, pre-existing)
  - Replace `ckpt_transition` string concatenation with `jq --arg` (MEDIUM, pre-existing, latent security pattern)
  - Fix `run_outer_loop` hardcoded `"inner"` in `ckpt_transition` call (LOW, pre-existing)
  - Replace PR prompt example URL with `gh pr view` command (LOW, pre-existing, recorded in tech-debt)
  - Use single-quoted MERGE_EOF delimiter in `integration_merge` (LOW, convention consistency)
