# sync-docs report: plan-visual-review

## Cycle 1

- Date: 2026-10-06
- Plan: `docs/plans/active/2026-10-05-plan-visual-review.md`
- Pipeline cycle: 1 of 2。差分は base `d7877756` から branch HEAD `d85e3ab1`(feat/plan-visual-review)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-06-plan-visual-review.md`(`5d7aa671`。F-1〜F-5 は `c4a66c0d` で修正済み)、
  `docs/reports/verify-2026-10-06-plan-visual-review.md`(`f557d5b9`。再実行後は partial-pass。V-1・V-2 は解消、残りは AC10 と AC12 の後半)、
  `docs/reports/test-2026-10-06-plan-visual-review.md`(`d85e3ab1`。pass、1,648 件)

## Summary

verify が残した AC10(`AGENTS.md`、`.ralph/core/AGENTS.core.md`、`README.md`、`ralph-workflow.md` が `/plan` の図解と承認に触れていない)を直した。root と `templates/base/` の両側を同じ文面で更新し、`AGENTS.md` の管理ブロックは `.ralph/core/AGENTS.core.md` と同じ 2 行の差し替えにした。あわせて、`/plan` の流れと `scripts/` を一覧している他の文書を見て、`plan-visual.sh` と承認の手順が抜けていた 3 つの文書(`repo-map.md`、`definition-of-done.md`、`codex-setup.md`)を足した。plan の本文には触れず、チェックボックスだけ付けた。

`/plan` と `/implement` の SKILL.md の記述(承認の記録、digest の範囲、3 択、`/pr` の `--attach` と省略条件)と文書の文面が食い違わないよう、SKILL.md を読み直して書いた。

## Changes made

| File | Change |
|------|--------|
| `AGENTS.md`(管理ブロック内) | Primary loop の 2 を「…creates plan, shows a visual review page and asks for approval: Approve / Needs changes」に、3 を「…checks the plan's approval digest…」に。ブロックの外の Repo map の `scripts/` に `` `/plan` visual review helper `plan-visual.sh` (`open` / `shot` / `digest`) `` を足した |
| `.ralph/core/AGENTS.core.md` | 管理ブロックの元。上と同じ 2 行の差し替え |
| `templates/base/AGENTS.md`(管理ブロック内)、`templates/base/.ralph/core/AGENTS.core.md` | 同じ 2 行の差し替え。template の `AGENTS.md` には Repo map がないので `scripts/` の追記はなし |
| `README.md` | Operating loop の 2(Plan)にビジュアルレビューページと承認、図解を省ける条件を足した。3(Implement)に承認 digest の確認、9(PR)に全体図の添付(`gh` が `--attach` を持つとき)を足した。README に template 側はない |
| `.claude/rules/ralph/ralph-workflow.md` と `templates/base/.claude/rules/ralph/ralph-workflow.md` | `/plan` の項に図解ページと承認(`Status: Approved` と `- Approved:` の digest)を、`/implement` の項に digest の確認と、未承認・承認後の変更のときに聞くことを、`/pr` の項に全体図の添付を足した。2 面は byte 同一 |
| `docs/architecture/repo-map.md` | Skills の `plan/` の行に図解ページと承認ゲートを足した。`scripts/` の一覧に `plan-visual.sh` を足した。Runtime state に `.harness/state/plan-visual/`(コミットしない)を足した。root だけの文書 |
| `docs/quality/definition-of-done.md` と `templates/base/docs/quality/definition-of-done.md` | 最初の項目を「Active plan exists and passed the `/plan` approval gate(`- Status: Approved` と一致する digest。または承認なしで続けると利用者が選び、plan に書いた)、または不要と明示された」にした。2 面は byte 同一 |
| `docs/recipes/codex-setup.md` と `templates/base/docs/recipes/codex-setup.md` | `$plan` の行に図解ページと承認を足した。続けて「Codex は番号つきの選択肢で承認を取る。画像を読めなければ自己チェックを省き、`## Visual review` に書く」の 3 行を足した(`plan/SKILL.md` step 10.c と 12.c の記述どおり)。2 面は byte 同一 |
| `docs/plans/active/2026-10-05-plan-visual-review.md` | チェックボックスだけ。`- [x]` を AC1〜AC11(AC2b を含む)と進捗の「Review / Verification / Test artifact created」に付けた。AC12 と「PR created」は付けていない。本文は変えていない |
| `docs/reports/sync-docs-2026-10-06-plan-visual-review.md` | この report |

AC に付けた印の根拠: AC1・AC2・AC2b・AC3・AC4・AC7・AC11 は test report の実行結果、AC5・AC6・AC8・AC9 は verify report の Met、AC10 はこの pass の編集。AC12 の後半は `/pr` で `--attach` を使ってみるまで確かめられない。

### 文面を決めた理由

- README の「図解を省ける条件」は `plan/diagrams.md` の「When to draw, when to skip」と同じ(3 種のどれかで、かつ interaction・state・data format・module boundary を変えない)。「または」と読めると、1 ファイルでも契約を変える plan が省けるように読めるので、「変えない」を文に残した。
- `definition-of-done.md` は、`/implement` step 4 が「続ける」を許すので、「承認が必須」とは書かず、「承認済み、または続けると選んだことが plan に書いてある」にした。
- `AGENTS.md` の管理ブロックは downstream の `ralph upgrade` で配られるため、`.ralph/core/AGENTS.core.md` と template 側 2 面も同じ文面にした。`check-sync.sh` のブロック比較(4 面)で一致を確認した。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `AGENTS.md`(root と template)と `.ralph/core/AGENTS.core.md`(root と template) | 管理ブロックの 4 面が一致(`check-sync.sh` の block-aware 比較は PASS)。ブロックの外は owner ごとに自由 |
| `README.md` | Quick start の `/plan`、`/spec → /plan → /implement`、`$plan`、Operating loop の mermaid はそのまま有効。説明文の 2・3・9 だけ直した。ディレクトリ図の `scripts/` は `etc.` で終わるので変更なし |
| `.claude/rules/ralph/subagent-policy.md`(root と template) | Planning 節に「visual review approval gate」が入っていて byte 同一(S3 で済み)。変更なし |
| `.claude/rules/ralph/post-implementation-pipeline.md` | パイプラインの順序は変えていない。`check-pipeline-sync.sh` が REFS 6 本すべて OK |
| `.claude/rules/ralph/planning.md` | 承認ゲートに触れていない。誤りではないので変更せず(下の Found but left) |
| `docs/quality/quality-gates.md` | CI のゲートは増えていない。`tests/test-plan-visual.sh` と `tests/test-new-feature-plan.sh` は `scripts/verify.local.sh` の `tests/test-*.sh` の列挙で走る。変更なし(root と template で KNOWN_DIFF のファイル) |
| `docs/plans/README.md`、`docs/plans/templates/feature-plan.md`(root と template) | テンプレートは S3 で `- Approved:` と `## Visual review` が入っていて byte 同一。README は `Status` の値を挙げていないので変更なし |
| `.codex/README.md`、`.codex/AGENTS.override.md` | `$plan` の記述はワークツリーの話だけで、誤りではない。変更なし |
| `CLAUDE.md` | 管理するのは manual-trigger skill の話だけで、`/plan` の手順は書いていない。変更なし |
| 引用元の step 番号(`implement/SKILL.md` の step 10 / 12、`pr/SKILL.md` の step 10.c、`cross-review/SKILL.md` の step 11.c) | verify で一致を確認済み。今回の編集は SKILL.md に触れていない |
| ミラー | `check-skill-sync.sh` は 13 skill 一致。今回は SKILL.md を編集していないので `sync-skills.sh` は流していない |

## Found but left

- `.claude/rules/ralph/planning.md`(root と template)は「承認ゲートを通す」とは書いていない。plan の中身の規則(目的、受け入れ条件、リスクなど)のファイルで、`/plan` の手順は SKILL.md と `ralph-workflow.md` が持つ。足すかどうかは判断が分かれるので、足していない。足すなら 1 行(`- Plans are approved at the /plan gate before /implement`)で、root と template に同じ文面を入れる。
- この pass で直した `repo-map.md`、`definition-of-done.md`、`codex-setup.md` は plan の Scope の「ドキュメント」の列挙(`subagent-policy.md`、`AGENTS.md`、`AGENTS.core.md`、`README.md`、`ralph-workflow.md`)と Affected areas にない。plan の本文は承認済みで変えられないので、そのままにした。PR 本文か Known gaps に「/sync-docs が 3 文書を追加で直した」と書く扱いが妥当。
- plan の Progress checklist の項目名「Visual review approved」は、テンプレートと `/plan` step 12.e の「Plan approved」と違う(verify の Observational checks に既出)。checklist は digest の対象外だが、既に承認の記録なので直していない。
- 未確認のまま(plan の Open questions と verify の Coverage gaps に既出で、この pass で状況は変わらない): Codex CLI が PNG を読めるか、`gh pr create --attach` の EMU アカウントと private repo での挙動、`xdg-open` が戻らない環境での `open`。AC12 の後半は `/pr` で確かめる。PR 本文に「upgrade した下流は `feature-plan.md` の advisory を取り込む」を書く(V-2、`/pr` の担当)。
- insight event(`./scripts/insights-append.sh --phase sync_docs`)は追記していない。`/sync-docs` の SKILL.md にも `doc-maintainer.md` にも追記の指示がなく、`docs/insights/events/2026-10-06-plan-visual-review.jsonl` には self_review・verify(2 行)・test の 4 行だけがある(直前の rename-work-skill と同じ扱い)。
- tech-debt の行は足していない。この pass で新たに見つかった、まだ記録のない負債はない。

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | exit 0。IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | exit 0。`13 skill(s) in lock-step` |
| `bash scripts/check-template-purity.sh` | exit 0。`PASS: no meta-repo-specific references found in templates.` |
| `bash tests/test-check-template.sh` | exit 0。55 passed、0 failed、0 skipped |
| `./scripts/check-template.sh` | exit 0。`Template structure looks good.` |
| `./scripts/check-pipeline-sync.sh` | exit 0。`all pipeline steps referenced`(`definition-of-done.md`、`README.md`、`AGENTS.md` を含む) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-05-plan-visual-review.md` | `d4918bfcec38`。plan の `- Approved:` の値と同じ(チェックボックスを付けた後) |
| `git diff --check` | exit 0 |
| `./scripts/run-verify.sh` | exit 0。local verifier の PASS 29、FAIL 0。gofmt ok、golangci-lint `0 issues.`、`go test` は internal/* がすべて ok、branch secret scan `scanned d7877756..d85e3ab1 against origin/main: clean`。ログ: `docs/evidence/verify-2026-10-06-032616.log`(gitignore 対象) |

`run-test.sh` はこの step では流していない。編集したのは文書と plan のチェックボックスだけで、Go と shell には触れていない。直前の test report が pass(`d85e3ab1`)。

## Diff size (for /pr)

`git diff d7877756...HEAD`(この pass の編集を含まない HEAD `d85e3ab1`)は 46 files changed、3,435 insertions、39 deletions。この pass の作業ツリーの編集(この report を除く)は 13 files、60 insertions、39 deletions。walkthrough を書くかどうかは `/pr` が決める。この step では書いていない。

## Files changed in this pass

- `AGENTS.md`
- `.ralph/core/AGENTS.core.md`
- `templates/base/AGENTS.md`
- `templates/base/.ralph/core/AGENTS.core.md`
- `README.md`
- `.claude/rules/ralph/ralph-workflow.md`
- `templates/base/.claude/rules/ralph/ralph-workflow.md`
- `docs/architecture/repo-map.md`
- `docs/quality/definition-of-done.md`
- `templates/base/docs/quality/definition-of-done.md`
- `docs/recipes/codex-setup.md`
- `templates/base/docs/recipes/codex-setup.md`
- `docs/plans/active/2026-10-05-plan-visual-review.md`(チェックボックスのみ)
- `docs/reports/sync-docs-2026-10-06-plan-visual-review.md`(この report)
