# Cross-review triage report: overlay-scaffold-v2-p4 (Phase 4)

- Date: 2026-08-18
- Plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md
- Self-review report: docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md
- Verify report: docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md
- Implementation context summary: 3 件とも self-review HIGH-1 の修正(b1babe7)で導入した `OpDeleteOldPathAdoptFork` 系と、移行の manifest 直構築が v2 エンジンの分類を迂回する箇所の取りこぼし。(1) は新 op kind の検証対象漏れ(プラン AC-16 の「移設元/先」契約に対し先の検証が欠落)、(2) は Phase 4 スライス 1 で直した Known gap(seed 衝突 advisory)が移行経路では manifest 直記録により再び隠れるという盲点、(3) はレポートの fork diff 対象に新 kind を足し忘れた単純漏れ。いずれも修正は局所的。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `OpDeleteOldPathAdoptFork` の検証が OldPath のみで、NewPath が symlink / symlink 親でも旧ファイルを削除し fork を不安全な移設先に記録する | 実問題。プラン AC-16(移設元/先の検証)の契約違反。validateMigrationOp で NewPath にも同一検証(ValidateRealParentChain + 葉 Lstat)を適用 + symlink テスト | internal/cli/migrate.go:995-996 |
| 2 | [P2] レガシー移行の汎用 desired-sweep が manifest 未登録の seed 衝突パスに TemplateHash=現行を即記録し、連鎖 upgrade から見ると「追跡済み seed」になって AC-1 の advisory が両レポートから消える | 実問題。スライス 1 の Known gap 修正が移行経路で迂回される盲点。buildMigratedManifest の汎用 sweep で「未登録 + ディスク既存 + 内容相違の seed」は TemplateHash を旧値なし(または disk 実態)で記録して advisory が連鎖側で発火するようにする — 実装は連鎖側分類と整合する形を実装者が選択 | internal/cli/migrate.go:1479 |
| 3 | [P2] 移行レポートの fork diff 対象が `OpForkRelocate`/`OpForkInPlace` のみで、`OpDeleteOldPathAdoptFork`(採用 fork)の diff が欠落 — AC-8 違反 | 実問題。forkEntries に新 kind を含める単純修正 + レポート内容テスト | internal/cli/migrate.go:1521-1522 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
