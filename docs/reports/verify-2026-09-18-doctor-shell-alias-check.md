# Verify report: doctor-shell-alias-check

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-doctor-shell-alias-check.md` (issue #162)
- Verifier: `verifier` subagent (Claude Code, standard flow)
- Branch: `feat/doctor-shell-alias-check`, HEAD `22927b1` (base `main`)
- Scope: spec compliance (AC-1..AC-6) + static analysis via `./scripts/run-static-verify.sh` (changed-language scope: Go + markdown/skills). No behavioral tests run (`/test` owns that); no source/plan/doc edits made.
- Pipeline cycle: 1 (of cap 2).

## Overall verdict: PASS

AC-1 through AC-5 are met. AC-6 (`Closes #162`) is out of scope for `/verify`
by design — it lands at `/pr`. Static analysis is fully green. Every clause
in the revised AC-2/AC-3 text is pinned by a named test, and the N1–N6
revalidation fixes recorded in the plan's Deviation notes (landed in
`33e5bad`/`4352419` with no second reviewer pass) were independently
cross-checked against the current code and hold up. One small precision gap
in AC-3's "every Detail lists scanned files" claim and two minor
doc-simplification notes are recorded below as non-blocking.

## Static analysis

`git status --porcelain` was clean before and after (nothing to lose; only
gitignored `docs/evidence/*.log` files were produced).

```
$ ./scripts/run-static-verify.sh
==> shellcheck hook + verify scripts                 OK
==> sh -n (11 hook scripts, root + template)          OK (all 22)
==> jq -e . .claude/settings.json                     OK
==> jq -e . templates/base/.claude/settings.json      OK
==> Codex hook single-source guard                    OK
==> Codex inline hook detector smoke test             OK
==> Codex PR provenance policy guard                  OK
==> scripts/check-sync.sh            PASS (IDENTICAL 158, DRIFTED 0, TEMPLATE_ONLY 11, KNOWN_DIFF 5)
==> scripts/check-pipeline-sync.sh   OK
==> scripts/check-skill-sync.sh      [ok] 13 skill(s) in lock-step
==> scripts/check-template-purity.sh PASS
==> Language scope: changed (changed_languages) -> golang
==> golang verifier: gofmt: ok / go vet: clean (silent) / golangci-lint: 0 issues. / staticcheck: clean (silent, confirmed on PATH)
==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-09-18-133519.log
```

`go test` was **not** executed — `run-static-verify.sh` forces
`HARNESS_VERIFY_MODE=static`, and `packs/languages/golang/verify.sh` gates
`go test ./...` behind a separate `mode == test` branch that `run_static()`
never reaches (confirmed by reading the script, not by inference). staticcheck
is present on `PATH` (`staticcheck 2026.1 (v0.7.0)`) and produces no output on
a clean pass, consistent with its normal silent-success behavior; its absence
from the log is not evidence it didn't run.

## Per-AC results

**AC-1 — Check registration (PASS).**
`grep -c 'checkShellAliases' internal/cli/doctor.go` = 2 (the call site at
`doctor.go:129` plus the "Check 8b" doc comment at `doctor.go:124-129` that
names the function). The registration sits immediately after Check 8 (herdr,
`doctor.go:120-122`), commented as "Check 8b", and passes
`herdrResult.Status == "pass"` — the named variable from Check 8, exactly as
specified.

**AC-2 — detection clauses, all pinned by named tests (PASS).**

| AC-2 clause | Pinning test(s) |
| --- | --- |
| codex warn Detail has `file:line` (home as `~`), flag name, recipe path | `TestCheckShellAliases_CodexModelFlag_Warn` |
| `-m value` / `-mvalue` / `--model=value` forms | `TestCheckShellAliases_ModelFlagForms_Warn`, `TestCheckShellAliases_CodexModelFlag_Warn` (space form) |
| `-s value` / `-svalue` forms | `TestCheckShellAliases_CodexShortFlags_Warn` (`sandbox short flag with space` / `sandbox short flag concatenated`) |
| `-a value` form | `TestCheckShellAliases_CodexShortFlags_Warn` (`ask-for-approval short flag`) |
| trailing-comment `--model` not detected | `TestCheckShellAliases_TrailingComment_NotMisreadAsFlag` |
| one alias statement with `codex=` and `claude=`, each attributed correctly | `TestCheckShellAliases_TwoAliasesOnOneStatement_EachAttributedCorrectly` |
| symlink case → exactly 1 finding | `TestCheckShellAliases_SymlinkedRc_DedupedToOneFinding` |
| `ZDOTDIR` case → detected | `TestCheckShellAliases_Zdotdir_Detected` |

No AC-2 clause is left without a pinning test. Supporting unit-level tests
(`TestShellAliasConflictingFlags`, `TestShellAliasStatements`) additionally
pin the flag-token boundary rules (`--models-dir`/`--m` non-matches, bare
`-m`, `--m` non-match) and the statement/quote/continuation-line reader.

**AC-3 — severity matrix, all pinned by named tests (PASS).**

| AC-3 clause | Pinning test(s) |
| --- | --- |
| herdr + codex finding → `warn` | `TestCheckShellAliases_CodexModelFlag_Warn`, `TestCheckShellAliases_CodexModelOnly_...`, `TestCheckShellAliases_CodexPermissionOnly_...`, `TestCheckShellAliases_CodexModelAndPermission_...` |
| herdr + claude `--permission-mode` finding → `warn` | `TestCheckShellAliases_ClaudePermissionMode_WarnsWhenHerdrPresent` |
| herdr-absent finding → `info` | `TestCheckShellAliases_HerdrAbsent_CodexConflictDowngradesToInfo`, `TestCheckShellAliases_ClaudePermissionMode_HerdrAbsent_InfoWithSuffix` |
| claude `--model`-only finding → `info` (always, herdr-independent) | `TestCheckShellAliases_ClaudeModelFlag_Info`, `TestCheckShellAliases_ClaudeModelOnly_InfoWithSeatStartsClauseNotGuardedClause` |
| unparseable alias (`not fully parsed`), no other finding → `info` | `TestCheckShellAliases_UnterminatedQuoteAtEOF_NotFullyParsed` |
| no alias / no conflict → `pass`, harmless alias named by `file:line` | `TestCheckShellAliases_NoRcFiles_Pass`, `TestCheckShellAliases_HarmlessAlias_Pass` |
| home unresolvable → `info` | `TestCheckShellAliases_EnvResolutionError_Info` |
| unreadable rc, no other finding → `info` | `TestCheckShellAliases_UnreadableRc_Info` |
| every Detail lists scanned files | `TestCheckShellAliases_PassDetail_NamesScannedFiles`, `TestCheckShellAliases_OverLongLine_..._ReportsPartiallyRead` (see gap below) |
| not counted in `countFailed` | `checkShellAliases` has no `"fail"` branch anywhere in `internal/cli/doctor_shell_alias.go` (read directly — only `pass`/`info`/`warn` are assignable to `r.Status`); the pre-existing `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` (`internal/cli/doctor_org_test.go:186`) is unmodified by this diff, so its exit-code assertion is unaffected by this check's insertion order |

**Minor gap (non-blocking):** the home-unresolvable early-return path
(`doctor_shell_alias.go:498-503`) does **not** append a "scanned N shell rc
file(s)" clause — it returns immediately with only
`"could not resolve home directory: ... — shell alias check skipped"`. AC-3's
blanket clause "どの Detail も走査したファイルを列挙する" is therefore true
for every Detail *after* scanning starts, but not for the one precondition
failure that happens before any candidate list can even be built. This reads
as a reasonable, self-explanatory exception (there is nothing to name when
home can't be resolved) rather than a bug, and `TestCheckShellAliases_EnvResolutionError_Info`
pins the actual behavior — flagging it here only because the AC text doesn't
carve out the exception explicitly.

**N1–N6 revalidation fixes (self-review "Revalidation" section, landed in
`33e5bad`/`4352419`) — cross-checked against current code, all confirmed:**

- **N1** (severity/wording split by flag class, not driver): confirmed —
  `shellAliasFlagIsPermission` (`doctor_shell_alias.go:393-400`) and the
  `codexHasModel`/`codexHasPermission`/`claudeHasModel`/`claudeHasPermission`
  branches (`:552-611`) produce exactly the model-only / permission-only /
  both-clauses wording pinned by `TestCheckShellAliases_CodexModelOnly_...`,
  `TestCheckShellAliases_CodexPermissionOnly_...`,
  `TestCheckShellAliases_CodexModelAndPermission_...`. claude
  `--permission-mode` is warn (`:657`); claude `--model`-only stays info.
- **N2** (statement splitting, continuation lines, `not fully parsed`):
  confirmed — `shellAliasStatements`/`parseAliasStatements` split on
  `; | &`, recognize `then`/`else`/`do`/`{`/`builtin` markers, and
  `scanShellAliasFile` joins up to 32 continuation lines before giving up
  (`:340-381`); pinned by `TestCheckShellAliases_SecondStatementOnSameLine_Detected`,
  `TestCheckShellAliases_AliasAfterAndAnd_Detected`,
  `TestCheckShellAliases_AliasInsideThenClause_Detected`,
  `TestCheckShellAliases_ContinuedDoubleQuotedValue_WarnsAtFirstLine`,
  `TestCheckShellAliases_ContinuedSingleQuotedValueAcrossThreeLines_Warns`,
  `TestCheckShellAliases_UnterminatedQuoteAtEOF_NotFullyParsed`.
- **N3** ("partially read" vs "could not read", scanned-count includes
  partial reads): confirmed — `checkShellAliases:519-531` splits
  `couldNotRead`/`partiallyRead` and counts `partiallyRead` files into
  `scannedOK`; pinned by
  `TestCheckShellAliases_OverLongLine_KeepsPartialFindingsAndReportsPartiallyRead`.
- **N4** ("claude に短縮形はない" → scoped to the two named flags): confirmed
  — the doc comment on `shellAliasConflictingFlags` (`:418-420`) now reads
  "claude has no short form for either `--model` or `--permission-mode`",
  not a blanket claim.
- **N5** (`shellAliasWord` → `shellAliasAssignment`): confirmed — the type is
  `shellAliasAssignment` (`:231-234`); no `shellAliasWord` identifier remains
  (`grep -c shellAliasWord internal/cli/*.go` = 0).
- **N6** (stale Assumptions text): confirmed — the plan's current
  Assumptions section (lines 42-48) describes the shell-word-reader approach
  actually implemented, with no leftover reference to a regex-based reader.

**AC-4 — documentation sync (PASS).**

```
cmp .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md       # identical
cmp .agents/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md       # identical
cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md   # identical
./scripts/check-skill-sync.sh     # [ok] 13 skill(s) in lock-step
./scripts/check-sync.sh           # PASS, IDENTICAL 158, DRIFTED 0
./scripts/check-template-purity.sh   # PASS
```

Both the `/org` skill's prerequisites bullet and the recipe's prerequisites
bullet now state that `ralph doctor`'s "Shell aliases (codex/claude)" check
reports offending aliases by `file:line`, and both replace issue #162's
original "claude も同じ形で衝突するはず(未検証)" with the CLI-measured
result (claude accepts a repeated flag; last value wins) — matching the
Deviation notes' "CLI 実測" entry and the Design decisions section verbatim
in substance. `docs/evidence/codex-seat-permissions-2026-09-18.md` gained the
one-line addendum recorded in the plan's Deviation notes
(`git diff main...HEAD -- docs/evidence/codex-seat-permissions-2026-09-18.md`
shows exactly that one `+` line). `README.md`'s doctor description
(`README.md:127`) stays generic by the plan's own explicit Scope-5 decision
("sync-docs が判断") — not a gap for `/verify`, and no other doc in
`README.md`/`AGENTS.md`/`docs/quality/` enumerates individual doctor checks
that would need updating (`grep -rn 'Shell alias' README.md AGENTS.md
docs/quality/` → 0 hits besides the two files already checked).

**Doc-simplification note (non-blocking):** both the skill bullet and the
recipe bullet summarize severity as "info when a claude alias only adds
`--model`, warn otherwise" (recipe) / "herdr があり... claude の
`--permission-mode` の所見があれば warn" (skill), which is accurate given
each doc's own stated context (`/org` skill and the seat-permissions recipe
both already assume herdr is installed and passing — the recipe's own
Prerequisites bullet says "`ralph doctor` passes for codex, herdr, and
agmsg"). Taken out of that context the wording elides the `herdrPresent`
precondition for a codex-only finding (codex found + herdr absent is still
`info`, not `warn`) — worth a one-clause tightening if either doc is
revisited for an audience that reads it without the herdr assumption, but
not a factual error within its stated scope.

**AC-5 — static build clean, local evidence consistent with code (PASS,
static half; `go test`/`./scripts/run-verify.sh` deferred to `/test` by
design).** `gofmt`/`go vet`/`golangci-lint`/`staticcheck` are all clean (see
Static analysis above). The plan's final AC-5 evidence entry (Deviation
notes, "2026-09-18 work: AC-5 の実機 evidence(最終)") was cross-checked
word-for-word against the current `checkShellAliases` string-building logic
(`doctor_shell_alias.go:575-654`):

- codex clause "`ralph org spawn always passes --model, so every spawn
  fails`" = the exact literal at `:582`.
- claude clause "`claude accepts a flag given twice and the last value
  wins`" / "`ralph org spawn always passes --model after the alias, so its
  value applies and the seat still starts`" / "`the alias's other flags
  reach every seat`" = the exact literals at `:597`, `:600`, `:606`.
- trailing "`scanned 4 shell rc file(s): ... (files they source are not
  followed)`" = the exact `scannedClause` format at `:638-639`.

This is a deterministic code-vs-doc-text comparison, not an independent
re-run of the binary on this machine — I did not execute `go run ./cmd/ralph
doctor` myself (that would read the verifier's own machine's shell rc files,
outside this task's scope and not requested by the handoff). The earlier,
now-superseded Deviation notes entry (line 120, "改訂後", predating the
Slice D/N1 fix) still contains the pre-fix wording ("so ralph org spawn
fails" without the "every"/"-with-alias" phrasing) — this is expected: plan
Deviation notes are an append-only running log, and the "(最終)" entry is
explicitly marked as the superseding one.

**AC-6 — PENDING /pr.** `Closes #162` is correctly absent from this branch;
that line belongs in the PR description, produced at `/pr`.

## Documentation-drift check

- `doctor.go`'s "Check 8b" comment (`:124-129`) — "with herdr, codex findings
  and claude `--permission-mode` findings warn, and a claude `--model`
  finding stays informational" — matches the `checkShellAliases` severity
  switch (`:656-663`) exactly; this comment was itself one of the N-series
  fixes (commit `4352419`, "correct the Check 8b comment"), confirmed
  correct as it now stands.
- `checkShellAliases`'s own doc comment (`:477-494`) and the Design
  decisions section of the plan describe the same driver/flag-class matrix
  in the same terms (ralph always passes `--model`; permission-class flags
  only on edits/autonomous seats; codex rejects a repeated flag, claude
  accepts it).
- `shellAliasConflictingFlags`'s doc comment (`:402-425`) states the
  codex-cli 0.154.0 / claude 2.1.274 measurements cited in the plan's Design
  decisions section verbatim (short-flag collision behavior, `--model`/
  `--permission-mode` last-value-wins).
- No stray reference to the abandoned regex-based line reader (original
  Scope item 1) remains in code, tests, or the plan's current prose — all
  superseded by the word-reader design (plan's "改訂" paragraph, item 1).

## Known gaps / informational carryover (non-blocking)

- The home-unresolvable Detail path does not name a "scanned files" clause
  (see AC-3 gap above) — self-explanatory exception, not a bug.
- The skill/recipe severity summaries elide the `herdrPresent` precondition
  for codex findings (see AC-4 doc-simplification note above) — accurate
  within each doc's stated context, worth tightening only if that context
  assumption is ever dropped.
- `go test ./internal/cli/...`, the local `ralph doctor` re-run, and
  `./scripts/run-verify.sh` are `/test`'s responsibility per the pipeline
  contract; not run here.

No CRITICAL/HIGH-equivalent finding, inaccurate claim, or AC gap was found
beyond the two non-blocking notes above.

## Evidence commands run

```
grep -n 'checkShellAliases' internal/cli/doctor.go                              # 2 hits (call + comment)
grep -n 'Check 8\|herdrResult\|Shell aliases' internal/cli/doctor.go             # registration order confirmed
grep -n '^func Test' internal/cli/doctor_shell_alias_test.go                    # 40 tests
grep -c shellAliasWord internal/cli/*.go                                        # 0
cmp .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md       # identical
cmp .agents/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md       # identical
cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md   # identical
./scripts/check-skill-sync.sh                                                   # [ok] 13 skill(s) in lock-step
./scripts/check-sync.sh                                                         # PASS, DRIFTED 0
./scripts/check-template-purity.sh                                              # PASS
./scripts/run-static-verify.sh                                                  # PASS, evidence: docs/evidence/verify-2026-09-18-133519.log
git diff main...HEAD -- docs/evidence/codex-seat-permissions-2026-09-18.md      # +1 line addendum only
git status --porcelain                                                          # clean before and after
```

## Cycle 2 (post cross-review fix + cycle-2 self-review)

- Date: 2026-09-19
- HEAD: `c8cd863` (was `0c22388` at cycle-1 verify; 15 commits since)
- Trigger: cross-review (cycle 1) found 4 issues (2 ACTION_REQUIRED: AR-1
  missing `.zprofile`/`.zlogin` candidates and the `$ZDOTDIR` directory set,
  AR-2 the codex sandbox/approval clauses conflated two different flag
  classes; 2 WORTH_CONSIDERING: WC-1 quoted flag names inside an alias value,
  WC-2 silently dropping a candidate whose `os.Stat` fails for a reason other
  than not-exist). User decision: fix all four and re-run the full pipeline
  (cycle 2/2). Landed in `c5d646f` (cross-review fixes) + `c8e20e5` (stale
  seam-comment cleanup). The cycle-2 self-review (`cd0e010`, "Cycle 2"
  section) then found and fixed 1 MEDIUM regression + 5 LOW in the same
  cycle, landed in `3783610` — **no reviewer pass ran on `3783610`**, so it
  was checked against the plan and the C2 findings with extra care below.
  `git status --porcelain` was clean before and after this cycle-2 pass;
  nothing was edited or committed here beyond this report and its insight
  event.

**Overall cycle-2 verdict: PASS.** AC-1 through AC-5 remain met (AC-6 still
correctly deferred to `/pr`). Static analysis is fully green. Every clause in
改訂 items 8–10 is pinned by a named test. One LOW-equivalent documentation-
precision finding is recorded below (not introduced by a code defect, and not
tied to any AC) — non-blocking.

### Static analysis (re-run at `c8cd863`)

```
$ ./scripts/run-static-verify.sh
==> scripts/check-sync.sh            PASS (IDENTICAL 158, DRIFTED 0)
==> scripts/check-pipeline-sync.sh   OK
==> scripts/check-skill-sync.sh      [ok] 13 skill(s) in lock-step
==> scripts/check-template-purity.sh PASS
==> golang verifier: gofmt: ok / go vet: clean (silent) / golangci-lint: 0 issues. / staticcheck: clean (silent)
==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-09-19-015105.log
```

`cmp .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md`,
`cmp .agents/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md`,
and `cmp docs/recipes/codex-seat-permissions.md
templates/base/docs/recipes/codex-seat-permissions.md` are all identical.
`go test` was not run (same structural reason as cycle 1: `run-static-verify.sh`
forces `HARNESS_VERIFY_MODE=static`, and `packs/languages/golang/verify.sh`
gates `go test ./...` behind a `mode == test` branch `run_static()` never
enters) — `/test`'s job per the pipeline contract.

### AC re-confirmation

**AC-1 — unaffected.** `internal/cli/doctor.go` is not in this cycle's diff
(`git diff 0c22388..HEAD --stat` lists no `doctor.go` entry); the cycle-1
PASS carries forward unchanged.

**AC-2 / AC-3 — still met, expanded coverage.** The severity matrix and
Detail-content clauses from cycle 1 are unchanged in substance (the switch at
`doctor_shell_alias.go:847-854` is byte-identical to cycle 1's). The new
flag-class split (codex sandbox vs. approval, tracked separately per AR-2)
adds two more warn sub-cases, both pinned:

| New clause (AR-2 / plan Design decisions) | Pinning test(s) |
| --- | --- |
| codex `--sandbox`-only → sandbox clause, no "every spawn fails", no approval clause | `TestCheckShellAliases_CodexSandboxOnly_WarnsWithSandboxClauseOnly` |
| codex `--ask-for-approval`-only → approval clause ("to autonomous seats only... edits and guarded seats silently run"), not the sandbox wording | `TestCheckShellAliases_CodexApprovalOnly_WarnsWithApprovalClauseOnly` |
| all three flag classes (model, sandbox, approval) → three clauses in that order | `TestCheckShellAliases_CodexAllThreeFlagClasses_ClausesInOrder` |

改訂 items 8–10, each mapped to named tests:

| 改訂 item | Content | Pinning test(s) |
| --- | --- | --- |
| 8 (AR-1) | zsh candidates = `.zshenv`/`.zprofile`/`.zshrc`/`.zlogin` under each of `$ZDOTDIR` (if absolute), `~`, `~/.config/zsh`; bash adds `.bash_login` | `TestCheckShellAliases_ZdotdirZprofile_Warn`, `TestCheckShellAliases_ZdotdirZlogin_Warn`, `TestCheckShellAliases_ConfigZshZprofile_WarnsWhenZdotdirUnset`, `TestCheckShellAliases_BashLogin_Warn`, `TestCheckShellAliases_Zdotdir_Detected`, `TestCheckShellAliases_ZdotdirSymlinkedToHomeRc_ReportsZdotdirPath` (new candidate-order dedup) |
| 9 (WC-2) | a candidate `os.Stat` failure other than not-exist reports its parent dir once as "could not read", → `info`; a not-a-directory component (`ENOTDIR`) stays a silent skip, not a report | `TestCheckShellAliases_InaccessibleConfigZshDir_InfoOnce`, `TestCheckShellAliases_ConfigIsRegularFile_PassNoShellRcFileFound` |
| 10 (WC-1) | the alias value is re-tokenized with the same quote-aware reader before flag matching, so a quoted flag name/letter inside the value is still caught | `TestCheckShellAliases_QuotedFlagNameInValue_Warn`, `TestCheckShellAliases_QuotedPermissionModeInValue_Warn`, `TestCheckShellAliases_QuotedShortFlagInValue_Warn`, `TestShellAliasValueTokens` |

No 改訂-8/9/10 clause is left without a pinning test.

**N1–N6 (cycle 1) still hold** — unaffected by this cycle's diff (confirmed
by reading `git diff 0c22388..HEAD -- internal/cli/doctor_shell_alias.go`;
the N-series wording/logic is carried forward, only the sandbox/approval
split and the `codex_verified` mention were added inside it).

**Cycle-2 self-review findings (`cd0e010`, landed in `3783610`,
un-reviewed) — cross-checked against the code, all confirmed correct:**

- **C2-1** (newline/CR inside a joined value must act as whitespace, not
  glue onto the token): confirmed — `shellAliasStatements`'s whitespace case
  now includes `\n`/`\r` (`doctor_shell_alias.go:202`), with a doc comment
  (`:152-155`) that names the alternative reading (the short-flag
  concatenation rule accidentally accepting a newline) as the bug this fixes,
  not evidence the old test already covered it. Pinned by
  `TestCheckShellAliases_ContinuedSingleQuotedValueAcrossThreeLines_Warns`
  (its own comment documents the red/green distinction) and the new
  `TestCheckShellAliases_ContinuedSingleQuotedValueAcrossTwoLines_WarnsAtFirstLine`.
- **C2-2** (an unclosed alias statement that names no codex/claude
  assignment must still be reported "not fully parsed"): confirmed —
  `shellAliasFileScan.Unclosed` is now populated independent of whether the
  statement produced any `Defs` entry (`:437-439`), and `checkShellAliases`
  folds `Unclosed` into `notFullyParsed` before the "no alias found" branch
  can fire (`:835,838-839`). Pinned by
  `TestCheckShellAliases_UnclosedAliasStatementWithoutCodexClaudeName_NotFullyParsed`
  and the continuation-cap variant
  `TestCheckShellAliases_UnclosedStatementAtContinuationCap_NamesFirstLineAndFindsLaterAlias`.
- **C2-3** (`#` inside the alias's own VALUE must not be treated as a
  comment, since herdr's interactive zsh pane has `interactivecomments`
  off): confirmed — `shellAliasStatements` gained a `stopAtComment`
  parameter; `parseAliasStatements` (rc-line level) passes `true`,
  `shellAliasValueTokens` (value-level) passes `false`
  (`doctor_shell_alias.go:287,513`), each with a doc-comment rationale.
  Pinned by `TestCheckShellAliases_HashInsideValue_StillWarns`; the
  pre-existing `TestCheckShellAliases_TrailingComment_NotMisreadAsFlag`
  (rc-line-level `#`) still passes unaffected, since its fixture's value
  never contains a `#`.
- **C2-4** (name the `codex_verified` gate in the codex sentence and the
  skill): confirmed in the Detail text — `shellAliasCodexSentence`'s sandbox
  and approval clauses both now read "(modes/a mode a codex seat gets only
  once `[org.permissions].codex_verified = true`)"
  (`doctor_shell_alias.go:618-623`), pinned by the `codex_verified = true`
  assertions in `TestCheckShellAliases_CodexSandboxOnly_...` and
  `TestCheckShellAliases_CodexApprovalOnly_...`. See the drift finding below
  for a wording-precision note on this specific addition.
- **C2-5** (stale "値は空白でトークン化" left in Assumptions): confirmed —
  the plan's Assumptions section (current text, line 49) now reads "値も同じ
  reader で語に分けてから判定する(Scope 改訂 10...)", matching
  `shellAliasValueTokens`; no leftover "空白でトークン化" phrase remains
  (`grep -c '空白でトークン化' docs/plans/active/2026-09-18-doctor-shell-alias-check.md` = 0).
- **C2-6** (extract the codex/claude sentence-building and Detail/scanned-
  clause logic out of `checkShellAliases`): confirmed —
  `shellAliasCodexSentence`, `shellAliasClaudeSentence`,
  `shellAliasScannedClause`, and `shellAliasDetail` are now standalone, pure,
  independently-tested functions (`doctor_shell_alias.go:606-698`); each has
  its own unit test (`TestShellAliasCodexSentence`,
  `TestShellAliasClaudeSentence`). `checkShellAliases` itself is
  substantially shorter than the plan's stated pre-fix count.

### Documentation-drift / precision finding (non-blocking, LOW-equivalent)

**The C2-4 addition slightly loosens established fail-closed terminology in
two places — worth a wording tightening, not a functional bug.**

- `internal/cli/doctor_shell_alias.go:479-483` (the `shellAliasFlagClass`
  doc comment, Go-internal, not user-facing): "...so on a default project no
  codex seat ever reaches edits or autonomous and every codex seat is
  guarded regardless of its configured mode."
- `.claude/skills/org/SKILL.md` (+3 mirrors, operator-facing): "codex 座席が
  edits / autonomous になれるのは `[org.permissions].codex_verified = true`
  のときだけで、既定の false では codex 座席はすべて guarded として動く。"

Verified against `internal/org/spawn.go:449-452` (not touched by this PR):
when `permissionArgsForDriver` returns an error (codex mode is
autonomous/edits and `CodexVerified` is false), the real path calls
`o.reject(p, permErr)` — the **whole spawn is rejected** (`SpawnOutcomeFailed`);
the seat never starts in any mode, it does not silently run capped to
guarded. The two new sentences above, read at face value ("is guarded
regardless of its configured mode" / "座席はすべて guarded として動く"),
can be misread as "the seat still starts, just capped to guarded" — the
opposite of what actually happens. The established, more precise phrasing
already exists elsewhere in the codebase, unchanged by this PR:
`internal/config/config.go:74` ("codex seats accept only guarded until the
operator sets CodexVerified") and the recipe's own, already-correct,
untouched first paragraph (`docs/recipes/codex-seat-permissions.md:3-4`,
"codex seats accept only `guarded`... and reject `autonomous` / `edits` with
a fail-closed error").

This does **not** affect the actual `ralph doctor` output an operator sees:
`shellAliasCodexSentence`'s own rendered clauses are careful and accurate —
"modes a codex seat **gets** only once `codex_verified = true`" (not "modes
it silently degrades from"). It also does not affect any AC (no AC text
makes this claim) or any test (no test asserts this specific sentence). I
read this as a LOW-equivalent documentation-precision finding worth a small
rewording (e.g. "...so a default project's codex seats can only successfully
spawn in guarded mode; requesting edits/autonomous fails the spawn outright
rather than running capped") if `doctor_shell_alias.go` or the skill is
touched again — not something that should trigger a third pipeline cycle by
itself.

### Team-lead's specific questions, answered

1. **Recipe not touched for the `codex_verified` gate — confirmed, and the
   existing text is the more accurate of the two.** The recipe's first
   paragraph (`docs/recipes/codex-seat-permissions.md:3-4`) already states
   the fail-closed-rejection precondition precisely ("reject `autonomous` /
   `edits` with a fail-closed error"), predating this PR entirely (not in
   `git diff 0c22388..HEAD`). The cycle-2 reviewer's judgment that no
   duplicate sentence was needed there is correct. If anything, the new
   sentences added to the Go doc comment and the skill (see the drift
   finding above) should have matched the recipe's own wording rather than
   introducing a looser paraphrase.
2. **AC-5 evidence — the recorded (Slice E) evidence remains consistent with
   the current code; no fresh evidence line is required.** Confirmed by
   reading `git show 3783610 -- internal/cli/doctor_shell_alias.go` in full:
   Slice F (the only commit after Slice E's real-machine run) does not touch
   `shellAliasRcCandidates` (the candidate list/order is unchanged) and does
   not change the `hasModel`-only branch of either
   `shellAliasCodexSentence`/`shellAliasClaudeSentence` — the only textual
   change it makes is adding the `codex_verified` clause to the
   sandbox/approval branches, which this machine's aliases (both `--model`-only, per `docs/evidence/codex-seat-permissions-2026-09-18.md`) never
   trigger. So the Slice E note (line 131: "引き続き warn(codex `:35`、
   claude `:34`)" plus the new 4-file scanned order) is not stale — it
   describes exactly what `c8cd863`'s code would still produce for this
   machine's two aliases. I did not re-run `go run ./cmd/ralph doctor`
   myself for the same reason as cycle 1 (out of scope, would read my own
   machine's shell rc).
3. **Candidate-list doc comment / code / plan 改訂-8 agreement — confirmed
   exact match.** `shellAliasRcCandidates`'s doc comment
   (`doctor_shell_alias.go:64-77`), its implementation (`:78-129`: `zshDirs
   := [ZDOTDIR(if absolute), Home, Home/.config/zsh]`, each with `.zshenv`/
   `.zprofile`/`.zshrc`/`.zlogin` in that order, then `.zsh_aliases`, then
   the bash files including the newly-added `.bash_login`, then fish), and
   the plan's 改訂 item 8 text all state the same file set and directory
   order with no discrepancy.

### Evidence commands run (cycle 2)

```
git log --oneline 0c22388..HEAD                                                 # 15 commits
git diff 0c22388..HEAD --stat                                                   # doctor.go absent (AC-1 unaffected)
grep -n '^func Test' internal/cli/doctor_shell_alias_test.go                    # 60 tests
git show 3783610 -- internal/cli/doctor_shell_alias.go                          # full diff read; candidate list untouched
git show 3783610 -- internal/cli/doctor_shell_alias.go | grep -c shellAliasRcCandidates   # 0 (function not touched)
grep -n 'permissionArgsForDriver\|codexEditsArgs\|codexAutonomousArgs\|CodexVerified' internal/org/permissions.go
sed -n '440,470p' internal/org/spawn.go                                         # confirms o.reject on permErr (spawn fails, not downgraded)
cmp .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md       # identical
cmp .agents/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md       # identical
cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md   # identical
./scripts/check-skill-sync.sh                                                   # [ok] 13 skill(s) in lock-step
./scripts/check-sync.sh                                                         # PASS, DRIFTED 0
./scripts/check-template-purity.sh                                              # PASS
./scripts/run-static-verify.sh                                                  # PASS, evidence: docs/evidence/verify-2026-09-19-015105.log
git diff 0c22388..HEAD -- docs/tech-debt/README.md                              # +1 row, config-aware-grading follow-up, accurate
git status --porcelain                                                          # clean before and after
```

### Cycle-2 verdict

| AC | Verdict |
| --- | --- |
| AC-1 | PASS (unaffected; `doctor.go` not touched this cycle) |
| AC-2 | PASS — all clauses including the new sandbox/approval split pinned by named tests |
| AC-3 | PASS — severity matrix unchanged in substance; new sub-cases pinned |
| AC-4 | PASS — 4-way skill `cmp` and recipe pair `cmp` both identical; `check-skill-sync.sh`/`check-sync.sh`/`check-template-purity.sh` all green |
| AC-5 | PASS (static half) — `gofmt`/`go vet`/`golangci-lint`/`staticcheck` all clean; Slice E's real-machine evidence still accurate for the current code (see above) |
| AC-6 | Still correctly deferred to `/pr` |

**Overall cycle-2 verdict: PASS.** No CRITICAL/HIGH-equivalent finding. One
LOW-equivalent documentation-precision note (C2-4's "guarded regardless of
configured mode" phrasing) is recorded above for an optional future
tightening; it does not block this cycle and does not affect any AC.
