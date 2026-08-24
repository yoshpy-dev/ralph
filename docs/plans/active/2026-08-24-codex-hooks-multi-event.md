# codex-hooks-multi-event

- Status: Active
- Owner: Claude Code
- Date: 2026-08-24
- Related request: Codex 配布 hooks 配線の multi-event 化(#149 で「PostToolUse 以外は別タスク」と明示的に繰り延べた項目の回収)
- Related issue: N/A
- Type: feat
- Branch: feat/codex-hooks-multi-event
- Canonical ref: docs/plans/archive/2026-08-20-codex-hooks-json-wiring.md(Non-goals)+ 2026-08-24 のメンテナ環境実測(ローカル差分 15 ファイルの根本原因分析)

## Objective

配布 `.codex/hooks.json` を PostToolUse 単独から multi-event(PreToolUse / SessionStart / SessionEnd / PreCompact / UserPromptSubmit を追加)へ拡張し、Codex 側の hooks パリティを Claude Code 側(7 イベント配線済み)と揃える。これによりメンテナ/下流環境が hooks.json をローカル改変して補填する必要を無くす。

## 背景(実測済みの前提)

- Claude Code 側は `.claude/settings.json` で 7 イベントすべてが `ralph-dispatch.sh <event>` 配線済みで、各 `.claude/hooks/<event>.d/` に shim(10-pre-bash-guard.sh 等)が配置済み。**Codex 側に足りないのは hooks.json のエントリだけ**。
- dispatcher は stdin 無解釈透過+cwd 相対 `.d` 解決。#149 で確立した command 形(`cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh <event>`)をイベント名だけ変えて流用する。
- Codex の公式イベント一覧(learn.chatgpt.com/docs/hooks、2026-08-20 取得): SessionStart / SessionEnd / PreToolUse / PostToolUse / PermissionRequest / PreCompact / PostCompact / UserPromptSubmit / SubagentStart / SubagentStop / Stop。**PostToolUseFailure は存在しない**(Claude Code 固有 — 配線対象外として明記する)。
- SessionStart / UserPromptSubmit / Stop は codex-cli 0.147.0 実機で発火実績あり(2026-08-19/20 の live-fire ログに hook 行)。PreToolUse も hooks.json 経由の発火実績あり(ユーザローカル構成)。
- matcher 意味論: PreToolUse/PostToolUse はツール名への正規表現。SessionStart は `startup|resume|clear|compact`、PreCompact は `manual|auto`、SessionEnd は reason。matcher 省略/`""`/`"*"` は全一致。

## Scope

1. **`.codex/hooks.json`(root + templates/base、byte-identical)への 5 イベント追加**:
   - PreToolUse: matcher `Bash`(pre-bash-guard の対象。Codex のシェルツール名が `Bash` として matcher 照合されることは実機で発火実績があるが、Slice 1 で payload の実ツール名を再確認し、必要なら `Bash|shell|exec_command` 等へ拡張)
   - SessionStart: matcher 省略(全 source)
   - SessionEnd: matcher 省略
   - PreCompact: matcher 省略(`manual|auto` 全対象)
   - UserPromptSubmit: matcher 省略
   - command はすべて `cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh <event>` 形。
2. **decision-JSON 契約の実機確認(Slice 1)**: pre_bash_guard(PreToolUse deny/ask)と prompt_gate(UserPromptSubmit)の decision 出力が Codex 側で尊重されるかを live-fire で確認。尊重されない場合も hook 自体は無害(警告・記録として機能)— 結果を配布ドキュメントに事実として明記する。
3. **shim スクリプトのペイロード互換確認**: 各 `.d` shim が呼ぶ本体(pre_bash_guard / session_start_context / session_end_summary / precompact_checkpoint / prompt_gate)の入力抽出が Codex ペイロードで機能するか確認。#149 で確立した `tool_input.command` / `cwd` 抽出パターンの類例が必要なら同様に拡張(PreToolUse の Bash は `tool_input.command` トップ想定 — 実測で確定)。
4. **doctor / テスト追随**: `checkCodexEffectiveConfig` の dispatcher ルーティング検査を「PostToolUse に 1 件以上」から「配布対象イベント集合の各イベントに dispatcher ルーティングが存在」へ拡張(欠落イベントは warn)。`tests/test-hook-wiring.sh` の期待も同様に拡張。
5. **ドキュメント**: `.codex/README.md` / `.codex/hooks/README.md` / recipes のイベント一覧更新。PostToolUseFailure が Codex 非対応である旨と、trust 再承認(エントリ追加 = 新ハッシュ)の注意を明記。
6. **live-fire 検証**: 信頼済みメタリポ+fresh `ralph init` fixture で、追加イベントの発火(最低限 PreToolUse / SessionStart / UserPromptSubmit)と dispatcher 第 3 層到達を bypass 付きで実証。証跡を docs/evidence/ に保存。

## Non-goals

- PermissionRequest / PostCompact / SubagentStart / SubagentStop / Stop の配線(Claude 側にも対応する `.d` が無い。必要になった時点で別タスク)
- PostToolUseFailure の Codex 配線(イベントが存在しない — ドキュメント明記のみ)
- gh_account 系などメンテナ個人グルーの同梱(マージ後にローカルの `.claude/hooks/local/<event>.d/` へ移設する — 本プランのスコープ外のローカル作業)
- `shell_environment_policy` のユーザレベル移動(同上、ローカル作業)
- dispatcher 本体・shim・hook 本体スクリプトの機能変更(ペイロード互換で必要になる抽出拡張を除く)

## Assumptions

- Codex は同一イベント名(PreToolUse 等)を hooks.json で受理する(公式一覧+実機発火実績)。SessionEnd/PreCompact は発火実績が未取得のため Slice 1 で確認し、発火しないイベントがあれば「配線はするが現行 CLI では発火しない」旨をドキュメント化して同梱する(将来バージョンでの互換を優先)か、外すかを実測結果で決める。
- trust は per-command-hash のため、既存 PostToolUse エントリのハッシュは不変(command 文字列を変えない)。追加 5 エントリのみ新規承認が必要。

## Affected areas

- `.codex/hooks.json` + `templates/base/.codex/hooks.json`
- `.claude/hooks/` の hook 本体(ペイロード互換で抽出拡張が必要な場合のみ)+ twins
- `internal/cli/doctor.go`(ルーティング検査の集合化)+ doctor テスト
- `tests/test-hook-wiring.sh`
- `.codex/README.md` / `.codex/hooks/README.md` / `docs/recipes/codex-setup.md` + twins
- `docs/evidence/`(live-fire)

## Acceptance criteria

- [ ] AC-1 `.codex/hooks.json`(両コピー byte-identical)が 6 イベント(PostToolUse + 追加 5)を dispatcher 経由で配線し、既存 PostToolUse エントリの command 文字列は不変(trust ハッシュ維持)。
- [ ] AC-2 live-fire(信頼済みメタリポ、bypass 付き): PreToolUse(Bash 実行)・SessionStart・UserPromptSubmit の 3 イベントで dispatcher → 第 3 層 drop-in の実行を実証。SessionEnd / PreCompact は発火可否を実測記録(発火しない場合はドキュメント化判断を Design decisions に追記)。証跡を docs/evidence/ に保存。
- [ ] AC-3 fresh `ralph init` fixture でも追加イベントの発火を確認(最低 1 イベント、bypass 付き)。
- [ ] AC-4 pre_bash_guard が Codex の Bash 実行ペイロードからコマンドを抽出できる(実ペイロード fixture テスト。deny 判定の decision-JSON が Codex で尊重されるかの実測結果を evidence に記録 — 尊重されない場合も配線は維持し事実を明記)。
- [ ] AC-5 doctor が配布対象イベント集合の各イベントについて dispatcher ルーティングの存在を検査し、欠落を warn する(negative テスト付き)。既存の検査群(スキーマ/直接参照/併存/features.hooks)は不変 green。
- [ ] AC-6 `tests/test-hook-wiring.sh` が 6 イベントの配線を検査する。
- [ ] AC-7 ドキュメント更新(イベント一覧、PostToolUseFailure 非対応、trust 再承認の注意)。
- [ ] AC-8 `./scripts/run-verify.sh` exit 0、check-sync / purity / 全テスト green。

## Implementation outline

1. **Slice 1: 実測(書き捨て)** — このメタリポ(trusted)で 5 イベントを仮配線し、イベントごとの発火可否・ペイロード形(ツール名、cwd、抽出対象フィールド)・decision-JSON の尊重有無を capture。配布形(matcher 含む)を確定し、プランの Assumptions を更新。
2. **Slice 2: 配線+消費者追随(原子的 1 コミット)** — hooks.json 6 イベント化(両コピー)、必要なペイロード抽出拡張(hook 本体+twins)、doctor 検査集合化+テスト、test-hook-wiring 拡張、docs 更新、fixture live-fire 証跡。

## Verify plan

- 静的解析: `./scripts/run-static-verify.sh`(check-sync / purity 含む)
- 仕様整合: 公式イベント一覧との突合、#149 の確立事項(cd-first、trust UX)との無矛盾
- 証跡: docs/evidence/(イベント別 live-fire)

## Test plan

- 単体: doctor のイベント集合検査(欠落 warn ×各イベント、全配線 pass)、ペイロード抽出拡張の fixture(拡張した場合)
- 統合: live-fire(AC-2/3)
- 回帰: 既存 653 shell + Go 全パッケージ、既存 doctor 検査の不変 green
- エッジ: matcher 省略イベントの発火、Bash 以外のツールで PreToolUse ガードが誤発火しないこと(matcher で絞る場合)

## Risk register

| リスク | 影響 | 緩和 |
|---|---|---|
| SessionEnd / PreCompact が 0.147.0 で発火しない | 配線が dead entry になる | Slice 1 で実測し、発火しない場合は「将来互換のため配線+現状非発火を明記」か「外す」を実測に基づき決定(Design decisions に記録) |
| pre_bash_guard の deny が Codex で無効 | ガードが警告どまり | 事実を evidence とドキュメントに明記。配線自体は記録・警告として有効 |
| Bash 以外への PreToolUse 誤適用 | 無関係ツールで guard 実行 | matcher をツール名で絞る(Slice 1 で実ツール名確定) |
| trust 再承認の下流負担 | 追加 5 エントリの対話承認 | 既存 PostToolUse の command を不変に保ちハッシュ維持。README に承認手順明記 |
| イベント増による hook 実行オーバーヘッド | セッション操作ごとの数十 ms | shim は軽量(既存 Claude 側で常用実績)。問題なし |

## Rollout or rollback notes

- 加算的(エントリ追加+検査拡張)。ロールバックは追加エントリ削除+検査期待の revert で単純
- マージ後のローカル作業(本プラン外): gh_account 系を `.claude/hooks/local/<event>.d/` へ移設、`.codex/hooks/` コピー群の削除、`shell_environment_policy` のユーザレベル移動 → メンテナ環境の main 差分ゼロ化

## Design decisions

- **イベント集合は Claude 側 `.d` 実在分に限定**(PostToolUseFailure を除く 6 イベント): shim が既に存在し実行実績があるものだけを配る。`.d` の無いイベントの新設は別タスク。
- **command 形は #149 確立形の踏襲**(cd-first)。既存 PostToolUse エントリは 1 文字も変えない(trust ハッシュ維持)。
- Critical forks: None(未確定点は Slice 1 の実測で決まり、いずれも 1 スライス未満の手戻り)

## Open questions

- SessionEnd / PreCompact の 0.147.0 実発火可否(Slice 1)
- Codex の Bash 系ツールの実ツール名と PreToolUse ペイロード形(Slice 1)
- decision-JSON(deny/ask)の Codex 側尊重有無(Slice 1)

## Evidence targets

- `docs/reports/self-review-2026-08-24-codex-hooks-multi-event.md`
- `docs/reports/verify-2026-08-24-codex-hooks-multi-event.md`
- `docs/reports/test-2026-08-24-codex-hooks-multi-event.md`
- `docs/reports/sync-docs-2026-08-24-codex-hooks-multi-event.md`
- `docs/reports/cross-review-triage-codex-hooks-multi-event.md`
- `docs/evidence/`(イベント別 live-fire)

## Progress checklist

- [ ] Slice 1: 実測(発火可否・ペイロード・decision 尊重)
- [ ] Slice 2: 配線+消費者追随(原子的)
- [ ] Post-implementation pipeline
- [ ] /pr

## Readiness checklist

- [x] Claude 側 7 イベント配線+`.d` shim の実在を確認済み(2026-08-24)
- [x] 公式イベント一覧と実機発火実績の突合済み(PostToolUseFailure 非対応を特定)
- [x] 未確定 3 点は Slice 1 の実測で確定する段取り
