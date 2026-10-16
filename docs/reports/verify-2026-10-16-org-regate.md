# Verify report: org-regate

- Date: 2026-10-16
- Plan: docs/plans/active/2026-10-16-org-regate.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は f841204a7bc0 で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 のうち、merge のあとの取り直し。8 段目の 8a-3b)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 1)
- Scope: `git diff 670ec219...HEAD`(HEAD 0b6ba232、24 コミット、36 ファイル、+6790/-324)。仕様適合(AC1〜AC13、D1〜D9、FR-7 と FR-9)、実装中に記録したずれ 6 件、self-review の N1〜N4 の直し(0b6ba232)、文書のずれ、静的解析、HEAD の binary とローカルの bare origin での観察。テストは走らせていない(/test の受け持ち)。テストはコードを読み、どの AC をどう固定しているかを確かめた。行番号は HEAD 0b6ba232 のもの
- Evidence: `docs/evidence/verify-2026-10-16-org-regate.log`(plan の digest、run-static-verify の出力、Go の検査、probe 1 と 2 のスクリプトと出力。`$PROBE1`・`$PROBE2` は scratchpad の probe のディレクトリ、`~` は `$HOME`)。wrapper の全ログは `docs/evidence/verify-2026-10-16-154249.log`。どちらも gitignore の対象

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 断る 11 の場合(何も記録せず、worktree も origin も変えず、終了コード 1、理由と次の手を stderr) | 満たす(binary でも確認) | `--branch` なしと `--org-id` は CLI が台帳を読む前に断る(`internal/cli/org_regate.go:116`、`:119`)。結びつきなし・org が走っている(`internal/org/regate.go:440` の `regateFirstCheck`)、worktree なし・ブランチ違い・detached・rebase か merge の途中・汚れた作業ツリー・スクリプトなし(`regateCheckWorktree`、`:529`、`regateCheckInProgress`、`:607`)、ほかの regate(`:280` の `acquireHeldFileLock` と `errFileLockHeld`)。probe 1 の A1〜A3・E1、probe 2 の G1〜G3・G8 は、どれも終了コード 1 で stdout は空、HEAD・origin・manifest の行数は変わらない。テスト(読んだだけ): `TestRegate_RefusedBeforeTheFetch`、`TestRegate_RefusedWhileAnotherRegateHoldsTheLock`、`TestOrgRegate_RefusalPrintsNothingAndChangesNothing`、`TestOrgRegate_OrgIDRefusedBranchRequiredNoArguments` |
| AC2 fetch の失敗、origin にブランチがない、HEAD が `origin/<b>` と違う。fetch と push に `GIT_TERMINAL_PROMPT=0` | 満たす(env はテストのコードで確認) | `regateFetch`(`internal/org/regate_git.go:30`)は 2 回に分けた明示の refspec で fetch し、`couldn't find remote ref` を分けて言う(`:44`)。HEAD の確かめは `regateCheckOriginHead`(`:82`)。env は `regateFetchEnv`(`:136`、`GIT_TERMINAL_PROMPT=0` と `LC_ALL=C`)と `regateRemoteEnv`(`:128`、push)。probe 2 の G5(origin から `feat/auth` を消す)は「origin has no branch feat/auth: it was deleted after the merge」、G6(origin の URL を無いパスに)は `failed (exit 128)` と git の stderr の 5 行、G4(push していない commit)は「has commits that origin's tip ... does not: ... push them」で、どれも終了コード 1。env は `TestRegate_FetchRunsWithNoTerminalPrompt` と `TestRegateRemoteEnv` が固定する(/test で確かめる) |
| AC3 何もしない道と、壊れた記録のあとの取り直し | 満たす(binary でも確認) | `regateBranch.passedOn`(`internal/org/regate.go:421`)は、読める、あとに壊れた行がない、`gate_passed`、head と base と base_ref が今のもの、のすべてを求める。`:331` で `RegateOutcomeUpToDate`。probe 1 の B3 と D5 は `UP_TO_DATE:`・`HEAD:`・`BASE:`・前の `LOG:` を出して終了コード 0、manifest の行数とスクリプトの呼び出しは増えない。壊れた Details と壊れた JSON 行のあとに取り直して merge-order が `ready` を出すことは `TestRegate_GatesWhenNoRecordLetsItStop` の 2 つの case が固定する(/test で確かめる) |
| AC4 取り込みの道(D3 の merge commit、記録、force なしの push、stdout の 6 行、merge-order が `ready`) | 満たす(binary でも確認) | `regateMergeTip`(`internal/org/regate_merge.go:242`)、`regateGate` の記録(`internal/org/regate.go:655` の `Via: gateViaRegate`)、`regatePush`(`regate_merge.go:473`。refspec は `<sha>:refs/heads/<b>` で `+` を付けない)。probe 1 の B1: `GATE: pass`・`MERGED:`・`HEAD:`・`BASE:`・`LOG:`・`PUSHED:` の順で終了コード 0。commit は `Merge origin/main 71ff0650ba62 into feat/auth`、親は merge 前の HEAD と取り込んだ先頭、author は呼び出し元の git の名前。manifest の最後の行は `org_id: auth` の `gate_passed` で、Details は `... herdr_pane=unset via=regate log="..."`。B2 の merge-order は `ready` |
| AC5 merge commit を作れない(衝突、pre-merge-commit の hook、名前がない) | 満たす(binary でも確認) | 名前は merge の前に `git var` で確かめる(`regateCheckIdentity`、`regate_merge.go:202`)。衝突と hook は `regateAbortLeftMerge`(`:327`)が `git merge --abort` のあとに HEAD と作業ツリーを確かめる。probe 2 の G10(README.md の衝突)は `conflicts in README.md`・`git merge --abort undid the merge it left in progress` を出し、HEAD は前のまま、`MERGE_HEAD` なし。G7(`user.useConfigOnly`)は `git var GIT_AUTHOR_IDENT failed` と設定の仕方を出し、merge は走らない。hook が止める場合と、signal で止まった merge は `TestRegate_RefusedWhenTheMergeCommitCannotBeMade`(5 case)と `TestRegate_AMergeEndedByASignalSaysHowToPutTheWorktreeBack` が固定する(/test で確かめる) |
| AC6 スクリプトの失敗、記録の前の中断、`head_moved` と `tree_changed` | 満たす(一部はテストのコードで確認) | `regateMerged`(`regate_merge.go:133`)は `static_failed`・`test_failed` で `regateUndoMerge`(`:405`、`git reset --keep <merge の前の HEAD>`)、`head_moved`・`tree_changed` では戻さず `regateMergeLeft`(`:448`)で状態と戻すコマンドを出す。probe 1 の C1(`run-test.sh` が exit 1): `GATE: fail`・`REASON: test_failed`・`MERGED:`・`LOG:`、スクリプトの最後の行、「the merge commit ... was undone (git reset --keep ...)」、終了コード 1。HEAD は merge の前に戻り、記録は `via=regate` の `gate_failed`。中断と `head_moved`・`tree_changed` は `TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge`、`TestRegate_AMergeCommitIsKeptWhenUndoingItCouldLoseWork`、`TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing` が固定する(/test で確かめる) |
| AC7 push の失敗と記録のあとの中断、打ち直し、進んだ既定のブランチ、D4 の手 | 満たす(binary でも確認) | `regatePush` は失敗を `PushFailed` にし、記録と HEAD を残す(`regate_merge.go:473`)。打ち直しは `regateHeadNotOnOrigin`(`:62`)。probe 1 の D1(origin の pre-receive が断る): push の前までの 5 行と `failed (exit 1)` で終了コード 1、HEAD は merge commit のまま。D3 の打ち直しは `PUSH_ONLY:` と `PUSHED:` で終了コード 0、D4 の merge-order は `ready`。probe 2 の G9b(push が落ちたあとに main が進む)は push だけをせずに断り、`reset --keep refs/remotes/origin/feat/auth` の手を出した。その手どおりに戻した G9c は新しい先頭を merge して push した。push を受け取ったあとの中断と、fetch と push の間の他人の push は `TestRegate_APushThatLandedWhoseAnswerWasLost`、`TestRegate_AnInterruptAfterTheRecordKeepsItAndTheMerge`、`TestRegate_AFailedPushIsNotRetriedPastWhatChanged` が固定する(/test で確かめる) |
| AC8 スクリプトの間に org か結びつきが変わったら何も記録せず merge を戻す | 満たす(テストのコードで確認) | `recordRegate`(`internal/org/regate.go:681`)は manifest のロックの下で読み直し、結びつきなし、org_id か中身の違い、位置の違い、走っている org のどれでも記録せずに断る。`regateMerged` の `res.Refused` の枝(`regate_merge.go:163`)が merge を戻す。`TestRegate_NothingRecordedWhenTheBranchChangedWhileTheScriptsRan`、`TestRegate_ARefusedRecordAfterTheMergeUndoesIt` が固定する(/test で確かめる) |
| AC9 gate だけの道 | 満たす(テストのコードで確認) | `internal/org/regate.go:335` の `regateGate` で、merge も push もしない。`TestRegate_GatesWhenNoRecordLetsItStop`(7 case、worktree と origin の ref が変わらないことも見る)、`TestRegate_GateOnlyRecordsAGatePassedViaRegate`、`TestOrgRegate_GateOnlyPassPrintsNoMergeAndNoPush`。probe ではこの道を単独では打っていない |
| AC10 merge-order の理由の欄 | 満たす(binary でも確認) | `namesRegate`(`internal/org/mergeorder.go:348`)、`mergeRegateNext`(`:962`)、`mergeRegateFailedDetail`(`:980`)、`regate` の文(`:1027`)、`rebase` の文(`:1046`)。probe 1 の C2(regate の `gate_failed`)は merge commit とそれを戻したことと log のパスを、D2(push が落ちたあと)は「pushed after the gate, or the gate's head was not pushed」と `ralph org regate --branch feat/auth` を出す。probe 2 の G11a(org_id auth を `feat/other` で立て直した)は「org_id auth now runs another feature, on the branch feat/other: once auth is disbanded, run ralph org regate --branch feat/auth」を出し、走っている org が今結びついている `feat/other` の行は今までの `ralph org gate --org-id auth` の文のまま |
| AC13 Spawn との排他(D9) | 満たす(テストのコードで確認) | spawn は `checkCapacityAndStart` の先頭(`internal/org/spawn.go:1284`、冪等の判定と stale の補償のあと)、start は ensure の前(`internal/org/feature.go:216`)で `regateSpawnErr`(`spawn.go:1327`)を呼ぶ。ロックの待ちは `regateLockWait = time.Second`(`regate.go:93`)。probe 2 の G8(perl の flock で `regate.lock` を 3 秒持つ)は約 1 秒で断られた。spawn と start の断りは `TestOrgSpawn_RefusedWhileRegateRunsOnTheOrg`(最初の確かめのあとと記録の直後の 2 つの hook)、`TestOrgSpawn_RegateLockStates`(4 状態)、`TestStartFeature_RefusedWhileRegateRunsOnTheOrg` が固定する(/test で確かめる)。spawn と start を binary では打っていない |
| AC11 文書 | 満たす | 下の Documentation drift の表。AC11 が挙げる面(skill の 8 面、仕様の注、tech-debt、`AGENTS.md`、CLI の help)はすべて更新済み。挙げられていない `docs/quality/quality-gates.md` にずれが 1 件ある(V1) |
| AC12 テストはネットワーク・herdr・osascript・端末を打たない。go1.26 と go1.25.8 | 一部(静的な部分を確認) | 足したテストの行に URL、`osascript`、`tmux`、本物の `herdr` の呼び出しはない。origin は `t.TempDir()` の下の bare repository(`newRegateFixture`、`internal/org/regate_test.go:52`)。CLI の merge-order は `setMergeOrderPATH` で gh のない PATH にして打つ(`internal/cli/org_regate_test.go:174`)。`go build` と `go vet` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で通った。テストを両方の版で走らせるのは /test |

D1〜D9:

- D1: `--org-id` を断り(`org_regate.go:116`)、`orgLedgerMutating` で台帳を決める(`:123`)。`Regate` は見張りの関門も承認の確かめも呼ばない(`regate*.go` に `WatchGate` と承認の呼び出しはない)
- D2: `readMergeLedger` と `collectMergeTargets` をロックの下で読む(`regate.go:443` の `withManifestLockContext`)。結びつきは同じブランチの最後の完全なもの(`collectMergeTargets` は後の `scope_reserved` で上書きする)。走っているかは `RunningOrgs`(`mergeorder.go:439`)
- D3: merge の文は `Merge origin/<既定> <12 桁> into <b>`(`regate_merge.go:244`)。先頭を含めば merge しない(`regate.go:329`)。rebase と force push の道はない
- D4: 上の AC2 と AC7。push だけの例外は `passedOn` と `origin/<b>` が HEAD の祖先であること(`regate_merge.go:64-72`)
- D5: 戻すのは HEAD がまだ merge commit で作業ツリーがきれいなときだけ(`regateUndoMerge`、`regate_merge.go:405-424`)。記録のあとの中断は push の失敗と同じ扱い(`regatePushInterrupted`、`:508`)
- D6: 記録の Details は `via` を `log` の前に置く(`internal/org/gate.go:173`)。probe の manifest の行で確かめた
- D7: `regate.lock` は `Regate` の最初から `defer` の解放まで持つ(`regate.go:280-288`)。push と巻き戻しはその中にある
- D8: 2 回の fetch、明示の refspec、`GIT_TERMINAL_PROMPT=0`、1 回 120 秒(`regateGitTimeout`、`regate.go:110`)。fetch の失敗は何も記録しない(probe G5、G6)
- D9: 排他を取った直後に印を空にし(`regate.go:291`)、manifest のロックの下で org_id・branch・worktree・pid の印を書き(`:466`)、終わりに消さない。spawn は待たない共有の flock(`fileLockHeld`、`internal/org/lockfile.go:182`)で確かめ、印の org_id が自分のものと同じときだけ断る

FR-7 と FR-9:

- FR-7「後から入る PR は、main に rebase してゲートを取り直してから出す」: push する head(merge commit)で gate を取り、記録し、force なしで push する。仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:75`)が 7b の注の読み方(「push する head で最後の gate を取る」)に従うと書く。FR-7 の checkbox は付けない理由も同じ注にある
- FR-9「director は ralph と gh の操作に限る」: regate は ralph の動詞で、`/director` skill は人に頼まれたときだけ 1 本ずつ打ち、衝突と失敗を git で直さない(`.claude/skills/director/SKILL.md:457`、`:515`)
- 8a-3a に残すもの: diff の Go と設定に `auto_merge` はない(grep)

### 実装中に記録したずれ(承認済みの計画の文との突き合わせ)

| ずれ | 計画の文と合うか | Evidence |
| --- | --- | --- |
| D4 の `via=regate` を `reset --keep` の手の条件に足した | 合う | 承認の digest f841204a7bc0 の D4 は「最新の記録が regate の書いた(`via=regate`)読める記録で、その head が HEAD で、HEAD の第 1 親が `origin/<b>` の先頭」と書く。`regateResetHint`(`regate_merge.go:92-104`)は同じ 3 条件に、D2 が例外から外す「あとに壊れた行がある」を足して確かめる。push だけの例外は `via` を見ない(leader の通した head を押す)ことも Progress の S3 の注のとおり。probe 2 の G9b で手が出た |
| D9 の印(取った直後に空にする、終わりに消さない) | 合う | D9 の文(0e719e0a で直して承認し直したもの)と `regate.go:291`・`:466`、解放の `defer`(`:288`)が一致する。印が空か読めない、別の org の印、ロックが空いている場合に spawn が続くことは `TestOrgSpawn_RegateLockStates` が固定する |
| AC10 の文「走っている org が今結びついているブランチは今のまま」 | 合う | `namesRegate` は「走っていない」か「org が別のブランチで走っている」ときだけ真(`mergeorder.go:339-349`)。probe 2 の G11a で、`feat/other` は gate の文のまま、`feat/auth` は regate の文 |
| merge・abort・戻し・`git var` を別のプロセスグループで動かし、fetch と push は ralph のグループに残す | 合う | 計画はプロセスグループを決めていない。Risks の 121 行は「ssh の鍵の入力は止めない(`core.sshCommand` を上書きしない)」で、fetch と push は `regateRemoteGit`(`regate_git.go:145`、`ownGroup=false`)で ralph のグループに残る。merge・abort・戻しは `regateUncancelledGit`(`regate_merge.go:28`)、`git var` は `regateBoundedGit(..., true, ...)`(`:206`、0b6ba232 で理由のコメントを足した)。D5 の「中断されたら merge commit を戻す」は、merge を終えてから戻すので成り立つ |
| 中断のあと merge を待つ間の stderr の 1 行 | 合う | 計画にない足し。stdout の行と終了コードは変わらない。help(`org_regate.go:104-108`)と `/org` skill の `regate` の行がこの 1 行を書く |
| regate の rebase 途中の断りは `git rebase --abort` だけを案内する | 合う | AC1 は「rebase の途中」を断り、理由と次の手を出すことだけを求める。D3 は rebase と force push をしない。`regateWords` の `rebaseSteps`(`regate.go:350-355`)で regate だけが変わり、gate の文は `gateRebaseSteps`(`gate.go:415`)のまま。probe 2 の G2 の文は「undo it with git rebase --abort (regate merges origin's default branch instead, and a finished rebase would need a force push)」 |

### self-review の直し(0b6ba232)

self-review の報告(`docs/reports/self-review-2026-10-16-org-regate.md`)は HEAD 07b9a465 の時点のもので、N1〜N4 を開いたまま挙げる。0b6ba232 がそのあとに 4 件を直した。

| 指摘 | Status | Evidence |
| --- | --- | --- |
| N1 merge のあとに HEAD か親を読めないときの error | 直った | `regateMergeUnchecked`(`regate_merge.go:311`)が「merge は残っている、何も走らせず記録していない、`git log` を見る、`reset --keep <merge の前の HEAD>` で戻す」を言う。HEAD の読み、親の読み、親の違いの 3 つの return がこれを通る(`:254-266`)。推奨の `regateMergeNotBack(moved=true)` ではなく別の helper だが、案内の中身(`reset --keep` で merge の前の HEAD へ)は同じ。doc の列挙(`:234-241`)も直った。新しいテスト `TestRegate_AMergeCommitItCannotReadNamesResetKeep` |
| N2 `regateMergeTip` の長さと `r` の隠れ | 直った | 失敗した merge の扱いを `regateAbortLeftMerge`(`:327`)に切り出し、`regateMergeTip` は 62 行。closure の引数は `c rune`(`:286`) |
| N3 `git var` のグループの理由 | 直った | `:204-205` に 1 行の理由 |
| N4 tech-debt の行数と行番号 | 直った | 0b6ba232 が「713 lines」「1495 lines」「`:299-303`」を消した。`docs/tech-debt/README.md:278` の「851 lines」は解消済みの行の取り消し線の中で、当時の記録 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | scope は changed、言語は golang。hook の shellcheck と `sh -n`、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、gofmt ok、golangci-lint `0 issues`、`secret-scan-branch: scanned 670ec219..0b6ba232 against origin/main: clean` |
| `go vet ./...`(go1.26.0) | pass | 出力なし、rc 0 |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod の Go |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし、rc 0 |
| `git diff --check 670ec219...HEAD` | pass | 出力なし |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass | IDENTICAL 170、DRIFTED 0、KNOWN_DIFF 5 |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | f841204a7bc0 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 面 | yes(I1 を除く) | 4 面の md5 は `52efc20f57b38022ee4076d85733cf2e` で同じ。`regate` の行(`.claude/skills/org/SKILL.md:188`)の確かめの順、fetch、`reset --keep` の手、merge の文、出力の行、戻す条件、push の打ち直し、stderr の 1 行、終了コード、spawn と start の断りは、コードと probe の出力に合う。merge-order の行の理由の欄の説明、gate の記録の節の `via=regate`(`:927`)、「merge してよい順」の手順(`:1010` から)、見張りなしで使える動詞の一覧(`:1255`)、台帳を書き換える動詞の一覧に `regate` が入った |
| `/director` skill の 4 面 | yes | md5 は `2aa01e8014c06fab49ff8b58b5688d34` で同じ。「merge のあとの取り直し」(`:457`)は、人に頼まれたときだけ、1 本ずつ、バックグラウンドで、失敗と衝突は人に伝える、走っている org には打たない、の手順。してはいけないこと(`:515`)と「まだないもの」(`:521`)も合う |
| CLI の help(`ralph org regate`、`ralph org`、`--org-id`、`ralph org merge-order`) | yes | probe 1 の F1・F2 の出力。`--org-id` の説明に regate が入り、merge-order の `regate` の分類と理由の欄の説明(`internal/cli/org_merge_order.go:73-84`)は probe の出力と合う |
| `AGENTS.md` の repo-map(90 行) | yes | `regate.go`・`regate_merge.go`・`regate_git.go` があり、書いた振る舞いと合う |
| `README.md`(124、264 行) | yes | 動詞の表と merge-order の段落。merge-order の「fetch・gate・merge をしない」は merge-order について今も正しい |
| 仕様の FR-7 の注(75 行) | yes | 名指すテスト 2 つ(`TestRegate_MergesTheTipInAndPushes`、`TestOrgRegate_MergesPushesAndThenIsUpToDate`)は実在する。FR-7 の checkbox を付けない理由がある |
| `docs/tech-debt/README.md` | yes | 取り直しの行は RESOLVED の注と取り消し線になり、238 行に 8a-3b の注、275・276・279・280 行が足された。どれも 5 列。Trigger と Related が名指すテストと関数(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestRegate_FetchFailures` ほか 4 つ、`idempotentRespawn`、`checkCapacityAndStart`、`regateSpawnErr`、`recordRegate`、`mergeRegateFailedDetail`)はすべて実在する。新しい行は計画を `docs/plans/active/` で指し、/pr の `archive-plan.sh` が書き換える |
| `internal/cli/org_legacy_ledger.go:21-26` | yes | 台帳を書き換える動詞の doc comment に `regate` |
| V1 `docs/quality/quality-gates.md:80` と `templates/base/docs/quality/quality-gates.md:79` | no(LOW) | Cross-org watch gate の行の「`stop`, `disband`, `escalate`, `approve`, `gate`, `merge-order`, ... work with no watch」に `regate` がない。regate は見張りを見ない(D1)ので、一覧が足りない。挙動は変わらない。lead が把握済みで、/sync-docs で 2 面とも直す |
| 計画の AC と Progress の checkbox | 遅れている(情報) | AC1〜AC13 はすべて `[ ]`、Progress の「Verification artifact created」も `[ ]`。Progress に 0b6ba232(N1〜N4 の直し)の注がない |

情報(ずれに数えない):

- I1 `/org` skill の `regate` の行は、`reset --keep` の手の条件を「最新の gate の記録が読める、その HEAD の `via=regate` の記録で、HEAD の第 1 親が origin の先頭」と書く。コードは記録のあとに壊れた行がないことも求める(`regate_merge.go:94` の `b.corruptAfterGate`)。同じ行の `UP_TO_DATE:` の文はこの条件を書いている。コードのほうが手を出す場合が狭く、手が出ないだけなので、誤った操作には導かない
- I2 `regateSpawnErr` は `regate.lock` を確かめられないとき(読めない、flock の失敗)も spawn を断る(`spawn.go:1327-1333`)。D9 はこの場合を決めていない。断る側に倒しており、doc comment が書く
- I3 `regateGitTimeout`(120 秒)は fetch と push のほかに merge・abort・戻し・`git var` にも掛かる。D8 が名指すのは fetch と push だけで、Progress の S1 の注が「remote を叩く git の 1 回ごとの 120 秒の上限」として記録し、help と skill は merge の待ちを「最大 120 秒」と書く
- I4 Test plan の「CLI の…台帳の決め方」のうち、regate が `orgLedgerMutating`(古い台帳の座席が動いている間は断る)で台帳を決めることは、コード(`org_regate.go:123`)で確かめたが、固定するテストはない。`TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` の表は `gate` と `approve` も持たないので、今までの慣行と同じ
- I5 tech-debt の 275 行(古い台帳で Reserve 付きの leader の spawn が冪等の枝で排他を通らない)は、読んで行の中身と名指す関数を確かめただけで、場面は再現していない

## Observational checks

HEAD の binary(`go build -o $PROBE1/bin/ralph ./cmd/ralph`)を、`$PROBE/bin`(ralph と Homebrew の git の symlink)、`/usr/bin`、`/bin` だけの PATH で打った。gh は PATH にない。HOME と `GIT_CONFIG_GLOBAL` は probe の中の偽の場所で、origin は probe の中の bare repository。台帳は main checkout の `.harness/state/org` で、`--state-dir` で渡した。manifest は auth の完全な結びつき(`feat/auth`、worktree は probe の linked worktree)と `disbanded` の 2 行から始めた。gate のスクリプトは `echo` と、環境変数で決める exit だけの stub。ネットワーク、herdr、本物の GitHub は打っていない。

- probe 1(A〜F): 断り 3 つ、main が進んだあとの取り込み・記録・push と merge-order の `ready`、UP_TO_DATE、テストの失敗と merge の戻し、merge-order の regate の `gate_failed` の文、origin の pre-receive が断る push、merge-order の新しい `regate` の文、PUSH_ONLY、再びの UP_TO_DATE、走っている org の断り、help。上の AC1・AC3・AC4・AC6・AC7・AC10
- probe 2(G1〜G11): 汚れた作業ツリー、止まった rebase、merge の途中、push していない commit、origin にないブランチ、fetch の失敗、git の名前がない、ほかの regate のロック(約 1 秒で断る)、push の失敗のあとに main が進んだときの `reset --keep` の手とその手のあとの取り直し、衝突、org_id を別の機能で立て直した org の merge-order の文と regate の断り。上の AC1・AC2・AC5・AC7・AC10・AC13

## Coverage gaps

- テストは走らせていない。go1.26 と `GOTOOLCHAIN=go1.25.8` の `go test ./internal/org/ ./internal/cli/` は /test で確かめる
- 中断の道(スクリプトの間、merge の間、記録の直後、push の間の SIGINT・SIGTERM・SIGHUP)、`head_moved` と `tree_changed`、hook が git を殺す場合、fetch と push の env は、テストのコードを読んで確かめただけ
- AC13 の spawn と `start --plan` の断りは binary で打っていない。spawn は herdr と agmsg を、start は見張りを要るため。テストが固定する
- 端末の Ctrl-C が前景のプロセスグループに届く実機の確認、ssh の origin、本物の GitHub への push、Linux での動き
- gate だけの道(AC9)を binary で単独には打っていない

## Verdict

- Verdict: pass(V1 は `docs/quality/quality-gates.md` の 2 面の動詞の一覧の LOW のずれで、/sync-docs で直す。merge を止めるものではない)
- Verified: AC1〜AC8 と AC10 の本体(binary とローカルの bare origin で確認)、AC9 と AC13 のコードとテストの読み、AC11 の文書の面、AC12 の静的な部分。D1〜D9、FR-7 と FR-9、8a-3a のものを入れていないこと。実装中のずれ 6 件はどれも承認済みの計画の文と合う。self-review の N1〜N4 の直し(0b6ba232)。静的解析(run-static-verify、go vet、gofmt、go1.25.8 の build と vet、`git diff --check`、skill の 8 面の一致、check-sync、plan の digest)
- Partially verified: AC6〜AC9 と AC13 のうちテストだけが固定する場合(上の Coverage gaps)、AC12 のテストの隔離(コードを読んだ範囲)
- Not verified: テストの実行、端末の Ctrl-C の実機の確認、ssh と GitHub の origin、Linux、tech-debt 275 行の場面
