# Cross-review triage report: org-runtime-mechanism

- Date: 2026-08-01
- Plan: docs/plans/active/2026-08-01-org-runtime-mechanism.md
- Base branch: main (merge-base 9abaaed)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-01-org-runtime-mechanism.md
- Self-review report: docs/reports/self-review-2026-08-01-org-runtime-mechanism.md
- Verify report: docs/reports/verify-2026-08-01-org-runtime-mechanism.md
- Implementation context summary: PR①機構層。spawn saga は「エンベロープ検証 → dry-run 分岐 → 冪等/stale 分岐 → 副作用」の順で実装。self-review の HIGH(herdr 名前空間)と in-PR MEDIUM 2件は 9bfe07e で修正済み。max_seats の read-then-append 競合は self-review 時点で認識され docs/tech-debt/README.md に登録済み。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 冪等チェックより前に max_seats 検証が走るため、org が cap 到達時に既存 active 座席の spawn リトライが冪等成功ではなく `rejected` になる(例: `max_seats=1` で唯一の座席を再 spawn → rejected イベント記録) | コード確認済み(spawn.go: `ValidateSpawn` が L128、冪等分岐が L148)。AC-3 の文書化された冪等保証が at-cap 境界で破れる真正の正当性バグ。修正は分岐順序の入替+境界テスト追加で安価。 | internal/org/spawn.go:126-150, internal/org/spawn_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] 同一 org_id への並行 spawn で read→validate→append が直列化されておらず、max_seats 超過や同一 seat_id の二重 saga が理論上可能 | 真正の TOCTOU だが、self-review 時点で認識済みかつ docs/tech-debt/README.md に登録済みの意図的繰延。PR① の利用形態(単一オペレータ/単一 Lead の逐次 CLI 呼び出し)では実発火が考えにくく、恒久対応(flock 等のプロセス間直列化)は Lead が高速に並行 spawn する PR③ までに実装するのが費用対効果に合う。 | internal/org/spawn.go:120-128 |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
