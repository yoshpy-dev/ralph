# Test report: approve-resolve-decision

- Date: 2026-11-20(計画の日付は 2026-11-19)。cycle 1(`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-approve-resolve-decision.md`(issue #345)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `b7a9552e`(origin/main `32274500` の merge を含む。worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから。scope は wrapper の既定の changed(`==> Language scope: changed (changed_languages)`、pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。full run のあと、`internal/org` と `internal/cli` に絞った `go test`(`-race -count=20`、mutant、gap の probe、coverage)を打った
- Evidence: `docs/evidence/test-2026-11-19-approve-resolve-decision.log`(6,943 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。1 節が run-test.sh の出力そのもので、2〜11 節が絞った run、mutant、probe のコード、coverage。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-19-161035.log` にも書いた(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-20 01:10:35 〜 01:43:34 JST、1,979 s、rc 0 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本 | 46 本(`OK` 46 行) | 0 | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | Go の段の前まで |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` のパッケージもない | 下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.916 s |
| `internal/cli` | ok | 854.607 s |
| `internal/config` | ok | 37.140 s |
| `internal/gitroot` | ok | 6.193 s |
| `internal/insights` | ok | 1.482 s |
| `internal/org` | ok | 973.173 s |
| `internal/org/driver` | ok | 4.342 s |
| `internal/org/protocol` | ok | 3.244 s |
| `internal/scaffold` | ok | 5.059 s |
| `internal/toolchain` | ok | 40.517 s |
| `internal/upgrade` | ok | 2.738 s |

実行の条件:

- 負荷(`sysctl -n vm.loadavg`): 始まり `5.38 9.35 21.63`、終わり `16.29 24.36 43.03`。途中で 1 分値が 192.90 まで上がった(01:25 ごろ。ほかのセッションの Xcode の build で `swift-frontend` が 6 本、CPU を 90% ずつ使っていた)
- 自分の run との重なり: mutant の run(01:11 〜 01:29 ごろ、1 本ずつ順に)は shell の段と Go の段の始まりに重なった。`-race` の run と probe(01:40 〜 01:43)は Go の段の終わりに重なった。`internal/org` の 973.2 s は golang の pack の `-timeout=30m` の 54%、`internal/cli` の 854.6 s は 47%
- 起動: `run_in_background` は使わず、perl の fork と `setsid` で runner(`$TMPDIR` の下の scratch)を切り離した。INT、HUP、QUIT、TERM は DEFAULT に戻し、標準入出力は `/dev/null`。runner は始まりと終わりの時刻、負荷、rc を meta file に書く。待つのは `caffeinate -w <pid>` を 175 s ずつ区切って繰り返した。shell の SKIP は上の 2 件だけで、signal を使う suite は skip していない
- 走らせる前に `pgrep` で、この machine に別の `run-test.sh` も `run-verify.sh` も `go test` もないことを確かめた。full run は 1 回だけ

落ちたテストがないので、full run の打ち直しはしていない。

## 絞った run(flakiness と `-race`)

| Command | 結果 | 条件 |
| --- | --- | --- |
| AC-10 の 3 本(`TestResolveInboxItem_ApproveWaitsForAResolveThatHoldsTheLock`、`_ResolveWaitsForAnApproveThatHoldsTheLock`、`_ItemsThatAreNotPlanQuestionsDoNotWaitForTheManifestLock`)、`-race -count=20` | 60 / 60 PASS、DATA RACE 0、1 回の最長 0.04 s | full run の Go の段の途中(01:40:15、負荷 9.91) |
| 同じ 3 本、`-race -count=20` | 60 / 60 PASS、DATA RACE 0、1 回の最長 0.07 s | 12 CPU に perl の空回り 36 本を重ねた(負荷 37.87 → 45.65) |
| 同じ 3 本、`-race -count=200` | 600 / 600 PASS、DATA RACE 0 | full run のあと |
| `internal/org` の新しい 14 本(`TestResolveInboxItem_` の 13 本と `TestOrgSend_SentEventRecordsTheTextSHA256`)、`-race -count=20` | top-level 280 / 280、subtest 340 / 340、DATA RACE 0 | Go の段の途中 |
| `internal/cli` の新しい 4 本、変わった 3 本、関わる既存の 2 本(`TestOrgApprove_RecordsAndPrintsTheSendThenTheResolve`、`TestOrgInboxResolve_NoteRulesAndTransitions`)、`-race -count=20` | 180 / 180、DATA RACE 0 | Go の段の途中 |
| `TestRenderRolePrompt_Leader_ApprovalWaitReadsTheResolvedItem`、`TestDirectorSkill_ApprovalResolveWaitsForTheDecision` ほか 2 本、`-v` | 4 本とも PASS、SKIP 0(skill の写しが見つからないときの `t.Skipf` に落ちていない) | — |

AC-10 の 3 本は 1 本あたり 240 回、合わせて 720 回走らせて、落ちも race もなかった。lock を待つ側は lock file を開いたまま `LOCK_NB` を 25 ms ごとに打ち直す(`acquireFlock`)ので、`waitForLockWaiter` が数える open の数は待つ間ぶれない。3 s の期限に対して 1 回の最長は 0.07 s だった。

## 計画のテストと AC の対応

| AC / Test plan の項 | テスト | 結果 |
| --- | --- | --- |
| AC-1(DECISION のない resolve は exit 1 で断られ、件は open か acked のまま。エラーに send の打ち方と `--without-decision`) | org: `TestResolveInboxItem_RefusedUntilTheApprovalDecisionIsSent`(open と acked)、`_RefusalRepeatsTheStateDir`。CLI: `TestOrgInboxResolve_ApprovedItemNeedsTheDecisionSent`(exit 1、`--state-dir` を繰り返す、stdout と stderr は空、inbox は書かれない) | pass |
| AC-2(DECISION を送ったあとは通る) | 上の 2 本の後半、`TestOrgApprove_TheDecisionAndResolveCommandsRunInAPOSIXShell`(approve が出した 2 つの打ち方を `sh` で読んだ引数で実行) | pass |
| AC-3(`--without-decision` で通り、stderr に 1 行) | org: `_WithoutDecisionResolvesAndSaysNoDecisionWasFound`、`_WithoutDecisionOnItemsThatNeedNoneSaysNothing`。CLI: `TestOrgInboxResolve_WithoutDecisionSaysNoDecisionWasFound`(stderr を丸ごと比べる) | pass |
| AC-4(数えない `sent` 8 種) | `_SentsThatAreNotCounted` の subtest 8 本(dry run、承認より前、leader 以外、別の org、別の計画、直しの依頼、別の TYPE、`text_sha256` のない古い `sent`)。どれも同じ fixture に承認の DECISION を足すと通る対照つき。CLI は本物の `--dry-run` の send でも断られる | pass |
| AC-4(160 バイトと 180 バイトのパス) | `_LongPlanPathsAreToldApartByTheWholeText`(2 つの長さで、Details が同じになることを確かめてから、直しの依頼と違う digest を Send で送って断られ、承認の DECISION で通る。断り 4 件と通り 2 件) | pass |
| AC-4(`submit_unconfirmed=true` は数え、stderr に `ralph org read`) | org: `_SubmitUnconfirmedCountsAndIsReported`(確かめた send が 1 つでもあれば印は消える)、`_SentSubmitUnconfirmedReadsTheLeadingWords`(6 行)。CLI: `TestOrgInboxResolve_UnconfirmedSubmitSaysToReadTheLeadersPane` | pass |
| AC-5(今のままの件) | `_ItemsWithNoApprovalResolveAsBefore` の subtest 5 本(`PLAN:` のない QUESTION、承認のない計画の件、disband の前の承認だけ、承認の記録のある BLOCKED、本文にだけ `PLAN:` のある QUESTION)、`_UnknownResolvedAndBadNoteKeepTheirErrors` | pass。ただし G2、G3 |
| AC-6(leader の prompt) | `TestRenderRolePrompt_Leader_ApprovalWaitReadsTheResolvedItem`(手順 3 と 4、立て直しの受信箱の項の文の順。note と状態の値は ralph が書くものと突き合わせる) | pass(文を固めるだけ。G7) |
| AC-7(director skill の 4 写し、approve と resolve の help) | `TestDirectorSkill_ApprovalResolveWaitsForTheDecision`(4 写し、手順 6 に flag がない)、`TestOrgApproveHelp`、`TestOrgInboxResolveHelp_SaysTheApprovalDecisionRule`。`scripts/check-skill-sync.sh` は verify の担当(verify の報告で pass) | pass |
| AC-8(org と CLI の両方で、送らずに閉じると断られ、flag で通り、送ってから閉じると通る) | AC-1〜3 の org と CLI のテスト | pass |
| AC-9(`sent` の `text_sha256`) | `TestOrgSend_SentEventRecordsTheTextSHA256`(Details が切れる長さの本文の sha256、claude の leader なので paste の印を除いた本文の hash、dry run にはなく JSON の key も出ない)、`_RefusedUntil…` の最後の event | pass |
| AC-10(manifest の lock の中で検査と書き込み) | 上の AC-10 の 3 本 | pass。ただし G1 |
| Test plan の Integration(終了コード、エラーの文、flag の stderr、`--state-dir`) | CLI の 4 本 | pass |
| Test plan の Regression(今の inbox と approve のテスト、prompt と skill の文を固めるテスト) | full run。`org_approve_test.go` の 3 本は新しい検査に合わせて直してある(dry run だけの send では resolve できなくなった) | pass |
| Test plan の Edge cases: acked の件 | `_RefusedUntil…`(org の層だけ。CLI は open だけ) | pass |
| Test plan の Edge cases: 件の org が disband 済み | AC-5 の「disband の前の承認だけ」 | pass |
| Test plan の Edge cases: 同じ件に承認が 2 つ(打ち直し) | テストがない。ralph の書く台帳では起きない(G5) | gap |

## Mutation

`internal/org/approve_resolve.go` と `verbs.go` の写しを `-overlay` で差し替えた。worktree のファイルは書き換えていない(前後で `git status --porcelain --ignored` を比べ、go-build の cache 以外は同じ)。16 個のうち既存のテストが 9 個を落とし、7 個が残った。

| Mutant | 変えたところ | 結果 |
| --- | --- | --- |
| K | `withManifestLock` を素の関数呼び出しにする | AC-10 の 3 本が落とす |
| N1 〜 N4 | `DryRun`、`OrgID`、`SeatID == LeaderIdentity`、`TextSHA256 != ""` の条件をそれぞれ外す | AC-4 の subtest がそれぞれ 1 本で落とす |
| N5 | disband のあとから読む `start = d + 1` を 0 にする | AC-5 の disband の subtest が落とす |
| N6 | `--without-decision` を無視する | `_WithoutDecisionResolves…` が落とす |
| N7 | paste の印を付けた本文の hash を記録する | AC-9 のテストを含む 5 本が落とす |
| N8 | `submit_unconfirmed` を確かめた send でも消さない | 3 本が落とす |
| A | 承認の記録の `a.InboxID != id` を外す | 残る(org 360 本、CLI 65 本)。G2 |
| B | 最初の読みで見つからない ID を lock なしで resolve する | 残る。G4 |
| C | lock の中の読み直しで resolved の件を外す条件を消す | 残る。G3 |
| L | inbox への書き込みを manifest の lock を放したあとに移す(検査は lock の中のまま) | 残る(`-run TestResolveInboxItem_` を `-count=50`)。G1 |
| D | lock の中の読み直しで `isPlanQuestion` を外す | 残る。G5(ralph の書く台帳では同じ動き) |
| E | 承認の DECISION を最後の承認の分だけ数える | 残る。G5(同上) |
| G | `plan_approved` の `SeatID == ""` を外す | 残る。G5(同上) |

N5 は最初 `d + 0*d` と書いて元と同じ値になり、N8 は最初の書き方が build に失敗した(`declared and not used`)。どちらも書き直して打ち直し、上の結果はその打ち直しのもの。

A、B、C、L は、それぞれを狙った probe のテスト(`internal/org/zz_gap_probe_test.go` を `-overlay` で足した。commit していない。コードは evidence の 9 節)で落ちることを確かめた。元のコードでは 4 本とも `-race -count=20` で通った。A、B、C では 3 本の probe を一緒に走らせ、狙った 1 本だけが落ちた。L は狙った 1 本だけを `-count=20` で走らせ、20 回とも落ちた。D にはどの probe も効かない。

## Coverage

- Statement: `approve_resolve.go` は `internal/org` の絞った run(`-run 'TestResolveInboxItem_|TestOrgSend_SentEventRecordsTheTextSHA256|TestDirectorSkill_ApprovalResolveWaitsForTheDecision'`)で 70 / 76 = 92.1%。CLI の run(`-run 'Inbox|Approv'`、`-coverpkg` に cli と org)で 86.8%
- Branch: 測っていない
- Function: `ResolveInboxItem` 88.5%、`readApprovalDecisionCheck` 80.0%、`checkApprovalDecision` 95.5%、`findInboxItem`・`isPlanQuestion`・`sentSubmitUnconfirmed`・`approvalDecisionNotSentError`・`sentTextSHA256` は 100%。CLI の `newOrgInboxResolveCmd`、`printResolveNotes`、`newOrgApproveCmd`、`printApprovedPlans` は 100%
- Notes: 通らない block は 6 つ。5 つはエラーの経路(inbox store がない、inbox の読みの失敗、lock の中の検査の失敗、manifest の読みの失敗)。残る 1 つは `approve_resolve.go:218` の、別の件の承認を飛ばす `continue` で、mutant A が残るのと同じ穴。パッケージ全体の coverage は取っていない(full run は `-cover` なし)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | | | |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #345: director が DECISION を送らずに `ralph org inbox resolve` だけを打ち、件が閉じて leader に承認が届かない | 直っている | AC-1 の org と CLI のテスト。CLI のテストは approve が出した resolve の打ち方そのもので断られることを見る |
| 長い計画のパスで、承認と直しの依頼と違う digest が同じ `sent` の記録になる(計画助言の [HIGH]) | 起きない | `_LongPlanPathsAreToldApartByTheWholeText`、mutant N7 |
| resolve の検査のあとに approve が記録し、resolve が先の結果で閉じる(計画助言の [MEDIUM]) | 起きない | AC-10 の 3 本と mutant K。書き込みの側は G1 の probe で確かめた |
| 古い ralph の `sent`(`text_sha256` なし)と古い manifest の行 | 数えない。読み飛ばしで壊れない | AC-4 の「older ralph」の subtest、AC-9 の dry run の行に key が出ない確かめ、full run の既存の manifest のテスト |
| 既存の inbox、approve、prompt、skill のテスト | 変わらない | full run(`internal/org` と `internal/cli` が ok) |

## Test gaps

- G1(AC-10、閉じられる): AC-10 の 3 本は、検査が lock の中にあることは固めるが、inbox への書き込みが lock の中にあることは固めない。mutant L(書き込みを lock を放した直後に移す)は 50 回とも通った。approve は lock を 25 ms ごとに取り直す(`manifestLockPollInterval`)ので、その間に resolve の書き込みが終わり、approve はいつも「ほかの誰かが resolve した」と読む。probe `TestGapProbe_ResolveWritesTheInboxUnderTheManifestLock` は、止めた resolve を進める前にテストが inbox の lock を持ち、300 ms のあいだ approve が承認を記録しないことを見る。元のコードで 20 回とも通り、mutant L を 20 回とも落とした。足すなら `approve_resolve_test.go` の lock のテストの隣に
- G2(AC-5、閉じられる): 同じ org に承認を記録した別の計画の件があるとき、承認のない計画の件の resolve を打つテストがない。mutant A では、その resolve が別の件の承認で断られる。たとえば e1 を承認して `--without-decision` で閉じ、leader が直した計画で e2 を上げ、人が直しを求めて e2 を resolve すると、mutant A では断られる。probe `TestGapProbe_ResolveIgnoresAnotherItemsApproval` が落とす
- G3(閉じられる): 承認を記録して resolve 済みの計画の件に、もう一度 resolve を打つテストがない。mutant C では、2 度目の resolve が「already resolved」ではなく DECISION の断りになる(`--without-decision` で閉じたあとに打ち直すと起きる)。probe `TestGapProbe_ResolvedApprovedItemKeepsTheResolvedError` が落とす
- G4(閉じられる、起きにくい): `ResolveInboxItem` のコメントにある「最初の読みで見つからない ID も lock を取る」を固めるテストがない。mutant B は残る。probe `TestGapProbe_UnknownAtFirstReadWaitsForTheLockAndIsChecked`(テストが manifest の lock を持つ間に、まだない `e2` の resolve を始め、その間に件を上げて承認を記録する)が 3.01 s で落とす。実際には最初の読みと lock の間に escalate と approve の両方が入る必要がある
- G5(ralph の書く台帳では同じ動き): mutant D、E、G は残る。approve は org 単位の `plan_approved` を、`PLAN:` のある QUESTION にだけ、1 つの命で 1 件に 1 度だけ書く(`approve.go:290`、`:295`、`:374`、`checkNotApprovedYet`)。そのため、この 3 つの違いは手で書き換えた台帳でしか出ない。Test plan の Edge case「同じ件に承認が 2 つ(打ち直し)」も同じ理由でテストがない
- G6: エラーの経路(inbox store がない、inbox と manifest の読みの失敗)はテストがない(Coverage の 5 block)
- G7: AC-6 は prompt の文を固めるだけで、本物の leader が待つ間に起こされて `ralph org inbox show` で note を読むかは見ていない。herdr と本物の座席を使う run はしていない
- G8(小さい): screen の経路(`via=screen`)の `sent` の `text_sha256` を見るテストはない。herdr の経路と同じ `appendEvent` の 1 文なので、今は statement で覆われている。CLI の層では acked の件の断りを見ていない(org の層では見る)

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,979 s)、AC-10 の 3 本の `-race` 計 720 回、新しい org のテスト 14 本と CLI のテスト 9 本の `-race -count=20`、prompt と skill のテスト 4 本
- Fail: 0
- Blocked: なし
