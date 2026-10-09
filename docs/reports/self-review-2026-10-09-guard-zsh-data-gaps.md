# Self-review report: guard-zsh-data-gaps

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-zsh-data-gaps.md
- Branch: fix/guard-zsh-data-gaps(base 0931f791、HEAD d1302b7a)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1
- Scope: diff の品質だけを見た(安全性、コメントと記録の正確さ、読みやすさ)。対象は `git diff origin/main...HEAD`(5 ファイル、+255/-38)。コードの変更は 139d6627、tech-debt の変更は d1302b7a。テスト、静的解析、仕様との照合は行っていない。probe には plan・テスト・`scratchpad/zg/` にある形だけを使い、新しい形は作っていない

## Evidence reviewed

- 139d6627 の guard の差分と、その前後の関数を読んだ: `xnote`(`:260-265`)、`lex_word`、`lex_dq`、`lex_dollar`(`:415-437`)、`lex_brace`、`read_body`、`heredoc_done`(`:634-649`)、`lex_hd`(`:657-666`)、`end_cmd`、`stage_note`(`:932-974`)、`pipe_decide`、`in_data`、`msg_word`・`msg_attached`・`msg_check`(`:1273-1296`)、`safe_heredoc_msg`、END ブロック(`:1412-1439`)。行番号は HEAD の `.claude/hooks/pre_bash_guard.sh`(1483 行)のもの。
- root と template の写し: `cmp .claude/hooks/pre_bash_guard.sh templates/base/.claude/hooks/pre_bash_guard.sh` は差分なし。
- awk のプログラムの中の単一引用符: `:166-1439` の各行を awk の `index($0, sprintf("%c", 39))` で調べ、0 行だった。diff で足した行にも単一引用符はない。
- probe(`scratchpad/zg/run.sh`): `l01`〜`l16`、`m1`〜`m3`、`ml1`〜`ml3`、`h1`・`h2`、`w1`〜`w5`、`td-h`・`td01`〜`td06` を渡した。AC1 の形はすべて新版 deny/deny(jq あり/なし)、旧版 deny。AC2 の形はすべて新版 none/none。
- テストの全行の比較(`scratchpad/zg/sr1/`): `tests/test-pre-bash-guard.sh` のコマンドの配列 11 個の 575 行を 1 行 1 ファイルに書き出し(`dump.sh`)、HEAD、base(origin/main 0931f791)、旧版(`tests/fixtures/guard-1c4cea5a/`)に、jq あり・なしで渡した(`one.sh`、`summ.sh`)。
  - 配列の期待値と HEAD の判定が違う行: 0
  - HEAD の jq あり・なしが違う行: 0
  - base から判定が変わった行: 13。B 節 `guard_deny_only_forms` に足した 11 行(none から deny)と、D 節 `edge_none` の `tr` の 2 行(deny から none)だけで、既存の行は 1 行も変わっていない。C 節 `ac3` の 29 行は none のまま。
- tech-debt の新しい行(`docs/tech-debt/README.md:166`)の (a)〜(d) の例を HEAD・base・旧版に渡した: 4 つとも base none、HEAD deny、旧版 deny で、行の記述と合う。通ると書いてある例(`git commit -m 'use ${HOME}'`、`git commit -m "${msg}"`、推奨の HEREDOC の形、`echo "${HOME}" 'sudo ls'`)は HEAD で none。

### 依頼された確認への答え

1. `${…}` の `xnote` と中の `$(…)` の `xnote` の順序: 中の置換が先に登録され(`lex_brace` から呼ぶ `lex_dollar`)、外の `${…}` があとに登録される。`stage_note` の挿入ソート(`:956-962`)は開始位置の `>` で比べるので、外の範囲が中の範囲の前に来る。切り出しのループ(`:966-972`)は `xe <= cur` の範囲を飛ばし、`cur` は戻らない。そのため、中に含まれる範囲がデータ区間を余計に分けることはない。リダイレクトの先の `${f}` も同じ扱いになる。ヒアドキュメントの本文の `lex_hd` は `MAIN = 0` のあとのキューの処理(`:1427-1433`)で走り、`DCTX` が 0 なので `xnote` は `:261` で何もせずに返る。本文の範囲が XS に紛れ込むこともない。
2. `msg_check` の順序: 置換ありの語の検査(`:1290-1293`)が先で、`${` の検査(`:1294`)があと。どちらにも `safe_heredoc_msg` の例外が付いている。plan の Design decisions の順序と同じ。推奨の HEREDOC の形で本文に `${HOME}` と見張りの語を書いた例(`ml3`)は none で、データのまま。
3. `lex_hd` の HSUB と `git commit -F -` の deny: HSUB を読むのは `:1436`(データ区間を作るところ)だけ。`git commit -F -` の deny は `heredoc_done` の `:644` にあり、主の字句解析の中で `read_body` から呼ばれ、本文の部分文字列(`$(` とバッククォート)だけを見る。`lex_hd` はそのあとのキューの処理で走るので、HSUB はこの deny に影響しない。probe でも `h2`(本文 `use ${HOME} here`)は none、`ml2`(本文に `$(`)は deny で、どちらも base と同じ。
4. awk のプログラムの単一引用符: なし(上の Evidence を参照)。
5. root と template の写し: バイト単位で同じ。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `lex_dollar` のコメントは「`$(...)` と `${...}` 全体を `xnote` で登録するので、どちらもデータにならない」と書く。しかし `xnote` が効くのは引数のデータ区間 (a) だけ。メッセージの語は `:1295` で語全体を `add_data` し、範囲を切り出さない。ヒアドキュメントの本文も `:1436` で本文全体を足す。(b) と (c) で `${` をデータから外しているのは、`msg_check` の `:1294` と `lex_hd` の `:663`。語全体をデータにする区間を将来足す人がこの一文を信じると、同じ穴がまた開く。また `stage_note` のソートの前のコメント(`:954-955`)は、順序が崩れる例としてリダイレクトしか挙げていない。今は「`${...}` は中の置換のあとに登録される」場合もある | `.claude/hooks/pre_bash_guard.sh:408-409`、`:954-955`、`:1294-1295`、`:1436` | `:408-409` を「どちらも引数のデータ区間 (a) から外れる。(b) と (c) では `msg_check` と `lex_hd` が `${` を外す」の趣旨に直す。`:954-955` に `${...}` の場合を足す |
| LOW | readability | `edge_none` の前のコメントは、「`${...}` のまわりの文字はデータのまま」の例として、見張りの語のない `git commit -F -` の本文を挙げている。`lex_hd` の変更(`:663`)で、この本文には HSUB が立ち、データ区間ではなくなった(`:1436`)。この行が none になるのは、本文に見張りの語も `$(` もないからで、データだからではない。plan の AC2 も「本文に見張りの語がない」と書いている | `tests/test-pre-bash-guard.sh:1039-1041` | この例を「データのまま」の列挙から外し、「本文に `${...}` がある `git commit -F -` も deny にはならない(本文はデータ区間でなくなるが、見張りの語も `$(` もない)」のように分けて書く |
| LOW | maintainability | tech-debt の 2 行の書き方が、コードや例と少しずれている。(1) 新しい行の (c) は「同じ語の中の `${…}` の外の文字はデータのまま」と書くが、例の `echo "${HOME}" 'sudo ls'` は 2 つの語でできている。同じ語の中でも外れるのは範囲だけ、というのはコード(`stage_note` が範囲だけを切り出す)からは言えるが、例では示せていない。新しい形は作らないことにしたので、probe では確かめていない。(2) (a) と Why deferred は、止めすぎの原因を「単一引用符の中の `${`」とだけ書く。`:1294` は引用符を見ずに `index(WR, "${")` で調べるので、`\${` や `$'…'` の中の `${` のように、別の理由で文字のままの `${` でもデータ区間がなくなる(コードを読んだだけで、probe はしていない)。Trigger の直し方(`lex_dollar` が `${` を読んだときに立てる語ごとの印)はこれらも含むので、記述だけ広げれば足りる。(3) (d) は「`stat` で始まる呼び出し」と書くが、`end_cmd` の `:748` は最上位のどのコマンドについても先頭の語を調べる。`stat` が 2 つ目以降のコマンドの先頭にあっても、呼び出し全体のデータ区間がなくなる。plan の Risks も同じ書き方をしている。(4) 解消済みにした行の Trigger の「done as written, with `${…}` excluded as a whole span」は、書かれていた直し方(`${` を持つ語がある data command にはデータ区間を与えない)と違う方法で直したことを、「as written」と言いながら後半で認めている。範囲だけを外したので、`echo "${HOME}" 'sudo ls'` が none のまま残った | `docs/tech-debt/README.md:165`、`:166`; `.claude/hooks/pre_bash_guard.sh:748`、`:1294`; probe `td01`〜`td06` | (1) は「同じ語の中」を「同じコマンドの中」にする。(2) は「引用符やバックスラッシュで文字のままの `${`」に広げる。(3) は「最上位のどこかに `stat` で始まるコマンドがある呼び出し」にする。(4) は「done, by excluding the `${…}` span instead of the command's whole data region」のように書く |

CRITICAL・HIGH・MEDIUM はない。

### 直す範囲の線引きの外の指摘

なし。`${` がデータ区間に入りうる道をコードで 1 つずつたどった: 引数は `lex_word`(`:344-348`)、`lex_dq`(`:389-391`)、`lex_brace`(`:450`)、メッセージは `msg_word`・`msg_attached`・`msg_check`、本文は `lex_hd`(`:663`)、コメント (d) は実行されない。どの道でも `${…}` がデータに残るところは見つからなかった。別の仕組みで評価される形(ヘッダーの Not covered にある算術など)は、新しい形を作らない取り決めに従って調べていない。

## Positive notes

- 判定を変える変更は、`tr` を NOEXEC に足したことを除いて、すべてデータ区間を減らす向き(止める向き)だけ。`tr` の変更は `scan_words` が後ろの語を読まなくなるだけで、見張りの一致を無視するには、これまでどおり DATACMD の条件を満たす必要がある。この変更で、DATACMD の名前はすべて NOEXEC にも入った(入っていなかったのは `tr` だけ)。
- `lex_hd` の HSUB の検査は、`lex_dollar` が P を進める前に `at(P + 1)` を読む。`\${` は BS の分岐で 2 文字飛ばすので、コメントの「An escaped \${ is text」と動きが合う。
- `msg_check` が `raw` でなく `WR` を見るので、引用符で旗が分かれた形(`--message''=`、`-"m"`)のように、`msg_attached` が空の `raw` を渡す場合も閉じている(probe `w2`・`w3`・`w5` は deny)。
- テストのコメントが行ごとに旧版の判定を書いていて、probe の結果と合う(B 節の 11 行は旧版ですべて deny、D 節の新しい行で旧版が deny にするのは echo の行と推奨の HEREDOC の形の行だけ)。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しく先送りにするものはない。上の LOW 3 件は、この PR の中で直せる記述の修正。

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 3 件はコメントと記録の書き方)
- Follow-ups:
  - `/sync-docs` 向け: tech-debt の guard の限界の行(`docs/tech-debt/README.md:160`)は、DATACMD から外したコマンドとして `test` と `[` しか挙げていない(「`test` and `[` are not in it」「Three commands carry a condition of their own」)。この PR で `stat` も外れた。AC6 が求める「未解決の記述」ではないが、一覧が足りなくなっている。
  - LOW の 3 件目の (2) と (3) は、コードを読んで言えることで、probe では確かめていない。直すときに無害な形で確かめるなら、`/test` か次の cycle で行う。
