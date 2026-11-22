# Self-review report: issue-depends-wait

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(`cycle-count.json` は cycle 2、既定の上限 2 の最後の周)。Run 1 は下にそのまま残す(見出しは 1 段下げた)。今の判定はこの節の Recommendation の行。所見の ID は Run 1 の M1・L1〜L6 と取り違えないよう `C2-` を付けた(テストのコメントが `self-review M1`・`L1` などで Run 1 を指している)
- Scope: origin/main `2a069b50` から HEAD `ff59a0e1` までの `git diff origin/main...HEAD`(19 ファイル)。重点は cross-review の ACTION_REQUIRED 2 件の直し `ff59a0e1`(director skill の 4 写しに各 +43/-7 行、仕様 1 行、`skill_issue_depends_test.go` の +141/-16 行)。`ff59a0e1` は全 hunk と周りの文を読んだ。それより前の部分は、Run 1 の所見が直ったかを現物で確かめる範囲と、`ff59a0e1` の文が頼る範囲だけ読み直した。テストは `go test -count=1 ./internal/org/ -run 'Skill|DirectorSkill'` を 1 回だけ走らせた(ok、1.089 s)。`./scripts/run-test.sh`、静的解析、仕様との突き合わせ、文書のずれは見ていない

### 確かめたこと

- `git show ff59a0e1` の全 hunk: director skill(「起票と記録」の 1 の (c) と最後の段落、同 4 の文、引き継ぎの 1 段落、「merge を待つ issue」の 2)、仕様 FR-9 の 2026-11-22 の項目の追記、`skill_issue_depends_test.go`(定数 6 つ、新しい `TestDirectorSkill_StartCheckCountsAReadyDependencyBranchOnlyByTheDependencyCheck`、`checkMergeOrderJSONNames` の 4 行、`Org.now` の確かめ)
- 周りの文: 「起票と記録」の 1〜5(skill 697〜835 行)、引き継ぎの全文(2456〜2530 行)、「merge を待つ issue」の 1〜3(2530〜2740 行)、「してはいけないこと」の先頭(gh は「ralph と gh の操作」に入り、`gh pr view` の読みを止める規則はない)
- 4 写し: director の 4 つの `md5` は同一(`9d2e3194…`)、org の 4 つも同一(`f6fdd1b5…`)
- skill の文をコードと照合した
  - `binding` は最新の結びつき: `collectMergeTargets`(`mergeorder.go:576-582`)が、完全な機能の `scope_reserved` ごとに `t.binding` を上書きする。`binding.ts` はその event の `ts`
  - `pr` は番号最大の PR: `pickMergePR`(`mergeorder.go:1108`、fork の PR を除く)。class `merged` は選んだ PR の state から決まる(`classifyMergeTarget`、`:1185-1187`)ので、`merged` の件には必ず `pr` がある
  - `binding.ts` の形: `Org.now`(`spawn.go:411`)は `.UTC().Format(time.RFC3339)`。`scope_reserved` を書く 3 か所(`spawn.go:1540`、`:1691`、`verbs.go:2010` の `reserveAgain`)はどれも `o.now()`
  - 同じ秒: `pickFollowupPRs`(`mergeorder_followup.go:355`)は `at.Before(added)` だけを「前」とするので、記録と同じ秒の merge は「後」。(c) の「より前でない」と同じ向き
  - gh の並び: skill 2791 行の `gh pr view <番号> --repo github.com/<owner>/<repo> --json body --jq .body` と同じ(番号が先、`--repo` が後)
  - note の語: `validateFollowupBranch` はブランチ名に `'` を許す。step 2 の `<ブランチ>` は分割計画の機能の `<Type>/<slug>`(Type は一覧、slug は検証済み)なので、単一引用符を壊さない
- 台帳の 561・562 行(Run 1 の L4・L6)が `ff59a0e1` で成り立たなくなっていないか: 行が言うこと(移す手順、Type の読み方、テストの重複)は `ff59a0e1` が触れていない。取り消した行も、残すべきなのに消えた行もない

### Run 1 の所見の行き先

| ID | 行き先 | 現物の確認 |
| --- | --- | --- |
| M1 | `03ced266` で直った | 問いは「起票と記録」の 1 で 1 回。follow-up のあるブランチは引き継ぎと「merge を待つ issue」の 2 で聞かない(今の skill の文で確認) |
| L1 | 直った | 確かめの規則は 1 に 1 か所で、テストが 1 回だけ出ることを固定する。`ff59a0e1` は (c) と gh のコマンドも同じ固定に入れた |
| L2 | 直った | 例の番号は `<locks の番号>` の形 |
| L3 | 直ったが、`ff59a0e1` で再びずれた | C2-L1 |
| L4 | 台帳 561 行(`/sync-docs` が入れた) | 行は残っている |
| L5 | 直った | 引き継ぎの知らせは「まだ人に伝えていないもの」だけ 1 回で、頼る先の issue の番号と `after: none` の注意を含む |
| L6 | 台帳 562 行(同上) | 行は残っている |

### `ff59a0e1` で見た点(依頼の観点ごと)

- **食い違い**: 文同士に、agent が逆の動作をする食い違いは見つからなかった。順序で読み違えうる所が 1 つあり、C2-L3
- **繰り返し(done と記録し直し、引き継ぎ)**: 記録し直した follow-up は、記録より後の merge だけで `ready` になる。そのあとの報告ではブランチは `merged` で STATE が `waiting` になり、「記録の直前に merge 済み」の道に入るが、(a)(b)(c) のどれかが落ちるので待ち終えたと数えず、done も記録し直しも打たない。報告のたびには繰り返さず、確かめに落ちる merge が新しく起きるたびに 1 回だけ。固定の文の done の件(`NOTE` は今の印の行、`DONE_NOTE` は固定の文)は、引き継ぎの除外で「done と open の混ざり」の規則(`DONE_NOTE` を人の答えと読んで残りの open を done にする)に入らない。この除外がないと、記録し直した follow-up が人の答えなしに done にされる。除外の文は混ざりの規則より前にある。なお、手元の時計が GitHub より遅れて記録の時刻が merge の時刻より前になると、記録し直した件がすぐ `ready` になり、時計が追いつくまで繰り返しうる。これは `ready` の比べ方(`added` と `mergedAt`)の性質で、この PR が作ったものではない
- **done → 記録し直しの順と、途中の停止**: 先に add すると「同じ issue とブランチの open の件があれば足さない」(4)に当たって足せないので、この順でよい。done のあとで止まったブランチには今の印の行の follow-up がなく、引き継ぎは固定の文の done の件を数えず、そのブランチを follow-up のないブランチとして記録し、頼る先の問いは聞かない。例外の置き場所に難があり、C2-L3
- **同じ秒**: (c) の「より前でない」は記録の比べ方と同じ向きで、`ts` も `mergedAt` も秒の単位。同じ秒の merge は数える。食い違いなし
- **gh の形**: 並びは 2791 行と同じ。`--jq .mergedAt` が null をどう出すかは確かめていない(class が `merged` の件では null にならない)
- **テスト**: (c) の文、gh のコマンド、時刻の形、前の試みの PR の説明、読めないときの文、step 2 の 7 つの文、引き継ぎの 3 文、note の語(step 2 と引き継ぎで同じ定数)を、順序つきの部分文字列で固定している。規則と gh のコマンドは skill 全体で 1 回だけ出ることも固定する。`ff59a0e1` の前の skill は (c) の文を持たないので、この固定は前の版で落ちる。`Org.now` の確かめ(JST の時計で UTC の `Z` の形)は、`binding.ts` の形をコードに結ぶ部分で、コードを変えれば落ちる。弱い所は C2-L4
- **実装側が挙げた範囲外の 3 つの危険**
  1. 同じブランチ名の別の分割計画の org が後から立つ: 最新の結びつきがその org になり、本物の頼る先の merge が (a)(b) に落ちる。step 2 では本物の `ready` を done にして記録し直し、人に PR の番号を伝える(気づける)。起票と引き継ぎでは、すでに merge された頼る先を待ち続ける。同じ名前のブランチが 2 つの org に結びつくには、先の org の worktree の記録を越えて `start` が通る必要があり、まれ。今は直さず台帳に入れる
  2. `reserveAgain` が `binding.ts` を merge より後にする: `reserveAgain` を呼ぶのは、disband と stop で自分の pane の close が失敗した補償(`verbs.go:1718`、`:1806`)だけ。`ralph org merge` は org の disband を待つので、補償の `scope_reserved` はふつう merge より前に書かれる。merge がその前に済んだ(人が GitHub で merge した、merge queue)ときだけ (c) が落ち、そのとき org は `running` に戻るので、「起票と記録」の 4 のただし書きでも数えない。今は直さず、1 と同じ台帳の行に入れる
  3. gh の呼び出しが報告ごとに 1 回増える: 呼ぶのは class `merged` で (a)(b) に当たるブランチだけでよいが、skill はそう書いていない。C2-L2 の (c)

### Findings (Run 2)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-L1 | LOW | maintainability | `/org` skill の 1 文(「merge を待つ issue」、4 写し)は、頼る先の merge の確かめを「(a) か (b) で名指す PR の merge だけを merge 済みと数える」と、規則の全部のように言う。`ff59a0e1` が足した (c)(その PR が結びつきより後に merge されたもの)と、`ready` の follow-up への同じ確かめは言わない。`ff59a0e1` は director の 4 写し、仕様、director のテストだけを直し、`/org` の文とその固定(`TestOrgSkill_FollowupSubsectionLetsADependencyWaitPastAnEarlierMerge`)に触れていない。Run 1 の L3 と同じ形で、2 つの skill が同じ規則の違う版を言う。`/org` の文だけを読む人は、(a) に当たる前の試みの PR の merge を数える。文は末尾で `/director` skill の 1 と 4 を指すので、害は小さい | `org/SKILL.md` 2436-2442(4 写し)、`skill_issue_depends_test.go` 706-730、`git show --stat ff59a0e1`(org の写しがない) | 文を「(a) か (b) に当たり、その PR が結びつきより後に merge されたものだけ」にするか、(a)(b) の列挙をやめて「`/director` skill の頼る先の merge の確かめに当たる merge だけ」にする。4 写しと固定の 1 か所 |
| C2-L2 | LOW | exception-handling | (c) の「時刻を読めない」に 3 つの穴がある。(a) 起票と引き継ぎでは、読めないブランチは「数えず、聞かずに待つ」で、4 が follow-up を記録する。follow-up は記録より後の merge だけを数えるので、すでに merge された本物の頼る先は、gh の一時的な失敗 1 回で、来ない merge を待つ状態になる。人には伝えるが、読み直しも、人が取れる手(「数える」と言う)も書いていない。step 2 の道は次の報告で読み直す(回復する)のに、起票と引き継ぎの道は回復しない。(b) 「読めなかったことを人に伝える」に回数の限りがない。step 2 は「次の報告で確かめ直す」ので、gh が失敗し続ければ報告のたびに伝えうる。Run 1 の L5 と、#365 の閉じた issue の確かめの「まだ人に伝えていないものだけ」と同じ形。(c) 時刻を読むのは (a) か (b) に当たる件だけでよいが、(c) の文はその順序を言わない。(a)(b) に落ちる件でも gh を打て、その失敗を「読めなかった」として伝える(step 2 では done にも記録し直しもしない)読み方ができる。報告ごとの gh の増加(上の 3)はこの読み方の費用 | `director/SKILL.md` 752-762(c)、786-788(4)、2614-2618(2) | (c) に「(a) か (b) に当たる件だけ時刻を読む」と「読めなければ 1 回読み直し、それでも読めなければ人に伝える(伝えた跡があれば言い直さない)」を足す。起票と引き継ぎでは、読めなかったブランチを記録して待つことと、人が「数える」と言えることを一緒に伝える |
| C2-L3 | LOW | readability | 引き継ぎの例外「ただし、頼る先の merge の問いは聞かない」は、それが直す規則(follow-up のないブランチを記録するとき、`merged` で `feature` だけが合えば 1 回聞く)より 13 行前にあり、規則の本文は例外を持たない(「follow-up のないそのブランチが…聞く」「聞くのはこの場合だけで、follow-up のないほかのブランチと…」)。規則の文だけを読む agent は、done にして記録し直す前に止まったブランチで問いを聞く。同じ段落の「(done の件も含めて)なく」も、固定の文の done の件を除く前の定義を持ち越して読まないと、除外と逆に読める。除外と例外は段落の前半に 1 度あるだけで、テストは両方の文を固定するが、置き場所と順序は固定しない | `director/SKILL.md` 2484-2490(除外と例外)、2497-2512(「done の件も含めて」と規則) | 規則の本文に「(2 で done にして記録し直す前に止まったブランチは除く)」を足し、「(done の件も含めて)」を「(固定の文の done の件を除く)」にする |
| C2-L4 | LOW | maintainability | テストの 2 点。(a) 新しい `Org.now` の確かめの 2 つ目(`time.Parse(time.RFC3339, directorDependsTimeExample)` が UTC を返す)は、1 つ目(`o.now()` が定数と等しい)が通れば定数は UTC の形なので、重ねて標準ライブラリを確かめているだけ。gh の `mergedAt` が同じ形であることは、コメント(「`pickFollowupPRs` が同じ layout で読む」)が言うだけで、何も固定しない。(b) ファイルは 607 行から 732 行になった。次にテストを足すと、global の目安の 800 行に届く。兄弟の `skill_split_questions_test.go`(1204 行)が台帳の行 426 の形に入った前例がある | `skill_issue_depends_test.go`(`ff59a0e1` の前は 607 行、今は 732 行) | (a) 2 つ目を外すか、`pickFollowupPRs` に UTC の `mergedAt` を通す確かめに替える。(b) 次にテストを足すときに、確かめ別にファイルを分ける |

### Positive notes (Run 2)

- (c) の根拠の文(`binding` は最新の結びつき、`pr` は番号最大の PR)は、`collectMergeTargets` と `pickMergePR` と一致する。cross-review の 2 件の道(前の試みの PR が最新の結びつきと組む、`ready` が同じ名前の別の機能の merge で立つ)は、どちらも (a)(b)(c) と「確かめ直して記録し直す」で塞がる
- done を先、記録し直しを後にした順序は、4 の重複の判定(open の件があれば足さない)と合う。途中で止まった状態は引き継ぎが拾い、除外の文が固定の文の done の件を「人の答え」と読むのを防ぐ
- 時刻を `binding.ts`(UTC、秒)と `mergedAt` で比べる向きは、`ready` の比べ方と同じで、同じ秒の扱いが 2 か所でずれない。`Org.now` を JST の時計で固定したテストが、`binding.ts` の形をコードに結んでいる
- 固定の文を step 2 と引き継ぎで同じ定数から引き、`ValidateFollowupNote` と `'` の有無まで確かめる。文を片方だけ変えるとテストが落ちる
- `ff59a0e1` は skill の 4 写し(`.agents` と `templates/base` を含む)を揃えている。仕様の項目にも (c) と再記録を足した

### Tech debt identified (Run 2)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 同じブランチ名の別の分割計画の org が後から立つか、`reserveAgain` が `binding.ts` を merge より後にすると、頼る先の本当の merge が (a)(b)(c) に落ちる。step 2 は本物の `ready` を done にして記録し直し(人に PR の番号を伝える)、起票と引き継ぎはすでに merge された頼る先を待ち続ける。`reserveAgain` は disband と stop の自分の pane の close の失敗の補償だけが呼び、merge は disband を待つので、前者もふつうは merge より前 | まれな 2 つの道で、頼る先が着手できると出ない | 直すには、merge-order の JSON が結びつきの履歴(最新だけでなく org ごと)か PR の merge の時刻を出す必要があり、コードの変更になる。この PR の範囲は skill の文 | 着手できると出ないという報告、または merge-order の JSON に結びつきの履歴か PR の merge の時刻が入るとき | この report の C2 の確かめ、cross-review triage(cycle 1 の 1 と 2) |
| `/director` skill の頼る先の確かめの小さな所見 4 つ(C2-L1: `/org` skill の 1 文に (c) がない、C2-L2: 時刻を読めないときの回復と伝える回数の限りと読む順序、C2-L3: 引き継ぎの例外がそれの直す規則と離れている、C2-L4: テストの重なる確かめ 1 つとファイルが 732 行) | いずれも、頼る先の確かめの共通の経路は変えず、読み方がずれる所と伝え方の所 | cycle 2 は上限の最後の周で、直すと 4 写しとテストと全パイプラインの再実行が要る | 頼る先の merge の確かめの次の変更、または `/org` skill のその項の変更 | この report の C2-L1〜C2-L4、台帳 561・562 行 |

_台帳 `docs/tech-debt/README.md` には追記していない(Run 1 と同じ理由で、依頼の範囲が report・計画の Progress・insight event の 1 コミット、複数のレーンが台帳の末尾に追記して衝突する)。上の 2 行は、次の `/sync-docs` か `/pr` の前に台帳へ入れてほしい。cycle 2 は上限の最後の周なので、ここで直さない所見は台帳の行になって初めて残る。_

### Recommendation (Run 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4)
- Follow-ups:
  - cross-review の 2 件の直しは、skill の文の上でも、テストの上でも成り立っている。LOW 4 件は、頼る先の確かめを使う道の共通の動作を変えないので、止める理由にしない
  - C2-L1・C2-L2・C2-L3 は同じ skill の同じ文を触る。直すなら 4 写し(`scripts/sync-skills.sh`)、`/org` の固定、director の固定を 1 回で変え、cap を上げて `/self-review` からの再実行が要る(skill とテストに触れるので、記録だけの修正の例外には当たらない)。直さずに進めるなら、上の台帳の 2 行を `/sync-docs` で入れ、PR の本文の Known gaps に「gh が一時的に失敗すると、すでに merge された頼る先を待ち続けることがある」と「同じ名前のブランチの別の分割計画の org が後から立つと、頼る先の merge を数えないことがある」を書く
  - テストの実行と静的解析、AC の突き合わせ、文書のずれは `/verify`・`/test`・`/sync-docs` の担当。`/sync-docs` は、`ff59a0e1` が足した (c) と再記録の文を、仕様と台帳で確かめる

## Run 1 (cycle 1)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `2a069b50` から HEAD `2407bb35` までの `git diff origin/main...HEAD`(12 ファイル)のうち、diff の質を見るもの。director skill(「起票と記録」の 1〜5 と #364 の例、「merge を待つ issue」の引き継ぎ・2・3)、org skill の例外 1 文、仕様 FR-9 の項目、`internal/org/skill_issue_depends_test.go`(新規)と `skill_split_questions_test.go` の固定 5 か所。skill の文は、手順として読む agent が迷う所、周りの #365 の文との食い違い、コマンド名と JSON のキー名、止まる・繰り返す所を見た。テストの実行、静的解析、仕様との突き合わせ、文書のずれは見ていない

### Evidence reviewed

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

### Findings

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

### Positive notes

- 4 写しが同一で、skill が名指すコード上の名前(JSON のキー、`issue:` の行、DETAIL、Type の既定、`merged` のクラス)は、すべてコードと合う。新しいテストは JSON のキーを `reflect` でコードの構造体のタグと突き合わせ(`checkMergeOrderJSONNames`)、Type の既定と `MergeClassMerged` も定数で突き合わせている。skill の名前だけが変わって固定が通る形にならない
- Go のコードは変えていない。新しいテストは新しいファイル(381 行)に置き、台帳の行 426 の Trigger を満たさない作り
- `merged` だけで数えない理由(過去の別の分割計画の同名ブランチ)と、(b) が要る理由(issue から着手した頼る先は分割計画を書き直すので `split` が元の id と違う)が、該当の文のそばに書いてある。読み手が規則を足し引きするときに根拠がある
- 周りの #365 の文との矛盾は見つからなかった: 「merge と auto_merge」と後始末の `binding.issue` の確かめ、`start --issue` の同じ番号の付け直し、「予約外の文書の follow-up」の起票した機能の除外は、どれも新しい文と別の面を言う
- 「待つブランチがなければ…着手を自動では聞かない」の文と、予約の重なりだけで決まる今の待ち(「重なる org が 2 つ以上なら、そのブランチを全部待つ」)は残っている(計画の Regression)
- 重なりと依存が同じブランチを指すときの重複、頼る先が 2 つのとき(45 の例)、頼る先が merge 済みで待たないとき(`after: none` になる)が、手順と例の両方に出ている

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 頼る先の merge の確かめの文が、「起票と記録」の 1 の定義と、4 か所の言い直し(director の 4・引き継ぎ・2、org skill)に分かれている(L1) | 次に規則を変える人が 5 か所を揃える必要があり、4 写しとそれぞれの fragment でも揃える。1 か所の直し漏れは、前の分割計画の merge を数える方に倒れる | 計画の Scope が「起票と記録」の 1 に定義を置く形で、小節を切り出す変更は文の入れ替えが大きい | 頼る先の merge の確かめの次の変更、または M1 を直すとき | この report の L1・M1、`docs/plans/active/2026-11-22-issue-depends-wait.md`(Scope) |
| 3 の「移す」手順が、答え済みの issue も選び、slug を変える案を先に案内する(L4) | slug か Type を変えるまれな場合に、答えた人にもう一度聞く、または待つ側の全 issue を書き換える | 計画の Assumptions が、slug を変える経路は今ある書き換えの手順を残す、と許している | 「merge を待つ issue」の 3 の次の変更、または移す手順を使った報告 | この report の L4、同計画の Assumptions |

_台帳 `docs/tech-debt/README.md` には追記していない。依頼の範囲が report・計画の Progress・insight event の 1 コミットで、複数のレーンが台帳の末尾に追記して衝突するため。上の 2 行は、M1 を直さずに進めるときに `/pr` の前に台帳へ入れてほしい(M1 は debt ではなく直す所見)。_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 6)
- Follow-ups:
  - M1 は skill の文の直しで、コードの振る舞いは変わらない。ただし「feature だけが合う」問いは AC6 の中心の場面なので、cross-review の前に直すのが安い。直すなら 4 写し(`scripts/sync-skills.sh` で `.agents` を作る)と、固定する fragment(`skill_issue_depends_test.go` の 3 つのテストと `skill_split_questions_test.go` の 2 か所)を 1 回で直す。直したら `/self-review` 以降の再実行が要る(skill と Go のテストに触れるので記録だけの修正ではない)
  - L1・L3・L5 は M1 と同じ文を触るので、M1 を直すときにまとめる。L2・L6 は文とテストの小さな直し。L4 は台帳の行にして `/pr` に進んでよい
  - 直さずに進めるなら、PR の本文の Known gaps に「前の別の分割計画の同名ブランチが merge 済みのとき、director は報告のたびに人に聞き、答えを残さない」を書く
  - テストの実行と静的解析、AC の突き合わせは `/verify` と `/test` の担当
