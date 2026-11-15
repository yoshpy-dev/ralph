# `ralph org reseat` の実機の確認(2026-11-12)

計画 `docs/plans/active/2026-11-12-org-reseat.md` の S5(AC-12)の記録。分離した
herdr の上に機能の org を 1 つ立て、`ralph org reseat` を 2 回打った。1 回目は
座席が動いているとき(AC-1)、2 回目は herdr server を止めて立て直したあと
(AC-8)である。

結果を先に書く。

- AC-1(reseat): leader と reviewer の 2 席の org で、reseat は reviewer、leader の
  順に止め、leader、reviewer の順に立てた(終了コード 0、15 秒)。leader は新しい
  workspace の root の pane(左半分)に入った。`scope_reserved` は 1 件のまま
  増えず、止めた座席の `stopped` と立てた座席の `spawn_started`・`spawned` に
  `reseat=<id>` が付き、org の `reseated` が 1 件書かれた。implementer の座席と、
  implementer の spawn と TASK が承認の検査を通ることは、実機では確かめていない
  (下の「確かめていないこと」)。
- AC-8(server を立て直したあとの reseat): herdr 0.7.5 は、止める前に保存した
  `session.json` から workspace と 2 つの pane の記録を戻したが、pane に端末も
  プロセスもなかった。reseat はその 2 つの pane を閉じて止め(`pane=closed`。
  Ctrl-C は `pane_not_found` で失敗と記録)、新しい workspace に 2 席を立て直した
  (終了コード 0、16 秒)。
- 開いた問い(agmsg の team): 全座席が Leave しても team は消えなかった。
  `team_id` は最初の作成から disband のあとまで同じで、名簿と 5 件のメッセージの
  履歴も残った。
- 片付けのあと、ほかの herdr server(pid 34407)は同じ pid と起動時刻のまま動いて
  いた。専用のディレクトリは消した。

## 環境と分離

| 項目 | 値 |
|---|---|
| マシン | macOS 26.5.2(Darwin 25.5.0) |
| ralph | タスクの worktree の `dd2043706b5de5d820d1f717f818f17db30ca793` を `go build -o /tmp/ralph-rs/bin/ralph ./cmd/ralph` で作った。`ralph version` は `ralph dev (unknown unknown)` |
| herdr | 0.7.5(Homebrew) |
| Claude Code | 2.1.316 |
| agmsg | 1.5.1。`~/.agents/skills/agmsg` から `scripts`・`plugins`・`types`・`SKILL.md`・`VERSION` を `/tmp/ralph-rs/agmsg` に写した |
| Go | 1.27.3 |
| 座席 | claude。既定の `[org.role_models]` で leader も reviewer も `opus` |

ビルドのあと、worktree の HEAD は `8e4efee7` に進んだ。`dd204370..8e4efee7` で
変わったのは skill の文書 8 つ(`templates/base/` の写しを含む)と計画だけで、
Go のコードは変わっていない。`templates/` は go:embed で binary に入るので、
binary の違いは埋め込まれた skill の文面だけで、reseat の動きには関わらない。

分離は `docs/recipes/isolated-herdr-server.md` の節の順に行った。

- 専用のディレクトリは `/tmp/ralph-rs`(`mkdir -m 700`)。
- ラッパー `/tmp/ralph-rs/run` は `env -i` で、`HOME`(実際のもの)、`USER`、
  `LOGNAME`、`SHELL=/bin/bash`、`LANG`、`TERM=xterm-256color`、`TMPDIR`、`PATH`、
  `XDG_CONFIG_HOME` と `XDG_STATE_HOME`(`/tmp/ralph-rs` の下)、
  `HERDR_SOCKET_PATH=/tmp/ralph-rs/h.sock`、`CLAUDE_ENV_FILE`、
  `GIT_CONFIG_GLOBAL`(git の設定は `~/.config/git/config` にある)、
  `GH_CONFIG_DIR`(`~/.config/gh`)だけを渡した。`PATH` は `/tmp/ralph-rs/bin`、
  claude の実体のディレクトリ、node のディレクトリ、`/opt/homebrew/bin`、システムの
  ディレクトリの順。`ralph` と `herdr` のコマンドはすべてこのラッパーを通した。
- `/tmp/ralph-rs/claude-env` に同じ `PATH` を書き、`osascript` の shim を
  `/tmp/ralph-rs/bin` に置いた。shim は最後まで 1 度も呼ばれなかった。
- `herdr --help` の `Config:` と `Logs:` はどちらも `/tmp/ralph-rs/xdg-config/herdr/`
  の下を指した。
- 始める前の `pgrep -fl 'herdr server'` は `34407 /opt/homebrew/bin/herdr server`
  だけを出した(起動は 2026-11-11 08:14:35)。この server にはどのコマンドも
  送っていない。
- 専用の server は `nohup /tmp/ralph-rs/run herdr server &` で起動した(pid 35350)。
  `lsof -p` で開いていたのは、作業ディレクトリ `/private/tmp/ralph-rs`、
  `herdr-server.out`、`xdg-config/herdr/herdr-server.log`、unix ソケット 2 つ、
  バイナリ、`/dev/null` だけだった。`ps eww` の変数名はラッパーが渡したものだけ
  だった。

### 台帳を持つ repo

recipe の scratch の repo の形にした(dedicated clone ではない。「recipe と違った
ところ」の 1)。

- `/tmp/ralph-rs/work` に `git init -b main`。repo の `user.name` と `user.email`、
  空のディレクトリを指す `core.hooksPath`。
- タスクの worktree から `scripts/ralph-worktree.sh`、`scripts/ralph-common.sh`、
  `scripts/plan-visual.sh` を写し、base のコミットに入れた。
- `.gitignore` に `.harness/state/`、`.harness/logs/`、`.claude/worktrees/`、
  `ralph.local.toml`。
- `ralph.toml` は `[org].agmsg_home = "/tmp/ralph-rs/agmsg"` だけ。承認方式は
  既定の `per-plan`、`org.review.plan_advisory` は既定の `false`。
- remote は 1 つもない(`git remote` は空)。

### 信頼の画面と、捨てる pane での確かめ

捨てる workspace の pane で `type claude ralph git jq; alias` を打つと、`ralph` は
`/tmp/ralph-rs/bin/ralph`、`claude` は claude の実体を指し、`alias` は何も出さな
かった。pane の shell の `PATH` は、`path_helper` によってシステムのディレクトリが
前に来ていた(F2 のとおり)。`claude --model haiku` を起動すると
`/private/tmp/ralph-rs/work` の信頼の画面が出たので、Down と Enter で答えた。
claude の Bash ツールでの `command -v git ralph claude osascript jq` は、ralph と
osascript について `/tmp/ralph-rs/bin` の写しを出した。そのあと `/exit` で抜け、
workspace を閉じた。

### ralph doctor と ralph status

`ralph doctor` は "All checks passed."。警告は codex が PATH にないこと(この確認
では使わない)、`.codex/config.toml`・`settings.json`・manifest が scratch の repo に
ないことの 4 つだった。「Shell aliases」は `~/.zshrc` の `claude` の alias を
info で挙げたが、pane の shell は bash なので届かない。`ralph status` の台帳は
`/private/tmp/ralph-rs/work/.harness/state/org (source: git-main-worktree)` で、
見張りは `org watch: not running` だった。

## 機能の org を立てる

分割計画 `/tmp/ralph-rs/work/.harness/state/org/splits/rs-check.md` を書いた。
前文は `## Objective` の 1 節、機能は `rs-demo` の 1 つ(`- Type: docs`、
`- Reserve: docs/rs-demo/`、`- Depends on: none`)。本文で leader に、10 行ほどの
計画を書いてコミットしたら、図解ページ・計画助言・承認の QUESTION・座席の spawn・
TASK の送信をせずに待つよう頼んだ。`./scripts/plan-visual.sh digest` が出した
`a93984d4c2fe` を `- Approved: 2026-11-12 sha256:a93984d4c2fe` に書いた。

```
$ /tmp/ralph-rs/run ralph org start --plan .harness/state/org/splits/rs-check.md --feature rs-demo
ralph org start: started the cross-org watch (pid 55849); log: /private/tmp/ralph-rs/work/.harness/state/org/watch.log
spawned seat "leader" (org_id=rs-demo driver=claude model=opus pane_id=w2:p1 dry_run=false)
worktree: /private/tmp/ralph-rs/work/.claude/worktrees/org-rs-demo
branch: docs/rs-demo
hint: ralph org status --org-id rs-demo ; attach with herdr to observe the leader pane
```

leader は約 30 秒で計画 `docs/plans/active/2026-11-12-rs-demo.md` をコミット
(`76fd2f4`)して手番を終えた。その途中で leader は自分から
`git push -u origin docs/rs-demo` を打ったが、remote がないので失敗した。push は
どこにも届いていない。leader は reviewer を立てなかったので、leader が立てるのと
同じ形で reviewer を立てた。

```
$ /tmp/ralph-rs/run ralph org spawn --org-id rs-demo --id reviewer --role reviewer --cwd /private/tmp/ralph-rs/work/.claude/worktrees/org-rs-demo --scope docs/rs-demo/
spawned seat "reviewer" (org_id=rs-demo driver=claude model=opus pane_id=w2:p2 dry_run=false)
```

reviewer は役割の指示を読み、TASK を待つと書いて手番を終えた。reseat の前の状態は
次のとおり。

```
SEAT_ID	ROLE	DRIVER	MODEL	STATE	PANE_ID
leader	leader	claude	opus	spawned (active)	w2:p1
reviewer	reviewer	claude	opus	spawned (active)	w2:p2
reserved: docs/rs-demo/
feature: rs-check/rs-demo branch docs/rs-demo worktree /private/tmp/ralph-rs/work/.claude/worktrees/org-rs-demo
```

manifest は 14 行で、`scope_reserved` は 1 件。herdr の workspace は `w2`
(label `rs-demo`)の 1 つで、pane は `w2:p1`(leader)と `w2:p2`(reviewer)だった。

## AC-1: 座席が動いているときの reseat

### --dry-run

```
$ /tmp/ralph-rs/run ralph org reseat --org-id rs-demo --dry-run
stop: reviewer (reviewer, claude opus)
stop: leader (leader, claude opus)
start: leader (leader, claude opus)
start: reviewer (reviewer, claude opus)
rc=0
```

manifest は打つ前も後も 14 行で、sha256 の先頭 16 桁も同じ
(`5305742ffc2e57a1`)だった。台帳のディレクトリに `reseat.lock` は作られなかった。

### reseat

```
$ /tmp/ralph-rs/run ralph org reseat --org-id rs-demo      (13:43:57Z)
stopped: reviewer (reviewer, claude opus)
stopped: leader (leader, claude opus)
started: leader (leader, claude opus, pane w3:p1)
started: reviewer (reviewer, claude opus, pane w3:p2)
reseated org_id rs-demo (reseat=20261112T134358-6072)
rc=0                                                       (13:44:12Z)
```

```
$ /tmp/ralph-rs/run ralph org status --org-id rs-demo
SEAT_ID	ROLE	DRIVER	MODEL	STATE	PANE_ID
leader	leader	claude	opus	spawned (active)	w3:p1
reviewer	reviewer	claude	opus	spawned (active)	w3:p2
reserved: docs/rs-demo/
feature: rs-check/rs-demo branch docs/rs-demo worktree /private/tmp/ralph-rs/work/.claude/worktrees/org-rs-demo
```

manifest は 31 行になり、`scope_reserved` は 1 件のままだった。足された 17 行の
うち、`spawn_step` を除いた行は次のとおり(`{ts,event,seat_id,pane_id,details}` に
絞った)。

```
{"ts":"2026-11-12T13:43:59Z","event":"stopped","seat_id":"reviewer","pane_id":"w2:p2","details":"ctrl_c=ok pane=closed leave=ok reseat=20261112T134358-6072"}
{"ts":"2026-11-12T13:44:00Z","event":"stopped","seat_id":"leader","pane_id":"w2:p1","details":"ctrl_c=ok pane=closed leave=ok reseat=20261112T134358-6072"}
{"ts":"2026-11-12T13:44:00Z","event":"spawn_started","seat_id":"leader","pane_id":null,"details":"reseat=20261112T134358-6072"}
{"ts":"2026-11-12T13:44:01Z","event":"org_workspace_closed","seat_id":"","pane_id":"w2","details":"not found at spawn"}
{"ts":"2026-11-12T13:44:01Z","event":"org_workspace_created","seat_id":"","pane_id":"w3","details":null}
{"ts":"2026-11-12T13:44:06Z","event":"spawned","seat_id":"leader","pane_id":"w3:p1","details":"scope=split rs-check feature rs-demo (reserve: docs/rs-demo/) permission_mode=autonomous reseat=20261112T134358-6072"}
{"ts":"2026-11-12T13:44:06Z","event":"spawn_started","seat_id":"reviewer","pane_id":null,"details":"reseat=20261112T134358-6072"}
{"ts":"2026-11-12T13:44:12Z","event":"spawned","seat_id":"reviewer","pane_id":"w3:p2","details":"scope=docs/rs-demo/ permission_mode=autonomous reseat=20261112T134358-6072"}
{"ts":"2026-11-12T13:44:12Z","event":"reseated","seat_id":"","pane_id":null,"details":"id=20261112T134358-6072 seats=leader:started,reviewer:started"}
```

`spawn_step` は leader が `pane_created how=root tab=w3:t1`、`agent_started`、
`agmsg_joined leader_self=true`、reviewer が `pane_created how=split-right tab=w3:t1`、
`agent_started`、`agmsg_leader_joined ok`、`agmsg_joined`、`agmsg_announced` だった。
`model-receipts.jsonl` には 2 席ぶんの行(claude、commanded_model `opus`、
honored `unknown`)が足された。

### herdr の配置

`w2` は 2 つの pane が閉じたところで herdr が消し(`org_workspace_closed ... not found
at spawn`)、新しい workspace `w3`(label `rs-demo`、tab `w3:t1` の 1 つ)ができた。
`herdr pane layout --pane w3:p1` は次のとおりで、leader が左半分、reviewer が右半分
にいる。

```
{"panes":[{"pane_id":"w3:p1","rect":{"height":23,"width":27,"x":26,"y":1}},{"pane_id":"w3:p2","rect":{"height":23,"width":27,"x":53,"y":1}}],"splits":[{"direction":"right","ratio":0.5}]}
```

### 立て直した座席

立て直した leader は、計画がコミット `76fd2f4` で入っていて未コミットの変更が
ないことを確かめ、図解ページ・計画助言・QUESTION・spawn・TASK をせずに待つと書いて
手番を終えた。reviewer は役割の指示を読み直して TASK を待った。

### AC-1 の判定

2 席の org について合格。止める順(reviewer、leader)、立てる順(leader、reviewer)、
leader が root の pane に入ること、`scope_reserved` が増えないこと、`reseat=<id>` と
`reseated` の記録を確かめた。implementer を含む 3 席の場合と承認の検査は確かめて
いない。

## AC-8: herdr server を立て直したあとの reseat

### server を止めて立て直す

止める前の `session.json` には workspace `w3` が 1 つあり、pane 2 つのラベル、
agent 名(`rs-demo_leader`、`rs-demo_reviewer`)、cwd、claude のセッションの id が
入っていた。次の再起動で何が戻るかを見るため、このファイルは消さずに残した
(「recipe と違ったところ」の 2)。

```
$ /tmp/ralph-rs/run herdr server stop          (13:45:07Z)
rc=0
$ pgrep -fl 'herdr server'
34407 /opt/homebrew/bin/herdr server
```

`h.sock` と `h-client.sock` は消え、座席の claude のプロセスも残っていなかった。
herdr の log には、2 つの pane の `pane session terminated ... signal=Kill` と、
その直前の `session saved ... workspaces=1` が出た。同じ形で server を起動し直した
(pid 8430)。

```
$ nohup /tmp/ralph-rs/run herdr server > /tmp/ralph-rs/herdr-server2.out 2>&1 &   (13:45:20Z)
$ pgrep -fl 'herdr server'
8430 herdr server
34407 /opt/homebrew/bin/herdr server
```

### 戻った pane

herdr の log は `session restore evaluated ... workspaces=1` を出した。戻ったものは
次のとおり。

- `herdr workspace list` は `w3`(label `rs-demo`)を出し、`herdr pane list` は
  `w3:p1`(label `leader`)と `w3:p2`(label `reviewer`)を、agent `claude`、
  `agent_status` `idle`、もとの cwd で出した。`herdr pane get` も同じ内容を返し、
  `herdr pane layout` は左右に半分ずつの配置を返した。
- 一方、`herdr pane read` と `herdr pane process-info` はどちらの pane にも
  `pane_not_found` を返した。server(pid 8430)に子プロセスはなく、log にもこの
  2 つの pane の `spawning pane terminal` はなかった。pane の記録だけが戻り、
  shell も claude も動いていない。
- この間 `ralph org status --org-id rs-demo` は、2 席とも `spawned (active)` と
  出し続けた。

### server が止まっていたあいだの見張り

見張り(pid 55849)は止めずにおいた。13:45:10 に 2 席の liveness の条件が立ち、
次のことが起きた。

- reviewer の ALERT は agmsg で leader に送られた(watchdog は 13:45:13 に team に
  入った)。本文は `herdr agent get failed for seat reviewer: herdr: exit status 1: ...`。
- leader の ALERT は leader 自身についてのものなので、受信箱の件 `e1`(TYPE `ALERT`、
  `REASON: leader_is_anomaly_subject`、notified `deferred`)になった。director は
  いないので、ack の期限まで人には送られない。`watch.log` には
  `WATCHDOG ESCALATION: org=rs-demo alert=rs-demo/leader/liveness@... item=e1 ...`
  の 1 行が出た。

server を立て直したあと、戻った pane を見張りが動いている座席とみなしたかどうかは
確かめていない。

### reseat

```
$ /tmp/ralph-rs/run ralph org reseat --org-id rs-demo --dry-run
stop: reviewer (reviewer, claude opus)
stop: leader (leader, claude opus)
start: leader (leader, claude opus)
start: reviewer (reviewer, claude opus)
rc=0
$ /tmp/ralph-rs/run ralph org reseat --org-id rs-demo      (13:45:53Z)
stopped: reviewer (reviewer, claude opus)
stopped: leader (leader, claude opus)
started: leader (leader, claude opus, pane w4:p1)
started: reviewer (reviewer, claude opus, pane w4:p2)
reseated org_id rs-demo (reseat=20261112T134553-ff12)
rc=0                                                       (13:46:09Z)
```

dry-run の前後で manifest は 31 行のままだった。reseat のあとは 48 行で、
`scope_reserved` は 1 件のまま。`spawn_step` を除いた行は次のとおり。

```
{"ts":"2026-11-12T13:45:54Z","event":"stopped","seat_id":"reviewer","pane_id":"w3:p2","details":"ctrl_c=failed: herdr: exit status 1: {\"error\":{\"code\":\"pane_not_found\",\"message\":\"pane w3:p2 not found\"},\"id\":\"cli:request\"} pane=closed leave=ok reseat=20261112T134553-ff12"}
{"ts":"2026-11-12T13:45:54Z","event":"stopped","seat_id":"leader","pane_id":"w3:p1","details":"ctrl_c=failed: herdr: exit status 1: {\"error\":{\"code\":\"pane_not_found\",\"message\":\"pane w3:p1 not found\"},\"id\":\"cli:request\"} pane=closed leave=ok reseat=20261112T134553-ff12"}
{"ts":"2026-11-12T13:45:55Z","event":"spawn_started","seat_id":"leader","pane_id":null,"details":"reseat=20261112T134553-ff12"}
{"ts":"2026-11-12T13:45:55Z","event":"org_workspace_closed","seat_id":"","pane_id":"w3","details":"not found at spawn"}
{"ts":"2026-11-12T13:45:55Z","event":"org_workspace_created","seat_id":"","pane_id":"w4","details":null}
{"ts":"2026-11-12T13:46:00Z","event":"spawned","seat_id":"leader","pane_id":"w4:p1","details":"scope=split rs-check feature rs-demo (reserve: docs/rs-demo/) permission_mode=autonomous reseat=20261112T134553-ff12"}
{"ts":"2026-11-12T13:46:00Z","event":"spawn_started","seat_id":"reviewer","pane_id":null,"details":"reseat=20261112T134553-ff12"}
{"ts":"2026-11-12T13:46:09Z","event":"spawned","seat_id":"reviewer","pane_id":"w4:p2","details":"scope=docs/rs-demo/ permission_mode=autonomous reseat=20261112T134553-ff12"}
{"ts":"2026-11-12T13:46:09Z","event":"reseated","seat_id":"","pane_id":null,"details":"id=20261112T134553-ff12 seats=leader:started,reviewer:started"}
```

herdr の log では、どちらの pane も `pane.send_keys` が error、続く `pane.close` が
ok で、2 つ目の close のところで `workspace closed ... workspace_id="w3"` が出た。
戻った pane は herdr の上では「無い」(`pane=already closed`)ではなく、閉じられる
pane として扱われ、Stop はそれを閉じた。Ctrl-C は送れなかったが、`stop_failed`
にはならなかった。

新しい workspace `w4` の配置は、`w4:p1`(leader)が左半分(x=26、幅 27)、
`w4:p2`(reviewer)が右半分(x=53、幅 27)、split は `right` の 0.5 で、1 回目と
同じだった。立て直した leader は計画がコミット済みで作業ツリーがきれいなことを
確かめて待ち、reviewer は TASK を待った。

### AC-8 の判定

合格。server を立て直して座席のプロセスがなくなったあとも、reseat は 2 席を
止めて立て直した。止めるときの記録は `pane=already closed` ではなく
`ctrl_c=failed: ... pane_not_found ... pane=closed` で、`details` に herdr の
エラーの JSON がそのまま入る。

## 開いた問い: 全座席が Leave したときの agmsg の team

private の写し `/tmp/ralph-rs/agmsg` の team `ralph-rs-demo` を見た。

- `teams/ralph-rs-demo/config.json` の `team_id` は、最初の作成(13:42:14)から
  disband のあとまで同じだった。
- `roster.jsonl` には、1 回目の reseat で reviewer と leader の `member_left`
  (13:43:58、13:44:00)、leader と reviewer の `member_joined`(13:44:05、13:44:11)、
  2 回目で同じ並び(13:45:54 に 2 つの `member_left`、13:45:59 と 13:46:08 に
  `member_joined`)が続いた。どちらの reseat でも leader が先に入り直した。
  disband のあとも team のディレクトリは残り、`config.json` の `agents` には
  leader、reviewer、watchdog が残った。
- `db/messages.db` の `messages` は 5 件で、すべて team `ralph-rs-demo` の
  leader 宛てだった。reviewer の HELLO が 3 件(13:43:06、13:44:12、13:46:08)、
  watchdog の ALERT が 2 件(13:45:14、13:45:15)。最初の HELLO も 2 回の reseat の
  あとまで残った。

全座席が抜けても team も履歴も消えない。仕様の Open questions の答えになる。
ユーザーの agmsg の `teams/` に `rs-demo` の team は作られなかった。

## 確かめていないこと

- implementer を含む 3 席の reseat と、implementer の spawn と implementer への TASK
  が承認の検査を通ること(AC-1 の後半、issue #312 の受け入れ条件 2)。機能の計画の
  承認(`ralph org approve`)をすると leader が implementer を立てて作業を始め、
  費用が増えるので、承認しなかった。テストの側で確かめる。
- `--seat` と `--driver`・`--model`、途中の失敗、`reseat.lock` の競合(AC-3、
  AC-10、AC-17、AC-18)。この確認の範囲の外。
- server を立て直したあと、戻った pane(プロセスなし、`agent_status` は `idle`)を
  見張りが動いている座席とみなすかどうか。
- 立て直した座席が受信箱のメッセージをどう扱うか。leader は待つよう頼まれていた
  ので、reviewer の HELLO と watchdog の ALERT を読んだかは見ていない
  (`read_at` は 5 件とも空)。

## recipe と違ったところ

1. dedicated clone ではなく scratch の repo を使い、`scripts/` の 3 つのファイルを
   写して base のコミットに入れた。PR を開かない確認なので recipe の scratch の形に
   合う。`ralph org start --plan` に要る `scripts/ralph-worktree.sh` を足した点だけが
   recipe の例と違う。
2. 2 回目の server の起動の前に `session.json` を消さなかった。「Running again」は
   新しい実行のための手順で、この確認は戻った pane を見ることが目的だった。
3. 機能の org の座席が報告を送る場面だが、private の agmsg の写しを使った(recipe は
   自分の agmsg を勧める)。座席は leader に報告を送る必要がなく、team の様子を
   ユーザーの agmsg に触れずに見たかったからである。
4. 捨てる pane での確かめで `gh auth status` を打たなかった(push も PR もしない)。
5. 証拠に herdr の pane の id(`w3:p1` など)を書いた。recipe は書かないよう言うが、
   leader が root の pane に入ったことを示すのに要る。どれも消した専用の server の
   id で、claude のセッションの id、gh や git の設定、トークンは書いていない。
6. leader が task の頼みの外で `git push -u origin docs/rs-demo` を打った。remote が
   ないので失敗し、どこにも届いていない。

## 片付け

1. `ralph org disband --org-id rs-demo` は `stopped seat "leader"`、
   `stopped seat "reviewer"`、`disbanded org "rs-demo"` を出した(どちらも
   `ctrl_c=ok pane=closed leave=ok`、`org_workspace_closed workspace=already closed`)。
   受信箱の `e1` は開いたまま、専用の台帳ごと消した。
2. `ralph org watch --all --stop` は `stopped ralph org watch --all (pid 55849)` を
   出し、`ralph status` は `org watch: not running (stopped at 2026-11-12T13:47:24Z)`
   を出した。`~` に移ってからの `pgrep -fl /tmp/ralph-rs` は何も出さず、
   `lsof +D /tmp/ralph-rs` は専用の server(pid 8430)だけを出した。
3. `/tmp/ralph-rs/run herdr server stop` のあと、`pgrep -fl 'herdr server'` は
   `34407 /opt/homebrew/bin/herdr server` だけを出した。`ps -o lstart` の起動時刻は
   始める前と同じ 2026-11-11 08:14:35。`h.sock` と `h-client.sock` は消え、
   `lsof +D /tmp/ralph-rs` は何も出さなかった。
4. この記録と manifest の写しをディレクトリの外に置いてから、`/tmp/ralph-rs` を
   消した。

ディレクトリの外に残ったものは次のとおり。

- `~/.claude.json` の `/private/tmp/ralph-rs/work` の信頼の記録。
- `~/.claude/projects/` の下の、claude が動いた 2 つのディレクトリ(repo と機能の
  worktree)のセッションの記録。
- メインの checkout(`yoshpy-dev/ralph`)とその worktree は読んだだけで、ビルドの
  ほかは何も変えていない。メインの checkout の台帳は使っていない。
