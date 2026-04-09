# Sync-docs report: Ralph Loop v2 — Plan System Redesign

- Date: 2026-04-10
- Plan: docs/plans/archive/2026-04-09-ralph-loop-v2.md
- Agent: doc-maintainer (claude-sonnet-4-6)
- Scope: feat/ralph-loop-v2 branch — directory-based plans, parallel slices, integration branch, unified PR, legacy inline slice mode removal

---

## Changes made

### CLAUDE.md

**Drift found**: The `/loop` bullet only mentioned `standard` and `pipeline` modes, with no reference to `parallel slices` mode added in ralph-loop-v2.

**Fix**: Added a sentence to the `/loop` bullet describing parallel slices mode:
> Pipeline mode also supports **parallel slices** via `ralph-orchestrator.sh`: directory-based plan → multi-worktree → integration branch → sequential merge → unified PR.

File: `CLAUDE.md` line 13.

---

### docs/architecture/repo-map.md

**Drift found**: The `scripts/` line listed `ralph`, `ralph-pipeline.sh`, `ralph-orchestrator.sh`, `new-ralph-plan.sh` but omitted `archive-plan.sh` and did not call out `new-feature-plan.sh` explicitly.

**Fix**: Updated `scripts/` description to:
> scripts/: bootstrap, plan creation (`new-feature-plan.sh`, `new-ralph-plan.sh`), plan archival (`archive-plan.sh`), verification, status, Ralph Loop orchestration...

File: `docs/architecture/repo-map.md` line 43.

---

### docs/recipes/ralph-loop.md

**Drift found**: The "Integration with the operating loop" section only showed the standard mode flow diagram. Pipeline mode (single) and pipeline mode (parallel slices) flows were absent.

**Fix**: Expanded the section into three sub-sections:
- Standard mode (original diagram, unchanged)
- Pipeline mode (single) — `ralph run` flow
- Pipeline mode (parallel slices) — `new-ralph-plan.sh` + `ralph run --unified-pr` orchestrator flow with integration branch

File: `docs/recipes/ralph-loop.md` — "Integration with the operating loop" section.

---

### README.md

**Drift found**: Quick Start step 3 only showed `new-feature-plan.sh`. Users wanting to use Ralph Loop parallel slices had no reference to `new-ralph-plan.sh`.

**Fix**: Added a parallel slices example alongside the standard plan creation command:
```sh
# Ralph Loop parallel slices (directory-based plan)
./scripts/new-ralph-plan.sh login-form N/A 3
```

File: `README.md` step 3.

---

### scripts/check-template.sh

**Drift found**: `new-ralph-plan.sh` is a core script added in ralph-loop-v2 for directory-based plan generation, but was not listed as a required file in the template structure check. `archive-plan.sh` was already listed.

**Fix**: Added `scripts/new-ralph-plan.sh` to the required files list in `check-template.sh`. Verified `./scripts/check-template.sh` passes with exit 0.

File: `scripts/check-template.sh`.

---

### docs/plans/archive/2026-04-09-ralph-loop-v2.md

**Drift found**: Progress checklist had `[ ]` for review, verification, and PR items, even though self-review and verify artifacts were created.

**Fix**: Marked review and verification artifacts as complete with file references:
- `[x] Review artifact created (docs/reports/self-review-2026-04-10-ralph-loop-v2.md)`
- `[x] Verification artifact created (docs/reports/verify-2026-04-10-ralph-loop-v2.md)`

PR remains `[ ]` pending creation.

File: `docs/plans/archive/2026-04-09-ralph-loop-v2.md`.

---

## No changes required

| File / contract | Status | Notes |
| --- | --- | --- |
| `AGENTS.md` | NO DRIFT | Primary loop step 3 lists all three loop modes including parallel slices. Repo map lists `new-ralph-plan.sh`. `docs/plans/templates/` lists both templates |
| `.claude/skills/loop/SKILL.md` | NO DRIFT | Step 1.5 loop mode selection, Step 6 run commands for all modes including `--slices --unified-pr` |
| `.claude/skills/plan/SKILL.md` | NO DRIFT | Step 2.7 includes parallel slices option; Step 3 uses `new-ralph-plan.sh` for directory plans |
| `.claude/rules/subagent-policy.md` | NO DRIFT | Pipeline mode and orchestrator mode sections are accurate |
| `docs/quality/definition-of-done.md` | NO DRIFT | Parallel slices DoD section present with `_manifest.md`, sequential merge, unified PR criteria |
| `docs/quality/quality-gates.md` | NO DRIFT | Pipeline mode gates documented; `run_hook_parity()` function name matches implementation |
| `docs/plans/templates/ralph-loop-manifest.md` | NO DRIFT | Shared-file locklist, dependency graph, integration-level verify/test plans all present |
| `docs/plans/templates/ralph-loop-slice.md` | NO DRIFT | Objective, AC, Affected files, Dependencies sections present |
| `docs/tech-debt/README.md` | NO DRIFT | All relevant items tracked; resolved items marked with strikethrough |

---

## Verification

- `./scripts/check-template.sh` — exit 0 (pass)
- `bash -n` on all 17 scripts — all pass
- All changed files reviewed for consistency with implementation

---

## Summary

5 files updated to resolve documentation drift introduced by the ralph-loop-v2 implementation:
1. `CLAUDE.md` — added parallel slices mode mention
2. `docs/architecture/repo-map.md` — added `archive-plan.sh` and `new-feature-plan.sh` to scripts description
3. `docs/recipes/ralph-loop.md` — expanded Integration section with pipeline and parallel slices flows
4. `README.md` — added `new-ralph-plan.sh` to Quick Start
5. `scripts/check-template.sh` — added `new-ralph-plan.sh` as required file
6. `docs/plans/archive/2026-04-09-ralph-loop-v2.md` — updated progress checklist

No CRITICAL or HIGH drift found. All changes are additive (adding missing information) rather than corrective (fixing wrong information).
