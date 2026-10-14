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
