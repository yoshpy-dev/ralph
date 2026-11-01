# org-ledger-gaps: PR #290 の test gaps 3・4 の実物の確かめ

- Date: 2026-11-01(23:3x JST)
- Plan: `docs/plans/active/2026-11-01-org-ledger-gaps.md`(Scope の「PR #290 の test gaps 3・4」)
- Binary: HEAD 81deeebf を `go build -o <SCRATCH>/bin/ralph ./cmd/ralph` で build した(`ralph version` は `ralph dev (unknown unknown)`)。PATH の ralph は使っていない
- 打った場所: worktree と main checkout の外の一時の git repository。利用者の実際の org の台帳には触れていない
- 打つ前の環境: `env | grep -E '^RALPH_|^AGMSG|^HERDR'` は 0 件。`pgrep -fl 'ralph org watch'` も 0 件(rc 1)

出力のパスは長いので、次の 2 つを略して書く。略したところのほかは出力をそのまま写した。

- `<SCRATCH>` = `/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/8e827038-7e31-4559-9e37-74cf3c881135/scratchpad`
- `<GAP>` = `<SCRATCH>/gap`

## 作ったパス

| 名前 | パス | 作り方 |
| --- | --- | --- |
| PROBE | `/Users/hiroki.yoshioka/.ralph-probe-2cd83524` | 名前だけを決めた(`openssl rand -hex 4`)。どのコマンドも作らなかった(下の snapshot) |
| REPO | `<GAP>/repo.4pPwZb` | `mktemp -d`、`git init -b main`、空のコミット 1 つ(`GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null`、HOME は実物のまま) |
| ENVDIR | `<GAP>/envledger.l0b0WQ` | `mktemp -d`。`RALPH_ORG_STATE_DIR` に入れる「別の台帳」 |
| WATCHDIR | `<GAP>/watchledger.0ZiFXn` | `mktemp -d`。gap 4 の `--state-dir` |
| binary | `<SCRATCH>/bin/ralph` | 上の build |

## Gap 3: 実物の `ralph.local.toml` と実際の HOME の `~/`

REPO の root に個人用のファイルを置いた。

```
$ cat ralph.local.toml
[org]
state_dir = "~/.ralph-probe-2cd83524/ledger"
```

### 3a. 台帳が実際の HOME の下に決まる(読むだけのコマンド)

```
$ ralph config get org.state_dir
~/.ralph-probe-2cd83524/ledger
rc=0
$ ralph status
no org runtime state found (state-dir=/Users/hiroki.yoshioka/.ralph-probe-2cd83524/ledger (source: config)) — run `ralph org spawn` to start one.
org watch: not running
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work; once attached, a director merges the PRs ralph org merge-order lists as ready with ralph org merge, without asking ([org.director].auto_merge = true)
merge guarantee: unknown (no org ledger yet)
Run `ralph doctor` to check environment readiness.
rc=0
--- probe exists?
ls: /Users/hiroki.yoshioka/.ralph-probe-2cd83524: No such file or directory
```

`config get` は書いたとおりの値を出す。`ralph status` は `~/` を実際の HOME(`/Users/hiroki.yoshioka`)で展開した台帳を source `config` で示した。読むだけなので、PROBE は作られていない。書き込みで確かめる必要はなかった。

### 3b. `RALPH_ORG_STATE_DIR` が別の台帳を指し、個人用のファイルが台帳を決めているとき: 注意だけ

```
$ RALPH_ORG_STATE_DIR=<GAP>/envledger.l0b0WQ ralph status
note: RALPH_ORG_STATE_DIR is set (<GAP>/envledger.l0b0WQ), but ralph no longer reads it; using the org ledger at /Users/hiroki.yoshioka/.ralph-probe-2cd83524/ledger. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
no org runtime state found (state-dir=/Users/hiroki.yoshioka/.ralph-probe-2cd83524/ledger (source: config)) — run `ralph org spawn` to start one.
org watch: not running
rc=0

$ RALPH_ORG_STATE_DIR=<GAP>/envledger.l0b0WQ ralph org spawn --org-id probe --id implementer --role implementer --driver claude --cwd <GAP>/repo.4pPwZb --scope probe
rc=1
note: RALPH_ORG_STATE_DIR is set (<GAP>/envledger.l0b0WQ), but ralph no longer reads it; using the org ledger at /Users/hiroki.yoshioka/.ralph-probe-2cd83524/ledger. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
org: --model omitted; falling back to first [org].model_pool entry permitted for role implementer on claude: fable (pass --model explicitly)
rejected: org: no ralph org watch --all is running on the org state dir /Users/hiroki.yoshioka/.ralph-probe-2cd83524/ledger, so no new work starts; start one with ralph org watch --all --detach, or, for a new org, start it with ralph org start, which starts one
org: no ralph org watch --all is running on the org state dir /Users/hiroki.yoshioka/.ralph-probe-2cd83524/ledger, so no new work starts; start one with ralph org watch --all --detach, or, for a new org, start it with ralph org start, which starts one
```

台帳を変える `ralph org spawn` は、注意を 1 行出して先へ進んだ。そのあと、見張りがないので watch gate に断られた。gate は manifest を読む前に断るので(`internal/org/spawn.go` の Spawn の手順 0)、何も書かれない。前後の snapshot はどちらも次のとおりだった。

```
[snapshot] probe:
  bfs: error: /Users/hiroki.yoshioka/.ralph-probe-2cd83524: No such file or directory.
[snapshot] envdir:
  <GAP>/envledger.l0b0WQ
[snapshot] repo (excluding .git):
  <GAP>/repo.4pPwZb
  <GAP>/repo.4pPwZb/ralph.local.toml
```

### 3c. 個人用のファイルを消したとき: 台帳を変えるコマンドは何にも触れる前に断られる

```
$ rm ralph.local.toml
$ RALPH_ORG_STATE_DIR=<GAP>/envledger.l0b0WQ ralph status
note: RALPH_ORG_STATE_DIR is set (<GAP>/envledger.l0b0WQ), but ralph no longer reads it; showing the org ledger at <GAP>/repo.4pPwZb/.harness/state/org instead, and the commands that change the ledger are refused while the variable names another one. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
no org runtime state found (state-dir=<GAP>/repo.4pPwZb/.harness/state/org (source: git-main-worktree)) — run `ralph org spawn` to start one.

$ RALPH_ORG_STATE_DIR=<GAP>/envledger.l0b0WQ ralph org spawn --org-id probe --id implementer --role implementer --driver claude --cwd <GAP>/repo.4pPwZb --scope probe
rc=1
ralph org spawn: refusing to change the org ledger: RALPH_ORG_STATE_DIR names <GAP>/envledger.l0b0WQ, which ralph no longer reads, and the ledger ralph finds by default is <GAP>/repo.4pPwZb/.harness/state/org, so nothing was changed; to keep using <GAP>/envledger.l0b0WQ, write [org].state_dir = "<GAP>/envledger.l0b0WQ" in <GAP>/repo.4pPwZb/ralph.local.toml or pass --state-dir <GAP>/envledger.l0b0WQ, and unset RALPH_ORG_STATE_DIR; to use <GAP>/repo.4pPwZb/.harness/state/org, unset RALPH_ORG_STATE_DIR
```

拒否の文は「ralph org spawn: refusing to change the org ledger:」で始まる(AC10。頭のコマンドのパスは `resolveOrgLedger` の形)。`ralph.local.toml` のパスは台帳のパスから作られ、REPO の root を指している(AC7)。spawn の結果の行(`rejected:`)は出ず、resolveOrgLedger の段で返った。前後の snapshot はどちらも次のとおりで、REPO に `.harness` はできず、ENVDIR も空のままだった。

```
[snapshot] probe:
  bfs: error: /Users/hiroki.yoshioka/.ralph-probe-2cd83524: No such file or directory.
[snapshot] envdir:
  <GAP>/envledger.l0b0WQ
[snapshot] repo (excluding .git):
  <GAP>/repo.4pPwZb
```

## Gap 4: `--state-dir` 付きの `ralph org watch --detach` の子の watch.log

`ralph org watch --help` を先に読んだ。空の台帳では、期限の過ぎた受信箱の件も、登録された director も、走っている org もない。そのため、見張りは人への通知、herdr、agmsg のどれも呼ばない。`./ralph.toml` もないので wake_cmd もない。子は `--state-dir` の値と環境(HERDR の 4 つを除く)を受け継ぎ、その台帳が main worktree を名指さないので、親の作業ディレクトリ(REPO)で動く(`orgWatchChild`、`internal/cli/org_watch.go:406`)。止めるのは help のとおり `ralph org watch --stop`。

```
$ pgrep -fl "ralph org watch"  (before)
pgrep rc=1

$ RALPH_ORG_STATE_DIR=<GAP>/envledger.l0b0WQ ralph org watch --all --detach --state-dir <GAP>/watchledger.0ZiFXn --interval-seconds 5
rc=0
note: RALPH_ORG_STATE_DIR is set (<GAP>/envledger.l0b0WQ), but ralph no longer reads it; using the org ledger at <GAP>/watchledger.0ZiFXn. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
started ralph org watch --all (pid 28879); log: <GAP>/watchledger.0ZiFXn/watch.log

$ pgrep -fl "ralph org watch"  (after start)
28879 <SCRATCH>/bin/ralph org watch --all --state-dir <GAP>/watchledger.0ZiFXn --interval-seconds 5

$ ps -o pid,ppid,command -p 28879
  PID  PPID COMMAND
28879     1 <SCRATCH>/bin/ralph org watch --all --state-dir <GAP>/watchledger.0ZiFXn --interval-seconds 5

$ cat <GAP>/watchledger.0ZiFXn/watch.log
note: RALPH_ORG_STATE_DIR is set (<GAP>/envledger.l0b0WQ), but ralph no longer reads it; using the org ledger at <GAP>/watchledger.0ZiFXn. Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR.
watching every org (interval=5s once=false state-dir=<GAP>/watchledger.0ZiFXn (source: flag))

$ grep -c RALPH_ORG_STATE_DIR <GAP>/watchledger.0ZiFXn/watch.log
1
```

子の watch.log には、`RALPH_ORG_STATE_DIR` についての注意が 1 行だけある。子は env を受け継ぎ、`--state-dir` で台帳が決まる(source `flag`)ので、拒否でなく注意になる。親の stderr にも同じ注意が 1 行出た。ENVDIR は空のままだった。

```
$ ralph org watch --stop --state-dir <GAP>/watchledger.0ZiFXn
stopped ralph org watch --all (pid 28879)
rc=0

$ pgrep -fl "ralph org watch"  (after stop)
pgrep rc=1
$ ps -p 28879 >/dev/null 2>&1 && echo "pid 28879 alive" || echo "pid 28879 gone"
pid 28879 gone
```

止めたあとの WATCHDIR には `watch-alive.json`、`watch.lock`、`watch.log` の 3 つがあり、watch.log は上の 2 行のままだった。自分が起こした見張り(pid 28879)のほかに、`ralph org watch` の process は前にも後にもなかった。

## 後片付け

確かめが終わってすぐ、report を書く前に消した。

```
$ rm -rf "$PROBE" "$REPO" "$ENVDIR" "$WATCHDIR" "<SCRATCH>/bin"
gone: /Users/hiroki.yoshioka/.ralph-probe-2cd83524
gone: <GAP>/repo.4pPwZb
gone: <GAP>/envledger.l0b0WQ
gone: <GAP>/watchledger.0ZiFXn
gone: <SCRATCH>/bin
$ ls -d $HOME/.ralph-probe-*
(eval):1: no matches found: /Users/hiroki.yoshioka/.ralph-probe-*
$ pgrep -fl 'ralph org watch'
pgrep rc=1
```

## まとめ

- Gap 3: 実物の `ralph.local.toml` に書いた `~/.ralph-probe-2cd83524/ledger` は、実際の HOME の下の台帳に決まった(source `config`)。`RALPH_ORG_STATE_DIR` が別の場所を指していても、個人用のファイルが台帳を決めているあいだは、台帳を変える `ralph org spawn` も注意を 1 行出すだけで先へ進んだ(そのあと watch gate が断った)。個人用のファイルを消すと、同じ spawn は「ralph org spawn: refusing to change the org ledger:」で断られ、REPO、ENVDIR、PROBE のどれにも何もできなかった。
- Gap 4: `--state-dir` 付きの `ralph org watch --all --detach` の子の watch.log に、`RALPH_ORG_STATE_DIR` の注意がちょうど 1 行出た。見張りは `--stop` で止まり、残った process はない。
- 確かめていないこと: linked worktree やサブディレクトリから打った場合(テストの `TestOrgLedger_PersonalFileNamesTheLedgerFromAnyWorktree` が見ている)、`ralph org start` が見張りを起こす道(見張りの process が残るので打っていない)。
