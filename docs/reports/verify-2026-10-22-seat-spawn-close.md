# Verify report: seat-spawn-close

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-spawn-close.md`(Status: Approved。`./scripts/plan-visual.sh digest` は `550d25183df2` で、`- Approved:` 行と一致)
- Verifier: verifier サブエージェント(Claude Code、パイプライン cycle 1 / 2)
- Scope: `git diff 5850f89b...HEAD`(HEAD 37aae73a、25 ファイル、+2535/-265)。S1 80780740、S2 97782369、S3 abb11e8c、S4 57a124a4、S5 a7133901、self-review の報告 0a7ef72e、その直し dde26a8f・70b23052 と記録 3edc8590・37aae73a を含む。判定は HEAD の tree に対して行った。self-review の報告 `docs/reports/self-review-2026-10-22-seat-spawn-close.md` は Merge: マージ可(LOW 10)
- Evidence: run-static-verify.sh が書いた `docs/evidence/verify-2026-10-22-060129.log`(`docs/evidence/*.log` の gitignore によりコミットしない)。S4 の生の出力は session の scratchpad の `ssc4b-raw/`(`run-s4.sh`、`s4.log`、`manifest.jsonl`)を読んだ

## Spec compliance

テストの実行は `/test` の担当なので、AC1〜AC5、AC9、AC10 はコードとテストの本文を読んで確かめた。AC6 は evidence の文書を S4 の生の出力と突き合わせた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: pane を作ったあとの step の失敗で C-c と `PaneClose` を 1 回 10 秒の期限で打つ。not found は閉じ済み。Details の `compensation=`。pane の前の失敗と dry run は `no pane to compensate` | 満たす。実行は `/test` | `failStepWithNote`(`internal/org/spawn.go:2454`)が `closeSpawnPane`(2635 行)を呼ぶ。`closeSpawnPane` は `callWithTimeout`(`internal/org/verbs.go:786`、`context.Background()` に `driverCallTimeout`)で C-c と close を打ち、spawn の ctx を受け取らない。label の確かめはない。close の not found は `pane=already closed`(2651 行)。`failStep` の呼び出しは 10 か所で、pane を渡すのは `agent_start`(999、1064 行)、`prompt_file`(1047、1050 行)、`agmsg_join`(1110 行)、`agmsg_announce`(1143 行)。`workspace_create`(967 行)、`tab_create`(972 行)、dry run(2255、2267 行)は pane が空で、`no pane to compensate` になる。`spawn.go` に残る `context.Background()` は spawn の期限(677 行)だけ。テスト: `TestOrgSpawn_FailureAfterPane_SendsCtrlCThenClosesPane`(4 つの step、C-c のあとに close、`spawn_close_failed` なし、座席は active でない)、`TestOrgSpawn_FailureAfterSpawnDeadline_ClosesPaneUnderFreshDeadlines`(切れた spawn の期限のあとでも、どちらの ctx も生きていて期限を持つ)、`TestOrgSpawn_FailureAfterPane_PaneNotFound_CountsAsClosed`、`TestOrgSpawn_FailureAfterPane_CtrlCFails_CloseStillCloses`。pane の前の失敗は `spawn_test.go:1350`・`1373`、dry run は 3361 行が `no pane to compensate` を見る |
| AC2: HELLO の期限切れでも pane を閉じて `spawn_failed`。Leave に `driverCallTimeout` | 満たす。実行は `/test` | `compensateLeave`(2502 行)は `callWithTimeout` で Leave を打つ。HELLO の失敗(1128〜1144 行)は spawn の ctx を Leave の前に読み(1141 行)、Leave、C-c、close の順に打つ。`TestOrgSpawn_HelloTimeout_LeavesThenClosesPane` は、Leave の時点の herdr の呼び出しが `workspace_create, tab_create, agent_start` で、最後が `pane_send_keys, pane_close` になることと、3 つの呼び出しの ctx を見る。`TestOrgSpawn_HelloFailure_UnansweredLeave_TimesOut` は答えない Leave が 20 ms で切れて `leave=failed: context deadline exceeded` になることを見る |
| AC3: close の失敗(時間切れを含む)で `spawn_close_failed`、座席は active、エラーに pane の id と 2 つのコマンド。そのあと stop と `stop --all` が `spawn_step` の pane_id で閉じる | 満たす。実行は `/test` | 2464〜2475 行が `spawn_failed` の代わりに `EventSpawnCloseFailed`(`internal/org/seat.go:60`)を、座席の欄、pane_id、team、Details で書く。`stateEvents`(`seat.go:125`)と `activeEvents`(143 行)にこの event はなく、座席は最後の `spawn_step` のまま active になる。エラーは `herdr pane close <id>` と `stopSeatCommand`(2195 行)を含む。テスト: `TestOrgSpawn_FailureAfterPane_CloseFails_RecordsSpawnCloseFailed`(agent_start と agmsg_announce の 2 通り、欄、`isStateEvent`・`isActiveEvent` が偽、Status が active)、`TestOrgSpawn_FailureAfterPane_CloseTimesOut_RecordsSpawnCloseFailed`、`TestOrgStop_SeatLeftActiveBySpawnCloseFailure_ClosesRecordedPane`(`internal/org/verbs_test.go:2113`)、`TestOrgStopAll_StopsSeatLeftActiveBySpawnCloseFailure`(`internal/org/verbs_all_test.go:265`)。どちらの stop のテストも、閉じた pane が `spawnLeavingPaneOpen` が `spawn_step` から読んだ pane_id であることと、`stopped` の pane_id を見る |
| AC4: `compensateStale` は `confirmSeatPane` のあとでだけ C-c と close。not found と閉じたときは `spawn_failed`。label が違えば触らず `spawn_failed`。確かめられないか close の失敗は断って `spawn_close_failed`。pane の記録がない座席は今までどおり | 満たす。実行は `/test` | `compensateStale`(2558〜2590 行)の switch: `gone` は `ctrl_c=skipped pane=already closed`、`errLabelMismatch`(`verbs.go:880`)は `pane=left untouched, labelled differently`、ほかの err は `ctrl_c=skipped: pane not confirmed pane=not closed: <err>`(`stopSeatPane` の語と同じ。`verbs.go:1196`)で断る、確かめたときだけ `closeSpawnPane`。断るのは `refuseStaleRespawn`(2597 行)で、`spawn_close_failed` だけを書き、Phase 2 に進まないので `spawn_started` も `rejected` も書かない。label の違いは、tab の label の違いと、`confirmOrgWorkspace` の違いを外側の `%w` で包んだものだけが `errors.Is` で当たり、tab が見つからない場合は `%v` で包むので当たらない。テスト: `TestOrgSpawn_StaleSeat_ConfirmsPaneBeforeClosing`(5 通り。最初の herdr の呼び出しが `pane_get`)、`TestOrgSpawn_StaleSeat_PaneNotClosed_RefusesTheSpawn`(6 通り。足されるのは `spawn_close_failed` 1 件だけ、receipt なし、新しい saga の呼び出しなし)、`TestOrgSpawn_StaleSeat_NoPaneRecorded_GoesOnWithNoCheck`。計画にない追加の自分の pane の断り(Progress の S2 に記録)は `TestOrgSpawn_StaleSeat_CallersOwnPane_RefusedWithNothingWritten` |
| AC5: help、期限切れのエラーの値と案内、`/org` skill と写し | 満たす。実行は `/test` | `orgSpawnTimeoutFlagUsage`(`internal/cli/org.go:717`)を spawn(825 行)と start(1007 行)が使う。start は HELLO と leader の join がない(leader 自身の spawn)ので step の並びが短い。`spawnDeadlineHint`(`spawn.go:2488`)は、step のすぐあとに読んだ spawn の ctx が `DeadlineExceeded` のときだけ付く。`per-step` はコードに残っていない(`internal/cli/org_spawn_timeout_test.go` が無いことを見る)。テスト: `TestOrgSpawnAndStart_TimeoutMSHelp_DescribesTheWholeSpawnDeadline`、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`(本物の `ExecRunner` で stdout の `spawn failed:` に出る)、`TestOrgSpawn_SpawnDeadlineRunsOut_ErrorHintsAtTimeoutMS`、`TestOrgSpawn_FailureBeforeSpawnDeadline_NoTimeoutMSHint`、`TestOrgSpawn_HelloFailsInsideDeadline_LeavePastIt_NoTimeoutMSHint`。skill の 4 つの写しは `cmp` で同じ。spawn と start の欄と「無期限待機は避ける」の項が、期限を spawn 全体のものと書き、10 秒の close、`spawn_close_failed`、ロックの断り、label が違うときの扱い、途中の座席への拒否を書く |
| AC6: 分離した herdr での (a)〜(e) | 満たす | `docs/evidence/seat-spawn-close-2026-10-22.md` の (a)〜(f) を `s4.log` と `manifest.jsonl` と突き合わせた。(a) は `spawn_failed`(04:35:46)の Details `pane=closed`、エラーの案内、`agent_not_found`。(b) は 6 秒で `spawned`。(c) は status の 1 行。(e) は `spawn_close_failed`(04:36:24)と status の `spawn_step (active)`(ROLE・DRIVER・MODEL が空)。(e) の 1 の拒否は台帳の行数 51 のまま。(d) は stop、`stop --all`、disband の rc 0。binary は abb11e8c で、そのあとの dde26a8f・70b23052 は stop の案内に `--state-dir` を足すのを `--state-dir` を渡したときだけにしたので、`--state-dir` を渡していない `run-s4.sh` のコマンドに対する文は HEAD でも同じになる(コードから読んだ。打ち直していない)。evidence の (d) と (e) を引く記録の 4 か所に、実機が示していないことを書いた文がある(V1) |
| AC7: tech-debt、根拠の表、仕様の印・注・Status・PR 一覧 | 満たす(記録の文に LOW 2 件) | 下の「記録の確かめ」 |
| AC8: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0、go1.25.8 のテスト | orchestrator の run-verify の結果待ち | 指示どおり、run-verify と Go のテストは打っていない。静的な部分は下の Static analysis で rc 0。`GOTOOLCHAIN=go1.25.8` の `go build ./...` と `go vet ./internal/org/... ./internal/cli/...` も rc 0。結果は orchestrator が足す |
| AC9: 座席ごとのロック。重ねた spawn は何も書かずに断る。違う座席と org と dry run は待たない | 満たす。実行は `/test` | 612〜621 行が、dry run でない spawn で `Manifest.Read`(623 行)の前に `acquireHeldFileLock(…, spawnLockName(…), "spawn", 0)` を取り、`defer` で返るまで持つ。`errFileLockHeld` は `SpawnOutcomeRejected` で、herdr も agmsg も打たない。`spawnLockName`(`internal/org/lockfile.go:29`)は `spawn-<org_id>_<seat_id>.lock`。ロックを取るのはこの 1 か所だけで、manifest のロックとの順は 1 通り。テスト: `TestOrgSpawn_SeatLock_RefusesASecondSpawnOfTheSeat`(join で止めた 1 つ目の間に 2 つ目が断られ、event・receipt・driver の呼び出しが増えず、1 つ目の pane に C-c も close も届かず、1 つ目は `spawned`、返ったあとの再試行は冪等)、`TestOrgSpawn_SeatLock_OtherSeatsOrgsAndDryRunsDoNotWait`、`TestOrgSpawn_SeatLock_LockThatCannotBeTaken_FailsWithNothingWritten`(`internal/org/spawn_lock_test.go`)。実機は evidence の (f) |
| AC10: 途中の座席への再試行が検査で断られるとき `rejected` も receipt も書かず、stop・`stop --all`・正しい再試行が拾う | 満たす。実行は `/test` | `rejectUnlessInFlight`(2169 行)を、エンベロープ(810 行)、permission(824 行)、`--scope`(864 行)の 3 つの拒否が使う。`spawnPrecheckErr`(1327 行)も同じ文を返し、`feature.go:213` が `InboxHintStateDir` を渡す。テスト: `TestOrgSpawn_InFlightSeat_RefusedRetryWritesNothing`(2 つの残り方と 3 つの検査の 6 通り)、`TestOrgSpawn_SeatLeftOpen_AfterRefusedRetry`(1 座席の stop、`stop --all`、正しい再試行の 3 つの続き。閉じるのは記録した pane だけ)、`TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn`、`TestStartFeature_PrecheckStopHintRepeatsInboxHintStateDir`、`TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir`(`internal/cli/org_test.go:2462`) |

### 記録の確かめ(AC7)

- 印の数: 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` は 32 項目で `[x]` 26、`[ ]` 6(base 5850f89b は 22 と 10)。根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` は 31 行で 25 と 6(base は 21 と 10)。変わった 4 項目はどちらも FR-2、FR-4、受け入れ条件 2、受け入れ条件 5 で、残る `[ ]` 6 項目の名前も両方で同じ。表の数の行(79 行の「`[x]` が 25、`[ ]` が 6」)も表と合う。FR-3 と NFR 安全の不変条件は `[ ]` のままで、F4 の部分の注は直してある
- PR の一覧: 仕様の Status と根拠の表の「段ごとの PR」に出る番号は、どちらも同じ 31 個(`diff` で一致)で、#248 を含む。#248 が merge 済みであることは base 5850f89b の「Merge pull request #248」で確かめた。`gh` は打っていない。この PR は「F4 の直し: ブランチ `fix/seat-spawn-close`」として足され、31 本に数えていない。#248 の計画と 4 つの report は HEAD にある
- `fix/seat-agmsg-self-name` は、仕様で 2 個から 0 個、根拠の表で 6 個から 0 個、tech-debt で 3 個から 0 個になった。根拠の表の前文は「4a が F3、4b が F4、4c が F11」に直り、merge した PR の並びに「F3 の #248」が入った。F11 の行の「直す PR」は「座席の環境 4c」
- tech-debt の F4 の行(`docs/tech-debt/README.md:320`): 取り消し線と RESOLVED のコメント(319 行)がある。「the agmsg self-rename of the row above」は、F3 の行の名前「agmsg 1.5.1 renames the herdr agents and panes that ralph addresses」と「fixed in #248」に直っている。原因の時刻は試行の manifest(`docs/evidence/org-trial-2026-10-20/manifest.jsonl`)と一致した: pane-5 は 03:41:00 に始まり、座席の join の戻りが 03:41:53(53 秒)、`spawn_failed` が 03:42:06(66 秒)。pane-6 の HELLO は 03:42:44 から 03:42:51(7 秒)。pane-9 は 03:43:51 に始まり、座席の join の戻りが 03:45:44(113 秒)。値が記録にないことと推定であることは文に書いてある
- 新しい tech-debt の行(338〜341 行): 340 行が挙げる `return` は `spawn.go` の 978、1081、1104(`ensureLeaderJoined` の失敗)、1120、1149、1172 行にある。341 行のファイルの長さ(2656 行と 5786 行、base は 2357 行と 4688 行、`spawn_lock_test.go` は 198 行)は `wc -l` と合う。338 行の試行の時間は合わない(V2)。org-stop-all の行(173 行)の (b) は、Debt・Impact・Why deferred・Trigger の 4 欄で取り消してあり、`compensatePaneCtx` はコードに残っていない
- 引用: 足した記録の行が挙げるテストの関数名 63 個は、どれも `func <名前>(` が `internal/` にちょうど 1 つある。そのうち「`<ファイル>` の `TestX`」の形で書いた 62 か所は、どれもそのファイルにある
- 試行の所見の前文の数: 仕様の Status の「6 項目は `[ ]` のまま」と所見の一覧(F7、F13、「leader への ALERT」)は、残る 6 項目の注(FR-3 は ALERT、FR-5 は F7、FR-6 と NFR 安全の不変条件は F13、FR-10 と NFR ツール非依存は codex の実機)と合う

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(HEAD 37aae73a) | rc 0、`All verifiers passed.` | changed scope で、選ばれた pack は golang。shellcheck、hook の `sh -n`、settings.json の jq、Codex の 3 つの guard、check-sync(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の計画の参照、`gofmt: ok`、go vet、golangci-lint `0 issues.`、staticcheck(出力なし)、secret scan(`5850f89b..37aae73a`、clean)。orchestrator の run-verify が同じ worktree で走っている間に打ったが、ログは別の名前(`verify-2026-10-22-060129.log`)に書かれた |
| `GOTOOLCHAIN=go1.25.8 GOPROXY=off go build ./...` | rc 0 | go.mod は `go 1.25.8`、手元の go は go1.26.0 |
| `GOTOOLCHAIN=go1.25.8 GOPROXY=off go vet ./internal/org/... ./internal/cli/...` | rc 0 | vet はテストファイルもコンパイルする |
| `git diff --check 5850f89b...HEAD` | rc 0 | |
| 足した 2535 行の走査(ホームのパス、U+FFFD、16 桁以上の 16 進、token の形、herdr の pane の id、UUID) | 漏れなし | 当たったのは、self-review の報告が grep の語として `/Users/` を挙げる 1 行だけ。evidence の文書は pane の id を `<pane>`、一時ディレクトリを `<D>` に置き換えている |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 つの写し | 合う | `cmp` で同じで、check-skill-sync も通る。spawn・start の欄と `--timeout-ms` の項はコードと合う |
| `--timeout-ms` の help とコード | 合う(V5 は注) | spawn の step の並び(workspace と tab の作成、agent の起動、agmsg の 2 つの join、HELLO)と start の並び(join 1 つ、HELLO なし)は `Spawn` の `leaderSelfSpawn`(1098 行)の分岐と合う。10 秒は `defaultDriverCallTimeout` と同じ値で、CLI は `DriverCallTimeout` を上書きしない |
| `docs/specs/2026-08-01-org-runtime.md` FR-9 | 合う | 足した文(C-c と close、1 回 10 秒、閉じたときだけ `spawn_failed`、`spawn_close_failed` で active、stop が拾う、座席ごとのロック、stale の label の確かめ)は、コードと合う |
| `docs/specs/2026-08-01-org-runtime.md` FR-1 と受け入れ条件 | ずれ(V3) | プール外のモデルの拒否は receipts に記録すると書くが、途中の座席への再試行は記録しなくなった |
| `Spawn` の doc コメント | 一部ずれ(V4) | dry run の予測が本番の記録と常に一致するという文 |
| tech-debt の行と code | 1 行の数字がずれ(V2) | stop と spawn の競合の行(338 行)、status の空の欄の行(339 行)、manifest の書き込みの失敗の行(340 行)、大きさの行(341 行)の記述はコードと合う。338 行の試行の時間だけが合わない |
| 新しい event の読み手 | 合う | `stateEvents` と `activeEvents` に入れていない(意図どおり)。`leaderActivityEventCount`(`internal/org/watch.go:1340`)の集合にも入れず、doc(1259 行から)に数えないことと理由を書いた。`orgsToDisband` は座席の state event で決まり、閉じられなかった座席の `spawn_step` が org を対象に入れる。`openOrgWorkspaces` は org の event だけを読む。`report.go` はどの event も同じように出す |
| `AGENTS.md` の repo map | 直さなくてよい | `internal/org/` の項は spawn の補償やロックの個々を書いていない |
| `.claude/rules/ralph/agent-messaging.md`、README、`docs/recipes/`、`internal/org/prompts/` | 合う | spawn の補償や `--timeout-ms` の意味を書く文はない。`docs/recipes/codex-seat-permissions.md:22` の「`agent_start` の時間切れのあとで `spawn_failed`」は今も正しい |
| 計画の AC と Progress | 遅れている(V7) | 下の Findings |

## Observational checks

- 計画の digest を計算し直して `550d25183df2` を得た。`- Approved:` 行と一致する。承認のあとの計画の変更(57a124a4、4ebc315e)は Progress だけ
- S4 の生の出力: `s4.log` の (e) の status で、org `ssc` の seat-b の行は ROLE・DRIVER・MODEL の欄が空のまま `spawn_step (active)` と pane の id を出しており(`active 2/3`)、evidence の書き方と合う。(e) の 1 の前後の数は、どちらも manifest 51 行、receipt 0、`rejected` 0 だった。(f) の 2 つ目の spawn のエラーは `... the lock is held after 0s` で終わる(`acquireHeldFileLock` を待ち 0 で呼ぶため)。evidence はこれを写している。利用者には意味の薄い語だが、動作には関係しない
- herdr、agmsg、`gh`、osascript、network は使っていない。insight event は下の Coverage gaps

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | 4 か所の記録が、実機で「pane の close を断らせた座席は active のまま残って `stop --all` が止めた」と書くが、実機ではそうなっていない。S4 の manifest では、close を断らせた seat-b の pane(04:36:09 に作った pane)は、(e) の 3 の正しい再試行の `compensateStale` が 04:36:25 に閉じ(`superseded by respawn (... pane=closed)`)、(d) の `stop --all`(04:36:32)が止めたのはその再試行が新しく作った seat-b の pane だった。閉じられなかった座席を stop と `stop --all` が拾うことは、fake の herdr のテストだけが確かめている。evidence の文書そのものの (e) と (d) の書き方は正しい | `docs/specs/2026-10-07-org-multi-org-director.md:64`(FR-2 の注の「`stop --all` が止めた((a)、(e)、(d))」)、`:154`(受け入れ条件 2 の注の「実機((e)、(d))で確かめた」)、`docs/evidence/org-spec-evidence-2026-10-21.md:48`(FR-2 の行の「実機の (e)、(d) で `stop --all` が止めた」)、`docs/tech-debt/README.md:319`(「left the seat active until `stop --all` closed it」)。生の出力は scratchpad の `ssc4b-raw/manifest.jsonl` と `s4.log` の (e4) と (d) | 4 か所を「実機では、閉じられなかった座席の pane は次の正しい spawn の補償が閉じた((e))。stop と `stop --all` が拾うことはテストで確かめた」に直す。印の判定は変わらない(FR-2 の実機の記録は `herdr-pane-close-2026-10-07.md` と試行の Part 1 がある) |
| V2 | LOW | stop と spawn の競合の行が、spawn の時間を「試行の agmsg の step まで進んだ spawn で 44〜144 秒」と書くが、試行の manifest では 7〜144 秒だった(leader の spawn は 7、10、14 秒、座席の spawn は 27、29、44、66、144 秒) | `docs/tech-debt/README.md:338`(Impact の欄)。`docs/evidence/org-trial-2026-10-20/manifest.jsonl` の `spawn_started` から `spawned`・`spawn_failed` までの差。doctor-escape の implementer は 03:33:25〜03:33:52、reviewer は 03:33:52〜03:34:21 で、どちらも `agmsg_announced` を通っている | 「7〜144 秒(座席の spawn は 27〜144 秒)」に直すか、F4 の前後の report-escape の spawn(44〜144 秒)と範囲を書く |
| V3 | LOW | org runtime の仕様の FR-1 と受け入れ条件は、プール外のモデルの拒否を receipts に記録すると書く。D6 で、途中の座席への再試行の拒否(エンベロープ・permission・`--scope`)は `rejected` も receipt も書かなくなった。`/org` skill はこの例外を書いているが、仕様は書いていない | `docs/specs/2026-08-01-org-runtime.md:65`(「プール外モデル・座席上限超過・スコープ違反は動詞のバリデーションで拒否し、receipts に記録する」)、`:87`(「receipts に `honored=false` 相当の拒否記録が残る」)。`internal/org/spawn.go:2169`(`rejectUnlessInFlight`) | `/sync-docs` で FR-1 か FR-9 に「途中の座席(`spawn_started` か `spawn_step` が最後)への再試行の拒否は、pane を台帳に残すため何も書かない」と 1 文足す |
| V4 | INFO | `Spawn` の doc は、dry run が本番と同じ順に検査するので「`--dry-run` の予測した拒否は、同じ要求の本番が記録するものと常に一致する」と書く。途中の座席に対しては、dry run は今までどおり dry run の `rejected` と honored=false の receipt を書き、本番は何も書かず、エラーの文にも途中の座席の後半が付かない。最初の原因は同じ | `internal/org/spawn.go:562-563`、`:653-666`(dry run の `o.reject`)、`:2169`。skill の spawn の欄(`.claude/skills/org/SKILL.md:180`)の「途中で止まった座席への spawn が … 断られるときは、`rejected` も receipt も書かない」も dry run を分けていない | 直すなら、doc の文を「最初の原因は同じ。途中の座席に対しては本番は何も書かない」にする。動作の変更は要らない |
| V5 | INFO | spawn の期限の ctx は Phase 1 の前に作られ(677 行)、`compensateStale` の呼び出し(確かめの get 3 つと C-c と close、どれも 1 回 10 秒)はその ctx を使わないまま、期限の時間を使う。herdr が遅いと、stale の補償だけで期限が尽き、そのあとの `tab_create` が期限切れで失敗して、案内は step が時間を使ったように読める。base の `compensateStale` は spawn の ctx を使っていたので、補償そのものも期限で切れていた。help は「失敗した step のあとの片付けは別に 10 秒」とだけ書く | `internal/org/spawn.go:677`、`:897`、`:2558-2590`、`internal/cli/org.go:717-720`。base は `git show 5850f89b:internal/org/spawn.go` の `compensateStale(ctx, …)` | 普通の道では起きない(herdr が 1 回あたり数秒以上かかるときだけ)。直すなら、期限の ctx を stale の補償のあとに作るか、help と skill に「stale の補償も期限の時間に入る」と書く。どちらにするかは orchestrator が決める |
| V6 | INFO | 仕様の注の「新しい binary では `spawn_failed` の座席の pane は閉じてある」「`spawn_failed` の座席は動いていない」は、`failStepWithNote` の道では正しいが、`compensateStale` の label が違う場合は pane に触らずに `spawn_failed` を書く。元の pane が別の id で残っていれば座席は動きうる(計画の Risks が書いた場合で、herdr が状態を失って id を振り直したときだけ起きる) | `docs/specs/2026-10-07-org-multi-org-director.md:70`、`:141`、`internal/org/spawn.go:2572-2573` | 直すなら「label が違う stale の座席を除き」と添える |
| V7 | INFO | 計画の AC1〜AC10 の checkbox が `[ ]` のままで、Progress の「Review artifact created」も `[ ]` のまま(self-review の報告は 0a7ef72e でコミット済み) | `docs/plans/active/2026-10-22-seat-spawn-close.md:75-84`、`:143` | `/sync-docs` か `/pr` でほかの項目と一緒に直す |

CRITICAL、HIGH、MEDIUM はない。

## Coverage gaps

- AC8 の run-verify と、HEAD の Go のテスト(go1.26.0 と `GOTOOLCHAIN=go1.25.8`)は、orchestrator の run-verify と `/test` に残す。この verify では go test を打っていない
- 閉じられなかった座席を stop と `stop --all` が拾うことは、fake の herdr のテストだけが確かめている(V1)。本物の herdr で見るなら、S4 の (e) の 1 のあとに (e) の 3 を打たずに `ralph org stop --all` を打つのが最も小さい
- stale の座席の label が違う場合と、確かめの get が失敗する場合は、本物の herdr では見ていない(evidence の「確かめていないもの」のとおり)
- 本物の claude の座席、負荷の下での時間、古い binary が残した pane は確かめていない(計画の Non-goals と evidence のとおり)
- insight event: 指示が「この報告だけをコミットし、ほかの文書を変えない」だったので、`docs/insights/events/` には足していない。`./scripts/insights-append.sh --slug seat-spawn-close --flow standard --phase verify --verdict pass --cycle auto --source skill --events-dir <scratch>` で作った行は次のとおりで、足すかどうかは orchestrator が決める。`{"schema":1,"ts":"2026-10-22T06:14:51Z","slug":"seat-spawn-close","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Verdict

- Verdict: pass(CRITICAL、HIGH、MEDIUM はない。LOW 3 件(V1、V2、V3)はどれも記録と仕様の文のずれで、`/sync-docs` で直せる。INFO 4 件(V4〜V7)は任意。AC8 は orchestrator の run-verify の結果待ち)
- Verified: AC1〜AC5、AC9、AC10 のコードとテストの本文の対応(補償の 10 か所の呼び出し、10 秒の期限と spawn の ctx を使わないこと、`spawn_close_failed` が state でないこと、stale の補償の 5 つの分かれ、ロックの位置と順、途中の座席への 3 つの拒否と事前検査、stop の案内の `--state-dir`、help と案内の文)、AC6 の evidence と S4 の生の出力、AC7 の記録(印 26/6 と 25/6、1 項目ずつの一致、31 本の PR、`fix/seat-agmsg-self-name` の置き換え、前文の予定の順、F4 の行の時刻、新しい 4 行の参照、テスト名 63 個)、静的解析(run-static-verify の rc 0、go1.25.8 の build と vet、`git diff --check`、足した行の走査)、文書のずれ、計画の digest
- Partially verified: AC6 の (e) と (d) を引く記録の文(V1)
- Not verified: AC8(orchestrator の run-verify の結果待ち)、Go のテストの実行(`/test`)、閉じられなかった座席を本物の herdr で stop が拾うこと、本物の claude の座席

## 直したもの(cycle 1、報告のあと)

コードのコメントとテストの直しは 0a032b37、記録の直しはこの節を足したコミット(`docs: correct the seat-spawn-close records after verify and test`)である。上の判定と重大度は変えていない。

- V1: 直した(記録)。仕様の FR-2 と受け入れ条件 2 の注、根拠の表の FR-2 の行と F4 の直しの項(107 行)、tech-debt の F4 の RESOLVED のコメントを、実機では閉じられなかった座席の pane を同じ座席の次の正しい spawn の補償が閉じたこと((e))、`stop` と `stop --all` が閉じることはテストで確かめたことに分けて書いた。evidence の (d) は、`stop --all` が止めたのが (e) の 3 が立てた `seat-b` だと書いた。印の判定は変えていない
- V2: 直した(記録)。試行の manifest で、`spawn_started` から同じ spawn の `spawned` か `spawn_failed` までを測った。記録が agmsg の step(Details が `agmsg_` か `step=agmsg_` で始まる記録)に届いた spawn は 10 回のうち 8 回で、7〜144 秒だった。leader でない 5 回では 27〜144 秒だった。tech-debt の行に、この範囲と測り方を書いた
- V3: 直した(記録)。org runtime の仕様の FR-1 に、途中で止まった座席の打ち直しをエンベロープ・permission・`--scope` の検査で断るときは `rejected` も receipt も書かないこと(`--dry-run` は書く)を足した。受け入れ条件の 1 つ目からは FR-1 を引いた
- V4: 直した(0a032b37)。`Spawn` の doc に、途中の座席に対しては本番は何も書かずにエラーへ `inFlightSeatErr` の後半を足し、dry run は dry run の `rejected` と honored=false の receipt を書く、と違いを書いた。コメントだけ
- V5: 直した(0a032b37)。spawn の期限の ctx を作る行に、stale の補償の時間も `--timeout-ms` から出ること(呼び出しはそれぞれ 10 秒で区切られ、期限では切れない)を書いた。コメントだけで、動作、help、skill は変えていない
- V6: 直した(記録)。仕様の FR-3 と NFR の注に、記録した pane の id が今は別の label の pane を指すときは `compensateStale` が触らずに `spawn_failed` を書く例外を足した。根拠の表の NFR 安全の不変条件の行(58 行)にも同じ文があったので、同じ例外を足した
- V7: 残した。計画の checkbox は orchestrator が付ける
- ほか: 0a032b37 でファイルの長さが変わったので、tech-debt の大きさの行の数を直した(`spawn.go` 2665 行、`spawn_test.go` 5859 行、新しいテストの塊は約 1100 行)
