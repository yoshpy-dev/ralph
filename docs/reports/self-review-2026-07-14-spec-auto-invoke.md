# Self-review report: spec-auto-invoke

- Date: 2026-07-14
- Plan: docs/plans/active/2026-07-14-spec-auto-invoke.md
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: `git diff 540771c...HEAD` — diff quality only (naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, maintainability). Spec compliance, test coverage, and doc drift are out of scope (handled by /verify and /test).

## Implementation-context note (fallback)

Slice 2 (documentation reflection) was implemented by an `implementer` subagent whose
session terminated on an API error after all edits were completed. Per
`.claude/rules/subagent-policy.md`, the orchestrator adjudicated the produced diff and
committed it inline as a dispatch-failure fallback. This review therefore evaluates the
committed result rather than a subagent-returned report. The diff was inspected in full;
no evidence of a truncated or partial edit was found (see Evidence reviewed).

## Evidence reviewed

- `git diff --stat 540771c...HEAD` — 14 files changed, 2 files deleted, no unexpected paths.
- Frontmatter diff of all four `spec/SKILL.md` copies (root + templates/base × claude + agents mirror).
- Deletion of `.agents/skills/spec/agents/openai.yaml` and `templates/base/.agents/skills/spec/agents/openai.yaml` (the only `--diff-filter=D` entries).
- Doc changes in `CLAUDE.md`, `AGENTS.md`, `README.md`, `docs/architecture/repo-map.md`, `templates/base/CLAUDE.md`, `templates/base/AGENTS.md`.
- New regression case G in `tests/test-sync-skills.sh` (lines 192-231) and its trap wiring (lines 196, 237).
- `scripts/sync-skills.sh:143-156` — the openai.yaml create/remove branch exercised by case G.
- Mirror byte-identity: `cmp .claude/skills/spec/SKILL.md templates/base/.claude/skills/spec/SKILL.md` → identical; same for the agents mirror.
- Mojibake scan (U+FFFD) across changed markdown → none found. The em-dash `—` in the new description is intentional and consistent with surrounding docs.
- Residual "spec + manual" grep across `*.md` → only legitimate references remain (`/release` stays manual; README/CLAUDE.md describe spec as *no longer* manual; historical spec docs).

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | New test case G is placed physically **before** case F, but the header comment block lists F first and appends G after it. Execution order is unambiguous (labels + traps are correct) but the source ordering (A-E, G, F) is mildly surprising for a future reader scanning top-to-bottom. | `tests/test-sync-skills.sh:11-16` (comment lists A-F then G) vs `:192` (G) preceding `:233` (F) | Optional: either move case G after F to match the comment order, or reorder the comment to A-E, G, F. Non-blocking. |
| LOW | maintainability | The intermediate trap at line 196 (`... "$E_DIR" "$G_DIR"`) is immediately superseded by the final trap at line 237 which adds the snapshot dirs. The line-196 trap is only load-bearing if the script aborts between 196 and 237. This matches the existing incremental-trap idiom (each new `mktemp -d` extends the trap), so it is consistent, just worth noting the redundancy is intentional. | `tests/test-sync-skills.sh:196,237` | No change needed; consistent with cases A-E. |

## Positive notes

- **The regression test targets the exact branch this PR relies on.** Case G removes `disable-model-invocation` and re-syncs, asserting both `openai.yaml` deletion and empty `agents/` dir cleanup (`sync-skills.sh:151-155`). This is the very cleanup path that produced the two file deletions in this diff, so the test is a true guard rather than a coincidental exercise. Good instinct addressing Codex advisory #1.
- **Description rewrite includes both positive and negative trigger conditions** ("Invoke when … too vague or abstract for /plan"; "Do not invoke for reviews, Q&A … trivial fixes, or when the user explicitly requests another skill"). Since the frontmatter `description` is the model's auto-invocation signal, spelling out the negative space is the right defense against over-firing.
- **Mirror discipline is intact.** Root and `templates/base` copies are byte-identical for both SKILL.md variants; the two openai.yaml deletions are symmetric across root and template. No half-applied mirror edit.
- **No stray or contradictory leftover docs.** Every "manual" reference that survives is correct (`/release`, or historical spec docs, or sentences that explicitly describe spec as now auto-invoked).
- **No debug code, secrets, injection surface, or swallowed errors introduced.** The `rmdir ... 2>/dev/null || true` in `sync-skills.sh` is pre-existing and correctly tolerates a non-empty dir.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| _(none)_ | | | | |

No deferred work, shortcuts, or accumulated complexity identified. Nothing appended to `docs/tech-debt/`.

## Recommendation

- Merge: **Yes** (merge). No CRITICAL or HIGH findings. The two LOW findings are cosmetic (test-case source ordering and a benign redundant trap) and do not block.
- Follow-ups:
  - Optional: reconcile the case G vs case F source ordering with the header comment (LOW).
- Known gaps:
  - This review is diff-quality only. Spec-compliance grep assertions (acceptance criteria), `check-skill-sync.sh` / `check-sync.sh` / `run-verify.sh` execution, and the Go embed test are the responsibility of /verify and /test and were not run here.
  - Effective auto-invocation behavior (whether the model actually fires `/spec` at the intended threshold) is a runtime/behavioral property not observable from a static diff.
