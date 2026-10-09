# Test report: guard-zsh-data-gaps

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-zsh-data-gaps.md(Test plan の節、AC1〜AC6。AC1 は 8 項目 11 形で、Test plan の「AC1 の 5 形」は古い記述)
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: `git diff origin/main...HEAD`(base 0931f791、HEAD d4c7efd4)と、この /test で `tests/test-pre-bash-guard.sh` に足した 1 行(8e7bfbae)。コードの変更は 139d6627(`lex_dollar` の `${…}` の `xnote`、`msg_check` の `WR` の `${` の検査、`lex_hd` の HSUB、DATACMD から `stat`、NOEXEC に `tr`)、e77e2937 はコメントだけ。guard と `lib_json.sh` は変えていない
- Evidence: `docs/evidence/test-2026-10-09-guard-zsh-data-gaps.log`(gitignore の対象なので commit しない。`run-test.sh` と `run-verify.sh` の出力、mutation の全結果と置換の一覧、probe、Docker と zsh の出力を入れた。scripts は `scratchpad/zg/ts1/`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(d4c7efd4、テストを足す前) | shell 40 本 + Go 8 パッケージ | 40 本 + Go 8 | 0 | 0 | 830 s(ほかの session の負荷あり) |
| うち `tests/test-pre-bash-guard.sh` | 1946 | 1946 | 0 | 0 | — |
| うち `tests/test-lib-json.sh` | 126 | 126 | 0 | 0 | — |
| `tests/test-pre-bash-guard.sh` のコピー(HEAD の guard + 足したあとのテスト) | 1948 | 1948 | 0 | 0 | 484 s(4 本並列) |
| ubuntu:24.04、mawk、jq なし: guard / lib_json | 1937 / 126 | 971 / 87 | 0 / 0 | 966 / 39(jq の経路) | — |
| ubuntu:24.04、mawk、jq 1.7: guard | 1948 | 1948 | 0 | 0 | — |
| ubuntu:24.04、gawk 5.2.1、jq 1.7: guard / lib_json | 1948 / 126 | 1948 / 126 | 0 / 0 | 0 / 0 | — |
| `./scripts/run-verify.sh`(8e7bfbae、mode all、scope full) | 静的な検査 + shell 40 本 + Go 8 パッケージ | すべて | 0 | 0 | 513 s |

`run-test.sh` は、`.claude/hooks/pre_bash_guard.sh` が言語に分類されないので full にフォールバックし、Go のテストも流れた。`run-verify.sh` では、shellcheck、全 hook の `sh -n`、`settings.json` 2 つの `jq -e`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、gofmt、lint(0 issues)、branch の secret scan(0931f791..8e7bfbae、clean)が通り、最後に `All verifiers passed.` が出た。

Docker は 2 回流した。1 回目(d4c7efd4)は、`apt-get install jq gawk` で awk の alternative が gawk に切り替わり、「mawk + jq」のつもりの回が gawk で走った。2 回目(8e7bfbae)は、入れたあとに `update-alternatives --set awk /usr/bin/mawk` で mawk に戻してから流した。上の表は 2 回目の結果。1 回目も、jq なしの mawk(970 / 0)と gawk + jq(1946 / 0)は通っている。

## Coverage

- Statement: 計測していない(shell に計測の道具がない)
- Branch: 計測していない
- Function: 計測していない
- Notes: 代わりに mutation で測った。依頼の 7 個と、変わった行の条件の 3 個、計 10 個の置換を、guard・lib・旧版の fixture・テストのコピーで流した(テストは d4c7efd4 の版を固定して使った)。足す前は 9 個が赤になり、1 個(X1)が全件緑のまま残った。X1 を赤にする 1 行を足し、足したあとのテストで流し直すと赤になった。10 個すべてが赤になる。等価な mutant はない

### Mutation の結果

数字は赤になったテストの件数(jq あり・なしの 2 経路を別に数える)。B 節の行は `permission_mode` なしと `bypassPermissions` の 2 モードなので、1 行が 4 件になる。B 節の行が none に変わると、G 節の AC7 の比較(旧版 deny から新版 none になる形が `intentional_fixes` の 13 件と同じか)も 2 件赤になる。

| Mutation | 何を壊したか | 足す前 | 足したあと | 赤にした行 |
| --- | --- | --- | --- | --- |
| M1 | `lex_dollar` の `${` の分岐の `xnote(ctx, s, P)` を外す | 18 | — | B の `echo ${(e):-'…'}`、`echo x${(e):-'…'}y`、`echo ${(e):-\$(…)}`、`echo "${(e):-\$(…)}"`、G |
| M2 | `msg_check` の `${` の検査を外す | 22 | — | B の `git commit -m ${(e)…}`、`git tag -a v1 -m ${(e)…}`、引用符で旗を分けた 3 形、G |
| M3 | `msg_check` で `WR[ctx, j]` でなく切り出しの `raw` を見る | 14 | — | B の `--message''=`(commit と tag)、`-"m"` の 3 形、G |
| M4 | `${` の検査から推奨の HEREDOC の形の例外を外す | 2 | — | D の推奨の HEREDOC の形(本文に `${HOME}` と見張りの語) |
| M5 | `lex_hd` の `${` での HSUB を外す | 6 | — | B の `cat <<EOF` の本文 `${(e):-\$(…)}`、G |
| M6 | DATACMD に `stat` を戻す | 6 | — | B の `stat -A 'arr[$(…)]' /dev/null`、G |
| M7 | NOEXEC から `tr` を外す | 4 | — | D の `tr "sudo" "abcd"`、`echo x \| tr "sudo" "abcd"` |
| X1 | `msg_check` の 2 つの検査の順序を入れ替える(`${` を先に見る) | 0 | 2 | 足した D の `git commit --message "${msg}$(id)"` |
| X2 | `xnote` の範囲を `${` の 2 文字に縮める(`xnote(ctx, s, s + 2)`) | 18 | — | M1 と同じ行 |
| X3 | `lex_hd` の `${` の検査を `lex_dollar` のあとに動かす(P が進んだあとの文字を見る) | 6 | — | M5 と同じ行 |

置換を流す前に、AC1 の 11 形と AC2 の 8 形を各 mutant の hook に jq あり・なしで渡した(`ts1/quick.sh`)。予測は全件の結果と一致した。X1 だけは AC の形では区別できず、全件でも 1946 件が緑だった。

### 足したテスト(8e7bfbae)

`edge_deny`(D 節、`check_modes D deny absent`)に 1 行、2 件(1946 から 1948)。

- `git commit --message "${msg}$(id)"`: `msg_check` は置換ありの語の検査を先に、`${` の検査をあとに置く(plan の Design decisions が保つと決めた順序)。X1 はこの順序を入れ替えるので、`${` と `$(` を両方含むメッセージが `commit_message` の deny の前に返る。見張り(sentinel)のコミットメッセージの規則は `-m "` の並びしか読まないので、`--message "` の形では字句解析の規則だけが deny にする。HEAD は deny/deny、X1 は none/none、旧版は none(probe `ts1/x1d.txt`)。旧版が通す形なので、B 節(旧版も deny にする形)ではなく D 節に入れた
- 形は既存の行の組み合わせで作った。`git commit --message "$(id)"`(B 節 `ac2`)と、plan の AC2 の `git commit -m "${msg}"`。新しい回避の形は作っていない。先に `-m "` の形(`git commit -m "${msg}$(id)"`、`git commit -m "${x:-$(id)}"`)を試したが、見張りが deny にするので X1 でも deny のままで、区別できなかった

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

HEAD とこの /test のテストで、失敗はない。mutant の赤は上の表のとおりで、すべて置換によるもの。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AC1: zsh の `${(e)…}` と `stat -A` の 11 形が、base では none(旧版は deny) | Fixed、テストで固定 | B 節の 11 行が jq あり・なしで deny。M1・M2・M3・M5・M6 がそれぞれの行を赤にする |
| `tr "sudo" "abcd"` が base では deny(旧版は none) | Fixed、テストで固定 | D 節の 2 行が none。M7 が赤にする |
| AC2: `${…}` の外の文字と推奨の HEREDOC の形はデータのまま | 維持 | D 節の `echo "${HOME}" 'sudo ls'` と推奨の HEREDOC の形が none。M4 が後者を赤にする |
| AC3: `ac3` の 29 形は none、旧版 deny から新版 none は `intentional_fixes` の 13 件 | 維持 | C 節と G 節が通る。M1・M2・M3・M5・M6・X2・X3 では G 節が赤になり、比較が新しい B 節の行も見ていることを確かめた |
| 既存の 1886 件 | 維持 | 1946 件と 1948 件の全件が通る |
| plan の前提: zsh が `${(e)…}` の値を評価し直し、zsh/stat の `-A` が添字を評価する | 確かめた | `zsh -f`(5.9)で、置換を `echo RAN-n >&2` にした形を流した。`echo ${(e):-'$(…)'}`、`echo ${(e):-\$(…)}`、区切りに引用符のないヒアドキュメントの本文の `${(e):-\$(…)}`、`zmodload zsh/stat; stat -A 'arr[$(…)]' /dev/null` の 4 つで置換が走った。`(e)` のない `${x:-'$(…)'}` は文字のまま出た。`tr` は何も実行しない(`ts1/zsh-premise.out`) |

## Test gaps

- 1 行だけで固定している条件がある。M5 と X3(ヒアドキュメントの本文の `${`)は B 節の 1 行、M6(`stat`)は B 節の 1 行、M4(推奨の HEREDOC の形の例外)は D 節の 1 行、X1(検査の順序)は足した D 節の 1 行だけが赤にする。M3 も、引用符で旗を分けた 3 行でしか赤にならない。その行を消すと、その条件は誰も見ていない状態に戻る
- `${` の検査を zsh のフラグの形(`${(`)だけに狭める mutant は流していない。AC1 の行はどれも `${(e)` を含むので、おそらく赤にならない(未確認)。区別するには、フラグのない `${…}` と見張りの語を含む形を deny として固定する行が要るが、それは tech-debt に止めすぎとして記録した形(`echo ${x:-'sudo ls'}`、本文が `${HOME} sudo ls` のヒアドキュメント、`git commit -m 'mention ${HOME}; never sudo ls'`)にあたる。止めすぎを望む挙動として固定すると、あとで止めすぎを直すときにテストを直す必要が出るので、足していない。plan の Design decisions は `${` 全体を外すと決めており、狭める変更はレビューで止める前提になる
- D 節の行は G 節の比較の外にある。AC2 の 2 行(`echo "${HOME}" 'sudo ls'` と推奨の HEREDOC の形)は旧版 deny・新版 none だが、base でも none なのでこの PR で判定は変わっていない(verify の報告と同じ扱い)。足した行も D 節にある
- zsh の振る舞いは macOS の `zsh -f` 5.9 でだけ確かめた。起動ファイルの設定や Linux の zsh、bash での振る舞いは見ていない(bash には `(e)` フラグも zsh/stat もない)
- この session で効いている guard は main のチェックアウトのものなので、merge 後に Claude Code の中で効くかは確かめられない
- 置換は guard の awk のプログラムの変わった行だけに当てた。`lib_json.sh` は変わっていないので、mutation の対象にしていない

## Verdict

- Pass: `./scripts/run-test.sh`(d4c7efd4)と `./scripts/run-verify.sh`(8e7bfbae)が rc 0。guard のテストは 1946 件、1 行足して 1948 件がすべて通り、`lib_json.sh` のテストは 126 件が通る。ubuntu の mawk(jq あり・なし)と gawk でも全件通る。依頼の 7 個の mutation はどれも足す前のテストで赤になり、ほかの 3 個のうち順序の入れ替え(X1)だけが残ったので、それを赤にする 1 行を足した
- Fail: なし
- Blocked: なし
