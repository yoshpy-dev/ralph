# Post-implementation pipeline order

Single source of truth for the post-implementation pipeline. All flows (standard /work, Ralph Loop) must follow this order.

## Canonical order

```
/self-review → /verify → /test → /sync-docs → /codex-review → /pr
```

No step may be skipped. If any step triggers a fix-and-revalidate cycle (e.g., Codex ACTION_REQUIRED), the **full pipeline** re-runs from `/self-review` onwards.

## Step responsibilities

| Step | Agent | Purpose | Stop condition |
|------|-------|---------|----------------|
| `/self-review` | `reviewer` | Diff quality | CRITICAL findings |
| `/verify` | `verifier` | Spec compliance + static analysis | Fail verdict |
| `/test` | `tester` | Behavioral tests | Fail verdict |
| `/sync-docs` | `doc-maintainer` | Documentation sync | — |
| `/codex-review` | inline | Cross-model second opinion | ACTION_REQUIRED triggers re-run |
| `/pr` | inline | PR creation + plan archival | — |

## Re-run after Codex ACTION_REQUIRED fix

When fixing Codex findings, the re-run includes **all** steps:

```
fix → /self-review → /verify → /test → /sync-docs → /codex-review
```

Not just `/self-review → /verify → /test → /codex-review`. The `/sync-docs` step must be included because fixes may change behavior that requires documentation updates.

## Where this order is referenced

If you update this order, update all of these locations:
- `.claude/skills/work/SKILL.md` (Step 9)
- `.claude/skills/loop/SKILL.md` (After the loop section)
- `.claude/skills/codex-review/SKILL.md` (Case A and Case B re-run)
- `.claude/rules/subagent-policy.md` (Post-implementation pipeline table)
- `CLAUDE.md` (Default behavior)
- `docs/quality/definition-of-done.md` (Pipeline order)
