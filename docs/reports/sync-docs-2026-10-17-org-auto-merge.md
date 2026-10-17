# sync-docs report: org-auto-merge

- Date: 2026-10-18(JST。計画の日付は 2026-10-17)
- Plan: `docs/plans/active/2026-10-17-org-auto-merge.md`(この run で計画は触っていない)
- Pipeline cycle: 2。base は `15d22b1a`、branch は `feat/org-auto-merge`。run 開始時の HEAD は `668ce6d0`(run 開始時の値で、この report を含む commit の SHA ではない)。cycle 1 の sync-docs は `b5efa956`(V1・V2・I2・I4 の直しと、test の揺れの tech-debt の行。この report の前の版はそこにある)
- 先行 report: `docs/reports/self-review-2026-10-17-org-auto-merge.md`(cycle 2 の LOW 6 件は `31d27951`、Merge: yes。直しは S9)、`docs/reports/verify-2026-10-17-org-auto-merge.md`(`56003422`。pass。LOW の V3・V4 をこの run で扱う)、`docs/reports/test-2026-10-17-org-auto-merge.md`(`c7dfe662`。pass)、cross-review の直し S8(`9506ec64`、`b2f0b534`、`320b4d75`)
- コードの変更は `ralph org merge --help` の文 2 か所(`internal/cli/org_merge.go`)と、それを固定するテストの断片 2 つ(`internal/cli/org_merge_test.go`)だけで、動きは変えない。Edit で直した Markdown は 11 ファイル(`/org` skill と `/director` skill のそれぞれ 4 面、`README.md`、仕様、tech-debt)、この report。skill の `.agents` 側と template 側も、`scripts/sync-skills.sh` を使わず 4 面とも同じ文字列で Edit した

## Summary

verify の LOW 2 件を直した。V3 は、`--wait-checks` が「ほかの機能で走っている org は待たない」と、5 つの面の 8 か所が例外なしに書いていた食い違い。実際は、結びつき(binding)のない gate の記録だけのブランチでは `orgRunsOn` が立たない(`collectMergeTargets`、`internal/org/mergeorder.go`)ので、その org がほかの機能を走らせていても step 8 は待つ種類で断り、`--wait-checks` は org の disband か時間切れまで待つ。safe 側に倒れる動きで、tech-debt の 298 行が限界として書いている。8 か所に「結びつきのあるブランチでだけ見分ける。gate の記録だけのブランチでは、disband か時間切れまで待つ」の趣旨を足した。V4 は、tech-debt 243 行の Trigger の欄が classic の条件を S8a の前の規則(admin でない caller か `enforce_admins`)のまま書いていた食い違いで、S8a のあとの規則(admin かどうかによらず `enforce_admins` が有効)に直した。

S8・S9・`425a8543` による食い違いを、ほかの面にも当てた。見つかったのは 1 つで、tech-debt 292 行の RESOLVED の注が `merge_wait_test.go` を 712 行と書いていた(`425a8543` が 33 行足して 745 行)。直した。ほかの面(AGENTS.md、quality-gates の 2 面、tech-debt の 236〜243、259、291〜299 行の名前と行数)は合っている。

## Changes made

| File | Change |
|------|--------|
| `internal/cli/org_merge.go`(`ralph org merge` の help、`--wait-checks` の段落) | V3。(1) org の待ちの終わりを書く「until the org is disbanded, runs another feature, or the ledger cannot be read」に、「(told apart only on a branch with a feature binding; on a branch with only a gate record, the wait goes on until the org is disbanded or the time is up)」を足した。(2) 待たないものを並べる「An org that runs another feature now, a check that failed ...」を、「An org that runs another feature now (on a branch with a feature binding), a check that failed ...」にした。足した分で 9 行を折り直した(`strings.Fields` で空白を畳んで比べるテストなので、折り目は意味を持たない) |
| `internal/cli/org_merge_test.go`(`TestOrgMerge_Help`) | 上の 2 つの文を固定している断片を、新しい文に合わせた(2 つの断片の語を足しただけで、固定する内容は弱めていない)。ほかの断片は触れていない |
| `README.md`(`ralph org merge` の段落、`--wait-checks` の文) | V3。2 か所: 「until the org is disbanded or runs another feature」の直後に help (1) と同じ括弧、「an org that runs another feature now is not waited for」に「(on a branch with a feature binding)」 |
| `.claude/skills/org/SKILL.md`(`merge` の行) | V3。2 か所: 「ほかの機能で走り出すか」の直後に「(結びつきのあるブランチでだけ見分ける。gate の記録だけのブランチでは、disband か時間切れまで待つ)」、「ほかの機能で走っている org」の直後に「(結びつきのあるブランチのとき)」 |
| `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 同じ 2 か所(4 面とも同じ文字列で Edit した) |
| `.claude/skills/director/SKILL.md`(`auto_merge = true` の手順 2) | V3。「ほかの機能で走り出した org は待たない」の直後に「(結びつきのあるブランチでだけ見分ける。gate の記録だけのブランチでは、disband か時間切れまで待つ)」を足し、skill の折り方(表示幅 76 まで)に合わせて 4 行に折り直した。`internal/org/director_skill_sync_test.go` が固定する断片(「org の disband を待つ」で終わる)の外 |
| `.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | 同じ 1 か所(4 面とも同じ文字列で Edit した) |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 の 8a-3a の注) | V3。org の待ちの文「ほかの機能で走り出した org は待たない」の直後に、skill と同じ括弧を足した |
| `docs/tech-debt/README.md`(243 行、`github_protection` の意味の行の Trigger の欄) | V4。「a strict classic protection with required checks has a caller who is not an admin of the repository or `enforce_admins` enabled」を「a strict classic protection with required checks has `enforce_admins` enabled, whether or not the caller is an admin of the repository (`enforceClassic` in `internal/org/mergeenforce.go`)」にした。296 行、help、README、skill、仕様の注と同じ規則。5 つの欄のまま、`|` の追加はない |
| 同(292 行、RESOLVED の注) | `internal/org/merge_wait_test.go` の行数を 712 から 745 にした。S9b のあとに `425a8543` がテストを 1 本(33 行)足した。800 行の file size の下であることは変わらない |

skill の 4 面(`director` と `org` のそれぞれ)は `cmp` で同一バイト。

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `internal/cli/org_merge.go` の help | 直した(V3) | S8 の 3 つ(classic の `enforce_admins`、読み直しの head、org の disband の待ち)は verify が確かめた通りコードと合う。例外の句だけ足した |
| `internal/cli/org_merge_order.go` の help | 合っている | 「now runs another feature, they name it for once that org is disbanded」は merge-order の分類の話で、`--wait-checks` の待ちには触れない |
| `README.md` | 直した(V3) | 動詞の表(124 行)と `ralph status` の行(125 行)は合っている。S8 の 3 つは 266 行に書いてある |
| `/org` skill(4 面) | 直した(V3) | `merge` の行以外の「merge してよい順」の org の待ち、classic の条件、regate のロックの段落は合っている(verify の確認のとおり) |
| `/director` skill(4 面) | 直した(V3) | pin の断片(`directorSkillAutoMergeSteps`)は触れていない。`DirectorSkill` のテストは通った |
| 仕様 FR-8 の注(`:78`) | 直した(V3) | S8 の 3 つは書いてある |
| `docs/tech-debt/README.md` | 直した(V4、292 行の行数) | 236〜243、259、291〜299 行のセルの数は 5(243、293、296、298 行を再確認)。292 行の注の行数は `wc -l` と合う: `merge_fake_test.go` 662、`merge_test.go` 626、`merge_run_test.go` 795、`merge_wait_test.go` 745。293 行の 877、1334、318、480、1238 も合う。291〜299 行が名指す関数とテスト(`mergeLsRemoteRefusal`、`mergeGHSaid`、`listOriginBranches`、`realMergeOrderExec`、`realMergeDetachedExec`、`readChecks`、`mergeNoChecksSentence`、`mergeRefusalOrgRunning`、`runningRefusal`、`orgWaitEnd`、`mergeTargetOrg`、`collectMergeTargets`、`regateBoundedGit`、`enforceClassic`、`CheckMergeEnforcement`、`TestRegate_FetchFailures`、`TestMergeGHSaid`、`TestCheckMerge_ReadsTheChecks`、S9b で動いた fake の `mxExec` ほか)は `internal/` に実在する。名前を変えたテスト `TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits` は tech-debt には出てこない(参照は `merge_test.go` の注と test report の側) |
| `AGENTS.md` の `internal/org/` の行 | 変更不要 | `merge.go`・`merge_run.go`・`merge_wait.go`・`mergeenforce.go` を持ち、`merge_wait.go` の説明は「for checks still pending and for the branch's org to be disbanded」で S8c と合う。この行はテストのファイルをどの module でも名指さないので、`merge_fake_test.go` は足さない |
| `docs/quality/quality-gates.md` と template の写し | 変更不要 | 見張りの門の行が `merge` を見張りなしで動く verb に挙げる。merge の確かめを門の行にする話はない(cycle 1 と同じ) |
| `docs/architecture/repo-map.md`、`.claude/rules/ralph/`、`docs/recipes/` | 変更不要 | `ralph org merge`・`--wait-checks` の語を持たない(grep で 0 件) |
| 絶対パスの `$HOME` | なし | 今回の差分に `/Users/` も `hiroki.yoshioka` もない。U+FFFD もない |

## Left open

- test report(`c7dfe662`)は HEAD `56003422` の run で、この run が足した help の文とテストの断片を含まない。足した分は `TestOrgMerge_Help` と `OrgMerge` の全体(下の Verification)で通したが、test report は記録なので書き換えていない。help の文はコードの一部なので、cross-review の前に test をもう一度走らせるかは orchestrator の判断
- V3 の動き(gate の記録だけのブランチで `--wait-checks` が disband か時間切れまで待つ)自体は変えていない。tech-debt 298 行が限界として持ち続ける。この run は、その限界を 5 つの面が書いていなかった食い違いを直した
- README は `enabled` について「`ralph upgrade` does not add the key, so an existing `ralph.toml` gets the default true」と言うが、`auto_merge` には書いていない。`templates/base/ralph.toml` のコメントは「An absent auto_merge loads as false」と言う。既存の `ralph.toml` に `auto_merge` を足すのは人で、release ノートに書く項目になる(7 段目の `[org.approval]` と同じ)。README には足していない(cycle 1 から変わらない)
- `/director` skill の Codex の文は、Codex の shell ツールの timeout の上限と background の実行の有無を確かめていない。regate の前景の文と `/plan`・`/cross-review` の「its own background run」の書き方に合わせただけ(cycle 1 から変わらない)
- test report の揺れ(`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`、tech-debt 299 行)の原因は未特定。再び落ちたらその行の Trigger に従う(cycle 1 から変わらない)
- cycle 1 で残した verify の I3(計画の受け入れ条件の checkbox)は、計画の AC1〜AC13 が `[x]` で順に並んでいるので直っている

## Verification

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | 終了コード 0。PASS(KNOWN_DIFF 5) |
| `director` と `org` の skill の 4 面 | `cmp` で同一バイト(`.claude` と `.agents`、root と template。どれも終了コード 0) |
| `gofmt -l internal` | 出力なし、終了コード 0 |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/cli/` | 終了コード 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/ -run 'OrgMerge' -count=1 -timeout 30m` | 終了コード 0。`ok  github.com/yoshpy-dev/ralph/internal/cli  57.503s`(`TestOrgMerge_Help` は単独でも PASS) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run 'DirectorSkill\|OrgSkill\|Skill' -count=1 -timeout 30m` | 終了コード 0。`ok  github.com/yoshpy-dev/ralph/internal/org` |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/config/ -count=1` | 終了コード 0。`ok  github.com/yoshpy-dev/ralph/internal/config` |
| `./scripts/run-static-verify.sh` | 終了コード 0。tech-debt の plan の参照 OK、gofmt ok、`0 issues.`、`secret-scan-branch: scanned 15d22b1a..668ce6d0 against origin/main: clean`、`All verifiers passed.`。出力は scratch に保存(repo の外)。evidence は gitignore の対象で、commit に入らない |
| `./scripts/secret-scan-branch.sh --strict` | 終了コード 0 |
| `git diff --check` | 出力なし、終了コード 0 |
| `docs/tech-debt/README.md` の 243、293、296、298 行 | 区切りの縦線で割った欄はどれも 5。U+FFFD なし |

`./scripts/verify.local.sh` と Go のテストの全体は走らせていない。変えたのは文書と skill の写しと help の文(と、それを固定する断片)だけで、上の範囲(help を固定するテスト、skill を固定するテスト、skill の lock-step、template の一致、tech-debt の plan の参照、secret scan)を通した。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Codex の director が `--wait-checks 60m` を打つ実機の動き、`TestRegate_FetchFailures` の揺れの原因、help の文を足したあとの `internal/cli` と `internal/org` の Go のテストの全体(上の 3 つの `go test` の範囲だけ通した)

## Insight event

orchestrator が足す行(この run では jsonl に書いていない)。`ts` は足す時刻、ファイルは `docs/insights/events/<UTC の日付>-org-auto-merge.jsonl`:

```
./scripts/insights-append.sh --slug org-auto-merge --flow standard --phase sync_docs --cycle 2 --verdict pass --source skill --critical 0 --high 0 --medium 0 --low 0
```

```
{"schema":1,"ts":"<UTC>","slug":"org-auto-merge","flow":"standard","phase":"sync_docs","cycle":2,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```
