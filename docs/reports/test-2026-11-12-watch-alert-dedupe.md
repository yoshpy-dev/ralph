# Test report: watch-alert-dedupe

- Date: 2026-11-12(run-test.sh は 11:15:01 から 11:36:45 JST)
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(`## Test plan`、AC1〜AC15)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 3/3。cycle 2 のあと `cycle-count.json` の `cap` を 3 に上げた)
- Scope: `git diff 028a8f6d...HEAD`(HEAD `10a86e28`)。cycle 2(HEAD `0a957745`)のあとの production の変更は S10 `ec538952`(cycle 2 の cross-review の 1 件: 利用者の `diff.relative=true` と下のディレクトリの cwd で、commit のパスが cwd からの相対になる)と S11 `e90c85da`(S10 の `--no-relative` を `-c diff.relative=false` に替えた。cycle 3 の self-review の C3-1)の 2 つで、どちらも `scopeCommittedPaths`(`watch_scope.go`)の git の引数の 2 行と doc comment だけを変えた。テストの変更は `watch_scope_test.go` だけで、新しいテスト `TestWatch_ReservedScope_CommitPathsStayRootRelativeUnderDiffRelative`、leader の cwd を下のディレクトリにする fixture `newReservedScopeFixtureAt`、スタブが先頭の `-c <key=value>` の組を飛ばして subcommand で振り分ける `scopeSubcommand`、git の引数を文字列で固定する 4 か所の期待への `-c diff.relative=false` の追加である。ほかの commit は記録だけを変えた。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。worktree の production code とテストには手を入れていない。変異は `git archive HEAD` で作った scratchpad の写しに当て、probe は `go test -overlay` で足した
- Evidence: `docs/evidence/test-2026-11-12-watch-alert-dedupe.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、meta と wrapper、`(cached)` だった 7 パッケージの打ち直し、この diff のテストの PASS 行、`-race` の回、coverage、変異の harness・定義・結果、D 変異の失敗の文、probe の全文と結果、git の版の probe の script と出力を足した。cycle 2 の evidence は `docs/evidence/test-2026-11-12-watch-alert-dedupe-cycle2.log`、cycle 1 のものは `-cycle1.log` に残した。wrapper 自身のログは `docs/evidence/verify-2026-11-12-021502.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,304 秒。1 分の load は開始時 5.80、終了直後 5.79 |
| └ Go: `internal/org` | 上位 1,633 本 | ok | 0 | — | 703.4 秒 |
| └ Go: `internal/cli` | 上位 1,079 本 | ok | 0 | — | 504.9 秒 |
| └ Go: `internal/config` | — | ok | 0 | — | 14.8 秒 |
| └ Go: `cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | run-test.sh では `(cached)`。`-count=1` で打ち直して ok | 0 | — | 打ち直しは 0.7、1.9、1.3、2.7、1.3、1.7、2.9 秒 |
| この diff が足したテスト 49 本と期待を変えた 2 本(`-v -count=1`) | org 上位 48 本(subtest 込みの PASS 行 138)、cli 3 本 | 141 | 0 | 0 | 22 秒、4 秒 |
| 同じテストを `-race -count=3` | org の PASS 行 414、cli 9 | 423 | 0 | 0 | 81 秒、13 秒。`DATA RACE` の行は 0 |
| coverage の回(`-run 'Watch\|Inbox\|Scope\|OwnRecord\|Porcelain\|Escalat\|Deadman\|Stall'`、`-coverprofile`) | — | ok | 0 | — | 65 秒 |
| 変異: S10・S11 への 5 個(下の節) | 5 | 5 個とも死んだ | — | — | 1 個 10〜22 秒 |
| 変異: cycle 2 の 29 行の打ち直し | 29 | 死んだ 19、生き残り 8(cycle 2 と同じ 8 個)、build 失敗 2(cycle 2 と同じ C07、C08) | — | — | 1 個 1〜47 秒 |
| 変異: cycle 1 の `watch_scope.go` の死んだ変異 24 個の打ち直し | 24 | 24 個とも死んだ | — | — | 1 個 19〜27 秒 |
| probe(`go test -overlay`、HEAD と S10・S11 を戻した変異の下で) | 2 本(subtest 4 つ) | HEAD ですべて通った。変異の下では、対象の probe の 2 つの subtest だけが落ち、対照は通った | 0 | 0 | 1 回 3 秒 |
| git の版の probe(手元の git 2.49.0 と、docker の `alpine/git:v2.26.2`、`--network none`) | コマンド 6 つ × 2 版 | 下の節 | — | — | — |

上位の本数は `go test -list '.*'` で数えた。org は cycle 2 の 1,632 本から、S10 の新しいテストの 1 本だけ増えた。cli は変わらない。この diff 全体では org に 46 本、cli に 3 本が増え、消えたテストは無い。期待を変えた base のテスト 2 本は cycle 1・2 と同じ(`TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、`TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess`)。S10 と S11 がこの branch のテストで書き換えたのは、`_RunsOnlyTheGitCallsItNeeds` の git の引数の期待 4 か所に `-c diff.relative=false` を足したことと、`_AFailedCommitListingKeepsThePreviousHeadForTheNextCycle` のスタブとスタブ型 `scopeStubGit` の振り分けを `args[0]` から `scopeSubcommand(args)` に替えたことである。前者は期待の文字列を production の新しい引数に合わせたもので、後者は `-c` の組が先頭に付いても今までと同じ subcommand で振り分けるためのもの。どちらも assert を弱めていない。

run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(start と end と rc を meta に書く)を Bash tool の `run_in_background` で起動し、`caffeinate -w <pid>` を perl の `alarm` で 570 秒に区切った前景の待ちで終わりを待った。変異、`-v` の回、`-race` の回、coverage の回、probe は run-test.sh の shell の段と Go の段に重ねて走らせた。そのあいだに読んだ 1 分の load は最大 22.04 だった。run-test.sh は落ちず、打ち直しが要ったテストも無かった。

run-test.sh で `(cached)` になった 7 パッケージは cycle 2 と同じ顔ぶれで、`-count=1` で打ち直してすべて ok だった。

Skipped の 2 行は cycle 1・2 と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。

## 受け入れ条件とテスト

どのテストも run-test.sh と上の `-v` の回で PASS した。「変異」の列の M 番号は cycle 1 の定義(`-cycle1.log`)、C 番号は cycle 2 の定義(下の「cycle 2 の変異」の表)、D 番号はこの cycle の定義である。この cycle で M・C の変異を打ち直した範囲は「変異」の節にある。

| AC | テスト(ファイル) | 変異で確かめたこと |
| --- | --- | --- |
| AC1 件は 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | `TestWatch_Recurrence_AFlappingLeaderStallIsOneAlertItemUntilItIsResolved`(`watch_test.go`)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestInboxAppendRecurred_RecordsOnOpenAndAckedALERTItems`(`inbox_test.go`)、`TestOrgInbox_ShowsTheRecurrences`、`TestOrgInbox_ShowsOneRecurrenceAsOneTime`(`org_inbox_test.go`)、`TestWatch_Recurrence_AnEscalationAlreadyOnARecurredEventRecordsNothingNew` | cycle 1 の M01、M08、M38、M39、M42〜M45、M59。C14、C15、C16 はこの cycle の打ち直しでも死んだ |
| AC2 resolved のあとは新しい件 | flapping のテストの後半、`TestInboxAppendRecurred_RefusesAndWritesNothing`、`TestWatch_Recurrence_ARecurrenceThatIsNotRecordedSendsANewAlert`、`TestWatch_AlertEpisodeLookupsMatchTheKeyExactly`、escalation の打ち直しのテストの resolved の行 | cycle 1 の M02、M03。C14 |
| AC3 続きのあいだは leader に ALERT を送らない | flapping のテスト(ALERT 1 通、watcher の起動 1 回)、`TestWatch_Recurrence_AFiringAfterTheLeaderActedOnTheSeatsPendingAlertIsANewAlert`、`TestWatch_Recurrence_AManifestThatCannotBeReadAtTheFoldSendsANewAlert` | cycle 1 の M01。C01、C03、C06、C08b はこの cycle の打ち直しでも死んだ |
| AC4 deadman の前の再発は pending を増やさず、件に回数が残る | `TestWatch_Recurrence_ASeatsAlertFiringAgainBeforeTheDeadmanCarriesItsCountToTheItem`、期待を変えた `TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、leader が動いたかのテストの「何もしなかった」行 | cycle 1 の M04、M05、M07。C09〜C13 はこの cycle の打ち直しでも死んだ。C24 は生き残った(Test gaps 3) |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | `TestWatch_Recurrence_ANewWatchFindsTheOpenItemOfTheKeyInTheInbox`、`TestWatch_Recurrence_AnEscalationWhoseKeyHasAnItemThatIsNotResolvedIsARecurrenceOfIt`、escalation の打ち直しのテスト | cycle 1 の M06、M08、M09、M10、M52、M53。C14〜C16、C23 |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たない | `TestWatch_LeaderStall_IsNotRaisedWhileItsOrgHasAnItemItsLeaderRaised`、`TestWatch_LeaderStall_IsRaisedWhenItsOrgHasNoItemItsLeaderRaisedOpen` | cycle 1 の M11〜M15。S8〜S11 はこの道を変えていない |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば立つ | `TestWatch_LeaderStall_IsNotRaisedWhileAnotherSeatIsWorking`、`_IsRaisedWhenNoOtherSeatIsWorking`、`_IsRaisedWhenANewSeatsFirstReplySaysWorking` | cycle 1 の M16〜M19。M56 は生き残った(Test gaps 6) |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | `TestWatch_ReservedScope_ChangesInsideTheReservationRaiseNothing`、`_TheOrgsOwnNewPlanAndReportsRaiseNothing`(`watch_scope_test.go`、実際の git)、`TestScopePathInside_FollowsTheReservationRules`、`TestParsePorcelainZ_EntriesInOrderWithBothSidesOfARename`。S10 で `_CommitPathsStayRootRelativeUnderDiffRelative`(`diff.relative=true` と `internal/org` の cwd で、予約の中の commit と自分の新しい計画では立たない) | cycle 1 の M20、M27、M33、M46、M60。D01〜D03 は予約の中の commit で `x.go` の scope_change を立てて死んだ |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | `TestWatch_ReservedScope_AChangeOutsideRaisesScopeChangeNamingItsPaths`(7 つの場合、実際の git)、`_OnlyANewPathOrACommitOutsideRaisesScopeChange`、`_WithNoOriginRefACommitOutsideRaisesScopeChange`、`_AConflictResolvedOutsideInAMergeRaisesScopeChange`、`_ADanglingOriginHEADFallsBackToMainThenToTheDiff`。S10 の新しいテストの最後の段(`diff.relative=true` と下の cwd でも、予約の外の `cmd/y.go` の commit で立つ) | cycle 1 の M21〜M24(M23 は M23b)、M31、M37、M51、M61、C17、C19〜C22 はこの cycle の打ち直しでも死んだ。C18 は git の引数を固定する行でだけ死んだ(Test gaps 7)。外の commit を見落とす向きは probe で確かめた(下の節、Test gaps 12) |
| AC10 予約のない org は今までどおり | `TestWatch_ReservedScope_RunWatchAllReadsGitRunAndAnOrgWithoutOneKeepsGitStatus` と既存の scope_change のテスト | 予約のない org の道を変える変異は当てていない。S10・S11 は予約のない org の `git status --porcelain` を変えていない |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | `TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite` | cycle 1 の M26、M27 |
| AC15 rebase で入った upstream の commit では立たない | `TestWatch_ReservedScope_UpstreamCommitsBroughtInRaiseNothing`(rebase と merge)、`_ARebasedOwnCommitOutsideRaisesScopeChange` と S8 の 2 本 | cycle 1 の M25。C19〜C22 |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | `TestWatch_ReservedScope_RecordsTheBaseHadAndAnUnknownBaseRaiseScopeChange`(8 つの場合)、`TestOwnRecordCandidate_PlansAndReportsButNotTheirTemplates` | cycle 1 の M26、M28b、M29。M30 は生き残った(Test gaps 1) |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | `TestWatch_StatusFile_WrittenBeforeRecurrencesLoadsAndItsPendingAlertTakesARecurrence`、`TestWatch_StatusFile_APendingAlertWithLastRecurredTSLoadsAndDropsIt`、`TestWatch_ReservedScope_AStatusFileWithoutTheScopeBaselineLoads`、`TestInboxRead_LinesWithoutWokenFoldAsBefore` | cycle 1 の M32、M42。C04 は生き残った(Test gaps 4) |
| AC13 help と文書 | `TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope`(help の文を固定する) | 文書の側は /verify と /sync-docs の範囲で、テストは無い |

計画の L1(commit の一覧に失敗したら前の HEAD を据え置く)は `TestWatch_ReservedScope_AFailedCommitListingKeepsThePreviousHeadForTheNextCycle`、`_AFailedCommitListingAsksWhetherGitHasThePreviousHead`、`_APreviousHeadGitDoesNotHaveIsSkippedQuietly` が固定する。S11 でスタブの振り分けが `scopeSubcommand` になったあとも、L1 の変異 M34、M35、C21、C22 はこの 3 本で死んだ。git を打つ回数と引数は `_RunsOnlyTheGitCallsItNeeds` が固定し、S10・S11 で `log` と `diff` の行の先頭に `-c diff.relative=false` が足された。

### cycle 2 の cross-review の 1 件(S10)と self-review の C3-1(S11)とテスト

| 直し | テスト | 直しの前の形を変異で戻したとき |
| --- | --- | --- |
| S10: 利用者の `diff.relative=true` と下のディレクトリの cwd でも、commit のパスを root からの相対で読む(`log` と予備の `diff` の両方) | `TestWatch_ReservedScope_CommitPathsStayRootRelativeUnderDiffRelative`(origin/main がある場合の `log`、origin の ref が無い場合の `diff`) | D02n(`log` から `-c diff.relative=false` を外す): `log` の行だけが「scope_change ALERTs after a commit inside the reservation = [... x.go]; want none」で落ちた。D01n(`diff` から外す): `diff` の行だけが同じ文で落ちた。予約の中の `internal/org/x.go` が cwd からの相対の `x.go` になり、予約の外と判定される |
| S11: `--no-relative` ではなく `-c diff.relative=false` を使い、git 2.28 より古い git でも commit の一覧が失敗しない | 手元にその版の git が無いので Go のテストは無い。`_RunsOnlyTheGitCallsItNeeds` が引数の文字列を固定する | 下の git の版の probe で、git 2.26.2 では `--no-relative` が `log` で rc 128(`fatal: unrecognized argument: --no-relative`)、`diff` で rc 129(usage)になり、`-c diff.relative=false` の形は両方 rc 0 で root からの相対のパスを出すことを確かめた |

## 変異

harness は cycle 1・2 と同じ形で、`mutate.sh` が 1 個ずつ、元の文字列がファイルに 1 回だけ出ることと元と新が違うことを確かめてから置き換え、`go test ./internal/org/ -run <filter> -count=1 -timeout 8m` を打ち、写しの元のファイルに戻して `cmp` で確かめた。変異を当てる前に、filter をすべて合わせた回を変異なしの写しで打ち、ok だった(27.6 秒)。打ち終えたあと、写しの `internal/org/watch_scope.go` は worktree のものと同じだった。変異のあいだ、worktree の `git status --porcelain --ignored` は、run-test.sh が書く `.harness/cache/go-build/` の下を除いて変わらなかった。

### S10・S11 への 5 個

| 変異 | 変えたこと | filter | 結果 |
| --- | --- | --- | --- |
| D01n | 予備の `diff` から `-c diff.relative=false` を外す | 新しいテストだけ | 死んだ(`diff` の行だけ) |
| D02n | `log` から `-c diff.relative=false` を外す | 新しいテストだけ | 死んだ(`log` の行だけ) |
| D01 | D01n と同じ | scope の filter 全体 | 死んだ(新しいテストと `_RunsOnlyTheGitCallsItNeeds`) |
| D02 | D02n と同じ | scope の filter 全体 | 死んだ(同じ 2 本) |
| D03 | 両方から外す(S10 の前の形) | scope の filter 全体 | 死んだ(新しいテストの 2 行と、`_RunsOnlyTheGitCallsItNeeds` の引数の 4 か所) |

依頼にあった「2 つのコマンドそれぞれから外すと新しいテストが落ちる」は D01n と D02n で確かめた。どちらも、落ちたのはその変異が変えたコマンドを使う行だけで、もう一方の行は通った。

### cycle 2 の 29 行と cycle 1 の `watch_scope.go` の 24 個の打ち直し

S10・S11 は `scopeCommittedPaths` の引数と、`watch_scope_test.go` のスタブの振り分けを変えた。前の cycle で死んだ変異の元の文字列がずれていないか、スタブの振り分けを替えたあとも同じテストが殺すかを見るため、cycle 2 の変異の定義 29 行(M30、M48、M50、M56 と C01〜C24、C07b、C08b)と、cycle 1 の変異のうち `watch_scope.go` に当てて死んだ 24 個(M20〜M22、M24〜M27、M28b、M29、M31、M32、M34〜M37、M46、M47、M49、M51、M57、M58、M60、M61、M23b)を、同じ定義と filter のまま HEAD の写しで打ち直した。元の文字列はどれも HEAD でちょうど 1 回出た(`APPLY_FAILED` は 0)。

- cycle 1 の 24 個はすべて死んだ。`--cc` の隣の M23b、`git log` の引数を変える C17〜C19 も、S10・S11 で `-c diff.relative=false` が先頭に付いたあと同じテストで死んだ
- cycle 2 の 29 行は、死んだ 19 個、生き残った 8 個(M30、M48、M50、M56、C02、C04、C07b、C24)、build の通らない 2 個(C07、C08。cycle 2 で C07b、C08b に替えたもの)で、cycle 2 と同じ分かれ方だった

生き残った 8 個のパッケージ全体での打ち直しは、この cycle では打っていない。cycle 2 で `internal/org` と `internal/cli` 全体でも生き残ったことを確かめており、そのあとに増えたテストは S10 の 1 本だけで、それは上の filter(`TestWatch_ReservedScope` を含む)の中に入っているからである。

### probe

HEAD で通り、変異の下で赤になるテストを、`go test -overlay` で `internal/org/` に足して確かめた(コミットしていない。全文は evidence にある)。

- 外の commit の見落とし: 新しいテストと同じ repo(`diff.relative=true`、leader の cwd は `internal/org`、予約は `internal/org/`)で、予約の外の `cmd/y.go` だけを commit する。HEAD では `log` と `diff` の両方の行で scope_change が 1 件立ち、D03(両方から `-c` を外す)の下では両方の行で「scope_change ALERTs = []; want one naming ["cmd/y.go"]」になった。`diff.relative` を設定しない対照の 2 行は、どちらでも通った。新しいテストはこの向きを最後の段で assert するが、変異の下では前の段(予約の中の commit)の `t.Fatalf` で止まるので、この向きの赤は probe で確かめた(Test gaps 12)

### git の版の probe

S11 の理由(`--no-relative` は git 2.28 から。それより古い git では commit の一覧が失敗し続ける)を確かめるため、`internal/org/x.go` と `cmd/y.go` を変える commit を作り、`diff.relative=true` を設定して `internal/org` から、本番と同じ引数の `log` と `diff` を、`-c diff.relative=false` を付けた形、`--no-relative` を付けた形、何も付けない形で打った。

| git | `-c diff.relative=false` の `log` と `diff` | `--no-relative` の `log` と `diff` | 何も付けない `log` と `diff` |
| --- | --- | --- | --- |
| 2.49.0(手元) | rc 0、`cmd/y.go` と `internal/org/x.go` | rc 0、同じ 2 つ | rc 0、`x.go` だけ(cwd からの相対で、cwd の外が落ちる。cross-review の指摘の形) |
| 2.26.2(docker の `alpine/git:v2.26.2`) | rc 0、`cmd/y.go` と `internal/org/x.go` | `log` は rc 128(`fatal: unrecognized argument: --no-relative`)、`diff` は rc 129(usage) | rc 0、`cmd/y.go` と `internal/org/x.go`(2.26 は `diff.relative` を読まない) |

2.28 ちょうどの git は手元でも docker でも用意できず、打っていない(Test gaps 13)。

## Coverage

- Statement: coverage の回は filter を付けたので、パッケージ全体の数字(40.1%)は意味を持たない。この diff が足したり変えたりした production の行(`git diff -U0 028a8f6d..HEAD` の行と profile の block を突き合わせた)にかかる block は 340 statement 中 329(96.8%)で、cycle 2 と同じ。cycle 2 のあと(`fe4eff60..HEAD`)の行では 3 statement 中 3
- Branch: 測っていない(Go の標準の道具は statement だけ)
- Function: `watch_scope.go` の `scopeCommittedPaths` と `scopeDefaultBranchRef` は 100%。ほかは cycle 2 と同じで、`watch_recurrence.go` は `escalateAsRecurrence` の 85.7% を除いて 100%、`watch.go` は `checkDeadman`、`raiseOrClear`、`leaderIsWaiting`、`orgHasOpenLeaderItem`、`seatBusy`、`evaluateSeat` が 100%、`evaluateCycle` 97.0%、`escalateToInbox` 93.8%、`leaderActedSinceAlert` 88.9%、`watch_scope.go` は `scopeRecordsAtBase` 76.9%、`dropOwnRecords` 92.9%、`evaluateReservedScope` 95.0%、`runGitBounded` 92.3% で、ほかは 100%
- Notes: 打たれていない 11 statement は cycle 2 と同じ場所である。S10・S11 の doc comment の行が増えたので、`watch_scope.go` の行番号は 6 つずつずれた(cycle 2 の 319、323、327、359、413、417 は HEAD の 325、329、333、365、419、423)。中身は `watch.go:1893`(新しい件を記録したあと pending の回数を移せなかったときの stderr の行)、`watch_recurrence.go:206`(`escalateAsRecurrence` の追記が resolved 以外の理由で失敗したときの行)、`watch_scope.go:325`、`:329`、`:333`(merge-base と ls-tree の失敗、M30 の道)、`:365`(同じ失敗を受けて何も除かない枝)、`:419`(`gitRun` が nil のときの既定)、`:423`(porcelain が読めない周期)、`:51`(`ErrWaitDelay` を成功とみなす枝)、`watch.go:424`(`realGitStatus`。filter の外の `TestRealGitStatus_IsCutOffAtTheDriverCallTimeout` が打つ)、`watch.go:1766`(`leaderActedSinceAlert` の herdr の `agent get` の文字列が基準と違うときの `return true`、C24)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、`(cached)` の 7 パッケージの打ち直し、この diff のテストの `-v` の回、`-race` の回、coverage の回、probe の HEAD の回で、落ちたテストも、打ち直しが要ったテストも無かった。flaky と分類したものも無い | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #307: 同じ条件の ALERT が立つたびに受信箱の件が 1 つ増える | 直っている | flapping のテスト(件 1、`recurred` 4、ALERT 1 通)。cycle 1 で、修正前の 2 つの道を外す変異(M01 + M06)では 5 回で 5 件になることを確かめた |
| issue #307: 待っているだけの leader の stall | 直っている | AC6 と AC7 のテスト |
| issue #307: 予約の中だけの変更の scope_change | 直っている | AC8 のテスト。AC15 で rebase と merge の誤報も固定されている |
| cross-review cycle 1 の 4 件(S8) | 直っている | 下の「cycle 2 の記録」の S8 の表。この cycle の打ち直しでも、C03、C14、C17、C20 は専用のテストで死んだ |
| cross-review cycle 2 の 1 件(`diff.relative=true` と下の cwd で予約の判定を両向きに誤る) | 直っている | 新しいテストが予約の中の commit の誤報を、probe が予約の外の commit の見落としを、S10 の前の形に戻す D03 の下で赤にする |
| self-review cycle 3 の C3-1(`--no-relative` が古い git で失敗する) | 直っている | git 2.26.2 の probe で `-c diff.relative=false` の形が rc 0 で root からの相対のパスを出し、`--no-relative` の形は rc 128 と 129 で失敗した |
| 既存の deadman(leader の活動の 3 つの情報源と、probe の「使えない」印の扱い) | 変わっていない | 既存の `TestWatch_Deadman_*` は期待を変えずに通り、C09〜C13 はこの cycle の打ち直しでも死んだ。herdr の `agent get` の情報源だけは前から固定されていない(Test gaps 3) |
| 既存の liveness、watch の立て直し(同じ alert_id の件を二重に作らない)、予約のない org の scope_change | 変わっていない | `internal/org` と `internal/cli` 全体が ok。期待を変えた既存のテストは cycle 1 の 2 本だけ |
| 古い watch-status と受信箱 | 読める | AC12 のテスト |

## Test gaps

1〜11 は cycle 2 の report(commit `fe4eff60`)の番号のまま残した。tech-debt の 462 行はこの番号で M30、M50、M48 を引く。12 と 13 はこの cycle で足した。

1. M30(LOW、cycle 1 から): origin の既定のブランチはあるが merge-base が読めないとき(共通の祖先が無い、git が期限で切られた)に、何も除かない枝が固定されていない。変異の下では、自分の記録の候補をすべて除き、予約の外の変更を知らせない側に倒れる。cycle 1 の probe の形の 1 本で死ぬ。この cycle の打ち直しでも生き残った
2. M50(LOW、cycle 1 から): HEAD が読めない周期に前の HEAD を据え置く条件(`head != ""`)が固定されていない。変異の下では、そのあいだの予約の外の commit が知らされない。cycle 1 の probe の形の 1 本で死ぬ。この cycle の打ち直しでも生き残った
3. C24(LOW、この diff の前からある穴): leader の活動の情報源のうち、herdr の `agent get` の文字列が基準と違えば活動とみなす `return true`(`watch.go:1766`)を、org と cli のどのテストも固定していない。S8 でこの判定は `foldIntoPendingAlert` も使うようになったので、その情報源だけで leader が動いた場合に、再発が新しい ALERT になるか続きに畳まれるかも固定されていない。cycle 2 の probe の形の 1 本(leader の文字列を 1 回変え、seat の pending が消えること)で死ぬ。この cycle の打ち直しでも生き残った
4. C04(LOW): 1 つの鍵に pending を 2 つ持つ status file(この計画の前のビルドが書いたもの)で、古いほうが答えられたあと新しいほうを試す道が固定されていない。変異の下では、まだ答えられていない pending があるのに新しい ALERT を送る。cycle 2 の probe の形の 1 本で死ぬ。この cycle の打ち直しでも生き残った
5. C02(INFO): leader についての pending には leader が動いたかの判定をかけない分岐が固定されていない。leader についての pending は送った周期の deadman の掃除で escalate されて消えるので、畳むときに残っているのは、その周期が掃除の前に終わったうえで leader の条件がいったん消えてまた立った場合に限られる
6. M48(INFO、cycle 1 から): `scopeDefaultBranchRef` が `symbolic-ref` の答えを `refs/remotes/origin/` の下のものに限る検査が固定されていない。cycle 2 で、origin/HEAD を作業中のブランチに向ける probe で死ぬことを確かめた。M56(INFO、等価)は、leader は org に 1 席なので受信箱を周期に 1 回だけ読む印の違いが外から見えない
7. C18(INFO): `--cc` を `-c` にした変異は、git の引数を文字列で固定する行でだけ死んだ。`-c` と `--cc` で出るパスが違う merge はテストに無い
8. C07b(INFO、等価): manifest を一度だけ読む印を無視しても、同じ周期にもう一度読むだけで結果は変わらない
9. 打たれていない失敗の道(Coverage の Notes): 新しい件を記録したあと pending の回数を移せなかったときの行(`watch.go:1893`)は stderr の 1 行だけを書く
10. Test plan の Integration の項(`watch_all_test.go` の流れで件数を照らす)は、その形のテストになっていない。新しいテストは `evaluateCycle` と `escalateToInbox` を直に打つ。INFO
11. 実機の道: 新しい binary で機能の org を走らせて受信箱の件を数え直すことはしていない。テストは偽の herdr と一時 repo の実際の git だけで確かめている。S9 の記録にある「再発を足した直後に watch が止まると、立て直した watch が回数を 1 多く数える」穴にはテストが無い。cycle 2 の /sync-docs で、この穴は tech-debt の 462 行の (7) に、C24 と C04 は (8) に入った
12. 新しいテストの 2 つの向き(INFO、この cycle から): `TestWatch_ReservedScope_CommitPathsStayRootRelativeUnderDiffRelative` は予約の中の commit を先に見て `t.Fatalf` で止まるので、S10 の前の形では、予約の外の commit を見落とす向きの assert まで進まない。この向きは上の probe で確かめた。2 つの向きは同じ 1 つの git の設定から出るので、片方だけを壊す変異は本番の形では考えにくく、INFO とした
13. git の版(INFO、この cycle から): Go のテストは手元の git 2.49.0 だけで走る。S11 の理由は git 2.26.2 と 2.49.0 の probe で確かめたが、`diff.relative` と `--no-relative` が入った 2.28 ちょうどの git は打っていない。2.28 より古い git では `diff.relative` 自体が無いので、新しいテストはそこでは変異の有無にかかわらず通る

## Verdict

- Pass: 8(8 種類の実行がすべて通った。`./scripts/run-test.sh` は rc 0 で、shell 45 本と Go 10 パッケージが通り、`(cached)` だった 7 パッケージも `-count=1` で通った。この diff のテスト 49 本と期待を変えた 2 本の `-v` の回、その `-race -count=3` の回(`DATA RACE` 0)、coverage の回、probe の HEAD の回も 0 failed。git の版の probe は、S11 の形が 2.26.2 と 2.49.0 の両方で root からの相対のパスを出すことを示した。AC13 を除く 14 個の AC と AC13 の help の文には、どれも PASS するテストがあり、名前は上の表にある。cross-review cycle 2 の 1 件の直し(S10)は、`log` と `diff` のそれぞれから `-c diff.relative=false` を外す変異で新しいテストの対応する行が赤になり、外の commit の見落としの向きも probe で赤になる。S10・S11 への変異 5 個はすべて死に、前の cycle で死んだ変異 43 個(cycle 2 の 19 個と cycle 1 の `watch_scope.go` の 24 個)も HEAD でまた死んだ。生き残った 8 個は cycle 2 と同じ LOW か INFO の穴(Test gaps 1〜8)である。変えた production の行の coverage は 340 statement 中 329、cycle 2 のあとの行では 3 statement 中 3)
- Fail: 0
- Blocked: 0。Test gaps 1〜4 のテストを足すか、tech-debt の行に残すかは orchestrator が決める

## cycle 2 の記録

cycle 2(HEAD `0a957745`、report のコミット `fe4eff60`)は pass だった。run-test.sh は rc 0 で 1,343 秒(org 722.1 秒、cli 536.0 秒)、この diff のテスト 48 本と期待を変えた 2 本の `-v` と `-race -count=3` も通った。S8・S9 への変異 23 個のうち 19 個が死に、cycle 1 からの 4 個と合わせた 8 個は `internal/org`(716.0 秒)と `internal/cli`(429.4 秒)全体を同時に打っても生き残った。cycle 1 で死んだ 57 個も HEAD で死んだ。gap の probe は M30、M50、M48、C04、C24 の 5 本。cycle 2 の report の全文は `git show fe4eff60:docs/reports/test-2026-11-12-watch-alert-dedupe.md` で読める。C 番号の定義は下の 2 つの表のとおりである。

### cycle 1 の cross-review の 4 件(S8)とテスト(cycle 2)

| S8 の直し | テスト | 直しの前の形を変異で戻したとき |
| --- | --- | --- |
| (1) leader が動いた周期の再発を古い pending に畳まず、新しい ALERT にする | `TestWatch_Recurrence_AFiringAfterTheLeaderActedOnTheSeatsPendingAlertIsANewAlert`(manifest の `sent`、agmsg の履歴、何もしない、の 3 行)、`TestWatch_Recurrence_AManifestThatCannotBeReadAtTheFoldSendsANewAlert` | C03(判定を外し、S8 の前と同じく必ず畳む): `sent` と履歴の行で「1 ALERTs to the leader; want 2」「pending alerts = map[]」 |
| (2) `git log` に `--cc` を足し、merge 自身の編集を数える | `TestWatch_ReservedScope_AConflictResolvedOutsideInAMergeRaisesScopeChange`(実際の git で衝突を予約の外で解く) | C17(`--cc` を外す): 「scope_change ALERTs = []; want one naming ["other/x.go"]」 |
| (3) 指す先の無い origin/HEAD を飛ばして main・master・`git diff` に落とす | `TestWatch_ReservedScope_ADanglingOriginHEADFallsBackToMainThenToTheDiff`(origin/main がある場合と無い場合) | C20(指す先を確かめない): 2 つの行とも「scope_change ALERTs = []」 |
| (4) `recurred` の event に載った alert_id の escalation の打ち直しで何も足さない | `TestWatch_Recurrence_AnEscalationAlreadyOnARecurredEventRecordsNothingNew`(件が open と resolved) | C14(この確かめを外す): open の行では同じ件に `recurred` がもう 1 つ足され、resolved の行では新しい件 `e2` ができた |

### cycle 2 の変異(S8・S9 の変更への 23 個)

| 変異 | 変えたこと | 結果(cycle 2。cycle 3 の打ち直しでも同じ) |
| --- | --- | --- |
| C01 | `foldIntoPendingAlert` で leader が動いたかを見ない(`false &&`) | 死んだ |
| C02 | leader についての pending にも leader が動いたかの判定をかける(`if true`) | 生き残った(Test gaps 5) |
| C03 | どの pending にも判定をかけない(`if false`、S8 の前の形) | 死んだ |
| C04 | 答えられた pending を消したあと、同じ鍵の次の pending を試さずに抜ける(`continue` を `return false` に) | 生き残った(Test gaps 4) |
| C06 | manifest が読めないとき畳む側に倒す(`return true`) | 死んだ |
| C07b | manifest を一度だけ読む印を無視する(`!read \|\| true`) | 生き残った(等価、Test gaps 8) |
| C08b | 読んだ manifest の event を捨てる(`rr.Events[:0]`) | 死んだ |
| C09、C12 | `leaderActedSinceAlert` の `>` を `>=` に(manifest の数、履歴の行数) | 死んだ |
| C10、C11 | probe の基準が「使えない」印のときも比べる(herdr の `agent get`、履歴の行数) | 死んだ |
| C13 | checkDeadman が `leaderActedSinceAlert` を呼ばない | 死んだ |
| C24 | herdr の `agent get` の文字列が基準と違っても活動とみなさない | 生き残った(Test gaps 3) |
| C14 | `escalateToInbox` で `recurred` に載った alert_id を見ない | 死んだ |
| C15、C16 | `alertItemWithRecurredID` の event の種類の条件、alert_id の条件を外す | 死んだ |
| C23 | `pendingAlertForKey` が alert_id を返さない | 死んだ |
| C17 | `git log` から `--cc` を外す | 死んだ |
| C18 | `--cc` を `-c` に | git の呼び出しのテストの引数の行でだけ死んだ(Test gaps 7) |
| C19 | `--cc` を `-m` に | 死んだ |
| C20 | origin/HEAD の指す先を確かめない | 死んだ |
| C21 | `namesCommit` がいつも真 | 死んだ |
| C22 | main・master を確かめずに使う | 死んだ |

C07(`if !read {` を `if true {`)と C08(`events = rr.Events` を `nil`)は変数が使われなくなって build が通らなかったので、同じ意味の C07b と C08b に替えた。cycle 2 の gap の probe は、M30(origin/main を HEAD と共通の祖先の無い root commit に向ける)、M50(1 周期だけ `rev-parse --verify -q HEAD` を失敗させる)、M48(origin/HEAD を作業中のブランチに向ける)、C04(1 つの鍵に pending を 2 つ持つ古い status file)、C24(leader の `agent get` の文字列を 1 回変える)の 5 本で、どれも HEAD で通り、対象の変異の下でだけ落ちた。

## cycle 1 の記録

cycle 1(HEAD `b8058948`、report のコミット `3506f99a`)は pass だった。run-test.sh は rc 0 で 1,623 秒(org 910.6 秒、cli 673.0 秒)、新しいテストの `-v` と `-race -count=3` も通った。変異 61 個のうち 57 個が死に、M30(LOW)、M50(LOW)、M48(INFO)、M56(INFO、等価)の 4 個が org と cli 全体でも生き残った。4 個とも cycle 3 の HEAD でまだ生き残っている。cycle 1 の report の全文は `git show 3506f99a:docs/reports/test-2026-11-12-watch-alert-dedupe.md` で読める。
