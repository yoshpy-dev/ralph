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

director の受信箱の wait(`ralph org wait --inbox`)に `--director <id>` を足す。この wait は、待っているあいだ自分の pid と時刻を印のファイル(`director-wait.json`)に書き、`ralph status` と `ralph org director status` がその事実を出す。これで、昇格したセッションの登録にも「その director の wait がいつから走り、最後にいつ印を書いたか、いつどう終わったか」が付く。同じ wait は、件を受け付けていた横断の見張りが受け付けなくなったと分かれば、open の件がないときに限り決まった終了コードで返る。director はそれで見張りの停止に気づき、`/director` skill の手順で、見張りの様子の事実と起動のコマンドを人に伝え、自分では起動し直さずに待ち続ける。`--director` のない wait は今のまま変わらない。

## Scope

- `ralph org wait --inbox --director <id>`: 始める前と待つあいだ、今の登録が `<id>` かを確かめる。違えば、待たずに(または待つのをやめて)終了コード 5 で返る
- 印 `director-wait.json`(台帳の下、`director-wait.lock` の flock の中で、一時ファイルと rename で上書き): wait ごとの `wait_id`、wait の id と登録の `attached_at`、pid、始めた時刻、最後に書いた時刻、最後に見た見張りの様子、終わった時刻と終わり方
- 見張りの確かめ: `--director` の wait は `ProbeWatch` を一定の間隔で読み、件を受け付けていた見張りが受け付けなくなったと分かれば、open の件がないときに限り終了コード 4 で返る。stderr の 1 行は、lock と見張りの印の事実だけを言い、止まった原因は言わない。始めたときから受け付けていなければ、`note:` の 1 行を出して待ち続ける
- 終了コード 4 と 5 を `cmd/ralph/main.go` で返す(3 は upgrade の drift が使っている)
- 表示: `ralph org director status` の `wait:` の行、`ralph status` の `director:` の行(昇格したセッションのとき)、どちらの `--json` にも `director.wait`
- 文書: `/director` skill の 4 面(wait に `--director` を付ける、終了コード 4 と 5 と `note:` の扱い)、headless の director の雛形 `internal/org/prompts/director.md`、help、README、仕様の注、tech-debt、AGENTS.md
- evidence: Claude Code の background の task がセッションの終わり方ごとに残るかの実験(2026-10-19、計画の前に打った分)を写す。分離した台帳で、作った binary の wait、見張りの停止、登録の入れ替え、2 つの wait の印の持ち主を確かめる

## Non-goals

- wait が自分の持ち主(wait を打ったセッション)の終わりを見て止まること(consult の判定で B として切り離した)。実験では、claude の SIGKILL と `/exit` の「Move to background and exit」のあとで background の task が残り、そのシェルの親が pid 1 になった。Monitor で打った wait の親のつながりと、daemon に移ったセッションが task の終わりを受けて何をするかは確かめていない。codex は app-server の daemon が wait の親なので、この見方では分からない。tech-debt に 1 行足し、確かめたあとで 8b-2c にする
- 見張りの確かめ(`CheckDirector`、`director-alive.json`)を昇格したセッションに広げること。印は wait が書き、表示が直接読む。見張りが止まっていても印は読める
- `--director` のない `ralph org wait --inbox`、座席の `ralph org wait` の変更
- wait か director が自分の判断で見張りを起動し直すこと。見張りの印からは止め方を確かめられないので(Assumptions)、起動し直すのは人が言ったときだけにする
- 見張り(`internal/org/watch.go`)の印や止め方の変更
- 印から「生きている・死んだ」を決めること。表示は印の事実(pid、時刻、終わり方)だけを出す
- push だけの登録(wait を打たない director)の印
- 9 段目の試行

## Assumptions

- `ralph org wait --inbox` は台帳の誰でも打てる。印を書くのは `--director <id>` を付け、その id が今の登録と合う wait だけなので、leader の座席が打った `--director` のない wait は印を書かない
- `ProbeWatch`(`internal/org/watchalive.go`)は、見張りのプロセスが終わって `watch.lock` が外れると、すぐ `not_running` を返す。固まった見張りは、印の時刻が窓(`deadman_minutes`、既定 10 分、間隔の 2 倍より短くはならない)より古くなるまで `running` のまま
- 見張りの印の state からは、人が止めたのか勝手に止まったのかを確かめられない(Codex の計画助言 1)。`stopAndRelease`(`internal/org/watch.go`)は、stopped の印を書けなくても stderr に 1 行出して停止の要求を消し、lock を外す。`finish` は `stopAndRelease` を defer で呼ぶので、panic でも stopped を書く。SIGKILL、再起動、Go の fatal error では何も書かない
- Claude Code 2.1.291 の background の task は、`/exit` の既定(Exit and stop tasks)、`claude -p` の終わり、pane を閉じたとき(SIGHUP)はセッションと一緒に消え、claude の SIGKILL と「Move to background and exit」では残る(計画の前の実験。S5 で evidence に写す)

## Affected areas

- `internal/org/escalate.go`(`WaitInbox` に director の wait の選択を足す)
- `internal/org/director_wait.go`(新規: 印の型、`director-wait.lock` の下の読み書きと持ち主の確かめ、登録との照合、見張りの確かめと事実の文、誤り)
- `internal/cli/org.go`(`newOrgWaitCmd` の `--director`、`runOrgWaitInbox`、signal の扱い)
- `cmd/ralph/main.go`(終了コード 4 と 5。cli が誤りから終了コードを決める関数を持つ)
- `internal/cli/org_director.go`、`internal/cli/org_director_check.go`、`internal/cli/status.go`(`wait:` の行、`director:` の行、`--json` の `director.wait`、help)
- テスト: `internal/org/director_wait_test.go`(新規)、inbox の wait のテスト、`internal/cli` の wait のテスト、`internal/cli/org_director_test.go`、`status_test.go`
- 文書: `.claude/skills/director/SKILL.md` と写しの 3 面、`internal/org/prompts/director.md`、`README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`、`AGENTS.md`
- evidence: `docs/evidence/director-wait-session-end-2026-10-19.md`(実験)、`docs/evidence/director-wait-livefire-2026-10-19.md`(実機)

## Visual review

- Page: `.harness/state/plan-visual/org-director-wait.html`(かんたん版の図 2 枚: `overview`、`easy-branches`。折りたたみの中に `detail-overview`、`detail-flow`、`detail-mark`、`detail-watch`)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: 全体、かんたん版の 2 枚、折りたたみの 4 枚を `shot` で撮って読んだ。流れ図で生存線が 2 つの文字に重なっていたので背景を足し、撮り直して直ったことを確かめた
- Codex の計画助言を受けた直しのあと、`overview`(あなたに伝える箱)、`easy-branches`(起動し直さない 3 つの分かれ)、`detail-flow`、`detail-mark`(`wait_id`、`watch_accepting`、`director-wait.lock`)、`detail-watch`(事実の文の表)と「承認の前に」を描き直し、lint(exit 0)と 7 枚の撮り直しと読み直しをした

## Design decisions

- D1 `--director <id>`: `--inbox` のときだけ受け付け、`d<N>` の形でなければ何も読まずに終了コード 1 で断る。wait は始める前に `director.jsonl` を読み、今の登録が `<id>` で stopped でないとき(state のない昇格したセッションの登録、headless の `starting` と `running`)だけ待つ。そうでなければ、印を書かずに `ErrDirectorWaitNotCurrent` で返る(終了コード 5、文は今の登録を名指す)。待つあいだも印を書き直すたびに登録を読み直し、今の登録が `<id>` でなくなったか、`attached_at` が変わったか、stopped になったら、印に `not_current` と書いて終了コード 5 で返る(`--replace` された古いセッションの wait は、これで自分で止まる)
- D2 印 `director-wait.json`: `wait_id`(`<pid>-<始めた時刻の nanosecond>`。乱数の 16 進は secret の scan に掛かりうるので使わない)、`id`、`attached_at`(登録に結ぶ。表示は 8b-2a の `DirectorAliveFor` と同じく id と attached_at が合う印だけを読む)、`pid`、`started_at`、`seen_at`、`watch_accepting`(最後に見た見張りが件を受け付けていたか)、終わったときは `ended_at`、`end`、`detail`(短い文)。書くのは始めたとき、`directorWaitMarkInterval`(30 秒)ごと、見張りの様子が変わったとき、終わるとき
  - 書き込みはいつも `director-wait.lock` の flock の中で、印を読み、持ち主を確かめてから `writeJSONFileAtomic` で書く。始めた wait は前の印を読んでから自分の `wait_id` で上書きし、持ち主になる。そのあとの書き込みは、ディスクの印の `wait_id` が自分のときだけ行う。持ち主でなくなった wait(後から別の wait が始めた)は、stderr に 1 回だけ書き、印には二度と書かずに待ち続ける(Codex の計画助言 2。`A が始める → B が始める → A の書き直し → B の終わり` の順でも、B の終わりが残る)
  - 印を書けなくても wait は続け、stderr に 1 回だけ書く
  - `end` は次のどれか: `items`(open の件を返した、終了コード 0)、`timeout`(`--timeout-ms` の時間切れ、終了コード 1)、`watch`(D3、終了コード 4)、`not_current`(D1、終了コード 5)、`error`(受信箱や登録が読めない、終了コード 1)、`signal`(SIGINT、SIGTERM、SIGHUP。印を書いてから、その signal で終わる)。SIGKILL では書けないので、`seen_at` が古いまま残る
- D3 見張りの確かめ: `--director` の wait は、始めたときと `directorWaitWatchProbeInterval`(5 秒)ごとに `ProbeWatch` を読み、`WatchAcceptsWork` で受け付けているかを決める。`stale` のときだけは、watch gate(`NewWatchGate`)と同じく `DefaultWatchGateSettle` のあいだ読み直し、そのあいだに受け付けるようになれば受け付けているとする(起動し直した見張りが lock を取って最初の印を書く前の瞬間を、止まった見張りと取り違えないため)
  - 終了コード 4(`ErrDirectorWaitNoWatch`)で返るのは、受け付けていた見張りが受け付けなくなったときだけで、その回の受信箱に open の件があれば終了コード 0 を先にする(consult の 2 回目の判定)。「受け付けていた」は、この wait が前の回に受け付けていると見たこと、または始めたときなら、同じ登録の前の印の `watch_accepting` が true だったことを言う。前の印を引き継ぐので、Codex の 300 秒の wait の繰り返しで、wait と wait のあいだに見張りが止まっても次の wait が 4 で返る
  - 始めたときから受け付けていない(前の印がない、または前の印も false)なら、stderr に `note:` の 1 行を出して待ち続ける。受け付けていなかった見張りが受け付けるようになったら、`note:` の 1 行を出し、印の `watch_accepting` をすぐ書き直す
  - 4 の 1 行と `note:` の 1 行は、lock と見張りの印の事実だけを言い、「人が止めた」「落ちた」とは言わない: lock が外れていて印が running か stopping のまま(pid、最後に書いた時刻、「stopped の印を書かずに終わった」)、lock が外れていて印が stopped(pid、`stopped_at`)、lock が外れていて印がない、lock は持たれているが settle のあとも印が古いか running でない(pid、最後に書いた時刻、`WatchKillHint`)、止まる途中(stopping)、`--once` の見張り。どの文も `ralph org watch --all --detach` を名指し(`--state-dir` を付けたときは付ける)、見張りがないあいだの escalate は人にすぐ届くこと(`inbox_no_watch`)と、それより前に director に残された件は、見張りがまた動くまで期限で人に届かないことを言う。値は escape する
- D4 終了コード: `cmd/ralph/main.go` は、cli の関数で誤りから終了コードを決める。`ErrDirectorWaitNoWatch` は 4、`ErrDirectorWaitNotCurrent` は 5、upgrade の drift は今どおり 3、ほかは 1。`--director` のない wait は 4 と 5 を返さない
- D5 表示: 今の登録に合う印(id と attached_at)があれば、`ralph org director status` は `wait:` の行に、走っている wait なら「wait(pid N)が <started_at> に始まり、最後に印を書いたのは <seen_at>」、終わった wait なら「wait(pid N)は <ended_at> に終わった(<end>)」を出す。「director が待っている」とは書かない。走っている wait の `seen_at` が 2 回分(60 秒)より古ければ「それから印を書いていない(30 秒ごとに書く)」を足す。合う印がなければ「この登録の `wait --inbox --director` の印はない」。`ralph status` の `director:` の行は、昇格したセッションのときだけ、「登録はセッションが動いているかを表さない」のあとに同じ事実を短く足す。headless の登録の `director:` の行は今の確かめの文のまま。`--json` は `director.wait`(wait_id、pid、started_at、seen_at、watch_accepting、ended_at、end、detail)を、合う印があるときだけ載せる。印の値は今の表示と同じく escape する。印が読めないときは理由を 1 つ足し、status は失敗しない
- D6 skill と雛形: `/director` skill の 4 面は、wait の例に `--director <自分の id>` を付け、終了コード 4 と 5 と `note:` の扱いを書く
  - 4 なら、stderr の 1 行(見張りの様子の事実)と `ralph org watch --all --detach` の 1 行を人に伝え、自分では起動し直さず、wait を打ち直す(consult の 2 回目の判定)。wait も受信箱も見張りに依らないので、件の処理は続ける。人が起動し直すよう言ったときだけ、そのコマンドを打つ
  - `note:` の 1 行で見張りが受け付けていないと出たら、人に伝える。前の wait にも同じ `note:` が出ていたなら伝え直さない(Codex の 300 秒の繰り返しで、同じ知らせを人に何度も送らない)
  - 5 なら director として動くのをやめ、そのことを人に伝える(detach は打たない)
  - Codex の 300 秒の wait の繰り返しでも同じ。headless の雛形は wait の例に `--director {{ID}}` を付ける
- D7 B を切り離す: tech-debt に 1 行足す。漏れた wait(claude の SIGKILL、Move to background のあと)は、人か director が新しい wait を打つまで印を書き続け、表示は「wait(pid N)が始まり、最後に印を書いたのは t」と出し続ける。新しい wait を打てば持ち主が替わり、漏れた wait は印を書かなくなる(D2)。残る穴は、新しい wait が SIGKILL で消えたあと、漏れた wait が印なしで待ち続けること(同じ director の wait が 2 本ある時点で異常なので受け入れる)。実験の事実と、残る確かめ(Monitor の親のつながり、daemon のセッションの動き、codex の daemon)を書き、きっかけを 8b-2c にする。247 行はこの段で入った印のことに書き換え、209 行は `--director` の wait を打つ director が気づくこと、push だけの director と leader はまだ気づかないこと、誰も自動では起動し直さないことに書き換える
- D8 実機: 分離した台帳(herdr は使わない)で、作った binary を使う。shell から `ralph org director attach --driver claude` で登録し、`ralph org watch --all --detach` を立て、`wait --inbox --director d1 --timeout-ms 0` を background で打つ。そのうえで (a) status の `wait:` の行、(b) 見張りを `--stop` で止めたときの終了コード 4(lock が外れ印が stopped の文)と、打ち直した wait が `note:` を出して待ち続けること、(c) 見張りを `--detach` で立て直してから SIGKILL したときの 4(lock が外れ印が running のままの文)、(d) `--timeout-ms` の wait が時間切れで終わったあとに見張りを止め、次の wait が始めたときに 4 で返ること、(e) `attach --replace` のあとの 5、(f) SIGTERM で終わった wait の `end=signal`、(g) SIGKILL で終わった wait の古い `seen_at`、(h) 2 本目の wait を打つと 1 本目が印を書かなくなり、2 本目の終わりが残ることを確かめる

Critical forks: 8b-2b の中身と分け方、印を書く wait の区別、印の置き場所と読み手、終了コードを consult(consult-plan-promoted)に聞いた。判定は「A(印)と C(見張りの停止)を 1 本、B(持ち主の終わりで止まる)は切り離す。印は `--director <id>` を付けた wait だけが書き、表示が直接読む。CheckDirector は触らない。終了コードは 3 を避ける」で、D1〜D4、D7 に入れた。Codex の計画助言 1(見張りの印からは止め方を確かめられず、人が止めた見張りを起動し直しうる)を受けて同じ advisor にもう一度聞き、「自動では起動し直さない。4 なら事実と起動のコマンドを人に伝えて wait を打ち直す。4 は受け付けていた見張りが受け付けなくなったときだけで、始めたときに受け付けていなければ `note:` を出して待ち続ける」という判定を D3 と D6 に入れた。wait と wait のあいだの停止を拾うため、前の印の `watch_accepting` を引き継ぐこと(D2、D3)は、ユーザーの指示「計画の分かれ目も自分で選ぶ」に従って自分で選んだ。Codex の計画助言 2(印の持ち主)は D2 の flock と `wait_id` で直した。

## Acceptance criteria

- [ ] AC1 フラグ: `--director` は `--inbox` のときだけ受け付け、`d<N>` の形でない値と `--inbox` のない指定は何も読まずに終了コード 1 で断る。今の登録がその id でない(登録がない、別の id、stopped)なら、印を書かずに終了コード 5 で返り、文が今の登録を名指す
- [ ] AC2 印: `--director` の wait は始めたとき、30 秒ごと、見張りの様子が変わったとき、終わるときに、`director-wait.lock` の中で `director-wait.json` を一時ファイルと rename で書き、D2 の field を持つ。終わり方ごと(items、timeout、watch、not_current、error、signal)に `end` と終了コードが決まった値になる。印を書けなくても wait は続き、stderr に 1 回だけ書く
- [ ] AC3 持ち主: 印の書き込みは、ディスクの印の `wait_id` が自分のときだけ行う。偽の時計で `A が始める → B が始める → A の書き直し → B の終わり` の順を打つと、印は B のものが残り、B の `end` が載り、A はそれ以降一度も書かない
- [ ] AC4 登録の変化: 待っているあいだに今の登録が `--replace`、detach、stop で変わると、次の印の書き直しで `not_current` を書いて終了コード 5 で返る
- [ ] AC5 見張り: `--director` の wait は、受け付けていた見張りが受け付けなくなった(not_running、settle のあとの stale、stopping、`--once`)と分かり、その回の受信箱に open の件がなければ、終了コード 4 で返る。open の件があれば終了コード 0。前の印の `watch_accepting` が true で、始めたときに受け付けていなければ 4。前の印がないか false なら、`note:` を出して待ち続ける。lock を持ち印がまだない見張りが settle の内に running を書けば、4 にならない。stderr の 1 行は D3 の 6 つの事実を書き分け、原因を言わない: 見張りが panic で止まったときと同じ印(lock が外れ、印は stopped)と、stopped の印を書けずに止まったときと同じ印(lock が外れ、印は running)を fixture で作り、どちらの文も事実だけで、「人が止めた」「落ちた」と言わないことを確かめる
- [ ] AC6 終了コード: `ralph` は D4 のとおり 4 と 5 を返し、upgrade の 3 とほかの 1 は変わらない。`--director` のない `wait --inbox` と座席の `wait` は、印を書かず、見張りを読まず、今と同じ出力と終了コードを返す(今のテストがそのまま通る)
- [ ] AC7 表示: `ralph org director status` の `wait:` の行、`ralph status` の `director:` の行(昇格したセッションのとき)、両方の `--json` の `director.wait` が、走っている wait、印を書き直していない wait、終わった wait(終わり方ごと)、印がない、別の登録の印、読めない印の各行で D5 のとおりに出る。印の値は escape される。headless の登録の `director:` の行は変わらない
- [ ] AC8 文書: `/director` skill の 4 面が同じ中身で、`--director` の wait と終了コード 4・5 と `note:` の扱い(D6。自分では起動し直さない)を書く。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。雛形、help、README、仕様の注(8b-2b で決めたこと、8b-2a の注の続き)、tech-debt(247 行と 209 行の書き換え、B の新しい行)、AGENTS.md が新しい動きを説明する
- [ ] AC9 テストは本物の herdr、osascript、端末、Orca、人の tmux、gh、network を打たず、repo の台帳に書かない。signal は自分で起こしたプロセスにだけ送る。実験の記録と D8 の実機の確認を evidence に残す(`$HOME` は `~`、token と pane の値は伏せる)

## Implementation outline

1. S1 org: `director_wait.go`(印の型、`director-wait.lock` の下の読み書きと持ち主、登録との照合、見張りの確かめと事実の文、誤り)と、`WaitInbox` の director の選択(AC1〜AC5 の org の分)
2. S2 cli: `--director`、終了コードの関数と `main.go`、signal で印を書いてから終わること(AC1、AC2 の signal、AC6)
3. S3 表示: `wait:` の行、`director:` の行、`--json`、help(AC7)
4. S4 文書(AC8)
5. S5 evidence: 実験の記録と実機の確認(AC9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、golangci-lint、secret scan)、`go vet ./...`、`GOTOOLCHAIN=go1.25.8` の build と vet、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: FR-6(見張りがないあいだの escalate は人にすぐ届く)と期限の送りが変わらないこと。FR-9 の昇格したセッションの登録、attach、detach、`--replace` が今どおり動くこと。FR-10 の push の起こし口が変わらないこと。8b-2a の確かめと印(`director-alive.json`)が変わらないこと。`internal/org/watch.go` が変わらないこと
- Documentation drift to check: 「registration does not tell whether its session is still running」が、昇格したセッションについて wait の印と並んで正しく残っているか。skill の待ち方の段落(Claude Code と Codex)が `--director` を付け、終了コード 1 の失敗の扱いと 4・5・`note:` が食い違わないか。skill のどこにも、director が自分の判断で見張りを起動し直す手順が残っていないか。tech-debt の 247 行と 209 行が、この段で閉じた分と 8b-2c に残す分を書き分けているか
- Evidence to capture: 実験の記録(Claude Code の版、終わり方ごとの結果、親のつながり)、実機で打ったコマンドと出力、`director-wait.json` の行、`ralph status` の `director:` の行、終了コード。`$HOME` は `~`、token と pane の値は伏せる

## Test plan

- Unit tests:
  - 印の読み書き(field、一時ファイルと rename、flock、壊れた印、ない印、持ち主のときだけ書く)
  - 登録との照合の表(今の登録がない、別の id、stopped、`attached_at` の違い、昇格したセッション、headless の starting と running)
  - 見張りの事実の文の表(lock が外れ印が running・stopping・stopped・ない、settle のあとの stale、stopping、once、running)。panic と書けなかった stopped を写した fixture を含め、どの文も原因を言わない
  - 終了コードの関数(4、5、3、1、nil)
- Integration tests:
  - `WaitInbox` の director の選択: open の件で 0、時間切れで 1、受け付けていた見張りが受け付けなくなって 4(open の件があれば 0 が先)、始めたときから受け付けていなければ `note:` で待ち続ける、前の印の `watch_accepting` を引き継いで 4、settle の内に受け付けるようになれば待ち続ける、登録の入れ替えで 5、受信箱が読めないとき。偽の時計と短い間隔で、印の書き直しの回数と内容を確かめる
  - 持ち主: `A が始める → B が始める → A の書き直し → B の終わり` の順
  - CLI: `--director` の断りの表、終了コード、stdout の件の行、stderr の 1 行。テストの中で起こした wait のプロセスに SIGTERM を送り、印の `end=signal` と終わり方を確かめる
  - 表示: `director status` と `ralph status` とその `--json`
- Regression tests: `--director` のない `wait --inbox` と座席の `wait` のテスト、8a〜8b-2a の attach、detach、start、stop、escalate、期限、起こす手順、確かめのテストがそのまま通る
- Edge cases:
  - 見張りが止まった回と、件が届いた回が重なる(0 が先)
  - 見張りが `--stop` の途中(stopping)
  - 印のディレクトリに書けない(wait は続く)
  - `--timeout-ms` が印の間隔より短い
  - 登録の `attached_at` が手で書き換えられた
  - 前の印が別の登録のもの(引き継がない)
- Evidence to capture: `go test` の出力(go1.26 と go1.25.8)、`-race`、mutation の結果

## Risks and mitigations

- 漏れた wait(claude の SIGKILL、Move to background のあと)が印を書き続け、表示が「wait(pid N)が始まり、最後に印を書いたのは t」と出し続ける: 表示は事実だけを出し「生きている」とは言わない。新しい wait を打てば持ち主が替わる(D2)。B として tech-debt に書き、確かめてから 8b-2c で直す
- 人が止めた見張りを director が起動し直す: 誰も自分の判断では起動し直さない(D6)。見張りの印からは止め方を確かめられないので(Assumptions)、4 の文は事実だけを言い、起動し直すのは人が言ったときだけにする
- 見張りが止まっても自動では戻らない: 見張りがないあいだの escalate は人にすぐ届き(`inbox_no_watch`)、受信箱にも残るので件はなくならない。止まるのは期限の送りだけで、director が人に事実と起動のコマンドを伝える
- 始めたときから見張りがないと 4 を返さない: 前の印の `watch_accepting` を引き継ぐので、受け付けていた見張りが wait と wait のあいだに止まった場合は 4 になる。一度も見張りを見ていない登録は `note:` だけで、skill は `note:` を一度人に伝える(D6)
- `note:` が Codex の 300 秒の繰り返しのたびに出る: skill は前の wait にも同じ `note:` が出ていたなら伝え直さない(D6)
- 古い skill の director が新しい binary を使う: 古い skill は `--director` を付けないので、4 と 5 は返らず、動きは変わらない
- 新しい skill の director が古い binary を使う: 古い binary は `--director` を知らないフラグとして終了コード 1 で断る。skill は今どおりそれを失敗として人に伝える
- 起動し直した見張りが lock を取り、最初の印を書く前に wait が読むと `stale` に見える: watch gate と同じく `DefaultWatchGateSettle` のあいだ読み直してから決める(D3)
- 見張りの確かめが固まった見張りに気づくのは窓(既定 10 分)のあと: 窓は escalate と watch gate と同じで、D3 の文に書く。lock が外れた見張りは次の確かめ(5 秒以内)で分かる
- 印の書き込みが 30 秒ごと、見張りの確かめが 5 秒ごとに増える: どちらも小さなファイルの読み書きと flock の確かめだけ
- 2 つの wait が同じ登録で走る: 書き込みは flock の中で持ち主を確かめてから行い、印は後の wait のものが残る(D2)。先の wait は印を書かずに待ち続ける

## Rollout or rollback notes

- 既定の動きの変化はない。`--director` を付けた wait だけが印を書き、見張りを確かめ、4 と 5 を返す。skill と雛形が `--director` を付けるので、新しい skill の director から動きが変わる。release のノートに書く(新しい skill には新しい binary が要る)
- 戻すときは、そのまま revert してよい。前の ralph は `director-wait.json` と `director-wait.lock` を読まずに無視し、`--director` を知らないフラグとして断る(skill も戻す)

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
