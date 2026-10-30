# sync-docs report: regate-ci-checks

- Date: 2026-10-30(JST。計画の日付は 2026-10-29)
- Plan: `docs/plans/active/2026-10-29-regate-ci-checks.md`(issue #264。承認の digest `994880bfa86d` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の 1 行だけで、本文の節と AC の checkbox は触れていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `origin/main` の `c03647f9`、branch は `feat/regate-ci-checks`。run 開始時の HEAD は `bcd143e2`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-29-regate-ci-checks.md`(run 2 まで。CRITICAL・HIGH・MEDIUM は 0)、`docs/reports/verify-2026-10-29-regate-ci-checks.md`(pass。LOW の V1〜V5、情報の I1〜I4)、`docs/reports/test-2026-10-29-regate-ci-checks.md`(pass。Test gaps 4 つ)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、この run の前): 58 ファイル、+6881/-349。walkthrough を書くかどうかは `/pr` が決める
- 制約: Go のコードは、help の文字列 2 か所と doc コメント 1 か所だけ変えた。振る舞いは変えていない。打ったのは gofmt、`go vet`、help と skill を固定するテスト、`check-skill-sync.sh`、`check-sync.sh`、`git diff --check`、`plan-visual.sh digest`。`run-static-verify.sh` と `run-test.sh` の全体は打っていない(verify と test が済ませた後の、文字列と文書だけの変更のため)

## Summary

verify が /sync-docs に渡した V1〜V5 をすべて直した。V1・V3 は regate の `UP_TO_DATE:` が `gate_passed` だけを数えるように読める文、V2 は quality-gates の「同じ 2 つのスクリプト」、V4・V5 は skill と README の「台帳ごとに 1 つ」「spawn と start が断られる」に `ci` の待ちの例外がないこと。

そのほか、V4・V5 と同じ抜けを 4 か所で見つけて直した。`ralph org spawn --help` と `ralph org start --help` が、regate の間は spawn と start が断られると書くだけで、`ci` の regate が `--wait-checks` で待つ間は `regate.lock` を外していて断られないことを書いていなかった(`regateSpawnErr` は `regate.lock` が空いていれば marker を読まない)。`/org` skill の spawn の行と start の行も同じだった。

## Changes made

| File | Change |
|------|--------|
| `internal/cli/org_regate.go`(help の Long、`UP_TO_DATE:` の段落) | V1。「a gate_passed of HEAD」を「a pass of HEAD on that tip (a gate_passed, or a gate_ci_passed that a ci regate recorded)」にした。`passedOn`(`internal/org/regate.go`)が local でも `gate_ci_passed` を UpToDate と読むため。印字する行(`printRegateLines`)は前から「a pass of HEAD」 |
| `internal/cli/org_regate.go`(`printRegateResult` の doc comment) | V3。「recorded a gate_passed or reached the push」を「recorded a pass (a gate_passed, or a gate_ci_passed) or reached the push」にした |
| `internal/cli/org.go`(`spawn` と `start` の `--help`) | 既存の文のあとに 1 文を足した。「A ci regate that waits for the PR's checks has released regate.lock and does not refuse it/them meanwhile (see ralph org regate --help).」。既存の文は変えていないので、それを固定するテストの断片はそのまま通る |
| `.claude/skills/org/SKILL.md` の `regate` の行 | V4。(a)「台帳ごとに 1 つ」に「`ci` の regate が checks を待つ間を除く」、(b)`UP_TO_DATE:` の条件を「`gate_passed` か `gate_ci_passed`」、(c) 行末の「spawn と start が断られる」に、待つ間はロックを外しているので断られず、待ちのあとにその org が走っていれば regate のほうが断るという例外。手順の節(`merge してよい順` の 2)と help の同じ例外の言い方に合わせた |
| `.claude/skills/org/SKILL.md` の `spawn` の行と `start` の行 | 「regate がその org のブランチで作業している間は断られる」の文に、括弧で `ci` の regate の待ちの例外を足した(既存の文は変えず、固定している断片のあとに足した) |
| `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 上の写し。`scripts/sync-skills.sh` で `.agents` を作り、`templates/base` の 2 つは `cp`(sync-skills は template を作らない) |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`(Quality pipeline gate の行の末尾) | V2。「runs the same two scripts, records the same events with `via=regate`, and pushes a pass」のあとに、`[org.gate].regate_checks = "ci"`(既定は `local`)ではスクリプトを走らせず、merge commit を先に push し、PR の checks から `gate_ci_passed` か `gate_ci_failed`(`via=regate_ci`)を記録し、checks が終わるまでは何も記録しない(exit 6)という文を足した。両方の copy は KNOWN_DIFF だが、この文は同じにした |
| `README.md`(`ralph org merge-order` の段落の regate の説明) | V5。(a)「before anything else it checks」を「right after its fetches, before the merge, it checks」にした。確かめるのは lock・結びつき・worktree の確かめと fetch のあとで、help の「Right after the fetches」に合わせた。(b)「One regate, merge, or cleanup runs per ledger at a time」の括弧に、待つ `ci` の regate は待つ merge と同じくロックを外すことを、spawn と start の断りの文の末尾に「except while a ci regate waits for the checks」を足した |
| `internal/cli/org_regate_test.go`(`TestOrgRegate_Help`) | 固定する断片を 1 つ足した(V1 の文) |
| `internal/cli/org_cleanup_test.go`(`TestOrgStartSpawnAndMergeOrder_HelpFollowTheCleanupAfterAMerge`) | `start` と `spawn` の help に足した文を固定する断片を 1 つずつ足した |
| `internal/org/org_skill_send_sync_test.go`(`orgSkillCIRegateRules`) | regate の行に 3 断片(台帳ごとに 1 つの例外、`gate_ci_passed`、行末の例外)、`spawn` の行と `start` の行の節を足した。断片は skill の中の順に並べた。doc コメントも直した |
| `docs/tech-debt/README.md`(`regate.go`・`regate_ci.go`・`org_regate_test.go` が 800 行を越えた行) | `internal/cli/org_regate_test.go` の行数を「760 → 1271」から「760 → 1273」にした(上のテストの断片が 2 行増やした)。`wc -l` で数え直した |
| `docs/plans/active/2026-10-29-regate-ci-checks.md` | `## Progress checklist` に「Sync-docs report created」の行を 1 つ足した |
| `docs/reports/sync-docs-2026-10-29-regate-ci-checks.md` | この report |
| `docs/insights/events/2026-10-30-regate-ci-checks.jsonl` | `sync_docs` の行を足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `ralph org regate --help` の PUSH_ONLY の条件(`internal/cli/org_regate.go` の「latest gate record is a gate_passed of HEAD on origin's default branch tip」)と、印字する `PUSH_ONLY:` の行 | 変更なし | verify の V1 の注どおり、字義どおりに合う。`ci` の pass は `regateHeadNotOnOrigin`(`regate_merge.go`)で push だけの打ち直しから外れ、断られるので、PUSH_ONLY になる記録は `gate_passed` だけ |
| `/director` skill(「merge のあとの取り直し」) | 変更なし | F2 が例外を入れていて、`TestDirectorSkill_PinsTheCIRegate` が固定する。終了コード 6 と `CHECKS:`・`NEXT:` の書き方も help と合う |
| 仕様 FR-7 の注(`docs/specs/2026-10-07-org-multi-org-director.md`、2026-10-30 の箇条)と 8a-3b の箇条 | 変更なし | 注は D1〜D7 と合う(待つ間に `regate.lock` を外すことも書いてある)。8a-3b の箇条は日付つきの履歴で、`local` の動きとして合う |
| `AGENTS.md` の地図(`internal/org/` の行) | 変更なし | 「`regate.lock` keeps a spawn and a start of that org out while it runs」は地図の 1 句で、例外は README と skill と help が持つ。AGENTS.md は短く保つ約束なので足さない。verify も合うと見た |
| `ralph org --help` の要約(「merges the default branch into a disbanded feature org's branch, runs the gate on it, and pushes it」) | 変更なし | `ci` では gate が PR の checks に変わり、push が先になるが、動詞の一覧の中の 1 文の要約で、テストが固定する前半は合う。細部は `ralph org regate --help` にある |
| `ralph org merge --help` と `ralph org cleanup --help` の「regate が断られる」「ロック」 | 変更なし | merge と cleanup の側から見た文で、`ci` の regate の待ちとは関係がない |
| `ralph org merge-order --help`、README の merge-order の段落の `gate_ci_passed` | 変更なし | S6・S7 が済ませていて、合う |
| `docs/recipes/`、`docs/architecture/repo-map.md`、`.claude/rules/ralph/`(`model-routing.md` を除く) | 変更なし | `git grep` で regate を挙げる行がない。`model-routing.md:151` の `[org.gate]` は S7 が足した |
| `docs/specs/2026-08-01-org-runtime.md:37` の注(「regate は今までどおり 2 つのスクリプトを走らせる」) | 変更なし | verify の I4 のとおり、日付のある注で、その仕様は履歴を書き換えない約束 |
| `templates/base/ralph.toml`、`ralph.toml`、`scripts/check-sync.sh` の KNOWN_DIFFS のコメント | 変更なし | verify が合うと確かめた |

## Commands run

| Command | Result |
|---------|--------|
| `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/` | rc 0 |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgRegate\|Help\|OrgStartSpawnAndMergeOrder'` | ok、99.1 秒。`TestOrgRegate_Help` と `TestOrgStartSpawnAndMergeOrder_HelpFollowTheCleanupAfterAMerge` の新しい断片を含む |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|Prompt'` | ok、28.3 秒。`TestOrgSkill_PinsTheCIRegate` が 4 つの copy すべてで新しい断片を見つけた |
| `./scripts/check-skill-sync.sh` | rc 0、14 skill が一致 |
| `./scripts/check-sync.sh` | PASS。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `git diff --check` | rc 0、出力なし |
| 変更した 14 ファイルの U+FFFD | 0 件 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-29-regate-ci-checks.md` | `994880bfa86d`(承認の行と同じ) |

固定するテストの断片が、直す前の文で落ちることは確かめていない(断片は新しい文を verbatim で持つので、文を戻せば落ちる作りだが、変異は打っていない)。

## Not done

- AC1〜AC10 の checkbox は付けていない。verify の AC1 と AC9 の「Partly」のうち、AC9 の文書のずれは V1〜V5 を直して解消した。AC1 の文言の変化(I1)は意図どおりで、Progress の S7 に記録がある。checkbox は `/pr` の前の段が付ける
- 本物の GitHub の CI での `ci` の regate は、仕様の注のとおり打っていないので、FR-7 の印は `[ ]` のまま

## Insight event

`./scripts/insights-append.sh --slug regate-ci-checks --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-30-regate-ci-checks.jsonl` に 1 行足した。
