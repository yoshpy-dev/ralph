# Walkthrough: org-envelope-low-findings

- Date: 2026-09-17
- Plan: docs/plans/archive/2026-09-17-org-envelope-low-findings.md(PR 作成時に active から移動)
- Issue: #154
- Branch: chore/org-envelope-low-findings(base: main @ 4905c34)
- 差分規模: 22 files / +815 −54(うちコード・スキル・雛形は 14 files / +229 −53。残りは plan と 5 本の pipeline レポート)

## 読む順番

1. `internal/cli/doctor_codex_models.go` — 本 PR で唯一の実行時挙動変更。`codexModelsCachePath` が `(string, error)` を返し、空でない `CODEX_HOME` を literal に使う(codex の `find_codex_home` と同じ判定)。home 解決失敗は `checkCodexModelSlugs` が `info` で報告する。doc comment の outcomes は 6 件。
2. `internal/cli/doctor_org_test.go` — 追加テスト 3 件。末尾空白付きディレクトリ(trim 禁止の回帰ガード)、空白のみ `CODEX_HOME`(挙動が実際に変わった唯一の入力)、`HOME=""` での info 経路。
3. `internal/org/prompts_test.go` — `markdownSection` ヘルパー(見出しを行単位でアンカー、空セクションは空文字)とそのテーブルテスト、fan-out 検査の節スコープ化、lead fixture の `EnvelopeSummary(config.Default().Org)` 化。
4. `internal/config/config_test.go`、`internal/cli/org_test.go` — アサート精度の引き締め(改名 + doc comment、エラー文言の完全一致、`strings.Count == 1`)。
5. `internal/org/watch.go` — `pruneRetiredConditions` の冗長 delete 1 行削除(挙動不変、既存回帰テストで担保)。
6. `internal/org/prompts.go`、`internal/org/spawn.go`、`internal/config/config.go` — 陳腐化した doc comment の修正のみ。
7. `.claude/skills/org/SKILL.md` + 3 ミラー、`internal/org/prompts/lead.md` — 文言修正。`check-skill-sync.sh` / `check-sync.sh` green。
8. `docs/tech-debt/README.md` — 対象行を `~~` + `(RESOLVED 2026-09-17 in chore/org-envelope-low-findings)` でクローズ。

## コミット単位

| SHA | 内容 |
|---|---|
| 314b89f | Slice A: 項目 1・3・7・8(Go コメント/コード)+ テスト 2 件 |
| 746c70d | Slice B: 項目 5・9・10(テスト精度) |
| 3a9362c | Slice C: 項目 4・6、tech-debt 行クローズ、4 面ミラー同期 |
| ff30ee2 | Slice D: self-review L1〜L8(同 cycle 内修正) |
| b677a95 / 02ad186 | Slice E: self-review N1〜N3(inline) |
| その他 | plan の進捗・逸脱記録、self-review / verify / test / sync-docs / cross-review レポート |

## 設計判断(plan Design decisions より)

- `CODEX_HOME` は trim しない。tech-debt 行は「untrimmed で join」を欠点として挙げていたが、codex 自身が `is_empty()` でのみ濾して値を literal に使うため、trim すると doctor が codex と別のディレクトリを見る。Codex plan advisory の指摘を codex ソース(rust-v0.149.1 / rust-v0.154.0)で裏取りして採用。
- 項目 2(`DefaultModelForDriver` doc)は 463e943 で修正済みを確認し、変更なし。
- self-review で出た LOW(L1〜L8、N1〜N3)はすべて同 cycle 内で修正。本 PR の趣旨が「先送り LOW の解消」なので、新たな先送り行を作らない。

## レビューで特に見てほしい箇所

- `codexModelsCachePath` の空白のみ `CODEX_HOME` の扱い(未設定扱い → literal)。codex 側では fatal になる値なので実運用では発生しないが、doctor の info 文言が変わる。
- `markdownSection` の `-1` オフセット(空セクションで次セクション本文を返さないための処置)と、そのテーブルテストの第 5 ケースがフィクスチャの行中言及 `see ## D` で判別可能になっていること。
