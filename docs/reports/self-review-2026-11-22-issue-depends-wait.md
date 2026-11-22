# Self-review report: issue-depends-wait

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `2a069b50` から HEAD `2407bb35` までの `git diff origin/main...HEAD`(12 ファイル)のうち、diff の質を見るもの。director skill(「起票と記録」の 1〜5 と #364 の例、「merge を待つ issue」の引き継ぎ・2・3)、org skill の例外 1 文、仕様 FR-9 の項目、`internal/org/skill_issue_depends_test.go`(新規)と `skill_split_questions_test.go` の固定 5 か所。skill の文は、手順として読む agent が迷う所、周りの #365 の文との食い違い、コマンド名と JSON のキー名、止まる・繰り返す所を見た。テストの実行、静的解析、仕様との突き合わせ、文書のずれは見ていない

## Evidence reviewed

- `.claude/skills/director/SKILL.md` の diff の全 hunk(230 行)と、周りの文: 「機能ごとの出力先」の 1〜3、「人への聞き方」、「起票と記録」の全文、「merge を待つ issue」の全文(引き継ぎ、1〜3)
- skill の 4 写しは `md5` で director 同士、org 同士とも同一(`.agents` と `templates/base` を含む)
- skill が名指す名前をコードと照合した
  - merge-order の JSON: `entries`(`mergeorder.go:238`)、`branch`・`class`・`binding`、`binding` の `split`・`feature`・`issue`(`:326-340`、`issue` は `int64` の `omitempty`)
  - `ralph org status` の `issue: #<番号>` の行(`internal/cli/org.go:2324`)。#365 のテスト(`org_start_issue_test.go:188`)が固定している
  - 機能のブランチは `Type + "/" + Slug`(`feature.go:449`)、Type の既定は `feat`(`split.go:99`、`:718`)
  - PR のないブランチの follow-up は `waiting` で DETAIL が `<branch> has no PR yet: ...`(`mergeorder_followup.go:315`)。`followup add` はブランチの形だけを確かめる(`followup.go:239-253`)
  - `ralph org followup list` は open だけ、done は `--all`(skill 2559 行付近の文と合う)
- 新しいテストファイルの全文と、`skill_split_questions_test.go` の diff。テストの組み立て(`skillOrderedFragmentProblems` は空白を除いた順序つきの部分文字列、`directorSkillNumberedStep` は先頭が空白でない行で切る)を読み、新しい段落と例が節の切り出しに影響しないことを確かめた
- `docs/tech-debt/README.md` を、diff が触れた所(director skill の「起票と記録」、`skill_split_questions_test.go`)で引いた。diff に台帳の hunk はない。行 426(`skill_split_questions_test.go` が 800 行を超えた)の Trigger は「このファイルに次のテストを足す、またはその節の読み手を変える」で、この PR は新しいテストを新しいファイル(381 行)に置き、既存ファイルは固定の文言だけを直した(1198 → 1202 行)ので満たさない

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | readability | 「`feature` だけが合うときは人に聞く」が、AC6 が防ぎたい場面のほぼ全部に当たり、聞く場面と答えの行き先が決まっていない。(a) ralph が作るブランチは `<Type>/<slug>` なので、前の別の分割計画の `feat/locks` の `binding.feature` は必ず `locks` で、(a)(b) に当たらなければ常に「feature だけが合う」になる。skill が「待つ」と決めている「`binding` がないか、`feature` が違う」は、ralph が作っていないブランチでしか起きない。(b) 2 の確かめ(`merged` なのに `waiting` の件を「merge と進捗の報告のたびに」読む)も同じ分岐に入るので、頼る先がまだ立たないあいだ、報告のたびに同じ問いが出る。人の答えを残す場所がない(follow-up の note は印の行で、答えを書く欄がない)ため、「前の分割計画の merge だ」と答えた人にも次の報告でまた聞く。(c) 問いの形がない。「人への聞き方」は選択肢で聞く場面を挙げていて、この問いは入っていない。選択肢も推奨もなく、push の wake がある director が AskUserQuestion を使うと、見張りの Enter が選択肢を選ぶ(同節)。(d) 例の最後の文は「(`feature` が `locks` なら人に聞く)、44 は `feat/locks` を待ち続ける」と、問いの答えにかかわらず待つと読める | `director/SKILL.md` 740-753(1 の「feature だけ」)、794-795(4)、2425-2426(引き継ぎ)、2509-2510(2)、836-839(例)、519-543(「人への聞き方」)。`feature.go:449` | 問う場所を、答えで動作が変わる 1 か所(起票で、頼る先のブランチを `after:` に入れる前)に絞る。報告のたびの確認(引き継ぎと 2)では「feature だけが合う」を merge 済みと数えず、聞かない(待ち終えていない。前の別の分割計画の merge は `waiting` のまま)。起票の問いには選択肢(「別の分割計画の merge(待つ、推奨)」「頼る先の機能の merge(待たない)」)を決め、「人への聞き方」の列挙に足す。例の最後の文は「人が前の分割計画の merge と答えれば、44 は `feat/locks` を待ち続ける」と答えを言う。固定するテストは 5 か所の fragment と例の 1 fragment |
| L1 | LOW | readability | 頼る先の merge の確かめが、「頼る先の機能」の待ちを決める 1 の中に定義され、そこを指す文が 4 か所で少しずつ違う言い方で書き直されている(起票の 4、引き継ぎ、2、org skill)。定義は「頼る先の slug」と「この分割計画」で書かれているのに、4 か所は「分割計画の機能の `<Type>/<slug>` に当たるブランチ(頼る先の機能のブランチなど)」と広げて使い、頼る先でない同じ分割計画の機能のブランチ(重なる org が終わって disband したあとの merged など)に (a) だけが効くことは書いていない。また (b) の番号の出どころ「起票のときはこの節の 3 で出た番号」は、読み手には今の手順の 3(1 のあとに来る)に見え、実際は頼る先の issue を先に起票したときの 3 の出力を指す。頼る先を issue に起票していない(今すぐ着手した、まだ起票していない)ときは番号がなく (b) は使えない、とも書いていない。走っている org が `feature` だけ合って (a)(b) に当たらないときの扱いも書かれていない(読み手は「走っていなければ」の側へ進み、同じ名前のブランチを待つので結果は同じだが、その旨はない) | 726-739(定義)、794-797、2425-2428、2509-2511(再掲)。732(「この節の 3」) | 確かめを「頼る先の merge の確かめ(ブランチ、分割計画の id、機能の slug、issue の番号か なし)」として 1 つの小節に切り出し、各所は小節の名前で呼ぶ。番号は「頼られる issue の行から読む。ない機能は (a) だけ」と言う。4 写しと固定するテストが 1 か所に集まる |
| L2 | LOW | readability | #364 の例が読みにくい。(a) 印の行 2 つを 1 つのコード枠に並べ、どの issue(44 と 45)の行かを枠のそばに書いていない。41〜43 の `after: none` の行は出していない。(b) 「ralph の repository の issue #364」と言いながら起票した issue を 41〜45 とし、この repository の今の番号(370 台)と合わない。例の番号と分かるが、読み手は `#41` を実在の issue と取りうる | 811-839 | 枠の中を `# issue 44` `# issue 45` のコメント行で分けるか、枠を 2 つにする。番号は `<N1>`〜`<N5>` か、明らかに架空の数にする。固定するテストは行の完全一致で拾っているので、枠の外の文だけ直せばよい |
| L3 | LOW | maintainability | org skill の 1 文は「merge-order の `binding` がその機能を名指す PR の merge だけを merge 済みと数え、…数えないときは follow-up を記録して待つ」と言う。director skill は (a) か (b) の 2 通りで名指し、`feature` だけが合うときは記録の前に人に聞く。org の文は (b)(issue の番号)を落とし、聞く分岐もなく、「数えないときは記録して待つ」と言い切る。org skill は leader と座席も読み、director の手順を直接は打たないので害は小さいが、2 つの skill が違うことを言う | `org/SKILL.md` 2436-2440、`director/SKILL.md` 726-753 | org の文を「(a) か (b) に当たる PR の merge だけを数え、`feature` だけが合うときは `/director` skill の手順で人に聞く」にする。M1 を直したあとの文に合わせる |
| L4 | LOW | maintainability | 3 の「移す」手順が、移さなくてよい issue まで選ぶ。選び方は「`followup list` の AFTER_BRANCH と、引き継ぎのコマンドで出した issue の印の行の `after:` に古いブランチがある issue ごと」で、`list` は open の件だけなので前者は良いが、後者は人がすでに答えた issue(「後に回す」「要らない」で done にした issue は GitHub で open のまま残る、と同じ節が言う)も選ぶ。そうなると `after:` を書き換えて新しい follow-up を記録し、新しいブランチが merge されたときに、答えた人に同じ issue をもう一度聞く。ほかに 2 点。(b) worktree の記録とぶつかったときの案内は「slug を変えるか、その worktree とブランチが要らないかを人に伝えて聞く」と slug を変える案を先に置き、今回の移す手順の費用(待つ側の全 issue を書き換える)を知らせない。(c) 元の機能の Type を「issue の本文の `- Type:` の行の値」と読むが、行の数と値の検査を言わない。印の行は正規表現で確かめるのに、この値は branch の先頭になる(`splitFeatureTypes` は 11 個、`split.go:96`)。本文の機能の記述が `- Type:` 行を含むと、どの行か分からない | 2579-2581、2586-2606(3)、`split.go:96` | (a) 選ぶ issue を、古いブランチに open の follow-up がある issue と、印の行の `after:` に古いブランチがあって今の印の行の follow-up が 1 件もない issue に絞る。(b) 案内に「slug を変えると待つ側を移す手順が要る」を足し、worktree の片付けを先に聞く。(c) 「行頭の最初の `- Type:` の行の値で、`splitFeatureTypes` のどれか。違えば分割計画の機能の値」と書く |
| L5 | LOW | maintainability | 引き継ぎの新しい知らせ「PR も org もないブランチを待つ follow-up の issue と待つブランチを人に伝える」は、毎回の引き継ぎで言い直す形で、#365 が閉じた issue の確かめに付けた「まだ人に伝えていないものだけ」の限りがない。伝える中身も待つ側の issue(44)と `feat/locks` だけで、そのブランチを作る issue(41、`after: none` で「着手を自動では聞かない」)を名指さない。人が受け取るのは「44 が `feat/locks` を待つ」で、41 を着手するまで 44 は進まないことへの手がかりがない。起票時の 3 の文(「44 は `locks` に頼るので `feat/locks` を待つ」)も同じ | 2455-2459、822(例の「人には、44 は」)、「起票と記録」3 | 知らせに待つ側の `頼られる issue: #<番号>` のうちまだ着手していないものを書き、「伝えた跡がなければ」と 1 回の限りを付ける(起票時の知らせにも番号を入れる)。#365 の「会話にも、要約した会話にも伝えた跡がないもの」の文と同じ言い方にする |
| L6 | LOW | maintainability | `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半は、既存のテストと同じことを固定する。PR のない follow-up が `waiting` で DETAIL が `<branch> has no PR yet` になることは、`mergeorder_followup_test.go:197`(`no PR is waiting`)と `internal/cli/org_merge_order_followup_test.go:156` がすでに固定している。前半の `.git` の確かめは一時ディレクトリの直下だけを見る。コメントの「どの git repository の外」の保証にはならず(`TMPDIR` が repository の中にあれば外ではない)、store を直接呼ぶので、skill が言う `ralph org followup add` の CLI の経路は通らない | `skill_issue_depends_test.go` 185-207 | 後半は既存のテストを指すコメント 1 行に変えて消す。前半は「store の `Add` はブランチの有無を見ない」と言い切るコメントにし、`.git` の stat を外す(CLI の経路が要るなら `internal/cli` のテストで) |

## Positive notes

- 4 写しが同一で、skill が名指すコード上の名前(JSON のキー、`issue:` の行、DETAIL、Type の既定、`merged` のクラス)は、すべてコードと合う。新しいテストは JSON のキーを `reflect` でコードの構造体のタグと突き合わせ(`checkMergeOrderJSONNames`)、Type の既定と `MergeClassMerged` も定数で突き合わせている。skill の名前だけが変わって固定が通る形にならない
- Go のコードは変えていない。新しいテストは新しいファイル(381 行)に置き、台帳の行 426 の Trigger を満たさない作り
- `merged` だけで数えない理由(過去の別の分割計画の同名ブランチ)と、(b) が要る理由(issue から着手した頼る先は分割計画を書き直すので `split` が元の id と違う)が、該当の文のそばに書いてある。読み手が規則を足し引きするときに根拠がある
- 周りの #365 の文との矛盾は見つからなかった: 「merge と auto_merge」と後始末の `binding.issue` の確かめ、`start --issue` の同じ番号の付け直し、「予約外の文書の follow-up」の起票した機能の除外は、どれも新しい文と別の面を言う
- 「待つブランチがなければ…着手を自動では聞かない」の文と、予約の重なりだけで決まる今の待ち(「重なる org が 2 つ以上なら、そのブランチを全部待つ」)は残っている(計画の Regression)
- 重なりと依存が同じブランチを指すときの重複、頼る先が 2 つのとき(45 の例)、頼る先が merge 済みで待たないとき(`after: none` になる)が、手順と例の両方に出ている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 頼る先の merge の確かめの文が、「起票と記録」の 1 の定義と、4 か所の言い直し(director の 4・引き継ぎ・2、org skill)に分かれている(L1) | 次に規則を変える人が 5 か所を揃える必要があり、4 写しとそれぞれの fragment でも揃える。1 か所の直し漏れは、前の分割計画の merge を数える方に倒れる | 計画の Scope が「起票と記録」の 1 に定義を置く形で、小節を切り出す変更は文の入れ替えが大きい | 頼る先の merge の確かめの次の変更、または M1 を直すとき | この report の L1・M1、`docs/plans/active/2026-11-22-issue-depends-wait.md`(Scope) |
| 3 の「移す」手順が、答え済みの issue も選び、slug を変える案を先に案内する(L4) | slug か Type を変えるまれな場合に、答えた人にもう一度聞く、または待つ側の全 issue を書き換える | 計画の Assumptions が、slug を変える経路は今ある書き換えの手順を残す、と許している | 「merge を待つ issue」の 3 の次の変更、または移す手順を使った報告 | この report の L4、同計画の Assumptions |

_台帳 `docs/tech-debt/README.md` には追記していない。依頼の範囲が report・計画の Progress・insight event の 1 コミットで、複数のレーンが台帳の末尾に追記して衝突するため。上の 2 行は、M1 を直さずに進めるときに `/pr` の前に台帳へ入れてほしい(M1 は debt ではなく直す所見)。_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 6)
- Follow-ups:
  - M1 は skill の文の直しで、コードの振る舞いは変わらない。ただし「feature だけが合う」問いは AC6 の中心の場面なので、cross-review の前に直すのが安い。直すなら 4 写し(`scripts/sync-skills.sh` で `.agents` を作る)と、固定する fragment(`skill_issue_depends_test.go` の 3 つのテストと `skill_split_questions_test.go` の 2 か所)を 1 回で直す。直したら `/self-review` 以降の再実行が要る(skill と Go のテストに触れるので記録だけの修正ではない)
  - L1・L3・L5 は M1 と同じ文を触るので、M1 を直すときにまとめる。L2・L6 は文とテストの小さな直し。L4 は台帳の行にして `/pr` に進んでよい
  - 直さずに進めるなら、PR の本文の Known gaps に「前の別の分割計画の同名ブランチが merge 済みのとき、director は報告のたびに人に聞き、答えを残さない」を書く
  - テストの実行と静的解析、AC の突き合わせは `/verify` と `/test` の担当
