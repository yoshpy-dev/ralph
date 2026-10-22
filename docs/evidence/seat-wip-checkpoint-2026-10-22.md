# 座席の pane の WIP コミットを止める値の実機確認

- Date: 2026-10-22(JST 17:23〜17:32、UTC 08:23〜08:32)
- Plan: `docs/plans/active/2026-10-22-seat-wip-checkpoint.md`(S3、AC6)
- 生の出力はセッションの scratchpad に置き、リポジトリにはコミットしていない。要点はこのファイルに写した。ホームディレクトリは `~`、分離に使った一時ディレクトリは `<D>` に置き換えた。pane の id、session の id、一時の repo のコミットの id は写していない。

## 環境

| 項目 | 値 |
|---|---|
| マシン | macOS(Darwin 25.5.0) |
| herdr | 0.7.5。`herdr server`(ヘッドレス)を分離して起動した |
| claude | Claude Code 2.1.291、`--model haiku`(Haiku 4.5) |
| git | 2.49.0 |
| shell | zsh 5.9(pane の shell) |
| hook | この PR の S1 のコミットの `.claude/hooks/` と `.claude/settings.json`(`git archive` で丸ごと写した) |
| ralph | 使っていない。herdr を直接打った |

### 分離のしかた

- herdr server は `/tmp` の下の一時ディレクトリ `<D>` で起動した。`env -i` で環境を空にしてから、`HOME`(本当の HOME。claude の認証のため)、`USER`、`LOGNAME`、`SHELL`、`LANG`、`TERM`、`TMPDIR`、`PATH`、`XDG_CONFIG_HOME` と `XDG_STATE_HOME`(どちらも `<D>` の下)、`HERDR_SOCKET_PATH`(`<D>/h.sock`)だけを渡した。この server に打つ herdr のコマンドも同じ環境で打った。このセッションの `HERDR_PANE_ID` と Orca の変数は渡していない。
- 一時の repo `<D>/work` を作り、S1 のコミットから `.claude/hooks/` と `.claude/settings.json` を写した。settings.json は SessionEnd と PreCompact を `./.claude/hooks/ralph-dispatch.sh` に送る。repo には名前とメールを repo の設定で入れ、`core.hooksPath` を `<D>` の下の空のディレクトリにして、ユーザーの git hook が走らないようにした。`.gitignore` に `.harness/state/` と `.harness/logs/` を書いた。base のコミットを main に置き、`feat/probe` を切り出してから汚れたファイル 1 つ(`notes.txt` の変更)を作った。
- pane は 2 つ作った。どちらも `herdr tab create --cwd <D>/work` で作り、off の pane には `--env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off` を、対照の pane には `--env AGMSG_SELF_NAME=off` だけを付けた。
- pane の shell はユーザーの zsh の設定を読み、その設定は `claude` に model と permission の引数を足す alias を定義している。agent を起動する前に、どちらの pane でも `unalias claude` を打ち、`whence -p claude` が `~/.local/share/mise/installs/npm-anthropic-ai-claude-code/latest/node_modules/.bin/claude` を指すことを確かめた。
- claude は `herdr agent start <名前> --kind claude --pane <pane> -- --model haiku` で起動した。herdr の返した `argv` は、どちらも `["claude","--model","haiku"]` だった。1 つ目の起動でフォルダ信頼の確認が出たので「Yes, I trust this folder」を選んだ。2 つ目は同じフォルダなので確認は出なかった。

## 結果

### (a) 2 つの値は pane の shell に届く

それぞれの pane の shell で `AGMSG_SELF_NAME` と `RALPH_WIP_CHECKPOINT` を書き出した。

| pane | `AGMSG_SELF_NAME` | `RALPH_WIP_CHECKPOINT` |
|---|---|---|
| off(`--env` を 2 つ) | `off` | `off` |
| 対照(`--env` を 1 つ) | `off` | 未設定 |

### (d) PreCompact: off ならコミットせず、対照ならコミットする

どちらの pane でも、claude に「Reply with the single word ok. Do not use any tools.」を送り、`ok` の返事を見てから `/compact` を送った。`/compact` の前に木は `notes.txt` の変更 1 つで汚れていた。

| pane | `precompact-checkpoint.md` | コミット | index |
|---|---|---|---|
| off | 書かれた。最後の行は `- WIP commit: skipped (RALPH_WIP_CHECKPOINT=off)` | 増えなかった(base の 1 つのまま) | 変わらなかった(`git diff --cached` は空) |
| 対照 | 書かれた。skipped の行はない | `wip: checkpoint before context compaction` が 1 つ増えた(`notes.txt` の 1 行) | 変更はそのコミットに入った |

PreCompact は `/compact` で起こせた。

### (b) SessionEnd: off の pane を C-c と close で終わらせるとコミットしない

`ralph org stop` の `stopSeatPane`(`internal/org/verbs.go`)は、pane に C-c を 1 回送り、間を置かずに `herdr pane close` を打つ。同じ順で打った。打つ前の木は `notes.txt` の変更 1 つで汚れていた。

- `session-end.md` が書かれた。`## Git status` の下は ` M notes.txt` で、最後の行は `- WIP commit: skipped (RALPH_WIP_CHECKPOINT=off)` だった。
- コミットは増えなかった。木は ` M notes.txt` のままで、`git diff --cached` は空だった。
- close のあと、`claude --model haiku` のプロセスは残っていなかった。

### (c) SessionEnd: 対照の pane では WIP コミットが 1 つ増える

(d) の対照のコミットのあとで、`notes.txt` に 1 行足して木を汚し直し、(b) と同じく C-c と close で終わらせた。

- `session-end.md` が書かれた。skipped の行はない。
- `wip: checkpoint before session end` が 1 つ増えた(`notes.txt` の 1 行)。木はきれいになった。
- close のあと、`claude --model haiku` のプロセスは残っていなかった。

### 分かったこと

- claude の hook は、herdr が pane に足した環境を受け継ぐ。`ralph-dispatch.sh` を通っても値は落ちない。計画の D1 を見直す必要はない。
- `ralph org stop` と同じ C-c と close で、claude の SessionEnd の hook は走りきる。試行の F11 の WIP コミットが、止められた座席の SessionEnd から出たという推定と合う(試行のコミットを作ったセッションを、この確かめで特定したわけではない)。

## 触れていないもの

- ユーザーの herdr server には触れていない。確かめのあとに `pgrep -fl 'herdr server'` で見ると、動いている herdr server はなかった。`~/.config/herdr` の下には、確かめを始める前の JST 17:15 より新しいファイルがない。
- Orca は打っていない。
- 確かめのあと、分離した server を `herdr server stop` で止め、そのプロセスがないことを確かめてから `<D>` を消した。cwd が `<D>` の下にある claude がないこと(`lsof -a -d cwd -c claude`)も確かめた。

### 確かめで残ったもの

- claude のフォルダ信頼の確認に「信頼する」と答えたので、`~/.claude.json` に `<D>/work` を信頼済みとする項目が 1 つ増えた。`<D>` は消したが、この項目は残した。ほかの Claude Code のセッションも同じファイルに書くので、手では消していない。
- 2 つの claude の session の記録が、`~/.claude/projects/` の下に残っている。
