# Test report: org-gate

- Date: 2026-10-13(JST 16:35〜17:21)
- Plan: docs/plans/active/2026-10-13-org-gate.md(`Status: Approved`。`./scripts/plan-visual.sh digest` の出力 69f088275e54 は `- Approved:` 行と一致する)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1)
- Scope: HEAD eb5ffdac(base 1bddc871 = origin/main、13 コミット、34 ファイル、+6,595/-207)。/verify が見た c84d8e19 から HEAD までに変わったのは verify の報告と evidence、insight event、計画の 7 ファイルだけで、コードとテストは /verify のときと同じ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、12 コア)、go 1.26.0、git 2.49.0、`/bin/sh` は bash 3.2.57、dash と perl あり
- Evidence(どれもコミットする。`docs/evidence/verify-*.log` は gitignore の対象なので使っていない):
  - `docs/evidence/test-2026-10-13-org-gate-run-test.txt`: `./scripts/run-test.sh`(changed)と `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` の 2 回の出力。どちらも末尾に `RC=` がある
  - `docs/evidence/test-2026-10-13-org-gate-go-test.txt`: `go test ./... -count=1 -v -cover`
  - `docs/evidence/test-2026-10-13-org-gate-race.txt`: `go test -race -count=1 -v ./internal/org/ ./internal/cli/`
  - `docs/evidence/test-2026-10-13-org-gate-shell.txt`: 変更が直接かかわる shell のテスト 6 本を `sh` で直接打った出力
  - `docs/evidence/test-2026-10-13-org-gate-flaky.txt`: concurrency のテストの 5 回連続、負荷をかけた 3 回、gate のテストの `-count=3`
  - `docs/evidence/test-2026-10-13-org-gate-mutation.txt`: mutant の overlay と差分、結果、落としたテストと最初の失敗の行、probe のテストの全文と結果
  - wrapper が書いた `docs/evidence/verify-2026-10-13-073507.log` と `verify-2026-10-13-074441.log` はコミットしない
  - insight event: `docs/insights/events/2026-10-13-org-gate.jsonl` に test の 1 行(`--cycle auto`)を足し、この報告と同じコミットに入れた
- テストもコードも変えていない。mutant の写し、probe のテスト、計測用に手を入れたテストの写しは scratchpad の `og/` に置いた。worktree に書いたのは、この報告、上の evidence、insight event だけ。テストの前後で `~/.config/herdr` の下に 16:30 より新しいファイルはなく、main のチェックアウトと worktree の `git status --short` は空のまま、`$TMPDIR` に `ralph-verify-*` の一時ファイルは残っていない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`scripts/run-verify.sh` が差分にあるので `shared:scripts/run-verify.sh` で full に落ちた) | shell 41 本(PASS の行 4,118)、Go 8 パッケージ | すべて | 0 | shell 1 | 9 分 23 秒、rc 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | shell 41 本(PASS の行 4,118)、Go 8 パッケージ | すべて | 0 | shell 1 | 9 分 1 秒、rc 0 |
| `go test ./... -count=1 -v -cover` | top-level 1,541、subtest 2,001 | top-level 1,538、subtest 2,001 | 0 | 3 | 3 分 38 秒(cli 214.4 s、org 104.3 s)、rc 0 |
| `go test -race -count=1 -v ./internal/org/ ./internal/cli/` | top-level 1,259、subtest 1,804 | top-level 1,258、subtest 1,804 | 0 | 1 | 3 分 28 秒(cli 201.9 s、org 104.1 s)、rc 0。`DATA RACE` は 0 件 |
| shell のテスト 6 本を直接(下の表) | 176 | 176 | 0 | 0 | 47 秒、どれも rc 0 |
| 揺れの確認(下の「Flakiness」) | concurrency 5 回 + 負荷下 3 回、gate のテスト 61 本 × 3 回 | すべて | 0 | 0 | - |
| mutation(Go 10 件、shell 5 件) | 15 件 | red 15 件 | 生き残り 0 | - | Go は 1 件 36〜82 s、並列 2 |

- shell の PASS の行は、`PASS` を含む行から集計の行(`PASS: <数>` と `PASS: <数> / <数>` だけの行、25 本)を除いて数えた。前の /test(org-plan-approval、40 本、4,081)より suite が 1 本、行が 37 増えた。増えたのは新しい `tests/test-run-verify-concurrency.sh` の 37 件で、ほかの suite の数は変わらない。changed と full の 2 回の `PASS` の行は、経過時間の数字を除くと 1 行も違わなかった
- `FAIL` を含む行は、集計の `FAIL: 0` と、PASS の行の中の文(たとえば `PASS: ... (output names FAIL: ...)`)だけだった。shell の Skipped は前と同じ `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので、2.41 より古い git の場合を飛ばす)
- `run-test.sh` の中の Go は、2 回とも `internal/cli` と `internal/org` を流し直し(changed で 133.5 s と 63.0 s、full で 128.2 s と 63.7 s)、ほかの 6 パッケージは `(cached)` だった。キャッシュを使わない結果は 3 行目にある
- Go の件数は、前の /test(top-level 1,507、subtest 1,925)より top-level が 34、subtest が 76 増えた。差分で足した `func Test` は 34 本で(`git diff 1bddc871..HEAD -- '*_test.go'` の `+func Test`)、どれも PASS だった。消えた `func Test` は 0。`-race` も top-level と subtest がちょうど 34 と 76 増えた(前は 1,225 と 1,728)。Go の Skipped は前と同じ 3 件(`-race` は org と cli だけなので 1 件)
- load average は、changed の `run-test.sh` の開始時に 4.18、full の開始時に 4.94、`go test ./...` の開始時に 6.03、`-race` の開始時に 6.33 だった。前に flake を起こした doctor のテスト(`TestRunDoctor*`)と `TestRunWatcher_TimeoutIndependentOfSmallInterval` は、どの実行でも落ちなかった

### 直接打った shell のテスト

| テスト | 結果 | 時間 |
| --- | --- | --- |
| `tests/test-run-verify-concurrency.sh`(新規) | 37 / 37、rc 0 | 14.2 s |
| `tests/test-run-verify-scope.sh` | 19 / 19、rc 0 | 9.1 s |
| `tests/test-verify-mode-split.sh` | 59 / 59、rc 0 | 5.6 s |
| `tests/test-run-verify-branch-secret-scan.sh` | 32 件 PASS、0 FAIL、rc 0 | 13.9 s |
| `tests/test-verify-local-hook-tests.sh` | 5 passed、0 failed、rc 0 | 3.4 s |
| `tests/test-post-edit-verify.sh` | 24 件 PASS、0 FAIL、rc 0 | 1.0 s |

concurrency のテストは dash があるので、Case 6 を `/bin/sh` と dash の 2 つで回す(37 件。計画 Notes の「28 件」は S3 の時点の数)。perl もあるので、プロセスのグループに送る部分も飛ばしていない。

## Coverage

- Statement(`go test ./... -count=1 -cover`): `internal/org` 93.5%(前の /test は 94.4%)、`internal/cli` 86.7%(86.6%)、`internal/config` 93.3%、`internal/org/protocol` 100.0%、driver 93.1%、insights 86.1%、scaffold 75.7%、upgrade 91.2%(この 6 つは前と同じ)
- `internal/org` の下がり方は、新しい 2 ファイルの分で説明がつく。`go test -count=1 -coverprofile ./internal/org/`(rc 0、69.7 s、93.5%)の profile で、`gate.go` は 222/261 文(85.1%)、`gate_run.go` は 96/112 文(85.7%)、その 2 つを除いた org は 3,475/3,682 文(94.4%)で前と同じ。`internal/cli/org_gate.go` は `-run 'OrgGate'` の profile で 39/40 文(97.5%)、通らないのは `cmd.Context()` が nil のときの 1 行だけ
- Function: 100% は `gateScriptEnv`、`gateEnvValue`、`details`、`gateRecordFromDetails`、`gateDirtyRefusal`、`gateHead`、`stopGateStrays`、`stopGateGroup`、`push`、`leaderActivityEventCount`、`printGateResult` など。100% でない関数の通らない区間は、ほとんどが git の呼び出し、manifest の読み書き、ログの書き込みの失敗を返す枝だった。そのほかに通らないもの:
  - `Gate` 77.5%: `Env` が nil のときの `os.Environ()`、`Dir` が空のときの `os.Getwd()`(`gate.go:248-257`)。CLI は `Env` を渡し `Dir` を空にするので、CLI のテストでは通る。org 層のテストはどちらも渡す
  - `runGateScript` 91.7%: スクリプトが起動できないとき(`cmd.ProcessState == nil`、`gate_run.go:205-208`)。下の probe で動きを確かめた
  - `gateOutput.Write` 78.6% と `tail` 66.7%: 改行で終わらない書き込み(`gate_run.go:262-264`)、改行のない最後の行を tail に足す枝(`:287-289`)、20 行と途中の 1 行で 21 行になったときに切る枝(`:290-292`)。コミットしたテストのスクリプトの出力は、どれも改行で終わる。下の probe で動きを確かめた
  - `recordGate` 81.8%: 走らせている間に、disband ではなく結びつきが不完全になった場合(`gate.go:723-724`)。disband、立て直し、別の機能への結びつきは `TestGate_OrgChangedWhileTheScriptsRan` が通す
  - `gateOrgLife` 93.3%: 結びつきがあって走っていない場合(`gate.go:354-357`)。コメントのとおり、今の `RunningOrgs` では起きない守り
- Branch: Go の標準ツールには branch coverage がないので測っていない。shell のスクリプトには計測の道具がない

## Test plan との対応

| Test plan の項目 | 確かめたテスト | 結果 |
| --- | --- | --- |
| Unit: base の決め方(origin/HEAD あり、なしで main、master だけ、どれもない、origin なしで手元のブランチ) | `TestGate_BaseResolution` の 9 subtest(origin/HEAD が trunk、同名のローカルブランチ `origin/trunk` があるとき、origin/HEAD の先がない、origin に main と master、master だけ、手元の master だけ、手元の main だけ、origin なし、どれもない)、`TestGate_RefusedBeforeTheScripts` の「no default branch」の 2 subtest | PASS |
| Unit: 祖先の検査 | `TestGate_RefusedBeforeTheScripts` の `HEAD_does_not_contain_origin/main`、`..._local_main,_with_no_origin` | PASS |
| Unit: 作業ツリーの検査(未追跡、変更、ignore されたファイル) | 同じテストの untracked、modified、staged、many untracked の 4 subtest、`TestGate_PassesFromASubdirectoryAndWithIgnoredFiles` | PASS |
| Unit: 結びつきの検査(なし、不完全、disband 済み、別の worktree、サブディレクトリ、別のブランチ、detached HEAD) | 同じテストの 12 subtest(無効な org_id、台帳にない org、`--plan` なしで始めた org、不完全な結びつき、disband 済み、消えた worktree、main のチェックアウト、git の外、別のブランチ、detached HEAD、衝突で止まった rebase の 2 通り)、`TestOrgGate_RefusalPrintsNothingAndRecordsNothing/another_git_worktree`、サブディレクトリは上の `PassesFromASubdirectory...` | PASS |
| Unit: スクリプトの有無と実行の可否 | 同じテストの 4 subtest(static がない、test がない、static が実行できない、test がディレクトリ) | PASS |
| Unit: 2 つのスクリプトの成否と順 | `TestGate_ScriptFailuresAndChangesRecordGateFailed`(呼ばれた順 `calls` を比べる)、`TestGate_PassRecordsGatePassed` | PASS |
| Unit: 環境の固定(除く変数、足す変数) | `TestGateScriptEnv`、`TestGate_ScriptEnvironmentIsFixed`、`TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier`(出荷する `run-verify.sh` で、`RALPH_VERIFY_BASE=HEAD` などを与えても言語の検証と branch の secret scan が走る) | PASS |
| Unit: 走らせている間の HEAD の移動と作業ツリーの汚れ | `TestGate_ScriptFailuresAndChangesRecordGateFailed` の `HEAD_moves_during_the_scripts` と `a_script_leaves_a_file_in_the_worktree`、CLI の `TestOrgGate_FailurePrintsTheReasonAndExits1` の同名の 2 subtest | PASS |
| Unit: 記録の Details、ログの置き場所 | `TestGateRecordDetails_RoundTrip`、`TestGate_PassRecordsGatePassed`(`git status --ignored` まで空)、`TestGate_LogDirCannotBeMade` | PASS |
| Unit: ロックを持たないこと | `TestGate_ManifestLockNotHeldWhileScriptsRun` | PASS |
| Unit: `leaderActivityEventCount` | `TestLeaderActivityEventCount_GateEvents` | PASS |
| Unit: 計画を archive して戻したときの TASK の検査(AC13) | `TestOrgSend_ApprovalCheck_PlanArchivedAndRestored` | PASS |
| Shell: 重ねた 2 つの実行の終了コード、ファイルがないとき、`verify-scope` が残ること | `tests/test-run-verify-concurrency.sh` の Case 1〜6(37 件)。既存の `test-run-verify-scope.sh` ほか 5 本も PASS | PASS |
| Integration: CLI の `ralph org gate`(3 行の出力、終了コード、`--org-id` の必須、`--state-dir`) | `TestOrgGate_PassPrintsGateHeadBaseAndLog`、`TestOrgGate_FailurePrintsTheReasonAndExits1`、`TestOrgGate_PrintGateResultEdges`、`TestOrgGate_OrgIDRequiredAndNoArguments`、`TestOrgGate_StateDirPicksTheLedger`、`TestOrgGate_LegacyLedgerGuard`、`TestOrgGateHelp` | PASS |
| Integration: 機能の org の台帳を作り、worktree で打つ往復 | `TestOrgGate_StartPlanThenGateInTheFeatureWorktree` | PASS |
| Regression: org 層と CLI の既存のテスト、`TestRenderRolePrompt_*`、Roster と status | 既存のテストはすべて PASS(消えたテストは 0)。手順の番号を書き換えた `TestRenderRolePrompt_Leader_FeatureOrgProcedure` も PASS。gate の event が座席の状態を変えないことは `TestGateEvents_AreNotASeat`(Roster、`RunningOrgs`、`orgsToDisband`、report の Roster、`Status`) | PASS |
| Edge: 同じ head で pass のあと fail | `TestGate_PassThenFailOnTheSameHead` | PASS |
| Edge: gate の中断 | `TestGate_InterruptStopsTheScriptsAndRecordsNothing`(SIGTERM を無視するスクリプトを含む 2 subtest)、`TestGate_InterruptedBeforeTheScripts`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`(SIGINT、SIGTERM、SIGHUP、ctx の取り消し)、`TestOrgGate_OrgDisbandedWhileTheScriptsRanNamesTheLog`、`TestGate_StopsWhatAScriptLeftRunning` | PASS |
| Edge: ログのディレクトリがない | pass のテストはどれも `gates/` のない台帳から始まり、`MkdirAll` が作る。作れないときは `TestGate_LogDirCannotBeMade` | PASS |
| Edge: `HERDR_PANE_ID` のあり・なし | `TestGate_RecordsWhoRanIt` の 6 subtest(空の pane id、USER が 2 回、空白を含む USER、空の USER を含む) | PASS |
| Edge: 見張りのない台帳、未承認の per-plan の org | `TestGate_NoWatchGateNoApprovalCheck` | PASS |
| Evidence: `go test ./...`、`-race`(org と cli)、mutation、実機確認 | 上の表と下の Mutation。実機確認は /verify の probe(`docs/evidence/verify-2026-10-13-org-gate-probe.txt`)が HEAD c84d8e19 の binary で行い、そこから HEAD まで Go のファイルは変わっていないので、この回では打ち直していない | 済み |

## Mutation

どの mutant も、scratchpad に写したファイルの 1 か所を書き換え、tracked のファイルには触れていない。Go の mutant は `go test -count=1 -overlay <json> ./internal/org/ ./internal/cli/ -run 'Gate|LeaderActivityEventCount' -v` で流した(cli は org を import するので、overlay は CLI のテストにも効く)。10 件はどれもコンパイルできた(`go vet -overlay` の rc 0 をコンパイルの確かめにだけ使った)。shell の mutant は、`tests/test-run-verify-concurrency.sh` が `$PROJECT_ROOT/scripts/run-verify.sh` を写して使うことを利用し、書き換えた `scripts/run-verify.sh` とコミットしたままのテストを置いた scratch のディレクトリから打った。書き換えていない写し(X0)は 37 / 37 で通った。

依頼の 3 つの動き:

| ID | 何を壊したか | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| H1 | `gateCheckAfter` の HEAD の比較を外す(`if false && head != t.head`) | red | `TestGate_ScriptFailuresAndChangesRecordGateFailed/HEAD_moves_during_the_scripts`(`Passed:true` になった)、`TestOrgGate_FailurePrintsTheReasonAndExits1/HEAD_moves_during_the_scripts`(`GATE: pass` を出し、`reason=head_moved` の拒否を返さなかった) |
| H2 | 走らせたあとの検査 `gateCheckAfter` を呼ばない | red | H1 の 2 つと、同じ 2 本の `a_script_leaves_a_file_in_the_worktree`(tree_changed) |
| E1 | 環境の変数を何も除かない(`gateStrippedEnvPrefixes` を空に) | red | `TestGateScriptEnv`、`TestGate_ScriptEnvironmentIsFixed`、`TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier`(`roots=nowhere` が渡り、branch の secret scan が飛ばされた)、`TestOrgGate_PassPrintsGateHeadBaseAndLog` |
| E2 | `RALPH_VERIFY_BASE` だけを除かずに通す | red | `TestGateScriptEnv`、`TestGate_ScriptEnvironmentIsFixed`(`[RALPH_VERIFY_SCOPE=full RALPH_VERIFY_BASE=HEAD]`)、`TestOrgGate_PassPrintsGateHeadBaseAndLog` |
| E3 | 除くが `RALPH_VERIFY_SCOPE=full` を足さない | red | E1 と同じ 4 本。本物の `run-verify.sh` のテストでは、範囲が `run-static-verify.sh` と `run-test.sh` の既定の `changed` になり、fixture の HEAD は base と同じなので差分がなく、落ちる Go の検証が走らないまま gate が pass した |
| E4 | `RALPH_VERIFY_` だけを除き、`HARNESS_VERIFY_` を通す | red | `TestGateScriptEnv`、`TestGate_ScriptEnvironmentIsFixed` |
| X1 | `run-verify.sh` が終了コードを読めないとき 0 にする(`*)` の枝の `verify_status=1` を 0 に) | red | Case 1 `killed block makes run-verify.sh fail (exit 0, want non-zero)` |
| X2 | 空の終了コードを 0 とみなす(変更前の「ファイルがなければ成功」と同じ意味。case の前に `verify_status="${verify_status:-0}"`) | red | Case 1 の 2 件(終了コード 0 になり、読めなかったという文も出ない) |
| X3 | 終了コードの受け渡しを共有のパス `.harness/state/verify-exit-code` に戻す | red | Case 2 `run A (failing) exits 1 after run B passed (exit 0, want 1)`、Case 3・4 の共有のファイルを書かないことの 2 件 |

AC12 のほかの 2 つと、計画の Evidence の「事前の検査の条件、記録の値」から 4 つを足した:

| ID | 何を壊したか | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| X4 | `.harness/state/verify-scope` を最後に置かない | red | Case 3・4 の `... leaves .harness/state/verify-scope with the scope lines` の 2 件 |
| X5 | EXIT trap の後片付けを外す(`trap : EXIT`) | red | `... leaves no per-run files` の 10 件(Case 1〜6) |
| A1 | 祖先の検査で、祖先でない(`merge-base --is-ancestor` の exit 1)も通す | red | `TestGate_RefusedBeforeTheScripts` の祖先の 2 subtest、`TestOrgGate_RefusalPrintsNothingAndRecordsNothing/a_HEAD_that_does_not_contain_origin/main`、`TestOrgGate_StartPlanThenGateInTheFeatureWorktree` |
| D1 | 作業ツリーの検査を `--untracked-files=no` にする | red | untracked と many untracked の拒否、tree_changed(org と CLI) |
| R1 | 記録の `base=` に HEAD を書く | red | `TestGateRecordDetails_RoundTrip` と CLI の 3 本。org 層の gate を打つテストは落ちなかった(下の Test gaps) |
| R2 | `GateResult.Base` に HEAD を入れる | red(CLI だけ) | `TestOrgGate_PassPrintsGateHeadBaseAndLog`、`TestOrgGate_FailurePrintsTheReasonAndExits1` の 4 subtest、`TestOrgGate_StartPlanThenGateInTheFeatureWorktree`。org 層は 1 本も落ちなかった |

生き残りは 0 件。E2 では本物の `run-verify.sh` のテストは通った。`RALPH_VERIFY_SCOPE=full` のとき `run-verify.sh` は `RALPH_VERIFY_BASE` を読まない(読むのは `detect-changed-languages.sh` だけで、changed の範囲のときにしか呼ばれない)ので、E2 は出荷するスクリプトでの動きを変えない。環境をそのまま比べる 2 本の unit test が落とす。

### probe(コミットしていない)

coverage で通らなかった区間のうち、利用者に見える 2 つを、scratch の `_test.go` を `-overlay` で足して HEAD のコードで確かめた。どちらも PASS で、動きはコメントのとおりだった。全文は mutation の evidence にある。

- `TestProbe_GateOutputPartialAndLongLines`: 3 回に分けて書いた 1 行、CRLF の行、改行のない最後の行が tail に入る。4,596 バイトの行は 2 回に分けても改行なしでも 4,096 バイトで切られる。25 行と途中の 1 行のあとの tail は `line 7` から途中の行までの 20 行
- `TestProbe_GateScriptThatCannotStart`: 実行の許可はあるが shebang の interpreter がない static のスクリプトは、`static_failed` になり、test のスクリプトは走らず、`gate_failed` を 1 行書き、tail とログに `ralph org gate: ./scripts/run-static-verify.sh could not run: fork/exec <script のパス>: no such file or directory` と `exited -1` が出る。この文はスクリプトのパスを挙げるので、ない interpreter(たとえば CRLF の shebang)を探す手がかりにはならない。不具合ではなく、文の改善の余地として記録する

## Flakiness

- `tests/test-run-verify-concurrency.sh` を 5 回続けて打った(JST 17:01〜17:02、load average 4.0〜4.6): 5 回とも 37 / 37、rc 0、1 回 13.8〜14.2 s
- Case 6 の余裕を測るため、`poll_for_file` が 0.05 s の poll を何回したかを出すように写しを直し(差分は 2 行、flaky の evidence にある)、24 本の `yes > /dev/null` で CPU を埋めて 3 回打った(load average は 28.8、63.4、96.6 と上がった): 3 回とも 37 / 37、rc 0、1 回 19.4〜20.7 s。グループへの TERM は、上限 40 回(約 2 s)のところ 1〜2 回の poll で終わった。pid だけへの TERM は、20 回(約 1 s)の poll のあいだ終わらず、assertion の向きどおりだった。この assertion は負荷で遅くなるほど通りやすい。stub の verifier の待ちの上限は 200 回(約 10 s)で、開始を待つ poll は負荷の下でも 9〜15 回だったので、こちらも余裕がある
- `go test ./internal/org/ ./internal/cli/ -run 'Gate' -count=3 -v`(JST 17:07〜17:09、開始時の load average 15.3): rc 0。名前に `Gate` を含む top-level の 61 本(この PR で足した 32 本と、6b の見張りの関門のテストなど既存の 29 本)× 3 回で 183 件、subtest 429 件がすべて PASS。同じテストの 3 回の時間の幅が大きいのは `TestGate_StopsWhatAScriptLeftRunning`(3.3〜7.0 s)と `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`(2.0〜5.7 s)で、どちらも待ちの上限には届いていない
- mutation の 10 件(上)も `-run 'Gate|...'` を流したもので、mutant と関係のないテストはどの回でも落ちなかった

揺れは見つからなかった。混んだ Linux の CI の runner での Case 6 は、この回では確かめていない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

テストの失敗はなかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 変更前の `run-verify.sh` では、終了コードのファイルがないと成功になり、重なった実行の片方の失敗が消えた(B10、Codex plan advisory の MEDIUM) | 直っている | Case 1・2 が PASS。変更前の意味に戻す X2 と、共有のパスに戻す X3 は red |
| 打った側の `RALPH_VERIFY_BASE=HEAD` で言語の検証が飛ぶ(B9、Codex plan advisory の HIGH) | 直っている | `TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier` が PASS。対照の実行は言語の pack を選ばないことを先に確かめる。E3 で red |
| archive したあとに gate が落ちると per-plan の TASK が断られて戻れない(B11、Codex plan advisory の HIGH) | 戻し方が効く | `TestOrgSend_ApprovalCheck_PlanArchivedAndRestored` が PASS |
| 既存の org 層と CLI のテスト(AC9) | 通る | 消えたテストは 0、すべて PASS。`-race` も 0 fail |

## Test gaps

- org 層の gate のテストの fixture は、feature のブランチに main より先のコミットがなく、HEAD と base の SHA が同じになる。そのため org 層の gate を打つテストは、`GateResult.Base` と記録の `base=` が HEAD と入れ替わっても気づかない(R2 は org 層では生き残り、R1 は `details()` を直接見る `TestGateRecordDetails_RoundTrip` だけが落とした)。CLI のテストは HEAD と base が違う fixture で両方を落とすので、合わせれば閉じている。org 層で閉じるなら、`TestGate_PassRecordsGatePassed` などで feature のブランチにコミットを 1 つ足してから打つ(`gateCommit` で足せる)
- `gateOutput` の改行のない出力の枝(tail の途中の行、4,096 バイトの上限)と、起動できないスクリプトの枝は、コミットしたテストが通らない。probe では期待どおりに動いた。テストを足すなら probe の 2 本がそのまま使える
- `recordGate` の「走らせている間に結びつきが不完全になった」枝と、`gateOrgLife` の「結びつきがあって走っていない」枝はテストがない(後者は今のコードでは起きない守り)
- テンプレート側の `templates/base/scripts/run-verify.sh` はテストが直接は打たない。root の写しとバイト単位で同じことは /verify の `cmp` と `check-sync.sh` が確かめている
- herdr の pane の中の本物の leader(claude と codex)が雛形どおりに gate をバックグラウンドで打ち、出力のファイルを読めるか、codex の sandbox で fetch とスクリプトが動くか、leader のツールの timeout が送る信号は、この回でも確かめていない(/verify の Coverage gaps と同じ)
- Linux(dash が `/bin/sh` の環境)と混んだ CI の runner での `tests/test-run-verify-concurrency.sh` は、この回では確かめていない。macOS では `/bin/sh`(bash 3.2)と dash の 2 つで通った
- mutation は依頼の 3 つの動きと AC12・事前の検査・記録の値に絞った 15 件で、ロックを持たないこと(AC5)、中断、見張りの数え方(AC6)、雛形の語句(AC8)の mutant は流していない

## Verdict

- Verdict: pass(ブランチ全体、cycle 4 の HEAD 1e4a2fa4 まで。下の Pass・Fail・Blocked の 3 行は cycle 1 の HEAD eb5ffdac での結果で、cycle 2〜4 の結果は、下の「Cycle 2」「Cycle 3」「Cycle 4」の各節の「判断(cycle N)」にある。cycle 3 と cycle 4 は go1.25.8 と go1.26.0 の両方で走らせた。cycle 4 で残った穴は、skill の 4 面の締めの順を読むテストがないこと(mutant E が生き残り)で、テストの失敗はない)
- Pass: `./scripts/run-test.sh` は changed と full のどちらも rc 0(shell 41 本、PASS の行 4,118、Go 8 パッケージ)。`go test ./... -count=1` は top-level 1,538 本が PASS で 0 fail、足した 34 本もすべて PASS で、消えたテストはない。`-race`(org と cli)も 0 fail で、`DATA RACE` は 0 件。直接打った shell のテスト 6 本は 176 件すべて PASS。concurrency のテストは 5 回連続と負荷の下の 3 回で揺れず、gate のテストの `-count=3` も 0 fail。mutation は依頼の 3 つの動き(head_moved、環境の固定、終了コードのファイルがないとき)の 9 件を含む 15 件がすべて red
- Fail: 0
- Blocked: なし

## Cycle 2

- Date: 2026-10-13(JST 20:39〜21:25)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 の最後)。cross-review cycle 1 の ACTION_REQUIRED #1 を直した b04289c1、self-review cycle 2(bedc0ebc)、verify cycle 2(0bc87603、pass)のあとに走らせた
- Scope: HEAD 0bc87603。cycle 1 の HEAD eb5ffdac から変わったコードとテストは b04289c1 の 3 ファイル(`internal/org/gate.go`、`internal/org/lockfile.go`、`internal/org/gate_test.go`)だけで、ほかは sync-docs の 6716a028 が直した文書(AGENTS.md、README.md、quality-gates の 2 面)、計画の Notes、各段の報告と evidence、insight event。behavioral test だけを実行した(静的解析は /verify の担当)。この節の行番号はすべて HEAD 0bc87603 のもの
- Evidence(上書きしたものと、末尾に足したもの):
  - `docs/evidence/test-2026-10-13-org-gate-run-test.txt`: 上書きした。`./scripts/run-test.sh`(changed)1 回の出力で、末尾に `RC=` がある。cycle 1 の 2 回分の出力は `git show 0bc87603:docs/evidence/test-2026-10-13-org-gate-run-test.txt` で読める
  - `docs/evidence/test-2026-10-13-org-gate-go-test.txt`: 上書きした。`go test ./internal/org/... ./internal/cli/... -count=1 -timeout 30m -v` と、末尾に `internal/insights` の打ち直しと `internal/org` の coverage の要約
  - `docs/evidence/test-2026-10-13-org-gate-race.txt`: 上書きした。`go test -race ./internal/org/ ./internal/cli/ -run 'Gate|Lock' -count=3 -timeout 30m -v`
  - `docs/evidence/test-2026-10-13-org-gate-flaky.txt`: 末尾に「Cycle 2」を足した。テストごとの回数と時間の幅、遅かった 5 つの subtest、fork の probe、生の出力
  - `docs/evidence/test-2026-10-13-org-gate-mutation.txt`: 末尾に「Cycle 2」を足した。mutant の overlay と差分、結果、落としたテストと最初の失敗の行、probe のテストの全文と結果
  - `$HOME` は `~` に、scratchpad のパスは `<scratchpad>` に置き換えた。置き換えが要ったのは go-test の evidence の 4 行(doctor のテストが出す `~/.agents/skills/agmsg` と `~/.codex/models_cache.json`)と、mutation の evidence の overlay の JSON だけだった。どの evidence にも U+FFFD はない
  - wrapper が書いた `docs/evidence/verify-2026-10-13-113922.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-gate.jsonl` に test の 1 行(`--cycle auto` で cycle 2、verdict pass)を足し、この報告と同じコミットに入れた。行は `./scripts/insights-append.sh --events-dir <scratchpad>` で作り、Edit で足した
- 依頼の 2 つの `go test` には `-timeout 30m` を足した。混んだときに既定の 10 分で package が止まらないようにするためで、テストの選び方と回数は依頼のとおり
- テストもコードも変えていない。mutant の写し、probe のテスト、fork の probe は scratchpad の `og2/` に置いた。テストの前後で `~/.config/herdr` の下に新しいファイルはなく、main のチェックアウトの `git status --short` は空のまま、worktree で変わったのはこの報告、上の evidence、insight event だけで、`$TMPDIR` に `ralph-verify-*` の一時ファイルは残っていない。osascript を呼んだかどうかは、この回では確かめていない(テストのコードは b04289c1 の 2 本のほかは cycle 1 と同じ)

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。cycle 1 と同じく `shared:scripts/run-verify.sh` で full に落ち、golang の pack を選んだ) | shell 41 本(PASS の行 4,118)、Go 8 パッケージ | すべて | 0 | shell 1 | 13 分 54 秒、rc 0 |
| `go test ./internal/org/... ./internal/cli/... -count=1 -timeout 30m -v` | top-level 1,324、subtest 1,953 | top-level 1,323、subtest 1,953 | 0 | 1 | 3 分 15 秒(cli 194.1 s、org 99.3 s)、rc 0 |
| `go test -race ./internal/org/ ./internal/cli/ -run 'Gate\|Lock' -count=3 -timeout 30m -v` | top-level 83 × 3 = 249、subtest 489 | すべて | 0 | 0 | 2 分 56 秒(org 173.9 s、cli 102.0 s)、rc 0。`DATA RACE` は 0 件 |
| 揺れの確認(下の「Flakiness(cycle 2)」) | 8 本 × 20 回、8 本 × 5 回(`-race`) | すべて | 0 | 0 | 4 分 3 秒 |
| mutation(Go 9 件) | 9 件 | red 7 件 | 生き残り 2 件(下の表) | - | 1 件 38〜50 s、並列 2 |

- shell の PASS の行は cycle 1 と同じ数え方で 4,118 行。cycle 1 の changed の実行と比べると、違うのは `tests/test-pre-bash-guard.sh` の H の 3 行の経過時間の数字だけだった。Skipped は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0)
- `run-test.sh` の中の Go は `internal/cli`(159.7 s)と `internal/org`(68.6 s)を流し直し、ほかの 6 パッケージは `(cached)` だった。6 パッケージのうち `internal/insights` は org を import する。`(cached)` は org の中身まで同じときの結果しか使わないので、b04289c1 のあとに流した結果が残っていたことになる。キャッシュに頼らないよう `go test -count=1 ./internal/insights/` を打ち直し、rc 0(0.9 s)だった。残りの 5 パッケージは org を import しない(`go list -deps`)
- 時間が cycle 1(9 分 23 秒)より長いのは、おそらくほかの作業の負荷のため。load average は開始時に 5.93 で、終わる 20:53 には 145.73 まで上がり、`go test` の開始時も 81.88 だった。その中で前に flake を起こした doctor のテストと `TestRunWatcher_TimeoutIndependentOfSmallInterval` は落ちなかった
- `go test` の件数は、同じ 4 パッケージの cycle 1(cli 615、org 643 と skip 1、driver 48、protocol 15、subtest 1,949)より org の top-level が 2、subtest が 4 増えた。増えたのは `TestGate_InterruptAfterTheScriptsRecordsNothing` と `TestRecordGate_InterruptRecordsNothing`(subtest 4 本)で、ほかのパッケージと org のほかのテストの数は変わらない。Skipped は cycle 1 と同じ `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`
- `-race` の `-run 'Gate|Lock'` に入る top-level は 83 本(org と cli)で、3 回とも 0 fail だった

### Regression: `withManifestLock` と `withFileLock` の呼び元(cycle 2)

b04289c1 で `withManifestLock` と `withFileLock` は `context.Background()` を渡して `withFileLockContext` を呼ぶ形になった。呼び元のテストは上の `go test` の実行でどれも PASS だった。

| 呼び元 | 名前を挙げるテスト(すべて PASS) |
| --- | --- |
| `spawn.go:621`、`:796`(spawn と予約) | `TestWithManifestLock_SerializesConcurrentCallers`、`TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded`、`..._MaxOrgsNeverExceeded`、`..._MaxTotalSeatsNeverExceeded`、`TestOrgSpawn_ConcurrentReservations_OnlyOneOfOverlappingWins`。`TestOrgSpawn*` は 150 本すべて PASS |
| `verbs.go:1361`(`compensateUnderLock`。`CloseDeferredSelfPane` と `CloseDeferredSelfWorkspace` が呼ぶ) | `TestOrgCloseDeferredSelf_NewerRecordsWhileClosing_Stay`(manifest のロックを持ったまま close を待たせる) |
| `inbox.go:327`(inbox のロック) | `TestInboxLock_IsSeparateFromTheManifestLockAndSerializesWrites`、`TestInboxStore_ReportsAnUnreadableInboxAndAnUnavailableLock`、`TestInboxConcurrentAcks_OnlyOneChangesTheItem`、`TestInboxConcurrentAckAndResolve_LeaveAValidState`、`TestInboxEscalate_ConcurrentCallsGetDistinctIDs`、`TestEscalate_ConcurrentEscalatesGetDistinctIDs`。`TestInbox*` 23 本、`TestEscalate*` 10 本 |
| `approve.go:147` | `TestOrgApprove*` 24 本(`TestOrgApprove_AllOrNothing`、`..._AppendFailureRecordsNothing` を含む) |
| watch のロック(`acquireHeldFileLock`。b04289c1 は変えていない) | `TestRunWatchAll_*Lock*` 6 本、`TestAcquireWatchLock_*` 2 本、`TestProbeWatch_*` 2 本、CLI の `TestOrgWatch*Lock*` 6 本 |
| gate(manifest のロックを持たないこと) | `TestGate_ManifestLockNotHeldWhileScriptsRun` |

send(`verbs.go:316`)は manifest のロックを取らない。`TestOrgSend*` の 53 本も PASS だった。

ロックが 5 秒より長く持たれたときの文と待ちの長さは、コミットしたテストがどれも通らない(`lockfile.go:97` の文は coverage 0)。scratch の `_test.go` を `-overlay` で足した probe `TestProbe_LockTimeoutMessageOfExistingCallers` で HEAD のコードを確かめた。`withManifestLock` と inbox の `withFileLock` は、どちらも 5.00 秒待ち、fn を走らせず、base 1bddc871 の `withFileLock` と同じ文 `org: acquire manifest lock <path>: timed out after 5s: context deadline exceeded`(inbox は `inbox lock`)を返した(rc 0、10.8 s)。下の C2h を入れると、同じ probe は 2 つとも `the wait was stopped: %!w(<nil>)` を返して red になったので、この probe は文の変化を見分ける。

### Flakiness(cycle 2)

依頼の 2 本と、gate の中断・ロックの既存のテストを `-count=20`(org、続けて cli)と `-race -count=5`(両方)で流した(JST 21:01〜21:06、開始時の load average 3.0〜4.6、この session のほかの実行はなし)。どの回も rc 0 で、`DATA RACE` は 0 件。

| テスト | `-count=20` | 時間の幅 | `-race -count=5` | 時間の幅 |
| --- | --- | --- | --- | --- |
| `TestGate_InterruptAfterTheScriptsRecordsNothing`(新規) | 20 / 20 | 1.15〜1.44 s | 5 / 5 | 1.15〜1.24 s |
| `TestRecordGate_InterruptRecordsNothing`(新規、subtest 4 本はどれも 20 / 20 と 5 / 5) | 20 / 20 | 1.23〜1.81 s(ロックの待ちの subtest は 0.45〜0.65 s) | 5 / 5 | 1.29〜1.35 s |
| `TestGate_InterruptStopsTheScriptsAndRecordsNothing` | 20 / 20 | 1.60〜1.88 s | 5 / 5 | 1.64〜2.22 s |
| `TestGate_InterruptedBeforeTheScripts` | 20 / 20 | 0.22〜0.28 s | 5 / 5 | 0.22〜0.28 s |
| `TestGate_ManifestLockNotHeldWhileScriptsRun` | 20 / 20 | 1.02〜1.35 s | 5 / 5 | 1.09〜1.35 s |
| `TestWithManifestLock_SerializesConcurrentCallers` | 20 / 20 | 0.48〜0.52 s | 5 / 5 | 0.48〜0.51 s |
| `TestInboxLock_IsSeparateFromTheManifestLockAndSerializesWrites` | 20 / 20 | 0.16〜0.17 s | 5 / 5 | 0.16〜0.17 s |
| `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`(CLI、subtest 4 本はどれも 20 / 20 と 5 / 5) | 20 / 20 | 2.60〜13.50 s | 5 / 5 | 2.77〜3.43 s |

- 新しい 2 本の「中断から 3 秒以内に返る」という assertion は、テスト全体の時間でも 1.81 秒以下なので、3 秒の上限まで 1.19 秒以上ある。load average 81.88 の中で走った `go test` の回でも、2 本は 1.29 s と 1.32 s で通った
- CLI のテストの subtest 80 回(信号 3 種と ctx の取り消しを 20 回ずつ)のうち 5 回(SIGINT 2 回、SIGHUP 3 回)が 5.68〜5.71 s かかり、ほかは 0.63〜1.11 s だった。差の 5 秒は `gateKillGrace`(`internal/org/gate_run.go:54`)と同じで、スクリプトのグループに SIGTERM を送ったあと、グループに残ったプロセスが SIGKILL まで消えなかったときの長さに当たる。この道は b04289c1 が変えていない `runGateScript` の中で、中断がスクリプトの実行中に届く場合の動き。テストはどの回も通り(上限 15 秒、終わったあとのグループの検査も通った)、記録は残らなかった。cycle 1 の evidence にある実行には、5 秒台の subtest は 1 つもない
- その原因として、スクリプトの `sleep` を fork している最中にグループへの SIGTERM が届くと、作られた子がその SIGTERM を受けずに残る場合を考え、同じ形のスクリプトに `started` ができた直後にグループへ SIGTERM を送る probe(`og2/forkrace/race.py`)を作った。最初の 150 回では 7 回、グループにプロセスが残った(1 ms か 3 ms 待ってから送った 150 回ずつは 0 回)。その後の 900 回(load average 3.3〜3.4)では 0 回だった。おそらく fork の途中に届いた信号の取りこぼしで、混み具合によって起きる。未確認です。この推測が正しければ、利用者がスクリプトの実行中に Ctrl-C を押したとき、まれに gate が返るまで 5 秒ほど余分にかかる

揺れによる失敗は見つからなかった。

### Mutation(cycle 2)

どの mutant も、scratchpad に写したファイルの 1 か所を書き換え、tracked のファイルには触れていない。`go build -overlay <json> ./internal/org/` で 9 件ともコンパイルできることを確かめてから、`go test -count=1 -overlay <json> ./internal/org/ ./internal/cli/ -run 'Gate|Lock' -v` で流した(1 回 83 本、JST 21:14〜21:17、並列 2、load average 3.4〜6.9)。

依頼の 5 件:

| ID | 何を壊したか | 結果 | 落としたテストと失敗の行 |
| --- | --- | --- | --- |
| C2a | (a) `recordGate` のロックを取る前の ctx の検査を外す(`gate.go:720` を `if false && ...`) | red | `TestRecordGate_InterruptRecordsNothing/ctx_done_before_the_lock`。ロックの中の検査が追記を止めるので manifest は変わらず、`gate_test.go:974` の「時計を 1 回読んだ(ロックの中に入った)」だけが落とした |
| C2b | (b) ロックの中、追記の直前の検査を外す(`gate.go:747`) | red | `.../ctx_done_under_the_lock,_right_before_the_append`(`gate_test.go:1023`、`recordGate = "", <nil>` で記録した) |
| C2c | (c) ロックの待ちの timeout を `context.Background()` から作る(`lockfile.go:90`) | red | `TestGate_InterruptAfterTheScriptsRecordsNothing`(`:919`、中断から 4.72 s)、`.../ctx_done_while_waiting_for_the_lock`(`:1006`、4.80 s)。待ちは 5 秒の timeout で終わり、文は「the wait was stopped」のまま、manifest も変わらないので、落としたのは 3 秒の時間の assertion だけ |
| C2d | (d) `Gate` が `recordGate` の error を `gateInterrupted` を通さずに返す(`gate.go:302`) | red | `TestGate_InterruptAfterTheScriptsRecordsNothing`(`:926`、ロックの文 `... the wait was stopped: context canceled` がそのまま返った) |
| C2e | (e) `withFileLockContext` の「the wait was stopped」の枝を外す(`lockfile.go:94`) | red | `.../ctx_done_while_waiting_for_the_lock`(`:1011`、`timed out after 5s: context canceled`) |

足した 4 件(順序、呼び出しの場所、既存の呼び元への影響):

| ID | 何を壊したか | 結果 | 落としたテストと失敗の行 |
| --- | --- | --- | --- |
| C2f | `recordGate` の最後の switch から中断の case を外す(中断の error にも「run the gate again」が付く、`gate.go:756`) | 生き残り(同値) | なし。`recordGate` の呼び元は `gate.go:301` の 1 か所だけで、ctx が終わっていれば `gateInterrupted` が error の文を使わずに中断の文を作るので、利用者に見える出力は変わらない |
| C2g | ロックの中の検査を追記のあとに動かす | red | `.../ctx_done_under_the_lock,_right_before_the_append`(`:1023`、manifest が変わった) |
| C2h | 「the wait was stopped」の枝の条件を `waitCtx.Err()` にする(本当の 5 秒の timeout も「the wait was stopped: %!w(<nil>)」と言う、`lockfile.go:94`) | 生き残り(コミットしたテスト)、probe で red | コミットしたテストはなし。上の Regression の probe が 2 つとも落とした |
| C2i | `withManifestLockContext` が `withFileLockContext` に `context.Background()` を渡す(`lockfile.go:66`) | red | C2c と同じ 2 本(`:919` 4.73 s、`:1006` 4.80 s)と、`:1009`(error が `context.Canceled` でない) |

CLI のテストは 9 件のどれでも落ちなかった。CLI の中断のテストはスクリプトの実行中に信号を送るので、スクリプトのあとの中断は org 層のテストだけが扱う。HEAD の binary でロックの待ちに SIGTERM と SIGHUP を送る確認は、/verify cycle 2 の probe(`docs/evidence/verify-2026-10-13-org-gate-probe.txt` の「Cycle 2: interrupt probe」)が行っている。

### Coverage(cycle 2)

`go test -count=1 -coverprofile ./internal/org/`(rc 0、85.6 s)で `internal/org` は 93.6%(cycle 1 は 93.5%)。`recordGate` は 81.8% から 85.2%、`Gate` は 77.5% から 80.0% に上がり、`withManifestLockContext` は 100%、`withFileLockContext` は 92.9%。通らない区間は、`withFileLockContext` では本当の timeout の文(`lockfile.go:97`、C2h が生き残った所)、`recordGate` では manifest の読み出しと追記の失敗(`gate.go:726-728`、`:750-752`)、中断でない error の文(`:759-760`)、cycle 1 からの結びつきが不完全になった枝(`:734-735`)、`Gate` では cycle 1 からの `Env` と `Dir` の既定(`:250-259`)、manifest の読み出しの失敗(`:263-265`)、ログを閉じるときの失敗(`:284-286`)と、スクリプトのあとの worktree の読み直しが ctx で切れた枝(`:292-294`)。

### Test gaps(cycle 2)

- 本当のロックの timeout(5 秒)の文と待ちの長さを固定するテストがない。b04289c1 は「withManifestLock と withFileLock は動きを変えない」と書くが、`withFileLockContext` の枝の条件を変える C2h はコミットしたテストでは生き残った。probe は HEAD が base と同じ文を返すことを確かめた。閉じるなら probe の 2 本(1 本 5 秒)をそのまま足すか、`manifestLockTimeout` をテストで縮められる変数にする
- スクリプトのあと、worktree の読み直し(`gateCheckAfter` の git)の最中に届いた中断は、どのテストも狙っていない(`gate.go:292-294` は coverage 0)。/verify cycle 2 の Coverage gaps と同じ
- CLI の層で、スクリプトのあとの中断を通すテストはない(上の Mutation)。HEAD の binary では /verify の probe が確かめた
- 中断がスクリプトの実行中に届いたとき、まれにグループのプロセスが SIGTERM で消えず、`gateKillGrace` の 5 秒のあとの SIGKILL まで待つ(上の Flakiness)。原因は推測で、確かめていない
- cycle 1 の Test gaps(org 層の fixture で HEAD と base が同じこと、`gateOutput` の改行のない出力、Linux と混んだ CI での `tests/test-run-verify-concurrency.sh`、本物の leader の座席)は、b04289c1 が関係しないので、そのまま残る

### 判断(cycle 2)

- Pass: `./scripts/run-test.sh`(changed、full に落ちた)は rc 0 で、shell 41 本、PASS の行 4,118、Go 8 パッケージ。`go test ./internal/org/... ./internal/cli/...` は top-level 1,323 本と subtest 1,953 本が PASS で 0 fail、足した 2 本(subtest 4 本)も PASS で、消えたテストはない。`-race -run 'Gate|Lock' -count=3` は 249 回すべて PASS で `DATA RACE` は 0 件。新しい 2 本と既存の中断・ロックのテストは `-count=20` と `-race -count=5` で 0 fail。mutation は依頼の 5 件(ロックの前の検査、ロックの中の検査、ロックの待ちの ctx、`gateInterrupted` を通すこと、「the wait was stopped」の枝)がすべて red で、足した 4 件のうち 2 件も red。既存のロックの呼び元のテストはすべて PASS
- 生き残り: C2f は利用者に見える出力を変えない同値の mutant。C2h はテストの穴で、probe では HEAD の動きが base と同じことを確かめた
- Fail: 0
- Blocked: なし
- 判断: pass(cycle 2)

## Cycle 3

- Date: 2026-10-13(JST 23:03〜23:50)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 3 回目。PR #225 の CI で `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` の SIGINT・SIGTERM・SIGHUP の 3 subtest が落ちたので、上限を 3 に上げて回した。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま(self-review と verify の cycle 3 と同じ)。直しの 8b681dc3、self-review cycle 3(a538c899)、verify cycle 3(697e1577、pass)のあとに走らせた
- Scope: HEAD 697e1577。cycle 2 の HEAD 0bc87603 から変わったコードとテストは、8b681dc3 の 2 ファイル(`internal/cli/org_gate.go` と `internal/cli/org_gate_test.go`、+122/-7)だけだった(`git diff --stat 0bc87603..697e1577 -- '*.go' scripts tests templates go.mod go.sum .github`)。ほかは各段の報告、evidence、walkthrough、計画の archive。behavioral test だけを実行した(静的解析は /verify の担当)。この節の行番号はすべて HEAD 697e1577 のもの
- Toolchain: 8b681dc3 の効果は Go 1.25 でしか見えないので、Go のテストを go1.25.8(`GOTOOLCHAIN=go1.25.8`。`go.mod:3` の版で、release の build もこれを使う)と go1.26.0(手元の既定)の 2 つで走らせた。下の表と evidence の各 block に、どちらで走らせたかを書いた。`./scripts/run-test.sh` は手元の既定の go1.26.0 で走る
- Evidence(上書きしたものと、末尾に足したもの):
  - `docs/evidence/test-2026-10-13-org-gate-run-test.txt`: 上書きした。go1.26.0 での `./scripts/run-test.sh`(changed)1 回の出力で、先頭に toolchain、末尾に `RC=` がある
  - `docs/evidence/test-2026-10-13-org-gate-go-test.txt`: 上書きした。Block 1 は go1.25.8 の `go test ./... -count=1 -timeout 30m -v`、Block 2 は go1.26.0 の `go test ./internal/org/... ./internal/cli/... -count=1 -timeout 30m -v`、Block 3 は go1.25.8 の coverage。PASS の行を数えるため、依頼の 2 つのコマンドに `-v` を足した。テストの選び方と回数は依頼のとおり
  - `docs/evidence/test-2026-10-13-org-gate-race.txt`: 上書きした。go1.25.8 の `go test -race ./internal/cli/ ./internal/org/ -run 'Gate|Lock|SignalContext' -count=3 -timeout 30m -v`
  - `docs/evidence/test-2026-10-13-org-gate-flaky.txt`: 末尾に「Cycle 3」を足した。テストと subtest ごとの回数と時間の幅、生の出力
  - `docs/evidence/test-2026-10-13-org-gate-mutation.txt`: 末尾に「Cycle 3」を足した。mutant ごとの overlay の JSON、差分、toolchain、結果、落としたテストと失敗の行、修正前のコードの再現、probe のテストの全文と結果、`-count=10` の繰り返し
  - `$HOME` は `~` に、scratchpad のパスは `<scratchpad>` に置き換えた。置き換えが要ったのは、go-test の evidence の 8 行(doctor のテストが出す `~/.agents/skills/agmsg` と `~/.codex/models_cache.json`。Block 1 と 2 に 4 行ずつ)と Block 3 の profile のパス、mutation の evidence の overlay の JSON とコマンドの行だけだった。足した 2 つの evidence は、元の中身が新しい中身の先頭とバイト単位で一致する。どの evidence にも U+FFFD はない
  - wrapper が書いた `docs/evidence/verify-2026-10-13-140441.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-gate.jsonl` に test の 1 行(`--cycle 3`、verdict pass)を足し、この報告と同じコミットに入れた
- テストもコードも変えていない。mutant の写し、probe のテスト、修正前のファイルの写しは scratchpad の `og3/` に置いた。テストのあと、`~/.config/herdr` の下に 23:00 より新しいファイルはなく、main のチェックアウトと worktree の `git status --short` は(この報告を書く前に)空で、`$TMPDIR` に `ralph-verify-*` は残らず、テストのスクリプトの `sleep 30` も残っていない。osascript は、CLI の `TestMain` が通知を差し替え(`internal/cli/main_test.go:63`)、org で本物を呼びうる唯一のテスト `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` は darwin で skip する(go-test の evidence の SKIP)。プロセスとしては見張っていない
- mutant を絞って流した回(JST 23:06〜23:09)は、`run-test.sh` の shell の suite と重なった。`run-test.sh` の Go の段(cli が 139.5 s で、23:14:36 に終わった)とは重なっていない。ほかの実行は 1 つずつ順に走らせた

### Test execution(cycle 3)

| Suite / Command | Toolchain | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。cycle 1・2 と同じく `shared:scripts/run-verify.sh` で full に落ち、golang の pack を選んだ) | go1.26.0 | shell 41 本(PASS の行 4,118)、Go 8 パッケージ | すべて | 0 | shell 1 | 9 分 56 秒、rc 0 |
| `go test ./... -count=1 -timeout 30m -v` | go1.25.8 | top-level 1,545、subtest 2,005 | top-level 1,542、subtest 2,005 | 0 | 3 | 3 分 10 秒(cli 189.0 s、org 101.4 s)、rc 0 |
| `go test ./internal/org/... ./internal/cli/... -count=1 -timeout 30m -v` | go1.26.0 | top-level 1,326、subtest 1,953 | top-level 1,325、subtest 1,953 | 0 | 1 | 3 分 20 秒(cli 198.9 s、org 103.5 s)、rc 0 |
| `go test -race ./internal/cli/ ./internal/org/ -run 'Gate\|Lock\|SignalContext' -count=3 -timeout 30m -v` | go1.25.8 | top-level 85 × 3 = 255、subtest 489 | すべて | 0 | 0 | 2 分 56 秒(org 164.4 s、cli 94.4 s)、rc 0。`DATA RACE` は 0 件 |
| 揺れの確認(下の「Flakiness(cycle 3)」) | 両方 | 3 本 × 20 回を 2 つの toolchain で、3 本 × 5 回(`-race`、go1.25.8) | すべて | 0 | 0 | 3 分 3 秒 |
| 修正前のコードの再現(f001bd4f の `org_gate.go` と `org_gate_test.go` を overlay で読ませ、中断のテストだけ) | 両方 | 1 本(subtest 4 本) | go1.26.0 で 4 本、go1.25.8 で 1 本 | go1.25.8 で 3 本 | - | 3.3〜3.6 s |
| mutation(Go 14 件と対照 1 件) | go1.25.8(M_a と対照は go1.26.0 でも) | 14 件 | red 11 件 | 生き残り 3 件(下の表) | - | 絞った回は 1 件 4〜56 s、cli 全体は 1 件 177〜184 s |

- shell の PASS の行は cycle 1・2 と同じ数え方で 4,118 行。suite ごとの PASS・FAIL・SKIP の数は cycle 2 の実行と 41 本すべて同じだった(`diff` が空)。Skipped は前と同じ `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0)
- `run-test.sh` の中の Go は `internal/cli`(139.5 s)と `internal/org`(72.2 s)を流し直し、ほかの 6 パッケージは `(cached)` だった。8b681dc3 が変えたのは `internal/cli` だけで、6 パッケージはどれも `internal/cli` を import しない(`go list -deps`)。Block 1 は 8 パッケージすべてをキャッシュなしで流した
- go1.25.8 の全体の件数は、cycle 1(top-level 1,538、subtest 2,001)に cycle 2 の 2 本(subtest 4 本)と 8b681dc3 の 2 本を足した数とちょうど合う。go1.26.0 の org と cli は cycle 2(top-level 1,323、subtest 1,953)より top-level が 2 本増えた。cycle 2 の go-test の evidence とテスト名を `comm` で比べると、増えたのは `TestOrgGate_SignalContextParentCauseAndStop` と `TestOrgGate_SignalContextKeepsLaterSignalsCaught` だけで、消えたテストはない。Skipped は全体で cycle 1 と同じ 3 件、org と cli で cycle 2 と同じ 1 件
- `-race` の `-run` に入る top-level は 85 本で、cycle 2 の 83 本に新しい 2 本が加わった。85 本とも 3 回ずつ PASS した
- load average は、`run-test.sh` の開始時に 4.08、go1.25.8 の `go test` の開始時に 4.77、go1.26.0 で 5.24、`-race` で 4.21、揺れの確認で 4.19〜5.51 だった。前に flake を起こした doctor のテストと `TestRunWatcher_TimeoutIndependentOfSmallInterval` は、どの実行でも落ちなかった

### 2 つの toolchain での違い(cycle 3)

修正前のコード(f001bd4f の 2 ファイル)で `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` を走らせた結果:

- go1.25.8: SIGINT、SIGTERM、SIGHUP の 3 subtest が `org_gate_test.go:701` で落ちた(`gate = org: gate: interrupted (context canceled): ...`)。ctx の取り消しの subtest は通った。PR #225 の CI の失敗(verify cycle 3 が引く run 38401983445 の 5970〜5976 行)と、落ちた subtest も文も同じ
- go1.26.0: 4 subtest とも通った

HEAD では、2 つの toolchain のどちらでも 4 subtest と新しい 2 本が通った(上の表)。8b681dc3 で go1.25.8 の結果は赤から緑に変わり、go1.26.0 では前から緑だった。helper を `signal.NotifyContext` に戻す mutant M_a も同じで、go1.25.8 では中断のテストの 3 subtest と `TestOrgGate_SignalContextKeepsLaterSignalsCaught`(子が `cause: context canceled` を出した)が落ち、go1.26.0 では 3 本とも通った。self-review cycle 3 の C3-1(手元が 1.26 だと、元に戻しても緑のまま)は、この実行で確かめた。

CI の verify は、8b681dc3、a538c899、697e1577 の 3 つとも success だった(`gh run list --branch feat/org-gate`)。697e1577 の run 38410029660 のログには `ok github.com/yoshpy-dev/ralph/internal/cli 34.718s` がある。CI の Go の版はログに出ないので、確かめていない(verify cycle 3 と同じ)。

### Flakiness(cycle 3)

依頼の 3 本を、go1.25.8 と go1.26.0 で `-count=20` ずつ、go1.25.8 で `-race -count=5` 流した(JST 23:24〜23:27、開始時の load average 4.19〜5.51、この session のほかの実行はなし)。どの回も rc 0 で、`DATA RACE` は 0 件。

| テスト | go1.25.8 `-count=20` | 時間の幅 | go1.26.0 `-count=20` | 時間の幅 | go1.25.8 `-race -count=5` | 時間の幅 |
| --- | --- | --- | --- | --- | --- | --- |
| `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` | 20 / 20 | 2.61〜7.95 s | 20 / 20 | 2.70〜12.90 s | 5 / 5 | 2.79〜7.83 s |
| 上の subtest `SIGINT` | 20 / 20 | 0.65〜5.84 s | 20 / 20 | 0.66〜5.76 s | 5 / 5 | 0.71〜0.82 s |
| 上の subtest `SIGTERM` | 20 / 20 | 0.63〜0.78 s | 20 / 20 | 0.64〜1.02 s | 5 / 5 | 0.66〜0.71 s |
| 上の subtest `SIGHUP` | 20 / 20 | 0.64〜5.79 s | 20 / 20 | 0.66〜5.72 s | 5 / 5 | 0.66〜0.73 s |
| 上の subtest `the_command's_context_is_cancelled` | 20 / 20 | 0.65〜0.78 s | 20 / 20 | 0.65〜0.76 s | 5 / 5 | 0.69〜5.70 s |
| `TestOrgGate_SignalContextParentCauseAndStop`(新規) | 20 / 20 | 0.00 s | 20 / 20 | 0.00 s | 5 / 5 | 0.00 s |
| `TestOrgGate_SignalContextKeepsLaterSignalsCaught`(新規) | 20 / 20 | 0.32〜0.34 s | 20 / 20 | 0.32〜0.34 s | 5 / 5 | 1.34〜1.35 s |

- cycle 2 で見た 5.7 s 前後の subtest は、この回も出た。go1.25.8 で 80 回中 2 回(SIGINT 5.84 s、SIGHUP 5.79 s)、go1.26.0 で 80 回中 3 回(SIGINT 5.76 s、SIGHUP 5.72 s と 5.69 s)、`-race` で 20 回中 1 回(ctx の取り消し 5.70 s)。ほかの subtest は 0.63〜1.02 s だった。差の 5 秒は cycle 2 と同じく `gateKillGrace`(`internal/org/gate_run.go:54`)に当たる
- `-race` の 1 回は信号ではなく ctx の取り消しの subtest で起きた。ctx の取り消しでも、gate はスクリプトのグループを同じ道で止める。そのため 5 秒の遅れは ralph が信号をどう受けるかではなく、グループを止める側(`runGateScript`)で起きている。どちらの toolchain でも、8b681dc3 より前のコード(cycle 2 の 5/80)でも起きるので、8b681dc3 が持ち込んだものではない。原因は cycle 2 と同じく推測で、確かめていない
- どの subtest も上限の 15 秒までには届かなかった(最大 5.84 s)。新しい子プロセスのテストは、子の中の 100 ms と 200 ms の待ちに起動の分を足して 0.32〜0.34 s、`-race` で build した子は起動が遅く 1.34〜1.35 s。子の中の「5 秒以内に cancel される」の上限には届いていない

揺れによる失敗は見つからなかった。

### Mutation(cycle 3)

どの mutant も、scratchpad に写した `internal/cli/org_gate.go` の 1 か所を書き換え、tracked のファイルには触れていない。`GOTOOLCHAIN=go1.25.8 go build -overlay <json> ./internal/cli/` で 14 件ともコンパイルできることを確かめてから、go1.25.8 で `go test -count=1 -overlay <json> ./internal/cli/ -run '^TestOrgGate_(InterruptStopsTheScriptAndRecordsNothing|SignalContext)' -v` を 1 件ずつ流した(top-level 3 本、JST 23:06〜23:09)。書き換えていない写し X0 は、go1.25.8 でも go1.26.0 でも 3 本とも通った。この回で生き残った 3 件は、cli パッケージ全体(top-level 617 本)を go1.25.8 で流し直した。

依頼の 6 つ:

| ID | 何を壊したか | 結果 | 落としたテストと失敗の行 |
| --- | --- | --- | --- |
| M_a | (a) helper の中身を `signal.NotifyContext(parent, sigs...)` にする(`org_gate.go:127` の前で return) | go1.25.8 で red、go1.26.0 では生き残り | go1.25.8: 中断のテストの SIGINT・SIGTERM・SIGHUP(`org_gate_test.go:701`、`interrupted (context canceled)`)、`TestOrgGate_SignalContextKeepsLaterSignalsCaught`(`:796`、子が `cause: context canceled` を出した)。go1.26.0 では 3 本とも通った |
| M_b | (b) 最初の信号で `cancel(nil)` を呼ぶ(`:134`) | red | M_a の go1.25.8 と同じ 4 か所 |
| M_c | (c) 最初の信号の cancel の直後に `signal.Stop(ch)` を呼ぶ | red | `TestOrgGate_SignalContextKeepsLaterSignalsCaught` だけ(`:793`、子が 2 つ目の SIGTERM で `signal: terminated` で終わった)。`-count=10` で 10 回とも red |
| M_c2 | (c) の変形: `signal.Stop(ch)` を cancel の前に呼ぶ | red | M_c と同じ。`-count=10` で 10 回とも red |
| M_d | (d) stop が cancel しない(`:140` を外す) | red | `TestOrgGate_SignalContextParentCauseAndStop` だけ(`:746`、`after stop: ctx.Err() = <nil>`) |
| M_e1 | (e) 親が終わっていても goroutine を起こす(`:130` を `if true`) | 生き残り(同値) | 絞った回でも cli 全体でも 0 fail。親が終わっていれば goroutine は `<-ctx.Done()` ですぐ抜け、cause は親のもののまま残る(`WithCancelCause` の 2 回目からの cancel は何もしない) |
| M_e2 | (e) の変形: 条件を逆にして、親が生きているときに goroutine を起こさない | red | 中断のテストの信号の 3 subtest(`:698`、15 秒以内に返らない)、`TestOrgGate_SignalContextKeepsLaterSignalsCaught`(子の `:773`、5 秒以内に cancel されない) |
| M_f1 | (f) cause を `s.String()` だけにする(` signal received` を外す) | red | 中断のテストの信号の 3 subtest(`interrupted (interrupt)` など)、`TestOrgGate_SignalContextKeepsLaterSignalsCaught`(`:796`) |

足した 6 つ(信号ごとの文、親、呼び出しの場所、channel):

| ID | 何を壊したか | 結果 | 落としたテストと失敗の行 |
| --- | --- | --- | --- |
| M_f2 | cause をどの信号でも `interrupt signal received` にする | red | 中断のテストの SIGTERM と SIGHUP(SIGINT は通る)、`TestOrgGate_SignalContextKeepsLaterSignalsCaught`(`:796`) |
| M_h | `context.WithCancelCause(context.Background())` で作る(`:127`。親が終わっても届かない) | red | `TestOrgGate_SignalContextParentCauseAndStop`(`:740`)、中断のテストの ctx の取り消し(`:698`、15 秒) |
| M_i | 呼び出し(`:97`)から SIGHUP を外す | red | 中断のテストの SIGHUP(`:698`、15 秒) |
| M_i2 | 呼び出しに `context.Background()` を渡す | red | 中断のテストの ctx の取り消し(`:698`、15 秒) |
| M_d2 | stop が `signal.Stop(ch)` を呼ばない(`:141` を外す) | 生き残り(コミットしたテスト)、probe で red | cli 全体でも 0 fail。下の probe が落とした |
| M_j | 信号の channel を buffer なしにする(`:128`) | 生き残り | cli 全体でも 0 fail |

M_c と M_c2 を落とすのは子プロセスのテストだけで、CLI の中断のテストは通った。`signalSelf`(`org_gate_test.go:637-646`)がテストの側でも同じ信号を受けるので、後の信号が ralph を終わらせるかどうかは、中断のテストからは見えない。

生き残りの 3 件:

- M_e1 は同値の mutant
- M_d2: stop のあとも信号を受け取って捨て続ける。CLI では stop は RunE の `defer`(`org_gate.go:98`)で走り、そのあとはすぐにプロセスが終わる。利用者に見える違いは「gate が返ってから ralph が終わるまでの短い間の Ctrl-C が捨てられる」ことだけで、おそらく実害はない。未確認です。helper の doc(`:109-125`)は stop が sigs の登録を外すと書くが、それを固定するテストはない。scratch の `_test.go` を `-overlay` で足した probe `TestProbe_SignalContextStopUnregisters` は、子プロセスで stop を呼んでから自分に SIGHUP を送り、子が SIGHUP で終わることを見る。HEAD では 10 回とも PASS、M_d2 では 10 回とも FAIL だった(`child err = <nil>`、子が `survived` を出した)。テストを足すなら、この probe がそのまま使える。全文は mutation の evidence にある
- M_j: buffer のない channel には、`signal.Notify` は受け手が待っていないときに送らず、信号を捨てる。goroutine が select で待っている間に来た信号は届くので、スクリプトが起動してから送るテストの信号は届く。捨てられるのは、`gateSignalContext` が登録してから goroutine が select に入るまでの短い間に来た信号で、そのとき gate は中断されない(登録は済んでいるので ralph も終わらない)。この間を狙うテストは作りにくい。この形は staticcheck の SA1017 と go vet の sigchanyzer が指摘する検査の対象だが、静的解析は /verify の担当なので、この回では流していない。指摘されるかは未確認です

### Coverage(cycle 3)

go1.25.8 の `go test -count=1 -cover ./internal/cli/` で `internal/cli` は 86.8%(cycle 1 は 86.7%)。`-run 'OrgGate'` の profile で、`org_gate.go` は 51 文のうち 50 文を通り、`gateSignalContext` は 100%、`newOrgGateCmd` は 93.3%。通らない 1 文は cycle 1 と同じ、`cmd.Context()` が nil のときの `context.Background()`(`org_gate.go:87-89`)。子プロセスの中の実行は profile に入らないが、goroutine の信号の case は親のプロセスで CLI の中断のテストが通す。ほかのパッケージは cycle 2 からコードが変わっていないので、この回では測っていない。

### Test gaps(cycle 3)

- 手元の go1.26.0 では、helper を `signal.NotifyContext` に戻しても(M_a)どのテストも落ちない。go.mod の 1.25.8 で走る環境だけが見分ける。`./scripts/run-test.sh` は手元の Go で走るので、1.26 の開発機では見分けない。CI の verify は `setup-go` を使わず runner の Go で走り、版はログに出ない。runner の Go が 1.26 以上になると、CI でも見分けなくなる。self-review cycle 3 の C3-1 を実行で確かめた形になる。閉じるなら、CI の verify でも release.yml と同じく `go-version-file: go.mod` で版を固定する
- stop が信号の登録を外すこと(M_d2)を固定するテストがない。probe で閉じられることは確かめた
- buffer のない channel(M_j)で信号が捨てられる間は、テストでは狙いにくい
- 中断がスクリプトの実行中に届いたとき、まれに 5 秒余分にかかる(上の Flakiness)。2 つの toolchain で、180 回中 6 回起きた。原因は推測で、確かめていない
- cycle 1・cycle 2 の Test gaps(org 層の fixture で HEAD と base が同じこと、`gateOutput` の改行のない出力、本当のロックの timeout の文、Linux と混んだ CI での `tests/test-run-verify-concurrency.sh`、本物の leader の座席)は、8b681dc3 が関係しないので、そのまま残る

### 判断(cycle 3)

- Pass: `./scripts/run-test.sh`(go1.26.0、changed、full に落ちた)は rc 0 で、shell 41 本、PASS の行 4,118、Go 8 パッケージ。go1.25.8 の `go test ./...` は top-level 1,542 本と subtest 2,005 本が PASS で 0 fail、go1.26.0 の org と cli も top-level 1,325 本と subtest 1,953 本が PASS で 0 fail。足した 2 本はどちらの toolchain でも PASS で、消えたテストはない。go1.25.8 の `-race -run 'Gate|Lock|SignalContext' -count=3` は 255 回すべて PASS で、`DATA RACE` は 0 件。依頼の 3 本は、2 つの toolchain の `-count=20` と go1.25.8 の `-race -count=5` で 0 fail。修正前のコードは go1.25.8 で CI と同じ 3 subtest が落ち、HEAD では通る。mutation は依頼の 6 つのうち同値の (e) を除く 5 つと、その変形・足した mutant を合わせた 11 件が red
- 生き残り: M_e1 は同値。M_d2 はテストの穴で、probe で閉じられることを確かめた(コミットしていない)。M_j は、テストで狙いにくい短い間だけに効く
- Fail: 0
- Blocked: なし
- 判断: pass(cycle 3)

## Cycle 4

- Date: 2026-10-14(JST 01:07〜01:40)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 4 回目。cross-review cycle 3 の #1(gate をやり直すとき、reviewer がコミット済みのレポートを書き換えていて `git rebase` が断られる)を直すため、上限を 4 に上げて回した。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま。直しの f1b0352e、self-review cycle 4(31ae9f7d)、verify cycle 4(1e4a2fa4、pass)のあとに走らせた
- Scope: HEAD 1e4a2fa4。cycle 3 の HEAD 697e1577 から変わったコードとテストは f1b0352e の 6 ファイルだけだった(`git diff --stat 697e1577..1e4a2fa4 -- '*.go' scripts tests templates internal go.mod go.sum .github .claude .agents .codex`)。leader の雛形 `internal/org/prompts/leader.md`(手順 9・10 と節の前置き)、`internal/org/prompts_test.go`(既存の 2 本の assertion の書き換えと、手順 9 の順と手順 10 の否定の assertion。`func Test` は増えていない)、org skill の 4 面。Go のコードは変わっていない。雛形は `//go:embed prompts/*.md`(`internal/org/prompts.go:18`)で `internal/org` に入るので、build が変わるのは `internal/org` と、それに依存する `internal/cli`、`internal/insights`、`cmd/ralph`(`go list -deps`)になる。behavioral test だけを実行した(静的解析は /verify の担当)。この節の行番号はすべて HEAD 1e4a2fa4 のもの
- Toolchain: `./scripts/run-test.sh` は手元の既定の go1.26.0、全体の `go test` は go1.25.8(`GOTOOLCHAIN=go1.25.8`)、`RenderRolePrompt` の繰り返しと mutation は 2 つの toolchain の両方で走らせた
- Evidence(上書きしたものと、末尾に足したもの):
  - `docs/evidence/test-2026-10-13-org-gate-run-test.txt`: 上書きした。go1.26.0 での `./scripts/run-test.sh`(changed)の 2 回目の出力で、先頭に toolchain と 1 回目の要約、末尾に `RC=` がある(1 回目と 2 回目の違いは下の「Test execution(cycle 4)」)
  - `docs/evidence/test-2026-10-13-org-gate-go-test.txt`: 上書きした。go1.25.8 の `go test ./... -count=1 -timeout 30m -v` の 2 回目。PASS の行を数えるため、依頼のコマンドに `-v` を足した
  - `docs/evidence/test-2026-10-13-org-gate-flaky.txt`: 末尾に「Cycle 4」を足した。toolchain とテストごとの回数と時間の幅、生の出力
  - `docs/evidence/test-2026-10-13-org-gate-mutation.txt`: 末尾に「Cycle 4」を足した。mutant ごとの overlay の JSON、差分、2 つの toolchain での結果、落ちた assertion の行(assertion が出す節や項目の全文は先頭の 3 行だけ残し、行末の空白は削った)、skill の mutant E の要約、git の probe の script と出力
  - `docs/evidence/test-2026-10-13-org-gate-race.txt` と `-shell.txt` は、この回は走らせておらず、前の回のまま(f1b0352e は Go のコードも shell のスクリプトも変えていない)
  - `$HOME` は `~` に、scratchpad のパスは `<scratchpad>` に置き換えた。置き換えが要ったのは、go-test の evidence の 4 行(doctor のテストが出す `~/.agents/skills/agmsg` と `~/.codex/models_cache.json`)と、mutation の evidence の overlay の JSON とコマンドの行だけだった。足した 2 つの evidence は、元の中身が新しい中身の先頭とバイト単位で一致する。どの evidence にも U+FFFD はない
  - wrapper が書いた `docs/evidence/verify-2026-10-13-160809.log` と `verify-2026-10-13-162234.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-gate.jsonl` に test の 1 行(`--cycle 4`、verdict pass)を足し、この報告と同じコミットに入れた
- テストもコードも変えていない。mutant の写し、skill を戻した clone、git の probe は scratchpad の `og4/` に置いた。テストのあと、`~/.config/herdr` の下に 01:05 より新しいファイルはなく、main のチェックアウトの `git status --short` は空のまま、`$TMPDIR` に `ralph-verify-*` は残っていない。worktree で変わったのは、この報告、上の 4 つの evidence、insight event だけ。osascript は cycle 3 と同じく、CLI の `TestMain` が通知を差し替え、org の `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` は darwin で skip した。プロセスとしては見張っていない

### Test execution(cycle 4)

| Suite / Command | Toolchain | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。cycle 1〜3 と同じく `shared:scripts/run-verify.sh` で full に落ち、golang の pack を選んだ)2 回目 | go1.26.0 | shell 41 本(PASS の行 4,118)、Go 8 パッケージ | すべて | 0 | shell 1 | 9 分 45 秒、rc 0 |
| 同じコマンドの 1 回目(`nohup … &` で起動) | go1.26.0 | shell 41 本(PASS の行 4,114)、Go 8 パッケージ | すべて | 0 | shell 3 | 10 分 0 秒、rc 0 |
| `go test ./... -count=1 -timeout 30m -v` 2 回目 | go1.25.8 | top-level 1,545、subtest 2,005 | top-level 1,542、subtest 2,005 | 0 | 3 | 3 分 46 秒(cli 224.9 s、org 135.8 s)、rc 0 |
| 同じコマンドの 1 回目(`nohup … &` で起動) | go1.25.8 | 2 回目と同じ | 2 回目と同じ | 0 | 3 | 3 分 21 秒(cli 199.3 s、org 106.6 s)、rc 0 |
| `go test ./internal/org/ -run 'RenderRolePrompt' -count=5 -v` | 両方 | 19 本 × 5 回 × 2 = 190 | 190 | 0 | 0 | 0.4 s ずつ、rc 0 |
| mutation(雛形の 13 件と対照 1 件、`-overlay`) | 両方 | 13 件 | red 13 件 | 生き残り 0 | - | 1 件 0.7〜3.3 s |
| mutation E(skill の 4 面を旧い順に戻した clone で `GOFLAGS=-count=1 RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`) | go1.26.0 | shell 41 本(PASS の行 4,114)、Go 8 パッケージ | すべて | 0 | shell 3 | 10 分 32 秒、rc 0(生き残り) |
| git の probe(R1〜R5、下の表) | - | 5 場面 | 期待どおり | - | - | 1 秒未満 |

- 1 回目の `run-test.sh` と `go test` は、zsh から `nohup <script> &` で起動したので、SIGINT と SIGHUP を無視した状態で始まった。同じ起動で python の `signal.getsignal` が 2 つとも `SIG_IGN` を返し、前景では返さないことを確かめた。`tests/test-check-template.sh` の I は、dash に INT と HUP を送り、信号が無視される環境ではその 2 つを SKIP にする(`tests/test-check-template.sh:712-713`)。1 回目はこの 4 件の PASS が 2 件の SKIP に変わり、ほかの 40 本の suite の数は cycle 3 と同じだった。同じ suite を前景で打ち直すと 60 passed、0 failed、0 skipped だった。そのあと信号を無視しない起動(Bash ツールのバックグラウンド実行)で 2 つとも打ち直し、evidence には 2 回目を残した。1 回目の 2 つの結果は scratchpad にだけある
- shell の PASS の行は、cycle 1〜3 と同じ数え方で 2 回目が 4,118 行。suite ごとの PASS・FAIL・SKIP の数は cycle 3 の実行と 41 本すべて同じだった(`diff` が空)。Skipped は前と同じ `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0)
- 2 回目の `run-test.sh` の中の Go は `internal/org`(68.0 s)だけを流し直し、ほかの 7 パッケージは `(cached)` だった。`internal/cli` は 1 回目の結果(124.0 s で流し直した回。`internal/org` は 67.6 s)を使った。`internal/insights` は 1 回目でも `(cached)` で、それより前に同じコードで流した結果を使っている。キャッシュなしの結果は go1.25.8 の `go test ./...` にある(どのパッケージも `-count=1`)
- go1.25.8 の全体の件数は cycle 3 と同じで(top-level 1,542、subtest 2,005、skip 3)、テスト名を `comm` で比べても差はない。f1b0352e はテストを足さず、既存の `TestRenderRolePrompt_Leader_FeatureOrgGate` と `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の中の assertion を変えた
- load average は、1 回目の `run-test.sh` の開始時(01:08)が 51.42 で、ほかの session の負荷による。2 回目の開始時は 5.18、go1.25.8 の 2 回目は 5.78、繰り返しの回は 6.20、clone の回は 6.41 だった。前に flake を起こした doctor のテストと `TestRunWatcher_TimeoutIndependentOfSmallInterval` は、どの実行でも落ちなかった。`-v` を付けた 2 回の go1.25.8 の実行では、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` の subtest は 0.66〜1.81 s で、cycle 2・3 で見た 5 秒台は出なかった

### Flakiness(cycle 4)

`go test ./internal/org/ -run 'RenderRolePrompt' -count=5 -v` を go1.25.8、続けて go1.26.0 で前景から流した(JST 01:13、1 回目の `run-test.sh` の shell の suite と重なった)。19 本とも 2 つの toolchain で 5 / 5、0 fail で、どの回も 0.00 s だった。subtest はない。19 本は描いた雛形の文字列を読むだけで、時間や外のプロセスに頼らないので、揺れる要素はおそらくない。揺れによる失敗は見つからなかった。

### Mutation(cycle 4)

どの mutant も、scratchpad に写した `internal/org/prompts/leader.md` の 1 か所を書き換え、tracked のファイルには触れていない。雛形は `//go:embed` で読まれるので、overlay の JSON で `.md` を差し替えると build に入る。書き換えていない写し X0 は 2 つの toolchain で 19 本とも通り、各 mutant では描いた雛形が変わったので、overlay が embed に届くことも確かめられた。`go test -count=1 -overlay <json> ./internal/org/ -run RenderRolePrompt -v` を、go1.26.0(JST 01:09〜01:11)、go1.25.8(01:11〜01:13)の順に 1 件ずつ流した。13 件とも、2 つの toolchain で落ちたテストと行が同じだった。

assertion の行は、`prompts_test.go:526` が節の前置き(`TestRenderRolePrompt_Leader_FeatureOrgProcedure`)、`:796` と `:798` が節全体の順のループ(語句がない、順が違う)、`:845` が項目ごとの語句の表、`:884` が f1b0352e で足した手順 9 の順のループ、`:891` が同じく足した手順 10 の否定。`:526` のほかはすべて `TestRenderRolePrompt_Leader_FeatureOrgGate` の中にある。

依頼の 6 つ:

| ID | 何を壊したか | 結果 | 落としたテストと失敗の行 |
| --- | --- | --- | --- |
| M_a1 | (a) 手順 9 を rebase が先の順に戻す: fetch と rebase を先に書き、レポートのコミットと `git status --porcelain` の確認を fetch の失敗の文のあとへ動かす | red | `:798`(fetch・rebase・`--abort` が前の手順より先)、`:845`(「何も出なくなったら、`git fetch origin <既定のブランチ>` を打つ」がない)、`:884` |
| M_b1 | (b) `git status --porcelain` の確認を外す(コミットだけにする) | red | `:796`、`:845`、`:884` |
| M_c | (c) implementer のほかの残りを BLOCKED で上げる枝を外す | red | `:845` だけ |
| M_d1 | (d) 手順 10 を旧い文「report と reviewer のレポート(`docs/reports/` の下)をコミットする」に戻す | red | `:796`、`:845`、`:891` |
| E | (e) skill の 4 面を f1b0352e の前の文(旧い順)に戻す | 生き残り(予想どおり) | なし。下の説明 |
| M_f | (f) implementer の残りの枝から「(per-plan では `PLAN:` のヘッダを付ける)」を外す(rebase の衝突の枝の同じ語句は残す) | red | `:845` だけ |

足した 8 つ(全体の戻し、順だけの変更、各枝の語句、前置き):

| ID | 何を壊したか | 結果 | 落としたテストと失敗の行 |
| --- | --- | --- | --- |
| M_a2 | (a) の変形: 雛形全体を f1b0352e の前の版に戻す(直しの revert) | red | 2 本。`TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`:526`)と `..._FeatureOrgGate`(`:796` 3 件、`:845` 7 件、`:884`、`:891`) |
| M_a3 | (a) の変形: 語句はすべて残し、手順 9 の頭に「先に `git fetch origin <既定のブランチ>` と `git rebase origin/<既定のブランチ>` を打ち、」を足す(順だけが変わる) | red | `:798`(fetch と rebase)、`:884` |
| M_b2 | (b) の変形: 確認は残し、fetch の前の「何も出なくなったら、」を外す | red | `:845`、`:884` |
| M_c2 | (c) の変形: 「自分ではコミットしない」を「自分でコミットする」にする | red | `:845` だけ |
| M_d2 | (d) の変形: 手順 10 の「report だけをコミットする(…)」は残し、「念のため report と reviewer のレポートをもう一度コミットする。」を足す | red | `:891` だけ |
| M_g | implementer の残りの枝から「終わったら reviewer にレビューを頼み直す(手順 8 の往復に戻る)」を外す | red | `:845` だけ |
| M_h | 節の前置きの一覧を旧い文(「10 の report と reviewer のレポートと計画の移動」)に戻す | red | `..._FeatureOrgProcedure` の `:526` だけ |
| M_i | やり直しの理由の文(「gate をやり直すときは、… 断られる」)を外す | red | `:845` だけ |

- f1b0352e で足した 2 つの assertion のうち、手順 10 の否定(`:891`)は M_d2 を単独で落とした。語句を足して矛盾させる変更は、否定の assertion がないと見分けられない
- 手順 9 の順のループ(`:884`)が落とした 5 件(M_a1、M_a2、M_a3、M_b1、M_b2)は、どれもほかの行でも落ちた。順を変える M_a1 と M_a3 は節全体の順のループ(`:798`)が、語句を削る M_b1 と M_b2 は語句の表(`:845`)が落とす。語句の表は「何も出なくなったら、`git fetch …` を打つ」を続けて固定し、節全体の順のループはコミットと確認を fetch の前に並べているので、`:884` だけが見分ける変更は作れなかった。この 13 件では、`:884` を外しても red の数は変わらない
- E: `-overlay` は build の中身だけを差し替え、テストが実行中に `os.ReadFile` で読むファイルには効かない。そこで worktree を scratchpad に `git clone` し(1e4a2fa4、`origin/HEAD` を main に設定)、4 面を `git show f1b0352e^:<path>` の中身に置き換えて、テストの段全体を `GOFLAGS=-count=1 RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` で流した。rc 0 で、Go の 8 パッケージはキャッシュなしですべて ok、shell の suite ごとの数は、同じく `nohup` で起動した worktree の 1 回目と 41 本すべて同じだった(`tests/test-check-template.sh` の 2 件の SKIP は起動のしかたによるもの)。org の skill を読むテストは `internal/org/send_defaults_sync_test.go` があるが、読むのは既定の待ち時間の数字だけで、締めの順は見ない。4 面を同じ向きに変えたので、drift の検査も見分けないはず(verify cycle 4 の Coverage gaps と同じ。この回では静的な検査は流していない)

### git の probe(cycle 4)

verify cycle 4 の probe(P1〜P3、`docs/evidence/verify-2026-10-13-org-gate-probe.txt` の「Cycle 4」)を R1 で繰り返し、手順 9 の語句がかかわる場面を 4 つ足した。ralph の binary は使わず、git 2.49.0 だけで、scratchpad に bare の origin、`feat/x` の clone、origin/main を進める別の clone を作った。`GIT_CONFIG_GLOBAL` と `GIT_CONFIG_SYSTEM` は `/dev/null` にした。ネットワーク、herdr、agmsg は使っていない。script と出力は mutation の evidence の末尾にある。

| # | 入力 | 結果 |
| --- | --- | --- |
| R1 | gate のやり直し。コミット済みのレポートを書き換え(` M docs/reports/review.md`)、origin/main が進んだ | `git rebase origin/main` は rc 1 で「error: cannot rebase: You have unstaged changes.」。始まっていない rebase への `git rebase --abort` は rc 128。レポートをコミットすると porcelain は空になり、rebase は rc 0、`--is-ancestor` も rc 0。verify の P2・P3 と同じ |
| R2 | 書き換えたレポートを `git add` だけして、コミットしていない(`M  docs/reports/review.md`) | rebase は rc 1 で「Your index contains uncommitted changes.」。コミットすると rc 0 |
| R3 | 未追跡のファイルだけが残っている(`?? leftover.txt`) | rebase は rc 0 で通り、未追跡のファイルは rebase のあとも残る。git は止めないので、残りに気づくのは手順 9 の `git status --porcelain` の確認だけ |
| R4 | コミット済みのレポートを書き換え、`git -c rebase.autoStash=true rebase` | rebase は rc 0 で通り、書き換えはコミットされないまま rebase のあとに戻る(` M docs/reports/review.md`) |
| R5 | レポートは前にコミット済みで、書き換えもない(self-review cycle 4 の C4-1) | `git add` は rc 0、`git commit` は rc 1 で「nothing to commit, working tree clean」。porcelain は空で、rebase は rc 0 |

- R1 と R2 から、追跡しているファイルにコミットしていない変更が残れば、作業ツリーでも index でも rebase は断られる。手順 9 が求める「コミットしてから porcelain が空」は、この 2 つをどちらも満たす
- R3 と R4 は、git が rebase を断らない場合を示す。R4 のように利用者の git の設定に `rebase.autoStash=true` があると、旧い順でも rebase は通り、書き換えたレポートは旧い手順 10 でコミットされるので、cross-review cycle 3 #1 の筋書きは起きない。新しい順は、設定によらず rebase の前にコミットする。R3 の未追跡の残りは、どちらの順でも rebase を止めない。新しい順では rebase の前の porcelain の確認で止まり、TASK か BLOCKED に回る。確認をすり抜けた残りは gate が断る。未追跡のファイルを gate が断ることは、既存の `TestGate_RefusedBeforeTheScripts` の「an untracked file」と CLI の `TestOrgGate_RefusalPrintsNothingAndRecordsNothing` の「an untracked file」が固定していて、この回の実行でも通った
- R5: コミットするものがなければ `git commit` は rc 1 で終わるが、作業ツリーは空なので、手順 9 は先へ進める。雛形は、この rc 1 をどう扱うかを書いていない。leader がこれを失敗と読んで止まるかどうかは、本物の座席では確かめていない

### Coverage(cycle 4)

Go のコードは変わっていないので、statement coverage は測り直していない(cycle 3 の数字のまま)。雛形と skill の文は coverage の対象にならないので、上の mutation で代わりに確かめた。

### Test gaps(cycle 4)

- skill の 4 面の締めの順を固定するテストがない(E が生き残った)。足すなら、verify cycle 4 が書くとおり、`prompts_test.go` の `markdownItem` と同じやり方で `.claude/skills/org/SKILL.md` の「org の終わり方」の手順 1 と 2 を読み、「reviewer のレポート」が `git fetch` より前にあること、手順 2 に「report だけをコミットする」があることを見る Go のテスト 1 本が最小になる。残りの 3 面は drift の検査が 1 面と同じであることを見る
- 手順 9 の順のループ(`:884`)は、この回の mutant ではほかの assertion と重なり、単独では何も落とさなかった。守りが重なっているだけで、穴ではない
- 本物の leader の座席が手順 9 の枝(implementer の未コミットの変更を TASK で返す、それ以外を BLOCKED で上げる)を選べるか、`git commit` が空振り(R5)したときに止まらずに進むかは確かめていない。テストは雛形の語句まで(verify cycle 4 と self-review cycle 4 の Known gaps と同じ)
- probe は git だけで打ち、手順 9 から gate までを ralph の binary で通してはいない。git の動き(R1〜R5)と、gate が汚れた作業ツリーを断ること(既存のテスト)は、それぞれ別に確かめた
- cycle 1〜3 の Test gaps(org 層の fixture で HEAD と base が同じこと、`gateOutput` の改行のない出力、本当のロックの timeout の文、stop が信号の登録を外すこと、go1.26 の開発機では `signal.NotifyContext` への戻しを見分けないこと、中断のまれな 5 秒、Linux と混んだ CI での `tests/test-run-verify-concurrency.sh`)は、f1b0352e が関係しないので、そのまま残る

### 判断(cycle 4)

- Pass: `./scripts/run-test.sh`(go1.26.0、changed、full に落ちた)は 2 回とも rc 0 で、信号を無視しない 2 回目は shell 41 本、PASS の行 4,118、suite ごとの数が cycle 3 と同じ、Go 8 パッケージ。1 回目の 2 件の SKIP は `nohup` の起動による環境の違いで、その suite を前景で打ち直すと 60 / 60 だった。go1.25.8 の `go test ./... -count=1` は 2 回とも top-level 1,542 本と subtest 2,005 本が PASS で 0 fail、テスト名は cycle 3 と同じ。`RenderRolePrompt` の 19 本は 2 つの toolchain の `-count=5` で 190 回すべて PASS。mutation は、雛形の 13 件(依頼の (a)(b)(c)(d)(f) と、その変形・足した mutant)が 2 つの toolchain で red。git の probe は、レポートを先にコミットすれば rebase が通り、残りがあれば rebase か porcelain の確認のどちらかで止まることを示した
- 生き残り: E(skill の 4 面を旧い順に戻す)は、依頼の予想どおり生き残った。skill の締めの順を読むテストがないための穴で、上の Test gaps に最小のテストを書いた
- Fail: 0
- Blocked: なし
- 判断: pass(cycle 4)
