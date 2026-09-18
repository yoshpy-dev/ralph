# org-codex-slug-update-procedure

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-18
- Related request: 既定 `[org].model_pool` の codex スラッグ(`gpt-6-astra` 等)は codex 側のモデル更新で消えうる。陳腐化の検知は `ralph doctor` の「Org codex model slugs」Check が担うが、既定値の更新手順(ロックステップ面の一覧と下流配布)が文書化されていない(issue #156 のやること 3)。あわせて観測(やること 1)の初回データ点を記録する
- Related issue: 156
- Type: docs
- Branch: docs/org-codex-slug-update-procedure

## Objective

issue #156 のうち今すぐ完了できる部分を PR にする: (1) 運用者向けに「doctor が warn したら自分の `ralph.toml` をどう直すか」を `/org` skill の既定 model_pool 表の直後に 1 段落追加する。(2) メンテナ向けに「既定スラッグを更新するときに同時に変える面の一覧と、下流への配布経路」を root 専用ドキュメント(`docs/specs/2026-08-01-org-runtime.md`)に 1 節追加する。(3) 観測の初回データ点(2026-09-18)を issue コメントに記録する。2〜4 週間の観測と、消失時の既定更新 PR は本 PR の範囲外で、issue は open のまま残す(PR は `Refs #156`)。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | 運用者向け段落 | `.claude/skills/org/SKILL.md`(+ `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の 3 ミラー) | 「### 既定の model_pool」節末尾の「claude はエイリアス、codex はスラッグ…」段落の直後に 1 段落。要点: codex スラッグは codex のモデル更新で消える。`ralph doctor` の Check が warn したら `~/.codex/models_cache.json`(`$CODEX_HOME` で上書き可)の現行スラッグを確認し、自分の `ralph.toml` の `[org].model_pool` と、そのスラッグを参照する `[org.roles]` を置き換える。`ralph.toml` は seed-once で `ralph upgrade` は触らないため、既定値が上流で更新されても自分の `ralph.toml` に明示した `model_pool` は自動では変わらない。`model_pool` を書かず `driver_pool` のみの設定なら `ralph` バイナリの既定が使われるが、既定はバイナリに埋め込まれているため、追従には **バイナリ更新(`brew update && brew upgrade ralph` 等のインストール経路)→ `ralph version` で確認 → `ralph upgrade`** の順が要る。`ralph upgrade` 単体は今入っているバイナリの埋め込みテンプレートを適用するだけで、新しい既定は届かない |
| 2 | メンテナ向け節 | `docs/specs/2026-08-01-org-runtime.md`(check-sync の ROOT_ONLY 除外対象で templates に出ない) | 「2026-09-16 改訂」節の直後に「### 運用ノート: 既定 codex スラッグの更新手順」を追加。内容: (a) 同時に変える面 — `internal/config/config.go` の `Default()`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` の `RALPH_ORG_MODEL_POOL`、`.claude/skills/org/SKILL.md` の既定表(+3 ミラー、`scripts/sync-skills.sh` と cp)、本 spec の改訂注記 (c)、既定値をハードコードするテスト(`internal/config/config_test.go`、`internal/config/defaults_sync_test.go` がロックステップを検査、`internal/org/envelope_summary_test.go`、`internal/cli/org_test.go`)。(b) 検証 — `./scripts/run-verify.sh`、`ralph doctor`。(c) 配布は 2 段階 — `/release` でタグ → 下流は **まずバイナリを更新**(`brew update && brew upgrade ralph`、または各自のインストール経路)し `ralph version` で新バージョンを確認 → **その後** `ralph upgrade` で core(skill・`ralph-config.sh`)を置換する。`ralph upgrade` は `cmd/ralph/main.go` が `scaffold.EmbeddedFS` に注入した「今入っているバイナリの」埋め込みテンプレートを適用するため、バイナリ更新を飛ばすと旧既定・旧 core のまま成功したように見える。seed-once の `ralph.toml` はどちらでも更新されないので、下流運用者は項目 1 の手順で自分の `model_pool` を直す。(d) 観測手順と消失の判定基準 — `checkCodexModelSlugs` はローカルの `models_cache.json` を読むだけで更新も鮮度確認もしないので、観測は毎回 **cache を更新してから** 行う: `codex exec --sandbox read-only 'echo ok' </dev/null` 等で codex を一度起動(2026-09-18 に `codex exec` 実行後 mtime が 12:31 → 12:40 に進むことを確認)→ `stat` で cache の mtime が観測時刻に更新されたことを確認 → `ralph doctor` → issue #156 のコメントに「日時 / cache mtime / `codex --version` / モデル数 / 5 スラッグの有無 / doctor 結果」を記録。mtime が更新されていない観測は不成立(inconclusive)として数えない。一時的な消失(2026-09-17 未明に数時間で復帰した事例)と区別するため、更新済み cache で 2〜4 週間続けて消えたままの場合だけ既定から外す(または後継スラッグへ置換する) |
| 3 | 観測の初回データ点 | issue #156 コメント(gh) | 2026-09-18 12:40 JST 時点(cache mtime 12:40:07、`codex exec` 実行後に更新済み、codex-cli 0.154.0): `~/.codex/models_cache.json` に 7 モデル、既定 5 スラッグ(`gpt-6-astra` / `gpt-5.6-sol` / `gpt-5.6-terra` / `gpt-5.6-luna` / `gpt-5.5`)すべて存在、`ralph doctor` の Check は pass。次回観測の目安(1 週間後、2 週間後、4 週間後)と、観測前の cache 更新手順(項目 2 の (d))も書く。PR の `/pr` 段階で投稿する(gh の書き込みは `yoshpy-dev` アカウント) |

## Non-goals

- 既定 `model_pool` の変更(`gpt-6-astra` は 2026-09-18 時点で存在しており、消失していない)
- 観測の自動化(cron / CI での doctor 実行)。手動観測 + issue コメントで足りる規模
- `ralph upgrade` に seed-once `ralph.toml` の `model_pool` を書き換える機能を足すこと
- `docs/recipes/codex-setup.md` の変更(recipes は seed-once で下流に固定コピーされるため、変わりうる手順は core の skill 側に置く)
- issue #156 のクローズ(観測が残る。PR は `Refs #156`)

## Assumptions

- 所有区分は `docs/specs/2026-08-17-overlay-scaffold-v2.md` の L5 seed-once(`ralph.toml`、`docs/recipes/`)と core(`.claude/skills/`、`scripts/ralph-config.sh`)に従う。`templates/base/ralph.toml` は `[org].model_pool` を有効行として明示している(2026-09-18 に確認)ので、`ralph init` した下流プロジェクトは既定で明示 `model_pool` を持つ
- `docs/specs/` は check-sync の ROOT_ONLY 除外対象で、メンテナ専用の記述を置いても templates との drift にならない
- `/org` SKILL.md の追加段落は 4 面ミラーを `scripts/sync-skills.sh` + cp で揃え、`check-skill-sync.sh` / `check-sync.sh` で一致を検証する(#154 と同じ手順)

## Affected areas

- `.claude/skills/org/SKILL.md` と 3 ミラー
- `docs/specs/2026-08-01-org-runtime.md`
- issue #156(コメント)

## Design decisions

Critical forks: None。

既定として採った選択:

- 文書は読者で分ける。運用者向け(自分の `ralph.toml` の直し方)は下流に配布される core の `/org` skill へ、メンテナ向け(ロックステップ面の一覧・配布経路・消失判定)は root 専用の spec へ。issue が候補に挙げた `docs/recipes/` は seed-once で下流に固定コピーされ、更新が届かないため採らない
- 観測ログは `docs/evidence/` ではなく issue コメントに置く。時系列で追記する性質と、消失時の判断を issue 上で行う流れに合う(issue の受け入れ条件はどちらも許容)
- 3 件とも数行の文書変更で、implementer への handoff より変更コストが小さいため inline で実施する(`.claude/rules/ralph/subagent-policy.md` の trivial edit 例外)
- Codex plan advisory(codex-cli 0.154.0)の 2 所見を採用: HIGH-1「`ralph upgrade` は既存バイナリの埋め込みテンプレートを適用するだけなので、既定の配布にはバイナリ更新 → `ralph upgrade` の 2 段階が要る」(`cmd/ralph/main.go:18` の `scaffold.EmbeddedFS = ralph.TemplatesFS` で確認)、MEDIUM-2「doctor は cache を更新も鮮度確認もしないので、観測は cache 更新 + mtime・codex バージョンの記録を必須にし、未更新の観測は不成立とする」(`doctor_codex_models.go` に ModTime / exec 呼び出しがないことを確認)

## Acceptance criteria

- [ ] AC-1: `/org` SKILL.md の「### 既定の model_pool」節に、doctor の warn を受けて自分の `ralph.toml` を直す手順、`ralph.toml` が seed-once で `ralph upgrade` に上書きされない旨、既定の追従には「バイナリ更新 → `ralph upgrade`」の順が要る旨を含む段落がある。`grep -c 'seed' .claude/skills/org/SKILL.md` と `grep -c 'brew upgrade' .claude/skills/org/SKILL.md` が各 1 以上。4 ミラーが byte 一致(`cmp` ×2)し、`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が pass
- [ ] AC-2: `docs/specs/2026-08-01-org-runtime.md` に「### 運用ノート: 既定 codex スラッグの更新手順」節があり、`config.go` / `templates/base/ralph.toml` / `ralph-config.sh`(2 面)/ SKILL.md(4 面)/ spec 改訂注記 / テスト 4 ファイル / `run-verify.sh` / `ralph doctor` / `ralph upgrade` / `brew upgrade` / `ralph version` / `EmbeddedFS` / `mtime` / `codex --version` の各語を含む(`grep -c` で各 1 以上)。配布の 2 段階と観測前の cache 更新手順が明文化されている
- [ ] AC-3: issue #156 に 2026-09-18 の観測コメント(cache mtime、codex バージョン、モデル数、5 スラッグの有無、doctor 結果、観測前の cache 更新手順、次回観測の目安)が投稿されている
- [ ] AC-4: `./scripts/run-verify.sh` green(文書のみの変更だが、sync 系ゲートを含むため)
- [ ] AC-5: PR 本文は `Refs #156`(`Closes` ではない)で、issue は open のまま残る

## Implementation outline

1. Slice A(inline): 項目 1。`.claude/skills/org/SKILL.md` に段落追加 → `./scripts/sync-skills.sh` → `templates/base/` 2 面へ cp → `check-skill-sync.sh` / `check-sync.sh` → commit `docs: tell /org operators how to refresh stale codex slugs in ralph.toml`
2. Slice B(inline): 項目 2。spec に節追加 → `./scripts/run-verify.sh` → commit `docs: record the default codex slug update procedure in the org runtime spec`
3. Slice C(`/pr` 段階): 項目 3 の issue コメント投稿(gh、`yoshpy-dev`)
4. post-implementation pipeline → `/cross-review` → `/pr`(`Refs #156`)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(markdown 変更のみだが sync 系ゲートを含む)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC-1・AC-2 の grep、4 ミラーの `cmp`
- Documentation drift to check: 新段落・新節の記述が実装と一致すること — `ralph.toml` が seed-once であること(overlay-scaffold-v2 spec L5)、`checkCodexModelSlugs` の参照パス(`$CODEX_HOME` 優先、既定 `~/.codex/models_cache.json`)、`defaults_sync_test.go` が検査する面(`ralph-config.sh` と `templates/base/ralph.toml` と `Default()`)
- Evidence to capture: grep / cmp / sync スクリプトの出力

## Test plan

- Unit tests: 変更なし(文書のみ)。回帰として `go test ./internal/config/... -count=1`(`defaults_sync_test.go` が spec の記述と矛盾しないことの確認に使う)
- Integration tests: なし
- Regression tests: `./scripts/run-test.sh`
- Edge cases: SKILL.md の段落追加が既定表の直後の段落と `## 動詞リファレンス` 見出しの間に入り、`markdownSection` 相当の節区切り(`## `)を壊さないこと。spec の新節が既存の見出し階層(`###`)に揃うこと
- Evidence to capture: `docs/reports/test-*.md`

## Risks and mitigations

- 文書が実装(所有区分・パス)とずれる → verify で overlay-scaffold-v2 spec と `doctor_codex_models.go` に照合する
- 4 面ミラーの取り残し → sync-skills + cp + 2 つの sync チェック
- issue を open のまま PR をマージすることの誤解 → PR 本文と issue コメントに「観測継続のため open 維持」を明記

## Rollout or rollback notes

文書のみ。`/org` skill の段落は core として次回 release 後に `ralph upgrade` で下流に届く。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

- 2026-09-18 plan: 観測の初回データ点は plan 作成時に取得済み(cache 7 モデル、5 スラッグ全存在、doctor pass)。cache mtime は最初の読み取り時 12:31、その後の `codex exec`(advisory 実行)で 12:40:07 に更新されたことを確認 — codex 起動が cache を更新する根拠
- 2026-09-18 plan: Codex plan advisory(`</dev/null` 付きで正常完了)の HIGH-1 / MEDIUM-2 を裏取りのうえ採用。Scope 1〜3、Design decisions、AC-1〜AC-3 を改訂

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
