# Test report: seat-wip-checkpoint

- Date: 2026-10-22(JST 19:33〜19:53、UTC 10:33〜10:53)
- Plan: `docs/plans/active/2026-10-22-seat-wip-checkpoint.md`(試行の所見を直す系列の 4c。AC1〜AC9 の挙動、計画の Test plan の境界の場合、回帰)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD e127f68c(base c90d01b4)。コードとテストは self-review の直しの a79b0f82 と同じで、その後の 1c78d731 以降は記録だけを足している。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64、12 CPU)、git 2.49.0、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8 GOPROXY=off`。toolchain はキャッシュ済みで、network は使っていない)
- Evidence: `docs/evidence/test-2026-10-22-seat-wip-checkpoint.log`(各実行の出力、時刻と load average、mutant の定義と結果、probe の script と出力、canary の記録、関数ごとの coverage。ホームは `~`、session の scratchpad は `<scratch>` と書いた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-22-103317.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。hook の mutant は hook 4 本とテストの写しを scratchpad に置いて打ち、Go の mutant は `go test -overlay` で scratchpad のファイルに差し替えて打った。worktree のファイルには書いていない。WIP の hook は scratchpad の一時の repo の中でだけ打ち、この checkout を cwd にして打ったことはない。本物の herdr・agmsg・osascript・Orca・tmux・gh や network を打つものは実行していない(6 の canary)

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS=-timeout=30m ./scripts/run-test.sh` | shell 44 本、Go 8 パッケージ | shell 44 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1 件(下) | 738 秒、rc 0。cli 272.3 秒、org 340.1 秒、driver 1.8 秒、config 0.9 秒。insights・protocol・scaffold・upgrade は `(cached)` |
| 1b | `go test -count=1 ./internal/insights/` | 1 パッケージ | 1 | 0 | 0 | 0.9 秒。1 で `(cached)` だった insights は internal/org と internal/org/driver に依存するので打ち直した |
| 2 | `GOTOOLCHAIN=go1.25.8 GOPROXY=off go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/` | 4 パッケージ | 4 | 0 | 0 | 322 秒、rc 0。org 321.0 秒、cli 250.7 秒、driver 1.4 秒、protocol 1.1 秒。3〜7 と重ねて打った |
| 3 | 新しいテスト、期待を変えたテスト、同じ関数を見る前からのテスト(org 8 本、driver 11 本、cli 4 本)を `-count=3 -v`(go1.26.0) | 23 × 3 = 69(subtest を含む RUN 186) | 186 | 0 | 0(`--- SKIP` は 0 行) | org 27 秒、driver 1 秒、cli 9 秒 |
| 4 | 3 と同じに `-race` | 69 | 69 | 0 | 0 | 42 秒。`WARNING: DATA RACE` は 0 行 |
| 5 | `bash tests/test-wip-checkpoint-hooks.sh`(1 の中でも走った) | 108 | 108 | 0 | 0 | 16 秒 |
| 6 | 3 の 23 本と 5 を canary の PATH と偽の HOME で(下) | 23 + 108 | 23 + 108 | 0 | 0 | canary の呼び出しは 0 回 |
| 7 | mutant 53 本(下の表) | 53 | - | 52 本が赤 | - | 生き残りは W16 の 1 本(master の行がない。G1) |
| 8 | hook と手順 14 のコマンドの使い捨ての probe(scratchpad の一時の repo。コミットしない) | 12 | 12 | 0 | 0 | 下の「Probe」 |

- 1 の scope は changed の要求だが、`==> Language scope: full fallback (unclassified:.claude/hooks/precompact_checkpoint.sh)` で full に落ちた。pack は golang。shell の 44 本は local verifier が test mode で走らせる全 suite で、どの suite も最後の行が `OK` だった。依頼にあった `tests/test-wip-checkpoint-hooks.sh`(108 passed)、`tests/test-hook-wiring.sh`(PASS 68)、`tests/test-ralph-dispatch.sh`(PASS 33)もこの中で走ったので、別には打っていない。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばしている
- 1 の Go は `-count=1` を付けていないが、変えた cli・org・driver と config は実際に走った。`(cached)` の protocol・scaffold・upgrade は、`go list -test -deps` で internal/org、internal/org/driver、internal/cli のどれにも依存しないことを確かめた。insights は 1b で打ち直した。キャッシュが当たったのは、a79b0f82 で implementer か verifier が同じ中身を走らせていたからだと思われる(未確認)
- 1 の load average(1 分)は開始時 5.6、終了時 10.0。2 は開始時 7.6 で、3〜7 と重なった
- 3 の 23 本は、org の `TestRenderRolePrompt_Leader_PushRefusesWipCommits`、`TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError`、`TestRenderRolePrompt_Leader_FeatureOrgProcedure`(変えていない回帰の確かめ)、`TestDirectorLaunchScript_*` の 4 本(`TextQuotesEveryValue`、`ChecksInThenExecsTheAgent`、`PathAndModes`、`RefusesValuesItCannotWrite`)、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn`、driver の `TestSeatPaneEnv_*` 2 本、`TestWIPCheckpointOff_IsWhatTheHooksRead`、`TestHerdr_TabCreate_PaneEnv*` 8 本、cli の `TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`、`TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`、`TestOrgDirectorStart_LaunchesRegistersAndStops` である。RUN の内訳は org 87、driver 69、cli 30。`PushRefusesWipCommits` の hook の subtest は、hook が見つからないと `t.Skipf` するが、2 本とも PASS で SKIP はなかった
- 6 の canary: `herdr`・`agmsg`・`osascript`・`open`・`tmux`・`gh`・`orca`・`claude`・`codex`・`curl`・`wget`・`nc`・`ssh` を、呼ばれたら記録して exit 97 する script にした dir を PATH の先頭に置き、`HOME` を空の scratch の dir にした(`GOCACHE`・`GOMODCACHE` は元の値、`GOPROXY=off`)。cli のテストの stub の herdr はその前に置かれるので、canary が受けるのは stub から漏れた呼び出しである。終わったあと、canary の `herdr` を直に打つと exit 97 で 1 行記録したので、canary は呼ばれれば記録する。stub を外したときに canary が拾うことは、前の 4b の /test で同じ `setupOrgStubPATH` について確かめてある(`docs/reports/test-2026-10-22-seat-spawn-close.md` の 8)

## AC とテストの対応

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `tests/test-wip-checkpoint-hooks.sh`(108 件)、`TestWIPCheckpointOff_IsWhatTheHooksRead`(`internal/org/driver/paneenv_test.go:40`) | hook 2 本 × root と `templates/base/` の写し 2 本のそれぞれで、feature ブランチの汚れた木(追跡しているファイルの変更と未追跡のファイル)に、値なし・`on`・`OFF`・空なら件名どおりの `wip:` のコミットが 1 つ増え、両方のファイルが入り、md に skipped の行がない。`off` なら HEAD も index も変わらず、未追跡のファイルは未追跡のまま、md の最後の行が skipped の行(1 行だけ)。main の汚れた木と feature のきれいな木は、値なしでも `off` でもコミットせず、skipped の行もない。どの場合も exit 0 で、md の 1 行目は表題、`hook-failures.log` はない。Go のテストは hook 4 本が `WIPCheckpointOff` の名前と値をそのまま比べていることを確かめる。mutant W1〜W15 が赤 |
| AC2 | `TestSeatPaneEnv_ListsBothEntriesInOrder`(`paneenv_test.go:14`)、`TestSeatPaneEnv_ReturnsACopy`(26)、`TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt`(`internal/org/driver/herdr_test.go:321`)、`..._PaneEnvConcurrentCallsCheckOnce`、`..._PaneEnvCanceledCallerIsNotCached`、cli の `TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`(`internal/cli/org_runtime_env_test.go:80`)、`TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`(`internal/cli/org_test.go:424-429`)、`TestOrgDirectorStart_LaunchesRegistersAndStops`(`internal/cli/org_director_headless_test.go:208`) | 一覧は `AGMSG_SELF_NAME=off`、`RALPH_WIP_CHECKPOINT=off` の順で、呼ぶたびに新しい slice を返す。`--env` のある herdr では tab create ごとに `--label` のあとへ `--env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off` が付き、help は NewHerdr の値とその写しで 1 回だけ、8 並行でも 1 回。本番の組み立ての 2 か所(座席の runtime と headless の director)の PaneEnv が `driver.SeatPaneEnv()` と等しく、stub の herdr への呼び出しは help 1 回と tab create 2 回。CLI の spawn と director start の tab create の行が 2 つの `--env` で終わる。mutant D1〜D4、D9、C1〜C3 が赤 |
| AC3 | `TestDirectorLaunchScript_TextQuotesEveryValue`(`internal/org/director_cmd_test.go:443`)、`TestDirectorLaunchScript_ChecksInThenExecsTheAgent`、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn` | 起動スクリプトの全文が、`RALPH_DIRECTOR_TOKEN` のあと、check-in の行の直前に `AGMSG_SELF_NAME='off'`、`export AGMSG_SELF_NAME`、`RALPH_WIP_CHECKPOINT='off'`、`export RALPH_WIP_CHECKPOINT` をこの順で持つ(agmsg home があってもなくても)。端末の環境を 2 つとも `on` にしてスクリプトを本当に打つと、check-in の偽の ralph と偽の agent の両方が 2 つとも `off` を見る。`ralph org director start` の launch_cmd の経路でも agent が `wipcheckpoint=off` を見る。mutant O1〜O5 が赤(O3 を赤にしたのは全文のテストだけ。G2) |
| AC4 | `TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence`(`herdr_test.go:446`、6 subtest)、`TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce`(help に `--env` がない・help が失敗・help が時間切れ)、`TestHerdr_TabCreate_PaneEnvNilWarnWritesToStderr`、cli の `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`(spawn と director start) | `--env` のない herdr では tab create に `--env` が付かず、warning は 1 行だけで、PaneEnv のすべての値を挙げ、知っている値ごとの帰結を PaneEnv の順に `; ` でつなぎ、助言を `. Use ...` の別の文にする。値を逆の順にすると帰結も逆の順になり、知らない値は帰結を足さない。CLI の spawn と director start は成功し、stderr はちょうど 1 行で、2 つの帰結の語を含む。mutant D5〜D8、D10〜D12 が赤 |
| AC5 | `TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError`(`internal/org/prompts_test.go:1387`)、変えていない `TestRenderRolePrompt_Leader_FeatureOrgProcedure` | 運用規律の項が、終了コード 1 と `spawn failed:` で始まり、`rejected:` の断りを外し、BLOCKED の上げ方(TASK_ID は `org-a`)を先に書く。(a) `seat left active` では `--force` を足さずに stop を打ち、stop の終了コード 0 と status の `(active)` がないことを確かめてから打ち直し、stop が 1 で返るか座席が active のままなら BLOCKED。(b) `org: spawn step` で始まり `seat left active` を含まないときは、HELLO が届いていても座席は動いておらず、`retry with a larger --timeout-ms` があれば大きい値で打ち直す。それ以外のエラーは BLOCKED。(c) 2 回続けば BLOCKED。4 つの場合は spawn failed: の項の入れ子。項が引く語が Spawn の返すエラーにあることも、fake の herdr で agent start と close を失敗させて確かめる。手順の位置を固めるテストは変えずに通る。mutant L6〜L14、E1〜E3 が赤 |
| AC5b | `TestRenderRolePrompt_Leader_PushRefusesWipCommits`(`prompts_test.go:976`) | 手順 14 が `14. HEAD(` で始まり、5 つの句(`git log --format='%h %s' origin/<既定のブランチ>..HEAD` を打つこと、既定のブランチの決め方、`wip:` があれば gate も push も打たず BLOCKED、`wip:` の由来、HEAD が同じでも打つこと)を持ち、HEAD の比較 → `git log` → `wip:` の判定 → 打ち直し → push の順に並ぶ。hook 2 本の件名が `wip: ` で始まる。mutant L1〜L5 が赤 |
| AC6 | なし(実機の確かめ) | `docs/evidence/seat-wip-checkpoint-2026-10-22.md` の (a)〜(d) を読んだだけで、打ち直していない(本物の herdr と claude を使うため)。代わりに 8 の probe で、hook を `ralph-dispatch.sh SessionEnd` と `PreCompact` から打ち、dispatcher を通っても値が落ちないことを決まった手順で確かめた |
| AC7 | なし(文書) | /verify の範囲 |
| AC8 | なし(記録) | /verify の範囲 |
| AC9 | 1、1b、2 | 変えたパッケージの Go のテストが go1.26.0(1)と go1.25.8(2)で通る。run-verify の rc 0 は orchestrator の 9d29ee0d での実行(verify の報告) |

## 計画の境界の場合と、依頼の確かめの項目

| 項目 | テスト |
| --- | --- |
| 値が unset・`on`・`OFF`・空はコミットし、`off` だけ飛ばす | `test-wip-checkpoint-hooks.sh` の `case_commits`(4 値)と `case_off`。ほかの値(`Off`、` off`、`off `、`false`、`0`、`no`)は 8 の probe で、どれもコミットした |
| main ではコミットしない | `case_no_commit main dirty unset` と `... off` |
| きれいな木では何もしない | `case_no_commit feature/wip-test clean unset` と `... off` |
| 一覧の中身と順 | `TestSeatPaneEnv_ListsBothEntriesInOrder`、`TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`。順を入れ替えた D1 が赤 |
| `--env` が 2 つ付く、確かめは 1 回 | `TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt`、`..._ConcurrentCallsCheckOnce`、`..._CanceledCallerIsNotCached`、`TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`、spawn と director start の CLI のテスト |
| warning が値ごとに帰結を書く、`--env` の確かめの失敗 | `TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence`、`TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce`(help の失敗と時間切れ)、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` |
| 起動スクリプトが 2 つを export する | `TestDirectorLaunchScript_TextQuotesEveryValue`、`..._ChecksInThenExecsTheAgent`、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn` |
| 手順 14 の順 | `TestRenderRolePrompt_Leader_PushRefusesWipCommits`。`git log` の確かめを push の文のあとへ移した L1 が赤。手順 14 のコマンドが本物の hook の `wip:` のコミットを、そのあとに普通のコミットがあっても拾うことは 8 の probe で確かめた |
| 運用規律の (a)〜(c) | `TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError`。L6〜L14 が赤 |

## Mutant

hook の mutant は、hook 4 本と `tests/test-wip-checkpoint-hooks.sh` を scratchpad に写し、1 本ずつ文字どおりの置き換え(1 か所だけに当たることを確かめる)をしてテストを丸ごと打った。Go の mutant は `go test -overlay` で、driver は package 全体、org と cli は下の `-run` で打った。build に失敗した mutant はない。

| ID | 置き換え | 結果 |
| --- | --- | --- |
| W1 | session_end(root)の判定を「空でなければ飛ばす」に | 赤(`on`・`OFF` の 4 件) |
| W2 | 大文字と小文字を区別しない比較に | 赤(`OFF` の 2 件) |
| W3 | skipped の行を書かない | 赤 |
| W4 | `off` でも `git add -A` を打つ | 赤(index が変わる) |
| W5 | 比べる値を `of` に(`off` を認めない) | 赤 |
| W6 | `off` で行を書いたうえでコミットもする | 赤 |
| W7 | `off` なら main やきれいな木でも skipped の行を書く | 赤(3 件) |
| W8 | skipped の行の文を変える | 赤 |
| W9 | skipped の行で md を上書きする(`>>` を `>` に) | 赤(1 行目が表題でない) |
| W10 | コミットの件名を変える | 赤(4 件) |
| W11 | `git add -A` を `git add -u` に(未追跡のファイルが入らない) | 赤(4 件) |
| W12 | 写し(templates/base)の session_end だけ W1 | 赤 |
| W13 | precompact(root)で W5 | 赤 |
| W14 | precompact(写し)で W3 | 赤 |
| W15 | `main` を守る case から外す | 赤 |
| W16 | `master` を守る case から外す | 生き残り(G1) |
| D1〜D4 | `SeatPaneEnv` の順を入れ替える、共有の slice を返す、1 つ目だけ返す、`WIPCheckpointOff` の値を `=0` に | 4 本とも赤 |
| D5〜D8、D10〜D12 | WIP の帰結を返さない、帰結を `, ` でつなぐ、知らない値にも空の帰結を足す、1 つ目の帰結だけ書く、帰結が 2 つ以上のときだけ書く、warning の値の一覧を 1 つ目だけに、助言を小文字の `use` に | 7 本とも赤 |
| D9 | `TabCreate` が PaneEnv の 1 つ目だけ `--env` にする | 赤 |
| O1〜O2 | 起動スクリプトが一覧の 1 つ目だけ、2 つ目だけを export する | 赤 |
| O3 | 起動スクリプトが代入だけして `export` しない | 赤(全文のテストだけ。G2) |
| O4 | export を check-in の行のあとに置く | 赤 |
| O5 | export する値を変える | 赤 |
| L1〜L5 | 手順 14: `git log` の確かめを push の文のあとへ、「HEAD が同じでも」の句を消す、`wip:` を `WIP:` に、TASK_ID を org_id 以外に、「gate も push も」を「push を」に | 5 本とも赤 |
| L6〜L14 | 運用規律: `--force` を足さない指示を消す、stop の終了コード 0 の確かめを消す、「座席がまだ active」を消す、HELLO の文を消す、「それ以外のエラー」の項を消す、「2 回」を「3 回」に、(a) を入れ子から外す、BLOCKED の TASK_ID を座席の id に、「終了コード 1」を消す | 9 本とも赤 |
| E1〜E3 | spawn のエラーの `seat left active`(close の失敗)、`retry with a larger --timeout-ms`、stale の断りの `seat left active` の文を変える | 3 本とも赤 |
| C1〜C3 | 座席の runtime の PaneEnv を `AGMSG_SELF_NAME=off` だけに、headless の director の runtime を同じに、座席の runtime の順を入れ替える | 3 本とも赤 |

org の `-run` は `TestDirectorLaunchScript|TestStartDirector_LaunchCmd|TestDirectorCmd|TestStartDirector`(O)と `TestRenderRolePrompt`(L)と `TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError`(E)、cli の `-run` は 3 の cli の 4 本。生き残りの W16 は filter のない丸ごとの suite で打っている。

## Probe

scratchpad の一時の repo(`GIT_CONFIG_GLOBAL` を一時のファイルに、`GIT_CONFIG_NOSYSTEM=1`、`commit.gpgsign false`)で、HEAD の hook を打った。

| probe | 結果 |
| --- | --- |
| session_end を値 `Off`・` off`・`off `・`false`・`0`・`no` で(feature の汚れた木) | 6 通りとも `wip: checkpoint before session end` が 1 つ増え、skipped の行はない(D1 のとおり `off` ちょうどだけが飛ばす) |
| session_end を値なしで master の汚れた木 | コミットしない |
| `.claude/hooks/` を丸ごと写した repo で `printf '{}' \| ./.claude/hooks/ralph-dispatch.sh SessionEnd`、値なしと `off` | 値なしは `wip:` のコミットが 1 つ増え、`off` はコミットせず skipped の行が 1 行。どちらも rc 0 |
| 同じく `PreCompact` | 同じ結果(件名は `... context compaction`) |
| 手順 14 のコマンド: origin に main を push し、feature で普通のコミット、本物の session_end の `wip:` のコミット、普通のコミットの順に作ってから `git log --format='%h %s' origin/main..HEAD` | 3 行が出て、件名が `wip:` で始まる行は 1 つ。`wip:` のコミットのあとに普通のコミットがあっても拾う |

## Coverage

- Statement: internal/org/driver 95.2%(package 全体、`-count=1 -coverprofile`)。変えた関数は `SeatPaneEnv` 100%、`paneEnvConsequence` 100%、`warnNoPaneEnv` 100%、`NewHerdr` 100%、`TabCreate` 83.3%(当たらないのは envelope の解析の失敗の枝で、前からある)。`directorLaunchScript.text` は `TestDirectorLaunchScript|TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn` だけで 100%。`newOrgDirectorHerdrRuntime` は 100%、`newOrgRuntimeAt` は `TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv` だけで 85.7%(変えた行は当たる)
- Branch: Go と shell に branch coverage の道具は使っていない。hook の新しい枝(`off` の枝と、元の `git add -A` と `git commit` の枝)は、どちらも hook 4 本それぞれで 5 の suite が通る
- Function: org と cli の package 全体の coverage は取っていない(1 と 2 で丸ごと走らせたので、時間のかかる coverprofile の実行は省いた)
- Notes: 雛形(`leader.md`)と skill の文は coverage の対象にならない。雛形の文は mutant L1〜L14 で、テストが文の一つ一つに効くことを確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 試行の F11: 座席の Claude Code のセッションが、終わるときに機能の worktree の未コミットの変更を `wip:` でコミットした | 直った(新しい binary が `--env` のある herdr で作る pane と、新しい hook の組み合わせ) | 5、2 の AC2・AC3 のテスト、8 の dispatcher の probe。実機は AC6 の evidence(打ち直していない) |
| 4a(#248)の `AGMSG_SELF_NAME=off` を pane と起動スクリプトに付けること | 保たれている | 期待を 2 つ目の値に合わせただけの `TestHerdr_TabCreate_*`、`TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`、`TestDirectorLaunchScript_*`。agmsg 自身の実行の環境(`AGMSG_SELF_NAME=off` だけ)を固める `internal/org/driver/agmsg_test.go:269` は変わっておらず通る |
| 人が打つ対話セッションの WIP コミット(`RALPH_WIP_CHECKPOINT` を設定しない) | 前と同じ | 5 の値なしの行、8 の値なしの dispatcher の probe |
| 4b(#249)の spawn の失敗の扱い | 保たれている | 1 と 2 の org と cli の package 全体。E1〜E3 で、運用規律の項が引く語が spawn のエラーと結び付いていることも確かめた |
| 手順 14 の既存の句と手順の位置 | 保たれている | `TestRenderRolePrompt_Leader_FeatureOrgProcedure` は diff に入っておらず、3 で通る |

## Test gaps

- G1(LOW、前からの範囲): `tests/test-wip-checkpoint-hooks.sh` に master・unknown・detached HEAD の行がない。master を守る case から外した W16 が生き残った。この branch は `case` の行を変えていないので、今回の変更の穴ではない。master は 8 の probe で値なしのとき 1 回だけ確かめた。detached HEAD では `git rev-parse --abbrev-ref HEAD` が `HEAD` を返して `*)` の枝に入り、今までどおりコミットする(読んだだけで、打っていない)
- G2(LOW): 起動スクリプトを本当に打つテストは、端末の環境に 2 つの値を `on` で入れてから打つ。sh では環境から来た変数への代入は `export` がなくても子に渡るので、`export` を書かない O3 はこれらのテストでは緑で、全文を比べる `TestDirectorLaunchScript_TextQuotesEveryValue` だけが赤にした。全文のテストがあるので今は固まっているが、端末の環境に値がない場合を本当に打つ行を 1 つ足すと、全文のテストの書き方に頼らなくなる
- G3(LOW): `ralph org start --plan` の leader の pane の tab create を CLI から打って `--env` を見るテストはない。start は `newOrgRuntime` から `newOrgRuntimeAt` の Herdr を使い、座席の pane を作る `TabCreate` の呼び出しは `internal/org/spawn.go:979` の 1 か所なので、`TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv` と spawn の CLI のテストで間接に固まっている
- G4(文書、/verify の範囲): `/org` skill の手順 6 に足した `wip:` の確かめの文と、雛形の手順 14 の文を突き合わせるテストはない
- G5: AC6 の実機の確かめ(本物の herdr と claude)は打ち直していない。hook と settings.json が S1 のあと変わっていないことは verify の報告が `git diff --stat` で確かめている
- G6: 雛形の手順 14 と運用規律は leader への指示で、本物の leader が従うかは確かめていない(計画の Risks のとおり)

## Verdict

- Pass: AC1〜AC5b と AC9 の behavioral test がすべて通った(go1.26.0 と go1.25.8、`-race`、`-count=3`、canary)。mutant 53 本のうち 52 本が赤。生き残りの W16 は今回変えていない枝(G1)
- Fail: なし
- Blocked: なし

## Addendum(2026-10-22、sync-docs)

この報告のあとの 62f92f42(`test:` の 1 コミット)が G1 と G2 を埋めた。変えたのは `tests/test-wip-checkpoint-hooks.sh` と `internal/org/director_cmd_test.go` だけで、hook・コード・雛形は変えていない。上の表と Verdict は /test の時点のまま残し、ここに足すだけにする。sync-docs が 62f92f42 の tree で打ち直したのは、hook のテスト(`bash tests/test-wip-checkpoint-hooks.sh`)と、下の W16・O3 の 2 本の mutant だけで、Go のテスト全体とほかの mutant は打ち直していない。

- G1(埋めた): hook のテストは 108 件から 156 件になった(rc 0、156 passed、0 failed)。hook 4 本それぞれに、master の汚れた木(値なしと `off`。どちらもコミットなしで skipped の行もない)と、detached HEAD の汚れた木(値なしでその HEAD にコミットが 1 つ、`off` でコミットなし)の行を足した。detached HEAD では `git rev-parse --abbrev-ref HEAD` が `HEAD` を返し、`unknown` にならないので、hook は feature ブランチとして扱って `wip:` のコミットを作る。これは前からの振る舞いで、変えずにテストで固めた。tech-debt に行を足した(「The session-end and pre-compaction WIP hooks make a `wip:` commit on a detached HEAD」)。W16(master を守る case から外す)は、session_end(root)の master の 2 件(値なしと `off`)が赤になり、生き残らなくなった(hook 4 本とテストを scratchpad に写して打ち直した。154 passed、2 failed)。`unknown` の枝(`git rev-parse` が失敗するとき)を打つ行は足していない
- G2(埋めた): `TestDirectorLaunchScript_ChecksInThenExecsTheAgent` に subtest `the terminal passes neither value` を足した。端末の環境に 2 つの値がない状態で起動スクリプトを打ち、check-in の環境と agent の両方が `selfname=off` と `wipcheckpoint=off` を見ることを確かめる。O3(`export` を書かない)は、この subtest が赤にする(`go test -overlay` で打ち直した。`-run 'TestDirectorLaunchScript_ChecksInThenExecsTheAgent|TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn'` で落ちたのはこの subtest だけ)。全文を比べる `TestDirectorLaunchScript_TextQuotesEveryValue` に頼らずに固まる
- G3〜G6: 変わらず残る
- mutant: /test の 53 本のうち、生き残りだった W16 は赤になった。ほかの 52 本は打ち直していない。hook のテストの書き方が変わったので、W1〜W15 の 62f92f42 のあとの結果は未確認
