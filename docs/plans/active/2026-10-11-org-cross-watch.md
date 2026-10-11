# org-cross-watch

- Status: Approved
- Approved: 2026-10-11 sha256:d0a6f2735552
- Owner: Claude Code
- Date: 2026-10-11
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 6 段目(FR-6 横断の見張りと受信箱の期限)の前半。ユーザーは「続けてください」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-cross-watch

## Objective

1 つのプロセス `ralph org watch --all` が、同じ台帳を使うすべての org のパルス(今の watch の stall・生存・担当範囲の変更とデッドマン)と、受信箱の件の期限を見るようにする。期限が切れた件は、今ある人への経路(`escalations.jsonl`、stderr、osascript)に送る。

見張りは台帳の下のロックを持ち続け、サイクルごとに生存の印を書く。印とロックから「見張りが動いているか」を判定する関数を用意し、`ralph status` に出す。

6 段目は 2 つの PR に分ける(ユーザーの決定)。この PR は見張りそのものを入れ、spawn・TASK の送信・start の動きは変えない。start が見張りを起動すること、見張りが動いていないときに spawn と TASK の送信を断ることは、次の PR(6 段目の後半)で入れる。

## Scope

- 設定 `[org.director]`
  - `ack_minutes`(既定 10)と `resolve_minutes`(既定 60)。どちらも 1 以上。3 面(`internal/config/config.go` の `Default()` と検証、`templates/base/ralph.toml`、`scripts/ralph-config.sh`)を揃え、lock-step のテストに足す
  - `enabled` などの director の設定は 8a 段目で足す
- 受信箱の期限(org 層)
  - ack の期限: 件が open(`state == open`)で、`escalated_at + ack_minutes` を過ぎ、`notified` がまだ 1 つもない。open からそのまま resolved になれる(`inbox.go` の Resolve)ので、「acked でない」ではなく open で判定する(consult の指摘)
  - resolve の期限: 件が resolved でなく、`escalated_at + resolve_minutes` を過ぎ、reason `inbox_resolve_deadline` の `notified` がまだない
  - 1 サイクルで両方が切れていたら、resolve の期限だけを送る(その `notified` で ack の期限の条件が外れる)
  - 判定は、件の状態と `notified` の reason と時刻だけを見る純粋な関数にする。8a 段目で escalate の側を切り替えるとき、この関数はそのまま使う
  - 送り方は 5 段目の人への経路と同じ。`escalations.jsonl` に件の ID・org_id・reason(`inbox_ack_deadline` か `inbox_resolve_deadline`)の 1 行、見張りの stderr に 1 行の表示、デスクトップ通知(org・ID・TYPE・どちらの期限か。本文は載せない)、そのあと reason と osascript の結果つきの `notified`。件の状態は変えない
  - `escalations.jsonl` の行か `notified` が書けなかった件は、同じ見張りのプロセスの中では、同じ件・同じ reason について `ack_minutes` に 1 回までしか送り直さない(書けない台帳で 30 秒ごとに通知が鳴らないため)。この覚えはメモリの中だけで、見張りを起動し直すとすぐ送り直す
  - 期限は毎サイクル `inbox.jsonl` から計算し直す。見張りを起動し直しても、切れた件は最初のサイクルで届き、送り済みの件は `notified` で重ねて鳴らない
  - 時刻の読めない件(手で書き換えた行など)は飛ばし、同じプロセスの中で件ごとに 1 回だけ stderr に注意を出す
- escalate の動き
  - 変えない。escalate は 5 段目と同じく記録と同時に人へ送り、`notified`(reason `inbox_no_director`)を書く(ユーザーの決定)。その件には ack の期限は効かず、60 分たっても resolved でなければ resolve の期限でもう一度届く。PR を作ったときの RESULT も同じなので、人は 60 分以内に `ralph org inbox resolve` する
  - 仕様の受け入れ条件「director が登録されていない、when escalate、then ack の期限が切れた時点で届く」は、director を登録できる 8a 段目で満たす。仕様の FR-6 にその注を書く
- `ralph org watch --all`
  - `--all` を必須にする。`--org-id` を付けたら、見張りは全 org を見ることと `--all` を使うことを言って断る(1 org ごとの watch は廃止。残すと `--all` と同じ org を 2 つのプロセスが見て、`watch-status-<org>.json` の書き手が 2 つになる)
  - 1 サイクル: 先に受信箱の期限を見る。次に、台帳の manifest で走っている org(`RunningOrgs`、3 段目)ごとに、今の 1 org のサイクル(stall・生存・担当範囲の変更・デッドマン)を回す。最後に生存の印を書く。受信箱の期限は herdr にも agmsg にも git にも頼らないので先に回す(Codex plan advisory の指摘 1)。org ごとの状態は今までどおり `watch-status-<org>.json`(`ralph status` が読む)。走っていない org(disband したもの)は見ない
  - 1 つの org が全体を止めないようにする(同じ指摘): パルスの外部の呼び出し(herdr の `agent get`、agmsg の Join・Send・History、`git status`)は 1 回ごとに `callWithTimeout`(`Org.DriverCallTimeout`、既定 10 秒)で打ち切る。`git status` は context つきで起動する。1 つの org の失敗(manifest の読み取り、状態ファイルの書き込み、呼び出しの打ち切り)は stderr に 1 行書いてその org の分だけ飛ばし、ほかの org と生存の印は同じサイクルで進める。今の `RunWatch` は manifest の読み取りや状態ファイルの書き込みに失敗すると全体を終えるので、`--all` の繰り返しではそうしない
  - 間隔は今までどおり `--interval-seconds`(既定 `[org.watchdog].interval_seconds`、30 秒)。`--once` は 1 サイクルで終わる
  - on-demand の watcher(LLM の判定)は今までどおり。single-flight は全 org で 1 つ
- 1 プロセスの保証と生存の印
  - 見張りは台帳の下の `watch.lock` を flock(LOCK_EX)で持ち続ける。取るときは短く(2 秒ほど)待つ。判定関数が一瞬だけ取る共有ロックと競って起動に失敗しないため。取れなければ、印の pid を添えて「ほかの `watch --all` が動いている」と言い、終了コード 1
  - 生存の印は台帳の下の `watch-alive.json`。中身は `state`(`running` か `stopped`)、`pid`、`started_at`、`ts`(最後のサイクルの時刻)、`interval_seconds`。ロックを取ったらすぐ(最初のサイクルの前に)自分の pid と `started_at` で書き、そのあと毎サイクル書く。一時ファイルと rename で書く(読む側が途中の内容を見ないため)
  - SIGINT と SIGTERM を受けたら、そのサイクルを終えてから `stopped` と `stopped_at` を書き、ロックを放して終わる。今の CLI にはシグナルの処理がないので、watch のコマンドで `signal.NotifyContext` を使う。サイクルの間の待ちは 1 秒ごとに区切り、その都度シグナルと止める頼み(`--stop`)を見る
  - 判定関数: ロックがほかのプロセスに持たれているか(LOCK_SH|LOCK_NB を一瞬取って確かめる)と、印から、`running`(ロックあり、印が running、`ts` が `[org].deadman_minutes` 以内)、`stale`(ロックあり、印がない・古い)、`not_running`(ロックなし)の 3 つを返す。次の PR の spawn と TASK の検査もこの関数を使う
- `ralph org watch --all --detach`
  - 見張りを切り離した子プロセスで起動する。`os.Executable()` の ralph を `org watch --all --state-dir <台帳>` で起動し、`--config` を指定されていればそれも、`--interval-seconds` を指定されていればそれも渡す。setsid で、stdin は `/dev/null`、stdout と stderr は台帳の下の `watch.log` に追記する
  - 子の印が `running` で pid が子のものになるまで最大 10 秒ほど待ち、pid と `watch.log` のパスを出して終了コード 0。子が先に終わったら、`watch.log` のパスを出して終了コード 1
  - 待ちきれなかったときは、自分が起動した子(pid は起動したときの値で、印からは読まない)を止めて回収し、ロックが放されたことを確かめてから、`watch.log` のパスを出して終了コード 1(Codex plan advisory の指摘 3)。失敗を返したのに子がロックを持って動き続け、次の `--detach` が `stale` で断られる形を残さないため
  - すでに `running` なら、その pid を出して終了コード 0(何も起動しない)。`stale` なら起動せず、pid と止め方を出して終了コード 1
  - 子の起動は差し替えられるようにし(`internal/cli/main_test.go` の TestMain で package 変数を設定する形)、テストが test binary を exec しないようにする(consult の指摘)
  - herdr の中では動かさない。安全の層(仕様の NFR「全 LLM を止めても、ralph の仕組みだけで期限切れの件を人に知らせる」)を herdr の server から切り離すため
- `ralph org watch --stop`
  - シグナルは送らない(Codex plan advisory の指摘 2)。印の pid は、新しい見張りがロックを取ってから印を書くまでの間は古いままなので、その pid が別のプロセスに使い回されていると、関係のないプロセスを止めてしまうため
  - 判定が `running` か `stale` なら、台帳の下に止める頼み `watch-stop.json`(`requested_at`)を書き、ロックが放されるまで最大 30 秒ほど待つ。見張りは 1 秒ごとに頼みを読み、`requested_at` が自分の `started_at` より後なら、そのサイクルを終えて `stopped` を書き、頼みを消してロックを放す。放されたら終了コード 0。放されなければ、印の pid と「自分で kill する」ことを出して終了コード 1
  - 自分の `started_at` より前の頼み(前の見張りに宛てて残ったもの)は、起動のときに消して従わない
  - `not_running` なら「動いていない」と言って終了コード 0
  - 見張りは自分では止まらない(ユーザーの決定)。止めるのはこの動詞か kill
- `ralph status`
  - 見張りの状態(`running` / `stale` / `not_running`、pid、最後のサイクルの時刻、`stopped` のときは `stopped_at`)を 1 行で出す。`--json` には上の階層に `org_watch` を足す。org ごとの `watch`(`watch-status-<org>.json` の心拍)はそのまま
- tech-debt の整理(動作を変えない。`docs/tech-debt/README.md` の org-inbox-escalate の小さな指摘の行で、6 段目を trigger にしたもの)
  - (c) `internal/cli/org.go` の `newOrgEscalateCmd` から `yesNo` までを `internal/cli/org_inbox.go` に移す(watch の変更を足す前に)
  - (d) `lockfile.go` の待ち時間の定数の doc に、受信箱のロックにも使うことを書く
  - (e) `Org.Inbox` の doc に `WaitInbox` も読むことを書く
  - (f) `RunWatch` の `escalations.jsonl` のパスを `EscalationsPathIn` で作る
  - (g) `HintStateDir` を `org.Org` のフィールドにし、`EscalateParams` と `NotifyInboxItem` / `sendInboxItemToHuman` の文字列の引数から外す。escalate の `RunE` は解決済みの台帳を渡し、`orgInboxHintStateDir` で解決し直さない。引用符の付け方は今の `shellQuote`(いつも付ける)のまま残し、`TestInboxCommand` は変えない
  - (h) `PrintableInboxText` を `inbox.go` に移す。`newOrgEscalateFallbackRuntime` と `newOrgRuntimeAt` の受信箱の 3 つのフィールドを 1 つの helper で設定する
- 文書
  - `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面): 動詞リファレンスの watch の行、長時間運用の注意(`--org-id` から `--all --detach` に)、受信箱の節(期限、escalate はすぐ届く、RESULT も 60 分以内に resolve)、`ralph status` の見張りの行
  - README と AGENTS.md の repo map(`internal/org/` の説明に、横断の見張り・受信箱の期限・生存の印)
  - 仕様の FR-6 に 6 段目の前半の注(決めたこと、受け入れ条件 103 行は 8a 段目で満たすこと、後半に残したこと)
  - `docs/quality/quality-gates.md` の watch の行、`.claude/rules/ralph/agent-messaging.md` の watchdog の記述(`ralph org watch` の形が変わるなら、写しも)
  - tech-debt: 上で片づけた行を閉じる。シグナル処理を前提に書かれた古い doc comment(`newOrgWatchCmd` の「SIGINT で context が取り消される」)が、今回の変更で正しくなることを確かめる

## Non-goals

- start が見張りを起動すること、見張りが `running` でないときに `ralph org start`・`spawn`・TASK の送信を断ること、leader の雛形と対話セッション昇格の手順の書き換え(6 段目の後半の PR)
- escalate がすぐ人へ送るのをやめること(8a 段目。director を登録できるようになってから)
- `[org.director].enabled` などの director の設定、director の登録(8a 段目)
- 期限が切れたときに座席を止めること(council の「既存処理をいつ止めるかは異常の種類ごとに決める」。どの段でも未定)
- 人が受け取ったことの新しい記録。受領は今の `acked`、試行は `notified`、座席の停止は manifest の `stopped` で、別々に記録されている
- 受信箱の JSON key `body` の名前を決めること(release の前に決める。tech-debt の (a))
- 見張りを OS のサービス(launchd など)として常駐させること

## Assumptions

- 走っている org の判定は 3 段目の `RunningOrgs` のとおり(最後の disband より後に動いている座席・閉じていない workspace・予約のどれかがある org)。disband した org の保留中の ALERT は、もう見ない(org を終えたので)
- flock は同じホストのプロセスの間で効く(manifest と受信箱のロックと同じ前提)。台帳はローカルのディスクにある
- 切り離した子プロセスのデスクトップ通知(osascript)は、利用者のログインセッションの中で動く限り表示される。実機では確かめる
- 見張りの 1 サイクルは、座席の数だけ herdr の `agent get` を呼ぶ。全体の上限 30 席で数秒に収まる見込み(確かめていない)
- 次の PR の検査は、この PR の判定関数をそのまま使う

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`、`internal/config/defaults_sync_test.go`、`templates/base/ralph.toml`、`scripts/ralph-config.sh`
- `internal/org/inbox.go`(`PrintableInboxText` の移動)、`internal/org/escalate.go`(人への経路に reason、`HintStateDir`)、新しいファイル `internal/org/deadline.go`(期限の判定と送り方)
- `internal/org/watch.go`(`--all` の繰り返し、呼び出しの打ち切り、`EscalationsPathIn`)、新しいファイル `internal/org/watchalive.go`(ロック・生存の印・止める頼み・判定)、`internal/org/lockfile.go`(doc と、持ち続けるロック)、`internal/org/spawn.go`(`Org` のフィールド)
- `internal/cli/org.go`(watch のコマンド、`--detach`・`--stop`、シグナル)、新しいファイル `internal/cli/org_inbox.go`(移動)、`internal/cli/status.go`、`internal/cli/main_test.go`(起動の差し替え)
- テスト: `internal/org/deadline_test.go`(新規)、`internal/org/watchalive_test.go`(新規)、`internal/org/watch_test.go`、`internal/cli/org_test.go`(watch のテストの書き換え)、`internal/cli/status_test.go`、`internal/cli/org_inbox_test.go`
- 文書: `/org` skill の 4 面、README、AGENTS.md、仕様、`docs/quality/quality-gates.md`、`.claude/rules/ralph/agent-messaging.md`(と写し)、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-cross-watch.html`(図 1 全体図、図 2 1 件が人に届く時点、図 3 見張りが動いているかの判定、図 4 2 つの PR の分け方)
- 自己確認: ページ全体と全体図(`--fragment overview`)を `scripts/plan-visual.sh shot` で撮って読んだ。全体図の上の余白を詰めて撮り直した。Codex の指摘を入れたあと、図 1(受信箱の期限を先に見る順、呼び出しの打ち切り、止める頼み)と図 3(`--stop` の列)を描き直して撮り直した。文字の重なり・切れ・線の通り抜けはない

## Design decisions

- 6 段目の escalate は今までどおり記録と同時に人へ送る(ユーザーの決定、案 A)。director を登録する手段が 8a 段目まで無く、期限の形にすると、どの件も人には 10 分遅れて届くため。見張りの期限は二の網として働く: 通知が終わっていない件を ack の期限で拾い、60 分たっても解決しない件を resolve の期限でもう一度送る
- 見張りが動いているかは、ロックと生存の印の両方で判定する(ユーザーの決定)。仕様は印の時刻だけを書くが、kill された見張りの印は `deadman_minutes`(10 分)新しいままなので、ロックも見て kill の直後から「動いていない」と分かるようにする。この PR では判定と表示だけで、断るのは次の PR
- 見張りは kill されるか `--stop` で止めるまで動く(ユーザーの決定)。止まる条件を持たせると、人が見張りを起動してから spawn までの間に止まって spawn が断られうるため
- 6 段目は 2 つの PR に分ける(ユーザーの決定)。見張りと期限を先に入れ、実機の受信箱で期限の動きを確かめてから、spawn・TASK・start の検査を被せる。検査は spawn・send・start を使う 100 本ほどのテストの前提を変えるので、次の PR にまとめる
- Codex plan advisory の指摘 3 件(1 つの org が全体を止める、`--stop` が関係のないプロセスを止める、`--detach` の失敗のあとに子が残る)は、ユーザーの判断で計画に入れた。受信箱の期限を先に見る順と呼び出しの打ち切り、止める頼みのファイル、起動した子の回収にした
- 見張りは herdr の外の切り離した子プロセスで動かす(consult の判定)。安全の層を herdr の server から切り離すため。stderr の表示は `watch.log` に入るので、人に届くのは osascript と `escalations.jsonl` になる
- 1 org ごとの watch は廃止する(consult の判定)。`--all` と並ぶと同じ org の状態ファイルの書き手が 2 つになるため。inbox の動詞が `--org-id` を断るのと同じ形にする
- Critical forks: 上の 4 つをユーザーに聞いた(通知の時点、spawn の検査の形、見張りの停止、PR の分け方)。すべておすすめの案を選んだ

## Acceptance criteria

- [ ] AC1: `[org.director]` の `ack_minutes` は既定 10、`resolve_minutes` は既定 60 で、`Default()`・`templates/base/ralph.toml`・`scripts/ralph-config.sh` が揃う(lock-step のテスト)。0 以下は設定の検証エラーになる
- [ ] AC2: 期限の判定は表のとおりになる。open で notified なしで ack の期限を過ぎた → ack。acked → ack の期限は効かない。notified あり(5 段目の即時の通知)→ ack の期限は効かない。resolved → どちらも効かない。resolve の期限を過ぎて resolved でなく、resolve の期限の notified がない → resolve(1 回だけ)。両方が切れている → resolve だけを 1 回。open から直接 resolved になった件 → どちらも効かない
- [ ] AC3: 期限が切れた件は、`escalations.jsonl` の 1 行(inbox_id、org_id、reason)、見張りの stderr の 1 行、デスクトップ通知(本文なし)、reason と osascript の結果つきの `notified` で人に届き、件の状態は変わらない
- [ ] AC4: `escalations.jsonl` に書けないとき、`notified` は書かれず、同じプロセスの中では `ack_minutes` に 1 回までしか送り直さない。見張りを起動し直すと、最初のサイクルで送り直す
- [ ] AC5: 期限を過ぎた件がある台帳で見張りを起動すると、最初のサイクルで届く。すでにその期限の notified がある件は、起動し直しても届かない(仕様 AC 105 行の後半)
- [ ] AC6: escalate は 5 段目と同じくすぐ人へ送る。その件は ack の期限では二度目が鳴らず、resolve の期限を過ぎても resolved でなければもう一度届く
- [ ] AC7: `ralph org watch --all` は、走っているすべての org のパルスを 1 プロセスで見る。org ごとの ALERT・デッドマン・`watch-status-<org>.json` は今の 1 org の watch と同じになる。disband した org は見ない
- [ ] AC7b: 2 つの org のうち先に見る org の herdr の呼び出しが返らないとき(テストで止める)、その呼び出しは `DriverCallTimeout` で打ち切られ、同じサイクルで受信箱の期限の件が届き、もう 1 つの org のパルスが回り、生存の印が書かれる。先に見る org の状態ファイルが書けないときも同じ(Codex plan advisory の指摘 1)
- [ ] AC8: `ralph org watch` に `--org-id` を付けると断られ、`--all` を付けないと断られる(どちらも終了コード 1、何も書かない)
- [ ] AC9: 同じ台帳に 2 つ目の `watch --all` を起動すると、1 つ目の pid を添えて終了コード 1 で終わる。1 つ目が終わる(正常終了でも kill でも)とロックが放され、次の起動が通る
- [ ] AC10: 見張りはロックを取った直後と毎サイクルに `watch-alive.json` を書き(state `running`、pid、started_at、ts)、SIGINT か SIGTERM で `stopped` を書いて終わる
- [ ] AC11: 判定関数は、ロックあり・印が新しい → `running`、ロックあり・印がないか古い → `stale`、ロックなし(印が running のままでも)→ `not_running` を返す
- [ ] AC12: `--detach` は子の印が running になってから pid と `watch.log` のパスを出して終了コード 0。すでに running なら起動しない。stale なら起動せず終了コード 1。子には `--state-dir` と、指定されていれば `--config` と `--interval-seconds` が渡る。テストは test binary を exec しない
- [ ] AC12b: `--detach` が待ちきれなかったとき、起動した子は止められて回収され、ロックが放されてから終了コード 1 になる(テストでは印を書かない子で確かめる。Codex plan advisory の指摘 3)
- [ ] AC13: `--stop` はシグナルを送らない。running か stale の見張りに止める頼みを書き、見張りが `stopped` を書いてロックを放したら終了コード 0。30 秒ほどで放されなければ pid を示して終了コード 1。動いていなければ終了コード 0 で「動いていない」と言う
- [ ] AC13b: 見張りは、自分の `started_at` より前に書かれた止める頼みには従わない(起動のときに消す)。印の pid が古いまま(ロックを取った直後の見張りを模す)でも、`--stop` はどのプロセスにもシグナルを送らない(Codex plan advisory の指摘 2)
- [ ] AC14: `ralph status` は見張りの状態を出し、`--json` には `org_watch` が入る
- [ ] AC15: spawn・send・start・stop・disband・escalate の動きは変わらない(それらの既存のテストは書き換えずに通る)
- [ ] AC16: tech-debt の (c)〜(h) の整理は動作を変えない(既存のテストが書き換えずに通る。(h) の helper は escalate の壊れた設定のテストが拾う)
- [ ] AC17: 文書(skill の 4 面、README、AGENTS.md、仕様の注、quality-gates、agent-messaging)が新しい watch の形に合い、`scripts/check-sync.sh` と `scripts/check-skill-sync.sh` が通る

## Implementation outline

1. S0 tech-debt の整理(動作を変えない): (c) の移動、(d)(e) の doc、(f)、(g)、(h)
2. S1 設定 `[org.director]` の `ack_minutes` と `resolve_minutes`(3 面と検証とテスト)
3. S2 受信箱の期限(`internal/org/deadline.go`): 判定の純粋な関数、reason つきの人への経路、1 回分の掃き出し(送り直しの間隔の覚えを含む)。テストは状態 × notified × 時刻の表
4. S3 `watch --all`: 持ち続けるロック、生存の印、判定関数(`internal/org/watchalive.go`)、全 org の繰り返し(`RunningOrgs`、org ごとのサイクル、期限、印)、シグナル、CLI の `--all` 必須と `--org-id` の拒否、`--detach`(起動の差し替え)、`--stop`。1 org の watch の CLI のテストを書き換える
5. S4 `ralph status` の見張りの行と `org_watch`
6. S5 文書と tech-debt

各スライスは implementer に渡し、1 スライス 1 コミット。S3 が大きければ、ロックと印と判定(org 層)と CLI の 2 つに分ける。

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`(gofmt、go vet、staticcheck など。スライスごと)、最後に `HARNESS_VERIFY_MODE=static RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`
- Spec compliance criteria to confirm: FR-6 の 6 項目のうち、この PR で満たすもの(1 プロセスの全 org の見張り、ack と resolve の期限と人への経路、生存の印の書き込み、再起動のあとの計算し直し、記録の分離)と、次の PR に回すもの(印が古いときの start・spawn・TASK の拒否、start の起動、見張りなしの横断の stop の確認)を AC と対応づける
- Documentation drift to check: `ralph org watch --org-id` を書いた箇所が残っていないか(skill の 4 面、README、quality-gates、agent-messaging、雛形)。`scripts/check-sync.sh`、`scripts/check-skill-sync.sh`
- Evidence to capture: verify の報告(`docs/reports/verify-2026-10-11-org-cross-watch.md`)

## Test plan

- Unit tests: 期限の判定の表(AC2)、送り方と notified(AC3)、送り直しの間隔(AC4)、判定関数の 3 状態(AC11、ロックはテストの中で別の fd で持つ)、印の書き方(rename、running と stopped)、設定の既定と検証(AC1)
- Integration tests: 2 つの org と受信箱を持つ台帳で `--all` の繰り返しを `Cycles` で回す(AC5、AC7)。先の org の herdr の呼び出しを止める・状態ファイルを書けなくする(AC7b)。2 つ目の見張りの拒否(AC9)。CLI の `watch --all --once`、`--org-id` の拒否、`--detach`(差し替えた起動で。印を書かない子で待ちきれない場合も、AC12b)、`--stop`(テストの中で動かす見張りに頼みを書く。古い頼みと古い pid の場合も、AC13b)、`ralph status --json`
- Regression tests: 既存の watch のテスト(1 org のサイクル)、escalate・inbox・wait のテスト、spawn・send・start・stop・disband のテストを書き換えずに通す(AC15、AC16)
- Edge cases: 時刻の読めない件、resolve_minutes が ack_minutes より短い設定、受信箱のファイルがない台帳、走っている org がない台帳、`escalations.jsonl` に書けない台帳、SIGTERM がサイクルの途中で届く、印が壊れている(JSON でない)とき判定が `stale` か `not_running` になる
- Evidence to capture: test の報告(`docs/reports/test-2026-10-11-org-cross-watch.md`)。実機の確認(osascript が切り離した子から表示されるか、`--detach` と `--stop`)ができれば `docs/evidence/` に残す。できなければ未確認として PR に書く

## Risks and mitigations

- テストが test binary を exec してしまう(`os.Executable()` は `go test` では test binary): 起動を package 変数で差し替え、TestMain で無害な実装を入れる。`--detach` のテストは差し替えた起動で行う
- 判定関数の共有ロックと見張りのロックの取り合い: 見張りはロックを取るとき短く待つ。判定関数は LOCK_SH|LOCK_NB で一瞬だけ取る。テストで競らせる
- 書けない台帳で通知が鳴り続ける: 同じ件・同じ reason の送り直しを `ack_minutes` に 1 回までにする(AC4)
- disband した org の保留中の ALERT を見なくなる: org を終えたので見ないと決めた。Assumptions に書いた
- RESULT の件が 60 分で 2 回目を鳴らす: 受信箱の節に「PR を作った RESULT も resolve する」と書く(ユーザーに伝えて選んでもらった)
- 1 org の watch の廃止で、今 1 org の watch を動かしている人の手順が壊れる: release ノートに「更新の前に `ralph org watch` を止め、更新のあとは `ralph org watch --all --detach`」と書く(#208・#209 と同じ release の注意に足す)
- 見張りの 1 サイクルが長くなる(座席の数だけ herdr を呼ぶ): 30 席でも数秒の見込み。test の段で 1 サイクルの時間を測れれば測る
- 1 つの org の呼び出しが返らず、全 org の期限の通知と生存の印が止まる(Codex plan advisory の指摘 1): 受信箱の期限をパルスより先に見る。外部の呼び出しを 1 回ごとに `DriverCallTimeout` で打ち切る。org ごとの失敗はその org の分だけ飛ばす(AC7b)。打ち切りが続く org があると、1 サイクルは席の数 × 10 秒まで延びうる。その間も受信箱の期限は先に済んでいるが、生存の印が `deadman_minutes` を越えて遅れれば `stale` になる。それは「見張りが働いていない」という正しい知らせとして扱う
- `--stop` が関係のないプロセスを止める(同じ指摘 2): シグナルを送らず、台帳の頼みのファイルで見張りに自分で止まってもらう(AC13、AC13b)。応えない見張りは pid を示して人に任せる
- `--detach` が失敗を返したのに子が動き続ける(同じ指摘 3): 自分が起動した子だけを止めて回収する(AC12b)

## Rollout or rollback notes

- 戻すときは、先に `ralph org watch --stop` で見張りを止め(止まらなければ印の pid を kill し)、`ralph status` で `not_running` を確かめてから、この PR を revert したバイナリに入れ替える。revert だけでは、動いている見張りのプロセスは止まらない(Codex plan advisory の指摘 3)
- 台帳に残る `watch-alive.json`・`watch.lock`・`watch-stop.json`・`watch.log` は、古いバイナリが読まないので害はない。受信箱に書かれた期限の `notified` は、5 段目のバイナリでも `notified` として読める(reason が違うだけ)
- 1 org の watch を廃止するので、revert するまで `ralph org watch --org-id` は使えない
- release ノート: 「更新の前に、動いている `ralph org watch` を止める。更新のあとは `ralph org watch --all --detach` で起動する」

## Open questions

- 切り離した子プロセスの osascript の通知が、実機で表示されるか(確かめるのは test の段か実機の確認)
- 次の PR(6 段目の後半): start の起動と spawn・TASK の拒否、対話セッション昇格の手順、leader の雛形。この PR の判定関数をそのまま使う

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
  - S0(00e9da99): tech-debt (c)〜(h) の整理。escalate と inbox のコマンドを `internal/cli/org_inbox.go` に移し、`Org.InboxHintStateDir` を足し、受信箱の 3 つのフィールドは `wireOrgInbox` で設定する。internal/cli のテストは無変更。S5 で tech-debt の行を閉じるとき、(a) の `inboxSummary` の場所を `org_inbox.go` に直す
  - S1(9cabdeb4): `[org.director]` の `ack_minutes`(10)と `resolve_minutes`(60)、3 面と検証。`deadman_minutes` の古い「予約」の説明も直した
  - S2(b50d7e03): `internal/org/deadline.go`。`InboxDeadlineDue`(純粋な判定)、`SweepInboxDeadlines`、`InboxDeadlineMemory`。計画からの足し: 設定の分が 1 未満の Org では掃き出しを断る(設定なしで組んだ Org が全件を一度に送らないため)。デスクトップ通知の題は期限のときだけ "ralph org watch"
  - S3 は 2 つに分ける: S3a(org 層。ロック・生存の印・止める頼み・判定・`RunWatchAll`。CLI は今の `RunWatch` のまま)、S3b(CLI を `RunWatchAll` に切り替え、`--all`・シグナル・`--detach`・`--stop`、1 org の `RunWatch` を消してテストを移す)
  - S3a(996e5c48): `internal/org/watchalive.go`(`AcquireWatchLock`、`ProbeWatch`、印と止める頼み)と `RunWatchAll`。計画からのずれ: `started_at` はロックを待つ前に決め、起動のときに消す頼みは `started_at` より前のものだけ(ロックを取ってから消すまでの間に書かれた `--stop` を消さないため)。時刻はナノ秒つき。呼び出しの打ち切りは loop の ctx ではなく Background から作る(SIGINT がサイクルの途中で来ても偽の liveness ALERT を出さないため)。そのぶん、herdr が止まっていると止まるまでがサイクルの終わりまで延びる。足したこと: manifest の org_id を識別子として確かめてから状態ファイル名にする、読めない状態ファイルの org はそのサイクルだけ飛ばす
  - S3b(33f6383b): watch のコマンドを `internal/cli/org_watch.go` に移し、`--all` 必須・`--org-id` の拒否・SIGINT と SIGTERM・`--detach`(起動は `orgWatchLaunch` で差し替え、TestMain は起動すると panic する偽物)・`--stop`(頼みのファイル、config は読まない)。1 org の `RunWatch` と `WatchParams` を消し、そのテスト 3 本は `RunWatchAll` に書き換えた。`TestOrgWatch_RequiresOrgID` は消し、`--org-id` の場合は `TestOrgWatch_RefusesFlagCombinationsAndWritesNothing` に移した(`--org-id` は必須から拒否に変わった。self-review L7 で記録を直した)。デッドマンの osascript にも 10 秒の上限。範囲外の 1 行(`org_legacy_ledger_test.go` の watch の引数を `--all --once` に)は orchestrator が許した
  - S4(04f9237b): `ralph status` に `org watch:` の行、`--json` に `org_watch`。deadman は `./ralph.toml`(読めなければ既定の 10 分と stderr の警告)。implementer が一時の台帳で `--detach` と `--stop` を実機で試し、プロセスが残らないことも確かめた。残る穴: `--config` で別の `deadman_minutes` を渡した見張りとは、status の判定がずれうる(S5 で tech-debt に書く)
  - S5(505c58c1): skill の 4 面、README、AGENTS.md、仕様の FR-6 の注、quality-gates、tech-debt((c)〜(h) を閉じ、新しい 4 行)。agent-messaging.md は今も正しいので変えていない。配る側の `templates/base/docs/quality/quality-gates.md` の watch の行は orchestrator が直した(fb52a143、check-sync の KNOWN_DIFF のファイル)
  - 実装は終わり。次はパイプライン(cycle 1/2)
  - self-review(6a6976d2): Merge yes、MEDIUM 1・LOW 7。1b64ae2b で M1 と L1・L2・L4・L6・L7 を直し、L3 は一部を直した(状態ファイルの保存は atomic にした。読めないファイルの警告は 1 回だけになったが、その org を黙って飛ばし続ける点は再実行の N2 として tech-debt に書いた)。L5(切り離した子の cwd)は tech-debt に書いた。計画からのずれ(M1): 生存の印の判定の窓は、Scope と AC11 に書いた「`deadman_minutes` 以内」ではなく、`deadman_minutes`(1 未満なら既定の 10)と印の `interval_seconds` の 2 倍の大きいほう。`deadman_minutes = 0`(デッドマンは時間切れで上げない、の意味)の台帳で `--detach` が健全な子を止めないため。設定で 0 を拒む案は、0 を使う設定を壊すので採らなかった。status の stale の文言もこの規則に合わせた(orchestrator が直した)。self-review を回し直す
- [x] Review artifact created
  - self-review の回し直し(7afda04c): Merge yes、LOW 5(N1〜N5)。94d5248c で N1・N4・N5 を直し、N2・N3 は tech-debt に書いた。94d5248c はコメントと使われない引数だけなので self-review を通していない(verify が読んで、新しい問題はないとした)
- [x] Verification artifact created
  - verify(a411bb57): pass。AC17 は一部(skill の 4 面と仕様の FR-6 の注に窓の古い規則が残る。sync-docs に回す)。記録のずれ 3 件は妥当と判定
- [x] Test artifact created
  - test(fd35c1eb): pass。1,368 本で失敗 0、-race も 0。一時の台帳で `--detach`・`kill -9` のあとの判定と起動し直し・2 つ目の見張りの拒否・`--stop` を実機で確かめた(`docs/evidence/org-cross-watch-live-2026-10-11.md`、Linux のコンテナでも)。mutation 82 件のうち生き残った本物の穴 5 件(P2・X3・L6・A6・S6)は、4aaef4ed でテストを足して塞いだ(本番のコードは無変更、テストのあとの追加なので test の報告の Test gaps に追記)。本物の osascript は確かめていない
- [ ] PR created
