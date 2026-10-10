# Test report: guard-awk-split

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-awk-split.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: base 0abfede5 → HEAD 3b6cfc9a。`./scripts/run-test.sh`(changed-language scope の指定だが、`.awk` を言語に分類できないので full fallback で走った)、base と HEAD の guard の判定の比較、AC5 の mutation、AC4 の fallback、mawk と gawk での実行。追跡しているファイルは変えていない(mutation と guard の写しはすべて scratchpad の中)
- Evidence: `docs/evidence/test-2026-10-09-guard-awk-split.log`(gitignore の対象なのでコミットしない)。使ったスクリプトは scratchpad の `tst/`(`extract.sh`、`extract2.sh`、`compare.sh`、`mkmut.sh`、`probe-mut.sh`、`docker-inner.sh`、`fallback.sh`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(shell 40 本 + Go) | 40 suites / 3,917 assertions | 40 suites | 0 | 1(下のほかの shell の行) | 9 分 34 秒(全体) |
| `tests/test-pre-bash-guard.sh`(macOS、BWK awk 20200816、jq あり・なし) | 2,063 | 2,063 | 0 | 0 | run-test の中 |
| `tests/test-check-template.sh` | 60 | 60 | 0 | 0 | run-test の中 |
| `tests/test-lib-json.sh` | 126 | 126 | 0 | 0 | run-test の中 |
| `tests/test-ralph-dispatch.sh` | 33 | 33 | 0 | 0 | run-test の中 |
| ほかの shell 36 本 | 1,635 | 1,635 | 0 | 1(`test-secret-scan-branch.sh` が、git 2.41 未満の実機の場合を手元の git 2.49 では飛ばす。stub の古い git の場合は通る) | run-test の中 |
| `go test ./...`(run-test の golang verifier) | 8 packages | 8 | 0 | — | `internal/org` 26.3 秒と `internal/scaffold` 0.47 秒は実行、残り 6 つは cached |
| S1(da3b55d2)の guard のテスト、macOS | 2,032 | 2,032 | 0 | 0 | 5 分 03 秒 |
| S1 の guard のテスト、Docker の mawk / gawk | 2,032 / 2,032 | 2,032 / 2,032 | 0 / 0 | 0 / 0 | 21 秒 / 20 秒 |
| HEAD の guard のテスト、Docker ubuntu:24.04 の mawk 1.3.4 | 2,063 | 2,063 | 0 | 0 | 16 秒 |
| HEAD の guard のテスト、Docker ubuntu:24.04 の gawk 5.2.1 | 2,063 | 2,063 | 0 | 0 | 23 秒 |
| base と HEAD の判定の比較(macOS、jq あり・なし) | 1,982 runs | 1,982 一致 | 0 | — | 3 分 33 秒 |
| mutation 9 個(macOS の行の probe、Docker の mawk と gawk で全件) | 9 × 3 | 9 × 3 が赤 | — | — | — |
| `.awk` が欠けた・読めない写しの手での確認 | 56 runs | 56 | 0 | — | — |

件数の数え方: guard のテストは 1 つの命令を jq の経路と jq のない経路で 2 回数え、B 節は permission_mode なしと bypassPermissions の 2 通りなので 1 行が 4 件になる。HEAD で足された 31 件は、B 節 group 11 の 3 行 × 4 = 12、D 節 `edge_none` の 5 行 × 2 = 10、F 節の `rules.awk` のない写しの 3 行 × 2 = 6、3 つの `.awk` の parse の 3 件で、2,032 + 31 = 2,063 と合う。

## Coverage

- Statement: 計測なし(shell と awk の計測の道具はない)
- Branch: 計測なし
- Function: 計測なし
- Notes: 変更の中心は判定を変えないファイルの移動なので、行の網羅率の代わりに、base の guard と HEAD の guard に同じ入力を渡して出力をバイト単位で比べた(下の Regression checks)。AC5 の行は mutation で効き目を確かめた。Go のテストは `internal/org` と `internal/scaffold` が実行され、ほか 6 つは cached(Go の cache は埋め込んだ `templates/` の中身も鍵に含むので、cached はこの木と同じ入力で前に通ったことを表す)。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 分割で判定が変わらないこと(base 0abfede5 の 1 ファイルの guard と HEAD の 4 ファイルの guard) | 一致 | base のテストの A〜D 節と G 節の行(固有の命令 575、permission_mode の違いを含む固有の payload 945)、PR #213 の probe の 17 行(`mp/l01.txt`〜`l17.txt`。13 行はテストの行と同じ payload)、参考に E 節(27)と F 節の入れ子と再読の上限の行(15)。合わせて固有の payload 991 を jq の経路と jq のない経路で両方の guard に通し、stdout のバイト列と exit status を比べた。1,982 runs のうち違いは 0。1,982 runs の HEAD の判定の内訳は deny 1,196、none 786、それ以外と exit 0 以外は 0。HEAD の jq の経路と jq のない経路の出力も 991 件すべて同じ |
| S1 のコミットで、足す前の 2,032 件が通ること(AC3) | 通る | macOS 2,032/0、mawk 2,032/0、gawk 2,032/0 |
| awk がないとき、awk が exit 2 のときの旧版の 4 規則の fallback(F 節の既存の行) | 通る | `F. no awk on PATH` の 6 件と `F. awk exits 2` の 4 件がすべて PASS |
| PR #213 の test report で残った mutation M20 と M13 の系統 | 赤になる | 下の Mutation の表。赤になったのはどれも HEAD で足した行(と、B 節の行から作る G 節の比較)だけで、base からある行はどの mutant でも赤にならない。base のテストではこれらの mutant が残ることと合う |

### Mutation(AC5)

HEAD の 3 つの `.awk` を scratchpad に写し、写しの 1 行だけを変えた(どれも `diff` で 1 hunk、mawk・gawk・BWK awk で parse が rc 0)。各 mutant で、Docker(ubuntu:24.04)の mawk と gawk でテスト全体を回し、macOS の BWK awk では A〜G 節の全行(固有の 1,027 行、jq あり・なしで 2,054 runs)を期待する判定と比べた。3 通りとも赤になった行は同じだった。

| Mutant | 変更 | 赤になった行 | 件数(全体のテスト) |
| --- | --- | --- | --- |
| M20 | `rules.awk:430` の `RE_NOTFLAG` から `=` を外す(`[^~^+#]`) | B `echo $=arr['$(sudo ls)']`(deny → none) | 6(B の 4 件と、G の比較の jq・no-jq の 2 件) |
| M20b | `^` を外す(`[^~=+#]`) | B `echo $^arr['$(sudo ls)']` | 6(同上) |
| M20c | `+` を外す(`[^~=^#]`) | B `echo $+arr['$(sudo ls)']` | 6(同上) |
| M13 | `lex.awk:357` の `LD_EXP` から空白の条件を外す | D `git commit -m "costs $ 5; never sudo ls"`(none → deny) | 2 |
| M13b | 文字列の終わりの条件を外す | D `rg 'sudo ' foo$` | 2 |
| M13c | タブの条件を外す | D `$` のあとがタブの commit の行 | 2 |
| M13d | 改行の条件を外す | D `$` のあとが改行の commit の行 | 2 |
| MLDQ | `lex.awk:350`、二重引用符の外の `$"` で `LD_EXP = (c == DQ)`(`$"` だけ 1 にする) | D `git commit -m $"never sudo ls"` | 2 |
| M13e(参考、依頼の外) | 二重引用符の中の閉じる `"` の条件を外す | D `git commit -m "never sudo ls for 5$"` と `rg "foo$" 'sudo ' .`(どちらも base からある行) | 4 |

M13e は依頼の 4 条件に入っていない同じ式の残りの 1 条件で、base からある 2 行がすでに固定している。新しい行は作っていない。

### Fallback(AC4)

F 節の 3 件(`rules.awk` のない写し、jq あり・なし)と parse の 3 件は、run-test の中で PASS した。手でも、HEAD の `pre_bash_guard.sh`、`lib_json.sh`、3 つの `.awk` を scratchpad に写し、1 つを消した写し 3 つと、1 つを `chmod 000` にした写し 3 つを作って、4 つの命令を jq あり・なしで通した(macOS、一般ユーザー)。

| 写し | `echo 'never use sudo here'` | `ls` | `sudo ls` | `git commit --no-verify -m x` |
| --- | --- | --- | --- | --- |
| 3 つとも揃った写し(対照) | none | none | deny | deny |
| `lex.awk` がない | deny | none | deny | none |
| `commands.awk` がない | deny | none | deny | none |
| `rules.awk` がない | deny | none | deny | none |
| `lex.awk` が 000 | deny | none | deny | none |
| `commands.awk` が 000 | deny | none | deny | none |
| `rules.awk` が 000 | deny | none | deny | none |

jq の経路と jq のない経路で全部同じ結果で、exit status はすべて 0。6 つの写しで guard と同じ引数の awk を直接回すと、どれも rc 2(`awk: can't open file pre_bash_guard_<x>.awk`)で止まる。awk の経路では none の `echo 'never use sudo here'` が deny になり、awk の経路では deny の `--no-verify` の commit が none になるので、旧版の 4 規則が判定したと分かる。

### ほかの awk

Docker の ubuntu:24.04 に jq 1.7、mawk 1.3.4、gawk 5.2.1 を入れ、`/opt/mawk-bin/awk` と `/opt/gawk-bin/awk` をそれぞれ PATH の先頭に置いてテスト全体を回した(`command -v awk` がそれぞれの awk を指すことを確かめた。apt で gawk を入れると `/usr/bin/awk` が gawk に切り替わるので、PATH で選んだ)。HEAD は mawk 2,063/0/0、gawk 2,063/0/0、S1 は 2,032/0/0 と 2,032/0/0。`/bin/sh` は dash。

## Test gaps

- テストの F 節が自動で見るのは `rules.awk` がない写しだけ。`lex.awk` と `commands.awk` がない写しと、読めない(`chmod 000`)写しは、この report で手で確かめただけで、テストの行はない。読めない写しは root では再現しない(root は 000 のファイルも読める)ので、CI のコンテナで行にするなら一般ユーザーで回す必要がある
- base と HEAD の判定の比較は macOS の BWK awk だけで行った。mawk と gawk では、テスト全体が通ることを確かめただけ
- `.awk` の構文の検査は、テストの F 節の parse の 3 件だけ(static verify は `.awk` を見ない。verify report の V-4)。guard は awk の stderr を捨てるので、構文の誤りはテストの外では fallback に黙って落ちる
- Go のテストは 8 つのうち 6 つが cached。このブランチは Go のコードを変えていない
- legacy layout(v1)からの `ralph upgrade` で `.awk` が届くか、merge のあと main のチェックアウトで Claude Code が分割後の guard を使うかは、この /test では見ていない(verify report の Coverage gaps と同じ)

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 40 本 3,917 件、Go 8 packages)。guard のテストは macOS・mawk・gawk で 2,063/0、S1 では 2,032/0。base と HEAD の guard は 1,982 runs で出力が 1 件も違わない。M20・M20b・M20c、M13 の系統 4 つ、`$"` の mutant は、どれも足した行だけで赤になる。`.awk` が欠けた・読めない写しは 6 通りとも旧版の 4 規則の fallback になる
- Fail: なし
- Blocked: なし
