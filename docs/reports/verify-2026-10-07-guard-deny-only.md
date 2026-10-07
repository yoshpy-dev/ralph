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
