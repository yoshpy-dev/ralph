# sync-docs report: guard-test-gaps

## Cycle 1

- Date: 2026-10-10
- Plan: `docs/plans/active/2026-10-10-guard-test-gaps.md`
- Pipeline cycle: 1。差分は base `49ac046c` から branch HEAD `0104799d`(test/guard-test-gaps)まで。コードの変更は `468fc73c`(`pre_bash_guard_rules.awk` の `WRAPPER`、`pre_bash_guard_commands.awk` の `cmd_pos` の 1 行、テスト)。`4f3a4414` は guard のコメントだけ、`73486db9` は tech-debt だけ
- 先行 report(それぞれを追加した commit。各 report 内の `HEAD:` の欄は、その report を commit する前の値なので使っていない):
  `docs/reports/self-review-2026-10-10-guard-test-gaps.md`(`8f237091`、再 review は `c5e7c90e`。Merge yes。LOW 1・2 は `4f3a4414` で修正済み、LOW 3 が sync-docs 向け)、
  `docs/reports/verify-2026-10-10-guard-test-gaps.md`(`a1b57346`。pass、V-1〜V-3 と情報 I-1〜I-4)、
  `docs/reports/test-2026-10-10-guard-test-gaps.md`(`0104799d`。pass、guard のテスト 2,090/0、mutation 19 個が 3 つの awk で赤、判定の比較は違い 0)

## Summary

古くなっていたのは tech-debt の 2 行(テストの穴の行の (h)(i) の閉じ方、guard の限界の行の 1 文)と、plan の進捗だった。self-review の LOW 3、verify の V-1・V-2・V-3、test report の結果を反映した。V-2・V-3 は LOW 3 と同じ指摘なので、まとめて直した。

テストの穴の行は、(i) の原文を (h) と同じく `~~` で消した。Trigger の列の「新しい包みは `WRAPPER` に入れて自分の行を持つ」は返済のきっかけではなく決まりごとなので、(h) の閉じの文へ移し、Trigger の列は取り消し線の文と「done」だけにした。移した先に、`cmd_pos` の分岐も要ること、名前だけで分岐がないと `cmd_pos` が名前の位置を返して後ろの `sh -c` を読まないこと、9 つの包みは 9 行が押さえるが 10 個目を行なしで足したずれは捕まらないこと(test report の Test gaps。試していない)を書いた。

guard の限界の行は、前の PR の閉じの文が「包みの名前を `cmd_pos` の `nm == "…"` から読む」と今の仕組みとして書いていたのを、`468fc73c` 以降は DATACMD と `WRAPPER` を 1 つの dump から実行時に読むと直した。同じ文の「`BEGIN` が DATACMD を出す」も、`WRAPPER` も出すと合わせた。

ほかの文書(AGENTS.md、README.md、`.claude/rules/ralph/`、`docs/quality/`、`docs/architecture/`、`docs/recipes/`、`docs/specs/`、`templates/base/`)に、この PR で間違いになる記述はなかったので、変更していない。

差分の大きさ(`git diff 49ac046c...0104799d --shortstat`、この report を足す前): 13 files changed, 548 insertions(+), 82 deletions(-)。docs と tests を除くと(guard の 6 ファイル、root と template の写し)58 insertions(+), 30 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(テストの穴の行、163 行目) | 6 点を直した。(1) LOW 3 / V-2: Debt の (i) の原文(「No row kills a mutant …」から「in `edge_none`, the same with three backslashes.」まで)を `~~` で消し、「Closed in test/guard-test-gaps (468fc73c) …」の文はそのまま残した。(2) LOW 3 / V-3: Trigger の「a new wrapper goes into `WRAPPER` and gets its own row of `edge_deny`」を外し、Debt の (h) の閉じの文へ移した(上の Summary のとおり)。Trigger は「(h) and (i) done in test/guard-test-gaps; or a report from a gawk or busybox awk host.」になった。(3) (h) の閉じ: 18 個の mutant は BWK awk、mawk、gawk のすべてで赤、と書いた。(4) (i) の閉じ: m1 は 3 つの awk で赤、base と HEAD の比較(固有の payload 4,347、jq あり・なしで 8,694 組・17,388 回の実行、うち 153 件が包みの端の入力)は 3 つの awk で違い 0、mutant 3 個を同じ比較に通すと 50・50・60 組が変わる、を足した。(5) (c): 2090 件の回(macOS の BWK awk、ubuntu:24.04 の mawk 1.3.4 と gawk 5.2.1、jq あり。比較は 3 つの awk で違い 0)を足した。(6) Related に `docs/reports/test-2026-10-10-guard-test-gaps.md`(Mutation、判定の比較、Test gaps)を足した。Impact・Why deferred の (h)(i) は、すでに閉じた書き方なので変更なし |
| `docs/tech-debt/README.md`(guard の限界の行、160 行目) | V-1: refactor/guard-debt-cleanup の閉じの文の括弧を直した。「the test runs the three `.awk` files with an extra `BEGIN` that prints DATACMD」を「prints DATACMD and `WRAPPER` in one dump」にし、「the wrapper names are read from the `nm == "…"` comparisons in `cmd_pos`」を「were read from the `nm == "…"` comparisons in `cmd_pos` until test/guard-test-gaps (468fc73c), and since then are read from the `WRAPPER` list at run time」にした。ほかの列に同じ記述はなかった |
| `docs/plans/active/2026-10-10-guard-test-gaps.md` | `## Progress checklist` と AC5 のチェックだけを変えた。self-review の行の、残した文の場所を直した(`pre_bash_guard.sh` のヘッダー item 6、データ区間の許可リストの 1 語目の話。wrapper は広い意味)。再 review(c5e7c90e)、verify(a1b57346)、test(0104799d)、sync-docs の 4 行を Implementation started の下に足し、Review / Verification / Test artifact created と AC5 にチェックを付けた。「PR created」は `/pr` が付けるのでそのまま。`./scripts/plan-visual.sh digest` は編集のあとで `94d656c12780`(`- Approved:` 行と一致) |
| `docs/insights/events/2026-10-10-guard-test-gaps.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`、cycle 1) |
| `docs/reports/sync-docs-2026-10-10-guard-test-gaps.md` | この report |

tech-debt の行は、行の番号ではなく関数・変数・commit・report の節の名前で書いた(新しい `file:line` の参照はない)。plan の `git diff` で変わった hunk は、AC5 の 1 行と Progress checklist の 2 か所だけ。

handoff との違いが 2 点ある。handoff は「8,694 runs」と「BSD awk」と書いたが、test report は 4,347 payload × jq あり・なし = 8,694 組(base と HEAD の 2 回ずつで 17,388 回の実行)、macOS の awk は BWK awk 20200816 と書く。tech-debt の (c) と plan には test report の言い方に合わせて「8,694 組」「BWK awk」と書いた。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `grep -rn 'nm ==\|WRAPPER\|wrapper names' docs/ .claude/rules/ AGENTS.md README.md templates/base/ --include='*.md'` | 当たったファイルは、`docs/tech-debt/README.md`(160・163 行目、上のとおり直した)、この plan、この PR の 3 report、guard-debt-cleanup の plan と 5 report のみ。plan(archive)と `docs/reports/` は当時の記録なので変更しない。`docs/quality/`、`.claude/rules/`、`AGENTS.md`、`README.md`、`templates/base/` には当たりなし |
| `git grep -i 'DATACMD\|NOEXEC\|cmd_pos\|wrapper'`(`docs/quality`、`docs/architecture`、`docs/recipes`、`docs/specs`、`.claude/rules`、`AGENTS.md`、`README.md`、`CLAUDE.md`、`templates/base`。`.awk`・`.sh`・`.json` を除く) | 当たったのは、「wrapper」を一般の意味で使う文(`run-static-verify.sh` と `run-test.sh`、verifier と tester の agent 定義、Codex の hook の README、codex-cli-parity の spec の表)と、codex の呼び出しの `command` の説明だけで、guard の `WRAPPER` とは別の話。変更なし |
| tech-debt の表の形(スクラッチの `tdcheck.py`、`git show HEAD:` との行ごとの比較) | 変わった行は 160 と 163 だけ(190 行のまま)。2 行とも区切り ` \| ` は 4 つ(5 列)、`\|` で分けた片は 7 のまま。列ごとのバッククォートは偶数、`~~` は偶数、新しい `file:line` 参照なし、タブと CR なし。ほかの行は HEAD と同じ |
| `docs/tech-debt/README.md`(163 行目の Related)の plan のパス | `docs/plans/active/2026-10-10-guard-test-gaps.md` のまま。`/pr` の `archive-plan.sh` が動かす前に書き換える(verify I-1)ので、手では直さない。`verify.local.sh` の「tech-debt README plan references」は OK |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。template に写しのあるファイルは変えていない |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この sync の差分は tech-debt、plan の進捗、report、insight だけ。該当なし |

## Found but left

- verify I-4: `pre_bash_guard.sh` のヘッダー item 3 の段落の詰め直しが、途中の行(「its flags), and stdbuf. Names are」)で止まっている。体裁だけで、中身は正しい。この依頼は `.claude/hooks/*` と `templates/base/.claude/hooks/*` を変えない取り決めなので触っていない。直すなら guard のコメントだけの commit になり、`/self-review` から回し直す対象になる
- verify I-1: Related の plan のパスは `active/` のまま(`/pr` が書き換える。`archive/` を先に書かない)
- 10 個目の包みを行なしで足したずれ(名前だけで分岐がない、またはその逆)は、テストで捕まえる仕組みがない。tech-debt の (h) の閉じの文に、決まりごとと「試していない」を書いた。新しい行は足していない(self-review、verify、test の 3 report が、判定を悪くする向きではないとして記録だけにした)
- マージ後に main のチェックアウトから動く guard の効き目は、マージ前には確かめられない(test report の Test gaps と同じ)

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-10-guard-test-gaps.md` | `94d656c12780`(編集のあと。plan の `- Approved:` 行の `sha256:94d656c12780` と一致。AC5 のチェックと Progress の変更では変わらない) |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0、FAIL 0(`check-skill-sync` 13 件、`check-template-purity` PASS、tech-debt README plan references OK、pipeline order の参照 OK)。編集のあとに実行 |
| tech-debt の 2 行の形(`tdcheck.py`) | 上の表のとおり。変わった行は 160 と 163 だけ |

この commit のあとに `./scripts/secret-scan-branch.sh --strict` を流し、exit 0 なら push する(結果は返信に書く)。

## Not verified

- guard のテストスイートと mutation は流していない(`/test` が確認済み)。tech-debt に書いた数(2090/0、4,347 payload、8,694 組、17,388 回、153 件、18 個と 19 個の mutant、50・50・60 組)は test report からの転記。macOS の BWK awk 20200816、mawk 1.3.4、gawk 5.2.1 だけで、busybox awk は見ていない
- 「ほかの文書に間違いになる記述はない」は、`nm ==`、`WRAPPER`、`wrapper names`、`DATACMD`、`NOEXEC`、`cmd_pos`、`wrapper` で grep した範囲の確認。guard の包みを別の言い方で書いた文を全文書で網羅したわけではない。未確認です
- `~~` で消した (i) の原文は、現在形のまま(「No row kills a mutant …」)残してある。元の文の意味は変えていない。(h) の閉じの「not tried」は、test report の Test gaps の言い方(10 個目の包みのずれは行がなければ捕まらない)を、「名前だけで分岐がない場合」に当てはめて書いた。その場合を実際に流して確かめてはいない(B の mutant 9 個は、既存の 9 つの包みについてこの形を試している)
