# sync-docs report: org-cross-review

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(issue #269。承認の digest `f30958e4f38f` は編集の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。計画の編集は `## Acceptance criteria` の印 10 個、artifact の印 3 個、`## Progress checklist` の 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json`)。base は `origin/main` の `4fef6b8c`、branch は `feat/org-cross-review`。run 開始時の HEAD は `433df4ae`
- 先行 report: self-review(run 1 と run 2、Merge: yes)、verify(pass。LOW の V1・V2 は F5 `62d62be0` で直っている)、test(run 1 と run 2、pass。Test gaps は run 1 の 1〜5 と run 2 の 1〜4、残った変異は R1c〜R1e)。本物の CLI での確かめは `docs/evidence/xreview-live-probe-2026-11-01.md` の 2 回(586f7934 と 62d62be0)
- 制約: Go のファイルと役割の雛形(`internal/org/prompts/*.md`)は変えていない

## Summary

雛形の今の動き(`leader.md` の手順 9・12・14 と `### cross-review`、`reviewer.md` の「cross-review の TASK」)と文書を突き合わせた。F1〜F6 のあとで古くなっていたのは次の 4 か所で、どれも直した。

- 仕様の 2026-11-01 の注: implementer の driver の出どころ(手順 5 の spawn のまま)と、数える report(このブランチが足したものに限ると書いていない)
- `/org` skill: 終わり方 1 が driver の読み方と数える report を書いていない。タスクの reviewer の行の段落が、計画助言だけが使うように読める
- `.claude/rules/ralph/model-routing.md` と写し: 相手のモデルを呼ぶ所の一覧に reviewer 座席の cross-review がない
- `templates/base/ralph.toml` の `[reviewer]` のコメント: 同じく呼び手の一覧に cross-review がない

tech-debt には依頼の 2 行を足し(テストの穴 9 つ、座席の中での実行)、雛形の大きさの行の数を `433df4ae` で数え直し、ほかの 13 行に注か追記を足した。依頼の 2b(手順 9 のふつうのレビューの report の `git add --`)は、既存の行の項目 (o) が同じ点を持っていたので、新しい行にせずその行に注を足した。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md`(2026-11-01 の注) | `IMPLEMENTER_DRIVER:` の説明を「手順 5 で implementer を spawn したときの driver」から、記憶からは書かず TASK を作るたびに `ralph org status --json` で台帳の implementer の座席から読む、に(F6)。「数はコミット済みの report から読む」を、このブランチが base から足してコミットした report から読み、同じ org_id の前の org の report は数えない、に(F1・F4) |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 終わり方 1 の cross-review の段落に 2 文を足した(driver は TASK を作るたびに `ralph org status --json` で読み、読めなければ BLOCKED。数えるのはこのブランチが足した report だけ)。足した文のあとの行は段落の終わりの手前まで折り返し直した(表示幅 70〜78)。タスクの `- reviewer の codex:` と `- reviewer の claude:` の行の段落に、leader が cross-review の TASK の `REVIEWER_CODEX:` と `REVIEWER_CLAUDE:` に写すことと、行がなければ reviewer が incomplete で返すことを足した。root と template の `.claude` は Edit で直し、`.agents` の 2 つは `sync-skills.sh` を 2 回(root と template)打って作り直した。4 つは `cmp` で同じ |
| `.claude/rules/ralph/model-routing.md`、`templates/base/.claude/rules/ralph/model-routing.md` | Cross-review sync note に、reviewer 座席の cross-review も設定を読まず、leader がタスクの 2 行を TASK に写すことを足した。effort の段落の一覧に reviewer 座席の cross-review を足した。root だけの「Where the values live」に、xreview のテスト 2 ファイルが何を固定しているかの 1 行を足した(テストの名前で書いた) |
| `templates/base/ralph.toml`(`[reviewer]` のコメント) | 呼び手の一覧を「/plan, /cross-review, and a feature org's plan advisory and cross-review」に。leader が `ralph org start --plan` の読んだ値を cross-review の TASK で reviewer 座席に渡すことを足した。キーのコメント行(`# claude_model = ...` など)は変えていない |
| `docs/tech-debt/README.md` | 新しい 2 行と、既存の 14 行への注・追記(下の「tech-debt」) |
| `docs/plans/active/2026-11-01-org-cross-review.md` | AC1〜AC9 と AC5b、Review・Verification・Test artifact created に印。Progress の Implementation started の下にこの pass の 1 行 |
| `docs/reports/sync-docs-2026-11-01-org-cross-review.md` | この report |
| `docs/insights/events/2026-11-01-org-cross-review.jsonl` | `sync_docs` の行(cycle 1、pass) |

## 依頼の項目ごとの結果

| 項目 | 結果 |
|------|------|
| 1 文書と雛形の一致 | ずれは上の 4 か所で、直した。既知の仕様 133 行の driver の文も直した。項目ごとの照合は下の「F1〜F6 の照合」と「Surfaces checked」 |
| 2a テストの穴 | run 1 の 1〜5 と run 2 の 1〜4 を (1)〜(9) として 1 行に足した。(5) は座席の中での実行で、2c の行を指す |
| 2b 手順 9 の `git add --` | 新しい行にしなかった。org-gate の cycle 3・4 の小さい所見の行(「Small findings of org-gate cycles 3 and 4」)の (o) が、手順 9 と `/org` skill の終わり方 1 のレビューの report のコミットが「名前のパスだけをステージする」と書かないことを、すでに持っている。この branch は手順 9 と終わり方 1 を変えたのでその行の Trigger に当たり、注を足した(F5 が範囲外として見つけ直したこと、この branch の cross-review の report のコミットは確かめた 1 つのパスを `git add --` でステージすること) |
| 2c 座席の中での実行 | 新しい行を足した。近い既存の行(「Whether a codex reviewer can run the gate's scripts」)は gate のスクリプトと leader の fetch だけを扱い、cross-review が呼ぶ相手の CLI と claude の座席を含まないので、新しい行からその行を指した |
| 2 Trigger の注 | 下の「Trigger の掃き出し」。Trigger に当たった 10 行に持ち越しの注、当たらなかったが中身が変わった 3 行に追記 |
| 3 計画 | 印 13 個と Progress の 1 行。digest は `f30958e4f38f` のまま |
| 4 大きさ | 下の「800 行の目安」。雛形の行の数を直した |
| 5 skill を直したあとの検査 | 下の「Verification」 |

## F1〜F6 の照合

| 雛形の動き | 文書 | 結果 |
|------------|------|------|
| 数えるのはこのブランチが base から足した report だけ(F1・F4、`--diff-filter=A`) | 仕様の注、`/org` 終わり方 1 | 直した。`/director` の「cross-review の上限」は complete の回と report の `- Cycle:` の `<n>` で説明し、矛盾しない。README・`config.go`・template の ralph.toml は「complete runs」までの要約で、矛盾しない |
| `k:` はこの org_id の全期間の番号 + 1(F4) | README と `docs/reports/README.md` は名前の形 `c<k>` だけ | ずれなし |
| `base:` の行、`base.err`(F1・F4) | どの文書にも出ない(雛形の中の手順) | ずれなし |
| cap のファイルと、上げたあとの `cap:` の打ち直し(F1) | `/org` 終わり方 1、`/director`、仕様の注 | 合っている |
| `FORK 1:` の答えと `新しい上限 <数>`(F3) | `/director` の「cross-review の上限」の 3 | 合っている(数は `FORK 1:` の行からだけ読む、`<n>` 以下か数がなければ `<n>` + 1) |
| `log:` の行の 200 字、`~` と `.` への置き換え(F1・F4) | `docs/recipes/codex-seat-permissions.md` は `incomplete (<cli> rc=<n>: <a line of its log>)` | 切り詰めは書かないが、矛盾しない |
| HEAD・REPORT・XREVIEW_BASE・REVIEWER_* の確かめ、base の `+` と `@`(F5・F6) | どの文書にも出ない | ずれなし |
| implementer の driver を `ralph org status --json` から読む(F6) | 仕様の注、`/org` 終わり方 1 | 仕様は直し、`/org` は足した。README・`scripts/xreview-helpers.sh`・`paneenv.go`・recipe は「leader の TASK の driver」と書き、矛盾しない。計画の Assumptions の 3 つ目は古いまま(下の「判断を残したもの」) |
| 同じ `<k>` の送り直しは `-r3` まで(F6) | どの文書にも出ない | ずれなし |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md`(278 行の pane の `RALPH_PRIMARY_CLI` と 284 行の cross-review の段落) | 合っている | reviewer は TASK の driver で選ぶ、`xreview-<org_id>-c<k>`、complete の回だけを数える、`XREVIEW: cap_reached`、`Cross-review:` の 3 つの値 |
| `AGENTS.md` | 変更なし | 地図は機能の org の手順を並べない |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` | 直した | 上の表 |
| `templates/base/ralph.toml` | `[org.review]` は合っている。`[reviewer]` は直した | `[reviewer]` のコメントは相手のモデルを呼ぶ所を 3 つだけ挙げていた |
| `internal/config/config.go` | `OrgReviewConfig` は合っている。`ReviewerConfig` と `reviewerWordPattern` のコメントは古い | 2 つのコメントは `[reviewer]` を使う所を「/plan's plan advisory, /cross-review, and the plan advisory of a feature org's leader」と「the skills and a feature org's leader」とだけ書く。reviewer 座席も値をコマンドに写す。Go のファイルなので直していない(下の「判断を残したもの」) |
| `.claude/rules/ralph/model-routing.md` と写し | 直した | 上の表 |
| `/director` skill(4 つ) | 合っている | 件の処理の上限の件と `CROSS_REVIEW:`、停滞の扱い、「cross-review の上限」の 1〜4(report は機能の worktree で読む、`FORK 1:` の形、c は人が決める) |
| `/org` skill(4 つ) | 直した | 上の表。前提の pane の値、停滞の段落、終わり方 6、2 つの要約は合っていた |
| `docs/recipes/codex-seat-permissions.md` と写し | 合っている | network の注の `skipped`・`incomplete` の書き方は reviewer の手順 2 と 4 と同じ。probe の TASK(`TASK_ID: t-1`)は `gate-` でも `xreview-` でもないので、レビューの TASK として読む説明のままでよい |
| `docs/recipes/isolated-herdr-server.md` と写し、`scripts/xreview-helpers.sh` と写し、`docs/reports/README.md` と写し | 合っている | S4・S5 と F2 で直っていた。`git grep` で「seat's cross-review may pick」「treats the seat's driver as claude」は report のほかに残っていない |
| `docs/quality/`、`.claude/rules/ralph/agent-messaging.md`、`docs/architecture/repo-map.md` | 変更なし | quality の 2 つは標準フローの `/cross-review` だけを書く。agent-messaging は TASK の本文の行の名前を並べない |

## tech-debt

### 新しい行

| 行 | 中身 |
|----|------|
| 「Nine test gaps of org-cross-review」 | test report の run 1 の Test gaps 1〜5 と run 2 の 1〜4。コードは関数とテストの名前で指し、`file:line` は書いていない。Trigger は項目ごと |
| 「The cross-review of a feature org has not run inside a seat」 | 2 回の live probe はどちらも座席の外の shell で打った。codex の座席の network、claude の座席の permission mode と入れ子の codex、pane の環境、leader と director の往復が未確認。近い行(gate のスクリプト)を Trigger から指した |

### 既存の行(Trigger の掃き出し)

開いた行の Trigger の欄を、この branch が変えたファイル・節・関数の名前で拾い、Debt の欄にファイルの一覧を持つ行(「any of the five」など)も拾った(`sweep2.py` と `sweep3.py` は scratch)。

| 行(先頭) | 当たったか | 書いたこと |
|------------|-----------|-----------|
| `pick_reviewer` の三つ目の腕 | 当たらない(新しい呼び手は driver を claude と codex に絞る) | Impact に 4 つ目の呼び手を足し、「the three callers」を「the first three callers」に。Trigger の括弧に reviewer 座席を足した |
| codex の reviewer の残りのリスク | 当たらない(標準フローの codex の呼び出しは変えていない) | 注: live probe の P1 で `--disable plugins` だけでは apps が残るとわかり、leader の計画助言と reviewer 座席は `--disable apps` も付ける。(a) に使う形はこれを含む。標準フローの 2 つは付けていない |
| FR-7 の品質ゲートはプロンプト頼み | 当たった(機能の org の品質の流れに cross-review を足した) | 持ち越しの注 |
| org-feature-worktree の小さい所見 | 当たった((g) の「leader.md の機能の org の節の編集」) | 持ち越しの注。(g) は f1b0352e で直っていて、cross-review の report は節の頭の「9 の reviewer のレポート」に入る |
| org-gate の cycle 3・4 の小さい所見 | 当たった((o)・(p)) | 持ち越しの注(依頼の 2b を含む)。(p) は、この branch が同じ段落に 41 桁と 54 桁の短い行を足したことも書いた |
| org-director-attach のテストの穴 3 つ | 当たった(`/director` の受信箱の処理) | 持ち越しの注。新しいテストも、写しが読めないと skip する (1) と同じ形で読む |
| Codex の reviewer に gate の間 send が届かない | 当たらない(実機の run がない) | 追記: cross-review の相手の呼び出し(最長 20 分)の間も同じ |
| `verbs.go` ほか 5 つが 800 行を超える | 当たった(leader の雛形のテスト) | 持ち越しの注と行数(`prompts_leader_plan_test.go` 1278 → 1302、`spawn.go` 2700 → 2706) |
| recipe が `docs/evidence/` を指す | 当たった(2 つの recipe の編集) | 注: 新しい引用は ralph repository の記録として書いてある。`codex-seat-permissions.md` は 3 ファイル、5 つの recipe で 12 |
| RC1(座席の agmsg) | 当たった(`reviewer.md` の編集) | 持ち越しの注 |
| RC4(座席の規則) | 当たった(leader.md の機能の org の手順と `/org` skill) | 持ち越しの注 |
| 手順 14 のあとの implementer の spawn し直し | 当たった(手順 14 の編集) | 持ち越しの注 |
| script-local-config の 5 ファイルの大きさ | 当たった(`config.go` 1180 → 1190、コメントだけ) | 持ち越しの注 |
| 雛形の大きさ(この branch の行) | 数の直し | `leader.md` 1,298、`reviewer.md` 542、テスト 1,449 と 1,268(`433df4ae`)。前の数(`dbfc3ef2`)も残した |

外した候補: 177 行(`/org` の command table。この branch は表を変えていない)、183 行(補償の文。触れていない)、192・196・241〜249 行(director の段階の Trigger。director の起こし方と merge は変えていない)、226 行(実機の run が Trigger。新しい行から指した)、316 行(Trigger は `director_start.go` と `director_wait.go`。`herdr_test.go` の 8 行は Debt の欄の名前だけ)、352 行(`/org` の受信箱の期限)、361 行(spawn の失敗の道筋。`spawn.go` の変更は `TabCreate` の前のコメント)、364 行(`gate.go`・`gate_run.go`)、376・377 行(`/director` の send の確かめと wait の終了コード。「答える順」は変えていない)、378 行(`reject`)、408 行(手順 9 の sync-docs のコミットの引き方。変えたのは段落の頭だけ)、236 行の (q)(`TestRenderRolePrompt_Leader_FeatureOrgGate` は変えていない)。

## 800 行の目安

| ファイル | base → HEAD(`433df4ae`) | 台帳 |
|---------|-------------|------|
| `internal/org/prompts/leader.md` | 975 → 1298 | この branch の行。数を直した |
| `internal/org/prompts/reviewer.md` | 224 → 542 | 同上 |
| `internal/org/prompts_xreview_leader_test.go` | 新規 1449 | 同上 |
| `internal/org/prompts_xreview_reviewer_test.go` | 新規 1268 | 同上 |
| `internal/org/skill_xreview_test.go` | 新規 485 | 800 未満 |
| `internal/org/prompts_leader_plan_test.go` | 1278 → 1302 | `verbs.go` ほかの行。注を足した |
| `internal/org/spawn.go` | 2700 → 2706 | 同上(コメント) |
| `internal/config/config.go` | 1180 → 1190 | script-local-config の行。注を足した(コメント) |
| `internal/org/driver/herdr_test.go` | 1456 → 1464 | 行は足さない。前から超えていて、変えたのは警告の文を固定する行だけ |
| `/org` skill(4 つ) | 2040 → 2087(この pass のあとは 2095) | Markdown で、800 行の目安はコードのファイルのもの |
| `/director` skill(4 つ) | 1231 → 1296 | 同上 |

## 判断を残したもの

- `internal/config/config.go` の `ReviewerConfig` と `reviewerWordPattern` の doc コメントは、`[reviewer]` の値を使う所に reviewer 座席の cross-review を挙げていない。Go のファイルの変更はパイプラインの回し直しになるので直さず、lead に回す。直すなら、`templates/base/ralph.toml` のコメントと同じ一覧にする
- 計画の `## Assumptions` の 3 つ目(「implementer の driver は leader が手順 5 で spawn に付けたものを使う」)は F6 で古くなった。依頼で計画は印と Progress の 1 行のほかを変えないので、Progress の行に古いことを書いた
- org-gate の行の (p)(skill の段落の折り返し)は払わなかった。この branch が同じ段落に短い行を足したが、4 つの写しの折り返し直しは cross-review の計画の Scope の外。注にそのことを書いた
- leader の driver のコマンドは jq を使う(jq がなければ BLOCKED)。`/org` skill の前提は jq を挙げていない。leader の計画助言の codex の道筋も前から jq を使い、挙げていないので、この branch の新しいずれではないと判断し、直していない

## PR の Known gaps に回すもの

1. 座席の中(sandbox、network、permission mode、pane の環境)での cross-review は確かめていない(新しい tech-debt の行、AC9 の known gap)
2. leader と director が実機で上限の QUESTION を往復することは確かめていない(同じ行)
3. テストの穴 9 つ(新しい tech-debt の行)。とくに、残った変異 R1c〜R1e(雛形とテストの定数を一緒に変える場合)と、本物の `ralph org status --json` での driver のコマンドの確かめ

## Verification

| コマンド | 結果 |
|---------|------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-01-org-cross-review.md` | `f30958e4f38f`(編集の前後とも) |
| `./scripts/sync-skills.sh` と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` | rc 0。`/org` skill の 4 つは `cmp` で同じ |
| `./scripts/check-skill-sync.sh` | 14 本が lock-step |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-template-purity.sh` | PASS |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|Director'` | ok(58.1 秒) |
| `go test -count=1 ./internal/config/ -run 'Template\|Default'` | ok(template の ralph.toml のコメントを変えたため) |
| `/org` skill と model-routing.md を読むほかのテスト(名前に `Skill` も `Director` も入らないもの) | `go test -count=1 ./internal/org/ -run 'RenderRolePrompt_Implementer\|RenderRolePrompt_Leader_Resends\|RenderRolePrompt_Reviewer_ReviewTask\|RenderRolePrompt_Reviewer_GateMeans\|ReportNames\|SendDefaults\|ReadRepoFile\|CodeBlockAfterLine'` は ok(トップレベル 16 本が PASS、FAIL 0)。`tests/test-agent-models.sh` 47/47、`tests/test-self-review-scope.sh` 206/206、`tests/test-slice-verification.sh` 162/162 |
| `git diff --check` | 空 |
| tech-debt の確認(`tdcheck.py`、`git show HEAD:` と比べた) | 変わったのは 31・154・157・197・236・259・335・362・368・369・373・380・411・416 行と新しい 417・418 行だけ。どれも 5 欄、欄ごとの backtick と `~~` は偶数、新しい `file:line` の参照は 0、`docs/plans/` の参照はすべて存在する、U+FFFD は 0。Edit の `old_string` は、事前にファイル全体で 1 回ずつ、狙いの行にだけ現れることを数えた |
| `git diff --numstat`(commit の前) | skill 18/10 が 4 つ、model-routing 10/4 と 9/4、plan 14/13、仕様 1/1、tech-debt 16/14、template の ralph.toml 7/6、insight event 1/0 |

`run-verify.sh` と `run-test.sh` は打っていない(Go のファイルと雛形は変えていない)。

## Insight event

`./scripts/insights-append.sh --slug org-cross-review --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` は rc 0。`cycle` は 1 で、既存の `docs/insights/events/2026-11-01-org-cross-review.jsonl`(UTC の日付)に 1 行が足され、新しいファイルはできなかった(`git status --short` で `M`)。self_review(2026-10-31 のファイル)、verify、test の行もそろっている。
