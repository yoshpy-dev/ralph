# Verify report: guard-deny-only

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-deny-only.md(Status Approved、digest 7efd47f36781。`scripts/plan-visual.sh digest` で計算し直して一致)
- Verifier: verifier subagent (Claude)、cycle 1(S2c のあと)
- Scope: `git diff origin/main...HEAD`(merge-base f423f230、HEAD a1488093。9eca7573 は main を取り込んだマージ)。AC1〜AC9 の照合、静的解析、文書のずれ。テストスイートは流していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-10-07-guard-deny-only.log`(gitignore の対象なので commit しない)

## 調べ方

- 判定の観察: `tests/test-pre-bash-guard.sh` の A〜D 節(297〜806 行)の配列を、`check_modes` を差し替えた bash で読み出した。699 行(例とモードの組)を、新版の root と template、旧版(`tests/fixtures/guard-1c4cea5a/`)に、jq のある PATH と jq を外した PATH で渡し、none / deny / ask を数えた。fixture は `git show origin/main:` の guard と `lib_json.sh` にバイト単位で一致する
- この verify では、テストファイルにない新しい形を足していない。比較はテスト自身の例(A〜D 節)と plan の AC に書かれた形に限った

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: どのモードでも ask を返さない。PR #206 で ask を期待していた行は none | 満たす | A 節の旧 ask 行 44 件を 4 モード(なし、`default`、`auto`、`bypassPermissions`)で渡し、176 回すべて none(新版の root・template、jq あり・なし)。旧版は同じ行に、モードなし・`default`・`auto` で 44 件とも ask を返し、`bypassPermissions` では none(比較対象として期待どおり)。guard は `permission_mode` を読まない(`grep` で読み取りがなく、ヘッダー 3〜4 行目にそう書いてある)。699 行のどこにも ask や想定外の出力はない |
| AC2: 列挙した形がどのモードでも deny | 満たす | plan の AC2 の 75 形(sudo 15、force push 14、hard reset 10、コミットメッセージ 11、文字列ごと実行する形 17、`--no-verify` 8)は、テストの `ac2` 配列(395〜476 行)と一つずつ突き合わせて過不足がない。B 節(`ac2` と self-review の種類 33 件、計 107 件)は、モードなしと `bypassPermissions` の 214 回すべて deny(4 経路とも)。ほかのモードは、guard がモードを読まないことと AC1 の 4 モードの結果から同じと判断した |
| AC3: 列挙した形がどのモードでも none | 満たす | plan の AC3 の 29 形は、テストの `ac3` 配列(529〜561 行)と過不足がない。C 節の 58 回(2 モード)はすべて none(4 経路とも) |
| AC4: jq がなくても同じ結果。`lib_json.sh` がエスケープを戻す | 満たす(テストの中身と観察) | 699 行で、新版の jq あり・なしの判定は全件一致した。`lib_json.sh` を直接読み、本物の改行(`a\nb`)と文字の `\n`(`a\\nb`)が jq あり・なしで同じバイト列になり、互いに区別されることを `od` で確かめた。`\uXXXX` の範囲と `\u0080` 以上を残すことは `tests/test-lib-json.sh` の C・D 節(211〜244 行)が覆っている。テストを流すのは `/test` の担当 |
| AC5: tech-debt の 124・127・158・160 行目、文書に ask の記述がない | 満たす(行番号のずれは /sync-docs で直す) | 124 行目(HEREDOC の誤検知)と 127 行目(Codex の ask)は RESOLVED、旧 159 行目(遅さ)も RESOLVED になり、158・160 行目は書き直されている。guard を説明する生きた文書(`.claude/rules`、`.codex`、`README.md`、`docs/recipes`、`docs/quality`、skill)に、guard の ask や確認の記述はない。guard の限界の行(現 160 行目)が古いのは既知で、下の D-1 にまとめた |
| AC6: 5 つの検査が通る | 静的な 4 つは満たす | `run-static-verify.sh` の中で `check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh` がすべて OK。`check-sync.sh` は単独でも rc 0。`run-verify.sh` の全体(テストを含む)は `/test` で確かめる |
| AC7: 旧版の deny は新版でも deny。例外は見張りの一致がすべてデータ区間の中にある形だけで、テストの例外はすべて AC3 にある | 満たす(書き方の穴を V-1 に記録) | 比較の例の集まり(A の deny 行 10、B 107、C 29、モードなし)で、旧版 deny から新版 none になる形は 13 件だった。これはテストの `intentional_fixes`(949〜963 行)の 13 件と一致し、C 節で旧版が deny にする 13 件とも一致する(jq あり・なしとも)。旧版 none から新版 deny に変わった形は 25 件で、どれも判定が強くなった方向。self-review の種類(文字列ごと実行する形)は B 節に入っている。`my-sudo ls`・`x.sudo ls` は下の V-1 を参照 |
| AC8: 200 KB のコマンドが macOS の awk で 5 秒以内 | 満たす | BWK awk 20200816 で、テストの H 節と同じ 3 種を jq なしで測った: 200,000 文字の引用符つき文字列 0.30 秒、13,400 個の短いコマンド 1.03 秒、バッククォートの中の 200,000 個のバックスラッシュ 0.65 秒(jq ありでは 0.14・0.88・0.40 秒)。3 つとも deny。テストの上限は 10 秒で、AC の 5 秒より緩い(V-2) |
| AC9: 止める側に倒す場合 | 満たす | テストの F 節の 11 件を直接渡した: 入れ子 4 段の 2 件と `${...}` 23 段の 2 件は none、5 段と 24 段と 150 段の 7 件は deny(jq あり・なしとも)。awk を外した PATH(jq はあり)では `sudo ls`、`git push --force`、`git reset --hard`、`git commit -m "$(id)"` の 4 件が deny、`ls` は none。読み直しのテキストの上限(`| sh` の 4 重)はテストの F 節だけが覆っていて、ここでは渡していない |

### V-1: `my-sudo ls`・`x.sudo ls` と AC7 の書き方(LOW、コードの変更は不要)

self-review の S2c の LOW は、`my-sudo ls` と `x.sudo ls` が旧版 deny、新版 none になること(見張りの語の境界 `NONB` に `-` と `.` が入るため、`.claude/hooks/pre_bash_guard.sh:1203`、`:1146`)を AC7 の比較に入れて固定するよう求めている。

判断: AC7 は書かれたとおりに満たしている。理由は 3 つある。

1. AC7 は比較の例の集まりを「AC2 と AC3 の全部と、PR #206 のテストの deny の行」と決めている。2 つの形はそのどれにも入っていないので、「テストに入れる例外はすべて AC3 に挙げる」の対象にならない
2. 見張りの語の境界は plan の Scope に `(^|[^A-Za-z0-9_.-])sudo[[:space:]]` と明記されていて、`visudo -c` が通るのも同じ仕組みによる。`visudo -c` は AC3 と `intentional_fixes` に入っている。2 つの形は見張りに一致しないので、「一致がすべてデータ区間の中にある」が空の一致で成り立つ形として扱える
3. 本物の `sudo`(先頭、`/usr/bin/sudo`、`./sudo`、`=sudo`、`\sudo`)は deny のまま。テストの D 節は `./sudo ls`(804 行)を deny、`my-sudo ls`・`x.sudo ls`(739〜740 行)を none で固定している

ただし、旧版との比較(G 節)はこの差を見ていない。D 節の例のうち、旧版 deny から新版 none になる形は 36 件あり、AC3 に挙がっていない。34 件はデータ区間の形(`echo` や `grep` の引数、コメント、`cat` に流すヒアドキュメントの本文、`git commit`・`git tag` のメッセージ)で、残りの 2 件がこの語の境界の形。AC7 の「テストに入れる例外はすべて AC3 に挙げる」は、比較の例の集まりの中でだけ成り立っている。

勧め(どちらか 1 つ、merge を止めない): (a) `my-sudo ls` を AC3 と `intentional_fixes` に足し、比較で固定する。(b) AC7 の文を「比較の例の集まりに入る例外は AC3 に挙げる」に直し、語の境界による差は D 節で固定していると書き足す。/verify はコードもテストも plan も直していない。

### V-2: AC8 の上限とテストの上限の差(LOW)

AC8 は 5 秒以内と決めているが、テストの H 節は 10 秒を上限にしている(`tests/test-pre-bash-guard.sh:1077`)。実測は 1.03 秒以下なので AC は満たすが、5〜10 秒に遅くなってもテストは通る。上限を 5 秒にそろえるか、10 秒の理由(CI の遅い機械など)をテストのコメントに書くとよい。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `lib_json.sh` を分類できないため full にフォールバックした。shellcheck(hook と verify のスクリプト)、全 hook の `sh -n`、`settings.json` の `jq -e`、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、gofmt と golangci-lint(0 issues)、branch の secret scan(f423f230..a1488093、clean) |
| `shellcheck -S warning`(guard、`lib_json.sh`、`post_edit_verify.sh`、2 つのテスト) | rc 0 | warning 以上は 0 件。info は SC1091・SC2016(guard、旧版にも同じ 2 件)、SC1003・SC2016・SC2329(テスト、引用符の中の例と `trap` から呼ぶ関数)だけ |
| `dash -n`(guard、`lib_json.sh`、`post_edit_verify.sh`) | OK | |
| `bash -n`(2 つのテスト) | OK | |
| `cmp` root と `templates/base/`(`pre_bash_guard.sh`、`lib_json.sh`、`post_edit_verify.sh`、`.codex/README.md`) | 4 つとも同一 | guard は両方とも実行権つき |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md` の guard の限界の行(160 行目) | いいえ(既知、/sync-docs で直す) | D-1。「991 lines」と書くが今は 1286 行。参照する行番号(`:100-948`、`:674`、`:677-694`、`:749-772`、`:914`、`:950-965` など)は S2c より前のもの。見張りが旧版から引き継いだ誤検知(`cp sudo dest`、`apt-get install sudo vim`、`git push --force-if-includes`、読むだけのコマンドの出力を `sed`・`sort` に流す形)を挙げていない。4 つとも旧版 deny・新版 deny で、テストの D 節(789〜792 行など)にある |
| `.claude/hooks/pre_bash_guard.sh` のヘッダー(1〜105 行) | はい | 6 項目(字句解析、読み直し、組み立て、規則、止める側に倒す場合、見張り)と「Not covered」が、コードの `sentinel`(1143〜1174 行)、`DATACMD`(1196 行)、`NONB`(1203 行)、フォールバック(1249〜1260 行)と合っている。見張りの sudo の後ろの空白を「space, tab or newline」と書くが、コードは CR・VT・FF も含む(Scope の `[[:space:]]` どおり)。判定が強くなる側なので直さなくてよい |
| `.codex/README.md`(root と template、115〜117 行) | はい | `deny` だけを返す hook と書いている |
| `.claude/rules/ralph/git-commit-strategy.md`(root と template、70 行目) | はい | 「blocks dangerous patterns」とだけ書いていて、ask に触れていない |
| plan の AC のチェック | いいえ(/verify では直さない) | AC2・AC3・AC7 のチェックが外れたまま。この report で満たすと判断したので、次に plan を更新するときにチェックを入れる。Progress checklist の「Verification artifact created」も同じ |
| `post_edit_verify.sh` のコメント | はい | root と template が同一で、`lib_json.sh` の戻しの説明と合っている |

## Observational checks

- 699 行の判定表(節、期待値、モード、新版 root jq/no-jq、template jq/no-jq、旧版 jq/no-jq、コマンド)は evidence の log にある。新版の 4 経路は 699 行すべてで期待値と一致し、互いにも一致した。旧版の jq あり・なしの判定も全行で一致した
- `lib_json.sh` の jq なしの経路で、改行・タブ・`\"`・`\\`・`\/` が jq と同じバイト列になることを `od -c` で確かめた
- このセッションで実際に効いている Bash の guard は main のチェックアウトの旧版なので、新版を Claude Code の実際の呼び出しで確かめることは merge 前にはできない

## Coverage gaps

- テストファイルにない例: plan の Verify plan は AC7 を「テストとは別の例でも確かめる」としている。この verify ではテストの D 節(比較の対象外の 189 行)と AC の形までにとどめ、テストファイルにない新しい形は渡していない。テストファイルの外での「旧版の deny は新版でも deny」は、self-review の cycle 2 の記録(例の外の H-1・M-1 の全区分が deny)に頼っていて、この verify では確かめていない
- `run-verify.sh` の全体、`tests/test-pre-bash-guard.sh`、`tests/test-lib-json.sh`、`tests/test-post-edit-verify.sh` の実行は `/test` の担当
- ubuntu の mawk・GNU sed・dash での判定は、この verify では見ていない(plan の Test plan にある)
- 読み直しのテキストの上限(`| sh` を 4 重にした形)は、テストの F 節だけが覆っている
- Codex の PreToolUse での deny の効き目は、この PR で変わっていない(`.codex/README.md` の記述のとおり)

## Verdict

- Verdict: pass
- Verified: AC1〜AC9 を、テストの中身の照合、テスト自身の例 699 行を新版(root・template)と旧版に jq あり・なしで渡した判定、200 KB の時間、awk なしの経路で確かめた。AC7 の例外 13 件は AC3 の旧版 deny の 13 件と一致する。静的解析は `run-static-verify.sh` が rc 0、shellcheck の warning 以上は 0 件、template とのバイト一致、plan の digest の一致
- Partially verified: AC7 の書き方(V-1、LOW: `my-sudo ls`・`x.sudo ls` を含む D 節の 36 件は比較の外)、AC8 のテストの上限(V-2、LOW: 10 秒)、tech-debt の guard の限界の行(D-1、/sync-docs で直す)、plan の AC のチェックの遅れ
- Not verified: テストスイートの実行、テストファイルにない例での旧版との比較、ubuntu の mawk での判定、merge 後の Claude Code での実際の効き目

---

## pipeline cycle 2(cross-review の指摘の修正のあと)

- Date: 2026-10-08
- Verifier: verifier subagent (Claude)。pipeline の 2 周目で、上限 2 の最後の周
- Scope: `git diff origin/main...HEAD`(merge-base f423f230、HEAD 0ef6fc4f)。この周のコードの変更は 46806dc9(cross-review の指摘の修正)と 0ef6fc4f(guard のヘッダーのコメントだけ。self-review の P2-1・P2-2)。plan の digest は `scripts/plan-visual.sh digest` で計算し直して 7efd47f36781 に一致した
- 上の cycle 1 の節と、その `## Verdict` は、その時点(HEAD a1488093)の判定として残した。いまの判定は、この節の最後の `## Verdict(pipeline cycle 2)` にある
- Evidence: `docs/evidence/verify-2026-10-07-guard-deny-only.log` の末尾に cycle 2 の出力を足した(gitignore の対象なので commit しない)

### 調べ方(cycle 2)

- テスト自身の例: `tests/test-pre-bash-guard.sh` の A〜C 節(`# ── A. AC1` から `# ── D. Edge cases` の手前まで)を、`check_modes` を差し替えた bash で読み、例とモードの組 562 行を取り出した。これを新版の root と template に、jq のある PATH と jq を外した PATH で渡した(2,248 回)。比較の例の集まり(398 行の `collect_corpus=yes` から 625 行の `collect_corpus=no` までに集める 172 行)は、旧版(`tests/fixtures/guard-1c4cea5a/`)にも 2 つの PATH で渡した。fixture は origin/main(f423f230)の guard と `lib_json.sh` にバイト単位で一致する
- A〜C 節の例を cycle 1 の verify の時点(1346740f)と HEAD の両方から取り出して `diff` した。増えたのは B 節の `guard_deny_only_forms`(26 件、544〜586 行)だけで、ほかの例は変わっていない
- cross-review の穴の probe: 依頼の `scratchpad/xr1/run.sh` に `xr1/*.txt` の 54 件を渡した(新版 root と旧版、jq あり・なし)
- self-review の P2-5: orchestrator の `scratchpad/xr3/q1.txt` を guard に渡した。`echo` しかしない `xr3/sem5.sh` を bash・zsh・dash で流し、shell の読み方を確かめた
- この周でも、テストファイルと probe にない新しい形は作っていない

### Spec compliance(cycle 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす | A 節の旧 ask 行 44 件(4 モード)と、モードなしの旧 none 行 22 件は、新版の 4 経路(root・template × jq あり・なし)ですべて none。A 節の deny 行 10 件(4 モード)はすべて deny。562 行 × 4 経路のどこにも、ask、想定外の出力、0 以外の終了コードはない。guard は `permission_mode` を読まない(ヘッダー 4 行目。`grep` で読み取りがない) |
| AC2 | 満たす | B 節の 133 件(`ac2` 75、`self_review_forms` 32、`guard_deny_only_forms` 26)は、2 モード(モードなし、`bypassPermissions`)と新版の 4 経路のすべてで deny。`ac2` の 75 件は cycle 1 で plan の AC2 と一つずつ突き合わせたもので、この周では変わっていない |
| AC3 | 満たす | C 節の 29 件は、2 モードと新版の 4 経路のすべてで none。`ac3` の 29 件は cycle 1 から変わっていない。46806dc9 の変更(`redir_safe` の `.claude/hooks/pre_bash_guard.sh:852-858`、`printf -v` の `:871`、`NODATA` の `:269`・`:274`・`:694`・`:699`)は、AC3 の形の判定を変えていない。deny に移った `(echo sudo ls)` と `if grep -q 'sudo ' file; then echo ok; fi` は D 節の例で、AC3 には入っていない |
| AC4 | 満たす | 562 行で、新版の jq あり・なしの判定は全件一致した。`lib_json.sh` は cycle 1 から変わっていない(`git diff 1346740f..HEAD -- .claude/hooks/lib_json.sh` が空) |
| AC5 | 満たす(古い記述は /sync-docs で直す) | ask に触れる文書がないこと、124・127 行目の RESOLVED、158・160 行目の書き直しは cycle 1 のまま。160 行目の guard の限界の行は 46806dc9 で古くなった(下の文書のずれの表。既知の P2-4) |
| AC6 | 静的な 4 つは満たす | `run-static-verify.sh` が rc 0 で、その中の check-sync(DRIFTED 0)、check-pipeline-sync、check-skill-sync(13 skill)、check-template-purity が OK。`check-sync.sh` と `check-skill-sync.sh` は単独でも rc 0。`run-verify.sh` の全体は /test で確かめる |
| AC7 | 満たす | 比較の例の集まり 172 行(A の deny 行 10、B 133、C 29)で、旧版 deny から新版 none に変わる形は、jq あり・なしとも 13 件だった。この 13 件は `intentional_fixes`(1083 行から)の 13 件と一致し、C 節で旧版が deny にする 13 件とも一致する。C 節の外には 1 件もない。旧版 none から新版 deny に変わる形は 25 件(強くなる向き)、旧版 ask から新版 none は C 節の 3 件(`echo x > .env`、`rm -rf build/`、`gh pr create --title t`)。新しい穴の形 26 件はすべて比較の例の集まりに入っていて、新版の 4 経路と旧版の 2 経路のすべてで deny。probe の 54 件と P2-5 の 1 件は、次の小節に書いた |
| AC8 | 満たす | BWK awk 20200816、jq なしで、テストの H 節と同じ 3 種が 0.31 秒、1.06 秒、0.69 秒(jq ありでは 0.14・0.94・0.36 秒)。46806dc9 が変えたヒアドキュメントの経路も、危険な文字を含まない本文で測った。行末にバックスラッシュがある 66,000 行の本文(行をつなぐ経路、330 KB の payload)が 0.79 秒、ふつうの 66,000 行が 0.48 秒 |
| AC9 | 満たす | テストの F 節の形を直接渡した。入れ子 4 段と `${...}` 23 段の 4 件は none、5 段・24 段・150 段の 7 件は deny(jq あり・なしとも)。awk を外した PATH(jq はあり)では、`sudo ls`、`git push --force`、`git reset --hard`、`git commit -m "$(id)"` が deny、`ls` が none。46806dc9 は上限の判定に触れていない |

### AC7 の例外の確かめ

AC7 の例外は、見張りの一致がすべてデータ区間の中にある形に限られる。cross-review の 3 件と consult の 3 件は、guard がデータ区間と判定した場所を shell が実際には実行する形だった。修正のあとの判定は次のとおり。

- probe の 54 件(`xr1/*.txt`)のうち 53 件は、新版も旧版も deny/deny(jq あり/なし)だった。triage の穴の形(`p1a`〜`p1c`、`p2a`、`p3a`、`v01`〜`v10`、`v-ml1`〜`v-ml3`)、consult の穴の形(`c1a`、`c1b`、`c2a`、`c3a`)、ヒアドキュメントの区切りの形(`h01`〜`h08`)、`#` の読み方(`k01`〜`k10`)、パイプのつなぎ方(`w01`〜`w10`)がすべて含まれる
- 残る `v-ml4.txt`(`cat <<'OUT' "a`、改行、`sudo ls`、改行、`b"`、改行、`x`、改行、`OUT`)だけは、新版 none/none、旧版 deny/deny。一致した `sudo ls` は、ダブルクォートで囲んだ `cat` の引数の中にある。引用符の中の改行はコマンドを区切らないので、ヒアドキュメントの本文は引数が閉じた行の次の `x` から始まる。引数は表示されるだけなので、(a) のデータ区間にあたる
- P2-5 の `xr3/q1.txt` も、新版 none/none、旧版 deny/deny。`$(...)` の中で宣言して改行の前に閉じたヒアドキュメントのあとに、`cat <<'B'` とその本文の `sudo ls` が続く形。同じ形で本文の行を `echo LINE4-RAN` にした `sem5.sh` を bash・zsh・dash で流すと、3 つとも `[]` の次に `A` と `echo LINE4-RAN` を表示した。`$(...)` の中のヒアドキュメントの本文は空になり、次の行の `cat <<'B'` がコマンドとして走る。4 行目は B の本文として表示されるだけで、実行されない。guard の読み方と同じなので、穴ではない(orchestrator の結論と一致する)
- 2 つともテストの比較の例の集まりの外にあるので、「テストに入れる例外はすべて AC3 に挙げる」には関わらない

plan の 45 行目の「fd の複製」を、fd 0〜2 への複製(と閉じる `>&-`)に読み替えた点は、データ区間を減らす向きにしか働かない。AC7 の「旧版の deny は新版でも deny」を弱めず、AC3 の形はすべて none のまま(上の表)。Progress の 173 行目に、止める側への逸脱として書いてある。

### Findings(cycle 2)

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V2-1 | LOW(文書のずれ。cycle 1 の見落とし) | `internal/org/prompts/implementer.md:27` は「このリポジトリの `pre_bash_guard.sh` フックはシェルレベルで `-m "$(` 形式を拒否するため、両エージェント共通で確実に使えるのは `-F` 形式」と書く。このブランチの guard は推奨の HEREDOC の形(`-m "$(cat <<'EOF'` … `)"`)を通す(AC3)ので、理由の一部が古い。勧めている `-F` の形はそのまま使えるので、実害はない。ずれは S2(e48797a2)からあったが、cycle 1 の verify は guard に触れる文書を洗うときに `internal/` を見ていなかった | /sync-docs で、置換を含む `-m "$(...)"` は止まり、通るのは推奨の HEREDOC の形だけ、と書き直す。または理由を `git-commit-strategy.md` への参照だけにする |
| V2-2 | LOW(plan の記録) | Progress の 173 行目は、fd の複製を 0〜2 に限ったことを止める側への逸脱として書き、`NODATA` と `printf -v` にも触れている。`&>`・`&>>` を `/dev/null` に向けた形がデータ区間を失うこと(self-review の P2-3)は書いていない。plan の 45 行目は `/dev/null` への出力を安全としている。止める側への変化なので、AC3・AC7 には影響しない | /pr の前に、173 行目か tech-debt の guard の行に足す(P2-3 は tech-debt に送ることになっている) |
| V2-3 | LOW(コメント) | 0ef6fc4f のあとのヘッダーの (6)(a)(`.claude/hooks/pre_bash_guard.sh:79-86`)は、出力の行き先を「a copy of fd 0, 1 or 2」と書く。`redir_safe`(`:855`)が安全とみなす `>&-`(閉じる)は書いていない。self-review の P2-2 は `>&-` も足すよう勧めていた。閉じた出力はどこにも書かれないので、判定の説明として誤りではない | 次に guard を変えるときに、P2-6 と一緒にヘッダーへ足す(tech-debt のコメントのずれの行に含める) |
| V2-4 | LOW(テストの穴) | `tests/test-pre-bash-guard.sh:536-543` のコメントは、新しい穴の形に「a heredoc read across a $(...) or a backtick」を挙げる。しかし配列に入っているのは `$(...)` の形(`:560`)だけで、バッククォートの形(probe の `v-ml2.txt`)はテストのどこにもない。いまの guard は `v-ml2.txt` を jq あり・なしとも deny にする(旧版も deny) | 次に guard のテストを触るときに、`v-ml2.txt` の形を `guard_deny_only_forms` に足す。tech-debt のテストの穴の行に書いてもよい |

cycle 1 の V-1 は解消のまま(G 節が `my-sudo ls`・`x.sudo ls` の旧版の側を固定し、tech-debt の行の (a) に書いてある)。V-2(テストの H 節の上限 10 秒、`tests/test-pre-bash-guard.sh:1227`)は変わっていない。

cycle 1 の記録の訂正: cycle 1 の AC2 の欄の「self-review の種類 33 件」は 32 件だった(`self_review_forms` の要素数)。合計の 107 件は合っている。

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `lib_json.sh` を分類できないため full にフォールバックした。shellcheck(hook と verify のスクリプト)、全 hook の `sh -n`(root と template)、`settings.json` の `jq -e`、Codex の hook の 3 検査、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、gofmt と golangci-lint(0 issues)、branch の secret scan(f423f230..0ef6fc4f、clean) |
| `shellcheck -S warning`(guard、`lib_json.sh`、`post_edit_verify.sh`、2 つのテスト) | rc 0 | guard とテストの info は SC2016 が 58 件、SC1003 が 4 件、SC1091 と SC2329 が 1 件ずつで、種類は cycle 1 と同じ |
| `dash -n`(guard、`lib_json.sh`、`post_edit_verify.sh`)、`bash -n`(2 つのテスト) | OK | |
| `cmp` root と `templates/base/`(`pre_bash_guard.sh`、`lib_json.sh`、`post_edit_verify.sh`、`.codex/README.md`) | 4 つとも同一 | guard は両方とも実行権つき |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md:160`(guard の限界の行) | いいえ(既知の P2-4。/sync-docs で直す) | 「2026-10-07 at 095af1d7」に測ったと書き、ファイルを「991 から 1286 行」、awk の本体を「1114 行(128〜1243 行目)」と書く。いまは 1360 行で、awk の本体は 137〜1317 行目。46806dc9 でデータ区間がなくなる形(トップレベルのサブシェル・グループ・複合コマンド、リダイレクトのついた `exec`)と、`&>/dev/null` の誤検知を挙げていない。/verify は確かめただけで、直していない |
| `docs/tech-debt/README.md:163`(テストの穴の行) | 一部古い | 「the guard test passes 1576/0 under mawk」は 39ed2759 の時点の数で、いまのテストは 1688 件(Progress の 173 行目)。/sync-docs で P2-4 と一緒に見るとよい |
| `tests/test-pre-bash-guard.sh:546` | いいえ(既知の P2-4) | deny に移した 2 件を「false none before」と書く。2 件は表示するだけの形で、データ区間を与えない規則の代価で deny になった |
| `.claude/hooks/pre_bash_guard.sh` のヘッダー | おおむね合う | 0ef6fc4f で P2-1(行継続でつないだヒアドキュメントは、その本文だけがデータでなくなる。`:103-105`、コードは `:583`)と P2-2(fd 0〜2、`printf without -v`、`&>`・`&>>`。`:79-86`)が直った。残りは V2-3 の `>&-` と、P2-6 の文脈ごとの配列の一覧 |
| `internal/org/prompts/implementer.md:27` | いいえ | V2-1 |
| plan の Progress checklist | 一部遅れ | 173 行目のあとに、self-review の cycle 2(8f852e4f)と 0ef6fc4f の記録がない。V2-2 の `&>` の件も同じ。/verify では plan を直していない |
| `.codex/README.md`、`.claude/rules/ralph/git-commit-strategy.md`(root と template) | はい | cycle 1 から変わっていない。guard を deny だけの hook として書き、ask に触れていない |

### Observational checks(cycle 2)

- 562 行の判定表(行、節、期待値、モード、比較の対象か、新版 root の jq/no-jq、template の jq/no-jq、旧版の jq/no-jq、コマンド)は evidence の log にある
- 新版の 4 経路は 562 行すべてで期待値と一致し、互いにも一致した。旧版の jq あり・なしも、比較の対象の 172 行すべてで一致した
- このセッションで効いている Bash の guard は main のチェックアウトの旧版なので、新版を Claude Code の実際の呼び出しで確かめることは merge 前にはできない

### Coverage gaps(cycle 2)

- テストファイルと probe の外の形: 依頼どおり、新しい形は作っていない。AC7 の「旧版の deny は新版でも deny」を確かめたのは、テストの比較の例 172 行と probe の 55 件の範囲に限られる
- probe の 54 件のうち、テストの比較の例とバイト単位で同じものは 16 件(`c1a`〜`c3a`、`p1a`〜`p1c`、`p2a`、`p3a`、`v01`、`v02`、`v06`、`v07`、`v10`、`v-ml1`、`v-ml3`)。6 つの穴の型は、それぞれ 1 件以上がテストに入っている。ほかの 38 件(`v-ml2` を含む)はテストで固定されておらず、いまは deny を返すことを probe で見ただけ
- `RESW` の一覧にない予約語(bash の `coproc` など)の扱いは見ていない(self-review の未確認の点のまま)
- ubuntu の mawk・GNU sed・dash での判定は見ていない
- テストスイートの実行と、46806dc9 の仕組みごとの mutation は /test の担当

## Verdict(pipeline cycle 2)

- Verdict: pass
- Verified: AC1〜AC9 を、テスト自身の例 562 行を新版(root・template)に、そのうち比較の例 172 行を旧版にも、jq あり・なしで渡した判定で確かめた。AC7 の例外は 13 件で、AC3 の旧版 deny の 13 件と一致する。新しい穴の形 26 件はすべて比較の例に入り、新旧とも deny。cross-review の probe 54 件のうち 53 件は新旧とも deny で、残る 1 件と P2-5 の 1 件は、一致がデータ区間の中にある。200 KB の時間(46806dc9 のヒアドキュメントの経路を含む)、awk なしの経路、上限の超過も確かめた。静的解析は `run-static-verify.sh` が rc 0、shellcheck の warning 以上は 0 件、template とバイト単位で一致、plan の digest も一致した
- Partially verified: tech-debt の guard の 2 行とテストのコメントが古い(既知の P2-4。/sync-docs で直す)。LOW の 4 件は、V2-1 の org の implementer の prompt、V2-2 の plan の記録、V2-3 のヘッダーの `>&-`、V2-4 のバッククォートの形がテストにないこと。cycle 1 の V-2(テストの上限 10 秒)も残る
- Not verified: テストスイートの実行、テストファイルと probe にない形での旧版との比較、ubuntu の mawk での判定、merge 後の Claude Code での実際の効き目
