# Test report: seat-spawn-close

- Date: 2026-10-22(JST 15:13〜15:50、UTC 06:13〜06:50)
- Plan: `docs/plans/active/2026-10-22-seat-spawn-close.md`(試行の所見を直す系列の 4b。AC1〜AC10 の挙動、計画の Test plan の境界の場合、回帰)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目
- Scope: コードは 37aae73a(base 5850f89b = origin/main)で打った。途中で HEAD は verifier の a0ef094d に進んだが、足されたのは `docs/reports/verify-2026-10-22-seat-spawn-close.md` だけで、コードとテストは同じ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS 26.5.2(Darwin 25.5.0、arm64、12 CPU)、git 2.49.0、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8 GOPROXY=off`。toolchain はキャッシュ済みで、network は使っていない)
- Evidence: `docs/evidence/test-2026-10-22-seat-spawn-close.log`(各実行の出力、時刻と load average、coverage の関数ごとの値、mutant の定義と結果、canary の記録。ホームは `~`、session の scratchpad は `<scratch>` と書いた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-22-061359.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない。orchestrator の `GOFLAGS=-timeout=30m ./scripts/run-verify.sh`(37aae73a、rc 0、731 秒、`docs/evidence/verify-2026-10-22-060044.log`)は打ち直していない
- 追跡しているファイルは、この報告のほかは変えていない。mutant と probe は `go test -overlay` で scratchpad のファイルに差し替えて打ち、worktree のファイルには書いていない。worktree の `git status --porcelain --ignored` の開始時と終了時の差は、`.harness/cache/go-build/` の build cache と run-test.sh の evidence の log だけだった。`go env -w` は打っていない。本物の herdr・agmsg・osascript・Orca・端末・tmux・gh や network を打つものは実行していない。新しいテストと期待を変えたテストが本物のそれらに届かないことは 8 の canary で確かめた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS=-timeout=30m ./scripts/run-test.sh` | shell 43 本、Go 8 パッケージ | shell 43 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1 件(下) | 763 秒、rc 0。cli 268.1 秒、org 336.3 秒、config 0.5 秒。insights・driver・protocol・scaffold・upgrade は `(cached)` |
| 1b | `go test -count=1 ./internal/insights/` | 1 パッケージ | 1 | 0 | 0 | 0.8 秒。1 で `(cached)` だった insights は internal/org に依存するので打ち直した |
| 2 | `GOTOOLCHAIN=go1.25.8 GOPROXY=off go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` | 4 パッケージ | 4 | 0 | 0 | 331 秒、rc 0。org 329.3 秒、cli 266.5 秒、driver 1.4 秒、protocol 1.1 秒。1 の Go の部分と重ねて打った |
| 3 | 新しいテストと期待を変えたテスト(org 41 本、cli 7 本)を `-count=3 -v`(go1.26.0) | 48 × 3 = 144(subtest 216) | 144(216) | 0 | 0(`--- SKIP` は 0 行) | 27 秒。org 11.3 秒、cli 14.5 秒 |
| 4 | 3 と同じ(go1.25.8) | 144(subtest 216) | 144(216) | 0 | 0 | 30 秒。org 11.3 秒、cli 14.3 秒 |
| 5 | 3 と同じに `-race` | 144(subtest 216) | 144(216) | 0 | 0 | 46 秒。`WARNING: DATA RACE` は 0 行 |
| 6 | 時間に敏感な org 12 本と cli の `TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` を `-count=10 -v`。後ろで 7 と、自分で 420 秒で止まる perl の busy loop 12 本を走らせた | 130(subtest 140) | 130(140) | 0 | 0 | 69 秒、rc 0。load average は開始 26.5、org の終わり 52.9、終了 78.8 |
| 7 | `go test -count=1 -timeout 30m -coverprofile ./internal/org/... ./internal/cli/...`(go1.26.0) | 4 パッケージ | 4 | 0 | 0 | 561 秒、rc 0。org 552.3 秒、cli 447.6 秒、driver 3.1 秒、protocol 3.2 秒。6 と 9 と重なり、load average は最大で約 100 |
| 8 | 3 の 48 本を canary の PATH と偽の HOME で(下) | 48 | 48 | 0 | 0 | canary の呼び出しは 0 回 |
| 9 | mutant 47 本(下の表) | 47 | - | 46 本が赤 | - | 生き残りは L7 の 1 本 |
| 10 | 9 の L7 を org のパッケージ全体で(`-count=1 -timeout 30m`) | 1 パッケージ | 1(生き残り) | 0 | 0 | 370 秒、rc 0 |
| 11 | ロックの断りが待たないことを見る使い捨ての probe(overlay で足した test file。コミットしない) | 1 | HEAD で緑 | L7 で赤 | 0 | 1.6 秒、3.2 秒 |

- 1 の scope は changed で、`==> Language scope: changed (changed_languages)`、pack は golang。shell の 43 本は local verifier が test mode で走らせる全 suite で、どの suite も最後の行が `OK` だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばしている
- 1 の Go は `-count=1` を付けていないが、変えた org と cli、それに config は実際に走った。`(cached)` の driver・protocol・scaffold・upgrade は、`go list -test -deps` で internal/org にも internal/cli にも依存しないことを確かめた。insights は internal/org に依存するので 1b で打ち直した。org と cli は 1、2、7 の 3 回、パッケージ全体で走った
- 1 の load average(1 分)は開始時 6.0、Go の間 6〜14、終了時 8.0。2 は開始 5.8 で、1 の Go と重なった間は 6〜14 だった
- 3〜5 の 48 本は下の「AC とテストの対応」の表のテストで、どれも実行され、飛ばされていない。org の 41 本はこの branch で足した 25 本と、期待を変えた 16 本である
- 6 の 12 本は、spawn の期限を 500 ms にした 4 本(`SpawnDeadlineRunsOut`、`FailureAfterSpawnDeadline`、`HelloTimeout_Leaves`、`HelloFailsInsideDeadline`)、`DriverCallTimeout` を 20 ms にした 3 本(`HelloFailure_UnansweredLeave`、`FailureAfterPane_CloseTimesOut`、`FailureBeforeSpawnDeadline`)、ロックの 3 本、`StaleSeat_PaneNotClosed`、`SeatLeftOpen_AfterRefusedRetry` である。所要時間は負荷の下でも変わらなかった。500 ms の 2 本は 0.50〜0.52 秒、`HelloFailsInsideDeadline` は 0.80〜0.81 秒、`SpawnDeadlineRunsOut` は 1.51〜1.53 秒、cli の HelloPastTimeoutMS は 3.11〜3.20 秒。期限の前の step が負荷で期限を食いつぶした回はない
- 7 は coverage を取るためだが、6 の busy loop と 9 の mutant が重なった、いちばん重い負荷の下でのパッケージ全体の実行にもなった。load に弱いことが分かっているテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time` ほか)も落ちなかった
- 8 の canary: `herdr`・`agmsg`・`osascript`・`open`・`tmux`・`gh`・`orca`・`claude`・`codex`・`curl`・`wget`・`nc`・`ssh` を、呼ばれたら記録して exit 97 する script にした dir を PATH の先頭に置き、`HOME` を空の scratch の dir にした(`GOMODCACHE`・`GOCACHE` は元の値、`GOPROXY=off`)。cli のテストの `setupOrgStubPATH` は自分の stub の herdr をその前に置くので、canary が受けるのは stub から漏れた呼び出しである。感度: `setupOrgStubPATH` から stub を PATH に置く 1 行を消した写しを overlay にすると、canary は `herdr workspace create` を 5 回記録し、cli の 5 本が赤になった。新しい cli の `TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` は本物の `driver.ExecRunner` で stub の `send.sh`(`exec sleep 30`)を打つが、20 回以上打ったあと `sleep 30` のプロセスは 1 つも残っていなかった

## AC とテストの対応

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestOrgSpawn_FailureAfterPane_SendsCtrlCThenClosesPane`(`internal/org/spawn_test.go:1541`、agent_start・prompt_file・agmsg_join・agmsg_announce の 4 subtest)、`TestOrgSpawn_FailureAfterSpawnDeadline_ClosesPaneUnderFreshDeadlines`(1611)、`TestOrgSpawn_FailureAfterPane_PaneNotFound_CountsAsClosed`(1711)、`TestOrgSpawn_FailureAfterPane_CtrlCFails_CloseStillCloses`(1737)、`TestOrgSpawn_FailureInjection_WorkspaceCreate_NoCompensationAttempted`(1327)、`..._TabCreate_NoCompensationAttempted`(1353)、`..._AgentStart_CompensatesExistingPane`(1376)、`..._AgmsgSend_CompensatesExistingPane`(1405)、`..._AgmsgJoin_SeatJoinFails_CompensatesExistingPane`(1434)、`TestOrgSpawn_PromptFileWriteFailure_FailsStepPromptFile`(2739)、`TestOrgSpawn_DryRun_PromptFilePathError_FailsStep`(3327)、cli の `TestOrgSpawn_FailureInjection_AgentStart_CompensatesPane`(`internal/cli/org_test.go:529`) | pane を作ったあとの 4 つの step で、herdr の呼び出しが C-c、close の順で終わり、`spawn_failed` が pane を持ち、Details が `compensation=ctrl_c=ok pane=closed`、座席は active でない。Join が spawn の期限を使い切ったあとでも、C-c と close は生きていて期限を持つ ctx を受け取る。close の not found は `pane=already closed` で `spawn_close_failed` を書かない。C-c が失敗しても close は打たれる。workspace_create・tab_create・dry run の失敗は `compensation=no pane to compensate` で、C-c も close も打たない(dry run は herdr を 1 回も呼ばない)。cli では stub の herdr の log が `pane send-keys <pane> C-c`、`pane close <pane>` で終わる。mutant C1〜C5 が赤 |
| AC2 | `TestOrgSpawn_HelloTimeout_LeavesThenClosesPane`(1640)、`TestOrgSpawn_HelloFailure_UnansweredLeave_TimesOut`(1684) | HELLO が期限を越えると、Leave の時点の herdr の呼び出しは workspace_create、tab_create、agent_start だけで、そのあと C-c、close が続く(順は Leave、C-c、close)。Leave・C-c・close の 3 つとも生きていて期限を持つ ctx を受け取る。答えない Leave は `DriverCallTimeout` で切られ、Details に `leave=failed: context deadline exceeded`、pane は閉じる。mutant C6〜C8 が赤 |
| AC3 | `TestOrgSpawn_FailureAfterPane_CloseFails_RecordsSpawnCloseFailed`(1767、agent_start・agmsg_announce の 2 subtest)、`TestOrgSpawn_FailureAfterPane_CloseTimesOut_RecordsSpawnCloseFailed`(1837)、`TestOrgStop_SeatLeftActiveBySpawnCloseFailure_ClosesRecordedPane`(`internal/org/verbs_test.go:2113`)、`TestOrgStopAll_StopsSeatLeftActiveBySpawnCloseFailure`(`internal/org/verbs_all_test.go:265`) | close が not found 以外で失敗すると、`spawn_failed` を書かずに `spawn_close_failed` を書き、座席の欄・pane_id・agmsg の team(join のあとだけ)・Details(step、error、補償)を持つ。`spawn_close_failed` は state event でも active event でもなく、座席は `spawn_step` の pane で active、`o.Status` にも active で出る。エラーは step の原因と close の誤りを両方包み、`pane "pane-7" not closed`、`herdr pane close pane-7`、`ralph org stop --org-id org-a --seat seat-1` を含む。close の時間切れも同じ。そのあと 1 座席の stop と `stop --all` が、`spawn_step` の記録した pane_id だけを閉じて `stopped` を書く。mutant F1〜F3 が赤 |
| AC4 | `TestOrgSpawn_StaleSeat_ConfirmsPaneBeforeClosing`(2067、5 subtest)、`TestOrgSpawn_StaleSeat_NoPaneRecorded_GoesOnWithNoCheck`(2119)、`TestOrgSpawn_StaleSeat_PaneNotClosed_RefusesTheSpawn`(2150、6 subtest)、`TestOrgSpawn_StaleSeat_CallersOwnPane_RefusedWithNothingWritten`(2228)、期待を変えた `TestOrgSpawn_StaleInFlight_*`(1058、1151、1207)と `TestOrgSpawn_Reserve_ExistingLeader_Phase2`(5341) | stale の補償は、どの herdr の呼び出しより先に記録の pane を get で確かめ(呼び出しの列は pane_get、tab_get、workspace_get、pane_send_keys、pane_close)、確かめたときだけ C-c と close を打つ。check の not found と close の not found は `pane=already closed`、tab か workspace の label が違えば C-c も close も打たずに `pane=left untouched, labelled differently: ...` で進み、新しい spawn は spawned になる。get の失敗・時間切れ、tab か workspace を herdr が知らない場合、close の失敗・時間切れの 6 通りでは、`spawn_close_failed` だけを書いて断り(`spawn_started`・`rejected`・`spawn_failed`・receipt なし、新しい saga の呼び出しなし)、座席は記録の pane で active のまま。pane の記録がない座席は herdr を呼ばずに進む。記録の pane が打った本人の pane なら何も書かずに断る(計画にない S2 の追加)。mutant S1〜S9、V1〜V3 が赤 |
| AC5 | cli の `TestOrgSpawnAndStart_TimeoutMSHelp_DescribesTheWholeSpawnDeadline`(`internal/cli/org_spawn_timeout_test.go:17`)、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`(51)、`TestOrgSpawn_SpawnDeadlineRunsOut_ErrorHintsAtTimeoutMS`(`internal/org/spawn_test.go:1873`、3 subtest)、`TestOrgSpawn_FailureBeforeSpawnDeadline_NoTimeoutMSHint`(1944、2 subtest)、`TestOrgSpawn_HelloFailsInsideDeadline_LeavePastIt_NoTimeoutMSHint`(1986) | spawn と start の help が spawn 全体の期限と step の並び、補償の 10 秒を書き、`per-step` を含まない。期限切れで落ちた step(Join、HELLO と close の失敗、agent start の busy の再試行)のエラーは step の原因のすぐあとに期限の値(500 ms)と `--timeout-ms` を上げる案内を持ち、Details は持たない。agent start の busy では原因が `DeadlineExceeded` を包まなくても案内が付く。期限の内側の失敗には付かない(close が自分の 10 秒で時間切れになっても付かない)。HELLO が期限の内側で失敗し、Leave が期限を越えても付かない。cli では本物の ExecRunner で 3000 ms の期限が切れ、stdout の `spawn failed:` の行とエラーの両方に案内が出る。mutant H1〜H6 が赤。skill の文は /verify の範囲で、テストはない |
| AC6 | なし(実機の確かめ) | `docs/evidence/seat-spawn-close-2026-10-22.md` の (a)〜(f) を読んだだけで、打ち直していない(本物の herdr を使うため)。(e) は計画の「fake の herdr を使うテストの出力」ではなく実機で取ってある |
| AC7 | なし(記録) | /verify の範囲 |
| AC8 | 1、1b、2 | 変えたパッケージの Go のテストが go1.26.0(1、7)と go1.25.8(2)で通る。run-verify の rc 0 は orchestrator の実行(上の Evidence) |
| AC9 | `TestOrgSpawn_SeatLock_RefusesASecondSpawnOfTheSeat`(`internal/org/spawn_lock_test.go:87`)、`TestOrgSpawn_SeatLock_OtherSeatsOrgsAndDryRunsDoNotWait`(141)、`TestOrgSpawn_SeatLock_LockThatCannotBeTaken_FailsWithNothingWritten`(180)、前からある `TestOrgSpawn_Concurrent*` | 1 つ目の spawn を座席の Join で止めている間、ロックファイル `spawn-org-a_seat-1.lock` があり、同じ座席の 2 つ目は `errFileLockHeld` を包む rejected で断られ、event・receipt・herdr と agmsg の呼び出しは増えず、1 つ目の pane に C-c も close も届かない。止めを外すと 1 つ目は spawned、そのあとの打ち直しは冪等に返る。別の座席、別の org の同じ座席 id、同じ座席の dry run は待たずに通る。ロックの directory が作れないときは held の断りでない Failed で、driver も receipt も触らない。mutant L1〜L6 が赤(前からある並行のテストも L3 では 1 本、L4 では 3 本が赤)。断りが待たないこと(L7)は固まっていない(G1) |
| AC10 | `TestOrgSpawn_InFlightSeat_RefusedRetryWritesNothing`(`internal/org/spawn_test.go:2264`、6 subtest)、`TestOrgSpawn_SeatLeftOpen_AfterRefusedRetry`(2420、3 subtest)、期待を変えた `TestOrgSpawn_StaleInFlight_StatelessEnvelopeViolation_RejectedBeforeCompensation`(1099)、`TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn`(2321)、`TestStartFeature_PrecheckStopHintRepeatsInboxHintStateDir`(`internal/org/feature_test.go:685`)、期待を変えた codex の 3 本(`internal/org/verbs_test.go:4959`、5025、5082) | `spawn_close_failed` のあとと落ちた spawn のあとの 2 通り × model・codex の permission・`--scope` の 3 通りで、断られた再試行は event も receipt も書かず、driver を呼ばず、座席は pane-7 で active のまま。エラーは検査の原因、途中の座席とその pane、stop の打ち方を含む。そのあと 1 座席の stop、`stop --all`、正しい再試行のどれも記録の pane だけを閉じる。`start --plan` の事前検査も Spawn と同じ文を出す。codex の 3 本は、古い binary が書いた `rejected` を足して #173 の場面を作り直し、検査は強めてある。mutant R1〜R9 が赤 |
| self-review L1 の直し | `TestOrgSpawn_StopHints_RepeatInboxHintStateDir`(2357、4 経路 × 3 通り)、cli の `TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir`(`internal/cli/org_test.go:2462`) | 4 つの経路のエラーの stop の打ち方が、`InboxHintStateDir` があるときだけ引用符つきの `--state-dir` で終わる。CLI では `--state-dir` で付き、`RALPH_ORG_STATE_DIR` では付かない。mutant T1・K1 が赤 |

計画の Test plan の境界の場合と、それを確かめるテスト:

| 境界の場合 | テスト |
| --- | --- |
| `PaneClose` が not found | `FailureAfterPane_PaneNotFound_CountsAsClosed`、`StaleSeat_ConfirmsPaneBeforeClosing` の「not found at the close」 |
| C-c だけ失敗して close は通る | `FailureAfterPane_CtrlCFails_CloseStillCloses`(failStep の道だけ。G5) |
| close が時間切れ | `FailureAfterPane_CloseTimesOut_RecordsSpawnCloseFailed`、`StaleSeat_PaneNotClosed_RefusesTheSpawn` の「the close times out」 |
| stale の座席の pane が label の違う pane を指す | `StaleSeat_ConfirmsPaneBeforeClosing` の tab と workspace の 2 subtest |
| stale の座席に pane の記録がない | `StaleSeat_NoPaneRecorded_GoesOnWithNoCheck` |
| `spawn_close_failed` のあとの同じ座席の spawn | `SeatLeftOpen_AfterRefusedRetry` の「a valid retry」(あいだの断られた再試行は何も書かないので、台帳は `spawn_close_failed` の直後と同じ)、`InFlightSeat_RefusedRetryWritesNothing` |
| 同じ座席の spawn が重なる | `SeatLock_RefusesASecondSpawnOfTheSeat` |
| 途中の座席への不正な再試行 | `InFlightSeat_RefusedRetryWritesNothing` |
| ロックファイルの directory が作れない | `SeatLock_LockThatCannotBeTaken_FailsWithNothingWritten` |
| 期限の切れた spawn の ctx でも C-c と close が打たれる | `FailureAfterSpawnDeadline_ClosesPaneUnderFreshDeadlines`、`HelloTimeout_LeavesThenClosesPane`(fake が受けた ctx の期限を見る) |
| spawn の失敗が stdout の `spawn failed:` に出る | cli の `HelloPastTimeoutMS_PrintsTheDeadlineHint` |

## Mutant

HEAD の 1 ファイルを、文字列 1 か所(1 回だけ現れることを確かめた)を置き換えた写しに `-overlay` で差し替えて打った。org の mutant は `^(TestOrgSpawn_|TestSpawnPrecheckErr_|TestStartFeature_|TestOrgStop_|TestOrgStopAll_)`(216 本)を、cli の K1 は `^(TestOrgSpawn_|TestOrgSpawnAndStart_|TestOrgStart_)` を打った。差分と出力は evidence にある。S9 の最初の写しは `gone` が使われなくなって build が通らなかったので、`case gone && false:` に直して打ち直した(下の表はその結果)。ほかの 46 本の log に build の失敗はない。

| ID | 変えたところ | 結果(赤にしたテスト) |
| --- | --- | --- |
| L1 | 座席のロックを取らない | 赤(SeatLock_RefusesASecond、LockThatCannotBeTaken) |
| L2 | dry run もロックを取る | 赤(SeatLock_OtherSeatsOrgsAndDryRuns) |
| L3 | ロックの名前を org ごとにする | 赤(SeatLock の 2 本、`TestOrgSpawn_ConcurrentSpawns_MaxTotalSeatsNeverExceeded`) |
| L4 | ロックの名前を座席 id だけにする(org をまたいで同じ名前) | 赤(SeatLock の 2 本、前からある `TestOrgSpawn_Concurrent*` 3 本) |
| L5 | held を Failed の道に流す | 赤(SeatLock_RefusesASecond) |
| L6 | 取った直後にロックを外す | 赤(SeatLock_RefusesASecond) |
| L7 | ロックを 1 回だけ試さず、2 秒待つ | **生き残り**(G1)。11 の probe で赤 |
| R1 | 途中の座席にも `reject()` を使う(`rejectUnlessInFlight` の分岐を消す) | 赤(InFlightSeat、SeatLeftOpen、StatelessEnvelopeViolation、StopHints、SpawnPrecheckErr_InFlightSeatRefusal、codex の 3 本) |
| R2 | エンベロープの検査だけ `reject()` に戻す | 赤(R1 と同じ) |
| R3 | permission の検査だけ `reject()` に戻す | 赤(InFlightSeat の codex の 2 subtest) |
| R4 | `--scope` の検査だけ `reject()` に戻す | 赤(InFlightSeat の `--scope` の 2 subtest) |
| R5 | 事前検査が途中の座席の文を付けない | 赤(SpawnPrecheckErr_InFlightSeatRefusal、StartFeature_PrecheckStopHint) |
| R6 | 事前検査が `hintStateDir` を捨てる | 赤(R5 と同じ) |
| R7 | `StartFeature` が事前検査に `""` を渡す | 赤(StartFeature_PrecheckStopHint) |
| R8 | 途中の座席の文から pane を落とす | 赤(InFlightSeat、StatelessEnvelopeViolation、SpawnPrecheckErr_InFlightSeatRefusal、StartFeature_PrecheckStopHint) |
| R9 | `spawnInFlight` が `spawn_step` を途中と見なさない | 赤(InFlightSeat、SeatLeftOpen、StaleSeat の 3 本、StopHints、codex の 3 本ほか) |
| C1 | `closeSpawnPane` が close を打たない | 赤(25 本。FailureAfterPane の 5 本、FailureAfterSpawnDeadline、FailureInjection の 3 本、HelloTimeout、stop と stop --all の 2 本ほか) |
| C2 | C-c が失敗したら close を打たずに返る | 赤(FailureAfterPane_CtrlCFails) |
| C3 | close の not found を閉じ済みと見なさない | 赤(FailureAfterPane_PaneNotFound、StaleSeat_ConfirmsPane の「not found at the close」) |
| C4 | close を `context.Background()` で打つ | 赤(CloseTimesOut、FailureAfterSpawnDeadline、FailureBeforeSpawnDeadline、HelloTimeout、StaleSeat_PaneNotClosed) |
| C5 | C-c を `context.Background()` で打つ | 赤(FailureAfterSpawnDeadline、HelloTimeout) |
| C6 | Leave を `context.Background()` で打つ | 赤(HelloFailure_UnansweredLeave、HelloTimeout) |
| C7 | HELLO の失敗で Leave の前に pane を閉じる | 赤(HelloTimeout、HelloFailure_UnansweredLeave、CloseFails、SendsCtrlC、AgmsgSend_CompensatesExistingPane) |
| C8 | spawn の ctx を Leave のあとで読む | 赤(HelloFailsInsideDeadline_LeavePastIt) |
| H1 | 案内を `errors.Is(cause, DeadlineExceeded)` で決める | 赤(SpawnDeadlineRunsOut の agent start busy) |
| H2 | 案内を付けない | 赤(SpawnDeadlineRunsOut) |
| H3 | 案内をいつも付ける | 赤(FailureBeforeSpawnDeadline、HelloFailsInsideDeadline) |
| H4 | 案内を Details にも入れる | 赤(SpawnDeadlineRunsOut) |
| H5 | agent start の再試行の道で ctx を渡さない | 赤(SpawnDeadlineRunsOut) |
| H6 | agmsg_join の道で ctx を渡さない | 赤(SpawnDeadlineRunsOut) |
| F1 | 閉じられなかったときも `spawn_failed` を書く | 赤(CloseFails、CloseTimesOut、InFlightSeat、SeatLeftOpen、stop と stop --all の 2 本ほか) |
| F2 | `spawn_close_failed` に team を書かない | 赤(CloseFails の agmsg_announce) |
| F3 | close の失敗を無視して `spawn_failed` の道に進む | 赤(F1 の 8 本と StopHints) |
| S1 | stale の pane を確かめない | 赤(StaleSeat の 2 本、StaleInFlight の 2 本、Reserve_ExistingLeader_Phase2、StopHints) |
| S2 | label の違いを確かめられない扱いにする(断る) | 赤(StaleSeat_ConfirmsPane) |
| S3 | label が違っても閉じる | 赤(StaleSeat_ConfirmsPane) |
| S4 | 確かめられなくても進む | 赤(StaleSeat_PaneNotClosed、StopHints) |
| S5 | close が失敗しても進む | 赤(StaleSeat_PaneNotClosed、StopHints) |
| S6 | 打った本人の pane の断りを消す | 赤(StaleSeat_CallersOwnPane、StopHints) |
| S7 | 断るときに `spawn_failed` を書く | 赤(StaleSeat_PaneNotClosed) |
| S8 | 断りを Rejected にする | 赤(StaleSeat_PaneNotClosed、StopHints) |
| S9 | check の not found を閉じ済みと見なさない(`case gone && false:`) | 赤(StaleSeat_ConfirmsPane) |
| T1 | `stopSeatCommand` が `--state-dir` を捨てる | 赤(StopHints、SpawnPrecheckErr_InFlightSeatRefusal、StartFeature_PrecheckStopHint) |
| V1 | `labelMismatchError.Is` がいつも false | 赤(StaleSeat_ConfirmsPane) |
| V2 | workspace の label の違いを `fmt.Errorf` に戻す | 赤(StaleSeat_ConfirmsPane の workspace) |
| V3 | tab の label の違いを `fmt.Errorf` に戻す | 赤(StaleSeat_ConfirmsPane の tab) |
| K1 | CLI が `InboxHintStateDir` を入れない | 赤(cli の `TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir`) |

- 47 本のうち 46 本が赤、生き残りは L7 の 1 本。L7 は 10 で org のパッケージ全体でも打ったが、それでも全部緑だった(`ok internal/org 369.8s`)。どのテストも 2 つ目の spawn の待ち時間を見ていない
- 11 の probe: 1 つ目の spawn を Join で止めたまま 2 つ目を打ち、エラーが `the lock is held after 0s`(`acquireHeldFileLock` の文、`internal/org/lockfile.go:183`)を含むことを見る使い捨てのテストを overlay で足した。HEAD では緑(0.00 秒)、L7 と重ねると赤(2.01 秒待ってから `after 2s` で断る)

## Coverage

- Statement: org 94.7%、driver 95.0%、protocol 100.0%、cli 88.4%(7 の go1.26.0、パッケージごと)。4 パッケージの合計は 92.3%。#248 の報告の値(org 94.7%、driver 94.6%、cli 88.4%)とほぼ同じ
- Branch: Go の標準のツールにはないので測っていない
- Function(変えた関数): `rejectUnlessInFlight`・`inFlightSeatErr`・`stopSeatCommand`・`spawnInFlight`・`spawnPrecheckErr`・`failStep`・`spawnDeadlineHint`・`compensateStale`・`closeSpawnPane`・`spawnLockName`・`confirmSeatPane`・`confirmOrgWorkspace`・`labelMismatchError` の `Error` と `Is`・`orgSpawnTimeoutFlagUsage` は 100%。`failStepWithNote` 92.9%、`refuseStaleRespawn` 80.0%、`compensateLeave` 83.3%、`Spawn` 92.4%、`newOrgSpawnRuntimeAt` 90.0%
- Notes: この branch で足した行のうち通らない block は 2 つで、どちらも `spawn_close_failed` の追記が失敗したときに誤りを `errors.Join` で足す行(`internal/org/spawn.go:2471-2473` の `failStepWithNote`、`2605-2607` の `refuseStaleRespawn`)である(G2)。`compensateLeave` の `team == "" || agentID == ""`(2503-2505)、`Spawn` の manifest の lock と追記の失敗(884-886、977-979 ほか)は base からある行で、追記の失敗で pane が閉じない道は tech-debt の行に記録がある

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | 1〜8 のどの実行でも失敗はなかった | - |

load に弱いことが分かっているテストも、org と cli をパッケージ全体で打った 1、2、7 で 1 度も落ちなかった。7 は load average が最大で約 100 だった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 所見 F4: HELLO の時間切れのあと pane と herdr の agent の名前が残り、次の spawn が `agent_name_taken` で落ちる | ralph の側の補償を AC1〜AC3 のテストが固める | 上の AC の表と mutant C1〜C8・F1〜F3。本物の herdr での確かめは S4 の evidence(`docs/evidence/seat-spawn-close-2026-10-22.md`)で、今回は打っていない |
| 今までの spawn・stop・stop --all・disband のテスト | 通る | 1、2、7。期待を変えたテストは、Details の `C-c sent` を新しい語に、stale の補償の呼び出しの列を確かめと close を含む 5 つに、途中の座席への拒否を「何も書かない」に、cli の最後の herdr の呼び出しを C-c と close の 2 つに直したもので、どれも計画の AC1・AC4・AC10 の変更である |
| 期待を変えたテストで弱めた assert がない | 弱めていない | テストの diff の削除行(コメントを除く)を全部見た。消えた assert はどれも新しい期待で置き換わっている。stale の座席を seed するテストには `putSeatPaneInHerdr` で fake の herdr に pane と label を足し、確かめを通す形にした。codex の 3 本は、Outcome だけを見ていたところに、receipt がないことと座席が `spawn_step` に残ることの検査が足された |
| dry run の失敗は `no pane to compensate` のまま | 通る | `TestOrgSpawn_DryRun_PromptFilePathError_FailsStep`(`internal/org/spawn_test.go:3327`)。herdr を 1 回も呼ばないことの検査も足された |
| 今の並行の spawn のテストがロックで待たない | 通る | `TestOrgSpawn_ConcurrentSpawns_*`、`TestOrgSpawn_ConcurrentReservations_*` が 1、2、7 で緑。L3・L4 でこれらが赤になるので、ロックの名前の分け方をこれらも見ている |

## Test gaps

- G1(LOW): 座席のロックの断りが待たないこと(`acquireHeldFileLock(..., "spawn", 0)`、`internal/org/spawn.go:613`)を固めるテストがない。L7(2 秒待つ)が生き残った。計画の D5 は「待たずに 1 回だけ試し」と書き、待つ案を「最長の待ちが倍になる」として退けている。`TestOrgSpawn_SeatLock_RefusesASecondSpawnOfTheSeat`(`internal/org/spawn_lock_test.go:105`)の期待の文の一覧に `the lock is held after 0s` を足せば足りる。時刻を測らずに済み、11 の probe で HEAD で緑、L7 で赤になることを確かめた
- G2(LOW): `spawn_close_failed` の追記が失敗したとき、その誤りを返すエラーに足すこと(`internal/org/spawn.go:2471-2473`、`2605-2607`)を通るテストがない(coverage 0)。足さなくても座席は `spawn_step` のまま active で、stop で拾えるのは変わらない。stop の `stop_failed` の追記の失敗(`internal/org/verbs.go:1170`)にもテストはなく、この repo の今までの扱いと同じである。足すなら、fake の herdr の close の中で manifest のファイルを読み取り専用にしてから誤りを返す形で書ける
- G3(INFO): AC3 の `spawn_close_failed` を見る表(`internal/org/spawn_test.go:1767`)は agent_start と agmsg_announce の 2 つの step だけで、prompt_file と agmsg_join で close が失敗する行がない。4 つとも同じ `failStepWithNote` を通り、team が `""` になる agmsg_join の場合は agent_start の行が同じ形を見ている
- G4(INFO): CLI の水準で spawn の close が失敗する場合(stdout、終了コード、`spawn_close_failed`)を見るテストがない。cli の herdr の stub は `ORG_STUB_CLOSE_FAIL_IDS` を持つが、stop と director のテストだけが使う。エラーの文は org のテストが固め、CLI での見え方は実機の evidence の (e) にある
- G5(INFO): stale の補償で C-c だけ失敗して close が通る場合のテストがない。同じ `closeSpawnPane` を failStep の道の `TestOrgSpawn_FailureAfterPane_CtrlCFails_CloseStillCloses`(`internal/org/spawn_test.go:1737`)が見ている
- G6(INFO): `spawnPrecheckErr` の doc(`internal/org/spawn.go:1316-1322`)は、事前検査が通ったあとで `compensateStale` が断りうると書くが、`StartFeature` でその道(leader の座席の stale の pane が確かめられない)を通すテストはない。`TestSpawnPrecheckErr_MatchesSpawn` の stale の 2 行(`internal/org/feature_test.go:1393`、`1446`)は `spawn_started` だけを seed し、pane を記録しないので、`compensateStale` は確かめずに進む
- 確かめていないこと: 本物の herdr と agmsg での spawn(AC6 は evidence を読んだだけ)、`/org` skill の文(AC5 の後半、/verify の範囲)、記録(AC7)、ロックを取らない古い binary の spawn との競合(計画の範囲外。tech-debt の行がある)、動いている spawn の途中の stop(Non-goals、tech-debt の行がある)、codex の座席の spawn の失敗

## Verdict

- Pass: run-test.sh(shell 43 本、Go 8 パッケージ)、org・cli・driver・protocol の `-count=1` を go1.26.0 と go1.25.8 で、新しいテストと期待を変えたテスト 48 本の `-count=3 -v` を go1.26.0・go1.25.8・`-race` で(どれも 144/144、subtest 216/216、SKIP 0)、時間に敏感な 13 本の `-count=10` を load average 26〜79 の下で(130/130)、coverage の実行を load average 最大約 100 の下で、canary の PATH で 48 本(本物の herdr などへの呼び出し 0 回、感度あり)。mutant は 47 本のうち 46 本が赤で、生き残りは G1 の L7 だけ
- Fail: なし
- Blocked: なし
