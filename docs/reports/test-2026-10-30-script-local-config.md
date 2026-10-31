# Test report: script-local-config

- Date: 2026-11-01(JST。UTC では 2026-10-31)
- Plan: `docs/plans/active/2026-10-30-script-local-config.md`(`## Test plan` と AC1〜AC7、Progress の S1〜S7 と self-review の直し 00c494cc)。Refs #276(B2)
- Tester: tester サブエージェント(`/test`、cycle 1。`cycle-count.json` は 1)
- Scope: `git diff origin/main...HEAD`(merge-base 1cc84ffa、HEAD 643a92c3、51 ファイル)。2cb62e4b は origin/main(#292)の取り込み。Go は `internal/config`、`internal/cli`(doctor、`config get`)、`internal/org`(`gate_run.go` は注釈だけ、gate のテスト)。shell は `scripts/` の 6 本(`ralph-config.sh`、`plan-visual.sh`、`xreview-helpers.sh`、`secret-scan-branch.sh`、`detect-changed-languages.sh`、`run-verify.sh`)と写し、`tests/` の 8 本、共通の偽の ralph `tests/lib/fake-ralph.sh`。Go のテストの関数は 13 本増え、3 本が消えた(`TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides` は `defaults_sync_settings_test.go` に移っただけ、ほかの 2 本は名前と範囲を広げた後継に置き換わった)
- Evidence: `docs/evidence/test-2026-10-30-script-local-config.log`(`run-test.sh` の全出力、`-v` の 3 回、coverage、変異 7 個の差分と結果、`/Applications` を隠した回、`run-verify.sh` の全出力)。`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない(`git check-ignore` で確かめた)。wrapper 自身の写しは `docs/evidence/verify-2026-10-31-161811.log`(run-test)と `docs/evidence/verify-2026-10-31-165607.log`(run-verify)で、どちらも同じく対象外

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed の既定。「Language scope: full fallback (shared:scripts/detect-changed-languages.sh)」、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 10 パッケージ ok(`(cached)` なし) | 0 | 2 行(下の表の plan-visual と secret-scan-branch) | 1,801 秒(01:18:10〜01:48:11 JST)。load は開始 6.56 / 7.96、終了 9.97 / 9.98 |
| └ Go: `internal/org` | 1,361 本(トップレベル、`go test -list`) | ok | 0 | 0 | 960.9 秒 |
| └ Go: `internal/cli` | 932 本(同) | ok | 0 | 0 | 558.6 秒 |
| └ Go: `internal/config` | 104 本(同) | ok | 0 | 0 | 11.1 秒 |
| └ Go: `cmd/ralph`、`internal/{gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 1.0〜6.1 秒 |
| `go test ./internal/config/ -count=1 -v` | 104 本、`--- PASS` 611 行 | 611 | 0 | 0 | 31 秒 |
| `go test ./internal/cli/ -run 'Retired\|Doctor\|ConfigGet\|Config' -count=1 -v` | 109 本、`--- PASS` 250 行 | 250 | 0 | 0 | 79 秒 |
| `go test ./internal/org/ -run 'Gate' -count=1 -v` | 66 本、`--- PASS` 243 行 | 243 | 0 | 0 | 282 秒 |
| `./scripts/run-verify.sh`(AC7。既定の all mode、full scope、rc 0) | 静的の検査 + shell 45 本 + Go 10 パッケージ + branch の secret scan | すべて通った(下の AC7 の節) | 0 | 2 行(run-test と同じ) | 1,711 秒(01:56:07〜02:24:38 JST)。cli 597.0 秒、org 845.8 秒、config 17.9 秒、ほかの 7 パッケージは `(cached)` |
| 変異 7 個(`git archive HEAD` の写しで 1 個ずつ) | 7 | 7 個とも落ちた | — | — | 1 本 23〜159 秒 |
| `tests/test-plan-visual.sh`(`/Applications` を隠した写し、Test gaps の 1) | 283 | 283 | 0 | 0 | 23 秒 |
| coverage(config は全体、cli は `-run` で絞った) | — | ok | 0 | 0 | 27 秒、69 秒 |
| `go build ./...` | — | rc 0 | — | — | — |

起動の仕方: `run-test.sh` と `run-verify.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。打つ前の load は 1 分 6.56・5 分 7.96(run-test)、10.05・10.28(run-verify)で、lead の条件(30 未満)を満たしていた。途中の見本も 5.65〜12.10 だった。`-v` の 3 回は 2 つの run の間に前景で打ち、変異と coverage は `run-verify.sh` の shell の段と並べて打った。

30 分の時間切れはなく、既知の flaky な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` も 2 回とも通った。打ち直しはしていない。前後の `git status --porcelain` は、計画の AC7 の 1 行と、この report を除いて空だった。

### 計画のテストの要点(shell)

数は `run-test.sh` の出力から取った。`run-verify.sh` の回も同じ数だった。実装の Progress と self-review の直しの記録の数とも一致する。

| テスト | 結果 | 見ているもの |
| --- | --- | --- |
| `tests/test-ralph-config.sh` | 272/272 | helper の表の 10 キー、`ralph_setting` と `ralph_setting_or_fail` の終了コード(exit 0、0 で空、127、古い ralph、1、2、3、mktemp の失敗)、既定値が空のキーの exit 0 の空 |
| `tests/test-plan-visual.sh` | 277 passed、0 failed。SKIP 1 行(ブラウザの自動検出の 2 件、6 行。Test gaps の 1) | `[plan_visual].opener`・`browser` の値・空・`none`、ralph なし、古い ralph、`ralph.local.toml` の誤り、env が効かないこと(8 行)、`lint`・`digest` で ralph を打たないこと |
| `tests/test-xreview-helpers.sh` | 53/0 | `detect_base_branch` の `[standard].base_branch`、exit 2・3 で 0 以外を返し stdout が空(3h)、ralph なしと古い ralph で自動の段、env が効かないこと |
| `tests/test-secret-scan-branch.sh` | 263/0。SKIP 1 行(git 2.41 より古い実物の git。この diff に関わらない) | `GITHUB_BASE_REF` があると ralph を打たない(偽の ralph の呼び出し記録が空)、設定の誤りで `--strict` が exit 3、古い ralph で自動の base、`RALPH_XREVIEW_BASE` が効かないこと |
| `tests/test-detect-changed-languages.sh` | 113/113 | `[verify].base` の値・空、exit 2・3 で `config_error:verify.base` の full、`no_changes` を出さない、ralph なし、`ralph-config.sh` がない repo、`RALPH_VERIFY_BASE` が効かないこと |
| `tests/test-run-verify-branch-secret-scan.sh` | 50/0 | skip が `true` のときだけ飛ばす。`false`・127・古い ralph・exit 3・`ralph-config.sh` なしでは scan が走る。`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` が効かないこと |
| `tests/test-run-verify-concurrency.sh` | 37/37 | 撤去した変数の unset の行を直した分(S5) |
| `tests/test-run-verify-scope.sh` | 25/25 | D7 の回帰: main の上で、空なら `no_changes`(対照)、exit 3 なら full になり、わざと失敗させた golang と python の検証が走って run が落ちる |

### 計画のテストの要点(Go)

| パッケージ | テスト(`-v` の回で PASS を確かめた) |
| --- | --- |
| `internal/config` | `TestLocalOnlyKeys_HoldsTheScriptKeys`、`TestLoad_ScriptLocalOnlyKeysInRalphToml`(サブテスト 50。5 キーと大文字小文字の変化)、`TestLoad_LocalScriptKeys`(3)、`TestLoad_LocalScriptValueValidation`(59。D3 の通る例と断る例)、`TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides`(10)、`TestTemplateDrift_EveryStandardReviewerPlanVisualAndVerifyKeyIsChecked`、`TestDefaultsLockStep/shell_and_skills`(helper の表と `Default()`) |
| `internal/cli` | `TestConfigGet_PrintsTheScriptLocalOnlyValues`、`TestConfigGet_InvalidScriptLocalOnlyValueExits3`(7)、`TestConfigGet_RalphTomlThatSetsAScriptLocalOnlyKeyExits3`(10)、`TestCheckRetiredEnv_ScriptVariables`(5)、`TestCheckRetiredEnv_ScanSkipOtherThanOne`(3)、`TestCheckRetiredEnv_AllElevenInTableOrder`、`TestRunDoctorOpts_RetiredScriptEnv`(3。設定あり・空・未設定) |
| `internal/org` | `TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier`。対照の run は個人用の値(base `HEAD`、skip `true`)で Go の検証と scan を飛ばす。gate の run は ralph に `verify.skip_branch_secret_scan` を `RALPH_LOCAL_CONFIG=off` で 1 回だけ聞き、scan を走らせる。full scope なので `verify.base` は聞かない |

### AC7: `./scripts/run-verify.sh`

既定(`HARNESS_VERIFY_MODE=all`、`RALPH_VERIFY_SCOPE=full`)で rc 0、最後の行は「==> All verifiers passed.」だった。中身は次のとおり。

- local verifier: shellcheck(hook と verify のスクリプト、`tests/lib/*.sh` を含む)、`sh -n`、`jq -e .` の 2 本、check-sync(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 6、「PASS: all files in sync.」)、check-pipeline-sync、check-skill-sync(「14 skill(s) in lock-step」)、check-template-purity。どれも `OK`。
- shell のテスト: 45 本 OK。`OK` の行は local verifier の分を含めて 78。
- golang verifier: gofmt `ok`、golangci-lint `0 issues.`、go vet と staticcheck は無出力(Skipping の行なし)、go test は 10 パッケージ ok。
- branch の secret scan: 「secret-scan-branch: scanned 1cc84ffa..643a92c3 against origin/main: clean」。

AC7 の残り(`go build ./...`)は、この回でも rc 0 だった。計画の AC7 を `[x]` にした。本文は変えていない。`./scripts/plan-visual.sh digest` は直したあとも `cdd1e5ba1234` のまま。

この repo の PATH の ralph は 6.0.0 で、`config get` がない(`unknown command "config" for "ralph"`、rc 1)。2 つの run では、4 つのスクリプトが実物の古い ralph の道を通った。helper の行は stderr に 1 行も出ず、scan は自動の base `origin/main` で走った。AC4 の「古い ralph では今の既定の振る舞いのまま」を実物で見たことになる。

## Coverage

- Statement: `internal/config` はパッケージ全体で 97.2%。変更した関数は次のとおり(cli は絞った回の profile)。
  - config: `validateScriptValues` 100%、`isScriptWord` 100%、`Default` 100%、`LocalOnlyKeys` 100%、`validateLoaded` 98.5%
  - cli: `retiredScriptEnv` 100%、`checkRetiredEnv` 100%、`retiredSettingEnv` 100%、`newConfigGetCmd` 100%、`configValue` 100%
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes:
  - `internal/config/config.go` と `local.go` で打たれていないブロック(`config.go:843`、`:880`、`:899`、`:913`、`:1024`、`local.go:103`、`:144`)は、どれもこの diff の hunk の外にある。この diff で足した文はすべて打たれている
  - `internal/org/gate_run.go` は注釈だけの変更なので測っていない
  - shell には計測の道具がない。分岐は上の表の行と、下の変異で見た
  - cli のパッケージ全体の 41.2% は、絞った回なので意味がない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

`run-test.sh`、`run-verify.sh`、`-v` の 3 回、coverage の 2 回、`/Applications` を隠した回に失敗はなかった。

## Regression checks

実装の各スライスで implementer が変異を確かめている(計画の Progress)。この回では、計画の Test plan が挙げた「helper の空の扱い」「スクリプトが env を読む形に戻す」と、D7 の 2 か所、CI の `GITHUB_BASE_REF` を、7 個の変異で独立に確かめ直した。worktree のファイルは変えていない。どれも `git archive HEAD` の写しに 1 か所だけ入れ(1 回だけ現れる文字列を置き換える perl)、その写しで関係するテストを打った。赤の行は、変異が届く行かを文面で確かめた。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| M1 D7: `verify.base` の読み出しの失敗で、full ではなく自動の base に進む(`scripts/detect-changed-languages.sh` の `fallback_full "config_error:verify.base"` を `base_ref=""` に) | 落ちた | detect 105/113(exit 3 と exit 2 で「falls back to full」「does not report no_changes」など 8 行)、run-verify-scope 21/25(「settings error: the run fails (exit 0)」など 4 行) |
| M2 run-verify が `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` でも飛ばす形に戻す | 落ちた | 47/50(「RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1: runs the scan」など 3 行) |
| M3 D7: `detect_base_branch` が読み出しの失敗で `return 1` せず自動の段に進む | 落ちた | xreview-helpers 49/53(3h の exit 3・exit 2 で rc 0、stdout が `main`)、secret-scan-branch 258/263(「settings error: --strict exits 3 (exit 0, want 3)」など 5 行) |
| M4 `GITHUB_BASE_REF` があるときも `detect_base_branch` を打つ | 落ちた | 261/263。「GITHUB_BASE_REF: ralph is not run」が落ちた。もう 1 行(「settings error: ralph is asked once」)は、前の行の呼び出し記録が残ったことによる連鎖 |
| M5 helper: exit 0 の空を、既定値があるキーでも値として扱う | 落ちた | 256/272(4 つの shell で「exit 0 with no value, for a key with a default, prints what a failure gives」など) |
| M6 helper: exit 0 の空を、既定値が空のキーでも失敗として扱う | 落ちた | ralph-config 252/272、detect 109/113(「empty [verify].base diffs against origin/main」と、対照の「clean main reports no_changes」が `config_error:verify.base` になった) |
| M7 plan-visual が `RALPH_PLAN_VISUAL_OPENER` を先に読む形に戻す | 落ちた | 272/277(「RALPH_PLAN_VISUAL_OPENER does not override [plan_visual].opener none」など 5 行) |

## Test gaps

1. `tests/test-plan-visual.sh` のブラウザの自動検出の 2 件(6 行)は、この Mac では SKIP になる。`/Applications` に Chrome があり、検出が PATH より先にそこを見て、本物のブラウザを起動してしまうため。代わりに、`scripts/plan-visual.sh` の 2 か所とテストの guard の `"/Applications/` を `"/nonexistent-Applications/` に替えた写しで打ち、283/283(277 + 6)だった。差分は evidence にある。手を加えていない形は Linux の CI で走るので、PR の CI のログで SKIP でなく通ったかを見る。検出の `/Applications` の枝そのものを打つテストは、どの環境にもない。この diff は検出の順を変えていない
2. shell のテストはどれも偽の ralph(`tests/lib/fake-ralph.sh`)で値を渡す。HEAD から build した本物の binary をスクリプトの前に置く確かめは /verify の probe(report の 2 節、8 節、d7.sh)で、この回では打っていない。実物の古い ralph(6.0.0)の道は、上の AC7 の節のとおり 2 つの run で通った
3. shell には coverage の計測がない。分岐の確かめは、テストの行と変異 7 個による
4. 既知の 3 件(self-review の L6、verify の I2・I3)はテストの不足ではないので、ここでは数えない。I2 の「設定の誤りがあると既定のモードの scan が cannot scan で飛び、run-verify は通る」は、`test-secret-scan-branch.sh` の「settings error: default mode exits 0」「default mode reports cannot scan」の行が今の振る舞いとして固定している
5. `test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の SKIP は前からあるもので、この diff に関わらない

どれも計画の判定(5 つのキー、local-only、D3 の検証、helper の空と失敗の扱い、D7、env が効かないこと、CI と gate、doctor の warn)を変えない。それらの判定は、上の表のテストが通り、変異で落ちることを確かめた。

## Verdict

- Verdict(cycle 1、HEAD 643a92c3): pass
- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本、Go 10 パッケージ)、lead が挙げた shell の 8 本と Go の 3 パッケージの絞った回、AC7 の `./scripts/run-verify.sh`(rc 0)と `go build ./...`、変異 7 個はすべて落ちた
- Fail: 0
- Blocked: 0
- 補足: ブラウザの自動検出の 2 件は、この Mac では SKIP。`/Applications` を隠した写しでは通った。PR の CI のログで Linux の結果を見る
