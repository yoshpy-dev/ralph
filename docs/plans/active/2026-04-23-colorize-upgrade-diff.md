# Colorize and line-number ralph upgrade diff view

- Status: In Progress
- Owner: Claude Code
- Date: 2026-04-23
- Related request: テンプレート配布の上書き/スキップ/diff 表示で diff が視覚的に見えづらいので色付けと行番号付与をしたい
- Related issue: -
- Branch: feat/colorize-upgrade-diff

## Objective

`ralph upgrade` の対話プロンプトで `[d]iff` を選んだときに表示される unified diff を、ANSI 色付け＋行番号付与で人間が一目で読める形に置き換える。

## Scope

- `internal/upgrade/unified_diff.go` の出力フォーマット拡張（行番号メタデータを各行に付与する API を追加、または既存 API を拡張）。
- `internal/upgrade/` に色付けユーティリティ（純粋関数）を追加。
- `internal/cli/upgrade.go` の `showDiff` 経路で TTY/`NO_COLOR` 判定し、端末出力時のみ色を付与。
- 既存テスト（`unified_diff_test.go`、`cli_test.go`）の期待値更新と、新フォーマット・色付け・TTY 判定の追加テスト。

## Non-goals

- TUI (`cmd/ralph-tui`) の色付けは対象外。
- syntax highlighting（言語別）は対象外。色は diff 行種別のみ。
- diff アルゴリズムの差し替え（LCS のまま）。
- Windows コンソールの仮想端末モード自動有効化（go の標準で十分なケースのみ）。

## Assumptions

- 出力先は基本的に `os.Stdout`（cobra/CLI）。テストでは `bytes.Buffer`。
- 既存の `UnifiedDiff` 出力を直接パースしている下流コードは無い（`docs/reports/self-review-...md` で「display artifact」と明記済み）。
- 色付けは「端末への出力」かつ「`NO_COLOR` 未設定」のときのみ。

## Affected areas

- `internal/upgrade/unified_diff.go` — 行番号付与
- `internal/upgrade/colorize.go`（新規）— ANSI 色付けヘルパ
- `internal/upgrade/unified_diff_test.go` — 既存テスト更新
- `internal/upgrade/colorize_test.go`（新規）— 色付け純粋関数テスト
- `internal/cli/upgrade.go` — TTY 判定 + colorize 呼び出し
- `internal/cli/cli_test.go` — `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff` の期待値更新

## Design decisions

- **案B（行番号付与 + ANSI 色付け）採用**。理由: 案A（ハンクヘッダのみ可読化）は実装が軽いが「特定の行がどれか」を目視カウントする問題を解決しない。テンプレ差分は数十〜数百行規模で、行番号付与のコストは無視できる。
- **行番号フォーマット**: `<old> <new> │ <prefix><content>` 形式。例:
  ```
  ── L10 (旧 7行 → 新 8行) ──
    10   10 │  context
    11      │ -removed
         11 │ +added
    12   11 │  context
  ```
  旧/新の行番号を別カラムに配置し、削除行は新カラム空白、追加行は旧カラム空白。
- **色配置**: ヘッダ太字、`-` 系赤、`+` 系緑、ハンクヘッダ（`──` 行）シアン。git の標準配色に揃える。
- **API 形状**: `UnifiedDiff` の戻り値を変えず、新たに `RenderDiff(oldText, newText []byte, opts RenderOptions) string` を追加する案も検討したが、`UnifiedDiff` の内部表現（`hunk` 構造体）を再利用したいので **`UnifiedDiff` 自体の出力フォーマットを変更** する。古い `@@ -10,7 +10,8 @@` フォーマットは破棄。下流パーサ無しのため安全。
- **色付けはレンダリング後に適用**: `UnifiedDiff` は色なしテキストのみ返し、`Colorize(diff string) string` が行頭プレフィックスで色を判定して ANSI を挿入する純粋関数。テストしやすく、TTY/NO_COLOR 判定と分離できる。
- **TTY 判定**: `golang.org/x/term`（既に indirect 依存にある `charmbracelet/x/term`）の `IsTerminal` を使用。`NO_COLOR` 環境変数が空でなければ無効化（標準準拠 https://no-color.org）。

Critical forks: なし（案A vs 案B はユーザーと合意済み）。

## Acceptance criteria

- [ ] `ralph upgrade` の `[d]iff` 出力が、各行に `旧行番号 新行番号` の2カラムを持つ。
- [ ] 端末出力時、`-`/`---` 行は赤、`+`/`+++` 行は緑、ハンク見出しはシアン、ファイルヘッダは太字で表示される。
- [ ] `NO_COLOR=1` を設定するとANSI エスケープが出力されない。
- [ ] パイプ／リダイレクト時（非 TTY）には ANSI エスケープが出力されない。
- [ ] 既存の `UnifiedDiff` の意味的同値性は維持（追加・削除・コンテキストのトリオは変わらない）。
- [ ] `./scripts/run-verify.sh` が通る。

## Implementation outline

1. `internal/upgrade/unified_diff.go`
   - `hunk` の各 op に `oldLineNo`, `newLineNo`（0 ベース内部）を持たせる（既存 `groupHunks` の `indexed` を流用）。
   - レンダ部を「ハンクヘッダ + 各行 `<old4> <new4> │ <prefix><line>`」に書き換える。
   - 旧/新ともに該当無い列は空白で埋める。
   - ハンクヘッダは `── L<oldStart>+<oldCount> → L<newStart>+<newCount> ──` の形式に変更。
2. `internal/upgrade/colorize.go`（新規）
   - `Colorize(diff string) string` を追加。行頭プレフィックス（`---`/`+++`/`──`/`<digits>... │ -`/`<digits>... │ +`）に応じて ANSI 挿入。
   - ANSI 定数を非エクスポートで定義。
3. `internal/cli/upgrade.go`
   - `runUpgrade` で `colorize := !noColor() && term.IsTerminal(int(os.Stdout.Fd()))` を計算し、`runUpgradeIO` に追加引数として渡す。
   - `runUpgradeIO` → `resolveConflict` → `showDiff` まで `colorize bool` を伝播。
   - `showDiff` で `if colorize { diff = upgrade.Colorize(diff) }`。
4. テスト
   - `unified_diff_test.go`: 既存 `assertContains` の `-d\n` / `+d\n` 等を新フォーマット（`│ -d` / `│ +d`）に置き換え。`@@ -1,1 +0,0 @@` 系の検証を `── L1 → L0 ──` 等の新ヘッダに置き換え。
   - `colorize_test.go`（新規）: 色付け前後で行頭にエスケープが入ること、コンテキスト行（` `）には色が付かないこと、`Colorize("")` が空を返すこと。
   - `cli_test.go: TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff`: 期待値を新フォーマットに更新。色付けは TTY 不在のため出ない（`bytes.Buffer` 経由）ことを確認するアサーションを追加。

## Verify plan

- Static analysis checks: `gofmt`, `go vet`, `go build ./...`
- Spec compliance criteria to confirm:
  - Acceptance criteria 6 項目を `docs/reports/verify-...md` で列挙し○×を付与。
- Documentation drift to check:
  - `README.md` / `docs/` に diff 出力サンプルがあれば更新。
  - `CHANGELOG.md` 相当があれば追記。
- Evidence to capture: `go test ./internal/upgrade/... ./internal/cli/... -run Diff` の出力。

## Test plan

- Unit tests:
  - `TestUnifiedDiff_*`: 行番号カラムの正しさ、削除/追加で対側カラムが空白になること、複数ハンクで番号が連続すること。
  - `TestColorize_AddedLine` / `TestColorize_RemovedLine` / `TestColorize_HunkHeader` / `TestColorize_NoOpOnContext` / `TestColorize_PreservesNewlines`.
- Integration tests:
  - `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff`: 新フォーマット文字列が含まれること、`bytes.Buffer`（非 TTY）では ANSI が含まれないこと。
- Regression tests:
  - `TestUnifiedDiff_OrderStability`: 同一入力で同一出力（決定論性）を維持。
- Edge cases:
  - 空ファイル → 非空ファイル（旧カラム全て空白）。
  - 非空ファイル → 空ファイル（新カラム全て空白）。
  - 末尾改行差異（`\ No newline at end of file` の表示は維持）。
  - 4桁を超える行番号（フォーマット崩れの確認）。
- Evidence to capture: 上記テスト出力、`docs/reports/test-...md`。

## Risks and mitigations

- **下流が `UnifiedDiff` の旧フォーマットをパース**: リスク低（`docs/reports/...` で「display artifact, not parsed」と明記）。`grep -r "@@ -" .` で念のため確認。
- **Windows ターミナルの ANSI サポート**: PowerShell/Windows Terminal は標準対応。CMD.exe は古い環境で崩れる可能性 → `NO_COLOR=1` で回避可能と README で案内。
- **行番号カラム幅**: 大きいファイルで行番号 5 桁になるとフォーマット崩れ → `%4d` ではなく動的幅算出。

## Rollout or rollback notes

- 単一 PR で配布。フィーチャーフラグ無し（小規模 UX 改善）。
- ロールバックはコミットを revert するだけで完結。

## Open questions

- 色のテーマ（git 標準配色）でユーザー合意済み。pastel への変更要望が出たらフォローアップ。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (`feat/colorize-upgrade-diff`)
- [x] Implementation started
- [x] `internal/upgrade/unified_diff.go` 行番号付与＋ハンクヘッダ刷新
- [x] `internal/upgrade/colorize.go` ANSI ヘルパ追加（純粋関数）
- [x] `internal/cli/upgrade.go` TTY/NO_COLOR 判定 + colorize 配線
- [x] テスト更新（`unified_diff_test.go`, `colorize_test.go`, `cli_test.go`）
- [x] `./scripts/run-verify.sh` グリーン (evidence: `docs/evidence/verify-2026-04-23-040315.log`)
- [x] 視覚確認: 実バイナリで scaffold → upgrade → `[d]iff` を実行し新フォーマット出力を確認
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
