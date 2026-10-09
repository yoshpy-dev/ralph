# Test report: guard-msg-param-flag

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-msg-param-flag.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: `git diff origin/main...HEAD`(base c3a9242e、HEAD 6157d685)の振る舞いのテスト。変更されたコードは `.claude/hooks/pre_bash_guard.sh`(template の写しとバイト単位で同じ。`cmp` rc 0)と `tests/test-pre-bash-guard.sh`。静的解析は /verify の担当なので実行していない。mutation と probe の形は、テストの配列・plan・依頼にある形と、mutant を殺すための無害な none の行だけを使い、guard を避ける新しい形は作っていない
- Evidence: `docs/evidence/test-2026-10-09-guard-msg-param-flag.log`(`docs/evidence/*.log` は gitignore の対象なのでコミットしない。scratch は `scratchpad/tmf/`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(worktree、macOS、BSD awk。`RALPH_VERIFY_SCOPE=changed` は `.claude/hooks/pre_bash_guard.sh` が言語に分類されないので full に切り替わった) | shell 40 本で 3,881、Go 8 package | 3,881 / 8 package ok | 0 | 0 | 648 s(rc 0) |
| うち `tests/test-pre-bash-guard.sh` | 2,032 | 2,032 | 0 | 0 | — |
| うち `tests/test-lib-json.sh` | 126 | 126 | 0 | 0 | — |
| うち `tests/test-ralph-dispatch.sh` | 33 | 33 | 0 | 0 | — |
| `bash tests/test-lib-json.sh`(単独で再実行) | 126 | 126 | 0 | 0 | 5 s |
| `bash tests/test-ralph-dispatch.sh`(単独で再実行) | 33 | 33 | 0 | 0 | 13 s |
| guard のテスト、ubuntu:24.04、mawk 1.3.4、jq なし、`sh` は dash | 2,021 | 1,013 | 0 | 1,008(jq の経路) | 9 s |
| 同、mawk 1.3.4 + jq 1.7 | 2,032 | 2,032 | 0 | 0 | 19 s |
| 同、gawk 5.2.1 + jq 1.7 | 2,032 | 2,032 | 0 | 0 | 21 s |
| HEAD のテストを base の guard(c3a9242e)に当てた(修正を戻した mutation、macOS) | 2,032 | 1,958 | 74 | 0 | 514 s |
| 同、ubuntu の mawk + jq | 2,032 | 1,958 | 74 | 0 | — |

- Go は golang verifier の `go test ./...` で 8 package すべて ok。`internal/org` だけ 12.6 s で走り、残りは `(cached)`。この branch は Go のコードを変えていない。
- ubuntu の 3 回では、`readlink -f "$(command -v awk)"` で awk の実体を毎回確かめた(apt で gawk を入れると awk の alternative が切り替わるため)。H 節(約 200 KB のコマンド)は jq なしで 0.07〜0.76 s、macOS で 0.30〜1.15 s(上限 10 s)。
- shell の 40 本の内訳は evidence log の 1 節にある(例: secret-scan-branch 246、ralph-worktree 143、codex-exec-invocation 144、plan-visual 103)。`tests/test-secret-scan.sh`(123)は既知の flaky(固定の `/tmp` のパス)だが、今回は落ちなかった。

## Coverage

- Statement: 計測なし(shell の計測ツールはない)
- Branch: 計測なし
- Function: 計測なし
- Notes: テストの件数と mutation で見た。
  - この PR が足した行は 29 行で、1,948 件から 84 件増えて 2,032 件になった。B 節 group 11 の 13 行(2 モード × jq あり・なしで 52 件)、D 節 `edge_deny` の 3 行(6 件)、`edge_none` の 9 行(18 件)、`edge_sentinel_deny` の 4 行(8 件)。
  - A〜D 節の 580 行を 1 行 1 ファイルに書き出して HEAD に当てると、期待と違う行は 0 だった(jq あり・なし)。base に当てると 23 行が変わり、どれもこの PR が足した行だった(下の Regression checks)。
  - mutation は 22 個で、16 個が赤になった。残った 6 個のうち、1 個は等価、4 個は止める側にしかずれない形、1 個は穴の側(M20)。詳しくは下の「Mutation」。

### AC5

| 条件 | 結果 | 根拠 |
| --- | --- | --- |
| `bash tests/test-pre-bash-guard.sh` が通る | Met | run-test の中で 2,032/0/0。ubuntu の mawk・gawk(jq あり)でも 2,032/0、jq なしは 1,013/0/1,008 skip |
| `bash tests/test-lib-json.sh` が通る | Met | 126/0/0(run-test の中と単独の 2 回) |
| `./scripts/run-verify.sh` が rc 0 | Met(2 回に分けて実行) | `run-test.sh` は `run-verify.sh` を `HARNESS_VERIFY_MODE=test` で動かすもので、rc 0。`HARNESS_VERIFY_MODE=static` の半分は verify の報告の `./scripts/run-static-verify.sh` rc 0 による。/test は静的解析を動かさない取り決めなので、既定の mode all の 1 回では実行していない |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | HEAD ではどの suite も失敗しなかった。base に当てた 74 件の失敗は、修正を戻したときに赤になるかの確認で、期待どおりの失敗(下) | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| PR #211 の止めすぎ: 単一引用符・`\${`・`$'…'` で `${` を書き、見張りの語も含むメッセージ(コミット 3 形、タグ 1 形)が deny | 直った | `edge_none` の 4 行。base deny、HEAD none(jq あり・なし) |
| PR #210 の self-review の C4R-1: `rg 'foo$' 'sudo ' .` が deny | 直った | `edge_none` の `rg 'foo$'` と `rg "foo$"`。base deny、HEAD none |
| rg の変数の語(`rg $x`、`rg "$x"`、`rg $(echo --pre)`、バッククォート)がデータになる穴 | 直った | B 節 group 11 の 4 行。base none、HEAD deny |
| zsh の添字(`$arr[`、`$h[`、`$~arr[`、`$[`、`$@[`、`$#x[`、`$*[`、二重引用符の中で開く添字、`-m $arr[…]`)の穴 | 直った | B 節 group 11 の 9 行。base none、HEAD deny |
| 数字・特別なパラメータ・入れ子の添字(`rg $1`、`rg "$@"`、`-m "use $1 …"`、`$a[$b[1]]'…'`) | 直った | `edge_sentinel_deny` の 4 行。base none、HEAD deny |
| PR #211 が止めた `${(e)…}` のメッセージ(`-m`、`--message''=`、`-"m"`、タグ) | 保たれている | B 節 group 10 の 5 行は HEAD でも deny。M02・M07・M19 で赤になる(下) |
| AC2b: 添字の中の置換は読み直す(`git commit -m $arr[$(date)]`、`echo $arr[$(s''udo ls)]`) | 保たれている | `edge_deny` の 2 行。base も HEAD も deny |
| 閉じない `[` が `;` の後ろを飲み込まない(`echo $a[ ; git push origin --force`) | 保たれている | `edge_deny`。base も HEAD も deny |
| AC1 の残り(`rg -n 'sudo ' .`、`git commit -m 'use ${HOME}'`、`git commit -m "never sudo ls for 5$"`) | 保たれている | `edge_none`。base も HEAD も none |
| AC3 の 29 形と G 節の `intentional_fixes` の 13 件 | 保たれている | C 節と G 節は HEAD ですべて PASS。base では G 節の 2 件(jq あり・なし)が落ち、「旧版 deny で none」の一覧に group 11 の 13 行が加わる |
| 既存の 1,948 件 | 保たれている | base に当てたときの 74 件の失敗は、足した 23 行の 72 件と G 節の 2 件だけ。既存の行は base でも HEAD でも通る |

### Mutation

guard の写し(`lib_json.sh` を同じディレクトリに置いた)を scratch で書き換え、テストをまるごと流した(ubuntu の mawk + jq、`GUARD_TEST_JOBS=6`)。追跡しているファイルは変えていない。FAIL の件数は、行の件数と G 節の 2 件の合計。

| Mutant | 変更 | FAIL | 赤にした行 |
| --- | --- | --- | --- |
| base | 修正を戻す(c3a9242e の guard) | 74 | 足した 23 行と G 節 |
| M01(plan: 印を立てない) | `lex_dollar` の `LD_EXP = !(…)` を 0 に | 20 | `rg $x`、`rg "$x"`、`-m $arr[…]`、`rg $1`、`rg "$@"`、`-m "use $1 …"`、G 節 |
| M19 | `lex_cmds` の `add_word` に `LW_EXP` の代わりに 0 を渡す | 40 | M01 の行と `${(e)…}` の 5 行 |
| M07 | `${` の分岐の `LD_EXP = 1` を 0 に | 22 | `${(e)…}` のメッセージ 5 行 |
| M02(plan: msg_check で印を見ない) | `if (WEXP[ctx, j] && …)` を `if (0 && …)` に | 28 | `${(e)…}` の 5 行、`-m $arr[…]`、`-m "use $1 …"` |
| M03(plan: rg で印を見ない) | rg の条件から `WEXP` を外す | 14 | `rg $x`、`rg "$x"`、`rg $1`、`rg "$@"` |
| M04(plan: rg で WS を見ない) | rg の条件から `WS` を外す | 6 | バッククォートの rg の 1 行 |
| M06 | rg の条件から `WANSI` を外す | 8 | `rg $'\x2d-pre'`、`rg $"sudo ls" .` |
| M05(plan: 添字の xnote を外す) | `if (subp) xnote(…)` を `if (0)` に | 36 | 添字の 8 行と `$a[$b[1]]` |
| M14 | `lex_word` で `LD_SUB` を読まない | 32 | 引用符の外の添字 7 行と `$a[$b[1]]` |
| M15 | `lex_dq` で `LD_SUB` を読まない | 6 | 二重引用符の中で開く添字の 1 行 |
| M09 | `lex_word` で `DQ_SUB` を読まない | 6 | 同じ 1 行 |
| M08 | `lex_word` で `DQ_EXP` を読まない | 10 | `rg "$x"`、`rg "$@"`、`-m "use $1 …"` |
| M10 | 特別なパラメータの分岐を外す | 10 | `$@[`、`$*[` |
| M11 | `RE_NOTFLAG` から `#` を外す | 6 | `$#x[` の 1 行 |
| M12 | `LD_EXP` の「二重引用符の中で閉じる `"`」の条件を外す | 4 | `edge_none` の `-m "never sudo ls for 5$"`、`rg "foo$"` |
| M18 | `$'…'` の分岐で `LD_EXP = 1` | 2 | `edge_none` の `-m $'mention ${HOME}; …'` |
| M17 | `$(` の分岐の `LD_EXP = 1` を 0 に | 0 | 等価。`WEXP` を読むのは rg の条件(`WS` も見る)と `msg_check`(先に `WS` を見て、HEREDOC の形でなければそこで戻る。HEREDOC の形なら `WEXP` の条件も `!safe_heredoc_msg` で偽)の 2 か所だけで、`$(` はいつも `LD_SUBST` を立てて `WS` を 1 にする |
| M13 / M13b / M13c / M13d | `LD_EXP` の「後ろが空白」「文字列の終わり」「タブ」「改行」の条件を 1 つずつ外す | 0 | 残った。どれも `$` を展開ありとみなす側(止める側)にずれる。下の無害な none の行で、それぞれ 1 つだけ赤になることを probe で確かめた |
| M20 | `RE_NOTFLAG` から `=` を外す | 0 | 残った。穴の側にずれる。`echo $=arr['$(sudo ls)']` は M20 で none、HEAD で deny。テストに `$=`・`$^`・`$+` を含む行は 0 行(grep) |

plan の Test plan が挙げた 5 つ(印を立てない、`msg_check` で印を見ない、rg で印を見ない、rg で `WS` を見ない、`$name[` の `xnote` を外す)はすべて赤になった。M04・M09・M11・M15・M18 は 1 行だけで赤になる。

M13 の系統を赤にする none の行(probe。HEAD none/none、base none/none、旧版 deny/deny、jq あり・なし):

| 行 | 赤にする mutant |
| --- | --- |
| `git commit -m "costs $ 5; never sudo ls"` | M13 だけ |
| `git commit -m "costs $<タブ>5; never sudo ls"` | M13c だけ |
| `git commit -m "costs $<改行>5; never sudo ls"` | M13d だけ |
| `rg 'sudo ' foo$` | M13b だけ |

## Ad-hoc probes

verify の V-4 が「テストの配列にない」とした 3 形。判定は HEAD の root と template、base(c3a9242e)、旧版(`tests/fixtures/guard-1c4cea5a/`)で、jq あり/なし、permission_mode なしと `bypassPermissions` の両方(`scratchpad/tmf/probe4.sh`)。deny の理由はどれも sudo の文。

| 形 | HEAD(root・template、2 モード) | base | 旧版 |
| --- | --- | --- | --- |
| `echo $=arr['$(sudo ls)']` | deny/deny | none/none | deny/deny |
| `echo "$arr['$(sudo ls)']"` | deny/deny | deny/deny | deny/deny |
| `echo ${arr['$(sudo ls)']}` | deny/deny | deny/deny | deny/deny |

1 つ目は base の穴をこの PR が塞いだ形で、上の M20 を赤にできる唯一の形でもある。後の 2 つは plan の Objective 3 のとおり base でも deny。

## Test gaps

- **M20(穴の側の survivor)**: zsh の修飾の記号 `~ = ^ + #` のうち、テストで固定されているのは `~`(`$~arr[`)と `#`(`$#x[`)だけ。`=` を外した M20 はテストを通る。`^` と `+` も同じく固定する行がない(行の grep で 0 件。mutant は作っていない)。HEAD の判定は正しい(`echo $=arr['$(sudo ls)']` は deny)。B 節 group 11 に `echo $=arr['$(sudo ls)']` を足すと M20 が赤になる(旧版も deny なので B 節に置ける)
- **M13 の系統(止める側の survivor)**: 「展開しない `$`」の 5 つの条件のうち、テストで固定されているのは「二重引用符の中で閉じる `"`」(M12)だけ。外れても旧版と同じく止めるだけなので穴にはならないが、AC1 が直した止めすぎが一部戻っても気づけない。上の 4 行を D 節 `edge_none` に足すと 4 つとも赤になる
- 1 行だけで赤になる mutant が 5 つある(M04、M09、M11、M15、M18)。その行を消すか書き換えると、条件が固定されなくなる
- テストのファイルに行を足すのは依頼の範囲の外なので、この cycle では足していない。足すなら guard のテストを次に変えるとき、または cross-review の修正とまとめて行う
- zsh と bash での実際の評価は確かめ直していない(plan の Assumptions と self-review の `dqsub.zsh` による)
- macOS の awk(BSD)は run-test で、mawk と gawk は ubuntu で見た。busybox awk は見ていない
- この session で効いている guard は main のチェックアウトのものなので、merge 後に Claude Code で実際に効くかは確かめられない

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 40 本で 3,881 件、Go 8 package)。`tests/test-pre-bash-guard.sh` は 2,032/0(macOS の BSD awk、ubuntu の mawk と gawk。jq なしは 1,013/0/1,008 skip)。`tests/test-lib-json.sh` は 126/0、`tests/test-ralph-dispatch.sh` は 33/0。AC5 は Met(`run-verify.sh` の static の半分は verify の rc 0 による)。修正を戻すと 74 件が赤になり、plan の Test plan の 5 つの mutation もすべて赤になった
- Fail: なし
- Blocked: なし
- PR を止める失敗はない。Test gaps の M20 と M13 の系統は、テストの行を足せば固定できる。どちらも HEAD の判定は正しい
