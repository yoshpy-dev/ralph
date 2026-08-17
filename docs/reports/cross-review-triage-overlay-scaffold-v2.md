# Cross-review triage report: overlay-scaffold-v2 (Phase 1)

- Date: 2026-08-17
- Plan: docs/plans/active/2026-08-17-overlay-scaffold-v2.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-17-overlay-scaffold-v2.md
- Self-review report: docs/reports/self-review-2026-08-17-overlay-scaffold-v2.md
- Verify report: docs/reports/verify-2026-08-17-overlay-scaffold-v2.md
- Implementation context summary: Phase 1 は CLI 未配線のエンジンプリミティブ。指摘 3 件はいずれも Phase 3 が配線時に継承する API 契約に関わるため、今修正する方が安い。self-review の LOW 所見(report 日付の未サニタイズ)と指摘 3 が重複しており、レビュー間で相互裏付けあり。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] テンプレートから削除された core パスがディスクにも既に存在しない場合、planner が何も emit せず、stale な manifest エントリを掃除する信号が呼び出し側に渡らない | 実問題。Phase 3 の manifest 再構築戦略が未確定のため、planner 側で削除信号(manifest 除去リスト)を出す方が戦略非依存で完全。API 追加は未配線の今が最も安い | internal/upgrade/replaceplan.go:253-254 |
| 2 | [P2] `ApplyOps` が op のパスを再検証せず `targetDir` に join するため、手組みの `ReplacePlan` で target 外への書き込み/削除が可能 | 実問題(defense-in-depth)。スペックの Security considerations とプラン AC-9 の趣旨(書き込みプリミティブの自己検証)に合致。`cleanPathKey` の再実行 1 箇所で塞がる | internal/upgrade/replaceplan.go:356 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 3 | [P2] `UpgradeReportRelPath` が `date` をサニタイズせず、`x/../../../AGENTS` のような値で docs/reports 外のパスを返し得る | 呼び出し元は Phase 3 の内部コード(自前生成の日付)で悪用経路は薄いが、self-review LOW 所見とも一致し、version と同じサニタイズ + 最終 prefix 検証で 2 行程度。実害は Debatable、修正価値は高い | internal/upgrade/report.go:195 |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
