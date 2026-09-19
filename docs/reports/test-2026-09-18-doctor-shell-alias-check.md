# Test report: doctor-shell-alias-check

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-doctor-shell-alias-check.md` (issue #162)
- Tester: `tester` subagent (Claude Code, standard flow)
- Branch: `feat/doctor-shell-alias-check`, HEAD `36bda6d` (base `main`)
- Scope: behavioral tests only (changed-language scope: Go). No static
  analysis, linting, or drift checks — those are `/verify`'s job and are
  already covered by `docs/reports/verify-2026-09-18-doctor-shell-alias-check.md`
  (PASS).
- Evidence: `docs/evidence/test-2026-09-18-doctor-shell-alias-check.log`

## Overall verdict: PASS

Every command below exited 0 with 0 failures and 0 skips. The hermeticity
probe confirms the `TestMain` seam (not `$HOME`) determines test behavior.
The eight built-binary edge-case fixtures all matched their expected
Detail text and severity exactly, including the two non-obvious negative
assertions (fixture c's codex sentence must *not* say "every spawn fails";
fixture e must *not* say "has no conflicting flags").

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh` (changed-language scope) | 27 shell suites + 8 Go packages | all | 0 | 0 |
| 2 | `go test ./internal/cli/ -run 'ShellAlias\|TestRunDoctorOpts' -count=1 -v` | 42 (Go `--- PASS` lines) | 42 | 0 | 0 |
| 3 | `go test ./internal/... -count=1` (regression) | 8 packages | 8 | 0 | 0 |
| 4a | `go test ./internal/cli/ -race -count=1` | package `internal/cli` | ok | 0 | 0 |
| 4b | `go test ./internal/cli/ -coverprofile=... -count=1` | package `internal/cli` | ok, 82.5% stmt cov | 0 | 0 |
| 5a | `go test ./internal/cli/ -run 'TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault' -count=1 -v` | 1 | 1 | 0 | 0 |
| 5b | `go test ./internal/cli/ -run 'TestRunDoctorOpts' -count=1 -v` under a fake `HOME` with a real alias-bearing `.zshrc` | 4 | 4 | 0 | 0 |
| 6 | Built-binary edge-case probes (a–h + exit-code check) | 9 checks | 9 | 0 | 0 |

### 1. `./scripts/run-test.sh`

Mode `test`, requested scope `changed`. Ran the full shell test corpus (27
`==> tests/test-*.sh` suites, e.g. `test-agent-phase-boundaries.sh` 44/44,
`test-self-review-scope.sh` 64/64, `test-terraform-gitignore.sh` 47/47,
`test-xreview-helpers.sh` 29/29) — every suite reported `FAIL: 0`, no
suite reported a nonzero fail count anywhere in the log
(`grep -E 'FAIL: [1-9]'` on the raw log returns nothing). Language-scope
detection selected `golang` (this task's Go-only diff) and ran the Go
verifier: all 8 packages `ok` (cached from the prior `/verify` run in this
same worktree). Final line: `All verifiers passed.` Evidence:
`docs/evidence/verify-2026-09-18-134000.log` (gitignored, produced by this
run).

### 2. `go test ./internal/cli/ -run 'ShellAlias|TestRunDoctorOpts' -count=1 -v`

42 `--- PASS` lines, 0 `--- FAIL`, 0 `--- SKIP`, package `ok` in 2.682s.
This is the plan's full item-4 test list ((a)–(k) plus the revision-era
additions) together with the pre-existing `TestRunDoctorOpts_*` integration
tests (`TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected`,
`TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault`,
`TestRunDoctorOpts_ShellAliasesCheck_WarnsThroughTheSeam`, and one more)
run together, confirming AC-2/AC-3's per-clause test list from the verify
report all still pass and that the check's insertion does not perturb the
pre-existing herdr exit-code test.

No `--- SKIP` lines appeared. That matches the plan's Risks note that the
chmod-000 unreadable-rc path is exercised through the injected
`resolveEnv`/`shellAliasEnvFromOS` seam and fixture files, not through a
skip-on-root/windows guard — this package has no such guard to trigger in
the first place (that description in the handoff referred to a general Go
testing pattern, not a specific test in this file; confirmed by grep —
no `t.Skip` appears in `doctor_shell_alias_test.go`).

### 3. `go test ./internal/... -count=1` (regression, uncached)

All 8 packages `ok`: `internal/cli` 39.724s, `internal/config` 0.999s,
`internal/insights` 1.277s, `internal/org` 11.173s, `internal/org/driver`
1.356s, `internal/org/protocol` 2.057s, `internal/scaffold` 2.338s,
`internal/upgrade` 3.097s. No regressions outside the touched package.

### 4. Race detector and coverage, `internal/cli`

`-race -count=1`: `ok`, 37.256s, no data races reported.

`-coverprofile -count=1`: `ok`, 82.5% of statements — up from the
81.1%/80.9% baselines recorded in earlier cycles for this package (this
plan adds a large, well-tested new file). Per-function coverage for
`internal/cli/doctor_shell_alias.go` (`go tool cover -func`):

| Function | Coverage |
| --- | --- |
| `shellAliasEnvFromOS` | 100.0% |
| `shellAliasRcCandidates` | 94.4% |
| `shellAliasStatements` | 98.5% |
| `parseAliasStatements` | 93.3% |
| `parseAliasWords` | 84.2% |
| `scanShellAliasFile` | 100.0% |
| `shellAliasFlagIsPermission` | 100.0% |
| `shellAliasConflictingFlags` | 100.0% |
| `shellAliasUnreadableReason` | 83.3% |
| `checkShellAliases` | 98.0% |

No function in this file is under 80% (the handoff's threshold for a named
gap), so there is no blocking coverage gap. The two lowest functions are
recorded here as non-blocking coverage notes, not findings:

- `parseAliasWords` (84.2%): reading the source
  (`internal/cli/doctor_shell_alias.go:277`), the untested branch is most
  likely the `i >= len(words)` early `return nil` — an alias statement
  whose tokens are entirely option flags (e.g. `alias -g codex=...` with
  nothing after `-g`) or the fish two-word form with no value token after
  the name (`alias codex` alone). No fixture in the test file constructs
  either shape.
- `shellAliasUnreadableReason` (83.3%): reading the source
  (`internal/cli/doctor_shell_alias.go:466`), this has three branches
  (`bufio.ErrTooLong`, `*fs.PathError`, generic `err.Error()` fallback);
  the fixture-driven tests exercise the permission-denied `*fs.PathError`
  path, so the untested branch is almost certainly the generic fallback
  (an error that is neither `ErrTooLong` nor a `PathError` — hard to
  construct from a real file read and not attempted here).

### 5. Hermeticity of the `doctorShellAliasEnv` seam

`TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault` — PASS in isolation
(0.43s / 0.42s across two runs).

Cross-check: created a fake `HOME` containing a real
`alias codex="codex -m x"` in `.zshrc`, then ran
`go test ./internal/cli/ -run 'TestRunDoctorOpts' -count=1 -v` twice — once
with the real developer `HOME` (baseline) and once with `HOME` pointed at
the fake directory (`ZDOTDIR=""`, `GOCACHE`/`GOMODCACHE` pinned to their
current values via `go env` to avoid a cold cache under the overridden
environment). Both runs produced the identical 4-test pass/fail/skip set
(`diff` on the sorted `--- PASS/FAIL/SKIP: <name>` lines was empty). This
confirms `TestMain`'s pin of `doctorShellAliasEnv` to a non-existent home
(`internal/cli/main_test.go`) — not the ambient `$HOME` — is what decides
`runDoctor*` test outcomes; the 13 pre-existing `runDoctor*` sites do not
read a developer's real rc files even when one exists and contains a
matching alias.

### 6. Built-binary edge-case probes

Built `go build -o <scratch>/ralph-bin ./cmd/ralph` (exit 0). herdr is
installed on this machine (`herdr 0.7.5` at `/opt/homebrew/bin/herdr`), so
codex findings are `warn` as expected throughout. For each fixture, a
fresh temp `HOME` was created under the scratch dir (never the real
`~`), the rc was written, and
`HOME=<dir> ZDOTDIR= <bin> doctor 2>&1 | grep 'Shell aliases'` was run
from the worktree root.

| Fixture | Expected | Actual | Match |
| --- | --- | --- | --- |
| a. `.zshrc`: `alias codex='codex'` | pass, `~/.zshrc:1`, "has no conflicting flags" | `pass — alias codex in ~/.zshrc:1 has no conflicting flags. scanned 1 shell rc file(s): ~/.zshrc ...` | yes |
| b. `.bashrc`: `alias claude='claude --model fable --effort xhigh'` | info (claude `--model` only), "the seat still starts" | `info — alias claude in ~/.bashrc:1 adds --model — ... so its value applies and the seat still starts; ...` | yes |
| c. `.zshrc`: `alias codex='codex -a never' claude='claude --permission-mode plan'` | warn; codex sentence has edits/autonomous+guarded clause, NOT "every spawn fails"; claude sentence has "a guarded seat runs with the alias's permission mode" | `warn — alias codex ... adds --ask-for-approval (-a) — ...ralph passes --sandbox / --ask-for-approval only to edits and autonomous seats, so those spawns fail while a guarded seat silently runs with the alias's value; ... alias claude ... adds --permission-mode — ... a guarded seat runs with the alias's permission mode; ...` | yes (literal string `every spawn fails` absent; guarded-clause text present verbatim) |
| d. `.config/fish/config.fish`: `alias codex "codex -mgpt-5.5"` | warn | `warn — alias codex in ~/.config/fish/config.fish:1 adds --model (-m) — ...` | yes |
| e. `.zshrc`: `alias codex='codex` (unclosed quote), nothing else | info, "not fully parsed: ~/.zshrc:1", no "has no conflicting flags" | `info — not fully parsed: ~/.zshrc:1 — the alias value continues past what was read. scanned 1 shell rc file(s): ~/.zshrc ...` | yes (phrase "has no conflicting flags" absent) |
| f. `.zshrc` mode 000 | info, "could not read: ~/.zshrc (permission denied)" | `info — could not read: ~/.zshrc (permission denied) — aliases there were not checked. scanned 0 shell rc file(s) ...` | yes |
| g. no rc file at all | pass, "no shell rc file found to scan" | `pass — no shell rc file found to scan` | yes |
| h. `.zshrc`: `# alias codex="codex -m x"` | pass, "no codex/claude alias found" | `pass — no codex/claude alias found. scanned 1 shell rc file(s): ~/.zshrc ...` | yes |

Fixture f's rc mode was restored to `644` after the probe (confirmed via
`stat -f '%Lp'`), so scratch-dir cleanup is unaffected.

Exit-code check: the full `doctor` run's process exit code is unaffected
by this check in every case — `echo $?` for fixture g (pass) and fixture c
(warn on both codex and claude findings) are both `0`. This is consistent
with `checkShellAliases` never producing a `"fail"` status (confirmed by
reading `internal/cli/doctor_shell_alias.go` — only `pass`/`info`/`warn`
are assignable to the result's Status), so `countFailed` never counts it.

No alias VALUE beyond the fixtures listed above was printed or recorded,
and the real `~/.zshrc` was never read (every probe used a scratch-dir
`HOME`).

## Coverage

- Statement (`internal/cli` package): 82.5% (`go test -coverprofile`).
- Function (`doctor_shell_alias.go`, all 10 functions): 83.3%–100.0%; see
  table in section 4 above. None under 80%.
- Branch: not separately instrumented by `go test`'s coverage tool; the
  per-function percentages above are the closest available proxy.
- Notes: coverage rose from the 81.1%/80.9% baselines recorded for
  `internal/cli` in earlier cycles (see tester memory), consistent with
  this plan adding ~665 lines of well-tested new code
  (`internal/cli/doctor_shell_alias.go`) plus ~851 lines of tests
  (`internal/cli/doctor_shell_alias_test.go`).

## Failure analysis

None. No test failed in any of the six execution steps.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` (herdr/agmsg-absent exit code) | still PASS, unaffected by this check's insertion order | step 2 log, `--- PASS` line present |
| Pre-existing `runDoctor*` tests reading real `$HOME` | confirmed still hermetic — identical pass/fail set under a fake `HOME` with a real matching alias vs. the developer's real `HOME` | step 5 log (`diff` empty) |
| Full package regression (`internal/...`) | all 8 packages `ok`, uncached | step 3 log |

## Test gaps

- `parseAliasWords`'s all-flags-no-name branch (an alias statement whose
  words are entirely leading option flags, e.g. `alias -g codex=...` with
  nothing following, or fish's two-word form with a name but no value
  token) has no fixture. Non-blocking (function is at 84.2%, well above
  the 80% floor) — worth a follow-up fixture if this file is touched
  again.
- `shellAliasUnreadableReason`'s generic `err.Error()` fallback (an error
  that is neither `bufio.ErrTooLong` nor `*fs.PathError`) is not
  exercised. Non-blocking; this branch is hard to trigger with a real
  file read in a unit test, which is presumably why it was left uncovered.
- The plan's Non-goal explicitly excludes recursive tracking of `source`d
  files and dynamic (interactive-shell) alias evaluation — both are
  deliberately out of scope and not test gaps, just documented
  limitations (Detail text says so in every finding, confirmed in the
  probe table above).
- Codex seat spawn with an actual conflicting alias was not exercised
  end-to-end (the plan's Non-goals explicitly excludes this — CLI-level
  behavior was already measured during `/plan`'s advisory and `/work`'s
  AC-5 evidence capture, not repeated here).

## Verdict

- Pass: 6/6 execution steps (27 shell suites, 42+4 targeted Go tests, 8/8
  package regression, race-clean, 9/9 edge-case probes)
- Fail: 0
- Blocked: 0

Tests pass. Proceeding to `/pr` is appropriate from a testing standpoint.

## Cycle 2 (2026-09-19)

- Plan: same, `docs/plans/active/2026-09-18-doctor-shell-alias-check.md`
  (issue #162)
- Tester: `tester` subagent (Claude Code, standard flow)
- Branch: `feat/doctor-shell-alias-check`, HEAD `812d5c4` (cycle-1 report
  commit was `9178954`; new commits since:
  `eadf0ac`/`e4e98ab`/`913735a`/`f1898d8`/`c5d646f`/`c8e20e5`/`3c8fa2b`/
  `cd0e010`/`56852c6`/`4d26aed`/`3783610`/`c8cd863`/`573e474`/`812d5c4`)
- Reason: cross-review cycle 1 found 4 issues (user chose fix-all +
  full-pipeline re-run); cycle-2 self-review found 6 more (all fixed);
  cycle-2 `/verify` is PASS (`573e474`). This is the cycle-2 `/test` pass
  (pipeline cycle 2 of cap 2).
- Scope: behavioral tests only, same as cycle 1. No code/plan/docs edits
  made here.
- Evidence: appended to the same
  `docs/evidence/test-2026-09-18-doctor-shell-alias-check.log` (gitignored),
  under a `CYCLE 2 (2026-09-19)` marker.

### Overall verdict: PASS

Same programme as cycle 1, re-run against the cycle-2 code
(`c5d646f`/`3783610` rewrote the candidate-file list, split codex's
sandbox/approval clauses, added a `codex_verified`-aware Detail, and fixed
the newline-inside-value / `#`-in-value / unclosed-statement-without-a-
name regressions that cross-review and cycle-2 self-review flagged).
Every command exited 0 with 0 failures and 0 skips. Both functions flagged
as sub-100%-but-above-80% in the cycle-1 report (`parseAliasWords`,
`shellAliasUnreadableReason`) are now at 100% — `eadf0ac` added the two
tests that close them. All 11 new fixtures (a–k) plus the exit-code check
matched their expected Detail text and severity exactly, and the three
cycle-1 regression fixtures re-run under (k) still match cycle 1's report
verbatim — no regression from the cycle-2 rewrite.

### Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh` (changed-language scope) | 27 shell suites + 8 Go packages | all | 0 | 0 |
| 2 | `go test ./internal/cli/ -run 'ShellAlias\|ParseAliasWords\|TestRunDoctorOpts' -count=1 -v` | 64 (Go `--- PASS` lines) | 64 | 0 | 0 |
| 3 | `go test ./internal/... -count=1` (regression) | 8 packages | 8 | 0 | 0 |
| 4a | `go test ./internal/cli/ -race -count=1` | package `internal/cli` | ok | 0 | 0 |
| 4b | `go test ./internal/cli/ -coverprofile=... -count=1` | package `internal/cli` | ok, 82.8% stmt cov | 0 | 0 |
| 5a | `-run 'TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault' -count=1 -v` (isolated) | 1 | 1 | 0 | 0 |
| 5b | `-run 'TestRunDoctorOpts' -count=1 -v` under fake `HOME` with a real alias-bearing `.zshrc`, vs. real `HOME` baseline | 4 vs. 4 | 4 / 4 | 0 | 0 |
| 6 | Built-binary edge-case probes (a–k, 11 fixtures + exit-code check) | 12 checks | 12 | 0 | 0 |

The `-run` pattern grew to include `ParseAliasWords` per the handoff (the
new `TestParseAliasWords_*` names from `eadf0ac` don't match the plain
`ShellAlias` substring); the `--- PASS` count rose from cycle 1's 42 to 64,
consistent with `eadf0ac`'s 2 new tests plus the larger surface `c5d646f`/
`3783610` added (new candidate list, split codex clauses, 4 new pure
helper functions each getting their own test coverage).

### Coverage, `internal/cli/doctor_shell_alias.go`

| Function | Cycle 1 | Cycle 2 |
| --- | --- | --- |
| `shellAliasEnvFromOS` | 100.0% | 100.0% |
| `shellAliasRcCandidates` | 94.4% | 94.4% |
| `shellAliasStatements` | 98.5% | 98.5% |
| `parseAliasStatements` | 93.3% | 93.3% |
| `parseAliasWords` | 84.2% | **100.0%** |
| `scanShellAliasFile` | 100.0% | 100.0% |
| `shellAliasFlagClass` (renamed from `shellAliasFlagIsPermission`) | 100.0% | 100.0% |
| `shellAliasValueTokens` (new) | — | 100.0% |
| `shellAliasConflictingFlags` | 100.0% | 100.0% |
| `shellAliasUnreadableReason` | 83.3% | **100.0%** |
| `shellAliasCodexSentence` (new) | — | 100.0% |
| `shellAliasClaudeSentence` (new) | — | 100.0% |
| `shellAliasScannedClause` (new) | — | 100.0% |
| `shellAliasDetail` (new) | — | 100.0% |
| `checkShellAliases` | 98.0% | 97.3% |

No function is under 80% (lowest is `shellAliasRcCandidates`/
`parseAliasStatements` at 93.3–94.4%, both unchanged from cycle 1 and
already above the floor). Package-level `internal/cli` coverage: 82.8%
(up from 82.5% in cycle 1). Both cycle-1 gaps
(`parseAliasWords`'s all-flags-no-value branch,
`shellAliasUnreadableReason`'s generic-error fallback) are now closed by
`eadf0ac`'s `TestParseAliasWords_AllOptionFlags_NoName_ReturnsNil` (or
equivalent) and a plain-error-fallback test — confirmed by the 84.2%→100%
and 83.3%→100% jumps, not by name (the raw log has the full `-v` test
list if exact names are needed).

### Hermeticity

Same cross-check as cycle 1, re-run against the cycle-2 code: isolated
`TestRunDoctorOpts_ShellAliasesCheck_HermeticByDefault` — PASS (0.57s).
Fake-`HOME` cross-check (a real `alias codex="codex -m x"` in `.zshrc`,
`GOCACHE`/`GOMODCACHE` pinned via `go env`) — the 4-test
`TestRunDoctorOpts` pass/fail/skip set was identical between the real and
fake `HOME` runs (`diff` empty). The `doctorShellAliasEnv` seam still
holds after the cycle-2 rewrite.

### Built-binary edge-case probes (cycle 2)

Built a fresh `go build -o <scratch>/ralph-bin-c2 ./cmd/ralph` (exit 0).
herdr still installed (`herdr 0.7.5`), so codex findings are `warn` as
expected. Same discipline as cycle 1: fresh scratch-dir `HOME` per
fixture, never the real `~`, `grep 'Shell aliases'` on the doctor output.

| Fixture | Expected | Actual | Match |
| --- | --- | --- | --- |
| a. C2-1 regression: `.zshrc` `alias codex='codex --model` + newline + `  gpt'` | warn `--model`, `:1` | `warn — alias codex in ~/.zshrc:1 adds --model — ...` | yes |
| b. `.zshrc`: `alias msg='don` + newline + `alias codex=codex` (unclosed quote) | info, `not fully parsed: ~/.zshrc:1`, NOT `no codex/claude alias found` | `info — not fully parsed: ~/.zshrc:1 — the alias value continues past what was read. ...` | yes (forbidden phrase absent) |
| c. `.bashrc`: `alias codex='codex #keep --model x'` | warn `--model` | `warn — alias codex in ~/.bashrc:1 adds --model — ...` | yes |
| d. `ZDOTDIR=<home>/zd`, `zd/.zprofile`: `alias codex='codex "-a" never'` | warn `--ask-for-approval (-a)`; contains `to autonomous seats only`, `codex_verified = true`, `edits and guarded seats silently run with the alias's approval policy`; NOT `to edits and autonomous seats` | `warn — alias codex in ~/zd/.zprofile:1 adds --ask-for-approval (-a) — ... ralph passes --ask-for-approval to autonomous seats only (a mode a codex seat gets only once [org.permissions].codex_verified = true), so those spawns fail while edits and guarded seats silently run with the alias's approval policy; ...` | yes (all 3 required phrases present verbatim, forbidden phrase absent) |
| e. `.zshrc`: `alias codex='codex -s read-only'` | warn; contains `to edits and autonomous seats`, `alias's sandbox`, `codex_verified = true`; NOT `approval policy` | `warn — alias codex in ~/.zshrc:1 adds --sandbox (-s) — ... ralph passes --sandbox to edits and autonomous seats (modes a codex seat gets only once [org.permissions].codex_verified = true), so those spawns fail while a guarded seat silently runs with the alias's sandbox; ...` | yes (all 3 required phrases present verbatim, forbidden phrase absent) |
| f. `.zlogin` only: `alias claude='claude --permission-mode plan'` | warn; `a guarded seat runs with the alias's permission mode` | `warn — alias claude in ~/.zlogin:1 adds --permission-mode — ... so a guarded seat runs with the alias's permission mode; ...` | yes |
| g. `.bash_login` only: `alias claude='claude --model x'` | info; `the seat still starts` | `info — alias claude in ~/.bash_login:1 adds --model — ... so its value applies and the seat still starts; ...` | yes |
| h. `~/.config/zsh/.zshrc` exists, `~/.config/zsh` chmod 000 | info; `could not read: ~/.config/zsh (permission denied)` exactly once; NOT `no shell rc file found to scan` | `info — could not read: ~/.config/zsh (permission denied) — aliases there were not checked. scanned 0 shell rc file(s) ...` | yes (single occurrence, forbidden phrase absent) |
| i. `~/.config` is a regular file | pass; `no shell rc file found to scan`; no `could not read` | `pass — no shell rc file found to scan` | yes (forbidden phrase absent) |
| j. `.zshrc`: `command -v codex >/dev/null && alias codex="codex -m x"` | warn `:1` | `warn — alias codex in ~/.zshrc:1 adds --model (-m) — ...` | yes |
| k. Cycle-1 regression re-check: fixture a (harmless) → pass; fixture g (no rc) → pass; fixture h (commented) → pass | identical to cycle-1 report | `pass — alias codex in ~/.zshrc:1 has no conflicting flags. ...`; `pass — no shell rc file found to scan`; `pass — no codex/claude alias found. ...` | yes, byte-identical to cycle 1 |

`~/.config/zsh` was restored to mode `755` after fixture h (confirmed via
`stat`). Exit-code check: fixture k-a (pass) and fixture j (warn) both
exit `0` — the check still never fails the process.

No alias VALUE beyond the fixtures above was printed or recorded; no real
rc file was read.

### Regression checks (cycle 2)

| Previously broken/flagged behavior | Status | Evidence |
| --- | --- | --- |
| Cycle-1 flagged coverage gaps (`parseAliasWords`, `shellAliasUnreadableReason`) | closed — both now 100% | coverage table above |
| Cycle-1 probe fixtures a/g/h (pass cases) | unchanged output, byte-identical | probe k row above |
| `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` and the hermeticity seam | still PASS / still hermetic | step 2 and step 5 logs |
| Full package regression (`internal/...`) | all 8 packages `ok`, uncached | step 3 log |
| C2-1 (cross-review cycle-1 finding): `--model` split across a newline inside a quoted alias value was previously mis-parsed | fixed and now covered — fixture a | probe table above |

### Test gaps (cycle 2)

Both test gaps named in the cycle-1 report are now closed (see coverage
table). No new gap was introduced by the cycle-2 rewrite — the 4 new pure
helper functions (`shellAliasValueTokens`, `shellAliasCodexSentence`,
`shellAliasClaudeSentence`, `shellAliasScannedClause`, `shellAliasDetail`)
are all at 100%. Remaining non-goals are unchanged from cycle 1 (no
recursive `source` tracking, no dynamic/interactive alias evaluation, no
live codex-seat-spawn end-to-end check).

### Verdict (cycle 2)

- Pass: 6/6 execution steps (27 shell suites, 64 targeted Go tests, 8/8
  package regression, race-clean, 12/12 edge-case probes including 3
  cycle-1 regression re-checks)
- Fail: 0
- Blocked: 0

Tests pass. No findings. Proceeding to `/pr` is appropriate from a testing
standpoint.
