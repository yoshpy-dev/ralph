# Walkthrough: split-preamble(試行の所見 F5、分割計画の前文を全機能の leader に渡す)

- Date: 2026-10-23
- Plan: docs/plans/active/2026-10-22-split-preamble.md(/pr で archive に移す)
- PR: 未定(PR を作ったあとに番号を書く)
- Branch: fix/split-preamble(base a7bdc884、この walkthrough の前の HEAD b59ca7df。22 commits、29 files、+1244/-96)
- 読む範囲: コードは Go の 5 files(+81/-25)で、本体は `internal/org/split.go`、`internal/org/feature.go`、`internal/cli/org.go`。`prompts.go` と `spawn.go` は doc コメントだけ。leader の雛形 `internal/org/prompts/leader.md`(+16/-5)。テストは 4 files(+408/-3)で、`split_test.go`、`feature_test.go`、`prompts_test.go`、`internal/cli/org_feature_test.go`。文書は `/org` と `/director` の skill の 4 写しずつ(8 files)、README、仕様、根拠の表。残りは tech-debt、計画、後工程の報告 5 本(cross-review は triage)、insight event
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → 読み込み(S1) → タスクと雛形(S2、S2b) → 文書(S3) → 記録(S4) → 後工程と、その中の直し → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行の所見を直す系列の 5 本目(F5)。4c は #250(F11)
- F5: 試行(stage 9)で、`ralph org start --plan` は leader のタスクに機能の節だけを入れた。分割計画の `## 目的` に書いた座席の決まり(implementer は opus、reviewer は sonnet、`docs/tech-debt/README.md` と予約の外は編集しない)は、どちらの leader にも届かなかった。計画の Risks は、試行の doctor-escape の reviewer が opus で動いたことを、承認の前に立てる reviewer と同じ形として挙げる
- 前文: ヘッダのあとから `## Features` の見出しの前までの `## ` の節。見出しの行を含み、フェンスのコードブロックは囲みの行ごと残し、前後の空行を詰める。試行の分割計画では `## 目的` から「予約の外のファイルは編集しない」の行まで(ファイルの 6〜15 行)
- 直し: `parseSplitPlan` が前文を `SplitPlan.Preamble` に読む。`featureLeaderTask` が、機能の本文の前に 1 行の前置き(`featurePreambleLead`、「全機能の座席に適用する決まり」で始まる)と、前文をバッククォートのフェンスで囲んだブロックを入れる。フェンスは前文の最長のバッククォートの並びより 1 つ長く、最低 3。前文が空白だけなら何も足さず、タスクは前とバイトで同じ。前文の中の `{{ORG_ID}}` は、role prompt の置き換えが 1 回の走査なので置き換わらない
- leader の雛形の手順 1・3・5: 手順 1 で前文を機能の計画の `## 全機能の座席に適用する決まり` の節に囲みの行ごと原文のまま写す。手順 3 で承認の前に立てる reviewer にも手順 5 の driver と model を使う。手順 5 で driver と model を spawn の引数に使い、ほかの決まりは TASK の本文(2,000 字まで)に写さず計画の節を読ませる
- ralph は leader がこの手順に従ったかを確かめない(tech-debt に行がある)。ヘッダの自由文、`## Features` の中の最初の `### ` より前の行、`## Features` より後の節は、今までどおりどの leader にも渡らず、読み込みも拒否しない(既存の分割計画を読めるままにする)
- 記録に残る `fix/seat-wip-checkpoint` を `#250` に直した。この PR 自身は 4c と同じく、merge の前のブランチ名 `fix/split-preamble` で書き、番号は次の PR が書く
- 戻し方: この PR を revert する。分割計画の書式も承認の digest の規則も変えていないので、台帳の分割計画はどちらの binary でも読める

## 2. 読む順(commit ごと)

1. 0c1c0ca5 計画、7a4d495d 計画の追記
   - Objective、Scope、Non-goals、Design decisions(D1〜D5)、AC1〜AC9 を読む。承認の digest は `f0735c9a1ac6`。S2 のあとの consult の指摘 3 件(手順 3 の reviewer、囲みの行ごと写す、前文が履歴に入る)を入れて、`f6a95e46e01d` から承認し直した(7a4d495d)。以降、本文は変えていない(Progress と AC の checkbox だけが動く)
   - D1: 前文の範囲。tech-debt の F5 の行の契機の欄が書く範囲と同じで、試行の分割計画がそのまま効く。専用の節だけを渡す案は分割計画の書き直しと、節の名前の書き間違いで黙って落ちることが問題になる
   - D2: 前置きの 1 行のあと、フェンスで囲む。前文の `## 目的` などが、leader の role prompt の節(`## スター型トポロジのルール` など)と並んで読めないようにするため
   - D4: `## Features` より後の節とヘッダの自由文は拒否しない。拒否すると、台帳にある既存の分割計画が読めなくなる
   - D5: Codex の計画助言の指摘 1(MEDIUM)。leader から座席への TASK の本文は 2,000 字までなので、前文は TASK に写さず、機能の計画の節から読ませる。leader のタスクは role prompt のファイルの `{{TASK}}` に入り agmsg を通らないので、前文自体は 2,000 字を超えても削られない
2. 03651616 S1 読み込み(AC1、AC2)
   - `internal/org/split.go:339` と `:355`: 最初の `## ` の見出しより後で `## Features` の見出しより前(`featuresLine == 0`)の行を `preamble` に足す。足すのは見出しの行(`## Features` を除く)と、見出しのあとの本文の行。digest が読まない行を断る `rejectDigestSkippedLine`(`:326`)と `isStatusOrApprovedLine`(`:349`)は足す前にあるので、そういう行は前文に入らない。フェンスの判定(`:329`)が見出しの判定より前にあるので、フェンスの中の `## Features` は見出しにならず前文に残る。`## Features` より後の節は `featuresLine != 0` で足さない。CRLF は `:325` で `\r` を落としてから足す
   - `:610` の `finishSplitPlan` が `trimBlankLines(preamble)`(`:634`)を `SplitPlan.Preamble`(`:98`)に入れる。ヘッダのあとすぐ `## Features` なら空。`LoadSplitPlan` は 1 回読んだ同じバイトから前文と `Digest` を作るので、承認と前文のあいだに読み直しの隙間はない(self-review)
   - split.go の先頭の書式の例に `## <a section such as Objective>` を足し、渡る範囲と渡らない 3 か所を書いた
   - テスト: `TestLoadSplitPlan_TrialPreamble`(`internal/org/split_test.go:520`)が、追跡している試行の分割計画 `docs/evidence/org-trial-2026-10-20/splits/trial.md` を読み、前文が `## 目的` から「予約の外のファイルは編集しない」の行までで、題・`- Status:`・`- Approved:`・`## Features` を含まないことを見る。`TestLoadSplitPlan_Preamble`(`:564`、12 件)は境界を 1 行 1 事象で打つ。`TestLoadSplitPlan_Rejects` には、前文の中の `- Status:`・`- [x]`・閉じていないフェンスを断る 3 行を足した
   - 見ること: 前文は承認の digest がもとから覆う範囲で、digest の規則に触れていない
3. ef33170f S2 タスクと雛形と help(AC3〜AC6)
   - `internal/org/feature.go:409-412`: `featureLeaderTask` が、前文が空白だけでなければ、今の行のあとに空行、`featurePreambleLead`(`:375`)、フェンス、前文、フェンスの順で足し、そのあとに機能の本文が来る。`fenceFor`(`:423`)は最長の並び + 1、最低 3。バイトの走査だが、バッククォートは ASCII なので UTF-8 の多バイト文字の途中に当たらない
   - `leader.md` の手順 1 に、タスクの囲みを機能の計画に写す 1 文、手順 5 に、driver と model を spawn の引数に使い、ほかの決まりを TASK の本文に写さない 1 文。前置きの行と同じ語「全機能の座席に適用する決まり」を使う
   - `internal/cli/org.go:920-922` の `ralph org start` の help と `:1044` のエラーを、タスクは「機能の本文と分割計画の前文」から作ると直した
   - テスト: `TestFeatureLeaderTask_Preamble`(`internal/org/feature_test.go:581`、9 件)、`TestFenceFor`(`:642`)、`TestStartFeature_LeaderPromptHoldsPreamble`(`:668`。`fakeHerdr` で `StartFeature` を回し、2,000 字を超える前文が削られず入ること、`{{ORG_ID}}` が残ること、role prompt の既存の節が 1 つずつ見つかることを見る)、`TestRenderRolePrompt_Leader_FeatureOrgSeatRules`(`internal/org/prompts_test.go:1049`)、`TestOrgStartHelp_TaskSource`(`internal/cli/org_feature_test.go:524`)
   - 見ること: 前文のない計画のタスクを全文で比べる `TestFeatureLeaderTask` と、手順の位置を固める `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`prompts_test.go:475`)は変えずに通る
4. b10c6f25 S2b(AC5)
   - S2 のあとの consult の指摘。`leader.md` の手順 3(`:133-136`)に、承認の前に立てる reviewer も手順 5 の driver と model を使う 1 文を足した。承認の前の reviewer は手順 5 を読む前に立ちうるので、足さないと、前文の model を知らないまま立てることになる。手順 1(`:90-94`)には「囲みの行も含めて」原文のまま写す句を足した。囲みを外して写すと、前文の `## 目的` などが機能の計画の節として並ぶため
   - `TestRenderRolePrompt_Leader_FeatureOrgSeatRules` が、3 手順の句と順を固める
5. 1b65bf49、546ad029 S3 文書(AC7)
   - `/org` skill の分割計画の節(`.claude/skills/org/SKILL.md:489-506`): 例に `## 座席の決まり`(`:448`)を足し、前文が渡ること、渡らない 3 か所、leader が計画に写すこと、機能の計画はコミットされるので前文はリポジトリの履歴に入り、コミットしてはいけないことを前文に書かない、を書いた。start の手順 6 のタスクの順(`:576-580`)と Leader のサイクル(`:1214-1225`)にも前文を足した
   - `/director` skill(`.claude/skills/director/SKILL.md:465-469`)に、全機能に共通の決まりは `## Features` より前の節に書く、と 1 文。仕様の FR-4(`docs/specs/2026-10-07-org-multi-org-director.md:80`)と README の 252 行に、前文が渡ることを足した
   - 写しは `.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` の 3 組で、`/org` と `/director` の 4 写しずつがバイト一致。546ad029 は 1b65bf49 の「台帳(git が無視する場所)」を「既定では git が無視する場所」に直した
6. c407bd67、7cc48f33 S4 記録(AC8)
   - `docs/tech-debt/README.md`: F5 の行(323 行)を取り消し線と RESOLVED の注(直上の 322 行のコメント)で閉じた。新しい open の行(347 行)は、leader が前文に従ったかを ralph が確かめないこと、渡らない 3 か所に決まりを書くと黙って落ちることを書き、直す案(分割計画に ralph が読める driver と model の節を足す)を、この PR でしなかった理由(D4、D5、Non-goals)と一緒に持つ
   - `fix/seat-wip-checkpoint` を `#250` に直した。直した場所は仕様の Status の一覧、根拠の表、tech-debt の 3 か所で、仕様と根拠の表の PR の一覧は 33 本になる。この PR 自身は `fix/split-preamble` と書き、33 本に数えない。仕様の印(`[x]` 26、`[ ]` 6)と根拠の表(25 / 6)は変わらない。F5 が挙げる FR-4 はもとから `[x]`
   - 7cc48f33 は、根拠の表が #250 の状態を読んだ gh のコマンドを正確な形に直し、計画の Progress に S1〜S4 のコミットを書いた
7. 後工程と、その中の直し
   - ed5d57d6 self-review(マージ可、LOW 4)。b09b7c12 で L1〜L4 を直した: `featurePreambleLead` の doc を「Steps 1, 3, and 5」に(L1)、`/org` skill の折り返し(L2、4 写し)、tech-debt の注のコミット id の区切り(L3)、前置きの行を「原文のまま下の囲みに入れた」に(L4、定数とテスト 2 か所を同じコミットで)。c0ff8398 は折り返しで落ちた `- 承認方式:` の前の空白を戻した
   - ab89243c verify(pass、INFO 3)。3b21b51b で I1(`RolePromptVars` の doc が置き換えを「plain strings.ReplaceAll」と書くが実装は `strings.NewReplacer` の 1 回の走査。`Task` の 2 つの doc が `start --plan` の `featureLeaderTask` に触れない。どちらも base から同じ文)と I3(根拠の表の「確かめていないこと」に F5 の項目)を直した。I2(計画の印)は 8303bd52 が直した。8303bd52 は self-review と verify の insight event も記録した
   - de6ca6b2 test(pass、mutant 41 本中 38 本が赤)。33e51acc は、1 回目の `run-test.sh` で落ちた internal/cli の競合を tech-debt の 348 行にした。7a955d2c は、生き残った mutant 3 本が示した G1・G2 を埋めた: `TestFeatureLeaderTask_Preamble` に依存のある機能 `auth-ui` の行(G1)と、前文の最後の行の末尾の空白が残る行(G2)、`TestLoadSplitPlan_Preamble` に同じ形の行。コードと雛形は変えていない。18c93ff0 が test の報告に「この報告のあとに埋めた穴」を足した
   - 30068218 sync-docs(食い違いなし。この run が変えたのは報告だけ)、e9824d39 その insight event
   - b59ca7df cross-review の triage(cycle 1/2、指摘 0)。`codex exec review --base main` を read-only で打ち、指摘なしで終わった
   - `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は e9824d39 で走らせている(結果は AC9)

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | S1。`TestLoadSplitPlan_TrialPreamble`(`internal/org/split_test.go:520`)。verify が試行の分割計画の 6〜15 行と、コードの範囲を突き合わせた |
| AC2 | S1。`TestLoadSplitPlan_Preamble`(`split_test.go:564`、12 件)、`TestLoadSplitPlan_Rejects` の前文の 3 行。フェンスの中の `## Features`、`## Features` より後の節、前後の空行、空の前文、CRLF、`## Features` の末尾の空白を打つ |
| AC3 | S2。`TestFeatureLeaderTask_Preamble`(`internal/org/feature_test.go:581`)、`TestFenceFor`(`:642`)。4 連 → 5 連、3 連のフェンス → 4 連、チルダと inline code → 3 連、空白だけ → 何も足さない、依存のある機能(7a955d2c) |
| AC4 | S2。`TestStartFeature_LeaderPromptHoldsPreamble`(`feature_test.go:668`)。2,000 字超の前文が削られず、`{{ORG_ID}}` が残る。mutant P1(`{{TASK}}` を先に置き換える)が赤になったのはこのテストだけ |
| AC5 | S2、S2b。`TestRenderRolePrompt_Leader_FeatureOrgSeatRules`(`internal/org/prompts_test.go:1049`)。手順の位置を固める `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`:475`)は diff に入らず通る |
| AC6 | S2。`TestOrgStartHelp_TaskSource`(`internal/cli/org_feature_test.go:524`)、`TestOrgStartPlan_FlagCombinationsRefused` の「a task argument」の行 |
| AC7 | S3。verify が 4 写しずつの `cmp`、`check-skill-sync.sh`(14 skill)、`check-sync.sh`(DRIFTED 0)を見た。「`## Features` の外の節は自由に書ける」の古い文が残っていないことを、sync-docs が grep で見た |
| AC8 | S4。verify が取り消し線と RESOLVED の注、注が挙げるテスト 7 本の実在、`fix/seat-wip-checkpoint` が archive と過去の報告のほか残っていないこと、PR の一覧 33 本と集合の一致、印 26/6 と 25/6 を数え直した。sync-docs も数え直した |
| AC9 | 未定(run-verify の結果を待つ)。`GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は e9824d39 で走らせている。ほかに、`run-test.sh` が 2 回目で rc 0(1,447 秒)、go1.25.8 の org と cli のテストが rc 0、go1.26 と go1.25.8 の `go vet` と go1.25.8 の `go build ./...` が rc 0。計画の AC9 の checkbox は `[ ]` のまま |

テストの強さ(test の報告より):

- 新しいテスト 7 本と、行や確かめを足した既存のテスト 3 本は、go1.26 で `-count=3`(RUN 321)と `-race` を通った。org と cli の package 全体は go1.25.8 でも通った
- herdr・agmsg・osascript・gh・tmux・claude・codex などを、呼ばれたら記録して終わる canary にした PATH と空の HOME で 12 本を打ち、外への呼び出しは 0 回だった
- mutant は 41 本のうち 38 本が赤になった。生き残った 3 本(G1 依存のある機能だけに出す、G2・G3 前文を `strings.TrimSpace` する)は、7a955d2c の行で 1 本ずつ赤になった
- テストの diff で消えた行(コメントを除く)は、cli の `TestOrgStartPlan_FlagCombinationsRefused` の「a task argument」の期待 1 行(新しい文を足した期待に置き換え。古い期待の部分は今のエラーにも含まれる)と、`TestLoadSplitPlan_ParsesFeatures` の doc コメント 2 行だけ。`git diff -U0` で数えた

## 4. 確かめていないこと

- 実機では打っていない(計画の Non-goals)。本物の herdr と leader の座席で、leader が前文を機能の計画に写すか、前文の driver と model で座席を立てるかは見ていない。Go のテストは leader の role prompt の `## タスク` の節まで。ralph は spawn の `--driver` と `--model` を前文と突き合わせない(tech-debt の 347 行)。モデルの receipt が記録するのは、spawn が命じた model と座席が報告した model で、前文が求めた model ではない
- 前文の決まりは、per-plan では人が機能の計画を承認するときに見られるだけで、検査はない。承認の前の reviewer は、手順 3 の 1 文に従うかどうかが leader 次第
- 渡らない 3 か所(ヘッダの自由文、`## Features` の最初の `### ` より前の行、`## Features` より後の節)に決まりを書くと、今までどおり黙って届かない。skill に書いただけで、読み込みは拒否しない
- 前文は機能の計画に写されるとコミットされ、機能の PR と計画の archive に入る。`/org` skill に書いたが、コミットしてはいけないことが前文に書かれていても、ralph は止めない
- 既存の分割計画で前文を背景のつもりで書いていた作者には、渡る中身が増える。計画の Risks は release のノートに書くとしている。repo に CHANGELOG はなく、書くのは release の仕事
- 7a955d2c のあとに打ち直したのは、前文に関わるテスト(go1.26 と go1.25.8)と `internal/org` の全体(383 秒)。`./scripts/run-test.sh` 全体と internal/cli は、その tree で打ち直していない。mutant は生き残りの 3 本だけで、ほかの 38 本は打ち直していない(テストの変更は行を足しただけ)
- 1 回目の `run-test.sh` で、internal/cli の `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` が 1 件落ちた。この diff が触れない前からの SIGTERM の競合で、この PR では直していない。単独の `-count=20` と 2 回目の `run-test.sh` は通り、tech-debt の 348 行に原因と直し方がある
- run-verify の結果は未定(AC9)。結果が出たら、AC9 の行を書き換え、計画の AC9 の checkbox を付ける
- 記録に残る `fix/split-preamble` を、この PR の番号に置き換えること。次の PR でする。置き換え先は、仕様の Status の一覧の F5 の行(24 行)、根拠の表の 16 行・24 行・103 行・113 行、tech-debt の 322 行の RESOLVED の注の頭と 323 行の結びの括弧。置き換えると PR の一覧は 34 本になる
- Linux では、手元で走らせていない(CI が見る)
