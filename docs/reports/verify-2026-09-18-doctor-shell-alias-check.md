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
