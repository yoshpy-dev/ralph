# Cross-review triage report: overlay-scaffold-v2-p2 (Phase 2)

- Date: 2026-08-17
- Plan: docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md
- Self-review report: docs/reports/self-review-2026-08-17-overlay-scaffold-v2-p2.md
- Verify report: docs/reports/verify-2026-08-17-overlay-scaffold-v2-p2.md
- Implementation context summary: 2 件とも manifest v3 の所有メタデータが Phase 3 の置換プランナの前提と食い違う箇所で、Phase 2 が所有記録の唯一の生成者である以上、今修正しないと Phase 3 が誤った入力を継承する。self-review / verify は所有マップの「値」までは検証していなかった(AC-2 のスポットチェックは AGENTS.md/.gitignore/CLAUDE.md 等が対象で、`.ralph/local/**` と pack add 経路は未カバー)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `ownerForScaffoldPath` の catch-all が `.ralph/local/**`(.gitkeep 骨格)を owner=core として記録する。スペックの層モデルでは `.ralph/local/` は L3 overlay(不可侵)であり、Phase 3 プランナが core として全置換対象に含めると、ユーザ drop-in 領域にテンプレート操作が計画され得る | 実問題。所有分類テーブルの漏れで、修正は 1 分岐追加(`.ralph/local/` → seed: 欠落時のみ生成・以後不可侵、advisory 対象は .gitkeep のみで無害)+ テスト。スペック L3 との整合を回復する | internal/cli/init.go:296-299 |
| 2 | [P2] `ralph pack add`(pack.go)が v2 manifest に対して owner なしのエントリ(pack payload + `.claude/rules/ralph/<lang>.md`)を書き、プランナに legacy-skipped 扱いされる。AC-6 の「pack rule は owner=core で追跡」に反する | 実問題。init 経路のみ owner を付け、pack add 経路が漏れた。layout=v2 のとき追加パスに SetOwner(core) を呼ぶ + テストで塞がる | internal/cli/pack.go:55 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
