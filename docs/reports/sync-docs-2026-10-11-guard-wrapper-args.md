# sync-docs report: guard-wrapper-args

## Cycle 1

- Date: 2026-10-11
- Plan: `docs/plans/active/2026-10-11-guard-wrapper-args.md`
- Pipeline cycle: 1。差分は base `577ebeaf` から branch HEAD `5f54e09b`(test/guard-wrapper-args)まで。コードの変更は `tests/test-pre-bash-guard.sh` だけ(78616500、888ec6f8、160eef84、20e282be、db5134f1、10d73d88)。guard のファイル(`.claude/hooks/`、`templates/base/.claude/hooks/`)と `tests/fixtures/` は変わっていない
- 先行 report(それぞれを追加した commit。各 report 内の `HEAD:` の欄は、その report を commit する前の値なので使っていない):
  `docs/reports/self-review-2026-10-11-guard-wrapper-args.md`(`a57b6758`、再確認は `bf6b12b8`、`8d3624a5`、`3c5b0066`、`c34f8604`。Merge yes)、
  `docs/reports/verify-2026-10-11-guard-wrapper-args.md`(`356535bb`、再 verify は `aafae600`。どちらも pass。V-2 と I-1 が sync-docs 向け)、
  `docs/reports/test-2026-10-11-guard-wrapper-args.md`(`5f54e09b`。pass、guard のテスト 2,227/0、AC2 の (a)〜(g) はすべて記録した行が FAIL)

## Summary

古くなっていたのは、tech-debt のテストの穴の行(163 行目)の (h) の閉じの文 4 か所と Related、plan の進捗だった。

- V-2 (1): guard-test-gaps の閉じの「Nine rows of `edge_deny`, one per wrapper」は、9 行が `wrapper_deny` へ移ったあとは行のない配列を指す。history は書き換えず、「(moved to `wrapper_deny` in test/guard-wrapper-args)」を足した
- V-2 (2): 「The criterion leaves out three things.」のあとに 4 つの文が続いていた。4 つ目(分岐が載せていないツールのオプションは、どの行も押さえない)は、基準が外した項目というより、行がブランチに従いツールには従わないという別の点なので、「Beyond those three, an option of the real tool …」と書いて 3 つの数を保った
- I-1: Related の plan の項のすぐ後に、`docs/reports/test-2026-10-11-guard-wrapper-args.md`(Mutation: the evidence for mutations (a) to (g) in the closure of (h); Test gaps 5)を足した
- handoff が挙げた 3 点のほかに 2 点を直した。(1) 「A name removed from `WRAPPER` now fails both its rows in D」の「both its rows」は、名前ごとの行が 1 行だった頃(jq あり・なしの 2 件)の言い方で、今は名前ごとに 2〜6 行ある。「every row of that name in D, with and without jq」にし、test report の (c)(`nohup` の 2 行が両経路で落ちた)を括弧で添えた (2) 閉じの「all caught」と (g) の文は、どの awk で流したかを書いていなかった。前の閉じは「BWK awk, mawk, and gawk」と書くので、読み手が同じ範囲と取りうる。test report は、写しを gawk で全部(37 件)、mawk で 22 件、macOS の awk で 4 件((f)、`timeout` の長いオプション、`env --unset`、`nice -5`)流し、FAIL の記録はどれも gawk と byte 単位で同じと書く(Test gaps 5)ので、その範囲を (g) の文のあとに足した

閉じの文の主張は、test report の記録と照らして合っていた。

- (a)〜(e) は、それぞれ F の集合の検査が 1 件(a、b)、5 件(c。集合の検査 1 と D の `nohup` 2 行×2 経路)、3 件(d。集合の検査 1 と対照 2)、2 件(e。対照 2)で FAIL した。閉じの「a wrapper added without a row, a row whose name left `WRAPPER`, and a row whose deny comes from a sentinel match in its inner command or in the wrapper's arguments」の 4 つの場合に当たる((a) と (b) はどちらも「行のない包み」)
- (f) は 8 件(`exec -a n`、`time -o f`、`time -f %e`、`time --format %e` の jq あり・なし)で、閉じの「fails eight runs」と一致する。行を足す前のテスト(23b1ac05)では同じ写しが 2,127/0 で緑だったことも、「the suite stayed green」と合う
- (g) は、狙った道の行だけが jq あり・なしの 2 件で落ちた(g1〜g9)。`--` の行を消しただけの写し(g3、g5、p1)と `skip_env` の代入の行を消した写し(p2)は等価で緑、`return i` に替えた写し(g3r、g5r)は落ちる。閉じの「except those whose path a row cannot tell apart」と「with the `--` line … removed, … the row stays denied」と合う
- 「25 rows (888ec6f8 and 20e282be)」は、`wrapper_deny` の配列の数え直しで 25(env 5、command 2、exec 1、nohup 1、time 4、nice 2、stdbuf 2、timeout 3、xargs 5)。最初の 9 行のうち引数を `sh -c` の前に置くのは `nice -n 5`、`stdbuf -o0`、`timeout 5` の 3 行だけで、これも合う
- `env -S` の道は D の「Forms only the lexer denies」の群の 2 行が押さえる(写し s1、s2 で落ちた)。sentinel つきの行は、包みをすべて一覧から消した写し(w0)でも deny のままだった。閉じの文のこの 2 つの主張と合う

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(テストの穴の行、163 行目) | 5 か所を直した。(1) (h) の guard-test-gaps の閉じの「Nine rows of `edge_deny`」に「(moved to `wrapper_deny` in test/guard-wrapper-args)」を足した(V-2 の 1)。(2) 「An option of the real tool that a branch does not list is pinned by no row」を「Beyond those three, an option of …」にした(V-2 の 2)。(3) 「both its rows in D」を「every row of that name in D, with and without jq, and the comparison in F (mutation (c): the two `nohup` rows fail on both paths)」にした。(4) (g) の文のあとに、写しを流した awk の範囲(gawk、mawk 22 個、macOS の awk 4 個)と test report の節を足した。(5) Related の plan の項のあとに test report を足した(I-1)。ほかの文は変えていない。列は 5 のまま |
| `docs/plans/active/2026-10-11-guard-wrapper-args.md` | `## Progress checklist` だけを変えた。S8 の「この commit」を `10d73d88` にし、self-review の再確認 4(`c34f8604`)、再 verify(`aafae600`)、test(`5f54e09b`)、sync-docs(この commit)の 4 行を Implementation started の下に足し、Review / Verification / Test artifact created にチェックを付けた。「PR created」は `/pr` が付けるのでそのまま。AC1〜AC4 の欄は触っていない(下の Found but left)。`./scripts/plan-visual.sh digest` は編集のあとも `5ebd931f0fcc`(`- Approved:` 行と一致) |
| `docs/insights/events/2026-10-11-guard-wrapper-args.jsonl` | `self_review` の event を 1 行(verdict pass、cycle 1、critical 0、high 0、medium 1、low 8)と、`sync_docs` の event を 1 行(verdict pass、`--cycle auto`、cycle 1)追記した。self-review report の件数は、最初の review が LOW 6(1〜6)、再確認 1 が MEDIUM 1(7)、再確認 2 が LOW 2(8、9)、再確認 3 と 4 が 0 なので、handoff の値と一致する。追記後の `git status --short` は既存の jsonl の `M` だけで、日付つきの別ファイルは増えていない |
| `docs/reports/sync-docs-2026-10-11-guard-wrapper-args.md` | この report |

tech-debt の行は、行の番号ではなく関数・配列・test の節の名前で書いた(新しい `file:line` の参照はない)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `docs/tech-debt/README.md` の `edge_deny`、`wrapper_deny`、`nine`、`wrapper_row` を含む箇所 | 163 行目に集中。160 行目の `edge_deny` は別の項目の話で、wrapper の行は出てこない。163 行目の残りは、取り消し線の中の history(「the nine rows catch that for the nine wrappers」など)と、すでに `wrapper_deny` を指す文だった。取り消し線の中は直していない |
| 163 行目の Impact、Why deferred、Trigger の (h) | すでに閉じた書き方で、行の数にも `edge_deny` にも触れていない。変更なし |
| `git grep -l -E 'edge_deny\|wrapper_deny\|wrapper_row_forms\|test-pre-bash-guard'`(`docs/reports`、`docs/plans`、`docs/insights`、`tests/`、`.sh`、`.awk` を除く) | `docs/tech-debt/README.md` だけ |
| `git grep -n -E '2,?108\|2,?127\|2,?191\|2,?227'`(同じ除外) | 当たりなし。件数を書いた文書は tech-debt を含めてない |
| `git grep -l -i -E 'pre_bash_guard\|test-pre-bash'`(README.md、AGENTS.md、CLAUDE.md、`docs/quality`、`docs/architecture`、`docs/recipes`、`docs/specs`、`.claude/rules`、`templates/base/.claude/rules`、`.codex`) | `git-commit-strategy.md`(root と template)、`.codex/README.md`、`docs/specs/2026-04-16-ralph-cli-tool.md` の 4 つ。どれも guard の存在を挙げるだけで、テストの節や包みの行は書いていない。変更なし |
| `.claude/hooks/pre_bash_guard_rules.awk` の `WRAPPER` の上のコメント(`templates/base/` の写しも同じ) | 「a D row of tests/test-pre-bash-guard.sh pins each wrapper, and its F section reads this list」は今も正しい(`wrapper_deny` の行も D 節で流れ、F 節は一覧を読む)。guard のファイルは plan の Non-goals なので触っていない。変更なし |
| tech-debt の表の形(スクラッチの `tdcheck.py`、`git show HEAD:` との行ごとの比較) | 変わった行は 163 だけ(201 行のまま)。区切り ` \| ` は 4 つ(5 列)、エスケープのない `\|` は base と同じ 6 個(行頭と行末を含む)。列ごとのバッククォートは偶数、`~~` は偶数、U+FFFD なし。新しい `file:line` 参照なし |
| `docs/tech-debt/README.md`(163 行目の Related)の plan のパス | `docs/plans/active/2026-10-11-guard-wrapper-args.md` のまま。`/pr` の `archive-plan.sh` が書き換えるので、手では直さない。`verify.local.sh` の「tech-debt README plan references」は OK |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。template に写しのあるファイルは変えていない |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この sync の差分は tech-debt、plan の進捗、report、insight だけ。該当なし |

差分の大きさ(`git diff 577ebeaf...5f54e09b --shortstat`、この report を足す前): 7 files changed, 852 insertions(+), 43 deletions(-)。docs を除いたコードの変更は `tests/test-pre-bash-guard.sh` の 229 insertions(+), 42 deletions(-)(271 行)。walkthrough は書いていない(`/pr` が決める)。

## Found but left

- plan の AC1〜AC4 の欄は未チェックのまま(verify の I-3 の前半)。ここでは Progress checklist の artifact の 3 行だけを直した。前の PR(guard-wrapper-rows)も sync-docs は AC の欄に触れていない。チェックを付けるのは lead か `/pr` の判断に任せた。AC1〜AC3 は verify と test の report で満たしたと読める。AC4 の `./scripts/run-verify.sh` は、1 回の実行ではなく、verify の `run-static-verify.sh`(rc 0)と test の `run-test.sh`(rc 0)に分かれて通っている
- verify I-2 と self-review の任意の Follow-up: `pre_bash_guard_rules.awk` の `WRAPPER` のコメントは、F 節が一覧と手書きの行を突き合わせることを書いていない。偽ではなく、この PR は guard のファイルを変えない取り決めなので、次に guard を直す機会に足す
- test report の Test gaps のうち、tech-debt に書き足していないもの。(3)(`env` の `--chdir` のような、同じ道の別の旗は押さえない。k1 だけ実行、ほかは未確認)は、閉じの文の「per branch of `cmd_pos` and not per flag」がすでに言っている。(4)(D の lexer だけが deny する `echo x | xargs sh -c 'git push origin --force'` も、素の包みの道を押さえる)は、閉じの文の結論と矛盾しない補足。(6)(対照の判定が `unparsed` や `ask` のときの FAIL の文言は、guard を変えずに出せず実行していない。同じ分岐は x3 で実行した)は、テストの文言の話で、tech-debt の行の主題から外れる。いずれも test report にあるので、そこで追える
- マージ後に main のチェックアウトから動く guard の効き目は、マージ前には確かめられない(この PR は guard を変えていない)

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-11-guard-wrapper-args.md` | `5ebd931f0fcc`(編集のあと。plan の `- Approved:` 行の `sha256:5ebd931f0fcc` と一致。Progress checklist の変更では変わらない) |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(check-pipeline-sync、`check-skill-sync` 13 件、`check-template-purity`、tech-debt README plan references がすべて OK)。編集のあとに実行 |
| tech-debt の 163 行目の形(`tdcheck.py`) | 上の表のとおり。変わった行は 163 だけ |
| `./scripts/insights-append.sh`(`self_review`、`sync_docs`) | どちらも rc 0。追記後の `git status --short` は既存の jsonl の `M` だけ |

この commit のあとに `./scripts/secret-scan-branch.sh --strict` を流す(結果は返信に書く)。push はしない(handoff の指示)。

## Not verified

- guard のテストスイートと mutation は流していない(`/test` が確認済み)。tech-debt に足した「22 個が mawk、4 個が macOS の awk、FAIL の記録は gawk と同じ」は、test report の Mutation の表と Test gaps 5 からの転記で、この sync では実行していない。どの 22 個かは test report が挙げていないので、tech-debt には「those of (f), `timeout --signal`, `env --unset`, and `nice -5`」と書いた macOS の 4 個だけを名指しした
- `wrapper_deny` の 25 行は配列を読んで数えた。実行して deny を確かめたのは verify の probe と test report(34 行と 34 の対照、136 回)
- 「ほかの文書に間違いになる記述はない」は、上の grep の範囲の確認。guard の包みを別の言い方で書いた文を全文書で網羅したわけではない。未確認です
