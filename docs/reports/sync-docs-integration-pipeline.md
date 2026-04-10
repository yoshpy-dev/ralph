# Sync-docs Report: Integration Pipeline Feature

**Date:** 2026-04-10
**Branch:** feat/ralph-loop-v2
**Feature:** Integration pipeline (`--skip-pr --fix-all`) added to Ralph Orchestrator

## Changes implemented

1. `scripts/ralph-pipeline.sh` — Added `--skip-pr` and `--fix-all` flags
2. `scripts/ralph-orchestrator.sh` — Added `run_integration_pipeline()` function; `main()` now calls it after sequential merge; unified PR body updated to mention integration pipeline checklist item
3. `.claude/rules/subagent-policy.md` — Added description of integration pipeline step in the `/loop` section
4. `.claude/skills/loop/SKILL.md` — Updated "After the loop" section to describe the integration pipeline

## Files checked

| File | Drift found | Action |
|------|-------------|--------|
| `CLAUDE.md` | Yes — Ralph Loop flow description omitted integration pipeline step | Updated: added `integration pipeline (--skip-pr --fix-all)` between sequential merge and unified PR |
| `AGENTS.md` | Yes — Primary loop Step 3 Ralph Loop description omitted integration pipeline | Updated: same addition |
| `docs/recipes/ralph-loop.md` | Yes — "Integration with the operating loop" section omitted integration pipeline step | Updated: added `Integration pipeline on merged branch (--skip-pr --fix-all)` bullet |
| `docs/quality/definition-of-done.md` | Yes — Ralph Loop checklist and prose both omitted integration pipeline | Updated: added checklist item and updated prose |
| `.claude/rules/post-implementation-pipeline.md` | No drift — this file describes per-slice pipeline order; integration pipeline is orchestrator-level and already handled in `subagent-policy.md` | No change |

## Verdict

No structural changes required. All drift was additive: the integration pipeline step was simply missing from flow descriptions in CLAUDE.md, AGENTS.md, docs/recipes/ralph-loop.md, and docs/quality/definition-of-done.md. These are now consistent with the implemented behavior in `ralph-orchestrator.sh`.
