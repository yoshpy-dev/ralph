# Self-review report: org-cross-review

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(issue #269、D1〜D10、D6b、D7b、AC1〜AC9、スライス S1 e3c04e18・586f7934、S1b a420eae0、S2 2478e069、S1c aa0a0fd5、S2b 82df1474、S3 93d58b65、S4 91c3798b、S5 106bc425)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base 4fef6b8c、HEAD e84c3054、31 ファイル、+3865/-160)。reviewer の雛形の `## cross-review の TASK`、leader の雛形の手順 9 の段落・`### cross-review`・ミッション 5・手順 12 の停滞・手順 14 の PR の本文と RESULT、計画助言の codex の `--disable apps`、`internal/org/driver/paneenv.go` の警告文と `spawn.go`・`config.go` のコメント、`templates/base/ralph.toml` のコメント、`/director` と `/org` の skill と写し、README・recipe・`docs/reports/README.md`・仕様の注、evidence、新しいテスト 3 本と変えたテスト 4 本。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1(MEDIUM)と L1〜L7(LOW)。
- Run 2: 2026-11-01、HEAD ed8f6805、パイプラインは cycle 1 のまま。run 2 の所見は R2-1〜R2-4(すべて LOW)で、run 1 の ID とは別の番号にした。
- 構成: 先頭が run 2 の節、次が run 1 の節。run 1 の節は、見出しに「(run 1)」を足して 1 段下げ、`Merge:` の行の名前を「Run 1 の判定」に変えたほかは、run 1 のままにした。ファイルの `Merge:` の行は run 2 の節の 1 行だけ。

## Run 2(HEAD ed8f6805)

### Scope(run 2)

- ブランチ全体: `git diff origin/main...HEAD`(base 4fef6b8c、38 ファイル、+4370/-195)。
- 重点: run 1 の report(a47bd973)からの直し `git diff a47bd973..HEAD`(21 ファイル、+570/-151)。2632f14f(計画の Affected areas と tech-debt の行。L7)、F1 522fb8cd(M1 の `base:` の行と `--diff-filter=AM`、L1 の上限を上げたあとの打ち直し、L3 の leader の段の頭、L4 の `log:` の 200 字の切り詰め)、F2 a6a39a6c(L2 と L3 の skill、L5 の helper のコメント、L6 の evidence、README)、F3 2c41af71(上げた上限の数を `FORK 1:` の行の `新しい上限 <数>` からだけ読む)、ed8f6805(計画の Progress)。
- 走らせたもの: scratchpad で、leader の `### cross-review` の状態のコマンドを `{{ORG_ID}}` = `org-a` で抜き出し、stub の ralph を置いて bash で打った(R2-1、R2-4、base が動いたときの確かめ)。reviewer の awk を、この機械の awk(macOS、version 20200816)で UTF-8 の行と UTF-8 でないバイトの行に打った。elif のあとの else の `$?` を bash と `zsh -f` で確かめた。テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。写しは `cmp` だけ打った。

### Run 1 の所見の扱い(run 2)

| ID | 扱い | 根拠 |
| --- | --- | --- |
| M1 | 解決。ただし R2-1 | 状態のコマンド(`internal/org/prompts/leader.md:802-804`)が `origin/<既定のブランチ>` を手順 9 と同じ順(`origin/HEAD` の先、`main`、`master`)で決め、`git diff --name-only --no-renames --diff-filter=AM "$base...HEAD"` に出る report だけを数える。base が決まらなければ `base: error` で BLOCKED(`:833-840`)。手順 14 の report のパスも同じ組(`:607-610`、`:1016-1020`)。テストは `TestRenderRolePrompt_Leader_XreviewStateCommandCountsOnlyThisBranchsReports` と `...StateCommandFindsStep9sBase`。run 1 の再現(main に前の org の complete の c1・c2)は scratchpad で `done: 0`・`k: 1`・`last: none` になった |
| L1 | 解決 | 新しい上限を決めるのは leader だけになった(`leader.md:912-919`: `FORK 1:` の行の `新しい上限 <数>` が `done:` より大きければその数、それ以外とその句がないときは `done:` + 1)。director は数を計算せず、人が言った数を `FORK 1:` の行にだけ書く(`.claude/skills/director/SKILL.md:871-892`)。2 つ目の QUESTION の a は状態のコマンドを打ち直してから TASK を送り、`CYCLE:` の上限はその `cap:`(`leader.md:922-925`、`:946-947`)。F3 で、`SUMMARY:` の `e5` の 5 を上限と読む道も閉じた |
| L2 | 解決 | `director/SKILL.md:857-859` が、report は機能のブランチにしかないので機能の worktree(`.claude/worktrees/org-<org_id>`)で読む、と書く。写し 4 つは `cmp` で同じ |
| L3 | 解決 | `leader.md:362` と `.claude/skills/org/SKILL.md:1106` の段の頭が「cross-review が終わったら(off で TASK を送らなかったときも)」になり、`syncDocsHandOff` と `orgSkillSyncDocsHandOff` も合わせた。前の段の終わり(`leader.md:354`、`org/SKILL.md:1102`)も同じ句なので重なるが、指す時点は同じで食い違いはない |
| L4 | 解決(長さ)。絶対パスの点は残る | `reviewer.md:247` と `:276` が `LC_ALL=C` の awk で 200 字に切る。手順 4 の文(`:312-318`)と、「あ」300 字の行を 200 字に切るテストがある。run 1 の所見が「起きること」に書いた、log の行のローカルの絶対パスが公開の PR の本文に出る点は、勧めに入れていなかったので手当てがない。codex と claude の失敗の行にパスが入ることはおそらく少ない(未確認)ので、所見にはしない |
| L5 | 解決 | `scripts/xreview-helpers.sh:87-108` が 4 つ目の呼び出し元(reviewer 座席の cross-review)と、引数があれば `RALPH_PRIMARY_CLI` を読まないことを書く。写しと同じ |
| L6 | 解決 | `docs/evidence/xreview-live-probe-2026-11-01.md:68` を `<scratchpad>` に伏せ、`:72` で理由を書いた。ブランチが変えたファイル(この report を除く)に `/Users/` と `-Users-` はない |
| L7 | 解決。ただし数が古くなった(R2-2) | 計画の Affected areas(`docs/plans/active/2026-11-01-org-cross-review.md:41-45`)が実際のテストのファイルを挙げ、`docs/tech-debt/README.md:416` に大きさの行がある |

### 依頼された点の確認(run 2)

- コミットしていない report: 数えない。`git diff <base>...HEAD` はコミットだけを比べる。leader は返事を受けるたびに report をコミットしてから先に進み(`leader.md:982-985`)、`action_required` では、そのあとで状態のコマンドを打ち直す(`:994`)。手順 9 は fetch の前に `git status --porcelain` が空であることを確かめる。コミット前の report が数から漏れる道は見当たらない。テストにも「書いたがコミットしていない c1 は数えない」場合がある。
- 消した report と名前を変えた report: `--no-renames` なので、消した report は D で数えず、名前を変えた report は新しい名前の A として数える(正規表現に合うときだけ)。`k:` は残った番号から決まるので、消した番号を使い直しうるが、雛形は report を消さない前提(`leader.md:851-852`)なので所見にしない。前の org の report をこのブランチが書き換えたときは R2-1。
- rebase のあとに base が動く: scratchpad で、`origin/main` が進んだだけのとき、rebase のあと、`origin/main` を merge したあとのどれも `done: 1`・`k: 2` のままだった(merge base からの差分なので、ブランチが足した report は A のまま)。`since:` は rebase と merge のあと `changed` になり、雛形の説明(`:858-860`)どおり。
- awk の切り詰め: macOS の awk で、「あ」300 字は 200 字(600 バイト)、「aé」の 125 回は 200 字、4 バイトの絵文字 250 字は 200 字(800 バイト)、0xFF の 300 バイトと Latin-1 の é を混ぜた行は 200 バイトで切れた。UTF-8 の続きのバイト(0x80〜0xBF)だけの行は頭のバイトがないので数えられず、1,000 バイトがそのまま通る。codex と claude の log にそういう行が出ることはまずないので、所見にしない。awk の前の `grep '[^[:space:]]'` は seat の locale で動くので、GNU grep(Linux)は UTF-8 として壊れたバイトを含む行を出さず、`log:` はその前の行になるか出ないはず(未確認。この機械の `/usr/bin/grep` は BSD grep で、バイトのまま出した)。S1c からある形で、今回の直しが入れたものではない。gawk と mawk では打っていない。
- elif のあとの else の `$?`: bash で 1 と 128、`zsh -f` で 128 になり、`base: error (git diff exit <n>)` は `git diff` の終了コードを出す。
- 3 つの文書の一致: TASK の本文の行の並び(`reviewerXreviewTaskKeys` が見る)と `XREVIEW:` の値の枝は run 1 から変わらない。上限の QUESTION の `FORK 1:`、答えの `FORK 1: a 上限を上げる(新しい上限 <数>)`、leader の読み方(`新しい上限 <数>` だけを読み、`done:` 以下なら `done:` + 1)、director の説明(`<n>` は `EVIDENCE:` の report の `- Cycle:` の n)はそろう。`EVIDENCE:` は最後の complete の report で、complete の回の `CYCLE:` の n はその TASK のときの `done:` + 1 なので、その `<n>` は leader の `done:` と同じ値になる。数え方は leader の `done:`、`/org` skill(`:1091-1095`)、仕様の 2026-11-01 の注で食い違わない(`/org` と仕様は「このブランチの」とは書かないが、反対のことも書かない)。
- 大きさ: `leader.md` は 1,251 行、`reviewer.md` は 524 行、`prompts_xreview_leader_test.go` は 1,125 行、`prompts_xreview_reviewer_test.go` は 1,081 行、`skill_xreview_test.go` は 485 行(R2-2)。

### Findings(run 2)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R2-1 | LOW | 正しさ(数える report の組) | `--diff-filter=AM` の M は、注が書く「同じ日付・同じパスへの上書き」だけでなく、このブランチが前の org の report をどんな理由で書き換えても入る。scratchpad で、main に 2026-10-01 の org-a の complete の c1・c2 がある状態から、ブランチで c2 の見出しの 1 行を直してコミットすると、cross-review を 1 回も送っていないのに `done: 1`・`k: 3`・`last:` に前の org の c2 が出た。`since:` は前の org の HEAD と比べ、手順 14 の `Cross-review:` の行にも前の org の report が並ぶ。上書きの場合にも別の問題がある。`k:` はこのブランチの report だけから決まるので、同じ日に同じ org_id(既定は機能の slug)が使い直されると、reviewer は main にある前の org の report のパスに書き、この PR の merge で前の org の記録が tree から消える(history には残る)。PR の差分にも前の org の report の変更として出る。どちらも org_id の使い直しでだけ起き、数えすぎは人に聞く側に倒れる | `internal/org/prompts/leader.md:803`(`--diff-filter=AM`)、`:805-806`(`k:`)、`:847-852`(注)、`internal/org/prompts_xreview_leader_test.go:950-957`(上書きを数える場合の固定) | `k:` は前と同じく `git ls-files` で、この org_id の追跡中の report(日付を問わない)の番号のいちばん大きいもの + 1 にし、新しい report が既にあるパスに来ないようにする。`done:`・`last:`・手順 14 の組は `--diff-filter=A` にする。上書きが起きなくなるので M は要らない。上書きを残すなら、注を「このブランチが書き換えた前の org の report も数える」に直し、前の org の記録を上書きすることを計画の Risks に書く |
| R2-2 | LOW | 保守性(tech-debt の行の数) | L7 の直しで足した行は 2632f14f で書かれ、そのあとの F1〜F3 で数が古くなった。行は `leader.md` を「about 1,225」、`reviewer.md` を「about 519」、`prompts_xreview_leader_test.go` を「about 920」、`prompts_xreview_reviewer_test.go` を「about 1,054」と書くが、HEAD ではそれぞれ 1,251、524、1,125、1,081 行。leader のテストは 205 行(22%)増え、「about」の幅を超える。隣の `status.go` と `mergeorder.go` の行は数えたコミットを書くが、この行は書かない | `docs/tech-debt/README.md:416`。`wc -l` は 2632f14f で 1,225・519・920・1,054、522fb8cd で 1,248・524・1,115・1,081、2c41af71 で 1,251・524・1,125・1,081 | `/pr` の前の最後のコミット(sync-docs のあと)で `wc -l` を取り直し、数えたコミットを行に書く |
| R2-3 | LOW | 保守性(テストが確かめるもの) | `TestRenderRolePrompt_Leader_XreviewRaisedCapReachesTheTask` の実行の半分は、L1 の規則(`新しい上限 <数>` が `done:` より大きければその数、それ以外は `done:` + 1)を Go で書き直し(`raised := done + 1` から)、その値を cap のファイルに書いて状態のコマンドが読み戻すことを見る。規則を実行するのは leader(モデル)で、状態のコマンドは設定とファイルの大きいほうを出すだけなので、雛形の規則を変えてもこの半分は通る(直す前の雛形でも通る)。`CYCLE:` の 2 つの比較は、すぐ上の `got["cap"] != c.want` から従う。L1 を守っているのは句の固定のほう。加えて doc comment は新しい上限を「the DECISION's number when it is above done:」と書き、F3 の「`FORK 1:` の行の `新しい上限 <数>` からだけ」に追いついていない(同じファイルの定数の説明は追いついている) | `internal/org/prompts_xreview_leader_test.go:1001-1010`(doc comment)、`:1012-1017`(句の固定)、`:1055-1058`(Go の規則)、`:1064`(`CYCLE:` の比較)、比べた箇所 `:32-36` | doc comment を F3 の言い方に直し、実行の半分が確かめるのは「cap のファイルに書いた上限を状態のコマンドが `cap:` で出し、`done:` は変わらない」ことだけで、規則は句で固定している、と書く。Go の規則の写しは、DECISION ごとに書く数を並べた表に置き換える |
| R2-4 | LOW | 正しさ(BLOCKED の本文) | base が決まらない枝は `$p-base.err` を書かない(`2> "$p-base.err"` は elif の `git diff` にだけ付く)ので、前の回の `git diff` の失敗がファイルに残る。scratchpad で、merge base のない `origin/main` で打ち(`base: error (git diff exit 128)`、base.err は `fatal: origin/main...HEAD: no merge base`)、そのあと `origin/main` を消して打つと、行は `base: error (no origin/HEAD, origin/main, or origin/master)` なのに base.err は前の「no merge base」のままだった。leader は「base.err の要点(空なら行の理由)」を BLOCKED に書くので、古い理由を人に見せる | `internal/org/prompts/leader.md:803`、`:837-840` | `: > "$p-added.txt"` と同じ所で `: > "$p-base.err"` も空にする。固定のテスト(`...StateCommandFindsStep9sBase`)に、base.err が空であることを足す |

### Positive notes(run 2)

- M1 の直しは、手順 9 と同じ順の base の決め方をコマンドに入れ、base が決まらないときを BLOCKED に倒した。テストは bash と `zsh -f` で `origin/HEAD`・`main`・`master` の順と、どれもない場合を打つ。
- L1 の直しは、新しい上限を計算する所を leader の 1 か所にし、director は人の数を運ぶだけにした(run 1 の「1 つの値に 2 つの式」が消えた)。F3 は `SUMMARY:` の受信箱の ID を拾う道を閉じ、skill の例の DECISION が `FORK 1:` の行のほかに受信箱の ID しか数を持たないこと、leader の文がその例と同じ形(`(新しい上限 <数>)`)を読むことをテストが見る。
- L4 のテストは 3 バイトの文字で切り詰めを打ち、期待値が正しい UTF-8 の 200 字であることも確かめる。

### Coverage gaps(run 2)

- テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。
- awk は macOS の awk でだけ打った。CI の Linux の awk(mawk か gawk)と GNU grep では打っていない。
- 座席の中での実行は run 1 と同じく確かめていない(AC9 の known gap)。

### Recommendation(run 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。run 1 の M1 と L1〜L7 は解決した。L4 の絶対パスの点は勧めの外で残る。R2-1 は同じ org_id の使い直しでだけ起き、数えすぎは人に聞く側に倒れる。R2-2 は台帳の数、R2-3 はテストの説明と形、R2-4 は BLOCKED に書く理由で、どれも挙動の大筋は変えない)
- Follow-ups:
  - R2-1 と R2-4 は状態のコマンドと説明と固定のテストを変える。R2-2 は台帳の行だけ、R2-3 はテストだけ。
  - 直さずに PR に進むなら、R2-1、R2-3、R2-4 を 1 行にまとめて台帳に足し、R2-2 は sync-docs で数を直す。

## Run 1(HEAD e84c3054)

### Evidence reviewed(run 1)

- `git diff origin/main...HEAD` の全体と、`git log --oneline origin/main..HEAD` の 13 コミット。
- 雛形のコマンドの抜き出しと実行(scratchpad): leader の `### cross-review` の最初のコマンドを `{{ORG_ID}}` = `org-a` で抜き出し、main に org-a の complete の report c1・c2 があるリポジトリで新しいブランチを切って bash で打った(M1 の根拠)。テストもほかの静的検査も走らせていない(`/verify` と `/test` の範囲)。
- 写しの比較: `/director` と `/org` の skill の 4 つの写しは `.claude/skills/` とバイト単位で同じ(`cmp`)。`docs/recipes/codex-seat-permissions.md`、`docs/recipes/isolated-herdr-server.md`、`docs/reports/README.md` は `templates/base/` の写しと同じ。
- 古い記述の掃除: `git grep` で「seat's cross-review」「座席の cross-review」「treats the seat's driver as claude」「後の版から」を探した。残りは README の 278 行(計画が sync-docs に回すと書いた分)と、仕様の 2026-10-29 の決定の記録(日付つきの記録なので残してよい)だけだった。

### Findings(run 1)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | 正しさ(回数の数え方) | M1: 上限に数える report を、このブランチが足したものに絞っていない。`git ls-files` は base から来た report も出すので、同じ org_id が前に使われてその PR が main に入っていると、前の org の report が `done:` と `last:` に入る。scratchpad で、main に org-a の complete の report c1・c2 がある状態で新しいブランチを切って打つと、`done: 2`・`cap: 2`・`since: changed` が出た。leader はこの org で cross-review を 1 回も送らずに、上限の QUESTION の 2 つ目(「最後の complete の回のあとに見せていない変更がある」)を人に上げる。前の org の report が 1 本なら、この org は 1 回しか回せず、それを誰も知らない。org_id の使い回しは repo が想定している場合で(`internal/org/cleanup.go:34-36` の「a worktree another feature made on the same org_id」、`internal/org/approval.go:373` の「an approval from an earlier life of the same org_id never counts」)、ここだけが前の org の記録を数える。手順 14 の `Cross-review:` の行に並べる report のパスにも、前の org の report が入る | `internal/org/prompts/leader.md:798-799`、`:835-846`(`last:` と `since:` の説明)、`:602-610` と `:989-994`(手順 14 の値と report のパス) | 数える対象を、このブランチで足した report にする。手順 9 の fetch のあとなので `origin/<既定のブランチ>` は読める。たとえば `git diff --name-only --no-renames --diff-filter=A "origin/<既定のブランチ>...HEAD" -- 'docs/reports/review-xreview-*.md'` を `git ls-files` の代わりに使う(コマンドに既定のブランチの決め方を入れるか、手順 9 で決めた名前を書き換える場所を作る)。`TestRenderRolePrompt_Leader_XreviewStateCommandCountsTheReviewersReports` に、base のブランチにある同じ org_id の complete の report を数えない場合を足す |
| LOW | 正しさ(上限を上げる答え) | L1: 上限の QUESTION に a で答えたときの新しい上限を、leader と director が別の式で決めている。director は人が数を言わないとき、`EVIDENCE:` の report の `- Cycle: <n>/<上限>` の `<上限>` + 1 を DECISION に書く。leader は 2 つ目の QUESTION の a で「すぐ TASK を送る」が、TASK の `CYCLE:` の `<上限>` は `cap:` の値で、それは cap のファイルを書く前に出した値なので、TASK は `CYCLE: 3/2` になり、reviewer はそのまま report に写す。この回のあとに上限の QUESTION がまた上がると、director は 2 + 1 = 3 を返し、`done:` が 3 のまま `cap: 3` になって同じ問いがすぐ戻る。人が 4 以上の数を言うまで a が効かない。leader の側の「書いていなければ `done:` + 1」は、D7b が同じ問いの繰り返しを避けるために選んだ式だが、director はいつも数を書くので使われない | `internal/org/prompts/leader.md:894-901`、`:922`、`.claude/skills/director/SKILL.md:869-874` | director は、人が数を言わなければ DECISION に数を書かない(leader が `done:` + 1 を使う)か、`<n>` + 1 にする。leader の 2 つ目の QUESTION の a は、cap のファイルを書いたあとに小節の初めのコマンドを打ち直してから TASK を送る、と書く。写しの 4 つと固定のテスト(`skill_xreview_test.go` の上限の節)も合わせる |
| LOW | 読みやすさ(director の読む場所) | L2: `/director` の「cross-review の上限」は `EVIDENCE:` の report を読めと言うが、どこで読むかを書いていない。report は機能のブランチにしかなく、director が動く main worktree のルートにはない。同じ skill の「計画の承認」は「機能の worktree(`.claude/worktrees/org-<org_id>`)で」と場所を書いている。director が report を開けなければ、1 の「どちらの件か」も、3 の今の上限も読めない | `.claude/skills/director/SKILL.md:854-857`、`:872-874`、比べた箇所 `:794` | 「機能の worktree(`.claude/worktrees/org-<org_id>`)の `EVIDENCE:` の report」と書く(写し 4 つ) |
| LOW | 読みやすさ(手順 9 の段落の頭) | L3: 差し込んだ段落も、そのあとの sync-docs の段落も「rebase が済んだら(要らなかったときも)」で始まる。差し込んだ段落は「cross-review が終わったら … 次の段落の sync-docs に進む」と言うが、sync-docs の段落の頭は sync-docs を rebase の直後に置いたままに読める。compaction のあとに sync-docs の段落から読み直した leader は、cross-review を飛ばしうる。`/org` skill も同じ形 | `internal/org/prompts/leader.md:351`、`:362`、`.claude/skills/org/SKILL.md:1074`、`:1106` | sync-docs の段落の頭を「cross-review が終わったら(off で送らなかったときも)」などに変え、その文を固定する `syncDocsHandOff`(`internal/org/prompts_leader_syncdocs_test.go:20`)と `orgSkillSyncDocsHandOff`(`internal/org/skill_doc_followup_test.go:19`)を合わせる |
| LOW | 保守性(理由の長さと行き先) | L4: incomplete の理由に入る `log:` の行は、log の 1 行をそのまま出し、長さを切らない。その値は report の `Reviewer status:`、reviewer の RESULT の `XREVIEW: incomplete (...)`(本文は 2,000 字まで)、leader の PR の本文の `Cross-review:` と RESULT の `CROSS_REVIEW:` に写る。codex の log の最後の行が API のエラーの JSON なら数百字になりうる。leader の RESULT の `DOC_FOLLOWUP:` には 2,000 字に収まらないときの書き方があるが、`CROSS_REVIEW:` にはない。PR の本文は公開されるので、log の行に入ったローカルの絶対パスもそのまま出る | `internal/org/prompts/reviewer.md:247`、`:276`、`:302-313` | コマンドの側で切る(`l=$(grep '[^[:space:]]' "$p.log" \| tail -n 1 \| cut -c1-200)`)。手順 4 に「理由は 1 行で、200 字まで」と書き、テストの `failLog` の比較はそのまま通る |
| LOW | 保守性(helper の説明) | L5: `pick_reviewer` の説明は呼び出し元を 3 つと書き、「All three callers pass the driver they resolved: the CLI running the skill …, which RALPH_PRIMARY_CLI … overrides」と言う。この diff で reviewer 座席の cross-review が 4 つ目の呼び出し元になり、渡すのは自分の CLI ではなく TASK の implementer の driver で、`RALPH_PRIMARY_CLI` は効かない(D6 が頼る「引数があれば env を読まない」性質)。説明だけを読んで helper の引数の扱いを変えると、この呼び出しの前提を確かめないまま変えうる | `scripts/xreview-helpers.sh:87-100`、`templates/base/scripts/xreview-helpers.sh:87-100` | 4 つ目の呼び出し元と、引数で implementer の driver を渡し env を読ませないことを、2 つの写しに足す。この slice の外として sync-docs に回すなら、その旨を計画の Progress に書く |
| LOW | security(evidence の伏せ字) | L6: codex の出力を写した行に、scratchpad の絶対パスがそのまま入っている。パスには利用者名と session の ID が含まれる。repo の evidence は `$HOME` を `~` に直して載せる約束で、この行だけが外れている | `docs/evidence/xreview-live-probe-2026-11-01.md:68` | `<scratchpad>/probe-269/internal/org/prompts/reviewer.md:229-231` のように伏せる。同じファイルのほかの行(`~/.codex/memories/MEMORY.md` など)はすでに伏せてある |
| LOW | 保守性(計画の記録) | L7: 計画の記録が diff に追いついていない。Affected areas は新しい `internal/org/prompts_xreview_test.go` と、`prompts_leader_syncdocs_test.go`・`director_skill_sync_test.go` の変更を挙げるが、実際に作ったのは `prompts_xreview_leader_test.go`(920 行)・`prompts_xreview_reviewer_test.go`(1,054 行)・`skill_xreview_test.go`(382 行)で、挙げた 2 つは変えていない。Progress はこの分け方を書いていない(S1c と S2b の outline の行にファイル名が出るだけ)。Risks の「`leader.md` が 975 行からさらに伸びる … 大きさは tech-debt で扱う」に当たる tech-debt の行もない(`leader.md` は 975 → 1,225 行、`reviewer.md` は 224 → 519 行) | `docs/plans/active/2026-11-01-org-cross-review.md:42-43`、`:113`、`docs/tech-debt/README.md`(行なし) | Affected areas を実際のファイルに直すか、Progress に分けた理由を書く。tech-debt に 1 行足す(案は下の Follow-ups) |

### Positive notes(run 1)

- TASK の本文の行は両方の雛形で同じ並び(`IMPLEMENTER_DRIVER:`・`XREVIEW_BASE:`・`CYCLE:`・`REPORT:`・`REVIEWER_CODEX:`・`REVIEWER_CLAUDE:`)で、`reviewerXreviewTaskKeys` が reviewer の節から読んだ並びで leader の側を確かめる。`XREVIEW:` の値は、reviewer が返す 5 つ(pass、action_required、skipped、incomplete、git_changed)と、行が欠けたときの QUESTION のすべてに leader の枝がある(`leader.md:957-987`)。上限の QUESTION の `XREVIEW: cap_reached` のヘッダと `FORK 1:` の行は、`validateEscalation` と `protocol.HeaderValues` で読めることをテストが確かめ、`/director` の答えの `FORK 1: <記号> <案>` を leader が読む。
- reviewer のコマンド: `IMPLEMENTER_DRIVER:` の値を一重引用符の `case` で先に確かめるので、`$(echo claude)` のような値も展開されずに `driver: invalid` になる(テストあり)。CLI の確かめは `sh -c 'command -v claude'` で、zsh の alias を CLI とみなさない(alias だけの場合のテストあり)。codex の 3 回の呼び出しはどれも `--disable plugins --disable apps` を持ち、`-c` はすべて `exec` の前にある。`select(.enabled != false)` で、欄のない server も有効と数える(閉じる側に倒れる)。見張りの形と、前後の git の状態の比較(未追跡のファイルの中身の hash を含む)は計画助言と同じで、出力はすべて無視される `.harness/state/` に書く。
- leader のコマンド: `case "$rc:$v"` で 0・1・127 とそれ以外を分け、`cap:` は正の整数だけを受け、cap のファイルは 1 行目だけを読み、設定より下げない(18 の場合のテストあり)。report の名前は正規表現で日付と org_id を固定し、`${r##*-c}` は最後の `-c` を取るので、org_id に `-c` を含んでも番号を読み違えない(`org-a-cache` のテストあり)。`since:` は `git diff` が失敗すれば `changed` に倒れ、人に聞く側に寄る。
- テスト: 抜き出したコマンドを bash と `zsh -f` でスタブに対して打つ。`RALPH_PRIMARY_CLI` に implementer と同じ側の値を入れて、env を読む実装なら落ちるようにしてある。背景のジョブの通知が行の間に入っても通る `reviewerXreviewHasLines` で、負荷で落ちた 2 か所(S1c)を直した。見張りの確かめは見張りだけが出せる `rc=143` で見る。
- `paneenv.go` の警告文は、`leader.md:672` の形(`claude` か `codex` のときだけ env が自分の CLI に勝つ)と合う。新しいテストの 2 本は、警告に「a seat's cross-review」が戻らないことも確かめる。

### Coverage gaps(run 1)

- テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。
- reviewer の codex のコマンドについて、`--disable apps` を知らない古い codex の場合(`mcp: not off (enabled: )` から `skipped (codex: MCP servers not off)` へ)を打つテストはない。計画助言の側には `STUB_OLD=apps` の場合がある。`/test` で見る点として残す。
- 座席の中(codex の sandbox と network、pane の環境)での実行は、計画どおり確かめていない(AC9 の known gap)。
- 新しいテストの 2 本は 800 行の目安を超える(920 行と 1,054 行)。同じパッケージには 6,000 行のテストもあり、話題ごとに分けてあるので所見にはしなかった。L7 の tech-debt の行に入れる案にした。

### Recommendation(run 1)

- Run 1 の判定: yes(CRITICAL と HIGH はない。M1 は同じ org_id の 2 回目の機能でだけ起き、その場合でも人に聞く側に倒れるが、PR の前に直すことを勧める)
- Follow-ups:
  - M1 と L1〜L7 の直し。M1 と L1 はコマンドと固定のテストの両方を直す。
  - tech-debt の行の案(L7): 「`internal/org/prompts/leader.md` は #269 で 975 行から 1,225 行、`reviewer.md` は 224 行から 519 行になった。`prompts_xreview_leader_test.go`(920 行)と `prompts_xreview_reviewer_test.go`(1,054 行)は 800 行の目安を超える。役割の雛形を節ごとのファイルに分けるか、座席が読む量を測ってから決める」。この commit は report と event だけに絞るので、tech-debt には書いていない。
  - L5 と README の 278 行(計画が sync-docs に回した分)は、sync-docs でまとめて見られる。
