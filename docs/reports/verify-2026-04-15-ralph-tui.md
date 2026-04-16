# Verify report: ralph-tui (cycle 2)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/_manifest.md (slice-6)
- Verifier: pipeline-verify (autonomous)
- Scope: spec compliance + static analysis + documentation drift
- Evidence: `docs/evidence/verify-2026-04-15-ralph-tui-c2.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| `cmd/ralph-tui/main.go` が全コンポーネントを初期化し `tea.NewProgram` で TUI を起動すること | met | `main.go:56-82`: state.ReadFullStatus, watcher.New, action.NewExecutor, ui.New all initialized; `tea.NewProgram(model)` at line 79 |
| `go build -o bin/ralph-tui ./cmd/ralph-tui` で単一バイナリが生成されること | met | `go build -ldflags="-s -w" -o /tmp/ralph-tui-verify ./cmd/ralph-tui` succeeded (exit 0) |
| バイナリサイズが 30MB 以下であること (`-ldflags="-s -w"` 適用) | met | 4.3MB (well under 30MB threshold) |
| バイナリに `--version` フラグがあり、ビルド時の git commit hash を表示すること | met | `version.go:6-10`: Version/GitCommit/BuildDate vars via ldflags; `main.go:22-28`: `--version` flag handler; `build-tui.sh:42`: `-X main.GitCommit=${_commit}` |
| `scripts/ralph` の `cmd_status()` に `--no-tui` フラグが追加されていること | met | `scripts/ralph:149`: `--no-tui) _no_tui=1` |
| TTY 検出: TTY + バイナリ存在 + `--no-tui` 未指定 → TUI 起動 | met | `scripts/ralph:176`: `if [ "$_no_tui" -eq 0 ] && [ -t 1 ] && [ -x "$_tui_bin" ]` with source-freshness check before `exec` |
| 非 TTY または `--no-tui` → 既存テーブル出力 | met | `scripts/ralph:199-213`: Falls through to `_render_table` when TUI conditions not met |
| `--json` → 既存 JSON 出力 (TUI に影響しない) | met | `scripts/ralph:158-171`: JSON mode handled before TUI check, calls `_render_json` directly |
| TUI バイナリが存在しない場合に既存出力にフォールバックすること | met | `scripts/ralph:176`: `[ -x "$_tui_bin" ]` check; missing binary falls through to table output |
| TUI バイナリが Go ソースより古い場合に警告を表示してからフォールバックすること | met | `scripts/ralph:178-181`: `find ... -newer "$_tui_bin"` with stderr warning, falls through to table output |
| `scripts/ralph retry <slice-name>` サブコマンドが新設されていること | met | `scripts/ralph:220-309`: `cmd_retry()` function; dispatched at line 468 |
| `retry` がオーケストレータの PID/status/locklist/並列制限を検証してから実行すること | partially met | Status check (lines 236-257), orchestrator state (260-264), parallel limit (267-285), worktree existence (288-292) implemented. **Missing**: locklist conflict check (plan step b) and dependency completion check (plan step d) |
| `scripts/ralph abort --slice <slice-name>` フラグが追加されていること | met | `scripts/ralph:321`: `--slice) shift; _target_slice="${1:?requires slice name}"` |
| `abort --slice` が既存 abort フロー（アーカイブ・監査ログ）を単一スライスに限定して実行すること | met | Lines 339-442: single-slice PID kill, state archive, worktree removal, audit JSON — all scoped to `_target_slice` |
| `scripts/build-tui.sh` が `go build` を実行し `bin/ralph-tui` にバイナリを配置すること | met | `build-tui.sh:41-44`: `go build ... -o "$_output" "${REPO_ROOT}/cmd/ralph-tui"` where `_output="${REPO_ROOT}/bin/ralph-tui"` |
| `.gitignore` に `bin/` が追加されていること | met | `.gitignore:7`: `bin/` |

**Summary**: 15 met, 1 partially met, 0 not met (out of 16 acceptance criteria)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `go vet ./...` | pass | No issues found |
| `go build ./cmd/ralph-tui` | pass | Binary: 4.3MB (stripped) |
| `gofmt -l cmd/ internal/` | pass | No formatting issues |
| `./scripts/run-static-verify.sh` | pass | Go verifier passed (gofmt + go vet + go test compilation) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `CLAUDE.md` | yes | No TUI mentions needed (confirmed by plan: "スコープ外のため") |
| `AGENTS.md` | yes | No TUI mentions needed (confirmed by plan) |
| `.claude/rules/architecture.md` | yes | No TUI-specific rules needed |
| `.claude/rules/testing.md` | yes | No changes needed |
| `README.md` | likely yes | Plan does not list README as requiring updates; TUI is an internal tool |
| `docs/plans/active/2026-04-15-ralph-tui/_manifest.md` | yes | Progress reflects current state |
| `.gitignore` | partial drift | `coverage.out` is tracked in git but not in `.gitignore` (identified by self-review CRITICAL) |

## Observational checks

### Self-review CRITICAL findings (cross-validated)

1. **coverage.out tracked in git**: Confirmed. `git ls-files -- coverage.out` returns a match. 654 lines of generated test coverage data. Not in `.gitignore`. This is a repo hygiene issue, not a spec compliance failure, but should be fixed before merge.

2. **SelfReviewResult type mismatch**: Confirmed. `internal/state/types.go:86` declares `SelfReviewResult *string` but `.harness/state/pipeline/checkpoint.json` writes `"self_review_result": {"critical": 2, "high": 0, "medium": 3, "low": 2}` (an object, not a string). This will cause `json.Unmarshal` to silently fail for any checkpoint that has gone through self-review. Test data uses `null`, masking the bug.

### Retry command gap

`cmd_retry()` is missing two checks specified in the implementation outline:
- **Locklist conflict check** (plan step b): Should verify that no currently-running slice shares locked files with the target slice
- **Dependency completion check** (plan step d): Should verify that all dependency slices (from the manifest dependency graph) have status `complete`

These are safety checks that prevent retry from creating race conditions or starting work with unmet prerequisites.

### Executor error swallowing

`main.go:72-73`: If `action.NewExecutor(repoRoot)` fails, the error is silently discarded and `executor` remains nil. All actions would then fail with no user feedback. (MEDIUM finding from self-review, not a spec compliance issue.)

## Coverage gaps

- **Behavioral verification**: Cannot verify TUI rendering, keybind handling, or pane layout without running the binary interactively or via teatest. This is the test agent's responsibility.
- **Regression verification**: Cannot verify that `ralph status --json` output is identical to pre-change output without a baseline snapshot. This is the test agent's responsibility.
- **Build script**: `scripts/build-tui.sh` was not executed in this verification run (would require Go toolchain and modify `bin/`). Build was verified directly via `go build`.
- **Locklist and dependency checks in retry**: Cannot verify because they are not implemented.

## Verdict

- **Verified** (15 AC): main.go initialization, go build, binary size, --version, --no-tui, TTY detection, non-TTY fallback, --json, binary-missing fallback, outdated-binary warning, retry subcommand exists, abort --slice flag, abort --slice scoping, build-tui.sh, .gitignore bin/
- **Partially verified** (1 AC): retry validation (status + parallel limit verified; locklist + dependency checks missing)
- **Not verified** (0 AC): none

**Overall verdict: partial** — all acceptance criteria are met or partially met, but 2 CRITICAL self-review findings (coverage.out, SelfReviewResult type) remain unfixed and the retry command is missing 2 planned safety checks.
