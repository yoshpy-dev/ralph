# split-preamble

- Status: Approved
- Approved: 2026-10-22 sha256:f0735c9a1ac6
- Owner: Claude Code
- Date: 2026-10-22
- Related request: 試行の所見を直す系列の 5 本目「分割計画の前文」(ユーザーの依頼「2. 修正可能なものは修正してください」、2026-10-21)
- Related issue: N/A
- Type: fix
- Branch: fix/split-preamble

## Objective

`ralph org start --plan` が立てる leader のタスクに、分割計画の前文(`## Features` より前の `## ` の節)を入れ、分割計画に一度だけ書いた全機能に共通の決まり(座席の model、してはいけないことなど)が、どの機能の leader にも届くようにする。試行(stage 9)の所見 F5 を閉じる。あわせて、記録に残る「fix/seat-wip-checkpoint」を「#250」に直す。

## Scope

- 分割計画の読み込み(`internal/org/split.go` の `parseSplitPlan`)が、前文を `SplitPlan.Preamble` に読む。前文は、ヘッダ(最初の `## ` の行より前)のあとから `## Features` の見出しの前までの `## ` の節で、見出しの行を含み、フェンスのコードブロックは囲みの行ごとそのまま残し、前後の空行を詰める
- leader のタスク(`internal/org/feature.go` の `featureLeaderTask`)が、今の行のあと、機能の本文の前に、前文を 1 行の前置きとフェンスで囲んだブロックで入れる。前文が空ならタスクは今と同じ
- leader の雛形(`internal/org/prompts/leader.md`)の「機能ごとの org」の手順 1 に、タスクの前文を機能の計画に囲みの行ごと原文のまま写す 1 文を、手順 3 に、承認の前に立てる reviewer にも手順 5 の driver と model を使う 1 文を、手順 5 に、前文の driver と model を spawn の引数に反映し、ほかの決まりは TASK の本文に写さず計画を読むよう指す 1 文を足す(D5)
- `ralph org start` の help の「its task from the feature's body」と、`--plan` に task の引数を渡したときのエラーの文を、前文も入る形に直す
- 文書: `/org` skill(4 写し)の分割計画の節と start の手順 6、`/director` skill(4 写し)の「分割計画と機能ごとの org」、仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-4、README の機能ごとの org の段落
- 記録: tech-debt の F5 の行を RESOLVED にする。根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` の F5 の行の直す PR。仕様・根拠の表・tech-debt に残る「fix/seat-wip-checkpoint」を「#250」に直し、仕様と根拠の表の PR 一覧を 33 本にする

## Non-goals

- 分割計画の書式を変えること(専用の節を足す、`## Features` より後の節を拒否する、など)。既存の分割計画はそのまま読める
- ヘッダの自由文(題など)、`## Features` の中の最初の `### <slug>` より前の行、`## Features` より後の節を leader に渡すこと。今までどおりどこにも渡らず、skill にそう書く
- 前文を座席(implementer・reviewer)に ralph が直接渡すこと。leader が雛形の指示で TASK に書く
- leader が前文の決まりに従ったかを ralph が検査すること(model の検査など)
- 前文の長さの上限
- 機能の本文の渡し方を変えること(今と同じく生のまま末尾に付ける)
- 実機の確かめ(文字の組み立てだけなので、Go のテストで role prompt まで見る)

## Assumptions

- leader のタスクは role prompt のファイルの `{{TASK}}` に入り(`internal/org/prompts.go` の `strings.NewReplacer`)、agmsg の 2,000 字の上限はかからない。leader から座席への TASK には上限がかかる(`internal/org/verbs.go` の `Send` が `protocol.DefaultMaxBodyChars` で検査する)ので、前文は TASK の本文で運ばない(D5)。`NewReplacer` は 1 回の走査なので、前文の中の `{{ORG_ID}}` のような文字は置き換えられない
- 承認の digest はファイル全体から取るので、前文はもとから承認の対象に入っている。digest の規則は変えない
- 前文の中の `- Status:`・`- Approved:`・`- Branch:`・`## Progress checklist`・行頭の `- [x]` は、今の読み込みがすでに拒否する
- 機能の本文には生の `## ` も `### ` も入らない(入れば節が終わる)。前文は定義上 `## ` の見出しで始まるので、前文だけを囲む理由になる(consult)

## Affected areas

- `internal/org/split.go`、`internal/org/split_test.go`
- `internal/org/feature.go`、`internal/org/feature_test.go`
- `internal/org/prompts/leader.md`、`internal/org/prompts_test.go`
- `internal/cli/org.go`(start の help とエラーの文)と、その文を固めるテスト
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と写し(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)
- `docs/specs/2026-10-07-org-multi-org-director.md`、`README.md`
- `docs/tech-debt/README.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`

## Visual review

- ページ: `.harness/state/plan-visual/split-preamble.html`(かんたん版の図 2 つ: overview、easy-parts。詳しく: detail-overview、detail-task、detail-parse)
- lint: `./scripts/plan-visual.sh lint` rc 0
- 自己点検: ページ全体と 5 つの図を `shot` で撮って読んだ。detail-overview の辺のラベルが cli の箱に重なっていたので箱を下げて撮り直し、すべて重なり・切れなし。D5 を足したあと、overview・ページ全体・detail-overview を撮り直して読んだ(lint rc 0、重なり・切れなし)
- Codex の計画助言: 指摘 1 件(MEDIUM、座席への TASK の 2,000 字の上限)。D5、AC4・AC5、Test plan、Risks に入れた
- S2 のあとの consult の指摘 3 件(手順 3 の reviewer、囲みの行ごと写す、前文が履歴に入る)を AC5、Scope、Risks、S2b に入れ、detail-overview と「承認の前に」を直して撮り直した(lint rc 0、重なり・切れなし)。計画は承認し直した

## Design decisions

- D1 渡す範囲: 前文 = ヘッダのあとから `## Features` の見出しの前までの `## ` の節(案 A)。tech-debt の行の契機の欄が書く「the text between the header lines and `## Features`」と同じ範囲で、試行の分割計画がそのまま効く。専用の節だけを渡す案(B)は、試行の分割計画を書き直す必要があり、節の名前を間違えると黙って落ちる。書いたものをすべて渡す案(C)は、`## Features` の見出しや前後の継ぎ目の形が決めにくい。consult(consult-plan-preamble)の判定も A
- D2 渡し方: 前文をフェンスで囲む。フェンスはバッククォートで、前文の中の最長のバッククォートの並びより 1 つ長く、最低 3。前文の `## 目的` などの見出しが leader の role prompt の節(`## スター型トポロジのルール` など)と並んで読めないようにするため。前に「全機能の座席に適用する決まり」と言い切る 1 行を置き、leader.md の手順 5 の 1 文と同じ言葉にそろえる(囲むと参考と読まれるおそれがある、という consult の指摘)。順は、今の行、空行、前置きの行とブロック、空行、機能の本文
- D3 空の前文: 前文が空(節がない、空行だけ)なら、前置きもブロックも出さず、タスクは今とバイトで同じ
- D4 拒否しない: `## Features` より後の節やヘッダの自由文は、今までどおり読み込みを通し、どこにも渡さない。拒否すると、ユーザーの台帳の既存の分割計画が読めなくなる
- D5 座席への渡し方(Codex の計画助言の指摘 1): leader は前文を、手順 1 で書く機能の計画に原文のまま写す(手順 1 はどちらの承認方式でも書き、per-plan では人がその計画を承認する)。座席には、driver と model を spawn の引数で渡し、ほかの決まりは TASK に書いた機能の計画のパスから読ませる。前文を TASK の本文に写すと、長い前文で 2,000 字の上限に当たり、削ると決まりが落ちるため。leader が写したかを ralph は確かめない(雛形の指示だけ)
- Critical forks: None(D1〜D4 は consult と相談して決め、D5 は Codex の指摘から決めた。どれも 1 スライスの中で戻せる)

## Acceptance criteria

- [x] AC1: `LoadSplitPlan` が試行の分割計画 `docs/evidence/org-trial-2026-10-20/splits/trial.md` を読むと、`Preamble` は `## 目的` の見出しから「予約の外のファイルは編集しない」の行までで、「座席の決まり」を含み、`# 9 段目` の題、`- Status:`・`- Approved:` の行、`## Features` の見出し、機能の節を含まない。機能(slug、Type、Reserve、Depends on、Body)は今と同じに読める
- [x] AC2: 境界。前文の中のフェンスに入った `## Features` の行は前文の一部として残り、本当の `## Features` を見誤らない。`## Features` より後の `## ` の節は `Preamble` にも機能の本文にも入らない。前文の前後の空行は詰める。ヘッダのあとすぐ `## Features` なら `Preamble` は空。CRLF の分割計画でも前文の行は `\r` を持たない
- [x] AC3: `featureLeaderTask` は、前文が空でなければ、今の行のあとに空行、前置きの 1 行、フェンスで囲んだ前文、空行、機能の本文の順で書く。フェンスはバッククォートで、前文の中の最長のバッククォートの並びより 1 つ長く、最低 3(前文に 4 連があれば 5 連)。前文が空なら、タスクは今と同じ文字列
- [x] AC4: stub の herdr で `ralph org start --plan`(または `StartFeature`)を回すと、leader の role prompt の `## タスク` の節に前置きの行と前文のブロックがあり、role prompt の既存の節(`## スター型トポロジのルール` など)は 1 つずつ見つかる。前文の中の `{{ORG_ID}}` は置き換えられずに残る。2,000 字を超える前文でも start は通り、前文は削られずに role prompt に入る
- [x] AC5: leader の雛形の「機能ごとの org」の手順 1 に、タスクの「全機能の座席に適用する決まり」を機能の計画に囲みの行ごと原文のまま写す 1 文があり、手順 3(per-plan で承認の前に reviewer を立てるところ)に、そのときも手順 5 の決まりの driver と model を使う 1 文があり、手順 5 に、その決まりの driver と model を spawn の引数に反映し、ほかの決まりは TASK の本文に写さず、TASK に書いた機能の計画から読ませる 1 文がある。どちらも前置きの行と同じ言葉(「全機能の座席に適用する決まり」)を使う。`internal/org/prompts_test.go` がその 3 文を固め、手順の位置を固める既存のテストは変えずに通る
- [x] AC6: `ralph org start` の help と、`--plan` に task の引数を渡したときのエラーが、タスクは機能の本文と分割計画の前文から作られると書く。それを固めるテストの期待も直す
- [x] AC7: 文書を合わせる。`/org` skill(4 写し)の分割計画の節が「全機能の leader に渡るのは `## Features` より前の `## ` の節だけで、ヘッダの自由文、`## Features` の中の最初の `### ` より前の行、`## Features` より後の節は渡らない」と書き、start の手順 6 のタスクの中身に前文を足す。`/director` skill(4 写し)の分割計画の手順に、全機能に共通の決まりは `## Features` より前の節に書くと 1 文。仕様の FR-4 と README の段落に前文が渡ることを足す。写しはバイト一致
- [x] AC8: 記録を合わせる。tech-debt の F5 の行を取り消し線と RESOLVED の注で閉じる。根拠の表の F5 の行の直す PR を書く。仕様・根拠の表・tech-debt に残る「fix/seat-wip-checkpoint」を「#250」に直し(tech-debt の RESOLVED の注と、`wip:` の行の「Step 14 ... (fix/seat-wip-checkpoint)」を含む)、PR 一覧を 33 本にする。この PR は、4c と同じく merge の前のブランチ名 `fix/split-preamble` で書き、番号は次の PR が書く。仕様の印の数(`[x]` 26 / `[ ]` 6、根拠の表は 25 / 6)は、F5 が挙げる FR-4 が `[x]` なので変わらないはず
- [x] AC9: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0。変えたパッケージの Go のテストが `GOTOOLCHAIN=go1.25.8` でも通る

## Implementation outline

1. S1(implementer): D1・D4。`parseSplitPlan` が前文を読む。`SplitPlan.Preamble` と doc、split.go の先頭のコメントの書式の例。split_test.go(AC1、AC2)
2. S2(implementer): D2・D3・D5。`featureLeaderTask` の前文のブロック、leader.md の手順 1 と手順 5 の 1 文ずつ、start の help とエラーの文、それぞれのテスト(AC3〜AC6)
   - S2b(implementer、S2 のあとの consult の指摘): leader.md の手順 1 の「囲みの行ごと」と手順 3 の 1 文、prompts_test.go の固め(AC5)
3. S3(implementer): 文書(AC7)。skill の写しは `scripts/sync-skills.sh` で作り、`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` で確かめる
4. S4(implementer、sonnet): 記録(AC8)

## Verify plan

- Static analysis checks: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh`(gofmt、go vet、shellcheck、check-sync、check-skill-sync ほか)
- Spec compliance criteria to confirm: AC1〜AC9。仕様の印の数が変わらないこと(変わるならその理由)
- Documentation drift to check: 分割計画のどこが leader に渡るかを書いた文(`/org`・`/director` の skill、README、仕様、start の help、`featureLeaderTask` と `StartFeature` の doc)が食い違わないか。「`## Features` の外の節は自由に書ける」の古い文が残っていないか
- Evidence to capture: run-verify の rc と時間

## Test plan

- Unit tests: split_test.go に、試行の分割計画のファイルを読むテスト(AC1)と境界のテスト(AC2: 前文の中のフェンスの `## Features`、`## Features` より後の節、前後の空行、空の前文、CRLF、前文の中の `### ` の行)。feature_test.go の `featureLeaderTask` に、前文あり・なし・4 連のバッククォートを含む前文・前文の中のフェンス(AC3)
- Integration tests: `StartFeature` か CLI の `ralph org start --plan` を stub の herdr で回し、leader の role prompt を読む(AC4。2,000 字を超える前文も)。prompts_test.go で手順 1 と手順 5 の 1 文ずつ(AC5)。start の help とエラーの文(AC6)
- Regression tests: 前文のない分割計画で、タスクが今とバイトで同じ。今の split・feature・prompts・start のテストが、期待の文を直すほかは変えずに通る。手順の位置を固める `TestRenderRolePrompt_Leader_FeatureOrgProcedure` は変えない
- Edge cases: 前文が空行だけ、前文の中の `{{ORG_ID}}`、前文の中の 3 連と 4 連のバッククォート、チルダのフェンス、`## Features` の見出しの末尾の空白(今の読み込みは `TrimRight` で許す)
- Evidence to capture: `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` を go1.26 と go1.25.8 で

## Risks and mitigations

- 前文に長い背景があると、全機能の leader の role prompt がその分長くなる: 上限は設けず、skill に「全機能に共通の決まりを前文に書き、長い背景は機能の本文か別のファイルに書く」と書く
- 囲んだ前文を leader が参考と読み、決まりとして扱わない: 前置きの行で「全機能の座席に適用する決まり」と言い切り、手順 1 で機能の計画に写し、手順 5 で spawn の引数に反映すると書く。leader が従わなければ届かない(雛形の指示であって ralph の検査ではない)
- leader が決まりを座席への TASK の本文に写し、長い前文で 2,000 字の上限に当たる、または削って決まりを落とす(Codex の指摘 1): 手順 5 で TASK の本文に写さないと書き、決まりは手順 1 で写した機能の計画から読ませる(D5)
- leader が手順 1 で決まりを写すときに言い換える: 「原文のまま」と書く。per-plan では人が計画を承認するときに見られる
- leader が囲みを外して写すと、前文の `## 目的` などが機能の計画の節として並ぶ: 手順 1 に「囲みの行ごと」と書く(S2 のあとの consult の指摘)
- per-plan で手順 3 の reviewer を承認の前に立てると、手順 5 の driver と model の 1 文を読む前に立つ(試行の doctor-escape の reviewer が opus で動いたのと同じ形): 手順 3 にも 1 文を足す(S2 のあとの consult の指摘)
- 分割計画は台帳(git が無視する場所)にあってコミットされないが、手順 1 で機能の計画に写した前文はコミットされ、機能の PR と計画の archive に入る: `/org` skill の分割計画の節に、前文はリポジトリの履歴に入ると書く(S3。S2 のあとの consult の指摘)
- 既存の分割計画で、前文を背景のつもりで書いていた作者には、渡る中身が増える: release のノートに書く。承認の digest は変わらない
- `## Features` より後の節に決まりを書いた作者は、今までどおり届かない: skill に書く。拒否はしない(D4)
- 前文のフェンスの長さを数え違えると、前文の中のフェンスが外のフェンスを閉じる: AC3 のテストで 4 連のバッククォートを含む前文を見る

## Rollout or rollback notes

- 効くのは次の release の binary から。scaffold の skill の文は `ralph upgrade` で入る
- 戻すときはこの PR を revert する。分割計画の書式も digest も変えないので、台帳の分割計画はどちらの binary でも読める

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(ユーザーの決定により自動承認、2026-10-22。S2 のあとの consult の指摘を入れて承認し直した: f6a95e46e01d → f0735c9a1ac6)
- [x] Implementation started(S1 03651616、S2 ef33170f、S2b b10c6f25、S3 1b65bf49 と 546ad029、S4 c407bd67)
- [x] Branch created
- [x] Review artifact created(self-review ed5d57d6、Merge OK、LOW 4 件は b09b7c12 と c0ff8398 で直した)
- [x] Verification artifact created(verify ab89243c、pass、INFO 3 件。I1 と I3 は次のコミットで直し、I2 はこの印)
- [x] Test artifact created(test de6ca6b2、pass。G1・G2 は 7a955d2c で埋めた。1 回目の run-test.sh で落ちた internal/cli のテストはこの diff と関係しない前からの競合で、tech-debt の行にした(33e51acc))
- [x] run-verify: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は e9824d39 から打ち、rc 0(2026-10-22T16:00:33Z に開始。走っているあいだに walkthrough の 6e1ae46b が入り、branch の secret scan は a7bdc884..6e1ae46b を読んだ。6e1ae46b は報告 1 つだけ)。internal/cli と internal/org は cache なしで ok。go1.25.8 は test の報告と 7a955d2c の implementer が打った
- [x] Cross-review(cycle 1/2、指摘 0 件、triage b59ca7df)
- [ ] PR created
