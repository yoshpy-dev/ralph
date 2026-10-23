# Walkthrough: busy-leader(試行の所見 F7、座席を待つ leader に `ralph org send` が届くようにする)

- Date: 2026-10-23
- Plan: docs/plans/active/2026-10-23-busy-leader.md(/pr で archive に移す)
- PR: #253
- Branch: fix/busy-leader(base a58f7c64、この walkthrough の前の HEAD d05c1477。22 commits、35 files、+2323/-211)
- Base: a58f7c64(origin/main、「Merge pull request #252」)
- 読む範囲: コードは Go の 4 files(+81/-6)で、`internal/org/prompts.go`(+33)、`internal/org/verbs.go`(+23/-1)、`internal/org/spawn.go`(+11/-4)、`internal/cli/org.go`(+14/-1)。leader の雛形 `internal/org/prompts/leader.md`(+63/-17)。テストは 8 files(+891/-13)で、新しい `internal/org/prompts_leader_inbox_test.go`(+381)、`director_skill_sync_test.go`(+172/-6)、`org_skill_send_sync_test.go`(+123)、`verbs_test.go`(+92)が大半。文書は `/org` と `/director` の skill の 4 写しずつ(8 files、+352/-144)、README、`docs/quality/quality-gates.md` の 2 写し、仕様、根拠の表、tech-debt、実機の記録(+200)。残りは計画、後工程の報告 5 本(cross-review は triage)、insight event
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → leader の雛形と置き場所(S1) → Send のエラー(S2) → 実機(S3) → skill(S4) → 記録(S5) → 後工程と、その中の直し → 設計の判断 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行の所見を直す系列の 7 本目(F7)。6 本目は #252(F6・F13)
- F7: 試行(stage 9)で、report-escape の leader が implementer の RESULT を待つあいだ、agmsg の `inbox.sh` を 9 秒おきに 60 回呼ぶループを前面で走らせた。herdr はそのあいだ leader を working と報告し続け、`Send`(`internal/org/verbs.go`)の idle/done の待ちが 4 回切れ(04:09Z〜04:16Z)、director の DECISION は最初の試みの 49 分後に届いた
- 方式(D1、案 A): `Send` の待ちは変えず、leader が手番を終えて待つようにする。手番を終えた pane は herdr が `done` と報告するので、待ちが通る
- 直し 1(雛形): Claude Code の leader は、agmsg の受信箱を Monitor で見張って手番を終える。gate と `ralph org wait` は `run_in_background` で走らせる。Codex の leader には Monitor がないので、座席を待つあいだ send が届かないことを雛形に書いた(D6。直してはいない)
- 直し 2(置き場所): 雛形の `inbox.sh` と `history.sh` のパスに、置き換え文字 `{{AGMSG_SCRIPTS}}` を足した。値は ralph が解いた agmsg の置き場所の `scripts` の絶対パスを単一引用符で囲んだもの。`[org].agmsg_home` か `RALPH_ORG_AGMSG_HOME` で別の agmsg を使う org の leader が、別の受信箱を見張らないようにする(D4)
- 直し 3(エラー): `Send` の idle/done の待ちが `--timeout-ms` の期限で切れたときだけ、エラーに理由と打ち直し方を足した(`sendWaitDeadlineHint`)。動きは変えていない。`ralph org send --help` にも同じことを書いた
- 直し 4(skill): `/director` skill の「答える順」と「計画の承認」に、長い send は `--timeout-ms 1500000` で Bash の `run_in_background` から打つこと、同じ leader への send は 1 本ずつにすること、終了コード 0 のあとも `ralph org read` で DECISION が入力欄に残っていないことを確かめてから resolve することを書いた(D8、D12)。`/org` skill の `send` の行、「受信箱」節、`approve` の行、「機能の計画の承認」の 4・5、typed protocol 節の leader の段落も合わせた。4 写しずつ
- 記録: 仕様の FR-5 を `[x]` にした(`[x]` 29、`[ ]` 3。base は 28 と 4)。NFR ツール非依存は `[ ]` のままで、Codex の leader の欠けを注に足した。tech-debt の F7 の行は、取り消し線で閉じず、Codex の leader に限る文に書き換えた(D10)。#252 で約束した self-review F7(a) の行を足し、記録に残る `fix/watch-escalation-inbox` を `#252` に直し、PR の一覧を 35 本にした
- 実機(分離した herdr 0.7.5 の server、本物の claude haiku の leader): (a)〜(f) と disband がすべて通った。Monitor が動いたまま手番を終えた pane を herdr は `done` と報告し、既定の 30 秒の send は 1 秒で届いた。座席の名前で agmsg から送った文は 9.7 秒で Monitor の知らせになり、leader の手番を人の入力なしに始めた。前面で 90 秒かかるコマンドの leader への 5 秒の send は、新しいエラーの文で終了コード 1 になり、何も打たれなかった。leader の claude を `kill -KILL` で止めると、Monitor のループは 8 秒で終わった
- 戻し方: この PR を revert する。leader の雛形は binary に入っているので、効くのは新しい binary で spawn した leader から。skill の送り方は `ralph upgrade` で入り、古い binary でも使える

## 2. 読む順(commit ごと)

1. 7881cae6 計画
   - Objective、Scope、Non-goals、Design decisions(D1〜D12)、AC1〜AC7 を読む。承認の digest(計画の `- Approved:` の行)は、verify と sync-docs が `./scripts/plan-visual.sh digest` で取り直した値と同じ。承認のあと、本文は変えていない(Progress と AC の checkbox だけが動く)
   - Codex の計画助言(HIGH 2 件)は D11(ループの寿命)と D12(DECISION が届いたかの確かめ)に入った。図は `.harness/state/plan-visual/` の下にあり、コミットしていない
   - 見ること: Non-goals は、案 B(`Send` が待ち行列に入れ、見張りが leader の idle のときに打つ)、案 C(working の pane に待たずに打つ)、案 D(DECISION を agmsg で送る)、Codex の leader に send が届くようにすること、leader への ALERT の送信が時間切れになる件、implementer と reviewer の雛形、`ralph org send` の既定の `--timeout-ms`、agmsg 自身の配達の設定
2. 24e99eb1、10e284af、02e27384 S1 leader の雛形(AC1)
   - `internal/org/prompts/leader.md:406` の `## agmsg の受信箱の運用` を書き直した。前置き(`:408-413`)は、DECISION と STOP は `ralph org send` が pane に打つこと、`Send` は手番が終わって herdr が `idle` か `done` と報告するのを既定で 30 秒まで待ってから打つので、手番の途中には届かないこと、前面で長く待たず手番を終えて待つことを書く。「能動的に確認」「適当な間隔」の文は、テストの禁止語の一覧を除いて repo 全体で 0 件になった
   - 項目は 5 つ。Monitor(`:415`)、`inbox.sh --quiet` の既読と `history.sh`(`:432`)、`run_in_background`(`:438`)、Codex(`:443`)、`ralph org wait`(`:449`)
   - Monitor のコマンド(`:421`)は 1 行。`p=$PPID` を読み、`kill -0 "$p"` が通り、1500 秒の期限の前のあいだ、`inbox.sh {{TEAM}} leader --quiet` を 10 秒ごとに打って、空行を落とす `grep` を通す。ループが終わると終わりの 1 行を出す。`timeout_ms` には 1800000 を渡すので、ループは自分の期限(25 分)で Monitor の期限(30 分)より先に終わり、leader の claude がいなくなれば 10 秒ほどで終わる(D11)
   - 10e284af は consult の助言で、見張りを重ねない 1 文(2 つの見張りが既読を取り合う)と、agmsg の `watch.sh` を張らない 1 文、`history.sh` の件数 50 を足し、zsh でもコマンドを走らせるテストを足した。02e27384 は、S3 の (a) で haiku が Monitor の必須の `description` を付けずに 1 回断られたので、`description` を渡す文を足した
   - ほかの「待つ」が節を指す: 使命の 4(`:26-28`)、機能ごとの org の手順 3(`:133`)・7(`:180-181`)・12(`:227-237`)、件を上げたあと(`:388-390`)、運用規律の 2 項(`:474`、`:482-483`)。手順 12 の gate は、Claude Code では `run_in_background` で走らせて終わりの知らせで読み、Codex では前面で打ち、そのあいだ send は届かないと書く
   - テスト(新しい `internal/org/prompts_leader_inbox_test.go`): `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn`(`:74`、節の文を逐語で固め、古い文がないことを見る)、`TestRenderRolePrompt_Leader_WaitsPointAtTheAgmsgInboxSection`(`:140`、ほかの待ちが節を指し、`ralph org wait` は節と使命の 4 にしか出ない)、`TestRenderRolePrompt_Leader_MonitorCommandRuns`(`:258`、描いたコマンドを bash と `zsh -f` で、偽の `inbox.sh`・`date`・`sleep` で走らせる)、`TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone`(`:322`、親の sh が目印のファイルを待ったあとに終わると、ループが期限の前に終わる。終わらなければ 30 秒でプロセスグループごと止めて落ちる)。前からある `TestRenderRolePrompt_Leader_FeatureOrgPlanApproval`(`internal/org/prompts_test.go:581`)、`_FeatureOrgGate`(`:765`)、`_NamesWhichInbox`(`:1346`)は、雛形の書き換えに合わせて期待を 5 つ直した(手順 12 の gate の待ちは Claude Code と Codex の 2 文になり、Codex の前面の timeout の文は残る)
   - 見ること: `$PPID` が leader の claude を指すこと(実機の (e) が見た)、取りこぼし(D3)の扱いが `history.sh` の読み直しだけであること、Codex の項に「区切れば届く」と書いていないこと
3. 24e99eb1、1183b08e `{{AGMSG_SCRIPTS}}` の配線(AC2)
   - `internal/org/prompts.go:62-71` の `RolePromptVars.AgmsgHome`、`:79` の `agmsgScriptsWord`(空なら `driver.ResolveAgmsgHome("")`、`scripts` を足して絶対パスにし、`shellQuote` で 1 語にする)。`RenderRolePrompt` は、雛形に置き換え文字があるときだけ値を作る(`:117-121`。1183b08e の直し)。replacer の対は `:134`
   - `internal/org/spawn.go:189` の `Org.AgmsgHome`。`Spawn`(`:1014`)と `dryRunSpawn`(`:2270`)が role prompt に渡す
   - `internal/cli/org.go:353` が `driver.ResolveAgmsgHome(orgCfg.AgmsgHome)` を 1 回だけ解き、`Agmsg` の `Home` と `AgmsgHome`(`:366`)の両方に入れる。env の `RALPH_ORG_AGMSG_HOME` が `[org].agmsg_home` に勝つ
   - テスト: `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`(`prompts_leader_inbox_test.go:197`、6 件。空白と単一引用符、末尾の `/`、相対、環境変数、空)、`TestOrgSpawn_LeaderRole_AgmsgHomeQuotedIntoPromptFile`(`internal/org/spawn_test.go:3315`、2 件)、`TestNewOrgRuntimeAt_AgmsgHomeIsTheAgmsgDriversHome`(`internal/cli/org_runtime_env_test.go:66`、2 件)
   - 見ること: `{{AGMSG_SCRIPTS}}` を持つ雛形は `leader.md` だけ(implementer・reviewer・director に `agmsg` の語がない)。`Org{` を作る所は 4 つあり、配線があるのは `internal/cli/org.go` の 1 つで、残る 3 つは spawn しない
4. f0961a8b、3d1946d6 S2 `Send` のエラー(AC3)
   - `internal/org/verbs.go:394-402`: idle/done の待ちが失敗したとき、`errors.Is(ctx.Err(), context.DeadlineExceeded)` のときだけ `sendWaitDeadlineHint(timeoutMS)`(`:517`)を、元のエラーの `%w` のあとに足す。書き出し `org: send: wait for seat %q idle/done:` は変えない。期限はエラーの文でなく ctx の状態で見る。herdr 自身の `--timeout` は ctx のあとに切れるので、どちらも拾える。`Send` の doc(`:318-323`)にも書いた
   - hint の文は、手番が `--timeout-ms` の期限(ms の値つき)のうちに終わらなかったこと、何も打っていないこと、前面で待つ座席には手番が終わるまで届かないこと、あとで打ち直すか `--timeout-ms` を長くして打ち直すこと。3d1946d6 は英語の言い回しの直し
   - `ralph org send --help`(`internal/cli/org.go:1131-1138`)に同じ段落がある。もう 1 文、submit の確かめは座席の手番が working か blocked になったことしか見ないので、座席が自分で始めた手番(Monitor に起こされた leader)も確かめられたと読むこと、届いたかは `ralph org read` で見ることを書く
   - テスト: `TestOrgSend_IdleDoneWaitTimesOut_SaysWhyAndHowToRetry`(`internal/org/verbs_test.go:1249`、2 件。driver が ctx の期限を包む形と、herdr 自身の `--timeout` が切れる形。文字列全体、`Progress`、`PaneID`、呼び出し、event の数を見る)、`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`(`:1181`。期限の前に失敗した待ちの文は前と同じで、全文一致の 1 本を足した)、`TestOrgSend_HelpSaysItWaitsForTheTurnToEnd`(`internal/cli/org_send_help_test.go:15`。1d088f87。verify の穴を埋めた)
   - 見ること: 予算の検査、承認の再検査、submit の確かめの文は変えていない。時間切れでない失敗(herdr の失敗、座席がない)の文も同じ
5. 4f9babac S3 実機の確かめ(AC5)
   - `docs/evidence/busy-leader-2026-10-23.md`(200 行)。環境は macOS、herdr 0.7.5 のヘッドレス server を `env -i` と XDG の 2 変数と `HERDR_SOCKET_PATH` で分離、Claude Code 2.1.291 の haiku、agmsg 1.5.1、osascript は呼び出しを記録するだけの shim。ralph は 3d1946d6(S1 と S2)を worktree の外で build した
   - (a) leader が自分で Monitor を張って手番を終え、herdr は 00:22:25 から `done`(`interactive_ready` は true)。(b) 既定の 30 秒の send は 1 秒で終了コード 0、DECISION は送られた入力として pane に出て、入力欄は空。(c) 座席の名前で agmsg の `send.sh` から送った文が、9.7 秒後に Monitor の知らせとして届き、leader の手番を始めた
   - (d) 前面で 90 秒かかるコマンドを走らせる leader に `--timeout-ms 5000` の send を打つと、5 秒後に終了コード 1、stderr は新しいエラーの 1 行で、pane にも台帳にも何も打たれなかった(Claude Code は前面の `sleep 90` を自分で断るので、perl の 90 秒の待ちを使った)。(e) leader の claude を `kill -KILL` で止めると、ループは 8 秒で終わり、`inbox.sh` のプロセスも残らなかった。disband のあとの `ps` にも残らない
   - (f) 重なり: `--enter-delay-ms` を長くした send の待ちのあいだに Monitor が leader を起こした。2 回とも DECISION は失われなかった。1 回目は入力欄に残り、Monitor の手番が終わってから Enter が入った。2 回目は手番の途中に Enter が入り、Claude Code が待ち行列に入れて(入力欄に `Press up to edit queued messages`)手番のあとに扱った。2 回目の send は終了コード 0 で終わり、stderr に注意は出なかった。submit の確かめが見た `working` は、Monitor の知らせで始まった手番のものだった。確かめは、DECISION が入力欄に残った場合ともこの待ち行列の場合とも見分けられないので、D12 の `ralph org read` が要る
   - 見ること: 「確かめていないもの」と「触れていないもの」の節。ユーザーの herdr server と `~/.config/herdr` に触れていないこと、フォルダ信頼の登録と session の記録が残ったこと。記録に pane・session の id と token の値はない
6. 460e341c S4 skill(AC4)
   - `/director` skill(`.claude/skills/director/SKILL.md`)の「答える順」(`:688`)の 2: 送る前の待ち、Claude Code は `--timeout-ms 1500000` と `run_in_background`(`:711-713`)、Codex は `--timeout-ms 300000` のように区切る、同じ leader への send は 1 本ずつ、stderr が `the seat's turn did not end within the --timeout-ms deadline` を含むときは resolve せず打ち直す、終了コード 0 でも `ralph org read` で DECISION が入力として出ていて入力欄に残っていないことを確かめる(`Press up to edit queued messages` が出ていれば待ち行列なので送れたとしてよい。DECISION が残っていれば Enter を 1 回だけ送る。`:735`)。「計画の承認」の 3(`:768-772`)と 5(`:776`)、止めない約束(`:966-967`)、冒頭の 2 か所(`:666-670`)も合わせた
   - `/org` skill(`.claude/skills/org/SKILL.md`)の `send` の行(`:187`)、`approve` の行(`:198`)、「受信箱」節の答える順(`:338-343`)、機能の計画の承認の 4・5(`:867-880`)、経路 B の手順 4(`:1285`)、typed protocol 節の leader の段落(`:1319-1329`。Claude Code の leader には届き、Codex の leader には座席を待つあいだ届かない)
   - 写しは `.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` の 3 組で、`/org` と `/director` の 4 写しずつがバイト一致(verify が `cmp` で見た)。`check-skill-sync.sh` は 14 skill
   - テスト: `TestDirectorSkill_PinsTheAnswerOrderSendAndPaneCheck`(`internal/org/director_skill_sync_test.go:750`。4 写しの断片を順に探し、skill の時間切れの句が `sendWaitDeadlineHint` の文に入っていることも見る)、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`(`internal/org/org_skill_send_sync_test.go:102`)。補助は `directorSkillSubsection`(`:197`、旧 `directorSkillWaitSubsection` を見出しを引数に取る形にした)、`skillOrderedFragmentProblems`(`:687`)、`directorSkillNumberedStep`(`:708`、`TestDirectorSkillNumberedStep` が `:780`)
   - 見ること: skill の英語の断片と `sendWaitDeadlineHint` が 1 つの定数(`directorSkillSendWaitTimeoutPhrase`、`:642`)で結ばれていること。460e341c の件名は `docs:` だが、`ralph org send --help`(`internal/cli/org.go`)とテストを含む(self-review F3。下の「確かめていないこと」)
7. 2df13dfd、d4922368 S5 記録(AC6)、e5910d90 sync-docs
   - 仕様(`docs/specs/2026-10-07-org-multi-org-director.md`): FR-5 を `[x]`(`:84`)にし、印(2026-10-23)の注(`:92`)に、直した内容、実機の (a)〜(c) と (f) の結果、雛形が 3d1946d6 のもの(02e27384 の前)であること、Codex の leader の欠けを tech-debt の行に結ぶことを書いた。NFR ツール非依存の注(`:151`)に Codex の leader の欠けを足した。PR の一覧は 35 本(`:5`)で、#252 を数え(`:25`)、この PR は数えない(`:26`)。Status の節(`:28`)は `[ ]` が 3(FR-3、FR-10、ツール非依存)
   - 根拠の表(`docs/evidence/org-spec-evidence-2026-10-21.md`): F7(26 行)、FR-5(51 行)、NFR ツール非依存(59 行)、NFR 入れ方(62 行)、判定の数(`[x]` 28、`[ ]` 3。79 行)、段ごとの PR(105 行)、確かめていないこと(117 行)。FR-12 は表に入らないので、仕様の 29 と表の 28 は食い違わない
   - tech-debt(`docs/tech-debt/README.md`): F7 の RESOLVED の注(`:326`。直したこと、実機の数値、テスト 6 つと、直していないこと: Codex・`guarded`・opus・sonnet の leader、25 分の期限での終わりと張り直し、gate と `ralph org wait` の `run_in_background`、既読の取りこぼし、02e27384 のあとの雛形の実機)と、Codex の leader に限る行(`:327`)。F7(a) の行(`:345`)は、`sendInboxItemToHuman` と `inboxHumanTextFor` が文字列を 5 つ取り、`alertID` と `reason` を取り違えてもコンパイルが通ること、呼び出しは 4 か所であることを書く。800 行の目安の行(`:352`)は、この PR で伸びた 5 ファイルの行数を `wc -l` の値で書く。#252 の行(`:324`、`:334`、`:342`〜`:344`)の `fix/watch-escalation-inbox` は `#252` に直した
   - e5910d90(sync-docs)が直したのは、README(`:254`、`:262`)と `docs/quality/quality-gates.md:81`・`templates/base/docs/quality/quality-gates.md:80` の「送れたあと(exited 0)に resolve」という書き方。skill と合わせて、終了コード 0 のあとに `ralph org read` で入力欄を確かめることを足した。ほかに、根拠の表 117 行の古いテストのファイル名、実機の記録の 1 文(雛形に `description` の文が入ったのは実機を打ったあと)、tech-debt の注のテスト名 1 つ
   - 見ること: `fix/watch-escalation-inbox` が `docs/specs`・`docs/evidence`・`docs/tech-debt` から 0 件になっている(base は 3・7・7 件)。この PR のブランチ名 `fix/busy-leader` が残る所は、仕様の 26・92・151 行、根拠の表の 16・26・51・59・105・117 行、tech-debt の 326・352 行で、番号は次の PR が書く
8. 後工程と、その中の直し
   - 81f30f43 self-review(初回は LOW 7、再レビューで F8 が加わり 8)。1183b08e(S6)で直した: F1(親がいなくなったらループが終わることを走らせるテスト)、F2(置き換えの値を、雛形に置き換え文字があるときだけ作る)、F4(テストの doc の折り返し)、F5(`directorSkillRepoRoot` と `directorSkillSubsection` の doc に `/org` skill のテストも使うと書いた)、F7(新しい leader の受信箱のテストを `prompts_leader_inbox_test.go` に移し、`prompts_test.go` の増えた分をなくした。800 行の目安を超える 5 ファイルの行は tech-debt `:352`)。F6 は取り下げ(`scripts/archive-plan.sh` が /pr で tech-debt の `active/` の計画の参照を `archive/` に書き換える)。36725559(S7)は F8(動かしたテストのファイル名を 5 か所の記録で直した)と、親の sh が固定の 1 秒でなく目印のファイルを待つ形への直し。最終は Merge: OK(CRITICAL・HIGH・MEDIUM 0、LOW 1 件が未了)
   - 6478fe25 verify(pass。LOW 2、INFO 2)。LOW-1(README と quality-gates の答える順)と LOW-2(根拠の表の古いファイル名)と INFO-1 は e5910d90 で直した。INFO-2(仕様 126 行の 8a-1 の記録と、`ralph org approve` の出力の「once that send succeeded」)は残した。help の段落を固めるテストの欠けは 1d088f87 が埋めた
   - 89ce30ea test(pass、mutant 28 本中 26 本が赤、穴 1)。cc03ff0e が AC7 に印を付けた
   - e5910d90 sync-docs(上の 7)。d05c1477 cross-review の triage(cycle 1/2、指摘 0)。`codex exec review --base main` を read-only で打ち、指摘なしで終わった。codex は read-only の sandbox が Go の build ディレクトリを書けないので、Go のテストは走らせていない
   - insight event は `docs/insights/events/2026-10-23-busy-leader.jsonl` の 5 行(self_review、verify、test、sync_docs、cross_review。どれも cycle 1)

## 3. 設計の判断(計画の D1〜D12)

- D1 方式: 案 A(leader が手番を終えて待つ)。案 B は見張りに打つ役目と待ち行列が増え、案 C は手番の途中の pane に打つので承認の画面に Enter が入る危険が戻り、案 D は agmsg の FROM が認証されないので DECISION の権威が崩れる
- D2 見張りの形: `inbox.sh --quiet` のループを Monitor で見張る。`ralph org wait --seat` は座席の herdr の状態しか見ず、見張りの ALERT や座席の QUESTION を拾えない。agmsg の `watch.sh` は team を leader だけに絞れるかが分からない。空行は `grep` で落とし、知らせの数を減らす
- D3 取りこぼし: `inbox.sh --quiet` は表示した時点で既読にするので、Monitor が表示の直後に止まると、知らせに載らないまま既読になりうる。雛形に、張り直す前と、届くはずの RESULT が来ないときは `history.sh` で読み直すと書く。既読にしない読み方は agmsg の側の仕事で、ralph は持たない
- D4 置き場所: 雛形に `{{AGMSG_SCRIPTS}}` を足し、`Org` に欄を 1 つ足す。固定の `~/.agents/skills/agmsg` にすると、別の agmsg を使う org の leader が別の受信箱を見張る
- D5 長いコマンド: Claude Code の leader は gate と `ralph org wait` を `run_in_background` で走らせる。Codex の leader は前面で走らせ、そのあいだは届かない
- D6 Codex の leader: 届くようにはしない。雛形に待つあいだは届かないと書き、区切れば届くとは書かない(区切った wait のあいだの手番も working なので)。返事だけを待つときは手番を終える
- D7 エラーの文: 時間切れは ctx の期限で見分け、そのときだけ文を足す。ほかの失敗の文は変えない
- D8 director の送り方: Claude Code の director は `--timeout-ms 1500000` の send を `run_in_background` で走らせる(Bash の前面の打ち切りは最長 10 分)。終わりの 1 回だけ知りたいので Monitor でなく `run_in_background`。同じ leader への send は 1 本ずつ(`Send` には座席ごとのロックがない)。Codex の director は `--timeout-ms 300000` のように区切って打ち直す
- D9 FR-5 の印: (a)〜(c) の実機の記録がそろったときだけ `[x]` にする。注で Codex の leader の欠けを NFR ツール非依存に結ぶ
- D10 tech-debt の F7 の行: 取り消し線で閉じず、Codex の leader に限る文に書き換えて残す。Claude Code の直しは上の RESOLVED の注に書く
- D11 ループの寿命(Codex の計画助言 1): Monitor のループは、自分の期限(1500 秒)と親の生死(`kill -0 $PPID`)で終わる。Monitor の `timeout_ms` だけに任せると、leader が落ちたあと古いループが `inbox.sh --quiet` を打ち続け、立て直した leader のメッセージを先に既読にしうる
- D12 DECISION が届いたかの確かめ(Codex の計画助言 2): `Send` の submit の確かめは、Enter のあとに座席が working になったかしか見ない。Monitor の知らせで leader の手番が別に始まると、DECISION が入力欄に残っていても送れたと読む。director は終了コード 0 のあと `ralph org read` で pane を読んでから resolve する。`Send` に pane の確かめを足す案は、herdr の pane の読みを Claude Code と Codex の画面の形に結びつけるので採らなかった

## 4. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | S1。`TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn`(`internal/org/prompts_leader_inbox_test.go:74`)、`_WaitsPointAtTheAgmsgInboxSection`(`:140`)、`_MonitorCommandRuns`(`:258`、bash と zsh の 2 件)、`_MonitorCommandEndsWhenItsParentIsGone`(`:322`、2 件)。test の報告の mutant L1〜L5(`kill -0` を消す、`sleep 1`、grep の空白行、`p=$$`、`2>&1` を消す)と P4 が赤。L1 と L4 は文字列の比較のテストを外した集合で打っても、親の確かめを走らせるテストだけで赤になる。self-review が bash 3.2 と `zsh -f` で、親が先に終わる形を自分で走らせ、約 11 秒で終わって残らないことを確かめた |
| AC2 | S1。`TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`(`:197`、6 件)、`TestOrgSpawn_LeaderRole_AgmsgHomeQuotedIntoPromptFile`(`internal/org/spawn_test.go:3315`)、`TestNewOrgRuntimeAt_AgmsgHomeIsTheAgmsgDriversHome`(`internal/cli/org_runtime_env_test.go:66`)。mutant P1〜P5、S1、C1、C2 が赤。S2(`dryRunSpawn` が置き場所を渡さない)は生き残り、同値と判断した。dry run は role prompt のファイルを書かず、描いた文は `needsPromptFile` の判定にしか使わず、leader の雛形は置き場所によらず改行を含む |
| AC3 | S2。`TestOrgSend_IdleDoneWaitTimesOut_SaysWhyAndHowToRetry`(`internal/org/verbs_test.go:1249`)、`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`(`:1181`)、`TestOrgSend_HelpSaysItWaitsForTheTurnToEnd`(`internal/cli/org_send_help_test.go:15`)。mutant V1・V2・V3・V5・K8・C3 が赤。V4(hint の ms を既定を入れる前の `p.TimeoutMS` にする)は生き残った。CLI の既定は 30000 なので、`--timeout-ms 0` か負の値を明示したときだけ届き、文が「0 ms」になるだけで動きは変わらない。テストにするには既定の 30 秒を待つか継ぎ目が要るので、PR の Known gaps に書く |
| AC4 | S4。`TestDirectorSkill_PinsTheAnswerOrderSendAndPaneCheck`(`internal/org/director_skill_sync_test.go:750`)、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`(`internal/org/org_skill_send_sync_test.go:102`)。mutant K1〜K7 は 4 写しのうち 1 つだけを変え、変えた写しの名前を出して赤になった。verify が 4 写しの `cmp` と `check-skill-sync.sh`(14 skill)を見た |
| AC5 | S3。`docs/evidence/busy-leader-2026-10-23.md` の (a)〜(f) と disband。verify が内容を読んで確かめ、実機は打ち直していない。数値(9.7 秒、8 秒など)は、生の出力を消したあとの記録どおり |
| AC6 | S5 と e5910d90。verify が、仕様の印(`[x]` 29、`[ ]` 3)、根拠の表(28 と 3)、PR の一覧(仕様と根拠の表の両方で 35 本、重複なし)を数えた。動いた印は FR-5 だけ。#252 が merge 済みであることは、base の a58f7c64 が「Merge pull request #252」であることで確かめた(gh は打っていない)。#252 の計画と 4 つの報告のファイルはある。tech-debt の 3 行(F7 の行、F7(a)、ファイルの大きさ)はコードと `wc -l` に合う(verify の Documentation drift の表)。記録が挙げるテストの名前 26 個は、どれも `func <名前>(` で 1 件だけ見つかった |
| AC7 | `./scripts/run-verify.sh`(`GOFLAGS=-timeout=30m`)は d4922368 のコードで rc 0(887 秒、「All verifiers passed」。plan の Progress は、同じコードの次の commit で self-review の報告だけの 81f30f43 の名で書く)。そのあとのコードの変更は S6・S7・1d088f87。HEAD の側は、`./scripts/run-static-verify.sh` が rc 0(verify が 429d7546 で 27 秒、plan の記録は 89ce30ea で 19 秒)、`GOFLAGS=-timeout=30m ./scripts/run-test.sh` が 1d088f87 で rc 0(844 秒、shell 44 本と Go 8 パッケージ)、`GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -count=1 -timeout 30m` が rc 0(375 秒)。1d088f87 のあとに変わったのは、Go と shell を含まない文書と報告だけ(`git diff 1d088f87 HEAD --name-only` で確かめた) |

テストの強さ(test の報告より):

- Monitor のコマンドの 2 本は、go1.26 の 5 回と、org パッケージ全体の `-coverprofile` の実行と並べた 20 回、go1.25.8 の 5 回で一度も落ちなかった。最も遅い 1.59 秒は、テスト自身の期限 30 秒の 5.3%
- 呼ばれたら記録する canary を置いた PATH と空の HOME で、leader の雛形、置き場所、`Send` の hint、skill、help の新しいテストの集合と spawn の 1 本を打ち、canary の呼び出しは 0 回だった。全部のあと、`ps` に `ralph-org-a leader` のループは残っていない
- mutant は 28 本中 26 本が赤。生き残りは同値の S2 と、穴 1 の V4。mutant は `go test -overlay` で scratchpad のファイルに差し替えて打ち、作業ツリーの追跡ファイルは変えていない
- `internal/org` の statement coverage は 95.0%(前回の報告は 94.9%)。差分で足した行と変えた行の block はすべて 1 回以上通った。`internal/cli` の coverage は計っていない

## 5. 確かめていないこと

- 実機は haiku の leader の 1 回で、3d1946d6 の binary で打った。opus と sonnet の leader、`guarded` の leader、Codex の leader は動かしていない。雛形が haiku に効いたことは、opus の leader に効くことを意味しない(試行の leader は opus だった)
- 02e27384 で雛形に足した `description` の文は、実機で打ち直していない。haiku が Monitor の必須の `description` を付けずに 1 回断られ、自分で付け直した(記録の (a))ことへの対応で、テストが文を固めているだけ
- Monitor のループの 25 分の期限での終わり、終わりの 1 行のあとの張り直し、張り直す前の `history.sh` での読み直しは見ていない(実機の leader は 1 つの見張りを約 14 分使い続けた)。`ralph org wait` と gate を `run_in_background` で走らせる形も、座席を spawn していないので見ていない。`inbox.sh --quiet` が表示した直後に Monitor が止まって取りこぼす場合(D3)も見ていない
- 本物の Monitor は実機の 1 回だけで、単体テストは偽の `inbox.sh`・`date`・`sleep` で走らせる。Monitor のコマンドを走らせたのは macOS の bash 3.2 と zsh(`-f`)で、GNU の grep や Linux の dash では走らせていない。`--line-buffered` は両方の grep にある
- director の LLM が skill のとおりに動くか(長い send を `run_in_background` で打つ、同じ leader に 2 本重ねない、pane を読んでから resolve する)は見ていない。skill はテストで文を固め、4 写しをバイトで比べただけ。Codex 側の skill の写し(`.agents/skills/` と `templates/base/.agents/skills/`)は、Codex の session で読ませていない
- Codex の leader には、座席を待つあいだ send が届かない(D6)。tech-debt の F7 の行(`:327`)が残る。将来の道として、`Send` が agmsg に渡して見張りが idle のときに打つ形(案 B)と、agmsg の Codex の monitor(app-server bridge)を挙げた。雛形が Codex の leader にそう書くことは、文のテストで固めただけで、挙動のテストはない
- `Send` には座席ごとのロックがない。同じ leader への send が重なると文が入力欄で混ざりうる。skill が 1 本ずつと書くだけで、コードは変えていない(範囲の外)
- test の穴 1(V4)は、テストにせず PR の Known gaps に書く
- AC7 は、`./scripts/run-verify.sh` を HEAD で 1 本のコマンドとしては打っていない。上の表のとおり、d4922368 の run-verify と、HEAD での run-static-verify と run-test を足した形。HEAD の秘密の走査は verify が `a58f7c64..429d7546` で clean にした。そのあとの commit は文書と報告だけ
- release の issue に書くこと(self-review F3、LOW、未了)。`.goreleaser.yml` の changelog は `^docs:`・`^test:`・`^chore:` を除くので、`docs:` の 460e341c(`ralph org send --help` の変更と、新しい skill の送り方)は自動の changelog に出ない。計画の「Rollout」の release のノートの文(Claude Code の leader は Monitor で見張って手番を終えるので send が届く、Codex の leader は待つあいだ届かない、待ちが切れたときのエラーの文、新しい skill の送り方は古い binary でも使える)を、/pr の本文に書き、release の issue にも 1 行足す。repo に CHANGELOG はない
- 記録に残る `fix/busy-leader` を、この PR の番号に置き換えること。次の PR でする。置き換え先は `grep -rn 'fix/busy-leader' docs/specs docs/evidence docs/tech-debt` で見つかる行(上の 7 の見ること)。置き換えると PR の一覧は 36 本になる
- 実機が残したもの: `~/.claude.json` にフォルダ信頼の項目が 1 つ増え、claude の session の記録、plugin の記録、`~/.claude/sessions/` の下の pid の記録が残った。手では消していない
- 手元は macOS(Darwin 25.5.0、arm64)だけで、Linux では走らせていない(CI が見る)
