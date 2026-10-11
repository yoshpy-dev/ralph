# Test report: org-cross-watch

- Date: 2026-10-12(cycle 2。JST 00:08〜00:40)
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み、digest d0a6f2735552)
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 2(`cycle-count.json` は 2、上限 2)。cross-review cycle 1(triage 7a7e4ea7)の ACTION_REQUIRED #1・#2 の直し(49d8d605)と、self-review cycle 2 の直し(d2a750ec)のあとの実行。cycle 1 の結果は末尾の「付録」に残してある
- Scope: HEAD b0a226a9(base 577ebeaf)。cycle 1 の test(ed1636cb)のあとにコードを変えたのは 49d8d605 と d2a750ec で、`internal/org/watch.go`、`internal/org/watchalive.go`、`internal/cli/org_watch.go`、`internal/cli/status.go` と、そのテスト 4 本(`git diff --stat ed1636cb..HEAD -- '*_test.go'` は +415/-14)。04bcb342 は文書だけ。behavioral test だけを実行した(静的解析は /verify の担当)。依頼の流れのとおり、`run-test.sh` と `go test`(`-race` を含む)、AC とテストの対応の更新、`go test -overlay` の mutation、HEAD のビルドでの実機確認を行った
- Evidence: `docs/evidence/test-2026-10-12-org-cross-watch-c2.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore -v` で `.gitignore:58` に当たる)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-11-150821.log`(時刻は UTC)。実機確認は `docs/evidence/org-cross-watch-live-2026-10-11.md` の「cycle 2」の節(コミットする)
- テストは足していない。mutant の写しと probe のテストは scratchpad の `ocw2/` に置き、`go test -overlay` で使った。worktree で書き換えたのは、このレポート、実機の記録、insight event だけ

## Latest run: cycle 2(HEAD b0a226a9)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(HEAD、変更言語の scope) | shell 40 本(PASS の行 3,962)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 6 分 46 秒、rc 0 |
| `go test ./... -count=1 -v -cover` | top-level 1,382(org 561、cli 543)、subtest 1,454 | top-level 1,379、subtest 1,454 | 0 | 3(cycle 1 と同じ 3 本) | 88 s(cli 85.8 s、org 27.9 s)、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -count=1` | 2 パッケージ | 2 | 0 | 0 | 100 s(org 31.6 s、cli 95.0 s)、rc 0 |
| `go test -race -run 'Watch\|Probe\|Alive\|Stop\|BeforeStop\|Status' -count=5 -v`(org と cli) | top-level 1,055(org 131 本 × 5、cli 80 本 × 5) | すべて | 0 | 0 | 129 s、rc 0。`DATA RACE` は 0 件 |
| `go test -coverprofile`(org、cli) | 2 パッケージ | 2 | 0 | 0 | 93 s、rc 0 |
| probe のテスト(`-overlay` で足す 2 本、HEAD で `-count=3` と `-race -count=3`) | 12 | 12 | 0 | 0 | 7 s |
| mutation(`go test -overlay`、HEAD のテスト) | 29 件 | red 27 件 | 生き残り 2 件(下の節) | - | 1 件 22〜124 s |
| 実機確認(macOS 2 本) | 下の「実機確認」 | すべて期待どおり | 0 | - | 合わせて 40 秒ほど |

- shell の件数は suite ごとに PASS・FAIL・SKIP の行を数えた。集計の行は、`PASS: 29` の形も `PASS: 47 / 47` の形も、`SKIP: 0` も除いた。40 本すべてが `OK` で終わり、FAIL の行は 0。suite ごとの件数は、cycle 1 のログ(`docs/evidence/verify-2026-10-11-094617.log`)を同じ awk で数えた結果と一致した(`diff` が空)
- cycle 1 の報告の 3,972 は、`PASS: 47 / 47` の形の集計の行 10 行を数えに入れていた。正しい数は cycle 1 も cycle 2 も 3,962 で、テストの数は変わっていない。付録の数は書き換えていない
- Skipped の 1 件は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)
- `run-test.sh` の言語の scope は、cycle 1 と同じく `scripts/ralph-config.sh` が言語に分類されないので full に落ちた(`Language scope: full fallback`)。中の `go test ./...` は `internal/org` だけを実行し、ほかはキャッシュの結果だった。そのため 2 行目で全パッケージを `-count=1` で流し直した。go は 1.26.0(darwin/arm64)
- Go の top-level の件数は、org 561 と cli 543 が SKIP を含む数で、PASS は org 560、cli 543。Skipped の 3 件は cycle 1 と同じ(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` と、`internal/scaffold` の前からある 2 件)
- top-level のテストは cycle 1 より 11 本多い。名前の一覧を `comm` で比べ、消えたテストがないことを確かめた。増えたのは 4aaef4ed の 5 本(付録の Test gaps の追記にあるもの)と、49d8d605 の 6 本(org 5、cli 1)で、11 本とも通った
- 実行中、ほかのセッションの負荷で load average は 6〜11(12 コア)だった。既知の flaky なテスト(tester の MEMORY.md の 3 本)はどれも落ちなかった

### Coverage

- Statement: `internal/org` 94.1%、`internal/cli` 86.3%、`internal/config` 93.2%(cycle 1 は 94.0%、86.3%、93.2%)。ほかは insights 86.1%、driver 93.1%、protocol 97.9%、scaffold 75.7%、upgrade 91.2%
- Function(49d8d605 と d2a750ec が変えた関数): 100% は `RunWatchAll`、`finish`、`drainBeforeStop`、`stopRequested`、`ReadWatchStopRequest`、`readWatchStopRequestAt`、`runOrgWatchAll`、`orgWatchLine`、`buildStatusOrgWatchJSON`。100% でないのは `takeStopRequestAtStart` 85.7%(取り出しの rename が「頼みがない」以外の理由で失敗したときの stderr の行と、取り出したファイルを消せないときの行)、`takeWatchStopRequest` 85.7%(その rename の失敗を包む行)、`stopAndRelease` 77.8%(stopped の印を書けない、ロックを放せないときの行。cycle 1 の `finish` と同じ枝)、`runOrgWatchStop` 85.7%(cycle 1 と同じ)、`orgWatchMarkDetails` 91.7%(印に pid も時刻もないときの空文字)
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 取り出しの rename の失敗の枝は、mutation で生き残った R5 の枝と同じ(下の節)。`drainBeforeStop` は 100% だが、mutation の B2b が生き残った

### AC とテストの対応(cycle 1 からの変更)

cycle 1 の表(付録)から変わった行だけを書く。AC1〜AC8、AC11、AC12、AC12b、AC15〜AC17 は、コードもテストも cycle 1 のあと変わっていないので、付録の対応がそのまま使える(4aaef4ed で AC7・AC11・AC12 に足したテストは、付録の Test gaps の追記にある)。テストはすべて HEAD で通った。「実機」は上の「実機確認」、「probe」は下の「Probe のテスト」(コミットしていない)の番号。

| AC | 固定するテスト(cycle 2 で足したもの、変わったもの) | 補い |
| --- | --- | --- |
| AC9 2 つ目の見張りの拒否、終わればロックが放される | 足した: `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning`(終わり方 3 通り。BeforeStop の間、2 つ目の `AcquireWatchLock` と `RunWatchAll` が `ErrWatchRunning` になり、断られた側は BeforeStop を呼ばない)、`TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`(CLI。watcher の呼び出しが終わるまで、2 つ目の `--once` が断られる)、`TestRunWatchAll_APanicInBeforeStopStillWritesTheStoppedMarkAndReleasesTheLock`(panic でもロックが放される)。cycle 1 の 3 本も通る | 実機 1(HEAD のビルド。2 つ目の `--all --once` が pid 41370 を名指しして rc 1。`kill -9` のあとの `--detach` が新しい pid で通る) |
| AC10 印の書き込み、SIGINT と SIGTERM | 足した: 上の BeforeStop の表(drain の間、印の `ts` が interval ごとに書き直され、終わると stopped)と panic のテスト。4aaef4ed の `TestRunWatchAll_WritesTheStoppedMarkBeforeReleasingTheLock`。cycle 1 の 3 本も通る | probe 10(drain の書き直しが、ロックを放したあとに着かない)、実機 1(間隔 2 秒で `ts` が 15:13:46 から 15:13:50 に進む) |
| AC13 `--stop` | 足した: `TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`(watcher の呼び出しの間の `--stop` は「動いていない」と言わずに待ち、見張りが終わってから `stopped` を返す)。変わった: `TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal` が、時間切れの文の新しい言い回し(watcher の呼び出しの待ち)も見る | 実機 1(`--stop` が 0.436 s で rc 0。台帳に `watch-stop.json` も `.watch-stop.json.taken` も残らない) |
| AC13b 古い頼みに従わない、シグナルを送らない | 足した: `TestRunWatchAll_AStopRequestWrittenWhileTheStartTakesAnOldOneIsKept`、`TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed`、`TestTakeWatchStopRequest_MovesTheRequestSoALaterOneIsANewFile`、4aaef4ed の `TestRunWatchAll_StartedAtIsTakenBeforeWaitingForTheLock`。変わった: `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` に読めない頼みの行と、`.taken` が残らないこと、stderr が空であることの確認が入った | probe 11(取り出しの rename が失敗したときの stderr)、実機 1 の手順 6(`requested_at` が 2026-10-01 の頼みを手で置いてから `--detach`。頼みは起動のときに消え、見張りは止まらずに動き続ける)、実機 2(残った `.watch-stop.json.taken`) |
| AC14 `ralph status` の行と `org_watch` | 変わった: status のテスト 6 本が `mark updated <ts>` と `--json` の `updated_at` を見る(d2a750ec で `last cycle` と `last_cycle_ts` から変わった)。mutant の T1a・T1b・T2 で落ちる | 実機 1(`org watch: running (pid 41263, mark updated …)`、kill のあとの `not running (a running mark is left from pid 41263, mark updated …)`、止めたあとの `not running (stopped at …)`。`--json` の `org_watch.updated_at`) |

behavioral test のない AC は、cycle 1 と同じく AC17(文書)だけである。

### cross-review の直しを固定するテスト

| 直し | テスト | 落とす mutant(下の表の ID) |
| --- | --- | --- |
| #1 watcher の後始末の間もロックを持ち、印は running のまま(49d8d605) | `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` | B1、B2、B2d、B4、B5、B6 |
| 同上 | `TestRunWatchAll_APanicInBeforeStopStillWritesTheStoppedMarkAndReleasesTheLock` | B3、B5 |
| 同上 | `TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`(CLI) | B1c、B7、B9 |
| 同上 | `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall`(cycle 1 からある) | B8 |
| #2 起動のときの取り出しを 1 回の rename にし、新しい頼みを消さない(49d8d605) | `TestRunWatchAll_AStopRequestWrittenWhileTheStartTakesAnOldOneIsKept` | R1、R1b、R3 |
| 同上 | `TestTakeWatchStopRequest_MovesTheRequestSoALaterOneIsANewFile` | R1 |
| 同上 | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed` | R1、R1b、R2、R2b、R2c、R3、R3b、R4 |
| 同上 | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`(行と確認を足した) | R2、R3、R3c、R4、R6 |
| 同上 | `TestRunWatchAll_StartedAtIsTakenBeforeWaitingForTheLock`(4aaef4ed) | R4 |
| d2a750ec の言葉 | `TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal` | S1 |
| 同上 | status のテスト(`TestStatusCmd_OrgWatchLineAndJSONPerWatchState` ほか) | T1a(4 本で落ちる)、T1b(1 本)、T2(2 本) |

### Mutation

`scratchpad/ocw2/mut2.py` は cycle 1 の `ocw-test/mut.py` と同じ作りで、`-timeout 180s` を足した。書き換える文字列がファイルに 1 回だけあることを、当てる前に確かめた。org 層の mutant は `internal/org` の全体を走らせ、生き残ったら `internal/cli` の全体も走らせた。CLI の mutant は `internal/cli` の全体を走らせた。29 件ともコンパイルが通った(R5 と R5b は、使われなくなる import を残すために `false &&` と `true ||` の形にした)。

依頼の 7 点と、その mutant:

| 依頼 | mutant | 結果 |
| --- | --- | --- |
| drain をロックを放したあとにする(直す前の順) | B1(org の順を入れ替える)、B1c(CLI を直す前の形に戻す: BeforeStop を使わず、`RunWatchAll` が返ってから watcher を取り消して待つ) | どちらも red |
| drain の間に印を書き直さない | B2 | red。近い書き換えのうち B2b は生き残り(下)、B2c と B2d は red |
| `stopAndRelease` を defer しない(panic の経路) | B3 | red |
| 取り出しを rename でなく、読んでから消す | R1(rename をやめ、`watch-stop.json` をその場で読んで、あとで消す)、R1b(直す前の `dropOldStopRequest` に戻す) | どちらも red |
| 開始より後の頼みを元に戻すのを飛ばす | R2 | red |
| `.taken` を消さない | R3(いつも)、R3b(戻せなかったとき)、R3c(読めない頼みのとき) | 3 件とも red |
| `--stop` の時間切れの言い回し | S1(直す前の文に戻す) | red |

全体は 29 件で、27 件が red、2 件(B2b、R5)が生き残った。「落としたテスト」は、最初に落ちたパッケージのもの。

| ID | 書き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| B1 | `finish`: `stopAndRelease` を先に呼び、そのあと drain(直す前の順) | red | `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` |
| B1c | CLI: BeforeStop を使わず、`RunWatchAll` が返ってから watcher を取り消して待つ(直す前の形) | red | `TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds` |
| B2 | drain の間、印を書き直さない | red | `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` |
| B2b | drain が、印を書き直す goroutine の終わりを待たずに返る | **生き残り**(org と cli の全体。`-race` では落ちる、下の注) | (probe 10 が落とす) |
| B2c | B2b に加えて、goroutine が止まる前にもう 1 回印を書く | red(注のとおり、落とし方は時間の重なりによる) | `TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark`(cli) |
| B2d | drain の間、印を 1 回しか書き直さない | red | `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` |
| B3 | `finish`: `stopAndRelease` を defer しない | red | `TestRunWatchAll_APanicInBeforeStopStillWritesTheStoppedMarkAndReleasesTheLock` |
| B4 | ロックを取れずに断った `RunWatchAll` も BeforeStop を呼ぶ | red | `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` |
| B5 | BeforeStop を呼ばない | red | `TestRunWatchAll_APanicInBeforeStopStillWritesTheStoppedMarkAndReleasesTheLock`, `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` |
| B6 | BeforeStop を 2 回呼ぶ | red(テストの BeforeStop が channel を 2 回閉じて panic。`watch_all_test.go:1156`) | `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` |
| B7 | CLI の BeforeStop: `--once` でも watcher を取り消す | red | `TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds` |
| B8 | CLI の BeforeStop: 長く動く見張りでも watcher を取り消さない | red | `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` |
| B9 | CLI の BeforeStop: watcher を待たない | red | `TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds` |
| R1 | 取り出し: rename をやめ、`watch-stop.json` をその場で読み、あとでそのパスを消す | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed`, `TestRunWatchAll_AStopRequestWrittenWhileTheStartTakesAnOldOneIsKept`, `TestTakeWatchStopRequest_MovesTheRequestSoALaterOneIsANewFile` |
| R1b | 起動のときの扱いを直す前の `dropOldStopRequest` に戻す(読む、seam、`watch-stop.json` を消す) | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed`, `TestRunWatchAll_AStopRequestWrittenWhileTheStartTakesAnOldOneIsKept` |
| R2 | 開始より後の頼みを元に戻さない(覚えるだけで、取り出したファイルは消す) | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed`, `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| R2b | `stopTaken` を立てない | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed` |
| R2c | `stopRequested` が `stopTaken` を見ない | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed` |
| R3 | `.taken` をいつも消さない | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed`, `TestRunWatchAll_AStopRequestWrittenWhileTheStartTakesAnOldOneIsKept`, `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| R3b | 戻せなかったとき `.taken` を消さない | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed` |
| R3c | 読めない頼みのとき `.taken` を消さない | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| R4 | 開始より後の頼みも、古い頼みと同じく消す | red | `TestRunWatchAll_AStopRequestThatCannotBePutBackIsStillFollowed`, `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`, `TestRunWatchAll_StartedAtIsTakenBeforeWaitingForTheLock` |
| R5 | 取り出しの rename が「頼みがない」以外の理由で失敗しても、stderr に何も出さない | **生き残り**(org と cli の全体) | (probe 11 が落とす) |
| R5b | 頼みがないときも stderr に 1 行出す | red | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated` |
| R6 | 読めない頼みも、開始より後の頼みとして戻して従う | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S1 | `--stop` の時間切れの文を直す前に戻す(watcher の呼び出しの待ちを言わない) | red | `TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal` |
| T1a | status: running と stale の行を `last cycle` に戻す | red | `TestStatusCmd_EmptyRosterStillShowsTheOrgWatch`, `TestStatusCmd_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad`, `TestStatusCmd_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml`, `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` |
| T1b | status: not running の行を `last cycle` に戻す | red | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` |
| T2 | `--json` の key を `last_cycle_ts` に戻す | red | `TestStatusCmd_EmptyRosterStillShowsTheOrgWatch`, `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` |

#### 生き残った mutant

- **B2b**: `drainBeforeStop` の doc は「印を書き直す goroutine が終わるのを待ってから返るので、最後の書き直しは stopped の印より前に来る」と書く。`-race` なしでこれを見分けるテストはない。待たずに返ると、BeforeStop が返ったときに書きかけだった書き直しが、`stopAndRelease` が stopped を書いてロックを放したあとに着きうる。新しい見張りがすぐロックを取っていれば、その running の印を古い見張りの印が上書きする(cycle 1 の L6 と同じ形の危険)。既存の `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` は、`-race` をつけると `w.mark` への書き込みの DATA RACE で落ちる(`-race -count=3 -run BeforeStop` で、報告 2 件、落ちたのは 1 回)。ただし CI と `run-test.sh` は `-race` を使わない(`.github/workflows/` と `packs/languages/golang/verify.sh` に `-race` はない)。probe 10 は HEAD で `-count=3` と `-race -count=3` のどちらも通り、B2b と B2c では 3 回とも落ちる
- **R5**: 取り出しの rename が「頼みがない」以外の理由で失敗したときの stderr の行(`watch.go:706-708`、`watchalive.go:237`)は、どのテストも通らない(coverage 0)。この行が消えても動きは変わらず、古い頼みが取り出せずに残ったことを知らせる行がなくなるだけ。probe 11(`.watch-stop.json.taken` に空でないディレクトリを置いて rename を失敗させる)は HEAD で通り、R5 で落ちる

注:

- B2c は org の全体では生き残り、cli の `TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark` で落ちた。落ちた理由は、遅れて着いた印の書き込みがテストの TempDir の後始末と重なったこと(`TempDir RemoveAll cleanup: … directory not empty`)で、印の中身を見る assertion ではない。時間の重なりによる落ち方なので、毎回は落ちない。probe 10 は 3 回とも落とした
- 29 件を 2 本に分けて走らせた(1 本目は B1〜R1b、2 本目は R2〜T2)。実行中の load average は 6〜11。落ちたテストはどれも、書き換えた動きを名指しする assertion(B6 は panic、B2c は上の注)で落ちていることをログで見た。時間切れだけで落ちたものはない
- 依頼どおり、テストは足していない。足すなら probe 10 と probe 11 の 2 本で、それぞれ B2b(と B2c)と R5 を閉じる。どちらも scratchpad の `ocw2/probe/zz_probe_c2_test.go` にある。足すかは orchestrator が決める

### Probe のテスト

cycle 1 の probe 1〜9(付録)に続けて番号を振った。scratchpad の `ocw2/probe/zz_probe_c2_test.go`(`internal/org`)に書き、`go test -overlay` で足した(overlay は `probe-head.json`、`probe-B2b.json`、`probe-B2c.json`、`probe-R5.json`)。コミットしていない。

| # | probe | HEAD | 確かめたこと |
| --- | --- | --- | --- |
| 10 | `TestZZProbe_NoRefresherRewriteLandsAfterTheLockIsReleased` | 通る(`-count=3`、`-race -count=3`) | 時計の差し替え(`Org.Now`)で、BeforeStop が返る前に始まった印の書き直しを 300 ms 止める。`RunWatchAll` が返った直後に新しい見張りの running の印(pid 4343)を書き、500 ms 後もそのままであることを見る。HEAD は書き直しの終わりを待ってから stopped を書くので通る。B2b と B2c は、古い見張りの stopped の印(pid は test binary のもの)で上書きされて落ちる |
| 11 | `TestZZProbe_ATakeThatFailsIsSaidOnStderrAndTheRunGoesOn` | 通る(同上) | `.watch-stop.json.taken` に空でないディレクトリを置き、古い頼みの rename を失敗させる(darwin で確かめた)。HEAD は `watchdog: org: watch: take the stop request: …` を 1 行出して 3 サイクル回り、古い頼みはそのまま残る。R5 は stderr が空で落ちる |

### 実機確認

`docs/evidence/org-cross-watch-live-2026-10-11.md` の「cycle 2」の節にコマンドと出力を足した(cycle 1 の記録はそのまま)。HEAD b0a226a9 を `go build` したバイナリを scratchpad に置き、cycle 1 と同じ規則で動かした。台帳は scratchpad の下だけで、すべてのコマンドに `--state-dir` を付けた。manifest も受信箱も置かない(herdr・agmsg・osascript は呼ばれない)。ralph は `env -i` で空の PATH と scratchpad の HOME で起こした。利用者の herdr サーバー、`~/.config/herdr`、本物の台帳には触れていない。始める前と最後の `pgrep -fl 'org watch --all'` は何も返さなかった。

1. macOS(依頼の流れ): `--detach` は rc 0 で pid 41263 を出し、0.134 s で返った。`ralph status` は `org watch: running (pid 41263, mark updated 2026-10-11T15:13:45Z)`、`--json` の `org_watch` は `updated_at` を持つ(`last_cycle_ts` はない)。`kill -9 41263` のあと、status は `org watch: not running (a running mark is left from pid 41263, mark updated …)`、`--json` は `state: not_running`、`mark_state: running`。もう一度の `--detach --interval-seconds 2` は rc 0 で新しい pid 41370 を出し、4.5 s で印の `ts` が進んだ。そのあいだの `--all --once` は rc 1 で `(pid 41370, from …/watch-alive.json)` を名指しした。`--stop` は rc 0、0.436 s で返った。status は `not running (stopped at …)` で、台帳(`ls -la`)に `watch-stop.json` も `.watch-stop.json.taken` も残らなかった。続けて、`requested_at` が `2026-10-01T00:00:00Z` の `watch-stop.json` を手で置いてから `--detach` した。頼みは起動のときに消え(`.taken` も残らない)、見張りは 4.5 s 後も動いて `ts` を進めていた。`--stop` で止まった
2. macOS(足した確認): `.watch-stop.json.taken` が残った台帳を手で作った(取り出しと削除の間で見張りが落ちた形)。古い `watch-stop.json` と並んでいるときは、起動の取り出しが `.taken` を上書きしてから消すので、どちらも残らない。`.taken` だけのときは、見張りは普通に動くが、`.taken` は `--stop` のあとも残る(下の Test gaps)

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の top-level 1,368 本 | 通る | 名前の一覧で消えたものがないことを確かめ、全部が PASS |
| cycle 1 の生き残り 5 件(P2、X3、L6、A6、S6)を閉じた 4aaef4ed のテスト 5 本 | 通る | `-race -count=5` でも通る |
| 1 org のパルス、escalate・inbox、spawn・send・start・stop・disband | 通る | それらのコードとテストファイルは cycle 1 のあと変わっていない。全体の `-count=1` で通る |
| `ralph status` の既存の表示と `--json` の形 | 通る | `TestStatusCmd_*` 25 本。変わったのは org watch の行の言葉と key だけ |
| cycle 1 の実機の流れ(HEAD のビルドで) | 通る | 実機 1 |

### Test gaps

- B2b(drain で印を書き直す goroutine の終わりを待つこと)は、`-race` の実行でしか固定されていない。CI と `run-test.sh` は `-race` を使わない。probe 10 で閉じられる
- R5(取り出しの rename の失敗を知らせる stderr の行)はテストがない。probe 11 で閉じられる
- 残った `.watch-stop.json.taken` は、次に頼みのある起動まで消えない(実機 2)。読まれないファイルなので動きには関わらないが、台帳に残り続ける。テストはこの形を作らない。取り出しと削除の間は数 µs なので、そこに `kill -9` を当てることはせず、ファイルを手で置いて形を作った
- BeforeStop が返らない場合は、doc の前提(上限のある時間で返す)だけで、仕組みでは打ち切らない(verify の C2-3 と同じ)。テストは返る BeforeStop だけを使う
- 実機では seat を置かない規則なので、watcher の呼び出しの途中で `--stop` を打つ流れ(#1 の直しの本題)は、CLI のテスト(`TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds`、stub の claude)でだけ確かめた。後始末が 30 秒を超えて `--stop` が時間切れになる流れは、止まらない見張りのテストで文を見るだけ
- Linux は cycle 2 では走らせていない。`--detach` と setsid のコード(`runOrgWatchDetach`、`startDetachedProcess`)は cycle 1 のあと変わっていない
- cycle 1 の Test gaps のうち、本物の osascript の通知、本物の herdr が返らない場合、`--config` の `deadman_minutes` と status のずれ、同時の `--detach` の rc 1 は変わらない

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、Go 8 パッケージ)。全パッケージを `-count=1` で流し直して 0 fail(top-level 1,379 本が PASS、cycle 1 より 11 本多い)。`-race`(org と cli)と、watch・判定・止める頼み・BeforeStop・status のテストの `-race -count=5` も 0 fail で、DATA RACE は 0 件。cross-review の #1・#2 の直しと d2a750ec の言葉には、それぞれ直す前の形で落ちるテストがあり、依頼の 7 点の mutant はすべて落ちた(29 件のうち 27 件が red)。HEAD のビルドの実機では、`mark updated` と `updated_at`、`kill -9` のあとの判定と起動し直し、2 つ目の見張りの拒否、`--stop`、起動の前に置いた古い頼みの扱いが期待どおりで、`.watch-stop.json.taken` もプロセスも残らなかった
- Fail: 0
- Blocked: なし。生き残り 2 件(B2b は `-race` でだけ落ちる、R5 は stderr の 1 行)と Test gaps の残りは merge を止めない。probe 10・11 をテストに足すかは orchestrator が決める

## 付録: cycle 1 の結果(HEAD dd63a414 の時点)

cycle 1 の header(Date からテストを足していない旨の行まで)と本文を、そのとき書いたまま残す。見出しは 1 段下げて「(cycle 1)」を付け、判定の見出しは `## Verdict` から「判定(cycle 1)」に変えた。shell の PASS の行の 3,972 は、上の Test execution の注のとおり 3,962 が正しい(数え方の違いで、テストの数は変わっていない)。Test gaps の末尾の「追記」は、cycle 1 のあと 4aaef4ed でテストを足したときに書いたもの。

- Date: 2026-10-11(JST。実行は 18:46〜19:30)
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み、digest d0a6f2735552)
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 1(`cycle-count.json` は 1、上限 2)
- Scope: HEAD dd63a414(base 577ebeaf)。behavioral test だけを実行した(静的解析は /verify の担当)。依頼された 4 つを行った: `run-test.sh` と `go test`(`-race` を含む)、AC とテストの対応、`go test -overlay` の mutation、一時の台帳での実機確認。あわせて、計画の Edge cases のうちテストのないものを probe のテストで確かめ、Linux の setsid を Docker のコンテナで確かめた
- Evidence: `docs/evidence/test-2026-10-11-org-cross-watch.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore -v` で `.gitignore:58` に当たることを確かめた)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-11-094617.log`(時刻は UTC)。実機確認は `docs/evidence/org-cross-watch-live-2026-10-11.md`(コミットする)
- テストは足していない。mutant の写しと probe のテストは scratchpad に置き、`go test -overlay` で使った。worktree の追跡ファイルは書き換えていない

### Test execution(cycle 1)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(HEAD、変更言語の scope) | shell 40 本(PASS の行 3,972)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 7 分 1 秒、rc 0 |
| `go test ./... -count=1 -v` | top-level 1,368(org 551、cli 541)、subtest 1,447 | すべて | 0 | 3(下の注) | 92 s(cli 91.4 s、org 27.7 s)、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -count=1` | 2 パッケージ | 2 | 0 | 0 | 97 s(org 31.0 s、cli 92.5 s)、rc 0 |
| `go test -race -run 'Watch\|Probe\|Alive\|Deadline\|Status\|Inbox\|Escalate' -count=3`(org と cli) | top-level 627(org 124 本 × 3、cli 85 本 × 3) | すべて | 0 | 3(1 本 × 3) | 59 s、rc 0。`DATA RACE` は 0 件 |
| `go test -coverprofile`(org、cli、config) | 3 パッケージ | 3 | 0 | 0 | 92 s、rc 0 |
| Linux の `ubuntu:24.04`(aarch64)で、クロスビルドした test binary | cli の watch と status 22 本、org の watch・期限・受信箱 126 本 | 147 | 1(環境) | 0 | cli 2 s、org 4 s |
| probe のテスト(`-overlay` で足す 9 本、`-race -count=3`) | 27 | 27 | 0 | 0 | 7 s |
| mutation(`go test -overlay`、HEAD のテスト) | 82 件 | red 73 件 | 生き残り 9 件(下の節) | - | 1 件 30〜140 s |
| 実機確認(macOS 2 本、Linux 1 本) | 下の「実機確認」 | すべて期待どおり | 0 | - | 合わせて 1 分ほど |

- shell の件数は suite ごとに PASS・FAIL・SKIP の行を数えた(`PASS: 29` のような集計行は除いた)。40 本すべてが `OK` で終わり、FAIL の行は 0。Skipped の 1 件は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)
- `run-test.sh` の言語の scope は、`scripts/ralph-config.sh` が言語に分類されないので full に落ちた(`Language scope: full fallback`)。中の `go test ./...` は `internal/org` だけを実行し、ほかはキャッシュの結果だった。そのため 2 行目で全パッケージを `-count=1` で流し直した。go は 1.26.0(darwin/arm64)
- Go の Skipped の 3 件は、`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin では skip する。Linux のコンテナでは走って通った)と、`internal/scaffold` の前からある 2 件(`TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`)
- この枝で足した top-level のテストは 62 本(`internal/cli/org_watch_test.go` 15、`internal/cli/status_test.go` 6、`internal/config/config_test.go` 4、`internal/org/deadline_test.go` 11、`internal/org/watch_all_test.go` 15、`internal/org/watch_test.go` 4、`internal/org/watchalive_test.go` 7)。62 本とも通った。消えた 4 本は `TestOrgWatch_RequiresOrgID`(拒否のテストに移った)と、`RunWatch` の 3 本(`RunWatchAll` の 3 本に置き換わった)
- Linux の 1 件の失敗は `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` で、コンテナに git がない(`exec: "git": executable file not found in $PATH`)ため。macOS では通っている。コードの失敗ではない
- 実行中、ほかのセッションの負荷で load average は 4〜93(12 コア)だった。既知の flaky なテスト(tester の MEMORY.md の 3 本)はどれも落ちなかった

### Coverage(cycle 1)

- Statement: `internal/org` 94.0%、`internal/cli` 86.3%、`internal/config` 93.2%(前の計画の時点は 93.8%、86.0%、92.8%)。ほかの 5 パッケージは測り直していない
- Function(この枝が足した関数のうち 100% でないもの): `launchOrgWatch` 0.0%(本物の ralph を exec する本番の起動。TestMain が panic する偽物に差し替えるので、どのテストも通らない。これで AC12 の「テストは test binary を exec しない」を確かめた。本物の起動は実機確認で通した)、`startDetachedProcess` 85.7%(ログやディレクトリを開けない枝)、`runOrgWatchStop` 85.7%(頼みを書けない枝)、`runOrgWatchDetach` 95.7%、`acquireHeldFileLock` 81.2%・`fileLockHeld` 85.7%・`AcquireWatchLock` 83.3%(flock や open の失敗、印の読めない拒否)、`newWatchAllRun` 71.4%(引数が空のときの既定値)、`cycle` 85.7%、`pulse` 90.9%、`writeMark` 66.7%・`finish` 66.7%(印を書けない枝)、`dropOldStopRequest` 80.0%、`ReadWatchStopRequest` 87.5%、`RemoveWatchStopRequest` 75.0%、`writeJSONFileAtomic` 85.7%
- 100% の関数: `deadline.go` の 7 関数すべて(`InboxDeadlineDue`、`SweepInboxDeadlines` ほか)、`ProbeWatch`、`watchAliveWindow`、`AppliesTo`、`RunWatchAll`、`pulseOrg`、`waitForNextCycle`、`callWithTimeout`、`realGitStatus`、`checkOrgWatchFlags`、`runOrgWatchAll`、`orgWatchChildArgs`、`statusDeadmanMinutes`、`orgWatchLine`、`buildStatusOrgWatchJSON`
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 行の coverage は「条件を外しても気づくか」を示さない。そのため下の mutation で確かめた。`ProbeWatch` は 100% だが、確かめられないロックの扱い(P2)は mutation で生き残った

### AC とテストの対応(cycle 1)

テストはすべて HEAD で通った。「実機」は `docs/evidence/org-cross-watch-live-2026-10-11.md` の確認、「probe」は下の「Probe のテスト」(コミットしていない)の番号。

| AC | 固定するテスト | 補い |
| --- | --- | --- |
| AC1 設定の既定、検証、3 面 | `TestDefault_OrgDirector`、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`、`TestLoad_OrgDirectorRoundTrip`(resolve_minutes が ack_minutes より短い設定を含む)、`TestLoad_OrgDirectorRejects`(2 キー × 0 と -1)、`TestDefaultsLockStep` の 2 行 | なし |
| AC2 期限の判定表 | `TestInboxDeadlineDue`(22 行と、時刻の読めない件 8 通り)。表の 7 項目、期限ちょうど、時計の逆行を含む | probe 1(resolve_minutes 5 < ack_minutes 10 の判定と掃き出し) |
| AC3 人への経路と状態の不変 | `TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman`、`_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral`、`_SkipsResolvedItemsAndKeepsIDOrder` | 本物の osascript は確かめていない(Test gaps) |
| AC4 書けない台帳での送り直しの間隔 | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes`、`_NotifiedUnwritableIsHeldBack`、`_HoldBackIsPerReason` | なし |
| AC5 起動し直したときの計算し直し | `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`、`TestRunWatchAll_SendsAnInboxItemPastItsDeadlineInTheFirstCycleOnce`、`TestOrgWatch_Once_SendsAnInboxItemPastItsDeadline` | probe 2(`inbox.jsonl` のない台帳) |
| AC6 escalate はすぐ送り、二度目は resolve の期限だけ | `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline`。escalate の既存のテスト(名前に Escalate を含む 25 本)も通る | なし |
| AC7 全 org を 1 プロセスで、disband した org は見ない | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`、`TestRunWatchAll_MultipleOrgs_SeparateStatusFiles`、1 org のサイクルの既存のテスト(`TestWatch_*` 28 本) | probe 5(30 席・5 org の 1 サイクルの時間) |
| AC7b 返らない呼び出し、書けない状態ファイル | `TestRunWatchAll_AHungHerdrCallOfTheFirstOrgIsCutOffAndTheCycleGoesOn`、`_EveryHungDriverCallOfTheFirstOrgIsCutOff`、`_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg`、`_AManifestThatCannotBeReadSkipsOnlyThePulse`、`_AnInboxThatCannotBeSweptDoesNotStopThePulse`、`TestRealGitStatus_IsCutOffAtTheDriverCallTimeout` | probe 8(受信箱の期限をパルスより先に見る順)。本物の herdr が返らない場合は確かめていない(Test gaps) |
| AC8 `--org-id` と `--all` なしの拒否 | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing`(9 通り、stdout が空で state dir を作らない) | なし |
| AC9 2 つ目の見張りの拒否、終わればロックが放される | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`、`TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds`、`TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock` | 実機 1(別プロセスの `--once` が pid を名指しして rc 1。`kill -9` のあとロックが放され、次の `--detach` が通る。verify が挙げた kill の穴)、実機 2(8 つ同時の `--detach` で、動く見張りは 1 つ) |
| AC10 印の書き込み、SIGINT と SIGTERM | `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd`、`TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark`、`TestWriteWatchAlive_ReplacesThroughARenameAndLeavesNoTemporaryFile` | probe 3(サイクルの途中で ctx が取り消される)、probe 4(stopped の印を書いてからロックを放す順)、実機 1(間隔 2 秒で `ts` が進む) |
| AC11 判定関数の 3 状態 | `TestProbeWatch_TellsRunningStaleAndNotRunningApart`(17 行、窓の境界)、`TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`、`TestReadWatchAlive_MissingAndCorrupt`、`TestAcquireWatchLock_WaitsOutAShortSharedLock` | probe 6(確かめられないロックは持たれているとみなす) |
| AC12 `--detach` | `TestOrgWatchDetach_AlreadyRunning_StartsNothing`、`_Stale_StartsNothingAndSaysHowToStopIt`、`_StartsTheChildAndWaitsForItsRunningMark`、`_ChildExitsFirst_ExitsOneNamingTheLog`、`_DeadmanMinutesZero_StartsTheChild`、`_StartDetachedProcess_NewSessionDevNullStdinAppendedLog`。test binary を exec しないことは `launchOrgWatch` の coverage 0.0% で確かめた | 実機 1・3(本物の子、macOS と Linux の setsid)、probe 7(ほかの見張りの印を自分の子と取り違えない) |
| AC12b 待ちきれなかった子の回収 | `TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock`(ロックが放される場合と残る場合) | なし |
| AC13 `--stop` | `TestOrgWatchStop_NotRunning_SaysSoAndWritesNoRequest`、`_StopsARunningWatch`、`_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal`、`TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt` | 実機 1・3(本物のプロセスが 0.43 s で止まる) |
| AC13b 古い頼みに従わない、シグナルを送らない | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`(前・同時刻・1 ns 後)、`TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt`、`TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal`(印が生きている別プロセスを指す) | probe 9(ロックを待つ間に書かれた頼みに従う) |
| AC14 `ralph status` の行と `org_watch` | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState`、`_OrgWatchLineFollowsTheStateDirLine`、`_EmptyRosterStillShowsTheOrgWatch`、`_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml`、`_OrgWatchWithDeadmanMinutesZeroJudgesByTheDefault`、`_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad` | 実機 1(running、kill のあとの not running と残った印、stopped) |
| AC15 spawn・send・start・stop・disband・escalate は変わらない | それらのテストファイルは差分に入っていない(`git diff --stat 577ebeaf...HEAD -- '*_test.go'` の 12 本に含まれない)。名前に Spawn・Send・Start・Stop・Disband を含む top-level のテスト 313 本と、Escalate を含む 25 本が通った | なし |
| AC16 tech-debt (c)〜(h) は動作を変えない | `internal/cli/org_inbox_test.go` と `internal/org/inbox_test.go` は無変更で通る。(h) を拾う `TestOrgEscalate_BrokenConfig_AlertsTheHumanAndExitsOne` と `TestInboxCommand` も通る。`escalate_test.go` の 9 か所の呼び出しの形の変更は、verify が assertion は変わっていないと読んだ | なし |
| AC17 文書 | behavioral test はない。`check-sync.sh` と `check-skill-sync.sh` は verify の静的な検査が持つ | 対象外(/verify と /sync-docs) |

behavioral test のない AC は AC17 だけで、文書の AC なので /test の対象外とした。

### Mutation(cycle 1)

`scratchpad/ocw-test/mut.py` は、`internal/org/watchalive.go` などの写しを書き換えて `go test -overlay` で走らせる。書き換える文字列がファイルに 1 回だけあることを、当てる前に確かめた。org 層の mutant は `internal/org` の全体を走らせ、生き残ったら `internal/cli` の全体も走らせた。CLI の mutant はまず `-run 'OrgWatch|StatusCmd'` で走らせ、生き残ったものを `internal/cli` の全体で流し直した。

依頼された 9 点と、その mutant:

| 依頼 | mutant | 結果 |
| --- | --- | --- |
| ロック(`AcquireWatchLock` が持ち続ける) | L1、L2、L3、L4、L5b | 5 件とも red |
| probe のロックの確認 | P1、P2、P3、P4、P5 | P1・P3・P4 は red。**P2 が生き残り**(確かめられないロックの扱い)。P5 は等価 |
| 窓の規則(deadman の下限と間隔の 2 倍) | W1(下限)、W2(2 倍の項)、W3、W4、W5、W6、W7 | 7 件とも red |
| `InboxDeadlineDue` の ack と resolve の順 | D1 | red |
| ack の期限の open の条件 | D2 | red |
| `RunWatchAll` の org ごとの分離 | A1、A2、A3、A5、A7 | 5 件とも red。分離とは別の A4 は等価、A6(順)は生き残り |
| 止める頼みに従うこと | S1、S2、S3b、S4、S5b、S6、S7、S8 | S6 以外は red。**S6 が生き残り**(開始の時刻を読む位置) |
| `--detach` の期限切れの子の回収 | X1、X2 | どちらも red |
| `--org-id` の拒否 | F1 | red |

全体は 82 件で、73 件が red、9 件が生き残った。生き残りの内訳は、本物の穴が 5 つ(P2、X3、L6b と L6c、A6、S6。L6b と L6c は同じ順の入れ替えの 2 通り)、等価なものが 3 つ(P5、A4、X12)。下の表の「落としたテスト」は、CLI の mutant については `-run 'OrgWatch|StatusCmd'` の 1 回目のもの。

| ID | 書き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| L1 | `acquireHeldFileLock` が flock を取った直後に外す(ロックを持たない) | red | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`, `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestProbeWatch_TellsRunningStaleAndNotRunningApart`, `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| L2 | `RunWatchAll` が最初のサイクルの前にロックを放す | red | `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds`, `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| L3 | `AcquireWatchLock` が待たない(1 回だけ試す) | red | `TestAcquireWatchLock_WaitsOutAShortSharedLock` |
| L4 | 拒否のエラーが `ErrWatchRunning` を包まない | red | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`, `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds` |
| L5b | 拒否のエラーが印の pid を名指ししない | red | `TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark`, `TestRunWatchAll_ASecondWatchIsRefusedUntilTheFirstEnds` |
| L6b | `finish` がロックを先に放し、50 ms おいて stopped の印を書く | **生き残り** | (probe 4 が落とす) |
| L6c | `finish` がロックを先に放し、すぐ stopped の印を書く | **生き残り** | (probe 4 が落とす) |
| P1 | `ProbeWatch` がロックを見ない(running の印だけで running) | red | `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestProbeWatch_TellsRunningStaleAndNotRunningApart`, `TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| P2 | `ProbeWatch` が、確かめられないロックを空いているとみなす | **生き残り** | (probe 6 が落とす) |
| P3 | `fileLockHeld` が、ロックのファイルがないときに持たれているとみなす | red | `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding`, `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| P4 | `fileLockHeld` が O_CREATE で開く(watch.lock を作る) | red | `TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding` |
| P5 | `fileLockHeld` が共有ロックを LOCK_UN で外さない | **生き残り**(等価) | |
| W1 | 窓: `deadman_minutes` が 1 未満のときに既定の 10 にしない | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W2 | 窓: 間隔の 2 倍の項を外す | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W3 | 窓: 間隔の 2 倍を 1 倍にする | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W4 | 窓: ちょうど窓の年齢を stale にする(`>` を `>=`) | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| W5 | 窓: `deadman_minutes` を見ず、いつも既定の 10 | red(org では生き残り、cli で red) | `TestStatusCmd_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml` |
| W6 | status: `./ralph.toml` ではなく既定の deadman で判定する | red | `TestStatusCmd_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml` |
| W7 | `ProbeWatch`: ロックが持たれていて印が stopped なら running | red | `TestProbeWatch_TellsRunningStaleAndNotRunningApart` |
| D1 | 期限: ack を resolve より先に見る(順を入れ替える) | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`, `TestSweepInboxDeadlines_HoldBackIsPerReason` |
| D2 | ack の期限: open の条件を外す(acked にも効く) | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_SkipsResolvedItemsAndKeepsIDOrder` |
| D3 | ack の期限: 「notified が 1 つもない」を外す | red | `TestInboxDeadlineDue` ほか 5 本 |
| D4 | ack の期限: ack の期限の notified だけが止める | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`, `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline` |
| D5 | resolve の期限: notified を見ず毎回送る | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice`, `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline` |
| D6 | resolve の期限: どの notified でも止まる(escalate した件に届かない) | red | `TestInboxDeadlineDue` ほか 3 本 |
| D7 | resolve の期限: ちょうど resolve_minutes で送る(`>` を `>=`) | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline` |
| D8 | ack の期限: ちょうど ack_minutes で送る(`>` を `>=`) | red | `TestInboxDeadlineDue` |
| D9 | resolved の件にも resolve の期限が効く | red | `TestInboxDeadlineDue`, `TestSweepInboxDeadlines_SkipsResolvedItemsAndKeepsIDOrder` |
| D10 | 送り直しの間隔を resolve_minutes にする | red | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes` |
| D11 | 送り直しの間隔のちょうどでも控える(`<` を `<=`) | red | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes` |
| D12 | 送れなかった件を覚えない(毎サイクル送り直す) | red | `TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes`, `TestSweepInboxDeadlines_HoldBackIsPerReason`, `TestSweepInboxDeadlines_NotifiedUnwritableIsHeldBack` |
| D13 | 控える鍵に reason を入れない | red | `TestSweepInboxDeadlines_HoldBackIsPerReason` |
| D14 | 時刻の読めない件の注意を毎回出す | red | `TestSweepInboxDeadlines_WarnsOnceAboutAnEscalatedAtThatDoesNotParse` |
| D15 | 1 分未満の期限を断らない | red | `TestSweepInboxDeadlines_ReturnsAnErrorWhenItCannotStart` |
| D16 | 期限の送信の reason を `inbox_no_director` にする | red | `TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman` ほか 10 本 |
| D17 | ack の期限でも resolve の期限の文面にする | red | `TestSweepInboxDeadlines_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral`, `TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman` |
| A1 | 1 つ目の org の失敗でパルスを終える(ほかの org を回さない) | red | `TestRunWatchAll_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg` |
| A2 | すべての org が 1 つの状態を共有する | red | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`, `TestRunWatchAll_MultipleOrgs_SeparateStatusFiles` ほか 4 本 |
| A3 | すべての org が 1 つの状態ファイル名を使う | red | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated`, `TestRunWatchAll_MultipleOrgs_SeparateStatusFiles` ほか 7 本 |
| A4 | 走らなくなった org の状態をメモリから消さない | **生き残り**(等価) | |
| A5 | disband した org も含め manifest のすべての org を回す | red | `TestRunWatchAll_EachRunningOrgGetsTheSingleOrgCycle_ADisbandedOrgIsNotEvaluated` |
| A6 | パルスを受信箱の期限より先に回す | **生き残り** | (probe 8 が落とす) |
| A7 | 受信箱を掃き出せないとき、パルスと印を飛ばす | red | `TestRunWatchAll_AnInboxThatCannotBeSweptDoesNotStopThePulse` |
| A8 | 最初のサイクルの前に印を書かない | red | `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| A9 | サイクルのあとに印を書き直さない | red | `TestRunWatchAll_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg`, `TestRunWatchAll_WritesTheMarkAtStartAndEveryCycleAndStoppedAtTheEnd` |
| A10 | `callWithTimeout` に期限がない | red | panic: `TestRunWatchAll_AHungHerdrCallOfTheFirstOrgIsCutOffAndTheCycleGoesOn` の 10 s の guard(下の注) |
| A11 | `realGitStatus` が git を context つきで起動しない | red | `TestRealGitStatus_IsCutOffAtTheDriverCallTimeout` |
| A12 | 状態ファイル名にする前に org_id を確かめない | red | `TestRunWatchAll_AnOrgIDThatIsNotAnIdentifierIsSkipped` |
| A13 | 読めない状態ファイルを毎サイクル言う | red | `TestRunWatchAll_AStatusFileThatCannotBeReadIsSaidOncePerOrg` |
| A14 | 読めた状態をメモリに残さない(毎サイクル読み直す) | red | `TestRunWatchAll_AStatusFileOfTheFirstOrgThatCannotBeWrittenSkipsOnlyThatOrg` |
| S1 | サイクルの間の待ちで止める頼みを読まない | red | `TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt`, `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S2 | 開始と同時刻の頼みにも従う | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`, `TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt` |
| S3b | どの頼みにも従う(古いものも) | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`, `TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt` |
| S4 | 起動のときに古い頼みを消さない | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S5b | 起動のときに、自分宛ての頼みも消す | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S6 | 開始の時刻をロックを取ったあとに読む | **生き残り** | (probe 9 が落とす) |
| S7 | 従った頼みを `finish` が消さない | red | `TestRunWatchAll_AStopRequestWrittenWhileItWaitsEndsIt`, `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart` |
| S8 | 頼みの時刻を秒で比べる(ナノ秒を落とす) | red | `TestRunWatchAll_FollowsOnlyAStopRequestLaterThanItsStart`, `TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt` |
| S9 | `--stop` が印の pid に SIGTERM も送る(安全のため、このプロセスの子だけに) | red | `TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal` |
| X1 | `--detach` の期限切れで子を止めない | red | `TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock` |
| X2 | `--detach` の期限切れでロックが放されるのを待たない | red | `TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock` |
| X3 | `--detach` が、印の pid が自分の子かを見ずに成功とする | **生き残り**(cli 全体でも) | (probe 7 が落とす) |
| X4 | `--detach`: stale でも起動する | red | `TestOrgWatchDetach_Stale_StartsNothingAndSaysHowToStopIt` |
| X5 | `--detach`: running でも起動する | red | `TestOrgWatchDetach_AlreadyRunning_StartsNothing` |
| X6 | 子の引数に `--config` を渡さない | red | `TestOrgWatchDetach_StartsTheChildAndWaitsForItsRunningMark` |
| X7 | 子の引数にいつも `--interval-seconds` を渡す | red | `TestOrgWatchDetach_StartsTheChildAndWaitsForItsRunningMark` |
| X8 | 子を setsid で起動しない | red | `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` |
| X9 | `watch.log` を追記でなく切り詰めて開く | red | `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` |
| X10 | 子が先に終わっても期限まで待つ | red | `TestOrgWatchDetach_ChildExitsFirst_ExitsOneNamingTheLog` |
| X11 | 子を止めるとき SIGTERM を待たずに SIGKILL | red | `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` |
| X12 | 子の stdin を /dev/null にしない(親から継ぐ) | **生き残り**(等価、cli 全体でも) | |
| F1 | `--org-id` を断らない | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| F2 | `--all` を求めない | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` のサブテスト `no --all`(`org_watch_test.go:174`)(下の注) |
| F3 | `--detach` と `--once` を受ける | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| F4 | `--stop` と `--detach` を受ける | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| F5 | フラグの検査を runtime を作ったあとにする(断る前に state dir を作る) | red | `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` |
| G1 | SIGTERM を扱わない(SIGINT だけ) | red | `TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark` の `terminated`(テストのプロセスが SIGTERM で終わった) |
| G2 | 長く動く見張りが止まったとき watcher の呼び出しを取り消さない | red | `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` |
| T1 | 空の roster の `--json` に `org_watch` を入れない | red | `TestStatusCmd_EmptyRosterStillShowsTheOrgWatch`, `TestStatusCmd_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad` |
| T2 | 表の側で `org watch:` の行を出さない | red | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState`, `TestStatusCmd_OrgWatchLineFollowsTheStateDirLine` |
| T3 | 残った running の印の行から pid を落とす | red | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` |
| T4 | `--json` に `mark_state` を写さない | red | `TestStatusCmd_OrgWatchLineAndJSONPerWatchState` |

#### 生き残った mutant(cycle 1)

本物の穴で、コミットされたテストが固定しないもの(probe で閉じることを確かめた):

- **P2**: `ProbeWatch` の doc は「確かめられないロックは持たれているとみなす」と書くが、それを固定するテストがない。`fileLockHeld` がエラーを返す場合(`watch.lock` が開けない)を、どのテストも作らない。probe 6(mode 000 の `watch.lock`)は HEAD で通り、P2 で落ちる(`not_running` を返す)
- **X3**: `--detach` が「印の pid が自分の子のもの」を見ずに成功とする。すべての `--detach` のテストで、running の印は子(か、子より前からの見張り)のものだけなので、ほかの見張りが先にロックを取った形がない。実機 2 の同時の `--detach` がまさにその形で、HEAD は正しく rc 1 を返した。probe 7 は HEAD で通り、X3 で落ちる(子が終わったのに `started ralph org watch --all (pid 4242)` を出す)
- **L6b・L6c**: `finish` の「stopped の印を書いてからロックを放す」順(doc は「`--stop` がロックの空きを見たら stopped の印も見える」と書く)。順を入れ替えると、ロックを取った新しい見張りの running の印を、古い見張りが stopped で上書きしうる。`TestOrgWatchStop_StopsARunningWatch` は見張りが返ったあとで印を読むので、順を見分けない。probe 4 は HEAD で 10 回 × 20 周すべて通り、L6c(sleep なし)は 10 回とも 1 周目で落ちた
- **A6**: 受信箱の期限をパルスより先に見る順(Codex plan advisory の指摘 1)。AC7b のテストは「同じサイクルで届く」ことを見るので、パルスのあとに送っても通る。順を入れ替えると、herdr が返らない org があるときに、期限の通知が打ち切りの合計(席の数 × 10 秒まで)だけ遅れる。probe 8(最初の herdr の呼び出しの時点で `escalations.jsonl` に行がある)は HEAD で通り、A6 で落ちる
- **S6**: 開始の時刻をロックを待つ前に読むこと(計画の S3a のずれ)。テストの時計は止まっているので、ロックの前に読んでもあとに読んでも同じ値になり、見分けられない。probe 9 は、ロックを持つ別の holder を置き、待っている間に時計を進めて頼みを書く。HEAD は 1 サイクルで止まり、S6 は頼みを古いものとして消して 3 サイクル回る。ただし probe 9 は「`RunWatchAll` が開始を読んだあと」を 100 ms の sleep で作っているので、コミットするなら時計の差し替えで同期させる

等価(穴ではない):

- **P5**: `fileLockHeld` が共有ロックを `LOCK_UN` で外さなくても、続く `Close` がロックを放す
- **A4**: 走らなくなった org の状態をメモリに残しても、その org がまた走るときは、HEAD でも毎サイクル保存した同じ内容をファイルから読み直す。違いはメモリの量だけ
- **X12**: `go test` は test binary の stdin をいつも `/dev/null` にする(stdin にパイプを渡しても変わらないことを確かめた)ので、子が親の stdin を継いでも `stdin: eof` になる。見分けるには、テストの中で `os.Stdin` をパイプに差し替える必要がある。本番の見張りは stdin を読まない

依頼どおり、テストは足していない。足すなら probe 4・6・7・8・9 の 5 本で、それぞれ 1 本ずつで生き残りを閉じる(scratchpad の `zz_probe_test.go` と `zz_probe_cli_test.go` にある)。足すかは orchestrator が決める。

mutant の当て方の注:

- L5 と S5 は書き換えで変数が使われなくなり、コンパイルに失敗した。`_ = mark`、`_ = req` を足した L5b と S5b で走らせ直した
- L6 は 2 回目の `Release` の stderr の 1 行(`file already closed`)だけで落ちた。順の違いで落ちたのではないので、`Release` を 1 回にした L6b(50 ms おく)と L6c(おかない)で走らせ直し、どちらも生き残った
- F2 と A10 は、終わらないテストが残ったので、そのテストのプロセスを SIGQUIT で止めた。F2 はその前にサブテストが落ちている。A10 は AC7b のテストだけで流し直し、10 s の guard の panic で落ちることを確かめた
- 実行中は load average が 6〜93 だった。落ちたテストはどれも、書き換えた動きを名指しする assertion で落ちていることをログで見た(時間切れだけで落ちたものはない)

### Probe のテスト(cycle 1)

計画の Edge cases と mutation の生き残りのうち、コミットされたテストが固定しないものを、scratchpad の `zz_probe_test.go`・`zz_probe_timing_test.go`(`internal/org`)と `zz_probe_cli_test.go`(`internal/cli`)に書き、`go test -overlay` で足して走らせた。コミットしていない。HEAD では 9 本とも、`-race -count=3` で通った。

| # | probe | HEAD | 確かめたこと |
| --- | --- | --- | --- |
| 1 | `TestZZProbe_ResolveMinutesShorterThanAckMinutes` | 通る | ack 10・resolve 5 では、5 分ちょうどまでは何も送らず、5 分 1 秒で resolve の期限を 1 回だけ送り、その `notified` で ack の期限も止まる(11 分、3 時間でも鳴らない)。計画の Edge cases「resolve_minutes が ack_minutes より短い設定」は、設定のテストだけで判定のテストがなかった |
| 2 | `TestZZProbe_SweepWithNoInboxFile` | 通る | `inbox.jsonl` のない台帳の掃き出しはエラーなしで何も送らず、`escalations.jsonl` も作らない |
| 3 | `TestZZProbe_ContextCancelledMidCycleFinishesTheCycle` | 通る | 1 つ目の org の git status の最中に ctx を取り消しても、そのサイクルは最後まで回り(2 つ目の org も 1 サイクル、生存の偽の ALERT なし)、そのサイクルの印を書いてから stopped で終わる。計画の Edge cases「SIGTERM がサイクルの途中で届く」の org 層 |
| 4 | `TestZZProbe_TheLockIsFreeOnlyAfterTheStoppedMark` | 通る(10 回 × 20 周) | ロックが空いたのを見た時点で、印はもう stopped。L6b と、sleep を抜いた同じ並べ替え(L6c)は 10 回とも 1 周目で落ちた |
| 5 | `TestZZProbe_CycleTimeWith30Seats` | 通る | 30 席・5 org、全席が毎サイクル担当範囲を変える(毎サイクル ALERT)。herdr と agmsg は偽物ですぐ答える。1 サイクルは 13〜42 ms(`-race` では 61〜222 ms)。本物の herdr と agmsg のプロセスの時間は含まない |
| 6 | `TestZZProbe_ALockThatCannotBeCheckedCountsAsHeld` | 通る | mode 000 で開けない `watch.lock` は持たれているとみなす(印がなければ stale、新しい running の印なら running)。P2 で落ちる |
| 7 | `TestZZProbe_DetachDoesNotTakeAnotherWatchsMarkForItsChild` | 通る | `--detach` の子がロックを取れずに終わる間に、ほかの見張りが新しい running の印を書いていても、`--detach` は自分の子を「started」とは言わず rc 1。X3 で落ちる |
| 8 | `TestZZProbe_InboxDeadlinesComeBeforeThePulse` | 通る | サイクルの最初の herdr の呼び出しの時点で、期限を過ぎた件の `escalations.jsonl` の行がもうある(受信箱の期限をパルスより先に見る)。A6 で落ちる |
| 9 | `TestZZProbe_AStopWrittenDuringTheLockWaitIsFollowed` | 通る | ほかの holder がロックを持つ間に `RunWatchAll` を起こし、待っている間に時計を進めて頼みを書く。ロックが空くと 1 サイクルで止まる。S6 で落ちる(3 サイクル回る)。「開始を読んだあと」は 100 ms の sleep で作っている |

### 実機確認(cycle 1)

`docs/evidence/org-cross-watch-live-2026-10-11.md` にコマンドと出力を残した。依頼の規則どおり、台帳は scratchpad の下だけ、manifest も受信箱も置かず(herdr・agmsg・osascript は呼ばれない)、ralph は `env -i` で空の PATH と scratchpad の HOME で起こした。利用者の herdr サーバー、`~/.config/herdr`、本物の台帳には触れていない。最後の `pgrep -fl 'org watch --all'` は何も返さなかった。

1. macOS(依頼の流れ): `--all --detach` は rc 0 で pid 9450 を出し、0.13 s で返った。子は新しいセッション(`PGID == PID`、`TTY ??`、`STAT Ss`、`PPID 1`)。`ralph status` と `--json` は running。`kill -9 9450` のあと、`ralph status` は `org watch: not running (a running mark is left from pid 9450, last cycle …)`、`--json` は `state: not_running`、`mark_state: running`。もう一度の `--detach --interval-seconds 2` は rc 0 で新しい pid 9606(印の `interval_seconds` は 2、4.5 s で `ts` が進んだ)。そのあいだの `--all --once` は rc 1 で `(pid 9606, from …/watch-alive.json)` を名指しした。`--stop` は rc 0、0.43 s で返り、status は `not running (stopped at …)`、`watch-stop.json` は残らなかった
2. macOS(足した確認): 新しい台帳に `--detach` を 8 つ同時に、4 回打った。どの回も動いている見張りは 1 つだけで、`started` 1 本、`already running` 5〜6 本、exit 1 が 1〜2 本(`the watch (pid N) exited before it was running`)。exit 1 の子はロックを 2 秒待ってから `another ralph org watch --all is running …` を `watch.log` に書いて終わっていた
3. Linux(足した確認、`ubuntu:24.04` のコンテナ、一般ユーザー): 同じ流れが通った。`--detach` の子は `SID 20`(自分の pid)で、起こしたシェルの SID 11 と違う。setsid は Linux でも効く

### Failure analysis(cycle 1)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし(HEAD の macOS の実行) | - | - | - |
| `TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall`(Linux のコンテナだけ) | `exec: "git": executable file not found in $PATH` | コンテナの `ubuntu:24.04` に git がない。テストの前提の環境の違いで、コードの失敗ではない | なし。CI(git のある Linux)では走る |

### Regression checks(cycle 1)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 1 org のパルスのサイクル(stall・生存・担当範囲・デッドマン) | 通る | `TestWatch_*` 28 本(assertion は変わっていないと verify が読んだ)。AC7 のバイト一致のテストが `RunWatchAll` と 1 org のサイクルを比べる |
| escalate・inbox・`wait --inbox` | 通る | 名前に Escalate を含む 25 本、Inbox を含む 60 本、Wait を含む 17 本 |
| spawn・send・start・stop・disband | 通る | 名前にそれらを含む top-level 313 本。テストファイルは無変更 |
| 旧い台帳の検査(`org_legacy_ledger_test.go` の watch の 1 行の書き換え) | 通る | `TestOrgLegacyLedger_*` 5 本 |
| `ralph status` の既存の表示と `--json` の形 | 通る | `TestStatusCmd_*` 25 本(うち新しい 6 本) |
| 設定の lock-step | 通る | `TestDefaultsLockStep` |

### Test gaps(cycle 1)

- 本物の osascript の通知を、切り離した子から出すこと: 依頼どおり確かめていない(実機の台帳に期限を過ぎた件を置いていない)。期限の通知の AppleScript 3 通り(ack の期限、resolve の期限、`<invalid>` に置き換えた件)は、`osacompile` でコンパイルだけを通した(実行していない)。子は `TTY ??` の新しいセッションで動くので、ログインセッションの中で通知が出るかは、計画の Open questions のまま残る
- Linux の setsid: Docker の `ubuntu:24.04`(aarch64)で `SID == PID` を確かめた。CI の Linux では、`TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` が process group を見るだけで、SID は見ない
- 本物の herdr が返らない場合: テストは偽物の herdr と agmsg で打ち切りを確かめる。本物の driver が `exec.CommandContext` と 2 秒の `WaitDelay` で切れることは、verify がコードを読んだだけで、実機では確かめていない。30 席の 1 サイクルは ralph の側だけで 13〜42 ms(probe 5)で、本物の herdr の `agent get` と agmsg のプロセスの時間は入っていない
- プロセスをまたぐロックの取り合い: 実機で、別プロセスの 2 つ目の見張り、`kill -9` のあとの解放、8 つ同時の `--detach`(4 回)を確かめた。コミットされたテストは、同じプロセスの中の 2 つの open file で排他を見る(flock は open file ごとに効くので、別プロセスと同じ意味を持つ)。`ralph status` の共有ロックと見張りの起動の取り合いを、プロセスをまたいで何度も試したことはない
- mutation の生き残りのうち本物の穴 5 件(P2、X3、L6b と L6c、A6、S6)は、テストで固定されていない。probe 4・6・7・8・9 がそれぞれを閉じることを確かめた。足すかは orchestrator に任せる
- 同時の `--detach` の 1〜2 本が、見張りが動いているのに rc 1 になる(下の「参考」)。テストはこの形を作らない(X3 の probe 7 が作る)
- `--config` で別の `deadman_minutes` を渡した見張りと、`./ralph.toml` で判定する `ralph status` のずれは、tech-debt の行のとおりでテストはない。self-review の N2(読めない状態ファイルの org を黙って飛ばし続ける)と N3(`--stop` で取り消した watcher の呼び出しを失敗として記録する)も tech-debt にあり、テストはその今の動きを固定している
- Linux のコンテナには git がないので、`TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall` だけはコンテナで走らせられなかった(macOS では通る)
- 追記(このレポートのあと、orchestrator の依頼でテストを足した): 上の本物の穴 5 件は、コミット `test: pin the cross-org watch behaviors that surviving mutants showed unpinned` のテストで閉じた。本番のコードは変えていない。5 本とも HEAD で通り(`-race -count=10` で 10 回とも)、上の probe と同じ overlay の JSON で mutant を当てると落ちる
  - P2: `TestProbeWatch_ALockFileThatCannotBeOpenedCountsAsHeld`(`internal/org/watchalive_test.go`)。mode 000 の `watch.lock` で、印がないときと stopped の印は stale、新しい running の印は running。P2 では 3 つとも `not_running` になって落ちる
  - X3: `TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild`(`internal/cli/org_watch_test.go`)。probe 7 の 300 ms の sleep をやめ、子の `exited` を初めから閉じておく。HEAD は pid 4343 の印を自分の子のものとせず、子が終わったことを返す。X3 は最初の確認で `started ralph org watch --all (pid 4242)` を出して落ちる。上の「同時の `--detach`」の形も、このテストが作る
  - L6b・L6c: `TestRunWatchAll_WritesTheStoppedMarkBeforeReleasingTheLock`(`internal/org/watch_all_test.go`)。probe 4 と同じ 20 周。L6b と L6c は `-count=10` の 10 回とも 1 周目で落ちた(L6c は `-race` でも 10 回とも)。HEAD では順がコードで決まっているので、いつも通る。mutant が落ちるのは、ロックの空きの確認が、ロックを放してから印を書くまでの隙間に入ったときだけなので、mutant の検出は確率による
  - A6: `TestRunWatchAll_SweepsTheInboxBeforeThePulse`(`internal/org/watch_all_test.go`)。サイクルの最初の herdr の `AgentGet` の時点で、`escalations.jsonl` に期限を過ぎた件の行がもうある。A6 では空のまま落ちる
  - S6: `TestRunWatchAll_StartedAtIsTakenBeforeWaitingForTheLock`(`internal/org/watch_all_test.go`)。probe 9 の 100 ms の sleep をやめ、`Org.Now` の最初の呼び出し(開始の読み取り)を合図にした。テストは `watch.lock` を持ったまま合図を待ち、時計を 1 分進め、開始と新しい時刻のあいだの頼みを書いてから、ロックを放す。使ったのは既存の `Org.Now` と `WatchAllParams.LockWait` だけ。HEAD は 1 サイクルで止まり、`started_at` は待つ前の時刻になる。S6 はロックを取るまで時計を読まないので合図が来ず、`LockWait` の 10 秒のあとロックを取れずに返り、落ちる
  - 残り: mutation の生き残りで本物の穴は、もうない。ほかの Test gaps の項目は変わらない

### 参考(テストの失敗ではない、cycle 1)

- 2 つ目の見張りは、ロックを取る前に `watching every org (interval=30s once=true …)` の banner を stdout に出してから断る(実機 1 の手順 4、Linux でも同じ)。banner は `runOrgWatchAll` が `RunWatchAll` の前に出すため。終了コードと pid の名指しは AC9 のとおり
- 同時の `--detach` のうち 1〜2 本は、見張りが動いているのに rc 1 になる(実機 2)。計画の AC12 の「子が先に終わったら終了コード 1」のとおりの動き。後半の PR で `ralph org start` が見張りを起動するとき、複数の start が同時に走ると、start が見張りの起動の失敗と読みうる。そのときは、rc 1 のあとに `ProbeWatch` が running なら成功として扱う、などを考えるとよい

### 判定(cycle 1)

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、Go 8 パッケージ)。全パッケージを `-count=1` で流し直して 0 fail、`-race`(org と cli)と、watch・判定・期限・受信箱・status のテストの `-race -count=3` も 0 fail。AC1〜AC16 と AC7b・AC12b・AC13b に behavioral test があり、すべて通る(AC17 は文書の AC で対象外)。依頼の 9 点の mutant は、P2 と S6 のほかはすべて落ち、82 件のうち 73 件が落ちた。実機では、`--detach`・`kill -9` のあとの判定と起動し直し・2 つ目の見張りの拒否・`--stop` が期待どおりに動き、プロセスは残らなかった(Linux のコンテナでも同じ)
- Fail: 0
- Blocked: なし。生き残った本物の穴 5 つ(P2、X3、L6、A6、S6)と Test gaps の残りは merge を止めない。probe 4・6・7・8・9 をテストに足すかは orchestrator が決める
