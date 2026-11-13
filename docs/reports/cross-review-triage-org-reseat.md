# Cross-review triage report: org-reseat

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-12-org-reseat.md
- Base branch: main (merge base 4220ea7c)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-org-reseat.md(承認 digest f7e78c0dd96c)
- Self-review report: docs/reports/self-review-2026-11-12-org-reseat.md(2 回目のパイプライン)
- Verify report: docs/reports/verify-2026-11-12-org-reseat.md(pass)
- Test report: docs/reports/test-2026-11-12-org-reseat.md(pass)
- Implementation context summary: 1 回目の cross-review の ACTION_REQUIRED 3 件(adc3e205、e9461b79、0468d96f)を直したあとの 2 回目。仕分けの線引き(人の方針): 今回の変更が持ち込んだ、知らせや案内が働かなくなる向きの穴は直す。前からある問題は tech-debt と Known gaps に回す。
- 判断の前に consult(consult-plan-idempotent)に聞いた。判定は「ACTION_REQUIRED で直し、cap を 3 に上げて回し直す」。idempotent が返るのは座席の最新の state event が `spawned` のときだけ(`internal/org/spawn.go` の roster の判定)で、reseat が止めた座席の最新は `stopped`、もともと active でない座席の最新も `spawned` ではないので、立てる時点の idempotent は別の spawn が割り込んだときに限る。

codex の要約(原文): "The restart path can silently accept a concurrently spawned seat with different settings and falsely report the requested switch as successful."

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] reseat が座席を止めたあと立て直す前に、動いている leader などが同じ座席を spawn すると、`Spawn` は driver とモデルを比べずに `SpawnOutcomeIdempotent` を返す。`startAll` はそれを started と数えるので、`--driver codex --model ...` を頼んだのに Claude の座席が動いている場合も成功と報告し、出力と `reseated` の記録が計画の設定を載せて、切り替えの失敗を隠す。`reseat.lock` は普通の spawn を排他しない | 実際に起きる。窓は秒の単位で狭いが、起きると出力が事実と違い、人は切り替わったと思う。計画の Known gaps が挙げるのは読み直しと Stop のあいだの窓だけで、止めてから立てるまでの窓で違う設定の座席を成功と報告する穴は今回の変更が持ち込んだ。直し方: idempotent のとき、返った座席の Role・Driver・Model を計画の座席と比べ、違えば start_failed にして、いまの設定と頼んだ設定を名指しする理由で止める。座席は active なので、出す打ち直し(P7 で active な座席を受け付ける)で止めて頼んだ設定で立て直せる。一致すれば started のまま | internal/org/reseat.go、internal/org/reseat_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
