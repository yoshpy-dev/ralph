# fix-template-distribution-gaps

- Status: Implementation Complete
- Owner: Claude Code
- Date: 2026-04-16
- Related request: fix-template-distribution-gaps
- Related issue: N/A
- Branch: fix/template-distribution-gaps
- Flow: 標準フロー (/work)

## Objective

`ralph init` で生成されるプロジェクトが、ドキュメント（CLAUDE.md, rules, skills）で参照しているスクリプト群を含んでおらず、開発フロー全体が動作しない問題を修正する。

## Scope

1. **templates/base/scripts/** にワークフローに必要なスクリプトを追加する
2. **commit-msg-guard.sh** の参照を修正する（実体がないため）
3. **quality-gates.md** のこのリポジトリ固有の参照を汎用化する（CI ワークフロー参照含む）
4. go:embed でスクリプトが正しく埋め込まれることを検証する
5. **upgrade.go** の `.sh` ファイルパーミッション修正（Codex 指摘 #1）

## Non-goals

- `.github/workflows/` テンプレートの追加（別タスク）
- Ralph Loop スクリプト群のリファクタリング
- スクリプトの機能変更
- 新しいスクリプトの作成

## Assumptions

- `render.go` は `.sh` ファイルに自動的に 0755 パーミッションを付与する（確認済み: L88）
- `go:embed` は `templates/` 配下の全ファイルを再帰的に埋め込む
- ソースリポのスクリプトはそのまま汎用プロジェクトでも動作する設計

## Affected areas

- `templates/base/scripts/` — 新規ディレクトリ（16スクリプト追加）
- `templates/base/.claude/rules/git-commit-strategy.md` — commit-msg-guard.sh 参照修正
- `templates/base/docs/quality/quality-gates.md` — このリポ固有の参照・CI ワークフロー参照を汎用化
- `internal/cli/upgrade.go` — `.sh` ファイルのパーミッション修正（0644 → 0755）
- `internal/scaffold/embed_test.go` — テンプレート検証テストの拡張

## Acceptance criteria

- [ ] AC1: `templates/base/scripts/` に以下の16スクリプトが存在する
  - run-verify.sh, run-static-verify.sh, run-test.sh
  - detect-languages.sh
  - archive-plan.sh, new-feature-plan.sh, new-ralph-plan.sh
  - codex-check.sh
  - ralph-loop-init.sh, ralph-loop.sh
  - ralph, ralph-config.sh, ralph-orchestrator.sh, ralph-pipeline.sh, ralph-status-helpers.sh
  - commit-msg-guard.sh
- [ ] AC2: commit-msg-guard.sh が実体として存在し、git-commit-strategy.md の参照が正しい
- [ ] AC3: quality-gates.md がこのリポ固有のスクリプト（check-template.sh, build-tui.sh 等）や存在しない CI ワークフローを参照していない
- [ ] AC4: `go build ./cmd/ralph/` が成功する（embed にスクリプトが含まれる）
- [ ] AC7: `upgrade.go` が `.sh` ファイルを `0755` パーミッションで書き込む
- [ ] AC5: `go test ./internal/scaffold/...` がスクリプトの存在を検証するテストを含みパスする
- [ ] AC6: 既存テスト (`go test ./...`) が全てパスする

## Implementation outline

### Slice 1: スクリプトの分類とコピー

配布すべきスクリプト（テンプレートから参照されている）と、このリポ固有のスクリプトを分類：

**配布する（templates/base/scripts/ にコピー）:**
- `run-verify.sh`, `run-static-verify.sh`, `run-test.sh` — 検証パイプライン
- `detect-languages.sh` — 言語検出（run-verify.sh が依存）
- `archive-plan.sh` — /pr スキルが使用
- `new-feature-plan.sh`, `new-ralph-plan.sh` — /plan スキルが使用
- `codex-check.sh` — /plan, /codex-review スキルが使用
- `ralph-loop-init.sh`, `ralph-loop.sh` — /loop スキルが使用
- `ralph`, `ralph-config.sh`, `ralph-orchestrator.sh`, `ralph-pipeline.sh`, `ralph-status-helpers.sh` — Ralph Loop 実行基盤

**配布しない（このリポ固有）:**
- `bootstrap.sh` — このリポのブートストラップ
- `build-tui.sh` — TUI バイナリビルド
- `install.sh` — curl 経由のインストーラー
- `init-project.sh` — レガシー init（Go 版に置換済み）
- `check-template.sh` — テンプレート検証（このリポ専用）
- `check-coverage.sh` — 言語パックカバレッジ（このリポ専用）
- `check-pipeline-sync.sh` — パイプライン同期チェック（このリポ専用）
- `new-language-pack.sh` — 言語パック作成（このリポ専用）

### Slice 2: commit-msg-guard.sh の作成

テンプレートの git-commit-strategy.md が参照している `commit-msg-guard.sh` を作成。ソースリポの `scripts/commit-msg-guard.sh` をベースに `templates/base/scripts/` に配置。

### Slice 3: quality-gates.md の修正

`templates/base/docs/quality/quality-gates.md` から：
- 配布しないスクリプトへの参照を削除または汎用化
- 存在しない `.github/workflows/` への参照を「要設定」に変更（Codex 指摘 #3）

### Slice 4: upgrade.go のパーミッション修正

`internal/cli/upgrade.go` の ActionAutoUpdate, ActionConflict (overwrite), ActionAdd で `.sh` ファイルに `0755` パーミッションを付与する。`render.go` と同じロジック。（Codex 指摘 #1）

### Slice 5: テスト追加と検証

- `internal/scaffold/embed_test.go` に、`templates/base/scripts/` の必須スクリプト存在確認テストを追加
- `go test ./...` で全テストパス確認
- `go build ./cmd/ralph/` でビルド確認

## Verify plan

- Static analysis checks: `go vet ./...`, `go build ./cmd/ralph/`
- Spec compliance criteria: templates/base/ 内の全ドキュメントが参照するスクリプトが templates/base/scripts/ に存在する
- Documentation drift: quality-gates.md がこのリポ固有スクリプトを参照していない
- Evidence: verify レポートに grep 結果を記録

## Test plan

- Unit tests: `go test ./internal/scaffold/...` — embed テストでスクリプト存在を検証
- Integration tests: `go test ./...` — 全テストパス
- Regression tests: 既存のテスト全てがパスすること
- Edge cases: .sh パーミッション（0755）が render.go で正しく付与されること
- Evidence: テスト結果を docs/reports/ に保存

## Risks and mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| スクリプトがこのリポ固有のパスをハードコードしている | 配布先で動かない | コピー前にパス依存をレビュー |
| go:embed のサイズ増加 | バイナリが大きくなる | ralph-orchestrator.sh (32KB) + ralph-pipeline.sh (41KB) で計 ~100KB 増。許容範囲 |
| 既存テストが壊れる | CI 失敗 | テスト追加前に既存テスト全パス確認 |

## Rollout or rollback notes

- テンプレートファイルの追加のみ。既存ファイルの変更は quality-gates.md と git-commit-strategy.md のみ
- `ralph upgrade` でスクリプトが既存プロジェクトに追加される（新規ファイルは add 判定）

## Open questions

なし

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (fix/template-distribution-gaps)
- [x] Slice 1: スクリプトのコピー (ff25006)
- [x] Slice 2: commit-msg-guard.sh の作成 (84fed56)
- [x] Slice 3: quality-gates.md の修正 (62e8c76)
- [x] Slice 4: upgrade.go のパーミッション修正 (d644bf1)
- [x] Slice 5: テスト追加と検証 (097b506)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
