# herdr のサーバーの自動起動の実機確認

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-autostart.md`(S5、AC1・AC2・AC3・AC4)
- Spec: `docs/specs/2026-11-06-org-herdr-autostart-panes.md`(FR-A1〜A5)
- 生の出力は分離に使った一時ディレクトリに置き、確かめたあとディレクトリごと消した。要点はこのファイルに写した。ホームディレクトリは `~`、分離に使った一時ディレクトリは `<D>`(`/tmp/ralph-iso`)と書く。

## 環境

| 項目 | 値 |
|---|---|
| マシン | macOS(Darwin 25.5.0) |
| herdr | 0.7.5。`<D>` の下に `XDG_CONFIG_HOME`・`XDG_STATE_HOME`・`HERDR_SOCKET_PATH` を向けて分離した(`docs/recipes/isolated-herdr-server.md`) |
| ralph | このブランチの 11c64fb1 を `go build -o <D>/bin/ralph ./cmd/ralph` で作った |
| repo | `<D>/work`。`git init` した scratch の repo で、`ralph.toml` は `[org].agmsg_home = "<D>/no-agmsg"`(存在しない場所。spawn が agmsg に進んでもユーザーの agmsg に書かないため) |
| 座席 | `--driver claude --model fable --role worker`。分離した環境の PATH に `claude`・`codex`・`agmsg` は無いので、spawn は herdr の `agent_start` の段で必ず失敗する。agent を起動せずに、そこまでの herdr の呼び出しを確かめるための設定 |

wrapper `<D>/run` は `env -i` で次だけを渡した。最後の 4 つは、落とす変数と残す変数を確かめるために足したダミーである。

```sh
HOME USER LOGNAME SHELL=/bin/bash LANG TERM=xterm-256color TMPDIR
PATH="<D>/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"
XDG_CONFIG_HOME="<D>/xc" XDG_STATE_HOME="<D>/xs" HERDR_SOCKET_PATH="<D>/h.sock"
CLAUDECODE=1 CLAUDE_CODE_PROBE=dummy HERDR_TAB_ID=w9:t9 CODEX_PROBE=keep
```

ユーザーの herdr のサーバー(pid 22408)にはどの段でも触れていない。確かめの前と後の `pgrep -fl 'herdr server'` は、どちらも 22408 だけを出した。

## 手順と結果

見張りは `<D>/run ralph org watch --all --detach` で起動した(pid 73617)。どの spawn も `<D>/run ralph org spawn --org-id iso1 --id <seat> --role worker --driver claude --model fable --cwd <D>/work --scope 'live check' --timeout-ms 8000` で打った。

### A. サーバーが止まっていて、ソケットのファイルも無い(AC1)

打つ前の `herdr status server --json` は `{"status":"not_running","running":false,…}` だった。seat `s1` の spawn の stderr は次の 2 行で、終了コードは 1 だった。

```
ralph org spawn: started the herdr server (pid 75330, socket /tmp/ralph-iso/h.sock); run 'herdr' in another terminal to see the seats' panes and to answer the first seat's prompts; log: /private/tmp/ralph-iso/work/.harness/state/org/herdr-launch.log
org: spawn step agent_start failed: herdr: timed out: context deadline exceeded (the spawn's --timeout-ms deadline of 8000 ms, shared by every herdr and agmsg step, ran out; retry with a larger --timeout-ms)
```

- 1 行目がゲートの起動の行で、2 行目は claude が無いための想定どおりの失敗である。
- pid 75330 は `/opt/homebrew/bin/herdr server` で、親は 1 だった(切り離されている)。`lsof` で見た開いているファイルは `<D>` の下のもの(`herdr-launch.log`、`<D>/xc/herdr/herdr-server.log`、`<D>/h.sock`、`<D>/h-client.sock`)だけだった。
- `herdr-launch.log` には herdr の起動の 4 行(`herdr server running; …`、2 つのソケット、ログの場所)と TUI の案内の 1 行だけがあった。環境の値は書かれていない。
- 台帳には `org_workspace_created` があり、herdr には label `iso1` のワークスペース w1 ができた。spawn の補償で座席のタブは閉じ、タブ「1」(仕様の B で直すもの)だけが残った。

### 起動したサーバーの環境(AC2)

座席が受け継ぐのはペインの環境なので、w1 の残ったペイン w1:p1 で `env` を打って読んだ。macOS の `ps -wwE` では、ほかのプロセスの環境は読めなかった。

| 変数 | wrapper(ralph の環境) | ペイン |
|---|---|---|
| `CLAUDECODE` | `1` | 無い |
| `CLAUDE_CODE_PROBE` | `dummy` | 無い |
| `HERDR_TAB_ID` | `w9:t9` | `w1:t1`(herdr がペインに付ける値。ralph の値は落ちている) |
| `CODEX_PROBE` | `keep` | `keep` |
| `XDG_CONFIG_HOME` | `<D>/xc` | `<D>/xc` |
| `XDG_STATE_HOME` | `<D>/xs` | `<D>/xs` |
| `HERDR_SOCKET_PATH` | `<D>/h.sock` | `<D>/h.sock` |

### C. サーバーが動いている(AC3)

A のあと、同じサーバーが動いているところで seat `s2` を spawn した。stderr に `started the herdr server` の行は無く(`grep -c` で 0)、失敗は A と同じ `agent_start` の段だった。打ったあとの `pgrep -fl 'herdr server'` は 22408 と 75330 だけで、新しいサーバーは起動していない。

### B. サーバーが落ちてソケットのファイルが残っている(AC1、事案と同じ状態)

`kill -9 75330` のあと、`<D>/h.sock` は残り、`herdr workspace list` は `Error: Os { code: 61, kind: ConnectionRefused, message: "Connection refused" }` で終わった。2026-11-05 の事案と同じ状態である。seat `s3` の spawn は、stderr に `ralph org spawn: started the herdr server (pid 89208, socket /tmp/ralph-iso/h.sock); …` を出して、A と同じ `agent_start` の段まで進んだ。起動し直したサーバーは session.json から w1(label `iso1`)を戻し、spawn はそこにタブを作った(台帳に新しい `org_workspace_created` は無い)。

### D. herdr が PATH に無い(AC4)

wrapper の PATH から `/opt/homebrew/bin` を外して seat `s4` を spawn した。終了コードは 1、stderr は次の 1 行だった。

```
org: spawn: herdr is not on PATH (herdr status server: herdr: exec: "herdr": executable file not found in $PATH), so ralph cannot start the herdr server, and nothing was made; install herdr, or start it in another terminal with 'herdr', then run ralph org spawn again
```

台帳 `manifest.jsonl` は打つ前と後でどちらも 10 行で、`s4` の記録は無かった。

## 台帳の記録

確かめのあとの `manifest.jsonl` の出来事は次の 10 件だった(`seat_id` と `details` の先頭)。

```
spawn_started s1
org_workspace_created -
spawn_step s1 tab_created
spawn_failed s1 step=agent_start error=herdr: timed out: context deadline ex…
spawn_started s2
spawn_step s2 tab_created
spawn_failed s2 step=agent_start …
spawn_started s3
spawn_step s3 tab_created
spawn_failed s3 step=agent_start …
```

## 確かめていないこと

- `ralph org start --plan` と `ralph org director start` を実機で打っていない。どちらも同じゲートを通る(`wireOrgHerdrServerGate`)ことと、呼ぶ位置(start --plan は worktree の前、director start は残った登録の確認の前)は、CLI と org のテストで確かめた。実機で打つと、leader か director の agent が起動して働き始めるためである。
- 2 つの ralph が同時にサーバーを起動しようとする場合(AC5)は、テストの偽物で確かめた。2 つ目の `herdr server` が `error: herdr server is already running` で終了コード 1 になることは、仕様を書くときに実機で確かめてある(仕様の Research findings)。
- `status server --json` を知らない古い herdr(D3 の例外)は、手元に無いので実機では確かめていない。
- 座席の agent が実際に起動するところまでは確かめていない(PATH に `claude` を置かなかった)。

## 片付け

`<D>/run ralph org disband --org-id iso1`(`disbanded org "iso1"`)、`<D>/run ralph org watch --stop`(`stopped ralph org watch --all (pid 73617)`)、`<D>/run herdr server stop` を打った。そのあとの `herdr workspace list` は空で、`pgrep -fl 'herdr server'` はユーザーの 22408 だけを出し、`lsof +D <D>` は何も出さなかった。最後に `<D>` を消した。
