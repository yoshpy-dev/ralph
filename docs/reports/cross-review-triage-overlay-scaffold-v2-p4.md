# Cross-review triage report: overlay-scaffold-v2-p4 (Phase 4)

- Date: 2026-08-18
- Plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md
- Self-review report: docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md (Cycle 3: pass, C3-1..C3-7 は 9ac24de で修正済み)
- Verify report: docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md (Cycle 3: pass, seed 全置換の FR-7 準拠判定を含む)
- Cycle-2 の AR#1〜AR#3 は 215c676 で修正済みで、cycle-3 の verify/test が契約一致と回帰テスト green を確認済み。本サイクルの所見は 1 件のみで、cycle-2 所見の再発ではなく、移設の衝突ケース (a) のハッシュ引き継ぎに関する新規エッジ。triager がコードを直接読んでメカニズムを確認した。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 未改変ルールの移設先が既に旧内容で存在する場合(衝突ケース (a)、手動コピーまたは中断移行の再実行)、relocationOutcome の `destHash == sourceHash` 分岐が OpDeleteOldPath を返し、buildMigratedManifest は移設先を新テンプレートハッシュで記録する。テンプレートがその間に進化していると、連鎖 v2 upgrade はディスクの旧内容を「未解決 drift」(FR-4)として据え置き、未改変ファイルが自動収束しない | 実問題(メカニズム確認済み: migrate.go:732-736 の settled 分岐+generic sweep の楽観的ハッシュ記録)。ただし (1) データ損失なし・無言でもない — drift は upgrade レポートと exit 3 で可視化され、FR-4 の正規手順 `ralph adopt <path>` 一発で新テンプレートに収束する、(2) 到達条件が狭い(移設先に旧内容が既存+テンプレート進化の重なり)、(3) 修正には settled ケースの識別とハッシュ引き継ぎの分岐(ケース (b) との区別)が必要で、最終 cap 到達サイクルでの追加 churn が相応にある。Real=yes / Worth fixing=debatable → WORTH_CONSIDERING | internal/cli/migrate.go:732-736, buildMigratedManifest generic sweep |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cap note

cap は cycle-2 後に操作者承認で 2→3 に引き上げ済みで、本サイクル(3/3)で再到達。WORTH_CONSIDERING 1 件の扱い(cap 再引き上げ / Known gap 化して PR / 中止)は操作者判断に委ねる。

## Cycle-2 record (superseded)

Cycle-2 のトリアージ(AR#1: 未改変 seed の FR-7 置換欠落、AR#2: unavailable pack の保全欠落、AR#3: 移設 OpDeleteOldPath の NewPath 未検証 — ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0)は 215c676 で修正され、cycle-3 の self-review(cd125b7)・verify(5b88bae)・test(20cc352)で契約一致・回帰テスト green を確認済み。self-review c3 の HIGH(C3-1: unavailable pack のレガシー rule パス削除)は 9ac24de で修正済み。

## Cycle-1 record (superseded)

Cycle-1 のトリアージ(AR#1: OpDeleteOldPathAdoptFork の NewPath 検証欠落、AR#2: 移行経路での seed 衝突 advisory 消失、AR#3: 採用 fork の diff レポート欠落 — いずれも ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0)は c51497e で修正され、cycle-2 の verify(0cc50e5)と test(d686226)で契約一致・回帰テスト green を確認済み。
