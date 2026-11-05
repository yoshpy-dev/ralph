# Verify report: org-herdr-autostart

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-autostart.md`(AC1〜AC9、D1〜D5、Progress の S1〜S5)。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の A(FR-A1〜A6、AC-A1〜A6)。B は次の計画なので見ていない
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff c206fdce...HEAD`(base main c206fdce、HEAD d77ee3c3、27 ファイル、+2609 / -44)。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない(`go test` は打っていない)。テストは中身を読んで、何を assert しているかだけを確かめた。HEAD から `go build` した binary を一時ディレクトリに置き、3 つの動詞の `--help` だけを打った。herdr は打っていない。worktree には、この report と insight event のほかに何も書いていない
- Evidence: `docs/evidence/verify-2026-11-06-org-herdr-autostart.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。中身は run-static-verify の全出力と、末尾の Probe 1〜6(計画の digest、S5 の検証コマンドと gofmt と 2 つの sync、実機の確かめの build 以後の変更、herdr に作らせる呼び出しと Spawn の呼び元の一覧、3 つの `--help`、文書の grep)。run-verify 自身のログは `docs/evidence/verify-2026-11-05-184604.log`(同じく対象外)
- 計画の digest: `./scripts/plan-visual.sh digest` は `9bda99914bec` で、`- Approved:` の行と同じ
- 実機の記録の binary: `docs/evidence/org-herdr-autostart-2026-11-06.md` は 11c64fb1 から build した。`git diff 11c64fb1 HEAD --name-only` は記録・計画・self-review の文書 4 つだけで、コードは HEAD と同じ

## Spec compliance

計画の AC は spec の FR と AC-A を束ねているので、行ごとに対応する FR と AC-A を書いた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 止まったサーバーを起動して進む(FR-A1・FR-A4、AC-A1) | Met | ゲートは `ensureOrgHerdrServer`(`internal/cli/org_herdr_server.go:127`)。状態が `running:false` なら `herdr` を PATH で探し、`herdr server` を `startDetachedProcess`(`org_watch.go:477`、Setsid、標準入力 `/dev/null`、ログへ追記)で起動し、`await`(`:173`)が 100ms ごとに最大 10 秒読む。答えたら `announce` が pid・ソケット・`herdr` での接続の案内・`herdr-launch.log` の 1 行を書く。CLI のつなぎは spawn(`org.go:943`)、start(`org.go:1161`)、start --plan(`org.go:1278`)、director start(`org_director_headless.go:352`)の 4 か所。テストは `TestOrgHerdrServerGate_AStoppedServerIsStartedAndAnnounced`(3 つの動詞の文面を固定)と `TestOrgHerdrServerGate_EachVerbStartsAStoppedServerAndGoesOn`(spawn・start・director start を本物のゲートと偽の seam で通し、1 行と launch 1 回と先の出力を確かめる)。実機の記録 A(ソケット無し)と B(ソケットが残った、事案と同じ Connection refused)で spawn が起動の 1 行を出して herdr の呼び出しに進んだ。start --plan の成功の道は、CLI のつなぎ(`org.go:1278`)と、org 層の `TestStartFeature_HerdrServerGate_AFailureMakesNoWorktree` の「passes」(ゲートが 2 回呼ばれて leader が立つ)の組み合わせで確かめた。2 回目の呼び出しは状態を 1 回読むだけで 1 行を出さないことはコードで読んだ。start --plan を本物のゲートで通す CLI のテストと実機はない(Coverage gaps) |
| AC2 起動する子の環境(FR-A3、AC-A2) | Met(実機は一部) | `orgHerdrServerDropsEnv`(`:315`)が `orgWatchChildDroppedEnv` の 4 つ、`CLAUDECODE`、`CLAUDE_CODE_` で始まる名前を落とし、`orgHerdrServerEnv`(`:292`)が PWD を HOME に替える。spec・計画・コード・テスト・help・skill・recipe の名前の並びは同じ。`TestOrgHerdrServerEnv_DropsThePaneAndClaudeCodeVariablesOnly` が落とす名前と残す名前(`XDG_*`、`HERDR_SOCKET_PATH`、`CODEX_HOME`、`CODEX_SANDBOX`、`CLAUDE_CONFIG_DIR`、境界の `CLAUDE_CODE`・`CLAUDECODE_X`)を順まで固定する。実機は、AC2 が書く `ps -E` ではなくペインの `env` で読んだ(macOS で読めなかったと記録にある)。I1 を参照 |
| AC3 動いているときは起動しない、dry-run と launch_cmd は呼ばない(FR-A1、AC-A3) | Met | `ensureOrgHerdrServer` の最初の case(`:131`)。`TestOrgHerdrServerGate_ARunningServerStartsNothing` が compatible の 3 通りで読み 1 回・起動なしを固定する。Spawn は `!p.DryRun` のときだけ呼ぶ(`spawn.go:640-644`)。org 層の `TestOrgSpawn_HerdrServerGate_ADryRunDoesNotAskIt` と CLI の `TestOrgSpawn_HerdrServerGate` の「a --dry-run does not call it」。director start は `!c.cmdLaunch` のときだけ呼ぶ(`director_start.go:1078-1082`)、`TestStartDirector_HerdrServerGate_ALaunchCmdStartDoesNotAskIt`。実機の記録 C で、動いているサーバーへの spawn は起動の行を出さず、新しいサーバーもなかった。start に `--dry-run` は無い |
| AC4 起動できないときは何も作らずに失敗(FR-A5、AC-A4) | Met | 失敗の文は 5 種類(PATH に無い、状態を読めない、起動の失敗、期限切れ、子が先に終わる。文の形は 7 通り)あり、どれも `then run ralph org <verb> again` で終わる。`herdr-launch.log` のパスは起動したあとの 2 通り(`await` の期限切れ、`afterExit`)だけに付く。テストは期限切れ(`NoAnswerWithinTheDeadline`、待ちを 200ms に縮める)、PATH に無い(`HerdrNotOnPATH`、lookup と状態の読み取りの 2 通り)、子が先に終わる(`AServerThatExitsFirst`)、読めない状態(`AStatusThatCannotBeReadFails`、runner のエラー・JSON でない・`running` 無し・打ち切り)。古い herdr の例外は `AHerdrWithoutStatusJSONWarnsAndPasses` で、driver の正規表現 `unknown option: --json([^\w-]|$)`(`herdr.go:685`)は `--jsonl` に当たらない(`TestHerdr_ServerStatus_OtherUnknownOptionIsNotUnsupported`)。start --plan は worktree の前に呼ぶ(`feature.go:239`)。CLI の `TestOrgStartPlan_HerdrServerGate_ARefusalMakesNoWorktree` が worktree・ブランチ・台帳・herdr と agmsg の呼び出しがないことを確かめる。director start は `TestOrgDirectorStart_HerdrServerGate_ARefusalWritesNothing` と org 層の `TestStartDirector_HerdrServerGate_AFailureWritesAndCallsNothing` で `director.jsonl` に何も書かない。spawn の失敗は `SpawnOutcomeFailed`(D2)。実機の記録 D で、PATH に無いときの文と台帳 10 行のままを見た |
| AC5 already running で終わっても読み直して進む(FR-A2、AC-A5) | Met(偽物だけ) | 起動の失敗は `:148-:157` で 1 回読み直す。子が先に終わったときは `afterExit`(`:204`)で 1 回読み直す。`TestOrgHerdrServerGate_ALaunchThatFails` と `TestOrgHerdrServerGate_AServerThatExitsFirst` がどちらも「running なら 1 行なしで通る」を固定する。2 つの ralph を同時に打つ実機はなく、勝った側が答える前に負けた側が終わる順は self-review の L1 が開いたまま |
| AC6 skill に書いてあり、2 つの sync が通る(FR-A6、AC-A6) | Met(FR-A6 の 1 ファイルを除く) | `/director` の「立てる」の 1 と 5、「機能ごとの出力先」の 3、「してはいけないこと」、`/org` の「前提」「bypass 初回承諾」「`ralph org start --plan`」の 2 に、人に起動を頼まないことと、`herdr` で接続して承諾画面に答えてもらうことがある。4 つの写しは `check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)で一致。計画の R3(PATH)と R6(戻った workspace の座席は止まっている)の文もある。spec の FR-A6 は `internal/org/prompts/director.md` も名指すが、この PR は変えていない(I2 の c) |
| AC7 3 つの動詞の `--help` | Met | `orgHerdrServerHelp`(`:39`)を 3 つの Long が前置きの 1 文つきで使う。HEAD の binary で 3 つの `--help` を打ち、段落が出ることを見た(Probe 5)。`TestOrgHerdrServerGate_TheHelpOfEachVerbSaysItStartsTheServer` が前置きと共通の 5 文を固定する |
| AC8 テストが本物を起動しない | Met | `main_test.go:192-195` でゲートを通す差し替えと、起動を panic にする差し替えを置く。`TestOrgHerdrServerLaunch_TestMainMakesARealLaunchPanic` がその両方を確かめる。ゲート自身のテストは `setOrgHerdrServerSeams` で起動を偽物にし、`t.Cleanup` で戻す |
| AC9 director start はゲートを残った登録の確認より前に呼ぶ(D1) | Met | `checkDirectorStart` の順は、入力の検査、launch_cmd の検査、ゲート(`director_start.go:1079`)、`Store.Read` と `checkCurrentDirector`(`:1091`)、見張りのゲート。`TestStartDirector_HerdrServerGate_RunsBeforeTheCheckOfAHeadlessHerdrDirector` は、ゲートが herdr を「起動」するまで agent get が失敗する偽物で、呼び出しの順が `herdr_server_gate, agent_get, pane_get, ...` になり d1 を片付けて d2 が立つことを固定する。失敗のときは `AFailureWritesAndCallsNothing` が herdr の呼び出し 0 と ps の読み 0 を確かめる。入力の検査で断られたときに呼ばないことは `RunsAfterTheInputChecks` |

FR と非機能要件のうち、AC に入らないものも見た。

- herdr に作らせる呼び出しはすべてゲートのうしろにある。本番のコードの `TabCreate` と `WorkspaceCreate` は `spawn.go:1045`・`:2460`(Spawn)と `director_start.go:1165`・`:1180`(StartDirector)だけで、Spawn の呼び元は `feature.go:281` と `org.go:949`・`:1168` だけである(Probe 4)
- 非機能要件の「動いているときは状態の確認 1 回だけ」は `ARunningServerStartsNothing` の `reads == 1`。`herdr server` の起動が driver の外(`internal/cli`)にあるのは、spec が `startDetachedProcess` と同じやり方を求めているためで、要件と合う
- Security considerations の「`herdr-launch.log` に環境の値を書かない」: コードはログに子の出力しか書かず、実機の記録 A でもログに環境の値はなかった

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang) | pass(exit 0) | shellcheck と `sh -n`、settings の jq、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照、gofmt ok、`go vet ./...`、golangci-lint 0 issues、staticcheck、branch の secret scan(c206fdce..d77ee3c3 clean) |
| `./scripts/plan-visual.sh digest <plan>` | `9bda99914bec` | `- Approved:` と同じ |
| S5 の検証コマンド `test -s docs/evidence/org-herdr-autostart-2026-11-06.md && test ! -e /tmp/ralph-iso` | pass | 分離に使ったディレクトリは残っていない |
| `gofmt -l internal/org internal/cli` | 空 | S2・S3 の検証コマンドの gofmt の部分 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/director` と `/org` の skill(4 つの写し) | Yes | AC6 の行。文面が言う stderr の 1 行と失敗の文の終わりは `announce` と `next` の書式と同じ |
| `docs/recipes/isolated-herdr-server.md`(と template の写し) | Yes | 自動起動の段落を足し、wrapper から打てば分離したサーバーになること、recipe は今までどおり自分で起動することを書く。`check-sync.sh` が両方を一致と見た |
| 3 つの動詞の `--help` | Yes(I3 の言い回しを除く) | AC7 の行 |
| `AGENTS.md` の repo map | No | I2 の a。`internal/org/` の項は `watchgate.go` を名指すが、並ぶ `herdrgate.go` を書いていない。`internal/cli/` の項はサブコマンドの名前だけでファイルを挙げないので、`org_herdr_server.go` を書くかは /sync-docs の判断 |
| `docs/specs/2026-08-01-org-runtime.md` | No | I2 の b |
| `internal/org/prompts/director.md` | 実質は Yes | I2 の c |
| `docs/recipes/codex-seat-permissions.md:18`(と template の写し) | 古い | I2 の d |
| `README.md:266` | 足りない | I2 の e |
| `/org` の動詞リファレンスの spawn・start・director の行 | 足りない | I2 の f |
| `docs/tech-debt/README.md` | 要判断 | V1(318 行)と I2 の g(276 行) |
| 計画と spec の checkbox | 遅れ | I2 の h |

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | D1 の順で、tech-debt 318 行の危険(2 つの herdr サーバーで director が 2 人になる)に新しい入り口ができた。同じ台帳で、headless の herdr の director d1 があるサーバーで動いているとき、`HERDR_SOCKET_PATH` か XDG の変数が別のソケットを指す端末で `ralph org director start` を打つと、これまではそのソケットにサーバーがないので `agent get` が接続のエラーになり、`unknown`(`agent_get_failed`)で断られていた。今はゲートがそのソケットに新しいサーバーを起動するので、`agent get` は `agent_not_found`、記録した pane は無く、確かめは `ended`(`pane_closed`)と読み、start は d1 の登録を片付けて、d1 が動いたまま d2 を立てる。318 行の Trigger の「the next change to how ralph reaches the herdr server」に当たると読める。計画の D1 と Risks は同じサーバーで herdr が止まった場合だけを書き、self-review もこの道を挙げていない | `internal/org/director_check.go:379-394`(接続のエラーは `unknown`、`agent_not_found` で pane が無ければ `ended`)、`director_start.go:1079`(ゲート)と `:1091`(確かめ)、`docs/tech-debt/README.md` 318 行。AC9 のテストは同じサーバーの再起動だけを模す | 直すなら、ゲートが起動したサーバーのソケットを `checkCurrentDirector` に渡し、起動した直後のサーバーで `pane_closed` を `ended` と読まない。直さないなら、318 行に carry-over の注(このゲートで、ソケットにサーバーがない場合も入り口になった)を足すか、計画の Risks に 1 行足す。どちらも cross-review の triage で決めてよい |
| I1 | INFO | 計画の AC2 は実機を `ps -E` で確かめるとしたが、記録はペインの `env` で読んだ(macOS の `ps -wwE` が他のプロセスの環境を読めなかったと書いてある)。表の `HERDR_TAB_ID` の欄の「ralph の値は落ちている」は、herdr がペインごとに `HERDR_TAB_ID` を付けるので、ペインの値からは言えない。`CLAUDECODE` と `CLAUDE_CODE_PROBE` が無く `CODEX_PROBE` が届いたことは、サーバーの環境がペインに届くことの対照になるので、Claude Code の変数を落としたことはほぼ確かである。落とす名前の全体はユニットテストが固定している | `docs/evidence/org-herdr-autostart-2026-11-06.md` の「起動したサーバーの環境(AC2)」 | 記録の `HERDR_TAB_ID` の注を「ペインでは herdr の値に替わるので、この方法では確かめられない」に直す。PR の本文の Verification に、AC2 の実機はペインの環境で読んだと書く |
| I2 | INFO | /sync-docs に回す文書のずれ(下の a〜h) | 各項 | 各項 |
| I3 | INFO | help の段落は失敗を「the command is refused with nothing made」と書くが、spawn は D2 のとおり `spawn failed:`(`SpawnOutcomeFailed`)を出し、`rejected:` ではない。spawn の help の前の段落は「Either refusal ... is recorded as `rejected`」と書くので、「refused」から `rejected:` を思い浮かべる読み手がいる。leader の雛形は `spawn failed:` の「それ以外のエラー」として BLOCKED に回すので、動きは困らない | `org_herdr_server.go:48`、`org.go` の spawn の Long、`internal/org/prompts/leader.md:1298-1325` | 直すなら「fails with nothing made」にし、help のテストの 4 つ目の文を合わせる。直さなくてよい |

I2 の内訳は次のとおり。

- a. `AGENTS.md:91` の `internal/org/` の項は見張りのゲート(`watchgate.go`)まで書くが、`herdrgate.go`(herdr のサーバーのゲート、Spawn・StartFeature・StartDirector が呼ぶ)を書いていない。`internal/cli/` の項はサブコマンドの名前だけなので、`org_herdr_server.go` は書かなくても項の粒度と合う
- b. spec の非機能要件は「振る舞いが変わる文書(org runtime の仕様、…)を、それぞれの PR で直す」と言い、Impact の表は `docs/specs/2026-08-01-org-runtime.md` に「自動起動の記述」を挙げる。この PR はその spec を変えておらず、計画の Affected areas にも入っていない。狭めた理由は計画に記録がない
- c. spec の FR-A6 は `internal/org/prompts/director.md` も名指す。この PR は変えていないが、雛形は「手順の全体は `/director` skill に従って」と言い、skill の変更が headless の director に届く。計画の Scope はこのファイルを挙げず、狭めたことの記録もない
- d. `docs/recipes/codex-seat-permissions.md:18` と template の写しの前提「The herdr socket is up: open the herdr TUI, or run `herdr server` headless.」は、start・spawn・director start が自分で起動するようになったので、もう前提ではない
- e. `README.md:266` は start が見張りを自分で起動することを書くが、herdr のサーバーを起動することを書いていない
- f. `/org` の動詞リファレンスの `spawn`・`start`・`director` の行(`.claude/skills/org/SKILL.md` の 335・344・356 行)は見張りのゲートを細かく書くが、herdr のゲートは「前提」にしかない。`start` の行の「見張りの起動と(`--plan` の)worktree の作成は、この期限に入らない」の並びに herdr の起動も入る
- g. tech-debt 276 行の Trigger「the next change to the herdr calls of `StartDirector`」: herdr の形の StartDirector は、この PR でゲートを通して `herdr status server --json` と `herdr server` を打つようになった。前の PR(org-role-models)は同じ Trigger に carry-over の注を残しているので、この PR も当たったかどうかを書くのがよい。返すのに要る実機(codex の director が start --plan を打つ)はしていない
- h. 計画の AC1〜AC9 と spec の FR-A1〜A6・AC-A1〜A6 は `[ ]` のまま。この run は report だけをコミットするので付けていない

## Observational checks

- HEAD の binary で `ralph org spawn --help`、`ralph org start --help`、`ralph org director start --help` を打ち、前置きの 1 文と共通の段落を見た(Probe 5)
- 実機の記録 `docs/evidence/org-herdr-autostart-2026-11-06.md` を AC1・AC2・AC3・AC4 に照らした。A と B は spawn で、ソケット無しと残ったソケットの両方を通る。C は動いているサーバーで起動しない。D は PATH に無いとき。どれも分離した herdr で、ユーザーのサーバー(pid 22408)の前後の `pgrep` を書いてある。記録に token と環境の値はない

## Coverage gaps

- Go のテストの結果(/test の担当)。AC の行のテストは中身を読んだだけで、通るかは見ていない
- start --plan を本物のゲート(偽の seam)で通す CLI のテストがない。`TestOrgHerdrServerGate_EachVerbStartsAStoppedServerAndGoesOn` は spawn・start・director start だけで、start --plan ではゲートが 2 回呼ばれても 1 行と起動 1 回で済むことを確かめるテストがない。spec の AC-A1 の Given/When はまさに start --plan である
- start --plan と director start の実機、2 つの ralph が同時に起動する実機(AC5、self-review L1)、`status server --json` の無い古い herdr、座席の agent が起動するところ(記録の「確かめていないこと」)
- director の Claude Code の Bash ツールや codex の sandbox の中からの切り離し起動(計画の R4)
- 起動したサーバーのプロセスそのものの環境(I1)
- V1 の 2 つのソケットの道は読んで組み立てただけで、テストも実機もない

確かさを一番上げる追加の確かめは、`TestOrgHerdrServerGate_EachVerbStartsAStoppedServerAndGoesOn` に start --plan の行(`directorStartLedger` の代わりに `newFeatureRepo` と `writeFeatureSplitPlan` の台帳)を 1 つ足し、stderr の `started the herdr server` が 1 回、`f.launches` が 1、`f.reads` が 3 であることを見るテストである。spec の AC-A1 の文面どおりの道を、決定論的に通せる。

## Verdict

pass

- Verified: AC1〜AC9 をコードとテストの assert と help の出力で、AC1・AC3・AC4 の spawn の道を実機の記録で確かめた。herdr に作らせる呼び出しがすべてゲートのうしろにあること。FR-A3 の名前が spec・計画・コード・テスト・help・skill・recipe で同じであること。静的解析の全項目と 2 つの sync。計画の digest は `9bda99914bec` のまま
- Partially verified: AC2 の実機(ペインの環境での代わりの確かめ、I1)。AC5 は偽物だけ。start --plan の成功の道は CLI のつなぎと org 層のテストの組み合わせだけ
- Not verified: Go のテストの結果と `run-test.sh`(/test)。start --plan と director start の実機。同時の起動。古い herdr。sandbox の中からの起動
- 所見: LOW 1 件(V1)、INFO 3 件(I1〜I3)。どれも merge を止めない
- /sync-docs に回すもの: I2 の a〜g。V1 は cross-review の triage で直すか tech-debt 318 行の注にするかを決める。I1 は記録の 1 行の直しで、/sync-docs か PR の本文で扱える
- Insight event: `docs/insights/events/2026-11-05-org-herdr-autostart.jsonl` に 1 行足した(phase verify、cycle 1、verdict pass)。ファイル名は `insights-append.sh` が UTC の日付で決め、self-review の行と同じファイルになった

## Cycle 2(HEAD 227bfd28)

- Date: 2026-11-06
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2。上限 2 の最後の run)
- この節より上は cycle 1(Run 1、HEAD d77ee3c3)の本文で、1 バイトも変えていない。`docs/tech-debt/README.md` の 318 行が V1 を、433 行が AC5 の行と Coverage gaps を引くためである。cycle 2 の結果はこの節だけに書いた
- Scope: `git diff c206fdce...HEAD`(HEAD 227bfd28、39 ファイル、+3245 / -71)。Run 1 のあとに増えたのは、test・sync-docs・cross-review・self-review の記録と、修正の f0edc152(7 ファイル、+120 / -7)。f0edc152 の本番のコードの変更は help の 1 行だけで(Probe C2-3)、ほかはコメント、テスト 2 本、実機の記録、tech-debt である。sync-docs の d6951690 は Run 1 のあとの commit なので、この run で初めて文書のずれを見た。AC1〜AC9 は HEAD のコード、テスト、文書に照らし直した。Go のテストと `./scripts/run-test.sh` は打っていない。HEAD から `go build` した binary を `/tmp` に置き、3 つの動詞の `--help` だけを打った。herdr は打っていない
- Evidence: `docs/evidence/verify-2026-11-06-org-herdr-autostart.log` の末尾に「Run 2」の区切りを付けて、run-static-verify の全出力と Probe C2-1〜C2-8 を足した(このログは `.gitignore` の `docs/evidence/*.log` に当たり、コミットしない)。run-verify 自身のログは `docs/evidence/verify-2026-11-05-200616.log`
- 計画の digest: `9bda99914bec` で、`- Approved:` の行と同じ(Probe C2-2)

### 前の回の所見の扱い(cycle 2)

| Run 1 の所見 | 状態 | 根拠 |
| --- | --- | --- |
| V1(別のソケットを指す端末の director start で director が 2 人になりうる) | defer(`docs/tech-debt/README.md` 318 行の Trigger の 2026-11-06 の注、f0edc152) | 注は、ゲートがその端末の変数が指す場所に空のサーバーを起動し、今の director を `agent_not_found` から `pane_closed` で ended と読んで片付け、2 人目を立てうることと、これまでは `agent_get_failed` の `unknown` で断られていたことを書く。コードを読んでの推論で、テストも実機もないことと、直すには登録に herdr のサーバーを記録する必要があることも書く。Related の列はこの report の V1 を引く。Run 1 の V1 の文と食い違わない |
| I1(実機の記録の `HERDR_TAB_ID`) | 解消(f0edc152) | `docs/evidence/org-herdr-autostart-2026-11-06.md` の表は「herdr がどのペインにも付ける値。ralph の値を落としたかは、この方法では確かめられない」になった。表の下の段落が、落とした根拠を `CODEX_PROBE` が届き `CLAUDECODE` と `CLAUDE_CODE_PROBE` が無いことに置き、`HERDR_*` は `TestOrgHerdrServerEnv_DropsThePaneAndClaudeCodeVariablesOnly`(`internal/cli/org_herdr_server_test.go:489`)が固定すると書く |
| I2 の a(`AGENTS.md` の repo map) | 解消(d6951690) | `internal/org/` の項に、`Spawn`(dry run を除く)・`StartFeature`・`StartDirector`(herdr の形)が呼ぶ herdr のサーバーのゲート(`herdrgate.go`)の句がある。コードの呼び出し位置と合う |
| I2 の b(`docs/specs/2026-08-01-org-runtime.md`) | 解消(d6951690) | 「2026-11-06 改訂」の節 (a)〜(c) を足した。対象の動詞、`launch_cmd` の形が対象外であること、B は対象外であることを書く。(b) の「断る」の語は下の I2-1 |
| I2 の c(`internal/org/prompts/director.md`) | 解消(判断の記録) | 雛形は変えていない。d6951690 の sync-docs report の「I2 の a〜h」の c の行に、雛形は herdr のサーバーの起動について何も言わず、手順を `/director` skill に任せるので、skill の変更が headless の director に届く、という判断が残っている。grep でも、雛形の herdr の語は pane で話す文と終え方の文だけだった。spec の FR-A6 の box を付けるかは orchestrator が決める |
| I2 の d(`docs/recipes/codex-seat-permissions.md:18`) | 解消(d6951690) | 前提を「サーバーは動いていなくてよい。`ralph org spawn` が起動する」に替えた。recipe の probe は `--dry-run` の無い `ralph org spawn`(210 行、303 行)なので、文と合う。template の写しは `cmp` で同一(Probe C2-7) |
| I2 の e(`README.md`) | 解消(d6951690) | 見張りの自動起動の文のあとに、3 つの動詞が止まったサーバーを起動して stderr に書き、できなければ何も作らずに失敗する、という 1 文がある |
| I2 の f(`/org` の動詞リファレンス) | 解消(d6951690) | `spawn`・`start`・`director` の行にゲートの位置と失敗の形がある。spawn は見張りのすぐあと、spawn ロックと台帳の読みの前で、`spawn failed:` で終わる。start --plan は台帳の読みと worktree の前で、director start は herdr の形だけ、今の登録を確かめる前である。どれも `spawn.go:648`、`feature.go:239`、`director_start.go:1079` の順と合う。4 つの写しは `cmp` で同一 |
| I2 の g(tech-debt 276 行) | 解消(d6951690) | Trigger の末尾に「当たったが払わなかった」の注がある。新しい 431・432 行も同じ commit で足された |
| I2 の h(計画と spec の box) | open | 計画の AC1〜AC9 と、Progress の Review・Verification・Test の artifact の行、spec の FR-A1〜A6 と AC-A1〜A6 は `[ ]` のまま。この run も触っていない。/pr の前に orchestrator が付ける |
| I3(help の「refused」) | help は解消(f0edc152)、skill の語は open | help は「the command fails with nothing made」になり、`TestOrgHerdrServerGate_TheHelpOfEachVerbSaysItStartsTheServer`(`org_herdr_server_test.go:772`)の期待も同じ。HEAD の binary の 3 つの `--help` に 3 回出て、`refused with nothing made` は report の外に残っていない(Probe C2-4)。skill に残る「断られ」は self-review の C2-1 で、register にはまだない。同じ語はほかにも 2 か所ある(下の I2-1) |
| Run 1 の Coverage gaps の 2 つ目(start --plan を本物のゲートで通す CLI のテストが無い) | 解消(f0edc152) | `TestOrgStartPlan_HerdrServerGate_StartsAStoppedServerOnceInStartsWords`(`org_herdr_server_test.go:731`)。Run 1 が勧めた形のとおり、答えを「止まっている、動いている」にして、stderr の `started the herdr server` が 1 回、行の頭が `ralph org start:`、起動が 1 回、状態の読み取りが 3 回(StartFeature のゲートが 2 回、Spawn のゲートが 1 回)であることを見る |

### Spec compliance(cycle 2)

| Acceptance criterion | Status | Evidence(Run 1 からの違い) |
| --- | --- | --- |
| AC1 止まったサーバーを起動して進む | Met | ゲートの本体(`ensureOrgHerdrServer`、`internal/cli/org_herdr_server.go:127`)と 4 か所のつなぎ(`org.go:943`・`:1161`・`:1278`、`org_director_headless.go:352`)は Run 1 と同じ。f0edc152 は、`switch` のあと(:142)と `await` の残り時間の取り直し(:186-187)にコメントを足しただけである。start --plan の成功の道に CLI のテストが加わった(上の表の最後の行)。spec の AC-A1 の Given/When は start --plan で、その実機は今もない |
| AC2 起動する子の環境 | Met(実機は一部) | `orgHerdrServerEnv` と `orgHerdrServerDropsEnv`(:302、:325)は変わっていない。実機の記録の書き方は I1 のとおり直った。起動したサーバーのプロセスそのものの環境は今も読んでいない |
| AC3 動いているときは起動しない、dry-run と launch_cmd は呼ばない | Met | コードは Run 1 と同じ(`spawn.go:647-651` の `!p.DryRun`、`director_start.go:1078-1082` の `!c.cmdLaunch`)。f0edc152 は、ゲートが冪等の early return より前にあることを Spawn の doc の step 1(`spawn.go:538` 以降)、PR① のコメント(:659-663)、`herdrgate.go:20-25` に書き、`TestOrgSpawn_HerdrServerGate_ARespawnOfAnActiveSeatAsksItToo`(`internal/org/herdrgate_test.go:95`)で、active な座席の打ち直しも通るゲートを 1 回呼び、台帳・receipt・driver・agmsg に何も足さないことを固定した。動いているサーバーなら CLI のゲートは状態を 1 回読むだけなので、AC3 の「起動しない」と合う |
| AC4 起動できないときは何も作らずに失敗 | Met | エラーの文は変わっていない(5 種類、7 通り。どれも `then run ralph org <verb> again` で終わる)。help が「fails」になり、spawn の `spawn failed:`(D2)と言葉が合った。上の打ち直しのテストは、失敗するゲートで active な座席の打ち直しが `SpawnOutcomeFailed` になり、何も変わらないことも見る |
| AC5 already running で終わっても読み直して進む | Met(偽物だけ) | コードは同じ。`afterExit` の doc(:208-213)が、同時に起動したときに負けた側が失敗するか、終わる子の pid で 1 行を出しうることと、打ち直せば進むことを書いた。register の 433 行が defer を記録する。2 つの ralph を同時に打つ実機はない |
| AC6 skill に書いてあり、2 つの sync が通る | Met | S4 の文に加えて、d6951690 が `/org` の動詞リファレンスの 3 行を足した(4 つの写しが同一)。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)が通る。FR-A6 の `director.md` は I2 の c の判断で変えていない。語のずれ(C2-1、I2-1)は AC6 の中身(起動を頼まない、`herdr` で接続を頼む)を変えない |
| AC7 3 つの動詞の `--help` | Met | HEAD の binary で 3 つの `--help` を打ち、前置きの 1 文と共通の段落を見た(Probe C2-4)。director start の help の別の段落に古い文が 1 つある(下の V2-1)。AC7 が求める段落そのものは正しい |
| AC8 テストが本物を起動しない | Met | `main_test.go:192-195` は Run 1 と同じ。新しい 2 本のテストも、ゲートの差し替え(`setOrgHerdrServerSeams`)か org 層の偽のゲート(`stubHerdrServerGate`)で動く |
| AC9 director start はゲートを残った登録の確認より前に呼ぶ | Met | 順は `director_start.go:1079`(ゲート)、`:1091`(`checkCurrentDirector`)、`:1095`(見張りのゲート)で、Run 1 と同じ(Probe C2-5)。テストも同じ |

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang) | pass(exit 0) | shellcheck と `sh -n`、settings の jq、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照、gofmt ok、`go vet ./...`(テストのファイルも型を確かめる)、golangci-lint 0 issues、staticcheck、branch の secret scan(c206fdce..227bfd28 clean) |
| `./scripts/plan-visual.sh digest <plan>` | `9bda99914bec` | `- Approved:` と同じ |
| `go build ./cmd/ralph`(出力は `/tmp`) | 成功 | 3 つの `--help` を打つためだけに作った |
| `cmp`(skill 2 つの 4 つの写し、recipe 2 つの写し) | 一致 | Probe C2-7 |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGENTS.md`、`README.md`、`docs/recipes/codex-seat-permissions.md`(2 つの写し)、`docs/quality/quality-gates.md`(2 つの写し) | Yes | d6951690 の文はコードの順と失敗の形に合う。quality-gates の Herdr server gate の行は root と template で同じ文(ファイル自体は `check-sync.sh` の KNOWN_DIFF) |
| `docs/specs/2026-08-01-org-runtime.md` の 2026-11-06 改訂 | Yes(語を除く) | (b) は失敗を「断る」と書く(I2-1) |
| `/org` と `/director` の skill(4 つの写し) | Yes(語を除く) | 自動起動、起動を頼まないこと、`herdr` で接続を頼むこと、ゲートの位置はコードと合う。「断られ」の語は self-review の C2-1 と I2-1 |
| `ralph org director start --help` | No | V2-1 |
| `ralph org spawn --help`、`ralph org start --help` | Yes | ゲートより後の拒否について、herdr を呼ばないと言う文はない |
| Spawn の doc、`herdrgate.go` の doc | Yes | ゲートは `spawn.go:648` で spawn ロックと台帳の読みより前にあり、doc の 3 か所が同じことを言う |
| 実機の記録 `docs/evidence/org-herdr-autostart-2026-11-06.md` | Yes | I1 |
| `docs/tech-debt/README.md` | 一部 | 318 行の注、431〜434 行の主張は HEAD のコードと合う(431 行の Spawn の順、432 行の `!c.cmdLaunch`、433 行の `afterExit` の doc、434 行の `orgHerdrServerDropsEnv` と `Compatible`)。Trigger の確かめと行の書き漏れは I2-2 と I2-3 |
| 計画と spec の box | 遅れ | I2-4(Run 1 の I2 の h) |

### Findings(cycle 2)

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V2-1 | LOW | `ralph org director start --help` の拒否の段落は「start is refused, and writes nothing and runs no herdr, when ... while a promoted session is attached」と言う。この文は base(c206fdce)から変わっていないが、b8580f0a でゲートが今の登録の確認より前に入ったので、herdr の形の start は、昇格したセッションの登録で断られる前に `herdr status server --json` を打ち、サーバーが止まっていれば `herdr server` も起動して残す。同じ help の最後の段落は「first makes sure the herdr server runs, before it looks for an attached director」と言うので、help の中で 2 つの文が食い違う。`/director` skill の「立てる」の 5 と register の 431 行は、この例外を書いている。Run 1 の verify、self-review の 2 回、sync-docs、codex のどれもこの文を挙げていない。挙動の問題ではなく文の問題である | `internal/cli/org_director_headless.go:100`(base の同じ行と同じ文)、`internal/org/director_start.go:1079`(ゲート)と `:1091`(`checkCurrentDirector`。昇格したセッションの登録は `director_start_cleanup.go:32` の `DirectorAttachedError` で断る)。ほかに挙げた拒否(`enabled = false`、codex の autonomous と edits、相対パスの `agmsg_home`、`HERDR_PANE_ID`)は、CLI か `checkDirectorStart` でゲートより前に断るので、文のとおり herdr を打たない。help の文を固定するテストはない(Probe C2-5) | 直すなら、昇格したセッションの場合を一覧から出し、「herdr の形の start は、そのときもう herdr のサーバーを確かめている(下の段落)」と 1 文で書く。Go の help の文字列が変わるので、cycle 2 が上限の最後の run である今は、上限を上げて全パイプラインを回し直すか、431 行に下位の項として足して defer するかを cross-review の triage で決める |
| I2-1 | INFO | self-review の C2-1(ゲートの失敗を skill が「断られ」と書くが、spawn は `spawn failed:` で `rejected` ではない)と同じ語が、C2-1 の挙げた `/org` skill の前提(35 行)と `/director` skill の「立てる」の 1(162 行)のほかに、2 か所ある。`docs/specs/2026-08-01-org-runtime.md` の 2026-11-06 改訂の (b)(「worktree も台帳の記録も作らずに断る」。spawn も含む)と、`/org` skill の `director` の行(「何も書かず herdr にも何も作らずに断られる」)である。director start には `rejected:` と `failed` の区別がなく、`director` の行はほかの拒否も「断られる」と書くので、後者は C2-1 の 162 行と同じ程度の揺れである | Probe C2-6 | C2-1 を register に足すときは、この 2 か所も行に入れる。直すときは spec の改訂の (b) を「失敗する」にする |
| I2-2 | INFO | tech-debt の Trigger の確かめ(全 open 行の Trigger を python で取り出し、f0edc152 と d6951690 が触った名前と突き合わせた)。(1) 191 行の Trigger「the next change to ... the `Spawn` doc」に f0edc152 がまた当たった(step 1 の段落と PR① のコメント)。d6951690 の 2026-11-06 の注は step 0 の変更と「2740 to 2767 lines ... at its HEAD」を書くが、`spawn.go` は今 2778 行で、step 1 の変更は注にない。(b)・(c) は払われていない。(2) 430 行の (e) の Trigger「the next change to `ResolveRoleModel` or to the idempotent respawn (document the two refusals in its doc, ...)」は、b8580f0a(冪等の打ち直しもゲートを通るようになった)と f0edc152(その打ち直しの doc を書き換えた)に当たると読める。cycle 1 の sync-docs の report は 430 行を当たりにも退けたものにも挙げていない。(3) 288 行の Trigger「the next change to the idempotent branch of `Spawn`」は、枝のコード自体は変わっておらず、その前にゲートが入っただけなので、当たるかは判断が分かれる | `docs/tech-debt/README.md` 191・288・430 行、Probe C2-8 | cycle 2 の /sync-docs が、191 行の注に step 1 と 2778 行を足し、430 行の (e) に注を足すか当たらない理由を report に書き、288 行の判断を report に残す |
| I2-3 | INFO | 新しい行の書き漏れ。(1) 431 行は director start について、今の登録の確認で断られた start だけを挙げる。`checkDirectorStart` は見張りのゲート(`director_start.go:1095`)もゲートのあとに置くので、見張りが止まりかけ・印が古い・`--once` で断られた herdr の形の start も、サーバーを起動したまま残しうる。(2) 434 行の (c) は 10 秒が書かれた場所を help、2 つの skill、`orgHerdrServerWaitDeadline` の既定値とするが、d6951690 のあとは `docs/quality/quality-gates.md`(2 つの写し)と 2026-08-01 の spec の改訂の (b)、`/org` skill の `spawn` と `start` の行にもある | `docs/tech-debt/README.md` 431・434 行、`director_start.go:1079-1095` | /sync-docs が 2 行に句を足す。挙動は変わらない |
| I2-4 | INFO | Run 1 の I2 の h と同じ。計画と spec の box、計画の Progress の 3 行が `[ ]` のまま | 計画の 66〜74 行と 129〜131 行、spec の 39〜44 行と 68〜73 行 | /pr の前に orchestrator が付ける。AC2 と AC5 は「一部」「偽物だけ」なので、付けるなら PR の本文にその範囲を書く |

### Observational checks(cycle 2)

- HEAD(227bfd28)から作った binary で `ralph org spawn --help`、`ralph org start --help`、`ralph org director start --help` を打ち、前置きの 1 文と共通の段落(「the command fails with nothing made」)を 3 つとも見た。director start の拒否の段落(V2-1)もこのとき読んだ
- 実機の記録の AC2 の表と、その下の段落を、Run 1 の I1 の勧めに照らした
- f0edc152 の 7 ファイルの diff を読み、本番のコードの変更が help の 1 行だけであることを確かめた(Probe C2-3)

### Coverage gaps(cycle 2)

- Go のテストの結果(/test の担当)。新しい 2 本のテストも、中身を読み、`go vet` で型が通ることを見ただけである
- start --plan と director start の実機。spec の AC-A1 は start --plan を Given/When に置くが、実機の記録は spawn だけを通した
- 2 つの ralph が同時に起動する実機(AC5、register 433 行)、`status server --json` の無い古い herdr、director の Bash ツールや codex の sandbox の中からの切り離し起動(計画の R4)
- 起動したサーバーのプロセスそのものの環境(AC2)
- V1 の 2 つのソケットの道(register 318 行)と V2-1 の文は、テストも実機もない

確かさを一番上げる追加の確かめは、分離した herdr(`docs/recipes/isolated-herdr-server.md`)で、サーバーを止めた状態(ソケットのファイルが残った状態)から `ralph org start --plan` を 1 回打ち、stderr の 1 行が `ralph org start:` で始まって 1 回だけ出ること、leader の座席ができること、`pgrep -fl 'herdr server'` で新しいサーバーが 1 つだけであることを見ることである。S5 の手順をそのまま使え、spec の AC-A1 の文面どおりの道を実機で通せる。V2-1 だけを決定論で押さえるなら、director start の help の拒否の段落に「runs no herdr」と昇格したセッションの場合が同じ一覧に並ばないことを見るテストが一番小さい。

### Verdict(cycle 2)

pass

- Verified: AC1〜AC9 を HEAD のコード、テストの assert、help の出力で確かめ直した。f0edc152 の本番のコードの変更は help の 1 行だけで、ゲートの位置と分岐は Run 1 と同じである。Run 1 の I1 と start --plan の CLI のテストの穴は解消し、I2 の a〜g は d6951690 で解消した。静的解析の全項目と 2 つの sync。計画の digest は `9bda99914bec` のまま
- Partially verified: AC2 の実機(ペインの環境での代わりの確かめ)。AC5 は偽物だけ。start --plan の成功の道は CLI のテストまでで、実機はない
- Not verified: Go のテストの結果と `run-test.sh`(/test)。start --plan と director start の実機。同時の起動。古い herdr。sandbox の中からの起動
- 所見: LOW 2 件(V1 は register 318 行に defer、V2-1 は新しい)、INFO 4 件(I2-1〜I2-4)。どれも merge を止めない
- 回すもの: V2-1 は cross-review の triage で、上限を上げて直すか 431 行に足して defer するかを決める。I2-1〜I2-3 は cycle 2 の /sync-docs(self-review の C2-1 を register に足すときに I2-1 の 2 か所も入れる)。I2-4 は /pr の前に orchestrator
- Insight event: 同じ `docs/insights/events/2026-11-05-org-herdr-autostart.jsonl` に 1 行足した(phase verify、cycle 2、verdict pass、low 2)。cycle 1 の verify の行は、Run 1 に LOW の V1 があったのに `"low":0` を記録していた。append だけのファイルなので、その行は直していない
