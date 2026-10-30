# Test evidence: standard-flow-config(cycle 1)

- HEAD: 5e400dac(merge-base 0351800d = origin/main)
- 全出力: `docs/evidence/test-2026-10-30-standard-flow-config.log`(gitignore、14,862 行)。`run-test.sh` の全出力、下の追加の回の出力、変異の差分、coverage の関数の行を順に入れた。wrapper 自身の写しは `docs/evidence/verify-2026-10-30-120355.log`(gitignore)

## run-test.sh

- 起動: Bash tool の `run_in_background` は使っていない。perl の fork と setsid で切り離し、INT・HUP・QUIT・TERM を既定に戻し、標準入出力を /dev/null にした。runner が log と meta を書き、前景の `caffeinate -w <pid>` で待った
- meta:

```
start=2026-10-30T21:03:55+0900
start_load={ 6.80 7.65 8.31 }
head=5e400dac1443da4f811ba748a771a59b7298dd1f
end=2026-10-30T21:30:22+0900
end_load={ 6.90 8.68 8.74 }
rc=0
```

- 途中の load の見本: 9.57(21:08)、7.65(21:18)、13.55(21:28)。開始のときは、別の worktree(leader-sync-docs)の `run-test.sh` も動いていた
- Go(`go test ./...`、`(cached)` はない):

```
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	0.765s
ok  	github.com/yoshpy-dev/ralph/internal/cli	569.003s
ok  	github.com/yoshpy-dev/ralph/internal/config	53.981s
ok  	github.com/yoshpy-dev/ralph/internal/gitroot	6.562s
ok  	github.com/yoshpy-dev/ralph/internal/insights	2.702s
ok  	github.com/yoshpy-dev/ralph/internal/org	907.724s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	5.105s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	3.032s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.514s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	4.524s
==> All verifiers passed.
```

- shell: `tests/test-*.sh` の 45 本すべてに `    OK` の行がある(45 行)。SKIP は `test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の 1 行だけ
- この diff に関わる 5 本の結果(`run-test.sh` の出力から):

| テスト | 結果の行 |
| --- | --- |
| `tests/test-agent-models.sh` | `PASS: 47 / 47`、`FAIL: 0` |
| `tests/test-codex-exec-invocation.sh` | `Results: 160/160 passed` |
| `tests/test-insights-append.sh` | `insights-append tests: 52 passed, 0 failed`(`PASS: 8a2. cycle-count.json with cap 3 → its cycle (2), not the cap` を含む) |
| `tests/test-ralph-config.sh` | `Results: 370/370 passed` |
| `tests/test-self-review-scope.sh` | `PASS: 206 / 206`、`FAIL: 0` |

## 追加 1: test-ralph-config.sh を 3 つの shell で

| コマンド | rc | 結果 | 時間 |
| --- | --- | --- | --- |
| `sh tests/test-ralph-config.sh` | 0 | 370/370 | 21 秒 |
| `bash tests/test-ralph-config.sh`(3.2.57) | 0 | 370/370 | 18 秒 |
| `zsh -f tests/test-ralph-config.sh`(5.9) | 1 | 13/55、42 FAILED | 1 秒 |
| `zsh --emulate sh -f tests/test-ralph-config.sh` | 0 | 370/370 | 26 秒 |
| origin/main の版(`git archive`)を `zsh -f` で | 0 | 19/19 | — |

`zsh -f` の失敗の行の例:

```
=== ralph_setting under /bin/sh /bin/bash /bin/dash /bin/zsh ===
  FAIL: /bin/sh /bin/bash /bin/dash /bin/zsh: exit 0 of standard.max_pipeline_cycles reviewer.claude_model reviewer.claude_effort reviewer.codex_model reviewer.codex_effort returns 0
    expected: 0
    actual:   127
```

原因は test の runner の側にある。`for key in $KEYS`(136・156・165・227 行)、`$RETIRED_ENV`(229・232 行)、`for s in $shells`(263 行)が、引用符なしの変数の単語分割に頼っている。zsh は `SH_WORD_SPLIT` なしでは分割しないので、5 つのキーの列が 1 つのキーに、shell の列が 1 つの shell の名前になる。helper(`scripts/ralph-config.sh`)の失敗ではない。helper を zsh で動かす確かめは、sh の回の中の `=== ralph_setting under /bin/zsh ===` の節(`/bin/zsh -f -c` で helper を source する、PASS 89 行)が通っている。

### helper の `mktemp -d` の失敗(テストにない分岐の ad hoc の確かめ)

必ず exit 1 する偽の `mktemp` を PATH の先頭に置いた(`PATH=<偽の mktemp の dir>:/usr/bin:/bin`、ralph は PATH にない)。

```
sh rc=0 out=gpt-6-astra err=ralph-config.sh: cannot make a temporary directory to run ralph config get reviewer.codex_model; using the default gpt-6-astra
bash rc=0 out=gpt-6-astra err=ralph-config.sh: cannot make a temporary directory to run ralph config get reviewer.codex_model; using the default gpt-6-astra
dash rc=0 out=gpt-6-astra err=ralph-config.sh: cannot make a temporary directory to run ralph config get reviewer.codex_model; using the default gpt-6-astra
zsh -f rc=0 out=gpt-6-astra err=ralph-config.sh: cannot make a temporary directory to run ralph config get reviewer.codex_model; using the default gpt-6-astra
```

## 追加 2: -count=3

| コマンド | rc | 結果 | 時間 | load |
| --- | --- | --- | --- | --- |
| `go test ./internal/config/ -run '^TestLoad_ReviewerValidation$' -count=3 -v` | 0 | 116 の名前(トップレベル 1、サブテスト 115)がどれも 3 回 PASS。`--- PASS` 348 行、`--- FAIL` 0 行 | 2 秒(テスト 0.688 秒) | 7.09 |
| `go test ./internal/org/ -run '^TestStartFeature_RefusesReviewerValuesThatSplitTheShell$' -count=3 -v` | 0 | 6 の名前(トップレベル 1、サブテスト 5)がどれも 3 回 PASS。`--- PASS` 18 行、`--- FAIL` 0 行 | 1 秒(テスト 0.555 秒) | 7.14 |

## 追加 3: 変異 5 個

shell の 2 個は `git archive HEAD scripts tests/test-ralph-config.sh` の写しの helper を変え、写しの中で `sh tests/test-ralph-config.sh` を打った。Go の 3 個は `go test -overlay` で 1 ファイルを差し替えた。どれも 1 か所だけの置き換えで、置き換える文字列が 1 回だけ現れることを確かめてから変えた。worktree の `git status --porcelain --ignored`(`.harness/cache/` を除く)は前後で同じだった。

| 変異 | 差分 | コマンド | 結果 |
| --- | --- | --- | --- |
| H1 helper の 127 の分岐 | `scripts/ralph-config.sh:81` の `[ "$_rs_rc" = 127 ]` を `[ "$_rs_rc" = 126 ]` に | `sh tests/test-ralph-config.sh` | 落ちた。rc 1、330/370。4 つの shell で 10 件ずつ、「no ralph on PATH … writes nothing on stderr」など |
| H2 古い ralph の判定 | `:73` の `[ "$_rs_rc" = 1 ] && grep -q 'unknown command "config"' …` を `[ "$_rs_rc" = 1 ]` に | 同上 | 落ちた。rc 1、354/370。4 つの shell で 4 件ずつ、「exit 1 writes one line on stderr」「exit 1's line names the key and the exit code」など |
| G1 doctor の表の 1 行 | `internal/cli/doctor.go:900` の `RALPH_CODEX_REASONING_EFFORT` の行の key を `[reviewer].codex_model` に | `go test -overlay … ./internal/cli/ -run 'RetiredEnv\|RetiredStandard' -count=1` | 落ちた(6 秒)。`TestCheckRetiredEnv_StandardFlowVariables/RALPH_CODEX_REASONING_EFFORT`、`TestRunDoctorOpts_RetiredStandardFlowEnv/set` |
| G2 `featureLeaderTask` の reviewer の行 | `internal/org/feature.go:489` の claude の行の effort を `r.ClaudeEffort` から `r.CodexEffort` に | org: `-run 'TestStartFeature_LeaderTaskCarriesTheReviewer\|TestFeatureLeaderTask_Reviewer\|PlanAdvisoryTakesTheReviewerFromTheTask\|PlanAdvisoryClaudeCommandRuns'`、cli: `-run 'TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings\|TestOrgStartPlan_ReviewerFromMainWorktree'` | 落ちた(org 12 秒、cli 25 秒)。org は `TestStartFeature_LeaderTaskCarriesTheReviewer/a_reviewer_given`、`TestFeatureLeaderTask_Reviewer` の 3 行(`every_value_given`、`the_codex_model_and_the_claude_effort_given`、`full_model_IDs`)、`…PlanAdvisoryTakesTheReviewerFromTheTask`、`…PlanAdvisoryClaudeCommandRuns/{bash,zsh}/a_clean_review`。cli は `TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings` の 2 行と `TestOrgStartPlan_ReviewerFromMainWorktree` の 4 行 |
| G3 config の検証の取り違え | `internal/config/config.go:1054` の `checkReviewerWord("codex_effort", r.CodexEffort)` を `r.CodexModel` に | `go test -overlay … ./internal/config/ -run 'Reviewer\|PipelineCycles' -count=1` | 落ちた(1 秒)。`TestLoad_ReviewerValidation` のサブテスト 26 件(`codex_effort "gpt 6" rejected`、`"a$b"`、`"\""`、`""` など) |

G2 では、`TestFeatureLeaderTask_Reviewer` の既定値の行は通った。既定の claude と codex の effort はどちらも `xhigh` なので、この変異と区別できない。ほかの行が 2 つの effort に別の値を使っているので、変異は落ちる。

## coverage

| コマンド | 結果 | 時間 |
| --- | --- | --- |
| `go test -count=1 -coverprofile ./internal/config/` | ok、パッケージ全体 97.0% | 20 秒 |
| `go test -count=1 -coverprofile ./internal/org/ -run 'StartFeature\|FeatureLeaderTask\|PlanAdvisory'` | ok(絞った回なのでパッケージの数字 19.6% に意味はない) | 46 秒 |
| `go test -count=1 -coverprofile ./internal/cli/ -run 'Retired\|OrgStart\|ConfigGet\|MainWorktree\|Doctor'` | ok(同 41.7%) | 164 秒 |

変更した関数:

```
internal/config/config.go:659:   Default                    100.0%
internal/config/config.go:863:   validateLoaded              98.4%
internal/config/config.go:1044:  ValidateReviewer           100.0%
internal/config/config.go:1059:  checkReviewerWord          100.0%
internal/org/feature.go:205:     StartFeature                95.7%
internal/org/feature.go:387:     featureReviewer            100.0%
internal/org/feature.go:406:     startFeatureLeaderParams   100.0%
internal/org/feature.go:469:     featureLeaderTask          100.0%
internal/cli/config.go:109:      configValue                100.0%
internal/cli/doctor.go:914:      retiredSettingEnv          100.0%
internal/cli/doctor.go:930:      checkRetiredEnv            100.0%
internal/cli/org.go:324:         loadMainWorktreeOrgConfig  100.0%
internal/cli/org.go:337:         loadMainWorktreeConfig     100.0%
internal/cli/org.go:648:         resolveConfig              100.0%
internal/cli/org.go:665:         resolveOrgStartReviewer     77.8%
internal/cli/org.go:1165:        runOrgStartPlan             81.8%
```

打たれていないブロックのうち、この diff で足したものは `resolveOrgStartReviewer` の 2 つのエラーの return(`org.go:668`、`:675`)と、`runOrgStartPlan` のその呼び出しのエラーの return(`:1185`)。ほかの未到達のブロック(`feature.go:235` の manifest の読み込みの失敗、`config.go:949` など)は、この diff より前からある。
