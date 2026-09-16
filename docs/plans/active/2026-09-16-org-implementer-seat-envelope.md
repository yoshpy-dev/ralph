# org-implementer-seat-envelope

- Status: Implemented (pipeline pending)
- Owner: Claude Code
- Date: 2026-09-16
- Related request: org runtime の役割・model_pool・budget 見直し(2026-09-16 チャットでの 3 点確認と方針確定)
- Related issue: N/A
- Type: feat
- Branch: feat/org-implementer-seat-envelope

## Objective

org runtime の座席機構を次の 3 点で見直す。

1. **implementer 座席の新設と座席内 fan-out の明文化**: 実装を担う座席が存在しない現状(lead 雛形は「reviewer / qa へ委譲」と書くが reviewer は read-only、qa は検証のみ)を解消し、implementer / reviewer / qa の全座席雛形で「座席内でサブエージェントを fan-out してよい(例: フロント / バック / インフラ実装の振り分け、レビュー観点の並列化、テストスイートの並列実行)」ことを明示する。
2. **model_pool 既定の刷新と `--model` 明示ルール**: 既定プールを claude `fable` / `opus` / `sonnet` / `haiku` + codex `gpt-6-astra` / `gpt-5.6-sol` / `gpt-5.6-terra` / `gpt-5.6-luna` / `gpt-5.5` にし、`ralph doctor` に codex スラッグの軽量検証(`models_cache.json` 読み取り、プロセス起動なし)を追加する。`--model` は必ず明示する運用ルールを skill と lead 雛形に置き、省略時は stderr 警告付きでプール先頭(claude は `fable`)にフォールバックする。
3. **budget 概念の削除**: `[org.budget]`(`seat_wall_clock_minutes` / `total_wall_clock_minutes` / `max_fix_rounds`)と watchdog の wall-clock 自動遮断を丸ごと撤去する。stall / 生存 / スコープ外変更 / デッドマンの各監視は存続する。

## Scope

### (1) 役割雛形

- `internal/org/prompts/implementer.md` 新設。内容は標準フローの `.claude/agents/implementer.md` の handoff 契約(scope、受け入れ基準、検証コマンド、スライス単位コミット、RESULT に commit SHA と `git show --stat HEAD` ポインタ)を座席向けに移植。
- `implementer.md` / `reviewer.md` / `qa.md` に「座席内 fan-out」節を追加: 座席は自身のドライバのサブエージェント機構(Claude Code: `Task`、Codex: `.codex/agents/` custom agents)で並列分担してよい。fan-out した子は座席の内部であり org の座席ではない(manifest に現れない、`max_seats` に数えない、lead への送信は座席本体だけが行う)。スター型と typed protocol は不変。
- `lead.md` の委譲先記述を「implementer / reviewer / qa」に修正し、`--model` 明示ルールを追記。
- `prompts_test.go` に implementer のレンダリングテストと、3 座席雛形が fan-out 節を含むことの回帰テストを追加。

### (2) model_pool / doctor / `--model`

- `internal/config/config.go` `Default()`、`templates/base/ralph.toml`、`scripts/ralph-config.sh`(+ `templates/base/scripts/ralph-config.sh` ミラー)の 3 面ロックステップで既定プールを更新。`config_test.go` / `defaults_sync_test.go` 追従。
- `templates/base/ralph.toml` の `[org.roles]` 例に implementer を加え、コメントで「claude はエイリアス、codex はスラッグ(`codex` にエイリアスなし)」を明記。
- `internal/cli/doctor.go` に Check「Org codex model slugs」を追加: `$CODEX_HOME/models_cache.json`(既定 `~/.codex/`)を読み、プール内 codex スラッグが一覧に無ければ `warn`、ファイル無し・codex 未導入なら `info` でスキップ。プロセス起動はしない。`--probe-models` は従来通り残す。テストは一時ディレクトリの fixture で pass / warn / skip の 3 経路。
- `internal/cli/org.go`: 現状 `spawn` は `--model` を必須フラグとして拒否し(`org.go:182` の required ループ)、`start` だけが `DefaultModelForDriver` へフォールバックしている(Codex 所見 3)。両者を共通ヘルパー `resolveModelOrWarn(cfg, driver, model, stderr)` に一本化し、`spawn` の required ループから `--model` を外し、フラグのヘルプ文を「省略時はプール先頭へフォールバック(警告付き)」に改める。省略時は stderr へ「`--model` 省略: `<driver>` のプール先頭 `<model>` にフォールバック。明示指定を推奨」を 1 行出す(fail-closed にしない)。`spawn` / `start` 両方に省略時の CLI テストを追加。
- `/org` skill に「`--model` は必ず明示する」ルールと既定プールの表を追加。`.claude/rules/ralph/model-routing.md` のエイリアス列挙に `fable` を追加。

### (3) budget 削除

- `internal/config/config.go`: `OrgBudgetConfig` 型、`Org.Budget` フィールド、`Default()` の値、`Load()` の 3 つの検証を削除。`config_test.go` / `defaults_sync_test.go` の該当ケースを削除。
- `templates/base/ralph.toml`: `[org.budget]` セクション削除。`scripts/ralph-config.sh` 両面: `RALPH_ORG_SEAT_BUDGET_MINUTES` / `RALPH_ORG_TOTAL_BUDGET_MINUTES` / `RALPH_ORG_MAX_FIX_ROUNDS` の定義と export を削除。`tests/test-ralph-config.sh` に knob 列挙があれば追従。
- `internal/org/watch.go`: `evaluateTotalBudget` / `evaluateSeatBudget`、`condSeatBudget` / `condTotalBudget`、`watchConditionRecord.Cutoff` ラチェット、`watchStatusFile.OrgStartTS`(budget 以外に利用者がいないことを確認のうえ)を削除。`evaluateCycle` / 座席評価の呼び出し順コメントを更新。`watch_test.go` の budget 系テスト(88 箇所参照)を削除し、残りのテストが `Cutoff` / `OrgStartTS` に依存していないことを確認。
- **旧 watch-status の prune**(Codex 所見 2): `watch-status-<org_id>.json` の `Conditions` / `PendingAlerts` / `Escalated` に budget 系エントリ(condition `seat_budget` / `total_budget`、および同 condition 由来の ALERT)が残っていると `checkDeadman` が全 PendingAlerts を走査して廃止済み制御の ALERT で人間へ誤エスカレーションしうる。読み込み直後に budget 系キーを prune する `pruneRetiredConditions` を追加し、旧 fixture(budget ALERT が pending の status ファイル)でエスカレーションが発生しないことを回帰テストで固定する。
- `internal/cli/org.go` の `watch` ヘルプ文、`internal/org/verbs.go` の Stop 理由コメント、`internal/org/watcher.go` のコメントから budget 言及を除去。
- `docs/tech-debt/README.md` の「`evaluateTotalBudget` の Cutoff ラチェット」行を obsolete としてクローズ。
- ドキュメント: spec `docs/specs/2026-08-01-org-runtime.md` の FR-2 / FR-7 / FR-8 / FR-10 / NFR「予算執行の決定論性」/ AC 2 件(wall-clock 遮断、fix ラウンド上限)/ Alternatives 表の「budget で統制」を改訂(履歴文書なので「2026-09-16 の判断で budget 概念を撤去」と注記する形にし、当時の記述は残す)。`docs/quality/quality-gates.md`(root + `templates/base/`)の Budget enforcement 行を削除。`/org` skill の「permission/budget 作法」節を「permission 作法」に改題し budget 記述を除去、`watch` 動詞の説明から budget 遮断を除去。

## Non-goals

- `max_fix_rounds` に代わる fix ラウンド執行機構の新設(lead の裁定に委ねる)。
- トークン量やコストに基づく新しい上限の導入。
- claude エイリアスのオフライン検証(CLI に一覧 API が無いため `--probe-models` に委ねる)。
- codex 座席の autonomous / edits permission mode の実機検証(`codex_verified` は据え置き)。
- `[org.roles]` の既定値追加(例のみ更新)。
- 標準フロー(`/spec` 〜 `/pr`)や `.claude/agents/` の変更。

## Assumptions

- codex のモデルスラッグは `~/.codex/models_cache.json` の `models[].slug` が正で、このマシン(codex-cli 0.149.1、2026-09-16)では `gpt-6-astra` / `gpt-5.6-sol` / `gpt-5.6-terra` / `gpt-5.6-luna` / `gpt-5.5` が `visibility: list` で存在する。`gpt-reserve` / `codex-auto-review` は `hide` のため既定に含めない。
- claude のエイリアス `fable` / `opus` / `sonnet` / `haiku` は `claude --help` の `--model` 説明に基づき有効(`haiku` は明示列挙されていないが既定プールで従来から使用中)。
- `DefaultModelForDriver` はプール先頭を返すため、claude の先頭を `fable` にすれば省略時フォールバックは `fable` になる(ユーザー確定)。
- `watchStatusFile.OrgStartTS` は budget 以外で参照されていない(grep で `evaluateTotalBudget` 以外の読み手なし)。実装時に再確認し、他の用途があれば残す。
- 座席内 fan-out は座席のドライバ機能に依存するため、ralph 側は雛形での許可と規約明文化のみで、機構的な制御は加えない。

## Affected areas

- `internal/config/config.go`, `config_test.go`, `defaults_sync_test.go`
- `internal/org/prompts/{implementer,reviewer,qa,lead}.md`, `prompts.go`(埋め込み対象は glob なので変更不要の見込み), `prompts_test.go`
- `internal/org/watch.go`, `watch_test.go`, `watcher.go`, `verbs.go`(コメント)
- `internal/cli/doctor.go`, `doctor_org_test.go`, `internal/cli/org.go`
- `templates/base/ralph.toml`, `scripts/ralph-config.sh`, `templates/base/scripts/ralph-config.sh`, `tests/test-ralph-config.sh`
- `.claude/skills/org/SKILL.md`(+ `.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` の 3 ミラーは `scripts/sync-skills.sh` / `check-sync.sh` で追従)
- `.claude/rules/ralph/model-routing.md`
- `docs/specs/2026-08-01-org-runtime.md`, `docs/quality/quality-gates.md`, `templates/base/docs/quality/quality-gates.md`, `docs/tech-debt/README.md`
- 触らない: `internal/org/spawn.go`(role の未知判定は雛形の有無で決まるため implementer.md 追加だけで有効化される)、`internal/org/protocol/`、`internal/upgrade/`

## Design decisions

- **D1 budget 概念の全面削除**(ユーザー確定、2026-09-16): 既定値の引き上げではなく `[org.budget]` と wall-clock 遮断を撤去する。理由: 上限が座席の生存時間基準で大規模開発の運用と合わない、`max_fix_rounds` は宣言のみで未執行、停滞・生存・スコープ外変更・デッドマンで安全側の監視は残る。
- **D2 `--model` は運用ルールで明示、機構は警告付きフォールバック**(ユーザー確定): fail-closed にはせず、省略時はプール先頭(claude は `fable`)へフォールバックし stderr で警告する。ルールは `/org` skill と lead 雛形に置く。
- **D3 プールの並び順**: claude は `fable, opus, sonnet, haiku`、codex は `gpt-6-astra, gpt-5.6-sol, gpt-5.6-terra, gpt-5.6-luna, gpt-5.5`(ユーザー指定順。先頭が省略時既定)。
- **D4 codex スラッグ検証はキャッシュ読み取りのみ**: `--probe-models` と別 Check にし、キャッシュが無い環境では `info` スキップ。キャッシュは codex が更新する best-effort データなので `fail` ではなく `warn` に留める。
- **D5 fan-out は座席内部の裁量**: ralph の座席機構(manifest / max_seats / スター型)は子サブエージェントを認識しない。雛形で「子は lead に送らない、成果は座席本体が RESULT に集約する」と規定する。
- **D6 Codex 計画アドバイザリ反映**(2026-09-16、ユーザー承認): (1) budget 撤去は config と watchdog を 1 スライスに統合、(2) 旧 watch-status の budget 系エントリを読み込み時に prune、(3) `spawn` の `--model` 必須チェックを外し `start` と共通ヘルパーで警告付きフォールバックに統一。
- Critical forks: budget の扱いと省略時既定モデルの 2 点を AskUserQuestion で解決済み(上記 D1 / D2)。

## Acceptance criteria

- [x] AC-1: `ralph org spawn --role implementer` が埋め込み雛形を展開して起動できる(`RenderRolePrompt("implementer", ...)` が ok=true、全プレースホルダ置換済み)。
- [x] AC-2: `implementer.md` / `reviewer.md` / `qa.md` が「座席内 fan-out」節を持ち、子サブエージェントが lead へ直接送信しない旨とスター型不変を明記している(`prompts_test.go` で 3 雛形の文言を回帰チェック)。`lead.md` の委譲先が implementer / reviewer / qa になっている。
- [x] AC-3: `config.Default().Org.ModelPool` が D3 の 9 エントリ・並び順で、`templates/base/ralph.toml` と `scripts/ralph-config.sh` 両面と `defaults_sync_test` でロックステップ一致。`DefaultModelForDriver(cfg, "claude")` が `fable`、`"codex"` が `gpt-6-astra` を返す。
- [x] AC-4: `ralph doctor` に codex スラッグ Check があり、fixture テストで (a) 全スラッグ一致 → pass、(b) 未知スラッグ → warn に当該スラッグ名、(c) キャッシュ無し → info スキップ、の 3 経路が通る。`--probe-models` の既存挙動は不変。
- [x] AC-5: `ralph org spawn` / `start` の両方で `--model` 省略時にエラーにならず、stderr に警告 1 行を出してプール先頭へフォールバックする(`spawn` の required ループから `--model` が外れ、ヘルプ文が更新されている)。CLI テストで spawn / start それぞれの省略経路を確認。`/org` skill に `--model` 明示ルールが記載されている。
- [x] AC-6: `[org.budget]` セクション、`OrgBudgetConfig`、`RALPH_ORG_{SEAT,TOTAL}_BUDGET_MINUTES` / `RALPH_ORG_MAX_FIX_ROUNDS` が 3 面から消え、`grep -rn -i 'budget' internal/ scripts/ templates/base/ralph.toml templates/base/scripts .claude/skills/org docs/quality` が identifier.go の「32 文字 budget」と watch.go の `scopeChangeBodyBudget`(いずれも別概念)以外ゼロ。`[org.budget]` を含む旧 ralph.toml を `Load()` すると未知セクションとして無視される(エラーにしない)ことをテストで確認。
- [x] AC-7: `ralph org watch` が budget 遮断なしで stall / 生存 / スコープ外変更 / デッドマンを従来通り検知する(`watch_test.go` の非 budget テストが全て pass、`Cutoff` / `OrgStartTS` 参照ゼロ)。
- [x] AC-7b: budget 系の `Conditions` / `PendingAlerts` / `Escalated` を含む旧 watch-status fixture を読み込んだ 1 サイクルで、当該エントリが prune され、デッドマン人間エスカレーションが発火しないことを回帰テストで確認。
- [x] AC-8: spec / quality-gates / tech-debt / `/org` skill / model-routing の記述が新状態と整合し、`./scripts/check-sync.sh` / `check-skill-sync.sh` / `check-pipeline-sync.sh` / `test-no-loop-references.sh` が green。
- [x] AC-9: `go build ./...` / `go vet ./...` / `go test ./...` / `./scripts/run-verify.sh` が green。

## Implementation outline

1. **Slice 1 — budget 概念の一括撤去**(Codex 所見 1 により config 側と watchdog 側を 1 スライスに統合。config から `Budget` を消すと `watch.go` の `w.cfg.Budget` 参照が残る中間状態はコンパイルできないため): `config.go` / `templates/base/ralph.toml` / `ralph-config.sh` 両面 / `tests/test-ralph-config.sh` の `[org.budget]` 削除、`config_test.go` / `defaults_sync_test.go` 追従、旧 `[org.budget]` 互換テスト、`watch.go` の 2 評価関数 / 条件キー / `Cutoff` / `OrgStartTS` 削除と `pruneRetiredConditions` 追加、`watch_test.go` の budget テスト削除と旧 fixture 回帰テスト、`org.go` watch ヘルプ・`verbs.go`・`watcher.go` コメント更新、tech-debt 行クローズ。(AC-6, AC-7, AC-7b)
2. **Slice 2 — model_pool 既定の刷新**: `config.go` `Default()` / `templates/base/ralph.toml` / `ralph-config.sh` 両面の 3 面同時更新、`config_test.go` / `defaults_sync_test.go` 追従、`DefaultModelForDriver` の先頭確認テスト。(AC-3)
3. **Slice 3 — doctor codex スラッグ Check + `--model` フォールバック統一**: `doctor.go` に Check 追加と fixture テスト、`org.go` の `resolveModelOrWarn` ヘルパー導入・spawn の required ループから `--model` 除外・ヘルプ文更新・spawn / start の省略時 CLI テスト。(AC-4, AC-5 機構側)
4. **Slice 4 — 役割雛形**: `implementer.md` 新設、3 雛形に fan-out 節、`lead.md` 改訂、`prompts_test.go` 追加。(AC-1, AC-2)
5. **Slice 5 — ドキュメント**: `/org` skill 改訂 → `sync-skills.sh` でミラー再生成、`templates/base/ralph.toml` の `[org.roles]` 例とコメント、spec 注記、quality-gates 両面、model-routing。(AC-5 ルール側, AC-8)

各スライスは `./scripts/run-verify.sh` 通過後に Conventional Commits で個別コミット。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(go vet / gofmt / shell lint)、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-template-purity.sh`、`./scripts/check-pipeline-sync.sh`、`tests/test-no-loop-references.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-9。spec FR-2 / FR-8 / NFR の改訂が実装と一致。
- Documentation drift to check: `/org` skill 4 面、`ralph.toml` テンプレのコメント、`docs/quality/quality-gates.md` 両面、`docs/tech-debt/README.md`、`.claude/rules/ralph/model-routing.md`、README の org runtime 節(budget 言及なしを確認)。
- Evidence to capture: `grep -rn -i budget` の結果、`ralph doctor` の Check 出力(pass / warn / skip)、`ralph org spawn --dry-run --role implementer` の出力、`docs/reports/verify-*.md`。

## Test plan

- Unit tests: `config_test.go`(新既定プール、`[org.budget]` 無視、`DefaultModelForDriver` 先頭)、`defaults_sync_test.go`、`prompts_test.go`(implementer レンダリング、fan-out 文言 3 雛形、lead 委譲先)、`doctor_org_test.go`(codex スラッグ 3 経路)、`watch_test.go`(残存テスト全 pass)。
- Integration tests: `ralph org spawn --dry-run --role implementer --driver claude --model fable` が manifest に `spawned` を記録する(既存 dry-run 経路)。`--model` 省略で stderr 警告。
- Regression tests: `go test ./...`、`tests/*.sh` 全件、`./scripts/run-test.sh`。
- Edge cases: `[org.budget]` を含む旧 ralph.toml、`models_cache.json` が壊れた JSON(warn ではなく info スキップ + 理由)、`CODEX_HOME` 上書き、codex がプールに無い場合の Check スキップ、`--model` 省略 + codex driver(先頭 `gpt-6-astra`)。
- Evidence to capture: `go test ./... -count=1` 出力、`docs/reports/test-*.md`。

## Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| `watch_test.go`(96k)の budget テスト削除で周辺テストの前提(`Cutoff` / `OrgStartTS`)が壊れる | Slice 1 の手戻り | 削除前に `Cutoff` / `OrgStartTS` の参照を grep で全列挙し、非 budget テストが依存していれば先に置き換える |
| Slice 1 が大きく(config + watch + tests + docs コメント)、途中で止まると中間状態がコンパイル不能 | 検証済みコミットの規約が守れない | Slice 1 内では `watch.go` の参照除去 → config 削除の順で編集し、`go build ./...` が通ってから 1 コミットにまとめる。停止時は `wip:` チェックポイントではなく revert で戻す |
| 旧 watch-status に残った budget ALERT が誤エスカレーションする | 廃止済み制御で人間が呼ばれる | `pruneRetiredConditions` + 旧 fixture 回帰テスト(AC-7b) |
| 下流プロジェクトの `ralph.toml` に `[org.budget]` が残る | `ralph upgrade` 後に Load エラー | 未知セクションは TOML デコードで無視されることをテストで固定(AC-6) |
| codex スラッグはモデル更新で陳腐化する | 既定プールが起動不能に | doctor Check で早期検知。既定は `ralph upgrade` で配布。spec に「既定は CLI ネイティブ名で、更新時は 3 面同時変更」と明記 |
| `fable` 先頭により省略時コストが上がる | 想定外の高コスト起動 | stderr 警告 + skill ルールで明示指定を促す(ユーザー確定) |
| 座席内 fan-out が典型的な役割逸脱(子が直接 lead へ送る)を招く | スター型の崩れ | 雛形で禁止を明記。機構的には agmsg identity が座席単位なので子は別 identity を持てない |
| spec の履歴記述と現状の乖離 | 読み手の混乱 | FR 本文に「2026-09-16 改訂」注記を付け、当時の判断は残す |

## Rollout or rollback notes

- 破壊的変更: `[org.budget]` と対応 env 3 つの削除、`ralph org watch` の wall-clock 遮断の廃止。PR body に明記し、`ralph upgrade` の upgrade report で `ralph.toml` 差分として下流に見える。
- ロールバック: 単一 PR の revert で戻る。manifest / receipts のスキーマは変更しないため状態ファイルの互換性は影響なし。`watch-status-<org_id>.json` は前方互換で読み、`Cutoff` / `org_start_ts` は読み捨て、budget 系の `Conditions` / `PendingAlerts` / `Escalated` エントリは読み込み時に prune する(AC-7b)。revert 後は prune 済みエントリが戻らないだけで、稼働中 org への影響はない。

## Open questions

- なし(budget と既定モデルの 2 点は解決済み)。

## Deviation notes (実装時)

- Slice 2 が `templates/base/ralph.toml` の `[org.roles]` 例とコメント更新(計画では Slice 5)を吸収した。同一ファイルの二重編集を避けるため。
- Slice 1 で `internal/cli/doctor_org_test.go` は編集不要だった(`Budget:` リテラルなし。"ContextBudget" はプローブ timeout の別概念)。
- Slice 1 で deadman 系テスト 2 件は削除せず、名前から "Cutoff" を外して残した(watchdog 自身の Stop を lead activity に数えない挙動は budget と独立)。
- Slice 3 の `--model` 省略時警告の文言アサートは spawn テストのみ。start は同一ヘルパー経由で、既存の `TestOrgStart_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` がフォールバック値を検証する。
- Slice 5: `docs/quality/quality-gates.md` と `.claude/rules/ralph/model-routing.md` の root/template 間に既存の KNOWN_DIFF があり、両面へ同旨の編集を当てて drift は増やしていない。

## Commits

- 7b6a720 docs: add plan
- ef3dce4 refactor: remove org budget concept and watchdog wall-clock cutoff (Slice 1)
- 3f9b4a0 feat: refresh default org model_pool (Slice 2)
- b901364 feat: doctor codex model-slug check and unified --model fallback warning (Slice 3)
- 185920c feat: add implementer seat template and in-seat fan-out guidance (Slice 4)
- f729a15 docs: align org runtime docs (Slice 5)

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
