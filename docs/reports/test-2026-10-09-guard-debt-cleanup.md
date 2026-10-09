# Test report: guard-debt-cleanup

- Date: 2026-10-09(実行は 2026-10-09 23:44 から 2026-10-10 00:20 JST まで)
- Plan: docs/plans/active/2026-10-09-guard-debt-cleanup.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: base a0094fe5 から HEAD 3358013b まで。`./scripts/run-test.sh`(changed-language scope の指定だが、`.awk` を言語に分類できないので full fallback で走った)、base と HEAD の guard の判定の比較(AC5)、AC5b の mutation、mawk と gawk での実行。追跡しているファイルは変えていない(guard の写しと mutation の木はすべて scratchpad の中)
- Evidence: `docs/evidence/test-2026-10-09-guard-debt-cleanup.log`(gitignore の対象なのでコミットしない)。スクリプトは scratchpad の `gdc/`(`extract2.sh`、`gen-edge.sh`、`gen-edge2.sh`、`compare2.sh`、`mutate.sh`、`docker-inner.sh`、`docker-cmp-inner.sh`、`sens.sh`、`sens2.sh`、`docker-suite-mut.sh`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(shell 40 本 + Go、rc 0) | 40 suites / 3,922 assertions | 40 suites | 0 | 1(下のほかの shell の行) | 8 分 27 秒 |
| `tests/test-pre-bash-guard.sh`(macOS、BWK awk 20200816、jq あり・なし) | 2,068 | 2,068 | 0 | 0 | run-test の中 |
| `tests/test-archive-plan.sh` | 22 | 22 | 0 | — | run-test の中 |
| `tests/test-check-template.sh` | 60 | 60 | 0 | 0 | run-test の中 |
| `tests/test-lib-json.sh` | 126 | 126 | 0 | 0 | run-test の中 |
| `tests/test-ralph-dispatch.sh` | 33 | 33 | 0 | — | run-test の中 |
| `tests/test-verify-local-hook-tests.sh` | 5 | 5 | 0 | — | run-test の中 |
| ほかの shell 34 本 | 1,608 | 1,608 | 0 | 1(`test-secret-scan-branch.sh` が、git 2.41 未満の実機の場合を手元の git 2.49.0 では飛ばす。stub の古い git の場合は通る) | run-test の中 |
| `go test ./...`(run-test の golang verifier) | 8 packages | 8 | 0 | — | `internal/org` 15.6 秒は実行、残り 7 つは cached(Go のファイルは変わっていない) |
| HEAD の guard のテスト、Docker ubuntu:24.04 の mawk 1.3.4 20240123 | 2,068 | 2,068 | 0 | 0 | 14 秒 |
| HEAD の guard のテスト、Docker ubuntu:24.04 の gawk 5.2.1 | 2,068 | 2,068 | 0 | 0 | 17 秒 |
| base と HEAD の判定の比較(macOS、jq あり・なし) | 3,913 payloads / 15,652 runs / 7,826 組 | 7,826 組が一致 | 0 | — | 12 分 02 秒 |
| 同じ比較、Docker の mawk / gawk | 7,826 組 / 7,826 組 | どちらも一致 | 0 / 0 | — | 44 秒 / 63 秒 |
| J の行(下を参照)の比較、macOS / mawk / gawk | 280 payloads / 560 組 × 3 | どれも一致 | 0 | — | 50 秒 / 4 秒 / 4 秒 |
| AC5b の mutation 4 個と対照 1 個(写しの木でテスト全体、macOS) | 5 runs | 対照は 2,068/0、4 個とも赤 | — | — | 4 分 57 秒(5 本を並列) |
| `read_body` の比較の感度を確かめる mutant 3 個(Docker の mawk) | H の 2,580 行と J の 280 行 × 3 | — | — | — | 1 分ほど |

`run-test.sh` の中で、DATACMD の不変条件の検査は `PASS  F. DATACMD invariant: ... (20 names, 9 wrappers, 21 reserved words)`、足した B 節の行 `exec >run.sh; echo 'sudo ls'` は permission_mode なしと bypassPermissions、jq あり・なしの 4 件とも deny で PASS。テストの H 節(約 200 KB の命令)の 3 件は 0.25 秒、1.07 秒、0.57 秒で、上限の 10 秒より十分短い。AC7 の残り半分(`run-verify.sh` の test mode)は、`run-test.sh` が `HARNESS_VERIFY_MODE=test` の `run-verify.sh` なので、この rc 0 で満たす。

件数の数え方: 2,063 件から増えた 5 件は、B 節の `exec >run.sh; echo 'sudo ls'` の 4 件(2 つのモード × jq あり・なし)と、F 節の不変条件の検査の 1 件。

## Coverage

- Statement: 計測なし(shell と awk の計測の道具はない)
- Branch: 計測なし
- Function: 計測なし
- Notes: 変更は判定を変えないことが前提の整理なので、行の網羅率の代わりに、base の guard と HEAD の guard に同じ入力を渡して出力をバイト単位で比べた(Regression checks)。比べた入力が `read_body` の変更を見分けられるかは、わざと壊した写しで確かめた(下の「比較の感度」)。Go のテストは `internal/org` だけが実行され、ほか 7 つは cached。この PR は Go のファイルを変えていない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 判定が変わらないこと(AC5、base a0094fe5 と HEAD 3358013b の guard) | 一致 | 下の「判定の比較」。固有の payload 3,913 を jq の経路と jq のない経路で両方の guard に渡し、15,652 回の実行の 7,826 組で stdout のバイト列と終了コードが一致(stderr も一致)。Docker の mawk と gawk でも同じ 7,826 組が一致 |
| 消した 2 つの規則の前提をテストが守ること(AC5b) | 守る | 下の「Mutation(AC5b)」。4 個とも赤になり、対照の写しは 2,068/0 |
| 足した行 `exec >run.sh; echo 'sudo ls'` は旧版も deny | deny | 比較の行に入っていて、base も HEAD も deny(4 通りとも) |
| AC7 の test mode の `run-verify.sh` | rc 0 | `./scripts/run-test.sh` が rc 0、40 本すべて OK |

### 判定の比較

guard の写しは、base が `git show a0094fe5:` の `pre_bash_guard.sh`・3 つの `.awk`・`lib_json.sh`、HEAD が `git show HEAD:` の同じ 5 ファイル(作業木のファイルと `cmp` で一致することを確かめた)。jq のない経路は、テストと同じ道具の一覧(`sh bash dash cat grep sed printf dirname env tr command test awk`)だけを symlink した PATH で、そこから jq が見えないことを確かめた。

入力(固有の payload 3,913)

- A: a0094fe5 のテストの行。PR #214 の tester の `extract2.sh` で、テスト自身の `json_escape`・`payload_json`・`check`・`check_modes` を `enqueue` だけ差し替えて読み、A〜D 節、E 節、F 節の入れ子と再読の上限、G 節の `intentional_fixes` の jq の経路の行を書き出した(1,030 行、固有の payload 998)。HEAD のテストからも同じように書き出し、足した `exec` の行の 2 つ(2 つのモード)を加えた。節ごとの固有の payload は A 238、B 366、C 53、D 301、E 27、F 15
- P: `scratchpad/mp/l*.txt`。glob は `l01.txt`〜`l17.txt` と、17 行を 1 つの命令にした `lines.txt` の 18 ファイルに当たる。最初の 17 行は PR #214 の tester の `probe.tsv` と同じ payload。18 のうち 13 はテストの行と同じ payload なので、P として数えたのは 5
- R: 予約語で始まる命令(消した `RESW` の規則)。`if`・`for`(`for ((…))` も)・`while`・`until`・`case`・`select`・`function`(`function f()` も)・`{ … }` で始まり、後ろに `; echo 'sudo ls'` を付けた形、`&&` やパイプでつないだ形、改行で区切った形、データを中に置いた形(`if true; then echo 'sudo ls'; fi`、`{ echo 'sudo ls'; }`)、引用符やバックスラッシュを付けた予約語(`'if'`、`\if`、`"{"`、`i\f`)、置換の中の予約語。さらに予約語の一覧 21 語(`if then elif else fi for while until do done case esac select function { } ! [[ ]] time coproc`)のそれぞれを、`w; echo 'sudo ls'`、`w echo 'sudo ls'`、`w true; echo 'sudo ls'`、`w true; git commit -m 'sudo ls'` の 4 形にした。合わせて 119 の命令を、permission_mode なしと auto の 2 通りで 238
- X: `exec` の形(消した `EXEC_SEEN` の規則)。依頼の `exec >f`、`exec 3>f`、`! exec >f`、`env exec >f`、`builtin exec >f` に `; echo 'sudo ls'` を付けたもの、それにリダイレクトのない `exec`、`command exec`、代入付き、`2>&1`・`</dev/null`・`>>`・`&>`・`{fd}>`・`3>&-`、改行区切り、パイプ・`&&`・`||`、引用符付きの `'exec'`・`\exec`、`if` の中、ほかの包み(`nohup`・`time`・`nice`・`timeout`・`xargs`・`stdbuf`・`env -i`)、サブシェルと置換の中。45 の命令を 2 通りで 90
- H: ヒアドキュメントの本文(`read_body` と `trailing_backslashes`)。区切りの書き方 7 通り(`<<EOF`、`<<'EOF'`、`<<"EOF"`、`<<\EOF`、`<<-EOF`、`<<-'EOF'`、`<< EOF`)× 行末のバックスラッシュ 0〜3 個 × 本文 12 通り(1 行目がバックスラッシュで終わる、1 行目を継いだあとの 2 行目・3 行目がバックスラッシュで終わる、`sudo ls` の行がバックスラッシュで終わる、終わりの行 `EOF` の後ろにバックスラッシュ、`E\` と `OF` に分けた終わりの行、バックスラッシュだけの行、終わりの行がなく最後の行がバックスラッシュで終わる、など)× 命令 2 つ(`cat`、`git commit -F -`)× 終わり方 4 通り(最後の改行なし、あり、後ろに `echo 'sudo ls'`、後ろに `echo hi`)、それに置換の中とパイプのあとのヒアドキュメント 4 行。2,692 行、固有の payload 2,580

結果

- macOS(BWK awk): 15,652 回の実行の 7,826 組で、stdout のバイト列か終了コードが違う組は 0、stderr が違う組も 0。HEAD の判定は deny 5,654、none 2,172、それ以外の出力と 0 以外の終了コードは 0。HEAD の jq の経路と jq のない経路の stdout の違いも 0
- Docker の mawk 1.3.4 と gawk 5.2.1: 同じ payload で、違う組はどちらも 0、stderr の違いも 0、HEAD の deny はどちらも 5,654(macOS と同じ数)
- HEAD の判定を節ごとに見ると(jq の経路)、R は deny 236・none 2、X は deny 90・none 0、H は deny 1,894・none 686。R の none の 2 つは `echo $(for x in a; do echo; done) 'sudo ls'` の 2 つのモードで、`for` は置換の中、上の段の命令は `echo` なので、`'sudo ls'` は `echo` の引数のデータになる。base も同じ none

### 比較の感度

0 件という結果が `read_body` の変更を見分けられる入力から出たものかを確かめるため、HEAD の写しの `trailing_backslashes` と `read_body` を 1 行ずつ壊した 3 つの mutant を作り、HEAD との判定の違いを数えた(Docker の mawk、jq の経路)。

| Mutant | 変更 | H の 2,580 行で HEAD と違う行 | J の 280 行で HEAD と違う行 |
| --- | --- | --- | --- |
| m1 | `while (q >= a && …)` を `q > a` に(数え始めの下限) | 0 | 30(HEAD deny → none 12、none → deny 18) |
| m2 | `q = e - 1` を `q = e` に(改行の位置から数えるので、継ぎが起きない) | 0 | 48(HEAD deny → none 6、none → deny 42) |
| m3 | 2 つ目の呼び出し `trailing_backslashes(p2, le)` を `(ls, le)` に | 0 | 0 |

- H の行はどの mutant でも判定が変わらなかった。入力のどこかにバックスラッシュと改行が並ぶと、データ区間はなくなり(`rules.awk:450` の `if (index(IN, BS "\n")) NODATA = 1`)、命令全体に掛かる見張りの規則が `sudo ls` を見て deny にする。バックスラッシュのない行(0 個)は、どの mutant でも継がない。H の行の後ろの命令は `echo 'sudo ls'` と `echo hi` で、命令として読まれても見張りの規則のほかに止める規則がない。だから本文がどこで終わるかは判定に効かず、H の行の一致だけでは `read_body` の関数化が正しいことは言えない
- そこで J の行を足した。終わりの行 `EOF` の 1 行前を、バックスラッシュ 0〜3 個で終わる行にし、`EOF` の後ろにテストの B 節の形 `git commit -n -m x` と最後の `EOF` を置く。継ぎが起きれば `git commit -n` は本文(none)、起きなければ命令(deny、`no_verify`)になる。本文に見張りの語を置かないので、判定は継ぎの数え方で決まる。本文 5 通り(1 行目を継ぐ、継いだあとの 2 行目、バックスラッシュだけの 1 行目、継いだあとのバックスラッシュだけの行、`E\` と `OF`)× 区切り 7 通り × 0〜3 個 × 最後の改行の有無で 280 行。base と HEAD の比較は macOS・mawk・gawk のどれでも 560 組が一致し、m1 と m2 は J の行で判定が変わる
- m3 は判定を変えようがない。2 つ目の呼び出しの `p2` は前の行の改行の次の位置なので、`at(p2 - 1)` は必ず改行で、数えるループは下限に届く前に止まる。下限を `ls` にしても数は変わらない
- テストスイート自身は m2 を殺し(D 節の `cat <<-EOF\n\tEO\\\nF\ngit push origin --force` と `cat <<EOF\nE\\\nO\\\nF\ngit push origin --force` の 2 行、jq あり・なしで 4 件が赤)、m1 は殺さない(2,068/0)。m1 は Test gaps に書いた

### Mutation(AC5b)

`git archive HEAD .claude/hooks tests` を scratchpad に展開した木を mutant ごとに作り、1 行だけを変えて、その木の `tests/test-pre-bash-guard.sh` を丸ごと回した(テストは自分の場所から `REPO_ROOT` を決めるので写しを試す)。変更は 1 行に当たることを確かめた。対照(変更なし)は 2,068/0。

| Mutant | 変更 | 結果 | 赤になった検査と行 |
| --- | --- | --- | --- |
| (a) | `rules.awk` の `datacmd_list` の文字列の最後に `exec` | 2,061/7 | F 節の不変条件の検査(`DATACMD has: exec (a wrapper that cmd_pos steps past)`)、B 節の `exec >run.sh; echo 'sudo ls'` の 4 件(got none)、G 節の旧版との比較の 2 件(jq あり・なし。旧版は deny、新しい版は none の一覧に `exec >run.sh; echo 'sudo ls'` が入り、`intentional_fixes` と食い違う) |
| (b) | `datacmd_list` の最後に `for` | 2,067/1 | F 節の不変条件の検査(`DATACMD has: for (a reserved word)`)だけ |
| (c) | DATACMD を埋めるループの次の行に `DATACMD["exec"] = 1` を足す | 2,061/7 | (a) と同じ 7 件。不変条件の検査は実行時の DATACMD を読むので、`datacmd_list` の外で足した名前も捕まえる |
| (d) | `commands.awk` の `end_cmd` から `if (DCTX[ctx] && !data_first_ok(ctx)) NODATA = 1` を消す | 1,988/80 | B 節 19 行 × 4 = 76 件、D 節の `=echo sudo ls` の 2 件、G 節の比較の 2 件。B 節の 76 件に `if grep -q 'sudo ' file; then echo ok; fi` の 4 件(got none)が入る。ほかに `{ echo 'sudo ls'; } \| sh`、`for x in 1; do echo 'sudo ls'; done \| bash`、`exec >run.sh; echo 'sudo ls'`(後ろに `sh` のある行とない行)、`builtin exec >run.sh; …`、`env echo 'sudo ls'`、`command echo 'sudo ls'`、`x=1 echo 'sudo ls'`、`./echo 'sudo ls'`、`>out.txt; echo 'sudo ls'` など |

- (a) と (c) で赤になるのは、足した行と不変条件の検査と G 節の比較で、後ろに `sh run.sh` を置いた既存の `exec` の行は deny のまま。plan の Design decisions のとおり、既存の行だけでは DATACMD への `exec` の追加に気づけない
- (b) を捕まえるのは不変条件の検査だけで、テストの行の判定は 1 つも変わらない。`for x in 1; do …; done` は `;` で段に分かれ、`do …` と `done` の段の 1 語目は DATACMD にないので、許可リストはやはりデータ区間を落とす。おそらくこれが理由だが、段ごとの確認はしていない
- (d) で `if grep -q 'sudo ' file; then echo ok; fi` が赤になるので、この行を止めているのは許可リストだけ(AC5b の最後の文)

### ほかの awk

Docker の `ralph-guard-awk:test`(ubuntu:24.04 に jq、gawk、mawk を入れ、`/opt/mawk-bin/awk` と `/opt/gawk-bin/awk` の symlink を置いた image。PR #214 の test で作ったもの)に `git archive HEAD .claude/hooks tests` を渡し、それぞれの bin を PATH の先頭にしてテストを回した。`command -v awk` は `/opt/mawk-bin/awk`(実体 `/usr/bin/mawk`)と `/opt/gawk-bin/awk`(実体 `/usr/bin/gawk`)。jq のない経路の行(`[no-jq]`)は 1,028 件ずつ走った。

- mawk 1.3.4 20240123: 2,068/0/0、14 秒
- gawk 5.2.1: 2,068/0/0、17 秒

どちらでも不変条件の検査は 20 names、9 wrappers、21 reserved words で PASS。

## Test gaps

- `trailing_backslashes` の数え始めの下限(m1)をテストスイートが押さえていない。バックスラッシュだけでできた本文の行(`\\` の 2 個なら継がず、1 個か 3 個なら継ぐ)の扱いを変えても 2,068 件は通る。base の `read_body` も同じ下限(`q >= ls`、`q >= p2`)だったので、この PR で生じた穴ではない。この PR の判定が変わっていないことは J の行の比較で確かめた。足すなら D 節に `cat <<EOF` + 改行 + `\\` + 改行 + `EOF` + 改行 + `git commit -n -m x` + 改行 + `EOF`(deny)と、その 3 個版(none)の 2 行で m1 が赤になる(J の行の 24 番と 34 番に当たる)。plan の Non-goals はテストの穴を外しているので、follow-up の候補として書く
- H の行の一致は、本文の継ぎの扱いについては何も言えない(比較の感度の節)。継ぎの扱いの一致は J の 280 行が支える
- 比べた入力は、テストの行、probe の行、上に書いた生成の行だけ。ほかの形は探していない(guard の test で新しい形を探さないという作業の範囲の取り決めのとおり)
- 不変条件の検査が包みの名前を `cmd_pos` の `nm == "…"` から読むので、包みの処理を別の関数に移すと読む名前が黙って減る(verify report の Coverage gaps と同じ。試していない)
- guard は main のチェックアウトから動くので、この branch の guard が session で効くことはマージ前には確かめられない

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 40 本 3,922 件、Go 8 packages)。依頼の 6 本は guard 2,068/0、archive-plan 22/0、check-template 60/0、lib-json 126/0、ralph-dispatch 33/0、verify-local-hook-tests 5/0。base と HEAD の判定の比較は 7,826 組(固有の payload 3,913、jq あり・なし)で違いが 0、J の 560 組でも 0。macOS、mawk、gawk のどれでも同じ。AC5b の 4 個の mutation はどれも依頼どおりの検査と行で赤になった。mawk と gawk でもテストは 2,068/0
- Fail: なし
- Blocked: なし
