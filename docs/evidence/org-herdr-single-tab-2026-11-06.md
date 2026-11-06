# 1 タブのペイン配置の実機確認

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(S7、AC11。AC1・AC3・AC4 の実機の側)
- Spec: `docs/specs/2026-11-06-org-herdr-autostart-panes.md`(B、AC-B9)
- 生の出力は分離に使った一時ディレクトリに置き、確かめたあとディレクトリごと消した。要点はこのファイルに写した。分離に使った一時ディレクトリは `<D>`(`/tmp/ralph-iso`)と書く。

## 環境

| 項目 | 値 |
|---|---|
| マシン | macOS(Darwin 25.5.0) |
| herdr | 0.7.5。`<D>` の下に `XDG_CONFIG_HOME`・`XDG_STATE_HOME`・`HERDR_SOCKET_PATH` を向けて分離した(`docs/recipes/isolated-herdr-server.md`) |
| ralph | このブランチの 1468232d を `go build -o <D>/bin/ralph ./cmd/ralph` で作った |
| repo | `<D>/work`。`git init` した scratch の repo で、`ralph.toml` は `[org].agmsg_home = "<D>/agmsg"`(recipe の「private agmsg」の写し。`scripts`・`plugins`・`types`・`SKILL.md`・`VERSION` だけを複製)と `[org.permissions].default = "guarded"` |
| 座席 | `--role worker --driver claude --model sonnet`、`--prompt` なし。未知の役割なので雛形も無く、座席の claude には何も送っていない |
| 通知 | recipe の `osascript` の shim を PATH の先頭に置いた。確かめの間に呼ばれた記録は無い |

wrapper `<D>/run` は `env -i` で `HOME`・`USER`・`LOGNAME`・`SHELL=/bin/bash`・`LANG`・`TERM`・`TMPDIR`・`PATH`(`<D>/bin`、claude の置き場所、`/opt/homebrew/bin`、システムの bin)・XDG の 2 つ・`HERDR_SOCKET_PATH` だけを渡した。ユーザーの herdr のサーバー(pid 22408)にはどの段でも触れていない。確かめの前と後の `pgrep -fl 'herdr server'` は、どちらも 22408 だけを出した。

## 手順と結果

見張りは `<D>/run ralph org watch --all --detach` で起動した。spawn はどれも `<D>/run ralph org spawn --org-id iso2 --id <seat> --role worker --driver claude --model sonnet --cwd <D>/work --timeout-ms 60000` で打った。herdr のサーバーは打つ前に止まっていて、1 席目の spawn が自動で起動した(stderr に `ralph org spawn: started the herdr server (pid 73543, socket /tmp/ralph-iso/h.sock); …`)。

### 1. leader → implementer → reviewer の配置(AC1)

3 席とも終了コード 0 で、座席のペインは `w1:p1`(leader)、`w1:p2`(implementer)、`w1:p3`(reviewer)だった。

```
ws w1 iso2 tabs 1 panes 3
tab w1:t1 iso2 panes 3
pane w1:p1 w1:t1 leader blocked
pane w1:p2 w1:t1 implementer blocked
pane w1:p3 w1:t1 reviewer idle
area {'height': 23, 'width': 54, 'x': 26, 'y': 1}
rect w1:p1 {'height': 23, 'width': 27, 'x': 26, 'y': 1}
rect w1:p2 {'height': 12, 'width': 27, 'x': 53, 'y': 1}
rect w1:p3 {'height': 11, 'width': 27, 'x': 53, 'y': 13}
split right 0.5
split down 0.5
```

ワークスペースのタブは 1 つで、ラベルは org_id(`iso2`)だった。leader が左半分、implementer が右上、reviewer が右下にあり、ペインのラベルはそれぞれの座席 id だった。空のタブ「1」は無い。`blocked` は claude の起動時の画面(新しいフォルダの信頼の確かめとみられる)で止まっている状態で、答えていない。配置と名前の確かめには関わらない。

台帳の spawn_step は次のとおりで、作り方とタブの id が残っている。

```
spawn_step leader w1:p1 pane_created how=root tab=w1:t1
spawn_step implementer w1:p2 pane_created how=split-right tab=w1:t1
spawn_step reviewer w1:p3 pane_created how=split-down tab=w1:t1
```

### 2. implementer を止める(AC3)

`ralph org stop --org-id iso2 --seat implementer` は `stopped seat "implementer"` を出した。そのあとの herdr は次のとおりで、implementer のペインだけが閉じ、reviewer が右の列全体に広がった。

```
pane w1:p1 w1:t1 leader
pane w1:p3 w1:t1 reviewer
rect w1:p1 {'height': 23, 'width': 27, 'x': 26, 'y': 1}
rect w1:p3 {'height': 23, 'width': 27, 'x': 53, 'y': 1}
```

続けて implementer を作り直すと、`w1:p4` が reviewer の下に入った(`pane_created how=split-down tab=w1:t1`。右上 27×12 が reviewer、右下 27×11 が implementer)。spawn の順に並ぶ規則どおりで、役割で位置を固定しないことは仕様の Design decisions で決めてある。

### 3. 全席を止めたあとの spawn(AC4)

implementer・reviewer・leader の順に止めると、`herdr workspace list` は空になり、`herdr workspace get w1` は `workspace_not_found` を返した。最後のペインが閉じたところで、herdr がワークスペースごと消した。

そのあと leader を spawn すると、`pane_id=w2:p1` で終わり、台帳には次の 3 行が続いた。

```
org_workspace_closed - w1 not found at spawn
org_workspace_created - w2
spawn_step leader w2:p1 pane_created how=root tab=w2:t1
```

新しいワークスペース `w2`(label `iso2`、タブ 1 つ、ペイン 1 つ)の、ルートペインに leader が入った。`herdr pane get w2:p1` のラベルは `leader`、`herdr tab get w2:t1` のラベルは `iso2` だった。

### 4. 解散

`ralph org disband --org-id iso2` は `stopped seat "leader"` と `disbanded org "iso2"` を出し、そのあとの `herdr workspace list` は空だった。

## 確かめていないこと

- headless の director(`ralph org director start`)を実機で立てていない。director に役割の雛形が渡り、モデルに指示が送られるためである。ルートペインに置くこと、ラベル、止める前の確かめは org のテスト(`director_start_root_pane_test.go`、`confirm_pane_test.go`)で確かめた。
- 前の版で作ったワークスペースに新しい座席を足す場合と、spawn がペインを作ったあと名前を付ける前に止まった場合の補償は、テストだけで確かめた。
- 2 席を同時に spawn する場合(配置のロック)は、テストだけで確かめた。
- 座席の agent が画面で止まらずに入力を待つ状態までは確かめていない(フォルダ信頼の画面に答えなかった)。

## 片付け

`<D>/run ralph org watch --stop`(`stopped ralph org watch --all`)、`<D>/run herdr server stop` を打った。そのあとの `pgrep -fl 'herdr server'` はユーザーの 22408 だけを出し、`lsof +D <D>` は確かめのコマンド自身しか出さなかった。最後に `<D>` を消した。ralph の agmsg の呼び出し(各座席の join と、leader 以外の座席の HELLO)は private の写しに向き、写しは `<D>/agmsg` の下に自分の `db` と `teams` を作った。ユーザーの agmsg の db の時刻は見ていない(ユーザーの別の org が同じ時間に使っていた)。
