# Test report: org-director-wait

- Date: 2026-10-19(JST 13:35〜16:00)
- Plan: docs/plans/active/2026-10-19-org-director-wait.md(8 段目の 8b-2b。director の受信箱の wait の印と、見張りの停止の知らせ)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 45f5d586(base origin/main e38b89a9、21 コミット、45 ファイル、+5,481/-465)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、Darwin 25.5.0、12 CPU)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)。実行中の load average は 4〜45
- Evidence: `docs/evidence/test-2026-10-19-org-director-wait.md`(各コマンドの結果の行、落ちたテストの出力と再実行、隔離の確かめ、coverage、負荷の下の実行)と `docs/evidence/test-2026-10-19-org-director-wait-mutation.txt`(mutant 56 個の表、殺したテストの最初の assertion、生き残りのパッケージ全体の再実行、probe の source と結果)。run-test.sh の全ログは `docs/evidence/verify-2026-10-19-043532.log`(gitignore の対象)。`<scratch>` はセッションの scratchpad
- テストもコードも変えていない。mutant と probe と canary の overlay は scratchpad に置き、`go test -overlay` で読ませた。`git status --porcelain` は実行の前もあとも空だった。Go の実行はすべて 1 つずつ順に打ち、パッケージ全体の実行を 2 つ同時には回していない
- 途中で API の接続エラーで一度止まった。止まる前に background で始めていた mutation の実行は最後まで走って記録が残っていたので、それを使って続けた

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(changed。golang を選んだ。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 887 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -v -timeout 30m` | top-level 2,210 | 2,206(PASS の行は subtest を含め 5,981) | 1(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、下の Failure analysis) | 3 | 577 秒、rc 1 |
| 同じ失敗の再実行 `-run '^…$' -count=10`(go1.25.8) | 10 | 10 | 0 | 0 | 9 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -count=1 -v -timeout 30m`(パッケージ全体の再実行) | top-level 1,096 | 1,095 | 0 | 1 | 454 秒、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -run 'Director\|WaitInbox\|OrgWait\|ExitCode\|SignalCause\|OrgGate\|Status\|Escalate\|Watch' -count=1 -timeout 30m`(go1.26.0) | top-level 664 | 663(org 388、cli 275。PASS の行 1,852) | 0 | 1 | 173 秒、rc 0、DATA RACE 0 |
| うちこの差分で足したテスト(org 25、cli 19) | top-level 44 | go1.25.8 の回、`-race` の回、canary の回、負荷の回のすべてで 44 PASS | 0 | 0 | - |
| canary を PATH の先頭に置いて新しいテスト 44 本(AC9、下の「隔離」) | 44 | 44 | 0 | 0 | 13 秒、canary の呼び出し 0 |
| 16 本の busy loop の下で新しいテストを `-count=5`(12 CPU、load average は終わりに 50) | org 125 回、cli 95 回 | すべて | 0 | 0 | org 18 秒、cli 53 秒 |
| mutation(`go test -overlay`。org 34、cli 22) | mutant 56 | 赤 50 | 生き残り 6(うち 5 は probe で赤、1 は等価) | - | org 9〜59 秒、cli 18〜48 秒(1 個ずつ順に) |

- run-test.sh の shell は 41 本すべてが `OK` で終わり、`FAIL:` の集計の行はどれも `FAIL: 0` だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell のテストを変えていない
- run-test.sh の Go は cli(287.4 s)、config、org(453.3 s)を流し直し、insights、org/driver、org/protocol、scaffold、upgrade は worktree の `.harness/cache/go-build` で `(cached)` だった。go1.25.8 の回は `-count=1` で 8 パッケージすべてを流し直した。org 以外の 7 パッケージは ok だった
- Skipped の 3 件は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin で飛ばす作り)、scaffold の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS` で、この差分と関係ない
- 消したテストはない。既存のテストで期待値を変えたのは 6 ファイルで、昇格したセッションの `director:` の行の末尾に足した `; no wait --inbox --director mark for it`(`org_director_test.go`、`org_director_wake_test.go`、`status_test.go`)、`director status` の最後の `wait:` の行(`org_director_test.go`、`org_director_headless_test.go`、`org_director_wake_test.go`)、headless の雛形の wait の 2 文と `--director d3` のコマンド(`internal/org/prompts_test.go`)、コメントだけの変更(`org_director_check_line_test.go`)。どれも D5 と D6 の文の変更に合わせたもの
- 知られた揺れの `TestRegate_FetchFailures/the_fetch_runs_past_the_bound` は、走らせたすべての回で PASS だった

## Coverage

- Statement: `-run` で絞った回の `-coverprofile` で、足した関数と動かした関数を見た(数字はパッケージ全体の値ではない)
  - org(`-run 'DirectorWait|WaitInbox'`): `internal/org/director_wait.go` の 25 関数のうち 17 が 100%。`loop` 97.6%、`begin` 96.4%、`probeAgain` 92.9%、`directorWaitWatchState` 87.5%、`WaitInboxForDirector` 86.7%、`sayLost` 66.7%、`directorWaitItemsDetail` 66.7%、小さな helper 2 つが 66.7%。`newDirectorWaitRun` は org の回で 54.5% だが、既定値の 5 つの block は cli の回で通る。`openInboxItems` 100%、`WaitInbox` 95.7%
  - cli(`-coverpkg` に org を含める): `ExitCode`、`readOrgDirectorWait`、`orgDirectorWaitFacts`、`orgDirectorWaitLine`、`orgDirectorWaitClause`、`notIgnoredSignals`、`printInboxWaitItems`、`readOrgDirectorReport`、`orgDirectorLine`、`printOrgDirectorReport` は 100%。`runOrgWaitInbox` 90.9%、`runOrgWaitInboxDirector` 88.9%、`signalCauseContext` 95.8%。`reraiseSignal` は 0% と出るが、これはテストバイナリの子のプロセスの中だけで走り(`TestOrgWaitDirector_ASignalEndsTheProcessByIt`)、その coverage が合わさらないため
- Branch: 計っていない(Go の標準の道具にない)。`director_wait.go` で両方の回とも通らなかった block は 10 個で、evidence の 5 節に並べた。挙動に関わるのは次の 4 つ
  - `:417-419`: `director-wait.lock` を取れなかった開始で、lock なしで前の印を読む `decide()`(下の Test gaps の G2)
  - `:649-653`: 印のファイルが消えたときの `sayLost`(G1)
  - `:531-533`: 回の settle の途中で ctx が終わる
  - `:672`: 2 件以上の open の件の detail(`<n> open inbox items`)
- Function: 上のとおり
- Notes: 数字は block が通るかを見るために使った

## Test plan coverage

| 計画の項目 | テスト | mutation |
| --- | --- | --- |
| 単体: 印の読み書き(field、一時ファイルと rename、flock、壊れた印、ない印、持ち主のときだけ書く) | `TestDirectorWaitMark_JSONKeys`、`TestReadDirectorWait_MissingCorruptAndRoundTrip`、`TestDirectorWait_AMarkTakenOverIsNotWrittenAgain`、`TestWaitInboxForDirector_AMarkThatCannotBeWrittenIsSaidOnce`、`_AWaitWhoseFirstWriteFailedWritesOnceItCan` | O01、O03〜O06、O27 は赤。O02、O35(消えた印)は生き残り |
| 単体: 登録との照合の表 | `TestDirectorWaitCheckCurrent`、`TestDirectorWaitFor`、`TestWaitInboxForDirector_WaitsForEachRegistrationThatIsCurrent`、`_RefusesBeforeItWaits` | O10、O17、O19、O25、O34 は赤 |
| 単体: 見張りの事実の文の表(panic と書けなかった stopped の fixture を含む) | `TestDirectorWaitWatchFacts_SaysOnlyWhatTheLockAndTheMarkShow`(11 行。`dirWaitCauseWords` の crash、died、killed、panic などがどの文にもないこと、制御文字がないことも見る) | O29、O33 は赤 |
| 単体: 終了コードの関数(4、5、3、1、nil) | `TestExitCode` | C01〜C03、C22、C23 は赤 |
| 結合: `WaitInbox` の director の選択(0、1、4、`note:`、引き継ぎ、settle、5、読めない受信箱) | `TestWaitInboxForDirector_AnOpenItemEndsWithItems`、`_TimesOutWithThePhraseAfterANote`、`_ReadsOnceMoreAtTheDeadline`、`_AWatchThatStopsAcceptingWorkEndsTheWait`(5 行)、`_AnOpenItemWinsOverAWatchThatStopped`、`_CarriesOverOnlyThisRegistrationsPreviousMark`、`_AStaleWatchThatWritesItsMarkWithinTheSettle`、`_AWatchThatStartsAcceptingWorkIsNoted`、`_ARegistrationThatChangesEndsTheWait`(5 行)、`_ACancelledContextEndsWithSignal`、`_ACancelDuringTheStartSettleKeepsTheCarriedWatch`、`_AReadThatFailsEndsWithError` | O07〜O09、O11〜O14、O16、O18、O20〜O24、O26、O30 は赤。O28 は生き残り |
| 結合: 持ち主の順(A が始める → B が始める → A の書き直し → B の終わり) | `TestDirectorWait_AMarkTakenOverIsNotWrittenAgain`(偽の時計で 1 歩ずつ。ファイルの byte を比べる)、`TestWaitInboxForDirector_TwoWaitsLeaveTheLaterOnesMark`(2 つの wait を本当に並べる) | O01、O05、O27 は赤 |
| 結合: CLI の断り、終了コード、stdout、stderr、SIGTERM の `end=signal` | `TestOrgWaitDirector_RefusesItsFlagsBeforeReadingAnything`、`_RefusesAnIDThatIsNotDN`、`_RefusesABadIDBeforeItLoadsTheConfig`、`_ARegistrationThatIsNotCurrentExits5`、`_PrintsOpenItemsAsThePlainWaitDoes`、`_TimesOutWithThePhrase`、`_AWatchThatStopsAcceptingWorkExits4`、`_AContextThatEndsWritesSignalAndRaisesNothing`、`_ASignalWritesItsEndThenRaisesItAgain`(SIGINT、SIGTERM、SIGHUP)、`_ASignalEndsTheProcessByIt`(子のプロセスで本物の送り直し、lock を持たせた 2 回目の SIGTERM、`trap '' HUP INT` で始めた子)、`_SignalCauseContextWithNoSignalsRegistersNone` | C04〜C06、C08〜C12、C21 は赤。C07 は等価 |
| 結合: 表示(`director status`、`ralph status`、両方の `--json`) | `TestDirectorWaitLine_ShowsTheMarkOfAPromotedSessionsWait`(21 行)、`_AHeadlessRegistrationsLineIsUnchanged`、`_DisabledNotAttachedAndOpenHeadless`、`_EscapesTheReadError`、`_EndToEndWithARealWait`、`_HelpDescribesTheWait` | C13、C14、C16〜C19、C24 は赤 |
| 回帰: `--director` のない `wait --inbox` と座席の `wait`、8a〜8b-2a のテスト | 既存の inbox の wait のテスト(`TestWaitInbox_ReturnsTheOpenItemsAtOnce`、`_TimesOutWithoutAnOpenItem` ほか)と、attach、detach、start、stop、escalate、期限、起こす手順、確かめのテストが、すべての回で PASS(go1.25.8 の 1 件は下)。`TestWaitInbox_WritesNoDirectorWaitMark`、`TestOrgWaitInbox_WithoutDirectorHandlesNoSignal` が、印を書かないことと signal を扱わないことを固定する | O30 は既存の `TestWaitInbox_*` 2 本でも赤 |

### AC ごとの対応

| AC | テスト(上の表の行) | mutant |
| --- | --- | --- |
| AC1 フラグ | CLI の断りの 4 本、org の `_RefusesBeforeItWaits`、`TestCheckDirectorWaitID_AcceptsOnlyDN`、`TestDirectorWaitCheckCurrent` | C10、C11、C12、O19、O25、O34 は赤 |
| AC2 印 | 印の読み書きの行、終わり方ごとの org の結合テスト、CLI の signal の 3 本 | O06、O20、O22、O26、O27、C04〜C06、C09、C21 は赤。O31、O32 は生き残り(G2、G4) |
| AC3 持ち主 | 持ち主の順の 2 本 | O01、O03〜O05 は赤。O02、O35 は生き残り(G1。計画の A→B→A→B の順は固定されている) |
| AC4 登録の変化 | `_ARegistrationThatChangesEndsTheWait`(replaced、detached、stopped、手で stopped にして current のまま、同じ id で attach し直し)、CLI の `replaced while it waits` | O16〜O19 は赤 |
| AC5 見張り | 見張りの結合テストと事実の文の表 | O07〜O14、O21、O23、O24、O29、O33 は赤。O28 は生き残り(G3) |
| AC6 終了コード | `TestExitCode`、CLI の `_Exits4`、`_Exits5`、回帰の行 | C01〜C03、C22、C23、O30 は赤 |
| AC7 表示 | 表示の 6 本と、期待値を変えた既存の 6 ファイル | C13、C14、C16〜C19、C24、O10 は赤 |
| AC8 文書 | テストが固定するのは help の文(`TestDirectorWaitLine_HelpDescribesTheWait`、30 秒は `org.DirectorWaitMarkInterval` と照合)、headless の雛形の wait の行(`TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure`、`_GivesEveryRalphOrgCommandTheConfig`)、run-test.sh の中の `tests/test-check-skill-sync.sh`。中身と drift は /verify の担当 | - |
| AC9 隔離と実機 | 下の「隔離」 | - |

## Edge cases

| 計画の edge case | テスト | 結果 |
| --- | --- | --- |
| 見張りが止まった回と、件が届いた回が重なる(0 が先) | `_AnOpenItemWinsOverAWatchThatStopped`(stale の settle の途中に件が届く。次の wait は引き継いだ印で 4)、`_CarriesOverOnlyThisRegistrationsPreviousMark` の「open の件のある開始」 | PASS。O11、O12 が赤 |
| 見張りが `--stop` の途中(stopping) | `_AWatchThatStopsAcceptingWorkEndsTheWait` の stopping の行、事実の文の表の stopping と「lock が外れ印が stopping」の行 | PASS。O33 が赤 |
| 印のディレクトリに書けない(wait は続く) | `_AMarkThatCannotBeWrittenIsSaidOnce`(印の場所がディレクトリ)、`_AWaitWhoseFirstWriteFailedWritesOnceItCan`(state dir を 0555 に) | PASS。O03、O04、O06 が赤。lock を取れないときの道は G2 |
| `--timeout-ms` が印の間隔より短い | `_ReadsOnceMoreAtTheDeadline`(印の間隔 1 時間、timeout 300 ms。終わりの印が書かれる) | PASS。O26 が赤 |
| 登録の `attached_at` が手で書き換えられた | `_ARegistrationThatChangesEndsTheWait` の「attached again under the same id」(director.jsonl を rename で差し替え) | PASS。O17、O18 が赤 |
| 前の印が別の登録のもの(引き継がない) | `_CarriesOverOnlyThisRegistrationsPreviousMark` の「another id」「another attached_at」 | PASS。O09、O10 が赤 |

## Mutation

mutant は 56 個(org 34、cli 22)で、絞った実行で 50 個が赤になった。生き残った 6 個は、パッケージ全体(org の 5 個は cli の絞った実行も)でも生き残った。そのうち 5 個は、それぞれ小さな probe テスト 1 本で赤になる(probe はコミットしない。overlay で足した)。残りの C07 は等価だった。表と生の出力は evidence の mutation の記録にある。

依頼された判断ごとの結果は次のとおり。

| 依頼の判断 | mutant | 結果 |
| --- | --- | --- |
| `wait_id` による持ち主の確かめ | O01(いつも持ち主)、O03、O04(最初の書き込みを断る)、O05(持ち主でなくなったあとも読み、言い続ける)、O27(`wait_id` を pid だけに) | 赤。O02(書いたあとの guard を外す)と O35(消えた印の判定を guard の前に)は生き残り。どちらも、書いたあとに印のファイルが消えた場合だけ挙動が変わる。probe P1 で赤 |
| 4 は「受け付けていた → 受け付けない」の変化のときだけ | O07(受け付けない回はいつも 4)、O08(開始の引き継ぎを捨てる) | 赤 |
| 前の印の `watch_accepting` は同じ登録のときだけ引き継ぐ | O09(どの印からも引き継ぐ)、O10(`DirectorWaitFor` を id だけで比べる) | 赤 |
| 4 の前にもう 1 回受信箱を読み、open の件が先 | O11(probe の回でそのまま 4)、O12(`noWatch` を open の件より先に見る) | 赤。どちらも `_AnOpenItemWinsOverAWatchThatStopped` で捕まる |
| stale を数える前の settle | O13(開始で settle しない)、O14(回で settle しない) | 赤 |
| 30 秒の読み直しで登録の変化に 5 | O16(読み直しで終わらない)、O17(`attached_at` の違いを見ない)、O18(読み直しに開始の登録を渡さない)、O19(stopped を current とみなす) | 赤 |
| 送り直しの前に `end=signal` を書く | O20(ctx の終わりで印を書かない)、C06(送り直さない)、C09(最初の signal で Notify を外す。lock を持たせた 2 回目の SIGTERM で子が先に終わる)、C21(signal がなくても送り直す) | 赤。C07(`stop()` を送り直しのあとに)は等価 |
| `ExitCode` の 3、4、5、1 | C01(4 と 5 を入れ替え)、C02、C22(包んだ sentinel を見ない)、C03(3 を消す)、C23(nil を 1 に) | 赤 |
| `notIgnoredSignals` | C04(無視された signal も登録する)、C05(無視された signal だけを登録する) | 赤。C04 は `trap '' HUP INT` で始めた子が SIGHUP で終わることで捕まる |

生き残りの扱いは次のとおり。

| mutant | 分類 | 根拠 |
| --- | --- | --- |
| O02、O35 | 埋められる穴(LOW、G1) | 書いたあとに印のファイルが消えた wait は、1 回言って二度と書かない(`sayLost` の印がない道、`:649-653`)。この道を通るテストがない。P1(印を `director-wait.lock` の下で消し、stderr の 1 行と、件で終わったあともファイルがないことを見る)が両方を赤にする |
| O31 | 埋められる穴(LOW、G2) | `director-wait.lock` を取れない開始でも、前の印を lock なしで読んで引き継ぐ(`:417-419`)。P2(lock の場所をディレクトリにし、受け付けていた前の印と見張りなし)で、HEAD は 4 で終わり、O31 は `note:` を出して時間切れになる |
| O28 | 埋められる穴(LOW、G3) | 開始の settle の途中で ctx が終わり、引き継ぐものもないとき、「この wait は待ち続ける」という `note:` を出さない。P3 が赤にする |
| O32 | 固定されていない(INFO、G4) | 印の `detail` の escape は、どの文の出力にも出ない(文の形は detail を出さず、JSON に任せる)。ファイルの JSON も encoding/json が制御文字を escape する。違いは読み込んだ値だけで、P4 が赤にする |
| C07 | 等価 | `reraiseSignal` は送る前に `signal.Reset(sig)` を打ち、これがどの channel の `signal.Notify` も外すので、`stop()` が送り直しの前でもあとでもプロセスは同じく終わる。本物の送り直しを子で打つ `TestOrgWaitDirector_ASignalEndsTheProcessByIt` も mutant で PASS した |

- P1〜P4 は HEAD で `-count=10` の 40 回すべて PASS、`-race -count=3` も PASS。各 mutant で落ちたのは、その mutant に向けた probe だけだった
- 1 本のテストでだけ赤になった mutant(O03、O04、O06、O09、O11、O16、O18、O24、O29、O33、O34、C03、C04、C08〜C14、C18、C19)は、落ちた assertion を読み、決めた挙動の違いであって時間切れや揺れではないことを確かめた(evidence の mutation の記録の 2 節)

## 隔離(AC9)

- この機械には `herdr`、`gh`、`tmux`、`orca`(`/opt/homebrew/bin`)と `osascript`、`open`、`curl`(`/usr/bin`)、`agmsg` が入っている。同じ名前(ほかに `codex`、`claude`)の canary を、呼ばれたら記録して exit 1 する script として PATH の先頭に置き、新しいテスト 44 本を流した。44 本すべて PASS、canary の呼び出しは 0 だった
- 感度の確かめ: `WaitInboxForDirector` の開始の前で `exec.Command("herdr", "--version")` を打つ overlay を、canary が先頭にあることを `command -v` で確かめてから流すと、2 本のテストで canary に 2 回届いた。テストの中から外の binary を打てば canary に届き、HEAD では届かない
- 台帳はすべて `t.TempDir()` と `--state-dir`。既定の台帳(main の checkout の `.harness/state/org`)と worktree の `.harness/state/org` は実行の前もあとも存在せず、`.git/ralph/` の下で 13:35 以降に変わったファイルはなかった。main の checkout の `.harness/state/` で変わったのは hook の `edited-files.log`、`needs-verify`、`tool_failures.count` だけで、worktree に `director-wait.*` は現れなかった
- 新しいテストが送る signal は、受け手を先に登録したテストのプロセス自身(`signalSelf`)と、テストが起こしたテストバイナリの子(`child.Process.Signal`、`Kill`)だけ。負荷の実行の busy loop は自分で起こし、script の path で止め、`pgrep` で残っていないことを確かめた
- 実機の確認(D8 の (a)〜(h))は実装の段で取った `docs/evidence/director-wait-livefire-2026-10-19.md` にあり、この /test では打ち直していない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(org、`director_cmd_test.go:310`、go1.25.8 の全体の回) | 期待した「300ms のうちに終わらない」でなく、`process=ended after SIGTERM` が返った | `docs/tech-debt/README.md:279` に記録済みの揺れ。テストは `sh -c "trap '' TERM; exec sleep 30"` が exec に届くのを 100 ms の sleep で待つので、負荷が高いと trap の前に SIGTERM が届いて `sh` が終わる。この回の直後の load average は 1 分で 36、5 分で 45 だった。`director_cmd_test.go` と `director_stop*.go` はこの差分で変わっていない | この差分では直さない。同じ toolchain で単独 10/10 PASS、org パッケージ全体の再実行も PASS。tech-debt の 279 行は「次にこのテストが落ちたとき」をきっかけにしているので、そのきっかけが起きた(下の Test gaps) |

この差分によるテストの失敗はない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `--director` のない `wait --inbox` の出力と終了コード | PASS | 既存の inbox の wait のテストがすべて PASS。`TestOrgWaitDirector_PrintsOpenItemsAsThePlainWaitDoes` が `--director` の有無で stdout が byte で同じことを見る。`openInboxItems` を壊す O30 は既存の `TestWaitInbox_*` 2 本でも赤 |
| 8a〜8b-2a の attach、detach、start、stop、escalate、期限、起こす手順、確かめ | PASS | 3 回の全体の実行で org と cli の既存のテストがすべて PASS(go1.25.8 の 1 件は上で、再実行で PASS) |
| headless の登録の `director:` の行(8b-2a の確かめの文) | PASS | `TestDirectorWaitLine_AHeadlessRegistrationsLineIsUnchanged` が印の有無で行が byte で変わらないことを見る。C16 が既存の `TestDirectorLine_EdgeCasesOfTheCheck` でも赤 |
| `ralph org gate` の signal の扱い(`signalCauseContext` に分けた `gateSignalContext`) | PASS | `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` がすべての回で PASS(`-race` の回を含む) |

## Test gaps

- G1(LOW): 書いたあとに印のファイルが消えた wait が二度と書かないこと(`director_wait.go:649-653`)を通るテストがない。O02、O35 が生き残る。evidence の P1 をそのまま足せば埋まる
- G2(LOW): `director-wait.lock` を取れない開始(`:417-419`)を通るテストがない。O31 が生き残る。P2 で埋まる
- G3(LOW): 開始の settle の途中で ctx が終わり、引き継ぐものもないときに `note:` を出さないことが固定されていない。O28 が生き残る。P3 で埋まる。今の `_ACancelDuringTheStartSettleKeepsTheCarriedWatch` は引き継ぐ印のある場合だけを見る
- G4(INFO): 印の `detail` の escape は固定されていない(O32)。どの文の出力にも出ないので、影響は `director-wait.json` を読み込んだ値だけ。P4 で埋まる
- G5(INFO): 見張りを読む 5 秒の間隔(`directorWaitWatchProbeInterval`)は、どのテストも固定していない。help は「Every 5 s」と書き、テストはどれも間隔を短く上書きするので、定数を変えても help との食い違いはテストで見つからない。30 秒の方は `TestDirectorWaitLine_HelpDescribesTheWait` が照合している
- G6(INFO): 通らない block のうち、2 件以上の open の件の detail(`:672`)、回の settle の途中の ctx の終わり(`:531-533`)、running、stopping、stopped 以外の見張りの印(`:708-709`)、最後の受け皿の文(`:725`)はテストがない。どれも文か、すぐ終わる道
- G7(INFO): C07 は等価だが、`reraiseSignal` のコメント「The caller's signal.Notify must be stopped before it runs」(`org_wait_director.go:124-125`)は、コードに要る条件より強い。`signal.Reset` が Notify を外す
- tech-debt の `docs/tech-debt/README.md:279`(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の揺れ)のきっかけ「次にこのテストが落ちたとき」が、この回の go1.25.8 の全体の実行で起きた。この計画の範囲の外なので、行の更新か直しを後続にする
- Linux の go と、claude の SIGKILL のあとの漏れた wait(計画の B)は試していない。後者は計画の Non-goals で 8b-2c に回してある
- insight event は、この task の指示(テストの report と evidence だけを書く)により追記していない

## Cycle 2 (8a33364b..3add2caf)

- Date: 2026-10-19(JST 17:29〜18:45)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 2 回目
- Scope: HEAD 3add2caf(base e38b89a9)。cycle 1 のあとの変更は 38cac6f1(`begin` が `director-wait.lock` の中で `director.jsonl` を読み直し、今の登録でなければ何も書かずに `ErrDirectorWaitNotCurrent`、読めなければ何も書かずにその誤りで返る。test seam `directorWaitBeforeBeginLock`)、b3b692e1(G1〜G5 と G7 のテスト、`DirectorWaitWatchProbeInterval` の export)、6086af65(`begin` を `decideStart` と `noteStartWatch` に分けた。動きは変えていない)と文書のコミット。behavioral test だけを実行した
- テストもコードも変えていない。mutant と probe と canary は scratchpad に置き、`go test -overlay` で読ませた。`git status --porcelain` は実行の前もあとも空だった。Go の実行は 1 つずつ順に打ち、パッケージ全体の実行は run-test.sh の中の 1 回だけだった
- 負荷: ほかのセッションの負荷で、load average は 7〜94 だった。こちらから負荷は足していない
- Evidence: `docs/evidence/test-2026-10-19-org-director-wait.md` の「Cycle 2」(C2-1〜C2-5)と、`docs/evidence/test-2026-10-19-org-director-wait-mutation.txt` の「Cycle 2」(mutant 30 個の表、殺したテストの最初の assertion、cli での再実行、パッケージ全体を流し直さなかった根拠、probe の source と結果、生き残りの分類)。run-test.sh の全ログは `docs/evidence/verify-2026-10-19-082913.log`(gitignore の対象)

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(changed。golang を選んだ。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(cycle 1 と同じ) | 1,869 秒、rc 0 |
| `go test ./internal/org/ -run 'DirectorWait\|WaitInbox\|CheckDirectorWaitID\|ReadDirectorWait' -count=10` | top-level 370(37 本 × 10) | 370(PASS の行は subtest を含め 870) | 0 | 0 | 57 秒 |
| `go test -race ./internal/org/ -run 'DirectorWait\|WaitInbox' -count=5` | top-level 185(37 本 × 5) | 185(PASS の行 435) | 0 | 0 | 45 秒、DATA RACE 0 |
| `go test ./internal/cli/ -run 'OrgWait\|OrgDirector\|DirectorWaitLine\|ExitCode' -count=3` | top-level 345(115 本 × 3) | 345(PASS の行 957) | 0 | 0 | 384 秒 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -run 'DirectorWait\|WaitInbox\|OrgWait\|DirectorWaitLine' -count=1` | top-level 67(org 40、cli 27) | 67(PASS の行 156) | 0 | 0 | 36 秒 |
| canary を PATH の先頭に置いて cycle 2 で足した 7 本 | 7 | 7 | 0 | 0 | canary の呼び出し 0(感度の確かめつき) |
| mutation(`go test -overlay`、`director_wait.go` だけ) | mutant 30 | 赤 23 | 生き残り 7(うち 5 は probe で赤、2 は等価) | - | org 10〜80 秒、cli 56〜81 秒(1 個ずつ順に) |

- run-test.sh の shell は 41 本すべてが `OK` で終わり、`FAIL:` の集計の行はどれも `FAIL: 0` だった。Skipped は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の 1 件
- run-test.sh の Go は cli(679.4 s)、config(1.8 s)、org(836.6 s)を流し直し、insights、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。cli と org が cycle 1(287 s と 453 s)より長いのは負荷のためで、終わったときの load average は 56 だった
- cycle 2 で足した 7 本(org 6 本、cli の `TestDirectorWaitLine_TheWaitHelpSaysTheIntervalsOfTheOrgLayer`)は、4 つの絞った回で 60、30、3、7 回 PASS し、落ちた回はない
- 揺れ: この cycle では 1 件もなかった。cycle 1 の go1.25.8 の回で落ちた `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` と、知られた揺れの `TestRegate_FetchFailures/the_fetch_runs_past_the_bound` は、run-test.sh の org パッケージ全体の回で通った(パッケージが ok)
- 隔離(AC9): canary の呼び出しは 0。canary が先頭にあることは `command -v` と、`sh -c` から 1 回打って記録に残ることで確かめた。既定の台帳と worktree の `.harness/state/org` は実行のあとも存在せず、`.git/ralph/` の下で 17:29 以降に変わったファイルはなかった

### cycle 1 の穴の扱い

| 穴 | cycle 2 のテスト | 確かめ |
| --- | --- | --- |
| G1(印が消えたあとに書き直さない) | `TestWaitInboxForDirector_AMarkRemovedWhileTheWaitRunsIsNotWrittenAgain` | cycle 1 の O02、O35 を今のコードで作り直した R02、R35 が赤 |
| G2(ロックを取れない開始の引き継ぎ) | `TestWaitInboxForDirector_AStartThatCannotTakeTheLockStillCarriesOverTheWatch` | O31 と同じ B09 と、条件を逆にした B10 が赤 |
| G3(settle を切られた開始は `note:` を出さない) | `TestWaitInboxForDirector_ACancelDuringTheStartSettleWritesNoNote` | O28 と同じ B21 が赤 |
| G4(detail の escape) | `TestWaitInboxForDirector_TheDetailIsEscaped` | O32 と同じ R32 が赤 |
| G5(5 秒の間隔) | `TestDirectorWaitLine_TheWaitHelpSaysTheIntervalsOfTheOrgLayer` | 定数と help の食い違いは捕まる。ゼロ値の既定が定数を使うことは捕まらない(下の G11) |
| G6(通らない block) | なし(`docs/tech-debt/README.md` の 308 行に残っている) | 通らない block は 9 個。cycle 1 の一覧から `begin` のロックを取れない道と `sayLost` の印がない道が消えた |
| G7(`reraiseSignal` のコメント) | コメントの直し | テストの対象ではない |

### Coverage(cycle 2)

- org の絞った回(`-run 'DirectorWait|WaitInbox|CheckDirectorWaitID|ReadDirectorWait'`)で、`internal/org/director_wait.go` の 27 関数のうち 19 が 100%。`begin`、`decideStart`、`noteStartWatch`、`directorWaitCheckCurrent`、`sayLost`、`sayWriteFailed` は 100%。`loop` 97.6%、`probeAgain` 92.9%、`WaitInboxForDirector` 87.5%、`directorWaitWatchState` 87.5%、小さな helper 3 つが 66.7%、`newDirectorWaitRun` 54.5%(cli の回では 100%)
- 両方の回で通らない block は 9 個で、evidence の C2-4 に並べた: nil の Inbox と空の StateDir の guard、`InboxPollInterval` の既定、回の settle の途中の ctx の終わり(`:587-589`)、2 件以上の open の件の detail(`:728`)、running、stopping、stopped 以外の見張りの印(`:764-765`)、最後の受け皿の文(`:781`)、`directorWaitTSOf(nil)`、`directorWaitMarkTS("")`。どれも cycle 1 の G6 にあったもの

### Mutation(cycle 2)

mutant は 30 個で、対象は cycle 2 のコードに絞った(`begin` のロックの中の読み直し、断りの 2 つの道、ロックを取れないときの道、`decideStart`、`noteStartWatch`、`WaitInboxForDirector` の `begin` の誤りの扱い)。ほかに cycle 1 の生き残り 3 個(O02、O35、O32)を作り直し、self-review の I1 の既定値を 2 個足した。org の絞った回で 23 個が赤になり、生き残った 7 個は cli の絞った回でも生き残った。7 個のうち 5 個は probe(P5〜P8)で赤になり、2 個は等価だった。

| 判断 | mutant | 結果 |
| --- | --- | --- |
| ロックの中の読み直しで、今の登録かを確かめる | B02(確かめを外す)、B06(断りを返さず、ロックの外で続ける)、B07(断りを捨てる) | 赤(`_ARegistrationReplacedBeforeTheStartTakesTheLock`) |
| ロックの中の読み直しで、`attached_at` を比べる | B01(`&w.reg` を nil に) | 生き残り(org と cli)。P5 で赤(G8) |
| ロックの中で `director.jsonl` を読めない | B03(読めないのを登録なしとみなす)、B04(prefix を落とす)、B05(誤りをロックから返して続ける) | 赤(`_ARegistrationThatCannotBeReadUnderTheStartsLockWritesNothing`) |
| `begin` の誤りを `WaitInboxForDirector` が返す | B13(誤りを捨てる)、B14(断ったときに end を書く) | 赤(上の 2 本) |
| ロックを取れない開始 | B09(`decideStart` を呼ばない)、B10(条件を逆に)、B25(引き継ぎで `noWatch` を立てない) | 赤(`_AStartThatCannotTakeTheLockStillCarriesOverTheWatch` ほか) |
| 開始の書き込みが失敗したことを言う | B12(`sayWriteFailed` を外す) | 生き残り。P7 で赤(G9) |
| 書けたら `written` を立てる | B11 | 赤(`_AMarkRemovedWhileTheWaitRunsIsNotWrittenAgain`) |
| `decideStart` | B15(`prevWaitID` を残さない)、B16(どの印からも引き継ぐ)、B17(引き継いだ値を使わない)、B18(開始の印に `watch_accepting` を書かない) | 赤 |
| 〃 | B19(開始の印に `seen_at` を書かない) | 生き残り。P8 で赤(G10) |
| 〃 | B08(ロックの中で `decided` を立てない)、B20(`wasAccepting` でなく `w.accepting` を返す) | 等価(下) |
| `noteStartWatch` | B21(settle を切られた開始を決まったとみなす)、B22(受け付けている見張りでも分岐に進む)、B23(`note:` の事実を空の見張りから作る)、B24(probe の結果を `w.watch` に残さない) | 赤 |
| ゼロ値の既定 | N1(5 秒を 1 時間に)、N2(30 秒を 1 時間に) | 生き残り(org と cli)。P6 で赤(G11) |
| cycle 1 の生き残り | R02、R35、R32 | 赤(G1、G4 のテスト) |

- 生き残りはパッケージ全体で流し直していない。代わりに、変異させた関数に届くテストがすべて 2 つの絞り込みに入ることを grep で確かめた。`WaitInboxForDirector`、`newDirectorWaitRun`、`DirectorWaitParams{` をコードで使うのは `internal/org/director_wait.go` と `internal/cli/org_wait_director.go` だけで、wait を走らせる org のテストは `director_wait*_test.go` の 3 ファイル、cli のテストは `org_wait_director_test.go` と `org_director_wait_line_test.go` だけで、これらのファイルのテスト関数はどれも絞り込みに当たる(evidence の mutation の記録の C2-3)
- probe 4 本(P5〜P8)は HEAD で `-count=10` の 60 行すべて PASS、`-race -count=3` も PASS(DATA RACE 0)。各 mutant で落ちたのは、その mutant に向けた probe だけだった。probe はコミットしていない
- 殺した 23 個は、落ちた assertion を 1 つずつ読み、時間切れや揺れではなく狙った挙動の違いで落ちたことを確かめた(evidence の mutation の記録の C2-2)

### self-review の I1 の確かめ

I1 の 2 つの指摘は、どちらも生き残る mutant で確かめられた。前半は INFO、後半は LOW の穴になる。

- 前半(ゼロ値の既定): `newDirectorWaitRun` の既定の 5 つの block は、org の回では通らないが、cli の回では通る(`newDirectorWaitRun` は cli の回で 100%)。「テストで 1 度も走らない」は org の回だけに当てはまる。ただし値はどのテストも見ていない。N1(`DirectorWaitWatchProbeInterval` の代わりに 1 時間)と N2(`DirectorWaitMarkInterval` の代わりに 1 時間)は org と cli の両方で生き残り、白箱の P6 で赤になった。既定の代入は名前の付いた定数を入れる 1 行ずつなので、危険は小さい(G11、INFO)
- 後半(`begin` の確かめの `&w.reg` を nil に): 本当の穴で、cycle 1 の cross-review の欠陥が同じ id のときにだけ戻る。B01 は org と cli の両方で生き残った。P5 は、d1 の wait が最初の確かめを通ったあと、ロックを取る前に、登録が同じ d1 のまま新しい `attached_at` になり(`director.jsonl` を消して attach し直す行と、手で書き換える行)、新しいセッションの d1 の wait が印を書く場合を作る。HEAD では古い wait は `ErrDirectorWaitNotCurrent`(「not the d1 attached 2026-10-19T08:00:00Z that this wait started for」)で返り、新しい wait の印は byte で変わらず、新しい wait の 1 分後の書き直しも届く(10/10、`-race` 3/3)。B01 では、古い wait がロックの中の確かめを通って新しい wait の印を自分の印で上書きし、最初の `refresh` で `not_current` の end を書いて返る。新しい wait は書いたあとなので別の `wait_id` を見て持ち主でなくなり、二度と印を書かない。印は、今の登録の wait が走っているのに古い wait の `not_current` を見せ続ける(evidence の C2-4 に出力)
  - 起こりうる場面: `nextID` は台帳の最大の番号の次なので、`attach --replace` や detach のあとの attach は新しい id になる。同じ id に新しい `attached_at` が付くのは、`director.jsonl` が消されて attach し直したとき(番号が d1 からやり直しになる)か、手で書き換えたときだけで、しかも `begin` の probe のあいだ(見張りが stale なら settle の最長 2 秒、そうでなければごく短い)に限られる。そのため LOW とした(G8)
  - 計画の Test plan の edge case「登録の `attached_at` が手で書き換えられた」は、30 秒ごとの `refresh` の道では `_ARegistrationThatChangesEndsTheWait` の行が固定しているが、`begin` のロックの中の読み直しの道では固定されていない
  - HEAD のコードは正しく、直すのはテストだけでよい。P5 をそのまま `director_wait_owner_test.go` に足せば埋まる

### Test gaps(cycle 2)

- G8(LOW): `begin` のロックの中の読み直しが `attached_at` を比べることを、どのテストも固定していない(B01、上の I1 の後半)。P5 で埋まる
- G9(LOW): 開始の書き込みが失敗したことを、開始のときに言うことが固定されていない(B12)。失敗する開始を見るテストはどれも障害を残したままなので、あとの書き込み(20 ms の `refresh` か終わり)が同じ 1 行を 1 回だけ出し、行の数も 1 のままになる。mutant では、その行は最初の `refresh`(本番では 30 秒後)か終わりにしか出ず、2 回目の書き込みが通れば出ない。P7(印の間隔を 1 時間にし、件を出す前に行が出ることを見る)で埋まる
- G10(LOW): 開始の印の `seen_at` が固定されていない(B19)。どのテストも、あとの書き込みが書き直した `seen_at` しか読まない。mutant では最初の `refresh` までの 30 秒、`director status` が「last wrote its mark at」のあとに時刻を出さず、`readOrgDirectorWait` は読めない `seen_at` を古いと判定しないので、そのあいだに SIGKILL された wait は「has not written it since」の付かない走っている wait として残り続ける。P8 で埋まる
- G11(INFO): ゼロ値の既定が `DirectorWaitWatchProbeInterval` と `DirectorWaitMarkInterval` を使うことが固定されていない(N1、N2、I1 の前半)。P6 で埋まる
- G6(INFO): cycle 1 から続く通らない block 9 個。tech-debt の 308 行のとおり
- 等価: B08 は、ロックを放したあとに `decideStart` をもう 1 回ロックなしで呼ぶ。読むのは自分が書いたばかりの印なので、`watch_accepting` と `noteStartWatch` の判断は変わらず、`prevWaitID` は自分の `wait_id` になるが、`written` が立つと `owns` は `prevWaitID` を見ない。違いが出るのは、ロックを放してから読み直すまでのあいだに別の wait が印を書いたときだけで、ロックのあとの seam がないとテストで作れない。B20 は、戻り値が `noteStartWatch` にしか届かず、`observed` が true の開始は最初の分岐が取り、false なら `w.accepting` が `wasAccepting` に等しいので、挙動は変わらない
- self-review の I2(ロックを取れない開始と `attach --replace` が重なる場合)は、self-review が上げないと決めたので試していない
- G8〜G11 は probe 4 本(evidence の mutation の記録の C2-4 に source)をそのままテストにすれば埋まる。cap 2 の 2 回目なので、足すか tech-debt に書くかは lead が決める。この /test ではテストを足していない

## Verdict

- Verdict: pass(ブランチ全体、`main...3add2caf`)
- Pass: cycle 2 の run-test.sh(shell 41 本と Go 8 パッケージ、1,869 秒、rc 0)、絞った 4 つの回(org `-count=10` 370 本、org `-race -count=5` 185 本で DATA RACE 0、cli `-count=3` 345 本、go1.25.8 の org と cli 67 本)、cycle 2 で足した 7 本(すべての回で PASS、canary の呼び出し 0)、cycle 2 のコードの mutant 30 個のうち 23 個が赤(生き残り 7 個のうち 5 個は probe 1 本ずつで赤になる LOW 3 件と INFO 1 件の穴、2 個は等価)。cycle 1 の生き残りのうち 5 個(O02、O35、O31、O28、O32)は、cycle 2 で足したテストで赤になった
- cycle 1 の verdict(履歴): pass。run-test.sh、go1.25.8 の全体(記録済みの揺れ 1 件のほかすべて PASS。その 1 件は単独 10/10、org パッケージの再実行も PASS)、`-race`(663 本、DATA RACE 0)、新しいテスト 44 本、mutant 56 個のうち 50 個が赤、canary の呼び出し 0
- Fail: なし(この差分によるもの)。この cycle では揺れもなかった
- Blocked: なし
- 残る穴: G8、G9、G10(LOW)、G11、G6(INFO)。どれも HEAD の挙動は正しく、テストが固定していないだけで、マージを止めるものではない
