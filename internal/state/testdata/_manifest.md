# test-plan

- Status: Draft

## Objective

Test plan for dependency parsing.

## Dependency graph

```
slice-1 (foundation) ──→ slice-2 (watcher)
slice-1 (foundation) ──→ slice-3 (layout)
slice-3 (layout)     ──→ slice-4 (panes)
slice-3 (layout)     ──→ slice-5 (actions)
slice-2, slice-4, slice-5 ──→ slice-6 (integration)
```

## Next section

This should not be parsed.
