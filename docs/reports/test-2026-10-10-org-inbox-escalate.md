# Test report: org-inbox-escalate

- Date: 2026-10-11(JST。実行は 01:03〜01:41。ファイル名の日付は計画の slug に合わせた)
- Plan: docs/plans/active/2026-10-10-org-inbox-escalate.md(承認済み、digest b09718e048cd)
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 1(`cycle-count.json` は 1、上限 2)
- Scope: HEAD 495aded1(base origin/main 382c18c8)。behavioral test だけを実行した(静的解析は /verify の担当)。計画の Test plan(unit、integration、regression、edge cases)と AC10 のテストの部分に加えて、依頼された 3 つを確かめた。verify が挙げた 2 つの mutant、`go test -race`、別々のプロセスから同時に打つ escalate
- Evidence: `docs/evidence/test-2026-10-10-org-inbox-escalate.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore -v` で `.gitignore:58` に当たることを確かめた)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-10-160334.log`(時刻は UTC)
- テストは足していない。mutant の写しと probe のテストは scratchpad に置き、`go test -overlay` と `go build -overlay` で使った。worktree の追跡ファイルは書き換えていない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(HEAD、変更言語の scope) | shell 40 本(PASS の行 3,944)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 6 分 30 秒、rc 0 |
| `go test ./... -count=1 -v` | top-level 1,309、subtest 1,349(org 516 と 815、cli 521 と 364) | すべて | 0 | 3(下の注) | 81.8 s(cli 81.0 s、org 25.2 s)、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -count=1` | 2 パッケージ | 2 | 0 | 0 | 86.4 s(org 26.2 s、cli 82.2 s)、rc 0 |
| `go test ./internal/org/ ./internal/cli/ -count=1 -coverprofile` | 2 パッケージ | 2 | 0 | 0 | 79.7 s、rc 0 |
| `-run 'WaitInbox\|Concurrent\|InboxLock' -count=10`(org と cli、裏で org と cli の全体を走らせながら) | top-level 200(20 本 × 10) | すべて | 0 | 0 | org 44.0 s、cli 8.8 s。裏の全体も rc 0 |
| mutation(`go test -overlay`、HEAD のテスト) | 42 件(依頼の 2 件と追加の 40 件) | red 41 件 | 生き残り 1 件(E13) | - | 1 件 25〜30 s(cli は 83〜87 s) |
| 別々のプロセスからの escalate(枝のバイナリ) | 1,480 件(2 並列で 200、8 並列で 1,280) | すべて | 0 | - | 1 回 2〜5 s |

- shell の件数は suite ごとに PASS・FAIL・SKIP の行を数えた(`PASS: 29` や `SKIP: 0` のような集計行は除いた)。40 本すべてが `OK` で終わり、FAIL の行は 0。Skipped の 1 件は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)。この差分は shell のスクリプトにもテストにも触れていない
- `run-test.sh` の中の `go test ./...` は `internal/org` だけを実行し、ほかはキャッシュの結果だった。そのため 2 行目で全パッケージを `-count=1` で流し直した。go は 1.26.0(darwin/arm64)
- Go の Skipped の 3 件は、`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin では本物の osascript が動くので skip する。Linux の CI では走る)と、`internal/scaffold` の前からある 2 件(`TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`)
- 新しいテストは top-level 67 本(`internal/org/inbox_test.go` 21、`escalate_test.go` 23、`internal/cli/org_inbox_test.go` 23)。66 本が通り、1 本は上の skip
- 実行中、ほかのセッションの負荷で load average は 4〜11(12 コア)だった。既知の flaky なテスト(MEMORY.md の 3 本)はどれも落ちなかった

## Coverage

- Statement: `internal/org` 93.8%、`internal/cli` 86.0%(前の計画の時点は 93.9% と 85.5%)。ほかの 6 パッケージは測り直していない
- Function(差分が足した関数):

| ファイル | 100% の関数 | 100% でない関数と、通らない文 |
| --- | --- | --- |
| `internal/org/inbox.go` | 21 関数のうち 19(`foldInbox`、`apply`、`update`、`Ack`、`Resolve`、`ValidateInboxNote`、`nextID` ほか) | `Escalate` 94.7%(読めたあとの追記の失敗)、`appendLocked` 66.7%(marshal・open・write・close の失敗) |
| `internal/org/escalate.go` | 14 関数のうち 8 | `Escalate` 93.8%、`EscalateNotRecorded` 85.7%、`NotifyInboxItem` 86.7%(どれも `Inbox == nil` の防御。`NotifyInboxItem` は読み込みの失敗も)、`WaitInbox` 96.3%(poll の間隔の既定値)、`inboxDesktopNotify` 63.6%(darwin の本物の osascript と、darwin 以外の `skipped`)、`bannerOrStderr` 66.7%(nil のとき) |
| `internal/cli/org.go`(inbox・escalate・`wait --inbox`) | 17 関数(手を入れた `newOrgRuntimeAt` を含む)のうち 13 | `runOrgWaitInbox` 90.0%(runtime を作れないとき)、`newOrgInboxCmd` 95.2%・`newOrgInboxShowCmd` 94.7%(受信箱の読み込みの失敗)、`printInboxItem` 96.2% |
| `internal/org/lockfile.go` | `withManifestLock` | `withFileLock` 93.3%(flock の取得の失敗)。`withManifestLock` の中身を切り出した部分で、前からある枝 |

- 通らない枝のうち 3 つは、下の「プロセスをまたぐ確認」で枝のバイナリを使って通した。poll の間隔の既定値(`wait --inbox` が 1 秒ごとに読み直す)、読めたあとの追記の失敗(`inbox.jsonl` を 0444 にした)、darwin で `DesktopNotify` が nil のとき(osascript のない PATH で `failed: exec: ...` になる)
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 行の coverage は「条件を外しても気づくか」を示さない。そのため次の節の mutation で確かめた

## 依頼された mutation

`scratchpad/oie-test/mut.py` は、`internal/org/inbox.go` などの写しを書き換えて `go test -overlay` でパッケージ全体を走らせる。書き換える文字列がファイルに 1 回だけあることを、当てる前に確かめた。org で生き残った mutant は cli でも走らせた。

| ID | 書き換え | 結果 | 落ちたテストと落ち方 |
| --- | --- | --- | --- |
| R1 | `appendLocked` の、末尾が改行でないときに改行を足す 3 行(`inbox.go:338-340`)を外す | red | `TestInboxEscalate_TornLastLineStaysSeparateAndTheNewEventIsReadable` の 2 ケースとも。新しい `escalated` が切れた行に混ざって読めなくなり、`Ack of the new item: ... ack e6: no such inbox item`(`inbox_test.go:336`) |
| R2 | `foldInbox` の、生の行から ID を拾う走査(`inbox.go:369-371`)を外す | red | `TestInboxEscalate_NeverReusesTheIDOfADamagedLine` の「the damaged line holds the largest ID」(`Escalate = e4, want e9`、`:310`)と、`_TornLastLineStaysSeparate…` の 2 ケース(`e5, want e6` と `e5, want e13`、`:333`) |

2 つとも落ちたので、Codex plan advisory の指摘 1 の直しはテストで固定されている。ただし、R2 を落とすのは 3 ケースのうち一部だけ。`_NeverReusesTheIDOfADamagedLine` の 1 つ目のケース(壊れた e2 が e1 と e3 の間にある)は、どちらにしても e3 が最大なので R2 でも通る。int64 を超える ID のケースも、もともと数えない値なので R2 でも通る。R2 を落とせるのは 2 つ目のケースと、切れた最後の行の 2 ケースだけで、これらを消すと気づけなくなる。

## 追加の mutation

同じ仕組みで 40 件を足した。39 件が red、1 件(E13)が生き残った。

| ID | 書き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| X1 | `withLock` が flock を取らずに `fn` を呼ぶ | red | 並行の 4 本、`TestInboxLock_IsSeparateFromTheManifestLockAndSerializesWrites`、`TestInboxStore_ReportsAnUnreadableInboxAndAnUnavailableLock`、`TestInboxStore_LivesUnderTheGivenStateDir` |
| X2 | `Escalate` の読む・ID を決める・書くを、ロックの外で行う | red | `TestInboxEscalate_ConcurrentCallsGetDistinctIDs`、`TestEscalate_ConcurrentEscalatesGetDistinctIDs`、X1 と同じロックの 2 本と `_LivesUnderTheGivenStateDir` |
| X3 | `update`(ack・resolve・notified)をロックの外で行う | red | `TestInboxConcurrentAckAndResolve_LeaveAValidState`、`TestInboxConcurrentAcks_OnlyOneChangesTheItem` |
| X4 | acked への ack が、もう一度 `acked` を書いて changed を返す | red | `TestInboxAckAndResolve_TransitionTable`、`TestInboxConcurrentAcks_OnlyOneChangesTheItem` |
| X5 | resolved への ack を、acked と同じく何もせず通す | red | `TestInboxAckAndResolve_TransitionTable`、`TestInboxConcurrentAckAndResolve_LeaveAValidState` |
| X6 | resolved への resolve を通す | red | `TestInboxAckAndResolve_TransitionTable` だけ |
| X7 | 畳み込みで、open でない件への `acked` を当てる | red | `TestInboxRead_SkipsAndCountsLinesItCannotUse` だけ |
| X8 | 畳み込みで、2 回目の `resolved` を当てる | red | 同上 |
| X9 | 畳み込みで、同じ ID の 2 回目の `escalated` が件を置き換える | red | 同上 |
| X10 | 読めた行の `id` を次の ID に数えない(生の行の走査だけ) | red | `TestInboxEscalate_CountsAnIDWrittenWithJSONEscapes` |
| X11 | 生の行の正規表現が、閉じる `"` まで求める | red | `_TornLastLineStaysSeparate…` の「cut inside the id」 |
| X12 | 生の行の正規表現が、`:` の前後の空白を許さない | red | `_NeverReusesTheIDOfADamagedLine` の 2 つ目のケース |
| X13 | `inboxIDNumber` が `e01`・`e0` を件の ID として受ける | red | `TestInboxRead_SkipsAndCountsLinesItCannotUse` だけ(`items = e0,e1,e01,e2, want e1,e2`) |
| X14 | 追記の行の末尾に改行を付けない | red | `_TornLastLineStaysSeparate…` |
| X15 | `update` の、ファイルがないときに先に返す枝を外す | red | `TestInboxTransitions_UnknownIDIsErrInboxUnknownID`(ディレクトリとロックができる) |
| X16 | note の上限を 501 字にする | red | `TestValidateInboxNote`、`TestInboxResolve_RejectedNoteWritesNothing` |
| X17 | note の制御文字を通す | red | 同上 |
| X18 | U+2028・U+2029 を改行として扱わない | red | `TestValidateInboxNote` |
| X19 | `nextID` の int64 の上限の検査を外す | red | `TestInboxEscalate_FailsWhenNoIDIsLeft` |
| E1 | どの TYPE でも escalate を受ける | red | `TestEscalate_RefusalsWriteNothingAndNotifyNoOne` |
| E2 | org_id の形を検査しない | red | 同上 |
| E3 | `protocol.Validate` を飛ばす(`Parse` だけ) | red | 同上、`TestEscalateNotRecorded_ChecksTheMessageThenAlertsTheHuman` |
| E4 | `escalations.jsonl` に書けなくても `notified` を書く(エラーは返す) | red | `TestEscalate_EscalationsUnwritable_ReportsTheIDAndInboxNotifyCompletesIt` |
| E5 | `escalations.jsonl` に書けなかったことを無視する | red | 同上、`TestEscalate_HintStateDirReachesTheBannerAndTheRecoveryCommands` |
| E6 | 記録は済み・通知は未完了のとき、`Recorded` を false にする | red | `_EscalationsUnwritable_…`、`_NotifiedUnwritable_…` ほか |
| E7 | 受信箱に書けないとき、人に知らせずに返る | red | `TestEscalate_InboxUnwritable_AlertsTheHumanAndFails` |
| E8 | 記録できなかったときのデスクトップ通知を出さない | red | 同上、`TestEscalateNotRecorded_…` |
| E9 | 通知の失敗を `ok` と記録する | red | `TestEscalate_DesktopNotificationFailureStillSucceeds` |
| E10 | resolved の件に `inbox notify` を通す | red | `TestNotifyInboxItem_RefusesUnknownAndResolvedItems` |
| E11 | `WaitInbox` が acked の件でも返る | red | `TestWaitInbox_ReturnsTheOpenItemsAtOnce`、`_TimesOutWithoutAnOpenItem` |
| E12 | 期限が来たあとにもう一度読まない | red | `TestWaitInbox_ReadsOnceMoreAtTheDeadline` |
| E13 | `WaitInbox` の `if timeout > 0` を `>= 0` にする(0 で無期限のはずが、すぐ期限切れになる) | **生き残り**(org と cli の全体) | なし。次の節 |
| E14 | 手で書き換えた行の TYPE を、そのままデスクトップ通知に渡す | red | `TestNotifyInboxItem_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral` |
| E15 | banner の org_id をエスケープしない | red | 同上 |
| C1 | `wait --inbox` が `--seat` を受ける | red | `TestOrgWaitInbox_RefusesSeatFlags` |
| C2 | inbox の動詞が `--org-id` を受ける | red | `TestOrgInbox_RefusesOrgID` |
| C3 | escalate が、通知まで済んだときだけ ID を出す | red | `TestOrgEscalate_RecordedNotNotified_ExitsOneThenInboxNotifyCompletesIt`、`TestOrgInbox_RecoveryCommandsRepeatAnExplicitStateDir` |
| C4 | `--all` なしの一覧に resolved を出す | red | `TestOrgInbox_ListAllAndJSON` |
| C5 | `ralph.toml` が読めないとき、人に知らせずにエラーを返す | red | `TestOrgEscalate_BrokenConfig_AlertsTheHumanAndExitsOne` |
| C6 | escalate が旧い台帳の検査を飛ばす | red | `TestOrgInbox_LegacyLedgerGuard` |

- X6・X7・X8・X9・X13 は、それぞれ 1 本のテストだけが落とす。どれもそのテストの専用のケースが落としているので穴ではないが、そのケースを消すと気づけなくなる
- E4 の 1 回目は、書き換えで変数が使われなくなりコンパイルに失敗した。`notified` を書いてからエラーを返す形に書き直して走らせ直した
- ロックを外す X1・X2・X3 は、`-run Concurrent -count=10` で 10 回とも落ちた(X1 の `TestInboxConcurrentAcks_OnlyOneChangesTheItem` だけは 9/10。同じ mutant をほかの 3 本が 10/10 で落とす)

### 生き残った E13: `--timeout-ms 0` の「無期限に待つ」が固定されていない

計画の Scope は「`--timeout-ms`(既定 60000、0 で無期限)」で、Edge cases にも `--timeout-ms 0` がある。HEAD の実装は合っているが、テストはこの動きを固定していない。timeout 0 を渡すテストは、どれも初めから open の件があるか(`TestWaitInbox_ReturnsTheOpenItemsAtOnce`、`TestOrgWaitInbox_ReturnsAnOpenItemAtOnce`)、読み込みに失敗するか(`TestWaitInbox_ReadFailureReturnsAtOnce`)、フラグの検査で止まる(`TestOrgWaitInbox_RefusesSeatFlags`)。どれも最初の 1 回の読み込みで決まるので、待つ枝に入らない。

固定されていない動き: 「timeout 0 で、初めは open の件がなく、あとから件が届く」ときに、期限切れにならずにその件を返すこと。

穴が本物であることは、`-overlay` で足した probe のテスト(`scratchpad/oie-test/zz_probe_test.go`、コミットしていない)で確かめた。空の受信箱で `WaitInbox(0)` を呼び、300 ms 後に別の goroutine が escalate する。HEAD では 0.31 s で e1 を返して通り、E13 では 78 µs で `no open inbox item arrived within 0 ms` を返して落ちた。枝のバイナリでも、`wait --inbox --timeout-ms 0` が 2.5 s 後に別のプロセスが打った escalate を 3.05 s で受けて rc 0 で返ることを確かめた(次の節)。依頼どおり、テストは足していない。

## プロセスをまたぐ確認

枝を `go build` したバイナリを scratchpad に置き、`scratchpad/oie-test/xproc.sh` で打った。どのプロセスも `env -i PATH=<空のディレクトリ> HOME=<scratchpad の下>` で、git の外のディレクトリから起こした。osascript は見つからない。herdr、`~/.config/herdr`、本物の台帳には触れていない。台帳のパスには空白を入れ、ディレクトリは作らずに始めた(最初の組が同時にディレクトリを作る)。

1. escalate を N 本同時に打つ組を R 回
2. 台帳の `inbox.jsonl` の末尾に、改行のない切れた行 `{"ts":"…","id":"e500` を足し、さらに N 本ずつ 10 回
3. e1〜e20 のそれぞれに、`inbox ack` を N 本と `inbox resolve` を 1 本、同時に打つ

`scratchpad/oie-test/xcheck.py` は次を確かめる。全プロセスの rc と stdout の `escalated e<N>`、ID が重ならず 1 から順に埋まり、2 の後は e501 から始まること。`inbox.jsonl` で読めない行が足した切れた行の 1 行だけであること。`escalated` と `notified` が件の数ずつあること。`escalations.jsonl` の `inbox_id` が重ならないこと。3 で各件の `acked` が 1 つ以下、`resolved` がちょうど 1 つで、`inbox --json` の `ignored_events` が 0 であること。

| バイナリ | N × R | 回数 | 結果 |
| --- | --- | --- | --- |
| HEAD | 2 × 40(計 100 件) | 2 | どちらも OK。ID の重なり 0、壊れた行は切れた行だけ、`notified` の osascript は 100 件とも `failed: exec: "osascript": executable file not found in $PATH`、終了コードはすべて 0。3 は 1 回目が `acked` 20・`already acked` 17・`refused: resolved` 3 |
| HEAD | 8 × 30(計 320 件) | 4 | 4 回とも OK(計 1,280 件、ID の重なり 0) |
| R1 | 2 × 40、8 × 30 | 各 1 | どちらも検出。2 の最初の escalate が切れた行に混ざり、rc 1(`notify e501: no such inbox item`)。e501 は読めない行になった |
| R2 | 2 × 40、8 × 30 | 各 1 | 検出。2 の ID が e501 ではなく e81(8 並列では e241)から始まった |
| X2(`Escalate` のロックなし) | 8 × 30 | 4 | 4 回のうち 1 回で ID が重なった(e158 が 2 回)。2 × 40 では 1 回のうち 0 回 |
| X1(ロックなし) | 8 × 30 | 4 | 4 回のうち 1 回で重なった(e187) |
| X3(`update` のロックなし) | 8 × 30 | 1 | 検出できなかった。2 × 40 でも同じ |

- HEAD の 6 回(計 1,480 件)で ID の重なりも壊れた行もなかった。プロセスをまたいでも flock が escalate を直列にしていることと矛盾しない。ただし、ロックを外した mutant を同じ方法で落とせたのは 8 並列で 4 回に 1 回ほど、ack と resolve の組では 0 回だった。プロセスの起動に比べて読み・決め・書くの区間が短く、同時に入ることが少ない。この確認だけでは「ロックが効いている」とは言えない。ロックの外し方 3 通りは、goroutine のテストが 10 回とも落とす(上の節)
- 2 の切れた行の扱いは、プロセスをまたいでも決まった結果になった。HEAD では新しいイベントが読める行として残り、R1 と R2 はどちらも毎回検出された
- X1 と X2 を台帳のディレクトリなしで始めると、最初の 1 本が `no such file or directory` で落ちた。ディレクトリを作る `MkdirAll` が `withFileLock` の中にあるためで、ID の重なりとは別の現象。8 並列の確認ではディレクトリを先に作った

ほかに枝のバイナリで確かめたこと:

- `wait --inbox --timeout-ms 0` を空の台帳で起こし、2.5 s 後に別のプロセスが `RESULT` を escalate すると、3.05 s で `e1	auth-core	RESULT	PR opened: …` を出して rc 0 で返った(1 秒ごとの既定の poll)。e1 を ack したあとの `--timeout-ms 1500` は 1.51 s で rc 1(`no open inbox item arrived within 1500 ms`)
- `inbox.jsonl` を 0444 にすると(読めるが追記できない)、escalate は NOT RECORDED の banner を出して rc 1 になり、`escalations.jsonl` に `inbox_not_recorded` の行が残った。`inbox ack e1` も rc 1。`inbox.jsonl` は変わらず、権限を戻したあとの escalate は e2 になった(書かれなかった件は ID を消費しない)
- osascript に渡す 3 種類の文字列(escalate の通知、`<invalid>` に置き換えた通知、記録できなかったときの通知)を、`osacompile` で実行せずにコンパイルだけした。3 つとも AppleScript として通った。本物の通知は出していない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| watch の escalation(`escalationRecord` に `inbox_id` を足し、`alert_id` を `omitempty` にした) | 通る | `TestWatch_*` 27 本、`TestPruneEscalated_*` 2 本、`TestEscalationRecord_WatchAndInboxLinesKeepTheirOwnID`(2 行目の `-v` の出力) |
| `ralph org wait --seat` の既存のテスト | 通る | `TestOrgWait_*` 7 本(`_ExplicitZeroTimeout_StaysUnbounded` を含む) |
| `withManifestLock` を `withFileLock` に切り出したこと | 通る | `TestWithManifestLock_SerializesConcurrentCallers`、`TestManifestStore_ConcurrentAppendsAllLinesIntact`、`TestOrgSpawn_Concurrent*` 4 本。`-count=10` でも 10 回とも通った |
| leader の雛形の書き換え(AC8) | 通る | `TestRenderRolePrompt_Leader_*` 8 本 |
| linked worktree から共通の台帳を読む(AC7) | 通る | `TestOrgInbox_LinkedWorktreeSharesTheMainCheckoutsInbox` |
| AC10 `./scripts/run-verify.sh` | テストの部分は通る | `./scripts/run-test.sh` が rc 0。静的な部分は verify の報告で rc 0 |

## Test gaps

- E13: `wait --inbox --timeout-ms 0` が「件が届くまで無期限に待つ」ことを固定するテストがない(上の節)。HEAD の動きは probe と枝のバイナリで確かめた。足すなら、空の受信箱で `WaitInbox(0)` を呼んで、あとから escalate した件を返すことを見る 1 本で足りる(probe がその形)。足すかどうかは orchestrator の判断に任せる
- 本物の osascript でデスクトップ通知が出ることは確かめていない。依頼どおり osascript のない PATH で打ち、AppleScript の文字列は `osacompile` で構文だけ確かめた。darwin 以外の `skipped` の枝(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`)は手元では skip され、Linux の CI でだけ走る
- プロセスをまたぐ同時の escalate では、HEAD の 1,480 件で ID の重なりはなかった。ただし上の表のとおり、この方法でロックの外し方を落とせるのは 4 回に 1 回ほどなので、プロセスをまたぐロックの保証は、goroutine のテストと、manifest と同じ flock の仕組みであることに頼っている
- `appendLocked` の open・write・close の失敗、`Inbox == nil` の防御、CLI の受信箱の読み込みの失敗は、テストでは通らない。open の失敗は 0444 の確認でバイナリから通した。ほかは起こしにくい防御の枝
- 実機(herdr の上の leader)が雛形のとおりに escalate を打つことは確かめていない。計画も求めていない
- 参考(テストの失敗ではない): 追記の open に失敗したときのエラー文で、パスが 2 回出る(`org: inbox: open <path>: open <path>: permission denied`)。`appendLocked` が `*PathError` を、パスを足して包むため。flock のファイルを開けないときの文(`org: open inbox lock file <path>: open <path>: …`)も同じ形で、こちらは manifest のロックの前からの形

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、Go 8 パッケージ)。全パッケージを `-count=1` で流し直して 0 fail、`-race`(org と cli)も 0 fail。並行と待つテストは負荷の下で `-count=10` でも 0 fail。依頼の 2 つの mutant(R1、R2)はどちらも落ち、追加の 40 件も 39 件が落ちた。別々のプロセスから同時に打った 1,480 件の escalate で、ID の重なりも壊れた行もなかった
- Fail: 0。flake も出なかった
- Blocked: なし。生き残った E13(`--timeout-ms 0` の無期限の待ちが固定されていない)と Test gaps の残りは merge を止めない。E13 のテストを足すかは orchestrator が決める
