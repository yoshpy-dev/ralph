# Verify report: org-cross-watch

- Date: 2026-10-12(cycle 2)
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み。`scripts/plan-visual.sh digest` の値 d0a6f2735552 は `- Approved:` 行と一致する)
- Verifier: verifier subagent (Claude)。パイプライン 2 回目(cycle 2、上限 2)。cross-review cycle 1(triage 7a7e4ea7)の ACTION_REQUIRED #1・#2 の直しと、self-review cycle 2 の直しのあとの実行。cycle 1 の結果は末尾の「付録」に残してある
- Scope: 仕様適合と静的解析。ブランチ全体は `git diff 577ebeaf...HEAD`(HEAD d2a750ec)。主に見たのは cycle 1 の verify(a411bb57)より後のコミットで、`git diff a411bb57..HEAD` は 21 ファイル、+1,957/-107。コードを変えたのは 4aaef4ed(テストだけ)、49d8d605(cross-review #1・#2 の直し)、d2a750ec(self-review cycle 2 の C2-1〜C2-4 の直し)の 3 つ。d2a750ec は self-review を通していないので全文を読んだ。テストは走らせていない(/test の担当)
- Evidence: `docs/evidence/verify-2026-10-11-org-cross-watch.log`(gitignore の対象で、コミットしない。cycle 2 の出力は末尾に足した)

## Latest run: cycle 2(HEAD d2a750ec)

### Spec compliance

a411bb57 より後にコードが変わったのは 4 ファイルだけである。`internal/org/watch.go`(`WatchHooks`、`RunWatchAll` の doc、`watchAllRun`、起動のときの止める頼み、終わりの順)、`internal/org/watchalive.go`(`takeWatchStopRequest` と doc)、`internal/cli/org_watch.go`(`runOrgWatchAll` の BeforeStop、`--stop` の時間切れの文)、`internal/cli/status.go`(表示の言葉と `--json` の key)。設定、`deadline.go`、`escalate.go`、`inbox.go`、`spawn.go`、`verbs.go`、`ProbeWatch`、`checkOrgWatchFlags`、`runOrgWatchDetach` は変わっていない(`git diff a411bb57..HEAD` の hunk で確かめた)。そのため AC1〜AC8、AC11、AC12、AC12b、AC16 は付録の cycle 1 の根拠がそのまま使える。下の表の行番号は HEAD のもの。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `[org.director]` の既定と検証、3 面の一致 | 満たす(変更なし) | `internal/config/`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` に a411bb57 以降の変更はない |
| AC2 期限の判定表 | 満たす(変更なし) | `internal/org/deadline.go` とそのテストに変更はない |
| AC3 人への経路と状態の不変 | 満たす(変更なし) | `deadline.go` と `escalate.go` に変更はない |
| AC4 書けない台帳での送り直しの間隔 | 満たす(変更なし) | 同上 |
| AC5 起動し直したときの計算し直し | 満たす(変更なし) | 1 サイクル(`cycle`、`internal/org/watch.go:607-617`)は変わっていない |
| AC6 escalate はすぐ送り、二度目は resolve の期限だけ | 満たす(変更なし) | `escalate.go` に変更はない |
| AC7 全 org を 1 プロセスで見る、disband した org は見ない | 満たす(変更なし) | `cycle`・`pulse`・`pulseOrg`(`watch.go:607-666`)は変わっていない |
| AC7b 1 つの org の呼び出しが返らない、状態ファイルが書けない | 満たす(変更なし) | 呼び出しの打ち切り(`evaluateCycle`、`callWithTimeout`)は変わっていない。49d8d605 が足した後始末の間はサイクルを回さないので、外部の呼び出しは増えない |
| AC8 `--org-id` と `--all` なしの拒否 | 満たす(変更なし) | `checkOrgWatchFlags`(`internal/cli/org_watch.go:106`)は変わっていない |
| AC9 2 つ目の見張りの拒否 | 満たす | ロックを持つ期間が延びた。`RunWatchAll` はロックを取ると `defer w.finish(lock)` を登録し(`watch.go:520-525`)、`finish`(`:751-754`)は `stopAndRelease` を defer してから `drainBeforeStop`(`:762-785`)で `hooks.BeforeStop` を呼ぶ。CLI の BeforeStop は on-demand の watcher を待つ(`org_watch.go:188-193`。`--once` でなければ先に取り消す)。そのため watcher の後始末の間も 2 つ目の見張りは断られる。テストは `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning`(`internal/org/watch_all_test.go:1122`。終わり方 3 通りで、2 つ目の `AcquireWatchLock` と `RunWatchAll` が `ErrWatchRunning` になり、断られた側は BeforeStop を呼ばない)と CLI の `TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`(`internal/cli/org_watch_test.go:783`)。kill のあとの解放は /test が実機で確かめた(`docs/evidence/org-cross-watch-live-2026-10-11.md`)。そのビルドは dd63a414 で 49d8d605 より前だが、kill の解放はカーネルの flock の性質で、49d8d605 はその経路を変えていない |
| AC10 印の書き込みと SIGINT・SIGTERM | 満たす | ロックの直後(`watch.go:527`)と毎サイクル(`cycle` の `:616`)に `writeMark` を呼ぶのは同じ。49d8d605 から、BeforeStop の間も interval ごとに running を書き直す(`:762-785`)。SIGINT・SIGTERM で ctx が取り消されると、CLI の BeforeStop が watcher を取り消して待ち、そのあと `stopAndRelease`(`:793`)が stopped と stopped_at を書いてロックを放す。印の `ts` の意味は d2a750ec で「running の印を最後に書いた時刻」と doc に書かれた(`internal/org/watchalive.go:72-76`)。AC10 の文(ロックの直後と毎サイクルに書く)とは矛盾しない。BeforeStop が panic したときは `TestRunWatchAll_APanicInBeforeStopStillWritesTheStoppedMarkAndReleasesTheLock`(`watch_all_test.go:1219`)、stopped と解放の順は `TestRunWatchAll_WritesTheStoppedMarkBeforeReleasingTheLock`(`:1066`、4aaef4ed) |
| AC11 判定関数の 3 状態 | 満たす(変更なし。窓は cycle 1 のずれのまま) | `ProbeWatch`(`watchalive.go:186-201`)と `watchAliveWindow`(`:204-209`)は変わっていない |
| AC12 `--detach` | 満たす(変更なし) | `runOrgWatchDetach`(`org_watch.go:324`)と `orgWatchChildArgs`(`:248`)は変わっていない |
| AC12b 待ちきれなかった子の回収 | 満たす(変更なし) | 同上。待ちきれなかった子は SIGTERM で止める。本物の子は SIGTERM で ctx が取り消され、watcher を取り消してから stopped を書くので、後始末が回収を遅らせることはない |
| AC13 `--stop` | 満たす | `runOrgWatchStop`(`org_watch.go:374-396`)はシグナルを送らない。後始末の間はロックが持たれたままなので、`--stop` は watcher の後始末が終わるのを待ってから「stopped」と答える(直す前は、この場面で「not running」とすぐ答えていた)。`--once` の見張りは watcher を取り消さないので待ちは最大でおよそ 70 秒になり、30 秒を超えると終了コード 1 になる。そのときの文は d2a750ec で「its current cycle, or its wait for an in-flight on-demand watcher call, ends」に直った(`:394`)。`TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal`(`org_watch_test.go:551`)がこの文を見る(`:581`)。止まる場合は `TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`(`:783`)が、後始末の間に `--stop` が頼みを書いて待ち、watcher が終わってから `stopped ralph org watch --all (pid N)` を出すことを見る |
| AC13b 古い頼みに従わない、古い pid にシグナルを送らない | 満たす | 起動のときの頼みは `takeWatchStopRequest`(`watchalive.go:231-241`)が `.watch-stop.json.taken`(`:36`)への rename で取り出してから読む。`takeStopRequestAtStart`(`watch.go:703-723`)は、`started_at` より後の頼みを元に戻して `stopTaken` で覚え、それ以外(前の見張りに宛てた頼み、読めない頼み)は取り出したファイルだけを消す。取り出しのあとに書かれた頼みは新しいファイルなので消えない。`WriteWatchStopRequest` は rename で書く(`writeJSONFileAtomic`、`watchalive.go:278`)ので、取り出しが拾うのは頼みの全体である。テストは既存の表 `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`(`watch_all_test.go:863`。読めない頼みのケース、`.taken` が残らないこと、stderr が空であること)と、新しい `_AStopRequestWrittenWhileTheStartTakesAnOldOneIsKept`(`:931`)、`_AStopRequestThatCannotBePutBackIsStillFollowed`(`:975`)、`TestTakeWatchStopRequest_MovesTheRequestSoALaterOneIsANewFile`(`internal/org/watchalive_test.go:384`)。古い pid にシグナルを送らない点は cycle 1 と同じ(`org_watch_test.go:551`) |
| AC14 `ralph status` の行と `org_watch` | 満たす | `org watch:` の行(`internal/cli/status.go:165-187`)と `--json` の `org_watch`(`:404-424`)。d2a750ec で行の言葉が `last cycle <ts>` から `mark updated <ts>` に、`org_watch` の key が `last_cycle_ts` から `updated_at` に変わった(`:408`)。この key はこのブランチで足したもので、まだ配っていない(計画の Progress `:218` に記録がある)。repo の中に `org_watch.last_cycle_ts` を読むものはない。残る `last_cycle_ts` は org ごとの `watch-status-<org>.json` の心拍で、別のものである。`TestStatusCmd_OrgWatchLineAndJSONPerWatchState`(`internal/cli/status_test.go:525`)など 4 本の期待値も同じコミットで合わせた |
| AC15 spawn・send・start・stop・disband・escalate は変わらない | 満たす(静的な部分) | a411bb57 以降に変わったテストファイルは `org_watch_test.go`、`status_test.go`、`watch_all_test.go`、`watchalive_test.go` の 4 本で、どれもこの PR の watch と status のテストである。spawn・send・start・stop・disband・escalate のコードとテストは変わっていない。消えた 14 行は、status の期待値の言葉 10 行、表のテストの doc 3 行、頼みを書く 1 行(読めない頼みのケースを足すために `else if` にした)だけ |
| AC16 tech-debt (c)〜(h) は動作を変えない | 満たす(変更なし) | (c)〜(h) の箇所に a411bb57 以降の変更はない |
| AC17 文書と sync の検査 | 一部 | `scripts/check-sync.sh`(DRIFTED 0)と `scripts/check-skill-sync.sh`(13 skill)は通る。cycle 1 の D1・D2(窓の規則)は 04bcb342 で直った。49d8d605 と d2a750ec の変更に文書が追いついていない箇所が残る(下の「Documentation drift」の D1〜D6。/sync-docs で直す) |

仕様 FR-6 との対応は、付録の cycle 1 の表から変わらない。49d8d605 と d2a750ec は、FR-6 の項目のうちどれをこの PR が受け持つかを変えていない。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | pass(rc 0) | `scripts/ralph-config.sh` が言語に分類されないため、scope は full に落ちた(`Language scope: full fallback (unclassified:scripts/ralph-config.sh)`)。shellcheck と hook の `sh -n`、guard の awk の構文、settings の `jq -e`、Codex の 3 つの guard、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の参照を通る。Go は `gofmt: ok`、`go vet` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし。branch の secret scan(577ebeaf..d2a750ec、origin/main 比)は clean |
| `HARNESS_VERIFY_MODE=static RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` | pass(rc 0) | 計画の Verify plan が名指しする最後の full gate。結果は上と同じ |
| `./scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 167、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 13 skill が lock-step。`/org` の 4 面はバイト一致(`cmp`) |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | d0a6f2735552 |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1 `/org` skill の 4 面 `:176`(動詞リファレンスの watch の行) | いいえ | `--stop` の説明「見張りがそのサイクルを終えて止まるまで最大 30 秒待つ」は、49d8d605 の終わりの順を書いていない。今の見張りは、最後のサイクルのあと on-demand の watcher の後始末が終わるまでロックを持ち、印も running のまま書き直す(`watch.go:751-785`)。後始末は、`--once` でなければ watcher を取り消してすぐ終わり、`--once` なら呼び出しが終わるまで(最大でおよそ 70 秒)続く。「そのサイクルと、動いている watcher の呼び出しの後始末を終えて止まるまで」の形にする |
| D2 `/org` skill の 4 面 `:775-777`(stale の定義) | いいえ | 「古いとは、最後のサイクルの時刻が…」。d2a750ec で印の `ts` は「running の印を最後に書いた時刻」になった(後始末の間も interval ごとに書き直す。`watchalive.go:72-76`)。`ralph status` の表示は `mark updated <ts>`、`--json` は `org_watch.updated_at` になったので、「印を最後に書いた時刻(`ralph status` の `mark updated`)が」にする |
| D3 `/org` skill の 4 面 `:780-781`(`--stop` の時間切れ) | いいえ | 「30 秒で止まらなくても頼みは残り、見張りはそのサイクルの終わりに止まる」。CLI のエラー文(`org_watch.go:394`)は d2a750ec で「そのサイクルか、動いている on-demand watcher の呼び出しの待ちが終わったときに止まる」になったので、skill も合わせる(self-review C2-4 の skill の分) |
| D4 仕様 FR-6 の注 `docs/specs/2026-10-07-org-multi-org-director.md:63` | いいえ | 2 か所が古い。(1)「生存の印 … を最初のサイクルの前と毎サイクルの終わりに書く」に、止まる前の watcher の後始末の間も interval ごとに書き直すことが入っていない。(2)「`running`(ロックあり、印が running で最後のサイクルの時刻 `ts` が窓の内)」の「最後のサイクルの時刻」は、今は「印を最後に書いた時刻」である。止まる前の後始末の間もロックを持ち running のままであること(cross-review cycle 1 #1 の直し)を 1 文足すとよい |
| D5 `docs/tech-debt/README.md:204`「Small leftovers of org-cross-watch」 | いいえ | (d) の括弧「unless `--once`, the watch cancels the hooks' context once `RunWatchAll` returns」が古い。今は `WatchHooks.BeforeStop` の中で、stopped の印とロックの解放より前に取り消す(`org_watch.go:188-193`)。(e) の Trigger「the next change to `runOrgWatchAll`」は 49d8d605 で発火したが、banner の位置は変わっていない。Trigger を書き換えるか、「carried over by feat/org-cross-watch」と書く。(e) が勧める仕組み(`WatchAllParams` の callback)は、今なら `WatchHooks` の `BeforeStop` と並べて置ける(self-review C2-5) |
| D6 `docs/tech-debt/README.md:202`「Stopping the cross-org watch waits for the current cycle while herdr hangs」 | 一部(任意) | 「a SIGINT, a SIGTERM, or a `--stop` request takes effect only when the cycle ends」と「which the watch follows when that cycle ends」に、`--once` の見張りへの `--stop` では watcher の呼び出しの待ちも加わることが入っていない(`--once` の BeforeStop は watcher を取り消さない)。行の主題は herdr が止まる場合なので、CLI の文に合わせて 1 句足す程度でよい。Trigger は発火していない |
| D7 `docs/evidence/org-cross-watch-live-2026-10-11.md`(`:29`、`:31`、`:199`、`:211`、`:238`、`:250`、`:286`、`:345`、`:434`、`:469`、`:504`、`:539`、`:584`、`:587`) | 記録としては正しい(任意) | dd63a414 のビルドの出力なので `last cycle …` と `"last_cycle_ts"` が残る。そのビルドの記録としては正しいので、引用は書き換えない。d2a750ec で `mark updated` と `updated_at` に変わったことを、末尾に 1 行の注として足すとよい |
| README `:124`、`:125`、`:253`、AGENTS.md `:90`、`docs/quality/quality-gates.md` と写し、`.claude/rules/ralph/agent-messaging.md` と写し | はい | `last cycle` も終わりの順も書いていない。`ralph status` については `org watch:` の行と `org_watch` の名前だけを書く |
| コードの doc(`WatchHooks`、`BeforeStop`、`WatchAlive.TS`、`RunWatchAll`、`finish`・`drainBeforeStop`・`stopAndRelease`、`runOrgWatchAll`、`newWatchdogHooks`、`ralph status --help`) | はい | d2a750ec で合った。`ralph org watch --help`(`org_watch.go:46-64`)は「rewrites its liveness mark every cycle」「SIGINT or SIGTERM ends it after the current cycle」と書き、後始末には触れないが、誤りではない |
| cross-review を引くコメント 8 か所 | はい(条件つき) | d2a750ec で「cross-review cycle 1 #1, docs/reports/cross-review-triage-org-cross-watch.md」の形になり、cycle を書いていないものは残っていない(`grep -rn 'ross-review #' internal/ cmd/` で cycle のない行は 0)。ただし指す先が読めるのは、この cycle の `/cross-review` が triage の cycle 1 の節を残す場合だけである。古い triage(たとえば `docs/reports/cross-review-triage-org-runtime-watchdog.md`)は `# Cycle 2` の節を後ろに足して前の cycle を残している。self-review C2-1 が勧めた triage の SHA(7a7e4ea7)はコメントに入っていない |
| 計画 | 記録の遅れ | Progress `:218` は d2a750ec を「そのあとのコミット」とだけ書く(コミットは自分の SHA を書けない)。次の記録で SHA を足す。AC の checkbox はすべて未チェックで、この repo では /pr の前に付ける。Scope `:62` の「最後のサイクルの時刻」は digest を保つため本文を変えず、Progress `:218` にずれを書いてある |

/sync-docs に渡すもの: D1〜D5。D6 と D7 は任意。/cross-review には、triage を書き直すときに cycle 1 の節を残すこと(コードのコメントが cycle 1 の #1・#2 を指すため)。

### Observational checks

#### 49d8d605(cross-review cycle 1 の #1・#2 の直し)

- #1: 終わりの順は、最後のサイクル、BeforeStop(ロックあり、印は running、interval ごとに書き直し)、stopped の印、止める頼みの削除、ロックの解放、になった。`finish` は `stopAndRelease` を defer するので、BeforeStop が panic しても stopped の印と解放が来る。ロックを取れなかった `RunWatchAll` は `finish` を登録しないので BeforeStop を呼ばない。CLI は `RunWatchAll` のあとの待ちを消したが、ロックを取れずに返る場合はサイクルが回っておらず watcher の goroutine もないので、待ち漏れはない
- 書き直しの goroutine と元の goroutine は `w.mark` を同時に触らない。元の goroutine は BeforeStop の中にいて、`<-refresherDone` を待ってから `stopAndRelease` に進む(`watch.go:779-784`)
- #2: rename は同じディレクトリの中で行うので原子的である。`stopAndRelease` の `stopRequested` は `stopTaken` なら真を返し(`watch.go:677-683`)、`RemoveWatchStopRequest` は頼みがなければ何もしない(`watchalive.go:258-264`)
- 計画への影響: Scope の「SIGINT と SIGTERM を受けたら、そのサイクルを終えてから `stopped` と `stopped_at` を書き、ロックを放して終わる」の間に、watcher の後始末が入った。Scope は digest で固定されていて、Progress `:217` に記録がある。AC9・AC10・AC13 の文とは矛盾しない。AC9 の「1 プロセスの保証」は、watcher の後始末の間も成り立つようになった

#### d2a750ec(self-review を通していないコミット)

9 ファイル、+74/-60。全文を読んだ。

- 動きが変わるのは、利用者に見える文字列 3 つだけである。`ralph status` の `last cycle` を `mark updated` に(`status.go:183`、`:202`)、`--json` の `org_watch.last_cycle_ts` を `updated_at` に(`:408`、`:419`)、`--stop` の時間切れの文(`org_watch.go:394`)。テストの期待値は同じコミットで合わせ、`--stop` の文には assertion を 1 つ足した(`org_watch_test.go:581`)
- コメントと doc: `WatchHooks` の型の doc(`watch.go:173-176`)、`BeforeStop` が上限のある時間で返る前提(`:194-198`)、`WatchAlive.TS` の意味(`watchalive.go:72-76`)、cross-review を引く 8 か所。doc が挙げる上限(`watcherInvokeTimeout` の 60 秒、`SendWatchdogAlertBounded` の `DriverCallTimeout`)は `internal/org/watcher.go:74` と `watch.go:1053-1057` にある
- self-review cycle 2 の指摘との対応: C2-1 は cycle を書いた(triage の SHA はない。上の表)。C2-2 は TS の doc、status の言葉、`WatchHooks` の doc で直った。C2-3 は doc の一文で直った(仕組みでは打ち切らない。self-review が「doc の一文で足りる」とした案)。C2-4 は案 (1) の文で直り、skill の分が D3 に残る。C2-5 は /sync-docs の分(D5)
- 範囲の外の変更、デバッグ出力、secret らしい文字列はない(secret scan も clean)。新しい問題は見つからなかった
- `--json` の key の改名は出力の契約を変えるが、まだ配っていない key で、repo の中に読み手はない。release ノートに書く必要はない

#### そのほか

- 承認 digest は一致する(d0a6f2735552)
- `/org` skill の 4 面はバイト一致(`cmp`)

### Coverage gaps

- テストは走らせていない(/test の担当)。上の判定は読み合わせによる
- 実機の確認(`docs/evidence/org-cross-watch-live-2026-10-11.md`)は dd63a414 のビルドで、49d8d605 と d2a750ec より前である。新しい終わりの順、起動のときの取り出し、`mark updated` と `updated_at` は実機で見ていない
- `--once` の後始末が 30 秒を超えるときの `--stop` を通るテストはない。新しい文は、止まらない見張りのテスト(`org_watch_test.go:551`)が文の組み立てとして見るだけである(self-review C2-4 の Coverage gaps と同じ)
- BeforeStop が返らない場合は doc の前提だけで、仕組みでは打ち切らない(C2-3)
- 実機で kill -9 のあとに `.watch-stop.json.taken` が残る場合は確かめていない
- 足すと確度がいちばん上がる確認: HEAD をビルドし、一時の台帳で cycle 1 の実機の手順(`--detach`、`ralph status` と `--json`、`kill -9`、起動し直し、`--stop`)をもう一度回す。`org watch: running (pid N, mark updated …)` と `org_watch.updated_at` が出ること、`--stop` が `stopped` を返したあと `ralph status` が `not running (stopped at …)` になること、台帳に `.watch-stop.json.taken` が残らないことを見る。/test の cycle 2 で回せる

## Verdict

pass

- Verified: AC1〜AC16 と AC7b・AC12b・AC13b は、コードとテストの読み合わせで満たす。cross-review cycle 1 の #1(watcher の後始末の間もロックを持つ)と #2(起動のときの取り出しが新しい頼みを消さない)の直しは AC9・AC10・AC13・AC13b の文と矛盾せず、それぞれに直す前のコードで落ちるテストがある。d2a750ec は利用者に見える文字列 3 つと doc だけで、新しい問題はない。静的解析(changed scope と full scope)、`check-sync.sh`、`check-skill-sync.sh` は rc 0。仕様 FR-6 の受け持ちは cycle 1 から変わらない
- Partially verified: AC17(D1〜D5 の文書が 49d8d605 と d2a750ec に追いついていない。/sync-docs で直す)、AC15(静的に読んだ部分だけ。テストは /test)
- Not verified: テストの実行、HEAD のビルドでの実機の動き(実機の記録は dd63a414 のもの)、切り離した子の osascript の表示、herdr が止まったときのサイクルの長さ

## 付録: cycle 1 の結果(HEAD 94d5248c の時点)

cycle 1 の header(Date から Evidence まで)と本文を、そのとき書いたまま残す。見出しは 1 段下げて「(cycle 1)」を付け、判定の見出しは `## Verdict` から「判定(cycle 1)」に変えた。本文の行番号は cycle 1 の時点のものである。「Documentation drift(cycle 1)」の D1・D2 は上の cycle 2 の D1〜D7 とは別の番号で、04bcb342 で直った。

- Date: 2026-10-11
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み。`scripts/plan-visual.sh digest` の値 d0a6f2735552 は `- Approved:` 行と一致する)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)
- Scope: 仕様適合と静的解析。対象は `git diff 577ebeaf...HEAD`(HEAD 94d5248c、42 ファイル、+6,105/-1,064)。self-review の再実行のあとに入った 94d5248c も読んだ。テストは走らせていない(/test の担当)
- Evidence: `docs/evidence/verify-2026-10-11-org-cross-watch.log`(gitignore の対象で、コミットしない)

### Spec compliance(cycle 1)

コードとテストを読み合わせて判定した。テストが通るかどうかは /test で確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `[org.director]` の既定と検証、3 面の一致 | 満たす | 型は `internal/config/config.go:150-157`、既定の 10 と 60 は `:196-199`、1 未満の拒否は `:390-395`。`templates/base/ralph.toml:111-118`、`scripts/ralph-config.sh:91-97`(root と `templates/base/` はバイト一致)。lock-step は `TestDefaultsLockStep` の 2 行(`internal/config/defaults_sync_test.go:173-176`)。`TestDefault_OrgDirector`(`config_test.go:795`)、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`(`:808`)、`TestLoad_OrgDirectorRoundTrip`(`:833`)、`TestLoad_OrgDirectorRejects`(`:881`。2 キー × 0 と -1、エラー文の全文) |
| AC2 期限の判定表 | 満たす | `InboxDeadlineDue`(`internal/org/deadline.go:57-73`)は resolved なら空を返し、resolve の期限を先に見て、ack の期限は open で `notified` が 1 つもない件にだけ効かせる。`TestInboxDeadlineDue`(`deadline_test.go:116`)の 22 行の表は AC2 の 7 項目をすべて含み、境界ちょうど(期限ちょうどは未到来)、時計の逆行、acked イベントの reason、時刻の読めない件 8 通りも見る |
| AC3 人への経路と状態の不変 | 満たす | `sendInboxItemToHuman`(`internal/org/escalate.go:291-305`)が reason つきの `escalations.jsonl` 行、banner、デスクトップ通知、reason と osascript の結果つきの `notified` を書く。期限の文面は `inboxDeadlineHumanText`(`deadline.go:221-232`)で、本文を載せない。`TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman`(`deadline_test.go:186`。行のキー 5 つ、banner の全文、本文が漏れないこと、状態、イベント列)、`TestSweepInboxDeadlines_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral`(`:279`)。banner は見張りの stderr に出る(`internal/org/watch.go:583`) |
| AC4 書けない台帳での送り直しの間隔 | 満たす | `InboxDeadlineMemory`(`deadline.go:89-124`)。間隔は `ack_minutes`(`:184`、`:200`)。`TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes`(`:332`。`notified` が書かれないこと、10 分の内は HeldBack、10 分で再試行、新しい memory と nil はすぐ試す)、`_NotifiedUnwritableIsHeldBack`(`:390`)、`_HoldBackIsPerReason`(`:435`) |
| AC5 起動し直したときの計算し直し | 満たす | 掃き出しのたびに `inbox.jsonl` を読む(`deadline.go:176`)。`TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`(`:453`)、`TestRunWatchAll_SendsAnInboxItemPastItsDeadlineInTheFirstCycleOnce`(`watch_all_test.go:321`)、CLI の `TestOrgWatch_Once_SendsAnInboxItemPastItsDeadline`(`internal/cli/org_watch_test.go:245`) |
| AC6 escalate はすぐ送り、二度目は resolve の期限だけ | 満たす | escalate と inbox notify は reason `inbox_no_director` のまま(`escalate.go:147`、`:197`)。`TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline`(`deadline_test.go:481`。11・30・60 分は鳴らず、61 分で resolve の期限が 1 回、起動し直しても増えず、resolved で止まる) |
| AC7 全 org を 1 プロセスで見る、disband した org は見ない | 満たす | `RunWatchAll`(`watch.go:494-524`)。`pulse` は `RunningOrgs`(`internal/org/reserve.go:404`)の org だけを回し、走らなくなった org を `statuses` から落とす(`watch.go:595-612`)。`TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`(`watch_all_test.go:246`。状態ファイルのバイト一致と ALERT の一致。disband した org-c には状態ファイルも herdr の呼び出しもない)。比べる相手は `evaluateCycle` を直に回す helper(`:224`)で、消した `RunWatch` ではない。1 org のサイクルの回帰は `watch_test.go` の既存テストが持つ。そのテストの書き換えは ctx 引数を外す 57 行と `RunWatch` の 3 本の置き換えだけで、assertion は変わっていない |
| AC7b 1 つの org の呼び出しが返らない、状態ファイルが書けない | 満たす | 呼び出しごとに `callWithTimeout`(`internal/org/verbs.go:635-639`。Background から期限を作る)。herdr の `agent get`(`watch.go:855`)、agmsg の Join・Send・History(`:983`、`:1009`、`:1125`、`:1169`)、`git status`(`:345-348`、`exec.CommandContext`)。本番の driver も ctx を守る(`internal/org/driver/driver.go:42` の `exec.CommandContext` と `:51` の `WaitDelay` 2 秒)。テストは `TestRunWatchAll_AHungHerdrCallOfTheFirstOrgIsCutOffAndTheCycleGoesOn`(`watch_all_test.go:362`)、`_EveryHungDriverCallOfTheFirstOrgIsCutOff`(`:406`)、`_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg`(`:448`。読めない場合と書けない場合)、`_AManifestThatCannotBeReadSkipsOnlyThePulse`(`:602`)、`_AnInboxThatCannotBeSweptDoesNotStopThePulse`(`:624`)、`TestRealGitStatus_IsCutOffAtTheDriverCallTimeout`(`:893`) |
| AC8 `--org-id` と `--all` なしの拒否 | 満たす | `checkOrgWatchFlags`(`internal/cli/org_watch.go:106-128`)は state dir を解決する前に呼ばれる(`:66-68`)。エラーは終了コード 1(`cmd/ralph/main.go:31`)。`TestOrgWatch_RefusesFlagCombinationsAndWritesNothing`(`org_watch_test.go:151`。9 通り。stdout が空で、state dir が作られない) |
| AC9 2 つ目の見張りの拒否 | 満たす(kill は未確認) | `AcquireWatchLock`(`internal/org/watchalive.go:119-136`)が `ErrWatchRunning` に印の pid を添える。`TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`(`watchalive_test.go:61`)、`TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds`(`watch_all_test.go:642`)、CLI の `TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock`(`org_watch_test.go:189`)。「kill でもロックが放される」は flock の性質に頼り、テストは `Release` で模すだけで、プロセスを実際に kill してはいない |
| AC10 印の書き込みと SIGINT・SIGTERM | 満たす | ロックの直後(`watch.go:509`)、毎サイクル(`:591`)、終わり(`finish`、`:696-710`)。シグナルは `signal.NotifyContext`(`org_watch.go:146`)。`TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd`(`watch_all_test.go:695`。最初のサイクルの前の印も見る)、`TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark`(`org_watch_test.go:219`)、`TestWriteWatchAlive_ReplacesThroughARenameAndLeavesNoTemporaryFile`(`watchalive_test.go:243`) |
| AC11 判定関数の 3 状態 | 満たす(窓は計画からのずれ) | `ProbeWatch`(`watchalive.go:177-191`)と `watchAliveWindow`(`:195-200`)。`TestProbeWatch_TellsRunningStaleAndNotRunningApart`(`watchalive_test.go:141`。17 行。JSON でない印、ts の読めない印、kill された見張りの running の印、窓の境界)、`TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`(`:208`)、`TestReadWatchAlive_MissingAndCorrupt`(`:278`)。窓の規則は AC11 の「`deadman_minutes` 以内」と違う(下の「計画からのずれ」1) |
| AC12 `--detach` | 満たす | `runOrgWatchDetach`(`org_watch.go:315-354`)、子の引数は `orgWatchChildArgs`(`:239-248`)、起動の差し替えは `orgWatchLaunch`(`:213`)で、TestMain は起動すると panic する偽物を入れる(`internal/cli/main_test.go`)。`TestOrgWatchDetach_AlreadyRunning_StartsNothing`(`org_watch_test.go:275`)、`_Stale_StartsNothingAndSaysHowToStopIt`(`:295`)、`_StartsTheChildAndWaitsForItsRunningMark`(`:361`。`--config` と `--interval-seconds` の有無)、`_ChildExitsFirst_ExitsOneNamingTheLog`(`:402`)、`_DeadmanMinutesZero_StartsTheChild`(`:646`)、`_StartDetachedProcess_NewSessionDevNullStdinAppendedLog`(`:591`。シェルスクリプトの子で setsid・/dev/null・追記を見る) |
| AC12b 待ちきれなかった子の回収 | 満たす | `org_watch.go:348-353`(起動した子を `stop` し、`waitOrgWatchReleased` でロックが放されたことを見てから返す)。`TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock`(`org_watch_test.go:420`。印を書かない子で、ロックが放される場合と残る場合) |
| AC13 `--stop` | 満たす | `runOrgWatchStop`(`org_watch.go:365-387`)はシグナルを送らない。`TestOrgWatchStop_NotRunning_SaysSoAndWritesNoRequest`(`:476`)、`_StopsARunningWatch`(`:492`。`--stop` と `--all --stop`)、`_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal`(`:524`)、`TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt`(`watch_all_test.go:771`) |
| AC13b 古い頼みに従わない、古い pid にシグナルを送らない | 満たす | `dropOldStopRequest`(`watch.go:659-669`)と `AppliesTo`(`watchalive.go:236-239`。厳密に後の頼みだけ)。`TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`(`watch_all_test.go:821`。前・同時刻・1 ns 後)、`TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt`(`watchalive_test.go:293`)。古い pid の場合は、印が生きている別プロセス(`sleep 30`)を指していても、そのプロセスが残ることを `org_watch_test.go:524` が確かめる |
| AC14 `ralph status` の行と `org_watch` | 満たす | `internal/cli/status.go:130`(state dir ごとに 1 回 probe)、`orgWatchLine`(`:164`)、`statusOrgWatchJSON`(`:388`、`:413`)。`TestStatusCmd_OrgWatchLineAndJSONPerWatchState`(`status_test.go:525`)、`_OrgWatchLineFollowsTheStateDirLine`(`:625`)、`_EmptyRosterStillShowsTheOrgWatch`(`:642`)、`_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml`(`:672`)、`_OrgWatchWithDeadmanMinutesZeroJudgesByTheDefault`(`:703`)、`_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad`(`:735`) |
| AC15 spawn・send・start・stop・disband・escalate は変わらない | 満たす(静的な部分) | 変わったテストファイルは 12 本で、spawn・send・start・stop・disband のテストファイルは 1 本も入っていない(`git diff --stat 577ebeaf...HEAD -- '*_test.go'`)。`internal/cli/org_test.go` の差分は watch のテスト 3 本だけで、`org_legacy_ledger_test.go` の 1 行も watch の引数。`spawn.go` と `verbs.go` の差分は doc と `InboxHintStateDir` のフィールドだけ。escalate の動きは `escalate.go` の差分で reason を引数にしただけで、値は同じ `inbox_no_director`。テストが通るかは /test |
| AC16 tech-debt (c)〜(h) は動作を変えない | 満たす(注記つき) | `internal/cli/org_inbox_test.go` は無変更で、`TestOrgEscalate_BrokenConfig_AlertsTheHumanAndExitsOne`(`:349`)が (h) の `wireOrgInbox`(`internal/cli/org.go:224-228`)を拾う。`TestInboxCommand`(`internal/org/escalate_test.go:615`)も無変更。注記: `internal/org/escalate_test.go` は 9 か所の呼び出しの形が変わった。(g) が `NotifyInboxItem` から引数を外し、`EscalateParams.HintStateDir` を `Org.InboxHintStateDir` に移したためで、計画の Scope がこの変更を求めている。assertion の行は変わっていない |
| AC17 文書と sync の検査 | 一部 | `scripts/check-sync.sh`(DRIFTED 0)と `scripts/check-skill-sync.sh`(13 skill)は通る。新しい watch の形(`--all`、`--detach`、`--stop`、`--org-id` の廃止、受信箱の期限)は skill の 4 面、README、AGENTS.md、仕様の注、quality-gates の 2 面に入っている。生存の窓の規則が古いまま残る箇所が 2 つある(下の「Documentation drift」の D1 と D2。/sync-docs で直す) |

#### 仕様 FR-6 との対応(cycle 1)

| FR-6 の項目(`docs/specs/2026-10-07-org-multi-org-director.md:56-62`) | この PR | 根拠 |
| --- | --- | --- |
| `watch --all` を 1 プロセス立て、全 org のパルスと受信箱の期限を見る | 満たす | AC5、AC7、AC7b |
| ack と resolve の期限が切れた件を人への経路に直接送る | 満たす | AC2、AC3、AC4、AC6 |
| 生存の印を書く | 満たす | AC10、AC11 |
| 印が古いときに start・spawn・TASK を断る、start が見張りを起動する | 後半の PR | 計画の Non-goals。判定関数 `ProbeWatch` だけを先に入れた |
| 横断の stop は見張りなしで使える | 後半の PR(確認) | 計画の Spec compliance criteria。この PR は stop を変えていない |
| 再起動で受信箱を読み直し、期限を計算し直す | 満たす | AC5 |
| 試行・受領・座席の停止を別々に記録する | 満たす | 期限の送信は `notified` を足すだけで、状態(`acked`、`resolved`)と manifest の `stopped` に触れない(AC3) |

仕様の受け入れ条件では、105 行(ack したが resolve しない件が resolve の期限で届く)をこの PR が満たす(`deadline_test.go:186` の「resolve deadline of an acked item」)。106 行は後半(起動し直した見張りが期限切れの件を届ける)だけをこの PR が満たし、前半(start の拒否)は次の PR に残る。104 行は計画どおり 8a 段目に回す。仕様の注(`:63`)はこの区分を書いている。

### Static analysis(cycle 1)

| Command | Result | Notes |
| --- | --- | --- |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | pass(rc 0) | `scripts/ralph-config.sh` が言語に分類されないため、scope は full に落ちた。shellcheck と hook の `sh -n`、settings の `jq -e`、Codex の 3 つの guard、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の参照を通る。Go は `gofmt: ok`、`go vet` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし。branch の secret scan(577ebeaf..94d5248c)は clean |
| `HARNESS_VERIFY_MODE=static RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` | pass(rc 0) | 計画の Verify plan が名指しする最後の full gate。結果は上と同じ |
| `./scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 167、DRIFTED 0、KNOWN_DIFF 5(`docs/quality/quality-gates.md` を含む。watch の行は root と `templates/base/` で同じ文) |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 13 skill が lock-step。`/org` の 4 面はバイト一致(`cmp`) |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | d0a6f2735552 |

### Documentation drift(cycle 1)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1 `/org` skill の 4 面 `:775`(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | いいえ | stale の説明が「`[org].deadman_minutes` より古い」のまま。今の窓は `deadman_minutes`(1 未満なら 10)と見張りの間隔の 2 倍の大きいほう(`watchalive.go:195-200`)。4 面を同じ文に直す。ほかの記述(動詞表 `:176`、受信箱の節、長時間運用、`--stop`)はコードと合う |
| D2 仕様 FR-6 の注 `docs/specs/2026-10-07-org-multi-org-director.md:63` | いいえ | `running`(ロックあり、印が running で `deadman_minutes` 以内)が古い規則。D1 と同じ窓の規則と、`deadman_minutes = 0` は「デッドマンを時間切れで上げない」の意味なので判定では既定の 10 に読み替えることを書く |
| 仕様 FR-6 の要件 `:59` | 直さなくてよい | 「印が古いとき(`deadman_minutes` より前)…断る」は後半の PR の要件で、この PR の動きではない。D2 の注で「古い」の判定が窓の規則になったと書けば足りる。要件の文を変えるかどうかは /sync-docs の判断に任せる |
| README `:124`、`:125`、`:253` | はい | `--all`、`--detach`、`--stop`、`--org-id` の廃止、期限の既定、`org watch:` の行と `org_watch`。窓の規則には触れていない |
| AGENTS.md の repo map `:90` | はい | `RunWatchAll`、`watch.lock`、`watch-alive.json`、`deadline.go` |
| `docs/quality/quality-gates.md` と `templates/base/` の写し | はい | watch の行は 2 面で同じ文 |
| `.claude/rules/ralph/agent-messaging.md` と写し | はい(変更なし) | watchdog の identity と `TO: leader` の記述は `--all` でも正しい。`--org-id` の形を書いていない |
| `OrgConfig.DeadmanMinutes` の doc(`config.go:58-63`)、`templates/base/ralph.toml:51-55`、`ralph status` の stale の行(`status.go:171`) | はい | 94d5248c で窓の規則と 0 の意味に合わせた |
| `ralph org watch --help`(`org_watch.go:46-64`) | はい | `--all`、`--once`、`--detach`、`--stop`、`--org-id` の拒否、SIGINT と SIGTERM |
| `docs/tech-debt/README.md` | はい | (c)〜(h) を閉じた記述はコードと合う(`org.go` は 1,544 行、`inboxSummary` は `org_inbox.go`、`EscalationsPathIn` は `newWatchAllRun`)。新しい行が挙げる関数とテスト(`TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly` は `org_test.go:2889`、`watcherErrorReason`、`HonoredUnknown`)は実在する |
| `watch --org-id` の残り | はい | コードでは廃止を説明するコメント(`watch.go:38`、`org_watch.go:34`)と拒否の文言(`:108`)、テストだけ。ほかは計画・archive・過去の report |
| 計画の Scope `:47` と AC11 `:133` | 対象外 | 古い窓の規則のままだが、承認 digest を保つため本文は変えず、Progress `:209` にずれを書いてある |
| 計画の Progress checklist | 記録の遅れ | 「Review artifact created」が未チェックのまま。self-review の report はある |

/sync-docs に渡すもの: D1(4 面、`:775`)と D2(仕様 `:63`)。`:59` は任意。

### Observational checks(cycle 1)

#### 計画からのずれの判定(cycle 1)

1. 生存の窓(M1)。`ProbeWatch` は印の年齢を `max(deadman_minutes(1 未満なら既定の 10), 2 × interval_seconds)` と比べる(`watchalive.go:177-200`)。妥当と判断する。`deadman_minutes = 0` には「デッドマンを時間切れで上げない」という意味がすでにあり(`watch.go:1288` の `> 0`)、`config.Load` は 0 を通す。前の規則ではこの設定で印がすべて stale になり、`--detach` が健全な子を止めて誤った理由で失敗していた。間隔が `deadman_minutes` より長い設定でも、健全な見張りがサイクルの間に stale になっていた。0 を設定で拒む案は 0 の意味を壊すので、判定の側で読み替える方が影響が小さい。代わりに、間隔の長い見張りが固まっても最大 2 間隔のあいだ running に見える。後半の PR の拒否も同じ規則を受け継ぐ。コードの doc、`ralph.toml` の雛形、status の文言は新しい規則を書く。skill の 4 面と仕様の注が古い(D1、D2)
2. S3a の止める頼みの時刻とナノ秒。`started_at` をロックを待つ前に決め(`watch.go:500-506`)、起動のときに消すのは `started_at` 以前の頼みだけ(`:659-669`)。妥当と判断する。ロックを取ってから印を書くまでの間に `stale` を見た人が打った `--stop` を消さずに従える。待っている間に前の見張りへ打たれた頼みは、前の見張りが `finish` で消す(`:702-706`)ので、後の見張りが誤って従う形は残らない。ナノ秒は、起動の直後の `--stop` が同じ秒に入るためで、`AppliesTo` の `time.Parse(time.RFC3339, …)` は小数秒を受け付ける。`WatchAlive` の doc(`watchalive.go:63-67`)が書いている。仕様の注は精度を書いていないが、利用者の操作には関わらない
3. S2 の 1 未満の分の拒否。`SweepInboxDeadlines` は `ack_minutes` か `resolve_minutes` が 1 未満なら掃き出さずにエラーを返す(`deadline.go:172-175`)。妥当と判断する。本番の Org は `config.Load` を通るので 1 未満にならない。設定なしで組んだ Org(ゼロ値の `Director`)が全件を一度に送る事故を防ぐ。見張りでは毎サイクル stderr に 1 行出る(`watch.go:584-586`)。利用者向けの文書に書く必要はない。関数と `InboxDeadlineSweep.Err` の doc が書いている

S3a の「呼び出しの期限を loop の ctx ではなく Background から作る」は、止めるまでがサイクルの終わりまで延びる代償を tech-debt の行と skill の `--stop` の記述(「30 秒で止まらなくても頼みは残り、見張りはそのサイクルの終わりに止まる」)が書いている。

#### 94d5248c(self-review のあとの直し)

self-review の再実行の N1・N4・N5 を直し、N2・N3 を tech-debt に書いたコミット。9 ファイル、+18/-15。

- N1: `config.go:58-63` と `templates/base/ralph.toml:51-55` が 0 の意味と窓の規則を書くようになり、`status.go:171` の stale の行と `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` の `staleTail`(`status_test.go:546`)が同じ文になった。doc の「0 (or any value below 1) means the deadman never escalates by timeout」は `checkDeadman` の `> 0`(`watch.go:1288`)と合う
- N4: `sendAlert` から使われない ctx の引数と doc の文が消え、本番の呼び出し(`watch.go:923`)とテスト(`watch_test.go:643`)も合わせた。`go vet` と staticcheck は通る
- N5: `writeWatchAlive` の doc が `writeJSONFileAtomic` を指す(`watchalive.go:158`)
- N2・N3: tech-debt の「Small leftovers of org-cross-watch」の行に (c) と (d) を足した。挙げた関数、テスト、`ralph insights` の `honored_unknown` は実在する。計画の Progress の L3 の記述も「一部を直した」に変わった
- 範囲の外の変更はない。動きが変わるのは status の文言だけで、テストも合わせてある。新しい問題は見つからなかった

#### そのほか(cycle 1)

- 承認 digest は一致する(d0a6f2735552)
- `ralph-config.sh` の root と `templates/base/` はバイト一致。`/org` skill の 4 面もバイト一致
- 本番の herdr と agmsg の driver は `exec.CommandContext` と 2 秒の `WaitDelay` で呼ぶので、AC7b の打ち切りはテストの fake だけでなく本番でも効く。打ち切りから戻るまで、kill のあと最大 2 秒延びうる

### Coverage gaps(cycle 1)

- テストは走らせていない(/test の担当)。上の判定は読み合わせによる
- AC9 の「kill でもロックが放される」は flock の性質に頼る。テストは `Release` で模すだけで、プロセスを kill していない
- 確認していない実機の動き: 切り離した子の osascript の通知が表示されるか、Linux の setsid、30 席で herdr が止まったときの 1 サイクルの長さ
- `--stop` で取り消した watcher の呼び出しを `ralph insights` がどう見せるか(tech-debt の (d))
- 状態ファイルを rename で書くことを見分けるテストはない(self-review の Coverage gaps と同じ)
- 足すと確度がいちばん上がる確認: 一時の台帳で実機を動かす。`ralph org watch --all --detach` で起動し、印の pid を `kill -9` で止めて、`ralph status` が `org watch: not running (a running mark is left from pid N, …)` を出すことと、次の `--detach` が通ることを見る。同じ台帳に 11 分前の open の件を置き、切り離した子から通知が出るかも見る。決定論の検査としては、別プロセスに `watch.lock` を持たせて kill し、`ProbeWatch` が `not_running` を返すテストが最小になる(子のプロセスはシェルの `exec sleep` ではロックを持てないので、test binary を使わない形を工夫する必要がある)

### 判定(cycle 1)

pass

- Verified: AC1〜AC16 と AC7b・AC12b・AC13b は、コードとテストの読み合わせで満たす。計画からのずれ 3 件は妥当で、Progress に記録がある。仕様 FR-6 のうちこの PR が受け持つ 5 項目を満たし、残る 2 項目は計画どおり後半の PR に回した。静的解析(changed scope と full scope の両方)、`check-sync.sh`、`check-skill-sync.sh` は rc 0。94d5248c は self-review の指摘の範囲に収まり、新しい問題はない
- Partially verified: AC17(D1 と D2 の 2 か所に古い窓の規則が残る。/sync-docs で直す)、AC9(kill の経路はテストがない)、AC15(静的に読んだ部分だけ。テストは /test)
- Not verified: テストの実行、切り離した子の osascript の表示、実際の kill のあとのロックの解放、Linux での setsid、herdr が止まったときのサイクルの長さ
