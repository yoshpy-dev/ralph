# worktree-gc-exit-code

- Status: Approved (user requested fix PR)
- Owner: Claude Code
- Date: 2026-07-12
- Related request: `ralph-worktree.sh gc` exits 1 even when it succeeds, whenever >= 1 stale entry was found/pruned
- Related issue: N/A
- Type: fix
- Branch: fix/worktree-gc-exit-code

## Objective

`gc_worktrees()` ends with `[ "$count" -eq 0 ] && printf 'No stale...'`
(scripts/ralph-worktree.sh:333). When stale entries were found (count > 0),
that final test evaluates false and becomes the function's return value, so
both `gc` (report mode) and `gc --prune` (which successfully deleted the
files) exit 1. Observed live while pruning 4 stale states. Callers chaining
`gc --prune && ...` or CI wrappers treat a successful cleanup as a failure.

## Scope

1. `scripts/ralph-worktree.sh` `gc_worktrees()`: make success exit 0
   regardless of count — replace the trailing short-circuit with an explicit
   `if`; end the function with `return 0`. Listing/prune behavior unchanged.
2. `templates/base/scripts/ralph-worktree.sh`: byte-identical mirror.
3. `tests/test-ralph-worktree.sh`: add gc cases (none exist today):
   (a) no state files -> exit 0 + "No stale" message;
   (b) one stale state (path missing) -> `gc` exits 0 and lists STALE,
       file NOT deleted;
   (c) `gc --prune` -> exits 0, file deleted, second run reports
       "No stale" with exit 0;
   (d) non-stale state (worktree path exists) -> not listed, not deleted.

## Non-goals

- No exit-code contract change beyond "success = 0" (no
  nonzero-when-stale-found reporting mode; nothing in the repo consumes one).
- Codex plan advisory intentionally skipped (2-line fix + tests; cross-review
  still runs before PR).

## Acceptance criteria

- [ ] AC1: `gc` and `gc --prune` exit 0 in all four test scenarios; prune
  still deletes exactly the stale files.
- [ ] AC2: mirrors byte-identical; check-sync passes.
- [ ] AC3: full `./scripts/run-verify.sh < /dev/null` and
  `./scripts/run-test.sh < /dev/null` pass.

## Design decisions

Success-exit-0 over exit-1-when-stale: the command's job is reporting/pruning
and it completed; a query-style nonzero contract would break `set -e`
callers and has no consumer. Critical forks: None.

## Test plan / Verify plan

Covered by AC1-AC3 (fixture-based gc cases + sync gates + full regression).
Evidence: docs/reports/{self-review,verify,test}-2026-07-12-worktree-gc-exit-code.md.

## Risks and mitigations

- Hypothetical caller relying on exit 1 to detect stale entries -> none in
  repo (grep); behavior documented in PR body.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (fix/worktree-gc-exit-code)
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created
