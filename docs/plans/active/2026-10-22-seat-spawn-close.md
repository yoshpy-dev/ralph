# seat-spawn-close

- Status: Approved
- Approved: 2026-10-22 sha256:550d25183df2
- Owner: Claude Code
- Date: 2026-10-22
- Related request: 試行の所見を直す系列の 4 本目の後半「座席の環境 4b」(ユーザーの依頼 2026-10-21「2. 修正可能なものは修正してください」)。consult(consult-plan-seatenv4b)の判定で、4b は F4 と記録、F11 は 4c に分けた
- Related issue: N/A
- Type: fix
- Branch: fix/seat-spawn-close

## Objective

spawn が途中で失敗したとき、その spawn が作った herdr の pane を閉じて、herdr の agent の名前 `<org_id>_<seat_id>` を空ける(所見 F4)。今の補償は pane に C-c を送るだけなので、pane と名前が残り、同じ座席の次の spawn が `agent_start` で `agent_name_taken` になる。HELLO が時間切れでも agent は動いていることがあり、試行では台帳が `spawn_failed` のまま座席が仕事をした。閉じれば、台帳の `spawn_failed` と実際が一致する。閉じられなかった pane は、stop の `stop_failed` と同じく座席を active のまま残し、`ralph org stop` と `stop --all` と次の spawn が拾えるようにする。あわせて、spawn と start の `--timeout-ms` の help が「per-step」と書いて実装(spawn 全体で 1 つの期限)と食い違うのを直し、時間切れの原因を試行の manifest の時刻で記録に書く。

## Scope

- spawn の失敗の補償(`internal/org/spawn.go` の `failStepWithNote` と `compensatePane`)で、その spawn がいま作った pane に C-c を送ってから `herdr pane close` を打つ。どちらも 1 回ごとに `driverCallTimeout`(既定 10 秒)の期限で打ち、spawn の期限(すでに切れていることがある)は使わない。herdr が pane を知らない(not found)ときは閉じ済みとする
- HELLO(`agmsg_announce`)の失敗の Leave(`compensateLeave`)も、同じ 10 秒の期限で打つ(今は期限なし)
- 閉じられなかったとき(not found 以外の失敗。時間切れを含む): `spawn_failed` を書かず、新しい non-state の event `spawn_close_failed` を書く。座席は active(spawn の途中)のまま残る。返すエラーに pane の id、手で閉じる `herdr pane close <id>`、`ralph org stop --org-id <o> --seat <s>` を書く
- 前の spawn の途中で止まった座席の補償(`compensateStale`): pane を `confirmSeatPane` で確かめてから C-c と close を打つ。確かめられない(get の失敗・時間切れ)か close が失敗したときは、新しい spawn を断り(`spawn_started` を書かない)、`spawn_close_failed` を書き、座席を active のまま残す。label が違うとき(記録した id が別の pane を指す)は、その pane に触らずに `spawn_failed` を書いて新しい spawn に進む
- 座席ごとの spawn のロック: dry run でない spawn は、台帳を読む前から返るまで、その座席(org_id と seat_id)のロックファイルを持つ。同じ座席の spawn がロックを持っている間に打たれた spawn は、何も書かずにすぐ断る。動いている spawn の途中の記録を stale と取り違えて閉じないため(計画の Codex 助言 HIGH 2)
- 途中の座席(`spawn_started` か `spawn_step` が最後)への再試行が、エンベロープ・permission・`--scope` の検査で断られるときは、`rejected` も receipt も書かない(`spawned` の座席への `idempotentRespawn` と同じ規則)。`rejected` は state event で pane_id を持たないので、書くと閉じられなかった pane が台帳から消える(計画の Codex 助言 HIGH 1)
- spawn と start の `--timeout-ms` の help を「spawn 全体の期限で、herdr と agmsg のすべての step が分け合う」に直す。期限切れが原因の失敗のエラーに、その期限の値と `--timeout-ms` を上げる案内を足す。`/org` skill(と写し)の同じ説明を合わせる
- 分離した herdr server での実機の確かめ(下の S4)。結果を `docs/evidence/seat-spawn-close-2026-10-22.md` に書く
- 記録: tech-debt の F4 の行(原因を manifest の時刻で書き、"the row above" の指し先を直す)、根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md`、仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の F4 が理由の印と注、記録に残る「fix/seat-agmsg-self-name」を「#248」に、仕様の Status と PR 一覧(31 本)

## Non-goals

- 4c の範囲: F11(座席の pane の中で session end の WIP コミットが走る)
- spawn の期限の仕組みを step ごとの期限に変えること、既定の 60000 を変えること(D2)
- manifest の書き込みの失敗で spawn が止まる道(`appendEvent` が失敗して `SpawnOutcomeFailed` を返す箇所)の補償。今までどおり spawn の途中のまま残り、次の spawn の `compensateStale` が拾う
- 古い binary が残した `spawn_failed` の座席の pane(試行の pane-9 のようなもの)を、新しい binary が見つけて閉じること
- 動いている spawn の途中で `ralph org stop` か `stop --all` がその座席の pane を閉じること(今もある競合。spawn のロックを stop が見る直しは、tech-debt の行に残す)
- HELLO が届いたのに時間切れになった場合に、leader の受信箱から HELLO を消すこと
- agmsg が負荷の下で遅い原因を突き止めること(試行の環境は残っていない)
- 系列の残り(5 分割計画の前文、6 見張りの escalation、7 busy な leader、8 文書のまとめ)と release

## Assumptions

- `TabCreate` が返した pane の id は、同じ spawn の中では spawn が作った pane を指す。herdr が数秒のうちに状態を失って id を振り直すことは考えない(consult の判定)。だから failStep の道では label を確かめない
- herdr の `pane close` は、pane の中のプロセスを終わらせ、その pane の agent の名前を空ける。試行で leader が pane を閉じたあと、4 回目の spawn は `agent_start` を通った(manifest の pane-9)。S4 でもう一度見る
- `stateEvents` にない event は、座席の状態を変えない(`internal/org/seat.go` の `stateEvents` の doc)。`stop_failed` がこの形で座席を active に残している
- 試行の pane-5 の spawn は既定の 60000 で打たれた(manifest の時刻が合う。D2 の下の計算)。pane-9 の spawn に渡した `--timeout-ms` は記録にない

## Affected areas

- `internal/org/spawn.go`(`failStepWithNote`、`compensatePane`、`compensateLeave`、`compensateStale`、Spawn の stale の分岐)
- `internal/org/seat.go`(`EventSpawnCloseFailed` の定数と doc)
- `internal/org/lockfile.go`(既存の `acquireHeldFileLock` を使う。必要なら座席のロックの名前を決める小さな関数を足す)
- `internal/cli/org.go`(spawn と start の `--timeout-ms` の help)
- `internal/org/spawn_test.go`、`internal/org/verbs_test.go` か `internal/org/verbs_all_test.go`(stop と stop --all が残った座席を拾うこと)、`internal/cli` の help のテスト
- `.claude/skills/org/SKILL.md`(と `.agents/skills/`・`templates/base/` の写し)の spawn と start の行、`--timeout-ms` の説明
- `docs/evidence/seat-spawn-close-2026-10-22.md`(新規)
- `docs/tech-debt/README.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、`docs/specs/2026-10-07-org-multi-org-director.md`

## Visual review

- Page: `.harness/state/plan-visual/seat-spawn-close.html`(今の形。かんたんな図 3 枚: `overview`、`easy-left-open`、`easy-time`。折りたたみの中: `detail-overview`、`detail-fail`、`detail-stale`、`detail-live`)
- Lint: `./scripts/plan-visual.sh lint` が rc 0
- Self-check: 全体と 7 枚の図を `shot` で撮って読んだ。図 2 の 1 つ目の箱の小さい文字が箱の中身と合っていなかったので直し、lint と図 2 の撮り直しをした。計画の Codex 助言(座席のロック、途中の座席への拒否)を受けて「承認の前に」と `detail-overview`・`detail-stale` を直し、lint と全体・その 2 枚の撮り直しをした。どの PNG にも文字の重なりと切れはない

## Design decisions

- D1 閉じられなかった pane の記録: `spawn_failed` を書いて警告だけ出す案(座席は active でなくなり、stop も `stop --all` も拾わず、次の spawn は名前が取られたまま落ちる)ではなく、座席を active のまま残す案を採る。仕様 FR-2 が stop に決めた「pane を閉じられなかった座席は `stop_failed` を書いて動いているまま残すので、打ち直せば拾える」と同じ形にするため。記録には新しい non-state の event `spawn_close_failed` を使う。`spawn_step` に失敗の文を書く案は、途中経過の event に失敗が混じり、event の一覧を読む人が誤読するので採らない。non-state の event は Roster が読まない(`stateEvents` の doc)ので、座席の状態は最後の `spawn_step` のまま active になる
- D2 期限の仕組み: spawn 全体で 1 つの期限(`--timeout-ms`、既定 60000)を残し、help とエラーの案内を直す。step ごとの期限に変える案は、spawn 全体の長さの上限がなくなり(step が 6 つあると最長 6 倍)、leader が spawn を待つ時間が読めなくなるので採らない。既定を上げる案は、試行の遅さの原因が分からないまま全員の待ち時間を延ばすので採らない。根拠の時刻: pane-5 は `spawn_started` 03:41:00 から座席の join の戻り 03:41:53 まで 53 秒で、HELLO に残ったのは約 7 秒、`spawn_failed` は 03:42:06(66 秒後)。pane-9 は join の戻りが 03:45:44(113 秒後)で、既定の 60 秒では join が先に切られる(driver は期限で子を止め、WaitDelay は 2 秒)ので、この spawn にはもっと大きい `--timeout-ms` が渡されたと考える(推定。記録にない。`/org` skill が「座席には有界タイムアウト(`--timeout-ms`)を必ず設定する」と書いており、leader が値を付けた筋は通る)
- D3 stale の補償の確かめ: `compensateStale` は台帳から読んだ pane を扱うので、stop と同じく `confirmSeatPane` で label を確かめてから C-c と close を打つ。今は確かめずに C-c を送っており、herdr が id を振り直したあとでは別の pane に C-c が届きうる。確かめられないとき(get の失敗・時間切れ)と close が失敗したときは、pane が残りうるので、座席を active に残して新しい spawn を断る。label が違うときは、その pane に触らずに `spawn_failed`(Details に別の label で触っていないこと)を書いて進む。stop が同じ場合に `--force` を求めるのは「止まった」と記録する責任があるからで、spawn はその責任を負わない。label が違えば元の pane はもう無いことが多く、名前が空いているかは次の `agent_start` が `agent_name_taken` で分かる(consult の指摘。断る案は、herdr が状態を失ったあとの再 spawn に毎回 `stop --force` を要る形にする)
- D5 動いている spawn と止まった spawn の見分け: 座席ごとのロックファイル(台帳の directory の下、org_id と seat_id から決まる名前。`_` はどちらの識別子にも使えないので `<org_id>_<seat_id>` で一意)に、既存の `acquireHeldFileLock`(`internal/org/lockfile.go`)で LOCK_EX を取り、spawn が返るまで持つ。待たずに 1 回だけ試し、取れなければ断る。flock はプロセスが終われば OS が外すので、ロックが取れたときの途中の記録は、終わった spawn のもの(落ちたか、閉じられずに返った)と分かる。spawn の記録に pid を書いて生死を読む案は、pid の使い回しと起動時刻の読み取りが要るので採らない。待ってから冪等に返す案は、最長の待ちが倍になるので採らない(計画の Codex 助言 HIGH 2)
- D6 途中の座席への拒否: エンベロープ・permission・`--scope` の検査は stale の補償より前にあり、今は `reject()` が `rejected` を書く。途中の座席に対しては `idempotentRespawn` と同じく何も書かずに断り、エラーに途中の座席があることと `ralph org stop` を添える。検査の順を変えて補償を先にする案は、Phase 1 と Phase 2 のロックの組み立てを変えることになるので採らない(計画の Codex 助言 HIGH 1)
- D4 補償の順と期限: HELLO の失敗では、今の順(Leave、C-c、close)を変えず、3 つとも 10 秒ずつの期限で打つ。最悪で spawn の期限のあと約 30 秒(と WaitDelay)で返る。Leave を期限なしで打つ今の形は、agmsg が返らないと spawn が返らない
- Critical forks: D1 と D2 は計画の分かれ目で、ユーザーの指示(critical forks は自分で選ぶ)により上のとおり選んだ。D3 と D4 は stop の既存の規則に合わせ(D3 の label が違う場合だけ、上の理由で stop と変えた)、1 スライスの中で戻せる。D1〜D4 は consult(consult-plan-seatenv4b)の判定を受けた。D5 と D6 は計画の Codex 助言の HIGH 2 件を受けて足したもので、既存のロックの部品と `idempotentRespawn` の規則を使う

## Acceptance criteria

- [ ] AC1: pane を作ったあとの step(`agent_start`、`prompt_file`、`agmsg_join`、`agmsg_announce`)で spawn が失敗すると、その pane に C-c を送り、`PaneClose` を打つ。どちらも spawn の期限ではなく 1 回ごとの `driverCallTimeout` の期限で打ち、spawn の期限が切れていても打たれる。label の確かめはしない。`PaneClose` が not found なら閉じ済みとする。`spawn_failed` の Details の `compensation=` は C-c の結果と `pane=closed` か `pane=already closed` を持つ。pane を作る前の失敗(`workspace_create`、`tab_create`)と dry run は今までどおり `no pane to compensate`
- [ ] AC2: HELLO の送信が期限切れで失敗したとき(届いたかどうか分からない場合)も AC1 と同じく pane を閉じて `spawn_failed` を書く。Leave は `driverCallTimeout` の期限で打たれる
- [ ] AC3: AC1 の `PaneClose` が not found 以外で失敗したとき(時間切れを含む)、`spawn_failed` を書かず、`spawn_close_failed`(座席の欄、pane_id、Details に step・error・補償の結果)を書く。座席は Roster で active のまま残る。返すエラーは pane の id、`herdr pane close <id>`、`ralph org stop --org-id <o> --seat <s>` を含む。そのあと `ralph org stop`(1 座席)と `ralph org stop --all` が、台帳の `spawn_step` に記録された pane_id でその座席の pane を閉じて `stopped` を書く(テストは stop が使った pane_id を見る)
- [ ] AC4: `compensateStale` は `confirmSeatPane` のあとでだけ C-c と close を打つ。pane がない(not found)か閉じたら、今までどおり `spawn_failed` を書いて新しい spawn に進む。label が違うときは C-c も close も打たずに `spawn_failed`(Details に別の label で触っていないこと)を書いて進む。確かめられない(get の失敗・時間切れ)か close が失敗したときは、新しい spawn を断り(`spawn_started` も `rejected` も書かない)、`spawn_close_failed` を書き、座席は active のまま残る。エラーは pane の id、`herdr pane close <id>`、`ralph org stop --org-id <o> --seat <s>` を含む。pane の記録がない stale の座席は今までどおり
- [ ] AC5: spawn と start の `--timeout-ms` の help が、spawn 全体の期限で herdr と agmsg の step が分け合うことを書く。期限切れが原因の spawn の失敗のエラーは、その値(ms)と `--timeout-ms` を上げる案内を含む。`/org` skill(と写し)の spawn と start の説明と `--timeout-ms` の項が同じことを書く
- [ ] AC6: 分離した herdr server と agmsg の写しで、新しい binary の `ralph org spawn` を打ち、`docs/evidence/seat-spawn-close-2026-10-22.md` に書く。(a) HELLO の送信だけを期限より長く止める stub(本物の send.sh で送ってから眠る)で、spawn が失敗し、`spawn_failed` の Details が `pane=closed` で、`herdr pane list` にその pane がなく、`herdr agent get <org_id>_<seat_id>` が not found を返す。(b) stub を外して同じ座席を spawn すると、`agent_start` を通って `spawned` になる。(c) `ralph status` の org にその座席が 1 行だけ active で出る。(d) stop で片付ける。(e) 閉じられなかった場合に `ralph status` で座席がどう見えるか(fake の herdr を使うテストの出力で 1 行)を書く。ユーザーの herdr server、`~/.config/herdr`、Orca に触れていない
- [ ] AC7: 記録を合わせる。tech-debt の F4 の行は、AC1〜AC6 が通ったときだけ RESOLVED にし、原因を D2 の時刻で書き(推定は推定と書く)、"the agmsg self-rename of the row above" を F3 の行の名前と「#248 で直した」に直す。根拠の表と仕様の FR-2、FR-4、FR-2 の受け入れ条件、FR-4 の受け入れ条件を `[x]` にし(仕様 22/10 から 26/6、根拠の表 21/10 から 25/6)、注に AC1〜AC6 のテスト名と evidence を書く。F13 も絡む NFR の「安全の不変条件」と、ALERT も絡む FR-3 は `[ ]` のまま、F4 の部分の注を直す。両方の文書の「fix/seat-agmsg-self-name」を「#248」に、PR 一覧に #248 を足し(31 本)、仕様の Status に「試行の所見 F3 の直し: #248」と、この PR の分を足す。根拠の表の「試行の所見」の前文の予定の順(「4a が F3、4b が F4・F11」)を、4b が F4、4c が F11 に直し、merge した PR の並びに #248 を足す
- [ ] AC8: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0。変えたパッケージの Go のテストが `GOTOOLCHAIN=go1.25.8` でも通る
- [ ] AC9: dry run でない spawn は、その座席のロックを台帳を読む前から返るまで持つ。同じ座席の spawn がロックを持っている間に打たれた spawn は、台帳にも receipt にも何も書かず、herdr も agmsg も打たずに、ほかの spawn が動いていることを言うエラーで断られる。決定的なテスト: 1 つ目の spawn を fake の agmsg の join で止めたまま、同じ座席の 2 つ目を打つと断られ、1 つ目の pane には C-c も close も届かない。止めを外すと 1 つ目は `spawned` になる。違う座席と違う org の spawn はこのロックで待たない(今の並行のテストが通る)。dry run はロックを取らない
- [ ] AC10: 途中の座席(`spawn_close_failed` のあとと、落ちた spawn のあと)への再試行が、エンベロープ・permission・`--scope` の検査で断られるとき、`rejected` も receipt も書かれず、座席は pane_id を持ったまま active で残る。テストは 3 つの続きを見る: 閉じられなかったあと許されない model で再試行し、(1) 1 座席の stop がその pane を閉じる、(2) `stop --all` がその pane を閉じる、(3) 正しい再試行が AC4 のとおり補償する

## Implementation outline

1. S1(implementer): AC1〜AC3。`compensatePane` を C-c と close にし、1 回ごとの期限で打つ。`compensateLeave` に期限。`EventSpawnCloseFailed` と、閉じられなかったときの記録と返すエラー。テスト(失敗する step ごと、期限が切れた spawn の補償、close の失敗と時間切れ、not found、そのあとの stop と stop --all)。今の 2 つのテストが Details の `C-c sent` を見ているので新しい文に直す
2. S2(implementer): AC4・AC9・AC10。座席ごとの spawn のロック、途中の座席への拒否を書かない形、`compensateStale` の確かめと、断る道と、label が違うときに触らず進む道。テスト(ロックを持つ spawn と重ねた spawn、確かめて閉じる、not found、label が違う、get の失敗、close の失敗、pane の記録がない、不正な再試行のあとの stop・stop --all・正しい再試行)。stale の座席を seed する今のテストは、fake の herdr に pane と label を足して、新しい規則で同じことを確かめる(弱めない)
3. S3(implementer): AC5。help、期限切れのエラーの案内、`/org` skill と写し(`scripts/sync-skills.sh`)。help のテスト
4. S4(orchestrator が打つ): AC6。分離した herdr server(`/tmp` の短い directory、XDG の 3 変数、`HERDR_SOCKET_PATH`、偽の HOME、`env -i`)、agmsg の写し(HELLO の stub)、scratch の台帳と設定(permission は guarded)、見張り(`watch --all --detach`)。座席の agent は、herdr が受け付けるなら PATH の先頭に置いた偽の `claude`(眠るだけ)、受け付けなければ 4a の (e) と同じく本物の claude(haiku)を scratch の cwd で起動する
5. S5(implementer、sonnet): AC7。あわせて、動いている spawn の途中で stop がその座席の pane を閉じうること(Non-goals)を tech-debt の新しい行に書く

## Verify plan

- Static analysis checks: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh`(gofmt、go vet、check-sync、check-skill-sync ほか)
- Spec compliance criteria to confirm: AC1〜AC10。印の数(仕様 26/6、根拠の表 25/6)。`spawn_close_failed` が `stateEvents` と `activeEvents` に入っていないこと
- Documentation drift to check: `docs/specs/2026-08-01-org-runtime.md` の FR-9(spawn の saga と補償の記録)、`/org` skill の spawn と stop の行、`.claude/rules/ralph/agent-messaging.md`。`stop_failed` を説明している箇所に、spawn の同じ形を書くべきところがないか
- Evidence to capture: run-verify の rc と時間、evidence の文書

## Test plan

- Unit tests: S1 と S2 の上の場合。期限の切れた spawn の context でも C-c と close が打たれる(fake の herdr が受け取った context の期限を見る)。close の時間切れは `driverCallTimeout` を短くした Org で確かめる
- Integration tests: `internal/cli` の spawn と start の help の文。spawn の失敗のエラーが stdout の `spawn failed:` に出る
- Regression tests: 今の spawn、stop、stop --all、disband のテストが通る。dry run の失敗が `no pane to compensate` のまま
- Edge cases: `PaneClose` が not found、C-c だけ失敗して close は通る、close が時間切れ、stale の座席の pane が label の違う pane を指す、stale の座席に pane の記録がない、`spawn_close_failed` のあとの同じ座席の spawn、同じ座席の spawn が重なる、途中の座席への不正な再試行、ロックファイルの directory が作れない
- Evidence to capture: `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` を go1.26 と go1.25.8 で

## Risks and mitigations

- HELLO が leader の受信箱に届いたあとで時間切れになると、leader は閉じた座席から HELLO を受け取る → 台帳の `spawn_failed` を正とする。leader の雛形は spawn の結果を見て次の手を決めるので、HELLO だけで座席を使い始めることはない。evidence の (a) で HELLO が届いたかを記録する
- `compensateStale` が新しい spawn を断る場合ができる(herdr に繋がらない、close が失敗する) → エラーに `herdr pane close` と `ralph org stop` の打ち方を書く。label が違う場合は断らないので、herdr が状態を失ったあとの再 spawn は今までどおり通る
- label が違う stale の座席を `spawn_failed` にして進むと、元の pane が別の id で残っていた場合に名前が取られたままになる → 次の `agent_start` が `agent_name_taken` で落ち、その spawn の補償が自分の pane を閉じて `spawn_failed` を書く。残った pane は今までと同じく人が閉じる(Details に label の違いを残す)
- 座席ごとのロックで、今まで「stale の補償をしてから進む」になっていた同じ座席の同時の spawn が、あとのほうの拒否に変わる → 冪等に返したいときは、1 つ目が返ってから打ち直せば `spawned` の座席への冪等の返事になる。エラーにそう書く。ロックファイルは座席ごとに 1 つ台帳の directory に残るが、中身はなく、数は座席の数まで
- 途中の座席への不正な再試行が台帳に残らなくなり、`ralph org report` の timeline でその試行が見えない → receipt も書かないのは `idempotentRespawn` と同じ。返すエラーは stdout と stderr に出る
- `ralph status` は閉じられなかった座席を途中の spawn(active)として出すだけで、close の失敗は出さない → 返すエラーと Details と `ralph org report` の event timeline で分かる。AC6 の (e) で見え方を記録する
- 失敗の道に 10 秒の呼び出しが 1 つ増え(close)、Leave にも期限が付くので、負荷の下で Leave が 10 秒で終わらず、agmsg の team に座席が残る → Details に `leave=failed` が残る。同じ id の次の join は既存の member として通る(agmsg の join は冪等)
- 新しい event が、event の一覧を種類で読む箇所(見張りの leader の活動の数え方、report)で扱われない → non-state の event は Roster が読まない作りなので状態は変わらない。見張りの活動の数え方には入れない(`spawn_started` がすでに同じ spawn を数える)。report の timeline はどの event も同じように出す
- S4 でユーザーの herdr server や Orca に触れる → コマンドはすべて `env -i` に scratch の `HERDR_SOCKET_PATH` と偽の HOME を渡して打ち、ORCA の変数を渡さない。始める前に socket の path を読み直す。本物の claude を使うときは scratch の cwd、guarded、haiku にし、終わったら pane と server を閉じる

## Rollout or rollback notes

- 下流では、次の release の binary で効く。scaffold の skill の文は `ralph upgrade` で入る
- release のノートに書くこと: spawn が失敗すると、その spawn が作った pane を閉じる。閉じられなかった座席は active のまま残り、`ralph org stop` で片付く(新しい event `spawn_close_failed`)。`--timeout-ms` は spawn 全体の期限
- 戻すときはこの PR を revert する。`spawn_close_failed` を書いた台帳は、古い binary でも読める(non-state の event は無視される)

## Open questions

- 偽の `claude` を herdr の `agent start` が受け付けるか(S4 で試す)
- HELLO の時間切れの本当の原因(負荷の中身)は分からないまま残る。F4 の行は、補償の直しと実機の確かめが通れば閉じ、原因の推定は行に残す

## Progress checklist

- [x] Plan reviewed(consult-plan-seatenv4b: 進めてよい。D3 の label が違う場合は断らずに進む形に変え、AC3 のテストで stop が `spawn_step` の pane_id を使うことを見る、AC6 に (e)、AC7 に根拠の表の前文を足した)
- [x] Plan approved(ユーザーの指示「作成したプランは自動承認します」による。計画の Codex 助言の HIGH 2 件(途中の座席への不正な再試行が `rejected` を書いて閉じられなかった pane を台帳から隠す、動いている spawn の途中の記録を stale と取り違えて閉じる)は、どちらもコードで確かめて、Scope、D5・D6、AC9・AC10、S2、Risks、図解ページに反映した)
- [x] Branch created
- [x] Implementation started
  - S1 80780740(AC1〜AC3)。範囲の外では、`internal/cli/org_test.go` の 1 件を新しい補償の順(C-c のあとに close)を見る形に直した(弱めていない)
  - S2 97782369(AC4・AC9・AC10)。計画にない追加: stale の座席の記録した pane が打った本人の pane(`HERDR_PANE_ID`)のときは、確かめも C-c も close もせず、何も書かずに断る(stop の自分の pane の規則と同じ。閉じると打ったセッションが終わるため)。`ralph org start --plan` の事前検査も、途中の座席への拒否で Spawn と同じ文を出す
  - S3 abb11e8c(AC5)。期限の案内は、step が返った直後の spawn の ctx の `Err()` で決める(agent start の再試行を期限が打ち切ると、原因が DeadlineExceeded を包まないため)
  - S4(AC6): `docs/evidence/seat-spawn-close-2026-10-22.md`。(a)〜(e) に加え、同じ座席の重ねた spawn をロックが断ること (f) も実機で見た
  - S5 a7133901(AC7)。仕様は 26/6、根拠の表は 25/6。tech-debt には新しい行を 3 つ足した(動いている spawn の途中の stop、`spawn_close_failed` の座席の status の空の欄、manifest の書き込みが失敗した道が pane を閉じないこと)。org-stop-all の行の (b) は S2 の label の確かめで直ったので RESOLVED にした
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
