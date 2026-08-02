# org-runtime-lead

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-02
- Related request: docs/specs/2026-08-01-org-runtime.md(PR 系列③: Lead 自律編成)
- Related issue: N/A
- Type: feat
- Branch: feat/org-runtime-lead

## Objective

org runtime PR③: Lead が組織を自律編成できるようにする。`/org` skill(Lead の操作マニュアル)、`ralph org start`(headless Lead 起動)、座席の permission-mode エンベロープ設定(PR② で座席が権限ダイアログ blocked になった問題の解消)、org manifest の成果物化(FR-9 後半)、および PR③ 送りの tech-debt 6 件をクローズする。

## Scope

- **座席 permission-mode**(ユーザー確定: 役割別エンベロープ設定・既定は自律): `[org.permissions]` に driver 非依存の mode 列挙(`autonomous` / `edits` / `guarded`)を役割別に定義(`default` キー+役割キー)。既定は全役割 `autonomous`。spawn が driver ネイティブフラグへ変換(claude: `--permission-mode bypassPermissions|acceptEdits|default`)。ロックステップ 3 面 + defaults_sync_test 拡張。
  - **最小制御ゲート**(Codex 所見 1): mode=autonomous の spawn は `--scope` 指定と有界タイムアウト(TimeoutMS 既定あり)を**必須**とし、`--scope` なしは fail-closed(明示フラグ `--allow-unscoped` でのみ解除、使用は manifest に記録)。適用された permission mode は `spawned` イベントと `org report` に記録し、監査可能にする。
  - **codex は fail-closed**(Codex 所見 2): codex 対話モードの権限フラグは実機検証まで信頼しない。mode=guarded(フラグ無し=CLI 既定)以外を codex 座席に指定した場合は明確なエラーで拒否し、receipt に記録する(スタブ argv が通るだけの「サイレント no-op」を構造的に排除)。実機検証後に解除する条件を tech-debt に記録。
- **`ralph org start`**(headless Lead 起動): `ralph org start --org-id <id> [--driver claude] [--model <m>] "<task>"` = 役割 `lead` の座席を spawn する糖衣(seat id `lead`、新設 `internal/org/prompts/lead.md` 雛形に task を展開)。lead 座席は herdr pane 内の常駐セッションとして稼働し、`/org` skill と `ralph org` 動詞で座席を編成する。現セッション昇格運用(こちらが主経路)は `/org` skill に手順として記載(機構変更なし)。
- **`/org` skill**(`.claude/skills/org/SKILL.md` + `.agents/skills/` ミラー): 動詞リファレンス(spawn/send/wait/read/stop/status/disband/report)、編成パターン(Solo / Leaded / Parallel の型と使い分け)、typed protocol(`.claude/rules/agent-messaging.md` 参照)、budget/エンベロープ作法、役割一覧(lead/reviewer/qa)、Lead 運用手順(現セッション昇格・headless の両方)、受信箱運用(agmsg 通知の扱い)。`templates/base/.claude/skills/` への同梱と `sync-skills.sh` 再生成、`check-skill-sync.sh` green。CLAUDE.md の auto-invoked skill 一覧へ追記(ミラー同期)。
- **org manifest 成果物化**(FR-9 後半): 新動詞 `ralph org report --org-id <id>` が manifest + receipts から編成履歴(roster、saga イベント、モデル receipts、known 残留)を `docs/reports/org-manifest-<org_id>-<date>.md` に書き出す。
- **tech-debt 6 件のクローズ**(全て docs/tech-debt/README.md 登録済み):
  1. announce(HELLO)失敗時の補償に best-effort `Leave` を追加(cross-review cycle-2 known gap)
  2. `leadIdentity` 定数化+lead の agmsg type を lead driver から導出(ハードコード解消)
  3. herdr エージェント名の manifest 永続化(`herdr_agent_name` フィールドを `spawned` に記録、send/wait/stop は記録値優先・導出はフォールバック)
  4. 並行 spawn TOCTOU: manifest ディレクトリのファイルロック(`flock` 相当)で read→validate→append を直列化
  5. doctor の agmsg 検査が解決済み home パスを表示
  6. `dryRunSpawn` の RenderRolePrompt / promptFilePath エラー握り潰し解消(実行パスと同じ失敗を返す)
- **実機スモーク**(手動・証拠必須): (1) 実機で autonomous 座席が bash コマンド(例: `git status` 読み取り)を**権限ダイアログで blocked にならずに**実行できること(PR② の blocked 観測の解消確認)。(2) `ralph org start` で headless lead 座席が起動し、lead.md 雛形のタスクを受けて稼働状態になること。(3) **Lead 最小 E2E**(Codex 所見 4): headless lead に「dry-run 座席を 1 つ spawn し、typed message を send し、status を確認して disband せよ」というタスクを与え、lead が `/org` skill の指示と `ralph org` 動詞で実際に制御面を操作できることを証明する(多座席フル編成は非目標のまま)。(4) スモーク後の stop/disband クリーンアップ完了(herdr workspace / agmsg team 残留なし)を確認。証拠は `docs/evidence/org-lead-smoke-*.txt`。

## Non-goals

- Watchdog 二層・budget 自動遮断・スコープ違反検知(PR④)。
- 旧系撤去・ドキュメント全面改稿(PR⑤)。`/org` skill 追加に伴う CLAUDE.md/AGENTS.md の追記は最小限に留める。
- Lead が実際に多座席編成を行うフル自律デモ(コスト大。スモークは lead 起動と権限解消の確認まで。フル編成の実証は PR④ の Watchdog と合わせた運用検証で行う)。
- 標準フロー/Ralph Loop の吸収(PR⑤)。
- effective model 実測(PR④ ウォッチャーと同時)。

## Assumptions

- claude CLI は対話モードで `--permission-mode`(`bypassPermissions` / `acceptEdits` / `default` / `plan`)を受け付ける。codex は sandbox / approval 系フラグで相当を表現(実装時に実フラグを確認し、スタブ argv テストで固定)。
- 座席の cwd は呼び出し側指定のまま(worktree 分離の強制は PR④/⑤ の運用設計)。autonomous 既定の安全性は「人間がエンベロープで役割別に絞れる」+「PR④ Watchdog」+「座席 worktree 運用」で担保する設計判断(ユーザー確定)。
- `ralph org start` の lead 座席も通常の座席として manifest / receipts に記録される(特別扱いしない)。
- flock は同一ホスト内の直列化のみ保証(分散はスコープ外)。

## Affected areas

- `internal/config/`(`[org.permissions]` 追加、ロックステップ、defaults_sync_test)
- `internal/org/`: spawn.go(permission フラグ、Leave 補償、leadIdentity、herdr_agent_name 記録、flock、dry-run エラー)、manifest.go(HerdrAgentName フィールド)、verbs.go(記録値優先の名前解決、report ロジック)、prompts/lead.md(新規)、prompts.go
- `internal/cli/`: org.go(`start` / `report` サブコマンド、permission 関連配線)、doctor.go(agmsg home 表示)
- `.claude/skills/org/SKILL.md`(新規)+ `.agents/skills/org/` ミラー + `templates/base/` 同梱、CLAUDE.md(+テンプレミラー)
- `scripts/ralph-config.sh` + `templates/base/scripts/ralph-config.sh`、`templates/base/ralph.toml`
- `docs/tech-debt/README.md`(6 行クローズ)
- 触らない: `scripts/ralph-orchestrator.sh` / `ralph-pipeline.sh` / `internal/ui` / `internal/state`

## Design decisions

- **フロー**: Standard flow (/work)(ユーザー確定)。実機スモークが対話観察必須、skill 記述が動詞最終形に依存。
- **権限モデル**: 役割別エンベロープ設定・既定 autonomous(ユーザー確定)。壁は機構層+PR④ Watchdog+worktree 運用が担い、人間はエンベロープで役割単位に絞れる。driver 非依存 enum → driver ネイティブフラグ変換は spawn 内の grep-able な変換関数に集約。
- **`org start` = lead 座席の spawn 糖衣**: Lead を特別な実行体にせず、既存の座席機構(saga / manifest / receipts / 役割雛形)をそのまま使う。機構の対称性を保ち、PR④ の Watchdog も lead を通常座席として監視できる。
- **herdr 名の永続化は additive**: `ManifestEvent` に `herdr_agent_name`(omitempty)を追加。旧イベントは記録なし → 導出フォールバックで互換。
- **Codex advisory 反映(4 件)**: ① autonomous の最小制御ゲート(--scope 必須 fail-closed・`--allow-unscoped` 明示解除・mode の manifest/report 記録)、② codex 権限の fail-closed(guarded 以外拒否+receipt)、③ スライス別 green ゲート+ロールバック基準の明文化、④ Lead 最小 E2E スモーク(dry-run 座席 spawn → send → status → disband)。
- Critical forks: フロー・権限モデルの 2 点を解決済み。残りは既存構造の延長。

## Acceptance criteria

- [x] AC-1: `[org.permissions]` が 3 面ロックステップで追加され(既定: `default = "autonomous"`)、役割別上書きが機能し、defaults_sync_test が drift を検出する。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (Spec compliance, AC-1), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC → behavioral test mapping, AC-1).
- [x] AC-2: spawn の AgentStart argv に mode 対応の driver ネイティブフラグが載る(claude: `--permission-mode bypassPermissions` 等。スタブ argv テストで claude × 3 mode を検証)。**codex 座席は guarded 以外を明確なエラーで拒否**し receipt に記録する(fail-closed テスト)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-2), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-2).
- [x] AC-2b(最小制御ゲート、実装時に監査経路を改善): mode=autonomous の spawn は `--scope` なしで fail-closed(`--allow-unscoped` でのみ解除・使用が manifest に記録される)。自己レビュー修正(commit `9a22942`)により、拒否パスは `reject()` を経由するようになり、fail-closed の判定自体は変えずに `rejected` manifest イベント + `honored=false` receipt を追加で記録する(拒否の監査性が向上)。適用 permission mode は `spawned` イベントに記録される。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-2b row and "Self-review fix-commit crosscheck" LOW row), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-2b).
- [x] AC-3: `ralph org start --org-id X "<task>"` が role=lead の座席を spawn し、lead.md 雛形に task / envelope 要約が展開される(スタブテスト+実機)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-3), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-3), `docs/evidence/org-lead-smoke-2026-08-02.txt` (Smoke B).
- [x] AC-4: `ralph org report` が manifest+receipts から編成履歴レポートを `docs/reports/` に生成する(スタブデータのユニットテスト)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-4), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-4).
- [x] AC-5: `/org` skill が存在し、`.agents/skills/` ミラーと `templates/base/` 同梱が check-skill-sync / check-sync green。CLAUDE.md の一覧に追記済み(ミラー同期)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-5, Static analysis: check-sync/check-skill-sync both PASS).
- [x] AC-6(tech-debt #1): announce 失敗時に join 済み座席へ best-effort `Leave` が走り、結果が spawn_failed の Details に記録される(失敗注入テスト)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-6), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-6), `docs/tech-debt/README.md` (row RESOLVED).
- [x] AC-7(tech-debt #2): `leadIdentity` 定数が唯一の "lead" 出所となり、lead の agmsg type が lead driver から導出される。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-7), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-7), `docs/tech-debt/README.md` (row RESOLVED).
- [x] AC-8(tech-debt #3): `spawned` イベントに `herdr_agent_name` が記録され、send/wait/stop が記録値を優先し導出をフォールバックにする(旧形式イベントとの互換テスト)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-8), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-8), `docs/tech-debt/README.md` (row RESOLVED).
- [x] AC-9(tech-debt #4): 並行 spawn がファイルロックで直列化され、同時 spawn で max_seats 超過が起きない(並行テスト)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-9), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-9), `docs/tech-debt/README.md` (row RESOLVED).
- [x] AC-10(tech-debt #5/#6): doctor が解決済み agmsg home を表示し、dry-run が RenderRolePrompt / promptFilePath のエラーを実行パスと同様に失敗として返す。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-10), `docs/reports/test-2026-08-02-org-runtime-lead.md` (AC-10), `docs/tech-debt/README.md` (row RESOLVED).
- [x] AC-11(実機スモーク): autonomous 座席が bash 読み取りコマンドを権限ダイアログなしで実行(PR② blocked の解消)、`ralph org start` の headless lead が稼働。証拠 `docs/evidence/org-lead-smoke-2026-08-02.txt`(Smoke A/B)。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-11, "Met (evidenced)"; Coverage gaps notes single-run, not independently re-executed).
- [x] AC-11b(Lead 最小 E2E): headless lead がタスク指示に従い、dry-run 座席の spawn → typed message send → status 確認 → disband を実機で完遂する(lead prompt + /org skill + 動詞の合成が機能する証明)。スモーク後に herdr workspace / agmsg team の残留がないこと。Evidence: `docs/evidence/org-lead-smoke-2026-08-02.txt` (lines 88-159), `docs/reports/verify-2026-08-02-org-runtime-lead.md` (AC-11b, "Met (evidenced)").
- [x] AC-12: `go test ./...` / `./scripts/run-verify.sh` green。既存フロー非干渉。tech-debt 6 行が RESOLVED 化。Evidence: `docs/reports/verify-2026-08-02-org-runtime-lead.md` (static half PASS), `docs/reports/test-2026-08-02-org-runtime-lead.md` (test half PASS — 978 shell assertions + 13/13 go packages + targeted `-race` run, 0 failures), `docs/tech-debt/README.md` (6 rows struck through, RESOLVED 2026-08-02 in feat/org-runtime-lead).

## Implementation outline

1. **Slice 1 — tech-debt 群(Go 強化バッチ)**: AC-6〜AC-10 の 6 件(announce Leave / leadIdentity / herdr_agent_name 永続化 / flock / doctor 表示 / dry-run エラー)。相互独立だが同一ファイル群のため 1 スライスで。
2. **Slice 2 — permission-mode エンベロープ**: `[org.permissions]` 3 面ロックステップ、mode→driver フラグ変換、spawn 配線、argv テスト。
3. **Slice 3 — lead.md + `org start` + `org report`**: 雛形、start/report サブコマンド、テスト。
4. **Slice 4 — `/org` skill**: SKILL.md 執筆、ミラー再生成(sync-skills.sh)、templates 同梱、CLAUDE.md 追記(+ミラー)。
5. **Slice 5 — 実機スモーク + 証拠 + ドキュメント**: 権限解消確認と headless lead 起動、evidence 保存、AGENTS.md 最小追従、tech-debt 行クローズ確認。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`。
- Spec compliance criteria to confirm: FR-2(roles/エンベロープ拡張)、FR-5(`ralph org start` の headless 経路)、FR-9 後半(`docs/reports/` への成果物化)、FR-10(`/org` skill)。PR①②の AC 群にリグレッションなし。
- Documentation drift to check: skill ミラー(check-skill-sync)、templates ミラー(check-sync)、CLAUDE.md/AGENTS.md、tech-debt 行の整合、spec との整合。
- Evidence to capture: argv テスト出力、実機スモークログ、report 動詞の生成物サンプル。

## Test plan

- Unit tests: permissions 設定の解決(既定/役割上書き/未知 mode 拒否)、mode→フラグ変換(claude/codex × 3 mode)、lead.md 展開、report 生成、herdr_agent_name の記録/優先/フォールバック、flock 直列化(並行 spawn)、dry-run エラー伝播、Leave 補償注入。
- Integration tests: スタブでの `org start` → status → stop、`org report` 出力、既存ライフサイクルのリグレッション。
- Regression tests: PR①②全テスト、defaults_sync_test、check-sync、check-skill-sync。
- Edge cases: permissions 役割キーがプール外役割、herdr_agent_name 無し旧イベント、ロック競合タイムアウト、report 対象 org が空。
- Evidence to capture: `go test ./...` / `run-test.sh` 出力、スモークログ。

## Risks and mitigations

| リスク | 影響 | 緩和 |
|---|---|---|
| autonomous 既定の安全性懸念 | 座席の暴走余地 | 最小制御ゲート(--scope 必須 fail-closed+有界タイムアウト+mode の manifest 記録)を PR③ 内で実装。役割別にエンベロープで絞れる+PR④ Watchdog を skill/rule に明記 |
| codex の権限フラグ形状が対話モードで異なる | codex 座席の mode 不発(サイレント no-op) | fail-closed: codex は guarded 以外を明確なエラーで拒否+receipt 記録。実機検証完了までこの制約を維持(解除条件を tech-debt に記録) |
| flock 導入による性能/デッドロック | spawn 遅延 | ロック範囲を read→validate→append 最小区間に限定、タイムアウト付き |
| /org skill と CLAUDE.md 追記のミラー漏れ | CI red | check-skill-sync / check-sync をスライス内で実行 |

## Rollout or rollback notes

- 純追加系。`[org.permissions]` 未設定の既存 config は既定値で従来挙動+autonomous フラグ付与(挙動変化は「座席が blocked にならない」方向のみ)。ロールバックは revert。
- **スライス別 green ゲート**(Codex 所見 3): 各スライスは単独で `go test ./...`・check-sync/check-skill-sync・既存 `ralph org` 動詞の挙動を green に保った状態でコミットする(既存の Validation Gate を明文化)。スライス別ロールバック基準: Slice 1(tech-debt)と Slice 2(permissions)は互いに独立に revert 可能、Slice 3(start/report)は Slice 2 に依存、Slice 4(skill)は文書のみで単独 revert 可能。`herdr_agent_name` は omitempty の additive フィールドのため revert 後も旧リーダーで読める。

## Open questions

- codex 対話モードの権限フラグの正確な形状(実装時にスタブで固定、実機確認は claude 優先)。
- `org report` の出力に receipts の honored 集計を含めるか(実装時判断、最小は roster+イベント履歴)。

## Implementation notes (deviations)

- **実機発見 #1(bypass 承諾ダイアログ)**: claude の `--permission-mode bypassPermissions` はマシンごと初回 1 回、対話での承諾ダイアログを表示する(承諾は永続化)。機構による自動承諾は行わない(人間の同意画面のため)。運用前提として /org skill に明記。初回承諾はオペレータが herdr pane で 1 回行う。
- **実機発見 #2(herdr の done 状態)**: 入力待ちの対話エージェントを herdr は `idle` ではなく `done` と報告する。`send` の事前待機を `idle|done` 受理に修正(commit d35f157)。
- **実機発見 #3(state-dir は cwd 相対)**: lead(座席)と operator が異なる cwd で `ralph org` を実行すると manifest が分裂する(スモークで実測: lead の spawn/report は lead の cwd 側 `.harness` に記録)。運用規約「lead と operator は同一ディレクトリ(通常リポジトリルート)で実行」を /org skill に明記。恒久対応(state-dir の org_id 単位の共有解決)は PR④ 以降の検討として tech-debt 化。
- **Smoke A/B 成功**: autonomous 座席が typed TASK の bash をダイアログなしで実行し protocol 準拠 RESULT を返却。headless lead(sonnet)が 50 秒で E2E(dry-run spawn → send 試行 → status → report 生成 → disband)を完遂。stop 時の best-effort Leave が agmsg team を自動清掃。
- **Cycle-2 fix-and-revalidate(2 commits)**: `de4de50`(cross-review ACTION_REQUIRED #1 の修正 — idempotent-before-scope-gate reorder): `--scope` なしで再スポーンされた既存座席(自律既定)が、AC-2b ゲートより先に idempotent early-return で処理されるよう Phase 1 のロック内クロージャを並び替え。regression test: `TestOrgSpawn_Idempotent_NoScopeRetry_ReturnsExistingSeat`。`69be944`(cycle-2 self-review M-1/M-2 の修正): M-1 — dry-run パスと実スポーンパスのゲート順序(envelope → permission → AC-2b)を一致させ、`Spawn` のドキュメントコメントの矛盾を解消(regression test: `TestOrgSpawn_DryRun_And_Real_AgreeOnRejectionCause_EnvelopeBeforeScopeGate`)。M-2 — Phase 2 の再ロック区間で idempotent チェックを再実行し、`compensateStale` のロックフリー区間で完了したレーサーによる `spawn_started` 重複を防止(regression test: `TestOrgSpawn_StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent`)。両修正の詳細検証は `docs/reports/verify-2026-08-02-org-runtime-lead.md` Cycle 2 セクション、発見の経緯は `docs/reports/self-review-2026-08-02-org-runtime-lead.md` Cycle 2 セクション参照。

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created (feat/org-runtime-lead)
- [x] Implementation started — Slices 1-5 complete (commits `c04186e`, `9fe2af1`, `4b7a2e6`+`7be4993`, `039b18e`, smoke fixes `d35f157`+`428177b`, self-review fix `9a22942`)
- [x] Review artifact created (`docs/reports/self-review-2026-08-02-org-runtime-lead.md`)
- [x] Verification artifact created (`docs/reports/verify-2026-08-02-org-runtime-lead.md`)
- [x] Test artifact created (`docs/reports/test-2026-08-02-org-runtime-lead.md`)
- [ ] PR created

## Readiness checklist

- [x] フロー確定(Standard /work)・権限モデル確定(役割別エンベロープ・既定 autonomous)
- [x] PR②の実機制約(herdr 名前制約・agmsg leave 等)はメモリ/plan に記録済みで前提に織り込み
- [ ] Codex plan advisory(次ステップ)
