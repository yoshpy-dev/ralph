# Verify report: org-herdr-single-tab

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(AC1〜AC12、D1〜D8、Progress の S1〜S7)。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B(FR-B1〜B8、AC-B1〜B9)と、B にも当たる非機能要件。A は PR #303 で merge 済みなので見ていない
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff 920832dd...HEAD`(base main 920832dd、HEAD 97526693、53 ファイル、+5111 / -846)。spec の準拠、静的解析、文書のずれを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない(`go test` は打っていない)。テストは中身を読み、何を assert しているかを確かめた。HEAD から `go build` した binary を `/tmp` に置いて 2 つの `--help` だけを打ち、終わったら消した。herdr は打っていない。worktree には、この report、insight event、証拠のログ(gitignore の対象)のほかに何も書いていない
- Evidence: `docs/evidence/verify-2026-11-06-org-herdr-single-tab.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。中身は run-static-verify の全出力と、末尾の Probe 1〜7(計画の digest、origin/main との試しの merge、実機の確かめの build 以後の変更、ファイルの行数、2 つの `--help`、残った tab の文、L1 の doc と `withPaneEnv`)。run-verify 自身のログは `docs/evidence/verify-2026-11-06-012753.log`(同じく対象外)
- 計画の digest: `./scripts/plan-visual.sh digest` は `61ba0f08bb4c` で、`- Approved:` の行と同じ
- origin/main: fetch したあとも 920832dd のままで、`git merge-tree --write-tree --name-only origin/main HEAD` は衝突なし(rc 0)
- 実機の記録の binary: `docs/evidence/org-herdr-single-tab-2026-11-06.md` は 1468232d から build した。`git diff 1468232d HEAD --name-only` は記録・insight・計画・self-review の文書 4 つだけで、コードは HEAD と同じ

## Spec compliance

計画の AC は spec の AC-B と FR-B を束ねているので、行ごとに対応するものを書いた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 leader → implementer → reviewer の呼び出しの列(AC-B1、FR-B1〜B3) | Met | `placeSeatPane`(`internal/org/spawn.go:2520-2608`)。使えるワークスペースが無いときは `WorkspaceCreate(ctx, p.Cwd, p.OrgID, primaryCLI)`(`:2557`)、`org_workspace_created`、`TabRename(ws.TabID, p.OrgID)`(`:2570`)、spawn_step、`PaneRename(pane, seat)`(`:2604`)の順。2 席目からは `resolveWorkspace` の `WorkspaceGet`、`PaneList`、`splitParent` が選んだ起点の `PaneSplit`、`PaneRename`。`TestOrgSpawn_Layout_LeaderImplementerReviewer`(`spawn_layout_test.go:100`)が herdr の呼び出しの列そのもの(`workspace_create, tab_rename, pane_rename, agent_start` のあと `workspace_get, pane_list, pane_split, pane_rename, agent_start` を 2 回)、`--env` の extraEnv、tab と pane の rename の引数、split の起点と向き、3 つの pane が同じ 1 つの tab にあること、tab が 1 つで label が org_id であること、spawn_step の Details を固定する。`tab_create` はこの列に無い。実機の記録の 1 で、tab 1 つ(label `iso2`)、左半分 27×23、右上 27×12、右下 27×11、pane の label が座席 id であることを見た |
| AC2 reviewer が implementer より先(AC-B2、FR-B2) | Met | `splitParent`(`spawn.go:2659`)は候補が 1 つなら右、2 つ以上なら leader 以外で `spawn_started` が最後の座席の pane を下に分ける。`TestOrgSpawn_Layout_ReviewerBeforeImplementer`(`:157`)が split の列 `root right`、`p-rev down`、`p-impl down`(4 席目まで)、`tab_create` 0 回、4 つの pane が同じ tab にあることを固定する |
| AC3 3 席の org で implementer を止める(AC-B3) | Met | `TestOrgSpawn_Layout_StopImplementerOfThree`(`:186`)が C-c と `pane close` が `p-impl` だけに行き、leader と reviewer が active のままであることを固定する。実機の記録の 2 で、implementer の pane だけが閉じ、reviewer が右の列を埋めた |
| AC4 使えないワークスペースのあとの spawn(AC-B4、FR-B4) | Met | `resolveWorkspace`(`spawn.go:2623`)が、台帳が開いていると記録したワークスペースの label を `WorkspaceGet` で読み、herdr に無ければ `not found at spawn`、label が違えば `label "<label>" at spawn` で `org_workspace_closed` を書いてから次へ進む。ほかの get の失敗は spawn を `workspace_get` で失敗にし、新しいワークスペースを作らない。`TestOrgSpawn_Layout_RecordedWorkspaceNotUsable`(`:214`)が gone と relabelled の 2 通りで、closed が created より先であること、閉じたワークスペースに `pane list` も split も打たないこと、ルートの pane に入ることを、get の失敗の 1 通りで何も作らないことを固定する。実機の記録の 3 で、全席を止めたあと `org_workspace_closed - w1 not found at spawn`、`org_workspace_created - w2`、`how=root tab=w2:t1` の 3 行を見た |
| AC5 別の org や別の label の pane を閉じない(AC-B5、FR-B6) | Met(L2 の場合を除く) | `confirmPane`(`internal/org/verbs.go:953`)は、pane の label が期待する label なら tab を読まず、label が空なら `confirmPaneTab`(`:997`)で tab の label を見て、どの場合も `confirmOrgWorkspace`(`:1028`)でワークスペースの label を見る。`TestConfirmSeatPane_TheRule`(`confirm_pane_test.go:39`)の 12 行と、`TestConfirmSeatPane_StopDisbandAndOwnPaneClose`(`:121`)の 5 通り × stop・disband・自分の pane の close が、別の org のワークスペースの pane、別の座席の label の pane、org の tab の中の label の無い pane に C-c も close も送らないことを固定する。例外は self-review の L2 で、座席 id が org_id と同じ座席(`ValidateIdentifier` は断らない)では、org の tab の label が座席 id と一致するので、org の tab の中の label の無い pane が stop と disband の確かめを通る。その fixture は `confirm_pane_test.go` に無い(grep で確かめた) |
| AC6 前の版の座席を止め、解散できる(AC-B6、FR-B6) | Met | `confirmPaneTab` の `gotTab == label`(`verbs.go:1011`)。`TestConfirmSeatPane_StopDisbandAndOwnPaneClose` の「an earlier version's seat」(pane の label 無し、tab の label が座席 id)が stop・disband・自分の pane の close で閉じることを固定する。前の版の org に新しく spawn する道(候補 0 なら `tab create`)は `TestOrgSpawn_Layout_NoLiveSeatPaneMakesATab`(`spawn_layout_test.go:654`)。実機では確かめていない(記録の「確かめていないこと」に書いてある) |
| AC7 pane を作ったあと名前を付ける前に止まった spawn の補償(AC-B7、FR-B6、D4・D5) | Met | `compensateStale`(`spawn.go:2956`)は、最後の記録が spawn_step のときだけ `paneCreatedTab` で Details の tab を読み(`:2965-2969`)、`confirmStaleSeatPane` に渡す。`confirmPane` は label が空で記録した tab の中にある pane だけを tab を読まずに受け入れる。`TestOrgSpawn_Layout_InterruptedAfterPaneCreated`(`spawn_layout_test.go:324`)が AC7 の 4 通り(ルートの pane、この版の tab の中の分割、前の版の tab の中の分割、`tab create`)を止まった状態から打ち直し、閉じるのがその pane だけで、打ち直しのあとに名前の無い pane が残らないことを確かめる。`TestCompensateStale_ClosesOnlyThePaneItsSpawnMade`(`confirm_pane_test.go:299`)が、別の tab の名前の無い pane、別の座席の label、別の org のワークスペース、spawn_step がもう 1 つあとにある場合に閉じないことを固定する。最後の記録が spawn_started の座席は、Roster の PaneID が空になる(`manifest.go:222`、spawn_started は pane を持たない)ので `no pane to compensate` になる |
| AC8 headless の director(AC-B8、FR-B7) | Met | `createWorkspaceAndPane`(`internal/org/director_start.go:1183`)が `WorkspaceCreate`、`RecordWorkspace`、ルートの pane の検査、`RecordPane`、`PaneRename(pane, DirectorTabLabel)`(`:1211`)、`TabRename(tab, DirectorWorkspaceLabel)`(`:1217`)の順に打つ。`abort`(`:1284`)は label を付ける前ならワークスペースだけを閉じる。`TestStartDirector_RecordsTheRootPaneThenLabelsIt`(`director_start_root_pane_test.go:47`)が呼び出しと記録の列と `tab create` 0 回を固定し、`TestStopDirector_ConfirmsThePaneByItsLabel`(`confirm_pane_test.go:399`)がこの版の director と前の版の director を閉じ、label の無い pane と別の director の pane を閉じないことを固定する。実機の director start はしていない(記録に理由がある) |
| AC9 同時の spawn(FR-B5、D3) | Met(テストだけ) | 配置のロック `layout-<org_id>.lock`(`internal/org/lockfile.go:43`)を `placeSeatPane` が spawn の期限まで待ってとり(`spawn.go:2530`)、`defer` で放す。ロックの中で台帳を読み直す。`TestOrgSpawn_Layout_ConcurrentSpawnsShareOneTab`(`spawn_layout_test.go:450`)が、片方を `PaneSplit` の中で止めて、もう片方が先の pane を下に分けることと、新しい org の最初の 2 席でワークスペースが 1 つになることを確かめる。待ちは `time.Sleep(100ms)` で、ロックが無くても通りうる向きの弱さは self-review の L8。実機の同時 spawn は無い |
| AC10 `--env` の確かめをサブコマンドごとに(FR-B8) | Met | `paneEnvProbes`(`internal/org/driver/herdr.go:91`)が `tab create`・`workspace create`・`pane split` の 3 つに確かめを 1 つずつ持ち、`paneEnvSupported`(`:522`)が `herdr <sub> --help` を 1 回だけ読む。`warnNoPaneEnv`(`:555`)の文は base の文の `tab create` をサブコマンドの語に置き換えた形で、`tab create` の文は base と同じ。`TestHerdr_EnvCheck_WithoutEnvFlagWarnsOncePerSubcommand`(`herdr_test.go:2009`)が `workspace create` と `pane split` で `--env` なしの argv と 1 行の警告を、`TestHerdr_EnvCheck_EachSubcommandChecksItsOwnHelp`(`:2078`)がサブコマンドごとの確かめを固定する |
| AC11 実機の記録(AC-B9) | Met | `docs/evidence/org-herdr-single-tab-2026-11-06.md` に、3 席の配置(tab 1 つ、矩形、label)、implementer の停止のあとの配置、全席の停止のあとの spawn(新しいワークスペース w2)、解散がある。binary はコードの最後の commit 1468232d。分離の手順、ユーザーの herdr に触れていないこと、片付けも書いてある。`$HOME` のパスと秘密の値は無い |
| AC12 文書と help、2 つの sync | Met | `ralph org spawn` と `ralph org director start` の `--help` を HEAD の binary で打ち、pane の配置と label の段落が出ることを見た(Probe 5)。stop・disband・director stop の help、`/org` と `/director` の skill の 4 つの写し、`docs/specs/2026-08-01-org-runtime.md` の 2026-11-06 改訂、`docs/recipes/isolated-herdr-server.md`(と写し)が pane と書く。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)は run-static-verify の中で通った。計画が範囲の外に置いた 3 つの文書のずれは下の Documentation drift に書いた |

FR と非機能要件のうち、AC に入らないものも見た。

- FR-B1 の「空のタブは残らない」は AC1 のテストの `len(h.tabLabels) == 1` と実機の記録 1。FR-B5 の「起点を決めてからラベルを付け終えるまで」は、ロックが `resolveWorkspace` の前から `PaneRename` のあとまでかかることで満たす
- FR-B6 の「補償で閉じてよい」は spec より狭い。spec は「その spawn が記録した pane id で、label が空、ワークスペースの label が org_id」を許すが、実装はさらに「記録した tab の中にある」ことを求める。spec の Security considerations の「確認を緩めない」の向きなので問題にしない
- 非機能要件 1(0.7.5 で確かめた操作だけ): 新しい操作は `workspace create --env`、`pane split`、`pane rename`、`tab rename`、`pane list --workspace` で、最後の 1 つは spec の列挙に無いが、計画の Assumptions が 0.7.5 で確かめたと書き、実機の記録でも使った
- 非機能要件 2(1 回の spawn で増える呼び出し): V1 を参照。2 席目からの spawn は列挙より多い
- 非機能要件 3(herdr を呼ぶのは driver の `Herdr` だけ): `internal/org` の本番のコードに `"herdr"` を exec する箇所は無い(grep で見た 2 件はどちらも定数の値)。新しい 4 つの操作は `HerdrClient` を通る

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | exit 0、`All verifiers passed.` | `Language scope: changed`、`Language packs selected: golang`。gofmt ok、`go vet ./...` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし。local verifier の shellcheck、hook の `sh -n`、guard の awk、settings の `jq -e`、Codex hook の 3 つの guard、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照はすべて OK。branch secret scan は `920832dd..97526693` で clean |
| `./scripts/plan-visual.sh digest <plan>` | `61ba0f08bb4c` | 計画の `- Approved:` と同じ |
| `git merge-tree --write-tree --name-only origin/main HEAD` | rc 0 | origin/main は 920832dd のまま。衝突なし |
| `go build -o /tmp/... ./cmd/ralph` と 2 つの `--help` | 成功 | spawn と director start の Long に pane の文、`--timeout-ms` に `workspace create or pane split, labels` が出る |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org spawn`・`stop`・`disband`・`director start`・`director stop` の help | Yes | コードの順と規則に合う。director start が stdout に書く `talk to the director in herdr` の行の変更と、director stop と wake の確かめの失敗の文の変更を引く文書は無い(`*.md` を grep した) |
| `/org`・`/director` の skill(4 つの写し) | Yes | pane の配置、label、前の版の org、サブコマンドごとの警告の文が、コードの文(`warning: herdr <sub> has no --env flag`)と合う |
| `docs/specs/2026-08-01-org-runtime.md` の 2026-11-06 改訂 | Yes | (a) 座席、(b) director、本文は書き換えない旨 |
| `docs/recipes/isolated-herdr-server.md`(と写し) | Yes | 248 行は前の版のワークスペースでだけ `tab create` を使うと書く。409 行の `herdr tab create` は人が使い捨ての pane を作る手順で、ralph の振る舞いの説明ではない |
| `docs/recipes/director-launch-cmd.md` 263〜268 行(と写し) | No(既知、/sync-docs) | herdr の director の pane を `herdr tab create --env ...` で作り、`tab create --help` を見て、`warning: herdr tab create has no --env flag` を出すと書く。今は `workspace create` のルートの pane で、警告も `workspace create` の語になる |
| `docs/recipes/director-wake.md` 139〜141 行(と写し) | No(既知、/sync-docs) | 見張りが director の pane を「the tab labelled `director_<token>`」で確かめると書く。今は pane の label(前の版の director だけ tab の label) |
| `.claude/rules/ralph/git-commit-strategy.md` 57 行(と写し) | No(既知、/sync-docs) | 「A herdr whose `tab create` has no `--env` creates the panes without it」。今は使うサブコマンドごと(`workspace create`・`pane split`・`tab create`) |
| `docs/tech-debt/README.md` 363 行 | No(既知、/sync-docs、self-review の L9) | `tab_created` の append。今は spawn_step の `pane_created` |
| `docs/tech-debt/README.md` の発火した Trigger | No(/sync-docs) | I2 を参照。7 行の Trigger がこの diff で発火し、注も支払いも無い。178 行の (c) はこの diff の D2 で支払われた |
| `internal/org/driver/paneenv.go:28`、`:52` の doc | No | V2 を参照 |
| `internal/org/spawn.go:102-108`(`HerdrClient.WorkspaceCreate` の doc) | No | self-review の L1。`withPaneEnv`(`herdr.go:499-503`)は extraEnv が空でも PaneEnv を付ける(Probe 7) |
| `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B の印 | No(/sync-docs か /pr) | FR-B1〜B8、AC-B1〜B9、非機能要件 4 つが `[ ]` のまま。Status は「非機能要件は B が入るまで印を付けない」と書く。付けるときは V1 と AC5 の L2 を項目の下に書く |
| 計画の AC1〜AC12 と Progress の「Review artifact created」 | No(印の遅れ) | AC は `[ ]` のまま、self-review の report はあるが Progress の印が無い。/pr の前に合わせる |
| `AGENTS.md`、`README.md`、`docs/quality/` | Yes | 座席を tab と書く文は無い(grep) |
| `internal/org/spawn_test.go` の `fakeHerdr` の doc | Yes | 計画の Verify plan が挙げた「TabCreate puts the pane … in a tab of its own」は、前の版の形と今の形を分けて書き直してある |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | 非機能要件 2「1 回の spawn で増える herdr の呼び出しは、状態の確認 1 回とラベル付け 1〜2 回に収める」を、2 席目からの spawn は字面では満たさない。base の 2 席目は `tab create` の 1 回だった。今は `workspace get`、`pane list`、`pane split`、`pane rename` の 4 回で、A の `status server` を足すと 4 回増える。列挙の「状態の確認 1 回」を A の確認とすると、`workspace get` と `pane list` の 2 回が上回る。`workspace get` は FR-B4 が求め、`pane list` は D1(R2 の対策)が求めるので、spec の中で FR-B4 と非機能要件 2 がぶつかっている。計画の Verify plan は非機能要件を挙げていない。最初の座席は `workspace create`、`tab rename`、`pane rename` の 3 回(base は 2 回)で、列挙に収まる。`--help` の確かめはどちらも 1 回で変わらない | `TestOrgSpawn_Layout_LeaderImplementerReviewer` の `next` の列(`spawn_layout_test.go:107`)。base の `resolveWorkspace`(`git show 920832dd:internal/org/spawn.go` の 2467 行)は herdr に何も聞かずに台帳の id を返していた | spec の印を付けるとき、非機能要件 2 は `[ ]` のまま残して理由を項目の下に書くか、FR-B4 と D1 の呼び出しを含む形に書き直す。コードは変えなくてよい(どの呼び出しも 0.1 秒以下のソケットの往復) |
| V2 | LOW | `internal/org/driver/paneenv.go` の 2 つの doc が、座席の `RALPH_PRIMARY_CLI` を spawn が「TabCreate に extraEnv で渡す」と書く。今は pane を作る呼び出し(`WorkspaceCreate`、`PaneSplit`、`TabCreate` のどれか)に渡す。このファイルは diff に入っていない。同じ diff の `internal/cli/org.go:422-424` のコメントは直してある | `paneenv.go:26-29`(`SeatPaneEnv`)、`:50-52`(`PrimaryCLIEntry`) | 2 か所を「the call that makes the seat's pane (WorkspaceCreate, PaneSplit, or TabCreate)」に直す。L1 と同じコメントだけの直しなので、まとめて直せる |

self-review の L1〜L9 は /cross-review で扱う(指示どおり直していない)。このうち AC に当たるのは L2(AC5 の例外)と L8(AC9 のテストの弱さ)で、上の表に書いた。

### INFO

- I1(既知のずれ): 上の表の `director-launch-cmd.md`、`director-wake.md`、`git-commit-strategy.md`(それぞれ `templates/base/` の写しと同じ中身、`cmp` で確かめた)と tech-debt の 363 行。/sync-docs に回す
- I2(tech-debt の Trigger の発火): 各行の 4 列目を python で読み、この diff に当たるものを並べた。注も支払いも無いまま発火したのは次の 7 行で、/sync-docs で各行に「この PR が当たったが支払わなかった」注を足すか、支払いを記録する
  - 178 行: (a)「the next change to ... `confirmSeatPane`」が発火した。`%v` で切る形は `confirmPaneTab`(`verbs.go:1005-1008`)に移っただけで残っている。行の本文の「in `confirmSeatPane`」と (d) の「the first `isNotFound` case of `confirmSeatPane`」は、それぞれ `confirmPaneTab` と `confirmPane` を指すようになった。(c)「the next change to `resolveWorkspace` ... run `confirmOrgWorkspace` before reusing an id and create a new workspace when it fails」は D2(`resolveWorkspace` が label を確かめ、合わなければ `org_workspace_closed` を書いて作り直す)で支払われたので、RESOLVED にできる
  - 183 行: (a)「the next change to `Stop`, `Disband`, or the herdr `get` parsers」が発火した(`Stop` の doc と `PaneGet` の読み取り)。読めない台帳と JSON でない get の場合は足されていない。本文の「`org_workspace_created` (`resolveWorkspace` ...)」の append は `placeSeatPane` に移った。(b) の「CLI のテストの sh の stub は close で状態を消さない」は、この diff で `workspace create` と `pane split` の pane については消すようになった(`TestOrgHerdrStub_PaneCloseForgetsThePaneThenItsTabAndWorkspace`)ので、文が一部古い
  - 276 行: 「the next change to the herdr calls of `StartDirector`」が発火した(`TabCreate` を外し、`PaneRename` と `TabRename` を足した)。codex の director の実機は無い
  - 319 行: 「the next change to `director_start.go`」が発火した(1260 → 1307 行)。行が挙げる `internal/org/driver/herdr_test.go` も 1693 → 2388 行になった
  - 363 行: 「the next change to the failure paths of the spawn saga」が発火した(`layout_lock` から `pane_rename` までの失敗の道)。spawn_step の append が失敗したときに pane を閉じない道(`spawn.go:2597-2603`)は残る
  - 364 行: 同じ Trigger で発火した。補償のコードの移動はしていない
  - 365 行: 行数の Trigger(1 列目の数 + 300)が発火した。`spawn.go` は 2680 + 300 = 2980 を超えて 3059 行、`spawn_test.go` は 5991 + 300 = 6291 を超えて 6406 行
  - 発火していないと判断したもの: 137 行(`fakeHerdr` に足したのは `closeRemoves` で、遅延の差し込みではない。`splitEnteredHerdr` はテストの包みで `fakeHerdr` の hook ではない)、200 行の (d)(`HerdrClient` の型の doc、91〜100 行は変わっていない。91 列の行は 98 行に残る。中のメソッドの doc だけが変わった)、268 行(director stop は `confirmSeatPane` のまま)、431 行(`HerdrServerGate` を呼ぶ場所は変わっていない)
  - 登録の無い大きさ: `internal/org/driver/herdr.go` が 732 → 963 行で 800 を超えた。`internal/org/spawn_layout_test.go` は新しく 812 行。どちらも register に行が無い。self-review の L4 は `spawn.go` だけを挙げる
- I3(spec と計画の印): 上の表の 2 行。/sync-docs か /pr の前に付ける
- I4(テストのコメント): `putSeatPaneInHerdr` の doc(`internal/org/spawn_test.go:2419-2422`)は「as TabCreate and WorkspaceCreate would have left it: in a tab of its own labelled seatID」と書く。前の版の座席の形を作る helper として読めば正しいが、今の spawn はその形を作らない。直すなら「as an earlier version left it」にする

## Observational checks

- HEAD の binary の `ralph org spawn --help` は、pane の配置(左、右上、右下)、pane の label、前の版の org の扱いを 11 行で書く。`ralph org director start --help` は、ワークスペースの最初の pane、ワークスペースと tab の label `ralph_director_<token>`、pane の label `director_<token>` を書く(Probe 5)
- 実機の記録(S7)は orchestrator が分離した herdr で取ったもので、この verify では herdr を打っていない。記録の矩形の数(27×23、27×12、27×11)と split の向きと比(`right 0.5`、`down 0.5`)は AC1 の形と合う

## Coverage gaps

- Go のテストの結果と `./scripts/run-test.sh` は見ていない(/test の担当)。テストは読んで assert を確かめただけで、通るかどうかは各スライスの検証の記録(計画の Progress)と /test に任せる
- 配置のロックは偽物の herdr のテストだけで、2 つのプロセスの実機の同時 spawn は無い。L8 の「ロックを外した版で赤になるか」は /test で一度確かめると、AC9 の確かさが上がる
- 前の版のワークスペースへの spawn(D1 の候補 0 と、前の版の tab の中の分割)、補償の 4 通り、headless の director は、実機で確かめていない(記録が書くとおり)
- AC5 の L2(座席 id が org_id と同じ座席)は、コードを読んで判断した。fixture は無い
- `--env` が無い古い herdr は、偽の runner のテストだけ
- Codex の側(`.codex/agents/`)の実行は見ていない。skill の `.agents/` の写しの一致だけを `check-skill-sync.sh` で確かめた

## Verdict

pass

- Verified: AC1〜AC12 をコードとテストの assert で、AC1・AC3・AC4 を実機の記録で、AC12 を HEAD の binary の `--help` と 2 つの sync で確かめた。静的解析の全項目(gofmt、go vet、golangci-lint、staticcheck、shellcheck、sync の 4 つ、secret scan)。計画の digest は `61ba0f08bb4c` のまま。origin/main との衝突は無い
- Partially verified: AC5(seat_id == org_id の座席は L2 の例外)。AC9 と AC7 と AC8 の director の側はテストだけで実機は無い。AC6 の前の版の org への spawn もテストだけ
- Not verified: Go のテストの結果と `run-test.sh`(/test)。同時 spawn と前の版の org と director の実機。古い herdr
- 所見: LOW 2 件(V1、V2)、INFO 4 件(I1〜I4)。どれも merge を止めない
- /sync-docs に回すもの: I1 の 4 つ(写しを含む)、I2 の tech-debt の 7 行の注(178 行の (c) は RESOLVED)と 2 つのファイルの大きさ、I3 の印。V2 と self-review の L1 はコメントだけの直しで、/cross-review の triage でまとめて扱える。V1 は spec の印を付けるときに項目の下に書く
