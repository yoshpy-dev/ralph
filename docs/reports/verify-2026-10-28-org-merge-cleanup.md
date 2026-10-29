# Verify report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(AC1〜AC6 と AC3b、D1〜D6、Verify plan、Progress checklist の S1〜S5・S4b・F1〜F4)。issue #263 の受け入れ条件 5 つ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-8 と、その注「merge のあとの後始末(#263)で決めたこと」(`:132`)
- Verifier: verifier サブエージェント(`/verify`、cycle 1)
- Scope: `git diff 15e301c8...HEAD`(HEAD ef9c348a、47 ファイル、+5693/-268)。テストは打っていない(`/test` の範囲)。観察の確かめとして、HEAD の binary を scratchpad に `go build` して help を読み、使い捨ての repo で git だけの probe を打った。`go test` は打っていない
- Evidence: `docs/evidence/verify-2026-10-28-org-merge-cleanup.log`(gitignore。`run-static-verify.sh` の全出力、`git diff --check`、U+FFFD、写しの `cmp`、計画の digest、git の probe とそのスクリプト、help の本文、文書の grep)。wrapper 自身のログは `docs/evidence/verify-2026-10-28-111201.log`

この下の「Spec compliance」から「Coverage gaps」までは cycle 1(HEAD ef9c348a)の記録で、行番号もその時点のもの。パイプラインの cycle 2(cross-review cycle 1 の指摘の直し、HEAD 09e1e93b)の verify は、「Coverage gaps」と「Verdict」の間の「cycle 2(HEAD 09e1e93b)」の節にある。いまのブランチの verdict は「Verdict」の節の先頭の cycle 2 の行で、cycle 1 の verdict はその下の「cycle 1(HEAD ef9c348a)」に残している。cycle 2 の所見の ID は `C2-D<n>`。

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

## cycle 2(HEAD 09e1e93b)

- Date: 2026-10-29
- Verifier: verifier サブエージェント(`/verify`、cycle 2)
- Scope: cycle 2 で変わったのは、cross-review cycle 1 の指摘 2 つの直し(8e46db0f S-X1、d3452a9d S-X2)、self-review cycle 2 の直し(54fa6351 C2-F、060102bb C2-L5)、記録のコミット(33b78f8c、04c83f5c、21785882、fdb9ee67、09e1e93b)。受け入れ条件はブランチ全体(`git diff main...HEAD`、merge-base 7522835c、57 ファイル、+7359/-335)に照らし直した。cycle 1 の verify のあとの /test・/sync-docs・main の merge(b164744f)・triage のコミットは、それぞれの report が扱うので読み直していない。テストは打っていない。観察の確かめは、HEAD から作った binary の help と、使い捨ての repo での git の probe 1 つだけ
- Evidence: `docs/evidence/verify-2026-10-28-org-merge-cleanup-cycle2.log`(gitignore、653 行。`run-static-verify.sh` の全出力、単独で打った同期の検査 2 つ、`git diff --check`、U+FFFD、写しの `cmp`、計画の digest、git の probe とそのスクリプト、help の本文、文書の grep。scratchpad で組んで `cp` し、`cmp` で同じことを確かめた)。wrapper 自身のログは `docs/evidence/verify-2026-10-29-114658.log`。cycle 1 の証拠のログは書き換えていない

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | 言語の範囲は changed で、選ばれた pack は golang だけ。`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし。local verifier の shellcheck、hook の `sh -n`、settings の `jq`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照がすべて通った。secret scan は 7522835c..09e1e93b で clean。所要 91 秒(20:46:58 から 20:48:29) |
| `./scripts/check-skill-sync.sh`(単独) | rc 0 | 14 skill が一致 |
| `./scripts/check-sync.sh`(単独) | rc 0、PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6。cycle 1 の 11 と 5 から変わったのは、merge した main の `scripts/check-sync.sh` が root の `ralph.toml` を KNOWN_DIFFS に足したため(issue #265)で、このブランチの変更ではない |
| `git diff --check main...HEAD -- ':!docs/evidence'` | rc 0 | 出力なし |
| 変更したファイルの U+FFFD | 0 件 | `git diff --name-only main...HEAD` の全ファイル |
| `cmp`(`/org` と `/director` の skill の 3 つの写し、recipe の写し) | 7 つとも rc 0 | |
| `./scripts/plan-visual.sh digest` | `0f38a9718a8b` | `- Approved:` の行(`:4`)と一致。cycle 2 の計画の変更は Progress checklist の hunk(`:209` から)だけ |
| `go build -o <scratchpad>/ralph-c2 ./cmd/ralph` | rc 0 | help を読むためだけに作った。go1.27.3、git 2.49.0 |

### Spec compliance(cycle 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | Met(静的) | cycle 2 は `internal/org/merge_run.go` と `internal/cli/org_merge.go` を変えていない。merge の片付けも `cleanupRun.run`(`internal/org/cleanup.go:375`)を通るので、S-X2 の登録の確かめは merge の続きにもかかる。merge の `readOrigin`(`internal/org/merge.go:360`)は `ReadMergeOrigin`(`internal/org/mergeorigin.go:89`)を使う形になり、断りの 3 つの文は diff で変わっていない。`isOriginsPR`(`internal/org/merge.go:136`)は `MergeOrigin.hasPR` に任せ、比べ方(host と owner/repo、大文字小文字を無視)は同じ |
| AC2 | Met(静的) | cleanup は merge-order を読む前に origin を読み(`cleanupReadOrigin`、`internal/cli/org_cleanup.go:168`)、`MergeOrderParams.Origin` を渡す(`:145`)。merge-order は `gh pr list` に `-R <host>/<owner>/<repo>` を付け、PR の url の host と owner/repo が origin と違えば全ブランチで PR を使わない(`lookupMergePRs`、`internal/org/mergeorder.go:646`、`:662`、`:684-688`)。`ralph org merge-order` の CLI(`internal/cli/org_merge_order.go:155`)は Origin を渡さず、一覧の動きは変わらない。足した `PRLookupWhy` は `json:"-"`(`internal/org/mergeorder.go:227`)で、`--json` も変わらない。前提が 1 つ増え、origin が GitHub の repository を名指さないか読めなければ何も消さずに exit 1 で断る(`cleanupOriginRefusal`、`internal/cli/org_cleanup.go:182`)。AC2 は GitHub で merge したブランチについての条件で、この断りは Non-goals の「gh のない環境で merge 済みを判定する道」と同じく判定できないときに消さない側にあるので、AC2 と食い違わない。テストは `TestOrgCleanup_CleansUpABranchMergedOnGitHub`(`internal/cli/org_cleanup_test.go:118`。gh の呼び出しを `-R github.com/o/r` 付きの 1 回と見る、`:154`)、`TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing`(`:232`)、`TestOrgCleanup_OriginThatNamesNoGitHubRepositoryIsRefused`(`:279`)、`TestOrgCleanup_OriginRefusalForEachProblem`(`:751`)、`TestMergeOrder_PinnedOriginReadsThePRsInItsRepository`(`internal/org/mergeorder_test.go:2224`)、`TestMergeOrder_PinnedOriginLeavesEveryPROutForAPROfAnotherRepository`(`:2264`)、`TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot`(`internal/org/merge_test.go:636`) |
| AC3 | Met(静的) | 確かめ 6 が、ディレクトリのない worktree の登録にも延びた(`checkRegistration`、`internal/org/cleanup.go:778`。`run` の `:412-424`)。登録が detached、別のブランチ、消えたブランチの上、HEAD が読めない、のどれかなら何も消さず、理由は登録の HEAD と、退避のコマンドか reflog の場所を名指す(`registrationAtRisk`、`:812`)。この断りは打ち直しても通らない(登録はそのまま残る)ので、D2 の原則(ralph が先に進めず人の判断が要るとき)どおり Manual(`git worktree remove <worktree> && <script>`、`:790`)を返す。ブランチが残っていれば、理由の文は Manual の前に `git log <head>..<branch>` で確かめるよう言う(`:785-788`)。記録が違うときの Manual と同じ並びで、`--force-branch` の script を確かめなしに渡す道にはならない。merge の終了コードは `printMergeResult`(`internal/cli/org_merge.go:299`)が片付けの結果を読まないことで変わらない(cycle 2 の変更なし)。テストは `TestCleanupFeatureWorktree_GoneWorktreeRegisteredOffItsBranchRemovesNothing`(`internal/org/cleanup_test.go:805`、4 つの場面)と `TestCleanupFeatureWorktree_RecordGoneNamesADetachedRegistrationsHead`(`:1008`、ディレクトリが残る場合と消えた場合) |
| AC3b | Met(静的) | script の止まり方で残る 2 つの状態は、今も続きから片付く。script が worktree を消したあと(登録も一緒に消える)は `case st.branchTip != ""`(`internal/org/cleanup.go:425-429`)、記録だけなら `default`(`:430-431`)。ディレクトリを人が消し、登録がブランチの上に残っているときも、`checkRegistration` が何も返さず確かめ 5 に進む。新しく断るのは、登録が残ったままブランチが消えた状態で、cycle 1 は記録だけの再開として登録を外していた(計画 `:217` の「挙動の変化」)。この状態は script の止まり方では生じない。script はディレクトリがあるときだけ `git worktree remove` を打ち、それが登録も外す(`scripts/ralph-worktree.sh:390-394`、probe の (d))。登録が残る間は `git branch -D` が rc 1 で断る(probe の (b))ので、ブランチを消すには `git update-ref -d` のような直の操作が要る(probe の (c)。そのとき登録の HEAD はゼロになり、branch の行は残る)。AC3b の「途中で止まった状態」は狭まっていない。テストは `TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft`(`internal/org/cleanup_test.go:705`)の 3 つの道とブランチが動いた 2 つの道で、cycle 2 で消えたケースはない(d3452a9d と 54fa6351 のテストのファイルに、`t.Run` やケースの削除行はない)。記録がないときは今も何も消さず、手で消すコマンドを出す(`recordGone`、`:502`)。登録が危ない形なら、同じ退避の文を足す(`:519-521`) |
| AC4 | Met(静的) | cycle 2 は `CleanupFeatureWorktree`、`cleanupFirstCheck`、`regate.lock` の扱いを変えていない(`internal/org/cleanup.go` の diff の hunk は注釈、`CleanupResult`、`cleanupState`、`run`、`readState`、`recordGone`、`branchMovedSteps`、新しい関数だけ)。origin の読み取りは lock の前に打ち、何も書かない |
| AC5 | Partially(/sync-docs) | leader の手順 15、`ralph org start --help`、regate の案内は cycle 2 の変更に関わらない。`ralph org cleanup --help` は origin への固定と断りの段落を足した(`internal/cli/org_cleanup.go:66-73`、終了コードの `:101-102`)。一方、cycle 2 の 2 つの動き(origin への固定と断り、登録の断りの Manual)が、`/org` skill の動詞の表と「org の終わり方」(と 3 つの写し)、README、仕様の注、2 つの help の Manual の列挙にまだない(下の C2-D1、C2-D2)。写しは一致する |
| AC6 | Partially(静的の半分) | `check-skill-sync.sh` と `check-sync.sh` は通った。`go vet ./...`(テストのファイルを含む)、golangci-lint、staticcheck が通り、テストはコンパイルできる。テストの実行は `/test` に残す |

issue #263 の受け入れ条件: 1(AC1)、2(AC2。origin が GitHub の repository を名指すことが前提に加わった)、3(AC3)、4(AC4)は Met。5 は AC5 と同じく Partially で、/sync-docs で直す。

設計判断との照合(cycle 2 の差分):

- D1: 消すのは今も script の `cleanup --force-branch` だけ。登録の断りの Manual も script を打つ
- D2: 確かめ 6 を登録に延ばしたことで、計画の本文の再開の文(「3・4・6 は worktree がないのでかけず、1・2・5 だけ」「worktree もブランチもなく、記録だけが残る: 1・2 だけ」、`:76-77`)と、登録が残る場合に食い違う。承認後に本文を変えない運用で、Progress(`:215`、`:217`)に指摘と挙動の変化が書いてある。記録された逸脱で、所見にしない(Info)。Manual の場面の列挙(`:66`)は括弧の中の例で、新しい場面は原則の文(ralph が先に進めず人の判断が要るとき)に合う
- D3: 変わらない。登録の断りは `CLEANUP: skipped (<理由>)` と `CLEANUP MANUAL:` で出る
- D4: 「gh がないか PR を読めないときは、merge 済みを判定できないと言って 1 で終わる」に、origin が GitHub の repository を名指さないか読めないときの 1 が加わった。判定できなければ消さないという D4 の方針の延長。SKIPPED の行と `--branch` の断りは、PR を使えなかった理由を括弧で出す(`cleanupLookupWhy`、`internal/cli/org_cleanup.go:300`)
- D5、D6: 変わらない

仕様 FR-8 との照合: FR-8 の要件の行(`docs/specs/2026-10-07-org-multi-org-director.md:130`)は merge の既定、`auto_merge`、merge してよい順についてで、cycle 2 の変更はそれに触れない。古くなったのはこの PR が足した注(`:133`)の 3 か所で、C2-D1 と C2-D2 に入れた。

### Documentation drift(cycle 2)

/sync-docs に渡す所見。直していない。

所見 C2-D1(LOW): cleanup の PR の読み方の文が、origin への固定と origin の断りを書いていない。

- `.claude/skills/org/SKILL.md:234`(cleanup の行)と 3 つの写し(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行)。「`merge-order --all` と同じく台帳のブランチを読み」とだけ書き、`-R <host>/<owner>/<repo>` と、別の repository の PR が返れば全ブランチで PR を使わないことがない。終了コード 1 の列挙にも、origin が GitHub の repository を名指さないか読めないときがない
- `README.md:278`。「reads the ledger's branches as `merge-order --all` does」と、終了コード 1 の列挙(skip と gh がないとき)
- `docs/specs/2026-10-07-org-multi-org-director.md:133`。「merge-order を `--all` と同じに読み」と「gh がなければ…1 で断る」
- `.claude/skills/director/SKILL.md:986-987`(「GitHub で merge した PR の後始末」の手順 5)と 3 つの写し。何も消さずに 1 で終わる場面として gh がないときだけを挙げる。手順 3 の「stderr の文をそのまま人に伝える」で補えるので、ここは Info 寄り
- help は在り(`internal/cli/org_cleanup.go:66-73`)。`:53` の「as ralph org merge-order --all does」は、続く段落が補う
- 関連: tech-debt の ssh の別名の行(`docs/tech-debt/README.md:247`)は、merge が断るので「the human merges in GitHub」と書く。cycle 2 から cleanup も origin の host を `-R` に渡すので、ssh の別名の origin(`github.com.emu` など)では gh が失敗し、全ブランチが SKIPPED になる。計画 `:216` が「PR の既知の穴」と書いているが、`/pr` で計画が archive されたあとも追えるよう、この行に cleanup も足すとよい。cycle 1 の cleanup は `-R` なしで gh に repository を選ばせていたので、ssh の別名でも PR を読めたかもしれない。おそらくそうだが、gh の別名の解き方は確かめていない

所見 C2-D2(LOW): 手で消すコマンドが出る場面の列挙に、cycle 2 の登録の断りがない。

- 列挙は「記録がないか結びつきと合わない、ブランチがなく worktree だけがある、消す途中で失敗した」。cycle 2 で、ディレクトリのない worktree の登録が binding のブランチにいない(detached、別のブランチ、HEAD が読めない)ときにも Manual が出るようになった(`internal/org/cleanup.go:790`)。登録が binding のブランチにいてそのブランチが消えた場合は、「ブランチがなく worktree だけがある」と読める
- `internal/cli/org_cleanup.go:94-96`(`ralph org cleanup --help`)。固定するテストは `internal/cli/org_cleanup_test.go:913`(`TestOrgCleanup_Help`)
- `internal/cli/org_merge.go:133-134`(`ralph org merge --help`)。テストは `internal/cli/org_merge_test.go:1070`(`TestOrgMerge_Help`)
- `.claude/skills/org/SKILL.md:233`(merge の行)、`:234`(cleanup の行)、`:1138-1142`(「org の終わり方」。「の 3 つ」と数を書く)と 3 つの写し。テストは `internal/org/org_skill_send_sync_test.go:166`、`:180`、`:202-204`
- `README.md:278`、`docs/specs/2026-10-07-org-multi-org-director.md:133`
- 影響: director は `CLEANUP MANUAL:` と `MANUAL:` の行をそのまま人に伝える(`.claude/skills/director/SKILL.md:869`、`:976-977`)ので、列挙が古くても行は人に届く。読み手が「Manual が出るのは 3 つの場面だけ」と受け取る
- 直すと help と skill の文を固定するテストも変わるので、記録だけの修正の例外には当たらない

所見 C2-D3(Info): 確かめ 6 の言い方。skill の cleanup の行(`:234`)、README `:278`、2 つの help、仕様 `:133` の「worktree がそのブランチをその head でチェックアウトしている」と、「org の終わり方」の断りの一覧の「worktree が detached HEAD か、別のブランチにいる」(`.claude/skills/org/SKILL.md:1130`)は、ディレクトリのない worktree の登録にも同じ確かめがかかることを言わない。誤りではないので、C2-D2 を直すときに一緒に足す程度でよい。

所見 C2-D4(Info): AGENTS.md の地図(`:90`)は `cleanup.go` を挙げ、merge と cleanup が共有する origin の読み取り `internal/org/mergeorigin.go` を挙げない。地図はすべてのファイルを挙げる形ではないので、足すかは任意。

所見 C2-D5(Info): 計画の Progress は、self-review cycle 2 run 2 の結果(fdb9ee67。LOW 3、Merge 可)と台帳の行(09e1e93b)をまだ書いていない(`:218-221`)。`/pr` の前に足せばよい。

在るもの: `ralph org cleanup --help` の origin の段落と終了コード、`/director` skill の `CLEANUP:` の読み方(行をそのまま伝え、Manual のない断りは打ち直す)、recipe、leader の手順 15、`ralph org start --help`、regate の案内。tech-debt の #263 の 6 行(`docs/tech-debt/README.md:383-388`)は、self-review の report の R3-L2・R3-L3・C2R2-L1〜C2R2-L3 の扱いと合う。

### Observational checks(cycle 2)

- git の probe(git 2.49.0、使い捨ての repo。スクリプトと出力は証拠のログの 4 節): (a) ディレクトリを消した worktree 2 つについて、`git worktree list --porcelain` はブランチの上のものに `HEAD <sha>` と `branch refs/heads/feat/a`、detached のものに `HEAD <sha>` と `detached` を、どちらも `prunable` の行と一緒に出す(`readState`、`internal/org/cleanup.go:459-477` が頼る形)。(b) その登録がチェックアウトしているブランチへの `git branch -D` は rc 1。(c) `git update-ref -d` は rc 0 で、そのあと登録は `HEAD 0000…` と branch の行を出す(`registrationAtRisk` のゼロの分かれ)。(d) ディレクトリがあれば `git worktree remove` は登録も外し、ブランチは残る(script の順)
- help の本文は HEAD から作った binary で読んだ(証拠のログの 5 節)

### Coverage gaps(cycle 2)

- テストの実行(cycle 2 の新しいテストと既存のテスト)と、変異が落ちることの確かめは `/test` に残す
- 実際の gh で cleanup を打つ確かめ(`-R` の値を gh が受けること、別の host の PR の url)はしていない。テストは偽の gh で見る
- ssh の別名の origin で cleanup が全ブランチを SKIPPED にすることは、計画 `:216` の記録とコードを読んだだけで、確かめていない
- cleanup は origin を打った場所で読み、消すのは state dir から引いた main worktree の下(self-review cycle 2 の Known gaps)。`--state-dir` で別の repo の台帳を指したときの組み合わせは確かめていない
- 実機の director と leader の振る舞いは cycle 1 と同じく確かめていない

## Verdict

- Verdict(cycle 2、HEAD 09e1e93b): pass
- Verified: AC1〜AC4 と AC3b(静的)、issue の受け入れ条件 1〜4、D1〜D6 の cycle 2 の差分、`ralph org merge-order` の一覧と `--json` が変わらないこと、merge の origin の断りの文が変わらないこと、script の止まり方で残る状態が新しい断りに入らないこと(git の probe)、`run-static-verify.sh`・`check-skill-sync.sh`・`check-sync.sh` の通過、写しの一致、計画の digest
- Partially verified: AC5 と issue の受け入れ条件 5(C2-D1・C2-D2 の文書と help の残り)、AC6(同期の検査と静的解析は通った。テストは `/test`)
- Not verified: テストの実行、実際の gh での cleanup、ssh の別名の origin での cleanup、実機の座席と director の振る舞い
- Result: pass。動きが受け入れ条件と食い違う所見はない。所見は LOW 2 つ(C2-D1: cleanup の PR の読み方の文に origin への固定と断りがない。C2-D2: 手で消すコマンドの場面の列挙に登録の断りがない)と Info 3 つ(C2-D3〜C2-D5)で、どれも /sync-docs に渡す。C2-D2 は help と skill を固定するテストも変わる

### cycle 1(HEAD ef9c348a)

- Verified: AC1〜AC5(静的)、issue の受け入れ条件 1〜5、D1〜D6、`ralph org merge` の「exit 0 は記録した merge」が変わっていないこと、`--force-branch` の前に head の照合があること、help と仕様の注と README と AGENTS.md の一致、写しの一致、計画の digest、git の動きの 3 つ、`run-static-verify.sh`・`check-sync.sh`・`check-skill-sync.sh` の通過
- Partially verified: AC6(同期の検査、`go vet`・golangci-lint・staticcheck は通った。テストは `/test`)
- Not verified: テストの実行、実機の座席と director の振る舞い
- Result: pass。所見は LOW-1(新しい動詞 `cleanup` が古い台帳の拒否と見張りなしの動詞の 2 つの列挙に入っていない。直すか tech-debt に記録するかは lead が決める)。Info は、issue の「打ち方」と D2 の打ち直しの違い、`feature.go:57-60` の注釈、Progress の「1137 行」、self-review の run 3 の insight event がないこと、F4 の退避の名前に残る狭い場面、merge の側のテストがリモートのブランチを消さないこと
