# Verify report: watch-usage-limited

- Date: 2026-11-13 JST(UTC では 2026-11-12)
- Plan: `docs/plans/active/2026-11-13-watch-usage-limited.md`(issue #312 の受け入れ条件 1、#312a)。`- Approved:` の digest `f2971e9c4a70` は、`./scripts/plan-visual.sh digest` で出し直した値と一致した(N2 の直しのあとに承認をやり直した版)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン cycle 1(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(origin/main `4220ea7c`。branch は 78611dbc でこれを取り込んだので merge-base も `4220ea7c`。HEAD `c8681f0d`。23 ファイル、2523 行追加、106 行削除)。Go の本体(`watch_usage_limit.go`、`usage_limit_screen.go`、`watch.go`、`escalate.go`、`resume_screens.go`、`internal/cli/org_watch.go`)とテスト 2 本のソースを読んだ。静的解析は `./scripts/run-static-verify.sh` の既定の範囲(changed、選ばれた pack は golang)。テストは走らせていない(`/test` の範囲)。HEAD の binary を scratch に build し、`ralph org watch --help` を出しただけで、herdr、agmsg、claude、codex は呼んでいない
- Evidence: `docs/evidence/verify-2026-11-13-watch-usage-limited.log`(git が無視する場所)。静的解析の全出力、digest、skill と rule の写しのハッシュ、`wc -l` の base と HEAD、変わったテストのファイル、`watch --help` の出力、条件の一覧の grep を入れた
- Insight event: `docs/insights/events/2026-11-12-watch-usage-limited.jsonl` に verify の 1 行(cycle 1、pass、low 6。V1〜V6 の数)を足した。指示どおり、この report のコミットには入れていない

## Spec compliance

「満たす」は、コードとテストのソースを読んで AC を満たすと判断したという意味。テストが通るかは `/test` が確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 stall が立つはずの座席が上限の文で止まっていれば stall は立たず、org の `usage_limited` が立つ。本文に座席、driver、上限の文(`resets …` を含む) | 満たす | `evaluateSeat`(`watch.go:1325`)は stall の判定が立つ座席だけ `readUsageLimitScreen` で画面を読み、上限なら `cycle.usageLimited` に足して `stalled=false` にする。周期の終わりに `raiseUsageLimited`(`watch.go:1101`)が org の条件を立てる。本文は `usageLimitedEscalationBody`(`watch_usage_limit.go:303`)が `<seat> (<driver>): <line>` を 1 座席 1 行で作る。テスト `TestWatchUsageLimit_SeatsStoppedOnAUsageLimitAreOneItemOfTheOrg` が、stall の ALERT も active な stall もないこと、本文の 3 行(`resets 3:45pm`、`resets 11pm (Asia/Tokyo)`、`try again at 3:05 PM.` を含む)を比べる |
| AC2 同じ周期に同じ org の座席が複数止まれば件は 1 件で、全員が並ぶ | 満たす | 鍵は `conditionKey(orgID, LeaderIdentity, condUsageLimited)` の 1 つ。同じテストが 3 座席で件 1 件、`SEATS: implementer, leader, reviewer` を確かめ、3 周期目に件も ALERT も増えないことも見る |
| AC3 deadman を待たずに同じ周期で件になり、理由は `usage_limit`。届け方は見張りのほかの ALERT の件と同じ | 満たす | Subject が leader なので `checkDeadman`(`watch.go:1812`)の leader の分岐で、その周期のうちに `escalateAlert` に入る。届け方は今までの `escalateToInbox`(`inboxDelivery`)のまま。テスト `TestWatchUsageLimit_TheItemIsDeliveredLikeTheWatchsOtherAlertItems` が `deadman_minutes = 60` で 4 通り(director 無効、有効で見張りが仕事を受ける、受けない、inbox なし)を比べ、どの場合も pending が残らないことを見る |
| AC4 条件が立っているあいだに座席が増えたら、件が open なら recurred 1 つと note、acked か resolve 済みなら新しい件、recurred を足すあいだに resolve されても新しい件 | 満たす | `raiseUsageLimited`(`watch_usage_limit.go:181`)が集合にない座席を `added` にして立ち上がりとし、`recordUsageLimitRecurrence`(`:239`)は `grown` のとき open の件だけに recurred を足す(`:254`)。`ErrInboxResolved` は黙って false(`:260`)。`escalateToInbox` は usage_limited を acked の件の recurrence に畳まない(`watch.go:2020`)。テスト `TestWatchUsageLimit_ASeatThatStopsLaterIsARecurrenceOfAnOpenItemAndANewItemOtherwise` が 4 通りを持つ |
| AC10 件の理由は `usage_limit` で、説明は org の座席が利用上限で止まったことを言う | 満たす | `watchEscalationReason`(`watch.go:1941`)は条件を先に見る。`watchEscalationBody` は usage_limited なら別の本文(`watch.go:2149`)。`assertUsageLimitedBody` が `REASON` と説明の全文を比べ、`about the leader itself` と `leader_is_anomaly_subject` が無いことも確かめる |
| AC5 上限の座席がなくなると条件が下り、再び立ったとき、resolve されていなければ recurrence、されていれば新しい件 | 満たす | 座席なしで `rec.Active=false`、集合を nil にする(`watch_usage_limit.go:185-191`)。再び立つときは `grown=false` なので open と acked の件に recurred を足す。テスト `TestWatchUsageLimit_TheConditionClearsAndALaterLimitFollowsTheEpisodeRule`(open、acked、resolved の 3 通り。状態ファイルに `usage_limited_seats` が残らないことも見る) |
| AC6 画面が読めない、通信断、判断できない画面の座席は今までどおり stall | 満たす | `readUsageLimitScreen` が false を返す道はどれも stall の判定を変えない。テスト `TestWatchUsageLimit_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore`(読みの失敗、長い誤りと制御文字、`claude-enotfound.txt`、空の pane、codex の座席に claude の画面、driver なし、pane なし)と `TestWatchUsageLimit_AScreenReadThatDoesNotAnswerIsCutOffAndTheSeatStalls`。読めなかったときは stall の ALERT の最後に理由の 1 行が付く(D4) |
| AC7 除外の前の stall が立たない座席の画面は読まない。読みは driver の呼び出しの上限で区切られる | 満たす | 読みは `if stalled {` の中だけ。`callWithTimeout` の中で `resumeReadLines`(60 行)を読む(`watch_usage_limit.go:109`)。テスト `TestWatchUsageLimit_TheScreenOfASeatThatIsNotStalledIsNotRead`(4 周期で読み 0 回)、最初のテストが各読みの期限と 60 行を見る。期限切れのテストは、`DriverCallTimeout` を 50 ms にして、応えない読みが打ち切られ、周期が 5 秒のうちに終わることを比べる |
| AC11 上限で止まった leader は、ほかの座席が動いていても、leader の件が未解決でも `usage_limited` に入る。上限でない leader は今までどおり除外 | 満たす | 画面の読みは `leaderIsWaiting`(`watch.go:1333`)より前。テスト `TestWatchUsageLimit_ALeaderStoppedOnAUsageLimitIsReportedWhileItWouldBeWaiting` が 3 つの待ち(座席が動いている、open の QUESTION、acked の BLOCKED)と上限の有無の 6 通りを持つ |
| AC8 上限の文の見分けが既存の fixture で期待どおり | 満たす | `TestUsageLimitScreen_EveryFixture` は `testdata/resume/` の 63 ファイルすべてに期待を持ち、上限は 3 つだけ。`TestUsageLimitSentences_AreStoppedSentences` が印と Stopped の文の対応を固定し(D5)、`TestUsageLimitScreen_EachStoppedSentence` がすべての Stopped の文とグリフで印の有無どおりに分かれることを見る |
| AC9 director と org の skill、agent-messaging.md、仕様、AGENTS.md がこの挙動に合う | 部分的に満たす(LOW) | 主な記述は合う(下の「Documentation drift」)。残りは V1(director skill の recurred の note の文が、条件が下りてから再び立つ場合を外す)と V2(org skill の長時間運用の項の一覧)、AC9 の外の V3〜V5。どれも `/sync-docs` で直せる |

issue #312 の受け入れ条件 1 は AC1〜AC5、AC10、AC11 が受け持ち、コードとテストのソースの上では満たす。実機で上限に当たった座席では確かめていない(Coverage gaps)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0、1 回目で通った) | 範囲は changed で、golang の pack だけが選ばれた。gofmt ok、`go vet ./...` と staticcheck は何も出さず、golangci-lint は `0 issues.`。shellcheck と `sh -n`、`jq -e`、Codex の hook の検査 3 つ、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`、tech-debt の plan の参照、branch の secret scan(`4220ea7c..c8681f0d` clean)も通った。並行の golangci-lint との重なりは起きなかった |
| `./scripts/plan-visual.sh digest <plan>` | `f2971e9c4a70` | plan の `- Approved:` と一致 |
| `shasum -a 256`(skill と rule の写し) | 一致 | director skill の 4 つは `d1226450f726`、org skill の 4 つは `2e8faa72e31e`、`agent-messaging.md` と templates/base の写しは `056f93e4842e` |
| `go build ./cmd/ralph`(出力は scratch)と `ralph org watch --help` | pass | help に `usage_limited` の条件と、増えた座席の扱い(open なら recurrence、acked か resolved なら新しい件)の文が出る。repo には何も残していない(`git status --porcelain` は jsonl の `??` だけ) |
| `git diff origin/main...HEAD --name-only -- '*_test.go'` | 新しい 2 本だけ | 既存のテストのファイルは変わっていない。計画の Regression tests(既存の見張りのテストがそのまま通る)の前提は満たす。通るかは `/test` |
| `wc -l` | 新しいファイルは 800 行未満 | `watch_usage_limit.go` 461、`usage_limit_screen.go` 96、`resume_screens.go` 668(655 から)。`watch.go` は 2146 から 2246(V6 の 357 行目の行) |

## Documentation drift

合っている面: `ralph org watch --help`、`.claude/rules/ralph/agent-messaging.md`(と写し)、`docs/specs/2026-08-01-org-runtime.md` の 2026-11-13 改訂 (a)〜(d) と FR-8 の印、AGENTS.md の `internal/org/` の行、README.md:280、org skill の `watch` の行と「見張りの ALERT の件」の項、director skill の ALERT の件の `CONDITION` と `REASON` の説明。条件が下りてから再び立つ場合を、help は「条件が立っているあいだ」に絞って書き、agent-messaging.md と仕様の (c) は #307a の規則に従うと書いていて、どちらもコードと合う。

| ID | Doc / contract | In sync? | Notes |
| --- | --- | --- | --- |
| V1 | `.claude/skills/director/SKILL.md:1037-1039`(4 つの写し) | 一部ずれる(LOW) | 「履歴の `recurred` の行の `note=` に座席が出ていれば、件が open のあいだにほかの座席も上限で止まったということで…(ack か resolve のあとに止まった座席は、新しい件になる)」は、条件が立っているあいだに座席が増えた場合だけに当たる。条件が下りてから再び立つと、`recordUsageLimitRecurrence` は `grown=false` で acked の件にも recurred を足し、note に止まった座席をすべて書く(`watch_usage_limit.go:254`。テストの AC5 の「e1 acked」がこの形)。このとき ack のあとに止まった座席は新しい件にならず、note は acked の件に付く。直すなら括弧の中を「条件が立っているあいだに、ack か resolve のあとに止まった座席は新しい件になる。条件がいったん下りてから止まった座席は、resolve されていない件の recurred になる」のように絞る |
| V2 | `.claude/skills/org/SKILL.md:2245-2247`(4 つの写し) | 一部ずれる(LOW) | 長時間運用の項の「全 org の停滞/生存/スコープ変更の ALERT、leader が応えない ALERT と leader 自身の ALERT を受信箱の ALERT の件にすること」に `usage_limited` がない。同じ skill の `watch` の行と「見張りの ALERT の件」の項は直っている |
| V3 | `README.md:133`(コマンド表の `ralph org …` の行) | 一部ずれる(LOW) | 「`watch --all` … (stall, liveness, and scope-change ALERTs to its leader, and an `ALERT` item in the inbox for an ALERT the leader did not act on or one about the leader; …)」に `usage_limited` がない。README.md:280 は直っている |
| V4 | `docs/quality/quality-gates.md:83` と `templates/base/docs/quality/quality-gates.md:82` | ずれる(LOW、S3 で既知) | Watchdog pulse layer の行は stall / liveness / scope-change と、deadman と leader 自身の 2 つの escalation だけを書く。`usage_limited` と、その件がその周期のうちに記録されることを足す。templates/base の写しを直すと、tech-debt の 118 行目の Trigger(「`templates/base/docs/quality/quality-gates.md` … への次のタッチ」)に当たるので、その行に注記が要る |
| V5 | `docs/specs/2026-10-07-org-multi-org-director.md:114` と `:117` | ずれる(LOW、:114 は S3 で既知) | :114(2026-10-23 の決めたこと)は escalation の理由を `leader_is_anomaly_subject` と `deadman_timeout` の 2 つと書き、受信箱に書けないときの条件にも usage_limited の座席の検査がない。:117(2026-11-12、#307 の前半)は「その件が open か acked のあいだに条件がまた立つと、見張りは新しい件を作らず…続きのあいだは leader に同じ ALERT を送らず」と書くが、usage_limited で座席が増え、件が acked のときは新しい件になり、leader にも ALERT がもう 1 通行く。どちらも日付の付いた記録なので書き換えず、FR-6 に 2026-11-13 の決めたことを 1 つ足すのがよい(org-runtime の仕様の 2026-11-13 改訂 (a)〜(c) を指す) |
| V6 | `docs/tech-debt/README.md` の Trigger | 注記が要る(LOW) | 下の 8 行の Trigger にこの branch が当たり、どれも払っていない。各行に「hit, not paid, carried over」の注記を足す。88 行目(`internal/org/watch.go` に触れたので (1) `status.Escalated` と (3) の空の `SEAT:` に当たる)、180 行目 (a)(org skill のコマンド表の `watch` の行。3233 字から 3378 字)、262 行目(director skill の「受信箱の処理」の「件の処理」を変えた)、353 行目(`sendAlert` が pending を返すようにした。送り方と期限は同じ)、357 行目(escalation のコード: `escalateAlert`、`escalateToInbox`、`checkWatchEscalationValues`、`watchEscalationBody`、`escalateWithoutInbox`。`watch.go` は 2146 行から 2246 行、`watch_test.go` は 4463 行のまま。新しいコードは `watch_usage_limit.go` に置いた)、462 行目(`evaluateSeat` の stall のコード)、464 行目(director skill を変えた。scaffold の版の知らせの扱いは今も書いていない)、477 行目(見張りの ALERT の条件を足し、`resume_screens.go` に印を足した。「Hooks need review」の条件は足していない。見張りが stall の座席の画面を読むようになったので、その条件を足すときは同じ読みに載せられる) |

INFO(判定には使わない):

- I1 tech-debt の 452 行目(claude の session、Opus、Sonnet と codex の 2 つの上限の文は実機で見ていない)の Impact は `ralph org resume` だけを書く。この branch から、見張りの `usage_limited` も同じ表を使う。実機の文が違えば、その座席は今までどおり stall になる(安全側)。Impact に一言足すかは `/sync-docs` の判断
- I2 leader の role prompt(`internal/org/prompts/leader.md`)は `usage_limited` の ALERT を受けたときの扱いを書いていない。計画の文書の範囲になく、ALERT の本文が org の条件だと書いているので、ずれとは扱わない
- I3 director skill の「座席を立て直す動詞は issue #312 で扱っていて、この版にはない」は、#312c(origin/docs/spec-org-reseat、`80f58f86`、未 merge)が先に入れば古くなる。その branch との試しの merge(`git merge-tree`)では、AGENTS.md と `docs/specs/2026-08-01-org-runtime.md` がぶつかり、skill の 4 つの写しは自動で混ざった。後から merge する側が直す(計画のリスクの 4 つ目)
- I4 計画の AC1〜AC11 の箱と Progress の `Implementation started` などは `[ ]` のまま。記録の遅れで、判定には使わない
- I5 未コミットの jsonl の self-review の行は最初の run の数(low 7)のまま。self-review の report もそう書いている
- I6 self-review の report は N2 を「未解決」、digest を `0746c70f0d2b` と書くが、どちらも書いた時点の記録。N2 は c8681f0d で直り、今の digest は `f2971e9c4a70` で一致する

## Observational checks

- HEAD の binary の `ralph org watch --help`: `usage_limited` が条件の一覧と、その説明の段落(stall を立てない、leader の鍵、その周期のうちに件、`REASON usage_limit`、`SEATS`、増えた座席の扱い)に出る。全文は evidence の「ralph org watch --help」の節
- 実行時の観察(偽の herdr での見張りの周期、実機の座席)はしていない。テストの偽の herdr が同じ形を通すので、`/test` の結果で代える

## Coverage gaps

- テストは走らせていない。新しい 17 本と、既存の見張りのテストがそのまま通るかは `/test` が確かめる
- 計画の Test plan の Edge cases のうち「leader 以外だけが上限」に当たるテストがない。上限で止まった座席がいて leader がいるが上限でない場合を、どのテストも作らない(AC4 のテストは reviewer が動いているが、leader は上限)。コードの上では Subject はいつも leader なので道は同じと読めるが、`/test` で 1 本足すと確かになる
- 計画の Progress が書く変異(S2 の 3 つ、S4 の 4 つ)は走らせ直していない。読んで推した範囲にとどまる
- 実機: claude の session、Opus、Sonnet と codex の上限の画面は見たことがない(tech-debt 452 行目)。見張りがこの文で usage_limited を立てることは、fixture の上でしか確かめていない
- self-review の L5(読みが 1 回失敗すると集合から外れ、次の周期で「増えた」と数えられる)と N1(読めなかった理由は leader への ALERT にだけ載り、受信箱の件には届かない)は計画どおり残る。PR の Known gaps に書く予定(計画の Progress)
- 決定論の検査がない面: 見張りの条件の一覧は 10 か所ほどにあり(README の 2 か所、quality-gates の 2 つ、org skill の 3 か所、director skill、agent-messaging、仕様 2 本、help)、今回も V2〜V5 が残った。足すなら、`internal/org` に条件の名前(`stall`、`liveness`、`scope_change`、`usage_limited`)を並べ、一覧を持つファイルの決まった行がすべての名前を含むことを見る小さなテストが一番安い

## Verdict

- Verdict: pass
- Verified: AC1〜AC8、AC10、AC11 をコードとテストのソースで確かめた。静的解析(gofmt、go vet、golangci-lint、staticcheck、sync 系の検査、secret scan)は通った。承認の digest は一致し、skill と rule の写しは同じ
- Partially verified: AC9。主な文書は合い、V1(director skill)と V2(org skill の 1 か所)が LOW で残る。AC9 の外の V3〜V6 も `/sync-docs` で直す
- Not verified: テストの実行、実機の上限の画面、変異の再実行、「leader 以外だけが上限」の場合
