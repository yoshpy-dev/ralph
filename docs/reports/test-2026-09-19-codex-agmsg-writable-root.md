# Test report: codex-agmsg-writable-root

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-codex-agmsg-writable-root.md` (issue #164)
- Tester: `tester` subagent (Claude Code, standard flow)
- Branch: `feat/codex-agmsg-writable-root`, HEAD `eec559b` (base `main`)
- Scope: behavioral tests only (`./scripts/run-test.sh`, changed-language scope: Go), plus targeted `go test` filters, regression across `./internal/...`, `-race`, coverage, hermeticity cross-checks, and built-binary probes named in the handoff. No static analysis, no source/plan/doc edits.
- Evidence: `docs/evidence/test-2026-09-19-codex-agmsg-writable-root.log`
- Pipeline cycle: 1 (of cap 2).

## Overall verdict: PASS

All requested test commands pass. All 13 fixture probes ((a)-(m)) match their expected `ralph doctor` output exactly. No findings.

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` — shell suites (28 files) | 286 (sum of the 11 files that print an explicit "PASS: N / N" count; the other 17 files assert without a single numeric summary line) | all | 0 | 0 | ~included below |
| `./scripts/run-test.sh` — golang verifier (`go test ./...`, cached) | 8 packages | 8 | 0 | 0 (2 `[no test files]` packages: root, `cmd/ralph`) | cached |
| `go test ./internal/cli/ -run 'Codex\|PathCovers\|CoveringWritableRoot\|TestRunDoctorOpts' -count=1 -v` | 94 (top-level + subtests) | 94 | 0 | 0 | 4.39s |
| `go test ./internal/... -count=1` (regression) | 8 packages | 8 | 0 | 0 | 45.6s (`internal/cli`) + rest |
| `go test ./internal/cli/ -race -count=1` | full `internal/cli` suite | ok | 0 | 0 | 38.1s |
| `go test ./internal/cli/ -coverprofile=... -count=1` | full `internal/cli` suite | ok | 0 | 0 | 35.3s, 83.5% of statements |
| `TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault` in isolation | 1 | 1 | 0 | 0 | 0.46s |
| `TestRunDoctorOpts*` set, default env vs. hostile `CODEX_HOME` | 6 each run | 6 / 6 | 0 | 0 | ~0.5-0.7s each |
| Hostile-`TMPDIR` probe (`TMPDIR=$HOME/ralph-tmpdir-probe-164t go test ... -run 'CodexAgmsgWritableRoot\|CoveringWritableRoot'`) | full filtered set | ok | 0 | 0 | 0.56s |
| Built-binary probes (fixtures a-m) | 13 | 13 match expected | 0 | 0 | n/a (manual) |

Shell suite counts (`./scripts/run-test.sh`, changed-language scope): all 28 files under `tests/` ran with `PASS: N / N` / `FAIL: 0` where a file prints a numeric summary (`test-agent-phase-boundaries.sh` 44/44, `test-detect-changed-languages.sh` 23/23, `test-detect-languages-terraform.sh` 8/8, `test-language-pack-monorepo-roots.sh` 29/29, `test-run-verify-scope.sh` 12/12, `test-self-review-scope.sh` 64/64, `test-terraform-pack-verify.sh` 36/36, `test-terraform-rule-frontmatter.sh` 11/11, `test-verify-mode-split.sh` 59/59, `test-xreview-helpers.sh` 29/29); no `FAIL: 0` block reported a nonzero count and `./scripts/run-test.sh` exited 0.

The `-run 'Codex|...'` filter also matches unrelated pre-existing tests elsewhere in `internal/cli` whose names happen to contain "Codex" (e.g. `doctor_codex_models.go` tests, some `upgrade_v2` tests) — 40 of the 94 matched cases are from `doctor_codex_writable_root_test.go` / `doctor_codex_writable_root_unix_test.go` specifically; the rest are adjacent, unrelated, and also pass.

## Coverage

- Statement (package `internal/cli`, full suite): **83.5%** of statements.
- Per-function coverage for `internal/cli/doctor_codex_writable_root.go` (`go tool cover -func`):

| Function | Coverage | Note |
| --- | --- | --- |
| `codexSandboxEnvFromOS` | 100.0% | |
| `codexConfigDisplayPath` | 83.3% | Uncovered: the `home != "" but path does not have the home prefix` fall-through (`return path` at the function's tail). No fixture supplies a `Home` that doesn't prefix `ConfigPath`. |
| `(*codexConfigDecodeError).Error` | **0.0%** | Never invoked. `codexConfigDecodeDetail` reads `Line`/`Column`/`HasPosition` directly via `errors.As` and never calls `.Error()`/`.String()`/`%v` on the typed error anywhere in this package. The method exists solely to satisfy the `error` interface so `readCodexUserConfig`'s `error`-typed return can carry it; the string it would produce is unreachable via any current call path, not a missed real-world case. |
| `codexConfigDecodeDetail` | 100.0% | |
| `readCodexUserConfig` | 88.0% | Uncovered: the `os.Stat` error branch for a non-`ErrNotExist` stat failure (line 170), the `os.Open` failure branch after a successful stat (177-179), and the `io.ReadAll` failure branch (183-185). All three need a permission-denied-class failure (e.g. `chmod 0o000` on the config file, only meaningful as a non-root user) or a mid-read I/O error; none of the current fixtures produce one. |
| `codexConfigReadReason` | 75.0% | Uncovered: the `errors.As(err, &pathErr)` true-branch (`return pathErr.Err.Error()`, line 213-215) — only reachable from the same permission-denied-class failures above. Every current caller reaches this function only via the two synthetic errors `readCodexUserConfig` raises itself ("not a regular file", "larger than 1 MiB"), both of which are plain `errors.New`, not `*fs.PathError`, so only the catch-all branch is exercised. |
| `codexWritableRoots` | 100.0% | |
| `codexSeatModesPossible` | 100.0% | |
| `codexSandboxReasons` | 100.0% | |
| `codexNotNeededWhyA` | 100.0% | |
| `codexNotNeededWhyB` | 100.0% | |
| `codexSandboxReasonClause` | 100.0% | |
| `agmsgStoreDir` | 100.0% | |
| `pathCovers` | 90.0% | Uncovered: `filepath.Rel`'s error branch (line 393-395). On POSIX, `filepath.Rel` between two absolute, cleaned paths essentially never errors (it only does for cross-volume paths on Windows); this is a defensive branch that's practically unreachable on this platform, not a real gap. |
| `resolveNearestExisting` | 90.9% | Uncovered: the `parent == dir` loop-termination fallback (line 422-424), reached only if `filepath.EvalSymlinks` fails all the way up to and including the filesystem root — not reproducible without breaking the root filesystem itself. Practically unreachable, not a real gap. |
| `coveringWritableRoot` | 100.0% | |
| `codexWritableRootSuffix` | 100.0% | |
| `codexWritableRootDetail` | 100.0% | |
| `checkCodexAgmsgWritableRoot` | 100.0% | |

- Branch/decision coverage: not separately instrumented (Go's `cover` tool reports statement coverage only); the per-function table above enumerates every uncovered statement block by hand.
- Notes: two functions are below the report's flag threshold of 80% (`(*codexConfigDecodeError).Error` at 0%, `codexConfigReadReason` at 75%). Both trace to the same root cause — no fixture currently exercises a permission-denied-class read failure (a real `*fs.PathError` from `os.Open`) — and are noted as a coverage gap below rather than a failure, since AC-2/AC-3 do not name that scenario and every currently-specified Scope-5 case is covered.

## Failure analysis

None. No test failed.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Self-review MEDIUM-1 (duplicate-key/table-conflict TOML errors leaking key names) | Still fixed | `TestCheckCodexAgmsgWritableRoot_DuplicateKey_InfoWithoutKeyNameOrMessage` passes; built-binary fixture (h) confirms live (`note`/`already defined` absent from the printed line). |
| Self-review MEDIUM-2 (`~` substitution reading the real `os.UserHomeDir()`) | Still fixed | `TestCheckCodexAgmsgWritableRoot_HomeSubstitution_TildeInDetail` passes; the seam-based `Home` field is the only source of `~` substitution. |
| Self-review MEDIUM-3 (unresolved textual comparison passing a store symlinked outside every configured root) | Still fixed | `TestCheckCodexAgmsgWritableRoot_StoreSymlinkedOutsideConfiguredRoot_Warn` passes. |
| Self-review NEW-1 (two reasons joined with an indistinguishable second "and") | Still fixed | `TestCheckCodexAgmsgWritableRoot_BothReasons_WarnDetailNumbersEachReason` + `TestCodexSandboxReasonClause` pass; built-binary fixture (f) confirms the `(1) ... and (2) ...` numbering live. |
| Self-review NEW-2 (empty-string `[org.permissions.roles]` key shadowing the real default) | Still fixed | `TestCheckCodexAgmsgWritableRoot_EmptyStringRoleKeyDoesNotSuppressWarn` + `TestCodexSeatModesPossible`'s matching case pass. |
| Self-review NEW-3 ("does not exist" parenthesized after the store path, reading as a claim about the store) | Still fixed | `TestCheckCodexAgmsgWritableRoot_MissingConfigCodexVerified_WarnNamesConfigAbsent` explicitly asserts the parenthesized form is absent; built-binary fixture (g) confirms the config-first wording live. |
| Self-review LOW-6 (reason (a) firing on `codex_verified` alone, ignoring `driver_pool`/role config) | Still fixed | `TestCheckCodexAgmsgWritableRoot_CodexNotInDriverPool_Pass` passes; built-binary fixture (d) confirms live. |
| Self-review LOW-7 (FIFO test unreachable on Windows via `syscall.Mkfifo`) | Still fixed | `doctor_codex_writable_root_unix_test.go` carries the `//go:build !windows` tag; the FIFO test passes on this darwin run and has its own internal 5s hang guard. |
| `go test ./internal/... -count=1` full regression (all 8 packages) | Green | See Test execution table; no package failed. |

## Test gaps

- **Permission-denied read path untested** (`readCodexUserConfig`'s `os.Stat`-error/`os.Open`-error/`io.ReadAll`-error branches, and `codexConfigReadReason`'s `*fs.PathError` branch): no fixture makes the config file unreadable (e.g. via `chmod 0o000`, meaningful only as a non-root user, which this darwin CI-equivalent shell is). Not required by any plan AC; flagged as a coverage blind spot for a possible follow-up unit test (`os.Chmod(cfgPath, 0o000)` then restore) rather than a defect.
- **`(*codexConfigDecodeError).Error()` is dead code in this package** (0% coverage, by design of the self-review MEDIUM-1 fix): it exists only to satisfy the `error` interface; nothing in `internal/cli` ever calls it. A one-line unit test could trivially reach 100%, but doing so tests nothing about the check's actual behavior.
- **Two defensive branches are effectively unreachable on POSIX**: `pathCovers`'s `filepath.Rel` error branch and `resolveNearestExisting`'s `parent == dir` fallback. Neither is realistically triggerable without a broken filesystem or a non-POSIX path form; not treated as a gap worth chasing.
- No gaps found relative to the plan's stated Test plan edge cases (1)-(7) — all seven are covered by name-matched tests (prefix collision, trailing-slash root, symlink, empty `writable_roots`, `CODEX_HOME` unset/containing spaces via temp dirs, config-as-directory, `~`-prefixed roots never matching).

## Probe table: built-binary fixtures (a)-(m)

All probes ran against a freshly built `ralph-bin` (`go build ./cmd/ralph`), a fresh scratch project directory per fixture containing only the stated `ralph.toml`, and a fresh scratch `CODEX_HOME` per fixture. The real, installed agmsg home (`~/.agents/skills/agmsg`) was used for all fixtures except (l).

| Fixture | Expected | Actual | Match |
| --- | --- | --- | --- |
| (a) `codex_verified=true`, config `model = "x"` | warn; Detail has store path, config path, recipe path, "a role resolves to edits or autonomous" | warn — all four elements present verbatim | Yes |
| (b) same `ralph.toml`; `writable_roots` = `[store]` | pass naming that root | pass — names `.../agmsg/db` exactly | Yes |
| (c1) root = ancestor `<home>/.agents` | pass | pass | Yes |
| (c2) root = sibling `.../dbx` | warn (prefix sibling) | warn | Yes |
| (d) `driver_pool = ["claude"]` (no codex), `codex_verified=true` | pass, "not needed: ... [org].driver_pool" | pass — exact text | Yes |
| (e) `default="guarded"`, `sandbox_mode="workspace-write"` | warn, ONLY the guarded reason, no numbering | warn — single unnumbered reason | Yes |
| (f) `default="guarded"`, `codex_verified=true`, `roles.reviewer="edits"`, `sandbox_mode="workspace-write"` | warn, BOTH reasons as `(1) ... and (2) ...` | warn — exact numbered form | Yes |
| (g) `ralph.toml` as (a); `CODEX_HOME` dir with no `config.toml` | warn starting with "`<config path>` does not exist, so no writable root covers the agmsg store" | warn — exact lead-in | Yes |
| (h) `ralph.toml` as (a); duplicate key (`note="a"`/`note="b"`) | info "could not be decoded as a codex config"; no `note`/`already defined` | info — exact wording, no leak | Yes |
| (i) `ralph.toml` as (a); `sandbox_mode = 5` | info with "(line 1, column ...)" | info — "(line 1, column 16)" | Yes |
| (j1) `ralph.toml` as (a); `config.toml` is a directory | info "not a regular file" | info — exact wording | Yes |
| (j2) `ralph.toml` as (a); `config.toml` is a FIFO | info "not a regular file"; must not hang | info — exact wording; returned promptly (exit 0) under a manual 10s background-kill guard (macOS zsh ships neither `timeout` nor `gtimeout`) | Yes |
| (k1) `AGMSG_STORAGE_PATH=<scratch>/store/`; root = default `db` dir | warn naming `<scratch>/store` (slash stripped), with the override suffix | warn — store path has no trailing slash, suffix present | Yes |
| (k2) same override; root = `<scratch>/store` | pass with the suffix | pass — suffix present | Yes |
| (l) `RALPH_ORG_AGMSG_HOME=<scratch>/no-agmsg` (nonexistent) | info "agmsg not installed at ..." | info — exact wording | Yes |
| (m) this repository's own root, real config | pass, "not needed: ... codex_verified is false and no role resolves to guarded" | pass — exact text | Yes |
| Exit code comparison | warn fixture (a) exit code == pass fixture (b) exit code | both `0` | Yes |

No probe contradicted its expectation. Hygiene: no fixture or this report contains any `api_key=`/`password=`/`access_token=`-shaped string; the real `~/.codex/config.toml` was never read or printed directly — fixture (m) only grepped the doctor output line from a live `ralph doctor` run against the repository's real config.

## Verdict

- Pass: yes — all shell suites, all targeted and regression Go tests, `-race`, hermeticity cross-checks, and all 13 built-binary probes match expectations. Proceed to `/sync-docs`.
- Fail: none.
- Blocked: none.

## Cycle 2

- Date: 2026-09-19
- Tester: `tester` subagent (Claude Code, standard flow)
- Branch: `feat/codex-agmsg-writable-root`, HEAD `a16e3f6` (base `main`)
- Scope: behavioral tests only, same command set as Cycle 1, re-run against the cross-review cycle-1 fixes (`ab09dbc`, `91a5542`, `c03c21e`, `735fc45`, `5fcf070`, plus a self-review test-only follow-up `78c4ed9`). No static analysis, no source/plan/doc edits.
- Evidence: `docs/evidence/test-2026-09-19-codex-agmsg-writable-root-cycle2.log`
- Pipeline cycle: 2 (of cap 2).

### Overall verdict: PASS

All requested test commands pass. All 17 built-binary probes (a, b, c1, c2, c3, d, e, f1, f2, g1, g2, h, i, j1, j2, k, l) plus the exit-code comparison match their expected `ralph doctor` output exactly. No findings.

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (full fallback scope: shell suites + golang, triggered by `.gitallowed` being unclassified) | all shell files with a numeric `PASS: N / N` summary line (`test-agent-phase-boundaries.sh` 44/44, `test-detect-changed-languages.sh` 23/23, `test-detect-languages-terraform.sh` 8/8, `test-language-pack-monorepo-roots.sh` 29/29, `test-run-verify-scope.sh` 12/12, `test-self-review-scope.sh` 64/64, `test-terraform-pack-verify.sh` 36/36, `test-terraform-rule-frontmatter.sh` 11/11, `test-verify-mode-split.sh` 59/59, `test-xreview-helpers.sh` 29/29) + 8 golang packages | all | 0 | 0 | ~57s (exit 0) |
| `go test ./internal/cli/ -run 'Codex\|PathCovers\|PathCrosses\|CoveringWritableRoot\|TestRunDoctorOpts' -count=1 -v` | 202 (top-level + subtests) | 202 | 0 | 0 | 6.54s |
| `go test ./internal/... -count=1` (regression) | 8 packages | 8 | 0 | 0 | 44.1s (`internal/cli`) + rest |
| `TMPDIR=/tmp go test ./internal/cli/... -count=1` (CI simulation) | full `internal/cli` suite | ok | 0 | 0 | 59.5s |
| `go test ./internal/cli/ -race -count=1` | full `internal/cli` suite | ok | 0 | 0 | 58.2s |
| `go test ./internal/cli/ -coverprofile=... -count=1` | full `internal/cli` suite | ok | 0 | 0 | 49.9s, 84.0% of statements |
| `TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault` in isolation | 1 | 1 | 0 | 0 | 1.43s |
| `TestRunDoctorOpts*` set, default env vs. hostile `CODEX_HOME` (`sandbox_mode = "workspace-write"`) | 6 each run | 6 / 6 | 0 | 0 | ~0.4-0.7s each; identical PASS/FAIL/SKIP name sets, only timing differs |
| Hostile-`TMPDIR` probe (`TMPDIR=$HOME/ralph-tmpdir-probe-164t2 go test ./internal/cli/ -run 'Codex\|CoveringWritableRoot' -count=1`) | full filtered set | ok | 0 | 0 | 2.14s |
| Built-binary probes (fixtures a, b, c1-c3, d, e, f1-f2, g1-g2, h, i, j1-j2, k, l + exit-code comparison) | 18 | 18 match expected | 0 | 0 | n/a (manual) |

The `-run 'Codex|...'` filter count rose from 94 (cycle 1) to 202 because the cross-review cycle-1 fixes added `PathCrosses` to the filter (per the handoff) and the fixes themselves added new tests (protected-directory rule, implicit temp roots, model eligibility, mixed-spelling coverage). As in cycle 1, the filter also matches unrelated pre-existing tests elsewhere in `internal/cli` whose names happen to contain "Codex" (e.g. `doctor_codex_models.go`, some `upgrade_v2` tests); all pass and are not part of this check's own test file.

### Coverage

- Statement (package `internal/cli`, full suite): **84.0%** of statements (up from cycle 1's 83.5%; the delta comes from the self-review test-only follow-up `78c4ed9`, which closed two of cycle 1's flagged gaps).
- Per-function coverage for `internal/cli/doctor_codex_writable_root.go` (`go tool cover -func`), all 27 functions:

| Function | Coverage | Note |
| --- | --- | --- |
| `codexSandboxEnvFromOS` | 100.0% | |
| `codexConfigDisplayPath` | 83.3% | Uncovered: line 140, the `home != "" but path does not have the home prefix` fall-through (`return path`). Same gap as cycle 1; no fixture supplies a `Home` that doesn't prefix `ConfigPath`. |
| `(*codexConfigDecodeError).Error` | **100.0%** | Was 0.0% in cycle 1; `78c4ed9` added `TestCodexConfigDecodeError_ErrorCarriesNoConfigText`, which now calls `.Error()` directly (with and without a position) to pin the exact wording. |
| `codexConfigDecodeDetail` | 100.0% | |
| `readCodexUserConfig` | 92.0% | Up from cycle 1's 88.0%. Uncovered: line 199 (`os.Stat`'s non-`ErrNotExist` error branch) and lines 212-214 (`io.ReadAll` failure branch). The `os.Open` failure branch (206-208) is now covered — `78c4ed9` added `TestCheckCodexAgmsgWritableRoot_UnreadableConfig_InfoWithReasonOnly` (unix-only, `chmod 0o000` on a non-root user), which reaches it via `os.Open`, not `os.Stat`/`io.ReadAll`. |
| `codexConfigReadReason` | **100.0%** | Was 75.0% in cycle 1; the same new chmod-0o000 test supplies a real `*fs.PathError` from `os.Open`, exercising the `errors.As(err, &pathErr)` true-branch. |
| `codexWritableRoots` | 100.0% | |
| `codexModelPoolModels` | 100.0% | New function (cross-review WC-2 fix, `ab09dbc`/`c03c21e`). |
| `codexModelPermittedForRole` | 100.0% | New function (same fix). |
| `codexSeatModesPossible` | 100.0% | |
| `codexSandboxReasons` | 100.0% | |
| `codexNotNeededWhyA` | 100.0% | |
| `codexNotNeededWhyB` | 100.0% | |
| `codexSandboxReasonClause` | 100.0% | |
| `agmsgStoreDir` | 100.0% | |
| `pathCovers` | 90.0% | Uncovered: line 473-475, `filepath.Rel`'s error branch. Same as cycle 1 — practically unreachable on POSIX between two absolute cleaned paths, not a real gap. |
| `pathCrossesCodexProtectedDir` | 100.0% | New function (cross-review AR-1 fix, `ab09dbc`). |
| `resolveNearestExisting` | 90.9% | Uncovered: line 561-563, the `parent == dir` loop-termination fallback. Same as cycle 1 — only reachable if `EvalSymlinks` fails to the filesystem root, not reproducible. |
| `codexRootCoverage` | 100.0% | New function (cross-review C2-1 fix, `735fc45`/`5fcf070`) — the 4-way configured/resolved mixed-spelling comparison. |
| `coveringWritableRoot` | 100.0% | |
| `codexImplicitWritableRoots` | 100.0% | New function (cross-review WC-1 fix, `ab09dbc`/`91a5542`) — the `/tmp` and `$TMPDIR` implicit-root list. |
| `codexCoveringImplicitRoot` | 90.9% | New function. Uncovered: line 703-704, the `!covers { continue }` branch — reached only when an implicit root candidate genuinely does not cover the target at all (as opposed to covering-but-blocked). With at most two implicit roots (`/tmp`, `$TMPDIR`) in the current fixtures, every fixture that includes a non-covering implicit root also has it either cover or be the only candidate, so this specific "does not cover, keep looking" branch is never hit. Narrow, not required by any AC. |
| `codexWritableRootSuffix` | 100.0% | |
| `codexPaneEnvironmentMayDifferClause` (const) | n/a | not a function |
| `codexImplicitRootDetail` | 85.7% | New function. Uncovered: line 756-757, the `!exists` branch of the config-missing wording (`"%s does not exist"` for an implicit-root pass where the config file itself is absent). Every current implicit-root-pass fixture (including this cycle's g1) supplies a config file, so only the `exists` branch (`is not set in %s`) is exercised. Narrow, not required by any AC. |
| `codexBlockedAncestorClause` | 100.0% | |
| `codexWritableRootDetail` | 100.0% | |
| `checkCodexAgmsgWritableRoot` | 100.0% | |

- Branch/decision coverage: not separately instrumented (statement coverage only, as in cycle 1); the table above enumerates every uncovered statement block by hand.
- Every function is at or above the 80% flag threshold this cycle (cycle 1 had two below: `(*codexConfigDecodeError).Error` at 0% and `codexConfigReadReason` at 75%; both are now 100%, closed by the self-review follow-up `78c4ed9`, not by anything in this cycle's own diff).

### Failure analysis

None. No test failed.

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| All cycle-1 fixes (self-review MEDIUM-1/2/3, NEW-1/2/3, LOW-6/7) | Still fixed | Full regression run (`go test ./internal/... -count=1`) green; targeted filter includes all cycle-1 test names, all PASS. |
| Cross-review AR-1 (broad ancestor root treated as covering, ignoring codex's `.git`/`.agents`/`.codex` read-only protection) | Fixed and covered | `pathCrossesCodexProtectedDir` 100%; built-binary fixture c1 (root = `<home>`) warns with the exact blocked-ancestor parenthetical; fixture c2 (root = `<home>/.agents`) passes. |
| Cross-review WC-1 (default-writable `/tmp`/`$TMPDIR` roots not counted, causing unnecessary warns) | Fixed and covered | `codexImplicitWritableRoots`/`codexCoveringImplicitRoot`/`codexImplicitRootDetail` all exercised; fixture g1 (`AGMSG_STORAGE_PATH` under `/tmp`, no exclude) passes with the exact wording; fixture g2 (`exclude_slash_tmp = true`) warns. |
| WC-1 follow-up (the `/tmp` literal baked into the check would make `t.TempDir()` fixtures collide with it whenever the real `TMPDIR` is `/tmp`, as on the ubuntu CI runner) | Fixed and covered | `TMPDIR=/tmp go test ./internal/cli/... -count=1` is ok (full package, not just the filtered subset); hostile-`TMPDIR` probe against `$HOME/ralph-tmpdir-probe-164t2` is also ok. |
| Cross-review WC-2 (reason counted a role even when no codex model in the pool was permitted for it) | Fixed and covered | `codexModelPoolModels`/`codexModelPermittedForRole` 100%; fixture f1 (`implementer` restricted to `opus`, a claude model) passes as not-needed; fixture f2 (same role restricted to a codex model from the default pool) warns. |
| Cross-review C2-1 (protected-directory check only looked at the resolved spelling, missing a symlink that hides `.agents` after resolution) | Fixed and covered | `codexRootCoverage` 100%; `TestCodexRootCoverage_MixedSpellings_ProtectedSymlinkStillBlocks` passes; built-binary fixture h (symlinked `.agents` home, root = configured spelling) warns with the blocked-ancestor parenthetical. |
| `go test ./internal/... -count=1` full regression (all 8 packages) | Green | See Test execution table; no package failed. |

### Test gaps

- **Two narrow branches newly introduced by the cross-review fixes are not yet exercised**: `codexCoveringImplicitRoot`'s "candidate does not cover at all" continue-branch (line 703-704), and `codexImplicitRootDetail`'s config-absent wording for an implicit-root pass (line 756-757). Neither is required by AC-2/AC-3/AC-6; both would need a fixture with two implicit roots where only one covers, or an implicit-root pass with no config file present at all. Flagged as a coverage blind spot for a possible follow-up test, not a defect.
- The three cycle-1 gaps not tied to a permission-denied scenario remain: `codexConfigDisplayPath`'s home-prefix-mismatch fallback (140), `pathCovers`'s `filepath.Rel` error branch (473-475), and `resolveNearestExisting`'s `parent == dir` fallback (561-563) — all three practically unreachable on POSIX without a broken filesystem or a `Home` value that doesn't prefix the config path in any current fixture.
- No gaps found relative to the plan's stated Test plan edge cases (1)-(7) (prefix collision, trailing-slash root, symlink, empty `writable_roots`, `CODEX_HOME` unset/containing spaces, config-as-directory, `~`-prefixed roots) — all still covered by name-matched tests, unchanged from cycle 1.

### Probe table: built-binary fixtures

All probes ran against a freshly built `ralph-bin` (`go build ./cmd/ralph`, HEAD `a16e3f6`), a fresh scratch project directory per fixture containing only the stated `ralph.toml`, and a fresh scratch `CODEX_HOME` per fixture, all under `$HOME/ralph-probe-164c2` (never under `/tmp` — a store there would be covered by the new implicit-root logic itself, which would distort every fixture except g, which tests that on purpose). The real, installed agmsg home (`~/.agents/skills/agmsg`) was used for every fixture except (k).

| Fixture | Expected | Actual | Match |
| --- | --- | --- | --- |
| (a) `codex_verified=true`, config `model = "x"` | warn; Detail has store path, config path, recipe path, "a role that can use a codex model resolves to edits or autonomous" | warn — all four elements present verbatim | Yes |
| (b) same `ralph.toml`; `writable_roots` = `[store]` | pass naming that root | pass — names `.../agmsg/db` exactly | Yes |
| (c1) root = `<home>` (ancestor, but a protected element lies between) | warn with the blocked-ancestor parenthetical naming `<home>` | warn — `(/Users/hiroki.yoshioka contains it, but codex keeps .git, .agents, and .codex directories under a writable root read-only)` | Yes |
| (c2) root = `<home>/.agents` | pass | pass — names `<home>/.agents` exactly | Yes |
| (c3) root = `<home>/.agents/skills/agmsg/dbx` (sibling) | warn without the parenthetical | warn — no parenthetical present | Yes |
| (d) `driver_pool = ["claude"]` (no codex), `codex_verified=true` | pass, "not needed: ... [org].driver_pool" | pass — exact text | Yes |
| (e) `codex_verified=true`, `model_pool` has only claude entries | pass, "not needed: ... [org].model_pool has no codex model" | pass — exact text | Yes |
| (f1) `default="guarded"`, `codex_verified=true`, `roles.implementer="autonomous"`, `[org.roles] implementer=["opus"]`, config `model="x"` | pass (not needed: no role able to use a codex model resolves to edits/autonomous) | pass — exact text (also correctly reports the guarded/`sandbox_mode` condition as unmet) | Yes |
| (f2) same, but `implementer` restricted to a codex model from the default pool | warn | warn — exact text, same as fixture (a) | Yes |
| (g1) `AGMSG_STORAGE_PATH=/tmp/ralph-probe-164-store`, config `model="x"` (no exclude) | pass, "is under /tmp, which workspace-write keeps writable by default (exclude_slash_tmp is not set in …)" + `AGMSG_STORAGE_PATH` suffix | pass — exact wording and suffix | Yes |
| (g2) same, `exclude_slash_tmp = true` | warn | warn — exact wording, override suffix still present | Yes |
| (h) symlinked `.agents` home; store reached through the symlink; root = configured home spelling | warn with the blocked-ancestor parenthetical naming that root | warn — exact parenthetical, naming the configured `<probe>/h/home` root | Yes |
| (i) `ralph.toml` as (a); `CODEX_HOME` dir with no `config.toml` | warn starting with "`<config path>` does not exist, so no writable root covers the agmsg store" | warn — exact lead-in | Yes |
| (j1) `ralph.toml` as (a); duplicate key (`note="a"`/`note="b"`) | info "could not be decoded as a codex config"; no `note`/`already defined` | info — exact wording, no leak | Yes |
| (j2) `ralph.toml` as (a); `config.toml` is a FIFO | info "not a regular file"; must not hang | info — exact wording; returned within 10s under a background-kill guard (exit 0) | Yes |
| (k) `RALPH_ORG_AGMSG_HOME=<scratch>/no-agmsg` (nonexistent) | info "agmsg not installed at ..." | info — exact wording | Yes |
| (l) this repository's own root, real config | pass, "not needed: ... codex_verified is false and no role that can use a codex model resolves to guarded" | pass — exact text | Yes |
| Exit code comparison | warn fixture (a) exit code == pass fixture (b) exit code | both `0` | Yes |

No probe contradicted its expectation. Hygiene: no fixture command or this report contains any secret-shaped `key=value` string; the real `~/.codex/config.toml` was never read or printed directly — fixture (l) only grepped the doctor output line from a live `ralph doctor` run against the repository's real config. All scratch fixture directories (`$HOME/ralph-probe-164c2`, `$HOME/ralph-codex-home-probe-164t2`, `$HOME/ralph-tmpdir-probe-164t2`) were removed after use.

### Verdict

- Pass: yes — all shell suites, all targeted and regression Go tests, the CI-simulating `TMPDIR=/tmp` run, `-race`, hermeticity cross-checks, and all 17 built-binary probes plus the exit-code comparison match expectations. Proceed to `/pr`.
- Fail: none.
- Blocked: none.
