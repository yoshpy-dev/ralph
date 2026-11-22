# sync-docs report: issue-depends-wait

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(承認の digest `7b44cea2d6bc`。Run 2 の前後とも同じ)。issue #364
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2、既定の上限 2 の最後の周)。今の内容は「Run 2」の節。Run 1(cycle 1)はその下に、見出しを 1 段下げてそのまま残した

## Run 2 (cycle 2)

- Branch: `feat/issue-depends-wait`。run 開始時の HEAD は `e67eda2c`(push 済み、worktree は clean。origin/main は `2a069b50`)
- 先行 report(引いた SHA は各 report を含む commit): self-review Run 2 `96bbc430`(Merge: yes、LOW 4。C2-L1〜C2-L4 は `792ffd97` で直った)、verify Run 2 `60862f1e`(pass、LOW 3・INFO 4)、test Run 2 `e67eda2c`(pass、`run-test.sh` rc 0、変異 20 個のうち 19 個が落ち、落ちなかった C3c は穴を測るために足した変異)
- cycle 1 の sync-docs(`f6d1cd35`)のあとに入った中身の変更は 2 つ。`ff59a0e1`(cross-review の 2 件の直し: 頼る先の merge の確かめに (c) を足し、「merge を待つ issue」の 2 で頼る先のブランチの `ready` を確かめ直す)と、`792ffd97`(self-review Run 2 の C2-L1〜C2-L4 の直し)。どちらも skill の 8 写しとテストだけで、Go の本番コードは変えていない(`git diff --stat f6d1cd35..HEAD` の Go は `skill_issue_depends_test.go` と `skill_split_questions_test.go` の 2 ファイル)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。この run は skill にもテストにも触れていないので `sync-skills.sh` は要らなかった
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff origin/main...HEAD` は 19 ファイル、+2,772/-105。Go はテスト 2 ファイルだけ(`skill_issue_depends_test.go` は新規で 790 行、`skill_split_questions_test.go` は固定の直し)で、テスト以外の Go は 0。walkthrough を書くかどうかは `/pr` が決める

### Summary

仕様 `2026-10-07-org-multi-org-director.md` の 2026-11-22 の項(173 行目)を `792ffd97` の skill に合わせた(verify V7)。台帳 `docs/tech-debt/README.md` は、新しい行 1 つ(563 行目)と、既存 3 行への追記(558、561、562)。台帳の Trigger は全行を 2 通りで照合し、cycle 2 の直しで当たった行は verify が挙げた 3 行(561 の (d)、562 の (1)・(2))と、558 の 1 行だった。README と AGENTS.md は直していない。計画は Progress の 1 行を足しただけで、本文は変えていない。

### Changes made

| File | Change | Commit |
|------|------|------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | 173 行目の 2026-11-22 の項の 2 か所。(1) (c) の文に、merge の時刻は `binding` が (a) か (b) に当たる件でだけ読むこと、読めないとき(gh の失敗、空の値)は数えないが (c) に当たらないと決まったのではないので、起票と引き継ぎは待って follow-up を記録し、「merge を待つ issue」の 2 は done にも記録し直しもせず着手も聞かず報告のたびに読み直すこと、読めなかったことはブランチごとにまだ伝えていないときだけ 1 回伝えること、を足した。(2) 引き継ぎで聞くのは follow-up のないブランチに限る文に、2 が done にしたあと記録し直す前に止まったブランチは follow-up のないブランチとして記録するが問いは聞かない、という例外を足した | この report の commit |
| `docs/tech-debt/README.md` | 新しい行 563(頼る先の確かめの限界 3 つ)。558: 「`/self-review`・`/verify`・`/sync-docs` が行を見落とす」の Trigger に当たった注記と Related。561: 先頭の文、(e) の新設(verify V10)、(d) の Trigger に当たったが払っていない注記(V9)、Related。562: 先頭の文、(3) の新設(test の cycle 2 の Test gaps 2)、Trigger (1)・(2) に当たったが払っていない注記とテストのファイルの行数(V8・V11)、Related | この report の commit |
| `docs/plans/active/2026-11-22-issue-depends-wait.md` | Progress の `Sync-docs artifact created` の下に cycle 2 の 1 行。本文は変えていない。digest は前後とも `7b44cea2d6bc`、`git diff --numstat` は +1/-0 | この report の commit |
| `docs/reports/sync-docs-2026-11-22-issue-depends-wait.md` | この report | この report の commit |
| `docs/insights/events/2026-11-22-issue-depends-wait.jsonl` | `sync_docs` の行(cycle 2、pass)。8 行から 9 行 | この report の commit |

Go のコードにも、テストにも、skill にも触れていない。パイプラインの再実行になる変更はない。

### 依頼の項目

#### 1. 仕様 173 行目の (c) の文(V7)

verify V7 が指した文は、`ff59a0e1` で足したまま「読めなければ数えない」とだけ言い、続く 2 の文の「当たらなければ…done にして同じブランチで記録し直し」と並ぶと、読めない時刻でも done と記録し直しを打つと読めた。`792ffd97` のあとの skill に合わせて 2 か所を直した。skill の根拠は次のとおり(`.claude/skills/director/SKILL.md`、行番号でなく節で引く)。

| 仕様の文 | skill の根拠 |
|------|------|
| merge の時刻は (a) か (b) に当たる件でだけ読む | 「起票と記録」の 1 の (c)「merge の時刻は、`binding` が (a) か (b) に当たる件でだけ、`gh pr view ...` で読む」 |
| 読めないときは数えないが、(c) に当たらないとは決まっていない。起票と引き継ぎは待って記録し、2 は done も記録し直しもせず着手も聞かず、報告のたびに読み直す | 同じ (c) の「ただし、(c) に当たらないと決まったのではないので、「merge を待つ issue」の 2 の done と記録し直しは打たない」以下と、2 の最後の「merge の時刻を読めずに確かめに当たらなかったときは…着手も聞かない」 |
| 読めなかったことは、ブランチごとにまだ伝えていないときだけ 1 回伝える | (c) の最後の文「そのブランチについてまだ人に伝えていないとき…だけ伝える」 |
| 2 で止まったブランチは、引き継ぎで頼る先の問いを聞かない | 引き継ぎの「ただし、follow-up のないそのブランチが、2 で done にしたあと記録し直す前に止まったブランチ…でなく」 |

例外の置き場所は、引き継ぎで聞く場合を限る文の中にした(後ろの括弧に足すと、self-review の C2-L3 が skill で直した読み違い、つまり規則の本文と離れた例外と同じになるため)。同じ項の前のほうの文「引き継ぎと着手を聞く前の判定が merge 済みと数える確かめも、起票と同じ (a) か (b) にした」は直していない。同じ項の最後の文が「頼る先の merge の確かめに (c) を足した」と言い、足したあとの確かめを全箇所に掛けているので、前の文は足す前の経緯として読める。

#### 2. 台帳 562 行(V8・V11)

| 項目 | 内容 |
|------|------|
| Trigger (1)(`skill_issue_depends_test.go` に次のテストを足す) | `ff59a0e1` が `TestDirectorSkill_StartCheckCountsAReadyDependencyBranchOnlyByTheDependencyCheck` を足して当たった。`TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半と `.git` の stat は `f6d1cd35` のまま(読んで確かめた)。払っていない |
| Trigger (2)(`directorDependsAskingSentences` か、それが読む箇所) | `ff59a0e1` と `792ffd97` が、それが読む引き継ぎと「merge を待つ issue」の 2 を変えて当たった。関数と 2 つの句(「`feature`だけが合」と「聞く」)は変わっていない(diff の hunk の見出しは関数の名前を出すが、hunk の中身は次のテストのコメントだった)。払っていない |
| 払わなかった理由 | cycle 2 は上限の最後の周で、どちらの払い方もテストの変更になり、全パイプラインの再実行が要る |
| テストのファイルの長さ(V11) | `wc -l` は 790 行(`f6d1cd35` では 607 行)で、800 行の目安の 10 行手前。次にテストを足すと越えるので、注記に数と払い方(確かめ別にファイルを分ける。(1) の後半を削っても 20 行足らずしか減らない)を書き、新しい行は作らなかった。Trigger (1) が「次にテストを足す」で、同じ場面で当たるため |
| 新しい (3) | test の cycle 2 の Test gaps 2(変異 C3c)。org skill の「条件を並べない」確かめは `(a)`・`(b)`・`(c)` の印だけを見るので、印なしで条件を並べ直す文は通る。562 行はこの PR のテストの穴の行で、(2) と同じ「言い回しに頼る確かめの限界」なので、同じ行の (3) にした(ほかの行は覆っていない)。Trigger は、節に `binding.ts`・`pr.number`・`mergedAt`(規則の定義だけが使う語。org skill には今 0 件)を許さないこと。依頼は「gap 1」と書いたが、report の番号は Test gaps 2(1 は director の実際の動きを追う run がないこと。PR の Known gaps に入れる) |

#### 3. 台帳 561 行の (d)(V9)と (e)(V10)

(d): `792ffd97` が `/org` skill の同じ小節の同じ文(頼る先の同名の merge の文)を、director の確かめを名前で呼ぶ形に書き直し、それでも「director が起票のときに 1 回だけ人に聞く」と言うままなので、Trigger に当たった。払っていない(org skill の 4 写しと `TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge` の固定を 1 つ変える)。Trigger の末尾に日付つきの注記、Related に verify の Run 2 の V9 を足した。

(e): V10 の理由の文(2 の「follow-up は記録より後の merge だけを数えるので、前の別の分割計画の同じ名前のブランチの merge では `waiting` のまま」)を、561 行の新しい項目にした。562 行や新しい行にしなかったのは、同じ節(「merge を待つ issue」)の文で、561 行の (a)〜(d) と同じく「その節の次の変更」で直す小さな所見だから。この文は `TestDirectorSkill_TakeoverAndStartQuestionCountADependencyMergedOnlyByItsBinding` の fragment に固定されているので、Why deferred にその名前を書いた。動きに食い違いがないこと(次の文が確かめ直しと done・記録し直しを書き、聞かないという結論は同じ)も Debt に書いた。

#### 4. 台帳の新しい行 563(self-review Run 2 の範囲外の危険、V12、C2-L2 の取らなかった 2 点)

| 項目 | 内容 |
|------|------|
| (a) `binding.ts` が merge より後になる | 書き手は 3 つ: 予約つきの最初の spawn、`idempotentRespawn`(予約のない running の leader に予約を記録する。`idempotentRespawnDecision` の「an org without one records it」)、`reserveAgain` の補償(disband と stop で自分の pane の close が失敗したとき)。self-review は `reserveAgain` だけを挙げていたので、V12 のとおり `idempotentRespawn` を足し、「merge のあとに書くことがあるかは確かめていない」と書いた。別の分割計画の同じ名前のブランチの org が後から立つ道も同じ行に入れた。起票と引き継ぎはすでに merge された頼る先を待ち続け、2 は本物の `ready` を done にして記録し直す(人に PR の番号を伝える) |
| (b) 報告ごとの gh | `792ffd97` で時刻を読むのは (a)・(b) に当たる件だけになったので、残る費用は「`merged` で (a)・(b) に当たり (c) に落ちるブランチが、新しい PR が開くまで報告のたびに `gh pr view` を打つ」こと |
| (c) 取らなかった 2 点 | C2-L2 の推奨のうち、「1 回読み直してから伝える」と「人が「この merge と数える」と言えることを伝える」。取った道(2 が報告のたびに読み直す)で回復するのは gh が答えたあと。gh が失敗し続けると、すでに merge された頼る先を待ち続け、人には 1 回だけ伝わる |
| Trigger | (a)(b): 頼る先が merge 済みなのに数えない報告、または merge-order の JSON が結びつきの履歴か PR の merge の時刻を出す変更。`reserveAgain` か `idempotentRespawn` の変更では先に merge のあとに書くかを確かめる。(c): 「起票と記録」の 1 の (c) の次の変更、または gh の失敗で待ち続ける報告 |
| 直さなかった理由 | (a) は merge-order の JSON(Go)の変更で、計画の Scope と Rollout は skill の文だけ。(c) は skill とテストの 4 写しで、cycle 2 は最後の周 |

self-review Run 2 の「Tech debt identified」の 2 行目(C2-L1〜C2-L4)は、4 つとも `792ffd97` で直ったので、残りだけを行にした。C2-L2 の取らなかった 2 点は 563 の (c)、C2-L4 の (b)(ファイルの長さ)は 562 の注記に入った。C2-L1・C2-L3 の残りはない。Related の cross-review の triage は「cycle 1 の cross-review(commit `25405c12`)」と書いた(次の cross-review が同じファイルを書き直すと、cycle の言い方が曖昧になるため)。

#### 5. 台帳 558 行(依頼にない追記)

verify Run 2 の Coverage gaps が「558 行の Trigger の候補を出す script の提案と同じ穴」と書いている。558 の Trigger は「`/self-review`・`/verify`・`/sync-docs` が、あとの段か人が見つける行を見落とすとき」で、cycle 2 では、この branch の cycle 1 の sync-docs が書いた 2 行(561・562)が `ff59a0e1` と `792ffd97` で当たり、self-review Run 2 は名指さず、verify が人の照合で見つけた。当たったと数え、「払っていない(計画の Scope は skill の文)」の注記を足した。要らなければ、558 の追記(Trigger の末尾の 1 文と、Related の計画の path 1 つ)を落としてよい。

### 台帳の Trigger の照合

依頼は「verify の 3 行を確かめ、ほかにもあれば足す」だった。開いた行(5 セル、取り消し線の文を除いた Trigger のセル)に対して、cycle 2 で変わったもの(director と org の skill、仕様、2 つのテストファイル、`directorDependsAskingSentences`、`gh pr view`、`mergedAt`、`binding`、`scope_reserved`、「merge を待つ issue」、「起票と記録」、`Depends on`、`after:`、`followup` の小節)と、この run が編集する台帳と仕様、計画の語で探した。広い語の組(219 行)は `/org` と `director` が path にも当たって役に立たず、狭い語の組(65 行)の Trigger の頭を読んだ(370・382・426 は全文も読んだ)。当たったのは verify の 3 行と、上の 558 だけ。当てなかった主な行:

| 行(最初の語) | Trigger | 当てない理由 |
|------|------|------|
| 426 `skill_split_questions_test.go` is over 800 lines | このファイルに次のテストを足す、節の読み手を変える | `792ffd97` は 1 つのテストの fragment を 1 文直しただけ(3 行)。テストも読み手も足していない |
| 427 `--limit` | `--limit` の次の変更、そのファイルに次のテストを足す | どちらもしていない |
| 370 `Send` and the leader template tests | `Send` か leader のテスト、挙げた 5 ファイルが 300 行増える | 挙げた 5 ファイル(`verbs.go`、`verbs_test.go`、`prompts_test.go`、`spawn.go`、`spawn_test.go`)は触っていない。`skill_split_questions_test.go` は注記に出るが、挙げた 5 つの中にない |
| 395 four test files | 4 つの cleanup 系ファイルの次の変更 | 4 つとも触っていない |
| 470 (取り消し線の行) | `/director` skill の次の変更 | 1 つ目のセルが取り消し線で、2026-11-13 に閉じた行。閉じたあとに注を足した PR はない(cycle 1 と同じ) |
| 267 inbox section | `受信箱の処理` の節の次の変更 | 触った節は `## 分割計画と機能ごとの org` と `## 進捗の報告` の下。`受信箱の処理` の中ではない |
| 182 (a) | `/org` skill のコマンド表の次の変更 | org skill の hunk は表の下の `followup` の小節(13 行)。F-8 の `stop`・`disband` のセルの長さも変わらない |
| 382、385、512、513 | split-plan の例、send と確かめの手順、受信箱の手順、hook の bullet | どれも触っていない |
| 186、188、191、193、194、195、196、225 | `reserveAgain`、`idempotentRespawn`、`scopeReservedEvent` の次の変更 | Go の本体は変えていない。563 行はそれらを読むだけ |
| 252、259、317、421、424 | `mergeorder*.go`、`FollowupStore`、`orgRunsOn` の次の変更 | Go の本体は変えていない |
| 480、547、555、557 | leader の task の `Depends on`、`FeatureBinding`、共有のパスの手順、`sameFeature` | どれも触っていない |

この run の編集(仕様 173 行、台帳、計画の Progress)が当てる行も探した。Trigger のセルを `tech-debt`、`docs/specs`、`2026-10-07`、`docs/plans/active`、`Progress` で探すと、仕様の文や台帳の文を「次に編集する」ことを Trigger にした行はなかった。179 は仕様の後の段(director の後の段階)、205 と 227 は plan の path と Progress の行を書き換えるコード、100・376・558 は台帳を引くだけか台帳の check の話で、そのほかの `Progress` は reseat の進捗の節の話。

### README と AGENTS.md ほか

直していない。`git grep` で `README.md`、`AGENTS.md`、`CLAUDE.md`、`docs/quality`、`docs/recipes`、`docs/architecture`、`.claude/rules`、`internal/org/prompts`、`templates/base` の docs と rules を、`頼る先の merge`、`merge を待つ`、`Depends on`、`binding.ts`、`頼る先`、`ralph-split` で探して 0 件(verify の「ずれなし」と同じ)。cycle 2 の変更は skill の文とテストだけで、`ralph org followup add --help` と `/org` のコマンド表は cycle 1 の判断のまま(一般の規則。例外は `/org` skill の小節にある)。

### handoff と実際の違い

- 依頼は test の cycle 2 の「gap 1」(C3c)だったが、report の番号は Test gaps 2。1 は director の実際の動きを追う run がないこと(cycle 1 の 1 と同じで、Known gaps)
- 依頼の V12 は台帳の行の文を「`reserveAgain` の補償、`idempotentRespawn` の書き足し」にすることだった。`idempotentRespawn` が merge のあとに書くかは確かめていないので、行にそう書いた
- 依頼にない追記が 2 つある(558 の注記、仕様の例外の置き場所を前の文に決めたこと)。どちらも上に理由を書いた
- 当たった行は verify の 3 行に 558 を足した 4 行。ほかに満たす行はなく、注を足していない

### 検査

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: 14 skill そろっている
- `go test -count=1 ./internal/org/ -run 'Skill|DirectorSkill|DocsMatch'`: ok(1.113 s)。この run は Go にも skill にも触っていないので、cycle 2 の状態の確かめの繰り返し
- 台帳の plan 参照(`docs/plans/(active|archive)/...md` の 148 path): 存在しない参照は 0
- 台帳: `git show HEAD:` と比べて、変わった行は 558・561・562 と、新しい 563 だけ。4 行とも 5 セル(`(?<!\\) \| ` で割る)、セルごとのバッククォートが偶数、`~~` が偶数、`file:line` なし、U+FFFD なし。ファイルは改行で終わり、最後の行の終わり方も元と同じ
- 計画の digest: `7b44cea2d6bc`(前後とも)。`git diff --numstat` は +1/-0

### PR の本文の Known gaps に入れるもの(cycle 2 の分。cycle 1 の 6 点は下の Run 1 のとおり変わらない)

1. cross-review の直しで、merge 済みと数える条件に (c)(PR の merge が結びつきの記録より前でない)を足し、「merge を待つ issue」の 2 は頼る先のブランチの `ready` も確かめ直して、当たらなければ done にして記録し直す。計画の Scope の 1 つ目と 4 つ目の項目は (a)・(b) だけで書かれていて、この 2 つを含まない(どちらも数える条件を狭める向き。計画の本文は digest で守られているので直していない。verify V13)
2. gh が失敗し続けると、すでに merge された頼る先を待ち続け、人には 1 回だけ伝わる。人が「この merge と数える」と言える道は skill に書かれていない(台帳 563 の (c))
3. 同じ名前のブランチの別の分割計画の org が後から立つか、`scope_reserved` が merge より後に書かれると(`reserveAgain` の補償。`idempotentRespawn` は確かめていない)、頼る先の本当の merge を数えないことがある(563 の (a))。報告ごとの gh の呼び出しも、そのブランチでは増える(563 の (b))
4. 台帳 561 の (d) と 562 の Trigger (1)・(2) は cycle 2 の直しで当たったが払っていない(上限の最後の周のため。561 と 562 に日付つきの注記)
5. テストの穴: org skill の「条件を並べない」確かめは印なしの言い直しに落ちない(562 の (3))。director の実際の動きを追う run はなく、(c) は文字列でなく時刻として比べるという文までしか固定していない(test の cycle 2 の Test gaps 1)。`skill_issue_depends_test.go` は 790 行で、次のテストが目安の 800 行を越える
6. 仕様 173 行は cycle 2 の決めごとまで書いた。仕様の日付つきの項を固定するテストはなく(cycle 1 の Test gaps 4)、今回は文書を直したが、固定はない

### Insight event

`./scripts/insights-append.sh --slug issue-depends-wait --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行(cycle 2、pass)を足した。rc 0。`git status --short` は既存の jsonl の `M` だけで、迷子のファイルはない(`--slug` に日付を付けていない。UTC の日付が 11-22 で、同じファイルに追記)。

### 実行結果

- `git status --short` は、変更が 4 ファイルの `M`(jsonl、計画、仕様、台帳)とこの report の `M` だけ。skill と Go は変わっていない
- この report の commit は `docs: add the cycle 2 sync-docs run of issue-depends-wait` の 1 つ。push はしていない

---

## Run 1 (cycle 1、履歴)

> 以下は cycle 1(run 開始時の HEAD `63ad5152`)の report をそのまま残したもの。見出しだけ 1 段下げた。

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(承認の digest `7b44cea2d6bc`。この run の前後とも同じ)。issue #364
- Branch: `feat/issue-depends-wait`。run 開始時の HEAD は `63ad5152`(origin/main は `2a069b50`、取り込み済み)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report(引いた SHA は各 report を含む commit。report の中の `HEAD:` の行の値ではない): self-review `c1ff3341`(Merge: yes、MEDIUM 1、LOW 6。M1・L1・L2・L3・L5 は `03ced266` で直った。L4・L6 は台帳へ)、verify `95d755b2`(pass、HEAD 03ced266、LOW 2・INFO 4)、test `63ad5152`(pass、`run-test.sh` rc 0、変異 10 個がすべて落ちた)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。skill は触っていないので `sync-skills.sh` は要らなかった
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff origin/main...HEAD` は 16 ファイル、+1,882/-92。Go は 2 ファイルだけで、どちらもテスト(`skill_issue_depends_test.go` +607、`skill_split_questions_test.go` +18/-12)。テスト以外の Go は 0。walkthrough を書くかどうかは `/pr` が決める

### Summary

仕様 `2026-10-07-org-multi-org-director.md` の 2026-11-22 の項(173 行目)を、`03ced266` が決めた skill に合わせた(verify V1・V2)。台帳 `docs/tech-debt/README.md` に新しい行を 2 つ足した(561・562 行目)。台帳の Trigger は全行を照合し、この PR が満たす行は 0 だった(verify の結論と同じ)。`followup add --help` と `/org` のコマンド表の一般の規則は直さなかった(V5、理由は下)。README と AGENTS.md は起票の待ちを書いていないので直していない。計画は Progress の行と AC1〜AC7 の印だけを足し、本文は変えていない。

### Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | 173 行目の 2026-11-22 の項の 2 か所。(1) 「`feature` だけが合うときは人に聞く。…同じ数え方にした」を、聞く場所と回数(起票の待つブランチを決める前に 1 回)、選択肢(「新しい merge を待つ」が推奨、「この merge と数える」)、答えが `after:` に残ること、follow-up のあるブランチは引き継ぎでも着手を聞く前の判定でも聞き直さないこと、引き継ぎで聞くのは follow-up のないブランチ(起票の記録が途中で止まった場合)の 1 回だけで `merged` なのに `feature` だけが合うものに限ること、merge 済みと数える確かめは起票と同じ (a) か (b) であること、に書き直した。(2) 「この場合だけは、今までどおり director が印の行を書き換える」を、書き換える 2 つの場合((1) 頼る先を start するときに slug か Type を変えたとき、(2) 引き継ぎで「この merge と数える」と答えたとき。どちらも「merge を待つ issue」の 3 の手順)に直し、続く `/org` skill の文の「この例外」を「頼る先のブランチの例外(前の別の分割計画の merge は数えず、記録して待つ)」に言い換えた | この report の commit |
| `docs/tech-debt/README.md` | 新しい行 2 つ(561・562 行目、末尾に追記)。561: 「merge を待つ issue」の 3 の小さな所見(self-review L4 の (a)(b)(c)、verify V3 の (d))。562: この PR のテストの穴(self-review L6、test の Test gaps 2)。既存の行は変えていない | この report の commit |
| `docs/plans/active/2026-11-22-issue-depends-wait.md` | Progress の `Test artifact created` の次に `Sync-docs artifact created` の行(`[x]`)。AC1〜AC7 に `[x]`(verify と test がすべて Met とした)。本文の文は変えていない。digest は前後とも `7b44cea2d6bc` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-issue-depends-wait.md` | この report | この report の commit |
| `docs/insights/events/2026-11-22-issue-depends-wait.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行(`self_review`・`verify`・`test` は揃っていた) | この report の commit |

Go のコードにも skill にも触れていない。パイプラインの再実行になる変更はない。

### 依頼の項目

#### 1. 仕様の 2026-11-22 の項(V1・V2)

verify V1 が指した「報告のたびに聞くとも読める」文は、起票で 1 回だけ聞く形に書いた。V2 が指した 2 つ目の書き換え(引き継ぎで「この merge と数える」と答えると、そのブランチを `gh issue edit` で `after:` から除く)は、書き換える場合の一覧に入れた。skill のどの文に当たるかは次のとおり(HEAD の `.claude/skills/director/SKILL.md`)。

| 仕様の文 | skill の根拠 |
|------|------|
| 起票で待つブランチを決める前に 1 回聞く。選択肢は「新しい merge を待つ」(推奨・先頭)と「この merge と数える」 | 「起票と記録」の 1 の「頼る先の merge の確かめ」の後ろの段落 |
| 答えは `after:` に残る。follow-up のあるブランチは聞き直さない | 同じ段落の「答えは印の行の `after:` に残る」、「merge を待つ issue」の 2 の「follow-up のあるそのブランチは、`feature` だけが合っても聞かない」 |
| 引き継ぎで聞くのは follow-up のないブランチだけ | 「merge を待つ issue」の引き継ぎの「ただし、follow-up のないそのブランチが…聞くのはこの場合だけ」 |
| 書き換えは 2 つの場合 | 同じ節の 3 の、slug か Type を変えたときの移す手順と、引き継ぎの「この merge と数える」の書き換え(「3 の書き換えの手順で…`after:` からそのブランチだけを除き」) |

計画の Assumptions(例外は 1 つ)と Scope の 4 つ目の項目(引き継ぎと 2 でも聞く)は、計画の本文なので直していない(本文を直すと digest が動く)。ずれは verify V4 が Progress に書いており、PR の本文の Known gaps にも入れてほしい(下)。

#### 2. 台帳の行

| 行 | 内容 | 出どころ |
|------|------|------|
| 561 | 「merge を待つ issue」の 3 の 4 点。(a) 移す手順が、人が答え済みの issue も選び、新しいブランチの merge で同じ人に聞き直す。(b) start が worktree の記録で断ったときの案内が、slug を変える案を先に置き、その費用(待つ側の全 issue を書き換える)を言わない。(c) 元の機能の Type を本文の `- Type:` の行から読むが、行の数と値(`splitFeatureTypes`)を確かめない。(d) `/org` skill の文が「起票のときに 1 回だけ」と言い、`/director` skill は引き継ぎでも 1 回聞く | self-review L4、verify V3 |
| 562 | この PR のテストの穴 2 つ。(1) `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半が、既存の 2 つのテスト(`TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs` の「no PR is waiting」の行、`TestOrgMergeOrder_FollowupSectionFollowsTheNotes` の `feat/none` の行)と同じことを固定し、前半の `.git` の確かめは `t.TempDir()` の直下しか見ない(CLI の経路も通らない)。(2) `directorDependsAskingSentences` が「`feature`だけが合」と「聞く」を両方含む文だけを数えるので、別の言い方の問いを足しても落ちない(変異 D3 は同じ言い方なので落ちる) | self-review L6、test の Test gaps 2 |

どちらも、Debt の 1 列目に原文の ID(L4、V3、L6、Test gaps 2)を書き、Trigger に直し方を、Related に計画を `docs/plans/active/...` の path で足した(`/pr` の archive が書き換える)。テストの場所は `file:line` ではなくテストの名前で引いた(L6 の `mergeorder_followup_test.go:197` と `internal/cli/org_merge_order_followup_test.go:156` は、それぞれ `TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs` と `TestOrgMergeOrder_FollowupSectionFollowsTheNotes` の中)。

self-review の「Tech debt identified」のもう 1 行(L1: 確かめの文が 5 か所に分かれる)は、`03ced266` が規則を 1 か所に置いたので(`TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling` が規則の文を 1 回だけ数える)、行にしていない。

#### 3. 台帳の Trigger(照合)

依頼は「verify は満たす行なしと言った。確かめて、あれば注を足す」だった。台帳の全行(取り消し線の文を除いた Trigger のセル)を、この diff が触れたもの(director skill と org skill の節の名前、2 つのテストファイル、`起票と記録`、`merge を待つ issue`、`after:`、印の行、`follow-up`、`merge-order`、`Depends on`)で 2 回探した。1 回目は名前の一致で 71 行、2 回目は話題の一致で 44 行が候補になり、約 60 行の Trigger の頭を読んだ。**満たす行は 0**(注は足していない)。

| 行(最初の語) | Trigger | 当てない理由 |
|------|------|------|
| 267 `Three test gaps of the org-director-attach PR` | `/director` skill の `受信箱の処理` の節の次の変更 | 触った節は `### 人への聞き方`・`### 起票と記録`(どちらも `## 分割計画と機能ごとの org`)と `### merge を待つ issue`(`## 進捗の報告`)。`## 受信箱の処理`(992 行目から)の中ではない |
| 274 (b) | `director_skill_sync_test.go` に次のテストを足す、`始める` の 3 | そのファイルは触っていない。新しいテストは新しい `skill_issue_depends_test.go` |
| 395 `Four test files went past the 800-line guide` | 4 つのファイル(`org_merge_test.go`、`merge_run_test.go`、`director_skill_sync_test.go`、`cleanup_test.go`)の次の変更 | どれも触っていない。`skill_split_questions_test.go` は 4 つに入っていない |
| 426 `internal/org/skill_split_questions_test.go is over 800 lines` | このファイルに次のテストを足す、節の読み手を変える | 6 つの既存のテストの fragment を直しただけ(+18/-12)。テストも読み手(`directorSkillIntro` ほか)も足していない。行は「count は書かない」と断っているので、古い `wc -l` もない |
| 427 `The --limit value of the two gh issue list commands` | `--limit` の次の変更、そのファイルに次のテストを足す | `--limit 1000` は変えていない(diff の hunk の見出しに文脈として出るだけ)。テストを足していない |
| 470 `~~The /director skill ... scaffold version note~~` | `/director` skill の次の変更 | 1 つ目のセルが取り消し線で、2026-11-13 に docs/director-notice-handling が閉じた行(最後の注は 11-15)。閉じたあとの director skill の変更に注は足されておらず、#365 の sync-docs も閉じた行として当てていない |
| 477、509、512、513、385、386 | gate accept、知らせの行の小節、hook QUESTION の項目、send の確かめの手順、wait の終了コード | その節も項目も触っていない |
| 182 `Two documentation findings of org-stop-all` (a) | `/org` skill のコマンド表の次の変更 | 表は触っていない。足した 1 文は表の下の `followup` の小節(`PR がもう merge されたブランチは follow-up にしない` の項目)にある。F-8 の `stop`・`disband` のセルの長さも変わらない |
| 382 `RC4` | `/org` skill の分割計画の例、`leader.md` の手順 | どちらも触っていない |
| 416、255 | 予約外の文書の follow-up の小節、origin の ssh の確かめ | その小節も確かめも触っていない |
| 547、555 (b)、557 (2) | `feature:` の行、`共有のパスの直し方` の手順 1、`--issue` の実地 | `feature:` の行も手順も変えていない(skill は `issue:` の行を読む側で、行を変えていない)。実地は報告待ち |
| 252、259、311、423、424、247〜249、241 | `mergeorder*.go`、`FollowupStore`、`gate.go` の次の変更 | Go の本体は触っていない |
| 558 `The tech-debt rows whose Trigger a branch meets are found by hand` | `/self-review`・`/verify`・`/sync-docs` が行を見落とす | 見落としはない(verify と今回の照合が同じ結論) |

#### 4. `followup add --help` と `/org` の表(V5)

直さなかった。理由は 3 つ。(1) help の文は「記録より前の merge は数えない。そのため、PR がもう merge されたブランチは follow-up にせず、issue をすぐ聞く」で、「PR がもう merge された」は follow-up が待つその PR を指す。頼る先の同じ名前の前の PR は待つ PR ではなく、待つのは次の PR なので、help の文と食い違わない。(2) 例外は `/org` skill の同じ小節に 2 つとも書いてある(答えの記録と、頼る先のブランチ)。答えの記録の例外も、この PR の前から表と help にはなかった。(3) help を変えると `internal/cli` の Go の変更になり、pin されたテストと台帳の Trigger を引く。得るものは小さい。director が読む手順は `/director` skill にある。

#### 5. README と AGENTS.md

直していない。README の follow-up の記述(134・294 行目)は CLI の振る舞いで、この PR は変えていない。AGENTS.md の `internal/org/` の行は follow-up の記録と merge-order の読みを述べ、skill の手順もテストのファイルも挙げない。`docs/recipes/`、`docs/quality/`、`.claude/rules/ralph/` に `ralph-split` や `merge を待つ issue` を引く所はない(`git grep` で `docs/evidence/` の履歴だけが当たる)。

#### 6. 検査

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: 14 skill そろっている
- `go test -count=1 ./internal/org/ -run 'Skill|DirectorSkill|DocsMatch|Followup'`: ok
- `go test -count=1 ./internal/cli/ -run 'Followup|Help|Skill|Docs'`: ok
- 台帳の plan 参照(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ検索): 存在しない参照は 0 件
- 台帳: 新しい 2 行は、5 セル(`(?<!\\) \| ` で割る)、セルごとのバッククォートが偶数、`~~` なし、`file:line` なし、U+FFFD なし。`git show HEAD:` と比べて、変わった既存の行は、最後の空行(新しい行 1 つに置き換わる)だけ
- 計画の digest: `7b44cea2d6bc`(前後とも)。`git diff --numstat` は +9/-7(AC の 7 行の印と Progress の 2 行)

### handoff と実際の違い

- 依頼は L4・L6・Test gaps 2 の 3 点だった。verify V3(INFO、org skill の「起票のときに 1 回だけ」)は、同じ小節の文なので 561 の (d) に入れた。落としてよい
- 依頼は仕様の「この場合だけ」を「2 つの場合」にすることだった。続く文の「この例外」(`/org` skill の規則に足した例外を指す)が、書き換えの 2 つの場合と読めるようになったので、「頼る先のブランチの例外」に言い換えた
- Trigger を満たす行は 0 で、注は足していない。台帳の既存の行は変えていない

### PR の本文の Known gaps に入れるもの

1. director が skill の手順どおりに動く run がない(test の Test gaps 1)。起票で 1 回だけ聞くこと、まだないブランチの follow-up を記録して `waiting` で待つこと、頼る先の PR の merge で待つ側が `ready` になり着手を聞くことは、skill の文とそれを固定するテスト、fixture の `FollowupAdd` と merge-order の `waiting` までで、実機の director では追っていない
2. 計画の本文と、仕上がりの skill が 2 点で違う(計画の本文は digest で守られるので直していない)。(a) Scope の 4 つ目の項目は「引き継ぎと 2 でも `feature` だけが合えば人に聞く」だが、`03ced266` で起票の 1 回だけになり、引き継ぎと 2 が聞くのは follow-up のないブランチだけ(verify V4。仕様の 2026-11-22 の項に今の形を書いた)。(b) Assumptions は印の行を director が書き換える例外を 1 つ(slug か Type を変える場合)と書くが、引き継ぎの「この merge と数える」の書き換えが 2 つ目(verify V2。仕様に 2 つとも書いた)。受け入れ条件 3(印の行を手で書き換えなくてよい)の例外が 2 つあることを PR の本文に書く
3. 「merge を待つ issue」の 3 の移す手順は、人が答え済みの issue も選び、slug を変える案を費用の説明なしに先に出し、`- Type:` の行の数と値を確かめない(self-review L4、台帳 561 の (a)(b)(c))
4. テストの穴: `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半が既存のテストと重なる、問いの禁止の検査が別の言い方を拾わない(self-review L6、test の Test gaps 2、台帳 562)。fragment の照合は固定した文のあいだに足した矛盾する文を拾わない(Test gaps 3、skill の文を固定するテストに共通)。仕様の日付つきの項を固定するテストはなく、V1・V2 は `03ced266` のあとも残った(Test gaps 4。今回は文書を直したが、固定はない)
5. `followup add --help` と `/org` のコマンド表は「merge 済みのブランチは follow-up にしない」を一般の規則のまま言う。例外は `/org` の小節にある(verify V5、上の理由で直していない)
6. (b) の `binding.issue` が実際の `ralph org start --plan --issue` で記録されることは、#365 の単体テストが固定している。実機の確かめはない(verify の Coverage gaps)

### Insight event

`./scripts/insights-append.sh --slug issue-depends-wait --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行(cycle 1、pass)を足した。rc 0。`git status --short` は既存の jsonl の `M` だけで、迷子のファイルはない(`--slug` に日付を付けていない)。UTC の日付が 11-22 なので `self_review`・`verify`・`test` の行と同じファイル。

### 実行結果

- 仕様の Edit は 3 回(2 か所と、1 か所の言い直し)。Edit は一意でない文を断るので、どれも置き換える前の文は 1 回だけだった
- `git status --short`: 変更は 4 ファイルの `M`(jsonl、計画、仕様、台帳)とこの report の `??` だけ。skill と Go は変わっていない
- この report の commit は `docs: sync docs for issue-depends-wait` の 1 つ。push はしていない
