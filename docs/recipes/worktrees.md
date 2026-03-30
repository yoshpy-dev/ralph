# Worktrees

Use worktrees only when isolation clearly improves outcomes.

Good candidates:
- parallel risky changes with low file overlap
- exploratory spikes you may discard
- review or verification in a clean checkout
- multiple agents writing in parallel

Avoid by default for:
- small changes
- strongly sequential tasks
- same-file heavy work

Notes:
- Claude Code already supports worktree-based isolation.
- Custom `WorktreeCreate` and `WorktreeRemove` hooks are advanced because they must return or consume paths correctly.
- Keep worktree automation optional until you feel real pain from single-tree work.
