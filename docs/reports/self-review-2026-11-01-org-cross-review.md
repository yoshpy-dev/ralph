# Self-review report: org-cross-review

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(issue #269、D1〜D10、D6b、D7b、AC1〜AC9、スライス S1 e3c04e18・586f7934、S1b a420eae0、S2 2478e069、S1c aa0a0fd5、S2b 82df1474、S3 93d58b65、S4 91c3798b、S5 106bc425)
- Reviewer: reviewer サブエージェント(`/self-review`。run 1 と run 2 はパイプラインの cycle 1、cycle 2 の節は cycle 2)
- Scope: `git diff origin/main...HEAD`(base 4fef6b8c、HEAD e84c3054、31 ファイル、+3865/-160)。reviewer の雛形の `## cross-review の TASK`、leader の雛形の手順 9 の段落・`### cross-review`・ミッション 5・手順 12 の停滞・手順 14 の PR の本文と RESULT、計画助言の codex の `--disable apps`、`internal/org/driver/paneenv.go` の警告文と `spawn.go`・`config.go` のコメント、`templates/base/ralph.toml` のコメント、`/director` と `/org` の skill と写し、README・recipe・`docs/reports/README.md`・仕様の注、evidence、新しいテスト 3 本と変えたテスト 4 本。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1(MEDIUM)と L1〜L7(LOW)。
- Run 2: 2026-11-01、HEAD ed8f6805、パイプラインは cycle 1 のまま。run 2 の所見は R2-1〜R2-4(すべて LOW)で、run 1 の ID とは別の番号にした。
- Cycle 2: 2026-11-02、HEAD b3750ea6、パイプラインの cycle 2(上限 2 の最後の回)。所見は C2-1〜C2-4(すべて LOW)で、cycle 1 の ID とは別の番号にした。cross-review の所見は、triage の report(`docs/reports/cross-review-triage-org-cross-review.md`)が回ごとに書き換わるので、「cross-review cycle 1 の ACTION_REQUIRED #1(3aeafce1)」のように回とコミットを付けて書く。
- 構成: 先頭が cycle 2 の節、次が cycle 1 の run 2 の節、最後が run 1 の節。run 1 の節は、見出しに「(run 1)」を足して 1 段下げ、`Merge:` の行の名前を「Run 1 の判定」に変えたほかは、run 1 のままにした。run 2 の節は、`Merge:` の行の名前を「Run 2 の判定」に変えたほかは、run 2 のままにした。ファイルの `Merge:` の行は cycle 2 の節の 1 行だけ。

## Cycle 2(HEAD b3750ea6)

### Scope(cycle 2)

- ブランチ全体: `git diff main...HEAD`(base 4fef6b8c、47 ファイル、+6048/-227)。
- 重点: run 2 の report(15e62eb5)からの差分 `git diff 15e62eb5..HEAD`(23 ファイル、+1828/-246)。F4 dbfc3ef2(R2-1〜R2-4)、F5 62d62be0(verify cycle 1 の V2: report の `HEAD:`、leader がコミットする `REPORT:`、reviewer の `XREVIEW_BASE:`・`REPORT:`・`REVIEWER_*` の確かめと、`last head:` の printf)、F6 17fc2b52(2 回目の live probe の所見: 台帳から読む implementer の driver、行の終わりに空白のある `Reviewer status: complete`、`-r3` までの送り直し、所見のパスの相対化と log の行のパスの置き換え)、F7 4d289952(cross-review cycle 1 の ACTION_REQUIRED #1 と #2)、文書のコミット(sync-docs の de47b7a0、設定のコメントの 2e75c82a、計画の b3750ea6)。
- 走らせたもの(scratchpad): reviewer の手順 1 の base の確かめを、行の頭に空白のない形と 3 字の空白を付けた形で `bash -c` と `zsh -fc` に打った(C2-2)。leader の `since:` の pathspec を、`docs/reports/` 直下の report だけのコミットのあと(same)と `docs/reports/templates/` のコミットのあと(changed)で bash と `zsh -f` に打った。reviewer の log の行のパスの置き換え(awk の `rep`)を macOS の awk(20200816)で打った。テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。

### Cycle 1 の所見の扱い(cycle 2)

| ID | 扱い | 根拠 |
| --- | --- | --- |
| R2-1 | 解決 | `done:` と `last:` の組は `--diff-filter=A`(`internal/org/prompts/leader.md:804`)、`k:` は `git ls-files` が出すこの org_id の report(日付も、前の org のものかも問わない)の番号のいちばん大きいもの + 1(`:806-807`、説明は `:854-858`)。前の org が使った番号を飛ばすので上書きは起きず、このブランチが前の org の report を書き換えても数えない(`:851-853`)。R2-1 の勧めのとおり |
| R2-2 | 解決。F7 のあとで数が少しずれた | `docs/tech-debt/README.md:416` は数えたコミット(433df4ae)と dbfc3ef2 での数を書く。HEAD では `leader.md` 1,300 行、`reviewer.md` 562 行、`prompts_xreview_leader_test.go` 1,485 行、`prompts_xreview_reviewer_test.go` 1,303 行で、行の 1,298・542・1,449・1,268 からいちばん大きくずれたのは `reviewer.md` の 3.7%。行は数えたコミットを書くので誤りではなく、所見にしない |
| R2-3 | 解決 | doc comment は F3 の読み方(`FORK 1:` の行の `新しい上限 <数>` からだけ読む)になり、実行の半分は状態のコマンドの読み戻しだけを見ると書く(`internal/org/prompts_xreview_leader_test.go:1048-1062`)。Go で書き直した規則は、DECISION ごとの上限を並べた表になった(`:1074-1082`) |
| R2-4 | 解決 | `: > "$p-base.err"` を `$p-added.txt` と同じ所で空にする(`leader.md:804`)。`...StateCommandFindsStep9sBase` が、base が決まらない回のあとで base.err が空であることを見る(`prompts_xreview_leader_test.go:1037-1039`) |
| cross-review cycle 1 の ACTION_REQUIRED #1(3aeafce1) | 解決 | `since:` の pathspec が `':(exclude,glob)docs/reports/*.md'` になった(`leader.md:810`)。glob の `*` は `/` をまたがないので、`docs/reports/templates/` の変更は changed になる。scratchpad で bash と `zsh -f` に打って確かめた。`docs/reports/README.md` も直下の `.md` なので除かれるが、計画の Progress がそのことを書いている |
| cross-review cycle 1 の ACTION_REQUIRED #2(3aeafce1) | 解決。ただし C2-1 と C2-2 | reviewer の手順 1 は、値を区切りを引用符で囲んだ heredoc で読み、`origin/` で始まることと、`-` で始まらず `..`・`@{`・空白・制御文字・引用符・`$`・`` ` ``・`\` を持たないことを見てから `git check-ref-format --branch` に渡す(`internal/org/prompts/reviewer.md:199-205`)。テストは bash と `zsh -f` で、通す名前(`release=stable`・`release(v2)`・`dévelop` など)と断る値(`origin/@{-1}`、コマンド置換を含む値など)を打つ |

### 依頼された点の確認(cycle 2)

- reviewer の base の確かめ(heredoc): 行の頭に空白のない形は、bash と `zsh -f` のどちらでも `base: ok` を出した。3 字の空白を付けた形は heredoc が閉じず、`bash -c` も `zsh -fc` も何も出さずに終了コード 0 で終わった(C2-2)。`case` のパターン(`*'@{'*`、`*\'*`、`*[[:space:][:cntrl:]]*`)は両方の shell で同じに読まれ、テストは `dévelop` と制御文字を含む値も打つ。
- leader の driver のコマンド(`leader.md:978-982`): `ralph org status --json` は `seats` を `make` で作るので、座席がなくても `null` ではなく `[]` を出す(`internal/cli/org.go:1857`)。jq の `.seats[]` は落ちずに `0 ` を返し、`driver: error (no implementer seat)` になる。`Roster` は止めた座席も返すので、止めた implementer の driver も読む(テストあり)。`--state-dir <台帳>` は、タスクの行にある引用済みの値をそのまま使う(`leader.md:134`、`internal/org/feature.go:485`)。jq がない環境では、implementer が codex のとき(相手は claude で、reviewer の側では jq は要らない)も leader が TASK を送れずに BLOCKED になる。jq は scaffold の hook でも使っており(jq がないときの道もある)、理由は status.err に出るので、所見にしない。
- `since:` の glob の pathspec は一重引用符の中なので、どちらの shell も展開しない。
- 3 つの文書: leader の TASK の例と、reviewer の RESULT と BLOCKED の例の TASK_ID は、どれも `xreview-{{ORG_ID}}-c1` になった。director の「cross-review の上限」が読む `<n>`(`EVIDENCE:` の report の `- Cycle:` の n)は、`k:` が前の org の番号を飛ばすようになっても `CYCLE:` の n(`done:` + 1)から来るので、leader の `done:` と同じ値のまま。`/org` skill(de47b7a0)は、driver を台帳から読むことと、このブランチが足した report だけを数えることを書き、雛形と食い違わない。
- 人に見える変化: `templates/base/ralph.toml` の `[reviewer]` のコメントと model-routing.md の 2 つの写しが、reviewer 座席の cross-review を足した。挙動が後退した所は見当たらない。
- log の行のパスの置き換え: macOS の awk で、worktree の下のパスは `./a.go:1` に、`$HOME/.codex` は `~/.codex` になり、worktree の名前を伸ばした `<worktree>-old` は置き換えなかった(設計どおり)。`$HOME` のすぐあとに `.` が来る場合(文の終わりの `... in /Users/<名前>.`)は、`.` を名前の文字とみなすので置き換えない。そういう行が codex と claude の log の最後に来ることはおそらくまれ(未確認)なので、所見にしない。
- 重複: パスの置き換えと切り詰めの awk(約 420 字)は、reviewer の codex と claude のコマンドに 1 つずつある。テストの `reviewerXreviewLogCut` が両方に同じ文字列があることを見るので、片方だけを変えると落ちる。所見にしない。
- 文書の食い違い(sync-docs の範囲なので所見にしない): 仕様の 2026-11-01 の注(`docs/specs/2026-10-07-org-multi-org-director.md:133`)は、上限の QUESTION の条件を「その回の report の `HEAD:` から `docs/reports/` の外が変わっていれば」と書き、F7 の「直下の `.md` のほか」に追いついていない。

### Findings(cycle 2)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | 保守性(tech-debt の行) | sync-docs(de47b7a0)が足した行は、そのあとの F7 で 3 つの項目が古くなった。(6) と (7) は `reviewerXreviewBaseOK` と、その表の行 `origin/a@{1}`、残った変異 R1c〜R1e を前提にするが、F7 はこの関数と文字の許可リストの規則を消し、base の確かめをコマンドにした(テストはそのコマンドを bash と `zsh -f` で打つ)。(2) は「手順 1 の `XREVIEW_BASE:` の形は文だけで、stub の行は形の正しい値だけ(一重引用符を含む値はない)」と書くが、F7 のあとは `XREVIEW_BASE:` をコマンドが確かめ、テストは引用符・`$`・コマンド置換を含む値を打つ。(6) と (7) の Trigger(「`reviewerXreviewBaseOK` か `XREVIEW_BASE:` の規則の次の変更」)は F7 で起きたが、行は直っていない。行を読んだ人は、もうない関数を探すか、済んだ作業をもう一度する | `docs/tech-debt/README.md:417`。`git grep -n reviewerXreviewBaseOK HEAD` は、日付つきの test と sync-docs の report を除くと、この行にしか出ない。F7(4d289952)が `internal/org/prompts_xreview_reviewer_test.go` から関数を消した。`git log -S reviewerXreviewBaseOK -- docs/tech-debt/README.md` は de47b7a0 だけ | (2) を `REPORT:` と `REVIEWER_*` の形に絞り、`XREVIEW_BASE:` は F7 でコマンドが確かめるようになったと書く。(6) と (7) は F7 で消えたもの(関数がなくなり、規則はコマンドで打つ)として行から外すか取り消し線にし、件数(Nine)と Trigger を合わせる |
| C2-2 | LOW | 正しさ(打ち間違いの扱い) | reviewer の手順 1 は、base の確かめのコマンドが `base: ok` を出さなければ QUESTION で leader に聞く。このコマンドは雛形の中では 3 字下げたリストの中にあり、行の頭に空白を付けたまま打つと heredoc が閉じず、何も出さずに終了コード 0 で終わる(scratchpad で `bash -c` と `zsh -fc` の両方)。reviewer の打ち間違いが「値の形が違う」と同じ QUESTION になり、leader は TASK の値(`XREVIEW_BASE: origin/main`)を見ても直す所がない。同じ値で送り直して `-r2` と `-r3` を使い切るか、「何が違ったのかわからなければ」の枝で人に BLOCKED を上げる。雛形は空白を付けないよう書くが、出力からは打ち間違いと形の違う値を見分けられない | `internal/org/prompts/reviewer.md:167-168`(QUESTION の条件)、`:186-188`(空白の注意)、`:199-205`(コマンド)、`internal/org/prompts/leader.md:1053-1061`(leader の QUESTION の扱い)、`internal/org/prompts_xreview_reviewer_test.go:1255-1259`(空白を付けた形が `base: ok` を出さないことだけを見る) | 手順 1 に「コマンドが `base: ok` も `base: invalid` も出さなければ打ち方の誤りなので、行の頭の空白を外して打ち直し、QUESTION にしない」と書き、QUESTION は `base: invalid` のときだけにする。テストは、空白を付けた形が `base:` の行を 1 つも出さないことを見る |
| C2-3 | LOW | 読みやすさ(送り直しの上限の範囲) | `-r2` と `-r3` の番号は、reviewer の QUESTION のあとと `XREVIEW: git_changed` のあとの送り直しで共有する。一方、F6 が足した「同じ `<k>` で送り直すのは 2 回まで」とその止め方は、QUESTION の枝の中にだけある。git_changed のあと人がやり直しを答えて `-r2` を送り、その回に QUESTION が来ると、次の `-r3` への QUESTION で止まるので、QUESTION の送り直しは 1 回しかできない(文は「2 回送り直してもそろわない」と書く)。`-r3` で git_changed が出て人がやり直しを答えたときに、`-r4` を送ってよいのか上限に当たるのかを、雛形は書いていない | `internal/org/prompts/leader.md:948-953`(番号の共有)、`:1046-1052`(git_changed)、`:1057-1061`(上限) | 上限が何を数えるかを 1 つに決め、送り直しの番号の段落(`:948-953`)に書く。たとえば「QUESTION のあとの送り直しは 2 回まで。git_changed のあとの送り直しは人の答えに従い、上限に数えない(番号は続ける)」。句の固定のテストも合わせる |
| C2-4 | LOW | 読みやすさ(コメントの折り返しと掃き残し) | 2e75c82a と de47b7a0 が書き換えた文の折り返しが周りとそろっていない。`internal/config/config.go:75` は「codex. Load refuses an」で行が切れ、`:830` は 99 字(周りは 80 字まで)。`.claude/rules/ralph/model-routing.md:90` は 106 字、`templates/base/.claude/rules/ralph/model-routing.md:90` は 107 字で、同じ段落のほかの行は 80 字前後。加えて、同じコメントの `config.go:79-80`(「the values are copied into the shell commands of the skills and the leader as one word」)は、書き換えた 1 文目(`:71-73`)と違い、値を写すもう 1 つの所(reviewer 座席)を書いていない | `internal/config/config.go:75-76`、`:79-80`、`:829-830`、`.claude/rules/ralph/model-routing.md:90`、`templates/base/.claude/rules/ralph/model-routing.md:90` | 4 か所を周りの幅で折り返し直し(`gofmt` はコメントの折り返しを見ない)、`config.go:79-80` に reviewer 座席を足す |

### Positive notes(cycle 2)

- F5 は、座席が書いた値を git に渡す前に形を確かめ、確かめられない値は git に渡さずに changed に倒した。`HEAD:` に `--output=<file>` を入れた report でファイルが書かれないことを、SHA-1 と SHA-256 のリポジトリで打つテストがある。
- F6 の driver のコマンドは、記憶ではなく台帳から読み、読めないときをすべて BLOCKED に倒す。stub の ralph は、引数の数と、`--state-dir` の値を 1 語で受けたかまで確かめる。
- F7 は文字の許可リストを git の規則に置き換えた。`git check-ref-format --branch` が `@{-1}` を通すことをテストの前提として確かめてから、コマンドがそれを断ることを打つ。値を heredoc で読むので、コマンド置換を含む値でもファイルが作られないことも打つ。

### Coverage gaps(cycle 2)

- テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。
- awk は macOS の awk でだけ打った。GNU の awk と grep では打っていない。
- 座席の中での実行は確かめていない(AC9 の known gap、`docs/tech-debt/README.md:418`)。

### Recommendation(cycle 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。cycle 1 の R2-1〜R2-4 と cross-review cycle 1 の ACTION_REQUIRED 2 件は解決した。C2-1 は台帳の行、C2-4 はコメントと文書の折り返しで、挙動を変えない。C2-2 と C2-3 は、座席の打ち間違いか git の変化が重なったときにだけ人に BLOCKED が上がる場合で、どちらも止まる側に倒れる)
- Follow-ups:
  - この回はパイプラインの上限(2)の最後の回なので、直さない LOW はそのまま持ち越しになる。直さずに PR に進むなら、C2-2 と C2-3 を 1 行にまとめて台帳に足し、C2-1 はその行を書くときに `:417` の行を合わせて直す。
  - 直すときに変わるもの: C2-1 は台帳の行、C2-2 は reviewer の雛形の文とテスト、C2-3 は leader の雛形の文と句の固定、C2-4 は Go のコメントと文書の 2 つの写し。

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

- Run 2 の判定: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。run 1 の M1 と L1〜L7 は解決した。L4 の絶対パスの点は勧めの外で残る。R2-1 は同じ org_id の使い直しでだけ起き、数えすぎは人に聞く側に倒れる。R2-2 は台帳の数、R2-3 はテストの説明と形、R2-4 は BLOCKED に書く理由で、どれも挙動の大筋は変えない)
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
