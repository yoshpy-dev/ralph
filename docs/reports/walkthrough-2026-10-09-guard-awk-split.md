# Walkthrough: guard-awk-split

- Date: 2026-10-09
- Plan: docs/plans/archive/2026-10-09-guard-awk-split.md(この PR の最後のコミットで archive に移す)
- Branch: refactor/guard-awk-split(base main 0abfede5)
- Diff(この walkthrough を除く): 20 files、+3,632 / -2,854。そのうち約 1,380 行は awk のプログラムを `.sh` から 3 つの `.awk` に移した行で、root と `templates/base/` の両方に出る

## 何を変えたか

`.claude/hooks/pre_bash_guard.sh` は 1,596 行で、そのうち 1,378 行が単一引用符で渡す awk のプログラムだった。800 行の目安を満たすため、プログラムを節の境目で 3 つのファイルに移し、`awk -f` で順に読むようにした。判定は 1 つも変えていない。

| ファイル | 行数 | 持つ節 |
| --- | --- | --- |
| `pre_bash_guard.sh` | 231 | ヘッダーの説明、JSON の取り出し、awk の呼び出し、旧版の 4 規則の fallback、deny の文言 |
| `pre_bash_guard_lex.awk` | 616 | Text access、Lexing |
| `pre_bash_guard_commands.awk` | 336 | Simple-command assembly、Data regions |
| `pre_bash_guard_rules.awk` | 465 | Rule judgement、The sentinel、Main(BEGIN と END) |

`.awk` が 1 つでも読めないと awk は exit 2 で止まり、今までの「awk がない」ときと同じく旧版の 4 規則で決まる。欠けたファイルは `check-template.sh` の `required_files` で見えるようにした。

あわせて、tech-debt の guard の行に溜まっていたコメントの直しと、PR #213 が持ち越したテストの行を片づけた。

## 読む順

1. plan の Design decisions(3 つに分ける理由、fallback に任せる理由、2 段のロールバック)
2. S1(da3b55d2)の差分。`.sh` は `174,1553c174` の 1 hunk で、3 つの `.awk` は base の 175〜1552 行目をそのまま切り出したもの。`cat` でつないで `cmp` すると一致する
3. `.sh` の awk の呼び出しの上のコメント(3 つのファイルの分担と読む順。説明はここ 1 か所にまとめた)と、各 `.awk` の先頭のコメント
4. S2(7347fa6a)と S2b(dfb8785a): テストの行、F 節の fallback と parse の確認、`check-template.sh` とそのテスト
5. S3(f593bbfc)、c3a323aa、S5(8c61c6cb): コメントだけの直し。awk のコードの行は S1 から変わっていない(コメントと空行を除くと base と一致、982 行)
6. `docs/tech-debt/README.md` の guard の限界の行、テストの穴の行、末尾の `.awk` の分類の行

## 計画からの逸脱

- S2b: `check-template.sh` の `required_files` を変えると `tests/test-check-template.sh` の固定の写しが 9 件落ちるので、このテストを範囲に足した。plan の Affected areas の見落としで、AC6 に要る変更
- S3 で、テストに `git commit -m $"never sudo ls"` の 1 行を足した(S2 でヘッダーに書いた `$"…"` を固定する行)
- c3a323aa は、S3 の implementer が見つけた `new_ctx` のコメントの抜けを inline で直したもの
- S5 は self-review の LOW 1〜6 を verify の前に直し、reviewer に見直してもらった(PR #213 では self-review のあとに直して規則から外れたため)

## 残る穴

- 字句解析とシェルで読みが違う 2 つの形(`$'\echo' 'sudo ls'` と `$"echo" 'sudo ls'`)は、新版が通し、旧版は止める。どちらも分割の前から通っていた。シェルが実際に動かすのは ESC と `cho`、または `$echo` という名前のコマンドで、普通は存在しない。tech-debt の Debt (b) に書いた
- tech-debt (e) のコードの項目(`read_body` の重複したループ、判定を変えない予約語と `exec` の規則)と、awk のコメントの `SQ` の綴りは残した
- `scripts/detect-changed-languages.sh` が `.awk` を分類できず、static verify が full の範囲で走る。`verify.local.sh` に `.awk` の構文の確認がない。どちらも tech-debt の新しい行に書いた
