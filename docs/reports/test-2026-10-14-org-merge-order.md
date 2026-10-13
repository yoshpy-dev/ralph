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

- Verdict: pass(ブランチ全体、cycle 2 の HEAD e346083d の時点)
- Pass: `./scripts/run-test.sh`(changed)は rc 0 で、shell 41 本すべて OK、Go 8 パッケージすべて ok、shell の集計の行は cycle 1 と同じ。`./internal/org/... ./internal/cli/...` は go1.25.8 と go1.26.0 のどちらでも、top-level 1,375 本(PASS 1,374、前からの skip 1)と subtest 2,025 本(すべて PASS)で、fail は 0 だった。merge-order のテスト 49 本(cycle 2 で足した 12 本を含む)もすべて PASS で、消えたテストはない。`-race` も ok。`(cached)` だった 6 パッケージも打ち直して ok
- Pass(mutation): cycle 2 のコードの mutant 87 件のうち 72 件が red。生き残り 15 件のうち 10 件は probe の行で落ちることを確かめ、5 件は本物の入力では動きが変わらない。cycle 1 の穴 6 件(M03、M05、M09、C06、C07、C12)は cycle 2 のテストで red になった。cycle 1 の mutant も HEAD のテストで 67 件のうち 65 件が red で、残りは cycle 1 と同じ M13・M36
- Pass(本物の origin と `gh`): テストは本物の `gh` に届かない(canary 0 件)。読むだけの確認で、この worktree では PR #225 が merged と出た。fork を模した repo では upstream の PR を捨てて set-default を勧め、修正前の binary は同じ PR を使った
- 数の内訳: cycle 1 の時点の数は上の各節、cycle 2 の数は下の `## Cycle 2` にある
- Fail: 0
- Blocked: なし

## Cycle 2

- Date: 2026-10-14(JST 07:55〜09:00)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 の最後)。self-review の cycle 2(LOW 6 件、Merge: yes)と verify の cycle 2(pass、LOW 2 件)のあとに走らせた
- Scope: HEAD e346083d(base ac34ccf3)。cycle 2 でコードとテストを変えたコミットは 15d4b947 だけで、`git diff cab2fd50..HEAD -- internal/` は 5 ファイル、+842/-52。中身は、gh の PR を origin の repository と突き合わせる直し、L1 の `--org-id` と但し書き、L2 の gh の失敗の注記のまとめ、help の文、テスト 12 本と legacy-ledger の表の 2 行。ee302b06 から e346083d までは文書と報告だけ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は cycle 1 と同じで(macOS arm64、go1.26.0 と go1.25.8、git 2.49.0、`/bin/sh` は bash 3.2.57)、gh は 2.102.0
- Evidence(どれもコミットする。wrapper が書いた `docs/evidence/verify-2026-10-13-225615.log` は gitignore の対象なのでコミットしない):
  - `docs/evidence/test-2026-10-14-org-merge-order-c2-run-test.txt`: `./scripts/run-test.sh`(changed)の出力を cycle 1 と同じ方法で抜いたもの、`(cached)` だった 6 パッケージの打ち直し、cycle 1 の行との diff
  - `docs/evidence/test-2026-10-14-org-merge-order-c2-go.txt`: go1.25.8 と go1.26.0 での `./internal/org/... ./internal/cli/...` の結果。パッケージごとの数、SKIP の行、足した 12 本と legacy-ledger の 2 行の `--- PASS` の行、merge-order のテスト 49 本の行、前に揺れたテストの行、merge-order の 2 ファイルの関数ごとの coverage、`-race` の結果
  - `docs/evidence/test-2026-10-14-org-merge-order-c2-mutation.txt`: cycle 2 のコードの mutant(build できたもの 94 件)、cycle 1 の mutant を HEAD のテストで打ち直した 67 件、生き残りの分類、probe の行 PB1〜PB8 の結果とその source
  - `docs/evidence/test-2026-10-14-org-merge-order-c2-real-origin.txt`: 本物の GitHub の origin と本物の `gh` に対する読むだけの確認(この worktree、fork を模した一時の repo、修正前の binary)
  - insight event: lead の指示(報告と evidence だけをコミット)と、self-review と verify の cycle 2 の行を orchestrator が別のコミットで足した流れに合わせ、`docs/insights/events/2026-10-13-org-merge-order.jsonl` には書いていない。`./scripts/insights-append.sh --slug org-merge-order --flow standard --phase test --verdict pass --cycle auto --source skill --events-dir <scratch>` で作った行は `"phase":"test","cycle":2,"verdict":"pass"` で、`--cycle auto` は 2 を読んだ
- テストもコードも変えていない。mutant の写し、probe のテスト、偽の `gh`、git と gh の呼び出しを記録する wrapper は scratchpad に置き、`go test -overlay` と PATH で読ませた。worktree に書いたのは、この報告の追記と上の evidence 4 本だけで、mutation のあとも `git status --porcelain` は空、`git diff --quiet` は rc 0 だった。テストの前後で `~/.config/herdr` の下に 07:56 より新しいファイルはなく、main のチェックアウトの `git status --short` は空のまま、`$TMPDIR` に `ralph-verify-*` は残っていない。herdr の server も osascript も起こしていない

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed、go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(cycle 1 と同じ) | 10 分 48 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/... ./internal/cli/... -count=1 -v -coverprofile` | org: top-level 680、subtest 1,361。org/driver: 48、89。org/protocol: 15、56。cli: 632、519 | org の top-level 679 本、ほかはすべて | 0 | org 1 | 241 s(org 111.2 s、cli 237.2 s)、rc 0 |
| `go test ./internal/org/... ./internal/cli/... -count=1 -v`(go1.26.0) | 上と同じ | 上と同じ | 0 | org 1 | 243 s(org 108.9 s、cli 241.7 s)、rc 0 |
| うち merge-order のテスト(`TestMergeOrder_*` 34 本、`TestOrgMergeOrder_*` 15 本) | top-level 49、subtest 69 | 両方の toolchain で 49、69 | 0 | 0 | - |
| `-race`(go1.26.0): `./internal/org/ -run MergeOrder -count=3`、`./internal/cli/ -run OrgMergeOrder` | - | 2 つとも ok | 0 | - | 15.8 s、35.4 s |
| `(cached)` だった 6 パッケージを `-count=1`(go1.26.0) | config、insights、org/driver、org/protocol、scaffold、upgrade | 6 パッケージすべて ok | 0 | - | 合わせて約 8 s |
| mutation: cycle 2 のコード(`go test -overlay`、go1.26.0) | build できた mutant 94 件 | red 79 件 | 生き残り 15 件(下の「Mutation(cycle 2)」) | build できない 2 件は書き直して 2 件とも red | 1,599 s |
| mutation: cycle 1 の mutant を HEAD のテストで打ち直し | build できた 67 件 | red 65 件 | 生き残り 2 件(cycle 1 と同じ M13、M36) | build できない 3 件は書き直した形で red | 約 19 分 |
| probe(overlay だけの 8 本、コミットしない) | HEAD と生き残り 15 件 | HEAD では 8 本すべて通る。閉じられる生き残り 10 件は、どれも狙った probe で落ちる | - | - | - |
| 本物の origin と `gh` の確認(読むだけ) | ls-remote 1 回、HEAD の binary 6 回、修正前の binary 1 回 | すべて期待どおり | 0 | - | - |

- shell は 41 本すべてが `OK` で終わり、集計の `FAIL:` の行はどれも `FAIL: 0` だった。抜き出した行から行番号を外して cycle 1 の evidence と diff を取ると、違うのは時刻、Go の 2 つの所要時間、末尾の行だけで、suite ごとの集計の行はすべて同じ。Skipped は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の 1 件
- cycle 1 は `./internal/org/ ./internal/cli/` だけを数えた。その 2 パッケージで比べると、top-level は org 670 → 680(+10)、cli 630 → 632(+2)で、cycle 2 で足した `func Test` 12 本と合う。subtest は org 1,329 → 1,361(+32)、cli 517 → 519(+2、legacy-ledger の 2 行)。`git diff ac34ccf3...HEAD -- '*_test.go'` で消えた `func Test` は 0。2 つの toolchain で数は同じだった
- `run-test.sh` の中の Go は `internal/cli`(182.4 s)と `internal/org`(97.2 s)を流し直し、ほかの 6 パッケージは `(cached)` だった。`internal/org` を import するのは `internal/insights` だけ(`go list -deps`)で、6 つとも `-count=1` で打ち直して ok
- 前に揺れたテスト(`TestRunDoctor*`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どちらの toolchain でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は 3.30 s と 3.08 s で、+5 s の subtest は出なかった。load average は 3〜9 だった
- 本物の `gh` に届かないこと: 呼ばれたら記録して exit 97 で終わる偽の `gh`(canary)を、受け継ぐ PATH の先頭に置いて、2 回の全体の実行、`-race`、mutation のすべての実行を走らせた。canary の記録は 0 件。足した legacy-ledger の 2 行は開発者の PATH を受け継ぐ(`setupOrgStubPATH`)が、共有の台帳に結びつきがないので merge-order は git も gh も打たない。canary の 0 件はこの 2 行も含む

### Coverage(cycle 2)

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.0%(cycle 1 は 93.9%)、`internal/cli` 86.8%(同じ)、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%
- `internal/org/mergeorder.go` は 296/297 文(99.7%、cycle 1 は 212/213)、`internal/cli/org_merge_order.go` は 39/43 文(90.7%、同じ)。通らない文は cycle 1 と同じ 5 つ(`Exec` が nil のときの `run = realMergeOrderExec`、CLI の 4 つ)
- cycle 2 で足したか書き直した関数(`lookupMergePRs`、`mergeGHFailures` の `add` と `notes`、`mergeRepo` の `String` と `same`、`mergeOriginRepo.check`、`readMergeOriginRepo`、`mergeRepoFromOriginURL`、`mergeRepoFromPRURL`、`mergeGateCommand`)は、どれも 100%。それでも下の mutant が 15 件生き残った。穴は通らない行ではなく、テストの入力が区別しない値にある
- Branch: 測っていない(cycle 1 と同じ)。代わりに mutation で分岐ごとの判定を確かめた

### cycle 1 の穴は閉じたか

| cycle 1 の穴 | 閉じたテスト | mutant(HEAD のテスト) |
| --- | --- | --- |
| M05: 読めない記録を、その org の最初の結びつきに付ける | `TestMergeOrder_UnreadableRecordAttachesToTheLatestOfTwoEarlierBindings`(cycle 1 の P2 と同じ並び) | red |
| M03: 壊れた行の位置を最初の行で比べる | `TestMergeOrder_LatestGateRecordWins` に足した「of corrupt lines before and after the only gate record, the last one decides」(P1)。注記の壊れた行の数も数えるようになった | red |
| M09: `org_id` のない行を数える | `TestMergeOrder_RecordsWithoutAnOrgIDAreNotRead`(P3) | red |
| C06・C07: 古い worktree の台帳の守り | `TestOrgLegacyLedger_ReadOnlyVerbsNoteAndContinue` に足した `org_merge_order` と `org_merge_order_json` の行(P4) | 2 件とも red |
| C12: 注記を `inboxField` に通さない | `TestOrgMergeOrder_MultiLineGHStderrStaysOneNoteLine`(P5) | red |

- cycle 1 の mutant を HEAD のテストで打ち直した。cycle 1 で red だったものは全部 red のままで、生き残りは cycle 1 が「本物の入力では同じ動き」と分けた M13・M36 だけだった。cycle 2 で行が動いた 4 件(M20、M23、M24、M38)と、cycle 1 で build できる形に直した 4 件(M17b、M23b、M39b、M51b)は HEAD に合わせて書き直し、8 件とも red

### Mutation(cycle 2)

- cycle 2 のコードに 87 件の mutant を作った(build できなかった K06・K07 は、同じ変更を build できる形にした K06b・K07b で数える)。ほかに、cycle 1 の穴の 6 件と、self-review C2-1 の推奨の直しを入れた H01 を 1 件、同じ方法で読ませた。cycle 2 のコードの 87 件は 72 件が red、15 件が生き残った

| 対象 | mutant | red | 生き残り |
| --- | --- | --- | --- |
| `mergeOriginRepo.check`(K01〜K11) | 11 | 11 | なし |
| `lookupMergePRs`(L01〜L08: PR を捨てる、gh を止める、`gh_available`、注記) | 8 | 6 | L04、L05 |
| `mergeGHFailures`(G01〜G07: L2 のまとめ) | 7 | 6 | G04 |
| `mergeRepo.same`(S01〜S05) | 5 | 4 | S04 |
| `mergeRepoFromOriginURL`(O01〜O19) | 19 | 14 | O03、O04、O06、O16、O19 |
| `readMergeOriginRepo`(U01〜U06、O20、O21) | 8 | 5 | U02、O20、O21 |
| `mergeRepoFromPRURL`(P01〜P10) | 10 | 8 | P04、P05 |
| L1 の Detail と注記(D01〜D03、`--org-id` を外す D04a〜g、但し書きを外す E04a〜h) | 18 | 17 | D02 |
| 注記の定数 `mergeOrderNoPRs`(D06) | 1 | 1 | なし |

- gh を打ち止める経路は、3 つの壊し方のどれも落ちた: 不一致のあとも gh を打ち続ける(L02)、先に付けた PR を残す(L01)、`gh_available` を false にする(L03)。get-url を毎回打つ(K01、K02)、get-url が失敗したら確かめずに使う(K04)、get-url の timeout を外す(U05。90 s の `-timeout` で落ちた)も red
- L1 は、`--org-id` を外す mutant 7 件と但し書きを外す mutant 8 件を、Detail と注記の場所ごとに 1 件ずつ作った。15 件とも `TestMergeOrder_DetailsNameTheGateCommandWithTheOrg` が落とした。このテストは分類ごとに 1 行を持つ
- H01(結びつきのない枝に、gate の記録の org_id を `--org-id` として付ける。self-review C2-1 の推奨)も同じテストで red になった。テストは今の「`--org-id` を出さない」を固定しているので、C2-1 を直すときはその assertion も書き換えることになる

生き残り 15 件のうち 10 件は、HEAD で通る probe の行が落とす(evidence の PB1〜PB8)。確信が増える順に挙げる。

- S04(PB4): owner だけを比べる。owner が同じで repository が違う PR の行がない。今のテストで「別の repository」とする行は、どれも owner が違う(upstream-org/app、someone/app)。`gh repo set-default` が同じ owner の別の repository(acme/app に対する acme/app-docs など)を指していると、その PR の state で枝が隠れる。足す行は PB4 で、`TestMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut` の表に 1 行で入る
- O03・O04(PB5、PB5b): `mergeOriginURLSchemes` から `http` か `git` を外す。`TestMergeOrder_PRIsUsedWhenItsURLNamesOriginsRepository` には `http://` と `git://` の origin の行があるが、その PR は acme/app のものなので、確かめが通っても飛ばされても PR は使われる。この 2 行は「読めて一致した」と「読めなかった」を区別しない。URL を読む関数の表には `http://` と `git://` の行がない。足す行は、その表の 2 行(PB5)か、別の repository の PR が捨てられることを見る行(PB5b)
- D02(PB7): Detail の `--org-id` を、最新の結びつきではなく gate の記録の org から取る。fixture では結びつきの org と gate の記録の org がいつも同じ(org-a)なので、取り違えても分からない。PB7 は同じ枝を org-a、org-b の順に結び、その前に org-a の gate_failed を置き、`--org-id org-b` を期待する
- L04(PB1): 先の枝で gh が失敗し、後の枝で別の repository の PR が返ると、先の失敗の注記が消える。足す行は PB1
- G04(PB3): 同じ理由の失敗がちょうど 5 本のとき「and 0 more」が付く。テストの本数は 7、2、1 で、5 がない。足す行は PB3
- P04・P05(PB8)と O16(PB5_PB6): owner や repository が空の URL(`https://github.com//app/pull/7`、`git@github.com:acme//`)を読む。P04・P05 でも PR を全部捨てるのは同じで、注記の文が「gh reads /app」に変わるだけ。本物の gh はこの形の url を返さないので、足しても確信はあまり増えない
- L05(PB2): gh が失敗したあとで見つからなくなると、先の失敗の注記が消える。実行の途中で gh が PATH から消えた場合だけで、現実にはまず起きない

残りの 5 件は、本物の入力では動きが変わらない。

- O06(`!scp` の判定を外す): コロンのない文字列は残りが空になり、path の要素が 1 つなので、どのみち読めない
- O19(最初の `@` で切る): host は空かどうかしか見ない。違いが出るのは host の部分が `@` で終わる URL だけ
- O20・O21(get-url の出力の 1 行目だけを読む、前後の空白を落とす): `git remote get-url origin` は LF で終わる 1 行しか出さない
- U02(外側の ctx が終わったときも timeout と書く): 呼び元の `check` が先に `ctx.Err()` を見て interrupted を返す

### 本物の origin と `gh`(読むだけ)

verify の cycle 2 が「足すと確信がいちばん増える確認」に挙げた run を打った。加えて、fork を模した一時の repo で、cycle 2 の直しの前提(fork の作業場所では gh が upstream を読む)を確かめた。GitHub には何も書いていない(`gh repo set-default` が書いたのは一時の repo の `.git/config` だけ)。

- この worktree(remote は origin の `ssh://git@github.com/yoshpy-dev/ralph.git` だけ)で、feat/org-gate(PR #225)とこの枝を名指す一時の台帳を作り、HEAD の binary を打った。`--all` の PR の列は `#225 https://github.com/yoshpy-dev/ralph/pull/225` で、分類は merged。`--json` では feat/org-gate が隠れて、注記が隠した数を言う。`--all --json` では `pr` に number 225、state MERGED が入る。get-url は gh が最初の PR を返したあとに 1 回だけ打たれた。この repo では突き合わせが通り、結果は cycle 1 と同じになる
- fork を模した一時の repo(origin が yoshpy-dev/ralph、upstream が cli/cli)で、cli/cli の merge 済み PR #14650 の枝名を台帳に置いた
  - HEAD: 本物の gh は upstream(cli/cli)を読んだ。merge-order は PR #14650 を使わず、注記は「gh reads cli/cli, not origin's yoshpy-dev/ralph: ... run gh repo set-default yoshpy-dev/ralph in this repository」で、gh は 1 回で止まった
  - 修正前の binary(cab2fd50): 同じ台帳で cli/cli の PR #14650 を使い、その枝を merged にした(`--all` がなければ隠れる)。cross-review の指摘を本物の gh で再現した
  - 注記のとおり `gh repo set-default yoshpy-dev/ralph` を打つと、gh は origin を読み、PR #225 が merged に戻った
  - origin を `git+ssh://` に書き換え、set-default を外すと、`mergeRepoFromOriginURL` が読めないので確かめを飛ばし、cli/cli の PR #14650 をまた使った。verify の K12 と tech-debt の `mergeOriginRepo.check` の行が書く既知の穴で、本物の gh でも起きる。このとき feat/org-gate は、cli/cli にその名前の PR がないので no_remote になった(verify の D3 (c))
- gh は PR を引くたびに `git remote -v` と `git config --get-regexp ^remote\..*\.gh-resolved$` を自分で打った(記録用の wrapper に残った)。gh が remote と set-default から読む repository を選ぶ、という `mergeorder.go` の先頭の説明と合う

### Failure analysis(cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks(cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| gh が origin でない repository の PR を返すと、それで枝を隠す(cross-review cycle 1 の ACTION_REQUIRED #1) | 直った(偽の gh と本物の gh の両方) | mutation の K01〜K11 と L01〜L03 がすべて red。real-origin の 2 節では HEAD が upstream の PR を捨て、修正前の binary は同じ台帳で使う |
| cycle 1 の穴 M03・M05・M09・C06・C07・C12 | 6 件とも red | 上の「cycle 1 の穴は閉じたか」 |
| cycle 1 の mutant で red だったもの | HEAD のテストでも red | 67 件のうち 65 件。生き残りは同じ M13・M36 |
| gate の記録の読み方(`gateRecordFromDetails`)を変えない | 変わっていない | `git diff ac34ccf3...HEAD -- internal/org/gate.go internal/org/gate_run.go internal/org/reserve.go internal/org/manifest.go` は空 |
| Go 1.25 だけで起きる違い | 差はない | go1.25.8 と go1.26.0 で数と結果が同じ |
| 前に揺れたテスト | 揺れなかった | 2 回の全体の実行のどちらでも PASS |

### Test gaps(cycle 2)

覆われていない場合を、確信が増える順に挙げる。どれもテストの失敗ではない。

- S04: owner が同じで repository が違う PR。足す行は PB4
- O03・O04: `http://` と `git://` の origin を読むこと。今の 2 行は区別しないので、足す行は PB5 か PB5b
- D02: 結びつきの org と gate の記録の org が違う枝の Detail。足す行は PB7。self-review C2-1 を直すときに、結びつきのない枝の場合と一緒に固めるのがよい
- L04: gh の失敗の注記と repository の注記が並ぶ場合。足す行は PB1
- G04: ちょうど 5 本の失敗。足す行は PB3
- P04・P05・O16・L05: 本物の gh や git が作らない入力。足しても確信はあまり増えない
- 既知の穴(テストではなく設計の範囲): origin の URL が owner/repo を持たないときは確かめない(verify の K11・K12、本物の gh で再現)、別の repository にその名前の PR がないときは注記が出ない(D3 (c))、host を比べない。どれも `docs/tech-debt/README.md` の `mergeOriginRepo.check` の行にある
- get-url の timeout は `GHTimeout`(既定 15 s)を流用する(self-review C2-2)。20 ms で止まることはテストが確かめるが、既定の値そのものは固めていない。cycle 1 の ls-remote と gh の既定の値と同じ扱い
- verify の cycle 2 が「binary で打っていない」とした `--json` の PR の欄は、real-origin の 1 節で本物の PR #225 を出した。偽の gh を使う binary の run では、まだ打っていない
