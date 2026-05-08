# Self-review report: Ralph Loop Codex driver (Phase 2) — cycle 4

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Reviewer: Claude Code (`reviewer` subagent, post-cycle-3 cross-review fix)
- Scope: delta of commit `f735299` only — diff quality of the three cycle-3 cross-review P2 fixes (TTY guard in fake-claude/fake-codex stdin drain; `checkLoopDriver` fail-on-missing-codex + env-aware display; two new focused tests + one updated existing test). Files in scope: `tests/fixtures/cli-stubs/claude`, `tests/fixtures/cli-stubs/codex`, `internal/cli/doctor.go`, `internal/cli/doctor_loop_test.go`. No spec compliance and no test execution per `/self-review` contract.

## Evidence reviewed

- `git show --stat f735299` — 4 files, +121 / -23. Matches the four files the user enumerated; no collateral edits.
- `git diff f735299~1..f735299` (full hunks read for all four files).
- `internal/cli/doctor.go:1-20,440-479` — full surrounding context; `os/exec` was already imported (line 10), so the new `exec.LookPath("codex")` call introduces no new import.
- `internal/cli/run.go:75-78,173-185` — confirms `appendEnvIfMissing` does the same env > TOML > default resolution on the run-side; the new `pick` closure mirrors that contract.
- `internal/config/config.go:60-82` — `config.Default().Loop` exposes `Driver`, `CodexSandbox`, `CodexApprovalPolicy`, `ClaudeReviewerModel`, all four of which the refactor now resolves through `pick`.
- `internal/cli/doctor_loop_test.go` (full file) — confirms the existing `TestCheckLoopDriver_PriorityAndSource` cases that now require a sham codex binary (`wantValue == "codex"`) get one in setup, while default/claude cases skip the stub.
- `grep -rn "head -c 4096"` — only `tests/fixtures/cli-stubs/claude` and `tests/fixtures/cli-stubs/codex` host this pattern in tracked code; no other fixture would have benefited from the same TTY guard.
- `find templates/base -name "doctor.go" -o -name "cli-stubs"` → empty. The templates/base mirror does not include Go sources or test fixtures, so there is no mirror-discipline drift to worry about for this commit.
- Existing cycle-1/2/3 self-reviews — used as a stylistic reference for the report.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | The new `pick` closure throws away the `source` value for `sandbox`, `approval`, and `reviewer` (`sandbox, _ := pick(...)`). That is correct for today's display, which only surfaces `source:` for the driver. But if future doctor work wants to flag "sandbox set via env, approval via TOML, reviewer via default", the data has to be re-derived. Acceptable as-is; flagged only so it is visible. | `internal/cli/doctor.go:459-461` | Optional: a future change could bundle `(value, source)` into a small `resolved` struct if per-knob source surfacing becomes useful. Not blocking. |
| LOW | maintainability | The two new tests and the updated existing test independently rebuild the "write a `#!/bin/sh\nexit 0\n` stub and prepend its directory to `PATH`" pattern (3 copies, ~5 lines each). Trivial duplication, but a `func writeShamCodex(t *testing.T) string` helper would make the pattern grep-able and self-documenting. | `internal/cli/doctor_loop_test.go:68-75, 119-124, used implicitly in :97-99` | Optional cleanup in a follow-up. Out of scope for the cycle-4 minimum-viable fix. |
| LOW | readability | `TestCheckLoopDriver_FailsWhenCodexMissing` sets `t.Setenv("PATH", t.TempDir())` (line 99) — an empty directory, so `exec.LookPath("codex")` returns ErrNotFound on POSIX. This works on Linux/macOS dev machines and CI, but on a hypothetical sandbox where `t.TempDir()` happened to live next to a system `codex` binary in the same dir, the test's intent could leak. Not a real risk in this repo. | `internal/cli/doctor_loop_test.go:99` | None. The current pattern is the idiomatic Go way to isolate `LookPath`. |
| LOW | naming | `pick` is a shorter/more generic name than `appendEnvIfMissing`; the two helpers share intent (env > TOML > default) but live in different files and read differently. A future reader who searches for the resolution rule may find one but not the other. | `internal/cli/doctor.go:446` vs. `internal/cli/run.go:173-185` | Optional follow-up: extract a single `internal/config.Resolve(envKey, tomlVal, defaultVal) (string, source)` helper used by both call sites. Tracked as tech-debt below; do not block merge. |

No CRITICAL, no HIGH, no MEDIUM findings. The fix is minimum-viable and surgical.

### Confirmed against the four user-supplied focus areas

1. **TTY guard is minimum-viable and behaviour-preserving for the non-interactive happy path.** Confirmed.
   - The new branch reads `if [ -t 0 ]; then stdin_buf=""; else stdin_buf="$(head -c 4096 || true)"; fi`. When stdin is a pipe, heredoc, or `< /dev/null` (the way `tests/test-ralph-cli-driver.sh` and the orchestrator drive both stubs), `[ -t 0 ]` is false and the original `head -c 4096 || true` runs unchanged. Stub semantics, JSON shape, and exit code are byte-identical to cycle-3 in non-TTY runs.
   - The interactive case correctly degrades to `stdin_buf=""`, which is then JSON-encoded as the empty string in the call log — the only observable difference, and only when a developer runs the stub by hand. No test asserts `.stdin` is non-empty under TTY conditions, so no existing test breaks.
   - Both stubs adopt the same shape (claude:40-47, codex:41-49); neither stub forgets to add the comment that explains *why*. The codex stub additionally points at the issue (`#44 cycle-3 cross-review P2 #1`), and the claude stub references the codex stub by name — this asymmetric breadcrumb pair is fine.
2. **`checkLoopDriver` refactor stays readable and matches `appendEnvIfMissing` semantics.** Confirmed.
   - The `pick` closure is 11 lines, single-purpose, captures only `getenv` from the outer scope, and returns `(value, source)`. The only quirk worth noting (TOML matching default still reports `source: toml`) is preserved from the pre-refactor code and still has its inline comment.
   - The old switch had four cases (`envVal != ""`, `tomlVal != "" && tomlVal != defaultVal`, `tomlVal != ""`, default) that collapsed cleanly into the closure's three returns. The collapse is faithful: case 2 and case 3 both produced `(tomlVal, "toml")` in the old code, so the new `if tomlVal != ""` covers both without behaviour change.
   - Calling `pick` four times (driver/sandbox/approval/reviewer) is a tiny readability win over the old "only resolve driver, then read TOML directly for the rest" mix-and-match. The new shape makes the env > TOML > default rule obvious for every displayed field.
   - The `exec.LookPath("codex")` short-circuit happens after `pick` resolution but before the pass branch, so the fail message can include the resolved `effective` and `source`. The message "%s (source: %s) — codex CLI not found in PATH; `ralph run` preflight will fail" is specific enough that a user can act on it without re-reading the source.
   - `os/exec` was already imported (line 10), so the new call site adds zero imports and zero indirect dependencies.
   - Comparison to `appendEnvIfMissing`: the two helpers express the same priority but in dual forms — `appendEnvIfMissing` is "does the child env already have it? if not, append" (used when constructing `cmd.Env`), `pick` is "what value should I display now?" (used when computing doctor output). Distinct call shapes, equivalent priority. The doc-comment on `checkLoopDriver` explicitly cites the env > TOML > default contract.
3. **New tests are deterministic.** Confirmed.
   - No `time.Sleep`, no goroutines, no network, no syscalls beyond `os.WriteFile` to `t.TempDir()`. Both new tests use `t.Setenv`, which is automatically reverted at test end and is safe under `t.Parallel()` (although these tests do not opt into parallel — also fine, since `t.Setenv` forbids it).
   - PATH manipulation prepends a fresh `t.TempDir()` containing exactly one shell stub with `0755` mode. `exec.LookPath` honours this PATH on every supported OS the repo CI targets (Linux, macOS).
   - `TestCheckLoopDriver_FailsWhenCodexMissing` uses `t.Setenv("PATH", t.TempDir())` — an *empty* tempdir, so the LookPath result is deterministic regardless of what's installed on the dev box. No reliance on parent env.
   - `TestCheckLoopDriver_EnvOverridesShownInDetail` correctly *prepends* the sham dir to `os.Getenv("PATH")` rather than replacing it (line 124) because the test does not need to mask the rest of PATH — but it could not rely on a system codex either, because the `pick` for env wins regardless. This is harmless asymmetry with the failing-codex test, but worth noting for future readers.
   - The updated `TestCheckLoopDriver_PriorityAndSource` follows the same pattern: it gates the sham-stub setup on `tc.wantValue == "codex"`, so the claude/default cases run unaffected and the codex cases get a deterministic stub. The error message on line 78 was upgraded to include `r.Detail` for easier triage when the sham setup ever fails.
4. **No CRITICAL/HIGH issues introduced.** Confirmed.
   - No secrets, no credentials, no injection vectors. The only new shell content is `#!/bin/sh\nexit 0\n` written to a tempdir under the test process's UID — no traversal, no eval.
   - No swallowed errors. The `exec.LookPath` error is checked and converted to a `fail` checkResult with explanatory detail. The `os.WriteFile` errors in tests are surfaced via `t.Fatal`. The `pick` closure has no failure modes (pure data resolution).
   - No new debug code, no `fmt.Println`, no `log.Print`, no `TODO` markers, no commented-out code.
   - Security boundary check: the failing-codex branch does *not* propagate the user-supplied `effective` value into a shell command; it goes into `fmt.Sprintf` with `%s` and stays inside the doctor result struct. No shell escape needed.
   - The stub `0755` mode is intentional (the file must be executable for `exec.LookPath` to find it) and is scoped to a `t.TempDir()` that the runtime cleans up at test end. No widening of the test process's own permissions.

## Positive notes

- The two stub edits land in lock-step: identical guard shape, identical comment style, both reference the cycle-3 cross-review P2 #1 finding so a future reader can trace the *why* in one grep.
- The `pick` closure removes a real readability tax (the four-arm switch was hard to scan) without losing the "TOML-matches-default still says toml" subtlety. The inline comment for that subtlety survived intact.
- The fail message on missing codex names exactly the next thing that breaks (`ralph run` preflight), turning a "doctor passed, ralph run failed, why?" cycle into a single self-explanatory line. This is the kind of detail that matters for an operator running `ralph doctor` on a fresh machine.
- The new tests are exemplary in scoping: each one names the cycle-3 finding it pins (lines 93-96 and 113-116) and tests one observable property each, so a future regression points at exactly one fix to revert/repair.
- Existing `TestCheckLoopDriver_PriorityAndSource` was upgraded surgically — only the codex-driver branches grow a sham stub, the claude/default branches stay untouched. No risk of "fix made the existing tests slower or noisier."
- No dead code, no orphan helper, no unused import. The refactor strictly tightens.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Two near-identical env > TOML > default resolvers (`appendEnvIfMissing` in `run.go`, new `pick` closure in `doctor.go`) — different shapes, same priority. | LOW — divergent maintenance: a future change to the priority rule would have to touch both call sites in lock-step. | Cycle-4 was strictly the cycle-3 P2 fix; extracting a shared `config.Resolve(envKey, tomlVal, defaultVal) (string, source)` helper is a separate refactor. | Next time either the env list or the priority rule changes (e.g. adding a fourth knob, or supporting a `~/.ralphrc` layer between env and TOML). | This report; commit `f735299`; `internal/cli/run.go:173-185`. |
| Three copies of the "write `#!/bin/sh\nexit 0\n` and prepend its dir to PATH" pattern in `internal/cli/doctor_loop_test.go`. | LOW — readability only; if future tests touch PATH-resolved binaries, they will copy-paste a fourth time. | Strict scope discipline for the cycle-4 fix. | Next test that needs a sham PATH binary, or any opportunistic `internal/cli/*_test.go` cleanup. | This report; `internal/cli/doctor_loop_test.go:68-75, 119-124`. |

_(Both rows are LOW prompt/test ergonomics drift; appending to `docs/tech-debt/README.md` would over-record. The rows above are sufficient as a breadcrumb. Promote either if the same pattern recurs in another PR.)_

## Recommendation

- Merge: yes. The cycle-4 delta is a minimal, well-scoped, well-documented fix for the three cycle-3 cross-review P2 findings. All four user-supplied focus checks pass. No CRITICAL or HIGH findings; only LOW maintainability/naming observations that are explicitly out of cycle-4 scope.
- Follow-ups:
  - Optional: extract a shared `config.Resolve` helper to deduplicate `appendEnvIfMissing` and the new `pick` closure (tech-debt row 1).
  - Optional: extract a `writeShamCodex(t)` test helper to dedupe the three PATH-stub copies (tech-debt row 2).
  - Pipeline must continue to `/verify` → `/test` → `/sync-docs` → `/cross-review` → `/pr` per `.claude/rules/post-implementation-pipeline.md`.
