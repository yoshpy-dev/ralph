# overlay-scaffold-v2 — Phase 2: テンプレート v2 再編 + init

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-17
- Related request: docs/specs/2026-08-17-overlay-scaffold-v2.md(Phase 2 / 全 5 段。Phase 1 は PR #143 でマージ済み)
- Related issue: N/A
- Type: feat
- Branch: feat/overlay-scaffold-v2-p2

## Objective

`templates/base/` を 5 層オーバーレイの v2 レイアウトに再編し、`ralph init` が v2 レイアウト + manifest v3(所有属性付き)を直接生成するようにする。同一 PR でメタリポジトリ自身の root ハーネス(`.claude/`・`CLAUDE.md`・`AGENTS.md`・`.gitignore`・`scripts/run-*.sh`)も v2 へ移行し、`check-sync.sh` の同期ゲートを正直に保つ。upgrade の配線変更は行わない(Phase 3)。

## Scope

**テンプレート再編(`templates/base/`)**

- ralph ガイダンスの移設: `CLAUDE.md` の Default behavior 等の内容と既存 shipped rules(9 ファイル)を `.claude/rules/ralph/` 配下へ。`CLAUDE.md` は最小シード化(ralph 記述ゼロ、rules 自動ロードに委ねる)
- `AGENTS.md`: ユーザ所有スケルトン + managed block(`<!-- BEGIN RALPH MANAGED (ralph:agents-md) -->`)。block 生成元は `.ralph/core/AGENTS.core.md`(新設)
- `.gitignore`: ralph 必須エントリを managed block(`ralph:gitignore`)で区切る
- hooks の dispatcher 化: `settings.json` の各イベントは単一の `./.claude/hooks/ralph-dispatch.sh <event>` を指す。**hook 実装は現位置(`.claude/hooks/*.sh`)を維持**し(`dirname "$0"` による `lib_json.sh` 解決と `.codex/config.toml` からの直接参照を壊さないため)、`.claude/hooks/<event>.d/NN-<name>.sh` には実装を `exec` する薄い呼び出しエントリのみを置く。実行順は core `.d` → `.ralph/local/hooks/<event>.d/`(コミット対象)→ `.claude/hooks/local/<event>.d/`(gitignore 済み)の辞書順、stdin は各スクリプトへ再供給
- dispatcher の **JSON マージ意味論**(現行 hook が `permissionDecision` / `additionalContext` JSON を出力する事実に基づく): 各スクリプトの stdout を JSON として解釈し、(1) decision 系フィールド(`permissionDecision` 等)を含む出力が現れたら**最初の decision が勝ち**、その JSON を即座に出力して残りをスキップ、(2) `additionalContext` は全スクリプト分を結合して単一 JSON に集約、(3) 非 JSON のプレーンテキストは context として扱い集約、(4) 非ゼロ exit は最初の 1 件で中断・伝播。制約と意味論は dispatcher ヘッダに明記し、実ペイロードでテストする
- `.ralph/local/` 骨格: `hooks/<event>.d/`・`verify.d/`・`test.d/`(.gitkeep)
- `scripts/run-verify.sh` / `run-static-verify.sh` / `run-test.sh`: core 処理後に対応する `.ralph/local/*.d/*.sh` を実行する拡張点を追加

**`ralph init` の v2 生成(`internal/cli/init.go` ほか)**

- manifest v3 で書き出す: `SetLayoutV2()` + 全エントリに owner 付与(`SetFileOwned` / seed 判定)。所有クラス: `AGENTS.md`・`.gitignore` = block / `CLAUDE.md`・`ralph.toml`・`docs/**`・`.github/workflows/verify.yml` = seed / それ以外(`.claude/**`・`.agents/**`・`.codex/**`・`scripts/**`・`.ralph/core/**`・packs)= core
- 既存ファイルがある場合: seed は現行どおり触らない。block 面(`AGENTS.md`・`.gitignore`)は Phase 1 の block エンジンで block を追記(block 外は不可侵)
- baseline 書き込み(`.ralph/baseline/`)は現行維持(撤去は Phase 3)

**言語パック追従**

- pack rule のレンダリング先を `.claude/rules/<lang>.md` → `.claude/rules/ralph/<lang>.md` へ(`internal/cli/language_pack.go` の `packRuleRelPath`、`internal/cli/upgrade.go` の参照、`scripts/check-sync.sh` の `pack_rule_source`)

**旧 upgrade の過渡ガード**

- 旧 `ralph upgrade` エンジンが manifest の `meta.layout = "v2"` を検出したら fail-closed する(「layout v2 は Phase 3 の upgrade を待つ」旨の明確なエラー)。「リリースしないから安全」への依存を排除し、Phase 2 コードが誤って v2 レイアウトへ旧エンジンを適用する経路(--force による seed/block 面の全文上書き等)を機械的に塞ぐ

**check-sync の block-aware 化**

- `scripts/check-sync.sh` に block 所有面(`AGENTS.md`)の比較モードを追加: root と template を whole-file 比較する代わりに、双方の managed block 内容が `.ralph/core/AGENTS.core.md`(テンプレート源)と一致することを検査し、block 外は所有者(root=メタリポ、template=下流スケルトン)ごとの自由領域として比較対象外にする。これにより AGENTS.md の KNOWN_DIFFS を解除しつつゲートを正直に保つ

**メタリポジトリ root の同時移行**

- root `.claude/rules/` → `rules/ralph/` 移設、`CLAUDE.md` 最小化、`AGENTS.md` block 化、`.gitignore` block 化、hooks dispatcher 化、`.ralph/local/` 骨格、`scripts/run-*.sh` 拡張点 — テンプレートと同内容
- `scripts/check-sync.sh` の `KNOWN_DIFFS` から `AGENTS.md` を解除(tech-debt 回収)。`CLAUDE.md`/`model-routing.md` の既存 KNOWN_DIFFS は root 固有記述の残存量を見て維持または縮小
- root `AGENTS.md` のメタリポジトリ固有内容(Repo map 等)は block 外のユーザ所有領域として残す

## Non-goals

- `ralph upgrade` の配線変更・対話エンジン撤去・baseline 廃止(Phase 3)
- 旧レイアウト下流の移行ロジック(Phase 4)
- `eject`/`adopt`/`doctor --strict`/混入ガード(Phase 5)
- `.codex/` の再編(dispatcher の Codex パリティ、`config.toml` hooks の `.d` 化)— Phase 5。root と template の `.codex/` は現状のまま同期維持
- `.agents/skills/` ミラー機構の変更(skills は L1 のまま現行同期)
- リリースタグ発行

## Assumptions

- テンプレートはリリースタグまで下流に届かないため、「Phase 2 の init が生成する v2 レイアウト + 旧 upgrade コード」という過渡状態は main 上のみで、下流には露出しない
- 旧 upgrade は manifest v3 の追加フィールドを無視して動作する(Phase 1 AC-8 で `Managed` 互換を保証済み)
- Claude Code は `.claude/rules/` を再帰ロードするため `rules/ralph/` への移設で挙動は変わらない(公式ドキュメント確認済み)
- 現行 hook のうち `pre_bash_guard.sh` は `permissionDecision` JSON、`post_edit_verify.sh` は `additionalContext` JSON を出力する(Codex アドバイザリで確認)。dispatcher は上記の JSON マージ意味論を実装しない限り安全でない — 単純な stdout 連結は採らない

## Affected areas

- `templates/base/` ほぼ全域(`.claude/`、`CLAUDE.md`、`AGENTS.md`、`.gitignore`、`scripts/run-*.sh`、`.ralph/core/`・`.ralph/local/` 新設)
- `internal/cli/init.go`、`internal/cli/language_pack.go`、`internal/cli/upgrade.go`(pack rule パス参照のみ)、`internal/scaffold/render.go`(必要なら)
- root ハーネス一式(テンプレートと同構造)
- `scripts/check-sync.sh`(KNOWN_DIFFS、pack_rule_source)、`scripts/run-*.sh`
- テスト: `internal/cli` init/pack/upgrade 系、`tests/` の該当 shell テスト

## Design decisions

- 【fork 解決済み・ユーザ承認】メタリポジトリ root を同一 PR で v2 へ同時移行する。check-sync ゲートを正直に保ち、dogfooding 原則と整合。KNOWN_DIFFS 肥大(全面 suppression)を回避し、Phase 1 の AGENTS.md KNOWN_DIFFS tech-debt を回収
- dispatcher はスペックどおり「イベントごとに単一エントリ」。core hook も `.d` ディレクトリへ移設し、settings.json の hooks 節を将来にわたり不変化(hooks レイヤリングが同名イベント上書きである問題の構造的解決)。実行順は `NN-` プレフィクスの辞書順
- `.codex/` は Phase 2 で触らない(スコープ肥大回避。Codex は AGENTS.md(block 内容)と `.claude/rules/` 参照規約で v2 ガイダンスに追従できる)
- 所有クラスの決定は init 時のパスマップ(コード内テーブル)で行い、manifest v3 に記録する。Phase 3/4 はこの記録を消費する
- 【Codex アドバイザリ反映(4 件)】(1) dispatcher は JSON マージ意味論(decision 先勝ち・additionalContext 集約)を実装 — 「テキストのみ」という当初仮定は誤りだった(pre_bash_guard / post_edit_verify が JSON 出力) (2) hook 実装は現位置維持 + `.d` は薄い呼び出しエントリ方式 — `dirname "$0"` の lib 解決と `.codex/config.toml` の直接参照を保全 (3) check-sync を block-aware 化して AGENTS.md の KNOWN_DIFFS を正直に解除 (4) 旧 upgrade に `meta.layout="v2"` fail-closed ガードを追加し「リリースしないから安全」への依存を排除
- Critical forks: root 同時移行の 1 件のみ。他は spec / Phase 1 で決定済み

## Acceptance criteria

- [ ] AC-1 新規ディレクトリでの `ralph init` が v2 レイアウトを生成する: `.claude/rules/ralph/`、dispatcher 化 settings.json、`.claude/hooks/<event>.d/`、`.ralph/core/AGENTS.core.md`、`.ralph/local/` 骨格、最小 CLAUDE.md、managed block 付き AGENTS.md・.gitignore
- [ ] AC-2 生成された manifest が v3 である: `meta.layout = "v2"`、全エントリに owner(AGENTS.md/.gitignore = block、CLAUDE.md/ralph.toml/docs/workflows = seed、他 = core)
- [ ] AC-3 既存 CLAUDE.md/AGENTS.md/.gitignore があるディレクトリへの init: CLAUDE.md は不変(seed-once)、AGENTS.md/.gitignore は block のみ追記され block 外バイトが保全される
- [ ] AC-4 dispatcher: イベント発火で core `.d` → `.ralph/local` → `.claude/hooks/local` の順に実行され、stdin が各スクリプトへ渡り、非ゼロ exit が伝播する。**JSON マージ意味論**が実ペイロードでテストされる: PreToolUse の `permissionDecision`(deny が単一の有効 JSON として保たれる)、PostToolUse の `additionalContext` 集約、UserPromptSubmit / SessionStart の context 出力 — いずれも出力が単一の有効な hook 応答であること
- [ ] AC-5 `run-verify.sh` / `run-test.sh` が `.ralph/local/verify.d|test.d` の drop-in を core 処理後に実行する(fixture テスト)
- [ ] AC-6 言語パック: pack rule が `.claude/rules/ralph/<lang>.md` に描画され、manifest で owner=core として追跡される。`check-sync.sh` の pack_rule_source が新パスで機能する
- [ ] AC-7 root 同時移行後、block-aware 化した `./scripts/check-sync.sh` が KNOWN_DIFFS から `AGENTS.md` を外した状態で DRIFTED=0 になる(AGENTS.md は managed block 内容 = `.ralph/core/AGENTS.core.md` の一致で判定)
- [ ] AC-8 メタリポジトリのハーネスが v2 で動作する: `./scripts/run-verify.sh` exit 0(drop-in 拡張点含む)+ **設定済みの全 Claude / Codex hook コマンドが実行可能である**ことのスモークテスト(移設後のパス欠損検出)
- [ ] AC-9 既存テストスイート(shell + Go)が全パスし、`ralph upgrade` の挙動配線変更が「pack rule パス定数の追従 + v2 fail-closed ガード」に限られる
- [ ] AC-10 旧 upgrade ガード: `meta.layout = "v2"` の manifest に対して `ralph upgrade`(旧エンジン)が書き込みゼロで fail-closed し、Phase 3 を案内するエラーを返す(テストで保証)
- [ ] AC-11 メタリポジトリ固有内容(Repo map の internal/ 記述等)が `templates/base/AGENTS.md` に混入していない(block 外領域の分離をテスト or grep ゲートで確認)

## Implementation outline

1. テンプレート再編(ガイダンス移設・CLAUDE.md 最小化・AGENTS.core.md + block シード・.gitignore block)
2. dispatcher(JSON マージ意味論 + 実ペイロードテスト)+ hooks `.d` 呼び出しエントリ + `.ralph/local/` 骨格 + `run-*.sh` 拡張点(テンプレート側)
3. `ralph init` v2 生成(manifest v3・所有マップ・既存ファイルへの block 追記)+ テスト
4. pack rule パス移設(init/pack/upgrade/check-sync)+ 旧 upgrade の v2 fail-closed ガード + テスト
5. check-sync の block-aware 化 + root メタリポジトリ移行 + KNOWN_DIFFS 解除 + 同期ゲート・hook スモーク全パス確認
6. sync-docs(AGENTS.md block 化に伴う repo map・recipes・rules 参照の追従)

各スライスは implementer 委譲、1 スライス 1 コミット。スライス 5 は root ハーネス自体を変えるため、コミット前に `run-verify.sh` と hook スモークを同スライス内で実施。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(golang + shell)。`check-sync.sh` / `check-skill-sync.sh` / `check-pipeline-sync.sh` 全パス
- Spec compliance criteria to confirm: スペック層モデル(L1/L2/L3/L5 の物理配置)、FR-5(block 仕様の適用)、FR-11(init v2)、AC-9(init 直後の doctor 整合は Phase 5 の --strict 導入前なので現行 doctor の非退行のみ)
- Documentation drift to check: AGENTS.md(root: block 化 + Repo map 保全)、CLAUDE.md 最小化に伴う参照、docs/recipes の rules/hooks パス記述、`.claude/rules/ralph/` 移設後の相互参照
- Evidence to capture: init 生成ツリーのスナップショット、manifest v3 サンプル、dispatcher スモークログ、check-sync 出力

## Test plan

- Unit tests: 所有マップ(パス→owner)の判定、dispatcher の実行順/exit 伝播/stdin 再供給(bash テスト)、block 追記付き init の分岐
- Integration tests: 一時ディレクトリでの `ralph init` end-to-end(新規 / 既存ファイルあり)、pack 付き init、`run-verify.sh` drop-in 実行
- Regression tests: 既存 `internal/cli` / `internal/scaffold` / `internal/upgrade` 全テスト、shell スイート全ファイル(hooks 移設の影響で `tests/` の hook 系テストはパス更新が必要)
- Edge cases: AGENTS.md にユーザ内容のみで block なし(追記)、block 破損(据え置き + 報告)、.gitignore 末尾改行なし、イベント `.d` が空、drop-in 非実行属性ファイル(スキップ)、既存 `.claude/rules/<lang>.md`(旧 pack rule パス)が残るディレクトリへの pack 適用
- Evidence to capture: `docs/reports/test-*.md` にカバレッジと失敗分析

## Risks and mitigations

- dispatcher 化で hook 意味論が変わるリスク(stdout 連結・実行順)→ 現行スクリプトは exit code + テキスト出力のみ(Assumptions)。dispatcher 単体テスト + root 同時移行によるドッグフーディングで同 PR 内検証。JSON decision を返す hook を将来追加する場合の制約を dispatcher ヘッダに明記
- root 移行でこのリポジトリの開発ハーネス自体が壊れるリスク → スライス 5 を独立コミットにし、`run-verify.sh` + hook スモークを同スライスで必須化。壊れた場合は該当コミットのみ revert 可能
- 旧 upgrade × 新テンプレートの過渡不整合(managed block 面を旧エンジンが conflict 扱いする等)→ main 上のみ・リリースなしで下流非露出(Assumptions)。メタリポジトリは manifest を持たないため upgrade 対象外
- `tests/` の hook パス前提が広く壊れる可能性 → スライス 2 でテスト更新を同梱、full-suite で検出
- 大規模 diff(テンプレート + root の二重変更)→ walkthrough 必須。テンプレートと root の対応は check-sync が機械保証

## Rollout or rollback notes

- リリースタグは切らない(系列完了まで)。main 上の過渡状態は下流に届かない
- スライス単位 revert 可能。root 移行(スライス 5)は単独 revert でメタリポジトリのみ旧レイアウトへ戻せる(テンプレートとの乖離は KNOWN_DIFFS 一時追加で吸収可能)

## Deviations

- 【着手時に発見】`.gitignore` は HTML コメントを許容しないため、managed block マーカーはファイル種別ごとのコメント形式が必要: Markdown 面は `<!-- BEGIN RALPH MANAGED (ralph:<surface>) -->`(Phase 1 実装どおり)、`.gitignore` は `# BEGIN RALPH MANAGED (ralph:gitignore)` / `# END RALPH MANAGED`。Phase 1 の block エンジンにマーカー形式のパラメータ化(surface → marker style)を追加する(スライス 3 で実装、スペック FR-5 の形式定義に `#` 形式を追補)
- 【スライス 6 で記録】commit `8c47467`(`fix: use writef for init block-surface progress output`)は共有 worktree の index 状態により、スライス 5 のステージ済みファイル 19 件を意図せず同梱した。内容面への影響はなく(スライス 5 の変更そのものは正当かつ別コミット `7ab1d36` で正しく記録済み)、コミット境界の帰属(attribution)のみのずれ。実害なしと判断し、当該コミットの revert・squash は行わない
- 【スライス 6 で記録】`tests/test-terraform-rule-frontmatter.sh` と `scripts/check-pipeline-sync.sh` の更新は、当初のスライス割り当て(スライス 4: pack rule パス移設)には明記されていなかったが、pack rule パスが `.claude/rules/<lang>.md` → `.claude/rules/ralph/<lang>.md` へ移設されたことに伴う直接的な追従として、スライス 5 の実装中に同梱・文書化された(スライス 5 の逸脱記録を参照)

## Open questions

- AGENTS.md managed block に入れる「最低限の要点」の分量(スペック Open questions 引き継ぎ)— スライス 1 のドラフトを cross-review で判定
- 既存 KNOWN_DIFFS(`CLAUDE.md`、`model-routing.md`)を root 最小化後に解除できるか — スライス 5 で判定

## Progress checklist

- [x] Plan reviewed
- [x] Branch created(feat/overlay-scaffold-v2-p2)
- [x] Implementation started(slice SHAs: 5388a10, 24a611a, 87c31f5, 27c5714, 7ab1d36 + 8c47467)
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] Sync-docs artifact created
- [ ] PR created
