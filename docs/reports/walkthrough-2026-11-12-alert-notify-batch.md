# Walkthrough: 見張りの ALERT の通知を送る段ごとにまとめ、`wait --inbox` を種類で絞る(alert-notify-batch)

- 計画: `docs/plans/archive/2026-11-12-alert-notify-batch.md`(PR の作成時に archive へ移す)
- issue: #307 の後半。前半は PR #314。この PR で issue の受け入れ条件 4 を満たす
- 差分: `git diff 94e61c38...HEAD`(43 ファイル、約 +2,360 / −110。うちテストを除く Go は約 +420、Go のテストは約 +1,200)

## 1. 通知のまとめ(読む価値が一番高い)

- `internal/org/alert_notify.go`: まとめの器 `alertNotifyBatch` と、段の終わりに 1 回送る `flushAlertNotifyBatch`。
  - `[org.watchdog].alert_notify = "off"` なら送らず、各件の `notified` の `osascript` に `off` を残す。
  - そうでなければ 1 回だけ送る。1 件ならその件の今までの文、2 件以上なら「N ALERT items; the oldest is e<N> of org <org>」(最も古い件は e の番号が最小の件。org の値は今と同じく識別子だけを出し、本文は出さない)。
  - そのあと各件に、その件の理由で `notified` を足し、追記の失敗を件ごとに返す。
- `internal/org/escalate.go`: `sendInboxItemToHuman` が最後の引数で器を受け取る。器があり TYPE が ALERT のときは、`escalations.jsonl` の行と banner をすぐ出し、通知と `notified` は器に任せる。器が無い道(leader の件、`ralph org inbox notify`、`Escalate`)は今までと同じ。
- `internal/org/deadline.go`: 期限の掃き出しが器を作り、掃き出しの終わりに送る。`notified` の追記に失敗した件は、今までの 1 件ずつの送りの失敗と同じく記憶に残し、ack の期限の長さのあいだ送り直さない(計画の助言の指摘 2、D2b)。掃き出しは見張りの周期の 1 段目のままなので、director の確かめや座席の見回りより先に送り終える(指摘 1、D2)。
- `internal/org/watch.go`: org ごとの `checkDeadman` が器を作り、その段の終わりに送る(`flushDeadmanAlertNotifications`)。周期の初めの突き合わせ(#307a の S12)と、受信箱に書けないときの予備の道は器を通らない。
- テスト: `internal/org/alert_notify_test.go`(掃き出しのまとめ、`off`、追記の失敗と記憶、1 件の文、止まった見張り、手で送る通知)、`deadline_test.go`・`watch_test.go`・`watch_all_test.go`(deadman の段、2 つの org、周期の順)。既存の 2 本は件ごとの通知の数を固定していたので、まとめた 1 回に期待を直した。

## 2. `wait --inbox --types`

- `internal/org/escalate.go` の `openInboxItems(types)` と `WaitInbox(timeout, types...)`、`internal/org/director_wait.go` の `DirectorWaitParams.Types`。2 つの待ちは同じ絞り込みを通るので、返す件、`--director` の終了コード 4 の「open の件が無い」、印の件数が同じ種類で数えられる。`--types` で時間切れになったときの印の文は種類を名指す(stderr の文は skill が一致を見るので変えない)。
- `org.ParseInboxWaitTypes`: QUESTION、BLOCKED、RESULT、ALERT の大文字だけを受け付け、待つ前に断る。
- `internal/cli/org.go` と `internal/cli/org_wait_director.go`: `--types` のフラグ(`--inbox` のときだけ)と help。

## 3. 設定

- `internal/config/config.go`: `OrgWatchdogConfig.AlertNotify`(`batch` が既定、`off`。ほかは断る)。`templates/base/ralph.toml` に既定値と、`alert_notify` を読まない 4 つの送り(leader の件、`inbox notify`、周期の初めの再送、予備の道)を書いた。

## 4. 手順書と文書

- director skill(4 つの写し): 件を QUESTION・BLOCKED → RESULT → ALERT の順に処理する、ALERT の件の `recurred` の回数の読み方、`--types` の使いどころ(既定の待ちを続ける理由: 見ない ALERT は ack の期限で人に送られる、D5)。
- org skill(4 つの写し)、`ralph org watch --help`、README、`docs/quality/quality-gates.md`、仕様 FR-5・FR-6 の注、tech-debt(#307a の「Eleven gaps」の (4) を RESOLVED にし、引き継ぎの注と、この PR のテストの穴の行を足した)。

## 見どころ

- 知らせが消える向きの変化は無い。まとめの送りの前に見張りが止まった件は `notified` が残らず、次の見張りが送る(掃き出しの件は次の最初の掃き出し、deadman の段の件は ack の期限か、記録と同時に送る見張りなら周期の初めの突き合わせ)。
- テストの穴: 掃き出しの通知が director の確かめより前に出ることを固定するテストが無い(tester が 40 行ほどの probe で変異を捕まえられることを確かめ、tech-debt に記録した)。
- 実機の macOS で通知の回数が減ることは確かめていない。
