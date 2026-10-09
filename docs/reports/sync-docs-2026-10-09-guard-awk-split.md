# sync-docs report: guard-awk-split

## Cycle 1

- Date: 2026-10-09
- Plan: `docs/plans/active/2026-10-09-guard-awk-split.md`
- Pipeline cycle: 1。差分は merge base `0abfede5` から branch HEAD `1b02beef`(refactor/guard-awk-split)まで。コードの変更は `da3b55d2`(awk のプログラムを 3 つの `.awk` に移した)だけで、あとの guard の変更はコメント(`7347fa6a`、`f593bbfc`、`c3a323aa`、`8c61c6cb`)
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-09-guard-awk-split.md`(`bbedeabe`、S5 の再 review は `6dab8a5d`。Merge yes、LOW 7 件のうち 1〜6 は `8c61c6cb` で修正済み、LOW 7 が sync-docs 向け)、
  `docs/reports/verify-2026-10-09-guard-awk-split.md`(`3b6cfc9a`。partial-pass、V-1〜V-4)、
  `docs/reports/test-2026-10-09-guard-awk-split.md`(`1b02beef`。pass、guard のテスト 2,063/0、判定の比較は違い 0)

## Summary

古くなっていたのは tech-debt の 3 行(guard の限界の行、テストの穴の行、`.awk` を分類できない行)と、plan の進捗だった。self-review の LOW 7(と LOW 1 の Debt (b) の反映)、verify の V-1・V-3・V-4、test report の Test gaps を反映した。V-2 は LOW 7 と同じ指摘なので一緒に直した。

Impact (d) と Why deferred (d) を閉じ、Debt (e) と Trigger (e) は `SQ` の綴りを「開いたままのコメントの項目」として揃えた。guard の判定を新しく書いた形は `$"echo" 'sudo ls'` だけで、probe で取り直した(下の表)。

ほかの文書(AGENTS.md、README.md、`.claude/rules/ralph/git-commit-strategy.md` と template の写し、`.codex/README.md` と template の写し、`docs/recipes/`、`docs/architecture/repo-map.md`、`docs/quality/`)に、guard が 1 ファイルであることを前提にした記述はなかったので、変更していない。

差分の大きさ(`git diff 0abfede5...HEAD --stat`、この report を足す前の 1b02beef 時点): 18 files changed, 3497 insertions(+), 2853 deletions(-)。docs と tests を除くと 10 files changed, 2938 insertions(+), 2816 deletions(-)で、ほとんどは awk のプログラムを 3 つの `.awk` に移した行(`cmp` で base と一致を確認済み)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(guard の限界の行) | 10 点を直した。(1) Debt (b) の「One escape goes the other way」のあとに 2 つ目の形 `$"echo" 'sudo ls'` を足した(字句解析と bash は `echo`、zsh と dash は `$echo` という名前のコマンドと読む。base 0abfede5 でも none だったので分割が足した穴ではない。self-review LOW 1、`8c61c6cb` が `data_first_ok` のコメントに書いた)。(2) Debt (d) の行数を、最後に変えた `8c61c6cb` の値(measured at 1b02beef)の 231・616・336・465 に直した。(3) Debt (e) の直したコミットの一覧に `8c61c6cb` を足した。(4) Debt (e) の「only the code items stay open」を、`SQ` の綴りを 1 つの開いたコメントの項目として数える書き方に直した。(5) Impact (d) を取り消し線にして、閉じたことと 4 ファイルの行数を書いた。(6) Why deferred (d) を取り消し線にして、閉じた結果(AC7: `ralph init` と `ralph upgrade` が 3 つの `.awk` を core として届け、created 3 / updated 2。`legacyRalphHookCommands` は settings.json から呼ぶ hook のコマンドだけで `.awk` は入らない)と、v1 layout からの移行は流していないことを書いた。(7) Why deferred の長い (a) の中の「the comment items of (e) were closed」に「except the `SQ` spelling」を足した。(8) Why deferred (e) に、`SQ` を残した理由(判定に効かず、3 つのファイルのコメント行に触れる)を書いた。(9) Trigger (e) に、`SQ` の綴りは awk のコメントを次に編集する変更で直す、と足した。(10) Related に plan、self-review(LOW 1、LOW 7)、verify(V-1〜V-3)、test report(Test gaps)と、3 つの `.awk` のパスを足した |
| `docs/tech-debt/README.md`(テストの穴の行) | 5 点を足した。(1) 冒頭に「refactor/guard-awk-split added (c) and (g) from its own test report」。(2) Debt (c) に、2026-10-09 の mawk 1.3.4・gawk 5.2.1 での 2063/0 と、base と分割後の guard の判定の比較(991 payloads、1982 runs、違い 0)が BWK awk だけで走ったこと。Impact (c) に同じ注記。(3) 新しい (g): F 節が自動で見るのは `rules.awk` のない写しと 3 つの parse だけで、`lex.awk`・`commands.awk` のない写しと読めない(mode 000)写しは手で確かめただけ、読めない写しは root では再現しない。Impact (g)、Why deferred (g) も足した。(4) Trigger に、awk の呼び出しか `awk_status` の block か F 節の次の変更(lex と commands の写し、root では飛ばす読めない写しを足す)。(5) Related に plan(Design decisions、Acceptance criteria)と test report |
| `docs/tech-debt/README.md`(`.awk` を分類できない行) | V-4 を足した。Debt に、static verify に `.awk` を読む検査がない(`scripts/verify.local.sh` は `.sh` だけに shellcheck と `sh -n` をかける)ので、構文エラーを捕まえるのは F 節の parse だけで、guard は awk の stderr を捨てて黙って旧版の 4 規則に落ちること。Impact に、構文エラーは static verify を通ってテストでだけ見えること。Why deferred に、F 節の parse が当面の検査であること。Trigger に、同じ変更か次の `verify.local.sh`・`.awk` の変更で、lex・commands・rules を guard の順で `awk -f` に渡し `</dev/null` で回す 1 段を足す(root と template、rc 0 で出力が空のときだけ通す)。Related に verify report(V-4) |
| `docs/plans/active/2026-10-09-guard-awk-split.md` | `## Progress checklist` だけを変えた。S5 の再 review(6dab8a5d)、verify(3b6cfc9a)、test(1b02beef)、sync-docs の 4 行を Implementation started の下に足し、Review / Verification / Test artifact created にチェックを付けた。「PR created」は `/pr` が付けるのでそのまま。`./scripts/plan-visual.sh digest` は変更のあとで `cfb34e566022`(`- Approved:` 行と一致) |
| `docs/insights/events/2026-10-09-guard-awk-split.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`) |
| `docs/reports/sync-docs-2026-10-09-guard-awk-split.md` | この report |

tech-debt の行は、行の番号ではなく関数・変数・commit・テストの節の名前で書いた(`file:line` の参照は足していない)。Debt (d) の行数だけは、この行がもともと持っているもので、`8c61c6cb`(guard の 4 ファイルを最後に変えた commit)の値を 1b02beef で測った。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `grep -rl pre_bash_guard`(`docs/`、`.claude/rules/`、`AGENTS.md`、`README.md`、`CLAUDE.md`、`.codex/`、`templates/base/`、`.agents/`、`.claude/skills`、`.claude/agents`。`docs/reports`・`docs/plans`・`docs/insights`・`docs/evidence` は除く) | 当たったのは `docs/specs/2026-04-16-ralph-cli-tool.md`、`docs/tech-debt/README.md`、`.claude/rules/ralph/git-commit-strategy.md` と template の写し、`.codex/README.md` と template の写しの 6 つだけ |
| `docs/specs/2026-04-16-ralph-cli-tool.md` | 269 行目は `ralph upgrade` の出力例に `pre_bash_guard.sh` を 1 行挙げるだけの、2026-04 の仕様の例。guard の構成は書かない。変更なし |
| `.claude/rules/ralph/git-commit-strategy.md`(69・71 行目)と template | `pre_bash_guard.sh` を入口として挙げ、HEREDOC の commit の形の扱いと enforcement を言うだけ。`cmp` で root と template は同じ。変更なし |
| `.codex/README.md`(116 行目)と template | `pre_bash_guard.sh` の deny が実際にコマンドを止めること、deny だけを返すことを書く。`cmp` で同じ。変更なし |
| `AGENTS.md` の `.claude/hooks/` の行(103・104 行目) | guard のファイルの名前を挙げていない(`check_mojibake.sh` だけを例に挙げる)ので、guard の論理がどこにあるかで読み手を誤らせない。AGENTS.md を短く保つため、何も足していない。変更なし |
| `README.md`(30・111 行目) | 「bash guard」「Bash guardrails」と言うだけ。変更なし |
| `docs/recipes/`、`docs/architecture/repo-map.md`、`docs/quality/` | guard の構成を書いた記述はなかった(`docs/recipes/codex-seat-permissions.md` の `guarded` は座席の権限の語)。変更なし |
| tech-debt の guard を挙げるほかの行(101、102、125、128、129、162、166、167) | どれも `pre_bash_guard.sh` を入口か変更の対象として挙げ、関数名(`msg_check`、`lex_dollar` など)で書く。awk のプログラムが `.sh` の中にあると言う文はない。行 125 以降の記録は当時のとおり。変更なし |
| `.claude/hooks/pre_bash_guard.sh` のヘッダー、`scripts/check-template.sh`、3 つの `.awk` の先頭のコメント | verify の確認のとおり合っている(読む順、5 ファイルを一緒に置くこと、欠けると fallback)。この sync では触っていない |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。この sync は template の側に写しのあるファイルを変えていない |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この sync の差分は tech-debt、plan の進捗、report、insight だけで、hook の配線、skill、rule、script の追加・削除はない。該当なし |

### probe(2026-10-09)

`scratchpad/probe.py`。base `0abfede5` の guard と `lib_json.sh`、HEAD(1b02beef)の guard・`lib_json.sh`・3 つの `.awk`(どちらも `git show` で取り出し、HEAD の側は作業木と `cmp` で一致を確かめた)、旧版 `tests/fixtures/guard-1c4cea5a/`。判定は jq あり/なしの順で、2 つの対照(単純な `sudo ls` は deny、`ls` は none)で写しが判定していることを先に確かめた。permission_mode は付けていない。`sudo` を含む形は Bash のコマンド文字列に書かず、スクリプトの中に置いた。

| 形 | base | HEAD | 旧版 |
|----|------|------|------|
| `sudo ls`(対照) | deny/deny | deny/deny | deny/deny |
| `ls`(対照) | none/none | none/none | none/none |
| `echo 'sudo ls'`(データ) | none/none | none/none | deny/deny |
| `$'\echo' 'sudo ls'`(行にもとからある形) | none/none | none/none | deny/deny |
| `$"echo" 'sudo ls'`(今回足した形) | none/none | none/none | deny/deny |
| `$"echo" "sudo ls"` | none/none | none/none | deny/deny |

`$"echo" 'sudo ls'` は base でも none なので、分割が足した穴ではない、という自己 review LOW 1 の記述と合う。zsh と dash の読み方(`$echo` という名前のコマンド)は self-review の `t8.sh` の結果による。この sync では zsh と dash を流していない。

## Found but left

- verify の Coverage gaps にある legacy layout(v1)からの `ralph upgrade` で `.awk` が届くかは、tech-debt の Why deferred (d) に「流していない」と書いたが、debt の項目にはしていない。同じ upgrade engine に chain するので、届かない理由は見当たらないが、確かめた人がいない
- merge 後に main のチェックアウトで Claude Code が分割後の guard を使うことは、merge 前には確かめられない(verify の Coverage gaps と同じ)
- `docs/tech-debt/README.md` の 3 行以外には触れていない。AGENTS.md には何も足していない

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-09-guard-awk-split.md` | `cfb34e566022`(編集のあと。plan の `- Approved:` 行の `sha256:cfb34e566022` と一致) |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `./scripts/check-skill-sync.sh` | 13 skill(s) in lock-step |
| tech-debt の表の形(スクラッチの `tdcheck.py`、`git show HEAD:` との行ごとの比較) | 変わった行は 160、163、189 だけ。3 行とも区切りは 7 つ(5 列)のまま、列ごとのバッククォートと取り消し線の `~~` は偶数、新しい `file:line` 参照なし(`ubuntu:24` は版の表記で参照ではない)。追加した文字は 189 行目の ASCII だけ |
| `python3 scratchpad/anchors.py` | 編集前に、23 か所の old_string がそれぞれ 1 回だけ現れることを確かめた(1 か所は本文と違う文字列を書いていて 0 回だったので、実在する文字列に直してから編集した) |
| `python3 scratchpad/probe.py` | 上の表 |
| `./scripts/run-static-verify.sh` | rc 0(`Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard_commands.awk)`、tech-debt README plan references OK、check-template-purity PASS、check-skill-sync 13 件、gofmt ok、0 issues、branch secret scan `0abfede5..1b02beef` clean。編集のあとに実行。この sync の commit は含まない。commit のあとに `./scripts/secret-scan-branch.sh --strict` を流してから push する) |

## Not verified

- guard のテストスイートと mutation は流していない(`/test` が確認済み)。tech-debt に書いた数(2063/0、991 payloads、1982 runs、6 copies × 4 commands)は test report からの転記
- zsh と dash での `$"echo"` の読み方、v1 layout からの upgrade、gawk と mawk での判定の比較は、この sync では流していない
- 「ほかの文書に guard が 1 ファイルである前提の記述はない」は、`pre_bash_guard` で grep した範囲の確認。guard を別の言い方(「Bash guard」「guardrail」)で書いた文は、README.md と `docs/architecture/repo-map.md`・`docs/quality/` で目で確かめたが、全文書を別の語で網羅したわけではない。未確認です
