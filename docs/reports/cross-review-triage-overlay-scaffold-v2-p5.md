# Cross-review triage report: overlay-scaffold-v2-p5 (Phase 5)

- Date: 2026-08-19
- Plan: docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md
- Self-review report: docs/reports/self-review-2026-08-19-overlay-scaffold-v2-p5.md(C1: HIGH 1 + MEDIUM 8 + LOW 4 → 7cf4f47 で全修正)
- Verify report: docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md(pass)
- 2 件とも `ralph doctor --strict`(FR-9)の fail-open。self-review M2(corrupt manifest の fail-open)と同型のクラスで、M2 修正が覆わなかった残り 2 経路。reviewer は実バイナリをビルドし、両シナリオで `doctor --strict` が exit 0 になることを実測している(このレポートの根拠として最も強い部類)。triager もコード経路を読んで確認した。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 追跡ファイルが読めない(例: `scripts/run-verify.sh` がディレクトリ化)と `resolveOwnershipPlan` がエラーになり、`checkScaffoldIntegrity` は warn を返すため `--strict` でも exit 0 — FR-9 ゲートが壊れた scaffold ほど素通しになる | 実問題(reviewer が実測: rc=0)。self-review M2 は manifest パースエラーを strict 違反化したが、ownership planning の実行時エラー(ディスク読み取り失敗)は warn のまま残った。M7 の status 側「劣化して続行」は正しい判断だが、doctor の integrity ゲートでは逆で、検査不能=違反として扱うべき。修正: planning エラーを strict-eligible の違反 result にする+回帰テスト | internal/cli/doctor.go:726 付近 |
| 2 | [P2] `.ralph/core/settings.ralph.json` が v2SkipPaths 経由で (e) manifest/実体整合から除外され、(c) も欠落スナップショットを `{}` として扱うため、manifest 追跡中のスナップショットを削除しても `--strict` が両検査 pass — manifest/disk 不一致の見逃し | 実問題(reviewer が実測: rc=0)。v2SkipPaths の除外理由は「(b)/(c) の担当面」だが、スナップショットはどちらの検査でも存在自体を担保されていない。修正: (e) の除外集合からスナップショットを外す(または存在チェックを (c) に追加)+削除シナリオの回帰テスト。upgrade 側の `{}` フォールバック(非破壊劣化)は仕様どおりで変更しない | internal/cli/doctor.go:954-956、v2SkipPaths との関係 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
