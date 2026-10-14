# Test report: org-merge-guarantee

- Date: 2026-10-14(JST 18:17〜19:30)
- Plan: docs/plans/active/2026-10-14-org-merge-guarantee.md(`Status: Approved`、Approved sha256:d2923a8bf359)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1)
- Scope: HEAD 38ac8953(base a16d7c4f、26 ファイル、+4,075/-89)。/verify が見た 1e71cadb から増えたのは verify の報告と insight event だけで、コードとテストは同じ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0、gh 2.102.0
- Evidence(どれもコミットする。`.txt` は `docs/evidence/` で追跡される。wrapper が書いた `docs/evidence/verify-2026-10-14-091749.log` は gitignore の対象なのでコミットしない):
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-run-test.txt`: `./scripts/run-test.sh`(changed)の出力から suite ごとの集計の行と Go の行を抜いたもの。`(cached)` だった 4 パッケージを `-count=1` で打ち直した結果と、org-merge-order の 4 回目との比較(shell の行はすべて同じ)も入れた
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-go.txt`: `go test ./internal/cli/... ./internal/org/... -count=1 -v -coverprofile` を go1.25.8 と go1.26.0 で打った結果、`-race`、追加した 44 本のテストの PASS の行、関数ごとの coverage
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-mutation.txt`: `go test -overlay` の mutant 74 件と対照 X0 の結果、生き残り 5 件を落とす probe の行 PB1〜PB4、TestMain の固定を外したときの感度の確認
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-real-gh.txt`: 本物の gh で、merge queue で決まる経路を読むだけで確かめた記録(zed-industries/zed、Homebrew/homebrew-core、astral-sh/uv)
- テストもコードも変えていない。mutant、probe のテスト、偽の `gh`(canary)は scratchpad に置き、`go test -overlay` で読ませた。worktree に書いたのは、この報告と上の evidence 4 本と insight event だけ

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed、go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 701 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/... ./internal/org/... -count=1 -v -coverprofile` | cli: top-level 651、subtest 556。org: 708、1,461。org/driver: 48、89。org/protocol: 15、56 | 同じ数から org の skip 1 を引いた数 | 0 | org 1 | 344 秒(cli 340.7 s、org 148.0 s)、rc 0 |
| `go test ./internal/cli/... ./internal/org/... -count=1 -v`(go1.26.0) | go1.25.8 と同じ | go1.25.8 と同じ | 0 | org 1 | 429 秒(cli 421.2 s、org 175.8 s)、rc 0 |
| うちこのブランチで足したテスト 44 本 | top-level 44、subtest 128 | 44、128(両方の版) | 0 | 0 | - |
| `(cached)` だった config、insights、scaffold、upgrade を `-count=1`(go1.26.0) | 4 パッケージ | 4 | 0 | - | 4 秒 |
| `go test -race`(org は `-run 'MergeGuarantee\|MergeOrder'`、cli は `-run 'MergeGuarantee\|OrgMergeOrder\|TestStatusCmd'`) | 2 パッケージ | 2 | 0 | - | 74 秒、DATA RACE なし |
| mutation(`go test -overlay`、go1.26.0) | mutant 74 件と対照 X0 | red 69 件(R14 は 1 回目に build できず、書き直して red) | 生き残り 5 件(下の「Mutation」) | - | 約 33 分 |
| 本物の gh での確認(読むだけ、HEAD の binary) | repository 3 つで doctor、status、merge-order | すべて期待どおり | 0 | - | - |

- shell は 41 本すべてが `OK` で終わり、集計の `FAIL:` の行はどれも 0。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell を変えていないが、`scripts/verify.local.sh` がいつも shell の suite を流す。行番号と時間を除くと、shell の行は org-merge-order の 4 回目(`test-2026-10-14-org-merge-order-c4-run-test.txt`)と 1 行も違わない
- `run-test.sh` の中の Go は `internal/cli`(166.9 s)と `internal/org`(82.0 s)を流し直し、ほかの 6 パッケージは `(cached)` だった。org/driver と org/protocol は上の org/... の実行で、残りの 4 つは `-count=1` で、どれも ok を確かめた
- go1.25.8 と go1.26.0 で、テストの名前の行(PASS/SKIP/FAIL、時間を除いて並べた 3,585 行)は同じだった
- org の Skipped は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin で飛ばす作りで、この差分と関係ない)
- 差分で足した `func Test` は 44 本(`git diff a16d7c4f...HEAD -- '*_test.go'` の `+func Test`)、消した `func Test` は 0 本。44 本とその subtest はどちらの版でも PASS だった
- 前に揺れた doctor のテスト(`TestRunDoctor*`)、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は、どの実行でも落ちなかった
- どの実行でも、PATH の先頭に偽の `gh`(呼ばれた引数を書いて exit 97 で終わる)を置いた。呼ばれた回数は Go の 2 回の実行、mutant、probe、`-race` を通して 0 回

## Coverage

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.3%、`internal/cli` 86.9%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%
- 新しいファイルと変えたファイル: `internal/org/mergeguarantee.go` 253/260 文(97.3%)、`internal/cli/doctor_merge_guarantee.go` 20/20 文(100%)、`internal/cli/status.go` 217/226 文(96.0%)、`internal/cli/org_merge_order.go` 40/44 文(90.9%)、`internal/org/mergeorder.go` 317/318 文(99.7%)
  - `mergeguarantee.go` で通らない 7 文のうち 5 文は、`Now`、`Exec`、`LookPath` が nil のときに本物の時計、`realMergeOrderExec`、`exec.LookPath` を使う行(176、187、190、741、764)。テストは本物の git と gh を打たないので(AC8)、ここは実機の証跡でしか通らない。残りの 2 文は、state dir の `os.Stat` が「ない」以外のエラー(権限など)を返したときの save(647-648)と read(721-722)
  - `status.go` の新しい行で通らないのは `cmd.Context()` が nil のときの `context.Background()`(154)だけ。ほかは前からある行
  - `org_merge_order.go` と `mergeorder.go` で通らない行は、どれも前からある行(エラーを返す行と `Exec` の既定値)
- Function: `mergeguarantee.go` の 29 関数のうち 25 が 100%(`CheckMergeGuarantee` 81.2%、`saveMergeGuarantee` 93.3%、`ReadMergeGuarantee` 92.3%、`readMergeGuaranteeOrigin` 94.1%)。`doctor_merge_guarantee.go` の 3 関数、`mergeGuaranteeLine`、`printMergeOrder`、`printStatusJSONAllOrgs` は 100%
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

## Test plan coverage map

計画の Test plan の項目ごとに、それを確かめるテストを挙げる。「mutant」の列は、その項目の判定を壊した mutant がテストで落ちたかどうか(id は mutation の evidence の A)。

| Test plan の項目 | テスト | mutant |
| --- | --- | --- |
| ruleset: strict で必須の検査あり、0 件、strict が false、merge_queue と必須の検査、merge_queue だけ、規則なし | `TestMergeGuarantee_RulesetsDecideTheKind`(10 行。ほかに「strict の規則が 2 つなら多いほう」「strict と merge queue の両方で検査 0 件」「検査のない strict の規則と別の規則の検査」「規則が別の種類だけ」) | D01〜D08、D06b、O01 red |
| classic: strict true / false、必須の検査なし(`off`)、strict を読めない(404 `Not Found`、403)、保護なし | `TestMergeGuarantee_ClassicProtectionDecidesTheKind`(17 行。403 は body と stderr の 2 形、`Branch not protected`、`Required status checks not enabled`、500、strict のない答え、JSON でない答え、contexts と checks の重複) | C01〜C04、D04、D05、O02、S01、S02、S05、S06、S08 red。S07 は生き残り(下) |
| ruleset と classic の組み合わせ(merge_queue と classic の必須の検査) | 同じ表の最後の 3 行(merge queue と classic の検査、両方の検査で strict なし、ruleset の検査と admin の要る strict) | D03 red |
| 失敗: origin なし、host か owner/repo のない URL、gh が PATH にない、gh の失敗(ssh の別名)、期限切れ、JSON が読めない、`default_branch` が空 | `TestMergeGuarantee_FailuresAreUnknownWithAReason`(15 行。gh がないときは LookPath が 1 回で gh は 0 回)、`TestMergeGuarantee_DeadlineExceededIsUnknown`(5 か所)、`TestMergeGuarantee_OneDeadlineCoversEveryCall`、`TestMergeGuarantee_CallerContextDoneIsUnknown`、`TestMergeGuarantee_LongErrorIsCutToOneLine` | D10、H09、T01〜T04 red |
| host: https・ssh・scp 形から host を読む、`ssh.github.com` の読み替え、どの gh にも `--hostname` | `TestMergeGuarantee_HostComesFromOriginsURL`(7 行)、`TestMergeGuarantee_ReadsTheHostWithTheOwnerAndRepo`(8 件)。判定のテストはどれも `assertGHCalls` で全部の gh の引数、場所、`GH_PROMPT_DISABLED=1` を確かめる | H01〜H04、H07、H08 red |
| paginate: 1 つの配列、連結された配列 | `TestMergeGuarantee_PaginatedRulesAreAllRead`(7 行) | D11 red |
| 改名(`full_name` が違う) | `TestMergeGuarantee_RenamedRepositoryStillDecides` | - |
| キャッシュ: 書いて読む、壊れた、版の違い、一時ファイルが残らない、24 時間の境目、origin の違い(大文字小文字を無視)、origin を読めない、書けないときに消す、消せないとき | `TestMergeGuarantee_SaveThenReadRoundTrips`、`_SaveWritesAnUnknownToo`、`_SaveWithoutALedgerDirCreatesNothing`、`_WriteFailureRemovesTheOldCache`、`_RemovalFailureIsReported`、`_ReadWithoutAUsableCacheIsUnknown`(8 行)、`_ReadMakesAnOldCacheStale`(6 行)、`_ReadComparesOriginWithTheCache`(10 行)、`_ReadBoundsGitWithItsTimeout`、`_ForDefaultBranch`、`_DisplayJSONShape` | V01〜V05、R01〜R11、R13、R14 red |
| Integration: doctor(偽の seam と一時の台帳、台帳のある・なし) | `TestCheckMergeGuarantee_*`(4 本)、`TestMergeGuaranteeResult_EverySaveOutcome`(15 件)、`TestMergeGuaranteeEnvFromOS_RealCommandsAndTheResolvedStateDir`、`TestRunDoctorFull_PrintsTheMergeGuarantee`(pass、warn、info のどれでも「All checks passed.」)、`TestRunDoctor_TestMainPinRunsNoGitOrGh` | K01〜K06 red |
| Integration: status(台帳なし、キャッシュあり・なし・壊れた・古い、org あり・なし、origin が違う、repo の外、`--json`) | `status_merge_guarantee_test.go` の 8 本(本物の git と一時の repo。origin の向け直しは `git remote set-url`)、`TestStatusCmd_OrgWatchLineFollowsTheStateDirLine`(行の位置) | K10〜K13 red |
| Integration: merge-order(本物の git と bare の origin、既定のブランチが合う・違う、origin を別の repository に向け直す、`--json`) | CLI: `TestOrgMergeOrder_MergeGuaranteeLineFollowsTheDefaultBranchLine`(4 行)、`_NoBranchPrintsTheGuaranteeLineFirst`、`_MergeGuaranteeWithControlCharactersStaysOneLine`、`_NoLedgerPrintsTheNoBranchesNoteWithoutRunningGit`、`_WritesNothingAndTakesNoLock`。org: `TestMergeOrder_FillsTheMergeGuaranteeFromTheLedgersCache`(6 行)、`_NoTargetsReadsTheGuaranteeWithoutTheLsRemote`。`parseMergeOrderText` は保証の行がないと失敗するので、既存のテキストのテストはどれも行の位置を確かめる | MO1〜MO4、K20、K21 red |
| Regression: org/... と cli/... の全体を go1.25.8 と go1.26 で。既存の doctor・status・merge-order のテストが壊れない | 上の 2 回の全体の実行。既存のテストで期待を変えたのは 8 本(evidence の go の「Existing tests whose expectations this branch changed」)で、消したテストはない | - |
| Edge: 既定のブランチの名前に `/` | `TestMergeGuarantee_EscapesTheBranchAndRepositoryInAPIPaths`(`release/v1` は `release%2Fv1`、owner の `{owner}` も escape) | H05、H06 red |
| Edge: classic と ruleset の両方 | classic の表の最後の 3 行 | D03 red |
| Edge: `gh api` の 404 の本文の message が想定外 | なし(下の「Test gaps」の 1) | S03、S03b、S04 は生き残り |
| AC8: 本物の gh を打たない、repo の台帳に書かない | TestMain の `doctorMergeGuaranteeEnv` の固定、`TestRunDoctor_TestMainPinRunsNoGitOrGh`、`TestStatusCmd_MergeGuaranteeRunsNoGH`、偽の `gh` は 0 回 | 固定を外すと赤くなる(下) |

## Mutation

- 対象は判定(C4)、呼び出しの順、classic と strict の endpoint、host と引数(C7)、期限(C1)、キャッシュの書き込み(C2、C5)、読み込み(C5、C7)、merge-order の読み方、doctor、status、merge-order の CLI。ファイルごとに `mergeguarantee.go` 57 件、`mergeorder.go` 5 件、`doctor_merge_guarantee.go` 4 件、`doctor.go` 2 件、`status.go` 4 件、`org_merge_order.go` 2 件。対照 X0(手を加えないファイルを overlay で読ませたもの)は生き残った
- 74 件のうち 69 件が red(68 件は assert、S05 は nil の参照で panic)。生き残りは 5 件で、どれも scratch の probe の行(evidence の C と E)で落ちた。HEAD ではどの probe も通る
  - S03、S03b: strict の endpoint の 404 で、message が `Not Found` でも保護なしの 2 つでもないとき、strict を false と読む(`unknown` が `pr_checks_only` になる)。probe PB1(`Branch not found` の 404 で、最後の `return` の `unknown`)で落ちた。計画の Edge case にある入力の行がないため(Test gaps の 1)
  - S04: 同じ 404 を admin の権限の `unknown` と読む(種類は同じで detail だけ違う)。PB1 で落ちた
  - S07: body の status の 403 を読まない。stderr の `(HTTP 403)` が残るので、テストの 403 の 2 形のどちらでも同じ動きになる。本物の gh は 403 で stderr にいつも `(HTTP 403)` を出し、`realMergeOrderExec` はそれをエラーに付けるので、実際の入力では同じ動き。PB4(stderr に `(HTTP 403)` のない 403)で落ちた
  - D09: ruleset の必須の検査で context が空のものを数える。GitHub の答えで context が空のものは見ていないので、実際の入力ではおそらく同じ動き(未確認)。PB3 で落ちた
- AC8 の感度: `internal/cli/main_test.go` の `doctorMergeGuaranteeEnv` の固定を overlay で外し、`RALPH_ORG_STATE_DIR` に既にある scratch のディレクトリを渡して doctor のテストを打つと、`TestRunDoctor_TestMainPinRunsNoGitOrGh` が落ち、そのディレクトリに `merge-guarantee.json`(kind `unknown`)が書かれた。固定を戻すと何も書かれない。repo の台帳のディレクトリ(main のチェックアウトの `.harness/state/org`)はもともとないので、固定がなくても今回の実行では書かれなかった(C2 で、doctor はないディレクトリに書かない)。「repo の台帳にキャッシュがない」という確認だけでは固定の効き目は分からず、この感度の確認で補った

## 本物の gh での確認(読むだけ)

verifier の gap「merge queue で決まる経路は偽の seam だけ」のために、merge queue を使う公開の repository を探し、HEAD の binary で doctor、status、merge-order を打った(`git init` した空の repo に origin だけを足し、台帳は scratch。GitHub には何も書いていない)。

- zed-industries/zed(ruleset に `merge_queue`、strict でない必須の検査 2 件、classic の保護は無効): doctor は `pass — github_protection: ... requires its 2 required status checks ...: a ruleset has a merge queue, and rulesets require the checks`。status と merge-order は同じ種類を、キャッシュの `checked_at` と一緒に出した
- Homebrew/homebrew-core(`merge_queue`、strict でない必須の検査 4 件): 同じく `pass`、`a ruleset has a merge queue, and rulesets require the checks`
- astral-sh/uv(strict でない必須の検査 1 件、merge queue なし): `warn — pr_checks_only: ... rulesets require 1 required status check, but nothing requires branches to be up to date before merging with them; set ...`
- merge queue と classic の検査の組み合わせ(`rules.mergeQueue && classic > 0`)は、上の 3 つとも classic の保護が無効なので確かめられていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の status の行の並び(state-dir の行、org watch の行、roster) | 保たれている。保証の行は org watch の行のすぐ下に入る | `TestStatusCmd_OrgWatchLineFollowsTheStateDirLine`、`TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchLine` |
| 7c-1 の merge-order の分類と順、ls-remote を打たない早い return、何も書かない | 保たれている。早い return は、キャッシュがあれば local の get-url だけを打ち、ls-remote と gh は打たない | `TestMergeOrder_*` と `TestOrgMergeOrder_*` 全体、`TestMergeOrder_NoTargetsRunsNoGit`、`TestMergeOrder_WritesNothing`、`TestOrgMergeOrder_WritesNothingAndTakesNoLock` |
| `mergeRepoFromOriginURL` を `mergeOriginFromURL` に分けた(7c-1 の origin の照合) | 保たれている | 7c-1 の `TestMergeOrder_ReadsOwnerAndRepoFromOriginAndPRURLs`、`TestMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut`、`TestMergeOrder_LocalPathOriginSkipsTheRepositoryCheck`、`TestOrgMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut` が両方の版で PASS |
| 前に揺れた doctor と watcher のテスト | 落ちていない | evidence の go の「Tests that were flaky before」 |

## Test gaps

1. 計画の Edge case「`gh api` の 404 の本文の message が想定外(`unknown`)」の行がない(verifier の指摘を確認した)。strict の endpoint の失敗の行は 500 の `Server Error` だけで、`Not Found` でも保護なしの 2 つでもない 404 の行はない。HEAD は正しく最後の `return` の `unknown` を返す(probe PB1)が、これを `pr_checks_only` に変える変更(S03、S03b)をテストは止められない。GitHub は別の endpoint でこの形の 404 を返している(verifier が `branches/no-such%2Fbranch-xyz` で `Branch not found` を見た。strict の endpoint で返るかは未確認)。一番小さい足し方は、`TestMergeGuarantee_ClassicProtectionDecidesTheKind` の表に `mgAPIErrorReply("Branch not found", "404")` で detail まで比べる 1 行を足すこと(S04 も落ちる)
2. classic の strict true(`github_protection`)は偽の seam のテストだけ(verifier の指摘を確認した)。endpoint は admin の権限が要り、手元の gh はどの公開 repository でも admin でないので、実機では見られない
3. merge queue で決まる `github_protection` は、今回の本物の gh の確認で一部を埋めた(verifier の指摘の一部を覆した): ruleset の merge queue と ruleset の必須の検査の経路は zed-industries/zed と Homebrew/homebrew-core で `pass` になった。merge queue と classic の必須の検査の経路は、例になる公開の repository が見つからず、偽の seam のテストだけのまま
4. ruleset の必須の検査で context が空のもの(D09)の行がない。実際の答えではおそらく起きない(未確認)
5. 台帳のディレクトリがないときの doctor は `checkMergeGuarantee` の層(`TestCheckMergeGuarantee_NoLedgerDirIsNotCreated`)だけで確かめ、`runDoctorFull` を通した件はない。`runDoctorFull` は同じ関数を呼ぶだけなので影響は小さい。実機の証跡には台帳のない doctor がある
6. state dir の `os.Stat` が「ない」以外のエラーを返すとき(save は書けないとして古いキャッシュを消し、read は `unreadable`)のテストはない(coverage の 647-648、721-722)
7. 本物の git と gh を使う既定値(`Exec`、`LookPath` が nil)は、AC8 のとおりテストでは打たない。実機の証跡(`docs/evidence/merge-guarantee-livefire-2026-10-14.md`)と今回の本物の gh の確認だけが通す
8. verifier と self-review が未確認とした 2 つは今回も確かめていない: `protection/required_status_checks` が `%2F` を含む名前を受け付けるか(admin の権限が要る)、gh の stderr の制御文字。`mergeGuaranteeErrText` は空白(改行とタブを含む)を 1 つの空白にまとめ(`TestMergeGuarantee_LongErrorIsCutToOneLine`)、status と merge-order の行は `inboxField` で escape する(制御文字のテスト 2 本)が、doctor の行は Detail をそのまま出す(`doctor.go:208`)ので、空白でない制御文字(ESC など)が gh の stderr にあれば doctor の行にそのまま出る。テストはない

## Verdict

- Pass: yes。`./scripts/run-test.sh`(changed)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。`go test ./internal/cli/... ./internal/org/... -count=1` は go1.25.8 と go1.26.0 の両方で rc 0(cli 651/556、org 708/1,461 と skip 1、org/driver 48/89、org/protocol 15/56、名前は両方の版で同じ)。足した 44 本のテストはどれも PASS。`-race` も通った。テストは本物の gh を打たず(偽の `gh` は 0 回)、repo の台帳と worktree の台帳に `merge-guarantee.json` はない(`.harness/state/org` 自体がどちらにもない)
- Fail: なし
- Blocked: なし
- Verified: 計画の Test plan の Unit、Integration、Regression の項目と、Edge case の 3 つのうち 2 つ(上の対応表)。mutant 74 件のうち 69 件が red。merge queue と ruleset の検査で決まる `github_protection` を本物の gh で確かめた
- Not verified: Test gaps の 1〜8。どれも PR を止めるものではない。1 は計画の Edge case の 1 行が欠けたもので、HEAD の挙動は probe で正しいと確かめた

## Cycle 2

- Date: 2026-10-14(JST 21:53〜22:50)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 の最後)。self-review の cycle 2(432e5912、LOW 4 件、Merge: yes)と verify の cycle 2(a99faa87、pass、LOW 2 件)のあとに走らせた
- Scope: HEAD 034dec6f(base a16d7c4f)。cycle 2 でコードとテストを変えたコミットは 59abeaec だけで、`git diff fb954403 HEAD -- internal/` は `internal/org/mergeguarantee.go` と `internal/org/mergeguarantee_test.go` の 2 ファイル(+95/-23)。中身は、`mergeGuaranteeErrText` のエスケープ(空白類を畳み、1 文字ずつ `PrintableInboxText` に通し、そのあとで 300 バイトに切る)、`readMergeClassicStrict` の 404 `Not Found` と 403 を別の case に分けたこと、テストの 4 行と 2 本。ほかのコミットは計画の 1 行、登録簿の 244 行目、報告と insight event だけ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は cycle 1 と同じ(macOS arm64、go1.26.0 と go1.25.8、git 2.49.0、gh 2.102.0)。load average は 5〜29
- Evidence(どれもコミットする。wrapper が書いた `docs/evidence/verify-2026-10-14-125414.log` は gitignore の対象なのでコミットしない):
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-c2-run-test.txt`: `./scripts/run-test.sh`(changed)の出力を cycle 1 と同じ方法で抜いたもの、`(cached)` だった 4 パッケージの打ち直し、cycle 1 の行との diff
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-c2-go.txt`: go1.25.8 と go1.26.0 での `./internal/cli/... ./internal/org/...` の結果。パッケージごとの数、`-race`、SKIP の行、両方の版と cycle 1 との名前の diff、cycle 2 で足した・変えたテストの行、前に揺れたテストの行、関数ごとの coverage
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-c2-mutation.txt`: cycle 2 のコードの mutant 31 件と対照 X0、生き残りに当てた probe の行 PC1〜PC5、同じ動きの生き残りの理由、本物の GitHub の答え、cycle 1 の一覧 70 件を HEAD で流し直した結果、mutant の一覧と probe の source
  - `docs/evidence/test-2026-10-14-org-merge-guarantee-c2-ac8.txt`: 本物の gh に届かないこと(canary とその感度)、台帳に `merge-guarantee.json` がないこと、TestMain の固定を外したときの感度
- テストもコードも変えていない。mutant の写し、probe のテスト、偽の `gh`(canary)は scratchpad に置き、`go test -overlay` と PATH で読ませた。worktree に書いたのは、この節と上の evidence 4 本と insight event だけ。evidence は cycle 1 と同じく scratch の script が組み立てた生の出力で、`git diff --check` は空、U+FFFD は 0 個だった。mutation を終えて evidence を置く前の `git status --porcelain` は空で、main のチェックアウトの `git status --porcelain` も空。`~/.config/herdr` の下に 21:50 より新しいファイルはなく、`$TMPDIR` に `ralph-verify-*` は残っていない。herdr の server も osascript も起こしていない

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed、go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(cycle 1 と同じ) | 717 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/... ./internal/org/... -count=1 -v -coverprofile` | cli: top-level 651、subtest 556。org: 710、1,463。org/driver: 48、89。org/protocol: 15、56 | 同じ数から org の skip 1 を引いた数 | 0 | org 1 | 310 秒(cli 306.5 s、org 142.4 s)、rc 0 |
| `go test ./internal/cli/... ./internal/org/... -count=1 -v -coverprofile`(go1.26.0) | go1.25.8 と同じ | go1.25.8 と同じ | 0 | org 1 | 274 秒(cli 270.3 s、org 121.8 s)、rc 0 |
| うち cycle 2 で足した・変えたテスト(下の一覧) | top-level 2(新)、subtest 4(新 2、改名 2) | 両方の版ですべて | 0 | 0 | - |
| `(cached)` だった config、insights、scaffold、upgrade を `-count=1`(go1.26.0) | 4 パッケージ | 4 | 0 | - | 約 4 秒 |
| `go test -race`(org は `-run 'MergeGuarantee\|MergeOrder'`、cli は `-run 'MergeGuarantee\|OrgMergeOrder\|TestStatusCmd'`、go1.26.0) | 2 パッケージ | 2 | 0 | - | 66 秒(org 6.9 s、cli 48.9 s)、DATA RACE なし |
| mutation: cycle 2 のコード(`go test -overlay`、go1.26.0) | mutant 31 件と対照 X0 | red 19 件 | 生き残り 12 件(下の「Mutation(cycle 2)」) | build できないものはなし | 1,217 秒 |
| mutation: cycle 1 の一覧を HEAD で流し直し | 70 件(74 件のうち、元の行がなくなった S02、S04、S06、S07 を除く) | red 69 件 | 生き残り 1 件(D09、cycle 1 と同じ) | - | 1,366 秒 |
| probe(overlay だけの 5 本、コミットしない) | HEAD と生き残り 12 件 | HEAD ではすべて通る。閉じられる生き残り 8 件は、どれも狙った行で落ちる | - | - | - |
| 本物の gh(読むだけ) | strict の endpoint に 2 回 | 下の「本物の GitHub の答え」 | 0 | - | - |

- shell は 41 本すべてが `OK` で終わり、集計の `FAIL:` の行はどれも 0。抜き出した行から行番号、時刻、Go の `ok` の行、末尾の行を外して cycle 1 の evidence と diff を取ると空で、suite ごとの集計の行は 1 行も違わない。Skipped は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の 1 件
- `run-test.sh` の中の Go は `internal/cli`(189.0 s)、`internal/org`(84.9 s)、`internal/org/driver` を流し直し、config、insights、org/protocol、scaffold、upgrade は `(cached)` だった。`internal/insights` は `internal/org` を import する(`go list -deps`)ので、cached の結果に頼らず 4 つを `-count=1` で打ち直して ok。org/protocol は上の org/... の実行で uncached に通った
- cycle 1 と比べた数: org の top-level は 708 から 710(+2)、subtest は 1,461 から 1,463(+2)。cli、org/driver、org/protocol は同じ。名前の行(時間を除き `LC_ALL=C` で並べた 3,589 行)を cycle 1 と diff すると、違いは 59abeaec のテストの直しそのものだった: 403 の 2 行の改名(`strict needs admin access: 403 in the body`、`... on stderr only` から `strict answers 403: ...` へ)、足した 2 行(`strict answers 403: in the body only`、`strict answers a 404 with another message`)、足した 2 本。消えたテストはない
- go1.25.8 と go1.26.0 で、名前の行 3,589 行は同じだった
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どちらの版でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は 3.87 s と 4.51 s で、+5 s の subtest は出なかった
- org の Skipped は cycle 1 と同じ `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`

### cycle 2 で足した・変えたテスト

次のテストは、go1.25.8 と go1.26.0 の両方で PASS だった(evidence の go の「The tests cycle 2 added or changed」)。org の `-race` の実行は `-run 'MergeGuarantee|MergeOrder'` なので、どれも race detector の下でも通っている。

- 403 の 3 行(`TestMergeGuarantee_ClassicProtectionDecidesTheKind` の表): `strict answers 403: in the body`(本文の status と stderr の両方が 403)、`strict answers 403: on stderr only`(stdout が空)、`strict answers 403: in the body only`(新しい行。stderr は `exit status 1` だけ)。最初の 2 行は cycle 1 の行の改名で、期待の Detail を 403 の文(admin のないことを原因の 1 つとしか言わない文)に変えた
- 404 の行: `strict answers a 404 with another message`(新しい行。`Branch not found` の 404 は最後の `return` の一般の `unknown`)
- エスケープと切る順: `TestMergeGuarantee_ErrorIsCutAfterEscaping`(新しい本。`a` と ESC 100 個は 297 バイトで切って `...`、ESC 75 個のちょうど 300 バイトは切らない)
- 制御文字のエスケープ: `TestMergeGuarantee_ControlCharactersInAnErrorAreEscaped`(新しい本。ESC、BEL、NUL、DEL、C1 の `\u009b` を Go の escape で書く。最後の確かめが、キャッシュから読んだ行を `PrintableInboxText` にもう一度通しても同じ文になること)
- 関数が変わったので見直した既存の行: `TestMergeGuarantee_LongErrorIsCutToOneLine`、`strict needs admin access: 404 Not Found`、`ruleset checks and a strict that needs admin access`、CLI の `TestOrgMergeOrder_MergeGuaranteeWithControlCharactersStaysOneLine` と `TestStatusCmd_MergeGuaranteeFromACacheWithControlCharactersStaysOneLine`。どれも PASS

### Coverage(cycle 2)

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.3%、`internal/cli` 86.9%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%。どれも cycle 1 と同じ
- `internal/org/mergeguarantee.go` は 254/261 文(97.3%、cycle 1 は 253/260)。増えた 1 文は 403 の case の `return` で、通っている。通らない 7 文は cycle 1 と同じもので、`Now`、`Exec`、`LookPath` が nil のときの本物の既定値(175、186、189、755、778 行)と、state dir の `os.Stat` が「ない」以外のエラーを返したときの save(661 行)と read(735 行)
- cycle 2 で変えた `readMergeClassicStrict` と `mergeGuaranteeErrText` は、どちらも 100%。それでも下の mutant が 8 件、閉じられる形で生き残った。穴は通らない行ではなく、呼び出し箇所ごとの入力と、テストの値が区別しない数にある
- Branch: 測っていない(cycle 1 と同じ)。代わりに mutation で分岐ごとの判定を確かめた

### cycle 1 の Gap 1 は閉じたか

cycle 1 で生き残った S03、S03b、S04、S07 は、`readMergeClassicStrict` の `switch` が変わったので、同じ変更を HEAD の行に書き直して打った。4 件とも red になり、どれも cycle 2 で足した行だけが落とした。

| cycle 1 の mutant | 変更 | HEAD での id | 落とした行 |
| --- | --- | --- | --- |
| S03 | strict の endpoint の、message が想定外の 404(本文の status)を strict false と読む(`unknown` が `pr_checks_only` になる) | G03 | `strict answers a 404 with another message` |
| S03b | 同じ変更を stderr の `(HTTP 404)` で | G03b | 同じ行 |
| S04 | どの 404 も admin の文の `unknown` にする | G04 | 同じ行 |
| S07 | 本文の status の 403 を読まない | G07 | `strict answers 403: in the body only` |

- cycle 1 の一覧を HEAD で流し直すと、元の行がまだある S03 と S03b はそのまま red になった。元の行がなくなった S02、S04、S06、S07 は、上の G04、G07 と下の F06、F04 として書き直し、4 件とも red
- cycle 1 で red だった 69 件(R14 は cycle 1 で書き直した形)は、HEAD のテストでも全部 red のままだった。生き残りは cycle 1 と同じ D09 だけで、cycle 2 の fix の範囲外(Gap 4)。HEAD の行に書き直した G09 も生き残り、cycle 1 の probe PB3 で落ちることは変わらない

### Mutation(cycle 2)

cycle 2 の差分の周りに 31 件の mutant を作った(`mergeguarantee.go` だけ)。19 件が red、12 件が生き残った。対照の X0(手を加えないファイルを overlay で読ませたもの)は生き残った。

| 対象 | mutant | red | 生き残り |
| --- | --- | --- | --- |
| cycle 1 の Gap の書き直し(G03、G03b、G04、G07、G09) | 5 | 4 | G09 |
| 403 と 404 の分け方(F01、F02、F04〜F09、F08b) | 9 | 5 | F01、F05、F08、F08b |
| `mergeGuaranteeErrText`(E01〜E10) | 10 | 9 | E06 |
| 呼び出し箇所ごとに、Detail に `mergeGuaranteeErrText(err)` でなく `err.Error()` を入れる(P296、P353、P466、P471、P474、P575、P794) | 7 | 1(P296) | 6 |

- `mergeGuaranteeErrText` の壊し方は、同じ動きの E06 を除いてすべて red だった。直す前の順(生で切ってから escape、E01)と、`>` を `>=` にするもの(E03)、escape して書いてから 300 バイトで切るもの(E07)、escape の長さを 1 バイトと数えるもの(E08)は `TestMergeGuarantee_ErrorIsCutAfterEscaping` が落とした。escape を外すもの(E05、L3 を戻す)と C0 だけを escape するもの(E10、DEL と `\u009b` が生で残る)は `TestMergeGuarantee_ControlCharactersInAnErrorAreEscaped` が落とした。空白を畳まないもの(E02、E09)は `TestMergeGuarantee_LongErrorIsCutToOneLine` と `TestMergeGuarantee_FailuresAreUnknownWithAReason` が落とした
- 403 と 404 の分け方は、L4 を戻すもの(F02、403 にまた admin の文を出す)が 403 の 3 行で、stderr の `(HTTP 403)` を読まないもの(F04)が `on stderr only` で、`Not Found` の case を外すもの(F06)が `404 Not Found` の行で、403 を strict false と読むもの(F07)と 403 の Detail から branch を外すもの(F09)が 403 の行で、それぞれ red になった

生き残り 12 件のうち 8 件は、HEAD で通る probe の行が落とす(evidence の mutation の B)。確信が増える順に挙げる。

- P353、P466、P471、P474、P575、P794(PC3 の 5 行と PC4): 呼び出し箇所 1 つの Detail を `err.Error()` で作る。`mergeGuaranteeErrText` を呼ぶ非 test の行は 12 か所あるが、制御文字や改行を持つエラーを渡すテストは repository の読み(296 行。P296 は red で、E02・E09 を落とした `gh fails on an ssh alias host` の行もここ)の 1 か所だけだった。mutant を作った 6 か所(353、466、471、474、575、794 行)は、生の `err.Error()` にしてもどのテストも変わらない。残りの 5 か所(218、261、308、362、578 行)には mutant を作っていない。今は 12 か所すべてがこの関数を通る(verify の grep)。テストが止めないのは、次に誰かが 1 か所を生の `err.Error()` に戻したときや、新しい gh の手順を足したとき。PC3 は gh と git の各手順(get-url、repository、rules、branch protection、strict の 404 `Not Found`・403・ほかの失敗)に ESC と BEL と改行を持つエラーを渡して、Detail に生の制御文字がなく `\x1b[31m...\a second line` があることを確かめる表で、PC4 は cache を読む側の get-url の失敗(794 行)を同じように確かめる
- F08、F08b(PC2、PC1): 403 の Detail と 404 `Not Found` の Detail が、必須の検査の数を 1 と書く。admin と 403 の行はどれも必須の検査が 1 つ(`mgClassicRequires("ci")`)なので、`mergeChecksText(checks)` を `mergeChecksText(1)` にしても分からない。PC1 と PC2 は検査を 2 つ(`ci`、`lint`)にして「requires 2 required status checks」を確かめる。表に 1 行ずつ足せば閉じる

残りの 4 件のうち 3 件は、本物の入力では動きが変わらない。

- F01(403 の case を `Not Found` の case より前に置く): 違いが出るのは、本文の message が `Not Found` で、本文の status か stderr が 403 を言う答えだけ。GitHub の 404 の本文は `"status":"404"` だった(下)
- F05(本文の status を文字列の `"403"` とだけ比べる): 違いが出るのは status が数の答えだけ。GitHub は status を文字列で書いた(下)
- E06(`PrintableInboxText` の `keepLineBreaks` を true にする): `strings.Fields` が先に `\n` と `\t` をすべて外しているので、渡る文字に改行もタブもない

最後の G09 は cycle 1 の D09 で、Gap 4 のまま残る(下の Test gaps)。

PC5 は mutant を落とすためでなく、HEAD の動きを記録するための行で、HEAD で通った: escape 済みの文(`already \x1b escaped`)は変わらない、不正な UTF-8 のバイトは U+FFFD になる(キャッシュの JSON が作る文字と同じ)、NEL(U+0085、空白でも制御文字でもある)は escape されずに 1 つの空白に畳まれる、ESC・NUL・`\u009b`・U+200B・U+2028 を escape した文は `PrintableInboxText` にもう一度通しても変わらない。

### 本物の gh に届かないこと、台帳に書かないこと(AC8)

- 呼ばれたら記録して exit 97 で終わる偽の `gh`(canary)を、受け継ぐ PATH の先頭に置いて、`run-test.sh`、2 回の全体の実行、`-race`、mutant と probe のすべての実行、`(cached)` の打ち直し、下の固定を外す確認を走らせた。canary の記録は 0 件
- canary の感度: scratch のテスト 1 本(overlay で足しただけ)で、`CheckMergeGuarantee` を本物の既定値(`Exec` と `LookPath` が nil)のまま worktree に向けて呼んだ。canary の写しを PATH の先頭に置くと、canary は 1 回呼ばれ(`api --hostname github.com repos/yoshpy-dev/ralph`)、Detail は `exit status 97` を言った。受け継ぐ PATH を通って本物の gh に届くテストがあれば、上の記録に残る。cycle 1 の固定を外す確認では、doctor のテストが `PATH` を自分の stub のディレクトリだけにする(`doctorMGProject` の `t.Setenv("PATH", binDir)`)ので git も見つからず、canary の感度はそこからは分からなかった。この計画で canary が実際に呼ばれることを示したのは、この回が初めて
- repo の台帳(main のチェックアウトの `.harness/state/org`)と worktree の台帳はどちらもなく、`.harness` の下に `merge-guarantee*` は 0 個。cycle 1 と同じく、ディレクトリがないのでこれだけでは固定の効き目は分からない
- 固定の感度(`internal/cli/main_test.go` は cycle 1 から変わっていない): cycle 1 の overlay で TestMain の `doctorMergeGuaranteeEnv` の固定を外し、`RALPH_ORG_STATE_DIR` に既にある scratch のディレクトリを渡すと、`TestRunDoctor_TestMainPinRunsNoGitOrGh` が落ち、そのディレクトリに `merge-guarantee.json`(kind `unknown`)が書かれた。固定があると rc 0 で、何も書かれない。cycle 1 と同じ結果

### 本物の GitHub の答え(読むだけ)

F01 と F05 が同じ動きかどうかを決めるために、admin でない token で strict の endpoint を 2 回読んだ(GitHub には何も書いていない)。

- `gh api --hostname github.com repos/cli/cli/branches/trunk/protection/required_status_checks` は rc 1 で、stdout は `{"message":"Not Found","documentation_url":"...","status":"404"}`、stderr は `gh: Not Found (HTTP 404)`。status は文字列だった
- ないブランチ(`no-such-branch-xyz-c2`)の同じ endpoint も、同じ `Not Found` の 404 を返した。admin でない読みでは、cycle 2 で足した行の `Branch not found` の 404 は返らなかった。admin のある読みで返るかと、この endpoint の 403 は見ていない(self-review C2-2 と verify が挙げたとおり)

### Failure analysis(cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

### Regression checks(cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の Gap 1(S03、S03b、S04、S07) | 4 件とも red | 上の「cycle 1 の Gap 1 は閉じたか」 |
| cycle 1 で red だった mutant | HEAD のテストでも red | 70 件のうち 69 件。生き残りは cycle 1 と同じ D09 |
| 403 と 404 を分けても種類は変わらない(どちらも `unknown`、`Branch not protected` などは strict false) | 変わっていない | 表の 404 `Not Found`、403 の 3 行、`Branch not protected`、`Required status checks not enabled` が PASS。F07(403 を strict false と読む)が red |
| 長いエラーを rune の途中で切らない(cycle 1 の `utf8.RuneStart` の処理を 1 文字ずつの読みに替えた) | 保たれている | `TestMergeGuarantee_LongErrorIsCutToOneLine`(`é` 400 個)が PASS。E07(書いてから 300 バイトで切る)が red |
| shell の suite | 変わっていない | cycle 1 の行との diff が空 |
| Go 1.25 だけで起きる違い | 差はない | go1.25.8 と go1.26.0 で数と名前が同じ |
| 前に揺れたテスト | 揺れなかった | 2 回の全体の実行のどちらでも PASS |

### Test gaps(cycle 2)

cycle 1 の Test gaps の番号のまま、今の状態を書く。

| cycle 1 の Gap | 今 | 根拠 |
| --- | --- | --- |
| 1. strict の endpoint の、message が想定外の 404(と mutation の S07) | 閉じた | `strict answers a 404 with another message` と `strict answers 403: in the body only`。G03、G03b、G04、G07 が red |
| 2. classic の strict true は偽の seam だけ | 残る | endpoint に admin の権限が要る |
| 3. merge queue と classic の必須の検査の組み合わせは偽の seam だけ | 残る | 例になる公開の repository がない |
| 4. ruleset の必須の検査で context が空のもの(D09) | 残る(登録簿 244 行目の (g)) | G09 が生き残り、cycle 1 の PB3 で落ちる |
| 5. 台帳のない doctor を `runDoctorFull` で通した件がない | 残る | cycle 2 は doctor を変えていない |
| 6. state dir の `os.Stat` が「ない」以外のエラーを返すとき | 残る(登録簿 244 行目の (g)) | coverage の 661 行と 735 行 |
| 7. 本物の git と gh を使う既定値 | 残る(AC8 のとおりテストでは打たない) | canary の感度の確認で 1 回だけ通した(canary の gh、コミットしない) |
| 8. `%2F` と gh の stderr の制御文字 | 制御文字は閉じた。`%2F` は残る(登録簿 244 行目の (g)) | `TestMergeGuarantee_ControlCharactersInAnErrorAreEscaped`。verify の cycle 2 は HEAD の binary と偽の gh で doctor と status の行が `\x1b[31m...` になることを確かめた(verify の c2 の evidence の mode=esc) |

cycle 2 で新しく見つけた穴。どれもテストの失敗ではない。確信が増える順に挙げる。

9. 呼び出し箇所ごとのエスケープ: mutant で確かめた 7 か所のうち、制御文字や改行を持つエラーでテストを通るのは repository の読み(296 行)だけで、rules(353 行)、branch protection(575 行)、strict の 404 `Not Found`・403・ほかの失敗(466、471、474 行)、cache を読む側の get-url(794 行)は、`err.Error()` に戻しても落ちるテストがない(P353〜P794)。閉じる足し方は PC3 の表(7 行)と PC4 の 1 本で、どちらも HEAD で通る
10. 403 と 404 `Not Found` の Detail の検査の数: admin と 403 の行の必須の検査はどれも 1 つなので、数を 1 に決め打ちしても分からない(F08、F08b)。表の 2 行の検査を 2 つにすれば閉じる(PC1、PC2)
11. 本物の GitHub の strict の endpoint の答えのうち、cycle 2 で足した 2 行の入力(403 と `Branch not found` の 404)は見ていない。admin でない読みでは、ないブランチでも `Not Found` だった(上)
12. PC5 の動き(不正な UTF-8 が U+FFFD になる、NEL が空白に畳まれる、2 回目の escape で変わらない)にテストがない。害はない(self-review の cycle 2 も不正な UTF-8 について同じ判断)
13. self-review C2-1 の出どころ(cache の書き込みと削除の OS のエラー、state dir の path、scp 形の origin の URL)は、doctor の Detail に escape されずに入りうる。テストもない。直すかどうかは C2-1 の扱いに従う

### Tech debt(登録簿 244 行目に足す候補)

足すかどうかと文は /sync-docs が決める。244 行目の (g) は「cases with no test」を並べているので、そこへ足すのが近い。

- 9 の呼び出し箇所ごとのエスケープ(gh と git の各手順に制御文字のエラーを渡す表がない)。閉じる足し方: PC3 と PC4(`docs/evidence/test-2026-10-14-org-merge-guarantee-c2-mutation.txt` の F に source)
- 10 の 403 と 404 `Not Found` の Detail の検査の数。閉じる足し方: 表の 2 行の検査を 2 つにする(PC1、PC2)
- 2 と 3(classic の strict true と、merge queue と classic の必須の検査の組み合わせ)は偽の seam でしか確かめていない、という記録。今の登録簿にない
- 11(strict の endpoint の 403 と `Branch not found` の 404 は実物を見ていない)。self-review C2-2 と同じ件なので、C2-2 を足すならその行に合わせてよい
- 5 と 12 は小さく、足さなくてよい

### Verdict(cycle 2)

- Verdict: pass(cycle 2 の HEAD 034dec6f の時点)
- Pass: `./scripts/run-test.sh`(changed)は rc 0 で、shell 41 本すべて OK、Go 8 パッケージすべて ok、shell の集計の行は cycle 1 と同じ。`go test ./internal/cli/... ./internal/org/... -count=1` は go1.25.8 と go1.26.0 の両方で rc 0(cli 651/556、org 710/1,463 と skip 1、org/driver 48/89、org/protocol 15/56、名前は両方の版で同じ)。cycle 2 で足した・変えたテスト(403 の 3 行、`Branch not found` の 404 の行、`TestMergeGuarantee_ErrorIsCutAfterEscaping`、`TestMergeGuarantee_ControlCharactersInAnErrorAreEscaped`)はどちらの版でも PASS。`-race` も ok。`(cached)` だった 4 パッケージも打ち直して ok
- Pass(mutation): cycle 1 の Gap 1 の生き残り S03、S03b、S04、S07 は HEAD で 4 件とも red。cycle 2 の差分の周りの mutant 31 件のうち 19 件が red で、生き残り 12 件のうち 8 件は probe の行で落ちることを確かめ、3 件は本物の入力では動きが変わらず、1 件(G09)は fix の範囲外の Gap 4。cycle 1 の一覧も HEAD のテストで 70 件のうち 69 件が red
- Pass(AC8): テストは本物の gh を打たない(canary 0 件、本物の既定値で呼ぶと canary が 1 回呼ばれる感度も確かめた)。repo の台帳にも worktree の台帳にも `merge-guarantee.json` はなく、TestMain の固定を外すとキャッシュが書かれてテストが落ちる
- Fail: 0
- Blocked: なし
- Verified: 上の Test execution の全行、cycle 1 の Gap 1 と Gap 8 の前半が閉じたこと、cycle 1 の mutant の殺す力が落ちていないこと、本物の GitHub の strict の endpoint が status を文字列で返すこと
- Not verified: Test gaps の 2〜7、8 の `%2F`、9〜13。どれも PR を止めるものではない
