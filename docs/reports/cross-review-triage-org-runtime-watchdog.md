# Cross-review triage report: org-runtime-watchdog

- Date: 2026-08-03
- Plan: docs/plans/active/2026-08-02-org-runtime-watchdog.md
- Base branch: main (merge-base e7a32b9)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] デッドマンの「lead 活動」判定が org 非スコープ: 共有 state-dir 上の別 org のイベントでも pending alert が解消され、停滞 org のエスカレーションを取り逃がす | 真正。verify が AC-5 の narrowed fix として指摘し tech-debt 化した根本原因を、Codex が具体的な取り逃がしシナリオ付きで独立指摘。修正は活動カウントを alert の org_id でフィルタするだけで安価。tech-debt row はこの修正でクローズする。 | internal/org/watch.go |
| 2 | [P2] 全席停止済みの org に対する watch(再)起動で total-budget の誤 ALERT+pending deadman が発生 | 真正。active 座席ゼロなら遮断対象がなく ALERT は誤報。early return で安価に修正可能。 | internal/org/watch.go |

## Decision

Cycle 1/2(cap 未到達)。ユーザーの継続自律指示に基づき Fix → フルパイプライン再実行(cycle 2)を選択。

---

# Cycle 2 (2026-08-03)

- Cycle: 2/2 (cap 到達)
- HEAD: d6ddf61
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

| # | Reviewer finding | Triage rationale |
|---|---|---|
| 3 | [P1] デッドマンの第 3 活動源(agmsg history)が lead 発信以外の全トラフィック(watchdog 自身の後続 ALERT 含む)で pending を解消し、エスカレーションを取り逃がす | 真正かつ最重要(デッドマンは人間への最後の安全網)。manifest 側で 2 回修正した同型バグの history 版。修正は history 行を lead 発信のみにフィルタするだけで安価。 |
| 4 | [P2] total-budget 遮断直後の同一サイクルで stale スナップショットの座席を再評価し、遮断済み座席へ誤 ALERT/deadman 記録 | 真正(誤報ノイズ)。遮断後は当該サイクルの後続評価から除外する早期 continue で安価に修正可能。 |

## Decision(cap-reached Option 1)

安全網(デッドマン)の取り逃がしという P1 の性質から、ユーザーの継続自律指示のもと **cap を一時的に 3 に引き上げて修正**を選択(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` 相当。post-implementation-pipeline.md の cap-reached Option 1)。修正後にフルパイプライン cycle 3 を実施する。
