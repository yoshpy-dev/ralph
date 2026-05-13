# fix-codex-review-wording-residues

- Status: Implementation complete (awaiting post-implementation pipeline)
- Owner: Claude Code
- Date: 2026-05-13
- Related request: 3.5.0 `/codex-review` → `/cross-review` リネーム後に残った "codex review" / "codex ACTION_REQUIRED" 文字列残骸の清掃
- Related issue: 51
- Branch: fix/51/codex-review-wording-residues

## Objective

`/codex-review` → `/cross-review` リネーム（3.5.0）以降も残存している 3 種類の文字列残骸を、トップレベルと `templates/base/` ミラーの計 6 ファイルで一括清掃し、`ralph upgrade` を経由した下流プロジェクトでも整合した命名を提供する。

## Scope

- 以下 6 ファイルの string-only リネームのみ：
  - `docs/recipes/ralph-loop.md` (line 316: `codex ACTION_REQUIRED` → `cross-review ACTION_REQUIRED`)
  - `.agents/skills/loop/prompts/pipeline-outer.md` (lines 4, 76: `codex review` → `cross-review`)
  - `.claude/skills/loop/prompts/pipeline-outer.md` (lines 4, 76: 同上)
  - `templates/base/docs/recipes/ralph-loop.md` (line 316: 同上)
  - `templates/base/.agents/skills/loop/prompts/pipeline-outer.md` (lines 4, 76: 同上)
  - `templates/base/.claude/skills/loop/prompts/pipeline-outer.md` (lines 4, 76: 同上)

## Non-goals

- 挙動変更は一切なし。コードパス、シェルスクリプト、設定値、テストは無変更。
- 以下は意図的に履歴として残してある残存箇所であり、本修正では一切触らない（後述 AC-5 の allowlist）：
  - `docs/plans/archive/` 配下のアーカイブ済みプラン
  - `docs/reports/` 配下の不変履歴アーティファクト
  - `docs/recipes/codex-setup.md`（および `templates/base/` のミラー）の "codex-review → cross-review rename" 記述
  - `docs/specs/2026-05-07-codex-cli-parity.md` のスペック本文（リネーム作業そのものの spec）
  - `docs/tech-debt/README.md` の RESOLVED 取り消し線付きエントリ（履歴記述）
- `_codex_log` / `_has_codex` などの実装内部変数のリネーム（issue #44 / Loop driver 化で別途完了済み）。

## Assumptions

- トップレベルと `templates/base/` ミラーは `scripts/check-sync.sh` で同期検査されており、双方を同時に修正する必要がある。
- AC-9 allowlist（`docs/plans/archive/`、`docs/reports/`、`docs/recipes/codex-setup.md`）は既存プランで確立済みで、本修正でも踏襲する。

## Affected areas

- `docs/recipes/ralph-loop.md` および `templates/base/docs/recipes/ralph-loop.md`
- `.agents/skills/loop/prompts/pipeline-outer.md` および `templates/base/.agents/skills/loop/prompts/pipeline-outer.md`
- `.claude/skills/loop/prompts/pipeline-outer.md` および `templates/base/.claude/skills/loop/prompts/pipeline-outer.md`

## Design decisions

Critical forks: None — issue 提示の修正方針（string-only replacement）が一意で、許容リストも既存プラン（`docs/plans/archive/2026-05-07-codex-cli-parity.md` の AC-9）で確定済み。代替設計の余地なし。

## Acceptance criteria

- [x] AC-1: `docs/recipes/ralph-loop.md` line 316 が `→ if cross-review ACTION_REQUIRED: regress to Inner Loop` になっている
- [x] AC-2: `.agents/skills/loop/prompts/pipeline-outer.md` line 4 と 76 が `cross-review` 表記に統一されている
- [x] AC-3: `.claude/skills/loop/prompts/pipeline-outer.md` line 4 と 76 が `cross-review` 表記に統一されている
- [x] AC-4: `templates/base/` 配下の 3 ファイルがトップレベルと byte-identical
- [x] AC-5: 以下の除外パスを除いた `git grep -nE 'codex[- ]review|codex ACTION_REQUIRED'` の残存ヒットが 0 件であること。
  - 除外パス（意図的履歴）:
    - `docs/plans/archive/`
    - `docs/reports/`
    - `docs/recipes/codex-setup.md` および `templates/base/docs/recipes/codex-setup.md`
    - `docs/specs/2026-05-07-codex-cli-parity.md`
    - `docs/tech-debt/README.md`
  - 検証コマンド: `git grep -nE 'codex[- ]review|codex ACTION_REQUIRED' -- ':!docs/plans/archive' ':!docs/reports' ':!docs/recipes/codex-setup.md' ':!templates/base/docs/recipes/codex-setup.md' ':!docs/specs/2026-05-07-codex-cli-parity.md' ':!docs/tech-debt/README.md'` の出力が空
- [x] AC-6: `./scripts/run-verify.sh` が green
- [x] AC-7: `./scripts/check-sync.sh` および `./scripts/check-skill-sync.sh` が pass

## Implementation outline

1. トップレベル 3 ファイルを編集（`Edit` ツールで対象行を厳密置換）
2. `templates/base/` ミラー 3 ファイルを同一内容で編集
3. `git grep` で残存をスイープ確認
4. `./scripts/check-sync.sh` および `./scripts/check-skill-sync.sh` で同期チェック
5. `./scripts/run-verify.sh` で総合検証
6. Conventional commit でコミット
7. Post-implementation pipeline 起動

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`
- Spec compliance criteria to confirm: AC-1〜AC-7 すべて
- Documentation drift to check: トップレベル ↔ `templates/base/` の byte-identical 性。他に "codex review" / "codex ACTION_REQUIRED" 文字列を参照しているドキュメントがないか。
- Evidence to capture: `git grep` 出力、`./scripts/run-verify.sh` 出力、`cmp` 結果

## Test plan

- Unit tests: 既存テストの green 維持確認（ロジック変更なし）
- Integration tests: なし（文書のみの修正）
- Regression tests: `./scripts/check-sync.sh` で template mirror parity を再確認
- Edge cases: 修正対象外の意図的残存箇所（`docs/recipes/codex-setup.md` 内のリネーム履歴記述、`scripts/check-sync.sh` 内の allowlist リテラル、archive、reports）が誤って書き換わっていないこと
- Evidence to capture: `./scripts/run-verify.sh` 出力、grep 出力の比較

## Risks and mitigations

- R-1: 過剰置換で意図的残存箇所まで書き換えてしまうリスク → mitigation: 厳密 `Edit` の old_string 指定で line-anchored 置換、最後に grep スイープで allowlist 整合確認。
- R-2: トップレベルとミラーで微妙にずれる → mitigation: 修正後に `./scripts/check-sync.sh` 実行で byte-identical 検証。

## Rollout or rollback notes

- Rollout: 単一コミットとして PR。マージ後の下流ユーザは次回 `ralph upgrade` で取得（upgrade engine は文書差分を auto-update として扱う）。
- Rollback: 単一コミットを revert すれば完全戻し可能。挙動変更がないため互換性影響なし。

## Open questions

なし（issue 自体が修正案を明示しており、ambiguity ゼロ）。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
