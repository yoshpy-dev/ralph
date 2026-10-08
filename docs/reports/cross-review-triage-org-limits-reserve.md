# Cross-review triage report: org-limits-reserve

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Base branch: main (51855166)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-08-org-limits-reserve.md(AC1〜AC15、S1〜S4、承認 digest 1a165903b5df)
- Self-review report: docs/reports/self-review-2026-10-08-org-limits-reserve.md(Merge 可。MEDIUM の F-1 と LOW の F-3 は 8dd19634 で直した)
- Verify report: docs/reports/verify-2026-10-08-org-limits-reserve.md(pass)
- Implementation context summary: 座席を立てるとき、台帳のロックの中で `max_seats` → `max_orgs` → `max_total_seats` → 予約の順に判定する。すでに立っている leader への spawn に `--reserve` を渡すと、`idempotentRespawn` が、既存の座席を返す前に予約を判定する(AC14)。review は HEAD 196205f8 に対して read-only の sandbox で動いた。テストは回していない(本人の申告)

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] すでに立っている leader への予約の経路は、座席が動いているかを確かめない。古い台帳で leader の `spawned` のあとに、`stopped` なしで `disbanded` が来ていると、Roster はその座席を `spawned` のまま Active でないと示す。この座席に `--reserve` で打ち直すと、上限を見ずに `scope_reserved` を書く。`max_orgs = 1` でほかの org が走っていても、2 つ目の org が走っている扱いになる | 本物の問題と判断した。`Spawn` の 2 か所(`internal/org/spawn.go` の Phase 1 と Phase 2)は、最後の状態が `spawned` なら Active を見ずに `idempotentRespawn` を呼ぶ。`idempotentRespawn` は予約を書くだけで、上限は判定しない。予約は「走っている org」の条件の 1 つなので、走っていなかった org が枠を取る。今の `Disband` はかならず先に座席を止める(`stopped` か `stop_failed`、`--force` でも `stopped`)ので、起きるのは古い ralph の台帳だけ。ただ、この PR が入れる上限をすり抜ける穴で、計画の範囲(AC1、AC14)の中にあり、直すのは数行で済む | internal/org/spawn.go(idempotentRespawn と、その 2 つの呼び出し元) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
