# org-herdr-autostart

- Status: Approved
- Approved: 2026-11-06 sha256:9bda99914bec
- Owner: Claude Code
- Date: 2026-11-06
- Related request: 2026-11-06 のユーザーの依頼「org runtimeを使用中に…herdrのサーバーが動いておらず（Connection refused）…もし今後同様の事案が発生した場合に、なんらかの方法でherdrを自動的に起動して欲しいです。」。/spec で決めた仕様は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の A(FR-A1〜A6、AC-A1〜A6)。同じ仕様の B(1 タブのペイン配置)は、この計画の PR のあとに別の計画にする
- Related issue: N/A
- Type: feat
- Branch: docs/spec-org-herdr-autostart-panes

## Objective

herdr のサーバーが止まっていても、`ralph org start`・`ralph org spawn`・`ralph org director start`(herdr の形)が herdr にワークスペースやペインを作らせる前に `herdr server` を切り離して起動し、そのことを stderr の 1 行で知らせる。director と `/org` の手順書は、人に herdr の起動を頼まず、承諾画面のために `herdr` で接続することだけを頼むようにする。

## Scope

- **driver(FR-A1)**: `internal/org/driver/herdr.go` に、`herdr status server --json` を打って `running`・`compatible`・`socket` を返すメソッドを足す。
- **org のゲート(FR-A1)**: `internal/org` の `Org` に、見張りのゲート(`WatchGate`)と同じ形の herdr のゲート(関数のフィールド)を足す。Spawn は実際の spawn のときだけ、StartFeature は worktree を作る前に、見張りのゲートのすぐあとに呼ぶ。StartDirector は herdr の形のときだけ、残った登録の確認(`checkCurrentDirector`。headless の director の生死を herdr に聞く)より前に呼ぶ(D1)。ゲートが失敗を返したら、台帳にも herdr にも何も書かずに返る。
- **CLI の自動起動(FR-A1〜A5)**: `internal/cli` に、状態を読み、止まっていれば `herdr server` を見張りと同じ切り離し起動(`startDetachedProcess`)で起動し、`running` になるまで最大 10 秒待つ処理を置く。子の環境から仕様の FR-A3 の変数を落とし、作業ディレクトリを `HOME`、出力を台帳の `herdr-launch.log` にする。起動したら stderr に 1 行を書く。start(両形)・spawn・director start で Org のゲートにつなぐ。3 つの動詞の `--help` に自動起動を書く。
- **手順書(FR-A6)**: `/director` と `/org` の skill(`.agents/` の写しと `templates/base/` の写しを含む)に、herdr の起動を人に頼まないことと、承諾画面のために `herdr` で接続するよう頼むことを書く。分離した herdr の手順(`docs/recipes/isolated-herdr-server.md`)に、サーバーが無ければ ralph が自分の環境で起動することを書く。
- **実機の確かめ(AC-A1・AC-A2)**: 分離した herdr で、サーバーが止まった状態から ralph がサーバーを起動することと、起動したサーバーの環境を確かめ、`docs/evidence/` に残す。

## Non-goals

- 仕様の B(1 タブのペイン配置、ペインのラベル、停止前の確認の移し替え)。次の計画で行う
- サーバーの再起動で戻った古い org のワークスペースの検出と片付け
- 承諾画面への自動の応答、端末のウィンドウを開くこと
- 版の違うサーバー(`compatible` が false)への対処
- 見張り・`send`・`read`・`wait` など、何も作らない動詞での自動起動
- 設定のキーを足すこと(自動起動は常に行う)

## Assumptions

- herdr 0.7.5 の `herdr status server --json` は、動いているときに `{"status":"running","running":true,"compatible":true,"socket":"…",…}`、止まっているとき(ソケットのファイルが残っていても無くても)に `{"status":"not_running","running":false,"compatible":null,"socket":"…",…}` を終了コード 0 で返す(2026-11-06 に確かめた。仕様の Research findings)。
- `herdr server` は、ソケットのファイルが残っていても約 0.5 秒で `running` になる。2 つ目の `herdr server` は `error: herdr server is already running` と終了コード 1 で終わる(同上)。
- setsid で起動した子は、起動した ralph が終わっても動き続ける(見張りの切り離し起動が同じやり方で動いている)。
- director の Claude Code の Bash ツールからも、見張りと同じく切り離し起動ができる。できないとき(サンドボックスなど)は FR-A5 の失敗になり、エラーが手で起動する方法を言う。

## Affected areas

- `internal/org/driver/herdr.go`、`herdr_test.go`(サーバーの状態を読むメソッド)
- `internal/org/spawn.go`(`Org` のゲートのフィールドと Spawn の呼び出し)、`feature.go`(StartFeature の呼び出し)、`director_start.go`(StartDirector の呼び出し)、ゲートを呼ぶ関数を置くファイル(`watchgate.go` の隣に `herdrgate.go`)と、それぞれのテスト
- `internal/cli/org_herdr_server.go`(新規。状態の読み取り、起動、待ち、環境の整理、stderr の 1 行、ゲートへのつなぎ)とそのテスト、`internal/cli/org.go`(start・spawn のつなぎと `--help`)、`internal/cli/org_director_headless.go`(director start のつなぎと `--help`)、`internal/cli/main_test.go`(本物の herdr を起動しないための差し替え)
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md`、`.agents/skills/` の写し(`scripts/sync-skills.sh`)、`templates/base/` の写し
- `docs/recipes/isolated-herdr-server.md`、`docs/evidence/`(実機の確かめの記録)

## Visual review

- Page: `.harness/state/plan-visual/org-herdr-autostart.html`(やさしい図 3 枚: `overview`・`easy-env`・`easy-fail`。折りたたみの中に `detail-overview`・`detail-sequence`・`detail-gate`・`detail-env`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: 全体と 7 枚の図を `shot` で撮って読んだ。1 回目に、詳細 2 の自分への矢印のラベルが隣の線にかかっていたことと、詳細 4 の変数名の間の空白が詰まっていたことを直し、撮り直して崩れがないことを確かめた。計画助言の所見を入れたあと、詳細 3 に行を 1 つ足して lint と撮り直しをし直した

## Design decisions

Critical forks: None。仕様で人が決めた 4 点(常に自動起動、サーバーだけを起動、spawn の順の配置、director も同じ形)のうち、この計画に関わるのは前の 2 つ。そのほかは次の既定で進める。

- D1 ゲートの位置: 台帳の読み書きと herdr の呼び出しより前に置く。spawn と start は見張りのゲートのすぐあと。Spawn は `--dry-run` のときゲートを呼ばない(仕様の FR-A1)。StartFeature の中の Spawn もゲートをもう一度呼ぶが、サーバーが動いていれば状態の読み取り 1 回で終わる。director start(herdr の形)は、入力の検査のあと、残った登録の確認より前に呼ぶ(codex の計画助言の所見 1)。残った登録が herdr の headless の director だと、その確認は herdr に agent の有無を聞き、herdr が止まっていると `unknown` で start を断るからである。先に herdr を起動すれば、戻ったペインに agent はいないので確認が「終わった」と判定し、start が片付けて進む。この順のため、促した director が登録済みで断られる start も herdr を起動するが、害はない。`launch_cmd` の形はゲートを呼ばないので、herdr の headless の director の登録が残り herdr が止まっていると、今までどおり断られる(R7)。
- D2 ゲートが失敗したときの Spawn の結果は `failed`(入力を断る `rejected` ではない)で、台帳には何も書かない。
- D3 状態の読み取りの失敗は、何も作らずに失敗にする(codex の計画助言の所見 2。通すと、StartFeature が worktree を作ってから Spawn が失敗し、事案と同じく worktree とブランチが残る)。失敗にするのは、`herdr` が PATH に無いとき(herdr が無いと言う)、読み取りが 5 秒で終わらないとき、runner のエラー、壊れた JSON、`running` の無い JSON である。例外は、出力(標準出力か、runner のエラーに含まれる標準エラー)に `unknown option: --json` があるときだけで、`status server --json` を知らない古い herdr とみなし、stderr に警告を 1 行書いてゲートを通す(自動起動はできず、今までどおりに動く)。herdr 0.7.5 は知らない引数に `unknown option: <引数>` を返す(2026-11-06、`herdr pane read` の `--source` の値と `pane process-info` の引数で確かめた)。
- D4 待ちの間に起動した子が終わったときは、状態をもう一度読み、`running` なら進む(別の ralph が先に起動した場合。仕様の FR-A2)。`running` でなければ、ログのパスを付けて失敗にする。
- D5 ログのファイル名は `herdr-launch.log`(herdr 自身のログ `herdr-server.log` と区別する)。

## Acceptance criteria

- [ ] AC1(仕様の AC-A1): herdr のサーバーが止まっているとき(ソケットのファイルが残っている場合と無い場合)、start(両形)・spawn・director start(herdr の形)のゲートが `herdr server` を起動し、`running` になってから herdr の呼び出しに進み、stderr に起動の 1 行(pid とソケットのパスと `herdr` で接続する案内)が出る。CLI のテストと実機の記録で確かめる。
- [ ] AC2(仕様の AC-A2): 起動する子の環境に `HERDR_PANE_ID`・`HERDR_TAB_ID`・`HERDR_WORKSPACE_ID`・`HERDR_ENV`・`CLAUDECODE`・`CLAUDE_CODE_*` が無く、`XDG_CONFIG_HOME`・`XDG_STATE_HOME`・`HERDR_SOCKET_PATH`・`CODEX_*` は残る。テストと、実機で起動したサーバーの環境(`ps -E`)で確かめる。
- [ ] AC3(仕様の AC-A3): サーバーが動いているときは `herdr server` を起動しない。spawn の `--dry-run` と director start の `launch_cmd` の形はゲートを呼ばない。
- [ ] AC4(仕様の AC-A4・FR-A5): 10 秒たっても `running` にならないとき、`herdr` が PATH に無いとき、子がすぐ終わって `running` でもないとき、状態を読めないとき(5 秒の打ち切り、runner のエラー、壊れた JSON、`running` の無い JSON)は失敗し、エラーに `herdr-launch.log` のパス(起動したときだけ。PATH に無いときは herdr が無いこと、読めないときはその理由)と、別の端末で `herdr` を打つ案内がある。`start --plan` は worktree もブランチも作らず、director start は登録を書かない。例外は `unknown option: --json` を返す古い herdr だけで、警告を 1 行出して通す(D3)。
- [ ] AC5(仕様の AC-A5): 起動した `herdr server` が `already running` で終わっても、状態をもう一度読んで `running` なら進む。
- [ ] AC6(仕様の AC-A6): `/director` と `/org` の skill(と `.agents/`・`templates/base/` の写し)に、herdr の起動を人に頼まないことと、承諾画面のために `herdr` で接続するよう頼むことがあり、`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る。
- [ ] AC7: 3 つの動詞の `--help` が、herdr のサーバーが止まっていれば起動することを言う。
- [ ] AC8: CLI のテストが本物の `herdr server` を起動しない(TestMain の差し替えで、起動しようとすると panic する)。
- [ ] AC9: herdr の headless の director の登録が残っていて herdr が止まっているとき、director start(herdr の形)は残った登録を確かめる前にゲートを呼ぶ。org のテストで、ゲートが `CheckDirector` より先に呼ばれることと、ゲートの失敗で `CheckDirector` も登録の書き込みも起きないことを確かめる(D1)。

## Implementation outline

1. S1 driver のサーバーの状態: `internal/org/driver/herdr.go` に `herdr status server --json` を打ち、`running`・`compatible`(null なら不明)・`socket` を返すメソッドと型を足す。runner のエラーはそのまま返し、JSON が読めなければエラーにする。argv と、running が true / false / 壊れた JSON / runner のエラーのテストを足す — verify: `test -z "$(gofmt -l internal/org/driver)" && go vet ./internal/org/driver/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/driver/`
2. S2 org のゲート: `Org` に herdr のゲートのフィールドを足し、`internal/org/herdrgate.go` に nil なら何もしない呼び出しを置く。Spawn(`--dry-run` でないとき)と StartFeature(worktree の前)は見張りのゲートのすぐあとに、StartDirector(herdr の形のとき)は残った登録の確認より前に呼ぶ(D1・D2)。ゲートの失敗で、Spawn が台帳にも herdr にも触れないこと、`--dry-run` で呼ばれないこと、StartFeature が worktree を作らないこと、StartDirector が `CheckDirector` を呼ばず登録を書かず herdr を呼ばないこと、`launch_cmd` の形で呼ばれないこと、herdr の headless の登録が残っているときゲートが `CheckDirector` より先に呼ばれることをテストする(AC3・AC4・AC9 の org 側) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Gate|Spawn|StartFeature|Director'`
3. S3 CLI の自動起動: `internal/cli/org_herdr_server.go` に、状態の読み取り(5 秒で打ち切り、読めなければ失敗、`unknown option: --json` だけは警告して通す)、`herdr` の PATH の確かめ、環境の整理、`startDetachedProcess` での起動、100ms ごとに 10 秒までの待ち、子が先に終わったときの読み直し、stderr の 1 行、失敗の文を置き、start(両形)・spawn・director start で Org のゲートにつなぐ。テストのための差し替えの変数(ゲート全体、起動、待ちの長さ)を置き、`main_test.go` の TestMain で、ゲートは通し、起動は panic にする。3 つの動詞の `--help` を直す(D3・D4・D5、AC1・AC2・AC4・AC5・AC7・AC8 の CLI 側) — verify: `go build ./... && test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Herdr|OrgStart|OrgSpawn|Director|Help'`
4. S4 手順書: `/director` の skill の、機能の org を立てる手順と「してはいけないこと」の近くに、herdr が止まっていても start が起動すること、人には起動を頼まず、最初の座席の承諾画面のために `herdr` で接続するよう頼むことを書く。`/org` の skill の「bypass 初回承諾」と前提に同じことを書く。`scripts/sync-skills.sh` で `.agents/` を作り直し、`templates/base/` の写しをそろえる。`docs/recipes/isolated-herdr-server.md` に、サーバーが無ければ ralph がその環境で起動すること(wrapper から打てば分離した環境のサーバーになる)を書く(AC6) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill|Director'`
5. S5 実機の確かめ: このブランチの ralph を build し、分離した herdr(`docs/recipes/isolated-herdr-server.md`)の環境で、サーバーが止まった状態(ソケットのファイルが残った状態を含む)から `ralph org spawn` を打ち、サーバーが起動すること、stderr の 1 行、ワークスペースができること、起動したサーバーの環境に AC2 の変数が無いことを確かめ、`docs/evidence/org-herdr-autostart-2026-11-06.md` に残す。ユーザーの herdr のサーバーには触れず、終わったら分離した環境のサーバーを止めて専用のディレクトリを消す(AC1・AC2 の実機) — verify: `test -s docs/evidence/org-herdr-autostart-2026-11-06.md && test ! -e /tmp/ralph-iso`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt・go vet・golangci-lint・skill と template の同期)。
- Spec compliance criteria to confirm: 仕様の FR-A1〜A6 と AC-A1〜A6 を、上の AC1〜AC8 とテストの名前、実機の記録に照らす。FR-A3 の落とす変数と残す変数が、コードとテストと仕様で同じであること。
- Documentation drift to check: `/director` と `/org` の skill の 3 つの写し、`docs/recipes/isolated-herdr-server.md`、3 つの動詞の `--help`、`AGENTS.md` の repo map(新しいファイルを足すなら)。
- Evidence to capture: `docs/reports/verify-*.md`、実機の記録 `docs/evidence/org-herdr-autostart-2026-11-06.md`。

## Test plan

- Unit tests: driver の状態の読み取り(argv、JSON の 4 通り)。CLI の環境の整理(落とす変数と残す変数を名前で固定)。ゲートの処理(動いている → 起動しない、止まっている → 起動して待つ → stderr の 1 行、起動が失敗 → 読み直して running なら進む、子が先に終わって running でない → ログのパス付きで失敗、10 秒の待ち切れ → 失敗、herdr が PATH に無い → 失敗、読み取りの打ち切り・runner のエラー・壊れた JSON・`running` の無い JSON → 失敗、`unknown option: --json`(標準出力と runner のエラーの両方)→ 警告して通す)。
- Integration tests: org の Spawn・StartFeature・StartDirector でゲートを呼ぶ位置と、失敗のときに何も書かないこと(読めない状態を返す偽のゲートで、StartFeature が worktree を作らず、director start が登録を書かないこと)。director start で、herdr の headless の登録が残っているとき、ゲートが `CheckDirector` より先に呼ばれること。CLI の start・spawn・director start で、ゲートの差し替えが失敗を返したときに動詞がその失敗で終わり、台帳に何も無いこと。
- Regression tests: `./scripts/run-test.sh` で、既存の org と CLI のテストがゲートの差し替えの下で通ること。
- Edge cases: ソケットのファイルが残った状態と無い状態(実機)、`--dry-run`、`launch_cmd` の形、`compatible` が null や false の状態(ゲートは running だけを見る)。
- Evidence to capture: `docs/reports/test-*.md`、実機の記録。

## Risks and mitigations

- R1 落とし漏れ: Claude Code が `CLAUDE_CODE_` で始まらない名前の変数を足すと、座席に漏れる。仕様の FR-A3 の名前と接頭辞をテストで固定し、実機の記録で起動したサーバーの環境を見る。
- R2 テストが本物の herdr を起動する: TestMain で起動を panic にし、ゲートを差し替える(AC8)。
- R3 起動したサーバーの PATH: サーバーは ralph を打った端末の PATH を受け継ぐ。人が端末で `herdr` を打ったときと同じで、座席の `claude`・`codex` が見つからないときは、今と同じ spawn の失敗になる。手順書には、herdr が座席の CLI を見つけられる端末から打つよう書く。
- R4 Bash ツールのサンドボックス: director の環境が切り離し起動を許さないと、ゲートが失敗する。エラーが手で起動する方法を言うので、今と同じく人に頼む道が残る。
- R5 古い herdr: `status --json` を知らない herdr では、`unknown option: --json` のときだけゲートが警告を出して通し、今までどおりに動く(D3)。古い herdr がこれと違う文で断ると、ゲートは失敗にする。そのときのエラーは herdr の出力を含めるので、herdr を新しくすればよいとわかる。
- R7 `launch_cmd` の director start と残った herdr の登録: `launch_cmd` の形はゲートを呼ばないので、herdr の headless の director の登録が残り herdr が止まっていると、今までどおり `unknown` で断られる。仕様は herdr の形だけを対象にしたので、この計画では直さない。`ralph org director stop --force` か、herdr を手で起動すれば進める。
- R6 戻った古いワークスペース: 自動で起動したサーバーは、前に動いていた org のワークスペースを戻すが、中の座席は戻らない。検出と片付けは仕様の対象外なので、S4 で `/director` の skill に 1 文だけ書く(ralph が herdr を起動したと言ったら、それより前に動いていた org の座席は止まっているので、人に伝える)。stderr の 1 行には書かない。

## Rollout or rollback notes

- merge のあとの次の release で配る。`ralph.toml` も台帳の形も変わらず、新しく増えるのは台帳の `herdr-launch.log` だけ。
- 戻すときは PR を revert する。起動済みの herdr のサーバーはそのまま残り、人が `herdr server stop` で止められる。

## Open questions

なし。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(`[reviewer].codex_model` と `codex_effort` の値)。所見は 2 件で、2 件とも計画に入れた(2026-10-11 の指示で、計画助言の修正提案は聞かずに直す)。(1) [HIGH] director start は残った headless の登録を herdr に確かめてから見張りのゲートに進むので、herdr が止まっていると `unknown` で先に断られる → D1 を直し、herdr の形ではゲートを残った登録の確認より前に呼ぶことにし、AC9 と R7 を足した。(2) [HIGH] 状態を読めないときに通すと、StartFeature が worktree を作ってから Spawn が失敗し、事案と同じく worktree が残る → D3 を直し、読めなければ失敗にし、通すのは `unknown option: --json` の古い herdr だけにした。AC4・S2・S3・Test plan・R5 と図の詳細 3 を合わせて直した
- [x] Plan approved — 2026-10-11 のユーザー指示(作った計画は自動承認する)により自動承認した
- [x] Branch created — `docs/spec-org-herdr-autostart-panes`(/spec の worktree を引き継いだ)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
