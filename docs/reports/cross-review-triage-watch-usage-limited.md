# Cross-review triage report: watch-usage-limited

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-13-watch-usage-limited.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-13-watch-usage-limited.md
- Self-review report: docs/reports/self-review-2026-11-13-watch-usage-limited.md(Merge: yes、未解決の LOW は L5・N1・N2)
- Verify report: docs/reports/verify-2026-11-13-watch-usage-limited.md(pass、AC9 の V1・V2 は sync-docs で直した)
- Implementation context summary: usage_limited の件に座席が増えたとき、件が open なら recurred として記録し、acked か resolved なら新しい件にする(計画の AC4、agent-messaging.md の規則)。open かどうかは `recordUsageLimitRecurrence` が inbox を読んだ時点で決めている。codex は read-only の sandbox で Go のテストを走らせられなかったと書いている(所見はコードを読んだもの)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 座席が増えたときの open の確かめが inbox のロックの外にある。`Inbox.Read` で open を見てから `AppendRecurred` までのあいだに director が ack すると、`AppendRecurred` は acked の件にも書くので、増えた座席が既読の件の recurred になる。そのあとの周期はその座席を見たものとして扱うので、新しい件ができない | 本物の穴で、この変更が持ち込んだ黙る向きのもの。窓はミリ秒の幅だが、起きると director は増えた座席を知らないまま件を resolve しうる。`AppendRecurred` に「open のときだけ」をロックの中で確かめる形を足し、acked で断られたら新しい件に回すだけで直せる。費用が小さいので直す | internal/org/watch_usage_limit.go:253-259、internal/org/inbox.go:398-420 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] recurred の note は 500 文字までで、増えた座席が多いと(例: 28 文字の id が 20 席)`+N more:` の id の並びまで `…` で切られ、いくつかの id が件に残らない。最初の件と違い、recurred には完全な `SEATS` の見出しがない | 切り詰めは本当にあるが、届くのは既定の `max_seats = 5` を大きく超える設定のときだけ(機能の org は 3 席)。`+N more` の数は残り、watch の stderr の行(`seats %s stopped on a usage limit; recorded as a recurrence ...`)は切らずに全部の座席を書く。状態ファイルの `usage_limited_seats` も全部の id を持つ。直すには recurred を分けるか証拠のポインタを足す設計が要るので、今回は tech-debt と PR の Known gaps に残す | internal/org/watch_usage_limit.go:276-278、:336-343 |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Decision

ACTION_REQUIRED の 1 件は、ユーザーの常の指示(cross-review の ACTION_REQUIRED 以上は聞かずに直す)に従って直し、パイプラインを `/self-review` から回し直す(cycle 2)。WORTH_CONSIDERING の 2 は、`docs/tech-debt/README.md` に行を足し、PR 本文の Known gaps に書く。
