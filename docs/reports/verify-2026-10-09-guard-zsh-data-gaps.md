# Verify report: guard-zsh-data-gaps

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-guard-zsh-data-gaps.md(`- Approved:` の digest 7a6840f0ccaf は `./scripts/plan-visual.sh digest` の値と一致)
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: `git diff origin/main...HEAD`(base 0931f791、HEAD e77e2937)。コードは 139d6627、tech-debt は d1302b7a、self-review の LOW 3 件の言い回しの直しは e77e2937。テストスイートは実行していない(AC5 は /test が見る)。probe には plan・テスト・tech-debt の行・`scratchpad/zg/` にある形だけを使い、guard を避ける新しい形は作っていない
- Evidence: `docs/evidence/verify-2026-10-09-guard-zsh-data-gaps.log`(scripts は `scratchpad/zg/vf1/`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 11 形がどのモードでも deny(jq あり・なし)、旧版も deny | Met | probe の対応: `l01`(`echo ${(e):-'$(sudo ls)'}`)、`m3`(`x…y`)、`w1`・`w4`(`\$` の形、引用符なし・二重引用符)、`h1`(`cat <<EOF` の本文)、`m1`(commit `-m`)、`m2`(tag `-m`)、`w2`・`w3`・`w5`(`--message''=`、`-"m"`、tag `--message''=`)、`l10`(`stat -A`)。11 形とも HEAD の root と template で deny/deny、旧版 deny、base(0931f791)では none/none。11 形はすべて B 節の `guard_deny_only_forms` に入っている(配列を source して取り出し、probe のファイルと JSON 文字列で完全一致)。B 節は `check_modes B deny absent bypassPermissions` で、guard は `permission_mode` を読まない(`:4` のコメントだけ) |
| AC2: 8 形がどのモードでも none | Met | probe の対応: `l13`・`l15`(`tr "sudo" "abcd"`、`echo x \| tr …`)、`l06`・`l07`(`-m "${msg}"`、`-m "$msg"`)、`l08`(`echo "${HOME}" 'sudo ls'`)、`ml3`(推奨の HEREDOC の形、本文に `${HOME}` と `sudo ls`)、`h2`(`git commit -F -` の本文 `use ${HOME} here`)、`l11`(`stat -f %z file`)。8 形とも HEAD の root と template で none/none。base で判定が違うのは `tr` の 2 形だけ(deny から none)。8 形はすべて D 節の `edge_none` に入っている |
| AC3: `ac3` の 29 形は none のまま、G 節の例外は `intentional_fixes` の 13 件のまま | Met(テストの実行は /test) | `ac3`(29 行)と `intentional_fixes`(13 行)は base と HEAD で同じ(取り出した配列の `diff` が rc 0)。`ac3` は archive の plan の AC3 の 29 項目と対応する: 1 行の 28 形は plan のコードスパンと完全一致、残る 1 行(区切りに引用符のあるヒアドキュメントを `git commit -F -` に流す形)は plan の言葉の説明と手で照合した。HEAD は 29 形とも none/none、旧版が deny にするのは 13 形で、`intentional_fixes` と同じ(ask の 3 形は deny に数えない)。G 節の論理(`:1400-1434`): 比較の例は `collect_corpus=yes` の間(`:401-721`、A 節の `pr206_deny`、B 節、C 節)に集め、経路ごとに「旧版 deny かつ新版 none」の例を並べ替えて `intentional_fixes` と比べ、さらに C 節の旧版 deny が `intentional_fixes` と同じことを確かめる。B 節に足した 11 形は新旧とも deny なので、この一覧に入らない。base の A〜D 節で `${` を含む行は 3 行(D 節)だけで、3 行とも base と HEAD で判定は同じ。`tr` と `stat` の語を含む行は base にはない |
| AC4: root と template の guard がバイト単位で同じ | Met | `cmp` rc 0(`lib_json.sh` も rc 0)、`./scripts/check-sync.sh` PASS(DRIFTED 0、ROOT_ONLY 0) |
| AC5: guard のテスト、lib_json のテスト、`run-verify.sh` が通る | Not verified here | /test の担当。`run-static-verify.sh` の静的な部分は rc 0。implementer の報告では 1946 件が通る(1886 + B 節 44 + D 節 16) |
| AC6: tech-debt の解消済みの行と、guard の限界の行・テストの穴の行から (a)〜(c) の未解決の記述がなくなる | Met | 「Findings of the last `/cross-review` run」の行(`docs/tech-debt/README.md:164-165`)は取り消し線と RESOLVED のコメント(139d6627)になり、書かれた直し方は code と配列に合う。guard の限界の行(`:160`)とテストの穴の行(`:163`)に `${(e)`、`stat -A`、`tr` の未解決の記述はない。新しい行(`:166`)の (a)〜(d) の例は、下の Observational checks の表のとおり、行の記述と一致する。ただし `:160` と `:163` には、この PR で古くなった記述が別に残る(V-1、V-2) |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | `.claude/hooks/pre_bash_guard.sh` が言語に分類されず full に切り替わった。shellcheck(hook と verify の scripts)OK、root と template の hook すべての `sh -n` OK、check-sync PASS、check-pipeline-sync OK、check-skill-sync 13 件 OK、template-purity PASS、tech-debt の plan 参照 OK、golang(gofmt ok、0 issues)、branch secret scan(0931f791..e77e2937)clean |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `cmp` guard の root と template | rc 0 | `lib_json.sh` も rc 0 |
| `shellcheck -S warning`(guard 2 つ、テスト) | rc 0 | 指摘 0 件 |
| `sh -n` guard、`bash -n` テスト | rc 0 | |
| awk のプログラムの単一引用符 | 0 個 | プログラムは `:165` の `awk '` から `:1444` の閉じ引用符まで。間の 1278 行に単一引用符はない |
| plan の digest | 一致 | 7a6840f0ccaf |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| guard のヘッダーのデータ区間 (a)〜(c)(`:80-100`) | Yes | (a) は `lex_dollar` の `${` の分岐の `xnote`(`:431`)、(b) は `msg_check` の `WR` の検査(`:1298`)と推奨の HEREDOC の形の例外、(c) は `lex_hd` の HSUB(`:666`)と合う。行番号は HEAD のもの(self-review の `:1294`・`:663` は e77e2937 の前の d1302b7a のもの) |
| `lex_dollar`・`stage_note`・`lex_hd`・`msg_check` のコメント、BEGIN の DATACMD と NOEXEC のコメント | Yes | self-review LOW 1 は e77e2937 で直った |
| テストのヘッダーの B、B 節の 10 番目のコメント、`edge_none` のコメント | Yes | self-review LOW 2 は e77e2937 で直った |
| tech-debt の新しい行(`:166`) | Yes | self-review LOW 3 の (1)〜(4) は e77e2937 で直った。(a) の「the `\${` form was not probed」は書いた時点では正しい。/verify が probe した結果は下の V-4 |
| tech-debt の guard の限界の行(`:160`) | No | V-1(`stat` が抜けている)、V-2(発火した Trigger) |
| tech-debt のテストの穴の行(`:163`) | No | V-2(発火した Trigger) |
| `.claude/rules/ralph/git-commit-strategy.md:69`、`internal/org/prompts/implementer.md:27-30`、`.codex/README.md:114-118` | Yes | 読むだけのコマンドを「`echo` や `grep` など」と例で挙げるだけで、DATACMD の一覧は書いていない |
| plan | 一部 | V-3(Test plan の「AC1 の 5 形」)。AC のチェックボックスが空なのはいつもの遅れで、判定には使わない |

### Findings

- V-1(LOW、/sync-docs 向け): guard の限界の行(`docs/tech-debt/README.md:160`)は、DATACMD から外したコマンドとして `test` と `[` だけを挙げる。2 か所ある。データ区間の定義の括弧の中の「`test` and `[` are not in it」と、条件の段落の「`test` and `[` are not in `DATACMD` at all, because `-v` in zsh and bash 5 evaluates a subscript the same way」。この PR で `stat` も外れた(zsh/stat の `-A NAME` が NAME の添字を評価するため)。同じ段落に `stat` を足せば合う。データ区間の定義の「a `git commit` or `git tag` `-m` value, the body of a heredoc」は `${` の条件を書いていないが、もともと「置換がないとき」の条件も省いた要約なので、足すかどうかは任意
- V-2(LOW、/sync-docs 向け): この PR で 3 つの Trigger が発火したが、行には記録されていない。どれも plan の範囲の外なので、コードの直しではなく「fix/guard-zsh-data-gaps でも持ち越した」記録が要る
  - `:160` の (d)「The next change that grows the guard: move the awk program out first」: guard は 1461 行から 1487 行に、awk のプログラムは 1260 行(`:159-1418`)から 1280 行(`:165-1444`)に増えた。plan の Non-goals はファイルを分けないと決めているが、行にはその判断がない
  - `:160` の (e)「The next change to the guard: fix the comments in the same change」: (e) のコメントは残っている。たとえば、awk のプログラムに単一引用符を書かない決まりは、いまも `lex_dollar` のコメント(`:416`)にしかない。テストのヘッダーの B は、この PR で zsh の `${(e)...}` と `stat -A` が足された分だけ進んだ
  - `:163` の (a) の Trigger「The next change to `scan_words`, `noexec_list`, `datacmd_list`, …(pin the unpinned limits … then)」: この PR は `noexec_list` に `tr` を足し、`datacmd_list` から `stat` を外したが、固定していない限界(blind spots、runner-string gap、`rg $x` など)は固定していない
- V-3(LOW、plan の記述): plan の Test plan は「AC1 の 5 形(B 節…)」と書くが、AC1 は e40087ef で 8 項目 11 形に広がった。テストは 11 形とも入っているので、実害はない。Test plan は digest の範囲にあるので /verify では直さない。直すなら Progress に注記する
- V-4(情報): 新しい行の (a) が言葉だけで書く形のうち、`\${` で文字のままにした形を probe した。`git commit -m "mention \${HOME}; never sudo ls"` は base none、HEAD deny、旧版 deny で、行の記述と合う。self-review LOW 3 の (3) の形(`stat` が 2 つ目のコマンドの先頭にある `echo 'sudo ls'; stat f`)も base none、HEAD deny、旧版 deny で、e77e2937 で直した (d) の書き方(「any top-level command」)と合う。なお、この行の Why deferred は、文字のままの `${` を「inside single quotes」とだけ書く。(a) の本文は `\${` まで広げたので、合わせるかどうかは /sync-docs に任せる

## Observational checks

probe(`scratchpad/zg/run.sh` と `vf1/revs.sh`、`vf1/td.sh`)。判定は jq あり/なし。「旧版」は `tests/fixtures/guard-1c4cea5a/`、base は origin/main 0931f791。

| 形 | base | HEAD(root と template) | 旧版 |
| --- | --- | --- | --- |
| AC1 の 11 形 | none/none | deny/deny | deny |
| `tr "sudo" "abcd"`、`echo x \| tr "sudo" "abcd"` | deny/deny | none/none | none |
| AC2 のほかの 6 形 | none/none | none/none | `echo "${HOME}" 'sudo ls'` と推奨の HEREDOC の形は deny、ほかは none |
| tech-debt `:166` (a) `git commit -m 'mention ${HOME}; never sudo ls'` | none/none | deny/deny | deny/deny |
| (a) の対照 `git commit -m 'use ${HOME}'`、`git commit -m "${msg}"`、推奨の HEREDOC の形 | none/none | none/none | 推奨の形だけ deny |
| (b) `cat <<EOF` の本文 `${HOME} sudo ls` | none/none | deny/deny | deny/deny |
| (c) `echo ${x:-'sudo ls'}` | none/none | deny/deny | deny/deny |
| (c) の対照 `echo "${HOME}" 'sudo ls'` | none/none | none/none | deny/deny |
| (d) `stat f; echo 'sudo ls'` | none/none | deny/deny | deny/deny |
| V-4 `git commit -m "mention \${HOME}; never sudo ls"`、`echo 'sudo ls'; stat f` | none/none | deny/deny | deny/deny |

plan の Edge cases も確かめた。二重引用符の中の `${(e)…}`(`l02`、`l03`)、`${…}` の中に `$(` がある形(D 節の `echo "${x:-$(sudo ls)}"`)、単一引用符の中の `'${x:-$(sudo ls)}'`(none)は、どれも base と HEAD で判定が同じ。

D 節の `edge_none` に足した 2 形(`echo "${HOME}" 'sudo ls'` と推奨の HEREDOC の形)は旧版 deny・新版 none だが、D 節は G 節の比較の外にある。2 形とも base でも none なので、この PR で判定は変わっていない。guard-deny-only の verify の V-1(D 節の行は比較の外)と同じ扱いになる。

## Coverage gaps

- テストスイート(`tests/test-pre-bash-guard.sh`、`tests/test-lib-json.sh`、`run-verify.sh`)は実行していない。AC5 と、G 節の比較が実際に通るかは /test が見る
- zsh の振る舞い(`${(e)…}` の再評価、zsh/stat の `-A`)は /verify ではもう一度確かめていない。plan の Assumptions(`zsh -f` 5.9)と cross-review の記録による
- テストと plan と tech-debt にない形での旧版との比較はしていない(新しい回避の形は作らない)。`$'…'` の中の `${` を含むメッセージ(self-review LOW 3 の (2))は probe していない
- ubuntu の mawk・gawk での判定は見ていない
- この session で効いている guard は main のチェックアウトのものなので、merge 後に Claude Code で実際に効くかは確かめられない

## Verdict

- Verdict: pass
- Verified: AC1〜AC4 と AC6。AC1 の 11 形と AC2 の 8 形を、HEAD の root と template、base、旧版に jq あり・なしで渡し、AC1 は base none から HEAD deny、AC2 は HEAD none になった。どの形もテストの B 節と D 節の配列に入っている。`ac3` の 29 形と `intentional_fixes` の 13 件は base から変わらず、HEAD は 29 形とも none、旧版 deny は 13 形で `intentional_fixes` と同じ。静的解析は `run-static-verify.sh` が rc 0、shellcheck の warning 以上は 0 件、template とバイト単位で一致、awk のプログラムに単一引用符はなく、plan の digest も一致した。self-review の LOW 3 件は e77e2937 で直っている
- Partially verified: tech-debt の `:160` と `:163` の古い記述(V-1 の `stat`、V-2 の発火した Trigger。/sync-docs に渡す)、plan の Test plan の「5 形」(V-3)
- Not verified: テストスイートの実行(AC5)、zsh での再確認、テストと記録にない形での旧版との比較、mawk・gawk での判定、merge 後の Claude Code での実際の効き目
