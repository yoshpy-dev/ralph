# Self-review report: guard-wrapper-rows

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-guard-wrapper-rows.md
- Branch: test/guard-wrapper-rows(base 382c18c8、最初の review は HEAD 4234f2a3、再 review は HEAD fc97d108)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1
- Scope: diff の品質だけを見た(`run_guard` の切り出しで `decide` の動きが変わらないか、新しい検査の名前・文言・失敗のしかた、コメントの正確さ、tech-debt の書き方)。対象は `git diff 382c18c8..HEAD`(計画の 106 行を除くと 2 ファイル、`tests/test-pre-bash-guard.sh` と `docs/tech-debt/README.md`)。テスト、静的解析、仕様との照合は行っていない。probe は scratch に置いた guard の写し(HEAD から 1 か所ずつ変えた 13 組)に、コマンドを stdin で直接渡した(macOS の /usr/bin/awk、jq あり・なし)。テストのファイルは動かしていない

## Evidence reviewed

- `decide` の切り出し(`tests/test-pre-bash-guard.sh:248-270`)。base の `decide` と見比べて、変わる点は 2 つだけ。(1) `local out rc got` と `case` の判定が `run_guard` に移り、`PATH`・hook・payload が引数になった。判定の 4 分岐(none、deny、ask、unparsed)と `rc=$?` の取り方は同じで、`2>/dev/null` の位置も同じ。(2) 結果の書き込みが、関数の最後の `printf > file` から、呼び出し側の `run_guard … > "$workdir/r.$i"` に移った。このため結果のファイルは run の前に空で作られる。読む側は `result_of`(`:286-292`)、キューの判定ループ(`:1706`)、G 節の `read -r old_got _` で、どれも `wait` のあとに読む。run が途中で殺された場合は、base では `missing 1`、今は空の 2 欄になるが、どちらも `[ "$rc" = 0 ]` を満たさず FAIL になり、G 節の「old deny、new none」の比較にも入らない。緑の run では差が出ない。
- 新しい検査(`:1594-1642`)の判定。2 形の「どちらか」を deny かつ exit 0 で見ていて、`got` と `rc` を `read -r got rc <<< "$(run_guard …)"` で受ける。`WRAPPER` が空なら FAIL(`:1611-1612`)、jq がなければ jq 側だけ SKIP(`:1617-1621`)で、no-jq 側は動く。ラベルの `[jq]` / `[no-jq]` と SKIP の書き方は `check`(`:297-310`)と同じ。変数 `w`、`p`、`k`、`got`、`rc`、`escaped`、`label` はどれも G 節以降で代入してから使うので、グローバルに出しても漏れない。
- 新しい検査の前提を guard の写しで確かめた。9 つの包み × 2 形 × jq あり・なしの 36 回は、`timeout` だけ 2 形目(`5 sh -c …`)で deny、ほかの 8 つは 1 形目で deny、もう一方の形は none だった。2 形でちょうど一方が deny になる、というコメントと計画の前提と合う。36 回の合計は 2 秒。
- 新しい検査が見逃さないことを、guard の写しで確かめた。`cmd_pos` の分岐を 1 つ消した 9 組(env、command、exec、nohup、time、nice、stdbuf、timeout、xargs)と、`WRAPPER` に `chrt` を足して分岐を足さない 1 組で、2 形とも none になった。10 組すべて、jq あり・なしの両方でこの検査が FAIL になる。計画の AC2 の結果(`chrt` と 9 通り)と合う。
- 引数の取り方を壊した写しの結果は Findings の 4。`timeout` の `+ 1` を消した写しと、`nice` の `n`(値を取る旗)を消した写しでは、1 形目が deny のままでこの検査は PASS になり、D 節の `timeout 5 sh -c …` と `nice -n 5 sh -c …` の行が none になる。
- 数字の確かめ。`WRAPPER` は 9 語(`pre_bash_guard_rules.awk:407`)で、コメントの「timeout takes the next word as its duration, the other eight read the next word as the command」は `cmd_pos`(`pre_bash_guard_commands.awk:131-140`)と合う。`edge_deny` の 9 行(`:920-928`)は `nice -n 5`、`stdbuf -o0`、`timeout 5` だけが素の形と違う。
- tech-debt の行(`docs/tech-debt/README.md:163`)。表の列は base と同じ 5 列(区切りでない `|` は足していない)。`~~` は 12 個から 14 個で偶数。足した Related の `docs/plans/active/2026-10-10-guard-wrapper-rows.md` は、`scripts/archive-plan.sh:54-90` の書き換えの対象になる形(後ろがバッククォート)で、`/pr` が archive へ書き換える。
- `templates/base/` に `test-pre-bash-guard.sh` の写しはない。`git diff --check` は空。足したコメント行はどれも 78 桁以内。秘密情報、デバッグ用の出力、TODO の残りはない。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | (1。35f00d33 で修正済み)検査が FAIL したときの文言が、直し方として「`wrapper_row_forms` に形を足す」ことだけを挙げる。この検査が捕まえるために作られた原因、つまり名前を `WRAPPER` に足して `cmd_pos` の分岐を足し忘れた場合は、文言のどこにも出てこない。FAIL を見た人は、テストの形を足す方向へ向かう。計画の Risks も同じ偏りで、「正しく足した包みでも FAIL になる」場合だけを書いていた。文言の後半の「wrapper $w on the $p path」はラベルの `$w` と `[$p]` の繰り返し | `tests/test-pre-bash-guard.sh:1638`。`chrt` を `WRAPPER` に足した写しでは 2 形とも none になり、出る文言は形を足すことだけを勧める | 原因を 2 つ並べる。例: 「`$w` が `WRAPPER` にあるのに `cmd_pos` に分岐がないと、その後ろの `sh -c` は読まれず none になる。分岐がある包みなら、引数の取り方が 2 形に合っていないので、`wrapper_row_forms` に形を足す」。`$w`・`$p` の繰り返しは削る |
| LOW | readability | (2。35f00d33 で修正済み)ヘッダーの D 項目(`:59`)と `edge_deny` の 9 行の上のコメント(`:910`)の「the nine names of WRAPPER」は、この PR が認める形(名前と分岐を足し、行は足さない 10 個目の包み)が入った時点で偽になる。直す前の「each name of WRAPPER」も同じ理由で偽になる。数を入れる文が、今回足した能力で古くなる | `:59`、`:910`。`:100` と `:1596` の「the nine rows」「the nine wrapper rows」は行の数なので、10 個目が入っても正しい | 数を `WRAPPER` に結びつけない。たとえば `:59` は「wrappers (a hand-written row for each wrapper before sh -c)」、`:910` は「One row per wrapper below, written by hand (nine today)」 |
| LOW | readability | (3。35f00d33 で修正済み。(a) の書き換えには新しい指摘がある、下の 6)コメントと文言に、実際より強い、または理由の足りない書き方が 3 つある。(a) `:917-919` の「the wrapper rows of F, which run the same rows for each name」。F が動かすのは `W sh -c …` と `W 5 sh -c …` の 2 形で、D の行とは同じでない(D の `nice -n 5 …` と `stdbuf -o0 …` は F が動かさない)。(b) `:250` の「deny (the exact deny JSON)」。`:259` の判定は固定の前半と後半の間に任意の `permissionDecisionReason` を許すので、「exact」ではない。(c) `:1602-1603` の「not through the queue, so the two forms are compared here」。「so」の前後がつながらず、キューを使わない理由が書かれていない。計画の Assumptions にある理由は、`run_queue` が F と G の後に走るので、「2 形のどちらか」を 1 か所で見るには F の中で直接動かすほうが短い、というもの | `:250`、`:259`、`:917-919`、`:1602-1603`、計画の Assumptions | (a)「…which run, for each name of the WRAPPER list read at run time, the plain form and a form with a duration」。(b)「deny (the guard's deny JSON, any reason)」。(c)「…not through the queue: run_queue runs after F and G, and the verdict here needs both forms' decisions together」 |
| LOW | maintainability | (4。テストのヘッダー側は fc97d108 で修正済み。tech-debt 側は未修正で、/sync-docs に回す)tech-debt の (h) の閉じの文の「so a new wrapper needs only its name in `WRAPPER` and its branch in `cmd_pos`」は、F の新しい検査が押さえる範囲より広く読める。F の 2 形が見るのは分岐が「ある」ことで、引数の読み方が正しいことではない。写しで `timeout` の `+ 1` を消すと、`timeout sh -c …` が deny になるので F は PASS のままで、D の `timeout 5 sh -c …` だけが none になる。`nice` の `n` を消した写しも同じ。9 つの包みは D の行が引数の読み方を押さえているが、10 個目の包みは、行を足さないと誰も押さえない。続く文の「The nine hand-written rows stay, because only they catch a name removed from `WRAPPER`」も、残す理由を 1 つしか挙げていない。ヘッダーの F 項目(`:101-103`)の「it covers a new wrapper without a hand-written row in D」も、分岐の有無についてだけ正しい | `docs/tech-debt/README.md:163` の (h) の閉じ(「Closed in test/guard-wrapper-rows (0eebc7e7): …」)。probe: `timeout` の `+ 1` なし: 形1 deny・形2 none・F PASS・D の行 none。`nice` の `n` なし: 同じ | 閉じの文を「a new wrapper needs its name in `WRAPPER` and its branch in `cmd_pos`, and an `edge_deny` row when it takes options or a duration: the rows of F pin that the branch exists, not how it skips its arguments」の方向に直す。`:101-103` も「…covers a new wrapper's branch without a hand-written row」に |
| LOW | readability | (5。未修正。任意なので見送る)新しい検査(`:1611-1642`)は、メインのスクリプトの上に 31 行をそのまま置き、`else`、`for w`、`for p`、`for k`、`if` の 5 段に入る。経路の選び方、SKIP、1 形ごとの実行、判定が 1 か所に混ざる。このファイルは `check` や `lib_case` のように、まとまった処理を関数にしている | `:1611-1642` | 任意。`wrapper_row_verdict <name> <path>` のような関数に、2 形を動かして「denied か、どの形がどう判定されたか」を返す部分を出し、ループには SKIP・PASS・FAIL の記録だけを残す |
| LOW | readability | (6。再 review で見つけた。未修正)`edge_deny` のコメントの「not the rows below: only here are nice -n 5 and stdbuf -o0 run」は、そのまま読むと偽。`nice -n 5 sudo ls` が `:903` に、`stdbuf -oL sudo ls` が `:905` にあり、`nice -n 5` はそこでも動く。書きたいのは、`sh -c` の前で動かすのはこの 9 行だけ、ということ。同じ文の「a form with a duration (W 5 sh -c ...)」も、8 つの包みでは `5` は期間でなくコマンド名として読まれる(F の検査のコメントは「timeout takes the next word as its duration, the other eight read the next word as the command」と正しく書いている) | `tests/test-pre-bash-guard.sh:922-923`、`:903`、`:905` | 「…not the rows below: only the rows below run nice -n 5 and stdbuf -o0 before sh -c」。「a form with a duration」は「a form with a leading 5 (a duration only for timeout)」 |

最初の review では CRITICAL、HIGH はなく、MEDIUM 1、LOW 4(上の 1 から 5)だった。MEDIUM 1 件は文言だけの変更で直せ、LOW の 4 件は判定にも guard の動きにも関係しなかった。再 review(下)で 1 から 3 の修正を確かめ、6 を新しく挙げた。

## 再 review(35f00d33、fc97d108)

対象は `git diff 336f5856 fc97d108`(`tests/test-pre-bash-guard.sh` と、計画の Progress の 5 行)。テストは動かしていない。

- 変更の範囲: 両方の版から `^\s*#` の行を取り除いて `diff` を取ると、実行行の差は `record_fail` の文字列 1 行(`:1643`)だけ。ほかの差はコメント。足したコメント行はどれも 78 桁以内。文字列は二重引用符の対が 1 組で、中に引用符、バッククォート、`$(` はなく、展開は `$label`、`$w`、`$p`、`$got_forms` だけ。guard のファイルは変わっていない(`git diff --stat` はテストと計画の 2 ファイル)。
- 1(MEDIUM): 修正済み。文言は先に「`$w` が `WRAPPER` にあって `cmd_pos` に分岐がないなら、後ろの `sh -c` は読まれない: 分岐を足す」、あとに「分岐があるなら引数が 2 形に合わない: `wrapper_row_forms` に形を足す」と、原因を 2 つ、この順に書く。勧めた形と同じ。残り: 文言の「wrapper $w on the $p path」はラベルの `$w` と `[$p]` の繰り返しのままで、勧めた削りは入っていない。直さなくてよい。
- 2(LOW、「the nine names」): 修正済み。ヘッダー(`:59-60`)は「nine hand-written rows before sh -c, one for each name WRAPPER had when they were written」、`edge_deny` のコメント(`:912`)は「One row for each name WRAPPER had when these rows were written (nine; …)」で、数が書いた時点に結びつく。10 個目が入っても偽にならない。`:101` の「The nine rows in D」は行の数なので、そのままで正しい。見た目だけの点: ヘッダーの組み直しで `:61` が 54 桁で終わり、次の行は元の幅のまま。直さなくてよい。
- 3(LOW、コメント 3 つ): (b)(c) は修正済みで正確。(b) `:252` は「the deny JSON with any permissionDecisionReason」で、`:259` のパターンと合う。(c) `:1607-1609` の「run_queue runs after F and G, and a queued case has one expected decision, while this check needs at least one of two decisions to be deny」は、`q_expect` の値(none、deny、any、`-`)に「2 つのうちどちらか」がないことと合う。(a) は書き換えたが、新しい不正確さが入った(6)。
- 4(LOW、tech-debt とヘッダー): テストのヘッダー側は修正済み。`:104-105` の「It does not check how the branch reads the wrapper's own arguments (a hand-written row in D does)」は、probe の結果(`timeout` の `+ 1` なし、`nice` の `n` なしで F は PASS、D の行は none)と合う。括弧の内側は、引数付きの行(`nice -n 5`、`stdbuf -o0`、`timeout 5`)にだけ当てはまり、素の 6 行は引数を読まないが、読み違いにはならない。tech-debt 側は依頼どおり /sync-docs に回す。`docs/tech-debt/README.md:163` の「so a new wrapper needs only its name in `WRAPPER` and its branch in `cmd_pos`」は、まだ元のまま。
- 5(LOW、関数に出す): 見送り(任意)。
- 6(LOW、新規): 上の表のとおり。`edge_deny` のコメントの「only here are nice -n 5 and stdbuf -o0 run」。判定には関係しない。
- 計画: Progress の「S2 完了」の行が S3b の後ろへ移っている(`bf3241dc` の hash が入った)。順序だけの変更で、内容は同じ。

## Positive notes

- `decide` の切り出しは、判定の 4 分岐と `rc` の取り方をそのまま移していて、キューを通す経路(`:269`)と直接動かす経路(`:1631`)が同じ関数を使う。判定の写しが 2 つに増える形を避けている。
- 「2 形のどちらか」の判定は、前提(9 つの包みでちょうど一方が deny)を guard で確かめたうえで選ばれていて、`timeout` のように次の語を飛ばす包みを、名前を特別扱いせずに通す。分岐を消した 9 組と、分岐のない 10 個目の名前が、すべてこの検査の FAIL になることを独立に確かめた。
- 文言に名前と 2 形の判定(`-> none (exit 0)`)を出すので、FAIL を見ればどの名前がどの経路で読まれなかったかが分かる。`WRAPPER` が空のときは FAIL にし、jq がないときは jq 側だけを SKIP にして no-jq 側を動かす。
- 9 行を残し、残す理由と、新しい検査との役割分担を `edge_deny` の上のコメントに書いた。ヘッダーの F 項目もこの分担に合わせて直してある。guard のファイルは触れていない。

## Coverage gaps

- テストのファイル全体は動かしていない(この step の範囲外)。2,108 件が通ることと、mawk・gawk での結果は /test で確かめる。probe は macOS の awk だけで、`for (k in WRAPPER)` の順序が awk ごとに違うため、結果の行の並びは awk によって変わる(どの行も独立なので判定には関係しない)。
- AC2 の「分岐を消した 9 通りと `chrt`」は、guard の写しに 2 形を直接渡して FAIL になることを確かめた。テストのファイルの中で FAIL と記録されることは見ていない。
- 引数の読み方を壊した写し(Findings の 4)は `timeout`、`nice`、`stdbuf` の 3 組だけ。`stdbuf` は、値を取る旗の指定を消しても D の行(`stdbuf -o0 …`、値が旗に付いた形)が deny のままだった。このファイルの `stdbuf` の行は `-oL` と `-o0` の付いた形だけ(`grep -n stdbuf` で確認)で、旗の値が次の語に分かれる形の行はない。これはこの PR の前からある範囲で、この PR は悪くしていない。

## Recommendation

- Merge: yes(CRITICAL、HIGH なし。最初の review は MEDIUM 1、LOW 4。fc97d108 の再 review のあと、直った指摘は 1、2、3、4 のテスト側、open は LOW 3 件で、4 の tech-debt 側、5(任意)、6)
- Follow-ups:
  - 4 の tech-debt 側は、tech-debt の 1 文で、記録だけの修正の例外には当たらない(`docs/tech-debt/README.md` は条件 2 の対象外)ので、/sync-docs の段で直す。
  - 6 は `edge_deny` のコメント 1 文の直しで、判定に関係しない。次にテストのファイルを触る機会に直せばよく、直すためだけに pipeline を回す必要はない。
  - /test の report ができたら、tech-debt の行の Related に `docs/reports/test-2026-10-10-guard-wrapper-rows.md`(Mutation)を足す。閉じの文の「a tenth name `chrt` and each of the nine branches removed, all caught」の根拠は、今は計画の Progress checklist にしかない(前の PR は Related に test report を挙げていた)。
  - 任意: `.claude/hooks/pre_bash_guard_rules.awk:402-405`(と `templates/base/` の写し)の「a D row of tests/test-pre-bash-guard.sh pins each wrapper, and its F section reads this list」は、今も偽ではない。guard を次に直す機会に、F が一覧から行を作ることも書く。この PR では guard のファイルを変えない決まりなので触れていない。
