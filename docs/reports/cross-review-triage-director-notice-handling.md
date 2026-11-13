# Cross-review triage report: director-notice-handling

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-13-director-notice-handling.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-13-director-notice-handling.md
- Self-review report: docs/reports/self-review-2026-11-13-director-notice-handling.md(cycle 2 の再確認で Merge: yes。未解決の LOW は L3、N6、C2-7、C2-8)
- Verify report: docs/reports/verify-2026-11-13-director-notice-handling.md(cycle 2、pass。V2-1 は C2-7 と同じ穴)
- Test report: docs/reports/test-2026-11-13-director-notice-handling.md(cycle 2、pass)
- Cycle 1 の経過: ACTION_REQUIRED 2 件(止めない hook の QUESTION が開いたままだと見張りが leader の stall を黙らせる、人の答えの前に org が disband すると件が残る)を聞かずに直した。直しは 8ae5ae1c、7bcef73e、8e8d26b5(director は hook の QUESTION を DECISION を送らずにすぐ resolve する、人の答えは `HOOK_SEAT:` の付いた DECISION、leader は承認・cap・ほかの待ちでその DECISION を除く)。cycle 1 の版のこの報告は 105d75cb
- Cycle 2 の所見はどちらも、director のいない設定(`[org.director].enabled = false`、受信箱の件は人に直接届く)の話。self-review cycle 2 の C2-7 と verify cycle 2 の V2-1 が LOW として見つけ、sync-docs cycle 2 が `docs/tech-debt/README.md` に行を足していた(director のない設定の行)。codex は read-only の sandbox でテストを走らせていない(所見はコードを読んだもの)
- 番号: cycle 1 の指摘は下の `## Cycle 1` の節に 1 と 2 のまま残し、cycle 2 は 3 から振る

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 3 | [P2] director のいない設定では、hook の QUESTION を誰もすぐには resolve しない。`/org` skill の受信箱の手順は人が答えるまで件を開いたままにし、そのあいだ `orgHasOpenLeaderItem` が leader の stall を黙らせる。leader は作業を続けているので、本当に止まっても知らせが出ない | 本物で、このブランチが持ち込んだ道(止めない hook の QUESTION)。director のない設定は AGENTS.md と仕様が書く、支えている設定。直しは安い: leader が上げる QUESTION の本文に「leader は答えを待たずに作業を続けているので、この件は答えの前に resolve してよい」と書かせる。director のない設定ではこの QUESTION が人に直接届くので、人はそこで知る。`/org` skill はレーン B の #322 も直しているので変えない。見張りのコードに印を足す道は、計画の範囲(Go を変えない)の外 | internal/org/prompts/leader.md(「運用規律」の hook の項)、internal/org/watch.go:1228-1242(読むだけ) |
| 4 | [P2] director のいない設定では、人は `/org` skill の手順で普通の DECISION を送り、`HOOK_SEAT:` を付けない。承認の待ちや cap の待ちの leader に届くと、手順 4 は直しの依頼と読み、cap の読み手は QUESTION を上げ直す | 本物で、3 と同じ道。直しも同じ所: leader の QUESTION の本文に「答えは `HOOK_SEAT: <座席 id>` のヘッダを付けた DECISION で送ってほしい」と書かせる。director がいる設定では director の skill が同じことを書いているので、どちらの道でもヘッダが付く | internal/org/prompts/leader.md(同じ項)、.claude/skills/org/SKILL.md(読むだけ) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Decision

ACTION_REQUIRED の 2 件は、ユーザーの常の指示(cross-review の ACTION_REQUIRED 以上は聞かずに直し、上限に届いたら上げて回し直す)に従い、この task の上限を 3 に上げて直し、パイプラインを `/self-review` から回し直す(cycle 3)。直しは leader.md の hook の項(QUESTION の本文に、答えの前に resolve してよいことと `HOOK_SEAT:` のヘッダで答えてほしいことを書かせる)、そこを固める `internal/org/prompts_notice_test.go` の断片、director のない設定の tech-debt の行(払う)、multi-org-director の仕様の #318 の項の 1 句に限る。`/org` skill は変えない。

## Cycle 1

- Reviewer status: complete
- Cycle: 1/2
- Total reviewer findings: 2
- Cycle 1 counts: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context(cycle 1)

- Self-review report: docs/reports/self-review-2026-11-13-director-notice-handling.md(Merge: yes、未解決の LOW は L3 と N6)
- Verify report: docs/reports/verify-2026-11-13-director-notice-handling.md(pass)
- Test report: docs/reports/test-2026-11-13-director-notice-handling.md(pass)
- Implementation context summary: leader は `ralph org spawn` の hook の行を見たら QUESTION を上げ、作業を止めずに続ける(leader.md の「運用規律」、「件を上げる」の例外)。director の skill の「人に向けた知らせの行」は、その QUESTION を「ほかの QUESTION と同じく扱う」としていた。つまり人の答えを待ってから DECISION を送り、resolve する流れだった。計画は本体の Go を変えない(文と prompt のテストだけ)。codex は read-only の sandbox で Go のテストを起動できなかったと書いている(所見はコードを読んだもの)。

### ACTION_REQUIRED(cycle 1)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 止めない hook の QUESTION が open か acked のまま残ると、見張りの `orgHasOpenLeaderItem` はそれを leader が答えを待っている印とみなし、leader の stall の ALERT を黙らせる(ほかの座席が動いていなくても)。見張りはこの QUESTION とほかの QUESTION を見分けられない | 本物の穴で、このブランチが持ち込んだもの。`orgHasOpenLeaderItem`(`internal/org/watch.go:1228`)は、org の QUESTION・BLOCKED・RESULT のうち resolve されていない件があれば真を返す。skill のとおり人の答えを待つと、件は人が答えるまで(数時間になりうる)開いたままで、その間 leader が本当に止まっても知らせが出ない。codex は機械が読める印を足す直し方を勧めたが、計画は Go を変えないので、skill の文で直す: director は行と問いを人に伝えたら、人の答えを待たずに暫定の DECISION を送って resolve する。開いている時間はほかの QUESTION と同じく director が処理するまでの間に縮む | .claude/skills/director/SKILL.md(4 写し、「人に向けた知らせの行」の hook の項)、internal/org/watch.go:1228-1242(読むだけ) |
| 2 | [P2] この QUESTION は作業を止めないので、人が答える前に leader が終わって disband しうる。ほかの QUESTION と同じく扱うと、director は resolve の前に DECISION を送る必要があるが、`Org.Disband` は leader の pane を閉じている。件は resolve できないまま残り、resolve の期限の知らせも出る | 本物。止めない QUESTION にしたことで、disband が人の答えより先に来る道ができた。skill には RESULT の件に「leader はそのあと disband するので答えは送らず、`--note` を付けて resolve する」という前例がある。1 の直しで件はすぐ resolve されるので窓は小さくなるが、director が件を処理する前に org が disband 済みのときの道も書く: DECISION は送らず、`--note` に人に伝えたことを書いて resolve する。人の答えが来たときに org が disband 済みなら、送る先がないことを人に伝える | .claude/skills/director/SKILL.md(4 写し、同じ項) |

### Decision(cycle 1)

ACTION_REQUIRED の 2 件は、ユーザーの常の指示(cross-review の ACTION_REQUIRED 以上は聞かずに直す)に従って直し、パイプラインを `/self-review` から回し直す(cycle 2)。直しは skill の 4 写しの hook の項、そこを固める `internal/org/prompts_notice_test.go` の断片、multi-org-director の仕様の #318 の項の 1 句に限る。leader.md は変えない(「DECISION が届いたら、そのとおりにしてください」が暫定の DECISION と人の答えの DECISION の両方に当てはまる)。見張りのコードに止めない QUESTION の印を足す直し方は、計画の範囲(Go を変えない)の外なので採らない。cycle 2 の self-review で、暫定の DECISION が leader の承認と cap の読み手と食い違うと分かり、DECISION を送らずにすぐ resolve し、人の答えに `HOOK_SEAT:` を付ける形に変えた(leader.md も変えた)。
