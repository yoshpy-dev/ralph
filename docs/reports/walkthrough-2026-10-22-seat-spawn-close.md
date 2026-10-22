# Walkthrough: seat-spawn-close(座席の環境 4b: F4、失敗した spawn が pane を閉じる)

- Date: 2026-10-22
- Plan: docs/plans/active/2026-10-22-seat-spawn-close.md(/pr で archive に移す)
- PR: #249
- Branch: fix/seat-spawn-close(base 5850f89b、この walkthrough の前の HEAD 4cb15162。19 commits、32 files、+3034/-270)
- 読む範囲: コードとテストは 14 files(+2219/-214)。Go の本体は `internal/org/spawn.go`(+422/-114)が中心で、`verbs.go`(+38/-13)、`lockfile.go`、`seat.go`、`watch.go`(doc だけ)、`feature.go`(1 行)、`internal/cli/org.go` を足して +518/-137。残りはテストで、`spawn_test.go` が +1216、`spawn_lock_test.go` が新規 198 行。文書は skill 4 つの写しと quality-gates 2 つを含む 11 files。残りは計画、後工程の報告 5 本(cross-review は triage)、insight event
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → 失敗した spawn の補償(S1) → 座席のロックと stale の確かめ(S2) → 期限の説明(S3) → 実機(S4) → 記録(S5) → 後工程と、その中の直し → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行の所見を直す系列の 4 本目の後半(4b)。4a は #248(F3、agmsg の名前の書き換え)。F11 は 4c に送った
- F4: 試行(stage 9)で、report-escape の implementer の spawn が `agmsg_announce` で時間切れになった。補償は pane に C-c を送るだけで閉じなかったので、herdr の agent の名前 `<org_id>_<seat_id>` が残り、次の 2 回の spawn は `agent_start` で `agent_name_taken` になった。4 回目の座席は動いて TASK を処理したのに、台帳と `ralph status` は `spawn_failed` のままだった
- 時間切れの原因: spawn は herdr と agmsg のすべての step で 1 つの期限(`--timeout-ms`、既定 60000)を分け合う。試行の manifest の時刻では、座席の join が 53 秒目に戻り、HELLO に約 7 秒しか残らなかった。`--timeout-ms` の help は「per-step」と書いていて、実装と食い違っていた。agmsg がなぜ遅かったかは分かっていない
- 直し: pane を作ったあとの step で失敗したら、その pane に C-c と close を打つ。閉じられなければ新しい event `spawn_close_failed` を書き、座席を active のまま残して `ralph org stop` で片付ける。あわせて、座席ごとの spawn のロック、途中の座席への打ち直しの拒否が何も書かないこと、stale の補償の label の確かめ、`--timeout-ms` の説明を直した
- 戻し方: この PR を revert する。`spawn_close_failed` を書いた台帳は古い binary でも読める(state でない event は無視される)

## 2. 読む順(commit ごと)

1. b4b023a1 計画
   - Objective、Scope、Non-goals、Design decisions(D1〜D6)、AC1〜AC10 を読む。承認の digest は `550d25183df2` で、後工程のあいだ本文は変えていない(Progress と AC の checkbox だけが動く)
   - D1: 閉じられなかった pane は、`spawn_failed` を書いて警告だけ出す案にせず、座席を active のまま残す。仕様 FR-2 が stop に決めた `stop_failed` と同じ形で、記録には新しい state でない event `spawn_close_failed` を使う
   - D2: 期限は spawn 全体で 1 つのまま、help とエラーの案内を直す。step ごとの期限にすると spawn 全体の上限がなくなる(6 step で最長 6 倍)。既定を上げる案は、遅さの原因が分からないので採らない
   - D3: stale の補償は、stop と同じく label を確かめてから閉じる。label が違うときは、その pane に触らず `spawn_failed` を書いて進む。stop が同じ場合に `--force` を求めるのは「止まった」と記録する責任があるからで、spawn はその責任を負わない。consult の判定で、断る案からこの形に変えた
   - D4: HELLO の失敗は Leave、C-c、close の順で、3 つとも 10 秒ずつ
   - D5・D6: 計画の Codex 助言の HIGH 2 件を、コードで確かめて足した。動いている spawn の途中の記録を stale と取り違えて閉じること(D5、座席ごとのロック)と、途中の座席への不正な再試行が `rejected` を書いて、閉じられなかった pane を台帳から隠すこと(D6)
2. 80780740 S1 失敗した spawn が pane を閉じる(AC1〜AC3)
   - `internal/org/spawn.go:2644` の `closeSpawnPane`。C-c、`herdr pane close` の順に、1 回ごとに `callWithTimeout`(`internal/org/verbs.go:786`)で `defaultDriverCallTimeout`(10 秒、`verbs.go:758`)の期限を付けて打つ。spawn の期限(`spawn.go:686`)は使わない。時間切れで落ちた spawn の ctx は、もう切れているため。pane の id はその spawn の `TabCreate` が返したもので、label は確かめない。not found は閉じ済みとして `pane=already closed`、id が空なら `no pane to compensate`
   - `spawn.go:2463` の `failStepWithNote`。閉じたら `spawn_failed`(Details は `compensation=ctrl_c=ok pane=closed`)。閉じられなければ、`spawn_failed` の代わりに `spawn_close_failed`(`internal/org/seat.go:60`)を書く。この event は `stateEvents` にも `activeEvents` にも入れていないので、座席は pane の id を持つ最後の `spawn_step` のまま active になる。返すエラーには `herdr pane close <id>` と `stopSeatCommand`(`spawn.go:2204`)を入れる。`failStep`(`spawn.go:2403`)は note を付けない呼び出し
   - pane を渡す呼び出しは `spawn.go` の 1008、1056、1059、1073、1119、1152 行。`workspace_create`(976)と `tab_create`(981)は pane がまだないので `no pane to compensate`
   - HELLO の失敗(`spawn.go:1150`〜1152)は、spawn の ctx の `Err()` を Leave の前に読む。Leave は `compensateLeave`(`spawn.go:2511`)で、これも 10 秒の期限で打つ。順は Leave、C-c、close
   - stop 側は変えていない。stop と `stop --all` は、active の座席を `spawn_step` に記録された pane の id で閉じて `stopped` を書く。閉じられなかった座席がその道に乗ることは、`internal/org/verbs_test.go:2113` と `internal/org/verbs_all_test.go:265` が固める
   - 見ること: `spawn_close_failed` は `watch.go:1329` の `leaderActivityEventCount` の集合に入れていない。spawn が書く event で数えるのは `spawn_started` だけで、同じ spawn を 1 回数える。理由は doc(`watch.go:1259`〜1270)にある(self-review L2。dde26a8f で書いた)
3. 97782369 S2 座席のロックと stale の確かめ(AC4、AC9、AC10)
   - ロック: `internal/org/lockfile.go:29` の `spawnLockName` が `spawn-<org_id>_<seat_id>.lock` を返す。識別子に `_` は使えないので、座席ごとに 1 つに決まる。`spawn.go:617`〜626 で、dry run でない spawn が台帳を読む前に `acquireHeldFileLock`(`lockfile.go:165`)を待ち 0 で 1 回だけ試す。ほかの spawn が持っていれば、何も書かず herdr も agmsg も打たずに `SpawnOutcomeRejected` で断る。取れたら `defer` で返るまで持つ。flock はプロセスが終われば OS が外すので、ロックが取れたときの途中の記録は、終わった spawn のものと分かる。manifest のロックとの順は、座席のロックが先の 1 通り
   - 途中の座席への拒否: `spawn.go:2178` の `rejectUnlessInFlight`。エンベロープ(819)、permission(833)、`--scope`(873)の 3 つの拒否が通る。最後の state event が `spawn_started` か `spawn_step`(`spawnInFlight`、2210)の座席には、`rejected` も receipt も書かず、`inFlightSeatErr`(2190)の文で断る。`rejected` は pane の id を持たない state event で、書くと座席が inactive になり、閉じられなかった pane が stop にも次の spawn にも見えなくなるため。dry run は今までどおり記録する(`Spawn` の doc、`spawn.go:562`〜568)
   - `ralph org start --plan` の事前検査 `spawnPrecheckErr`(`spawn.go:1336`)も、`refuse`(1346)で同じ文を返す。`feature.go:213` が `o.InboxHintStateDir` を渡す
   - stale の補償: `Spawn` は stale を `spawn.go:878` で見つけ、台帳のロックの外で `spawn.go:906` が `compensateStale`(2567)を呼ぶ。`compensateStale` は、記録の pane が打った本人の pane(`HERDR_PANE_ID`)なら何も書かずに断る(計画にない追加。閉じると打ったセッションが終わるため)。そうでなければ `confirmSeatPane`(`verbs.go:906`)で tab と workspace の label を確かめる。not found なら `pane=already closed` で進む。label が違えば(`errLabelMismatch`、`verbs.go:880`)、C-c も close も打たず `spawn_failed` を書いて進む。確かめられなければ(get の失敗・時間切れ)断る。確かめられたら `closeSpawnPane` を打ち、close が失敗したら断る。断る道は `refuseStaleRespawn`(2606)で、`spawn_close_failed`(Details は `respawn refused (...)`)だけを書き、`spawn_started` も `rejected` も書かずに `SpawnOutcomeFailed` を返す
   - `verbs.go` の `labelMismatchError`(885)は、`confirmSeatPane` と `confirmOrgWorkspace`(954)の label が違う誤りに型を付ける。文は変えず、`errors.Is` で見分けるだけで、ほかの呼び出し(`stopSeatPane` など)は `err != nil` しか見ない
   - 見ること: stale の補償は spawn の期限(`spawn.go:686`)が始まったあとに走るので、その時間は `--timeout-ms` から出る。各呼び出しは 10 秒の上限で区切られ、spawn の期限では切れない(`spawn.go:682`〜685 のコメント)。verify の V5 で記録した
4. abb11e8c S3 期限の説明(AC5)
   - `internal/cli/org.go:717` の `orgSpawnTimeoutFlagUsage`。spawn(825 行)と start(1007 行)の `--timeout-ms` が、spawn 全体の期限を herdr と agmsg の step が分け合うこと、失敗した step のあとの片付けは 1 回 10 秒で別に区切られることを書く。start は HELLO がないので、step の並びが短い
   - `spawn.go:2497` の `spawnDeadlineHint`。期限切れが原因の失敗のエラーに、期限の値と「`--timeout-ms` を大きくして打ち直す」を、step の原因のすぐあとに付ける。Details には付けない。付けるかどうかは、step が返った直後に呼び出し側が読んだ spawn の ctx の `Err()`(`failStepWithNote` の `deadlineErr`)で決める。`errors.Is(cause, DeadlineExceeded)` では決めない。agent start の再試行を期限が打ち切ると、原因は `agent_pane_busy` の誤りで、期限を包まないため
   - `/org` skill の spawn と start の欄と、「無期限待機は避ける」の項(`.claude/skills/org/SKILL.md:1275` 付近)。写しは `.agents/skills/org/`、`templates/base/.claude/skills/org/`、`templates/base/.agents/skills/org/` の 3 つで、4 つともバイト一致
   - 見ること: HELLO の道が `deadlineErr := ctx.Err()` を Leave の前に置く(`spawn.go:1150`)のは、Leave が期限を越えて眠ったあとに読むと、期限の内側で失敗した HELLO にも案内が付くため
5. 57a124a4 S4 実機の確かめ(AC6)
   - `docs/evidence/seat-spawn-close-2026-10-22.md`。分離した herdr 0.7.5 の server、agmsg 1.5.1 の写し(HELLO の send だけを期限より長く止める stub)、偽の `claude`(入力欄に似た枠を出して stdin を読むだけの binary)で打った。ユーザーの herdr server と Orca には触れていない。binary は abb11e8c
   - (a) HELLO の時間切れのあと、`spawn_failed` の Details が `pane=closed` で、`herdr pane list` に座席の pane がなく、`herdr agent get` が `agent_not_found` を返した。(b) 同じ座席の次の spawn が 6 秒で `spawned` になった。(c) status に active で 1 行。(d) stop、`stop --all`、disband。(e) pane の close を断らせた座席が active のまま残った。(f) 同じ座席の spawn を重ねると、あとのほうがロックで断られ、台帳は 1 つ目の分だけだった
   - (e) で見えたこと: `ralph status` で、その座席の ROLE・DRIVER・MODEL が空になる。tech-debt に新しい行として記録した
   - 見ること: (e) の座席は、実機では次の正しい spawn の補償が閉じた。(d) の `stop --all` が止めたのは、その再試行が立てた座席。閉じられなかった座席を stop と `stop --all` が拾うことは、fake の herdr のテストだけが確かめている(verify の V1。記録 4 か所の言い過ぎを直した)
6. a7133901、4ebc315e S5 記録(AC7)
   - `docs/tech-debt/README.md`: F4 の行(319〜320)を RESOLVED にし、原因を試行の manifest の時刻で書いた(推定は推定と書く)。org-stop-all の行(173)の (b) は、S2 の label の確かめで直ったので RESOLVED。新しい行を 3 つ足した(動いている spawn の途中の stop、`spawn_close_failed` の座席の status の空の欄、manifest の書き込みが失敗した道が pane を閉じないこと)。4 つ目の `spawn.go` と `spawn_test.go` の大きさの行は、self-review の L7 で 3edc8590 に足した。F4 の行にあった「the row above」は、F3 の行の名前と「#248 で直した」に直した
   - `docs/specs/2026-10-07-org-multi-org-director.md` と `docs/evidence/org-spec-evidence-2026-10-21.md`: FR-2、FR-4 と、その受け入れ条件 2、5 を `[x]` にした。印の数は仕様が 26/6、根拠の表が 25/6。FR-3 と NFR の安全の不変条件は `[ ]` のままで、F4 の部分の注を直した。「fix/seat-agmsg-self-name」を「#248」に直し、PR 一覧を 31 本にした
   - `docs/specs/2026-08-01-org-runtime.md`: FR-9 に補償、`spawn_close_failed`、ロック、label の確かめを足した。FR-1 の例外は verify の V3 で足した(3b07848d)
   - 4ebc315e は計画の S5 の checkbox
7. 後工程と、その中の直し
   - 0a7ef72e self-review(マージ可、LOW 10)。dde26a8f と 70b23052 で L1、L2、L8、L9 を直した。L1 は復旧の案内 4 か所を `stopSeatCommand` 1 つにまとめ、`Org.InboxHintStateDir` があれば `--state-dir` を足す。CLI は `newOrgSpawnRuntimeAt`(`internal/cli/org.go:149`、161 行)が `orgInboxHintStateDir`(`internal/cli/org_inbox.go:335`)の値を入れ、`start --plan` の事前検査にも渡す。L2 は `leaderActivityEventCount` の doc。L8 は 3 本のテストの `TimeoutMS` を 50 から 500 ms に。L9 は `checkRetryRefusedThenAppendLegacyRejected`(`internal/org/verbs_test.go:4582`)の改名。3edc8590 と 37aae73a が L5〜L7 と記録を直した。L3(`failStepWithNote` の引数の数)、L4(stale の補償の `ctrl_c=skipped: pane not confirmed` という語を `stopSeatPane` と同じに保つ)、L10(10 秒の写し)は残した
   - a0ef094d verify(pass、LOW 3、INFO 4)。V1〜V6 を 0a032b37(doc のコメント)と 3b07848d(記録)で直した。V7(計画の checkbox)は 628369b9
   - 180d3683 test(pass、mutant 47 本中 46 本が赤)。0a032b37 で G1(ロックの断りが待たないこと。期待の文に `the lock is held after 0s` を足した)と G2(`spawn_close_failed` の追記が失敗したときの `errors.Join`。テスト 2 本)を直した
   - 628369b9 AC の checkbox と verify・test の insight event
   - 2bced00b sync-docs。直したのは `docs/quality/quality-gates.md` とその template の写しの Envelope validation の行 1 つで、途中の座席の打ち直しが `rejected` を書かない例外を足した
   - 4cb15162 cross-review の triage(指摘 0)

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | S1。`TestOrgSpawn_FailureAfterPane_SendsCtrlCThenClosesPane`(`internal/org/spawn_test.go:1542`、4 つの step)、`TestOrgSpawn_FailureAfterSpawnDeadline_ClosesPaneUnderFreshDeadlines`(1612、切れた spawn の期限のあとでも C-c と close の ctx が生きていて期限を持つ)、`..._PaneNotFound_CountsAsClosed`(1712)、`..._CtrlCFails_CloseStillCloses`(1738)。cli は `TestOrgSpawn_FailureInjection_AgentStart_CompensatesPane`(`internal/cli/org_test.go:529`) |
| AC2 | S1。`TestOrgSpawn_HelloTimeout_LeavesThenClosesPane`(1641)、`TestOrgSpawn_HelloFailure_UnansweredLeave_TimesOut`(1685) |
| AC3 | S1。`TestOrgSpawn_FailureAfterPane_CloseFails_RecordsSpawnCloseFailed`(1768)、`..._CloseTimesOut_RecordsSpawnCloseFailed`(1838)、`..._CloseAndRecordFail_ErrorJoinsTheAppendError`(1871)、`TestOrgStop_SeatLeftActiveBySpawnCloseFailure_ClosesRecordedPane`(`internal/org/verbs_test.go:2113`)、`TestOrgStopAll_StopsSeatLeftActiveBySpawnCloseFailure`(`internal/org/verbs_all_test.go:265`) |
| AC4 | S2。`TestOrgSpawn_StaleSeat_ConfirmsPaneBeforeClosing`(2107、5 通り)、`..._NoPaneRecorded_GoesOnWithNoCheck`(2159)、`..._PaneNotClosed_RefusesTheSpawn`(2190、6 通り)、`..._PaneNotClosed_RecordFails_ErrorJoinsTheAppendError`(2268)、`..._CallersOwnPane_RefusedWithNothingWritten`(2301) |
| AC5 | S3。`TestOrgSpawnAndStart_TimeoutMSHelp_DescribesTheWholeSpawnDeadline`(`internal/cli/org_spawn_timeout_test.go:17`)、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`(同 51、本物の `ExecRunner` で stdout の `spawn failed:` に出る)、`TestOrgSpawn_SpawnDeadlineRunsOut_ErrorHintsAtTimeoutMS`(`internal/org/spawn_test.go:1913`)、`..._FailureBeforeSpawnDeadline_NoTimeoutMSHint`(1984)、`..._HelloFailsInsideDeadline_LeavePastIt_NoTimeoutMSHint`(2026)。skill の文はテストがなく、verify が 4 つの写しの `cmp` と本文の突き合わせで見た |
| AC6 | S4。verify が (a)〜(f) を生の出力と突き合わせた。(e) と (d) を引く記録の言い過ぎは V1 として直した |
| AC7 | S5。verify が印の数(26/6、25/6)、31 本の PR、「fix/seat-agmsg-self-name」の置き換え、F4 の行の時刻、新しい行の参照、記録が挙げるテスト名 63 個を数え直して合った。sync-docs が 26/6 と 31 本をもう一度数えた |
| AC8 | `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は 2bced00b で rc 0(753 秒、FAIL 0)。そのあとの 4cb15162 は triage の報告と insight event だけを足した。go1.25.8 では、tester が 37aae73a で org・cli・driver・protocol を通し(331 秒)、implementer が 0a032b37 で同じ範囲を通した(332 秒) |
| AC9 | S2。`TestOrgSpawn_SeatLock_RefusesASecondSpawnOfTheSeat`(`internal/org/spawn_lock_test.go:87`。1 つ目を座席の join で止めた間に 2 つ目が断られ、event・receipt・driver の呼び出しが増えず、1 つ目の pane に C-c も close も届かない。止めを外すと 1 つ目は `spawned`)、`..._OtherSeatsOrgsAndDryRunsDoNotWait`(141)、`..._LockThatCannotBeTaken_FailsWithNothingWritten`(180)。実機は S4 の (f) |
| AC10 | S2。`TestOrgSpawn_InFlightSeat_RefusedRetryWritesNothing`(`internal/org/spawn_test.go:2337`、6 通り)、`TestOrgSpawn_SeatLeftOpen_AfterRefusedRetry`(2493。stop、`stop --all`、正しい再試行の 3 つの続きで、閉じるのは記録した pane だけ)、`TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn`(2394)、`TestOrgSpawn_StopHints_RepeatInboxHintStateDir`(2430)、`TestStartFeature_PrecheckStopHintRepeatsInboxHintStateDir`(`internal/org/feature_test.go:685`)、cli は `TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir`(`internal/cli/org_test.go:2462`) |

テストの強さ(test の報告より):

- 新しいテストと期待を変えたテスト 48 本は、go1.26 と go1.25.8 の両方で `-count=3` と `-race` を通り、どれも 144/144 だった。時間に敏感な 13 本は、load average 26〜79 の下の `-count=10` で 130/130
- herdr・agmsg・osascript・gh などを、呼ばれたら記録して終わる stub にした PATH で 48 本を打ち、外への呼び出しは 0 回だった
- mutant は 47 本のうち 46 本が赤になった。生き残ったロックの取得を 2 秒待たせる 1 本(G1)は、0a032b37 で期待の文を足し、赤になることを確かめた
- 期待を変えたテスト(Details の `C-c sent` を新しい語に、stale の補償の呼び出しの列を確かめと close を含む形に、途中の座席への拒否を「何も書かない」に)は、消えた assert がすべて新しい期待で置き換わっていて、弱めていない

## 4. 確かめていないこと

- 本物の claude の座席での確かめ。S4 は偽の `claude` で、herdr の pane と agent の名前と台帳までを見た
- 試行のような負荷の下での時間。HELLO の遅れは stub で作った。agmsg が遅かった原因は分かっていない
- stale の座席の label が違う場合と、確かめ(herdr の get)が失敗する場合。fake の herdr を使うテストだけで確かめた
- 閉じられなかった座席を stop と `stop --all` が拾うこと。fake の herdr のテストだけで確かめ、実機では見ていない
- 実機の evidence は abb11e8c の binary で取った。レビューのあとの直し(案内の `--state-dir`、doc の文、テスト)は打ち直していない。evidence は `--state-dir` を使っていないので、引用したエラー文と Details は直しのあとも同じはず
- ロックを取らない古い binary の spawn との競合。更新の途中で起きうることをコードから読んだだけで、再現していない。stop と disband が座席のロックを見ないことと合わせて、tech-debt に行がある
- manifest の書き込みが失敗した道は、pane を閉じない。次の spawn の `compensateStale` が拾うが、`tab_created` の追記が失敗した場合は記録に pane の id がなく、拾えない。tech-debt に行がある
- 補償にかかる時間。失敗のあとの C-c と close は、1 回ずつ最大 10 秒かかる。前の spawn が途中で止まった座席の後始末は spawn の期限が始まったあとに走るので、その時間は `--timeout-ms` から出る
- label が違う場合に pane に触らず進むこと。herdr が id を振り直したときに、元の pane が別の id で動き続けることがありうる。そのときは次の `agent_start` が `agent_name_taken` で落ち、その spawn が自分の pane を閉じる
- leader の雛形(`internal/org/prompts/leader.md`)に、spawn の失敗と `spawn_close_failed` のときの手が書かれていない。sync-docs が挙げた前からの欠けで、この PR では書いていない。エラーの文が `herdr pane close` と `ralph org stop` を示すが、実際の leader が従うかは確かめていない
- HELLO が届いたあとで時間切れになると、leader の受信箱に、閉じた座席の HELLO が残る(S4 の (a) で確かめた。消さないと計画で決めた)
- 古い binary が `spawn_failed` のまま残した pane を、新しい binary が見つけて閉じること(計画の Non-goals)
- codex の座席の spawn の失敗
- 記録に残る「fix/seat-spawn-close」を、この PR の番号に置き換えること。次の PR でする。F11 は 4c
- Linux では、手元で走らせていない(CI が見る)
