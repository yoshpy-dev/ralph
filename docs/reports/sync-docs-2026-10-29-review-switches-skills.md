# sync-docs report: review-switches-skills

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-review-switches-skills.md`(issue #266。承認の digest `3cf8dfa26cbf` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` に足した 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `9837f744`、branch は `feat/review-switches-skills`。run 開始時の HEAD は `3a094892`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-29-review-switches-skills.md`(baa2f400 と 1031e5f4 の更新、fd5daaaf の run 2。run 2 は Merge: OK)、`docs/reports/verify-2026-10-29-review-switches-skills.md`(f0290dc4、pass、LOW の V1〜V4)、`docs/reports/test-2026-10-29-review-switches-skills.md`(cd97750e、pass)
- 差分の大きさ(`git diff 9837f744...HEAD --shortstat`、この run の前): 61 ファイル、+1135/-292。walkthrough を書くかどうかは `/pr` が決める

## Summary

この branch は issue #266 の対応で、標準フローの `/plan` の計画助言と `/cross-review` が `[standard].plan_advisory` / `[standard].cross_review`(既定 false)に従うようにした。skill は repo の root で `ralph config get` を打ち、終了コードで分岐する。driver は skill を動かす CLI で、`RALPH_PRIMARY_CLI` が上書きし、相手の CLI を `pick_reviewer` が決める。飛ばした review は計画の Progress、triage report、PR 本文に残る。`ralph insights backfill` は skipped の cross-review を n/a として読む。

verify の LOW 4 件(V1〜V4)はすべて直した。sweep で、同じ branch が作ったずれをもう 1 件見つけて直した(`docs/recipes/codex-setup.md` の `origin/$BASE`)。Go のコード、計画の本文、`/implement` 以外の skill は触れていない。

## Changes made

| File | Change |
|------|--------|
| `README.md:135` | V1。`ralph config get` の行の「only an older ralph … as off」を、off にするのは 2 つの失敗(古い ralph の exit 1 と `unknown command "config"`、PATH に `ralph` がない exit 127)で、ほかの失敗(exit 2 と 3 を含む)は止まって報告する、に直した。verify の文案のまま。skill の 11.a と手順 0 の 4 分岐と合う |
| `docs/tech-debt/README.md:31`(`pick_reviewer` の行) | V2。Debt 欄と Related 欄の行番号を `scripts/xreview-helpers.sh:70-78` にした(`grep --line-number` で測った。`pick_reviewer()` は 70、`*)` の arm は 76、閉じ括弧は 78)。Impact 欄の「the only surviving caller is `/cross-review`'s auto-detect」を、呼び手が `/plan` 11.b(この branch で追加)と `/cross-review` 手順 2 の 2 つで、同じ 1 回の snippet(`claude` か `codex` から始め、`RALPH_PRIMARY_CLI` は小文字にして 2 つのどちらかのときだけ採る)で driver を決めるので、渡る値は `claude`/`codex` だけ、に直した。Trigger を「`pick_reviewer` を呼ぶ場所を足し、その場所が driver の引数を `claude`/`codex` に限っていない」に直した。**行は開いたまま**(下の「V2 の判断」) |
| `docs/tech-debt/README.md:154`(codex reviewer の残りの危険の行) | V3。冒頭の「`/plan`'s Codex advisory」を「`/plan`'s plan advisory when codex is the reviewer」にした。(c) を、`/plan` 11.c の claude の呼び出しは `--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk`(claude 2.1.304 で確かめた)で読む道具だけになったが、`/cross-review` の claude の道は `--permission-mode auto` のまま既定の道具・利用者の MCP・project の hook を持ち、この branch が足したのは `RALPH_WIP_CHECKPOINT=off` と前後の git の確かめだけで、`command`・`</dev/null`・20 分の watchdog・出力の file もない(self-review の L5)、に置き換えた。Trigger の「the first review of the claude reviewer's permission mode」を「…of `/cross-review`'s claude reviewer (its permission mode and tools)」にした。Related に self-review の L5 を足した |
| `docs/tech-debt/README.md:376`(RC11、watcher の `claude -p` が利用者の hook を走らせる) | verify の任意の項目。Trigger の「claude に flag か settings の上書きがあるかは未確認」に、recheck では未確認だったこと、この branch で `claude -p --restricted` が user・project・local の settings を読まないと確かめ(claude 2.1.304)`/plan` 11.c が使うこと、watcher は変えていないことを足した。行は開いたまま |
| `.claude/skills/implement/SKILL.md:59`(13.e)と `.agents/skills/implement/SKILL.md`、`templates/base/.claude/skills/implement/SKILL.md`、`templates/base/.agents/skills/implement/SKILL.md` | V4。「The review runs when `[standard].cross_review` is on and the reviewer CLI exists」に、「and the diff against the base branch is not empty」を足した。`.claude/` と `templates/base/.claude/` を手で直し、`.agents/` の 2 つは `./scripts/sync-skills.sh` の既定と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills` で作り直した |
| `.claude/rules/ralph/ralph-workflow.md:41-42` と `templates/base/.claude/rules/ralph/ralph-workflow.md` | V4。「and that CLI is available」に「(for `/cross-review`, also when the diff is not empty)」を足した。80 桁の折り返しに合わせて 4 行を組み直した。2 つの写しは同じ文 |
| `docs/recipes/codex-setup.md:65-67` と `templates/base/docs/recipes/codex-setup.md` | V4。「`claude` is available; otherwise」を「`claude` is available, and the diff is not empty; otherwise」にした。次の文の「`$plan`'s plan advisory pairs the same way」は、計画助言に差分の条件がないので、「with no diff condition」を足した |
| `docs/recipes/codex-setup.md:73-74` と `templates/base/docs/recipes/codex-setup.md` | sweep で見つけたずれ。「feeds the diff between `origin/$BASE` and `HEAD`」は、この branch の N1 の直しで claude の prompt が local の base(`git diff <base>...HEAD`、手順 4 が表示した値)に変わったので古くなった。「the local base branch (`git diff <base>...HEAD`)」にした |
| `docs/quality/definition-of-done.md:30` と `templates/base/docs/quality/definition-of-done.md` | verify の任意の項目。skip の理由を並べる文に「an empty diff」を足した。skill は空の差分でも `skipped (empty diff)` の report を書くので、並べた理由に欠けていた |
| `docs/plans/active/2026-10-29-review-switches-skills.md` | `## Progress checklist` に「Sync-docs report created」の行を足した(本文の節は触れていない。digest は同じ) |
| `docs/reports/sync-docs-2026-10-29-review-switches-skills.md` | この report |
| `docs/insights/events/2026-10-28-review-switches-skills.jsonl` | `sync_docs` の行を `insights-append.sh` で足した(file 名は UTC の日付で、self-review・verify・test の行と同じ file。追記した時刻は 23:46Z) |

### V2 の判断: `pick_reviewer` の行を開いたままにした

行が追っているのは、`pick_reviewer` の 3 つ目の arm(`*) printf 'codex'`)が、未知の driver を黙って codex の reviewer にする点である。arm はこの branch でも変わっておらず(`scripts/xreview-helpers.sh:76`)、呼び手が引数を `claude`/`codex` に限っているので今も届かない。

Trigger は「2 つ目の呼び出し場所を足し、その場所が driver の引数を先に検証していない」だった。2 つ目の呼び手(`/plan` 11.b)はこの branch で足されたが、引数を限っている(`case "$override" in claude|codex)` と、skill が書き換える `DRIVER=<claude|codex>` の固定値)ので、Trigger の後半は満たしていない。依頼の「2 つ目の呼び手で Trigger が当たった」は前半だけが当たっている。そのため RESOLVED にも「Trigger hit, not paid」の carry note にもせず、3 つ目の呼び手が同じ限定を省く場合を拾えるよう Trigger の言い方を直した。

### verify V4 のうち直さなかった文

verify が「必要条件の言い方で偽ではない」とした次の文は直していない。`AGENTS.md:37`、`.ralph/core/AGENTS.core.md:26` と `templates/base/AGENTS.md`、`templates/base/.ralph/core/AGENTS.core.md`、`docs/architecture/repo-map.md:52`、`.claude/rules/ralph/post-implementation-pipeline.md:13`、`.claude/rules/ralph/subagent-policy.md:19-21`(と写し)、README の `:97-100` と `:232`。どれも「only when … on and that CLI is available」と書き、空の差分を足していない文が偽になることはない。`AGENTS.md` は短い地図のままにしておく方針も踏まえ、語を足さなかった。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md:97-103` と `:232` | 変更なし(同期済み) | 「only when … is on (both default off) and that CLI is available」。必要条件の言い方 |
| `AGENTS.md:31`・`:37`、`.ralph/core/AGENTS.core.md`、`templates/base/AGENTS.md` | 変更なし(同期済み) | 手順 2 と 8 の括弧が switch、CLI、skip の記録を言う。`AGENTS.md` の管理ブロックと `AGENTS.core.md` は同じ文 |
| `.claude/rules/ralph/post-implementation-pipeline.md:13`、`subagent-policy.md:19-21`・`:79`(と写し) | 変更なし(同期済み) | 「`/cross-review` always runs」と「calls the reviewer only when …」 |
| `.claude/rules/ralph/model-routing.md:69-75`、`:148` | 変更なし(同期済み) | `:69-75` は plan skill も 3 つの変数を同じ fallback で読むと書く。`:148` の `defaults_sync_test.go` の行は「plan/cross-review SKILL.md fallbacks」と総称で、この branch が足した `{"plan", "RALPH_CLAUDE_REVIEWER_MODEL"}` の組を含むので書き足さない(root だけの写し) |
| `.codex/AGENTS.override.md:24-28` と `templates/base/.codex/AGENTS.override.md` | 変更なし(同期済み) | `/plan` と post-implementation pipeline の前に `RALPH_PRIMARY_CLI=codex`。2 つは byte 一致 |
| `.codex/README.md:63` の並列表(「Cross-model second opinion」の行) | 変更なし | 何のコマンドを呼ぶかの行で、条件は言わない。`/plan` の計画助言は載っていないが偽ではない |
| `docs/recipes/codex-setup.md:36` | 変更なし(同期済み) | Codex だけの機械で claude を呼ぶ計画助言と cross-review が走らない、は D5 の判別と合う |
| `.claude/skills/pr/SKILL.md:34` と `template.md:33-34`、`docs/reports/templates/cross-review-triage-report.md:9`(と写し) | 変更なし(同期済み) | 記録の語彙と PR の雛形の 4 値の対応は verify が一致を確かめている |
| `templates/base/ralph.toml:19-29`、`:253-256` | 変更なし(同期済み) | F1 で 127 を足してある。`[org.review]` 側の「later release」は #267・#269 まで正しい |
| `internal/cli/config.go:56-66`(`config get --help`) | 変更なし(情報) | exit 1 の行の「Treat that, and only that, as off」は exit 1 の中の話で、ralph 自身は 127 を返さない。127 の行を足すなら `internal/cli/config_test.go:228` の固定も直す。Go のコードは依頼の範囲外 |
| `scripts/xreview-helpers.sh` の `pick_reviewer` の見出しコメント、`scripts/ralph-config.sh` のコメント、`internal/org/driver/paneenv.go` | 変更なし(同期済み) | 今の呼び出しの説明と合う |
| `docs/tech-debt/README.md:32`(`count_triage_findings` の行)の `scripts/xreview-helpers.sh:86-` | 直さなかった | base の時点で関数は 89 行目で、すでに 3 行ずれていた。この branch で 90 行目になった。この branch が作ったずれではなく、開いた範囲(`86-`)の参照なので、次にこの行を触るときに関数名の引用に直す |
| 現在形の文書の `git grep` | 残りなし | `Codex advisory`・`silently skip`・`auto-detect`・「cross-review/advisory の近くの optional」・`origin/$BASE` を、README、AGENTS、`.ralph/core`、`docs/recipes`、`docs/quality`、`docs/architecture`、`docs/tech-debt`、`.claude/rules`、`.claude/skills`、`.codex` で打った。残りは codex の hook の話(`.codex/README.md:157`、`docs/recipes/codex-setup.md:32`)、別の話題の `silently skip`、解決済みの tech-debt の行、過去の計画の所見番号を引く Go のコメントと tests のコメント。`docs/evidence/`、`docs/reports/`、`docs/plans/archive/` の過去の記録は当時の状態の記録なので洗っていない |
| 文を固定するテスト | なし | 直した文(`The review runs when`、`cross-reviewer when`、`or the other CLI unavailable`、`diff between`)を `tests/`・`internal/`・`scripts/` で `git grep` した。0 件 |

## Release notes

(PR 本文の「リリースノート」にそのまま使う。`ralph.toml` は seed なので、`ralph upgrade` は既存の project の `ralph.toml` に表を足さない。#265 の release のノートの項 2・5 と合わせて読む。)

1. 標準フローの計画助言(`/plan` 手順 11)と `/cross-review` は、`ralph.toml` の `[standard].plan_advisory` と `[standard].cross_review`(どちらも既定 false)に従う。以前は相手の CLI があれば回っていたので、今の動きを続けたい project は `[standard]` の 2 つを true で書く(この repo は root の `ralph.toml` で 4 つの実施有無を true にしてある)。
2. skill は repo の root で `ralph config get <key>` を打つので、新しい binary が要る。古い ralph(exit 1 と `unknown command "config"`)か、PATH に `ralph` がない(exit 127)ときは off として扱い、飛ばしたことを記録する。それ以外の失敗(exit 2、3、ほかの exit 1)は止まって報告する。release 済みの v6.0.0 には `config` がないので、この版を入れるまで、設定を on にしても飛ぶ(`go install ./cmd/ralph` か次の release)。
3. driver が Codex のときも計画助言が走る。以前の計画助言は driver にかかわらず codex を呼んでいたので、driver が Codex でも codex が計画を見ていた。driver は skill を動かす CLI(Claude Code → `claude`、Codex → `codex`)で、`RALPH_PRIMARY_CLI` が `claude` か `codex`(大文字小文字を問わない)なら上書きし、`which` では決めない。reviewer はその反対で、`pick_reviewer` が決める。
4. 計画助言の claude の呼び出しは読む道具だけで動く。`--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk` で、command を走らせる道具、MCP、user・project・local の settings(hook を含む)を止め、`RALPH_WIP_CHECKPOINT=off` を渡し、呼び出しの前後で git の状態(HEAD、index、作業ツリーと未追跡)を比べて、違えば止める。`--restricted` は settings を読まないので、settings 経由の認証(`apiKeyHelper`、settings の `env` で Bedrock や Vertex を選ぶ構成)の環境で claude が認証できるかは未確認で、できなければ `incomplete` に落ちる。
5. `/cross-review` の claude の道(driver が Codex のとき)は `--permission-mode auto` のまま、`RALPH_WIP_CHECKPOINT=off` と同じ前後の git の確かめを足した。claude の prompt の base は、`origin/$BASE` ではなく local の base(手順 4 が `base=<base>` と表示した値)になった。`.codex/AGENTS.override.md` は `/plan` の前にも `RALPH_PRIMARY_CLI=codex` を渡すよう言う。
6. 飛ばした review は記録される: 計画の `Plan reviewed` の項目(`plan advisory: skipped (<reason>)`)、triage report(`Reviewer status: skipped (<reason>)`。空の差分も `skipped (empty diff)`)、PR 本文の Verification。`ralph insights backfill` は skipped の cross-review を n/a として読む(以前は `incomplete` だけを n/a にし、skipped の report は triage の件数だけで verdict を決めていた)。

自動の changelog(`.goreleaser.yml` は `^docs:`・`^test:`・`^chore:` を除く)に載る commit は、この branch の `feat:` 3 本と `fix:` 2 本である。

| commit | 題 |
|--------|----|
| `07cf95f8` | `feat: read skipped cross-reviews as n/a and check the root ralph.toml strictly` |
| `93dec2cc` | `feat: make the plan advisory ask the other model and follow [standard].plan_advisory` |
| `9e233825` | `feat: make cross-review follow [standard].cross_review and record skipped reviews` |
| `08b2ceba` | `fix: set the driver in the same shell as pick_reviewer and record every skipped review` |
| `e2d842cd` | `fix: give the cross-review claude prompt the printed base instead of a lost shell variable` |

`feat:` の 3 本は題が内容を言っている。ただし `[standard]` の既定 off で動きが変わることは題に出ないので、release のノートの 1 と 2 を先に読ませる。`fix:` の 2 本は、同じ branch の未 release の機能の欠陥を直したもの(`e2d842cd` は、空の差分の確かめと claude の prompt の雛形が、前の呼び出しの `$BASE` に頼る欠陥を直した。雛形の `origin/$BASE` は旧い版からの形)で、changelog に「fix」として出るため、利用者が不具合の修正と読みかねない。release のノートの 3〜5 を先に読ませるか、手で整理する。`docs:` の commit(計画、self-review、verify、test、sync-docs の report と Progress の記録、文書の直し)は載らない。この run の commit も `docs:` である。

## Notes for reviewers(PR 本文に入れる点)

- Known gaps(依頼どおり据え置き): N2(driver の上書きの処理が skill の本文の 8 か所に同じ形で入った。テストがなく、`check-skill-sync.sh` は写しの一致しか見ない)、L5(`/cross-review` の claude の道が `command`・`</dev/null`・watchdog・出力の file を持たず、`/plan` 11.c の claude の形とそろっていない。tech-debt の `:154` (c) に書いた)。
- test の変異 a3(`After triage:` のあとの `pendingNoReview` のリセットを消す)は base でも生き残る既存の穴(test report)。
- 実行していない確かめ(verify の Not verified と同じ): skill を通しで動かした確かめ(Claude Code の `/plan` 11 が codex を呼ぶ道、Codex の driver の `/plan` 11 が claude を呼ぶ道、driver が Codex の `/cross-review` の claude の道)、R4(Codex の sandbox の中から `claude -p` が通るか)、settings 経由の認証と `--restricted` の組み合わせ。
- issue #266 の「やること」は判別の順を「`RALPH_PRIMARY_CLI`、`which`、`pick_reviewer`」と書くが、計画の D5 は `which` を外した(Codex の計画助言 MEDIUM 2)。受け入れ条件には影響しない。
- `ralph config get --help`(`internal/cli/config.go`)に exit 127 の行はない。ralph 自身は 127 を返さないので誤りではない。足すなら `internal/cli/config_test.go` の固定も直す別の変更になる。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。root と `templates/base/` の写しを同じ文で直したので byte 一致のまま |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-template-purity.sh` | rc 0。`PASS: no meta-repo-specific references found in templates.` |
| `git diff --check` | rc 0、出力なし(commit の前、追跡ファイルの変更に対して) |
| `bash tests/test-slice-verification.sh` | rc 0。`PASS: 162 / 162`、`FAIL: 0`(`/implement` の文を固定するテスト。13.e の直しのあとに打った) |
| `./scripts/sync-skills.sh`(既定と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`) | どちらも rc 0。変わったのは `implement/SKILL.md` の 2 つの `.agents/` の写しだけ |
| tech-debt の 3 行(31、154、376)の整合 | 変わった行は 3 行だけ(`git show HEAD:` と比較)。どれも 5 列(`(?<!\\) \| ` で分割)で、列ごとの backtick は偶数、`~~` は偶数。`file:line` の参照で変わったのは行 31 の `xreview-helpers.sh` の 2 か所(`66-73` → `70-78`)だけ。行 154 の `probe.go:26`、行 376 の `:34`・`:54` は前から同じ |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-29-review-switches-skills.md` | `3cf8dfa26cbf`(`- Approved:` 行の値と同じ。Progress の編集の前後で同じ。`git diff --numstat` は 1 行の追加のみ) |
| 打っていない | 全体の `go test`、`run-verify.sh`、`run-static-verify.sh`、`run-test.sh`。依頼で止められた。この run が変えたのは文書、skill の 2 行(と写し)、計画の Progress、この report、insight event で、コードの動きは変わらない |
| insight event | `./scripts/insights-append.sh --slug review-switches-skills --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記済み(rc 0、`2026-10-28-review-switches-skills.jsonl` の末尾の 1 行。`git status --short` に未追跡の file はない) |
