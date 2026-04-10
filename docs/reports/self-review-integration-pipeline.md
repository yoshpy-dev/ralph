# Self-review report: Integration Pipeline Feature (feat/ralph-loop-v2)

- Date: 2026-04-10
- Plan: feat/ralph-loop-v2 integration pipeline additions
- Reviewer: reviewer agent (self-review specialist)
- Scope: Diff quality only — `scripts/ralph-orchestrator.sh` (~107 new lines), `scripts/ralph-pipeline.sh` (~35 new lines for `--skip-pr`/`--fix-all`), `.claude/rules/subagent-policy.md` (+2 lines), `.claude/skills/loop/SKILL.md` (+1 line update)

## Evidence reviewed

- `git show 9ca7b90` — feat: add integration branch, sequential merge, and unified PR support
- `git show 0281edb` — fix: remove unsupported --pipeline flag and add missing terminal statuses
- `git show 17e678a` — docs: clarify standard flow vs Ralph Loop execution model differences
- `scripts/ralph-orchestrator.sh` lines 240–270, 409–595, 600–855 (read directly)
- `scripts/ralph-pipeline.sh` lines 21–54, 520–540, 724–739 (read directly)
- `.claude/rules/subagent-policy.md` (full file)
- `.claude/skills/loop/SKILL.md` (full file)
- `docs/tech-debt/README.md` (to check for pre-existing debt entries)
- `MEMORY.md` (recurring patterns)

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | null-safety | `create_integration_branch()` return value unchecked in `main()` — if `git branch` fails (e.g., due to permissions) the function returns 1 and `set -eu` exits, but `INTEGRATION_BRANCH` is already set to `"integration/${_slug}"` before the failure. Callers in the pipeline loop will then reference an invalid branch. | `ralph-orchestrator.sh:616`: `create_integration_branch "$PLAN_SLUG" "$_base_branch"` with no `|| exit` wrapper; `ralph-orchestrator.sh:264`: `git branch "$INTEGRATION_BRANCH" "$_base" 2>/dev/null \|\| { log_error ...; return 1; }` | Add explicit guard: `create_integration_branch "$PLAN_SLUG" "$_base_branch" \|\| { log_error "Cannot create integration branch. Aborting."; exit 1; }` to make the failure path obvious and intentional rather than relying on implicit `set -eu` propagation. |
| MEDIUM | readability | `create_unified_pr()` captures `gh pr create` with `2>&1` merged into `_pr_url`. If `gh` emits warnings to stderr (e.g., "Notice: ..."), those strings contaminate `_pr_url`, breaking downstream uses including the JSON report field and the `log "Unified PR created: ${_pr_url}"` output. | `ralph-orchestrator.sh:481–513`: `_pr_url="$(gh pr create ... 2>&1)"` | Capture stdout only: `_pr_url="$(gh pr create ... 2>/dev/null)"`, and log stderr separately to the orchestrator log if needed. |
| MEDIUM | readability | PR body test plan has `- [ ] Integration merge passed without conflicts` as an unchecked item, even though the PR is only created after `integration_merge()` succeeds. This misrepresents the merge state to reviewers. | `ralph-orchestrator.sh:507`: `- [ ] Integration merge passed without conflicts` inside `create_unified_pr()`, which is only called when `integration_merge` returns 0. | Change to `- [x] Integration merge passed without conflicts`. |
| MEDIUM | readability | PR title hardcodes `feat:` regardless of task type. When the plan slug is a bugfix or refactor, the PR title is incorrect. Also, the raw slug (e.g., `feat: 2026-04-10-my-feature`) includes a date prefix that adds noise to the GitHub PR list. | `ralph-orchestrator.sh:484`: `--title "feat: ${_plan_slug}"` | Either read the objective from `_manifest.md` (using `grep -m1 'Objective:' ...`), or strip the date prefix from the slug. At minimum, use `chore:` as a safer generic default or pass the task type through from the orchestrator invocation. |
| MEDIUM | maintainability | `integration_branch` field in the execution report JSON (line 832) and the orchestrator state JSON (line 667) is suppressed when `--unified-pr` is not set. Since integration branch is now **always** created (comment at line 615: "Always create an integration branch for sequential merge"), the field should always be populated. Users running without `--unified-pr` will not see which branch was created in the report. | `ralph-orchestrator.sh:832`: `"integration_branch": "$([ "$UNIFIED_PR" -eq 1 ] && echo "${INTEGRATION_BRANCH}" || echo "")"` | Always emit `"integration_branch": "${INTEGRATION_BRANCH}"` in both report JSON blocks. |
| LOW | readability | `MERGE_EOF` heredoc delimiter is unquoted in `integration_merge()`. The repo convention (documented in `.claude/rules/git-commit-strategy.md` and MEMORY) requires single-quoted delimiters. The content (`${_slice_branch}`, `${_int_branch}`) is safe (branch names have no special characters), so there is no functional risk, but it is a convention violation. Pre-existing in `ralph-orchestrator.sh:441–443`. | `ralph-orchestrator.sh:441`: `git merge ... -m "$(cat <<MERGE_EOF` | Change to `<<'MERGE_EOF'` and use literal variable names in the message, or accept as pre-existing known debt (see MEMORY). |
| LOW | readability | Redundant `git checkout` in `main()` after `integration_merge()` succeeds. `integration_merge()` already restores `HEAD` to `_orig_branch` before returning. Then `main()` captures `_orig_branch` again (same value) and checks out `INTEGRATION_BRANCH`. The double-checkout sequence is harmless but confusing. | `ralph-orchestrator.sh:793–794` vs `ralph-orchestrator.sh:458`: both do `git checkout "$_orig_branch"` | Remove the redundant `_orig_branch` capture and checkout in `main()`, or add a comment explaining that `integration_merge()` already restores HEAD. |
| LOW | maintainability | `_conflicts` counter in `integration_merge()` is incremented on conflict but never read after that. The function returns 1 immediately after the first conflict, making the counter permanently 0 or 1 with no consumer. This entry is already recorded in `docs/tech-debt/README.md` line 24. | `ralph-orchestrator.sh:415,447`: `_conflicts=0` ... `_conflicts=$((_conflicts + 1))` | Remove the counter or use it in the log message: `log_error "Sequential merge aborted at slice ${s} (${_conflicts} conflict(s))."`. (Pre-existing tech debt.) |
| LOW | readability | `run_integration_pipeline()` writes a fresh `INT_PROMPT` to `.harness/state/loop/PROMPT.md` using an unquoted heredoc (`<<INT_PROMPT`). This causes shell expansion of `${_int_branch}`, `${_base_branch}`, `${_plan_slug}` at write time — which is intentional — but also means any future maintainer adding backticks or `$(...)` to the prompt template would silently expand them. | `ralph-orchestrator.sh:538`: `cat > .harness/state/loop/PROMPT.md <<INT_PROMPT` | Add a comment noting that the heredoc is intentionally unquoted to allow variable substitution into the prompt, so future maintainers do not inadvertently add shell-interpreted content. |

## Positive notes

- `--skip-pr` and `--fix-all` flag implementation in `ralph-pipeline.sh` is clean and well-placed. The `--fix-all` logic correctly reads the sidecar file before applying the override, and the `SKIP_PR=1` branch explicitly sets `status = "complete"` so downstream callers get a deterministic checkpoint state.
- `integration_merge()` correctly saves and restores `HEAD` both on success and on the conflict error path (line 449, 458).
- The prompt adaptation loop in `run_integration_pipeline()` (`sed "s|...|...|g"` per prompt file) cleanly specializes the integration context without forking the prompt files.
- The 3-tier detection for self-review results (sidecar → JSON output → grep fallback) is preserved correctly in the `--fix-all` path.
- `subagent-policy.md` and `SKILL.md` changes are accurate, concise, and do not introduce ambiguity.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `create_unified_pr()` captures `gh pr create` with `2>&1`, contaminating `_pr_url` with any gh stderr warnings | MEDIUM: wrong URL stored in report/checkpoint if gh emits a notice | Functional risk is low (gh notices don't appear on clean push), fix requires testing with live gh environment | Next revision of `create_unified_pr()` or when a contaminated URL is observed in practice | docs/reports/self-review-integration-pipeline.md |
| PR title hardcodes `feat:` regardless of task type and includes raw slug with date prefix | MEDIUM: incorrect commit type on non-feature plans; noisy PR list | Requires reading _manifest.md at PR creation time; slightly more complex | Next `create_unified_pr()` enhancement or when a non-feature Ralph Loop plan is executed | docs/reports/self-review-integration-pipeline.md |

## Recommendation

- Merge: **YES with follow-ups**
- No CRITICAL findings. Two MEDIUM findings (unchecked PR checklist item, `2>&1` contamination) are low-risk in practice but should be fixed before this feature is used in production Ralph Loop runs.
- Follow-ups (in priority order):
  1. Fix `- [ ] Integration merge passed without conflicts` → `- [x]` in `create_unified_pr()` PR body (trivial, no risk).
  2. Change `gh pr create ... 2>&1` to `gh pr create ... 2>/dev/null` to prevent stderr contamination of `_pr_url`.
  3. Always emit `integration_branch` in report JSON regardless of `--unified-pr` flag.
  4. Address PR title hardcoding in a subsequent enhancement.
