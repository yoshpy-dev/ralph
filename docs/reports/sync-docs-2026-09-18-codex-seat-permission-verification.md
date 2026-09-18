# sync-docs report: codex-seat-permission-verification

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-codex-seat-permission-verification.md` (issue #155)
- Prior reports: `docs/reports/self-review-2026-09-18-codex-seat-permission-verification.md`
  (Merge, no open findings after cycle-1 fix-and-revalidate),
  `docs/reports/verify-2026-09-18-codex-seat-permission-verification.md` (PASS),
  `docs/reports/test-2026-09-18-codex-seat-permission-verification.md` (PASS)

## Summary

This task's own scope (Scope items 4-5) is already a documentation task, and
the self-review's fix-and-revalidate cycle (`51df396`) plus the verify
report's Documentation-drift check already swept the primary doc surfaces
this change touches: `templates/base/ralph.toml`, `internal/org/permissions.go`
comments, `internal/config/config.go` comments, `docs/recipes/codex-setup.md`,
the `/org` skill (4 mirrors), and `docs/tech-debt/README.md`. This pass
independently re-checked the seven surfaces the orchestrator flagged plus the
plan's own Progress checklist, using fresh greps and reads against the
worktree rather than trusting the prior reports' citations. **No files were
changed in this pass** — every surface checked was already accurate or
already out of scope.

## Surfaces checked (no changes needed — already correct)

1. **`README.md`** — `grep -n "recipes\|codex-setup\|worktrees.md\|adding-a-language"
   README.md` returns no hits; the file never lists or links `docs/recipes/`
   contents, so there is no stale recipe reference to update. The one
   permission-related row (`README.md:286`, the Known-differences table's
   "Org seat permission policy" entry) describes the `[org.permissions]`
   enum generically ("same enum, mapped to Codex's own sandbox/approval
   flags") and does not claim anything about `codex_verified`'s gating or
   live-verification status that this change would contradict.

2. **`AGENTS.md` repo map, `docs/recipes/` entry (`AGENTS.md:112`)** —
   reads `docs/recipes/ — hands-on recipes (Codex setup, language packs,
   worktrees)`. `ls docs/recipes/` shows five files: `adding-a-language-pack.md`,
   `agent-teams.md`, `codex-seat-permissions.md` (new), `codex-setup.md`,
   `worktrees.md`. The parenthetical was already non-exhaustive before this
   change — `agent-teams.md` is not named either — so it is a deliberate
   "a few examples" map line, not a checklist. Left unchanged: adding the new
   recipe by name would be inconsistent with the existing precedent of
   omitting `agent-teams.md`, and `AGENTS.md`'s own hard rule is to keep this
   file a short map, not an encyclopedia of every recipe.

3. **`docs/specs/2026-08-01-org-runtime.md`** — `grep -n "permission 作法\|
   permission mode\|permission_mode\|guarded\|autonomous"` against the spec
   returns only one substantive hit, line 159: "自律座席の権限: 座席の
   permission mode / sandbox は現行 `RALPH_PERMISSION_MODE` / codex sandbox
   設定の流儀を継承し、エンベロープで指定可能にする。" This is a
   requirements-level design statement (permission mode is envelope-
   specifiable) that this task's evidence neither contradicts nor
   supersedes — it says nothing about codex's live-verification status one
   way or the other. `grep -n "codex_verified\|fail-closed" docs/specs/2026-08-01-org-runtime.md`
   is empty, so there is no now-inaccurate assertion to correct and no
   2026-09-18 revision-note entry is warranted under the spec's own
   convention (its `### <date> 改訂` headings and inline
   `(2026-09-16 改訂で撤去)` annotations mark only requirements the spec
   itself previously stated and this task's own PR changed — neither is
   true here). This matches the verify report's independent finding at
   `docs/reports/verify-2026-09-18-codex-seat-permission-verification.md:57`.

4. **`.claude/rules/ralph/agent-messaging.md`** and
   **`.claude/rules/ralph/model-routing.md`** — `grep -n "RESULT\|writable.*root\|agmsg"`
   against both files: `model-routing.md` has zero hits (it is scoped to
   Claude model-tier assignment, unrelated to codex seat permissions).
   `agent-messaging.md` mentions `RESULT`/`agmsg` only in the typed-protocol
   sense (the `RESULT` message *type*, and "seats communicate over an agmsg
   team") — it documents the message *shape* and validation contract, not
   the seat process's sandbox/writable-root prerequisites for the underlying
   agmsg SQLite write to succeed. That mechanic (codex's `--sandbox
   workspace-write` requiring the agmsg DB directory as a writable root,
   evidence P3) is already documented at the point where an operator acts on
   it: `templates/base/ralph.toml`'s `codex_verified` comment ("the agmsg DB
   directory must be a writable root or seats cannot send RESULT") and
   `docs/recipes/codex-seat-permissions.md`. A reader of `agent-messaging.md`
   is not misled by its absence there — the doc never claims RESULT delivery
   is guaranteed independent of the driver's own sandbox config — so no
   pointer was added, per the orchestrator's own "only if a reader would
   otherwise be misled" bar.

5. **`docs/tech-debt/README.md`** — `git diff e23c1f4...HEAD -- docs/tech-debt/README.md`
   is empty (confirmed independently; the verify report notes the same at
   `docs/reports/verify-2026-09-18-codex-seat-permission-verification.md:47`).
   `grep -n -i "codex_verified\|codex.*permission\|permission.*codex\|fail-closed"
   docs/tech-debt/README.md` returns zero hits — no open row references
   `codex_verified` or codex fail-closed verification, so there is nothing
   to close and no row was added, consistent with the plan's Non-goals
   ("tech-debt 行のクローズ … 該当行は存在しないため、`ralph.toml` の誤った
   ポインタを直すだけ").

6. **`docs/evidence/README.md`** conventions vs.
   `docs/evidence/codex-seat-permissions-2026-09-18.md`** — the README asks
   for "what was verified / how it was verified / what remains unverified"
   and evidence stronger than assertion (test output, structured logs,
   precise diff references) rather than raw dumps. The new evidence file
   follows this: it records versions, effective config keys, pass/partial/
   inconclusive verdicts per run, and points at scratchpad logs and manifest
   files rather than pasting them inline (e.g. `docs/evidence/codex-seat-permissions-2026-09-18.md:5`,
   "生ログ: セッションの scratchpad … リポジトリにはコミットしていない").
   No convention mismatch found.

7. **Plan Status / Progress checklist** — `docs/plans/active/2026-09-18-codex-seat-permission-verification.md:3`
   reads `Status: In progress`, which is accurate (the PR does not exist
   yet). The Progress checklist (lines 128-134) has every item checked
   through "Test artifact created" and correctly leaves "PR created"
   unchecked. No edit needed.

## Additional check: skill-mirror and root/template sync state

Not one of the orchestrator's seven items, but load-bearing for "is
documentation in sync" — re-ran the sync gates directly rather than trusting
the prior reports:

- `./scripts/check-sync.sh` → `PASS: all files in sync.` (`DRIFTED: 0`,
  `KNOWN_DIFF: 5`, matching the pre-existing known-diff set).
- `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`.
- `cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md` → identical.
- `cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md` → identical.
- `diff .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md` → all four identical.

No mirror regeneration was needed since no skill body was edited in this
pass.

## Files changed in this pass

None. Every documentation surface this change touches was already updated
by the implementation slice (Slice B, `docs/recipes/codex-seat-permissions.md`
+ template, `docs/recipes/codex-setup.md` See-also, `templates/base/ralph.toml`,
`internal/org/permissions.go` and `internal/config/config.go` comments,
`scripts/ralph-config.sh` + template, the `/org` skill's 4 mirrors) and its
self-review fix-and-revalidate cycle (`51df396`). The seven surfaces flagged
for this pass were confirmed already accurate or correctly out of scope; no
edit, sync-skills regeneration, or tech-debt row was required.

## Known gaps

- Did not re-run `go test ./internal/org/...` or `./scripts/run-verify.sh`;
  that is `/verify`'s and `/test`'s completed job (both reports cited above
  already PASS).
- Did not independently reproduce the codex behavioral claims in the
  evidence file (sandbox rejection, approval-prompt conditions); relied on
  the self-review's and verify report's own citations against the code
  (`internal/org/spawn.go`, `internal/cli/org.go`) plus this pass's own doc-surface
  greps, consistent with `/sync-docs`'s documentation-sync scope.
- The self-review flagged `internal/config/config.go:112` as a remaining
  stale `docs/plans/active/2026-08-02-org-runtime-watchdog.md` citation;
  the plan's own Deviation notes record that the orchestrator fixed it
  in-cycle at `7d07e2f`. Re-checked independently in this pass:
  `grep -n "docs/plans/active/2026-08-02-org-runtime-watchdog\|docs/plans/archive/2026-08-02-org-runtime-watchdog"
  internal/config/config.go` now shows both of that file's citations
  (lines 87, 112) already pointing at `archive/`. The self-review's broader
  "one of 19 such citations" note is still true for the *class* of stale
  citations elsewhere: `grep -rl "docs/plans/active/2026-08-02-org-runtime-watchdog"
  internal/` still finds three files (`internal/org/watcher.go`,
  `internal/org/statedir.go`, `internal/org/watch.go`), none of which is in
  this plan's Affected areas or was ever a claim this task's evidence
  depends on. Left unfixed, per the self-review's own Follow-ups note ("a
  separate chore should sweep the class") — out of scope for this task and
  for `/sync-docs`, not a gap this pass introduces.

## Cycle 2 (post cross-review AR-1 fix)

- Date: 2026-09-18
- Trigger: cross-review triage found one ACTION_REQUIRED item (AR-1);
  `33158e2` carried `--org-id`/`--config`/`--state-dir` through every recipe
  follow-up command, and `f769f40` (cycle-2 self-review LOWs) added the
  `task.txt` typed-protocol example, split `send` from `wait`, added a
  `--config` on the cleanup `status` line, and reflowed comments in the
  recipe and `scripts/ralph-config.sh` + template. HEAD is now `e12bacb`
  (cycle-2 self-review/verify/test sections committed, all Pass/PASS).
- Prior reports (cycle 2 sections): `docs/reports/self-review-2026-09-18-codex-seat-permission-verification.md`
  (Merge; cycle-2 LOW-4 fixed at `f769f40`, verify re-run green per
  `docs/evidence/verify-2026-09-18-085148.log`), `docs/reports/verify-2026-09-18-codex-seat-permission-verification.md`
  and `docs/reports/test-2026-09-18-codex-seat-permission-verification.md`
  (cycle-2 sections, both PASS).

### (a) New recipe command lines vs. other org-verb-documenting surfaces

- **`.claude/skills/org/SKILL.md` 動詞リファレンス (lines 90-99)** — every
  verb example in that table already shows `--org-id X` (`spawn`, `send`,
  `wait`, `read`, `status`, `stop`, `disband`, `report`, `watch`, `start`).
  The recipe's cycle-2 diff brings its own `send`/`wait`/`stop`/`disband`/
  `status` lines up to the same pattern (`--org-id perm-auto` /
  `--config ralph-autonomous.toml` / `--state-dir <scratch>/state-auto`, and
  the `perm-edits` equivalents). No contradiction — the recipe now matches
  the skill's own documented usage more closely than it did before, not less.
- **`docs/recipes/worktrees.md`** — its one `ralph org` mention (line 55,
  "`ralph org spawn` records each seat's worktree path in the org manifest")
  is a one-line factual note about manifest recording, not a command example
  with its own `--org-id` convention to reconcile. Unaffected.
- **`docs/recipes/agent-teams.md`** — `grep -n "ralph org"` against the file
  returns zero hits; it does not document org verbs at all. Unaffected.
- **`.claude/rules/ralph/agent-messaging.md`** Message shape section
  (lines 59-65: `TYPE:` / `TASK_ID:` / other header lines / blank line /
  body) — the recipe's new `task.txt` example
  (`docs/recipes/codex-seat-permissions.md:90-99`) is `TYPE: TASK`,
  `TASK_ID: t-1`, a blank line, then a three-step numbered body, and the
  recipe's own RESULT step names `TYPE: RESULT, TASK_ID: t-1`. Both TYPE
  values and the TASK_ID-required pairing (`TASK`/`RESULT` both require
  `TASK_ID` per the enum table, `agent-messaging.md:39-41`) match the rule
  doc exactly; body length is a few hundred characters, well under the
  2,000-character cap the rule doc states (`agent-messaging.md:80-87`). No
  drift.

### (b) Mirror and sync-gate re-check

Re-ran directly against the cycle-2 HEAD rather than trusting the commit
messages:

- `cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md` → identical.
- `cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md` → identical (untouched in cycle 2).
- `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` → identical.
- `./scripts/check-sync.sh` → `PASS: all files in sync.` (`DRIFTED: 0`, `KNOWN_DIFF: 5`, unchanged from cycle 1).
- `./scripts/check-template-purity.sh` → `PASS: no meta-repo-specific references found in templates.`
- `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step` (no skill body was touched in cycle 2, so this is a re-confirmation, not new coverage).

### (c) Plan Status / Progress checklist vs. reality

`docs/plans/active/2026-09-18-codex-seat-permission-verification.md:3`
still reads `Status: In progress`; the Progress checklist (around lines
130-137) has everything through "Test artifact created" checked and
"PR created" still unchecked — accurate, since no PR exists yet. The plan's
Deviation notes already carry a 2026-09-18 self-review(cycle 2) entry
recording the LOW-4 fix and the verify re-run evidence path. No edit
needed.

### Files changed in cycle 2

None. All three checks confirmed the cycle-2 docs-only fix commits
(`33158e2`, `f769f40`) are internally consistent with every other doc
surface that documents org verbs or the agmsg message shape, both mirror
pairs stayed byte-identical, and the plan's own status tracking is already
accurate. No new drift was introduced by the cross-review AR-1 fix or the
cycle-2 self-review LOWs.
