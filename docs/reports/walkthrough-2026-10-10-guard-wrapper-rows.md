# Walkthrough: guard-wrapper-rows

- Date: 2026-10-10
- Plan: docs/plans/archive/2026-10-10-guard-wrapper-rows.md(この PR の最後のコミットで archive に移す)
- Branch: test/guard-wrapper-rows(base main 382c18c8)
- Diff(この walkthrough を除く): 9 files、+577 / -15(大半は報告と plan)。コードの変更は `tests/test-pre-bash-guard.sh` だけで、guard のファイルは変わっていない

## 何を変えたか

PR #217 で残したものを直した。guard の包みには、`WRAPPER` の名前と `cmd_pos` の分岐が要る。名前だけ足して分岐を足し忘れると、包みは飛ばされず、後ろの `sh -c` の中を読まない。これまでは手書きの行を足さない限り、このずれはテストで捕まらなかった。

テストの F 節に、実行時に読んだ `WRAPPER` の名前ごとに包みの行を作って確かめる検査を足した。名前 W ごとに `W sh -c 'git commit -n -m x'` と `W 5 sh -c 'git commit -n -m x'` を jq あり・なしで guard に渡し、どちらかが deny なら通す。2 形にするのは、`timeout` だけが次の語を時間として飛ばすため。

## 読む順

1. plan の Design decisions(一覧から行を作る理由、2 形にする理由、手書きの 9 行を残す理由)
2. `tests/test-pre-bash-guard.sh` の `run_guard`(`decide` から判定の読み分けを切り出したもの)
3. 同じファイルの F 節の、DATACMD の検査のあとの新しい検査(`wrapper_row_forms` と、名前 × 経路 × 2 形の三重ループ)と FAIL の文言
4. ヘッダーの D・F 項目と、`edge_deny` の 9 行の上のコメント(役割分担)
5. `docs/tech-debt/README.md` のテストの穴の行の (h) の閉じの文

## 役割分担

- 手書きの 9 行(D 節 `edge_deny`): 名前が `WRAPPER` から消えたことを捕まえる。包みごとの引数(`nice -n 5`、`stdbuf -o0`、`timeout 5`)の読み方も押さえる
- 新しい検査(F 節): `WRAPPER` にある名前に `cmd_pos` の分岐があることを確かめる。新しい包みに手書きの行がなくても働く。包みの引数の読み方は見ない

## 計画からの逸脱

- `decide` から `run_guard` を切り出して、新しい検査と共有した(判定の読み分けを 2 か所に書かないため)。base のテストを HEAD の guard で回すと 2,090 件で失敗 0、節ごとの件数も同じ
- self-review の MEDIUM 1 と LOW 1・2(FAIL の文言、コメントの言い過ぎ)と、再 review の LOW 6(書き換えで入ったコメントの誤り)を、verify の前に直した

## 確かめたこと

- テストは 2,108 件で失敗 0(macOS の awk、mawk、gawk)
- `WRAPPER` に分岐のない `chrt` を足した写しでは、新しい検査の `chrt` の 2 件だけが赤。`cmd_pos` の分岐を 1 つずつ消した 9 通り、`WRAPPER` を空にした写しも赤(3 つの awk で同じ)

## 残したもの

- `edge_deny` のコメントの「包みごとの引数を `sh -c` の前で動かすのはこの 9 行だけ」は、字面では少し広い(B 節の `xargs -I{} sh -c {}` も包みに引数を付けている)。次にこのファイルを触るときに直す
