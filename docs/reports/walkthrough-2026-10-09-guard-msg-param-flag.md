# Walkthrough: guard-msg-param-flag

- Date: 2026-10-09
- Plan: docs/plans/archive/2026-10-09-guard-msg-param-flag.md(この PR の最後のコミットで archive に移す)
- Branch: fix/guard-msg-param-flag(base main c3a9242e)
- Diff(この walkthrough を除く): 11 files、+962 / -90。報告・insight・plan・`templates/` の写しを除くと 3 files、+230 / -49

## 何を変えたか

`pre_bash_guard.sh` の判定を 3 点直した。どれも、引用符の外か二重引用符の中で実際に展開される `$` を、語ごとの印で見分ける直し方でそろえた。

- 止めすぎ(PR #211 が増やしたもの): 単一引用符・`\${`・`$'…'` で `${` を文字として書き、見張りの語も含むコミットとタグのメッセージが deny になっていた。`msg_check` が生の語の `${` を見ていたため。今は `lex_dollar` が立てる「展開」の印(`WEXP`)を見る
- 穴(旧版は deny): `rg $x sudo pat .` のような変数の語を持つ rg を読むだけのコマンドとして扱っていた。変数の値が `--pre=…` ならプログラムを実行する。今は、展開する `$`、`$'…'`・`$"…"`、置換のどれかを持つ語があれば rg をデータ用のコマンドにしない。単一引用符の中の `'foo$'` は展開しないので、`rg 'foo$' 'sudo ' .` は none になった(旧版も base も deny)
- 穴(旧版は deny、plan を書く途中で見つけた): zsh は引用符の外の `$arr['$(cmd)']` の添字を評価し、単一引用符の中の置換を実行する。添字を始める `$` から語の終わりまでを、引数のデータ区間から外した。中は読み飛ばさないので、添字の中の `$(…)` は今までどおり置換として読み直される

データ区間でなくなった文字は、旧版の 4 規則(見張り)が旧版どおりに決める。

## 読む順

1. plan の Objective と Design decisions(印を `lex_dollar` で立てる理由、添字を語の終わりまで外して中を読み飛ばさない理由)
2. `.claude/hooks/pre_bash_guard.sh` の `lex_dollar`(`LD_EXP`、`LD_ANSI`、`LD_SUB`)と、その上のコメント
3. 同じファイルの `lex_word`・`lex_dq`・`add_word`(印を語ごとに持つ `WEXP`・`WANSI`、添字の `xnote`)
4. `msg_check`(推奨の HEREDOC の形の判定のあとに `WEXP` の検査)と、`stage_note` の rg の分岐
5. `tests/test-pre-bash-guard.sh` の B 節 group 11(旧版も deny にする 13 行。AC7 の比較の例に入る)、D 節の `edge_none`(AC1 の 9 行)、`edge_deny`(AC2b の 3 行)、`edge_sentinel_deny`(この PR で増える止めすぎの 4 行)
6. `docs/tech-debt/README.md` の guard の限界の行、PR #211 の止めすぎの行、この PR で増えた止めすぎの行

## 計画からの逸脱

- S1b: 特別なパラメータの添字(`$@[…]`、`$*[…]`、`$#x[…]`)を添字の判定に足した。Objective 3 の範囲で、Scope の「名前 0 文字以上」を広げたもの(止める側)
- コメントの直し(899fff16)は self-review の後、verify の前に入れた。guard のファイルを変えたので、pipeline の規則どおりなら self-review からやり直す対象になる。コメントだけでコードの行は変わっておらず、verify・test・Codex は直した後を見た。reviewer だけがこの 3 か所を見ていない
- Scope の「PR #211 が置いた形を `edge_none` に移す」に当たる行はなかった(base の行で判定が変わったものはない)

## 残る穴と止めすぎ

- 止めすぎ: 展開する `$` を持つメッセージで、見張りの語も含むもの(`git commit -m "costs 5$, never sudo ls"` など)、変数の語を持つ rg で見張りの語もあるもの、添字のあとの同じ語の見張りの語。どれも旧版も deny。tech-debt の新しい行に書いた
- テストの穴: zsh の修飾 `=`・`^`・`+` の添字(mutation M20 が残る)と、展開しない `$` の条件(M13 の系統)を固定する行がない。guard のファイルを分ける次の PR で足す
- guard のヘッダーの (b) が `$"…"` を展開しない `$` に挙げていない(振る舞いは変わらない)。同じく次の PR で直す
