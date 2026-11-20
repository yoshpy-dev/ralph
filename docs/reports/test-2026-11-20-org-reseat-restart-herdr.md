# Test report: org-reseat-restart-herdr

- Date: 2026-11-20(cycle 1。`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md`(issue #354、仕様 `docs/specs/2026-11-20-org-reseat-restart-herdr.md` の AC1〜AC8)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `d05fa43e`(main `26b9925c` を merge 済み、worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから起動した。Bash tool の前景の上限 600 s を超えるので、perl の fork と setsid で切り離し、INT・HUP・QUIT・TERM を既定に戻し、標準入出力を `/dev/null` とログにつないだ。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと AC ごとの `-v` の run、`go test -overlay` による変異 9 個、絞った run の coverage、追跡しないテストファイルによる穴の確かめを行った。本物の herdr のサーバーには何も打っていない(テストはどれも fake の herdr を使い、穴の確かめは status の読みだけを fake にした)
- Evidence: `docs/evidence/test-2026-11-20-org-reseat-restart-herdr.log`(7,222 行。run-test.sh の出力 5,829 行のあとに、実行の時刻と load、AC ごとの `-v` の run、変異の old/new と結果、穴の確かめのテストファイル、coverage を足した。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-20-114616.log` にも書いた(同じく追跡外)
- 環境: go1.27.3 darwin/arm64、git 2.49.0、12 コア

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-20 20:46:16 〜 21:08:55 JST、1,359 s、rc 0。開始時の 1 分 load 7.02、終了時 8.27、30 s ごとの 1 分 load は 6.4〜17.3 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL: 0` の要約行 29 本、`N passed, 0 failed` の要約行 14 本の合計 833) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 10 分(20:46〜20:56 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 下の表 |
| `go test -count=1 ./internal/org/ -run 'Reseat' -v` | 107 関数、`=== RUN` 266 行(subtest を含む) | 266(関数 107、subtest 159) | 0 | 0 | 8 s(パッケージ 7.354 s) |
| `go test -count=1 ./internal/org/driver/ -v -run 'PaneListAll\|ServerStop\|PaneList'` | 4 関数、`=== RUN` 36 行 | 36(関数 4、subtest 32) | 0 | 0 | 2 s(パッケージ 0.899 s) |
| `go test -count=1 ./internal/cli/ -run 'Reseat\|HerdrServer\|PSEnv\|Environ' -v` | 64 関数、`=== RUN` 211 行 | 211(関数 64、subtest 147) | 0 | 0 | 147 s(パッケージ 146.230 s) |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.805 s |
| `internal/cli` | ok | 741.594 s |
| `internal/config` | ok | 20.939 s |
| `internal/gitroot` | ok | 6.634 s |
| `internal/insights` | ok | 4.717 s |
| `internal/org` | ok | 757.295 s |
| `internal/org/driver` | ok | 2.734 s |
| `internal/org/protocol` | ok | 2.192 s |
| `internal/scaffold` | ok | 4.090 s |
| `internal/toolchain` | ok | 18.564 s |
| `internal/upgrade` | ok | 2.746 s |

`internal/org` は golang pack の 1 パッケージの上限(30 分)の 42% で終わった。Go の段(20:56 ごろから)には、私の絞った run が重なった: 変異の run(org は 1 個 13〜15 s、cli は 1 個 5〜108 s)、絞った coverage の run(org 19 s、cli 153 s)、穴の確かめ(5 s 前後)。その間の 1 分 load は 7.7〜16.4。落ちたテストはなく、打ち直したテストもない。依頼で名前の出ていた既知の flake、`internal/org` の `TestGate_TwoGatesRunTheirScriptsOneAtATime` も落ちなかった。

### 計画の AC とテスト

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | org `TestReseat_RestartHerdr_StopsTheServerBetweenTheStopsAndTheStarts`、cli `TestOrgReseat_RestartHerdrStopsTheServerBetweenTheStopsAndTheStarts` | 座席を止める → 2 回目の pane の一覧 → `server_stop` → status が stopped → ゲート → 座席を立てる、の呼ぶ順を step の列で固定。`reseated` の記録 |
| AC1b | org `TestPlanReseat_RestartHerdr_RefusesASubsetOfTheActiveSeats`、cli `TestOrgReseat_RestartHerdrRefusals/a_subset_of_the_active_seats` | `--seat` で一部だけを選ぶと、status の読みもゲートも 0 回で断る |
| AC2 | org `TestReseat_RestartHerdr_RefusesAnotherPaneBeforeAnyStop/{real,dry_run}`、`TestReseat_RestartHerdr_DryRunListsThePanesAndStopsNothing`、cli `TestOrgReseat_RestartHerdrRefusesAnotherPane`、`TestOrgReseat_RestartHerdrDryRunPrintsTheServerStepsAndStopsNothing` | ほかの pane があれば何も止めずに断り、id・label・cwd と、記録だけが戻った pane の見分け方を出す。dry run は何も止めない |
| AC3 | org `TestPlanReseat_RestartHerdr_RefusedInAHerdrPane`、cli `TestOrgReseat_RestartHerdrRefusals/from_a_herdr_pane` | `HERDR_PANE_ID` があれば断る |
| AC4 | org `TestReseat_RestartHerdr_APaneMadeWhileTheSeatsStopKeepsTheServer`、`TestReseat_RestartHerdr_APaneListThatCannotBeRead/{before_the_stops,before_the_server's_stop}` | 座席を止めたあとに pane ができていれば、`server_stop` を打たず、座席を止めたまま、`--restart-herdr` と `--claude-config-dir` を写した打ち直しを出す |
| AC4b | org `TestReseat_RestartHerdr_TheRetryAfterTheServerStartedLeavesOutRestartHerdr`、`TestReseatRetryHerdrFlags`、cli `TestOrgReseat_RestartHerdrRetryAfterTheServerStarted` | サーバーを起動し直したあとの打ち直しは `--restart-herdr` を持たず `--claude-config-dir` を持つ。その命令を実際に打つと、サーバーを止めずに残りの座席だけを立てる |
| AC5 | cli `TestOrgHerdrServerEnvOverride_ApplySetsReplacesOrRemovesClaudeConfigDir`、`TestOrgHerdrServerGate_StartsTheServerWithTheClaudeConfigDirOverride`、`TestOrgHerdrServerGate_TheWiringCarriesTheClaudeConfigDirOverride`、`TestOrgHerdrServerCheckClaudeConfigDir`、`TestOrgReseat_ClaudeConfigDirIsCheckedFirst`、`TestOrgReseat_ClaudeConfigDirReachesTheHerdrServerGate` | `/x` で置き換え、`''` で外し、存在しないディレクトリは断る。打ち直しにも写る |
| AC6 | cli `TestOrgHerdrServerReadClaudeConfigDir_*`(8 関数)、`TestOrgPSEnvValue_TheValueEndsAtTheNextEntryOfAnyName`、`TestOrgHerdrServerEnvCommand_*`、`TestOrgReseat_HerdrServerLineAndNotes`、`TestOrgReseat_RestartHerdrPrintsTheServersClaudeConfigDirAgain`、`TestOrgReseat_NoServerLineAgainWithoutARestart` | fake の lsof と ps で読めたとき・読めないとき・pid が複数のとき・値に空白があるとき。`PSReadsANonASCIIValueUnderUTF8` は本物の ps を使う。立て直しのあとにもう一度読む行 |
| AC7 | 11 パッケージの全テスト | 下の Regression checks |
| AC8 | cli `TestOrgReseatHelp_SaysHowToRestartTheHerdrServer`、`TestOrgHerdrServerHelp_SaysReseatCanSetClaudeConfigDir`、local verifier の `test-check-sync.sh`・`test-check-skill-sync.sh` | help の文。文書の同期の確かめ自体は /verify が行った |

止まらない・止めるのが失敗する・起動できない、の端の場合は org の `TestReseat_RestartHerdr_AServerThatDoesNotStop`(2 subtest)、`AStopThatFails`(3)、`AGateThatFailsAfterTheStop`、`TestPlanReseat_RestartHerdr_RefusesAServerItCannotRestart`(4)が通った。`--restart-herdr` なしでサーバーに触れないことは `TestReseat_WithoutRestartHerdrTheServerIsLeftAlone` が通った。

### 変異(`go test -overlay`、追跡ファイルは変えていない)

| # | 変異 | run | 結果 |
| --- | --- | --- | --- |
| 1a | `reseat.go` の `restartHerdr` で 2 回目の pane の確かめをまるごと外す(`PaneListAll` を呼ばず、`others` と `err` は空) | org `-run Reseat` | 検出。3 関数が落ちた: AC1 の順の固定(`["… "close" "server_stop" …]`、2 回目の `pane_list_all` がない)、AC4 の `APaneMadeWhileTheSeatsStopKeepsTheServer`、`APaneListThatCannotBeRead/before_the_server's_stop` |
| 1b | 一覧は呼ぶが結果を見ない(`if false && len(others) > 0`) | org `-run Reseat` | 検出。AC4 の `APaneMadeWhileTheSeatsStopKeepsTheServer` だけが落ちた(`Reseat error`、`reseat_restart_test.go:470`)。cli `-run OrgReseat_RestartHerdr` では 16 件すべて通った(下の Test gaps 3) |
| 2a | 打ち直しで `--restart-herdr` を常に残す(`herdr.restartHerdr = r.plan.RestartsHerdr`) | org `-run Reseat`、cli `-run OrgReseat_RestartHerdr` | 検出。org は AC4b の `TheRetryAfterTheServerStartedLeavesOutRestartHerdr`、`AStopThatFails/the_server_stopped_anyway`、`AGateThatFailsAfterTheStop` の 3 関数。cli は `TestOrgReseat_RestartHerdrRetryAfterTheServerStarted` と `TestOrgReseat_RestartHerdrPrintsTheServerLineAfterAFailedStart` |
| 2b | `reseatRetryHerdrFlags` が常に ` --restart-herdr` を足す(`if true`) | org `-run Reseat` | 検出。15 関数(`TestReseatRetryHerdrFlags` の 3 subtest、既存の `TestReseatRetryCommand` の 5 subtest、`TestReseat_RetryAfterAFailedStart` など) |
| 3 | `--seat` の一部だけの指定を断る確かめを外す(`if false && len(left) > 0`) | org `-run Reseat`、cli `-run OrgReseat_RestartHerdr` | 検出。org は AC1b の `TestPlanReseat_RestartHerdr_RefusesASubsetOfTheActiveSeats`(`refusal`、`status reads 1, gate runs 1; want none`)、cli は `TestOrgReseat_RestartHerdrRefusals/a_subset_of_the_active_seats` |
| 4 | `orgPSEnvKey` を `` ` [A-Za-z_][A-Za-z0-9_]*=` `` に戻す | cli `-run 'PSEnv\|HerdrServerReadClaudeConfigDir'` | 検出。`TestOrgPSEnvValue_TheValueEndsAtTheNextEntryOfAnyName` の 6 subtest(名前に hyphen、dot、bash の関数、数字で始まる名前、hyphen の名前の前の空の値、2 項目の 1 つめ)。ほかの 44 件は通った |
| 5a | 立て直しのあとの `printOrgReseatRestartedHerdrServerNotes` の呼び出しを外す | cli `-run OrgReseat_` | 検出。`TestOrgReseat_RestartHerdrPrintsTheServersClaudeConfigDirAgain` の 7 subtest と `TestOrgReseat_RestartHerdrPrintsTheServerLineAfterAFailedStart` |
| 5b | 同じ呼び出しに `res.HerdrRestarted` ではなく `true` を渡す | cli `-run OrgReseat_` | 検出。`TestOrgReseat_NoServerLineAgainWithoutARestart` の 3 subtest(`--restart-herdr` なし、ほかの pane で断る、止めたあと起動できない) |
| 6 | 追加。`orgHerdrServerEnvProbeNow` の `goos` を `"linux"` にする | cli `-run 'HerdrServerRead\|PSEnv\|HerdrServerEnv\|GapProbe_'` | 既存のテストでは生き残った(subtest を含む 63 件がすべて通る)。穴の確かめのテストを足すと検出(`open /proc/<pid>/environ: no such file or directory`)。下の Test gaps 1 |

依頼された 5 種(1〜5)は、どれも名前の挙がった AC のテストで落ちた。ビルドの失敗はない(6 の最初の形は `runtime` が未使用になってビルドが落ちたので、`"linux" + runtime.GOOS[:0]` で作り直した)。変異の前後で `git status --porcelain --ignored`(`.harness/cache/` を除く)は変わっていない。

## Coverage

- Statement: 変更行の coverage は、変更した Go のファイル(テストを除く)の文のある追加行 435 のうち 428 行が実行された(98.4%)。絞った run の profile から数えた(org `-run Reseat`、driver 全体、cli `-run 'Reseat|HerdrServer|PSEnv|Environ'`)。ファイルごとに `internal/org/reseat.go` 166/166、`internal/cli/org_reseat.go` 46/46、`internal/cli/org_reseat_herdr_server.go` 69/69、`internal/cli/org_herdr_server.go` 7/7、`internal/org/driver/herdr.go` 8/8、`internal/cli/org_herdr_server_env.go` 132/139
- Branch: 計測していない(Go の標準の道具にない)
- Function: driver パッケージ全体 97.2%(全テスト)。org と cli は絞った run なのでパッケージ全体の数字は出していない
- Notes: 実行されなかった 7 行はすべて `org_herdr_server_env.go` にある。80〜81 行(`filepath.Abs` の失敗)、136〜137 行(`orgHerdrServerEnvProbeNow`、本番の probe の組み立て)、144〜145 行(`orgHerdrServerClaudeConfigDirReader` の既定の関数)、384 行(stderr が空のときのコマンドの失敗)。136〜145 行は TestMain が読みを nil にするので、どのテストからも通らない(Test gaps 1)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC7: `--restart-herdr` なしの reseat の既存のテスト | 通った | `internal/org` と `internal/cli` の全テストが run-test.sh で ok。既存のテストファイルの変更は 4 つ: `reseat_test.go` は `TestReseatRetryCommand` の呼ぶ関数を `reseatRetryCommand` から `reseatRetryCommandWithHerdr(…, reseatRetryHerdr{}, …)` に替えた 2 行だけで、期待値の行は変えていない(main の `reseat.go:1526` にあった `reseatRetryCommand` は HEAD にない。計画の S2 の「既存の関数…は変えずに足した」は、この関数については改名が正しい)。`spawn_test.go` と `watch_test.go` は `HerdrClient` に足した `PaneListAll`・`ServerStop` を fake に足しただけ。cli の `main_test.go` は TestMain で読みを nil にする 1 行(計画のずれ (1)、サーバーの行が既存の stderr の固定を変えないため)。interface が広がったので、main の `spawn_test.go` を HEAD に当てる run はビルドが通らず、行っていない |
| 既知の flake `TestGate_TwoGatesRunTheirScriptsOneAtATime` | 落ちなかった | run-test.sh の `internal/org` が ok。打ち直しは要らなかった |
| `ps eww` の値が次の項目に流れ込む(self-review M1、53c12663 で修正) | 直ったまま | 変異 4 で 6 subtest が落ちる |

## Test gaps

1. 本番の probe の組み立て(`orgHerdrServerEnvProbeNow`)と既定の読み(`orgHerdrServerClaudeConfigDirReader`)は、どのテストからも通らない。TestMain が読みを nil にし、ほかのテストは probe を自分で組むため。`goos` を `"linux"` にする変異 6 では、既存の 63 件(subtest を含む)がすべて通った。実機の記録(`docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md`)は lsof と ps を手で打ったもので、ralph のこの道を通っていない。追跡しないテストファイルを overlay で足して確かめた: status の読みだけを fake にして、テストのバイナリ自身を `CLAUDE_CONFIG_DIR=/probe/acct b` で起動して unix socket で待たせると、HEAD の本番の probe は本物の lsof と ps で `herdr server: CLAUDE_CONFIG_DIR=/probe/acct b` を返し、変異 6 では落ちた。herdr を打たないので、このテストを足せば穴は塞がる(darwin と linux 以外、lsof がないときは skip)。重さは LOW
2. 上の確かめの途中で、self-review の L6 を本物の ps で確かめた。listener を Apple の platform binary(`/usr/bin/perl`)にすると、`ps eww` は環境を出さず、引数の `$s=IO::Socket::UNIX->new(…)` を環境の項目と数えるので、読みは「読めない」ではなく `unset` を返した。homebrew の herdr は環境を出す(実機の記録の 5)ので、ralph が起こすサーバーでは起きない。L6 は直さずに doc に書いたとおりで、新しい指摘ではない
3. 変異 1b(2 回目の pane の一覧の結果を見ない)は、cli の `-run OrgReseat_RestartHerdr` の 16 件では落ちない。CLI に、止めているあいだに pane ができる場合のテストがないため。org の AC4 のテストが落とすので、CLI の層は org の結果を出すだけで済む
4. 計画と仕様で確かめないと決めたもの: 座席のある org での `ralph org reseat --restart-herdr` の通しの実行、新しいアカウントの資格で座席が動くこと(2 つのアカウントを持つ人の手でしか確かめられない)、最後の pane の確かめから `herdr server stop` までの窓(Design decisions 3、tech-debt に記録)

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,359 s、`(cached)` なし)、AC ごとの `-v` の run(org 107 関数 266 件、driver 4 関数 36 件、cli 64 関数 211 件)、AC1〜AC8 のテスト、依頼された変異 5 種(8 個)をすべて検出
- Fail: 0
- Blocked: なし
