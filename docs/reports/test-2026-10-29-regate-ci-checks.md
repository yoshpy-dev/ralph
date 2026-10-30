# Test report: regate-ci-checks

- Date: 2026-10-30(JST 12:54〜13:45)
- Plan: `docs/plans/active/2026-10-29-regate-ci-checks.md`(AC1〜AC10、Test plan)。issue #264
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの cycle 1
- Scope: HEAD 1994eba2(base は origin/main c03647f9、`git diff origin/main...HEAD`)。振る舞いのテストだけを実行した。静的解析は /verify が済ませている。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.49.0。ほかのセッションの `go test ./...` が同時に走っており、1 分の load average は 5.51〜17.24 の間で動いた
- Evidence: `docs/evidence/test-2026-10-29-regate-ci-checks.log`(gitignore の対象で、コミットしない。`run-test.sh` の全出力、30 秒ごとの load、AC1 の削除行、変異の diff と結果)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-30-035635.log`(名前の時刻は UTC)
- Insight event: `docs/insights/events/2026-10-30-regate-ci-checks.jsonl` に test の行を 1 行足した(`--verdict pass --cycle auto`、cycle 1)
- 追跡しているファイルは、この報告と insight event のほかは変えていない。変異は scratchpad の写しを `go test -overlay` で当てた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS="-timeout=30m $(go env GOFLAGS)" ./scripts/run-test.sh`(scope は changed で、`unclassified:ralph.toml` のため full に落ちた。pack は golang) | shell 45 本 + Go 10 パッケージ | shell 45 本(`PASS` の行 4,796)、Go 10 パッケージ(`(cached)` なし) | 0 | shell の 1 行(前からある「git 2.41 未満の実機」の行) | rc 0、2,256 秒(12:56:35〜13:34:11)。load 5.51〜17.24 |
| 2 | `go test -count=1 -v -run '^TestRegateCI_' ./internal/org/`(変異なし) | トップレベル 28(サブテスト 102) | 28(102) | 0 | 0 | rc 0、256 秒。load 13.88→7.81 |
| 3 | 変異 8 つ(下の「変異の確かめ」) | 8 実行 | - | 8 実行とも rc 1 | - | 20〜121 秒。load 7.56〜14.06 |
| 4 | `go test -count=1 -coverprofile -run '<ci の regate、prchecks、CI の記録を読むテスト>' ./internal/org/` | 1 パッケージ | 1 | 0 | 0 | rc 0、241 秒。load 7.05→6.38 |

1 の Go のパッケージごとの時間: `cmd/ralph` 0.864 秒、`internal/cli` 927.023 秒、`internal/config` 9.869 秒、`internal/gitroot` 6.703 秒、`internal/insights` 2.164 秒、`internal/org` 1,196.820 秒、`internal/org/driver` 5.769 秒、`internal/org/protocol` 3.823 秒、`internal/scaffold` 1.049 秒、`internal/upgrade` 2.089 秒。

`run-test.sh` は 10 分の Bash の上限を越えるので、perl の fork と setsid で切り離し(INT・HUP・QUIT・TERM を DEFAULT に戻し、標準の入出力は /dev/null)、終わりは `caffeinate -w` で前景で待った。shell の SKIP の行は前と同じ 1 行で、シグナルを使う shell のテストは SKIP しなかった。2〜4 は 1 と時間が重なっている(2 と変異は 1 の shell の段と Go の段の最中、4 は 1 のあと)。時間の条件で落ちたテストはなく、打ち直しはしていない。

## Coverage

- Statement: 4 の実行で `internal/org/regate_ci.go` 91.6%(286 文のうち 262)、`internal/org/prchecks.go` 100%(50 文のうち 50)。`regate_ci.go` の関数 27 個の平均は 94.9%。80% 未満は `regateCIPushSaid` 75.0% と `writeGateChecksLog` 77.8%
- Branch: 測っていない(Go の標準の道具にない)
- Function: 上の 27 関数。どれも 1 回以上呼ばれた
- Notes: 4 は ci の regate のテストだけを打った値で、`Regate_` の local のテストと `internal/cli` は含まない。`regate.go`・`regate_merge.go`・`gate_record.go` の変更は local の道と混ざっているので、ファイルごとの値を出していない。`regate_ci.go` の通らなかった文は「Test gaps」の 4 に書いた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | - | - | - |

1〜2 と 4 で落ちたテストはない。3 の失敗は変異によるもので、下の節に書いた。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC1: 既定の `local` の regate が今と同じに動く(origin/main にあった org 層の regate のテストが、変えずに通る) | Pass | `git diff --numstat origin/main...HEAD` は `internal/org/regate_test.go` が +51/-0、`internal/org/regate_merge_test.go` が +114/-0 で、消えた行も書き換えた行もない。どちらも 1 の `internal/org`(1,196.820 秒)の中で通った。テストの helper を持つほかのファイルで行が消えたのは `internal/org/gate_test.go` の 7 行だけで、roster の読み手のテストに CI の event を足したもの(regate のテストは使っていない) |
| AC1 の CLI 側: `internal/cli/org_regate_test.go` の書き換え(+516/-5) | Pass(文言の変化は意図どおり) | 消えた 5 行の内訳: (1) local の `UP_TO_DATE:` の行の期待 2 か所(`:160` の `TestOrgRegate_MergesPushesAndThenIsUpToDate`、`:412` の `TestOrgRegate_PrintRegateResult`)で、「is a gate_passed of HEAD」を「is a pass of HEAD」にした。CLI の出力(`internal/cli/org_regate.go:358`)が、記録の種類によらない文になったため。local の regate も `gate_ci_passed` を UpToDate と読むので、文を合わせた(計画の S7、verify の I1)。(2) help を固定する 1 行(`:806`〜)で、「While it runs, … are refused」の後ろに ci の待ちで `regate.lock` を外すあいだの例外を足して延ばした(self-review の L6、F2)。(3) `TestOrgRegate_PrintRegateResult` の注釈 2 行を、終了コードと ci の行を書く文に直した。ほかの期待は変わっていない |
| merge と merge-order(`internal/org`、`internal/cli`) | Pass | 1 の両パッケージ。`TestMerge_MergesABranchWhoseLatestRecordIsAGateCIPassed`、`TestMergeOrder_ReadsTheCIRecordsOfRegate` が CI の記録を読ませる |
| PR の前の gate(`ralph org gate`) | Pass | 1 の `internal/cli` と `internal/org`。CLI の差分は `printGateFailDetail` の引数と `ci_failed` の表示だけ |

## 変異の確かめ

どれも追跡しているファイルを変えず、scratchpad の写しを `go test -count=1 -v -overlay <json>` で当てた。8 つとも build が通り、テストが落ちた。変異のない同じコードは 2 の `^TestRegateCI_`(28/28)と 1 の `internal/org` 全体で通った。

| # | 変異(手順の番号) | `-run` | 結果 | 落ちたテスト(落ち方) |
| --- | --- | --- | --- | --- |
| m1a | 1: `readRegateChecks`(`regate_ci.go:461`)の `case len(c.pending) > 0:` を `case false && …` にし、pending の check を pass と同じに扱う | `^TestRegateCI_(GateOnly\|MergePath)` | rc 1、121 秒、load 7.56→14.06 | `GateOnlyRecordsNothingWhileTheChecksAreNotDone/a_check_is_pending`(`Outcome:"gated", Passed:true` を記録した) |
| m1b | 1 の別の形: `regateChecks`(`regate_ci.go:530`)の `case read.pending != "":` を外し、終わっていない checks を pass として記録する | 同上 | rc 1、80 秒、load 14.06→10.76 | `GateOnlyRecordsNothing…` のサブテスト 6 つすべてと `MergePathPushesTheMergeCommitBeforeTheChecks`(どれも pass を記録した) |
| m2a | 2: 最初の PR の head の確かめ(`:436` と `:439`)を外す | 同上 | rc 1、81 秒、load 9.09→9.90 | `GateOnlyRecordsNothing…/the_PR's_head_is_not_HEAD_yet` と `MergePathPushes…`。checks を読んだあとの head の読み直し(`:473`)が止めるので結果は pending のままで、落ちたのは pending の理由の文と gh の呼び出しの一覧 |
| m2b | 2: checks を読んだあとの head の読み直し(`:473`)を外す | 同上 | rc 1、85 秒、load 9.90→8.62 | `GateOnlyRecordsNothing…/the_PR's_head_moved_while_its_checks_were_read`(pass を記録した) |
| m2ab | 2 の全体: m2a と m2b を合わせ、PR の head を一度も確かめない | 同上 | rc 1、76 秒、load 7.67→10.84 | `GateOnlyRecordsNothing…` のサブテスト 2 つ(`Outcome:"gated", Passed:true`)と `MergePathPushes…`(`Outcome:"merged", Passed:true`)。古い head の checks で pass を記録する |
| m3a | 3: `Regate` の gate だけの道(`regate.go:550`)を `if false && ci != nil` にし、ci でもスクリプトを走らせる | `^TestRegateCI_(GateOnly\|LocalChecks)` | rc 1、75 秒、load 10.85→10.28 | `GateOnlyRecordsFromThePRsChecks`・`GateOnlyRecordsNothing…`・`GateOnlyRefusesWhatTheChecksDoNotTell` のサブテスト 20 個(スクリプトの結果で `gated`、pass を記録した)。`LocalChecksRunTheScriptsAndCallNoExec` は通った(local の道なので想定どおり) |
| m3b | 3: merge の道(`regate_merge.go:277`)を `if false && ci != nil` にし、ci でも merge のあとにスクリプトを走らせる | `^TestRegateCI_MergePath` | rc 1、20 秒、load 10.28→9.08 | `MergePathPushesTheMergeCommitBeforeTheChecks` と `MergePathRecordsTheChecksOfThePushedMergeCommit` のサブテスト 2 つ(スクリプトの pass で `merged`) |
| m4 | 4: `regateCIPush`(`regate_ci.go:297`)で、読み返した先頭が merge の前の先頭のときの `regateUndoMerge` を消す | `^TestRegateCI_AFailedPush` | rc 1、30 秒、load 9.08→9.72 | `AFailedPushIsDecidedByOriginsTipReadBack` の `origin's_tip_is_still_the_head_merged_onto` と `a_push_ended_by_a_signal,_origin's_tip_still_the_head_merged_onto`、`AFailedPushWhoseUndoFailsNamesWhatEachGitSaid`(`Undone` が false。同じテストは HEAD が merge の前に戻ったことも確かめる) |

手順の 4 つはどれも、`RegateCI` の narrow な `-run` だけで落ちた。`Regate_` を足す必要はなかった。m2a だけでは記録は起きず、PR の head の確かめは 2 段(checks の前と後)で守られている。

## Test gaps

計画の Test plan の項目ごとに、対応するテストを確かめた。単体(設定の既定と検証の 9 行 `TestLoad_OrgGate`、`ralph config get` の 2 行、記録の書き読み `TestGateRecordDetails_CIRecord`、merge-order の文言 `TestMergeOrder_ReadsTheCIRecordsOfRegate`)、結合の 14 項目(passed、failed、pending、0 件、読めない、PR の head が古い、PR がない、gh がない、GitHub でない、push の失敗の分かれ、`--wait-checks` の偽の時計、待つあいだの lock、取り直しの確かめ)、push の失敗の 4 つの分かれ、AC10、同じブランチの regate が 2 つ、push のあとの割り込みには、それぞれテストがある。次の 4 つは、テストがないか一部だけ。

1. Rollback の確かめ(計画の Test plan の「今の event 名だけを選ぶ読み方が `gate_ci_*` を読まないこと」): event 名が `gate_passed`・`gate_failed` と違うことは `TestGateRecordDetails_CIRecord` が固定する。この PR の前の形の読み手(`case EventGatePassed, EventGateFailed`)や origin/main の binary に、`gate_ci_*` を含む台帳を読ませるテストや実測はない(verify の report も実測していないと書く)。いちばん小さい確かめは、origin/main を `git archive` で取り出して build した binary で、`gate_ci_passed` だけを持つ台帳に `ralph org merge-order` を打ち、そのブランチが ready にならないことを見る 1 回の観察
2. Edge case「待つあいだに既定のブランチが進む(記録の base が古くなり merge-order が `rebase` と言う)」: merge-order の分類は、組み立てた台帳の行 `rebase: origin/main moved after the gate_ci_passed`(`internal/org/mergeorder_test.go`)が確かめる。regate の待ちの最中に origin/main を進め、記録の base が古い先頭になり merge-order が `rebase` を返すまでを通すテストはない。足すなら、`TestRegateCI_WaitChecksWaitsUntilTheChecksAreDone` に、眠りのあいだに origin の main へコミットを push する行を 1 つ足し、記録の `base=` が前の先頭で、`fx.mergeClass` が `rebase` になることを見る
3. 本物の GitHub の PR と CI での `ci` の regate(PR の head の更新の遅れを含む): テストは偽の gh だけ。計画の Open questions と仕様の注のとおり未確認
4. `regate_ci.go` の通らなかった文(4 の実行): push の前に remote-tracking ref を読めないときの undo(`:256-262`)、ci の push が `regateGitTimeout` を越えたとき(`:279`)、push の失敗のあとの読み返しの最中の割り込み(`:291-292`)、log や記録の I/O の失敗、取り直しの確かめで git が失敗したとき(`:817-849`)。どれも失敗の道で、行を足すなら `TestRegateCI_AFailedPushIsDecidedByOriginsTipReadBack` に push が時間を越える行(regate の local のテストにある時間切れの stub と同じ形)を 1 つ足すのが小さい

## Verdict

- Pass: `run-test.sh` が rc 0(shell 45 本、Go 10 パッケージ)。AC1 の org 層のテストは消えた行も書き換えた行もなく通った。変異 8 つがすべて落ち、変異のないコードは通った
- Fail: 0
- Blocked: 0
- Verdict: pass。テストの穴は 4 つ(Rollback の実測、待つあいだに既定のブランチが進む場合の通しのテスト、本物の CI、失敗の道の数か所)で、どれも PR を止めるものではない
