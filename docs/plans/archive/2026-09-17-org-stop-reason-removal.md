# org-stop-reason-removal

- Status: Done (PR #158)
- Owner: Claude Code
- Date: 2026-09-17
- Related request: PR #152 の budget 撤去で本番プロデューサを失った `StopParams.Reason`(と `Stop` の Details 追記)を削除する。`leadActivityEventCount` の `reason=watchdog_` 除外は、旧 watchdog が書いた manifest との互換ガードとして明示的に残す(issue #153 の選択肢 2 を、Codex plan advisory の HIGH 所見を受けて「プロデューサ側のみ削除」に絞った形)
- Related issue: 153
- Type: refactor
- Branch: refactor/org-stop-reason-removal

## Objective

`internal/org/verbs.go` の `StopParams.Reason` フィールドと `Stop` の `" reason=<Reason>"` Details 追記を削除し、`StopParams` を実際のプロデューサ(`ralph org stop`、`Disband`)が使う形だけにする。`internal/org/watch.go` の `leadActivityEventCount` にある `strings.Contains(ev.Details, "reason=watchdog_")` 除外は削除せず、「PR #152 以前の watchdog(budget cutoff)が書いた `stopped` イベントを lead activity に数えないための legacy 互換ガード」として doc comment を書き換える。合成 `stopped` を `Stop(Reason)` 経由で作っていた deadman テスト 2 件は、旧形式イベントを `Manifest.Append` で直接書く legacy fixture に書き換えて残し、保存済み `ManifestLen` ベースラインが旧 cutoff と共存しても alert が誤って消えないことを固定する回帰テストを 1 件追加する。最後に `docs/tech-debt/README.md` の該当行をクローズする。実行時挙動は変わらない。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | `StopParams.Reason` 削除 | `internal/org/verbs.go` | フィールドと doc comment(233-240 行)を削除。`Stop` 内の `if p.Reason != "" { details += " reason=" + p.Reason }` を削除 |
| 2 | 除外分岐の位置づけ変更 | `internal/org/watch.go` | `leadActivityEventCount` の `strings.Contains(ev.Details, "reason=watchdog_")` は**残す**。doc comment (b) の「no current pulse-layer condition calls Stop, so this exclusion is dormant today but stays in place for any future watchdog enforcement action」を、「この形の `stopped` を書く経路は PR #152 の budget 撤去以降存在しない。除外は 2026-09-17 以前の watchdog(budget cutoff)が書いた manifest との互換ガードで、(i) 旧 cutoff を lead activity に数えない、(ii) 撤去前に保存された pending alert の `ManifestLen` ベースラインと現在の数え直しの規則を一致させる、の 2 点のために残す」に書き換える。(a) 末尾の改名前テスト名参照も更新 |
| 3a | legacy fixture 化 | `internal/org/watch_test.go` | `TestWatch_Deadman_WatchdogsOwnStopEvent_DoesNotClearPendingAlert` → `TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert` に改名。`o.Stop(StopParams{..., Reason: ...})` を `o.Manifest.Append(ManifestEvent{..., Event: EventStopped, Details: "pane=ok leave=ok reason=watchdog_cutoff seat_wall_clock=30m observed=31m0s"})` に置き換え(旧 watchdog が書いたイベントの再現)。doc comment を「legacy manifest 互換」の説明に書き換える |
| 3b | legacy fixture 化 | `internal/org/watch_test.go` | `TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_WatchdogStopDoesNot` → `…_LegacyWatchdogStopDoesNot` に改名。seat-3 の cutoff を同じく `Manifest.Append` の旧形式イベントに置き換え、`sent` の正の検証(H3-1 pin)はそのまま残す。doc comment の参照名を更新 |
| 3c | Reason なしに書き換え | `internal/org/watch_test.go` | `TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert` の `Stop` 呼び出しから `Reason:` を外す。doc comment(1118-1126 行)と行内コメント(1149 行)の参照名を 3a の新名に更新し、「手動停止は Details に reason を持たない」旨に直す |
| 3d | アップグレード境界の回帰テスト追加 | `internal/org/watch_test.go` | `TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop`: 既存の `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` と同じ手書き status JSON 方式で、(i) manifest に seat-1 の `spawned` と旧 cutoff `stopped`(`reason=watchdog_…`)を持たせ、(ii) pending liveness alert を `manifest_len` = 現行規則で数えた値(旧 cutoff を除外した数)で保存し、(iii) 新規イベントなし・probe 不変・DeadmanMinutes 超過で 1 cycle 回す → alert が誤って消えず escalation が 1 回起きること、(iv) 続けて lead の `sent` を追加して次 cycle で alert が消えること、を検証する |
| 4 | tech-debt 行クローズ | `docs/tech-debt/README.md` | `StopParams.Reason` の行(129 行)を `~~` で打ち消し、Debt item セル末尾に `(RESOLVED 2026-09-18 in refactor/org-stop-reason-removal)`、Trigger セル末尾に「プロデューサ側のみ削除。reader 側の除外は legacy 互換ガードとして意図的に残す(Codex plan advisory HIGH-1: 撤去前に保存された pending alert の `ManifestLen` ベースラインが数え直し規則の変更でずれ、alert が誤って消える)」を追記。Related セルに本 plan の archive パスと self-review レポートを追加 |

## Non-goals

- `ralph org stop --reason` の追加(issue の選択肢 1 / 3 は不採用)
- `leadActivityEventCount` の除外分岐の削除と、それに伴う `ManifestLen` ベースライン移行ロジック(Design decisions の第 2 フォークで不採用)
- deadman の判定方式(全件数え直し vs alert 後の増分)の変更
- watchdog の他の条件・escalation 経路の変更。`watch.go` の escalation レコードにある `Reason` JSON フィールド(272 行・1013 行付近)は本 seam と無関係で触らない
- `Disband` の変更(`Stop` を Reason なしで呼んでおり、そのまま)
- docs/specs・recipes・rules の更新(grep で当該 seam への言及なしを確認済み。履歴レポート・archive plan の記述は当時の記録として残す)

## Assumptions

- deadman 判定(`checkDeadman`、`watch.go:967` 付近)は「alert 発生時に保存した `pending.ManifestLen`」と「現在の manifest 全件を `leadActivityEventCount` で数え直した値」の比較であり、alert 後の増分だけを見る方式ではない。したがって数え方の規則を変えると、撤去前に保存された pending alert のベースラインがずれる(Codex plan advisory HIGH-1 で判明。当初の「増分しか見ないので影響なし」という前提は誤りで撤回)。除外を残すことでこのずれは起きない
- `reason=watchdog_` を Details に持つ `stopped` を書く本番経路は PR #152 以降存在しない。テストでは旧 watchdog が書いた形を `Manifest.Append` で再現する(`Stop(Reason)` というプロデューサを装わない)
- issue の受け入れ条件「`grep -rn 'Reason' internal/org/verbs.go internal/org/watch.go` が空」は文字どおりには満たせない。`watch.go` には本 seam と無関係な escalation レコードの `Reason string \`json:"reason"\`` があるため。AC-1 は StopParams 由来の参照に限定した grep に置き換え、issue クローズ時にこの差異と「reader 側の除外を残した理由」をコメントで明示する

## Affected areas

- `internal/org/verbs.go`(`StopParams`, `Stop`)
- `internal/org/watch.go`(`leadActivityEventCount` の doc comment のみ。コードは不変)
- `internal/org/watch_test.go`(deadman テスト 3 件の書き換え + 1 件追加)
- `docs/tech-debt/README.md`(1 行)

## Design decisions

Critical forks: 2 件、いずれもユーザーと解決済み。

- **フォーク 1: `StopParams.Reason` の処遇 → 削除**(2026-09-17、AskUserQuestion。選択肢: 削除 / `--reason` 配線 / 配線+除外分岐削除)。理由: 本番プロデューサがゼロで、org runtime spec にも watchdog が `Stop` を呼ぶ enforcement の記述がない。`--reason` を配線しても除外分岐は到達不能のまま残る。将来 watchdog に enforcement を足す時に、その実装と同じ PR でプロデューサを再追加すればよい
- **フォーク 2: reader 側の除外分岐を消すか → プロデューサ側のみ削除、除外は legacy 互換ガードとして残す**(2026-09-18、AskUserQuestion。選択肢: プロデューサ側のみ削除 / 全削除+ベースライン移行 / 全削除でリスク受容)。Codex plan advisory(codex-cli 0.154.0)の HIGH-1「除外を消すと、旧 cutoff を含む manifest ではアップグレード後の再起動時に変化していない manifest でも数が増え、保留中 alert が lead activity と誤判定されて消える。条件は Active のままなので dedupe で再 alert も抑止され、revert では戻らない」を `checkDeadman` の比較式と `loadWatchStatus` の復元経路で確認して採用。ベースライン移行を書くのはクリーンアップ issue の範囲を超えるため不採用

既定として採った選択:

- 3a/3b は削除ではなく legacy fixture 化。除外分岐を残す以上、その挙動を pin するテストは必要で、しかも「旧 watchdog が書いたイベント」という本物のデータ形を再現するほうが、消したプロデューサを装うより正確
- 3d を追加。Codex の指摘したシナリオ(保存済みベースライン + 旧 cutoff + 新規イベントなし)を status JSON fixture で直接固定する。既存の `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` と同じ方式なので追加コストは小さい
- issue の grep ゲートは StopParams 由来に限定して読み替える(Assumptions 参照)

## Acceptance criteria

- [x] AC-1: `grep -n 'Reason' internal/org/verbs.go` が空。`grep -n 'Reason:' internal/org/watch_test.go` が空。`grep -rn 'p.Reason' internal/` が空(watch.go の escalation レコード `Reason` フィールドは対象外で残る)
- [x] AC-2: `leadActivityEventCount` のコードは不変で `reason=watchdog_` 除外が残っている(`grep -c 'reason=watchdog_' internal/org/watch.go` が 2 以上: コードとコメント)。doc comment に「dormant」「future watchdog enforcement」の記述がなく(`grep -n 'dormant\|future watchdog enforcement' internal/org/watch.go` が空)、legacy 互換ガードである旨と PR #152 への参照がある。`go vet ./internal/org/...` clean
- [x] AC-3: `grep -n 'WatchdogsOwnStopEvent\|_WatchdogStopDoesNot' internal/org/watch_test.go` が空(改名済み)。`TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert` と `TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot` が `o.Manifest.Append` で `reason=watchdog_` 付き `stopped` を書き、`o.Stop(` 経由ではない。`TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert` が Reason なしで pass
- [x] AC-4: `TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop` が存在し、(i) 新規イベントなしで escalation が 1 回起きる、(ii) `sent` 追加後の cycle で alert が消える、の両方を assert して pass
- [x] AC-5: `go test ./internal/org/... ./internal/cli/... -count=1` green。`./scripts/run-verify.sh` green
- [x] AC-6: `docs/tech-debt/README.md` の `StopParams.Reason` 行が `~~` + `(RESOLVED 2026-09-18 in refactor/org-stop-reason-removal)` でクローズされ、「プロデューサ側のみ削除、reader 側は legacy 互換で残す」と Codex HIGH-1 の理由が記録されている。行のパイプ数は 6 のまま、他行は無変更
- [x] AC-7: `grep -rn 'StopParams.Reason' docs/specs docs/recipes README.md .claude templates/base` が空(履歴レポート `docs/reports/` と archive plan は対象外)

## Implementation outline

1. Slice A(Go): 項目 1・2・3a〜3d。`gofmt -l internal/`、`go vet ./internal/org/...`、`go test ./internal/org/ -run 'TestWatch_Deadman' -count=1 -v`、`go test ./internal/org/... ./internal/cli/... -count=1`、`./scripts/run-verify.sh` → commit `refactor: drop the producer-less StopParams.Reason and pin the legacy watchdog-stop guard`
2. Slice B(docs、inline 例外: 単一ファイルの数行): 項目 4。パイプ数と他セルの不変を確認 → commit `docs: close the StopParams.Reason tech-debt row (#153)`
3. `./scripts/run-verify.sh` を通し、post-implementation pipeline へ

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt / go vet / golangci-lint / staticcheck)、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC-1〜AC-7 の grep ゲートを実行し空/非空を確認。`leadActivityEventCount` のコードが base(main @ c6b8127)と同一であること(`git diff main -- internal/org/watch.go` がコメント行のみ)
- Documentation drift to check: tech-debt 行のクローズ書式が他の struck 行と同じ、`/org` SKILL.md の `stop` 行(`--reason` を書いていないので変更不要)、spec に seam の言及がないこと
- Evidence to capture: grep ゲートの出力、`git diff main -- internal/org/watch.go` の出力、静的解析の exit code

## Test plan

- Unit tests: `internal/org` の deadman 系(`LegacyWatchdogStopEvent` / `CrossOrgActivity` / `SeatSentEvent…_LegacyWatchdogStopDoesNot` / `LeadSpawnedEvent` / `LeadSpawnsReplacementSeat` / `ManualStopOfOtherSeat` / 新規 `PersistedAlertBaseline…`)と `Stop` / `Disband` を叩く既存テスト
- Integration tests: `internal/cli` の `ralph org stop` CLI テスト(`StopParams` の形が変わるので compile と挙動を確認)
- Regression tests: `go test ./internal/... -count=1`
- Edge cases: (1) 3d で `manifest_len` を「旧 cutoff を除外した数」で保存し、新規イベントなしのまま DeadmanMinutes を超えても alert が残り escalation が起きる(除外を消した実装ならここで alert が消えて fail する、判別可能なテスト)。(2) 手動 `stopped` は Details に `reason=` を持たなくても lead activity に数えられる(`ManualStopOfOtherSeat`)。(3) legacy fixture の Details は旧 `Stop` の実出力形(`pane=ok leave=ok reason=watchdog_cutoff …`)に合わせる
- Evidence to capture: `go test -count=1 -v` の対象テスト出力、`docs/reports/test-*.md`

## Risks and mitigations

- 除外分岐を残すことで「到達不能コード」が残るという issue の原問題が半分残る → reader 側は legacy データで到達可能であり、doc comment と tech-debt クローズ注記でその位置づけを明示する。issue クローズコメントでも明記
- 3d の fixture が現行の status JSON スキーマとずれて `loadWatchStatus` が読めない → 既存の `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` の fixture を雛形にし、`manifest_len` と `ts` の意味を `checkDeadman` の実装から取る
- 3b の書き換えで `sent` の正の検証が弱くなる → 変更は seat-3 の cutoff の書き方(`Stop` → `Manifest.Append`)だけで、assert は不変
- issue の grep AC と plan の AC-1 の差異、および「全削除」からの方針変更 → PR 本文と issue クローズコメントに理由(Codex HIGH-1)を明示

## Rollout or rollback notes

単一 PR、実行時挙動の変化なし(`leadActivityEventCount` のコードは不変、`Stop` は Reason なしの出力と同一)。revert は PR 単位で安全。下流プロジェクトへの配布物(templates/base)に変更はない。

## Open questions

なし。

## Deviation notes

- 2026-09-18 plan: Codex plan advisory(codex-cli 0.154.0、初回は stdin 待ちでハングしたため `</dev/null` を付けて再実行)が HIGH 1 件を報告(除外分岐の削除で保存済み `ManifestLen` ベースラインがずれ、アップグレード境界で保留 alert が誤って消える)。`checkDeadman` の比較式と `loadWatchStatus` の復元で機構を確認し、ユーザーに再度選択を求めて「プロデューサ側のみ削除」を採用。Scope 2・3a〜3d、Assumptions、AC-2〜AC-4、Non-goals を全面改訂
- 2026-09-18 work: `./scripts/branch-name.sh from-plan` は `refactor/153/org-stop-reason-removal` を返すが、`/plan` が登録した worktree state のブランチ `refactor/org-stop-reason-removal` を `/work` 手順 2d(既存 state を resume)に従い維持
- 2026-09-18 work: Slice A = bce892c(implementer 委譲)。逸脱 1 件を採用: 3d のベースラインを legacy cutoff 追加の **前** に計算する。plan の手順どおり追加後に計算すると、ベースラインも同じ関数で算出されるため除外分岐を消しても両方が同じだけ動き、テストが判別できない。前に計算すれば除外の有無だけが recount とベースラインの一致/不一致を決める(テストの doc comment「Discrimination note」に記録)
- 2026-09-18 work: Slice B は inline 例外(単一ファイル数行)で orchestrator が実施。パイプ数 6、Impact/Why deferred セルのハッシュ一致、struck 行 44→45 を確認。RESOLVED 日付は実施日の 2026-09-18(AC-6 の記述も合わせて修正)
- 2026-09-18 work: 全 7 AC のゲートを orchestrator が再実行して達成を確認。verify スクリプト green(evidence: `docs/evidence/verify-2026-09-18-013259.log`、gitignored)
- 2026-09-18 self-review(cycle 1): Merge 判定、MEDIUM 1(M1: watch.go の除外ガード理由 (i) は単独では成り立たない。alert 前の legacy cutoff はベースラインと recount の両方に入って相殺されるため、ガードが要るのは (ii) の「ベースラインと recount の数え方が異なる」場合、すなわち永続化ベースラインのアップグレード境界か新旧バイナリ混在窓のみ)+ LOW 5(L1 新テストの doc に既存テストとの差分を明記、L2 legacy fixture コメントの忠実性表現、L3 `newFixture` の naked return / `*int` / alertID の再構築、L4 Discrimination note に `lead_agent_get`/`history_lead_lines` の役割、L5 「a `stopped` event」→ lifecycle 5 種)。tech-debt 行 Related セルへの self-review レポート追記も follow-up。#154 と同じく全件を同 cycle 内で修正する(Slice C、implementer 委譲)
- 2026-09-18 work: Slice C = 72b89d5(implementer 委譲、3 ファイル)。逸脱 3 件はいずれも妥当: seat-3 側の fixture コメントは既に正確で変更不要、L3 の説明語「naked」を検証 grep と衝突しない表現に変更、検証 grep 2 件(`exactly as` / `@1000000000`)は plan 対象外の既存テスト(`ProbeOutage…` は main 由来、`PrunesRetiredBudgetEntries…` の fixture)にも一致するため対象テスト内のみで判定。orchestrator 側でテスト関数集合の差分(改名 2・追加 1 のみ)、watch.go がコメントのみの変更であること、deadman 16 件 pass を確認
- 2026-09-18 self-review 再検証: M1・L1〜L5・follow-up 全件解消。修正コミットから新規 LOW 4 件(N1: 「only production caller」だが call site は sendAlert/checkDeadman の 2 箇所、N2: 傘文「異なる規則」が (b) の同規則ケースを覆わない、N3: 省略フィールド一覧の `dry_run` は omitempty で元から書かれない、N4: 構造体 doc の in-flight 参照)を Follow-ups として記録。いずれもコメント一句のため inline で修正 = ec9a226。reviewer の再検証は 1 ラウンドのみ(設計上)で、以降は `/verify` と `/cross-review` に委ねる
- 2026-09-18 plan drift: Scope 項目 2 が指示した除外ガードの説明文「(i) 旧 cutoff を lead activity に数えない / (ii) ベースラインの一致」は、self-review M1(理由 (i) は単独では成り立たず、alert 前の cutoff はベースラインと recount で相殺される)を受けて「ベースラインに無く recount にだけ現れる cutoff の 2 ケース((a) 永続化ベースラインのアップグレード境界、(b) 新旧バイナリ混在窓)」に置き換えた。出荷するコメントは plan の原文ではなく M1 後の文言が正

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created (#158)
