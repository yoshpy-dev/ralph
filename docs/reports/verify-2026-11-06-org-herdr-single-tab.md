# Verify report: org-herdr-single-tab

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(AC1〜AC12、D1〜D8)。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B(FR-B1〜B8、AC-B1〜B9)と、B にも当たる非機能要件
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2。上限 2 の 2 回目で、最後の回)
- Runs: この report は 2 回分を持つ。先頭から「Run 1」の見出しの手前までが Run 2(今回)。「Run 1」は cycle 1 の report で、コミット feb2e943 の本文を 1 バイトも変えずに末尾へ残した。cross-review の triage と tech-debt の 435 行・439 行が Run 1 の V1・V2・I1〜I4 を引くので、ID も本文も変えていない。Run 1 の `file:line` は Run 1 の時点(HEAD 97526693)のもの。Run 2 の新しい所見は V3、V4、I5 と付けた
- 判定の行: `## Verdict` は Run 2 と Run 1 に 1 つずつある。今の判定は先に出る Run 2 のほう(どちらも pass)
- Scope: cycle 2 の修正の 2 コミット。dce22057(cross-review の ACTION_REQUIRED 2 件、self-review の L1・L3・L6、verify の V2、test の所見 2 つ、tech-debt の行)と f07959d6(stop と disband の help、テストのコメント)。4c2c324b は cycle 2 の self-review の report で、所見(C2-1〜C2-5)を AC と照らすために読んだ。e8bfac61(triage)と a6fe3a25(cycle 1 の sync-docs)は cycle 1 の終わりの記録で、Run 1 の所見の状態を確かめるときだけ読んだ。静的解析は branch 全体(`git diff 920832dd...HEAD`、HEAD 4c2c324b、65 ファイル)にかかる。Go のテストと `./scripts/run-test.sh` は打っていない(/test の担当)。テストは本文を読んで assert を確かめた。HEAD から build した binary を `/tmp` に置き、3 つの `--help` だけを打った。herdr は打っていない。worktree に書いたのは、この report、insight event の 1 行、gitignore の対象の証拠のログ 2 つだけ
- Evidence: `docs/evidence/verify-2026-11-06-org-herdr-single-tab-run2.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。中身は run-static-verify の全出力と Probe 1〜9(計画の digest、origin/main との試しの merge、`git diff --check`、ファイルの行数、3 つの `--help`、実機の確かめの build 以後のコードの変更、例外を書く文書、`--env` の無い herdr の説明、この report を引くファイル)。Run 1 の report が Run 1 のログ(`docs/evidence/verify-2026-11-06-org-herdr-single-tab.log`)の中身を引くので、そのログは書き換えずに別のファイルにした。run-verify 自身のログは `docs/evidence/verify-2026-11-06-035233.log`(同じく対象外)
- 計画の digest: `61ba0f08bb4c` のままで、`- Approved:` の行と同じ。cycle 2 は計画に触れていない
- origin/main: fetch のあとも 920832dd のまま。`git merge-tree --write-tree --name-only origin/main HEAD` は rc 0 で、衝突は無い

## Run 1 の所見の状態(HEAD 4c2c324b)

| ID | Run 1 | 状態 | 根拠 |
| --- | --- | --- | --- |
| V1 | LOW | defer(ユーザーの判断に残す) | コードは変えず、tech-debt の 439 行(「Non-functional requirement 2 of ...」、dce22057)に移った。行の Trigger は「spec についてのユーザーの決定」である。triage の「直さない」一覧は、spec の非機能要件 2 の印を `[ ]` のまま残して理由を書き、PR の既知の穴にも書くと決めている。cycle 2 は `placeSeatPane` の呼び出しを変えていないので、2 席目からの spawn は今も `workspace get`・`pane list`・`pane split`・`pane rename` の 4 回である |
| V2 | LOW | 解決 | `internal/org/driver/paneenv.go:26-30`(`SeatPaneEnv`)と `:51-54`(`PrimaryCLIEntry`)の doc が「the call that makes the pane (WorkspaceCreate, PaneSplit, or TabCreate)」になった(dce22057) |
| I1 | INFO | 解決 | a6fe3a25。`docs/recipes/director-launch-cmd.md` の 263〜269 行、`docs/recipes/director-wake.md` の 139〜143 行、`.claude/rules/ralph/git-commit-strategy.md` の 57〜60 行が今のコードの形を書き、どれも `templates/base/` の写しと `cmp` で同じ。tech-debt の 363 行は `pane_created` に直った |
| I2 | INFO | 解決(数に古さが残る) | a6fe3a25 が I2 の 7 行に注を足し、178 行の (c) を RESOLVED にし、ほかに 9 行に注を足し、大きさの行(435 行)を足した。cycle 2 のあと、435 行が書く数(`herdr.go` 963 行、`spawn_layout_test.go` 812 行)は HEAD で 968 行と 904 行になった。self-review の C2-3 が /sync-docs の直しとして挙げている |
| I3 | INFO | 未(/sync-docs か /pr) | 計画の AC1〜AC12、Progress の「Review artifact created」「Verification artifact created」「Test artifact created」、spec の FR-B1〜B8・AC-B1〜B9・非機能要件 4 つが、どれも `[ ]` のまま |
| I4 | INFO | 解決 | `putSeatPaneInHerdr` の doc(`internal/org/spawn_test.go:2419-2422`)が「as an earlier version left it ... with no pane label」になった(f07959d6) |

## Spec compliance

cycle 2 のコードが通る道に当たる AC を、行ごとに見直した。cycle 2 がその道を変えていない AC は、変えていないことを確かめたうえで Run 1 の判定を引き継いだ。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1〜AC4(配置、3 席の停止、使えないワークスペース) | Met(Run 1 のまま) | cycle 2 の `internal/org/spawn.go` の変更は、`HerdrClient` の `WorkspaceCreate` と `TabCreate` の doc だけである(dce22057 の hunk は 101〜116 行)。`placeSeatPane`・`resolveWorkspace`・`splitParent` は Run 1 と同じ。stop の確かめは、pane に label がある座席では `confirmPaneTab` を通らない(`confirmPane` の `paneLabel == ""` の枝だけが呼ぶ)ので、AC3 の道も変わらない。`TestOrgSpawn_Layout_FailuresBeforeThePane` に「workspace create names a root pane and no tab」の行(`internal/org/spawn_layout_test.go:687`、`noTabHerdr`)が足され、`placeSeatPane` の `ws.TabID == ""` の判定(`spawn.go:2567`)を固定した(test report の P5) |
| AC5 別の org や別の label の pane を閉じない | Met(Run 1 の L2 の例外は無くなった) | `confirmPaneTab` の `case gotTab == label && label != workspaceLabel`(`internal/org/verbs.go:1017`)。座席 id が org_id と同じ座席では、org の tab の label が座席 id と一致しても、label の無い pane を受け入れない。誤りの文には「which is also the workspace's label」が足される(`:1021-1023`)。`TestConfirmSeatPane_TheRule`(`internal/org/confirm_pane_test.go:43`)に足された 4 行(78〜87 行)と、`TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose`(`:173`)が、stop・disband・自分の pane の close の 3 つで、label の無い pane に C-c も close も送らず、ワークスペースを開いたまま残すことを固定する。修正前の分岐(`case gotTab == label: return nil`)では「no label in the org's tab」の行が nil を返すので、このテストは修正前のコードで赤になる(コードを読んで判断した。走らせていない)。director は `DirectorTabLabel`(`director_<token>`)と `DirectorWorkspaceLabel`(`ralph_director_<token>`)がいつも違うので、この条件に当たらない |
| AC6 前の版の座席を止め、解散できる | Met(座席 id が org_id と同じ前の版の座席を除く。意図した狭め) | 前の版の座席は今も `confirmPaneTab` が tab の label で受け入れる(`TestConfirmSeatPane_StopDisbandAndOwnPaneClose` の「an earlier version's seat」。元の表は `runSeatPaneCloses` に移っただけで、行は同じ)。例外は AC5 の直しの代償で、座席 id が org_id と同じ前の版の座席は、herdr の上では「この版の org の tab の中の label の無い pane」と区別できないので断る。triage の ACTION_REQUIRED 1、tech-debt の 440 行(「Since cycle 2 of org-herdr-single-tab ...」。手で閉じて `--force` で記録する手順を書く)、stop と disband の `--help`(f07959d6、Probe 5)がこれを書く。計画の D4・AC6 と、spec の FR-B6・AC-B6・Rollout の「今までどおり」は書いていない(V3) |
| AC7 名前を付ける前に止まった spawn の補償 | Met(Run 1 のまま) | 補償の受け入れ(`confirmPane` は `createdTabID == tabID` のとき `confirmPaneTab` を呼ばない)は変わらない。座席 id が org_id と同じ座席でも、記録した tab の中の label の無い pane は受け入れ、別の tab の pane は断る(`TestConfirmSeatPane_TheRule` の 82〜87 行) |
| AC8 headless の director | Met | start の順が `WorkspaceCreate`、`RecordWorkspace`(`internal/org/director_start.go:1202`)、`PaneRename`(`:1210`)、`RecordPane`(`:1215`)、`TabRename`(`:1219`)になった。`reg.PaneID` は rename のあとに入るので、rename の前に落ちた start の登録はワークスペースだけを持つ。`abort`(`:1288`)は `paneLabelled` の旗を捨て、`reg.PaneID` が空ならワークスペースだけを閉じる(`paneLabelled` は `internal/` に残っていない)。`TestStartDirector_LabelsTheRootPaneThenRecordsIt`(`internal/org/director_start_root_pane_test.go:50`)が呼び出しと記録の列(`pane_rename` が `write pane_created` より前)と `tab create` 0 回を、`TestStartDirector_APaneRenameThatFailsClosesTheWorkspaceItMade`(`:84`)が rename の失敗で pane をどの行にも書かないことを、`TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace`(`:238`)が、ワークスペースだけを記録した登録を `director stop` と start の片付けがワークスペースの label で閉じ、pane に何も送らないことを固定する。`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade`(`internal/org/director_start_test.go:850`)の `pane_created` の行は、label の付いた pane を確かめ、C-c と close を送ってからワークスペースを閉じる形(`directorCloseCalls`)に変わった。計画の D7 の「RecordWorkspace と RecordPane は今と同じ順で書く」は今も成り立つ(間に PaneRename が入った)。stop の側(この版と前の版の director)は Run 1 のまま |
| AC9 同時の spawn | Met(テストが 1 本増えた。実機は無い) | `TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled`(`internal/org/spawn_layout_test.go:549`)が、implementer の spawn を `PaneRename` の中で止め、その間に同じ org の reviewer の spawn が配置の呼び出し(`workspace_get`、`pane_list`、`pane_split`、`tab_create`、`workspace_create`)を 1 秒のあいだ 1 つも打たないことと、放したあと implementer の pane を下に分けることを固定する。FR-B5 の「ラベルを付け終えるまで」というロックの終わりの側は、Run 1 では固定されていなかった(test report の M2・M3)。テストのコメントの「M1〜M3 に対して 10/10 赤」は、test report の 23 行と 127 行に合う |
| AC10 `--env` の確かめをサブコマンドごとに | Met | サブコマンドごとの確かめは変わらない。`paneEnvConsequence`(`internal/org/driver/paneenv.go:66`)の `AGMSG_SELF_NAME=off` の節に、pane の名前も変わりうることと、stop が pane を閉じずに残すことが足された(self-review の L3)。FR-B8 と AC10 の「今と同じ警告」は、警告の形(1 行、サブコマンドの名、`--env` なしで作る)が同じことと読んだ。新しい文は `internal/org/driver/herdr_test.go:510` の定数と `:554` の期待が固定する。この帰結を skill が書いていないのは V4 |
| AC11 実機の記録 | Met(記録は 1468232d の binary。cycle 2 のあとは打ち直していない) | `git diff 1468232d HEAD -- internal cmd` は cycle 2 の 12 ファイル(Probe 6)。実機が通った道のうち、`placeSeatPane` はコメントだけが変わり、stop の確かめは label のある pane では `confirmPaneTab` を通らない。全席の停止のあとの spawn は `workspace create` の道で、`TabCreate` の変更に当たらない。director は実機で打っていない(Run 1 のとおり)。記録が見た事実は HEAD のコードでも同じになるとコードから判断したが、打ち直してはいない |
| AC12 文書と help、2 つの sync | Met(V3・V4 の文書のずれを除く) | stop と disband の `--help` は座席 id が org_id と同じ場合を書く(Probe 5)。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(IDENTICAL 172、DRIFTED 0)は run-static-verify の中で通った。skill と org runtime の仕様の改訂が例外を書かないことは V3、`--env` の無い herdr の帰結は V4 |

非機能要件も見直した。1(0.7.5 で確かめた操作だけ)と 3(herdr を呼ぶのは driver の `Herdr` だけ)は、cycle 2 が herdr の新しい操作も exec も足していないので Run 1 のまま。2 は V1(defer)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | exit 0、`All verifiers passed.` | `Language scope: changed`、`Language packs selected: golang`。gofmt ok、`go vet ./...` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし。local verifier の shellcheck、hook の `sh -n`、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照はすべて OK。branch secret scan は `920832dd..4c2c324b` で clean |
| `./scripts/plan-visual.sh digest <plan>` | `61ba0f08bb4c` | 計画の `- Approved:` と同じ |
| `git merge-tree --write-tree --name-only origin/main HEAD` | rc 0 | origin/main は 920832dd のまま |
| `git diff --check 920832dd...HEAD -- ':!docs/evidence'` | rc 0 | 末尾の空白などは無い |
| `go build -o /tmp/... ./cmd/ralph` と 3 つの `--help` | 成功 | stop と disband の Long に「unless that tab label is also the org_id」が出る。director start の Long は Run 1 と同じ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org stop`・`disband` の `--help` | Yes | f07959d6 の「unless that tab label is also the org_id」は、コードの `label != workspaceLabel` に合う。詰め直していない短い行は self-review の C2-4 |
| `ralph org director start` の `--help` | Yes | 「workspace_created and pane_created follow as herdr hands out the ids」は、`pane_created` が rename のあとになっても、記録の順(workspace、pane、launched)としては合う |
| `/org` skill の `stop` の行(346 行、4 つの写し) | No | V3 |
| `docs/specs/2026-08-01-org-runtime.md` の 2026-11-06 改訂 (a)(56 行) | No | V3 |
| 計画の D4・AC6、spec の FR-B6・AC-B6・183 行 | No(Progress と spec の印のとき) | V3。計画の側は self-review の C2-5 と同じ |
| `/org` skill(53〜62 行)と `/director` skill(316〜320 行)の、`--env` の無い herdr の説明 | No | V4 |
| `TabCreate` の doc(`internal/org/driver/herdr.go:333-344`)と `HerdrClient` の doc(`spawn.go:101-116`) | Yes | tab の id の無い返事を失敗にすること、director の start が `TabCreate` を呼ばないことが、コード(`herdr.go:378-380`、`createWorkspaceAndPane`)に合う |
| `paneenv.go` の 2 つの doc | Yes | V2 の解決 |
| `directorStartRun`・`abort`・`StartDirector` の step 3 の doc | Yes | コードの順に合う |
| `docs/tech-debt/README.md` | 一部 No(/sync-docs) | cycle 2 の新しい行(436〜440 行)は 5 列で、中身がコードと report に合う。435 行と 364 行の数は self-review の C2-3。C2-1 と C2-2 の行はまだ無い(I5)。Trigger の sweep(各行の 4 列目を python で読み、cycle 2 が変えた関数とファイルの名と突き合わせた)で、cycle 2 で新しく発火した行は無かった。`confirmPaneTab`、`createWorkspaceAndPane`、`TabCreate`、`director_start.go` を名指す行(178、183、276、319)は、cycle 1 の sync-docs がこの PR の注を付けた行である。268 行は director stop が自前の確かめに替わるとき、287 行は `abortCmd`、193 行は `Disband` の関数を名指すので、cycle 2 の変更(`confirmPaneTab` の中、`directorStartRun.abort`、stop と disband の help)は当たらない |
| 計画の Progress と AC の印、spec の印 | No | I3 のまま |
| `AGENTS.md`、`README.md`、`docs/quality/` | Yes | cycle 2 は座席の配置の書き方を変えていない |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | Run 1 のまま(defer)。spec の非機能要件 2 と FR-B4 のぶつかり | Run 1 の表、tech-debt の 439 行 | spec の印を付けるときに非機能要件 2 の下に理由を書き、PR の既知の穴に載せる(triage の決定どおり) |
| V3 | LOW | cycle 2 の直し(ACTION_REQUIRED 1)で、座席 id が org_id と同じ前の版の座席は、stop と disband の確かめを通らなくなった。stop と disband の `--help` と tech-debt の 440 行はこれを書くが、ほかの文書は前の版の座席を tab の label で受け入れるとだけ書く。(1) `/org` skill の `stop` の行の「前の版で立てた座席は pane に label がないので、その tab の label が座席 id か」(4 つの写し)。(2) org runtime の仕様の 2026-11-06 改訂 (a) の「前の版で座席ごとの tab に立てた座席は、tab の label で確かめる」。(3) 計画の D4 と AC6、spec の FR-B6(「今までどおり受け入れる」)、AC-B6(「今までどおり閉じられる」)、Rollout の 183 行。spec の中では、この場合に FR-B6 と Security considerations の 189 行(「ラベルの無いペインを閉じてよいのは、その spawn が自分で記録した id のときだけ」)がぶつかり、コードは 189 行の側をとった | `internal/org/verbs.go:1017`。`internal/cli/org.go:1640`、`:2035`。`.claude/skills/org/SKILL.md:346`(と `.agents/`、`templates/base/` の 3 つの写し)。`docs/specs/2026-08-01-org-runtime.md:56`。`docs/specs/2026-11-06-org-herdr-autostart-panes.md:56`、`:87`、`:183`、`:189`。Probe 7 | /sync-docs で (1) と (2) に「ただし、その tab の label が org_id でもあるとき(座席 id と org_id が同じ)は受け入れない」を足す(skill は `scripts/sync-skills.sh` と template の写しも)。(3) は本文を書き換えず、計画の Progress に cycle 2 の 1 項を足し(`plan-visual.sh` の digest は `## Progress checklist` を読まないので変わらない)、spec の FR-B6 と AC-B6 に印を付けるときに例外を項目の下に書き、PR の既知の穴にも書く |
| V4 | LOW | `--env` の無い herdr の警告は、cycle 2 で「agmsg が pane の名前も変えうるので、stop がその pane を閉じずに残し、手で閉じることになる」と言うようになった(self-review の L3)。この帰結は Run 1 の時点から振る舞いとしてあった(stop の確かめが pane の label を見るため)。一方、`/org` skill の `--env` の無い herdr の説明は、帰結を `agent_not_found` と `wip:` のコミットの 2 つだけ挙げる。`/director` skill は「agmsg が pane の名前を変えることがある」と書くが、director stop がその pane を閉じずに残しうることは書かない | `internal/org/driver/paneenv.go:69-71`。`.claude/skills/org/SKILL.md:53-62`、`.claude/skills/director/SKILL.md:316-320`(どちらも 4 つの写し)。Probe 8 | /sync-docs で 2 つの skill の該当の段落に、「pane の名前が変わると、stop(director は director stop)がその pane を閉じずに残すことがある」を 1 文足す |

### INFO

- I5(self-review の cycle 2 の所見と AC): C2-1(`TabCreate` が tab の id の無い返事を失敗にするとき、herdr が作った pane が記録なしに残る)は D6 の「`pane split` と `workspace create` の失敗はペインを作っていない」の外側の道で、0.7.5 の返事は両方の id を持つので、この版では起きない。C2-1 と C2-2(`placeSeatPane` の長さと `spawn.go` の大きさ)は、self-review が tech-debt の行の文を置いたが、register にはまだ無い。C2-3 の数の直しとあわせて /sync-docs で入れる。C2-5(計画が cycle 2 を記録していない)は V3 の (3) と重なる

## Observational checks

- HEAD の binary の `ralph org stop --help` と `ralph org disband --help` は、前の版の座席の tab の label に「unless that tab label is also the org_id」を付けて出す(Probe 5)。`ralph org director start --help` の記録の順の文は Run 1 と同じ
- 実機の記録(S7、1468232d の binary)は cycle 2 のあと取り直していない。この verify でも herdr は打っていない

## Coverage gaps

- Go のテストの結果と `./scripts/run-test.sh` は見ていない(/test の担当)。cycle 2 の新しいテスト 4 本(`TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose`、`TestStartDirector_LabelsTheRootPaneThenRecordsIt`、`TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace`、`TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled`)と変わった行は、読んで assert を確かめただけ
- 座席 id が org_id と同じ座席、director の新しい順、rename の前に落ちた director の start は、偽物の herdr のテストだけで、実機は無い
- 実機の記録は cycle 2 の前の binary で取った。cycle 2 が実機の通った道を変えていないことは、コードを読んで判断した
- `TabCreate` が tab の id の無い返事を返す herdr での spawn(C2-1)は、spawn の側のテストが無い
- 2 つのプロセスの同時 spawn、前の版の org への spawn、`--env` の無い古い herdr は、Run 1 と同じく実機が無い
- Codex の側(`.codex/agents/`)の実行は見ていない。skill の写しの一致だけを `check-skill-sync.sh` で確かめた

## Verdict

pass

- Verified: cycle 2 の直しで、AC5 の Run 1 の例外(L2)が無くなったこと、AC8 の director の start の新しい順と、rename の前に落ちた start の片付け、AC9 のロックの終わりの側のテスト、AC10 の警告の新しい文を、コードとテストの assert で確かめた。AC1〜AC4、AC7、AC11 の道が cycle 2 で変わっていないこと。stop と disband の `--help` の新しい文を HEAD の binary で確かめた。静的解析の全項目(gofmt、go vet、golangci-lint、staticcheck、shellcheck、sync の 4 つ、secret scan)。計画の digest は `61ba0f08bb4c` のまま。origin/main との衝突は無い
- Partially verified: AC6(座席 id が org_id と同じ前の版の座席は、意図して断るようになった。計画と spec と skill は書いていない)。AC11(実機は cycle 2 の前の binary)。AC7・AC8・AC9 はテストだけで実機は無い
- Not verified: Go のテストの結果と `run-test.sh`(/test)。cycle 2 の振る舞いの実機。同時 spawn、前の版の org、director の実機。古い herdr
- 所見: LOW 3 件(V1 は defer で Run 1 のまま、V3、V4)、INFO 1 件(I5)。どれも merge を止めない。Run 1 の V2・I1・I2・I4 は解決、I3 は未
- /sync-docs に回すもの: V3 の (1)(2)(skill の 4 つの写しと org runtime の仕様の改訂)、V4(2 つの skill の 4 つの写し)、I5 の tech-debt の行(C2-1、C2-2)と C2-3 の数、I3 の印と計画の Progress の cycle 2 の項(V3 の (3) と C2-5)。V1 と AC6 の例外は、spec の印を付けるときに項目の下に書き、PR の既知の穴にも載せる

---

## Run 1(パイプラインの cycle 1)

以下は cycle 1 の report を、変えずに残したもの。上の「Run 1 の所見の状態」が、ここの V1・V2・I1〜I4 の今の状態を言う。ここの `## Verdict` は cycle 1 の判定で、今の判定は上の Run 2 のほうである。

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
