# Post-implementation pipeline order

Single source of truth for the post-implementation pipeline (standard flow, `/implement`).

## Canonical order

```
/self-review → /verify → /test → /sync-docs → /cross-review → /pr
```

No step may be skipped. If any step triggers a fix-and-revalidate cycle (e.g., cross-review ACTION_REQUIRED), the **full pipeline** re-runs from `/self-review` onwards, except for the record-only fixes below.

### Exception: fixes confined to this task's pipeline records

A fix skips the full re-run only when all three of these hold:

1. The finding is about the record itself (a report's wording, an insight
   event's value), not about code, scripts, skills, rules, other docs, or the
   plan.
2. The fix commit changes only this task's records: report files directly
   under `docs/reports/` named after the plan slug (`<kind>-<date>-<slug>.md`,
   such as the self-review, verify, test, and sync-docs reports), and
   `docs/insights/events/<date>-<slug>.jsonl`. It does not touch
   `docs/reports/cross-review-triage-<slug>.md`, which holds the cross-review
   classifications and receives the check result below.
   `docs/reports/templates/` is an input that skills read, so it is never
   covered.
3. The fix changes no verdict line in this task's self-review, verify, or test
   report: the finding severities and the `Merge:` line of the self-review
   report, and the `## Verdict` lines of the verify and test reports. `/pr`'s
   pre-checks read those verdicts.

To check, commit the triage report first if it is not committed yet, make the
fix a single commit `<commit>`, then:

- `git diff --name-only <commit>^ <commit>` lists only the paths in condition 2.
- `git show --stat <commit>` shows the same files.
- `git diff <commit>^ <commit> -- <report>` for the self-review, verify, and
  test reports touches no verdict line.

Record the result (the `git show --stat` output and the verdict-line result)
in a separate commit `<record>` that only appends to the triage report
`<triage>` (`docs/reports/cross-review-triage-<slug>.md`):
`git diff --name-only <record>^ <record>` lists only `<triage>`, and the old
`<triage>` is a byte prefix of the new one, so text was added only at the end
and no existing line (an `After triage:` line, a classification) changed or
moved:

```
d=$(mktemp -d); git cat-file -e <record>^:<triage> && git show <record>^:<triage> > "$d/old" && git show <record>:<triage> > "$d/new" \
  && [ -z "$(tail -c1 "$d/old")" ] && head -c "$(wc -c < "$d/old")" "$d/new" | cmp -s - "$d/old"; rc=$?; rm -rf "$d"
```

`rc=0` means an append at the end. `git cat-file -e` requires the triage
report to exist before `<record>` (a record commit that creates it does not
qualify). The `tail -c1` test requires the old file to end with a newline (or
be empty); otherwise an append could extend its last line and still pass the
prefix check.

A diff that shows added lines only is not enough: a line inserted above the
`After triage:` line also shows as added only, and the triage parser reads the
first `After triage:` line. Say in the PR body that the exception was used.
`cycle-count.json` is not incremented. If any condition fails, or `<record>`
is not an append at the end, re-run the full pipeline (see "Re-run after
cross-review ACTION_REQUIRED fix" below).

Only `/cross-review` offers this exception (steps 8 and 9 of
`.claude/skills/cross-review/SKILL.md`). `/self-review`, `/verify`, and
`/test` write their own report for the current run, so they need none.

### CLI execution mode

Both Claude Code and Codex run the same canonical order, but the execution
model differs:

| Step | Claude Code (`/implement`) | Codex |
|------|------------------------|-------|
| `/self-review` | `Task(subagent_type="reviewer")` | `.codex/agents/reviewer.toml` custom agent |
| `/verify` | `Task(subagent_type="verifier")` | `.codex/agents/verifier.toml` custom agent |
| `/test` | `Task(subagent_type="tester")` | `.codex/agents/tester.toml` custom agent |
| `/sync-docs` | `Task(subagent_type="doc-maintainer")` | `.codex/agents/doc-maintainer.toml` custom agent |
| `/cross-review` | inline; calls `codex exec review` | inline; calls `claude -p` reviewer prompt |
| `/pr` | inline | inline |

Reports go to `docs/reports/` (CLI-neutral path) so the pipeline cycle counter
and PR pre-checks behave identically. The driver detection used by
`/cross-review` is documented in `.claude/skills/cross-review/SKILL.md`.

## Step responsibilities

| Step | Agent | Purpose | Stop condition |
|------|-------|---------|----------------|
| `/self-review` | `reviewer` | Diff quality only; no tests/static/spec/doc-drift/broad audit | CRITICAL findings |
| `/verify` | `verifier` | Spec compliance + static analysis via `./scripts/run-static-verify.sh`; changed-language scope by default; no tests | Fail verdict |
| `/test` | `tester` | Behavioral tests via `./scripts/run-test.sh`; changed-language scope by default; no static analysis | Fail verdict |
| `/sync-docs` | `doc-maintainer` | Documentation sync | — |
| `/cross-review` | inline | Cross-model second opinion using pinned plan/worktree state | ACTION_REQUIRED triggers re-run |
| `/pr` | inline | PR creation + plan archival + task worktree/local branch cleanup | — |

## Re-run after cross-review ACTION_REQUIRED fix

When fixing cross-review findings, the re-run includes **all** steps:

```
fix → /self-review → /verify → /test → /sync-docs → /cross-review
```

Not just `/self-review → /verify → /test → /cross-review`. The `/sync-docs` step must be included because fixes may change behavior that requires documentation updates.

### Pipeline cycle cap (default 2 total runs)

The post-implementation pipeline is capped at **2 total runs by default**: the initial run plus at most one fix-and-revalidate re-run. After the second run, the pipeline does not automatically regress even if cross-review still reports ACTION_REQUIRED.

Controlled by `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default `2`). The counter is persisted to `.harness/state/standard-pipeline/cycle-count.json`, keyed by the pinned plan path and task worktree state in `.harness/state/standard-pipeline/active-plan.json`. When the cap is reached, `/cross-review` drops the "fix" option from Case A/B and offers: (1) raise the cap and re-run, (2) fix records only (see "Exception: fixes confined to this task's pipeline records"; it stays available at the cap because it does not start a new run), (3) proceed to `/pr` and record remaining findings as known gaps, (4) abort.

Raise the cap only when you consciously accept additional churn; the default is a deliberate "fail fast, hand back to the operator" stance.

See `.claude/rules/ralph/subagent-policy.md` for execution model details.

## Where this order is referenced

If you update this order, update all of these locations:
- `.claude/skills/implement/SKILL.md` (Step 13)
- `.claude/skills/cross-review/SKILL.md` (Case A and Case B re-run)
- `.claude/rules/ralph/subagent-policy.md` (Post-implementation pipeline table)
- `docs/quality/definition-of-done.md` (Pipeline order)
- `README.md` (Quick start and Operating loop sections)
- `AGENTS.md` (Primary loop section)
