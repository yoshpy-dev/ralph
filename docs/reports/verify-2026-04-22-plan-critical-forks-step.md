# Verify report: plan-critical-forks-step

- Date: 2026-04-22
- Plan: N/A (no active plan document for this work — verified against the prompt's 8 acceptance criteria and the self-review report at `docs/reports/self-review-2026-04-22-plan-critical-forks-step.md`)
- Verifier: verifier subagent
- Scope: docs-only change (+76 / -6 across 12 files) on branch `docs/plan-critical-forks-step`. Static analysis + spec-compliance verification against the 8 ACs in the prompt. Docs drift scan for the "/plan's interaction model" description.
- Evidence: `docs/evidence/verify-2026-04-22-plan-critical-forks-step.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 — `/plan` SKILL.md Step 4.5 exists with three gate conditions and uses AskUserQuestion with 2-4 options per fork, writing to "Design decisions" section | Verified | `.claude/skills/plan/SKILL.md:44-47` — Step 4.5 header "**Critical forks (convergent)**" followed by the **all three** gate conditions (materially differ / cannot be resolved from repo / reversing mid-impl costs > one slice). Line 50: `Use AskUserQuestion with one focused question and 2-4 concrete options.` Line 51: `Record the chosen approach and its rationale in the plan's "Design decisions" section`. All three gate bullets are load-bearing and match the prompt's phrasing verbatim. |
| AC2 — Step 4.5 explicitly states it is "convergent" and contrasts with /spec's divergent brainstorming | Verified | `.claude/skills/plan/SKILL.md:44` — header is "Critical forks (convergent)". `.claude/skills/plan/SKILL.md:62` — `Purpose is **convergent** — narrow between enumerated options, not expand the design space. Divergent ideation belongs to /spec, not here.` Both the title label and a dedicated closing sentence assert convergence and explicitly delegate divergence to `/spec`. |
| AC3 — Step 4.5 includes a "Do NOT ask about" exclusion list (stylistic, rules-resolved, flow-level, reversible) | Verified | `.claude/skills/plan/SKILL.md:56-60` — literal `**Do NOT ask about**:` header followed by four bullets: (1) "Stylistic or easily reversible choices" (stylistic + reversible), (2) "Decisions already settled by `.claude/rules/`, `AGENTS.md`, or the upstream `/spec` output" (rules-resolved), (3) "The flow-level choice already made in step 2.7" (flow-level), (4) "Anything a reasonable default + explicit assumption would cover" (default-covered). All four categories from the prompt are present. |
| AC4 — `docs/plans/templates/feature-plan.md` has a "Design decisions" section between Affected areas and Acceptance criteria | Verified | `docs/plans/templates/feature-plan.md:18` = `## Affected areas`, `:20` = `## Design decisions`, `:25` = `## Acceptance criteria`. Lines 22-23 contain the rationale HTML comment and the "Critical forks: なし" fallback comment. Section order is exactly Affected areas → Design decisions → Acceptance criteria. |
| AC5 — `docs/plans/templates/ralph-loop-manifest.md` has a "Design decisions" section between Affected areas and Shared-file locklist | Verified | `docs/plans/templates/ralph-loop-manifest.md:20` = `## Affected areas`, `:22` = `## Design decisions`, `:27` = `## Shared-file locklist`. Lines 24-25 mirror the same bilingual comments. Section order is exactly Affected areas → Design decisions → Shared-file locklist. |
| AC6 — `.claude/skills/plan/template.md` (fallback) has the same section | Verified | `.claude/skills/plan/template.md:18` = `## Affected areas`, `:20` = `## Design decisions`, `:25` = `## Acceptance criteria`. Same bilingual rationale comments (lines 22-23). Structurally identical to `docs/plans/templates/feature-plan.md` at the section level (which is itself intentional since the fallback is a reduced mirror of the real template). |
| AC7 — All 6 mirror pairs under `templates/base/` are byte-identical (check-sync passes) | Verified | `scripts/check-sync.sh` output: `IDENTICAL: 107`, `DRIFTED: 0`, `PASS: all files in sync.`. Per-pair `cmp -s` check (in evidence log) confirms: SKILL.md ↔ mirror, template.md ↔ mirror, subagent-policy.md ↔ mirror, feature-plan.md ↔ mirror, ralph-loop-manifest.md ↔ mirror, and ralph-loop-slice.md ↔ mirror all identical. Note: the prompt says "6 mirror pairs"; the diff touches 5 template pairs + leaves the slice-template pair unchanged — all 6 pairs are byte-identical. |
| AC8 — CLAUDE.md and subagent-policy.md mention critical forks / critical-fork resolution | Verified | `CLAUDE.md:12` — `/plan` asks at minimum one decision — 標準フロー (/work) or Ralph Loop (/loop) — and, when critical forks are detected during drafting (two+ approaches with materially different risk/cost that cannot be resolved from repo context), asks targeted AskUserQuestion follow-ups before finalizing.` `.claude/rules/subagent-policy.md:44` — `/plan` AskUserQuestion list now includes `critical-fork resolution during drafting`. Both files reference the new step concisely and accurately. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0) | All shell `sh -n` checks OK (18 hooks × 2 mirrors), jq on both `settings.json` OK, `scripts/check-sync.sh` reports `IDENTICAL: 107 / DRIFTED: 0 / PASS`, Go gofmt/vet/test all OK (cached). |
| `./scripts/run-verify.sh` (full mode) | PASS (exit 0) | Same as static plus `tests/test-check-mojibake.sh` 11/11 PASS and Go verifier clean. Full log: `docs/evidence/verify-2026-04-22-090818.log` (auto-written by run-verify.sh). |
| `cmp -s <path> <templates/base mirror>` for 6 pairs | All identical | Confirms mirror byte-parity beyond what check-sync aggregates (check-sync already passes but this is per-file evidence). |
| `grep -rn -iE "(asks? only )?one decision\|asks one\|single decision" --include='*.md'` | 2 benign hits | Both are in `CLAUDE.md:12` / `templates/base/CLAUDE.md:12` and contain the updated phrase `asks at minimum one decision … and … critical forks … AskUserQuestion follow-ups`. No stale "only asks one decision" phrasing remains in docs. The third hit (`docs/reports/self-review-2026-04-21-ralph-default-opus-4-7.md:48`) is an unrelated historical report about a different plan — not drift. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `CLAUDE.md` / `templates/base/CLAUDE.md` (bilingual pair) | In sync | Line 12 in both files reflects the new interaction model ("at minimum one decision … critical forks … AskUserQuestion follow-ups"). `diff` shows only the three pre-existing known intentional divergences (Japanese vs English, the repo-local `/release` line, and `× N` phrasing). The new bullet is symmetrically translated. |
| `.claude/rules/subagent-policy.md` / mirror | In sync | Line 44's AskUserQuestion inventory for `/plan` now reads `task type selection, objective confirmation, flow selection, critical-fork resolution during drafting, Codex advisory response` — matches the new SKILL.md step 4.5 and the flow-selection step 2.7. |
| `.claude/skills/plan/SKILL.md` cross-references | In sync | `[template.md](template.md)` (line 51) resolves to the sibling fallback `.claude/skills/plan/template.md`, which carries the `## Design decisions` section at lines 20-23. The self-review's L1 notes this is a "ghost reference" (sibling-only) but the functional contract still holds: all four templates (fallback + feature-plan + ralph-loop-manifest + exempt slice) carry the section where applicable. Non-blocking. |
| `docs/plans/templates/ralph-loop-slice.md` | Intentionally in sync (unchanged) | Slice template deliberately untouched. Rationale: slice files are execution units without a plan-level "Design decisions" home, and forks are recorded in the manifest. Adding the section here would duplicate and risk drift. Self-review confirms and the exemption is explicitly stated in the plan diff's scope. |
| `AGENTS.md`, `README.md`, `docs/quality/definition-of-done.md` | In sync (no change needed) | These describe the overall pipeline order and skill boundaries. None of them described `/plan`'s internal interaction count in a way that conflicts with "at minimum one decision + critical forks". Grep for `one decision\|asks one\|single decision` across all `*.md` returned zero hits outside CLAUDE.md. |
| `anti-bottleneck` cross-reference (SKILL.md:93) | In sync | `.claude/skills/anti-bottleneck/SKILL.md` exists and is reachable. The Step 4.5 `Do NOT ask about` list functions as a concrete anti-bottleneck checklist for the plan skill. |
| `docs/tech-debt/README.md` | In sync | `git diff docs/tech-debt/` is empty. No pending debt entry introduced or orphaned. Self-review pass-1 had added an entry that pass-2 reverted cleanly. |

## Observational checks

- **Section positions verified by line number** (evidence log):
  - `docs/plans/templates/feature-plan.md` section order: 18 Affected areas → 20 Design decisions → 25 Acceptance criteria → 29 Implementation outline → 33 Verify plan → 40 Test plan.
  - `docs/plans/templates/ralph-loop-manifest.md` section order: 20 Affected areas → 22 Design decisions → 27 Shared-file locklist → 36 Dependency graph → 46 Integration-level verify plan → 53 Integration-level test plan.
  - `.claude/skills/plan/template.md` section order: 18 Affected areas → 20 Design decisions → 25 Acceptance criteria → 29 Implementation outline → 33 Verify plan.
- **Slice-template mirror check** (extra defense-in-depth):
  - `cmp -s docs/plans/templates/ralph-loop-slice.md templates/base/docs/plans/templates/ralph-loop-slice.md` → identical. Confirms the deliberate non-change is mirrored (no accidental one-sided edit).
- **Frontmatter budget note** (self-review L3):
  - `description:` on `.claude/skills/plan/SKILL.md:3` is 465 chars (self-review quote). Within operational tolerance but becoming wall-like. Not a verify failure; flagged for future tightening.
- **Working-tree caveat**:
  - All 12 changes are currently **uncommitted** in the working tree (including the self-review report). Per memory, "committed code is what ships, not the working tree." This verification reflects the working tree; a final confirmation after commit is advisable (recorded as a coverage gap, not a failure — the self-review also operated on the working tree and merge-YESed).

## Coverage gaps

- **Runtime semantics of `AskUserQuestion` flow in Step 4.5 are unverified.** Whether the model actually enumerates 2-4 options, whether it respects the "Do NOT ask about" exclusion list, and whether it writes to `## Design decisions` verbatim are runtime behaviors that cannot be statically verified from the prose alone. This is inherent to prose-level skill contracts; the same limitation applies to all `/plan` steps. Belongs to `/test` or a dogfooding run, not `/verify`.
- **Sibling `template.md` pointer (self-review L1).** `SKILL.md:51` links to the fallback `.claude/skills/plan/template.md`, but the actual plan-creation scripts copy the `docs/plans/templates/*` files. Functionally safe today (every template path that the reader or model could land on carries the section), but the pointer is technically a ghost reference. Non-blocking, tracked by self-review.
- **Uncommitted changes.** The branch has 12 modified files plus an untracked self-review report, none committed yet. A post-commit re-verify is recommended before PR to close the working-tree/committed-tree gap. Not a verify failure — all checks are green on the working tree.
- **No deterministic test covers "Design decisions" section presence across all four templates.** `scripts/check-sync.sh` guarantees mirror parity but not section-presence across the four canonical template files. If a future refactor deletes the section from one template but keeps the mirror consistent, check-sync would still pass. Proposing a smallest-useful-verifier below.

### Smallest useful verifier to add (high-leverage, low-cost)

A one-liner addition to `scripts/check-sync.sh` or a new `scripts/check-plan-templates.sh`:

```sh
# Assert that the "Design decisions" section exists in all four plan templates.
for f in \
  .claude/skills/plan/template.md \
  docs/plans/templates/feature-plan.md \
  docs/plans/templates/ralph-loop-manifest.md \
  templates/base/.claude/skills/plan/template.md \
  templates/base/docs/plans/templates/feature-plan.md \
  templates/base/docs/plans/templates/ralph-loop-manifest.md; do
  grep -q "^## Design decisions$" "$f" || { echo "MISSING: $f"; exit 1; }
done
```

This would turn AC4/AC5/AC6 into a deterministic static check and prevent silent regression if a future edit drops the section from one template (a known recurring blind-spot class for mirror-based repos — see memory). Not a merge blocker for this PR, but worth proposing as the follow-up verifier.

## Verdict

- Verified:
  - **AC1** — Step 4.5 with three gate conditions, AskUserQuestion 2-4 options, Design-decisions write-back. (`.claude/skills/plan/SKILL.md:44-52`)
  - **AC2** — Convergent label in header and dedicated closing sentence contrasting with `/spec`'s divergent brainstorming. (`.claude/skills/plan/SKILL.md:44,62`)
  - **AC3** — All four exclusion categories present in "Do NOT ask about" list. (`.claude/skills/plan/SKILL.md:56-60`)
  - **AC4** — `## Design decisions` between Affected areas and Acceptance criteria in `docs/plans/templates/feature-plan.md:20`.
  - **AC5** — `## Design decisions` between Affected areas and Shared-file locklist in `docs/plans/templates/ralph-loop-manifest.md:22`.
  - **AC6** — `## Design decisions` present in `.claude/skills/plan/template.md:20` (fallback).
  - **AC7** — All 6 mirror pairs byte-identical (`cmp -s` plus `scripts/check-sync.sh` `DRIFTED: 0 / PASS`).
  - **AC8** — CLAUDE.md line 12 and subagent-policy.md line 44 both mention critical forks / critical-fork resolution.
  - Static analysis: `./scripts/run-verify.sh` PASS (exit 0).
  - Documentation drift: no stale "one decision" / "asks one" phrasing remains in docs outside historical reports.

- Partially verified:
  - Cross-reference `[template.md](template.md)` on SKILL.md:51 resolves to the sibling fallback only; this is safe because every real template carries the section, but the pointer itself is not a precise landing reference. Matches self-review L1 (LOW).

- Not verified:
  - Runtime `AskUserQuestion` behavior under Step 4.5 (enumeration, exclusion-list adherence, Design-decisions write-back) — belongs to `/test` / dogfooding.
  - Post-commit state of the branch (current verification is on the working tree; 12 files still uncommitted).
  - Presence of a deterministic static check for the "Design decisions" section across all four plan templates (proposed smallest-useful verifier above — not required for this PR, recommended as follow-up).

- **Overall verdict: PASS.** All 8 acceptance criteria verified. Static analysis clean. No documentation drift. Docs-only change with zero runtime surface. Remaining items are runtime behaviors (proper domain of `/test`) and a single non-blocking L1 from self-review. Recommend proceeding to `/test` and then `/pr`.
