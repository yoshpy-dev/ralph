# Walkthrough: `ralph org reseat` が codex の hook の承認の行を出し、人向けの 2 つの行の一覧に reseat を足す(reseat-notices)

- 計画: `docs/plans/archive/2026-11-16-reseat-notices.md`(PR の作成のあとに archive へ移す)
- issue: なし。v8.2.0 の release のあとに見つけた漏れで、元は #310(hook の行)と #313(雛形の版の行)
- 差分: `git diff origin/main...HEAD`(この walkthrough を足す前で 26 ファイル、約 +1,300 / −112、base `30e122a1`)。Go の本体は 4 ファイルで約 +100、Go のテストは 5 ファイルで約 +450、skill・prompt・README は約 +70、残りは report・計画・文書
- パイプライン: 1 cycle。cross-review(codex)の指摘は 0 件

雛形の版の行は reseat がすでに出していた(`internal/cli/org_reseat.go:220`)。足りなかったのは hook の行と、2 つの行を出す動詞を並べる文書の一覧だった。reseat は `--driver codex` で座席を codex に替えて立て直せるので、承認されていない hook のまま codex の座席が知らせなく動きうる。

## 1. `ReseatParams.OnPlan`(`internal/org/reseat.go`。読む価値が一番高い)

- フィールドは `:74-87`。`Reseat` が呼ぶ場所は 2 つある。dry run は `PlanReseat` が通った直後(`:1128-1130`。ロックは取らない)。本物の reseat は、ロックを取り、`PlanReseat` が通り、ロックの印に reseat の id を書いたあとで、`reseatRun` を作る前(`:1151-1156`)。最初の `stopAll`(`:1170`)より前なので、途中で切れた reseat でも行は先に出る(計画の D1。spawn と start も agent を立てる前に出す)
- 呼ばない場合: flag の誤り(`:1138`)、ほかの reseat がロックを持つ(`:1141`)、`PlanReseat` の断り(`:1147`)、印の書き込みの失敗(`:1151-1153`)、`OnPlan` が nil のとき。呼ぶのは多くて 1 度
- ロックの契約(`:74-86` の doc): 本物の reseat ではロックの中で呼ばれるので、ralph の動詞を打たず、遅い処理も待ちもしない。計画を読むだけで書き換えない。呼ばれた時点で、印に reseat の id があり、台帳に新しい event はない
- 関数を渡す形にしたのは、internal/org が internal/cli を import できないため(計画の D2)

## 2. CLI 側 `printOrgReseatCodexHookTrustNotices`(`internal/cli/org_reseat.go:258-277`)

- `runOrgReseat` が `OnPlan` に渡す(`:226`)。`plan.Start` の座席ごとに `printCodexHookTrustNotice(w, seat.Driver, "org reseat", seat.SeatID, plan.Binding.Worktree)` を呼ぶ(`:275`)。claude の座席は `codex_hook_trust_notice.go:38` の driver の判定で何も出ない
- dir は機能の worktree。座席は皆そこで動き、`codexHookTrustFor` が linked worktree を main のチェックアウトに読み替えるので、`start --plan` が main のルートで確かめるのと同じ hooks.json を読む。cwd を渡すと、入れ子の `.codex/hooks.json` の層まで数える(S7 のテストが押さえる)
- `plan.Binding == nil` なら何もしない(`:271`)。`PlanReseat` は binding の無い org を断る(`reseat.go:422-424`)ので、本物の経路では来ない。断りは `ReseatPlan{}` を返すため、dry run の `err == nil` を外した変異(M6)は CLI のテストでは落ちず、org のテストだけが落とす

## 3. help の変更

- `ralph org reseat --help`: dry run の段落に 1 文(`org_reseat.go:99-100`。「Like a real reseat, it prints the codex hook trust note … for each codex seat it would start」)と、末尾に hook の段落(`:133-135`。雛形の版の段落の次)。`TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(`org_reseat_test.go:838`)が dry run の文(`:888`)と、末尾の 2 段落(`HasSuffix`、`:910`)を固定する
- 共有の `orgCodexHookTrustNoticeHelp`(`codex_hook_trust_notice.go:112-124`)を、複数の agent に合う言い方にした(「the agent」を「an agent」に `:112`、「that agent runs in」`:114`、「one note on stderr for each such agent」`:119`。折り返しも変わった)。この定数は spawn(`org.go:917`)、start(`org.go:1158`)、director start(`org_director_headless.go:165`)の `--help` にも入るので、4 つの help が同時に変わる
- 3 つの doc comment が reseat を挙げる: `printCodexHookTrustNotice`(`codex_hook_trust_notice.go:20-25`)、`orgCodexHookTrustNoticeHelp`(同 `:107-111`)、`orgScaffoldVersionNoticeHelp`(`scaffold_version_notice.go:107-114`)

## 4. `/director` skill と director の prompt

- 小節「人に向けた知らせの行」(`.claude/skills/director/SKILL.md:1612`): 雛形の版の行の動詞に reseat を足し(`:1617`)、「`ralph org` の 4 つの動詞」にした(`:1622`)。hook の行の動詞にも足し(`:1648-1649`)、reseat の行も start の行と同じくそのまま人に伝える(`:1653-1654`)
- 「座席を立て直す」(`:1469`)の手順 2 に、dry run が出した行は小節に従う、という指し先(`:1494-1495`)。手順 4 に「本当の reseat も dry run と同じ行を同じ座席に出す。dry run のあとで伝えた座席の行は、もう 1 度は伝えない」(`:1512-1513`)、小節の reseat の項に同じ決まり(「その行は 1 度だけ伝える」`:1655-1656`)。この skill は dry run のあとに本当の reseat を打たせるので、同じ行が 2 度出るため。計画の Scope の外で、S4 の「計画からのずれ」として Progress に記録してある
- `internal/org/prompts/director.md:62-67`(headless の director)も動詞の一覧に reseat。skill の 4 写しは同じ内容(`cmp` で確認)

## 5. テストと変異

- org: `TestReseat_OnPlanRunsOnceBeforeTheFirstStop`(`reseat_test.go:1473`。全座席、codex の 1 席、dry run。記録の順が `plan` のあとに `stop:…`)、`TestReseat_OnPlanIsNotCalledOnARefusal`(`:1561`。5 件の断り)
- CLI: `codex_hook_trust_notice_test.go` の `TestCodexHookTrustNotice_Reseat*` が 5 つ(`:448`、`:485`、`:513`、`:533`、`:589`)。`:448` は行を書いた時点の manifest が reseat 前と同じことで「止める前」を確かめる。`:589`(S7)は codex・claude・codex の 3 席を、入れ子の hooks.json のある下のディレクトリから渡す。`TestScaffoldVersionHelp_EachVerbDescribesTheNote`(`scaffold_version_notice_cli_test.go:275`、`:292` が reseat)、skill と prompt は `prompts_notice_test.go:176`、`:250`、`:301`
- 変異: 実装の S1、S2、S4 が 7、5、5 つを当て、すべて落ちた。tester は M1 から M10 を `go test -overlay` で当て、8 つが落ち、M4(dir を `"."`)と M10(最初の 1 席で抜ける)が残った。S7 がその 2 つを落とすテストを足した(S7 のあとの打ち直しはしていない)。`run-test.sh` は rc 0(Go 11 パッケージ、shell 45 本)

## 6. 文書

- `README.md:133`(hook の行を出す動詞に reseat と、座席ごとの 1 行)、`docs/specs/2026-10-07-org-multi-org-director.md:164`(#318 の項に日付つきの 1 文)、`.codex/README.md:206-209` と `templates/base/.codex/README.md`(同じ文)
- `docs/tech-debt/README.md`: 513 の (i) を払った印、513・514・515・480 に持ち越しの note、468・479・480・515 の動詞の一覧と今の状態の文、515 の (e) に印の書き込みの 1 文

## 見どころ

- 共有の hook の段落の文言が、reseat のほかに spawn、start、director start の `--help` も変える。座席 1 つの動詞では「for each such agent」は 1 行と読める。定数そのものを比べるテスト(`codex_hook_trust_notice_test.go:739`)は、文言が変わっても落ちない
- 計画の前に断られた reseat は行を出さない。start との違い: `start --plan` は driver を決めたあとすぐ行を出す(`org.go:1332`)ので、そのあと `StartFeature` に断られても行が残る。reseat の行は座席を名指すので、座席が決まる `PlanReseat` のあとにしか出せない。計画のあと、最初の止めの前に断られたときは start と同じく行が残る。`ReseatRefusedBeforeItsPlanPrintsNothing`(`codex_hook_trust_notice_test.go:533`)が前者を押さえる
- 開いている穴: (1) 印の書き込みが失敗すると `OnPlan` を呼ばずに返す枝(`reseat.go:1151-1153`)は、失敗させる差し込み口がなくテストがない(tech-debt 515 の (e)、test report の G4)。(2) 本物の herdr と codex では走らせていない(stub の herdr だけ。tech-debt 480 の (c)、515 の (g))。(3) reseat を走らせて雛形の版の行(`org_reseat.go:220`)を確かめる CLI のテストがない。この PR の前からの穴で、help の側(`scaffold_version_notice_cli_test.go:292`)だけを足した(tech-debt 515 の (i))
- 1 回の reseat で codex の座席が複数ある場合と codex の leader は、CLI の fixture が座席 1 つにしか `--driver` を渡せないので、`codex_hook_trust_notice_test.go:589` が関数を直接呼ぶ形で押さえる。reseat を最後まで走らせるテストではない
