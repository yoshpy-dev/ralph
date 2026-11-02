# config-path-and-ledger-docs の実物の確かめ(AC6、tech-debt 421 (e))

- Date: 2026-11-02(A は JST 16:46:07〜16:46:40、B は 16:45:19〜17:09:37)
- Plan: `docs/plans/active/2026-11-02-config-path-and-ledger-docs.md`(Scope の 421 (e)、D3、AC6)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- 報告: `docs/reports/test-2026-11-02-config-path-and-ledger-docs.md`
- ralph: 作業ツリーの HEAD 81e469b4 から `go build -ldflags "-X main.Version=test-cfgpath -X main.GitCommit=81e469b4 -X main.BuildDate=2026-11-02" -o <scratchpad>/test-delta/bin/ralph ./cmd/ralph` で作ったもの。`ralph version` は `ralph test-cfgpath (81e469b4 2026-11-02)`。PATH の `/opt/homebrew/bin/ralph` は `ralph 6.0.0 (44c14b97 2026-10-27T01:34:06Z)` で、`config get` がない
- 道具: codex-cli 0.163.0、Claude Code 2.1.308、herdr 0.7.5。ユーザーの `~/.codex/config.toml` の既定は `model = "gpt-6.1-sol"`、`model_reasoning_effort = "max"`。ralph の既定は `gpt-6-astra`、`xhigh`
- 写していないもの: token、認証の値、session id、pane id。codex の session の記録のファイル名は、UUIDv7 の先頭 8 桁(時刻の部分)までを書き、残りを `…` にした。ホームディレクトリは `~`、B の専用ディレクトリは `<D>`(`/tmp/rcfg` に 5 文字を足した名前。終わりに消した)と書く
- 生の出力: B の argv の記録、ledger、leader の役割指示のファイル、pane の写しは scratchpad の `test-delta/live-copy/` に写した。コミットしていない

## A: 既定と違う codex の slug で reviewer を起こす(D3、必ず取る)

### 値

slug は `gpt-6-luna` にした。`~/.codex/models_cache.json` の 10 件のうち、`gpt-6-luna` の `upgrade` は `null` で、退役の知らせがない。ralph の既定(`gpt-6-astra`)ともユーザーの codex の既定(`gpt-6.1-sol`)とも違う。

scratchpad の `test-delta/a/repo/` に `git init` で一時の repository を作り(ralph の checkout の外)、作業ツリーの `scripts/ralph-config.sh` を `scripts/` に写して、次の `ralph.toml` をコミットした。

```toml
[reviewer]
codex_model = "gpt-6-luna"
codex_effort = "low"
```

### 起こし方

PATH の先頭に build した ralph を置き(`command -v ralph` がそれを指すことを確かめた)、一時の repository の root から、skill と同じ形で読んで起こした。見張りの上限は Bash ツールの 600 秒に収まるよう 540 秒にした。

```
. ./scripts/ralph-config.sh; model=$(ralph_setting reviewer.codex_model); effort=$(ralph_setting reviewer.codex_effort)
rm -f <S>/a/codex-last.md; command codex -m "$model" -c "model_reasoning_effort=$effort" exec --sandbox read-only --ignore-rules -o <S>/a/codex-last.md "Reply with the single word ok." </dev/null > <S>/a/codex.log 2>&1 & cpid=$!; sleep 540 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "codex rc=$rc"
```

### 結果

| 項目 | 値 |
|---|---|
| helper が読んだ値 | `model=gpt-6-luna effort=low` |
| 対照(`ralph.toml` のない scratchpad の `test-delta/a/` から同じ helper) | `gpt-6-astra`、`xhigh`(既定値) |
| codex | `codex rc=0`、16:46:19 から 16:46:36 の 17 秒。`-o` のファイルは `ok` |
| codex の log の頭 | `model: gpt-6-luna`、`sandbox: read-only`、`reasoning effort: low` |
| session の記録 | `~/.codex/sessions/2026/11/02/rollout-2026-11-02T16-46-21-01a19b39-….jsonl`(`CODEX_HOME` は未設定)。`session_meta` は `originator=codex_exec`、cwd は一時の repository。`turn_context` は 1 件で、`model=gpt-6-luna`、`effort=low`、sandbox `read-only` |

session の記録の `turn_context.model` は、`ralph.toml` に書いた slug `gpt-6-luna` だった。effort も `low` だった。codex の log には、ユーザーの hook の `SessionStart Failed` と `Stop Failed` の行がある(ユーザーの設定によるもので、rc と答えには関係しない)。

### main checkout の `.codex/config.toml`

main checkout(`~/MyDev/github.com/yoshpy-dev/ralph`、HEAD 350f22ea)で確かめた。

| 時点 | `git status --porcelain .codex/config.toml` | `shasum -a 256` の頭 16 桁 |
|---|---|---|
| A の前(16:46:07) | 空 | `96919613aeacafd5` |
| A の後 | 空 | `96919613aeacafd5` |
| B の片付けの後(17:10) | 空 | `96919613aeacafd5` |

main checkout と作業ツリーの `git status --porcelain` も、終わりまで空だった。

## B: leader の通し(2 つの場合)

### 隔離の準備(`docs/recipes/isolated-herdr-server.md` との対応)

| レシピの段 | したこと |
|---|---|
| 専用ディレクトリ | `mkdir -m 700 /tmp/rcfg<5 文字>`(`<D>`) |
| wrapper `<D>/run` | `env -i` で `HOME`(本物)、`USER`、`LOGNAME`、`SHELL=/bin/bash`、`LANG`、`TERM=xterm-256color`、`TMPDIR`、`PATH`、`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH=<D>/h.sock`、`CLAUDE_ENV_FILE=<D>/claude-env`、`GIT_CONFIG_GLOBAL`(git の設定は `~/.config/git/config` にある)。gh は使わないので `GH_CONFIG_DIR` は入れていない |
| PATH | `<D>/bin`、node の install 先、`/opt/homebrew/bin`、`/usr/bin:/bin:/usr/sbin:/sbin`。`/opt/homebrew/bin` には古い ralph 6.0.0 があり、`<D>/bin` の ralph が先に見つかるかを前提の確かめで見た |
| `<D>/bin` | build した ralph の写し、osascript の shim(レシピのとおり)、引数を `<D>/argv-<名前>.log` に 1 行で書いてから本物を `exec` する `claude` と `codex` の shim(レシピにない追加。起こした引数を確実に残すため。`exec` するので、起きるプロセスは本物と同じ) |
| `CLAUDE_ENV_FILE` | `export PATH="<wrapper と同じ PATH>"` の 1 行 |
| agmsg | `~/.agents/skills/agmsg` の `scripts`、`plugins`、`types`、`SKILL.md`、`VERSION` を `<D>/agmsg` に写し、`ralph.toml` の `[org].agmsg_home` に書いた。この確かめでは座席を spawn しない(leader は承認の QUESTION で止まる)ので、レシピの言う写しの限りは当たらない |
| server | `nohup <D>/run herdr server &`。`lsof -p` で開いているファイルは `<D>` の下と `/dev/null` だけ。`ps eww` の環境変数の名前は wrapper のものだけ。`herdr --help` の `Config:` と `Logs:` は `<D>/xdg-config/herdr/` の下 |
| repository | `<D>/work` に `git init -b main`、`<D>/run ralph init --yes .`(build した ralph)、repository の設定で `user.name`、`user.email`、`commit.gpgsign=false`、空の `core.hooksPath`。bare の origin を `<D>/origin.git` に作って push し、`origin/HEAD` を main にした |
| folder trust | 使い捨ての pane で `claude --model haiku` を起こし、「Yes, I trust this folder」に Down と Enter で答えて `/exit` した。ダイアログは「This folder pre-approves 62 tool permissions in .claude/settings.json」と出した |
| doctor、status | `ralph doctor` は rc 0。herdr は pass、agmsg は写しを指して info(1.5.1)、「Codex sandbox (agmsg writable root)」は pass。「Shell aliases」は `~/.zshrc` の `claude` と `codex` の alias を warn したが、pane の shell は bash で、使い捨ての pane の `alias` は何も出さなかった。`ralph status` は台帳が `<D>/work/.harness/state/org`(`source: git-main-worktree`)、`org watch: not running` |

`ralph.toml` は template のまま次の 5 か所だけを変え、コミットした。`ralph config get` で読み直した値は右の列のとおり。

| key | 値 |
|---|---|
| `reviewer.claude_model` / `reviewer.claude_effort` | `sonnet` / `low` |
| `reviewer.codex_model` / `reviewer.codex_effort` | `gpt-6-luna` / `low` |
| `org.review.plan_advisory` | `true`(既定は false) |
| `org.agmsg_home` | `<D>/agmsg` |
| `org.watchdog.watcher_enabled` | `false`(見張りの claude の判定を起こさないため) |
| `org.approval.mode` | `per-plan`(template のまま) |

最初の書き換えでは `[org]` に `agmsg_home` を足してしまい、template の行と重なって `config get: ralph.toml: cannot be read or is invalid: toml: key agmsg_home is already defined`(rc 3)になった。template の行を書き換える形に直してから、上の値を読んだ。org を起こす前の準備の誤りで、結果には関係しない。

分割計画は台帳の `splits/cfgp.md` に置き、`./scripts/plan-visual.sh digest` の `d25a21373750` を `- Approved:` の行に書いた。機能は `lead-claude` と `lead-codex` の 2 つで、どちらも `- Type: docs`、`- Reserve: docs/<slug>.md`、`- Depends on: none`、本文は「`docs/<slug>.md` に 1 行を足す」。前文(`## Objective`)には「A throwaway repository for a live check of the leader's plan advisory」と書いた。

### 前提の確かめ

| 確かめ | 結果 |
|---|---|
| `pgrep -fl 'herdr server'`(16:45:19、準備の前) | 何も出さず rc 1。ユーザーの herdr server は動いていなかった |
| 同(server の起動の後) | 専用の server の 1 行だけ |
| 使い捨ての pane の shell(bash のログイン shell) | `type claude codex ralph` は 3 つとも `<D>/bin`。`ralph version` は `ralph test-cfgpath (81e469b4 2026-11-02)`。`alias` は何も出さない。PATH は `path_helper` のあとで `~/.local/bin`、system の dir、`<D>/bin`、node、`/opt/homebrew/bin` の順 |
| 同じ pane の `claude -p '<依頼>' --model haiku --allowedTools Bash`(claude の Bash ツール) | `command -v ralph` は `<D>/bin/ralph`、`claude`・`codex`・`osascript` は `<D>/bin` の shim、`ralph version` は `test-cfgpath (81e469b4 2026-11-02)`、`ralph config get org.review.plan_advisory` は `true` |
| 同じ pane の `codex -m gpt-6-luna -c model_reasoning_effort=low exec --sandbox workspace-write '<依頼>'`(codex の shell ツール) | `command -v ralph` は `/opt/homebrew/bin/ralph`、`claude` と `codex` は `~/.local/share/mise/shims/` のもの、`ralph version` は `ralph 6.0.0 (44c14b97 …)`、`ralph config get …` は `unknown command "config" for "ralph"` |
| 同じ形で、codex の shell を確かめる依頼 | `zsh:1: operation not permitted: ps`(sandbox が `ps` を止めた。エラーの頭から shell は zsh)、`ZDOTDIR=~/.config/zsh`、`SHELL=/bin/bash` |

codex の shell ツールは、pane の `SHELL`(`/bin/bash`)ではなく zsh でコマンドを打っている。ユーザーのログイン shell は `dscl` で `/bin/zsh`。zsh は `/etc/zshenv` で `ZDOTDIR=$HOME/.config/zsh` になり、`~/.config/zsh/.zshenv` は `path=( … )` で、受け継いだ PATH を含めずに PATH を組み直す(先頭は `$JAVA_HOME/bin`、mise の shims、`/opt/homebrew/bin` の順)。だから codex の leader が打つ `ralph` は、server や pane の PATH によらず `/opt/homebrew/bin/ralph` になる。codex が passwd の shell を使うことは、上の `zsh:1:` と `dscl` の値からの推測で、codex のソースは読んでいない。

`HOME` を偽にすれば zsh の設定ファイルの場所は変わるが、レシピと今回の指示が claude に偽の `HOME` を許さず、case 2 の leader が起こすのは claude なので使わなかった。ユーザーの zsh の設定ファイルも codex の設定も書き換えていない。よって case 2 の前提のうち「leader が打つ `ralph` が新しい binary である」は、server と pane の環境では整えられなかった。pane の shell の前提(上の 2 行目)と、worktree での `plan_advisory` の `true`(下)は整った。

### case 1: claude の leader → codex の reviewer

| 段 | 結果 |
|---|---|
| start(16:53:40) | `<D>/run ralph org start --plan .harness/state/org/splits/cfgp.md --feature lead-claude --driver claude --model sonnet`。16:53:48 に rc 0。`ralph org start: started the cross-org watch (pid 83175)`、`spawned seat "leader" (org_id=lead-claude driver=claude model=sonnet pane_id=<pane> dry_run=false)`、worktree `<D>/work/.claude/worktrees/org-lead-claude`、branch `docs/lead-claude` |
| leader の起動の引数(shim) | `--permission-mode bypassPermissions --model sonnet 役割指示を読み込んで従ってください: <D>/work/.harness/state/org/prompts/lead-claude_leader.md` |
| 前提(使い捨ての pane で `cd <worktree>`) | `command -v ralph` は `<D>/bin/ralph`、`ralph version` は `ralph test-cfgpath (81e469b4 2026-11-02)`、`ralph config get org.review.plan_advisory` は `true`(rc 0) |
| タスクの 2 行(役割指示のファイルの 1093〜1094 行) | `- reviewer の codex: gpt-6-luna low(モデルと effort。計画助言で codex を呼ぶときの -m と model_reasoning_effort に使う)` と `- reviewer の claude: sonnet low(モデルと effort。計画助言で claude を呼ぶときの --model と --effort に使う)`。`- 承認方式: per-plan` |
| 計画助言の codex(16:54:36、shim) | 1 回目は `--disable plugins --disable apps mcp list --json`。2 回目は同じ頭に 14 個の `-c mcp_servers.<名前>.enabled=false` を足した一覧。3 回目が本番で、14 個の `-c` のあとが `-m gpt-6-luna -c model_reasoning_effort=low exec --sandbox read-only --ignore-rules -o .harness/state/plan-advisory-out.md "You are an adversarial plan reviewer. …"`。cwd は feature の worktree |
| 同(`ps -ww -axo pid,ppid,args` を 0.5 秒ごとに見た記録、16:54:37) | node の codex と native の codex の 2 つのプロセスが、どちらも `-m gpt-6-luna -c model_reasoning_effort=low exec --sandbox read-only --ignore-rules -o .harness/state/plan-advisory-out.md` を含む |
| codex の session の記録 | `rollout-2026-11-02T16-54-37-01a19b40-….jsonl`。`originator=codex_exec`、cwd は `.claude/worktrees/org-lead-claude`、`turn_context` は `model=gpt-6-luna`、`effort=low`、sandbox `read-only` |
| worktree の `.harness/state/plan-advisory.log` の頭 | `model: gpt-6-luna`、`sandbox: read-only`、`reasoning effort: low` |
| 助言の結果 | 背景の codex は exit 0。`plan-advisory-out.md` は 1,153 バイト(所見 2 件)。`plan-advisory-git-before.txt` と `-after.txt` は `cmp` で同じ。MCP の一覧で有効だった server は 14 個 |
| QUESTION(16:56:00、受信箱の e1) | 本文に `PLAN: docs/plans/active/2026-11-02-lead-claude.md sha256:1ffbc7d81292`、`VISUAL: .harness/state/plan-visual/lead-claude.html`、`ADVISORY: codex の計画助言の所見は 2 件(HIGH 1、MEDIUM 1)。HIGH は「助言の確認はこの機能ではなく harness の役目」と Design decisions に書いた。MEDIUM はステージした差分の確認をスライスに足した。` |
| branch のコミット | 3 つ(`docs: add plan for lead-claude`、`docs: reflect plan advisory in lead-claude plan`、`docs: record visual review for lead-claude plan`) |
| leader の待ち方 | Monitor(受信箱の見張り、timeout 1800s)を張って手番を終えた |
| disband(16:56:59) | rc 0、`stopped seat "leader"`、`disbanded org "lead-claude"`。QUESTION には答えていない |

起こした codex の `-m` と `model_reasoning_effort` は、タスクの `reviewer の codex:` の行の `gpt-6-luna` と `low` に一致した。start から QUESTION まで 2 分 20 秒だった。

### case 2: codex の leader → claude の reviewer

| 段 | 結果 |
|---|---|
| start(16:57:23) | `--feature lead-codex --driver codex --model gpt-6-astra`。16:57:37 に rc 0。worktree `org-lead-codex`、branch `docs/lead-codex` |
| leader の起動の引数(shim) | `--sandbox workspace-write --ask-for-approval never --add-dir <D>/work/.harness/state/org --model gpt-6-astra 役割指示を読み込んで…`。effort は ralph が渡さないので、ユーザーの既定の `max` になった(pane の表示 `GPT-6-Astra · max`) |
| 前提(使い捨ての pane で `cd <worktree>`) | `<D>/bin/ralph`、`ralph test-cfgpath (81e469b4 2026-11-02)`、`plan_advisory` は `true`(rc 0)。タスクの 2 行は case 1 と同じ値 |
| codex のダイアログ | project の信頼の画面は出ず、「Hooks need review / 5 hooks are new or changed. / Hooks can run outside the sandbox after you trust them.」と「1. Review hooks / 2. Trust all and continue / 3. Continue without trusting (hooks won't run)」が出た。16:58:18 に 3 を選んだ(Down 2 回と Enter) |
| `~/.codex/config.toml` | 同じ 16:58:18 に書き換わり(mtime)、`[projects."/private/tmp/rcfgtcl99/work"]` と `trust_level = "trusted"` の 2 行が足された。ダイアログの答えと同時なので、codex が答えに合わせて書いたとみる。未確認 |
| leader の `ralph` の確かめ(17:00:07) | `command -v ralph`、`ralph org status … --json`、`ralph config get org.review.plan_advisory` を 1 回の shell で打ち、exit 1。1 行目は `/opt/homebrew/bin/ralph` |
| 同(17:03:49、`login:false`) | `command -v ralph; ralph config get org.review.plan_advisory` の出力は `/opt/homebrew/bin/ralph` と `unknown command "config" for "ralph"`、exit 1。ログインでない zsh でも同じ PATH(`.zshenv` は毎回読まれる) |
| ユーザーの設定による動き | ユーザーの codex の指示に従って、`/tmp/consult/20261102-170040/request.md` を書いて相談の sub-agent(`gpt-6-astra`、effort `high`)を起こした。natural-japanese の lint を `uv` で打とうとして、sandbox の `Operation not permitted` で失敗した。どちらも ralph の役割指示にはない |
| 計画 | `docs/plans/active/2026-11-02-lead-codex.md` を書いた(17:05、コミットしていない)。計画の Risks に「Git の書き込みには昇格が要るが、このセッションでは昇格が禁止されている。計画のコミットの前で止まり、BLOCKED で報告する」と「`ralph config get org.review.plan_advisory` は `unknown command "config"` で終わり、役割指示の古い版の手順では off 扱いになるので、この環境のままでは計画助言の確かめはできない」と書いた |
| 中断(17:06:11) | pane に「■ Conversation interrupted」が出て、leader は入力待ちになった。QUESTION も BLOCKED も上がっていない。session の記録には 08:06:11Z の `thread_settings_applied` があり、ユーザー入力はない。leader の codex が 16:57:26 に起こした codex の app-server daemon(0.166.0、親 pid 1)が、17:06:07 に子の `codex app-server` を起こし直している。中断はこの起こし直しと時刻が重なるが、原因は確かめていない |
| claude の起動 | なかった。`<D>/argv-claude.log` に case 2 の start のあとの行はなく、プロセスの記録にも leader の claude はない |
| 見張り | 17:05:45 に leader についての `scope_change` の ALERT を受信箱の e2 にした(leader の未追跡の計画のファイル)。既存の動き |
| disband(17:07:29) | rc 0、`stopped seat "leader"`、`disbanded org "lead-codex"` |

case 2 は、leader が `ralph config get` を打つ段で止まった。打った `ralph` が `/opt/homebrew/bin/ralph`(6.0.0)で、`unknown command "config"` を返したため、役割指示の規則では計画助言は off とみなされ、claude は呼ばれない。前提の「leader の `ralph` が新しい binary」が整っていない場合で、計画の D3 と AC7 のとおり「確かめていない」とする。

ralph の receipts には、case 2 の leader について spawn のときの `unknown`(「no codex session record for this spawn yet」。hooks のダイアログで最初の手番が始まっていなかった)と、stop のときの `gpt-6-astra`、`honored=true` の 2 行がある。

### 補い: leader の claude のコマンドを手で打つ(leader の通しではない)

case 2 の leader が打つはずだった claude のコマンドの形と値を確かめるため、case 2 の役割指示のファイル(展開済み)から計画助言の claude の囲みを取り出し、`<reviewer の claude のモデル>` と `<reviewer の claude の effort>` を、同じファイルのタスクの `- reviewer の claude:` の行の括弧の前の 2 語(役割指示の読み方のとおり)で、`<指示文>` を英文の指示文で埋めた。それを wrapper を通した bash で、case 2 の worktree から打った(17:07:59〜17:08:22)。

| 項目 | 値 |
|---|---|
| タスクの行から読んだ値 | `sonnet low` |
| 起きた claude の引数(shim) | `-p --model sonnet --effort low --restricted --strict-mcp-config --disallowedTools mcp__* --tools Read,Grep,Glob --permission-mode dontAsk --output-format text "You are an adversarial plan reviewer. …"` |
| 結果 | `claude rc=0`、`git: same`、出力 3,292 バイト(所見つき) |
| claude の session の記録(`~/.claude/projects/-private-tmp-rcfgtcl99-work--claude-worktrees-org-lead-codex/`) | assistant の 5 件の `model` はどれも `claude-sonnet-5-5`、entrypoint は `sdk-cli`。effort は記録に出ない |

これは leader が起こしたものではないので、case 2 の判定には使わない。囲みの形と、タスクの行の値をそのまま渡した claude が受け付けて動くことだけを示す。

## 片付け

| 段 | 結果 |
|---|---|
| org | 2 つとも disband 済み(上)。受信箱の e1(QUESTION)と e2(ALERT)は答えずに残し、台帳ごと消した。director は登録していない |
| 見張り | `<D>/run ralph org watch --all --stop` は `stopped ralph org watch --all (pid 83175)`。そのあとの `ralph status` は `org watch: not running (stopped at 2026-11-02T08:08:54Z)` |
| 使い捨ての workspace | `herdr workspace close` で閉じた。workspace の一覧は空 |
| `pgrep -fl <D>`(`~` から) | 何も出さず rc 1 |
| `lsof +D <D>`(server を止める前) | 専用の herdr server のほかに、codex の app-server daemon の 2 つ(daemon と子の app-server。どちらも cwd が case 2 の worktree) |
| server | `<D>/run herdr server stop`。`pgrep -fl 'herdr server'` は 17:09:37 に何も出さず rc 1。`<D>/h.sock` は消えた。`lsof +D` に残ったのは codex の 2 つだけ |
| osascript の shim | 1 回だけ呼ばれた: `display notification "org lead-claude: e1 (QUESTION) not acked within 10 min" with title "ralph org watch"`(08:06:16Z)。デスクトップには出ていない |
| 消したもの | `<D>` の全体と、case 2 の leader が書いた `/tmp/consult/20261102-170040/` |

codex の app-server daemon は止めていない。ユーザーごとに 1 つの daemon(socket は `/private/tmp/codex-daemon-502/` の下)で、ほかのセッションの codex が使っているかもしれないため。cwd は消したディレクトリを指したまま残る。

`<D>` の外に残ったもの:

- `~/.codex/config.toml` の `[projects."/private/tmp/rcfgtcl99/work"]`、`trust_level = "trusted"` の 2 行(case 2 のダイアログの答えと同時に書かれた)
- `~/.claude.json` の `/private/tmp/rcfgtcl99/work` の信頼の項目(folder trust の答え。レシピのとおり手で消していない)
- `~/.claude/projects/` の 3 つのディレクトリ(repository、2 つの worktree)と、`~/.codex/sessions/2026/11/02/` の session の記録(A、使い捨ての pane の 3 回、case 1 の reviewer、case 2 の leader と相談の sub-agent)
- `~/.config/zsh/.zcompdump-*` の更新(codex の shell の zsh が書いた)
- ユーザーの agmsg: `messages.db` と `messages.db-wal` の時刻は 10 月 23 日のままで、`teams/` にこの run の team はない。`messages.db-shm` は 17:00 に変わったが、同じ時間に別のセッションの `run/.lastcheck-brain` も変わっており、この run の書き込みとは見ていない。`~/.config/herdr` の下に 16:44 以降に変わったファイルはない

## 限り

- case 2 は前提が整わず、leader が claude を起こすところまで進んでいない。codex の leader の計画助言が `reviewer の claude:` の行の値で claude を起こすことは、実物では確かめていない。補いの手動の実行は leader の通しではない
- codex の leader の `ralph` を新しい binary にする方法を、server と pane の環境の中では見つけられなかった。ユーザーの `~/.config/zsh/.zshenv` が PATH を組み直す機械の話で、別の機械では整う見込みがある(未確認)
- case 2 では、ralph の役割指示にない動き(ユーザーの codex の指示による相談と lint、Git の昇格の禁止の解釈)が leader の手番を使った。effort `max` もユーザーの既定から来ている
- 分割計画の前文に「live check of the leader's plan advisory」と書いたので、leader はこれが確かめであることを知っていた。case 1 の codex はこれを曖昧さとして所見に挙げ、case 2 の leader は計画の Risks に引いた
- claude の effort は claude の session の記録に出ないので、claude の `--effort` の値は、補いの実行の引数(shim)でしか確かめていない
- 各場合は 1 回ずつで、claude の leader は sonnet、codex の leader は gpt-6-astra だけを試した
