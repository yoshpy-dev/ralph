# Test report: doctor-alias-herdr-guard

- Date: 2026-11-19。cycle 1(`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md`
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `b4e60570`(verify report の commit のあと。worktree は clean)、base `a8c5e289`。verify run 2 の HEAD `3be7cde6` からの差は verify report、計画、insight event の 3 ファイルだけで、Go のファイルは変わっていない。`./scripts/run-test.sh` を 1 回、worktree のルートから打った。scope は wrapper の既定の changed で、選ばれた pack は golang(`changed_languages`)。local verifier は scope によらず `tests/*.sh` を全部回す。そのほかに `go test -count=1 -v ./internal/cli/ -run 'ShellAlias'` を 1 回打ち、AC ごとの根拠にした
- Evidence: `docs/evidence/test-2026-11-19-doctor-alias-herdr-guard.log`(6,440 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、負荷の読み、`-v` の run、coverage の run、穴の probe のテストの順に入れた

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-19 21:47:43 〜 22:13:10 JST、1,527 s、rc 0、`==> All verifiers passed.` |
| local verifier(`tests/*.sh` 45 本) | 45 本 | 45 本 | 0(FAIL の行は `FAIL: 0` の集計と、検出を確かめる PASS の行の中の文字だけ) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`git 2.41 より古い本物の git` の場合(手元は git 2.49.0)。reseat-notices の report と同じ 2 つ | 約 12 分 |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` のパッケージもない | 下の表 |
| `go test -count=1 -v ./internal/cli/ -run 'ShellAlias'` | top-level 98(`=== RUN` 343 行、subtest を含む) | 98(343) | 0 | 0 | 21:47:53 〜 21:47:58、4.1 s |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.765 s |
| `internal/cli` | ok | 751.540 s |
| `internal/config` | ok | 15.018 s |
| `internal/gitroot` | ok | 2.609 s |
| `internal/insights` | ok | 1.881 s |
| `internal/org` | ok | 879.591 s |
| `internal/org/driver` | ok | 3.097 s |
| `internal/org/protocol` | ok | 2.131 s |
| `internal/scaffold` | ok | 3.393 s |
| `internal/toolchain` | ok | 23.136 s |
| `internal/upgrade` | ok | 4.805 s |

実行の条件:

- 負荷(`sysctl -n vm.loadavg` の 1 分値、60 s ごと): 6.80〜17.01。22:10 ごろから、ほかのセッションの `go test ./internal/org/ ./internal/cli/` と `go test ./...`(worktree `org-ssh-alias-host`)が Go の段に重なった
- `GOFLAGS` は足していない。golang の pack(`packs/languages/golang/verify.sh:115`)が `-timeout=30m` を GOFLAGS の先頭に入れる。`internal/org` の 879.6 s は上限の 49%、`internal/cli` の 751.5 s は 42%
- 起動: scratch(repo の外)の runner を `perl` の fork と `setsid` で切り離して動かした。シグナルは INT、HUP、QUIT、TERM を DEFAULT に戻し、標準の入出力は `/dev/null` にした。runner は始まりと終わりの時刻、負荷、rc を meta file に書く。`run_in_background`、`nohup`、`&` は使っていない。待つのは `caffeinate -w <pid>` を 175 s ずつ区切って繰り返した
- 自分の別の run(`-v` の run、coverage の run、穴の probe)は、どれも shell の段のうちに終えた。Go の段(22:00 ごろから)には何も重ねていない
- run-verify.sh は自分の evidence も `docs/evidence/verify-2026-11-19-124744.log` に書いた(同じく追跡外)

落ちたテストがないので、パッケージの打ち直しはしていない。

## 計画のテストと AC の対応

テストの結果は `-v` の run から取った。subtest の数は `    --- PASS: <name>/` の行を数えた。

| AC / Test plan の項 | テスト | 結果 |
| --- | --- | --- |
| AC1(issue の 2 行だけの rc で info、Detail は 2 つの alias が pane で定義されないと言う) | `TestScanShellAliasFile_Issue343LinesAreBothMarked`、`TestCheckShellAliases_Issue343Rc_InfoNotWarn` | pass |
| AC2(8 つの形が info) | `TestCheckShellAliases_PaneGuardedForms_Info`(9 行。8 つの形と基本の形)。scanner の段は `TestScanShellAliasFile_MarksAliasesAHerdrPaneIDTestKeepsOutOfThePane`(28 行)、語の段は `TestShellAliasGuardTest_ReadsEachExactHerdrPaneIDForm`(24 行) | pass |
| AC3(挙げた形は warn のまま) | `TestCheckShellAliases_FormsAPaneCanDefine_StillWarn`(20 行)、`TestScanShellAliasFile_LeavesAliasesAPaneCanDefineUnmarked`(36 行)、`TestShellAliasGuardTest_OtherFormsAreNotGuards`(18 行)、`TestScanShellAliasFile_ElifStopsMarkingFromThatPointOn`、`..._UnclosedIfAtEOFDropsIfMarksButKeepsInlineMarks`、`..._StrayFiDropsIfMarksButKeepsInlineMarks`、`..._IfKeywordOutOfOrderDropsIfMarks`(5 行) | pass |
| AC4(条件付きと条件なしの codex で warn、warn の文は条件なしだけ) | `TestCheckShellAliases_PaneGuardedAndUnguardedCodex_WarnListsOnlyUnguarded` | pass |
| AC5(条件付きの Detail にも値が出ない) | `TestCheckShellAliases_PaneGuardedAliasValueNeverAppearsInDetail`(2 行)、AC1 の `..._Issue343Rc_InfoNotWarn`、前からある `TestCheckShellAliases_AliasValueNeverAppearsInDetail`(2 行) | pass |
| AC6(今のテストがすべて通る、期待値を弱めない) | 下の「AC6 の確かめ」 | pass |
| Test plan の Unit: 区切りを残しても語の列が変わらない | `TestShellAliasStatementsWithOps_KeepsTheOperatorBeforeEachStatement`(20 行)、変えていない `TestShellAliasStatements` と `TestShellAliasValueTokens` | pass |
| Test plan の Unit: if の入れ子(then、else、elif、入れ子の別の if、閉じていない if、対応のない fi) | AC2 と AC3 の scanner のテスト(`if command -v nested in the then side of a -z if`、`-z if nested in an unrelated if` の行を含む) | pass |
| Edge: 継続行をまたぐ if | `MarksAliases…` の `quoted value continuing onto the next line inside a -z if's then side` と `one-line if on a -z test whose alias value continues onto the next line` | pass |
| Edge: 同じ行に判定と alias が 2 組 | `MarksAliases…` の `two guarded aliases on one line` | pass |
| Edge: 条件付きで flag のない alias は harmless のまま | `TestCheckShellAliases_PaneGuardedAliasWithoutFlags_PassHarmless` | pass |
| Edge: claude の `--permission-mode` を足す条件付きの alias は warn にならない | `TestCheckShellAliases_PaneGuardedClaudePermissionMode_Info`、`..._PaneGuardedClaudePermissionModeBesideUnguardedClaudeModel_Info` | pass |
| Edge: herdr がない環境 | `TestCheckShellAliases_PaneGuardedCodex_HerdrAbsent_InfoWithoutSuffix` | pass |

### AC6 の確かめ

- base `a8c5e289` の `internal/cli/doctor_shell_alias_test.go` にあるテストの関数は 72 本。そのうち 71 本は `-run 'ShellAlias'` に当たり、`-v` の run で PASS した。残りの `TestParseAliasWords_NoDefinition` は名前が filter に当たらず、run-test.sh の `internal/cli` の ok の中で通った。`doctor_shell_alias_unix_test.go` の 3 本(この差分は触れていない)も `-v` の run で PASS
- 計画が名指す `TestCheckShellAliases_AliasAfterAndAnd_Detected` と `TestCheckShellAliases_AliasInsideThenClause_Detected` も PASS
- テストのファイルの差分は +858 −0。足した場所は import の 2 行(`fmt`、`slices`)と、base の最後の行(1718 行)のあとの 1 つの塊だけで、base のテストの本体には 1 行も入っていない(`git diff -U0` の hunk の見出しで確かめた)。足したテストの中の `return` は 1 つで、`assertPaneGuarded` が長さの違いを `t.Errorf` で報告したあとに添字で落ちないためのもの

## Coverage

- 測り方: `go test -count=1 ./internal/cli/ -run 'ShellAlias|ParseAliasWords' -coverprofile=…`(3.8 s)。パッケージ全体の数字(15.2%)は filter をかけた run の数字なので使わない
- Statement: `internal/cli/doctor_shell_alias.go` は 450/455 = 98.9%。この差分で足した行に当たる block(`git diff -U0 a8c5e289...HEAD` の足した行と profile の block の重なり)は 156/157 = 99.4%
- Function: このファイルの 33 関数のうち 28 が 100%。100% でないのは `condClosedBeforeThen` 80.0%、`shellAliasRcCandidates` 97.1%、`scanShellAliasFile` 97.8%、`shellAliasStatementsWithOps` 98.6%、`checkShellAliases` 98.8%
- Branch: Go の coverage は branch を測らない
- 通っていない block は 5 つ。この差分で足した行にあるのは `condClosedBeforeThen` の 662〜663 行の 1 つだけで、下の G1 で確かめた。ほかの 4 つ(198 行の `EvalSymlinks` の失敗、355 行の行末のバックスラッシュと改行、903 行の `os.Stat` の失敗、1424 行の harmless が 2 つ以上のときの `have`)は、この差分で足していない行にある

## 穴の確かめ

### G1: 同じ行の `&` のあとの `then`(LOW、黙る向き)

- `condClosedBeforeThen`(`internal/cli/doctor_shell_alias.go:660`)の最初の判定、`then` の前の区切りが `;` でも改行でもないときの `return false`(661〜663 行)を、どのテストも通らない。複数行の形(`if [[ -z $HERDR_PANE_ID ]] &` の次の行に `then`)は `LeavesAliases…` の `condition run in the background before then` が持つが、そちらは前の行の終わり方の判定(667 行)で落ちる
- この判定に当たる形は `if [[ -z $HERDR_PANE_ID ]] & then alias codex='codex -m x'; fi` のように、同じ行で `&` のすぐあとに `then` を書いたもの。`HERDR_PANE_ID=wD:p1` の zsh 5.9(`-f` でも)と bash 3.2 で打つと、条件が background で走って status 0 になり、pane の中でも alias が定義された
- 使い捨てのテスト(scratch の `zz_probe_test.go`、`go test -overlay` で足した。worktree は触っていない)に、この形と `if [[ -n $HERDR_PANE_ID ]] & then :; else alias codex='codex -m x'; fi` の 2 行を入れた。HEAD では 2 行とも印が付かずに通る。661 行を `if (op != ";" && op != "") && false {` にした変異では、2 行とも `PaneGuarded [true]` になって落ちる。同じ変異で `-run 'ShellAlias|ParseAliasWords'` の既存の 99 本はすべて通るので、変異は残る
- パッケージ全体の run はこの変異では打っていない。`HERDR_PANE_ID` を条件にした rc の fixture を書くのは `doctor_shell_alias_test.go` だけで(ほかの `internal/cli` のテストの `HERDR_PANE_ID` は org の verb の環境の値)、判定の tracker は `scanShellAliasFile` からしか呼ばれない(grep で確かめた)
- 今のコードは正しく、warn に残す向きで動いている。ただ、この行を消す変更は、計画が一番に挙げる危険(pane で効く alias を info にする)をテストに気づかれずに入れられる。直し方の案: `TestScanShellAliasFile_LeavesAliasesAPaneCanDefineUnmarked` に `{"condition run in the background before then on the same line", "if [[ -z $HERDR_PANE_ID ]] & then alias codex='codex -m x'; fi\n"}` の 1 行を足す。PR は止めない

### そのほかの穴

- G2: 本物の herdr の pane に `HERDR_PANE_ID` が入ることは、この run でも確かめていない(verify report と同じ)。テストは rc の読みだけを見る
- G3: 計画の Known gaps と register の 137 行の、印を誤って付ける 3 つの形(引用符付きの `"[["` など)は打ち直していない。self-review の zsh の probe が確かめている
- G4: build した `ralph doctor` を一時の HOME で打つ観察は、`/test` では打ち直していない。verify の run 1(HEAD `4fc29299`)で打っており、そのあと Go のファイルは変わっていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | | | |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| base の 72 本のテスト(`AliasAfterAndAnd_Detected`、`AliasInsideThenClause_Detected` を含む) | pass | 「AC6 の確かめ」 |
| 文の分け方の変更で今の語の読みが変わらない(計画の Risks の 4 つ目) | pass | 変えていない `TestShellAliasStatements`、`TestShellAliasValueTokens` が PASS |
| alias の値が Detail に出ない | pass | `TestCheckShellAliases_AliasValueNeverAppearsInDetail`(2 行)と、新しい文に当てた `..._PaneGuardedAliasValueNeverAppearsInDetail`(2 行) |
| ほかのパッケージと shell の suite | pass | Go の 11 パッケージがすべて ok、`tests/*.sh` の 45 本が通り、skip は前の report と同じ 2 つ |

## Test gaps

- G1(LOW): 同じ行の `& then` の行がなく、`condClosedBeforeThen` の 661〜663 行を消す変異が残る。1 行のテストの追加で閉じる
- G2〜G4: 上の「そのほかの穴」。どれも PR を止めない

## Verdict

- Pass: 56/56(Go の 11 パッケージと shell の 45 本。`==> All verifiers passed.`、rc 0)。`-run 'ShellAlias'` の top-level 98 本も全部 PASS
- Fail: 0
- Blocked: 0
- Verdict: **pass**。AC1〜AC6 のテストはすべて通った。AC7 の 2 つの script は verify の担当で、ここでは `tests/test-check-sync.sh` と `tests/test-check-skill-sync.sh` が local verifier の中で通ったことだけを見た。G1 は LOW の穴で PR を止めない。verify report が `/sync-docs` に渡した register の 373 行と 417 行の carry-over は、ここでは触れていない
