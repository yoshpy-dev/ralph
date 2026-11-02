# Test report: config-path-and-ledger-docs

- Date: 2026-11-02(run-test.sh は 16:40:01 から 17:11:12 JST)
- Plan: `docs/plans/active/2026-11-02-config-path-and-ledger-docs.md`(`## Test plan`、AC1〜AC3、AC5 のテスト、AC6 の実物の確かめ、AC7 のうち run-test.sh)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD 81e469b4、merge-base は origin/main 350f22ea。実行中に branch は動かなかった)。Go は `internal/cli`(`org.go` の `resolveConfig`、`status.go` の warning、テスト 3 ファイル)と `internal/config`(doc だけ)。`./scripts/run-test.sh` は changed scope のまま打ち、Language scope は changed(changed_languages)で golang pack が選ばれた(`go test ./...` の全パッケージ)。test mode なので `tests/test-*.sh` の 45 本もすべて走った
- Evidence:
  - `docs/evidence/test-2026-11-02-config-path-and-ledger-docs-live.md`(コミットする。AC6 の実物の確かめの手順、出力、片付け)
  - `docs/evidence/test-2026-11-02-config-path-and-ledger-docs.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。`run-test.sh` の全出力、名指しのテストの `-v` の出力、変更行の coverage、変異の 2 回の結果、A の codex の log の頭。5,818 行)
  - wrapper 自身のログは `docs/evidence/verify-2026-11-02-074002.log`(同じく対象外。名前の日付は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | shell 2 行(下の注)。Go は `-v` なしなので数えていない | 1,871 秒。load は開始時 3.56 / 4.37 / 5.24、終了時 5.01 / 5.97 / 6.02 |
| └ Go: `internal/org` | — | ok | 0 | — | 1,251.3 秒 |
| └ Go: `internal/cli` | — | ok | 0 | — | 1,002.9 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/insights`、`internal/upgrade` | — | ok | 0 | — | 43.6 秒、3.9 秒、3.1 秒、3.9 秒 |
| └ Go: `internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`cmd/ralph` | — | ok | 0 | — | 4.4 秒、1.6 秒、1.3 秒、0.8 秒 |
| 名指しのテスト、`internal/cli`(`go test -count=1 -v`、下の「Test plan とテストの対応」) | 親 11、サブテスト 34 | 親 11、サブテスト 34 | 0 | 0 | 19 秒 |
| 変更行の coverage(`-coverprofile`、`-run` で 6 本に絞った) | — | ok | 0 | — | 12 秒 |
| 変異、1 回目(5 つを並列、`-run` は計画の S1 の verify の範囲) | 5 | — | — | — | 5 つとも 15 分の時間切れ。判定なし(下の Regression checks) |
| 変異、2 回目(1 つずつ、名指しのテストに 2 本を足した 13 本) | base + 5 | base ok | 変異 4 つが落ちた | — | base 51 秒、各 22〜24 秒 |
| AC6 の実物の確かめ | A + B の 2 つの場合 | A、case 1 | — | case 2 は前提が整わず未確認 | 下の「AC6: 実物の確かめ」 |

起動の仕方: `run-test.sh` は Bash tool の 600 秒の上限を超えるので、`run_in_background` で runner を起動し、log と開始・終了・rc の meta を scratchpad に書かせた。待つあいだに名指しのテストと AC6 を進め、同じ手番の中で meta を読んだ。完了の通知もそのあとで届いた。

所要時間について: run-test.sh の最中に打ったのは、名指しのテスト(16:40:31 から 19 秒)と AC6(16:45〜17:09。herdr の server、claude と codex の session、`ps` を 0.5 秒ごとに見る記録)で、Go のテストと重なったのは名指しのテストだけ。`internal/org` の 1,251 秒は 30 分の上限(1,800 秒)の 70% で、org-ledger-leftovers の回の 1,250 秒とほぼ同じだった。時間切れにならず、既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` も落ちなかったので、run-test.sh は打ち直していない。10 パッケージのどれも `(cached)` ではなかった。

Skipped について: shell の 2 行は、どちらもこの diff と関係のない手元の環境による。`tests/test-plan-visual.sh` の「browser detection (a browser in /Applications would start)」と、`tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0 で、古い git は stub の行が見ている)。

## Coverage

- Statement: この diff が足した非テストの行のうち、コメントでない行は 6 行(`internal/cli/org.go` の `resolveConfig` の 666〜670 行と、`internal/cli/status.go` の 313 行の warning)。絞った回の cover profile で 6 行とも実行された。`internal/config/config.go` と `internal/cli/org_legacy_ledger.go` の変更はコメントだけ
- Branch: 計測なし(Go の道具は statement だけ)
- Function: `resolveConfig` 100%。`statusOrgConfig` 85.7%(実行されなかったのは、読み込みが通ったときの `return cfg, true` の 1 ブロックで、この diff は変えていない。絞った回が読み込みの失敗の行だけを打ったため)
- Notes: パッケージ全体の数字(絞った回の 9.5%)には意味がないので使わない

## Failure analysis

run-test.sh、名指しのテスト、coverage の回には失敗がない。

変異の 1 回目は 5 つとも `panic: test timed out after 15m0s` で止まった。どれも `--- FAIL` の行を出す前で、止まったときに走っていたのは `TestOrgStart_MainWorktreeRalphTomlLoadError` などの変異と関係のないテストだった。同じ filter の base は単独で 185 秒だったが、5 つを並列に打ったため遅くなった(17:18 の 1 分の load は 22.88)。テストの失敗ではなく、打ち方の誤りとして判定に使わず、2 回目を 1 つずつ、狭い filter で打ち直した(下の Regression checks)。

AC6 の case 2 は、テストの失敗ではなく実物の確かめの前提が整わなかったもの(下の「AC6: 実物の確かめ」)。

## Test plan とテストの対応

この branch が足したテストと、変えたテストは、名指しの回ですべて PASS した。

| 計画の項目 | テスト | 結果 |
| --- | --- | --- |
| AC1: root と `--config` の壊れた設定の文 | `TestOrgStartPlan_UnreadableConfigStartsNothing`(the main worktree's ralph.toml・from its root・the --config file) | PASS |
| AC1: spawn、`start <task>`、`director start` の同じ道 | `TestOrgSpawnStartAndDirectorStart_NameTheConfigFileTheyCouldNotLoad`(3 つの動詞 × `./ralph.toml` と `--config` の 6 サブテスト) | PASS |
| AC1、Risks の「パスの二重」: 2 つの形を固定する | `TestResolveConfig_PrefixesTheFileItWasAskedToRead`(a ralph.toml that cannot be opened・a ralph.local.toml that is not valid TOML) | PASS |
| a6be3f5e: `ralph status` の warning が名前を 2 回出さない | `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(4 サブテスト)、`TestStatusCmd_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad` | PASS |
| AC2: helper `commitRalphTomls` を使うテスト | `TestOrgStartPlan_ReadsEachConfigFileOnce`(4)、`TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir`(2)、`TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml`、`TestOrgStartPlan_OutsideLedger_RecordsTheApprovalModeOfTheFileItRead` | PASS |
| AC3: `resolveOrgLedger` のエラーの道 | `TestOrgStartPlan_RefusedLedgerStartsNothing` | PASS |
| AC3: `max_total_seats` の対照 | `TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml`(max_orgs・max_total_seats・--config・--config, max_total_seats・a ledger outside the main worktree・同 max_total_seats) | PASS |
| AC3: 外の台帳の承認方式 | `TestOrgStartPlan_OutsideLedger_RecordsTheApprovalModeOfTheFileItRead`(4 サブテスト) | PASS |
| AC5: README と skill の写しが定数の文を含む | `TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs`(README.md・.claude・templates) | PASS |
| AC6: 実物の確かめ | 下の節 | 一部 |
| AC7 のうちテスト、Integration、Regression | `./scripts/run-test.sh` 全体 | rc 0 |

## AC6: 実物の確かめ

手順、出力、片付けは `docs/evidence/test-2026-11-02-config-path-and-ledger-docs-live.md` にある。HEAD 81e469b4 から build した ralph(`ralph test-cfgpath (81e469b4 2026-11-02)`)を使った。PATH の `/opt/homebrew/bin/ralph` は 6.0.0 で、`config get` がない。

| 確かめ | 前提 | 結果 | 判定 |
| --- | --- | --- | --- |
| A: 既定と違う codex の slug(D3) | 一時の repository の `ralph.toml` の `[reviewer]` に `codex_model = "gpt-6-luna"`(models_cache で `upgrade` が null)、`codex_effort = "low"`。build した ralph を PATH の先頭に | helper は `gpt-6-luna`、`low`(`ralph.toml` のない場所では既定の `gpt-6-astra`、`xhigh`)。`command codex -m … -c model_reasoning_effort=… exec --sandbox read-only --ignore-rules -o …` は rc 0。session の記録 `rollout-2026-11-02T16-46-21-01a19b39-….jsonl` の `turn_context` は `model=gpt-6-luna`、`effort=low`。main checkout の `.codex/config.toml` は前後で `git status` が空、sha256 の頭は `96919613aeacafd5` のまま | 満たした |
| B case 1: claude の leader → codex の reviewer | 使い捨ての pane で feature の worktree の `command -v ralph` は build した ralph、`ralph config get org.review.plan_advisory` は `true`。claude の Bash ツールも build した ralph を見た(`CLAUDE_ENV_FILE`)。タスクの行は `- reviewer の codex: gpt-6-luna low(…)` と `- reviewer の claude: sonnet low(…)` | leader は計画をコミットしたあと codex を `-m gpt-6-luna -c model_reasoning_effort=low exec --sandbox read-only --ignore-rules -o .harness/state/plan-advisory-out.md "<指示文>"` で起こした(引数を記録する shim と `ps` の記録の両方)。session の記録の `turn_context` も `gpt-6-luna`、`low`。助言は完了(出力 1,153 バイト、`git: same`)し、leader は `ADVISORY:` の行つきの承認の QUESTION を上げて待った。start から 2 分 20 秒 | 満たした |
| B case 2: codex の leader → claude の reviewer | pane の shell と worktree の `plan_advisory`(`true`)は整った。ただし codex の shell ツールは zsh で打ち、ユーザーの `~/.config/zsh/.zshenv` が PATH を組み直すので、leader が打つ `ralph` は `/opt/homebrew/bin/ralph`(6.0.0)になった。server と pane の環境では直せず、偽の `HOME` は claude に使えないので使わなかった | leader は 17:03:49 に `ralph config get org.review.plan_advisory` を打ち、`unknown command "config" for "ralph"`(exit 1)を得た。役割指示の規則では off とみなす段で、claude は起こされなかった。そのあと leader の手番は「■ Conversation interrupted」で止まり(17:06:11、原因は未確認)、QUESTION は上がらなかった | 確かめていない(前提が整わない) |
| 補い(leader の通しではない) | case 2 の役割指示から計画助言の claude の囲みを取り出し、タスクの `reviewer の claude:` の行の値(`sonnet low`)で埋めて手で打った | 起きた claude の引数は `-p --model sonnet --effort low --restricted --strict-mcp-config --disallowedTools mcp__* --tools Read,Grep,Glob --permission-mode dontAsk --output-format text …`。`claude rc=0`、`git: same`。session の記録の model は `claude-sonnet-5-5` | case 2 の判定には使わない |

AC6 は一部満たした。A と case 1 は満たし、case 2 は前提が整わずに確かめていない。計画の D3 と AC7 のとおり、case 2 は「確かめていない」として tech-debt 421 行に残す材料になる。

ほかに見えたこと(証跡の文書に詳しい):

- case 2 で codex が出した「Hooks need review」のダイアログに「Continue without trusting (hooks won't run)」で答えた時刻に、ユーザーの `~/.codex/config.toml` に `[projects."/private/tmp/rcfgtcl99/work"]` の `trust_level = "trusted"` が足された。消していない
- case 2 の leader の codex が起こした codex の app-server daemon(ユーザーごとに 1 つ)が、片付けのあとも cwd を消したディレクトリに向けて残っている。ほかのセッションが使っているかもしれないので止めていない
- case 2 の leader は、ralph の役割指示にない動き(ユーザーの codex の指示による相談の sub-agent、natural-japanese の lint)に手番を使い、effort はユーザーの既定の `max` だった
- 専用の herdr server は止め、`pgrep -fl 'herdr server'` は前(16:45:19)も後(17:09:37)も何も出さなかった。ユーザーの agmsg の `messages.db` と `-wal`、`~/.config/herdr` は変わっていない

## Regression checks

変異は `go test -overlay` で `internal/cli` の変更行に入れた。2 回目は名指しの 11 本に、`org: load config: ` を `HasPrefix` で見る `TestOrgWaitDirector_RefusesABadIDBeforeItLoadsTheConfig` と `TestOrgStart_MainWorktreeRalphTomlLoadError` を足した 13 本で打った。base は rc 0。

| 変異 | 結果 | 落としたテスト |
| --- | --- | --- |
| m1: `resolveConfig` が包まずに `return cfg, err`(base の形) | 落ちた | `TestOrgStartPlan_UnreadableConfigStartsNothing`(from its root・the --config file)、`TestOrgSpawnStartAndDirectorStart_NameTheConfigFileTheyCouldNotLoad`(6 つすべて)、`TestResolveConfig_PrefixesTheFileItWasAskedToRead`(2)、status の 2 本 |
| m2: 前に付ける名前を `path` から `configPath` に(`--config` がないとき空) | 落ちた | 上の `./ralph.toml` と root のケース、`TestResolveConfig_…`、status の 2 本 |
| m3: status の warning を `load ralph.toml: %v` に戻す(名前が 2 回出る) | 落ちた | `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(4。a6be3f5e が足した `ralph.toml: ralph.toml` の条件) |
| m4: `%w` を `%v` に | 生き残った | — |
| m5: 前に付ける名前を `filepath.Abs(path)` に | 落ちた | `./ralph.toml` と root のケース、`TestResolveConfig_…`、status の 2 本 |

m4 は文の bytes が同じで、`resolveConfig` と `resolveOrgConfig` のエラーを `errors.Is` や `errors.As` で開く呼び出し元は本番のコードにない(`internal/cli` の `errors.Is` は `ralph config get` の `ErrConfigLoad` などで、`resolveConfig` の道を通らない)。今の振る舞いでは区別できない変異とみなした。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 421 (a)(b): org の動詞の壊れた設定のエラーが読んだファイルを名指さない | 直った | m1(base の形)で上のテストが落ち、HEAD で通る |
| S1 のあとの `ralph status` の warning の名前の二重(a6be3f5e) | 直った | m3 で落ち、HEAD で通る |

worktree の `git status --porcelain --ignored` は、変異の前後で証跡の文書の 1 行のほかに変わっていない(`.harness/cache` は除いた)。

## Test gaps

1. AC6 の case 2(codex の leader が `reviewer の claude:` の行の値で claude を起こす)は実物で確かめていない。この機械では codex の shell ツールが zsh でコマンドを打ち、ユーザーの `~/.config/zsh/.zshenv` が PATH を組み直すため、leader の `ralph` が 6.0.0 になる。tech-debt 192 行(leader が PATH の `ralph` を使う)と同じ根で、レシピ `docs/recipes/isolated-herdr-server.md` も codex の座席の PATH を扱っていない。/sync-docs で 421 行に「確かめていない」と理由を残す材料にする
2. 補いの手動の実行は leader の通しではない。leader の claude の囲みの値の埋め方(行の頭のあと括弧の前の 2 語)は、Go のテストと補いの実行でしか確かめていない
3. claude の `--effort` は claude の session の記録に出ないので、引数の記録でしか確かめていない
4. m4(`%w` を `%v`)は今のテストでは区別できない。呼び出し元がエラーを開かない今は振る舞いが同じなので、直すものはない
5. 変異の 1 回目は打ち方の誤り(並列と負荷による時間切れ)で判定がなく、2 回目は 13 本の狭い filter でだけ打った。`internal/cli` の全体で変異を打ってはいない
6. Go の SKIP の数は `-v` なしの run-test.sh では数えていない(既知の Linux だけの 2 本は darwin で SKIP する)
7. `internal/org` は 1,251 秒で、30 分の上限の 70% だった。AC6 の claude と codex の session と重なっていた

## Verdict

- Pass: run-test.sh rc 0(shell 45 本、Go 10 パッケージ、`(cached)` なし)、この branch が足したテストと変えたテストの親 11 本・サブテスト 34 本、変更行の coverage 6/6、変異 5 つのうち 4 つが落ちた(残る 1 つは区別できない変異)、AC6 の A と case 1
- Fail: 0
- Blocked: 0(テストについて。AC6 の case 2 は前提が整わず確かめていない。上の AC6 と Test gaps 1)
- Verdict: pass。/sync-docs へ進んでよい。AC6 は一部満たした(A と case 1 は満たし、case 2 は確かめていない)ので、/sync-docs が tech-debt 421 行に case 2 を「確かめていない」として残す
