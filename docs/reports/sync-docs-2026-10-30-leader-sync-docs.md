# sync-docs report: leader-sync-docs

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-leader-sync-docs.md`(issue #268。承認の digest `ab78e5f0645b` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の 1 行だけで、本文の節と AC の checkbox は触れていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `origin/main` の `0351800d`、branch は `feat/leader-sync-docs`。run 開始時の HEAD は `a5e28500`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-30-leader-sync-docs.md`(run 3 まで。CRITICAL・HIGH・MEDIUM は 0、Merge: yes)、`docs/reports/verify-2026-10-30-leader-sync-docs.md`(再実行で pass。V1・V2 は d8e73710 で直済み)、`docs/reports/test-2026-10-30-leader-sync-docs.md`(pass。変異 12 個がすべて落ちた)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、この run の前): 19 ファイル、+2855/-66。walkthrough を書くかどうかは `/pr` が決める
- 制約: コードは変えていない。打ったのは `check-template-purity.sh`、`check-skill-sync.sh`、`check-sync.sh`、`git diff --check`、`plan-visual.sh digest`、skill と雛形のテスト(`-run 'Skill|RenderRolePrompt'`)。`run-static-verify.sh` と `run-test.sh` の全体は打っていない(skill の文と写しの変更だけのため)

## Summary

実装が先に直していた文書(雛形、`/org` と `/director` skill とその写し、仕様 FR-4 の注、README の gate の段落、tech-debt の 2 行)は、下の表で見た範囲では手順の中身と合っていた。残っていたずれは、`/org` skill の中で機能の org の締めの順を要約した 2 か所で、どちらも sync-docs の段が抜けていた。

- 「leader の責務」の 7: 「reviewer のレポートをコミットして rebase し、…report を打って、…」の並び
- 「タスク終了時は必ず」: 「reviewer のレポートのコミット → rebase → report → …」の矢印の並び

「org の終わり方」本体と「完了条件」は sync-docs を持っているのに、この 2 つの要約だけが古い順のままで、これを読む leader は sync-docs の段を落とすおそれがある。2 か所とも rebase のあと、report の前に sync-docs を足した。手順の番号は変えていない。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`(「leader の責務」の 7) | 「既定のブランチに rebase し、」のあとに「sync-docs を doc-maintainer に任せてコミットし、」を足した。折り返しは 5 行のまま、幅は 76 以内 |
| `.claude/skills/org/SKILL.md`(「タスク終了時は必ず」) | 矢印の並びに「→ sync-docs のコミット」を rebase と report のあいだに足した。折り返しは 1 行分だけ組み直した |
| `.agents/skills/org/SKILL.md` | 上の写し(`scripts/sync-skills.sh`) |
| `templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 上の写し(`cp`。`sync-skills.sh` は template を作らない) |
| `docs/plans/active/2026-10-30-leader-sync-docs.md` | `## Progress checklist` に「Sync-docs report created」の行を 1 つ足した |
| `docs/reports/sync-docs-2026-10-30-leader-sync-docs.md` | この report |
| `docs/insights/events/2026-10-30-leader-sync-docs.jsonl` | `sync_docs` の行を足した(下の「Insight event」) |

足した 2 か所の文はテストが固定していない(`git grep` で `internal/` と `scripts/` に当たる断片がないことを確かめた)。固定済みの「完了条件」の文(`skill_doc_followup_test.go`)とは別の箇所である。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md` の `internal/org/` の行 | 変更なし | 機構の一覧で、leader の手順は載せていない。地図を短く保つ約束なので、sync-docs は足さない |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`(Quality pipeline gate の行) | 変更なし | 行が書くのは検証の段(impl の確かめ → reviewer → leader の裁定、最後の gate)で、leader の report・archive・fetch と rebase の段も載せていない。gate は rebase 後の head に対して走るので、sync-docs のコミットもその head に入り、「rebase した head で gate が走る」という行の文は今も合う。root と template は KNOWN_DIFF で、片方だけ変える理由もない |
| `docs/quality/definition-of-done.md`(org runtime の節) | 変更なし | 仕様と `agent-messaging.md` を指すだけの節 |
| `docs/recipes/` | 変更なし | 機能の org の終わりまでを順に追う recipe はない。`isolated-herdr-server.md:496` の `ralph org report` は後始末の手順で、leader の段とは別 |
| `docs/architecture/repo-map.md`(47・48 行) | 変更なし | skill の 1 行の紹介で、`/director` が受信箱を扱うと報告するとだけ書く。follow-up を集める節は、その範囲に入る |
| `docs/reports/README.md` と template の写し | 変更なし | `sync-docs-YYYY-MM-DD-short-slug.md` の名前が機能の org の sync-docs の report にもそのまま当たる |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 の注(126 行など) | 変更なし | 「手順 9〜11」を 2026-10-26 の日付つきで挙げる記録で、書き換えない約束。FR-4 の 2026-10-30 の注が今の並びを持つ |
| `.claude/rules/ralph/agent-messaging.md` | 変更なし | RESULT の本文の行(`GATE:`、`HEAD:` など)を一覧していない。`DOC_FOLLOWUP:` は雛形と skill の約束で、ralph の検証が見るのは TYPE・TASK_ID・字数だけ |
| `.claude/rules/ralph/subagent-policy.md`、`model-routing.md` | 変更なし | doc-maintainer を標準フローの pipeline のものとして書くが、ほかでは使わないとは書いていない。leader の子 subagent は `/org` skill の「座席内 fan-out」が持つ。`model-routing.md` の「org runtime model receipts」は座席の model の話で、座席の中の子には及ばない |
| `.claude/rules/ralph/git-commit-strategy.md` | 変更なし | leader の「手順 12 と 14」の `wip:` の確かめを挙げる。番号は変えていない |
| 「leader は subagent を使わない」と読める文 | なし | `git grep` で leader と subagent・fan-out が近い文を探した。見つかったのは `/org` skill の 1580 行(認めている文)と、implementer・reviewer の雛形の「座席内 fan-out」だけ |
| 機能の org の締めの順を要約する文 | 2 か所を直した | 上の Summary。雛形(`leader.md`)には順の要約がなく、README は gate の段落が sync-docs を持つ。`docs/tech-debt/README.md` の step 番号を挙げる行(228・230 行)は番号が変わらないので合う |
| README(268 行の「leader が rebase し、reviewer に gate を打たせ、PR を開く」) | 変更なし | 要約で、org の report・archive・secret scan も載せていない。順の詳細は 280 行の gate の段落が持ち、sync-docs はそこにある |
| `docs/tech-debt/README.md` の新しい 2 行(`/director` の集めの印、前の sync-docs のコミットの引き方) | 変更なし | 計画の Progress と合う。Codex の leader が custom agent を呼べることと、leader が手順を実機で走らせることは、行にせず、仕様 FR-4 の注(FR-4 の印を変えない)と PR の known gap に任せる。計画のリスクの項の決めどおり |

## Commands run

| Command | Result |
|---------|--------|
| `./scripts/check-template-purity.sh` | PASS(meta-repo 固有の参照なし) |
| `./scripts/check-skill-sync.sh` | rc 0、14 skill が一致 |
| `./scripts/check-sync.sh` | PASS。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `git diff --check` | rc 0、出力なし |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|RenderRolePrompt'` | ok、12.7 秒 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-leader-sync-docs.md` | `ab78e5f0645b`(承認の行と同じ) |

## Not done

- AC1〜AC8 の checkbox は付けていない。`/pr` の前の段が付ける
- 雛形の手順を leader が実機の org で走らせること、Codex の leader が pane の中で custom agent を呼べることは確かめていない(仕様 FR-4 の注と、計画のリスクの項のとおり。tech-debt の行にはしていない)
- 足した 2 か所の文は、変更前の skill で落ちるテストにしていない

## Insight event

`./scripts/insights-append.sh --slug leader-sync-docs --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-30-leader-sync-docs.jsonl` に 1 行足した。

## cycle 2

- Date: 2026-10-30
- Pipeline cycle: 2(`cycle-count.json` の値)。run 開始時の HEAD は `147e84bc`、前の sync-docs の commit は `e5f90219`。計画の digest は `220fa1f3994f`(cross-review cycle 1 の直しのあと `ef4a15a7` で付け直したもの)で、この run の前後で同じ
- 先行 report: `docs/reports/cross-review-triage-leader-sync-docs.md`(cycle 1 の ACTION_REQUIRED 1 件)、self-review cycle 2(`8bd985fb`、Merge: yes)、verify cycle 2(`b0845cd3`、pass、LOW の V3)、test cycle 2(`147e84bc`、pass。変異 21 個がすべて落ちた)
- 前回から変わったもの: F5 `4d0f70e4`(`git status --porcelain --untracked-files=all` にそろえ、未追跡のファイルは一覧に出たものを 1 つずつ消す)、F5b `aec1381f`(doc-maintainer を呼ぶ前に何も出ないことを確かめる)、F6 `41468881`(その振り分けに per-plan の `PLAN:` のヘッダ)。雛形、`/org` skill とその写し 3 つ、テスト、台帳の行(`94c10923`)に入っている

### Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-4 の 2026-10-30 の注、93 行) | verify の V3。「leader は `git status --porcelain` で変わったパスを読み」を、呼ぶ前に `git status --porcelain --untracked-files=all` が何も出さないことを確かめ(出れば呼ばず、fetch の前に変更が残っていたときと同じく扱う)、終わったら同じコマンドで変わったパスを 1 ファイルずつ読む、に直した。日付つきの決定の記録なので、ほかの文は変えていない |
| `docs/plans/active/2026-10-30-leader-sync-docs.md` | `## Progress checklist` に cycle 2 の行を 1 つ足した。本文と AC の checkbox は触っていない |
| `docs/reports/sync-docs-2026-10-30-leader-sync-docs.md` | この節 |
| `docs/insights/events/2026-10-30-leader-sync-docs.jsonl` | `sync_docs` の cycle 2 の行 |

### Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| README(268 行、280 行の gate の段落) | 変更なし | `git status` を名指さない。280 行の sync-docs の説明は要約で、呼ぶ前の確かめや `PLAN:` のヘッダの細部までは書かない |
| `AGENTS.md` | 変更なし | 地図で、leader の手順を載せていない |
| `/director` skill(「予約外の文書の follow-up」) | 変更なし | `git status` を名指さない。leader の sync-docs の説明は要約で、今の動きと矛盾しない |
| `/org` skill の要約 2 か所(前回足した「leader の責務」の 7 と「タスク終了時は必ず」) | 変更なし | 締めの順だけを書く。F5・F5b・F6 は sync-docs の段落の中の確かめ方の変更で、順は変わらない |
| `/org` skill の完了条件(「worktree の `git status --porcelain` が空」) | 変更なし | 終わりの状態の条件。長い形(`--untracked-files=all`)で何も出なければ短い形でも何も出ないので、矛盾しない。この文は `skill_doc_followup_test.go` が固定している |
| 雛形の手順 9 の最初の段落と `/org` skill の同じ段落(fetch の前の確かめ、`git status --porcelain` のまま) | 変更なし | `prompts_test.go` が固定していて変えない。短い形が見落とす未追跡のファイルは、F5b の呼ぶ前の確かめが埋める(計画の F5b の記録、verify cycle 2) |
| `docs/quality/quality-gates.md`、`definition-of-done.md`、template の写し | 変更なし | cycle 1 と同じ。F5・F5b・F6 は検証の段の書き方に触れない |
| `.claude/rules/ralph/`(`model-routing.md` の `git status --porcelain` など) | 変更なし | `model-routing.md:43` は implementer の報告の証拠の書き方で、leader の sync-docs とは別の話。ほかの rules は sync-docs の確かめ方を書かない |
| `docs/recipes/`(`codex-setup.md:183`、`worktrees.md:62`) | 変更なし | どちらも leader の sync-docs とは関係のない `git status --porcelain` |
| `docs/tech-debt/README.md` の 405 行 | 変更なし | `94c10923` が照らし合わせる側を `--untracked-files=all` に、払い方を `-z` つきの同じコマンドに直済み。雛形と合う |
| 仕様 FR-4 の注の、`git status` 以外の文 | 変更なし | director の集め方、凍らせたファイル、飛ばすときの条件は雛形と skill の今の文と合う |

### Commands run

| Command | Result |
|---------|--------|
| `./scripts/check-template-purity.sh` | PASS |
| `./scripts/check-skill-sync.sh` | rc 0、14 skill が一致 |
| `./scripts/check-sync.sh` | PASS。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `git diff --check` | rc 0、出力なし |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|RenderRolePrompt'` | ok、15.3 秒(この run は skill も雛形も直していないが、依頼のとおり打った) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-leader-sync-docs.md` | `220fa1f3994f`(承認の行と同じ) |

### Not done

- AC の checkbox は付けていない。雛形の手順を leader が実機の org で走らせること、Codex の leader が custom agent を呼べることは、cycle 1 のとおり確かめていない
- 仕様 FR-4 の注に、`--untracked-files=all` を付ける理由(新しいディレクトリが 1 行にまとまること、`status.showUntrackedFiles=no`)は書いていない。理由は雛形と `/org` skill の段落が持ち、注は要約のため

### Insight event

`./scripts/insights-append.sh --slug leader-sync-docs --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-30-leader-sync-docs.jsonl` に cycle 2 の行を足した。

## cycle 3

- Date: 2026-10-30
- Pipeline cycle: `cycle-count.json` は 2 のまま(cross-review cycle 2 の ACTION_REQUIRED を受けて、上限を `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げて回し直した run。計画の Progress の記録)。run 開始時の HEAD は `bc00d1d1`、前の sync-docs の commit は `822acf62`。計画の digest は `94dc29701d29`(cross-review cycle 2 の直しのあと `ebc2b64a` で付け直したもの)で、この run の前後で同じ
- 先行 report: `docs/reports/cross-review-triage-leader-sync-docs.md`(cycle 2 の ACTION_REQUIRED 1 件)、self-review cycle 3(`eea3eb84`、Merge: yes)、verify cycle 3(`d751a6e3`、pass、LOW の V4)、test cycle 3(`bc00d1d1`)
- 前回から変わったもの: F7 `44694f5b`(`/director` skill の手順 6 に、案を見せる前に follow-up の文書を読むテストと文を固定するテストと fixture を探して `- Reserve:` に入れ、人に言う)、F8 `b6a8060a`(その文を「一覧のファイルを読むテストと、その文を固定するテストと fixture を探し」に言い直した)。`/director` skill の写し 3 つ、テスト、計画の D4 に入っている

### Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-4 の 2026-10-30 の注、93 行) | verify の V4。「写しの組の follow-up は組のすべてを予約に入れ」のあとに、案を見せる前に follow-up の文書を読むテストと、その文を固定するテストと fixture も探して予約に入れ(予約の外のテストを implementer が直せず、文書の org が gate を通れないため)、足したものとその理由を人に言う、を足した。日付つきの決定の記録なので、ほかの文は変えていない |
| `docs/plans/active/2026-10-30-leader-sync-docs.md` | `## Progress checklist` に cycle 3 の行を 1 つ足した。本文と AC の checkbox は触っていない |
| `docs/reports/sync-docs-2026-10-30-leader-sync-docs.md` | この節 |
| `docs/insights/events/2026-10-30-leader-sync-docs.jsonl` | `sync_docs` の行(`--cycle auto` は `cycle-count.json` を読むので、cycle 2 と出る) |

### Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| README(280 行の gate の段落) | 変更なし | director が文書だけの機能を人と書くことまでを要約する。予約の中身は書かない |
| `/org` skill(「org の終わり方」の 1、doc-maintainer への指示の「組のすべてを予約できるように」) | 変更なし | doc-maintainer が follow-up に書くのは文書のずれと写しの組で、テストは探さない。テストを探すのは director で、`/director` skill の 6 にある |
| `/director` skill の概説(1093 行の「文書だけの機能 1 つの分割計画を人と書く」) | 変更なし | 要約で、手順 6 の予約の細部と矛盾しない |
| `docs/quality/`、`.claude/rules/ralph/`、`AGENTS.md`、template の写し | 変更なし | director の分割計画の予約の書き方を書いていない |
| `docs/tech-debt/README.md`(404 行の director の集めの印の行) | 変更なし | 集めの印と記録の場所の話で、予約にテストを入れることとは別 |
| 仕様 FR-4 の注の、テスト以外の文 | 変更なし | cycle 2 の確かめのとおり、雛形と skill の今の文と合う |

### Commands run

| Command | Result |
|---------|--------|
| `./scripts/check-template-purity.sh` | PASS |
| `./scripts/check-skill-sync.sh` | rc 0、14 skill が一致 |
| `./scripts/check-sync.sh` | PASS。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `git diff --check` | rc 0、出力なし |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill\|RenderRolePrompt'` | ok、23.8 秒(この run は skill も雛形も直していないが、依頼のとおり打った) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-leader-sync-docs.md` | `94dc29701d29`(承認の行と同じ) |

### Not done

- AC の checkbox は付けていない。director が実機で文書の機能の予約にテストを入れること、leader が実機で手順を走らせること、Codex の leader が custom agent を呼べることは、cycle 1 のとおり確かめていない
- 仕様 FR-4 の注には、director がテストを探す方法(repo のテストのファイルを、一覧の各ファイルのパスと follow-up で変わる文で検索する、など)は書いていない。方法は `/director` skill の手順 6 が持つ

### Insight event

`./scripts/insights-append.sh --slug leader-sync-docs --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-30-leader-sync-docs.jsonl` に 1 行足した。
