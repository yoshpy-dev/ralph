# Self-review report: standard-flow-orchestrator

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-standard-flow-orchestrator.md
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: `git diff main...HEAD` — docs/agent-definition-only diff (no code). Naming, readability, consistency across the 4-copy mirrors, unnecessary changes, typos, misleading statements, maintainability. Spec compliance and test execution explicitly out of scope (deferred to /verify and /test).

## Evidence reviewed

- Plan read in full (scope, non-goals, design decisions, AC1b deviation, Design decision 3 merge-conflict constraint).
- Full diff of all 17 changed files via `git diff main...HEAD`.
- 4-copy mirror integrity verified with `cmp`:
  - `.claude/agents/implementer.md` ↔ `templates/base/.claude/agents/implementer.md` — IDENTICAL
  - `.codex/agents/implementer.toml` ↔ `templates/base/.codex/agents/implementer.toml` — IDENTICAL
  - work SKILL.md: `.claude` ↔ `.agents` ↔ `templates/base` — IDENTICAL
  - cross-review SKILL.md: `.claude` ↔ `.agents` — IDENTICAL
  - subagent-policy.md root ↔ template — IDENTICAL
  - model-routing.md root ↔ template — differs only by the pre-existing `internal/config/config.go` KNOWN_DIFF block (registered in `scripts/check-sync.sh` line 95); the new "Standard flow delegation" section is present in BOTH copies.
- AC3 grep: `grep -rn 'claude -p --model opus' .claude .agents templates/base` → 0 hits (no residual hardcode).
- AC4 grep: `grep -l 'Standard flow delegation' .claude/rules/model-routing.md templates/base/.claude/rules/model-routing.md` → both files hit.
- Encoding: no U+FFFD / mojibake in any new file.
- Frontmatter shape of all existing agents compared against implementer.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | `implementer.md` frontmatter omits the `skills:` field that every other agent carries. reviewer→`self-review`, verifier→`verify`, tester→`test`, doc-maintainer→`sync-docs` all map to a dedicated skill; `implementer` is the only agent without one. This breaks the uniform frontmatter shape and could confuse future maintainers grepping for `skills:` to enumerate agent→skill wiring. | `grep -L 'skills:' .claude/agents/*.md` → only `implementer.md`. All four other agents have a `skills:` block. | Intentional-if: implementer's guidance lives in `work/SKILL.md`, not a dedicated skill, so there is no skill to reference. Acceptable as-is, but consider a one-line comment or explicitly documenting in the plan/subagent-policy that implementer is skill-less by design, so the omission does not read as an oversight. Non-blocking. |
| LOW | readability | The cross-review parenthetical is dense: "...at the end of this file). (the variable is read from the environment — e.g. exported by `ralph run` or sourced from `scripts/ralph-config.sh`; unset falls back to `opus`)". Two adjacent parenthetical clauses ("(see prompt template...)" then "(the variable...)") sit back-to-back and read awkwardly. | `.claude/skills/cross-review/SKILL.md` line ~55 (and 3 mirror copies). | Optional: fold into a single clause or move the env-var note to its own sentence. Content is correct and the narrow-scoping matches Codex finding 3; purely stylistic. Non-blocking. |
| LOW | consistency | Field-name drift between the two rule files for the same handoff contract. `model-routing.md` table row reads "Acceptance criteria — the ACs this slice addresses"; the work SKILL.md prose and subagent-policy.md say "acceptance criteria addressed". Minor, but the canonical field label differs slightly across the three locations describing one contract. | model-routing.md handoff table vs. subagent-policy.md prose vs. work SKILL.md step 6. | Optional: pick one label ("acceptance criteria addressed") everywhere so a reader diffing the three does not wonder whether they describe different fields. Non-blocking. |

## Positive notes

- **Mirror discipline is clean.** All four copies of each shared file are byte-identical (or KNOWN_DIFF-only for model-routing.md). The `chmod`/mode trap does not apply here (no scripts), and the KNOWN_DIFF boundary was respected exactly — the new section landed in both model-routing.md copies, only the pre-existing Go-layer block remains root-only.
- **Design decision 3 (merge-conflict avoidance vs PR #115) honored precisely.** The new section in model-routing.md is inserted after the tier-table paragraph and before `## Rules`; "Where the values live" is untouched. subagent-policy.md's new section sits after the pipeline `### Fallback` block, not in the /loop region. Disjoint hunks as planned.
- **AC1b deviation is well-recorded** in the plan (dispatch-failure fallback exercised instead of live dispatch; runtime smoke recorded as a known gap for both CLIs). Not a diff-quality issue; flagged here only to confirm it was reviewed and is intentional.
- **No secrets, no debug code, no swallowed errors.** The implementer prompt actively hardens against bad commit hygiene: mandatory `git status --porcelain` baseline check, explicit ban on `git add -A`/`-u`/`.`, "never commit failing state", "never weaken tests". Strong maintainability posture.
- **`${RALPH_CLAUDE_REVIEWER_MODEL:-opus}` fallback is safe** — unset preserves today's `opus` behavior exactly; the claim in the skill text is correctly narrowed to "read from the environment", not "reads ralph.toml".
- **`.md` vs `.toml` parity is faithful.** The Codex `developer_instructions` mirror the Markdown prompt in content, with only formatting/wrapping differences appropriate to each format (e.g. bold markers dropped in TOML, code block converted to indented line). No semantic drift.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| — | — | — | — | — |

_No new tech debt. The AC1b runtime-dispatch smoke gap is already tracked in the plan and slated for /test + PR body per the plan's own contract; no separate register entry needed._

## Recommendation

- Merge: YES (no CRITICAL or HIGH findings). One MEDIUM consistency observation (`skills:` frontmatter omission) and two LOW stylistic notes; all optional and non-blocking.
- Follow-ups: (1) decide whether implementer's skill-less frontmatter should be documented as intentional; (2) optional: unify the "acceptance criteria addressed" field label across the three contract locations; (3) optional: de-nest the cross-review env-var parenthetical.

## Cycle 2 addendum

- Date: 2026-07-11
- Trigger: pipeline cycle 2/2 re-run after cross-review ACTION_REQUIRED fixes (see `docs/reports/cross-review-triage-standard-flow-orchestrator.md`).
- New commits reviewed: `a910547` (baseline-rule narrowing + two-mode validation gate) and `1de7a20` (git-commit-strategy delegated-slice note).
- Scope unchanged: diff quality only; still a docs/agent-definition-only diff, no code.

### Evidence reviewed (cycle 2)

- `git show a910547 1de7a20` read in full.
- 4-copy mirror integrity re-verified with `cmp` on every file the two commits touched:
  - `.claude/agents/implementer.md` ↔ `templates/base/.claude/agents/implementer.md` — IDENTICAL
  - `.codex/agents/implementer.toml` ↔ `templates/base/.codex/agents/implementer.toml` — IDENTICAL
  - work SKILL.md: `.claude` ↔ `.agents` ↔ `templates/base/.claude` ↔ `templates/base/.agents` — all IDENTICAL
  - `.claude/rules/git-commit-strategy.md` ↔ template copy — IDENTICAL
- Mojibake/U+FFFD scan across all `main...HEAD` changed files — clean.
- Working tree clean (`git status --porcelain` empty); no stray edits.

### Three-way coherence check (implementer.md ↔ work SKILL.md ↔ git-commit-strategy.md)

Verified the reworded baseline rule and two-mode gate are mutually consistent, with no contradiction:

- **implementer.md / .toml (baseline rule):** run `git status --porcelain`; pre-existing mods OUTSIDE files-in-scope are normal (note, never stage, proceed); STOP only when dirt OVERLAPS files-in-scope. This resolves cross-review finding 1 (bookkeeping dirt no longer blocks dispatch) while preserving the anti-absorption guarantee for in-scope overlap.
- **work SKILL.md step 6:** orchestrator commits outstanding plan/bookkeeping edits "or confirm they do not overlap the slice's files in scope" before dispatch — the exact mirror image of the implementer's STOP condition. Producer/consumer symmetry is clean: what the orchestrator promises not to leave overlapping is precisely what the implementer refuses to absorb.
- **work SKILL.md step 7 (two-mode gate):** delegated → adjudicate report + confirm SHA (`git log -1 <sha>`) + optional spot-check, no re-stage/re-commit; inline → original verify→stage→commit. Resolves finding 2 (double-commit). The "one slice = one commit, owned by the implementer" invariant is stated explicitly and matches step 6's dispatch precondition.
- **git-commit-strategy.md (delegated-slice note):** restates the same ownership boundary (implementer verifies + owns commit; orchestrator adjudicates, does not re-stage/re-commit) and cross-links to SKILL.md step 7. No divergence from the SKILL wording; the earlier gap where git-commit-strategy.md described only the inline path is now closed. This resolves the cycle-1 concern that a reader landing on git-commit-strategy.md first would infer the orchestrator always commits.

### Cycle 2 findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| — | — | No new findings. | Two-mode step 7 reads coherently; baseline rule reworded identically across all 4 copies; no contradiction between implementer.md, work SKILL.md, and git-commit-strategy.md. | None. |

The three cycle-1 observations (1 MEDIUM `skills:` omission, 2 LOW stylistic) are untouched by these commits and remain as previously adjudicated (non-blocking).

### Cycle 2 verdict

- Merge: YES. No CRITICAL, no HIGH, no new findings. The cross-review ACTION_REQUIRED fixes are correctly and consistently applied across all mirror copies; the cycle-1 MERGE verdict still holds for the full diff.
- CRITICAL: none.
