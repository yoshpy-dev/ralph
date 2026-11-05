# Walkthrough: org-herdr-autostart(herdr のサーバーが止まっていれば、org の動詞が起動する)

- Plan: `docs/plans/archive/2026-11-06-org-herdr-autostart.md`
- Spec: `docs/specs/2026-11-06-org-herdr-autostart-panes.md`(この PR は A。B の 1 タブのペイン配置は次の PR)
- Base: main `c206fdce`。この walkthrough の前のコミット `244c5144` までで 39 ファイル、+3,565 / −74(report と evidence と仕様・計画を除くと 31 ファイル)

読む順は、driver(1)、Org のゲート(2)、CLI の自動起動(3)、cycle 2 の直し(4)、文書(5)です。括弧の中はコミットです。

## 1. driver: サーバーの状態を読む(S1 `b351bb47`)

- `internal/org/driver/herdr.go` の `Herdr.ServerStatus` は `herdr status server --json` を 1 回打ち、`HerdrServerStatus{Running, Compatible, Socket}` を返す。herdr 0.7.5 のこの出力は、ほかのサブコマンドと違い envelope ではなく素の JSON object なので、`parseHerdrEnvelope` を通さずに読む。
- 読めない出力(runner のエラー、壊れた JSON、`running` の無い object)はすべてエラーにする。推測で `running:false` を返す道はない。
- `unknown option: --json` だけは `ErrHerdrStatusJSONUnsupported` で区別する(`--json` を知らない古い herdr)。正規表現は `--jsonl` などに当たらない。

## 2. Org のゲート(S2 `b8580f0a`、cycle 2 `f0edc152`)

- `Org.HerdrServerGate func() error`(`internal/org/spawn.go`)と、nil なら何もしない `checkHerdrServerGate`(`internal/org/herdrgate.go`)。見張りのゲート(`WatchGate`)と同じ形にした。
- 呼ぶ位置(計画 D1):
  - `Spawn`: 入力の検査と見張りのゲートのすぐあと、`--dry-run` でないときだけ。座席のロック、台帳の読み取り、冪等の early return より前なので、有効な座席への打ち直しもゲートを通る(cycle 2 で doc とテストに固定した)。
  - `StartFeature`: 見張りのゲートのすぐあと、台帳の読み取りと worktree の ensure より前。事案(worktree とブランチだけが残った)はこれで起きなくなる。
  - `checkDirectorStart`: herdr の形のときだけ、入力の検査のあと、残った登録の確認(`checkCurrentDirector`)より前。確認は headless の director の生死を herdr に聞くので、herdr が止まっていると `unknown` で先に断られるためである(codex の計画助言の所見 1)。
- ゲートの失敗は `SpawnOutcomeFailed`(入力の断りの `rejected` ではない)で、台帳・receipt・herdr・agmsg のどれにも触れない(D2)。

## 3. CLI の自動起動(S3 `f3cca9e0`、cycle 2 `f0edc152`)

- `internal/cli/org_herdr_server.go` の `ensureOrgHerdrServer` が本体。状態を 5 秒の上限で読み、止まっていれば `exec.LookPath("herdr")` で探し、`startDetachedProcess`(見張りの切り離し起動と同じ関数。`org_watch.go`)で `herdr server` を起動し、100ms ごとに最大 10 秒、`running` になるまで読む。
- 起動したら stderr に 1 行(`ralph org <verb>: started the herdr server (pid …, socket …); run 'herdr' in another terminal …; log: <state dir>/herdr-launch.log`)。起動した子がもう終わっていれば、別の ralph のサーバーなので 1 行を出さない。
- 環境(`orgHerdrServerEnv`): `HERDR_PANE_ID`・`HERDR_TAB_ID`・`HERDR_WORKSPACE_ID`・`HERDR_ENV`、`CLAUDECODE`、`CLAUDE_CODE_*` を落とし、ほかは残す。サーバーの環境はすべてのペインに入るので、director の Claude Code のセッションの変数を座席に渡さない。作業ディレクトリは `HOME`。
- 失敗の文は `org: <verb>: …, so nothing was made; start herdr in another terminal with 'herdr', then run ralph org <verb> again` の形。herdr が PATH に無いとき、状態を読めないとき、10 秒で答えないとき、起動した子が答える前に終わったときに出る。
- つなぎ(`wireOrgHerdrServerGate`)は spawn・start `<task>`・start `--plan`・director start の 4 か所。`orgHerdrServerGateOverride` と、起動すると panic する `orgHerdrServerLaunch` を TestMain が置き、CLI のテストは本物の herdr を起動しない。

## 4. cycle 2 の直し(`f0edc152`、sync-docs `1b56097c`)

- cycle 1 の self-review M1: ゲートが冪等の early return より前にあることを、Spawn の doc の step 1 と `herdrgate.go` に書き、`TestOrgSpawn_HerdrServerGate_ARespawnOfAnActiveSeatAsksItToo` で固定した。
- cycle 1 の test C21: `TestOrgStartPlan_HerdrServerGate_StartsAStoppedServerOnceInStartsWords` が、start --plan で起動は 1 回、状態の読み取りは 3 回(StartFeature で 2 回、Spawn で 1 回)、1 行は start の言葉であることを固定する。
- 文言: help の「refused」を「fails」に、skill の「断られ」を「失敗し」に、`director start --help` の「runs no herdr」を直した。

## 5. 文書(S4 `11c64fb1`、sync-docs `d6951690`・`1b56097c`)

- `/director` と `/org` の skill(と `.agents/`・`templates/base/` の写し): 人に herdr の起動を頼まず、承諾画面のために `herdr` で接続するよう頼む。ralph が herdr を起動したと言ったら、それより前に動いていた org の座席は止まっていることも伝える。
- `docs/recipes/isolated-herdr-server.md`(と写し)、`docs/recipes/codex-seat-permissions.md`(と写し)、`README.md`、`AGENTS.md` の地図、`docs/quality/quality-gates.md`(と写し)、`docs/specs/2026-08-01-org-runtime.md` の 2026-11-06 改訂。
- `docs/tech-debt/README.md`: 新しい行(ゲートが後の断りより前にある副作用、`launch_cmd` の director start と残った herdr の登録、同時起動、`CODEX_*`・`Compatible`・10 秒の置き場所)と、Trigger に当たった行への注(318 行の別のソケットの director start を含む)。
- 実機の記録: `docs/evidence/org-herdr-autostart-2026-11-06.md`(分離した herdr で spawn の 4 通り)。
