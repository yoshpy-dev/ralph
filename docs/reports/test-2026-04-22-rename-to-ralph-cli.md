# Test report: Rename repo & rebrand to `ralph` CLI

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-rename-to-ralph-cli.md`
- Tester: tester subagent (Claude Code)
- Scope: Unit (`go test ./...`), integration (built-binary smoke: `ralph init` / `ralph doctor` / `ralph version` / `ralph --help`), regression (`scripts/install.sh` URL assembly + `./scripts/run-test.sh` full pass), edge (intentional old-name residue in archive/specs/active-plan)
- Evidence: `docs/evidence/test-2026-04-22-rename-to-ralph-cli.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go clean -testcache && go test ./... -count=1` (unit, fresh) | 9 packages with tests | 9 | 0 | 0 | ~24s wall |
| `./scripts/run-test.sh` (local verifier + go verifier) | 11 mojibake + 9 Go packages | 20 | 0 | 0 | ~15s |
| mojibake hook: `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | <1s |
| gofmt (via go verifier) | 1 | 1 | 0 | 0 | <1s |
| staticcheck (via go verifier) | 1 run (0 issues) | 1 | 0 | 0 | ~2s |
| `go build -o /tmp/ralph ./cmd/ralph` | 1 | 1 | 0 | 0 | ~5s |
| Integration: `/tmp/ralph version` | 1 | 1 | 0 | 0 | <1s |
| Integration: `/tmp/ralph --help` | 1 | 1 | 0 | 0 | <1s |
| Integration: `/tmp/ralph init --yes` (scratch dir scaffold) | 1 | 1 | 0 | 0 | <1s |
| Integration: `/tmp/ralph doctor` (post-init in scratch) | 5 checks | 5 | 0 | 0 | <1s |
| Regression: `install.sh` URL assembly points at `yoshpy-dev/ralph` | 1 | 1 | 0 | 0 | <1s |
| Regression: `./scripts/run-verify.sh` (via `run-test.sh`) | aggregate | pass | 0 | 0 | see evidence |

### Go test package detail

Ran with a cleared test cache (`go clean -testcache`) so no results are stale.

| Package | Status |
| --- | --- |
| `github.com/yoshpy-dev/ralph` | no test files |
| `github.com/yoshpy-dev/ralph/cmd/ralph` | no test files |
| `github.com/yoshpy-dev/ralph/cmd/ralph-tui` | no test files |
| `github.com/yoshpy-dev/ralph/internal/action` | ok |
| `github.com/yoshpy-dev/ralph/internal/cli` | ok |
| `github.com/yoshpy-dev/ralph/internal/config` | ok |
| `github.com/yoshpy-dev/ralph/internal/scaffold` | ok |
| `github.com/yoshpy-dev/ralph/internal/state` | ok |
| `github.com/yoshpy-dev/ralph/internal/ui` | ok |
| `github.com/yoshpy-dev/ralph/internal/ui/panes` | ok |
| `github.com/yoshpy-dev/ralph/internal/upgrade` | ok |
| `github.com/yoshpy-dev/ralph/internal/watcher` | ok |

No testdata drift from the import-path rename was observed. All packages import the new `github.com/yoshpy-dev/ralph/...` module path and tests that reference module-qualified symbols still resolve.

## Coverage

- Statement: not instrumented for this run (no `-coverprofile` supplied; scope of the change is a rename and does not alter branch coverage expectations).
- Branch: n/a.
- Function: n/a.
- Notes: The rename touches import paths and user-facing strings, not executable logic. Unit tests covering the critical runtime surfaces (`action`, `cli`, `config`, `scaffold`, `state`, `ui`, `upgrade`, `watcher`) all pass with the new module path, which is the most meaningful coverage signal for this PR. The `scripts/install.sh` regression is exercised structurally (URL assembly via `bash -x`) rather than via an automated test harness — the script has no dedicated test fixture under `tests/`.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `scripts/install.sh` URL must point at new repo | PASS | `bash -x scripts/install.sh --version 0.0.0` → `REPO=yoshpy-dev/ralph`, `URL=https://github.com/yoshpy-dev/ralph/releases/download/v0.0.0/ralph_0.0.0_darwin_arm64.tar.gz` (no old-repo substring) |
| `./scripts/run-verify.sh` full pass (called via `run-test.sh`) | PASS | local verifier (mojibake 11/11) + go verifier (gofmt ok, staticcheck 0 issues, all go packages ok) |
| Go module path migration leaves tests green | PASS | `go test ./...` all `ok`, no `cannot find module` / `import path` errors |
| `git remote -v` points at new URL | PASS | `origin  ssh://git@github.com/yoshpy-dev/ralph.git (fetch/push)` |
| Built binary self-identifies correctly | PASS | `ralph version` → `ralph dev (unknown unknown)` (expected for a non-goreleaser dev build); `ralph --help` leads with `ralph is a CLI tool for harness engineering.` |
| `ralph init` scaffolds cleanly into an empty directory | PASS | 114 files (106 base + 5×2 pack + `.ralph/manifest.toml` + `git init`) in `/tmp/ralph-smoke-MKV23L` |
| `ralph doctor` passes on a freshly-scaffolded project | PASS | 5/5 checks pass (Claude Code CLI, Hooks integrity, Manifest version = dev, Language packs = none, Go) |

## Edge cases

- Archived / historical docs retain old repo name by design (Non-goals in the plan). A repo-wide `rg harness-engineering-scaffolding-template` returns 7 matches, all in approved residue locations:
  - `docs/plans/active/2026-04-22-rename-to-ralph-cli.md` (the active plan itself — intentional; ignored per Non-goals)
  - `docs/plans/archive/2026-04-16-ralph-cli-tool.md`
  - `docs/plans/archive/2026-04-17-allow-go-and-repo-commands.md`
  - `docs/plans/archive/2026-04-17-mojibake-postedit-guard.md`
  - `docs/specs/2026-04-16-ralph-cli-tool.md`
  - `docs/reports/verify-2026-04-22-rename-to-ralph-cli.md` (context references to the rename itself)
  - `docs/reports/self-review-2026-04-22-rename-to-ralph-cli.md` (same)
- Runtime surface (`cmd/`, `internal/`, `scripts/`, `go.mod`, `.goreleaser.yml`, `README.md`, `AGENTS.md`, `CLAUDE.md`): 0 hits. No user-facing leakage.
- GitHub redirect (old URL → new URL): not re-verified in this run; already captured in the verify report. The plan's `git remote` state confirms the remote is updated, so any `git clone` against the old URL would rely on GitHub's automatic 301 (outside the scope of an automated unit test and covered by the plan's manual-confirmation step).
- `install.sh` with a sentinel version `0.0.0` intentionally returns HTTP 404 from GitHub release assets — URL assembly is the target of this check, not a successful download.

## Test gaps

- No automated integration test exists for `scripts/install.sh` end-to-end success (i.e., actually pulling a real release and extracting). This is a pre-existing gap, not introduced by this PR, and is already implicitly covered by the next Homebrew release cycle.
- No automated test pins the expected `ralph --help` leading line (`ralph is a CLI tool for harness engineering.`). Tolerable because the strings are inline constants in `cmd/ralph` and cheap to eyeball; add a smoke test only if branding drifts again.
- `go test` runs without `-race` in the standard suite. This is project policy, not a regression from this PR.

## Verdict

- Pass: **yes**
- Fail: 0
- Blocked: 0

All unit, integration, regression, and edge checks required by the plan's Test plan section pass. `tester` recommends proceeding to `/sync-docs` → `/codex-review` → `/pr`.
