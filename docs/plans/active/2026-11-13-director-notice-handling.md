# director-notice-handling

- Status: Approved
- Approved: 2026-11-13 sha256:debc4ca9fe00
- Owner: Claude Code
- Date: 2026-11-13
- Related request: issue #318(director の skill と prompt が、雛形が古いという知らせを見たときの扱いを決めていない)
- Related issue: 318
- Type: docs
- Branch: docs/director-notice-handling

## Objective

ralph の動詞が stderr に出す、人に向けた `note:` の行を director と leader が見たときの動きを決め、skill と role prompt に書く。対象の行は次の 2 つ。

- 雛形の版の知らせ(#313、`scaffoldVersionNotice`): `ralph org start`、`ralph org director attach`、`ralph org director start`、`ralph status` が、`.ralph/manifest.toml` の版が CLI より古いときに出す。末尾は `run 'ralph upgrade' in <dir>`
- codex の hook の承認の知らせ(#310、`codexHookTrustNotice`): `ralph org spawn`、`ralph org start`、`ralph org director start` が、codex で動く座席の hook が承認されていないか切られているときに出す。末尾は `run 'ralph doctor' for the keys and how to trust them`

director が文面どおりに `ralph upgrade` を流して人の知らないうちに追跡ファイルを書き換えること、codex の設定を書き換えて hook を承認すること、知らせを読み流して人に伝えないことを防ぐ。

## Scope

- `/director` skill(`.claude/skills/director/SKILL.md` と `templates/base/.claude/skills/director/SKILL.md`、`.agents` の 2 写しは `scripts/sync-skills.sh` で生成): 「進捗の報告」に小節を 1 つ足し、2 つの行を見たときの動きを書く。行が最初に出る 2 か所(「昇格」の 2 の `ralph status` と、「機能ごとの出力先」の `ralph org start`)から、その小節を 1 行で指す。「してはいけないこと」に 1 項を足す
- headless の director の role prompt(`internal/org/prompts/director.md`): ミッションに 1 項を足し、skill の小節を指す
- leader の role prompt(`internal/org/prompts/leader.md`): 「運用規律」に 1 項を足し、spawn が hook の知らせを出したときの動きを書く。「件を上げる」の「BLOCKED と QUESTION を上げたら、その件にかかわる作業は返事が来るまで止める」の文に、例外がその項にあることを一句で足す
- テスト: skill の 4 写しの小節と禁止の項(`internal/org/director_skill_sync_test.go`)、director の prompt(`internal/org/prompts_test.go`)、leader の prompt(同)。CLI の文面との結び付けは `internal/cli` のテストで、skill が 2 つの行の決まった部分を引いていることを確かめる
- `docs/tech-debt/README.md` の #313 の行(director の skill と prompt が知らせの扱いを決めていない)を RESOLVED にする

## Non-goals

- 知らせの文面と、出す動詞の変更(`--help` の段落も変えない。決めた動きは行が人に約束することを変えない)
- `/org` skill(人と一緒に座席を spawn するセッションの手順)への同じ規則の追加。人がそのセッションに居合わせるので、行は人の目に入る
- implementer と reviewer の prompt の変更。どちらの prompt も 2 つの行を出す動詞(`ralph status`、`ralph org start`、`spawn`、`director attach`/`start`)を打たせていない(`git grep` で確かめた)
- `ralph doctor` の `checkManifestVersion` の文面(別の tech-debt 行)
- director が `ralph upgrade` を流せるようにする仕組み

## Assumptions

- director の skill は、promote したセッションと headless の director の両方が読む。headless の prompt は skill を指し、prompt だけに書くことは要点に限る(既存の項と同じ書き方)
- `internal/org` のテストは `internal/cli` を import できない(`internal/cli` が `internal/org` を import する)。CLI の文面と skill の結び付けは `internal/cli` 側のテストに置く
- 版の行は、同じ版の組なら動詞を打つたびに出る。director が毎回人に伝えると同じ話が繰り返される

## Affected areas

- `.claude/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md`
- `internal/org/prompts/director.md`、`internal/org/prompts/leader.md`
- `internal/org/director_skill_sync_test.go`、`internal/org/prompts_test.go`(または新しい `prompts_notice_test.go`)
- `internal/cli/` のテスト 1 本(新しいファイル)
- `docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/director-notice-handling.html`(かんたんな図 2 枚: `overview` と `easy-hook`、詳しい図 2 枚: `detail-overview` と `detail-who`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 4 枚の図を `shot` で撮って読んだ。詳しい図 1 の矢印の交差と、詳しい図 2 の `director attach` の文字のはみ出しを直して撮り直し、どれもきれいに出ることを確かめた。codex の計画助言を入れたあと「承認の前に」の箱に 2 項を足し、lint(exit 0)とページ全体の撮影をやり直した(図は変えていない)

## Design decisions

Critical forks: スコープ(hook の行も扱うか)と D5(leader の QUESTION を止めないか)。どちらも /consult(consult-plan-notice、判定「進めてよい」)で確かめ、推奨案を自律で選んだ。consult の直し 3 点(「件を上げる」側の一句、行が最初に出る所からの指し、upgrade の流し方の例を書かない)を入れた。codex の計画助言の 2 件(MEDIUM)も入れた(D3 の読み直す場所、D5 のまとめ方)。レーン B のセッションが消えたので、調整役(レーン A)が引き取って入れた。

- D1 director の動き(版の知らせ): 行を見たら、その会話(headless なら pane か端末)で人に 1 度伝える。伝える文には行にある 3 つの値(雛形の版、CLI の版、upgrade を流す directory)を入れ、upgrade は追跡ファイルを書き換えるので流すかどうかと流し方は人が決めると添える(流し方の例は書かない。linked worktree での upgrade の動きをこの計画では確かめていないため)。同じ directory と版の組では、同じセッションで 2 度伝えない。作業は止めない
- D2 director は `ralph upgrade` を打たない。人に頼まれても打たず、人が流すか標準フロー(`/plan` → `/implement`)に任せるよう返す。「してはいけないこと」の「実装しない」と同じ理由(director がしてよいのは ralph と gh の操作だけで、コードと文書の変更は機能の org か標準フローに任せる)で、`ralph upgrade` は repo の追跡ファイルを書き換える
- D3 人が upgrade を済ませたと言ったら、director は、版の行が `run 'ralph upgrade' in <dir>` で名指した `<dir>`(main のチェックアウトの絶対パス、`orgScaffoldRoot`)にある skill(Claude Code は `<dir>/.claude/skills/director/SKILL.md`、Codex は `<dir>/.agents/skills/director/SKILL.md`)を絶対パスで読み直す。セッションが持っている skill の文は古い版のまま残るため(#313 のきっかけは、新しい CLI で古い雛形の skill に 1 週間従い続けた director)。director が linked worktree で動いていて、自分の作業ディレクトリの skill が `<dir>` の skill と違うときは、`<dir>` の方に従い、自分の worktree の写しが古いままだと人に伝える(codex の計画助言 2)。headless の director は、加えて人に `ralph org director stop` と `start` での立て直しを勧める。立て直すと新しい prompt と skill で始まる(stop は director 自身が打てない)
- D4 director の動き(hook の知らせ): `ralph org start` が出した hook の行も人に伝える。codex の設定(`$CODEX_HOME/config.toml`)を書き換えない、codex を起動して hook を承認しない。承認は hook の中身を読んで人が決める
- D5 leader の動き(hook の知らせ): `ralph org spawn` が hook の行を出したら、codex の設定を書き換えず、行をそのまま QUESTION(本文に行と座席)で上げる。この QUESTION は作業を止めない(座席はそのまま使う)。座席を止めるかは人が DECISION で決める。止めないのは #310 の決定(知らせるだけで、起動を断る設定は入れない)に合わせるため。「件を上げる」の止める規則にも、例外がこの項にあることを一句で書く(読む順でどちらから入っても気づくように)。まとめるのは同じ座席の同じ行だけにする。返事が来る前に同じ座席で同じ行がまた出ても、もう 1 度は上げない。別の座席で行が出たら、返事の前でも、その座席と行で QUESTION をもう 1 つ上げる(hook の状態は座席の作業ディレクトリと設定で違い、人が止める座席を選ぶには座席ごとの行が要るため。上げた件に後から足す手段はない。codex の計画助言 1)
- D6 leader は版の知らせを出す動詞を打たないので、版の行についての規則は leader に足さない

## Acceptance criteria

- [ ] AC1: director の skill の 4 写しに、版の知らせを見たときの動き(人に伝える、伝える文の 3 つの値、`ralph upgrade` を打たない、同じ組で繰り返さない、upgrade のあとに行の `<dir>` の skill を絶対パスで読み直し、自分の worktree の写しが違えば `<dir>` の方に従って人に伝える、headless なら立て直しを勧める)が書かれ、「昇格」の 2 と「機能ごとの出力先」の `ralph org start` からその小節を指している(issue #318 の受け入れ条件 1)
- [ ] AC2: 同じ小節に、hook の知らせを見たときの動き(人に伝える、codex の設定を書き換えない、hook を承認しない)が書かれている
- [ ] AC3: 「してはいけないこと」に、`ralph upgrade` を打たないことと codex の設定を書き換えないことが書かれている
- [ ] AC4: headless の director の prompt(herdr と launch_cmd の両方の形)に、小節を指す項があり、`ralph upgrade` を打たないことが書かれている(issue #318 の受け入れ条件 1)
- [ ] AC5: leader の prompt に、spawn の hook の行を見たときの動き(codex の設定を書き換えない、QUESTION で上げる、作業は止めない、まとめるのは同じ座席の同じ行だけで、別の座席の行は返事の前でも別の QUESTION で上げる)が書かれ、「件を上げる」の止める規則の文がその例外を指している
- [ ] AC6: prompt のテストが、director と leader の prompt にその文があることを確かめている(issue #318 の受け入れ条件 2)。skill のテストが 4 写しの小節と禁止の項を確かめている
- [ ] AC7: `internal/cli` のテストが、2 つの行の決まった部分(`run 'ralph upgrade' in`、`run 'ralph doctor' for`)を skill の小節が引いていることを確かめ、文面が変わったら落ちる
- [ ] AC8: `./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る

## Implementation outline

1. director の skill(2 つの元の写しと `.agents` の生成)と director の prompt、そのテスト(AC1〜AC4、AC6 の director の分、AC8) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'DirectorSkill|RenderDirectorPrompt'`
2. leader の prompt とそのテスト(AC5、AC6 の leader の分) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'RenderRolePrompt_Leader'`
3. CLI の文面と skill の結び付けのテスト(AC7) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'ScaffoldVersion|CodexHookTrustNotice|NoticeSkill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、secret scan)
- Spec compliance criteria to confirm: AC1〜AC8。issue #318 の 2 つの受け入れ条件が AC1・AC4 と AC6 に入っていること
- Documentation drift to check: `docs/tech-debt/README.md` の #313 の行、`docs/specs/2026-08-01-org-runtime.md` に director の動きを書いた所があるか、`--help` の段落(変えない判断の確認)
- Evidence to capture: verify の報告、変えた節の行番号

## Test plan

- Unit tests: skill の小節と禁止の項(4 写し)、director の prompt(herdr と launch_cmd)、leader の prompt
- Integration tests: なし(文の追加だけで、動詞の動きは変えない)
- Regression tests: 既存の `TestDirectorSkill_*`、`TestRenderDirectorPrompt_*`、`TestRenderRolePrompt_Leader_*` が変えずに通る
- Edge cases: 結び付けのテスト(AC7)は、`scaffoldVersionNotice` と `codexHookTrustNotice` で実際の行を作り、その行が `run 'ralph upgrade' in ` と `; run 'ralph doctor' for ` を含むことと、skill の 4 写しの小節がその 2 つの断片を引いていることを確かめる。文面を定数に切り出すことはしない(本体の Go を変えない)。skill の読み方は `internal/cli/org_retired_state_dir_condition_docs_test.go` の前例にならう
- Evidence to capture: test の報告、`internal/org` と `internal/cli` のテストの時間

## Risks and mitigations

- skill が長くなる(2,020 行): 小節は 20 行ほどに抑え、`/org` skill には足さない
- leader の QUESTION が増えて人の手間になる: hook の行は未承認か切られているときだけ出る。同じ座席の同じ行は 1 度だけ上げる。座席ごとの件は、人が止める座席を選ぶための情報なので減らさない
- 返事待ちのあいだに別の座席の行が消える: 別の座席の行は返事の前でも別の QUESTION で上げる(D5)。テストは leader の prompt にその文があることを確かめる
- upgrade のあとに director が古い skill を読み直す(director が linked worktree にいて、人は main を更新した): 行の `<dir>` の skill を絶対パスで読む(D3)。テストは skill の 4 写しにその文があることを確かめる
- 「作業を止めない QUESTION」が、「件を上げる」の節の「QUESTION を上げたらその件にかかわる作業を止める」と食い違って読まれる: 項の中で例外であることを書く
- 4 写しのずれ: `sync-skills.sh` と `check-skill-sync.sh`

## Rollout or rollback notes

- 文とテストの追加だけで、CLI の動きは変わらない。戻すときは commit を revert する
- 雛形に配る skill と prompt の変更なので、次の release の behavior note(#317 の仕組み)の対象になる

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。2 件(MEDIUM)。(1) 返事待ちのあいだに別の座席の hook の行が消える → D5 を「同じ座席の同じ行だけまとめ、別の座席は返事の前でも別の QUESTION」に直し、AC5 と Risks に入れた。(2) director が linked worktree にいると upgrade のあとに古い skill を読み直す → D3 を「行の `<dir>` の skill を絶対パスで読み、自分の写しが違えば `<dir>` に従って人に伝える」に直し、AC1 と Risks に入れた。どちらも聞かずに入れた(ユーザーの常の指示)
- [x] Plan approved — 自動承認(ユーザーの「推奨案を自動承認します」、2026-11-13)
- [x] Branch created
- [x] Implementation started
  - S1 a933346c(director の skill 4 写しの「人に向けた知らせの行」、director.md の指し先、`prompts_notice_test.go`、tech-debt 464 を RESOLVED)
  - S2 aa3d1878(leader.md の「運用規律」に hook の `note:` の行の項、「件を上げる」の止める規則に例外の一句)。handoff からの逸脱: 「spawn は成功した」とは書かず「spawn は止まらない。終了コード 0 なら座席は使える」にした(行は `rt.Spawn` の前に出るため、`internal/cli/org.go:966`)。テストファイルへの追記を shell の heredoc で行ったので、U+FFFD がないこと(2 ファイルとも 0 件)を orchestrator も確かめた
  - S3 21026fd4(`internal/cli/director_notice_skill_test.go`: 本体の関数で作った 2 つの行と skill の 4 写しの小節が同じ断片を持つ。断片を変えた写しで 4 写しとも落ちることを確かめた)。3 スライスとも検証コマンドは rc=0
- [x] self-review — 450535e2(MEDIUM 1、LOW 4、Merge: yes)
  - S4 579a35a6(implementer。M1・L1・L2・L4 の直し。検証コマンドは rc=0、変異 6 つすべて死亡、自己点検すべて clean)。計画からのずれ: D3 は「`<dir>` は main worktree のルート」を前提にしていたが、`ralph status` の行は打ったディレクトリを出す(`internal/cli/status.go:223`)。skill は org の 3 つの動詞の行と status の行を分けて書き、読み直すのは main worktree のルートの skill とした(status の行しかないときは `git worktree list` の 1 行目)。director.md の読み直しの文も「main worktree のルートの skill」に合わせた。計画の本文は変えず、ここに記録する
  - L3(hook の行の「で終わる」をテストが `Contains` でしか見ていない、小節を切る補助の終わりの規則が internal/org と違う)は直さない。今は差が出ず、断片が行に含まれることは確かめている
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
