# sync-docs report: split-preamble

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-split-preamble.md`(承認の digest `f0735c9a1ac6` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run は計画に触れていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `a7bdc884`、branch は `fix/split-preamble`。run 開始時の HEAD は `18c93ff0`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-22-split-preamble.md`(Merge OK。LOW 4 件は b09b7c12 と c0ff8398 で直った)、`docs/reports/verify-2026-10-22-split-preamble.md`(pass。INFO I1〜I3)、`docs/reports/test-2026-10-22-split-preamble.md`(pass。G1・G2 は 7a955d2c で埋まり、末尾の「この報告のあとに埋めた穴」に書いてある)
- この run が変えたのは、この report だけ。コード、雛形、skill、rules、仕様、根拠の表、tech-debt、計画、先行 3 report には触れていない
- 差分の大きさ(`git diff a7bdc884...HEAD --shortstat`、この run の前): 27 ファイル、+1117/-96。walkthrough を書くかどうかは `/pr` が決める

## Summary

`a7bdc884...18c93ff0` の差分は、分割計画の前文(`## Features` より前の `## ` の節)を `SplitPlan.Preamble` に読み、`ralph org start --plan` が作る leader のタスクに、「全機能の座席に適用する決まり」で始まる 1 行の前置きと、前文をバッククォートのフェンスで囲んだブロックとして入れる(`internal/org/split.go`、`internal/org/feature.go` の `featureLeaderTask` と `fenceFor`)。leader の雛形の手順 1・3・5 は、そのブロックを機能の計画に写し、書かれた driver と model で座席を立てるよう指す。`fix/seat-wip-checkpoint` は記録の中で `#250` に直った。

文書はこの branch の S3・S4 がすでに合わせていて、verify が突き合わせ済みだった。この run は、コードの読み(`parseSplitPlan`、`featureLeaderTask`、`fenceFor`、leader.md の手順、start の help)と、依頼が挙げた文書の言い分を突き合わせ直し、食い違いを見つけなかった。直したものはない。

## Changes made

| File | Change |
|------|--------|
| `docs/reports/sync-docs-2026-10-22-split-preamble.md` | 新規(この report) |

足した文にホームのパス、token はない。`git diff --check a7bdc884...HEAD` は rc 0、差分の U+FFFD は 0 件。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md`(repo map の `internal/org/` の行)、`templates/base/AGENTS.md`、`.ralph/core/AGENTS.core.md` | 変更なし | 行は「split plans and their approval digest (`split.go`)」と「the per-feature `ralph org start --plan` (worktree + bound leader with the recorded `[org.approval].mode`, `feature.go`)」で、ファイルの役割の地図。ファイルの境界は変わらず、前文はその中の 1 項目なので、すでに長い行に足さない。後 2 つの写しには `internal/org/` の行も `start --plan` もない |
| `README.md` | 変更なし(この branch が 252 行を直済み) | 「The leader's task holds the body of the feature's section and the plan's preamble (its `## ` sections before `## Features`, fenced) ...」が `featureLeaderTask` と合う。124 行の verb 表の `start` の行はタスクの中身を書かない |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 変更なし | Status の PR は 33 本(`#250` を数え、F5 のブランチ名は数えない)。python で数えると、Status の一覧は 33、根拠の表の「段ごとの PR」も 33 で、どちらも一致する。FR-4 の「試行の所見 F5 の直しで決めたこと」は、タスクの順、フェンスの長さ(最長のバッククォートの並び + 1、最低 3)、空の前文で何も足さないこと、渡さない 3 つの箇所と拒否しないこと、手順 1・3・5 の言い分を、コードと leader.md と合って書いている。印の数(26/6)は変わらない |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし | `start --plan`、分割計画、`{{TASK}}`、leader のタスクの中身を書いた文がない |
| `.claude/rules/ralph/agent-messaging.md`(と template の写し) | 変更なし | 2,000 字の上限は agmsg の本文と `escalate` の本文の規則。leader のタスクは role prompt のファイルの `{{TASK}}` に入り(`prompts.go` の `strings.NewReplacer`)agmsg を通らないので、上限は前文にかからない。leader.md の手順 5 が「TASK の本文は 2,000 字まで」と書くのは、leader から座席への TASK の規則で、この rule と合う |
| `/org` skill 4 写し | 変更なし | 分割計画の節(例の `## 座席の決まり`、前文の説明、渡らない 3 か所、leader が写すこと、前文がリポジトリの履歴に入ること)、start の手順 6 のタスクの順(分割計画・機能・worktree・ブランチ・予約・依存・承認方式・進め方・台帳の行、前文のブロック、機能の本文)が `featureLeaderTask` の並びと一致。Leader 運用の手順 1・2 が leader.md と合う。「`## Features` の外の節は自由に書ける」の古い文は、skill・README・仕様・docs のどこにも残っていない(`grep` で 0 件)。動詞表の `start` の行、`stop`・`disband` の cell には hunk が当たっていない |
| `/director` skill 4 写し | 変更なし | 「分割計画と機能ごとの org」の手順 1 の 1 文だけが変わり、`/org` skill と合う |
| skill の写しの一致 | 一致 | `cmp` で `.claude/skills/{org,director}/SKILL.md` と `templates/base/.claude/skills/` の写し、`.agents/skills/` と `templates/base/.agents/skills/` の写しが、どちらもバイト一致 |
| `ralph org start --help`(`internal/cli/org.go`) | 変更なし | 長い説明は「its task from the feature's body and the plan's preamble (its ## sections before ## Features)」と書き、`--plan` に task の引数を渡したエラーも「the feature's body and the plan's preamble are its task」。`--plan` と `--feature` のフラグの説明は、タスクの中身を書かない。`TestOrgStartHelp_TaskSource` と `TestOrgStartPlan_FlagCombinationsRefused` が 2 つの文を固める |
| `docs/tech-debt/README.md` | 変更なし | F5 の行は取り消し線と RESOLVED の注(322〜323 行)で閉じ、新しい open の行(347 行)が「ralph は前文に従ったかを確かめない」と、渡らない 3 か所を書く。注が挙げるテスト 7 本(`TestLoadSplitPlan_Preamble`、`TestLoadSplitPlan_TrialPreamble`、`TestFeatureLeaderTask_Preamble`、`TestFenceFor`、`TestStartFeature_LeaderPromptHoldsPreamble`、`TestRenderRolePrompt_Leader_FeatureOrgSeatRules`、`TestOrgStartHelp_TaskSource`)と SIGTERM の競合の行のテスト名は、どれも `func` が実在する。コミット SHA は 5 つとも branch の履歴にある。注の「前文は試行の分割計画で `## 目的` から『予約の外のファイルは編集しない』の行まで」は、`docs/evidence/org-trial-2026-10-20/splits/trial.md` を読んで合っている。F-8 の行(`/org` skill の `stop`・`disband` cell の長さ)は、cell が変わっていないので数字も変わらない。この branch が触れたファイルを `file:line` で引く行は新しくない |
| `docs/evidence/org-spec-evidence-2026-10-21.md` | 変更なし | F5 の行の「直す PR」の欄、F11 の欄の `#250`、PR 一覧(33 本)、「確かめていないこと」の F5 の項目(Go のテストだけで確かめたこと、確かめていないこと)が、tech-debt の行と実装に合う。F5 の行が引く句 `"~~`ralph org start --plan` puts only the feature's own section~~"` は、323 行の頭の 1 行だけに当たる |
| `docs/recipes/` | 変更なし | 7 本のどれにも `split plan`・`start --plan` がない |
| `docs/quality/quality-gates.md`(と template の写し)、`docs/architecture/repo-map.md`、`docs/quality/definition-of-done.md`、`.codex/README.md` | 変更なし | 「Split plan approval」の行は digest の比較と worktree の再利用の検査を書き、タスクの中身を書かない。ほかの 3 つに `split.go`・`feature.go`・split plan の語がない |
| コード内の doc(`internal/org/prompts.go` の `RolePromptVars`、`spawn.go` の `SpawnParams.Task`、`feature.go` の `StartFeature`、`split.go` の先頭と `parseSplitPlan`) | 変更なし | `RolePromptVars` の「`strings.NewReplacer` の 1 回の走査」は実装と合い、verify の I1 を 3b21b51b が直した。`Task` の 2 つの doc は `featureLeaderTask` に触れる |
| `fix/seat-wip-checkpoint` の残り | 記録だけ | 記録でない場所(仕様、根拠の表、tech-debt、skill、rules、コード)に残るものはない。残るのは archive の計画、seat-wip-checkpoint の先行 report と walkthrough、この branch の計画と report で、どれも当時の記録か、この PR が直す作業の説明 |
| `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh` | rc 0、rc 0 | 下の Verification |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| self-review の L1〜L4(LOW) | 4 件とも b09b7c12 と c0ff8398 で直っている(verify の Documentation drift の表が確かめ済み)。この run は触れていない |
| verify の I1(INFO)`RolePromptVars`・`Task`・`SpawnParams.Task` の古い doc | 3b21b51b が直した。コードの `NewReplacer` と合う |
| verify の I2(INFO)計画の印と Progress | 8303bd52 が直した。AC1〜AC8 は `[x]`、Progress は「PR created」だけが `[ ]` |
| verify の I3(INFO)根拠の表に F5 の直しの項目がない | 直ってある(「確かめていないこと」の F5 の項目) |
| test の G1・G2 | 7a955d2c が埋めた。上の「この報告のあとに埋めた穴」の節のとおり |
| test の「実機で leader が前文に従うか」 | 計画の Non-goals。tech-debt の open の行(347 行)が、ralph が確かめないことと一緒に書いている |

## Plan drift notes

- 計画の Affected areas は `internal/org/prompts.go` と `internal/org/spawn.go` を挙げないが、3b21b51b がこの 2 つの doc コメントだけを直した(verify の I1)。コードは変えていない。計画の本文は触らない決まりなので、ここに書くだけにした
- 計画にない tech-debt の行が 1 本ある(33e51acc)。`TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` の SIGTERM の競合で、この diff と関係なく、計画の Progress の「Test artifact created」に書いてある
- AC9(`GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0)は `[ ]` のまま。orchestrator がこのあと打つので、この run は印を付けていない

## Left open

- 既存の分割計画で前文を背景のつもりで書いていた作者には、渡る中身が増える。計画の Risks は release のノートに書くとしている。repo に CHANGELOG はなく、書くのは release の仕事で、PR の本文には `/pr` が書ける
- 仕様・根拠の表・tech-debt の `fix/split-preamble` は、PR の番号が決まるまで置き換えられない。次の PR が書く(4c の `fix/seat-wip-checkpoint` と同じ運び)。置き換え先は、仕様の Status の一覧の F5 の行(24 行)、根拠の表の「試行の所見」の本文(16 行)、F5 の行の「直す PR」の欄(24 行)、「段ごとの PR」の F5 の行(103 行)、「確かめていないこと」の F5 の項目(113 行)、tech-debt の 322 行の RESOLVED の注の頭と 323 行の RESOLVED の結びの括弧。このとき PR の一覧は 34 本になる
- 前文が機能の計画に写されたか、座席が前文の driver と model で立ったかは、実機で見ていない(Go のテストは leader の rendered role prompt まで)。tech-debt の 347 行が持つ

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `cmp`(`org`・`director` の skill、`.claude/skills` と `templates/base/.claude/skills`、`.agents/skills` と `templates/base/.agents/skills`) | 4 組とも一致 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-22-split-preamble.md` | `f0735c9a1ac6`(承認の digest と同じ) |
| `go vet ./internal/org/ ./internal/cli/`、`gofmt -l internal/org internal/cli` | どちらも出力なし。Go ファイルには触れていない |
| `git diff --check a7bdc884...HEAD`、U+FFFD の数 | rc 0、0 件 |
| テスト | 打っていない(`/test` の範囲。orchestrator が `run-verify.sh` と `run-test.sh` を打つ) |

## Insight event

追記していない(依頼が orchestrator に任せた)。打つなら次の 1 本。

```
./scripts/insights-append.sh --slug split-preamble --flow standard --phase sync_docs --verdict pass --cycle auto --source skill
```
