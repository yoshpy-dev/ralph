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
