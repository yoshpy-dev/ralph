# Walkthrough: org-codex-slug-update-procedure

- Date: 2026-09-18
- Plan: docs/plans/archive/2026-09-18-org-codex-slug-update-procedure.md(PR 作成時に active から移動)
- Issue: #156(観測継続のため open 維持、PR は Refs)
- Branch: docs/org-codex-slug-update-procedure(base: main @ 93cf3bb)
- 差分規模: 11 files / +534(文書のみ。実体は skill 4 面 + spec 1 節、残りは plan と 5 本の pipeline レポート)

## 読む順番

1. `.claude/skills/org/SKILL.md` の「### 既定の model_pool」節末尾の段落(3 ミラーは byte 一致)— 下流に配布される運用者向け。流れ: Check はローカル cache を読むだけで更新も鮮度確認もしない → codex を一度起動して cache を更新してから再確認、warn 1 回でスラッグを外さない(2026-09-17 の一時的消失の事例)→ `ralph doctor --probe-models` は成功なら存在確認、失敗は不在の証拠にならない → 消えたままなら自分の `ralph.toml` の `model_pool` / `[org.roles]` を書き換える → `ralph.toml` は seed-once で `ralph upgrade` は触らない。新しい既定はバイナリ更新後の `ralph upgrade` が出す seed advisory diff(`docs/reports/upgrade-*.md`)で見える → `driver_pool` のみの設定は埋め込み既定(宣言 driver に絞ったもの)がバイナリ差し替え時点で切り替わり、`ralph upgrade` は core を置換するだけで実効プールは変えない。
2. `docs/specs/2026-08-01-org-runtime.md` の「### 運用ノート: 既定 codex スラッグの更新手順」— メタリポのメンテナ向け(check-sync の ROOT_ONLY 除外なので下流には出ない)。(a) 同時に変える面: 3 面ロックステップ(`Default()` / `templates/base/ralph.toml` / `scripts/ralph-config.sh`、`defaults_sync_test.go` が検査)+ template 側 `ralph-config.sh`(`check-sync.sh` が検査)+ skill 4 面 + spec 注記 (c) + テスト 4 ファイル、最後に `git grep` で旧スラッグの残存を掃く(列挙外の 2 面を例示)。(b) 検証。(c) 配布は「バイナリ更新 → `ralph version` → `ralph upgrade`」の 2 段階で、`ralph upgrade` は今入っているバイナリの埋め込みテンプレート(`scaffold.EmbeddedFS`)しか適用しない。seed advisory(`AdvisoryEntry`)が下流の入口。(d) 観測は毎回 cache を更新してから、mtime / `codex --version` を記録。2 週間連続で消えたままなら外す判断に入り、迷えば 4 週間まで延長。

## コミット単位

| SHA | 内容 |
|---|---|
| 4fa920e | Slice A: skill 段落 + 4 面ミラー |
| b66ec5c | Slice B: spec 節 |
| 86b71fe | self-review H1・M1・M2・L1〜L4 |
| 07cdd0d | self-review follow-up 3 点(probe の非対称、driver_pool 絞り込み、位置表現) |
| その他 | plan の進捗・逸脱記録、self-review / verify / test / sync-docs / cross-review レポート |

## 設計判断(plan Design decisions より)

- 読者で置き場所を分ける: 運用者向けは core の skill(`ralph upgrade` で下流に届く)、メンテナ向けは root 専用 spec。issue が候補に挙げた `docs/recipes/` は seed-once で下流に固定コピーされ更新が届かないため採らない。
- Codex plan advisory の 2 所見を採用: HIGH「`ralph upgrade` は既存バイナリの埋め込みテンプレートを適用するだけ」(`cmd/ralph/main.go` で確認)、MEDIUM「doctor は cache を更新も鮮度確認もしない」(`doctor_codex_models.go` に ModTime / exec なし。`codex exec` 実行で mtime が 12:31 → 12:40:07 に進むことを確認)。
- self-review HIGH-1(配布側に cache 鮮度の注意が抜けていた)を受け、運用者段落を「warn 1 回で外さない」を軸に書き直した。
- 観測ログは `docs/evidence/` ではなく issue コメントに時系列で残す。

## レビューで特に見てほしい箇所

- 運用者段落が下流の一般読者にとって自己完結しているか(`internal/` / `cmd/` への参照や、メタリポ限定の指示語が入っていないこと)。
- spec (c) の 2 段階配布の説明と、skill 側の「`ralph upgrade` 単体では新しい既定は届かない」が互いに矛盾しないこと。
