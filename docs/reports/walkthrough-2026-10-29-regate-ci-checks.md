# Walkthrough: regate-ci-checks(`ralph org regate` の検査を、手元の gate のスクリプトから PR の CI の結果に切り替えられるようにする)

- Date: 2026-10-30(plan の日付は 2026-10-29)
- Plan: docs/plans/active/2026-10-29-regate-ci-checks.md(/pr で archive に移す)
- Issue: #264(org: regate の手元の test を CI の結果に置き換えられるようにする)
- PR: 未作成(/pr で作る)
- Branch: feat/regate-ci-checks(この walkthrough の前の HEAD は 077e3b5b。merge を除く 34 commits と、origin/main を取り込んだ merge commit 137ec668 の 1 つ)
- Base: c03647f9(origin/main、「Merge pull request #288」= org-merge-cleanup)。分岐元も同じ
- 読む範囲: `git diff origin/main...HEAD` は 64 files、+7043/-366(HEAD 077e3b5b。この walkthrough は数えていない)。Go の本番コードが 18 files(+1967/-245)、Go のテストが 20 files(+4007/-20)、skill の写しが 8 files(+384/-88)、記録(計画・report・insight event)が 8 files(+651)、残りの文書と設定が 10 files(+34/-13)。動きを決めるのは `internal/org/regate_ci.go`(新規 901 行)、`internal/org/prchecks.go`(新規 223 行)、`internal/org/regate.go`(+303/-71、いま 995 行)、`internal/org/regate_merge.go`(+65/-8)、`internal/org/regate_git.go`(+46/-4)、`internal/org/gate_record.go`(+50/-25)、`internal/cli/org_regate.go`(+184/-35)。テストの大半は新規の 4 files(`regate_ci_test.go` 767 行、`regate_ci_merge_test.go` 630 行、`regate_ci_wait_test.go` 726 行、`regate_ci_wait_lock_test.go` 377 行)と `prchecks_test.go`(320 行)、`internal/cli/org_regate_test.go`(+518/-5)。skill の 8 files は 2 つの skill の同じ文を 4 面に写したもの
- 後工程: パイプラインは 1 回(`cycle-count.json` は 1)。self-review は 2 run(run 1 は CRITICAL と HIGH が 0、MEDIUM 1、LOW 8。run 2 は MEDIUM 以上が 0、LOW 5。どちらも Merge 可)、verify は pass、test は pass、sync-docs は 1 回、cross-review(codex)は 1 回で所見 0(ACTION_REQUIRED 0、WORTH_CONSIDERING 0、Case C)。記録だけの修正の例外を使った記録は triage report にない

```
何を変えるか → 読む順 → 設定と記録 → PR の checks の読み取り → ci の regate の本体 → 待ち → 古い記録と PushOnly → CLI → skill・help・文書 → テスト → 直しの経過 → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで:
  - `ralph org regate --branch <ブランチ>` は、disband 済みの機能の org のブランチに origin の既定のブランチを merge commit で取り込み、`run-static-verify.sh` と `run-test.sh` を手元で走らせ、`via=regate` の `gate_passed` か `gate_failed` を記録し、通れば force なしで push する。
  - push すると、この repo の CI(`.github/workflows/verify.yml`)も `./scripts/run-verify.sh` を走らせるので、regate のあとにフルの test が手元と CI で 2 回走る。手元の 1 回は 640〜1,090 秒かかる(issue の記述。この walkthrough では測っていない)。
- この PR から:
  - 設定 `[org.gate].regate_checks` を足した。値は `"local"`(既定。動きは前のまま)か `"ci"`。ほかの値と空文字は `config.Load` が断る。
  - `ci` の regate は、fetch のあとで origin・gh・PR を確かめ(merge の前)、merge commit を作り、gate のスクリプトを走らせずに force なしで push し、push した head の PR の checks を読んで記録する。全部通れば `gate_ci_passed`、落ちた check があり pending がなければ `gate_ci_failed`(reason `ci_failed`)で、どちらも `via=regate_ci`。
  - checks がまだ終わっていないとき(PR の head が push した commit になっていない、pending の check がある、check が 0 件)は何も記録せず、`CHECKS: pending (<理由>)` と `NEXT:` の行を出して終了コード 6 で終わる。次に regate を打つと、HEAD が origin の先頭で既定のブランチを含み passed の記録がない形なので、merge せずに PR の checks を読み直す(D6)。
  - `ralph org regate --wait-checks <時間>` で、checks が終わるまで待てる。待つあいだは `regate.lock` を外し、読むだけで、終わったら 1 回だけ取り直して確かめ直してから記録する。
  - `ralph org merge-order` と `ralph org merge` と regate は、`gate_ci_*` を `gate_passed`・`gate_failed` と同じように読む。
  - この repo のルートの `ralph.toml` は `ci`、`ralph init` の template は `local`。
- issue との違い(issue #264 の本文と比べた):
  - issue は「`via` で手元の gate と区別する」と書く。この PR は `via=regate_ci` に加えて event の名前も分けた(`gate_ci_passed`・`gate_ci_failed`)。この PR を revert して古い binary に戻したとき、古い binary が `regate_ci` の pass を普通の pass と読み、意図して戻された origin のコミットを PushOnly で押し直さないため(plan advisory の指摘 1。base の `mergeorder.go:481` は `case EventGatePassed, EventGateFailed` だけを選ぶ)。
  - issue は、gh がない・PR が見つからない・checks が 1 つもない場合に、断るか手元の gate に戻るかを計画に任せた。計画は、gh がない、PR がない、origin が GitHub を名指さない、PR が open でない、base が既定のブランチでない、のどれでも merge の前に何もせず断る(exit 1)。手元の gate には戻らない(設定の意図を黙って変え、`ci` を選んだ repo でフルの test が突然走るため)。checks が 0 件のときは断らず pending(exit 6)。すべて skipping で pass が 1 つもないときは記録せず exit 1(コードの失敗ではないので `ci_failed` にしない)。
  - issue は「待つ時間の上限は `ralph org merge --wait-checks` に合わせる」と書く。`MergeWaitChecksRefusal`(0〜6 時間)と間隔(30 秒)を共用し、既定の 0 は 1 回だけ読む。`local` で 0 より大きい値は断る。
  - issue にないもの: 終了コード 6、`ci` の pass を PushOnly に入れず断ること(AC10)、push の失敗を origin の先頭の読み返しで 3 つに分けること(D3)。
  - issue は「この repo のルートには今 `ralph.toml` がない」と書くが、base にはすでにある(#265 が足した、`git show origin/main:ralph.toml` で確認)。この PR は `[org.gate]` の表を足し、ヘッダと `scripts/check-sync.sh` の `KNOWN_DIFFS` のコメントを直しただけ。
- 計画の本文との違い(承認の digest `994880bfa86d` は最後まで変わっていない。以下は Progress に書いて本文は変えていない):
  - local の `UP_TO_DATE:` の行(`internal/cli/org_regate.go:360`)が「a gate_passed of HEAD」から「a pass of HEAD」に変わった(base の `:224` で確認)。local の regate も `gate_ci_passed` を UpToDate と読む(`regateBranch.passedOn` `regate.go:662`)ので、記録の種類によらない文にした。動きは変わらず、org 層のテストに消えた行も書き換えた行もない(`regate_test.go` +51/-0、`regate_merge_test.go` +114/-0)。CLI のテストの期待は 2 か所を書き換えた(`TestOrgRegate_MergesPushesAndThenIsUpToDate` と `TestOrgRegate_PrintRegateResult`)。AC1 の括弧「今の regate のテストが変えずに通る」は、この 2 か所について字義どおりには満たさない。verify は AC1 を Partly とし、動きが同じなので AC1 に印を付けた。
  - `RegateOutcomeChecksPending` は S4b(688d4d6a)で入った。pending は何も記録しないので、`Gated`・`Merged` の「記録した」という約束を壊さないために別の outcome にした。終了コード 6 への対応(`ErrRegateChecksPending`、`internal/cli/exitcode.go:43-44`)は S6(a665eba8)で、計画の outline どおり。
  - 待ちは lock なしで読むだけにし、終わりに 1 回だけ取り直す(F1 e40b3466)。S5(33be080c)は 30 秒ごとに取り直していて、1 秒で取れなければ断って終わった(self-review L3)。取り直した lock の下でまた pending なら、待たずに exit 6 で終える。D5 の文面はこの形を含むが、書き分けてはいない(verify が Progress の F1 の記録を確認)。
  - 待つあいだに PR の head が試した head でも merge の前の head でもないとき、lock なしで `git ls-remote` を打って origin の先頭を読む(F1b)。2 つのコミットに分かれる: f32fe553 は「動いたら止める」だけで、GitHub の更新の遅れでも待たずに終わる穴があり、02219639 が origin の先頭が HEAD のままなら眠り続けるように直した。02219639 の subject は「fix: read origin's tip before a ci regate's wait stops at another PR head」、本文は空。handoff が指定した message と違う(handoff は未読。指定の中身は未確認)。
  - 事前の確かめ(`regateCIPreCheck`)は、計画の D2 では fetch の前に書いてあったが、fetch の直後に置いた(S4a 2ab52083)。fetch が変えるのは `refs/remotes/origin/*` だけ。UpToDate と push だけの打ち直しでも、gh と open な PR が要る(`Regate` の概要に注釈 13d19f3d)。
  - D4 の「落ちた check と pending の check が両方あるとき: pending と同じに扱う(merge と同じく…)」の括弧は正確でない。merge は落ちた check が 1 つでもあれば pending があっても止める(`mergeRefusalChecksNotPassed`)。regate は計画どおり、すべて終わるまで待つ(S3 ab1b0d43 の記録、verify の確認)。
  - `ci` の regate も、走らせない 2 つのスクリプトが実行できるファイルとしてあることを確かめる(`regateCheckWorktree` の `gateCheckScripts`、`regate.go:844`)。計画に記述がない(verify I3)。
  - 計画の `mergeorder.go:458`・`watch.go:1401` は計画の base での行で、HEAD では `:487`・`:1405`(verify I2)。
  - AC1〜AC10 の checkbox は 077e3b5b で付けた。Progress の「PR created」は未チェックで、/pr の前の状態。
- release のノート: 計画の案は 1 行で、`[org.gate].regate_checks = "ci"` で regate が手元の test の代わりに PR の CI の結果で記録を書けるようになった、と言う。
- 戻し方: この PR を revert する。台帳に増えるのは `gate_ci_passed` と `gate_ci_failed` の行だけで、古い binary は event の名前で gate の記録を選ぶので読み飛ばす(base のソースを読んだ。古い binary での実測は未実施)。merge-order はそのブランチの前の記録で分類する。

## 2. 読む順

1. 計画の D2〜D7 と Critical forks の 3 つ。注釈が D 番号を引くので、先に読むとコードと対応が取れる。
2. `internal/config/config.go`(3 節)。
3. `internal/org/prchecks.go` と `internal/org/merge.go` の差分(4 節)。
4. `internal/org/regate_ci.go` を `regateCIPreCheck`(`:66`)から `regateChecksWithWait`(`:677`)まで。あわせて `internal/org/regate.go` の `Regate`(`:462-562`)と `internal/org/regate_merge.go:277`(5 節、6 節)。
5. 記録と読み手: `internal/org/gate_record.go`、`seat.go`、`mergeorder.go`、`regate_merge.go:83-149`(7 節)。
6. `internal/cli/org_regate.go`、`org_gate.go`、`exitcode.go`(8 節)。
7. skill・help・文書(9 節)。写しは 1 面を読めば足りる。
8. テスト(10 節)。

## 3. 設定(`internal/config/config.go`、2 つの `ralph.toml`)

- `OrgGateConfig`(`:581-602`)に `RegateChecks`(toml `regate_checks`)。定数は `RegateChecksLocal = "local"` と `RegateChecksCI = "ci"`。`Default()` は `local`(`:660-662`)。`Load` は `local` と `ci` 以外を断る。空文字も断る(`:944-946`、`[org.approval].mode` と同じ形)。表が丸ごとない `ralph.toml` は既定の `local` のまま。
- `templates/base/ralph.toml:269-279` に `[org.gate]`、コメント、`regate_checks = "local"`。`ralph.toml:18-19`(この repo のルート)は `regate_checks = "ci"` で、ヘッダのコメント(`:5`)に理由を足した。`ralph init` で downstream に入る template には、この repo の値のことは書いていない(self-review L7 で消した)。
- regate は打った場所の `ralph.toml`(`--config` があればそのファイル)を読む。org 層は設定を読まず、CLI が `Checks: rt.Config.Gate.RegateChecks` を渡す(`internal/cli/org_regate.go:221-222`)。`ralph config get org.gate.regate_checks` で同じ値が読める。
- ずれの検査: `internal/config/defaults_sync_test.go` の `orgTemplateProbe` と `orgTemplateDrift` に `org.gate.regate_checks` の行。`internal/config/root_ralph_toml_test.go` は、ルートの `ralph.toml` が `ci` であることを要求する。
- 見てほしいところ: 設定を足し忘れる場所の探し方。`orgTemplateDrift` の行、`root_ralph_toml_test.go` の `ci` の要求、`scripts/check-sync.sh` の `KNOWN_DIFFS` のコメント、`.claude/rules/ralph/model-routing.md:151` の key 一覧(S1 の implementer が sync-docs に渡すと記録し、S7 で直した)。

## 4. PR の checks の読み取り(`internal/org/prchecks.go`、`internal/org/merge.go`)

- `ralph org merge` が持っていた gh の読み(`findPR`、`readChecks`、`ghWith`)を、regate からも呼べる関数に切り出した(S3 ab1b0d43)。
  - `runGH`(`:28`): `GH_PROMPT_DISABLED=1`、時間の上限つきで gh を打つ。前の `ghWith` の中身。
  - `findOriginPR`(`:89`): `-R <origin>` に固定して PR を読む。結果の `problem` は Usable・Interrupted・GHFailed・LimitForks・None・Foreign・NotOpen・WrongBase。
  - `readPRHead`(`:116`): `gh pr view --json headRefOid`。
  - `readPRChecks`(`:185`): `gh pr checks` の JSON を total・passed・pending・failed に分ける。出力が JSON でないときと gh の失敗は理由つきで返す。checks がないという gh の文は 0 件として読む。
- merge の側は、これを呼ぶ形に変えた。case の順、文言、`mergeRefusalKind` は前のまま。merge の既存のテストの期待は 1 つも変えていない(self-review run 1 が前後で突き合わせた)。
- 見てほしいところ:
  1. 切り出しが merge の動きを変えていないこと。判定は merge が「落ちた check があれば pending があっても NotPassed」、regate が「pending があるあいだは待つ」と違う。違いは `readPRChecks` が pending と failed を分けて返すことで吸収している。
  2. 台帳から来る値が gh の argv に入るのは、`--head <ブランチ>` と `-R <repo>` の flag の値と、数字の PR 番号だけで、positional はない(self-review run 1 が確認)。
- 固定しているテスト(`internal/org/prchecks_test.go`、新規 320 行): `TestPRFind_ReadsTheBranchsPRPinnedToOrigin`、`TestPRFind_GHThatFailsOrIsInterrupted`、`TestPRHead_ReadsTheHeadOrSaysWhyNot`、`TestPRChecks_CountsTheChecksGHListed`、`TestPRChecks_SaysWhyTheChecksAreUnread`。

## 5. `ci` の regate の本体(`internal/org/regate_ci.go`、`regate.go`、`regate_merge.go`)

- 道の選び方(`Regate` `regate.go:462`)。lock・結びつき・worktree の確かめと fetch のあと、`p.Checks == ci` なら `regateCIPreCheck` を 1 回打つ(`:530-535`)。そのあとの分かれは local と同じ形:
  - HEAD が origin の先頭でない: `regateHeadNotOnOrigin`(7 節)。
  - 既定のブランチの先頭を含まない: `regateMerged`(`regate_merge.go`)。`ci != nil` なら gate のスクリプトを呼ばず `regateCIMerged`(`regate_merge.go:277`)。
  - 含み、passed の記録が HEAD にある: UpToDate。checks は読まない。
  - 含み、passed の記録がない: `ci != nil` なら `regateChecksWithWait`(D6)、なければ `regateGate`(`:550-554`)。`ci == nil` の道は前のまま。
- 事前の確かめ(`regateCIPreCheck` `:66`)。順に、origin の URL が GitHub の repository を名指す(`ReadMergeOrigin`)、gh が PATH にある、`findOriginPR` が open で既定のブランチ向きの PR を origin の repository から見つける。どれかが通らなければ `nothing was merged, pushed, or recorded` と次の手を出して断る。次の手のコマンドは worktree とブランチを `cleanupShellWord` で quote する(self-review L8、69f80d8f)。
- merge の道(`regateCIMerged` `:183`)。`regateCIPush` で push してから `regateChecksWithWait` に進む。push のあとの断りと error は、必ず「merge commit は origin に残る」と言う(`regateCIPushedText` `:211`)。
- push(`regateCIPush` `:245`)。`git push origin <sha>:refs/heads/<ブランチ>` を force なしで打つ。push が失敗したら `regateReadOriginTip`(`regate_git.go:74`、`git ls-remote origin refs/heads/<ブランチ>`)で origin の先頭を読み返し、分ける(D3):

| 読み返した先頭 | 扱い |
|---|---|
| push の前の先頭(`res.PreMergeHead`)のまま | push は届いていない。`regateUndoMerge` で merge commit を外し、何も記録せず、理由と「直してから打ち直す」を出す |
| push した merge commit | 届いていた。`regateCILanded`(`:331`)で `Pushed` にして checks へ進む |
| 子孫、別のコミット、ブランチなし、読めない | 公開されたか言い切れない。HEAD を残し、何も記録せず、`git fetch origin` と `git log` で確かめて打ち直す手を出して断る(`regateCIOriginLog` `:349`) |
| push か読み返しの最中に ctx が終わる | HEAD を残し、「may or may not」の error |
| push の前に ctx が終わる、追跡 ref が読めない | 押していないので undo する |

- checks から記録へ(`readRegateChecks` `:396`、`regateChecks` `:519`)。PR の head を読み、checks を読み、head をもう一度読む:

| 読んだもの | 結果 |
|---|---|
| PR の head が試した head でない | pending(push の直後は merge の前の head のまま。それ以外は「GitHub の遅れか origin が動いたか」) |
| check が 0 件 | pending |
| pending の check がある(落ちた check があっても) | pending |
| checks を読んだあとに PR の head が動いた | pending(「head moved while its checks were read」) |
| 落ちた check があり pending がない | `gate_ci_failed`、reason `ci_failed`。`res.FailedChecks` に落ちた check |
| すべて pass か skipping で pass が 1 つ以上 | `gate_ci_passed` |
| すべて skipping | 記録せず断る(exit 1) |
| gh の失敗、時間切れ、JSON でない出力 | 記録せず断る(exit 1) |

  記録は `gateRecord{Head: t.head, Base/BaseRef: 取り込んだ既定のブランチの先頭, Via: regate_ci, Log: ...}`。`log=` は `gh pr checks` の JSON を書いたファイル(`writeGateChecksLog` `:573`)。`recordRegate` が、結びつきと org が最初の確かめのとおりかをもう一度見てから追記する。
- 見てほしいところ:
  1. 記録の前に push していること(local は記録のあとに push する)。公開の取り消しは、origin が元のままと読み返せたときだけ。force push はどこにもない。
  2. PR の head が記録する head と一致することを、checks の前と後の 2 回見ていること(test の変異 m2a・m2b・m2ab)。`gh pr checks` の出力に commit はないので、これがないと古い head の pass を新しい head の記録にしうる(self-review L4)。
  3. 記録の元の読みは、いつも `regate.lock` を持った状態の読みであること(7 節で触れる `assertLockedAtTheRecord`)。
  4. 残る窓は、2 回目の head の読みから記録の追記までと、head が A→B→A と動く形。merge-order の head の比べと merge の読み直しが止める(self-review run 2)。
- 固定しているテスト: `TestRegateCI_RefusedBeforeAnythingChanges`(`regate_ci_test.go:166`)、`TestRegateCI_ChecksOriginGHAndThePRBeforeEveryPath`(`:253`)、`TestRegateCI_CommandsQuoteTheWorktreeAndTheBranch`(`:301`、名指したコマンドを `sh -c` で走らせる)、`TestRegateCI_GateOnlyRecordsFromThePRsChecks`(`:392`)、`TestRegateCI_GateOnlyRecordsNothingWhileTheChecksAreNotDone`(`:486`)、`TestRegateCI_GateOnlyRefusesWhatTheChecksDoNotTell`(`:552`)、`TestRegateCI_UpToDateReadsNoChecks`(`:619`)、`TestRegateCI_LocalChecksRunTheScriptsAndCallNoExec`(`:746`)、merge の道は `regate_ci_merge_test.go` の `TestRegateCI_MergePathPushesTheMergeCommitBeforeTheChecks`(`:71`)、`TestRegateCI_MergePathRecordsTheChecksOfThePushedMergeCommit`(`:124`)、`TestRegateCI_ARefusalAfterThePushSaysTheMergeCommitStays`(`:187`)、`TestRegateCI_AFailedPushIsDecidedByOriginsTipReadBack`(`:248`、7 つの場面。origin が拒む形と `pre-push` hook が git を kill する形を本物の git で起こす)、`TestRegateCI_AFailedPushWhoseUndoFailsNamesWhatEachGitSaid`(`:391`)、`TestRegateCI_AnInterruptOnTheMergePath`(`:449`)、`TestRegateReadOriginTip`(`:606`)。

## 6. 待ち(`regateChecksWithWait` `:677`、`poll` `:756`、`regateCIRecheck` `:807`)

- 始め: `regateChecks` が pending を返し、`ci.wait` に時間が残るときだけ待つ。stderr に 1 行書き、`regate.lock` を外す(`w.held.release()` `:687`)。
- 待つあいだ(`poll`): `min(30 秒, 残り)` 眠って `readRegateChecks` を読む。書かず、lock も取らない。次のどれかで抜ける。
  - pending でなくなった(終わった、または断りになる読み)。
  - PR の head が動いた形(試した head でも merge の前の head でもない、または checks を読むあいだに動いた)で、lock なしの `regateReadOriginTip` が HEAD でない、ブランチがない、読めない、のどれかを返した。origin の先頭が HEAD のままなら GitHub の更新の遅れとして眠り続ける(02219639)。
  - 時間が尽きた: lock を取り直さず、`ChecksPending` の理由に「ralph org regate waited … and that ran out」を添えて exit 6。
  - ctx が終わった: lock を取り直さず、何も記録せず error。merge の道では、merge commit が origin に残ることを error に添える。
- 抜けたあと(`regateCIRecheck`): lock を `regateLockWait`(1 秒)で 1 回だけ取り直す。ほかの regate・merge・cleanup が持っていれば断る。marker を空にし、最初の確かめ(結びつきと走っている org)をやり直して `regateBranchChanged` で最初の読みと比べ、worktree の HEAD が `t.head` か、origin の先頭を `git ls-remote` で読み直して `t.head` かを確かめる。どれかが違えば何も記録せず断る。
- そのあと、ほかの regate が HEAD の pass を記録していれば UpToDate(`LogPath` はその log)。なければ lock の下で `regateChecks` を打ち直して記録する。lock の下でまた pending なら、待ちを再開せず exit 6。
- 待ちの始めと終わりは stderr に 1 行ずつ(`w.line`、`ended`、`stopped`)。
- 時間の検証(`regateWaitChecksRefusal` `:594`): 0〜6 時間の外と、`local` での 0 より大きい値は、lock も台帳も触る前に断る。
- 見てほしいところ:
  1. 眠っているあいだに lock が空いていること、記録の直前に lock を持っていて marker を書き直していること(`assertLockedAtTheRecord`)。
  2. 待つあいだの poll の結果は、待ちを終わらせる合図にしか使わないこと。記録は取り直した lock の下の読みから作る。
  3. 時間切れと割り込みでは取り直さないので、`Regate` の `defer` が lock を二度外すことはない(`regateHeldLock.release` は nil を見る)。
  4. 1 回の poll の読みの失敗(gh の一時的な失敗、`ls-remote` の失敗)で 60 分の待ちが早く終わりうる。lock の下で読み直して pending なら exit 6 になり、`/director` skill は打ち直す。merge の待ちも読みの失敗で終わる。
- 固定しているテスト(`regate_ci_wait_test.go`): `TestRegateCI_WaitChecksWaitsUntilTheChecksAreDone`(`:203`)、`TestRegateCI_WaitChecksOnTheMergePath`(`:294`)、`TestRegateCI_WaitChecksRunsOut`(`:340`)、`TestRegateCI_WaitChecksChecksAgainAfterASleep`(`:392`、確かめ直しの表)、`TestRegateCI_WaitChecksEndsWhenAnotherRegatePassedHEAD`(`:517`)、`TestRegateCI_AnInterruptWhileWaitingForTheChecks`(`:573`)、`TestRegateCI_WaitChecksIsCheckedBeforeAnythingIsDone`(`:685`)。`regate_ci_wait_lock_test.go`: `TestRegateCI_WaitChecksGoesOnWhileAnotherHoldsTheLock`(`:25`)、`TestRegateCI_WaitChecksPendingAgainUnderTheLock`(`:78`)、`TestRegateCI_WaitChecksPendingUnderTheLockAfterGHFailed`(`:111`)、`TestRegateCI_WaitChecksStopsOnceThePRsHeadShowsTheBranchMoved`(`:197`)。

## 7. 記録の読み手と PushOnly(`gate_record.go`、`seat.go`、`mergeorder.go`、`regate_merge.go`、`watch.go`)

- event と via: `seat.go` に `EventGateCIPassed = "gate_ci_passed"` と `EventGateCIFailed = "gate_ci_failed"`。`gate_record.go` に `gateViaRegateCI = "regate_ci"` と `GateReasonCIFailed = "ci_failed"`。`isGateReason` に `ci_failed` を足した(足さないと記録が unreadable になる)。`gateRecord.event()`(`:73`)は via で event の名前を分け、読む側の「`event()` と event の名前が合うときだけ読む」が組を守る。CI の Details を `gate_passed` で書いた行も、その逆も unreadable。Details の形は今の gate の記録と同じ。
- merge-order・merge・regate の読み: gate の記録を event 名で選ぶ箇所は `collectMergeTargets`(`mergeorder.go:452`)の `case`(`:487`)の 1 つで、4 つの event を選ぶように足した。merge-order の `ready` の判定(`classifyGatePassed` `:1153`)と merge は via を見ないので、そこから先は変えていない。regate は同じ収集を通して読む。
- via を見る 3 か所:
  - `mergeRegateFailedDetail`(`mergeorder.go:1108`)に `via=regate_ci` の分かれ。「CI failed (ci_failed) on <head>、regate が push した head。log に checks がある。head は origin にあるので reset するものはない。<次の手> again」。
  - `regateResetHint`(`regate_merge.go:165`)は前から `regate_ci` を対象にしていない。push 済みの merge commit の reset を勧めない。注釈とテストだけ足した。
  - `regateHeadNotOnOrigin`(`regate_merge.go:83`、AC10)。`regate_ci` の pass は push が届いたあとに書くので、そのあと HEAD が origin の先頭でないのは、だれかが origin を戻したか、上に積んだとき。押し直すとその人の作業を戻すので PushOnly に入れない。origin の先頭が HEAD の子孫なら `regateCIPassBehindText`(`:143`、`git fetch` と `merge --ff-only` で早送りしてから打ち直す。self-review L1)、それ以外は `regateCIPassMovedText`(`:129`)で断る。
- 数えないもの: `watch.go:1405` の leader の活動の数え方に `gate_ci_*` は足していない(注釈だけ。regate が書くのは disband 済みの org にだけで、数えると upgrade や rollback で pending の alert の基準がずれる)。`Roster` は org レベルの event として読み飛ばす。
- `ralph org gate` の表示: `printGateFailDetail`(`internal/cli/org_gate.go`)に `gateCIFailure{pr, failed, log}` を足し、`ci_failed` で落ちた check の一覧と log と hint を出す。`ralph org gate` 自身は `gateCIFailure{}` を渡す。
- 見てほしいところ: 書き手 1 か所(`recordRegate`)と読み手 1 か所(`collectMergeTargets`)を `git grep` で数え直したこと(self-review run 1。取りこぼしなし)。`gate_ci_*` の名前を足したのに `isGateReason` や via の分かれを足し忘れる穴が、テストで閉じていること。
- 固定しているテスト: `TestGateRecordDetails_CIRecord`(`gate_test.go:1514`)、`TestMergeOrder_ReadsTheCIRecordsOfRegate`(`mergeorder_test.go:894`。ready・rebase・regate の行、CI と local の記録が台帳の順に新しいほうが勝つこと、名前と中身が合わない行が unreadable)、`TestMerge_MergesABranchWhoseLatestRecordIsAGateCIPassed`(`merge_test.go:468`)、`TestRegate_ReadsACIRecordAsTheBranchsLatestRecord`(`regate_test.go`)、`TestRegate_ACIPassIsNotPushedAgainOnceOriginMovedOff`(`regate_merge_test.go:1498`。子孫の形を含む)、`TestRegate_ACIRecordNamesNoResetKeep`(`:1569`)、`TestOrgMergeOrder_HelpNamesTheCIGateRecords`(`org_merge_order_test.go`)。

## 8. CLI(`internal/cli/org_regate.go`、`org_gate.go`、`exitcode.go`、`org.go`、`cmd/ralph/main.go`)

- `ralph org regate`: フラグ `--wait-checks`(duration、既定 0)。`newOrgRuntime` で設定を読んでから `Regate` に `Checks`・`WaitChecks`・`Stderr` を渡す(`:221`)。範囲の外と `local` での正の値は、org 層が lock と台帳より前に断る。`RALPH_ORG_STATE_DIR` に向けたディレクトリは作られない(verify の probe)。
- 出力(`printRegateResult` `:282`、`printRegateLines` `:345`):
  - 記録したとき: `GATE: pass` または `GATE: fail` と `REASON:`、続けて `CHECKS: passed (PR #n)` か `CHECKS: failed (PR #n)`。fail は空行のあとに落ちた check の一覧と log(`printGateFailDetail`)。push 済みの merge commit は「origin に残る。force しないので直しはその上に積む」と言い、error は `recorded gate_ci_failed (reason=ci_failed) of the merge commit ..., which stays on origin's ...`(exit 1)。
  - checks が終わっていないとき: `CHECKS: pending (<理由>)`、`MERGED:`(merge 後)、`HEAD:`、`BASE:`、`PUSHED:`、`NEXT:`。error は `ErrRegateChecksPending` を包み、`ExitCode` が 6 を返す(`exitcode.go:43-44`)。`GATE:` の行は出ない。
  - 待つあいだにほかの regate が pass を記録したとき: `UP_TO_DATE: while regate waited for the checks of PR #n, another regate recorded a pass of HEAD ...`。
  - 断りで log があるとき: `ci` なら「the JSON of the checks of PR #n it read is in <log>」、local なら「the scripts' output is in」。
  - 知らない outcome には `GATE:` の行を出さずエラーにする(S4b で `case Gated, Merged:` に分けた。13d19f3d がテストで固定)。
- help: `ralph org regate --help` の Long に `ci` の段落(`:123` から)。`--wait-checks` の段落には、止まり方をすべて書いた(checks が終わる、gh が失敗する、PR の head が別のコミットで origin の先頭が HEAD でない・読めない、取り直しの確かめで違えば exit 1。F4)。最後の段落の「regate が走っているあいだ spawn と start が断られる」に、`ci` の待ちの例外を足した(L6)。`ralph org merge-order --help` に `gate_ci_*`。`ralph org spawn --help` と `ralph org start --help` に、待つ `ci` の regate は lock を外しているので断らない、という 1 文(sync-docs a0a8da30)。
- `cmd/ralph/main.go` の注釈に終了コード 6。`internal/cli/org.go` は上の help の文。
- 固定しているテスト(`org_regate_test.go`): `TestOrgRegate_CIGateOnlyPassRecordsFromThePRChecks`、`TestOrgRegate_CIMergePathPushesThenRecordsThePass`、`TestOrgRegate_CIFailureAfterAMergeNamesTheChecksAndKeepsThePush`(`:1070`)、`TestOrgRegate_CIPendingExits6AndTheNextRunReadsTheChecks`(`:1120`)、`TestOrgRegate_CIWaitChecksReachesTheWait`、`TestOrgRegate_CIWithNoPRIsRefusedBeforeTheMerge`(`:1202`)、`TestOrgRegate_WaitChecksRefusedBeforeTheLedgerIsRead`。`exitcode_test.go` に 6 の 2 行(素の error と包んだ error)。

## 9. skill・help・文書

- `/director` skill の「merge のあとの取り直し」(`.claude/skills/director/SKILL.md:953` の節、`:962-1027`): regate を打つ前に `ralph config get org.gate.regate_checks` で値を読むこと、`local` と `ci` の違い、`ci` の事前の確かめ(欠ければ何もせず exit 1)、`--wait-checks 60m`、`ci_failed` は merge commit が origin に残ること、新しい手順 4(終了コード 6 は断りでも失敗でもなく、`GATE:` ではなく終了コードと `CHECKS:`・`NEXT:` の行で見分け、人に聞き直さずに打ち直してよい。2 回続けて 6 なら人に伝える。CI が走らない repo で打ち直しが終わらないのを防ぐため)。`:883` にも終了コード 6 の 1 文。
- `/org` skill: `regate` の行(`.claude/skills/org/SKILL.md:242`。台帳ごとに 1 つの例外、`UP_TO_DATE:` の条件、spawn と start が断られる例外)、「gate の記録」の節(`:1214` の節の `:1237-1246`。event を分けた理由と `ci_failed` の読み方)、「merge してよい順」の節(`:1255` の節の `:1336-1339`、`:1352-1418`。待つあいだの spawn と start の例外、終了コード 6)。spawn の行と start の行にも、待つ `ci` の regate の例外を括弧で足した(sync-docs)。
- どちらも `.claude`・`.agents`・`templates/base/.claude`・`templates/base/.agents` の 4 面に同じ文があり、`check-skill-sync.sh`(14 skill が一致)と `check-sync.sh`(IDENTICAL 172、DRIFTED 0)が通った。`cmp` で 6 つの写しが一致(verify)。
- 固定するテスト: `TestDirectorSkill_PinsTheCIRegate`(`director_skill_sync_test.go:1139`)、`TestOrgSkill_PinsTheCIRegate`(`org_skill_send_sync_test.go:358`。regate の行の 3 断片、spawn・start の節を sync-docs で足した)、`TestOrgStartSpawnAndMergeOrder_HelpFollowTheCleanupAfterAMerge`(`org_cleanup_test.go`、start と spawn の help の新しい 1 文。+4 行)、`TestOrgRegate_Help`(V1 の文)。
- README(`README.md:280` の merge-order の段落): 事前の確かめの位置を「right after its fetches, before the merge」にし、待つ `ci` の regate は lock を外すと書いた。AGENTS.md の `internal/org/` の行(`:91`)に `regate_ci.go`、`gate_ci_passed`/`gate_ci_failed`、`via=regate_ci`、merge と共用する `prchecks.go`。仕様 `docs/specs/2026-10-07-org-multi-org-director.md:132` に FR-7 の注「2026-10-30 に決めたこと」。本物の CI で打っていないので FR-7 の印は `[ ]` のまま。`docs/quality/quality-gates.md` とその template の写しの Quality pipeline gate の行(regate は `ci` ではスクリプトを走らせない、V2)。`.claude/rules/ralph/model-routing.md:151` の key 一覧に `[org.gate]`(template には該当の文がない)。
- tech-debt: 3 行を足し(`docs/tech-debt/README.md:395-397`)、290 行に持ち越しの追記をした。内容は 12 節の Known gaps に書いた。

## 10. テストの見方

- 設定: `internal/config/config_test.go` の `TestDefault_OrgGate`、`TestLoad_OrgGate`(既定、`local`、`ci`、ほかの値、空文字)。`internal/cli/config_test.go` に `ralph config get org.gate.regate_checks` の 2 行(`local` と、ファイルで `ci`)。
- org 層は、本物の git と bare origin の fixture(`rcFixture`、`newRCWaitFixture`)に、`git remote get-url origin` と gh を偽にした `Exec`(`mxExec`)を足して打つ。`mxExec` は呼び出しごとに `-R`・env・deadline・dir を見る。push の失敗は、origin が拒む形と `pre-push` hook が git を kill する形の両方を本物の git で起こす。待ちは偽の時計と偽の sleep。
- `^TestRegateCI_` はトップレベル 28、サブテスト 102(test の report、変異なしで全部通った)。
- test の report の変異は 8 つで、8 つとも落ちた(`go test -overlay`、追跡しているファイルは変えていない):

| 変異 | 落ちたテスト |
|---|---|
| m1a: pending の check を pass と同じに扱う(`readRegateChecks` の case) | `GateOnlyRecordsNothingWhileTheChecksAreNotDone/a_check_is_pending` |
| m1b: `regateChecks` の `read.pending != ""` の case を外す | `GateOnlyRecordsNothing…` のサブテスト 6 つ、`MergePathPushes…` |
| m2a: 最初の PR の head の確かめを外す | `GateOnlyRecordsNothing…/the_PR's_head_is_not_HEAD_yet`、`MergePathPushes…`(結果は pending のままで、落ちたのは理由の文と gh の呼び出しの一覧) |
| m2b: checks を読んだあとの head の読み直しを外す | `GateOnlyRecordsNothing…/the_PR's_head_moved_while_its_checks_were_read`(pass を記録した) |
| m2ab: PR の head を一度も確かめない | `GateOnlyRecordsNothing…` のサブテスト 2 つ、`MergePathPushes…`(古い head の checks で pass を記録した) |
| m3a: gate だけの道で `ci` でもスクリプトを走らせる | `GateOnly…` のサブテスト 20 個 |
| m3b: merge の道で `ci` でも merge のあとにスクリプトを走らせる | `MergePathPushes…`、`MergePathRecords…` のサブテスト 2 つ |
| m4: 読み返した先頭が merge の前のままのときの `regateUndoMerge` を消す | `AFailedPushIsDecidedByOriginsTipReadBack` の 2 つの場面、`AFailedPushWhoseUndoFailsNamesWhatEachGitSaid` |

  m2a だけでは記録は起きず、PR の head の確かめは checks の前と後の 2 段で守られている。各スライスの implementer も変異を打っていて、Progress が数を書く(S1 は 5 つの外した版、S3 は 6、S4a は 6、S4b は 9、S5 は 14、S6 は 15、F1 は 7)。これらは Progress の記述を読んだもので、この walkthrough では再現していない。
- coverage(test の report、ci の regate と `prchecks` のテストだけを打った値): `regate_ci.go` が 91.6%(286 文のうち 262)、`prchecks.go` が 100%(50 文のうち 50)。80% 未満の関数は `regateCIPushSaid` の 75.0% と `writeGateChecksLog` の 77.8%。branch coverage は測っていない。
- テストファイルの行数: `regate_ci_test.go` は F3(aff0dcb0)で約 2,300 行から 4 files(767・571・726・300 行)に分けた。宣言 61 件がバイト単位で同じ(self-review run 2 が `go/parser` で比べた)。いまは 767・630・726・377 行で、どれも 800 行未満。

## 11. 直しの経過

- 計画: c6b9fe35。承認は自動(2026-10-11 のユーザー指示)。plan advisory は codex が HIGH 1 件(revert 後に古い binary が `regate_ci` の pass を押し直す)と MEDIUM 1 件(push の応答が失われ origin が進んでいるときに公開済みの merge を外す)を出し、D3・D4・D7 に反映した。
- 計画どおりのスライス: S1 3a55f6d1(設定)、S0 137ec668(#288 の merge のあと origin/main を取り込み。衝突は `internal/config/config.go` の 1 か所)、S2 9becd883(記録と読み手)、S3 ab1b0d43(checks の読み取りの共用)、S4a 2ab52083(事前の確かめ、gate だけの道)、S4b 688d4d6a(merge の道、push を記録の前に)、S5 33be080c(待ち)、S6 a665eba8(CLI)、S7 a3d8a1d8(文書)。計画への記録 commit は 78ba7910、573d7495、8c39305f、202c2f85、bc191c54、e1a18588、5112a656。
- 13d19f3d: 完了した S4b の implementer が、作業中に送った返事で再開して足した直し(知らない outcome の CLI の行のテストと、`Regate` の要約の注釈)。amend はさせず、202c2f85 の上の新しい commit にした。
- self-review run 1(report 03f57f7f、HEAD 5112a656): MEDIUM 1(M1)、LOW 8(L1〜L8)。扱いは次のとおり。
  - F1 e40b3466(L3・L4): 待つあいだは lock なしで読むだけにし、終わりに 1 回だけ取り直す。checks を読んだあとに head を読み直す。
  - F1b f32fe553 と 02219639: 待つあいだに PR の head が別のコミットなら、lock なしで `git ls-remote` を打つ。
  - F2 69f80d8f(L1・L2・L6・L7・L8): 子孫の形の断り、undo の文を git の出力の前へ、help と skill の例外、template のコメント、新しい文のコマンドの quote。
  - F3 aff0dcb0(M1 の一部): テストファイルの分割。
  - 5c850e50: 台帳に 3 行(`regate.go`・`org_regate_test.go` の行数、L5、末尾の打ち直しの案内)と 290 行に追記。
  - f103777d: 計画に記録。
- self-review run 2(report c94c22dc、HEAD f103777d): LOW 5(R2-1〜R2-5)。F3 までの前例にならい、run 3 は回さずに verify へ進んだ。
  - F4 8e86b75d(R2-3・R2-4・R2-5): `--wait-checks` の help に止まり方をすべて書く、gh の失敗で止まった待ちを「again」と言わない、push の失敗の出力を「git push said:」と名指す。
  - 513b22a0(inline): `RegateOutcomeChecksPending` と `ChecksPending` の doc に、gh の失敗で止まった待ちを足した。
  - 23dfc9c7(inline、台帳): 395 行に `regate_ci.go` を足し、払い方に待ちを `regate_ci_wait.go` へ出す手を先に置いた(R2-1)。290 行の「+85」を「+114」に(R2-1)。397 行に前からある local の文も quote しないことを足した(R2-2)。
- verify 1994eba2(HEAD 23dfc9c7、pass)、test bcd143e2(HEAD 1994eba2、pass)、sync-docs a0a8da30(V1〜V5 を直した)、cross-review triage 4e0bc99a、AC の checkbox と Progress 077e3b5b。

## 12. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(verify、HEAD 23dfc9c7。`ralph.toml` が分類されず full に落ち、pack は golang。gofmt ok、`go vet` 出力なし、golangci-lint `0 issues.`、staticcheck 出力なし、shellcheck と hook の `sh -n`、`check-sync`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-skill-sync`(14 skill)、`check-template-purity`、tech-debt の計画参照、secret scan は c03647f9..23dfc9c7 で clean)。`git diff --check` rc 0、U+FFFD 0、写しの `cmp` 6 つ一致、計画の digest が承認の行と同じ。
  - 手元で build した binary の probe(verify): `ralph config get org.gate.regate_checks` が、ルートで `ci`、`ralph.toml` のない dir と template の写しで `local`(rc 0)、`"remote"` で rc 3 と検証の文。`--wait-checks` の断り(`local` で `1m`・`-1s`・`7h`、`ci` で `7h`・`-1s`)は rc 1 で、台帳を読む前に断り、`RALPH_ORG_STATE_DIR` の dir は作られなかった。
  - `GOFLAGS="-timeout=30m ..." ./scripts/run-test.sh` rc 0(test、HEAD 1994eba2。shell 45 本、Go 10 パッケージ、2,256 秒)。`^TestRegateCI_` 28 本とサブテスト 102 が通った。変異 8 つがすべて落ちた。
  - cross-review: `codex exec review --base main`(`-c sandbox_mode=read-only` と `--ignore-rules`、約 5 分、exit 0)が所見 0。codex は read-only の sandbox で一時 build ディレクトリを作れず、テストを走らせていない。
  - sync-docs(a0a8da30)のあと、絞ったテストが通った: `internal/cli` の `OrgRegate|Help|OrgStartSpawnAndMergeOrder`(99.1 秒)、`internal/org` の `Skill|Prompt`(28.3 秒)、`check-skill-sync`、`check-sync`、`git diff --check`。
  - 受け入れ条件: AC2〜AC8 と AC10 は verify が静的に確かめた(AC8 は probe も)。AC1 と AC9 は Partly だった。AC9 の文書のずれ V1〜V5 は a0a8da30 で直した。AC1 の変化は 1 節のとおりで、ずれとして Progress に残した。
- 確かめていない・残っていること:
  - 本物の GitHub の PR と CI での `ci` の regate。push の直後に PR の head と checks が更新されるまでの時間(計画の Open questions)は未確認。FR-7 の印は `[ ]` のまま。
  - 古い binary(origin/main)が `gate_ci_*` を含む台帳を読む実測。古い読み手が event 名で選ぶことは、ソースで確かめただけ(test の Test gap 1。`git archive` で取り出した binary に `gate_ci_passed` だけの台帳を読ませる観察が最小)。
  - 待つあいだに既定のブランチが進み、記録の base が古くなって merge-order が `rebase` を返すまでの通しのテスト。分類の行は `mergeorder_test.go` が見るが、regate の待ちからそこまでは通していない(Test gap 2)。
  - `regate_ci.go` の通らなかった失敗の道: push の前に追跡 ref を読めないときの undo、push が `regateGitTimeout` を越えるとき、読み返しの最中の割り込み、log や記録の I/O の失敗、取り直しの確かめで git が失敗するとき(Test gap 4)。
  - sync-docs a0a8da30 の変更(help の文字列 2 か所、doc comment 1 か所、skill、README、quality-gates、固定するテストの断片)のあとに、`run-static-verify.sh` と `run-test.sh` の全体は打ち直していない(上の絞ったテストだけ)。固定するテストの断片が、直す前の文で落ちることは変異で確かめていない。
  - F4(8e86b75d)・513b22a0・23dfc9c7 は、self-review の run 3 を回していない。verify が HEAD 23dfc9c7 で静的に見ている。
  - cross-review の codex は 1 回だけ、テストなしで読んだ。
  - 02219639 の commit message が handoff の指定と違う件は、指定の中身を読んでいないので比べていない。
  - Known gaps(PR の本文に書く。tech-debt の行):
    - 290 行: `regate_test.go`(+51)と `regate_merge_test.go`(+114)の分割の Trigger をこの PR も引いたが払わなかった。新しいテストは新しい files に置いた。持ち越し。
    - 395 行: 800 行を越えたファイルが 3 つ。`regate.go`(763 → 995 行。feat/org-regate が分けて 2026-10-16 に RESOLVED にしていたもの)、`regate_ci.go`(新規 901 行)、`internal/cli/org_regate_test.go`(760 → 1273 行)。払い方は、待ちを `regate_ci_wait.go` に出し、`Regate` の `ci` の部分を `regate_ci.go` の helper の後ろへ移し、CLI テストの `ci` の部分を `org_regate_ci_test.go` へ。
    - 396 行: まだ作られていない check を数えない(self-review L5)。この repo の PR には workflow が 2 つあり(`verify.yml` は 10 分を越え、`check-template.yml` は短い)、verify の run の作成が遅れると check-template の pass だけで `gate_ci_passed` が書かれ、merge-order が `ready` を出しうる。merge は直前に checks を読み直し、保護ブランチの必須 checks が止める。
    - 397 行: regate の断りの末尾「run ralph org regate --branch <ブランチ> again」(`gateWording.again`)はブランチを quote しない。#264 が足した `ci` の文は自分のコマンドを quote するが、この共通の末尾と `--wait-checks` の断りは前のまま。前からある local の文(`regateCheckOriginHead`、`regateResetHint`、`regateMergeTip`、`regateMergeLeft`)も worktree とブランチを quote しない(self-review R2-2)。
