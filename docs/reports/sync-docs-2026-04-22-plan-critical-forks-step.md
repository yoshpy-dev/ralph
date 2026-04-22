# Sync-docs — plan-critical-forks-step

- Branch: `docs/plan-critical-forks-step`
- Date: 2026-04-22
- Slug: plan-critical-forks-step
- Scope: Documentation sync check after adding Step 4.5 "Critical forks" to `/plan` SKILL.md and a "Design decisions" section to the three plan templates. Upstream steps (`/self-review`, `/verify`, `/test`) all PASS.

## Drift scan summary

| Area | Status | Action |
|------|--------|--------|
| `CLAUDE.md` / `templates/base/CLAUDE.md` | In sync (upstream) | No change — already verified |
| `.claude/skills/plan/SKILL.md` / mirror | In sync (upstream) | No change — this is the source of the behavior |
| `.claude/skills/plan/template.md` / mirror | In sync (upstream) | No change — already verified |
| `docs/plans/templates/feature-plan.md` / mirror | In sync (upstream) | No change — already verified |
| `docs/plans/templates/ralph-loop-manifest.md` / mirror | In sync (upstream) | No change — already verified |
| `docs/plans/templates/ralph-loop-slice.md` / mirror | Intentional exemption preserved | No change — slice template correctly omits Design decisions (forks are manifest-level) |
| `.claude/rules/subagent-policy.md` / mirror | In sync (upstream) | No change — already lists `critical-fork resolution during drafting` |
| `AGENTS.md` / mirror | In sync (no change needed) | Primary loop describes Plan generically as "creates plan, selects flow" — consistent with the new interaction model |
| `README.md` | In sync (no change needed) | Operating loop Step 2 says "Select execution flow" — still a true superset statement; no drift with "at minimum one decision + critical forks" |
| `docs/quality/definition-of-done.md` | In sync (no change needed) | Describes gate/checklists, not `/plan`'s internal interactions |
| `docs/recipes/ralph-loop.md` / mirror | In sync (no change needed) | Describes loop mechanics, not `/plan` interactions |
| `docs/architecture/repo-map.md` | In sync (no change needed) | Structural reference; unchanged by this behavior |
| `.claude/skills/work/SKILL.md` / mirror | In sync (no change needed) | Does not describe `/plan`'s interaction model |
| `.claude/skills/loop/SKILL.md` / mirror | In sync (no change needed) | Does not describe `/plan`'s interaction model |
| `.claude/skills/spec/SKILL.md` / mirror | **Drift found — fixed** | `/spec` vs `/plan` comparison table's User interaction row for `/plan` read only "Flow selection (standard/Ralph)", which undersold the new behavior. Updated to "Flow selection (standard/Ralph) + critical-fork resolution (convergent)". |
| `.claude/rules/planning.md` / mirror | In sync (no change needed) | Lists plan content requirements (objective, scope, etc.); not affected by interaction-model changes. "Design decisions" is added at the template level |
| `.claude/rules/documentation.md` / mirror | In sync (no change needed) | Generic doc-hygiene rules |

## Drift found and corrected

### `.claude/skills/spec/SKILL.md` (and mirror)

**Before** (line 17):

```
| User interaction | Iterative brainstorming (divergent) + clarification (convergent) | Flow selection (standard/Ralph) |
```

**After**:

```
| User interaction | Iterative brainstorming (divergent) + clarification (convergent) | Flow selection (standard/Ralph) + critical-fork resolution (convergent) |
```

**Why this counts as drift.** The table is the authoritative summary of role separation between `/spec` and `/plan`. After Step 4.5 landed, `/plan` now has a second convergent user-interaction mode (critical-fork resolution) that materially changes its role vs `/spec`. The table's pre-change text implied `/plan` only asked one thing (flow selection), which contradicts:

- `CLAUDE.md:12`: "asks at minimum one decision … and, when critical forks are detected during drafting, asks targeted AskUserQuestion follow-ups"
- `.claude/skills/plan/SKILL.md:44-62`: Step 4.5 contract
- `.claude/rules/subagent-policy.md:44`: "task type selection, objective confirmation, flow selection, critical-fork resolution during drafting, Codex advisory response"

The fix parallels the table's existing `/spec` cell: both sides now name the divergent vs convergent interaction mode explicitly, so readers using only this table for role comparison see the full `/plan` interaction surface.

**Minimality.** Single line changed per file; no surrounding prose touched. Preserves the table's column alignment semantics (reader scans left-to-right across roles) and keeps the divergent/convergent framing consistent with `/spec`'s cell.

## Slice-template exemption — verified not contradicted

The slice template (`docs/plans/templates/ralph-loop-slice.md` and its mirror) intentionally has no "Design decisions" section. Rationale (documented in the self-review and verify reports for this branch): critical forks are plan-level decisions recorded in the manifest, not per-slice decisions. Adding a Design-decisions field to the slice template would either (a) duplicate the manifest or (b) create drift between N slice copies.

Grep check: `slice.*design|slice.*decision|slice.*fork` returns hits only in this branch's review/verify reports — no user-facing doc asserts slices should carry design decisions, so the exemption stands uncontradicted.

## Mirror parity

Post-edit verification:

```
$ diff .claude/skills/spec/SKILL.md templates/base/.claude/skills/spec/SKILL.md
(no output)

$ ./scripts/check-sync.sh | tail -7
=== Sync Summary ===
  IDENTICAL:      107
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  9
  KNOWN_DIFF:     3

PASS: all files in sync.
```

Both root and `templates/base/` mirrors carry the updated line; no new KNOWN_DIFF or DRIFTED entries introduced.

## Plans and reports

- No active plan file in `docs/plans/active/` for this branch (the change is docs-only and was scoped via the user prompt, not a checked-in plan). Nothing to flip on a progress checklist.
- Upstream reports already on disk and referenced here:
  - `docs/reports/self-review-2026-04-22-plan-critical-forks-step.md`
  - `docs/reports/verify-2026-04-22-plan-critical-forks-step.md`
  - `docs/reports/test-2026-04-22-plan-critical-forks-step.md`

## Follow-ups (out of scope for this sync)

- **Smallest-useful verifier (from /verify report).** A `grep -q "^## Design decisions$"` guard across the four plan templates would catch future regressions. Not added in this branch — flagged for a follow-up PR if the section is ever deleted accidentally.
- **SKILL.md frontmatter `description:` length (from /self-review pass 2, LOW).** `.claude/skills/plan/SKILL.md:3` is now 465 chars. Non-blocking; worth splitting on the next unrelated edit.

## Verdict

PASS — one drift found (`.claude/skills/spec/SKILL.md` role-separation table), fixed in both root and `templates/base/` mirrors with a minimal single-line edit. Check-sync PASS. Slice-template exemption uncontradicted. All cross-references between the upstream-edited files remain coherent. Ready for `/codex-review` → `/pr`.
