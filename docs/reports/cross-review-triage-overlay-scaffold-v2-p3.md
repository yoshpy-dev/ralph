# Cross-review triage report: overlay-scaffold-v2-p3 (Phase 3)

- Date: 2026-08-18(cycle 4 で最終更新)
- Plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 4/4 (final — cap は 2→3→4 と引き上げ。cycle-4 の残 1 件は操作者判断で Known gap 化)
- Total reviewer findings: 1(cycle 4。cycle 1〜3 の計 5 件は全て解消・再指摘なし)
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Self-review report: docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md
- Verify report: docs/reports/verify-2026-08-18-overlay-scaffold-v2-p3.md
- Implementation context summary: Phase 2〜3 で積み上げてきた symlink 封じ込めガード(init 面 → ApplyOps 葉ノード → renderMappedFile)の残穴 2 箇所。(1) は ApplyOps の Lstat が葉のみ検査で、親ディレクトリが symlink の場合に素通りする — Phase 1 由来の設計穴で cycle-1 の Lstat 追加でも塞がっていなかった。(2) は settings/スナップショットの例外面書き込みが ApplyOps を経由しないため、ガードの適用外だった。どちらも非対話エンジンとして出荷前に塞ぐべき実穴で、スペックの Security considerations(settings マージのパストラバーサル防御)にも直結する。

## Cycle 1 findings resolution

cycle 1 の P1 2 件(ApplyOps 親チェーン / 例外面書き込みガード)は fc8e9a9 で解消(report 書き込みの同型穴も含む)。verify/test の Cycle 2 節が契約一致・変異テストを確認済み。

## Cycle 2 findings resolution

cycle 2 の AR#1(dry-run 例外面予告)・AR#2(no-op 再実行の書き汚し)は 41ad745 で解消、その修正が導入した C3-1(seed advisory 隠蔽)/C3-2(version 凍結)退行は 6533227 で解消。verify/test の Cycle 3 節が確認済み。

## Cycle 3 findings resolution

cycle 3 の AR#1(`.codex/AGENTS.override.md` の L3 分類漏れ)は b3c6307 で解消(所有マップ seed 分岐 + e2e 2 本)。verify/test の Cycle 4 節が確認済み。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 新テンプレートリリースが seed 所有パスを新規追加し、下流に同パスの manifest 未登録ローカルファイルが既存する場合、untracked 分類器が owner 非考慮で「未解決 drift」(恒久 exit 3)に分類する — L5 契約(欠落時のみ生成・advisory)違反 | 実在する将来リスクだが、発火にはリリース(Phase 5 後)+ 新 seed パス + 下流の同パス既存ファイルの 3 条件が必要で、現時点の露出ゼロ・挙動は非破壊(据え置き + exit 3)。owner 帰属の分類ロジックは Phase 4 移行分類器の本籍地であり、そこで untracked パスの owner 考慮分類として対処する。操作者判断で Known gap 化(tech-debt 行 + PR Known gaps 記録) | internal/cli/upgrade_v2.go:90(untracked 分類、根本は PlanCoreReplaceDesired の owner 非考慮 untracked 経路) |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
