# Test report: guard-wrapper-rows

- Date: 2026-10-10(実行は 22:25 から 22:50 JST まで)
- Plan: docs/plans/active/2026-10-10-guard-wrapper-rows.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: base 382c18c8 から HEAD f9127f47 まで。コードの変更は `tests/test-pre-bash-guard.sh` だけ(F 節の包みの行の検査と、`decide` から切り出した `run_guard`)。実行したのは `./scripts/run-test.sh`(changed-language scope の指定だが、`tests/test-pre-bash-guard.sh` を言語に分類できないので full fallback で走った)、AC2 の mutation 11 個(写しの木でテスト全体)、`run_guard` の切り出しの前後の比較(base のテストのファイルを HEAD の guard で実行)、mawk と gawk での実行。追跡しているファイルは変えていない(写しの木はすべて scratchpad の中)
- Evidence: `docs/evidence/test-2026-10-10-guard-wrapper-rows.log`(`run-test.sh` の出力。gitignore の対象なのでコミットしない)。スクリプトとログは scratchpad の `twr/`(`mkmut.sh`、`run-docker.sh`、`run-mac.sh`、`out/<木>.<awk>.out`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(shell 40 本 + Go、rc 0) | 40 suites / 3,972 assertions | 40 suites | 0 | 1(下のほかの shell の行) | 7 分 9 秒 |
| `tests/test-pre-bash-guard.sh`(macOS、BWK awk 20200816、bash 3.2.57、jq 1.8.2 あり・なし) | 2,108 | 2,108 | 0 | 0 | run-test の中 |
| `tests/test-lib-json.sh` | 126 | 126 | 0 | 0 | run-test の中 |
| `tests/test-ralph-dispatch.sh` | 33 | 33 | 0 | — | run-test の中 |
| `tests/test-check-template.sh` | 60 | 60 | 0 | — | run-test の中 |
| ほかの shell 36 本 | 1,645 | 1,645 | 0 | 1(`test-secret-scan-branch.sh` が、git 2.41 未満の実機の場合を手元の git 2.49.0 では飛ばす。stub の古い git の場合は通る) | run-test の中 |
| `go test ./...`(run-test の golang verifier) | 8 packages | 8 | 0 | — | `internal/org` 20.9 秒は実行、残り 7 つは cached(Go のファイルは変わっていない) |
| HEAD のテスト、Docker `ralph-guard-awk:test`(ubuntu 24.04、bash 5.2.21、jq 1.7)の mawk 1.3.4 20240123 | 2,108 | 2,108 | 0 | 0 | 16 秒 |
| HEAD のテスト、同じ image の gawk 5.2.1 | 2,108 | 2,108 | 0 | 0 | 22 秒 |
| base 382c18c8 のテストのファイル + HEAD の guard、macOS BWK / mawk / gawk | 2,090 × 3 | 2,090 × 3 | 0 | 0 | 222 秒(mutation と並行)/ 17 秒 / 19 秒 |
| AC2 の mutation 11 個(写しの木でテスト全体)、macOS BWK / mawk / gawk | 33 runs | — | 33 runs すべて赤 | — | macOS 1 本 197〜224 秒(3 本ずつ並行、load 約 8〜9)、Docker 1 本 15〜22 秒 |
| 追加: `WRAPPER` から名前を 1 つ消す mutation 9 個、Docker mawk | 9 runs | — | 9 runs すべて赤 | — | 1 本 14〜16 秒 |

件数の数え方: guard のテストの 2,108 件は、base の 2,090 件に新しい検査の 18 件(9 つの名前 × jq あり・なし)を足した数。assertion の総数 3,972 は、各 suite の `PASS` の行を数えた和(summary の `PASS: <数>` の行は除いた)。この PR が変えたテストのファイルは guard のテストだけで、ほかの 39 本のテストのファイルは base と同じ。`run-test.sh` は `HARNESS_VERIFY_MODE=test` の `run-verify.sh` なので、この rc 0 が AC4 の test mode の実行にあたる。

新しい検査の 18 件の中身(macOS、mawk、gawk で同じ): 8 つの名前は `W sh -c 'git commit -n -m x'` が deny(exit 0)で `W 5 sh -c …` が none、`timeout` だけ逆(`timeout sh -c …` が none、`timeout 5 sh -c …` が deny)。jq あり・なしで同じ。plan の Assumptions のとおり、どの名前も 2 形のちょうど一方が deny になる。F 節の DATACMD の検査は `(20 names, 9 wrappers, 21 reserved words)` で PASS。H 節の 3 件は macOS で 0.30 秒、1.21 秒、0.67 秒(上限 10 秒)。

## Coverage

- Statement: 計測なし(shell と awk の計測の道具はない)
- Branch: 計測なし
- Function: 計測なし
- Notes: 行の網羅率の代わりに、新しい検査が赤になるべき写し 11 個(AC2)で赤になることと、`run_guard` の切り出しの前後で同じ記録が出ること(下の「`run_guard` の切り出し」)で確かめた。Go のテストは `internal/org` だけが実行され、ほか 7 つは cached。この PR は Go のファイルを変えていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `WRAPPER` に名前を足して `cmd_pos` に分岐を足さないと、`edge_deny` に行を足し忘れた場合にテストが捕まえない(PR #217 で残した穴、tech-debt の (h)) | テストが捕まえる | 下の「Mutation」の A_add_chrt。`chrt` を一覧に足した写しで、新しい検査が `chrt` を名指しして jq あり・なしの 2 件とも FAIL。ほかの記録は変わらない |
| `cmd_pos` の分岐を 1 つ消すと、その包みの後ろの `sh -c` を読まなくなる | テストが捕まえる | 下の「Mutation」の B_* 9 個。どれも新しい検査がその名前で FAIL になる(D 節の手書きの行も FAIL になる) |
| `WRAPPER` が読めないと検査が黙って通る | テストが捕まえる | E_empty_list。新しい検査が「no WRAPPER name was read」で FAIL |
| `decide` の判定(`run_guard` に切り出した) | 変わらない | 下の「`run_guard` の切り出し」。base のテストのファイルは HEAD の guard で 2,090/0、PASS の記録の集合は HEAD の記録から新しい 18 件を除いたものと同じ |
| guard のファイルが変わっていない(AC3) | 変わっていない | `git diff --stat 382c18c8 HEAD -- .claude/ templates/` が空 |
| AC4 の test mode の `run-verify.sh` | rc 0 | `./scripts/run-test.sh` が rc 0、40 本すべて OK |

### Mutation(AC2)

木は HEAD の作業木から scratchpad にコピーした(`.claude/hooks/` を `lib_json.sh` ごと、`tests/` を `tests/fixtures/` ごと)。書き換えは写しの 1 行だけ。作るたびに、作業木と違うファイルが狙った 1 つだけであること(`cmp`、fixtures は `diff -rq`)と、3 つの `.awk` が parse できること(入力なしと `ls` で出力なし、`sudo ls` で `sudo`)を確かめた。対照の木(書き換えなし)は mawk と gawk で 2,108/0。

| Mutant | 変更 | 赤になった記録(どれも jq あり・なしの 2 件ずつ) | macOS BWK | mawk | gawk |
| --- | --- | --- | --- | --- | --- |
| A_add_chrt | `rules.awk` の `wrapper_list` の末尾に `chrt` を足す(分岐は足さない) | 新しい検査の `chrt`(FAIL 2)。2 形とも none(exit 0)。D 節の 9 行を含め、ほかは通ったまま | 2108/2 | 2108/2 | 2108/2 |
| E_empty_list | `wrapper_list` を `split("", …)` にする | F 節の DATACMD の検査(no WRAPPER name read)1 件、新しい検査(no WRAPPER name was read …, so no wrapper row ran)1 件、D 節の 24 件(包みの 9 行、`env -S`・`--split-string`、`echo x \| xargs sh -c …`)。FAIL 26 | 2065/26 | 2065/26 | 2065/26 |
| B_env | `cmd_pos` の `if (nm == "env") …` の行を消す(続く `else if (nm == "command")` は構文を保つため `if` にした) | 新しい検査の `env`、D 節の `env sh -c …`、`env -S …`、`env --split-string=…`(FAIL 8) | 2100/8 | 2100/8 | 2100/8 |
| B_command | `else if (nm == "command") …` の行を消す | 新しい検査の `command`、D 節の `command sh -c …`(FAIL 4) | 2104/4 | 2104/4 | 2104/4 |
| B_exec | 同じく `exec` の分岐 | 新しい検査の `exec`、D 節の `exec sh -c …`(FAIL 4) | 2104/4 | 2104/4 | 2104/4 |
| B_nohup | 同じく `nohup` の分岐 | 新しい検査の `nohup`、D 節の `nohup sh -c …`(FAIL 4) | 2104/4 | 2104/4 | 2104/4 |
| B_time | 同じく `time` の分岐 | 新しい検査の `time`、D 節の `time sh -c …`(FAIL 4) | 2104/4 | 2104/4 | 2104/4 |
| B_nice | 同じく `nice` の分岐 | 新しい検査の `nice`、D 節の `nice -n 5 sh -c …`(FAIL 4) | 2104/4 | 2104/4 | 2104/4 |
| B_stdbuf | 同じく `stdbuf` の分岐 | 新しい検査の `stdbuf`、D 節の `stdbuf -o0 sh -c …`(FAIL 4) | 2104/4 | 2104/4 | 2104/4 |
| B_timeout | 同じく `timeout` の分岐 | 新しい検査の `timeout`、D 節の `timeout 5 sh -c …`(FAIL 4) | 2104/4 | 2104/4 | 2104/4 |
| B_xargs | 同じく `xargs` の分岐 | 新しい検査の `xargs`、D 節の `xargs sh -c …`、`echo x \| xargs sh -c …`(FAIL 6) | 2102/6 | 2102/6 | 2102/6 |

表の `…` は `'git commit -n -m x'`(D 節の `env -S` などは `'git push origin --force'`)。11 個とも、FAIL の行は 3 つの awk でバイト単位で同じだった。

- A_add_chrt の FAIL の文言は、名前、経路、2 形の判定(`chrt sh -c … -> none (exit 0); chrt 5 sh -c … -> none (exit 0)`)を出し、続けて「`WRAPPER` にあって `cmd_pos` に分岐がないなら分岐を足す、分岐があるなら `wrapper_row_forms` に形を足す」と書く。対照と比べて変わった記録は、FAIL の 2 件と、DATACMD の検査の PASS の文言(`9 wrappers` が `10 wrappers` になった)だけ
- B_* の 9 個では、新しい検査が FAIL にしたのは消した名前の 2 件だけで、ほかの 8 つの名前の 16 件は PASS のまま。消した名前の 2 形はどれも none(exit 0)
- E_empty_list の記録は 2,091 件(2,090 件に、名前ごとの 18 件の代わりの 1 件を足した数)。対照の PASS の記録のうち 43 件(新しい 18 件、D 節の 24 件、DATACMD の検査 1 件)が消え、ほかは同じ

追加で、`WRAPPER` から名前を 1 つずつ消した写し 9 個(L_*、Docker の mawk)も回した。どれも赤で、赤になったのは D 節の手書きの行だけ(env 6、xargs 4、ほか 7 つは 2)。新しい検査はどの写しでも 16 件(残りの 8 つの名前 × jq あり・なし)が PASS し、消した名前の記録は出ない。テストのコメントが書く役割分担(名前が一覧から消えたことは D 節の 9 行が捕まえ、一覧の名前に分岐があることは新しい検査が確かめる)のとおりに動いた。

### `run_guard` の切り出し

- base 382c18c8 の `tests/test-pre-bash-guard.sh` を HEAD の木(guard と fixtures は HEAD のもの)に置いて実行すると、macOS BWK、mawk、gawk の 3 つとも 2,090/0
- 節ごとの PASS の数は、base のテストが A 476、B 752、C 116、D 628、E 54、F 50、G 7、H 3、I 4(計 2,090)、HEAD のテストが同じで F だけ 68(計 2,108)。3 つの awk と、run-test の中の macOS の実行で同じ数
- PASS の記録の行(H 節の秒数だけ伏せた)を並べて比べると、base の 2,090 行はすべて HEAD にあり、HEAD にしかない行は 18 行で、どれも新しい検査の記録だった(macOS と mawk で確かめた)。キューに入れたどの場合も、切り出しの前後で同じ期待どおりの判定になった

## Test gaps

- 新しい検査は、実行時に読んだ `WRAPPER` の名前しか見ない。名前が一覧から消えたことは、D 節の 9 行だけが捕まえる(L_* の 9 個で確かめた。plan の Non-goals と Design decisions のとおり)
- 新しい検査は、包みが自分の引数(`nice -n 5`、`stdbuf -o0`、`timeout 5`)をどう読むかを見ない。それは D 節の手書きの行が見る(テストのヘッダーの F 項目に書いてあるとおり)
- 分岐はあるのに引数の取り方が 2 形のどちらにも合わない包みを足すと、新しい検査は FAIL になる(plan の Risks)。この FAIL の経路は今の 9 つの包みでは通らず、mutation でも作っていない
- jq がないときに jq の経路の記録を SKIP にする分岐(`have_jq != yes`)は、どの実行でも jq があったので通っていない
- `cmd_pos` に分岐があって名前が `WRAPPER` にない形は、plan の Non-goals で、捕まえない
- awk は macOS の BWK 20200816、mawk 1.3.4、gawk 5.2.1 の 3 つ。busybox awk は見ていない
- マージ後に main のチェックアウトから動く guard の効き目は、マージ前には確かめられない(この PR は guard を変えていないので、効き目も変わらない)

## Verdict

- Pass: `./scripts/run-test.sh` が rc 0(40 本、3,972 件、SKIP 1 は git のバージョンによるもの)。`tests/test-pre-bash-guard.sh` は macOS BWK・mawk・gawk で 2,108/0。AC2 の mutation 11 個は 3 つの awk の 33 runs すべてで新しい検査が狙った名前で赤(`chrt` を足した写しは新しい検査の 2 件だけが赤)。base のテストは HEAD の guard で 2,090/0、節ごとの数と記録は新しい 18 件を除いて同じ。guard のファイルは変わっていない
- Fail: なし
- Blocked: なし
