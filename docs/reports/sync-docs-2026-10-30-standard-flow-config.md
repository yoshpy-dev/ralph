# sync-docs report: standard-flow-config

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-standard-flow-config.md`(issue #276 の 3 本目 B の前半 B1。承認の digest `d8e03ef62de9` は run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。plan の編集は `## Progress checklist` の 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json`)。base は `origin/main` の `0351800d`、branch は `feat/standard-flow-config`。run 開始時の HEAD は `d041e080`。文書の変更を入れた commit は `2db81d65`(この report の commit は別)
- 先行 report: `docs/reports/self-review-2026-10-30-standard-flow-config.md`(Merge: yes。L1・L2・L4・L5 は S7 で直し、L3 は直さないと決めた)、`docs/reports/verify-2026-10-30-standard-flow-config.md`(pass。LOW の V1 は S8 で、V2 はこの run で直した。INFO の I1〜I5)、`docs/reports/test-2026-10-30-standard-flow-config.md`(pass。Test gaps 5 つ)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、`2db81d65` の時点): 60 ファイル、+3837/-787。walkthrough を書くかどうかは `/pr` が決める
- 制約: Go の変更はコメント 3 か所、`--config` の usage の 1 文、その pin の 1 項目だけ。打ったのは `go build ./...`、`go vet`、`gofmt -l`、`go test`(config 全体、cli の `-run 'Help|Config|OrgStart|Director'`、org の `-run Skill`、scaffold)、`check-sync.sh`、`check-skill-sync.sh`、`git diff --check`、`plan-visual.sh digest`、tech-debt の欄数・backtick・plan 参照の確認。`run-verify.sh` と `run-test.sh` は打っていない(`secret-scan-branch.sh --strict` は push の前に打つ)

## Summary

文書の本体(`model-routing.md`、`post-implementation-pipeline.md`、`subagent-policy.md`、definition-of-done、codex-setup、template の `ralph.toml`、4 つの skill、leader の雛形、help)は S3〜S9 が済ませていた。4 つの変数の名前が出るのは、「もう読まない」の説明、doctor の表とそのテスト、過去の出来事を書いた tech-debt の 3 行だけで、`${RALPH_…}` の読み出しは 4 つの変数のどれにも残っていない。残っていたずれを、この run で次のように直した。

- README: `ralph config get` の行(V1 と同じずれ)と、`ralph.local.toml` の節に段落を 1 つ。
- `/org` skill(4 つの写し): `start --plan` が `[reviewer]` を読む元を、承認方式・上限と同じ `ralph.toml` に(2 か所)。
- `ralph org --config` の usage に `start --plan` の `[org.approval].mode` と `[reviewer]`(V2)。pin を 1 項目。
- `PipelineConfig` を指していたコメント 3 か所。
- tech-debt: 161 行の (b) に 1 文、Trigger に当たった 7 行に持ち越しの注、154 行の古い言い回し 1 か所、新しい 3 行。
- plan: Progress に sync-docs の行を 1 つ。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` 161 行(`insights-append.sh` の 2 つの穴) | 項目 1。Debt の (b) の末尾に「`/cross-review` step 9 は `cap` を書くが、`cycle` は増やさない。(b) は前と同じく開いたまま」の 1 文。Related に plan(`docs/plans/active/` の形)。行番号は HEAD で読み直したが動いていなかった |
| 同 154 行(codex の reviewer の残る危険) | Trigger の「`$BASE` と `${RALPH_…}` の既定値を除いて」を「`$BASE`、`$model`、`$effort`」に直した(`${RALPH_…}` の既定値は S3 で消えた)。持ち越しの注: `/cross-review` 手順 4 と `/plan` 11.c の codex の呼び出し、`tests/test-codex-exec-invocation.sh` の書き換えに当たったが、(a)〜(e) は Scope の外で、書き換えは (e) を閉じない(`$`・backtick・`#` を弾かない) |
| 同 148 行(config と state dir の決め方が別) | 持ち越しの注。`resolveOrgConfig` が新しい `resolveConfig` を呼ぶ形になり、`withMainWorktreeOrgLimits` の doc も変えたので Trigger に当たった。開いた部分(config の探し方)には触れていない。足したのは、`start --plan` が読むものを主 worktree の 1 つのファイルにそろえる `resolveOrgStartReviewer` |
| 同 197 行(org-feature-worktree の小さい所見) | 持ち越しの注。`leader.md` の機能ごとの org の節(計画助言)を直したので (g) と閉じた (f)・(j) の Trigger に当たったが、払うものはなかった。開いた項目の列には、閉じ済みの (h) を入れなかった |
| 同 362 行(800 行を超えるファイル) | 持ち越しの注。`prompts_leader_plan_test.go` は 1179 から 1278 行(+99)、`prompts_test.go` は `featureLeaderTask` の呼び出し 3 つを直しただけで 2683 行のまま。+300 に届かない。第 1 欄の数は branch より前から古いので、origin/main の数を書いた |
| 同 368 行(recipe が `docs/evidence/` を引く行) | 持ち越しの注。`codex-setup.md` の 2 つの写しを直したが、引用は増やしも言い換えもしていない |
| 同 373 行(RC4) | 持ち越しの注。`leader.md` の計画助言と `/org` skill の 2 か所に当たったが、step 4 と分割計画の例には触れていない |
| 同 400 行(`local_test.go` が 800 行を超えた) | 持ち越しの注。`TestLoad_LocalFileOverridesPipelineCyclesAndReviewer` で 981 から 1011 行 |
| 同 404〜406 行(新しい 3 行) | 項目 7。(1) `defaults_sync_test.go`(671 から 946)と `org_feature_test.go`(647 から 819)が 800 行を超えた。branch が増やした、すでに超えていた 3 ファイルの数も添えた。(2) `ralph org start --plan` が主 worktree の `ralph.toml` を 2 回読む(self-review L3)。(3) test gaps 3 つ(`mktemp -d` の失敗の分岐、`resolveOrgStartReviewer` のエラーの return、`zsh -f` で動かない runner) |
| `README.md`(`ralph config get` の行) | 項目 2。「止まって報告する」のは、review の switch のように skill や seat が `ralph config get` を自分で打つ場合だと範囲を絞り、`standard.max_pipeline_cycles` と `[reviewer]` の 4 つのキーは `scripts/ralph-config.sh` の `ralph_setting` が読むので止まらず、2 つの失敗(older ralph、PATH にない)は黙って、ほかは stderr に 1 行で既定値に進む、と足した。`ralph config get --help`(S8)と同じ内容 |
| `README.md`(`ralph.local.toml` の節の末尾) | 段落を 1 つ足した(依頼の範囲を少し超える。下の「判断を残したもの」)。`[standard].max_pipeline_cycles` と `[reviewer]` の 4 つのキーを `ralph.toml` に書き、1 台だけなら `ralph.local.toml` に書くこと、skill は `ralph_setting` で読み、`start --plan` は `[reviewer]` を leader の TASK に写すこと、4 つの変数はもう読まず `ralph upgrade` が入れる skill と helper も読まないこと、doctor の warn、1 つの作業の上限は `/cross-review` が `cycle-count.json` に `cap` を書く、の順 |
| `.claude/skills/org/SKILL.md` と `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 項目 3。(a) タスクの reviewer の 2 行の段落(711 行あたり)の「`--config`、省略時は cwd の `./ralph.toml`」を、承認方式・上限と同じ規則(`--config` がなく台帳が main worktree のものなら main worktree のルートの `ralph.toml`、それ以外は `--config` のファイル、なければ cwd の `./ralph.toml`。主 worktree のルートの `ralph.local.toml` が上書きする)に。(b) 依頼の外だが同じずれ: 上限を主 worktree から読む規則の段落(311 行あたり)が「`start --plan` は承認方式も同じ規則で読む」だけだったので、`[reviewer]` の 4 つのキーも足した。`.agents/skills/org` は `sync-skills.sh` で作り直し、`templates/base/.agents/skills/org` は手で `cp` した(`sync-skills.sh` はルートの `.agents/skills` しか書かない) |
| `internal/cli/org.go`(`ralph org` の `--config` の usage) | 項目 4(V2)。「spawn and start read the org-wide limits (see ralph org spawn --help)」のあとに「start --plan also [org.approval].mode and the four keys of [reviewer] (see ralph org start --help)」。usage の後半(`escalate and director attach read …` から)を固定している既存の pin は、この挿入の後ろにあるので、そのまま通る |
| `internal/cli/org_director_test.go` | V2 の pin。`TestOrgDirectorHelp_DescribesTheRegistration` の `--help` の項目に、新しい断片を 1 つ足した。既存の項目は弱めていない |
| `internal/config/config.go` 2 か所、`internal/config/config_test.go` 1 か所 | 項目 5。「PipelineConfig の int は 0 のとき黙って埋める」「PipelineConfig と同じ zero-value の埋め方」と書いていた。`PipelineConfig` は `c2e0dcbd`(ループの撤去)で型ごとなくなっている。今あるのは文字列の既定を `""` のとき埋める書き方(`agmsg_home`、`[org.permissions].default`)なので、それを名指す文にした。意味は変えていない |
| `docs/plans/active/2026-10-30-standard-flow-config.md` | Progress に sync-docs の行。本文の節は触れていない。digest は前後で `d8e03ef62de9` |
| `docs/reports/sync-docs-2026-10-30-standard-flow-config.md` | この report |
| `docs/insights/events/2026-10-30-standard-flow-config.jsonl` | `sync_docs` の行(cycle 1、pass)を足した |

## 依頼の項目ごとの結果

| 項目 | 結果 |
|------|------|
| 1 tech-debt 161 行の (b) | 直した。Debt に 1 文、Related に plan。誤読を防ぐ向きに、「`cap` を書くようになったが、`cycle` は増えないので (b) は前と同じ」と書いた |
| 2 README の `config get` の行 | 直した。README に `ralph.toml` のキーの表はない。`[standard].max_pipeline_cycles` と `[reviewer]` は、`ralph.local.toml` の節に段落を足して書いた |
| 3 `/org` skill の reviewer の 2 行 | 4 つの写しを直した。`check-sync.sh` は DRIFTED 0、`check-skill-sync.sh` は 14 本が lock-step。依頼になかった上限の規則の段落(311 行あたり)も、同じずれなので直した |
| 4 V2(`--config` の usage) | 直した。help を固定しているテストは、usage の後半だけで、挿入の後ろにあり、弱めずに通る。pin を 1 項目足した。`go test ./internal/cli/ -run 'Help\|Config\|OrgStart\|Director'` は ok(217.5s) |
| 5 `PipelineConfig` | 直した。依頼の `:883`・`:935` と `config_test.go:419` はどれも HEAD で合っていた。`go build ./...` と `go vet ./internal/config/ ./internal/cli/` は rc 0、`gofmt -l` は空、`go test ./internal/config/` は ok |
| 6 ほかのずれ | 下の「Surfaces checked」。Trigger に当たった行は 7 行で、すべて持ち越しの注を付けた |
| 7 tech-debt に足すか | L3 は足した(404〜406 行の 2 つ目)。test の Test gaps は、1・2・3 を 1 行にまとめて足した。4・5 は足さない(下の「直さなかったもの」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| 4 つの変数(`RALPH_STANDARD_MAX_PIPELINE_CYCLES`、`RALPH_CLAUDE_REVIEWER_MODEL`、`RALPH_CODEX_REVIEWER_MODEL`、`RALPH_CODEX_REASONING_EFFORT`)の出現 | 「もう読まない」の説明、doctor の表、その検査、過去の記録だけ | `git grep`(`docs/reports`、`docs/plans`、`docs/insights`、`docs/specs`、`docs/evidence` を除く)。`model-routing.md`(root と template)、`post-implementation-pipeline.md`(root と template)、`scripts/ralph-config.sh` の冒頭(root と template)、README の新しい段落、`internal/cli/doctor.go`、テスト(env が効かないことの確認)。tech-debt は 134・162・215 行の 3 行で、どれも「パイプラインが上限 `RALPH_STANDARD_MAX_PIPELINE_CYCLES=n` に当たった」という過去の出来事なので残した(現在の挙動の主張ではない) |
| `${RALPH_…}` の読み出し(AC2 の grep) | 4 つの変数には 0 件 | `.claude/skills`・`.agents/skills` と template の写し、`scripts/`、`internal/org/prompts/`、docs。残る当たりは `RALPH_PRIMARY_CLI`(残す)、B2 の 5 つの変数、内部の受け渡しの変数 |
| `AGENTS.md` | 変更なし | `scripts/` の行は `ralph-config.sh` を名前で挙げるだけで、説明がない。短く保つ |
| `docs/architecture/repo-map.md` | 変更なし | S6 が `ralph-config.sh` の説明を `ralph_setting <key>` に直している。`cycle-count.json` の行は cap の状態を指すだけ |
| `docs/quality/definition-of-done.md`、`quality-gates.md` | 変更なし | DoD は S6 が直した(上限は `[standard].max_pipeline_cycles`)。quality-gates に新しい verifier の列挙は要らない(`tests/test-ralph-config.sh` は前からある) |
| `.claude/rules/ralph/` | 変更なし | `model-routing.md`・`post-implementation-pipeline.md`・`subagent-policy.md` は S6 が直済み。`model-routing.md` の「Where the values live」の `defaults_sync_test.go` の説明は今の照合と合っている |
| `cycle-count.json` の言及(cross-review、implement、pr、docs) | 合っている | `cap` を書く(cross-review 手順 9)、保つ(implement 3.c)、`/pr` が消す、の 3 つが一貫している |
| `templates/base/ralph.toml` の注釈 | 変更なし | 読めないときの扱い、許す文字、leader は start のときの値を使うこと、`cap` の 1 文が、実装(S2・S3・S9)と合っている(consult の注意 1) |
| `ralph config get --help`、`ralph org start --help` | 変更なし | S8・S9 が直済み。README の `config get` の行をこの help に合わせた |
| `ralph doctor` の README の行 | 変更なし | 撤去した変数の warn を列挙していない(A1・A2 のときの README も、`ralph.local.toml` の節に書いていた) |
| `docs/tech-debt/README.md` の欄数・backtick・plan 参照 | 通った | 5 欄の行が 304 から 307 に(新しい 3 行)。変えた 8 行と新しい 3 行は、すべて 5 欄で各欄の backtick と `~~` が偶数。`file:line` の参照は変えた行に 0 件。`docs/plans/(active\|archive)/` の参照に、存在しないパスは 1 件もない(新しい参照は `docs/plans/active/2026-10-30-standard-flow-config.md` で、`/pr` の `archive-plan.sh` が書き換える。`docs/plans/active/...` の 1 件は前からある省略表記) |
| `U+FFFD`、`git diff --check` | なし | 変えたファイルを走査 |

## 判断を残したもの

- README の `ralph.local.toml` の節に段落を足した。依頼は「README に `ralph.toml` の表やキーの説明があれば合わせる」で、表はなかった。ただ A1・A2 は、撤去した変数ごとにこの節へ段落を足している(`RALPH_ORG_AGMSG_HOME`、`RALPH_ORG_STATE_DIR`)ので、その形に合わせた。B2 の plan が、この節に個人用の値を足す。要らなければ、この段落だけ消せる(他の文書から参照していない)。
- `/org` skill の上限の規則の段落(311 行あたり)を直した。依頼は 711 行あたりだけだったが、S9 でその段落も事実と食い違った。
- 持ち越しの注で、197 行の開いた項目の列から (h) を外した。その行の Trigger が (h) を「done in feat/doctor-either-agent-cli」と書いているので、前の注の列は古い。前の注は書き換えていない。

## 直さなかったもの

| 項目 | 理由 |
|------|------|
| test の Test gaps 4・5(helper が読んだ値で実物の claude・codex を起こす通し、実物の v6.0.0 に skill を回させる確かめ) | コードの穴ではなく、実機の確かめ。4 は `docs/evidence/claude-effort-probe-2026-10-30.md` と、leader の雛形のコマンドを偽の claude・codex で動かすテストが見ている。5 は偽の ralph の exit 1 と `unknown command "config"` の行が見ている。PR の Known gaps に回す |
| leader が PATH の `ralph` を打つこと(`ralph config get org.review.plan_advisory` と `ralph org` の動詞) | plan の Non-goals。PATH に古い ralph があるずれは、192 行(別版の `ralph` で台帳が割れる)と同じ根。新しい行は足さなかった。PR の Known gaps に回す |
| B2 の 5 つの変数(`RALPH_PLAN_VISUAL_BROWSER`、`RALPH_PLAN_VISUAL_OPENER`、`RALPH_XREVIEW_BASE`、`RALPH_VERIFY_BASE`、`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN`)とその文書 | 依頼の「触らないもの」。B2 の plan で扱う |
| `docs/specs/` | 依頼の「触らないもの」(過去の記録) |
| tech-debt の 134・162・215 行の `RALPH_STANDARD_MAX_PIPELINE_CYCLES=n` | 過去の出来事の記録。現在の挙動の主張ではないので、書き換えない |
| `internal/cli/org_director_test.go`(+3 行で 1165 行) | 前から 800 行を超えている。3 行の pin なので行は足さず、report にだけ書く |
| plan の AC の checkbox、plan 本文(Rollout の「revert した ralph は `[reviewer]` を無視する(推測、未確認)」) | 依頼の「触らないもの」。確かめた事実は下の I3 に書いた。lead が PR 本文か Rollout に使う |

## PR の Known gaps に回すもの

1. 実物の claude・codex を、helper が読んだ値で起こす通しの確かめはしていない(test の Test gaps 4)。`--effort` が `-p` で効くことは evidence にある。
2. 実物の v6.0.0(`config get` がない)で skill を回してはいない(test の Test gaps 5)。偽の ralph の exit 1 と `unknown command "config"` の行で、既定値に戻ることは見ている。doctor を 6.0.0 に打った結果は I3。
3. leader は PATH の `ralph` で `ralph config get org.review.plan_advisory` と `ralph org` の動詞を打つ。PATH に古い版があるずれは残る(plan の Non-goals。192 行と同じ根)。この PR が直したのは reviewer の値だけ。
4. B2 の 5 つの変数は env のまま(次の PR)。
5. tech-debt の新しい 3 行: 800 行を超えた 2 ファイル、`start --plan` の 2 回読み、test gaps 3 つ(`mktemp -d` の失敗の分岐、`resolveOrgStartReviewer` のエラーの return、`zsh -f` で動かない runner)。
6. 設定を読めないときの `ralph_setting` は、exit 2・3 でも 1 行知らせて既定値で進む。`ralph config get` の review switch は、前のとおり止まる。2 つの扱いが分かれる(README と `ralph config get --help` に書いた)。

## PR 本文の材料

- verify の I3: release 済みの 6.0.0 は、新しいキーを書いた ralph.toml を warn なしで読む。PATH の ralph 6.0.0 の `doctor` を、`[standard].max_pipeline_cycles` と `[reviewer]` の 4 つのキーを有効な行で書いた ralph.toml に打ったところ、rc 0、`All checks passed.`、ralph.toml の warn も parse error もなかった(`docs/reports/verify-2026-10-30-standard-flow-config.md` の I3、`docs/evidence/verify-2026-10-30-standard-flow-config.md` の 6 節)。plan の Rollout が「推測、未確認」と書いた点は、直前の release では確かめられた。
- consult の注意 2(plan の Progress、S1): `claude_model`・`codex_model`・`codex_effort` に許す文字は `A-Z a-z 0-9 . _ : @ / + -` で、PR 本文に 1 行書く。
- plan の D6: `COVERAGE_THRESHOLD` と `RALPH_TERRAFORM_INIT_VALIDATE` は env のまま残す例外で、理由(CI には ralph がない、terraform のパックの選択)を一覧と PR 本文に書く。`RALPH_PRIMARY_CLI` も「内部の受け渡し」の例外の注つき。
- release のノートに書くこと: plan の Rollout の 6 項目(4 つの変数を読まなくなった、claude の既定が fable と xhigh、上限は `/cross-review` が `cap` を書く、leader の計画助言は start のときの値、doctor の warn、新しい binary が要る)。

## Verification (この pass)

| コマンド | 結果 |
|---------|------|
| `go build ./...`、`go vet ./internal/config/ ./internal/cli/`、`gofmt -l internal/config internal/cli` | rc 0、gofmt は空 |
| `go test ./internal/config/` | ok(25.0s) |
| `go test ./internal/cli/ -run 'Help\|Config\|OrgStart\|Director'` | ok(217.5s) |
| `go test ./internal/org/ -run Skill`(20 本)、`go test ./internal/scaffold/` | ok、FAIL 0 |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | 14 本が lock-step |
| `git diff --check`、`U+FFFD` の走査 | 空 |
| `./scripts/plan-visual.sh digest <plan>` | `d8e03ef62de9`(承認のときと同じ) |
| tech-debt の欄数・backtick・`~~`・`file:line`・plan 参照 | 通った(上の表) |

## Insight event

`./scripts/insights-append.sh --slug standard-flow-config --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` は rc 0。`cycle` は 1。`docs/insights/events/2026-10-30-standard-flow-config.jsonl` に 1 行が足され、ほかのファイルは増えなかった。
