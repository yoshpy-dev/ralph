# Cross-review triage report: overlay-scaffold-v2-p3 (Phase 3)

- Date: 2026-08-18(cycle 3 で更新)
- Plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached — cap は 2→3→4 と引き上げ承認済み)
- Total reviewer findings: 1(cycle 3。cycle 1 の 2 件は fc8e9a9、cycle 2 の 2 件は 41ad745 + 6533227 で解消・再指摘なし)
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Self-review report: docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md
- Verify report: docs/reports/verify-2026-08-18-overlay-scaffold-v2-p3.md
- Implementation context summary: Phase 2〜3 で積み上げてきた symlink 封じ込めガード(init 面 → ApplyOps 葉ノード → renderMappedFile)の残穴 2 箇所。(1) は ApplyOps の Lstat が葉のみ検査で、親ディレクトリが symlink の場合に素通りする — Phase 1 由来の設計穴で cycle-1 の Lstat 追加でも塞がっていなかった。(2) は settings/スナップショットの例外面書き込みが ApplyOps を経由しないため、ガードの適用外だった。どちらも非対話エンジンとして出荷前に塞ぐべき実穴で、スペックの Security considerations(settings マージのパストラバーサル防御)にも直結する。

## Cycle 1 findings resolution

cycle 1 の P1 2 件(ApplyOps 親チェーン / 例外面書き込みガード)は fc8e9a9 で解消(report 書き込みの同型穴も含む)。verify/test の Cycle 2 節が契約一致・変異テストを確認済み。

## Cycle 2 findings resolution

cycle 2 の AR#1(dry-run 例外面予告)・AR#2(no-op 再実行の書き汚し)は 41ad745 で解消、その修正が導入した C3-1(seed advisory 隠蔽)/C3-2(version 凍結)退行は 6533227 で解消。verify/test の Cycle 3 節が確認済み。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `.codex/AGENTS.override.md` が desired state に含まれ core 分類されるため、下流のカスタマイズが未解決 drift(毎回 exit 3)になり、未編集コピーはテンプレート変更で置換される — スペック L3(不可侵)違反 | 実問題。Phase 2 cycle-1 の `.ralph/local` → seed と同型の所有マップ分類漏れ。`ownerForScaffoldPath` に `.codex/AGENTS.override.md` → seed の分岐を追加すれば、init の記録・upgrade の分類とも L3 意味論(欠落時のみ生成・以後不可侵・テンプレート変更は advisory)に揃う | internal/cli/upgrade_v2.go:355(根因は internal/cli/init.go ownerForScaffoldPath) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
