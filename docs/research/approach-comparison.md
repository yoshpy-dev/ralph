# Existing harness engineering approaches: comparison and synthesis

This document distills the main approach families that informed this scaffold.

## Executive summary

There is no single best harness.

The common pattern across strong implementations is:
- keep always-on context small
- add structure around planning and verification
- use deterministic rails for hard constraints
- treat human attention as scarce
- increase harness complexity only when the model or task actually needs it

This scaffold therefore chooses a **hybrid default**:
- minimal always-on map
- on-demand workflows via skills
- deterministic runtime controls via hooks and scripts
- evidence files for review and verification
- optional escalation to subagents, worktrees, and agent teams
- optional evaluator loops only for tasks beyond solo reliability

## Approach families

| Approach family | Core move | Strengths | Weaknesses | Best fit | Borrowed into this scaffold |
| --- | --- | --- | --- | --- | --- |
| Prompt + verification baseline | Break work into smaller sessions, separate planning from execution, and give the agent a way to verify its own work | Low ceremony, immediate improvement, easy to start | Easy to regress without file-backed process | Solo agent work, small repos, early adoption | plan-first loop, verify-first mindset |
| Repo as system of record | Short map file plus structured docs, versioned plans, doc hygiene, explicit boundaries, observability | Excellent legibility, better resilience across long tasks, reduces hidden context | Requires documentation discipline | Long-running or multi-person agent work | `AGENTS.md` + `docs/` + first-class plans and reports |
| Planner / generator / evaluator harness | Separate planning, implementation, and QA into distinct agents with explicit grading criteria and feedback loops | Higher quality on complex tasks, especially subjective or bug-prone ones | Slow, expensive, evaluator tuning is hard | Frontier tasks beyond reliable solo capability | optional subagents, report artifacts, evidence contracts |
| Claude-native workflow layering | Use CLAUDE.md, rules, skills, subagents, hooks, and MCP for layered control | Native to Claude Code, flexible, strong progressive disclosure story | Easy to overbuild or create routing conflicts | Claude Code primary workflow | core control plane of this scaffold |
| Skill orchestration and description tuning | Treat SKILL.md as an orchestration layer, use support files and scripts, tune descriptions, audit routing quality | Great for reusable workflows, scales specialized behavior | Description conflicts and trigger drift can appear | Mature Claude Code setups with many workflows | workflow skills, support templates, harness audit skill |
| Runtime-enforced harness plugin | Guardrails on the execution path, stage-gated workflow, evidence pack, hook-heavy safety model | Strong protection and repeatability | More moving parts, can become heavy for simple repos | Teams or repos with strong safety and review needs | minimal runtime hooks, explicit reports, staged loop |
| Continuous loop / Ralph style | Keep the agent iterating continuously, often with a while loop and persistent task file | High autonomy, strong for greenfield exploration | Needs very careful tuning; can wander in brownfield work | greenfield, event-consistency-friendly tasks | idea of persistent plan/spec files, not the default runtime |

## Trade-offs that matter most

### 1. Context strategy

- Big manuals rot and consume context.
- Short maps plus structured deeper docs age better.
- Conditional rules and on-demand skills reduce noise.
- Context resets can help some models, but they add orchestration cost.
- Compaction is cheaper than full resets, but not always enough on weaker long-context behavior.

### 2. Verification strategy

- Self-verification improves agents dramatically when checks are runnable.
- Subjective QA usually needs explicit criteria and often a separate evaluator.
- Evaluators add lift, but only when the task is beyond solo reliability.
- If the evaluator is weak or under-calibrated, it becomes expensive theater.

### 3. Control plane

- Instructions are good for preferences and workflow.
- Hooks and scripts are better for hard guarantees.
- CI is the outer safety net; local hooks are the fast inner loop.
- Rules alone should not be trusted for invariants that must never break.

### 4. Parallelism

- Subagents are good when you want isolated context and summarized results.
- Agent teams are good when workers need to communicate directly.
- Worktrees are valuable when write isolation matters.
- All parallelism adds token and coordination cost.

### 5. Human bottlenecks

- Asking the human too often destroys throughput.
- Good harnesses teach the agent to verify, infer, or choose defaults first.
- Human review should focus on judgment and approval, not routine fact gathering.

## Design decisions in this repository

1. **Portable map first**
   - `AGENTS.md` is vendor-neutral.
   - `CLAUDE.md` imports `AGENTS.md`.

2. **Claude-native workflow layer**
   - Rules for path-scoped guidance
   - Skills for plan / work / review / verify
   - Subagents for planning, review, verification, doc maintenance

3. **Deterministic rails**
   - Minimal hooks for runtime guardrails
   - Reusable verification scripts
   - Simple CI check for template integrity

4. **Evidence as artifact**
   - Plans, reviews, verify reports, and walkthroughs are versionable files

5. **Language packs, not language lock-in**
   - Core scaffold is language agnostic
   - Depth lives in `packs/languages/`

6. **Optional escalation**
   - Worktrees, teams, evaluator loops, and external observability are recipes, not defaults

## What this scaffold intentionally does not do by default

- It does not start with agent teams turned on.
- It does not force custom worktree hooks on every repo.
- It does not assume a single language.
- It does not hard-code a giant always-on instruction file.
- It does not claim verification happened unless a verifier actually ran.

## When to increase complexity

Add more harness only if one of these is true:
- long tasks keep derailing
- quality regressions keep recurring
- review volume is overwhelming the human
- the repo is large enough that context thrash is common
- you need stronger guarantees around destructive actions or secrets
- parallel work is real, not just aspirational

## Source notes

Full source notes are in `docs/references/source-notes.md`.
