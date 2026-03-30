---
name: work
description: Execute an approved plan in small coherent slices, updating progress, evidence, and docs as implementation evolves.
disable-model-invocation: true
---
Work from the active plan, not from memory alone.

## Steps

1. Read the current active plan in `docs/plans/active/`.
2. Confirm acceptance criteria and verification strategy before editing code.
3. Implement in small slices that can be reviewed and verified independently.
4. Update the plan's progress checklist while working.
5. If the task splits cleanly, delegate focused research or review to subagents.
6. If repeated failures occur, reduce scope, inspect evidence, and revise the plan instead of thrashing.
7. Keep docs, contracts, and tests aligned with behavior changes.
8. Before presenting completion, run `/review` and `/verify` or equivalent deterministic checks.

## Strong defaults

- One slice at a time
- Evidence before confidence
- Versioned plan over chat-only plan
- Smaller diffs over heroic rewrites
