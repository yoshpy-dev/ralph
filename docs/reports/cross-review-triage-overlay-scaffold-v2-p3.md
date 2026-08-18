# Cross-review triage report: overlay-scaffold-v2-p3 (Phase 3)

- Date: 2026-08-18
- Plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md
- Self-review report: docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md
- Verify report: docs/reports/verify-2026-08-18-overlay-scaffold-v2-p3.md
- Implementation context summary: Phase 2〜3 で積み上げてきた symlink 封じ込めガード(init 面 → ApplyOps 葉ノード → renderMappedFile)の残穴 2 箇所。(1) は ApplyOps の Lstat が葉のみ検査で、親ディレクトリが symlink の場合に素通りする — Phase 1 由来の設計穴で cycle-1 の Lstat 追加でも塞がっていなかった。(2) は settings/スナップショットの例外面書き込みが ApplyOps を経由しないため、ガードの適用外だった。どちらも非対話エンジンとして出荷前に塞ぐべき実穴で、スペックの Security considerations(settings マージのパストラバーサル防御)にも直結する。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] ApplyOps の Lstat プリフライトは葉ノードのみ検査で、親ディレクトリが symlink(例: `scripts -> /tmp/outside`)の場合、欠落子パスが ErrNotExist となり MkdirAll/WriteFile が symlink 越しに targetDir 外へ書く | 実穴。パス各構成要素の Lstat 検査、または書き込み直前に親チェーンを解決して targetDir 配下であることの検証が必要。symlinked-parent のテスト付きで塞ぐ | internal/upgrade/replaceplan.go:475-480 |
| 2 | [P1] `.claude/settings.json` / `.ralph/core/settings.ralph.json` の例外面書き込みが ApplyOps を経由せず os.WriteFile 直書きのため、symlink 標的で targetDir 外へ書ける | 実穴。block 面と同じ Lstat + 非 regular 拒否をこの 2 書き込みに適用する。スペック Security considerations の settings マージ防御要件に合致 | internal/cli/upgrade_v2.go:563-567 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
