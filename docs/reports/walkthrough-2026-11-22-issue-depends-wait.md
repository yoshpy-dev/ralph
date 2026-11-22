# Walkthrough: issue-depends-wait

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-issue-depends-wait.md
- Issue: #364
- Base: main 2a069b50

差分は 19 ファイル、+2,953 行、−106 行。Go は 2 ファイルで、どちらもテスト(新しい `skill_issue_depends_test.go` が 790 行、既存の `skill_split_questions_test.go` が +20 −13 行)。ralph の本番コードと `--help` は変えていない。skill の文だけの変更なので、新しい binary は要らず、`ralph upgrade` で配られる。`/director` skill は 1 写しあたり +260 −23 行で、4 写し(`.claude`、`.agents`、`templates/base` の 2 つ)は同じ中身。`/org` skill は 1 写しあたり +8 行。約 1,070 行は報告 5 本、計画、insight。残りは仕様 1 行と tech-debt(新しい 3 行と、既存の 1 行への注記)。読むのは director skill の 1 写しと、テストの 2 ファイルで足りる。

## 1. 何が足りなかったか

director は、分割計画の機能を issue に起票するとき、印の行(`ralph-split: <分割計画の id>/<slug> after: <ブランチ>,...`)の `after:`(merge を待つブランチ)を、走っている org の予約との重なりからしか決めなかった。`- Depends on:` の頼る先は使われなかった。予約を重ならないように分けた分割計画では、issue の報告によると、頼る先が走っている org の機能は director が `after:` に手で書き、頼る先も issue に起票した機能は `after: none` になった。頼る先が merge されても「着手できる」と出なかった(issue #364 の例: `api` は `locks` に、`jobs-close` は `dataform` と `jobs-ingest` に頼る)。director は頼る先の org を立てるたびに、待つ側の印の行を `gh issue edit` で書き換え、follow-up を打ち直していた(issue の報告では 9 件のうち 4 件)。

直し方は、待つ単位をブランチのままにして、頼る先の機能のブランチ `<Type>/<slug>` を、まだなくても待つ形にした。ブランチの名前は分割計画の Type と slug で決まる。`ralph org followup add` はブランチの形だけを確かめて記録し(`validateFollowupBranch`、`internal/org/followup.go:239`)、merge-order は PR のないブランチの follow-up を `waiting` と出す(`internal/org/mergeorder_followup.go:315`)ので、コードは変えない。むずかしいのは「頼る先は merge 済みか」の判定で、ブランチの名前と `merged` だけで数えると、過去の別の分割計画の同じ名前のブランチを数えてしまう(計画助言の codex の指摘)。下の 2 の (a)〜(c) がその確かめ。

## 2. director skill の変更(`.claude/skills/director/SKILL.md`、読む順)

行番号は HEAD のもの。

- 人への聞き方(`:523-524`): 選択肢で聞く場面に、「起票と記録」の 1 の頼る先の merge の問いを足した。引き継ぎで聞くときも同じ形で聞く

「起票と記録」(`:699`)

1. 待つブランチを決める(`:704-803`)
   - 待つのは 2 つ(`:704-705`)。予約が重なる org のブランチ(`:704-716`、今までの手順)と、機能が頼る先の機能のブランチ(`:718-729`)。頼る先の org が走っていれば、その `feature:` の行のブランチを待つ。走っていなければ、分割計画の `<Type>/<slug>` を待つ(`- Type:` のない機能は `feat/<slug>`)。頼る先を issue に起票したときも、まだ立てていないときも同じで、ブランチがまだなくても待つ。同じブランチは 1 つにする(`:796-797`)
   - 単一の定義「頼る先の merge の確かめ」(`:731-770`)。規則はここにだけ書き、ほかの節は名前で呼ぶ。`ralph org merge-order --all --json` の `entries` で、そのブランチの件の `class` が `merged`、`binding` が (a) か (b)、`pr` が (c) に当たるときだけ merge 済みと数える
     - (a)(`:740`): `binding.split` がその分割計画の id で、`binding.feature` がその機能の slug
     - (b)(`:743`): `binding.feature` がその機能の slug で、`binding.issue`(#365 の `--issue` が記録する番号)が、その機能を起票した issue の番号。頼る先を issue から着手すると、director は機能 1 つの分割計画を書き直すので、`split` は元の id と違う
     - (c)(`:752`): `pr.number` の PR の merge の時刻(`gh pr view --json mergedAt`)が、`binding.ts` より前でない。`binding` は最新の結びつき、`pr` は番号が最大の PR なので、結びつきの org が PR を開く前に disband すると、`pr` が前の試みの PR になるため。時刻を読めないときは数えないが、(c) に当たらないと決まったのではない。「merge を待つ issue」の 2 は done も記録し直しも打たず、報告のたびに時刻を読み直し、人に伝えるのはそのブランチで 1 回だけ(`:762-770`)
   - 問い(`:772-797`): `binding.feature` だけが合うときは、待つブランチを決める前に 1 回だけ聞く。選択肢は「新しい merge を待つ」(推奨)と「この merge と数える」。答えは印の行の `after:` に残るので、4 と引き継ぎは聞き直さない
   - 待つブランチがないとき(`:799-803`): 重ならず、頼る先もない機能。頼られる機能を起票したので起票した機能(「機能ごとの出力先」の 2)も、頼る先があればそのブランチを待つ
2. issue の本文(`:804-822`): 機能の `- Type:` を書く(分割計画になければ `- Type: feat`、`:806-811`)。あとで機能 1 つの分割計画を書き直すとき、同じ Type を本文から読むため
3. 起票を人に伝える(`:824-830`): 待つブランチごとに、何のために待つか(予約が重なる org か、頼る先の機能の slug とそれを起票した issue の番号)を言う。頼る先のブランチにまだ PR も org もなく、頼る先の issue が `after: none` なら、人が着手を決めるまで待つ側は進まない、と言う
4. follow-up の記録(`:831-860`): 重複の判定は、`NOTE` の列が今の印の行と同じ open の件だけ(`:831-835`)。まだない(PR も org もない)ブランチも、そのまま記録する(`:836-840`)。分割計画の機能の `<Type>/<slug>` に当たるブランチは、`merged` でも頼る先の merge の確かめに当たるときだけ merge 済みと数え、当たらなければ聞かずに記録する(`:847-850`)
5. 記録が途中で止まったとき(`:862-863`): 文は変えていない。`after:` に頼る先のブランチが入るので、引き継ぎの記録し直しが頼る先のブランチにも働く

例(`:865-910`): #364 の 5 機能。頼られる 3 機能の issue は `after: none`、`api` の issue は `feat/locks`、`jobs-close` の issue は `feat/dataform,feat/jobs-ingest` を待つ(印の行は `:881-891`)。`locks` に着手して `feat/locks` の PR が merge されると、`api` の follow-up が `ready` になり、director が着手を聞く(`:893-899`)。起票のときに前の別の分割計画の `feat/locks` が merge 済みなら、1 の問いを 1 回聞く(`:901-910`)。

「merge を待つ issue」(`:2464`)

- 引き継ぎ(`:2473-2569`)
  - 数え方(`:2488-2503`): 今の印の行の follow-up だけ数える。固定の文の done の件(`<ブランチ> の merge は頼る先の merge でない`)は人の答えではないので数えない。merge 済みと数えるのは、頼る先の merge の確かめに当たるときだけ
  - follow-up のないブランチ(`:2510-2526`): `merged` なのに `feature` だけが合うときに限り、記録の前に 1 回聞く
  - まだない(PR も org もない)ブランチ(`:2546-2558`): follow-up は `waiting` のまま待つ(DETAIL は `<ブランチ> has no PR yet`)。まだ人に伝えていないものだけ 1 回伝える。待つ側の issue、待つブランチ、頼る先の issue の番号を言い、頼る先の issue が `after: none` なら、人が着手を決めるまで待つ側は進まない、と言う
- 2 着手を聞く前の判定(`:2578-2662`)
  - `ready` の確かめ直し(`:2606-2627`): 頼る先の機能のブランチは、`ready` の follow-up があっても、頼る先の merge の確かめに当たるときだけ待ち終えたと数える。follow-up の `ready` は記録より後の merge を数えるだけなので、同じ名前の別の分割計画の機能の PR の merge でも立つため。当たらなければ着手を聞かず、固定の note で done にし、今の印の行で記録し直し、PR の番号とともに人に伝えて待ち続ける(`:2612-2622`)。時刻を読めないときは、done も記録し直しもせず、着手も聞かない(`:2624-2626`)。重なる org のブランチの `ready` は確かめない(`:2626-2627`)
  - 問いの理由(`:2637-2643`): 待ったブランチが頼る先だけなら「頼る先の機能を待つために」、重なる org のブランチもあれば「コンフリクト回避と頼る先の機能を待つために」と言う
- 3 着手するとき(`:2666-2749`)
  - 書き直す機能 1 つの分割計画は、元の機能の slug と Type のままにする。Type は issue の本文の `- Type:` から読む(`:2704-2709`)。ほかの issue の `after:` と follow-up が、そのブランチを頼る先として待っているため
  - 前の start が残した worktree の記録とぶつかるなどで slug か Type を変えたときは、start のあとに、古いブランチを待つ issue の `after:` と follow-up を新しいブランチに移す(`:2709-2720`)

## 3. `/org` skill の 1 文

`.claude/skills/org/SKILL.md:2436-2443`(「merge を待つ issue」の節、`:2383`)。「PR がもう merge されたブランチは follow-up にしない」の箇条に、頼る先の例外を足した。director が起票した機能が頼る先の機能のブランチは、`merged` でも `ready` でも、director skill の頼る先の merge の確かめに当たるときだけ merge と数える。当たらなければ follow-up を記録して待つ。ブランチはまだなくてよい。(a)(b)(c) は並べず、規則が director skill の「起票と記録」の 1 にだけあると言う(cycle 2 の C2-L1 の直し)。4 写しは同じ中身。

## 4. 仕様

`docs/specs/2026-10-07-org-multi-org-director.md:173`(FR-9 の `2026-11-22 に決めたこと(issue #364)`)。待つブランチを頼る先からも決めること、merge 済みと数える (a)(b)(c)、問いを起票で 1 回にしたこと、引き継ぎで聞くのは follow-up のないブランチだけなこと、書き直しで Type と slug を変えないこと、director が印の行を書き換える 2 つの場合を書いた。`ff59a0e1` は仕様の項目を自分で直し、`03ced266` と `792ffd97` の決めごとは、それぞれの周の sync-docs が足した。

## 5. 2 周のパイプライン

cycle 1(上限 2 の 1 周目)

- self-review は MEDIUM 1、LOW 6、Merge: yes。M1 は、「`feature` だけが合うときは人に聞く」が、ralph が作るブランチ(`<Type>/<slug>`)では常にこの分岐に入り、報告のたびに繰り返される点。`03ced266` で、問いを起票の 1 回に絞り、答えを印の行の `after:` に残して、follow-up のあるブランチは聞き直さない形にした。L1・L2・L3・L5 も同じ commit で直し、L4・L6 は台帳の行(561・562)にした
- verify と test は pass
- cross-review(codex、read-only)は P2 が 2 件で、どちらも ACTION_REQUIRED。(1) 昔の同じ名前の PR の merge が、今の結びつきと組んで `merged` と出る(結びつきの org が PR を開く前に disband したとき)。(2) 別の分割計画の同じ名前のブランチの PR が記録より後に merge されると、follow-up が `ready` になる。`ff59a0e1` で、(1) には (c) を、(2) には「merge を待つ issue」の 2 の確かめ直しを足した

cycle 2(上限 2 の最後の周)

- self-review Run 2(HEAD `ff59a0e1`)は LOW 4(C2-L1〜L4)、Merge: yes。`792ffd97` で直した。`/org` skill の文は director の確かめを名前で呼ぶ(L1)。(c) は (a) か (b) に当たる件でだけ時刻を読み、読めなければ数えずに待つ(L2)。引き継ぎの例外を、それが直す規則の条件の中に移す(L3)。重なる確かめを 1 つ外す(L4)
- verify と test は pass。test の変異は 20 個のうち 19 個が落ちた
- cross-review(HEAD `1eacab1d`)は P2 が 2 件で、どちらも WORTH_CONSIDERING。上限の周なので直さず、下の 8 の 1 に書いた。起きたときの結果は、director が人に着手を聞くことで、自動で着手はしない

## 6. 先に読むテスト

`internal/org/skill_issue_depends_test.go`(790 行)。skill の文は、空白を無視した順序つきの部分文字列で固定している。

- `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn`(`:184`): 「起票と記録」の 1 から 4 の文。待つのは 2 つ、頼る先の `<Type>/<slug>`、`- Type:` の既定値(`feat`)、`merged` の class、skill が名指す JSON のキーと台帳の時刻の形を、このパッケージの値と突き合わせる
- `TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling`(`:323`): 問いの名前が「人への聞き方」と引き継ぎにあること。問い、規則、(a)(b)(c)、gh のコマンド、読めないときの 2 文の 8 つの断片が、director の 4 写しのそれぞれに 1 回だけ出ること。「起票と記録」の 4 に、`feature` だけが合うときの問いがないこと。引き継ぎと 2 のその問いは、follow-up のないブランチに限ること。`checkMergeOrderJSONNames`(`:382`)が、`pr`、`number`、`ts` と gh の `mergedAt` を構造体のタグと突き合わせる
- `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits`(`:418`): `followup add` が、まだないブランチを受け付けること。merge-order がそれを `waiting` と `<ブランチ> has no PR yet` で出すこと
- `TestDirectorSkill_TakeoverAndStartQuestionCountADependencyMergedOnlyByItsBinding`(`:465`): 引き継ぎと 2 が、merge 済みと数えるのを頼る先の確かめに当たるときだけにすること
- `TestDirectorSkill_StartCheckCountsAReadyDependencyBranchOnlyByTheDependencyCheck`(`:550`): `ready` の確かめ直し、固定の note、done の次に記録し直す順。`ff59a0e1` で足した
- `TestDirectorSkill_RewriteKeepsTheTypeAndSlugOrMovesTheWaitingIssues`(`:609`): 3 の Type と slug を変えない規則と、変えたときの移し替え
- `TestDirectorSkill_IssueDependsExampleFollowsAFiledDependency`(`:651`): #364 の例の印の行 2 本と待つブランチ。例に出る issue の番号が `#364` だけであること
- `TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge`(`:761`): `/org` skill の箇条が director の確かめを名前で呼び、(a)(b)(c) の印を持たないこと
- 既存の `internal/org/skill_split_questions_test.go` は、変えた文に合わせて固定を直した(+20 −13 行)。merge-order の JSON の `binding` と、PR のないブランチの `waiting` は、既存の `TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs`(`internal/org/mergeorder_followup_test.go:158`)と `TestOrgMergeOrder_FollowupSectionFollowsTheNotes`(`internal/cli/org_merge_order_followup_test.go:138`)が固定している

## 7. 確かめたこと

- `./scripts/run-test.sh`(`RALPH_VERIFY_SCOPE=full`)が rc 0 で 2 回。1 周目は HEAD `95d755b2`(1,233 s)、2 周目は HEAD `60862f1e`(1,402 s)。どちらも 46 本の shell のテストと 11 の Go パッケージ。2 周目で `(cached)` だった 8 パッケージは `-count=1` で打ち直して通った
- skill の文の変異は、1 周目が 10 個(skill の文 8、merge-order の overlay 2)ですべて落ちた。2 周目は 20 個のうち 19 個が落ちた。落ちなかった 1 個は穴を測るために足した変異で、下の 8 の 5
- `./scripts/run-static-verify.sh` は pass(rc=0、2 回)。`check-sync.sh` は DRIFTED 0、`check-skill-sync.sh` は 14 skill が lock-step。director の 4 写しは同じ中身(md5 `2e46821f…`)、`/org` の 4 写しも同じ(`bd1c9f28…`)。HEAD で取り直して確かめた
- 計画の digest `7b44cea2d6bc` は、本文から計算した値と一致した。AC1〜AC7 は 2 周目の verify でも Met
- skill が名指す JSON のキー(`pr`、`number`、`ts`)、gh の `mergedAt`、`binding.ts` の形(`Org.now` が UTC の RFC 3339 で書く)は、コードとテストで突き合わせた
- 計画助言(codex)の HIGH 1 件(過去の別の分割計画の同じ名前のブランチ)は、計画に入れた

## 8. 残したもの(Known gaps)

1. cross-review cycle 2 の 2 件(WORTH_CONSIDERING、直していない)
   - 古い同じ名前の PR が開いたままのとき、新しい試みが PR を開かずに disband し、そのあと古い PR が merge されると、(c) が通って待つ側の着手を聞く。merge-order の結びつき(`collectMergeTargets`)と PR(`pickMergePR`)を結ぶものがない。同じ名前のリモートブランチに古い PR があると新しい試みは force push できないので、起きるのは新しい試みが push の前に止まる場合に限る
   - 頼る先の slug を変えた(`locks` を `locks-v2` に)と、待つ側の `after:` は `feat/locks-v2` に書き換わるが、確かめは元の分割計画の `feat/locks` しか頼る先のブランチと見ない。`feat/locks-v2` に古い merge 済みの PR があると、`waiting` かつ `merged` の道で済みと数える。slug を変える道は人が関わる例外で、計画の Assumptions の 4 つ目(AC3 の例外。書き換えの手順が残る)と同じ道なので、1 つの項目にまとめた
2. gh が失敗し続けると、すでに merge された頼る先を待ち続ける。起票と引き継ぎは follow-up を記録し、follow-up は記録より後の merge だけを数える。2 は次の報告で時刻を読み直し、人に伝えるのは 1 回。人が「この merge と数える」と言える道は書いていない(台帳 563 の (c)。時刻を読み直してから伝える案と、答えの言葉を伝える案は採っていない)
3. 頼る先と同じブランチ名を、あとから別の分割計画の org が結ぶと、最新の結びつきが変わり、本物の merge が (a)(b) に落ちる。起票と引き継ぎは merge 済みの頼る先を待ち続け、2 は本物の `ready` を done にして記録し直し、PR の番号を人に伝える(台帳 563 の (a))
4. `binding.ts` は、そのブランチの最新の完全な `scope_reserved` の時刻。書く道は 3 つで、予約つきの最初の `Spawn`、`idempotentRespawn`(予約のない走っている leader に予約を記録する)、補償の `reserveAgain`(disband か stop で自分の pane の close が失敗したとき)。`ralph org merge` は disband を待つので、`reserveAgain` はふつう merge より前に書く。merge のあとに書く道があれば、本物の merge が (c) に落ちる。`idempotentRespawn` が merge のあとに書きうるかは確かめていない(台帳 563)。(c) に落ちるブランチは、新しい PR が開くまで、報告のたびに gh を 1 回打つ(台帳 563 の (b))
5. テストの穴(台帳 562)
   - どのテストも skill の文を固定するだけで、実際の director のセッションが手順どおりに動くことを追った run はない
   - 固定は文言の部分一致なので、別の言い方の問い(「slug だけが同じなら人に尋ねる」)を step 4、引き継ぎ、2 に足しても落ちない
   - `/org` skill の箇条の検査は `(a)`・`(b)`・`(c)` の印しか見ないので、印なしで条件を並べ直した文は通る(変異 C3c)
   - `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半は、2 つの既存のテストと重なる。前半は `NewFollowupStore(dir).Add` を直接呼ぶので、CLI の `followup add` の道は通らない
   - 仕様の 2026-11-22 の項目を固定するテストはない
6. 計画との差。計画の Scope の 1 つ目と 4 つ目は、頼る先の確かめを (a) と (b) だけとし、(c) と `ready` の確かめ直しを含まない。cross-review の直しで足したので、数える条件は計画より厳しい。計画の本文は digest で守られているので直していない。差は計画の Progress の verify の行に書いた
7. 台帳 561 の (d) と 562 の Trigger (1)(2) が、cycle 2 の直しで当たったが、払っていない(どちらの行にも日付つきの注記を書いた)。561 の (d) は、`/org` skill の文が「起票のときに 1 回だけ」と言い、引き継ぎで follow-up のないブランチに聞く 1 回を書いていない点。562 の (1)(2) は、新しいテストを足したことと、`directorDependsAskingSentences` が読む場所を変えたこと。どちらの直しも 4 写しかテストの変更になり、上限の周だったので見送った。561 の (a)(b)(c) は、移し替えの対象が答え済みの issue まで入ること、slug 変更の案内、Type の読み方の確かめで、頼る先の slug か Type を変える珍しい道に限る。(e) は 2 の理由の言い切りが広いだけで、動きは変わらない
8. 上限は cycle 2/2 に届いた。cross-review の 2 件は直さず、Known gaps にして PR に進める
