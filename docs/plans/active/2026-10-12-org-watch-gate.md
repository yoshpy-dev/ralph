# org-watch-gate

- Status: Approved
- Approved: 2026-10-12 sha256:de5ca6b7e6f0
- Owner: Claude Code
- Date: 2026-10-12
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 6 段目(FR-6 横断の見張りと受信箱の期限)の後半。前半は PR #222(dc41c4f5)で merge 済み。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-watch-gate

## Objective

見張りのいないところで自律の作業が始まらないようにする。横断の見張り(`ralph org watch --all`)が `running` でないとき、`ralph org spawn` と TASK の `ralph org send` を断る。`ralph org start` は、見張りが動いていなければ自分で起動してから進み、起動できないとき(ロックを持つ見張りの印が古い、起動した見張りが running にならない)は断る。

見張りは前半と同じ判定関数 `ProbeWatch`(ロックと生存の印)で読む。横断の stop と disband、escalate と受信箱の動詞は、見張りがいなくても今までどおり使える。leader の雛形と現セッション昇格の手順を、この拒否に合わせて書き換える。

## Scope

- 見張りの検査(org 層)
  - `Org` に検査の差し込み口を足す(たとえば `WatchGate func() error`)。nil なら検査しない。org 層のテスト(約 230 本)は今の前提のまま通る
  - `Spawn` は、入力の検査(`checkSpawnInput`)のあと、manifest に何かを書く前(dry-run の枝の前、`withManifestLock` の前)に検査する。`start <task>` と `start --plan` の leader も `Spawn` を通るので同じ検査を受ける
  - `StartFeature` は、worktree を作る前(`worktrees.Ensure` の前)に検査する。断ったときは worktree もブランチも作らない
  - `Send` は、本文の TYPE が `TASK` のときだけ検査する。型付きのメッセージは `protocol.Parse` の結果で、`--raw` のメッセージも `protocol.Parse` で 1 行目の TYPE を読み(検証はしない)、`TASK` なら検査する。1 行目が `TYPE:` でない raw のメッセージは TASK ではないので検査しない。検査は dry-run の枝(`sent` を書く)の前
  - 通すのは、`ProbeWatch` が `running` で、印が `--once` の見張りのものでないときだけ(下の「止まりかけと `--once` の見張り」)。`stale` のときは、ほかの見張りがロックを取ってから最初の印を書くまでの間かもしれないので、最大 2 秒ほど判定し直し、`running` になれば通す
  - 断ったときのエラーは、見張りの状態ごとに次の手を言う。`not_running` なら「`ralph org watch --all --detach` で起動する」、`stale` なら印の pid と「`ralph org watch --stop` で止めるか、pid を確かめてから kill する」、`stopping` と `--once` なら「その見張りが終わってから打ち直す」。stderr のエラーで終了コード 1。manifest にも receipt にも書かない(Design decisions)
- 止まりかけと `--once` の見張り(Codex plan advisory の指摘 1)
  - 印の状態に `stopping` を足す。見張りは最後のサイクルを終えて後始末(`drainBeforeStop`、watcher の待ち)に入るときに `stopping` を書き、後始末の間の書き直しも `stopping` で書き、最後に `stopped` を書いてからロックを放す。後始末の間はもう次のサイクルがないので、新しい作業を受け付けない
  - `ProbeWatch` は、ロックが持たれていて印が `stopping` で窓の内なら、新しい状態 `stopping` を返す(今は `stale` に読まれる)。`ralph status` は `stopping` を出し、`--json` の `org_watch.state` も `stopping` になる。`--stop` は今までどおりロックが放されるのを待つ
  - `--once` の見張りは印に `once: true` を書く。1 サイクルで終わるので、`running` でも新しい作業を受け付けない。`ralph status` の行に `(--once)` を足す
  - 2 つ目の `watch --all` はロックで断られるので、6a の cross-review #1(後始末の間もロックを持つ)の保証は変わらない
- CLI の配線
  - `spawn`・`start`・`send` の 3 つのコマンドだけが `Org.WatchGate` を設定する。`newOrgRuntimeAt` には入れない(stop・disband・escalate・inbox・watch も同じ関数で runtime を作るため)
  - 検査は、台帳の下の `ProbeWatch(<台帳>, now, <deadman>)` が `running` かを見る。`<deadman>` は、印に記録された見張り自身の `deadman_minutes`、なければ呼んだ側の設定の値(下の「生存の印に deadman を書く」)
  - テストの差し替え口: package 変数(たとえば `orgWatchGateOverride`)。本番では nil。`internal/cli/main_test.go` の TestMain が「いつも running」にし、CLI の既存のテスト(約 90 本)が見張りなしで今までどおり通るようにする。検査そのもののテストは、自分で差し替え口を外すか、ロックと印を置いて本物の判定を通す
- `ralph org start` が見張りを起動する
  - `start` の 2 つの形(`start --plan`、`start <task>`)はどちらも、runtime を作ったあと、何かを書く前に見張りを確かめる。`running` ならそのまま進む。`not_running` なら、`--detach` と同じ仕組み(`orgWatchLaunch`、`startDetachedProcess`)で見張りを起動し、印が running になるのを最大 10 秒待ってから進む。起動したことは stderr に 1 行(pid と `watch.log` のパス)
  - `stale` なら起動せず、最大 10 秒、`ProbeWatch` が `running` になるのを待つ(ほかの start か `--detach` の子がロックを取って最初の印を書く前かもしれない。Codex plan advisory の指摘 2)。`running` になれば進み、ならなければ上の `stale` のエラーで断る(仕様の受け入れ条件「横断の見張りを kill した、when start、then 生存の印が古ければ拒否される」)
  - `stopping` か `--once` の見張りが動いていれば、起動せず、その見張りが終わってから打ち直すよう言って断る(ロックが持たれているので起動できない)
  - 起動した見張りの待ちは、どの pid の見張りでも `ProbeWatch` が `running` になれば成功とする。起動した子が先に終わっても、ほかの子(同時の start か `--detach`)の見張りが `running` なら進む(tech-debt 206 行)。10 秒で `running` にならなければ、起動した子がまだ動いていれば止めて回収してから、`watch.log` のパスを添えて断る
  - start が起動する見張りには `--config` を絶対パスで渡し(指定されたときだけ)、`--state-dir` は解決した台帳を渡す。`--interval-seconds` は start にないので渡さない(設定の値で動く)
- `ralph org watch --all --detach` の直し(start と同じ仕組みを使うので、同じ PR で直す)
  - 子が先に終わり、`ProbeWatch` がほかの見張りの running を返すときは、`already running (pid N)` を出して終了コード 0(tech-debt 206 行。`TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild` は、ほかの見張りの印を自分の子のものと取り違えないことを今までどおり確かめ、終了コードの期待だけを変える)。起動したかどうかの表示は pid で分ける(自分の子なら `started`、ほかなら `already running`)
  - 最初の判定が `stale` なら、start と同じく最大 10 秒 `running` を待ち、`running` になれば `already running (pid N)` で終了コード 0、ならなければ今の `stale` のエラー。`stopping` か `--once` の見張りなら、起動せず終了コード 1(今は `--once` の見張りに `already running` と答える)
  - 子の作業ディレクトリは、台帳の main worktree のルート(`git worktree list --porcelain` の先頭。1 段目の FR-1 と同じ求め方)。repo の外で `--state-dir` を明示したときなど、求められなければ今までどおり打った場所。子の環境から `HERDR_PANE_ID`・`HERDR_TAB_ID`・`HERDR_WORKSPACE_ID`・`HERDR_ENV` を外す(ほかの変数は残す。herdr の server の場所を XDG の変数で分けている環境があるため)。`--config` は絶対パスにしてから渡す(tech-debt 205 行)
  - 断られる 2 つ目の見張りが、断られる前に起動の banner を出さないようにする。banner はロックを取ったあとに出す(`WatchAllParams` に、ロックを取ったあとに呼ぶ口を足す。tech-debt 204 行の (e))
- 生存の印に deadman を書く(tech-debt 201 行)
  - 見張りは印に自分の `deadman_minutes`(設定の値そのもの。1 未満もそのまま)を書く。`ProbeWatch` は、印に値があればそれで窓を決め(1 未満は 10 に読み替える規則は今のまま)、なければ呼んだ側の値を使う。前半の版が書いた印(値がない)も読める
  - `ralph status`・`--detach`・`--stop`・検査のどれも、同じ規則で判定する
- 横断の stop など、見張りなしで使える動詞
  - `stop`・`stop --all`・`disband`・`disband --all`・`escalate`・`inbox` の動詞・`wait`・`read`・`status`・`report`・`watch` は検査しない。テストで、見張りがない台帳(差し替え口を外した状態)でこれらが今までどおり動くことを確かめる(仕様 FR-6「横断の stop(FR-2)は、見張りが動いていなくても使える」)
- 雛形と手順
  - leader の雛形(`internal/org/prompts/leader.md`): `spawn` か TASK の `send` が「見張りが動いていない」で断られたら、打ち直さず、見張りを自分で起動せず(leader は herdr の中で動く)、`ralph org escalate` で BLOCKED(TASK_ID は org_id)を上げて止まる。`prompts_test.go` に語句の検査を足す。今の固定の語句(`## 運用規律` など)は崩さない
  - `/org` skill(4 面): `spawn`・`send`・`start`・`watch` の行に検査と起動を書く。「(A) 現セッション昇格」の手順は、座席を立てる前に `ralph org watch --all --detach` を打つと書く。長時間運用の節の「今は `ralph org start` も `spawn` も見張りを起動しないので、自分で起動する」を書き換える(start が起動する、spawn と TASK は断られる)
  - README、AGENTS.md(必要なら)、仕様の FR-6 に「6 段目の後半で決めたこと」の注(F4 の TASK の範囲、F7 の受け入れ条件の読み方を含む)、`docs/quality/quality-gates.md`(root と template。見張りに触れていれば)、tech-debt の 201・205・206 行と 204 行の (e) を閉じ、見張りが途中で落ちたときの穴(Design decisions の最後)を 1 行足す

## Non-goals

- `spawn` が見張りを起動すること(仕様の FR-6 で起動するのは start だけ。leader は herdr の中で動き、見張りは herdr の外で動かす)
- 7 段目の承認の検査(`[org.approval].mode`、per-plan の計画が未承認なら TASK を断る)。TASK の検査の差し込み口は 7 段目でも使える形にするが、承認は見ない
- 検査で断ったことの新しい manifest のイベント(Design decisions)
- 見張りを OS のサービス(launchd など)として常駐させること
- 見張りが返らない呼び出しを打ち切る仕組みの変更(tech-debt 202 行)、watcher の取り消しの記録(204 行の (d))、読めない状態ファイルの扱い(204 行の (c))
- 受信箱の JSON key `body` の名前を決めること(release の前に決める)
- escalate がすぐ人へ送るのをやめること(8a 段目)

## Assumptions

- 前半の判定関数 `ProbeWatch` と、ロック・生存の印・止める頼みの形(`internal/org/watchalive.go`)を使い、印に欄(`deadman_minutes`、`once`)と状態(`stopping`)を足し、`ProbeWatch` に状態 `stopping` を足す。新しい版は前半の版の印も読める(欄がなければ今の読み方)
- 見張りを起動できる環境(ralph のバイナリが `os.Executable()` で見つかる、台帳に書ける)で start が打たれる。起動できない環境では start が断られるのは正しい動き
- 検査と spawn の間に見張りが落ちることはありうる(検査は時点の判定)。その場合、座席は見張りなしで動き出し、次の start・spawn・TASK か、人が `ralph status` を見るか、見張りを起動し直すまで誰も気づかない(Design decisions の最後の穴)。仕様の求めは「見張りがいないと分かっているときに始めない」までと読む
- `protocol.Parse` は 1 行目が `TYPE: <値>` でなければ `ErrMissingType` を返し、値は大文字小文字をそのまま返す(`internal/org/protocol/protocol.go`)。TASK かどうかは `TrimSpace` した値の完全一致で見る

## Affected areas

- `internal/org/spawn.go`(`Spawn` の検査の位置)、`internal/org/feature.go`(`StartFeature`)、`internal/org/verbs.go`(`Send`)、`internal/org/org.go` か同等の `Org` の定義(差し込み口の欄)、新しいエラー型(たとえば `internal/org/watchgate.go`)
- `internal/org/watchalive.go`(印の `deadman_minutes`、`ProbeWatch` の窓)、`internal/org/watch.go`(印に書く、ロックのあとの口)
- `internal/cli/org.go`(spawn・start・send の配線、start の見張りの起動)、`internal/cli/org_watch.go`(起動の共通化、子の作業ディレクトリと環境、206 の直し、banner)、`internal/cli/main_test.go`(TestMain の差し替え口)、`internal/cli/status.go`(印の deadman で判定)
- テスト: `internal/org/` の検査と印のテスト、`internal/cli/org_test.go` か新しい `org_watch_gate_test.go`、`org_watch_test.go`、`status_test.go`
- 雛形: `internal/org/prompts/leader.md`、`internal/org/prompts_test.go`
- 文書: `/org` skill の 4 面、README、仕様 FR-6 の注、`docs/tech-debt/README.md`、quality-gates(必要なら)

## Visual review

- Page: `.harness/state/plan-visual/org-watch-gate.html`(図 1 全体図、図 2 start の流れ、図 3 見張りの状態ごとの通り方、図 4 検査を入れる位置)
- Self-check: 全体のページと全体図を `scripts/plan-visual.sh shot` で撮って読んだ。図 4 の左の列の文字が枠に触れていたので列を広げ、全体図の群の間のラベルの背景を白にして撮り直した。Codex plan advisory の指摘で計画を直したあと、図 2(stale の待ち、stopping と --once の拒否)と図 3(状態の列に stopping を足した)を描き直し、全体図の ProbeWatch と印の説明を直して、全体と全体図を撮り直した。どちらもきれい
- Codex plan advisory: 指摘 2 件(HIGH 1: 止まりかけと `--once` の見張りが検査を通す、MEDIUM 1: 最初の印の前の `stale` で同時の start を断る)。ユーザーの指示どおり聞かずに計画を直した(Scope の「止まりかけと `--once` の見張り」、start と `--detach` の `stale` の待ち、Design decisions の F9・F10、AC13〜AC15)

## Design decisions

前半から引き継ぐユーザーの決定:

- 見張りが動いているかは、ロックと生存の印の両方で判定する(`ProbeWatch`)。kill された見張りはロックが放されるので、すぐ `not_running` になる
- 見張りは kill か `--stop` まで動き続ける。start が起動した見張りも同じで、start は止めない

この計画で決めたこと(critical forks は consult に聞いて自分で選んだ。ユーザーの指示「計画の分かれ目も自分で選ぶ」。consult は 8 つすべてに同意した):

- F1 検査で断った spawn は manifest に記録しない(stderr のエラーと終了コード 1 だけ。receipt も書かない)。`rejected` は座席の最新の状態になる state event で、立っている座席の打ち直しで状態を上書きしうる。デッドマンの `leaderActivityEventCount` も数えてしまう。入力の検査(`checkSpawnInput`)、`StartFeature` の前置き、打ち直しの拒否も記録を書かないので、それと揃える。leader は BLOCKED を escalate するので、受信箱には残る。退けた案: `rejected` を書く(上の 2 つの危険)、新しいイベント(種類と report・status の扱いが増える)
- F2 検査は org 層の差し込み口(`Org.WatchGate`、nil で検査しない)で行い、CLI の spawn・start・send だけが設定する。`Spawn` は `checkSpawnInput` の直後の 1 か所、`StartFeature` は `worktrees.Ensure` の前に自分で呼ぶ(そのあと `Spawn` の中でもう一度呼ばれるが、時点の判定なので害はない)、`Send` は TASK のときだけ。退けた案: CLI の 4 か所だけで検査する(`start --plan` で worktree の前に入れる位置と、send の TYPE の判定を CLI で重ねることになる)
- F3 `--dry-run` の spawn と send も断る。dry-run は本番が通るかの予測なので、見張りの状態も含めて同じ答えを返す
- F4 断るのは TYPE が TASK のすべての send で、宛先の役割を問わない。仕様は「実装の TASK」と書くが、reviewer への TASK も自律の作業を始めるので、見張りの対象に入れる。7 段目は implementer への TASK に承認の検査を重ねる(仕様の FR-6 の注に書く)
- F5 raw のメッセージは `protocol.Parse` で 1 行目の TYPE を読み、`TrimSpace` した値が `TASK` に完全一致なら検査する。1 行目が `TYPE:` でなければ TASK ではないとして通す。小文字の `type: task` や先頭の空行は通るが、raw は manifest に `raw=true` が残るデバッグ用で、型付きの TASK はすべて検査する。退けた案: raw はすべて見張りを要る(デバッグの中継まで止まる)、大文字小文字を無視して探す(protocol の Parse と別の読み方が増える)
- F6 start と `--detach` が起動する子は、台帳の main worktree のルートで動き(求められなければ打った場所)、`--config` は絶対パスで渡し、環境から `HERDR_PANE_ID`・`HERDR_TAB_ID`・`HERDR_WORKSPACE_ID`・`HERDR_ENV` だけを外す。start と `--detach` の両方で同じにする。`--config` を渡さない `--detach` が読む `ralph.toml` は、打った worktree のものから main worktree のものに変わる(3 段目の「全体の上限は main の `ralph.toml`」と揃う)。退けた案: start のときだけ変える(2 つの起動で子の動きが分かれる)、cwd を台帳のディレクトリにする(`ralph.toml` が見つからない)
- F7 仕様の受け入れ条件「横断の見張りを kill した、when `ralph org start` を打つ、then 生存の印が古ければ拒否される」は、こう読む。kill された見張りはロックが放されて `not_running` になるので、start は見張りを起動して進む(FR-6「start は見張りが動いていなければ起動する」)。断るのは、ロックを持ったまま印が古い `stale`(返らない見張り)のとき。6a のユーザーの決定(ロックで kill の直後から動いていないと分かる)の帰結で、受け入れ条件の字義(kill → 拒否)は変わる。仕様の注、PR の本文、最後の報告にこの読み替えを書く
- F8 tech-debt の 201(印に deadman を書く)、204 の (e)(banner をロックのあとに)、205(子の作業ディレクトリと環境)、206(同時の `--detach` の終了コード 1)を、この PR で一緒に直す。どれも start の起動か `ProbeWatch` の判定に触れ、6a の tech-debt の Trigger が「6 段目の後半」を指している。206 は `--detach` の側で直す
- F9(Codex plan advisory の指摘 1 で直した): 新しい作業を受け付ける見張りと、ロックを持つ見張りを分ける。後始末の間の見張りは印に `stopping` を書き、`ProbeWatch` は新しい状態 `stopping` を返す。`--once` の見張りは印に `once: true` を書く。検査はどちらも通さない。6a で後始末の間も印を `running` に保ったのは「`--stop` と 2 つ目の見張りに、動いていると見せる」ためで、2 つ目の見張りはロックで、`--stop` はロックの解放の待ちで、今までどおり守られる
- F10(Codex plan advisory の指摘 2 で直した): ロックを取ってから最初の印を書くまでの間は `stale` に見える。start と `--detach` は最初の判定が `stale` でも、最大 10 秒 `running` を待ってから断る。spawn と TASK の検査は最大 2 秒。本当に返らない見張りへの拒否は、その分だけ遅れる
- 残す穴(tech-debt に 1 行足す): 見張りが途中で落ちても、次の start・spawn・TASK か人が `ralph status` を見るまで、誰も気づかない。検査は時点の判定で、走っている座席は止めない

## Acceptance criteria

- [ ] AC1: 見張りが `not_running` か `stale` の台帳で `ralph org spawn` を打つと、状態ごとの次の手(起動の仕方か止め方)を添えて終了コード 1 で断られ、manifest(`rejected` も含む)・receipt・herdr・agmsg のどれにも触れない。`--dry-run` も同じ。`running` なら今までどおり進む
- [ ] AC2: 見張りが `running` でない台帳で、TYPE が `TASK` の `ralph org send` は断られる(型付きのメッセージも `--raw` も、`--dry-run` も)。`TASK` 以外の TYPE と、1 行目が `TYPE:` でない raw のメッセージは、見張りがいなくても今までどおり送れる
- [ ] AC3: 見張りが `not_running` の台帳で `ralph org start`(`--plan` の形と `<task>` の形)を打つと、見張りが起動し、印が running になってから leader が立つ。stderr に起動した見張りの pid と `watch.log` のパスが出る。起動した見張りは start が終わったあとも動き続ける
- [ ] AC4: 見張りが `stale` のまま 10 秒たつ台帳で start を打つと、見張りを起動せず、worktree もブランチも作らず、`stale` のエラーで終了コード 1。起動した見張りが 10 秒で running にならないときは、子を止めて回収してから終了コード 1(テストは印を書かない子で確かめる)
- [ ] AC5: 起動した子が先に終わっても、`ProbeWatch` が running を返すなら、start は進み、`--detach` は `already running (pid N)` を出して終了コード 0。ほかの見張りの印を自分の子のものと取り違えない(pid の照合は今のまま)
- [ ] AC6: start と `--detach` が起動する子は、台帳の main worktree のルートで動き(求められなければ打った場所)、`--config` は絶対パスで渡され、環境に `HERDR_PANE_ID`・`HERDR_TAB_ID`・`HERDR_WORKSPACE_ID`・`HERDR_ENV` がない(テストは起動の差し替え口で引数・作業ディレクトリ・環境を確かめる)
- [ ] AC7: 見張りは印に自分の `deadman_minutes` を書き、`ProbeWatch` はそれで窓を決める。値のない印(前半の版)は呼んだ側の値で読む。`ralph status` は `./ralph.toml` と違う `--config` で動く見張りも、その見張りの値で判定する
- [ ] AC8: 見張りのいない台帳で、`stop`・`stop --all`・`disband`・`disband --all`・`escalate`・`inbox` の動詞・`wait`・`read`・`status`・`report` が今までどおり動く(検査の差し替え口を外したテスト)
- [ ] AC9: 同じ台帳に 2 つ目の `watch --all` を起動すると、起動の banner を出さずに断られる(stdout に banner がない)。1 つ目の banner は今までどおり出る
- [ ] AC10: CLI の既存のテストは、TestMain の差し替え口のもとで書き換えずに通る。org 層の既存のテストは、差し込み口が nil のまま書き換えずに通る。書き換えてよいのは 6a の見張りのテストのうち、この計画が変える動きを固定しているものだけで、どれも期待を変えた理由を Progress checklist に書く: AC5 の 1 本の終了コード、後始末の間の印を `running` と見るもの(`TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` など。`stopping` に変わる)、`--once` の見張りに `--detach` が `already running` と答えるもの(あれば)
- [ ] AC11: leader の雛形は、見張りの拒否で打ち直さず、見張りを起動せず、BLOCKED を escalate すると書き、`prompts_test.go` がその語句を固定する。今の固定の語句は崩れない
- [ ] AC12: 文書(skill の 4 面、README、仕様の注、tech-debt)が新しい動きに合い、`scripts/check-sync.sh` と `scripts/check-skill-sync.sh` が通る
- [ ] AC13: 見張りが後始末に入ると(テストでは `WatchHooks.BeforeStop` を待たせる)、印は `stopping` になり、ロックは持たれたまま、`ProbeWatch` は `stopping` を返す。その間、spawn と TASK の send は断られ、start と `--detach` は起動せずに終了コード 1、2 つ目の `watch --all` はロックで断られ、`ralph status` は `stopping` を出す。後始末が終わると `stopped` を書いてロックを放す
- [ ] AC14: `--once` の見張りは印に `once: true` を書き、その見張りが動いている間、spawn と TASK の send は断られ、start と `--detach` は起動せずに終了コード 1。`ralph status` の行に `(--once)` が出る
- [ ] AC15: ほかの見張りがロックを取ってから最初の印を書くまでを遅らせたテストで、その間に打った start は、断らずに待ち、印が `running` になってから進む(`--detach` は `already running` で終了コード 0)。spawn の検査は 2 秒以内に `running` になれば通る

## Implementation outline

1. S1 org 層の検査: `Org` の差し込み口と検査のエラー、`Spawn`・`StartFeature`・`Send`(TASK、raw の TYPE)の検査の位置、印の `deadman_minutes`・`stopping`・`once` と `ProbeWatch` の窓と `stopping` の状態。テストは org 層で、差し込み口を設定したものだけ
2. S2 CLI の検査の配線: 差し替え口と TestMain、`spawn`・`send` の配線、見張りなしで使える動詞のテスト、`ralph status` の判定
3. S3 start の起動と `--detach` の直し: 起動の共通化(作業ディレクトリ、環境、`--config` の絶対パス)、start の起動と待ち、206 の直し、banner をロックのあとに
4. S4 雛形と文書: leader の雛形と `prompts_test.go`、skill の 4 面、README、仕様の注、tech-debt

各スライスは implementer に渡し、1 スライス 1 コミット。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint)、`scripts/check-sync.sh`、`scripts/check-skill-sync.sh`
- Spec compliance criteria to confirm: FR-6 の残り 4 項目(start の起動、印が古いときの start・spawn・TASK の拒否、見張りなしの横断の stop、雛形と昇格の手順)を AC と対応づける。仕様の受け入れ条件「横断の見張りを kill した、when start、then 生存の印が古ければ拒否される」は、kill された見張り(ロックなし、`not_running`)では start が見張りを起動して進み、ロックを持ったまま印が古い見張り(`stale`)では断る、と読む(Design decisions)
- Documentation drift to check: skill の 4 面の `spawn`・`send`・`start`・`watch` の行と昇格の手順と長時間運用の節、README の org runtime の節、仕様 FR-6 の注、tech-debt の 201・204・205・206 行
- Evidence to capture: 一時の台帳での実機確認(見張りなしの spawn の拒否、start が見張りを起動する、kill のあとの start、`stale` の start の拒否、同時の start)を `docs/evidence/` に

## Test plan

- Unit tests: 検査の判定(running / stale / not_running ごとのエラーの文)、`Send` の TYPE の読み方(型付き、raw の TASK、raw の非 TASK、`TYPE:` のない raw)、印の `deadman_minutes` と窓(値あり、値なし、1 未満)
- Integration tests: CLI で、見張りなしの spawn・send・start の拒否と、見張りがいるときの成功(本物のロックと印)。start のテストは差し替え口(`orgWatchGateOverride`)を外して本物の `ProbeWatch` を通す(TestMain の「いつも running」のもとでは起動の経路を通らないため)。start が起動の差し替え口で子を起動し、印が running になってから進む。子が先に終わり別の見張りが running のとき。待ちきれないとき
- Regression tests: CLI の既存のテスト(TestMain の差し替え口)、org 層の既存のテスト(差し込み口が nil)、`TestOrgWatchDetach_*`、`TestRenderRolePrompt_*`
- Edge cases: `--dry-run` の spawn と send、raw の `TYPE: TASK` の前後の空白、`StartFeature` で worktree を作らないこと、見張りなしでの `stop --all` と `disband --all`、start が起動した見張りが自分の start より長く動くこと、後始末の間(`stopping`)と `--once` の見張り、最初の印の前の `stale`、`deadman_minutes` のない前半の印と `stopping` を知らない前半の版の印
- Evidence to capture: `go test ./...`、`-race`(org と cli)、mutation(検査の位置、TYPE の判定、206 の分岐)、実機確認

## Risks and mitigations

- 既存のテストが一斉に落ちる: CLI の約 90 本と org 層の約 230 本が spawn・send・start を使う。org 層は差し込み口を nil にして触れず、CLI は TestMain の差し替え口で「running」にする。差し替え口を外すテストだけが本物の判定を通る
- 見張りの起動で start が遅くなる: 起動は `not_running` のときだけで、待ちは最大 10 秒
- 起動した見張りが残る: start は見張りを止めない(ユーザーの決定「止めるまで動く」)。`ralph status` で見え、`--stop` で止められる
- 検査がすり抜ける: raw の TYPE を `protocol.Parse` で読むので、1 行目が `TYPE:` でない raw は TASK とみなさない。TASK の語を書いた raw を見張りなしで送れる抜け道になるが、raw は manifest に `raw=true` が残るデバッグ用で、型付きの TASK はすべて検査する
- leader が拒否で止まり、人が気づかない: leader は BLOCKED を escalate し、escalate は見張りがいなくてもすぐ人へ送る(5 段目からの動き)
- 印の状態を足すと、前半の版の読み手と食い違う: 前半の版の `ralph status` と `--detach` は `stopping` の印を `stale` と読む。台帳を共有する 2 つの版を同時に使うことは想定しない(release のノートに、更新の前に見張りを止めると書く。2 段目からの注意と同じ)
- 子の作業ディレクトリを main worktree のルートに変えると、子が読む `ralph.toml` が変わる: `--config` を渡さない start と `--detach` は main worktree の `ralph.toml` を読む。全体の上限を main の `ralph.toml` から読む 3 段目の決まりと揃う。README と skill に書く

## Rollout or rollback notes

- release のノート: 見張りのない台帳で `spawn` と TASK の `send` が断られる。更新のあとは、`ralph org watch --all --detach` を起動するか、`ralph org start` で始める(start が見張りを起動する)。更新の前に見張りを `--stop` で止め、更新のあとに起動し直す(印に足した状態と欄を、前半の版は読めない)。`ralph status --json` の `org_watch.state` に `stopping` が増える
- revert すると検査がなくなり、前半の動き(見張りは自分で起動する)に戻る。印の `deadman_minutes` の欄は前半の版では読まれない

## Open questions

- なし(critical forks は Design decisions に記録する)

## Progress checklist

- [x] Plan reviewed
  - consult(plan)は F1〜F8 に同意し、F7 を PR の本文で見せること、F4 の理由を仕様の注に書くこと、見張りが途中で落ちたときの穴を tech-debt に書くことを足した。Codex plan advisory の 2 件(F9、F10)で計画を直した
- [x] Plan approved
  - 自動承認(ユーザーの指示「作成したプランは自動承認します」、2026-10-11)。図解ページは `.harness/state/plan-visual/org-watch-gate.html`
- [x] Branch created
- [x] Implementation started
  - S1(20cf6fdf): org 層の検査(`Org.WatchGate`、`WatchAcceptsWork`、`NewWatchGate`、`*WatchGateError` と `ErrNoRunningWatch`、`DefaultWatchGateSettle` 2 秒)、印の `deadman_minutes`・`once`・状態 `stopping`、`ProbeWatch` の `stopping` と印の deadman の窓。スライスの境目のずれ: `ralph status` の `stopping` と `(--once)` の表示を S2 から S1 に移した(状態を足すコミットで、止まりかけの見張りが「not running」と出ないため)。`StartFeature` の検査は `checkSpawnInput` の直後(`Manifest.Read` と `spawnPrecheckErr` の前)に置いた。`worktrees.Ensure` より前で、`Spawn` と同じ順。max_orgs と見張りなしが重なったら見張りのエラーで断る。検査は `Spawn` の打ち直し(冪等の戻り)より前なので、立っている座席の打ち直しも見張りなしでは断られる(F2 の位置の帰結)。期待を変えた 6a のテスト(AC10): `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysRunning` を `...TheMarkSaysStopping` に改名して期待を stopping に、`TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds` の 1 回の probe を stopping を待つ形に(どちらも F9 で後始末の間の印が stopping になったため)。エラーの次の手に `--state-dir` を付けるかは S2 で決める。mutation 13 体はすべて落ちた
  - S2(831bc342): `wireOrgWatchGate` を spawn・`start <task>`・`start --plan`(`FeatureRepoRoot` の前)・send の 4 か所に入れた。差し替え口は `orgWatchGateOverride`(TestMain は「いつも通す」)と `orgWatchGateSettleOverride`(TestMain は 50ms)。spawn と `start <task>` は自分で台帳を 1 回解決してから `newOrgSpawnRuntimeAt` を呼ぶ形にし、呼び手のなくなった `newOrgSpawnRuntime` を消した。次の手の `--state-dir` は、明示したときだけ付ける(`WatchGateError.HintStateDir`、`NewWatchGate` の 2 番目の引数、`orgInboxHintStateDir` と同じ規則)。拒否の出し方は今の拒否と同じ(spawn と start は stdout に `rejected: <err>`、send は err だけ)。CLI の既存のテストは書き換えていない。見張りなしで使える動詞(status、read、report、escalate、inbox、`wait --inbox`、stop、disband、`--all`)のテストを足した。mutation 8 体はすべて落ちた。この時点では start は見張りを起動しない(S3)ので、見張りのない台帳の start は not_running の文で断られる
  - S3(36b34bee): `ensureOrgWatch`(`--detach` と start が共有する、図 2 の流れ)、子の起動の要求 `orgWatchLaunchRequest`(Args・LogPath・Dir・Env)、`OnLocked` で banner をロックのあとに。start の検査は「ensure を走らせてから `NewWatchGate`」で、org 層の検査の位置(`checkSpawnInput` の直後)で起動するので、入力の検査で断られる start は見張りを起動しない。ずれと決めたこと: (1) 検査は envelope・max_orgs・予約の検査と打ち直しの戻りより前にあるので、それらの理由で断られる start でも、見張りは先に起動して動き続ける(見張りは要るものなので害はない。PR の本文に書く)。(2) 子の環境は HERDR_* の 4 つを外すのに加え、作業ディレクトリを変えたときは `PWD` もそれに合わせる(Env を明示すると os/exec が `PWD` を直さないため)。(3) 起動した子が終わっても台帳が stale の間(ほかの見張りがロックを取って最初の印の前)は期限まで待つ。(4) `--detach` は `--once` と `stopping` の見張りに終了コード 1 で断る(前は `--once` に `already running`)。期待を変えた 6a のテスト: `TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild` の終了コードを 1 から 0 に(AC5、tech-debt 206)。stale のテスト 3 本は待ちを 300ms に縮めた(期待は同じ)。`TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock` に banner がないことの確認を足した。mutation 11 体はすべて落ちた
  - S4(ed66d634): leader の雛形の `## 運用規律` に、見張りの不在で断られたら打ち直さず、TYPE を変えたり `--raw` にしたりして送り直さず、見張りを自分で起動せず、BLOCKED を escalate して返事を待つ項目を足し、`TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry` で固定した(`WatchGateError` の全状態の文が雛形の見分け方の語を含むことも確かめる)。`WatchGateError` の stopping と `--once` の文の後半を、start が自分で起動することに合わせて直した。/org skill の 4 面、README、仕様の「6 段目の後半で決めたこと」の注(F7 の読み替えを明記)、agent-messaging(2 面)、quality-gates(2 面)、AGENTS.md の repo map、`watch --help` の `--detach` を直した。tech-debt の 201・205・206 と 204 の (e) を閉じ、見張りが途中で落ちたときの穴を新しい行にした。待つ条件は雛形の既存の語「返事が来るまで」にした(escalate への返事は DECISION の型で届かないため)。検証の途中で、この差分に触れない 2 本(`TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)が負荷の高い全体の実行で 1 回ずつ落ち、再実行と単独では通った
  - 記録の直し: S2 の記録を足したときに改行が抜け、下の「Review artifact created」の行が S3 の記録の末尾につながっていたので戻した
  - 実装は終わり。次はパイプライン(cycle 1/2)
- [x] Review artifact created
  - self-review(f7a6fe6c): Merge yes、LOW 6(L1〜L6)。dbeb45ca で L1〜L4 を直した(L1: 起動した子が終わったあと、ロックを止まりかけ・`--once`・stale の見張りが持っていれば、最初の判定と同じ拒否を返す。L2: not_running の文の `ralph org start` を「新しい org なら」に限った。L3: send の拒否の `org:` の重なりを除いた。L4: pid の表示と kill の案内を org 層の 1 つにし、`orgWatchGateOverride` を `func() error` に、`orgWatchDetachDeadline` を `orgWatchEnsureDeadline` に改名した)。L3 を調べたときに、CLI の send がすべてのエラーを `org: send:` で包み直していること(この PR より前から)が分かり、7d2e4bbb で包み直さないようにした。L5・L6(tech-debt と README の文)は sync-docs に回す
  - self-review の再実行(264951e6): Merge yes、新しい LOW 3(N1〜N3)。bda15444 で 3 つとも直した(N1: テストのコメントの空白、N2: `--detach` の拒否の文を、子を起動したあとにも合う「so no new watch runs」に、N3: dry-run の追記のエラーも org 層で `org: send:` で包み、CLI の文字列の比較をやめた)。bda15444 は文言とエラーの包み方だけなので self-review を通さず、verify に読んでもらう
- [x] Verification artifact created
  - verify(ae349cc4): pass。AC12 は一部(文書のずれ D1〜D7 を sync-docs で直す)、AC8 は一部(座席の `wait` と inbox の show/ack/resolve/notify は配線がないことを grep で見ただけ)。記録にないずれ 2 つ(stale を待つ間に持ち主が消えたら起動へ進む、bda15444 で 6a のテスト 1 本の文言が変わった)も妥当とした。気づき: `--detach` の拒否の文は `--state-dir` を明示しても案内のコマンドに付けない(LOW)
  - verify の cycle 2(755e082d): pass。8 本の同時 `--detach` のあとの `--stop` が 0.86 秒で通り、見張りが残らないことを実機で見た。sync-docs に回す文書のずれ: 必須 D1〜D5・D8・D9、任意 D6・D7・D10
- [x] Test artifact created
  - test(a0149a10): pass(1,426 本、-race も 0)。mutation 64 件中、本物の穴 6 件(本番の settle、stale の持ち主が消えたあとの起動、古い版の印の deadman、起動の失敗の文)を probe で閉じられると報告した。実機で後退を 1 件見つけた: 同時の `--detach` 8 本が返った直後の `--stop` が 30 秒で時間切れになり、ロックを待っていた負けた子が、止まった見張りのあとにロックを取って動き続ける(base では起きない。S3 の AC5 の直しで、`--detach` と start が自分の子の終了を待たずに返るようになったため)
  - 後退の直し(fdfb5b23): consult(stuck)に聞き、(a) 自分の子の終了を待つ案にした。ほかの見張りが作業を受け付けているときも、自分の子が終わるまで(org 層の `DefaultWatchLockWait` 2 秒 + 1 秒)待ってから `already running` を返し、待つ間に子がロックを取れば `started`、期限を過ぎて子が生きていれば止めて回収する。退けた案: 子のロック待ちを 0 にする(ProbeWatch の一瞬の共有ロックと競う)、止める頼みを消さない(6a の頼みの規則を崩す)。test の probe 5 本も取り込んだ。HEAD の版で落ち直したあとで通るテスト 4 本、実機で `--stop` が 0.85〜0.94 秒で通り見張りが残らないこと(3 回 × 2)を確かめた。残る穴(base にもある): `--detach` が返る前に別の端末から `--stop` を打つと、ロックを待つ子があとを取りうる。foreground の `watch --all` がロックを待つ間の `--stop` も同じ。tech-debt に書く
  - consult の指摘に従い、コードの直しなので self-review から全段を cycle 2 として回し直す(post-implementation-pipeline.md の「fix-and-revalidate なら全段」)
  - self-review の cycle 2(236dc72e): Merge yes、LOW 4(C2-1〜C2-4)。3fb0fd1b で C2-1〜C2-3 を直した(C2-1: 子を止めたあとにもう一度判定してから結果を決める。止めた子のあとで見張りがいなくなっていれば、新しい文で断る。C2-2: `runOrgWatchDetach` の doc。C2-3: `--detach` の拒否と起動の失敗の案内のコマンドに、`--state-dir` を明示したときだけ付ける。org 層に `CommandWithStateDir` を出した)。C2-4(残る穴の tech-debt の行と L5・L6)は sync-docs で入れる。3fb0fd1b は self-review を通さず、cycle 2 の verify に読んでもらう
  - test の cycle 2(926efd94): pass(top-level 1,441 本、-race -count=5 で DATA RACE 0)。mutation 32 件中 29 件が落ち、生き残りの 3 件のうち T2 と K9 は等価。K8(本番の `orgWatchChildExitWait` を 0 にする)は、どのテストもこの値を自分で縮めるため残った。実機では 8 本の同時 `--detach` のあとの `--stop` が 3 回とも 0.4〜1.2 秒で通り、見張りは残らなかった。cycle 1 の HEAD(ae349cc4)では今も後退が再現する
  - K8 の穴を 47635690 で閉じた(`TestOrgWatchChildExitWait_OutlastsTheChildsLockWait`、本番の値が `org.DefaultWatchLockWait` より長いことを確かめる)。テストを 1 本足しただけなので、run-verify(rc 0)と cross-review に読んでもらう。6a の 2ccdd11b と同じ扱いで、PR の本文に書く
  - sync-docs の cycle 2(2fada512): verify の D1〜D10 と self-review の L5・L6・C2-4 をすべて入れた(tech-debt の F-7・201・202・206 の行の直し、残る穴の新しい行(別の端末の `--stop` を、ロックを待つ側が追い越す。C2-1 の現れ方も書いた)、README の status と main worktree の条件、skill の 4 面の `watch` と `start` の行、`runOrgWatchStop` の doc)。run-verify は rc 0。test の報告の Test gaps には K8 が開いたまま残るが、47635690 で閉じた(PR の本文に書く)
  - cross-review の cycle 2(bbe363a8): 指摘 1 件、ACTION_REQUIRED。子の作業ディレクトリを main worktree のルートに移したので、相対の `RALPH_ORG_AGMSG_HOME` か、明示した `--config` の相対の `agmsg_home` が別の場所を指す。consult(plan)は分類と直し方(案 A、起点は打った場所)に同意した。ユーザーの指示どおり上限を 3 に上げ(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=3`。`cycle-count.json` は skill の決まりで 2 のまま)、直したあと self-review から全段を cycle 3 として回し直す
  - 直し(8d8ddbd2): 作業ディレクトリを変えるときだけ、相対の env の値、または env が未設定か空で `--config`(空でない)を明示したときのその `agmsg_home` の相対の値を、打った場所から絶対パスにして、子の env にただ 1 つの `RALPH_ORG_AGMSG_HOME` として渡す(`orgWatchChildAgmsgHome`)。絶対パス、`~/`、空、`--config` なし、作業ディレクトリを変えない子は今のまま。テストは `--detach` の 11 件と start の 2 件。mutation 6 体はすべて落ちた(IsAbs の条件を外す形は等価)
  - self-review の cycle 3(1005f3bc): Merge yes、LOW 2 件。C2-1〜C2-4・L5・L6 は直った。C3-1(3 つのコメントが「cross-review cycle 2 #1」と書くが、triage の報告は cycle ごとに書き直される)は、cycle 3 の triage で cycle 2 の内容を付録に残して指し先を保つ。C3-2(`org.CommandWithStateDir` が `org.InboxCommand` の 4 行の写し)は、直すと全段のやり直しになるので sync-docs で tech-debt に書く
- [ ] PR created
