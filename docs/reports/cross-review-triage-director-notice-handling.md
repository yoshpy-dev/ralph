# Cross-review triage report: director-notice-handling

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-13-director-notice-handling.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-13-director-notice-handling.md
- Self-review report: docs/reports/self-review-2026-11-13-director-notice-handling.md(Merge: yes、未解決の LOW は L3 と N6)
- Verify report: docs/reports/verify-2026-11-13-director-notice-handling.md(pass)
- Test report: docs/reports/test-2026-11-13-director-notice-handling.md(pass)
- Implementation context summary: leader は `ralph org spawn` の hook の行を見たら QUESTION を上げ、作業を止めずに続ける(leader.md の「運用規律」、「件を上げる」の例外)。director の skill の「人に向けた知らせの行」は、その QUESTION を「ほかの QUESTION と同じく扱う」としていた。つまり人の答えを待ってから DECISION を送り、resolve する流れだった。計画は本体の Go を変えない(文と prompt のテストだけ)。codex は read-only の sandbox で Go のテストを起動できなかったと書いている(所見はコードを読んだもの)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 止めない hook の QUESTION が open か acked のまま残ると、見張りの `orgHasOpenLeaderItem` はそれを leader が答えを待っている印とみなし、leader の stall の ALERT を黙らせる(ほかの座席が動いていなくても)。見張りはこの QUESTION とほかの QUESTION を見分けられない | 本物の穴で、このブランチが持ち込んだもの。`orgHasOpenLeaderItem`(`internal/org/watch.go:1228`)は、org の QUESTION・BLOCKED・RESULT のうち resolve されていない件があれば真を返す。skill のとおり人の答えを待つと、件は人が答えるまで(数時間になりうる)開いたままで、その間 leader が本当に止まっても知らせが出ない。codex は機械が読める印を足す直し方を勧めたが、計画は Go を変えないので、skill の文で直す: director は行と問いを人に伝えたら、人の答えを待たずに暫定の DECISION を送って resolve する。開いている時間はほかの QUESTION と同じく director が処理するまでの間に縮む | .claude/skills/director/SKILL.md(4 写し、「人に向けた知らせの行」の hook の項)、internal/org/watch.go:1228-1242(読むだけ) |
| 2 | [P2] この QUESTION は作業を止めないので、人が答える前に leader が終わって disband しうる。ほかの QUESTION と同じく扱うと、director は resolve の前に DECISION を送る必要があるが、`Org.Disband` は leader の pane を閉じている。件は resolve できないまま残り、resolve の期限の知らせも出る | 本物。止めない QUESTION にしたことで、disband が人の答えより先に来る道ができた。skill には RESULT の件に「leader はそのあと disband するので答えは送らず、`--note` を付けて resolve する」という前例がある。1 の直しで件はすぐ resolve されるので窓は小さくなるが、director が件を処理する前に org が disband 済みのときの道も書く: DECISION は送らず、`--note` に人に伝えたことを書いて resolve する。人の答えが来たときに org が disband 済みなら、送る先がないことを人に伝える | .claude/skills/director/SKILL.md(4 写し、同じ項) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Decision

ACTION_REQUIRED の 2 件は、ユーザーの常の指示(cross-review の ACTION_REQUIRED 以上は聞かずに直す)に従って直し、パイプラインを `/self-review` から回し直す(cycle 2)。直しは skill の 4 写しの hook の項、そこを固める `internal/org/prompts_notice_test.go` の断片、multi-org-director の仕様の #318 の項の 1 句に限る。leader.md は変えない(「DECISION が届いたら、そのとおりにしてください」が暫定の DECISION と人の答えの DECISION の両方に当てはまる)。見張りのコードに止めない QUESTION の印を足す直し方は、計画の範囲(Go を変えない)の外なので採らない。
