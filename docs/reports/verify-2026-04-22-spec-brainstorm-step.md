# Verify report: spec brainstorm step (docs/spec-brainstorm-step)

- Date: 2026-04-22
- Plan: (none — docs-only change; acceptance criteria supplied in-channel by the user and echoed here)
- Verifier: verifier subagent (/verify)
- Scope: Spec compliance + static analysis + documentation drift for the 7-file docs-only diff on `docs/spec-brainstorm-step`. Diff quality covered by `/self-review` (`docs/reports/self-review-2026-04-22-spec-brainstorm-step.md`). Behavioral tests out of scope.
- Evidence: `docs/evidence/verify-2026-04-22-spec-brainstorm-step.log` (static-verify run), `docs/evidence/verify-2026-04-22-065413.log` + `docs/evidence/verify-2026-04-22-065937.log` (prior `run-verify.sh` / `run-static-verify.sh` runs, identical OK outcome).

## Spec compliance

Acceptance criteria were provided inline (no `docs/plans/active/` plan). Each criterion is walked against the diff.

| # | Acceptance criterion | Status | Evidence |
| --- | --- | --- | --- |
| AC1 | `/spec` SKILL.md contains a Step 2 that uses `AskUserQuestion` iteratively for idea expansion | Verified | `.claude/skills/spec/SKILL.md:33` — `2. **Brainstorm to expand the idea** (壁打ち): Use `AskUserQuestion` iteratively to widen the problem space before converging.` Followed by 6 divergent-axis bullets, explicit "no iteration cap" + stop condition, and "divergent vs convergent" framing (line 41). |
| AC2 | Step 1 explicitly excludes `AskUserQuestion` (internal triage) | Verified | `.claude/skills/spec/SKILL.md:31` — `1. **Understand the request** (internal, no user interaction): … This is an internal triage step — do NOT call `AskUserQuestion` here. The list feeds the next step.` Guardrail is stated in-sentence, not merely implied. |
| AC3 | Step numbering is self-consistent throughout the document (cross-refs match) | Verified | `grep '^[0-9]+\. \*\*'` on `SKILL.md` returns 9 top-level steps in strict 1→9 order with no gaps. Cross-refs audited: (a) Step 2 L41 "distinct from step 5 which is **convergent**" → matches Step 5 heading "Clarify residual requirements" + "Purpose is **convergent**" (L63). (b) Step 6 L66 "findings from steps 2-5" → 2=Brainstorm, 3=Explore, 4=Research, 5=Clarify, correct inclusive span. (c) Step 7 L80 "proceed to step 8" → Step 8 is "Output selection", correct. (d) Step 9 L90 "write only after user approval in step 7" → Step 7 is the approval gate, correct. No stray references to old step numbers. |
| AC4 | Template mirror (`templates/base/.claude/skills/spec/SKILL.md`) is byte-identical to repo-root version | Verified | `diff .claude/skills/spec/SKILL.md templates/base/.claude/skills/spec/SKILL.md` → no output; follow-up test emits `BYTE-IDENTICAL`. Also confirmed transitively by `scripts/check-sync.sh` (IDENTICAL: 107 / DRIFTED: 0). |
| AC5 | `CLAUDE.md`, `AGENTS.md`, `README.md`, and their template mirrors all mention "iterative brainstorming" (or 壁打ち) in their `/spec` descriptions | Verified, with scoping note | `CLAUDE.md:10`, `AGENTS.md:22`, `README.md:131`, `templates/base/CLAUDE.md:10`, `templates/base/AGENTS.md:19` all contain "iterative brainstorming (壁打ち)" or "iterative brainstorming" in their `/spec` description. Scoping note: `templates/base/README.md` does not exist in this repo (verified via `ls`; `README.md` is root-only and intentionally not mirrored). AC5 therefore covers 5 of the 5 applicable files. |

All 5 acceptance criteria are verified.

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` (re-run 2026-04-22T06:59:37Z) | PASS (exit 0) | shellcheck on hooks + verify scripts: OK. `sh -n` on all 9 root hooks + 9 `templates/base/` mirrors: OK. `jq -e .` on `.claude/settings.json` + `templates/base/.claude/settings.json`: OK. `scripts/check-sync.sh`: IDENTICAL=107 / DRIFTED=0 / ROOT_ONLY=0 / TEMPLATE_ONLY=9 / KNOWN_DIFF=3 (the 3 KNOWN_DIFF files are the pre-existing English/Japanese split `AGENTS.md`, `CLAUDE.md`, and `.github/workflows/verify.yml`; untouched by this diff in terms of structure). gofmt: ok; golangci-lint: 0 issues; `go test ./...`: all packages OK (cached). |
| `./scripts/run-verify.sh` (prior run, slug-less) | PASS | `docs/evidence/verify-2026-04-22-065413.log` (sync check, mojibake check, Go tests all PASS per user-supplied context). |
| `diff .claude/skills/spec/SKILL.md templates/base/.claude/skills/spec/SKILL.md` | PASS | Byte-identical (no output). |
| `grep '^[0-9]+\. \*\*' .claude/skills/spec/SKILL.md` | PASS | 9 steps, strict monotonic numbering, no duplicates/gaps. |

Full static-verify transcript saved to `docs/evidence/verify-2026-04-22-spec-brainstorm-step.log` (396 lines).

## Documentation drift

Methodology: grep the repo for (a) any file that references `/spec` mechanism and still uses the old "codebase exploration, web research, and interactive clarification" phrasing without the new "brainstorming" prefix; (b) any file that still references the old 8-step structure; (c) mirror parity for every asymmetric file pair that mentions `/spec`.

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/skills/spec/SKILL.md` | In sync | Primary artifact. 9 steps, divergent (step 2) / convergent (step 5) framing, Step 1 explicit anti-question guardrail, all cross-refs correct. |
| `templates/base/.claude/skills/spec/SKILL.md` | In sync | Byte-identical to root (AC4). |
| `.claude/skills/spec/template.md` | In sync | Unchanged. Output template is step-number-agnostic (describes spec document structure, not workflow), so no update needed. |
| `CLAUDE.md` (root) | In sync | L10 carries "iterative brainstorming (壁打ち)" in the `/spec` description. Other `/spec` mentions (L9 parenthetical "refine vague ideas") are process-role labels, not mechanism descriptions, so no need to repeat "brainstorming" there. |
| `templates/base/CLAUDE.md` | In sync | L10 carries the same phrasing. Pre-existing English/Japanese divergence with root (12 lines per self-review L20) is not widened. |
| `AGENTS.md` (root) | In sync | L22 in `## Primary loop` carries the new phrasing. |
| `templates/base/AGENTS.md` | In sync | L19 carries matching phrasing. Pre-existing 43-line divergence with root (per self-review L20) is not widened. |
| `README.md` (root-only) | In sync | L131 under `## Operating loop` carries "iterative brainstorming (壁打ち)". L117 (loop-overview one-liner) is a slash-command sequence that does not describe mechanism, so `/spec` does not need to be re-described. No `templates/base/README.md` exists (README is intentionally not mirrored). |
| `docs/architecture/repo-map.md` | In sync (borderline) | L29 reads `.claude/skills/spec/: refine vague ideas into detailed specifications (manual trigger)`. This is a repo-map one-liner that names purpose but not mechanism; it does not conflict with the new Step 2. Not drift in the strict sense, but a minor consistency improvement would be to append "; expands sparse inputs via iterative brainstorming (壁打ち)". Not blocking — flagged as non-critical polish. |
| `.claude/rules/subagent-policy.md` L40 | In sync | "relies heavily on `AskUserQuestion` for requirement clarification" — consistent with the new Step 2 + Step 5 (both use `AskUserQuestion`). The rule captures the policy reason (why `/spec` stays inline) accurately for both steps. |
| `templates/base/.claude/rules/subagent-policy.md` L40 | In sync | Mirror of above. |
| `docs/quality/definition-of-done.md` | In sync (not touched) | No `/spec` step mechanism described; pipeline order unchanged. |
| `.github/workflows/verify.yml` | In sync (not touched) | Does not reference `/spec` mechanism. |
| Old-structure references (grep for `Clarify requirements` / `step 8.*spec` / `8 step`) | In sync | Zero hits outside this branch's own reports — no stale references to the pre-change structure anywhere in the repo. |

Net: no blocking documentation drift. One non-blocking polish opportunity (`docs/architecture/repo-map.md:29`) is flagged but is a **repo-map one-liner**, not a mechanism description, and therefore does not fail AC5. The project's documentation rule (`documentation rules`) says "Keep `AGENTS.md` as a map, not an encyclopedia" — the same applies to `repo-map.md`, which argues **against** expanding it.

## Observational checks

- Step 2's divergent/convergent framing reads correctly end-to-end: Step 2 (divergent, expand) → Steps 3–4 (research/exploration) → Step 5 (convergent, resolve residuals) → Step 6 (draft) → Step 7 (approval) → Step 8 (output selection) → Step 9 (execute). The sequence is coherent.
- Step 1's "do NOT call `AskUserQuestion` here" guardrail is a runtime contract that cannot be statically enforced; the repo has no lint for skill prose. This is flagged below under "Coverage gaps" and is expected for docs-only prose changes.
- The self-review report already identified 4 LOW findings (all non-blocking readability/ordering concerns, no CRITICAL/HIGH/MEDIUM). `/verify` confirms none of those LOWs cross the spec-compliance or drift threshold — they are prose-quality suggestions, not contract violations.

## Coverage gaps

- **Runtime enforcement of Step 1's "no `AskUserQuestion`" rule**: there is no hook or linter that prevents a future agent from calling `AskUserQuestion` during Step 1. This is an inherent limitation of prose-level contracts. Not actionable within this branch; noted for tech-debt if it becomes a recurring bug. (Not added to `docs/tech-debt/` — single-point observation, not a recurring pattern.)
- **Behavioral test for Step 2 brainstorming**: running `/spec` with a one-line input to confirm Step 2 actually fires is outside `/verify` scope and belongs to `/test`. `/test` for docs-only changes typically returns "no behavioral tests applicable" and that is expected here.
- **Smallest useful additional verifier** (would increase confidence most, optional): a one-liner in `scripts/check-sync.sh` or a new trivial shell step that asserts `diff -q .claude/skills/spec/SKILL.md templates/base/.claude/skills/spec/SKILL.md` returns empty. `check-sync.sh` already does this transitively, so adding a named dedicated check is nice-to-have, not required.

## Verdict

- **Verified**:
  - AC1, AC2, AC3, AC4 fully verified from the diff and grep evidence above.
  - AC5 verified for all 5 applicable files (`CLAUDE.md`, `AGENTS.md`, `README.md`, `templates/base/CLAUDE.md`, `templates/base/AGENTS.md`). `templates/base/README.md` is correctly absent.
  - Static analysis (shellcheck, sh -n, jq -e, check-sync.sh, gofmt, golangci-lint, `go test ./...`) all PASS.
  - Byte-identical mirror for `.claude/skills/spec/SKILL.md` ↔ `templates/base/.claude/skills/spec/SKILL.md`.
  - No documentation drift blocking PR creation. No stale references to the old 8-step structure anywhere in the repo.
- **Partially verified**:
  - `docs/architecture/repo-map.md:29` does not mention "brainstorming" for `/spec`. Classified as non-drift (one-liner map entry, not a mechanism description) and consistent with the project's "map, not encyclopedia" documentation rule. Listed as optional polish, not a failure.
- **Not verified** (out of /verify scope):
  - Behavioral execution of `/spec` with a sparse input to confirm Step 2 actually runs — belongs to `/test`.
  - Subjective readability of Step 2's bullet ordering — belongs to self-review (which already flagged it as LOW, non-blocking).

**Overall verdict: PASS.** All 5 acceptance criteria met, static analysis clean, no documentation drift. Proceed to `/test` (expected: no behavioral tests applicable for docs-only diff), then `/sync-docs` (expected: no additional sync work since mirrors are already aligned), then `/codex-review` (optional), then `/pr`.
