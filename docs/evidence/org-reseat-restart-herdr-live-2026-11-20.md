# reseat --restart-herdr の部品の実機の確かめ(2026-11-20)

計画 `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md` の Test plan の「実機の確かめ」の記録。このマシンの本物の herdr のサーバー(pid 59855)には別の repo の org の pane があるので止めず、`docs/recipes/isolated-herdr-server.md` の決まり(`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` を `/tmp/rrh` の下へ、HOME は本物、`env -i` で起動)で分離したサーバーを使った。座席は立てていない。

## 環境

- macOS 26.5.2、herdr 0.7.5(`/opt/homebrew/bin/herdr`)
- 分離したサーバーの起動: `env -i HOME="$HOME" PATH="$PATH" SHELL=/bin/bash TERM=xterm-256color XDG_CONFIG_HOME=/tmp/rrh/config XDG_STATE_HOME=/tmp/rrh/state HERDR_SOCKET_PATH=/tmp/rrh/h.sock CLAUDE_CONFIG_DIR=/tmp/rrh/claude-b nohup herdr server`(`CLAUDE_CONFIG_DIR` は確かめ用の偽のディレクトリ)
- 以下の herdr の命令は、どれも 3 つの変数を付けて分離したサーバーに打った

## 確かめたこと

1. 起動の前の `pgrep -f '/opt/homebrew/bin/herdr server'` は本物のサーバーの 59855 だけ
2. `herdr status server --json` は `running: true`、`socket: /tmp/rrh/h.sock`
3. `herdr workspace create --cwd /tmp/rrh --label rrh-shell --no-focus` で shell の pane を 1 つ作り、`herdr pane list`(`--workspace` なし)は pane を 1 つ返した: `w1:p1`、label なし(workspace の label は pane の label ではない)、cwd `/private/tmp/rrh`。ドライバの `PaneListAll` が読む形と同じ
4. `lsof -t /tmp/rrh/h.sock` は pid を 1 つ(78118)返し、`ps -o comm= -p 78118` は `herdr`(本物のサーバーでは `/opt/homebrew/bin/herdr` とパスで返った。どちらも末尾が `herdr`)
5. `ps eww -o command= -p 78118` から `CLAUDE_CONFIG_DIR=/tmp/rrh/claude-b`、`XDG_STATE_HOME=/tmp/rrh/state`、`HERDR_SOCKET_PATH=/tmp/rrh/h.sock` が読めた
6. `herdr server stop` は標準出力に何も出さずに終わった(ドライバの `ServerStop` が成功と読む形)。1 秒後の `herdr status server --json` は `{"status":"not_running","running":false,…,"socket":"/tmp/rrh/h.sock",…}`。pid 78118 は消え、ソケットのファイルもなくなった
7. 終わりの `pgrep` も本物のサーバーの 59855 だけ。本物のサーバーには触れていない

## 確かめていないこと

- `ralph org reseat --restart-herdr` を、座席のある org で通しで打つこと(座席は立てていない。順と断る形は fake のテストで確かめた)
- 止めたあとゲートが新しい `CLAUDE_CONFIG_DIR` で起動したサーバーで、claude の座席が新しいアカウントの資格で動くこと(2 つのアカウントを持つ人の手でしか確かめられない)
