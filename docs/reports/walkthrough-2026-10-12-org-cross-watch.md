# Walkthrough: org-cross-watch(6 段目の前半)

- Date: 2026-10-12
- Plan: docs/plans/archive/2026-10-11-org-cross-watch.md(PR 作成時に active から移す)
- Branch: feat/org-cross-watch(base 577ebeaf。PR 作成の時点で 39 commits、47 files、+9117/-1066。うち `internal/` は 26 files、+6371/-1022 で、半分を超える行がテスト)
- PR: #222

新しいファイルは `deadline.go`(受信箱の期限)と `watchalive.go`(ロックと生存の印と止める頼み)の 2 つで、横断の見張りの繰り返しは `watch.go` に足した。次の順に読むと、依存の向きに沿って追える。

## 1. 設定(`internal/config/config.go`、S1 9cabdeb4)

- `OrgDirectorConfig` の `AckMinutes`(既定 10)と `ResolveMinutes`(既定 60)。`[org.director]` の節がなければ `Default()` の値のまま、1 未満は検証エラー
- `templates/base/ralph.toml` と `scripts/ralph-config.sh`(と template の写し)を揃え、`defaults_sync_test.go` で 3 面の一致を固定した

## 2. 受信箱の期限(`internal/org/deadline.go`、S2 b50d7e03)

- `InboxDeadlineDue` は 1 件と時刻から、送る理由(`inbox_ack_deadline` / `inbox_resolve_deadline` / なし)を返す純粋な関数。resolve を先に見るので、両方が切れたときは resolve だけになる。ack は open で `notified` が 1 つもない件、resolve は resolved でなく理由 `inbox_resolve_deadline` の `notified` がない件(`inboxNotifiedFor`)に効く
- `SweepInboxDeadlines` は `inbox.jsonl` を読み、期限の切れた件ごとに `sendInboxItemToHuman` を呼ぶ。件の状態は変えない
- `InboxDeadlineMemory` は 1 つのプロセスの中の覚え。送れなかった件は `ack_minutes` に 1 回までしか送り直さず(`holdsBack`、`recordFailure`)、時刻の読めない件の警告は 1 回だけ出す(`warnOnce`)。起動し直すと覚えは空になり、期限は `inbox.jsonl` から計算し直す
- `internal/org/escalate.go`: `sendInboxItemToHuman` が理由を受け取るようにし、banner と通知の文言を理由ごとに作る(`inboxHumanTextFor`、`inboxDesktopNotify(title, message)`)。`PrintableInboxText` は `inbox.go` に移した(S0)

## 3. ロック、生存の印、止める頼み(`internal/org/watchalive.go`、`internal/org/lockfile.go`、S3a 996e5c48)

- `AcquireWatchLock` は台帳の下の `watch.lock` を flock で持つ(`acquireHeldFileLock`)。取れなければ、印の pid を添えた `ErrWatchRunning` を返す
- 生存の印 `WatchAlive`(state・pid・started_at・ts・interval_seconds・stopped_at)は `writeJSONFileAtomic`(一時ファイルと rename)で書く
- `ProbeWatch` はロックを LOCK_SH|LOCK_NB で一瞬取って持たれているかを見て(`fileLockHeld`)、印と合わせて `running` / `stale` / `not_running` を返す。確かめられないロックは持たれているとみなす。窓は `watchAliveWindow`(`deadman_minutes` を 1 未満なら 10 に読み替え、印の間隔の 2 倍と比べて大きいほう)
- 止める頼み `watch-stop.json` は `WriteWatchStopRequest` / `ReadWatchStopRequest` / `RemoveWatchStopRequest`。`AppliesTo` は、見張りの `started_at` より後に書かれた頼みかを見る。起動のときは `takeWatchStopRequest` が `.watch-stop.json.taken` への rename で取り出す(cross-review cycle 1 #2 の直し、49d8d605)

## 4. 全 org の繰り返し(`internal/org/watch.go`、S3a 996e5c48)

- `RunWatchAll` はロックを取り、印を書き、`takeStopRequestAtStart` で前の見張りに宛てた頼みを捨ててから、サイクルを回す
- 1 サイクル(`cycle`): `SweepInboxDeadlines` → `pulse`(manifest で走っている org ごとに `pulseOrg`。中身は今までの 1 org のサイクル `evaluateCycle`)→ `writeMark`。1 つの org の失敗は stderr に 1 行書いてその org だけ飛ばす
- herdr・agmsg・`git status` の呼び出しは `callWithTimeout` で 1 回 `DriverCallTimeout`(10 秒)に打ち切る。ALERT の送信は `SendWatchdogAlertBounded`
- 終わり方(`finish`): `drainBeforeStop` が、ロックを持ち印が running のまま `WatchHooks.BeforeStop` を呼び、その間 interval ごとに印を書き直す goroutine を回して、終わりを待つ。そのあと `stopAndRelease` が stopped の印を書き、この見張りに宛てた頼みを消し、ロックを放す。BeforeStop が panic しても `stopAndRelease` は走る(cross-review cycle 1 #1 の直し、49d8d605)

## 5. CLI(`internal/cli/org_watch.go`、S3b 33f6383b)

- `checkOrgWatchFlags`: `--all` が必須で、`--org-id` は断る。`--stop` と `--once`、`--detach` と `--once` は併用できない
- `runOrgWatchAll`: SIGINT と SIGTERM で ctx を取り消し、そのあとは既定の動きに戻す(2 回目ですぐ終わる)。`newWatchdogHooks` の on-demand の watcher は `BeforeStop` で待つ。`--once` でなければ、待つ前に呼び出しを取り消す
- `runOrgWatchDetach`: `ProbeWatch` が `running` なら何もせず 0、`stale` なら起動せず 1。`not_running` なら `launchOrgWatch`(setsid、stdin は /dev/null、出力は `watch.log` に追記)で子を起動し、子の印が running になるまで 10 秒待つ。待ちきれなければ子を止めて回収してから 1。テストは `orgWatchLaunch` の差し替え口で、TestMain は本物の起動を panic にする
- `runOrgWatchStop`: シグナルは送らない。頼みを書き、`waitOrgWatchReleased` でロックが放されるまで最大 30 秒待つ。間に合わなければ pid と kill の前に確かめることを示して 1

## 6. `ralph status`(`internal/cli/status.go`、S4 04f9237b)

- `orgWatchLine` が `org watch:` の行を作る(`mark updated <ts>` は `orgWatchMarkDetails`)。`buildStatusOrgWatchJSON` が `--json` の `org_watch`(state、pid、started_at、updated_at、stopped_at、mark_state)を作る
- `statusDeadmanMinutes` は `./ralph.toml` の `deadman_minutes` を読む(`--config` の見張りとずれうる点は tech-debt 201 行)

## 7. 文書(S5 505c58c1、fb52a143、sync-docs 04bcb342・0d161a80・a8597e3f)

- `/org` skill の 4 面: `watch` の行と長時間運用の節
- README、AGENTS.md、仕様 FR-6 の「6 段目の前半で決めたこと」、`docs/quality/quality-gates.md`(root と template)
- `docs/tech-debt/README.md`: この段で閉じた (c)〜(h) と、新しい 201〜206 行

## パイプラインの往復

| 回 | 見つかったもの | 直した commit |
|---|---|---|
| self-review 1 回目 | M1 `deadman_minutes = 0` で `--detach` が健全な子を止める、L1〜L7 | 1b64ae2b、188a640d(L5 は tech-debt) |
| self-review 再実行 | N1〜N5 | 94d5248c(N1・N4・N5)、N2・N3 は tech-debt |
| test cycle 1 | 生き残った mutant 5 件(P2・X3・L6・A6・S6) | 4aaef4ed(テストだけ) |
| cross-review cycle 1 | #1 watcher の待ちの前にロックを放す、#2 起動のときの頼みの削除の競合 | 49d8d605 |
| self-review cycle 2 | C2-1〜C2-5 | d2a750ec(C2-1〜C2-4)、C2-5 は sync-docs の a8597e3f |
| test cycle 2 | 生き残った mutant 2 件(B2b・R5) | 2ccdd11b(テストだけ) |
| cross-review cycle 2 | 指摘 0 件 | なし |
