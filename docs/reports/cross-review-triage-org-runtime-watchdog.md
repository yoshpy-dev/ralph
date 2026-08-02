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

---

# Cycle 3 (2026-08-03)

- Cycle: 3/3(引き上げ後 cap 到達)
- HEAD: 1b48b04
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=2, DISMISSED=0

cycle-2 の #3(P1 デッドマン取り逃がし)/#4 は修正済みで再指摘なし。新規 2 件はいずれも狭いエッジケースの堅牢化(P2)であり、コア経路(遮断・ALERT 配送・デッドマン・watcher)は実機スモークとテスト 406 件で証明済み。

| # | Reviewer finding | Triage rationale |
|---|---|---|
| 5 | [P2] プローブ不可時(`LeadAgentGet==""` / `HistoryLeadLines==-1`)に記録された alert が、プローブ復旧だけで lead 無応答でも解消される | 真正だが「プローブ全滅中に alert 発生 → 復旧」の限定窓。可用性状態の明示保持で対応可能。WORTH_CONSIDERING → tech-debt 登録し PR body Known gaps に記載。 |
| 6 | [P2] 初回 ALERT 時に `Agmsg.Join` が一時失敗すると `WatchdogJoined` が true で永続化され、以後 rejoin されず配送が失敗し続ける | 真正だが transient-join 限定。join 成功時のみフラグ設定の 1 行修正で対応可能。同上。 |

## Decision(cap-reached Option 2)

引き上げ後の cap(3)にも到達。サイクルごとに新規 P2 が発見される逓減局面であり、両件とも限定的エッジケースのため「記録して PR」を選択。tech-debt 2 行を追加し、PR body の Known gaps に記載する。
