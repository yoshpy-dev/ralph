# codex-hooks-json-wiring

- Status: Active
- Owner: Claude Code
- Date: 2026-08-20
- Related request: v5.0.0 リリース前提として、配布 Codex hooks 配線を「実際に発火する表現」へ修正する
- Related issue: N/A
- Type: fix
- Branch: fix/codex-hooks-json-wiring
- Canonical ref: docs/tech-debt/README.md の配布配線ギャップ行(#148 で登録)+ 2026-08-19/20 の実機 live-fire 検証結果

## Objective

`.codex/config.toml` のインライン `[[hooks.*]]` 配線(codex-cli 0.147.0 実機で不発と実証済み)を、発火が実証済みの `hooks.json` 表現へ移行し、doctor / テスト / 同期ゲートを追随させる。overlay-scaffold-v2 系列の「Codex hooks dispatcher パリティ」を実態と一致させ、v5.0.0 リリースの前提を満たす。

## 実機検証で確定している事実(2026-08-19/20、codex-cli 0.147.0、信頼済みチェックアウト)

1. project レイヤのインライン TOML `[[hooks.*]]` は exec モードで**一切実行されない**(match あり/なし/matcher 形式/`--dangerously-bypass-hook-trust` 付きのすべてで実測)。
2. 同レイヤの `hooks.json` 表現は発火する: PostToolUse → `ralph-dispatch.sh` → 第 3 層 `.claude/hooks/local/PostToolUse.d/` まで実測確認(trust 承認後は bypass なしでも発火)。
3. Codex のファイル編集ツール名は **`apply_patch`**(`tool_input.command`)。hooks.json の `matcher = "Edit|Write|MultiEdit"` は `apply_patch` にも発火した(matcher 意味論はツール名の逐語一致ではない模様 — 要調査)。
4. 未信頼ハッシュの hook は exec で**無警告 skip**。persisted trust は対話セッションの承認のみ(per-command ハッシュが `~/.codex/config.toml` の `hooks.state` に記録される)。
5. 両表現の併存は「prefer a single representation」警告を出す。

## Scope

1. **配線移行**: `.codex/config.toml`(root + `templates/base/`)から `[[hooks.PostToolUse]]` エントリを撤去し、同レイヤに `hooks.json` を新設して `./.claude/hooks/ralph-dispatch.sh PostToolUse` へルーティングする。config.toml には「hooks は hooks.json 側」という参照コメントを残す。root と template は byte-identical 維持(check-sync)。
2. **matcher 調査と決定**: `matcher = "Edit|Write|MultiEdit"` が `apply_patch` に発火した意味論を一次情報(Codex hooks リファレンス)で確認し、`apply_patch` を明示に含めるか判断。判断根拠を hooks.json 冒頭コメント(JSON はコメント不可のため config.toml の参照コメントまたは `.codex/README.md`)に記録。
3. **相対パス検証**: 配布物はプロジェクト相対(`./.claude/hooks/ralph-dispatch.sh`)である必要がある。この信頼済みメタリポで相対パス command の実発火を live-fire で検証してから確定する。
4. **doctor 追随**: `checkCodexEffectiveConfig` の検査を反転 — hooks.json を hooks の source of truth として検査し(存在+有効 JSON+PostToolUse に dispatcher ルーティング 1 件以上)、逆に config.toml 側に `[[hooks.*]]` が残っていれば「両表現併存」警告として報告する。
5. **テスト追随**: `tests/test-hook-wiring.sh` の `check_codex_config_toml` を hooks.json 検査に置換(dispatcher 経由必須・レガシー直接呼び出し検出は維持)。
6. **ドキュメント/登記**: `.codex/README.md` / `docs/recipes/codex-setup.md`(あれば)に trust UX(初回対話セッションで hook 承認が必要。未承認の exec は無警告 skip)を明記。tech-debt の配布配線ギャップ行を RESOLVED 化、行 115 の stale な「trust 承認のみ残」文も更新。AGENTS.md / CLAUDE.md の `.codex` 記述の整合。
7. **リリース前提の確認**: 本 PR マージ後に `/release`(v5.0.0)へ進める状態であることを PR に明記。

## Non-goals

- Codex hooks の PostToolUse 以外のイベント(SessionStart 等)の新規配線追加(現行配布は PostToolUse のみ。拡張は別タスク)
- `~/.codex/config.toml` の trust テーブルへの書き込みやその自動化(対話同意ゲートの迂回はしない)
- インライン TOML hooks が実行されない原因の upstream 調査/報告(挙動事実として扱う)
- doctor `checkHooks` の Codex 側網羅検査(tech-debt 行 37 — 本タスクは checkCodexEffectiveConfig の追随のみ)

## Assumptions

- hooks.json の command は単一シェル文字列(実測形: `'<path>' PostToolUse`)。相対パスの可否は Slice 1 で実測確認する(不可なら `$CODEX_PROJECT_ROOT` 等の変数展開可否を調査、最悪 `sh -c 'cd "$(dirname "$0")" ...'` 形を検討)。
- 下流プロジェクトがローカルに `.codex/hooks.json` を持っている場合、追跡ファイル化により checkout/upgrade 時の衝突が起きうる(このメタリポ自身がその状態)。scaffold 側は新規ファイル追加なので `ralph upgrade` の通常経路(untracked 衝突は drift/advisory)で扱われる — 影響確認を verify plan に含める。
- trust は per-command-string ハッシュのため、配布 command 文字列を安定させれば下流の再承認は command 変更時のみ。

## Affected areas

- `.codex/config.toml` + `templates/base/.codex/config.toml`(hooks エントリ撤去+参照コメント)
- `.codex/hooks.json` + `templates/base/.codex/hooks.json`(新規、byte-identical)
- `internal/cli/doctor.go`(`checkCodexEffectiveConfig` 反転)+ 既存テスト(doctor_hooks_test.go / cli_test.go の該当 fixture)
- `tests/test-hook-wiring.sh`
- `scripts/check-sync.sh`(新ファイルの同期対象化 — file-set ベースなら自動、要確認)
- `internal/scaffold` manifest 生成(新配布ファイルの owner 付与: `ownerForScaffoldPath` — `.codex/hooks.json` は core)
- `.codex/README.md`、`docs/recipes/codex-setup.md`(あれば)、`AGENTS.md`/`CLAUDE.md` の該当記述、`docs/tech-debt/README.md`

## Acceptance criteria

- [x] AC-1 `.codex/hooks.json`(root + template、byte-identical)が PostToolUse を `ralph-dispatch.sh` 経由にルーティングし、`.codex/config.toml` に `[[hooks.*]]` エントリが残っていない(参照コメントは残る)。
- [x] AC-2 live-fire 実測(2 系統、証跡を docs/evidence/ に保存):
  - (a) このメタリポ(信頼済み)で配布と同一形(相対パス command)の hooks.json により、**bypass なしの** `codex exec` でファイル編集 → dispatcher → `.claude/hooks/local/PostToolUse.d/` の drop-in 実行まで発火(trust は 2026-08-20 に承認済みのものを利用。command 文字列が変わる場合は再承認が必要 — その場合は bypass 併記で代替し理由を記録)。
  - (b) fresh `ralph init` fixture(一時ディレクトリ)で、hooks.json が Codex に発見・ロードされる証跡(exec stderr の hook 関連出力/警告)+`--dangerously-bypass-hook-trust` での発火。**制約の明記**: fresh fixture の非 bypass 発火は project trust+hook trust の対話承認ゲートに阻まれ自動化不能(codex-cli 0.147.0)。bypass は CI 向けサプリメントであり、非 bypass の実証は (a) が担う — この妥協を evidence とレポートに明記する(Codex 所見 2)。
- [x] AC-3 matcher の決定が一次情報または実測に基づき記録されている(`apply_patch` を含む/含めない判断と根拠)。
- [x] AC-3b `.codex/hooks.json` の**型付きスキーマ検証**: doctor(または専用チェック)が公式スキーマ形(トップレベル `hooks` → イベント名キー → matcher グループ配列 → `{type: "command", command: <string>}` ハンドラ)への適合を検査する。ネガティブテスト必須: 有効な JSON だが不正なスキーマ(トップレベルに直接 `PostToolUse`、`hooks` キー欠落、ハンドラ `type` 欠落、command が配列)をそれぞれ warn として検出する(Codex 所見 1)。
- [x] AC-4 `ralph doctor` が hooks.json を source of truth として検査する: hooks.json 欠落/不正 JSON/スキーマ不適合(AC-3b)/dispatcher ルーティング欠落で warn(--strict 対象外の環境チェックのまま)、config.toml に hooks 残存で併存警告、**`[features].hooks = false` の明示無効化も warn**(欠落時は現行 Codex 既定に従う — Slice 1 で既定値を一次情報確認)(Codex 所見 3)。既存 doctor テストは新契約に追随して green。
- [x] AC-5 `tests/test-hook-wiring.sh` が hooks.json の dispatcher ルーティングを検査し、レガシー直接呼び出し形(config.toml または hooks.json のどちらに再導入されても)を検出したら fail する。
- [x] AC-6 `ralph init` が新規プロジェクトに `.codex/hooks.json` を ship し、manifest に owner=core で記録される(init テストで固定)。
- [x] AC-7 `scripts/check-sync.sh` green(hooks.json が root/template 同期対象に入る)。`./scripts/check-template-purity.sh` green。
- [x] AC-8 trust UX(初回対話承認が必要、未承認 exec は無警告 skip)が `.codex/README.md`(+recipe があれば)に記載されている。
- [x] AC-9 tech-debt: 配布配線ギャップ行を RESOLVED 化、行 115 の「trust 承認のみ残」文を実績(2026-08-20 承認済み・bypass なし発火確認済み)に更新。
- [x] AC-10 `./scripts/run-verify.sh` exit 0、shell + Go 全テスト green。

## Implementation outline

1. **Slice 1: live-fire 事前検証(書き捨て、コミットなし)** — このメタリポで相対パス command と matcher 変種(`apply_patch` 明示含む)の発火を実測し、配布形を確定。結果を docs/evidence/ に保存し、プランの Assumptions を確定値で更新。
2. **Slice 2: 配線移行+全消費者の追随(原子的 1 スライス = 1 コミット)** — hooks.json 新設(root/template)、config.toml から hooks 撤去+参照コメント、`checkCodexEffectiveConfig` 反転(スキーマ検証 AC-3b・features.hooks=false 検査を含む)+doctor テスト更新、`tests/test-hook-wiring.sh` 置換、check-sync/purity green 化、`ownerForScaffoldPath`/init 経路+テスト。配線と検査の source-of-truth を同一コミットで切り替え、中間状態で doctor/テストが壊れる区間を作らない(Codex 所見 4)(AC-1/3b/4/5/6/7)。
3. **Slice 3: ドキュメント/登記** — trust UX 記載、tech-debt 2 行更新、AGENTS/CLAUDE 整合(AC-8/9)。

## Verify plan

- 静的解析: `./scripts/run-static-verify.sh`(check-sync / check-template-purity / gofmt / golangci-lint 含む)
- スペック整合: overlay-scaffold-v2 スペックの AC-10 脚注・`.codex` 記述と矛盾しないこと
- upgrade 影響: 既存 v2 プロジェクト(hooks.json 未追跡ローカル所持を含む)への `ralph upgrade` 挙動 — 新 core ファイル追加の分類(untracked 衝突は drift)をテストで確認
- 証跡: docs/evidence/(live-fire ログ含む)

## Test plan

- 単体: doctor の新契約(hooks.json 欠落/不正/ルーティング欠落/併存警告)、init の hooks.json ship + manifest owner
- 統合: live-fire(AC-2、メタリポ実機)、`ralph upgrade` で既存プロジェクトに hooks.json が新規配布される経路(clean / untracked-collision 両方)
- 回帰: 既存 shell 617+ / Go 全パッケージ、test-hook-wiring の新検査
- エッジ: hooks.json が既にローカル存在(内容相違)のプロジェクトへの upgrade、config.toml に hooks が残った状態の doctor 併存警告

## Risk register

| リスク | 影響 | 緩和 |
|---|---|---|
| 相対パス command が Codex で解決されない | 配布形が成立しない | Slice 1 で最初に実測。不可なら変数展開/ラッパ形を調査し、プラン更新後に続行 |
| matcher 意味論の誤解(apply_patch 発火の理由不明) | 将来バージョンで発火しなくなる | 一次情報確認+`apply_patch` 明示を含める保守的既定。根拠を記録 |
| 下流のローカル hooks.json との衝突 | upgrade 時 drift/衝突 | 既存の untracked-collision 分類(drift + eject/adopt 案内)で処理されることをテストで確認し、README に注記 |
| doctor 反転による既存プロジェクトの警告発生 | 移行期ノイズ | 併存は warn(fail ではない)。メッセージに移行手順を明記 |
| trust 再承認の下流負担 | 初回対話承認が必要 | command 文字列を安定化。README に UX 明記 |

## Rollout or rollback notes

- ロールバック手順(Codex 所見 4 を受け具体化): (1) root/template の hooks.json 削除、(2) config.toml の hooks エントリ復元(両コピー)、(3) doctor/`test-hook-wiring.sh` の検査反転を戻す、(4) init/manifest テストの期待値復元、(5) tech-debt/README 記載の巻き戻し — Slice 2 が原子的 1 コミットなので実質 `git revert` 1 発+ドキュメントスライスの revert で完結する
- マージ後、`/release` で v5.0.0 を発行する(本タスクがリリース前提)

## Deviations

- Self-review (report `docs/reports/self-review-2026-08-20-codex-hooks-json-wiring.md`) found 11 findings (0 CRITICAL / 1 HIGH / 5 MEDIUM / 5 LOW: H1, M1-M5, L1-L5), all fixed in a single follow-up commit `c72e644` rather than a plan-scope change — touched `.codex/README.md`, `.codex/config.toml`, `.codex/hooks/README.md`, `docs/recipes/codex-setup.md`, `internal/cli/doctor.go`, `internal/cli/cli_test.go`, `scripts/verify.local.sh`, `tests/test-hook-wiring.sh` (+ template twins).
- A follow-up commit `7af720a` updated `.codex/AGENTS.override.md`'s hooks row (root + template) for the hooks.json wiring; this file is outside the plan's `Affected areas` list because it was caught during the self-review fix pass, not planned up front.
- AC-2(b) live-fire evidence came out stronger than the plan anticipated: the fresh, *untrusted* `ralph init` fixture fired under `--dangerously-bypass-hook-trust`, not just the pre-trusted meta-repo path in AC-2(a). The non-bypass constraint documented in the plan's Assumptions/Risk register held only for the fresh-fixture case, as expected.
- Cycle 2 (post-cross-review re-run): cross-review AR#1 (`checkCodexEffectiveConfig` silently ignored a non-boolean `[features] hooks` value) was fixed inline in `d1df46f` under the trivial-edit exception rather than a dedicated implementer slice. Self-review cycle 2 then found 7 new findings (0 CRITICAL / 0 HIGH / 2 MEDIUM / 5 LOW: C2-M1, C2-M2, C2-L1–L5), all fixed in a single follow-up commit `bced11a` — touched `.codex/README.md`, `.codex/config.toml`, `.codex/hooks/README.md`, `docs/recipes/codex-setup.md`, `docs/specs/2026-08-17-overlay-scaffold-v2.md`, `docs/tech-debt/README.md`, `internal/cli/doctor.go` (+ template twins). The verifier's cycle-2 pass then caught one residual orphaned sentence fragment in `.codex/config.toml`'s hooks comment ("`ralph doctor` and the shell / flags that...", left dangling by the C2-L3 wording fix in `bced11a`), fixed in `67f56d5`.
- Pipeline cycle 3(操作者承認で cap 2→3): cross-review cycle-2 の AR 2 件(サブディレクトリ起動での dispatcher 空回り / apply_patch ペイロード不適合)を 4d8220c で修正し、サブディレクトリ起動 live-fire で再実証(docs/evidence/codex-hooks-livefire-cycle3-2026-08-20.md)。self-review cycle-3 が両修正の干渉(C3-H1: envelope パスがセッション cwd 相対のため git-root 実行下で mojibake 存在チェックが外れる)を検出し、payload の cwd フィールドで解決する修正を同 cycle 内で実施。

## Design decisions

- **表現の一本化(hooks.json 側)**: 実機で実行される表現が hooks.json のみと実証されたため。両表現併存は警告が出るうえ二重管理になるため、config.toml 側は撤去し参照コメントのみ残す。
- **doctor は環境チェックのまま(--strict 非対象)**: checkCodexEffectiveConfig は従来どおり環境系(FR-9 スコープ外)。強度は変えず検査対象だけ反転する。
- Critical forks: None(表現選択は実測事実から一意。matcher/相対パスは Slice 1 の実測で確定する運用)

## Open questions

すべて Slice 1 で解決(証跡: docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md):
- ~~相対パス command の可否~~ → 素の相対パスは公式に非推奨(hook cwd はセッション cwd)。**確定形(cycle 3 で更新): `cd "$(git rev parse --show-toplevel 相当)" && ./.claude/hooks/ralph-dispatch.sh PostToolUse` — git root へ cd してから起動する形**(Slice 1 の git-root パス解決形は cycle-2 cross-review AR#1 でサブディレクトリ起動時に dispatcher の cwd 相対 `.d` 解決が空回りすると判明し、cd 前置形へ更新。シェル評価の実測と発火実証は両形で取得済み — evidence 2 通参照)
- ~~matcher 意味論~~ → 一次情報: matcher はツール名への正規表現。ファイル編集の tool_name は `apply_patch`(`Edit`/`Write` はエイリアスとして受理)。**確定 matcher: `Edit|Write|MultiEdit|apply_patch`**(発火実証済み)
- ~~`[features].hooks` 欠落時の既定~~ → 公式ドキュメントに記載なし。doctor は欠落を許容し、明示 `false` のみ warn とする

## Evidence targets

- `docs/reports/self-review-2026-08-20-codex-hooks-json-wiring.md`
- `docs/reports/verify-2026-08-20-codex-hooks-json-wiring.md`
- `docs/reports/test-2026-08-20-codex-hooks-json-wiring.md`
- `docs/reports/sync-docs-2026-08-20-codex-hooks-json-wiring.md`
- `docs/reports/cross-review-triage-codex-hooks-json-wiring.md`
- `docs/evidence/`(live-fire ログ)

## Progress checklist

- [x] Slice 1: live-fire 事前検証(確定形: git-root 解決 command + apply_patch 込み matcher — evidence 参照)
- [x] Slice 2: 配線移行+全消費者追随(6717ff6)
- [x] Slice 3: ドキュメント/登記(2bda3a1)
- [ ] Post-implementation pipeline
- [ ] /pr

## Readiness checklist

- [x] 実機検証の確定事実を列挙済み(本文冒頭)
- [x] 影響面(doctor 反転/テスト/同期/manifest owner)を特定済み
- [x] 未確定 2 点(相対パス・matcher)は Slice 1 の実測で確定する段取り
