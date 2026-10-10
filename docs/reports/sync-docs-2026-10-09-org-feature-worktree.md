# sync-docs report: org-feature-worktree

## Cycle 1

- Date: 2026-10-09
- Plan: `docs/plans/active/2026-10-09-org-feature-worktree.md`(承認済み、digest `56e435bfa976`。編集の前後で同じ値)
- Pipeline cycle: 1(上限 2)。差分は merge base `765da6bd` から branch HEAD `09ad7d5a`(feat/org-feature-worktree)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-09-org-feature-worktree.md`(`2a544af9`、`1ffa4f15` で再実行の結果に更新。Merge 可、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 4 件)、
  `docs/reports/verify-2026-10-09-org-feature-worktree.md`(`421719f7`。pass、V-1〜V-4)、
  `docs/reports/test-2026-10-09-org-feature-worktree.md`(`09ad7d5a`。pass、T-1〜T-7)、
  実機の記録 `docs/evidence/org-feature-worktree-live-2026-10-09.md`

## Summary

古くなっていたのは、`ralph org start` の help の文字列 3 か所(V-1、V-2)、tech-debt の 3 行(関数名と、満たされたトリガー)、実機の記録の出力 1 行(V-4)、plan の進捗のチェックボックスだった。tech-debt に入れていなかった項目(plan の「実装中に見つけて送るもの」(1)〜(4)、self-review の L5・N1・N2、verify の V-3、test report の T-1〜T-7)は 6 行にして足した。

help の文字列は `internal/cli/org.go` の Go のコードなので、これを直した commit は Go の差分を含む。直したのは文字列だけで、関数の挙動は変えていない。pipeline を回し直すかどうかは lead の判断に任せる(この report の「Found but left」の末尾)。

差分の大きさ(`git diff origin/main...HEAD --stat`、`09ad7d5a` 時点、この sync-docs の commit を含まない): 31 files changed, 6815 insertions(+), 374 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `internal/cli/org.go`(V-1) | `--plan` の usage に、`--allow-unscoped` が併用できず拒否されることを足した(`checkOrgStartPlanInput` と `orgStartPlanConflicts` は拒否している。Long の説明と skill は書いてあった) |
| `internal/cli/org.go`(V-2) | start の Long に「org_id は slug か `--org-id`。20 文字まで。leader が `implementer` を herdr の 32 文字の agent 名 `<org_id>_<seat_id>` に収めるため」を足した。persistent の `--org-id` の usage にも「`--plan` では 20 文字まで」を足した。flag の usage にはバッククォートを使わない規則(`orgReserveFlagUsage` の comment)を守った |
| `internal/cli/org.go`(追加。verify は指摘していない) | spawn と start の `--cwd` の usage に「相対パスは打ったディレクトリから絶対パスにする」を足した。af138af6 の挙動で、`/org` skill の `spawn` の行にだけ書いてあり、help になかった |
| `docs/evidence/org-feature-worktree-live-2026-10-09.md`(V-4) | Run 3 の start の出力に、`runOrgStartPlan` が出す 3 行目 `hint: ralph org status --org-id hello ; attach with herdr to observe the leader pane` を足した。コードは af138af6 の時点でこの行を出していた。実機の出力そのものの再取得はしていない |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-4 の「4 段目で決めたこと」) | コードに合わせて 4 点を足した。(1) slug と `--org-id` は 20 文字まで(理由つき)、(2) 結びつきを `ralph org status` が `feature:` の行で出し、不完全な結びつきには `(incomplete record)` を付ける、(3) worktree の使い回しの検査は `canonical_ref`・パス・ブランチ・kind と実際のチェックアウト、(4) 出荷する `/org` skill には director の説明を書かない(director が入る段で足す)。既存の文は 1 つも変えていない。FR のチェックボックスは触っていない。ensure の失敗ごとの案内と相対 `--cwd` の修正は実装の詳細なので、仕様には足していない |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md` | 「Org runtime gates」の表に「Split plan approval」の行を足した(`start --plan` の承認 digest の検査、副作用の前の spawn の検査、worktree の使い回しの条件)。2 つのファイルに同じ行を入れた。もともとある 3 か所の差(secret-scan の行、check-template の行、Quality pipeline gate の行)は変えていない |
| `docs/tech-debt/README.md`(新しい 6 行) | (1) pane の `PATH` 上の `ralph` の版が違うと台帳が分かれる。(2) reviewer の report が `session_end_summary.sh`(`SessionEnd` の hook)の `wip: checkpoint before session end` で PR に入る。(3) 昇格した leader と `start <task>` の leader の `ralph org report` が task worktree に未追跡のファイルを残し、`cleanup` が止まる。(4) 同じ機能の `start --plan` の同時実行。(5) self-review の L5 (a)〜(e)・N1・N2、verify の V-3、この sync-docs が見つけた help の「20」が test にも結びついていないこと。(6) test report の T-1〜T-7(T-4 は (4) への参照)。行はどれも関数名と test 名で書き、`file:line` は使っていない |
| `docs/tech-debt/README.md`(行「Code-shape findings of org-limits-reserve」: F-2・F-6・F-7) | Trigger に 1 文足した。F-2(`scopeReservedEvent`)と F-6(`currentOrgLives`)のトリガーはこの branch が満たしたが、どちらも直していない。引数が `Reservation` に変わっただけで、`reserve` の bool、空のタイムスタンプ、`openOrgWorkspaces` との 2 つ目の fold は残るので、2 つとも open のまま。Related に self-review L2 を足した |
| `docs/tech-debt/README.md`(行「Wording findings of org-limits-reserve cycle 2」: C2-1〜C2-5) | (a) C2-1 の取り消し線と、8aae7ce2 で直した旨(`git grep 'decided first' internal/org` は 0 件)。(c) C2-3 の `if !seat.Active` の場所を `idempotentRespawnDecision` に直した。Trigger の (a) を取り消し線にし、(b)〜(e) が open のままであることを足した。Trigger の (c) にも関数名を足した。Related に plan と self-review L2 を足した |
| `docs/tech-debt/README.md`(行「Test gaps left by the org-limits-reserve cycle 2 fix」: T2-1〜T2-4) | T2-1 と T2-2 の `idempotentRespawn` を `idempotentRespawnDecision` に直した。T2-2 の「呼び出し元は 2 つ」を、`spawnPrecheckErr` を入れた 3 つに直した。Trigger (a) に、この branch が同じコードを変えたのに subtest を足さなかったこと、`TestSpawnPrecheckErr_MatchesSpawn` の「a legacy inactive leader past max_orgs」は重なる予約を持たないので順序を固定しないことを足した |
| `docs/plans/active/2026-10-09-org-feature-worktree.md` | `## Progress checklist` の中だけを変えた。「Verification artifact created」「Test artifact created」にチェックを付け、この sync-docs の 1 行を足した。本文は触っていない。`./scripts/plan-visual.sh digest` は編集の前後とも `56e435bfa976`(`- Approved:` の行と一致)。「PR created」は `/pr` が付けるのでそのまま |
| `docs/reports/sync-docs-2026-10-09-org-feature-worktree.md` | この report |
| `docs/insights/events/2026-10-09-org-feature-worktree.jsonl` | `sync_docs` の event を 1 行追記した(`scripts/insights-append.sh`、`--cycle auto`) |

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | `cmp` で 4 面とも同一。`./scripts/check-skill-sync.sh` は 13 skill が lock-step、`./scripts/check-sync.sh` は DRIFTED 0・IDENTICAL 164。skill の「20 文字まで」(3 か所)、`start` の行(`--cwd` / `--scope` / `--reserve` / `--allow-unscoped` の併用拒否)、`spawn` の行(相対 `--cwd`)、`status` の `(incomplete record)` は、最終のコードと合っている。skill には手を入れていない |
| `internal/org/prompts/leader.md` | 8aae7ce2 と 2c1bb13b のあとの版が、`featureLeaderTask` が組む task の行(`- 分割計画:`、`- ブランチ:`、`- 予約したパス:`、`- 進め方:`)、座席の `--id`(`implementer`、`reviewer`)、締めの順(stop → report → archive・commit → secret scan → push → `gh pr create` → disband)と合っている。N2(冒頭の例外が reviewer の report を挙げない)は tech-debt の 1 行に送った |
| `README.md` と `AGENTS.md` | `README.md` の `ralph org` の行と「One org handles one feature」の段落、`AGENTS.md` の `internal/org/` の行(`split.go`、`feature.go`)は、最終のコードと合っている。README に 20 文字の上限は書いていない。エラー文、help、skill に書いてあるので足していない |
| 編成パターン(Solo・Leaded・Parallel)の言及 | `git grep` で `docs/reports`、`docs/plans`、`docs/evidence`、`docs/insights` を除いて検索した。出荷する面(skill、雛形、README、AGENTS.md、`templates/base/`)は 0 件。残るのは、廃止を決めた仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)と、先頭に「この仕様の本文は書き換えない。新しい仕様が優先する」と書いた `docs/specs/2026-08-01-org-runtime.md`(31 行目の注記)と、廃止を固定するテスト(`prompts_test.go`)だけ |
| `docs/architecture/repo-map.md` | `.claude/skills/org/` と `.harness/state/org/` の行が、共通の台帳を書いている。この branch は script も skill も足していないので、変更なし |
| `docs/quality/quality-gates.md` | 「Org runtime gates」に承認の検査がなかったので足した(上の表)。`definition-of-done.md` は pipeline の順だけを書くので変更なし |
| `ralph org start --help`、`ralph org spawn --help`、persistent flags | 作り直した binary(scratchpad に置いた)で `ralph org start --help` を出し、Long と `--plan`、`--org-id`、`--cwd` の行が上の変更どおりに出ること、折り返しが崩れていないことを見た |
| help の文字列を固定するテスト | `internal/cli` に、この 3 つの文字列を固定するテストはない。`TestOrgSpawn_DeprecatedDriverFlagAlias_HiddenFromHelp` は `spawn --help` の `--leader-driver` の有無だけを見る |

## Found but left

- verify V-3(`doctor.go` の `(solo execution unaffected)`)は、plan の範囲の外にある文言なので直さず、tech-debt の 1 行((h))に送った
- self-review の L5 (a)〜(e)、N1、N2 は、挙動を変えない LOW なので直さず、tech-debt の 1 行に送った。N3(進捗に 8aae7ce2 の記録)は a49d38a1 で直っている
- 実機の記録のその他の記述(Run 1 の「pane の ralph が古い版」など)は、そのまま。再実行もしていない
- 計画の進捗の「実装中に見つけて送るもの」の (2) は「Stop hook」と書いているが、実際に commit するのは `SessionEnd` の hook(`.claude/hooks/session_end_summary.sh`、`ralph-dispatch.sh SessionEnd` から呼ばれる)だった。台帳の行はこちらの名前で書き、進捗の行にも 1 文足した。(2) の元の文は承認済みの記録ではない(進捗の中)が、書き換えずに残した
- README の `ralph org` の行に 20 文字の上限は足していない(上の表)。足すなら README の「One org handles one feature」の段落
- help の「20」は文字列の直書きで、定数 `maxFeatureOrgIDLen` と結びつくテストがない。tech-debt の (i) に書いた。今回はテストを足していない(コードの変更を help の文字列に留めるため)
- この sync-docs の commit は `internal/cli/org.go` を含む。`.claude/rules/ralph/post-implementation-pipeline.md` は、コードの修正のあとに全 pipeline を回し直すと書き、例外を「この task の記録だけを変える修正」に限っている。変えたのは help の文字列 3 つだけで、`go test ./internal/cli/ -run 'OrgStart|HiddenFromHelp|OrgStatus'` は通ったが、回し直しが要るかどうかは lead が決める(cycle は 1、上限 2)

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。編集の後に実行 |
| `./scripts/check-skill-sync.sh` | 13 skill(s) in lock-step |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-09-org-feature-worktree.md` | `56e435bfa976`(編集の前後で同じ。`- Approved:` の行と一致) |
| `go build -o <scratchpad>/ralph ./cmd/ralph` と `ralph org start --help` | 通った。help の出力を目で確かめた |
| `go test ./internal/cli/ -count=1 -run 'OrgStart\|HiddenFromHelp\|OrgStatus'` | ok(16.4 秒)。`internal/cli` の全体は流していない(help の文字列を固定するテストがないため) |
| `gofmt -l internal/cli/org.go` | 出力なし |
| tech-debt の表の形(scratchpad の `tdcheck.py`) | 変えた 3 行と新しい 6 行は、すべて 7 つの区切り(5 列)、列ごとのバッククォートは偶数、取り消し線の `~~` は偶数個、新しい `file:line` の参照なし。`HEAD` との行ごとの比較で、変わった既存の行は 182、184、185 だけ |
| `./scripts/run-verify.sh` | 下の「run-verify の結果」 |

## run-verify の結果

`./scripts/run-verify.sh`(mode all、scope full)を、この report、insights の event、tech-debt、spec、quality-gates、`internal/cli/org.go`、evidence、plan の編集がすべて終わった状態で流した。開始は 2026-10-09T12:34:04Z、終了は rc 0 で「All verifiers passed.」。開始後にリポジトリのファイルは変えていない(log より新しいファイルは 0 件)。

- 静的: shellcheck、hook の `sh -n`、`jq -e` の settings 2 つ、Codex の hook の 3 つのガード、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README の plan の参照(`OK`)
- shell のテスト: `tests/test-*.sh` を全部。`FAIL: <n>` の集計行はすべて `FAIL: 0`(0 以外は 0 件)、`PASS` の行は 3,823
- Go(scope は full、pack は golang): `gofmt: ok`、`0 issues.`(golangci-lint)、`go test ./...` はすべて ok(`internal/cli` 131.3 秒、`internal/org` 32.8 秒、ほかはキャッシュ)
- 分岐の secret scan: `765da6bd..09ad7d5a` は clean。この sync-docs の commit は、実行の時点でまだ range に入っていない
- evidence: `docs/evidence/verify-2026-10-09-123404.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)

## Not verified

- この sync-docs の commit を含む range の secret scan は、commit の前には流せない(分岐の scan は commit 済みの範囲だけを読む)。commit のあとに `./scripts/secret-scan-branch.sh --strict` を流す(その結果は commit の外、lead への報告に書く)
- `ralph org start --plan` を実機(herdr と Claude Code)で打ち直してはいない。実機の記録は af138af6 のバイナリで、8aae7ce2 と 2c1bb13b のあとの再実行はない(test report の T-7)。今回足した hint の行は、コードが出す文を写したもの
- tech-debt の新しい行の事実のうち、実機の Run 1 と Run 3 の観測(pane の `ralph` が v5.1.0 に解決されたこと、2 つの `wip:` commit)は、evidence の記述による。`session_end_summary.sh` の動作(`git add -A`、main と master 以外のブランチ)は、script を読んだ。同時 start の結果は、誰も測っていないと tech-debt に書いた
- `ralph-worktree.sh cleanup` が未追跡のファイルで止まることは、script を読んで(`git status --porcelain` は未追跡も数える)確かめた。昇格した leader が実際にその状況になる実行はしていない
- spec の FR-4 の追記のうち、「不完全な結びつき」の中身(キーの欠けと重複)は self-review の L4 の記述とコードの doc による。3 通りの壊れ方の test(`TestOrgStatus_IncompleteFeatureBinding`)は流していない

## Cycle 2

- Date: 2026-10-10
- Plan: `docs/plans/active/2026-10-09-org-feature-worktree.md`(承認済み、digest `56e435bfa976`。編集の前後で同じ値。変えたのは `## Progress checklist` の中だけ)
- Pipeline cycle: 2(上限 2、最後の回)。差分は cross-review の triage `183cb190` の後ろから HEAD `37178a7a` まで(d49bbc34、07d38e6d、c2a1f8c4、9c1d447f と、report と実機の記録の commit)
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-09-org-feature-worktree.md`(`c322e7b3`、cycle 2。Merge 可、LOW 6 件 C2-1〜C2-6)、
  `docs/reports/verify-2026-10-09-org-feature-worktree.md`(`9a2dc5ef`、cycle 2。pass、V2-1〜V2-3)、
  `docs/reports/test-2026-10-09-org-feature-worktree.md`(`43819223`、cycle 2。pass、T2-1〜T2-6)、
  実機の記録 `docs/evidence/org-feature-worktree-live-2026-10-09.md`(`37178a7a` で Run 4 を追加)、
  `docs/reports/cross-review-triage-org-feature-worktree.md`(cycle 1。ACTION_REQUIRED 2 件)

### Summary

古くなっていたのは、配る `templates/base/ralph.toml` の `max_orgs` のコメント 1 か所(V2-1、C2-1 の (b))、leader の雛形と `/org` skill と `featureLeaderTask` の doc の言い回し(C2-2)、仕様の FR-4 の「4 段目で決めたこと」(V2-3)、tech-debt の 6 行(V2-2、C2-6)、計画の進捗の並びだった。cycle 2 で直さなかった LOW と test report の残りの穴は、新しい行を作らず既存の行 188・192・193 に足した。

この commit は Go のファイルを 2 つ含む。`internal/org/prompts/leader.md`(`go:embed` される雛形)と `internal/org/feature.go`(doc comment だけ)。関数の挙動は変えていない。cycle の上限(2)に当たっているので、回し直すかどうか(上限を上げるか)は lead の判断に任せる(この節の「Found but left」の末尾)。

差分の大きさ(`git diff origin/main...HEAD --stat`、`37178a7a` 時点、この sync-docs の commit を含まない): 38 files changed, 7848 insertions(+), 441 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

### Changes made

| File | Change |
|------|--------|
| `templates/base/ralph.toml`(V2-1、C2-1 の (b)) | `max_orgs` のコメントを、`--help`(`orgWideLimitsHelp`)と skill の「全 org の上限と予約」と同じ条件に直した。`--config` がなく、state dir が main worktree の `.harness/state/org` のとき(既定の解決でも、`--state-dir` や `RALPH_ORG_STATE_DIR` で指しても)main の `ralph.toml` から読む。それ以外(`--config`、別の state dir、git の外)は `--config` のファイルか `./ralph.toml`。値は変えていない。ルートに `ralph.toml` はなく、同じ文を持つのはこのファイルだけ(`git grep` で確認) |
| `internal/org/prompts/leader.md`(C2-2 の (a)(b)(c)) | 「機能ごとの org」の手順 4 の `ralph org report` と手順 9 の `ralph org disband` に `--state-dir <台帳>` を足した(手順 9 の「ミッション」の 8 との関係も「`--state-dir` を足した形」と書いた)。導入の段落に、手順の `--state-dir <台帳>` はタスクの `- 台帳:` の行の値(引用符も含む)だと 1 文足した。「次の順で進めます。」を、手順の一覧の直前の独立した段落にした。pane の環境の言い切り「この pane の環境は start を打った環境と違う」を、「ralph は start に渡した `--state-dir` や `RALPH_ORG_STATE_DIR` をこの pane に渡さず、pane の環境は start を打った環境と同じとは限らない」に弱めた。`internal/org` に pane へ環境を渡す箇所がないことは `git grep RALPH_ORG_STATE_DIR` で確かめた(コードが渡さないことまで。pane に実際に届くかは未確認で、台帳の行 193 の (h)) |
| `.claude/skills/org/SKILL.md` と写し 3 面 | 台帳の段落の「leader には届かない」を、同じ弱めの文に直した。「org の終わり方」の導入に「下の `ralph org` のコマンドにも、タスクの `- 台帳:` の行の `--state-dir` を付ける」を足した。`./scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り直し、`templates/base/.claude/skills/org/SKILL.md` と `templates/base/.agents/skills/org/SKILL.md` に `cp`。4 面は `cmp` で同一 |
| `internal/org/feature.go`(C2-2 の (c)、コメントだけ) | `featureLeaderTask` の doc の「not the one start ran in」を「which is not necessarily the one start ran in」にし、「neither … reaches the leader's own commands」を「ralph passes neither … to the pane, so the leader's own commands cannot count on either」に直した。コードは変えていない |
| `docs/specs/2026-10-07-org-multi-org-director.md`(V2-3) | FR-4 の「4 段目で決めたこと」に、cross-review のあとの 3 点を足した。(a) 分割計画のコードフェンスの中の行は本文として残し、閉じていないフェンスは拒否、digest が読まない行の拒否はフェンスの中でも続ける。(b) leader のタスクの `- 台帳:` の行と、leader が `ralph org` のコマンドすべてに `--state-dir` を付けること。(c) main の台帳を指す `--state-dir` / `RALPH_ORG_STATE_DIR` でも、全 org の上限は main の `ralph.toml` から読む。既存の文は変えていない。チェックボックスは触っていない |
| `docs/tech-debt/README.md`(行 146) | C2-6 の (b)。「`--config` or `--state-dir` avoids it」が main の台帳を指す `--state-dir` では偽になったので、条件を書き分けた(`--config` と別の台帳を指す `--state-dir` は避けられる。main の台帳を指すものは 07d38e6d で main の `ralph.toml` も読むので避けられず、leader の `spawn` も止まる。コードから読んだだけで未実行と書いた)。Trigger に、07d38e6d が `withMainWorktreeOrgLimits` を変えてトリガーの一部を満たしたが、行は open のままと足した。Related に self-review を足した |
| `docs/tech-debt/README.md`(行 184 の (d)) | C2-6 の (c)。org-limits-reserve の C2-4 は、9c1d447f の `orgWideLimitsHelp` の書き直しと、この回の `templates/base/ralph.toml` のコメントで閉じた。提案した「neither … is set」の文は 07d38e6d で成り立たなくなったので、その文は使わず別の規則で直した旨を書いた。Debt の (d)、Impact の (d)、Trigger の (d) を取り消し線にした。Why deferred に (a) と (d) がこの branch の pipeline の中で直った旨を足した |
| `docs/tech-debt/README.md`(行 185 の (c)) | C2-6 の (d)。`orgWideLimitsHelp` の分は `TestOrgSpawnAndStartHelp_OrgWideLimitsSource`(9c1d447f)が spawn と start の `--help` の 2 文を固定したので取り消し線にし、`status` の `Long` と `--config` の説明が残ると書いた。Impact と Trigger も同じ範囲に直した |
| `docs/tech-debt/README.md`(行 188) | C2-6 の (a)(g)。Debt の「leader に state dir を渡すものがない」は d49bbc34 で `start --plan` については偽になったので、閉じた旨と実機の Run 4 の確認を書き、残りを 2 つにした(版の食い違いそのもの、`ralph org start <task>`(`--plan` なし)で `--state-dir` / `RALPH_ORG_STATE_DIR` を付けた start の leader に台帳が渡らないこと)。後者は cross-review #1 と同じ穴で、plan の範囲の外なので stage 4 では直していない。Impact に、旧版の pane の `ralph` に今のイベントを読ませることは未実行と足した。Why deferred と Trigger(Options)を、選んだ案(task に台帳の行)と残りの案に直した。Related に Run 4、triage の #1、self-review を足した |
| `docs/tech-debt/README.md`(行 192) | cycle 2 で直さなかった LOW を (j)〜(l) に足した。(j) C2-3: N1 の 3 つ目の形(台帳の段落の固定が、改行を消した文字列で折り方に依存する)。(f) のトリガーを d49bbc34 と今回の sync-docs の編集が満たしたが直していないこと、今回は固定が頼る 2 行を触っていないことを書いた。(k) C2-4: `split.go` のフェンスの説明が挙動より広い。(l) C2-5: `shellQuote` が `shellQuoteIfNeeded` の写しなこと、`mustAbs` の呼び出し。Impact、Why deferred、Trigger、Related も足した。C2-2 は今回直したので足していない |
| `docs/tech-debt/README.md`(行 193) | test report の残りの穴を足した。(g) T-7 は、Run 4 が 9a2dc5ef のバイナリで d49bbc34・07d38e6d・9c1d447f のあとに非既定の台帳で打ち直したので、「再実行していない」の部分を取り消し線にした(T2-1 と C2-6 の (f) がこれで閉じる)。(h) T2-2 pane の環境に start の `RALPH_ORG_STATE_DIR` が届くかの実地確認、(i) T2-3 `start <task>` の台帳(行 188 を指す)、(j) T2-4 まだない台帳を symlink の別名で渡す場合、(k) T2-5 閉じていないフェンスと digest が読まない行の、どちらのエラーが先か、(l) T2-6 probe を残していないこと。Impact、Why deferred、Trigger、Related も足した |
| `docs/plans/active/2026-10-09-org-feature-worktree.md` | `## Progress checklist` の中だけを変えた(`./scripts/plan-visual.sh digest` は前後とも `56e435bfa976`)。2026-10-10 の cross-review の行を、2026-10-09 の 2 行(実装中に見つけて送るもの、sync-docs の cycle 1)のあとに動かし、9c1d447f の記述を足し、pane の環境の言い切りを同じ弱めに直した。Run 4(37178a7a)の行と、この sync-docs の行を足した。チェックボックスは触っていない(「PR created」は `/pr` が付ける) |
| `docs/reports/sync-docs-2026-10-09-org-feature-worktree.md` | この cycle 2 の節 |
| `docs/insights/events/2026-10-09-org-feature-worktree.jsonl` | `sync_docs` の event を 1 行追記した(`--cycle 2`、verdict pass、findings 0) |

### Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `/org` skill の 4 面 | `cmp` で 4 面とも同一。`./scripts/check-skill-sync.sh` は 13 skill が lock-step、`./scripts/check-sync.sh` は DRIFTED 0・TEMPLATE_ONLY 11・KNOWN_DIFF 5 |
| `internal/org/prompts/leader.md` と `TestRenderRolePrompt_Leader_FeatureOrgProcedure` | `go test ./internal/org/ -run 'Prompt\|Role' -count=1` は ok。テストは「の行にある」と「`--state-dir`」の間の折り方、手順 4 の `ralph org report --org-id org-a` と手順 9 の `ralph org disband --org-id org-a` の文字列、手順 9 の項目に「最後のコマンド」があること、を見る。直した文はそれらを残した(手順 9 は、項目の最初の行に `ralph org disband …` が来て、次の行に「最後のコマンド」が来る形にした) |
| `leader.md` の「ミッション」の 6〜8 | 手順の `ralph org report` / `disband` に `--state-dir` を足していない。ミッションはすべての leader(昇格した leader、`start <task>` の leader を含む)に使われ、台帳の行がない。`start <task>` の台帳は台帳の行 188 に書いた |
| `README.md`、`AGENTS.md`、`docs/recipes/`、`docs/quality/quality-gates.md`、`docs/architecture/repo-map.md` | 変更なし。README は上限の読み元に触れず、quality-gates の「Split plan approval」の行(digest の検査、先読み、worktree の使い回し)はフェンスと台帳の行に触れていないが、行の記述は今も成り立つ。`git grep` で「届かない」「neither --state-dir」の残りを探し、skill の `142 行` の別の文脈(新しい既定が届かない)以外は、レポートと台帳の履歴の引用だけだった |
| `ralph org spawn --help` / `start --help` の全体の上限の段落 | 9c1d447f の版が、skill と `templates/base/ralph.toml` の文と同じ条件になった。このコメントの文言を固定するテストはない(help は `TestOrgSpawnAndStartHelp_OrgWideLimitsSource`) |

### Found but left

- 実機の記録の「結論」の「残すもの: (1) start を打った ralph と pane の中の ralph の版が違うと台帳が分かれうる」は、Run 4 のあとも版の食い違いそのものは残るので、そのままにした(`start --plan` の場所の件は結論の別の箇所と Run 4 が書いている)
- cycle 2 の LOW のうち C2-3〜C2-5 は、cap(2)に当たっているので直さず、台帳の行 192 の (j)〜(l) に送った。test report の T2-2・T2-3・T2-4・T2-5・T2-6 は行 193 の (h)〜(l) と行 188 に送った
- `leader.md` の `--state-dir <台帳>` は、台帳の行の値を引用符つきで写す約束になっている。Run 4 が実機で見たのは、leader が `/tmp/rs6/ledger`(空白も `'` もない短いパス)を `--state-dir` に付けたことまで。空白や `'` を含むパスの引用は `TestFeatureLeaderTask_StateDirIsOneShellWord` が `sh` で確かめるだけで、実機では未確認
- この commit は Go のファイル(`internal/org/prompts/leader.md` の文面と `internal/org/feature.go` のコメント)を含む。`.claude/rules/ralph/post-implementation-pipeline.md` は、コードの修正のあとに全 pipeline を回し直すと書き、例外を「この task の記録だけを変える修正」に限っている。cycle は 2(上限)なので、回し直すなら上限を上げる必要がある。変えたのは文面とコメントだけで、`go test ./internal/org/ -count=1` は ok(下の「Checks run」)。回し直しが要るかどうかは lead が決める

### Checks run

| Command | Result |
|---------|--------|
| `go test ./internal/org/ -run 'Prompt\|Role' -count=1` | ok(`leader.md` の編集の途中で 1 回 red だった。テストは手順 9 の項目を `ralph org disband --org-id org-a` を含む行から読むので、「最後のコマンド」をその行より前に置いた最初の版は落ちた。「最後のコマンド」を次の行に置いて green) |
| `go test ./internal/org/ -count=1` | ok(18.5 秒) |
| `gofmt -l internal/org/feature.go`、`go vet ./internal/org/` | 出力なし、ok |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。編集の後に実行 |
| `./scripts/check-skill-sync.sh` | 13 skill(s) in lock-step |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-09-org-feature-worktree.md` | `56e435bfa976`(編集の前後で同じ。`- Approved:` の行と一致) |
| tech-debt の表の形(scratchpad の `anchors.py` と `tdcheck.py`) | 編集の前に、計画した 31 個の anchor がすべて 1 回だけ出ることを確かめた。変えた行は 146、184、185、188、192、193 だけ。6 行とも 7 つの区切り(5 列)で変わらず、列ごとのバッククォートは偶数、取り消し線の `~~` は偶数個、新しい `file:line` の参照なし(`HEAD` との比較) |
| `./scripts/run-verify.sh` | 下の「run-verify の結果(cycle 2)」 |

### Not verified

- `ralph org start --plan` の leader が、変更後の `leader.md`(手順 4・9 の `--state-dir <台帳>`)に従って打つことは、実機で打ち直していない。Run 4 は 9a2dc5ef のバイナリで、そのあとの `leader.md` の変更は文面だけ(この sync-docs)
- pane の環境に start の `RALPH_ORG_STATE_DIR` が届くかどうかは、確かめていない。書いたのは「ralph は渡さない」(コードで確認)と「同じとは限らない」だけ(台帳の行 193 の (h))
- この sync-docs の commit を含む range の secret scan は、commit の前には流せない。commit のあとに `./scripts/secret-scan-branch.sh --strict` を流し、結果は lead への報告に書く
- tech-debt の行 146 の「main の `ralph.toml` が壊れていると leader の `spawn` も止まる」は、`withMainWorktreeOrgLimits` を読んだ推論で、実行していない(行にもそう書いた)

### run-verify の結果(cycle 2)

`./scripts/run-verify.sh`(mode all、scope full)を、`leader.md`、`feature.go`、skill の 4 面、`templates/base/ralph.toml`、仕様、tech-debt、計画、insights の event の編集がすべて終わった状態で流した。開始は 2026-10-09T17:32:39Z(JST 02:32)、終了は 17:41:59Z で、rc 0、最後の行は「All verifiers passed.」。この節(結果の記録)は、実行のあとにこの report へ足した。流した時点の report には、この節だけがない。

- 静的: shellcheck、hook の `sh -n`、`jq -e` の settings、Codex の hook のガード、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README の plan の参照
- shell のテスト: `FAIL: <n>` の集計行 24 本はすべて `FAIL: 0`、`PASS` の行は 3,823(cycle 1 の run-verify と同じ数)
- Go(scope は full、pack は golang): `gofmt: ok`、`0 issues.`(golangci-lint)、`go test ./...` は 8 パッケージすべて ok(`internal/cli` 83.9 秒、`internal/org` 18.9 秒、`internal/config`、`internal/scaffold` を流し直し、残りはキャッシュ)
- 分岐の secret scan: `765da6bd..37178a7a` は clean。この sync-docs の commit は、実行の時点でまだ range に入っていない
- evidence: `docs/evidence/verify-2026-10-09-173239.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)
