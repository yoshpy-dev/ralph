# Test report: org-state-dir-common

- Date: 2026-10-07(JST。実行の記録は UTC の 2026-10-07 07:11〜07:30)
- Plan: docs/plans/active/2026-10-07-org-state-dir-common.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: branch fix/org-state-dir-common の HEAD 43896feb と origin/main(merge base 1c4cea5a)の差分(23 ファイル、+1881/-103)。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC8 のうちテストで確かめられる部分を見た。AC9 は文言と drift check の基準なので /verify の担当
- Evidence: `docs/evidence/test-2026-10-07-org-state-dir-common.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-07-071117.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(既定の changed) | shell 39 ファイル(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 290 s、rc 0 |
| `go test ./... -count=1 -coverprofile` | 8 パッケージ(top-level の Test 関数 1,036 本。うち差分で増えたのは 14 本) | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 56 s、rc 0 |
| `go test ./internal/org/ -run 'ResolveOrgStateDir\|ParseMainWorktree\|LegacyWorktree\|CodexWritableRoot\|AddDir' -count=1 -v` | 60(top-level 13、subtest 47) | 60 | 0 | 0 | 4 s、rc 0 |
| `go test ./internal/cli/ -run Legacy -count=1 -v` | 66(top-level 37、subtest 29。うち `TestOrgLegacyLedger_*` は 5 本と subtest 18) | 66 | 0 | 0 | 10 s、rc 0 |
| `go test ./internal/cli/ -run TestStatusJSON_FromLinkedWorktreeReportsSharedLedger -count=1 -v`(AC7。`-run Legacy` には名前が当たらない) | 4(top-level 1、subtest 3) | 4 | 0 | 0 | 1.5 s、rc 0 |
| `go test ./internal/org/ -run 'ResolveOrgStateDir\|Legacy\|Permission' -count=1`(plan の Test plan の書き方) | - | ok | 0 | 0 | 3 s、rc 0 |
| 新しいテストを `-race -count=3` で(cli は `OrgLegacyLedger\|StatusJSON_FromLinkedWorktree`、org は上の 5 つ) | 3 回分 | ok | 0 | 0 | cli 25 s、org 12 s |
| mutation(私が作った 32 件。scratchpad の `git archive` の写しで実行し、worktree には触れていない) | 32 | red 30 件 | 生き残り 2 件 | - | 約 6 分 |
| ビルドしたバイナリの e2e probe(macOS、git 2.49.0) | 33 | 33 | 0 | 0 | rc 0 |
| 古い git の probe(Docker の alpine:3.4、git 2.8.6) | 4 | 4 | 0 | 0 | rc 0 |

- `run-test.sh` は `Language scope: changed (changed_languages)`、`Language packs selected: golang` で走った。shell の 39 本は `scripts/verify.local.sh` から走る。
- `run-test.sh` の中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。そのため `-count=1` で全パッケージを流し直した(2 行目)。
- shell の件数の数え方: 各 suite の区間で `PASS` / `FAIL` で始まる行(先頭の空白は許す)を数え、`PASS: 47 / 47` のような集計行は除いた。39 本とも、各 suite 自身の集計行と一致した。合計 2,047 件は #206 の test レポートの最後の件数と同じで、この差分は shell のテストを変えていない。
- 依頼にあった `tests/test-secret-scan.sh` の flake(「AC-15: missing diff.orderFile names the unscanned range」)は今回の実行では出なかった(123 / 123)。失敗したテストがないので、単独の再実行はしていない。

### shell の suite ごとの件数

| Suite | Passed / Total |
| --- | --- |
| `test-agent-models.sh` | 47 / 47 |
| `test-agent-phase-boundaries.sh` | 44 / 44 |
| `test-archive-plan.sh` | 22 / 22 |
| `test-branch-name.sh` | 26 / 26 |
| `test-check-mojibake.sh` | 15 / 15 |
| `test-check-skill-sync.sh` | 13 / 13 |
| `test-check-template.sh` | 55 / 55 |
| `test-codex-exec-invocation.sh` | 144 / 144 |
| `test-detect-changed-languages.sh` | 83 / 83 |
| `test-detect-languages-terraform.sh` | 8 / 8 |
| `test-ensure-pr-ready.sh` | 7 / 7 |
| `test-ensure-pr-title-prefix.sh` | 13 / 13 |
| `test-gc-artifacts.sh` | 11 / 11 |
| `test-hook-wiring.sh` | 68 / 68 |
| `test-insights-append.sh` | 51 / 51 |
| `test-language-pack-monorepo-roots.sh` | 29 / 29 |
| `test-new-feature-plan.sh` | 21 / 21 |
| `test-no-loop-references.sh` | 1 / 1 |
| `test-plan-visual.sh` | 103 / 103 |
| `test-post-edit-verify.sh` | 24 / 24 |
| `test-pr-owner-lookup.sh` | 20 / 20 |
| `test-pre-bash-guard.sh` | 324 / 324 |
| `test-ralph-config.sh` | 19 / 19 |
| `test-ralph-dispatch.sh` | 33 / 33 |
| `test-ralph-worktree.sh` | 143 / 143 |
| `test-run-verify-branch-secret-scan.sh` | 32 / 32 |
| `test-run-verify-scope.sh` | 19 / 19 |
| `test-secret-scan-branch.sh` | 246 / 246 |
| `test-secret-scan.sh` | 123 / 123 |
| `test-self-review-scope.sh` | 64 / 64 |
| `test-skill-insight-cycle.sh` | 20 / 20 |
| `test-sync-skills.sh` | 22 / 22 |
| `test-template-purity.sh` | 10 / 10 |
| `test-terraform-gitignore.sh` | 47 / 47 |
| `test-terraform-pack-verify.sh` | 36 / 36 |
| `test-terraform-rule-frontmatter.sh` | 11 / 11 |
| `test-verify-local-hook-tests.sh` | 5 / 5 |
| `test-verify-mode-split.sh` | 59 / 59 |
| `test-xreview-helpers.sh` | 29 / 29 |

## AC とテストの対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 main のルートとサブディレクトリ | `TestResolveOrgStateDir_GitRoot`、`_GitSubdirResolvesToToplevel`、`_LinkedWorktreeResolvesToMainWorktree` の main root / main subdirectory | pass |
| AC2 linked worktree のルートとサブディレクトリ | `_LinkedWorktreeResolvesToMainWorktree` の sibling と nested(`.claude/worktrees/nested`)の各 2 例 | pass |
| AC3 bare(`.git` 名)と `--separate-git-dir` | `_BareDotGitWorktreeFallsBackToToplevel`(2 例)、`_SeparateGitDir`(main の例と、AC にない linked worktree の例) | pass |
| AC4 flag と env の優先順、git の外 | `_ExplicitFlagWins`、`_EnvWinsOverGitAndCwd`、`_NonGitCwdFallsBackToCwd`(3 本とも差分なし) | pass |
| AC5 書き換えの 6 動詞の拒否 | `TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive`(spawn・spawn --dry-run・start・send・stop・disband・watch)、`_ExplicitStateDirSkipsGuard`(flag と env)、`_UnreadableLegacyLedger` | pass。終了コード 1 は in-process のテストでは見ていないので、e2e probe の E3 で確かめた |
| AC6 注意だけの動詞 | `_ReadOnlyVerbsNoteAndContinue`(7 例)、`_MutatingVerbWithoutActiveLegacySeats`(2 例)、`TestStatusJSON_FromLinkedWorktreeReportsSharedLedger` の main の例 | pass。「`--json` の stdout は変わらない」は、テストが見るのは JSON として読めることまでなので、e2e probe の E4 で古い台帳がない場合の stdout とバイト単位で比べた |
| AC7 `ralph status --json` の `state_dir` | `TestStatusJSON_FromLinkedWorktreeReportsSharedLedger`(3 例) | pass |
| AC8 codex の leader だけに `--add-dir` | `TestCodexWritableRootArgs`(18 例)、`TestOrgSpawn_Codex_WorkspaceWrite_AddDirWhenStateDirOutsideCwd`(2 モード × 3 例) | pass |

## Mutation

scratchpad に `git archive HEAD` で写しを作り、1 件ずつ書き換えて対象のテストを流し、元に戻した。最後に写しの `internal/` と worktree の `internal/` を `git diff --no-index` で比べ、差がないことを確かめた。M01・M07・M08 は最初の書き方だと「declared and not used」などでビルドが通らなかったので、コンパイルが通る形に直して流し直した(M01b・M07b・M08b)。件数はビルドが通った 32 件で、M07b は org と cli の両方のテストに当てた。

| 対象 | red になった mutation | 生き残り |
| --- | --- | --- |
| `statedir.go`(解決の順と戻り先) | main の中で worktree list を使う(M01b)、common dir と同じパスを通す(M02)、存在しないディレクトリを通す(M03)、bare を通す(M04)、相対パスを通す(M32)、git-main-worktree の段を外す(M09) | なし |
| `statedir.go`(古い台帳) | source の判定を外す(M05)、main の台帳と同じかの判定を外す(M06)、停止済みの座席も数える(M07b。org と cli の両方で red) | M08b: stat が ENOENT 以外で失敗したときに error を返さず「台帳なし」にする |
| `org_legacy_ledger.go` | `activeSeats >= 0`(M10)、読めない台帳で書き換えの動詞を止めない(M11)、注意を stdout に出す(M12)、`--state-dir` の案内を env の案内にする(M13) | なし |
| 呼び出し元(`org.go` の 10 か所、`status.go`、`insights.go`) | 書き換えの 6 動詞を読むだけにする(M14〜M19)、読むだけの 4 動詞を書き換えにする(M20〜M23)、`ralph status` と `ralph insights` の guard を外す(M24、M25) | なし |
| `permissions.go` / `spawn.go`(AC8) | 役割の判定を外す(M26)、symlink の解決を外す(M27)、`../` の判定を外す(M28)、guarded にも付ける(M31)、spawn の呼び出しを外す(M30) | M29: `pathWithinDir` の `rel != ".."` を外す |

生き残りの 2 件は下の Test gaps の 1 と 2。どちらも本番のコードの誤りではなく、テストがその場合を持たないことを示す。

## ビルドしたバイナリの probe

`go build ./cmd/ralph` を scratchpad に置き、scratchpad の一時リポジトリ(main と、その中の `.claude/worktrees/wt`)で動かした。PATH は git と `/usr/bin:/bin` だけにし、本物の herdr と agmsg には届かないようにした。33 件すべて pass。

- E1・E2: linked worktree のサブディレクトリと main のサブディレクトリから打った `ralph status --json` の stdout はバイト単位で同じで、`state_dir` は `<main>/.harness/state/org`、source は `git-main-worktree`。stderr は空
- E3: 古い台帳に動いている座席が 1 つあるとき、`ralph org stop` / `disband` / `send` / `watch --once` は終了コード 1。stderr に古い台帳・共通の台帳・`1 active seat(s)`・`--state-dir <古い台帳>` が出て、stdout は空。古い manifest は変わらず、共通の台帳のディレクトリはできなかった
- E4: 同じ状態で `ralph status --json` は rc 0、stdout は E1 とバイト単位で同じ。注意は stderr に 1 回。テキストの `ralph status` も注意は stderr だけ。`ralph insights --json` の注意は `RALPH_ORG_STATE_DIR=<古い台帳>` を案内した
- E5: main から打つと、worktree に古い台帳があっても注意は出ない
- E6: `RALPH_ORG_STATE_DIR` または `--state-dir` で古い台帳を選ぶと、注意も拒否もなく seat-1 が見える
- E7: 古い台帳のディレクトリを `chmod 000` にすると(stat が EACCES、`statedir.go:104` の経路)、`ralph org stop` は rc 1 で「cannot tell whether」と `--state-dir <共通の台帳>` を出し、`ralph status --json` は rc 0 で「cannot read」の注意を出して、stdout は E1 と同じ。mutation M08b が生き残った経路の本番の挙動は、これで確かめた

## 古い git での動作

self-review と verify が「コードを読んだだけ」としていた、`--path-format=absolute` がない git(2.31 より前)の経路を、Docker の alpine:3.4(git 2.8.6)で確かめた。Linux 向けにビルドしたバイナリを tar で stdin から渡した。

- git 2.8.6 は知らないオプションの `--path-format=absolute` をそのまま 1 行目に出し、出力が 3 行になる。`gitMainWorktree` は `len(dirs) != 2` で諦め、`git-toplevel` 層に落ちる
- main から打つと `<main>/.harness/state/org`(source `git-toplevel`)。linked worktree から打つと worktree 自身の `.harness/state/org`(source `git-toplevel`)で、変更前と同じ挙動になる。古い台帳の注意と拒否は出ない(source が `git-main-worktree` でないため)

つまり git 2.31 より前では、この変更は効かず、変更前の動作のまま動く。壊れることはないが、linked worktree の台帳は共通にならない。plan はこの場合を決めていない。

## Coverage

- Statement: `internal/cli` 84.8%、`internal/org` 91.2%(2026-10-05 の `rename-work-skill` のときはそれぞれ 84.7%・90.8%)。ほかの 6 パッケージは前回と同じ(`config` 92.3%、`insights` 86.1%、`org/driver` 92.0%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- 差分で足した関数: `ResolveOrgStateDir`・`parseMainWorktreeRecord`・`samePath`・`resolvedOrClean`・`guardLegacyOrgStateDir`・`codexWritableRootArgs` は 100%、`LegacyWorktreeStateDir` 90.0%、`gitMainWorktree` 85.7%、`pathWithinDir` 75.0%
- Branch / Function: Go の標準ツールでは branch coverage を取れない。関数単位は上のとおり
- Notes: 通らない行は、`statedir.go:104`(stat が ENOENT 以外で失敗)、`:95`(source が `git-main-worktree` なのに show-toplevel が失敗)、`:143`(rev-parse の出力が 2 行でない。古い git の probe で通ることを確かめた)、`:149`(main の中で show-toplevel が失敗。cwd が `.git` の中のとき)、`:155`(`git worktree list` の失敗)、`permissions.go:185`(`filepath.Rel` の失敗)、`status.go:49` と `insights.go:49` の `return err`(読むだけの guard は error を返さないので通らない)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

probe を書く途中で私の期待値の誤りが 1 件あった(org の状態がないときのテキストの `ralph status` は `state-dir: ... (source: ...)` ではなく `(state-dir=... (source: ...))` の形で出す)。期待値を直して流し直し、source が `git-main-worktree` であることは確かめた。本番のコードの問題ではない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| flag と env の優先順(`TestResolveOrgStateDir_ExplicitFlagWins`、`_EnvWinsOverGitAndCwd`) | pass、テストの中身は差分なし | ログの 2 節 |
| main のチェックアウトでの台帳の場所(既存の `GitRoot`・`GitSubdir` は source の期待値だけ `git-main-worktree` に変わり、パスは同じ) | pass | ログの 2 節。e2e の E2 も同じパス |
| codex と claude の permission の既存テスト(`TestPermissionArgsForDriver_*`、`TestOrgSpawn_Codex_*`) | pass(`internal/org` 全体が ok) | ログの 6 節 |
| `ralph org watch` の banner の source 表示(`TestOrgWatch_Once_BannerShowsStateDirSource`) | pass | ログの 6 節 |
| テストが main のチェックアウトに `.harness/state/org` を作らない(linked worktree の中から走るテストが共通の台帳に書かないこと) | 全実行のあとも main と worktree の両方に `.harness/state/org` はなく、`git status --porcelain` は両方 0 行 | ログの 10 節 |

## Test gaps

1. stat が ENOENT 以外で失敗する場合のテストがない(`statedir.go:104`、mutation M08b が生き残った)。本番の挙動は e2e の E7 で正しいことを確かめたが、回帰を止めるテストはない。verify の V-1(main のチェックアウトで stat が失敗すると、注意が「linked worktree の古い台帳」として出る)も同じ場所の話。足すなら、`TestLegacyWorktreeStateDir` に古い台帳のディレクトリを `chmod 000` にする例を足し(root で走るときは skip)、`err != nil` を見る。V-1 を直すなら、main のチェックアウトで同じことをして `("", 0, nil)` を見る例も足す
2. 台帳のディレクトリが座席の cwd の親になる場合(`filepath.Rel` が `..` を返す)のテストがない(mutation M29 が生き残った)。leader の cwd が台帳の中のディレクトリになることはまずないので、影響は小さい。足すなら、`TestCodexWritableRootArgs` に cwd が `<台帳>/prompts` の例を足して `--add-dir` を期待する
3. git 2.31 より前では共通の台帳にならず、変更前の動作のまま動く(上の「古い git での動作」)。テストではなく仕様の話で、plan も文書も最低の git の版を書いていない。PR の本文か recipe に「git 2.31 以降」と書くかは判断が要る
4. `--separate-git-dir` で git dir の名前が `.git` のリポジトリの linked worktree は、テストがない。plan の Non-goals に入っている
5. codex の座席で `--add-dir` が sandbox の中で効くかは自動のテストがない。S3 の実機確認(`docs/evidence/codex-add-dir-2026-10-07.md`、codex-cli 0.160.0)だけ
6. 読めない古い台帳の注意(`org_legacy_ledger.go:50-52`)に `--state-dir` の案内がない点(verify の V-2)は、テストも期待していない

## Verdict

- Pass: pass。`./scripts/run-test.sh` は rc 0(shell 2,047 / 2,047、Go 8 パッケージ ok)、`-count=1` の Go 全体と plan の指定の実行もすべて pass。flake は出なかった。mutation は 32 件中 30 件が red になり、生き残った 2 件は Test gaps の 1 と 2 に書いた
- Fail: なし
- Blocked: なし
