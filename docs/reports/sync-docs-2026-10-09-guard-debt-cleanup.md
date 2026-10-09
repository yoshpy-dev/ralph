# sync-docs report: guard-debt-cleanup

## Cycle 1

- Date: 2026-10-10
- Plan: `docs/plans/active/2026-10-09-guard-debt-cleanup.md`
- Pipeline cycle: 1。差分は base `a0094fe5` から branch HEAD `fa3ee868`(refactor/guard-debt-cleanup)まで。コードの変更は `f09d8209` と `566d8a8d`(`scripts/verify.local.sh`)、`d7dac506`(guard の 3 つの `.awk` とテスト)、`c7274f3b`(テストと guard のコメント)。`b312dc80` は `.awk` のコメントだけ、`f435dd5a` は tech-debt だけ
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-09-guard-debt-cleanup.md`(`2583435f`、S5 の再 review は `a709e0bd`。Merge yes。LOW 1 と 4 と LOW 2 の (1)・(3) は `c7274f3b` で修正済み、LOW 2 の (2) は記録した穴、LOW 3 が sync-docs 向け)、
  `docs/reports/verify-2026-10-09-guard-debt-cleanup.md`(`3358013b`。partial-pass、V-1〜V-6)、
  `docs/reports/test-2026-10-09-guard-debt-cleanup.md`(`fa3ee868`。pass、guard のテスト 2,068/0、判定の比較は違い 0、Test gaps に `trailing_backslashes` の下限)

## Summary

古くなっていたのは tech-debt の 3 行(guard の限界の行、テストの穴の行、`.awk` を分類できない行)と、plan の進捗だった。self-review の LOW 3 の 3 項目((a) 現在形の「still open」、(b) 同値 mutant の一覧、(c) `.awk` の行の閉じ方)、LOW 2 の残り、verify の V-1〜V-6、test report の Test gaps を反映した。LOW 3 の (4) は V-2、(1) は V-3、(2) は V-4、(3) は V-5 と同じ指摘なので、まとめて直した。

V-1 は消した仕組みを今のものとして書いていた段落で、これが一番大きい。データ区間のない場合を 5 つから 4 つにし、予約語で始まる複合コマンドと `exec` のリダイレクトを許可リストの場合に入れ、`(`・`)` の規則だけを `lex_cmds` の場合として残した。コード(`end_cmd`、`data_first_ok`、`in_data`、`lex_cmds`)を読んで書いたので、判定そのものは新しく試していない。

ほかの文書(AGENTS.md、README.md、`.claude/rules/ralph/`、`docs/quality/`、`docs/architecture/repo-map.md`、`docs/recipes/`、`templates/base/`)に、この PR で間違いになる記述はなかったので、変更していない。

差分の大きさ(`git diff a0094fe5...HEAD --shortstat`、この report を足す前の `fa3ee868` 時点): 16 files changed, 709 insertions(+), 124 deletions(-)。docs と tests を除くと 9 files changed, 168 insertions(+), 120 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(guard の限界の行) | 5 点を直した。(1) V-1: Debt のデータ区間の定義の段落「Five cases」を「Four cases」にし、1 つ目の許可リストの場合に、予約語か `{` で始まる複合コマンドと `exec` のリダイレクトを含めた(1 語目が DATACMD でも `git` でもないので `data_first_ok` が落とす。`d7dac506` 以降 `end_cmd` に専用の規則はない)。3 つ目は `(`・`)` の規則(`lex_cmds`。subshell は `(` の規則、`x)` と書く case の節は `)` の規則にも掛かる)にし、`EXEC_SEEN` の場合を外した。(2) V-2: (e) の閉じの文を「section F ... checks that `datacmd_list` holds no reserved word」から、実行時の DATACMD の配列を読む検査(3 つの `.awk` と、DATACMD を印字する `BEGIN` を足して回す。包みの名前は `cmd_pos` の `nm == "…"` から読む)に直した。(3) Trigger (e) の末尾も DATACMD の言い方にした(`datacmd_list` でも別の書き方でも)。(4) V-3: Why deferred の「its code items are still open」を「stayed open until refactor/guard-debt-cleanup」に、同じ段落の「its code items stay」を「stayed」にした。(5) 「The record as it stood before」以降(`RESW`・`EXEC_SEEN` を今の仕組みとして挙げる経緯の記録)は、当時の記録なので変えていない |
| `docs/tech-debt/README.md`(テストの穴の行) | 6 点を直した。(1) V-4: 「the equivalent mutants are now J02, L03, N02, N03, N04, and AL02 (... so N03 and N04 no longer exist)」を、N03・N04 が消えたことを先に書き、一覧を J02、L03、N02、AL02 にした。(2) V-6: Trigger の「section F of `tests/test-pre-bash-guard.sh`」に、この PR で発火したこと(F 節に実行時の DATACMD の不変条件の検査を足した)と、lex・commands の写しの検査は plan の Non-goals で持ち越したことを書いた。(3) 新しい (h): 不変条件の検査が包みの名前を `cmd_pos` の `nm == "…"` から読むので、別の関数に移すと黙って減る(self-review LOW 2 の残り、`a709e0bd` で記録した穴とした。試していない)。(4) 新しい (i): `trailing_backslashes` の下限を変える mutant(m1)を殺す行がない。base も同じ下限で、テストは 2,068/0 のまま、J の 280 行のうち 30 行は判定が変わる。足すなら `edge_deny` と `edge_none` に 2 行(バックスラッシュだけの行が 2 個と 3 個)。(h)・(i) は Debt、Impact、Why deferred、Trigger の 4 列に足し、冒頭の由来の文と Related(この plan、self-review、test report)にも足した。(5) (g): 「which is also the only syntax check they have」は S1 の `check_guard_awk` で偽になったので、「was the only syntax check they had until `check_guard_awk` in `scripts/verify.local.sh`」にした(指示にない直し。同じ行の `.awk` の行への参照が示す内容と合わせた)。(6) (c): 2026-10-09 の test report の結果(2,068/0 を macOS・mawk・gawk、判定の比較 3,913 payload で違い 0 を 3 つの awk で)を足した。busybox awk は流していないまま。N03・N04 は、「became equivalent when the allowlist went in」の経緯の文と、消えたことを書く文にだけ残した |
| `docs/tech-debt/README.md`(`.awk` を分類できない行) | V-5: Debt の症状の文(「has no language for `.awk` ... without a message.」)を取り消し線にし、閉じの文「Closed in refactor/guard-debt-cleanup. ...」はそのまま残した(Impact と Trigger の列と同じ形)。Why deferred の「Its parse check in section F covers the syntax of the three files for now.」を「Until refactor/guard-debt-cleanup, its parse check in section F was the only check of the syntax of the three files.」にした |
| `docs/plans/active/2026-10-09-guard-debt-cleanup.md` | `## Progress checklist` だけを変えた。S5 の再 review(a709e0bd)、verify(3358013b)、test(fa3ee868)、sync-docs の 4 行を Implementation started の下に足し、Review / Verification / Test artifact created にチェックを付けた。「PR created」は `/pr` が付けるのでそのまま。`./scripts/plan-visual.sh digest` は編集のあとで `f37da93972d7`(`- Approved:` 行と一致) |
| `docs/insights/events/2026-10-09-guard-debt-cleanup.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`、cycle 1) |
| `docs/reports/sync-docs-2026-10-09-guard-debt-cleanup.md` | この report |

tech-debt の行は、行の番号ではなく関数・変数・commit・テストの節の名前で書いた(新しい `file:line` の参照はない)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `grep -rn 'RESW\|EXEC_SEEN\|reserved-word rule\|exec rule' docs/ .claude/rules/ AGENTS.md README.md templates/base/ --include='*.md'` | 当たったのは `docs/tech-debt/README.md` の 160 行目だけ(`docs/reports/` と `docs/plans/` は当時の記録なので除く)。直したあとの 160 行目に残る `RESW`(2 か所)と `EXEC_SEEN`(5 か所)は、(e) の閉じの文と「The record as it stood before」の経緯で、データ区間の定義の段落には残っていない |
| `.claude/rules/ralph/git-commit-strategy.md`(69・71 行目)と template の写し | `pre_bash_guard.sh` の HEREDOC の扱いと enforcement を言うだけで、許可リストの外れの場合は「neither `git` nor a read-only one」と書き、この PR の変更と矛盾しない。変更なし |
| `.codex/README.md` と template、`docs/specs/2026-04-16-ralph-cli-tool.md`、`templates/base/scripts/check-template.sh` | `pre_bash_guard.sh` の名前を挙げるか、5 ファイルを一緒に置くことを言うだけ。この PR は 4 つの guard ファイルの中身を変えただけで、ファイルは増減していない。変更なし |
| `docs/architecture/repo-map.md`(61 行目) | `verify.local.sh` を名前だけで挙げ、中の検査を書かない。`check_guard_awk` を足す場所がない。変更なし |
| `docs/quality/`、`AGENTS.md`、`README.md` | `verify.local.sh` の検査の中身も、guard の構成も書いていない。変更なし |
| `scripts/verify.local.sh` の先頭のコメント | self-review LOW 4 が `c7274f3b` で直した(「static: shellcheck, sh -n, the awk parse of the guard, jq validity, template sync, tech-debt plan references」)。この sync では触っていない |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。template の側に写しのあるファイルは変えていない |
| `./scripts/check-skill-sync.sh` | 13 skill(s) in lock-step |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この sync の差分は tech-debt、plan の進捗、report、insight だけ。該当なし |

## Found but left

- self-review LOW 2 の (2)(`cmd_pos` の `nm == "…"` を読む方式)は、直さずに tech-debt のテストの穴の行 (h) に記録した。`a709e0bd` の再 review が「直さなくてよい」とした
- test report の m1(`trailing_backslashes` の下限)は、plan の Non-goals がテストの穴を外しているので、テストを足さずに (i) に記録した。足す 2 行の中身は test report の案のまま
- self-review の再 review が挙げた細かい点(`rules.awk` とテストの「a case clause meets the ) rule as well」は `(x)` と書く case の節に当てはまらない)は、コードのコメントなので sync-docs の範囲外。tech-debt の側は「a case clause written `x)`」と書いて合わせた
- guard は main のチェックアウトから動くので、この branch の guard が session で効くことは merge 前には確かめられない(test report の Test gaps と同じ)

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-09-guard-debt-cleanup.md` | `f37da93972d7`(編集のあと。plan の `- Approved:` 行の `sha256:f37da93972d7` と一致) |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `./scripts/check-skill-sync.sh` | 13 skill(s) in lock-step |
| tech-debt の表の形(スクラッチの `check_rows.py`、`git show HEAD:` との行ごとの比較) | 変わった行は 160、163、189 だけ。3 行とも区切り ` \| ` は 4 つ(5 列)のまま。ほかに区切りが 4 つでない行は 22 と 32 で、HEAD にもとからある(この sync の変更ではない) |
| 同 `check_rows2.py` | 3 行とも、列ごとのバッククォートと取り消し線の `~~` は偶数、新しい `file:line` 参照なし、足した非 ASCII 文字なし、置換文字(U+FFFD)なし |
| `./scripts/run-static-verify.sh` | rc 0(`Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard_commands.awk)`、tech-debt README plan references OK、check-template-purity PASS、check-skill-sync 13 件、gofmt ok、0 issues、branch secret scan `a0094fe5..fa3ee868` clean)。編集のあとに実行。この sync の commit は含まない。commit のあとに `./scripts/secret-scan-branch.sh --strict` を流してから push する |

## Not verified

- guard のテストスイートと mutation は流していない(`/test` が確認済み)。tech-debt に書いた数(2,068/0、3,913 payload、280 行、J の 30 行、9 wrappers)は test report と verify report からの転記
- V-1 の書き換えは、`pre_bash_guard_commands.awk` の `end_cmd`・`data_first_ok`・`in_data` と `pre_bash_guard_lex.awk` の `lex_cmds` を読んで書いた。書き換えた段落の各場合を guard に流して確かめてはいない(先行の self-review が許可リストの行を消した写しで、`case` の節、subshell、brace group、`if` の判定を確かめている。`a709e0bd`)
- 「ほかの文書に間違いになる記述はない」は、`pre_bash_guard`、`RESW`、`EXEC_SEEN`、`verify.local.sh`、`.awk` で grep した範囲と、README.md・`docs/quality/`・`docs/architecture/repo-map.md` を目で見た範囲の確認。guard を別の言い方で書いた文を全文書で網羅したわけではない。未確認です
