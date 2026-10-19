# Walkthrough: org-director-wait(8 段目の後半の 2 本目の後半、8b-2b)

- Date: 2026-10-19
- Plan: docs/plans/active/2026-10-19-org-director-wait.md(/pr で archive に移す)
- Branch: feat/org-director-wait(base e38b89a9、HEAD a005edec。この walkthrough を書いた時点で 39 commits、53 files、+7881/-472)
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 42 files、+5129/-472 になる。そのうちテストは 12 files、+3158/-20 で、足した `func Test` は 55 本。非テストの Go は 15 files、+1589/-344。残りは文書で、`/director` skill の 4 面(各 +83/-20)、`/org` skill の 4 面(各 +5/-3)、recipe `director-wake.md` の 2 面、headless の雛形 `internal/org/prompts/director.md`、README、`AGENTS.md`、仕様の注、tech-debt
- 後工程は 2 回まわした(cross-review の triage は `Cycle: 2/2 (cap reached)`)。1 回目の cross-review の ACTION_REQUIRED 1 件を直して全部を回し直し、2 回目の cross-review は 0 件だった(下の 8)

新しいファイルは `internal/org/director_wait.go`(799 行)、`internal/cli/org_wait_director.go`(137 行)、`internal/cli/org_director_wait_line.go`(116 行)、`internal/cli/org_director_report.go`(342 行、`org_director.go` から分けた)、`internal/cli/exitcode.go` と、それぞれのテスト。次の順に読むと、director の wait が始まり、印を書き、見張りの停止か登録の変化で返り、それが表示に出るまでを 1 本で追える。

```
wait --inbox --director d1 → 今の登録か → 見張りを確かめる → ロックの中で登録を読み直して持ち主になる
  → 繰り返す: 受信箱(1 秒)、見張り(5 秒)、登録と印(30 秒)
  → 終わり方を印に書いて返る(0 / 1 / 4 / 5 / signal)
  → ralph status と ralph org director status が今の登録に合う印だけを読む
```

## 1. 印と持ち主(`internal/org/director_wait.go`、S1、67938662)

- `WaitInboxForDirector` が `ralph org wait --inbox --director <id>` の org 層。受信箱の open の件を `WaitInbox` と同じく返し、そのほかに次の 3 つをする(D1〜D3)
- 印は `DirectorWaitMark`(`director-wait.json`、台帳の下)。field は `wait_id`(`<pid>-<始めた時刻の ns>`)、`id`、`attached_at`、`pid`、`started_at`、`seen_at`、`watch_accepting`、`ended_at`、`end`、`detail`。書き込みはいつも `director-wait.lock` の flock の中で、`writeJSONFileAtomic`(一時ファイルと rename)で行う
- 持ち主: 2 回目からの書き込み(`save`)は、ディスクの印の `wait_id` が自分のときだけ行う(`owns`)。持ち主でなくなった wait は `sayLost` で stderr に 1 回だけ書き、印には二度と書かずに待ち続ける。印が消えていた場合も同じ(Codex の計画助言 2)
- 終わり方 `end` は `items`(0)、`timeout`(1)、`watch`(4)、`not_current`(5)、`error`(1)、`signal`。`end` を書くのは持ち主のときだけ
- `DirectorWaitFor` は、印の `id` と `attached_at` が今の登録と合うときだけ印を返す。表示はこれで、別の登録の印を出さない(8b-2a の `DirectorAliveFor` と同じ形)
- `internal/org/escalate.go` は、`WaitInbox` と共有する `openInboxItems` を切り出しただけ。`spawn.go` はコメントだけ

## 2. 登録の確かめ(`directorWaitCheckCurrent`、`begin`、`refresh`)

- 登録を読むのは 3 回: 始める前(`WaitInboxForDirector`。違えば何も書かずに `ErrDirectorWaitNotCurrent`、終了コード 5)、最初の印を書く前の `director-wait.lock` の中(`begin`)、`DirectorWaitMarkInterval`(30 秒)ごと(`refresh`。違えば印に `not_current` を書いて 5)
- `directorWaitCheckCurrent(id, started, cur)` は、cur が id で stopped でなく、started が nil でなければ `attached_at` も同じときに通す。始める前の確かめは nil を渡し、`begin` と `refresh` は始めたときの登録を渡す
- 1 回目の cross-review の指摘(P2)で、`begin` のロックの中の読み直しを足した(38cac6f1)。前は、始める前の確かめと見張りの確かめ(settle は最大 2 秒)のあいだに `attach --replace` が入ると、古い wait が新しい wait の印を上書きし、新しい wait は次の `save` で持ち主でないと分かって印を書かなくなっていた。今は、今の登録でなければ何も書かずに 5 で、`director.jsonl` を読めなければ何も書かずにその誤りで返る。ロックを取れないとき(lock の path が書けない)は今までどおり、ロックなしで前の印を読み、書けなかったことを 1 回言って待ち続ける
- テスト用の差し込み `directorWaitBeforeBeginLock` は、見張りの確かめのあと、ロックの前に呼ばれる。本番では nil
- `begin` は 6086af65 で 44 行に分けた: 前の印を読み、引き継ぐ `watch_accepting` と `seen_at` を決める `decideStart`、見張りの様子で `note:` を出すか 4 の準備をする `noteStartWatch`。動きは変えていない

## 3. 見張りの確かめ(`probe`、`probeAgain`、`directorWaitWatchFacts`、D3)

- 始めたときと `DirectorWaitWatchProbeInterval`(5 秒)ごとに `ProbeWatch` を読み、`WatchAcceptsWork` で決める。`stale` のときだけ `DefaultWatchGateSettle` のあいだ読み直す(起動し直した見張りが最初の印を書く前の瞬間を止まりと取り違えないため)
- 終了コード 4(`ErrDirectorWaitNoWatch`)は、受け付けていた見張りが受け付けなくなったときだけで、その回の受信箱に open の件があれば 0 を先にする。始めたときは、同じ登録の前の印の `watch_accepting` が true なら「受け付けていた」とみなす。Codex の 300 秒の wait の繰り返しで、wait と wait のあいだに止まった見張りも次の wait が 4 で拾う
- 始めたときから受け付けていなければ `note:` を 1 行出して待ち続ける。受け付けるようになれば、もう 1 行出して印をすぐ書き直す
- 4 と `note:` の文(`directorWaitWatchFacts`、`directorWaitWatchState`)は、lock と見張りの印の事実だけを言い、止まった原因は言わない。どの文も `ralph org watch --all --detach` を名指し、見張りがないあいだの escalate は人にすぐ届くことと、それより前に director に残された件は見張りがまた動くまで期限で人に届かないことを言う。値は `PrintableInboxText` で escape する

## 4. CLI と終了コード(S2、000e83b9、4e4cc06c、5667f850)

- `internal/cli/org.go`: `--director` は `--inbox` のときだけ受け付ける。`runOrgWaitInbox` が `--director` を `runOrgWaitInboxDirector`(`org_wait_director.go`)に回す。`CheckDirectorWaitID` で `d<N>` の形を、設定を読む前に確かめる(5667f850。AC1 の「何も読まずに断る」)
- `runOrgWaitInboxDirector` は SIGINT、SIGTERM、SIGHUP を受け、印に `end=signal` を書いてから、`reraiseSignal` で同じ signal で終わる。始めたときに無視されていた signal(nohup など)は受け取らない(`notIgnoredSignals`、4e4cc06c)
- `internal/cli/org_gate.go`: どの signal で ctx が終わったかを返す `signalCauseContext` を足し、gate の `gateSignalContext` もこれを使うようにした
- `internal/cli/exitcode.go` の `ExitCode` が誤りから終了コードを決め、`cmd/ralph/main.go` がそれを返す。`ErrDirectorWaitNoWatch` は 4、`ErrDirectorWaitNotCurrent` は 5、upgrade の drift は今どおり 3、ほかは 1。`--director` のない wait は 4 と 5 を返さない

## 5. 表示(S3、fa95a3d8、c1d31e3f)

- `internal/cli/org_director_wait_line.go`: `readOrgDirectorWait` が今の登録に合う印を読み、`orgDirectorWaitLine` が `ralph org director status` の `wait:` の行を、`orgDirectorWaitClause` が `ralph status` の `director:` の行の続き(昇格したセッションのときだけ)を作る。走っている wait は「wait(pid N)が <started_at> に始まり、最後に印を書いたのは <seen_at>」、`seen_at` が 60 秒より古ければ「それから印を書いていない(30 秒ごとに書く)」を足す。終わった wait は終わった時刻と終わり方。「director が待っている」とは書かない
- `--json` は、合う印があるときだけ `director.wait` を載せる。印が読めないときは理由を 1 つ足し、status は失敗しない
- 1 回目の self-review の M1 で、director の report を `internal/cli/org_director_report.go` に分けた(c1d31e3f。`org_director.go` は 505 行になった)。headless の登録の `director:` の行は変えていない

## 6. 文書(S4、aba16ec2、61114ca4、以降の直し)

- `/director` skill の 4 面: wait の例に `--director <自分の id>` を付け、終了コード 4(事実と起動のコマンドを人に伝え、自分では起動し直さず、wait を打ち直す)、5(director をやめて人に伝える。detach は打たない)、`note:`(前の wait と同じなら伝え直さない)の扱いを書いた。headless の雛形は `--director {{ID}}` を付ける
- `/org` skill の 4 面、recipe `director-wake.md` の 2 面、README、`AGENTS.md`、仕様の 8b-2b の注を、`--director` の wait に合わせた
- tech-debt: 247 行(昇格したセッションの登録)と 209 行(見張りの停止)をこの段で入った分に書き換え、B(持ち主の終わりで止まる)を 8b-2c の新しい行にした。そのほか、テストの穴(308 行、残りは G6 だけ)、`director_wait.go` の行数(306 行、799 行)、run-verify の時間(307 行)、doctor のテストの揺れ(309 行)を書いた

## 7. evidence(S5、c780bc11)

- `docs/evidence/director-wait-session-end-2026-10-19.md`: 計画の前の実験。Claude Code の background の task が、セッションの終わり方ごとに残るか
- `docs/evidence/director-wait-livefire-2026-10-19.md`: 分離した台帳(`/tmp/dw8`)で、fa95a3d8 の binary を使った D8 の (a)〜(h)。38cac6f1 は最初の確かめとロックのあいだに登録が変わった場合の動きを変えたが、(a)〜(h) はその道を通らず、打ち直していない(冒頭に書いた)
- `docs/evidence/test-2026-10-19-org-director-wait.md` と `-mutation.txt`: 2 回の `/test` の生の出力、mutant の表、probe の source

## 8. 後工程の経過

- 1 回目: self-review(Merge: yes、MEDIUM 1、LOW 7、すべて直した)、verify(pass)、test(pass、mutant 56 個のうち 50 個が赤)、sync-docs、cross-review(ACTION_REQUIRED 1 件、上の 2)
- 2 回目(上限の 2 回目): self-review(Merge: yes、LOW 2 件は 6086af65 で直した)、verify(pass、LOW 1 件は sync-docs で直した)、test(pass、cycle 2 のコードに mutant 30 個、穴の 4 件 G8〜G11)、sync-docs、cross-review(0 件)
- 2 回目の test のあとに、穴 G8〜G11 の probe 4 本をテストにした(99144e8f)。本番のコードは変えていない。規則が全部の回し直しを求めるのは cross-review の直しのあとなので回し直していないが、この 4 本は 2 回目の self-review、verify、test の subagent が見ていない。見たのは、2 回目の cross-review、implementer の mutant の確かめ(B01、N1、N2、B12、B19 で赤)、PR の前の run-verify の全パッケージの回
- PR の前の `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は rc 1 だった。落ちたのは、このブランチが変えていない doctor のテスト 2 本で、load average 50 前後の中でスタブの `claude --version` が 5 秒の時間切れになった。2 本とも単独では 3 回とも通り、`internal/cli` の走らせ直しは 376 秒で通った。静的解析、shell のテスト 74 本、ほかの Go のパッケージ(`internal/org` は 1537 秒)は同じ回で通った

## 9. 読むときに見るとよいところ

- `begin` の順(`internal/org/director_wait.go` の `begin`): 見張りの確かめ → 差し込み → ロック { 登録の読み直し → 確かめ → 前の印 → 書く } → ロックが取れなかったときの道。ロックの中で読む `DirectorStore.Read` はロックを取らないので、ロックの入れ子はない
- `owns` と `sayLost`: 持ち主でなくなった wait が二度と書かないこと。`TestWaitInboxForDirector_ARegistrationReplacedBeforeTheStartTakesTheLock` と `..._ASameIDAttachedAgainBeforeTheStartTakesTheLock` が、上書きされた印が新しい wait のまま残り、新しい wait がそのあとも書けることを固定する
- 4 の条件(`loop` の見張りの枝): 受け付けていた見張りが受け付けなくなったときだけで、open の件が先
- 残り: 実機の打ち直しはしていない(上の 7)。Monitor で打った wait、codex の繰り返しで打った wait、nohup で始めた wait は実機で見ていない(nohup はテストだけ)。Linux での signal の扱いは手元で確かめていない。`ralph org director --help`(グループ)の段落は 8a-2 の言い方のまま
