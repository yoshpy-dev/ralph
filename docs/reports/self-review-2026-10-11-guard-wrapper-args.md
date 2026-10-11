# Self-review report: guard-wrapper-args

- Date: 2026-10-11
- Plan: docs/plans/active/2026-10-11-guard-wrapper-args.md
- Branch: test/guard-wrapper-args(base 577ebeaf、review は HEAD 76f2f826)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1
- Scope: diff の品質だけを見た(コメントと FAIL/PASS の文言が検査の中身と合うか、V-1 の例の正確さ、「`edge_deny` の 9 行」を前提にした文の残り、tech-debt の文の正確さ、新しいループの bash の引用)。対象は `git diff 577ebeaf...HEAD`(計画の 113 行を除くと 2 ファイル、`tests/test-pre-bash-guard.sh` と `docs/tech-debt/README.md`)。テスト、静的解析、仕様との照合は行っていない。probe は scratch に置いた guard の写し(1 か所ずつ変えたもの)に、コマンドを stdin で直接渡した(macOS の awk、jq あり・なし)。テストのファイルは動かしていない

## Evidence reviewed

- 判定を変えていないこと。`git diff --stat` は `tests/test-pre-bash-guard.sh`、`docs/tech-debt/README.md`、計画の 3 ファイルだけで、`.claude/hooks/`、`templates/` は diff にない。`edge_deny` から外した 9 行と `wrapper_deny` の 9 行は、base の版と HEAD の版を `cmp` で比べて一致した(`:1067-1075`)。`wrapper_deny` は `edge_deny` の `check_modes` の直後、`edge_none` の前に置かれ(`:1037`、`:1077`)、`check_modes D deny absent` の引数は同じ。`git diff --check` は空、足したコメント行はどれも 78 桁以内、TODO やデバッグ用の出力はない。
- V-1 のコメントの例(`:1045-1055`)。引用した 5 つの文字列は、どれも書いてある場所にある(`nice -n 5 sudo ls` が `:911`、`xargs -I{} sudo ls {}` が `:918`、`xargs -I{} flock /tmp/l git push -f` が `:878`、いずれも D の `edge_deny`。`echo 'sudo ls' | xargs -I{} sh -c {}` が `:521`、`echo x | xargs sh -c 'git reset --hard'` が `:580`、いずれも B)。`cmd_pos` の `xargs` の分岐(`pre_bash_guard_commands.awk:140`)を消した guard の写しで、コメントの主張を直接確かめた: コメントが挙げる `xargs` の 4 つ(`echo 'sudo ls' | xargs -I{} sh -c {}`、`echo x | xargs sh -c 'git reset --hard'`、`xargs -I{} sudo ls {}`、`xargs -I{} flock /tmp/l git push -f`)と、挙げていない `xargs -n 1 sudo ls` は jq あり・なしとも deny のまま、`xargs sh -c` と `xargs -I{} sh -c` の `git commit -n` の形は jq あり・なしとも none。「both paths」も正しい。
- `nice` の分岐が引数を読み違えた写し(`skip_opts` の値を取る旗を `n` から空にした)でも確かめた。`nice -n 5 sudo ls` は deny のまま、`nice -n 5 sh -c …` の行は none。tech-debt の文の `nice -n 5 sudo ls` の例は、文が根拠に挙げる「`xargs` の分岐を消した写し」の結果ではなく、この `nice` の写しの結果で成り立つ。事実としては正しい。
- 「`edge_deny` の 9 行」を前提にした文の残り。`grep -n 'edge_deny\|wrapper_deny\|nine'` で、`edge_deny` が残るのは `:826`(定義)、`:1037`(流す行)、`:1047`(V-1 の例の置き場所)、`:1126`(別の行の話)だけ。ヘッダーの D 項目(`:59-61`)、F 項目(`:99-111`)、`:920` の指示、F 節の PR #219 の検査の上のコメント(`:1626-1640`)は、どれも `wrapper_deny` に向いている。`docs/` と `templates/` に `edge_deny` の文字列は残っていない(計画と過去の報告を除く)。
- 新しいループの bash の引用。`${row%% *}` は二重引用符の中でも、` *` が glob として働き、1 語目を返す。`${row%"$wrapper_row_tail"}` は引用したパターンなので、末尾を文字どおりに削る。`case "$row" in *"$wrapper_row_tail")` も同じ。bash 3.2.57(macOS の `/bin/bash`)で 6 つの行(`nice -n 5 …`、`time -o 'a b' …`、名前に `*` を含む `x*y …`、末尾に空白のある行、空白のない `env`)を通し、1 語目と末尾の判定と対照の組み立てが期待どおりだった。スクリプトは `set -u` だけで `set -e` はなく、`[ … ] && found=yes` の形は止まらない。`for p in jq no-jq` のラベル、`record_skip`、`read -r got rc <<< "$(run_guard …)"` は PR #219 の検査と同じ書き方。
- 集合の比較(`:1701-1728`)。`WRAPPER` が空なら FAIL、行のない名前と一覧にない 1 語目と末尾の不一致を 1 回の FAIL にまとめて出す。文言は、足し忘れた人に「`wrapper_deny` に行を足す。引数を取るなら `sh -c` の前に置く」、一覧から消えた名前に「消した、または一覧にない包みの行」を案内していて、検査の目的と合う。PASS の文言は「行があることと形だけを確かめた。行が引数を押さえているかは見ていない」と書き、検査が強制する範囲を超えていない。
- tech-debt の行(`docs/tech-debt/README.md:163`)。表の列は base と同じ 5 列(`\|` にしたので、区切りでない `|` は足していない)。`~~` は 14 個から 16 個で偶数。閉じの文の中身は、`:1701-1759` のコードと合う(両方向の比較、末尾の形、jq あり・なしの対照)。足した Related の `docs/plans/active/…` の末尾は、過去の 5 行と同じ「`/pr` rewrites this path when it archives the plan」の書き方。Trigger の列は前の PR で打ち消し済み。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | (1)「引数の読み違えで落ちるのはこの行だけ」と読める文が、9 行すべてには当てはまらない。V-1 の直しの最後の文「only these rows fail when a branch misreads its wrapper's own arguments」(`:1054-1055`)、ヘッダー F 項目の「a hand-written row of wrapper_deny in D does」(`:105`)、F 節のコメントの「the rows of wrapper_deny do」(`:1636`)は、9 行のどれもが包みの引数の読み方を押さえるように読める。実際に引数を `sh -c` の前に置くのは `nice -n 5`、`stdbuf -o0`、`timeout 5` の 3 行だけで、`env`、`command`、`exec`、`nohup`、`time`、`xargs` の 6 行は引数なし。直後の段落(`:1062-1064`)は 3 行を名指しするので、読めば分かる。ただ 3 か所の文だけを読むと、6 つの包みの引数も押さえられていると取れる | probe(`time` と `exec` の `skip_opts` の旗を空にした写し): `time sh -c …` と `exec sh -c …` は deny のまま(新しい検査の末尾・対照の条件も満たす)。`time -o f sh -c …` と `exec -a n sh -c …` は none。この引数付きの形は、`grep` で `tests/test-pre-bash-guard.sh` のどの行にも見つからない(`time -o`、`exec -a` の行はなく、`env -u`、`xargs -I{}` の行はあるが `sudo` か force push を含む sentinel つきの形)。`git commit -n` の形で包みの引数を付けた行は、`nice -n 5`、`stdbuf -o0`、`timeout 5` の 3 行だけ | 3 か所の文に「引数付きの行(`nice -n 5`、`stdbuf -o0`、`timeout 5`)」と名指しする。`:1054-1055` は「fail when a branch misreads its wrapper's own arguments: only the rows that carry such arguments do (nice -n 5, stdbuf -o0, timeout 5)」 |
| LOW | maintainability | (2)tech-debt の (h) は打ち消して閉じたが、残りはどこにも Open として残らない。閉じの文は「a row without arguments passes」と言い、検査も本当に強制するのは行の存在までなので、書いていること自体は正しい。ただ (1) のとおり、今ある 9 行のうち 6 行が引数なしで、`env -u`、`exec -a`、`time -o`、`xargs -I{}`、`command -p` といった実在の引数の読み方は、どの行も押さえない。閉じの文は「author's call」とだけ書き、いま誰も押さえていない事実を書かない。Impact の列と Trigger の列にも、この残りは出ない。また、閉じの文の「(five mutations, all caught; test report of this PR)」の根拠は、今は計画の Progress checklist にしかなく、Related にも test report の path がない(前の PR の review でも同じ指摘をした) | `docs/tech-debt/README.md:163` の (h) の閉じ(「Closed in test/guard-wrapper-args (78616500): …」)、Related の列の末尾。probe は (1) と同じ | 閉じの文に 1 文足す。例: 「Today only the `nice -n 5`, `stdbuf -o0`, and `timeout 5` rows carry arguments; no row puts `env -u`, `exec -a`, `time -o`, or `xargs -I{}` before `sh -c`」。/test の report ができたら、Related に `docs/reports/test-2026-10-11-guard-wrapper-args.md`(Mutation)を足す |
| LOW | readability | (3)F 節のコメントの「The forms give no wrapper an argument of its own (nice -n 5, stdbuf -o0)」(`:1635-1636`)は、`timeout` で偽。2 形目の `timeout 5 sh -c …` の `5` は、同じコメントの数行上(`:1632-1634`)が「timeout takes the next word as its duration」と書く、`timeout` 自身の引数。結論(2 形では引数の読み方を押さえない)は、1 形目が `+ 1` なしでも deny になる(前の PR の review で probe 済み)ので成り立つ | `:1632-1636`。`:1643-1646` の 2 形 | 「give no wrapper an option of its own (nice -n 5, stdbuf -o0), and the 5 of the second form is timeout's duration, which the first form makes optional (it is denied without it)」の方向に。または括弧を「options」に絞る |
| LOW | readability | (4)数と「書いた時点」の書き方が、新しい検査が入って半端になった。`:1039-1040` の「one row for each name WRAPPER had when these rows were written (…; nine today)」と、ヘッダーの D 項目(`:59-61`)の同じ言い回しは、9 行を過去に固定された集合のように書く。いまは F の検査が、いまの `WRAPPER` のすべての名前に 1 行以上を強制する(`:1057-1059` のコメントが書くとおり)ので、規則は「名前ごとに 1 行以上」で、書いた時点の名前の話ではない。「nine today」は 2 か所(`:61`、`:1040`)と tech-debt の「The nine hand-written rows stay」で、10 個目を足した時点で古くなる。ヘッダー D 項目の「wrappers (with wrapper_deny: hand-written rows of a wrapper before sh -c 'git commit -n -m x', one for each name …)」も、括弧の中の主語と動詞がつながらず、読みづらい | `:59-61`、`:1039-1040`、tech-debt の閉じの文 | 数を外して規則を書く。例: 「One row or more for each name of WRAPPER, written by hand (the F check below requires it)」。ヘッダー D 項目は「wrappers (the wrapper_deny rows: a hand-written row of each wrapper before sh -c 'git commit -n -m x')」 |
| LOW | naming | (5)新しい変数の名前が既存の名前の系統とぶつかる。`wrapper_row_forms`(`:1643`)は、`WRAPPER` の名前から作る「行」の形を指す。新しい `wrapper_row_tail`、`wrapper_row_control`(`:1699-1700`)は、手書きの `wrapper_deny` の行に使う。`grep wrapper_row` が 2 種類の行を混ぜて返し、ラベルの「F. wrapper row from the run-time WRAPPER」(作った行)と「F. hand-written wrapper rows (wrapper_deny)」(手書きの行)の区別も名前に出ない | `grep -n wrapper_row tests/test-pre-bash-guard.sh` の 17 行(新しい 2 変数は 10 行) | `wrapper_deny_tail`、`wrapper_deny_control` にする。変数は 2 つ、10 行 |
| LOW | exception-handling | (6)対照の FAIL の文言は、判定が none 以外のときだけ成り立つ。条件は `[ "$got" = none ] && [ "$rc" = 0 ]`(`:1753`)で、`got=none` のまま `rc` が 0 でない場合(guard が途中で落ちた)と `got=unparsed` の場合も同じ文言に入り、「its control … -> none (exit 1), not none: the deny of this row does not come only from …」のように、自分で食い違う。PR #219 の検査の FAIL の文言は、判定と終了コードを並べるだけで、原因を言い切らない | `:1753-1757`。PR #219 の検査の FAIL は `:1674` | 「…-> $got (exit $rc), expected none with exit 0」まで出し、「deny の出どころが中の `git commit -n` だけではない」の説明は `got=deny` のときだけ足す |

CRITICAL、HIGH、MEDIUM はない。LOW 6 件は、判定にも guard の動きにもテストの結果にも関係しない(コメントと文言と名前、tech-debt の 1 文)。

## Positive notes

- 手書きの行を別の配列に移すだけで、guard、ラベル、D 節の流し方を変えていない。9 行は 1 文字も変わらず、`check_modes` の引数が同じなので、ラベルも件数も変わらない。
- 検査が強制する範囲を、コメント、PASS の文言、FAIL の文言、tech-debt の閉じの文のどれにも「行の存在と形まで」と書いている。計画の Non-goals(テストは実際のツールのオプションを知らない)と合い、検査の中身より強い言い方はこの diff にない(上の 1、3 は、その中の細かい文の正確さ)。
- 末尾の形を決め、末尾を無害にした対照が none であることを確かめる設計は、sentinel に掛かる中身の行と、包みの引数に sentinel の語を入れた行(`time -o 'sudo log' sh -c …`)の両方を捕まえる。この例のコメントでの引き方(末尾は保つが、対照は `sudo` の語で deny になる)も、計画の AC2 (e) と合う。
- 集合の比較を両方向にし、問題を 1 回の FAIL にまとめるので、複数の行をいっぺんに直せる。FAIL の文言は、足し忘れ、名前の消失、形の崩れのそれぞれに、直す先(`wrapper_deny`)を示す。
- V-1 の直しは、「この行は引数の読み違えで落ちない」という主張を、`xargs` の分岐を消した写しで確かめた結果(5 つが deny のまま、2 つが none)に結びつけて書き、事実から外れていない。

## Coverage gaps

- テストのファイル全体は動かしていない(この step の範囲外)。2,127 件が通ることと、mawk・gawk での結果は /test で確かめる。probe は macOS の awk だけ。
- 新しい検査が FAIL になる 5 つの mutation(計画の AC2 (a)〜(e))は、テストのファイルの中で FAIL と記録されることを見ていない。guard の写しで、前提(`xargs` の分岐を消した結果、`nice` と `time` と `exec` の引数の読み違え)を確かめただけ。
- 6 つの引数なしの包みの引数の読み方は、この PR の後も誰も押さえない(上の 2)。計画の Non-goals にある範囲で、この PR が悪くしたのではない。

## Recommendation

- Merge: yes(最初の review の判定。CRITICAL、HIGH、MEDIUM なし、LOW 6。再確認のあとの判定は末尾の「再確認」の Merge)
- Follow-ups:
  - 1、3、4、5、6 はテストのコメント、文言、名前の直しで、判定に関係しない。直すなら、pipeline を回し直す前に 1 回の fix commit にまとめるのがよい。直さない場合は、次にこのテストのファイルを触る機会に回してよい。
  - 2 の tech-debt の文と Related の足しは、`docs/tech-debt/README.md` は記録だけの修正の例外の対象外なので、/sync-docs の段で直す。test report の path は /test の後に足す(前の PR と同じ手順)。
  - 任意: `.claude/hooks/pre_bash_guard_rules.awk`(と `templates/base/` の写し)の「a D row of tests/test-pre-bash-guard.sh pins each wrapper, and its F section reads this list」は、今も偽ではない。「pins each wrapper」は、分岐の有無を指す文として読めば正しい。guard を次に直す機会に、F が一覧と手書きの行を突き合わせることも書く。この PR は guard のファイルを変えない決まりなので触れていない。

## 再確認(888ec6f8、160eef84、d2cb7794)

- Date: 2026-10-11
- Branch: test/guard-wrapper-args(HEAD d2cb7794)
- Scope: 計画が広がり(23b1ac05、承認 digest e69b20224a60)、LOW 2 を受けて 16 行を足した。`git diff a57b6758..HEAD` を、diff の品質だけで見た(テスト 141 行の差、tech-debt の 1 行、計画の Progress)。テスト、静的解析、仕様との照合は行っていない。guard のファイルは変わっていない(`git diff a57b6758..HEAD -- .claude templates scripts` は空)。probe は scratch に置いた guard の写し(macOS の awk、jq あり)に、行を stdin で直接渡した

### 確かめたこと

- 変更の範囲。足したコメント行はどれも 78 桁以内、`git diff --check` は空。`nine` の語はテストのファイルから消え、`wrapper_row_*` は `wrapper_row_forms` の系統だけになった(`grep -n`)。tech-debt の行は 5 列のまま(`\|` の数は base と同じ)、`~~` は 16 個で偶数、「The nine hand-written rows stay」の文は外れている。
- 16 行(`:1090-1105`)。1 語目はどれも `WRAPPER` の名前で、末尾は `sh -c 'git commit -n -m x'`、集合の検査と対照(末尾を `-m x` に替えた形)の条件に合う。`%e`、`{}`、`=` は `$'…'` の中で、ラベルとペイロードの組み立て(`printf` の引数渡し、`json_escape`)に支障を出さない形。計画の Assumptions のとおり、HEAD の guard で 16 行とも deny。
- 引数の読み方を変えた写しを 31 通り作り、25 行(元の 9 行と足した 16 行)を流して、どの行が none になるかを見た。引数なしの行(`env sh -c …`、`command sh -c …`、`nohup sh -c …`、`time sh -c …`、`xargs sh -c …` など)は、31 通りのどれでも落ちなかった。引数を置く行は、その引数だけで落ちる。たとえば `time` の旗の指定を空にすると `time -o f` と `time -f %e`、`exec` の指定を空にすると `exec -a n`、`timeout` の `+ 1` を消すと `timeout 5`、`timeout -s KILL 5s`、`timeout -k 1 5` が落ちた。落ちた写しは 16 通り、どの行も落ちなかった写しは 15 通り(下の 7)。
- LOW 3 の文「a timeout branch that skipped no duration would deny the first form instead」を写しで確かめた。`timeout` の `+ 1` を消した写しで、`timeout sh -c …`(1 形目)は deny、`timeout 5 sh -c …`(2 形目)は none。コメントのとおり。

### LOW 1〜6 の状況

| LOW | 状況 | 確かめた点 |
| --- | --- | --- |
| 1 | 修正済み | `:1065-1067` は「only rows of this kind fail when a branch misreads its wrapper's own arguments, and only for the arguments they put before sh -c」。ヘッダー F 項目は「of the rows of wrapper_deny in D, only those that put such arguments before sh -c do, each for the arguments it puts there」。F 節のコメント(`:1664-1672`)の「the rows of wrapper_deny that put such arguments before sh -c do」も同じ。上の 31 通りの結果(引数なしの行は落ちず、引数を置く行はその引数だけで落ちる)と合う |
| 2 | 修正済み。ただし閉じの文に新しい指摘がある(7) | 閉じの文が、最初の 9 行のうち引数を置くのは 3 行だけだったこと、`time` と `exec` の旗を空にすると緑だったこと、足した 16 行、(f) の結果(8 件)、「分岐が載せていないツールの旗はどの行も押さえない」ことを書く。残りの Related の test report の path は、/test のあとに足す(前の PR と同じ手順) |
| 3 | 修正済み | `:1664-1672`。「give no wrapper an option」に直り、`timeout` の 5 の扱いを `+ 1` を消した写しの結果つきで書く。上の確認のとおり正確 |
| 4 | 修正済み | ヘッダーの D 項目は「wrapper_deny holds the hand-written rows that run a wrapper before sh -c 'git commit -n -m x', bare or with arguments of its own; F requires one row or more for each name of WRAPPER」。`:1040` 付近のコメントは「one row or more for each name of WRAPPER, written by hand (the F check below requires it)」。数が消えた。見た目だけの点: ヘッダーの組み直しで「and unknown runners,」が短い行になる。直さなくてよい |
| 5 | 修正済み | `wrapper_deny_tail`、`wrapper_deny_control`(`:1736` 付近)。コメント、ラベル、FAIL の文言、`case` の参照まで、旧名は残っていない |
| 6 | 修正済み | `:1793-1795`。`why=""` を毎回初期化し、`got=deny` のときだけ理由を足す。FAIL の本文は「expected none with exit 0」まで。`got=none` で `rc` が 0 でない場合や `unparsed` の場合は、理由なしで判定と終了コードだけが出る。この分岐は guard を変えずに出せないので、実行では見ていない(計画の Progress も同じ書き方)。コードを読んで確かめた |

### 新しい指摘

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | (7)「分岐ごと、値を取る引数の書き方ごとに 1 行以上」という書き方が、足した 16 行の実際の範囲より広い。同じ書き方が 3 か所にある。行の群の上のコメント(`:1088-1089`)、`wrapper_deny` の上のコメント(`:1049-1056`)、tech-debt の閉じの文。分岐が読む書き方のうち、どの行も押さえないものが 4 組ある(詳細は下) | 引数の読み方を変えた写し 31 通りのうち、15 通りでどの行も落ちなかった。うち 10 通りが行の不足、5 通りは行を足す必要のないもの(詳細は下) | 次のどちらか。(i) 行を 7 行足し、閉じの文を足した範囲に合わせる。(ii) 行を足さず、3 か所の言い方を実際に押さえるものの名指しに直し、押さえないものを閉じの文に書く。verify の前なので、どちらでも再実行の費用は増えない |

#### 指摘 7 の詳細

分岐が読む書き方のうち、どの行も押さえないもの:

- (a) `timeout` の長いオプションと次の語(`--signal KILL`、`--kill-after 1`)。長いオプションの行は `time`、`nice`、`stdbuf` だけ。`timeout` の分岐の一覧(`" --signal --kill-after "`)を空にした写しで、25 行のどれも落ちなかった。
- (b) `stdbuf` の短い旗と次の語(`-o L`)。`stdbuf -o0` は値が旗に付く形で、`takes_next` は旗の集合に入っていても入っていなくても 0 を返すので、集合 `"ioe"` を空にした写しでも落ちなかった。
- (c) `env`、`command`、`xargs` の `--`。`nohup --` が押さえるのは `skip_opts` の `--` の読み違い(`return i` にした写し)だけで、その行の削除は、長いオプションの経路が同じ結果を返すので等価の写しになる。`skip_env`、`skip_command`、`skip_xargs` の `--` を `return i` にした写しは、どれも落ちなかった。
- (d) `env` の長いオプション(`--unset`、`--chdir`)と、`xargs` の値を取る長いオプション(`--max-args` など)。それぞれの判定を落とした写しで落ちなかった。
- 加えて、`xargs -I{}` と `stdbuf -o0` の行は、31 通りのどれでも落ちなかった。値が旗に付く形は、旗の集合に関係なく同じ結果になる。

どの行も落ちなかった 15 通りのうち、行の不足を示す 10 通りが上の (a)〜(d)。ほかの 5 通りは行を足す必要がない。`skip_opts` の `--` の行の削除(等価)、`skip_env` の代入の分岐の削除(`cmd_pos` のループの先頭が代入を飛ばすので、等価とみられる。未確認)、`skip_opts` の数字の旗(`nice -5` の旧式の形で、計画が足さないと決めた「man にある形」の外とみられる。未確認)、`time` の `--output` の 1 語(同じ一覧の `--format` の行が通る)、`xargs` の任意の値の旗 `ile`。

足す行の候補は、HEAD の guard ですべて deny で、対応する写しですべて none になった(行が押さえることを確かめた)。`timeout --signal KILL 5s sh -c …`、`stdbuf -o L sh -c …`、`env -- sh -c …`、`command -- sh -c …`、`xargs -- sh -c …`、`xargs --max-args 1 sh -c …`、`env --unset HOME sh -c …` の 7 行。

(i) の場合、25 行が 32 行になり、件数は D 節で 14、F 節の対照で 14 増える。足したあとも、一覧の旗を 1 つずつは押さえない(`--kill-after` や `--chdir` など)ので、閉じの文の「each way」は、押さえる範囲(書き方の種類ごと、分岐ごと)に言い直す。(ii) の場合の言い方の例は「a short flag and the next word for exec, time, nice and timeout; a long option and the next word for time, nice and stdbuf」で、押さえないものとして (a)〜(d) を閉じの文に書く。なお、閉じの文の「An option of the real tool that a branch does not list is pinned by no row」は、分岐が載せていない旗の話で、(a)〜(d) は分岐が載せている書き方の話なので、今の文ではカバーされていない。

## 再確認の結果

- 修正は LOW 1〜6 のすべてで確かめた。新しい指摘は MEDIUM 1(7)。判定の変更、guard の変更、テストの結果に関係しない(コメント、tech-debt の文、行の追加の範囲の話)。
- 足した 16 行そのものは、1 語目、末尾、対照、`$'…'` の書き方に問題がない。名前の揃えと対照は、25 行のどれも新しい検査を満たす形。

### Coverage gaps(再確認)

- テストのファイルは動かしていない(この step の範囲外)。2,191 件が通ることと mawk・gawk の結果は /test で確かめる。probe は macOS の awk だけ。
- 16 行を足したので、どの名前にも 2 行以上ある。1 行を消しても集合の検査は FAIL せず、その行が押さえていた書き方が黙って外れる。計画の Progress が「AC2 (b) の読み方」として記録している範囲で、この PR が決めたこと。
- 31 通りの写しは、`cmd_pos` と `skip_*` の引数の読み方の部分だけ。`takes_next` 自体、`$'…'` の引数、`-S` の経路(`queue`)は見ていない。

### Recommendation(再確認)

- Merge: yes(CRITICAL、HIGH なし。MEDIUM 1(7)、LOW は 1〜6 がすべて修正済み)
- Follow-ups:
  - 7 は、verify に進む前に直すのがよい(上の (i) か (ii))。(i) なら行の追加とコメント、(ii) ならコメント 3 か所と tech-debt の閉じの文。どちらも判定を変えない。
  - 2 の Related の test report の path は、/test の report ができたあとに足す。
  - 任意(前のまま): `.claude/hooks/pre_bash_guard_rules.awk`(と `templates/base/` の写し)の「a D row of tests/test-pre-bash-guard.sh pins each wrapper, and its F section reads this list」は、今も偽ではない。guard を次に直す機会に、F が一覧と手書きの行を突き合わせることも書く。
