# org-herdr-single-tab

- Status: Approved
- Approved: 2026-11-06 sha256:61ba0f08bb4c
- Owner: Claude Code
- Date: 2026-11-06
- Related request: 2026-11-06 のユーザーの依頼「herdrを立ち上げると、orgごとにそれぞれのワークスペースが正常に存在していたが、どのワークスペースも「1（タブ）、leader（タブ）、implementer（タブ）、reviewer（タブ）」のようになっていました。…理想は、orgごとにそれぞれのワークスペースが存在し、どのワークスペースもそれぞれ1つのタブのみ存在し、そのタブの中で「leader（ペイン）、implementer（ペイン）、reviewer（ペイン）」のような構成になっていることです。…各ペインにはそれぞれleader、implementer、reviewerの名前をつけたいです。」。/spec で決めた仕様は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B(FR-B1〜B8、AC-B1〜B9)。同じ仕様の A(herdr の自動起動)は PR #303 で merge した
- Related issue: N/A
- Type: feat
- Branch: feat/org-herdr-single-tab

## Objective

org のワークスペースをタブ 1 つにし、座席をそのタブの中のペインとして作る。最初の座席は `herdr workspace create` が作るルートペインに入り、2 席目以降は生きている座席のペインを分けて入る。各ペインには座席 id のラベルを付け、止める・解散する・補償で閉じる前の確認はペインのラベルで行う。`ralph org director start` の headless のワークスペースも、タブ 1 つ・ペイン 1 つにする。

## Scope

- **driver(FR-B1〜B3、FR-B8)**: `internal/org/driver/herdr.go` の `WorkspaceCreate` が `--env` を取り、ワークスペースの id に加えてルートペインとタブの id を返すようにする。`PaneGet` はペインのラベルも返す。新しく `PaneSplit`(`pane split <id> --direction right|down --ratio 0.5 --cwd --env`)、`PaneRename`、`TabRename`、`PaneList`(`pane list --workspace <id>`)を足す。`--env` があるかの確かめを、使うサブコマンド(`workspace create`、`pane split`、`tab create`)ごとにする。
- **CLI のテストの偽物**: `internal/cli/org_test.go` の `herdrStub` が、新しいサブコマンドと `pane get` のラベルに答えるようにする。
- **停止前の確認(FR-B6)**: `confirmSeatPane`(`internal/org/verbs.go`)を、ペインのラベルで確かめる形にする。前の版で作った座席(タブのラベルが座席 id)と、分けたあとラベルを付ける前に止まった spawn のペインも受け入れる(D4)。
- **spawn の配置(FR-B1〜B5)**: `resolveWorkspace` と座席のペインの作り方(`internal/org/spawn.go`)を D1〜D3 のとおりに変える。`fakeHerdr`(`internal/org/spawn_test.go`)と、呼び出しの順を確かめているテストを新しい形に合わせる。
- **director(FR-B7)**: `StartDirector`(`internal/org/director_start.go`)が、ルートペインに director を置き、ペインとタブにラベルを付ける。
- **文書と help**: `/org` と `/director` の skill(4 つの写しずつ)、`ralph org spawn` と `ralph org director start` の `--help`、org runtime の仕様の改訂、分離した herdr の recipe のうち、座席をタブと書いているところ。
- **実機の確かめ(AC-B9)**: 分離した herdr で、座席 3 席の配置、1 席の停止、全席の停止のあとの spawn を確かめ、`docs/evidence/` に残す。

## Non-goals

- 役割で位置を固定すること(`herdr pane swap`)。配置は spawn の順(仕様の Design decisions)
- 既存のワークスペースを新しい形に移すこと。前の版のワークスペースにあとから spawn した座席は、生きている座席のペインを分けて入る
- ペインの大きさを座席の数に合わせて均すこと(3 席を超えると下の列は 1/2 ずつ小さくなる)
- 仕様の A に残った穴(同時起動、別のソケットの director start など。tech-debt 318・431〜434 行)
- サーバーの再起動で戻った古い org のワークスペースの検出と片付け

## Assumptions

2026-11-06 に分離した herdr 0.7.5 で確かめた(仕様の Research findings と、計画を書く前の確かめ)。

- `herdr workspace create --cwd X --label L --env K=V` の応答は、ワークスペース(`workspace_id`)、ラベル「1」のタブ(`tab.tab_id`)、ルートペイン(`root_pane.pane_id`)を含み、`--env` の値はルートペインに入る。
- `herdr pane split <pane> --direction right|down --ratio 0.5 --cwd X --env K=V --no-focus` は同じタブに新しいペインを作り、応答の `pane.pane_id` にその id を返す。`--env` の値は新しいペインにだけ入り、親のペインの環境は受け継がない。claude が動いているペインも分けられ、分けたあとも agent は `idle` のまま。
- `pane split` に `--label` は無い。`herdr pane rename <pane> <label>` で付けたラベルは `pane get` の `pane.label` に出て、`agent start` のあとも、サーバーの再起動(SIGKILL のあと)のあとも残る。`herdr tab rename <tab> <label>` もある。
- `herdr pane list --workspace <id>` は、そのワークスペースのペイン(`pane_id`、`tab_id`、ラベルがあれば `label`)を返す。
- 1 つのタブのペインをすべて閉じると、ワークスペースごと消え、`workspace get` は `workspace_not_found` を返す。`closeOrgWorkspace`(`verbs.go`)は、herdr に無いワークスペースを「すでに閉じた」として記録する。
- ペインを閉じると、同じタブの隣のペインがその場所を埋める(herdr の分割の振る舞い)。

## Affected areas

- `internal/org/driver/herdr.go`、`herdr_test.go`(API と `--env` の確かめ)
- `internal/org/spawn.go`(`HerdrClient`、`resolveWorkspace`、座席のペインの作り方、配置のロック、spawn_step の Details)、`internal/org/verbs.go`(`confirmSeatPane`)、`internal/org/director_start.go`(director のワークスペース)、`internal/org/lockfile.go`(配置のロックの名前)
- `internal/org/spawn_test.go`(`fakeHerdr`)と、herdr の呼び出しの順を確かめているテスト(`director_start_test.go`、`director_start_cleanup_test.go`、`director_stop_test.go`、`feature_test.go`、`herdrgate_test.go`、`verbs_test.go` など)
- `internal/cli/org_test.go`(`herdrStub`)と、herdr の呼び出しを数えている CLI のテスト(`org_runtime_env_test.go`、`org_start_plan_config_test.go` など)、`internal/cli/org.go` と `org_director_headless.go` の `--help`
- `.claude/skills/{org,director}/SKILL.md` と写し、`docs/specs/2026-08-01-org-runtime.md`、`docs/recipes/isolated-herdr-server.md`(と写し)、`docs/evidence/`(実機の記録)

## Visual review

- Page: `.harness/state/plan-visual/org-herdr-single-tab.html`(やさしい図 2 枚: `overview`・`easy-stop`。折りたたみの中に `detail-overview`・`detail-sequence`・`detail-anchor`・`detail-confirm`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: 全体と 6 枚の図を `shot` で撮って読んだ。図 1 の見出しの対比の言い方を直し、consult の指摘で D4 を補償だけに絞ったのに合わせて詳細 4 の 3 行目を直して、撮り直した

## Design decisions

Critical forks: None。仕様で人が決めた点(配置は spawn の順、director も同じ形、既存のワークスペースは移さない)に従い、そのほかは次の既定で進める。

- D1 座席のペインの作り方(FR-B1〜B3):
  - 使えるワークスペースが無い(D2)とき: `workspace create --cwd <cwd> --label <org_id> --env <PaneEnv> --env RALPH_PRIMARY_CLI=<driver>` のルートペインに座席が入る。続けてタブのラベルを org_id にし(`tab rename`)、ペインのラベルを座席 id にする(`pane rename`)。
  - 使えるワークスペースがあるとき: 起点の候補は、台帳でその org の座席(active か in-flight で、ペインの id を記録したもの)のうち、herdr が `pane list --workspace` でそのワークスペースにあると答えたペイン。候補が 1 つなら右に、2 つ以上なら leader 以外で最後に spawn した座席のペインを下に、`pane split --ratio 0.5` で分け、新しいペインに座席 id のラベルを付ける。
  - 候補が 0 のとき(前の版のワークスペースで座席のペインがもう無い、ralph の作っていないペインだけが残っているなど): 今までどおり `tab create` でタブを足し、ペインに座席 id のラベルを付ける。
- D2 ワークスペースを使う前の確かめ(FR-B4): 台帳が開いていると記録したワークスペースの label を `workspace get` で読む。herdr に無ければ、またはラベルが org_id でなければ、`org_workspace_closed` を記録し(Details に理由)、D1 の「無いとき」に進む。
- D3 配置のロック(FR-B5): D2 の確かめから、ペインのラベル付けと spawn_step の記録までを、org ごとのロック(`layout-<org_id>.lock`、台帳のディレクトリ)の中で行う。待つのは spawn の残り時間まで。座席ごとの spawn ロックは今までどおり別にとる。
- D4 停止前の確かめ(FR-B6): `pane get` が返すペインのラベルが期待する座席 id なら受け入れる。ペインのラベルが空で、タブのラベルが座席 id なら、前の版の座席として受け入れる。どの場合もワークスペースのラベルが org_id であることを確かめる。それ以外は今までどおりラベルの食い違い(`errLabelMismatch`)で、閉じない。director も同じ規則で、期待するラベルは `DirectorTabLabel(token)` と `DirectorWorkspaceLabel(token)`。さらに補償(`compensateStale`)だけは、古い spawn の最後の記録が D5 の spawn_step(ペインを作ったときのタブの id を記録したもの)のときに限り、ペインのラベルが空で、そのペインが記録したタブの中にあり、ワークスペースのラベルが org_id なら、その spawn が作ったあと名前を付ける前に止まったペインとして受け入れる(仕様 FR-B6 の「補償で閉じてよい」)。ルートペイン(タブは「1」のまま)、前の版のタブの中で分けたペイン、`tab create` で足したペインのどれでも当てはまる(codex の計画助言の所見 2)。stop・disband・director の確かめはこの場合を受け入れない(consult の指摘。同じ org のタブで人が開いたシェルを閉じないため)。
- D5 spawn_step の記録: ペインを作ったら、名前を付ける前に spawn_step を書く。PaneID はそのペインで、Details は `pane_created` に、作り方(`root`、`split-right`、`split-down`、`tab`)と、herdr が応答で返したそのペインのタブの id(`tab=<id>`)を添える。D4 の補償の受け入れはこのタブの id を読む。`tab_created` を読むコードは無い(`spawn.go` が書くだけ)。
- D6 失敗の扱い: `pane split` と `workspace create` の失敗はペインを作っていないので、今までどおり spawn_failed にする。`pane rename` の失敗は、作ったペインを補償で閉じて spawn_failed にする(`closeSpawnPane`。ラベルの確かめは無い)。`tab rename` の失敗は見た目だけの問題なので、spawn を失敗にせず spawn_step の Details に書く。ルートペインを閉じるとワークスペースも消えるが、次の spawn が D2 で扱う。
- D7 director(FR-B7): `workspace create --label DirectorWorkspaceLabel(token) --env <PaneEnv>` のルートペインに director が入り、ペインのラベルを `DirectorTabLabel(token)`、タブのラベルを `DirectorWorkspaceLabel(token)` にする。`RecordWorkspace` と `RecordPane` は今と同じ順で書く。
- D8 driver の API: `WorkspaceCreate` は extraEnv を取り、`HerdrWorkspace{WorkspaceID, TabID, RootPaneID}` を返す。`PaneGet` は `HerdrPane{TabID, WorkspaceID, Label}` を返す。`TabCreate` は D1 の候補 0 のときのために残し、ペインの id に加えてタブの id を返す。新しい `PaneSplit` も、新しいペインの id とタブの id を返す(D5 が記録する)。既存のメソッドの形を変えるのは S1 だけで、呼び出し側は S1 の中で機械的に合わせる(振る舞いは変えない)。

## Acceptance criteria

- [x] AC1(仕様の AC-B1): 新しい org で leader → implementer → reviewer の順に spawn すると、herdr への呼び出しは `workspace create`(leader、`--env` 付き)、`tab rename <tab> <org_id>`、`pane rename <root> leader`、`pane split <leader のペイン> --direction right`、`pane rename <新> implementer`、`pane split <implementer のペイン> --direction down`、`pane rename <新> reviewer` になり、`tab create` は呼ばれない。org のテストで固定し、実機(AC11)でタブが 1 つ・左半分・右上・右下・ラベルを確かめる。
- [x] AC2(仕様の AC-B2): reviewer を implementer より先に spawn しても、3 つのペインが 1 つのタブに入り、`tab create` は呼ばれない。
- [x] AC3(仕様の AC-B3): 3 席の org で implementer を止めると、implementer のペインだけに C-c と `pane close` が送られる。
- [x] AC4(仕様の AC-B4): 台帳が開いていると記録したワークスペースが herdr に無い(またはラベルが org_id でない)とき、次の spawn は `org_workspace_closed` を記録してから新しいワークスペースを作る。
- [x] AC5(仕様の AC-B5): 記録したペインの id が、別の org のワークスペースのペイン、または別のラベルのペインを指しているとき、止める・解散する・補償で閉じる前の確かめが閉じない。ラベルが空で、タブのラベルが org_id のペインも、stop と disband は閉じない。
- [x] AC6(仕様の AC-B6): 前の版で作った座席(タブのラベルが座席 id、ペインのラベルが空)を止め、解散できる。
- [x] AC7(仕様の AC-B7): ペインを作ったあと名前を付ける前に止まった spawn(最後の記録が D5 の spawn_step)の補償(`compensateStale`)が、そのペインだけを閉じ、打ち直しのあとに名前の無いペインが残らない。ルートペイン(`tab rename` も済んでいない)、新しい形のタブの中の分割、前の版のタブの中の分割、`tab create` の 4 つの場合を、作った直後で止まった状態から確かめる。記録したタブと違うタブにある、名前の無いペインは閉じない。最後の記録が spawn_started(ペインの記録が無い)の座席には当てはまらない。
- [x] AC8(仕様の AC-B8): `ralph org director start`(herdr の形)は、ルートペインに director を置き、ペインとタブにラベルを付け、`tab create` を呼ばない。`director stop` はそのペインとワークスペースを閉じ、前の版で立てた director(タブのラベル)も閉じられる。
- [x] AC9(FR-B5): 同じ org の 2 席を同時に spawn しても、2 席とも同じタブに入り、同じペインを 2 席が分けない(配置のロック)。
- [x] AC10(FR-B8): `workspace create` と `pane split` に `--env` があるかを、それぞれのヘルプで確かめる。無い herdr では今と同じ警告を出し、`--env` なしで作る。
- [x] AC11(仕様の AC-B9): 分離した herdr で、3 席の配置(タブ 1 つ、左半分・右上・右下、ラベル)、implementer の停止のあとの配置、全席の停止のあとの spawn(新しいワークスペース)を確かめた記録が `docs/evidence/` にある。
- [x] AC12: `/org` と `/director` の skill、2 つの動詞の `--help`、org runtime の仕様の改訂、recipe が、座席をタブでなくペインと書き、`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る。

## Implementation outline

1. S1 driver の API: `internal/org/driver/herdr.go` に `PaneSplit`、`PaneRename`、`TabRename`、`PaneList` を足し、`WorkspaceCreate` を extraEnv と `HerdrWorkspace` に、`PaneGet` を `HerdrPane` に、`TabCreate` をペインとタブの id を返す形に変える(D8)。`--env` の確かめをサブコマンドごとにする(D7 の前提、AC10)。`HerdrClient`(`spawn.go`)、呼び出し側(`spawn.go` の `resolveWorkspace`、`director_start.go`、`verbs.go` の `confirmSeatPane`)、`fakeHerdr` を機械的に合わせ、振る舞いは変えない — verify: `go build ./... && test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/... ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/driver/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Spawn|Stop|Disband|Director|Gate' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Org|Director'`
2. S2 CLI の偽物: `internal/cli/org_test.go` の `herdrStub` が `pane split`(新しいペインの id を返し、そのペインのタブとワークスペースを記録する)、`pane rename`、`tab rename`、`pane list --workspace`、`pane get` のラベル、`workspace create` のルートペインの記録、`workspace create --help` と `pane split --help` に答えるようにする。CLI のテストの期待は変えない — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Org|Director'`
3. S3 停止前の確かめ: `confirmSeatPane` を D4 の規則にし、補償だけが使う受け入れ(最後の記録が D5 の spawn_step のときの、ラベルが空で記録したタブの中にあるペイン)を `compensateStale` からだけ渡せる形にする。D5 の Details の書き方と読み方の関数もこのスライスで置く。前の版の座席、この版の座席、補償でのラベルを付ける前に止まった spawn のペイン、stop と disband では同じペインを閉じないこと、別の org や別のラベルのペインの場合をテストする(AC5、AC6、AC7 の確かめの側) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Confirm|Stop|Disband|Compensat|Stale|Director'`
4. S4 spawn の配置: `spawn.go` を D1〜D3、D5、D6 のとおりにする(配置のロックの名前は `lockfile.go`)。`fakeHerdr` に分割とラベルとペインの一覧を足し、呼び出しの順を確かめるテストを新しい形に合わせる(確かめる内容を弱めない)。CLI のテストのうち herdr の呼び出しを数えるものも合わせる。AC1〜AC4、AC7、AC9 のテストを足す — verify: `go build ./... && test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Org|Director|Spawn|Start'`
5. S5 director: `StartDirector` を D7 のとおりにし、director のテストを合わせ、AC8 のテストを足す — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Director' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Director'`
6. S6 文書と help: `ralph org spawn` と `ralph org director start` の `--help`、`/org` と `/director` の skill(`scripts/sync-skills.sh` と template の写し)、`docs/specs/2026-08-01-org-runtime.md` の改訂、`docs/recipes/isolated-herdr-server.md`(と写し)で、座席と director をタブと書いているところを直す(AC12) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Help' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill|Director'`
7. S7 実機の確かめ: このブランチの ralph を build し、分離した herdr(`docs/recipes/isolated-herdr-server.md`)で、scratch の repo(`[org.permissions].default = "guarded"`、`[org].agmsg_home` は recipe の「private agmsg」の写し。spawn は agent の起動のあとで agmsg に join するので、存在しない場所では座席のペインが補償で閉じられる)に、recipe の `osascript` の shim を PATH の先頭に置いて、claude の座席を役割の雛形と初めの指示なし(未知の役割、`--prompt` なし。モデルに何も送らない)で 3 席 spawn し、`herdr tab list`・`pane list`・`pane layout` で配置とラベルを、implementer の停止のあとの配置を、全席の停止のあとの spawn を確かめ、`docs/evidence/org-herdr-single-tab-2026-11-06.md` に残す。フォルダ信頼の画面は recipe の手順で答える。ユーザーの herdr には触れず、終わったら分離した環境を片付ける(AC11) — verify: `test -s docs/evidence/org-herdr-single-tab-2026-11-06.md && test ! -e /tmp/ralph-iso`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt・go vet・golangci-lint・staticcheck・skill と template の同期)。
- Spec compliance criteria to confirm: 仕様の FR-B1〜B8 と AC-B1〜B9 を、上の AC1〜AC12 とテストの名前、実機の記録に照らす。D4 の受け入れる 3 つの場合と断る場合が、コードとテストで同じであること。
- Documentation drift to check: 座席や director を「タブ」と書く文(skill の 4 つの写し、`--help`、recipe、org runtime の仕様、`AGENTS.md` の地図、`docs/quality/quality-gates.md`)。`fakeHerdr` の doc コメントの「TabCreate puts the pane … in a tab of its own」。
- Evidence to capture: `docs/reports/verify-*.md`、実機の記録 `docs/evidence/org-herdr-single-tab-2026-11-06.md`。

## Test plan

- Unit tests: driver の新しいメソッドの argv と応答の読み取り(`pane split` の `pane.pane_id`、`workspace create` の `root_pane` と `tab`、`pane get` の `label`、`pane list` の配列)、サブコマンドごとの `--env` の確かめ。D4 の規則の表(受け入れる 3 つと断る場合)。D1 の起点の選び方(候補 0・1・2・3、leader が候補にいない場合、台帳にあって herdr に無いペイン)。
- Integration tests: org の Spawn を `fakeHerdr` で通して、AC1・AC2・AC4・AC7・AC9 の呼び出しの順と台帳の記録を確かめる。Stop と Disband を通して AC3・AC5・AC6 を確かめる。StartDirector と StopDirector で AC8 を確かめる。CLI の spawn と start を `herdrStub` で通して、herdr の呼び出しの行を確かめる。
- Regression tests: `./scripts/run-test.sh`。既存のテストで、タブを前提にした期待を変えるものは、確かめる内容(どのペインを閉じるか、何を記録するか)を保ったまま新しい呼び出しに合わせる。
- Edge cases: AC7 の 4 つの場合を、spawn_step を書いた直後に止まった台帳と herdr の状態から打ち直すテスト(閉じるのは作ったペインだけ、打ち直しのあとに名前の無いペインが残らない)、ルートペインの `pane rename` の失敗(ペインを閉じ、ワークスペースが消える)、`tab rename` の失敗(spawn は続く)、起点のペインが分ける直前に閉じた(`pane split` が `pane_not_found`)、前の版のワークスペースで座席がすべて止まっている(候補 0 で `tab create`)、`--dry-run`(herdr を呼ばない)。
- Evidence to capture: `docs/reports/test-*.md`、実機の記録。

## Risks and mitigations

- R1 テストの書き換え: `fakeHerdr` と CLI の偽物を前提にしたテストが多い(`tab_create` を書くテストは org に 32 か所、CLI に 9 か所)。S1 と S2 で API と偽物を先に変え、S4 で呼び出しの順を合わせる。期待を書き換えるときは、確かめる内容(閉じるペイン、記録する出来事)を変えず、計画とレポートに書く(テストの規則)。
- R2 別の org のペインを分ける: herdr が id を振り直すと、記録したペインの id が別のペインを指すことがある。起点は、ラベルが org_id と確かめたワークスペースの中で herdr が答えたペインからだけ選ぶ(D1、D2)。
- R3 確かめと分割の間にワークスペースが消える: 同時に打たれた stop が最後のペインを閉じると、`pane split` が `pane_not_found` で失敗する。spawn は spawn_failed になり、打ち直せば D2 が新しいワークスペースを作る。配置のロックは stop をふさがない。
- R4 古い herdr: `pane split`・`pane rename`・`tab rename`・`pane list --workspace` は 0.7.5 で確かめた。それより古い herdr で無ければ spawn が失敗する。org runtime はもともと 0.7.5 を前提にしている。
- R5 前の版のワークスペースとの混在: 前の版で作った org に新しい座席を spawn すると、生きている座席のペインを分けて、そのタブの中に入る(D1)。止める・解散するは D4 で前の版の座席も受け入れる。
- R6 3 席を超える org: 4 席目からは最後の非 leader のペインを下に半分ずつ分けるので、下ほど小さくなる(Non-goals)。
- R7 戻すとき(codex の計画助言の所見 1): 戻した版の止める前の確かめは、タブのラベルが座席 id であることを求める。この版の座席は org_id のタブの中にあるので食い違いになり、`--force` を付けても C-c も `pane close` も送らずに止めたと記録し、agmsg の team からも外す。エージェントは動き続ける。そのため、戻す前にこの版で動いているものを片付ける手順を、Rollback に書き、PR の本文にも書く。

## Rollout or rollback notes

- merge のあとの次の release で配る。`ralph.toml` も台帳の形も変わらない(spawn_step の Details の語と、spawn が書く `org_workspace_closed` が増えるだけ)。
- 動いている org はそのまま動く。新しい形にしたい org は、解散して立て直す。
- 戻すときは PR を revert する。戻した版はこの版の座席と director を止められない(R7)ので、戻した binary を入れる前に、この版の binary のまま次を済ませる。(1) この版で立てた org をすべて `ralph org disband` で解散する。(2) この版で立てた headless の director を `ralph org director stop` で止める。(3) `herdr workspace list` で、それらのワークスペース(label が org_id か `ralph_director_<token>`)が残っていないことを確かめる。
- 片付けの前に戻してしまったときは、`--force` を使わない(止めたと記録するだけで、エージェントは動き続ける)。この版の binary に戻して (1)〜(3) を済ませるか、残ったワークスペースを `herdr workspace close <id>` で閉じてから、戻した版で `ralph org disband --force` を打って台帳を合わせる。

## Open questions

なし。

## Progress checklist

- [x] Plan reviewed — consult-advisor(下書きのあと): S7 の agmsg を recipe の private の写しにする(spawn は agent の起動のあと agmsg に join する)、D4 の名前の無いペインの受け入れを補償だけに絞る、S1 の検証に CLI を足す、の 3 点を入れた。plan advisory: ran with codex(`[reviewer].codex_model` と `codex_effort` の値)。所見は 2 件で、2 件とも計画に入れた(2026-10-11 の指示で、計画助言の修正提案は聞かずに直す)。(1) [HIGH] 戻した版はこの版の座席を止められず、`--force` は止めたと記録するだけでエージェントが動き続ける → Rollback を「戻す前に解散・director の停止・ワークスペースの確かめ」に書き直し、`--force` で済むという記述を消し、R7 を足した。(2) [MEDIUM] 前の版のタブの中の分割や `tab rename` の失敗のあとで止まった spawn のペインが、補償の受け入れに当てはまらず残る → D5 で作ったときのタブの id を記録し、D4 の補償の受け入れを「名前が空で記録したタブの中」にし、AC7 を 4 つの場合に広げ、Test plan に止まった状態からの打ち直しを足した。図の詳細 4 と「気をつけること」も直した
- [x] Plan approved — 2026-10-11 のユーザー指示(作った計画は自動承認する)により自動承認した
- [x] Branch created — `feat/org-herdr-single-tab`(task worktree `.claude/worktrees/org-herdr-single-tab`、main 920832dd から)
- [x] Implementation started
  - S1 c6b44e28(implementer): driver の `PaneSplit`・`PaneRename`・`TabRename`・`PaneList` と、`WorkspaceCreate`・`TabCreate`・`PaneGet` の形の変更、サブコマンドごとの `--env` の確かめ。ずれ: `WorkspaceCreate` は S1 の間 extraEnv があるときだけ PaneEnv を付けた(今の引数を保つため。S5 で外した)
  - S2 c39a4e17(implementer): CLI のテストの `herdrStub` が新しい操作に答える。ずれ: `tab create` で作ったペインだけは閉じても偽物の状態に残す(既存のテストが固定しているため)
  - S3 4950e51f(implementer): `confirmSeatPane` をペインのラベルで確かめる形にし、補償だけが使う `confirmStaleSeatPane` と D5 の `paneCreatedDetails`・`paneCreatedTab` を置いた
  - S4 9df9735c(implementer): spawn が座席を 1 タブのペインとして作る(配置のロック、ワークスペースの確かめ、分割の起点、D5 の記録)。ずれ: step 名 `layout_lock`・`manifest_read` を足した。`tab rename` の失敗の記録は空白区切り。implementer が `spawn.go` のコメント 3 か所を python で書き換えた(Edit を使う決まりに反する。U+FFFD が無いことを orchestrator が確かめた)。残る窓: 最初の座席で `WorkspaceCreate` から spawn_step までの間(`tab rename` の 1 回)にプロセスが落ちると、名前の無いルートペインが記録なしで残る(ワークスペースは記録済みなので disband で閉じられる)
  - S5 3b7d188b(implementer): headless の director をルートペインに置き、ペインとタブにラベルを付ける。`WorkspaceCreate` は PaneEnv を常に付ける。ずれ: 手渡しの一覧に無いテストのファイル 4 つ(director の引数と出力を固定するもの)も合わせた
  - S6 1468232d(implementer): spawn・stop・disband の help、`--timeout-ms` の help、コメント、`/org`・`/director` の skill と写し、org runtime の仕様の改訂、分離した herdr の recipe。残り: `docs/recipes/director-launch-cmd.md`・`docs/recipes/director-wake.md`・`.claude/rules/ralph/git-commit-strategy.md`(と写し)の古い文は sync-docs に回す
  - S7 b4bf05bf(inline。実機の確かめはユーザーの herdr に触れない手順が要るので、orchestrator が行った): `docs/evidence/org-herdr-single-tab-2026-11-06.md`。検証コマンドは通った。機械的な確認: 不要な変更 clean、誤字と貼り間違い clean、デバッグの残り clean、秘密情報 clean、範囲外のファイル clean
- [x] Review artifact created — `docs/reports/self-review-2026-11-06-org-herdr-single-tab.md`(cycle 1 97526693、cycle 2 4c2c324b。どちらも Merge: yes)
- [x] Verification artifact created — `docs/reports/verify-2026-11-06-org-herdr-single-tab.md`(cycle 1 feb2e943、cycle 2 609e95ff。どちらも pass)
- [x] Test artifact created — `docs/reports/test-2026-11-06-org-herdr-single-tab.md`(cycle 1 d4fa6ce2、cycle 2 1c74df5f。どちらも pass)
  - sync-docs: cycle 1 a6fe3a25、cycle 2 は `docs: sync docs for org-herdr-single-tab cycle 2` のコミット。cross-review(codex): cycle 1 で ACTION_REQUIRED 2 件(`docs/reports/cross-review-triage-org-herdr-single-tab.md`、e8bfac61)
  - cycle 2(dce22057、f07959d6)は cross-review の ACTION_REQUIRED 2 件を直した。(1) 座席 id が org_id と同じとき、名前の無い pane を tab の label で受け入れない(`confirmPaneTab` の `label != workspaceLabel`)。(2) director の start を `WorkspaceCreate`、`RecordWorkspace`、`PaneRename`、`RecordPane`、`TabRename` の順にした。あわせて、`TabCreate` が tab の id の無い返事を失敗にする(self-review L6)、`--env` の無い herdr の警告に「agmsg が pane の名前も変えうるので、stop が pane を閉じずに残す」を足した(L3)、古いコメント(L1、V2、I4)と stop・disband の help の直しを入れた
  - D4 の例外(承認の digest を変えないよう、本文の D4 と AC6 は書き換えずにここに書く): 座席 id が org_id と同じ前の版の座席は、tab の label が workspace の label(org_id)でもあるので、stop と disband の確かめを通らない。AC5 の穴(L2)を塞いだ代償で、AC6 はこの座席を除いて満たす。pane は手で閉じ、`--force` で止めたと記録する。補償(`confirmStaleSeatPane`)の受け入れは変わらない。スキル、org runtime の仕様、spec の FR-B6・AC-B6・Rollout、stop・disband の help に同じ例外を書いた
  - Risks への足し(L3。本文は変えない): `--env` の無い herdr で作った pane は、agmsg が pane の名前を書き換えると label の確かめが食い違い、stop が pane を閉じずに残す。0.7.5 の herdr には `--env` があるので、この版では起きない。`/org` と `/director` の skill に書いた
  - AC11 の実機の記録は、cycle 2 の前の binary(1468232d)で取った。cycle 2 の変更は実機で打ち直していない
  - spec の非機能要件 2 は FR-B4 とぶつかるので `[ ]` のまま(verify V1、`docs/tech-debt/README.md`)。ユーザーの判断に残す
  - cycle 2 で直さなかった LOW は `docs/tech-debt/README.md` に行を足した: `TabCreate` の返事に tab の id が無いとき pane が記録なしで残る(self-review C2-1)、`placeSeatPane` の長さと `how` の文字列の突き合わせ(L4、C2-2)、コメントと help の詰め直し(C2-4、test gaps 2)、並行テストの待ちに期限が無いこと(L8 の行に test gaps 1 を追記)
  - cross-review cycle 2(codex、daf8ba9f): 所見 0 件。上限(2)に届いたが、Case C で /pr に進んだ
- [x] PR created — https://github.com/yoshpy-dev/ralph/pull/304
