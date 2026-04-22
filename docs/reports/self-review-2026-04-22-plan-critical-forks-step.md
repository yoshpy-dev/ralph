# Self-review report: plan-critical-forks-step

- Date: 2026-04-22 (pass 2 — re-review after HIGH fix)
- Plan: N/A (no active plan document for this work — reviewed against the diff and the prompt context)
- Reviewer: reviewer subagent
- Scope: diff quality only for uncommitted changes on branch `docs/plan-critical-forks-step` (docs-only; +76 / -6 across 12 files)

## Evidence reviewed

- `git status` → 12 modified, 0 untracked (besides this report file itself). All paths are docs/skill guidance files, no code.
- `git diff --stat` →
  - `.claude/rules/subagent-policy.md` (+1 / -1)
  - `.claude/skills/plan/SKILL.md` (+21 / -1)
  - `.claude/skills/plan/template.md` (+5 / -0)
  - `CLAUDE.md` (+1 / -1)
  - `docs/plans/templates/feature-plan.md` (+5 / -0) **← new in pass 2**
  - `docs/plans/templates/ralph-loop-manifest.md` (+5 / -0) **← new in pass 2**
  - `templates/base/.claude/rules/subagent-policy.md` (+1 / -1)
  - `templates/base/.claude/skills/plan/SKILL.md` (+21 / -1)
  - `templates/base/.claude/skills/plan/template.md` (+5 / -0)
  - `templates/base/CLAUDE.md` (+1 / -1)
  - `templates/base/docs/plans/templates/feature-plan.md` (+5 / -0) **← new in pass 2**
  - `templates/base/docs/plans/templates/ralph-loop-manifest.md` (+5 / -0) **← new in pass 2**
- Mirror parity checks (`cmp` exit 0 = byte-identical):
  - `cmp .claude/rules/subagent-policy.md templates/base/.claude/rules/subagent-policy.md` → identical
  - `cmp .claude/skills/plan/SKILL.md templates/base/.claude/skills/plan/SKILL.md` → identical
  - `cmp .claude/skills/plan/template.md templates/base/.claude/skills/plan/template.md` → identical
  - `cmp docs/plans/templates/feature-plan.md templates/base/docs/plans/templates/feature-plan.md` → identical
  - `cmp docs/plans/templates/ralph-loop-manifest.md templates/base/docs/plans/templates/ralph-loop-manifest.md` → identical
  - `diff CLAUDE.md templates/base/CLAUDE.md` → only pre-existing intentional divergences (Japanese/English, repo-local `/release` line, `× N` phrasing). The new bullet is symmetrically translated.
- H1 resolution verification (pass 2 focus):
  - `docs/plans/templates/feature-plan.md:20-23` now contains `## Design decisions` between `## Affected areas` and `## Acceptance criteria`.
  - `docs/plans/templates/ralph-loop-manifest.md:22-25` now contains `## Design decisions` between `## Affected areas` and `## Shared-file locklist`.
  - `docs/plans/templates/ralph-loop-slice.md` deliberately untouched — per-slice files are execution units (see slice template's sections: Objective / Acceptance criteria / Affected files / Dependencies / Implementation outline / Verify plan / Test plan / Notes). The forks are plan-level decisions; recording them in slice files would duplicate and risk drift.
  - `scripts/new-feature-plan.sh:25` still copies `docs/plans/templates/feature-plan.md` (now carries the new section).
  - `scripts/new-ralph-plan.sh:67,79` still copies `ralph-loop-manifest.md` (carries section) and `ralph-loop-slice.md` (intentionally skipped).
- Tech-debt revert verification:
  - `git diff docs/tech-debt/` → empty (no pending change).
  - `grep -n "Design decisions\|critical forks\|Critical forks" docs/tech-debt/README.md` → no hits.
  - Confirmed: the entry that pass 1 added has been cleanly reverted; no residual reference.
- Cross-reference check of step 4.5's internal pointers:
  - `[template.md](template.md)` (SKILL.md:51) resolves to sibling `.claude/skills/plan/template.md` which contains the section at lines 20-23.
  - The pointer is a **sibling link**, so it cannot literally refer to `docs/plans/templates/*` — but since all four templates (fallback + 2 real + slice-exempt) consistently carry the section where applicable, the reader lands on a correct reference in every path.
  - "step 2.7" (SKILL.md:59) → exists at SKILL.md:22-28.
  - `/spec` reference → skill exists at `.claude/skills/spec/`.
- Secrets / debug-code / typo scan: no tokens, URLs, TODO markers, commented-out code, or absolute user paths introduced.
- Frontmatter character count: `description:` field on `SKILL.md:3` is 465 chars after this diff (up from 332 pre-diff). Load-bearing but wall-like — see L2.

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | The SKILL.md step 4.5 pointer `see [template.md](template.md)` is a sibling link to `.claude/skills/plan/template.md`, which the plan-creation scripts do **not** actually copy — they copy `docs/plans/templates/feature-plan.md` or `ralph-loop-manifest.md`. Functionally harmless now (all four templates carry the section, so any template the reader or model lands on has the anchor), but the pointer is technically a ghost reference: it names a fallback file, not the live one. Downgraded from HIGH in pass 1 because the user-facing templates now carry the section — the direction mismatch no longer causes the model to lose its landing site. | `SKILL.md:51` (sibling link), `scripts/new-feature-plan.sh:25`, `scripts/new-ralph-plan.sh:67` | Optional: reword to `record in the plan's "Design decisions" section (present in all plan templates under docs/plans/templates/ and .claude/skills/plan/template.md)` on a future unrelated edit. Not merge-blocking. |
| LOW | readability | Step 4.5's sub-numbering (`a.`, `b.`, `c.`) mixes with the top-level numeric scheme (4, 4.5, 5, 6, 6.5, 7) consistently with step 2.5/2.7, but the "For each critical fork identified" block is indented with 3 spaces while some surrounding bullets in other steps use 5. Renderers handle both. | SKILL.md:49-52 vs SKILL.md:69-78 | Acceptable — leave as-is. Only worth normalizing if a future diff already touches neighboring steps. |
| LOW | readability | `description:` frontmatter on `SKILL.md:3` is now 465 chars (up from 332). Each addition is load-bearing, but the line is becoming a wall for humans and for model context that skims YAML. | `.claude/skills/plan/SKILL.md:3` | No change now. Consider splitting into a shorter summary + `long_description:` on the next unrelated edit. |
| LOW | typo | "choose informedly" reads awkwardly; "make an informed choice" is more idiomatic. Not mis-spelled. | SKILL.md:50 | Optional reword. Not merge-blocking. |
| LOW | readability | The two newly-added `Design decisions` sections use bilingual HTML comments (`判断・採用した選択肢・理由（rationale）`). This matches the existing bilingual style elsewhere in the repo (e.g., CLAUDE.md mix of Japanese and English), but the sibling fallback `template.md` uses the same comment verbatim — good consistency. No action needed; noting it so future translators know the strings are duplicated across all four templates. | `docs/plans/templates/feature-plan.md:22-23`, `docs/plans/templates/ralph-loop-manifest.md:24-25`, `.claude/skills/plan/template.md:22-23` (+ three `templates/base/` mirrors) | Keep as-is. If ever translating, grep all six locations together. |

## Positive notes

- **H1 from pass 1 is fully resolved.** The `## Design decisions` section now exists in both user-facing plan templates (`docs/plans/templates/feature-plan.md` and `ralph-loop-manifest.md`) in the correct position — between `## Affected areas` and `## Acceptance criteria` / `## Shared-file locklist`. Section placement matches the natural "decisions precede commitments" flow of a plan.
- **Slice-template exemption is well-reasoned and verified against the slice's actual structure.** `ralph-loop-slice.md` has Objective / Acceptance criteria / Affected files / Dependencies / Implementation outline / Verify plan / Test plan / Notes — no natural home for plan-level forks. Adding a Design-decisions section there would either (a) duplicate the manifest's record or (b) create drift between slice copies. Correct call to exempt.
- **Mirror discipline is perfect across all 6 pairs** (root ↔ `templates/base/`): `cmp` exits 0 for every pair. Every new file in this diff has a correctly-placed mirror.
- **Tech-debt revert is clean.** `git diff docs/tech-debt/` is empty and `grep` finds no lingering reference. Pass 1's "self-blocking by adding then satisfying its own tech-debt entry" concern is gone.
- **Cross-references between the edited files remain coherent.** CLAUDE.md → points to critical forks; SKILL.md 4.5 → points to template.md; template.md has the section; user-facing templates have the section; subagent-policy.md describes the new AskUserQuestion usage. Every pointer resolves.
- **Diff remains docs-only.** Zero code, zero scripts, zero runtime surface. No regression risk from this surface alone.
- **Scope discipline in pass 2 fix.** Only the 4 template-related paths were added — no drive-by edits to unrelated sections of the plan templates, no reformatting, no rewording of adjacent sections. This is the minimal viable fix for H1.
- **Section-placement choice is defensible.** Placing "Design decisions" *before* "Acceptance criteria" / "Shared-file locklist" means criteria can reference decided options (e.g., "Acceptance criterion: X uses approach A from Design decisions"), rather than the reverse which would require forward references.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(No new tech debt from pass 2. The pass-1 debt item was resolved in-PR — the `docs/plans/templates/` additions make `/plan` step 4.5 executable without template scaffolding follow-ups.)_

## Recommendation

- Merge: **YES — ready to merge.** Pass-1 HIGH (H1) is fully resolved in the simplest, highest-leverage way (edit the two user-facing templates and their mirrors, leave slice template exempt for correct design reasons).
- Verdict summary: **0 CRITICAL. 0 HIGH. 0 MEDIUM. 5 LOW** (all advisory, none merge-blocking).
- The LOWs from pass 1 that are still present (`informedly`, 463→465-char frontmatter, indent mix) are truly stylistic — worth picking up on a future unrelated edit, not on this PR.
- New LOW (L1) observes that `SKILL.md:51`'s sibling link still points at the fallback `template.md`; safe because every actual template now carries the section, but worth a reword on the next unrelated edit for precision.
- Follow-ups (all non-blocking):
  - Optional SKILL.md:51 reword to name all template locations explicitly.
  - Optional SKILL.md:50 reword ("informedly" → "make an informed choice").
  - Watch the `description:` frontmatter length on the next edit.
  - If a future plan PR translates the bilingual comments, grep all 6 `Design decisions` sections to update them together.
