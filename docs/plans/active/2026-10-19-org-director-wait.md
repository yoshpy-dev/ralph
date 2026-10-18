# org-director-wait

- Status: Draft
- Approved: TBD
- Owner: Claude Code
- Date: 2026-10-19
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 8 段目の後半の 2 本目の後半(8b-2b)。昇格したセッションの登録が生死を出さないこと(tech-debt の 247 行)と、見張りの停止に director が気づかないこと(209 行の半分)。8b-2a は PR #237 で merge 済み(main は e38b89a9)。中身と分け方は consult(consult-plan-promoted、2026-10-19)の判定による。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-director-wait

## Objective

director の受信箱の wait(`ralph org wait --inbox`)に `--director <id>` を足す。この wait は、待っているあいだ自分の pid と時刻を印のファイル(`director-wait.json`)に書き、`ralph status` と `ralph org director status` がその事実を出す。これで、昇格したセッションの登録にも「その director の wait がいつから走り、最後にいつ印を書いたか、いつどう終わったか」が付く。同じ wait は横断の見張りが件を受け付けていないと分かれば、open の件がないときに限り決まった終了コードで返る。director はそれで見張りの停止に気づき、`/director` skill の手順で人に伝え、落ちた見張りなら起動し直す。`--director` のない wait は今のまま変わらない。

## Scope

- `ralph org wait --inbox --director <id>`: 始める前と待つあいだ、今の登録が `<id>` かを確かめる。違えば、待たずに(または待つのをやめて)終了コード 5 で返る
- 印 `director-wait.json`(台帳の下、一時ファイルと rename で上書き): wait の id と登録の `attached_at`、pid、始めた時刻、最後に書いた時刻、終わった時刻と終わり方
- 見張りの確かめ: `--director` の wait は `ProbeWatch` を一定の間隔で読み、見張りが件を受け付けていなければ、open の件がないときに限り終了コード 4 で返る。stderr の 1 行に、見張りがどう止まったか(stop で止まった、印を残したまま終わった、印が古い、`--once`)を書く
- 終了コード 4 と 5 を `cmd/ralph/main.go` で返す(3 は upgrade の drift が使っている)
- 表示: `ralph org director status` の `wait:` の行、`ralph status` の `director:` の行(昇格したセッションのとき)、どちらの `--json` にも `director.wait`
- 文書: `/director` skill の 4 面(wait に `--director` を付ける、終了コード 4 と 5 の扱い)、headless の director の雛形 `internal/org/prompts/director.md`、help、README、仕様の注、tech-debt、AGENTS.md
- evidence: Claude Code の background の task がセッションの終わり方ごとに残るかの実験(2026-10-19、計画の前に打った分)を写す。分離した台帳で、作った binary の wait、見張りの停止、登録の入れ替えを確かめる

## Non-goals

- wait が自分の持ち主(wait を打ったセッション)の終わりを見て止まること(consult の判定で B として切り離した)。実験では、claude の SIGKILL と `/exit` の「Move to background and exit」のあとで background の task が残り、そのシェルの親が pid 1 になった。Monitor で打った wait の親のつながりと、daemon に移ったセッションが task の終わりを受けて何をするかは確かめていない。codex は app-server の daemon が wait の親なので、この見方では分からない。tech-debt に 1 行足し、確かめたあとで 8b-2c にする
- 見張りの確かめ(`CheckDirector`、`director-alive.json`)を昇格したセッションに広げること。印は wait が書き、表示が直接読む。見張りが止まっていても印は読める
- `--director` のない `ralph org wait --inbox`、座席の `ralph org wait` の変更
- wait が見張りを起動すること。起動し直すのは skill の手順に従う director で、wait は返るだけ
- 印から「生きている・死んだ」を決めること。表示は印の事実(pid、時刻、終わり方)だけを出す
- push だけの登録(wait を打たない director)の印
- 9 段目の試行

## Assumptions

- `ralph org wait --inbox` は台帳の誰でも打てる。印を書くのは `--director <id>` を付け、その id が今の登録と合う wait だけなので、leader の座席が打った `--director` のない wait は印を書かない
- `ProbeWatch`(`internal/org/watchalive.go`)は、見張りのプロセスが終わって `watch.lock` が外れると、すぐ `not_running` を返す。固まった見張りは、印の時刻が窓(`deadman_minutes`、既定 10 分、間隔の 2 倍より短くはならない)より古くなるまで `running` のまま
- 見張りは Ctrl-C、SIGTERM、`--stop` で止まると印に `stopped` を書く(`internal/org/watch.go`)。SIGKILL や落ちた見張りは `running` か `stopping` を残したまま `watch.lock` を外す。この 2 つは印の state で見分けられる
- Claude Code 2.1.291 の background の task は、`/exit` の既定(Exit and stop tasks)、`claude -p` の終わり、pane を閉じたとき(SIGHUP)はセッションと一緒に消え、claude の SIGKILL と「Move to background and exit」では残る(計画の前の実験。S5 で evidence に写す)

## Affected areas

- `internal/org/escalate.go`(`WaitInbox` に director の wait の選択を足す。または新しい `internal/org/director_wait.go` に分ける)
- `internal/org/director_wait.go`(新規: 印の型、読み書き、登録との照合、見張りの確かめ、終わり方と誤り)
- `internal/cli/org.go`(`newOrgWaitCmd` の `--director`、`runOrgWaitInbox`、signal の扱い)
- `cmd/ralph/main.go`(終了コード 4 と 5。cli が誤りから終了コードを決める関数を持つ)
- `internal/cli/org_director.go`、`internal/cli/org_director_check.go`、`internal/cli/status.go`(`wait:` の行、`director:` の行、`--json` の `director.wait`、help)
- テスト: `internal/org/director_wait_test.go`(新規)、`internal/org/escalate_test.go` または inbox の wait のテスト、`internal/cli/org_wait_inbox_test.go` か既存の wait のテスト、`internal/cli/org_director_test.go`、`status_test.go`
- 文書: `.claude/skills/director/SKILL.md` と写しの 3 面、`internal/org/prompts/director.md`、`README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`、`AGENTS.md`
- evidence: `docs/evidence/director-wait-session-end-2026-10-19.md`(実験)、`docs/evidence/director-wait-livefire-2026-10-19.md`(実機)

## Visual review

- Page: `.harness/state/plan-visual/org-director-wait.html`(かんたん版の図 2 枚: `overview`、`easy-branches`。折りたたみの中に `detail-overview`、`detail-flow`、`detail-mark`、`detail-watch`)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: 全体、かんたん版の 2 枚、折りたたみの 4 枚を `shot` で撮って読んだ。流れ図で生存線が 2 つの文字に重なっていたので背景を足し、撮り直して直ったことを確かめた

## Design decisions

- D1 `--director <id>`: `--inbox` のときだけ受け付け、`d<N>` の形でなければ何も読まずに終了コード 1 で断る。wait は始める前に `director.jsonl` を読み、今の登録が `<id>` で stopped でないとき(state のない昇格したセッションの登録、headless の `starting` と `running`)だけ待つ。そうでなければ、印を書かずに `ErrDirectorWaitNotCurrent` で返る(終了コード 5、文は今の登録を名指す)。待つあいだも印を書き直すたびに登録を読み直し、今の登録が `<id>` でなくなったか、`attached_at` が変わったか、stopped になったら、印に `not_current` と書いて終了コード 5 で返る(`--replace` された古いセッションの wait は、これで自分で止まる)
- D2 印 `director-wait.json`: `id`、`attached_at`(登録に結ぶ。表示は 8b-2a の `DirectorAliveFor` と同じく id と attached_at が合う印だけを読む)、`pid`、`started_at`、`seen_at`、終わったときは `ended_at`、`end`、`detail`(短い文)。書くのは始めたとき、`directorWaitMarkInterval`(30 秒)ごと、終わるとき。書き込みは `writeJSONFileAtomic`。印は台帳に 1 つで、後から始めた wait が上書きする。終わるときは、ディスクの印の pid が自分のときだけ書き換える(別の wait が始めた印を消さない)。印を書けなくても wait は続け、stderr に 1 回だけ書く
  - `end` は次のどれか: `items`(open の件を返した、終了コード 0)、`timeout`(`--timeout-ms` の時間切れ、終了コード 1)、`watch`(D3、終了コード 4)、`not_current`(D1、終了コード 5)、`error`(受信箱や登録が読めない、終了コード 1)、`signal`(SIGINT、SIGTERM、SIGHUP。印を書いてから、その signal で終わる)。SIGKILL では書けないので、`seen_at` が古いまま残る
- D3 見張りの確かめ: `--director` の wait は、始めたときと `directorWaitWatchProbeInterval`(5 秒)ごとに `ProbeWatch` を読み、`WatchAcceptsWork` が false で、その回に読んだ受信箱に open の件がなければ、`ErrDirectorWaitNoWatch` で返る(終了コード 4)。open の件があれば終了コード 0 を先にする。`stale` のときだけは、watch gate(`NewWatchGate`)と同じく `DefaultWatchGateSettle` のあいだ読み直し、そのあいだに受け付けるようになれば待ち続ける。起動し直した見張りが lock を取って最初の印を書く前の瞬間を、止まった見張りと取り違えないためである。stderr の 1 行は、見張りがどう止まったかを印の state で書き分ける
  - `stopped`: stop で止まった(Ctrl-C、SIGTERM、`--stop`、`--once` の終わり)。印の `stopped_at` を名指す
  - `died`: `watch.lock` は外れているが、印は `running` か `stopping` のまま(SIGKILL、落ちた)
  - `stale`: lock は持たれているが、印が窓より古いか、settle のあとも running の印がない(固まっている)。`WatchKillHint` を名指す
  - `stopping`: 止まる途中。`once`: `--once` の見張りが走っている
  - `missing`: 印がない(見張りが一度も動いていない)
  - どの文も、見張りが受け付けていたあいだに escalate した件は、見張りが動くまで期限で人に届かないこと、見張りがないあいだの escalate は人にすぐ届くこと(`inbox_no_watch`)を言う
- D4 終了コード: `cmd/ralph/main.go` は、cli の関数で誤りから終了コードを決める。`ErrDirectorWaitNoWatch` は 4、`ErrDirectorWaitNotCurrent` は 5、upgrade の drift は今どおり 3、ほかは 1。`--director` のない wait は 4 と 5 を返さない
- D5 表示: 今の登録に合う印(id と attached_at)があれば、`ralph org director status` は `wait:` の行に、走っている wait なら「pid N が <started_at> から待っていて、最後に印を書いたのは <seen_at>」、終わった wait なら「pid N の wait が <ended_at> に終わった(<end>)」を出す。走っている wait の `seen_at` が 2 回分(60 秒)より古ければ「それから印を書いていない(30 秒ごとに書く)」を足す。合う印がなければ「この登録の `wait --inbox --director` の印はない」。`ralph status` の `director:` の行は、昇格したセッションのときだけ、「登録はセッションが動いているかを表さない」のあとに同じ事実を短く足す。headless の登録の `director:` の行は今の確かめの文のまま。`--json` は `director.wait`(pid、started_at、seen_at、ended_at、end、detail)を、合う印があるときだけ載せる。印の値は今の表示と同じく escape する。印が読めないときは理由を 1 つ足し、status は失敗しない
- D6 skill と雛形: `/director` skill の 4 面は、wait の例に `--director <自分の id>` を付け、終了コード 4 と 5 の扱いを書く
  - 4 で `died` なら `ralph org watch --all --detach` で見張りを起動し直し(`already running` の終了コード 0 もそのまま受ける)、見張りが止まっていたことと、その間に期限を迎えた件は見張りの最初のサイクルで送られることを人に伝えて、また待つ。起動し直しても `ralph status` の `org watch:` が running にならなければ、待つのをやめて人に伝える
  - 4 で `stopped`、`stopping`、`stale`、`once`、`missing` なら起動し直さない。人が止めたか、人の手が要る。そのことを人に伝え、人の指示を待つ(wait は打たない。見張りがないあいだの escalate は人にすぐ届く)
  - 5 なら director として動くのをやめ、そのことを人に伝える(detach は打たない)
  - Codex の 300 秒の wait の繰り返しでも同じ。headless の雛形は wait の例に `--director {{ID}}` を付ける
- D7 B を切り離す: tech-debt に 1 行足す。漏れた wait(claude の SIGKILL、Move to background のあと)は印を書き続け、表示は「pid N が待っている」と出し続ける。実験の事実と、残る確かめ(Monitor の親のつながり、daemon のセッションの動き、codex の daemon)を書き、きっかけを 8b-2c にする。247 行はこの段で入った印のことに書き換え、209 行は `--director` の wait を打つ director が気づくこと、push だけの director と leader はまだ気づかないことに書き換える
- D8 実機: 分離した台帳(herdr は使わない)で、作った binary を使う。shell から `ralph org director attach --driver claude` で登録し、`ralph org watch --all --detach` を立て、`wait --inbox --director d1 --timeout-ms 0` を background で打つ。そのうえで (a) status の `wait:` の行、(b) 見張りを `--stop` で止めたときの終了コード 4(`stopped`)、(c) 見張りを SIGKILL したときの 4(`died`)、(d) `attach --replace` のあとの 5(`not_current`)、(e) SIGTERM で終わった wait の `end=signal`、(f) SIGKILL で終わった wait の古い `seen_at` を確かめる

Critical forks: 8b-2b の中身と分け方、印を書く wait の区別、印の置き場所と読み手、終了コードを consult(consult-plan-promoted)に聞いた。判定は「A(印)と C(見張りの停止)を 1 本、B(持ち主の終わりで止まる)は切り離す。印は `--director <id>` を付けた wait だけが書き、表示が直接読む。CheckDirector は触らない。終了コードは 3 を避ける」で、D1〜D4、D7 に入れた。見張りの止まり方で起動し直すかを分けること(D3、D6。人が止めた見張りを director が起動し直さないため)は、ユーザーの指示「計画の分かれ目も自分で選ぶ」に従って自分で選んだ。

## Acceptance criteria

- [ ] AC1 フラグ: `--director` は `--inbox` のときだけ受け付け、`d<N>` の形でない値と `--inbox` のない指定は何も読まずに終了コード 1 で断る。今の登録がその id でない(登録がない、別の id、stopped)なら、印を書かずに終了コード 5 で返り、文が今の登録を名指す
- [ ] AC2 印: `--director` の wait は始めたとき、30 秒ごと、終わるときに `director-wait.json` を一時ファイルと rename で書き、D2 の field を持つ。終わり方ごと(items、timeout、watch、not_current、error、signal)に `end` と終了コードが決まった値になる。終わるときは、ディスクの印の pid が自分のときだけ書き換える。印を書けなくても wait は続き、stderr に 1 回だけ書く
- [ ] AC3 登録の変化: 待っているあいだに今の登録が `--replace`、detach、stop で変わると、次の印の書き直しで `not_current` を書いて終了コード 5 で返る
- [ ] AC4 見張り: `--director` の wait は、見張りが件を受け付けていない(not_running、stale、stopping、`--once`)と分かり、その回の受信箱に open の件がなければ、終了コード 4 で返る。open の件があれば終了コード 0。stderr の 1 行は D3 の 6 つ(stopped、died、stale、stopping、once、missing)を書き分ける。見張りが動いていれば待ち続ける
- [ ] AC5 終了コード: `ralph` は D4 のとおり 4 と 5 を返し、upgrade の 3 とほかの 1 は変わらない。`--director` のない `wait --inbox` と座席の `wait` は、印を書かず、見張りを読まず、今と同じ出力と終了コードを返す(今のテストがそのまま通る)
- [ ] AC6 表示: `ralph org director status` の `wait:` の行、`ralph status` の `director:` の行(昇格したセッションのとき)、両方の `--json` の `director.wait` が、走っている wait、印を書き直していない wait、終わった wait(終わり方ごと)、印がない、別の登録の印、読めない印の各行で D5 のとおりに出る。印の値は escape される。headless の登録の `director:` の行は変わらない
- [ ] AC7 文書: `/director` skill の 4 面が同じ中身で、`--director` の wait と終了コード 4・5 の扱い(D6)を書く。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。雛形、help、README、仕様の注(8b-2b で決めたこと、8b-2a の注の続き)、tech-debt(247 行と 209 行の書き換え、B の新しい行)、AGENTS.md が新しい動きを説明する
- [ ] AC8 テストは本物の herdr、osascript、端末、Orca、人の tmux、gh、network を打たず、repo の台帳に書かない。signal は自分で起こしたプロセスにだけ送る。実験の記録と D8 の実機の確認を evidence に残す(`$HOME` は `~`、token と pane の値は伏せる)

## Implementation outline

1. S1 org: `director_wait.go`(印の型と読み書き、登録との照合、見張りの確かめと止まり方の分類、誤り)と、`WaitInbox` の director の選択(AC1〜AC4 の org の分)
2. S2 cli: `--director`、終了コードの関数と `main.go`、signal で印を書いてから終わること(AC1、AC2 の signal、AC5)
3. S3 表示: `wait:` の行、`director:` の行、`--json`、help(AC6)
4. S4 文書(AC7)
5. S5 evidence: 実験の記録と実機の確認(AC8)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、golangci-lint、secret scan)、`go vet ./...`、`GOTOOLCHAIN=go1.25.8` の build と vet、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: FR-6(見張りがないあいだの escalate は人にすぐ届く)と期限の送りが変わらないこと。FR-9 の昇格したセッションの登録、attach、detach、`--replace` が今どおり動くこと。FR-10 の push の起こし口が変わらないこと。8b-2a の確かめと印(`director-alive.json`)が変わらないこと
- Documentation drift to check: 「registration does not tell whether its session is still running」が、昇格したセッションについて wait の印と並んで正しく残っているか。skill の待ち方の段落(Claude Code と Codex)が `--director` を付け、終了コード 1 の失敗の扱いと 4・5 が食い違わないか。tech-debt の 247 行と 209 行が、この段で閉じた分と 8b-2c に残す分を書き分けているか
- Evidence to capture: 実験の記録(Claude Code の版、終わり方ごとの結果、親のつながり)、実機で打ったコマンドと出力、`director-wait.json` の行、`ralph status` の `director:` の行、終了コード。`$HOME` は `~`、token と pane の値は伏せる

## Test plan

- Unit tests:
  - 印の読み書き(field、一時ファイルと rename、壊れた印、ない印、自分の pid のときだけ終わりを書く)
  - 登録との照合の表(今の登録がない、別の id、stopped、`attached_at` の違い、昇格したセッション、headless の starting と running)
  - 見張りの止まり方の分類の表(stopped、died、stale、stopping、once、missing、running)
  - 終了コードの関数(4、5、3、1、nil)
- Integration tests:
  - `WaitInbox` の director の選択: open の件で 0、時間切れで 1、見張りなしで 4(open の件があれば 0 が先)、登録の入れ替えで 5、受信箱が読めないとき。偽の時計と短い間隔で、印の書き直しの回数と内容を確かめる
  - CLI: `--director` の断りの表、終了コード、stdout の件の行、stderr の 1 行。テストの中で起こした wait のプロセスに SIGTERM を送り、印の `end=signal` と終わり方を確かめる
  - 表示: `director status` と `ralph status` とその `--json`
- Regression tests: `--director` のない `wait --inbox` と座席の `wait` のテスト、8a〜8b-2a の attach、detach、start、stop、escalate、期限、起こす手順、確かめのテストがそのまま通る
- Edge cases:
  - 2 つの `--director` の wait が同じ登録で走る(後の印が残り、先の wait の終わりは後の印を消さない)
  - 見張りが止まった回と、件が届いた回が重なる(0 が先)
  - 見張りが `--stop` の途中(stopping)
  - 印のディレクトリに書けない(wait は続く)
  - `--timeout-ms` が印の間隔より短い
  - 登録の `attached_at` が手で書き換えられた
- Evidence to capture: `go test` の出力(go1.26 と go1.25.8)、`-race`、mutation の結果

## Risks and mitigations

- 漏れた wait(claude の SIGKILL、Move to background のあと)が印を書き続け、表示が「pid N が待っている」と出し続ける: 表示は事実だけを出し「生きている」とは言わない。B として tech-debt に書き、確かめてから 8b-2c で直す
- 人が止めた見張りを director が起動し直す: D3 で止まり方を書き分け、D6 で起動し直すのは `died` だけにする。`stopped` は Ctrl-C、SIGTERM、`--stop` の 3 つで、人か release の手順が止めた見張りである
- 起動し直せない見張り(lock を持ったまま固まった、`--once` が走っている)で、wait と起動のし直しが回り続ける: 起動し直すのは `died` だけで、そのあとも running にならなければ待つのをやめる(D6)
- 古い skill の director が新しい binary を使う: 古い skill は `--director` を付けないので、4 と 5 は返らず、動きは変わらない
- 新しい skill の director が古い binary を使う: 古い binary は `--director` を知らないフラグとして終了コード 1 で断る。skill は今どおりそれを失敗として人に伝える
- 起動し直した見張りが lock を取り、最初の印を書く前に wait が読むと `stale` に見える: watch gate と同じく `DefaultWatchGateSettle` のあいだ読み直してから返る(D3)。テストで、lock を持ち印がまだない見張りが settle の内に running を書けば wait が待ち続けることを確かめる
- 見張りの確かめが固まった見張りに気づくのは窓(既定 10 分)のあと: 窓は escalate と watch gate と同じで、D3 の文に書く。落ちた見張りは lock が外れた直後に分かる
- 印の書き込みが 30 秒ごと、見張りの確かめが 5 秒ごとに増える: どちらも小さなファイルの読み書きと flock の確かめだけ
- 2 つの wait が同じ登録で走る: 印は後の wait のもの。終わるときは自分の pid の印だけを書き換える。読み書きのあいだに別の wait が書く隙は残り、印が一時的に先の wait の終わりを出しうる。次の書き直し(30 秒)で戻る

## Rollout or rollback notes

- 既定の動きの変化はない。`--director` を付けた wait だけが印を書き、見張りを確かめ、4 と 5 を返す。skill と雛形が `--director` を付けるので、新しい skill の director から動きが変わる。release のノートに書く(新しい skill には新しい binary が要る)
- 戻すときは、そのまま revert してよい。前の ralph は `director-wait.json` を読まずに無視し、`--director` を知らないフラグとして断る(skill も戻す)

## Open questions

- なし(分かれ目は consult の判定と、Design decisions の自分の選択で決めた)

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
