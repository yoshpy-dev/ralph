# sync-docs report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(承認の digest `97abe345a188` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の末尾に足した 3 行だけ)
- Pipeline cycle: 2(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2。この report は cycle 1 の report を置き換える)。base は `622cdcfb`、branch は `perf/org-narrow-verification`。run 開始時の HEAD は `1637025d`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(cycle 2 の 2 回目、LOW 5、Merge yes。R2-1〜R2-4 は F8 `28d83b96` で直した)、`docs/reports/verify-2026-10-27-org-narrow-verification.md`(`7458318d`、pass。LOW 1 と既知 1 をこの run に渡した)、`docs/reports/test-2026-10-27-org-narrow-verification.md`(`1637025d`、pass。`run-test.sh` の rc 1 は前からある不安定な 1 本だけ)、`docs/reports/cross-review-triage-org-narrow-verification.md`(cycle 1 の ACTION_REQUIRED 1 件は F4 以降で解消)
- 差分の大きさ(`git diff 622cdcfb...HEAD --shortstat`、この run の前): 24 ファイル、+2475/-105。walkthrough を書くかどうかは `/pr` が決める

## Summary

cycle 2 の直し(F4 `ffafc9d2`、F5 `a7d879d9`、F6 `854be9db`、`51cf59e2`、F8 `28d83b96`)で増えた契約のうち、文書に出ていなかった 2 つを足した。leader が止まったスライスを頼み直すときの本文の行 `RESUME:` と、implementer の片付けの境界(このスライスの変更だけを直し、前からあった変更には触れず、コミットしない 3 つの場合は BLOCKED)である。`docs/specs/2026-08-01-org-runtime.md` の「2026-10-27 改訂」(a)に詳しく、`docs/specs/2026-10-07-org-multi-org-director.md` の FR-7「2026-10-27 に決めたこと」に短く書いた。FR-7 の印は `[ ]` のまま。

tech-debt には 2 行を足し、既存の 1 行に 3 回目の記録を足した。足した 2 行は、テストの整理(R2-5)の行と、標準フローの implementer に同じ形の片付けの穴が残る行である。3 回目の記録を足したのは、不安定な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の行である。

そのほかの面(quality-gates と写し、recipe と写し、`agent-messaging.md`、AGENTS.md、README.md)は、cycle 2 の直しで古くなった文がなく、変えていない(下の表)。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-08-01-org-runtime.md` | 「2026-10-27 改訂」(a)の末尾に 6 文を足した。(1) 片付けの境界: 片付けはこのスライスの変更だけに行い、TASK ごとの最初の編集の前の `git status --porcelain` に出ていた変更(stage 済みを含む。「前からあった変更」)は書き換えも削除も unstage もしない。unstage はパスを指定した `git restore --staged -- <パス>` で、パスなしの `git reset` は使わない。(2) コミットせずに BLOCKED を返す 3 つの場合(このスライスの秘密情報を scope の中で消せない、前からあった変更に秘密情報がある、stage 済みの前からあった変更に scope の外のパスがある)。秘密情報はファイルと行で示し、値は書かない。(3) `RESUME:`: 本文の行(ヘッダではない)に、止まった TASK がコミットせずに残した scope の中のパスを並べる。複数のスライスの TASK ではスライスの名前で始まる行を 1 行ずつ。implementer は引き継ぎ、scope の外のパスは引き継がずに BLOCKED。(4) 秘密情報で止まったあとは、leader が送る前に `git diff HEAD -- <パス>` で値が消えたことを確かめ、消えていないとき、または前からあった変更にあるときは再開せず `ralph org escalate` で BLOCKED を上げる。どの文も `internal/org/prompts/implementer.md`(15〜31 行、61〜80 行)と `leader.md`(40〜60 行)の HEAD の文に対応する。履歴の記述は書き換えていない |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-7「2026-10-27 に決めたこと」の「implementer は…打つことはしない。」のあと、「詳細は…」の前に 3 文を足した。`RESUME:` の行(止まった TASK が残したパス、複数のスライスでは名前つき、引き継ぎ、scope の外は BLOCKED)、片付けはこのスライスの変更だけ(前からあった変更の定義)、コミットしない 3 つの場合。値の確かめと escalate の条件は「詳細は…」の先の org runtime の仕様に任せた。「印(2026-10-27)」が挙げるテストに、新しい 3 本(`prompts_verification_predate_test.go` の `…SelfCheckLeavesChangesThatPredateTheSlice`、`…Leader_ResendsAStoppedSliceWithAResumeLine`、`…Implementer_TakesOverThePathsOfAResumeLine`)を足した。FR-7 は `[ ]` のまま。2026-10-26 の項目と 7b の注は書き換えていない |
| `docs/tech-debt/README.md`(末尾に 2 行) | (1) R2-5(self-review の案 B を R2-5 に絞った行): 同じ 2 文を `TestRenderRolePrompt_Implementer_MechanicalSelfCheckBeforeEachCommit` と `…SelfCheckLeavesChangesThatPredateTheSlice` が 2 か所で固定していること、`prompts_verification_predate_test.go` に `RESUME:` の 3 本が入っていること、順序の照合が `phrasesInOrder` と手書きの `at = i` / `i < at` の 2 通りあること、テスト名が `前からあった変更`(`implementerPredatingChanges`)でなく「slice」と言っていること。R2-1〜R2-4 は F8 で解消済みと明記した。(2) 標準フローの implementer: `.claude/agents/implementer.md` と `.codex/agents/implementer.toml` の「unstage it and stop」、`/implement` 手順 7 の「delete a new untracked file that holds it」と inline の「unstage it, take it out of the worktree」、`model-routing.md` の「the orchestrator first takes the secret out of the worktree」に、前からあった変更を除く文がない。写しの全部を名指した。#261 で merge 済みで #262 の範囲の外。trigger は標準フローの implementer の契約か手順 7 を次に直すとき。Related は `docs/plans/active/` の計画(`archive-plan.sh` が `/pr` で `archive/` に書き換える)。どちらも `\|` は 6 個(5 セル)、`~~` なし、U+FFFD なし。行番号でなく関数名で引いた |
| `docs/tech-debt/README.md`(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の行、285 行目) | 1 列目に 3 回目を足した(この PR の cycle 2 の `/test`、`run-test.sh` が rc 1、1 分の load average が 204 まで、単独の `-count=20` で HEAD は 6/20、base `622cdcfb` は 11/20 失敗、`go test ./internal/org/ -count=1` の打ち直しは通過)。3 列目に、この計画も `director_cmd*.go` と `director_stop*.go` を変えないので 3 回目も繰り越した、と足した。4 列目の trigger に、3 回目が trigger の前半に当たったが繰り越した、test report は別の変更を勧める、と足した。5 列目に計画と test report を足した。5 セルのまま。テストは直していない |
| `docs/plans/active/2026-10-27-org-narrow-verification.md` | `## Progress checklist` の `- [ ] PR created` の前に 3 行を足した(cycle 2 の verify `7458318d`、test `1637025d`、sync-docs)。この節は approval digest の範囲の外 |
| `docs/insights/events/2026-10-27-org-narrow-verification.jsonl` | `sync_docs` の event を 1 行追記した(cycle 2、verdict pass)。既存のファイルへの追記 |
| `docs/reports/sync-docs-2026-10-27-org-narrow-verification.md` | この report(cycle 1 の report を置き換えた) |

## Surfaces checked for drift

cycle 2 の直しは、go:embed の雛形 2 つ(implementer と leader。reviewer は触っていない)、`/org` skill の 4 つの写し、テストだけを変えた。コマンドと、座席が打つ検査は変わっていない。

| Surface | 結果 | 根拠 |
|---------|------|------|
| `docs/quality/quality-gates.md`(82 行目)と `templates/base/docs/quality/quality-gates.md`(81 行目) | 変更なし | 「impl exit checks (the slice's verification command from the leader's TASK and a mechanical self-check on the staged diff)」と reviewer の行は、cycle 2 の直しと食い違わない。この行は片付けの境界にも `RESUME:` にも触れず、触れる必要もない。2 つの差は KNOWN_DIFF の「(see docs/tech-debt)」だけ |
| `docs/recipes/codex-seat-permissions.md`(218〜223 行目)と `templates/base/` の写し | 変更なし | reviewer の最初の動作(TASK が名指すコマンド、なければ両方)は cycle 2 で変わっていない。implementer の片付けは recipe の範囲の外。`check-sync.sh` で一致 |
| `.claude/rules/ralph/agent-messaging.md` | 変更なし | この rule は ralph が検証するヘッダの契約だけを書く。`RESUME:` は `VERIFY:`・`CHECKS:` と同じ本文の行で、ralph は読まない(計画の D4)。cycle 1 の判断のまま |
| `.claude/rules/ralph/model-routing.md`、`subagent-policy.md`(`templates/base/` の写しを含む) | 変更なし(tech-debt に記録) | 「mechanical self-check」の 5 項目の正本の文は標準フローの implementer の契約で、org の雛形の 5 項目と項目の並びが同じまま。`model-routing.md` の「the orchestrator first takes the secret out of the worktree」は標準フローの穴の一部なので、tech-debt の(2)の行に入れた。この PR は標準フローに触れない |
| `AGENTS.md`、`templates/base/AGENTS.md`、`.ralph/core/AGENTS.core.md`、`README.md` | 変更なし | `RESUME:`、「機械的な確かめ」、「前からあった変更」で `git grep` して、当たりなし。org の説明は verb と仕組みの一覧で、座席の雛形の中身は書いていない |
| `docs/architecture/repo-map.md` | 変更なし | skill、hook、script、verb の追加と削除はない |
| `docs/tech-debt/README.md` の 157 行目(FR-7 の gate の行) | 変更なし(正確) | cycle 1 の sync-docs が直した「runs the commands its review TASK names」と 2026-10-27 の繰り越しの注は、cycle 2 の直しが座席の打つコマンドを変えていないので今も正しい |
| `/org` skill の 4 つの写し | 変更なし(一致) | この run は触っていない。`check-skill-sync.sh` が 14 skill の一致を確かめ、`OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill` のテストが通った |
| tech-debt の計画参照 | OK | `docs/plans/active/` と `archive/` の参照 93 件に、存在しないものは 0 件。`scripts/verify.local.sh` の `check_tech_debt_plan_refs` と同じ検査 |

## Decisions on the reports' remaining items

| 出どころ | 項目 | 扱い |
|----------|------|------|
| verify | 【LOW】仕様 2 つに `RESUME:` の行と片付けの境界がない | 直した。org runtime の仕様の(a)に詳しく、director の仕様の FR-7 に短く書き、FR-7 の印はテストの名前を足したが `[ ]` のまま(Changes made) |
| verify | 【既知】tech-debt に R2-5 と標準フローの片付けの 2 行 | 足した。案 B は R2-1〜R2-4 が F8 で解消済みなので R2-5 に絞った(Changes made) |
| verify | 【情報】`implementer.md` の短い行(段落としては同じ文に読める) | 変更なし。意味は変わらない。コードと雛形は sync-docs の範囲の外 |
| self-review | 案 A(標準フローの片付け) | 足した(上の 2 行目)。案 A が挙げた面に `model-routing.md` を足した |
| self-review | 案 B(R2-1〜R2-5) | R2-5 だけ足した(同上) |
| test | `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の 3 回目 | 既存の行に足した。テストは直していない(この計画の範囲の外。test report が別の変更を勧める) |
| test | Test gaps 1〜4、6(実機での確かめ、文があるかしか見ないこと、新旧の binary が混ざる org、mutant の範囲、coverage) | tech-debt の行は足していない。cycle 1 の判断と同じで、コードのある負債ではなく前の org の雛形の変更と同じ種類の確認の穴。実機の確認は director の仕様の FR-7 の `[ ]` が持ち、この変更を含む binary での次の実行の記録で付け直す |

## Plan drift notes

- 計画本文の Scope と Affected areas は、S1〜S5 のとおりに実行した。F1〜F8 で足した分(`CHECKS:` の行、送り直しの上限、複数スライスの形、片付けの境界、`RESUME:` の行、テストの別ファイル化)は Progress に記録がある。Design decisions と Risks は digest の範囲なので触れていない。
- 計画の Scope に仕様の「2026-10-27 改訂」の節があり、(a)〜(d)を cycle 1 で書いた。この run は(a)に片付けと `RESUME:` を足したので、(a)は implementer の検証と確かめ、片付け、再開の 3 つを持つ。改訂の節の見出しと(b)〜(d)は変えていない。
- 残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける。

## `/pr` と release に渡すこと

- 実機の org での確かめはない。leader が `VERIFY:`・`CHECKS:`・`RESUME:` の行を書くこと、implementer が前からあった変更に触らず BLOCKED で止まること、reviewer が `CHECKS:` の行のコマンドだけを打つこと、新旧の binary の座席が混ざったときの動きは、雛形の文のテストでだけ確かめた(test の Test gaps 1〜3)。FR-7 は `[ ]` のままで、次の実行の記録で付け直す。
- 差分は 24 ファイル、+2475/-105(`git diff 622cdcfb...HEAD --shortstat`、この run の前)。walkthrough を書くかどうかは `/pr` が決める。
- release ノートに手で足す 1 行(CHANGELOG は編集していない): 「org の implementer 座席は、スライスごとに leader の TASK の `VERIFY:` の行にあるいちばん狭い検査で検証し、コミットの前に 5 項目の機械的な確かめを行う。機能ごとの org のレビューの TASK は `run-static-verify.sh` だけを走らせ、フルのテストは PR の前の gate の TASK で 1 回だけ走る。」
- release ノートに足す補足は cycle 1 のまま。org の役割雛形は go:embed なので、新しい binary で spawn した座席から効き、古い binary で立てた org の座席は今の雛形のまま動く。TASK に `CHECKS:` の行がないレビューでは、reviewer は今までどおり両方を打つ。レビューの `GATE: pass` は静的検査が通ったことだけを指す。`ralph org gate` と `ralph org regate` は変わらない。
- コミットの型は `perf:` が 11 件、`docs:` が 21 件(この report のコミットを除く)。`.goreleaser.yml` の changelog は `^docs:`、`^test:`、`^chore:` を除くので、`perf:` の 11 件だけが自動の changelog に載る。
- tech-debt の行の Related 列は `docs/plans/active/` を指す。`/pr` の `archive-plan.sh` が `archive/` に書き換える。
- `run-test.sh` の 1 回の実行は、前からある不安定な 1 本(上の tech-debt の行)で rc 1 になることがある。cycle 2 の `/test` はその 1 本だけが落ち、`internal/org` の全体の打ち直しは通った。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `go test ./internal/org/ -run 'OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill' -count=1` | rc 0。`ok  github.com/yoshpy-dev/ralph/internal/org  1.028s` |
| `git diff --check` | rc 0、出力なし |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-27-org-narrow-verification.md` | `97abe345a188`(`- Approved:` 行の値と同じ。Progress の編集の前後で同じ) |
| tech-debt の計画参照 | 存在しない参照 0 件(93 件) |
| tech-debt の編集した行(285 行目と末尾の 2 行) | `\|` 6 個(5 セル)、`~~` 0 個、U+FFFD なし |
| `/natural-japanese` の lint(2 つの仕様) | 追加した文への指摘なし。出たのは、ファイル全体への既存の指摘(和欧文間の半角空白、文末コロン、語彙)で、この repo の書き方のまま残した |
| テスト | 上のほかは打っていない(`/test` の範囲。この run が変えたのは文書、計画の Progress、report、insight event だけで、コードとスクリプトには触れていない) |
| insight event | `./scripts/insights-append.sh --slug org-narrow-verification --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を実行した。既存のファイルへの追記(cycle 2) |
