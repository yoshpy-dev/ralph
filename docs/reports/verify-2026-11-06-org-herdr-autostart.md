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
