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
