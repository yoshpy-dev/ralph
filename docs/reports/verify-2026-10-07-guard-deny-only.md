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

---

## pipeline cycle 2 の再実行(F2-1 の修正のあと)

- Date: 2026-10-08
- Verifier: verifier subagent (Claude)。/test の F2-1 を直したあと、2 周目の中で回し直した verify。cycle は 2 のまま。ID は前の節と混ざらないよう `V3-` で始めた
- Scope: `git diff 7eeefee5..HEAD`(HEAD c61ab2bf)。判定を変えたのは 4e829e34 だけで、END の `set_text(IN)` の直後(`.claude/hooks/pre_bash_guard.sh:1307`)に、生のコマンドに `\` と改行の並びがあれば `NODATA` を立てる 1 行を足した。63b6743a と c61ab2bf はコメントだけを変えた。テストは 3e9afad6(tester の 6 形)と 4e829e34(B 節の 8 の 3 形)。`lib_json.sh` は変わっていない。plan の digest は 7efd47f36781 に一致した
- 上の 2 つの節の `## Verdict` は、それぞれの時点の判定として残した。いまの判定は、この節の最後の `## Verdict(pipeline cycle 2 の再実行)` にある
- Evidence: `docs/evidence/verify-2026-10-07-guard-deny-only.log` の末尾(「pipeline cycle 2 re-run」の見出しから)。gitignore の対象なので commit しない

### 調べ方(再実行)

- テスト自身の例: 前の節と同じ手順で、A〜C 節から例とモードの組 572 行を取り出し、新版の root と template に jq あり・なしで渡した。比較の例の集まりは 177 行(A の deny 行 10、B 138、C 29)で、旧版(`tests/fixtures/guard-1c4cea5a/`)にも 2 つの PATH で渡した。fixture は origin/main(da4dccb0)の guard と `lib_json.sh` にバイト単位で一致する。B 節の `guard_deny_only_forms` は 26 件から 31 件になった(3e9afad6 が 2 件、4e829e34 が 3 件)
- F2-1 の形: tester の case ファイル 28 件(`scratchpad/ts3/cases3`〜`cases5`)、orchestrator の `xr3/f1.txt`〜`f7.txt`、self-review の `sr4/*.txt` 18 件、P2-5 の `xr3/q1.txt` を、`xr1/run.sh` で新旧の guard に jq あり・なしで渡した。cross-review の `xr1/*.txt` 54 件も渡し直した
- テストファイル、probe、報告にない形は作っていない

### Spec compliance(再実行)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす | 572 行 × 4 経路(root・template × jq あり・なし)が、すべて期待値と一致した。A 節の旧 ask 行 44 件は 4 モードで none、deny 行 10 件は 4 モードで deny。ask、想定外の出力、0 以外の終了コードはない |
| AC2 | 満たす | B 節の 138 件(`ac2` 75、`self_review_forms` 32、`guard_deny_only_forms` 31)は、2 モードと 4 経路のすべてで deny。`ac2` はこの周でも変わっていない |
| AC3 | 満たす | C 節の 29 件は、2 モードと 4 経路のすべてで none。29 件にも `intentional_fixes`(`tests/test-pre-bash-guard.sh:1111`)の 13 件にも、`\` と改行の並びは 1 つもない(jq の `test("\\\\\n")` で 0 件。同じ式は B 節の 7 件に当たる)。そのため 4e829e34 の行は、AC3 の判定を変えない |
| AC4 | 満たす | 572 行で、jq あり・なしの判定は全件一致した。`lib_json.sh` は 7eeefee5 から変わっていない |
| AC5 | 満たす(古い記述は /sync-docs で直す) | ask に触れる文書がないことは、前の節のまま。tech-debt の 2 行の古い点は、下の「/sync-docs に渡す一覧」に挙げた |
| AC6 | 静的な 4 つは満たす | 下の Static analysis。`run-verify.sh` の全体は /test で確かめる |
| AC7 | 満たす | 比較の例 177 行で、旧版 deny から新版 none に変わる形は jq あり・なしとも 13 件だった。`intentional_fixes` の 13 件と一致し、C 節で旧版が deny にする 13 件とも一致する。C 節の外には 1 件もない。旧版 none から新版 deny は 25 件、旧版 ask から新版 none は C 節の 3 件で、前の節と同じ。`guard_deny_only_forms` の 31 件はすべて比較の例に入り、新旧とも deny。F2-1 の形は次の小節に書いた |
| AC8 | 満たす | BWK awk 20200816、jq なしで、H 節と同じ 3 種が 0.30・1.04・0.66 秒(jq ありでは 0.15・0.86・0.41 秒)。4e829e34 で `NODATA` が立つ、行末にバックスラッシュがある 66,000 行の本文(330 KB)は 0.75 秒 |
| AC9 | 満たす | F 節の形 11 件は前の節と同じ判定(none 4 件、deny 7 件。jq あり・なしとも)。awk を外した PATH では 4 規則が deny、`ls` が none |

### F2-1 の形

- tester の case ファイルのうち、旧版が deny にする 21 件(`cases4` の 15 件、`cases3` の `cm-sudo`・`dq-grep-sudo`・`dq-sudo`・`hd-brace-sudo`・`uq-sudo`、`cases5` の `f2-commitF-sudo`)は、新版も jq あり・なしとも deny になった
- `xr3/f1.txt`〜`f7.txt` の 7 件と、`sr4/` の `a1`・`a3`・`a4`・`a5` は、新版・旧版とも deny/deny
- 旧版が none にする 7 件のうち、`hd-bq-lex`・`uq-lex`・`f3-commitF-id-control` の 3 件は、新版で deny(強くなる向き)。残る 4 件は新版も旧版も none/none。`cm-id` と `f4-commit-m-dq-id` は `git commit -m "$\`、改行、`(id)"`。`dq-lex` は `echo "$\`、改行、`(git push origin --force)"`。`f1-commitF-id` は、区切りに引用符のないヒアドキュメントを `git commit -F -` に流し、本文に `$\`、改行、`(id)` がある形。self-review の Tech debt にある字句解析の穴(`lex_dollar` の `:388` は `$` の次の 1 文字で `(` を見る)で、旧版より弱くはない
- 比較の例の外で、旧版 deny から新版 none になる形は 5 件ある。`sr4/b6`(`grep -n 'git push --force' docs.md`)と `sr4/c1`(推奨の HEREDOC の形)は、AC3 と同じ形。残る 3 件は、見張りの一致がデータ区間の中にある。`xr1/v-ml4.txt` と `xr3/q1.txt` は前の節のとおり。`sr4/a2-dq-bscrlf.txt`(`echo "$\`、CR、LF、`(sudo ls)"`)は、`\` のあとが CR なので `index` に当たらない。self-review は bash 3.2・bash 5.2・zsh・dash がこの形を文字のまま出すことを確かめた。この verify では shell を流していない

### R2-1〜R2-3 の直し(c61ab2bf)の確かめ

| 指摘 | 状態 | 根拠 |
| --- | --- | --- |
| R2-1 | 解消(plan の記録だけが残る) | ヘッダーの方針の段落(`:98-111`)は、行継続の規則が引用符を見ないこと、ただの文字の `\` と改行(単一引用符の中、引用符つき区切りのヒアドキュメントの本文やコメントの行末、`\\` のあと)でもデータ区間が消えること、推奨の HEREDOC の形は行末が `\` の行がないときだけ通ることを書く。コードの `index(IN, BS "\n")`(`:1307`)は引用符を見ないので、書き方と合う。`sr4/` の `b1`〜`b5`・`b7`・`c2`・`c3` は新版 deny、行継続のない `c1` は none で、書いてあるとおり。END のコメント(`:1303-1306`)は「in bash and dash」になった。plan の Progress の 176 行目は、まだ単一引用符の中の行継続だけを挙げている(下の一覧の 13) |
| R2-2 | ほぼ解消(テストのコメントが 1 か所残る。V3-1) | ヘッダーの「(only that body loses its region)」は消え、段落の折り返しも直った。ヘッダーは結果だけを書き、本文ごとの規則(`read_body` の `:589`)には触れない。コードのコメント(`:525-527` の `HBSNL`、`:560-562` の `joined`)は「その本文にデータ区間を与えない」と書くだけで、結果と食い違わない。`edge_sentinel_deny` のコメント(`tests/test-pre-bash-guard.sh:907-910`)は行継続の規則による deny と書き直され、「edge_none に移す」の文は消えた |
| R2-3 | 解消 | `guard_deny_only_forms` の前のコメント(`tests/test-pre-bash-guard.sh:537-546`)に「(8, found by the cycle 2 /test as F2-1) a backslash-newline anywhere」が入った |

前の節の V2-3(ヘッダーの `>&-`)は 63b6743a で解消した(`:85-86`)。V2-4(バッククォートの形がテストにない)は 3e9afad6 で解消した(`tests/test-pre-bash-guard.sh:565`。比較の例に入り、新旧とも deny)。V2-1 と V2-2 は下の一覧に入れた。

### Findings(再実行)

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V3-1 | LOW(テストのコメント) | `guard_deny_only_forms` の 3 のコメント(`tests/test-pre-bash-guard.sh:566-569`)は「a joined body is not data」と書き、本文ごとの規則が `:572` の形を止めているように読める。4e829e34 のあとは、この形も 4 の `:575` の形も行継続の規則で deny になる。本文ごとの規則を外す変異(test の J02・L03)は、この 2 形では赤にならない(self-review の R2-2)。`:907-910` は直ったが、ここは c61ab2bf で変わっていない。判定には影響しない | 次に guard のテストを触るときに、行継続の規則でも deny になることを書き足す。/sync-docs では、tech-debt のテストの穴の行に「本文ごとの規則を固定するテストがない」と書く(下の一覧の 9) |
| V3-2 | LOW(体裁) | c61ab2bf が END のコメントを直したとき、`.claude/hooks/pre_bash_guard.sh:1305` が 87 桁のまま残り、段落の折り返しが途中で止まっている(R2-2 の (3) と同じ型) | 次に guard を変えるときに折り返す |

### Static analysis(再実行)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `lib_json.sh` を分類できないため full にフォールバックした。shellcheck(hook と verify のスクリプト)、全 hook の `sh -n`(root と template)、`settings.json` の `jq -e`、check-sync、check-pipeline-sync、check-skill-sync(13 skill)、check-template-purity、tech-debt の plan 参照、gofmt と golangci-lint(0 issues)、branch の secret scan(f423f230..c61ab2bf、clean) |
| `shellcheck -S warning`(guard、`lib_json.sh`、`post_edit_verify.sh`、2 つのテスト) | rc 0 | guard とテストの info は SC2016 が 58 件、SC1003 が 4 件、SC1091 と SC2329 が 1 件ずつで、前の節と同じ |
| `dash -n`(guard、`lib_json.sh`、`post_edit_verify.sh`)、`bash -n`(2 つのテスト) | OK | |
| `cmp` root と `templates/base/`(`pre_bash_guard.sh`、`lib_json.sh`、`post_edit_verify.sh`、`.codex/README.md`) | 4 つとも同一 | guard は両方とも実行権つき |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill |

### /sync-docs に渡す一覧

`docs/tech-debt/README.md:160`(guard の限界の行):

1. 測った時点: 「each probed against the guard on 2026-10-07 at 095af1d7」。guard の判定を最後に変えたのは 4e829e34 で、コメントは c61ab2bf。測り直した時点に書き換える
2. 行数: (d) の「The guard file is 1286 lines」はいま 1372 行。「the awk program alone is 1114 lines (from the `awk '` on line 128 to its closing quote on line 1243 at 095af1d7)」は 1187 行(143 行目の `awk '` から 1329 行目の閉じる引用符まで、c61ab2bf)。冒頭の「S2c took the file from 991 to 1286 lines」は、S2c の時点の話として残せる。行番号でなく関数名で指す方針はそのままでよい
3. (a) のデータ区間の説明は「DATACMD の引数 … コメント」で終わっている。データ区間が 1 つもなくなる場合を足す: トップレベルのサブシェル・グループ・複合コマンドと、リダイレクトのついた `exec`(46806dc9)、コマンドのどこかにある `\` と改行(4e829e34)。`in_data()` のコメント(`.claude/hooks/pre_bash_guard.sh:839-842`)とヘッダーの `:98-111` が同じことを書いている
4. 新しい誤検知の型(どれも旧版も deny なので、AC7 には反しない):
   - グループと複合コマンド: `(echo sudo ls)` と `if grep -q 'sudo ' file; then echo ok; fi`(B 節で deny に固定)。ファイルを回す `for` の中の `grep -n`、`(cd docs && grep -rn …)` も同じ(self-review の P2-4)
   - `&>` と `&>>` を `/dev/null`・`/dev/stderr` に向けた形: `echo 'sudo ls' &>/dev/null` は deny、同じ意味の `>/dev/null 2>&1` は none(P2-3。テストは `tests/test-pre-bash-guard.sh:584` で deny に固定)
   - `\` と改行(R2-1): 行継続で書いた複数行のコマンド、行継続のあとの推奨のコミット形(deny の理由の文が、すでに使っている HEREDOC の形を勧める)、推奨の形の本文の行末の `\`、コメントの行末の `\`、単一引用符の中の `\` と改行、`\\` と改行、区切りの語の `EO\`・改行・`F`(`edge_sentinel_deny` の `:911` で固定)。回避策は、行継続を使わずに書くか、コマンドを分けること
5. 既知の字句解析の穴(新旧とも none): `git commit -m "$\`、改行、`(id)"`。同じ型に `echo "$\`、改行、`(git push origin --force)"` と、区切りに引用符のないヒアドキュメントを `git commit -F -` に流し、本文に `$\`、改行、`(id)` がある形がある。`lex_dollar`(`:388`)が `$` の次の 1 文字だけで `(` を見るため。bash と dash は置換として実行し、zsh はしない(self-review の `sr4/s1`)。(b) の見えないものに足すか、新しい項目にする。旧版も none なので回帰ではない
6. self-review の再実行の Tech debt の 1 行目(R2-1〜R2-3 のコメントのずれ)は、c61ab2bf でほぼ解消した。そのまま載せず、残る V3-1・V3-2・P2-6 と、`tests/test-pre-bash-guard.sh:549` の「including two that were false none before」(P2-4 のテストのコメント。2 件は表示するだけの形で、データ区間を与えない規則の代価で deny になった)だけを、コメントのずれとして書く
7. Related の列に cycle 2 の報告を足す: self-review の「pipeline cycle 2」と「pipeline cycle 2 の再実行」の節(P2-3、P2-4、R2-1、Tech debt)、test の cycle 2 の節(F2-1、Test gaps)、この verify の cycle 2 の節と再実行の節

`docs/tech-debt/README.md:163`(テストの穴の行):

8. (c) の「the guard test passes 1576/0 under mawk 1.3.4 and under gawk 5.2.1」は 39ed2759 の時点の数。cycle 2 の /test は、3e9afad6 の状態で 1704/0(mawk 1.3.4、gawk 5.2.1)。HEAD のテストは 1716 件(plan の Progress)で、mawk と gawk ではまだ流していない
9. (a) の Not pinned に、5 の字句解析の穴を足す(テストに例がない)。V3-1 の、本文ごとの規則(`read_body` の `:589`)を固定するテストがないことも足す
10. Related の列に、test の cycle 2 の節を足す

ほかの文書:

11. V2-1: `internal/org/prompts/implementer.md:27` は、まだ「`pre_bash_guard.sh` フックはシェルレベルで `-m "$(` 形式を拒否する」と書く。このブランチの guard は、行末が `\` の行がなければ推奨の HEREDOC の形を通す(AC3)。置換を含む `-m "$(...)"` は止まり、推奨の形だけが通る、と書き直すか、理由を `git-commit-strategy.md` への参照だけにする。go:embed で読む prompt なので、変えると Go のビルドに入る
12. V2-2: plan の Progress の 174 行目に、P2-3 が入ったことを確かめた(「P2-3(`&>/dev/null` も deny になる。旧版も deny)は直さず記録に回す」)。tech-debt にはまだない(4 の 2 つ目)
13. plan の Progress(/verify では直していない): 176 行目の F2-1 の修正の行は、止めすぎる範囲を「単一引用符の中の行継続」とだけ書く(R2-1 が合わせるよう勧めた)。self-review の再実行(373fa29d)と c61ab2bf の記録もまだない

### Coverage gaps(再実行)

- テストファイル、probe、報告にない形は作っていない。AC7 の「旧版の deny は新版でも deny」を確かめたのは、比較の例 177 行と、probe と case ファイルの 108 件(`xr1` 54、`ts3` 28、`xr3` 8、`sr4` 18)の範囲に限られる
- `sr4/a2` の CRLF の形を shell がどう読むかは、self-review の確かめに頼った
- 4e829e34 のあとの guard を、mawk・gawk・busybox の awk では流していない(/test の担当)
- テストスイートの実行と、4e829e34 の行を外す変異は /test の担当

## Verdict(pipeline cycle 2 の再実行)

- Verdict: pass
- Verified: AC1〜AC9 を、テスト自身の例 572 行を新版の 4 経路に、そのうち比較の例 177 行を旧版の 2 経路にも渡して確かめた。AC7 の例外は 13 件のままで、`intentional_fixes` と一致する。AC3 と `intentional_fixes` の形に `\` と改行の並びはない。F2-1 の形のうち旧版が deny にする 21 件と、`xr3/f1`〜`f7` の 7 件は、新旧とも deny。R2-1〜R2-3 のコメントの直しは、コードと probe の結果に合う。静的解析は `run-static-verify.sh` が rc 0、shellcheck の warning 以上は 0 件、template とバイト単位で一致、plan の digest も一致した
- Partially verified: LOW の 2 件(V3-1 のテストのコメント、V3-2 の折り返し)。tech-debt の 2 行、`internal/org/prompts/implementer.md:27`、plan の Progress の古い記述(上の一覧。/sync-docs と orchestrator が直す)
- Not verified: テストスイートの実行、テストファイルと probe にない形での旧版との比較、`sr4/a2` の shell での読み方、ubuntu の mawk・gawk での判定、merge 後の Claude Code での実際の効き目

---

## cycle 3 (cap raised to 3)

- Date: 2026-10-08
- Verifier: verifier subagent (Claude)。cross-review の 2 周目のあと、ユーザーが上限を 3 に上げてから回した verify。cross-review の手順どおり `cycle-count.json` は 2 のまま。ID は前の節と混ざらないよう `V4-` で始めた
- Scope: `git diff 3950ffdd..HEAD -- .claude tests templates internal docs/tech-debt`(HEAD 849f5411)。e1dfb422 は main を取り込んだマージで、`internal/` の差分のうち main にないのは `internal/org/prompts/implementer.md` だけ(`git diff --stat origin/main...HEAD`)。guard の判定を変えたのは a9ef82b1(許可リスト、ヒアドキュメントの区切りの規則、push の値を取るオプション、reset の `--`)と 849f5411(reset の `--pathspec-from-file`)。849f5411 は self-review の cycle 3 のあとに書かれたので、それを読んだのはこの verify が初めて。plan の digest は `scripts/plan-visual.sh digest` で計算し直して 7efd47f36781 に一致した
- 上の 3 つの節の `## Verdict` は、それぞれの時点の判定として残した。いまの判定は、この節の最後の `## Verdict(cycle 3)` にある
- Evidence: `docs/evidence/verify-2026-10-07-guard-deny-only.log` の末尾(「cycle 3 (cap raised to 3)」の見出しから)。gitignore の対象なので commit しない

### 調べ方(cycle 3)

- テスト自身の例: 前の節と同じ手順で、A〜C 節から例とモードの組 594 行を取り出し、新版の root と template に jq あり・なしで渡した。比較の例の集まりは 188 行(A の deny 行 10、B 149、C 29)で、旧版(`tests/fixtures/guard-1c4cea5a/`)にも 2 つの PATH で渡した。fixture は origin/main(51855166)の guard と `lib_json.sh` にバイト単位で一致する
- A〜C 節の例を 3950ffdd、a9ef82b1、HEAD から取り出して `diff` した。3950ffdd から HEAD までに増えたのは、B 節の `guard_deny_only_forms` の 11 件(180c7389 の 1 件、a9ef82b1 の 10 件)だけ。a9ef82b1 が C 節に入れた push と reset の 6 件は 849f5411 で D 節の `edge_none` に移り、C 節は 3950ffdd と同じ 29 件に戻った
- D 節の 3 つの配列(`edge_deny` 118 件、`edge_none` 83 件、`edge_sentinel_deny` 41 件)も、新版の 4 経路と旧版の 2 経路に渡した
- probe: `xr1/run.sh` で、`xr3/c2p*.txt` の 5 件、`xr1/*.txt` の 54 件、`xr3/q1.txt` と `xr3/f1.txt`〜`f7.txt` を新旧の guard に渡した
- 849f5411 の reset の規則: 使い捨てのリポジトリ(scratchpad の `vf5/reset-repo`、git 2.49.0)に中身が空の `--` という名前のファイルを置き、`opt_is` が受け付ける `--pathspec-from-file` の前置き 9 通りと `=` つきの 2 形を git に渡した。同じ文字列を guard にも渡した
- tech-debt の行と implementer の prompt が判定を書いている形を、facd295b、a9ef82b1、HEAD の guard と旧版に渡した(`vf5/docforms.sh`)
- テストファイル、probe、報告と文書にない新しい形は作っていない

### Spec compliance(cycle 3)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす | 594 行 × 4 経路(root・template × jq あり・なし)が、すべて期待値と一致した。A 節の旧 ask 行 44 件は 4 モードで none、deny 行 10 件は 4 モードで deny。新版のどの経路にも、ask、想定外の出力、0 以外の終了コードはない |
| AC2 | 満たす | B 節の 149 件(`ac2` 75、`self_review_forms` 32、`guard_deny_only_forms` 42)は、2 モードと 4 経路のすべてで deny。`ac2` は 3950ffdd から変わっていない(cycle 1 で plan の AC2 と 1 件ずつ突き合わせたもの) |
| AC3 | 満たす | `ac3`(`tests/test-pre-bash-guard.sh:632-666`)は 29 件で、plan の AC3 の 29 項目と順番どおり 1 対 1 に対応する。27 項目は plan のコードの書き方と値まで一致し、残る 2 項目(推奨の HEREDOC の形と、引用符つきの区切りのヒアドキュメントを `git commit -F -` に流す形)は plan の説明どおりの形になっている(`vf5/ac3cmp.sh`)。29 件は 2 モードと 4 経路のすべてで none |
| AC4 | 満たす | A〜C 節の 594 行と D 節の 242 行で、jq あり・なしの判定は全件一致した。`lib_json.sh` は 3950ffdd から変わっていない |
| AC5 | 満たす(古い記述は /sync-docs で直す) | ask に触れる文書はない。a9ef82b1 で古くなった記述が、tech-debt の 124・125・160・163 行目と `internal/org/prompts/implementer.md:27-30` にある(下の「/sync-docs に渡す一覧」) |
| AC6 | 静的な 4 つは満たす | 下の Static analysis。`run-verify.sh` の全体は /test で確かめる |
| AC7 | 満たす | 比較の例 188 行で、旧版 deny から新版 none に変わる形は jq あり・なしとも 13 件だった。`intentional_fixes`(`tests/test-pre-bash-guard.sh:1184-1198`)の 13 件と一致し、C 節で旧版が deny にする 13 件とも一致する。C 節の外には 1 件もない。旧版 none から新版 deny は 25 件、旧版 ask から新版 none は C 節の 3 件で、前の節と同じ。C 節から D 節に移った 6 件は新旧とも none なので、比較の例から外れても例外の集まりは変わらない。`guard_deny_only_forms` の 42 件はすべて比較の例に入り、新旧とも deny。probe は次の小節に書いた |
| AC8 | 満たす | BWK awk 20200816、jq なしで、H 節と同じ 3 種が 0.29・1.10・0.67 秒(jq ありでは 0.13・0.90・0.38 秒)。行末にバックスラッシュがある 66,000 行の本文(330 KB)は 0.85 秒 |
| AC9 | 満たす | F 節の形 11 件は前の節と同じ判定(none 4 件、deny 7 件。jq あり・なしとも)。awk を外した PATH では、4 規則が deny、`ls` が none |

### AC7 の probe(cycle 3)

- cross-review の 2 周目の P1 の 3 形(`xr3/c2p1a`〜`c2p1c`: 区切りの `$'\x45'`、`env -S`、`builtin exec`)は、新版・旧版とも deny/deny(jq あり/なし)。3 形はどれも B 節の `guard_deny_only_forms` にバイト単位で同じ例があり、比較の例に入っている
- WORTH_CONSIDERING の 2 形(`c2p2a` の `git push origin -ofoo`、`c2p2b` の `git reset HEAD -- --hard`)は、新版 none/none、旧版 none/none。2 形とも D 節の `edge_none` に同じ例がある
- `xr1/*.txt` の 54 件のうち 53 件は、新旧とも deny/deny。残る `v-ml4.txt` は新版 none、旧版 deny で、前の節と同じ(一致した語は、表示するだけの `cat` の引数の中にある)。`xr3/q1.txt` も前の節と同じ none/deny、`xr3/f1.txt`〜`f7.txt` は新旧とも deny/deny
- D 節の 242 行は、新版の 4 経路のすべてで期待値と一致した。`edge_none` の 83 件のうち 39 件は旧版が deny にする。どれもデータ区間の形か `my-sudo ls`・`x.sudo ls` で、比較の例の外にある(cycle 1 の V-1 のまま)

### 849f5411 の確かめ

1. `reset_rules`(`.claude/hooks/pre_bash_guard.sh:1137-1148`)は、語ごとに `opt_is(a, "--pathspec-from-file")` を `--` の判定より先に見て、`=` がなければ次の語を読み飛ばす。`opt_is`(`:1101-1107`)は `--` で始まり 4 文字以上の語にしか当たらないので、`--` そのものがこの前置きと取り違えられることはない。`--pathspec-file-nul` はこの名前の前置きではないので、値を取らないオプションとしてそのまま進む(`git reset -h` でも値を取らない)
2. 前置きの照合(git 2.49.0 の実測、`vf5/reset-opt.log`):
   - `--pa`、`--pat`、`--path`、`--paths`、`--pathspec`、`--pathspec-`、`--pathspec-f` の 7 通りは、git が `ambiguous option` で rc 129 を返し、作業ツリーの変更は残った。guard はこの 7 通りの後ろの `-- --hard` を deny にする。失敗するコマンドを止めるだけなので、害はない
   - `--pathspec-fr` と `--pathspec-from-file` は、git が `--` をファイル名として読み、rc 0 で hard reset をした(変更が消えた)。guard は 2 つとも deny
   - `=` つき: `--pathspec-from-file=f --hard` は git が hard reset をし(rc 0)、guard は deny。`--pathspec-from-file=f -- --hard` は git が `'--pathspec-from-file' and pathspec arguments cannot be used together` で rc 128 を返し、guard は none。どちらも git の読み方と合う
   - 旧版は `git reset --pathspec-from-file -- --hard` と `… f --hard` を none にする(D 節の `edge_deny` の 2 件)。新版はこれを deny にするので、強くなる向き
3. deny の理由の文(`:1429`)は、シェルのダブルクォートの中にある。バッククォートは 0 個、`$` は既存の `\$()` の 1 か所だけでエスケープされている。ダブルクォートとバックスラッシュも新しく増えていない。commit_message の deny を root・template × jq あり・なしで出させると、4 つとも `jq -e` で読める JSON になり、理由の文は jq あり・なしで同じだった。文の中身は V4-1 を参照
4. ヘッダーの `:117-120`(推奨の HEREDOC の形は、同じ呼び出しのどのコマンドも読むだけのコマンドか git で始まり、どの行も `\` で終わらないときだけ通る)は、判定と合う。単独の形と `git add a.txt &&` の後ろの形は none、`make test &&` と `./scripts/run-verify.sh && git add a.txt &&` の後ろの形は deny だった(`vf5/docforms.log`。facd295b の guard はどれも none、旧版はどれも deny)
5. テスト: `ac3` は plan の AC3 と一致した(上の表)。D 節に移した 6 件と新しい `git reset --pathspec-from-file=f -- --hard` は、新版の 4 経路で none(旧版も none)。`edge_deny` に足した 2 件は、新版の 4 経路で deny
6. awk の本文(`:155-1398`)に単一引用符は 0 個。root と template の guard はバイト単位で同一

### 前の節の指摘の状態(self-review cycle 3)

| 指摘 | 状態 | 根拠 |
| --- | --- | --- |
| C3-1 | guard の側は解消。文書の側は /sync-docs で直す | 849f5411 で、ヘッダーの `:117-120` と理由の文(`:1429`)が、HEREDOC の形は git commit を単独のコマンドで打つときだけ通ると書くようになった。`internal/org/prompts/implementer.md:27-30` と tech-debt の 124・125 行目は、まだ行末の `\` だけを条件に挙げている(下の一覧の 1〜3) |
| C3-2 | 解消 | 上の「849f5411 の確かめ」の 1〜2。git の実測と guard の判定が合う |
| C3-3 | 残る | `:717` の予約語の規則と `:722` の exec の規則、`EXEC_SEEN`(`:764`、`:773`、`:1367`)は、849f5411 で変わっていない |
| C3-4 | 残る | 区切りの規則のコメント(`:536-540`)と `data_first_ok` のコメント(`:744-745`)は変わっていない |
| C3-5 | 一部だけ直った | 849f5411 は push と reset の 6 件を D 節に移したとき、コメントの頭を「Cross-review cycle 2 (P2-4, P2-5)」にし、AC3 に入れない理由を足した(`tests/test-pre-bash-guard.sh:920-926`)。ID の `P2-4`・`P2-5` はそのままで、triage の番号(4・5)と合わない。`:817` の「Cycle 2 (P2-4)」、`:1008` の「change A, cycle 2」、`:616-620` の変数の行の説明も変わっていない |

### Findings(cycle 3)

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V4-1 | LOW(理由の文) | 849f5411 の理由の文(`.claude/hooks/pre_bash_guard.sh:1429`)は「HEREDOC の形は git commit を単独のコマンドで打ったときだけ通ります」と書く。判定はこれより広い。`git add a.txt && git commit -m "$(cat <<'EOF'` … の形は none で、`git commit -F - <<'EOF'` の HEREDOC は `make test &&` の後ろでも none だった(`vf5/docforms.log`)。逆に awk がない環境では、単独の形も旧版の規則で deny になる。文のとおりにすれば通る(行末の `\` がなければ)ので、利用者を誤らせることはない | 次に guard を変えるときに、つないだ呼び出しでも通る形として `git commit -F -` か `-F <file>` を挙げる。/sync-docs では、この文の「だけ」をそのまま文書に写さず、判定の条件(同じ呼び出しのトップレベルのコマンドがすべて読むだけのコマンドか git)を書く |
| V4-2 | LOW(記録) | self-review の cycle 3 の Tech debt の行は 849f5411 より前に書かれ、C3-1 の理由の文とヘッダー、C3-2 の reset の値を、残る項目に挙げている。この 2 つは 849f5411 で直った | /sync-docs では、C3-3・C3-4・C3-5(一部)と、C3-1 の文書の側だけを tech-debt に載せる(下の一覧の 10〜12) |
| V4-3 | LOW(plan の記録) | plan の Progress は a9ef82b1 の行(182 行目)で止まっていて、self-review の cycle 3(48628bd2)と 849f5411 の行がない。182 行目の「止める側への逸脱」は、つないだ呼び出しの中の推奨のコミット形が deny になること(C3-1)と、reset が `--pathspec-from-file` の値を読み飛ばすことを挙げていない。/verify は plan を直していない | /pr の前に、orchestrator が Progress に足す |

### Static analysis(cycle 3)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `lib_json.sh` を分類できないため full にフォールバックした。shellcheck(hook と verify のスクリプト)、全 hook の `sh -n`(root と template)、check-sync、check-pipeline-sync、check-skill-sync(13 skill)、check-template-purity、tech-debt の plan 参照、gofmt と golangci-lint(0 issues)、branch の secret scan(51855166..849f5411、clean) |
| `shellcheck -S warning`(guard、`lib_json.sh`、`post_edit_verify.sh`、2 つのテスト) | rc 0 | guard とテストの info は SC2016 が 58 件、SC1003 が 4 件、SC1091 と SC2329 が 1 件ずつで、前の節と同じ |
| `dash -n`(guard、`lib_json.sh`、`post_edit_verify.sh`)、`bash -n`(2 つのテスト) | OK | |
| `cmp` root と `templates/base/`(`pre_bash_guard.sh`、`lib_json.sh`、`post_edit_verify.sh`、`.codex/README.md`、`.claude/rules/ralph/git-commit-strategy.md`) | 5 つとも同一 | guard は両方とも実行権つき |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill |

### /sync-docs に渡す一覧(cycle 3)

C3-1 の文書の側(推奨の形は通る、と書いている記述):

1. `internal/org/prompts/implementer.md:27-30` は、HEREDOC の形は「コマンドのどの行もバックスラッシュで終わらなければ通る」と書く。a9ef82b1 からは、同じ Bash の呼び出しのトップレベルのコマンドがすべて、読むだけのコマンドか git で始まることも条件になる。この prompt は少し前の行で、検証のコマンドを走らせてから `git add` してコミットする手順を書いている。これを 1 つの呼び出しにつなぐと止まる(`./scripts/run-verify.sh && git add a.txt && git commit -m "$(cat <<'EOF'` … は deny、`git add a.txt &&` の後ろだけなら none)。「コミットは単独のコマンドで打つ(`git add` とはつないでよい)」の意味を足し、つなぐときの形として `-F <ファイル>` か `git commit -F - <<'EOF'` を挙げるとよい(後者は `make test &&` の後ろでも none)。go:embed で読む prompt なので、変えると Go のビルドに入る。`internal/org/*_test.go` にこの文を固定するテストはない
2. `docs/tech-debt/README.md:124`(RESOLVED のコメント)は「Two remainders」として、awk がない場合と行末の `\` だけを挙げる。3 つ目として、同じ呼び出しに読むだけのコマンドでも git でもないトップレベルのコマンドがあると deny になる場合を足す(例は self-review の C3-1 の `make test &&`、`./scripts/run-verify.sh &&`、`GIT_EDITOR=true git commit …`、後ろの `&& ./scripts/secret-scan-branch.sh --strict`)
3. `docs/tech-debt/README.md:125` の RESOLVED の括弧書き(「推奨の HEREDOC の形は通る。残るのは、awk がない環境の代替規則と、コマンドのどこかの行末に `\` がある場合の deny」)にも、同じ場合を足す
4. `.claude/rules/ralph/git-commit-strategy.md`(root と template、同一)は、70 行目の Enforcement が「blocks dangerous patterns at command time」とだけ書き、推奨の形が通るとは書いていないので、古くはない。59〜66 行目で HEREDOC の形を勧めているので、「git commit は単独のコマンドで打つ」と一言足すかどうかは /sync-docs の判断でよい。足すなら root と template を同じに保つ

`docs/tech-debt/README.md:160`(guard の限界の行):

5. 測った時点: 「each probed against the guard on 2026-10-08 at facd295b, where the guard's decisions last changed in 4e829e34 and its comments in 4ffe74fe」。判定を最後に変えたのは 849f5411(その前が a9ef82b1)、コメントも 849f5411。測り直した時点に書き換える
6. 行数: 「took it to 1372」と (d) の「The guard file is 1372 lines at facd295b … the awk program alone is 1187 lines (from the `awk '` on line 143 to its closing quote on line 1329)」は、849f5411 で 1442 行、awk の本体は 1246 行(154 行目の `awk '` から 1399 行目の閉じる引用符まで)
7. 許可リスト: (a) のデータ区間の説明に、データ区間が生まれるのは、トップレベルのどの単純コマンドも、1 語目(引用符を外した値。代入と前置きを読み飛ばす前の語)が `/` を含まない `DATACMD` の名前か `git` のときだけ、という条件を足す(`data_first_ok`、`end_cmd` から呼ぶ)。「Three cases have no data region at all」には、許可リストに外れる場合と、ヒアドキュメントの区切りに `$` かバッククォートがある場合(`lex_redir`)を足す。`NODATA` を立てる場所も「set in `lex_cmds` and `end_cmd`」に `lex_redir` を足す
8. 新しい誤検知(どれも旧版も deny なので AC7 には反しない):
   - 前置きつきの読むだけのコマンド: `env echo 'sudo ls'`、`command echo 'sudo ls'`、`nice grep 'sudo ' f`、`x=1 echo 'sudo ls'`(B 節で固定)、zsh の `=echo sudo ls`(`edge_sentinel_deny` で固定)、`exec echo 'sudo ls'`(テストにない。`vf5/docforms.log` で deny)
   - 道のある名前の読むだけのコマンド: `./echo 'sudo ls'`、`/tmp/x/cat 'sudo ls'`(B 節で固定)
   - 推奨のコミット形と同じ呼び出しに、読むだけのコマンドでも git でもないコマンドがある形(上の 2 の例。テストはない)
   - いまの行は「`exec echo 'sudo ls'` passes」と書く。facd295b では none だったが、a9ef82b1 から deny になった(旧版も deny)。この文を直す
9. Why deferred の (a) は、データ区間を与えない作りを「the last pipeline run the default cap allows (`RALPH_STANDARD_MAX_PIPELINE_CYCLES`, 2)」で入れたと書く。許可リストは、上限を 3 に上げた 3 回目の run で、cross-review の 2 周目の P1 の 3 形を受けて入れた。この経緯を足す
10. (e) のコメントのずれに、C3-3(`end_cmd` の予約語の規則と exec の規則、`EXEC_SEEN` は、許可リストのあとでは判定を変えない。いまの (e) は「`cmd_pos` sets `EXEC_SEEN` as a side effect」と書くが、その副作用はもう判定に効かない)と、C3-4(区切りの規則のコメントと `data_first_ok` のコメントが、`$'...'`・`$"..."` の読み方について shell と合わない。`$'echo'` はデータになる)を足す
11. (e) に C3-5 の残り(`tests/test-pre-bash-guard.sh:817` の「Cycle 2 (P2-4)」、`:920` の「Cross-review cycle 2 (P2-4, P2-5)」、`:1008` の「change A, cycle 2」、`:616-620` の変数の行の説明)を足す
12. C3-1 の理由の文とヘッダー、C3-2 の reset の値は 849f5411 で直ったので、載せない(V4-2)。理由の文の言い方は V4-1 のとおりで、載せるなら LOW の 1 文にとどめる
13. Related の列に、self-review の「cycle 3 (cap raised to 3)」の節、この verify の cycle 3 の節、cross-review の triage の 2 周目を足す

`docs/tech-debt/README.md:163`(テストの穴の行):

14. 「leave all 1730 tests green」と (c) の「1730/0 under mawk 1.3.4 and under gawk 5.2.1」は facd295b の時点の数。a9ef82b1 のあとは 1800 件(plan の Progress)で、849f5411 は C 節の 6 件(2 モードと G 節の旧版の 2 経路)を D 節(1 モード)に移し、3 件を足した。HEAD の件数は /test が数え直す
15. C3-3 の 2 つの変異(予約語の規則と exec の規則を外す)は、許可リストのあとでは等価になった。/test の確かめのあとで、J02・L03・N02 と同じ「固定できない等価な変異」に足す

### Coverage gaps(cycle 3)

- テストファイル、probe、報告と文書にない形は作っていない。AC7 の「旧版の deny は新版でも deny」を確かめたのは、比較の例 188 行、D 節の 242 行、probe の 67 件(`xr3/c2p*` 5、`xr1` 54、`xr3` 8)、文書に書かれた形 21 件の範囲に限られる
- self-review の C3-2 が「調べていない」とした、`no_verify_rules`(`:1149-1154`、`git merge`・`rebase`・`am` の `--no-verify`)が `--` で止まる前に値を取るオプションを読み飛ばさない点は、この verify でもコードを読んだだけで、形を渡していない。`commit_rules` は値を取るオプション(`-m`、`--message`、`--file`、`--author` などの長いオプション、短い `m`・`F`・`C`・`c`・`t`)の値の語ごと読み進めるので、値の `--` では止まらない(`:1157-1204`)。旧版には `--no-verify` の規則がないので、AC7 には関わらない
- ubuntu の mawk・gawk・busybox の awk での判定は見ていない。テストスイートの実行と、a9ef82b1・849f5411 の規則を外す変異は /test の担当
- このセッションで効いている Bash の guard は main のチェックアウトの旧版なので、新版を Claude Code の実際の呼び出しで確かめることは merge 前にはできない

## Verdict(cycle 3)

- Verdict: pass
- Verified: AC1〜AC9 を、テスト自身の例 594 行を新版の 4 経路に、そのうち比較の例 188 行を旧版の 2 経路にも渡して確かめた。`ac3` は plan の AC3 の 29 項目と 1 対 1 に戻った。AC7 の例外は 13 件のままで、`intentional_fixes` と一致する。cross-review の 2 周目の P1 の 3 形は新旧とも deny、P2 の 2 形は新版 none。D 節の 242 行も新版の 4 経路で期待値どおり。849f5411 の reset の規則は、git 2.49.0 の実測(前置き 9 通りと `=` つきの 2 形)と合い、理由の文にバッククォートとエスケープしていない `$` はなく、出力は 4 経路とも正しい JSON だった。静的解析は `run-static-verify.sh` が rc 0、shellcheck の warning 以上は 0 件、template とバイト単位で一致、plan の digest も一致した
- Partially verified: LOW の 3 件(V4-1 の理由の文の言い方、V4-2 の self-review の Tech debt の行が 849f5411 より前のもの、V4-3 の plan の Progress)。self-review の C3-3〜C3-5 は残る。tech-debt の 124・125・160・163 行目と `internal/org/prompts/implementer.md:27-30` の古い記述は、上の一覧で /sync-docs に渡す
- Not verified: テストスイートの実行(HEAD の件数を含む)、テストファイルと probe と文書にない形での旧版との比較、`no_verify_rules` の値を取るオプション、ubuntu の mawk・gawk での判定、merge 後の Claude Code での実際の効き目
