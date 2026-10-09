# Self-review report: guard-debt-cleanup

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-debt-cleanup.md
- Branch: refactor/guard-debt-cleanup(base a0094fe5、HEAD c845d19a)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1
- Scope: diff の品質だけを見た(消した 2 つの規則の安全性、`trailing_backslashes` の等価性、名前とコメントの正確さ、不変条件の検査の読み方、`verify.local.sh` の段、tech-debt の書き方)。対象は `git diff a0094fe5..HEAD`(12 ファイル、+363/-124)。テスト、静的解析、仕様との照合は行っていない。probe は scratch に置いた guard の写し 4 組(base、HEAD、それぞれから allowlist の行を消した写し)に、45 形を 1 ファイル 1 形で awk に直接渡しただけ

## Evidence reviewed

- 消した 2 つの規則が許可リストの影にあったことを、コードを読んで確かめた。
  - `NODATA` を読むのは `in_data`(`commands.awk:245`)だけで、それを呼ぶのは END の `sentinel()` だけ。`NODATA` を 0 に戻す行は BEGIN にしかない。だから、予約語の規則が `judge` の前に立てていた `NODATA` を、`judge` の後の許可リストが立てても判定は同じ。
  - `cmd_pos` を呼ぶのは `judge`(`rules.awk:18`)だけで、`judge` は字句解析をやり直さない。`eval` と `-c` の文字列は `queue` に積まれて END で読まれ(`rules.awk:455-460`)、`$(...)` とバッククォートは外側のコマンドの `end_cmd` より前に読み終わる(`lex.awk:328`、`:446`)。だから旧版で `judge` の直後に読んだ `EXEC_SEEN` は、そのコマンド自身の `cmd_pos` の結果だった。
  - `cmd_pos` が `exec` まで進むのは、1 語目が予約語・`!`・`{`・`}`・`function`・代入・包み(env など 9 つ)のどれかか、1 語目が `exec` のときだけ。どれも DATACMD でも `git` でもない。1 語目の値が DATACMD の名前なら、斜線がなく `=` で始まらないので `cname` はその名前を返し、`cmd_pos` はそこで止まる。予約語は引用符を含まないので、`WR[ctx, 1]` が予約語なら `WV[ctx, 1]` も同じ語で、`data_first_ok` は 0 を返す。
- probe(`scratchpad/sr-gdc/`、macOS の /usr/bin/awk): plan の Edge cases(`if`・`for`・`while`・`until`・`select`・`{ … }`・`function`・`case`、`exec >log`・`exec 3>f`・`! exec`・`env exec`・`X=1 exec`・`=exec`・`nohup exec`・`command exec`、パイプの中の `exec`、置換の中の `exec`、`"if"`・`i\f`・`then`・`fi`・`}` で始まる形)と、`read_body` の境界(偶数・奇数のバックスラッシュ、続けて 2 回つなぐ行、本文の最後の行に改行がない `le > N`、`<<-`、引用した区切り)の 45 形。base と HEAD の判定(awk が出す規則の名前)は 45 形とも同じだった。gawk と mawk はこの機械にない。
- 単一の規則を外した写しで役割を分けた: HEAD から許可リストの行を消すと 45 形のうち 30 形が none になる。base から同じ行を消した写しと比べると、そのうち 26 形は base では消した 2 つの規則が deny にしていた形で、HEAD ではすべて許可リストが deny にしている。
- `trailing_backslashes(a, e)`: 中身は旧版の 2 つのループと同じ(`q = e - 1`、`while (q >= a && at(q) == BS)`)。下限は 1 つ目が `ls`、2 つ目が `p2` で、旧版と同じ。2 つ目の呼び出しは旧版と同じく `le <= N` を調べる前に数える。`read_body` から局所変数 `q` を外したが、`read_body` の中にほかの `q` はないので、大域変数は増えていない。
- S2(b312dc80): 3 つの `.awk` からコメントの行を除くと、a0094fe5 と b312dc80 で `cmp` が一致。語の境界を付けた `grep -nE '^[[:space:]]*#.*(^|[^A-Za-z0-9_])(SQ|DQ)([^A-Za-z0-9_]|$)'` は 0 件。戻した引用符(`--message''=${...}`、`$arr['$(cmd)']`、`("$arr["'$(cmd)'"]")`、`$'\x45'` など)は元の `SQ`・`DQ` の並びと合う。
- 写し: guard の 4 ファイルは root と `templates/base/` で `cmp` が一致。行数は `.sh` 233、`lex` 621、`commands` 335、`rules` 467。
- 消した名前: `git grep 'EXEC_SEEN\|resw_list\|\bRESW\b'` が当たるのは plan と tech-debt の経緯の文だけ。
- 不変条件の検査(`tests/test-pre-bash-guard.sh:1489-1541`): `set -u` の下でも、空の配列を展開するのは両方とも空でない `else` の枝だけ。`nm == "` は 7 文字なので `substr(s, RSTART + 7, RLENGTH - 8)` は名前だけを取る。`cmd_pos` の中に行頭の `}` はないので、awk の読み取りは関数の終わりで止まる。
- `verify.local.sh` の段: `run` の `if "$@"` の中で呼ぶので、`set -e` で途中終了しない。`.awk` が欠けると awk が非 0 で終わり、そのメッセージを出して FAIL になる。`pre_bash_guard.sh` のない木(`tests/test-archive-plan.sh` の fixture)だけを飛ばす。
- tech-debt の 3 行は、区切りでない `|` がどれも 6 個(5 列)で、`~~` は対になっている。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | (1。c7274f3b で修正済み)「許可リストは、複合コマンドのデータ区間を落とす唯一の規則」という文が、`case` には当てはまらない。`case` の節の `)` は、`lex_cmds` の `)` の規則(深さ 0 の `)` で `NODATA`)にも掛かる。`in_data` のコメントも、`(`・`)` の規則が掛かる場面を「a subshell, or the () of a function definition」とだけ書き、`case` の節の `)` を挙げていない。判定には関係しない | `rules.awk:405-410`(「the allowlist in end_cmd is the only rule that drops the data regions of a compound command」)、`tests/test-pre-bash-guard.sh:1489-1491`(例に `case` を挙げる)と `:89-95`、`commands.awk:240-241`。`lex.awk:151`。probe: HEAD から許可リストの行を消した写しで、`case x in x) echo '…'` は deny のまま。節のない `case x in esac; echo '…'` は none になる | 例を `if, for, while, until, select, {` にし、「a case clause's ) also trips the ( rule of lex_cmds」を添える(または「only」を外す)。`in_data` の括弧に「a case pattern's )」を足す |
| LOW | maintainability | (2。c7274f3b で (1) と (3) を修正。(2) は未修正で、直さなくてよい)不変条件の検査が読むのは、`split("…", datacmd_list, " ")` の 1 行と `cmd_pos` の中の `nm == "…"` だけ。(1) DATACMD に別の書き方(`DATACMD["exec"] = 1` の行など)で名前を足しても検査は通る。(2) `cmd_pos` の包みの比較を一部だけ別の関数に移すと、読める名前が減っても 1 つ以上残れば通る。(3) `rules.awk` が読めないとき、`grep -c` は何も出さず、メッセージは「is found  times」(数が空)になる。FAIL にはなる | `tests/test-pre-bash-guard.sh:1502-1523`。probe: `guard_awk` と同じ 3 つの `-f` の後ろに `BEGIN { for (k in DATACMD) print k }` だけのファイルを足して空の入力で回すと、20 語(`datacmd_list` の全部)を出して exit 0(BEGIN は書いた順に動き、END は空の入力では何も出さない) | DATACMD は実行時の配列から読む(すぐ上の F 節の `guard_awk` を使える)。包みの名前には `exec` と `env` が入っていることも確かめる。数が空のときは「cannot read」と出す |
| LOW | maintainability | (3。未修正。coordinator の判断で /sync-docs に回した)tech-debt の閉じ方が列と行でそろっていない。(1) guard の行の Why deferred の前半に「the comment items of (e) were closed in refactor/guard-awk-split except the `SQ` spelling, and its code items are still open」が現在形で残る。閉じたことは同じ列の後半に 1 文足しただけ。(2) Test gaps の行は「the equivalent mutants are now J02, L03, N02, N03, N04, and AL02 (… so N03 and N04 no longer exist)」で、1 つの文の中で一覧と括弧が食い違う。(3) `.awk` の行は Impact と Trigger を取り消し線にしたが、Debt item の列は取り消し線も `(RESOLVED …)` もなく、「A syntax error in one of the three files is caught only by the parse check in section F」が現在形で残る。前の PR(guard-awk-split)の self-review LOW 7 と同じ形 | `docs/tech-debt/README.md:160`(Why deferred)、`:163`、`:189`。閉じた行の書き方は `:17`、`:19` など(`~~…~~ (RESOLVED <date> in <branch>)`) | (1) 括弧の終わりを「…; refactor/guard-debt-cleanup closed the `SQ` spelling and the code items」にする。(2) 「are now J02, L03, N02, and AL02 (N03 and N04 went with the two rules in d7dac506)」にする。(3) Debt item の列を取り消し線にし、`(RESOLVED 2026-10-09 in refactor/guard-debt-cleanup)` を付ける。(4) c7274f3b で検査が DATACMD を実行時に読むようになったので、`:160` の Debt (e) の「checks that `datacmd_list` holds no reserved word」も DATACMD の言い方に直す |
| LOW | readability | (4。c7274f3b で修正済み)`verify.local.sh` の先頭のコメントにある static の一覧に、足した awk の構文の段がない | `scripts/verify.local.sh:7-8`(「static: shellcheck, sh -n, jq validity, template sync, tech-debt plan references」) | 「sh -n, the awk parse of the guard,」を足す |

CRITICAL、HIGH、MEDIUM はない。表の Evidence の行番号は、最初に review した c845d19a のもの。各行の先頭の括弧は、下の「c7274f3b の再 review」の結果。

細かい点(直さなくてよい): 不変条件の検査に書いた予約語の一覧は bash の語に `coproc` を足したもので、zsh の `repeat`・`foreach`・`end` は入っていない。DATACMD にこれらが入ることは考えにくいので、指摘にしていない。`rules.awk` 先頭のコメントの 3 行目は「, DATACMD and RESW」を消した分だけ短く、詰め直していない。

### 直す範囲の線引きの外の指摘

なし。消した規則は、上の Evidence のとおり許可リストが同じ場面で必ず `NODATA` にしていた。base と HEAD を 45 形で比べても違いはなかった。秘密の値、デバッグ用の出力、握りつぶしたエラーは diff にない。`check_guard_awk` は awk の stderr を出力に混ぜて「空であること」を求めるので、awk の警告も FAIL になる。すぐ上の F 節の構文の検査と同じ厳しさで、書いてある意図(print nothing)とも合う。

## c7274f3b の再 review(S5)

対象は c7274f3b(LOW 1、2、4 の修正)。HEAD の 1bbca910 は plan の Progress に 2 行足しただけ(`git diff --stat c7274f3b 1bbca910`)。

確かめたこと:

- guard のコードの行: `git diff -U0 2583435f c7274f3b -- .claude/hooks templates/base/.claude/hooks` の追加・削除の行のうち、`^[-+][[:space:]]*#` でないものは 0 行。3 つの `.awk` からコメントの行を除くと、2583435f と c7274f3b で `cmp` が一致。guard の 4 ファイルは root と `templates/base/` で一致。
- コメントと `lex_cmds` の突き合わせ(`lex.awk:146-151`): `(` は深さを 1 つ上げて `NODATA` を立てる。`)` が `NODATA` を立てるのは、深さ 0 で closer が `")"` でないときだけ。だから subshell が掛かるのは開きの `(` の規則だけで、`case` の節の `x)` は `)` の規則に掛かる。probe(HEAD から許可リストの行を消した写し): `case x in x) …`、`case x in (x) …`、`(echo …)`、`f() { …; }` は deny のまま。`{ …; }`、`if true; then …; fi`、`function f { …; }` は none になる。新しいコメントの「a brace group, and an if, for, while, until or select with no ( or ) at the top level, meet only the allowlist」と合う。
- 実行時の読み取り: テストと同じ `BEGIN { for (k in DATACMD) print k; exit }` のファイルを 3 つの `-f` の後ろに付けて回した。c7274f3b の写しでは 20 語で rc 0。`rules.awk` の BEGIN に `DATACMD["exec"] = 1` の行を足した写しでは 21 語で、`exec` が入る(c845d19a の `split` の行を読む方法では見えなかった)。`rules.awk` に構文エラーを入れた写しでは rc 2 で、stderr に awk の「syntax error at source line …」が出る。テストはこの stderr の先頭 300 バイトを FAIL のメッセージに入れる。`exit` のあとも guard の END は動くが、入力が空なら何も出さない(F 節の input [] の行が確かめている)。
- `guard_awk` と `$workdir` はどちらも、F 節の前のほうで条件なしに定義されている。名前の読み取りは 1 行 1 語の `while read` になり、空の行は飛ばす。

LOW ごとの結果:

| LOW | 結果 | 根拠 |
| --- | --- | --- |
| 1 「唯一の規則」 | 修正済み | `rules.awk:405-414` と `tests/test-pre-bash-guard.sh` の 2 か所は「the allowlist … is what drops the data regions」になり、`case` の節は `)` の規則にも、subshell は `(` の規則だけに掛かると書き分けた。`in_data` の括弧は「the ( and ) rules of lex_cmds: a subshell, the ) after the pattern of a case clause, or the () of a function definition」。上の probe と `lex_cmds` の分岐の両方と合う |
| 2 不変条件の検査 | 一部修正 | (1) DATACMD は実行時の配列から読み、別の書き方で足した名前も読む。(3) awk が非 0 で終わると、終了コードと stderr を出して FAIL になる。(2) 包みの名前は今も `cmd_pos` の `nm == "…"` を読むだけで、一部を別の関数に移すと黙って減る。テストのコメントは「they are code, not data」と理由を書いている。LOW のまま残すが、直さなくてよい |
| 4 `verify.local.sh` の一覧 | 修正済み | 先頭のコメントが「static: shellcheck, sh -n, the awk parse of the guard, jq validity, template sync, tech-debt plan references」になった |

新しい指摘: なし。細かい点(直さなくてよい)が 1 つある。`rules.awk` とテストの「a case clause meets the ) rule as well, by the ) after its pattern」は、`(x)` と書く POSIX の形の節には当てはまらない。この形では閉じの `)` が深さ 1 で読まれ、掛かるのは開きの `(` の規則のほう。どちらにしても `case` は許可リストのほかにもう 1 つの規則に掛かるので、判定と結論は変わらない。

最初の review の LOW 2 の Evidence にあった「23 語」は誤りで、`datacmd_list` は 20 語。表の該当の欄を直した。

insight のイベントは足していない。skill が求めるのは review の 1 回の実行につき 1 行で、この再 review は同じ cycle 1 の report に追記するものだから。

## Positive notes

- 規則を消すだけで終わらせず、規則が守っていた前提(DATACMD に予約語も包みも入れない)を、テストの検査と DATACMD の一覧の上のコメントに書いた。`exec >run.sh; echo 'sudo ls'` の行のコメントは、既存の 2 行がなぜその前提を確かめられないか(後ろの `sh run.sh` の 1 語目にも許可リストが働く)を書いており、行を足した理由が読める。
- `trailing_backslashes` は中身を変えず、範囲の始まりと終わりだけを引数にした。2 つの呼び出しを旧版のループと 1 行ずつ突き合わせられる。
- S2 をコメントの行だけの変更に分けたので、コメントを除いた `cmp` 1 回で「コードは変わっていない」を確かめられた。
- `verify.local.sh` の段は guard と同じ順・同じ `LC_ALL=C` で 3 つを 1 つの awk に読ませるので、ファイルをまたぐ関数の呼び出しの欠けも捕まえる。S1b の飛ばし方は fixture だけを外し、guard の `.sh` があって `.awk` が欠ける木は FAIL のまま残す。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しく先送りにするものはない。LOW 3 は登録簿の書き方の直しで、この PR の中(`/sync-docs` など)で直せる。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM なし。LOW 1 と 4 は c7274f3b で修正済み、LOW 2 は (1) と (3) を修正済みで、残る (2) は直さなくてよい。c7274f3b で guard のコードの行は変わっていない。LOW 3 は tech-debt の書き方で、`/sync-docs` に回した)
- Follow-ups:
  - LOW 3 は `/sync-docs` 向け: `docs/tech-debt/README.md:160` の Why deferred の前半と、Debt (e) の「checks that `datacmd_list` holds no reserved word」(今は DATACMD を実行時に読む)、`:163` の同値な mutant の一覧、`:189` の Debt item の列。
  - LOW 2 の (2) は、包みの名前に `exec` と `env` が入っていることを確かめる 1 行で閉じられる。急がない。
  - probe の注意: 見張りの語を含む形は python で語を分けて組み立て、Write で書いたスクリプトから 1 ファイル 1 形で awk に直接渡した(`scratchpad/sr-gdc/{gen.py,run.sh}`)。awk に直接渡すと、出力が deny かどうかではなく規則の名前で比べられる。
