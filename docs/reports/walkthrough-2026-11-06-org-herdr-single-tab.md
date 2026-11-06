# Walkthrough: org-herdr-single-tab(org の座席を 1 タブのペインとして並べる)

- Plan: `docs/plans/archive/2026-11-06-org-herdr-single-tab.md`
- Spec: `docs/specs/2026-11-06-org-herdr-autostart-panes.md`(この PR は B。A の herdr の自動起動は PR #303 で merge 済み)
- Base: main `920832dd`。この walkthrough の前のコミット `daf8ba9f` までで 66 ファイル、+6,366 / −930(report、evidence、insights、計画を除くと 58 ファイル、+5,162 / −930)。差分の半分以上はテスト(`spawn_layout_test.go`、`herdr_test.go`、`confirm_pane_test.go`、`spawn_test.go` など)

読む順は、driver(1)、止める前の確かめ(2)、spawn の配置(3)、headless の director(4)、cycle 2 の直し(5)、文書(6)です。括弧の中はコミットです。

## 1. driver: ペインの操作(S1 `c6b44e28`)

- `internal/org/driver/herdr.go` に `PaneSplit`(`pane split <pane> --direction right|down --ratio 0.5 --cwd --env --no-focus`)、`PaneRename`、`TabRename`、`PaneList`(`pane list --workspace`)を足した。
- `WorkspaceCreate` は `HerdrWorkspace{WorkspaceID, TabID, RootPaneID}` を、`TabCreate` と `PaneSplit` は `HerdrPaneRef{PaneID, TabID}` を返す形に変えた。herdr 0.7.5 の `workspace create` は必ずタブ「1」とルートペインを作るので、その 2 つの id を返して最初の座席に使う。
- `PaneGet` と `PaneList` は `HerdrPane{PaneID, TabID, WorkspaceID, Label}` を返す。ラベルの無いペイン(herdr が null を返すか項目を省く)は `""`。
- `--env` を持たない herdr への警告(`internal/org/driver/paneenv.go`)は、サブコマンドごとに確かめる。
- CLI のテストの偽の herdr(`internal/cli/org_herdr_stub_test.go`、S2 `c39a4e17`)は、新しい操作に答える。

## 2. 止める前の確かめをペインのラベルで行う(S3 `4950e51f`、cycle 2 `dce22057`)

herdr はペインの id を振り直すので、stop・disband・自分のペインの close は、閉じる前に記録した id のペインが本当にその座席のものかを確かめる。これまではタブのラベルで確かめていた。1 タブに座席が並ぶと、タブのラベルでは座席を区別できない。

- `internal/org/verbs.go` の `confirmSeatPane` は、ワークスペースのラベルが org_id で、ペインのラベルが座席 id であることを確かめる(計画 D4)。
- 前の版が作った座席はペインにラベルが無く、タブのラベルが座席 id なので、「ペインのラベルが無く、タブのラベルが座席 id」のときも受け入れる(`confirmPaneTab`)。ただし cycle 2 で、タブのラベルがワークスペースのラベル(org_id)と同じときはこの受け入れを使わない形にした(5 を参照)。
- spawn が止まったあとの補償(`compensateStale`)だけが使う `confirmStaleSeatPane` は、ラベルの無いペインでも、作ったときに記録したタブの中にあれば受け入れる。ペインを作ってから名前を付ける前に止まった spawn の後始末のためである。
- spawn_step の Details に `pane_created how=<root|split-right|split-down|tab> tab=<id>` を書く(`paneCreatedDetails`、計画 D5)。補償はここからタブの id を読む(`paneCreatedTab`)。

## 3. spawn が座席を 1 タブのペインとして作る(S4 `9df9735c`)

`internal/org/spawn.go` の `placeSeatPane` が本体。

- 配置のロック: org ごとの `layout-<org_id>.lock`(`layoutLockName`、`internal/org/lockfile.go`)を、ワークスペースを確かめるところから新しいペインにラベルを付け終えるまで持つ(計画 D3)。同じ org の 2 つの spawn が同じペインを分割したり、ワークスペースを 2 つ作ったりしない。座席のロックのあとに取る。
- ワークスペースを使う前に確かめる(`resolveWorkspace`、計画 D2): 記録したワークスペースが herdr に無いか、ラベルが org_id でなければ、`org_workspace_closed`(`not found at spawn` または `label "<x>" at spawn`)を記録して新しく作る。全席を止めると herdr がワークスペースごと消すので、次の spawn はここを通る。
- 最初の座席: `WorkspaceCreate` が作ったルートペインをそのまま使い、タブに org_id のラベルを付ける。空のタブ「1」は残らない。
- 2 席目から(`splitParent`、計画 D1): 台帳の有効な座席と作りかけの座席のペインのうち、`pane list` にあるものを候補にする。候補が 1 つなら右に分割、2 つ以上なら最後に作った leader 以外の座席を下に分割、0 なら新しいタブを作る。spawn の順に並ぶので、leader、implementer、reviewer の順に作ると、左半分、右上、右下になる。役割で位置を固定しないことは仕様の Design decisions で決めてある。
- ペインを作ったら spawn_step を書き、そのあと `pane rename` で座席 id のラベルを付ける。名前付けの失敗は `pane_rename` の段の失敗として記録する。

## 4. headless の director をルートペインに置く(S5 `3b7d188b`、cycle 2 `dce22057`)

- `internal/org/director_start.go`: `ralph org director start` も `WorkspaceCreate` のルートペインを使い、空のタブを残さない。ペインのラベルは `director_<token>`(`DirectorTabLabel`)、ワークスペースとタブのラベルは `ralph_director_<token>`(`DirectorWorkspaceLabel`)(計画 D7)。
- `director_stop.go`、`director_check.go`、`director_wake.go` の確かめは、2 と同じくペインのラベルで行い、前の版が作った director(ラベルはタブにある)も受け入れる。
- `ralph org director start` の出力は `talk to the director in herdr: pane <id>, labelled <label>, in the workspace labelled <label>` に変わった(`internal/cli/org_director_headless.go`)。

## 5. cycle 2 の直し(`dce22057`、`f07959d6`)

cycle 1 の cross-review(codex)の ACTION_REQUIRED 2 件と、同じ cycle で直すと決めた LOW を直した。

- ACTION_REQUIRED 1: 座席 id と org_id が同じとき(`--org-id review --id review`)、新しい版のタブのラベルも座席 id と同じになる。herdr が id を振り直して名前の無いシェルに当たると、前の版の座席として受け入れて閉じてしまう。`confirmPaneTab` は、タブのラベルがワークスペースのラベルと同じときタブのラベルで受け入れない。代わりに、座席 id と org_id が同じ前の版の座席は確かめを通らなくなる(手で閉じて `--force`)。`TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose` が固定する。
- ACTION_REQUIRED 2: director の start が `RecordPane` のあと `PaneRename` の前に死ぬと、名前の無いペインが登録に残り、片付けも stop も確かめで断るので立て直せない。start の順を `WorkspaceCreate`、`RecordWorkspace`、`PaneRename`、`RecordPane`、`TabRename` にした。rename の前に死ねば登録はワークスペースだけになり、片付けがワークスペースごと閉じる。`TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace` が固定する。
- あわせて: `TabCreate` が応答にタブの id が無いとき失敗にする(self-review L6)、`--env` の無い herdr の警告に「agmsg がペインの名前を変えると、stop がペインを閉じずに残す」を足す(L3)、ラベルを付け終えるまで配置のロックを持つことを固定するテスト(`TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled`)、古いコメントと stop・disband の help の直し。

## 6. 文書(S6 `1468232d`、sync-docs `a6fe3a25`・`b9a0c24a`)

- `/org` と `/director` の skill(と `.agents/`・`templates/base/` の写し): 座席は 1 タブのペインとして並ぶこと、止める前の確かめはペインのラベルで行うこと、座席 id と org_id が同じ前の版の座席の例外、`--env` の無い herdr の注意。
- `docs/specs/2026-08-01-org-runtime.md` の 2026-11-06 改訂、`docs/recipes/isolated-herdr-server.md`・`director-launch-cmd.md`・`director-wake.md`、`.claude/rules/ralph/git-commit-strategy.md`(と写し)。
- spec の B の印: FR-B1〜B8 と AC-B1〜B9 は `[x]`。非機能要件 2(1 回の spawn で増える herdr の呼び出し)は FR-B4 とぶつかるので `[ ]` のまま、理由を書いた。
- `docs/tech-debt/README.md`: 最初の座席の `tab rename` の窓(L5)、`DirectorTabLabel` の名前(L7)、並行テストの待ち(L8)、非機能要件 2(V1)、座席 id と org_id が同じ前の版の座席、`TabCreate` の返事に tab の id が無いとき(C2-1)、`placeSeatPane` の長さと `how` の文字列(C2-2)、コメントと help の詰め直し(C2-4)。
- 実機の記録: `docs/evidence/org-herdr-single-tab-2026-11-06.md`(分離した herdr で、3 席の配置、implementer の stop、全席を止めたあとの spawn、解散)。cycle 2 の前の binary で取った。
