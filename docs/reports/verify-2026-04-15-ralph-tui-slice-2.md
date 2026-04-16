# Verify report: ralph-tui (slice-2)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/slice-2-ralph-tui.md
- Verifier: pipeline-verify (autonomous)
- Scope: spec compliance + static analysis + documentation drift
- Evidence: `docs/evidence/verify-2026-04-15-073224.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| `internal/watcher/watcher.go` が `.harness/state/orchestrator/` と worktree 内の `checkpoint.json` を監視できること | met | watcher.go:47-49 orchDir を fsnotify に登録、watcher.go:218-234 `addWorktreeWatches()` が worktree の `.harness/state/pipeline/` を走査、watcher.go:204-212 `collectWatchPaths()` が `checkpoint.json` パスを収集 |
| ファイル変更時に `StateChangedMsg` が Bubble Tea のメッセージチャネルに送信されること | met | watcher.go:109-112 eventLoop が fsnotify イベントを `StateChangedMsg` に変換、watcher.go:171-177 scanFiles がポーリング時に同メッセージを送信。テスト `TestStateChangedMsg_OnFileWrite`、`TestStateChangedMsg_OnFileCreate` で確認 |
| ログファイルの末尾追従が動作し、新しい行が追加されると `LogLineMsg` として配信されること | met | tailer.go:124-136 `readLoop` が 200ms ポーリングで新行を検出、tailer.go:182 `LogLineMsg` を送信。テスト `TestTailer_NewLines`、`TestTailer_MultipleLines` で確認 |
| 監視対象ファイルが存在しない場合にパニックせず graceful にスキップすること | met | watcher.go:47 `os.Stat` ガード付き、watcher.go:222-224 ReadDir エラーで静かに return、tailer.go:39-43 `os.IsNotExist` で `waitForFile` に遷移。テスト `TestWatcher_GracefulOnMissingDir`、`TestTailer_MissingFile` で確認 |
| `watcher.Stop()` でクリーンアップが行われること | met | watcher.go:90-99 `sync.Once` で done チャネル close + fsWatcher.Close()、tailer.go:111-121 同様に done close + file.Close()。テスト `TestWatcher_StopCleanup`、`TestTailer_StopCleanup` で二重 Stop 安全性と Stop 後挙動を確認 |
| ポーリングフォールバック (5秒間隔) が利用可能なこと | met | watcher.go:35 `pollInterval: 5 * time.Second`、watcher.go:39-42 fsnotify 失敗時にポーリング切り替え、watcher.go:60-71 `NewWithPolling` コンストラクタ、watcher.go:134-150 `pollLoop` 実装。テスト `TestWatcher_PollingFallback`、`TestWatcher_PollingDetectsNewFile`、`TestWatcher_PollingDetectsRemoval` で確認 |
| テストカバレッジが 80% 以上であること | likely but unverified | テストファイルは 455 行、14 テストケースで包括的。カバレッジ数値はテストエージェントが `go test -cover` で計測する |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass | gofmt: ok, 0 issues |
| `go vet ./internal/watcher/...` | pass | 出力なし（問題なし） |
| `go build ./internal/watcher/...` | pass | コンパイル成功 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `CLAUDE.md` | yes | watcher パッケージはインフラ内部。CLAUDE.md への記載不要（マニフェストの Integration-level verify plan に「TUI 関連の記述追加は不要」と明記） |
| `AGENTS.md` | yes | 同上。`internal/watcher/` は外部契約に影響しない |
| `.claude/rules/architecture.md` | yes | 新パッケージは grep-able な命名、明示的モジュール境界の規則に準拠 |
| `.claude/rules/testing.md` | yes | テストスイートは edge case を含み、テスト名が意図を説明している |
| `docs/plans/active/2026-04-15-ralph-tui/slice-2-ralph-tui.md` | yes | 実装がプランの outline と一致 |
| `README.md` | yes | ユーザー向け動作変更なし。watcher は内部パッケージ |

## Observational checks

- **メッセージ型の tea.Msg 準拠**: `StateChangedMsg`、`LogLineMsg`、`WatcherErrorMsg` は全て Bubble Tea の `tea.Msg` インタフェース（空インタフェース）を満たす。`WatcherErrorMsg` は追加で `error` インタフェースも実装（messages.go:28）。
- **並行安全性**: Tailer は `sync.Mutex` でファイルハンドルとオフセットを保護。Watcher はバッファ付きチャネル（cap=64）とノンブロッキング `sendMsg` で goroutine 安全性を確保。
- **self-review の MEDIUM 指摘との関連**: goroutine リーク（SwitchFile）と fsWatcher.Add エラー無視は self-review で正しく捕捉されている。spec compliance の観点では機能要件を満たしており、これらは tech debt として追跡済み。

## Coverage gaps

- **テストカバレッジ 80% 数値**: テストエージェントが計測するため、本検証では定量確認していない。テストケースの網羅性から 80% 以上は妥当と推定。
- **fsnotify 経由の watcher テスト**: `TestStateChangedMsg_OnFileWrite` と `TestStateChangedMsg_OnFileCreate` は fsnotify パスをテストしているが、CI 環境（Docker 内等）では fsnotify が不安定な場合がある。ポーリングテストは安定。
- **worktree checkpoint 監視**: `addWorktreeWatches` と `collectWatchPaths` の worktree 走査ロジックは直接テストされていない（統合テストで間接的にカバー）。

## Verdict

- Verified: AC 1-6（fsnotify 監視、StateChangedMsg 送信、LogLineMsg 配信、graceful スキップ、Stop クリーンアップ、ポーリングフォールバック）
- Partially verified: AC 7（テストカバレッジ 80% — テストエージェントで計測予定）
- Not verified: なし
