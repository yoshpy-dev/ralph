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

- Verdict: pass
- Pass: `./scripts/run-test.sh` は changed と full のどちらも rc 0(shell 41 本、PASS の行 4,118、Go 8 パッケージ)。`go test ./... -count=1` は top-level 1,538 本が PASS で 0 fail、足した 34 本もすべて PASS で、消えたテストはない。`-race`(org と cli)も 0 fail で、`DATA RACE` は 0 件。直接打った shell のテスト 6 本は 176 件すべて PASS。concurrency のテストは 5 回連続と負荷の下の 3 回で揺れず、gate のテストの `-count=3` も 0 fail。mutation は依頼の 3 つの動き(head_moved、環境の固定、終了コードのファイルがないとき)の 9 件を含む 15 件がすべて red
- Fail: 0
- Blocked: なし
