# Cross-review triage report: overlay-scaffold-v2-p3 (Phase 3)

- Date: 2026-08-18(cycle 2 で更新)
- Plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached — cap は 2→3 へ引き上げ承認済み)
- Total reviewer findings: 2(cycle 2。cycle 1 の P1 2 件は fc8e9a9 で解消・再指摘なし)
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Self-review report: docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md
- Verify report: docs/reports/verify-2026-08-18-overlay-scaffold-v2-p3.md
- Implementation context summary: Phase 2〜3 で積み上げてきた symlink 封じ込めガード(init 面 → ApplyOps 葉ノード → renderMappedFile)の残穴 2 箇所。(1) は ApplyOps の Lstat が葉のみ検査で、親ディレクトリが symlink の場合に素通りする — Phase 1 由来の設計穴で cycle-1 の Lstat 追加でも塞がっていなかった。(2) は settings/スナップショットの例外面書き込みが ApplyOps を経由しないため、ガードの適用外だった。どちらも非対話エンジンとして出荷前に塞ぐべき実穴で、スペックの Security considerations(settings マージのパストラバーサル防御)にも直結する。

## Cycle 1 findings resolution

cycle 1 の P1 2 件(ApplyOps 親チェーン / 例外面書き込みガード)は fc8e9a9 で解消(report 書き込みの同型穴も含む)。verify/test の Cycle 2 節が契約一致・変異テストを確認済み。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `--dry-run` が block/settings のみ変化するケースで例外面の更新を予告せず、「変更なし」表示のまま実行時に AGENTS.md/.gitignore/settings を書き換える | 実問題(スペックの dry-run 契約: 置換 + block 更新 + advisory の予告)。dry-run パスで block/settings のマージ結果を純計算(書き込みなし)してプレビューに含める修正 | internal/cli/upgrade_v2.go:95-96 |
| 2 | [P2] 収束後の同一バージョン再実行が日付付きレポートと manifest を書き、リポジトリを汚す(NFR-1 の「no-op 再実行 = 書き込みゼロ」違反) | 実問題。プランと例外面マージが全て無変化のとき、レポート/manifest 書き込みをスキップし stdout に no-op を明記する(スペックの「レポートに no-op 明記」との矛盾は stdout 通知と解釈して解消し、スペックに追記) | internal/cli/upgrade_v2.go:172-177 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
