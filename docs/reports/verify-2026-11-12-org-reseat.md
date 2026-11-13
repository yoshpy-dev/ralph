# Verify report: org-reseat(`ralph org reseat`、#312c の 1 本目)

- Date: 2026-11-13(JST)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(spec `docs/specs/2026-11-12-org-reseat.md`)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 3 回目(cycle 2 の cross-review のあと cap を 3 に上げた回。`cycle-count.json` は `cycle` 2、`cap` 3)。テストは走らせていない(`/test` の担当。同じ機械でほかのセッションがテストを回しているので、`go test` は 1 本も打っていない)
- この report は 2 回目(80f58f86)を置き換える。1 回目の本文は `git show af927216:docs/reports/verify-2026-11-12-org-reseat.md`、2 回目は `git show 80f58f86:docs/reports/verify-2026-11-12-org-reseat.md` で読める。tech-debt の register が引く前の回の ID(I5、I6、I8、I10、V14、V15)は、下の「前の回の drift と INFO」に残した。この回の drift は V17 から番号を続ける
- 3 回目の中の更新: 3 回目の最初の本文(bab7b33a)のあとに、help の直し b142944d、self-review の report 1326092b、origin/main(e902705c、#312a の PR #327 と PR #325)の merge 8460bf64 が入った。その 3 つを下の「bab7b33a のあとの 3 コミット」で読み、ほかの節は HEAD に合わせて直した。最初の本文は `git show bab7b33a:docs/reports/verify-2026-11-12-org-reseat.md` で読める
- Scope: `git diff e902705c...HEAD`(merge base e902705c、HEAD 8460bf64、35 ファイル)。merge の前の範囲(`git diff 4220ea7c...975e9f20`)と同じ 35 ファイルで、merge のあとに Go で変わったのは `internal/cli/org_reseat.go`(help、+8/-2)と `internal/cli/org_reseat_test.go`(+6)だけ。`internal/org/` の reseat の 3 ファイルと `feature.go` は bab7b33a と同じバイト列。main から入った Go のうち、`internal/org/` の変更(`watch*.go`、`usage_limit_screen.go`、`inbox.go`、`escalate.go`、`resume_screens.go`)は reseat のファイルから呼ばれない(grep)。`internal/cli/` で reseat が呼ぶのは `scaffold_version_notice.go` の `printScaffoldVersionNotice`(`runOrgReseat` から)だけで、main の変更はその中の `semverWithV` を同じ本体の `upgrade.SemverWithV`(`internal/upgrade/behavior_notes.go:183`)に置き換えた移動なので、reseat の出力は変わらない。判定の範囲は 1 本目の FR-1〜FR-8、FR-5a、FR-11、D1〜D6 と D2 の注、計画の AC-1〜AC-12 と AC-15〜AC-21。FR-9、FR-10、AC-13、AC-14 は 2 本目なので対象外
- 4 回目(cycle 3、cap 3、HEAD d04482b3): 末尾の「4 回目: cycle 3 の cross-review の直し」の節に足した。今の判定はその節の `## Verdict` で、それより上の節は 3 回目(2194a9e2)のまま残した
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

---

# 4 回目: cycle 3 の cross-review の直し(HEAD d04482b3)

- Date: 2026-11-13(JST)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 4 回目で、`cycle-count.json` は `cycle` 3、`cap` 3(3 回目の cross-review、7ea546da の ACTION_REQUIRED 2 件を直す回)。テストは走らせていない(`/test` の担当。ほかのセッションが共有のテストの枠を使っているので、`go test` は 1 本も打っていない)
- Scope: 前の回(2194a9e2)のあとの 9 コミット。test の report c856fc27、sync-docs の d3d15d13、承認の記録し直し b8e6ed08(digest `95a056c00a9a`)、origin/main(bbe2807d、PR #328 の `ralph org send` の bracketed paste)の merge 46dfc5a8、triage 7ea546da、直し ffd32109(DECISION の `send` と、その前に出す `read` も `--config` を繰り返す)、84346015(start の前に org の disband を見つけたら終える)、c4f55aef(stop の前も同じ)、b568ca8c(disband で終えたときの `not stopped:` の行)、self-review d04482b3(Merge yes、LOW L22〜L25)。merge base は bbe2807d で、`git diff bbe2807d...HEAD` は前の回と同じ範囲に、上の Go の 6 ファイル(`internal/org/reseat.go`、新しい `reseat_disband_test.go`、`reseat_test.go`、`reseat_start_test.go`、`internal/cli/org_reseat.go`、`org_reseat_test.go`)の変更が加わった形
- 判定の範囲: lead が名指した FR-5a、FR-6、FR-7、FR-8、非機能要件 1、計画の P6 と Risks のほかの動き手の行、AC-15〜AC-18。ほかの AC は前の回の判定のままで、4 つの直しが触れていないことを hunk の関数で確かめた(下の Observational checks)
- Evidence: `docs/evidence/verify-2026-11-12-org-reseat.log` の末尾の「4th run (cycle 3, cap 3)」の節(`run-static-verify.sh` の全出力、HEAD から作った binary の `ralph org reseat --help`、merge 46dfc5a8 の 3-way の行の数え合わせと語単位の突き合わせ、skill の 8 つの写しの sha256、hunk の関数の一覧、plan を指す comment と triage を番号で引く comment の一覧、行数、plan の digest、`cycle-count.json`)。ラッパー自身の記録は `docs/evidence/verify-2026-11-13-082603.log`

## 4 つの直し

| # | 直し | 判定 | 根拠 |
| --- | --- | --- | --- |
| 1 | ffd32109: DECISION の `ralph org send` と、その前に出す leader の pane の `ralph org read` も、絶対パスにした `--config` を繰り返す | 反映済み | `ReseatDecisionCommand` と新しい `reseatLeaderReadCommand`(`internal/org/reseat.go`)は、打ち直しと断りと同じ `reseatVerbCommand` を通る(verb と引数、`--config`、`--state-dir`、send だけ最後に `--text`)。`finish` が両方を作り、`ReseatResult.DecisionRead` に入れる。CLI の `printOrgReseatResult` は `res.DecisionRead` をそのまま出し、引数 `hintStateDir` と `orgReadCommandHint` の呼び出しは消えた。`ralph org send` と `ralph org read` は `newOrgRuntimeAt`・`newOrgRuntime` から `resolveOrgConfig` で設定を読むので(`internal/cli/org.go`)、直しの理由は成り立つ。help、`HintConfigPath` の doc、help の pin の「read no ralph.toml」は消えた。テスト: `TestReseatDecisionCommand_RepeatsTheConfigAndTheStateDir`、`TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`、`TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` |
| 2 | 84346015: 各座席を立てる直前に manifest を読み直し、計画のあとの `disbanded` があれば、その座席から先を立てずに終える | 反映済み | `PlanReseat` が `ReseatPlan.LastDisbandedIndex`(`reseatLastDisbanded`、無ければ -1)を持たせ、`startAll` の先頭の `endIfDisbanded` が `reseatDisbandedNow` で比べる。違えば `ReseatResult.Disbanded` を立て、`reseatDisbandedErr` は「disbanded at <TS>」と、先に立った座席は `ralph org status` で動いているかを見ること、`ralph org start --plan` で立て直すことを言う。`finish` は `reseated` を書いたあと、`Disbanded` なら `Retry`・`Decision`・`DecisionRead` を作らずに返る。打ち直しを出さない理由は doc のとおりで、disband のあとの命には結びつきが無く、reseat は `reseatNoBindingErr` で断られる。`reseatLifeSeats` の命の始まりは `reseatLastDisbanded(...) + 1` に置き換えただけで、値は前と同じ(無いとき 0)。テスト: `TestReseat_OneSeatOfAnOrgDisbandedBeforeItsStart`、`TestReseat_AnOrgDisbandedAfterTheLeaderStarted`、`TestReseat_AnOrgDisbandedBeforeTheReseatStartsEverySeat`(過去の disband がある org で誤って終えないこと)、`TestReseat_AManifestUnreadableBeforeAStart`、`TestReseatDisbandedErr` |
| 3 | c4f55aef: 各座席を止める直前の読み直しでも、同じ disband を見る | 反映済み | `stopAll` の読み直しは `checkBeforeStop` に移り、1 回の読みで disband を座席の確かめより先に見る(`reseatDisbandedIn`)。disband のあとの命には座席の state event が無いので、先に見ないと `Changed` と取り違える。その取り違えは起きなくなった。`reseatSeatNow` は manifest を読まず、読んだ events を受け取る形になった。最初の座席の前で終われば `attempted` は立たず、`reseated` は書かない(FR-8 の「最初に止める座席の止める直前の読み直しで終わった reseat」と同じ枝)。2 つ目の座席の前なら `reseated` を `disbanded` のあとに書く。テスト: `TestReseat_AnOrgDisbandedBeforeItsFirstStop`、`TestReseat_AnOrgDisbandedBeforeItsSecondStop`、`TestReseat_AManifestUnreadableBeforeAStop` |
| 4 | b568ca8c: disband で終えたとき、止めていない座席を `left running:` ではなく `not stopped: <seat> (the org was disbanded meanwhile)` と出す | 反映済み | `orgReseatLeftLine` に `disbanded` の引数と枝が加わった。disband は org の座席を止めるので(`Disband` の doc)、`left running:` は偽になる。`TestOrgReseatLeftLine` が outcome と `disbanded` の組を表で固定する |

self-review の L22〜L25(help の 40 文字の行と 108 文字の comment 行、`Decision`・`DecisionRead` が空であることを固定する test が無いこと、`Retry`・`Decision` の 4 つの doc に disband の例外が無いこと、`changed`・`disbanded` の語と 2 つの check の真偽の向き)は、コードを読んで同じ結論になった。L24 は Go の doc comment の drift で、下の V31 の register の (n) に入る。

## merge 46dfc5a8(origin/main bbe2807d、PR #328)

| 確かめ | 結果 |
| --- | --- |
| 3-way の行の数え合わせ(base e902705c、両親 b8e6ed08 と bbe2807d) | AGENTS.md と `docs/tech-debt/README.md` で、合わない行は両側が同じ 1 行を変えた所だけ(AGENTS.md の `internal/org/` の行、tech-debt の「Two documentation findings of org-stop-all」の行)。自動で merge された org skill の 4 つの写し、`.claude/rules/ralph/agent-messaging.md` とその `templates/base` の写し、`docs/specs/2026-08-01-org-runtime.md`、`internal/cli/org.go`、`internal/org/verbs.go` は全行が合う |
| 両側が変えた 2 行の語単位の突き合わせ | base にどちらの側の変更も当てると merge の行と一致する(AGENTS.md は両順で一致。tech-debt の行は main の持ち越しの注を先に置いた順で一致)。落ちた語も、どちらにも無い語も無い |
| AGENTS.md の `internal/org/` の行の読み | 「…the screen fallback of send (…), the bracketed paste of send (`send_paste.go`: …), `ralph org status --live` (…), the reseat of a feature org's seats (`reseat.go`: …), and the pane lock (…)」と並び、括弧は閉じ、backtick は対になっている。文として正しく読める。reseat の説明は要約の水準で、`--config` と disband の終わり方は言わないが、偽の文は無い |
| main の Go と reseat | main の変更は `Send` の bracketed paste(`internal/org/verbs.go` の `SendParams` と `Send`、`send_paste.go`)と send の help で、reseat のファイルからは呼ばれない。reseat が出す DECISION の本文(`ReseatDecisionText`)に paste の印は入らないので、claude の leader に送っても send の新しい断りに当たらない |
| main が持ち込んだ「まだ無い」の文 | main の追加行を `#312`、`reseat`、「この版にはない」「対象ではない」「not yet」で引いた。当たったのは tech-debt の行の #312a の持ち越しの注(履歴の文)だけで、偽になった文は無い |
| 写し | `check-sync.sh` は DRIFTED 0。skill の 8 つの写しは組ごとに sha256 が一致する(org `66082311895b` 2,516 行、director `5b2c1c81ccdc` 2,136 行) |

## Spec compliance(この回に関わる行)

| 項目 | Status | Evidence |
| --- | --- | --- |
| FR-5a | 満たす(注) | 止める直前の読み直しは、座席の最新の state event に加えて、計画のあとの disband も見る(`checkBeforeStop`)。座席の記録が変わった場合の動き(止めずに終え、打ち直しを出す)は前のまま。disband の場合は打ち直しを出さず、`ralph org start --plan` で立て直すことを言う。FR-5a の文は disband を言わない(V26) |
| FR-6 | 満たす(注) | 起動の前にも disband を見る。FR-6 の「reseat が止めたまま残した座席は、いつも打ち直しの `--seat` に入る」は、disband で終えた reseat では当てはまらない(止めた座席は `left stopped:` の行に出るが、打ち直しは出ない)。打ち直しを出しても disband のあとの命では断られるので、コードの選び方は理由が立つ(triage と consult の判断)。文書の drift とし、コードの不適合とはしない(V26) |
| FR-7 | コードは triage の直しどおり。仕様の文が偽(V25) | 仕様の FR-7 は「DECISION の `send` と、その前の `read`、`Spawn` の error が名指す `ralph org stop` には付けない」と書くが、ffd32109 のあと `send` と `read` は `--config` を繰り返す。triage(7ea546da)は「仕様の FR-7 と org skill の…も逆向きに直す」と決めている。`ralph org stop` の例外は残る(`stopSeatCommand`)。FR-7 の DECISION の条件(「leader が対象に無く立てた座席があるときは(途中で起動が失敗したときも)」)も、disband で終えた reseat では出ない(V26) |
| FR-8 | 満たす | disband で終えた reseat も、何かを止めたか立てようとしたあとなら `reseated` を書き(`TestReseat_AnOrgDisbandedBeforeItsSecondStop` が `disbanded` のあとにあることまで見る)、最初の座席の前の読み直しで終えたら書かない(`...BeforeItsFirstStop`)。FR-8 の例外の文「最初に止める座席の止める直前の読み直しで終わった reseat」は、disband もその読み直しで見つかるので当てはまる |
| 非機能要件 1 | 満たす(注) | 「止めたあとに失敗したときは、座席の状態と打ち直し方を出す」「reseat が止めたまま残した座席は、いつも打ち直しに入る」は、disband で終えた reseat では打ち直しが出ないので文字どおりには当てはまらない。座席の状態は stdout の行が言い、立て直し方は error が `ralph org start --plan` と言う(V26) |
| 計画の P6、Risks の 2 つ目 | 文書が直しに遅れている(V27) | P6(64 行目)は、ほかの動き手との重なりを、止める前は直前の読み直し、起動のときは idempotent の比べで防ぐと書き、disband を言わない。Risks(124 行目)は台帳を変える動き手に disband を挙げず、残る窓を「読み直しと Stop の書き込みのあいだ」だけと書く。直しのあとは、起動の前の読み直しと `Spawn`・`StartFeature` の書き込みのあいだの窓も残る(`Disband` は `reseat.lock` を取らない。triage は Known gaps に書くと決めた) |
| AC-15 | 満たす | 上の FR-8 と同じ。disband の場合も、書いた `reseated` は座席ごとの結果(`stopped`、`not_stopped`、`not_started`、`started`)を持つ |
| AC-16 | 満たす(注) | 動詞とアカウントの移し方は、skill の 4 つの写しと help が言う(前の回のまま)。help は disband の終わり方を言うようになったが、skill は言わず、org skill の `--config` の文は偽になった(V25、V29、V30) |
| AC-17 | 満たす(注) | `reseat.lock` は前のまま。止める直前の読み直しは disband を先に見るようになった。AC-17 が挙げる場合(ほかの動き手が座席を立て直した、止めた)は、前のとおり止めずに終え、状態と打ち直し方を出す(`TestReseat_ASeatChangedBeforeItsStop`)。org の disband は、座席を止めたうえで org の命を終えるので別の終わり方になり、打ち直しの代わりに `ralph org start --plan` を言う。disband が途中で、座席の `stopped` だけが書かれ `disbanded` がまだのときは、座席の記録が変わった場合として読まれ、打ち直しが出る(self-review の読みと同じ。打ち直しは disband が済んだあとに断られる) |
| AC-18 | 満たす | 起動の失敗(leader、非 leader)のあとの打ち直しは変わっていない。manifest を読み直せないときは `Disbanded` を立てず、打ち直しを出す(`TestReseat_AManifestUnreadableBeforeAStart`)。disband は起動の失敗ではない |
| AC-3 | 満たす | DECISION の行は、`--config` を渡したとき、それを繰り返すようになった(`TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`) |

ほかの AC(AC-1、AC-2、AC-4〜AC-11、AC-19〜AC-21)と FR(FR-1〜FR-4、FR-5、FR-11)は、前の回の判定のまま。4 つの直しの hunk は `PlanReseat` の中では plan の literal の 1 行(`LastDisbandedIndex`)だけで、検査、対象の決め方、`--seat` の受け付け、止める順と立てる順、dry run の出力には触れていない。AC-12(実機)は I9 のまま一部で、binary dd204370 はこの回の直しより前。

## Static analysis(4 回目)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(exit 0) | 1 回で通った。末尾は `==> All verifiers passed.` |
| ├ shellcheck、`sh -n`、guard の awk の構文、hook の JSON(`jq -e`)、Codex の hook の single-source と検出の smoke、PR provenance | pass | |
| ├ `scripts/check-sync.sh` | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| ├ `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照 | pass | |
| ├ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`、`staticcheck ./...` | pass | `gofmt: ok`、golangci-lint `0 issues.`、vet と staticcheck は出力なし。`Skipping` の行は無い |
| └ `secret-scan-branch.sh` | pass | `scanned bbe2807d..d04482b3 against origin/main: clean` |
| `go build ./cmd/ralph` と `ralph org reseat --help` | pass | disband の文(「A disband of the org while reseat runs ends it before the next stop or start too, with no command to start the rest」)と、`read`・`send` も `--config` を繰り返す文が出る。「read no ralph.toml」は出ない |
| `./scripts/plan-visual.sh digest` | 一致 | `95a056c00a9a`。計画の `- Approved:` の digest(b8e6ed08)と同じ |

## Documentation drift(4 回目)

前の回の V17、V18、V20〜V24 と INFO I14 は、sync-docs の d3d15d13 で直った(仕様の FR-6・FR-8・非機能要件 1・D3・Status、計画の P6 と Risks と Progress、README、skill の 8 つの写し、tech-debt の持ち越しの注と (l)(m)、org runtime の仕様の改訂 (d) の指し先)。この回の drift は V25 から番号を続ける。V25 は MEDIUM、V26〜V31 は LOW で、すべて `/sync-docs` に回す。どれもコードの動きは変えない。

V25 は、`--config` を渡したすべての reseat の出力について、仕様と skill が逆のことを言う文なので MEDIUM とした。triage が「逆向きに直す」と決めているので、この PR の `/sync-docs` で直し、register に持ち越さない。V26〜V30 は、reseat のあいだに org が disband される狭い場合の終わり方を、文書がまだ言わないもの。仕様と README と skill の「いつも打ち直しに入る」「DECISION が出る」は、その場合に偽になる。

| ID | Doc / contract | In sync? | Notes |
| --- | --- | --- | --- |
| V25 | 仕様の FR-7(`docs/specs/2026-11-12-org-reseat.md:73`)と org skill(4 つの写し)の失敗の段落の最後の文(`.claude/skills/org/SKILL.md:1737-1738`) | No(MEDIUM) | 仕様は「DECISION の `send` と、その前の `read`、`Spawn` の error が名指す `ralph org stop` には付けない」、skill は「DECISION の `send` と `read`、`spawn` の失敗の理由に入る `stop` のコマンドには付かない。」と書く。ffd32109 のあと、`send` と `read` は絶対パスにした `--config` を繰り返す。直し方: どちらも「打ち直し、断り、DECISION の `send` とその前の `read` に付ける。`Spawn` の error が名指す `ralph org stop` には付かない」の形にする。org skill の表の `reseat` の行(371 行目)の「残りを立てるコマンドは…`--config` を引き継ぐ(断りが出す `ralph org stop`・`spawn`・`start` のコマンドも同じ)」は偽ではないが、同じ直しで DECISION の `send` と `read` も足すと揃う |
| V26 | 仕様(`docs/specs/2026-11-12-org-reseat.md`) | No(LOW) | disband の終わり方を言わない。(a) FR-5a(69 行目)の読み直しは「その座席の最新の state event」だけを確かめると書く。今は計画のあとの disband を先に見て、見つけたら止めずに終え、打ち直しを出さない。(b) FR-6(71 行目)の「reseat が止めたまま残した座席は、いつも打ち直しの `--seat` に入る」と、72 行目の起動のときの重なりの段落は、起動の前の disband の確かめを言わない。(c) FR-7(73 行目)の DECISION の条件に、disband で終えたときは出さないことが無い。(d) 非機能要件 1(81 行目)の「止めたあとに失敗したときは、座席の状態と打ち直し方を出す」「いつも打ち直しに入る」に、disband の例外が無い。成り立つ形は「disband で終えたときは打ち直しも DECISION も出さず、error が `ralph org start --plan` で立て直すことと、先に立った座席は `ralph org status` で見ることを言う」。(e) D3(48 行目)の重なりの扱いと D5(50 行目)の「途中で起動が失敗したときも、失敗の前に立った座席があれば…DECISION を出す」に disband が無い。(f) Status に 4 回目のパイプラインの段落(ffd32109、84346015、c4f55aef、b568ca8c)が無い。FR-8(74 行目)の例外の文は、disband もその読み直しで見つかるので直さなくてよい。添えるなら「(org の disband を見つけたときも)」 |
| V27 | 計画(`docs/plans/active/2026-11-12-org-reseat.md`) | No(LOW) | (a) P6(64 行目)の重なりの防ぎ方に、止める前と立てる前の読み直しで org の disband を見ること(見つけたら打ち直しを出さずに終え、`ralph org start --plan` を言う)が無い。(b) Risks の 2 つ目(124 行目)の動き手に disband が無く、残る窓は「読み直しと Stop の書き込みのあいだ」だけと書く。今は起動の前の読み直しと `Spawn`・`StartFeature` の書き込みのあいだの窓も残り、`Disband` は `reseat.lock` を取らない(triage の consult の結論で、Known gaps と tech-debt の候補)。P6 と Risks は digest の範囲に入るので、直すと digest が `95a056c00a9a` から変わり、承認の記録し直しが要る(b8e6ed08 と同じ)。(c) Progress checklist(138 行目から)に 4 回目の項(承認の記録し直し b8e6ed08、main の merge 46dfc5a8、triage 7ea546da、直し 4 コミット、self-review d04482b3)が無い。Progress は digest の範囲の外。AC-17(90 行目)は disband を挙げないので偽ではない |
| V28 | `README.md:303`(reseat の段落) | No(LOW) | (a) 「a stop or start that fails ends the reseat there with the seats stopped, started, and left, the reseat command that starts the rest」と「right before each stop it reads the ledger again and leaves alone a seat another command changed」が、disband の終わり方(次の stop か start の前で終え、打ち直しは出ず、`ralph org start --plan` で立て直す)を言わない。(b) 「When it started only seats other than the leader, the output ends with the `ralph org send` of a DECISION」は、disband で終えた reseat では出ない。help にはもう入った中身で、help の文が手本になる。DECISION の `read` と `send` が `--config` を繰り返すことは README は言っていないので偽ではないが、足すなら同じ箇所 |
| V29 | org skill(4 つの写し)の表の `reseat` の行(`.claude/skills/org/SKILL.md:371`)と「座席の立て直し」 | No(LOW) | V25 の文のほかに、disband の終わり方が無い。(a) 表の行の「途中で失敗すると、止めた座席・立てた座席・残った座席と、残りを立てる reseat のコマンドを出して」「座席を止める直前に台帳を読み直し」「leader 以外の座席だけを立て直したときは(途中で失敗したときも、その前に立てた座席について)…DECISION の…行が出る」。(b) 手順 3(1667 行目)と手順 4(1671 行目)。(c) 失敗の段落(1727 行目から)の理由の一覧(`stop_failed`、起動の失敗、座席の記録の変化)に disband が無く、残った座席の行の一覧に `not stopped:` が無く、「`to start the seats left, run:` に続く reseat のコマンド(立てていない座席が残るとき)」は disband で終えた reseat では偽。(d) DECISION の段落(1753 行目から)の「途中で失敗したときも立てた座席について…`ralph org read` のコマンドと…`ralph org send` の行が出る」も同じ。手順 5(1682 行目)の `reseated` の例外は、disband も最初の読み直しで見つかるので偽ではない。表の行は 2,619 文字で、直すと tech-debt 180 行目の (a) の Trigger に当たる(下の V31) |
| V30 | director skill(4 つの写し)の「座席を立て直す」の 7(`.claude/skills/director/SKILL.md:1490` から) | No(LOW) | (a) 「理由が直ったら、`to start the seats left, run:` のあとに出た reseat のコマンド…をそのまま打つ」に、disband で終えたときの扱いが無い。そのときは打ち直しが出ないので、director は人に、org が reseat のあいだに disband されたこと、先に立った座席は `ralph org status --org-id <org>` で動いているかを見ること、立て直すなら `ralph org start --plan` からで、それを打つかは人が決めることを伝える(director は人が言わないうちに org を立てない。per-plan の org は新しい命で計画の承認を求め直す)。(b) 1503〜1505 行目の「leader が対象でなく、先に立った座席があるときは、失敗の出力にも、その座席について `ralph org read` と DECISION の `ralph org send` の行が出る」は、disband で終えたときは偽。手順 6 は DECISION の `read` と `send` の `--config` を言わないので偽ではない |
| V31 | `docs/tech-debt/README.md` の reseat の 3 行(495〜497 行目)と、triage が回すと決めた 2 件 | No(LOW) | 下の表 |
| — | `AGENTS.md:91`、`docs/quality/quality-gates.md`(2 つの写し)、`docs/specs/2026-08-01-org-runtime.md` の 2026-11-12 改訂、help、`--config` と `--seat` のフラグの usage | Yes(I15、I16) | AGENTS.md の行は merge のあとも正しく読める(上の表)。org runtime の仕様の改訂 (c) の「座席を止める直前に台帳を読み直して、確かめのあとで変わった座席は止めない」は要約の水準で、disband と DECISION の `--config` を言わないが、偽の文は無い(I15) |

V31 の中身(tech-debt の各欄は python で 5 つに切り、Trigger の欄を 4 つの直しの hunk の関数に突き合わせた):

| 行 | 直すこと |
| --- | --- |
| 495(小さな指摘、Thirteen) | (j) の L15 から「`read` と `send` は ralph.toml を読まない」の半分を外す(ffd32109 が help、`HintConfigPath` の doc、pin から消した)。(j) の Trigger の直し方「the `ralph org read` and `ralph org send` of the output take `--state-dir` only」は、今のコードと逆になったので、`stop` の例外だけを言う形に書き直す。L22〜L25 を (n) として足し、見出しを Fourteen にする(L24 の 4 つの doc の例外の抜けも (n) に入る)。(e) は 6 か所から 7 か所(`internal/org/reseat_disband_test.go:11` が増えた)。(h) は 5 か所から 9 か所(増えたのは `internal/cli/org_reseat_test.go:288`、`internal/org/reseat_disband_test.go:11-12`、`internal/org/reseat_start_test.go:30`、`internal/org/reseat_test.go:1739` の「cross-review cycle 3, finding 1 / 2」。commit で引くなら ffd32109、84346015)。Trigger に当たったが払っていないものに持ち越しの注が要る: (d) `PlanReseat`(84346015 が plan の literal に `LastDisbandedIndex` を足した)、(f) help と `reseatLifeSeats`(ffd32109、84346015、c4f55aef が help を変え、84346015 が `reseatLifeSeats` の命の始まりを `reseatLastDisbanded` に置き換えた。help は今も `ralph org status` を読めと言わない)、(i) `runOrgReseat` と `printOrgReseatResult`(ffd32109、b568ca8c。codex の hook の通知は今も出ない)、(l) `orgReseatLeftLine` と `printOrgReseatResult`(b568ca8c が足したのは disband の枝だけで、`start_failed` かつ `Changed` の枝は無く、関数の doc の「A seat whose stop or start was tried has its line already」も残る)、(m) `orgReseatHelp`(3 コミットが変えたが段落を詰め直しておらず、L20 の 50 文字の行に L22 の 40 文字の行が加わった。L21 の「also when it ended early」は、最初の stop の前で disband を見つけた reseat にも当てはまらない)。当たっていないもの: (a)、(b)(`PlanReseat` の断りの文と `startFeatureInput` は変わっていない)、(c)、(g)、(k)(`reseatNoActiveSeatErr` は変わっていない)。Related に self-review の「cycle 3 の差分(cross-review の修正 4 コミット、7ea546da..b568ca8c)」と、この report の 4 回目の節を足す |
| 496(ファイルの大きさ) | Trigger(「The next change to `reseat.go`, `reseat_test.go`, …」)に 4 つの直しが当たり、払っていない。数字: `reseat.go` 1,361 → 1,537、`reseat_test.go` 1,855 → 1,868(ffd32109)、`reseat_start_test.go` 237 → 267、新しい `reseat_disband_test.go` 366(800 行の下)、`internal/cli/org_reseat.go` 516 → 521、`internal/cli/org_reseat_test.go` 858 → 923(「The split of `org_reseat_test.go` goes with the next change to it」の次の変更で、また分けていない)。数えたのは b568ca8c(Go を変えた最後のコミット)で、HEAD と同じ。`feature.go` は 848 のまま。`internal/org/verbs.go` は main の #328 で 2617 → 2635 になったが、この branch の増分(+20)の文は偽ではない(I17) |
| 497(test の穴) | (d) Gap 4 は払われた。`TestReseat_AnOrgDisbandedBeforeItsFirstStop`(c4f55aef)が、確かめと最初の stop のあいだの disband を固定する。しかも (d) の文「the retry command it prints names an org that has no binding any more」は偽になった(今は打ち直しを出さず、見つけるのは `reseatSeatNow` ではなく `reseatDisbandedIn`)。取り消し線で閉じる。(e) Gap 5 の「the re-read of the manifest」は、`TestReseat_AManifestUnreadableBeforeAStop` と `...BeforeAStart` が届くようになったので外す(`reseat.lock` の marker、乱数、`reseated` の追記は残る)。(c) は一部だけ: `TestOrgReseatLeftLine`(b568ca8c)が `left running:`、`left as it is:`、`not started:`、`not stopped:` の文面を関数の水準で固定するが、CLI で reseat を走らせて出す test は無く、`stop failed:` は今も固定されていない。Trigger に当たって払っていないもの: (c)(d) の `printOrgReseatResult` と `reseatSeatNow`、(i)(j) の `runOrgReseat`、(m) の `printOrgReseatResult` と `orgReseatLeftLine`(disband で終えた reseat の出力も CLI の test が無い)、(n) の `orgReseatHelp`。(g) 実機は、この回の直しも見ていない(binary は dd204370 のまま)。self-review の L23(`Decision`・`DecisionRead` が空であることを、どの test も直しを外して通る)は 495 の (n) に入れるなら、ここには要らない |
| 新しい行(2 件) | triage(7ea546da)は、consult の結論として次の 2 つを「tech-debt に回す」「候補として書く」と決めたが、register にまだ無い(`git grep` で `ApprovalDecisionCommand` も `Disband` と `reseat.lock` の組も見つからない)。(1) `Disband`(`internal/org/verbs.go`)は `reseat.lock` を取らないので、止める前と立てる前の読み直しから `Stop`・`Spawn`・`StartFeature` の書き込みまでの窓に入った disband を、reseat は見ない。`Disband` にロックを取らせると窓は閉じるが、`Disband` の意味が変わる。361 行目(Stop と Disband が座席の spawn のロックを取らない)と同じ型なので、そこから指してもよい。(2) `ApprovalDecisionCommand`(`internal/org/approve.go`)の DECISION の `ralph org send` は `--config` を付けず、`orgReadCommandHint`(`internal/cli/org.go:644-647`)の doc は「`Read` は Config を読まないので `--config` は要らない」と書く。どちらも別のディレクトリの壊れた `./ralph.toml` で `org: load config` で落ちる(ffd32109 が reseat で直した理由と同じ)。前からの形 |

sync-docs が V25 と V29 で org skill の reseat の行と「座席の立て直し」を、V30 で director skill を直すと、tech-debt 180 行目の (a)(表の行の長さ、今は 2,619 文字)、464 行目(director skill の次の変更)、497 行目の (h)(skill の reseat の節の編集)の Trigger に当たる。それぞれ持ち越しの注が要る。

## 前の回の INFO のうち、この回で変わったもの

| ID | 今 |
| --- | --- |
| I11、I12 | 変わらない。disband の確かめは idempotent の数え方と `changed=` の書き方に触れていない |
| I13 | 用が済んだ。triage は 7ea546da で 3 回目の cross-review の 2 件に書き直された |
| I15(新) | org runtime の仕様の 2026-11-12 改訂 (c) は、disband の終わり方と DECISION の `--config` を言わない。要約の水準で偽の文は無い。足すなら (c) に 1 文 |
| I16(新) | help の「When the leader is not one of the seats and keeps running, the output ends with the `ralph org send` command of a DECISION … also after a start that failed」は、disband のあとは leader が動き続けないので条件から外れ、偽ではない。L21 の文は、最初の stop の前で disband を見つけた reseat にも当てはまらない(V31 の (m)) |
| I17(新) | `internal/org/verbs.go` は main の #328 で 2617 → 2635 行になった。tech-debt 496 行目の「`internal/org/verbs.go` (2597 → 2617)」は、この branch の増分を言う文なので偽ではない |

## Observational checks(4 回目)

- HEAD から `go build` した binary で `ralph org reseat --help` を見た。disband の文と、`read`・`send` も `--config` を繰り返す文が出て、「read no ralph.toml」は出ない。段落の途中に 40 文字の行(``with `ralph org start --plan`. A seat it``)と 50 文字の行が出る(L22、L20)
- 7ea546da から HEAD までの `reseat.go` と `org_reseat.go` の hunk を、囲む関数で数えた。`PlanReseat` の中の変更は plan の literal の 1 行だけで、`reseatNoActiveSeatErr`、`reseatIdempotentMismatchErr` の本体、`orgReseatHerdrServerProbe`、`orgReseatHoldSignals` は変わっていない(tech-debt の Trigger の突き合わせに使った)
- plan を指す comment(7 か所)と、triage を番号で引く comment(9 か所)を `git grep` で数えた。折り返しで regexp が切れる箇所は、行を目で読んで数えた
- merge 46dfc5a8 の 3-way の数え合わせと語単位の突き合わせ(スクリプトは repo の外に置いた)
- `./scripts/plan-visual.sh digest`(一致)、skill の 8 つの写しの sha256(組ごとに一致)
- 仕様、計画、README、skill の 4 つの写し、tech-debt を、「--config」「disband」「打ち直し」「DECISION」で引いて、disband の終わり方と DECISION の `--config` に触れる文を拾った

## Coverage gaps(4 回目)

- テストは走らせていない。上の判定は、コードと、テストが該当の動きを固定していることを読んで出した。合否は `/test` で確かめる。merge で入った main の Go(send の bracketed paste)のテストも走らせていない
- disband で終えた reseat の CLI の出力(`not stopped:` の行、打ち直しと DECISION が無いこと、stderr の error)を、CLI を走らせて固定する test は無い。割り込みのフックが `internal/org` にあるので、CLI の test は `TestOrgReseatLeftLine` の関数の表だけ
- `Decision` と `DecisionRead` が disband で空になることは、`finish` の早い return を外しても通る(self-review の L23)。効く場面は `--seat implementer --seat reviewer` で reviewer の前に disband が入る reseat で、test が無い
- 窓: 止める前と立てる前の読み直しから、`Stop`・`Spawn`・`StartFeature` の書き込みまでのあいだの disband は見ない(`Disband` は `reseat.lock` を取らない)。disband が途中(座席の `stopped` は書かれ、`disbanded` がまだ)のときは、座席の記録が変わった場合として打ち直しを出し、その打ち直しは disband のあとに断られる
- 実機では、この回の直しを見ていない(binary dd204370)。ほかは前の回の Coverage gaps のまま
- Insight event: `./scripts/insights-append.sh --slug org-reseat --flow standard --phase verify --verdict pass --cycle auto --medium 1 --low 6 --source skill` で `docs/insights/events/2026-11-13-org-reseat.jsonl` に 1 行足した(`"cycle":3`、`"medium":1,"low":6`)。lead が示したコマンドに件数を足したのは、件数なしだと findings が 0 で記録され、この report の V25〜V31 と合わないため
- 足すとよい確かめ: V25 は、cross-review の直しがコードの向きを逆にしたとき、その向きを書いた文(仕様の FR と skill)を同じ cycle で拾えなかった型。直しのコミットの直後に、変えた関数名と、消した文の語(「read no ralph.toml」「付けない」「付かない」)で仕様と skill を grep するのが、いちばん安い。V26〜V30 は、文書がコードの終わり方の追加に遅れた型で、tech-debt 497 行目の (h)(skill の reseat の行と help を比べる test)が止める

## Verdict

- Verified: 静的検査はすべて pass。cycle 3 の cross-review の直し 4 コミット(DECISION の `send` と `read` の `--config`、stop と start の前の disband の確かめ、`not stopped:` の行)は、コードとテストに反映されている。FR-5a、FR-6、FR-8、非機能要件 1、AC-3、AC-15〜AC-18 は、コードとそれを固定するテストの対応で満たす(disband の場合は文書の drift として注を付けた)。merge 46dfc5a8 は、衝突した AGENTS.md と tech-debt の行で両側の語をすべて残し、AGENTS.md の `internal/org/` の行は正しく読める。skill の 8 つの写しは組ごとに揃う。計画の digest は承認の記録(b8e6ed08)と同じ
- Partially verified: AC-12(実機は前の回のまま。この回の直しは見ていない)
- Not verified: テストの実行(`/test`)。実機での disband の割り込み、DECISION の `--config`、cycle 2 と cycle 3 の直し
- 判定: pass。drift は MEDIUM 1 件(V25)と LOW 6 件(V26〜V31)で、すべて `/sync-docs` に回す。V25 は、仕様の FR-7 と org skill が「DECISION の `send` と `read` には `--config` を付けない」と書く文で、ffd32109 のあと偽になった。triage が文書を逆向きに直すと決めているので、この PR の `/sync-docs` で直し、register に持ち越さない。コードは triage の決めた向きで、AC を偽にしないので fail にはしない。V26〜V30 は、reseat のあいだの disband という狭い場合の終わり方を文書がまだ言わないもので、V31 は register の数字と持ち越しの注と、triage が回すと決めた 2 件
