# ralph CLI ツール化

- Status: In Progress (Phase 6b, 9 remaining)
- Owner: Claude Code
- Date: 2026-04-16
- Related request: テンプレートリポジトリから配布可能な CLI ツールへの変換
- Related issue: N/A
- Related spec: docs/specs/2026-04-16-ralph-cli-tool.md
- Branch: feat/ralph-cli-tool (created)
- Flow: 標準フロー (/work)

## Objective

harness-engineering-scaffolding-template を `ralph` CLI ツールに変換する。`ralph init` でスキャフォールド展開、`ralph upgrade` でテンプレート更新、パイプラインコアロジックを Go に統合し、Homebrew / curl / GitHub Releases で配布する。

## Scope

- リポ構造の再編（`cmd/ralph/`, `templates/`, `internal/` 再構成）
- cobra + huh によるサブコマンド体系の実装
- `go:embed` テンプレート埋め込み + マニフェスト管理
- `ralph init` / `ralph upgrade` / `ralph doctor` / `ralph pack add`
- パイプラインコアロジック（orchestrator, pipeline, claude -p 呼び出し）の Go 移植
- TOML 設定（`ralph.toml`）+ プロンプトテンプレート解決
- TUI の `cmd/ralph/` への統合
- goreleaser + Homebrew tap + curl installer

## Non-goals

- Claude Code 以外のエージェント対応
- リモートテンプレートレジストリ
- Web UI / GUI
- Go template 動的レンダリング
- プラグインシステム

## Assumptions

- Go 1.25.0 を使用（現状の go.mod と同一）
- Bubble Tea v2 / lipgloss v2 は既存を維持
- `claude` CLI がユーザー環境に存在する前提（`ralph doctor` で検証）
- GitHub リポジトリ名は変更しない（またはリネームは別タスク）

## Affected areas

| 領域 | 変更内容 |
|------|---------|
| `cmd/ralph-tui/` → `cmd/ralph/` | エントリポイント統合・cobra root |
| `internal/cli/` (新規) | 全サブコマンド実装 |
| `internal/scaffold/` (新規) | テンプレート埋め込み・マニフェスト |
| `internal/upgrade/` (新規) | diff・コンフリクト解決 |
| `internal/pipeline/` (新規) | orchestrator / runner / claude 呼び出し |
| `internal/config/` (新規) | TOML パーサー |
| `internal/prompt/` (新規) | プロンプト解決（プロジェクト側 → フォールバック） |
| `internal/{state,watcher,ui,action}` | 既存パッケージの import path 調整 |
| `templates/` (新規) | 現在のスキャフォールドファイルを移動 |
| `scripts/` | ユーザー向けは templates/ へ、ビルド用のみ残す |
| `go.mod` | cobra, huh, go-toml/v2 追加 |
| `.goreleaser.yml` (新規) | リリース自動化 |

## Acceptance criteria

### ビルド・基盤

- [ ] AC1: `go build ./cmd/ralph/` でシングルバイナリがビルドできる
- [ ] AC2: `ralph version` が `semver + commit SHA + build date` を stdout に出力する（例: `ralph 0.1.0 (abc1234 2026-04-16T10:00:00Z)`）
- [ ] AC3: `ralph help` が全サブコマンド（init, upgrade, run, status, retry, abort, doctor, pack, version）を一覧表示する
- [ ] AC16: 全既存テストが新パッケージ構造でパスする

### init（トランザクション安全性含む）

- [ ] AC4: `ralph init` が新規ディレクトリに `.claude/`, `AGENTS.md`, `CLAUDE.md`, `docs/`, `ralph.toml`, `.ralph/manifest.toml` を生成する
- [ ] AC5: `ralph init` が既存プロジェクトでファイル単位上書きし、`templates/` に存在しないユーザー独自ファイルを保持する
- [ ] AC6: `ralph init` で `.git` 未存在時に自動 `git init`、存在時にスキップ
- [ ] AC17: `ralph init` が途中で中断された場合（SIGINT / 権限エラー）、マニフェストがディスク上のファイル状態と整合していること（マニフェストは全ファイル書き込み成功後に最後に書かれる）
- [ ] AC18: `ralph init` 中断後に再度 `ralph init` を実行して正常に完了できること

### upgrade（トランザクション安全性 + ファイルライフサイクル）

- [ ] AC7: `ralph upgrade` で未編集ファイル（マニフェストのハッシュと一致）は自動上書き、編集済みファイルはコンフリクトUI（overwrite / skip / diff）を表示
- [ ] AC19: `ralph upgrade` でテンプレートから削除されたファイルは、警告付きでユーザーに通知される（自動削除しない）
- [ ] AC20: `ralph upgrade` でリネームされたファイルは、旧ファイルの削除通知 + 新ファイルの追加として処理される
- [ ] AC21: `ralph upgrade` が途中で中断された場合、`.ralph/journal.toml` に進捗が記録され、再実行時にリカバリできること
- [ ] AC22: `ralph upgrade` 後、マニフェストに存在しないユーザー作成ファイル（管理ディレクトリ `.claude/` 内含む）が一切変更されていないこと

### doctor

- [ ] AC8: `ralph doctor` が以下を検証し、各項目の pass/warn/fail を表示する: (a) Claude Code CLI のPATH存在, (b) `.claude/settings.json` 内hookの実ファイル整合性, (c) マニフェストバージョンと CLI バージョンの差分, (d) 言語パック `verify.sh` の実行可能性, (e) Go の利用可能性（設定で `require_go = true` の場合）

### pack

- [ ] AC9: `ralph pack add <lang>` が `templates/packs/<lang>/` の内容を展開し、マニフェストに追加する

### パイプライン（段階的移行: ラッパー → Go ネイティブ）

- [ ] AC10a: (Phase 6a) `ralph run` が既存 `scripts/ralph-orchestrator.sh` をラップして実行する（TOML 設定を環境変数に変換して渡す）
- [ ] AC10b: (Phase 6b) `ralph run` が Go ネイティブで `claude -p` パイプラインを実行する
- [ ] AC11: `ralph status` が TTY で TUI 起動、`--no-tui` でテーブル出力、`--json` でJSON出力
- [ ] AC12: `ralph retry` / `ralph abort` が既存シェルスクリプトと同一の状態遷移を行う。具体的に: (a) checkpoint.json の phase_transitions が同じ遷移を記録, (b) `.harness/state/` 内のファイル構造が同一, (c) resume 時に checkpoint から復帰可能, (d) abort 時に worktree cleanup と state archival が完了

### 設定・プロンプト

- [ ] AC13: `ralph.toml` の全フィールド（model, effort, max_iterations, max_parallel, slice_timeout, permission_mode, prompts.dir）が正しくパースされ、デフォルト値にフォールバックする
- [ ] AC14: `.ralph/prompts/self-review.md` が存在する場合はそれが `claude -p` に渡され、存在しない場合はバイナリ内蔵デフォルトが使われる

### 配布

- [ ] AC15: goreleaser で darwin/linux × amd64/arm64 のクロスビルドが成功する
- [ ] AC23: `scripts/install.sh` が HTTPS + チェックサム検証を含み、OS/arch を自動検出してバイナリをインストールする
- [ ] AC24: `ralph init` の実行時間が10秒以内、`ralph upgrade`（ファイル数100未満）が5秒以内であること

## Implementation outline

### Phase 1: リポ構造再編 + cobra 基盤

1. `cmd/ralph/main.go` 作成（cobra root command）
2. `internal/cli/root.go` に cobra 初期化、`version.go` に version サブコマンド
3. `internal/cli/help.go` に help サブコマンド
4. 既存 `cmd/ralph-tui/main.go` のロジックを `internal/cli/status.go` に統合
5. `go.mod` に `spf13/cobra` 追加
6. 既存テストが新構造でパスすることを確認

### Phase 2: テンプレート埋め込み + scaffold 基盤

1. `templates/base/` にスキャフォールドファイルを移動（`.claude/`, `AGENTS.md`, `CLAUDE.md`, `docs/`, `ralph.toml`）
2. `templates/packs/` に `packs/languages/` を移動
3. `templates/prompts/` にデフォルトプロンプトテンプレート作成
4. `internal/scaffold/embed.go` に `go:embed` 定義
5. `internal/scaffold/manifest.go` にマニフェスト TOML 読み書き
6. `internal/scaffold/render.go` にファイル展開ロジック（SHA256 ハッシュ計算含む）

### Phase 3: ralph init（トランザクション安全性付き）

1. `internal/cli/init.go` に init サブコマンド
2. `charmbracelet/huh` でインタラクティブプロンプト（プロジェクト名、言語パック、Ralph Loop、TUI）
3. `internal/scaffold/transaction.go` にトランザクション書き込みロジック:
   - ステップ1: temp ディレクトリ（`.ralph/.staging/`）にファイルを展開
   - ステップ2: `.ralph/journal.toml` に書き込み対象リストを記録
   - ステップ3: staging → 最終パスへ atomic move（ファイルごと）
   - ステップ4: 全ファイル成功後にマニフェスト書き込み（最後に書く）
   - ステップ5: journal と staging を削除
   - 中断時: 次回 init/upgrade で journal を検出し、リカバリまたはクリーンアップ
4. `.git` 自動初期化ロジック
5. 既存プロジェクト後付け時のファイル単位上書きロジック

### Phase 4: ralph upgrade（ファイルライフサイクル管理付き）

1. `internal/upgrade/diff.go` にハッシュ比較ロジック
2. `internal/upgrade/conflict.go` にコンフリクト解決 UI（overwrite / skip / diff）
3. `internal/upgrade/lifecycle.go` にファイルライフサイクル管理:
   - **追加**: テンプレートに新ファイルがある → 追加（通知）
   - **更新**: テンプレートのハッシュが変わった → 未編集なら自動上書き、編集済みならコンフリクトUI
   - **削除**: テンプレートからファイルが消えた → 警告を表示し削除はユーザー判断（自動削除しない）
   - **リネーム**: 旧パス削除通知 + 新パス追加として処理
   - **ユーザー所有**: マニフェストに存在しないファイルは一切触らない（`.claude/` 内含む）
4. `internal/cli/upgrade.go` にサブコマンド
5. トランザクション書き込み（Phase 3 の transaction.go を再利用）
6. マニフェスト更新ロジック（journal 付き、最後に書き込み）

### Phase 5: TOML 設定 + プロンプト解決

1. `internal/config/config.go` に `ralph.toml` パーサー（`pelletier/go-toml/v2`）
2. `internal/prompt/resolver.go` にプロンプトテンプレート解決（プロジェクト側 → 内蔵フォールバック）

### Phase 6a: パイプライン — ラッパーファースト（既存シェルエンジン維持）

1. `internal/cli/run.go` — `ralph.toml` を読み、環境変数に変換して `scripts/ralph-orchestrator.sh` を `os/exec` で呼び出し
2. `internal/cli/retry.go` — 同様に `scripts/ralph` の retry ロジックをラップ
3. `internal/cli/abort.go` — 同様にラップ
4. パリティテスト作成: 既存シェルスクリプトの出力（checkpoint.json, `.harness/state/` 構造, report ファイル）をスナップショットし、Go ラッパー経由で同一結果を確認

### Phase 6b: パイプライン — Go ネイティブ移植

1. `internal/pipeline/orchestrator.go` — マルチワークツリー並列実行
2. `internal/pipeline/runner.go` — スライスごとのパイプライン（Inner/Outer Loop アーキテクチャ維持）
3. `internal/pipeline/claude.go` — `claude -p` の `os/exec` 呼び出し（引数リスト形式、シェル経由なし）
4. `internal/pipeline/checkpoint.go` — checkpoint.json 読み書き（既存フォーマット互換）
5. `internal/pipeline/stuck.go` — stuck detection（3連続ノーコミット検出）
6. `internal/pipeline/cleanup.go` — abort 時の worktree cleanup + state archival
7. Phase 6a のパリティテストが Go ネイティブ実装でもパスすることを確認
8. パリティテスト全パス後に Phase 6a のラッパーコードを削除

### Phase 7: doctor + pack add

1. `internal/cli/doctor.go` — 環境チェック
2. `internal/cli/pack.go` — 言語パック追加

### Phase 8: 配布設定

1. `.goreleaser.yml` 作成
2. Homebrew tap リポジトリ用 Formula テンプレート
3. `scripts/install.sh` curl ワンライナー用インストーラー
4. GitHub Actions ワークフロー（release.yml）

### Phase 9: クリーンアップ

1. 旧 `cmd/ralph-tui/` 削除
2. 旧 `scripts/ralph`, `scripts/ralph-*.sh` 削除
3. 旧 `packs/languages/` 削除（templates/ に移動済み）
4. README.md, AGENTS.md, CLAUDE.md をツール向けに更新

## Verify plan

- Static analysis checks: `go vet ./...`, `staticcheck ./...` が全パッケージでパスすること
- Spec compliance criteria to confirm:
  - 全サブコマンド（init, upgrade, run, status, retry, abort, doctor, pack, version, help）が `ralph help` 出力に存在
  - `templates/` 内のファイルが `go:embed` で正しく埋め込まれている（`TestEmbedFS` で全ファイル走査）
  - マニフェスト TOML が spec 定義のスキーマ（`[meta]` + `[files]`）で生成される
  - TOML 設定の全フィールドがパース可能で、未指定フィールドにデフォルト値が適用される
  - `ralph doctor` が spec 記載の5項目（Claude CLI, hooks整合性, バージョン差分, 言語パック, Go）を検証
  - `ralph init` が10秒以内、`ralph upgrade`（100ファイル未満）が5秒以内で完了（ベンチマークテスト）
  - パイプラインパリティ: Go 実装の checkpoint.json / `.harness/state/` 出力がシェルスクリプトのスナップショットと構造的に一致
- Documentation drift to check: README.md, AGENTS.md, CLAUDE.md がツール配布モデルを反映していること
- Evidence to capture: `ralph version` 出力、`ralph init` のスキャフォールド生成結果、`ralph doctor` 出力、パリティテスト結果

## Test plan

- Unit tests:
  - `internal/scaffold/` — マニフェスト読み書き、ハッシュ計算、ファイル展開、トランザクション書き込み
  - `internal/upgrade/` — ハッシュ比較、コンフリクト検出、ファイルライフサイクル（追加/更新/削除/リネーム）
  - `internal/config/` — TOML パース、デフォルト値、全フィールドのラウンドトリップ
  - `internal/prompt/` — フォールバック解決
  - `internal/pipeline/` — コマンド構築、設定マッピング、checkpoint 読み書き、stuck detection
- Integration tests:
  - `ralph init` → 一時ディレクトリでスキャフォールド生成 → マニフェスト検証
  - `ralph init` → 既存ファイルあり → ファイル単位上書き検証（ユーザー独自ファイル保持）
  - `ralph init` → SIGINT 中断 → 再実行でリカバリ完了
  - `ralph upgrade` → 編集済み/未編集ファイルの振り分け検証
  - `ralph upgrade` → テンプレートからファイル削除 → 警告表示・ユーザーファイル無変更確認
  - `ralph upgrade` → 中断 → journal からリカバリ
  - `ralph doctor` → 各チェック項目の pass/warn/fail 出力検証
  - パイプラインパリティ: シェルスクリプトスナップショット vs Go 実装出力の構造比較
- Regression tests: 既存 `internal/{state,watcher,ui,action}` のテストが全パス
- Edge cases:
  - 空ディレクトリへの init
  - 破損したマニフェストでの upgrade（パース失敗 → エラーメッセージ + 修復提案）
  - `claude` CLI 未インストール時の run（明確なエラー + `ralph doctor` 案内）/ doctor（warn 表示）
  - 読み取り専用ディレクトリへの init（権限エラー → ロールバック）
  - `.ralph/.staging/` が残っている状態での init/upgrade（前回中断の検出 → リカバリ提案）
  - 管理ディレクトリ内のユーザー作成ファイル（`.claude/rules/my-custom.md`）が upgrade で無変更
- Benchmark tests:
  - `ralph init` の実行時間 < 10s
  - `ralph upgrade`（100ファイル未満）の実行時間 < 5s
- Evidence to capture: テスト実行結果、カバレッジレポート、パリティテストスナップショット

## Risks and mitigations

| リスク | 影響 | 確率 | 軽減策 |
|--------|------|------|--------|
| Go 移植のシェルスクリプトとの振る舞い差異 | パイプライン障害 | 高 | **Phase 6a でラッパーファースト** → パリティテスト作成 → Phase 6b でネイティブ移植。パリティテスト全パスまでシェルスクリプト削除禁止 |
| init/upgrade 中断によるユーザーリポ破損 | ユーザーデータ損失 | 中 | **トランザクション書き込み**: staging → journal → atomic move → マニフェスト最後。中断テスト必須 |
| ファイル削除/リネーム時のユーザーファイル誤削除 | ユーザーデータ損失 | 中 | **ファイルライフサイクルマトリクス**: 削除は警告のみ（自動削除しない）、マニフェストにないファイルは一切触らない |
| `go:embed` のファイルサイズ増大 | バイナリ肥大化 | 低 | テンプレートは小さいテキストファイル。監視のみ |
| 既存テストの import path 破壊 | 開発停滞 | 中 | Phase 1 で構造変更後すぐにテスト修正 |
| cobra + huh の組み合わせの学習コスト | 実装遅延 | 低 | 両ライブラリとも十分なドキュメントあり |
| 既存ユーザー（テンプレートリポからフォーク済み）への影響 | 混乱 | 中 | README に移行ガイドを記載。旧テンプレート使用は引き続き可能 |
| 「シングルバイナリ」と外部依存 `claude` CLI の矛盾 | ユーザー混乱 | 低 | README と `ralph doctor` で明示: バイナリは外部ランタイム不要だが、パイプライン実行には `claude` CLI が必要 |

## Rollout or rollback notes

- Phase 1 完了時点で `go build` が通ることを確認。通らなければ構造変更を巻き戻し
- 各フェーズは独立にコミット。問題発生時は該当フェーズのみ revert 可能
- 既存の `scripts/ralph` は **Phase 6b パリティテスト全パス後まで残す**。Go 実装が全機能を代替するまで削除しない
- Phase 6a（ラッパー）は安全なチェックポイント: この時点で Go CLI + 既存シェルエンジンが共存し、ロールバック不要
- init/upgrade の中断リカバリ: `.ralph/journal.toml` と `.ralph/.staging/` による自動回復
- v0.x でリリースし、安定したら v1.0 に移行

## Open questions

（なし — 全て spec で解決済み）

## Progress checklist

- [x] Plan reviewed (Codex advisory: 4 findings → all addressed)
- [x] Branch created (feat/ralph-cli-tool)
- [x] Spec completed (docs/specs/2026-04-16-ralph-cli-tool.md)
- [x] Phase 1: リポ構造再編 + cobra 基盤 (fbb999b)
- [x] Phase 2: テンプレート埋め込み + scaffold 基盤 (b1c71eb)
- [x] Phase 3: ralph init（トランザクション安全性付き）(7cb959b) — 基本機能完了、トランザクション安全性は Phase 4 と共に追加予定
- [x] Phase 4: ralph upgrade（ファイルライフサイクル管理付き）(4055c4b)
- [x] Phase 5: TOML 設定 + プロンプト解決 (6c68acb)
- [x] Phase 6a: パイプライン — ラッパーファースト (e30ef9c) — パリティテストは Phase 6b で追加
- [ ] Phase 6b: パイプライン — Go ネイティブ移植（パリティテスト全パス必須）— 別PRで実施
- [x] Phase 7: doctor + pack add (e30ef9c)
- [x] Phase 8: 配布設定 (c04d2d3)
- [ ] Phase 9: クリーンアップ（パリティテスト全パス後のみ旧スクリプト削除）
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
