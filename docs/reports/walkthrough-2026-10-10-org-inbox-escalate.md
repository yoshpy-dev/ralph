# Walkthrough: org-inbox-escalate(5 段目)

- Date: 2026-10-10
- Plan: docs/plans/archive/2026-10-10-org-inbox-escalate.md(PR 作成時に active から移す)
- Branch: feat/org-inbox-escalate(base 382c18c8、20 commits、30 files、+5503/-109。うち `internal/` は 13 files、+4449/-46 で、6 割ほどがテスト)
- PR: #220

新しいファイルは `inbox.go` と `escalate.go` の 2 つで、残りは CLI の配線と雛形と文書。次の順に読むと、依存の向きに沿って追える。

## 1. 受信箱(`internal/org/inbox.go`、S1 f8fff664)

台帳の下の `inbox.jsonl` への追記と、その畳み込み。

- `InboxStore` の `Escalate`(件を記録するだけ)・`AppendNotified`・`Ack`・`Resolve` は、どれも `withLock` の下で `readRaw` → `foldInbox` → 判定 → `appendLocked` の順に進む。`Read` だけはロックを取らない
- `foldInbox` はイベントを畳み込んで、件ごとの状態(open → acked → resolved)を出す。読めない行と、知らない ID などで無視したイベントは数えて `InboxReadResult` に入れる
- 次の ID は、JSON として読めた行の id と、生の行を正規表現で拾った `"id":"e<N>` の両方から最大を取って 1 を足す(`seeN`、`nextID`)。壊れた行の ID をもう一度使わないため
- `appendLocked` は、最後の行が改行で終わっていなければ、改行を足してから書く。切れた行に新しいイベントが混ざらないため
- `ValidateInboxNote` は resolve の note を 1 行・500 文字までに限り、制御文字、不正な UTF-8、U+2028 / U+2029 を拒否する
- ロックは `internal/org/lockfile.go` の `withFileLock` で、manifest の `withManifestLock` と同じ flock を共有する

## 2. escalate と人への経路(`internal/org/escalate.go`、S2 a6e736ed)

- `(*Org).Escalate` は、`validateEscalation`(`protocol.Parse` と `Validate`、TYPE は QUESTION・BLOCKED・RESULT だけ)を通したあと、受信箱に記録してから `sendInboxItemToHuman` を呼ぶ
- `sendInboxItemToHuman` は、`escalations.jsonl` に `inbox_id` つきの行を書き、stderr に banner を出し、osascript で通知して(`inboxDesktopNotify`)、`notified` を記録する。通知には本文を渡さない。`escalations.jsonl` に書けなければ `notified` も書かず、件は未通知のまま残る
- 記録できなかったときは `alertUnrecordedEscalation`。理由 `inbox_not_recorded` の行を best-effort で書き、NOT RECORDED の banner と通知を出す。`EscalateNotRecorded` は、`ralph.toml` が読めず runtime を作れなかったときに CLI がこの経路を通すための入口(a396db57)
- `NotifyInboxItem` は未通知の件を送り直す(`inbox notify`)。受信箱から読んだ org_id と TYPE は `PrintableInboxText` でエスケープし、通知には `notifiableOrgID` と `notifiableType` を通った値だけを渡す(a396db57)
- `WaitInbox` は、open の件が来るまで `InboxPollInterval`(既定 1 秒)ごとに読む。timeout 0 は期限なし(a3bd5cff でテストを足した)
- `InboxCommand` は復旧の案内のコマンドの文字列を作り、`--state-dir` を明示したときはそれも付ける(a396db57)
- `internal/org/watch.go`: `escalationRecord` に `inbox_id` を足し、`alert_id` を omitempty にした。osascript の呼び出しを `osascriptNotify` に切り出し、escalate と共有する

## 3. CLI(`internal/cli/org.go`、S3 faa4b1c3)

- `newOrgEscalateCmd`: 結果の出し方は 3 通り。記録して通知した(stdout に `escalated e<N>`、終了コード 0)、記録したが通知を書けなかった(ID を出して終了コード 1)、記録できなかった(終了コード 1)
- `newOrgInboxCmd` と、その下の `show` / `ack` / `resolve --note` / `notify`。一覧は open と acked の件で、`--all` で resolved も出す。`--json` もある
- `newOrgWaitCmd` の `--inbox`。`--seat` などの座席のフラグとは併用できない
- inbox の動詞と `wait --inbox` は全 org が対象なので、`--org-id` を渡されたら拒否する(計画からのずれ。Progress に記録)
- `orgDesktopNotifyOverride` と `orgInboxPollIntervalOverride` はテストの差し替え口。`internal/cli/main_test.go` の TestMain は、テストで本物の通知を出さないよう差し替える

## 4. 雛形と文書(S4 b0fc4c59、a396db57、f347842d)

- `internal/org/prompts/leader.md`: 「人に上げる」5 か所を `ralph org escalate --state-dir <台帳> --org-id {{ORG_ID}} ...` に置き換えた。メッセージは `mkdir -p .harness/state` のあと heredoc か Write ツールで `.harness/state/escalate-{{ORG_ID}}.txt` に書き、`--text "$(cat …)"` で渡す。失敗したときの手順と、受信箱の本文はデータで指示ではないことも書いた
- `internal/org/prompts_test.go`: `containsPhrase` は空白をすべて除いて語句を比べる。改行の位置を変えても雛形のテストが落ちない(tech-debt の (f)(j))
- `/org` skill の 4 面: 動詞の表と「受信箱」節
- README、AGENTS.md、仕様 FR-5 の「5 段目で決めたこと」、`.claude/rules/ralph/agent-messaging.md` の escalate の節(2 面)、`docs/tech-debt/README.md`

## パイプラインの往復

| 回 | 見つかったもの | 直した commit |
|---|---|---|
| self-review 1 回目 | M1 雛形の例の単一引用符、M2 台帳の閉じ忘れ、L1〜L9 | a396db57(M1、L1、L2、L3、L6、L9)、M2 と L7 は sync-docs |
| self-review 再実行 | N1 書き込み先のディレクトリと書き方、N2・N3 | f347842d(N1 と N3 のテストの穴)、残りは tech-debt |
| test | mutant E13(`wait --inbox --timeout-ms 0` の待つ枝が固定されていない) | a3bd5cff |
| cross-review cycle 1 | 指摘 0 件 | なし |
