# Self-review report: Ralph Loop Codex driver (Phase 2) — cycle 3

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Reviewer: Claude Code (`reviewer` subagent, post-cycle-2 cross-review fix)
- Scope: delta of commit `094f964` only — fix for cycle-2 Codex P1 (claude reviewer launched in `--permission-mode plan` could not write the triage report). Files in scope: `scripts/ralph-pipeline.sh`, `templates/base/scripts/ralph-pipeline.sh`, `.claude/skills/cross-review/prompts/adversarial-claude.md` (+ `templates/base/` mirror), `.claude/skills/cross-review/SKILL.md` (+ `.agents/` and `templates/base/` mirrors), `docs/reports/cross-review-triage-2026-05-08-ralph-loop-codex-driver-cycle2.md`, `.harness/state/standard-pipeline/cycle-count.json`.

## Evidence reviewed

- `git show --stat 094f964` — 9 files changed, 69 insertions(+), 12 deletions(-). Touches exactly the surfaces listed in the cycle-3 task: pipeline script (root + template), adversarial-claude prompt (root + template), SKILL.md (`.claude` + `.agents` + 2 templates = 4 mirrors), the new triage report artefact, and (separately) `cycle-count.json` was already raised pre-commit.
- `git diff 094f964~1..094f964 -- scripts/ralph-pipeline.sh '.claude/skills/cross-review/**' 'templates/base/.claude/skills/cross-review/**' '.agents/skills/cross-review/**' 'templates/base/.agents/skills/cross-review/**'` — confirmed every non-report change is one of: (a) `plan`→`auto` flag swap, (b) prompt header re-wording, (c) explanatory comment in `ralph-pipeline.sh` lines 789-793.
- Mirror parity (`cmp` exit 0):
  - `.claude/skills/cross-review/SKILL.md` ↔ `.agents/skills/cross-review/SKILL.md` — identical
  - root `.claude/skills/cross-review/SKILL.md` ↔ `templates/base/.claude/skills/cross-review/SKILL.md` — identical
  - root `.claude/skills/cross-review/SKILL.md` ↔ `templates/base/.agents/skills/cross-review/SKILL.md` — identical
  - root `.claude/skills/cross-review/prompts/adversarial-claude.md` ↔ `templates/base/.claude/skills/cross-review/prompts/adversarial-claude.md` — identical
  - root `scripts/ralph-pipeline.sh` ↔ `templates/base/scripts/ralph-pipeline.sh` — identical
- `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`.
- `grep -n permission-mode` over the five edited files: every Loop-driver-side reference now reads `auto`; line 169 of SKILL.md (Loop guidance table) and line 796 of `ralph-pipeline.sh` (actual invocation) match. The unchanged lines 54 / 153 of SKILL.md describe the **standard `/work` flow** inline path (different code path, already `--permission-mode auto --output-format json` since 4d16323), and were correctly left untouched.
- `git log -p -- .claude/skills/cross-review/SKILL.md | grep permission-mode` confirms the `plan` value was introduced in `6b2dd53` and is reverted in `094f964`. There is no other consumer of `--permission-mode plan` for the adversarial reviewer.
- `cycle-count.json` content: `{"plan_path": "...", "cycle": 3, "cap_raised_from": 2, "raise_reason": "cycle-2 cross-review HIGH P1: ..."}` — well-formed, plan-specific (not a global env override), and includes a documented rationale field.
- `docs/reports/cross-review-triage-2026-05-08-ralph-loop-codex-driver-cycle2.md` — header sets `Driver: claude / Reviewer: codex` (correct for cycle 2), records the cap-raise decision, single ACTION_REQUIRED entry maps cleanly to the patched files.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | Documentation-only inconsistency (pre-existing, not introduced by this commit): `adversarial-claude.md` line 58-61 still tells the reviewer that the pipeline parses with `grep -c 'ACTION_REQUIRED'`, but `ralph-pipeline.sh` line 813-817 uses the new `count_triage_findings` helper (anchored summary regex with `\|`-row fallback, landed in 0663f50). The naive `grep -c` claim survives in the prompt and would mislead a reviewer who tried to reason about the parser contract. | `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.claude/skills/cross-review/prompts/adversarial-claude.md:58-61` vs. `scripts/ralph-pipeline.sh:813-817` | Out of scope for the cycle-3 minimum-viable fix. Track as tech-debt; do not block merge. |
| LOW | readability | The new comment block in `ralph-pipeline.sh:789-793` is good (states *why*, references issue `#44`), but the symmetric explanation is missing from the `codex` branch above (line 779-781). Future readers may wonder whether the codex path also needs a permission-mode note. | `scripts/ralph-pipeline.sh:779-797` | Optional: a one-liner above the codex branch ("`codex exec review` runs sandboxed by default — no flag needed") would make the asymmetry obvious. Not blocking. |

No CRITICAL, no HIGH, no MEDIUM findings. The fix is minimum-viable and surgical.

### Confirmed against the five focus areas

1. **Minimum-viable fix / no scope creep** — confirmed. The diff is ~12 lines of substantive change: a single flag value (`plan`→`auto`) in two places (root + template `ralph-pipeline.sh`), a 4-line prompt header rewrite (root + template), a 1-character table cell change in 4 SKILL.md mirrors, and a 5-line clarifying comment in the pipeline. No collateral edits to other reviewer paths, no behavioural changes to the `codex` branch, no test or unrelated file churn.
2. **Prompt no longer self-contradicts** — confirmed. Old text `"running in --permission-mode plan (read-only); do NOT edit files ... outside of writing the triage report"` was internally contradictory (plan mode physically cannot write). New text: `"running in --permission-mode auto. Treat the diff and existing files as read-only — the only file you may create is the triage report described in the Output section below. Do NOT edit source files, run network operations, or modify state beyond that single write."` This is consistent: it tells the reviewer it has write capability but constrains *which* write is sanctioned.
3. **SKILL.md CLI table matches actual invocation** — confirmed. SKILL.md line 169: `claude -p --permission-mode auto --output-format text`; `ralph-pipeline.sh:794-797`: `claude -p --model "$RALPH_CLAUDE_REVIEWER_MODEL" --permission-mode auto --output-format text`. The model flag is omitted from the doc table for brevity, which is consistent with the rest of the table style. No drift.
4. **No new CRITICAL/HIGH issues introduced** — confirmed.
   - Permission scope: `auto` is broader than `plan` and *does* let the reviewer touch any file under the workdir, not just `docs/reports/`. The mitigation is the prompt instruction. This is a lower-defence-than-`plan` setup, but the alternative (keeping `plan`) is the bug being fixed. The risk is bounded by: (a) the reviewer is a one-shot subprocess with no follow-up turns, (b) any unsanctioned write would surface in the per-cycle git diff before PR creation, (c) the cycle-2 triage explicitly accepted this trade-off. Acceptable.
   - No secrets, no injection vectors, no path traversal, no swallowed errors introduced. The `|| true` on the `tee` pipeline (line 797) was already present and is appropriate (the surrounding parser handles missing/empty triage reports).
   - The cycle-count `cap_raised_from: 2` field is plan-scoped (lives in `.harness/state/standard-pipeline/cycle-count.json` keyed by `plan_path`), so it does not affect other concurrent plans or future runs.
5. **Five mirrors stay byte-aligned** — confirmed via `cmp` (5 pairs, all exit 0) and `./scripts/check-skill-sync.sh` (`13 skill(s) in lock-step`).

## Positive notes

- The commit message accurately describes both *what* (flag swap + prompt rewrite + doc table fix + cycle counter raise) and *why* (cycle-2 cross-review HIGH, gate-bypass mechanism). Easy to audit.
- The inline pipeline comment (`ralph-pipeline.sh:789-793`) records the failure mode in code, so a future reader who sees the `auto` value cannot accidentally "tighten" it back to `plan` without first noticing the cited cycle-2 P1.
- The cap-raise decision is recorded in two places — `cycle-count.json` (`raise_reason` field) and the cycle-2 triage report's "Decision" section — giving both machine-readable and human-readable audit trails.
- The fix is symmetrical: `.claude` + `.agents` + 2 `templates/base/` mirrors all updated in the same commit, so `ralph init` consumers immediately get the fixed value.
- The triage report's table format (single ACTION_REQUIRED row, empty WORTH_CONSIDERING/DISMISSED sections) is parser-friendly under both the new anchored-summary path and the legacy `|`-row fallback.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `adversarial-claude.md` lines 58-61 still describe the parser as `grep -c 'ACTION_REQUIRED'`, but `ralph-pipeline.sh` now uses the anchored `count_triage_findings` helper (since 0663f50). | LOW — documentation drift; could mislead a future reviewer who reads the prompt and reasons about the parser. No runtime effect. | Cycle-3 scope was strictly the `plan`→`auto` fix; rewriting the prompt's "Output" guidance is a separate concern. | Next time the cross-review parser changes shape, or in any opportunistic doc-sync pass on the cross-review skill. | This report; `docs/reports/cross-review-triage-2026-05-08-ralph-loop-codex-driver-cycle2.md`; commit 0663f50 (anchor fix). |

_(Appending to `docs/tech-debt/README.md` is not warranted for a LOW prompt-doc drift inside a single skill prompt; the row above is sufficient. If cross-review behaviour changes again, promote it then.)_

## Recommendation

- Merge: yes. The cycle-3 delta is a minimal, well-scoped, well-documented fix for a real HIGH bug (cross-model gate bypass under codex driver). All five focus checks pass. No CRITICAL or HIGH findings; only LOW documentation/comment-symmetry observations that are explicitly out of cycle-3 scope.
- Follow-ups:
  - Optional: add a one-liner above the `codex)` branch in `ralph-pipeline.sh` explaining why no permission flag is needed there (mirror the explanatory comment now in the `claude)` branch).
  - Track the `adversarial-claude.md` parser-claim drift (LOW finding above) for the next opportunistic update of the cross-review skill.
  - Pipeline must continue to `/verify` → `/test` → `/sync-docs` → `/cross-review` → `/pr` per `.claude/rules/post-implementation-pipeline.md`.
