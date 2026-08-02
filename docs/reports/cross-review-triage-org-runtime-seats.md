# Cross-review triage report: org-runtime-seats

- Date: 2026-08-02
- Plan: docs/plans/active/2026-08-02-org-runtime-seats.md
- Base branch: main (merge-base ffda48f)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=1

## Triage context

- Active plan: docs/plans/active/2026-08-02-org-runtime-seats.md
- Self-review report: docs/reports/self-review-2026-08-02-org-runtime-seats.md
- Verify report: docs/reports/verify-2026-08-02-org-runtime-seats.md
- Implementation context summary: 座席 ID は `ValidateIdentifier`(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)で検証済みだが、ハイフンを許可しているため `<org_id>-<seat_id>` のハイフン連結は一意でない。herdr エージェント名(spawn.go `herdrAgentName`)とプロンプトファイル名(`promptFilePath`)の両方が同じ連結規則を使う(= 同一根本原因)。今日の org/seat ID はオペレータ入力だが、PR③ で Lead LLM 生成になるため衝突面は広がる。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `<org_id>-<seat_id>` のハイフン連結が一意でない(`a-b`+`c` と `a`+`b-c` が同じ `a-b-c` に衝突)。herdr エージェント名の衝突で send/wait が別 org の座席を操作し得る(所見1)。同じ連結規則のプロンプトファイルパスも衝突し、後発 spawn が先行座席の読む役割指示を上書きし得る(所見2)。 | 2 所見は同一根本原因のため 1 件に統合。ID charset がハイフンを許すので理論的に真正。名前空間分離は本 PR の中核保証(self-review HIGH で herdr 名前空間を導入した経緯)であり、修正は「連結セパレータを ID charset 外の文字(`.`)に変更+テスト+実機で herdr が `.` 入り名を受理することの確認」で安価。PR③ で ID が LLM 生成になる前に塞ぐべき。 | internal/org/spawn.go(herdrAgentName / promptFilePath)、関連テスト |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
| 2 | [P2] プロンプトファイル名の衝突(所見2 単体として) | 所見1 と同一根本原因・同一修正のため ACTION_REQUIRED #1 に統合(独立の欠陥としては重複) | already-addressed |

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
