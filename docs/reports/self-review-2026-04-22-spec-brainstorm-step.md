# Self-review report: spec brainstorm step (docs/spec-brainstorm-step)

- Date: 2026-04-22
- Plan: (none — docs-only change, reviewed against the diff directly)
- Reviewer: reviewer subagent (self-review)
- Scope: Diff quality only (naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, maintainability). Spec compliance, test coverage, and doc drift are out of scope (covered by /verify, /test, /sync-docs).

## Evidence reviewed

- `git diff --stat` on branch `docs/spec-brainstorm-step` (7 files, +69 / −41):
  - `.claude/skills/spec/SKILL.md`
  - `templates/base/.claude/skills/spec/SKILL.md`
  - `AGENTS.md`
  - `CLAUDE.md`
  - `README.md`
  - `templates/base/AGENTS.md`
  - `templates/base/CLAUDE.md`
- Mirror parity verification:
  - `diff .claude/skills/spec/SKILL.md templates/base/.claude/skills/spec/SKILL.md` → byte-identical.
  - Root vs `templates/base/` divergence in `AGENTS.md` / `CLAUDE.md` was confirmed as **pre-existing** (English/Japanese split, repo-map strip-down for the distributed template). Stashing the current diff and re-running `diff` yielded the same pre-diff divergence (43 lines for `AGENTS.md`, 12 lines for `CLAUDE.md`). The diff does not widen this gap.
  - The specific line changed in both root and `templates/base/` variants of `AGENTS.md` and `CLAUDE.md` is the same semantic edit ("…via iterative brainstorming, codebase exploration, web research…").
- Step-number cross-reference audit inside `SKILL.md`:
  - Step 6 references "findings from steps 2-5" — matches new numbering (2=brainstorm, 3=explore, 4=research, 5=clarify).
  - Step 7 says "proceed to step 8" — matches.
  - Step 9 says "write only after user approval in step 7" — 承認 gate is indeed in step 7. Correct.
  - Step 2 says "distinct from step 5 which is **convergent**" — step 5 is "Clarify residual requirements (convergent)". Consistent.
- Grep across repo for stale references to the old 8-step structure or old phrase "Clarify requirements":
  - `grep -i "step 8.*spec|spec.*8 step|Clarify requirements"` → no matches.
  - The only `Understand the request` hit is the step-1 heading itself (root + templates/base), which is expected.
- `anti-bottleneck` skill referenced in the diff exists at `.claude/skills/anti-bottleneck/SKILL.md` (not renamed).
- No binary, generated, or secret-bearing files touched. No code changes. All edits are prose in Markdown.

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | Step 2 ("Brainstorm") combines "mandatory when sparse", "iteratively", "no iteration cap", and "one at a time" — a reader may interpret this as licence for runaway questioning. The stop condition ("until further questions stop yielding new information" / "until the user signals that the idea is sufficiently shaped") is present but buried two bullets below the "no iteration cap" clause. | `.claude/skills/spec/SKILL.md:33-43` | Consider promoting the stop condition to the first sentence, e.g. "Use `AskUserQuestion` iteratively (no fixed cap; stop once the user signals the idea is shaped or questions stop yielding new information) to widen the problem space …". Not blocking — current text is technically complete. |
| LOW | readability | Tension between step 2's divergent brainstorming and the Anti-bottleneck section. Step 2 correctly notes "Respect anti-bottleneck: before asking, check whether the repo already answers the question", but the listed axes (目的・背景、対象ユーザー、成功条件、既知の制約) are mostly *un*answerable from repo context, which could lead a strict reader to shortcut the whole step via anti-bottleneck. | `.claude/skills/spec/SKILL.md:33-43` vs `:105-113` | Optional: add a one-liner clarifying that divergent questions about *user intent, goals, and scope priorities* are always legitimate user-facing questions (the repo cannot answer "why does the user want this?"). Not blocking. |
| LOW | readability | Frontmatter `description` field grew from 168 to 201 characters. Still within typical skill-description budgets, but worth noting since this field is surfaced verbatim in skill listings. | `.claude/skills/spec/SKILL.md:3` (+ mirrored copy) | No action required; flagging only. |
| LOW | naming | New terms `divergent` / `convergent` are introduced only in this skill and not used elsewhere in the repo (verified via grep). They are inline-defined ("expand options" / "narrow toward a single, implementable spec"), so standalone readers will understand them. | `.claude/skills/spec/SKILL.md:17,41,63` | None — definitions are adequate. Flagging as a naming note only. |

No CRITICAL, HIGH, or MEDIUM findings.

Explicitly checked and clean:
- **Typos / copy-paste errors**: none. Step renumbering is internally consistent (verified via grep for `step 6|step 7|step 8|step 9` within the spec skill).
- **Debug code / TODO markers / commented-out blocks**: none introduced.
- **Secrets / credentials**: none — docs-only diff.
- **Null safety / exception handling / security**: N/A for prose changes.
- **Unnecessary changes**: no formatting-only churn, no unrelated files. All seven modified files map to the same semantic change (brainstorm step + step-number shift + "clarify residual requirements" rename). The AGENTS.md/CLAUDE.md/README.md updates are one line each and directly describe the new behavior.
- **Mirror discipline**: `.claude/skills/spec/SKILL.md` ↔ `templates/base/.claude/skills/spec/SKILL.md` are byte-identical. The top-level `AGENTS.md` / `CLAUDE.md` pair with their `templates/base/` mirrors received the same semantic edit on the corresponding line, preserving (not widening) the pre-existing English/Japanese divergence.

## Positive notes

- Clean docs-only diff with no accidental code, binary, or state-file inclusions.
- Mirror parity was maintained end-to-end for the one file that is required to be identical (`SKILL.md`), and the asymmetric `AGENTS.md` / `CLAUDE.md` pair received matching semantic edits without importing repo-specific text into the distributed template.
- Step 1 is correctly clarified as internal-only with an explicit "do NOT call `AskUserQuestion` here" guardrail. This is a useful defensive note that prevents the agent from asking questions before brainstorming.
- The divergent/convergent framing (step 2 vs step 5) is a genuinely useful conceptual contribution — it gives future agents a clear reason to keep the two phases separate rather than collapsing them.
- Cross-references (step 2 → "distinct from step 5", step 6 → "findings from steps 2-5", step 7 → "proceed to step 8", step 9 → "user approval in step 7") are all consistent with the new numbering.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(No tech debt introduced by this diff. Nothing to append to `docs/tech-debt/`.)_

## Recommendation

- Merge: **yes** (diff quality — no blocking findings). `/verify` and `/test` still need to sign off on spec compliance and behavioral impact respectively before PR creation.
- Follow-ups (optional, non-blocking):
  - Consider reordering step 2's bullets so the stop condition appears before "no iteration cap" to reduce the risk of runaway brainstorming.
  - Consider one clarifying sentence reconciling divergent brainstorming axes with the Anti-bottleneck section for readers who encounter them out of order.
