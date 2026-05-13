# Self-review report: fix-xreview-placeholder-substitution (cycle 2)

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md`
- Reviewer: `reviewer` subagent (self-review skill, Claude Code) — re-run after cycle-1 CRITICAL fix
- Scope: diff quality only on `git diff main...HEAD` (6 commits on `fix/50/xreview-placeholder-substitution`, cumulative +1028/-20 across 12 files). Spec compliance, test execution results, and doc drift deferred to `/verify` and `/test`.
- Previous report: `docs/reports/self-review-2026-05-13-fix-xreview-placeholder-substitution.md` (cycle 1) — 1 CRITICAL, 2 MEDIUM, 3 LOW.
- Cycle-2 trigger: cycle-1 CRITICAL "Renderer failure silently bypasses the gate"; fix landed in commit `12a1984`.

## Evidence reviewed

- `git log main..HEAD --oneline` (6 commits: `0304686` → `4f15681` → `d2dd875` → `f3363b6` → `fd3e958` → `12a1984`).
- `git diff main...HEAD -- scripts/ralph-pipeline.sh` (entire cross-review block, lines 760–907; +86/−20 vs main).
- `git show 12a1984` — the cycle-2 fix commit only (the file body now reflects this commit).
- `scripts/ralph-pipeline.sh:760-907` — the full cross-review phase, including the new `_render_failed` flag, the awk renderer, the allowlist guard, the parser, the checkpoint/report_event JSONL lines, and the gate decision.
- `tests/test-xreview-gate-regression.sh` Phase 5 (lines 278-355) — gate decision driven end-to-end on a render-failure path, plus the three drift assertions (5d, 5e, 5f) against the production script.
- `.claude/skills/cross-review/prompts/adversarial-claude.md` — the metacharacter comment now lists `\` alongside `#` / `&` / `/` (cycle-1 LOW fix).
- `docs/tech-debt/README.md` — new row appended for the renderer-duplication MEDIUM (cycle-1 MEDIUM #1 deferred deliberately).
- `templates/base/` mirrors — `cmp` confirms byte-identical with the root-side files (`scripts/ralph-pipeline.sh`, `adversarial-claude.md`, both `SKILL.md` bodies, the `.agents/skills/cross-review/SKILL.md` Codex-side body). `scripts/check-sync.sh` and `scripts/check-skill-sync.sh` both green per the user-provided status.
- Cycle-1 CRITICAL re-check: traced every code path that could exit the renderer block to confirm `_render_failed` is set on all three failure modes (`render_failed_awk`, `render_failed_unresolved_placeholders`) and that the empty-`_base` case has been deliberately removed (line 777's `_base="${_base:-main}"` makes it unreachable — cycle-1 LOW fixed via deletion).
- Gate ordering verified by direct line-number inspection: `_render_failed` test at line 893, `_action_required` test at line 898, `FIX_ALL + _worth_considering` test at line 904. Test assertion 5f uses awk to confirm this ordering and would fail if a future refactor reordered the checks.

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | **Triage parser still runs after a render failure and can overwrite `_action_required` from a stale triage file.** On the `_render_failed=1` path the `claude -p` call is correctly skipped, but control still falls through to lines 869–874, where `find "$REPORTS_DIR" -name 'cross-review-triage-*' -newer "${PIPELINE_DIR}/checkpoint.json"` is unconditional. If a stale triage report from an earlier cycle in the same `REPORTS_DIR` happens to be newer than the cycle's checkpoint, `_action_required` / `_worth_considering` / `_dismissed` will be populated from it and recorded in both the `ckpt_update` checkpoint and the `report_event` JSONL (line 884–885). The gate decision still fails closed at line 893 (because `_render_failed=1` short-circuits before `_action_required` is consulted), so this is **not** a correctness regression. But the telemetry can show `"action_required":N, "render_failed":1` for a cycle whose reviewer was never invoked, which is misleading for operators triaging post-mortem JSONL. | `scripts/ralph-pipeline.sh:869-874` (unconditional parser), `:884-885` (telemetry recording), `:893-896` (short-circuit). | Either (a) gate the parser block on `[ "$_render_failed" -eq 0 ]` so a render failure does not silently pick up an unrelated triage file, or (b) document in a comment that the triage counts are advisory when `_render_failed=1`. Option (a) is the cleaner fix and matches the "fail closed, observable" intent. Not blocking for this cycle. |
| LOW | maintainability | **Drift assertion 5f matches by regex pattern, not by the canonical comparison operator.** `tests/test-xreview-gate-regression.sh:347-348` uses `awk "/_render_failed.*-ne/ ..."` and `/_action_required.*-gt/`. If a future refactor changes the operator (e.g. `[ "$_render_failed" = "1" ]` instead of `-ne 0`), the drift guard would silently fail to find the line and the awk `END` would evaluate `saw_render > 0 && saw_action > 0` as false → exit 1 → the test fails. That is the *correct* outcome for an unrelated stylistic change. The risk is the reverse direction: someone adds a *second* `_render_failed.*-ne` check elsewhere in the file later than the gate, but `if (!saw_render) saw_render = NR` is properly idempotent and pins on the first hit, so this is actually safe. No change required; flagging only because the awk pattern coupling looks fragile at first read. | `tests/test-xreview-gate-regression.sh:345-355`. | Optional: add a one-line comment above the awk explaining the `if (!saw_*)` first-hit semantics. Not blocking. |

## Positive notes

- **Cycle-1 CRITICAL is genuinely fixed.** The render-failure short-circuit at line 893 precedes every other gate path, and `_render_failed=1` is set on both reachable failure modes (`render_failed_awk` at line 829, `render_failed_unresolved_placeholders` at line 840). The previously-unreachable empty-`_base` branch was correctly removed rather than kept as defensive theater. Test assertion 5b (`RENDER_FAILED=1 gate_decision` returns non-zero) drives the exact failure shape end-to-end.
- **Gate ordering is correct and locked in by drift assertion.** `_render_failed` (893) → `_action_required` (898) → `FIX_ALL + _worth_considering` (904). The 5f awk drift assertion will trip if a future refactor reorders these checks, which is the right place to catch the regression.
- **Telemetry parity:** the `render_failed` field is added to both the `ckpt_update` (`.cross_review_triage`) JSON and the `report_event "cross-review"` JSONL line at lines 884–885. Operators get the failure signal in two channels, not just the log.
- **Cycle-1 LOWs are all addressed:**
  - The unreachable empty-`_base` branch has been deleted (not retained as defensive theater) — confirmed by reading the renderer block: the only sources of `_render_failed=1` are awk-exit-non-zero and the allowlist guard.
  - `\` has been added to the prompt-file metacharacter comment list (`adversarial-claude.md:6`), aligning with the pipeline comment (lines 796–801) and the test parameterisation (`feature\back` case in `test-xreview-prompt-render.sh`).
  - The `grep ... || true` LOW from cycle 1 was *not* addressed in this commit; I am not re-raising it because it's a stylistic preference and the cycle-1 review explicitly tagged it LOW / non-blocking.
- **Cycle-1 MEDIUM #1 (renderer duplicated 3 ways) is correctly logged as tech debt** rather than fixed inline. The new row in `docs/tech-debt/README.md` cites the cycle-1 report, the file paths, and a concrete pay-down trigger ("next time the renderer grows"). This is the right call — the extraction would be a larger change than the bug fix and is explicitly noted as out of the issue #50 scope.
- **Cycle-1 MEDIUM #2 (`_adv_prompt=""` overloaded as sentinel) is dissolved by the fix.** The new `_render_failed` flag replaces the overloaded-empty-string pattern; the `claude -p` invocation is now gated on `[ "$_render_failed" -eq 0 ]` (line 844), which reads as "did rendering succeed?" — exactly the cycle-1 recommendation.
- **Test discipline is excellent.** Phase 5 of `test-xreview-gate-regression.sh` adds two end-to-end behavioral assertions (5b: render-failure regresses; 5c: render-success with no findings proceeds — the "no false positives" check) plus three drift assertions (5d, 5e, 5f) against the production script. The 5c assertion is the easy one to forget; its presence is a sign of careful test design.
- **Mirror discipline holds.** `templates/base/scripts/ralph-pipeline.sh`, `templates/base/.claude/skills/cross-review/prompts/adversarial-claude.md`, and both `SKILL.md` mirrors are byte-identical with their root counterparts. `check-sync.sh` and `check-skill-sync.sh` both green per the user-supplied status.
- **No new code smells introduced.** No new debug prints, no commented-out code, no hardcoded secrets, no broadened `eval`/`set +e` blocks. The new code reuses the existing `log_error` / `report_event` / `ckpt_update` infrastructure. The awk block is single-quoted (no shell expansion in the program text) and replacement values flow via `ENVIRON[]` (no command substitution surface) — same security posture as the cycle-1 renderer.
- **Commit message documents the failure shape and the fix.** `12a1984`'s body explicitly cites issue #50, names the silent-bypass mechanism, and explains why `_render_failed` is checked before `_action_required`. Future grep on "silent bypass" / "fail closed" will find it.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Triage parser at `scripts/ralph-pipeline.sh:869-874` runs unconditionally even on render-failure paths and can record misleading `action_required` counts in checkpoint + JSONL telemetry | LOW — observability noise, not a correctness bug (gate still fails closed at line 893 because `_render_failed` short-circuits) | Fix is one `if` wrapping the parser block, but it is genuinely orthogonal to the cycle-2 CRITICAL; folding it in risks scope creep on the issue-#50 PR | Next time an operator reports confusing JSONL telemetry on a render-failure cycle, OR when the parser is touched for any other reason | This report (LOW #1); `scripts/ralph-pipeline.sh:869-874` |

_(Row already exists in `docs/tech-debt/README.md` for the renderer-duplication MEDIUM from cycle 1. The new row above is non-blocking and can be appended on follow-up if the PR ships as-is.)_

## Recommendation

- Merge: **merge** — the cycle-1 CRITICAL is genuinely fixed (no silent-bypass path remains), all cycle-1 LOWs called out as actionable were addressed in commit `12a1984`, the cycle-1 MEDIUM that was correctly deferred is recorded as tech debt, and the only new findings are two LOWs that do not affect correctness. The post-implementation pipeline can proceed to `/verify`.
- Follow-ups:
  1. **(non-blocking)** Wrap the triage parser at `scripts/ralph-pipeline.sh:869-874` in `if [ "$_render_failed" -eq 0 ]; then ... fi` so checkpoint + JSONL telemetry reflect the actual reviewer activity for the cycle. Add a corresponding test that asserts `_action_required=0` and `_worth_considering=0` in the recorded checkpoint when `_render_failed=1`.
  2. **(non-blocking, tech-debt)** Pay down the renderer-duplication MEDIUM the next time the renderer grows (extract to `scripts/ralph-cli-driver.sh` alongside `count_triage_findings` and `pick_reviewer`).
  3. **(non-blocking, doc)** Consider adding a one-line comment above the 5f drift-assertion awk explaining the `if (!saw_*)` first-hit semantics, for future test readers.

## STOP-condition summary

Per `.claude/skills/self-review/SKILL.md` and `.claude/rules/post-implementation-pipeline.md`, the pipeline must stop on CRITICAL findings before `/verify`.

- **CRITICAL findings: 0** — the cycle-1 CRITICAL is fixed; no new CRITICALs introduced.
- **HIGH findings: 0**.
- **MEDIUM findings: 0** — cycle-1 MEDIUM #1 logged as tech debt; cycle-1 MEDIUM #2 dissolved by the fix.
- **LOW findings: 2** — unconditional parser-on-render-failure (telemetry noise only); drift-assertion regex coupling (defensive note only). Both non-blocking.

**No STOP condition. The pipeline may proceed to `/verify`.**
