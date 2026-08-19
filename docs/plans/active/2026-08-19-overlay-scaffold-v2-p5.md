# overlay-scaffold-v2-p5

- Status: Active
- Owner: Claude Code
- Date: 2026-08-19
- Related request: overlay-scaffold-v2 Phase 5(最終段): eject/adopt、doctor --strict、status 所有表示、上流固有記述混入ガード、Codex hooks dispatcher パリティ、系列リリース準備
- Related issue: N/A
- Type: feat
- Branch: feat/overlay-scaffold-v2-p5
- Canonical ref: docs/specs/2026-08-17-overlay-scaffold-v2.md(Phase 1 = #143、Phase 2 = #144、Phase 3 = #145、Phase 4 = #146)

## Objective

overlay-scaffold-v2 系列の残 FR(FR-2 eject / FR-3 adopt / FR-9 doctor --strict / FR-10 混入ガード / FR-12 status 所有表示)と Phase 5 トリガーの tech-debt(Codex hooks dispatcher パリティ)を実装し、系列をリリース可能な状態で完了させる。

## Scope

1. **FR-2 `ralph eject <path>`**: core 所有(または未解決 drift 状態の core)パスを fork(ユーザ所有)へ移す。manifest に `owner=fork` + `forked_from_version` + ディスク実体ハッシュを記録。ディスク書き込みゼロ。以後の upgrade は置換せず advisory(既存の classifyFork 経路が引き受ける — 新規分類ロジック不要)。**対象は core replace planner が分類するパスのみ**: v2 例外面(`.claude/settings.json`、settings スナップショット、AGENTS.md / `.gitignore` の block 面)は planner 外の専用機構(settings 3-way マージ / block エンジン)が書き換えるため、fork 記録では保護できない。これらは明確なメッセージ(各面のカスタマイズ手段への誘導付き)で拒否する(Codex 所見 1)。
2. **FR-3 `ralph adopt <path>|--all`**: fork 記録(または fork 記録なしの未解決 drift)を core 所有へ戻す。ディスクを現行埋め込みテンプレート内容で置換し、manifest を `owner=core` + 現行ハッシュに更新。`--all` は全 fork + 全 drift を一括処理。破壊的操作(ユーザ内容の破棄)のため、(1) migration と同じ **git クリーン前提**(汚れていれば中断 — 破壊された内容の唯一の復元経路を保証する)、(2) y/N 確認 + `--yes`、(3) 書き込み前の全対象プリフライト(封じ込め+symlink 検証+テンプレート存在確認)を必須とする(Codex 所見 2)。対象の例外面拒否は eject と同一。
3. **FR-9 `ralph doctor --strict`**: (a) core 全ファイルのハッシュ一致(fork 除く)、(b) managed block 健在(マーカー整合 + block 内容一致)、(c) settings.json の ralph 所有キー健在、(d) 追跡ファイルのコンフリクトマーカー不在、(e) manifest と実体の整合 — を検査。`--strict` で違反時 exit 1、なしでは警告表示に留める(既存 doctor の checkResult 形式に統合)。
4. **FR-12 `ralph status` 所有表示**: v2 manifest のパスごとの所有属性(core / fork / seed / block)と未解決 drift の一覧を表示するセクションを追加。既存の org seat 表示は現状維持(org state が無い下流プロジェクトでも所有表示は動く)。レガシー manifest では移行案内。
5. **FR-10 上流固有記述混入ガード**: `templates/` 配下にメタリポ固有参照(`yoshpy-dev`、リポジトリ URL、`/release` スキル等 maintainer 専用要素)が混入していないことを検査する `scripts/check-template-purity.sh` を新設し、`run-static-verify.sh` 系列(CI)に組み込む。検出パターンはスクリプト内で宣言的に管理し、意図的な出現(あれば)は allowlist で明示。
6. **Codex hooks dispatcher パリティ**(tech-debt 行 103)**+ pre_bash_guard の jq 経路修正**(tech-debt 行 100 — トリガー「次の hooks 変更」が本スライスで発火): `.codex/config.toml`(root + `templates/base/`、byte-identical 維持)の `[[hooks.*]]` 直接呼び出し(check_mojibake.sh ×2)を dispatcher 経由に置き換え、`.claude/hooks/<event>.d/` → `.ralph/local/hooks/<event>.d/` → `.claude/hooks/local/<event>.d/` の 3 層が Codex でも機能するようにする。Codex hooks の stdin/プロトコルが Claude Code と異なる場合は薄い変換シムを置く。同時に `pre_bash_guard.sh` の jq 経路が実 PreToolUse ペイロードの `.tool_input.command`(ネスト)ではなくトップレベル `.command` を読んでいて常に素通しになる欠陥を修正(root + テンプレート両方)し、実ペイロード形の fixture テスト(jq あり経路)を追加する(Codex 所見 3)。`tests/test-hook-wiring.sh` を直接呼び出し検出で強化。
7. **ドキュメント/スペック整合**: スペックの FR チェックボックス更新、tech-debt 行 103・100 の RESOLVED 化、README / AGENTS.md / 関連 rules の更新。

## Non-goals

- リリースタグの発行(`/release` は repo-maintainer の手動トリガー。P5 マージ後の hand-off 事項として記録するのみ)
- tech-debt 行 99(パス形状遷移の分類)— トリガーは「replaceplan.go の分類への次のタッチ」であり、本フェーズの eject/adopt/doctor は分類ロジックに触れない。行はそのまま維持
- Phase 4 Known gap(settled-relocation hash carry)の解消 — トリガー未到達
- `ralph status` の org 表示側の仕様変更
- 新しい言語 pack・テンプレート内容の追加

## Assumptions

- `scaffold.Manifest` の v3 所有 API(`SetFileFork` / `SetFileOwned` / `SetOwner`)と `PlanCoreReplaceDesired` の分類(fork advisory / drift)は Phase 1〜4 で完成しており、eject/adopt/doctor はこれを消費するだけで新分類を導入しない。
- doctor --strict の検査 (a)(e) は `PlanCoreReplaceDesired` のドライ実行(書き込みなし)で導出できる。(b) は block エンジンの検査関数、(c) は `OwnedSettingsPaths` + スナップショットで導出できる。
- Codex hooks のプロトコル調査はスライス着手時に実施(`codex` CLI のドキュメント/実挙動)。ralph-dispatch.sh と互換なら直接差し替え、非互換なら変換シム。どちらでも `.d` 3 層の実行順序契約は同一。
- eject/adopt/doctor --strict/status 所有表示は v2 layout(`meta.layout="v2"`)前提。レガシー manifest では Phase 4 と同じ fail-closed + 移行案内。

## Affected areas

- `internal/cli/eject.go` / `adopt.go`(新規)+ root.go 配線
- `internal/cli/doctor.go`(--strict フラグ + scaffold 整合チェック群)
- `internal/cli/status.go`(所有表示セクション)
- `internal/cli/upgrade_v2.go`(doctor/status から再利用する分類ヘルパの公開度調整があれば)
- `.codex/config.toml` + `templates/base/.codex/config.toml`(hooks 配線)
- `scripts/check-template-purity.sh`(新規)、`scripts/run-static-verify.sh`(組み込み)
- `tests/test-hook-wiring.sh`、新規 shell テスト(purity ガード)
- `docs/specs/2026-08-17-overlay-scaffold-v2.md`、`docs/tech-debt/README.md`、README.md、AGENTS.md

## Acceptance criteria

- [ ] AC-1 `ralph eject <path>`: 追跡済み owner=core パスを fork 化(owner=fork、forked_from_version=manifest 記録バージョン、hash=ディスク実体)。ディスク書き込みゼロ。
- [ ] AC-2 eject は未解決 drift 状態(fork 記録なしのハッシュ不一致)の core パスにも適用でき、直後の `ralph upgrade` が当該パスを drift(exit 3)ではなく fork advisory として扱う(FR-4 の解消経路)。
- [ ] AC-3 eject/adopt のエラー系: 未追跡パス / 既に fork(eject)・既に core(adopt)/ owner=seed・block / **v2 例外面(`.claude/settings.json`、settings スナップショット、AGENTS.md・`.gitignore`)** / レガシー manifest / ディスク欠落(eject)— いずれも明確なメッセージで拒否し、manifest・ディスク無変更。例外面の拒否メッセージは各面の正規カスタマイズ手段(settings はそのまま編集可 / block 外領域 / `.ralph/local/`)へ誘導する。
- [ ] AC-4 `ralph adopt <path>`: fork 記録パスのディスクを現行テンプレート内容で置換し、owner=core + 現行ハッシュに更新。fork 記録は消える。
- [ ] AC-5 adopt は fork 記録なしの未解決 drift にも適用できる(FR-4: 改変破棄)。
- [ ] AC-6 adopt の安全性: git 作業ツリーがクリーンであることを前提条件とし、汚れていれば書き込みゼロで中断(migration と同じ検査。破壊内容の git 復元可能性を保証)。y/N 確認(`--yes` で省略)。現行テンプレートが当該パスを ship しない(retired)fork は明確なメッセージで拒否。書き込み前に全対象をプリフライト(パス封じ込め+symlink 検証+テンプレート存在、Phase 3/4 と同じ `ValidateRealParentChain` 系)し、1 件でも失敗なら書き込みゼロで中断。
- [ ] AC-7 `ralph adopt --all`: 全 fork + 全 drift を一括 adopt。対象一覧を確認前に表示。対象ゼロなら no-op 明示。**部分失敗テスト**: 書き込み途中/manifest 書き込み失敗を注入し、(a) プリフライトで検出可能な失敗はゼロ書き込み、(b) 実行中失敗は git で全復元可能(クリーン前提の帰結)であることを固定する。
- [ ] AC-8 `ralph doctor --strict`: FR-9 の (a)〜(e) を検査し、違反 1 件以上で exit 1。`--strict` なしは同内容を警告表示のみ(exit 0、既存挙動維持)。
- [ ] AC-9 doctor --strict は fresh `ralph init` 直後および upgrade 収束済みプロジェクトで green(false positive ゼロの回帰テスト)。eject 済み fork は (a) の違反にならない。
- [ ] AC-10 `ralph status`: v2 プロジェクトでパスごとの所有属性と未解決 drift 一覧を表示。出力マトリクスをテキスト/`--json` の両形式で固定する: scaffold(v2 manifest)有無 × org state 有無 × レガシー manifest × `--org-id` / `--state-dir` 指定 — 各組み合わせの挙動を決定的テストで先に定義(`--json` は既存スキーマに additive なキー追加のみ。既存キーの形・意味は不変)。既存 org 表示・org 系テストは無変更 green。
- [ ] AC-11 `scripts/check-template-purity.sh` が `templates/` のメタリポ固有参照を検出して exit 1。現行 templates/ では green。`run-static-verify.sh` から呼ばれ、CI で実行される。検出の実効性は意図的に汚した fixture で shell テスト化(tech-debt 行 106 の教訓: 検出分岐に fixture を必ず用意)。
- [ ] AC-12 Codex hooks が dispatcher 経由で `.d` 3 層を実行する(root と templates/base は byte-identical、`scripts/check-sync.sh` green)。`tests/test-hook-wiring.sh` はレガシー直接呼び出し形を検出したら fail する。
- [ ] AC-12b `pre_bash_guard.sh` の jq 経路が実 PreToolUse ペイロード形(`.tool_input.command` ネスト)からコマンドを抽出し、deny/ask ルールが実際に発火する(root + テンプレート両方)。実ペイロード fixture の jq あり経路テストで固定。tech-debt 行 100 を RESOLVED 化。
- [ ] AC-13 eject → adopt の往復で manifest とディスクが収束済み状態に戻る(round-trip 統合テスト)。
- [ ] AC-14 スペックの FR-2/3/9/10/12 チェックボックス更新、tech-debt 行 103 RESOLVED 化、README / AGENTS.md 更新。
- [ ] AC-15 `./scripts/run-verify.sh` exit 0、shell + Go 全テスト green。

## Implementation outline

1. **Slice 1: eject + adopt(FR-2/3、AC-1〜7、13)** — `internal/cli/eject.go` / `adopt.go` 新規。manifest ロード(v2 バリア)→ 対象解決(単一パス / --all 列挙)→ eject: SetFileFork(書き込みゼロ)/ adopt: 確認 → 封じ込め検証 → テンプレート内容書き込み → SetFileOwned。manifest 書き込みは全操作成功後(コミットバリア)。単体+統合(round-trip)テスト。
2. **Slice 2: doctor --strict(FR-9、AC-8/9)** — doctor.go に scaffold 整合チェック群を追加(PlanCoreReplaceDesired ドライ実行 + block 検査 + settings 所有キー + コンフリクトマーカー + manifest/実体整合)。--strict フラグで fail 化。
3. **Slice 3: status 所有表示(FR-12、AC-10)** — status.go に所有セクション追加(manifest 読み + drift は doctor と同じドライ実行ヘルパを共用)。
4. **Slice 4: FR-10 purity ガード + Codex dispatcher パリティ + pre_bash_guard 修正(AC-11/12/12b)** — check-template-purity.sh 新設 + run-static-verify.sh 組み込み + fixture テスト。Codex hooks プロトコル調査 → dispatcher 配線(または変換シム)→ pre_bash_guard.sh の `.tool_input.command` 抽出修正(root + テンプレート)+ 実ペイロード fixture テスト → test-hook-wiring.sh 強化 → check-sync green 維持。
5. **Slice 5: ドキュメント/スペック整合(AC-14)** — スペックチェックボックス、tech-debt、README、AGENTS.md。リリース hand-off 注記。

各スライスは implementer 委譲(構造化ハンドオフ)、1 スライス 1 コミット、`./scripts/run-verify.sh` を検証コマンドに含める。

## Verify plan

- 静的解析: `./scripts/run-static-verify.sh`(gofmt / golangci-lint / check-sync / check-pipeline-sync / check-skill-sync + 新規 purity ガード)
- スペック準拠: FR-2/3/9/10/12 の各文言と実装の突合(特に FR-3「唯一の再採用経路」— `--force` が復活していないこと、FR-9 (a)〜(e) の網羅)
- ドキュメント drift: README のコマンド一覧 / AGENTS.md repo map / recipes、`ralph upgrade` help テキストとの整合
- 証跡: `docs/evidence/verify-*.log`

## Test plan

- 単体: eject/adopt の manifest 遷移(全 owner 種 × 追跡状態のマトリクス)、doctor 各チェックの pass/fail、status 出力、purity ガードのパターン検出
- 統合: eject → upgrade(advisory 化)→ adopt(復帰)→ upgrade(no-op)の round-trip、doctor --strict on fresh init / converged / 意図的に壊した各状態(block マーカー破損、settings キー欠落、コンフリクトマーカー、manifest 不整合)
- 回帰: 既存 599 shell + Go 全パッケージ、org status 系テスト無変更 green
- エッジ: adopt 対象の symlink 親、retired fork、--all 対象ゼロ、レガシー manifest、ディスク欠落 fork の adopt(テンプレートから再生成)
- 証跡: `docs/reports/test-*.md` + カバレッジ

## Risk register

| リスク | 影響 | 緩和 |
|---|---|---|
| Codex hooks プロトコルが ralph-dispatch.sh の stdin 契約と非互換 | Slice 4 の手戻り | 調査を Slice 4 冒頭に置き、非互換なら変換シム(契約は .d 3 層の実行順序のみ維持) |
| adopt の破壊性(fork 内容破棄)と部分失敗 | ユーザ内容喪失 + fork 記録とディスクの不整合 | git クリーン前提(汚れていれば中断 — 復元経路の保証)、y/N + `--yes`、対象一覧の事前表示、全対象プリフライト(1 件でも不合格ならゼロ書き込み)、失敗注入テスト |
| doctor --strict の false positive(収束済みでも fail) | CI ノイズ | AC-9 の回帰テストで fresh/converged の green を固定 |
| purity ガードの過剰検出(正当な文字列を fail) | CI ブロック | allowlist を宣言的に持ち、現行 templates/ green を AC に含める |
| status への所有表示追加が org 表示の既存契約を壊す | 下流の status 利用破壊 | 既存 org テスト無変更 green を AC-10 に含める |

## Rollout or rollback notes

- 全変更は加算的(新コマンド 2 つ、新フラグ 1 つ、status セクション追加、新スクリプト 1 つ、hooks 配線変更)。既存コマンドの挙動変更は Codex hooks 配線のみで、失敗時は旧直接呼び出しに戻すだけでロールバック可能
- リリースタグは本 PR マージ後に maintainer が `/release` で発行(hand-off 事項)

## Design decisions

- **status の形**: FR-12 はコマンド名を `ralph status` と明記しているため、既存 org status への追加セクションとする(新サブコマンドは作らない)。org 表示の既存契約は不変。
- **adopt の確認 UX**: 破壊的(fork 内容破棄)のため Phase 4 migration と同じ y/N + `--yes` を採用。単一パスも確認対象(誤爆時の損失がユーザ編集内容そのものであるため)。
- **retired fork の adopt**: 現行テンプレートに存在しないパスの fork は adopt 先が存在しないため拒否(削除は adopt の意味論ではない。ユーザが不要なら手動削除+manifest はその後の upgrade 整合に委ねる)。
- **eject の drift 適用**: FR-4 が「eject(改変維持)または adopt(改変破棄)で解消」と明記しているため、fork 記録がない drift 状態にも eject を適用可能とする。
- **eject/adopt の対象境界**(Codex 所見 1): core replace planner が分類するパスのみ。v2 例外面(settings.json / スナップショット / block 面)は専用機構が書き換えるため fork 記録で保護できず、偽の保護を約束するより明示拒否が正しい。
- **adopt の復元保証**(Codex 所見 2): バックアップ機構の自作ではなく migration と同じ git クリーン前提を採用。プリフライトのゼロ書き込み保証+クリーン git で「検出可能な失敗は無傷、実行中失敗は git 復元可能」の 2 段構え。
- Critical forks: None(上記はいずれもスペック文言・確立済みパターン・Codex 所見からの導出で、ユーザ判断を要する分岐はなし)

## Open questions

- ~~Codex hooks の実プロトコル(stdin JSON の形、イベント名)~~ → Slice 4 で解決(742260f): codex-cli 0.147.0 の実機テストでイベント発火(PostToolUse/Stop)は確認。project-scoped `[[hooks.*]]` の発火は scratch リポジトリでは trust 制約により再現できなかったが、`ralph-dispatch.sh` は stdin を無解釈のまま各 `.d` スクリプトへ透過するため、dispatcher 経由化はペイロード形の差異に依存しない(構造的に安全)。シム不要。詳細コメントは `.codex/config.toml` に記載。**残ギャップ**: 信頼済みプロジェクトでの対話的 `codex trust` 経由の実発火確認は未実施(Slice 5 で tech-debt 追記)。

## Deviations

- Slice 4(742260f): FR-10 ガード配線先は `run-static-verify.sh`(env ラッパ)ではなく実体の `scripts/verify.local.sh` `run_static_checks()`。`check-template-purity.sh` は downstream に ship しないため `check-sync.sh` の ROOT_ONLY_EXCLUSIONS に追加。
- Slice 4 の purity スキャンが現行 `templates/` に既存リーク 5 件を検出(KNOWN LEAK として allowlist 化、ガードは green 維持): quality-gates.md の check-sync.sh / check-template.yml 引用、adding-a-language-pack.md の check-sync.sh 引用 ×4 行、check-pipeline-sync.sh / xreview-helpers.sh のコメント内メタリポ slug 引用。→ Slice 5 で修正し allowlist から除去する(スコープ追加)。
- CLAUDE.md の「check-skill-sync.sh は meta-repo only」記述が stale(templates/base に同スクリプトは ship 済み)→ Slice 5 で修正(スコープ追加)。

## Evidence targets

- `docs/reports/self-review-2026-08-19-overlay-scaffold-v2-p5.md`
- `docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md`
- `docs/reports/test-2026-08-19-overlay-scaffold-v2-p5.md`
- `docs/reports/sync-docs-2026-08-19-overlay-scaffold-v2-p5.md`
- `docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md`
- `docs/evidence/verify-*.log`

## Progress checklist

- [x] Slice 1: eject + adopt(9bf09e2)
- [x] Slice 2: doctor --strict(b884a38)
- [x] Slice 3: status 所有表示(a61dbb5)
- [x] Slice 4: purity ガード + Codex dispatcher パリティ + pre_bash_guard 修正(742260f)
- [ ] Slice 5: ドキュメント/スペック整合
- [ ] Post-implementation pipeline(self-review → verify → test → sync-docs → cross-review)
- [ ] /pr

## Readiness checklist

- [x] Canonical ref(スペック)確認済み、FR-2/3/9/10/12 の文言を直接引用
- [x] 既存 API(manifest v3 所有 / PlanCoreReplaceDesired / block エンジン)の再利用点を特定済み
- [x] Codex hooks の現状(直接呼び出し 2 エントリ、root=template identical)確認済み
- [x] tech-debt の Phase 5 トリガー行(103)と非対象行(99)を区別済み
