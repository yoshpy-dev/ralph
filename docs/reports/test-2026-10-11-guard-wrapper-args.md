# Test report: guard-wrapper-args

- Date: 2026-10-11(実行は 14:59 から 15:27 JST まで)
- Plan: docs/plans/active/2026-10-11-guard-wrapper-args.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: base 577ebeaf から HEAD aafae600 まで。コードの変更は `tests/test-pre-bash-guard.sh` だけで、guard のファイル(`.claude/hooks/`)と `tests/fixtures/` は `git diff 577ebeaf HEAD` が空。実行したのは `./scripts/run-test.sh`、guard のテストの macOS の awk・mawk・gawk での実行、base のテストのファイルを HEAD の guard で流す回帰の比較、AC2 の mutation (a)〜(g) と追加の写し、34 行と 34 の対照を guard に直接渡す probe。追跡しているファイルは変えていない(写しの木はすべて scratchpad の `test-wrapargs/` の中)
- Evidence: `docs/evidence/test-2026-10-11-guard-wrapper-args.log`(索引と `run-test.sh` の出力)。個別のログは同じ名前に `-run-test`、`-macos-awk`、`-mawk`、`-gawk`、`-regression`、`-mutations`、`-probe` を付けたファイル。どれも gitignore の対象なのでコミットしない。スクリプトは scratchpad の `test-wrapargs/`(`mutate.sh` は `s5-wrapargs/mutate.sh` に b、r、w0、s1、s2、k1、x3 を足したもの、`run-mutations.sh`、`run-mutations-mawk.sh`、`run-mac.sh`、`docker-head.sh`、`docker-tree.sh`、`compare-labels.sh`、`probe.sh`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0。changed-language scope の指定だが、`tests/test-pre-bash-guard.sh` を言語に分類できないので full fallback で走った) | shell 40 本(PASS の行 4,081、summary の行を除く)+ Go 8 packages | 40 本 + 8 | 0 | 1 | 474 秒 |
| うち `tests/test-pre-bash-guard.sh`(macOS、BWK awk 20200816、bash 3.2.57、jq 1.8.2 あり・なし) | 2,227 | 2,227 | 0 | 0 | run-test の中 |
| うち ほかの shell 39 本 | — | 39 本 | 0 | 1(`test-secret-scan-branch.sh` が、git 2.41 未満の実機の場合を手元の git 2.49.0 では飛ばす。stub の古い git の場合は通る) | run-test の中 |
| うち `go test ./...`(golang verifier) | 8 packages | 8 | 0 | — | `internal/org` 23.8 秒は実行、残り 7 つは cached(Go のファイルは変わっていない) |
| `bash tests/test-pre-bash-guard.sh`、macOS の `/usr/bin/awk`(run-test と並行) | 2,227 | 2,227 | 0 | 0 | 107 秒 |
| 同じテスト、Docker `ralph-guard-awk:test` の mawk 1.3.4 20240123(`git archive HEAD` を stdin で渡した) | 2,227 | 2,227 | 0 | 0 | 17 秒 |
| 同じテスト、同じ image の gawk 5.2.1 | 2,227 | 2,227 | 0 | 0 | 21 秒 |
| base 577ebeaf のテストのファイル + HEAD の guard、gawk / mawk | 2,108 × 2 | 2,108 × 2 | 0 | 0 | 19 秒 / 18 秒 |
| mutation の写し(テスト全体)、gawk | 37 runs(AC2 の 15、前の版のテストでの 10、等価の 4、追加の 7、変更なしの 1) | — | 下の Mutation の表 | — | 1 本 18〜22 秒 |
| 同じ写しのうち 22 個、mawk | 22 runs | — | gawk と FAIL の記録が 22 個とも byte 単位で同じ | — | 1 本 約 20 秒 |
| 同じ写しのうち (f)、(g) の g1・g6・g9、macOS の awk | 4 runs | — | gawk と FAIL の記録が 4 個とも byte 単位で同じ | — | 4 本並行で 430 秒 |
| 34 行と 34 の対照の直接の probe(jq あり・なし) | 136 | 136 | 0 | 0 | 下の Edge cases |

節ごとの件数(macOS の awk、mawk、gawk で同じ): A 476、B 752、C 116、D 678、E 54、F 137、G 7、H 3、I 4。合計 2,227 は plan の AC3 の式 2,108 + 50(D: 25 行 × 2 経路)+ 1(F: 集合の検査)+ 68(F: 34 行の対照 × 2 経路)と一致する。F の 137 件の内訳は、base からある 68 件(DATACMD の不変条件 1、PR #219 の包みの行 18 など)と、新しい 69 件。新しい集合の検査は `(9 names, 34 rows; this checks only that the rows exist and their shape, not whether a row pins its wrapper's own arguments)` で PASS した。H の 3 件は macOS で 0.30 秒、1.16 秒、0.62 秒(上限 10 秒)。

## Coverage

- Statement: 計測していない(shell の計測の道具がない)
- Branch: 計測していない
- Function: 計測していない
- Notes: 範囲で数えると、実行時の `WRAPPER` の 9 つの名前すべてに `wrapper_deny` の行があり(名前ごとに 2〜6 行)、34 行すべてが jq あり・なしで deny、34 の対照すべてが none になった。plan の「分岐ごとの基準」が挙げる道は、AC2 の (f) と (g) で、道を壊すと行が落ちることを確かめた(行が区別できない 4 つの道を除く。Test gaps の 2)。新しい検査の FAIL の分岐(行のない名前、一覧にない 1 語目、末尾の不一致、対照が deny、対照が deny 以外、`WRAPPER` が空)は、それぞれ (b)、(c) と r、(d)、(e)、x3、w0 の写しで実行した

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | HEAD では、3 つの awk のどれでも FAIL がない | — |

mutation の写しで出た FAIL は、写しが狙った記録だけが落ちたもので、HEAD の失敗ではない(下の Mutation)。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 今の 2,108 件(base のテスト)が HEAD の guard でも通る | 通る(gawk、mawk とも 2,108/0) | `-regression.log` |
| base の結果の行が HEAD の結果の行にすべて残る | 残る。結果の行(状態と文言の全体、H の時間と payload の大きさだけ正規化)を節ごとに比べると、base にだけある行は A〜I のどの節でも 0。HEAD にだけある行は D が 50、F が 69、ほかの節は 0 | `-regression.log` の比較の表(gawk、mawk で同じ) |
| D 節の 9 行を `edge_deny` から `wrapper_deny` に移してもラベルが変わらない | 変わらない。上の比較で D の base にだけある行が 0 なので、9 行のラベルはそのまま。HEAD にだけある D の 50 行は、コマンドで数えると 25 種で、Scope の 25 行と同じ | 同上 |
| G 節の corpus と件数が変わらない | 変わらない(G 7 件、行も同じ) | 同上 |
| guard のファイルが変わっていない | `git diff --stat 577ebeaf HEAD -- .claude/hooks tests/fixtures` が空 | このレポートの実行 |

## Mutation

写しの木(guard の 5 ファイル、テストのファイル、`tests/fixtures/guard-1c4cea5a/`)に 1 つずつ変更を入れ、テスト全体を流した。FAIL の記録は gawk の結果で、mawk(22 個)と macOS の awk(4 個)の結果は byte 単位で同じだった。写しごとの diff と FAIL の記録の全文は `-mutations.log` にある。

### AC2 (a)〜(g)

| ID | 写しの変更 | 結果(PASS / FAIL) | FAIL の記録 |
| --- | --- | --- | --- |
| (a) | `WRAPPER` に `chrt` を足し、`cmd_pos` に `skip_opts(ctx, i + 1, "", "")` の分岐を足す | 2,228 / 1 | F の集合の検査 1 件: `(10 names, 34 rows: chrt is in WRAPPER but has no row: add a row for chrt to wrapper_deny; …)`。PR #219 の検査の `chrt` の 2 件は PASS |
| (b) | `nohup` の 2 行(素の行と `nohup --`)を消す | 2,218 / 1 | F の集合の検査 1 件: `(9 names, 32 rows: nohup is in WRAPPER but has no row: …)` |
| (c) | `WRAPPER` から `nohup` を消す | 2,220 / 5 | F の集合の検査 1 件(`nohup sh -c …` と `nohup -- sh -c …` の 2 行が `starts with nohup, which is not in WRAPPER`)と、D の `nohup sh -c …`・`nohup -- sh -c …` の jq あり・なし 4 件(got none) |
| (d) | 素の `nohup` の行の中を force push にする | 2,224 / 3 | F の集合の検査 1 件(`does not end in sh -c 'git commit -n -m x'`)と、その行の対照 2 件(`its control cannot be built`)。D のその行は sentinel で deny のまま PASS |
| (e) | 素の `time` の行を `time -o 'sudo log' sh -c …` にする(末尾は保つ) | 2,225 / 2 | その行の対照の jq あり・なし 2 件: `its control time -o 'sudo log' sh -c 'git commit -m x' -> deny (exit 0), expected none with exit 0: the deny of this row does not come only from …`。集合の検査は PASS |
| (f) | `cmd_pos` の `time` と `exec` の分岐の値を取る旗を空にする | 2,219 / 8 | D の `exec -a n`、`time -o f`、`time -f %e`、`time --format %e` の jq あり・なし 8 件(got none)。`time --format=%e` は deny のまま。macOS の awk でも同じ 8 件 |
| (f)-pre | (f) と同じ guard に 23b1ac05 のテスト(16 行を足す前) | 2,127 / 0 | なし(緑)。足した行がこの写しを捕まえている |
| g1 | `timeout` の分岐の値を取る長いオプションの一覧を空にする | 2,225 / 2 | D の `timeout --signal KILL 5s sh -c …` × 2。macOS の awk でも同じ |
| g2 | `stdbuf` の分岐の値を取る短い旗を空にする | 2,225 / 2 | D の `stdbuf -o L sh -c …` × 2 |
| g3r | `skip_env` の `--` の行を `return i` にする | 2,225 / 2 | D の `env -- sh -c …` × 2 |
| g4 | `skip_command` の `--` の行を消す | 2,225 / 2 | D の `command -- sh -c …` × 2 |
| g5r | `skip_xargs` の `--` の行を `return i` にする | 2,225 / 2 | D の `xargs -- sh -c …` × 2 |
| g6 | `skip_env` の比較から `--unset` を外す | 2,225 / 2 | D の `env --unset HOME sh -c …` × 2。macOS の awk でも同じ |
| g7 | `skip_xargs` の値を取る長いオプションの一覧から `--max-args` を外す | 2,225 / 2 | D の `xargs --max-args 1 sh -c …` × 2 |
| g8 | `skip_xargs` の一覧にない長いオプションの `else i++` を `else return i` にする(`=` の 1 語の道) | 2,225 / 2 | D の `xargs --max-args=1 sh -c …` × 2 |
| g9 | `skip_opts` の `-` と数字の旗の行を消す | 2,225 / 2 | D の `nice -5 sh -c …` × 2。macOS の awk でも同じ |
| g*-pre | g1〜g9 の 9 つの写しに b7885236 のテスト(9 行を足す前) | 9 つとも 2,191 / 0 | なし(緑)。足した 9 行がこれらを捕まえている |

(g) の 9 つの写しは、どれも狙った道の行だけが jq あり・なしの 2 件で落ち、その行の対照は PASS のままだった。

### 等価の写し(緑を期待)

| ID | 写しの変更 | 結果 | 読み |
| --- | --- | --- | --- |
| g3 | `skip_env` の `--` の行を消す | 2,227 / 0 | `--` は値を取らない旗の道(`-` で始まる語を 1 つ飛ばす)を通って `sh` で止まる。テストのコメントの「行が区別できない道」のとおり |
| g5 | `skip_xargs` の `--` の行を消す | 2,227 / 0 | `--` は一覧にない長いオプションの道を通って `sh` で止まる。同上 |
| p1 | `skip_opts` の `--` の行を消す | 2,227 / 0 | テストのコメントと tech-debt が挙げる 3 つ目の `--` の道。同上 |
| p2 | `skip_env` の代入の行を消す | 2,227 / 0 | `cmd_pos` が代入を自分で飛ばす。同上 |

### 追加の写し(plan の Test plan の edge case と、コメントの主張の確かめ)

| ID | 写しの変更 | 結果 | FAIL の記録と読み |
| --- | --- | --- | --- |
| b | 素の `nohup` の 1 行だけ消す(`nohup --` は残す) | 2,223 / 0 | 緑。行の存在しか見ないので、名前に行が 1 行残れば通る。plan の Non-goals と Progress の「AC2 (b) の読み方」のとおり |
| r | `wrapper_deny` に `chrt sh -c …` の行を足す(`WRAPPER` には足さない) | 2,228 / 3 | F の集合の検査 1 件(`starts with chrt, which is not in WRAPPER: … or the row names a wrapper the guard does not have`)と D のその行の 2 件(got none)。対照の 2 件は PASS |
| w0 | `WRAPPER` を空にする | 2,133 / 77 | F 3 件(DATACMD の不変条件、PR #219 の検査、新しい検査の `no WRAPPER name was read from the three .awk files, so the rows were not compared`)、D の `wrapper_deny` の 34 行 × 2 = 68 件、D の lexer だけが deny する 3 行 × 2 = 6 件(`env -S 'git push origin --force'`、`env --split-string=…`、`echo x \| xargs sh -c 'git push origin --force'`)。対照の 68 件は PASS。V-1 のコメントが挙げる sentinel つきの行(`nice -n 5 sudo ls`、`xargs -I{} sudo ls {}`、`xargs -I{} flock /tmp/l git push -f`、B の 2 行)は、包みがすべて一覧から消えても deny のままだった |
| s1 | `skip_env` の `-S` の行を消す | 2,225 / 2 | D の `env -S 'git push origin --force'` × 2。コメントの「`env -S` の道は `edge_deny` の lexer だけが deny する行が押さえる」のとおり |
| s2 | `skip_env` の `--split-string=` の行を消す | 2,225 / 2 | D の `env --split-string='git push origin --force'` × 2。同上 |
| k1 | `skip_env` の比較から `--chdir` を外す | 2,227 / 0 | 緑。`--chdir` は `--unset` と同じ道の別の旗で、基準は旗ごとではなく分岐ごとなので、これを押さえる行はない(Test gaps の 3) |
| x3 | guard の 2 行目に `exit 3` を入れる(どの判定も none、exit 3) | 14 / 2,213 | 対照の 68 件の FAIL の文言は `its control … -> none (exit 3), expected none with exit 0` で、deny のときの理由(`rewrite the row`)は付かない。Progress が「読んで確かめた」とした LOW 6 の「判定が deny でないとき」の文言を、実行で確かめた |

## Edge cases(guard への直接の probe)

`probe.sh` が HEAD のテストのファイルから `wrapper_deny` の 34 行を読み、各行と対照を case のファイルに書いて、worktree の guard に渡した。payload は `jq -Rs` で組み、対照は中の `-n -m x` を `-m x` に置き換えて作った(テストの末尾の置き換えとは別の作り方)。jq なしの経路は、テストと同じ道具だけを置いた PATH。

- 34 行: jq あり・なしとも deny、exit 0
- 34 の対照: jq あり・なしとも none、exit 0
- 参考: 旧版の guard(`tests/fixtures/guard-1c4cea5a/`)は、jq ありで 34 行も 34 の対照も none(旧版は `sh -c` の文字列を読まないので、どの行も deny しない)

## Test gaps

1. 新しい検査は行の存在と形だけを見る。名前に行が 1 行でも残れば、ほかの行を消しても緑(写し b、2,223/0)。行がその包みの引数を `sh -c` の前に置いているかは判断しない。plan の Non-goals と、PASS の文言のとおり
2. 行が区別できない道が 4 つある。`skip_opts`・`skip_env`・`skip_xargs` の `--` の行と、`skip_env` の代入の行は、消しても語が別の道を通って `sh` で止まるので、どの行も落ちない(写し g3、g5、p1、p2)。テストのコメントと tech-debt に書いてある
3. 旗ごとには押さえない。同じ比較の中のもう 1 つの旗(`skip_env` の `--chdir`)を外した写し k1 は緑だった。`timeout` の `--kill-after`、`stdbuf` の `--input` と `--error` など、同じ道の別の旗もおそらく同じ扱いになる。k1 以外は実行しておらず、未確認です。基準が「分岐ごと、旗ごとではない」と決めた範囲で、分岐の表にないツールの旗も、どの行も押さえない
4. `wrapper_deny` の外にも、素の包みの道を押さえる行がある。w0 で、D の lexer だけが deny する `echo x | xargs sh -c 'git push origin --force'` も落ちた。`wrapper_deny` の上のコメントの「Other rows also run a wrapper」の列挙には入っていないが、包みの引数を置かない行なので、コメントの結論(引数の読み違えで落ちるのは `wrapper_deny` の行だけ)とは矛盾しない。情報として残す
5. macOS の awk で流した mutation は (f)、g1、g6、g9 の 4 つだけ。ほかは gawk と mawk で、両者の FAIL の記録は 22 個とも同じだった
6. 対照の判定が `unparsed` や `ask` になる場合の文言は、guard を変えずに出せないので実行していない。同じ `why=""` の分岐は x3(none、exit 3)で実行した

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` rc 0(shell 40 本と Go 8 packages)。`tests/test-pre-bash-guard.sh` は macOS の awk、mawk 1.3.4、gawk 5.2.1 で 2,227/0/0、節ごとの件数は plan の値どおり。base のテストは HEAD の guard で 2,108/0 で、base の結果の行はすべて HEAD に残る(D +50、F +69)。AC2 の (a)〜(g) はどれも、plan の Progress に書かれた記録が落ちた。(f) と (g) は前の版のテストでは緑で、足した行が捕まえている。等価の写し 4 つは期待どおり緑。34 行と 34 の対照は直接の probe でも deny と none
- Fail: なし
- Blocked: なし
