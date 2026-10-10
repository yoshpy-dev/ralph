# sync-docs report: guard-wrapper-rows

## Cycle 1

- Date: 2026-10-10
- Plan: `docs/plans/active/2026-10-10-guard-wrapper-rows.md`
- Pipeline cycle: 1。差分は base `382c18c8` から branch HEAD `f55e8a5a`(test/guard-wrapper-rows)まで。コードの変更は `tests/test-pre-bash-guard.sh` だけ(`0eebc7e7` が F 節の包みの行の検査と `run_guard` の切り出し、`35f00d33` が FAIL の文言とコメント、`fc97d108` と `6ae86c5f` がコメントだけ)。guard のファイルは変わっていない
- 先行 report(それぞれを追加した commit。各 report 内の `HEAD:` の欄は、その report を commit する前の値なので使っていない):
  `docs/reports/self-review-2026-10-10-guard-wrapper-rows.md`(`336f5856`、再 review は `65d608b9`。Merge yes。LOW 3 の tech-debt 側が sync-docs 向け)、
  `docs/reports/verify-2026-10-10-guard-wrapper-rows.md`(`f9127f47`。pass、V-1・V-2 と情報 I-1〜I-3)、
  `docs/reports/test-2026-10-10-guard-wrapper-rows.md`(`f55e8a5a`。pass、guard のテスト 2,108/0、mutation 11 個が 3 つの awk で赤)

## Summary

古くなっていたのは、tech-debt のテストの穴の行の (h) の閉じの文 1 句と、plan の進捗だった。self-review の LOW 3(tech-debt 側)と verify の V-2 は同じ指摘で、「新しい包みは名前を `WRAPPER` に、分岐を `cmd_pos` に足せば足りる」と読める句を直した。新しい検査が確かめるのは、分岐があること(決まった 2 形が `sh -c` に届くこと)までで、分岐が包み自身の引数をどう読むかは見ない。引数や時間を取る包み(`timeout` の `+ 1`、`nice` の `n` オプション)の読み方を押さえるのは、`edge_deny` の手書きの行だけである。テストのヘッダーの F 項目は `fc97d108` でこう直っているので、2 つの文書の言い方がそろった。mutation の結果(10 個目の名前 `chrt` と、9 つの分岐を 1 つずつ消したもの、すべて赤)は残し、3 つの awk で赤だったことと test report の節名(Mutation)を足した。Related には、「all caught」の根拠として test report を足した。

ほかの文書(AGENTS.md、README.md、`.claude/rules/ralph/`、`docs/quality/`、`docs/architecture/`、`docs/recipes/`、`docs/specs/`、`templates/base/`)に、この PR で間違いになる記述はなかったので、変更していない。

差分の大きさ(`git diff 382c18c8...f55e8a5a --shortstat`、この report を足す前): 7 files changed, 463 insertions(+), 15 deletions(-)。docs を除いたコードの変更は `tests/test-pre-bash-guard.sh` の 91 insertions(+), 14 deletions(-)(105 行)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(テストの穴の行、163 行目) | 2 点を直した。(1) LOW 3 / V-2: (h) の 2 つ目の閉じの文(「Closed in test/guard-wrapper-rows (0eebc7e7): …」)の「so a new wrapper needs only its name in `WRAPPER` and its branch in `cmd_pos`, and a name without a branch fails the check」を、「so a new wrapper whose name has no branch in `cmd_pos` fails the check」にし、続けて「The check confirms only that the branch exists, that is, that the two fixed forms reach `sh -c`; it does not check how the branch reads the wrapper's own arguments (for example the `+ 1` that makes `timeout` skip its duration, or the `n` option of `nice`), which only a hand-written row in `edge_deny` pins, so a wrapper with arguments of its own still needs a row there.」を足した。mutation の括弧は「all caught: red on BWK awk, mawk, and gawk; test report of this PR, Mutation」とした。「The nine hand-written rows stay …」と `wrapper_row_forms` の文は変えていない。(2) Related に `docs/reports/test-2026-10-10-guard-wrapper-rows.md`(Mutation、(h) の閉じの「all caught」の根拠)を足した。Impact・Why deferred・Trigger の (h) は、すでに閉じた書き方なので変更なし |
| `docs/plans/active/2026-10-10-guard-wrapper-rows.md` | `## Progress checklist` だけを変えた。S3b と S3c の行の「この commit」を `fc97d108` と `6ae86c5f` にし、verify(`f9127f47`)、test(`f55e8a5a`)、sync-docs(この commit)の 3 行を Implementation started の下に足し、Review / Verification / Test artifact created にチェックを付けた。「PR created」は `/pr` が付けるのでそのまま。`./scripts/plan-visual.sh digest` は編集のあとで `595308ce450b`(`- Approved:` 行と一致) |
| `docs/insights/events/2026-10-10-guard-wrapper-rows.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`、cycle 1) |
| `docs/reports/sync-docs-2026-10-10-guard-wrapper-rows.md` | この report |

tech-debt の行は、行の番号ではなく関数・変数・test の節の名前で書いた(新しい `file:line` の参照はない)。plan の `git diff` で変わったのは Progress checklist の中だけ(S3b・S3c の 2 行、Review / Verification / Test の 3 行のチェック、足した 3 行)。

handoff の言い方との違い: handoff は base のテストのファイルについて「節ごとの数が同じ」と書いたが、test report は F だけが 50 から 68 になる(新しい検査の 18 件)と書く。ほかの節(A 476、B 752、C 116、D 628、E 54、G 7、H 3、I 4)は同じ。plan には test report の言い方で書いた。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `git grep -E '2,?090\|2,?108\|tenth wrapper\|a tenth\|wrapper row\|run-time .?WRAPPER\|nine hand-written\|2068'`(`docs/reports/`、`docs/plans/`、`tests/fixtures/` を除く) | 当たったのは `docs/tech-debt/README.md` の 163 行目とテストのファイルだけ。163 行目は上のとおり直した。テストのファイルは、ヘッダーの D・F 項目と `edge_deny` のコメントを `fc97d108` と `6ae86c5f` までに直してあり、verify が drift なしと確認している(V-1 を除く) |
| `git grep -l -i -E 'DATACMD\|NOEXEC\|cmd_pos\|wrapper_row\|edge_deny'`(`docs/quality`、`docs/architecture`、`docs/recipes`、`docs/specs`、`.claude/rules`、`AGENTS.md`、`README.md`、`CLAUDE.md`、`templates/base`。`.awk`・`.sh`・`.json` を除く) | 当たりなし |
| `git grep -l test-pre-bash-guard`(`docs/reports`、`docs/plans`、`tests/`、`.sh`、`.awk` を除く) | `docs/tech-debt/README.md` だけ。README.md、AGENTS.md、`docs/quality/`、`docs/architecture/`、`.claude/rules/` はこのテストのファイルの件数も節も書いていない |
| `docs/tech-debt/README.md` の `WRAPPER` を含む行 | 160 行目(guard の限界の行)と 163 行目だけ。160 行目は「wrapper names are read from the `WRAPPER` list at run time」と書き、今も正しい。変更なし |
| tech-debt の表の形(スクラッチの `tdcheck.py`、`git show HEAD:` との行ごとの比較) | 変わった行は 163 だけ(199 行のまま)。区切り ` \| ` は 4 つ(5 列)、`\|` で分けた片は 7 のまま。列ごとのバッククォートは偶数、`~~` は偶数。新しい `file:line` 参照なし |
| `docs/tech-debt/README.md`(163 行目の Related)の plan のパス | `docs/plans/active/2026-10-10-guard-wrapper-rows.md` のまま。`/pr` の `archive-plan.sh` が動かす前に書き換える(verify I-1)ので、手では直さない。`verify.local.sh` の「tech-debt README plan references」は OK |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。template に写しのあるファイルは変えていない |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この sync の差分は tech-debt、plan の進捗、report、insight だけ。該当なし |

## Found but left

- verify V-1(LOW): テストの `edge_deny` の 9 行の上のコメントの「only these rows run each wrapper with arguments of its own」は、B 節の `xargs -I{} sh -c {}` も自分の引数を付けた `xargs` を `sh -c` の前で動かすので、字面どおりには広い。意図して残した。直すのはテストのファイルのコメント 1 文で、この依頼は `tests/*` を変えない取り決めである。verify の言うとおり、中身(引数の読み方を `sh -c` の前で押さえるのは D の 9 行だけ)は保たれる。直すなら次にこのファイルを触るときに、「only these rows run nice, stdbuf and timeout with arguments of their own (nice -n 5, stdbuf -o0, timeout 5) before sh -c」の形にする(verify の提案)
- verify I-3: `pre_bash_guard_rules.awk` の `WRAPPER` の上のコメントは、F 節がこの一覧から行を作ることを書いていない。偽ではなく、この PR は guard のファイルを変えない取り決めなので、次に guard を直す機会に足す
- verify I-1: Related の plan のパスは `active/` のまま(`/pr` が書き換える。`archive/` を先に書かない)
- self-review LOW 4(検査を関数に出す): plan が任意として見送ったので、記録だけ
- マージ後に main のチェックアウトから動く guard の効き目は、マージ前には確かめられない(test report の Test gaps と同じ。この PR は guard を変えていない)

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-10-guard-wrapper-rows.md` | `595308ce450b`(編集のあと。plan の `- Approved:` 行の `sha256:595308ce450b` と一致。Progress checklist の変更では変わらない) |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(`check-sync`、`check-pipeline-sync`、`check-skill-sync` 13 件、`check-template-purity`、tech-debt README plan references がすべて OK)。編集のあとに実行 |
| tech-debt の 163 行目の形(`tdcheck.py`) | 上の表のとおり。変わった行は 163 だけ |

この commit のあとに `./scripts/secret-scan-branch.sh --strict` を流し、exit 0 なら push する(結果は返信に書く)。

## Not verified

- guard のテストスイートと mutation は流していない(`/test` が確認済み)。tech-debt に書いた「all caught: red on BWK awk, mawk, and gawk」は test report の Mutation の節(11 個の mutant を 3 つの awk に通して 33 runs すべて赤)からの転記。macOS の BWK awk 20200816、mawk 1.3.4、gawk 5.2.1 だけで、busybox awk は見ていない
- 「`timeout` の `+ 1`」と「`nice` の `n` オプション」は、`pre_bash_guard_commands.awk` の `cmd_pos`(`nm == "timeout"` の分岐の `skip_opts(…) + 1`、`nm == "nice"` の分岐の `skip_opts(ctx, i + 1, "n", " --adjustment ")`)を読んで書いた。実行して確かめたのは verify の probe と test report の mutation で、この sync では実行していない
- 「ほかの文書に間違いになる記述はない」は、上の grep の範囲の確認。guard の包みを別の言い方で書いた文を全文書で網羅したわけではない。未確認です
