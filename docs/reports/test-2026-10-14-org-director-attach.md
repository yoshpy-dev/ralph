# Test report: org-director-attach

- Date: 2026-10-15(JST 04:08〜04:45)
- Plan: docs/plans/active/2026-10-14-org-director-attach.md(`Status: Approved`、Approved sha256:27414f27525b)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD a5f5353b(base ef575748、15 コミット、54 ファイル、+7,177/-693)。/verify が見た 676e7f98 から増えたのは verify の報告(7cd73066)と insight event(a5f5353b)だけで、コードとテストは同じ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0。開始時の load average は 222(別のセッションの iOS simulator)で、下の実行の間は 5〜15
- Evidence: `docs/evidence/test-2026-10-14-org-director-attach.md`(1. run-test.sh の抜粋、2. Go の結果と数・`-race`・canary・coverage、3. HEAD の binary の実機確認の script と出力、4. mutation の一覧と probe)。wrapper が書いた `docs/evidence/verify-2026-10-14-190833.log` は gitignore の対象なのでコミットしない
- テストもコードも変えていない。mutant と probe のテストは scratchpad に置き、`go test -overlay` で読ませた。worktree に書いたのは、この報告と上の evidence だけ

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`scripts/ralph-config.sh` の変更で full に広がった。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 677 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test -count=1 -v -coverprofile ./internal/...` | cli: top-level 678、subtest 599。org: 746、1,523。org/driver: 48、89。org/protocol: 15、56。config: 37、34。insights: 36、7。scaffold: 30、0。upgrade: 114、13 | 同じ数から skip を引いた数 | 0 | org 1、scaffold 2 | 215 秒(cli 209.5 s、org 97.0 s)、rc 0 |
| `go test -count=1 -v ./internal/...`(go1.26.0) | go1.25.8 と同じ | go1.25.8 と同じ | 0 | org 1、scaffold 2 | 211 秒(cli 210.8 s、org 84.6 s)、rc 0 |
| うちこの差分で足したテスト 70 本 | top-level 70、直下の subtest 99 | 両方の版ですべて | 0 | 0 | - |
| `go test -race`(org は `-run 'Director\|Escalate\|Inbox\|Approve\|Deadline\|NotifyInboxItem'`、cli は `-run 'OrgDirector\|OrgEscalate\|OrgInbox\|OrgApprove\|StatusCmd\|PrintApprovedPlans'`) | 2 パッケージ | 2 | 0 | - | 35 秒、DATA RACE なし |
| mutation(`go test -overlay`、go1.26.0) | mutant 42 件と対照 2 件 | red 41 件(A2 は build できず、A2b に書き直して red) | 生き残り 1 件(D5) | - | 493 秒 |
| probe のテスト(overlay だけ、コミットしない) | 2 本 | HEAD では両方通る(1 本は `-race` でも) | 狙った mutant(D5、A1)のもとで落ちる | - | - |
| HEAD の binary の実機確認(scratch の repo、下の「実機の確認」) | 4 段と 3 件 | すべて期待どおり | 0 | - | 約 2 分 13 秒(3 件は数秒) |

- shell は 41 本すべてが `OK` で終わり、集計の `FAIL:` の行はどれも 0。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell のテストを変えていない。行番号と時間を除くと、suite の行は org-merge-guarantee の 2 回目(`test-2026-10-14-org-merge-guarantee-c2-run-test.txt`)と同じで、違うのは scope の行(今回は `full fallback (unclassified:scripts/ralph-config.sh)`)だけ
- `run-test.sh` の中の Go は cli(195.4 s)、insights、org(87.1 s)を流し直し、config、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。この 5 つは上の 2 回の `-count=1` の実行で ok を確かめた
- go1.25.8 と go1.26.0 で、テストの名前と結果の行(時間を除いて並べた 4,028 行)は同じだった
- Skipped は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(org。darwin で飛ばす作り)と scaffold の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`。どれもこの差分と関係ない
- 差分で足した `func Test` は 70 本、消した `func Test` は 7 本。消した 7 本は、approve が件を resolve することを確かめていた 7a のテスト 6 本と、status の行の順のテストの旧名 1 本で、どれも足した 70 本の中に置き換えがある(計画の D7 と Progress checklist の「計画からのずれ (3)」のとおり)
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どの実行でも落ちなかった。go1.26.0 の `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は 7.53 s で、前から記録している +5 s の場合に当たる(15 s の予算の中)
- AC9: Go の 2 回の実行、`-race`、mutate.py の 44 回は、PATH の先頭に canary の `osascript` と `herdr` を置いて走らせた(呼ばれたら記録する。`herdr --version` だけは本物に渡す)。呼ばれた回数は 0 回。手で同じ PATH から打つと 3 回とも記録されたので、PATH の検索は canary に届く。テストのあと、main のチェックアウトにも worktree にも `.harness/state/org` はなく、repo の下に `director.jsonl` は 1 つもない。attach と escalate は台帳のディレクトリを作るので、repo の台帳に書くテストがあればディレクトリが残る。この確認は空振りにならない

## 実機の確認(HEAD の binary)

HEAD を go1.25.8 で build した binary を、scratch の repo で打った。環境は `env -i` で、PATH には git と ralph と偽の `osascript`(引数を `osascript.log` に書いて exit 0。人への経路の代わり)だけを置いたので、herdr、agmsg、本物の osascript は動かない。main worktree の `ralph.toml` は `[org.watchdog].interval_seconds = 5`、`[org.director]` の `enabled = true`、`ack_minutes = 1`、`resolve_minutes = 2`。見張りは `ralph org watch --all --detach` で立て、`--stop` で止めた。終わったあと `pgrep` で ralph の process は残っていない。evidence の 3 に script と出力がある。

1. 見張りなし(P1): `ralph org director attach --driver claude` は stdout に `attached d1 (driver=claude)` と `detach it with: ralph org director detach --id d1` の 2 行、stderr に警告(`(none runs)`、`ralph org watch --all --detach` の打ち方)を出し、exit 0。`ralph status` は `org watch: not running` の次に `director: attached d1 (driver=claude, attached at ..., by $USER) -- the registration does not tell whether ...` を出し、`--json` の `director` は `state: attached` と登録の欄を持つ。`ralph org director status` と `--json` も同じ登録を出す。この状態の escalate(e1)は `delivery: now`、理由 `inbox_no_watch` ですぐ送られ、偽の osascript が 1 回呼ばれた
2. 見張りあり(P2、Lead の優先項目): 19:11:38 に d1 が attached のまま escalate した e2 は、stderr に `ORG ESCALATION RECORDED ... left in ... for the director` を出して exit 0。`escalations.jsonl` は増えず、osascript も呼ばれず、`ralph org inbox` の NOTIFIED は `deferred`、`inbox show e2` は `notified: deferred (left to the director, or to the inbox deadlines of ralph org watch --all; to send it to the human now: ralph org inbox notify e2)` を出した。5 秒ごとに読むと、19:12:36(t+57 s)は `deferred`、19:12:41(t+63 s)には送られていた。inbox.jsonl の `notified` は 19:12:38、理由 `inbox_ack_deadline`(escalate からちょうど 60 秒)。`escalations.jsonl` に同じ理由の行が 1 つ、osascript に `org org-a: e2 (QUESTION) not acked within 1 min`(title `ralph org watch`)、watch.log に `ORG INBOX DEADLINE` の行が出た。director を外したあと(登録なし)に escalate した e3 も同じく deferred になり、19:12:43 に `inbox_ack_deadline` で送られた(前の sweep の 5 秒後)。登録の有無で動きは変わらなかった
3. 同じ P2 の続き: すぐ ack した e4 は ack の期限では送られず(NOTIFIED は `deferred` のまま)、19:13:44 に resolve の期限 `inbox_resolve_deadline` で送られた。e1、e2、e3 も、escalate から 2 分を過ぎた最初の sweep(19:13:38 と 19:13:44)で resolve の期限の通知を 1 回ずつ受けた。approve のあと send の前に止まった acked の件が進む経路と同じ形で、件が期限から外れないことを実際の時計で見た
4. 入れ替えと detach(P2 の中): USER なし、`HERDR_PANE_ID` ありで B が `--replace` すると `attached d2 (driver=codex), replacing d1 (...)`。d1 の `detached` と d2 の `attached` は同じ秒の 2 行で、`user` は空、`herdr_pane` は true(値は書かれない)。A の `detach --id d1` は exit 1 で「the current registration is d2 ... does not need to detach」と言い、d2 は残った。`ralph org director status` は `user: -`、`herdr_pane: yes`、status の行は `by` を出さない。`detach`(`--id` なし)、`detach --id d2 --force`、`attach --org-id org-a` はどれも exit 1。`detach --id d2` は通り、もう一度打つと `no director is attached` で exit 1。そのあと `director status --json` は `state: not_attached`
5. 見張りを止めたあと(P3): `ralph status` は `org watch: not running (stopped at ...)`。escalate(e5)はまたすぐ送られ、理由は `inbox_no_watch`
6. main worktree の規則(P4、AC5): linked worktree の `ralph.toml` だけ `enabled = false` にしても、そこで打った attach は通り(d3)、escalate(e6)の理由は `inbox_no_watch`(`inbox_director_disabled` ではない)。逆に main を false、linked worktree を true にすると、linked worktree の attach は「`[org.director].enabled is false in the main worktree's .../repo/ralph.toml`」で exit 1。`ralph status`(source `git-main-worktree`)、`ralph status --state-dir <台帳>`(source `flag`。`git worktree list --porcelain` を打つ分岐)、その `--json`、`director status` は、どれも disabled を出した。verify と self-review が未確認とした分岐を binary で通した
7. `enabled = false` で見張りが動いているとき(P4c): escalate(e7)はすぐ送られ、理由は `inbox_director_disabled`
8. 追加の 3 件(evidence の 3.3): 環境変数 `RALPH_ORG_DIRECTOR_ENABLED` は true でも false でも `ralph.toml` の値を変えない(`scripts/ralph-config.sh` のコメントどおり)。d7 を含む壊れた行のある `director.jsonl` では attach が d8 を出し、`ralph status` は stderr に `1 unreadable director line(s) skipped` を出す。`director.jsonl` を mode 000 にすると、`ralph status` は `director: unreadable (... permission denied)`、`--json` と `director status --json` は `state: unreadable` と `error` を出し、escalate は今までどおり記録して送る

## Coverage

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.5%、`internal/cli` 87.3%、`internal/config` 93.3%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%(org-merge-guarantee の 1 回目は org 94.3%、cli 86.9%)
- 新しいファイルと変えたファイル: `internal/org/director.go` 129/135 文(95.6%)、`internal/cli/org_director.go` 135/138 文(97.8%)、`internal/org/deadline.go` 62/62 文(100%)、`internal/cli/org_approve.go` 20/20 文(100%)、`internal/cli/org_inbox.go` 98.3%、`internal/org/inbox.go` 96.6%、`internal/cli/status.go` 96.2%、`internal/org/approve.go` 94.1%、`internal/org/escalate.go` 92.1%
  - `director.go` で通らない 6 文は、plain な attach と detach の追記の失敗(231、287-289)、event の marshal の失敗(337-339)、write と close の失敗(346-352)。`--replace` の追記の失敗は `TestDirectorAttach_AReplaceThatCannotWriteRecordsNothing` が通す
  - `org_director.go` で通らない 3 文は、store が `ErrDirectorAttached` 以外のエラーを返したとき(121)、`filepath.Abs` が失敗したときの `./ralph.toml`(319)、`os.Getwd` の失敗(334-336)
  - ほかのファイルで通らない文は、この差分の前からある行(I/O のエラー、`o.Inbox == nil`)
- Function: `director.go` の 18 関数のうち 15 が 100%(`Attach` 96.4%、`Detach` 96.4%、`appendLocked` 76.5%)。`org_director.go` の 14 関数のうち 11 が 100%(`newOrgDirectorAttachCmd` 96.6%、`orgDirectorEnabledFile` 85.7%、`orgDirectorCaller` 75.0%)。ほかのファイルの変えた関数では `escalateDelivery`、`inboxDeferredBanner`、`checkNotApprovedYet`、`ApprovalResolveCommand`、`checkResolvedMeanwhile`、`InboxDeadlineDue`、`withMainWorktreeDirectorEnabled`、`loadMainWorktreeOrgConfig`、`statusDirectorEnabled`、`wireOrgEscalateWatch`、`inboxNotifiedColumn`、`inboxItemDeferred`、`printApprovedPlans` が 100%
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

## Test plan coverage map

計画の Test plan の項目ごとに、それを確かめるテストを挙げる。「mutant」の列は、その項目の判定を壊した mutant が落ちたかどうか(id は evidence の 4.1)。

| Test plan の項目 | テスト | mutant |
| --- | --- | --- |
| config: `enabled` の既定、`[org.director]` がないとき、`enabled = false`、lock-step | `TestDefault_OrgDirector`、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`、`TestLoad_OrgDirectorRoundTrip`、`TestDefaultsLockStep` | CF1 red |
| director store: attach、detach、`--replace`、二重の attach と登録のない detach と `id` の違う detach の拒否(A → B → A)、`--force`、`id` の割り当て、壊れた行、切れた最後の行、同時の attach、畳み込みの順 | `director_test.go` の 24 本(`TestDirectorAttach_*`、`TestDirectorDetach_*`、`TestDirectorRead_*`、`TestDirectorLock_IsItsOwnLockAndSerializesWrites`、同時の attach と replace の 2 本) | D1〜D4、D6〜D9 red。D5 は生き残り(下) |
| escalate: 見張りが受け付けている、not_running・stale・stopping・`--once`、`enabled = false`、attached でも同じ、通知の失敗の扱い | `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch`(9 つの場合 × attached あり・なし)、`TestEscalate_DeferredBanner`、`TestEscalate_RefusalProbesNoWatch`、`TestEscalate_SendAtOnceFailureIsHandledAlikeForEitherReason`、`TestNotifyInboxItem_SendsADeferredItemNow` | E1〜E8 red |
| inbox: `delivery` の読み書き、`delivery` のない古い行 | `TestInboxEscalate_WritesAndFoldsTheDelivery`、`TestInboxEscalate_RefusesAnUnknownDelivery`、`TestInboxRead_FoldsEventsIntoItemsInIDOrder` | I1、I2 red |
| deadline: 記録だけの件が ack の期限で `inbox_ack_deadline`(偽の時計)、`inbox_manual_notify` の件に ack の期限が効かない、`delivery` が結果を変えない | `TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline`、`TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline`、`TestInboxDeadlineDue`(3 つの delivery を同じ期待値で回す行)。実際の時計では上の「実機の確認」の 2 と 3 | DL1、DL2 red |
| approve: resolve しない、記録済みの件の拒否、送る打ち方と resolve の打ち方の順、止まった件が期限で人に届く | org: `TestOrgApprove_RecordsAndLeavesTheItemAndThePlan`(inbox のファイルが byte で同じ)、`_RefusesAnItemAlreadyApproved`、`_AnApproveThatStopsBeforeTheSendReachesAHumanAtADeadline`、`_TakesNoInboxLock`、`_ResolvedBySomeoneElse`。CLI: `TestOrgApprove_RecordsAndPrintsTheSendThenTheResolve`、`_ASecondApproveOfTheItemIsRefused`、`_TheDecisionAndResolveCommandsRunInAPOSIXShell`、`TestPrintApprovedPlans_NoResolveCommandForAnItemResolvedElsewhere` | A1、A2b、A3、A4、C6 red |
| Integration: `ralph org director attach\|detach\|status`(一時の台帳、`--json`、`--org-id` の拒否、`enabled = false`) | `TestOrgDirector_*` の 14 本、`TestOrgDirectorHelp_DescribesTheRegistration` | C2、C3、C10、C11、C16 red |
| Integration: `ralph org escalate`(偽の見張りの印、main と feature の `ralph.toml`) | `TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly`、`_ProbesTheLedgersWatch`(本物の `ProbeWatch` と持ったロック)、`_DirectorEnabledFromTheMainWorktreeRalphToml`(8 つの場合)、`_MainWorktreeRalphTomlLoadError_AlertsTheHumanAndExitsOne` | C1、C4、C5、C7 red |
| Integration: `ralph status` の `director:` の行と JSON | `TestStatusCmd_DirectorLinePerState`(7 つの状態 × org の有無)、`_DirectorEnabledFromTheMainWorktree`、`_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`、`_DirectorDamageWarnsOnStderr`、`_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines` | C8、C9、C12〜C15 red |
| Regression: org、cli、config の全体を go1.25.8 と go1.26 で。今の escalate のテストは理由の文字列のほかは変わらない | 上の 2 回の全体の実行(どちらも rc 0、名前も同じ) | - |
| Edge: `director.jsonl` が読めない | `TestOrgDirector_StatusOfAnUnreadableFile`、`TestDirectorStore_ReportsAnUnreadableFileAndAnUnavailableLock`、実機の 8 | - |
| Edge: `USER` がない | `TestStatusCmd_DirectorLinePerState` の「attached with a pane and no user」、実機の 4 | C15 red |
| Edge: `HERDR_PANE_ID` がある | `TestOrgDirector_AttachReplaceAndDetachByID`(値を書かないことも見る)、実機の 4 | C16 red |

## Mutation

- 対象は director の store(D1〜D9)、escalate の分け方と記録(E1〜E8)、inbox の `delivery`(I1、I2)、期限(DL1、DL2)、approve(A1〜A4)、CLI の director・escalate・inbox・approve・status(C1〜C16)、config の既定(CF1)。ファイルを overlay で差し替え、パッケージのテストを `-run` で絞って流した。対照の X0 と X0c(手を加えないファイルを overlay で読ませたもの)は通った
- 42 件のうち 41 件が red。A2 は `declared and not used: j` で build できなかったので、条件の末尾に `|| true` を足した A2b に書き直し、`TestOrgApprove_ResolvedBySomeoneElse` で落ちた
- 生き残りは D5 の 1 件。D5 は `director.go` の畳み込みで、今の登録と違う `id` の `detached` でも今の登録を終わらせる。org の全体(top-level 746 本)を流しても生き残った。`TestDirectorRead_SkipsAndCountsLinesItCannotUse` には d1 が今の登録のときの `detached d2` があるが、D5 のもとではこの行が d1 を終わらせ、あとの `detached d1` のほうが無視される側に回る。そのため今の登録(d4)も無視した数(13)も変わらず、2 つの行が打ち消し合う。`attached d1` と `detached d2` だけのファイルを読む probe(PD)は、HEAD で通り、D5 のもとで落ちた
  - 実際の入力への影響は小さい。store が書く `detached` は、ロックの中で読んだ今の登録の `id` のものだけなので、違う `id` の `detached` は手で書き換えたときにしか現れない(`Attach` と `Detach` のコードを読んで確かめた)
- self-review と verify が未確認とした「同じ件への approve の同時実行」には、2 つの goroutine が同じ件を同時に approve する probe(PA、30 回)を書いた。HEAD では毎回 1 つだけが記録し、もう 1 つは `ErrPlanAlreadyApproved` で断られた(`-race` でも同じ)。`checkNotApprovedYet` を外した A1 のもとでは 1 回目から 2 つとも記録して落ちたので、この probe は二重の記録を見分ける。2 つの goroutine は同じ `*Org` を使うが、`withManifestLock` は呼ぶたびに `manifest.lock` を開き直すので、flock は 2 つの process のときと同じく直列にする

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 7c-2 の status の行の並び(state-dir の行、org watch の行、保証の行) | 保たれている。`director:` の行は org watch の行のすぐ下、保証の行の上に入る | `TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines`、`TestStatusCmd_OrgWatchLineFollowsTheStateDirLine`、実機の 1 |
| 見張りがないときの escalate の即時の通知(6 段目までの動き) | 保たれている。変わったのは理由の文字列(`inbox_no_director` から `inbox_no_watch`)だけ | 既存の escalate のテストが両方の版で PASS、実機の 1 と 5 |
| 古い記録の `inbox_no_director` の `notified` | 送った件として読まれ、ack の期限は効かない | `TestInboxDeadlineDue` の「notified before stage 8a (inbox_no_director)」の行(DL2 で落ちる) |
| 6 段目の期限の計算 | `delivery` を読まず、結果は同じ | `TestInboxDeadlineDue` の 3 つの delivery の行、DL1 red |
| 前に揺れた doctor、watcher、gate のテスト | 落ちていない | evidence の 2.1 |

## Test gaps

1. D5(畳み込みで、違う `id` の `detached` を無視すること)を止めるテストがない。今の行は打ち消し合って見えない。一番小さい足し方は、`attached d1` と `detached d2` だけのファイルで、今の登録が d1 で無視した数が 1 であることを確かめる行(evidence の 4.2 の PD と同じ形)。手で書き換えたときにしか起きないので、PR を止めるものではない
2. 同じ件への approve の同時実行は、コミットされたテストがない。probe PA で HEAD の動きを確かめたが、PA は scratchpad にあるだけ
3. `--once`、stale、stopping の見張りがあるときの escalate と attach は、本物の `ProbeWatch` と持ったロックと手で書いた印の Go のテスト(`TestOrgEscalate_ProbesTheLedgersWatch`、`TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork`)だけで確かめた。binary では見ていない
4. approve は binary で打っていない(機能の org の束縛と計画の digest が要り、herdr なしで組むには manifest を手で書く必要がある)。記録だけになること、send と resolve の順、send の前に止まった件が期限で届くことは、偽の時計の Go のテストだけ。実機の 3 で、acked のまま resolve されない件が resolve の期限で届くことは実際の時計で見た
5. 実機の期限は `ack_minutes = 1`、`interval_seconds = 5` で見た。既定の 10 分と 30 秒は、偽の時計のテスト(10 分ちょうどでは送らず 10 分 1 秒で送る)だけ
6. escalate のあとで見張りが止まったときに記録だけの件が届かない穴は、計画の Non-goals と tech-debt に書かれたとおりで、確かめていない
7. 通らない文のうち、plain な attach と detach の追記・write・close の失敗、`os.Getwd` と `filepath.Abs` の失敗にはテストがない(Coverage の項)
8. verify が未確認とした `ralph init` の実行(`/director` skill が scaffold に入ること)は、今回も打っていない

## Verdict

- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。`go test -count=1 ./internal/...` は go1.25.8 と go1.26.0 の両方で rc 0(cli 678/599、org 746/1,523 と skip 1、ほか 6 パッケージ。名前は両方の版で同じ)。足した 70 本のテストはどれも PASS。`-race` も通った。テストは本物の osascript と herdr を打たず(canary は 0 回)、repo の台帳に `director.jsonl` はない(`.harness/state/org` 自体がどちらにもない)
- Fail: なし
- Blocked: なし
- Verified: 計画の Test plan の Unit、Integration、Regression、Edge case の各項目(上の対応表)。mutant 42 件のうち 41 件が red。HEAD の binary で、見張りのある間の escalate が記録だけになり、ack の期限(`inbox_ack_deadline`)と resolve の期限(`inbox_resolve_deadline`)で実際に人への経路に届くこと(verify が未確認とした項目)、見張りがないときと `enabled = false` のときの即時の通知、入れ替えと A → B → A の detach、linked worktree からの main worktree の規則(`git worktree list` の分岐を含む)を確かめた
- Not verified: Test gaps の 1〜8。どれも PR を止めるものではない
