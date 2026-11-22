# Walkthrough: org-xreview-own-model

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-org-xreview-own-model.md
- Issue: #360
- Base: main ccdf6293

差分は 43 ファイル、+3,257 行、−1,835 行。そのうち約 1,260 行は計画、報告、insight の 8 ファイル。動きを決めるのは雛形 2 つ(`reviewer.md` と `leader.md`、+185 −289)と、`prompts.go` と `spawn.go` の約 25 行。ほかの Go の本番コード(`config.go`、`paneenv.go`、`init_settings.go`)はコメントと `ralph init` の問いの説明文だけ。テストは 8 ファイルで +1,271 −1,203 と大きいが、雛形の文を固定するテストを書き直したため、行数の多くは入れ替え。残りは skill の 8 写し、model-routing、recipe、README、仕様、tech-debt、`templates/base/ralph.toml` のコメント。下の順に読むと早い。

## 1. 何が問題だったか

機能の org の cross-review では、reviewer 座席が相手の CLI(implementer が claude なら codex、codex なら claude)を、座席の中から入れ子で呼んでいた。main の `reviewer.md` は、`IMPLEMENTER_DRIVER:` の行から `pick_reviewer` で相手を選び、MCP を止め、`sleep 1200` の見張りを付けて呼ぶ形だった。

issue #360 では、codex の reviewer 座席の中で、入れ子の codex が `failed to initialize in-process app-server client: Operation not permitted (os error 1)` で起動できなかった。claude の座席から呼ぶ codex は動いた。ほかに、相手の呼び出しを最大 20 分待つことと、通信が切れて incomplete になることがあった。

ユーザーの決定(2026-11-21)は、原文で次のとおり。

> 「というか、org runtimeでは、cross-review関係なく、reviewer座席がレビューを担う役割です。同じ系統でも異なる系統でも関係ありません。reviewer座席に指定されたモデルがレビューを担ってください。」

これを「reviewer 座席が自分のモデルで差分を読み直す」と読んだ。系列で扱いを変えず、相手の CLI は呼ばない。cross-review の段(TASK、上限、PR の本文の `Cross-review:` の行)は残した。issue の「report の形と leader への返し方は変えない」に合うため。

## 2. 座席の雛形に driver と model を渡す(AC4)

- `RolePromptVars` に `Driver` と `Model` を足した(`internal/org/prompts.go:71`、`:77`)。`RenderRolePrompt` が `{{DRIVER}}` と `{{MODEL}}` に置き換える(`prompts.go:146-147`)。model が空なら `ModelOrCLIDefault` で「CLI の既定」と書く
- `Spawn` の本物と dry run が `p.Driver` と `p.Model` を渡す(`internal/org/spawn.go:1120`、`:2423`)。`AgentStart` に渡す値と同じ
- reviewer の雛形の 4 行目に `- driver: {{DRIVER}} / model: {{MODEL}}` が出る(`internal/org/prompts/reviewer.md:4`)。座席が自分の CLI を推して書くより確か。director の雛形は別の関数で `{{DRIVER}}` を使う(`prompts.go:277`)ので、AC4 のテストの外

## 3. reviewer 座席の「cross-review の TASK」(`reviewer.md:164-393`)

- 冒頭(`:166-186`): 座席自身が差分を読み直して所見を出し、triage して report に書く。レビューするのは座席のモデルで、`codex exec`、`claude -p`、ほかの CLI で別のモデルを呼ばない(`:170-172`)。TASK の本文で読むのは `XREVIEW_BASE:`、`CYCLE:`、`REPORT:` の 3 行(`:175-180`)。`IMPLEMENTER_DRIVER:`、`REVIEWER_CODEX:`、`REVIEWER_CLAUDE:` の行があっても読まない(`:182-184`)ので、前の版の leader が送る TASK でも動く
- 手順 1(`:188-222`): 3 行の確かめ。欠けた行や形の違う値は、何も書かずに QUESTION。今までどおり
- 手順 2(`:224-249`): 読む前の git の状態を `-git-before.txt` に、base からの差分を `.diff` に書く。`git diff` が失敗すると `diff: failed`(後述の周回 1 の直し)
- 手順 3・4(`:250-294`): 差分と変わったファイルを自分で読んで所見を出し(正しさ、security、エラーの扱い、論理、テストの抜け、変更に特有の見落とし。HIGH/MEDIUM/LOW)、2 つの軸で triage する。表と分け方は今のまま
- 手順 5(`:295-314`): report を書く前に git の状態を `-git-after.txt` に書いて `cmp` で比べる。違えば report を書かず、`XREVIEW: git_changed` の BLOCKED を返す。比べる区間は、相手の呼び出しの前後から、読む前と report を書く前に変わった。読んでいるあいだに HEAD が動くと見ていない変更に pass が付くため、codex の計画助言の指摘で残した
- 手順 6・7(`:315-369`): report の形は今のまま。`Reviewer:` は `reviewer 座席 <seat_id>(<driver>、<model>)`(`:331`)。`- Driver: ...  Reviewer: ...` の行は `- Reviewer:` だけの行になり、`Reviewer output:` の行は `- Diff: <XREVIEW_BASE>...<HEAD の SHA>`(`:342`)に替わった。返事は `XREVIEW: pass`、`action_required`、`incomplete`
- 消したもの: 相手の CLI を呼ぶコマンド、`pick_reviewer`、MCP を止める手順、20 分の見張り、`skipped (<相手> not available)` などの skip の理由
- 座席内 fan-out(`:456-484`): 判断を担う子は座席と同じモデルで動かす。安いモデルに任せてよいのは、レビューの TASK のゲートのコマンドだけ。cross-review の 2 から 5 のあいだは、子にも worktree に書かせない(5 の比べが `git: changed` になるため)

## 4. leader、skill、recipe、文書

- leader の「cross-review」の小節(`internal/org/prompts/leader.md:1051-1328`): TASK の本文は 3 行だけ(`:1247-1249`)。台帳から implementer の driver を読むコマンドと `driver: error` の扱い、`REVIEWER_CODEX:` と `REVIEWER_CLAUDE:` の行を消した。例の TASK は 3 行(`:1256-1264`)。`XREVIEW: git_changed` の扱い(BLOCKED を上げ、戻したら送り直す、`:1302`)、送り直しの上限、上限の数え方と上限の QUESTION は今のまま。`Cross-review: ran (<report の Reviewer:>)` の決め方も今のままで、値が座席の名前になる(`:1286`、`:1322-1328`)。前の版の reviewer が返す `skipped` は、leader が今も読める(`:1288-1291`)
- 小節の外で cross-review に触れる文も直した。停滞の ALERT のときの待ち方(`:190-194`、`:774-782`)、reviewer の QUESTION の言い方(`:254`)、次へ進む条件(`:623`)、PR の本文の `Cross-review:` の値(`:876-883`)、RESULT の例(`:911`、`CROSS_REVIEW: ran (reviewer 座席 reviewer(codex、gpt-6.1-sol))`)。`### 計画助言`(`:925-1050`)は触っていない。タスクの `- reviewer の codex:` と `- reviewer の claude:` の行は計画助言が使うので残る
- skill(org と director、それぞれ 4 写し): org の「役割ごとのモデル」(`.claude/skills/org/SKILL.md:249`)、「org の終わり方」(`:1325`)、「座席内 fan-out」(`:2419`)、director の「分割計画の前の質問」(`.claude/skills/director/SKILL.md:545`)、「cross-review の上限」(`:1407`)。違う系列の目が欲しい人は `[org.role_models].reviewer` を implementer と違う driver にする、と書いた
- `docs/recipes/codex-seat-permissions.md` の「The cross-review of a codex reviewer seat」(`:333-360`)は、入れ子の CLI を呼ばない話に書き直した。「The network of a codex seat」(`:197-201`)には、座席のエージェントが外のどのホストにも出られること(宛先の許可リストはない)と閉じ方の 1 文を足した(AC8、#362 の consult の指摘)
- そのほか: `model-routing.md:85-93`、`docs/recipes/isolated-herdr-server.md:258-259`、`README.md:286`、`:292`、仕様 `docs/specs/2026-10-07-org-multi-org-director.md:145`(2026-11-22 の項目)、`internal/config/config.go:74`、`:749-773` と `templates/base/ralph.toml` のコメント。`scripts/xreview-helpers.sh` の `pick_reviewer` の doc comment は、呼び出し元から reviewer 座席を外した(コメントだけ)

## 5. 2 周のパイプライン

周回 1 は、cap 2 のうちの 1 回目。

- self-review Run 1 は MEDIUM 1、LOW 4。MEDIUM は、cross-review の返事を待つあいだの「画面が動かなくても待つ」例外に上限がなかったこと。f95a08f9 で、画面が動いているあいだだけ待ち、10 分動かなければ leader が BLOCKED を上げる形にした(`leader.md:190-194`、`:774-782`、`:1269-1273`)
- verify Run 1 の V1(`ralph init` の「変更のレビュー」の問いが、org の cross-review を「もう一方の AI」と説明したまま)は、7c7cdec5 で説明文だけ直した
- main を 2 回取り込んだ(b98b1b96 は #362 と #366、6b671276 は #363 と #370)。衝突は tech-debt と director の skill の追記で、両側を残した
- cross-review 1 回目(codex、HEAD 82090f46)の指摘は 1 件(P2)で ACTION_REQUIRED。手順 2 で `git diff` が失敗すると、手順 5 の比べを通らずに incomplete の report を書いて返していた。失敗のあいだに HEAD が動いても `git_changed` が上がらず、leader が sync-docs に進む。前の版は差分が書けなくても git の状態を比べていた

周回 2 は、cap の 2 回目。

- eff83438: `diff: failed` のとき手順 3・4 を飛ばして 5 に進み、`git: changed` なら `XREVIEW: git_changed` の BLOCKED、`git: same` のときだけ 6 で `Reviewer status: incomplete (git diff failed: <要点>)` の report を書く(`reviewer.md:241-248`)
- 7eb0179d: 文言だけの直し(fan-out の子の作業ファイルの置き場所、手順 5 の括弧)
- 18360371: `diff: failed` のあとに手順 5 のコマンドを走らせる 2 行を、bash と zsh で足した。何も変えない行は `git: same`、あいだにコミットを足す行は `git: changed`(`internal/org/prompts_xreview_reviewer_test.go:839-845` の `reviewerXreviewAfterAFailedDiff`、走らせるのは `:907-927`)。a607b72d と 5af1a67b はそのテストのコメントと変数の直しだけ
- cross-review 2 回目(codex、HEAD b8245ea1)は指摘なし。上限の 2/2 に届いた

## 6. 先に読むテスト

- `TestRenderRolePrompt_Reviewer_XreviewReviewsOnTheSeatsOwnModel`(`internal/org/prompts_xreview_reviewer_test.go:196`): 自分のモデルでレビューすると言うこと、相手を呼ぶ語(`reviewerXreviewGone`、`:52`)が 2 つの文のほかに節に出ないこと、手順 1・2・5 のコマンドが `codex` と `claude` を走らせないこと(AC1)
- `TestRenderRolePrompt_Reviewer_XreviewTaskCarriesItsLinesInTheBody`(`:249`)、`…XreviewCommandsWriteTheStateAndTheDiff`(`:765`): 3 行だけ読むこと、前の版の TASK の行があっても同じコマンドになること(AC2)
- `TestRenderRolePrompt_Reviewer_XreviewComparesTheGitStateBeforeTheReport`(`:415`)、`…XreviewCommandOutcomes`(`:876`): git の状態の比べを bash と zsh で走らせる。`diff: failed` のあとの 2 行は `:907` の `for`(AC1、AC3)
- leader: `TestRenderRolePrompt_Leader_XreviewTaskCarriesTheThreeLinesTheReviewerReads`(`internal/org/prompts_xreview_leader_test.go:424`)、`…XreviewReadsNoDriverAndNamesNoOtherModel`(`:505`)、`…XreviewHandlesEveryReply`(`:540`)(AC5)
- AC4: `TestRenderRolePrompt_Reviewer_ShowsTheSeatsDriverAndModel`(`internal/org/prompts_test.go:2177`)、`TestRenderRolePrompt_SeatTemplates_NoDriverOrModelPlaceholderLeft`(`:2211`)、`TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel`(`internal/org/spawn_role_models_test.go:334`)
- AC7: `TestOrgSkill_RoleModelsSayTheReviewerSeatReviewsOnItsOwnModel`(`internal/org/skill_xreview_test.go:663`)、`TestDirectorSkill_XreviewSaysTheReviewerSeatReviewsOnItsOwnModel`(`:703`)、`TestOrgAndDirectorSkills_SayNothingOfTheOtherModelsCrossReview`(`:726`)

## 7. 確かめたこと

- self-review は 3 周で、どれも Merge: yes(CRITICAL 0、HIGH 0。Run 1 は MEDIUM 1 と LOW 4、Run 2 は LOW 3、Run 3 は LOW 2)
- verify は 3 周(HEAD b98b1b96、7eb0179d、a607b72d)で pass。`./scripts/run-static-verify.sh` は full の範囲で rc 0。AC1 から AC8 を満たす(AC6 と AC1 の字面は下の残したもの)
- test は 3 周(HEAD 7c7cdec5、5c5ead66、5af1a67b)で pass。`./scripts/run-test.sh` は 3 回とも rc 0。Run 3 は 1,288 s で、shell 46 本と Go 11 パッケージに落ちたテストはない。変異は Run 1 で 25 個のうち 22 個が落ち、残った 3 つは下の 557(c)。Run 2 で残った変異は、18360371 の行で落ちる
- sync-docs は 2 周(Run 1 の文書の直しは 0d44799f、Run 2 は b8245ea1)
- cross-review は 2 周。1 周目の ACTION_REQUIRED 1 件は eff83438 で直し、2 周目は指摘なし。codex は read-only の sandbox で一時ディレクトリを作れず、Go のテストは走らせられなかった(書式、同期、shell の構文の確認だけが通った)
- この walkthrough を書く前に、HEAD 481fb73d で `./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` を回し、どちらも rc 0(AC7)
- test Run 3(5af1a67b)のあと、sync-docs Run 2 が `docs/recipes/codex-seat-permissions.md`(と雛形の写し)と tech-debt を変えた。そのあとの通しのテストは回していない

## 8. 残したもの

- tech-debt の行 553: leader の計画助言(`### 計画助言`)は、今も座席の中から相手の CLI を入れ子で呼ぶ。codex の leader では同じ初期化の失敗が起きうる。codex の leader の sandbox の中で `claude -p` が起動するかは確かめていない
- 行 556: `ralph init` の「変更のレビュー」の問いは、CLI が 1 つのとき聞かず、`[org.review].cross_review` を false で書く。org の cross-review は相手の CLI を要らなくなったので、1 つのときも聞けるが、製品の判断として変えていない。直したのは説明文だけ
- 行 557 の (a)〜(d): (a) model が空のとき `Cross-review: ran (reviewer 座席 reviewer(codex、(CLI default)))` と括弧が重なる、(b) 手順 3 の「読み終えられない」道は、手順 2 のように `git: changed` の枝を書かず、手順 5 に任せる、(c) git の状態の部品(`git diff` など)を雛形と共有の定数から一緒に外すと、走らせる行が落ちない(雛形だけの変更なら行の一致で落ちる)、(d) 前の版の TASK の行から作ったコマンドと 3 行から作ったコマンドの比べは、作りの上で同じになり、単独では何も落とさない
- 本物の座席の中での cross-review は走らせていない。確かめたのは、雛形の文、bash と zsh で走らせたコマンド、fake の herdr での spawn まで
- 前の版の reviewer 座席は、新しい leader の 3 行の TASK に QUESTION を返す(`IMPLEMENTER_DRIVER:` の行がないため)。動いている org の座席は、立てたときの雛形のまま。release ノートに、混ぜた版は `ralph org reseat` でそろえると書く(ノートは `/release` で書く)
- verify V7: AC6 は「`xreview-helpers.sh` は差分に入らない」だが、`pick_reviewer` の doc comment の 1 か所を直したので、字面は満たさない。`#` で始まらない変更行は 0 で、関数は 1 バイトも変わらない。振る舞いが変わらないことで満たしたと読む。AC の文は変えていない
- verify V8: AC1 の「`codex exec`、`claude -p` を持たない」は、Scope の「別のモデルを呼ばないと明記する」と字面でぶつかる。雛形は、呼ばないと言う 1 文にだけ 2 語を持つ(Scope に合わせた読み方)
- AC2 は「`git: same` のとき」という条件を付けて満たす。`git diff` が失敗したあとも、先に手順 5 の比べを通すため。AC の文は承認の digest に入るので変えていない
- 計画は系列の決まりで自動承認した。人の目での承認はしていない。同じ系列の reviewer が implementer と同じ見落としをする危険は、ユーザーの決定で受け入れた
