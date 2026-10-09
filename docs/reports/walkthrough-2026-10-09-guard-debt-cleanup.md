# Walkthrough: guard-debt-cleanup

- Date: 2026-10-10
- Plan: docs/plans/archive/2026-10-09-guard-debt-cleanup.md(この PR の最後のコミットで archive に移す)
- Branch: refactor/guard-debt-cleanup(base main a0094fe5)
- Diff(この walkthrough を除く): 18 files、+826 / -124。報告・insight・plan・`templates/` の写しを除くと 7 files、+182 / -65

## 何を変えたか

PR #214 のあとに tech-debt に残った、guard まわりの 3 つを片づけた。判定は 1 つも変えていない。

- `scripts/verify.local.sh` の static の検査に、guard の 3 つの `.awk` を guard と同じ順で 1 つの awk に読ませる段(`check_guard_awk`)を足した。これまで `.awk` の構文エラーを捕まえるのはテストだけだった。guard の `.sh` がない木(テストの fixture)は、ほかの `.sh` の段と同じく飛ばす
- awk のコメントで単一引用符を `SQ`、二重引用符を `DQ` と書いていたのを、本物の引用符に戻した(15 行)。awk のプログラムを単一引用符で渡していたころの名残で、PR #214 で制約がなくなっていた
- `read_body` の、行末のバックスラッシュを数える 2 つのループを `trailing_backslashes` にまとめた
- `end_cmd` の 2 つの規則を消した。1 語目が予約語なら `NODATA`、`exec` のあとにリダイレクトがあれば `NODATA`、の 2 つで、どちらも許可リスト(`data_first_ok`)が同じ場面で必ず `NODATA` にする。DATACMD と予約語の一覧は重ならず、`cmd_pos` が飛ばす包みも DATACMD にない。あわせて `RESW` の一覧と `EXEC_SEEN` も消した
- 規則が守っていた前提をテストにした。実行時の DATACMD を読み、予約語と包みが入っていないことを確かめる検査と、`exec >run.sh; echo 'sudo ls'` の行(後ろに `sh` を置かない形)を足した

## 読む順

1. plan の Assumptions と Design decisions(規則を消してよい理由、Codex の plan advisory を受けて足したテスト)
2. `pre_bash_guard_commands.awk` の `end_cmd`、`cmd_pos`、`data_first_ok`、`in_data` と、`pre_bash_guard_rules.awk` の `datacmd_list` の上のコメント
3. `pre_bash_guard_lex.awk` の `read_body` と `trailing_backslashes`
4. `tests/test-pre-bash-guard.sh` の F 節の最後の不変条件の検査と、B 節の `exec` の行
5. `scripts/verify.local.sh` の `check_guard_awk` と、`run_static_checks` の段
6. `docs/tech-debt/README.md` の guard の限界の行、テストの穴の行、末尾の `.awk` の行

## 計画からの逸脱

- S1b: S1 の段が、guard のない fixture で `.awk` を開けずに FAIL し、`tests/test-archive-plan.sh` が落ちた。`pre_bash_guard.sh` のない木は飛ばすようにした(S1 ではテストを回していなかった)
- AC2 の grep は部分一致で、変数名を指すコメントの `DQ_SUB` などにも当たる。語の境界を付けた grep で確かめた
- 不変条件の検査は、plan ではソースの `split` の行を読む形だった。self-review の指摘で、実行時の DATACMD を読む形に変えた(`DATACMD["exec"] = 1` のように別の書き方で足した名前も捕まえる)

## 確かめたこと

- テストは 2,068 件で失敗 0(macOS の awk、mawk、gawk)
- base(a0094fe5)とこの PR の guard に 3,913 件の入力を渡し、判定の違いは 0 件(3 つの awk すべて)。`read_body` を壊すと判定が変わる 280 行も含む
- DATACMD に `exec` を足すと不変条件の検査と `exec` の行が、`for` を足すと不変条件の検査が、許可リストの行を消すと `if grep -q` の行が、それぞれ落ちる

## 残る穴

- 不変条件の検査は、包みの名前を `cmd_pos` の中の `nm == "…"` から読む。包みを別の関数に移すと、検査から黙って外れる
- `trailing_backslashes` の下限を変える mutant を殺す行がない(base でも同じ)。足す 2 行の案を tech-debt に書いた
