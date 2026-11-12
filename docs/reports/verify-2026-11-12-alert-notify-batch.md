# Verify report: 見張りの ALERT の desktop notification を周期ごとにまとめ、`wait --inbox` を種類で絞る(#307 の後半)

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-alert-notify-batch.md`(digest `a805c3b2b4b4`。`./scripts/plan-visual.sh digest` の値と `- Approved:` の行が一致)
- Verifier: verifier subagent(Claude Code)。パイプラインの cycle 1(`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff 94e61c38...HEAD`(HEAD `59961e6a`、36 ファイル、+2002/-93)。コードの commit は S1 `01fcf3fb`、S2 `85889ad9`、S3 `9175cae1`、S4 `5752476a`(文書)、S5 `1f2197dc`(self-review の L1〜L4 の直し)。記録の commit(計画、self-review の 2 回、insight)は照合にだけ使った。origin/main は `94e61c38` のまま(`git ls-remote`)なので base との衝突はない。テストは走らせていない(/test の役目)。テストの欄は、テストがあり assert が AC を言っていることまでを読んで確かめた
- Evidence: `docs/evidence/verify-2026-11-12-alert-notify-batch.log`(git が無視する)。静的解析の全出力、skill の 4 つの写しの `cmp`、計画の digest、`AlertNotify` を読む箇所と送りの呼び出し元、行数の base と HEAD、tech-debt の Trigger の照合。スクリプト自身の写しは `docs/evidence/verify-2026-11-12-061012.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 掃き出し 1 回と org 1 つの deadman の段で、ALERT の件の通知は 1 回、文に件数・最も古い件・その org。各件の `notified` に同じ結果、line と banner は件ごと | Met | 掃き出しは `internal/org/deadline.go:224` で batch を作り、`:243` で各件を `sendInboxItemToHuman` に渡し、`:247` で 1 回 flush してから Sent と Failed を分ける。`internal/org/escalate.go:494-501` が `escalations.jsonl` の行と banner を先に書き、`:502-505` で ALERT の件だけを batch に入れる。`internal/org/alert_notify.go:78-99` が通知を 1 回送り、各件に同じ `osascript` で `AppendNotified` する。文は `:109-121`(e の数が最小の件。`notifiableOrgID` で守る)。deadman の段は `internal/org/watch.go:1725` の batch を `escalateAlert`(`:1752`)から `escalateToInbox`、`escalateRecordedItem`、`sendAlertItemToHuman`(`:1919`、`:1950`、`:2002`、`:2013`)へ渡し、`:1755` で flush する。テスト `TestSweepInboxDeadlines_SendsTheAlertItemsOfOneSweepInOneNotification`(`alert_notify_test.go:117`。ALERT 3 件と QUESTION 1 件で通知 2 回、`ok` と `failed: <err>` の 2 形)、`TestWatch_Deadman_SendsTheAlertItemsOfOneOrgsStepInOneNotification`(`:339`。org 2 つで 2 回) |
| AC1b 掃き出しのまとめた送りは director の確かめと座席の見回りより前に終わる | Met | `watch.go:753-765` の `cycle()` は `SweepInboxDeadlines`(flush を含む)が返ってから `checkDirector`(`:761`)、`wake`、`pulse`(`:763`)へ進む。テスト `TestRunWatchAll_SendsTheSweepsAlertNotificationBeforeThePulse`(`alert_notify_test.go:281`)は pulse の最初の herdr の呼び出しの時点で通知 1 回、未通知の件 0 を見る。director の確かめより前であることは、コードの順で確かめた(テストに headless の director は登録されていない。Coverage gaps 1) |
| AC1c `notified` の追記に失敗した件は送りの失敗として記憶され、ack の期限の長さのあいだ送り直されない。追記できた件は送り直されない | Met | `deadline.go:246-261`。flush の結果の err が `send.Err` を上書きし、失敗は `holdsBack` に渡したのと同じ `t.key` で `mem.recordFailure` する。テスト `:215` は、flush の中で e2 の行を壊し、12 分で held back、21 分で e2 だけを 1 件の文で送り直し、22 分で何も送らず、e1 と e3 は 1 回だけであることを見る |
| AC2 ALERT の件が 1 件だけなら今までと同じ文で 1 回 | Met | `alert_notify.go:110-113` は 1 件なら `inboxHumanTextFor`(件ごとの送りと同じ関数)の題と文を返す。テスト `TestAlertNotifyText_NamesTheCountTheOldestAndItsOrg`(`:502`)の 1 件の 2 形、`TestSendInboxItemToHuman_AnAlertItemOfABatchNeverFlushedIsSentAtTheAckDeadline`(`:436`) |
| AC3 `off` では通知なし、`osascript` は `off`、banner と行は出て、期限で送り直されない | Met | `alert_notify.go:84-88`(`inboxNotifyOff` は `escalate.go:75` の `"off"`)。行と banner は batch に入る前に書かれる(`escalate.go:494-501`)。`notified` の event があるので ack の期限に掛からない。テスト `:178`(次の掃き出しと、記憶のない立て直しの掃き出しで 0 件。QUESTION だけ通知)、`:383`(deadman の段) |
| AC4 leader の件、`inbox notify`、予備の道は 1 件ずつで `alert_notify` を読まない | Met | `Escalate`(`escalate.go:221`)と `NotifyInboxItem`(`:276`)は batch に nil を渡す。`reconcilePendingAlertsWithInbox`(`watch_recurrence.go:138`)も nil。予備の道 `escalateWithoutInbox` は batch を受け取らない(`watch.go:1854-1856`)。掃き出しの中の leader の件は `takes` が ALERT 以外で false を返す(`alert_notify.go:59-61`)ので、順番が来たときに通知される。`Watchdog.AlertNotify` を読む本番コードは `alert_notify.go:85` の 1 か所だけ(evidence の 4)。テスト `TestNotifyInboxItem_AlertNotifyOffDoesNotStopTheSendsOutsideTheWatchsSteps`(`:466`) |
| AC5 送りの前に止まった見張りの件は `notified` を持たず、次の見張りの ack の期限で送られる | Met | batch に入った件は行と banner だけを持ち、flush されなければ `notified` が無い(`escalate.go:502-505`)。テスト `:436` は flush しないまま掃き出しが 10 分で送らず 11 分で送ることを見る。deadman の段の件については、次の見張りが送る時刻が ack の期限より早くなる形がある(V5、INFO) |
| AC6 既定は `"batch"`、ほかの値は検証で断る。テンプレートに既定値とコメント、既定値の一致のテスト | Met | `internal/config/config.go:818`(既定)、`:1036-1040`(空文字を含むほかの値を断る)。`templates/base/ralph.toml:251-267`。`internal/config/defaults_sync_test.go` の `orgTemplateDrift` と `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides` に `org.watchdog.alert_notify` の行。`TestLoad_OrgWatchdogAlertNotify`(空文字、`each`、`BATCH`、`"off "` を断る)、`internal/cli/config_test.go` の `ralph config get` の 2 行。`Load` は知らないキーを断らない(`Undecoded` を使っていない)ので、計画の Rollout の「古い binary では知らないキー」は合う |
| AC7 `--types QUESTION,BLOCKED,RESULT` は ALERT だけのあいだ返らず、指定の種類が届けばその件だけを出す。既定は今と同じ。知らない種類と空の項目は待つ前に断る(issue の受け入れ条件 4) | Met | `internal/cli/org.go:1546-1548`(`--inbox` が無ければ断る)、`:1599-1606`(state dir を決める前に `org.ParseInboxWaitTypes`)、`:1614`。`escalate.go:364-374`(4 つの TYPE の完全一致だけ。空、小文字、空白入り、知らない値を断り、受け付ける値を出す)、`:341-353`(`openInboxItems` の絞り込み)。テスト `TestOrgWaitInbox_TypesPrintOnlyTheListedItems`、`TestOrgWaitInbox_TypesKeepWaitingPastAnAlertItem`(300 ms で時間切れ、exit 1、文は今のまま。あとから来た QUESTION だけを出す)、`TestOrgWait_TypesIsRefusedWithoutInboxOrWithABadValue`(state dir を作らない)、`TestParseInboxWaitTypes_*`、`TestWaitInbox_Types*`。issue #307 の受け入れ条件 4「`ralph org wait --inbox` で、ALERT を除いて待てる」はこの形で満たす(V11 も参照) |
| AC8 `--director` と `--types` で、終了コード 4 の判定と印の件数は指定の種類だけで数える | Met | `internal/org/director_wait.go:197`(`Types`)。loop の open の読みは `:528` の `openInboxItems(w.p.Types)` だけで、`:533` の件数、`:536-537` の `endNoWatch` への分岐、`:540` の時間切れの detail(`directorWaitTimeoutDetail`、`:746-751`)はその結果で決まる。stderr の時間切れの文(`inboxWaitTimeoutPhrase`)は変わらない。CLI は `internal/cli/org_wait_director.go:73`。テスト `TestOrgWaitDirector_AnAlertItemLeftOutByTypesDoesNotHoldBackExit4`、`TestOrgWaitDirector_TypesPrintAndCountOnlyTheListedItems`(印の detail が `1 open inbox item`)、`TestOrgWaitDirector_ABadTypesValueIsRefusedBeforeTheRegistrationCheck`(exit 5 でなく 1、印も lock も作らない)、`TestWaitInboxForDirector_Types*` |
| AC9 director skill と org skill(4 つの写しずつ)が処理の順、ALERT の回数の読み方、`--types`、`alert_notify` を書き、`check-skill-sync.sh` が通る | Met | director skill: `.claude/skills/director/SKILL.md:68-75`(まとめと `off`、4 つの例外)、`:892-897`(`--types` と既定の待ちを続ける理由)、`:940-942`(QUESTION・BLOCKED → RESULT → ALERT)、`:988-992`(`recurred: <N> times (last <ts>)`。`internal/cli/org_inbox.go:482-487` の書式、1 回なら `1 time` と合う)。org skill: wait の行(`:360`)、watch の行(`:367`)、inbox の行(`:370`)、受信箱の節(`:658-665`)。`check-skill-sync.sh` は ok(14 skill)。director と org の 4 つの写しは `cmp` でバイト列が同じ |
| AC10 `watch --help`、`wait --help`、仕様の注、tech-debt の行が挙動に合う | Met | `internal/cli/org_watch.go:83-91`、`internal/cli/org.go:1508-1519` と `:1579-1580`。仕様は FR-5 の `:98` と FR-6 の注 `:117`。tech-debt 462 行の (4) は 4 つの列すべてで取り消し線と closed になり、Related の列に計画がある。help はテスト `TestOrgWatch_HelpDescribesTheAlertNotifyBatch` と `TestOrgWait_HelpDescribesTypes` が文を固定する。要約の言い方に INFO が 1 つ(V4) |

計画の決定との照合: D1〜D6 はコードと一致した。D2 の「1 周期の ALERT の通知は多くても 1 + (escalate した org の数)回」は、batch を作るのが `deadline.go:224` と `watch.go:1725`(`checkDeadman` は org ごとに 1 回、`watch.go:1075`)だけであることで確かめた。D2b は AC1c の行。D4 は AC5 の行と V5。計画の本文(digest の内側)は変わっていない。

issue #307 との照合: 受け入れ条件 4 はこの PR で満たす(AC7)。「やること」の通知のまとめと切る設定も満たす(AC1〜AC6)。受け入れ条件 1〜3 は PR #314 の範囲で、この verify では見ていない。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(exit 0) | `templates/base/ralph.toml` が言語に分類されないため full に落ちた。shellcheck と `sh -n`(hooks)、`jq -e`(settings)、Codex の hook の 3 つの guard、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照、golang(gofmt ok、`go vet ./...`、golangci-lint `0 issues.`、staticcheck)、branch の secret scan(`94e61c38..59961e6a` clean) |
| `./scripts/check-skill-sync.sh` と 4 つの写しの `cmp` | Pass | director と org のそれぞれで `.claude`、`.agents`、`templates/base` の 2 つの写しが同じバイト列 |
| `./scripts/plan-visual.sh digest` | Pass | `a805c3b2b4b4`、`- Approved:` の行と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org watch --help` | Yes | まとめ、`off`、4 つの例外がある。1 件の段の文の言い方は V4(INFO)。`While an ALERT for the same org, seat, and` の行が詰め直されず短いまま(V10、INFO) |
| `ralph org wait --help` | Yes | `--types` の値、既定、断る形、`--director` との組み合わせ、外した ALERT も ack の期限に人へ届くこと |
| `templates/base/ralph.toml` | Yes | 既定値とコメント、4 つの例外。V4(INFO) |
| director skill(4 つの写し) | Yes | V4(INFO)。既定の待ちを続ける理由(D5)がある |
| org skill(4 つの写し) | Yes | wait、watch、inbox の行と受信箱の節。inbox の行の例外の省略は self-review が観察として残した要約の範囲 |
| 仕様 FR-5・FR-6(`docs/specs/2026-10-07-org-multi-org-director.md`) | Yes | `:98` と `:117`。lead が /sync-docs 向けに挙げた「注 117 に alert_notify の 4 つの例外が要るかもしれない」は、注にすでに 4 つ(leader の件、`ralph org inbox notify`、受信箱に書けないときの予備の道、周期の初めの escalate のし直し)があり、S5 でそろえた他の 5 か所と同じ中身であることを確かめた。直すことはない。D4 の言い方は V5(INFO) |
| tech-debt「Eleven gaps」(462 行)の (4) | Yes | 4 つの列で閉じた。Related の列の「AC7 to AC10, which close (4)」は V9(INFO)。同じ行のほかの項目の Trigger は V1 |
| tech-debt のほかの行 | No(LOW) | V1〜V3。この PR が引いた Trigger に、払わなかった記録が無い |
| `AGENTS.md` | Yes | V7(INFO) |
| `docs/quality/quality-gates.md`(2 つの写し) | Yes | V6(INFO) |
| `.claude/rules/ralph/agent-messaging.md`、役割 prompt | Yes | desktop notification の言い方を持たない(grep 0 件) |
| 計画 | Yes | lead の既知の項目(Risks の通知の遅れの文が言い過ぎ)は Progress の self-review の行に記録がある。AC のチェックボックスは V8(INFO) |

### 所見

LOW(3 件。どれも /sync-docs への記録の依頼で、コードは変えない。前の PR の書き方「<日付>: <branch> hit this trigger with ..., but did not pay it: ... Carried over to the next change that meets this trigger」に合わせるとよい)

- V1(LOW)tech-debt 462 行「Eleven gaps」: この PR は (4) を閉じたが、同じ行で引いた Trigger の記録が無い。(a) 行全体の Trigger の「or the start of #307b」を取り消し線にしただけで、#307b が始まって (1)〜(3)、(5)〜(11) を払わなかったことを書いていない。(b) 項目 (9) の「the next change to `escalateToInbox`、... `reconcilePendingAlertsWithInbox`」を引いた(どちらも batch の引数と doc が変わった。`watch.go:1905`、`watch_recurrence.go:136-138`)。(c) 項目 (10) の「the next change to the "recorded" branches of `escalateToInbox` or to `reconcilePendingAlertsWithInbox`」を引いた(`watch.go:1919` が batch を渡す)。(d) 項目 (11) の「the next edit of the doc of `escalateAlert`」を引いた(`watch.go:1843-1848` に batch の段落を足した)。(e) D4 が新しい止まる窓(deadman の段で batch に入れてから flush するまで)を作ったが、(9) の窓の一覧(W1〜W9)に無い。V5 の形もこの窓から生じる
- V2(LOW)tech-debt 319 行: Trigger の「or the next change that takes `internal/org/director_wait.go` past 800」をそのまま満たした。`director_wait.go` は 799 → 821 行(S3 と S5)、`director_wait_test.go` も 687 → 824 行で 800 を越えた。分割も記録も無い
- V3(LOW)次の行の Trigger を引いたが、記録が無い。357(watch の escalation とテスト。`watch.go` 2090 → 2139 行、`watch_test.go` は 1 行、`watch_recurrence.go` 273 → 275 行。行が持つ数は 14bc17a3 の 2090 のまま)、356(`sendInboxItemToHuman` と呼び出し元の失敗の行。`alertSendNotCompleted` を足した)、355(`SweepInboxDeadlines`)、262(director skill の「受信箱の処理」、`WaitInbox`、`SweepInboxDeadlines`、`RunWatchAll` の doc)、258(`deadline.go`)、381(`director_wait.go` の wait の loop)、303(`watch_test.go`)、415(`internal/config/config.go` 1138 → 1172 行)。どの行も前の PR が同じ形の記録を残している

INFO

- V4: watch の help(`org_watch.go:83-87`)、テンプレート(`ralph.toml:254-256`)、`config.go` の doc(`:327-330`)、director skill(`:68-70`)は、段ごとの 1 回の通知が「件数と最も古い件を出す」とだけ書く。件が 1 件の段は今までの件ごとの文になる(AC2、`alert_notify.go:110-113`)。org skill と仕様の注は「2 件以上なら」「1 件だけなら今までと同じ文」と書き分けている。要約の範囲
- V5: D4、AC5、仕様の注 117 は、送りの前に止まった見張りの件を「次の見張りが ack の期限に送り直す」と書く。deadman の段の件は状態ファイルが保存されない(保存は `checkDeadman` のあと、`watch.go:1075-1080`)ので、次の見張りの最初の周期の `reconcilePendingAlertsWithInbox` が `escalateRecordedItem`(`watch.go:1994-2002`)に通す。その見張りが記録と同時に送る見張り(director が無効、`--once`、止める頼みが出ている)なら、件はその場で 1 件ずつ送られ、ack の期限より早い。知らせる側に倒れる向きで、「周期の初めの送り直し」の例外として help と skill に書かれている。/sync-docs で仕様の注に一言足すかを決めてよい
- V6: `docs/quality/quality-gates.md:83`(テンプレートの写しも)の watchdog の行は「an item past a deadline is sent to the human (`escalations.jsonl`, the watch's stderr, desktop notification)」のまま。まとめと `off` は書いていない。#307a はこの行に再発を足したので、同じ粒度で 1 句足す余地がある
- V7: `AGENTS.md` の `internal/org/` の項は `alert_notify.go` も `--types` も挙げない。`AGENTS.md` は org のファイルをすべては挙げない(ほかに 15 ファイルが無い)ので drift ではない。計画の S4 の判断と同じ
- V8: 計画の AC1〜AC10 のチェックボックスはすべて `[ ]`、Progress の「Review artifact created」も `[ ]` のまま(self-review の報告はある)。/pr までに付ける記録の遅れ
- V9: 462 行の Related の列の「(D2, D3, D5, D6, and AC7 to AC10, which close (4))」は、(4) のうち通知のまとめの半分を閉じる AC1〜AC6 を挙げていない(D2 と D3 は挙げている)
- V10: watch の help の `"notified one by one and do not read it.\n"` のあとの `"While an ALERT for the same org, seat, and\n"` が詰め直されず、`--help` の出力で短い 1 行になる
- V11: issue #307 の「何が問題か」4 は、pull の director の待ちが ALERT のたびに起こされることも挙げる。`/director` skill は既定の待ち(すべての種類)のまま(D5)なので、director の待ちは今も新しい ALERT の件で返る。受け入れ条件 4 が求めるのは「ALERT を除いて待てる」ことで、それは満たす。PR 本文で D5 の理由(外した ALERT も ack の期限に人へ届くので、director が見ないあいだに人へ送られる)を書くとよい
- 322 行は Trigger(`director_wait.go` の文)を引いたが、新しい文 `directorWaitTimeoutDetail` には `TestDirectorWaitTimeoutDetail` があり、行の求める「触った block ごとのテスト」は満たす。358 行(引数は 7 つ目だが string ではなく、本番の呼び出しは 4 か所のまま)、453 行(`validateLoaded` の `[org.watchdog]` の部分)、210 行(`evaluateCycle` の doc だけ)は引いたとまでは言えない

## Observational checks

- 計画の digest は `a805c3b2b4b4` で、承認の行と一致した
- origin/main は base の `94e61c38` のままで、branch は origin/main の上にある
- self-review の L1〜L4 の直し(`1f2197dc`)を読み直した。L1 の 4 つの例外は、テンプレート、`config.go`、watch の help、director skill、org skill、`alert_notify.go` の先頭の comment、仕様の注 117 の 7 か所で同じ中身
- `ralph org inbox show` は `off` の `notified` を `notified: yes (<ts>)` と出し、履歴の行に `osascript=off` が出る(`internal/cli/org_inbox.go:473`、`:502`)。org skill の「どちらでも件の `NOTIFIED` は `yes`」と合う

## Coverage gaps

1. AC1b の「director の確かめより前」は、`cycle()` のコードの順で確かめた。テストは headless の director を登録しないので、最初の herdr の呼び出しは pulse のもの。最も小さい足しは、herdr の headless の director を登録した fixture で、`checkDirector` の最初の herdr の呼び出しの時点で通知が 1 回送られていることを見るテスト 1 本
2. V5 の形(deadman の段で止まり、記録と同時に送る見張りが立て直す)はテストが無い。/test の判断に任せる
3. 実機の macOS の通知(回数、NotificationCenter の負荷)は見ていない。issue の数字(1 日 242 回の通知)がどれだけ減るかは、下流の運用でしか測れない
4. テスト、`go build`、mutant は走らせていない(/test の役目)

## Verdict

- Verified: AC1〜AC10(AC1b、AC1c を含む)を、コードの道と assert を読んで Met と判定した。issue #307 の受け入れ条件 4 はこの PR で満たす。`./scripts/run-static-verify.sh` は exit 0。skill の 4 つの写しは同じ。計画の digest は一致
- Partially verified: AC1b の director の確かめの半分(コードの順だけ。Coverage gaps 1)。AC5 の deadman の段の形(V5)
- Not verified: テストの実行結果、実機の通知
- 所見: LOW 3(V1〜V3、tech-debt の記録。/sync-docs へ)、INFO 8(V4〜V11)。CRITICAL、HIGH、MEDIUM はない
- 判定: pass
