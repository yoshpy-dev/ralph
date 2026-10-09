# Test report: org-feature-worktree

- Date: 2026-10-09(JST。実行は UTC の 11:34〜12:14)
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md(承認済み、digest 56e435bfa976)
- Tester: tester subagent (Claude Opus 5.5)、パイプライン 1 回目(`cycle-count.json` は 1、上限 2)
- Scope: HEAD 421719f7(base origin/main 765da6bd)。behavioral test だけを実行した(静的解析は /verify の担当)。重点は 3 つ。self-review が実行での確認を求めた 2 点(M1 の 20 文字と 21 文字の境界、ensure の失敗の案内を本物のスクリプトの文から選ぶこと)、計画の Test plan の edge case にテストがあるか、3 段目の予約と補償と `start <task>` が壊れていないか
- Evidence: `docs/evidence/test-2026-10-09-org-feature-worktree.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-09-113451.log`
- 足したテスト: `internal/org/feature_test.go` に 2 本、`internal/org/split_test.go` に 1 本(テストのファイルだけ、+122 行)。理由は「依頼された 2 点」と「Edge cases」の節

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

## Verdict

- Verdict: pass
- Pass: `run-test.sh` が rc 0(shell 40 本、PASS の行 3,797、FAIL 0、SKIP 1。Go 8 パッケージ)。`internal/org` と `internal/cli` を `-count=1` で流した 2 回目の実行が rc 0。`-race`、`-count=10`、残りの 6 パッケージも通った。依頼の 2 点は、既存のテストと足した 3 本の実行で確かめた。計画の Test plan の edge case 10 件にはどれもテストがある
- Fail: なし。HEAD の 1 回目の `internal/cli` の 1 件は、この差分の外の doctor のテストの負荷による flake と判断した(単独 5/5、2 回目の全体実行で通過)。原因の特定は推測で、未確認です
- Blocked: なし。T-1〜T-7 は merge を止めない
