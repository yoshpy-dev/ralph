# Test report: standard-config-gaps

- Date: 2026-11-02(run-test.sh は 02:07:12 から 02:24:41 JST)
- Plan: `docs/plans/active/2026-11-02-standard-config-gaps.md`(`## Test plan`、AC5、D4、AC6 のうち run-test.sh)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff d217c575...HEAD`(HEAD 60838e34。origin/main は d217c575 のまま)。Go は `internal/cli`、shell は `tests/test-ralph-config.sh`。`./scripts/run-test.sh` は changed scope のまま打ち、`tests/test-ralph-config.sh` が分類できないので full fallback になった(選ばれた pack は golang)。test mode なので `tests/test-*.sh` の 45 本もすべて走った
- Evidence:
  - `docs/evidence/test-2026-11-02-standard-config-gaps-live.md`(コミットする。AC5 の実物の確かめのコマンドと出力)
  - `docs/evidence/test-2026-11-02-standard-config-gaps.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。`run-test.sh` の全出力 5,617 行、変異 m1 の定義と 2 回の結果、coverage の数字)
  - wrapper 自身のログは `docs/evidence/verify-2026-11-01-170713.log`(同じく対象外。名前の日付は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope → full fallback、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(下の注) | 1,049 秒。load は開始時 2.48 / 3.35 / 3.53、終了時 2.89 / 4.09 / 4.27 |
| └ shell: `tests/test-ralph-config.sh` | 272 | 272 | 0 | 0 | — |
| └ Go: `internal/org` | — | ok | 0 | 0 | 480.8 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 332.5 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/org/protocol`、`internal/insights` | — | ok | 0 | 0 | 11.1 秒、4.5 秒、4.1 秒、3.9 秒 |
| └ Go: `internal/upgrade`、`internal/scaffold`、`internal/org/driver`、`cmd/ralph` | — | ok | 0 | 0 | 2.8 秒、1.8 秒、1.5 秒、0.7 秒 |
| `tests/test-ralph-config.sh` を `sh`・`bash`・`zsh -f`・`dash` で | 272 × 4 | 4 つとも 272/272、rc 0 | 0 | 0 | — |
| coverage の回: `go test -count=1 -coverprofile -run 'OrgStart\|OrgFeature\|OrgReserve\|OrgSpawn\|Approv\|OrgWideLimits' ./internal/cli/` | — | ok | 0 | 0 | 80.2 秒 |
| 変異 m1(`go test -overlay`、下の Regression checks) | 1 | 生き残った | — | — | 絞った回 74.4 秒、`internal/cli` 全体 513.2 秒 |
| AC5 の実物の確かめ | 3 項目 | 3 | 0 | 0 | 下の「AC5」 |

起動の仕方: `run-test.sh` は Bash tool の 600 秒の上限を超えうるので、lead の指示どおり `run_in_background` で起動し、log と開始・終了・rc の meta を書かせた。待つあいだに AC5 を進め、同じ手番の中で wrapper の pid を `caffeinate -w` で待って meta を読んだ。完了の通知もそのあとで届いた。

Skipped の 2 行は、どちらもこの diff と関係のない手元の環境による。

- `tests/test-plan-visual.sh` の「browser detection (a browser in /Applications would start)」。`/Applications` にブラウザがある機械で、本物のブラウザを起こさないための SKIP
- `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」。手元の git は 2.49.0 で、古い git は stub の行が見ている

`go test ./...` の 10 パッケージは、どれも `(cached)` ではなかったので打ち直していない。30 分の時間切れにはならず、既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` も落ちなかったので、打ち直しはしていない。

## Coverage

- Statement: 絞った回の profile なので、パッケージ全体の数字(18.0%)には意味がない。変更した行を含む関数は次のとおり。
  - `applyMainWorktreeOrgLimits` 100%、`withMainWorktreeOrgLimits` 100%、`loadMainWorktreeConfig` 100%、`newOrgRuntimeFromConfig` 100%、`resolveConfig` 100%、`resolveOrgStartConfig` 100%、`runOrgStartPlan` 90.5%、`newOrgSpawnRuntimeAt` 87.5%、`newOrgRuntimeAt` 75.0%
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes: 打たれなかったブロックは 4 つで、どれも main からある error の return。`internal/cli/org.go:163`(spawn の runtime の設定の読み込みの失敗)、`:381`(`newOrgRuntimeAt` の読み込みの失敗)、`:1207`(start --plan の `resolveOrgLedger` の失敗)、`:1225`(`org.FeatureRepoRoot` の失敗)。この diff が足した行(`resolveOrgStartConfig` の 2 つの読み込みと error の文、`runOrgStartPlan` の runtime の組み立てと `InboxHintStateDir`)はすべて打たれている。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

`run-test.sh`、4 つのシェルでの `tests/test-ralph-config.sh`、coverage の回のどれにも失敗はなかった。

## Test plan とテストの対応

| 計画の項目 | テスト |
| --- | --- |
| AC2: 読み込みの回数(main の経路、main の root、`--config`、外の台帳) | `TestOrgStartPlan_ReadsEachConfigFileOnce`(`internal/cli/org_start_plan_config_test.go:63`、4 ケース) |
| AC2: 既定でない台帳で残った leader の止める案内 | `TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir`(同 `:205`、`--state-dir` と `[org].state_dir`) |
| AC2: 決まる承認方式と `[reviewer]` が今と同じ | 既存の `TestOrgStartPlan_ApprovalModeFromMainWorktree`、`TestOrgStartPlan_ReviewerFromMainWorktree`(S1 で `org_feature_reviewer_test.go` へ移した)が run-test.sh の中で通った |
| AC3: 読めない main の `ralph.toml` と `--config` | `TestOrgStartPlan_UnreadableConfigStartsNothing`(同 `:127`、2 ケース。文の頭、原因が `toml: `、worktree・ブランチ・記録・manifest・herdr のどれもない) |
| AC4: `zsh -f` | 上の 4 つのシェルの回 |
| Edge cases: main の `ralph.local.toml` だけが `[reviewer]` を変える | 既存の `TestOrgStartPlan_ReviewerFromMainWorktree` |
| 回帰: spawn の上限 | 既存の `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml`、`TestOrgSpawn_MainWorktreeRalphToml_OnlyOrgWideLimitsTaken` が通った |

## Regression checks

implementer は S2 と F1 で変異を確かめている(計画の Progress: 数えるテストは元のロジックで 3 ケースとも落ちた、`InboxHintStateDir` を外すと落ちた、main の値を当てない変異と cwd の `[reviewer]` を使う変異は落ちた、F1 の `-overlay` の変異 3 つも落ちた)。この回では、start --plan が spawn と別の経路で上限を当てるようになった点を選び、呼ぶ所の変異を 1 つ作った。worktree のファイルは変えず、1 行の置き換えを `go test -overlay` で渡した。build は通り、前後の `git status --porcelain --ignored` は go の build cache(`.harness/cache/`)を除いて同じだった。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| m1 start --plan が main worktree の `max_orgs` と `max_total_seats` を当てず、承認方式だけを当てる(`internal/cli/org.go:715` の `applyMainWorktreeOrgLimits(cfg.Org, mainCfg.Org)` を、`cfg.Org` の `Approval` だけを main の値にしたものに置き換え) | 生き残った | `-run 'OrgStart\|OrgFeature\|OrgReserve\|OrgSpawn\|Approv\|OrgWideLimits'` で ok(74.4 秒)、`internal/cli` 全体でも ok(513.2 秒) |

`applyMainWorktreeOrgLimits` そのものは spawn と plain start のテストが見ている。start --plan が上限を main worktree から取ることを固定するテストはない。`internal/cli` のテストで start --plan を打つ 7 ファイル(`org_feature_test.go` ほか)に、`max_orgs` か `max_total_seats` を書く fixture はなかった(help の文を固定する 2 か所と、AC3 の error の文だけ)。base では start --plan が plain start と同じ `newOrgSpawnRuntimeAt` を通っていたので、上限の規則は plain start のテストで覆われていた。この diff で start --plan が自分の経路を持ったため、その経路の上限の当て方を見るテストがなくなった。下の Test gaps の 1 に書く。

## AC5: PR #291 の Known gaps の実物の確かめ

コマンドと出力は `docs/evidence/test-2026-11-02-standard-config-gaps-live.md` にある。

| 項目 | 結果 |
| --- | --- |
| (a) v6.0.0 での扱い | PATH の ralph は `/opt/homebrew/bin/ralph`、`ralph 6.0.0 (44c14b97 …)`。作業ツリーの root で `ralph_setting reviewer.codex_model` は `gpt-6-astra`、`ralph_setting reviewer.claude_effort` は `xhigh`、`ralph_setting_or_fail standard.base_branch` は空で、3 つとも rc 0、stderr は空(zsh と sh で同じ)。`ralph config get standard.cross_review` は exit 1、stderr は `unknown command "config" for "ralph"`。`/cross-review` step 0 と `/plan` step 11.a の規則では「ralph without config get; treated as off」の skip になる(skill は回していない) |
| (b) fable の `--effort` | `In one sentence, why is the sky blue?` を low・medium・high・xhigh で 1 回ずつ。4 つとも rc 0、`modelUsage` は `claude-fable-5-1`、`is_error` false、`.err` は空。thinking は 4 つとも 0、output は 69・68・69・97。この質問では効き目が見えないので、追加で計算の質問を low と xhigh で打つと、thinking は 0 と 78 に分かれた。`-p --effort` は fable にも届いている |
| (c) 設定の値で claude と codex を起こす | 一時の repository(checkout の外、`git init`)の `ralph.toml` に haiku・low・gpt-6-astra・low を書き、HEAD から build した ralph を PATH の先頭に置いた。`ralph_setting` の 4 つはその値を返した(`ralph.toml` のない所では fable・xhigh)。claude は `modelUsage` が `claude-haiku-5-5`、`is_error` false、答え `ok`。codex は rc 0、`-o` は `ok`、ログの頭は `model: gpt-6-astra` と `reasoning effort: low` |
| main の `.codex/config.toml` | codex の前後とも `git status --short -- .codex/config.toml` は空、`shasum -a 256` も同じ。書き換わっていない |
| 後片付け | 一時の repository、build した binary、(a)(b)(c) の出力、読み取りの script を evidence を書いたあとで消し、`test -e` で消えたことを確かめた。codex と watchdog の `sleep 300` のプロセスも残っていない |

## Test gaps

1. start --plan の上限(`max_orgs`、`max_total_seats`)を main worktree の `ralph.toml` から取ることを、どのテストも固定していない。変異 m1 は `internal/cli` 全体で生き残った。本番のコードは正しく当てている(`resolveOrgStartConfig` が `applyMainWorktreeOrgLimits` を呼び、その関数は spawn のテストが見ている)。閉じるには、`TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml` と同じ形のテストを start --plan に 1 本足す: main の `ralph.toml` に `max_orgs = 1`、サブディレクトリの `ralph.toml` に `max_orgs = 99`、org-a が動いている状態で、サブディレクトリから start --plan を打つと `max_orgs 1 reached` で断られる。`max_total_seats` の行も同じ形で足せる。PR の前に足すかは lead の判断
2. coverage の Notes の 4 つの error の return(`org.go:163`、`:381`、`:1207`、`:1225`)は、絞った回では打たれなかった。どれも main からある分岐で、この diff はそこを変えていない。全体の回の coverage は測っていない
3. verify の V1(main worktree の root から打つと、壊れた同じ `ralph.toml` を cwd の読み込みが先に読み、文が `org: load config:` になる)はテストがない。base でも同じ文で、計画は tech-debt と PR の Known gaps に残す
4. AC5 の限り: どの値も 1 回ずつで統計ではない。(b) の効き目の判断は追加の質問の low と xhigh の 1 組に拠り、medium と high は追加の質問で打っていない。(c) の codex の model は ralph の既定と同じ値なので、model の値が `ralph.toml` から届くことは helper の読み取りと effort の結果から推している。skill(`/cross-review`、`/plan`)そのものは回していない。feature org の leader の通しは計画の Non-goals のとおり打っていない

## Verdict

- Pass: run-test.sh rc 0(shell 45 本、Go 10 パッケージ、`(cached)` なし)、`tests/test-ralph-config.sh` は 4 つのシェルで 272/272、変更した関数の coverage、AC5 の 3 項目と `.codex/config.toml` の確認
- Fail: 0
- Blocked: 0
- Verdict: pass。/sync-docs へ進んでよい。Test gaps の 1(start --plan の上限を固定するテストがない、変異 m1 が生き残る)は、PR の前に 1 本足すか Known gaps に書くかを lead が決める
