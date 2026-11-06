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
