# Self-review report: org-herdr-single-tab

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(spec `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 2(上限 2 の 2 回目で、最後の回)
- Runs: この report は 2 回分を持つ。先頭から「Run 1」の見出しの手前までが Run 2(今回、cycle 2)。「Run 1」は cycle 1 の report で、コミット 97526693 の本文を 1 バイトも変えずに末尾へ残した。Run 1 の ID(L1〜L9)と `file:line` は Run 1 の時点(HEAD 4732f977)のもので、tech-debt の register の行が引いているので、番号も本文も変えていない。Run 2 の新しい所見は C2-1 から付ける。
- 判定の行: `Merge:` の行は Run 2 と Run 1 に 1 つずつある。今の判定は Run 2 のほう(どちらも yes)。
- 現在の状態: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5(C2-1〜C2-5)。cycle 1 の LOW 9 件は、解決 5(L1、L2、L3、L6、L9。L3 の残りは C2-5、L6 の残りは C2-1)、defer で register に行がある 3(L5、L7、L8)、未 1(L4、C2-2)。

## Run 2(パイプラインの cycle 2)

- Scope: `git diff 920832dd...HEAD` の全体(base main 920832dd、HEAD f07959d6、65 ファイル、+5836 / -889)。特に cycle 2 の修正 2 コミット。dce22057(cross-review の ACTION_REQUIRED 2 件、self-review の L1・L3・L6、verify の V2、test の所見 2 つ、register の行)と f07959d6(stop と disband の help、テストのコメント)。コミットの親子は一本道で、f07959d6 ← dce22057 ← e8bfac61(triage)。この 2 つは cycle 1 の self-review・verify・test・sync-docs・cross-review のどの範囲にも入っていない。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。

### cycle 1 の所見の状態(HEAD f07959d6)

| ID | cycle 1 | 状態 | 確かめたこと |
| --- | --- | --- | --- |
| L1 | LOW | 解決 | `HerdrClient.WorkspaceCreate` の doc(`internal/org/spawn.go:102-108`)は「固定の PaneEnv に続けて extraEnv。director の start は extraEnv を渡さないので、固定のものだけ」と書く。`withPaneEnv`(`internal/org/driver/herdr.go:504-513`、env が空でなければ付ける)と、`director_start.go` の 2 つのコメントに合う。同じ interface の `TabCreate` の「director の start は呼ばない」は、`director_start.go` の step 3 の `no TabCreate` に合う。verify の V2(`paneenv.go` の 2 つの doc)と I4(`putSeatPaneInHerdr` の doc、f07959d6)も直った |
| L2 | LOW | 解決 | `confirmPaneTab`(`internal/org/verbs.go:1003`)の `case gotTab == label && label != workspaceLabel`(`:1017`)。修正前は `case gotTab == label: return nil` で、「org の tab の中の label の無い pane」を受け入れていた。テストは `TestConfirmSeatPane_TheRule` の 4 行と、stop・disband・自分の pane の close を通す `TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose`(`confirm_pane_test.go:173`)。「no label in the org's tab」の行は修正前のコードで nil を返すので赤になる。help の 2 か所は f07959d6。代償(前の版の seat_id == org_id の座席が止められなくなる)は register の行に書いてある |
| L3 | LOW | コードは解決、plan 側は未 | `paneEnvConsequence`(`paneenv.go:66`)の AgmsgSelfNameOff の節が、pane の名前も変わることと、stop が閉じずに残すことを言う。`herdr_test.go` の定数と 1 行の期待が同じ文に変わっている。plan の Risks に 1 行足す半分は、plan が cycle 2 で触られていないので未(C2-5) |
| L4 | LOW | 未(一部だけ register) | 修正は `spawn.go` のコメントだけで、`placeSeatPane` は 89 行のまま(`spawn.go:2520-2608`)。triage の「直す」「直さない」のどちらの一覧にも L4 が無い。register の行 364 は `spawn.go` の行数の増加を書くだけで、(a) と (c) は無い(C2-2) |
| L5 | LOW | defer、register 済み | `docs/tech-debt/README.md` の「The first seat of a new org has a window between `WorkspaceCreate` …」の行。窓の説明は `spawn.go:2557-2572` の順に合い、`tab_rename=failed:` の note は `paneCreatedDetails(...) + note` に合う |
| L6 | LOW | 本体は解決、残りあり | `TabCreate` が tab id の無い返事を失敗にする(`herdr.go:378-380`)。`TestHerdr_TabCreate_TabIDFallsBackToResultTab`(`herdr_test.go:230`)に 2 行。L6 の勧めの「欠けた返事で作った pane は補償で閉じる」半分は無い(C2-1) |
| L7 | LOW | defer、register 済み | 「Since org-herdr-single-tab (D7), `DirectorTabLabel` …」の行。名前の一覧は L7 の evidence に合う |
| L8 | LOW | defer、register 済み | 「`TestOrgSpawn_Layout_ConcurrentSpawnsShareOneTab` ("after the leader" …」の行。変異の回数(70/70)は test report の変異の表に合う。新しい `TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled` の本文は、tester が worktree の外に書いた `TestGap_LayoutLockHeldUntilThePaneIsLabelled`(全文が `docs/evidence/test-2026-11-06-org-herdr-single-tab.log` にある。gitignore の対象)と、関数名のほかは同じだった(2 つを突き合わせた)ので、「M1〜M3 で 10/10 赤」はこのテストにも当たる |
| L9 | LOW | 解決 | 「The paths of `Spawn` that stop when a manifest append fails after the pane exists …」の行は、`pane_created`(「it was `tab_created` before …」)と「unlabelled pane」で書き直されている(a6fe3a25) |

### Evidence reviewed(Run 2)

- dce22057 の diff の全部(11 ファイル、+363 / -117)と f07959d6 の diff の全部(2 ファイル)を読んだ。
- `confirmSeatPane`・`confirmStaleSeatPane`・`confirmPane`・`confirmPaneTab` を HEAD の本文で読み直した(`verbs.go:895-1050`)。`confirmPaneTab` の呼び出しは `confirmPane` の 1 か所だけで、引数の `workspaceLabel` はそこから渡る。新しいコメント「director の 2 つの label は常に異なる」は、`DirectorTabLabel` が `director_<token>`、`DirectorWorkspaceLabel` が `ralph_director_<token>` なので成り立つ(`director_start.go:93-113`)。
- director の start は `createWorkspaceAndPane`、`directorStartRun` の doc、`abort` の doc、`StartDirector` の doc の step 3 と 6 の前の段落を、コードの順(WorkspaceCreate、RecordWorkspace、返事の検査、PaneRename、RecordPane、TabRename)に照らした。`paneLabelled` は、コードにも文書にも残っていない(`grep` で、残るのは Run 1 の本文だけ)。旧いテスト名 `TestStartDirector_RecordsTheRootPaneThenLabelsIt` を指すものも無い。
- 登録に WorkspaceID だけがあり PaneID が空の場合の扱いを読んだ。`confirmDirectorResources` と `closeConfirmedDirectorResources` は、記録の無い id を飛ばす(`director_stop.go:622-660`)。`director_check.go` は、state が starting の登録を herdr の agent でなく start のプロセスで見る(`checkDirectorStarter`)ので、RecordWorkspace と RecordPane の間が PaneRename 1 回ぶん長くなっても、見張りがその登録を ended と読むことはない。
- 新しいテスト 3 つを本文まで読んだ。`TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose`(共通の `runSeatPaneCloses` に元の表をそのまま移した)、director の順と「落ちたあと」の 2 つ(`TestStartDirector_LabelsTheRootPaneThenRecordsIt`、`TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace`)、`TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled`(`renameEnteredHerdr`)。
- register の行 363・364・435〜440 を読み、コードと突き合わせた。新しい行は 6 本の `|` で、列が 5 つに収まる。`file:line` の指し先を持たない。行 440 の「`--force` は pane を閉じずに止めたと記録する」は `Stop` の `p.Force`(`verbs.go:1185`、`:1203`)に合う。
- cycle 1 の report に載せた `file:line` ではなく、この report で引く `file:line` は、書く前に `sed -n` で HEAD の行を読み直した。

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | exception-handling | L6 の直しは、`tab create` の返事に pane id があって tab id が無いときを失敗にした。その時点で herdr は tab と pane を作り終えていて、tab には `--label <座席 id>` が付いている。`TabCreate` はエラーと一緒に空の `HerdrPaneRef{}` を返して pane id を捨て、`placeSeatPane` は `fail("tab_create", err, ctx.Err(), "")` で pane を持たない spawn_failed を書く。台帳にその pane の記録は無く、閉じる者がいない。L6 の勧めは「欠けた返事で作った pane は補償で閉じる」までだったが、直しは失敗にする半分だけである。結果として、`placeSeatPane` の doc の「A step that fails before the pane exists …」の列にある `tab_create` が、この入力では偽になる。register の行(「The first seat … window」)は `WorkspaceCreate` の返事の確かめだけを書く。spawn の側に `tab_create` の失敗の行が無い(`TestOrgSpawn_Layout_FailuresBeforeThePane` の 5 行は pane_list、pane_split、workspace_create の 3 種)。0.7.5 の返事は両方の id を持つ(`realTabCreateEnvelope`)ので、この版では起きない。起きても残るのは tab 1 つの名前の無いシェルで、disband が閉じる | `internal/org/driver/herdr.go:378-380`。`internal/org/spawn.go:2511-2513`(doc)、`:2591`(`fail("tab_create", …, "")`)。`internal/org/spawn_layout_test.go:673`(表に `tab_create` の行が無い) | 2 通り。(1) `TabCreate` が pane id を入れた ref をエラーと一緒に返し(Runner のエラーの経路は `PaneID: out` に stdout の生の値を入れるので、tab id の欠けだけ区別する)、tab の枝が `fail("tab_create", err, …, pane.PaneID)` で閉じる。表に 1 行足す。(2) 直さずに、register の行に「`tab create` の返事の確かめも同じ」と足し、`TabCreate` の doc と `placeSeatPane` の doc に「失敗でも herdr が pane を作っていることがある」と書く。最後の cycle なので (2) を勧める。下の Tech debt に文を置いた |
| C2-2 | LOW | maintainability | L4 が、直されず、register にも行を持たない。L4 は (a) `placeSeatPane` が 89 行で、ロック、台帳の読み直し、3 通りの pane の作り方、記録、名前付けを持つこと、(b) `spawn.go` が 3059 行になったこと、(c) `splitParent` が返す `how` の文字列と driver の方向を `splitHow == paneHowSplitDown` で比べてつなぐこと。cycle 2 の修正は `spawn.go` のコメントしか変えていないので、(a) と (c) は cycle 1 と同じである。triage は L1・L3・L6 を直す側、L5・L7・L8 を直さない側に挙げ、L4 はどちらにも無い。register の行(「`internal/org/spawn.go` is 2665 lines …」)は (b) の一部を持つが、移す対象は補償のコードと `seat-spawn-close` のテストで、新しい `placeSeatPane`・`resolveWorkspace`・`splitParent`・`openOrgWorkspaces`・`paneCreated*` のまとまりも、(a)、(c) も書いていない。最後の cycle で直さない LOW は defer なので、行が要る | `internal/org/spawn.go:2520-2608`、`:2582`。`docs/tech-debt/README.md` の行 364 の「`spawn.go` went from 2778 lines … to 3059」 | Tech debt に貼れる行を置いた |
| C2-3 | LOW | maintainability | cycle 2 の修正が足した引用の一部は、この cycle の `/test` と `/cross-review` が同じファイルを書き直すと指す先が変わる。(a) `ACTION_REQUIRED 1` と `ACTION_REQUIRED 2` を cycle の指定なしで `cross-review of plan …` として引く 3 か所(`internal/org/confirm_pane_test.go:165`、`internal/org/director_start_root_pane_test.go:48-49` と `:229-230`)と、register の行 440。triage は `Cycle: 1/2` で、cross-review は同じ名前のファイルを前の cycle の分から書き直してきた(同じ型の finding を前の PR でも出した)。(b) `In the /test report of plan …, this test failed 10 of 10 runs against each of M1 …`(`internal/org/spawn_layout_test.go:542`)と `P5 of the /test report …`(`:730`)。10/10 の元は gitignore の `.log` で、コミットされる記録は test report だけである。この report の cycle 1 の ID を引く `self-review L6 of plan …`(`internal/org/driver/herdr_test.go:226`)は、Run 1 がそのまま残るので安全。(c) register の行 364 の「`spawn_layout_test.go`, 812 lines, and `confirm_pane_test.go`, 453」と、行 435 の「812 lines」「963 lines」は d4fa6ce2 で数えた値で、cycle 2 の修正のあとは 904、509、968 行になった(`spawn_layout_test.go` と `confirm_pane_test.go` は同じ修正で増えた)。行 435 の先頭は d4fa6ce2 と明記するが、行 364 の 2 つの数には付いていない | 上の `file:line`。`wc -l` は `spawn_layout_test.go` 904、`confirm_pane_test.go` 509、`herdr.go` 968(HEAD f07959d6) | register の側(行 364・435・440)は `/sync-docs` が直せる。数には「d4fa6ce2 で」を付けるか HEAD の値に替え、行 440 の `ACTION_REQUIRED 1` は「cycle 1 の cross-review(コミット e8bfac61)の ACTION_REQUIRED 1」にする。Go のコメント 5 か所は、cross-review が回る前に「cycle 1 の cross-review(e8bfac61)」「test report の layout lock の節(cycle 1)」の形にするのがよいが、Go のファイルの変更なので記録だけの修正の例外に入らず、上限 2 を超える。lead が、コメント 5 か所のために回し直すかを決める(直さなければ defer で、引用に plan の slug があり、git の履歴で辿れる) |
| C2-4 | LOW | readability | 文を差し込んだ 3 か所で、段落を詰め直していない。`internal/org/verbs.go:927` の `// org's tab, is not. Stop` は 26 字で、前後は 68〜76 字。`ralph org stop --help` の `that fails the check (a label differs, or`(`internal/cli/org.go:1642`)と `ralph org disband --help` の `that fails the check is not closed and`(`:2037`)は 40 字台で、前後は 74〜79 字。help は利用者の画面に出る | `internal/org/verbs.go:923-928`。`internal/cli/org.go:1638-1646`、`:2033-2041` | 3 か所とも詰め直す。help の分は、次に stop・disband の help を触るときでよい |
| C2-5 | LOW | maintainability | plan が cycle 2 を何も記録していない。plan の最後のコミットは 4732f977(S1〜S7 の記録)で、cycle 2 の修正は plan に触れていない。D4 は「ペインのラベルが空で、タブのラベルが座席 id なら、前の版の座席として受け入れる」と書いたままで、seat_id == org_id の例外を持たない。AC6(「前の版で作った座席を止め、解散できる」)と、spec の FR-B6 の同じ約束も、例外を言わない。cycle 1 の L3 が勧めた Risks の 1 行(agmsg が pane の label を書き換えうる)も無い。`/pr` は plan を archive するので、例外のない D4 が残る記録になる。例外の正しさと AC6 の判定は `/verify` の担当で、ここでは記録が無いことだけを言う。D7 の「RecordWorkspace と RecordPane は今と同じ順」は、PaneRename が RecordPane の前に動いても今も真 | `docs/plans/active/2026-11-06-org-herdr-single-tab.md` の D4、AC6、Risks、Progress checklist。`git log -- docs/plans/active/2026-11-06-org-herdr-single-tab.md` の先頭が 4732f977 | `## Progress checklist` に cycle 2 の 1 項を足す。`scripts/plan-visual.sh` の `digest_body` はその節を読まないので、承認の digest は変わらない。書くのは、D4 の例外(seat_id == org_id は tab の label で受け入れない)、L3 の Risk(agmsg が pane の label を書き換えると stop が閉じない)、`TabCreate` が tab id の無い返事を失敗にすること、director の PaneRename を RecordPane の前に動かしたこと |

### Positive notes

- L2 の直しは、判定を 1 か所(`confirmPaneTab`)に置き、`label != workspaceLabel` の条件で、新しい版の座席(pane に label がある)と、補償の「記録した tab の中の pane」には触れない。テストは、元の表の本文を `runSeatPaneCloses` に移して座席 id を引数にしたので、元の 5 通りは同じ厳しさのまま、seat_id == org_id の 2 通りが同じ 3 つの動詞を通る。`wantLastEvent` も座席と org を見るようになった。
- director の直しは、`paneLabelled` を捨て、`reg.PaneID` が空であることで「まだ label が無い」を表す。持つ状態が 1 つ減り、`abort` の doc と `directorStartRun` の doc が同じ文で書き換わっている。テストは 3 層(順、台帳の行に pane が出ないこと、記録に失敗したときの pane の閉じ方)で、RecordPane が失敗する場合は「label のある pane を C-c と close してからワークスペース」まで固定する。
- 新しい `TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace` は、ワークスペースだけを記録した登録を seed して、stop と start の片付けが、pane に何も送らずにワークスペースの label で閉じることを固定する。この直しが頼る経路そのものである。
- ロックのテストは、ロックを外した版で赤になる形(layout の呼び出しの一覧が空でなければ失敗)で、`t.Fatalf` の前に release を閉じる。失敗の判定が時間に頼るのは、ロックが壊れている場合だけである。
- register の新しい行は、列の数が合い、`file:line` を持たず、`--force` の働きの説明が `Stop` のコードに合う。L9 は行そのものを `pane_created` に直す形で閉じている。
- 秘密、デバッグの残り、範囲外のファイル、U+FFFD は見つからなかった。

### Tech debt identified

最後の cycle なので、直さない LOW は defer になる。この report は `docs/tech-debt/README.md` を書き換えない(コミットは report と insight event の行だけ)。`/sync-docs` が貼れるように、行の文を置く。列は register の 5 列(Debt item、Impact、Why deferred、Trigger to pay down、Related plan/report)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `TabCreate` (`internal/org/driver/herdr.go`) fails a reply that names the pane and no tab, after herdr has made the pane and its tab, which `tab create` labelled with the seat id (self-review L6 of cycle 1 made it fail). It returns no ids with the error, so `placeSeatPane` (`internal/org/spawn.go`) fails the spawn at `tab_create` with no pane to close and no `pane_created` record, and the pane is left open on no record. The doc of `placeSeatPane` lists `tab_create` among the steps that fail before the pane exists, and no spawn test reaches a `tab_create` failure | LOW. Needs a herdr whose `tab create` reply omits the tab id; 0.7.5's carries both. One shell in a tab of its own stays in the org's workspace until a disband closes the workspace | Closing the pane needs the id returned with the error, and the Runner-error path of `TabCreate` (which puts the raw stdout in `PaneID`) told apart from it; making the reply fail was the smaller change | A report of a stray pane after a failed `tab create`, or the next change to the `tab_create` branch of `placeSeatPane`: return the pane id with the error, fail the step with it so `closeSpawnPane` closes it, and add a `tab_create` row to `TestOrgSpawn_Layout_FailuresBeforeThePane` | `docs/reports/self-review-2026-11-06-org-herdr-single-tab.md` (L6, C2-1); `internal/org/driver/herdr.go` (`TabCreate`) |
| `placeSeatPane` (`internal/org/spawn.go`) is 89 lines: it holds the layout lock, the re-read of the ledger, the three ways to make the pane (the workspace's root pane, a split, a tab), the `pane_created` record, and the rename, at a nesting depth of 4. It, `resolveWorkspace`, `splitParent`, `openOrgWorkspaces`, `paneCreatedDetails`, and `paneCreatedTab` make a group that touches no other step of `Spawn`, in a file that went from 2778 to 3059 lines with org-herdr-single-tab. `splitParent` returns the way as a string (`split-down`), and `placeSeatPane` turns it into a driver direction with `splitHow == paneHowSplitDown`, so a way added on one side is not caught by the types | LOW. Harder to read and review; no behavior is wrong | A move-only split is outside the plan's Scope (S4), and the cap of the pipeline was reached with the other LOWs of cycle 1 | The next change to `placeSeatPane`, `splitParent`, or the spawn saga: move the group into `internal/org/spawn_layout.go`, split the root case from the split-or-tab case, and have `splitParent` return the driver direction | `docs/reports/self-review-2026-11-06-org-herdr-single-tab.md` (L4, C2-2); `docs/plans/active/2026-11-06-org-herdr-single-tab.md` (S4) |

`/sync-docs` が直せる既存の行の編集(C2-3 の register の側)。

| Row | State now | Fix |
| --- | --- | --- |
| 「`internal/org/spawn.go` is 2665 lines …」(行 364) | `spawn_layout_test.go`, 812 lines, and `confirm_pane_test.go`, 453 に、数えたコミットが付かない | 「at d4fa6ce2」を付けるか、HEAD の 904 と 509 に替える |
| 「`internal/org/driver/herdr.go` grew from 732 to 963 lines …」(行 435) | 「812 lines」の `spawn_layout_test.go` と、963 の `herdr.go` は d4fa6ce2 の値で、cycle 2 の修正のあとは 904 と 968 | 同上 |
| 「Since cycle 2 of org-herdr-single-tab (cross-review ACTION_REQUIRED 1, self-review L2) …」(行 440) | `cross-review ACTION_REQUIRED 1` が cycle の指定なしで、triage は書き直される | 「the cycle 1 cross-review (commit e8bfac61), ACTION_REQUIRED 1」にする。同じ行の Related 列の `(ACTION_REQUIRED 1)` も同様 |

### Coverage gaps

- Go のテスト、`go vet`、静的解析は打っていない(`/verify` と `/test` の担当)。新しいテストが修正前のコードで赤になるかは、修正前の `confirmPaneTab` の分岐(`case gotTab == label: return nil`)と、director の旧い順を diff から読んで判断した。走らせてはいない。
- `herdr_test.go` の cycle 2 で触れていない行、`internal/cli` の偽の herdr の本文は読んでいない。CLI の 2 つの偽物の `tab create` の返事が tab id を持つことだけは `grep` で確かめた(`org_test.go:250`、`org_runtime_env_test.go:111`)。
- `/org`・`/director` の skill の 4 つの写しと recipe が、stop の確かめの「tab の label」の文に seat_id == org_id の例外を書くべきかは見ていない(文書の drift は `/verify`・`/sync-docs` の担当)。
- C2-3 の「cross-review が triage を書き直す」は、前の PR の様子と `Cycle: 1/2` の見出しからの推測で、この cycle の `/cross-review` がどうするかは未確認。
- 2 つのプロセスで同時に spawn する実機の確かめは無い(cycle 1 と同じ)。

### Recommendation

- Merge: yes。CRITICAL と HIGH と MEDIUM はない。LOW は 5 件(C2-1〜C2-5)。cycle 1 の ACTION_REQUIRED 2 件に当たる直し(L2、director の順)は、どちらも修正前のコードで赤になるテストを持ち、新しい欠陥は作っていない。
- Follow-ups:
  - 最後の cycle(2/2)なので、直さない LOW は defer になる。C2-1 と C2-2 は上の 2 行を register に貼る。C2-3 の register の側は上の編集の表で直す。C2-5 は plan の Progress に 1 項(digest は変わらない)。`/sync-docs` がこの 3 つを担当できる。
  - Go のコメントと help の側(C2-3 の 5 か所、C2-4 の 3 か所)は、コメントだけの変更でも Go のファイルなので、記録だけの修正の例外に入らない。cap が 2 のままなら回し直せないので、直さずに進めるか、cap を上げるかを lead が決める。C2-3 の引用が一番早く効くのは、cross-review が triage を書き直す前である。
  - C2-1 の (1) は `internal/org/driver/herdr.go` と `spawn.go` の変更になるので、同じ理由で上の (2)(記録だけ)を勧めた。
  - PR の本文の既知の穴に、前の版の seat_id == org_id の座席(止められず、手で閉じて `--force`)と、verify の V1(spec の中の要件のぶつかり)を載せる(triage の記録どおり)。

---

## Run 1(パイプラインの cycle 1)

以下は cycle 1 の report を、変えずに残したもの。上の Run 2 の「cycle 1 の所見の状態」が、ここの L1〜L9 の今の状態を言う。ここの `Merge:` の行は cycle 1 の判定で、今の判定は上の Run 2 のほうである。

# Self-review report: org-herdr-single-tab

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(spec `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B、S1〜S7、D1〜D8、AC1〜AC12)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 1(上限 2 の 1 回目)
- Scope: `git diff 920832dd...HEAD`(base 920832dd、HEAD 4732f977、51 ファイル、+5037 / -846)。コードは S1 c6b44e28(driver)、S2 c39a4e17(CLI の偽物)、S3 4950e51f(確かめ)、S4 9df9735c(spawn の配置)、S5 3b7d188b(director)。文書は S6 1468232d、実機の記録 S7 b4bf05bf、plan の記録 4732f977。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。
- 現在の状態: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 9(L1〜L9)。

## Evidence reviewed

- `internal/org/driver/herdr.go` の diff を全部読んだ。`WorkspaceCreate`・`TabCreate` の返り値、新しい `PaneSplit`・`PaneRename`・`TabRename`・`PaneList`、`PaneGet` の `HerdrPane`、サブコマンドごとの `--env` の確かめ(`paneEnvProbes`、`withPaneEnv`、`paneEnvSupported`、`warnNoPaneEnv`)。`PaneSplit`・`PaneList`・`PaneGet` は `herdrGetResult` を通るので、読めない応答は error で、id を推測で返す経路はない。`herdrPaneJSON` から `HerdrPane` への型変換は、2 つの struct のフィールドが一致している間だけ通る(どちらかに足すとコンパイルが落ちるので、気づける側の脆さである)。
- `internal/org/spawn.go` の diff を全部読んだ。`HerdrClient` の形、`placeSeatPane`(2520-2608 行)、`resolveWorkspace`、`splitParent`、`openOrgWorkspaces` の doc、`paneCreatedDetails` と `paneCreatedTab`、`compensateStale` の `createdTabID`、`closeSpawnPane` の doc。`dryRunSpawn` の Details が `paneCreatedEvent` になった点も読んだ。
- 配置のロックを `lockfile.go` の `acquireHeldFileLock` に照らして読んだ。待ちは非ブロッキングの flock を spawn の期限までポーリングする形で、ctx の期限より長くは待たない。期限を過ぎると `errFileLockHeld` を `layout_lock` の失敗にし、`deadlineErr` を `DeadlineExceeded` にして `failStep` に渡す。解放は `defer lock.release()` 1 か所で、成功、各 `fail(...)`、`pane_rename` の失敗(`closeSpawnPane` をロックを持ったまま打つ)のどれでも通る。プロセスが落ちれば OS が flock を放す。ロックの順は「座席の spawn ロック、次に配置のロック」で、`appendEvent` は `O_APPEND` の 1 回の write(`ManifestStore.Append`)でロックを取らないので、doc の「台帳の append はロックを取らない」は正しく、循環はない。ロックの中で台帳を読み直すので、別の座席が先に記録した pane が見える。
- `internal/org/verbs.go` の diff を全部読んだ。`confirmSeatPane`・`confirmStaleSeatPane`・`confirmPane`・`confirmPaneTab`、`labelWords.pane`、`errLabelMismatch` の doc。呼び出し元(Stop、`CloseDeferredSelfPane`、`director_check.go`、`director_stop.go`、`director_wake.go`)を `grep` で全部確かめ、補償だけが `confirmStaleSeatPane` を呼ぶことを確かめた。
- `internal/org/director_start.go`・`director_stop.go`・`director_check.go`・`director_wake.go` の diff を全部読んだ。`createWorkspaceAndPane` の順(WorkspaceCreate、RecordWorkspace、ルートペインの検査、RecordPane、PaneRename、TabRename)、`paneLabelled` と `abort` の「ラベルを付ける前はワークスペースだけ閉じる」、`closeDirectorWorkspace` が not found を閉じ済みとして扱うこと(ルートペインを閉じるとワークスペースも消えるため必要)。
- `internal/cli/org.go`・`org_director_headless.go` の help の変更と、`docs/specs/2026-08-01-org-runtime.md`・recipe・`.claude/skills/{org,director}/SKILL.md` の diff を読んだ(コードとの食い違いが目に入るかどうかだけ。文書の drift は見ていない)。
- テストは、`internal/cli/org_test.go` の `herdrStub` の diff(最初の 330 行)、`internal/cli/org_herdr_stub_test.go` の最初の 200 行、`internal/org/spawn_test.go` の `fakeHerdr` の diff(最初の 420 行)、`internal/org/spawn_layout_test.go` の関数一覧と並行 spawn のテスト(415-535 行)を読んだ。期待の書き換えは、閉じる pane と記録する出来事を保っている(`tab create` の数を見るテストは、`workspace create`・`tab rename`・`pane rename`・`workspace get`・`pane list`・`pane split` の呼び出しの列を見るテストになり、`--env` の列も同じ厳しさで確かめている)。
- `docs/evidence/org-herdr-single-tab-2026-11-06.md` を全部読んだ。`$HOME` のパスも秘密の値もなかった。
- `spawn.go` の追加行に非 ASCII の文字も U+FFFD もない(plan の記録にある python での書き換え 3 か所を含む)。
- `docs/tech-debt/README.md` を、herdr の tab・`confirmSeatPane`・`tab_created` に触れる行について照らした(178、268、329、363 行)。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | maintainability | `HerdrClient.WorkspaceCreate` の doc が、今のドライバーの挙動と逆を言う。doc は「`extraEnv` を渡すと固定の PaneEnv に続けて付ける。渡さないと entry は 1 つも付けない」と書く。S1 の間だけその形だった(plan の Progress の S1 の「ずれ」)が、S5 で「PaneEnv を常に付ける」に変えた。director の start は `extraEnv` なしで呼び、pane に `AGMSG_SELF_NAME=off` と `RALPH_WIP_CHECKPOINT=off` が入る。同じ diff の `director_start.go` の 2 つのコメントは「driver が PaneEnv を付ける」と書くので、3 か所のうち 1 か所だけが食い違う。読み違えると、director の pane から PaneEnv を外す変更が入りうる。 | `internal/org/spawn.go:102-108`(「given none, it adds no entry at all」)。`internal/org/driver/herdr.go:499-509`(`withPaneEnv` は PaneEnv があれば `extraEnv` が空でも付ける)。`internal/org/director_start.go:880`、`:1187`。 | doc を「固定の PaneEnv に続けて `extraEnv` を付ける。どちらも空なら `--env` を付けない」に直す。コメントだけの変更。 |
| L2 | LOW | security | `confirmPaneTab` は「pane の label が空で、tab の label が座席 id」なら前の版の座席として受け入れる(D4)。この版の org の tab の label は org_id である(`placeSeatPane` の `TabRename(ws.TabID, p.OrgID)`)。`ValidateIdentifier` は seat_id と org_id が同じ値でも断らない(組み合わせの長さの検査だけ)。seat_id が org_id と同じ座席(例: `--org-id review --id review`)では、org の tab の label と座席 id が一致するので、記録した id が label の無い pane(人が org の tab で開いたシェル)を指すと、stop と disband の確かめを通り、C-c と `pane close` が送られる。AC5 と D4 が「閉じない」と決めた場合に当たる。起きるには、記録した id が label の無い pane に再利用される必要がある。実機の記録では id は単調に増えた(`w1:p4`)ので、条件は狭い。失敗の向きは閉じすぎ。director の label は `_` を含み座席 id と重ならないので、director には当たらない。 | `internal/org/verbs.go:997-1018`(`case gotTab == label`)。`internal/org/spawn.go:2570`(tab の label は org_id)。`internal/org/identifier.go:38`、`internal/org/spawn.go:1323-1341`(seat_id と org_id が同じ値を断る検査はない)。 | `confirmPane` で `label == workspaceLabel` のときは tab の label による受け入れを使わない(その座席は label の食い違いとして閉じずに残る)。3 行ほど。org_id と同じ座席 id の fixture を `confirm_pane_test.go` に 1 つ足す。spawn で seat_id == org_id を断る手もあるが、既存の org の台帳に触れるので前者を勧める。 |
| L3 | LOW | maintainability | stop・disband・補償の確かめの鍵が、pane の label になった。tech-debt の 329 行は、agmsg 1.5.1 が `herdr pane rename` で pane の label を hex に書き換えることを記録している(stage 9 の F3)。ralph は `AGMSG_SELF_NAME=off` を agmsg の呼び出しと、`--env` のある herdr で作る pane に入れて抑える(#248)。`--env` が無いサブコマンド(0.7.5 より古い herdr)で作った pane は抑えられず、書き換えられた座席は label の食い違いになって、`ralph org stop` に閉じられずに残る。前の版は tab の label を見ていたので、この書き換えでは食い違わなかった。安全側の失敗で、この版が前提にする 0.7.5 では起きない。ただ `--env` が無いときの警告の節は「pane の herdr agent の名前が変わり agent_not_found になる」としか言わず、plan の R1〜R7 にもこの依存がない。 | `docs/tech-debt/README.md:329`。`internal/org/driver/paneenv.go:66-68`(`AgmsgSelfNameOff` の節)。`internal/org/verbs.go:953-976`。plan の Risks(R1〜R7)。 | 警告の節に「pane の label が書き換わると ralph org stop が閉じない」を足し、plan の Risks に 1 行足す。警告の文面は `herdr_test.go` が固定しているはずなので、直すならテストも合わせる。 |
| L4 | LOW | readability | 3 点。(a) `placeSeatPane` は本体が約 90 行(2520-2608)で、ロック、台帳の読み直し、3 通りの pane の作り方(root、split、tab)、記録、名前付けを 1 つに持ち、入れ子は最大 4 段(`else` の中の `if`/`else` の中の `if`)。(b) `spawn.go` は 2778 行から 3059 行になった。新しいコード(`placeSeatPane`、`resolveWorkspace`、`splitParent`、`openOrgWorkspaces`、`paneCreated*`)は他の spawn の手順に触れない 1 つのまとまり。(c) `splitParent` は `how` の文字列(`split-down`)を返し、`placeSeatPane` が `splitHow == paneHowSplitDown` で driver の方向の定数に写す。`how` の語彙と driver の方向が文字列の比較でつながっていて、片方に語を足しても型で気づけない。 | `internal/org/spawn.go:2520-2608`、`:2582`(`if splitHow == paneHowSplitDown`)、`:2659`(`splitParent`)。 | 動きを変えずに `spawn_layout.go` へ移し、root の作り方と split か tab の作り方を 2 つの関数に切る。`splitParent` は方向を返し、`how` はそこから決める。spawn の手順を次に触るときでよい。 |
| L5 | LOW | exception-handling | 窓の長さ。最初の座席では、`WorkspaceCreate` のあと `org_workspace_created` を書き、`TabRename` を 1 回打ってから spawn_step(`pane_created`)を書く。この間にプロセスが落ちると、ルートの pane は台帳に無く、label も無い(plan の Progress の S4 に同じ記録がある)。同じ座席の打ち直しは、spawn_started しか無いので補償の対象を持たず、ワークスペースは台帳が開いていると記録し label も合うので使われる。`PaneList` は座席の候補を 0 と答え、`tab create` で tab を足す。org の tab に label の無いシェルの pane が残り、1 タブの配置が崩れる(disband は閉じる)。窓は `TabRename` 1 回ぶん長い。`ws.RootPaneID == "" || ws.TabID == ""` の枝(応答が欠けたとき)も、開いたワークスペースだけを記録して return するので、次の spawn が同じ形になる。0.7.5 の応答には両方ある。 | `internal/org/spawn.go:2557-2572`(`WorkspaceCreate`、`org_workspace_created`、欠けの検査、`TabRename`、のちに spawn_step)。 | tab の label は見た目だけ(D6)なので、spawn_step を書いたあとに `TabRename` を打てば、窓は台帳の append 1 回になる。失敗の記録は `pane_rename` のあとの別の spawn_step か note に回す。直さないなら Tech debt に残す。plan の記録にはあるが register には無い。 |
| L6 | LOW | null-safety | `TabCreate` は応答に tab の id が無くても成功を返す(`HerdrPaneRef{TabID: ""}`)。`PaneSplit` と、`placeSeatPane` が検査する `WorkspaceCreate` は欠けると失敗にする。`placeSeatPane` は `how=tab` でも `tab=` を空のまま書き、`paneCreatedTab` は空の tab を「記録なし」と読むので、補償の受け入れ(D4 の「`tab create` の場合」)が黙って狭まる。0.7.5 の `tab create` の応答は `root_pane.tab_id` を持つのでこの版では起きない。 | `internal/org/driver/herdr.go:343-390`(`TabCreate`)。`internal/org/spawn.go:2590-2593`、`:2883`、`:2891-2906`(`tabOK` と `tabID == ""`)。 | `placeSeatPane` の tab の分岐でも `pane.TabID == ""` を失敗にする(欠けた応答で作った pane は補償で閉じる)。しないなら `TabCreate` の doc に、欠けると補償の受け入れが狭まることを書く。 |
| L7 | LOW | naming | `DirectorTabLabel`、`DirectorStartResult.TabLabel`、`DirectorCleanupError.TabLabel`、`confirmDirectorResources` と `typeDirectorWakeLine` の局所変数 `tabLabel` は、この版から pane の label を指す。tab の label は `DirectorWorkspaceLabel` になった。`DirectorTabLabel` の doc(`director_start.go:97-103`)は説明するが、識別子は逆を言う。`grep TabLabel` で pane の処理に着く。名前は公開で、テストの使用も多いので、rename は diff が広い。 | `internal/org/director_start.go:97-103`、`:792`、`:1211`。`internal/org/director_stop.go:517`、`:623`。`internal/cli/org_director_headless.go:377`。 | 次に director の start/stop を触るときに `DirectorPaneLabel` へ改める。それまで defer。 |
| L8 | LOW | maintainability | 並行 spawn のテスト(「after the leader」)は、reviewer の spawn がロックの前で待つまでを `time.Sleep(100 * time.Millisecond)` で待つ。遅い環境で reviewer がその間にロックの待ちに届かなければ、ロックが無くても通る。偽の失敗は出ず、偽の成功だけが出る向き。 | `internal/org/spawn_layout_test.go:462`。 | 配置のロックを外した版でこのテストが赤になることを、`/test` で一度確かめる。待ちを観測する印を足すのは難しいので、確かめるだけでよい。 |
| L9 | LOW | maintainability | `docs/tech-debt/README.md:363` の行は、台帳の append が失敗して pane が残る経路を `tab_created` の append と「空の tab」の言葉で説明する。この diff で記録の名前は `pane_created` になり、残る pane は label の無い root か split の pane になるので、行の語が合わない。行の中身(append の失敗で pane が残る)は今も正しい。 | `docs/tech-debt/README.md:363`。`internal/org/spawn.go:2597-2603`(spawn_step の append の失敗は、そのまま `SpawnOutcomeFailed` で返る)。 | 語を直す。register は書き換えない(この report の範囲の外)ので、sync-docs か `/pr` の前に。 |

## Positive notes

- driver の応答の読み方がそろっている。`PaneSplit`・`PaneList`・`PaneGet` は `herdrGetResult` を通り、読めない応答や、id の欠けた応答は error になる。「人が読めない応答を、確かめを通った扱いにしない」という既存の契約を新しい 3 つの呼び出しも守る。
- `--env` の確かめがサブコマンドごとの `paneEnvProbe` になり、mutex をかけたまま 1 回だけ調べる。ctx が切れた失敗は保存しない。警告の文は、どのサブコマンドかを名指しする。
- 補償だけが受け入れる「label の無い pane」は、`confirmStaleSeatPane` という別の入口にしてあり、stop・disband・director の確かめからは呼べない(呼び出し元を `grep` で確かめた)。受け入れには、最後の記録が D5 の spawn_step で、pane がその tab にあることが要り、ワークスペースの label の確かめはどの場合も残る。
- 配置のロックは、順(座席の spawn ロック、次に配置のロック)、待ちの上限(spawn の期限)、解放(`defer` 1 か所)、台帳の読み直しがそろっている。`pane_rename` の失敗で pane を閉じる間もロックを持つので、別の座席がその pane を分けようとしない。
- `splitParent` は台帳だけでなく herdr の `pane list` の答えに載る pane だけを起点にする。再利用された id が別のワークスペースの pane を指していても、そこへは分けない(R2)。
- `paneCreatedTab` は `how` の語を 4 つに絞り、空の tab を断る。書く側(`paneCreatedDetails`)と読む側が同じファイルの隣にあり、doc が語順を契約と書いている。
- director は、ラベルを付ける前の失敗ではワークスペースだけを閉じる(`paneLabelled`)。ラベルの無いルートの pane は確かめを通らないので、確かめに頼らずに済ませる設計で、コメントがその理由を言う。
- `fakeHerdr` に `closeRemoves` と、herdr の応答を模した自分用のテスト(`TestSpawnFakeHerdr_ModelsPanesOfOneTab`)がある。偽物の振る舞いが固定されているので、spawn のテストが偽物の誤りを拾いにくい。
- 秘密、デバッグの残り、範囲外のファイル、U+FFFD は見つからなかった。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L4: `placeSeatPane` が約 90 行で、新しいコードが `spawn.go`(3059 行)に入った。`how` と方向が文字列でつながる。 | 読みにくさと、次に spawn の手順を触るときの diff の大きさ。挙動は壊さない | 動きを変えない移動で、LOW 9 件のために全パイプラインを回し直すほどではない | 次に `spawn.go` の spawn の手順か `placeSeatPane` を変えるとき | 本 report の L4 |
| L5: 最初の座席で、`WorkspaceCreate` から spawn_step までの間(`TabRename` 1 回と台帳の append 1 回)にプロセスが落ちると、名前の無いルートの pane が記録なしで残り、同じ座席の打ち直しが `tab create` で tab を足す。 | org の tab に label の無いシェルが残り、1 タブの配置が崩れる。disband で閉じる。狭い窓 | 直すと spawn_step の書き方と AC7 のテストが変わる。plan の Progress が窓を既知としている | 実機で「名前の無いペインが残る」報告が出たとき、または `placeSeatPane` を次に変えるとき | 本 report の L5、plan の Progress の S4 |
| L7: `DirectorTabLabel` ほかの名前が pane の label を指す。 | 名前が逆を言う。grep が誤誘導する | 公開の名前の rename は diff が広い | 次に director の start/stop を変えるとき | 本 report の L7 |

_(register の行への足し方の案。この report は `docs/tech-debt/README.md` を書き換えない。L4・L5・L7 は、見出しを「LOW findings of org-herdr-single-tab's self-review left as they are」とする 1 行にまとめ、(a) L4、(b) L5、(c) L7 とし、各 (x) に上の Impact、Trigger、Related を書く。L9 は 363 行の `tab_created` を `pane_created` に、「empty tab」を「unlabelled pane」に直す。)_

## Coverage gaps

- Go のテスト、`go vet`、静的解析は打っていない(`/verify` と `/test` の担当)。import が残っているか、コンパイルが通るかは、各スライスの検証が通ったという plan の記録を信頼した。
- テストは全部は読んでいない。`internal/org/driver/herdr_test.go`(+729 行)、`confirm_pane_test.go`(453 行)、`director_start_root_pane_test.go`(220 行)、`spawn_layout_test.go` の 1〜414 行と 535 行以降、director・watch・verbs の書き換えたテストは、関数名の一覧と、一部の本体だけを見た。L2 の fixture の有無(seat_id == org_id)は `confirm_pane_test.go` を `grep` していないので、足りるかは未確認。
- 配置のロックの動きは、コードと `acquireHeldFileLock` を読んで判断した。2 つのプロセスで同時に spawn した実機の確かめは無い(evidence も「テストだけで確かめた」と書く)。
- L3 は、agmsg が `AGMSG_SELF_NAME=off` の無い pane で pane の label を書き換えることを、tech-debt の 329 行の記録から判断した。この review では agmsg を動かしていない。
- skill の 4 つの写しは `.claude/skills/` の diff だけ読んだ。写しの一致は `check-skill-sync.sh`(`/verify`)の担当。

## Recommendation

- Merge: yes。CRITICAL と HIGH と MEDIUM はない。
- Follow-ups:
  - L1(コメント)、L2(`confirmPane` の 3 行とテスト 1 つ)、L3(警告の文と plan の Risks)は小さく、まとめて 1 回の修正で済む。cycle 1 なので、直せば `/self-review` から全パイプラインをもう一度回す(上限 2 の 2 回目)。L2 は stop・disband が閉じる対象に関わるので、3 つのうち直すなら L2 を優先する。
  - 直さずに進むなら、L1〜L3 も defer になる。その場合は L4・L5・L7 の register の行(上の案)に L1〜L3 を足す。
  - L9 は register の語だけなので、sync-docs か `/pr` の前に直せる。記録だけの修正の例外には入らない(register は task の pipeline の記録ではない)。
  - L5 は plan の Progress が既知の窓として書いているが、register には無い。`/pr` までに register に残すか、`TabRename` の位置を替えるかを決める。
