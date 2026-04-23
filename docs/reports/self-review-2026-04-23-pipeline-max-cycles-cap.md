# Self-review report: pipeline-max-cycles-cap

- Date: 2026-04-23
- Plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: 5 commits on `feat/pipeline-max-cycles-cap` (`5a42478`, `009428f`, `7568755`, `7b2e2e2`, `878f5d2`); `git diff main...HEAD` = 18 files, +319/-41. No spec-compliance or test-coverage evaluation — those belong to `/verify` and `/test`.

## Evidence reviewed

- `git diff main...HEAD --stat`: 18 files touched. Groupings:
  - Config / scripts: `scripts/ralph-config.sh`, `scripts/ralph-pipeline.sh` (+ mirrored `templates/base/`).
  - Skills: `.claude/skills/work/SKILL.md`, `.claude/skills/codex-review/SKILL.md`, `.claude/skills/pr/SKILL.md` (+ mirrored `templates/base/`).
  - Rules / docs: `.claude/rules/post-implementation-pipeline.md`, `docs/quality/definition-of-done.md`, `docs/recipes/ralph-loop.md` (+ mirrored `templates/base/`).
  - Plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md` (new file, 168 lines).
  - Tests: `tests/test-ralph-config.sh`.
- Template parity: ran `cmp` on all 8 mirrored pairs — every pair is byte-identical.
- `scripts/check-sync.sh`: PASS (`ROOT_ONLY: 0`, `TEMPLATE_ONLY: 9`, `KNOWN_DIFF: 3`).
- `bash tests/test-ralph-config.sh`: 27/27 pass, including the 3 new assertions for `RALPH_STANDARD_MAX_PIPELINE_CYCLES`.
- Added-line scans: no `console.log`/`TODO`/`FIXME`/`debug` leftovers; no hardcoded secrets, tokens, API keys, or passwords.
- `scripts/archive-plan.sh` accepts absolute paths (`[ -f "$arg" ] || [ -d "$arg" ]` branch at lines 23-24), so the change in `/pr` Step 5 from `<slug>` to `<absolute-plan-path>` is compatible.
- `grep` confirmed `_finalize "max_outer_cycles"` is the actual call site at `scripts/ralph-pipeline.sh:971`, so the rules-file reference is accurate.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | unnecessary-change / readability | `/codex-review` Step 3 "Load triage context" still instructs "Read the active plan from `docs/plans/active/`", which directly contradicts the newly added Step 0 "Hard prohibition: Do NOT rediscover the plan by rescanning `docs/plans/active/` once `active-plan.json` exists." A future agent reading Step 3 literally could justify rescanning the directory — defeating the very guarantee the Codex-adversarial-finding-#2 change was introduced to provide. | `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.claude/skills/codex-review/SKILL.md:28` (prohibition) vs `:45` (unchanged rescan instruction). Mirrored in `templates/base/.claude/skills/codex-review/SKILL.md`. | Update Step 3 bullet to: "Read the active plan using the path recorded in `active-plan.json` (Step 0) — do not rescan `docs/plans/active/`." Apply the same edit to the templated copy. |
| LOW | maintainability | `/codex-review` Step 7 says "If the user chooses a re-run path AND `active-plan.json` exists: increment `cycle-count.json` (`cycle += 1`)." The prompt does not specify whether "re-run path" includes the cap-reached Option 1 ("上限を一時的に引き上げて再実行"). Because Option 1 instructs the user to export a higher cap and re-run, the counter presumably should still increment on that path — but the skill text doesn't say so, and an agent could treat "raise the cap" as a pre-re-run action that skips the increment. | `.claude/skills/codex-review/SKILL.md:105` (Option 1 of cap-reached flow) and `:120-123` (Step 7's "re-run path" wording). | Clarify Step 7 with explicit enumeration: "A 're-run path' is any option that returns to `/self-review`: Case A Option 1, Case B Option 1 (both not-cap-reached), and the cap-reached Case A Option 1." |
| LOW | maintainability | `/work` SKILL.md numbers the new sub-step `0.5.` but does not renumber subsequent steps; Step 0 is (correctly) left as-is, but the visual ordering `0 → 0.5 → 1 → ...` is unconventional and slightly harder to reference than using e.g. `0a`/`0b` or promoting to a full numeric step. Markdown still renders it, and run-time behavior is unaffected. | `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.claude/skills/work/SKILL.md:15` (the `0.5.` line). | Optional: rename to `0a.` for consistency with existing `3a.` sub-step convention in the same file (line 27). Non-blocking. |
| LOW | readability | Plan's Implementation Outline ("Step 10 (new)") described `/pr` cleanup as a new `/work` step, but the implementation folded cleanup into `/pr` Step 6 directly. `/work` SKILL.md step 9f mentions the cleanup only in prose, not as a dedicated step. This is a minor spec-vs-code deviation that `/verify` will flag as a drift concern but is otherwise harmless — the behavior is correctly owned by `/pr` SKILL.md. | `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md:95` vs `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.claude/skills/pr/SKILL.md:33-35`. | Non-blocking. Consider adding a one-line deviation note to the plan so `/verify` has a paper trail. |
| LOW | maintainability | The deferred "script the counter" decision (Codex advisory #1) is tracked in the plan's Open Questions but not in `docs/tech-debt/`. If this diff merges as-is, the JSON manipulation remains entirely in prompt-text form with no deterministic helper. Risk: agents may hand-roll `jq`/`sed` and produce inconsistent `cycle-count.json` shapes across invocations. Not CRITICAL because the prompts are clear today, but worth recording as debt. | `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md:147` (Open question), no entry in `docs/tech-debt/README.md`. | Add a one-line entry to `docs/tech-debt/README.md` pointing to the plan, with the trigger "next time a `/codex-review` re-run miscounts or two skills disagree on JSON shape." Captured in the "Tech debt identified" section below. |
| INFO | unnecessary-change | None. Every file in the diff is listed in the plan's "Affected areas" and every change maps to an Acceptance criterion. No formatting-only churn, no stray imports, no accidental file inclusions. | `git diff main...HEAD --stat` cross-referenced against plan lines 45-54. | No action. |

## Positive notes

- **Template parity is perfect.** All 8 mirrored file pairs (`scripts/` ↔ `templates/base/scripts/` and `.claude/` ↔ `templates/base/.claude/`) are byte-identical under `cmp`. This is exactly the discipline the reviewer memory (`pattern_mirror_discipline`) flags as commonly forgotten, and it was executed correctly across two separate commits (`5a42478` and `7568755`).
- **Test coverage for the new variable is symmetrical with existing variables.** `test_defaults`, `test_env_override`, and `test_validate_all` each got one new assertion, and `validate_all` got an extra "rejects zero" case — consistent with the surrounding table-driven style.
- **The cycle-counter semantics in the skills are internally consistent.** `/work` initializes `cycle: 1` at step 0.5.d; `/codex-review` Step 0 reads without incrementing; Step 7 increments only on user-chosen re-run; `/pr` deletes both state files on success. The arithmetic `CAP_REACHED = (cycle >= RALPH_STANDARD_MAX_PIPELINE_CYCLES)` correctly yields one re-run at the default cap of 2 (first run sees `cycle=1`, `CAP_REACHED=false`; second run sees `cycle=2`, `CAP_REACHED=true`).
- **Rules file `.claude/rules/post-implementation-pipeline.md` got a well-placed new subsection ("Pipeline cycle cap") under "Re-run after Codex ACTION_REQUIRED fix" rather than a separate top-level section, which keeps the SSoT structure tidy.
- **`/pr` SKILL.md correctly leaves state files in place on failure** (line 35) — this matches the plan's intent that "`/pr` 失敗時にカウンタが残ること（再実行で続きから数えられること）".
- **No debug markers, no secret-like strings, no commented-out code, no leftover TODOs.** Hook scans and manual diff inspection came up clean.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Cycle-count and active-plan JSON manipulation lives only in skill prompt text; no `scripts/` helper. | Cross-skill drift risk: two skills could write mutually incompatible JSON shapes under edge conditions (missing file, partial write, concurrent `/work` sessions). | Plan explicitly chose skill-docs-only delta to minimize surface; Codex advisory #1 was see-and-deferred. | Next time `/codex-review` miscounts, or two skills disagree on JSON shape, or a user reports `active-plan.json` corruption. | `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md` (Open questions, "Codex 指摘 1"). |

_(The table row above should also be appended to `docs/tech-debt/README.md`. This diff does not touch that file, so the append is a follow-up action — flagged here rather than quietly added to the diff.)_

## Recommendation

- Merge: **Yes, with one fix first.** The MEDIUM finding (Step 3 contradicting Step 0 in `/codex-review` SKILL.md) should be resolved before PR creation because it directly undermines the plan's adopted Codex adversarial finding #2. This is a one-line edit in two files (source + template). Everything else is LOW/INFO and can be handled as follow-ups.
- Follow-ups:
  - Fix the Step 3 rescan instruction to reference `active-plan.json` (both `.claude/skills/codex-review/SKILL.md:45` and its template mirror).
  - Optional: clarify which options count as "re-run path" in `/codex-review` Step 7 (LOW).
  - Optional: record the "script the counter" debt in `docs/tech-debt/README.md` (LOW).
  - `/verify` should confirm `CLAUDE.md` / `AGENTS.md` / `README.md` sync (plan listed them as required; diff did not touch them). Out of scope for `/self-review`.

---

## Cycle 2 re-review (2026-04-23, commit `e27102a`)

- Scope: Focused re-review of the Codex ACTION_REQUIRED fix commit only. Two files touched plus their `templates/base/` mirrors: `.claude/skills/work/SKILL.md` (Step 0.5.d) and `.claude/skills/codex-review/SKILL.md` (Case B cap-reached branch).
- Method: `git diff 7f3e9f5..e27102a --` on the 4 paths; `cmp` on both mirror pairs; side-by-side comparison of Case A vs Case B cap-reached wording; inspection of `/work` Step 0.5.d's three-branch logic.

### Evidence reviewed

- Both mirrors byte-identical under `cmp` (`.claude/skills/work/SKILL.md` ↔ `templates/base/.claude/skills/work/SKILL.md`; same for `codex-review`).
- `/work` Step 0.5.d now enumerates 3 branches: (i) file missing → initialize `cycle: 1`; (ii) file exists AND `plan_path` matches → preserve existing counter; (iii) file exists AND `plan_path` differs → AskUserQuestion (reset vs abort). All three scenarios covered.
- `/codex-review` Case B cap-reached branch now invokes AskUserQuestion with 3 options (raise cap / PR / abort), symmetric with Case A cap-reached (lines 102–107 vs 115–120).
- Step 7's re-run increment wording (line 126) already explicitly names "the cap-reached `上限を一時的に引き上げて再実行` option" as a re-run path — covers both Case A Option 1 and Case B Option 1 without further edits. The cycle-1 LOW finding on enumerating re-run paths is partially addressed as a byproduct.

### Findings (cycle 2 only)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability / copy-paste consistency | Case A Option 1 says `have the user set a higher ... (e.g. export it) and re-run`; Case B Option 1 says `have the user export a higher ... and re-run`. Same intent, slightly different wording. Not a defect; only worth flagging because the triage report explicitly asked for copy/paste symmetry. | `.claude/skills/codex-review/SKILL.md:105` vs `:118` | Optional: align to one phrasing ("export a higher ... and re-run"). Non-blocking. |
| LOW | readability / copy-paste consistency | Case A Option 2 label is `指摘は記録し PR を作成する` (explicit "record findings" verb in the label); Case B Option 2 label is just `PR を作成する` with the recording behavior only in the English gloss. Agents parsing only the Japanese label may miss the "add to Known gaps" step for Case B. | `.claude/skills/codex-review/SKILL.md:106` vs `:119` | Optional: change Case B label to `指摘は記録し PR を作成する` for parity. Non-blocking. |
| INFO | naming | Case B header parenthetical (`cap-reached flow, Case B variant`) is slightly asymmetric with Case A's (`cap-reached flow`). Accurate and readable; no action. | `.claude/skills/codex-review/SKILL.md:102`, `:115` | None. |

No CRITICAL, HIGH, or MEDIUM findings introduced by the fix. Pre-existing cycle-1 MEDIUM (Step 3 rescan contradiction) is untouched by this commit and remains outside cycle-2 scope.

### Positive notes (cycle 2)

- Step 0.5.d's three-branch structure is the minimum sound contract: initialize / preserve / conflict-resolve. The "Inform the user of the resumed cycle number" sentence is a nice operator-visibility touch.
- The Case B cap-reached edit reuses Case A's exact question template, swapping only `要対応` → `検討推奨`. Symmetry achieved.
- Step 7 already accounted for the new cap-reached re-run path via the `(including the cap-reached ...)` parenthetical — no additional edit was needed, which is the right minimal change.
- Mirror discipline maintained across both edits; no mirror drift risk.
- No new debug code, no secrets, no swallowed errors introduced.

### Verdict — cycle 2

**Pass.** The fix commit resolves both ACTION_REQUIRED findings as described, with no new CRITICAL/HIGH/MEDIUM issues. The two LOW copy-paste consistency nits are optional polish and do not block PR. Mirror parity is intact.

---

## Cycle 3 re-review (2026-04-23, commit `12b87ee`)

- Scope: Cycle-2 Codex ACTION_REQUIRED fixes only. 5 files: `/work` SKILL (Step 0 → 0.5 → 0.7 reorder), `/codex-review` SKILL Step 7 (cap-override no-increment), triage report, plus both `templates/base/` mirrors.
- Method: `git show --stat 12b87ee`; `cmp` on both mirror pairs; `grep "Step 0\.5|Step 0\.7"` across `.claude/skills/` and `templates/` for dangling refs; inspection of Step 7's two-bullet structure against Case A/B Option 1 wording.

### Evidence reviewed

- Template parity: `cmp` PASS on both pairs (`work/SKILL.md`, `codex-review/SKILL.md` vs their `templates/base/` mirrors — byte-identical).
- Step renumbering: `0 → 0.5 → 0.7 → 1`. Step 0 now owns plan-path resolution; Step 0.5 reads "based on the plan resolved in Step 0" (:14) and uses "resolved plan" in (a) and (e) (:15, :19); Step 0.7(b) references "the Step-0 resolved absolute path" (:22). No downstream step reads the plan before Step 0.
- Dangling-reference check: no live `.claude/skills/**` or `templates/**` file references the removed `Step 0.5.d` label. Historical mentions only survive in the plan (`docs/plans/active/...:94`) and prior cycle reports, which is correct — those are historical artifacts.
- Step 7 (`codex-review/SKILL.md:125-127`) now has two explicit bullets: non-cap re-run increments; cap-reached Option 1 does NOT increment and instructs the user to `export RALPH_STANDARD_MAX_PIPELINE_CYCLES=<current cycle + 1>`. Case A Option 1 at :105 and Case B Option 1 at :118 both say "上限を一時的に引き上げて再実行" — verbatim match with Step 7's cap-override bullet. Routing is coherent.
- Triage report header updated to `Cycle: 2/2 (cap reached)` with cycle-1 findings archived as historical, cycle-2 ACTION_REQUIRED entries renumbered `#3`, `#4`. Counts consistent (`ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0`).

### Findings (cycle 3 only)

No CRITICAL, HIGH, or MEDIUM findings introduced by this commit.

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | Step 7 cap-override bullet says `<current cycle + 1>` as the env var value, which is correct for raising the cap by +1 but hides the nuance that users who want N extra passes must set `<current cycle + N>`. Rationale sentence captures the "why" but not the N-pass generalization. | `.claude/skills/codex-review/SKILL.md:127` | Optional: add "(or higher for multiple extra passes)". Non-blocking. |

### Positive notes (cycle 3)

- The three-step split (0 resolve → 0.5 branch → 0.7 pin) is the minimum sound ordering: plan selection precedes every plan-file mutation and every state-file write. Addresses cycle-2 ACTION_REQUIRED #3 at the root cause rather than patching individual operations.
- Step 7's explicit two-bullet structure (non-cap vs cap-override) eliminates the cycle-1 LOW finding that asked for enumeration of re-run paths. Both paths now name their trigger conditions.
- Cap-override rationale is inline ("incrementing here would immediately re-trip the raised cap") — future agents won't need to reconstruct the reasoning.
- Mirror discipline maintained; no new debug code, secrets, swallowed errors, or unrelated edits.

### Verdict — cycle 3

**Pass.** Both cycle-2 ACTION_REQUIRED findings are resolved at the root cause. Template parity intact, renumbering internally consistent with no dangling references, cap-override wording coherent across Case A Option 1, Case B Option 1, and Step 7. One optional LOW nit. Merge-ready from the diff-quality perspective.
