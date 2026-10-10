# Self-review report: guard-test-gaps

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-guard-test-gaps.md
- Branch: test/guard-test-gaps(base 49ac046c、HEAD e8a188f6)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1
- Scope: diff の品質だけを見た(`cmd_pos` の `WRAPPER` の判定が判定を変えないか、名前とコメントの正確さ、不変条件の検査の読み方と失敗のしかた、D 節に足した 11 行、tech-debt の書き方)。対象は `git diff 49ac046c..HEAD`(7 ファイル、+236/-72)。テスト、静的解析、仕様との照合は行っていない。probe は scratch に置いた guard の写し(base の 4 ファイルと `lib_json.sh`、HEAD から 1 か所ずつ変えた写し 20 組)の awk に、コマンドを stdin で直接渡した(macOS の /usr/bin/awk 20200816)

## Evidence reviewed

- `cmd_pos`(`pre_bash_guard_commands.awk:122-144`)の判定が変わらないことを、コードと probe の両方で確かめた。
  - 足した行は `if (!(nm in WRAPPER)) return i`(`:131`)の 1 行だけ。base では 9 つの `nm == "…"` のどれにも当たらない名前は、連鎖の最後の `else return i`(`:141`)でその位置を返していた。`WRAPPER`(`rules.awk:407`)の 9 語と分岐の 9 語は同じ集合なので、どの名前でも行き先は base と同じ。`else return i` は、`WRAPPER` にあって分岐のない名前が来たときの受け皿として残り、`rules.awk:402-404` のコメントの「without either, the wrapper is taken as the command name」と合う。
  - `nm` は `cname` の値なので、`/usr/bin/env`、`=env`、`\env`、`"env"` も `env` として `WRAPPER` に当たる。`base()` が 256 文字を超える語に返す空文字は `WRAPPER` になく、base と同じく `return i` になる。awk の `in` は要素を作らない。
  - `cmd_pos` を呼ぶのは `judge`(`rules.awk:18`)だけで、`judge` は END の `lex_cmds` からしか呼ばれない。`WRAPPER` を埋める BEGIN はそれより前に終わる。`WRAPPER`・`wrapper_list` という名前はほかで使われていない。
  - 差分 probe: 20 の名前(9 つの包み、`builtin`・`sudo`・`sh`・`eval`・`git`・`echo`・`noglob`・`doas`、似た名前の `nice2`・`Env`・`envx`)× 9 通りの書き方(素、`/usr/bin/`、`=`、`\`、二重引用符、単引用符、`./`、`$'…'`、引用符で割った形)× 8 つの選択肢 × 7 つの続くコマンド × 3 つの前置き(なし、代入、`!`)と、それを `$(…)` に入れた形を、base と HEAD の awk に渡して出力と終了コードを比べた。結果は末尾の「差分 probe の結果」に書いた。
- 9 行の包みの行(`tests/test-pre-bash-guard.sh:894-908`)が、それぞれ自分の包みだけに頼っていることを mutation で確かめた。包みの名前を 1 つ `WRAPPER` から消した写し 9 組と、分岐を 1 つ当たらない名前に変えた写し 9 組で 9 行を流すと、18 組すべてで、消した包みの行だけが none になり、ほかの 8 行は `no_verify` のままだった。行のコメントの「Without the name in WRAPPER or without its branch, the wrapper is the command name, the -c string is not read, and no sentinel rule matches」のとおりで、`-n` は sentinel の 4 つの規則に入らず、`'git commit -n -m x'` は 1 語なので `scan_words` の `git` にも当たらない。既存の `env -u HOME sudo ls` などは、包みを飛ばさなくても `scan_words` と sentinel が deny にするので、この 9 行が初めて包みの飛ばし方を押さえている。
- heredoc の 2 行(`:957-963`、`:1072-1076`): `od -c` で、deny の行の本文の行は `\` が 2 つ、none の行は 3 つだった。bash、zsh、dash、macOS の /bin/sh のどれでも、2 つの方は `EOF` で本文が終わって次の行がコマンドとして動き、3 つの方は次の `EOF` とつながって `\EOF` になり、次の行が本文になる。コメントの説明と合う。`trailing_backslashes`(`lex.awk:560-565`)の下限を `q > a` にした写しでは 2 行とも逆になり(deny の行が none、none の行が `no_verify`)、`q >= a - 1` にした写しでは 2 行とも変わらない(行の前の文字は改行なので数に入らない)。コメントの「pin the lower bound」は、意味のある向きについて正しい。
- 不変条件の検査(`tests/test-pre-bash-guard.sh:1518-1572`): dump の BEGIN は `-f` の最後なので guard の BEGIN の後に動き、`exit` の後に動く guard の END は入力が空なので何も出さない(同じ節の `input []` の行が押さえている)。出力の行は `D ` と `W ` の接頭辞で振り分けるので、guard の END が何かを出しても名前の一覧に混ざらない。base の読み方ではそれが DATACMD の名前として入った。awk の失敗、DATACMD が空、`WRAPPER` が空は、それぞれ別の文言で FAIL になる。消した `commands_awk` を読む箇所は残っていない。
- 写し: `pre_bash_guard.sh`、`pre_bash_guard_commands.awk`、`pre_bash_guard_rules.awk` は root と `templates/base/` で `cmp` が一致。
- tech-debt の行(`docs/tech-debt/README.md:163`): 区切りでない `|` は 6 個(5 列)。足した文に `|` はない。(h) の閉じ方(「a wrapper branch moved out of `cmd_pos` stays in the check」「all 18 such mutants were red」)は、上の mutation の結果と合う。
- 秘密情報、デバッグ用の出力、TODO の残りはない。足したコメント行はどれも 78 桁以内。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | BEGIN の一覧を数えるコメント 2 か所が、`WRAPPER` を足す前のまま「NOEXEC と DATACMD」の 2 つだけを挙げる。どちらも、どのファイルに何があるかを示す案内の文で、`WRAPPER` を探す人はここを読む | `pre_bash_guard.sh:181-182`(「BEGIN with the command lists NOEXEC and DATACMD」)、`pre_bash_guard_rules.awk:2-3`(「Main has BEGIN (with the command lists NOEXEC and DATACMD)」)。`templates/base/` の写しも同じ | 2 か所とも「NOEXEC, WRAPPER and DATACMD」にし、`templates/base/` にも写す |
| LOW | readability | guard の先頭の item 3 は包みの 9 つの名前を散文で並べる。名前は `WRAPPER` と一致していて正しいが、同じ item の NOEXEC(「NOEXEC in pre_bash_guard_rules.awk」)や item 6 の DATACMD と違って、一覧の名前を挙げていない。また item 6 は `sh` と `builtin` も「a wrapper」と呼ぶが、どちらも `WRAPPER` になく、`cmd_pos` はそこで止まる。この PR で「wrapper」は `WRAPPER` の語を指す名前になったので、ずれが目立つようになった | `pre_bash_guard.sh:32-34`(「the wrappers env (-i, -u NAME, NAME=value), command, exec, nohup, time, nice (-n N), timeout (and its duration), xargs (and its flags), and stdbuf」)、`:125`(「a wrapper such as env, command, sh, nice, builtin or exec」)。`:125` の文は base からあり、diff には入っていない | item 3 に「(WRAPPER in pre_bash_guard_rules.awk)」を足す。`:125` は「a wrapper such as env, command, nice or exec, a shell such as sh, builtin,」のように分ける。急がないので、次に guard の先頭を直すときでよい |
| LOW | maintainability | tech-debt の行で、(h) と (i) の閉じ方がそろっていない。Debt の列では (h) の本文を `~~` で消しているが、(i) の本文は消さずに「Closed in …」の 1 文を後ろに足しただけで、「No row kills a mutant …」「The test suite stays at 2068/0 under that mutant」が現在形のまま残る。Impact と Trigger の列は (h) と (i) の両方を消している。また Trigger の列に足した「a new wrapper goes into `WRAPPER` and gets its own row of `edge_deny`」は、返済のきっかけではなく決まりごとで、「;」で並ぶきっかけの列挙に混ざると「新しい包みを足すこと」がきっかけのように読める | `docs/tech-debt/README.md:163` の Debt の列の (i)(「(i) No row kills a mutant that changes the lower bound of `trailing_backslashes` … Closed in test/guard-test-gaps (468fc73c): it added those two rows」)、Trigger の列(「(h) and (i) done in test/guard-test-gaps; a new wrapper goes into `WRAPPER` and gets its own row of `edge_deny`; or a report from a gawk or busybox awk host」) | Debt の列の (i) も (h) と同じく本文を `~~…~~` で消してから「Closed in …」を続ける。Trigger の列の決まりごとの文は外す(同じことは `rules.awk:402-406` のコメントに書いてある)か、「(h) and (i) done in test/guard-test-gaps (a new wrapper goes into `WRAPPER` with its own row of `edge_deny`)」のように括弧に入れる |

CRITICAL、HIGH、MEDIUM はない。3 件とも判定には関係しない。

細かい点(直さなくてよい): F 節の label の「(both lists read at run time from the three .awk files)」は、直前に DATACMD・予約語・`WRAPPER` の 3 つが出てくるので、「both」がどれを指すか一瞬迷う。予約語はテストの中の一覧なので、「(DATACMD and WRAPPER read at run time …)」と書けば迷わない。テストの先頭の F 項目(`:92-95`)も同じ。

## Positive notes

- 判定の変更を、`cmd_pos` の 1 行と BEGIN の 1 つの一覧に絞っている。分岐の中身(`skip_env`、`skip_opts` の引数など)には触れていないので、差分 probe で比べる範囲が小さく済んだ。
- 包みの行に `sudo` ではなく `sh -c` の中の `-n` を使ったので、sentinel と `scan_words` のどちらにも当たらず、行が包みの飛ばし方だけに頼る。18 組の mutation でそのとおりに赤くなった。
- 不変条件の検査は、ソースの書き方ではなく guard が実際に使う配列を読むようになり、dump の出力に接頭辞を付けたことで、guard の END の出力が名前に混ざる道もなくなった。
- heredoc の 2 行は、どちらも sentinel に当たらない `-n` を使い、行の形をそろえてあるので、`trailing_backslashes` の下限が変わったときだけ逆になる。

## Coverage gaps

- 差分 probe は macOS の /usr/bin/awk だけで流した。gawk と mawk では流していない(plan の Progress には implementer が gawk と mawk でテストを流した記録がある)。
- `WRAPPER` の名前と分岐が 1 対 1 であることを機械的に確かめる仕組みはない。今の 9 つは D 節の 9 行が押さえるが、10 個目を足すときに行を足すことは、`rules.awk:402-406` のコメントと tech-debt の Trigger の列が求めているだけ。分岐のない名前は `else return i` で今と同じ扱いになり、名前のない分岐は届かないだけなので、どちらも判定を悪くする向きではない。テストの網の話なので /test に任せる。
- `commands.awk` が `rules.awk` の `WRAPPER` に頼るようになった。古い `rules.awk` と新しい `commands.awk` を混ぜて置くと、`WRAPPER` が空になって包みの後ろの `sh -c` の文字列が読まれなくなる(sentinel の 4 つの規則と `scan_words` は残る)。guard の先頭は 3 つの `.awk` を一緒に置くことを求めていて、DATACMD も前から同じ形で `rules.awk` に頼っているので、指摘にはしていない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しい tech-debt はない。上の LOW はどれもこの PR の中で直せる大きさなので、`docs/tech-debt/` には足していない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はない。LOW 3 件はコメントと tech-debt の書き方で、判定には関係しない)
- Follow-ups: LOW 1 と LOW 3 はこの PR の中で直すことを勧める(コメント 2 か所と、tech-debt の 1 行)。LOW 2 は次に guard の先頭のコメントを直すときでよい。

## 差分 probe の結果

20 の名前 × 3,024 形 = 60,480 形で、base と HEAD の出力(規則の名前と終了コード)の違いは 0 だった。9 つの包みの名前も、包みでない 11 の名前も、どの書き方・前置き・`$(…)` の中でも同じ判定になった。macOS の大文字と小文字を区別しないファイルシステムで `env` と `Env` の結果のファイルが重なったので、この 2 つは別の名前のファイルで流し直し、どちらも違いは 0 だった。
