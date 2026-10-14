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

## Cycle 2

- Date: 2026-10-15(JST 06:02〜06:32)
- Plan: docs/plans/active/2026-10-14-org-director-attach.md(`Status: Approved`、Approved sha256:27414f27525b。digest が承認時と同じことは verify cycle 2 が確かめた)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 2 回目(`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定 2 の最後)
- Scope: HEAD 9548a8cd(base ef575748、26 コミット)。cycle 1 の a5f5353b から Go が変わったのは 3 コミットで、ファイルは `internal/org/director.go`、`director_test.go`、`internal/cli/org_director.go`、`org_director_test.go` の 4 つ。624f603b は `director status` の help の 1 文とその test、634b019a は cross-review cycle 1 の ACTION_REQUIRED #1 の直し(`appendLocked` が write か close の失敗で追記前の大きさに切り戻す、test の seam `write` と `truncate`、テスト 5 本)、f4c1a394 は self-review cycle 2 の C2-L1〜C2-L5 の直し(CLI の `notRestoredError`、`org: director:` を 1 回に、コメントと doc、tech-debt の 1 行、CLI のテスト 1 本)。残りは記録と文書。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0。開始時の load average は 2.8(15 分平均 15.4)で、run-test.sh の shell の間は別のセッションの負荷で 1 分平均が 40 近くまで上がった
- Evidence: `docs/evidence/test-2026-10-14-org-director-attach.md` の `## Cycle 2`(C2-1 run-test.sh の抜粋、C2-2 Go の結果と数・`-race`・canary・coverage、C2-3 mutation と probe のテスト、C2-4 HEAD の binary の実機確認)。wrapper が書いた `docs/evidence/verify-2026-10-14-210346.log` は gitignore の対象なのでコミットしない
- テストもコードも変えていない。mutant と probe のテストは scratchpad に置き、`go test -overlay` で読ませた。本物の ENOSPC を起こすための 1 MB の HFS+ ディスクイメージも scratchpad に作り、scratchpad の中に `-nobrowse` で mount して、終わったあと detach した。worktree に書いたのは、この節と evidence の `## Cycle 2` だけ

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`scripts/ralph-config.sh` の変更で full に広がった。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 679 秒、rc 0 |
| `go test -count=1 -v ./internal/...`(go1.26.0) | cli: top-level 679、subtest 601。org: 751、1,525。org/driver: 48、89。org/protocol: 15、56。config: 37、34。insights: 36、7。scaffold: 30、0。upgrade: 114、13 | 同じ数から skip を引いた数 | 0 | org 1、scaffold 2 | 199 秒(cli 198.9 s、org 78.4 s)、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test -count=1 -v ./internal/...` | go1.26.0 と同じ | go1.26.0 と同じ | 0 | org 1、scaffold 2 | 206 秒(cli 205.1 s、org 85.7 s)、rc 0 |
| うち cycle 1 のあとで足したテスト 6 本 | top-level 6(org 5、cli 1)、直下の subtest 4 | 両方の版ですべて | 0 | 0 | - |
| `go test -count=1 -run 'Director\|NotRestored' -v ./internal/org/ ./internal/cli/` | org: top-level 37、subtest 44。cli: 28、43 | すべて | 0 | 0 | 11 秒、rc 0 |
| `go test -count=1 -race ./internal/org/ ./internal/cli/ -run 'Director'` | 2 パッケージ(top-level 65 本) | 2 | 0 | - | 17 秒、DATA RACE なし |
| mutation(`go test -overlay`、go1.26.0) | mutant 18 件と対照 2 件 | red 15 件 | 生き残り 3 件(R11、H4、H5) | - | 絞った回は 1 件 4〜9 秒、生き残りの全体の回は 84 秒と 203 秒 |
| probe のテスト(overlay だけ、コミットしない) | 1 本(PT) | HEAD で通る | 狙った mutant(R11)のもとで落ちる | - | - |
| HEAD の binary の実機確認(一時の `--state-dir`、下の「実機の確認」) | 通常のディスクで 3 段、1 MB のディスクイメージで 3 段 | すべて期待どおり | 0 | - | 数秒ずつ |

- shell は 41 本すべてが `OK` で終わり、`FAIL:` の行はどれも 0。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)。cycle 1 と同じ形で抜き出した 215 行は、行番号と時間を除くと cycle 1 の evidence の 1 と行ごとに同じだった(違いは cycle 1 が手で足した `rc=0` と `real` の 2 行だけ)
- `run-test.sh` の中の Go は cli(199.4 s)、insights、org(81.4 s)を流し直し、config、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。この 5 つは上の 2 回の `-count=1` の実行で ok を確かめた
- go1.25.8 と go1.26.0 で、テストの名前と結果の行(時間を除いて並べた 4,038 行)は同じだった。cycle 1 の 4,028 行との差は、足した 6 本と subtest 4 本の 10 行で、消えた行はない
- Skipped は cycle 1 と同じ 3 本(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`、scaffold の `TestBaseFS_WithMockFS` と `TestAvailablePacks_WithMockFS`)
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どの実行でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は 2.59 s と 2.69 s で、+5 s の場合には当たらなかった
- AC9: Go の全体の 2 回、絞った回、`-race`、mutation のすべての回、probe のテスト、coverage の回は、PATH の先頭に canary の `osascript` と `herdr` を置いて走らせた。呼ばれた回数は 0 回。手で同じ PATH から打つと 2 つとも記録されたので、PATH の検索は canary に届く。テストのあと、main のチェックアウトにも worktree にも `.harness/state/org` はなく、repo の下に `director.jsonl` は 1 つもない

### cycle 2 の変更とテストの対応

「mutant」の列は、その変更の判定を壊した mutant が落ちたかどうか(id は下の Mutation の表と evidence の C2-3)。

| 変更 | テスト | mutant |
| --- | --- | --- |
| write が途中で止まったら、同じロックの中で追記前の大きさに切り戻す(replace、plain な attach と detach。ファイルがなかった attach は空のファイルに戻す) | `TestDirectorAttach_AReplaceCutOffPartwayIsRestored`、`TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored`(2 つの subtest)。実機の B2 と B5a | R1、R2、R2w、R6、R8 red |
| close の失敗も同じく切り戻す | `TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored` | R2c、R4、R5 red |
| 切り戻せなかったときは `ErrDirectorWriteNotRestored` を包み、「nothing was recorded」と言わない | `TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo`。実機の B3 と B5b | R3、R3b、R7 red |
| CLI の `notRestoredError`(sentinel のときだけ `ralph org director status` を足し、`--state-dir` を渡した呼び出しには `--state-dir` を付ける。ほかの error は素通し) | `TestOrgDirector_NotRestoredErrorNamesTheStatusCommand`(`--state-dir` のありとなし) | H1、H2、H3 red |
| attach と detach の RunE が `notRestoredError` を通る | なし。実機の B3(attach)と B5b(detach)で HEAD の動きを確かめた | H4、H5 生き残り |
| 末尾に改行のない raw への追記が失敗したとき、前に足した改行ごと切り戻す | なし。probe の PT が HEAD で通り、R11 のもとで落ちる | R11 生き残り |
| ほかの id の `detached` は今の登録を終わらせない(cycle 1 の生き残り D5) | `TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration` | D5 red(cycle 1 は生き残り) |
| `org: director:` を 1 回だけ言う(C2-L2) | 切り戻しのテストのうち 3 本が文の全体を、close の 1 本が文の先頭を等号で比べる。実機の A3、B2、B3、B5 の文でも 1 回 | - |
| `director status` の help の 1 文(624f603b) | `TestOrgDirectorHelp_DescribesTheRegistration` | - |

### 実機の確認(HEAD の binary)

HEAD を go1.25.8 で build した binary を、`env -i` で PATH に git、ralph、偽の `osascript` だけを置いて打った。cwd は scratch の git repo(`ralph.toml` なし。`[org.director]` は既定の `enabled = true`)で、台帳はどの回も一時の `--state-dir`。herdr、agmsg、本物の osascript は動いていない(偽の osascript の記録は 0 行)。evidence の C2-4 に script と出力がある。

1. A1: `attach --driver claude` は stdout に `attached d1 (driver=claude)` と `detach it with: ralph org director detach --id d1 --state-dir '<台帳>'`、stderr に見張りなしの警告(その打ち方にも `--state-dir` が付く)を出して exit 0。`status` は d1 の欄(`user: u`、`herdr_pane: no`)を出し、`status --json` は `state: attached`、登録、`corrupt_lines: 0`、`ignored_events: 0` を出した
2. A2: `attach --driver codex --replace` は `attached d2 (driver=codex), replacing d1 (...)` を出した。`detach --id d1` は exit 1 で「the current registration is d2 ... does not need to detach」と言い、ファイルは `cmp` で同じ。`detach --id d2` は通り、`status` は `not attached`。ファイルは attached d1、detached d1、attached d2、detached d2 の 4 行
3. A3(読み取り専用): director.jsonl を mode 0444 にすると、`attach --replace`、`detach --id d3`、`detach --force` はどれも exit 1 になった。macOS は write ではなく open で断った(`open <path>: open <path>: permission denied`)ので、この 3 つは切り戻しの経路を通らない。文はどれも「nothing was recorded, so d3 ... is still the current registration」で、ファイルは `cmp` で同じ。0644 に戻すと d3 は今の登録のままで、`detach --id d3` が通った
4. B2(本物の ENOSPC): 1 MB の HFS+ イメージ(allocation block 4,096 bytes)に台帳を置いた。replace の 2 行(detached 244 bytes、attached 277 bytes)のうち detached の全体と attached の途中までが 1 つの block に収まるよう、d1 の attached の行の `cwd` を伸ばして director.jsonl を 3,714 bytes(境目の 382 bytes 手前)にし、残りを `dd` で埋めた(`df` の空きは 0)。`attach --replace` は exit 1 で `nothing was recorded, so d1 ... is still the current registration: append to <path>: write <path>: no space left on device` と言い、ファイルは `cmp` で元の bytes と同じ、`status` は d1 だった
5. B3(切り戻せない場合): 同じ状態で director.jsonl に `chflags uappnd`(追記だけを許す)を付けると、write は ENOSPC で失敗し、truncate は `operation not permitted` で断られた。exit 1 の文は `org: director: attach: append to <path>: write <path>: no space left on device, and director.jsonl could not be restored to its size before the append (truncate <path>: operation not permitted), so it may hold part of the write: read the registration with ralph org director status --state-dir '<台帳>'` で、「nothing was recorded」とは言わない。ファイルは 4,096 bytes になり、増えた 382 bytes は detached d1 の 1 行全体と attached d2 の途中だった。`status` は `not attached` と、stderr に `1 unreadable director line(s) skipped` を出した。cross-review cycle 1 が指摘した「detached が完全な行で残り、登録がなくなる」状態を本物の ENOSPC で作り、HEAD がそれを文で知らせることを見た。B2 は truncate だけが通る同じ条件なので、B2 でも 382 bytes が書かれてから切り戻されたとみている(おそらく。B2 の途中の状態は見ていない)
6. B5(detach の経路): ファイルを境目の 122 bytes 手前にして埋め直すと、`detach --id d1` と `detach --force` は ENOSPC で失敗して切り戻され(`cmp` で同じ)、「nothing was recorded, so d1 ... is still the current registration」と言った。`uappnd` を付けた `detach --id d1` は、detach の RunE から B3 と同じ形の文(`...: read the registration with ralph org director status --state-dir '<台帳>'`)を出した。ファイルには detached d1 の途中の 122 bytes が残り、`status` は d1 を今の登録として出し、壊れた 1 行の警告を出した

- B3 と B5b で、attach と detach の RunE がどちらも `notRestoredError` を通り、`--state-dir` つきの打ち方を足すことを HEAD の binary で確かめた。Go のテストはこの配線を固定していない(Mutation の H4、H5)
- verify cycle 2 が C2-V2 で「おそらく」とした本物の write の失敗の文は、`append to <path>: write <path>: no space left on device` と path が 2 回出る形だった。seam のテストが固定する `append to <path>: no space left on device` とは形が違う。動きは変わらない
- 後片付け: `uappnd` を外し、埋めたファイルを消して `hdiutil detach` で外した(`"disk14" ejected.`)。mount の一覧に残りはない

### Coverage

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.6%、`internal/cli` 87.3%(cycle 1 は 94.5%、87.3%)
- `internal/org/director.go` は 145/146 文(99.3%。cycle 1 は 129/135)。通らない 1 文は `appendLocked` の event の marshal の失敗(366-368)。cycle 1 で通らなかった write と close の失敗、plain な attach と detach の追記の失敗は、新しいテストが通す。関数では `appendLocked` が 95.0% で、ほかの 19 関数は 100%(`restoreLocked`、`directorAppendFailed` を含む)
- `internal/cli/org_director.go` は 138/141 文(97.9%)。通らない 3 文は、attach の `return rt.notRestoredError(err)`(121。store が `ErrDirectorAttached` 以外の error を返したとき)、`filepath.Abs` の失敗(335)、`os.Getwd` の失敗(350-352)。`notRestoredError` 自体は 100%。detach の `return rt.notRestoredError(err)`(183)は「no director is attached」などの拒否で通るが、そのとき helper は error を素通しするので、`return err` に戻しても結果は変わらない(H5)
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

### Mutation

- 対象は 2 つの直しのコミットで、`appendLocked`・`restoreLocked`・`directorAppendFailed` の切り戻し(R1〜R11)、cycle 1 の生き残り D5、CLI の `notRestoredError` と attach・detach の RunE(H1〜H5)。ファイルを overlay で差し替え、org は `-run Director`、cli は `-run OrgDirector` で流した。生き残りはパッケージ全体(`-run .`)でも流した。対照の X0 と X0c(手を加えないファイルを overlay で読ませたもの)は通った。どの差し替えも元の文字列がちょうど 1 か所にあり、飛ばした mutant はない

| ID | 変異 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| R1 | write の失敗で truncate を飛ばす | red | `AReplaceCutOffPartwayIsRestored`、`AReplaceThatCannotBeCutBackSaysSo`、`PlainAttachAndDetachCutOffPartwayAreRestored` の 2 つの subtest |
| R2 | 切り戻す大きさを `len(raw)` でなく 0 にする(両方の分岐) | red | `AReplaceCutOffPartwayIsRestored`、`AReplaceThatCannotBeCutBackSaysSo`、plain の detach、`ACloseFailureAfterAFullWriteIsRestored` |
| R2w | 同じ変異を write の失敗の側だけに | red | `AReplaceCutOffPartwayIsRestored`、`AReplaceThatCannotBeCutBackSaysSo`、plain の detach |
| R2c | 同じ変異を close の失敗の側だけに | red | `ACloseFailureAfterAFullWriteIsRestored` |
| R3 | truncate が失敗しても、戻せたときの error を返す(sentinel なし) | red | `AReplaceThatCannotBeCutBackSaysSo` |
| R3b | 文は同じまま、sentinel を `%v` にして包まない | red | `AReplaceThatCannotBeCutBackSaysSo`(`errors.Is` の確認) |
| R4 | close の失敗を成功として扱う | red | `ACloseFailureAfterAFullWriteIsRestored` |
| R5 | close の失敗で truncate を飛ばす(error は返す) | red | `ACloseFailureAfterAFullWriteIsRestored` |
| R6 | 本番の truncate を、何もせず成功を返す関数にする | red | `AReplaceCutOffPartwayIsRestored`、plain の 2 つ、`ACloseFailureAfterAFullWriteIsRestored` |
| R7 | `directorAppendFailed` の not-restored の分岐を消す(戻せなくても「nothing was recorded」と言う) | red | `AReplaceThatCannotBeCutBackSaysSo` |
| R8 | 切り戻す大きさを `len(raw)+1` にする(失敗した書き込みを 1 byte 残す) | red | 切り戻しのテスト 4 本すべて |
| R11 | 末尾に改行のない raw のとき、前に足した改行を残して切り戻す | 生き残り(全体でも) | - |
| D5 | ほかの id の `detached` でも今の登録を終わらせる(cycle 1 の生き残り) | red | `TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration` |
| H1 | `notRestoredError` が status の打ち方を足さない | red | `TestOrgDirector_NotRestoredErrorNamesTheStatusCommand` の 2 つの subtest |
| H2 | 打ち方から `--state-dir` を落とす(`rt.command` を通さない) | red | 同じテストの `with --state-dir` |
| H3 | どの error にも打ち方を足す | red | 同じテストの 2 つの subtest(ほかの error が素通しされることの確認) |
| H4 | attach の RunE が helper を通さず `return err` にする | 生き残り(全体でも) | - |
| H5 | detach の RunE が同じ | 生き残り(全体でも) | - |

- handoff が名指した 6 つは、write の失敗で truncate を飛ばす(R1)、既存のファイルを 0 に切り詰める(R2、R2w、R2c)、truncate の失敗で戻せたときの error を返す(R3)、close の失敗を成功にする(R4)、CLI の helper の打ち方を落とす(H1)、打ち方の `--state-dir` を落とす(H2)で、どれも red
- 18 件のうち 15 件が red、3 件(R11、H4、H5)が生き残った
- R11: 切り戻しのテストはどれも、末尾に改行のある raw から始まる。R11 のもとでは、壊れた最後の行のあとに改行が 1 つ残る。壊れた行は壊れた行のまま読まれ、次の追記は改行を足さないので、登録の読み方は変わらない。ただしファイルは追記前の bytes と同じにならず、`appendLocked` の doc の「cuts the file back to len(raw) before returning」(`director.go:351`)と食い違う。attached d1 と切れた行だけのファイルで replace を失敗させる probe PT は、HEAD で通り、R11 のもとで落ちた(evidence の C2-3.3)
- H4、H5: store の seam(`write`、`truncate`)は unexported なので、CLI のテストは not-restored の error を RunE に返させられない。`TestOrgDirector_NotRestoredErrorNamesTheStatusCommand` は helper を直接呼ぶ。attach の 121 行は通らず、detach の 183 行は拒否の error でしか通らないので、どちらを `return err` に戻しても赤くならない。cli 全体(top-level 679 本)でも生き残った。HEAD の配線は実機の B3 と B5b で確かめた

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の 70 本と既存のテスト | すべて PASS。名前の並びは cycle 1 から 10 行増えただけで、消えた行はない | evidence の C2-2.2 |
| replace の 2 行を 1 回の write で書く(cycle 1 の L5) | 保たれている。A2 のファイルは同じ秒の detached d1 と attached d2 を持つ | 実機の A2、`TestDirectorAttach_*` の既存のテスト |
| open の失敗では何も書かず、今の登録を名指して「nothing was recorded」と言う | 保たれている | `TestDirectorAttach_AReplaceThatCannotWriteRecordsNothing`、実機の A3 |
| `ErrDirectorAttached` の拒否の文と、CLI が足す 2 つの打ち方(cycle 1 の L7) | 保たれている。`notRestoredError` は `ErrDirectorAttached` を素通しする | `director_test.go:207` の文の固定、`TestOrgDirector_NotRestoredErrorNamesTheStatusCommand` の素通しの確認(H3 で落ちる) |
| A → B → A の detach の拒否 | 保たれている | 実機の A2 |
| 前に揺れた doctor、watcher、gate のテスト | 落ちていない | evidence の C2-2.1 |

### Test gaps

1. attach と detach の RunE が `notRestoredError` を通ることを固定するテストがない(H4、H5)。HEAD の動きは実機の B3 と B5b で確かめた。足すには、CLI のテストから store の seam を差せるようにする必要がある
2. 末尾に改行のない raw への追記が失敗した場合の切り戻しにテストがない(R11)。probe PT の形の 1 本で閉じられる。PT は scratchpad にあるだけ
3. 切り戻し自体の失敗を実機で起こしたのは `chflags uappnd`(truncate が EPERM)だけ。ほかの理由で truncate が失敗する場合は、Go のテストの seam だけで確かめている
4. B2 と B5a で途中までの書き込みが実際に起きてから切り戻されたことは、同じ条件の B3 と B5b からの推測で、B2 と B5a の途中の状態は見ていない
5. ralph のプロセスが write と切り戻しの間で死んだ場合(verify cycle 2 の観察)は確かめていない
6. cycle 1 の Test gaps のうち、1(D5)は閉じた。7 のうち write と close の失敗、plain な attach と detach の追記の失敗も、新しいテストが通すようになった。2〜6、8 と、7 の `os.Getwd` と `filepath.Abs` の失敗は、今回の変更の外でそのまま残る

### Verdict

- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。`go test -count=1 ./internal/...` は go1.26.0 と go1.25.8 の両方で rc 0(cli 679/601、org 751/1,525 と skip 1、ほか 6 パッケージ。名前は両方の版で同じ)。cycle 1 のあとで足した 6 本はどれも PASS。`-run 'Director|NotRestored'` の回と `-race` も通った。テストは本物の osascript と herdr を打たず(canary は 0 回)、repo の台帳に `director.jsonl` はない
- Fail: なし
- Blocked: なし
- Verified: cross-review cycle 1 #1 の直し(write と close の失敗の切り戻し、戻せたときと戻せなかったときの文の分け方)と self-review cycle 2 の C2-L1、C2-L2 の直しを、テストと mutant 18 件(red 15 件)で確かめた。handoff が名指した 6 つの mutant はどれも red。cycle 1 の生き残り D5 は新しいテストが落とす。HEAD の binary で、本物の ENOSPC の途中までの書き込みが元の bytes に戻ること(`attach --replace`、`detach --id`、`detach --force`)、切り戻せないときに attach と detach の両方が「nothing was recorded」と言わず `--state-dir` つきの `ralph org director status` を足すこと、読み取り専用のファイルでは open で断られてファイルが変わらないことを確かめた
- Not verified: Test gaps の 1〜6。どれも PR を止めるものではない
