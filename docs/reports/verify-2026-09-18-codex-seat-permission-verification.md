# Verify report: codex-seat-permission-verification

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-codex-seat-permission-verification.md` (issue #155)
- Verifier: `verifier` subagent (Claude Code, standard flow)
- Branch: `docs/codex-seat-permission-verification`, HEAD `3dab6bc4` (base `main` `e23c1f4`, 12 commits ahead)
- Scope: spec compliance (AC-1..AC-6) + static analysis via `./scripts/run-static-verify.sh` (changed-language scope: Go + markdown). No behavioral tests run (`/test` owns that); no source/prose edits made.

## Overall verdict: PASS

All six acceptance criteria are met (AC-6's `Closes #155` half is out of scope for `/verify` by design — it lands at `/pr`; its seat-cleanup half is met). Static analysis is fully green. No inaccurate claim or unresolved documentation drift was found; two small, previously-known, non-blocking loose ends carry over from the self-review as informational only.

## Static analysis

`git status --porcelain` was clean before and after (no uncommitted changes; nothing to lose). `RALPH_VERIFY_SCOPE` defaulted to changed scope.

```
$ ./scripts/run-static-verify.sh
==> scripts/check-sync.sh        PASS (IDENTICAL 158, DRIFTED 0)
==> scripts/check-pipeline-sync.sh   OK
==> scripts/check-skill-sync.sh  [ok] 13 skill(s) in lock-step
==> scripts/check-template-purity.sh PASS
==> golang verifier: gofmt: ok / go vet: clean (silent) / golangci-lint: 0 issues. / staticcheck: clean (silent)
==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-09-18-075351.log
```

`go test` was **not** executed — `run-static-verify.sh` forces `HARNESS_VERIFY_MODE=static`, and `packs/languages/golang/verify.sh` gates `run_tests`/`go test ./...` behind a separate `mode == test` branch (`packs/languages/golang/verify.sh:108-121`) that static mode never reaches. Confirmed by reading the script, not by inference.

Also independently confirmed `./scripts/check-template-purity.sh` (separate PASS, no meta-repo-specific references in templates — this matters because the self-review's fix round replaced a leaked developer path with a placeholder in `529b360`).

## Per-AC results

**AC-1 — autonomous evidence (PASS).** `docs/evidence/codex-seat-permissions-2026-09-18.md` records versions (codex-cli 0.154.0, herdr 0.7.5, agmsg 1.1.13, ralph temp binary @ `e38dc27`), the effective codex config keys, `spawned` Details (`permission_mode=autonomous`), the child-process command line (`--sandbox workspace-write --ask-for-approval never --model gpt-5.5`), a pass verdict, a pane excerpt with no approval dialog, the sent TASK, `wait`/RESULT summary, and stop/disband/status cleanup. `grep -c '/Users/' docs/evidence/codex-seat-permissions-2026-09-18.md` = 0.

**AC-2 — edits evidence (PASS).** Same file records the edits child-process line (`--sandbox workspace-write`, no `--ask-for-approval`), unprompted in-cwd edit success, the outside-write three-way verdict (Run E1 = partial, Run E2/E2b = "pass（機構として）"), and an independent post-hoc confirmation that the target file does not exist. The recipe's "edits prompts outside cwd" claim is correctly hedged behind the model-initiated-escalation condition rather than stated unconditionally (`docs/recipes/codex-seat-permissions.md:117-128`), matching the plan's AC-2 instruction to only make that claim when the verdict is pass.

**AC-3 — recipe (PASS).** `cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md` → identical. `cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md` → identical (the added See-also line is present in both, wrapped at ~79 columns per the self-review LOW-1 fix). Recipe content covers prerequisites (codex/herdr/agmsg, shell-alias hazard, agmsg writable root), scratch config for both modes, per-mode spawn commands with distinct org ids/state dirs, observation points, a pointer to the meta-repo's own result, the `codex_verified = true` opt-in step tied to version/config with a re-verify clause, and a two-org cleanup step. `./scripts/check-sync.sh` → PASS (DRIFTED 0; `docs/recipes/` correctly outside the ROOT_ONLY exclusion set, so both copies are checked).

**AC-4 — comment/doc repositioning (PASS).**
- `grep -c 'tech-debt' templates/base/ralph.toml` = 0; both `[org.permissions]` comment blocks now point at `docs/recipes/codex-seat-permissions.md` instead.
- `internal/org/permissions.go` carries the verification date, codex-cli version, and evidence path (`internal/org/permissions.go:91-99` — read directly, matches plan wording). `git diff main -- internal/org/permissions.go`, filtered to non-comment lines, is exactly the two `fmt.Errorf` string changes at what is now lines 122 and 127 (`"...requires [org.permissions].codex_verified=true; only guarded is allowed until then (fail-closed; verify the codex flag mapping on this machine first: docs/recipes/codex-seat-permissions.md)"`); no other logic line changed — confirmed by reading the full diff hunk, not by grep alone.
- `internal/config/config.go`'s `OrgPermissionsConfig`/`CodexVerified` doc comment is repositioned the same way (verification date + evidence path + "default false by design" + recipe pointer) and both of its `docs/tech-debt/README.md` references are gone; its two `docs/plans/active/2026-08-02-org-runtime-watchdog.md` citations were also repointed to `docs/plans/archive/2026-08-02-org-runtime-watchdog.md`, and that path exists under `archive/`, confirming the citation fix is accurate (`ls docs/plans/archive/2026-08-02-org-runtime-watchdog.md` succeeds; the `active/` copy is gone).
- `grep -c 'tech-debt' internal/config/config.go internal/org/permissions.go` = 0 for both (verified via a combined grep — no hits at all, i.e. the file:line pairing wasn't needed because there's nothing to report).
- `grep -c '暫定制約' .claude/skills/org/SKILL.md` = 0; the permission-mode bullet was reworded to describe default-false-fail-closed plus the `codex_verified`/recipe opt-in path, and a new prerequisite bullet about shell aliases was added, matching the plan's Scope 5 description exactly.
- 4-way `cmp` across `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`, `templates/base/.agents/skills/org/SKILL.md` → all identical. `check-skill-sync.sh` → `[ok] 13 skill(s) in lock-step`.
- `docs/tech-debt/README.md` has zero diff against `main` (`git diff e23c1f4...HEAD -- docs/tech-debt/README.md` is empty) and `grep -c codex_verified docs/tech-debt/README.md` = 0, consistent with the plan's Non-goal that no register row exists to close.

**AC-5 — green build (PASS, static half; full go test deferred to `/test` by design).** `gofmt`, `go vet`, `golangci-lint`, and `staticcheck` are all clean under `./scripts/run-static-verify.sh` (see Static analysis above). `go test ./internal/org/...` itself was intentionally **not** run here — that is `/test`'s job per the pipeline contract, and the self-review's own note already confirmed AC-5's verify-script half was re-run after the `51df396` fix round (evidence log `docs/evidence/verify-2026-09-18-074556.log`, gitignored but its existence and the self-review's citation of it are consistent with `docs/evidence/*.log` being gitignored by convention).

**AC-6 (verify-relevant half) — PASS; `Closes #155` half deferred to `/pr` as expected.** The opt-in section of the recipe ties `codex_verified = true` to "the codex version and the effective `~/.codex/config.toml` you verified with" and explicitly calls for re-running after a codex upgrade or a config change (`docs/recipes/codex-seat-permissions.md:150-153`). Seat-cleanup is evidenced in the evidence file's 後始末 section: real `~/.codex/config.toml` restored from backup (byte-verified), duplicated `auth.json` removed, `herdr server stop` run, `herdr agent list` empty, all verification panes closed, and no `codex-perm-outside-*` file left under the real home. `docs/reports/self-review-...md` independently notes both `perm-auto` and `perm-edits` org cleanups are covered after the MEDIUM-2 fix. `Closes #155` is correctly not yet present anywhere in this branch — that line belongs in the PR description, produced at `/pr`.

## Documentation-drift check

- The recipe's technical claims were cross-checked against the current code, not trusted from prose: `spawned` Details really only ever record `scope=`/`allow_unscoped=`/`permission_mode=` and never argv (`internal/org/spawn.go:742-745,821-826`); `agmsgTeam` really is `fmt.Sprintf("ralph-%s", orgID)` (`internal/org/spawn.go:885-889`), matching the recipe's `history.sh ralph-perm-auto lead 20` example; every flag named in the recipe's two spawn commands (`--org-id`, `--id`, `--role`, `--driver`, `--model`, `--cwd`, `--scope`, `--config`, `--state-dir`) is a real flag on `ralph org spawn` (`internal/cli/org.go:40-42,234-241`); `ralph doctor` has no `codex_verified` check (`grep -n 'CodexVerified\|codex_verified' internal/cli/doctor.go` — zero hits), matching the recipe's explicit "`ralph doctor` does not validate `codex_verified`" line; and the idempotent-respawn hazard the recipe warns about in its edits step ("reusing `perm-auto` would hit the idempotent respawn path") matches `internal/org/spawn.go:417-421`'s documented behavior.
- The three comment/doc surfaces (`templates/base/ralph.toml`, `internal/org/permissions.go`, `internal/config/config.go`) and the `/org` skill bullet all now state the same policy consistently: default `false` by design (not by verification gap), per-machine opt-in via the recipe, flags inherit the operator's own codex config, and the agmsg-writable-root caveat is called out at least once in each surface that discusses opt-in mechanics.
- `docs/specs/2026-08-01-org-runtime.md` was checked for anything now contradicted by this PR: its only permission-mode-related line (`docs/specs/2026-08-01-org-runtime.md:159`, "自律座席の権限 … 現行 codex sandbox 設定の流儀を継承し、エンベロープで指定可能にする") is a high-level design statement that this PR's evidence does not contradict — it says nothing about live-verification status one way or the other.
- `internal/org/permissions_test.go`'s three updated assertions track the new error string exactly (`requires [org.permissions].codex_verified=true`) and the distinctness subtest still correctly asserts the *absence* of that substring for the unrelated "unknown permission mode" error — the two error paths remain discriminable.

## Known gaps / informational carryover (non-blocking)

These were already surfaced and explicitly accepted as non-blocking by the self-review's own final round; re-confirmed here as still present and still non-blocking, not re-litigated:

- `scripts/ralph-config.sh:55-63` and its template mirror still describe `codex_verified` with the pre-this-PR "fail-closed … until an operator has live-verified their installed codex CLI's flags" wording and do not name the recipe. It is still accurate under the new framing (nothing it says is false), just the one member of the three-way lock-step trio (`internal/config/defaults_sync_test.go:158`) that does not point at `docs/recipes/codex-seat-permissions.md`. Not required by any AC; flagged as a natural follow-up if this trio is touched again.
- Whether codex expands `~` inside `sandbox_workspace_write.writable_roots` remains genuinely untested (the evidence and recipe both say so plainly rather than asserting an unverified behavior) — this is documented honestly as an open question, not a drift.

No new drift, inaccurate claim, or AC gap was found beyond what the self-review already tracked and resolved.

## Evidence commands run

```
grep -c '/Users/' docs/evidence/codex-seat-permissions-2026-09-18.md            # 0
grep -c 'tech-debt' templates/base/ralph.toml                                    # 0
grep -c 'tech-debt' internal/config/config.go internal/org/permissions.go        # 0 (both)
grep -c '暫定制約' .claude/skills/org/SKILL.md                                   # 0
grep -c 'codex_verified' docs/tech-debt/README.md                                # 0
cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md   # identical
cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md       # identical
cmp .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md                      # identical
cmp .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md       # identical
cmp .claude/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md       # identical
git diff e23c1f4...HEAD -- internal/org/permissions.go                           # 2 error-string lines only, non-comment
git diff e23c1f4...HEAD -- internal/config/config.go                             # comment-only
git diff e23c1f4...HEAD -- docs/tech-debt/README.md                              # empty
grep -Eo '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' docs/evidence/codex-seat-permissions-2026-09-18.md   # no matches
./scripts/check-sync.sh                                                          # PASS, DRIFTED 0
./scripts/check-skill-sync.sh                                                    # PASS
./scripts/check-template-purity.sh                                               # PASS
./scripts/run-static-verify.sh                                                   # PASS, evidence: docs/evidence/verify-2026-09-18-075351.log
```

## Cycle 2 (post cross-review AR-1 fix)

- Date: 2026-09-18
- HEAD: `d40c6d2` (was `3dab6bc4` at cycle-1 verify; 9 commits since, all `docs:`)
- Trigger: cross-review (codex, cycle 1/2) returned one ACTION_REQUIRED
  (`docs/reports/cross-review-triage-codex-seat-permission-verification.md`
  P2/AR-1): the recipe's `send`/`wait`/`stop`/`disband`/`status` follow-up
  commands dropped `--org-id`/`--config`/`--state-dir`, so they either fail
  outright or silently target the default state directory. User decision:
  fix then re-run the full pipeline. Fix landed at `33158e2`. The cycle-2
  self-review (`docs/reports/self-review-2026-09-18-codex-seat-permission-verification.md`,
  "Cycle 2" section) then found and fixed four further LOWs at `f769f40`
  (undefined `task.txt` in the recipe's typed-protocol example, `send`/`wait`
  fenced together hiding the composer-trap warning, the cleanup `status`
  line missing `--config`, two dangling-word line wraps in the recipe and in
  `scripts/ralph-config.sh` + its template mirror). `git status --porcelain`
  was clean before and after this cycle-2 pass; nothing was edited or
  committed here.

**(a) AC-3 / AC-4 re-confirmed at `d40c6d2`.**

```
cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md   # identical
cmp docs/recipes/codex-setup.md templates/base/docs/recipes/codex-setup.md                          # identical
cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh                                  # identical
cmp .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md                                          # identical
cmp .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md                           # identical
cmp .claude/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md                           # identical
grep -c 'tech-debt' templates/base/ralph.toml internal/config/config.go internal/org/permissions.go  # 0/0/0
grep -c '暫定制約' .claude/skills/org/SKILL.md                                                       # 0
./scripts/check-sync.sh          # PASS, IDENTICAL 158, DRIFTED 0
./scripts/check-skill-sync.sh    # [ok] 13 skill(s) in lock-step
./scripts/check-template-purity.sh   # PASS
```

No regression on either AC: the cycle-1 findings the fix commits touched
(the recipe and `scripts/ralph-config.sh`) are still byte-identical between
root and `templates/base/`, and the cycle-1 tech-debt-pointer/`暫定制約`
removals in the Go doc comments and the skill are untouched by cycle 2
(neither fix commit touches `internal/config/config.go`,
`internal/org/permissions.go`, `templates/base/ralph.toml`, or the skill
files — confirmed via `git diff 3dab6bc...HEAD --stat`, which lists only the
recipe pair, `scripts/ralph-config.sh` + template, the evidence/plan/report
files, and the two new report files).

**(b) Recipe command lines re-verified against the CLI and the protocol
package.** `--org-id`, `--config`, and `--state-dir` are declared as
`cmd.PersistentFlags()` on the `org` root command
(`internal/cli/org.go:40-42`), so every subcommand inherits them, including
`send`/`wait`/`stop`/`disband`/`status` — the recipe's claim that they are
required/accepted on every follow-up verb is accurate, not just true for
`spawn`. Per-verb flags used in the recipe all exist: `send` has `--to`,
`--text`, `--timeout-ms` (`internal/cli/org.go:377-379`); `wait` has
`--seat`, `--until`, `--timeout-ms` (`:426-428`). The quoted failure text
`org: --org-id is required` is the literal string in `requireOrgID`
(`internal/cli/org.go:69-74`). `internal/org/protocol/protocol.go` confirms
the enum (`TypeTask`/`TypeResult`/... , `internal/org/protocol/protocol.go:25-35`),
that `TASK` is in `taskIDRequiredTypes` (`:56-59`), and
`DefaultMaxBodyChars = 2000` (`:43`) — matching the recipe's new `task.txt`
example line-for-line (`TYPE: TASK` / `TASK_ID: t-1` / blank line / body,
"TYPE must be one of the protocol's enum values, TASK needs a TASK_ID, and
the body is capped at 2,000 characters"). `newOrgStatusCmd` also calls
`newOrgRuntime(cmd, *stateDir, *configPath)` (`internal/cli/org.go:515`), so
"`status` does load the config" (self-review cycle-2, finding LOW row 3) is
correct — the cleanup `status` line's now-added `--config` is consistent
with its neighbours rather than merely harmless.

**(c) `scripts/ralph-config.sh` / template byte-identity and lock-step
value (static half).** `cmp` confirms byte-identity (above). The
lock-step *test* (`go test ./internal/config/ -run TestDefaultsLockStep`)
was not run here — that is `/test`'s call, per the task handoff. Relying on
the static verifier and a direct code read instead:
`RALPH_ORG_PERMISSIONS_CODEX_VERIFIED="${RALPH_ORG_PERMISSIONS_CODEX_VERIFIED:-false}"`
is unchanged by either cycle-2 fix commit (only the surrounding comment was
reflowed/reworded at `c357cac` and re-wrapped at `f769f40`); `templates/base/ralph.toml`
still has `codex_verified = false`
(`grep -n 'codex_verified = false' templates/base/ralph.toml` → 1 hit); and
`internal/config/config.go`'s `CodexVerified bool` field default and
`TestDefaultsLockStep`'s check at `internal/config/defaults_sync_test.go:158-159`
compare exactly these three value sources — none of which changed value in
cycle 2, only comment text. The three-way lock-step therefore still holds at
the value level; `/test` still owns re-running the test itself.

**(d) Static verifier re-run at `d40c6d2`.**

```
$ ./scripts/run-static-verify.sh
==> scripts/check-sync.sh            PASS (IDENTICAL 158, DRIFTED 0)
==> scripts/check-pipeline-sync.sh   OK
==> scripts/check-skill-sync.sh      [ok] 13 skill(s) in lock-step
==> scripts/check-template-purity.sh PASS
==> golang verifier: gofmt: ok / go vet: clean / golangci-lint: 0 issues.
==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-09-18-085927.log
```

Scope classifier picked `scripts/ralph-config.sh` as the changed file this
cycle (comment-only shell edit) and still ran the full golang verifier via
the "full fallback" path — same behavior as cycle 1, no scope regression.
`go test` was not reachable in this run for the same structural reason
documented in cycle 1 (`run-static-verify.sh` forces
`HARNESS_VERIFY_MODE=static`; `packs/languages/golang/verify.sh` gates
`go test ./...` behind a `mode == test` branch `run_static()` never enters).

**(e) Documentation drift.** Read
`docs/reports/cross-review-triage-codex-seat-permission-verification.md`
end to end and the plan's cycle-2 deviation-note entries
(`docs/plans/active/2026-09-18-codex-seat-permission-verification.md`,
"cross-review(cycle 1)" and "self-review(cycle 2)" bullets) against the
actual commits:

- The triage report's Decision line ("AR-1 fixed in `33158e2`: the recipe's
  `send` / `wait` / `stop` / `disband` / `status` commands now carry the
  matching `--org-id`, `--config`, and `--state-dir`... Cycle counter
  advanced to 2/2") matches `git show 33158e2 --stat` (touches only the
  recipe pair, +62/-20) and the recipe content read in (a)/(b) above.
- The triage report's closing sentence — "the re-run's results are recorded
  as a cycle-2 section below when it completes" — has no cycle-2 section in
  that file yet. This is **not drift**: cross-review is the pipeline step
  *after* sync-docs (`/self-review → /verify → /test → /sync-docs →
  /cross-review → /pr`), so cycle-2 cross-review has not run yet at this
  point in the pipeline; the sentence is a forward reference to a step this
  verify pass does not reach, not an unfulfilled promise.
- The plan's cross-review(cycle 1) deviation note names the fix commit
  (`33158e2`) and its effect correctly (verified against `git show 33158e2`
  above). The plan's self-review(cycle 2) deviation note names four LOW
  findings and the fix commit `f769f40`; all four are present and correctly
  described in the self-review report's "Cycle 2" section and independently
  confirmed against `git show f769f40` in (a) above — no mismatch between
  what the plan claims was fixed and what the commit actually changed.
- No other documentation surface (recipe, skill, `ralph.toml`,
  `internal/config/config.go`, `internal/org/permissions.go`) was touched
  by either cycle-2 fix commit, so none of the cycle-1 doc-drift conclusions
  need revisiting.

## Cycle-2 verdict

| AC | Verdict |
| --- | --- |
| AC-1 | PASS (unchanged since cycle 1; not touched by cycle-2 fixes) |
| AC-2 | PASS (unchanged since cycle 1; not touched by cycle-2 fixes) |
| AC-3 | PASS — recipe/template and `codex-setup.md`/template still byte-identical; `check-sync.sh` DRIFTED 0; the AR-1 fix and the four cycle-2 LOW fixes make the recipe's own commands internally runnable (correct flags throughout, a defined `task.txt`, the composer-trap warning ahead of `wait`, and a consistent cleanup block) |
| AC-4 | PASS — tech-debt pointers and `暫定制約` remain absent everywhere checked; all skill mirrors and `scripts/ralph-config.sh`/template still byte-identical; no cycle-2 commit touched the Go doc comments or `templates/base/ralph.toml`, so cycle-1's PASS there is unaffected |
| AC-5 | PASS (static half) — `./scripts/run-static-verify.sh` green at `d40c6d2`; `go test` deferred to `/test` by design, not a gap in this check |
| AC-6 | PASS (verify-relevant half, unchanged); `Closes #155` still correctly deferred to `/pr` |

**Overall cycle-2 verdict: PASS.** No inaccurate claim found in the triage
report or the plan's cycle-2 deviation notes. No regression to any cycle-1
AC. The fix commits are scoped exactly as described (recipe pair +
`scripts/ralph-config.sh` pair; no source logic, no Go doc comments, no
skill files touched), and their technical claims about the CLI and protocol
package hold up against the current code.

