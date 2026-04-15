# Slice: panes — Pane component implementations

- Slice number: 4
- Parent plan: docs/plans/active/2026-04-15-ralph-tui/_manifest.md
- Status: Complete

## Objective

5つのペインコンポーネント（スライス一覧、詳細、依存関係、ログ、プログレスバー）を実装する。各コンポーネントは Bubble Tea の sub-model パターンで独立した Init/Update/View を持つ。

## Acceptance criteria

- [x] スライス一覧ペインで j/k による上下移動が動作すること
- [x] スライス一覧にステータスアイコン (+ * - ! ?) と色分けが表示されること
- [x] `/` でスライス名のフィルタリングが動作すること
- [x] 詳細ペインに選択スライスのステータス・フェーズ・サイクル・経過時間・テスト結果が表示されること
- [x] 依存関係ペインに ASCII ツリーが表示され、完了スライスが色分けされること
- [x] ログペインが bubbles/viewport を使用し、j/k でスクロール可能であること
- [x] プログレスバーに完了率・ETA・完了数/総数が表示されること
- [x] テストカバレッジが 80% 以上であること

## Affected files

- `internal/ui/panes/slicelist.go` (新規)
- `internal/ui/panes/detail.go` (新規)
- `internal/ui/panes/deps.go` (新規)
- `internal/ui/panes/logview.go` (新規)
- `internal/ui/panes/progress.go` (新規)
- `internal/ui/panes/slicelist_test.go` (新規)
- `internal/ui/panes/detail_test.go` (新規)
- `internal/ui/panes/deps_test.go` (新規)
- `internal/ui/panes/logview_test.go` (新規)
- `internal/ui/panes/progress_test.go` (新規)

## Dependencies

slice-3

## Implementation outline

1. `internal/ui/panes/slicelist.go`:
   - `type SliceListModel struct` — スライス一覧 (bubbles/list ベース)
   - j/k で移動、選択変更時に `SliceSelectedMsg` を送信
   - ステータスアイコン: complete→`+`(green), running→`*`(cyan), pending→`-`(dim), failed→`!`(red), unknown→`?`(dim)
   - `/` でフィルタモード切替、Enter で確定、Esc でキャンセル
2. `internal/ui/panes/detail.go`:
   - `type DetailModel struct` — 選択スライスの詳細表示
   - フィールド: Status, Phase (色付き), Cycle (X/Y), Elapsed, Test Result, PR URL
   - `SliceSelectedMsg` を受けて表示内容を更新
3. `internal/ui/panes/deps.go`:
   - `type DepsModel struct` — 依存関係 ASCII ツリー
   - ツリーレンダリング: 各スライスに `├──` / `└──` のコネクタ
   - 完了スライスは green、running は cyan、pending/failed は dim/red
   - 選択スライスをハイライト
4. `internal/ui/panes/logview.go`:
   - `type LogViewModel struct` — ログビューポート (bubbles/viewport ベース)
   - `LogLineMsg` を受けてコンテンツに追加
   - auto-scroll: 新しい行が追加されるとビューポートを末尾にスクロール
   - ANSI エスケープシーケンスをフィルタリング
   - フォーカス時は j/k でスクロール可能
5. `internal/ui/panes/progress.go`:
   - `type ProgressModel struct` — プログレスバー
   - 表示: `[####.....] 40% (2/5)  ETA: 8m`
   - ETA 計算: 完了スライスの平均所要時間 × 残りスライス数

## Verify plan

- Static analysis checks: `go vet ./internal/ui/panes/...`
- Spec compliance criteria to confirm:
  - ステータスアイコンが仕様と一致 (ralph-status-helpers.sh 162-171行のアイコン定義と同一)
  - 依存関係ツリーがプランファイルの dependency graph と対応
- Evidence to capture: `go vet` 出力

## Test plan

- Unit tests: `go test ./internal/ui/panes/...`
  - スライス一覧: j/k 移動、フィルタ、アイコン表示
  - 詳細: 各フィールドの表示
  - 依存関係: ツリーレンダリング (線形、分岐、循環なし)
  - ログビュー: 行追加、自動スクロール、ANSI フィルタ
  - プログレスバー: パーセンテージ計算、ETA 計算
- Edge cases: 0 スライス、1 スライス、依存なし、ログ空、長いスライス名
- Evidence to capture: `go test -cover` レポート

## Notes

- `ralph-status-helpers.sh` のアイコン・色定義を Go 側でも同じ対応にする
- ETA 計算ロジックは `ralph-status-helpers.sh` 301-314行の既存実装を参考にする
