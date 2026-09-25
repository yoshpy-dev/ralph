# sync-docs report: doctor-shell-alias-rc-types

- Date: 2026-09-25
- Plan: `docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md` (issue #167)
- Prior reports: `docs/reports/self-review-2026-09-25-doctor-shell-alias-rc-types.md`
  (Pass, recommend merge, LOW 6, fixed), `docs/reports/verify-2026-09-25-doctor-shell-alias-rc-types.md`
  (Pass), `docs/reports/test-2026-09-25-doctor-shell-alias-rc-types.md` (Pass)

## Summary

`ralph doctor`'s "Shell aliases (codex/claude)" check
(`internal/cli/doctor_shell_alias.go`) now (1) reports a candidate rc that
exists but is not a regular file (FIFO, device, socket; symlinks followed)
as `could not read: <file> (not a regular file)` instead of calling
`os.Open` on it, and (2) resolves a relative `$ZDOTDIR` against doctor's own
working directory, joining `$ZDOTDIR`-derived candidates by string
concatenation (not `filepath.Join`) so a `..` in the candidate resolves
through symlinks the same way the OS/zsh would, not lexically. No CLI flag,
Status contract, or severity rule changed.

This pass re-checked every documentation surface the verifier flagged as a
`/sync-docs` follow-up, made one targeted edit, added one new tech-debt row
for an out-of-scope finding recorded in the plan's Deviation notes, and
found no other stale prose.

## Documentation drift checked (no changes needed)

- **`docs/recipes/codex-seat-permissions.md`** (root and
  `templates/base/docs/recipes/codex-seat-permissions.md`) — `diff` clean
  (byte-identical). Both mentions of the check ("Shell aliases break every
  codex spawn...", "`ralph doctor`'s ... check reports...") describe the
  alias/flag-conflict mechanism, not the rc candidate list or `$ZDOTDIR`
  resolution. No drift from this fix.
- **`.claude/skills/org/SKILL.md`** and its three mirrors
  (`.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md`) — all four `md5`-identical.
  The one on-topic line ("`ralph doctor` の「Shell aliases (codex/claude)」
  Check が該当") only names the check's existence; no candidate-list or
  `$ZDOTDIR` detail to go stale.
- **`README.md`** — `grep -n "ZDOTDIR\|shell alias\|Shell aliases"` finds
  nothing; the Commands table describes `ralph doctor` generically, not
  per-check Detail text.
- **`docs/specs/`** — `grep -rln "shell alias\|Shell aliases\|ZDOTDIR"` finds
  no spec file. Nothing to update.
- **`docs/tech-debt/README.md:132`** (the `checkShellAliases` row) — content
  about config-aware severity grading is still accurate; this fix does not
  touch the flag-class/severity `switch`. Edited below (not left as-is): the
  row's own trigger condition ("next touch to
  `internal/cli/doctor_shell_alias.go` for another reason") fired with this
  PR, per the plan's self-review deviation note, and needed an explicit
  annotation rather than silent staleness.

## Files changed in this pass

- **`docs/tech-debt/README.md`**
  - `checkShellAliases` row, trigger cell: appended a note that #167
    (`fix/doctor-shell-alias-rc-types`, 2026-09-25) fired the first half of
    the existing trigger (a touch to `doctor_shell_alias.go` for FIFO/
    non-regular-rc handling and relative-`$ZDOTDIR` resolution) without
    taking on the config-aware-grading debt itself — the plan's Non-goals
    explicitly kept severity rules unchanged — and restated the remaining
    trigger (next change to the severity `switch`, or to how `ralph`
    resolves/passes `[org.permissions]` flags to a driver, or the original
    operator-report condition).
  - New row: `tests/test-ralph-dispatch.sh` case I ("SIGTERM cleanup left
    stray ralph-dispatch-* temp files") failed once during a full
    `./scripts/run-verify.sh` run on 2026-09-25 (recorded in the plan's
    Deviation notes as an out-of-scope finding) and passed standalone
    (26/26) and in every later full run. The case snapshots the shared
    `$TMPDIR` for new `ralph-dispatch-*` files and treats any new one as a
    leak; a concurrent Claude Code/Codex session's own hook run through
    `.claude/hooks/ralph-dispatch.sh` writes into the same `$TMPDIR` (the
    one leaked name observed was a `ralph-dispatch-merged.*` file), so the
    likely cause is test isolation, not a real leak (unverified). Deferred
    as out of #167's scope; trigger is the next touch to the test file or a
    recurrence.
- **`docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md`**
  - Added Deviation-notes bullets for `/verify` (904e5c1, PASS), `/test`
    (c74b7a0, PASS; mutations 5/5 discriminating; live demo; 97-100%
    coverage of the changed functions; 5 pre-existing/defensive gaps), and
    this `/sync-docs` pass, dated 2026-09-25, matching the style of the
    existing bullets.
  - Ticked `Verification artifact created` and `Test artifact created` in
    the Progress checklist. No AC checkbox touched. `grep -c '^# '` = 1
    before and after.
- **`docs/insights/events/2026-09-25-doctor-shell-alias-rc-types.jsonl`** —
  appended via `./scripts/insights-append.sh` (sync_docs, cycle 1, complete,
  0/0/0/0).

## Mirror/sync verification

```
$ ./scripts/check-sync.sh          # PASS: all files in sync. IDENTICAL: 159, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 11, KNOWN_DIFF: 5
$ ./scripts/check-skill-sync.sh    # [ok] check-skill-sync: 13 skill(s) in lock-step
$ ./scripts/check-template-purity.sh   # PASS: no meta-repo-specific references found in templates.
```

No skill body or `templates/base` mirror was touched this pass (the diff
only added rows/bullets in `docs/tech-debt/README.md` and the plan), so
`sync-skills.sh` was not re-run; the three gates above were re-run directly
against the current worktree state rather than trusting prior reports' word.

## Walkthrough

`git diff main...HEAD --stat -- internal/`: 3 files, 533 insertions(+), 43
deletions(-) — 576 changed lines, over the 500-line rule-of-thumb. Not
writing a walkthrough report in this pass; per the task handoff, `/pr`
decides whether one is warranted.

## Known gaps

- Did not re-run `./scripts/run-verify.sh` or `go test ./internal/cli/...`;
  that is `/verify`'s and `/test`'s completed job (both cited above, Pass).
- AC-9 (`./scripts/run-verify.sh` green, `TMPDIR=/tmp go test
  ./internal/cli/... -count=1` pass) is already confirmed by `/verify` and
  `/test`; not re-verified here.
