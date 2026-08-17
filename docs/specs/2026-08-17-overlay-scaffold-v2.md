# オーバーレイ型スキャフォールド配布と非対話 upgrade(layout v2)

## Summary

ralph の配布物を所有権ベースの 5 層オーバーレイ構造に再編し、`ralph upgrade` を「機械所有面の全置換 + managed block 更新 + 助言 diff」の非対話操作にする。ハッシュ 3-state ベースの対話型コンフリクト解消は完全廃止し、下流リポジトリのカスタマイズを構造的に upgrade の書き込み対象外へ移す。

## Background and problem

`ralph init` 済みリポジトリでは、下流でハーネス環境(CLAUDE.md、rules、skills、hooks、settings)が継続的に最適化される。現行の `ralph upgrade` はハッシュ 3 点比較(記録済みテンプレートハッシュ / ディスク実体 / 新テンプレート)で全配布面を一律に扱うため、以下の不具合報告が多発している:

- 上流(ralph メタリポジトリ)固有の記述・設定が下流に上書き配布される
- 下流で最適化済みのファイルが `--force` や誤操作で上書きされる
- カスタマイズ量に比例してコンフリクト解消プロンプトが増え、解消負担が大きい
- 解消結果がテンプレートともローカル最適化とも中途半端になり、ハーネスの調和が崩れる

### Current state

- `internal/upgrade/diff.go` が auto-update / conflict / add / remove / skip を判定し、conflict は対話プロンプト(`[o]/[s]/[d]`、baseline があれば `[a]/[k]/[e]`)で解消する
- 3-way マージ機構(`internal/upgrade/merge.go`)は存在するが、ローカル編集がテンプレート変更と完全一致する場合しか自動解決しない。ローカルのみが編集した領域も「要解決」となり、ファイル単位の三択に落ちる
- 所有権を事前宣言する手段がなく、「一度 conflict を踏んで skip する」(`Managed=false`)しかない
- `--force` は unmanaged(keep local 済み)ファイルまで再採用・上書きする
- テンプレートは CLAUDE.md / AGENTS.md / rules / skills を完成品ファイルとして配布し、下流の拡張ポイントがない
- Debian Policy が明文化するとおり、ユーザとプログラム(エージェント)の双方が日常的に編集するファイルにハッシュ比較型 upgrade を適用すると conflict は構造的に減らない

### Desired state

- `ralph upgrade` はプロンプトなしで完了し、機械所有面と managed block 内以外の変更を一切行わない
- 下流のカスタマイズは専用の層(ユーザ所有エントリ、overlay、drop-in)に置かれ、upgrade と物理的に交差しない
- upgrade の差分は全て機械所有ファイルの置換となり、PR としてレビュー可能
- 所有権の移動は明示コマンド(`eject` / `adopt`)のみで発生する
- ドリフト(core 改変、block 破損、pin 不一致)は `ralph doctor --strict` が機械判定し CI でゲートできる

## Requirements

### 層モデル(所有権と物理配置)

所有権は物理ディレクトリではなく manifest 上の属性(`core` / `fork` / `seed` / `block`)で定義する。ホストツールが探索パスを固定する面は現パスのまま機械所有とし、新規 `ralph init` と移行後のレイアウトは同一(単一レイアウト)。

| 層 | 内容 | upgrade の動作 |
|---|---|---|
| L1 core(機械所有) | `.claude/skills/`(ralph 出荷 skill)/ `.claude/agents/*.md` / `.claude/rules/ralph/`(ralph ガイダンスの移設先)/ `.claude/hooks/`(`local/` 除く、dispatcher 化)/ `scripts/`(ralph 出荷スクリプト、現パス維持)/ `.codex/`(`AGENTS.override.md` 除く)/ `.agents/skills/` ミラー / `.ralph/core/`(固定パス要求のない実体: AGENTS block 生成元、共有ライブラリ)/ 言語パック(`packs/languages/` 実体 + `.claude/rules/ralph/<lang>.md`) | 全置換。マージなし。コミット対象(gitignore しない) |
| L2 thin entry(ユーザ所有 + managed block) | `CLAUDE.md`(ralph 記述ゼロ、seed のみ)/ `AGENTS.md`(`<!-- BEGIN RALPH MANAGED -->` … `<!-- END RALPH MANAGED -->` ブロック内のみ ralph 所有)/ `.gitignore`(ralph 必須エントリの managed block) | block 内のみ更新。block 外不可侵 |
| L3 overlay(不可侵) | `.claude/rules/` の `ralph/` 以外 / core 集合外の名前のユーザ skill / `.ralph/local/`(`hooks/<event>.d/`・`verify.d/`・`test.d/` 等のコミット対象 drop-in)/ `.claude/hooks/local/`(gitignore 済みローカル)/ `.claude/settings.local.json` / `.codex/AGENTS.override.md` | 一切触らない |
| L4 pin + receipt | `.ralph/manifest.toml` v3(適用テンプレートバージョン、`meta.layout = "v2"`、パスごとの所有属性と fork 記録) | 適用完了時のみバージョン前進。`.ralph/baseline/` キャッシュは廃止 |
| L5 seed-once | `ralph.toml` / `CLAUDE.md` 最小シード / `docs/quality/` / `docs/plans/templates/` / `docs/reports/templates/` / `docs/recipes/` / `docs/insights/README.md` / `.github/workflows/verify.yml` 等 | 欠落時のみ生成。テンプレート側が変わった場合は advisory diff の表示・レポート出力のみ(自動適用しない) |

例外面の扱い:

- `.claude/settings.json`: JSON でコメント構文がなく managed block が成立しないため、**ralph 所有キー集合のディープマージ**とする。ユーザ追加の permissions エントリは保全し、ralph 所有キーのみ更新する。hooks はイベントごとに単一の dispatcher(`.claude/hooks/ralph-dispatch.sh <event>`)を指し、dispatcher が `.claude/hooks/<event>.d/`(core)→ `.ralph/local/hooks/<event>.d/`(ユーザ)→ `.claude/hooks/local/<event>.d/`(ローカル)の順で実行する。settings.json の hooks レイヤリングが「同名イベント上書き」である問題をディレクトリ側で解決する
- `scripts/run-verify.sh` / `run-static-verify.sh` / `run-test.sh`: core 実装の実行後に対応する `.ralph/local/*.d/` の drop-in を実行する拡張点を持つ
- advisory diff の定義: すべての advisory diff は「ディスク実体 vs 新テンプレート内容」の unified diff とする。「テンプレート側が変わった」ことの検出は、manifest に記録された適用時テンプレートハッシュと新テンプレートハッシュの比較で行う。baseline 廃止後も旧テンプレート内容の復元を必要としない、常に計算可能な定義である(fork の advisory も同様に「fork 実体 vs 新 core」)

### Functional requirements

- [ ] FR-1 `ralph upgrade` の非対話化: (1) L1 全置換(fork 記録のあるパスは置換抑止)→ (2) `.agents/skills/` ミラー再生成 → (3) settings.json ディープマージ → (4) managed block 更新 → (5) L5 欠落シード補充 → (6) advisory diff 収集 → (7) レポート出力(`docs/reports/upgrade-<version>-<date>.md`)→ (8) manifest 書き込み(バージョン前進は全工程成功時のみ)
- [ ] FR-2 `ralph eject <path>`: core ファイルを fork(ユーザ所有)へ移す。manifest に fork 記録(`forked_from_version` を含む)を書き、以後の upgrade は当該パスを置換せず、新 core との diff を advisory としてレポートする
- [ ] FR-3 `ralph adopt <path>|--all`: fork を core 所有へ戻す(ディスクを現行テンプレート内容で置換し fork 記録を削除)。現行 `--force` フラグは廃止し、これが唯一の再採用経路となる
- [ ] FR-4 予期しない core 改変(fork 記録なしのハッシュ不一致)の扱い: upgrade は当該ファイルを**上書きせず**「未解決 drift」としてレポートに記録し、exit code で警告する。`eject`(改変維持)または `adopt`(改変破棄)で解消するまで、当該パスは据え置き
- [ ] FR-5 managed block 仕様: 一意マーカー(`<!-- BEGIN RALPH MANAGED (ralph:<surface>) -->` / `<!-- END RALPH MANAGED -->`)、1 ファイル 1 ブロック。マーカー欠落時はファイル末尾に追記して報告。マーカー破損(片側欠落・重複)時は当該ファイルを据え置き、doctor が fail。HTML コメントを許容しないファイル種別(`.gitignore` 等)では、同じ surface キーを保ったまま行コメント形式のマーカー(`# BEGIN RALPH MANAGED (ralph:<surface>)` / `# END RALPH MANAGED`)を用いる
- [ ] FR-6 旧レイアウト移行: `ralph upgrade` が旧レイアウト(manifest v1/v2、`meta.layout` なし)を検出したら、移行計画(パスごとの分類: 置換 / 移設 / fork 保全 / 削除)をプレビュー表示し、明示確認後に実行する。git 作業ツリーがクリーンであることを前提条件とし、汚れていれば中断する。移行結果は `docs/reports/ralph-migration-<date>.md` に出力する
- [ ] FR-7 移行時の分類規則: 旧 manifest のハッシュ判定で未改変と分かったファイルは新レイアウトへ置換・移設(旧パスが廃止された場合は削除)。改変済みファイルは自動 fork 保全(対話なし)し、新 core との diff をレポートへ。unmanaged(旧 skip 記録)ファイルは fork として引き継ぐ
- [ ] FR-8 移行時の CLAUDE.md / AGENTS.md: CLAUDE.md が旧テンプレート未改変なら最小シードへ置換、改変済みなら一切触らない(ralph ガイダンスは `.claude/rules/ralph/` から供給されるため、どちらでも機能は成立)。AGENTS.md は旧テンプレート由来の未改変内容を managed block に置換し、ユーザ追記は block 外に保全する
- [ ] FR-9 `ralph doctor --strict`: (a) core 全ファイルのハッシュが埋め込みテンプレートと一致(fork 記録済みを除く)、(b) managed block の健在(マーカー整合 + block 内容が期待値と一致)、(c) settings.json の ralph 所有キーの健在、(d) 全追跡ファイルにコンフリクトマーカー不在、(e) manifest と実体の整合 — を検査し、違反があれば exit 1。`--strict` なしの doctor は同内容を警告表示に留める
- [ ] FR-10 上流固有記述の混入ガード: `templates/` 配下にメタリポジトリ固有の参照(リポジトリ名、maintainer 専用要素等)が含まれないことを検査するスクリプトを追加し、メタリポジトリの CI(`run-static-verify.sh` 系列)に組み込む
- [ ] FR-11 `ralph init`: v2 レイアウトを直接生成する(managed block 付き AGENTS.md、最小 CLAUDE.md シード、dispatcher 化 settings.json、`.ralph/local/` 骨格を含む)。既存ファイルがある場合の挙動は現行踏襲(上書きしない)
- [ ] FR-12 `ralph status`: パスごとの所有属性(core / fork / seed / block)と未解決 drift の一覧を表示する
- [ ] FR-13 対話型コンフリクト解消の完全撤去: `resolveConflict` / `resolveConflictWithBaseline` / conflict marker 編集フロー / `--diff` の対話文脈 / `.ralph/baseline/` 書き込みを削除する。`--dry-run` は新動作(置換・block 更新・advisory の予定一覧)のプレビューとして維持する

### Non-functional requirements

- [ ] NFR-1 冪等性: 同一バイナリでの `ralph upgrade` 再実行は no-op(書き込みゼロ、レポートに no-op 明記)
- [ ] NFR-2 非破壊性: upgrade / 移行のいかなる経路でも、fork・L2 block 外・L3・L5 既存ファイルの内容が失われない。移行前に git クリーン状態を強制することで全変更が git で復元可能
- [ ] NFR-3 worktree 互換: L1 はコミット対象であり、新規 worktree / クローン直後でもハーネスが完全動作する(husky の gitignore 起因の hook 消失事故の回避)
- [ ] NFR-4 ネットワーク非依存: upgrade・移行はバイナリ埋め込みテンプレートのみで完結する
- [ ] NFR-5 Codex パリティ: `.codex/` / `.agents/skills/` の機械所有化後も、既存の同期ゲート(`check-sync.sh` / `check-skill-sync.sh`)が成立し続ける

## Acceptance criteria

- [ ] AC-1 Given v2 レイアウトのリポジトリで L2 block 外・L3・L5 に任意のユーザ編集がある、when `ralph upgrade` を実行する、then プロンプトなしで完了し、機械所有ファイルと managed block 内以外は 1 バイトも変更されない
- [ ] AC-2 Given AGENTS.md の block 外にユーザ記述がある、when upgrade を実行する、then block 内のみが新テンプレート内容に更新され、block 外は保全される
- [ ] AC-3 Given settings.json にユーザ追加の permissions エントリがある、when upgrade を実行する、then ユーザエントリが保全され、ralph 所有キーのみ更新される
- [ ] AC-4 Given core の skill ファイルを fork 記録なしで改変している、when upgrade を実行する、then 当該ファイルは上書きされず「未解決 drift」としてレポートされ、`doctor --strict` が exit 1 になる
- [ ] AC-5 Given `ralph eject` 済みの core ファイル、when upgrade を実行する、then 当該ファイルは置換されず、新 core との diff が advisory としてレポートに含まれる
- [ ] AC-6 Given 旧レイアウト(v1/v2 manifest)のリポジトリに改変済み skill と未改変 rule がある、when upgrade で移行を確認・実行する、then 改変済み skill は fork として保全され diff がレポートに出力され、未改変 rule は新配置に置換され、`meta.layout = "v2"` の manifest v3 が書かれる
- [ ] AC-7 Given 移行またはアップグレードが途中で失敗する、when 再実行する、then manifest のバージョン・layout は前進しておらず、再実行で残工程が完了する
- [ ] AC-8 Given 同一バージョンで upgrade 済み、when 再度 upgrade を実行する、then 書き込みが発生しない(no-op)
- [ ] AC-9 Given 新規ディレクトリ、when `ralph init` を実行する、then 生成レイアウトが移行後リポジトリと同一構造になり、`doctor --strict` が exit 0 になる
- [ ] AC-10 Given `.ralph/local/hooks/PostToolUse.d/` にユーザ drop-in がある、when 対象イベントが発火する、then core hook の後にユーザ drop-in が実行される
- [ ] AC-11 Given `templates/` にメタリポジトリ固有参照を混入させる、when 混入ガードスクリプトを実行する、then exit 1 で検出される
- [ ] AC-12 Given 対話解消コードの撤去後、when `internal/upgrade` / `internal/cli` を検索する、then 対話プロンプト・conflict marker 編集・baseline 書き込みの経路が存在しない

## User stories

1. As a 下流リポジトリのメンテナ, I want `ralph upgrade` がカスタマイズに触れないこと, so that ハーネス最適化を失う恐れなく上流改善を取り込める。
2. As a 下流リポジトリのメンテナ, I want upgrade 差分が機械所有ファイルの置換のみで構成されること, so that upgrade を通常の PR としてレビュー・ロールバックできる。
3. As a ralph 上流のメンテナ, I want 混入ガードと doctor --strict, so that 上流固有設定の漏出と下流のサイレントドリフトを CI で防げる。
4. As a core をどうしても改造したいユーザ, I want `ralph eject` / `ralph adopt`, so that フォークの意思決定が明示的・可逆・追跡可能になる。

## Constraints

### In scope

- `internal/upgrade` / `internal/scaffold` / `internal/cli`(init, upgrade, status, doctor)の再設計
- `templates/base/` のレイアウト再編(rules 移設、AGENTS block 生成元、dispatcher、drop-in 骨格)
- manifest v3 スキーマと旧 manifest(v1/v2)からの移行
- 言語パックの新所有モデルへの追従
- 混入ガードスクリプトとメタリポジトリ CI への組み込み
- メタリポジトリ自身(dogfooding)の新レイアウトへの移行

### Out of scope

- Claude Code プラグインによる配布(rules / CLAUDE.md 内容を配布できず、Codex パリティも崩れるため見送り。将来検討)
- ハンク単位 3-way マージの改善(廃止を選択)
- 旧バージョンテンプレートの再構築・リモート取得(テンプレートはバイナリ埋め込み。pin はレシートであり再生成ソースではない)
- org runtime(`internal/org`)の変更
- `ralph pack` の UX 変更(所有モデルの内部追従のみ)

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `internal/upgrade/` | 対話解消・3-way マージ・baseline の撤去、置換/block/advisory エンジンへの書き換え | High |
| `internal/scaffold/` | manifest v3、baseline 廃止、所有属性の導入 | High |
| `internal/cli/upgrade.go` | 全面書き換え(非対話化、移行統合) | High |
| `internal/cli/init.go` | v2 レイアウト生成(block、dispatcher、local 骨格) | Medium |
| `internal/cli/doctor.go` / `status.go` | strict 検査・所有表示の追加 | Medium |
| `templates/base/` | rules 移設、AGENTS.core.md、settings.json dispatcher 化、drop-in 骨格 | High |
| `templates/base/CLAUDE.md` | 最小シード化(ガイダンスは rules へ移設) | Medium |
| 下流リポジトリ | 一回限りの確認付き移行(自動 fork 保全) | High |
| `scripts/check-sync.sh` 等の同期ゲート | 新レイアウトパスへの追従 | Low |

## Dependencies

- Claude Code の公式挙動(検証済み): `.claude/rules/` の再帰自動ロードと連結、hooks の任意パス参照、settings.json の permissions レイヤーマージ、`CLAUDE.md` `@import`(本設計では非依存)
- Codex の AGENTS.md 読み込み(managed block はこの前提のみに依存。@import 相当は不使用)
- 既存の `scripts/ralph-worktree.sh` / 同期ゲートスクリプト群

## Research findings

### Codebase analysis

- 現行 upgrade は `ComputeDiffs`(`internal/upgrade/diff.go`)+ 対話解消(`internal/cli/upgrade.go`)+ baseline 3-way(`internal/upgrade/merge.go`)。3-way はローカル編集がテンプレート変更と完全一致した場合のみ自動解決し、非重複変更も要対話となる
- `Managed=false`(事後の skip 記録)が唯一の所有権表現。事前宣言は存在しない
- `.ralph/` は gitignore されておらず manifest/baseline はコミット対象 → L1 コミット方針と整合
- `.claude/hooks/local/` の gitignore 慣行が既にあり、drop-in 設計はこれを拡張する
- `scripts/run-*.sh` はスキル・エージェント定義・rules から多数参照されており、パス維持が低コスト

### Best practices

調査 2 系統(OSS テンプレート更新ツール / Claude Code 公式機構)の要点:

- projen: 生成物 read-only + 生成マーカー + anti-tamper CI。「編集面を 1 箇所に限定し全置換」で衝突が原理的に存在しない
- husky v9: 機構を全置換ディレクトリに隔離し、ユーザ hook を素ファイルで分離。v4 の「ユーザ所有ファイル内にツール設定が同居」を捨ててマージ問題自体を消した。gitignore された機構ディレクトリが新 worktree で消える事故 → 本設計は L1 をコミット対象とする
- pre-commit: ユーザ面はバージョン 1 行。upgrade 差分の最小化の参照モデル
- oh-my-zsh / systemd drop-in / WordPress child theme: コア全置換 + 別ディレクトリ後勝ち overlay。drop-in の実行順規約が必須
- copier / cruft(現行方式に最も近い): 衝突マーカーのコミット事故、「未適用でも記録バージョンが前進する」サイレントドリフトが既知の失敗モード → FR-9(d)、AC-7 で対策
- Debian dpkg conffile: 「プログラムが機械的に書き換えるファイルを conffile にしてはならない」— エージェントが日常編集するファイルへのハッシュ 3-state 適用をやめる根拠
- ansible blockinfile / conda init / nvm: managed block の実績と痛点(マーカー破損時の重複挿入、block 内ユーザ編集の無言消滅)→ FR-5 で破損時据え置き + doctor 検知

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| ハンク単位 3-way 自動マージの強化 | 既存機構の延長で実装可 | ユーザ+エージェント双方が編集し続ける限り conflict は構造的に残る。マージエンジンの維持費 | No |
| ralph.toml での per-path ポリシー宣言 | 小さい差分で即効 | 宣言の維持責任がユーザに残る。構造問題は未解決 | No(所有属性として manifest に吸収) |
| Claude Code プラグイン配布 | ホスト標準の配布機構 | rules / CLAUDE.md 内容を配布不可、Codex パリティ崩壊 | No(将来検討) |
| CLAUDE.md への @import 行追記 | 追記が 1 行で済む | import 欠落時エラーのリスク。rules 自動ロードなら追記自体が不要 | No(rules 移設を採用) |
| 5 層オーバーレイ + 全置換 + managed block(本仕様) | conflict が起こり得ない層へ物理移動。非対話・冪等・PR レビュー可能 | 一回限りの移行コスト。AGENTS.md のみ block 依存が残る | **Yes** |

## Security considerations

- managed block 更新・settings ディープマージは既存ファイルへの書き込みであり、パストラバーサル防御(`filepath.IsLocal` 相当)を共有バリデータ `scaffold.CleanLocalRelPath` で維持する
- settings.json ディープマージは ralph 所有キー集合外に書き込まないことをテストで保証する(permissions の意図しない拡大を防ぐ)
- dispatcher が実行する drop-in はリポジトリ内パスに限定する(`.ralph/local/` / `.claude/hooks/`)。リポジトリ外パスの実行は行わない
- 移行はクリーンな git 状態を前提とするため、全変更が git で監査・復元可能

## Open questions

- `.claude/skills/` のシンボリックリンク対応は公式未確認のため、設計は実ファイル再生成方式に固定(symlink 非依存)。将来公式サポートが確認できれば `.ralph/core/` への一本化を再検討できる
- Codex が `.claude/rules/` を読むのは機構ではなく運用規約であるため、AGENTS.md managed block に残す「最低限の要点」の分量は実装時に調整する(block 肥大と規約依存のバランス)
- upgrade レポートの exit code 設計(未解決 drift 時に 0 / 非 0 のどちらを返すか)は CI 運用への影響を見て実装計画で確定する

## References

- 調査出典(OSS): copier.readthedocs.io(update / configuring)、cruft.github.io + GitHub issues #47/#49/#53/#181/#206、projen.io(workflow / ejecting)、nx.dev(automate-updating-dependencies)、angular.dev(cli/update)、github.com/ohmyzsh/ohmyzsh/wiki/Customization、developer.wordpress.org(child-themes)、kubectl.docs.kubernetes.io(kustomization)、man7.org(systemd.unit.5)、debian.org/doc/debian-policy(ch-files, conffile)、docs.ansible.com(blockinfile)、github.com/nvm-sh/nvm、typicode.github.io/husky(get-started / migrate-from-v4 / how-to)+ issues #929/#1574、pre-commit.com、chezmoi.io(include-files-from-elsewhere / target-types)
- 調査出典(Claude Code 公式): code.claude.com/docs/en/memory(@import)、settings(優先順位・permissions マージ)、plugins / discover-plugins、claude-directory(rules / skills / agents 探索、symlink)、hooks-guide / hooks(任意パス)
- 関連実装: `internal/upgrade/diff.go`、`internal/upgrade/merge.go`、`internal/cli/upgrade.go`、`internal/scaffold/manifest.go`、`internal/scaffold/baseline.go`
