# sync-docs report: guard-zsh-data-gaps

## Cycle 1

- Date: 2026-10-09
- Plan: `docs/plans/active/2026-10-09-guard-zsh-data-gaps.md`
- Pipeline cycle: 1。差分は merge base `0931f791` から branch HEAD `349d838c`(fix/guard-zsh-data-gaps)まで。コードは `139d6627`、コメントの直しは `e77e2937`、テストの追加は `8e7bfbae`
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-09-guard-zsh-data-gaps.md`(`ae957fa9`。Merge yes、LOW 3 件、`e77e2937` で直した)、
  `docs/reports/verify-2026-10-09-guard-zsh-data-gaps.md`(`d4c7efd4`。pass、V-1〜V-4)、
  `docs/reports/test-2026-10-09-guard-zsh-data-gaps.md`(`349d838c`。pass、1948/0、mutation 10 件すべて red)

## Summary

古くなっていたのは、tech-debt の 3 行(guard の限界の行、テストの穴の行、`${…}` の止めすぎの行)と、plan の進捗のチェックボックスだった。verify の V-1(`stat` が抜けている)、V-2(発火した Trigger の持ち越しが書かれていない)、V-4(`\${` の形の probe)は tech-debt に反映した。V-3(Test plan の「AC1 の 5 形」)は plan の本文が承認 digest の範囲なので直していない。

DATACMD、NOEXEC、データ区間を書いた文書は、guard 本体(root と template)、テスト、tech-debt のほかになかった。guard を指す `.claude/rules/ralph/git-commit-strategy.md`、`internal/org/prompts/implementer.md`、`.codex/README.md` は、読むだけのコマンドを `echo` や `grep` の例で挙げるだけで、一覧は持たない。

差分の大きさ(`git diff origin/main...HEAD --stat`、349d838c 時点): 9 files changed, 518 insertions(+), 40 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(guard の限界の行) | 5 点を足した。(1) データ区間の定義の括弧と、条件の段落の 2 か所で、DATACMD の外のコマンドに `stat` を足した。理由は zsh の `stat -A NAME`(zsh/stat モジュール)が NAME の添字を評価すること。(2) 定義の末尾に 1 文足した: `${…}` の範囲もデータ区間ではない(`lex_dollar` の `xnote`、`msg_check` の `${` の検査、`lex_hd` の `HSUB`)。ただし範囲の外の文字、推奨の HEREDOC 形式のコミット、区切りに引用符のあるヒアドキュメントの本文はデータのまま。(3) (d) の行数を 1487 行(e77e2937)と awk の 1280 行(165 行目から 1444 行目)に直した。(4) Trigger の (d) と (e) に、この branch で発火して持ち越したことを足した(どちらも plan の範囲外)。(5) (e) の項目は変わっていないこと、テスト側で変わった 1 点(ヘッダーの B と group 10)を書いた。Related に plan と verify report を足した |
| `docs/tech-debt/README.md`(テストの穴の行) | この plan が固定した形を足した。B 節 group 10 の 11 行(`${…}` の形、`stat -A`)、`edge_none` の 8 行(`tr` 2 件、`${msg}`、`$msg`、`echo "${HOME}" 'sudo ls'`、推奨の HEREDOC 形式、`git commit -F -` の本文、`stat -f`)、`edge_deny` の 1 行(`git commit --message "${msg}$(id)"`)。(c) の awk の件数に 2026-10-09 の分を足した: 1948/0(ubuntu の mawk+jq と gawk+jq、macOS)、mawk で jq なしは 971/0。Trigger が発火したが、固定していない限界(blind spots、runner-string gap、`rg $x`、`ssh host`、`apt-get install sudo vim`、`\| sed`、字句解析だけの穴)は固定しなかったので、そのまま残ることを書いた。test report の Test gaps の 2 点(`${(` だけに狭める mutant を流していないこと、止めすぎを deny として固定していないこと)も足した。Related に plan と test report を足した |
| `docs/tech-debt/README.md`(`${…}` の止めすぎの行) | (a) の「`\${` の形は probe していない」を、probe した結果に直した(`git commit -m "mention \${HOME}; never sudo ls"` は 0931f791 で none、いまは deny、旧版も deny)。Why deferred の「単一引用符の中の `${`」を「単一引用符の中、または `\${` と書いた `${`」に広げた。Related に verify report(V-4)を足した |
| `docs/plans/active/2026-10-09-guard-zsh-data-gaps.md` | 進捗のチェックボックスだけを変えた。Implementation started の下に self-review・verify・test・sync-docs の 4 行を足し、Review / Verification / Test artifact created にチェックを付けた。「PR created」は `/pr` が付けるのでそのまま。本文は触っていない。`./scripts/plan-visual.sh digest` は変更の前後とも `7a6840f0ccaf`(plan の `- Approved:` 行と一致) |
| `docs/insights/events/2026-10-09-guard-zsh-data-gaps.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`) |
| `docs/reports/sync-docs-2026-10-09-guard-zsh-data-gaps.md` | この report |

tech-debt の「Findings of the last `/cross-review` run of fix/guard-deny-only」の行(取り消し線の行と、その上の RESOLVED コメント)と、`${…}` の止めすぎの行の本文(a)〜(d)は、コードと合っていたので変えていない(下の probe)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `git grep` で DATACMD、NOEXEC、`datacmd_list`、`noexec_list`、「data region」、「data command」、「read-only command」、`stat -A`、`zsh/stat`、`${(e` を検索(`docs/reports`、`docs/plans`、`tests/fixtures` を除く。パターンは `scratchpad/zg/sd1/pat1.txt`) | 当たったのは `.claude/hooks/pre_bash_guard.sh` と `templates/base/.claude/hooks/pre_bash_guard.sh`(同じバイト列)、`tests/test-pre-bash-guard.sh`、`docs/tech-debt/README.md` の 4 つだけ。guard のヘッダーの (a)〜(d) と各関数のコメントは、この branch の差分で `stat`、`${…}`、`tr` に合わせてある(差分を読んで確かめた) |
| 同じ範囲で `pre_bash_guard`、`sentinel`、`wrapper scan`、`${...}`、`tr "sudo"` を検索(パターンは `sd1/pat2.txt`) | 設計を書いた文書は他になかった。`.claude/rules/ralph/git-commit-strategy.md` の 69 行目(と template)は「`git` でも読むだけのコマンドでもない」と言うだけ。`internal/org/prompts/implementer.md` の 27〜30 行目は `echo` や `grep` を例に挙げるだけ。`.codex/README.md` の 114〜118 行目(と template)は deny だけを返すことを書く。どれも DATACMD の一覧を持たないので変更なし |
| `AGENTS.md`、`README.md`、`CLAUDE.md`、`.claude/rules/ralph/` の `git-commit-strategy.md` 以外、`.claude/skills/`、`.agents/skills/`、`docs/quality/` | 上の 2 つの検索に当たらなかった。guard の内部の記述はない。変更なし |
| 2 つ目の検索のその他の当たり | hook の配線(`PreToolUse.d/10-pre-bash-guard.sh` と template)、`internal/cli/` の migrate のコードとテスト(guard のパスを挙げるだけ)、`docs/specs/2026-04-16-ralph-cli-tool.md` の 269 行目(`ralph upgrade` の出力例で guard のファイル名を挙げるだけ)、`docs/evidence/` の過去の記録。guard の判定の説明はどれもない。変更なし |
| `internal/cli/migrate.go` | 77 行目が `./.claude/hooks/pre_bash_guard.sh` を挙げる(guard の限界の行の (d) が「77 行目」と書く箇所)。今も 77 行目。この差分は `internal/` を変えていない |
| tech-debt の「Findings of the last `/cross-review` run」の行(取り消し線)と RESOLVED コメント | 書かれた直し方(`${…}` を範囲ごと外す、メッセージの語の `${` を外す、`lex_hd` で `${` を置換あり扱い、`stat` を外す、`tr` を `noexec_list` に足す)が、コードの差分と合う。取り消し線の中の (a)〜(c) は解消前の記述として残す(行は経緯を残す方針) |
| tech-debt の `${…}` の止めすぎの行 | 下の probe で (a)〜(d) の例が base と旧版と HEAD で書かれたとおり。(a) の `\${` の形だけ、書き方を直した |
| root と template の guard、`check-sync.sh`、`check-skill-sync.sh` | `cmp` で同じ。`check-sync.sh` PASS、`check-skill-sync.sh` 13 skill |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この差分は guard の本体とテストを変えただけで、hook の配線、skill、rule、script の追加・削除はない。該当なし |

### probe(2026-10-09)

`scratchpad/zg/run.sh`(HEAD の guard と旧版 `tests/fixtures/guard-1c4cea5a/`)と `scratchpad/zg/sd1/base-run.sh`(base 0931f791 の guard と `lib_json.sh`、`cmp` で `git show origin/main:` と同じことを確認)。判定は jq あり/なしの順。

| 形 | base | HEAD | 旧版 |
|----|------|------|------|
| `git commit -m 'mention ${HOME}; never sudo ls'` | none/none | deny/deny | deny |
| `git commit -m "mention \${HOME}; never sudo ls"` | none/none | deny/deny | deny |
| `git commit -m 'use ${HOME}'` | (流していない) | none/none | none |
| `git commit -m "${msg}"` | none/none | none/none | none |
| `git commit -m "$msg"` | (流していない) | none/none | none |
| 推奨の HEREDOC 形式、本文に `${HOME}` と `sudo ls` | (流していない) | none/none | deny |
| `cat <<EOF` の本文 `${HOME} sudo ls` | (流していない) | deny/deny | deny |
| `cat <<'EOF'` の本文 `${HOME} sudo ls`(引用符つきの区切り) | none/none | none/none | deny |
| `echo ${x:-'sudo ls'}` | none/none | deny/deny | deny |
| `echo "${HOME}" 'sudo ls'` | none/none | none/none | deny |
| `echo 'sudo ls'; stat f`、`stat f; echo 'sudo ls'` | none/none | deny/deny | deny |
| `stat -A 'arr[$(sudo id; echo 1)]' /dev/null` | none/none | deny/deny | deny |
| `stat -f %z file` | (流していない) | none/none | none |
| `tr "sudo" "abcd"`、`echo x \| tr "sudo" "abcd"` | deny/deny | none/none | none |
| `echo ${(e):-'$(sudo ls)'}` | none/none | deny/deny | deny |
| `git commit -F -` の本文 `use ${HOME} here` | (流していない) | none/none | none |

## Found but left

- verify V-3: plan の Test plan は「AC1 の 5 形」と書くが、AC1 は e40087ef で 8 項目 11 形に広がった。テストは 11 形とも入っている。Test plan は承認 digest の範囲なので直さず、plan の進捗(verify の行)に書いた
- guard の限界の行の冒頭にある、行数の経緯の文(「…took it to 1461」)は、S2c から cap-4 の run までの履歴なので変えていない。現在の行数は (d) に書いた
- guard の限界の行の本文にある、2026-10-08 に probe した個々の形(許可リスト、`printf`、`rg`、`--no-verify` など)は、再実行していない。この差分が触るのは `${…}`、`stat`、`tr` の経路だけで、それらの形は書かれた commit での結果のまま
- (e) のコメントの項目は直していない。この plan の範囲は `${…}` の形と `stat`・`tr` の一覧だけで、(e) の Trigger は持ち越しとして記録した
- `docs/tech-debt/README.md` の 3 行以外には触れていない

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | 13 skill(s) in lock-step |
| `./scripts/run-static-verify.sh` | rc 0(tech-debt README plan references OK、`check-template-purity.sh` PASS、gofmt ok、0 issues、branch secret scan `0931f791..349d838c` clean。編集の後に実行) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-09-guard-zsh-data-gaps.md` | `7a6840f0ccaf`(編集の前後で同じ) |
| tech-debt の表の形(スクラッチの `tdcheck.py`) | 変えた 3 行とも 7 つの区切り(5 列)、列ごとのバッククォートは偶数、新しい `file:line` 参照なし。変えた行は 160、163、166 だけ(`HEAD` との行ごとの比較) |
| `bash run.sh` / `base-run.sh` の probe | 上の表 |

## Not verified

- zsh の振る舞い(`stat -A` の添字の評価、`${(e)…}` の再評価)は、この sync では流していない。tech-debt に書いた `stat -A` の説明は、test report の「Regression checks」(`zsh -f` 5.9 で 4 つの形の置換が走ったこと)と plan の Assumptions による
- guard のテストスイートは流していない(`/test` が 1948/0 を確認済み)。tech-debt に書いた件数(1948/0、971/0)は test report からの転記
- tech-debt の「mutation 10 件」「1 行で固定されている条件」は test report の記述を写した。mutation は再実行していない
- 「ほかの文書に DATACMD・NOEXEC・データ区間の記述はない」は、上の語で grep した範囲の確認である。別の言い方で書かれた記述は拾えていない可能性がある。未確認です
