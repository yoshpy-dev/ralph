# Test report: init-config-questions

- Date: 2026-11-02(run-test.sh は 14:29:49 から 15:19:00 JST)
- Plan: `docs/plans/active/2026-11-02-init-config-questions.md`(`## Test plan`、AC1〜AC10)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff origin/main...HEAD`(origin/main c8c8c0f1、HEAD 2fb3284d)。Go は `internal/cli`(`init.go`、新しい `init_settings.go`、新しいテスト 3 本)。`./scripts/run-test.sh` は changed scope のまま打ち、選ばれた pack は golang(`Language scope: changed (changed_languages)`)。test mode なので `tests/test-*.sh` の 45 本もすべて走った
- Evidence:
  - `docs/evidence/test-2026-11-02-init-config-questions.log`(`run-test.sh` の全出力 5,618 行)
  - `docs/evidence/test-2026-11-02-init-config-questions-probes.log`(実物の binary を pty で動かした確かめの expect の台本、各回の出力、書かれた `ralph.toml` と雛形の diff、manifest の行、upgrade の dry-run と doctor の出力、変異 4 つの差分、絞った coverage の回の数字)
  - どちらも `.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない。wrapper 自身のログは `docs/evidence/verify-2026-11-02-052949.log`(同じく対象外。名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0、`All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(下の注) | 2,951 秒。load は開始時 7.31 / 10.23 / 10.51、15:08 に 100.45 / 65.14 / 43.03、終了時 14.92 / 22.14 / 29.83 |
| └ Go: `internal/org` | — | ok | 0 | 0 | 1,654.9 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 1,292.6 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/upgrade`、`internal/org/driver` | — | ok | 0 | 0 | 45.8 秒、4.0 秒、3.6 秒、3.1 秒 |
| └ Go: `internal/scaffold`、`internal/org/protocol`、`internal/insights`、`cmd/ralph` | — | ok | 0 | 0 | 2.7 秒、2.3 秒、2.1 秒、0.7 秒 |
| 新しいテスト 29 個を `go test -count=1 -v -run '^(…)$' ./internal/cli/` で | 29(subtest 90) | 29(subtest 90) | 0 | 0 | 2.8 秒 |
| 絞った coverage の回: `go test -count=1 -coverprofile -run 'Init\|PackQuestion\|CLIQuestion\|SettingsForm\|RunDoctorOpts_InitSettings\|AnsweredFS\|ExecuteInit' ./internal/cli/` | — | ok | 0 | 0 | 14.9 秒 |
| 実物の binary の確かめ(下の「実物の binary」) | 7 回(pty 6 回、`--yes` 1 回) | 7 | 0 | 0 | 1 回 2〜10 秒 |
| 同じ確かめを変異 4 つの binary で(`go build -overlay`) | 4 | 4 つとも確かめが落ちた | — | — | — |
| `ralph upgrade --dry-run` と `ralph doctor` を答えを書いた 3 つの project で | 6 + 対照 1 | 6、対照は期待どおり 1 件 | 0 | 0 | — |

起動の仕方: `run-test.sh` は Bash tool の 600 秒の上限を超えるので、perl の fork と setsid で切り離し(INT・HUP・QUIT・TERM は既定に戻し、標準入出力は `/dev/null`)、log と開始・終了・rc の meta を書かせた。同じ手番の中で wrapper の pid を `caffeinate -w` で待って meta を読んだ。

やり直しはしていない。1 回目が rc 0 で通り、lead の挙げた load で落ちやすいテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`)も落ちなかった。どのパッケージも `(cached)` ではない。

Skipped の 2 行は、どちらもこの diff と関係のない手元の環境による。

- `tests/test-plan-visual.sh`: `SKIP: browser detection (a browser in /Applications would start)`
- `tests/test-detect-changed-languages.sh` の系: `SKIP: the real git older than 2.41 case (git 2.49.0 is 2.41 or later; the stubbed old git covers it)`

時間の注: `internal/org` は 1,654.9 秒で、30 分の時間切れの 92% に届いた。Go の段のあいだ、ほかの 2 つの session の負荷で 1 分の load が 100 を超えた。この回の Go の段では、自分の側の実行は待つだけにした(pty の確かめと coverage の回は shell の段のうちに済ませた)。

## 実物の binary

`go build -o <scratch>/bin-head/ralph ./cmd/ralph` で HEAD の binary を作り、`env -i`(HOME は scratch、`GIT_CONFIG_GLOBAL=/dev/null`)で `ralph init` を expect の pty で動かした。PATH には `claude` か `codex`(または両方)の偽の binary だけを置いた。台本と出力は probes の log にある。

| 回 | 条件 | 答え | 結果 |
| --- | --- | --- | --- |
| t1 | TERM=dumb、PATH に両方 | Both、y n y n | CLI の問いは「Both (Recommended)」。4 問が順に出た。雛形との差は `require_claude_cli`・`require_codex_cli` が true、`[standard]` と `[org.review]` の `plan_advisory` が true(`cross_review` は false のまま)、`regate_checks = "ci"`、`auto_merge = false` の 6 行だけ(AC1)。`ralph config get` で同じ値を読めた |
| t2 | TERM=dumb、PATH に codex だけ | Codex、n y | 「Codex (Recommended)」。問いは CI と merge の 2 つだけ。差は `require_codex_cli = true`、`driver_pool = ["codex"]`、claude の `model_pool` の 4 行が `# ` 付き、`watcher_enabled = false`(AC2) |
| t3 | TERM=dumb、PATH にどちらもない | Claude Code、y y | 「(Recommended)」の付く選択肢なし。差は `require_claude_cli = true`、`driver_pool = ["claude"]`、codex の 4 行が `# ` 付き、`regate_checks = "ci"` |
| t4 | `ralph.toml` が既にあり `--force` なし | — | 案内の 1 行が出て、CLI の問いも yes/no も出ない。`ralph.toml` は `cmp` で変わらず、`base (160 files: 159 created, 0 updated, 1 skipped)`。manifest に `disk_hash` なし(AC6) |
| t5 | `ralph.toml` が既にあり `--force` | Both、y y y y | 聞かれて答えを書いた(`1 updated`、`wrote your answers`)。manifest は `template_hash` が雛形、`disk_hash` が書いた内容のハッシュ(AC6、D6) |
| t6 | `--yes` | — | 問いなし。`ralph.toml` は雛形と `cmp` で同じバイト列、manifest に `disk_hash` なし(AC4) |
| f1 | TERM=xterm-256color(全画面) | Both、y n y n | 書かれた `ralph.toml` は t1 と `cmp` で同じ。2 つ目のフォームは全画面でも動いた。台本の最後の `expect eof` が先の eof と重なり expect 自体は rc 1 で終わったが、init は最後まで走った(`.git` と `pre-commit`・`pre-merge-commit` の hook、manifest の `disk_hash` がある) |
| t1〜t3 | 直後の `ralph upgrade --dry-run` | — | 3 つとも rc 0、`advisories: 0 files`(AC7)。対照に t1 の写しの manifest で `template_hash` を答えのハッシュにすると `advisories: 1 files` になった |
| t1〜t3 | `ralph doctor`(PATH はその CLI の偽の binary だけ) | — | 3 つとも rc 0、`All checks passed.`、`ralph.toml` の warn の行なし。ないほうの CLI は `warn — … unusable (not required)`。t3 は `Org codex model slugs: info — no codex entries in model_pool` で、`# ` を付けた行を読まない(AC5) |

t1 と t2 と t5 の manifest の `template_hash` は雛形の `sha256:45b78f5a…`、`disk_hash` は書いた内容のハッシュ(`shasum -a 256` の値と同じ。t1 は `sha256:91eaa177…`、t2 は `sha256:7bf17c1c…`)。

## Coverage

- Statement: 絞った回で、変更の行(`git diff -U0 origin/main...HEAD` の `init.go` と `init_settings.go` の追加行のうち profile の block に入る行)は 221 行中 198 行(89.6%)を通った。関数ごとには `init_settings.go` の 15 関数のうち 13 が 100%、`answeredInitSettingsTOML` 85.7%、`initTOMLCommentModelPool` 95.7%。`init.go` は `executeInit` 86.9%、`runInitInteractive` 26.3%
- Branch: Go の標準の道具は branch coverage を出さない
- Function: 上の数字は `go tool cover -func` を worktree の中で回したもの
- Notes: 通らなかった 23 行は次の 4 か所。
  - `init.go:165-189`(`runInitInteractive` の TERM の確かめより後ろ、CLI の問いの group、2 つ目のフォーム、`cfg.Settings` の受け渡し): 端末がないと届かない行で、Go のテストからは通らない(呼ぶテストは既存の project で先に返る `TestRunInitInteractive_ExistingProjectSkipsForm` と、pipe で止まる `TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm` の 2 つだけ)。上の pty の確かめで実物の binary を通した
  - `init.go:376-377`(答えを計算したあとの `MkdirAll` の失敗)、`init.go:458-459`(`SetFileOwned` の失敗): どちらもエラーを返すだけの行
  - `init_settings.go:64-65`(埋め込みの雛形に `ralph.toml` がないときの `fs.ReadFile` の失敗): 埋め込みの FS では起きない
  - `init_settings.go:248-249`(`[org].model_pool = [` の行が 2 つあるときのエラー): 下の Test gaps の 2

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `--yes` の `ralph.toml` は雛形と同じバイト列(AC4) | pass | `TestExecuteInit_SettingsNilWritesTheTemplateRalphTOML`、実物の binary の t6 |
| TERM=dumb で stdin が端末でないとき、フォームの前で止まる | pass | `TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm`(`internal/cli` ok) |
| 既存の project ではフォームを出さずに upgrade の案内へ | pass | `TestRunInitInteractive_ExistingProjectSkipsForm`(`internal/cli` ok) |
| 言語パックの問い(検出、`--pack`、skip) | pass | `TestNewPackQuestion_*`、`TestInitCmd_Yes*`(`internal/cli` ok)。pty の回でも pack の問いは前と同じ形で出た |
| 雛形と `Default()` の一致 | pass | `TestDefaultsLockStep` ほか(`internal/config` ok、45.8 秒)。雛形は変えていない |

## Test gaps

1. `runInitInteractive` の配線(`init.go:165-189`)を通す Go のテストがない。端末がないと届かない行で、絞った coverage でも 0 回だった。pty の確かめが落ちることを、`go build -overlay` で作った変異 4 つの binary で確かめた: m1 `cfg.Settings = &settings` を消す(答えが書かれず `ralph.toml` が雛形のまま)、m2 `askInitSettings` に常に `force` を渡す(t4 で CLI の問いが出て台本が exit 5)、m3 `newCLIQuestion` に何も見つけない `lookPath` を渡す(t2 で「(Recommended)」が消えた)、m4 CLI の問いを 1 つ目のフォームに入れない(台本が CLI の問いを待って exit 3)。Go のテストはこれらの行を通らないので、4 つとも Go のテストでは見つからない。手で回す pty の確かめしか守りがない状態で、S3 の implementer の pty の記録と今回の確かめがそれにあたる。小さな足し方の案: `runInitInteractive` に 2 つのフォームを走らせる関数を差し替えられる口(テストでは accessible モードで入力と出力を渡す)を作り、Both と 1 つの CLI と `ralph.toml` がある場合の 3 行を `executeInit` まで通す。ブロックはしない
2. `[org].model_pool = [` の行が 2 つある雛形のエラー(`init_settings.go:248-249`)を当てるテストがない。`initTOMLSet` の同じ形のエラーは `TestInitSettings_DuplicateOrCommentedKeyIsAnError` が当てている。そういう雛形は TOML の key の重複で `config.Load` も通らないので、影響は小さい。足すなら `TestInitSettings_ModelPoolProblemsAreErrorsForOneCLI` の表に「model_pool が 2 つ」の 1 行
3. `init.go:376-377` と `:458-459` のエラーを返すだけの行は通らない。前からある同じ形の行も通っていないので、今回の変更で増えた穴ではない
4. accessible モード(TERM=dumb)では huh が description を出さないので、「(Recommended): found on this machine's PATH.」の行と各問いの説明は TERM=dumb の利用者には見えない(S3 の記録どおり huh の制約)。全画面の表示は `TestCLIQuestion_ExplainsRecommendedOnlyWhenAnOptionCarriesIt` などが `View()` で当てている
5. `internal/org` が 30 分の時間切れの 92% まで伸びた。今回の diff は `internal/org` に触れていないので、負荷による。load が高いまま次の run-test.sh を回すと時間切れになりうる

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 45 本、Go 10 パッケージ)、新しいテスト 29 個(subtest 90)、実物の binary の確かめ 7 回と upgrade・doctor の 6 回。AC1、AC2、AC4〜AC7 は実物の binary でも確かめた
- Fail: 0
- Blocked: 0
