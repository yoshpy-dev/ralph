# Walkthrough: org-merge-cleanup(機能の PR が merge されたら、その機能の worktree・ブランチ・記録を ralph が消す)

- Date: 2026-10-30(plan の日付は 2026-10-28)
- Plan: docs/plans/active/2026-10-28-org-merge-cleanup.md(/pr で archive に移す)
- Issue: #263(org: 機能の worktree を merge のあとに自動で片付ける)
- PR: 未作成(/pr で作る)
- Branch: feat/org-merge-cleanup(この walkthrough の前の HEAD は 49aee225。merge を除く 62 commits と、origin/main を取り込んだ merge commit 3 つ)
- Base: e3a3fbf7(origin/main、「Merge pull request #287」= local-config-org)。分岐元も同じ
- 読む範囲: `git diff origin/main...HEAD` は 58 files、+8927/-353。Go の本番コードと leader の雛形が 19 files(+2156/-178)、Go のテストが 15 files(+4451/-51)、skill の写しが 8 files(+616/-108)、記録(計画・report・insight event)が 8 files(+1671)、残りの文書が 8 files(+33/-16)。動きを決めるのは `internal/org/cleanup.go`(新規 1086 行)、`internal/org/mergeorigin.go`(新規 112 行)、`internal/org/merge_run.go`、`internal/org/mergeorder.go`、`internal/cli/org_cleanup.go`(新規 460 行)、`internal/cli/org_merge.go`。テストの大半は新規の `internal/org/cleanup_test.go`(1809 行)と `internal/cli/org_cleanup_test.go`(1142 行)で、skill の 8 files は 2 つの skill の同じ文を 4 面に写したもの
- 後工程: パイプラインを 3 回回した(`RALPH_STANDARD_MAX_PIPELINE_CYCLES` を 3 に上げた。`cycle-count.json` は 2 のまま)。self-review は 7 run(CRITICAL と HIGH は 0。MEDIUM は M1・M2・C3-M1 の 3 件で、すべて直した)、verify と test は cycle 3 が pass、sync-docs は 3 回、cross-review(codex)は 3 回。1 回目と 2 回目は ACTION_REQUIRED が 2 件ずつで、直してから回し直した。3 回目は ACTION_REQUIRED 0、WORTH_CONSIDERING 2。記録だけの修正の例外は使っていない

```
何を変えるか → 読む順 → 片付けの本体(org 層) → merge の続きと origin に固定した PR の読み方 → CLI の 2 つ → 雛形・skill・help・文書 → テスト → 直しの経過 → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで:
  - `ralph org start --plan` が機能の worktree(`.claude/worktrees/org-<org_id>`)を作り、`ralph-worktree.sh` に記録 `org-<org_id>`(cleanup-policy は manual)を残す。disband でも merge でも消えず、人が merge のあとに `./scripts/ralph-worktree.sh cleanup --id org-<org_id>` を打っていた。
- この PR から:
  - `ralph org merge` は `pr_merged` を記録したあと、`regate.lock` を持ったまま片付ける。結果は stdout の `CLEANUP: done` か `CLEANUP: skipped (<理由>)` で、手で消すコマンドがあれば続けて `CLEANUP MANUAL:`。終了コードは変えない(exit 0 は記録した merge)。
  - 新しい動詞 `ralph org cleanup [--branch <ブランチ>]` が、GitHub で直接 merge した PR を同じ片付けで消す。行は `CLEANED:`・`SKIPPED:`・`MANUAL:`。すべて片付くか片付けるものがなければ 0、1 本でも残れば 1。
  - 消すのは script の `cleanup --id org-<org_id> --force-branch` を 1 回打つことだけ。Go の側が持つのは、消す前の確かめと、台帳への `worktree_cleaned` の記録。
  - 片付けの間は `regate.lock` の marker にその org_id を書き、同じ org_id の spawn と start を断る。
- issue との違い:
  - issue の受け入れ条件 3 は、断ったときに「手で消すときの打ち方を出す」と書く。D2 は、理由を直して打ち直せば続く断りでは打ち直しだけを出し、手で消すコマンドは ralph が先に進めない場面(記録がない、記録が結びつきと合わない、ブランチがなく worktree だけがある、登録の HEAD が危ない、消す途中の失敗)に限った。`--force-branch` の script を渡すと確かめ 5・6 を飛ばして、push していないコミットのあるブランチまで消すため(self-review の M2)。
  - 動詞の名前 `cleanup` と、終了コード(残れば 1)は、issue が計画に任せた部分を D4 で決めた。
- 計画の本文との違い(承認の digest が変わったときは取り直して自動承認した。最後は `0f38a9718a8b`):
  - D2 に台帳の確かめ(`checkLedger`)がなく、D6 の「gitignore されたファイルは止める理由にしない」に台帳のディレクトリの例外がない。cross-review の 2 回目の指摘で足し、Progress に書いて本文は変えていない(verify の C3-D5)。
  - AC1〜AC6 の checkbox と、Progress の「パイプライン cycle 2」「PR created」は未チェックのまま。/pr の前の状態。
- release のノート: sync-docs の report の「`/pr` と release に渡すこと」に案がある。要点は、`ralph org merge` が merge のあとに worktree・ブランチ・記録を消すこと、消すと失うものがあれば何も消さず `CLEANUP:` の行で知らせて exit 0 のままなこと、`ralph org cleanup` が足されたこと。
- 戻し方: この PR を revert する。台帳に増えるのは `worktree_cleaned` の行だけで、古い binary は読まずに無視する。古い binary で merge したブランチは、新しい binary の `ralph org cleanup` で片付く。

## 2. 読む順

1. 計画の D2(確かめ 0〜6 と再開)と D3。これを読んでから `cleanup.go` に入ると、コードの注釈と対応が取れる。
2. `internal/org/cleanup.go`(3 節)。削除の判断はここに集まっている。
3. `internal/org/merge_run.go` の呼び出し、`internal/org/mergeorigin.go`、`internal/org/mergeorder.go`(4 節)。
4. `internal/cli/org_cleanup.go`、`internal/cli/org_merge.go`(5 節)。
5. 雛形・skill・help・文書(6 節)。写しの 8 files は 1 面を読めば足りる。
6. テスト(7 節)。

## 3. 片付けの本体(`internal/org/cleanup.go`)

- 入口は 2 つ。`CleanupFeatureWorktree`(`:185`)は `regate.lock` を 1 s 待って自分で取り、前の regate が残した marker を空にしてから続ける。取れなければ「another ralph org regate, merge, or cleanup runs」で断る。`cleanupFeatureWorktreeLocked`(`:222`)は lock を持つ呼び手用で、merge が使う。どちらも先に `checkCleanupParams`(`:255`)を通り、Head が commit SHA でなければ断る。確かめ 5・6 は Head との比べなので、Head が分からないまま進めないため。
- 確かめの順(`cleanupFirstCheck` の `:289` から `run` の `:392`)。表の Manual は、断りが「手で消すコマンド」を返すかどうか。

| 確かめ | 場所 | 断るとき | Manual |
|---|---|---|---|
| 0 lock | `:192` | 別の regate・merge・cleanup が lock を持つ | なし |
| 1 結びつきと org | `:289`、marker は `:319` | 完全な結びつきがない(gate の記録だけを含む)、org_id が `ValidateIdentifier` を通らない、org が走っている | なし |
| 台帳 | `checkLedger` `:472` | 台帳のディレクトリが機能の worktree かその中 | なし(次の手は台帳を動かすこと) |
| 2 記録 | `checkRecord` `:640` | 記録の branch・worktree_path・kind が結びつきと違う | 違いによる。ブランチ違いは `git branch -D` だけ。path と kind の違いは script。記録のパスの worktree が台帳を持つときはなし |
| 3 作業ディレクトリ | `checkWorktree` `:683` | ralph を worktree の中から打った | なし |
| 4 汚れ | `checkClean` `:726` | rebase か merge の途中、`status --porcelain --untracked-files=all` が空でない | なし |
| 5 ブランチの先頭 | `checkBranchTip` `:771` | ローカルのブランチの先頭が merge された head と違う | なし。ブランチがなく worktree だけがあるときは script |
| 6 チェックアウト | `checkCheckout` `:850` | worktree が記録のブランチにおらず、その HEAD が merge された head でない | なし |

  - 台帳の確かめは記録を読む前に置いてある(`:397`)。D2 の本文では確かめ 3 の一部だが、断りの文が台帳を消すコマンドを名指さないようにするため、確かめ 2 より前に打つ。
  - 確かめ 4 は ignored のファイルを数えない(D6)。script と `git worktree remove` が今もそう扱うため。ふつうの台帳は main worktree の `.harness/state/org` にあり、機能の worktree の外なので当たらない。`--state-dir` か `RALPH_ORG_STATE_DIR` が機能の worktree の中を名指すと、`.gitignore` が `.harness/state/` を名指すので確かめ 4 を素通りし、台帳が worktree と一緒に消える。`checkLedger` はこの穴のためにある(cross-review 2 回目の P1)。
  - `pathWithin`(`:516`)は名前で一致しなければ、祖先を `os.SameFile` でたどる。macOS の既定の APFS は大文字小文字を区別せず、文字列の前置の比べでは同じディレクトリを見落とすため(C3-M1)。台帳と確かめ 3 の作業ディレクトリの比べが使う。
- 再開(`run` の `:425-452`)。script は worktree、ブランチ、記録の順に消し、止まると前の段だけが消えた状態が残る。残ったものにだけ確かめをかける。
  - ディレクトリがあれば `checkWorktree`。
  - ディレクトリがなく git の登録だけ残るとき(`checkRegistration` `:885`)は、登録が記録のブランチにいてそのブランチが残っていることを求め、通れば確かめ 5 を `update-ref` の戻し方で打つ。登録を外すのはそのパスだけの `git worktree remove` で、`prune` は使わない(ほかの worktree の登録にも効くため。`remove` の `:965-977`)。
  - 登録がなくブランチだけ残れば確かめ 5。記録だけが残れば、確かめ 1・2 のあとに記録だけを消す。
  - 記録がなければ何も消さない(`recordGone` `:599`)。残ったものがあれば Manual に `git worktree remove` と `git branch -D` を返し、何も残っていなければ `NothingLeft`。
- 登録の断り(`registrationAtRisk` `:919`)。`git worktree remove` は登録の HEAD と reflog を捨てる。detached、別のブランチ、消えたブランチ、HEAD がゼロの登録は断り、HEAD を残す方法を理由に入れる。残す先は `<ブランチ>-unmerged-<先頭 7 文字>`(`keptBranch` `:814`)で、HEAD がゼロなら reflog の場所を示す。ralph は登録を変えられず打ち直しても同じ断りになるので、Manual(`git worktree remove <wt> && <script>`)を返す。
- 消す・記録する(`remove` `:958`、`record` `:998`)。ctx が済んでいれば何も消さずエラー。登録だけ残る worktree の `git worktree remove` が失敗したら `Failed`(何も消していない)。次の `c.worktrees.Cleanup`(`:978`)が `--force-branch` を渡す唯一の道で、ここまでの確かめを通らないと来ない。script が失敗したら `Failed` と Manual を返し、消した分は消えたまま。成功したら `worktree_cleaned`(`branch= head= via= worktree=<quote>`)を manifest のロックの下で追記する。消したあとなので ctx の取り消しは外す。追記に失敗したら `Cleaned` のまま `Recorded` を false にしてエラーを返す。
- 断りの文は 1 行で、`PrintableInboxText` を通す。文の中のコマンドは `cleanupGitCmd`・`cleanupShellWord` で語ごとに quote する。
- 見てほしいところ:
  1. 何かを消す道が `remove` の 2 か所(登録の `git worktree remove` と `worktrees.Cleanup`)だけであること。
  2. 確かめ 5・6 の基準が Head であること。出どころは、merge では gate の head(Merge が PR を MERGED かつその head と読み直した値)、cleanup では gh の `headRefOid`。squash merge のあとも PR の head は変わらないので、`git branch -d` が止まる場面でも照合できる。
  3. 打ち直せば続く断り(3〜6)に script の Manual を付けていないこと。付けると、先の確かめを飛ばして `-D` する道ができる。
  4. marker は確かめ 1 を通ったときに manifest のロックの下で書かれ、lock を放すまで残る。確かめ 2 以降で断ったあとも、lock を持つ merge の中では残る(`TestCleanupFeatureWorktreeLocked_RefusalKeepsTheMarkerWhileTheLockIsHeld`)。
- 固定しているテスト(`internal/org/cleanup_test.go`):
  - 実 script と bare origin で squash merge、origin のブランチ削除のあとに片付く: `TestCleanupFeatureWorktree_SquashMergedBranch`(`:357`)、ignored は一緒に消える(D6): `TestCleanupFeatureWorktree_IgnoredFilesGoWithTheWorktree`(`:382`)。
  - 確かめ 3〜6 の各断りが何も消さず、記録せず、lock を放す: `TestCleanupFeatureWorktree_RefusedInTheWorktree`(`:469`)。確かめ 5 の head が手元にないときの fetch は `TestCleanupFeatureWorktree_HeadNotHereNamesItsFetch`(`:753`)、退避の名前は `TestCleanupFeatureWorktree_MovedBranchKeptUnderItsTip`(`:813`)。
  - 確かめ 0・1・2: `TestCleanupFeatureWorktree_RefusedAtTheLedgerAndTheRecord`(`:1367`)、`TestCleanupFeatureWorktree_RecordOfAnotherPathOrKindNamesTheChecks`(`:1485`)、`TestCleanupFeatureWorktree_RefusedWhileRegateLockIsHeld`(`:1583`)、marker が同じ org_id の spawn を断って別の org は通す: `TestCleanupFeatureWorktree_MarkerRefusesSpawnOfTheOrg`(`:1281`)。
  - 台帳: `TestCleanupFeatureWorktree_RefusesTheWorktreeThatHoldsItsLedger`(`:657`)、`TestCleanupFeatureWorktree_RecordWhoseWorktreeHoldsTheLedgerNamesNoManual`(`:1532`)、`TestCleanupPathWithin_ComparesByFileIdentity`(`:1746`)。
  - 再開と登録: `TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft`(`:875`)、`TestCleanupFeatureWorktree_GoneWorktreeRegisteredOffItsBranchRemovesNothing`(`:975`)、`TestCleanupFeatureWorktree_RecordGoneRemovesNothing`(`:1108`)、`TestCleanupFeatureWorktree_RecordGoneNamesADetachedRegistrationsHead`(`:1178`)。
  - 失敗と中断: `TestCleanupFeatureWorktree_ScriptFailureRecordsNothing`(`:1239`)、`TestCleanupFeatureWorktree_InterruptedBeforeTheRemoval`(`:1258`)。
  - 断りの文のコマンドが書かれたとおりに動く(空白を含む repo で `sh` に打たせ、最後に次の cleanup が全部の確かめを通る): `TestCleanupFeatureWorktree_RefusalCommandsRunAsWritten`(`:1665`)。

## 4. merge の続きと、origin に固定した PR の読み方

- `merge_run.go:259-266`: `mergeLocked` は `mergeAndRecord` のあと、`res.Recorded` で `p.RepoRoot` があるときだけ `mergeCleanup`(`:290`)を呼ぶ。ctx が済んでいれば呼ばず、`Refused` に `ralph org cleanup --branch` の打ち方を入れる(`:277`)。片付けのエラーは `MergeResult.CleanupErr` の文字列に入れ、Merge のエラーにはしない。lock は片付けの間も持ったままで、`Merge` の頭の「marker を空にして、書かない」(`:233`)は「`pr_merged` を記録するまで」に直した。同じ注釈が `lockfile.go`・`regate.go`・`spawn.go` にもあり、`regateSpawnErr` の断りの文は「regate が走っているか、merge か cleanup が片付けている」と言う。
- `MergeParams.RepoRoot`(`merge.go:95`)が空なら片付けない。org 層の既存の merge のテストは script のない `/repo` を使うため。渡すのは CLI だけ。
- `mergeorigin.go`: merge の `readOrigin`(`merge.go:360`)が持っていた「`git remote get-url origin` から host と owner/repo を読む」を `ReadMergeOrigin` に切り出した。失敗の種類は `MergeOriginProblem` で返し、`readOrigin` が前と同じ文に直す。`MergeOrigin.Arg()` は gh の `-R` の値、`hasPR` は PR の url を host と owner/repo で比べる。
- `mergeorder.go`: `MergeOrderParams.Origin`(`:175`)があると `gh pr list` に `-R` を渡し(`:662`)、返った PR の url が origin のものか比べ(`:685`)、違えば全ブランチの PR を使わず理由を `PRLookupWhy`(JSON に出さない)に入れる。ゼロ値なら前の動きのまま。`ralph org merge-order` の一覧は変わらない。もう 1 つ、`MergeOrderBinding.BoundTo`(`:270`、設定は `:517`)を足した。org_id が別の機能に結びつけ直されたあとの古い結びつきで、`--json` に `bound_to` が出る(結びつけ直しのときだけ)。
- 見てほしいところ:
  1. `ralph org cleanup` が PR を origin の repository に固定する理由。固定しないと、`gh repo set-default` や upstream の remote で gh が別の repository の PR を返し、その MERGED を根拠にこの repo のブランチを `-D` しうる(cross-review 1 回目の指摘)。
  2. merge の `readOrigin` の置き換えが、断りの文を変えていないこと。既存の merge のテストと `TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot`(`merge_test.go:636`)、`TestMergeOrigin_GetURLThatFailsTimesOutOrIsInterrupted`(`:675`)が見る。
  3. 片付けで失敗しても、記録済みの merge は取り消さないこと。
- 固定しているテスト:
  - `TestMerge_CleansUpTheFeatureWorktreeOnceRecorded`(`merge_run_test.go:813`)、`TestMerge_RefusedCleanupKeepsTheMerge`(`:855`)、`TestMerge_RunsTheCleanupOnlyAfterARecordedMerge`(`:923`。RepoRoot なし、merge を断った、記録できなかった、のどれでも呼ばない)、`TestMergeCleanupInterrupted_QuotesTheBranchInItsCommand`(`:999`)、`TestMerge_CleanupOutcomesWithFakeWorktrees`(`:1022`)。
  - `TestMergeOrder_PinnedOriginReadsThePRsInItsRepository`(`mergeorder_test.go:2224`)、`TestMergeOrder_PinnedOriginLeavesEveryPROutForAPROfAnotherRepository`(`:2264`)、`TestMergeOrder_PRLookupWhySaysWhyGHsAnswerWasNotUsed`(`:2310`)、`TestMergeOrder_BindingBoundToNamesTheOrgsLaterBranch`(`:1177`)。

## 5. CLI(`internal/cli/org_cleanup.go`、`org_merge.go`)

- `ralph org cleanup` の流れ(`org_cleanup.go:131-176`): `--org-id` と空の `--branch` を断る → 古い台帳のガード(mutating 扱い、`:141`)→ main worktree の root(`org.FeatureRepoRoot`、`:155`)→ origin を読む(`cleanupReadOrigin` `:162`。読めない、GitHub を名指さない場合は、台帳を読む前に何も消さず exit 1)→ `MergeOrder{All: true, Origin}`(`:166`)→ `cleanupSteps`(`:249`)→ `runCleanupSteps`(`:348`)。
- `cleanupSteps`: 結びつきがあり、最新の結びつき(`BoundTo` が空)で、gh が MERGED と読むブランチを `headRefOid` で片付ける。PR を読めなかったブランチ(failed、limit_forks、skipped)は黙って飛ばさず `SKIPPED:` の行にする(`cleanupLookupWhy` `:321`)。PR が open・closed・なし、gate の記録だけ、片付け済み(`NothingLeft`)は行を出さない。`--branch` は `cleanupNamedStep`(`:285`)が、台帳にない、結びつきがない、PR がない、PR を読めない、MERGED でない、を断る。
- `runCleanupSteps`: ブランチごとに ctx を見る。最後のブランチの片付けの途中で割り込まれたときは、`finish()` より先に割り込みのエラー(exit 1)を返す(`:365`。cross-review 2 回目の P2)。`CLEANED:` の行は先に出ている。
- `org_merge.go`: `mergeCleanupRoot`(`:279`)が repo root を求める。見つからなければ片付けを呼ばず、`Refused` つきの結果を作って(`:248-253`)`CLEANUP: skipped (… main worktree … was not found …)` を出す。`printMergeCleanup`(`:365`)が `done` と `skipped`、`CLEANUP MANUAL:` を出す。`printMergeResult` の終了コードの決め方は変えていない(`err` か `Failed`・`Refused` のときだけ 1)。
- `org.go` は登録(`:81`)、`--org-id` の説明、`ralph org --help`、`start` と `spawn` の help に「片付けの間は断られる」を足した。`org_regate.go` は help の「regate, merge, or cleanup」。regate の worktree がないときの案内(`internal/org/regate.go:564-571`)は、merge 済みなら `ralph org cleanup --branch`、閉じた PR なら script の cleanup に `--force-branch`。
- 見てほしいところ:
  1. 終了コードの対称。merge は片付けの結果によらず 0(記録した merge)、cleanup は 1 本でも `SKIPPED:` なら 1。
  2. `cleanupSteps` が PR を読めないブランチを行にして残すこと。黙って飛ばすと、片付けが終わっていないのに exit 0 になる。
- 固定しているテスト(`org_cleanup_test.go`): `TestOrgCleanup_CleansUpABranchMergedOnGitHub`(`:119`)、`TestOrgCleanup_WithoutGHRemovesNothing`(`:177`)、`TestOrgCleanup_BranchWhosePRIsNotMergedIsRefused`(`:203`)、`TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing`(`:233`)、`TestOrgCleanup_OriginThatNamesNoGitHubRepositoryIsRefused`(`:280`)、`TestOrgCleanup_LeavesOutCleanedAndReboundBranches`(`:449`)、`TestOrgCleanup_InterruptedDuringTheLastCleanupExitsOne`(`:634`)、`TestCleanupSteps_PicksTheMergedBranches`(`:707`)、`TestCleanupPrinter_EachOutcome`(`:886`)。merge 側は `org_merge_test.go` の `TestOrgMerge_CleansUpTheFeatureWorktree`(`:614`)、`TestOrgMerge_NoMainWorktreeSkipsTheCleanup`(`:688`)、`TestPrintMergeResult_EachOutcome`(`:333`)。

## 6. 雛形・skill・help・文書

- leader の雛形 `internal/org/prompts/leader.md:486-492`: 手順 15 の「worktree とブランチは消さない」に、消すのは `ralph org merge` か `ralph org cleanup` と足した。未コミットか未追跡が残ると断られるので、手順 9・10・12・14 のとおりに何も残さない、とも書いた。固定は `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`prompts_test.go:493`)。
- `/org` skill: 動詞の表の `merge`(`:243`)と `cleanup`(`:244`)、start の節、「org の終わり方」(`:989` から。merge のあとの後始末、確かめの一覧、手で消すコマンドの場面、gitignore の段落が `:1146` 以降)。`/director` skill: `CLEANUP:` の読み方(`:885-901`)、「GitHub で merge した PR の後始末」の 5 手順(`:989` から)、「してはいけないこと」に、頼まれないまま MANUAL のコマンドを打たない(`:1060`)。どちらも `.claude`・`.agents`・`templates/base/.claude`・`templates/base/.agents` の 4 面に同じ文があり、`.agents` は `sync-skills.sh` で作り、`templates/base` は `cp` で写した(`check-skill-sync.sh` と `check-sync.sh` で一致を確かめる)。固定は `TestOrgSkill_PinsTheCleanupAfterAMerge`(`org_skill_send_sync_test.go:269`)、`TestDirectorSkill_PinsTheCleanupAfterAMerge`(`director_skill_sync_test.go:1044`)。
- 確かめの一覧と「手で消すコマンドが出る場面」の文は、`ralph org cleanup --help`、`ralph org merge --help`、`/org` skill、`/director` skill、README(`:282-284`)、仕様 FR-8 の注の 6 か所にほぼ同じ形である。確かめを足すと、この 6 か所とそれぞれの固定のテストが動く。help の固定は `TestOrgCleanup_Help`(`org_cleanup_test.go:1018`)、`TestOrgMerge_Help`(`org_merge_test.go:1039`)、`TestOrgStartSpawnAndMergeOrder_HelpFollowTheCleanupAfterAMerge`(`org_cleanup_test.go:1101`)。
- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md:134` に、FR-8 の注として「merge のあとの後始末(#263)で決めたこと」を足した(1 行)。AGENTS.md の `internal/org/` の行に `cleanup.go` と `mergeorigin.go`、quality-gates の watch gate の行(root と template の 2 面)に `cleanup`、herdr の recipe(2 面)に片付けの案内。
- tech-debt に 6 行(`docs/tech-debt/README.md:386-391`)。800 行を越えたファイル、script に時間の上限がないことと start の窓、2 つの quote 関数の差、確かめ 5 だけが fetch を案内すること、など。

## 7. テストの見方

- org 層は、実 script と bare origin の repo(`newWorktreeCleanupFixture` `:44`、`squashMerge` `:336`)で片付く道を、fake の worktrees(`newWorktreeCleanupFakeFixture` `:1326`)で断る道を打つ。変異は `go test -overlay` で打ち、cycle 1 は 26 個中 25 個が落ち(残る 1 個は等価)、cycle 2 は 13 個、cycle 3 は 10 個がすべて落ちた。
- 大文字小文字の subtest 6 つ(`pathWithin` の C3-M1 を見る)は、大文字小文字を区別する volume では `t.Skipf` で飛ぶ。CI は `ubuntu-latest` だけなので、CI では打たれない。手元の macOS では打たれ、名前だけの比べに戻した版を落とす。
- テストの数は、`internal/org` が 1,281 本、`internal/cli` が 880 本(test の cycle 3、トップレベル)。

## 8. 直しの経過

- 計画どおりのスライス: S1 e49cddb5(org 層)、S2 b8209a37(merge の続き)、S3 87b6e72d(`ralph org cleanup`)、S4 cb7ed292 と S4b 111c62a2(雛形・skill・help・lock の断りの文)、S5 c0383ffe と 7d3e3f9f(文書)。
- self-review の直し(F1〜F4): F1 a3af6114(打ち直せば続く断りから Manual を外す。M2)、F2 cbaa92f4(skipped の読み方。M1)、F2b 794aee4d(ブランチの quote)、F3 5ddf5532(断りの文が安全なコマンドを名指す)、F4 3dfd434e(退避の名前に tip を入れる)。
- cross-review 1 回目: S-X1 8e46db0f(PR の読みを origin に固定)、S-X2 d3452a9d(消えた worktree の登録が記録のブランチにいるときだけ進む)、その review の直し 54fa6351。
- cross-review 2 回目: S-X3 4d7cbfd8(台帳の確かめ、最後のブランチの途中の割り込みは exit 1)、S-X4 9dd79b94(`pathWithin` がファイルの同一性で比べる)、7f91e08f(台帳を動かす手順の文)、sync-docs 502f661c。
- main の取り込み: b164744f(#280〜#285)、65a7b067(#286)、7b3efd9f(#287、衝突は 8 ファイルで、行ごとに片側だけの変更はその側、両側の新しい行は両方を取った)。

## 9. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(verify の cycle 3、HEAD 995f8b51: gofmt、go vet、golangci-lint 0 件、staticcheck、`check-sync`(IDENTICAL 172)、`check-skill-sync`(14 本)、secret scan)
  - `./scripts/run-test.sh` rc 0(test の cycle 3、HEAD be6415c2: shell 45 本、Go 9 package)。cleanup と merge のテストを絞って coverage 付きで打った回は、`internal/org/cleanup.go` が 89.6%、`internal/cli/org_cleanup.go` が 96.0%
  - sync-docs の cycle 3(502f661c)と 7b3efd9f の main 取り込みのあとは、build・vet・`check-skill-sync`・`check-sync` と、絞ったテスト(`internal/org` の `Cleanup|Merge|Skill|Prompt|StateDir|Feature`、`internal/cli` の `OrgCleanup|OrgMerge|Help|Config` ほか)が通った
- 確かめていないこと・残っていること:
  - 上の絞ったテストのあとに、`run-test.sh` を全体で打ち直してはいない(HEAD 49aee225 では未実行)
  - 実際の gh と、ssh の別名の origin(merge と同じく `SKIPPED:` になる)。実機の herdr での leader と director が、手順 15 と skill の文のとおりに動くこと
  - 台帳を動かしたあとの打ち直し(`--state-dir` か `RALPH_ORG_STATE_DIR`)が確かめ 4 以降に進むこと。コードを読んだだけ
  - Known gaps(PR の本文に書く):
    - 別の org の座席が `--cwd` で機能の worktree の中にいると、確かめ 1 は見ない(結びつきの org だけを見る。cross-review 3 回目の P2、WORTH_CONSIDERING)
    - `StartFeature` の `regateSpawnErr` は manifest のロックの外で打たれる。並行した start が `ensure` を通ったあとに片付けが終わると、座席が消えた作業ディレクトリを渡される(同上の P2、tech-debt の L6(b)。その行の「start の最後の spawn は Spawn の確かめで断られる」は、lock を放したあとには効かないので直す)
    - `runWorktreeScript` は script を時間の上限なしで打つ(L6(a))
    - 大文字小文字の subtest が CI で skip になること。NFC と NFD の違いを打つテストはない
    - `worktree.useRelativePaths=true` の git では、登録の gitdir が相対パスになり、HEAD がゼロの登録の案内の文字(`<worktree>/.git`)と合わない
