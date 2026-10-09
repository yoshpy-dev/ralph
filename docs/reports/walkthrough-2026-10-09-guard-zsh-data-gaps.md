# Walkthrough: guard-zsh-data-gaps

- Date: 2026-10-09
- Plan: docs/plans/archive/2026-10-09-guard-zsh-data-gaps.md(この PR の最後のコミットで archive に移す)
- Branch: fix/guard-zsh-data-gaps(base main 0931f791)
- Diff(この walkthrough を除く): 10 files、+620 / -42。報告・insight・plan・`templates/` の写しを除くと 3 files、+95 / -23

## 何を変えたか

PR #210 で残した `pre_bash_guard.sh` の 3 つの穴を塞いだ。

- zsh の `${(e)…}` は値を評価し直し、字句解析が引用とみなした中身(`'$(cmd)'` や `\$(cmd)`)の置換を実行する。`${…}` の範囲をデータ区間から外し、`${` を含むコミット・タグのメッセージの語と、`${` のある区切りに引用符のないヒアドキュメントの本文もデータにしない
- zsh の `stat -A NAME`(zsh/stat モジュール)は NAME の添字を評価するので、`stat` を読むだけのコマンドの一覧から外した
- `tr` を引数を実行しないコマンドの一覧に足し、`tr "sudo" "abcd"` の誤検知をなくした

データ区間でなくなった文字は、旧版の 4 規則(見張り)が旧版どおりに決める。

## 読む順

1. plan の Objective と Design decisions(範囲の線引き)
2. `.claude/hooks/pre_bash_guard.sh` の `lex_dollar` の `${` の分岐(`xnote`)と、その上のコメント
3. 同じファイルの `msg_check`(推奨の HEREDOC の形の判定のあとに `${` の検査)と `lex_hd`(`${` で HSUB)
4. BEGIN の DATACMD と `noexec_list`
5. `tests/test-pre-bash-guard.sh` の B 節の 10 の群(AC1 の 11 形。AC7 の比較の例に入る)と、D 節の `edge_none` の AC2 の形、`edge_deny` の 1 行
6. `docs/tech-debt/README.md` の解消済みにした行と、新しい止めすぎの行

## 計画からの逸脱

なし。plan の承認の前に、consult と Codex の plan advisory の指摘(バックスラッシュで `$(` を隠す形、引用符を挟んでつなげたメッセージの形)を範囲に入れた。

## 残る穴

- 止めすぎ: 単一引用符や `\${` で `${` を文字として書き、見張りの語も含むメッセージ(`git commit -m 'mention ${HOME}; never sudo ls'`)、`${…}` のある区切りに引用符のないヒアドキュメントの本文の見張りの語、`${…}` の中の見張りの語、`stat` で始まるコマンドのある呼び出し。どれも旧版も deny。語ごとの印で見分ける直し方は tech-debt に書いた
- `${…}` 以外の仕組みで zsh や bash が評価する形は、この PR の範囲の外
