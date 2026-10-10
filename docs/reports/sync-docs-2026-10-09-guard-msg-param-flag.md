# sync-docs report: guard-msg-param-flag

## Cycle 1

- Date: 2026-10-09
- Plan: `docs/plans/active/2026-10-09-guard-msg-param-flag.md`
- Pipeline cycle: 1。差分は merge base `c3a9242e` から branch HEAD `a9d255bd`(fix/guard-msg-param-flag)まで。コードは `1e032dea` と `43e73568`、コメントの直しは `899fff16`
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-09-guard-msg-param-flag.md`(`26a22ee3`。Merge yes、LOW 3 件)、
  `docs/reports/verify-2026-10-09-guard-msg-param-flag.md`(`6157d685`。partial-pass、V-1〜V-4)、
  `docs/reports/test-2026-10-09-guard-msg-param-flag.md`(`a9d255bd`。pass、2,032/0、mutation 22 個のうち 16 個が red)

## Summary

古くなっていたのは tech-debt の 3 行(guard の限界の行、PR #211 の止めすぎの行、この PR の新しい止めすぎの行)と、plan の進捗だった。self-review の LOW 1 と LOW 2、verify の V-1 と V-2 は tech-debt に反映した。LOW 3(コメント)は 899fff16 で直っていて、guard とテストには触れていない。V-3(guard のヘッダーの (b) が `$"…"` を挙げていない)と test report の test gaps(M20、M13 の系統)は、guard とテストの変更になるので、tech-debt の (e) に持ち越しとして書いた。V-4(Scope の「`edge_none` に移す」に当たる行がなかった)は plan の進捗に書いた。

DATACMD、NOEXEC、データ区間を書いた文書は、guard 本体(root と template)、テスト、tech-debt のほかになかった。guard を指す `.claude/rules/ralph/git-commit-strategy.md`(と template の写し。`cmp` で同じ)、`internal/org/prompts/implementer.md`、`.codex/README.md` は、読むだけのコマンドを `echo` や `grep` の例で挙げるか、deny だけを返すことを書くだけで、一覧は持たない。

差分の大きさ(`git diff origin/main...HEAD --stat`、この report を足す前の a9d255bd 時点): 9 files changed, 825 insertions(+), 90 deletions(-)。docs 以外(guard の root と template、テスト)は 3 files changed, 376 insertions(+), 87 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(guard の限界の行) | 11 点を直した。(1) Debt 列のデータ区間の定義から「`msg_check` の `${` の検査」を外し、1e032dea 以降は「展開する `$` を持つメッセージの語」(`msg_check` が語の印 `WEXP` を読む)と「引数にある zsh の添字の `$` から語の終わりまで」(`lex_word` の `xnote`。43e73568 から `$@[` と `$#x[` も)がデータ区間でないことを書いた。(2) 同じ列の rg の条件を、「`$'…'` か `$"…"` の語がない」から「展開する `$`・置換・ANSI-C か locale の引用のどれも語にない」に直した。(3) 「printf と rg は語を書かれたとおりに読む」を、rg は `lex_dollar` の印を読む、と直した。(4) Impact (a) の rg の止めすぎから、新しい止めすぎの行を指した。(5) Impact (b) の「rg が `--pre` と読む変数の語」を取り消し線にして、解消済み(1e032dea)と書いた。(6) Why deferred の「rg の変数の語は open」を、解消済みと書いた。(7) Trigger (b) の rg の変数の語の指示を取り消し線にして、解消済みと、固定先が B 節 group 11(`guard_deny_only_forms`)であることを書いた(`edge_sentinel_deny` ではない)。(8) Debt (d) の行数を、guard 1596 行(899fff16)、awk 1380 行(174 行目から 1553 行目)に直した。(9) Trigger (d) と (e) に、この branch でまた発火して持ち越したことを書いた(plan の Non-goals がファイルの分割を別の PR に回していること、Design decisions が 3 点に限っていること)。(10) Debt (e) に、この branch でも (e) の項目は変わっていないこと、新しい持ち越し 2 点(V-3、test gaps)を足した。PR B で足す。(11) Related に plan と 3 つの report を足した |
| `docs/tech-debt/README.md`(この PR の新しい止めすぎの行) | 冒頭の commit を「1e032dea and 43e73568」にした(特別なパラメータの添字は 43e73568 で入った)。(a) の範囲を、「展開する `$`」から「後ろが空白・終わり・閉じる二重引用符でない `$`」に直し、bash が文字のままにする `5$,` も含むことと、`git commit -m "costs 5$, never sudo ls"` の判定(c3a9242e で none、いまと旧版は deny)を書いた。(c) の例に `$#x[` を足した |
| `docs/tech-debt/README.md`(PR #211 の止めすぎの行) | Trigger の末尾に句点を足した(内容は変えていない) |
| `docs/plans/active/2026-10-09-guard-msg-param-flag.md` | 進捗のチェックボックスだけを変えた。Implementation started の下に self-review、899fff16(コメントの直し)、verify、test、sync-docs の 5 行を足し、Review / Verification / Test artifact created にチェックを付けた。V-4 は verify の行に書いた。「PR created」は `/pr` が付けるのでそのまま。本文は触っていない。`./scripts/plan-visual.sh digest` は変更の前後とも `ebf5a9ac3a15`(plan の `- Approved:` 行と一致) |
| `docs/insights/events/2026-10-09-guard-msg-param-flag.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、`--cycle auto`) |
| `docs/reports/sync-docs-2026-10-09-guard-msg-param-flag.md` | この report |

tech-debt の行は、行の番号ではなく関数・変数・commit の名前で書いた(`file:line` の参照は足していない)。(d) の awk の行番号だけは、この行がもともと持っているもので、899fff16 で測った。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `git grep` で DATACMD、NOEXEC、`datacmd_list`、`noexec_list`、「data region」、「data command」、「read-only command」、`msg_check`、`lex_dollar`、`WEXP`、`rg --pre` を検索(`docs/reports`、`docs/plans`、`tests/fixtures`、`docs/evidence`、`docs/insights` を除く。パターンは `scratchpad/sd/pat.txt`) | 当たったのは `.claude/hooks/pre_bash_guard.sh` と `templates/base/.claude/hooks/pre_bash_guard.sh`(同じバイト列)、`tests/test-pre-bash-guard.sh`、`docs/tech-debt/README.md` の 4 つだけ |
| 同じ範囲で `pre_bash_guard`、`sentinel`、`zsh subscript` を検索(`*.go` を除く) | 設計を書いた文書は他になかった。`.claude/rules/ralph/git-commit-strategy.md` の 69 行目(と template)は「`git` でも読むだけのコマンドでもない」と言うだけ。`internal/org/prompts/implementer.md` の 24〜33 行目は `echo` や `grep` を例に挙げるだけ。`.codex/README.md` の 114〜119 行目(と template)は deny だけを返すことを書く。`docs/specs/2026-04-16-ralph-cli-tool.md` は guard のファイル名を挙げるだけ。`scripts/secret-scan-branch.sh`、`docs/insights/README.md`、`tests/test-terraform-gitignore.sh` の `sentinel` は別の意味。どれも DATACMD の一覧や展開する `$` の扱いを持たないので変更なし |
| `.claude/rules/ralph/git-commit-strategy.md` と `templates/base/.claude/rules/ralph/git-commit-strategy.md` | `cmp` で同じ。この PR で変わる振る舞い(メッセージの `${` や展開する `$` の判定)は書いていない。変更なし |
| guard のヘッダーのデータ区間 (a)〜(d) と各関数のコメント | verify の確認のとおり、(a) の rg の条件、(b) の展開する `$`、添字の範囲は `stage_note`・`lex_dollar`・`lex_word` と合う。合わないのは V-3(`$"…"`)だけで、guard の変更になるので持ち越した |
| `AGENTS.md`、`README.md`、`CLAUDE.md`、`.claude/skills/`、`.agents/skills/`、`docs/quality/` | 上の 2 つの検索に当たらなかった。guard の内部の記述はない。変更なし |
| harness 内部の整合(skill、hook の配線、rule、language pack、script の一覧) | この差分は guard の本体、テスト、tech-debt の記録だけで、hook の配線、skill、rule、script の追加・削除はない。該当なし |
| tech-debt の PR #211 の止めすぎの行の (a) | 取り消し線と「Closed in fix/guard-msg-param-flag」の注記は b4646f85 のまま。(b)〜(d) の例は verify V-2 の確認のとおり(deny のまま、対照の 2 形は none)。Trigger の句点だけ直した |
| tech-debt のテストの穴の行 | rg の変数の語の「Not pinned」は b4646f85 で取り消し線になっている。この PR が足した行と test gaps は guard の限界の行の (e) に書いたので、この行は変えていない(verify は「ほぼ Yes」) |

### probe(2026-10-09)

`scratchpad/sd/probe.py`(HEAD の guard、base `c3a9242e` の guard と `lib_json.sh`、旧版 `tests/fixtures/guard-1c4cea5a/`)。判定は jq あり/なしの順。permission_mode は付けていない。tech-debt に足した形のうち、guard の判定を新しく書いたものだけを流した。

| 形 | base | HEAD | 旧版 |
|----|------|------|------|
| `git commit -m "costs 5$, never sudo ls"` | none/none | deny/deny | deny/deny |
| `git commit -m "use $1 never sudo ls"` | none/none | deny/deny | deny/deny |
| `git commit -m $"never sudo ls"`(V-3) | none/none | none/none | deny/deny |
| `echo $=arr['$(sudo ls)']`(M20 の形) | none/none | deny/deny | deny/deny |
| `echo $^arr[…]`、`echo $+arr[…]`(中は上と同じ) | none/none | deny/deny | deny/deny |
| `echo $@[…]`、`echo $#x[…]`(中は上と同じ) | none/none | deny/deny | deny/deny |
| `rg $x sudo pat .` | none/none | deny/deny | deny/deny |
| `rg 'foo$' 'sudo ' .` | deny/deny | none/none | deny/deny |
| `git commit -m 'mention ${HOME}; never sudo ls'` | deny/deny | none/none | deny/deny |
| `git commit -m "${msg}"` | none/none | none/none | none/none |

## Found but left

- V-3(guard のヘッダーの (b) が `$"…"` を挙げていない)と test gaps(`=`・`^`・`+` の修飾の行がない、M13 の系統の行がない)は、tech-debt の (e) に持ち越した。直すと guard かテストが変わり、pipeline の再実行になる。PR B(guard のテストにも触れる)で直す想定
- verify V-4: plan の Scope は「`edge_sentinel_deny` などから `edge_none` に移す」と書くが、移す行はなかった。Scope は承認 digest の範囲なので plan の本文は直さず、進捗の verify の行に書いた
- plan の Objective 3 と Test plan が名前を挙げた `echo "$arr['$(sudo ls)']"` と `echo ${arr['$(sudo ls)']}` はテストの配列にないが、base でも deny(test report の Ad-hoc probes)。足すかどうかは PR B に任せる
- guard の限界の行の本文にある、2026-10-08 に probe した個々の形(許可リスト、`printf`、`--no-verify` など)は、再実行していない。この差分が触るのは `${…}`、rg、zsh の添字の経路だけで、それらの形は書かれた commit での結果のまま
- 引用符のない区切りのヒアドキュメントの本文の中の添字(self-review が未確認とした点)は、tech-debt に足していない。plan の Objective 3 の対象外で、確かめた人がいない
- `docs/tech-debt/README.md` の 3 行以外には触れていない

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | 13 skill(s) in lock-step |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-09-guard-msg-param-flag.md` | `ebf5a9ac3a15`(編集の前後で同じ。`- Approved:` 行の `sha256:ebf5a9ac3a15` と一致) |
| tech-debt の表の形(スクラッチの `tdcheck.py`) | 変えた 3 行(160、166、167)とも 7 つの区切り(5 列)、列ごとのバッククォートは偶数、取り消し線の `~~` は偶数、新しい `file:line` 参照なし。変えた行は 160、166、167 だけ(`HEAD` との行ごとの比較) |
| `python3 scratchpad/sd/anchors.py` | 編集前に、17 か所の old_string がそれぞれ 1 回だけ現れることを確かめた |
| `python3 scratchpad/sd/probe.py` | 上の表 |
| `./scripts/run-static-verify.sh` | rc 0(tech-debt README plan references OK、`check-template-purity.sh` PASS、check-skill-sync 13 件、gofmt ok、0 issues、branch secret scan `c3a9242e..a9d255bd` clean。編集の後に実行。この sync の commit は含まない。commit の後に `./scripts/secret-scan-branch.sh --strict` を流してから push する) |

## Not verified

- guard のテストスイートは流していない(`/test` が 2,032/0 を確認済み)。tech-debt に書いた mutation の名前と結果(M12、M13 の系統、M20)は test report からの転記で、mutation は再実行していない
- zsh の振る舞い(添字の評価、`$@[` や `$#x[` が添字になること)は、この sync では流していない。tech-debt に書いた説明は、plan の Assumptions、self-review の `dqsub.zsh`、test report による
- tech-debt の (d) の行数は 899fff16 で測った。a9d255bd 以降の commit はどれも docs だけなので、guard のファイルは変わっていない
- 「ほかの文書に DATACMD・NOEXEC・データ区間の記述はない」は、上の語で grep した範囲の確認である。別の言い方で書かれた記述は拾えていない可能性がある。未確認です
- PR B(guard の分割)の範囲と、そこで test gaps を足すことは、依頼の文面による。plan の Non-goals は「別の plan と PR で扱う」と書くだけで、PR B という名前は plan にない
