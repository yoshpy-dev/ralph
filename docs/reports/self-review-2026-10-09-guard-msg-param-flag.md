# Self-review report: guard-msg-param-flag

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-msg-param-flag.md
- Branch: fix/guard-msg-param-flag(base c3a9242e、HEAD b4646f85)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1
- Scope: diff の品質だけを見た(安全性、コメントと記録の正確さ、読みやすさ)。対象は `git diff origin/main...HEAD`(5 ファイル、+500/-84)。guard の変更は 1e032dea と 43e73568、tech-debt の変更は b4646f85。テスト、静的解析、仕様との照合は行っていない。probe には plan・テスト・`scratchpad/mp/` にある形と、見張りの語を文字として書いた無害なメッセージ 1 つ(`fp01`)だけを使った。zsh の語の境目は、置換を含まない `print` で確かめた

## Evidence reviewed

- guard の差分と、その前後の関数を読んだ: `xnote`(`:271-276`)、`lex_cmds`(`:281-326`)、`lex_word`(`:338-397`)、`lex_dq`(`:404-439`)、`lex_dollar`(`:485-527`)、`lex_brace`(`:531-547`)、`lex_bq`、`lex_redir` の `<(`・`>(`(`:605-613`)、`read_body`・`heredoc_done`・`lex_hd`(`:747-756`)、`add_word`(`:779-788`)、`stage_note`(`:1028-1074`)、`commit_rules`・`tag_rules`・`msg_word`・`msg_attached`・`msg_check`(`:1373-1400`)、`safe_heredoc_msg`、BEGIN の新しい定数。行番号は HEAD の `.claude/hooks/pre_bash_guard.sh`(1593 行。base は 1487 行)のもの。
- root と template の写し: `cmp .claude/hooks/pre_bash_guard.sh templates/base/.claude/hooks/pre_bash_guard.sh` は差分なし。
- awk のプログラム(`:174-1550`)の中の単一引用符: awk の `index($0, "\047")` で各行を調べ、0 行だった。
- probe(`scratchpad/zg/run.sh`、`W=` にこの worktree を渡した): `mp/l01`〜`l17` と `mp/impl/p01`〜`p32`・`h1`。AC1 の形はすべて新版 none/none(jq あり/なし)。AC2 と AC2b の形はすべて新版 deny/deny。
- テストの全行の比較(`scratchpad/mp/sr1/`): `tests/test-pre-bash-guard.sh` のコマンドの配列 12 個(`intentional_fixes` を含む)の 618 行を 1 行 1 ファイルに書き出し、HEAD、base(c3a9242e の写し。`lib_json.sh` を同じディレクトリに置いた)、旧版(`tests/fixtures/guard-1c4cea5a/`)に、jq あり・なしで渡した。
  - 配列の期待値と HEAD の判定が違う行: 0
  - HEAD の jq あり・なしが違う行: 0
  - base から判定が変わった行: 23。none から deny が 17 行(B 節 `guard_deny_only_forms` の group 11 の 13 行と、D 節 `edge_sentinel_deny` に足した 4 行)、deny から none が 6 行(D 節 `edge_none` に足した 4 つのメッセージと、`rg 'foo$' 'sudo ' .`・`rg "foo$" 'sudo ' .`)。none になった 6 行はどれも旧版 deny。見張りの語のそばの `$` は、単一引用符の中、`\$`、`$'…'` の中、閉じる `"` の直前のどれかで、bash も zsh も展開しない。既存の行の判定は 1 行も変わっていない。C 節 `ac3` の 29 行と `intentional_fixes` の行は none のまま。
- `fp01`(`git commit -m "costs 5$, never sudo ls"`): HEAD deny/deny、base none/none、旧版 deny。
- zsh 5.9(`zsh -f scratchpad/mp/sr1/dqsub.zsh`、`arr=(a b c)`): `print -r -- "A1" "$arr["2"]"` は `A1 b` を出す。二重引用符の中で開いた添字は、閉じる `"` の後ろでも同じ語の中なら続く。`"$arr[" 2 "]"` は `invalid subscript` になり、空白はまたがない。`man zshparam` の Subscript Parsing も、添字の中の `"` は通常の引用の規則どおり対になると書く。添字は字句解析の語の終わりで終わるので、`lex_word` が語の終わりまでを外すやり方は zsh と合う。

### 依頼された確認への答え

1. AC1・AC2・AC2b: 上の probe のとおり。AC1 はどのモードでも none、AC2 と AC2b はどのモードでも deny。
2. 正しさ
   - 印を立てる順序: `$(` の分岐は `lex_cmds` から戻ったあと(`:490-495`)、`${` の分岐は `lex_brace` から戻ったあと(`:500-505`)に、`LD_SUBST`・`LD_EXP`・`LD_ANSI`・`LD_SUB` の 4 つを書く。`lex_brace` の中で呼んだ `lex_dollar` や `lex_dq` が立てた印は、ここで上書きされる。`$'…'` と裸の `$` の分岐は入れ子を呼ばない。そこで呼ぶ `skip()` と `at()` が動かすのは窓の変数だけ。
   - `LD_SUB` のリセット: `lex_dollar` の 4 つの分岐がすべて `LD_SUB` を書く。`lex_word` と `lex_dq` は局所変数 `subp` を毎回 0 から始め、`lex_dollar` を呼んだ直後にしか `LD_*` を読まない。`lex_brace` と `lex_hd` は `LD_SUB` を読まない。
   - `lex_dq` から `lex_word` への受け渡し: `lex_dq` は局所の `xp`・`subp` を最後に `DQ_EXP`・`DQ_SUB` へ写す(`:435-437`)。`lex_brace` を経た内側の `lex_dq` が大域の `DQ_*` を書いても、外側の `lex_dq` が最後に書き直す。`lex_word` は `lex_dq` が戻った直後に読む(`:359-361`)。
   - `add_word` の 2 か所: `lex_cmds`(`:320`)は `LW_EXP`・`LW_ANSI` を渡す。`lex_redir` の `<(…)`・`>(…)`(`:609`)は `0, 0` を渡す。この語は `WS` が 1 で、印を読む 2 か所(`msg_check` は `WS` を先に見る。rg の判定は `WS` も見る)の結果は変わらないので、食い違いはない。
   - 1 つの語に `$` が複数ある場合: `if (LD_SUB && !subp)` で最初の添字の `$` だけを残し、語の終わりで 1 回だけ `xnote` する(`:389`)。片は左から読むので、残るのはいちばん前の位置で、早すぎも遅すぎもしない。添字の中の `$(…)` の範囲が先に、添字の範囲があとに登録されるが、`stage_note` の挿入ソート(`:1056-1062`)と `xe <= cur` の読み飛ばし(`:1069`)で、内側の範囲は外側に含まれて消える。リダイレクトの先の語に添字がある場合も、外側のリダイレクトの範囲に含まれる。
   - `$a[` のあとの `;`: `lex_word` は引用符の外の空白と演算子で止まる(`:378`)ので、`;` の後ろは次のコマンドとして読まれる。`echo $a[ ; git push origin --force` は deny/deny。
   - awk の単一引用符と root/template: 上の Evidence のとおり。
3. PR #211 からの後退: `${(e)…}` のメッセージ、`--message''=${…}`、`-"m"${…}` は deny/deny のまま。推奨の HEREDOC の形(`h1` と、本文に `${HOME}` を書いた `edge_none` の行)は none のまま。`ac3` の 29 行は変わらない。
4. コードの品質、コメント、tech-debt: 下の Findings。
5. 線引きの外: 下の「直す範囲の線引きの外の指摘」。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | tech-debt の guard の限界の行(`:160`)は、rg の変数の語の穴を Debt の列でだけ解消済みにした。ほかの列には未解決のまま残っている。(1) Impact の (b) は、guard を「a variable word that rg reads as `--pre`」で迂回できると書く。(2) Why deferred は「the items of (e) and the rg variable word of (b) stay open」と書く。(3) Trigger の (b) は「the next change to `stage_note`: … add `rg $x sudo pat .` to `edge_sentinel_deny` in the same commit」と指示しているが、この PR はその行を B 節 `guard_deny_only_forms` に置いた。次に読む人がこの指示に従うと、同じ行を 2 か所に置くことになる。(4) Impact の (a) は rg の止めすぎを「`$'…'` か `$"…"` の語」とだけ書く。今は展開する `$` や置換のある語も含み、`rg "foo$"` は外れた。新しい行(`:167`)の (b) に書いてあるので、そこを指せば足りる。(5) Trigger の (d)「The next change that grows the guard」と (e)「The next change to the guard」は、この PR でまた発火した。guard-zsh-data-gaps のときは持ち越しを書いたが、今回は書かれていない | `docs/tech-debt/README.md:160` の Impact、Why deferred、Trigger の各列。base の guard は 1487 行、HEAD は 1593 行(`wc -l`) | この cycle の `/sync-docs` で直す(docs だけ)。(1)〜(3) は rg の変数の語を解消済みにし、固定先を「B 節 group 11」にする。(4) は `:167` の (b) を参照させる。(5) には「fired again at fix/guard-msg-param-flag (+106 lines) and carried over: the plan's Non-goals leave the split to a separate PR, and its Design decisions limit the fix to three points」の趣旨を足す |
| LOW | maintainability | 新しい行(`:167`)と解消済みにした行(`:166`)の書き方が、コードと少しずれている。(1) `:167` の冒頭は「fix/guard-msg-param-flag (commit 1e032dea) adds」と書くが、(c) に挙げた特別なパラメータの添字(`$@[` など)は 43e73568 で入った。(2) `:167` の (a) は、止めすぎの範囲を「any `$` that expands (a variable, `$1`, `$@`, as well as `${…}`)」と書く。しかし `LD_EXP`(`:519`)は、後ろが空白・終わり・閉じる `"` でない `$` すべてに立つ。`lex_dollar` のコメント(`:463-465`)も「`$;` のように文字のままの `$` にも立つ」と認めている。`fp01` の `git commit -m "costs 5$, never sudo ls"` では、bash にとって `$,` は文字のままだが(zsh は確かめていない)、HEAD は deny、base は none になる。旧版も deny なので、guard-deny-only の AC7 は保たれている。(3) `:166` の Trigger の末尾「no report has come in」に句点がない | `docs/tech-debt/README.md:166`、`:167`。`.claude/hooks/pre_bash_guard.sh:463-465`、`:519`。probe `fp01`(HEAD deny/deny、base none/none、旧版 deny) | この cycle の `/sync-docs` で直す。(1) は「commits 1e032dea and 43e73568」にする。(2) は「a `$` not followed by a blank, the end, or a closing double quote: a variable, `$1`, `$@`, `${…}`, and some `$` that bash leaves as text, such as `5$,`」の趣旨に広げる。(3) は句点を足す |
| LOW | readability | コメント 3 か所が仕組みを正確に書いていない。(1) `lex_word` のコメントは、添字の範囲を「xnote で登録するので決してデータにならない」と書く。しかし `xnote` が外すのは引数のデータ区間 (a) だけで、メッセージの語は `msg_check` が語全体を `add_data` する。添字のあるメッセージがデータにならない理由は別にある。`LD_SUB` が立つのは `$` の次が名前・修飾の記号・特別なパラメータ・`[` のときで、そのとき `LD_EXP` も必ず 1 になり(`:519`、`:524`)、`msg_check` がそれを見る(`:1398`)。この前提はどこにも書かれていない(B 節の `git commit -m $arr['$(sudo ls)']` の行が固定している)。`lex_dollar` のコメント(`:477-478`)は「argument data region」と正しく書いている。(2) `stage_note` のソートの前のコメントは、順序が崩れる例としてリダイレクトと `${...}` を挙げる。添字の範囲も、中の置換のあとに登録される(`$arr[$(date)]`、`:389`)。(3) テストの group 10 のコメントは「msg_check reads the whole source word」と書く。今の `msg_check` が読むのは語の印 `WEXP` で、ソースの文字は見ない | `.claude/hooks/pre_bash_guard.sh:333-337`、`:477-478`、`:519-524`、`:1053-1055`、`:1398`。`tests/test-pre-bash-guard.sh:668` | (1) は「so it is never part of the argument data region (a); a message word with it is no data either, since every $ that starts a subscript also sets LD_EXP」の趣旨にする。(2) は「and a subscript after the substitutions inside it」を足す。(3) は「msg_check reads the per-word mark」にする。どれもコメントだけの変更。guard のファイルを変えると pipeline の再実行になるので、別の理由で guard を直すときにまとめるか、`:160` の (e) に足して持ち越す |

CRITICAL・HIGH・MEDIUM はない。

細かい点(直さなくてよい): ヘッダーの `:141-142` と語の配列の説明の `:246-247` は、差分を小さくするために詰め直しておらず、短い行が残っている。

### 直す範囲の線引きの外の指摘

なし。展開する `$` の印と添字の範囲が、データ区間に入りうる道をコードで 1 つずつたどった。引数は `lex_word`・`lex_dq`・`lex_brace`、メッセージは `msg_word`・`msg_attached`・`msg_check`、rg は `stage_note`。`$name[…]` 以外の仕組みで評価される形は見つからなかった。

未確認の点が 1 つある。引用符のない区切りのヒアドキュメントの本文 (c) では、`lex_hd` は `LD_SUB` を見ない。本文の中では単一引用符がただの文字なので、`'$(…)'` の `$(` は `lex_hd` に見え、HSUB が立つ。バックスラッシュで `$` を文字にした形を zsh の添字が評価し直すかどうかは、新しい形を作らない取り決めに従って調べていない。plan の Objective 3 は引用符の外の形を対象にしているので、指摘にはしていない。

## Positive notes

- 印を `lex_dollar` で立てる設計で、PR #211 の止めすぎ(単一引用符などの中の `${`)と、PR #210 の self-review の C4R-1(`rg 'foo$'`)を、引用符を別にたどり直さずに直した。`lex_dollar` は shell が読む `$` でしか呼ばれないので、印の意味がコメントどおりに保たれている。
- `LD_EXP` を広めに立てるので、外れるのは止める側だけ。`fp01` のような止めすぎも旧版が止める形で、AC7 を壊さない。
- 添字の範囲を `]` まで読まず語の終わりで切るので、`;` の後ろのコマンドを飲み込まない。zsh の語の境目とも合う(`dqsub.zsh`)。
- awk の組み込み関数 `exp` を避けて、局所変数を `xp` にしている。
- テストのコメントが、PR #211 と旧版の判定を行ごとに書いており、3 方向の比較と合う。`edge_none` の「PR #211 denied the four messages and both rg patterns」と「the old guard denies all eight rows」、`edge_sentinel_deny` の「PR #211 let all four through」、`edge_deny` の「The old guard lets all three through」。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

新しく先送りにするものはない。上の LOW 3 件は、この PR の中で直せる記述の修正。

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 3 件は記録とコメントの書き方)
- Follow-ups:
  - `/sync-docs` 向け: LOW の 1 件目と 2 件目の tech-debt の修正(docs だけ)。
  - LOW の 3 件目はコメントだけだが guard のファイルを変える。cross-review で別の修正が出たときにまとめて直すか、`docs/tech-debt/README.md:160` の (e) に足して持ち越す。
  - probe の注意: guard の写しを scratch に置くときは、`lib_json.sh` を同じディレクトリに置く。置かないと、写しはどの行にも何も返さず、全行が none に見える(この review の 1 回目の比較で起きた)。
