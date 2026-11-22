# 座席の送信と codex の hook の確かめ(2026-11-22)

issue #361 の計画 `docs/plans/active/2026-11-22-org-result-missing.md` の S0(受け入れ条件 AC1)の記録。codex の座席がゲートの結果を出したあと `• Hook failed └ hook exited with code 1` を出し、RESULT が leader に届かなかった。この repo の Codex の PreToolUse の hook が座席の agmsg の送信を落としたのかを、手元で確かめた。モデルは呼んでいない(`codex exec` も `claude -p` も使っていない)。

## 環境

- codex-cli 0.166.0、git 2.49.0、macOS 26.5.2(25F84)、agmsg 1.5.1
- jq 1.8.2(`/opt/homebrew/bin`)と jq 1.7.1-apple(`/usr/bin`)。`/bin/sh` は bash 3.2.57
- hook はこの worktree(414eb7f7)のもの。PreToolUse に関わる `.claude/hooks/` の 7 ファイルと `.codex/hooks.json` は、`templates/base/` の写しと `cmp` で同じ

## codex 0.166.0 の hook の動き(ソースで確かめた)

`gh api` で openai/codex のタグ `rust-v0.166.0` のファイルを読んだ。

- 入力: Bash の呼び出しでは `tool_name` が `"Bash"`、`tool_input` が `{"command": "<文字列>"}`(`codex-rs/core/src/tools/handlers/unified_exec/exec_command.rs` の `pre_tool_use_payload`)。Claude Code と同じく `tool_input.command` にコマンドの文字列が入る。ほかの鍵は `session_id`、`turn_id`、`transcript_path`(null のことがある)、`cwd`(ツールの作業ディレクトリ)、`hook_event_name`、`model`、`permission_mode`、`tool_use_id`(`codex-rs/hooks/schema/generated/pre-tool-use.command.input.schema.json`)
- 起動: `$SHELL -lc '<hooks.json の command>'`、`SHELL` がなければ `/bin/sh -lc`(`codex-rs/hooks/src/engine/command_runner.rs` の `build_command`)。作業ディレクトリはセッションの cwd(`codex-rs/core/src/hook_runtime.rs` の `tool_hook_cwd`)で、環境はセッションの環境の写し。`TMPDIR` もそのまま渡る。sandbox はかけない。時間切れは 600 秒
- 終了コード: 0 なら標準出力の JSON を読み、deny なら Blocked。2 なら標準エラーを理由にして Blocked。それ以外は Failed で、文言は `hook exited with code N`(`codex-rs/hooks/src/events/pre_tool_use.rs` の `parse_completed`)。`hook_runtime.rs` の `run_pre_tool_use_hooks` は Blocked のときだけコマンドを止めるので、PreToolUse の hook が Failed で終わってもコマンドはそのまま走る
- 画面: Failed は `• Hook failed` と `└ hook exited with code N` の 2 行(`codex-rs/tui/src/history_cell/hook_cell.rs`)。どのイベントの hook かも、hook の標準エラーも出ない。この repo の `.codex/hooks.json` は PreToolUse(Bash)、PostToolUse(Edit・Write・MultiEdit・apply_patch)、SessionStart、UserPromptSubmit を `ralph-dispatch.sh` に渡すので、画面の 1 行からはどの hook が落ちたかを決められない

## 1 段目: `ralph-dispatch.sh PreToolUse` に座席の形のコマンドを渡す

座席が打ちそうな 18 のコマンドを作り、それぞれを codex の形(1 行の JSON)と Claude Code の形の入力に載せて、worktree の root から `./.claude/hooks/ralph-dispatch.sh PreToolUse` に渡した。

| 名前 | コマンド(送信は `~/.agents/skills/agmsg/scripts/send.sh feat-a reviewer leader …`) |
|---|---|
| c01 | 短い本文をダブルクォートで(本文に改行) |
| c02 | `"$(cat /tmp/feat-a-result.md)"` |
| c03 | `body=$(cat <<'MSG' … MSG)` のあと `"$body"` |
| c04 | 引数の中に `"$(cat <<'MSG' … MSG)"` |
| c05a | シングルクォートの本文: バッククォート、`$(not run)` の文字、日本語、`→`、`PLAN: … sha256:<64 桁>`、`./scripts/run-static-verify.sh` |
| c05b | c05a と同じ本文をダブルクォートで。バッククォートと `$` は `\` で逃がす |
| c05c | ダブルクォートの本文に、逃がさないバッククォート(`` `ralph org gate --org feat-a` ``) |
| c06a・c06b・c06c | 本文かコメントに guard が拒む語(`git push --force`、`git reset --hard`、`sudo `) |
| c07a・c07b・c07c | ゲートの前に打つコマンド: `ralph org gate --org feat-a`、`./scripts/run-static-verify.sh && ./scripts/run-test.sh`、`cd <worktree> && ./scripts/run-test.sh 2>&1 \| tail -50` |
| c08 | `history.sh feat-a reviewer 20` |
| c09 | heredoc で本文のファイルを書き、送り、`history.sh` を読む(1 回の呼び出しで) |
| c10a・c10b | 長い本文をダブルクォートで(12 KB と 125 KB) |
| c11 | 本文に `\` で終わる行 |

結果は次のとおり。

| 走らせ方 | 回数 | 終了コード |
|---|---|---|
| dispatcher を直に、codex の形と Claude Code の形 | 36 | すべて 0。c06a・c06b・c06c の 6 回は deny の JSON を出した(codex の画面では `Blocked by hook` になる)。ほかは出力なし。1 回 0.14〜1.5 秒 |
| `.codex/hooks.json` の command を `/bin/zsh -lc`・`/bin/sh -lc`・`/bin/bash -lc` で、root と `internal/org` から(c01・c02・c05a・c06a・c09・c10b) | 24 | すべて 0(c06a は deny) |
| jq を除いた PATH で dispatcher を直に(lib_json.sh の sed と awk の道。c01・c02・c05a・c06a・c09・c10b) | 6 | すべて 0(c06a は deny) |

条件を変えた結果は次のとおり(入力は c01 か c02 の codex の形)。

| 条件 | 終了コード | 標準エラー |
|---|---|---|
| hook の cwd が git の外(3 つのシェルで) | 127 | `fatal: not a git repository …` と `./.claude/hooks/ralph-dispatch.sh: No such file or directory` |
| `TMPDIR` が存在しない | 1 | `mktemp: mkstemp failed on <TMPDIR>/ralph-dispatch-stdin.XXXXXX: No such file or directory` |
| `TMPDIR` に書けない(mode 555) | 1 | 同じ行で、末尾が `Permission denied` |
| `TMPDIR` を消す | 0 | なし |
| `env -i HOME=… PATH=/usr/bin:/bin` | 0 | なし |
| `tool_input.command` が配列 | 0 | なし |
| `tool_input` が文字列 | 5 | なし(jq の失敗。`pre_bash_guard.sh:167`) |
| 途中で切れた JSON | 5 | なし |
| 空の入力 | 0 | なし |

1 を返したのは `TMPDIR` が無いか書けないときだけで、`ralph-dispatch.sh:108` の `mktemp` が `set -eu` の下で失敗するのが原因。dispatcher はどのイベントでも先にこの `mktemp` を通るので、この条件では PreToolUse に限らず hooks.json のすべての hook が毎回 1 で終わる。座席の形の入力(上の 3 つの表の 66 回)では、1 を返さなかった。

## 2 段目: codex の sandbox の中で送信を走らせる

1 段目は座席の形の入力では 1 を返さず、codex は PreToolUse の 1 でコマンドを止めない。hook だけでは届かなかった理由にならないので、2 段目も行った。

- agmsg 1.5.1 の skill の `scripts/` と小さな静的ディレクトリを一時ディレクトリに写し、`db/`・`teams/`・`run/` は空で始めた。本物の `~/.agents` の DB と herdr のサーバーには触れていない
- 環境は `env -i` で作った。一時の `HOME`、空の一時の `CODEX_HOME`(利用者の `~/.codex/config.toml` は読まれない)、`AGMSG_SELF_NAME=off`、`AGMSG_SELF_RENAME=off`
- cwd は一時の git repo で、そこから `send.sh probe-team reviewer leader "$(cat result.md)" --force` を走らせた(一時の team には登録がないので `--force`)
- sandbox は `codex sandbox -c 'sandbox_mode="workspace-write"' -c 'sandbox_workspace_write.network_access=true' -- <コマンド>`。座席の `--sandbox workspace-write` と #362 の `-c` に合わせた

| ケース | send.sh | messages に保存 |
|---|---|---|
| sandbox なし | 0、`Sent to leader in team probe-team` | された |
| sandbox、agmsg の写しは書ける場所(`/tmp` の下) | 0、同じ | された |
| sandbox、agmsg の DB のディレクトリだけ書けない(写しを利用者の `TMPDIR` の下に置き、座席の `TMPDIR` は `/tmp` の下にした) | 1、標準出力も標準エラーも空 | されなかった |
| 同じ条件の `history.sh probe-team reviewer 5` | 1、`Parse error in 6th command line argument: unable to open database file (14)` | — |
| sandbox、`/tmp` と `TMPDIR` も書けない(`exclude_slash_tmp`・`exclude_tmpdir_env_var`) | 1、空 | されなかった |
| sandbox なし、`TMPDIR` が存在しない | 0、`Sent to …` | された |
| 同じ条件の `history.sh` | 13、`mktemp: mkstemp failed …` | — |
| 同じ条件の PreToolUse の hook | 1 | — |

`bash -x` で追うと、DB に書けないときは `storage_send` の中の `agmsg_sqlite -bail <db>/messages.db` が 1 を返し、`send.sh` の 142 行目 `MSG_ID="$(storage_send …)"` で `set -e` により何も出さずに終わる。

補足: codex は hook を sandbox の外で走らせる(上のソース)。参考に hook を sandbox の中でも走らせると、`TMPDIR` に書けるときは 0、書けないときは `mktemp` の `Operation not permitted` で 1 だった。PostToolUse(apply_patch の入力、絶対パスと相対パス)、SessionStart、UserPromptSubmit の hook も `/bin/zsh -lc` で走らせ、すべて 0 だった。

## 結論

再現しなかった。この repo の PreToolUse の hook は、座席の形の送信のコマンドを渡しても 1 を返さない。1 を返すのは `TMPDIR` が無いか書けないときだけで、そのとき送信は通る。codex 0.166.0 は PreToolUse の hook が 1 で終わってもコマンドを止めない。`Hook failed … code 1` の hook が送信を落としたのではない(ソースと hook 単体の終了コードからの判断で、座席の TUI の中では確かめていない)。

RESULT が届かなかった理由の見込み(未確認): 送信そのものの失敗。座席の sandbox で agmsg の DB のディレクトリに書けないと、agmsg 1.5.1 の `send.sh` は何も出さずに 1 で終わり、何も保存しない。`ralph doctor` の agmsg の書き込みの確かめ(`internal/cli/doctor_codex_writable_root.go` の `checkCodexAgmsgWritableRoot`)が見ているのと同じ条件にあたる。issue の環境でこの条件が成り立っていたか、`Hook failed` の行がどのイベントの hook だったかはわからない。

## 直しについて

- hook は送信を落としていないので、この計画で hook は変えない
- 届かなかったことを見えるようにするのは、計画の S1(座席が終了コードと履歴を確かめる)と S2(見張りの `result_missing`)。S1 には気をつける点が 2 つある。送信の失敗は標準エラーが空のことがあるので、`SEND FAILED:` の行の「エラーの要点」は空になりうる。DB に書けない条件では `history.sh` も 1 で終わるので、履歴を読めないことを送れた証拠に数えてはいけない
- 計画の外に残すもの: `ralph-dispatch.sh` の `mktemp` が失敗したときの扱い(`TMPDIR` が壊れると hook がすべて 1 で終わる)と、agmsg の `send.sh` が何も出さずに 1 で終わる点(agmsg の側で、計画の Non-goals)

## 確かめていないこと

- herdr の pane に立てた codex の座席(TUI)の中での再現と、Failed の PreToolUse のあとにコマンドが走ること(モデルを呼ぶので、この段では行っていない。後者はソースからの判断)
- issue の環境の `TMPDIR`、`~/.codex/config.toml` の writable_roots、agmsg の版
- issue の座席で `Hook failed` を出した hook のイベント
