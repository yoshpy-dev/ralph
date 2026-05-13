# Self-review report: fix-codex-review-wording-residues (issue #51)

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-fix-codex-review-wording-residues.md`
- Reviewer: `reviewer` subagent (Claude Code, Opus 4.7 1M)
- Scope: diff against `main` (commit `8f1bd82`) — 6 changed source files (3 top-level + 3 `templates/base/` mirrors) + 1 new plan file

## Evidence reviewed

- `git diff main...HEAD --stat` — 7 files, +124 / -10 LOC (114 of which are the new plan file)
- Full `git diff main...HEAD` — string-only edits, no logic changes
- `cmp` of all three mirror pairs:
  - `.claude/skills/loop/prompts/pipeline-outer.md` ↔ `templates/base/.claude/skills/loop/prompts/pipeline-outer.md` — byte-identical
  - `.agents/skills/loop/prompts/pipeline-outer.md` ↔ `templates/base/.agents/skills/loop/prompts/pipeline-outer.md` — byte-identical
  - `docs/recipes/ralph-loop.md` ↔ `templates/base/docs/recipes/ralph-loop.md` — byte-identical
- AC-5 allowlist grep `git grep -nE 'codex[- ]review|codex ACTION_REQUIRED' -- ':!docs/plans/archive' ':!docs/reports' ':!docs/recipes/codex-setup.md' ':!templates/base/docs/recipes/codex-setup.md' ':!docs/specs/2026-05-07-codex-cli-parity.md' ':!docs/tech-debt/README.md' ':!docs/plans/active/2026-05-13-fix-codex-review-wording-residues.md'` — empty
- Case-insensitive sweep including `/codex-review` and `Codex-review` — one residual hit in `docs/evidence/colorize-upgrade-diff-2026-04-23-nocolor.txt` (frozen evidence capture; see Finding F-1)
- Code/script sweep over `scripts/`, `.claude/hooks/`, `cmd/`, `internal/`, `packs/`, `templates/` — no source/script references to the old wording (only the allowlisted `codex-setup.md` rename narrative)
- Commit message `8f1bd82` — Conventional Commits format, accurate scope, no leaked credentials, ends with `Closes #51`

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | unnecessary-change | Plan progress checklist marks `Implementation started` but leaves `Review artifact created` / `Verification artifact created` / `Test artifact created` / `PR created` unchecked. Status header says "Implementation complete (awaiting post-implementation pipeline)" which is consistent with the checklist, so this is not a defect — just calling out that follow-on pipeline steps will need to tick these boxes via `/sync-docs`. | `docs/plans/active/2026-05-13-fix-codex-review-wording-residues.md:108-114` | None — expected for a plan still in flight. Mention so the doc-maintainer pass updates the checkboxes. |
| LOW | maintainability | `docs/evidence/colorize-upgrade-diff-2026-04-23-nocolor.txt:37` still contains `Codex-review (auto, optional — cross-model second opinion)`. This file is a frozen 2026-04-23 capture of `ralph upgrade` diff output (an evidence artifact, conceptually identical to `docs/reports/` and `docs/plans/archive/`). The plan's AC-5 allowlist does not enumerate `docs/evidence/`, but rewriting it would invalidate the evidence. | `docs/evidence/colorize-upgrade-diff-2026-04-23-nocolor.txt:37` and `docs/evidence/README.md` (defines evidence as "anything stronger than assertion" — i.e., immutable captures) | Do not rewrite the file. Consider extending the AC-5 allowlist to include `docs/evidence/` for future similar cleanups so the same kind of fossilized hit is not flagged repeatedly. Optional follow-up, not a blocker for this PR. |

No CRITICAL, HIGH, or MEDIUM findings.

## Positive notes

- Diff is mechanically minimal: only the four lines flagged in the issue are changed, plus their three `templates/base/` mirrors. No incidental whitespace, formatting, or unrelated edits.
- Top-level ↔ `templates/base/` byte-identity is preserved (verified with `cmp`), matching the mirror discipline pattern recorded in agent memory.
- Replacement is grep-stable: `codex review` (free phrase) → `cross-review` (hyphenated, the canonical command name in 3.5.0+) aligns with how the term is used elsewhere in the same files.
- Plan explicitly enumerates an exclusion allowlist for AC-5 and parameterizes the verification grep accordingly — addresses the Codex plan-advisory HIGH finding cleanly.
- Commit message uses Conventional Commits, describes intent and out-of-scope items, and does not contain backticks/`$(...)` inside double quotes (no `commit-msg-guard.sh` risk).
- No secrets, debug code, exception handling, null safety, or security concerns are relevant to this diff (documentation strings only).
- No behavior change: scripts, hooks, Go code, configs, and tests are untouched.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `docs/evidence/` not in standard cleanup allowlist | Future renames will keep hitting the same fossilized evidence captures during verification grep | Out of scope for this string-only PR; touching evidence files invalidates them | Next codebase-wide string rename that surfaces evidence-file hits | (this report) |

No new entries appended to `docs/tech-debt/README.md` — the item above is small enough that it can be solved by a one-line allowlist edit on the next cleanup task; recording it here is sufficient.

## Recommendation

- Merge: yes (after `/verify` and `/test` confirm). Diff is a clean, scoped, string-only cleanup with mirror parity verified.
- Follow-ups:
  - Doc-maintainer pass should tick the remaining `Progress checklist` boxes in the plan file once the pipeline artifacts (verify, test, PR) exist.
  - Optional, low-priority: in the next codex/cross-review wording sweep, add `docs/evidence/` to the AC allowlist so frozen evidence captures stop appearing in case-insensitive sweeps.
