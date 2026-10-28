# sync-docs report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(承認の digest `97abe345a188` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` に足した 2 行だけ)
- Pipeline cycle: 2 の回し直し(cap 3)。`.harness/state/standard-pipeline/cycle-count.json` の値は 2 のまま(`/cross-review` の決まりで、cap を上げても数え直さない)。この report は、cycle 2 の最初の sync-docs の report を置き換える。base は `622cdcfb`、branch は `perf/org-narrow-verification`。run 開始時の HEAD は `69b112f6`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(cycle 2 の 3 回目、LOW 3、Merge yes。R3-3 は 246c7432 で解消)、`docs/reports/verify-2026-10-27-org-narrow-verification.md`(`bbb02949`、pass。LOW 1 は F12 `d721ed7d` で解消、既知の渡し 1 はこの run で扱った)、`docs/reports/test-2026-10-27-org-narrow-verification.md`(`69b112f6`、pass。`run-test.sh` の rc 1 は前からある不安定な 1 本だけ)、`docs/reports/cross-review-triage-org-narrow-verification.md`(2 回目の ACTION_REQUIRED 2 件は F9・F10・F12 で解消)
- 差分の大きさ(`git diff 622cdcfb...HEAD --shortstat`、この run の前): 31 ファイル、+2641/-112。walkthrough を書くかどうかは `/pr` が決める

## Summary

2 回目の cross-review の指摘 2 件(狭めた Go の検査に timeout がなく `go test` の既定の 10 分で切られる、入れ子の `go.mod` の module を repo の root から打つと落ちる)を直した F9 `3c64d65d`、F10 `662d3e47`、246c7432、F12 `d721ed7d` のあとの文書の同期である。直しが変えたのは、go:embed の雛形 2 つ、`/org` skill の 4 つの写し、plan の雛形の 6 つの写し、2 つのテストだけで、座席が打つ検査の「触ったパッケージの gofmt・vet・test」という中身は変わっていない。そのため、仕様 2 つ、quality-gates、標準フローの slice の検査を書いた文書は古くなっておらず、変えていない(下の表)。

足したのは 4 つある。tech-debt の R2-5 の行に (e) R3-1 と (f) R3-2、不安定な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の行に 4 回目、計画の Progress の 2 行、この report の release ノートの行への補足である。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(381 行目、R2-5 の行) | 5 セルのまま、全セルに足した。1 列目: 見出しの括弧に「run 3、R3-1 と R3-2」を足し、(e) R3-1 と (f) R3-2 を足した。(e) は `internal/org/prompts_verification_test.go` が F9 で 800 行の目安を越えたこと(`3c64d65d^` で 799 行、`3c64d65d` で 889 行、246c7432 で 891 行、`69b112f6` で 908 行。F12 が `TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck` を伸ばした)と、Go のスライスの検査の定数 2 つ(`goSliceCheckTimeoutSentences`、`goSliceCheckNestedModuleSentence`)とそれを使うテスト 2 本が別ファイルに移せること。(f) は `tests/test-slice-verification.sh` が plan の雛形の 6 つの写しで主コマンドと入れ子の例の 2 つしか pin せず、`-timeout=30m` を `GOFLAGS` の先頭に置く理由の文と、root に `go.work` がなければ Go は入れ子の module に入らないという文は pin しないこと(org の雛形の同じ 2 文は (e) の定数が pin する)。2 列目: 影響を足した(ファイルが読みにくい、理由の文を消しても shell のテストが通る、ただしコマンドの pin が座席の動作を守る)。3 列目: 繰り越す理由(動作を変えず、直すと全段の回し直しになる)。4 列目: trigger に、`prompts_verification_test.go` に次にテストを足すとき、plan の雛形の Go の例を次に直すときを足した。5 列目: 計画の Progress、self-review の R3-1 と R3-2、`tests/test-slice-verification.sh` を足した。R3-3 は 246c7432 で直ったので足していない。`\|` は 6 個(5 セル)、`~~` なし、U+FFFD なし |
| `docs/tech-debt/README.md`(285 行目、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の行) | 5 セルのまま。1 列目に 4 回目を足した(この PR の cycle 2 の回し直しの `/test`、`69b112f6`。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` が rc 1 でこの 1 本だけが落ち、1 分の load average は 53.9 まで上がった。`internal/org` を `-count=1` で打ち直すと 620.7 秒で通り、`go test` の既定の 10 分を越えたが、スライスの検査が今持つ `-timeout=30m` の下だった)。3 列目は「3 回目と 4 回目も繰り越した」に、4 列目は 4 回目も trigger の前半に当たって繰り越した、に直した。5 列目は、`docs/reports/test-2026-10-27-org-narrow-verification.md` が今は回し直しの report で、3 回目の `-count=20` の数字は `1637025d` にあると注記した。テストは直していない。`\|` は 6 個、`~~` なし、U+FFFD なし |
| `docs/plans/active/2026-10-27-org-narrow-verification.md` | `## Progress checklist` の `- [ ] PR created` の前に 2 行を足した(cycle 2 の回し直しの test `69b112f6`、この run の sync-docs)。この節は approval digest の範囲の外で、digest は `97abe345a188` のまま |
| `docs/insights/events/2026-10-27-org-narrow-verification.jsonl` | `sync_docs` の event を 1 行追記した(cycle 2、`--verdict complete --source pipeline`)。既存のファイルへの追記 |
| `docs/reports/sync-docs-2026-10-27-org-narrow-verification.md` | この report(cycle 2 の最初の sync-docs の report を置き換えた) |

## Surfaces checked for drift

F9〜F12 が変えた範囲は `git diff --name-only 6a5bed1e..HEAD`(前回の sync-docs の commit から)で、雛形 2 つ、`/org` skill の 4 つの写し、`plan/template.md` と `feature-plan.md` の 6 つの写し、2 つのテスト、report と plan と insight event である。skill、hook、script、verb の追加と削除はない。

| Surface | 結果 | 根拠 |
|---------|------|------|
| `docs/specs/2026-08-01-org-runtime.md` の「2026-10-27 改訂」(a)(35 行目)、`docs/specs/2026-10-07-org-multi-org-director.md` の FR-7「2026-10-27 に決めたこと」(127 行目) | 変更なし(正確) | (a) は「Go なら触ったパッケージの gofmt・vet・test」、FR-7 は「触ったところを覆ういちばん狭い検査」と書き、コマンドの形は書かない。新しい形も触ったパッケージの gofmt・vet・test で、偽になった文はない。timeout と入れ子の module はコマンドの中身で、雛形のテストが照合している。verify が任意として挙げた (a) への 1 句(「test は言語パックと同じ `-timeout=30m` を `GOFLAGS` の先頭に置き、入れ子の module ではその root で打つ」)は足していない。古くなっていないものは変えない |
| `docs/quality/quality-gates.md`(82 行目)と `templates/base/docs/quality/quality-gates.md`(81 行目) | 変更なし | 「the slice's verification command from the leader's TASK and a mechanical self-check on the staged diff」はコマンドの形を書かない。この cycle の直しは触っていない |
| `docs/recipes/codex-seat-permissions.md` と `templates/base/` の写し | 変更なし | reviewer の動作は変わっていない。`check-sync.sh` で一致 |
| 標準フローの slice の検査: `.claude/rules/ralph/git-commit-strategy.md`(Validation Gate の 2)、`.claude/skills/implement/SKILL.md` の手順 6、`.claude/rules/ralph/model-routing.md`(handoff の表)、`.agents/skills/implement/SKILL.md`、`templates/base/` の写し | 変更なし | 4 つとも「the narrowest check that covers the files the slice touches」と書き、Go のコマンドは書かず、`docs/plans/templates/feature-plan.md` の `## Implementation outline` を指す。Go の形はその雛形の HTML コメント(39 行目)にだけあり、F10 が新しい形にした。指す先が変わっても、指す文は偽にならない。`git diff 622cdcfb...HEAD --stat -- .claude/rules .claude/skills/implement .claude/agents .codex/agents` は空。`git-commit-strategy.md` と `implement/SKILL.md` は root と template が `cmp` で同一。`model-routing.md` の root と template の差(109 行目)は前からの KNOWN_DIFF で、この cycle のものではない |
| `.claude/agents/implementer.md`、`.codex/agents/implementer.toml`、`templates/base/` の写し | 変更なし | `go vet` と `go test` の文字がない(`git grep` で確かめた)。標準フローの implementer の契約は plan の雛形のコマンドを使う |
| `.claude/rules/ralph/golang.md`(11 行目「`go test ./...` before completion」) | 変更なし | 完了の前の検査の話で、スライスごとの検査ではない |
| 裸の `go vet ./<dir>/ && go test` | 座席が写す文書には 0 件 | `git grep -E 'go vet \./[^ ]+ && go test'` は、この report を除いて 7 件当たり、すべて記録だった。計画の Implementation outline の S1〜S3 の verify の欄(3 件、承認済みで digest の範囲、スライスごとの `-run` つきの検査)、triage の経緯の説明、test report の対照 a0、verify report の説明、別の PR(#278)の walkthrough の例である。雛形、skill、agent 定義、plan の雛形、quality-gates、recipe、仕様には残っていない |
| `.claude/rules/ralph/agent-messaging.md`、`AGENTS.md`、`templates/base/AGENTS.md`、`.ralph/core/AGENTS.core.md`、`README.md`、`docs/architecture/repo-map.md` | 変更なし | 座席の雛形の中身と検査のコマンドは書いていない。verb・skill・hook の追加と削除はない |
| `/org` skill の 4 つの写し | 変更なし(一致) | F12 の句「Go なら「役割」の節の implementer の項の形で…」は 4 つの写しに 1 回ずつある。`check-skill-sync.sh` が 14 skill の一致を確かめ、`OrgSkill` のテストが通った |
| `docs/tech-debt/README.md` の 157 行目(FR-7 の gate の行) | 変更なし(正確) | この cycle の直しは reviewer の雛形にも座席が打つ検査の種類にも触れていない |
| tech-debt の計画参照 | OK | `docs/plans/active/` と `archive/` の参照は重複を除いて 93 件で、存在しないものは 0 件。`scripts/verify.local.sh` の `check_tech_debt_plan_refs` と同じ検査 |

## Decisions on the reports' remaining items

| 出どころ | 項目 | 扱い |
|----------|------|------|
| verify | 【LOW 1】`/org` skill の route (A) の leader の手順 3 に Go の形がない | F12 `d721ed7d` で直し済み。skill の 4 つの写しにその句があり、`TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck` が照合し、test の mutant m3 が落とした |
| verify | 【既知】tech-debt の R2-5 の行に (e) R3-1 と (f) R3-2。(g) は足さない | 足した(Changes made)。(g) の (a) は 246c7432 で、(b) の文の崩れも直った |
| verify | 【Info】仕様に timeout と入れ子の module の 1 句を足してもよい(任意) | 足していない(上の表) |
| verify | 【Info】雛形と言語パックの小さな違い 2 つ(`go env GOFLAGS` が空のとき値の末尾に空白が 1 つ残る、`go env` が失敗したときの戻りがない) | 変更なし。どちらも動作に害がない |
| verify | 【Info】plan の雛形の英語の文は "a timeout the caller set" のままで、org の雛形の「環境の `GOFLAGS` か `go env -w` で」と語が違う | 変更なし。指すものは同じで、(f) の pin を足すときに合わせられる |
| verify | 【Info】計画の Progress の consult の確かめは、root からの `go vet` が「cannot find main module」で落ちると書くが、go1.27.3 の文言は配置により「go.mod file not found in current directory or any parent directory」か「main module … does not contain package …」 | 変更なし。落ちるという結論は同じで、Progress の他の記録に影響しない。直すなら lead が決める(`cannot find main module` の語だけ) |
| self-review | R3-1、R3-2 | tech-debt の R2-5 の行に足した |
| self-review | R3-3 | 246c7432 で直し済み。足していない |
| test | `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の 4 回目 | 既存の行に足した(Changes made)。テストは直していない(計画の範囲の外。test report が別の変更を勧める) |
| test | Test gaps 1〜3、6(実機の確かめ、文があるかしか見ないこと、実際の座席の timeout、coverage) | tech-debt の行は足していない。cycle 1 の判断と同じで、コードのある負債ではなく、前の org の雛形の変更と同じ種類の確認の穴。実機の確認は director の仕様の FR-7 の `[ ]` が持つ |
| test | Test gaps 4(mutant は名指しの 4 つだけ、plan の雛形の理由の文を pin しない) | (f) に入れた |

## Plan drift notes

- 計画本文の Scope と Affected areas は、S1〜S5 のとおりに実行した。F1〜F12 と 246c7432 で足した分は Progress に記録がある。Design decisions と Risks は digest の範囲なので触れていない。cross-review 2 回目が指した 2 件は、計画の D1(「触ったパッケージの gofmt・vet・test」)を変えず、コマンドの形だけを変えた
- Implementation outline の S1〜S3 の verify の欄は、裸の `go test ./internal/org/ -run …` のままである。承認済みの digest の範囲で、そのスライスが実行した検査の記録なので変えていない。座席が写すコマンドではない
- 残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける。

## `/pr` と release に渡すこと

- 実機の org での確かめはない。leader が `VERIFY:`・`CHECKS:`・`RESUME:` の行を書くこと、implementer が入れ子の module のパッケージで `(cd <module> && …)` を選ぶこと、reviewer が `CHECKS:` の行のコマンドだけを打つこと、新旧の binary の座席が混ざったときの動きは、雛形の文のテストでだけ確かめた(test の Test gaps 1〜3)。新しい Go の検査の形は、使い捨ての module で `go vet` と `go test` を直接打って確かめている(timeout は既定で 30m、環境の 45m と `go env -w` の 50m が勝つ。入れ子の module は root から落ち、`(cd <module> && …)` で通る)。FR-7 は `[ ]` のままで、次の実行の記録で付け直す。
- 差分は 31 ファイル、+2641/-112(`git diff 622cdcfb...HEAD --shortstat`、この run の前)。walkthrough を書くかどうかは `/pr` が決める。
- release ノートに手で足す 1 行(CHANGELOG は編集していない): 「org の implementer 座席は、スライスごとに leader の TASK の `VERIFY:` の行にあるいちばん狭い検査で検証し、コミットの前に 5 項目の機械的な確かめを行う。いちばん狭い Go の検査は、言語パックの `-timeout=30m` を `GOFLAGS` の先頭に置いて 30 分の timeout を保ち(環境の `GOFLAGS` か `go env -w` で決めた timeout があればそちらが効く)、入れ子の `go.mod` の module では `(cd <module> && …)` でその root に入って打つ。この形は org の座席の雛形と `/org` skill、標準フローの plan の雛形(`docs/plans/templates/feature-plan.md` と `/plan` の `template.md`)で同じである。機能ごとの org のレビューの TASK は `run-static-verify.sh` だけを走らせ、フルのテストは PR の前の gate の TASK で 1 回だけ走る。」
- release ノートに足す補足。org の役割雛形は go:embed なので、新しい binary で spawn した座席から効く。古い binary で立てた org の座席は今の雛形のまま動き、裸の `go test` を打つため、10 分を越えるパッケージのスライスの検証は打ち切られうる。TASK に `CHECKS:` の行がないレビューでは、reviewer は今までどおり両方を打つ。レビューの `GATE: pass` は静的検査が通ったことだけを指す。`ralph org gate` と `ralph org regate` は変わらない。
- コミットの型は `docs:` が 30 件、`perf:` が 11 件、`fix:` が 4 件(この report のコミットを除く)。`.goreleaser.yml` の changelog は `^docs:`、`^test:`、`^chore:` を除くので、`perf:` の 11 件と `fix:` の 4 件(`3c64d65d`、`662d3e47`、246c7432、`d721ed7d`)が自動の changelog に載る。`fix:` の 4 件はこの PR の `perf:` の commit の直しで、同じ release に入る。手で書く release ノートでは新しい形の一部として扱い、既存の不具合の修正とは書かない。
- tech-debt の行の Related 列は `docs/plans/active/` を指す。`/pr` の `archive-plan.sh` が `archive/` に書き換える。
- `run-test.sh` の 1 回の実行は、前からある不安定な 1 本(上の tech-debt の行)で rc 1 になることがある。回し直しの `/test` はその 1 本だけが落ち、`internal/org` の全体の打ち直しは通った。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill' -count=1` | rc 0。`ok  github.com/yoshpy-dev/ralph/internal/org  0.650s` |
| `sh tests/test-slice-verification.sh` | rc 0。`PASS: 162 / 162`、`FAIL: 0` |
| `git diff --check` | rc 0、出力なし |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-27-org-narrow-verification.md` | `97abe345a188`(`- Approved:` 行の `sha256:97abe345a188` と同じ。Progress の編集の前後で同じ) |
| tech-debt の計画参照 | 存在しない参照 0 件(重複を除いて 93 件) |
| tech-debt の編集した行(285 行目と 381 行目) | `\|` 6 個(5 セル)、`~~` 0 個、U+FFFD 0 件。行数は 382 のまま |
| `/natural-japanese` の lint(この report) | 指摘は 1 件で、和欧文間の半角空白(16 行)。この repo の report の書き方のまま残した |
| テスト | 上のほかは打っていない(`/test` の範囲)。この run が変えたのは文書、計画の Progress、report、insight event だけで、コードとスクリプトには触れていない |
| insight event | `./scripts/insights-append.sh --slug org-narrow-verification --flow standard --phase sync_docs --verdict complete --source pipeline --cycle auto` を実行した。既存のファイルへの追記(cycle 2) |
