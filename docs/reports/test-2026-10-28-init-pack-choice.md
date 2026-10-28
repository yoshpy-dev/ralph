# Test report: init-pack-choice

- Date: 2026-10-28(UTC 16:07〜17:00)
- Plan: `docs/plans/active/2026-10-28-init-pack-choice.md`(Test plan、AC1〜AC8、Progress checklist の slice の注記と、TTY が要る行の既知の穴)
- Tester: tester サブエージェント(Claude Opus 5.5)。pipeline の cycle 1
- Scope: HEAD 63ccc8c8(origin/main ae64da70 を取り込み済み。この branch の変更は `git diff origin/main...HEAD`)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.50.1、expect 5.45
- Evidence: `docs/evidence/test-2026-10-28-init-pack-choice.log`(1〜10 の節: 名指しの実行、check-template、mutant ごとの差分と落ちた行、G の mutant、pty の script と出力、単独の打ち直し、`run-test.sh` の 2 回分、coverage)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は `go test -overlay` と `go build -overlay` で、元の `init.go` を scratchpad に写して置き換えた。G の mutant と origin/main の比較は scratchpad の `git archive` の写しで走らせた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `go test ./internal/cli/ -run 'Init\|Pack' -count=1 -v` | トップレベル 49(サブテスト 75) | 49(75) | 0 | 0 | rc 0、8 秒(`ok … 7.703s`) |
| 2 | `./tests/test-check-template.sh` | 60 | 60 | 0 | 0 | rc 0、26 秒 |
| 3 | `go test -overlay` の mutant 20 個と対照(下の「mutant」) | 21 実行 | 対照は通り、mutant 14 個が落ちた | 生き残り 6 個 | - | 1 回 10〜25 秒 |
| 4 | pty の確認(HEAD と mutant の binary、下の「pty の確認」) | 12 実行 | 期待どおり | - | - | 1 回 15 秒以内 |
| 5 | `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` 1 回目 | shell 45 本 + Go 9 パッケージ | shell 45 本、Go 8 パッケージ | Go 1(`internal/cli` の `TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`) | 0 | rc 1、1,651 秒(16:07:48〜16:35:19Z) |
| 6 | `go test ./internal/cli/ -run '^TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint$' -count=20 -v`(HEAD と origin/main の写し) | 20 + 20 | 20 + 20 | 0 | 0 | rc 0、70 秒と 68 秒(1 回 3.17〜4.94 秒と 3.13〜3.69 秒) |
| 7 | `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` 2 回目(5 の失敗のあとの打ち直し、同じ HEAD) | shell 45 本 + Go 9 パッケージ | shell 45 本、Go 9 パッケージ | 0 | 0 | rc 0、1,195 秒(16:38:55〜16:58:50Z) |

- 1 の 49 本は、この branch の `internal/cli/init_packs_test.go` の 17 本(サブテスト 75 はすべてこの 17 本のもの)と、名前が `Init` か `Pack` に当たる既存の 32 本(`TestExecuteInit_V2_*`、`TestRunInitInteractive_ExistingProjectSkipsForm`、`TestAddPack_*`、`TestCheckInstalledPacks_*`、`TestClassifyMigration_*Pack*`、`TestRunUpgradeV2_*Pack*` など)。SKIP は 0
- 2 の G の行: `PASS: G. a fresh ralph init scaffold passes check-template.sh with no FAIL line`。H(`packs/` のない project)の 2 行も PASS
- 5 と 7 の shell の部分は、45 本(`tests/test-*.sh` の数と同じ)の要約行が 1 行も違わなかった(`diff` で確かめた)。`PASS` で始まる行はどちらも 4,796 行で、0 でない `FAIL:` の数を持つ要約行はない
- 5 の Go: `cmd/ralph` 0.987 秒、`internal/cli` FAIL 523.666 秒、`internal/config` 3.350 秒、`internal/insights` 1.871 秒、`internal/org` 693.006 秒、`internal/org/driver` 3.719 秒、`internal/org/protocol` 3.664 秒、`internal/scaffold` 4.165 秒、`internal/upgrade` 5.114 秒。どれも `(cached)` ではない
- 7 の Go で走ったのは `internal/cli`(432.198 秒)、`internal/org`(536.513 秒)、`internal/config`(1.088 秒)で、残りの 6 パッケージは 5 で同じ入力のまま通った結果の `(cached)` だった
- 1 分の load average は、5 の始まりが 6.6、16:32Z に 19.5、終わりが 8.4。7 は 5.9、5.3、4.2 と下がった。5 の途中(16:10〜16:25Z ごろ)には、この報告の 3 と 4 の `go test`・`go build`・pty の実行を並べて走らせていた。7 の間に走らせたのは、30 秒の G の mutant(下)だけである

## Test plan の項目

| Test plan の項目 | テスト | 結果 |
| --- | --- | --- |
| 推定: 手がかりのファイルごとに、その言語だけ(glob を含む) | `TestRecommendedPacks_FindsEachMarker`(19 行。`tsconfig.build.json`、`requirements-dev.txt`、`main.tf`、`main.tofu`、`main.tftest.hcl`、`.terraform.lock.hcl` を含む)、`TestPackMarkers_NameEveryRepositoryPack` | PASS |
| 推定: 下の階層だけ、マーカー名のディレクトリ、まだない対象 | `TestRecommendedPacks_LooksOnlyAtFilesDirectlyInTheTarget`(サブテスト 10: 空、`sub/go.mod` だけ、`go.mod` という名前のディレクトリ、前方一致だけの名前、ファイルへの symlink、ディレクトリへの symlink、切れた symlink、複数のパックの並び、使えないパックのマーカー、まだない対象) | PASS |
| 選択肢: `--pack` なしで何も選ばない、`--pack` で選ぶ、(Recommended) は表示だけ、skip が最後 | `TestPackOptions_LabelsSelectionAndSkip`(5 行。期待の並びの最後がいつも skip で、`slices.Equal` が順まで見る) | PASS |
| 選択の確かめ(4 つ) | `TestValidatePackSelection`(7 行: skip だけ、1 つ、2 つ、skip と言語、skip が先、nil、空) | PASS |
| 選択から入れるパックへの直し | `TestPacksFromSelection_DropsSkip`(4 行) | PASS |
| 問いの配線(`Blur()` と `Error()`) | `TestNewPackQuestion_ValidatesTheSelectionOnBlur`(4 行)、`TestNewPackQuestion_ShowsEveryOptionWithItsLabel` | PASS |
| `--pack` の名前: 重なり、カンマ、知らない名前、空(`''`、`p` と `''`、`''` と `p`、`p,`) | `TestParsePackFlag_ReturnsThePacksToInstall`(7 行)、`TestParsePackFlag_RefusesEmptyAndUnknownNames`(9 行。先頭のカンマ、空白だけ、`skip` を含む) | PASS |
| `newInitCmd`: `--yes` だけ、`--yes --pack` | `TestInitCmd_YesAloneInstallsNoPackAndPrintsTheHint`、`TestInitCmd_YesWithPackInstallsExactlyThosePacks`(4 行: 1 つ、カンマ、繰り返しの順、同じ名前 2 回) | PASS |
| `newInitCmd`: 知らない名前と空の名前で、`--yes` の有無とも、何も書かずにエラー | `TestInitCmd_BadPackNameStopsBeforeAnyWrite`(6 行。各行で、既存の v2 project の `--yes` なし、新しい dir の `--yes` あり・なしの 3 回) | PASS |
| skip だけの選択を `executeInit` に通す | `TestExecuteInit_SkipAloneInstallsNoPackAndPrintsTheHint`、`TestExecuteInit_HintWithNoAvailablePacksLeavesOutTheList` | PASS |
| `TERM=dumb` の guard(止まらずに終わること) | `TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm`。HEAD では 0.00 秒。guard を消した mutant e では 10.00 秒の上限で落ち、パッケージの実行は 25 秒で終わった(止まらない) | PASS |
| 既存の project でフォームを出さない | `TestRunInitInteractive_ExistingProjectSkipsForm`(`cli_test.go:263`) | PASS |
| G(すべてのパック)と H(パックなし) | 2 の G と H の行 | PASS |

計画の Integration tests にある `--yes --pack golang,python` は、テスト用の embed FS のパックが golang と typescript なので、`golang,typescript` の行になっている。repository の 6 パックでの `--pack` は 2 の G が受け持つ(`dart,golang,python,rust,terraform,typescript`)。

## mutant

`internal/cli/init.go` を scratchpad に写し、1 回だけ現れる字面を perl で置き換えた(0 回か 2 回以上なら止める)。`go test ./internal/cli/ -run 'Init|Pack' -count=1 -timeout 4m -overlay <id>.json` で走らせた。対照(置き換えなし)は rc 0。変数が使われなくなる置き換えは build に失敗するので、`false &&` や `0 *` の形にした(build の失敗は 0 件)。

| id | 変更 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| a | `--pack` の検査を `--yes` の経路だけに移し、フォームの経路には生の値を渡す | 落ちる | `TestInitCmd_BadPackNameStopsBeforeAnyWrite`(6 行すべて。既存の project で nil が返る最初の確かめで止まり、フォームは待たない) |
| b | 確かめが skip と言語の組を通す | 落ちる | `TestValidatePackSelection`(2 行)、`TestNewPackQuestion_ValidatesTheSelectionOnBlur/skip_with_a_pack` |
| c | `packsFromSelection` が skip の値を通す | 落ちる | `TestPacksFromSelection_DropsSkip`(2 行)、`TestExecuteInit_SkipAloneInstallsNoPackAndPrintsTheHint` |
| d | `packOptions` がすべてのパックを選んだ状態にする | 落ちる | `TestPackOptions_LabelsSelectionAndSkip`(5 行)、`TestNewPackQuestion_ValidatesTheSelectionOnBlur`(3 行) |
| e | `TERM=dumb` の guard を消す | 落ちる(10.00 秒、止まらない) | `TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm` |
| f | `RunE` の `AvailablePacks: availPacks` を消す | 落ちる | `TestInitCmd_YesAloneInstallsNoPackAndPrintsTheHint` |
| i | 問いの `Validate` を外す | 落ちる | `TestNewPackQuestion_ValidatesTheSelectionOnBlur`(2 行) |
| j | 問いの `Height` を 0 にする | 落ちる | `TestNewPackQuestion_ShowsEveryOptionWithItsLabel` |
| j2 | `Height` を選択肢の数だけにする(タイトルと説明の行を数えない) | 落ちる | `TestNewPackQuestion_ShowsEveryOptionWithItsLabel` |
| k | マーカーの確かめから通常ファイルの条件を外す | 落ちる | `TestRecommendedPacks_LooksOnlyAtFilesDirectlyInTheTarget`(マーカー名のディレクトリ、ディレクトリへの symlink) |
| l | パックが 0 個のときの案内を出さない | 落ちる | `TestInitCmd_YesAloneInstallsNoPackAndPrintsTheHint`、`TestExecuteInit_SkipAlone…`、`TestExecuteInit_HintWithNoAvailablePacks…` |
| m | 確かめが空の選択を通す | 落ちる | `TestValidatePackSelection`(2 行)、`TestNewPackQuestion_ValidatesTheSelectionOnBlur/nothing` |
| n | `parsePackFlag` が重なりを残す | 落ちる | `TestParsePackFlag_ReturnsThePacksToInstall`(2 行)、`TestInitCmd_YesWithPackInstallsExactlyThosePacks/same_pack_twice` |
| o | `parsePackFlag` が空白を取らない | 落ちる | `TestParsePackFlag_ReturnsThePacksToInstall/spaces_around_names`、`TestParsePackFlag_RefusesEmptyAndUnknownNames/blank_name` |
| g | guard の `&&` を `\|\|` にする(端末でない stdin を `TERM` によらず断る) | 生き残る | - |
| h | guard から「stdin が端末でない」の条件を外す(`TERM=dumb` なら端末でも断る) | 生き残る | - |
| p | `runInitInteractive` の `cfg.Packs = packsFromSelection(cfg.Packs)` を消す | 生き残る(計画の既知の穴) | - |
| q | `runInitInteractive` が推定の結果を問いに渡さない | 生き残る(計画の既知の穴) | - |
| r | `runInitInteractive` が `--pack` の初期の選択を捨てる | 生き残る(計画の既知の穴) | - |
| s | `RunE` がフォームの経路に `Packs` のない `initConfig` を渡す | 生き残る | - |

- lead が名指しした a〜f はすべて落ちた
- 生き残った 6 つのうち p、q、r、s が変えるのは、`runInitInteractive` の guard より後ろの行と、`RunE` の `return runInitInteractive(...)`(`init.go:79`)である。g と h は guard の条件そのものを変えるが、guard に届くテストは DumbTerm のテストだけで、その `TERM=dumb` とパイプの stdin では HEAD も g も h もエラーを返す。パッケージの中で `runInitInteractive` を呼ぶテストは、`cli_test.go:285`(既存の project で guard より前に返る)と DumbTerm のテストだけである。`newInitCmd` を `--yes` なしで正しい `--pack` と動かすテストはない(`grep` で確かめた。`cmd/ralph` のテストも `init` を呼ばない)。パッケージ全体で走らせても落ちる経路がないので、全体の打ち直しはしなかった
- 2 の G の側は、`git archive HEAD` の写しで G の `go run` から `--pack "$packs"` を外すと、`FAIL: G. the fresh scaffold has no packs/languages/<name>/ for: dart golang python rust terraform typescript` で落ちた(59 passed、1 failed)

## pty の確認

生き残った mutant を、組み立てた binary で端末を通して確かめた。HEAD は `go build ./cmd/ralph`、mutant は同じ overlay で `go build -overlay`。expect が 100×40 の pty で `ralph init` を動かし、Project name で Enter、言語パックの問いで下に 6 回と Space(skip)または何もせずに Enter を送る。script は evidence の 5 節にある。

| 場面 | binary | 結果 |
| --- | --- | --- |
| `go.mod` のある dir で skip だけを選ぶ | HEAD | rc 0。`golang (Recommended)` が出る。manifest に `packs` の行がなく、`packs/` もない。終わりに `ralph pack add <language> to add a language pack later (available: dart, golang, python, rust, terraform, typescript)` |
| 同じ | p | manifest が `packs = ['/skip']`、`⚠ pack /skip: language pack "/skip" not found` が出る。落ちる |
| 同じ | q | `(Recommended)` が 1 つも出ない。落ちる |
| 空の dir で `--pack golang`、そのまま Enter | HEAD | rc 0、manifest が `packs = ['golang']` |
| 同じ | r、s | `choose at least one language pack, or skip` が出て問いから進まず、15 秒で打ち切った。manifest なし。落ちる |
| `TERM=dumb`、stdin が pty | HEAD | accessible モードの `Project name` の問いが出る |
| 同じ | h | `TERM=dumb and stdin is not a terminal, …` で断る。落ちる |
| `TERM=xterm-256color`、stdin が `/dev/null`、制御端末なし | HEAD | rc 1、`interactive form: huh: bubbletea: error opening TTY: … open /dev/tty: device not configured` |
| 同じ | g | rc 1、guard の `TERM=dumb and stdin is not a terminal` の文で断る(`TERM` は dumb ではない) |
| `TERM=xterm-256color`、stdin が `/dev/null`、pty の中 | HEAD | 15 秒の間、画面の消去を繰り返すだけで `Project name` が出なかった |
| 同じ | g | guard で断る |

- p、q、r、s、h は pty で落ちる。計画が「pty で代えた」とした 3 行(p、q、r)は、この報告でも pty で確かめた
- g の違いは、端末でない stdin のときにどのエラーで止まるかである。HEAD もその場合はフォームを出せず(制御端末なしで bubbletea のエラー、pty の中でも 15 秒で問いが出ない)、g は文の中の `TERM=dumb` が事実と合わなくなる。害は文言の範囲にとどまる

## Coverage

- Statement: `go test ./internal/cli/ -run 'Init|Pack' -coverprofile` で、変わった関数は `parsePackFlag`、`recommendedPacks`、`isPackMarker`、`packOptions`、`validatePackSelection`、`packsFromSelection`、`newPackQuestion`、`runInitNonInteractive` が 100.0%、`newInitCmd` 86.4%、`runInitInteractive` 50.0%、`executeInit` 83.1%(パッケージ全体はこの絞り込みで 17.4%)
- 通らない行: `newInitCmd` の `AvailablePacks` のエラー(52-53)、`filepath.Abs` のエラー(66-67)、`return runInitInteractive(absDir, cfg, force)`(79)。`runInitInteractive` のフォームの組み立てから `executeInit` まで(147-161)
- Branch: 測っていない。`validatePackSelection` の 3 つの分岐、`parsePackFlag` の 3 つの case、`isPackMarker` の通常ファイルの条件は、mutant b・m・n・o・k で 1 つずつ落ちることを確かめた
- Notes: 79 行と 147-161 行は単体テストの外にあり、上の pty の確認で代えた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`(`internal/cli/org_spawn_timeout_test.go:67`、5 だけ、3.05 秒) | `expected the error to name the agmsg_announce step and the hint …, got org: spawn step workspace_create failed: herdr: timed out: context deadline exceeded (the spawn's --timeout-ms deadline of 3000 ms, …)` | 負荷による時間切れ。テストは `--timeout-ms 3000` の期限の中で stub の herdr の step を済ませ、HELLO で期限が切れることを期待するが、5 では最初の step(`workspace_create`)の時点で 3000 ms を使い切った。この branch は `internal/org` と spawn のコードを変えておらず、`internal/cli` で変えたのは init の 3 ファイルである。`internal/cli` のテストに `t.Parallel` はなく、新しいテストが spawn のテストと同時に走ることはない。6 で単独に 20 回ずつ打つと HEAD も origin/main も 20/20 で通った(1 回 3.13〜4.94 秒)。`docs/tech-debt/` に、このテストの不安定さの行はない(seat-spawn-close の解決済みの行が名前を挙げるだけ) | この branch では直さない。7 の打ち直しで `internal/cli` は通った(432.198 秒)。lead の指示が打ち直しを認める 2 つの原因(30 分の時間切れ、`TestStopDirectorProcess_…`)のどちらでもないので、打ち直しを verdict に数えてよいかは lead が決める。テストの docstring は「3000 ms は負荷のある機械でも HELLO の前の step に足りる」と書くが、5 はそれが足りない場合を示した。tech-debt の行を足すか、期限を延ばすか、HELLO の前の step の所要時間を測るかを別の作業で決める |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の project で init がフォームを出さずに upgrade の案内に入る(`cli_test.go:263`) | 通る | 1、5、7 |
| 既存の `TestExecuteInit_V2_*` | 通る | 1、5、7 |
| `TERM=dumb ralph init dir < /dev/null` が問いを出し続けて止まらない(slice S3 の途中で見つかった) | 直っている | DumbTerm のテスト、mutant e が 10 秒で落ちて止まらない |
| skip の値がパックの名前として manifest に残る(計画の S3 が挙げた危険) | 起きない | mutant c と pty の p(manifest の `['/skip']`)を HEAD と比べた |
| G(すべてのパック)と H(パックなし)の check-template | 通る | 2、5、7、G の mutant |

## Test gaps

- `RunE` からフォームの経路への呼び出し(`init.go:79`)と、`runInitInteractive` の guard より後ろ(147-161 行)を実行するテストがない。mutant p、q、r、s が `go test` を生き残り、pty でだけ落ちる。79 行を通すだけなら、`--yes` なしで正しい `--pack` を渡し、既存の v2 project で nil が返ることを見る `newInitCmd` のテストで足りる。ただし、渡す `initConfig` の中身(s)までは押さえられない。中身まで押さえるには、フォームを動かす関数を差し替えられる形にする(製品のコードの変更)か、pty で動かすテストを置く必要がある
- guard の条件の精度: mutant g と h が生き残る。h は、端末があって `TERM=dumb` の環境(HEAD では accessible モードで問いが出る)を断る。テストで押さえるには pty を stdin にする必要がある。g は上のとおり文言の範囲の違い
- `TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` の負荷による失敗は、記録された行がない(Failure analysis)
- この報告で走らせたのは macOS の go1.27.3 だけで、CI の Linux では走らせていない。pty の確認は手動で、CI では走らない

## Verdict

- Pass: 1(トップレベル 49、サブテスト 75)、2(60/60、G PASS)、lead が名指しした mutant a〜f のすべて(ほかに i〜o の 8 つ)、pty の確認 12 実行、`./scripts/run-test.sh` の 2 回目(rc 0、shell 45 本、Go 9 パッケージ)
- Fail: なし。1 回目の `run-test.sh` の `internal/cli` の失敗は、この branch が変えていない spawn のテストの負荷による時間切れで、単独の 20 回(HEAD と origin/main)と 2 回目の全体で通った。lead の指示に挙がった打ち直しの原因ではないことを Failure analysis に書いた
- Blocked: なし
- Verdict: pass
