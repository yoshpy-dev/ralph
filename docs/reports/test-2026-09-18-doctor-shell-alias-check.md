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
