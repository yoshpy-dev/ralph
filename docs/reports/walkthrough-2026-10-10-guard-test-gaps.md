# Walkthrough: guard-test-gaps

- Date: 2026-10-10
- Plan: docs/plans/archive/2026-10-10-guard-test-gaps.md(この PR の最後のコミットで archive に移す)
- Branch: test/guard-test-gaps(base main 49ac046c)
- Diff(この walkthrough を除く): 15 files、+666 / -83。報告・insight・plan・`templates/` の写しを除くと 5 files、+98 / -68

## 何を変えたか

PR #215 で tech-debt に残した guard のテストの穴を 2 つ塞いだ。判定は 1 つも変えていない。

- (h) DATACMD の不変条件の検査は、`cmd_pos` が飛ばす包みの名前をソースの `nm == "…"` から読んでいたので、包みの分岐を別の関数に移すと検査から黙って外れた。包みの 9 つの名前を `pre_bash_guard_rules.awk` の BEGIN の一覧 `WRAPPER` にし、`cmd_pos` は一覧にない名前ならその位置を返すようにした。テストは実行時の DATACMD と `WRAPPER` を読む
- (h) の追加(Codex の plan advisory): 一覧と分岐がずれると、その包みは飛ばされず、後ろの `sh -c` の中を読まないので、そこに書いた `git commit -n` を見逃す。今のテストではこれに気づけなかった。包みごとに `env sh -c 'git commit -n -m x'` のような行を D 節の `edge_deny` に 9 行足した
- (i) `trailing_backslashes` の数え始めの下限を変える mutation を殺す行がなかった。ヒアドキュメントの本文の、バックスラッシュだけの行(2 個なら継がず、3 個なら次の行と継ぐ)のあとに `git commit -n -m x` を置く 2 行を足した

## 読む順

1. plan の Design decisions(一覧にする理由、包みごとの行を足す理由)
2. `pre_bash_guard_rules.awk` の BEGIN の `WRAPPER` と、`pre_bash_guard_commands.awk` の `cmd_pos` に足した 1 行
3. `tests/test-pre-bash-guard.sh` の F 節の最後の不変条件の検査(DATACMD と `WRAPPER` を 1 つの dump で読む)
4. 同じファイルの D 節の `edge_deny` の包みの 9 行と、`edge_deny`・`edge_none` のヒアドキュメントの 2 行
5. `docs/tech-debt/README.md` のテストの穴の行の (h)(i) と、guard の限界の行の閉じの文

## 計画からの逸脱

- `cmd_pos` が先頭で飛ばす予約語を一覧にする案(`CMDSKIP`)は、plan の承認の前に外した。テストの予約語の一覧がその 10 語をすべて含み、検査の網が広がらないため(consult の指摘)
- self-review の LOW 1・2(BEGIN の一覧を説明するコメントに `WRAPPER` がない、ヘッダーの包みの散文が一覧を指していない)を、verify の前に直して reviewer に見直してもらった

## 確かめたこと

- テストは 2,090 件で失敗 0(macOS の awk、mawk、gawk)
- `WRAPPER` の 9 つの要素を 1 つずつ消す写しと、`cmd_pos` の 9 つの分岐を 1 つずつ消す写しの 18 通り、下限を `q > a` に変える写しの 1 通りが、3 つの awk すべてでテストを赤にした
- base(49ac046c)とこの PR の guard に 4,347 件の入力(包みを重ねる・代入を前に置く・引用符やパスを付ける境界の入力 153 件を含む)を jq あり・なしで渡し、3 つの awk すべてで判定の違いは 0 件
- `timeout` の分岐を `cmd_pos` の外の関数に移した写しでも、検査は `timeout` を読む(前の読み方では抜けた)

## 残る穴

- 10 個目の包みを `WRAPPER` に足して `edge_deny` の行を足さないずれは、テストでは捕まらない。新しい包みには、一覧の名前、`cmd_pos` の分岐、`edge_deny` の行の 3 つが要る(tech-debt に書いた)
