# Self-review report: repo-wide-drift-fixes

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-repo-wide-drift-fixes.md
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: `git diff main...HEAD` — plan commit `849f60f` + fix commit `9e39175`; 18 changed files + plan (+484/-20). Diff quality only; no spec-compliance, test-coverage, or doc-drift evaluation.

## Evidence reviewed

- Full diff of all 19 files via `git diff main...HEAD`.
- `scripts/check-skill-sync.sh` new check 6 (prompts/ parity) read in full and cross-checked against the file's existing bash style (process substitution, `<<<`, `sed "s|^$root/||"` — all pre-existing patterns).
- `tests/test-check-skill-sync.sh` cases H–K read in full; `mk_skill_pair` / `run_case` harness confirmed to isolate check 6.
- Ran `bash tests/test-check-skill-sync.sh` (priority check on whether H–K exercise failure modes): 11/11 PASS. Ran `shellcheck scripts/check-skill-sync.sh`: clean, no warnings.
- Byte-identity of every synced pair via `cmp` (adversarial-claude.md ×4, pipeline-outer.md ×4, quality-gates.md pair, ralph-loop.md pair, AGENTS.md root vs template).
- Base-detection snippet in pipeline-outer.md compared against `scripts/ralph-pipeline.sh:807`.
- tech-debt line refs verified against `scripts/ralph-cli-driver.sh` via `grep -n` / `sed -n`.
- Existence + content of `docs/roadmap/`, `docs/research/`, `docs/references/`; workflow ownership of `check-coverage.sh` / `check-pipeline-sync.sh` via `grep -rln .github/workflows/`.
- Answer to open item (c): the anchor target heading in ralph-loop.md.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | pipeline-outer.md base-detection snippet uses `sed 's\|^origin/\|\|'` (anchored) while `scripts/ralph-pipeline.sh:807` uses `sed 's\|origin/\|\|'` (unanchored). The plan (Scope 2) said "mirroring ralph-pipeline.sh". The prompt's anchored form is arguably *more correct* (only strips a leading `origin/`), so this is a benign divergence, not a bug — but it is a literal deviation from the stated "mirror" intent and could confuse a future maintainer diffing the two. | `.claude/skills/loop/prompts/pipeline-outer.md` new snippet vs `scripts/ralph-pipeline.sh:807` | Leave as-is (the anchored form is safer); optionally align `ralph-pipeline.sh` to the anchored form in a future touch. No action required for this PR. |
| LOW | typo | `tests/test-check-skill-sync.sh` header comment (lines 3–4) still says "the five drift modes (inventory, body, name, description, policy)" after a sixth mode (prompts parity) was added and exercised by new cases H–K. Stale count within the changed file. | `tests/test-check-skill-sync.sh:3-4` unchanged in the diff; cases H–K added at lines 133–164 | Update the header comment to "six drift modes … prompts parity" for consistency with `check-skill-sync.sh`'s own updated header (which correctly lists check 6). Cosmetic; non-blocking. |

## Positive notes

- **All four pipeline-outer.md copies received a byte-identical edit** (`cmp` confirms). Mirror discipline held perfectly across root/template × claude/agents.
- **New adversarial-claude.md prompt is byte-identical across all four copies** and is correctly wired into `cross-review/SKILL.md` (lines 171/180/182). No secrets or debug markers in the new prompt.
- **check-skill-sync check 6 covers both directions** (missing-in-mirror and extra-in-mirror via `comm -23` / `comm -13`) plus byte-content (`cmp -s`), exactly as the plan required. It reuses the file's established idioms (`find … | sed "s\|^$dir/\|\|" | LC_ALL=C sort`, process substitution) so it stays stylistically consistent. `-maxdepth 1 -type f` correctly scopes to top-level prompt files.
- **Tests H–K genuinely isolate check 6:** `mk_skill_pair` builds a pair that passes checks 1–5, so H (asymmetric prompts/) and I (differing content) fail *specifically* on check 6, J (identical) passes, and K exercises the real repo tree as a no-regression guard. 11/11 pass; shellcheck clean.
- **tech-debt line refs are accurate:** `pick_reviewer` is at 184 (body 184–189), `count_triage_findings` at 155 (body/awk within 155–178) — both matched. Four active→archive plan-path corrections are all correct.
- **AGENTS.md KNOWN_DIFFS removal is safe:** `cmp AGENTS.md templates/base/AGENTS.md` is byte-identical today, so removing the entry (making future divergence fail) is correct and matches the plan's conditional ("if they differ, leave it"). The removed comment did not orphan the neighboring CLAUDE.md comment.
- **quality-gates workflow annotation is correct:** `grep -rln` confirms both `check-coverage.sh` and `check-pipeline-sync.sh` run only in `.github/workflows/verify.yml`; the added `(.github/workflows/verify.yml)` on both lines matches, and the root/template pair is byte-identical.
- **repo-map new dir entries are accurate:** `docs/roadmap/` (harness-maturity-model.md), `docs/research/` (approach-comparison.md), `docs/references/` (source-notes.md) all exist with real content matching the descriptions.
- **run.go and types.go edits are text-only, precise, and accurate.** Help strings now name the env override; the OrchestratorState comment correctly reflects subset/lenient-decode semantics.

### Answer to open item (c)

**Confirmed: the heading exists in both copies.** `## Integration with the operating loop` is present at line 283 in both `docs/recipes/ralph-loop.md` and `templates/base/docs/recipes/ralph-loop.md`. The new legacy-note anchor link `#integration-with-the-operating-loop` is the correct GitHub-generated slug for that heading. No dangling anchor.

## Tech debt identified

None. Both findings are trivial cosmetic nits within the changed diff, not deferred work; neither warrants a `docs/tech-debt/` entry.

## Recommendation

- Merge: **Yes.** No CRITICAL or HIGH findings. Two LOW cosmetic notes (a benign `sed`-anchor divergence and a stale "five drift modes" comment in the test header) may be addressed inline if convenient but do not block.
- Follow-ups: (1) optionally bump the test header comment from "five" to "six drift modes"; (2) optionally align the `sed` anchor between the prompt and `ralph-pipeline.sh`.
