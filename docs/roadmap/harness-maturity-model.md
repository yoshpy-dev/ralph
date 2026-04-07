# Harness maturity model

## Level 0: Ad hoc prompting
- Chat-only workflow
- No persistent plans
- No deterministic verification

## Level 1: Map + verify
- `AGENTS.md`
- `CLAUDE.md`
- one verify script
- simple done criteria

## Level 2: Workflow scaffold
- `/plan`, `/work`, `/self-review`, `/verify`, `/test`, `/pr`
- report templates
- archived plans
- explicit evidence

## Level 3: Deterministic rails
- hook guardrails
- CI checks
- path-scoped rules
- stronger repo contracts

## Level 4: Parallel specialization
- subagents
- optional worktrees
- stack-specific packs
- better observability

## Level 5: Frontier orchestration
- evaluator loops
- Ralph Loop for autonomous multi-iteration work (`/loop`)
- agent teams where justified
- richer logs, traces, browser automation
- systematic harness audits and simplification passes

## Advice

Do not jump straight to Level 5.
The right harness is the smallest one that reliably improves your current work.
