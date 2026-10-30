# Test report: standard-flow-config

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-standard-flow-config.md`(`## Test plan` と AC1〜AC8、Progress の S1〜S9)。Refs #276(B1)
- Tester: tester サブエージェント(`/test`、cycle 1。`cycle-count.json` は 1)
- Scope: `git diff origin/main...HEAD`(merge-base 0351800d、HEAD 5e400dac)。Go は `internal/config`、`internal/org`、`internal/cli`。shell は `scripts/ralph-config.sh`(と写し)、`tests/test-ralph-config.sh`、`tests/test-codex-exec-invocation.sh`、`tests/test-insights-append.sh`。ほかに skill、rules、文書、template。`./scripts/run-test.sh` は changed scope のまま打った。test mode は `tests/test-*.sh` を glob で全部打つので、shell の 45 本がすべて走った。テストの関数はこの diff で 19 本増え、消えたものはない
- Evidence: `docs/evidence/test-2026-10-30-standard-flow-config.md`(要点。commit する)、`docs/evidence/test-2026-10-30-standard-flow-config.log`(gitignore。`run-test.sh` の全出力、追加の回の出力、変異の差分、coverage の関数の行)。wrapper 自身の写しは `docs/evidence/verify-2026-10-30-120355.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 10 パッケージ ok(`(cached)` なし) | 0 | 1(`test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行。手元は 2.50.1) | 1,587 秒(21:03:55〜21:30:22 JST)。load は開始 6.80 / 7.65 / 8.31、終了 6.90 / 8.68 / 8.74、途中の見本 9.57、7.65、13.55 |
| └ Go: `internal/org` | 1,331 本(トップレベル、`go test -list`) | ok | 0 | 0 | 907.7 秒 |
| └ Go: `internal/cli` | 926 本(同) | ok | 0 | 0 | 569.0 秒 |
| └ Go: `internal/config` | 100 本 | ok | 0 | 0 | 54.0 秒 |
| └ Go: `cmd/ralph`、`internal/{gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 0.8〜6.6 秒 |
| `sh tests/test-ralph-config.sh` | 370 | 370 | 0 | 0 | 21 秒 |
| `bash tests/test-ralph-config.sh`(3.2.57) | 370 | 370 | 0 | 0 | 18 秒 |
| `zsh -f tests/test-ralph-config.sh`(5.9) | 55 | 13 | 42(rc 1) | 0 | 1 秒。原因は下の Failure analysis |
| `zsh --emulate sh -f tests/test-ralph-config.sh` | 370 | 370 | 0 | 0 | 26 秒 |
| `go test ./internal/config/ -run '^TestLoad_ReviewerValidation$' -count=3 -v` | 116 の名前 × 3 | 348 | 0 | 0 | 2 秒(load 7.09) |
| `go test ./internal/org/ -run '^TestStartFeature_RefusesReviewerValuesThatSplitTheShell$' -count=3 -v` | 6 の名前 × 3 | 18 | 0 | 0 | 1 秒(load 7.14) |
| 変異 5 個(shell 2 個は `git archive` の写し、Go 3 個は `go test -overlay`) | 5 | 5 個とも落ちた | — | — | 1 個 1〜25 秒 |
| coverage の回(config 全体、org と cli は `-run` で絞った) | — | ok | 0 | 0 | 20 秒、46 秒、164 秒 |

起動の仕方: `run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。打つ前の load は 1 分 6.80・5 分 7.65 で、lead の条件(30 未満)を満たしていた。開始のときは、別の worktree(leader-sync-docs)の `run-test.sh` も動いていた。shell の SKIP の行は前の回の報告と同じ 1 行だけだった。

30 分の時間切れはなく、既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` もこの回は通った。打ち直しはしていない。Go の追加の回(`-count=3`、変異、coverage)は `run-test.sh` が終わってから前景で打ち、どれも load 7〜12 だった。

### この diff に関わる shell のテスト

`scripts/verify.local.sh` の test mode は `tests/test-*.sh` を一覧ではなく glob で全部打つので、lead が挙げた 5 本はどれも `run-test.sh` の範囲に入り、この回で通った。

| テスト | 結果 | この diff との関わり |
| --- | --- | --- |
| `tests/test-ralph-config.sh` | 370/370 | helper `ralph_setting` を偽の ralph で、sh・bash・dash・zsh(`-f`)の中で打つ(S2) |
| `tests/test-codex-exec-invocation.sh` | 160/160 | skill の codex の呼び出しの形と `ralph_setting` の使い方、`${RALPH_…:-…}` がないこと(S3) |
| `tests/test-insights-append.sh` | 52 passed, 0 failed | `cap` のある `cycle-count.json` で `--cycle auto` が `.cycle` を読む行 8a2(S3) |
| `tests/test-agent-models.sh` | 47/47 | model-routing.md の tier の表と pin の文(S6 で文書を直した) |
| `tests/test-self-review-scope.sh` | 206/206 | skill と agent の文の範囲(S6) |

## Coverage

- Statement: `internal/config` はパッケージ全体で 97.0%。変更した関数は次のとおり(org と cli は絞った回の profile)。
  - config: `ValidateReviewer` 100%、`checkReviewerWord` 100%、`Default` 100%、`validateLoaded` 98.4%
  - org: `featureReviewer` 100%、`featureLeaderTask` 100%、`startFeatureLeaderParams` 100%、`StartFeature` 95.7%
  - cli: `configValue` 100%、`retiredSettingEnv` 100%、`checkRetiredEnv` 100%、`loadMainWorktreeConfig` 100%、`loadMainWorktreeOrgConfig` 100%、`resolveConfig` 100%、`resolveOrgStartReviewer` 77.8%、`runOrgStartPlan` 81.8%
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes:
  - この diff で足したブロックのうち打たれていないのは、`resolveOrgStartReviewer` の 2 つのエラーの return(`internal/cli/org.go:668`、`:675`)と、`runOrgStartPlan` のその呼び出しのエラーの return(`:1185`)だけ。`runOrgStartPlan` は先に `newOrgSpawnRuntimeAt` で同じ設定を読むので、読めない設定はそこで止まり、ここには届かない。S9 の consult の判断と同じだが、届かないことをテストで確かめてはいない
  - shell の helper には計測の道具がない。`tests/test-ralph-config.sh` が打つ分岐は、exit 0、0 で空、127、`unknown command "config"` の 1、そのほかの 1、2、3、表にないキー、撤去した 4 つの変数を設定した場合。`mktemp -d` の失敗の分岐(`scripts/ralph-config.sh:63-66`)は打たない(Test gaps の 2)
  - org と cli のパッケージ全体の数字は、絞った回なので意味がない(org 19.6%、cli 41.7%)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `zsh -f tests/test-ralph-config.sh`(lead の追加の確かめ。`run-test.sh` の範囲の外) | rc 1、13/55。例: `FAIL: /bin/sh /bin/bash /bin/dash /bin/zsh: exit 0 of standard.max_pipeline_cycles reviewer.claude_model … returns 0`(actual 127) | test の runner が、引用符なしの変数の単語分割に頼っている。`for key in $KEYS`(`tests/test-ralph-config.sh:136`、`:156`、`:165`、`:227`)、`$RETIRED_ENV`(`:229`、`:232`)、`for s in $shells`(`:263`)。zsh は `SH_WORD_SPLIT` なしでは分割しないので、5 つのキーの列が 1 つのキーになり、4 つの shell の列が 1 つの shell の名前になる。helper の不具合ではない。helper を zsh で動かす確かめは、sh の回の `=== ralph_setting under /bin/zsh ===` の節(`/bin/zsh -f -c` で helper を source する。PASS 89 行)が通っている。`zsh --emulate sh -f` では 370/370。origin/main の版(19 件、ループなし)は `zsh -f` でも 19/19 だった | ファイルは `#!/usr/bin/env sh` で、`verify.local.sh` も shebang で起こすので、gate は影響を受けない。runner を zsh でも動かしたいなら、先頭に `[ -n "${ZSH_VERSION:-}" ] && setopt SH_WORD_SPLIT` の 1 行を足す。採るかどうかは lead の判断に任せる |

`run-test.sh`、`-count=3` の 2 本、coverage の 3 回には失敗がなかった。

## Test plan とテストの対応

| 計画の項目 | テスト(この diff で増えたもの、またはこの diff で直したもの) |
| --- | --- |
| 5 つのキーの既定値、`ralph.toml` の値、`ralph.local.toml` の上書き | `TestDefault_PipelineCyclesAndReviewer`(`internal/config/config_test.go:1967`)、`TestLoad_PipelineCyclesAndReviewer`(`:1982`)、`TestLoad_LocalFileOverridesPipelineCyclesAndReviewer`(`internal/config/local_test.go`) |
| 検証のエラー(上限 0 と負、claude_effort の外、空、許さない文字) | `TestLoad_StandardMaxPipelineCyclesBoundary`(`config_test.go:2038`)、`TestLoad_ReviewerValidation`(`:2078`、`-count=3` で 3 回とも同じ結果) |
| `ralph config get` の 5 つのキーと exit 3 | `TestConfigGet_PrintsThePipelineCapAndReviewerValues`、`TestConfigGet_InvalidPipelineCapOrReviewerValueExits3`(`internal/cli/config_test.go`) |
| doctor の 4 行(設定あり・空・未設定、表の順) | `TestCheckRetiredEnv_StandardFlowVariables`、`TestCheckRetiredEnv_AllSixInTableOrder`、`TestRunDoctorOpts_RetiredStandardFlowEnv`(`internal/cli/doctor_retired_standard_env_test.go`) |
| helper の表と `Default()`、`[org.review]` と `[standard]` の上限、template の注釈、skill の本文 | `internal/config/defaults_sync_test.go`(`TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides`、`TestTemplateDrift_EveryStandardAndReviewerKeyIsChecked` と、既存の照合に足した行) |
| helper の終了コードと env が効かないこと | `tests/test-ralph-config.sh`(370 件) |
| skill の codex の呼び出しの形と helper の使い方 | `tests/test-codex-exec-invocation.sh`(160 件) |
| leader の計画助言が TASK の 2 行の値を使い、claude に `--effort` | `TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryTakesTheReviewerFromTheTask`(`internal/org/prompts_leader_plan_test.go:716`)と、直した `…CodexTurnsMCPOff`(`:655`)、`…ClaudeReadsOnly`(`:687`)、`…CodexCommandRuns`、`…ClaudeCommandRuns` |
| `featureLeaderTask` の 2 行と空の扱い、StartFeature の検証 | `TestFeatureLeaderTask_Reviewer`(`internal/org/feature_test.go:708`)、`TestStartFeature_LeaderTaskCarriesTheReviewer`(`:409`)、`TestStartFeature_RefusesReviewerValuesThatSplitTheShell`(`:449`、`-count=3` で 3 回とも同じ結果) |
| cli の `ralph org start --plan` の経路で ralph.toml と ralph.local.toml の値が TASK に出る、主 worktree から読む | `TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings`(`internal/cli/org_feature_test.go:507`)、`TestOrgStartPlan_ReviewerFromMainWorktree`(`:567`)、`TestOrgStartHelp_ReviewerSource`(`:668`) |
| `cap` のある cycle-count.json と `--cycle auto` | `tests/test-insights-append.sh` の 8a2 |
| 回帰 | `run-test.sh` の中で org 1,331 本、cli 926 本、config 100 本と shell の 45 本がすべて通った |

## Regression checks

実装の各スライスで implementer が変異を確かめている(計画の Progress)。この回では、lead が挙げた 3 か所に 2 か所を足し、5 個を独立に確かめ直した。worktree のファイルは変えていない。shell の 2 個は `git archive HEAD` の写しで、Go の 3 個は 1 ファイルを差し替える `go test -overlay` で入れた。前後の `git status --porcelain --ignored`(`.harness/cache/` を除く)は同じだった。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| H1 helper の 127 の分岐を外す(`scripts/ralph-config.sh:81` の `= 127` を `= 126` に) | 落ちた | 330/370。4 つの shell で 10 件ずつ(「no ralph on PATH … writes nothing on stderr」など) |
| H2 古い ralph の判定を「exit 1 なら何でも」に広げる(`:73` の `grep -q 'unknown command "config"'` を外す) | 落ちた | 354/370。4 つの shell で 4 件ずつ(「exit 1 writes one line on stderr」など) |
| G1 doctor の表の 1 行の key を取り違える(`internal/cli/doctor.go:900`、`RALPH_CODEX_REASONING_EFFORT` の行が `[reviewer].codex_model` を名指す) | 落ちた | `TestCheckRetiredEnv_StandardFlowVariables/RALPH_CODEX_REASONING_EFFORT`、`TestRunDoctorOpts_RetiredStandardFlowEnv/set` |
| G2 `featureLeaderTask` の claude の行に codex の effort を書く(`internal/org/feature.go:489`) | 落ちた | org 5 本(サブテスト 7 件)、cli 2 本(サブテスト 6 件)。既定値の行は 2 つの effort がどちらも `xhigh` なので通った。ほかの行が別の値を使うので変異は落ちる |
| G3 codex_effort の検証で codex_model を見る(`internal/config/config.go:1054`) | 落ちた | `TestLoad_ReviewerValidation` のサブテスト 26 件 |

## Test gaps

1. `tests/test-ralph-config.sh` の runner は `zsh -f` では動かない(Failure analysis)。gate には関わらないが、lead が求めた `zsh -f` の回は rc 1 だった。helper 自身の zsh での振る舞いは、sh の回の中の zsh の節で通っている
2. helper の `mktemp -d` の失敗の分岐(`scripts/ralph-config.sh:63-66`)を打つ行がない。テストの偽の `mktemp` は成功して作ったものを記録するだけ。この回で、必ず exit 1 する偽の `mktemp` を PATH の先頭に置き、sh・bash・dash・`zsh -f` で `ralph_setting reviewer.codex_model` を打った。4 つとも rc 0、stdout は `gpt-6-astra`、stderr は `ralph-config.sh: cannot make a temporary directory to run ralph config get reviewer.codex_model; using the default gpt-6-astra` の 1 行だった。同じ形の行を `tests/test-ralph-config.sh` に足せば閉じられる
3. `resolveOrgStartReviewer` のエラーの return と、`runOrgStartPlan` のその呼び出しのエラーの return(`internal/cli/org.go:668`、`:675`、`:1185`)は打たれていない。`runOrgStartPlan` は先に `newOrgSpawnRuntimeAt`(`newOrgRuntimeAt` と `withMainWorktreeOrgLimits`)で同じ設定を読み、読めない設定はそこで止まる。だから届かない、という判断を S9 の consult がしている。テストでは確かめていない
4. 実物の `claude -p --effort` と codex を、helper の読んだ値で起こす通しの確かめはしていない。skill の本文の照合(`defaults_sync_test.go`、`test-codex-exec-invocation.sh`)と、leader の雛形のコマンドを偽の claude・codex で動かすテスト(`…ClaudeCommandRuns`、`…CodexCommandRuns`)で見ている。`--effort` が効くことの確かめは `docs/evidence/claude-effort-probe-2026-10-30.md` にある
5. PATH の ralph(v6.0.0、`config get` なし)で、skill が実際に既定値で回ることは、偽の ralph の exit 1 と `unknown command "config"` の行で見ている。実物の v6.0.0 は打っていない

どれも計画の判定(既定値、検証、helper の終了コードの扱い、TASK の 2 行、doctor の warn)を変えない。それらの判定は、上の変異で落ちることを確かめた。

## Verdict

- Verdict(cycle 1、HEAD 5e400dac): pass
- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本、Go 10 パッケージ)、lead が挙げた shell の 5 本、`test-ralph-config.sh` の sh・bash・`zsh --emulate sh -f`、`-count=3` の 2 本(3 回とも同じ結果)、変異 5 個はすべて落ちた
- Fail: 0
- Blocked: 0
- 補足: `zsh -f tests/test-ralph-config.sh` は rc 1(13/55)。原因は runner の単語分割で、helper の不具合ではない。gate の外の確かめなので verdict は pass とし、runner に 1 行足すかどうかは lead の判断に任せる
