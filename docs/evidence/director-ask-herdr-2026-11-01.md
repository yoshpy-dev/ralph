# 実機の確かめ: herdr の pane の claude で AskUserQuestion を出す(issue #271 の S1)

- Date: 2026-11-01(JST 04:57〜05:01、UTC 2026-10-31 19:57〜20:01)。時刻はすべて UTC で書く
- Plan: `docs/plans/active/2026-11-01-director-split-questions.md`(S1、AC6 の前半、D6 の枝を選ぶ材料)
- 写した画面では、一時ディレクトリを `<D>`、pane の id を `<pane>`、`$HOME` を `~` に置き換えた。session の id は写していない。生の画面は `<D>` の下に置き、要点をこのファイルに写してから `<D>` ごと消した

## 用意

| 項目 | 値 |
|---|---|
| マシン | macOS(Darwin 25.5.0) |
| herdr | 0.7.5。`herdr server`(ヘッドレス)を分離して起動した |
| claude | Claude Code 2.1.308、`--model haiku`(画面の表示は Haiku 5.5) |
| pane の shell | bash 3.2.57(`/bin/bash` の login shell) |
| pane の大きさ | 23 行(`herdr pane get` の `viewport_rows`)、53 桁(読んだ画面の最も長い行) |

- 分離のしかた: `D=$(mktemp -d /private/tmp/askq.XXXXXX)` の下に `cfg`、`state`、`proj` を作り、`proj` を `git init -q` した。scratch の project は `<D>/proj` で、ralph の repo の外にある。herdr server は `env -i` で環境を空にしてから、`HOME`(本当の HOME。claude の認証のため)、`USER`、`LOGNAME`、`SHELL=/bin/bash`、`LANG`、`TERM`、`TMPDIR`、`PATH`、`XDG_CONFIG_HOME=<D>/cfg`、`XDG_STATE_HOME=<D>/state`、`HERDR_SOCKET_PATH=<D>/h.sock` だけを渡して背景で起動した。`ps eww` で読んだ server の環境変数は、渡した 11 個だけだった
- 先例(`docs/evidence/busy-leader-2026-10-23.md` の「分離のしかた」)と同じく、handoff の「3 つの env を付ける」に `env -i` を足した。この確かめを打った側の session には `CLAUDECODE`、`CLAUDE_CODE_MESSAGING_SOCKET`、Orca の hook の変数があり、pane の claude に渡ると入れ子の session として扱われるおそれがあったため。`SHELL=/bin/bash` は、ユーザーの zsh の設定が `claude` に引数を足す alias を持つため
- `PATH` は claude の実体のディレクトリ(`~/.local/share/mise/installs/npm-anthropic-ai-claude-code/latest/node_modules/.bin`)、`/opt/homebrew/bin`、`/usr/bin`、`/bin`、`/usr/sbin`、`/sbin` の順にした。node を入れなかったので、ユーザーの hook のうち node を使うもの(SessionStart、PreToolUse:AskUserQuestion、Stop、SessionEnd)は毎回 `/bin/sh: node: command not found` で失敗した。どれも non-blocking で、問いの画面と答えの流れは止まらなかった
- herdr のコマンドは、どれも同じ `env -i` と 3 つの変数を付けて打った。付けない herdr のコマンドは 1 つも打っていないので、ユーザーの herdr server と pane には触れていない
- pane の shell を先に確かめた。`$BASH_VERSION` は `3.2.57(1)-release`、`type claude` は上の実体のパス、`env | grep -cE "^(CLAUDE|ORCA)"` は `0`

## 手順

1. `herdr workspace create --cwd <D>/proj --label askq-s1` で workspace を作った。pane の id は出力の `result.root_pane.pane_id` から取った
2. pane で次の起動用の script を `herdr pane run <pane> <D>/launch.sh` で走らせた。script は `cd <D>/proj` のあと次の行を `exec` する。日本語の prompt を bash 3.2 の入力行に打たずに済むよう script にした。argv は handoff の形と同じ
   ```
   RALPH_WIP_CHECKPOINT=off AGMSG_SELF_NAME=off claude --permission-mode bypassPermissions --model haiku --tools AskUserQuestion --strict-mcp-config "AskUserQuestion ツールを 1 回だけ使い、問い『テストの問い 1』を、選択肢『A 案(推奨)』と『B 案』で聞いてください。答えを受け取ったら、選ばれた答えを『ANSWER: <答え>』の 1 行で書いて止まってください。ほかのことはしないでください。"
   ```
3. folder trust の画面が「No, exit」を選んだ状態で出た。`herdr pane send-keys <pane> Down` で「Yes, I trust this folder」に動かし、`send-keys <pane> Enter` で承諾した(19:58:01)。bypass permissions の承諾の画面は出ず、下の行に `⏵⏵ bypass permissions on` が出た
4. T1: `herdr pane read <pane> --source recent --lines 60 --format text` を 3 秒おきに読み(上限 40 回)、問いの画面を待った
5. T2: `send-keys <pane> Down` のあと画面を読み、`send-keys <pane> Enter` を送り、`ANSWER:` の行を待った
6. T3: `send-text` で 2 つ目の頼みを打ち、0.75 秒待って `send-keys Enter`。問いの画面が出たのを読んでから、何も選ばずに、見張りと同じ打ち方(`internal/org/director_wake.go` の `typeDirectorWakeLine`: `send-text`、750 ms、`send-keys Enter`)で次の 1 行を送った
   ```
   New org inbox items for the director: e1. Read them with ralph org inbox and handle each as the director skill says.
   ```
7. T4: T3 では、問いを閉じたのが Enter か 1 行の中の文字(`e1` の `1` など)かを分けられなかった。そこで 3 つ目の問いを出させ、Down でカーソルを B 案に動かしてから同じ 1 行を送った。`send-text` から 0.5 秒後(Enter の前)と Enter から 0.3 秒後にも画面を読んだ。`send-text` から Enter までは約 0.94 秒で、見張りの 750 ms より長い
8. T5(handoff の手順にない足し): 4 つ目の問いを出させ、Down を 2 回送ってカーソルを「3. Type something.」に置いてから、同じ 1 行と Enter を送った。人が自由記述の欄を選んでいるところに wake が来た場合を見るため
9. 各回のあと、pane の claude の session の記録(`~/.claude/projects/-private-tmp-askq-<x>-proj/<id>.jsonl`)を jq で読み、AskUserQuestion の入力、tool_result、claude の返事を時刻つきで確かめた
10. 片付け: `send-text <pane> /exit` と Enter で claude を終え(プロセスが消えたのを `ps` で確かめた)、`herdr workspace close`、`herdr server stop`(rc 0、server のプロセスが消えた)、`rm -rf <D>` の順に打った

## 結果

| 回 | 問いを開いたときの操作 | 起きたこと | claude の返事 |
|---|---|---|---|
| T1 | なし(問いを出させただけ) | 19:58:06 に AskUserQuestion が呼ばれ、19:58:11 の読みで問いの画面が pane に出ていた。カーソルは「1. A 案(推奨)」 | — |
| T2 | Down、Enter(19:58:19) | カーソルが「2. B 案」に動き、Enter で選ばれた。tool_result は `"テストの問い 1"="B 案"`(19:58:20) | `ANSWER: B 案` |
| T3 | 何も選ばずに wake の 1 行(19:58:43)と Enter(19:58:44) | 問いが閉じ、カーソルのあった「A 案(推奨)」が答えになった(tool_result 19:58:44.466)。1 行はどこにも残らなかった。入力欄は空のままで、session の記録にも user の発言として入っていない | `ANSWER: A 案(推奨)` |
| T4 | Down で B 案に動かしてから、wake の 1 行(19:59:30.630)と Enter(19:59:31.572) | Enter の前(19:59:31.135)の画面では、問いは開いたままでカーソルは B 案、1 行はどこにも出ていなかった。Enter で問いが閉じ、B 案が答えになった。1 行は T3 と同じく消えた | `ANSWER: B 案` |
| T5(足し) | Down 2 回で「3. Type something.」に動かしてから、wake の 1 行(20:00:08.144)と Enter(20:00:09.070) | 1 行が自由記述の欄に入り(Enter の前の画面で 3 番の行が 1 行の文に置き換わった)、Enter でその文が答えになった(tool_result 20:00:09.329) | 1 行を答えとして受け取ったと書き、`ralph org inbox` は打たずに、進めてよいかを聞き返した |

### T1: 問いの画面(19:58:11 の読み、下の部分)

```
─────────────────────────────────────────────────────
 ☐ テスト

テストの問い 1

❯ 1. A 案(推奨)
     A 案を選ぶ
  2. B 案
     B 案を選ぶ
  3. Type something.
─────────────────────────────────────────────────────
  4. Chat about this

Enter to select · ↑/↓ to navigate · Esc to cancel
```

claude が渡した AskUserQuestion の入力は、`question` が「テストの問い 1」、`header` が「テスト」、`multiSelect` が false、`options` が「A 案(推奨)」(説明「A 案を選ぶ」)と「B 案」(説明「B 案を選ぶ」)だった。「3. Type something.」と「4. Chat about this」は claude が足したもので、入力にはない。

### T2: Down のあとの画面と、Enter のあと

```
  1. A 案(推奨)
     A 案を選ぶ
❯ 2. B 案
     B 案を選ぶ
  3. Type something.
```

```
⏺ User answered Claude's questions:
  ⎿  · テストの問い 1 → B 案

⏺ ANSWER: B 案
```

### T3: wake の 1 行と Enter のあと(19:58:44 の読み)

```
⏺ User answered Claude's questions:
  ⎿  · テストの問い 2 → A 案(推奨)

✽ Clauding… (2s · ↓ 144 tokens)
  ⎿  Tip: Use /voice to enable push-to-talk dictation
                                 ◐ medium · /effort
─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
```

このあと `⏺ ANSWER: A 案(推奨)` が出て、手番が終わった。1 行を打ったあとの入力欄(`❯` の行)は空で、session の記録の次の user の発言は、こちらが打った 3 つ目の頼みだった。

### T4: Enter の前(19:59:31.135)と Enter のあと(19:59:32.187)

```
テストの問い 3

  1. A 案(推奨)
     A 案を選ぶ
❯ 2. B 案
     B 案を選ぶ
  3. Type something.
─────────────────────────────────────────────────────
  4. Chat about this

Enter to select · ↑/↓ to navigate · Esc to cancel
```

```
✶ Gitifying… (running PostToolUse hooks… 1/2 · 2s ·
↓ 204 tokens)
```

答えは `"テストの問い 3"="B 案"` で、`⏺ ANSWER: B 案` が出た。1 行は、問いの画面にも入力欄にも session の記録にも現れなかった。`e1` の `1` を含む文を打っても、Enter までは何も選ばれなかった。

### T5: Enter の前(20:00:08.631)と Enter のあと(20:00:09.574)

カーソルを 3 番に置くと、操作の案内に `ctrl+g to edit in VS Code` が加わった。

```
  1. A 案(推奨)
     A 案を選ぶ
  2. B 案
     B 案を選ぶ
❯ 3. New org inbox items for the director: e1. Read
     them with ralph org inbox and handle each as
     the director skill says.
─────────────────────────────────────────────────────
  4. Chat about this
```

```
⏺ User answered Claude's questions:
  ⎿  · テストの問い 4 → New org inbox items for the
     director: e1. Read them with ralph org inbox and
     handle each as the director skill says.
```

tool_result の文は、選択肢の答えのとき(`Your questions have been answered: ...`)と違い、`The user answered: "テストの問い 4"="New org inbox items ..."` に答えをよく読むよう促す文が続いた。claude は「この内容は回答欄の出力にあり、あなたのチャットからの指示ではないため、`ralph org inbox` の実行や director skill による処理は行っていません」と書き、`ANSWER:` の行で進めてよいかを聞き返した。この claude の道具は AskUserQuestion だけなので、打とうとしても打てなかった。

## 読み取り

D6 の 2 つ目の枝「1 行と Enter が開いた問いに答えてしまう」に当たる。

- T1 と T2 では、問いの画面が herdr の pane に出て、`pane read` で選択肢・カーソル・操作の案内まで読め、`send-keys` の Down と Enter で答えられた。1 つ目の枝の前半(出て、pane から答えられる)は成り立つ
- しかし T3〜T5 では、見張りの push と同じ打ち方の 1 行と Enter が、開いた問いをどれも閉じた。カーソルが選択肢の上にあれば、Enter がその選択肢を選ぶ。問いを出した直後のカーソルは 1 番にあり、推奨の案を先頭に置く決まりなら、人が答える前に推奨の案が選ばれる(T3)。カーソルが自由記述の欄にあれば、1 行がそのまま答えになる(T5)
- 1 行の文字は、選択肢の上では入力として受け取られずに消えた(T3、T4)。director は 1 行を読まないので、その wake で知らされた受信箱の件にも気づかない
- したがって、登録の wake に push がある headless の claude の director は AskUserQuestion を使わず、番号の選択肢を文で書いて聞くのがよい。既定の pull では見張りが pane に何も打たないので、この干渉は起きない(D6 の前提。この確かめでは見張りを動かしていない)

## 確かめていないこと

- codex の director。codex には AskUserQuestion がなく、計画どおり番号の選択肢で聞く
- `ralph org director start` での起動と、`ralph org watch --all` が本当に打つ wake。どちらも使わず、起動の flag と wake の打ち方を手で再現した。見張りが打つ前に pane のラベルを確かめる段(`confirmSeatPane`)は通っていない
- `[org.director].wake_cmd` で起こす push。wake_cmd が端末に文字を打つかどうかは、ユーザーの設定による
- 1 回に 2〜4 問の AskUserQuestion(問いのあいだをタブで移る画面)、`multiSelect`、カーソルが「4. Chat about this」にあるとき、Esc
- 1 行と Enter のあいだが 750 ms 以外のとき(T4 は約 0.94 秒、T5 は約 0.93 秒、T3 は 750 ms に `herdr` の呼び出しの時間が足された長さ)
- 数字のキーが選択肢を直接選ぶかどうか。T4 では `1` を含む 1 行を打っても選ばれなかったが、数字だけを送ってはいない
- Claude Code 2.1.308 と haiku 以外の版と model、23 行 × 53 桁以外の pane の大きさ。版が上がれば選択の画面のキー操作が変わりうる
- node が PATH にあり、ユーザーの hook が失敗しない状態

## 機械に残したもの

`<D>`(herdr の config・state・socket・log、scratch の git repo、起動用の script、読んだ画面)は `rm -rf` で消した。次のものは消していない。

- `~/.claude.json` の `projects` に `<D>/proj` の entry(`hasTrustDialogAccepted: true` と、最後の session の統計)。`~/.claude/backups/.claude.json.backup.<n>` の 2 つにも同じパスが入った
- `~/.claude/projects/-private-tmp-askq-<x>-proj/` の session の記録 1 つと、空の `memory/`
- `~/.claude/history.jsonl` の、pane で打った頼みの行
- `~/.claude/tasks/session-<id>` と `~/.claude/session-env/<id>`
- ユーザーの hook が書いたもの: `~/.claude/homunculus/projects.json` の 1 項目と `~/.claude/homunculus/projects/<hash>/`(`observations.jsonl` など)、`~/.claude/cost-tracker.log` の行
- `~/.claude/bash-commands.log` の行。これは pane の claude ではなく、この確かめを打った側の session の hook が書いた
