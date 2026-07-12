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

---

## Cycle 2 addendum

- Date: 2026-07-12
- Scope: new commits since cycle-1 MERGE verdict — `e11a49b` (sync-docs: Five→Six checks + codex-setup axes), `924d45b` (checklist tick), `501d164` (cross-review fixes: symbolic-ref base detection ×4 + recursive prompts-parity gate + test cases L/M + triage report). Reviewed via `git show 501d164 e11a49b`. Diff quality only.

### Evidence reviewed (cycle 2)

- **Symbolic-ref fallback (fresh clone, `origin/HEAD` unset):** `git init` throwaway repo → `git symbolic-ref --short refs/remotes/origin/HEAD` exits 128, stderr suppressed, stdout empty → `${_base:-main}` yields `main`. Fallback covers the fresh-clone / no-remote-HEAD case correctly.
- **Symbolic-ref on a feature branch (the P2 bug scenario):** synthetic repo with `origin/HEAD → origin/main`, checked out `feature/foo` → snippet resolves `_base=main` (not `feature/foo`). Confirms the fix defeats the original `HEAD@{upstream}` bug where the base collapsed to the same branch and made `git diff base...HEAD` empty.
- **4-copy byte-identity:** `cmp` confirms `.claude/skills/loop/prompts/pipeline-outer.md` == `.agents/...` == `templates/base/.claude/...` == `templates/base/.agents/...`. `check-skill-sync.sh` == `templates/base/scripts/check-skill-sync.sh`. All identical.
- **Recursive find correctness:** reproduced the old-vs-new logic on a fixture with a nested-only file (`cl/prompts/sub/x.md`, absent on `cx`). Old `-maxdepth 1`: `comm -23` yields empty → the drift is invisible (test would falsely pass). New recursive: `comm -23` yields `sub/x.md` → drift caught. Relative-path comparison via `sed "s\|^$dir/\|\|"` produces `sub/x.md` on both sides, so paths sort/compare correctly.
- **No false positive on empty dirs:** both sides empty → `find` returns nothing → `echo "$var"` emits one blank line on each side → `comm` sees them equal → `only_cl`/`only_cx` empty. The downstream `[ -z "$f" ] && continue` guard skips the blank line in the byte-compare loop.
- **Byte-compare loop uses relative path correctly:** `cmp -s "$cl_prompts/$f" "$cx_prompts/$f"` with `$f="sub/x.md"` reconstructs the correct nested absolute path on both sides.
- **Test cases L/M bite:** ran full suite → 13/13 PASS. L asserts exit 1 for a nested-only file; confirmed it would have FAILED against the old `-maxdepth 1` code (which misses the file) and PASSES against the recursive code — a genuine regression test, not a tautology. M asserts exit 0 for a nested byte-identical pair.

### New findings (cycle 2)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| HIGH (follow-up, not a diff defect) | maintainability / correctness | The P2 base-detection fix was applied to the 4 prompt copies but **not** to the two production call sites that actually gate cross-review with the same `HEAD@{upstream}` construct. `scripts/ralph-pipeline.sh:807` (and its `templates/base/` mirror) still does `_base="$(git rev-parse --abbrev-ref 'HEAD@{upstream}' … )"` and then gates cross-review on `git diff "${_base}...HEAD" --quiet` (line 809). On any pushed feature branch the upstream ref resolves to `origin/<same-branch>`, the diff is empty, and **cross-review is silently skipped** — the exact failure the P2 finding described, still live where it matters most. `.claude/skills/cross-review/SKILL.md:51` (+ `.agents/` mirror) has the same `HEAD@{upstream}` construct for the standard-flow base. The repo now has *inconsistent* base detection: `scripts/ralph:490` and `scripts/ralph-worktree.sh:85` already use the correct `git symbolic-ref … refs/remotes/origin/HEAD` form. | `grep -rn "HEAD@{upstream}\|symbolic-ref"` across `scripts/` + skills shows 4 buggy `HEAD@{upstream}` sites (ralph-pipeline.sh ×2 via mirror, cross-review/SKILL.md ×2 via mirror) vs 4 correct `symbolic-ref` sites (ralph, ralph-worktree, pipeline-outer.md ×2 via mirror). | Not a defect *introduced* by this diff — the diff strictly improves the prompt copies — so it does not block this PR. But file a HIGH follow-up (or tech-debt entry) to port the `symbolic-ref` form to `ralph-pipeline.sh:807` and `cross-review/SKILL.md:51` (+ both mirrors), since those are the sites that actually run the cross-review gate. Without it, the prompt now documents a base-detection strategy that diverges from the shell that executes it. |
| LOW | typo | `tests/test-check-skill-sync.sh:3-4` header now reads "the six drift modes (inventory, body, name, description, policy)" — the count was bumped Five→Six but the parenthetical still lists only five modes; **prompts parity** is missing from the enumerated list. Half-fix of the cycle-1 LOW nit (count updated, list not). | `tests/test-check-skill-sync.sh:3-4` | Append "prompts parity" to the parenthetical so the list matches the count and `check-skill-sync.sh`'s own six-check header. Cosmetic; non-blocking. |

### Cycle-1 LOW follow-ups status

- **`sed` anchor divergence (cycle-1 LOW):** now superseded — the prompt copies switched base commands entirely (`symbolic-ref` with anchored `s\|^origin/\|\|`), while `ralph-pipeline.sh:807` kept `HEAD@{upstream}` with unanchored `s\|origin/\|\|`. This has escalated from a benign `sed`-anchor nit to the HIGH strategy-divergence follow-up above.
- **Test-header "five drift modes" (cycle-1 LOW):** partially addressed (count is now "six") but the mode list is still incomplete — see the LOW typo above.

### Cycle 2 verdict

- **Merge: Yes.** No CRITICAL findings. The two `501d164` fixes are correct: symbolic-ref resolves the base to the repo default branch on feature branches and falls back to `main` on fresh clones; the recursive prompts-parity gate catches nested drift, avoids empty-dir false positives, and is guarded by genuine regression tests (L/M bite). All 13 test cases pass; 4-copy + mirror byte-identity holds.
- **One HIGH follow-up** (not a defect in this diff): port the same base-detection fix to `scripts/ralph-pipeline.sh:807` and `.claude/skills/cross-review/SKILL.md:51` (+ mirrors), the two sites that actually gate cross-review. Recorded as tech debt.
- **One LOW:** complete the test-header mode list (add "prompts parity").
