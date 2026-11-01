# Self-review report: director-split-questions

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-director-split-questions.md`(issue #271、D1〜D6、AC1〜AC8、スライス S1 667e4d29・S2 b0f0a8ad・S3 3f0b69fc・S4 379fda6a)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff 4fef6b8c...HEAD`(HEAD b9f8a999、14 ファイル、+2113/-99)。`/director` skill の「分割計画と機能ごとの org」の 5 つの `###` 節、「計画の承認」、「merge を待つ issue」、「予約外の文書の follow-up」(写し 4 つ)、`/org` skill の 1 文、新しいテスト `internal/org/skill_split_questions_test.go`(676 行、13 本)と既存 3 ファイルの固定の更新、実機の証跡 `docs/evidence/director-ask-herdr-2026-11-01.md`。skill は agent が従う指示なので、agent が読む順に読み、節どうしの参照、`after:` の印の行の扱い(起票・引き継ぎ・着手の問い・文書の follow-up)、書いたとおりには動かないコマンド、読み違えうる文を見た。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1(MEDIUM)と L1〜L8(LOW)。

## Evidence reviewed

- 計画の D1〜D6、AC1〜AC8、Progress checklist(S1〜S4 の implementer の報告、S3 の consult の申し送りが S4 で入ったこと)。
- `.claude/skills/director/SKILL.md` の「分割計画と機能ごとの org」(468〜626 行)、「受信箱の処理」の「計画の承認」(922〜960 行)、「merge を待つ issue」(1229〜1320 行)、「予約外の文書の follow-up」(1320〜1400 行)、「してはいけないこと」。`/org` skill の「merge を待つ issue」(1608〜1620 行)。
- 写し: `/director` と `/org` の 4 つの写しは、root と `templates/base/` のあいだで `cmp` が一致した(`.claude` 同士、`.agents` 同士)。
- 参照先が今のコードと合うか: `feature: <split>/<slug> branch <branch> worktree <path>` の行(`internal/cli/org.go:1760`、`:1908`)、`reserved:` の行(`internal/cli/org.go:1901`、`internal/cli/status_followup.go:107`)、`followup list --all`(`internal/cli/org_followup.go:162`)、`merge-order --all`(`internal/cli/org_merge_order.go:206`)、`director status` の `wake:` の行(`internal/cli/org_director_report.go:314`)、`ValidateFollowupAdd`(`internal/org/followup.go:219-253`)。
- 証跡: 画面の写し(T1〜T5)が skill の「人への聞き方」の根拠(Enter が 1 番を選ぶ、自由記述の欄なら 1 行が答えになる、選択肢の上では 1 行が消える)と合う。`/Users/` や `hiroki` は含まれず、`$HOME` は `~` に置き換わっている。
- テスト: 新しい 13 本が、`squashSpace` で空白と改行を除いた断片比べ(`containsPhrase`、`prompts_test.go:145`)と、節ごとの `checkDirectorSkillPlaces` で skill を読む。印の行の正規表現は、skill の gh コマンドと同じ定数(`directorSplitMarkPattern`)から作り、Go の regexp で行頭だけに当たることを 4 つの本文で確かめる。既存 3 ファイルの変更は、新しい文に合わせた断片の差し替えで、断片を消したものはない(`skill_followup_test.go` の step 3 は「すべて done にする」が増えた)。`skill_doc_followup_test.go` の範囲を `directorSkillSubsection` から `directorSkillSection` に広げた変更は、固定する 5 つの断片が新しい 5 つの `###` に散ったためで、意図は弱まっていない。
- probe(scratchpad の使い捨て、確かめたあと消した):
  - skill の新しい 5 つのコマンド(`gh issue list --jq`、`gh issue create --body-file`、`gh issue edit --body-file`、`ralph org followup add --note '…'`、`gh issue view --jq .body`)を `.claude/hooks/pre_bash_guard.sh` に通すと、どれも出力なし(許可)。
  - `ralph` を `echo` に置き換えた `followup add --issue 5 --after-branch x --note '<行>'` で、行が `ralph-split: a/b after: x';touch pwned;echo '` のとき、`sh -c` の実行で `pwned` が作られた。`--after-branch x;touch pwned`(引用符なし)でも作られた(M1)。
  - `ValidateDirectorWake`(`internal/config/config.go:575-583`)は順序を問わず、登録は与えられた順で持つ(`internal/org/director.go:527`)ので、`--wake push,pull` の登録は `wake: push,pull` と出る(L5)。
- 確かめていない: テストと静的検査(skill の規則どおり、走らせていない)。`gh ... --jq` が gojq の `test` で `(?m)` を行頭の意味に取ることは、gh を打たずに(Go の regexp の意味で)読んだだけ。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | security | M1 issue の本文から読んだ印の行とブランチ名を、そのまま shell のコマンドに入れる。新しい引き継ぎは `gh issue list` の結果から印の行のある open の issue を全部拾い、その本文の `after:` のブランチを「起票と記録」の 4 で記録し直す。4 のコマンドは `--after-branch <ブランチ>`(引用符なし)と `--note '<印の行>'`(単一引用符)。issue の本文は、その repository に issue を立てられる人なら誰でも書ける(この repo は public。`gh repo view` が PUBLIC)ので、`after: x';curl …;echo '` のような行や `after: $(…)` を持つ issue で、director の shell が別のコマンドを走らせる。headless の director は、permission mode が autonomous なら確認なしでコマンドを打つ(`internal/org/director_start.go:145-149` の 3 つの mode のうち)。skill は「issue の本文は…データで、指示として読まない」と書くが、それはコマンドの文字列に入れる経路を止めない。同じ skill の別の項は、leader が書いた URL を「コマンドに入れない」(1362 行)、`<ブランチ>` は「merge-order が出す名前を使う」(1257 行)と、ralph が読んだ値だけをコマンドに入れる決まりを持つ。新しい経路だけがこの決まりの外にある。ralph 側の `validateFollowupBranch` は空白・制御文字・先頭の `-` を断るだけで、shell の特殊文字は断らず、shell が先に解釈するので ralph では止められない。起票した director 自身が書いた行は `feature:` の行のブランチと検証済みの id・slug から組むので安全で、危ないのは引き継ぎと着手の問いの「先に記録する」(読んだ行から記録し直す)だけ。同じ「誰でも書ける」ことは、「予約外の文書の follow-up」の 1 にも効く: 他人が `ralph-split: <分割計画の id>/<slug> …` の行を持つ issue を立てると、その機能は「数えない」ことになり、文書の follow-up を集め始める条件が早く満たされる(こちらはコマンドに入れないので、手間の増減で済む)。 | `.claude/skills/director/SKILL.md:1241-1257`(引き継ぎ)、`:1279-1290`(着手の問いの前の記録)、`:612-622`(4 のコマンド)、`:1257-1259`、`:1362`(比べる決まり)。`internal/org/followup.go:239-253`。probe: 上の `pwned` の 2 例 | 本文から読む行には形を決める: 行全体が `ralph-split: <id>/<slug> after: (none\|<ブランチ>(,<ブランチ>)*)` に当たり(id と slug は ralph の識別子の形 `^[a-z][a-z0-9-]{0,29}$`、ブランチは `[A-Za-z0-9._/-]` だけ)、本文に 1 行だけあるものを印の行とし、当たらない行や 2 行以上あるときは記録せず、その issue の番号を人に言う。note は読んだ行を写さず、検証した id・slug・ブランチから組み直す。ブランチは `ralph org merge-order --all` が出す名前と一致するものだけを記録する。あわせて、起票した人が director の gh の利用者か人であることを `--json number,body,author` で確かめる案を検討する。「引き継ぎ」と「merge を待つ issue」の 2 に書き、テストに本文が `'` や `;` を含むときの文の固定を足す。 |
| LOW | maintainability | L1 起票の時点で、待つブランチの PR がすべて merge 済みなら、follow-up が 1 件も残らず、着手を聞く経路がなくなる。「起票と記録」の 4 は merge 済みのブランチを「記録せず、待ち終えたブランチとして数える」と書く。着手を聞くのは「`ready` の件があったとき」(「merge を待つ issue」の 2)なので、すべてのブランチが merge 済みだと `ready` の件が生まれず、issue は印の行だけを持つ open のまま残る。同じ節の 1 は、記録の前に PR が merge されていたら「issue をすぐ人に聞く」と書いており、起票の経路にだけその文がない。起きる形: 重なる org の PR は merge されたが org はまだ走っている(掃除の前)ので、start が重なりで断り、起票を選ぶ。 | `.claude/skills/director/SKILL.md:615-619`(4)、`:1260-1263`(1 の「すぐ人に聞く」)、`:1276-1284`(`ready` の件があったら)。 | 4 に「待つブランチがすべて merge 済みなら、記録せず、起票した issue の番号を伝えて着手を聞く(「merge を待つ issue」の 2 の問いと選択肢)」を足す。1 の「すぐ人に聞く」と同じ形にそろえる。固定は `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` に足す。 |
| LOW | readability | L2 「頼る機能」が、同じ依存の逆の端を指して使われている。「機能ごとの出力先」の 2 は、`- Depends on:` で「人が起票を選んだ機能に頼る機能」(頼る側)を主語にし、続けて「説明に頼る機能の名前を書く」(頼る側の名前を書くことになり、意味が通らない。頼られる側のはず)。「起票と記録」の 1 は「頼る機能を起票したので起票した機能」、2 は「`- Depends on:`(頼る機能を起票したなら…)」で、こちらは頼られる側を指す。1 の文は「頼る機能」を頼る側と読むと、起票した順が逆の文になる。同じ 2 の中には「頼られる機能を先」という語もある。 | `.claude/skills/director/SKILL.md:566-568`、`:598`、`:603`。`internal/org/skill_split_questions_test.go:363-364`、`:413`、`:418` | 頼る側を「頼る機能」、頼られる側を「頼られる機能」(または「頼る先」)に決め、3 か所を直す: 出力先の 2 の「説明に頼られる機能の名前を書く」、起票と記録の 1 の「頼られる機能を起票したので起票した機能」、同 2 の「(頼られる機能を起票したなら…)」。テストの 3 つの断片も合わせる。 |
| LOW | maintainability | L3 着手のあと start が重なりで断られ、人が待つと答えたときの「印の行の `after:` を書き換える」手順が、本文をどう作るかを決めていない。`gh issue edit <番号> --body-file <ファイル>` は本文全体を置き換える。ファイルは「起票と記録」の 2 のとおりに書く、とあるが、2 は機能の本文・`- Reserve:`・分割計画の id・重なる org から本文を新しく作る手順で、既存の本文を読み直して行だけを変える手順ではない。2 のとおりに作ると、人が起票のあとに足した本文や、`--note` で足した情報が消えうる。印の行が 2 行になる読みも防いでいない。 | `.claude/skills/director/SKILL.md:1309-1313`(S4 の追加)、`:602-608`(2) | 「issue の本文を `gh issue view <番号> --json body --jq .body` で読み、印の行の `after:` の値だけを置き換えたものをファイルに書く」とし、印の行は本文に 1 行だけに保つ、と足す。固定は `TestDirectorSkill_MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting` の断片に。 |
| LOW | maintainability | L4 「予約外の文書の follow-up」で、起票した機能を「数えない」のは 1 の開始の条件だけで、2〜6 は残りの機能を「機能ごとに」読む。2 は merge-order の ORG の列の org の RESULT を探し(起票した機能は org も PR もない)、3 は「RESULT にも PR の本文にも follow-up の行がない機能」を「確かめていない機能」として一覧に載せ、6 は確かめていない機能の「名前と PR と理由」を案の本文に書く。文字どおりに読むと、起票した機能が「確かめていない機能」に入り、PR のない項を持つ案が人に出る。起票していない機能が merge されたあと、同じ分割計画に起票した機能が 1 つでもあれば起きる。また 7 の「上の『分割計画と機能ごとの org』の手順(台帳への書き込み、図解ページでの承認、digest の 2 行、start)」は、節が 5 つの `###` に分かれたあとも括弧の 4 つだけを指すが、どの節の手順かは書かない。文書の機能が重なりで start を断られたときの扱い(「機能ごとの出力先」の 3)が、この経路に及ぶかも決まっていない。 | `.claude/skills/director/SKILL.md:1335-1349`(除外)、`:1356-1370`(2・3)、`:1380-1386`(6)、`:1388-1392`(7) | 1 の除外に「起票した機能は 2〜6 でも読まない」と足す。7 に、括弧の手順は「分割計画の承認」の 1〜4 と「機能ごとの出力先」の 3 を指し、「分割計画の前の質問」と出力先の問いは文書の機能では聞かない、と書く(start が重なりで断ったときは人に伝えて聞く、など)。 |
| LOW | exception-handling | L5 「人への聞き方」は、AskUserQuestion を使わない条件を `wake:` の行が「`push` か `pull,push`」と書く。登録の wake は順序を問わず、与えられた順のまま `wake:` の行に出る(`--wake push,pull` なら `push,pull`)。2 つの形だけに合わせる読み方をすると、`push,pull` の director が AskUserQuestion を使い、見張りの 1 行と Enter が推奨の案を選ぶ(証跡の T3)。危ない側に外れる。同じ段落は「登録の wake に push がある」とも書き、2 つの書き方が並んでいる。 | `.claude/skills/director/SKILL.md:484-486`。`internal/config/config.go:575-583`、`internal/org/director.go:527`、`internal/cli/org_director_report.go:314`。`internal/org/skill_split_questions_test.go:205-206` | 括弧を「`wake:` の行に `push` が含まれる(`push`、`pull,push`、`push,pull`)」にする。テストの断片も合わせる。 |
| LOW | maintainability | L6 計画の Scope と Risks は、見張りの 1 行が開いた問いに入る問題を実機で確かめ、見つかれば tech-debt に記録する、と約束している。S1 の証跡の T3〜T5 はそれが起きる(1 行と Enter が開いた問いを閉じる)ことを示したが、この差分に `docs/tech-debt/README.md` の変更はなく、台帳に AskUserQuestion や開いた問いの行もない。skill の回避(push の director は番号つきの選択肢)は入ったので、残る負債は「push の director が AskUserQuestion を使えない」ことと、見張りが開いた問いを見分けない実装。 | `docs/plans/active/2026-11-01-director-split-questions.md` の Non-goals の 2 つ目と Risks の 2 つ目。`docs/tech-debt/README.md`(grep で該当なし) | 1 行を足す: 見張りの push の 1 行と Enter が、pane で開いた AskUserQuestion を閉じる。回避は skill の決まり。直すなら、見張りが pane の画面を読んで開いた問いがあるときは打たない、または wake_cmd 側の工夫。trigger は push の director を AskUserQuestion で動かしたくなったとき。`/sync-docs` の範囲と重なるので、そちらで足してもよい。 |
| LOW | readability | L7 「merge を待つ issue」の 2 で、「follow-up のないブランチで、まだ merge されていないものは、『起票と記録』の 4 のとおり先に記録してから待つ」の文が、「印の行がなければ、その issue の open の follow-up がすべて `ready` のときだけ聞く」の直後にある。「follow-up のないブランチ」は印の行の `after:` にしか出ない語なので、直前の文(印の行がない場合)の続きに読むと、指すものがない。順は印の行がある場合の中に入れるべき。 | `.claude/skills/director/SKILL.md:1279-1286` | 文の順を、印の行があるとき(`after:` のブランチごとの条件と、follow-up のないブランチの記録)、印の行がないとき(open の follow-up がすべて `ready`)、両方(`waiting` が残れば聞かない)にする。テストの断片の順(`:527-531`)も合わせる。 |
| LOW | maintainability | L8 テストの置き場と固定の範囲。(a) `directorSkillSection` は `directorSkillSubsection`(`director_skill_sync_test.go:197`)の兄弟で、`skill_doc_followup_test.go` が使うが、定義は新しいテストファイルの 113 行にある。名前が近い 2 つの helper が別のファイルにあり、読み手は片方しか見つけない。(b) `TestDirectorSkill_AsksThePlanApprovalForksWithNumberedOptionsWhenTheWakePushes` が「`AskUserQuestion(Codex では番号つきの選択肢)` が残らない」を見るのは「計画の承認」の節だけ。「人への聞き方」が「どこでも従う」規則なので、skill 全体で同じ言い方が残らないことを固定すると、新しい問いが古い言い方で足されたときに落ちる(今の skill 全体では、該当の言い方は計画の承認以外にない)。 | `internal/org/skill_split_questions_test.go:113-137`、`:621-630`。`internal/org/director_skill_sync_test.go:197` | (a) `directorSkillSection` を `directorSkillSubsection` の隣に移す。(b) 否定の固定を skill 全体(`string(data)`)にかける。 |

## Positive notes

- 「人への聞き方」の規則が、実機の証跡の読み取り(T3〜T5)と 1 対 1 で合う。選択肢の上では 1 行が消えて director が wake の件に気づかない、という挙動まで書き、番号つきの選択肢の返事を待つあいだの見張りの 1 行は人の答えではない、と逆の向きの取り違えも止めている。
- 起票と記録の流れが、`after:` の印の行を 1 つの形で通している。起票は本文の最後の行と note に同じ行、引き継ぎと着手の問いは本文の行頭の印の行、文書の follow-up は `--state all` で同じ形を探す。`after: none` の扱い(記録するものがない、自動では聞かない)と、done の件を数えて人の決定を覆さない扱いも、引き継ぎと「起票と記録」の 4 で食い違わない。
- 記録の途中で止まった場合(起票のあと記録の前、1 件目のあと 2 件目の前)、止まっているあいだに待つブランチが merge された場合、打ち直しで 2 件にしない場合を、手順の文に持つ。
- 新しいコマンドは `pre_bash_guard.sh` に通り、`gh issue create` の題には `'` を入れない決まりがあり、本文は Write ツールでファイルに書く(`--body-file`)ので、題以外の自由文はコマンドの文字列に入らない。
- 印の行の正規表現を、skill の gh コマンドとテストが同じ定数から作る(`directorSplitMarkPattern`)ので、skill とテストの片方だけが変わることがない。
- `/spec` の手順番号とバッチの数(5 問ずつ)を、`/spec` skill の 4 つの写しの側で固定した(`TestDirectorSkill_AsksTheSpecQuestionsBeforeWritingASplitPlan`)。director の「手順 1〜3」という参照が古くなると落ちる。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 見張りの push の 1 行と Enter が、pane で開いた AskUserQuestion を閉じる(L6) | push の director は AskUserQuestion を使えず、番号つきの選択肢で聞く | skill の決まりで回避済み。実装を変えるのは Non-goal | push の director で AskUserQuestion を使いたくなったとき | `docs/evidence/director-ask-herdr-2026-11-01.md` |

_(この cycle では台帳に足していない。所見を直さずに残すなら、M1 は 1 行、LOW はまとめて 1 行にして `docs/tech-debt/README.md` に足す。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH は 0 件。M1 は、公開の repository で誰でも書ける issue の本文が director の shell に届く経路なので、PR の前に直すことを勧める。L1〜L8 は skill の文とテストの直し)
- Follow-ups:
  - M1・L1〜L5・L7 は skill の文の直しで、テストの固定も変わる。記録だけの直しの例外には入らないので、直したら `/self-review` から通し直す。
  - L6 は `/sync-docs` で台帳に足してもよい。
  - 実機で確かめていないこと: 見張りが本当に打つ wake、`ralph org director start` での起動、codex の director(証跡の「確かめていないこと」に既出)。`gh --jq` の `(?m)` は Go の regexp の意味で読んだだけで、gh を打っていない。
  - skill の挙動が変わる変更だが、release のノートの文は計画の Rollout にある。`/pr` で PR の本文に載せる。

## Run 2

- Date: 2026-11-01
- Reviewer: reviewer サブエージェント(`/self-review` の Run 2。パイプラインの cycle 1 の中)
- Scope: Run 1 のあとの `git diff 28d34ed2 HEAD`(S5 c72901e2 と記録の 5695396d、8 ファイル、+539/-207)と、ブランチ全体 `git diff 4fef6b8c...HEAD`(HEAD 5695396d)。見たのは 3 つ。(1) Run 1 の M1 と L1〜L8 が直ったか。(2) S5 の直しが持ち込んだ問題(印の行の正規表現と skill の文の食い違い、`gh issue list --json number,author,body --jq ... @tsv` の形、`gh api user --jq .login` の比べ方、節どうしの参照)。(3) ブランチ全体に Run 1 で見落とした所見がないか。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: Run 1 の M1・L1〜L8 と混ざらないよう、Run 2 の所見は R2-1・R2-2 とする。本文中の「Run 1 の L5」などは上の表の ID を指す。

### Run 1 の所見の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| M1 | 直った | 「起票と記録」に「印の行の形と読み方」(`.claude/skills/director/SKILL.md:633-658`)が入り、行全体の正規表現、本文にちょうど 1 行、書いた人が `gh api user --jq .login` と同じ、の 3 条件を課す。note は読んだ行を写さず組み直す。`--after-branch '<ブランチ>'` と `--note '<印の行>'` は単一引用符(`:623-627`)。本文から印の行を読む 4 経路(引き継ぎ `:1273-1288`、着手を聞く前 `:1310-1318`、書き換え `:1344-1350`、文書の follow-up の除外 `:1372-1385`)がすべてこれを参照し、「してはいけないこと」(`:1466-1468`)にも 1 項ある。Run 1 の probe の入力(`x';touch pwned;echo '`、`x;touch pwned`)と、`feat/x$(touch pwned)`、バッククォート、改行、`|` を含む行は、どれも正規表現に当たらない(Go の regexp で確かめた。テストの表にも前の 3 つがある)。当たる値は `[A-Za-z0-9._/,-]` だけなので、引用符の内外どちらでも shell の特殊文字を持たない。Run 1 が勧めた「merge-order の名前と一致するものだけを記録する」は取り入れていない。shell に届く経路は閉じているので、必須ではない |
| L1 | 直った | 「起票と記録」の 4(`:619-622`)に、待つブランチの PR がすべて merge 済みなら記録せず、すぐ着手を聞く文が入った。テストの断片もある |
| L2 | 直った | 頼る側を「頼る機能」、頼られる側を「頼られる機能」にそろえた(`:568`、`:598`、`:603`)。skill 内の「頼る」「頼られ」の 6 行を読み直し、残りの「頼る先」(`:568`、`:569`)と「頼る機能」(`:567`)は意味が合う。テストの 3 つの断片も合わせてある |
| L3 | 直った | 書き換えは `gh issue view` で本文を読み、`after:` の値だけを置き換えて `--body-file` に書く(`:1344-1350`)。印の行を 1 行に保ち、書き換えた行も形に当たることを確かめる |
| L4 | 直った | 1 に「起票した機能は 2〜6 でも読まない」(`:1379-1380`)、7 に括弧の手順の指す先と、文書の機能では前の質問と出力先の問いをしないこと、重なりで断られたときの扱い(`:1430-1434`)。7 の新しいテストの断片は skill の順に並ぶ |
| L5 | 直った | 「`push` が含まれる(`push`、`pull,push`、`push,pull`)」(`:485`)。テストの断片と `DirectorWakePull` の失敗文も合わせてある |
| L6 | 直った | `docs/tech-debt/README.md` の末尾に 1 行。5 列で、英語、末尾の改行あり。Related は計画の Non-goals と Risks、Run 1 の L6、実機の証跡を指す。plan のパスは `/pr` の archive-plan.sh が書き換える |
| L7 | 直った | 2 の文の順が「印の行があるとき、印の行がないとき、どちらでも waiting があれば聞かない」になった(`:1313-1318`)。テストの断片の順も合う |
| L8 | 直った | (a) `directorSkillSection` を `director_skill_sync_test.go:223` の `directorSkillSubsection` の隣に移した。(b) 「AskUserQuestion(Codex では番号つきの選択肢)」が残らないことを `string(data)` 全体で固定する |

### Evidence reviewed

- 差分: S5 の skill の変更(4 つの写しは同一)、テストの 3 ファイル、計画の Progress、台帳の 1 行。
- 写し: `/director` と `/org` の skill を `.claude` / `.agents` / `templates/base` の 3 方向で `cmp` した。すべて一致。
- 新旧の gh / ralph コマンド 4 つ(`gh issue list ... @tsv`、`gh api user --jq .login`、`gh issue view <番号> --json author --jq .author.login`、`ralph org followup add ... '<ブランチ>' ... '<印の行>'`)を `.claude/hooks/pre_bash_guard.sh` に通した。どれも出力なし(許可)。
- `[.number, .author.login] | @tsv` の形を、サンプルの JSON に jq で当てた。結果は `7<TAB>yoshpy-dev` の 1 行。gh の `--jq` は gojq なので、jq での確認にとどまる。
- 正規表現の値が ralph のコードと合う。id は `splitIDPattern`(`internal/org/split.go:63`、`{0,63}`)、slug は `identifierPattern`(`{0,29}`)を `maxFeatureOrgIDLen`(20、`internal/org/feature.go:74-79`)で切った `{0,19}`。機能のブランチは `feature.Type + "/" + feature.Slug`(`internal/org/feature.go:220`)で、Type は小文字の語の一覧(`internal/org/split.go:72`)なので、機能の org が持つブランチはすべて正規表現に当たる。
- Go の regexp の probe(使い捨て、確かめたあと消した): `...after: feat/x\r` は当たらない。同じ本文を `(?m)^ralph-split: ` で探すと当たる。`after: ..` は当たる(害はない)。`after: none,feat/x` は当たる(`none` という名のブランチとして)。
- 確かめていない: テストと静的検査(skill の規則どおり、走らせていない)。`gh issue list --json author --jq` を gh で打っていない。GitHub の web の編集欄が本文を CRLF にすること(R2-1)。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | R2-1 起票の経路が、読む側に断られる issue を作りうる。読む側は「行頭が `ralph-split: ` の行が本文にちょうど 1 行」で、その行が「行全体で」正規表現に当たることを求める。書く側の 2 は、機能の本文、`- Reserve:`、前文の要点、重なる org を書き、最後の行に印の行を書くとだけ言い、ほかの行の行頭を `ralph-split: ` にしないとは言わない。この repository のように印の行そのものを扱う仕事(director の印の行を直す機能など)では、機能の本文が `ralph-split: <id>/<slug> after: ...` の例を行頭に引くことがあり、そうなると 2 行で、director 自身が起票した issue を「2 行以上」として使わない。もう 1 つは行末の CR。`...after: feat/x\r` は正規表現に当たらない一方、緩い `(?m)^ralph-split: ` には当たるので、issue は一覧に出て形の確かめで落ちる。GitHub の web の編集欄で保存した本文は CRLF になる(おそらく。未確認)ので、人が起票のあとに web で本文を直した issue が同じ形で断られる。断られた issue は印の行のない扱いになり、「予約外の文書の follow-up」の 1 は、その機能をまだ merge されない機能として数える(`:1370-1371`)ので、集め始めが止まる。人には番号と理由が言われるので気づけるが、断られる原因を作るのは director の起票と人の通常の編集。失敗は閉じる側なので、MEDIUM にはしない | `.claude/skills/director/SKILL.md:601-607`(書く側の 2)、`:633`、`:646-647`(読む側)、`:654-658`(断られたときの扱い)、`:1370-1371`。probe: CRLF の行は当たらず、同じ本文の緩い探し方は当たる | 2 に「本文のほかの行は、行頭を `ralph-split: ` にしない(例を引くときは行頭を空けるか言い換える)」を足す。読む側の形に「行末の CR は行に含めない」を足す。テストの断片を 2 つ(`TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` の 2 と `TestDirectorSkill_UsesOnlyAMarkLineOfTheShapeFromAnIssueTheDirectorFiled`) |
| LOW | readability | R2-2 小さな文の直しが 5 つ。(a) `SKILL.md:487` が全角を 2 桁と数えて 108 桁で、周りの 67〜79 桁より長い。Run 1 の L5 の括弧を足した折り返しの直しが 1 行だけ残った。同じ文のテストの doc comment(`internal/org/skill_split_questions_test.go:175`)も 85 桁。pin は空白と改行を除いて比べる(`containsPhrase`)ので、折り返しを直しても落ちない。(b) 「起票と記録」の 4(`:622`)と印の行の読み方(`:643`)が「同じ節の 1」「同じ節の 2」と書く。指すのは `### merge を待つ issue` だが、書いてある場所は `### 起票と記録`で、そこの 1 は「待つブランチを決める」。直前に節の名前があるので読めるが、名前で書けば迷わない。(c) 印の行の読み方(`:635`)はブランチを「先頭が `-` でない」と書き、正規表現は先頭が `/` のものも断る。正規表現が正なので動作は同じ。(d) `/org` skill の 1 文(`.claude/skills/org/SKILL.md:1613`)の「note の行頭の」は、note 自体が印の行 1 行なので「note の」で足りる。(e) 「merge を待つ issue」の 3(`:1343-1350`)は「書き換えて、4 のとおり記録する。書き換えは、…」の順で、やり方の文が「記録する」のあとにある。順に打つ読み手は、やり方を読む前に記録へ進む | 各行は左の本文のとおり | (a) 487 行目を折り返し直し、テストの comment も合わせる。(b) 「「merge を待つ issue」の 1」「「merge を待つ issue」の 2 の着手を聞く前と 3 の書き換え」にする(テストの断片 2 つも)。(c) 「先頭が `-` と `/` でない」。(d) 「note の」。(e) 「書き換えは…」を先に置き、「そのあと、4 のとおり記録する」で終える |

### Positive notes

- 4 つの読む経路が、同じ段落の形と読み方を参照する。経路ごとに文を書き写していないので、条件を変えるときは 1 か所で済む。断られたときは「使わず番号と理由を人に言う」で止まり、記録も除外も黙って動かない。
- 正規表現のテストが、先頭の部分を `splitIDPattern` と `identifierPattern`、`maxFeatureOrgIDLen` から組んで比べ、危ない入力(`'`、`;`、`$(`、空白、先頭の `-`、末尾の `;` と `,`、21 字の slug)を表に持つ。当たる行のブランチと note が `ValidateFollowupAdd` を通ることも見る。
- 計画の Progress(5695396d)が、S5 の直しを Run 1 の ID(M1・L1〜L8)ごとに書き、consult から出た注意(gh のアカウントが変わる場合)も PR の Known gaps へ送ると残している。

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Run 1 の L6(見張りの push の 1 行と Enter が開いた問いを閉じる) | push の director は番号つきの選択肢で聞く | c72901e2 で台帳の末尾に 1 行入った。Run 1 の「この cycle では台帳に足していない」は解消 | 台帳の行のとおり | `docs/tech-debt/README.md` の末尾の行 |
| R2-1・R2-2 を直さずに残すなら | R2-1: 断られた issue のために文書の follow-up の集め始めが止まる。R2-2: 読みにくさだけ | LOW で失敗は閉じる側 | 印の行を扱う skill の次の変更 | この report の Run 2 |

_(R2-1・R2-2 を直さずに残すなら、plan は `/pr` で archive されるので、`/pr` の前に 1 行にまとめて `docs/tech-debt/README.md` に足す。pipeline は cycle 1 なので、cycle の末尾ではない。)_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2。Run 1 の M1 と L1〜L8 はすべて直っている。S5 の直しが持ち込んだのは LOW の R2-1 と R2-2 だけで、shell に届く経路は閉じたまま)
- Follow-ups:
  - R2-1 は skill の 2 文とテストの断片 2 つの直しで安い。直すなら skill とテストが変わるので記録だけの直しの例外には入らず、`/self-review` から通し直す。`/verify` の前に直せば、次の Run は R2-1・R2-2 の差分だけを見れば足りる。
  - `/verify` へ(この review の範囲外): 計画の D3(`docs/plans/active/2026-11-01-director-split-questions.md:119-128`)と AC4・AC7 は、読み取りの確かめ(形と書いた人)を書いていない。S5 の内容は Progress の記録(`:288`)にだけある。
  - `/pr` へ: S5 の consult が挙げた「director の gh のアカウントが変わると、自分で起票した issue が『ほかの人の issue』になる」は、Known gaps に書く。影響は番号と理由の通知だけでなく、R2-1 と同じ経路で、その機能が文書の follow-up の集め始めを止めること。この repo の作業では `gh auth switch` で gh のアカウントを切り替えることがあるので、起きる前提で書く。
  - 実機で確かめていないこと: `gh issue list --json author --jq` の実出力、web の編集欄の CRLF、見張りが本当に打つ wake、`ralph org director start` での起動、codex の director(Run 1 と同じ)。

## Run 3

- Date: 2026-11-01
- Reviewer: reviewer サブエージェント(`/self-review` の Run 3。パイプラインの cycle 1 の中)
- Scope: S6 の `git show ec988003`(9 ファイル、+133/-98)だけ。見たのは 2 つ。(1) Run 2 の R2-1 と R2-2 の (a)〜(e) が直ったか。(2) S6 の文とテストが持ち込んだ問題。S6 が触った段落の外は見ていない(ブランチ全体は Run 1・Run 2 で見た)。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: Run 1 の M1・L1〜L8、Run 2 の R2-1・R2-2 と混ざらないよう、Run 3 の所見は R3- とする。

### Run 2 の所見の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| R2-1 | 直った | 書く側: 「起票と記録」の 2(`.claude/skills/director/SKILL.md:607-609`)に「本文のほかの行は、行頭を `ralph-split: ` にしない(例を引くときは行頭を空けるか言い換える)」が入り、読む側の「ちょうど 1 行」と契約がそろった。読む側: 「行末の CR は行に含めない(取ってから確かめる)」(`:638`)。正規表現の行は S6 の差分に入っていない(`git show ec988003` で確認)ので、Run 2 の probe(CR つきの行は当たらず、CR を取れば当たる)がそのまま効く。テストの断片は 2 つとも skill の順に入った(`TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` の 2 と `TestDirectorSkill_UsesOnlyAMarkLineOfTheShapeFromAnIssueTheDirectorFiled`)。元の文を戻すと落ちる |
| R2-2 (a) | 直った | `:487-492` を 53〜74 桁に折り直した(周りは 67〜79 桁)。テストの doc comment も 74〜78 桁。ただし折り直した comment に短い行が 2 つ残った(R3-2) |
| R2-2 (b) | 直った | 「同じ節の」は skill 2 ファイル、テスト、計画のどこにも残っていない(grep)。「merge を待つ issue」の 1 の「すぐ人に聞く」は `:1297` にある |
| R2-2 (c) | 直った | 「先頭が `-` と `/` でない」(`:637`)は正規表現の先頭 `[A-Za-z0-9._]` と合う |
| R2-2 (d) | 直したが読みにくい | `/org` の文を「本文の行頭と note の ...」に変えた(R3-1) |
| R2-2 (e) | 直った | 「書き換えは…」を先に、「そのあと、4 のとおり記録する」を最後に置いた(`:1344-1356`)。順に打つと、本文を直してから記録するので、`gh issue edit` のあとで止まっても、引き継ぎが issue の印の行から記録し直せる。逆順だと、記録だけが新しく本文が古い状態が残る。テストの断片の順も合う |

### Evidence reviewed

- 差分: S6 の skill の変更(4 つの写しは同一)、`/org` の 1 文(4 つの写しは同一)、テスト 1 ファイル、計画の Progress(2042a8b6)。
- 写し: `/director` と `/org` の skill を `.claude` / `.agents` / `templates/base` の 3 方向で `cmp` した。すべて一致。
- 桁数: S6 が足した行の表示桁数を、全角を 2 桁として数えた。skill は 37〜74 桁、`/org` の 1 行だけ 80 桁(インラインコードで折れない)。
- 参照先: 「同じ節」の grep、「merge を待つ issue」の 1 の「すぐ人に聞く」、`:625` と一般の段落(`:660-661`)の note の規則を突き合わせた。
- 確かめていない: テストと静的検査(skill の規則どおり走らせていない)。web の編集欄が本文を CRLF にすること(Run 2 と同じ。R2-1 の直しは CR があってもなくても害がない形なので、確かめなくても足りる)。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | R3-1 `/org` の文が読み違えやすくなった。R2-2 (d) は「note の行頭の」を「note の」にするだけの直しだったが、S6 は「行頭」を本文の側に移して「issue の本文の行頭と note の `ralph-split: ...` の行が、待つブランチの全部を持つ」にした。並びが「本文の行頭」と「note の…の行」で釣り合わず、先頭の語が「本文の先頭」と読める。印の行は本文の最後の行(「起票と記録」の 2)で、行頭から書くのは行の中の話 | `.claude/skills/org/SKILL.md:1612-1614`(印の行は 1613)、テスト `TestOrgSkill_FollowupSubsectionSaysTheDirectorRecordsAFiledFeature` の断片 | 「issue の本文の `ralph-split: ...` の行(行頭から書く)と、note のその行が、待つブランチの全部を持つ」のように、本文の行と note の行を並べる。テストの断片 1 つも合わせる |
| LOW | readability | R3-2 R2-2 (a) の折り直しが、短い行を 2 つ残した。`TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` の doc comment の `// without '; one follow-up is recorded`(39 桁。次の行は `// per branch, not when ...`)と、`TestDirectorSkill_UsesOnlyAMarkLineOfTheShapeFromAnIssueTheDirectorFiled` の `// (maxFeatureOrgIDLen); it must`(32 桁。次の行は `// refuse a line ...`)。どちらも S6 が文を足して折り直したときに残った。R2-2 (a) と逆向き(長い行ではなく短い行)で、`gofmt` は直さない。同じ comment の周りは 70〜80 桁。3 つ目のテストの comment にも短い行が 2 つあるが(`// follow-ups stay done (no feature: line: step 1` など)、S3 以前からある | `internal/org/skill_split_questions_test.go:384`、`:466` | 2 か所を次の行と詰め直す。読みの問題だけなので、R3-1 と同じ commit で足りる |
| LOW | maintainability | R3-3 「merge を待つ issue」の 3 の書き換えのあと、note に何を書くかが 2 通りに読める。3 は「そのあと、「起票と記録」の 4 のとおり follow-up を記録する」(`:1355`)で終わる。4 は「note には 2 で書いた印の行を書く」(`:625`)で、この 2 は起票のときに書いた行、つまり書き換える前の `after:` を持つ。一般の段落(`:660-661`)は「確かめた id・slug・ブランチから組み直した印の行」を書けと言い、書き換えた行は `:1354-1355` で形を確かめるので、そちらを取るのが筋。どちらも読めるため、書き換え前の行を note に書くと、follow-up の `--after-branch` は新しいブランチ、note の `after:` は古いブランチ、という台帳ができる。台帳の note は `ralph org followup list` に出て、`/org` skill は「note の行が待つブランチの全部を持つ」と言う(R3-1 の文)。S5 から同じ形で、S6 が 3 の終わり方を直した場所なので挙げる。害は表示の食い違いにとどまる(引き継ぎは issue の本文を読む) | `.claude/skills/director/SKILL.md:625`、`:660-661`、`:1344-1356` | 3 に「note には書き換えた印の行を書く」を足す。または 4 を「note には印の行(起票のときは 2 で書いた行、書き換えたあとは書き換えた行)を書く」にする。テストの断片 1 つ |

### Positive notes

- 書く側と読む側の契約が、同じ段落の「ちょうど 1 行」と「本文のほかの行は行頭を印にしない」で閉じた。読む側が厳しくなったときは書く側も見る、という点が押さえられている。
- 3 の書き換えが、確かめに通らないときの扱い(書き換えず、記録もせず、番号と理由を人に言う)を、一般の段落と同じ 3 つの理由(形、2 行以上、書いた人)で書いた。理由の言い方が 2 か所でそろっている。
- 書き換えと記録の順が、途中で止まっても引き継ぎで直せる向きになった。

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R3-1〜R3-3 を直さずに残すなら | 表示と読みやすさだけ。shell に届く経路、記録の有無には影響しない | LOW で、skill の動作は変わらない | 印の行を扱う skill の次の変更 | この report の Run 3 |

_(R3-1〜R3-3 を直さずに残すなら、plan は `/pr` で archive されるので、`/pr` の前に 1 行にまとめて `docs/tech-debt/README.md` に足す。pipeline は cycle 1 なので、cycle の末尾ではない。)_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3。R2-1 と R2-2 は (d) の読みにくさを除いて直っている。S6 が持ち込んだのは表記と文の曖昧さだけで、shell に届く経路は閉じたまま)
- Follow-ups:
  - 3 件とも skill の文かテストの comment の直しで安い。R3-1 と R3-3 は skill とテストの断片が変わるので記録だけの直しの例外には入らず、直すなら `/self-review` から通し直す。`/verify` の前に直せば、次の Run は差分だけを見れば足りる。
  - 判断を残すだけの点(所見にしない): 3 で確かめに通らないとき、人が「待つ」と答えた約束は、会話と人への通知にしか残らない(「merge を待つ issue」の冒頭は、約束を台帳に記録すると言う)。S5 の consult の注意 2 を取り入れた形で、計画の Progress にもそう書いてある(意図した選択)。この場合も「merge を待つ issue」の 1 の記録(相手の org のブランチと平文の note)なら、印の行を使わずに台帳へ残せる。PR の Known gaps に「gh のアカウントが変わった director では、自分で起票した issue の待つ約束が台帳に残らない」と書くか、1 の記録に回すかを選ぶ。
  - 実機で確かめていないこと: Run 2 と同じ(`gh issue list --json author --jq` の実出力、web の編集欄の CRLF、見張りが本当に打つ wake、`ralph org director start` での起動、codex の director)。

## Cycle 2

- Date: 2026-11-01
- Reviewer: reviewer サブエージェント(`/self-review` のパイプライン cycle 2。cross-review cycle 1 の ACTION_REQUIRED を直したあとの回し直し)
- Scope: `git diff 101e7552..HEAD` の S9(9248d73b)、S10(e95f19e3)、計画の Progress 3 行(101cb03a)。見たのは 3 つ。(1) cross-review cycle 1 の triage #1〜#3 が直ったか。(2) 新しい文が「人に聞かないまま issue に着手しない」と「印の行の形と読み方」に食い違わないか。(3) `--jq` の件数の行が 2 か所で同じか、1 行目を件として読まないか。S9・S10 が触った段落と、それが参照する節だけを見た(ブランチ全体は Run 1〜3 で見た)。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: Run 1〜3 の M1・L1〜L8・R2-・R3- と混ざらないよう、この cycle の所見は C2- とする。cross-review の triage は `/cross-review` が同じパスに cycle ごとに書くので、triage の行は「cross-review cycle 1 の #n」と書き、この report の ID とは別に数える。

### cross-review cycle 1 の triage の状態

| triage | 状態 | 確かめたこと |
| --- | --- | --- |
| #1 引き継ぎで待つブランチがすべて merge 済みだと issue が浮かばない | 直った。ただし答えが残らない(C2-1) | `.claude/skills/director/SKILL.md:1295-1300` に、`after:` のブランチの PR がすべて `merged` で、その issue の follow-up が `followup list --all` に 1 件もなければ、記録せず、番号を伝えて 2 の問いですぐ着手を聞く、が入った。一部だけ merge 済みなら、merge されていないものを記録し、merged は待ち終えたものとして数える。この段落の冒頭(`:1280`)の「印の行は…確かめてから使う」の下にあるので、形と書いた人の確かめを通った issue にだけ効く。「起票と記録」の 4(`:620-624`)と同じ扱いで、2 の着手を聞く条件(`:1327-1332`、follow-up がなく merge-order で `merged` のブランチは聞いてよい)とも合う。4 つの写しは同一。テストの断片は `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` に 2 つ入り、skill の順に並ぶ |
| #2 start の断りは最初の重なりだけを出すので `after:` がほかのブランチを落とす | 直った | `:1356-1362` が `ralph status` の `reserved:` を読み直し、書き直した計画の `- Reserve:` と「機能ごとの出力先」の 1 の規則で比べて、重なる org を全部数える。ブランチを読むのは「起票と記録」の 1(org ごと、`feature:` の行)、書き換えるのは `after:` の全部、記録も全部(`:1372-1374`)。S9 が入れた「もう重ならなければ、書き換えずに start を打ち直す」は「人に聞かないまま issue に着手しない」(`:1379`)とぶつかっていたが、S10 が「書き換えず、そのことを人に伝えて start してよいかを聞く」に替えて解いた。ほかに `after:` を 1 つのブランチに書き換える文は skill にも `/org` skill(`:1613` は「全部」)にもない(grep) |
| #3 `--limit 1000` が切れると印の行がないと決める | 直った | 2 か所(`:1283` と `:1409`)とも `(length | tostring), (…)` で、絞り込む前の件数を 1 行目に出す。`gh` 2.102.0 で `--limit 3` を実際に打った(読み取りだけ): 出力は 1 行目 `3`、2 行目から `<番号>` タブ `<login>`。印の行のない 3 件では 1 行目だけが出る。`pre_bash_guard.sh` に同じコマンドを通すと出力なし(deny されない)。`length` は `--limit` で切ったあとの数なので、上限に届いたかを見分けられる。ヘルパー `directorSplitMarkIssuesCmd` が 2 か所のテストで共有されるので、skill のコマンドが片方だけ変わるとテストが落ちる |

### Evidence reviewed

- 差分: `git diff 101e7552..HEAD`(skill 1 ファイルが +30/-9、テストが +52/-28、計画が +3)。skill の 4 つの写し(`.claude`・`.agents`・`templates/base` の 2 つ)を `cmp` した。すべて一致。
- 桁数: S9・S10 が足した行の表示桁数を、全角を 2 桁として数えた。skill の本文は 17〜73 桁、テストの comment は 44〜79 桁。短い行は C2-5。
- 参照先: 「起票と記録」の 1・4・印の行の形と読み方(`:592-661`)、「merge を待つ issue」の 1〜3(`:1304-1379`)、「予約外の文書の follow-up」の 1(`:1391-1414`)を突き合わせた。`FollowupStore.Done`(`internal/org/followup.go:181-208`)は open の件なら状態によらず done にする。
- テストの読み方: `checkSkillPlaces` は空白と改行を除いて断片を順に探す(`internal/org/skill_doc_followup_test.go:59-79`)。そのため S10 の「start\n してよいか」のように行の折れ目が空白を挟む断片も当たる。断片の位置は skill の順と合う。
- 登録簿: この PR が足した `docs/tech-debt/README.md:417` の行を、S9・S10 のあとの HEAD に当てた(C2-6)。
- 確かめていない: テストと静的検査(skill の規則どおり走らせていない)、計画の Progress にある変異 13 個と 1 個が落ちたこと、issue が 1,000 本を超える repository での `gh` の実挙動、web の編集欄の CRLF。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | C2-1 引き継ぎの「すべて merge 済み」の問いは、答えがどこにも残らないので、director が入れ替わるたびに同じ issue を聞き直す。#1 の直しが引き継ぎの引き金にしたのは「follow-up が 1 件もない」状態。ほかの経路は、人が答えると open の follow-up をすべて done にする(3)ので、次の引き継ぎは done の件を見て聞かない。すべて merge 済みの経路は記録するものがない(「記録せず」)ので、答えても台帳は変わらず、次の引き継ぎで同じ状態に戻る。「要らない」と答えた issue は、issue が open のまま残り(`:1377-1378`)、閉じるのは人が頼んだときだけなので、頼まれるまで毎回聞かれる。「着手する」と答えて start した issue も、skill に `gh issue close` がなく、PR が issue を閉じるかは書いていないので、open のままなら毎回聞かれ、人がまた「はい」と答えれば同じ issue の org がもう 1 つ立つ。S5 から「起票と記録」の 4 にある同じ「記録せず」は、起票した会話の中で 1 回聞くだけだったので目立たなかった。#1 の直しが引き継ぎにも同じ扱いを足して、繰り返しになった。害は、聞き直しと(人が重ねて「はい」と答えたときの)重複した start。計画の「後に回す・要らないなら issue は open のまま残り、それが記録になる」は、すべて merge 済みの経路では「記録になる」が「毎回聞く」に変わる | `.claude/skills/director/SKILL.md:1295-1300`、`:620-624`、`:1348-1350`(open の follow-up を done にする)、`:1377-1379`。比べる対象: follow-up のある経路は done の件が残る | 答えを台帳に残す。すべて merge 済みの issue に人が答えたとき(「起票と記録」の 4 と引き継ぎの両方)、`ralph org followup add --issue <番号> --after-branch <merged のブランチの 1 つ>` を打ってすぐ `followup done <ID> --note <決めたこと>` を打つ。`Done` は open の件なら状態によらず done にする(`followup.go:181-208`)ので打てるはずだが、実際に打って確かめてはいない。そうすれば「follow-up が 1 件もない」は、まだ聞いていない issue だけになる。直さないなら、PR の Known gaps に「すべて merge 済みの issue は、答えたあとも director の引き継ぎごとに聞かれる。閉じるまで」と書き、「要らない」と答えた人に閉じるのは人の仕事と言う一文を skill に足す。どちらもテストの断片が変わる |
| LOW | maintainability | C2-2 テストの comment が `cross-review #1`・`#2`・`#3` と、cycle なしで書いている。triage の report は `/cross-review` が同じパス(`docs/reports/cross-review-triage-director-split-questions.md`。cross-review の skill `:119`)に cycle ごとに書き、その番号は cycle ごとに #1 から振り直される。cycle 2 の `/cross-review` が走ると、同じ `#3` が別の所見を指す。同じテストファイルの `self-review L5`・`R2-2` などは、self-review の report が追記だけで ID が cycle をまたいで一意なので、この問題がない。リポジトリには cycle を書く例がある(`internal/cli/org_merge_test.go:1061` の `cross-review cycle 2 #1`)。計画の Progress は「cross-review cycle 1」と書いていて、そちらは曖昧でない | `internal/org/skill_split_questions_test.go:68`、`:579`、`:586`、`:696`、`:802` | 5 か所を `cross-review cycle 1 #3` の形にする。cycle 2 の `/cross-review` の前に直せば、新しい triage の番号と混ざらない |
| LOW | readability | C2-3 引き継ぎの新しい文の数字が、どの節の番号か読み取れない。`:1298` は「「起票と記録」の 4 と同じく issue の番号を人に伝え、2 の問いと選択肢ですぐ着手を聞く」で、直前に「起票と記録」の 4 を名指すので、続く「2」が「起票と記録」の 2(本文を書く手順。問いはない)に読める。`:1300` の末尾「(2)」も同じ。同じ意味の「起票と記録」の 4(`:623`)は「「merge を待つ」issue の 2 の問い」と節名を付けている。テストの断片は曖昧な形をそのまま固定している | `.claude/skills/director/SKILL.md:1298`、`:1300`、`internal/org/skill_split_questions_test.go` の `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` の断片 | 「この節の 2 の問いと選択肢」「(この節の 2)」にする。テストの断片 2 つを合わせる |
| LOW | readability | C2-4 件数の行の説明が、2 か所で揃っていない。引き継ぎ(`:1286`)は「1 行目は絞り込む前の issue の数で、2 行目からが番号と書いた人」と言い、`--limit 5000` の例も付ける。「予約外の文書の follow-up」(`:1412`)は「1 行目は絞り込む前の issue の数」で止まり、データ行が 2 行目からとは言わない。件数の行は数字だけの 1 行で、実際の issue の番号と同じ値になりうる(`3` は #3 でもある)。2 行目の書いた人の列を読む `:652-653`(「2 列目」)は、1 行目の件数行に 2 列目がないことも言わない。読み違えると、件数を issue の番号として `gh issue view <件数>` を打ち、無関係な issue の本文が「形に当たらない」と人に報告される。害は誤報に近く、閉じた経路(確かめを通らない行は使わない)は崩れない | `.claude/skills/director/SKILL.md:1286`、`:1400-1414`、`:652-653` | `:1412` に「2 行目からが番号と書いた人」を足し(`--limit 5000` の例も)、テストの断片を 1 つ足す。より固くするなら、件数の行に印を付ける(`"total: " + (length | tostring)` のように、タブのない形)。コマンドの文字列が変わるので、写し 4 つとヘルパー 1 か所を同時に直す |
| LOW | readability | C2-5 S9・S10 の折り直しが、また短い行を残した(R3-2 と同じ形)。skill: `:1361` が 39 桁(「してよいかを聞く)。重なる org ごとに」)、`:1364` が 17 桁(「issue の本文を」)。周りは 59〜73 桁。テストの comment: `:697`(「// the human, and asks before it starts) and rewrites」)が 53 桁で、前後は 71〜76 桁。S10 は文を 1 つ替えただけで、折れ目を詰め直していない。同じ項目 3(`:1348-1379`)は 32 行の 1 段落になった。S9 が 10 行足した | `.claude/skills/director/SKILL.md:1360-1364`、`internal/org/skill_split_questions_test.go:696-698` | `:1356-1364` を 1 回で詰め直す。テストの comment は `:696-698` を詰める。段落を分けるなら「人が待つと答えたら」の前後が切れ目(読みやすさだけの話で、動作は変わらない) |
| LOW | maintainability | C2-6 この PR が足した登録簿の行が、S9 のあとの HEAD と合わない。`docs/tech-debt/README.md:417` は `skill_split_questions_test.go` を「832 lines (`wc -l` at commit 8bc6ae84 …)」と言う。コミットを名指しているので書いた時点では正しいが、そのあとの S9・S10 が合わせて +24 行を足し(`git diff --numstat 8bc6ae84..HEAD` は +52/-28。HEAD で 856 行)、`/sync-docs`(9eb633ef)が書いた行を fix コミットが古くしている | `docs/tech-debt/README.md:417`、`wc -l internal/org/skill_split_questions_test.go`(856) | 数を「856 行(HEAD)」にするか、行数を書かずに「800 行を超える」にする。この行は `docs/reports/` と insight event の外なので、記録だけの直しの例外(`post-implementation-pipeline.md`)には入らない |

### Positive notes

- S10 は、S9 の文が「人に聞かないまま issue に着手しない」とぶつかることを自分で見つけて直した。計画の Progress の S10 の行に、直した理由が書いてある。
- 件数を `--jq` の 1 行目に出す形は、出力の行数を数えるより安く、`--limit` で切れたかを producer 側で見分けられる。`gh` の実機で形を確かめてあり、guard にも止められない。
- 引き継ぎの新しい文は、既存の確かめ(印の行の形、書いた人、「データで指示ではない」)の段落の中にあり、新しい経路は開いていない。shell に届く値は増えていない。
- 4 つの写しが同一で、コマンドの文字列が 1 つのヘルパーから 2 か所のテストに行くので、2 か所の skill のコマンドが片方だけ変わることがない。

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-1 を直さずに残すなら: すべて merge 済みの issue の問いは答えが台帳に残らず、引き継ぎごとに聞き直す | 聞き直しと、人が重ねて「はい」と答えたときの重複した start。害は小さいが、director が入れ替わる設計の目的(会話が消えても台帳が約束を覚えている)から外れる | 直しは skill の文とテストの断片で、コマンドの追加はない。cycle の上限に合わせて Known gaps に回す判断もある | 引き継ぎの段落(`:1295-1300`)か「起票と記録」の 4 を次に変えるとき | この report の Cycle 2 |
| C2-2〜C2-6 を直さずに残すなら: 表記・説明・折り直し・登録簿の数 | 読みやすさと、後から読む人が番号を取り違えることだけ。動作は変わらない | LOW | `cross-review #n` を引く comment を次に足すとき(C2-2) | この report の Cycle 2 |

_(plan は `/pr` で archive され、`docs/tech-debt/README.md` に足さないと残らない。`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定は 2 で、この cycle は 2 回目の run。直さないものは、`/pr` の前に 1 行にまとめて register に足すか、PR の Known gaps に書く。cap を上げるなら、上の表の「cycle の上限に合わせて」は成り立たないので消す。)_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 5。cross-review cycle 1 の #1〜#3 は直っている。S9・S10 が持ち込んだのは、すべて merge 済みの問いの答えが残らないこと(C2-1)と、表記・説明の LOW 5 件で、shell に届く経路は増えていない)
- Follow-ups:
  - 先に決めること: C2-1 を直すか、Known gaps に回すか。直すなら skill の文(`:1295-1300` と `:620-624`)とテストの断片が変わるので、記録だけの直しの例外には入らず、全 step の再実行になる。cap が 2 のままなら再実行はできないので、cap を上げるか、Known gaps に回す。
  - C2-2 は、cycle 2 の `/cross-review` が triage を書く前に直すと、番号が混ざらない。comment だけの直しで安い(ただし例外の対象ではないので、直すなら C2-1 と同じ扱い)。
  - C2-3・C2-4・C2-5 は skill の文かテストの断片・comment の直しで、C2-1 を直すなら同じ commit で足りる。C2-6 は register の 1 行。
  - C2-1 と同じ根から出る点(別の所見にしない): 2 の問いの末尾は「人が会話にいなければ、問いは人が戻るまで待つ(follow-up は open のまま残り、次の報告でまた確かめる)」(`:1346-1347`)で、再び確かめる仕組みは open の `ready` の follow-up に頼る。すべて merge 済みの経路には open の follow-up がないので、人がいない間に出した問いは、次の引き継ぎまで確かめ直されない。この経路の状態が台帳にない、という C2-1 と同じ根で、C2-1 の直しで台帳に件を足すなら、その件の状態も合わせて決める。
  - 実機で確かめていないこと: Run 2・3 と同じ(web の編集欄の CRLF、見張りが本当に打つ wake、`ralph org director start` での起動、codex の director)に加えて、1,000 本を超える issue の repository での `gh issue list --limit 1000` の挙動、`followup done` を waiting の件に打つ実機の出力。

## Cycle 3

- Date: 2026-11-01
- Reviewer: reviewer サブエージェント(`/self-review` のパイプライン cycle 3。上限を 3 に上げたあとの 3 回目の run)
- Scope: `git show e775792d`(S12)と、その上の記録コミット 294791c1 の範囲。見たのは 3 つ。(1) cross-review cycle 2 の triage #1〜#3 が直ったか。(2) `- Depends on:` の番号を本文から読むところの安全さ(数字だけ、`gh issue view` の引数)。(3) 「今の `after:` のブランチの件」の数え方が引き継ぎ・「merge を待つ issue」の 2・3 で同じか、done と open の混在の扱いが「人に聞かないまま issue に着手しない」と食い違わないか。S12 が触った段落と、それが参照する節(「起票と記録」の 2・4、「印の行の形と読み方」)だけを見た。機械的な項目は実装側が見る範囲なので探していない。
- ID: Run 1〜3 の M1・L1〜L8・R2-・R3-、Cycle 2 の C2- と混ざらないよう、この cycle の所見は C3- とする。triage は `/cross-review` が同じパスに cycle ごとに書き直すので、triage の行は「cross-review cycle 2 の #n」と書く。

### cross-review cycle 2 の triage の状態

| triage | 状態 | 確かめたこと |
| --- | --- | --- |
| #1 `- Depends on:` の頼られる issue が open なら着手を勧めない | 直したが、番号の形が決まっていない(C3-1)。open の見方も粗い(C3-2) | `.claude/skills/director/SKILL.md:1357-1364` に、頼られる機能の issue が `OPEN` なら「後に回す」を推奨にして先頭に置く、が入った。コマンドは `gh issue view <番号> --json state --jq .state` で、`pre_bash_guard.sh` に通すと出力なし(deny されない)。番号は「数字だけの値」に限る |
| #2 引き継ぎで古い待ちの done の件を数えない | 直った。ただし 3 の文言は狭めていない(C3-3) | `:1292-1295` が `AFTER_BRANCH` の列が今の `after:` のブランチの件だけを数える。`followup list --all` の列は `ID STATE ISSUE AFTER_BRANCH ADDED_AT NOTE DONE_AT DONE_NOTE`(`internal/cli/org_followup.go:267-270`)で、書いてある列名は実在する |
| #3 done と open が混ざる issue を残さない | 直った。ただし note の使い方と org の確かめ方に穴がある(C3-4、C3-5) | `:1307-1314` が、混在は 3 で done を打つ途中で止まったものとして、done の件の `DONE_NOTE` を答えとして読み、残りの open の件を同じ note で done にする。`FollowupStore.Done` は open の件なら状態によらず done にし、done の件への再実行は何も変えずに終了コード 0(`internal/org/followup.go:181-208`、`/org` skill `:256`)なので、打ち直しで壊れない |

### Evidence reviewed

- 差分: `git show e775792d`(skill 4 ファイルが各 +20/-7 前後、テストが +28/-16 前後)。4 つの写しを `cmp` した。すべて一致。
- 桁数: S12 が足した行の表示桁数を、全角を 2 桁として数えた。skill の本文は 28〜80 桁、テストの comment は 65〜78 桁。短い行は項目の末尾(28 桁)と、次の行が 78 桁のコード片で折れた行(38 桁)だけで、段落の途中に置き去りの短い行はない。
- 並び: `checkSkillPlaces` は空白と改行を除いて断片を順に探す。新しい断片 5 つは skill の順(数え方の段落 → 一部 merge 済み → 混在 → `after: none`、2 の予約 → Depends on → 人がいないとき、3 の最初の文 → 途中で止まったとき → 1 件もないとき)に並んでいる。
- 参照先: 「起票と記録」の 2(`:604-605`)と 4(`:617-632`)、「印の行の形と読み方」(`:636-662`)、「機能ごとの出力先」の 2(`:560-572`)、`/org` skill の `- Depends on:` の定義(`:608`)と follow-up の節(`:1651-1664`)、`ralph status` の出力(`internal/cli/status.go` の `printStatusTableAllOrgs` は `org_id:` と `reserved:` を出し、`feature:` の行は出さない。`feature:` の行は `ralph org status --org-id` の `internal/cli/org.go:1908`)。
- 確かめていない: テストと静的検査(skill の規則どおり走らせていない)、計画の Progress にある変異 7 つが落ちたこと、実機の `gh issue view` の出力、`- Depends on:` に番号が入った本文を director が実際にどう書くか(skill に形がない。C3-1)。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | C3-1 `- Depends on:` の行の「issue の番号」の形が、書く側(producer)のどこにも決まっていないので、読む側(S12)の「数字だけの値」が、そのままでは動かないか、別の issue を引く。書く側は「起票と記録」の 2 に「`- Depends on:`(頼られる機能を起票したなら、先に起票したその issue の番号も)」とあるだけで、行の形がない。計画の `- Depends on:` は slug のカンマ区切り(`/org` skill `:608`)で、slug は英小文字・数字・`-`(`oauth2`、`auth-2fa` がありうる)。読む側は「番号は数字だけの値を使い、ほかの文字が混ざる値はコマンドに渡さない」。(a) director が `auth-token (#12)` や `auth-token #12` と書いた行を厳密に読むと、`(#12)` と `#12` は数字だけの値ではないので渡さず、推奨は変わらない。triage #1 が直したはずの「頼られる issue が未実装なのに着手を勧める」が、そのまま残る。(b) 数字を拾うと読み替えると、`oauth2` の `2` も拾い、`gh issue view 2` で無関係な issue の状態を引く。頼られる機能が複数のとき(「その issue」が単数)の扱いも決まっていない。shell に届くのは数字だけなので注入にはならない。害は推奨が誤ることで、人は答えを選べる(説明に頼られる issue が書かれないので、気づきにくい)。テストは文の断片だけを固定し(`skill_split_questions_test.go` の `:417` の断片、`:702-705` の断片)、行の形は固定しない | `.claude/skills/director/SKILL.md:604-605`、`:1357-1364`、`.claude/skills/org/SKILL.md:608`、`internal/org/skill_split_questions_test.go:417`、`:702-705` | 書く側で行の形を決める(例: 頼られる機能を起票したときだけ、slug のあとに ` #<番号>` を付ける。`- Depends on: auth-token #12, db #13`)。読む側は `#` のあとの数字の並びだけを取り(正規表現 `#([0-9]+)` を skill に書く)、複数あれば 1 つでも `OPEN` なら「後に回す」を推奨にして、open の issue を全部説明に書く。テストは行の形を 1 断片で固定する |
| LOW | maintainability | C3-2 「頼られる issue が open」は、頼られる機能が実装されたかを表さない。skill は issue を閉じない(「issue を閉じるのは人が頼んだときだけ」`:1409`、`gh issue close` は skill にない)。頼られる機能 B に「着手する」と答えて B の PR が merge されても、B の issue は PR の本文に `Closes #` がなければ open のまま(B の org を立てる `ralph org start` の leader が PR に closing keyword を書くかは、この skill も leader の prompt も決めていない)。そのあいだ、B に頼る A の問いは「後に回す」を推奨し続ける。「後に回す」「要らない」と答えた B も open のままなので、そちらは意図どおり。害は推奨が古くなることだけで、説明に B を書くので人は覆せる | `.claude/skills/director/SKILL.md:1357-1360`、`:1409`、`.claude/skills/pr/SKILL.md:33` | 直すなら、open の見方を B の follow-up と org の状態(B に `ready` の件がない、`ralph status` に B の org がない)に替えるか、skill に「B の issue が open でも B に着手して merge 済みなら数えない」と書く。直さないなら Known gaps に書く |
| LOW | consistency | C3-3 数え方の「同じ」が、引き継ぎの文言だけで、3 の文言は狭めていない。`:1292-1295` は「ここと「merge を待つ issue」の 2・3 で同じ」と言い切るが、3 の「印の行の issue に follow-up が 1 件もないとき」(`:1368`)は古い広い言い方のまま。引き継ぎは古い待ちの done の件(A)があり、今の `after:`(B)がすべて merge 済みで B の件がない issue を「聞く」に回し、答えを 3 のとおり残せと言う。3 だけを読むと、この issue には A の done の件があるので「1 件もない」に当たらず、`add` と `done` を打たない。答えが台帳に残らず、次の引き継ぎがまた聞く(C2-1 と同じ症状)。直前の括弧「(引き継ぎか「起票と記録」の 4 で、`after:` のブランチがすべて merge 済みだったとき)」が読み手を寄せるが、条件の語と食い違う。2 の「follow-up がなく、merge-order で `merged`」(`:1339-1340`)はブランチごとの言い方なので問題ない | `.claude/skills/director/SKILL.md:1292-1295`、`:1368-1370` | 3 を「今の `after:` のブランチの follow-up が 1 件もないとき」にして、テストの断片(`:755`)を合わせる |
| LOW | security | C3-4 台帳から読んだ note を、そのまま shell のコマンドに入れる。混在の経路は「`DONE_NOTE` を… 同じ note で `ralph org followup done` にする」(`:1309-1310`)。3 の例は `--note <決めたこと>` と引用符なし(`:1380`)で、director が自分の言葉で書く前提だった。台帳から読み戻した文は、`#`(引用符なしでは行の残りが注釈になり、note が途中で切れる)、`;`、`'`、`$(…)` を含みうる。`followup list` の出力は制御文字を `\x1b` の形に直すので、読み戻した文を貼ると別の文字列にもなる。同じ skill の印の行の経路は、行の形を正規表現で確かめ、値を単一引用符で囲み、note を確かめた部品から組み直す(`:626-627`、`:636-662`)のに、この経路にはそれがない。台帳を書けるのは director と、台帳のある state dir に shell が届く座席(leader を含む)。M1 と違い、書き手は組織の内側なので LOW | `.claude/skills/director/SKILL.md:1309-1310`、`:1380`、`internal/cli/org_followup.go:253-270`、`:626-627`、`:636-662` | 読み戻した文を貼らず、答えを 3 語(着手する、後に回す、要らない)の 1 つに読み取って、`--note '<3 語> (引き継ぎで残りを done)'` の形で組み直す。読み取れなければ人に聞く |
| LOW | maintainability | C3-5 「答えが着手するなら、機能の org が走っているかを `ralph status` で確かめる」(`:1310-1313`)は、混在の状態では常に「走っていない」になり、同じ穴の広いほうの窓を覆わない。(a) 3 の順は「すべて done → 計画を書き直す → 承認 → start」で、混在は done を打つ途中、つまり start の前なので org は立っていない。確かめは空振りで、実質「人にどうするかを聞く」だけ。(b) 同じ「着手すると答えたのに誰も start していない」状態は、すべて done にしたあと start の前(計画の書き直しと「分割計画の承認」で人を待つあいだ)にも起きる。ここは混在ではなく、今の `after:` に done の件が 1 つ以上あるので、引き継ぎは何もしない。issue は open のまま、org も open の件もなく、誰も気づかない。triage #3 が直すと決めた理由(黙って残る)は、(b) のほうが大きい窓に当たる。org の完了と未着手は、台帳の done の件と `ralph status` だけでは区別できないため、直しは要設計で、この cycle の上限には合わない。(c) `ralph status` は `org_id:` と `reserved:` を出し、`feature: <split>/<slug>` の行は出さない(`internal/cli/org.go:1908` は `ralph org status --org-id` の出力)ので、「その issue の機能の org」を選ぶ手がかりは org_id = slug(既定)しかなく、skill はそれを言わない | `.claude/skills/director/SKILL.md:1310-1313`、`:1365-1385`、`internal/cli/status.go` の `printStatusTableAllOrgs`、`internal/cli/org.go:1908` | 判断はリード。直さないなら Known gaps に「着手すると答えたあと start の前に director が入れ替わると、issue は誰にも拾われない」と書き、skill の確かめの文は「`ralph status` の org_id が slug と同じ org があるか」と手がかりを書く |
| LOW | maintainability | C3-6 この PR が足した登録簿の行が、S12 のあとのテストの行数と合わない(C2-6 の再発)。`docs/tech-debt/README.md:417` は `skill_split_questions_test.go` を「886 lines (`wc -l` after the fixes of the cycle 2 self-review …)」と言い、HEAD は 904 行。S12 が +18 行。「cycle 2 の self-review の直しのあと」と時点は書いてあるので偽ではないが、直しコミットが行数を古くするのはこれで 2 回目 | `docs/tech-debt/README.md:417`、`wc -l internal/org/skill_split_questions_test.go`(904) | 行数を書かず「800 行を超える」にする(直しのたびに古くなるため)。この行は記録だけの直しの例外(`post-implementation-pipeline.md`)に入らない |

### Positive notes

- 混在の扱いは、「着手は聞き直さない」と言いながら、答えが「着手する」で org がなければ人に聞くので、「人に聞かないまま issue に着手しない」(`:1410`)とは食い違わない。start に届く道は、人の答えと承認の両方を通る。
- `gh issue view <番号> --json state --jq .state` は読み取りだけで、番号が数字だけなら shell に届く値は増えない。`- Depends on:` の行を読む相手は、印の行と違って書いた人を確かめないが、読むのは状態だけで、出力は推奨の向きを変えるだけなので、閉じた経路は崩れていない。
- 数え方の段落は、使う列名(`AFTER_BRANCH`、`DONE_NOTE`)が実在し、混在の経路の `followup done` が打ち直しに強いことを、コードで確かめてある。テストの comment は `cross-review cycle 2 #n` の形で、C2-2 の指摘を踏まえている。

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C3-1 を直さずに残すなら: `- Depends on:` の番号の形が決まっておらず、triage #1 の直しが動かない可能性がある | 頼られる機能が未実装でも「着手する」が推奨される(人は覆せる) | cap 3 の最後の run。直しは skill の文とテストの断片で、記録だけの直しの例外には入らない | 起票の本文の形を次に変えるとき | この report の Cycle 3 |
| C3-5(b): 着手すると答えたあと start の前に director が入れ替わると、issue は誰にも拾われない | 黙って残る。org の完了と未着手を台帳と status だけで区別できない | 要設計。台帳に「着手を決めた」状態を持つか、start を done の前に置くかを決める必要がある | director の引き継ぎの設計を次に変えるとき | この report の Cycle 3 |
| C3-2〜C3-4、C3-6 を直さずに残すなら | 古い推奨、台帳の文の貼り直し、行数の古さ。動作の害は小さい | LOW | `- Depends on:` か `followup done` の経路を次に変えるとき | この report の Cycle 3 |

_(cap は 3 に上げてあり、この run が 3 回目。上の表のうち直さないものは、`/pr` の前に 1 行にまとめて register に足すか、PR の Known gaps に書く。plan は `/pr` で archive され、ここに書いただけでは残らない。)_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 5。cross-review cycle 2 の #1〜#3 は直っている。S12 が持ち込んだのは、`- Depends on:` の番号の形の未定義(C3-1)と、表記・説明・窓の広さの LOW 5 件で、shell に届く値は増えていない)
- Follow-ups:
  - 先に決めること: C3-1 を直すか。直すなら skill の文(`:604-605` と `:1357-1364`)とテストの断片が変わり、記録だけの直しの例外には入らないので全 step の再実行になる。cap は 3 で、この run が 3 回目。直さないなら cap を上げるか、PR の Known gaps に回す。
  - C3-3 は 1 語の直しで、C3-1 を直すなら同じ commit で足りる。C3-6 は register の 1 行。
  - C3-5(b)は要設計なので、Known gaps に回す判断が現実的。リードの決定事項。
  - 実機で確かめていないこと: Run 2・3、Cycle 2 と同じ(web の編集欄の CRLF、見張りが本当に打つ wake、`ralph org director start` での起動、codex の director、1,000 本を超える issue の repository)に加えて、`- Depends on:` に番号が入った本文を director が書いたときの実際の形。

## Cycle 4

- Date: 2026-11-01
- Reviewer: reviewer サブエージェント(`/self-review` のパイプライン cycle 4。4 回目の run)
- Scope: `git diff 3edfa501..HEAD` の S14(09c9f34d)と S15(ac5a0578)。見たのは 4 つ。(1) cross-review の 3 回目の run(triage は 3edfa501 の版)の #1・#2 が直ったか。(2) S15 のブランチごとの規則(記録の直前に merge 済み)が「記録より前の merge は数えない」(#270 の D5、`/org` skill の「merge を待つ issue」)と食い違わないか。(3) ブランチ名の使い回しで古い merge を待ち終えたと数える道がないか。(4) 「人に聞かないまま issue に着手しない」と食い違わないか。S14・S15 が触った段落と、それが参照する節(「機能ごとの出力先」の 3、「起票と記録」の 1・4、「merge を待つ issue」の引き継ぎ・2・3)だけを見た。機械的な項目は実装側が見る範囲なので探していない。
- ID: 前の節の ID と混ざらないよう、この cycle の所見は C4- とする。triage は `/cross-review` が同じパスに run ごとに書き直すので、triage の行は「cross-review の 3 回目の run の #n(3edfa501)」と書く(C4-4 参照)。

### cross-review の 3 回目の run の triage の状態

| triage | 状態 | 確かめたこと |
| --- | --- | --- |
| #1 最初の start の断りのあと、重なる org を数え直さない | 直った。数え直しに空の結果の枝がない(C4-3) | 「機能ごとの出力先」の 3(`.claude/skills/director/SKILL.md:582-585`)が、聞き直す前に「起票と記録」の 1 のとおり全部数え直し、説明に全部書く、と言う。「起票と記録」の 1(`:594-598`)は先頭で `ralph status` を打ち直して `reserved:` を `- Reserve:` と比べる。start のエラーが最初の 1 つだけを出すことは `reservationDecision`(`internal/org/reserve.go:539-585`。org_id 順で最初に重なった org だけを名指しして返す)で確かめた |
| #2 add のあと done の前に止まると、承認した着手が黙って失われる | 直った。ただし規則の理由の文が数え落としている(C4-1)。名前が紛らわしい(C4-2) | 「merge を待つ issue」の 2(`:1352-1365`)が、open の follow-up の STATE が `waiting` でも merge-order の entry が `merged` なら待ち終えたブランチに数える、と言う。引き継ぎ(`:1322-1325`)と 3(`:1406-1408`)はこの規則を指す。止まった件は、1 ブランチでも複数ブランチでも(add を打ったブランチは open の `waiting`、打たなかったブランチは follow-up なしの `merged`)、2 の判定で全部そろい、2 の問いで聞き、答えのあと 3 の「open の follow-up ごとに done」で閉じる。答えを聞く前に done にしない |

### Evidence reviewed

- 差分: `git diff 3edfa501..HEAD`(skill は 4 ファイルとも 42 行の変更、テストは 78 行の変更)。skill の 4 つの写しの追加・削除の行は一致し、root と `templates/base/` は `diff` で完全に一致、`.agents/` の写しも同じ。新しいコードブロックは足していない(散文だけ)ので、`pre_bash_guard.sh` に通すコマンドは増えていない。
- 桁数: S14・S15 が足した行の表示桁数を、全角を 2 桁として数えた。skill の本文は、段落の最後の行(48・20・35 桁)を除いて 53〜77 桁。テストの comment に 13・21・29・42・54 桁の行が 5 つある(C4-5)。
- 読んだコード: `internal/org/mergeorder_followup.go`(`:22` の使い回しの注、`followupMergeState` の `:330-334`)、`internal/org/mergeorder.go`(`pickMergePR` `:1000`、`classifyMergeTarget` の `MERGED` `:1072`)、`.claude/skills/org/SKILL.md:1629-1630`・`:1652-1656`。
- 確かめていない: テストと静的検査(規則どおり走らせていない)、計画の Progress にある変異 5 つ・9 つが落ちたこと、実機の `gh pr list` の出力。C4-1 は上のコードの読みで、走らせて確かめてはいない。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C4-1 S15 の規則の理由が、成り立つ原因を 2 つに数えているが、条件(open の件の STATE が `waiting`、entry が `merged`)は 3 つ目の原因でも成り立つ。理由の文は「`waiting` のまま `merged` のブランチは、確かめたあと記録の前に merge されたか、3 の答えの記録が途中で止まったもの」と言い切る(`:1358-1362`)。3 つ目はブランチ名の使い回し。merge-order の entry は番号が最大の PR だけを見る(`pickMergePR`)ので、前の org の PR #10 が merge 済みで、新しい org の PR がまだない名前は、entry が `merged` になる。follow-up の STATE は記録より前の merge を数えない(`:22`「A branch name can be used again」)ので、同じ名前が `waiting` になり、ledger 自身の DETAIL が「新しい PR の merge を待つ。来ないなら人に聞く」と言う(`:330-334`)。`/org` skill も「前に merge された同じ名前のブランチは数えない」(`:1629-1630`)。S15 は「来ない」と決めてしまい、まだ走っている org の機能に「実装が終わりました」と聞く。この組(entry が `merged` で件が `waiting`)を director 自身が作る道はない。「起票と記録」の 4(`:626-628`)が同じ entry の `merged` で記録を断るため。作れるのは、`followup add` が gh を読まず(`:1401`)どのブランチにも打てる点を使った手での記録や、別の director の記録。害は、問いの言い方の誤りと、答えのあと 3 が件を done にして待ちが消えること。start は予約の重なりで断る(「人に聞かないまま着手しない」は保たれる)。同じ述語(entry の `merged`)は「起票と記録」の 4 と引き継ぎ(`:1307`)にもあり、S15 が足したのは読む側の同じ誤差 | `.claude/skills/director/SKILL.md:1358-1362`、`:626-628`、`internal/org/mergeorder_followup.go:22`、`:330-334`、`internal/org/mergeorder.go:1000`、`:1072`、`.claude/skills/org/SKILL.md:1629-1630` | 条件に「そのブランチを `feature:` の行に持つ org が今走っていない」(`ralph status` の org を `ralph org status --org-id` で引く)を足すか、理由の文を「少なくとも」にして使い回しを Known gaps に書く。足りるのは 1 文。直さないなら Known gaps(「起票と記録」の 4・引き継ぎ・S15 の 3 か所が同じ述語。名前の使い回しで古い merge を待ち終えたと数える) |
| LOW | naming | C4-2 「記録の前に merge 済み」(follow-up がなく、entry が `merged`)と「記録の直前に merge 済み」(open の件が `waiting`、entry が `merged`)が、1 字違いで別の述語を指す(`:1355-1358`、`:1364`、`:1407`)。「直前」は止まった場合に合わない。3 の add と done が止まるのは、すべて merge 済みと分かったあとで、その merge は何日前でもよい。director が「直前」を時間の幅として読めば、古い merge の止まった件を拾わない(2 の列挙のうち「3 の答えの記録が途中で止まったもの」と矛盾する) | `.claude/skills/director/SKILL.md:1355-1358`、`:1364`、`:1406-1408` | 時点ではなく観測できる形で呼ぶ(例: 「open の件が `waiting` なのに、ブランチが `merged`」を 1 つの言い方にして、引き継ぎ・2・3 とテストの断片がそれを使う)。「直前」は理由の文だけに残す |
| LOW | consistency | C4-3 S14 の数え直しに、結果が空のときの枝がない。「merge を待つ issue」の 3(`:1423-1425`)は「もう重なる org がなければ、書き換えず、そのことを人に伝えて start してよいかを聞く」と言うが、S14 はそれと「同じ」と書きながら(`:597-598`)、その枝を写していない。「機能ごとの出力先」の 3 では、start が重なりで断ったあと数え直して 0 件なら、「issue に起票」を推奨し、説明に書く org がない。「起票と記録」の 1 では 0 件なら `after: none` の起票になり(`:604-607`)、人は「着手を自動では聞かない」と言われるだけ。数え直しは S14 で足したので、この道(聞いているあいだに重なる org が終わる)は S14 で初めて通る。以前は古い一覧のままだった。害は小さい(人は伝えられるので、頼めば start できる) | `.claude/skills/director/SKILL.md:582-585`、`:594-607`、`:1423-1425` | 「起票と記録」の 1 に「数え直して重なる org が 1 つもなければ、起票せず、そのことを人に伝えて今すぐ着手を勧めて聞き直す」を足し、「機能ごとの出力先」の 3 はそれを指す。直さないなら Known gaps |
| LOW | consistency | C4-4 テストの comment の `cross-review cycle 3 #n`(4 か所)は 3 回目の run を指すが、その triage の見出しは `Cycle: 2/3` で、次の `/cross-review`(`cycle-count.json` は cycle 3、cap 3)は同じパスを `Cycle: 3/3` で書き直す。書き直したあと、`cycle 3` の `#1`・`#2` はその 4 回目の run の行を指す。前の comment の `cycle 1`・`cycle 2` は見出しと合っていた。`cycle 3 #2` の 1 か所は行をまたいで書いてあり(855-856 行)、grep にも掛からない。計画の Progress は「cross-review 3 回目の run(codex、3edfa501 の triage、Cycle 2/3)」と run と commit で書いている | `internal/org/skill_split_questions_test.go:352`、`:393`、`:647`、`:724`、`:855-856` | 「cross-review third run (3edfa501) #n」のように run と commit で書く。次の `/cross-review` の前に直す(あとからでは指す先が曖昧になっている) |
| LOW | maintainability | C4-5 comment の折り返しの置き去りの短い行が 5 つ(R3-2・C2-4 の再発)。`skill_split_questions_test.go:398`(21 桁 `// depends on a filed`)、`:647`(54 桁)、`:730`(13 桁 `// gone. Then`)、`:856`(29 桁。cross-review の引用が 855-856 行で切れる)、`:858`(42 桁) | 同ファイルの上記の行。桁数は全角 2 桁で数えた | 段落を詰め直す。直さなくても動作に害はない |

### Positive notes

- 「人に聞かないまま issue に着手しない」とは食い違わない。S15 の道は、判定、2 の問い、人の答え、3 の done の順で、問いの前に done にしない(`:1324-1325`)。start に届くのは「着手する」の答えのあとだけ。
- 「記録より前の merge は数えない」の `ready` の定義は変えていない。S15 は `ready` の外に「待ち終えた」という director の側の数え方を足しただけで、STATE の列を書き換えず、`/org` skill の `:1652-1656`(merge 済みなら記録せずすぐ聞く)と同じ向き。
- S14・S15 は新しい読み込み(本文や台帳の文をコマンドに入れる道)を足していない。ブランチは merge-order の出力と ledger の列から取る。
- テストの変更は、断片の更新と、旧い言い方(「`waiting` のブランチが残っていれば聞かない」)が消えたことの否定の確認 1 つ。`containsPhrase` は空白と改行を除いて比べるので、skill の側で行をまたいで書かれても否定の確認は空振りしない。旧い言い方は S15 の前の skill にそのまま入っていた。

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C4-1: 3 か所(「起票と記録」の 4、引き継ぎ、「merge を待つ issue」の 2)が、entry の `merged` を「その待ちの PR が merge された」として読む。名前の使い回しで古い PR だと誤る | 「実装が終わりました」の誤った問い。start は断るので着手は起きない | 3 か所に同じ述語があり、直しは skill の 3 か所とテストの断片に及ぶ。上限の最後の run | follow-up の述語を次に変えるとき、または名前の使い回しが実際に出たとき | この report の Cycle 4 |
| C4-3: 数え直しが 0 件のときの枝がない | 起票した issue に `after: none` が付き、着手を聞く人がいない(人には伝える) | LOW、skill の文とテストの断片 | 「起票と記録」の 1 を次に変えるとき | この report の Cycle 4 |

_(この run は上限(cap 3)の最後の self-review。直さないものは、`/pr` の前に 1 行にまとめて Known gaps に書く。plan は `/pr` で archive され、ここに書いただけでは残らない。C4-4 は次の `/cross-review` の前なら 1 行の直しで足りる。)_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5。cross-review の 3 回目の run の #1・#2 は直っている。S14・S15 が持ち込んだのは、理由の文の数え落とし(C4-1)、紛らわしい名前(C4-2)、空の結果の枝(C4-3)、引用の曖昧さ(C4-4)、折り返し(C4-5)で、shell に届く値は増えていない)
- Follow-ups:
  - C4-1 と C4-3 は skill の文とテストの断片が変わり、記録だけの直しの例外には入らないので全 step の再実行になる。cap は 3 で、上限の最後の run。直さないなら PR の Known gaps に回す。リードの決定事項。
  - C4-4 は comment だけの直しだが、`internal/org/skill_split_questions_test.go` を変えるので、同じく例外には入らない。次の `/cross-review` が triage を書き直す前に直すと指す先が曖昧にならない。
  - 実機で確かめていないこと: 前の節と同じ(web の編集欄の CRLF、見張りが本当に打つ wake、`ralph org director start` での起動、codex の director、1,000 本を超える issue の repository)に加えて、名前を使い回したブランチで merge-order の entry と follow-up の STATE が実際に割れること(コードの読みだけ)。

## Cycle 5

- Date: 2026-11-01
- Reviewer: reviewer サブエージェント(`/self-review` のパイプライン 5 回目の run。上限を 4 に上げたあとの最後の run)
- Scope: `git diff 9b1175c3..HEAD` の S16(a7e7e4bd)と S17(cfbfe147)。見たのは 4 つ。(1) cross-review の 4 回目の run の #1〜#3 が直ったか。(2) 「起票と記録」の 4 のただし書き(走っている機能の org のブランチは `merged` でも数えない)が 3 か所で同じに効くか。(3) `./scripts/ralph-worktree.sh state-path org-<slug>` の使い方が script の help と合うか。(4) `--repo <owner>/<repo>` が gh の issue と pr のコマンドすべてに付き、決め方の sed が正しいか。S16・S17 が触った段落と、それが参照する節だけを見た。機械的な項目は実装側が見る範囲なので探していない。
- ID: 前の節と混ざらないよう、この run の所見は C5- とする。triage は `/cross-review` が同じパスに run ごとに書き直すので、triage の行は「cross-review の 4 回目の run の #n(9b1175c3)」と書く。

### cross-review の 4 回目の run の triage の状態

| triage | 状態 | 確かめたこと |
| --- | --- | --- |
| #1 ブランチ名の使い回しで、merge-order が前の PR を `merged` と出し、今の待ちを済んだと数える | 直った。ただし同じ述語の 4 つ目の場所が残っている(C5-1) | ただし書きは「起票と記録」の 4(`.claude/skills/director/SKILL.md:629-634`)に置かれ、引き継ぎ(`:1321-1323`)と「merge を待つ issue」の 2(`:1383-1385`)が指す。3 か所とも「走っている機能の org のブランチは `merged` でも merge 済みと数えない」で同じ向きに効く。「数えない」になったブランチは、4 では記録に回り、引き継ぎでは「PR もまだ merge されていない」の枝で記録に回り、2 では待ち終えに数えない。待つブランチは「起票と記録」の 1 で走っている org の `feature:` の行から取るので、使い回しがなければこの条件は働かない。働くのは、同じ名前の新しい org が走っていて新しい PR がまだないあいだだけ |
| #2 worktree を作ったあと spawn の前に予約を取られると、start が worktree を残し、書き直した分割計画の start を止める | 直った。ただし理由の文が残る場合を 2 つに限っている(C5-2) | 「merge を待つ issue」の 3(`:1439-1450`)が、書き直す前に `state-path org-<slug>` のファイルを確かめ、あれば slug を変えるか要らないかを人に聞く。`state-path <task-id>` は `scripts/ralph-worktree.sh` の help(`:13`)と合う。引数は 1 つで、記録の有無によらずパスを出す(`state_path`、`:66-73`)ので、「出すパスにファイルがあれば」の言い方で合う。start が org_id を slug にするのは `--org-id` を渡さないとき。skill の start のコマンド(`:580`)は渡さない。`org-` の前置きは `featureWorktreeID`(`internal/org/feature.go:163`)。`canonical_ref` に分割計画のパスが入る(`featureCanonicalRef`、`:178`)ので、別の id で書き直した計画は必ず断られる。直し方は `/org` skill の「start が拒否するもの」(slug を変えるか、要らなければ `cleanup`)と同じ。`cleanup` を人に頼まれないまま打たない点も合う |
| #3 fork の checkout で gh の既定が upstream だと、issue のコマンドが別の repository を指す | 直った。ただし host を落とす(C5-3)。固定は断片の写しだけ(C5-4) | gh の issue のコマンドは 11 か所すべてに `--repo <owner>/<repo>` が付く(`:508`、`:623`、`:669`、`:1305`、`:1310`、`:1371`、`:1404`、`:1462`、`:1468`、`:1500`、`:1508`)。`gh pr view` は 1 か所(`:1529`)で付く。ほかの `gh` の出現(`gh api user`、`gh pr merge`、`gh pr checks`)は director が打つ issue の読み書きではない。テストの断片も 12 か所すべてに `--repo` を持つ |

### Evidence reviewed

- 差分: `git diff 9b1175c3..HEAD`(skill は 4 ファイルとも 116 行の変更、テスト 2 ファイル、計画 3 行)。skill の 4 つの写しは `cmp` で同じ。
- 桁数: S16・S17 が足した行の表示桁数を、全角を 2 桁として数えた。skill の本文は最大 81 桁(`:1450`)で、折り返しの崩れはない。テストの comment に 17〜22 桁の行が 4 つある(C5-5)。
- 実行して確かめたこと: (1) skill の `sed` を、URL の形 18 通りに打った。`https://o/r`、`.git` の有無、末尾の `/`(`r.git/`、`r.git//`)、`git@github.com:o/r(.git)`、`ssh://git@host/o/r.git`、`ssh://git@host:22/o/r.git`、`https://user:tok@host/o/r.git`、`git://`、`git@github.com.emu:o/r.git`、`github.com:o/r`、名前に `.git` や `.github.io` を含むもの。どれも `o/r` を返した。返さなかったのは URL でないもので、`/Users/x/repo` は `x/repo`、`file:///tmp/o/r.git` は `o/r` を返す。host は常に落ちる(C5-3)。(2) skill が打たせる 6 つのコマンド(`sed` の行、`state-path`、`gh issue` の view・edit・create、`gh pr view`)を `.claude/hooks/pre_bash_guard.sh` に通した。どれも出力なし(deny されない)。(3) `gh issue view --help` で `-R, --repo [HOST/]OWNER/REPO` を確かめた。
- 読んだコード: `internal/org/feature.go`(`StartFeature` の順: `spawnPrecheckErr` `:238`、`checkFeatureWorktreeReuse` `:250`、`Ensure` `:253`、`Spawn` `:271`。spawn が止まったときの「worktree と branch が残る」の文 `:272-274`)、`internal/org/mergeorigin.go`(`Arg` `:28`、`ReadMergeOrigin` の拒否)、`scripts/ralph-worktree.sh`(help と `state_path`)、`.claude/skills/org/SKILL.md` の「start が拒否するもの」。
- 確かめていない:テストと静的検査(規則どおり走らせていない)、計画の Progress にある変異 10 個が落ちたこと、fork の checkout や GHES の origin で実際に gh を打つこと。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | consistency | C5-1 「merge を待つ issue」の 1 の「そのブランチの PR がもう merge されていれば記録せず、issue をすぐ人に聞く」(`:1353`)が、ただし書きの外にある。「起票と記録」の 4 の最後の文は「「merge を待つ issue」で merge 済みと数えるときも同じ」(`:633-634`)と節の全体にかかると言うが、1 にはそれを指す文がなく、テストの断片も 4・引き継ぎ・2 の 3 か所だけ。計画の S17 の行は、1 の使い回しの穴を「PR の Known gaps」と書く。skill の文(同じ)と計画の記述(穴)が食い違い、director は 1 を読むときに「起票と記録」の末尾まで戻るとは限らない。1 の経路は、人が「X の PR の merge を待つ」と決める場合で、ブランチは merge-order が出す名前(走っている org の機能のブランチを含む)。使い回しの名前なら、新しい PR の前に C4-1 と同じ「すぐ人に聞く」へ行く。「merge 済みなら記録せず聞く」の述語は同じ節の 4 か所にあり、3 か所だけが直った | `.claude/skills/director/SKILL.md:1353`、`:633-634`、`internal/org/skill_split_questions_test.go`(ただし書きの断片は `FilesTheIssueWithAMarkLine...` の step 4、`TakeoverRecords...`、`AsksWhetherToStart...` の 3 つ)、計画の Progress の S17 の行 | 1 に「ただし、「起票と記録」の 4 のただし書きに当たるブランチは数えない」の 1 文を足し、断片を 1 つ足す(S16 の 3 か所と同じ形)。直さないなら、「同じ」の文を「起票と記録」の 4・引き継ぎ・2 に絞って 1 を Known gaps にする。どちらにしても skill と計画の言い方をそろえる |
| LOW | doc accuracy | C5-2 S17 が書き直した 3 の括弧の文は、worktree が残る場合を 2 つに限る(「残るのは、…別の org が予約を取って leader の spawn で断られたときと、…PR が merge されずに閉じたとき」、`:1439-1443`)。コードでは、worktree を作ったあとの spawn が止まる理由はすべて残す。`r.Outcome` が spawned でも idempotent でもなく `r.Err` があれば、理由によらず「the worktree … and its branch … stay」を付ける(`internal/org/feature.go:271-274`)。予約の取り合い以外の spawn の失敗(driver の失敗、権限の envelope の断りなど)でも残る。害は小さい。director が打つ確かめは条件付きでなく(「書き直す前に確かめる」)、この文は理由の説明だから。ただ、起票の理由は「start が予約の重なりで断った」ことが多く、その場合の文は「ふつう残さない」と読めて、確かめを飛ばす理由になる。断りの文を閉じた一覧にしたのは、前の run の C4-1 と同じ形(原因を 2 つに数えた) | `.claude/skills/director/SKILL.md:1439-1443`、`internal/org/feature.go:238-274` | 「残るのは」を「worktree を作ったあとに start が止まったとき(leader の spawn の断りと失敗。確かめのあとに別の org が予約を取った場合を含む)と、…」にする。または原因の一覧をやめ、「前の start が worktree を残していることがある」だけにする |
| LOW | consistency | C5-3 `<owner>/<repo>` の決め方(`:687`)が、ralph 自身の origin の読み方より緩い。(a) host を落とす。`ralph org merge` と merge-order は `-R <host>/<owner>/<repo>`(`MergeOrigin.Arg`、`mergeorigin.go:28`)で読むが、skill の `sed` は host を捨てる。gh の既定の host が origin と違う checkout(github.com と GHES に両方ログインしていて origin が GHES、など)では、`--repo o/r` は別の host の同名 repository を指す。S16 の前の素の `gh issue ...` は checkout の remote から host を引けたので、この場合は前より悪い。(b) origin が URL でないとき(ローカルのパス、`file://`)、ralph は「repository なし」として断る(`ReadMergeOrigin` の `MergeOriginNoRepository`)が、`sed` は `/Users/x/repo` から `x/repo` を作る。gh が 404 を返せば止まるが、同名の repository が github.com にあれば、その repository に `gh issue create` を打つ。どちらも fork の checkout という S16 の想定の外。害は、github.com 以外か、ローカルの origin のときだけ | `.claude/skills/director/SKILL.md:678-688`、`internal/org/mergeorigin.go:28`、`:89-108`、probe の結果(Evidence reviewed の (1))、`gh issue view --help` の `[HOST/]OWNER/REPO` | 規則に「origin の host が github.com で、URL の形が上の 3 つのときだけ。それ以外(別の host、ローカルのパス、`file://`)は `--repo` を決めず、そのことを人に言う」の 1 文を足す。host を保つ書き方(`<host>/<owner>/<repo>`)にする手もあるが、`git@github.com.emu:` のような ssh の別名の host は gh が引けないので、1 文のほうが安い。直さないなら Known gaps |
| LOW | tests | C5-4 S16 の規則の「必ず付ける」と、`sed` の「`.git` があってもなくても読める」は、テストでは文字列の断片として固定されているだけ。`directorSplitOriginRepoCmd` は skill に入っていることを比べるだけで、一度も打たれない。`directorSplitRepoFlag` を使うのは 2 か所(`directorSplitMarkIssuesCmd` と規則の断片)で、残り 10 か所の断片は `--repo <owner>/<repo>` を直に書く。13 個目の `gh issue ...` が `--repo` なしで足されても、既存の断片はどれも落ちない(今の 12 か所は断片で固定されているので、今は漏れがない)。`directorSplitMarkLinePattern` は正規表現を表に打って確かめているので、同じ流儀が `sed` にも使える | `internal/org/skill_split_questions_test.go:83-84`、`:93`、`:526-531`、`:562-566` | skill 全体の `gh issue (create\|list\|view\|edit)` と `gh pr view` の出現がすべて `--repo <owner>/<repo>` を含む、という走査のテストを 1 つ足す。`sed` は `exec.Command("sh", "-c", ...)` で C5-3 の probe の 18 通りの表を打つ。ファイルは 800 行を超えていて、台帳の行(`docs/tech-debt/README.md` の 417 行目)に「次にテストを足すとき」の引き金がある。次の変更のときでよい |
| LOW | maintainability | C5-5 前の run で「直さない」と決めた C4-4・C4-5 が、S16 でそのまま増えた。(a) comment の折り返しの置き去りの短い行が 4 つ増えた(`skill_split_questions_test.go:429` の 22 桁 `// when every branch's`、`:529` の 22 桁、`:662` の 17 桁、`:760` の 17 桁)。(b) comment の `cross-review cycle 4 #n` が 5 か所(`:79`、`:426`、`:528`、`:759`、`:903`)。4 回目の run の triage の見出しは `Cycle: 3/3`、`cycle-count.json` は cycle 3・cap 4 で、見出しに `cycle 4` はない。5 回目の `/cross-review` は同じパスを書き直すので、このあと `cycle 4 #1`〜`#3` も、前の `cycle 3 #1`・`#2`(`:362`、`:403`、`:679`、`:757`)も、別の run の行を指す。skill の本文は折り返しの崩れがない(足した行は最大 81 桁で、段落の途中の短い行は長いコードの記述の前だけ) | 上記の行 | 直すなら 1 commit の comment の詰め直しと、`cycle N` を「cross-review の N 回目の run(<triage の commit>)」にする置き換え。ただし 5 回目の `/cross-review` の前でないと指す先が決まらない。直さないなら Known gaps の 1 行 |

### Positive notes

- 3 か所のただし書きは、条件(「走っている機能の org のブランチ」)、読み方(`ralph status` の org ごとに `ralph org status --org-id <org>`)、効果(`merged` でも merge 済みと数えない)が同じ言い方で、食い違わない。引き継ぎの文は、後ろの「PR もまだ merge されていない」の枝より前に置かれ、数えない枝がそのまま記録に回る順になっている。
- 使い回しでない通常の場合(待つブランチは走っている org のもので、古い PR がない)は、merge-order が `merged` と出さないので、ただし書きは何も変えない。過剰に止める道は見つからなかった。
- S16・S17 は、本文や台帳の文を shell に入れる道を足していない。`<owner>/<repo>` は手元の git の設定の値で、issue の本文のように外から書ける値ではない。`state-path` の引数は skill が持つ slug で、ralph の識別子の形。
- `cleanup` を人に頼まれないまま打たない、という線は `/org` skill の案内と同じで、director が worktree を消す道は開いていない。
- 差分は skill の 4 つの写しと対応するテストの断片に閉じ、写しは同じ。前の run で出した「新しいコードブロックの guard」は、`sed` の行を含む 6 つのコマンドで deny されないことを確かめた。

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C5-1: 「merge を待つ issue」の 1 の「もう merge されていれば記録せず聞く」は、名前の使い回しのただし書きの外 | 使い回しの名前の PR を待ち終えたと数え、まだ立っていない機能の完了を聞く。start は予約の重なりで断る | 上限の最後の run。1 文と断片 1 つで直るが、skill の 4 つの写しとテストに及ぶ | follow-up の述語を次に変えるとき、または名前の使い回しが実際に出たとき | この report の Cycle 5(C4-1 の残り) |
| C5-3: `--repo` の決め方が host を落とし、URL でない origin を読み違える | github.com 以外の host の origin や、ローカルの origin で、別の repository を指す。fork の checkout の想定では起きない | 想定の外の環境のときだけ。直し方に選択がある(host を保つか、1 文で止めるか) | GHES の利用者が director を使うとき | この report の Cycle 5 |

_(この run は上限(cap 4)の最後の self-review。直さないものは、`/pr` の前に 1 行にまとめて Known gaps に書く。plan は `/pr` で archive され、ここに書いただけでは残らない。)_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5。cross-review の 4 回目の run の #1〜#3 は直っている。S16・S17 が残したのは、同じ述語の 4 つ目の場所(C5-1)、閉じた一覧の理由の文(C5-2)、host と URL でない origin の読み(C5-3)、断片だけの固定(C5-4)、前の run の C4-4・C4-5 の増加(C5-5)で、shell に届く値は増えていない)
- Follow-ups:
  - 前の run の C4-1 は、S16 で 3 か所が直った。PR の Known gaps に書く C4-1 は「1 か所(C5-1)と、新しい org が PR なしで終わった場合」に狭めるか、C5-1 を直して閉じる。C4-3(数え直しが 0 件のときの枝)は直っていない。C4-2・C4-4・C4-5 は直していない(C5-5 に増えた分を足した)。
  - C5-1 と C5-2 は skill の文を変えるので全 step の再実行になる。上限は 4 で、これが最後の run。直さないなら C5-1〜C5-5 と、上の C4-2・C4-3・C4-4・C4-5 を PR の Known gaps の 1〜2 行にまとめる。リードの決定事項。
  - 実機で確かめていないこと: 前の節と同じに加えて、fork の checkout や GHES の origin で `--repo` つきの gh を打つこと、名前を使い回したブランチで ただし書きが実際に働くこと(コードと skill の読みだけ)。
