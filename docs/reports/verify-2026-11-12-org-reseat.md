# Verify report: org-reseat(`ralph org reseat`、#312c の 1 本目)

- Date: 2026-11-13(JST)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(spec `docs/specs/2026-11-12-org-reseat.md`)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 3 回目(cycle 2 の cross-review のあと cap を 3 に上げた回。`cycle-count.json` は `cycle` 2、`cap` 3)。テストは走らせていない(`/test` の担当。同じ機械でほかのセッションがテストを回しているので、`go test` は 1 本も打っていない)
- この report は 2 回目(80f58f86)を置き換える。1 回目の本文は `git show af927216:docs/reports/verify-2026-11-12-org-reseat.md`、2 回目は `git show 80f58f86:docs/reports/verify-2026-11-12-org-reseat.md` で読める。tech-debt の register が引く前の回の ID(I5、I6、I8、I10、V14、V15)は、下の「前の回の drift と INFO」に残した。この回の drift は V17 から番号を続ける
- 3 回目の中の更新: 3 回目の最初の本文(bab7b33a)のあとに、help の直し b142944d、self-review の report 1326092b、origin/main(e902705c、#312a の PR #327 と PR #325)の merge 8460bf64 が入った。その 3 つを下の「bab7b33a のあとの 3 コミット」で読み、ほかの節は HEAD に合わせて直した。最初の本文は `git show bab7b33a:docs/reports/verify-2026-11-12-org-reseat.md` で読める
- Scope: `git diff e902705c...HEAD`(merge base e902705c、HEAD 8460bf64、35 ファイル)。merge の前の範囲(`git diff 4220ea7c...975e9f20`)と同じ 35 ファイルで、merge のあとに Go で変わったのは `internal/cli/org_reseat.go`(help、+8/-2)と `internal/cli/org_reseat_test.go`(+6)だけ。`internal/org/` の reseat の 3 ファイルと `feature.go` は bab7b33a と同じバイト列。main から入った Go のうち、`internal/org/` の変更(`watch*.go`、`usage_limit_screen.go`、`inbox.go`、`escalate.go`、`resume_screens.go`)は reseat のファイルから呼ばれない(grep)。`internal/cli/` で reseat が呼ぶのは `scaffold_version_notice.go` の `printScaffoldVersionNotice`(`runOrgReseat` から)だけで、main の変更はその中の `semverWithV` を同じ本体の `upgrade.SemverWithV`(`internal/upgrade/behavior_notes.go:183`)に置き換えた移動なので、reseat の出力は変わらない。判定の範囲は 1 本目の FR-1〜FR-8、FR-5a、FR-11、D1〜D6 と D2 の注、計画の AC-1〜AC-12 と AC-15〜AC-21。FR-9、FR-10、AC-13、AC-14 は 2 本目なので対象外
- Evidence: `docs/evidence/verify-2026-11-12-org-reseat.log`(git の対象外)。この更新の分は末尾の「3rd run, update after b142944d, 1326092b, 8460bf64」の節にある(`run-static-verify.sh` の全出力、HEAD から作った binary の `ralph org reseat --help`、merge の 3-way の行の数え合わせと、両側が変えた行の語単位の突き合わせ、skill の 8 つの写しの sha256、行数、plan の digest、`cycle-count.json`)。ラッパー自身の記録は `docs/evidence/verify-2026-11-13-052101.log`

## bab7b33a のあとの 3 コミット

| # | コミット | 判定 | 根拠 |
| --- | --- | --- | --- |
| 1 | b142944d: `ralph org reseat --help` に、起動のときの検出と、打ち直しから外す座席を足した(V19 への直し)。pin を 2 つ足した | V19 は直った。足した文に偽は無い | 足した文を 1 つずつコードに当てた。「That command leaves out a seat another command stopped or started meanwhile (see below), so when no other seat is left, none is printed.」: `reseatRetryCommand`(`internal/org/reseat.go:1325`)は `Outcome != started` かつ `SeatID != skip` の座席だけを並べ、`skip` は `finish` が渡す `res.Changed`。`Changed` を立てるのは `stopAll` の止める前の読み直しと、`startAll` の `--driver`・`--model` が渡っていない不一致の 2 か所だけ。並べる座席が無ければ `""` で、`printOrgReseatResult`(`internal/cli/org_reseat.go:409`)は `res.Retry != ""` のときだけ `to start the seats left, run:` を出す。「A seat it starts that another command started meanwhile on another role, driver, or model ends the reseat there too.」: `startAll`(`reseat.go:1170`)の idempotent の枝と `reseatIdempotentMismatchErr`(`reseat.go:1225`、Role・Driver・Model を比べる)。「Without --driver and --model it is left as it is and out of the command that starts the rest; with them, that command stops it and starts it on them.」: `!seat.Overridden` のときだけ `Changed` を立てる。`Overridden` は `reseatSeat`(`reseat.go:755`)の 1 か所で、`--driver`・`--model` を渡した 1 座席にだけ付き、`reseatRetryCommand` は残る 1 座席が `Overridden` なら `--driver`・`--model` を足す。その打ち直しが座席を止めて頼んだ設定で立てることは `TestReseat_ASeatAnotherCommandStartedOnOtherSettings` が打ち直しを走らせて固定する。冒頭の無条件の「leaves out」は、`--driver`・`--model` を渡した場合と並べると食い違って読めるが、「(see below)」が指す段落で分けているので偽ではない(self-review と同じ読み)。HEAD から作った binary の help に 2 つの文が出ることを見た。self-review の L20(段落の途中の 50 文字の行、`command stops it and starts it on them. Every stop`)と L21(「the reseat ends with one reseated event of the org, also when it ended early」は、最初の座席の止める前の読み直しで終わった reseat に当てはまらない。`finish` は `r.attempted` が true のときだけ `reseated` を書き、`stopAll` はその読み直しで `attempted` を立てる前に返る)は、コードを読んで同じ結論になった。どちらも LOW で、register に入る(V23) |
| 2 | 1326092b: self-review の report に b142944d の節を足した(Merge yes、LOW だけ。新しい L20、L21) | 記録だけ | report 1 ファイルの追記で、コードと文書の契約は変えない。L19、L20、L21 を register に足す件は V23 に入れた |
| 3 | 8460bf64: origin/main(e902705c)の merge。AGENTS.md、README.md、`docs/quality/quality-gates.md` とその `templates/base` の写し、`docs/specs/2026-08-01-org-runtime.md`、`docs/tech-debt/README.md` の衝突を、両側を残して解いた | 両側の文はすべて残った。reseat の文と #312a の文の食い違いが 1 つある(V24) | merge base 4220ea7c、両親 1326092b と e902705c、merge 8460bf64 で、6 ファイルの行を数え合わせた(各行の数が「両親の数の和 − base の数」と合うか)。quality-gates の 2 つの写しと org runtime の仕様は全行が合う。合わない行は、両側が同じ 1 行を変えた 5 行だけ(AGENTS.md の `internal/org/` の行、README の `ralph org` の表の行、tech-debt の 3 行)。その 5 行は語単位で、base に両側の変更をすべて当てると merge の行と一致する(tech-debt の 3 行は、同じ位置に両側が足した持ち越しの注と Related を、main の分を先にして並べた形)。落ちた語も、どちらの側にも無い語も無い。自動で merge された skill の 8 つの写しも、同じ数え合わせで全行が合い、組ごとに sha256 が一致する(org `c75fc6d937fd` 2,496 行、director `e5dcf7466095` 2,127 行)。`check-skill-sync.sh` は 14 skill が揃い、`check-sync.sh` は DRIFTED 0。plan と reseat の仕様は merge で変わっておらず、plan の digest は `f7e78c0dd96c` のまま |

### reseat の文と #312a の文の突き合わせ

- director skill(4 つの写し)の `usage_limited` の件の扱い(`.claude/skills/director/SKILL.md:1028` から)は、1046〜1047 行目で「座席を立て直す動詞は issue #312 で扱っていて、この版にはない。」と書く。同じファイルの 1430 行目に、reseat 側が足した「## 座席を立て直す」(`ralph org reseat`)がある。#312a の文は reseat より前に書かれ、merge で偽になった(V24)
- org runtime の仕様の「2026-11-13 改訂(feat/watch-usage-limited)」の (d)「座席を別のアカウントや driver で立て直す動詞は、この改訂の対象ではない(issue #312 の受け入れ条件 2、#312c)」は、その改訂の範囲を言う文なので偽ではない。すぐ上の「2026-11-12 改訂(docs/spec-org-reseat)」が動詞を足しているが、(d) からそこへの指し先は無い(I14)
- AGENTS.md の `internal/org/` の行は、`watch_usage_limit.go` の説明と `reseat.go` の説明が別の位置に入り、食い違う文は無い。README の `ralph org` の表の行と watch の段落、quality-gates の Watchdog pulse layer の行、`.claude/rules/ralph/agent-messaging.md` の `usage_limited` の段落も、reseat の文と食い違わない
- org skill の `usage_limited` の項(`.claude/skills/org/SKILL.md:649` から)は reseat に触れず、reseat の「座席の立て直し」(1623 行目)は「利用上限で止まった座席を別のアカウントで続ける」ときの動詞だと書く。食い違いは無い
- director skill の「座席を立て直す」の 1 は「自分の判断では打たない(座席が利用上限で止まっていても、人に伝えるだけにする)」と書き、`usage_limited` の件の「止まった座席とリセットの時刻を伝えて、どうするかを人と決める」と合う

## cycle 3 の直し 2 件(cross-review 2 回目の ACTION_REQUIRED 1 件)

| # | 直し | 判定 | 根拠 |
| --- | --- | --- | --- |
| 1 | def8d2a3: 立て直しの起動が idempotent で返り、返った座席の役割・driver・モデルが計画と違えば、その座席を `start_failed` にしてそこで止まる | 反映済み | 判定は `startAll`(`internal/org/reseat.go:1170`)の中で、`spawn.Err == nil` かつ `Outcome == idempotent` のときだけ走る。`reseatIdempotentMismatchErr`(`reseat.go:1225`)が Role・Driver・Model を比べ、モデルの `""` どうしは同じとみる。`Spawn` が idempotent を返すのは最新の state event が `spawned` の座席だけで、reseat が止めた座席(最新は `stopped`)も、対象に入るが active でない座席も当たらない。この枝に来るのは、reseat の確かめか停止のあとにほかのコマンドが立てた座席だけになる(self-review の読みと同じ)。止まったあと、後ろの座席は `stopped` のまま打ち直しに入り、前で立った座席は leader が対象でなければ DECISION が名指す。`reseated` も書かれる。同じ設定の idempotent は前のまま `started`。テスト: `TestReseat_ASeatAnotherCommandStartedOnOtherSettings`、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`、`TestReseat_ASeatAnotherCommandStartedOnTheSameSettings`(2 枝。変えていない動きの固定)、`TestReseatIdempotentMismatchErr`。割り込みは新しいフック `beforeStart`(`reseat.go:863`)で入れる |
| 2 | df8f674e: その座席に `--driver`・`--model` が渡っていなければ `ReseatResult.Changed` にして打ち直しから外す。関数名を `reseatIdempotentMismatchErr` に改めた | 反映済み | `startAll` は `!seat.Overridden` のときだけ `Changed` を立て、`reseatRetryCommand`(`reseat.go:1325`)が `skip` で外す。外す理由は doc のとおり成り立つ。`--driver`・`--model` なしの打ち直しは `recorded()`(`reseat.go:229`)、つまり台帳の最新の `spawn_started` か `spawned` の設定で立てるので、ほかのコマンドが立てた設定のまま座席を止めて立て直すだけになる。`Overridden` が立つのは `reseatSeat`(`reseat.go:755`)の 1 か所で、`checkReseatFlags`(`reseat.go:522`)が `--driver`・`--model` を `--seat` 1 つに限る。だから渡した reseat の打ち直しは、その座席を `--driver`・`--model` つきで名指し、止めて頼んだ設定で立てる。`Reseat`(`reseat.go:1069`)は `stopAll` が true を返したときだけ `startAll` を呼ぶので、止める側の `Changed`(FR-5a)と起動側の `Changed` が同じ reseat に立つことはない。`Err` の付かない古い名前は Go のソースに残っていない(grep)。テスト: `TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`(打ち直しを走らせ、何も止めず reviewer だけが立ち、ほかのコマンドの implementer が sonnet のまま `reseat=` なしで残る)、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`(`Changed` が leader、`Retry` と `Decision` は空)、`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`(`Changed` は空で、打ち直しを走らせると codex で立ち、DECISION を出す) |

2 つの直しのあとも、reseat が止めたまま残した座席は必ず打ち直しに入る。`Retry` から外れるのは `Changed` だけで、`Changed` は止めていない座席(止める側)か、ほかのコマンドが立てて動いている座席(起動側)のどちらかだからである。一方、起動側の `Changed` が対象の最後の座席なら(leader だけの reseat など)、打ち直しは出ない。仕様と文書のいくつかの文は、この場合に文字どおりには当てはまらない(V17、V20、V21。help の V19 は b142944d で直った)。

## cross-review 1 回目の 3 件の反映(cycle 2 の直し)

2 回目の report の判定のまま。cycle 3 の差分は下の関数の中身に触れていないので、行番号だけを HEAD に合わせた。

| # | 直し | 判定 | 根拠 |
| --- | --- | --- | --- |
| 1 | `--seat` が、最初の spawn が `spawned` に届かず reseat が止めた座席を断る | 反映済み | `reseatLifeSeats`(`reseat.go:240`)が、今の命でその座席のどれかの event の Details が `reseat=<id>` で終われば `touched` を立て、`startable()`(`reseat.go:204`、`spawned \|\| active() \|\| touched`)を `reseatTargets`(`reseat.go:599`)と `reseatNoActiveSeatErr`(`reseat.go:650`)の両方が呼ぶ。`TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`、`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes` |
| 2 | leader を立て直さない reseat が途中で失敗すると DECISION が消える | 反映済み | `finish`(`reseat.go:1241`)は、`plan.Start` に leader がいれば DECISION を作らず、いなければ `Outcome == started` の座席だけで作る。CLI は失敗のときも `Retry` のあとに出す(`printOrgReseatResult`、`internal/cli/org_reseat.go:409`)。`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、`TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` |
| 3 | `--config` が打ち直しのコマンドから落ちる | 反映済み | `runOrgReseat`(`org_reseat.go:196`)が `filepath.Abs` した `--config` を `HintConfigPath`(`reseat.go:86`)に入れ、`reseatVerbCommand`(`reseat.go:569`)が `ShellWord` で足す。打ち直しと断り 3 つのすべてがこの関数を通り、DECISION の `send` と CLI の `read` は通らない。上の CLI のテストが、相対の `--config` で、打ち直しを別のディレクトリから打って通す |

## Spec compliance

各 AC は、コードを読んで判定し、それを固定するテストの名前を添えた。テストの合否は `/test` が確かめる。2 回目からの変化がない行は、2 回目と同じ根拠で、行番号だけを HEAD に合わせた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1 3 席の停止と leader → implementer → reviewer の起動、`scope_reserved` が増えない、承認の検査が通る | 満たす | `Reseat`(`reseat.go:1069`)、止める順は `PlanReseat`(`reseat.go:371`)の `stops`。`TestReseat_StopsAndStartsEverySeatInOrder`。実機は leader と reviewer の 2 席で合格(AC-12)。立て直しのあいだにほかのコマンドが同じ設定で立てた座席は `started` に数える(前からの動き。I11) |
| AC-2 leader が新しい workspace の最初の pane | 満たす | `TestReseat_LeaderIsTheFirstPaneOfANewWorkspace`。実機でも root の pane |
| AC-3 `--seat implementer --driver codex --model` と DECISION の行 | 満たす | `reseatSeat`(`reseat.go:755`)、`finish`。`TestReseat_OneSeatOnAnotherDriver`、`TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`。立て直しのあいだにほかのコマンドが別の設定でその座席を立てると、`start_failed` で DECISION は出ず、打ち直しが `--driver`・`--model` つきで出る。それを走らせると codex で立ち、DECISION を出す(`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`) |
| AC-4 止める前の断り(計画の変更、自分の節か前文、計画のファイル、見張り、worktree の記録) | 満たす | `PlanReseat` の 5〜9、`reseatWorktreeRecordErr`(`reseat.go:773`)。`TestPlanReseat_RefusesWhatTheStartsWouldRefuse`、`TestReseat_ARefusalWritesNothing` |
| AC-5 結びつきの無い org の断りと stop・spawn の打ち方 | 満たす | `reseatNoBindingErr`(`reseat.go:581`)。`TestPlanReseat_RefusesAnOrgWithoutABinding`、`TestOrgReseat_OrgWithoutBindingRefused` |
| AC-6 active な座席が無いときの断りと `--seat` の打ち方 | 満たす(注) | `reseatNoActiveSeatErr`(`reseat.go:650`)。並べるのは `--seat` が受け付ける座席(`startable()`)だけ(2 回目の注のまま。仕様の D2 の注は 31f77c53 でこれに合わせた)。`TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes`、`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes` |
| AC-7 座席の pane から打った reseat の断り | 満たす | `reseatSelfPaneErr`(`reseat.go:548`)。`TestPlanReseat_RefusedFromASeatPane` |
| AC-8 pane が無い座席の立て直し | 満たす | `TestReseat_SeatsWhosePanesAreGone`。実機は `pane=closed` で合格 |
| AC-9 `--dry-run` の出力 | 満たす | `printOrgReseatDryRun`(`org_reseat.go:379`)。`TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` |
| AC-10 `stop_failed` で何も立てない | 満たす | `stopAll`(`reseat.go:1132`)。`TestReseat_AStopThatFailsStartsNothing` |
| AC-11 フラグの組み合わせの断り | 満たす | `checkReseatFlags`(`reseat.go:522`)はロックより前。`TestPlanReseat_RefusesBadFlags`、`TestReseat_AFlagErrorTakesNoLock`、`TestOrgReseat_FlagRefusals` |
| AC-12 実機で AC-1 と AC-8 を 1 回ずつ | 一部 | `docs/evidence/org-reseat-live-2026-11-12.md`。AC-8 は合格、AC-1 は 2 席で合格。implementer と承認の検査は実機で見ていない。binary は dd204370 で、切れた reseat の座席、SIGPIPE、cycle 2 の 3 つの直し、cycle 3 の 2 つの直しのどれよりも前 |
| AC-15 `reseat=<id>` の note と `reseated` 1 件 | 満たす | `noteDetails`(`internal/org/verbs.go:1314`)、`reseatedDetails`(`reseat.go:1302`)。`TestReseat_RecordsItsIDOnEveryStopAndStart`。起動側の不一致でも `reseated` に座席ごとの結果(`start_failed`)が入る(cycle 3 の test が `assertReseated` で固定する)。ほかのコマンドが同じ設定で立てた座席の `spawned` には `reseat=<id>` が無い(I11)。最初の座席の止める前の読み直しで終わり、何も止めず立てなかった reseat は `reseated` を書かない(`finish` の `if r.attempted`、`TestReseat_ASeatChangedBeforeItsStop` の「the first seat to stop was stopped meanwhile」が固定する)。AC の「途中で失敗したとき」は何かを止めるか立てようとしたあとの失敗と読み、満たすとした。この条件は org skill の表の行と org runtime の仕様の改訂 (c) が書くが、reseat の仕様の FR-8 は書かない(V17 (e))。help の文は self-review の L21 |
| AC-16 skill の 4 つの写しと help が動詞とアカウントの移し方を言う | 満たす | 動詞とアカウントの移し方は、org skill の表の行と「座席の立て直し」、director skill の「座席を立て直す」、help が言う。8 つの写しは組ごとに sha256 が一致(org `c75fc6d937fd`、director `e5dcf7466095`。8460bf64 の merge のあと)。help は b142944d から起動のときの検出と打ち直しから外す座席を言う(V19 は直った)。skill はまだ言わない(V21、V22)。director skill の `usage_limited` の件の扱いが「座席を立て直す動詞は…この版にはない」と書き、同じ skill の「座席を立て直す」と食い違う(V24。merge で入った) |
| AC-17 `reseat.lock` と止める直前の読み直し | 満たす | `takeReseatLock`(`reseat.go:996`)、`stopAll` の `reseatSeatNow`(`reseat.go:1269`)。止める側は cycle 3 の差分の外で、`Changed` の立て方と `orgReseatLeftLine` の `left as it is:` の行(`org_reseat.go:473`)は前のまま。`ReseatResult.Changed`(`reseat.go:931`)の doc は 2 つの場合を書くようになったが、止める側の説明は変わっていない。`TestReseat_RefusedWhileAnotherReseatHoldsTheLock`、`TestReseat_ASeatChangedBeforeItsStop` |
| AC-18 起動の失敗のあと、出た打ち直しで立つ | 満たす | `reseatRetryCommand`(`reseat.go:1325`)、`startable()`。cycle 3 で、起動の失敗に「ほかのコマンドが別の設定で座席を立てた」場合が加わった。`--driver`・`--model` を渡した reseat では、打ち直しがその座席を頼んだ設定で立てる。渡さなかった reseat では座席を打ち直しから外し、打ち直しは残りの座席を立てる(どちらの test も打ち直しを走らせて確かめる)。外した座席はほかのコマンドの設定で動いているので、AC の「座席が立ち直る」に反しない。`TestReseat_RetryAfterAFailedStart`、`TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`、`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`、`TestReseatRetryCommand`、`TestOrgReseat_AFailedStartPrintsTheRetry`、`TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` |
| AC-19 `--dry-run` は見張りも server も起動しない | 満たす | `reseatDryRunGates`(`reseat.go:815`)、`wireOrgReseatDryRunGates`(`org_reseat.go:329`)。`TestPlanReseat_DryRunStartsNoWatchOrServer` |
| AC-20 切れた reseat の座席と signal | 満たす | `reseatLifeSeat.cutOff`、`orgReseatHoldSignals`(`org_reseat.go:294`)。`TestPlanReseat_FinishesAReseatThatWasCutOff`、`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt` |
| AC-21 独自の役割の座席の断り | 満たす | `reseatCustomRoleErr`(`reseat.go:701`)。`TestPlanReseat_RefusesACustomRoleSeat` |

仕様の FR と D の対応:

- FR-1・D1: 1 回目のまま。`--timeout-ms` は仕様に無いが、org skill の表の行が書いている
- FR-2・D2・D4: 2 回目のまま。仕様の FR-2 と D2 の注は 31f77c53 でコードに合わせた(V9 を直した)。cycle 3 は `--seat` の受け付けを変えていない(`reseatLifeSeats`・`startable()`・`reseatTargets` は差分の外)
- FR-3: AC-7
- FR-4・D3: 1 回目のまま。cycle 3 は検査を変えていない(`PlanReseat` は差分の外)
- FR-5a・FR-5・FR-6・FR-7・FR-8・D5: AC-17、AC-10、AC-18、AC-3、AC-15、AC-9、AC-19。cycle 3 で、FR-6 の起動に「ほかのコマンドが別の設定で立てた座席を `start_failed` にしてそこで止まる」動きが加わった。FR-6 の「打ち直し方(`--seat <立たなかった座席>`)を出して」と非機能要件 1 の「打ち直し方を必ず出す」は、その座席に `--driver`・`--model` が渡っていない reseat では文字どおりには当てはまらない(V17)。コードの不適合とはしない。reseat が止めたまま残した座席は必ず打ち直しに入り、外れるのはほかのコマンドの設定で動いている座席だけで、その設定は stderr の error が言う。外す動きは、打ち直しが動いている座席を同じ設定で立て直すだけになる穴(self-review の L18)を塞ぐために選ばれたものである。D5 は cycle 3 のあとも成り立つ(leader が対象なら DECISION を作らず、不一致の前に立った非 leader の座席は DECISION が名指す)。FR-8 の `reseated` と note の扱いは I11、I12
- FR-11・D6: 1 回目のまま。help は b142944d で起動のときの検出を言うようになった。アカウントの移し方の文は変わっていない
- 非機能要件: 1 は上の V17。2〜4 は 2 回目のまま。cycle 3 の直しは pane の環境に何も足していない。`reseated` の形も変わらず、座席の結果に `start_failed` が入る場合が増えただけ
- 仕様の「依存」の #312a の行(139 行目)の「reseat はそれに依存しないが、director skill の案内で打ち方をつなぐ」は、merge のあと director skill の `usage_limited` の件の扱いが reseat を指さず、逆に「この版にはない」と書くので、まだ満たされていない(V24)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(exit 0) | 1 回で通った。タイムアウトは無かった。末尾は `==> All verifiers passed.` |
| ├ shellcheck、`sh -n`、guard の awk の構文、hook の JSON(`jq -e`)、Codex の hook の single-source と検出の smoke、PR provenance | pass | ブランチの diff に shell は無い |
| ├ `scripts/check-sync.sh` | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| ├ `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照 | pass | |
| ├ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`、`staticcheck ./...` | pass | `gofmt: ok`、golangci-lint `0 issues.`、vet と staticcheck は出力なし。`Skipping` の行は無い(4 つとも走った)。merge で入った main の Go も含めて通る |
| └ `secret-scan-branch.sh` | pass | `scanned e902705c..8460bf64 against origin/main: clean` |
| `go build ./cmd/ralph` と `ralph org reseat --help` | pass | b142944d の 2 つの文が出る。本文は evidence の log にある |
| `./scripts/plan-visual.sh digest` | 一致 | `f7e78c0dd96c`。計画の `- Approved:` の digest(1af2444a で記録し直したもの)と同じ |

## Documentation drift

この回の drift は、まだ開いている V17、V18、V20〜V23 の 6 件(LOW)と、merge で入った V24 の 1 件(MEDIUM)で、すべて `/sync-docs` に回す。どれもコードの動きは変えない。help の V19 は b142944d で直った。

V17、V20、V21 は、cycle 3 の動きで文字どおりには当てはまらなくなった文を含む。当てはまらないのは、立て直しの数秒の窓でほかのコマンドが同じ座席を別の設定で立て、reseat にその座席の `--driver`・`--model` が渡っていない場合だけである。その場合も、reseat が止めたまま残した座席は必ず打ち直しに入り、外れた座席が動いている設定は stderr の error が言う。どの文も AC を偽にせず、直しは文書で済むので、fail にはしない。help(b142944d)の文が、直すときの文面の手本になる。

V24 は、director が `usage_limited` の件を受けたとき、つまり reseat が役に立つ場面で、人に「立て直す動詞は無い」と伝える文である。同じ skill の「座席を立て直す」が正しい手順を書いており、コードも AC も変わらないので fail にはしないが、この PR の `/sync-docs` で直し、register に持ち越さない。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| V24 director skill(4 つの写し)の `usage_limited` の件の扱い(`.claude/skills/director/SKILL.md:1046-1047`) | No(MEDIUM) | 「座席を立て直す動詞は issue #312 で扱っていて、この版にはない。」は、merge 8460bf64 のあと偽になった。同じファイルの「## 座席を立て直す」(1430 行目)が `ralph org reseat` を書く。#312a(PR #327)はこの文を reseat より前に書き、merge は衝突なしで両方を残した。直し方: その文を、人が別のアカウントで立て直すと決めたら下の「座席を立て直す」の手順で `ralph org reseat` を打つ、という指し先に替える(`/org` skill の「座席の立て直し」と同じ言い方)。仕様の「依存」の行(`docs/specs/2026-11-12-org-reseat.md:139`)の「director skill の案内で打ち方をつなぐ」は、この直しで満たされる。立て直したあと、その `usage_limited` の件を resolve することも、同じ箇所で一言あるとよい。4 つの写しを `scripts/sync-skills.sh` と `templates/base` への写しで揃える。この直しは tech-debt 496 行目の (h) の Trigger(director skill の「座席を立て直す」の節の編集)には当たらない(直すのは `### 件の処理` の中)が、464 行目(director skill の次の変更)には当たる |
| V17 仕様(`docs/specs/2026-11-12-org-reseat.md`) | No(LOW) | (a) 起動のときの検出(立て直しの起動が idempotent で返り、返った座席の役割・driver・モデルが計画と違えば `start_failed` にしてそこで止まる。同じなら `started`)を、FR-5a・FR-6・D3 のどれも言わない。FR-5a の読み直しは止める直前だけを書く。(b) FR-6(69 行目)の「起動の途中で失敗したら、立った座席と立たなかった座席、打ち直し方(`ralph org reseat --org-id <id> --seat <立たなかった座席>` …)を出して」は、その座席に `--driver`・`--model` が渡っていない reseat では当てはまらない。座席は `start failed:` の行に出るが、打ち直しの `--seat` に入らず、対象の最後なら打ち直しは出ない(`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel` が `Retry == ""` を固定する)。(c) 非機能要件 1(78 行目)の「止めたあとに失敗したときは、座席の状態と打ち直し方を必ず出す」も同じ場合に当てはまらない。成り立つ形は「reseat が止めたまま残した座席は必ず打ち直しに入る。ほかのコマンドが立てて動いている座席は外し、その座席の今の設定は error が言う」。(d) Status に 3 回目のパイプラインの段落が無い。(e) FR-8(71 行目)の「最後に、org レベルの event `reseated` … を 1 件書く」は条件を言わないが、コードは座席を 1 つも止めず立てようともしなかった reseat(最初の座席の止める前の読み直しで終わったもの)では書かない。self-review の L21(help の文)と同じ中身で、1 回目からこの形。org skill の表の行(「座席を 1 つでも止めるか立てようとした reseat は、途中で終わっても…」)と org runtime の仕様の改訂 (c) は条件を書いているので、その言い方に合わせればよい |
| V18 計画(`docs/plans/active/2026-11-12-org-reseat.md`) | No(LOW) | (a) P6(64 行目)の「ほかの動き手との重なりは直前の読み直しで防ぐ」は、止めてから立てるまでの重なり(cross-review 2 回目の指摘)を言わない。この重なりは起動のときの検出(def8d2a3、df8f674e)が受け持つ。(b) Risks の 2 つ目(124 行目)の「Spawn は自分のロックの下でもう一度判定するので、二重に立つことはない。断られたら FR-6 の打ち直しの案内に乗る」は、Spawn が断らずに idempotent で返す場合を言わない。P6 と Risks は digest の範囲に入るので、直すと digest が `f7e78c0dd96c` から変わり、承認の記録し直しが要る(1af2444a と同じ)。(c) Progress checklist に 3 回目の項(cross-review の triage bcadf0b4 と cap の引き上げ、def8d2a3、df8f674e、self-review の 6c24bfa2 と 975e9f20、help の直し b142944d、self-review の 1326092b、main の merge 8460bf64)が無い。Progress は digest の範囲の外 |
| V20 `README.md:303`(reseat の段落) | No(LOW) | 「a stop or start that fails ends the reseat there with the seats stopped, started, and left, the reseat command that starts the rest」と「right before each stop it reads the ledger again and leaves alone a seat another command changed」が、起動のときの検出と、打ち直しから外す座席を言わない。help(b142944d)にはもう入った中身で、行番号は merge で 300 から 303 に動いた。同じ段落の「the reseat ends with one `reseated` event of the org」も、V17 (e) と同じく条件を言わない |
| V21 org skill(4 つの写し)の表の `reseat` の行(`.claude/skills/org/SKILL.md:371`)と「座席の立て直し」(1623 行目から) | No(LOW) | (a) 表の行の「途中で失敗すると、止めた座席・立てた座席・残った座席と、残りを立てる reseat のコマンドを出して」と「座席を止める直前に台帳を読み直し、…その座席を止めずにそこで終わる」。(b) 手順 3・4(1667〜1677 行目)が起動のときの検出を言わない。(c) 途中で失敗したときの段落(1721 行目から)の「`to start the seats left, run:` に続く reseat のコマンドが出て」は、外した座席が最後の対象のとき当てはまらない。続く「`--seat` は起動に失敗した座席も受け付けるので、そのまま通る」は、`start failed:` の座席が打ち直しに入る前提で読めるが、外した座席は入らない。外した座席には `left as it is:` の行も出ない(self-review の L19)。なお、止める側の `Changed` が唯一の対象のときも打ち直しは出ず(`Retry` の `left` が空)、(c) の 1 文目は 1 回目から、その場合に当てはまっていなかった。行番号は merge で 11 行動いた(main の `usage_limited` の項の分) |
| V22 director skill(4 つの写し)の「座席を立て直す」の 7(`.claude/skills/director/SKILL.md:1489` から) | No(LOW) | stderr の理由が「ほかのコマンドが別の設定で座席を立てた」だったときの扱いを言わない。7 は「理由が直ったら、`to start the seats left, run:` のあとに出た reseat のコマンドを…そのまま打つ」と書く。`--driver`・`--model` を渡した reseat では、その打ち直しが、ほかのコマンドが立てた座席(TASK を持ち始めているかもしれない)を止めて頼んだ設定で立てる。渡さなかった reseat では、座席は打ち直しから外れ、最後の対象なら打ち直しは出ない。どちらの場合も「人に伝え、頼んだ設定に移すかを人に決めてもらう」の 1 文が要る。V17〜V23 のうち director の動きに関わるのはこれだけ(V24 は別の箇所)。行番号は merge で 1467 から 1489 に動いた |
| V23 `docs/tech-debt/README.md` の reseat の行(merge で 485〜487 行目から 494〜496 行目に動いた) | No(LOW) | (a) 495 行目(ファイルの大きさ)の Trigger「The next change to `reseat.go`, `reseat_test.go`, …」に def8d2a3 と df8f674e が当たった(`reseat.go` は 1,307 → 1,361 行)のに、持ち越しの注が無い。新しい `reseat_start_test.go` は 237 行で、800 行の下。同じ行の「The split of `org_reseat_test.go` goes with the next change to it」の次の変更は b142944d(+6 行、852 → 858)で、分けていない。`internal/cli/org_reseat.go` も 510 → 516 行になった(800 行の下)。(b) 494 行目の (e)「five comments point at `docs/plans/active/…`」は 6 か所になった(`reseat_start_test.go:9`)。(h)「four places」は 5 か所になった(`reseat_start_test.go:9-10` の「cross-review cycle 2, finding 1」)。b142944d はどちらも増やしていない。(c) 494 行目の Why deferred の「after the pipeline reached its cap (`max_pipeline_cycles` 2)」は、cap を 3 に上げたので今は偽。(d) self-review の L19(起動側の `Changed` の座席の stdout の行)と、1326092b の L20(help の段落の途中の短い行)、L21(`reseated` の help の文)は、lead の判断で register に入るが、まだ無い。足すなら見出しの件数(Eleven)も直す。(e) 494 行目と 495 行目の Related の「verify report の … in its second run」は、2 回目の本文が 80f58f86 に移った。ID はこの report の「前の回の drift と INFO」に残したので引けるが、commit を添えるとよい。(f) b142944d は `orgReseatHelp` と `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` を変えたので、494 行目の (f) の Trigger(「the next change to the help of `ralph org reseat`」)と (j) の Trigger(「The next change to `orgReseatHelp` … or `TestOrgReseatHelp_…`」)に当たった。どちらも払っていない。help は今も `ralph org status --org-id <id>` を読めとは言わず((f))、「The `ralph org read` and `ralph org send` in its output read no ralph.toml」の文も残る((j))。持ち越しの注が要る。cycle 3 のコミットは、494 行目の (a)〜(e)・(g)〜(k)、496 行目の (a)〜(j)、361 行目の Trigger には当たらない(`startAll` の終わり方は変わったが、止める順と立てる順、`printOrgReseatResult`、`reseatSeatNow`、`Stop`、`PlanReseat`、`runOrgReseat` は差分の外)。sync-docs が V21 と V22 で skill を直すと、496 行目の (h) に当たる。merge は reseat の 3 行に触れていない(語単位で両側を残したのは 118、180、464 行目) |
| `AGENTS.md:91`、`docs/quality/quality-gates.md`(2 つの写し)、`internal/cli/org_legacy_ledger.go` のコメント、`--config` と `--seat` のフラグの usage | Yes | cycle 3 の直しと merge は、ここに書いた reseat の中身(reseat の役割、見張りと server の gate、台帳の扱い、読む設定、既定の対象)を変えていない。AGENTS.md の行と quality-gates の行には main の `usage_limited` の文が加わり、reseat の文と食い違わない |
| `docs/specs/2026-08-01-org-runtime.md` の「2026-11-12 改訂」 | Yes(I7、I14) | 要約の水準で、偽の文は無い。merge で、すぐ下に main の「2026-11-13 改訂」が入った |

## 前の回の drift と INFO

| ID | 回 | 今 |
| --- | --- | --- |
| V1〜V8 | 1 回目 | a40c646f で直った |
| V9〜V13、V16 | 2 回目 | 31f77c53 で直った(仕様の FR-2・D2 の注・D5・FR-6・FR-7、計画の P7 と Progress、README、skill の 8 つの写し) |
| V14 | 2 回目 | help の 3 つの文(`read` と `send` は ralph.toml を読まない、`--config` を繰り返すコマンドの例外、断りが並べる座席)と `ReseatParams.HintConfigPath` の doc。b142944d は help を変えたが、この 3 つの文には触れていない。tech-debt 494 行目の (j)(k) に入っている |
| V15 | 2 回目 | tech-debt の持ち越しの注と行数。31f77c53 で直った。cycle 3 の分は V23 |
| V19 | 3 回目(bab7b33a) | b142944d で直った。help が起動のときの検出と、打ち直しから外す座席と、打ち直しが出ない場合を言う。新しい help の文に偽は無い(上の「bab7b33a のあとの 3 コミット」の 1) |
| I5 | 1 回目 | plan を指す comment(self-review の L6、tech-debt 494 行目の (e))。6 か所になった(V23 (b)) |
| I6 | 1 回目 | 行数(tech-debt 495 行目)。V23 (a) |
| I7 | 2 回目 | `docs/specs/2026-08-01-org-runtime.md` の改訂 (b)(c) は要約の水準で、起動のときの検出にも触れないが、偽の文は無い |
| I8 | 2 回目 | reseat は #310 の hook の承認の通知を出さない(self-review の L14、tech-debt 494 行目の (i))。cycle 3 も変えていない |
| I9 | 2 回目 | AC-12 の実機の binary(dd204370)は、cycle 2 と cycle 3 の直しより前 |
| I10 | 2 回目 | I5・I6 と、self-review の L13(triage を番号で引く test の comment、tech-debt 494 行目の (h))。L13 は 5 か所になった(V23 (b)) |

## Observational checks

- HEAD から `go build` した binary で `ralph org reseat --help` を見た。b142944d の 2 つの文があり、V19 の穴は閉じている。L20 の 50 文字の行も出る
- `./scripts/plan-visual.sh digest` で計画の digest を計算し直した(一致)
- skill の 8 つの写しの sha256 を取った(組ごとに一致)
- merge 8460bf64 の衝突した 6 ファイルと、自動で merge された skill の 8 つの写しで、行の数を 3-way で数え合わせた。両側が同じ行を変えた 5 行は、語単位で base に両側の変更を当てて merge の行と比べた(すべて一致)。スクリプトは repo の外に置いた
- `docs/tech-debt/README.md` の 361、494、495、496 行目を python で欄に切り、Trigger の欄を cycle 3 の差分と b142944d に突き合わせた(V23)
- reseat の文と #312a の文の食い違いを、`git grep` で「この版にはない」「立て直す動詞」「#312c」「usage_limit」を引いて探した(V24、I14)
- INFO:
  - I11 ほかのコマンドが同じ役割・driver・モデルで立てた座席を、reseat は `started` と数え、`reseated` に `started` と書き、DECISION の「ralph org reseat が座席 … を止めて立て直しました」に入れる。その座席の `spawned` に `reseat=<id>` は無い(`TestReseat_ASeatAnotherCommandStartedOnTheSameSettings` が固定する)。FR-8 と AC-15 の「立てた座席の `spawned` に `reseat=<id>`」は、この座席に当てはまらない。idempotent は 1 回目から `started` に数えていたので前からの動きで、cycle 3 で test が固定した。director skill の 6 は、leader がもう送り直していれば DECISION を打たないと書くので、害は見つけていない
  - I12 `--driver`・`--model` を渡した reseat で起動が `start_failed` になっても、`reseated` の ` changed=<座席>:<driver>/<model>` は頼んだ変更を書く(`seats=implementer:start_failed changed=implementer:codex/gpt-x`、test が固定する)。FR-8 の「driver と model を変えた座席」は、変えられなかった座席も含むことになる。spawn の失敗でも 1 回目から同じで、座席の結果は `seats=` で分かる
  - I13 cross-review の triage(bcadf0b4)の理由の欄「座席は active なので、出す打ち直し(P7 で active な座席を受け付ける)で止めて頼んだ設定で立て直せる」は、df8f674e のあとは `--driver`・`--model` を渡した reseat にだけ当てはまる。triage は次の `/cross-review` が書き直す記録なので、直す対象にしない
  - I14 org runtime の仕様の「2026-11-13 改訂」の (d)「座席を別のアカウントや driver で立て直す動詞は、この改訂の対象ではない(… #312c)」は範囲の文なので偽ではないが、merge のあとは同じ仕様の「2026-11-12 改訂」が動詞を足している。(d) の末尾に「(2026-11-12 改訂の `ralph org reseat`)」と添えると、読み手が迷わない。org skill の `usage_limited` の項も reseat を指さないが、偽の文は無い。どちらも直さなくてよい

## Coverage gaps

- テストは走らせていない。上の判定は、コードと、テストが該当の動きを固定していることを読んで出した。合否は `/test` で確かめる。merge で入った main の Go のテストも、この回は走らせていない
- CLI のテストは、起動側の `Changed` の出力(`start failed:` の行だけが出て、`left` の行も打ち直しも無い)を固定していない(grep)。org 層の test は `Retry` と `Decision` を固定する。b142944d の pin は help の文面を固定するだけで、座席が外れるかは見ない
- 不一致の座席の後ろの座席が打ち直しに入ることは、非 leader の test(implementer のあとの reviewer)が固定する。leader の不一致のあとに非 leader が続く reseat の test は無い
- 起動のときの検出が比べるのは役割・driver・モデルだけで、scope・cwd・permission mode は比べない(self-review の「指摘にしなかったもの」と同じ)。読み直しと Stop の書き込みのあいだの窓、Stop が座席の spawn のロックを取らないことは、計画の Risks のとおり残る(tech-debt 361 行目)
- 実機では、AC-1 の implementer と承認の検査、AC-3、AC-10、AC-17、AC-18、切れた reseat、signal、cycle 2 と cycle 3 の直しを見ていない(テストだけ)。reseat と #312a の `usage_limited` の件をつなぐ流れ(件を受けた director が人と決めて reseat を打つ)も、実機でもテストでも通していない
- Insight event: lead の指示(report だけをコミットする)に従い、`docs/insights/events/` には足していない。3 回目の中の更新なので、2 行目も作らない。3 回目の最初の本文で `./scripts/insights-append.sh --slug org-reseat --flow standard --phase verify --verdict pass --cycle auto --low 7 --source skill --events-dir <scratch>` から作った行は `{"schema":1,"ts":"2026-11-13T04:59:56Z","slug":"org-reseat","flow":"standard","phase":"verify","cycle":2,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":7},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`。この更新の件数は MEDIUM 1(V24)、LOW 6(V17、V18、V20〜V23)
- 足すとよい確かめ: V24 は、別のブランチで書いた文が merge で偽になった型で、衝突が無いので merge では誰の目にも入らない(self-review の L14 と同じ道)。merge のあとに、merge で入った側の skill の差分を「立て直」「reseat」で grep するのが、いちばん安い。V17〜V22 は、文書がコードの失敗の道に遅れた型で、tech-debt 496 行目の (h)(skill の reseat の行と help を比べるテスト)が止める

## Verdict

- Verified: 静的検査はすべて pass(merge で入った main の Go を含む)。b142944d で V19 は直り、足した help の文に偽は無い。merge 8460bf64 は衝突した 6 ファイルで両側の文をすべて残し、skill の 8 つの写しは組ごとに揃う。cycle 3 の 2 件(def8d2a3、df8f674e)と cycle 2 の 3 件は、コードとテストに反映されている。1 本目の FR-1〜FR-8、FR-5a、FR-11、D1〜D6 と D2 の注、AC-1〜AC-11、AC-15〜AC-21 は、コードとそれを固定するテストの対応で満たす。計画の digest は承認の記録と同じ
- Partially verified: AC-12(実機は AC-1 を 2 席で確かめ、implementer と承認と cycle 2・cycle 3 の直しは見ていない)
- Not verified: テストの実行(`/test`)。実機での AC-3、AC-10、AC-17、AC-18、切れた reseat、signal、cycle 2 と cycle 3 の直し
- 判定: pass。drift は MEDIUM 1 件(V24)と LOW 6 件(V17、V18、V20〜V23)で、すべて `/sync-docs` に回す。V24 は merge で偽になった director skill の文で、この PR の `/sync-docs` で直し、register に持ち越さない。コードも AC も偽にしないので fail にはしない。V17、V20、V21 は、ほかのコマンドが別の設定で座席を立てた狭い場合に文字どおりには当てはまらない文を含むが、AC を偽にせず、reseat が止めたまま残した座席は必ず打ち直しに入るので、fail にはしない
