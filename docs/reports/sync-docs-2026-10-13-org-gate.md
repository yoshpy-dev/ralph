# sync-docs report: org-gate

- Date: 2026-10-13(JST)
- Plan: `docs/plans/active/2026-10-13-org-gate.md`(承認済み。`./scripts/plan-visual.sh digest` は `69f088275e54` で、`- Approved:` の行と同じ。この run は計画を編集していない)
- Pipeline cycle: 1(`--cycle auto`)。差分の base は `1bddc871`、branch は `feat/org-gate`。run 開始時の HEAD は `bb2ed920`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-13-org-gate.md`(追記を含む。Merge 可、CRITICAL・HIGH 0)、`docs/reports/verify-2026-10-13-org-gate.md`(pass。「Documentation drift」の D1〜D4)、`docs/reports/test-2026-10-13-org-gate.md`(pass、失敗 0)
- 項目の番号: D1〜D4 は verify の「Documentation drift」の表の番号
- 内容の commit: この report、insight event、直した文書を含む 1 つの commit(`docs: sync docs for org-gate`)

## Summary

verify の D1〜D3 を文書だけで直した。D4(計画の Progress checklist と Notes)は、依頼どおり計画を編集していない。Go のコード、テスト、`--help` の文字列、leader と reviewer の雛形(`internal/org/prompts/`、go:embed)、skill、`scripts/` は触っていない。

そのほかに、動詞の一覧を持つ文書を掃いて、2 件を足した。README の `ralph org` の動詞の表に `gate` がなかった。quality-gates の「Cross-org watch gate」の行が「見張りがなくても動く動詞」を並べるのに、`gate` を挙げていなかった。

## Changes made

| File | D | Change |
|------|---|--------|
| `AGENTS.md:90`(repo map の `internal/org/` の行) | D1 | gate の項目を「`gate.go` with the script running in `gate_run.go`: ...」にした。スクリプトを走らせるのは e0adb359 で分けた `gate_run.go` で、この行に名前がなかった。同じ句の中に収め、行は 1 句しか伸びていない |
| `README.md`(機能ごとの org の段落、「One org handles one feature ...」) | D2 | 「The leader opens the PR with `gh pr create` and disbands the org」を「The leader rebases the branch on the default branch, runs `ralph org gate` (below), opens the PR with `gh pr create`, and disbands the org」にした。ほかの文は変えていない |
| `README.md`(`approve` の段落の次に新しい段落) | D2 | `ralph org gate` の段落を足した。escalate・watch・approve の段と同じ並びと文体で、leader が打つ時点(review のあと、rebase と全コミットのあと、push する head)、走らせる 2 つのスクリプトと環境の固定(`RALPH_VERIFY_*`・`HARNESS_VERIFY_*` を除き `RALPH_VERIFY_SCOPE=full`)、ログの置き場所(`<state dir>/gates/`)、`gate_passed`/`gate_failed` と `reason` の 4 つ、出力の `GATE:`・`HEAD:`・`BASE:`、終了コード 1、走らせる前の拒否、見張りと承認が要らないこと、gate なしで PR を出せること、を書いた。値と文言は `go run ./cmd/ralph org gate --help` と `gate.go` の拒否の文から取った |
| `README.md:124`(コマンド表の `ralph org ...` の行) | 掃引 | 動詞の列の末尾に `/gate` を足し、`approve ...` の文の次に `gate` の 1 文を足した |
| `docs/quality/quality-gates.md:82` と `templates/base/docs/quality/quality-gates.md:81`(「Quality pipeline gate」の行) | D3 | 「refuses before running ... on a dirty tree, a head that does not contain ..., or a directory outside ...」を、拒否を全部挙げる文にした: 完全な結びつきを持って走っている機能の org でない、worktree の外かブランチ違い(detached HEAD と rebase の途中を含む)、作業ツリーが汚れている、既定のブランチが見つからない、head が既定のブランチの先頭を含まない、`scripts/run-static-verify.sh` か `scripts/run-test.sh` がない・実行できない。2 面の追加文は同じで、残る差は元からの KNOWN_DIFF の「(see docs/tech-debt)」だけ(Python で、root の行からその句を除くと template の行と一致することを確かめた) |
| `docs/quality/quality-gates.md:80` と `templates/base/docs/quality/quality-gates.md:79`(「Cross-org watch gate」の行) | 掃引 | 「`stop`, `disband`, `escalate`, and the inbox verbs work with no watch」に `gate` を足した(両面とも同じ文)。gate は見張りの関門をかけない(AC7、`TestGate_NoWatchGateNoApprovalCheck`) |
| `docs/insights/events/2026-10-13-org-gate.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-gate`、`--cycle auto`、cycle 1、pass)。既存のファイルへの追記で、新しいファイルは増えていない |
| この report | - | 新規 |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | 変更なし。4 面の sha256 は 1 種類。動詞の表の `gate` の行(181 行)、「org の終わり方」の手順 1〜7(758 行から)、「gate の記録」(833 行から)を `--help` と `gate.go`・`gate_run.go`・`org_gate.go` に当て直した。拒否の条件、記録のキー、4 行と 5 行の出力、3 つの信号、拒否のときに stdout に何も出ないこと、見張りと承認が要らないことが合う。leader 雛形の手順 9〜16 とは順と revert の規則が同じ |
| `internal/org/prompts/leader.md`、`reviewer.md`(go:embed。触らない) | ずれなし。leader の手順 9〜16 と skill の手順 1〜7 の対応、reviewer の RESULT の `HEAD:`・`BASE:`(ralph は読まない)は、skill と tech-debt の記述と合う |
| `docs/specs/2026-10-07-org-multi-org-director.md:73`(FR-7「7 段目の後半(7b)で決めたこと」) | 変更なし。gate の動き、環境の固定、revert、`run-verify.sh` の一時ファイル、7c に残したもの(doctor、status、merge してよい順)が実装と合う。FR-7 と FR-8 の checkbox は 7c が残るので未完のまま |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし。`ralph org` の動詞は `start`・`spawn`・`status` を本文で挙げるだけで、`escalate`・`watch`・`approve` も載せていない。機能ごとの org の PR の手順は書いていない。冒頭の「2026-10-07 後継の仕様」の注が、機能ごとの org は後継の仕様を優先すると言っており、本文は書き換えない方針(同じ冒頭の注) |
| `docs/recipes/`(`codex-seat-permissions.md` とその template の写し、`worktrees.md`) | 変更なし。reviewer 雛形の変更は、通ったときの RESULT に `HEAD:`・`BASE:` を足しただけで、recipe の TASK(sandbox probe、ゲートを走らせない、RESULT を送る)の前提は変わらない。live の codex 座席では確かめていない。`worktrees.md` の `cleanup` の手順は変わっていない |
| `.claude/rules/ralph/`(`agent-messaging.md` と template の写し、`post-implementation-pipeline.md`、`subagent-policy.md`、`model-routing.md`、`git-commit-strategy.md`) | 変更なし。gate は agmsg のメッセージを送らず、`HEAD:`・`BASE:` は自由ヘッダなので、TYPE の表と TASK_ID の規則に影響しない。機能の org の PR の手順を書く rule はない。`run-verify.sh` の受け渡しのファイル名(`verify-exit-code`・`verify-scope`・`non_docs_change`)を挙げる rule や文書もない(`git grep` で出るのは `scripts/`、`tests/`、計画だけ) |
| `docs/quality/quality-gates.md` の `run-verify.sh` の説明(25〜39 行)と `docs/quality/definition-of-done.md` | 変更なし。`run-verify.sh` の既定の範囲が `full` であること、wrapper の既定が `changed` であることは、gate が `RALPH_VERIFY_SCOPE=full` を足す動きと矛盾しない。受け渡しのファイルを一時ファイルにしたのは内部の変更で、文書は名前を挙げていない。definition-of-done の org の節は、仕様と rule を指すだけ |
| `docs/architecture/repo-map.md`(`tests/` の行、`.harness/state/org/` の行) | 変更なし。`tests/` は `tests/test-*.sh` の総称で、新しい `tests/test-run-verify-concurrency.sh` を含む。`.harness/state/org/` の行は manifest・saga・receipts・watchdog の状態を挙げる要約で、受信箱や `splits/` も挙げておらず、新しい `gates/`(gate のログ)だけを足すと粒度がずれる |
| `templates/base/ralph.toml` | 変更なし。gate は設定を持たない(計画の Non-goals) |
| `templates/base/scripts/run-verify.sh`(root と同じ中身) | `cmp` で同一。ヘッダのコメントが `ralph org gate` を名指すが、`ralph` の CLI の名前で、meta-repo のパスではない。`./scripts/check-template-purity.sh` は rc 0 |
| `docs/tech-debt/README.md`(`:155` の追記と、末尾の 10 行) | 変更なし。参照するパスと関数・テストの名前をすべて取り出して `git grep` で確かめた(欠け 0。スクリプトは scratchpad)。行番号は使っておらず、関数名で指している。計画のパスは `docs/plans/active/` で、`archive-plan.sh` が `/pr` で書き直す。`run-static-verify.sh` の `tech-debt README plan references` も OK。leader 雛形の手順番号(10、12、13、14)は雛形の今の番号と合う |
| `AGENTS.core.md`(`.ralph/core/` と template の写し)、`templates/base/AGENTS.md` | 変更なし。`gate.go` を挙げていない(meta-repo の repo map だけが `internal/org/` を書く)。AGENTS.md の 90 行は managed block の外の meta-repo の行 |
| `docs/reports/`、`docs/evidence/`、`docs/plans/active/2026-10-13-org-gate.md` | 編集していない。計画の Progress checklist(`Test artifact created`)と AC の checkbox の印は、依頼に従って触っていない |

## Left open

- D4: 計画の Progress checklist の `Test artifact created`、AC1〜AC13 の checkbox は未チェックのまま。依頼が計画を編集しないと言っているので、`/pr` の前に計画の持ち主が付ける
- `docs/evidence/test-2026-10-13-org-gate-go-test.txt`(4 行)、`-race.txt`(4 行)、`-mutation.txt`(10 行)に、ホームの絶対パス(`/Users/<user>/...`)が残っている。`docs/quality/definition-of-done.md` の末尾は、`docs/evidence/` にコミットする evidence で `$HOME` を `~` にしてから commit するよう求めている。verify の evidence 4 つには残っていない。同じ形は前の PR の `org-plan-approval` の evidence(`go-test` と `go-race` は各 4 行、mutation は 1〜2 行)にもあり、base の 1bddc871 に入っている。evidence は /test の成果物で、この run の範囲(文書の同期)の外なので直していない。直すなら `/Users/<user>` を `~` に置き換える 1 コミットで足りる。mutation のファイルの `-overlay` の JSON の中のパスは、置き換えると JSON の意味(実在するパス)が変わる点に注意
- leader と reviewer の雛形どおりに、herdr の pane の中の本物の leader が gate を打てるかは未確認(verify の Coverage gaps と tech-debt の `:222`・`:228`)。文書では埋まらない
- release のノートと PR の本文は、この run の範囲に含めていない(`/pr`)

## Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-sync.sh`: rc 0、`IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`
- `./scripts/check-template-purity.sh`: rc 0、`PASS: no meta-repo-specific references found in templates.`(`templates/base/docs/quality/quality-gates.md` を変えたので打った)
- `./scripts/run-static-verify.sh`(changed の範囲): rc 0。`shared:scripts/run-verify.sh` の full fallback で golang の pack が走り、`gofmt: ok`、`0 issues.`、branch の secret scan は `1bddc871..bb2ed920` で clean。`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、`tech-debt README plan references` も OK。wrapper が書いた `docs/evidence/verify-2026-10-13-083024.log` は gitignore の対象で、コミットしていない
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-13-org-gate.md`: `69f088275e54`(変わっていない)
- skill の 4 面の `shasum -a 256`: 1 種類
- 2 つの quality-gates の「Quality pipeline gate」の行: root の行から `(see docs/tech-debt)` を除くと template の行に一致する(Python で確認)
- `./scripts/insights-append.sh --slug org-gate ...` のあとの `git status --short`: 既存の `docs/insights/events/2026-10-13-org-gate.jsonl` が `M` で、新しい `??` のファイルはない

## Cycle 2

- Date: 2026-10-13(JST)
- Plan: `docs/plans/active/2026-10-13-org-gate.md`(承認済み。digest は `69f088275e54` で、`- Approved:` の行と同じ。AC の checkbox を付けたあとも変わらない)
- Pipeline cycle: 2(`--cycle auto`。`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 回の最後)。run 開始時の HEAD は `7f99385e`(run 開始時の値で、この cycle の commit の SHA ではない)。cycle 1 の sync-docs の内容は `6716a028`
- cycle 1 のあとに入ったもの: `b04289c1`(スクリプトが終わったあと、記録の前に届いた中断も何も記録しない。`recordGate(ctx, ...)`、`withManifestLockContext`、`withFileLockContext`)、cross-review cycle 1 の triage(`789c8864`)、self-review cycle 2(`bedc0ebc`、LOW 6、C2-1 から C2-6)、verify cycle 2(`0bc87603`、pass、C2-D1 と C2-D2)、test cycle 2(`7f99385e`、pass、C2h が生き残り)
- 項目の番号: C2-1 から C2-6 は self-review cycle 2、C2-D1 と C2-D2 は verify cycle 2 の「Documentation drift(cycle 2)」の表、C2h は test cycle 2 の「Mutation(cycle 2)」の番号

### Summary

b04289c1 が変えた動きを書いた文書は、すべて合っていた。直した文書のずれは 0 件。変えたのは、tech-debt の 1 行、計画の AC の checkbox 13 個、insight event の 1 行で、Go のコード、テスト、`--help` の文字列、leader と reviewer の雛形、skill、`scripts/`、README、AGENTS.md、quality-gates、仕様は触っていない。

verify cycle 2 が挙げた C2-D1(tech-debt に C2-1 から C2-6 の行がない)と C2-D2(計画の AC の checkbox が未チェック)は、この run で閉じた。

### Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md:231`(org-gate の行の末尾、「The gate strips only the variables ...」の行の次) | 新しい 1 行「Small findings of org-gate cycle 2; none changes behavior (all LOW).」を足した。5 列で、既存の「Small findings of org-plan-approval ...」の行と同じ形(Debt に (a) から (i)、Impact、Why deferred、Trigger、Related)。(a) から (f) は self-review cycle 2 の C2-1 から C2-6、(g) は verify cycle 2 の `lockfile.go` の doc の古い記述(情報)、(h) は test cycle 2 の C2h(本物の 5 秒の `manifestLockTimeout` に届くテストがなく、`withFileLockContext` の「timed out after」の枝が未カバー)、(i) は CLI のテスト `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` が 80 回のうち 5 回、5.7 秒前後かかった件(5 秒は `gateKillGrace` と同じ。原因の推測は未確認と書いた)。Related は計画のパス(`docs/plans/active/` のまま。`archive-plan.sh` が `/pr` で書き直す)と、3 つの cycle 2 の report |
| `docs/plans/active/2026-10-13-org-gate.md:100-112`(`## Acceptance criteria`) | AC1 から AC13 の `- [ ]` を `- [x]` にした(13 行、`git diff --numstat` が 13 / 13)。verify の「Spec compliance」の表は AC1 から AC13 のすべてを「満たす」とし(cycle 2 は b04289c1 を受けて AC5 を見直し、満たすと判断)、test の cycle 1 と cycle 2 の判断は pass で失敗 0 件。AC9 で書き換えた既存の assertion の 3 本は、計画の Notes(177 行)の記録と合っている。これらの checkbox は `## Progress checklist` の外にあるが、依頼が AC の checkbox を付けるよう言ったので付けた。ほかの文は 1 字も変えていない |
| `docs/insights/events/2026-10-13-org-gate.jsonl` | `sync_docs` の event を 1 行足した(`--slug org-gate`、`--cycle auto`、cycle 2、pass)。既存のファイルへの追記で、新しいファイルは増えていない |
| この report | `## Cycle 2` を追記 |

tech-debt の行は、コードを `file:line` ではなく関数名・テスト名で指す(依頼は `lockfile.go` の 28-29、34、69 行と書いたが、対応する doc を `manifestLockTimeout`、`manifestLockPollInterval`、`withFileLock` の doc と名前で書いた。この PR の直しで行がずれるため)。

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `internal/cli/org_gate.go` の `--help`(73 行からの「An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the running script and records nothing」) | 合う。b04289c1 のあとは、スクリプトが終わってからの中断も何も記録しないので、cycle 1 の時点で文が破れていた窓が閉じた。スクリプトが走っていなければ「stops the running script」は空振りするだけで、誤りではない(verify cycle 2 と同じ判断)。文字列は変えていない |
| `/org` skill の 4 面(動詞の表の `gate` の行の末尾「SIGINT・SIGTERM・SIGHUP では走っているスクリプトを止め、何も記録しない」、「org の終わり方」の手順 4 の「ツールに切られた gate は何も記録しない」、「gate の記録」の「中断された gate は何も記録しない」) | 合う。変更なし。4 面の sha256 は 1 種類、`./scripts/check-skill-sync.sh` は rc 0(13 skill) |
| `internal/org/prompts/leader.md`(go:embed。手順 12 の「ツールに切られた gate は何も記録しない」) | 合う。触らない |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`(「Quality pipeline gate」「Cross-org watch gate」の行)、README の `ralph org gate` の段落、`AGENTS.md:90`、`docs/specs/2026-10-07-org-multi-org-director.md:73`(FR-7 の「7 段目の後半(7b)で決めたこと」) | 変更なし。この 5 か所は中断も manifest のロックの待ちも書いていない(`git grep -i` で `interrupt`・`中断`・`SIGINT`・`SIGTERM`・`SIGHUP`・`ctrl-c` を探して 0 件)ので、b04289c1 が変えたものを記述していない。README に「中断は何も記録しない」を足す案は、help と skill が書いているので採らなかった。7c に残したもの(doctor、status、merge してよい順)と FR-7 の checkbox の扱いは cycle 1 のまま |
| `docs/tech-debt/README.md` の既存の org-gate の行(`:221` から `:230`)と `:155` | 変更なし。中断に触れるのは SIGKILL で残るスクリプトの行(`ralph org gate` が SIGKILL で切られたときの行)で、「SIGINT、SIGTERM、SIGHUP か command の context の終わりで止まる」は b04289c1 のあとも合う。ロックの 5 秒の timeout の文を引く行はない。コードの指し方はどの行も関数名とテスト名で、行番号は使っていない |
| `internal/org/lockfile.go` の doc(`manifestLockTimeout`、`manifestLockPollInterval`、`withFileLock`) | ずれあり(Go のコメント。情報)。b04289c1 のあと `withManifestLock` は `withManifestLockContext` から `withFileLockContext` を通り、`withFileLock` は inbox のロック(`inbox.go`)だけが使う。3 つの doc は「`withFileLock` が両方のロックを担う」と書く。cycle 2 は上限の最後で、`.go` の編集は 3 回目の pipeline が要るので、tech-debt の行の (g) に記録し、直していない |
| `docs/architecture/repo-map.md`、`.claude/rules/ralph/`、`docs/recipes/`、`docs/quality/definition-of-done.md` | 変更なし。cycle 1 の表のとおり。b04289c1 は `gate.go`、`lockfile.go`、`gate_test.go` の 3 ファイルだけで、コマンド、スクリプト、skill、雛形、設定の名前は増えも減りもしていない |
| `docs/reports/`、`docs/evidence/` | 編集していない(この report を除く)。cycle 1 の「Left open」にあった evidence のホームの絶対パスは、`bf80094a` と test cycle 2 が `~` に直しており、`grep '/Users/[a-z]'` が `docs/evidence/` の org-gate の 10 ファイルで 0 件 |

### Left open

- Go のコメント・文言・テストの直し(C2-1 から C2-6、`lockfile.go` の doc、C2h のテスト、遅い CLI のテスト)は、tech-debt の `:231` に記録した。3 回目の pipeline が要るので、上限の最後のこの run では直さない。直すなら C2-2(中断の文言)、C2-4(`recordGate` の `switch`)、C2-1(コメントの出典)を先にし、C2h は 5 秒のテストか、テストで縮められる `manifestLockTimeout` を選ぶ
- verify cycle 2 の AC10 の LOW のずれ 2 件(C2-D1、C2-D2)は閉じた
- 本物の herdr の pane の中の leader が、gate をバックグラウンドで打って信号を送る形は未確認(verify の Coverage gaps。cycle 1 と同じ)
- cross-review cycle 2 の triage と、release のノート、PR の本文は、この run の範囲に含めていない(`/cross-review`、`/pr`)

### Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-sync.sh`: rc 0、`IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`。skill の 4 面の `shasum -a 256` は 1 種類
- `./scripts/check-template-purity.sh`: rc 0、`PASS: no meta-repo-specific references found in templates.`
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-13-org-gate.md`: `69f088275e54`(AC の checkbox を付けたあとも同じ)
- tech-debt の新しい行: HEAD の README と比べて既存の 230 行は 1 字も変わらず(先頭からの一致)、新しい行は 5 列(`|` で分けた 7 片)、列ごとの backtick は偶数、`file:line` の形は 0 件、計画のパスは `docs/plans/active/2026-10-13-org-gate.md` の 1 つ、U+FFFD は 0 件(Python。スクリプトは scratchpad)
- `./scripts/run-verify.sh`(full): rc 0、`==> All verifiers passed.`。tech-debt の plan 参照の検査(`tech-debt README plan references`)は OK、`check-sync.sh` は `PASS: all files in sync.`、`check-skill-sync.sh` は 13 skill、shell の test は `tests/` の 41 本が通り(FAIL の行 0)、golang の pack は `gofmt: ok`、`0 issues.`、`go test` は test のある 8 パッケージがすべて ok(`internal/cli` 138.8 s、`internal/org` 68.6 s、ほかは `(cached)`)、branch の secret scan は `scanned 1bddc871..7f99385e against origin/main: clean`。この 1 行を足す前の tree で打った。run のあとに変えたのは、この行だけ。wrapper が書いた `docs/evidence/verify-2026-10-13-123656.log` は gitignore の対象で、コミットしていない
- `./scripts/insights-append.sh --slug org-gate ...` のあとの `git status --short`: 既存の `docs/insights/events/2026-10-13-org-gate.jsonl` が `M` で、新しい `??` のファイルはない

## Cycle 3

- Date: 2026-10-13(JST)
- Plan: `docs/plans/archive/2026-10-13-org-gate.md`(f001bd4f で archive に移した。`./scripts/plan-visual.sh digest` は `69f088275e54` で、`- Approved:` の行と同じ。この run は計画を編集していない)
- Pipeline cycle: 3(PR #225 の CI が落ちたので、上限を 3 に上げて回した。`--cycle 3` を明示して付けた。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)。run 開始時の HEAD は `b72ddcce`(run 開始時の値で、この cycle の commit の SHA ではない)。cycle 2 の sync-docs の内容は `a1590e44`
- cycle 2 の sync-docs のあとに入ったもの: `84bbf4fb`(cross-review cycle 2 の triage、指摘 0)、`2d6567b9`(walkthrough)、`f001bd4f`(計画の archive と tech-debt の参照の書き換え)、`8b681dc3`(`gateSignalContext`。中断のエラーが信号の名前を言う)、`a538c899`(self-review cycle 3、LOW 2 件の C3-1 と C3-2)、`697e1577`(verify cycle 3、pass、C3-D1 から C3-D4)、`b72ddcce`(test cycle 3、pass。mutation の M_d2 と M_j が生き残り、M_a は go1.26.0 でだけ生き残る)
- 項目の番号: C3-1・C3-2 は self-review cycle 3、C3-D1 から C3-D4 は verify cycle 3 の「Documentation drift(cycle 3)」の表、M_a・M_d2・M_j は test cycle 3 の「Mutation(cycle 3)」の番号

### Summary

8b681dc3 は中断のエラーの文面(信号の名前)を変えたが、その文面を書いた文書(`--help`、skill の 4 面、雛形、README、AGENTS.md、quality-gates、仕様)はなく、動きより多くを主張する文もなかった。直したのは verify cycle 3 の記録のずれ 4 件(C3-D1 から C3-D4)で、変えたのは walkthrough 1 つ、tech-debt の 2 か所(古い名前の直しと新しい 1 行)、insight event の 1 行。Go のコード、テスト、`--help` の文字列、leader と reviewer の雛形、skill、`scripts/`、README、AGENTS.md、quality-gates、仕様、計画、ほかの報告は触っていない。

### Changes made

| File | D | Change |
|------|---|--------|
| `docs/reports/walkthrough-2026-10-13-org-gate.md:31-34`(節 4) | C3-D1 | 見出し(`:31`)の commit の列に 8b681dc3 を足した。`signal.NotifyContext` と書いていた行(`:33`)を `gateSignalContext` に替え、その項目を 1 つ足した(`:34`): 最初の信号で `<信号> signal received` を cause にして取り消す。`signal.NotifyContext` が信号を cause に付けるのは Go 1.26 からで、go.mod は 1.25.8(release の build も同じ)なので使わない。きっかけは PR #225 の CI の失敗。後の信号も stop が呼ばれるまで捕まえる。`signal.NotifyContext` は `org_watch.go` に 1 か所残る(cause を読まない) |
| 同 `:56` と表(`:68-71`) | C3-D1 | 「上限は既定の 2 のまま」を、CI の失敗のあとで上限を 3 に上げた経緯に替えた(`cycle-count.json` は 2 のまま、cycle 3 の cross-review はこのあとで結果は書かない)。表に 4 行足した: PR #225 の CI(run 38401704870 と 38401983445)、self-review cycle 3、verify cycle 3、test cycle 3。cross-review cycle 3 の行は作っていない(結果がまだない) |
| 同 `:75`(実機確認) | C3-D1 | cycle 3 の probe を足した。go1.25.8 で build した HEAD の binary への 7 通りの入力で、中断はどれも rc 1 で信号の名前を出し、修正前の f001bd4f の binary は「context canceled」を出した |
| 同 `:13` | C3-D2 | `leaderActivityEventCount` の項目の「(AC13)」を「(AC6)」にした。計画の AC6 が `leaderActivityEventCount` の項目で(`docs/plans/archive/2026-10-13-org-gate.md:105`)、AC13 は archive した計画の TASK の検査(同 `:112`)。2d6567b9 からあった誤り |
| 同 `:4-6`(見出しの 3 行) | C3-D1 | Plan の行を「PR 作成時に active から移す」から「f001bd4f で active から移した」にした。Branch の行を、この commit を含めた commits と files の数と、書いた時点の数(25 commits、45 files、+25324/-214)にした。読む範囲の行を、evidence の増加(約 2 万行から約 3 万行)、`internal/cli/org_gate_test.go` の +832 から +909、報告・evidence・insight を除いた数に合わせた。数は `git diff --shortstat 1bddc871` と `git rev-list --count 1bddc871..HEAD` で数えた(元の「報告と計画を除く」は、実際には計画を含めて報告・evidence・insight を除いた数だった) |
| `docs/tech-debt/README.md:228`(SIGKILL の行) | C3-D3 | Related の列の `signal.NotifyContext` を `gateSignalContext` にした。行の本文は動きと合っていたので変えていない |
| `docs/tech-debt/README.md:232`(新しい行) | C3-D4 | 「Small findings of org-gate cycle 3 ...」を足した。(j) C3-1(`gateSignalContext` の doc に、標準の関数へ戻す合図がない。go.mod の `go` 行が 1.26 以上になったとき)、(k) C3-2(テスト名が後の信号の半分しか言わない)、(l) M_d2(`stop` が `signal.Stop` を呼ばなくても落ちるテストがない。test cycle 3 の probe がそのまま使える)、(m) M_j(信号の channel に buffer がない mutant が生き残る。狙いにくい窓)、(n) toolchain の差(手元の Go 1.26 が Go 1.25 だけの退行を隠す。`verify.yml` に `setup-go` がなく、CI は go.mod の版ではなく runner の Go で走る。`setup-go` を足す chore PR は PR #225 のすぐあとの予定)。5 列で cycle 2 の行(`:231`)と同じ形。記号は cycle 2 の (a) から (i) の続き。コードは関数名とテスト名で指し、`file:line` は使っていない。計画のパスは archive のパスで書いた。Related は cycle 3 の報告 3 つ、mutation の evidence、コード、`verify.yml`、8b681dc3 |
| `docs/insights/events/2026-10-13-org-gate.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-gate`、`--cycle 3`、pass)。既存のファイルへの追記 |
| この report | - | `## Cycle 3` を追記 |

C3-1 と C3-2 は、self-review cycle 3 が cycle 2 の行(`:231`)の (j)・(k) に足すよう勧めていた。cycle 2 の行は 6 KB を超え、(a) から (i) が cycle 2 の上限と結びついているので、同じ形の兄弟の行に分けた。記号は続けたので、(j)・(k) の呼び名は変わらない。

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `ralph org gate --help`(`internal/cli/org_gate.go:73-76`) | 合う。「An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the running script and records nothing」は 8b681dc3 のあとも成り立ち、エラーの文面は約束していない。文字列は変えていない |
| `/org` skill の 4 面 | 変更なし。8b681dc3 は skill に触れておらず、4 面の sha256 は 1 種類で、`./scripts/check-skill-sync.sh` は rc 0(13 skill)。中断を述べる 3 か所(動詞の表の `gate` の行、手順の「ツールに切られた gate は何も記録しない」、「gate の記録」の「中断された gate は何も記録しない」)は動きと合う |
| `internal/org/prompts/leader.md`(go:embed) | 変更なし。中断を述べる「ツールに切られた gate は何も記録しない」は動きと合う |
| README、AGENTS.md、`docs/quality/quality-gates.md` と template の写し、`docs/specs/2026-10-07-org-multi-org-director.md`(FR-7)、`docs/architecture/repo-map.md`、`docs/recipes/`、`.claude/rules/ralph/` | 変更なし。`git diff --stat 0bc87603..HEAD` で、これらと `scripts/`、`tests/`、`internal/org/`、`templates/`、`.claude/`、`.agents/`、`.codex/` の差分は空。`git grep -i` で SIGINT・SIGHUP・interrupted・ctrl-c を探して、中断の動きを述べる行は skill の `gate` の行だけで(ほかは `watch` の行と、別の話の recipe の 2 行)、cause の文(`signal received`)を書く文書は 0 件 |
| `docs/tech-debt/README.md` の既存の org-gate の行 | `signal.NotifyContext` を名指していたのは `:228` の 1 行だけ(`git grep` で確認し、直した)。コードは関数名とテスト名で指し、`org_gate.go` の行番号で引く行はない。`:231`(cycle 2 の行)は変更なし |
| ほかの文書の中の `org_gate.go` と `org_gate_test.go` の行番号 | 8b681dc3 で `org_gate.go` の 96 行目より後ろが 38 行下がったが、`git grep 'org_gate\(_test\)\?\.go:[0-9]'` が reports・evidence・archive の計画の外で 0 件。報告の中の行は、verify cycle 3 の引き直しの表が現在の行を示す |
| 過去の報告 | 編集していない(この report と walkthrough を除く)。verify cycle 1・2 の表の `signal.NotifyContext`(`org_gate.go:96`)は、そのときの記録として残す |
| `.github/workflows/verify.yml` と `release.yml` | 変更なし。`verify.yml` に `setup-go` がないことは tech-debt の (n) に記録した。`release.yml:21` は `go-version-file: go.mod` |
| Go のコメントの計画のパス | verify cycle 3 の情報の項目と同じで、8 か所が `docs/plans/active/2026-10-13-org-gate` のまま。archive 済みのほかの計画も同じ慣例なので、ずれに数えず、Go のファイルは触らない |

### Left open

- cross-review cycle 3 はこの commit のあとに走る。walkthrough のパイプラインの表には結果を書いていない。結果が出たら `/pr` の前に 1 行足す
- PR #225 の本文は cycle 1・2 の結果だけを挙げ、cycle 3 と 8b681dc3 を書いていない(verify cycle 3 の情報の項目)。更新は `/pr` の範囲
- tech-debt の (j) から (n) は、cycle 3 が上限の最後なので `.go` の直しを含めて今は直さない。(n) の `setup-go` は、PR #225 のあとの chore PR で足す予定
- 本物の leader の座席での中断(herdr の pane、codex の sandbox、ツールの timeout が送る信号)は未確認で、cycle 1・2 と同じ
- CI の Go の正確な版はログに出ないので未確認(verify cycle 3 と同じ)

### Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-sync.sh`: rc 0、`IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`。root と template の写しは触っていないので、数は cycle 2 と同じ
- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`。skill の 4 面の `shasum -a 256` は 1 種類
- `./scripts/check-template-purity.sh`: rc 0、`PASS: no meta-repo-specific references found in templates.`
- `./scripts/plan-visual.sh digest docs/plans/archive/2026-10-13-org-gate.md`: `69f088275e54`(変わっていない)
- tech-debt の行: 変わったのは HEAD と比べて `:228` と `:232` の 2 行だけで、ほかの行は 1 字も変わらない。`:232` は 5 列(`|` で分けた 7 片)、列ごとの backtick は偶数、`file:line` の形は 0 件、`docs/plans/active/` は 0 件、archive の計画のパスは 1 つ、U+FFFD は 0 件。`:228` に `signal.NotifyContext` は残らない(Python。スクリプトは scratchpad)
- walkthrough: 見出しの数の置き場所(`:5`、`:6`)に未置換の目印は残らない。32 commits は `git rev-list --count 1bddc871..HEAD` の 31 にこの commit の 1 を足した数、46 files、+36404/-214、報告・evidence・insight を除いた 29 files、+5535/-214 は、この report の最終版を含む作業ツリーと `1bddc871` の `git diff --shortstat`(この commit は新しいファイルを足さない)
- `./scripts/run-verify.sh`(full): rc 0、`==> All verifiers passed.`。tech-debt の plan 参照の検査(`tech-debt README plan references`)は OK、`check-sync.sh` は `PASS: all files in sync.`、`check-skill-sync.sh` は 13 skill、shell の test は `FAIL: [1-9]` の行が 0 件、golang の pack は `gofmt: ok`、`0 issues.`、`go test` は 8 パッケージがすべて ok(`internal/cli` 133.4 s、`internal/org` 65.8 s、ほかは `(cached)`)、branch の secret scan は `scanned 1bddc871..b72ddcce against origin/main: clean`。この 1 行を埋める前の tree で打った。run のあとに変えたのは、この節の目印の置き換えだけ(行数は変わらない)。wrapper が書いた `docs/evidence/verify-2026-10-13-145601.log` は gitignore の対象で、コミットしていない
- `./scripts/insights-append.sh --slug org-gate ...` のあとの `git status --short`: 既存の `docs/insights/events/2026-10-13-org-gate.jsonl` が `M` で、新しい `??` のファイルはない

## Cycle 4

- Date: 2026-10-14(JST)
- Plan: `docs/plans/archive/2026-10-13-org-gate.md`(archive 済みで、この run は編集していない。`./scripts/plan-visual.sh digest` は `69f088275e54` で、`- Approved:` の行と同じ)
- Pipeline cycle: 4(cross-review cycle 3 の #1 を直すために、上限を 4 に上げて回した。`--cycle 4` を明示して付けた。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)。run 開始時の HEAD は `6f557f14`(run 開始時の値で、この cycle の commit の SHA ではない)。cycle 3 の sync-docs の内容は `09baf8eb`
- cycle 3 の sync-docs のあとに入ったもの: `b9a95f05`(cross-review cycle 3 の triage、ACTION_REQUIRED 1 件)、`f1b0352e`(reviewer のレポートのコミットを rebase の前に動かした。`leader.md` の手順 9・10、`prompts_test.go`、`/org` skill の 4 面)、`31ae9f7d`(self-review cycle 4、LOW 3 件の C4-1 から C4-3)、`1e4a2fa4`(verify cycle 4、pass、C4-D1 から C4-D3)、`6f557f14`(test cycle 4、pass。skill の締めの順を読むテストがなく、mutant E が生き残った)
- 項目の番号: C4-1 から C4-3 は self-review cycle 4、C4-D1 から C4-D3 は verify cycle 4 の「Documentation drift(cycle 4)」の表、E と R5 は test cycle 4 の「Mutation(cycle 4)」と「git の probe(cycle 4)」の番号

### Summary

f1b0352e は、reviewer が通したあとの順を「rebase → report と reviewer のレポートのコミット」から「reviewer のレポートのコミット → rebase → report だけのコミット」に変えた。この順を並べて書く文書のうち、合っていなかったのは README の gate の段落、walkthrough、tech-debt の行(C4-1 から C4-3 がない)の 3 つで、verify cycle 4 が挙げた C4-D1 から C4-D3 をこの run で閉じた。leader の雛形と skill の 4 面は f1b0352e 自身が同じ順に直していて、ほかの文書(quality-gates の 2 面、AGENTS.md、仕様の注、`--help`、skill の動詞の表)は順を書かないか、状態として書くので合っていた。

変えたのは README の 1 文、walkthrough、tech-debt の 1 行、insight event の 1 行で、Go のコード、テスト、`--help` の文字列、leader と reviewer の雛形、skill、`scripts/`、AGENTS.md、quality-gates、仕様、計画、ほかの報告は触っていない。

### Changes made

| File | D | Change |
|------|---|--------|
| `README.md:260`(`ralph org gate` の段落の 2 文目) | C4-D2 | 「the leader fetches and rebases the branch on the default branch, commits everything, and runs it」を「the leader commits the reviewer's reports, fetches and rebases the branch on the default branch, commits the org report and the plan's archive, and runs it」にした。reviewer が通したあとの動作を順に並べる文なので、最初の動作を reviewer のレポートのコミットにし、「commits everything」を report と archive の 2 つに分けた。ほかの文は変えていない。`README.md:252` の要約(「The leader rebases the branch on the default branch, runs `ralph org gate` ...」)はコミットに触れないので、ずれに数えず、変えていない |
| `docs/reports/walkthrough-2026-10-13-org-gate.md:5`・`:6` | C4-D1 | commit・file・行数を、cycle 4 の sync-docs の commit を含めた数に替えた(38 commits、46 files、+30459/-218。報告・evidence・insight を除くと 29 files、+5624/-218)。書いた時点の数(25 commits、45 files)は残し、増えた理由に cycle 4 の直し f1b0352e を足した。evidence の行数は「約 3 万行」から「約 2 万 3 千行」にした(`git diff --shortstat 1bddc871 -- docs/evidence` が 23034 行) |
| 同 `:45`・`:47`・`:48`(節 6) | C4-D1 | 見出しの commit の列に f1b0352e を足した。`:47` は雛形の順を「reviewer のレポートのコミット(`git status --porcelain` が空かを確かめる。残りは implementer に TASK でコミットを頼むか BLOCKED)→ fetch と rebase → `ralph org report` を打って report だけをコミット → 計画の archive を別のコミット → ...」に替えた。`:48` に項目を 1 つ足した: 順を変えた理由(gate をやり直す通過で、コミット済みのレポートを reviewer に書き換えられると、`git rebase` が `cannot rebase: You have unstaged changes` で断られ、`git rebase --abort` も効かない)、計画の Scope と AC8 は逆の順で書いてあり、計画は archive 済みで書き換えないので意図したずれであること、skill の 4 面も同じ順にしたこと |
| 同 `:51`・`:52` | C4-D1 | `:51` に、既存の assertion を書き換えた 3 本のうち、節の冒頭の文の 1 本を f1b0352e が書き直したこと、手順 9 の中の順序と手順 10 の否定の assertion が増えたこと、skill の締めの順を読むテストがないことを足した。`:52` の `/org` skill の行に「org の終わり方」の順を足した |
| 同 `:57` と表(`:73`〜`:76`) | C4-D1 | 上限の経緯に、cycle 3 の cross-review が ACTION_REQUIRED 1 件を出して上限を 4 に上げた話を足した(`cycle-count.json` は 2 のまま)。cross-review cycle 4 は「このあとで、この表に結果はない」とした。表に 4 行足した: cross-review cycle 3(ACTION_REQUIRED 1 件、f1b0352e)、self-review cycle 4(LOW C4-1 から C4-3、tech-debt の (o) から (q))、verify cycle 4(C4-D1 から C4-D3)、test cycle 4(pass、mutant E が生き残り、tech-debt の (r)・(s))。cross-review cycle 4 の行は作っていない(結果がまだない) |
| 同 `:80`(実機確認) | C4-D1 | cycle 4 の git だけの probe(verify の P1〜P3、test の R1〜R5)を足した。旧い順では `git rebase` が rc 1、始まっていない rebase への `git rebase --abort` が rc 128、先にレポートをコミットすれば rebase が rc 0 |
| `docs/tech-debt/README.md:232` | C4-D3 | 「Small findings of org-gate cycle 3 ...」の行を伸ばし、題を「Small findings of org-gate cycles 3 and 4; none changes behavior (all LOW).」にした。(o) C4-1(手順 9 の最初の commit が空振りする場合と add の範囲)、(p) C4-2(skill の 2 か所の中途半端な折り返し)、(q) C4-3(テストの doc comment が AC8 を上書きしたと言わない。計画のパスが古い引用 8 か所の 1 つ)、(r) 締めの順を読むテストがない(mutant E)、(s) 手順 9 の `git commit` に commit するものがないときの扱いを雛形が書かず、本物の leader の動きは未確認(R5)を足した。(j) から (n) と、列ごとの区切りは変えていない。(o) から (q) の本文は self-review cycle 4 の「Tech debt identified」の表の文を元にし、行番号は使わず、関数名・テスト名・文言で指した |
| `docs/insights/events/2026-10-13-org-gate.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-gate`、`--cycle 4`、pass)。既存のファイルへの追記 |
| この report | - | `## Cycle 4` を追記 |

### Surfaces checked for drift

締めの順を述べる文を、`git grep` で `rebase`、`reviewer のレポート`、`reviewer's report`、`commits everything`、`rebase → report` を探して突き合わせた(`docs/evidence/`、`docs/reports/` の過去の記録、archive の計画は除いた)。

| Surface | 結果 |
|---------|------|
| leader の雛形(`internal/org/prompts/leader.md` の手順 9・10)と `/org` skill の 4 面(「org の終わり方」の手順 1・2、「Leader 運用 2 経路」の手順 7 の文、「タスク終了時は必ず」の行、完了条件のチェックリスト) | 合う。f1b0352e が同じ順に直した。変更なし。skill の 4 面の `shasum -a 256` は 1 種類で、`./scripts/check-skill-sync.sh` は rc 0(13 skill)。skill の動詞の表の `gate` の行(`:181`)の「既定のブランチに rebase してすべてをコミットしたあと」は、gate を打つ時点の状態(rebase 済みで、すべてコミット済み)を言う文で、新しい順でも成り立つので変えていない(変えるなら 4 面を一緒に) |
| `ralph org gate --help`(`internal/cli/org_gate.go` の「after it rebases the feature branch on the default branch and commits everything」) | 合う(verify cycle 4 の情報と同じ判断)。状態の記述で、順を約束していない。文字列は変えていない |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`(「Quality pipeline gate」の行)、`AGENTS.md` の `internal/org/` の行(`:90`) | 変更なし。「on the rebased head」「on a feature org's rebased head」と拒否の条件だけを書き、締めの順を書かない |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-7 の 7b の注、`:73`) | 変更なし。「review のあとに report と計画の archive のコミットが増える」「計画の archive は report のコミットと別のコミットにし」は、新しい順でも成り立つ(reviewer のレポートのコミットも review のあと)。順を並べた文はない |
| `docs/recipes/`、`.claude/rules/ralph/`、`docs/architecture/repo-map.md`、`docs/quality/definition-of-done.md` | 変更なし。`git grep -n rebase` で、org の締めの順を述べる行は 0 件 |
| Go のコメントの `docs/plans/active/2026-10-13-org-gate` | 8 か所のまま。tech-debt の (q) に記録した。Go のファイルは触らない |
| `prompts_test.go` の「cross-review cycle 3 #1 of org-gate」の comment 2 か所 | 変更なし(verify cycle 4 の情報と同じ)。cross-review cycle 4 が triage 報告の先頭の表を書き換えるなら、cycle 3 の表を `## Cycle 3` の節に移して 1 段落を足す(self-review cycle 4 の Follow-ups 2)。これは `/cross-review` の範囲 |
| 過去の報告 | 編集していない(この report と walkthrough を除く)。verify cycle 4 の C4-D1 から C4-D3 の行は、そのときの記録として残す |

### Left open

- cross-review cycle 4 はこの commit のあとに走る。walkthrough の表には結果を書いていない。結果が出たら `/pr` の前に 1 行足す。triage 報告の先頭の表を書き換えるときは、cycle 3 の表を `## Cycle 3` の節に移し、`prompts_test.go` の 2 か所の「cross-review cycle 3 #1」がその節を指すことを 1 段落で言う(self-review cycle 4 の Follow-ups 2)
- PR #225 の本文は、雛形の順(23 行目)を旧い順で書き、各報告の要約は cycle 2 で止まる(verify cycle 4 の情報の項目)。更新は `/pr` の範囲
- tech-debt の (o) から (s) は、cycle 4 が上限(4 回)の最後なので直さない。直すなら (o) を先に、`leader.md`、skill の 4 面、`prompts_test.go` の手順 9 の行の検査を一緒に直す。(r) は、skill の「org の終わり方」を読む Go のテスト 1 本で閉じる
- (s) の本物の leader の動き(commit が rc 1 で終わったとき止まるか)、手順 9 の枝(implementer の未コミットの変更の TASK、BLOCKED)を本物の leader が選べるか、Linux と codex の sandbox での git は未確認(verify cycle 4 と test cycle 4 の Coverage gaps と同じ)

### Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-sync.sh`: rc 0、`IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`。root と template の写しは触っていない(README は写しのない root のファイル)ので、数は cycle 3 と同じ
- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`。skill の 4 面の `shasum -a 256` は 1 種類
- `./scripts/check-template-purity.sh`: rc 0、`PASS: no meta-repo-specific references found in templates.`
- `./scripts/plan-visual.sh digest docs/plans/archive/2026-10-13-org-gate.md`: `69f088275e54`(変わっていない)
- tech-debt の行: HEAD と比べて変わったのは `:232` の 1 行だけで、ほかの行は 1 字も変わらない。`:232` は 5 列(`|` で分けた 7 片)、列ごとの backtick は偶数、`file:line` の形は 0 件、`docs/plans/active/` は 0 件、plan のパスは `docs/plans/archive/2026-10-13-org-gate.md` の 1 つ、U+FFFD は 0 件(Python。スクリプトは scratchpad)
- walkthrough: 見出しの数の置き場所(`:5`、`:6`)に未置換の目印は残らない。38 commits は `git rev-list --count 1bddc871..HEAD` の 37 にこの commit の 1 を足した数、46 files、+30459/-218、報告・evidence・insight を除いた 29 files、+5624/-218 は、この report の最終版を含む作業ツリーと `1bddc871` の `git diff --shortstat`(この commit は新しいファイルを足さない)
- `./scripts/run-verify.sh`(full): rc 0、`==> All verifiers passed.`。tech-debt の plan 参照の検査(`tech-debt README plan references`)は OK、`check-sync.sh` は `PASS: all files in sync.`、`check-skill-sync.sh` は 13 skill、shell の test は `tests/` の 41 本で `FAIL: [1-9]` の行が 0 件、golang の pack は `gofmt: ok`、`0 issues.`、`go test` は 8 パッケージがすべて ok(`internal/cli` 140.3 s、`internal/org` 62.9 s、ほかは `(cached)`)、branch の secret scan は `scanned 1bddc871..6f557f14 against origin/main: clean`。この 1 行を埋める前の tree で打った。run のあとに変えたのは、この 1 行の置き換えだけ(行数は変わらない)。wrapper が書いた `docs/evidence/verify-2026-10-13-164451.log` は gitignore の対象で、コミットしていない
- `./scripts/insights-append.sh --slug org-gate ...` のあとの `git status --short`: 既存の `docs/insights/events/2026-10-13-org-gate.jsonl` が `M` で、新しい `??` のファイルはない
