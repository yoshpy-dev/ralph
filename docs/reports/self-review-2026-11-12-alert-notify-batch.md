# Self-review report: alert-notify-batch

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-alert-notify-batch.md`(issue #307 の後半)
- Branch: `fix/alert-notify-batch`、base は `origin/main` の `94e61c38`。レビューした HEAD は `31ee36a8`(S1 から S4 の 4 コミットと Progress の記録)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff 94e61c38...HEAD` の diff の質だけ。`internal/org/alert_notify.go` を全部、`escalate.go`・`deadline.go`・`watch.go`・`watch_recurrence.go`・`director_wait.go` の変更、`internal/cli/org.go`・`org_wait_director.go`・`org_watch.go`、`internal/config/config.go` の変更を読んだ。テストは各ファイルの形と、まとめ・失敗の記憶・`--types` の主なものを読んだ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)
- ID の付け方: このレポートの指摘は `L1` から `L4`

## Evidence reviewed

読んだもの。

- `internal/org/alert_notify.go`(121 行、全部)。`alertNotifyBatch`、`takes`、`add`、`flushAlertNotifyBatch`、`alertNotifyText`
- `internal/org/escalate.go`。`sendInboxItemToHuman` の `batch` 引数と、ALERT だけを batch に入れて返す分岐、`WaitInbox`、`openInboxItems`、`ParseInboxWaitTypes`
- `internal/org/deadline.go`。`SweepInboxDeadlines` の全体(`tried`、`notifiedErrs`、失敗の記憶の更新)と、それを使う `watch.go` の `cycle`(`sweep.Failed` の出し方)
- `internal/org/watch.go`。`checkDeadman`、`flushAlertNotifications`、`escalateAlert`、`escalateToInbox`、`escalateRecordedItem`、`sendAlertItemToHuman`、`alertSendNotCompleted`、`leaderActedSinceAlert`。`watch_recurrence.go` の `reconcilePendingAlertsWithInbox`
- `internal/org/director_wait.go`(`Types` と、ループの `openInboxItems(w.p.Types)`)、`internal/cli/org.go` と `org_wait_director.go` の `--types`、`org_watch.go` の help
- `internal/config/config.go`(定数、構造体、既定値、`validateLoaded`)、`templates/base/ralph.toml`
- 呼び出し元の洗い出し。`sendInboxItemToHuman` の本番の直接の呼び出しは 4 か所(`Escalate`、`NotifyInboxItem`、`SweepInboxDeadlines`、`sendAlertItemToHuman`)。batch を渡すのは `SweepInboxDeadlines` と、`checkDeadman` から来る `sendAlertItemToHuman` で、`Escalate` と `NotifyInboxItem` は `nil`。`reconcilePendingAlertsWithInbox` は `escalateAlert` に `nil` を渡し、`sendAlertItemToHuman` まで `nil` のまま届く。`AlertNotify` を読む本番コードは `alert_notify.go:85` の 1 か所だけ。`rt.Config = cfg` は `org_watch.go:208`
- tech-debt(`docs/tech-debt/README.md`)の変更を文字単位で差分にした。変わったのは 1 行(長い行)だけで、(4) に関わる 5 か所の取り消し線と `(RESOLVED ...)` / `(closed)` の追記、末尾の計画への参照の追加に限られる。ほかの文字は動いていない

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 見張りを黙らせる道。通知を止める分岐は `o.Config.Watchdog.AlertNotify != config.AlertNotifyOff` の反対側だけで、空文字や知らない値は 1 回の通知に倒れる(`alert_notify.go:85`)。`Load` は `batch` と `off` 以外を断る(`config.go:1036`)。batch に入れたあと flush までに抜ける道は無い。`checkDeadman` の flush は for の外で、ループの中に `return` は無く、`SweepInboxDeadlines` の flush も for の外。batch に入れたまま見張りが止まると `notified` が無く、次の見張りが ack の期限で送り直す(D4)
- 失敗の勘定。batch に入った件は、escalations.jsonl の行が書けなければその error が残り(`lineWritten=false` の件は `AppendNotified` を試さず flush の結果にも出ない)、書けていれば `notified` の追記の error に置き換わる。`Sent` に入るのは `send.Err == nil` の件だけで、失敗の記憶は flush のあとに全件分を更新する。batch に入らない件(ALERT 以外)の扱いは前と同じ
- batch に入らない道。`batch.takes` は nil と ALERT 以外で false。`escalateWithoutInbox` は batch の引数を持たない。`reconcilePendingAlertsWithInbox` と `Escalate` と `NotifyInboxItem` は `nil` を渡す
- 文の値の守り。2 件以上の文は org を `notifiableOrgID` に通し、件の ID は `inboxFold.apply` が `e<N>` 以外を捨てる(`inbox.go:625`)ので、そのまま書ける。本文と `alert_id` は通知に出ない。`slices.MinFunc` は空の slice で panic するが、`flushAlertNotifyBatch` が空を先に返し、1 件の場合は先に分けている
- `ParseInboxWaitTypes`。`strings.Split("", ",")` は `[""]` で、空の項目として断られる。各項目は `inboxItemTypes` と完全一致で、エラー時は nil を返す。2 つの待ちは同じ `openInboxItems` を通るので、件数・返す件・終了コード 4 の「open が無い」・印の件数が同じ種類で数えられる
- 差分の追加行に不可視 Unicode と U+FFFD は無い。80 桁を超えるコメント行は `watch.go` で 7 のまま(base と同数)、新しい `alert_notify.go` は 0。director と org の skill の 4 つの写しは、追加・削除された行の集合がファイルごとに一致する

読んでいない・実行していないもの。テスト、`go build`、`go vet`、linter は走らせていない(このレビューの対象外)。`watchAllRun` の周期を実際に回した順序は、テスト(`TestRunWatchAll_SendsTheSweepsAlertNotificationBeforeThePulse`)の読みだけで、実行していない。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `alert_notify` を読まない送りの一覧が 5 か所で別々の中身になっている。利用者が最初に読むテンプレートの comment は 2 つだけで、`off` でも通知が出る 2 つの道(周期の始めの再送、受信箱に書けないときの予備の道)を書いていない。 | テンプレート `templates/base/ralph.toml:257-258` は「leader の件と `inbox notify`」の 2 つ。`internal/config/config.go:333-335` は leader・`inbox notify`・予備の道の 3 つ。`internal/cli/org_watch.go:88-90`(help)は leader・予備の道・周期の始めの再送の 3 つ。org skill は 4 つすべて。director skill は例外を書かず「`off` なら出さない」だけ。コードでは `reconcilePendingAlertsWithInbox`(`watch_recurrence.go:138`)が `nil` の batch を渡し、`sendInboxItemToHuman` は batch が無いとき `AlertNotify` を読まずに通知する。`AlertNotify` を読むのは `alert_notify.go:85` だけ。 | org skill の 4 つの道の文を、テンプレートと `config.go` の comment に同じ形で置く(director skill は「ほかの道は `off` でも 1 件ずつ出る」の 1 句)。道が増えるたびに 5 か所を直すのを避けたいなら、`config.go` の comment を正本にして、ほかは「一覧は `OrgWatchdogConfig.AlertNotify` の comment」と指す形でもよい。 |
| LOW | readability | `--types` で待つとき、ALERT が open のまま時間切れになると、印の `detail` が「no open inbox item within N ms」になる。open の件はあるので事実と違う。`ralph status` と `ralph org director status` がこの印を読む。 | `internal/org/director_wait.go:539`(`w.end(DirectorWaitEndTimeout, fmt.Sprintf("no open inbox item within %d ms", ...))`)。stderr の文の方は `inboxWaitTimeoutPhrase`(`escalate.go:102`)で、director skill が一致を見て再実行を決めるので変えられない。印の `detail` は skill の一致の対象ではない(テストは `"200 ms"` を含むかだけを見る: `director_wait_test.go:183`)。 | `len(w.p.Types) > 0` のときだけ、印の `detail` を「no open inbox item of <種類> within N ms」にする。stderr の phrase は変えない。直さないなら、director skill の `--types` の段落に「時間切れの文は左に置いた種類の件が open でも同じ」と 1 行足す。 |
| LOW | maintainability | `SweepInboxDeadlines` が失敗の記憶のキー `inboxDeadlineKey` を 2 か所で別々に作っている。`holdsBack` に渡すキーは loop の中の `key` で、`recordFailure` に渡すキーは flush のあとの loop で `send.ID` と `send.Reason` から作り直したリテラル。`tried` の要素がキーを持っていないため。 | `deadline.go:231`(`key := inboxDeadlineKey{id: item.ID, reason: reason}`)と `deadline.go:249`(`mem.recordFailure(inboxDeadlineKey{id: send.ID, reason: send.Reason}, now)`)。キーに欄が増えたとき、片方だけ直ると失敗した送りが held back にならず毎周期やり直される。AC1c のテストは今の 2 欄では通るので、欄が増えるまで気づけない。 | `tried` を `[]struct{ send InboxDeadlineSend; key inboxDeadlineKey }` にするか、`InboxDeadlineSend` に `key()` を付けて 2 か所で同じものを使う。 |
| LOW | naming | `flushAlertNotifyBatch`(`alert_notify.go:78`、`*Org`)と `flushAlertNotifications`(`watch.go:1758`、`*watchRun`)は名前が近く、後者は前者を呼んで失敗の stderr の 1 行を足すだけ。`grep flushAlertNotif` が 2 つ返し、どちらが通知を出すのか名前から分からない。また `sendInboxItemToHuman` の 7 番目の引数に、`Escalate`(`escalate.go:219`)と `NotifyInboxItem`(`escalate.go:272`)が裸の `nil` を渡す。「batch なし = その場で通知」は関数の doc を読まないと分からない。 | 名前を `flushAlertNotifyBatchAndReport` のように、報告まで含むと分かるものに変えるか、`flushAlertNotifyBatch` の戻り値を呼び出し側が使う形(いまの `SweepInboxDeadlines`)に揃えて、watch 側の関数は `reportAlertNotifyErrors(results)` に分ける。`nil` の 2 か所には `/* no batch: notify at once */` を添える。 |

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。

## Positive notes

- 失敗の記憶の扱いを前の版と揃えた点。通知は 1 回でも、`notified` の追記の失敗は件ごとに `Failed` に入り、ack の期限の長さだけ送り直されない(D2b)。失敗が毎周期の通知に化ける形になっていない
- `lineWritten` の欄で、escalations.jsonl の行が書けなかった件を通知には含め、`notified` は書かない形にした。前の 1 件ずつの送り(通知してから行の error を返す)と同じ順の効果を保っている
- 「黙る」側の分岐が `off` の 1 つだけで、ほかはすべて通知に倒れる。batch を flush しないまま止まる場合の戻り先(ack の期限の再送)がコードと doc とテスト(`TestSendInboxItemToHuman_AnAlertItemOfABatchNeverFlushedIsSentAtTheAckDeadline`)で一致している
- `ParseInboxWaitTypes` が、受け付ける形を 1 つ(大文字、空白なし、コンマ区切り)に決め、断るときの文に受け付ける値を出している
- `WaitInbox` の種類を可変長の引数にして、範囲外の既存の呼び出しを変えずに通した(計画 S3 の記録どおり)

観察として残すもの(指摘にはしない)。

- 通知の遅れの上限は計画の Risks の文より大きい。計画は「同じ段のほかの件の書き込みの分だけ」遅れると書くが、実際は、`checkDeadman` の for の中で、先に escalate した件の通知が、あとの pending の `leaderActedSinceAlert`(herdr と agmsg の問い合わせ、1 回 `driverCallTimeout` 既定 10 秒まで)の後ろになり、期限の掃き出しでは ALERT 以外の件の通知(1 回 `inboxDesktopNotifyTimeout` 10 秒まで)の後ろになる。どちらも有限で、期限の送りを director の確かめより先に済ませる約束は守られている。計画の文の直しは `/verify` か `/sync-docs` の判断に任せる
- 掃き出し 1 回に複数の org の ALERT が入ると、通知の文は最も古い件の org だけを出す。計画の D2 どおりで、`TestRunWatchAll_TwoOrgsEscalatingInOneCycleAreEachLeftToTheDirectorUntilTheirAckDeadline` がこの形を固定している

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | - | - | - |

_(このレビューは `docs/tech-debt/README.md` を編集していない。L1 から L4 は小さく直せる。直さずに `/pr` の known gaps に回す場合は、L1 と L3 を 1 行にまとめて register に足すよう `/sync-docs` に頼む。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4)
- Follow-ups: L1 はテンプレートと `config.go` の comment の追記だけで直る。コメントの変更でもパイプラインは全体の再実行になる(cycle 2、cap 2 の最後の run)ので、ほかに直す理由ができたときにまとめて直すのが無駄が少ない。L2 から L4 は任意。直さない場合は、PR 本文の known gaps に L1 の 1 行を書く
