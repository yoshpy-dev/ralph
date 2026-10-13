# Test report: org-merge-order

- Date: 2026-10-14(JST 05:29〜06:20)
- Plan: docs/plans/active/2026-10-14-org-merge-order.md(`Status: Approved`。`./scripts/plan-visual.sh digest` の出力 c0b1e9230999 は `- Approved:` 行と一致する)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1)
- Scope: HEAD a8882d6d(base ac34ccf3 = origin/main、8 コミット、19 ファイル、+4,219/-66)。/verify が見た 346304fc から HEAD までに増えたのは verify の報告と evidence の 2 ファイルだけで、コードとテストは /verify のときと同じ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、12 コア)、go 1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0、`/bin/sh` は bash 3.2.57
- Evidence(どれもコミットする。wrapper が書いた `docs/evidence/verify-2026-10-13-202931.log` は gitignore の対象なのでコミットしない):
  - `docs/evidence/test-2026-10-14-org-merge-order-run-test.txt`: `./scripts/run-test.sh`(changed)の出力から、suite ごとの集計の行と Go の行を抜いたもの(抜き方は先頭に書いた)。`(cached)` だった 6 パッケージを `-count=1` で打ち直した結果も末尾にある
  - `docs/evidence/test-2026-10-14-org-merge-order-go1258.txt`: `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -count=1 -v -coverprofile`。パッケージの結果、FAIL と SKIP の行、merge-order のテスト 37 本と subtest 37 本の `--- PASS` の行、偽の `gh`(canary)の結果、新しい 2 ファイルの関数ごとの coverage
  - `docs/evidence/test-2026-10-14-org-merge-order-mutation.txt`: mutant 70 件(build できたのは 66 件)の結果、生き残り 8 件の分類、生き残りを落とす probe の行 P1〜P5 の結果、PATH の漏れの probe
  - `docs/evidence/test-2026-10-14-org-merge-order-real-origin.txt`: 本物の GitHub の origin と本物の `gh` に対する読むだけの確認(ls-remote、`gh pr list`、一時の台帳で HEAD の binary を打った出力)
  - insight event: `docs/insights/events/2026-10-13-org-merge-order.jsonl` に test の 1 行(`--cycle auto`)を足した。指示どおり、このファイルはコミットしない
- テストもコードも変えていない。mutant の写し、probe のテスト、偽の `gh` は scratchpad に置き、`go test -overlay` で読ませた。worktree に書いたのは、この報告と上の evidence 4 本と insight event だけ。テストの前後で `~/.config/herdr` の下に 05:28 より新しいファイルはなく、main のチェックアウトの `git status --short` は空のまま、`$TMPDIR` に `ralph-verify-*` は残っていない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed、go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 13 分 50 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -count=1 -v -coverprofile` | org: top-level 670、subtest 1,329。cli: top-level 630、subtest 517 | org: top-level 669、subtest 1,329。cli: top-level 630、subtest 517 | 0 | org 1 | 4 分 6 秒(org 109.9 s、cli 242.8 s)、rc 0 |
| うち merge-order のテスト(`TestMergeOrder_*` 24 本、`TestOrgMergeOrder_*` 13 本) | top-level 37、subtest 37 | 37、37 | 0 | 0 | org の 24 本は合わせて約 4.5 s、cli の 13 本は合わせて約 33 s |
| `go test`(go1.26.0)で `(cached)` だった 6 パッケージを `-count=1` | config、insights、org/driver、org/protocol、scaffold、upgrade | 6 パッケージすべて ok | 0 | - | 合わせて約 10 s |
| mutation(`go test -overlay`、go1.26.0) | build できた mutant 66 件 | red 58 件 | 生き残り 8 件(下の「Mutation」) | build できない 4 件は書き直して 4 件とも red | 約 19 分 |
| 本物の origin と `gh` の確認(読むだけ) | ls-remote 1 回、`gh pr list` 2 回、HEAD の binary 3 回 | すべて期待どおり | 0 | - | - |

- shell は 41 本すべてが `OK` で終わり、集計の `FAIL:` の行 25 本はどれも `FAIL: 0` だった。Skipped は前から同じ `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell を変えていないが、`scripts/verify.local.sh` がいつも shell の suite を流す
- `run-test.sh` の中の Go は `internal/cli`(225.0 s)と `internal/org`(84.8 s)を流し直し、ほかの 6 パッケージは `(cached)` だった。`go list -deps` で `internal/org` を import するのは `internal/insights` だけなので、6 つとも `-count=1` で打ち直して ok を確かめた
- go1.25.8 の org の Skipped は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin では飛ばす作りで、この差分と関係ない)
- 差分で足した `func Test` は 37 本(`git diff ac34ccf3...HEAD -- '*_test.go'` の `+func Test`)で、どれも go1.25.8 と go1.26.0 の両方で PASS だった。消えた `func Test` は 0
- 前に揺れた doctor のテスト(`TestRunDoctor*`)と `TestRunWatcher_TimeoutIndependentOfSmallInterval` は、どちらの実行でも落ちなかった。`run-test.sh` の途中で load average が 73 まで上がった(`tests/test-pre-bash-guard.sh` と、他のプロセスの `mds_stores` など)が、落ちたテストはない

## Coverage

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 93.9%、`internal/cli` 86.8%
- 新しい 2 ファイル: `internal/org/mergeorder.go` 212/213 文(99.5%)、`internal/cli/org_merge_order.go` 39/43 文(90.7%)
  - `mergeorder.go` で通らない 1 文は、`Exec` が nil のときの `run = realMergeOrderExec`(`mergeorder.go:205-207`)。org のテストはいつも偽の `Exec` を渡す。CLI のテストはこの行を通るが、`-coverprofile` は各パッケージ自身のテストしか数えない
  - `org_merge_order.go` で通らないのは、`guardLegacyOrgStateDir` のエラーを返す行(92-94)、`newOrgRuntimeAt` のエラーを返す行(96-98)、`cmd.Context()` が nil のとき(100-102)、`mergeOrderShortSHA` の 12 文字以下の入力を返す行(177)
- Function: `mergeorder.go` の 21 関数のうち 20 が 100%(`MergeOrder` は 96.2%)。`printMergeOrder` は 100%、`newOrgMergeOrderCmd` は 87.0%、`mergeOrderShortSHA` は 66.7%
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

## Test plan coverage map

計画の Test plan の項目ごとに、それを確かめるテストを挙げる。「mutant」の列は、その項目の判定を壊した mutant をテストが落としたかどうか。

| Test plan の項目 | テスト(`mergeorder_test.go` / `org_merge_order_test.go`) | mutant |
| --- | --- | --- |
| 分類ごとに 1 本以上: ready、rebase(main が進んだ、既定のブランチを変えた)、wrong_base、regate、no_remote | `TestMergeOrder_ClassifiesAGatePassedAgainstOrigin`(12 行)、`TestMergeOrder_DefaultBranchComesFromOriginsHEAD`。CLI: `TestOrgMergeOrder_TextListsEachClassInMergeOrderWithTheNotes`、`TestOrgMergeOrder_GateOnTheOldDefaultBranchIsRebaseAfterOriginHEADMoves`、`TestOrgMergeOrder_PRIntoAnotherBranchThanTheDefaultIsWrongBase` | M39b〜M44、M46 すべて red |
| 分類: failed、no_gate、no_remote | `TestMergeOrder_FailedNoGateAndNoRemote`。CLI: Text のテスト | M38、M39b red |
| 分類: unreadable | `TestMergeOrder_LatestGateRecordWins`、`TestMergeOrder_TrailingLineWithoutNewlineIsCorrupt` | M01、M37、M45 red |
| 分類: merged、closed | `TestMergeOrder_OrdersByClassThenGatePosition`、`TestMergeOrder_MergedOrClosedComesFirst`。CLI: `TestOrgMergeOrder_AllListsMergedAndClosedPRsTheDefaultLeavesOut`、`TestOrgMergeOrder_OnlyMergedPRsPrintsNoBranchToList` | M33、M35 red。M36(state の大文字小文字)は生き残り(本物の `gh` は大文字なので同じ動き) |
| 最新の記録を読む: 通ったあとに落ちたら failed、落ちたあとに通ったら ready、通ったあとの記録が読めなければ unreadable、通ったあとに壊れた行があれば unreadable、壊れた行が最後の記録より前なら影響しない | `TestMergeOrder_LatestGateRecordWins`(11 行) | M01、M02、M11、M37 red。M03(壊れた行の最初を見る)は生き残り: 壊れた行が記録の前と後の両方にある行がない |
| 読めない記録の付け先: 打った org のその時点の結びつき、前に結びつきがない記録はどこにも付かず注記だけ | `TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` | M04、M06 red。M05(org の最初の結びつきに付ける)は生き残り: 記録より前の結びつきが 1 つしかない |
| ls-remote の symref: HEAD の symref がない出力はエラー、`refs/heads/HEAD` を既定と読み違えない | `TestMergeOrder_ReadsTheLsRemoteListing`、`TestMergeOrder_LsRemoteProblemsClassifyNothing`(8 行)。CLI: `TestOrgMergeOrder_OriginStateUnknownExitsOneWithTheReason` | M12、M14〜M16、M17b red。M13(最後の symref の行を採る)は生き残り(git は HEAD の symref を 1 行しか出さないので同じ動き) |
| 対象: disband 済みの org、同じ org_id で 2 つの機能、同じブランチに 2 つの結びつき(補足は最新)、dry-run・読めない記録・完全でない結びつきを数えない | `TestMergeOrder_TargetsAreTheBranchesOfEveryBinding`、`TestMergeOrder_SameBranchBoundTwiceShowsTheLatestBinding`、`TestMergeOrder_DryRunSeatAndIncompleteRecordsDoNotCount`、`TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` | M06〜M08 red。M09(org_id のない行を数える)は生き残り |
| 順: ready の中が gate の時刻の順、分類の順、`--all` の有無 | `TestMergeOrder_OrdersByClassThenGatePosition`、`TestMergeOrder_MergedOrClosedComesFirst`。CLI: Text と `--all` のテスト | M10、M30〜M35 red。C08 red |
| PR の選び方: 番号が最大、`isCrossRepository` を外す、`gh` がない、失敗する、JSON が読めない | `TestMergeOrder_PicksTheHighestSameRepoPR`、`TestMergeOrder_GHMissingNotesAndIsNotRunAgain`、`TestMergeOrder_GHFailureForOneBranchLeavesItsPROut`。CLI: `--all` のテスト(fork の #9 を外す) | M23b〜M29 red |
| ls-remote: 出力の読み方(`/` を含む名前、空の出力)、失敗と timeout はエラー | `TestMergeOrder_ReadsTheLsRemoteListing`、`TestMergeOrder_LsRemoteProblemsClassifyNothing`、`TestMergeOrder_RealExecIsNotHeldByAChildHoldingThePipe`。CLI: `TestOrgMergeOrder_BranchNameWithTwoSlashesIsListedWhole` | M15、M18〜M21、M52 red |
| Integration: 本物の git と bare の origin、偽の `gh`、テキストと `--json`、終了コード、`--all` | `org_merge_order_test.go` の 13 本 | C01〜C05、C08〜C11、M51b red。C06、C07、C12 は生き残り(下) |
| Regression: `go test ./internal/org/... ./internal/cli/...` 全体、`gateRecordFromDetails` を変えない | 上の 2 回の全体の実行。`git diff ac34ccf3...HEAD -- internal/org/gate.go internal/org/gate_run.go internal/org/reserve.go internal/org/manifest.go` は空。gate のテスト 39 本(`TestGateRecordDetails_RoundTrip` を含む)は go1.25.8 で PASS | - |

AC との対応: AC1・AC11 は上の 1 行目、AC2 は 2 行目と「最新の記録を読む」、AC3 は「対象」、AC4 は unreadable と「読めない記録の付け先」と「対象」、AC5 は「PR の選び方」と merged・closed、AC6 は「順」と Text のテストの注記、AC7 は ls-remote の 2 行と `TestMergeOrder_ManifestThatCannotBeReadIsAnError`、AC8 は `TestMergeOrder_ResultJSONShape` と `TestOrgMergeOrder_JSONListsTheSameBranchesInTheSameOrderAsTheText`、AC9 は `TestMergeOrder_WritesNothing` と `TestOrgMergeOrder_WritesNothingAndTakesNoLock`。AC10(文書)は /verify の担当でテストの対象ではない。

### 計画の Edge cases

| Edge case | テスト | 結果 |
| --- | --- | --- |
| 結びつきはあるが一度もリモートに push していない | unit の `feat/none`(gate なし → no_gate)と `feat/gone`(gate あり → no_remote)、CLI の `feat/no-remote` | 覆う |
| gate の `base_ref` が `refs/heads/<名前>` | `TestMergeOrder_ClassifiesAGatePassedAgainstOrigin` の「ready: a base_ref under refs/heads」 | unit だけで覆う(M44 red) |
| 既定のブランチの名前が main 以外 | `TestMergeOrder_DefaultBranchComesFromOriginsHEAD`、CLI の trunk のテスト | 覆う |
| 既定のブランチを変えて古いブランチが残る | AC11 の unit 3 行、CLI の trunk のテスト | 覆う |
| PR の統合先が既定でない | wrong_base の unit 2 行、CLI の wrong_base のテスト | 覆う |
| ブランチ名に `/` が 2 つ以上 | unit の `feat/a/b/c`、CLI の `feat/x/y` | 覆う |
| manifest が空 | unit の「an empty manifest」(空行 2 つ)、CLI の空のファイル | 覆う |
| 台帳がない | unit の「no manifest」、CLI の state dir がない場合(state dir を作らないことも確かめる) | 覆う |
| manifest の最後の行が改行のない書きかけ | `TestMergeOrder_TrailingLineWithoutNewlineIsCorrupt` | unit だけで覆う |

計画の Edge cases はすべて、少なくとも unit のテストで覆われている。覆われていない場合は下の「Test gaps」にまとめた。

### 本物の `gh` に届かないこと

- Go のコードで `gh` を打つのは `internal/org/mergeorder.go:478` だけ(テスト以外の `.go` を `"gh"` で grep)
- org のテストは、`realMergeOrderExec` のテスト 2 本(`sh` だけを打つ)のほかは、どれも偽の `MergeOrderExec` を渡す
- CLI のテスト 13 本は、どれも PATH を一時のディレクトリだけにする(`setMergeOrderPATH` は本物の git を絶対パスで呼ぶ wrapper と、必要なら偽の `gh` だけを置く。`writeMergeOrderCallLoggers` は呼ばれたら記録して失敗する git と gh だけを置く)
- 実際に確かめた: 呼ばれたら記録して exit 97 で終わる偽の `gh`(canary)を、受け継ぐ PATH の先頭に置いて、go1.25.8 の org と cli の全体と、go1.26.0 の `-run 'OrgMergeOrder'` を走らせた。canary の記録は 0 件だった
- canary が漏れを見つけられることも確かめた: `setMergeOrderPATH` を「bin のあとに受け継いだ PATH を続ける」形に変えた写しを overlay で読ませると、canary に 25 回届いた。このとき落ちたのは Text と JSON のテストだけで、`gh` を持たないほかの 2 本(trunk のテスト、`--state-dir` のテスト)は `gh` の注記を見ないので、漏れても落ちない。漏れを防いでいるのは `setMergeOrderPATH` で、今の形では漏れない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| gate の記録の読み方(`gateRecordFromDetails`)を変えない(計画の Regression) | 変わっていない | `gate.go`・`gate_run.go`・`reserve.go`・`manifest.go` の差分は空。gate のテスト 39 本は go1.25.8 で PASS |
| Go 1.25 だけで起きる違い(org-gate cycle 3 で 1.25 と 1.26 の差が出た) | 差はない | go1.25.8 と go1.26.0 の両方で、merge-order のテスト 37 本と org・cli の全体が PASS |
| 前に揺れたテスト(`TestRunDoctor*`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`) | 揺れなかった | 2 回の全体の実行のどちらでも PASS |

## Mutation

- `mergeorder.go` に 58 件、`org_merge_order.go` に 12 件の mutant を作り、`go test -overlay` で読ませた。build できなかった 4 件(使わない変数や import が残った)は同じ変更を build できる形に書き直した(M17b、M23b、M39b、M51b)。build できた 66 件のうち 58 件が red(org のテストで 48 件、CLI のテストで 10 件)、8 件が生き残った
- 生き残りのうち 2 件は、本物の入力では動きが変わらない
  - M13: ls-remote の出力に `ref: …<TAB>HEAD` が 2 行あるとき最後の行を採る。git は HEAD の symref を 1 行しか出さない
  - M36: PR の state を大文字にそろえない。本物の `gh` は `MERGED` などの大文字で返す(real-origin の evidence)
- 残りの 6 件はテストの穴で、どれも 1 行の probe で落ちる。probe は HEAD では通り、mutant で落ちた(mutation の evidence の P1〜P5)
  - M05(P2): 読めない記録を、その org の最初の結びつきに付ける。今のテストは記録より前の結びつきが 1 つなので、最初と「記録より前の最新」が同じになる。P2 は org-a を x に結び、x で通し、disband し、y に結び、y で通したあとに読めない記録を置く。HEAD では x が ready で y が unreadable、M05 では x が unreadable で y が ready になる。y は前の `gate_passed` に戻って ready になり、C8 が防ごうとしている動きそのものになる
  - M03(P1): 壊れた行の位置を、最後ではなく最初の行で比べる。P1 は壊れた行を gate の記録の前と後に 1 つずつ置き、unreadable を期待する。M03 では ready になる
  - C12(P5): 注記を `inboxField` を通さずに出す。今のテストは注記に制御文字が入る場合を持たない。P5 は 2 行の stderr(`HTTP 401: Bad credentials` と `note: forged`)を出して exit 1 で終わる `gh` で打つ。C12 では注記が `note: forged): its PR is unknown…` の行に割れ、偽の注記の行ができる。本物の `gh` の認証のエラーは複数行の stderr を出すことが多いので、実際に起きうる
  - C06 と C07(P4): 古い worktree の台帳の守りを、変える動詞の扱いにする、または外す。merge-order を古い台帳のある linked worktree から打つテストがない(`TestOrgLegacyLedger_ReadOnlyVerbsNoteAndContinue` の表に merge-order の行がない)。P4 は、その表と同じ準備で `ralph org merge-order` が終了コード 0 で、stderr に注意を 1 回出すことを確かめる
  - M09(P3): `org_id` のない行を数える。ralph が書く行には `org_id` があるので、手で書き換えた台帳でしか起きない
- M51b(環境変数を受け継がない)は CLI のテストだけが落とした。org の `TestMergeOrder_RealExecRunsInDirWithEnvAndStderr` は「受け継いだ環境に足す」と書くが、上書きした値だけを見るので、受け継ぎが消えても落ちない

## Test gaps

覆われていない場合を、確信を増やす順に挙げる。どれもテストの失敗ではない。

- M05: 同じ org_id の結びつきが記録より前に 2 つあるときの、読めない記録の付け先。計画の Test plan の「同じ org_id で前後に別の機能があっても、記録より前の最新の結びつき」のうち、「前に 2 つ」の場合がない。足す行は P2 で、`TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` の隣に置ける
- C12: 注記の制御文字のエスケープ。足す行は P5 で、`TestOrgMergeOrder_ControlCharactersInLedgerAndGHValuesAreEscaped` の隣に置ける
- M03: 壊れた行が gate の記録の前と後の両方にある場合。足す行は P1 で、`TestMergeOrder_LatestGateRecordWins` の表に 1 行足せる
- C06・C07: merge-order の古い台帳の守り(計画の Scope は `ralph status` と同じ読み取りの守りと書く)。足す行は P4 で、`TestOrgLegacyLedger_ReadOnlyVerbsNoteAndContinue` の表に `{"org_merge_order", []string{"org", "merge-order"}, false, stateDirHint}` を足せる。その表は PATH を受け継ぐ(`setupOrgStubPATH`)ので、共有の台帳に結びつきを置かないこと(置くと本物の `gh` に届く)
- M09: `org_id` のない行。足す行は P3
- 既定の timeout(ls-remote 30 秒、`gh` 15 秒)の値そのものは固めていない。0 のときに既定を使うこと(M18、M22)は red になる。偽の seam で 20 ms の timeout を確かめ、`realMergeOrderExec` のテストで本物のプロセスの kill と `WaitDelay` を確かめている。ssh の passphrase で止まる場合は tech-debt に書いてある
- CLI を通して打っていない場合: `gh` が非 0 で終わる、`base_ref` が `refs/heads/<名前>`、event の名前と reason が合わない記録、改行のない最後の行。どれも unit のテストが覆い、`gh` の失敗と改行のない最後の行は /verify が binary で確かめた(R13、R11)。`gh` の失敗は上の P5 でも CLI を通して動いた
- 観察(穴ではない): 計画は ready の中を「gate の記録の時刻が古い順」と書き、コードとテストは manifest の中の位置で並べる。manifest は追記だけなので同じ順になるが、記録の `ts` の時計がずれた場合は追記の順が勝つ
- 本物の GitHub での確認は、この /test で 1 回だけ打った(real-origin の evidence)。GitHub は `ref: refs/heads/main<TAB>HEAD` を返し、`gh pr list --json` は 6 つの欄をすべて大文字の state で返し、merge 済みでブランチが消えた PR #225 は既定で隠れて `--all` で merged になり、origin の先頭で gate を通したブランチは ready になった。verify の報告が未確認とした 2 つの前提は、これで確かめた

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh`(changed)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。go1.25.8 の org と cli は top-level 1,299 本、subtest 1,846 本が PASS で 0 fail、足した 37 本もすべて PASS で、消えたテストはない。`(cached)` だった 6 パッケージも打ち直して ok。mutation は 66 件のうち 58 件が red で、生き残り 8 件のうち 6 件は probe の行で落ちることを確かめ、2 件は本物の入力では動きが変わらない。テストは本物の `gh` に届かない(canary 0 件)
- Fail: 0
- Blocked: なし
