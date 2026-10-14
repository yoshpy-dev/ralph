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

- Verdict: pass(ブランチ全体、cycle 4 の HEAD 6d64e96a の時点)
- Pass: `./scripts/run-test.sh`(changed)は rc 0 で、shell 41 本すべて OK、Go 8 パッケージすべて ok、shell の集計の行は cycle 1〜3 と同じ。`./internal/org/... ./internal/cli/...` は go1.25.8 と go1.26.0 のどちらでも、top-level 1,379 本(PASS 1,378、前からの skip 1)と subtest 2,033 本(すべて PASS)で、fail は 0 だった。`-race` を付けた同じ範囲の実行も ok で、race の報告は 0 件。merge-order のテスト 53 本(cycle 4 で足した 3 本と、その subtest 4 本を含む)もすべて PASS で、消えたテストはない。`(cached)` だった 5 パッケージも打ち直して ok
- Pass(mutation): cycle 4 のコードの mutant 29 件のうち 25 件が red。生き残り 4 件のうち 3 件(G04、G05、U05)は probe の行で落ちることを確かめた。どれも手で書いた台帳か注記の文の末尾だけの違いで、残る P09 は help の文の変更。cycle 3 の mutant を HEAD のテストで打ち直すと 44 件のうち 35 件が red で、生き残りは cycle 3 と同じ 9 件。N30・N31・N33、N34、N43・N44、N51 は red になっておらず、それぞれ cycle 3 の probe で落ちることは変わらない
- Pass(本物の origin と `gh`): テストは本物の `gh` に届かない(canary 0 件)。読むだけの確認で、GitHub で改名した vuejs/vue-next(今は vuejs/core)を origin の古い URL のまま打つと、gh の PR の url は新しい名前になり、HEAD は PR を使わずに 2 つの直し方を並べた注記を出した。1 つめの `git remote set-url origin` を打つと PR が使われた。修正前の binary が勧める `gh repo set-default vuejs/vue-next` は、gh が「does not correspond to any git remotes」で断った。cycle 3 までの確認(`--limit 1000`、fork の PR が上限を埋める vscode の patch-1)は下の `## Cycle 3` にある
- 数の内訳: cycle 1 の時点の数は上の各節、cycle 2〜4 の数は下の `## Cycle 2`、`## Cycle 3`、`## Cycle 4` にある
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

## Cycle 3

- Date: 2026-10-14(JST 10:26〜11:20)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 3 回目で最後(cross-review cycle 2 の ACTION_REQUIRED を直すために上限を 2 から 3 に上げた回。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)。self-review の cycle 3(LOW 5 件、C3-1〜C3-5)と verify の cycle 3(pass、D5〜D7)のあとに走らせた
- Scope: HEAD 7a8cac1e(base ac34ccf3)。cycle 3 でコードとテストを変えたのは d7599d82 と c885fcab で、`git diff e463eb9d..HEAD -- internal/` は 4 ファイル。中身は、`gh pr list` に渡す `--limit 1000`(`mergeOrderGHLimit`)、上限ぶんの PR がすべて fork のときの注記(`mergeGHLimitNotes`、切り出した `mergeNamedBranches`)、gate の記録だけが名指すブランチの Detail の `--org-id`(`mergeGateCommand`)、`GHTimeout` の doc、テスト 1 本(subtest 4 本)と期待の書き換え 3 か所。ほかのコミットは計画、報告、insight event、skill の 1 行だけ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は cycle 2 と同じで(macOS arm64、go1.26.0 と go1.25.8、git 2.49.0、`/bin/sh` は bash 3.2.57)、gh は 2.102.0
- Evidence(どれもコミットする。wrapper が書いた `docs/evidence/verify-2026-10-14-012647.log` は gitignore の対象なのでコミットしない):
  - `docs/evidence/test-2026-10-14-org-merge-order-c3-run-test.txt`: `./scripts/run-test.sh`(changed)の出力を cycle 1・2 と同じ方法で抜いたもの、`(cached)` だった 5 パッケージの打ち直し、cycle 2 の行との diff
  - `docs/evidence/test-2026-10-14-org-merge-order-c3-go.txt`: go1.25.8 と go1.26.0 での `./internal/org/... ./internal/cli/...` のパッケージごとの数、`-race` の結果、SKIP の行、2 つの toolchain の間と cycle 2 との間のテスト名の diff、merge-order のテスト 50 本の行、前に揺れたテストの行、merge-order の 2 ファイルの関数ごとの coverage
  - `docs/evidence/test-2026-10-14-org-merge-order-c3-mutation.txt`: cycle 3 のコードの mutant 44 件、cycle 2 の mutant を HEAD で打ち直した 87 件(88 行)、cycle 1 の穴 6 件、probe の行の結果とその source、mutant を作る script と打つ script
  - `docs/evidence/test-2026-10-14-org-merge-order-c3-real-origin.txt`: 本物の origin と本物の `gh` に対する読むだけの確認(この worktree、gh を直接打った 5 回、microsoft/vscode を origin にした一時の repo)
  - insight event: lead の指示どおり書いていない。`./scripts/insights-append.sh --slug org-merge-order --flow standard --phase test --verdict pass --cycle 3 --source skill --events-dir <scratch>` で作った行は `"phase":"test","cycle":3,"verdict":"pass"`。`--cycle auto` は `cycle-count.json` の 2 を読むので、verify と同じく 3 を渡した
- テストもコードも変えていない。mutant の写し、probe のテスト、偽の `gh`(canary)、git と gh の呼び出しを記録する wrapper、一時の台帳と repo は scratchpad に置き、`go test -overlay` と PATH で読ませた。worktree に書いたのは、この報告の追記と `## Verdict` の書き換え、上の evidence 4 本だけで、mutation のあとも `git status --porcelain` は空、`git diff --quiet` は rc 0 だった。テストの前後で `~/.config/herdr` の下に 10:26 より新しいファイルはなく、main のチェックアウトの `git status --short` は空のまま、`$TMPDIR` に `ralph-verify-*` は残っていない。herdr の server も osascript も起こしていない

### Test execution(cycle 3)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed、go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(cycle 1・2 と同じ) | 752 s、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/... ./internal/cli/... -count=1 -v -coverprofile` | org: top-level 681、subtest 1,365。org/driver: 48、89。org/protocol: 15、56。cli: 632、519 | org の top-level 680 本、ほかはすべて | 0 | org 1 | 277 s(org 138.8 s、cli 273.4 s)、rc 0 |
| `go test ./internal/org/... ./internal/cli/... -count=1 -v`(go1.26.0) | 上と同じ | 上と同じ | 0 | org 1 | 275 s(org 134.6 s、cli 274.2 s)、rc 0 |
| うち merge-order のテスト(`TestMergeOrder_*` 35 本、`TestOrgMergeOrder_*` 15 本) | top-level 50、subtest 73 | 両方の toolchain で 50、73 | 0 | 0 | - |
| `go test -race ./internal/org/... ./internal/cli/... -count=1`(go1.26.0) | 4 パッケージ | 4 パッケージすべて ok、`DATA RACE` は 0 件 | 0 | - | 255 s |
| `(cached)` だった 5 パッケージを `-count=1`(go1.26.0) | config、insights、org/protocol、scaffold、upgrade | 5 パッケージすべて ok | 0 | - | 合わせて約 6 s |
| mutation: cycle 3 のコード(`go test -overlay`、go1.26.0) | 44 件(build できない N20 は、書き直した N20b で数える) | red 35 件 | 生き残り 9 件(下の「Mutation(cycle 3)」) | - | 812 s |
| mutation: cycle 2 の mutant を HEAD のテストで打ち直し | 87 件(88 行) | red 73 件 | 生き残り 14 件(cycle 2 の 15 件から D02 が抜けた) | - | 1,473 s |
| mutation: cycle 1 の穴 6 件を HEAD のテストで打ち直し | 6 件 | red 6 件 | 0 | - | - |
| probe(overlay だけの 12 本、コミットしない) | HEAD と、help の N70 を除く生き残り 22 件 | HEAD では 12 本すべて通る。閉じられる生き残り 16 件は、どれも狙った probe で落ちる | - | - | - |
| 本物の origin と `gh` の確認(読むだけ) | HEAD の binary 5 回、修正前の binary 2 回、gh を直接 5 回 | すべて期待どおり | 0 | - | - |

- shell は 41 本すべてが `OK` で終わり、集計の `FAIL:` の行はどれも `FAIL: 0` だった。抜き出した行から行番号を外して cycle 2 の evidence と diff を取ると、違うのは時刻、Go の所要時間、`internal/org/driver` が今回は `(cached)` でなかったこと(0.827 s で ok)、末尾の行だけで、suite ごとの集計の行はすべて同じ。Skipped は cycle 1・2 と同じ `tests/test-secret-scan-branch.sh` の 1 件
- cycle 2 と比べると、top-level は org 680 → 681(+1)、subtest は org 1,361 → 1,365(+4)で、cycle 3 で足した `TestMergeOrder_GHLimitFullOfForksLeavesThePROut` とその subtest 4 本の数と合う。cli は 632 と 519 で変わらない。テスト名の一覧(PASS・SKIP の行から所要時間を外したもの)を cycle 2 の go1.25.8 の一覧と比べると、違いはこの 5 行の追加だけで、消えた名前はない。go1.25.8 と go1.26.0 の一覧は同じ
- 期待を書き換えた 3 か所(`TestMergeOrder_DetailsNameTheGateCommandWithTheOrg` の feat/orphan の行と、足した twice の行と手で作った target の行、`TestMergeOrder_RunsGitAndGHWithTheirArgsDirAndEnv` の argv、CLI の `mergeOrderGHCall`)を通るテストは、どれも両方の toolchain で PASS
- `run-test.sh` の中の Go は `internal/cli`(191.3 s)、`internal/org`(78.5 s)、`internal/org/driver` を流し直し、残りの 5 パッケージは `(cached)` だった。`internal/org` を import するのは `internal/insights` だけ(`go list -deps`)で、5 つとも `-count=1` で打ち直して ok
- `-race` は、cycle 2 では merge-order のテストだけに付けた。今回は org と cli の全体に 1 回付けて、race の報告は 0 件
- 前に揺れたテスト(`TestRunDoctor*`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どちらの toolchain でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は 3.66 s と 4.29 s で、+5 s の subtest は出なかった。load average は 4〜10 だった
- 本物の `gh` に届かないこと: 呼ばれたら記録して exit 97 で終わる偽の `gh`(canary)を受け継ぐ PATH の先頭に置き、`run-test.sh`、2 回の全体の実行、`-race`、mutation と probe のすべての実行を走らせた。canary の記録は 0 件

### Coverage(cycle 3)

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.1%(cycle 2 は 94.0%)、`internal/cli` 86.8%(同じ)、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%
- `internal/org/mergeorder.go` は 308/309 文(99.7%、cycle 2 は 296/297)、`internal/cli/org_merge_order.go` は 39/43 文(90.7%、同じ)。通らない文は cycle 1・2 と同じ 5 つ(`Exec` が nil のときの `run = realMergeOrderExec`、CLI の 4 つ)
- cycle 3 で足したか書き直した関数(`lookupMergePRs`、`mergeGHFailures.notes`、`mergeGHLimitNotes`、`mergeNamedBranches`、`lookupMergePR`、`mergeGateCommand`)は、どれも 100%。それでも下の 9 件が生き残った。cycle 2 と同じく、穴は通らない行ではなく、注記が並ぶ組み合わせと枝の本数の入力にある
- Branch: 測っていない(cycle 1・2 と同じ)。代わりに mutation で分岐ごとの判定を確かめた

### Mutation(cycle 3)

cycle 3 のコードに 44 件の mutant を作った。N20(上限の枝を消す)は `limitFullOfForks` が使われなくなって build できないので、同じ変更を `case false && limitFullOfForks:` と書いた N20b で数える。35 件が red、9 件が生き残った。

| 対象 | mutant | red | 生き残り |
| --- | --- | --- | --- |
| `--limit` の引数と `mergeOrderGHLimit`(N01〜N06: 引数を外す、`100` と書く、定数を 30 や 1001 にする、`--json` の後ろに置く、引数だけ 1 少なくする) | 6 | 6 | なし |
| 上限の判定 `pr == nil && len(prs) >= mergeOrderGHLimit`(N10〜N16) | 7 | 6 | N11 |
| `lookupMergePRs` の上限の枝(N20b〜N23: 枝を消す、`pr == nil` の後ろに置く、最後の枝だけを残す、最初の上限で gh を止める) | 4 | 4 | なし |
| `withNotes` の注記の中身と順(N30〜N36) | 7 | 3 | N30、N31、N33、N34 |
| `mergeGHLimitNotes`(N40〜N46: 単数と複数、0 本、数、名前、文の末尾) | 7 | 5 | N43、N44 |
| `mergeNamedBranches`(N50〜N55、gh の失敗の注記と共有) | 6 | 5 | N51 |
| `mergeGateCommand`(N60〜N65) | 6 | 6 | なし |
| help の文(N70) | 1 | 0 | N70 |

- `--limit` の argv は、別々の 2 か所が固めている。N01〜N06 は org の `TestMergeOrder_RunsGitAndGHWithTheirArgsDirAndEnv` が落とした。N01 と N03 を CLI のテストだけで打っても、`mergeOrderGHCall` で gh の呼び出しを比べる CLI のテスト 3 本が落ちた。N03(定数を 30 にすると、引数と判定の両方が 30 になる)は上限のテストでは落ちない。そのテストは fork の PR の本数を `mergeOrderGHLimit` から作るので、定数が変わっても筋が通ってしまう。落とすのは argv の 2 か所だけ
- 判定の境界は、どの壊し方も落ちた。`>` にする N10 と 1 本少なくても数える N12 は、上限ちょうど(注記あり)と 1 本少ない(注記なし)の対が落とし、`pr == nil` を外す N13 は、fork でない PR が 1 本混じる行が落とした。fork だけなら何本でも上限と数える N16 は、上限のテストの 1 本少ない行が落とした。数を見ない N14 は、上限のテストのほかに repository の確認のテストも落とした
- `mergeGateCommand` は 6 件とも red。c885fcab を戻す N61、gate の記録の org を結びつきの org より先に見る N60(cycle 2 の D02 と同じ)、org_id のない記録にも `--org-id` を付ける N62、裸の命令を変える N63、結びつきがあっても gate の記録の org を使う N64、org の代わりに event の名前を書く N65。twice の行と、手で作った target の行が効いている

生き残り 9 件のうち 7 件は、HEAD で通る probe の行が落とす(evidence の PC1〜PC4 と cycle 2 の PB3)。足すと確信が増える順に挙げる。

- N33(PC1)、N30・N31(PC1、PC2): 注記の並び。gh の失敗、上限、repository の不一致(または gh がない)の 3 つが同じ run に出る行がない。N33 は不一致で止まるときに上限の注記を捨て、N30 は上限の注記を gh の失敗の注記より前に、N31 は set-default や gh がないことの注記を先頭に置く。verify の Q5・Q6 は binary で並びを見たが、それを固めるテストはない。PC1 は、feat/a で gh が失敗し、feat/b で上限ぶんの fork の PR が返り、feat/c で upstream-org/app の PR が返る run で、注記がこの順に 3 つ並ぶことを見る。`TestMergeOrder_GHLimitFullOfForksLeavesThePROut` の subtest として 1 本で入る
- N43・N44(PC3): 上限の注記の複数形で、枝が 6 本以上の場合。テストの枝は 2 本なので、数を 5 で打ち切る N43 も、全部の名前を並べる N44 も同じ文になる。gh の失敗の注記には 7 本の行(`TestMergeOrder_GHFailingTheSameWayForSeveralBranchesIsOneNote`)があるが、上限の注記は数と名前を別の引数で組むので、その行は効かない。PC3 は 7 本の枝で「for each of 7 branches (feat/b01, feat/b02, feat/b03, feat/b04, feat/b05 and 2 more)」を見る
- N51(PB3、PC4): ちょうど 5 本で「and 0 more」が付く。cycle 2 の G04 と同じ変更で、`mergeNamedBranches` に移ったので、gh の失敗と上限の両方の注記に効く。どちらのテストにも 5 本の行がない。`mergeNamedBranches` を直接引く表にすれば、1 つの表で両方に効く
- N34(PC2): gh が途中で見つからなくなったときに、上限の注記を捨てる。gh が run の途中で PATH から消えた場合だけで、cycle 2 の L05 と同じく、現実にはまず起きない

残りの 2 件は、閉じる必要がないか、動きではない。

- N11(`>=` を `==` にする): 本物の gh は `--limit` より多くを返さない。下の「本物の origin と `gh`」で、vscode の patch-1 の PR 1009 本に対して `--limit 1000` は 1000 本、`--limit 1100` は 1009 本を返した。なので動きは変わらない
- N70(help が `--limit 30` と書く): help の文を読むテストはない。self-review の C3-3 が書くとおり、help と skill の 4 面にある 1000 は手で書いた数で、動きには関わらない

### cycle 2 の生き残りは HEAD で red か

cycle 2 の mutant の一覧を HEAD のテストで打ち直した。cycle 3 で行が動いた 7 件(L02〜L05、G03、G07、D01)は、同じ変更を HEAD に合わせて書き直した。87 件のうち 73 件が red で、生き残りは 14 件。cycle 2 の生き残り 15 件から D02 が抜けた。cycle 2 で red だったものは、全部 red のままだった。cycle 1 の穴 6 件(M03、M05、M09、C06、C07、C12)も打ち直して、6 件とも red。

| cycle 2 の閉じられる生き残り | HEAD で | 落とす probe(コミットしない) |
| --- | --- | --- |
| D02: Detail の org を gate の記録から取る | red。`TestMergeOrder_DetailsNameTheGateCommandWithTheOrg` に c885fcab が足した twice の行が落とす。D02 の古い置き換えの文字列は新しい switch の中でもまだ 1 回だけ一致するので、古い形(R2-D02)と書き直した形(R2-D02b、N60 と同じ)の両方で打ち、どちらも red | - |
| S04: owner だけを比べる | 生き残り | PB4 |
| O03・O04: `http` と `git` の scheme を読まない | 生き残り | PB5_PB6、PB5b |
| L04: 不一致で止まるときに、それまでの注記を捨てる | 生き残り(R2-L04b。今は gh の失敗と上限の両方の注記が消える) | PB1、PC1 |
| G04: ちょうど 5 本で「and 0 more」 | 生き残り(N51 と同じ) | PB3、PC4 |
| P04・P05・O16: owner や repository が空の URL | 生き残り | PB8、PB5_PB6 |
| L05: gh が見つからなくなったときに、それまでの注記を捨てる | 生き残り(R2-L05b) | PB2、PC2 |

- cycle 2 が本物の入力では同じ動きと分けた 5 件(O06、O19、O20、O21、U02)は生き残りのままで、どの probe でも落ちない
- cycle 3 で足した `TestMergeOrder_GHLimitFullOfForksLeavesThePROut` は、cycle 2 の repository の確認の mutant(R2-K09、R2-K11、R2-L02b)も落とした。上限の中から選んだ PR にも repository の確認が走ることを、4 つめの subtest が見ているため

### 本物の origin と `gh`(読むだけ)

GitHub には何も書いていない。gh は `gh pr list` だけ、git は ls-remote、`remote -v`、`config --get-regexp`、`remote get-url` だけを打った。HEAD(7a8cac1e)と修正前(e463eb9d)の binary は、どちらも go1.25.8 で scratchpad に作った。PATH の先頭には、argv を記録してから本物の git と gh を exec する wrapper を置いた。gh のアカウントは切り替えていない。

- この worktree(remote は origin の `ssh://git@github.com/yoshpy-dev/ralph.git` だけ)で、feat/org-gate(PR #225、merge 済みでブランチは消えた)だけを名指す一時の台帳を作り、HEAD の binary を打った
  - gh の argv は `pr list --head feat/org-gate --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository` で、HEAD の 3 回とも `--limit 1000` が付いた。修正前の binary の argv には `--limit` がない
  - `--all` の PR の列は `#225 https://github.com/yoshpy-dev/ralph/pull/225` で、分類は merged、Detail は「PR #225 is merged」。`--all --json` の `pr` は number 225、state MERGED、`head_ref_oid` 69141291…、`base_ref_name` main。`--all` を付けないと隠れて、注記が隠した数を言う。上限の注記は出ない。1 回は約 3 s で、修正前の binary と同じ出力だった
  - get-url は gh が PR を返したあとに 1 回だけ打たれた。台帳の sha256 は前後で同じ
- gh を直接打った(`gh pr list -R <repo> --head <name> --state all --limit <n>`)。verify が未確認とした「`--head` だけのとき、作成の新しい順に `--limit` まで返す」ことと、1000 本にかかる時間を見るため
  - cli/cli の patch-1: 302 本、すべて fork、番号の大きい順(作成日時も新しい順)、2.87 s
  - microsoft/vscode の patch-1: `--limit 1000` で 1000 本、すべて fork、番号の大きい順、9.64 s。`--limit 1100` では 1009 本で 7.62 s、1000 本の一覧は 1009 本の先頭 1000 本と同じだった。`--limit` を付けない(修正前の binary と同じ)と 30 本で、どれも fork の PR だった。gh は新しい順に返し、`--limit` で切る。どの API を使ったかは見ていない(`GH_DEBUG` は認証の header を出すおそれがあるので使わなかった)
  - torvalds/linux の master は 0 本だった(この repository は PR の一覧を返さない)
- microsoft/vscode を origin にした一時の repo(`git init` と `git remote add` だけで、コミットはない)で、patch-1 を名指す台帳を HEAD の binary で打った。origin に patch-1 のブランチはないので、分類は no_remote で PR の列は `-`。注記は「gh returned 1000 PRs for patch-1, its limit, all from forks, and the branch's own PR may be past them: no PR is used for it, so …」で、`--json` の `notes` の先頭にも同じ文が出た。終了コードは 0。上限の枝は repository の確認より前に抜けるので、get-url は打たれなかった。修正前の binary は同じ台帳で同じ分類を出し、注記は出さなかった(fork の PR 30 本だけを受けて、PR なしにした)。repo の `.git/config` は変わっていない
- HEAD の binary の vscode の run は 9.47 s と 9.11 s かかった。gh の 1 回で 1000 本を取るのに 9〜10 s かかり、`GHTimeout` の既定 15 s までの余裕は 5〜6 s ほどしかない。遅い回線では timed out の注記になりうる。その場合も、そのブランチは PR なしになって注記が付き、別の PR を黙って使うことはない(self-review と verify の読みのとおり)。普通のブランチ(PR が 1 本)では、run 全体が約 3 s だった

### Failure analysis(cycle 3)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks(cycle 3)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| gh の既定の上限 30 を、同じ名前の fork の PR が埋めると、ブランチ自身の PR が結果から落ちて、注記もなく PR なしになる(cross-review cycle 2 の ACTION_REQUIRED #1) | 直った(偽の gh と本物の gh の両方) | N01〜N06、N10、N12〜N16、N20b〜N23 が red。本物の gh で argv に `--limit 1000` が付いた。fork の PR が 1000 本返る vscode の patch-1 では、HEAD が注記を出し、修正前の binary は注記なしで PR なしにした |
| gate の記録だけが名指すブランチの Detail が `--org-id` を付けない(self-review C2-1) | 直った | N61(c885fcab を戻す)、N62〜N65 が red。verify の Q1 と Q8 が binary で同じことを見た |
| cycle 2 の mutant で red だったもの | HEAD のテストでも red | 87 件のうち 73 件が red。生き残り 14 件は、cycle 2 の 15 件から D02 を除いたもの |
| cycle 1 の穴 M03・M05・M09・C06・C07・C12 | 6 件とも red | mutation の evidence の C2 節 |
| Go 1.25 だけで起きる違い | 差はない | go1.25.8 と go1.26.0 で、数もテスト名も同じ |
| 前に揺れたテスト | 揺れなかった | 2 回の全体の実行と `-race` の実行のどれでも PASS |
| テストが本物の `gh` に届かない | 届かない | canary の記録は `run-test.sh` を含めて 0 件 |

### Test gaps(cycle 3)

覆われていない場合を、足すと確信が増える順に挙げる。どれもテストの失敗ではない。

- N30・N31・N33(と cycle 2 の L04): 注記の並び。gh の失敗、上限、repository の不一致が同じ run に出る行がない。足す行は PC1
- N43・N44: 6 本以上の枝が上限になる場合の注記。足す行は PC3
- N51(cycle 2 の G04): ちょうど 5 本の枝。足す行は PB3 か PC4。`mergeNamedBranches` を直接引く表にすれば、1 つで両方の注記に効く
- N34 と cycle 2 の L05: gh が途中で見つからなくなる場合。足す行は PC2 と PB2。現実にはまず起きない
- cycle 2 から残る S04、O03・O04、P04・P05・O16。cycle 3 はこの部分のテストを変えていない。足す行は PB4、PB5 か PB5b、PB8 と PB5_PB6
- help と skill の `--limit 1000` は手で書いた数で、どのテストも読まない(N70、self-review C3-3)
- 1000 本を取る時間: 本物の gh で 9〜10 s かかり、`GHTimeout` の 15 s に近い。テストの偽の gh は 1000 本を一度に返すので、時間は見ない。timed out になっても、結果は PR なしで注記が付く
- 既知の穴(設計の範囲で、cycle 2 と同じ): origin の URL が owner/repo を持たないときは確かめない、別の repository にその名前の PR がないときは注記が出ない、host を比べない。上限で PR を使わなかったブランチは `--json` の entry に印がなく、`notes` の文だけが知らせる(verify の D6、`docs/tech-debt/README.md` の 238 行)

## Cycle 4

- Date: 2026-10-14(JST 12:46〜13:30)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 4 回目で最後(cross-review cycle 3 の ACTION_REQUIRED 1 件と WORTH_CONSIDERING 1 件を直すために上限を 3 から 4 に上げた回。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)。self-review の cycle 4(LOW 4 件、C4-1〜C4-4、Merge: yes)と verify の cycle 4(pass、D8・D9)のあとに走らせた
- Scope: HEAD 6d64e96a(base ac34ccf3)。cycle 4 でコードとテストを変えたのは 813d6d2d と 1052a12a で、`git diff 999e0394..HEAD -- internal/` は 4 ファイル、+158/-22。中身は、gh が読む repository と origin のものが違うときの注記に 2 つの直し方(改名なら `git remote set-url origin <its URL>`、そうでなければ `gh repo set-default <origin の owner/repo>`)を並べること、読めない gate の記録を、結びつきの枝、なければその org の最新の読める記録の枝(`gatedBranch`)、の順に付けること、付け先のない記録の注記の文、help の文、テスト 3 本(subtest 4 本)、期待と doc の書き換え 3 か所。ほかのコミットは計画、報告、insight event だけ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は cycle 3 と同じで(macOS arm64、go1.26.0 と go1.25.8、git 2.49.0、`/bin/sh` は bash 3.2.57)、gh は 2.102.0
- Evidence(どれもコミットする。wrapper が書いた `docs/evidence/verify-2026-10-14-034645.log` は gitignore の対象なのでコミットしない):
  - `docs/evidence/test-2026-10-14-org-merge-order-c4-run-test.txt`: `./scripts/run-test.sh`(changed)の出力を cycle 1〜3 と同じ方法で抜いたもの、`(cached)` だった 5 パッケージの打ち直し、cycle 3 の行との diff
  - `docs/evidence/test-2026-10-14-org-merge-order-c4-go.txt`: go1.25.8 と go1.26.0 での `./internal/org/... ./internal/cli/...` のパッケージごとの数、`-race` の結果、SKIP の行、2 つの toolchain の間と cycle 3 との間のテスト名の diff、足した 3 本と書き換えた 3 本の行、merge-order のテスト 53 本の行、前に揺れたテストの行、merge-order の 2 ファイルの関数ごとの coverage
  - `docs/evidence/test-2026-10-14-org-merge-order-c4-mutation.txt`: cycle 4 のコードの mutant 29 件と対照の X0、順の mutant を subtest ごとに打った結果、CLI のテストだけで打った結果、probe の行の結果とその source、cycle 3 の mutant 44 件と probe 12 本を HEAD で打ち直した結果、mutant を作る script と打つ script
  - `docs/evidence/test-2026-10-14-org-merge-order-c4-rename.txt`: GitHub で改名した repository(vuejs/vue-next から vuejs/core)を、origin の古い URL のまま本物の gh で打った、読むだけの確認
  - insight event: lead の指示どおり書いていない
- テストもコードも変えていない。mutant の写し、probe のテスト、偽の `gh`(canary)、git と gh の呼び出しを記録する wrapper、一時の台帳と repo は scratchpad に置き、`go test -overlay` と PATH で読ませた。worktree に書いたのは、この報告の追記と `## Verdict` の書き換え、上の evidence 4 本だけで、mutation と probe を終えて evidence を置く前の `git status --porcelain` は空だった。テストの前後で `~/.config/herdr` の下に 12:46 より新しいファイルはなく、main のチェックアウトの `git status --short` は空のまま、`$TMPDIR` に `ralph-verify-*` は残っていない。herdr の server も osascript も起こしていない

### Test execution(cycle 4)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed、go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(cycle 1〜3 と同じ) | 882 s、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/... ./internal/cli/... -count=1 -v -coverprofile` | org: top-level 684、subtest 1,369。org/driver: 48、89。org/protocol: 15、56。cli: 632、519 | org の top-level 683 本、ほかはすべて | 0 | org 1 | 297 s(org 130.6 s、cli 293.6 s)、rc 0 |
| `go test ./internal/org/... ./internal/cli/... -count=1 -v`(go1.26.0) | 上と同じ | 上と同じ | 0 | org 1 | 314 s(org 152.8 s、cli 312.5 s)、rc 0 |
| うち merge-order のテスト(`TestMergeOrder_*` 38 本、`TestOrgMergeOrder_*` 15 本) | top-level 53、subtest 77 | 両方の toolchain で 53、77 | 0 | 0 | - |
| `go test -race ./internal/org/... ./internal/cli/... -count=1`(go1.26.0) | 4 パッケージ | 4 パッケージすべて ok、`DATA RACE` は 0 件 | 0 | - | 281 s |
| `(cached)` だった 5 パッケージを `-count=1`(go1.26.0) | config、insights、org/protocol、scaffold、upgrade | 5 パッケージすべて ok | 0 | - | 合わせて約 7 s |
| mutation: cycle 4 のコード(`go test -overlay`、go1.26.0) | 29 件(どれも build できた)と対照の X0 | red 25 件 | 生き残り 4 件(下の「Mutation(cycle 4)」) | - | 566 s |
| mutation: cycle 3 の mutant を HEAD のテストで打ち直し | 44 件 | red 35 件 | 生き残り 9 件(cycle 3 と同じ) | - | 847 s |
| probe(overlay だけ。cycle 4 の 3 本、cycle 3 の 12 本。コミットしない) | HEAD と生き残り 11 件 | HEAD では 15 本すべて通る。閉じられる生き残り 10 件は、どれも狙った probe で落ちる | - | - | - |
| 改名した本物の repository の確認(読むだけ) | HEAD の binary 5 回、修正前の binary 1 回、`gh repo set-default` 2 回、`git remote set-url` 2 回(一時の repo) | すべて期待どおり | 0 | - | - |

- shell は 41 本すべてが `OK` で終わり、集計の `FAIL:` の行はどれも `FAIL: 0` だった。抜き出した行から行番号を外して cycle 3 の evidence と diff を取ると、違うのは時刻、Go の所要時間、末尾の行だけで、suite ごとの集計の行はすべて同じ。Skipped は cycle 1〜3 と同じ `tests/test-secret-scan-branch.sh` の 1 件
- cycle 3 と比べると、top-level は org 681 → 684(+3)、subtest は org 1,365 → 1,369(+4)で、cycle 4 で足した 3 本(`TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranch`、`TestMergeOrder_UnreadableRecordPrefersTheBindingToAGatedBranch`、`TestMergeOrder_RepositoryNoteNamesTheFixForARenameAndForAnotherRepository`)と、その subtest 2 本ずつの数と合う。cli は 632 と 519 で変わらない。テスト名の一覧(PASS・SKIP の行から所要時間を外して並べ替えたもの)を cycle 3 の go1.25.8 の一覧と比べると、違いはこの 7 行の追加だけで、消えた名前はない。go1.25.8 と go1.26.0 の一覧は同じ
- 期待か doc を書き換えた 3 本(`TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt`、org と CLI の `PRFromAnotherRepositoryThanOriginsIsLeftOut`)は、両方の toolchain で PASS
- `run-test.sh` の中の Go は `internal/cli`(204.0 s)、`internal/org`(92.3 s)、`internal/org/driver`(0.723 s)を流し直し、残りの 5 パッケージは `(cached)` だった。`internal/org` を import するのは `internal/insights` だけ(`go list -deps`)で、5 つとも `-count=1` で打ち直して ok
- `-race` は org と cli の全体に 1 回付けて、race の報告は 0 件
- 前に揺れたテスト(`TestRunDoctor*`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どちらの toolchain でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は 2.81 s と 3.80 s で、+5 s の subtest は出なかった。load average は 5.6〜7.5 だった
- 本物の `gh` に届かないこと: 呼ばれたら記録して exit 97 で終わる偽の `gh`(canary)を受け継ぐ PATH の先頭に置き、`run-test.sh`、2 回の全体の実行、`-race`、mutation と probe のすべての実行を走らせた。canary の記録は 0 件。下の改名の確認だけは、テストではなく HEAD の binary を、記録用の wrapper を通して本物の gh で打った

### Coverage(cycle 4)

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.1%(cycle 3 と同じ)、`internal/cli` 86.8%(同じ)、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%
- `internal/org/mergeorder.go` は 309/310 文(99.7%、cycle 3 は 308/309)、`internal/cli/org_merge_order.go` は 39/43 文(90.7%、同じ)。通らない文は cycle 1〜3 と同じ 5 つ(`Exec` が nil のときの `run = realMergeOrderExec`、CLI の 4 つ)
- cycle 4 で書き直した関数(`collectMergeTargets`、`mergeLedgerNotes`、`mergeOriginRepo.check`)は、どれも 100%。それでも下の 3 件が生き残った。穴は通らない行ではなく、テストの読めない記録がどれも `branch=` を持たないことと、注記を前の部分だけで比べることにある
- Branch: 測っていない(cycle 1〜3 と同じ)。代わりに mutation で分岐ごとの判定を確かめた

### Mutation(cycle 4)

cycle 4 のコードに 29 件の mutant を作った。どれも build でき、25 件が red、4 件が生き残った。対照として、変えていないファイルを overlay で読ませる X0 を同じ runner で打ち、org と CLI のテストは通った(runner が mutant でない写しを落とさないことの確認)。

| 対象 | mutant | red | 生き残り |
| --- | --- | --- | --- |
| repository の注記の直し方と順(P01〜P08b: cycle 3 の文に戻す、set-default を外す、改名の手を外す、順を入れ替える、`renamed to` の 2 つを入れ替える、set-default に gh の repository を書く、set-url の remote を変える、owner で場合を推し量る 2 通り) | 9 | 9 | なし |
| help の文(P09) | 1 | 0 | P09 |
| `gatedBranch` の代入(G01〜G06: 代入しない、最初の読める記録を残す、読めない記録でも自分の `branch=` で上書きする、`branch=` を持つ読めない記録も代入する、読めない記録の自分の `branch=` を先に使う、org ごとでなく 1 つにする) | 6 | 4 | G04、G05 |
| `cmp.Or` の順(O01〜O05: 入れ替える、結びつきだけ(cycle 3 の付け先)、読める記録だけ、後に来た方を使う、読める記録が org の結びつきを動かす) | 5 | 5 | なし |
| 付け先のない記録の数と注記(U01〜U06: 数えない、2 つずつ数える、cycle 3 の文に戻す、0 でも出す、文の末尾を落とす、`gatedBranch` で付けた記録も数える) | 6 | 5 | U05 |
| ファイルごと戻す(R0: 999e0394 の `mergeorder.go`、R1052: 813d6d2d の `mergeorder.go`) | 2 | 2 | なし |

- 注記の 9 件は、どれも org の `TestMergeOrder_RepositoryNoteNamesTheFixForARenameAndForAnotherRepository` が落とした。このテストは注記の全文を比べるので、直し方の順を入れ替える P04 と、`renamed to` の前後の repository を入れ替える P05 は、このテストだけが落とした(元からある `TestMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut` は部分一致で見る)。owner が同じかどうかで改名か別の repository かを推し量る P08a(owner が違えば set-default だけ)と P08b(同じなら set-url だけ)は、同じ owner の改名と別の owner の 2 つの subtest がそれぞれ落とした。どちらの場合でも 2 つを並べる、という計画の決定がテストで固まっている。CLI の `TestOrgMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut` だけで P01〜P07 を打っても 7 件とも落ちたので、CLI の部分一致は org のテストとは別にこの文を固めている
- `gatedBranch` の代入を外す G01、最初の読める記録を残す G02、読めない記録でも自分の `branch=`(`moGarbled` では空)で上書きする G03、org ごとでなく 1 つの値にする G06 は、`TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranch` が落とした(G06 は `TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` も)。G02 は org-e の 2 本(feat/e と feat/e2)の行が、G06 は org-d の「付け先なし」の数が見分ける
- `cmp.Or` の 2 つを入れ替える O01 は、`TestMergeOrder_UnreadableRecordPrefersTheBindingToAGatedBranch` の 2 つの subtest が両方落とした。指示にあった「その org の最新の読める記録が、別の枝への結びつきのあとに来た」場合を壊す 2 件、後に来た方を使う O04 と、読める記録が org の結びつきを動かす O05 は、subtest「the binding before the gate record」だけが落とし、「the gate record before the binding」では通った(evidence の B)。2 つの並びを両方持つことが効いている。結びつきだけを見る O02(cycle 3 の付け先)は新しいテストが、読める記録だけを見る O03 は既存のテスト 2 本も落とした
- 数えない U01、2 つずつ数える U02、0 でも注記を出す U04、`gatedBranch` で付けた記録も数える U06、cycle 3 の文に戻す U03 は red。U03 を落としたのは、新しいテストの `HasPrefix` だけだった
- ファイルごと戻す 2 件も red。R0 は新しい 2 本と元からある repository のテストが、R1052 は `TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranch` が落とした

生き残り 4 件のうち 3 件は、HEAD で通る probe の行が落とす(evidence の C の PD1、PD1b、PD2)。足すと確信が増える順に挙げる。

- G05(PD1、PD1b)と G04(PD1): `branch=` を持つ読めない記録の付け先。G05 は読めない記録の自分の `branch=` を結びつきと読める記録より先に使い、G04 は event の名前と reason が合わない記録(details は読めて `branch=` を持つが readable ではない)で `gatedBranch` を上書きする。コミットされたテストの読めない記録は、どれも `moGarbled`(details は `head=not-a-sha garbled`)で `branch=` を持たないので、自分で別の枝を名指す読めない記録の行がない。PD1 は、結びつきのない org-g が feat/g で読める記録を書いたあと、feat/g2 を名指す食い違った記録を書く台帳で、HEAD は feat/g を unreadable にし、feat/g2 を対象にしない。G04 と G05 では feat/g が古い pass で ready に戻り、feat/g2 が unreadable になる。C8 が防ぐ動きそのものだが、食い違った記録は `ralph org gate` が書かない(記録の event は reason から作る)ので、手で書いた台帳にしか出ない。PD1b は結びつきのある org で同じ並びを作り、G05 だけを落とす。PD1b は 999e0394 の `mergeorder.go`(R0)でも通るので、結びつきのある側の規則は cycle 1 からあり、cycle 4 の前から固められていなかった
- U05(PD2): 付け先のない記録の注記の末尾「, so they are attached to no branch」を落とす。テストは `HasPrefix` と `Contains` で前の部分だけを見る。PD2 は全文を比べる。文だけの違い
- P09(probe なし): help から改名の文を落とす。help の文を読むテストはない(cycle 3 の N70 と同じ)

### cycle 3 の生き残りは HEAD で red か

cycle 3 の mutant の一覧(44 件。build できない N20 は N20b で数える)を HEAD のテストで打ち直した。cycle 4 は `lookupMergePRs`、`withNotes`、`mergeGHLimitNotes`、`mergeNamedBranches` に触れておらず、どの置き換えの文字列も HEAD で 1 回だけ一致したので、書き直しは要らなかった。35 件が red で、生き残りは cycle 3 と同じ 9 件。指示にあった N30・N31・N33、N34、N43・N44、N51 は、どれも生き残ったままで、red になったものはない。cycle 3 で red だったものは、全部 red のまま。

| cycle 3 の生き残り | HEAD で | 落とす probe(コミットしない) |
| --- | --- | --- |
| N30・N31: 注記の並び | 生き残り | PC1、PC2 |
| N33: 不一致で止まるときに上限の注記を捨てる | 生き残り | PC1 |
| N34: gh がないときに上限の注記を捨てる | 生き残り | PC2 |
| N43・N44: 6 本以上の枝が上限になるときの注記 | 生き残り | PC3 |
| N51: ちょうど 5 本で「and 0 more」が付く | 生き残り | PB3、PC4 |
| N11(`>=` を `==`)、N70(help) | 生き残り(本物の gh では同じ動き、文だけ) | なし |

- cycle 3 の probe 12 本(cycle 2 の PB1〜PB8、cycle 3 の PC1〜PC4)を写しのまま HEAD で打つと 12 本とも通り、生き残り 7 件はそれぞれ cycle 3 と同じ probe で落ちた(evidence の E)。cycle 4 の注記の文の変更は、これらの probe の期待に触れない
- cycle 4 で足した 3 本は、注記の並びも上限になる枝の本数も扱わないので、これらを落とさないのは予想どおり

### 改名した本物の repository(読むだけ)

verify の cycle 4 が「足すと確信がいちばん増える確認」に挙げ、self-review の cycle 4 の follow-up 4 が勧めた確認を、改名の分かっている公開の repository で打った。GitHub には何も書いていない。gh は `pr list`、`api`(GET)、`repo set-default`(一時の repo の `.git/config` だけを書く。今回は断られて何も書かなかった)だけを打った。gh のアカウントは切り替えていない。

- vuejs/vue-next は vuejs/core に改名されている。`gh api repos/vuejs/vue-next` は vuejs/core を返し、`git ls-remote --symref https://github.com/vuejs/vue-next.git HEAD` は GitHub の転送を通って `ref: refs/heads/main	HEAD` を返した
- 一時の repo(`git init` と `git remote add origin https://github.com/vuejs/vue-next.git` だけで、コミットはない)と、vuejs/core の 2 本の枝を名指す一時の台帳を作った。edison/transition-comment-root(PR #15853、main に merge 済みで枝は消えた)と、edison/refactor/slot-provider-owned-t06(PR #15858、minor への open)。どちらの PR も vuejs/core 自身の枝から出ている
- HEAD の binary(go1.25.8 で作った)を `--all` で打つと、gh が返した PR #15853 の url は `https://github.com/vuejs/core/pull/15853` で、新しい名前だった。merge-order は PR を使わず、注記は「gh reads vuejs/core, not origin's vuejs/vue-next: no PR is used, so …; if vuejs/vue-next was renamed to vuejs/core, point origin at it (git remote set-url origin <its URL>); otherwise run gh repo set-default vuejs/vue-next in this repository」だった。gh は 1 本目で止まり、get-url は 1 回。分類は t06 が ready(PR なし)、transition-comment-root が no_remote。`--json` の `notes` も同じ文で、終了コードはどれも 0
- 修正前(999e0394)の binary は、同じ台帳で同じ分類を出し、注記の終わりは「run gh repo set-default vuejs/vue-next in this repository」だけだった。その手を打つと、gh は「vuejs/vue-next does not correspond to any git remotes」と言って rc 1 で終わり、何も書かなかった。そのあと HEAD を打っても注記は変わらない。cross-review cycle 3 の指摘(改名した repository では、古い名前での set-default が効かない)を本物の gh で再現した
- 注記の 1 つめの直し方のとおり `git remote set-url origin https://github.com/vuejs/core.git` を打つと(一時の repo の `.git/config` だけを書く)、HEAD は PR を使った。t06 は PR #15858 が minor に向くので wrong_base、transition-comment-root は PR #15853 で merged になり、`--all` を付けないと隠れて「1 branch(es) whose PR is merged or closed are not shown」と出た。gh は 2 本とも打たれた
- 最後に origin を古い URL に戻した。台帳の sha256 は前後で同じ。gh は PR を引くたびに `git remote -v` と `git config --get-regexp ^remote\..*\.gh-resolved$` を自分で打った(cycle 2 と同じ)

### Failure analysis(cycle 4)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks(cycle 4)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| GitHub で改名した repository では、注記が勧める `gh repo set-default <origin の古い名前>` が効かない(cross-review cycle 3 の ACTION_REQUIRED #1) | 直った(偽の gh と本物の gh の両方) | P01〜P08b が red で、P01〜P07 は CLI のテストだけでも red。本物の vuejs/vue-next で、HEAD は 2 つの直し方を出し、1 つめを打つと PR が使われた。修正前の注記の手は gh が断った |
| gate の記録だけが名指すブランチが、読めない新しい記録のあとに古い gate_passed で ready になる(cross-review cycle 3 の WORTH_CONSIDERING #2) | 直った | G01〜G03、G06、O01〜O05、U01〜U04、U06、R0、R1052 が red。verify の V5・V7 は binary で同じことを見た |
| cycle 3 の mutant で red だったもの | HEAD のテストでも red | 44 件のうち 35 件が red。生き残り 9 件は cycle 3 と同じ |
| Go 1.25 だけで起きる違い | 差はない | go1.25.8 と go1.26.0 で、数もテスト名も同じ |
| 前に揺れたテスト | 揺れなかった | 2 回の全体の実行と `-race` の実行のどれでも PASS |
| テストが本物の `gh` に届かない | 届かない | canary の記録は `run-test.sh` を含めて 0 件 |

### Test gaps(cycle 4)

覆われていない場合を、足すと確信が増える順に挙げる。どれもテストの失敗ではない。

- G04・G05: 自分で別の枝を名指す読めない記録(event の名前と reason が合わない記録)の付け先。足す行は PD1(結びつきのない org)と PD1b(結びつきのある org)。手で書いた台帳にしか出ない
- U05: 付け先のない記録の注記の全文。足す行は PD2
- cycle 3 から残る N30・N31・N33、N34、N43・N44、N51 と、cycle 2 から残る S04、O03・O04、P04・P05・O16。cycle 4 はこの部分のテストを変えていない。足す行は cycle 3 と cycle 2 の Test gaps のとおり
- P09: help の改名の文はどのテストも読まない(N70 と同じ)
- self-review の C4-1(結びつきのない org の Detail が、断られる `ralph org gate --org-id` を勧める)は、テストが今の文を固めている(`TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranch` の feat/c の Detail)。直すときはその期待も書き換える
- 改名の確認は vuejs/vue-next の 1 件だけで、origin は https の URL だった。ssh の古い URL と、owner が変わる移管は打っていない。merge-order の側は owner と repository の名前を文字で比べるだけなので同じ注記になるはず。未確認です
- 改名と別の repository は今も見分けない(計画の決定)。注記は 2 つの直し方を並べるだけで、`--json` の entry に印はない(cycle 3 の D6 と同じ形)
