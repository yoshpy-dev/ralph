# Walkthrough: seat-wip-checkpoint(座席の環境 4c: F11、座席の pane で WIP のコミットを止める)

- Date: 2026-10-22
- Plan: docs/plans/active/2026-10-22-seat-wip-checkpoint.md(/pr で archive に移す)
- PR: #250
- Branch: fix/seat-wip-checkpoint(base c90d01b4、この walkthrough の前の HEAD 9fed5cc2。19 commits、45 files、+1860/-284)
- 読む範囲: コードとテストは Go の 14 files(+541/-102)と、hook 4 本(root と `templates/base/` の写し)、新しい `tests/test-wip-checkpoint-hooks.sh`(289 行)。Go の本体は新しい `internal/org/driver/paneenv.go`(42 行)、`herdr.go`、`director_cmd.go`、`internal/cli/org.go`、`org_director_headless.go` で、`agmsg.go` と `spawn.go` は doc だけ。leader の雛形 `internal/org/prompts/leader.md`(+36/-1)とそのテスト `prompts_test.go`(+199)。文書は skill 4 つの写し 2 組、`git-commit-strategy.md` と `director-launch-cmd.md` の写し、README。残りは計画、記録、後工程の報告 5 本(cross-review は triage)、insight event
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → hook(S1) → pane の値(S2) → 実機(S3) → leader の雛形と文書(S4) → 記録(S5) → 後工程と、その中の直し → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行の所見を直す系列の 4 本目の 3 つ目(4c)。4a は #248(F3、agmsg の名前の書き換え)、4b は #249(F4、失敗した spawn が pane を閉じる)
- F11: 試行(stage 9)の片付けで、report-escape の feature worktree の HEAD が、push されていない `wip: checkpoint before session end`(e8a0d3e3)だった。中身は leader が書いた org の report 1 つ。PR #241 には入らなかったが、regate やあとの push があれば入っていた
- 作ったのは誰か: 試行の記録は「PR のあとの leader のセッション」と書いていた。manifest の時刻では、コミットの時刻 05:56:51Z(notes の値)が implementer の stop 05:56:46Z と reviewer の stop 05:56:52Z のあいだにあり、PR #241 を名指す RESULT は 05:57:43Z に上がった。`ralph org stop` で終わった座席の SessionEnd の hook の見込みが高い。どのセッションかを名指す記録はない
- 直し: ralph が作る pane と launch_cmd の起動スクリプトに `RALPH_WIP_CHECKPOINT=off` を付け、hook はその値があると `git add -A` とコミットだけを飛ばす。値が届かない配備のために、leader の雛形の手順 14 が push の前に `wip:` のコミットを探して止まる。4b の follow-up として、雛形に spawn の失敗の扱いも足した
- 戻し方: この PR を revert する。値を読まない hook に戻れば、pane の環境に値が残っていても振る舞いは今までどおり

## 2. 読む順(commit ごと)

1. a189336f 計画、3c898c69 承認
   - Objective、Scope、Non-goals、Design decisions(D1〜D6)、AC1〜AC9 を読む。承認の digest は `b98f2806ab31` で、後工程のあいだ本文は変えていない(Progress と AC の checkbox だけが動く)
   - D1: 「座席である」と hook に知らせる印ではなく、WIP のコミットだけを止める値にした。hook は org のことを知らずに済み、人も自分のセッションで同じ値を設定できる。`off` 以外(空、`on`、`OFF`)は今までどおりコミットする
   - D2: 効かせる範囲は ralph が作る pane のすべて。leader も座席の pane で動き、機能の worktree を implementer と reviewer と分け合う。headless の director はふつう main で動くので hook はもともと飛ばすが、揃えた
   - D3: pane に足す値は 1 つの一覧に置き、本番の組み立てと起動スクリプトの export の両方がそれを読む
   - D5: 現行の手順 9(`git status --porcelain`)はコミット済みのものを見ず、手順 14 は HEAD が違えば gate を打ち直すので、打ち直した gate が通れば WIP のコミットも push されていた。計画の Codex 助言の指摘 1 で足した
   - D6: 実機の確かめは ralph を通さず、分離した herdr で `herdr tab create --env` を直接打つ。ralph が 2 つの `--env` を付けることは Go のテストで固める
2. 88706897 S1 hook(AC1)
   - `.claude/hooks/session_end_summary.sh:40` と `precompact_checkpoint.sh:43`。作業ツリーが汚れている枝の中で、`[ "${RALPH_WIP_CHECKPOINT:-}" = "off" ]` なら md に `- WIP commit: skipped (RALPH_WIP_CHECKPOINT=off)` を足し、`else` に元の `git add -A`、`git commit`、失敗の記録をそのまま入れ子にした。`case` の `main|master|unknown` と、きれいな木の枝は文字を変えていない。`set -eu` のもとで `${VAR:-}` は未設定でも落ちない
   - 写しは `templates/base/.claude/hooks/` の 2 本で、バイト一致
   - `tests/test-wip-checkpoint-hooks.sh`: hook 4 本それぞれを、一時の repo(feature ブランチ、追跡しているファイルの変更と未追跡のファイル)で、値なし・`on`・`OFF`・空・`off` で打つ。一時の repo は `guard_temp_repo` が一時ルートの下にあることを確かめてから打ち、git の設定は `GIT_CONFIG_GLOBAL` と `GIT_CONFIG_NOSYSTEM` で隔てる
   - 見ること: 値の判定は完全一致の `off` だけ。大文字小文字を畳まない
3. 416993cf S2 pane の値(AC2〜AC4)
   - `internal/org/driver/paneenv.go:25` の `SeatPaneEnv()` が `AgmsgSelfNameOff` と `WIPCheckpointOff` をこの順に返す。呼ぶたびに新しい slice を返す。名前は seat だが、doc(17〜24 行)のとおり headless の director の pane と起動スクリプトにも使う
   - 本番の組み立ては 2 か所: `internal/cli/org.go:361`(spawn と start の座席の pane)と `internal/cli/org_director_headless.go:789`(herdr の headless の director)。どちらも `driver.NewHerdr(..., driver.SeatPaneEnv()...)`
   - `internal/org/director_cmd.go:327`: 起動スクリプトが一覧の各値を `NAME='value'` と `export NAME` の 2 行で書く。値は `shellQuote` を通り、check-in の行の前に置く
   - `internal/org/driver/herdr.go:320` の `warnNoPaneEnv`: `--env` のない herdr では今までどおり値を付けずに pane を作り、warning を 1 行出す。帰結は `paneenv.go:32` の `paneEnvConsequence` が値ごとに返し、PaneEnv の順に `; ` でつなぐ。助言は `. Use ...` の別の文
   - 見ること: `agmsg.go` と `spawn.go` の変更は doc の文だけ(`AgmsgSelfNameOff` は一覧の 1 つ目になった)
4. 3dedaef9 S3 実機の確かめ(AC6)
   - `docs/evidence/seat-wip-checkpoint-2026-10-22.md`。分離した herdr 0.7.5 の server(`env -i` に必要な変数だけ、XDG の 2 変数と `HERDR_SOCKET_PATH` は一時ディレクトリの下)で、一時の repo に S1 のコミットの `.claude/hooks/` と `settings.json` を写した。本物の claude(haiku)を、`--env` を 2 つ付けた pane と 1 つだけの pane で起動した
   - (a) 2 つの値は pane の shell に届く。(d) `/compact` の PreCompact は、off の pane ではコミットせず skipped の行を書き、対照の pane では 1 つコミットした。(b) `ralph org stop` と同じ C-c と `herdr pane close` で、off の pane の SessionEnd は md を書いてコミットしなかった。(c) 対照の pane では 1 つコミットした
   - 分かったこと: hook は herdr が pane に足した環境を、`ralph-dispatch.sh` を通っても受け継ぐ。D1 を見直す必要はなかった
   - ユーザーの herdr server と Orca には触れていない。claude のフォルダ信頼の項目が `~/.claude.json` に 1 つ増え、session の記録が `~/.claude/projects/` に残った(4a と同じ)
5. b5b12ae9 S4 leader の雛形と文書(AC5、AC5b、AC7)
   - `internal/org/prompts/leader.md:244` の手順 14: HEAD の比較のあと、gate を打ち直す前と push の前に `git log --format='%h %s' origin/<既定のブランチ>..HEAD` を打ち、件名が `wip:` のコミットがあれば gate も push も打たずに BLOCKED を上げる。gate の前に作られた `wip:` は gate の HEAD にも入っているので、HEAD が同じでも打つ。既存の句(`14. HEAD(` で始まること、「HEAD が違えば push せず、gate を打ち直す(手順 12)」)は残した
   - `leader.md:419` の運用規律の新しい項: spawn が終了コード 1 で `spawn failed:` を出したら、エラーの文で分ける。`seat left active` なら `--force` を付けずに stop を打ち、stop の終了コード 0 と `ralph org status` の `(active)` がないことを確かめてから打ち直す。`org: spawn step` で始まれば pane は閉じてあり、HELLO が届いていても座席は動いていないので打ち直す(期限切れなら大きい `--timeout-ms`)。そのほかのエラーと、2 回続けての失敗は BLOCKED。`rejected:` の断りや、spawn が動いている・見張りがない・承認の検査は、この項に当たらない
   - テスト: `TestRenderRolePrompt_Leader_PushRefusesWipCommits`(`internal/org/prompts_test.go:976`)が手順 14 の句と順を、`TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError`(1387)が運用規律の項を固める。後者は `org: spawn step`・`seat left active`・`retry with a larger --timeout-ms` を、実際の `Spawn` の誤りと `refuseStaleRespawn`・`spawnDeadlineHint` から引いて突き合わせる
   - 文書: `.claude/rules/ralph/git-commit-strategy.md` の Org Runtime Commits(30 行)と End-of-Session(44〜50 行)、`/org` skill の手順 6(893 行付近)、`/director` skill(225、282、289 行付近)、`docs/recipes/director-launch-cmd.md`(133、254〜267 行)、README の 258 行。写しはすべてバイト一致
6. 80cb817a、075b2160 S5 記録(AC8)
   - `docs/tech-debt/README.md`: F11 の行と、座席が残したファイル(reviewer の報告など)を SessionEnd の hook が `wip:` でコミットする行を、取り消し線と `(RESOLVED ...)` と直上のコメントで閉じた。注は効く範囲を限って書いた(新しい binary が `--env` のある herdr で作った pane で、新しい hook が動くとき)
   - `docs/evidence/org-trial-2026-10-20.md`: F11 の段落の本文は書き換えず、その下に訂正の注(Correction)を足した。075b2160 は、古い記述の引用を出どころごとに原文のまま写すよう直したもの
   - 仕様と根拠の表: 「fix/seat-spawn-close」を「#249」に直し、PR 一覧を 32 本にした。F11 が挙げる FR-4 と FR-7 はどちらも `[x]` なので、印の数は仕様 26/6、根拠の表 25/6 のまま
7. 後工程と、その中の直し
   - 9d29ee0d、1c78d731 AC の checkbox。run-verify は 9d29ee0d で rc 0(914 秒)
   - 9cf4ed46 self-review(マージ可、LOW 7)、e5836fe3 その insight event。a79b0f82 で L1〜L7 を直した: `/director` skill の折り返し(L1)、`git-commit-strategy.md` の 1 文を下の節の語に揃えて別の段落に(L2)、運用規律の項の先頭の文を「BLOCKED を上げるときは」の形にし、`/org` skill の順を雛形に揃えた(L3、固めるテストも同じコミットで直した)、試行の訂正と tech-debt の推定の強さを揃えた(L4)、org-gate の行の (o) に注(L5)、`paneenv.go` の doc(L6)、warning の区切り(L7)
   - e127f68c verify(pass、LOW 2、INFO 2)。V1・V2(記録の文のずれ)は a535cc64 で直した
   - b1623dfd test(pass、mutant 53 本中 52 本が赤)。62f92f42 で G1(hook のテストに master と detached HEAD の行。108 件から 156 件)と G2(端末の環境に値がないときに起動スクリプトを打つ subtest)を埋めた。G1 で、detached HEAD では hook が `HEAD` を branch と読んでコミットすることが分かった(前からの振る舞い)
   - a535cc64 sync-docs。V1・V2、test の報告の Addendum、detached HEAD の tech-debt の行、計画の Progress
   - 4de6952a tech-debt の行: `wip:` を機能の PR から締め出すのは leader の雛形だけで、gate と regate はコミットの件名を見ない
   - 9fed5cc2 cross-review の triage(指摘 0)

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | S1。`tests/test-wip-checkpoint-hooks.sh`(156 件)、`TestWIPCheckpointOff_IsWhatTheHooksRead`(`internal/org/driver/paneenv_test.go:40`。hook 4 本に判定の文字列があることを探す) |
| AC2 | S2。`TestSeatPaneEnv_ListsBothEntriesInOrder`(`paneenv_test.go:14`)、`TestSeatPaneEnv_ReturnsACopy`(26)、`TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt`(`internal/org/driver/herdr_test.go:321`)、`..._PaneEnvKeepsOrder`(551)、`..._PaneEnvConcurrentCallsCheckOnce`(645)、`TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv`(`internal/cli/org_runtime_env_test.go:80`)、spawn と director start の CLI のテストの stub の herdr が受ける引数 |
| AC3 | S2。`TestDirectorLaunchScript_TextQuotesEveryValue`(`internal/org/director_cmd_test.go:443`、全文)、`TestDirectorLaunchScript_ChecksInThenExecsTheAgent`(559。端末の環境を `on` にした場合と、62f92f42 の値がない場合の両方で、check-in と agent が 2 つとも `off` を見る)、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn`(1221) |
| AC4 | S2。`TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence`(`herdr_test.go:446`)、`..._PaneEnvWithoutEnvFlagWarnsOnce`(346。help に `--env` がない・失敗・時間切れ)、`..._PaneEnvNilWarnWritesToStderr`(490)、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`(`org_runtime_env_test.go:134`、spawn と director start) |
| AC5 | S4。`TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError`(`internal/org/prompts_test.go:1387`)。手順の位置を固める `TestRenderRolePrompt_Leader_FeatureOrgProcedure` は diff に入らず通る |
| AC5b | S4。`TestRenderRolePrompt_Leader_PushRefusesWipCommits`(`prompts_test.go:976`)。hook 2 本の件名が `wip: ` で始まることも見る |
| AC6 | S3。verify が (a)〜(d) を evidence と突き合わせ、hook と settings.json が S1 のあと変わっていないことを `git diff --stat` で確かめた |
| AC7 | S4。verify が写しの `cmp` と、`AGMSG_SELF_NAME` を pane の環境として書いた文書に 2 つ目の値の書き漏れがないことを見た。check-sync と check-skill-sync は rc 0 |
| AC8 | S5。verify が取り消し線と RESOLVED の注、時刻の出どころ(manifest と inbox の行)、32 本と集合の一致、#249 の `MERGED`、「fix/seat-spawn-close」が残っていないこと、印 26/6 と 25/6 を数え直した |
| AC9 | `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は 9d29ee0d で rc 0(914 秒)。そのあとの Go とテストの変更(a79b0f82、62f92f42)は、tester と implementer が go1.26 と go1.25.8 で走らせた |

テストの強さ(test の報告より):

- 新しいテストと期待を変えたテスト 23 本は、go1.26 で `-count=3` と `-race` を通った。org と cli の package 全体は go1.25.8 でも通った
- herdr・agmsg・osascript・gh などを、呼ばれたら記録して終わる stub にした PATH と偽の HOME で 23 本と hook のテストを打ち、外への呼び出しは 0 回だった
- mutant は 53 本のうち 52 本が赤になった。生き残った W16(master を守る case から外す)は、62f92f42 の master の行で赤になった。O3(起動スクリプトで `export` を書かない)は、62f92f42 の subtest が全文のテストに頼らず赤にする
- テストの diff で消えた行(コメントを除く)は、`AGMSG_SELF_NAME=off` だけの期待(tab create の引数、warning の文、起動スクリプトの全文と偽の check-in が見る値)と、改名した `TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff`(今は `..._IsSeatPaneEnv`)だった。消えた行は読んだが、置き換え先の期待と 1 行ずつ突き合わせてはいない

## 4. 確かめていないこと

- 効くのは、新しい binary と `ralph upgrade` で入る新しい hook の両方がそろってから。片方だけでは今までどおり WIP のコミットを作る。この組み合わせの振る舞いは、コードから読んだだけで打っていない
- G3: `ralph org start --plan` の leader の pane の tab create を CLI から打って `--env` を見るテストはない。座席の pane を作る `TabCreate` の呼び出しは `internal/org/spawn.go` の 1 か所なので、間接には固まっている
- G4: `/org` skill の手順 6 に足した `wip:` の確かめの文と、雛形の手順 14 の文を突き合わせるテストはない
- G5: 実機の確かめは、レビューの直しのあとに打ち直していない。hook と settings.json はそのあと変わっていない
- G6: 雛形の手順 14 と運用規律は leader への指示で、本物の leader が従うかは確かめていない。push の前の `wip:` の検査は ralph のコマンドにはない(tech-debt に行がある)
- 実機の確かめは ralph の binary を使わず、herdr を直に打った。`--env` のない herdr での実機の振る舞い、codex の座席、機能の org を最初から最後まで回すことは確かめていない
- detached HEAD では、hook は値がなければ `wip:` をコミットする(前からの振る舞い。tech-debt に行がある)
- 62f92f42 のあとに打ち直した mutant は W16 と O3 だけ。hook のテストの書き方が変わったので、W1〜W15 の結果は未確認
- 試行の F11 のコミットを作ったセッションは、時刻からの推定で、記録では名指せない
- 記録に残る「fix/seat-wip-checkpoint」を、この PR の番号に置き換えること。次の PR でする
- Linux では、手元で走らせていない(CI が見る)
