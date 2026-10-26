# Walkthrough: leader-spawn-pr-rules(通しの確かめの所見 RC2 と RC5。機能の org の leader が座席の spawn に `--scope` を渡し、PR を `/pr` skill の決まりで書く)

- Date: 2026-10-26
- Plan: docs/plans/active/2026-10-26-leader-spawn-pr-rules.md(/pr で archive に移す)
- PR: ブランチ `fix/leader-spawn-pr-rules` の PR(番号はまだない。merge のあとは `gh pr list --state merged --head fix/leader-spawn-pr-rules` で引ける)
- Branch: fix/leader-spawn-pr-rules(この walkthrough の前の HEAD df711495。12 commits、18 files、+759/-62)
- Base: 619cf5f8(origin/main、「Merge pull request #257」)
- 読む範囲: コードは 1 commit(efddd301)に入っていて、Go の 2 files は `internal/org/feature.go`(+14/-5)と `internal/org/prompts/leader.md`(+19/-5)、テストの 2 files は `internal/org/feature_test.go`(+72/-10)と `internal/org/prompts_test.go`(+25/-1)。`/org` skill の 4 写しは各 +16/-8(合計 +64/-32)。記録は tech-debt(+7/-5)、仕様(+4)、根拠の表(+4/-4)。残りの 550 行は計画(+148)、後工程の報告 5 本(+397。cross-review は triage)、insight event(+5)で、追加 759 行の 72% にあたる
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件。記録だけの修正の例外は使っていない(triage の commit のあとに追記の commit はない)

```
何を直すか → 計画 → feature.go の 1 行(S1) → leader の雛形(S1・S2) → テスト(S4) → /org skill(S3) → 記録(S5) → 後工程 → 設計の判断 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 通しの確かめ(PR #257、`docs/evidence/org-recheck-2026-10-24.md`)の所見 RC1〜RC12 のうち、release の前に直すとユーザーが選んだ RC2 と RC5 だけを直す。残りは release のあと
- RC2: leader の雛形の「機能ごとの org」の手順 5 は、座席の spawn に `--cwd .` と `--model` だけを書き、`--scope` を書かない。既定の permission mode `autonomous` では、`ralph org spawn` は `--scope` も `--reserve` もない座席を断る(`autonomousScopeGateErr`、`internal/org/spawn.go:1693`)。確かめでは、雛形どおりの最初の spawn が implementer と reviewer の 4 回とも断られ、leader が自分で `--scope` を足して打ち直した
- RC5: 機能の org の PR(#255、#256)の題と #255 の本文が英語だった。この repo の `/pr` skill は手順 5 で題と本文を日本語にすると決めているが、leader の雛形は「`/pr` skill の雛形に合わせる」とだけ言い、言語の決まりは `template.md` でなく `SKILL.md` の手順にある
- 直し RC2: `featureLeaderTask`(`ralph org start --plan` が leader に渡すタスクを作る関数)が `- 予約したパス:` の行に、座席の spawn に付ける `--scope` の引数を、`shellQuote` で 1 語に引用して書く。leader の雛形は、implementer と reviewer の spawn にその引数を書き換えずに付けさせる
- 直し RC5: 雛形の手順 14 が、`/pr` skill があればその手順(`.claude/skills/pr/SKILL.md`。Codex では `.agents/skills/pr/SKILL.md`)のうち題と本文の決まりと `template.md` に合わせさせる。`/pr` skill そのものは実行しない(PR を作ったあと leader の cwd の worktree を消すため)
- `/org` skill の同じ向きの 2 か所も直した。4 写しは同じバイト列
- 実機では動かしていない。leader が引数をそのまま写して打つこと、その spawn が検査を通ること、PR を日本語で書くことは、テストで固定した文と行の形からの推論で、どこでも確かめていない。このため仕様の FR-4 と受け入れ条件 5 は `[ ]` のままで、「直した。次の実行で付け直す」の印(2026-10-26)を足した
- release のノートに書くこと(計画の Rollout): 「leader の雛形が座席の spawn に `--scope` を渡し、PR の題と本文を `/pr` skill の手順に合わせる」。雛形は binary に入るので、新しい binary で spawn した leader から効く。コードの commit(efddd301)の件名が `fix:` なので、自動の changelog に出る(`.goreleaser.yml` が除くのは `docs:`・`test:`・`chore:`)
- 戻し方: この PR を revert する

## 2. 読む順(ファイルごと)

commit は、70cc1d77 計画、efddd301 コードとテスト、c654d596 skill、7d51236d 記録、そのあとが後工程の report と計画の印。下の 1〜5 は efddd301・c654d596・7d51236d の差分、6 は残りの commit。

1. `internal/org/feature.go`(efddd301、+14/-5)
   - 変更の本体は 1 行。`featureLeaderTask`(`:401`)の `- 予約したパス:` の行(`:412`)が、`fmt.Sprintf("- 予約したパス: %s(座席の spawn には --scope %s を付ける)", reserve, shellQuote(reserve))` になった。`reserve` は `:406` で `f.Reserve` を `", "` でつないだ値。残りは doc comment で、`:379-383` の言い換えと、`:394-400` の新しい 7 行
   - 形は `- 台帳:` の行(`:416`)と同じ。表示用の素の値を 1 回、`shellQuote` で引用した値を 1 回、括弧の中に書く。leader の読み方が `--state-dir` と `--scope` でそろう
   - `shellQuote` で引用する理由(D6)。予約のパスの検査 `normalizeReservePath`(`internal/org/reserve.go:209`)が断るのは、空、絶対パス、カンマ・空白・制御文字、`..` だけで、`$`、バッククォート、単一引用符、二重引用符は通す。つないだ値は `, ` で空白を含むので、2 つ以上のパスは引用なしなら shell が分ける。leader に「二重引用符で囲む」と書かせる案では、`app/routes/$slug.tsx` の `$slug` やバッククォートの中を shell が展開し、誤った担当範囲を渡したり、パスの中のコマンドを leader の権限で走らせたりしうる(Codex の計画助言、HIGH)。`normalizeReservePath` を厳しくする案は、`$` を含む実在のパスを断るので採らなかった
   - 見ること: `shellQuote`(`:448`)は変えていない。単一引用符を `'\''` にして全体を単一引用符で包む。`- 予約したパス:` の行を解析するコードはない(読むのは雛形とテストだけ。計画の Assumptions と self-review が grep した)。`--scope` の値の形は spawn の検査に関わらず、`{{SCOPE}}` の置換(`internal/org/prompts.go:112`)と manifest の `scope=%s`(`spawn.go:1711`)が読むだけで、`, ` を含む値でも壊れない。空の `--scope ''` を書く経路もない(`parseSplitReserve` が空の値を、`finishSplitPlan` が `Reserve` のない機能を断る)
2. `internal/org/prompts/leader.md`(efddd301、+19/-5)
   - ミッションの 2(`:24-27`): spawn の説明の末尾に、`--scope` で担当範囲を渡すこと、`autonomous` では必須で断られること、機能ごとの org ではタスクの `- 予約したパス:` の行の括弧にある引数を使うこと、手順 5 を見ることを足した。分割計画を使わない org の leader も同じ spawn を打つので、ここにも書いた(D5)
   - 手順 3(`:141-143`): 承認の前に reviewer を spawn するときの flag を、手順 5 と同じにする(`--cwd .`・`--model`・`--scope`)
   - 手順 5(`:157-163`): implementer と reviewer の両方に、`--cwd .`・`--model` と並べて、括弧の中の `--scope` の引数を「書き換えずに付ける(ralph が shell 用に引用したものなので、引用符を足したり外したりしない)」。`autonomous` では `--scope` のない spawn が断られること、implementer の雛形は scope の中のパスだけを `git add` し(`implementer.md:21`)、reviewer の雛形は report と gate の生成物を scope の外でも書いてよい(`reviewer.md:10-12`)ので、どちらの `--scope` も予約にそろえる、という理由まで書く(D2)
   - 手順 14(`:285-289`): PR の文を「`/pr` skill の雛形に合わせる」から、`/pr` skill があればその手順(`.claude/skills/pr/SKILL.md`。Codex では `.agents/skills/pr/SKILL.md`)のうちタイトルと本文の決まり(言語、タイトルの prefix など)と、本文の雛形 `template.md` に合わせる、に替えた。「`/pr` skill そのものは実行しない」はそのまま。名指す先が `template.md` でなく `SKILL.md` なのは、日本語の決まりが 4 つの `SKILL.md` の手順 5(`:31`)にあり、`template.md` の 2 写しにはないため(D3)
   - 見ること: 手順 4 と、「機能ごとの org」の導入(`:61-70`。予約の外に書く手順を挙げる)は変えていない。導入が予約の外に書いてよいとする手順(計画、reviewer のレポート、report、計画の戻し)と、手順 5 の「reviewer は scope の外でも書いてよい」が矛盾しないこと。手順 3 の `--cwd .`・`--model`・`--scope` が、`/org` skill の手順 2 の括弧「`--model` と `--scope` は上と同じにする」と食い違わないこと(skill の手順 2 は `--cwd .` を昔から書いていない)
3. `internal/org/feature_test.go`(+72/-10)と `internal/org/prompts_test.go`(+25/-1)(efddd301)
   - `feature_test.go` の `TestFeatureLeaderTask`(`:516`): 期待値を新しい行の形にした。既存の 3 つの事例(2 パスと本文つき、台帳の空白、台帳の単一引用符)の期待値を直し、予約の単一引用符(`it's/auth.md` が `'it'\''s/auth.md'` になる)の事例を 1 つ足した
   - 新しい `TestFeatureLeaderTask_ScopeIsOneShellWord`(`:593-640`、AC1b): 7 つの予約(1 つ、複数、`$`、バッククォート、単一引用符、二重引用符、`$HOME`・`'`・`"`・バッククォート・`\`・`*` を混ぜた 5 つ)で、タスクの `- 予約したパス:` の行から括弧の中の引数を切り出し、`/bin/sh -c "set -- <引数>; printf '%s|%s' \"$#\" \"$1\""` に読ませる。結果が `1|<パスを ", " でつないだ値>` であること、つまり展開も分割も起きないことを見る。fixture は先に `normalizeReservePath` がそのまま通すことを `t.Fatalf` で確かめるので、通らない入力で緑になることがない
   - `prompts_test.go` の `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`:481`): 新しい pin が 6 つ。ミッションの 2 の文(`missionScope`、`:491-495`)、手順 5 の 3 句(`:562-565`。書き換えずに付ける、`autonomous` で断られる、`git add` と reviewer の理由)、手順 3 の 1 句(`:566`)、手順 14 の `/pr` skill の 1 句(`:569`)。タスクの行の形も pin する(`:553-559`)ので、`featureLeaderTask` が行の形を変えると、雛形のテストも落ちる。既存の pin(`--id` の文、手順 14 の `gh pr create` の 3 句、ミッションの「implementer 1 席と reviewer 1 席を既定」)は残した
   - mutant は `go test -overlay` で scratchpad の写しに差し替えて打ち、追跡しているファイルは変えていない(test の report)。タスクの行は 6 つ(`shellQuote` を外す、`"…"` で囲む、前の行の形に戻す、最初のパスだけ引用する、`,` でつなぐ、単一引用符を `'\''` にしない)、雛形は 6 つ(base の雛形、ミッションの文、手順 3、手順 5 の `--scope` の文、手順 14 を前の文に戻す、手順 5 の「書き換えずに」を「引用符で囲んで」にする)。12 個とも落ちた
   - 見ること: 追跡しているテストが読ませる shell は `/bin/sh` だけで、`$(...)`、`~`、`!`、先頭か末尾の単一引用符は fixture にない(test の LOW)。probe で 9 事例 × `/bin/sh`・zsh・bash の 27 通りが一致したので、固定しなかった。`shellQuote` が全部を単一引用符で包むので、`$` とバッククォートの事例が通る実装はこれらも通す
4. `/org` skill の 4 写し(c654d596、各 +16/-8)
   - `.claude/skills/org/SKILL.md` の 2 か所。「org の終わり方」の手順 6(`:1002-1010`)は、PR の文を雛形の手順 14 と同じ向きにした(`/pr` skill の手順の題と本文の決まりと `template.md` に合わせ、`/pr` skill は実行しない)。leader のサイクルの手順 2(`:1306-1313`)は、`--scope` を渡すこと、機能ごとの org ではタスクの予約の行の括弧の引数を書き換えずに付けること、承認の前に立てる reviewer も `--model` と `--scope` は同じにすることを足した
   - 写し 3 つ(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`)は `scripts/sync-skills.sh` でそろえた。4 つとも `md5 -q` が同じ値(この walkthrough を書く前に打った)。`scripts/check-skill-sync.sh` は「14 skill(s) in lock-step」、`scripts/check-sync.sh` は IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0(verify と sync-docs の記録)
   - 見ること: skill の `spawn` の行(`:208`)の「`--scope` は担当範囲の説明。autonomous では必須」と、新しい文が合うこと。`start --plan` の説明(`:651-670`)には `- 予約したパス:` の行の `--scope` の引数の説明を足していない(下の 5)。この PR は skill の文を固めるテストを足していない
5. 記録(7d51236d、0309acf8、8e6bec75)
   - tech-debt(`docs/tech-debt/README.md`): RC2(`:364` の RESOLVED の注、`:365` の行)と RC5(`:368` の注、`:369` の行)を、取り消し線の RESOLVED にし、ブランチ名 `fix/leader-spawn-pr-rules` を書いた。注は、直したこと、テストが固定したこと、していないこと(実機)を分けて書く。どちらの行も 5 欄。sync-docs(8e6bec75)は、この PR が Trigger に当たった 3 行に引き継ぎの注を足した。RC4 の行(`:367`)、ファイルの大きさの行(`:357`)、org-feature-worktree の小さな所見の行の (g)(`:197`)。どれも RESOLVED にしていない。RC4 は計画の Non-goals で、release のあとに回したので払っていない。(g) は f1b0352e で直っていたので、そう書いた
   - 仕様(`docs/specs/2026-10-07-org-multi-org-director.md`): Status に新しい段落(`:37`)で RC2 と RC5 を直したことを書き、FR-4 に印(2026-10-26)(`:90`)、受け入れ条件 5 に印(2026-10-26)(`:176`)を足した。FR-4(`:80`)と受け入れ条件 5(`:173`)の `[ ]` は動かしていない。`[ ]` は base も HEAD も 5、`[x]` は 27
   - 根拠の表(`docs/evidence/org-spec-evidence-2026-10-21.md`): RC2(`:53`)と RC5(`:56`)の「直す PR」と tech-debt の行の欄を、ブランチ名と RESOLVED の行に替えた。FR-4(`:76`)と受け入れ条件 5(`:93`)の行に、直したが `[ ]` のままという文を足した。判定の数(`:105`、`[x]` 26 と `[ ]` 5)は変わらない
   - FR-4 と受け入れ条件 5 が `[ ]` のままである理由(D4): 根拠の表の印の規則(`:5`)は、外の道具(herdr、codex、claude)に頼る項目の `[x]` に、その道具で動かした実機の記録を 1 つ求める。この PR は実機で動かしていないので付けられない。2026-10-22 の F4 の印が引いた `docs/evidence/seat-spawn-close-2026-10-22.md` のような記録が、次の実行で要る。印の文は、テストが固定したこと(行の形、`/bin/sh` が引数を 1 つの語に読むこと、雛形の文)と、確かめていないこと(leader がそのとおりに打つこと、spawn が検査を通ること、PR を日本語で書くこと)を分けて書く(self-review F1 の直し、0309acf8)
   - 見ること: 仕様の Status の「開いたまま残る」の 7 つ(RC1、RC3、RC4、RC6、RC9、RC10、RC11)が、根拠の表で「直す PR」が「未定」の 7 行と合うこと。RC8 と RC12 は repo の欠陥でないので行がない(self-review F2 の直し)
6. 後工程の report(43dc55e4、2947c524、1e2cf34f、8e6bec75、df711495)と計画の印(f9fe07ea、ba3c877c)
   - self-review(43dc55e4。Merge OK、LOW 2): F1 記録の「テストで確かめた」が、テストの中身より強い。F2 Status の段落が RC12 を開いたままに数えていた。どちらも 0309acf8 で、verify の前に cycle 1 の中で直し、report に追記した(findings と Merge の行は変えていない)
   - verify(2947c524。pass、LOW 3、情報 2): 6 つの AC の照合と、`run-static-verify.sh` rc 0、`go vet` を 2 つの版で rc 0。`shellQuote` を python に写した probe の 10 組(`a/$(id)/` や `-x/, docs/日本語.md` を含む)が `/bin/sh` で全部 `1|<元の値>`。LOW は tech-debt の 3 行の引き継ぎの注の欠け(V1〜V3)で、sync-docs が直した
   - test(1e2cf34f。pass、HEAD 2947c524): `./scripts/run-test.sh` が rc 0(904 秒、shell 44 本、Go 8 パッケージ)。`go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` が go1.26.0 で rc 0(919 秒)、`GOTOOLCHAIN=go1.25.8` で rc 0(448 秒)。名指しの 3 本は両方の版で `-count=3` の 9 回すべて PASS。12 個の mutant が全部落ちた
   - sync-docs(8e6bec75): 古い形の記述を AGENTS.md、README、recipes、quality、rules、`/director` skill、`/org` skill の残りの節で探し、0 件。tech-debt の 3 行に引き継ぎの注を足した。`./scripts/run-verify.sh` が rc 0。3 つの見送り(下の 5)
   - cross-review(df711495。triage、cycle 1/2、指摘 0): `codex exec review --base main` を read-only で打ち、rc 0。reviewer の判定は、実行して直すべき退行なし。4 写しの一致と、7 つの予約の shell 引用の probe が通ったと報告した。read-only の sandbox が Go の build の一時ディレクトリを書けず、codex の `go test` は始まらなかったので、テストは /test の結果による
   - 計画の印: f9fe07ea が AC1〜AC6 の checkbox、ba3c877c が Review・Verification・Test の artifact に印を付けた。計画の本文は承認のあと変わっていない(`git diff 70cc1d77 HEAD` で動いたのは AC の checkbox と Progress だけ)。承認の digest は 7fb9a3d2e3cf で、HEAD で `./scripts/plan-visual.sh digest` が返した値と同じ
   - insight event は `docs/insights/events/2026-10-26-leader-spawn-pr-rules.jsonl` の 5 行(self_review、verify、test、sync_docs、cross_review。どれも cycle 1)。UTC の日付がどれも 2026-10-26 だったので、ファイルは 1 つ

## 3. 設計の判断(計画の D1〜D6)

- D1 1 本の PR: RC2 と RC5 はどちらも leader の雛形と `/org` skill の同じ節を直し、同じテストで固定する。分けると同じファイルを 2 回触り、記録の更新も 2 回になる
- D2 `--scope` の値は予約: implementer も reviewer も `- 予約したパス:` の値をそのまま渡す。確かめでは hooks-matcher の reviewer が `review hooks-matcher; writes docs/reports/ only` を使ったが、reviewer の雛形が report の書き込みを scope の外でも許すので、説明を足す必要はない。値を 1 つの形に決めると、leader が自分で考える余地が減る
- D3 RC5 は雛形の文で直す: 言語の決まりを `template.md` に移す案は、`template.md` の 2 写しと scaffold の project の雛形を変え、project ごとに言語が違う場合にも当たる。雛形に「`/pr` skill の手順の題と本文の決まりに従う」と書けば、project の `/pr` skill が決めた言語にそのまま従う
- D4 印は `[ ]` のまま: 上の 2 の 5 のとおり
- D5 ミッションの 2 にも書く: 分割計画を使わない org でも leader は同じ spawn を打ち、`autonomous` で断られる。雛形の 1 文で済む
- D6 引用は ralph が作る: 上の 2 の 1 のとおり。Codex の計画助言(HIGH)を承認の前に入れた。計画の Progress によれば、consult の相談(3 点)も承認の前に入れた
- 計画の承認は、ユーザーの決まり「作成したプランは自動承認します」による(2026-10-26)

## 4. AC の根拠

| AC | 根拠 |
|----|------|
| AC1(RC2) | efddd301。`featureLeaderTask` の行(`feature.go:412`)、雛形のミッションの 2・手順 3・手順 5。`TestFeatureLeaderTask` と `TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`prompts_test.go:553-559` が行の形を固定)。verify は `autonomousScopeGateErr` を読み、値の形を見ず、空のときだけ断ること、implementer と reviewer の座席は `--reserve` を受けない(「--reserve on the leader seat only」)ので、`--scope` だけが検査を満たす手段であること、`--scope` は pflag の `StringVar` で値がそのまま入ること(値が `-` で始まっても `--flag arg` の形なら次の引数が値)を確かめた。実機の spawn は打っていない |
| AC1b | `TestFeatureLeaderTask_ScopeIsOneShellWord`(`feature_test.go:593`)。mutant の `shellQuote` を外す写しは、2 パスの予約で `$#` が 2、`$slug` が空に展開、バッククォートの `id` が実行され、単一引用符で sh が exit 2。`"…"` で囲む写しは、`$slug`、バッククォート、`say"hi"/` で落ちる。事例の ``a/`id`/`` は mutant の下で本物の `/bin/sh` が `id` を実行したが、害のないコマンドで、今後も害のないものにする(test の report) |
| AC2(RC5) | efddd301。手順 14(`leader.md:285-289`)。名指した `pr/SKILL.md` と `pr/template.md` の 8 ファイル(root と `templates/base/` の `.claude` と `.agents`)は実在する(verify)。4 つの `SKILL.md` の手順 5(`:31`)に「PR title and body must be written in Japanese」があり、`template.md` の 2 写しには言語の決まりがない。雛形の手順 14 を前の文に戻す mutant(p4)が落ちる |
| AC3 | c654d596。`/org` skill の 2 か所が雛形の手順 3・5・14 と同じ向き。4 写しが同じバイト列で、`check-skill-sync.sh` が 14 skill で通る |
| AC4 | 新しい pin 6 つ。base の雛形(p0)に掛けると新しい pin 6 つだけが落ち、既存の pin は落ちない。文を 1 つずつ戻す mutant(p1〜p5)が、対応する pin で落ちる。計画は「手で 1 回ずつ戻す」としたが、追跡しているファイルを変えないよう `go test -overlay` で差し替えた |
| AC5 | 7d51236d(と 0309acf8 の直し)。tech-debt の RC2 と RC5 の行が取り消し線の RESOLVED でブランチ名つき、仕様の FR-4 と受け入れ条件 5 が `[ ]` のまま(`[ ]` 5、`[x]` 27 が base と HEAD で同じ)、根拠の表の RC2 と RC5 の「直す PR」がブランチ名。verify が欄の数(tech-debt の 2 行が 5 欄、根拠の表は base と同じ)と、開いた 7 行の一致を見た |
| AC6 | slice ごとの `./scripts/run-verify.sh`(S2 と S5 のあと rc 0)、verify の `run-static-verify.sh` rc 0 と `go vet` 2 版 rc 0、test の `run-test.sh` rc 0(904 秒)と go1.26.0・go1.25.8 の `go test` rc 0、sync-docs の `run-verify.sh` rc 0(07:27〜07:48 UTC、`secret-scan-branch` が clean)。test の HEAD 2947c524 のあとに変わったファイルは、insight event、計画、report 3 本、tech-debt だけで、Go とスクリプトは含まない(`git diff 2947c524 HEAD --name-only` で、この walkthrough を書く前に確かめた) |

## 5. 確かめていないこと

- 実機の leader: 新しい binary の leader が、手順 5 のとおりに `--scope` の引数を書き換えずに写して spawn を打つか、その spawn が `autonomous` の検査を通るか、PR の題と本文を日本語で書くか。計画の Non-goals と R3 で、次の実行で確かめる。FR-4 と受け入れ条件 5 の `[ ]` は、そのときの記録で付け直す
- spawn の検査は、コードを読んだだけ(verify)。`ralph org spawn` に `, ` でつないだ値を打ったことはない。ただし `--scope` を読むコードは `{{SCOPE}}` の置換と manifest の `scope=%s` の書き込みだけなので、値の形は動きに関わらない
- 手元は macOS(Darwin 25.5.0、arm64)だけで、Linux では走らせていない(CI が見る)。AC1b のテストは `/bin/sh` を使い、macOS では bash だが、Linux の CI では dash でありうる。probe の zsh と bash は手元だけ。単一引用符で包んだ語は POSIX の `sh` でどれも 1 語に読まれるので、落ちる理由は見当たらない
- AC1b の fixture に、`$(...)`、`~`、`!`、先頭か末尾の単一引用符はない(test の LOW 3 点のうち 2 点。もう 1 点は、`/bin/sh` が構文の誤りで終わるとそこで `t.Fatalf` が止まり、後ろの事例を報告しないこと。落ちる側の動きで、判定は変わらない)。probe で通ったので tech-debt にしていない
- Codex の leader が `.agents/skills/pr/SKILL.md` を読むこと。ファイルがあることだけを verify が確かめた。scaffold した project の `/pr` skill が日本語の決まりを持つことは、4 つの `SKILL.md` の手順 5 の文面を読んで確かめたが、scaffold した project を作って読ませてはいない
- `/org` skill の `start --plan` の説明(`:651-670`)に、`- 予約したパス:` の行の `--scope` の引数の説明を足していない(verify の情報、sync-docs が見送り)。誤った文はなく、leader が読む手順(leader のサイクルの手順 2)と雛形に書いてある。足すと 4 つの写しで `:1002` より前に行が入り、報告が引く行番号がずれる。この節を次に編集するときに足せる
- 既知の不安定: S3 の slice で `go test ./internal/org/... ./internal/cli/...` を打った 1 回目に、`TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing/SIGTERM` が load average 約 12 の下で 10 秒の待ちを切らして落ち、2 回目は通った(この PR は regate のコードを変えていない)。/test の 3 回の全体の実行(load average は 108 まで上がった)では落ちなかった。tech-debt にこのテストの行はなく、足していない(test の report の判断のまま。もう一度落ちたときに足すかを決める)
- パッケージ全体の statement coverage は計っていない。`featureLeaderTask` と `shellQuote` は関数単位で 100.0%。雛形の文は coverage に出ず、mutant で確かめた
- 開いたままの tech-debt: RC1、RC3、RC4(引き継ぎの注あり)、RC6、RC9、RC10、RC11 の 7 行。skill・雛形・コードの直しを要し、この PR は直さない。仕様の `[ ]` はほかに FR-3(leader への ALERT)、FR-10、ツール非依存(Codex の leader)
- archive のあと、この計画の `docs/plans/active/` の path を指す参照が切れる。HEAD(この walkthrough の前)で 18 行、7 ファイルにある。`scripts/archive-plan.sh` が書き換えるのは tech-debt の参照だけ(`/pr` skill の手順 8、`pr/SKILL.md:57`)で、tech-debt の 7 行がその対象。残る 11 行(仕様の Status 1 行と、report 5 本の 10 行)は `active/` のまま切れる。この walkthrough の `Plan:` の行も同じになる。仕様の `:35` と根拠の表の 3 行目に、archive に動いたあとも `active/` のまま残る前例があり、self-review も取り上げないと決めた
- この walkthrough を書くために打ったのは、`git` の読み取り(`diff`、`log`、`show`、`grep`)、`wc`、`md5 -q`、`./scripts/plan-visual.sh digest`。テスト、`go`、`herdr`、`gh`、network には触れていない。テストと静的検査の結果は、各 report の記録に頼り、この walkthrough では打ち直していない
