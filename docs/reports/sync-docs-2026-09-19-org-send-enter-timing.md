# sync-docs report: org-send-enter-timing

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-org-send-enter-timing.md` (issue #163)
- Prior reports: `docs/reports/self-review-2026-09-19-org-send-enter-timing.md`
  (MERGE, all 7 cycle-1 findings + 4 fix-round findings resolved),
  `docs/reports/verify-2026-09-19-org-send-enter-timing.md` (Pass, AC-1..AC-6
  met, no documentation drift found), `docs/reports/test-2026-09-19-org-send-enter-timing.md`
  (Pass, all commands green, 3 named coverage gaps + 1 named edge case)

## Summary

Slice C (commit `77371a8`) already rewrote the recipe's manual-Enter
procedure and the `/org` skill's `send` row for the core Slices A/B
behaviour (750 ms pause, submit confirmation via herdr, `submit_unconfirmed=true`,
no-resend). `/verify` independently confirmed that pass against the code at
its own HEAD and found no drift. This pass re-checked those two surfaces
against the code as it stands *after* Slices D and E (the fail-closed
`--timeout-ms` budget check and the `EnterPressed`-split CLI notes, both
added after the doc pass) and found one real gap: neither surface told an
operator what happens when `send` itself fails (as opposed to warning on a
successful-but-unconfirmed send). Added one to two sentences to each. Also
swept the other doc surfaces the handoff named and added one tech-debt row
for the two test-coverage gaps the test report asked about explicitly.

## Files changed in this pass

- `.claude/skills/org/SKILL.md` — `send` row: added that a `--timeout-ms`
  budget too small for the pre-Enter pause fails before typing anything, and
  that on an error exit the operator must read the stderr note (typed-only →
  check the pane before resending; Enter-already-pressed → do not resend).
  Neither sentence existed before; both are operator-facing behaviour from
  Slices D/E that postdated Slice C's doc pass.
- `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md` — regenerated via
  `./scripts/sync-skills.sh` (root → `.agents/`) plus a manual copy to the two
  `templates/base/` paths (the sync script only mirrors `.claude/` →
  `.agents/`, not the template copies); all four are now byte-identical again.
- `docs/recipes/codex-seat-permissions.md` — added one sentence after the
  existing "warns that it could not confirm" paragraph: if `send` exits
  non-zero instead of warning, follow the stderr note rather than retrying
  blindly (typed-only vs. Enter-already-pressed). `750 ms` stayed on the same
  line as `` `--enter-delay-ms` `` (unchanged from Slice C; the new sentence
  was inserted after that line, not into it).
- `templates/base/docs/recipes/codex-seat-permissions.md` — copied verbatim
  from the root file (`diff` empty after the copy).
- `docs/tech-debt/README.md` — appended one row for the two coverage gaps the
  test report named explicitly in its handoff to this task: `Send`'s
  `findSeat` manifest-read-error branch and `PaneSendText`'s own error
  branch have no unit test (both need a `fakeHerdr` field addition the
  Slice E handoff scoped out of its file list), and the literal 750 ms
  default is exercised only through the `EnterDelayMS <= 0` fallback
  mechanism, never asserted as the value actually used end to end. 6-pipe
  row shape and backtick balance checked against the file's existing
  convention (see Gate results below).
- `docs/plans/active/2026-09-19-org-send-enter-timing.md` — added one
  Deviation notes bullet ("2026-09-19 sync-docs (cycle 1)") summarizing this
  pass's edits and the surfaces confirmed unchanged. Did not touch Scope,
  Non-goals, Acceptance criteria, or the Progress checklist (no "sync-docs
  artifact created" checklist item exists, and the skill body names no
  instruction to add or tick one, so none was added).

## Surfaces checked — no change needed

- **`docs/specs/2026-08-01-org-runtime.md`** — `grep -n 'org send\|send-keys\|Enter\|enter-delay'`
  is empty; the spec never describes `send`'s Enter timing at any level of
  detail, so nothing there is now stale.
- **`README.md`** — the one `ralph org send` mention (Org runtime section) is
  a bare example invocation with no timing claim.
- **`AGENTS.md`** — no `org send`/`send-keys`/`enter-delay` mentions.
- **`.claude/rules/ralph/agent-messaging.md`** — its `ralph org send`
  mentions are all about typed-protocol validation (TYPE enum, TASK_ID,
  size cap), an orthogonal concern to Enter timing; no claim there changed.
- **`internal/org/prompts/lead.md`** — the only role-prompt template
  mentioning `ralph org send` (two lines); both just name the command as
  the delegation mechanism, no timing/confirmation claim to update. The
  other three role templates (`reviewer.md`, `qa.md`, `implementer.md`)
  have no `send`/`Enter` mentions at all.
- **`docs/tech-debt/README.md`, pre-existing rows** — no existing row
  described the manual-Enter-after-send workaround from #155/#163 (the
  closest historical record is `docs/evidence/codex-seat-permissions-2026-09-18.md`
  P2, an evidence file, not a tech-debt row), so there was nothing to mark
  resolved; the self-review's own conclusion (all 7+4 findings fixed
  in-cycle, no deferral) still holds and is unaffected by this pass's new row.
- **`docs/evidence/codex-seat-permissions-2026-09-18.md`** — already carries
  the one-line pointer to the new evidence file (added in Slice C); still
  accurate, no further edit.
- **`docs/evidence/org-send-enter-timing-2026-09-19.md`** — the live-fire
  evidence file itself; already complete per `/verify`'s AC-5 citation, no
  change owed by a docs-only pass.

## Stale "manual Enter" wording sweep

`grep -rln "send-keys.*Enter\|pane send-keys"` across the tree, beyond the
files already covered by Slice C, turned up three more hits:
`docs/plans/archive/2026-09-18-codex-seat-permission-verification.md` and
`docs/reports/self-review-2026-08-02-org-runtime-lead.md` — read both; they
describe `herdr pane send-keys` for an unrelated purpose (dismissing an
approval/trust dialog with `C-c`/a keystroke during permission verification
and lock-contention review, not the paste-then-Enter submit workaround this
task replaces) and are archived/historical artifacts besides, so left
unedited. `docs/evidence/codex-seat-permissions-2026-09-18.md` was already
handled by Slice C's addendum line (see above).

## Gate results

| Check | Result |
| --- | --- |
| `./scripts/check-skill-sync.sh` | PASS — 13 skill(s) in lock-step |
| `./scripts/check-sync.sh` | PASS — `DRIFTED: 0` (158 IDENTICAL, 11 TEMPLATE_ONLY, 5 KNOWN_DIFF, all pre-existing and unrelated to this change) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references found in templates |
| `go test ./internal/org/ -run TestSendDefaults -count=1 -v` | PASS — `TestSendDefaults_DocsMatchConstant` |
| `.claude/hooks/check_mojibake.sh` | Not run directly — it reads a PostToolUse JSON payload from stdin (hook-only entry point); it already ran automatically after each `Edit` in this session via `ralph-dispatch.sh` with no BLOCK. Manually re-confirmed as a substitute: `grep -c $'\xef\xbf\xbd'` (U+FFFD) against all 8 touched files returns 0 for each. |
| Markdown table well-formedness (new tech-debt row) | Manually checked: 6 pipes (matches every other row in the file's 5-column table) and an even backtick count (62, balanced) on the new line |

## Known gaps

- Did not re-run `go test ./internal/org/... ./internal/cli/...`, `-race`, or
  `TMPDIR=/tmp` — those are `/test`'s completed job (Pass, cited above); this
  pass only re-ran the one test named in the handoff
  (`TestSendDefaults_DocsMatchConstant`) because it directly reads the two
  doc surfaces this pass edited.
- Did not re-run the range secret scan (`./scripts/secret-scan.sh --range ...`)
  — that is a `/pr`-time pre-check per this task's own constraints; manually
  spot-checked the diff for credential-shaped `key=value` strings (none) and
  confirmed no `.gitallowed` edit was made.

## Orchestrator correction (2026-09-19, after this report)

The tech-debt row described above listed `PaneSendText`'s own error branch
as untested. That was already out of date when this pass ran: commit
3e35c4d (before this pass) added `fakeHerdr.paneSendTextErr` and
`TestOrgSend_PaneSendTextFails_NoEnterNoEventNoTypedFlag`. The test report
still names it as a gap because it predates that commit. The row was
rewritten to keep only the two gaps that remain (the `findSeat` read-error
branch, and the literal 750 ms default not asserted at runtime), and the
skill `send` row's last sentence was reworded in Japanese only (no change
in meaning).
