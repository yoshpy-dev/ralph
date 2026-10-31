# xreview live probe (2026-11-01)

- 日付: 2026-11-01(A は 05:41:15〜05:46:35、B は 05:47:16〜05:59:02)
- コミット: 586f7934(`internal/org/prompts/reviewer.md` の `## cross-review の TASK` をこの版のまま使った)
- 差分: `git diff origin/main...HEAD`。fetch 後の merge-base は 4fef6b8c で、4 ファイル、1306 行の追加と 6 行の削除(計画、reviewer.md、テスト 2 本)
- 機械: Darwin arm64
- `codex --version`: codex-cli 0.163.0
- `claude --version`: 2.1.308 (Claude Code)
- 実行した場所: この機械の shell(Claude Code の Bash ツール、/bin/zsh 5.9、利用者の profile と alias を読み込んだ状態)。org の座席の中では走らせていない。座席の sandbox、network、pane の環境変数はこの記録では試していない
- worktree: 586f7934 を detach した probe 用の worktree を repo の外に作り、記録のあとに消した

## 埋めた値

- `{{ORG_ID}}` = `probe269`、`<k>` = `1`
- `XREVIEW_BASE` = `origin/main`、`CYCLE` = `1/2`
- `REPORT` = `docs/reports/review-xreview-2026-11-01-probe269-c1.md`(report は書いていない。この記録は手順 3 と 4 まで)
- `REVIEWER_CODEX` = `gpt-6-astra xhigh`、`REVIEWER_CLAUDE` = `fable xhigh`。586f7934 から `go build` した ralph の `ralph config get reviewer.<key>` の値
- 2 方向とも `<k>` = 1 なので `.harness/state/xreview-probe269-c1*` を共有する。A のファイルは B の前に worktree の外(scratchpad の `xreview-evidence/A/`)へ写した。B のファイルは `xreview-evidence/B/` にある

## A: `IMPLEMENTER_DRIVER: claude`(相手は codex)

### 手順 2 で打ったコマンド

```
. scripts/xreview-helpers.sh; r=$(pick_reviewer "claude"); echo "reviewer=$r"; case $r in codex) ./scripts/codex-check.sh ;; claude) command -v claude ;; *) false ;; esac; echo "available rc=$?"
```

### 手順 3 で打ったコマンド

1 回の shell で、`run_in_background` で走らせた。

```
p=".harness/state/xreview-probe269-c1"; mkdir -p .harness/state; rm -f "$p-out.md"
{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; } > "$p-git-before.txt"
command codex --disable plugins mcp list --json | jq -r '.[] | select(.enabled != false) | .name' > "$p-mcp.txt"
set -- --disable plugins; while IFS= read -r name; do [ -z "$name" ] || set -- "$@" -c "mcp_servers.$name.enabled=false"; done < "$p-mcp.txt"
n=$(command codex "$@" mcp list --json | jq '[.[] | select(.enabled != false)] | length')
if [ "$n" = 0 ]; then command codex "$@" -m "gpt-6-astra" -c "model_reasoning_effort=xhigh" -c sandbox_mode=read-only exec review --ignore-rules --base "origin/main" -o "$p-out.md" </dev/null > "$p.log" 2>&1 & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "codex rc=$rc"; else echo "mcp: not off (enabled: $n)"; fi
if test -s "$p-out.md"; then echo 'out: not empty'; else echo 'out: empty'; fi
{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; } > "$p-git-after.txt"
if cmp -s "$p-git-before.txt" "$p-git-after.txt"; then echo 'git: same'; else echo 'git: changed'; fi
```

### 出た行

```
reviewer=codex
codex-cli 0.163.0
available rc=0
codex rc=0
out: not empty
git: same
```

- 所要時間: 5 分 20 秒。20 分の見張りには届いていない
- git の状態: 前後で同じ(`cmp` 一致。どちらも HEAD 586f7934 の 1 行だけで、変更も未追跡のファイルもない)
- MCP: 一覧で有効だった server は 14 個(Figma、Sentry、atlassian、concur、context7、github、kintone、mobile-mcp、node_repl、notebooks、serena-app、serena-arche、serena-dialogos、visualization)。14 個を `-c mcp_servers.<名前>.enabled=false` で切った一覧で 0 になり、codex を呼んだ
- codex の log の頭: `model: gpt-6-astra`、`approval: never`、`sandbox: read-only`、`reasoning effort: xhigh`
- 出力ファイル: `-out.md` は 1103 バイト(5 行)。`.log` は 482187 バイト

### `-out.md` の全 5 行(15 行に満たない)

````
The new Codex invocation does not fully enforce its read-only isolation contract. Formatting and diff checks passed; the targeted Go tests could not run because the read-only sandbox prohibited creating the build directory.

Review comment:

- [P1] Disable connected apps in the Codex reviewer invocation — <scratchpad>/probe-269/internal/org/prompts/reviewer.md:229-231
  When the user has connected Codex apps, this isolation check leaves their tools enabled. With the installed Codex 0.163.0, these arguments produce zero enabled configured MCP servers while `features.apps` remains true. Apps are separate from the listed servers and [operate outside the command sandbox](https://learn.chatgpt.com/docs/config-file/config-reference), so the supposedly read-only reviewer can retain external write capabilities that the Git-state comparison cannot detect. Add `--disable apps` to the shared invocation arguments and cover this independent integration channel in the isolation tests.
````

triage できるか: できる。所見は 1 件で、重大度 `[P1]` と file:line(`internal/org/prompts/reviewer.md:229-231`、worktree の絶対パスで書かれている)が付いている。そのパスは利用者名と session の ID を含むので、上の写しでは scratchpad のディレクトリまでを `<scratchpad>` に伏せた。`codex exec review` は codex 組み込みのレビューの形で返すので、重大度は HIGH/MEDIUM/LOW ではなく P0〜P3 で書かれる。triage の表は重大度の書き方を問わないので、`Reviewer status: complete` の report を書ける。

所見の前提のうち、`--disable plugins` を付けても `apps` の feature が有効なことは `command codex --disable plugins features list` で確かめた(`apps  stable  true`)。apps の道具が `exec review` の中で実際に呼べるかは確かめていない。

## B: `IMPLEMENTER_DRIVER: codex`(相手は claude)

### 手順 2 で打ったコマンド

```
. scripts/xreview-helpers.sh; r=$(pick_reviewer "codex"); echo "reviewer=$r"; case $r in codex) ./scripts/codex-check.sh ;; claude) command -v claude ;; *) false ;; esac; echo "available rc=$?"
```

### 手順 3 で打ったコマンド

1 回の shell で、`run_in_background` で走らせた。`<指示文>` には節の英文を、`{{ORG_ID}}`、`<k>`、`<XREVIEW_BASE>` だけ書き換えて入れた。

```
p=".harness/state/xreview-probe269-c1"; mkdir -p .harness/state; rm -f "$p-out.md"
{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; } > "$p-git-before.txt"
if git diff "origin/main...HEAD" > "$p.diff"; then RALPH_WIP_CHECKPOINT=off command claude -p --model "fable" --effort "xhigh" --restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk --output-format text "You are an adversarial code reviewer. Your job is to break confidence in this change, not to validate it. Default to skepticism — assume the change can fail in subtle, high-cost ways until evidence says otherwise. The change is the diff in the file .harness/state/xreview-probe269-c1.diff, the commits of this branch since origin/main, and the current directory holds the repository after the change. Read the diff file first, then the files it touches as needed. Review for: (1) correctness issues, (2) security concerns, (3) error handling gaps, (4) logic errors, (5) missing test coverage, (6) blind spots specific to the change. Report only material findings. Number each finding with severity [HIGH/MEDIUM/LOW], and give its affected file:line, what can go wrong, why the code is vulnerable, the likely impact, and a concrete change that fixes it. Prefer one strong finding over several weak ones. If the change looks solid, say so directly with no findings." </dev/null > "$p-out.md" 2> "$p.log" & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "claude rc=$rc"; else echo 'diff: failed'; fi
if test -s "$p-out.md"; then echo 'out: not empty'; else echo 'out: empty'; fi
{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; } > "$p-git-after.txt"
if cmp -s "$p-git-before.txt" "$p-git-after.txt"; then echo 'git: same'; else echo 'git: changed'; fi
```

### 出た行

```
reviewer=claude
alias claude='claude --model opus --effort xhigh --enable-auto-mode --dangerously-skip-permissions'
available rc=0
claude rc=0
out: not empty
git: same
```

- 所要時間: 11 分 46 秒。20 分の見張りには届いていない
- git の状態: 前後で同じ(`cmp` 一致。HEAD 586f7934 の 1 行だけ)
- 出力ファイル: `-out.md` は 3925 バイト(23 行)。`.log`(stderr)は 0 バイト。`.diff` は 99550 バイトで、`diff: failed` は出ていない

### `-out.md` の先頭 15 行

````
I reviewed the diff, the reviewer template it changes, the helper scripts the new commands source, the fixture code the new tests reuse, and the leader's plan-advisory section the commands are modelled on. I could not execute the Go tests with the read-only tools I have, so I traced each shell test by hand against the stubs and the fixture's PATH and environment handling. They look sound, including the duplicate PATH override, the watchdog kill path, and the triage-counter round trip.

The change is a faithful port of the plan-advisory pattern and the fixture-driven tests cover the mechanics well. I found one material gap and three small ones.

**1. [MEDIUM] The incomplete path drops the error that explains why the other model failed.**
`internal/org/prompts/reviewer.md:283-285`. When the called CLI exits non-zero or produces no output, the reviewer writes `incomplete (<理由>)` with no guidance on what the reason must contain. The plan's own Risks section predicts the most likely real-world failure: a Codex reviewer seat under `--sandbox workspace-write` cannot reach the network, so the nested `claude -p` fails. In that case `command -v claude` succeeds, so the run lands on this incomplete branch, not the skipped branch that step 2 decorates with the codex-check reason line. The error text sits unused in the cycle's log file, and the PR body ends up saying `Cross-review: incomplete (claude rc=1)` on every PR of that org. The plan's stated mitigation, making the cause visible in the PR, is therefore implemented on the wrong branch. Fix: in the step 4 bullet, require the reason to be `<cli> rc=<n>` followed by `: ` and the first non-empty line of `.harness/state/xreview-{{ORG_ID}}-c<k>.log`, and name `rc=143` as the watchdog's 20-minute stop. Pin the sentence in `TestRenderRolePrompt_Reviewer_XreviewSortsTheRunByItsOutput`.

**2. [LOW] Step 2's command does not guard an invalid driver value itself.**
`internal/org/prompts/reviewer.md:183`. `pick_reviewer` lowercases its argument and returns `codex` for anything it does not recognise. Step 1 asks the reviewer to send a QUESTION first, but that is prose only and nothing in the test suite exercises a bad value through the command. A one-word guard at the head of the line makes the mechanical path fail closed too:

```
case "<IMPLEMENTER_DRIVER>" in claude|codex) ;; *) echo "driver: invalid"; false ;; esac
```

Add a `XreviewFindsNoCLI`-style case with `IMPLEMENTER_DRIVER: bedrock` that expects no call and `available rc=1`.
````

triage できるか: できる。所見は 4 件で、`[MEDIUM]` が 1 件、`[LOW]` が 3 件。どれも `internal/org/prompts/reviewer.md:<行>`(283-285、183、186-188、249)が付いている。指示文どおりの番号付きの形なので、`Reviewer status: complete` の report を書ける。

## 節のとおりに動かなかったところ

- 2 方向とも、節のコマンドは書いたとおりに走った。引用符の誤り、足りないフラグ、shell のエラーはなかった
- zsh では `command -v claude` が binary のパスではなく alias の定義を返した(B の 2 行目)。rc は 0 なので判定は変わらない。この機械では binary もある(`type -a claude` で実体 2 つ)ので結果は正しい。alias だけがあって binary がない環境では、手順 2 が `available rc=0` を返し、手順 3 の `command claude` で失敗して incomplete に落ちるとみられる(未確認)。skipped にはならない
- codex には節の英文の指示文が渡らない(`exec review` は組み込みのレビュー)。重大度は `[P1]` の形で返る
- codex は read-only の sandbox の中で、worktree の外の `~/.codex/memories/MEMORY.md` を `rg` で読んだ(log の最初の exec)。読むだけで、git の状態は変わっていない

## 結論

2 方向とも、586f7934 の節のコマンドをそのまま打って相手が最後まで走り、`<cli> rc=0`、`out: not empty`、`git: same` がそろった。どちらの出力にも重大度と file:line の付いた所見があり(codex 1 件、claude 4 件)、`Reviewer status: complete` の report を書ける。これはこの機械の shell で走らせた結果で、座席の中(sandbox、network、pane の環境)での動きはこの記録では確かめていない。
