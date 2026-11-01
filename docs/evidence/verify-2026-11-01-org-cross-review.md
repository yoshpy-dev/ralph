# Verify evidence: org-cross-review(issue #269、cycle 1)

- 日付: 2026-11-01(UTC 00:15〜00:28)
- HEAD: d8e02c1d、base: origin/main 4fef6b8c
- run-static-verify 自身のログ: `docs/evidence/verify-2026-11-01-001525.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。この file はその要約と、手で打った確かめの結果
- 手元のパスは `<scratchpad>`(この session の scratchpad)と `<worktree>` に伏せた。token や secret の値は書いていない

## run-static-verify(changed scope)

`./scripts/run-static-verify.sh` は rc=0、末尾は `==> All verifiers passed.`。

- `Mode: static`、`Requested scope: changed`。`scripts/xreview-helpers.sh` が分類されないので `Language scope: full fallback (unclassified:scripts/xreview-helpers.sh)` になり、golang の pack が走った
- shellcheck と `sh -n`(hook 10 本と templates/base の写し 10 本)、guard の awk の構文 2 つ、`jq -e .`(settings.json 2 つ)、Codex hook の検査 3 つ: すべて OK
- `scripts/check-sync.sh`: IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6、`PASS: all files in sync.`
- `scripts/check-pipeline-sync.sh`: 6 か所すべて `[ok]`
- `scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`
- `scripts/check-template-purity.sh`: `PASS: no meta-repo-specific references found in templates.`
- tech-debt README plan references: OK
- golang: `gofmt: ok`、golangci-lint `0 issues.`
- branch secret scan: `scanned 4fef6b8c..d8e02c1d against origin/main: clean`

## 個別に打った検査

| コマンド | rc | 出力の要点 |
| --- | --- | --- |
| `./scripts/check-template-purity.sh` | 0 | PASS |
| `./scripts/check-skill-sync.sh` | 0 | 14 skill が一致 |
| `./scripts/check-sync.sh` | 0 | DRIFTED 0、ROOT_ONLY 0 |
| `sh tests/test-xreview-helpers.sh` | 0 | PASS 53、FAIL 0(1f「explicit arg wins over env」を含む) |
| `git diff --check origin/main...HEAD` | 0 | 出力なし |
| `go build ./...` | 0 | 出力なし |
| `go vet ./internal/org/ ./internal/org/driver/ ./internal/cli/ ./internal/config/` | 0 | 出力なし |
| `gofmt -l internal cmd` | 0 | 出力なし |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-01-org-cross-review.md` | 0 | `7789ae616bd0`(計画の `- Approved:` と同じ) |

`sh tests/test-xreview-helpers.sh` は振る舞いのテストだが、lead の handoff が静的の検査の一覧に名指したので打った。`go test` と `./scripts/run-test.sh` は打っていない。

## 雛形の render

`go run -overlay` で scratch の `main.go`(`org.RenderRolePrompt` を `OrgID: "org-a"` で呼ぶ)を `cmd/zzrender/` に重ね、leader と reviewer の雛形を render した。rc=0、worktree の `git status --porcelain` は空のまま。

- render した leader と reviewer に `{{` は残っていない(0 件)
- 下の probe で使った 4 つのブロック(leader の状態のコマンド、reviewer の手順 2 の 1 行、codex と claude のコマンド)は、render した文にそのまま(reviewer は 3 字下げで)含まれる

## 状態のコマンドと reviewer のコマンドを手で打った結果

scratch の git repo(bare の origin と clone、`origin/HEAD` は main)に、stub の `ralph`・`codex`・`claude` を PATH の先頭に置いて打った。ブロックは `leader.md:797-810`、`reviewer.md:189`、`:242-250`、`:274-279` から sed で抜き、`{{ORG_ID}}` を `org-a` にした。`/bin/sh` と `zsh -f` の 2 つで打ち、SHA と scratch のパスのほかは同じ出力になった。

### leader の状態のコマンド(`### cross-review`)

| 場面 | 出た行 |
| --- | --- |
| A on(`true`)、cap 2、report なし | `switch: on`、`cap: 2`、`base: origin/main`、`done: 0`、`k: 1`、`last: none` |
| B off(rc 0 で `false`) | `switch: off (off: [org.review].cross_review)`、`cap: 2` |
| C PATH に ralph がない(127) | `switch: off (ralph not found; treated as off)`、`cap: 2` |
| D 古い ralph(rc 1、stderr に `unknown command "config"`) | `switch: off (ralph without config get; treated as off)`、`cap: 2` |
| E rc 2 / rc 1 でほかのエラー | `switch: error (exit 2)`・`cap: error (exit 2)` / `switch: error (exit 1)`・`cap: error (exit 1)` |
| F rc 0 で `maybe`、cap `0` | `switch: error (exit 0)`、`cap: error (exit 0)` |
| G ブランチが complete の c1(AR=1)を足し、そのあとは report のコミットだけ | `done: 1`、`k: 2`、`last action_required: 1`、`since: same` |
| I cap のファイル `3` / `03` / `1` / `5 extra` | `cap: 3` / `cap: 2` / `cap: 2` / `cap: 2` |
| H c1 のあとにコードのコミット | `since: changed` |
| K skipped の c2 を足す | `done: 1`、`k: 3`、`last:` は c1 のまま |
| AC5b-1 complete の c3(AR=0)で done 2 = cap 2、変更なし | `done: 2`、`cap: 2`、`last action_required: 0`、`since: same` |
| AC5b-2 そのあと gate の直しのコミット | `since: changed`(TASK を送らず 2 つ目の QUESTION に進む場面) |
| L コミットしていない c4 | `done: 2`、`k: 4`(どちらにも入らない) |
| M report の `- HEAD:` が存在しない object | `since: changed` |
| J main に前の org の c9(complete)、rebase 後 | `done: 3`(変わらない)、`k: 10` |
| J2 org_id が `bar-org-a` の report を足す | `done: 3`、`k: 10`(数えない) |
| N origin の ref がない | `base: error (no origin/HEAD, origin/main, or origin/master)` |

### reviewer の手順 2(相手の選び方)

| 場面 | 出た行 |
| --- | --- |
| R1 driver `claude`、pane の `RALPH_PRIMARY_CLI=claude` | `reviewer=codex`、`available rc=0` |
| R2 driver `codex`、pane の `RALPH_PRIMARY_CLI=claude` | `reviewer=claude`、`available rc=0` |
| R2b driver `codex`、pane の `RALPH_PRIMARY_CLI=codex` | `reviewer=claude`、`available rc=0` |
| R3 driver `bogus` | `driver: invalid`、`available rc=1` |
| R4 driver `claude; touch pwned` | `driver: invalid`、`available rc=1`、ファイルは作られない |
| R5 `scripts/xreview-helpers.sh` がない | `helpers: missing`、`available rc=1` |
| R6 driver `codex`、PATH に claude がない | `reviewer=claude`、`available rc=1` |
| R7 driver `claude`、PATH に codex がない | `reviewer=codex`、`codex binary not found`、`available rc=1` |

### reviewer の手順 3(相手の呼び出し)

| 場面 | 出た行 |
| --- | --- |
| X1 codex の stub が `-o` に書く | `codex rc=0`、`out: not empty`、`git: same`。引数は `--disable plugins --disable apps -m gpt-x -c model_reasoning_effort=xhigh -c sandbox_mode=read-only exec review --ignore-rules --base origin/main -o .harness/state/xreview-org-a-c5-out.md`(`-c` はすべて `exec` の前) |
| X2 codex の stub が rc 1、log の最後の行に `$HOME` と cwd | `codex rc=1`、`log: error: could not reach ~/.codex/x in ./y`、`out: empty` |
| X3 codex の stub が追跡中のファイルを書き換える | `git: changed` |
| X4 見張りを `sleep 2` に変えた写しで、30 秒眠る stub | `codex rc=143`、`out: empty` |
| Y1 claude の stub | `claude rc=0`、`out: not empty`、`git: same`。`.diff` は `git diff origin/main...HEAD` と同じ。引数は `-p --model opus --effort xhigh --restricted --strict-mcp-config --disallowedTools mcp__* --tools Read,Grep,Glob --permission-mode dontAsk --output-format text <指示文>` |
| Y2 claude の stub が stdout に何も出さず stderr に 1 行 | `claude rc=0`、`log: API Error: stub`、`out: empty` |
| Y3 base を `origin/nope` に変えた写し | `diff: failed`、`out: empty`、`git: same` |

bash の `/bin/sh` では、見張りを止めたときのジョブの通知(`Terminated: 15`)が出力の行のあいだに入る。判定に使う行は別の行に出るので、読み分けは変わらない。

### report の `- HEAD:` を option として読む場合

`git diff --quiet "--output=<scratchpad>/opt-inject.txt" HEAD -- . ':(exclude)docs/reports/'` は rc=0 を返し、0 バイトのファイルを作った。状態のコマンドの `h` がこの形なら、`since: same` を出し、ファイルを作るか空にする。report の所見 V2 の根拠。

## 意図しない本物の CLI の呼び出し(ずれ)

最初の zsh の実行は `zsh -f` でなく `/bin/zsh` で打ったので、zsh の起動ファイルが PATH の先頭を戻し、stub ではなくこの機械の本物の `ralph`、`codex`(0.163.0)、`claude` が scratch の repo で呼ばれた。

- ralph: `config get org.review.*` が `unknown command "config"` で返り、すべて `switch: off (ralph without config get; treated as off)`。手元に入っている ralph には `config get` がない
- codex: 状態は X1〜X4 の 4 回。どれも今の HEAD の reviewer のコマンドのまま(`--disable plugins --disable apps`、MCP を切った一覧で有効な server が 0、`-c` はすべて `exec` の前)。3 回はモデル `gpt-x` が API に断られて `codex rc=1`、log の最後の行は `ERROR: {"type":"error","status":400,...,"code":"model_not_found"}`。見張りを 2 秒にした X4 は、見張りに止められて `codex rc=0`、`out: empty`(雛形の「codex は見張りに止められても終了コード 0 を返すことがある」と同じ)。送った差分は scratch の repo の数行だけ
- claude: Y1 と Y2 の 2 回。指示文は stub 用の `PROMPT` の 1 語で、claude は「指示がない」と返した(`claude rc=0`、`out: not empty`、`git: same`)。読める道具は Read、Grep、Glob だけで、cwd は scratch の repo
- どちらも scratch の repo の外には書いていない。worktree の `git status --porcelain` は空のまま。上の表の結果は `zsh -f` で打ち直したもの
