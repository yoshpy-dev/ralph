# Test report: org-feature-worktree

- Date: 2026-10-09(JST。実行は UTC の 11:34〜12:14)
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md(承認済み、digest 56e435bfa976)
- Tester: tester subagent (Claude Opus 5.5)、パイプライン 1 回目(`cycle-count.json` は 1、上限 2)。2 回目(cycle 2、上限の回)は 2026-10-10 に HEAD 9a2dc5ef で行い、末尾の「Cycle 2」の節に書いた。下の Scope から Test gaps までは cycle 1 の記録
- Scope: HEAD 421719f7(base origin/main 765da6bd)。behavioral test だけを実行した(静的解析は /verify の担当)。重点は 3 つ。self-review が実行での確認を求めた 2 点(M1 の 20 文字と 21 文字の境界、ensure の失敗の案内を本物のスクリプトの文から選ぶこと)、計画の Test plan の edge case にテストがあるか、3 段目の予約と補償と `start <task>` が壊れていないか
- Evidence: `docs/evidence/test-2026-10-09-org-feature-worktree.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-09-113451.log`
- 足したテスト: `internal/org/feature_test.go` に 2 本、`internal/org/split_test.go` に 1 本(テストのファイルだけ、+122 行)。理由は「依頼された 2 点」と「Edge cases」の節。cycle 2 では `TestLoadSplitPlan_CodeFences` に 1 ケースを足した(「Cycle 2」の節の Mutation)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(HEAD、変更言語の scope) | shell 40 本(PASS の行 3,797)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 22 分 12 秒、rc 0 |
| `go test ./internal/org/ ./internal/cli/ -count=1 -coverprofile`(HEAD) | 2 パッケージ | `internal/org`(59.3 s) | `internal/cli` の 1 本(Failure analysis) | 0 | 3 分 54 秒、rc 1 |
| 落ちた 1 本を単独で `-count=5`(HEAD) | 5 | 5 | 0 | 0 | 8.7 s |
| `go test ./internal/org/ ./internal/cli/ -count=1 -coverprofile`(テストを足したあと) | 2 パッケージ | 2(`internal/org` 30.0 s、`internal/cli` 159.2 s) | 0 | 0 | 2 分 41 秒、rc 0 |
| 新しいテストと 3 段目の回帰の絞り込み(`internal/org`、下の注の正規表現) | top-level 92、subtest 450 | すべて | 0 | 0 | 4.6 s |
| 同じく `internal/cli` | top-level 27、subtest 30 | すべて | 0 | 0 | 14.3 s |
| `go test -race -count=1 ./internal/org/...` | 3 パッケージ | 3 | 0 | 0 | 32.2 s |
| `TestStartFeature`・`TestSplitPlanCheckApproved_CRLF`・`TestSpawnPrecheckErr_MatchesSpawn` を `-count=10` | top-level 180(18 本 × 10)、subtest 550 | すべて | 0 | 0 | 33.9 s |
| 残りの 6 パッケージを `-count=1`(config、insights、scaffold、upgrade、org/driver、org/protocol) | 6 パッケージ | 6 | 0 | 0 | 各 2〜4 s |
| mutation(`go test -overlay`、足したテストを入れた `internal/org` 全体) | 7 件 | red 7 件 | 生き残り 0 | - | 1 件 10〜45 s |

- shell の件数は suite ごとに PASS・FAIL・SKIP の行を数えた(`PASS: 29` のような集計行は除いた)。40 本すべてが `OK` で終わり、FAIL の行は 0。Skipped の 1 件は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)
- `run-test.sh` の中の `go test ./...` は `internal/org` だけを実行し(54.1 s)、ほかはキャッシュの結果だった。そのため 2 行目と 4 行目で `internal/org` と `internal/cli` を `-count=1` で流し直した。go は 1.26.0
- 絞り込みの正規表現は、`internal/org` が `Split|PlanDigest|StartFeature|SpawnPrecheckErr_MatchesSpawn|Feature|RelativeCwd|EnsureFailure|RenderRolePrompt_Leader_FeatureOrgProcedure|Reservation|Reserve|CloseDeferredSelf|ScopeReserved`、`internal/cli` が `OrgStartPlan|OrgStatus_Feature|OrgStatus_Incomplete|OrgStart|Reserve|Feature`。この差分が足したテストの関数(`internal/org` の 46 本と足した 3 本、`internal/cli/org_feature_test.go` の 7 本)が、どれもこの実行で `--- PASS` になったことを名前で照合した
- `run-test.sh` の実行中は、足したテストを worktree から外して scratchpad に置いた(2 行目までは HEAD のテストだけ)。足したあとの `internal/cli` は、足したテストが `internal/org` のファイルだけなので HEAD と同じテストである
- `TestStartFeature_RealWorktreeScript_CheckoutsMoved` の「記録のないディレクトリ」の段は、4 行目の coverage の実行のあとに足した。足したあと、`internal/org` 全体(`-count=1`、20.0 s)、そのテストの `-count=3`、5〜8 行目を流し直した
- 実行中、ほかのセッションの負荷で load average が 5〜15 分の平均で 90〜110 あった。`run-test.sh` の 22 分はそのためと見ている

## Coverage

- Statement: `internal/org` 93.8%(HEAD と、テストを足したあとで同じ)、`internal/cli` 85.5%(テストを足したあとの通った実行。HEAD の実行は 85.7% だが、落ちた doctor のテストの失敗の枝を通った分が入っている)。ほかの 6 パッケージは測り直していない
- 足したテストで `internal/org` の関数ごとの値は 1 つも変わらなかった(`go tool cover -func` の差分が空)。足したテストは、すでに通っている行を本物のスクリプトと git で通し直すもので、行の coverage は増やさない
- Function(新しいファイルと、差分が変えた関数):

| 関数 | 位置 | 値 | 通らない文 |
| --- | --- | --- | --- |
| `split.go` の 27 関数 | | 24 関数が 100.0% | `ResolveSplitPlanPath` 90.5%、`LoadSplitPlan` 92.9%(どちらも `filepath.Abs` の失敗)、`pathErrorCause` 75.0%(`*fs.PathError` でないエラー) |
| `StartFeature` | `feature.go:161` | 92.1% | leader の入力の `checkSpawnInput` が失敗する枝と台帳の読み込みの失敗(`:180`、`:184`)。分割計画の検査を通った入力では起きない防御の枝 |
| `ensureFailureErr`、`readStartFeature`、`checkFeatureWorktreeReuse`、`featureLeaderTask`、`startFeatureLeaderParams` | `feature.go` | 100.0% | なし |
| `scriptFeatureWorktrees` の `Lookup` / `Ensure` / `CurrentBranch` | `feature.go:426`、`:452`、`:467` | 78.6% / 87.5% / 75.0% | `state-path` が何も出さない、記録が読めない・JSON でない、`ensure` が何も出さない、`git branch --show-current` の失敗 |
| `reserve.go` の差分の関数(`Complete`、`sameFeature`、`validateFeatureBinding`、`scopeReservedDetails`、`reservationFromEvent`、`featureBindingFromTokens`、`ActiveFeature`、`reservationDecision`) | | 100.0% | なし |
| `checkSpawnInput`、`spawnPrecheckErr`、`spawnCapacityErr`、`idempotentRespawnDecision` | `spawn.go` | 100.0% | なし |
| `idempotentRespawn` | `spawn.go:1232` | 85.7% | 予約の記録の追記の失敗 |
| `reserveAgain`、`releasedReservation` | `verbs.go:1327`、`:1355` | 100.0% | なし |
| `checkOrgStartPlanInput`、`newOrgStartCmd`、`printStatusTable`、`printStatusJSON` | `internal/cli/org.go` | 100.0% | なし |
| `runOrgStartPlan` | `internal/cli/org.go:696` | 87.5% | `newOrgSpawnRuntimeAt` の失敗(`:699`)と `resolveModelOrWarn` の失敗(`:707`) |
| `FeatureRepoRoot` | `statedir.go:103` | `internal/org` のテストでは 0.0%、`internal/cli` の `OrgStartPlan` のテストから `-coverpkg=./internal/org/` で測ると 85.7% | CLI からだけ呼ばれる |

- Branch: Go の標準ツールには branch coverage がないので測っていない

## 依頼された 2 点の確認

### (1) M1 の境界(20 文字は implementer の spawn まで通り、21 文字は副作用の前に拒否される)

| 経路 | 20 文字 | 21 文字 |
| --- | --- | --- |
| `--org-id` | 既存の `TestStartFeature_OrgIDAtTheLengthLimit`: `StartFeature` で leader が立ち、同じ org で `o.Spawn` が seat id `implementer` を `SpawnOutcomeSpawned` で通す(`Spawn` は `checkSpawnInput` を呼ぶ)。`maxFeatureOrgIDLen` が 20 であることも固定する | 既存の `TestStartFeature_RefusedBeforeAnyRecord/an_--org-id_of_21_characters`: エラー文の全文、worktree の lookup も ensure も呼ばれないこと(`wantCalls` が空)、台帳・receipt・herdr・agmsg の数が 0 のまま |
| slug(`--org-id` なしの既定の org_id) | 足した `TestStartFeature_SlugAtTheLengthLimit/20_characters`: 20 文字の slug の分割計画で、org_id が slug、worktree が `.claude/worktrees/org-<slug>`、ブランチが `feat/<slug>` になり、leader が `implementer` を spawn できる | 既存の `TestLoadSplitPlan_Rejects/slug_of_21_characters` は読み込みの段のエラー文だけを見ていた。足した `TestStartFeature_SlugAtTheLengthLimit/21_characters` が `StartFeature` の段で、分割計画を読んだ結果を返さないこと、worktree の呼び出しがないこと、何も記録しないことを見る |

slug の側に `StartFeature` から implementer の spawn までのテストがなかったので足した。どちらも fake の herdr なので、確かめたのは ralph 側の判定(`checkSpawnInput` の `len(org_id)+1+len(seat_id) > 32`)である。herdr 本体の 32 文字の上限に 20 文字の org_id の `implementer` を当てた実行はない(Test gaps の T-1)。

### (2) ensure の失敗の案内を、本物のスクリプトの文から選ぶこと

| ensure の失敗 | 本物のスクリプトでの実行 | 案内 |
| --- | --- | --- |
| main が dirty | 既存の `TestStartFeature_RealWorktreeScript`(org 層)と `TestOrgStartPlan_RealWorktreeScript/a_dirty_main_checkout`(CLI)。未追跡のファイルを置いて別の機能を start する | 「make the main worktree <root> a clean checkout of the default branch and run start again」 |
| 同じ名前のブランチがすでにある | 既存の `TestStartFeature_RealWorktreeScript`。`git branch feat/auth-docs` を手で作る | ブランチの改名か削除の案内。clean checkout の案内が付かないことも見る |
| main が default branch でない | 足した `TestStartFeature_RealWorktreeScript_CheckoutsMoved`。main のチェックアウトで `git checkout -b side` | スクリプトの文「must start from clean default branch 'main' (current: side)」に clean checkout の案内。その機能の worktree はできず、何も記録しない |
| worktree のパスに記録のないディレクトリがある | 足した同じテスト。`.claude/worktrees/org-auth-docs` を手で作る | 「remove <path> if it is no longer needed (git worktree remove for a worktree), or use another --org-id」 |
| state の衝突、jq がない、default branch がない、`.codex/config.toml` の書き換え | fake の ensure(`TestStartFeature_EnsureFailureRefused` の 8 件)と、照合する 6 つの文がスクリプトにあることの検査(`TestEnsureFailureMessages_InWorktreeScript`)だけ | Test gaps の T-2 |

verify の Coverage gaps にあった「AC9 の default branch でない場合は fake だけ」は、足したテストで本物のスクリプトを通した。

## Mutation(足したテストが見分けるか)

`internal/org/feature.go` か `split.go` の写しを scratchpad で 1 か所だけ書き換え、足したテストを入れた overlay で `go test -overlay=<json> -count=1 -vet=off ./internal/org/` を流した(`-run` で絞っていない)。worktree のファイルは書き換えていない。

| # | Mutation | Red になったテスト |
| --- | --- | --- |
| K1 | `ensureFailureErr` が default branch でない文を見ない(dirty の文だけで clean checkout の案内を付ける) | `EnsureFailureRefused/main_is_on_another_branch`、足した `RealWorktreeScript_CheckoutsMoved` |
| K2 | `CurrentBranch` が `-C <path>` を落とす(テストのプロセスの cwd で git を打つ) | 既存の `RealWorktreeScript`、足した `RealWorktreeScript_CheckoutsMoved` |
| K3 | 再利用の検査がチェックアウトしているブランチを比べない | `WorktreeRecordMismatchRefused` の 2 件(fake)、足した `RealWorktreeScript_CheckoutsMoved` |
| K4 | `newSplitFeatureDraft` の slug の長さの検査を外す | `LoadSplitPlan_Rejects/slug_of_21_characters`、足した `SlugAtTheLengthLimit/21_characters`(org_id の検査で拒否はされるが、エラー文が slug の文でなくなる) |
| K5 | `readStartFeature` の org_id の長さの検査を外す | `RefusedBeforeAnyRecord/an_--org-id_of_21_characters` だけ。足した slug のテストは通る(21 文字の slug は先に読み込みで拒否されるので、想定どおり) |
| K6 | 上限を `implementer` でなく `reviewer` から計算する(23 文字) | 既存の `OrgIDAtTheLengthLimit`(定数の固定)、`RefusedBeforeAnyRecord`、`LoadSplitPlan_Rejects`、足した `SlugAtTheLengthLimit` の 2 件(20 の側は、23 文字の slug の leader が `implementer` を spawn できず落ちる) |
| K7 | `ensureFailureErr` の記録のないディレクトリの案内を外す | `EnsureFailureRefused/a_directory_in_the_way`(fake)、足した `RealWorktreeScript_CheckoutsMoved` |

- 7 件すべてが red になり、生き残りはない。どれも既存のテストでも red になる。足したテストだけが見分ける mutation はなく、足したテストの役目は、既存の fake で見ていた判定を本物のスクリプトと git で通すことと、slug の側の境界を `StartFeature` の段で固定すること
- K2 と K4 の実行では、変えていない `TestRunWatcher_TimeoutIndependentOfSmallInterval` も落ちた。`run-test.sh` の shell の suite と並べて流した回で、既知の負荷による flake(Failure analysis)

## Edge cases(計画の Test plan の 10 件)

| Edge case | 状態 | テスト |
| --- | --- | --- |
| CRLF の分割計画 | テストあり。承認の通り道は足した | `TestLoadSplitPlan_CRLF`(読み込み、値に `\r` が残らない)、`TestPlanDigest_MatchesScript` の `crlf` 2 件、`TestLoadSplitPlan_Rejects/progress_checklist_with_CRLF`。足した `TestSplitPlanCheckApproved_CRLF` は、本物の `plan-visual.sh digest` の値を `- Approved:` に書いた CRLF の分割計画が `CheckApproved` を通ることを見る |
| `- [X]` | テストあり | `Rejects/indented_capital_checked_box`、`RejectsEditsTheDigestSkips/box_ticked_with_a_capital`、`MatchesScript/checked_boxes` |
| `## Progress checklist` が最後の節 | テストあり | `Rejects/progress_checklist_at_the_end`、`RejectsEditsTheDigestSkips/progress_checklist_appended`、`MatchesScript/progress_checklist_at_the_end` |
| 機能の本文の中の `- Branch:` と 2 つ目の `- Status:` | テストあり | `Rejects` の `branch_in_a_feature`・`status_in_a_feature`・`second_status_in_the_header`、`RejectsEditsTheDigestSkips` の 2 件、`StartFeature_RefusedBeforeAnyRecord` の承認のあとに足した `- Branch:` |
| `Depends on: none` | テストあり | `ParsesFeatures`(auth-core の依存は nil)、`Rejects/depends_on_none_and_a_slug`、`TestFeatureLeaderTask`(依存なしは「なし」) |
| slug と同じ名前のブランチがすでにある | テストあり(本物のスクリプト) | `TestStartFeature_RealWorktreeScript` |
| disband のあとの、別の分割計画の同じ slug | テストあり(fake の worktree) | `TestStartFeature_AfterDisband/another_split_plan_with_the_same_slug`。本物のスクリプトの記録から `canonical_ref` を読み戻すことは `TestStartFeature_RealWorktreeScript` が見る |
| worktree の中で別のブランチをチェックアウトした | テストあり。本物の git は足した | `WorktreeRecordMismatchRefused` の「another branch checked out」「a detached HEAD」(fake)。足した `RealWorktreeScript_CheckoutsMoved` は、feature worktree で `git checkout -b moved` すると「which has branch "moved" checked out」で拒否され、元のブランチに戻すと再び使い回すことを見る |
| `--org-id` で slug と違う org_id | テストあり | `StartFeature_OrgIDAndTypeDefault`(`docs-x`)、`OtherBindingRefusedWithoutWorktree`(`--org-id` で別の機能を同じ org へ) |
| 予約の Details に注記と結びつきの両方がある | テストあり | `TestScopeReservedDetails_FeatureRoundTrip`、`TestReservationFromEvent_DamagedBindingMatchesNothing/reading_stops_at_the_note` |

依頼にあった「a moved checkout」は、計画の Test plan の「worktree の中で別のブランチをチェックアウトした場合」と読んだ。main のチェックアウトそのものを別の場所に移した場合は、計画に項目がなく、テストもない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`(`internal/cli`、HEAD の 1 回目の全体実行だけ) | `non-strict: err = doctor: 1 check(s) failed, want nil`(10.39 s) | この差分の外。`git diff --stat 765da6bd...HEAD -- internal/cli/doctor*.go` は空。テストは空の stub の `claude`・`codex`・`go` を置いて `runDoctorFull` を呼び、`probeBinary` が `--version` を 5 秒で打ち切る。load average が 100 を超えていた時間帯で、stub の起動が 5 秒を超えたと見ている。テストは doctor の出力を失敗時に出さないので、どの check が落ちたかは見ていない。推測で、未確認です | 単独の `-count=5` は 5 回とも通り(8.7 s)、2 回目の全体実行でも通った。この差分の修正は要らない。flake としてエージェントの記録に残す |
| `TestRunWatcher_TimeoutIndependentOfSmallInterval`(`internal/org`、mutation の 2 回だけ) | `invoke claude: claude: timed out: context deadline exceeded` | 既知の flake。実在のサブプロセスを起こすテストが並ぶ負荷のもとで出る。mutation はこのテストの経路を変えていない | mutation でない実行(`run-test.sh`、`-count=1` の 2 回、`-race`)ではすべて通った |

テストの失敗による fail はない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 3 段目の予約と、自分の close の失敗の補償 | 保たれている | 絞り込みで `CloseDeferredSelf` と `ReleasedReservation` の 25 本、予約の判定のテストが通る。AC10 の `TestOrgCloseDeferredSelf_CloseFails_BindingRestored` と `TestReleasedReservation_KeepsBinding` も通る |
| `--plan` なしの `ralph org start <task>` | 保たれている | `internal/cli` の `TestOrgStart_*` 15 本(予約、org 全体の上限、model、scope、`--cwd` と `--org-id` の必須)と `TestOrgStart_WithoutPlan_StillTakesExactlyOneTask` |
| spawn の冪等 | 保たれている | `TestStartFeature_SameStartAgainReusesWorktree`、`TestOrgSpawn_Feature_SameBindingAgainRecordsNothing`、`TestSpawnPrecheckErr_MatchesSpawn`(16 件、先読みと `Spawn` のエラーの一致) |
| 相対の `--cwd` を herdr にそのまま渡していた(af138af6 で修正) | 直っている | `TestOrgSpawn_RelativeCwdResolvedAgainstCallerWorkingDir` の 3 件 |
| digest の Go の実装とスクリプトのずれ | ずれていない | `TestPlanDigest_MatchesScript` の 114 件(生成した 13 件、雛形、`docs/plans/{active,archive}` の全計画)がすべて一致 |

## Test gaps

- T-1: herdr 本体の 32 文字の上限には当てていない。fake の herdr なので、確かめたのは ralph 側の `checkSpawnInput` の判定である。実機の記録(AC14 の Run 3)の slug は `hello`(5 文字)
- T-2: ensure の失敗のうち、state の衝突は本物のスクリプトでは start から届かない。再利用の検査が先に記録を読み、違えば ensure の前に拒否するためで、届くのは先読みと ensure の間に記録が変わる競合のときだけである。jq がない、default branch がない、`.codex/config.toml` の書き換え(案内を足さない側)も、fake の ensure とスクリプトの文の存在の検査だけで見ている
- T-3: 承認のあとに分割計画の改行を LF から CRLF に変えると、digest が変わる。scratchpad の probe では、同じ内容で LF が `8fdeb8b031c9`、CRLF が `7b450efec517` だった。start は「changed after its approval」で拒否するはずで、`scripts/plan-visual.sh digest` と同じ動きである。テストでは固定していない
- T-4: 同じ機能の `start --plan` を同時に 2 つ打つ場合(計画の進捗の (4))はテストがない(verify と同じ)
- T-5: CLI の `runOrgStartPlan` の `newOrgSpawnRuntimeAt` と `resolveModelOrWarn` の失敗の枝は通っていない。model の拒否は org 層の `RefusedBeforeAnyRecord/a_model_outside_the_pool` で見ている
- T-6: `StartFeature` の防御の枝(leader の入力の検査の失敗、台帳の読み込みの失敗)と、`scriptFeatureWorktrees` の出力が空・JSON でない・git が失敗する枝は通っていない(Coverage の表)
- T-7: 実機の AC14 は打ち直していない。push と `gh pr create` が通る場合と codex の leader は未確認(計画の Non-goals、verify と同じ)

## Cycle 2(d49bbc34・07d38e6d・9c1d447f のあと)

- 実行: 2026-10-10(JST 01:55〜02:15、UTC 16:55〜17:15)。HEAD 9a2dc5ef、パイプライン 2 回目(`cycle-count.json` は 2、上限 2)
- 対象: `git diff 183cb190 HEAD` のコードとテスト。cross-review の ACTION_REQUIRED の 2 件を直した差分で、leader の task の `- 台帳:` の行(`feature.go`)、分割計画のコードフェンス(`split.go`)、`--state-dir` か env で main の台帳を指したときの全体の上限の読み元(`statedir.go` の `LedgerMainWorktreeRoot`、`internal/cli/org.go`)、その help の文。shell のテストとスクリプトは cycle 1 から変わっていない(`git diff --stat 421719f7 HEAD -- tests/ scripts/` は空)
- Evidence: `docs/evidence/test-2026-10-10-org-feature-worktree-cycle2.log`(commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-09-165543.log`
- 足したテスト: `internal/org/split_test.go` の `TestLoadSplitPlan_CodeFences` に 1 ケース(テストのファイルだけ、+6 行とコメントの 1 行)。理由は下の Mutation の F09・F21
- 負荷: load average は 6〜9 だった(cycle 1 は 90〜110)

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(HEAD、変更言語の scope) | shell 40 本(PASS の行 3,807)、Go 8 パッケージ | すべて | 0 | shell 1(cycle 1 と同じ git 2.41 の件) | 9 分 9 秒、rc 0 |
| `go test ./internal/org/ ./internal/cli/ -count=1 -coverprofile -v`(HEAD) | top-level 962、subtest 1,079 | すべて | 0 | 0 | 1 分 28 秒(`internal/org` 19.2 s、`internal/cli` 86.0 s)、rc 0 |
| 一時の probe 3 本(`go test -overlay` で `internal/cli` に足した。worktree には書いていない) | top-level 3、subtest 16 | すべて | 0 | 0 | 16.1 s |
| mutation(`go test -overlay`、worktree のファイルは書き換えていない) | 33 件 | HEAD のテストで red 31 件 | 生き残り 2 件(F09、F21)。ケースを足して 2 件とも red | - | 1 件 17〜22 s |
| ケースを足したあとの `go test ./internal/org/ -count=1 -coverprofile` | 1 パッケージ | 1 | 0 | 0 | 17.0 s |
| `go test -race -count=1 ./internal/org/...`(ケースを足したあと) | 3 パッケージ | 3 | 0 | 0 | 22 s |
| cycle 2 のテストを `internal/org` で `-count=10`(ケースを足したあと) | top-level 160、subtest 2,090 | すべて | 0 | 0 | 17.2 s |
| 同じく `internal/cli` で `-count=5` | top-level 20、subtest 55 | すべて | 0 | 0 | 31.7 s |

- shell の PASS の行は、`PASS: 29` のような集計行を除いて数えた。cycle 1 の `run-test.sh` のログ(`verify-2026-10-09-113451.log`)を同じ方法で数えても 3,807 になる。cycle 1 の報告にある 3,797 との差は数え方の違いで、テストの増減ではない
- `run-test.sh` の中の `go test ./...` が実際に流したのは `internal/org` だけで(17.6 s)、ほかはキャッシュの結果だった。そのため表の 2 行目で、2 パッケージを `-count=1` で流し直した
- `-count=10` の正規表現は `TestLoadSplitPlan|TestPlanDigest_MatchesScript|TestFeatureLeaderTask|TestLedgerMainWorktreeRoot|TestStartFeature_StartsLeaderInFeatureWorktree|TestRenderRolePrompt_Leader_FeatureOrgProcedure|TestSplitPlanCheckApproved_CRLF`。`-count=5` は `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml|TestOrgSpawnAndStartHelp_OrgWideLimitsSource|TestOrgSpawn_MainWorktreeRalphToml_OnlyOrgWideLimitsTaken|TestOrgStartPlan_RealWorktreeScript`

### Coverage(cycle 2)

- Statement: `internal/org` 93.8%、`internal/cli` 85.5%(どちらも cycle 1 と同じ。2 パッケージを合わせて 88.9%)。ケースを足す前と後で、`internal/org` の関数ごとの値は 1 つも変わらなかった
- cycle 2 で足した関数と変えた関数は、どれも 100.0%: `parseSplitPlan`、`splitFence.read`、`unclosedErr`、`fenceRun`(`split.go`)、`featureLeaderTask`、`shellQuote`、`startFeatureLeaderParams`(`feature.go`)、`LedgerMainWorktreeRoot`(`statedir.go`)、`withMainWorktreeOrgLimits`、`newOrgSpawnRuntime`(`internal/cli/org.go`)
- `newOrgSpawnRuntimeAt` は 88.9% で、通らないのは `newOrgRuntimeAt` の失敗の return(`internal/cli/org.go:136`、3 段目からの行)。`StartFeature` は 92.1%、`FeatureRepoRoot` は `internal/org` の profile で 0.0% のままで、どちらも cycle 1 の表と同じ

### 依頼の 4 点の確認

#### (1) `- 台帳:` の行の引用

| 確かめたこと | 方法 | 結果 |
| --- | --- | --- |
| `featureLeaderTask` の行の形 | 既存の `TestFeatureLeaderTask`(空白を含むパスと `'` を含むパス) | 通る(`-count=10`) |
| `--state-dir` の値が sh で 1 語になる | 既存の `TestFeatureLeaderTask_StateDirIsOneShellWord`(5 つの値を `sh -c 'set -- <語>'` に通す) | 通る |
| spawn した leader のプロンプトに行が入る | 既存の `TestStartFeature_StartsLeaderInFeatureWorktree` | 通る |
| CLI から最後まで | probe A。`ralph org start --plan --feature a --state-dir <dir>` を本物の `ralph-worktree.sh` で打ち、`<dir>/prompts/a_leader.md` の `- 台帳:` の行を読む。`<dir>` は `plain/state`、`my ledgers/state`、`it's/state`、`a'b c/$HOME/*/"q"` の 4 つ | 4 つとも、行は生のパスで始まり、`--state-dir` の値は `'` を `'\''` にした単一引用符の 1 語。`sh -c 'set -- <語>'` で引数は 1 つ、値は `<dir>` と同じ |
| leader が行の語をそのまま打った場合 | probe A の続き。HEAD でビルドした `ralph` を `sh -c "<bin> org status --org-id a --state-dir <語>"` で feature worktree から打つ | 4 つとも `feature: demo/a branch feat/a worktree <wt>` が出る。`--state-dir` を付けずに同じ場所から打つと `no seats`(rc 0)で、既定の解決は main worktree の台帳を指し、start の台帳とは別になる。cross-review の 1 件目が書いた台帳の分かれ方を、実行で見た形 |

#### (2) linked worktree からの spawn と全体の上限

- 既存の AC13 の表(`TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml`)の 7 ケースが通る。07d38e6d で変わったのは 2 ケース(`--state-dir` と env で main の台帳を指す)で、許可から拒否に反転した。cwd の linked worktree の `ralph.toml` に `max_orgs = 99` を置くので、打った場所の設定を読む実装なら許されて落ちる。足した 2 ケース(相対の `--state-dir`、別の台帳を指す `--state-dir`)も通る
- AC13 の表が打つのは `start` で、leader が打つのは `spawn` である。どちらも `newOrgSpawnRuntime` を通る(`internal/cli/org.go:444` と `:611`)が、`spawn` で `--state-dir` を付けた実行は既存のテストになかったので、probe B で打った。main の `ralph.toml` は上限 1、cwd の linked worktree の `ralph.toml` は `max_orgs = 99` と `max_total_seats = 99`、org-a の leader が動いている状態から、`spawn` を 9 通り打った

| 台帳の指し方 | max_orgs(新しい org-b の seat) | max_total_seats(org-a に seat を足す。leader が implementer を立てる場合) |
| --- | --- | --- |
| 絶対パスの `--state-dir`(main の台帳) | 拒否(`max_orgs 1 reached`) | 拒否(`max_total_seats 1 reached`) |
| 相対パスの `--state-dir`(main の台帳) | 拒否 | 拒否 |
| `RALPH_ORG_STATE_DIR`(main の台帳) | 拒否 | 拒否 |
| 指定なし(既定の解決) | 拒否 | (打っていない。既存の `TestOrgSpawn_MainWorktreeRalphToml_OnlyOrgWideLimitsTaken` が main の subdirectory から見る) |
| 別の台帳を指す `--state-dir`(org-a が動いている) | 許可(cwd の 99) | 許可(cwd の 99) |

- probe B はテストとして残していない。下の mutation で、probe B を red にした変異(L01〜L04、W01)は、どれも AC13 の表でも red になった。probe B だけが見分ける変異はなかった

#### (3) 分割計画のコードフェンス

- 既存のテストが通る: `TestLoadSplitPlan_CodeFences`(HEAD で 14 ケースと、header のフェンスの中の `## Features` で header が終わらないことの確認。この回に 1 ケース足して 15 ケース)、`TestLoadSplitPlan_Rejects` のフェンスの 13 ケース、`TestLoadSplitPlan_FencedBodyIsApproved`、`TestPlanDigest_MatchesScript` の `code fences`(Go の `PlanDigest` とスクリプトの digest が一致)
- probe C で CLI から通した。分割計画は本物の `scripts/plan-visual.sh digest` の値で承認した
  - 機能 a の本文に、`## Usage`・`### other`・`- Reserve: x/`・`- Type: docs`・`- Depends on: b` を入れた ```` ```sh ```` のフェンスと、`## Features` を入れた `~~~` のフェンスを置いた。`start --feature a` は通り、leader のプロンプトに本文がフェンスごと、後ろの文まで入る。a の scope は `reserve: internal/a/` のままで、フェンスの中の `- Reserve:` は入らない。フェンスの後ろの機能 b も start できる
  - 閉じていないフェンスは「split plan <path>: line 15: the code fence ``` is never closed」で拒否される。フェンスの中の `- Branch:` は「line 16: a - Branch: line is not allowed」で拒否される。どちらも worktree はできず、台帳に何も記録しない

#### (4) cycle 1 のテストの回帰

`go test ./internal/org/ ./internal/cli/ -count=1 -v` の結果を名前で数えた。失敗は 0。

| 名前の一部 | top-level | subtest |
| --- | --- | --- |
| `TestStartFeature` | 16 | 39 |
| `TestSpawnPrecheckErr_MatchesSpawn` | 1 | 16 |
| `RelativeCwd` | 1 | 3 |
| `EnsureFailure` | 2 | 8 |
| `TestOrgStartPlan` | 4 | 14 |
| `CloseDeferredSelf` | 23 | 58 |
| `ReleasedReservation` | 2 | 13 |
| `Reserv`(予約の判定と補償を含む) | 41 | 120 |
| `TestOrgStart_`(`--plan` なしの start) | 15 | 13 |

`git diff 183cb190 HEAD` で消えたテストの関数はなく、足されたのは 5 本(`TestOrgSpawnAndStartHelp_OrgWideLimitsSource`、`TestFeatureLeaderTask_StateDirIsOneShellWord`、`TestLoadSplitPlan_CodeFences`、`TestLoadSplitPlan_FencedBodyIsApproved`、`TestLedgerMainWorktreeRoot`)。期待を変えたのは AC13 の表の 2 ケースの反転だけで、理由は 07d38e6d のメッセージと計画の進捗にある(feature branch の `ralph.toml` で全体の上限を変えさせない)。反転のあとも見分ける力は残っている(W01 が red)。

### Mutation(cycle 2)

`split.go`・`feature.go`・`statedir.go`・`internal/cli/org.go` の写しを scratchpad で 1 か所だけ書き換え、`go test -overlay=<json> -count=1 -vet=off` で流した。`internal/org` の変異は `internal/org` 全体(`-run` なし)、L01〜L06 と W01 は `internal/cli` を AC13 と上限のテストと probe B の正規表現でも流した。

| # | Mutation | Red になったテスト |
| --- | --- | --- |
| F01 | 字下げ 4 つでも開く | `CodeFences/4_spaces_is_no_fence` |
| F02 | バッククォート 2 つでも開く | `CodeFences/two_backticks_is_no_fence` |
| F03 | タブの字下げでも開く | `CodeFences/a_tab_is_no_fence_indentation` |
| F04 | 閉じる行の文字を比べない | `Rejects` の 2 件(`~~~` と ```` ``` ```` の取り違え)、`CodeFences` |
| F05 | 開いた長さより短い行でも閉じる | `Rejects/fence_of_4_backticks_closed_by_3`、`CodeFences` |
| F06 | 閉じる行の後ろの文字を見ない | `Rejects/fence_closed_by_a_line_with_text_after_it`、`CodeFences/a_fence_line_with_an_info_string_does_not_close` |
| F07 | 閉じる行の後ろのタブを許さない | `CodeFences/closing_fence_with_spaces_and_tabs_after_it,_then_a_section` |
| F08 | inline code の判定を外す | `CodeFences/inline_code_is_no_fence` |
| F09 | inline code の判定をチルダにも当てる(`~~~ a~b` が開かない) | HEAD のテストでは生き残り。足したケースで red |
| F21 | バッククォートのある info string を、チルダのフェンスでも開かせない | HEAD のテストでは生き残り。足したケースで red |
| F10 | 閉じていないフェンスを拒否しない | `Rejects` の閉じていないフェンスの 7 ケース |
| F11 | 開いた行の番号を 1 つずらす | `Rejects` の同じ 7 ケース(エラーの行番号) |
| F12 | エラー文の開いた run を 3 文字に固定する | `Rejects/fence_of_4_backticks_closed_by_3` |
| F13 | フェンスの中の `## ` を見出しとして読む | `CodeFences` の 9 ケース、`FencedBodyIsApproved` |
| F14 | フェンスの行を本文に足さない | `CodeFences` の 8 ケース、`FencedBodyIsApproved` |
| F15 | 機能の中のフェンスの行をフィールドと見出しとして読む | `CodeFences` の 5 ケース、`FencedBodyIsApproved` |
| F16 | `\r` を落とす前の行でフェンスを読む | `CodeFences/the_same_with_CRLF_lines` |
| F17 | フェンスの中では digest が読まない行を拒否しない | `Rejects` のフェンスの中の `- Branch:`・チェック済みの箱・`## Progress checklist` |
| F18 | フェンスの中では header より後の `- Status:` を拒否しない | `Rejects/status_inside_a_fence_in_a_feature` |
| F19 | header のフェンスの中の行を header の行として読まない | `Rejects` の header の 2 件(2 つ目の `- Status:` と `- Approved:`) |
| F20 | 最初の機能より前のフェンスの行をフィールドの検査に回す | `CodeFences/fence_before_the_first_feature` |
| Q01 | `shellQuote` が `'` をエスケープしない | `FeatureLeaderTask`、`FeatureLeaderTask_StateDirIsOneShellWord` |
| Q02 | 二重引用符で囲む | 上の 2 本と `StartsLeaderInFeatureWorktree`、`RenderRolePrompt_Leader_FeatureOrgProcedure` |
| Q03 | `--state-dir` の値を引用しない | Q02 と同じ 4 本 |
| Q04 | 台帳の代わりに repo の root を渡す | `StartsLeaderInFeatureWorktree` |
| Q05 | `'` を `\'` にする | `FeatureLeaderTask`、`FeatureLeaderTask_StateDirIsOneShellWord` |
| L01 | source が `flag` のとき main の root を返さない | `TestLedgerMainWorktreeRoot`、AC13 の `--state-dir` の 2 件、probe B |
| L02 | source が `env` のとき返さない | `TestLedgerMainWorktreeRoot`、AC13 の env、probe B |
| L03 | 台帳のパスを比べない(どの `--state-dir` でも main の上限) | `TestLedgerMainWorktreeRoot`、AC13 の別の台帳、probe B |
| L04 | 台帳でなく main の root と比べる | `TestLedgerMainWorktreeRoot`、AC13 の 3 件、probe B |
| L05 | `samePath` を文字列の比較にする | `TestLedgerMainWorktreeRoot`(symlink のケース)だけ。CLI のテストは通る |
| L06 | source が `git-toplevel` と `cwd` でも返す | `TestLedgerMainWorktreeRoot`(否定のケース)だけ。CLI のテストは通る |
| W01 | `withMainWorktreeOrgLimits` を cycle 1 の `MainWorktreeRoot` に戻す | AC13 の 3 件、probe B |

- HEAD のテストで 33 件のうち 31 件が red になった。生き残った F09 と F21 は同じ規則の穴で、CommonMark ではチルダのフェンスの info string にチルダもバッククォートも書けるし、コードもそう読む(`splitFence.read` は inline code の判定をバッククォートのフェンスだけに当てる)。その規則を固定するケースがなかった。`TestLoadSplitPlan_CodeFences` に「a tilde fence's info string may hold tildes and backticks」を足した。`~~~` の info string に `~` とバッククォートを 1 つずつ含め、フェンスの中に `## Usage` を置くケースである。HEAD では通り、F09・F21 では「line 11: the code fence ~~~ is never closed」で落ちる
- `internal/org` 全体の実行は 34 回(L01〜L06 を含む 32 件と、ケースを足したあとの F09・F21)。どの回でも、変えた規則と関係のないテストは 1 本も落ちなかった

### Flaky(cycle 2)

- flake は出なかった。cycle 1 で 1 回落ちた `TestRunDoctorFull_StrictFlipsExitCode_DriftedCore` は、2 行目の uncached の実行で通った。`TestRunWatcher_TimeoutIndependentOfSmallInterval` は 2 行目、`-race`、変異の `internal/org` 全体の 34 回(2 つずつ並べて流した)のどれでも落ちなかった。負荷が cycle 1 より低かったので、flake が出なくなったとは言えない

### Test gaps(cycle 2)

- T2-1: claude の leader が `- 台帳:` の行に従ってすべての `ralph org` のコマンドに `--state-dir` を付けるかは、実機で走らせていない(verify の Coverage gaps と同じ)。probe A で確かめたのは、行の語をそのまま打てば sh を通って本物のバイナリが start の台帳を読むところまで
- T2-2: herdr の pane に start を打った環境の `RALPH_ORG_STATE_DIR` が届くかどうかは試していない(self-review の C2-2 の (c))
- T2-3: `--plan` なしの `ralph org start <task> --state-dir X` の leader には台帳が渡らない(self-review の C2-6 の (g))。計画の範囲の外で、テストはない
- T2-4: まだない台帳を、symlink を含む別名のパスで `--state-dir` に渡す場合(`samePath` が文字列の比較になる。self-review の「finding にしないもの」)はテストがない。`TestLedgerMainWorktreeRoot` の symlink のケースは台帳がある状態だけを見る
- T2-5: 閉じていないフェンスの後ろに digest が読まない行がある場合、どちらのエラーが先に出るかを固定するテストはない(self-review の「finding にしないもの」)
- T2-6: probe A〜C はテストとして残していない。probe A と C は、既存のテストが固定する 3 つの部品(`featureLeaderTask` と `sh`、`StartFeature` からプロンプトまで、`LoadSplitPlan` とスクリプトの digest の一致)を CLI でつないで打ったもの。probe B は上の (2) のとおり
- cycle 1 の T-1〜T-6 は変わらない。T-7(実機の AC14)は、d49bbc34 で leader のプロンプトと task の文が変わったあとも打ち直していない

## Verdict

- Verdict: pass
- Pass(cycle 2): `run-test.sh` が rc 0(shell 40 本、PASS の行 3,807、FAIL 0、SKIP 1。Go 8 パッケージ)。`internal/org` と `internal/cli` を `-count=1` で流した実行が rc 0(top-level 962、subtest 1,079)。`-race`、`-count=10` と `-count=5` も通った。依頼の 4 点は、既存のテストと一時の probe 3 本の実行で確かめた。変異 33 件のうち HEAD のテストで生き残った 2 件は、足した 1 ケースで red になった
- Fail: なし。flake も出なかった
- Blocked: なし。T2-1〜T2-6 と cycle 1 の T-1〜T-7 は merge を止めない
- 参考(cycle 1 の判定、HEAD 421719f7): pass。`run-test.sh` が rc 0、`internal/org` と `internal/cli` の 2 回目の `-count=1` が rc 0。HEAD の 1 回目の `internal/cli` の 1 件は、この差分の外の doctor のテストの負荷による flake と判断した(単独 5/5、2 回目の全体実行で通過。原因の特定は推測で、未確認です)
