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
