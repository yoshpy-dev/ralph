# Verify report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(AC1〜AC6 と AC3b、D1〜D6、Verify plan、Progress checklist の S1〜S5・S4b・F1〜F4)。issue #263 の受け入れ条件 5 つ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-8 と、その注「merge のあとの後始末(#263)で決めたこと」(`:132`)
- Verifier: verifier サブエージェント(`/verify`、cycle 1)
- Scope: `git diff 15e301c8...HEAD`(HEAD ef9c348a、47 ファイル、+5693/-268)。テストは打っていない(`/test` の範囲)。観察の確かめとして、HEAD の binary を scratchpad に `go build` して help を読み、使い捨ての repo で git だけの probe を打った。`go test` は打っていない
- Evidence: `docs/evidence/verify-2026-10-28-org-merge-cleanup.log`(gitignore。`run-static-verify.sh` の全出力、`git diff --check`、U+FFFD、写しの `cmp`、計画の digest、git の probe とそのスクリプト、help の本文、文書の grep)。wrapper 自身のログは `docs/evidence/verify-2026-10-28-111201.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `ralph org merge` のあと worktree・ローカルのブランチ・記録が消え、`worktree_cleaned` が台帳にあり、`CLEANUP: done` が出る。squash merge でリモートのブランチが消えたあとも止まらない | Met(静的) | 片付けを呼ぶのは `pr_merged` を記録し、`RepoRoot` があるときだけ(`internal/org/merge_run.go:258-266`)。`mergeCleanup`(`:290`)が `regate.lock` を持ったまま `cleanupFeatureWorktreeLocked` を gate の head で打つ。消すのは `scriptFeatureWorktrees.Cleanup` の `cleanup --id <id> --force-branch`(`internal/org/feature.go:683-686`)、記録は `cleanupRun.record`(`internal/org/cleanup.go:758-770`、詳細は `branch= head= via= worktree=`)。CLI は `org.FeatureRepoRoot` を渡し(`internal/cli/org_merge.go:227`、`:258`)、`printMergeCleanup`(`:344`)が `CLEANUP:` の行を出す。テストは `TestOrgMerge_CleansUpTheFeatureWorktree`(`internal/cli/org_merge_test.go:614`、実 script と bare origin)と `TestMerge_CleansUpTheFeatureWorktreeOnceRecorded`(`internal/org/merge_run_test.go:813`)。squash merge とリモートのブランチの削除のあとの片付けは、`TestCleanupFeatureWorktree_SquashMergedBranch`(`internal/org/cleanup_test.go:286`)と `TestOrgCleanup_CleansUpABranchMergedOnGitHub`(`internal/cli/org_cleanup_test.go:109`)が見る。git 2.49.0 の probe で、squash merge とリモートの削除のあとは `git branch -d` が rc 1(not fully merged)、`-D` が rc 0 だった |
| AC2 GitHub で直接 merge したブランチを `ralph org cleanup` で片付けられる。`--branch` なしなら merge 済みで記録の残るすべて、`--branch` ならそれだけ | Met(静的) | 動詞の登録は `internal/cli/org.go:82`、本体は `internal/cli/org_cleanup.go:33`。`--branch` なしは merge-order を `All: true` で読み、結びつきが org の最新(`BoundTo` が空)で `merged` のブランチをすべて選ぶ(`cleanupSteps`、`:167-198`)。`--branch` はそのブランチだけで、`merged` でなければ断る(`cleanupNamedStep`、`:204-228`)。選び方は `TestCleanupSteps_PicksTheMergedBranches`(`internal/cli/org_cleanup_test.go:491`。merged が 2 本ある中で `--branch` が 1 本だけを返す組を含む)、実 script での片付けは `:109` が見る |
| AC3 未コミット・未追跡、ブランチの先頭が違う、detached HEAD か別のブランチ、記録が binding と違う、worktree の中から打った、のどれでも何も消さず理由と次の手を出す。merge の終了コードは 0 のまま | Met(静的) | 確かめは `checkWorktree`(`internal/org/cleanup.go:527-567`、3 と 6 の順)、`checkClean`(`:572-605`、4)、`checkBranchTip`(`:617-638`、5)、`checkCheckout`(`:690-711`、6)、`checkRecord`(`:491-517`、2)。打ち直せば続く断り(3〜6)には Manual がない。Manual があるのは、記録がない・記録が違う・ブランチがなく worktree だけ・Failed のときだけで、D2 と一致する。テストは `TestCleanupFeatureWorktree_RefusedInTheWorktree`(`internal/org/cleanup_test.go:398`、13 の場面)と `TestCleanupFeatureWorktree_RefusedAtTheLedgerAndTheRecord`(`:912`)。merge の終了コードは `printMergeResult`(`internal/cli/org_merge.go:299-316`)が片付けの結果を読まないことで変わらない。CLI の「an untracked file」(`internal/cli/org_merge_test.go:651`)が `CLEANUP: skipped` と exit 0 を見る |
| AC3b 途中で止まった状態(worktree だけ消えた、記録だけ残った)から、もう一度の cleanup で残りが片付く。記録がなければ何も消さず手のコマンドを出す | Met(静的) | `run`(`internal/org/cleanup.go:358-416`)が残った状態ごとに確かめを分ける(worktree がなくブランチが残れば 5 だけ、記録だけなら 1・2 だけ)。登録だけ残った worktree は、そのパスに `git worktree remove` を打つ(`remove`、`:725-737`。prune は使わない)。記録がないときは `recordGone`(`:459-481`)が何も消さず、残ったものを消すコマンドを返す。テストは `TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft`(`internal/org/cleanup_test.go:635`)と `TestCleanupFeatureWorktree_RecordGoneRemovesNothing`(`:721`)。git の probe で、ディレクトリが消えて登録だけ残った worktree のブランチに `git branch -f` は rc 128、`git branch -D` は rc 1 で断られ、`git update-ref` は rc 0。`git worktree remove <なくなったパス>` は rc 0 で、その登録だけを外した(もう 1 つのなくなった worktree の登録は残った) |
| AC4 org が走っているブランチと、`regate.lock` が取れないときは消さない | Met(静的) | 走っている org は `cleanupFirstCheck`(`internal/org/cleanup.go:271-305`)が manifest のロックの下で断る。lock は `CleanupFeatureWorktree`(`:167-191`)が `regateLockWait` で取り、取れなければ断る。merge の側は lock を持ったまま呼ぶ。確かめ 1 を通ったときに marker に org_id を書き(`:301`)、`regateSpawnErr` がその org の spawn と start を断る(`internal/org/spawn.go:1486`)。テストは `RefusedAtTheLedgerAndTheRecord` の「the org runs」2 つ、`TestCleanupFeatureWorktree_RefusedWhileRegateLockIsHeld`(`internal/org/cleanup_test.go:1071`)、`TestCleanupFeatureWorktree_MarkerRefusesSpawnOfTheOrg`(`:826`)、CLI の `TestOrgCleanup_RefusalsAreSkippedLines`(`internal/cli/org_cleanup_test.go:218`) |
| AC5 `/org`・`/director` の後始末の節、leader の手順 15、`ralph org start --help`、regate の案内が新しい動きに合い、写しが一致する | Met(LOW 1 つ) | leader の手順 15 は `internal/org/prompts/leader.md:441-447`。`/org` skill は動詞の表の `cleanup` の行(`.claude/skills/org/SKILL.md:225`)、start の節(`:647-648`)、「org の終わり方」の後始末(`:1102-1145`)。`/director` skill は `CLEANUP:` の読み方(`.claude/skills/director/SKILL.md:859-874`)、「GitHub で merge した PR の後始末」(`:959-987`)、してはいけないこと(`:1023-1024`)。regate の案内は `internal/org/regate.go:567`。help は下の Documentation drift。写しは `/org` と `/director` の 3 つずつと recipe の 1 つが `cmp` で同一。既存の動詞の列挙 2 つに `cleanup` が入っていない(所見 LOW-1) |
| AC6 新しいテストと既存のテスト、`check-skill-sync.sh`、`check-sync.sh` が通る | Partially(静的の半分) | `check-skill-sync.sh`(14 skill が一致)と `check-sync.sh`(DRIFTED 0)は通った。`go vet ./...`(テストのファイルを含む)、golangci-lint、staticcheck が通り、テストはコンパイルできる。テストの実行は `/test` に残す |

issue #263 の受け入れ条件:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 1. `ralph org merge` のあと worktree・local branch・記録が消えている(squash merge を含む) | Met | AC1 と同じ |
| 2. GitHub で直接 merge したブランチを新しい動詞で消せる | Met | AC2 と同じ。director は `/director` skill の「GitHub で merge した PR の後始末」の手順 1 で打つ |
| 3. 未コミットの変更か未追跡のファイルがあるときは消さず、理由と打ち方を出す | Met | AC3 と同じ。issue の「やること」は「手で消すときの打ち方を出す」と書くが、計画は D2(self-review M2)で、この断りでは手で消すコマンドを出さず、直したあとに打ち直す `ralph org cleanup --branch <ブランチ>` を出す形にした。`--force-branch` の script を渡すと確かめ 5・6 を飛ばすためで、理由は計画と仕様の注にある。PR の本文で触れるとよい(Info) |
| 4. org が走っているブランチ、regate か merge が走っているブランチは消さない | Met | AC4 と同じ |
| 5. `/org` と `/director` の skill の後始末の節が新しい動きに合っている | Met | AC5 と同じ |

設計判断との照合:

- D1: 消すのは script の `cleanup --force-branch` だけで(`internal/org/cleanup.go:738`)、Go の側は確かめと記録だけを持つ。`cleanup.go` は `MergeParams.Exec` を使わず、git は `cleanupGit` から `boundedGit`(`internal/org/regate_git.go:181`。`regateBoundedGit`(`:173`)と同じ本体)で打つ
- D2: 確かめ 0〜6 の順と再開の 3 つの道は上の AC のとおり。`--force-branch` を打つのは確かめ 5(ブランチの先頭が merge された head)を通ったあとだけ。記録が違うときの Manual は、理由の文が先に `git log <head>..<branch>` で確かめるよう言う
- D3: `MergeParams.RepoRoot` が空なら片付けない(`internal/org/merge.go:91-95`)。CLI だけが渡す。`CLEANUP: done`・`CLEANUP: skipped (<理由>)`・`CLEANUP MANUAL:` の 3 つの形で、`NOTE:` と同じ形の行を `CLEANUP MANUAL:` にしたことは Progress の S2 に記録がある
- D4: 行の形、終了コード、gh がないときの断り、`NothingLeft` と `BoundTo` の扱い(`internal/org/mergeorder.go:256`、`:501`)は上の AC2 のとおり
- D5: `EventWorktreeCleaned`(`internal/org/seat.go:134`)は `stateEvents`(`:143`)に入らない。`RunningOrgs`・Roster・`orgsToDisband`・`leaderActivityEventCount` に効かないことを `TestCleanupFeatureWorktree_SquashMergedBranch` が見る。`openOrgWorkspaces` はイベントの種類で絞るので当たらない。`ralph org report` は org のイベントをすべて表に出すので、`worktree_cleaned` もそこに出る(`internal/org/report.go:217`)
- D6: `checkClean` は `git status --porcelain --untracked-files=all` を読み、gitignore されたファイルは数えない。`TestCleanupFeatureWorktree_IgnoredFilesGoWithTheWorktree`(`internal/org/cleanup_test.go:311`)

計画の digest: `./scripts/plan-visual.sh digest` が `0f38a9718a8b` を返し、`- Approved: 2026-10-28 sha256:0f38a9718a8b`(`:4`)と一致する。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | 言語の範囲は changed で、選ばれた pack は golang だけ。`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint 2.15.0 は `0 issues.`、staticcheck 2026.2.2 は出力なし。local verifier の shellcheck、hook の `sh -n`(root と template の 10 本ずつ)、guard の awk の構文、settings の `jq`、Codex の hook の検査 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照がすべて通った。secret scan は 15e301c8..ef9c348a で clean。所要 76 秒(20:11:58 から 20:13:14)。負荷による timeout はなかった |
| `./scripts/check-skill-sync.sh`(上の中で) | rc 0 | 14 skill が一致 |
| `./scripts/check-sync.sh`(上の中で) | rc 0、PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `git diff --check 15e301c8...HEAD` | rc 0 | 出力なし |
| 変更したファイルの U+FFFD | 0 件 | |
| `cmp`(`/org` と `/director` の skill の 3 つの写し、recipe の写し) | 7 つとも rc 0 | |
| `go build -o <scratchpad>/ralph ./cmd/ralph` | rc 0 | help を読むためだけに作った。go1.27.3(go.mod と同じ) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org cleanup --help` | Yes | 消すもの、`--branch` の有無での選び方、確かめの順、`CLEANED:`・`SKIPPED:`・`MANUAL:` の行、Manual が出る 3 つの場面、終了コード、見張りと承認が要らないことが D2〜D4 と仕様の注に合う |
| `ralph org merge --help` | Yes | 片付けの段落(確かめ、`CLEANUP:` の 3 つの形、Manual の場面、「The cleanup never changes the exit code」)と、終了コードの「0 for a merge that was recorded, whatever the cleanup did」が D3 に合う。spawn と start を止めるのは片付けの間だけ、の文も 8a-3a の注の変更に合う |
| `ralph org start --help`・`spawn --help`・`regate --help`・`merge-order --help`・`ralph org --help` | Yes | start は merge 済みなら merge か cleanup、閉じた PR なら `--force-branch` の script、と書く。spawn と start は片付けの間の断りを書く。regate は lock を共有する動詞に cleanup を足した。merge-order は `bound_to` を書く。`ralph org --help` に cleanup の行と説明がある |
| 仕様 FR-8 の注(`docs/specs/2026-10-07-org-multi-org-director.md:132`) | Yes | 4 段目の注(`:88`)と 8a-3a の注(`:131`)の文は日付つきの履歴として残し、`:132` がどう変えたかを 1 文ずつ書く。この仕様の書き方に合う。`docs/specs/2026-08-01-org-runtime.md` には機能の worktree の後始末の文がない |
| AGENTS.md の地図(`:90`) | Yes | `cleanup.go` の項がある |
| README.md(`:252`、`:266`、`:268`、`:270`) | Yes | 252 は merge 済みなら ralph が消す形。266 は lock を共有する動詞に cleanup を足した。268 は merge の段落に、片付けと `CLEANUP:` の行、片付けによらない exit 0、片付けの間だけ spawn と start を断ることを足した。270 は `ralph org cleanup` の段落 |
| `docs/recipes/isolated-herdr-server.md:524-535` と写し | Yes | `cmp` で同一 |
| 手で消す案内の残り(Verify plan の grep) | Yes(Info) | `cleanup --id org-` が残るのは、`/org` skill の start の失敗の案内(`:727`、`:729`、`:777`)、閉じた PR の案内(`:1143`)、cleanup の説明の中の script 名だけ。Verify plan は「閉じた PR」と「片付けの断り」の 2 つと書くが、start の失敗の案内はこの計画の前からあり、Progress(`:193`)が直さないと記録している。その記録の「1137 行」は、F2・F3 の追記で今は 1143 行(Info) |
| 既存の動詞の列挙 | 一部(LOW-1) | 下の所見 |
| `internal/org/feature.go:57-60` の注釈 | 一部(Info) | `featureWorktreeCleanupPolicy` の注釈は「The worktree outlives disband and the PR」のまま。今は merge された PR のあとに `ralph org merge` か `ralph org cleanup` が消す。「nothing removes it but `ralph-worktree.sh cleanup`」は、2 つの動詞もその script を打つので今も正しい。`--cleanup-policy manual` の理由は変わらないので、直すなら「outlives disband, and is removed once its PR is merged」程度の言い換えで足りる |
| `docs/tech-debt/README.md:383-387` | Yes | L5、L6、R2-L5、R3-L2、R3-L3 の 5 行がある。行数の記述は HEAD の `wc -l` と合う(`cleanup.go` 846 行、`org_cleanup_test.go` 800 行、L5 の 4 ファイルはすべて 800 行を越える) |
| 計画と self-review の記録 | Yes(Info) | F4 の記録(計画 `:207`、self-review の report の末尾の節)は `git show --stat 3dfd434e` の 3 ファイル +124/-41 と合う。insight event は self-review の run 1 と run 2 の 2 行だけで、run 3 の行はない(Info。verdict には関わらない) |

所見 LOW-1: 新しい動詞 `cleanup` が、既存の動詞の列挙 2 つに入っていない。

- (a) 古い台帳の拒否。`ralph org cleanup` は `guardLegacyOrgStateDir(..., orgLedgerMutating)` を打ち(`internal/cli/org_cleanup.go:108`)、linked worktree の古い台帳に動いている座席があれば exit 1 で断る(`TestOrgCleanup_RefusedWhileALegacyLedgerHasActiveSeats` がある)。ところが、台帳を書き換える動詞の一覧の `/org` skill(`.claude/skills/org/SKILL.md:91-95`、3 つの写しも同じ)と、Go の注釈(`internal/cli/org_legacy_ledger.go:21-26`)は、`regate` / `merge` を挙げて `cleanup` を挙げていない
- (b) 見張りがなくても使える動詞。`/org` skill の `:1642-1645` は `approve`・`gate`・`regate`・`merge`・`merge-order` などを挙げ、`cleanup` を挙げていない。`cleanup` の行(`:225`)は「見張りも承認された計画も要らない」と書き、`cleanup.go` と `org_cleanup.go` は `checkWatchGate` を呼ばないので、食い違いではなく列挙の漏れ
- 影響: (a) は、古い台帳を残した worktree から打つ人が、cleanup は止まらないと読みうる。実際は断られ、その文が古い台帳と共通の台帳のパスを示すので、何も壊れない。(b) は行の本文で補える
- 手当ての案: 2 つの一覧に `cleanup` を足す(skill は `scripts/sync-skills.sh` で 4 つの写し)。skill の文を変えるので、記録だけの修正の例外には当たらない。この PR で直さないなら、sync-docs で tech-debt に 1 行記録する

## Observational checks

- git の probe(git 2.49.0、使い捨ての repo、スクリプトと出力は証拠のログの 6 節): コードの注釈が頼る git の動きを 3 つ確かめた。(a) ディレクトリが消えて登録だけ残った worktree のブランチに、`git branch -f` は rc 128、`git branch -D` は rc 1 で断られ、`git update-ref` は rc 0(`internal/org/cleanup.go:397-404` の注釈と、`recordGone` のテストの注釈どおり)。(b) `git worktree remove <なくなったパス>` は rc 0 で、その 1 つの登録だけを外した。(c) squash merge とリモートのブランチの削除のあと、`git branch -d` は rc 1、`git branch -D` は rc 0
- help の本文は HEAD から作った binary で読んだ(証拠のログの 7 節)
- F4 の退避の名前(`<branch>-unmerged-<tip の先頭 7 文字>`)を読んだ。同じ tip から前に退避したブランチがあり、断りのあとにブランチへコミットを足し、退避の `git branch` の失敗を見たまま reset まで打つと、足したコミットはブランチから外れて reflog にだけ残る。R3-L1 と同じ「失敗を見ても続けて打つ」場面で、前提がさらに 2 つ重なるときだけ起きる。直さなくてよい(Info)
- merge の側のテスト(`TestMerge_CleansUpTheFeatureWorktreeOnceRecorded` と CLI の `TestOrgMerge_CleansUpTheFeatureWorktree`)は、リモートのブランチを消さない。squash merge と削除のあとの片付けは cleanup の側のテストが見る。merge の側も同じ `cleanupRun.remove` と `--force-branch` を通るので、AC1 の squash の部分はこのコードの共有で満たすと読んだ

## Coverage gaps

- テストの実行(`internal/org` と `internal/cli` の新しいテストと既存のテスト)と、変異が落ちることの確かめは `/test` に残す
- 既知の穴として受け入れたもの(失敗ではない):
  - `runWorktreeScript` は script を時間の上限なしで打つ(tech-debt `:384` の (a)、L6)
  - `StartFeature` の `regateSpawnErr` は manifest のロックの外で打たれる(`internal/org/feature.go:224`。tech-debt `:384` の (b)、L6。Regate と同じ窓)
  - tech-debt の行 L5(`:383`)、R2-L5 の 2 つの quote の関数(`:385`)、R3-L2(`:386`)、R3-L3(`:387`)
- 実機の director と leader が、新しい手順 15 と skill の文のとおりに動くことは確かめていない
- 古い binary の leader の雛形が「人が消す」のままであることの影響は、計画のリスクの表のとおり(片付けは merge の側がするので結果は同じ)。確かめていない

## Verdict

- Verified: AC1〜AC5(静的)、issue の受け入れ条件 1〜5、D1〜D6、`ralph org merge` の「exit 0 は記録した merge」が変わっていないこと、`--force-branch` の前に head の照合があること、help と仕様の注と README と AGENTS.md の一致、写しの一致、計画の digest、git の動きの 3 つ、`run-static-verify.sh`・`check-sync.sh`・`check-skill-sync.sh` の通過
- Partially verified: AC6(同期の検査、`go vet`・golangci-lint・staticcheck は通った。テストは `/test`)
- Not verified: テストの実行、実機の座席と director の振る舞い
- Result: pass。所見は LOW-1(新しい動詞 `cleanup` が古い台帳の拒否と見張りなしの動詞の 2 つの列挙に入っていない。直すか tech-debt に記録するかは lead が決める)。Info は、issue の「打ち方」と D2 の打ち直しの違い、`feature.go:57-60` の注釈、Progress の「1137 行」、self-review の run 3 の insight event がないこと、F4 の退避の名前に残る狭い場面、merge の側のテストがリモートのブランチを消さないこと
