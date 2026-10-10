# Test report: guard-test-gaps

- Date: 2026-10-10(実行は 15:09 から 15:59 JST まで)
- Plan: docs/plans/active/2026-10-10-guard-test-gaps.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: base 49ac046c から HEAD a1b57346 まで。`./scripts/run-test.sh`(changed-language scope の指定だが、`.awk` を言語に分類できないので full fallback で走った)、AC3 と AC4 の mutation(写しの木でテスト全体を実行)、AC5 の base と HEAD の判定の比較、mawk と gawk での実行。追跡しているファイルは変えていない(guard の写しと mutation の木はすべて scratchpad の中)
- Evidence: `docs/evidence/test-2026-10-10-guard-test-gaps.log`(gitignore の対象なのでコミットしない)。スクリプトは scratchpad の `tgg/`(`mkmut.sh`、`run-docker-suites.sh`、`run-mac-suites.sh`、`dump-queue.sh`、`gen-wrap-edges.sh`、`build-cmp.sh`、`run-cmp-mac.sh`、`docker-cmp-inner.sh`、`docker-sens-inner.sh`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(shell 40 本 + Go、rc 0) | 40 suites / 3,944 assertions | 40 suites | 0 | 1(下のほかの shell の行) | 5 分 58 秒 |
| `tests/test-pre-bash-guard.sh`(macOS、BWK awk 20200816、jq あり・なし) | 2,090 | 2,090 | 0 | 0 | run-test の中 |
| `tests/test-archive-plan.sh` | 22 | 22 | 0 | — | run-test の中 |
| `tests/test-check-template.sh` | 60 | 60 | 0 | 0 | run-test の中 |
| `tests/test-lib-json.sh` | 126 | 126 | 0 | 0 | run-test の中 |
| `tests/test-ralph-dispatch.sh` | 33 | 33 | 0 | — | run-test の中 |
| `tests/test-verify-local-hook-tests.sh` | 5 | 5 | 0 | — | run-test の中 |
| ほかの shell 34 本 | 1,608 | 1,608 | 0 | 1(`test-secret-scan-branch.sh` が、git 2.41 未満の実機の場合を手元の git 2.49.0 では飛ばす。stub の古い git の場合は通る) | run-test の中 |
| `go test ./...`(run-test の golang verifier) | 8 packages | 8 | 0 | — | `internal/org` 13.0 秒は実行、残り 7 つは cached(Go のファイルは変わっていない) |
| HEAD の guard のテスト、Docker `ralph-guard-awk:test`(ubuntu 24.04)の mawk 1.3.4 20240123 | 2,090 | 2,090 | 0 | 0 | 15 秒 |
| HEAD の guard のテスト、同じ image の gawk 5.2.1 | 2,090 | 2,090 | 0 | 0 | 18 秒 |
| AC3・AC4 の mutation 19 個(写しの木でテスト全体)、macOS BWK / mawk / gawk | 57 runs | — | 57 runs すべて赤 | — | macOS 1 本 92〜315 秒(比較と並行)、Docker 1 本 14〜21 秒 |
| base と HEAD の判定の比較(macOS、jq あり・なし) | 4,347 payloads / 17,388 runs / 8,694 組 | 8,694 組が一致 | 0 | — | 14 分 43 秒 |
| 同じ比較、Docker の mawk / gawk | 8,694 組 / 8,694 組 | どちらも一致 | 0 / 0 | — | 69 秒 / 81 秒 |
| 比較の感度を確かめる mutant 3 個(Docker の mawk、jq あり・なし) | 8,694 組 × 3 | — | — | — | 129 秒 |

`run-test.sh` の中で、不変条件の検査は `PASS  F. DATACMD invariant: no name in DATACMD is a reserved word or a name in WRAPPER (both lists read at run time from the three .awk files) (20 names, 9 wrappers, 21 reserved words)`。D 節に足した 11 行は jq あり・なしの 22 件とも期待どおりで PASS(包みの 9 行は deny、バックスラッシュ 2 個の行は deny、3 個の行は none)。H 節(約 200 KB の命令)の 3 件は 0.27 秒、1.05 秒、0.63 秒で、上限の 10 秒より十分短い。AC7 の test mode の `run-verify.sh` は、`run-test.sh` が `HARNESS_VERIFY_MODE=test` の `run-verify.sh` なので、この rc 0 で満たす。

件数の数え方: guard のテストの 2,068 件から増えた 22 件は、D 節の 11 行 × jq あり・なし。F 節の検査は 1 件のまま(読み方が変わっただけ)。shell 全体の 3,944 件は前回(guard-debt-cleanup の /test)の 3,922 件にこの 22 件を足した数。

## Coverage

- Statement: 計測なし(shell と awk の計測の道具はない)
- Branch: 計測なし
- Function: 計測なし
- Notes: 行の網羅率の代わりに 2 つで確かめた。1 つは、`WRAPPER` の 9 つの要素と `cmd_pos` の 9 つの分岐を 1 つずつ消した写しで、テストが赤になること(AC3)。もう 1 つは、base と HEAD の guard に同じ入力を渡し、出力をバイト単位で比べること(AC5)。比べた入力が包みの判定とヒアドキュメントの継ぎの変化を見分けられることは、わざと壊した写し 3 つで確かめた(下の「比較の感度」)。Go のテストは `internal/org` だけが実行され、ほか 7 つは cached。この PR は Go のファイルを変えていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 判定が変わらないこと(AC5、base 49ac046c と HEAD a1b57346 の guard) | 一致 | 下の「判定の比較」。固有の payload 4,347 を jq の経路と jq のない経路で両方の guard に渡し、17,388 回の実行の 8,694 組で stdout のバイト列と終了コードが一致(stderr も一致)。Docker の mawk と gawk でも同じ 8,694 組が一致 |
| `WRAPPER` の要素と `cmd_pos` の分岐の片方を消すと、包みの後ろの `sh -c` を読まなくなること((h)、AC3) | テストが捕まえる | 下の「Mutation」。18 個とも、macOS・mawk・gawk の 3 つの awk で赤 |
| `trailing_backslashes` の下限を変える mutation をテストが殺せないこと((i)、AC4) | テストが捕まえる | 下限を `q > a` にした写しで、足した 2 行が jq あり・なしの 4 件とも赤(3 つの awk で同じ) |
| AC7 の test mode の `run-verify.sh` | rc 0 | `./scripts/run-test.sh` が rc 0、40 本すべて OK |

### Mutation(AC3、AC4)

木は HEAD の作業木から scratchpad にコピーし(`.claude/hooks/`、`tests/test-pre-bash-guard.sh`、`tests/fixtures/`)、1 か所だけ書き換えた。作るたびに、作業木と違うファイルが狙った 1 つだけであること(`cmp`)と、3 つの `.awk` が parse できること(入力なしと `ls` で出力なし、`sudo ls` で `sudo`)を確かめた。対照の木(書き換えなし)は作業木と `diff -r` で一致し、mawk と gawk で 2,090/0。

| Mutant | 変更 | 赤になった行(jq あり・なしの両方) | macOS BWK | mawk | gawk |
| --- | --- | --- | --- | --- | --- |
| L_env | `wrapper_list` から `env` を消す | `env sh -c '…'`、`env -S 'git push origin --force'`、`env --split-string='git push origin --force'`(3 行 × 2 = FAIL 6) | 2084/6 | 2084/6 | 2084/6 |
| L_command | 同じく `command` | `command sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| L_exec | 同じく `exec` | `exec sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| L_nohup | 同じく `nohup` | `nohup sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| L_time | 同じく `time` | `time sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| L_nice | 同じく `nice` | `nice -n 5 sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| L_stdbuf | 同じく `stdbuf` | `stdbuf -o0 sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| L_timeout | 同じく `timeout` | `timeout 5 sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| L_xargs | 同じく `xargs` | `xargs sh -c '…'`、`echo x \| xargs sh -c 'git push origin --force'`(FAIL 4) | 2086/4 | 2086/4 | 2086/4 |
| B_env | `cmd_pos` の `if (nm == "env") …` の行を消す(続く `else if (nm == "command")` は構文を保つため `if` にした) | L_env と同じ 3 行(FAIL 6) | 2084/6 | 2084/6 | 2084/6 |
| B_command | `else if (nm == "command") …` の行を消す | `command sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| B_exec | 同じく `exec` の分岐 | `exec sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| B_nohup | 同じく `nohup` の分岐 | `nohup sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| B_time | 同じく `time` の分岐 | `time sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| B_nice | 同じく `nice` の分岐 | `nice -n 5 sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| B_stdbuf | 同じく `stdbuf` の分岐 | `stdbuf -o0 sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| B_timeout | 同じく `timeout` の分岐 | `timeout 5 sh -c '…'`(FAIL 2) | 2088/2 | 2088/2 | 2088/2 |
| B_xargs | 同じく `xargs` の分岐 | L_xargs と同じ 2 行(FAIL 4) | 2086/4 | 2086/4 | 2086/4 |
| T_lower_bound(AC4) | `trailing_backslashes` の `while (q >= a && at(q) == BS)` を `q > a` に | バックスラッシュ 2 個の行が deny → none、3 個の行が none → deny(2 行 × 2 = FAIL 4) | 2086/4 | 2086/4 | 2086/4 |

表の `'…'` は `'git commit -n -m x'`。macOS の 19 本は判定の比較と並行して回したので 1 本 92〜315 秒かかった。赤の行は、19 個のどれでも 3 つの awk で同じ集合だった(FAIL の行を並べて比べた)。包みの mutant 18 個で赤になったのは、どれも消した包みの行だけで、ほかの包みの行は deny のまま。`env` と `xargs` は、もとからある D 節の行(`env -S`・`--split-string`、`echo x | xargs sh -c …`)も赤にする。消した包みは命令の名前として扱われ、後ろを読まなくなるので、`env` の `-S` の読み直しと `xargs` の後ろの `sh -c` も働かない。

### 判定の比較

guard の写しは、base が `git show 49ac046c:` の `pre_bash_guard.sh`・3 つの `.awk`・`lib_json.sh`、HEAD が `git show HEAD:` の同じ 5 ファイル(作業木のファイルと `cmp` で一致することを確かめた)。jq のない経路は、テストと同じ道具の一覧(`sh bash dash cat grep sed printf dirname env tr command test awk`)だけを symlink した PATH で、そこから jq が見えないことを確かめた。macOS の jq は 1.8.2、Docker は 1.7。

入力(固有の payload 4,347。payload のバイト列の sha256 で重ねを除いた)

- W: PR #215 の tester の比較の入力(`scratchpad/gdc/cmpw/p`)3,913。節ごとに A 238、B 366、C 53、D 301、E 27、F 15、H 2,580、P 5、R 238、X 90
- J: PR #215 の tester の J の行(`gdc/cmpj/p`)280。ヒアドキュメントの本文で、バックスラッシュ 0〜3 個で終わる行のあとに `EOF` を置き、そのあとに `git commit -n -m x` と `EOF` を続けた形。J は W の 3,913 件とは別の集まりで、W と J を足すと 4,193
- QB: 49ac046c のテストが本物の guard に渡す行のすべて。テストを「Run the queue」の前で切り、作った queue のうち `q_hook` が `$HOOK` のものを書き出した(2,044 回、固有の payload 1,001。1,000 は W と同じ)
- QH: HEAD のテストで同じことをした(2,066 回、固有の payload 1,012)。QB の 1,001 に、足した 11 行の 11 を足した数。11 のうちヒアドキュメントの 2 行は J の行と同じ payload、包みの 9 行は下の E の S と同じ payload
- E: 包みの判定(`cmd_pos` の `WRAPPER` の入口)の端の入力 153。どれも後ろに `sh -c 'git commit -n -m x'` を付けた。S は包み 1 つ(9、D 節の行と同じ)。W は包みを 2 つ続けた形で、9 つの包みの順序付きの組すべて(81。依頼の `env nohup sh -c …` と `nice -n 5 timeout 5 sh -c …` を含む)。A は包みの前の代入で、`X=1` と `X=1 Y=2`(18。依頼の `X=1 env sh -c …` を含む)。Q は引用符付きの名前で、`"env"`・`'env'`・`\env` の 3 通り(27。依頼の `"env" sh -c …` を含む)。P は `/usr/bin/<名前>`(9。依頼の `/usr/bin/env sh -c …` を含む)。Z は zsh の `=<名前>`(9)。`nice` は `-n 5`、`stdbuf` は `-o0`、`timeout` は `5` の引数を付けた

結果

- macOS(BWK awk): 17,388 回の実行の 8,694 組で、stdout のバイト列か終了コードが違う組は 0、stderr が違う組も 0。HEAD の判定は deny 6,336、none 2,358、それ以外の出力と 0 以外の終了コードは 0。HEAD の jq の経路と jq のない経路の stdout の違いも 0
- Docker の mawk 1.3.4 と gawk 5.2.1: 同じ 8,694 組で、違う組はどちらも 0、stderr の違いも 0、HEAD の deny はどちらも 6,336(macOS と同じ数)
- E の 153 行は、HEAD で jq あり・なしとも deny(S 9、W 81、A 18、Q 27、P 9、Z 9 のすべて)。base も同じ(違い 0 件)。依頼の 5 形も deny/deny

### 比較の感度

0 件という結果が、包みの判定とヒアドキュメントの継ぎの変化を見分けられる入力から出たものかを確かめるため、上の mutant の写しから 3 つを選び、base と同じ入力で比べた(Docker の mawk、jq あり・なし)。

| Mutant | base と違う組 | 違う payload の出どころ |
| --- | --- | --- |
| L_timeout(`WRAPPER` に `timeout` がない) | 50(25 payload × 2) | E の 24 と、D 節の `timeout 5 sh -c …` の行(QH と E の S が同じ payload)。`timeout` を含む E の行の数(S 1、W 17、A 2、Q 3、P 1、Z 1)とちょうど合う |
| B_nice(`cmd_pos` に `nice` の分岐がない) | 50(25 payload × 2) | 同じく `nice` を含む E の 24 と、D 節の `nice -n 5 sh -c …` の行 |
| T_lower_bound(下限が `q > a`) | 60(30 payload × 2) | J の 28 と、D 節に足したヒアドキュメントの 2 行(J と同じ payload)。PR #215 の /test の m1 の 30 と同じ数 |

3 つとも、壊した所が判定に効く入力だけで違いが出た。base と HEAD の違い 0 件は、包みの入口とヒアドキュメントの継ぎについて見分ける力のある入力から出た結果といえる。引用符付き・パス付き・`=` 付きの名前も L_timeout で違いが出たので、これらの書き方も `cname` を通って `WRAPPER` の判定に届いている。

## Test gaps

- `WRAPPER` に名前を足して `cmd_pos` に分岐を足さない(またはその逆の)、10 個目の包みのずれは、その包みの行がなければテストに捕まらない。今の 9 組は D 節の 9 行が押さえる。新しい包みに行を足すことは `rules.awk` の BEGIN のコメントと tech-debt の Trigger 列が求めているだけ(self-review と verify の Coverage gaps と同じ)
- テストの H 節(約 200 KB の命令 3 つ)は queue に入らないので、判定の比較の入力に入れていない。HEAD ではテストの中で 3 件とも deny で PASS
- awk は macOS の BWK 20200816、mawk 1.3.4、gawk 5.2.1 の 3 つ。busybox awk は見ていない
- E の端の入力は、包みの入口の判定を通る形だけで、いずれも deny。包みでない名前の後ろの `sh -c` などの none の形は、W と QB/QH にある既存の行(none 2,358)が比較している
- マージ後に main のチェックアウトから動く guard の効き目は、マージ前には確かめられない

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 40 本 3,944 件、Go 8 packages)。依頼の 6 本はすべて通り、`tests/test-pre-bash-guard.sh` は 2,090/0(macOS、mawk、gawk)。AC3 の 18 個と AC4 の 1 個の mutation は、3 つの awk のすべてで赤。AC5 の base と HEAD の比較は 3 つの awk で違い 0 件、感度の対照では違いが出る
- Fail: なし
- Blocked: なし
- Verdict: pass
