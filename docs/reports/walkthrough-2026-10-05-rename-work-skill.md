# Walkthrough: rename-work-skill

- Date: 2026-10-05
- Plan: docs/plans/archive/2026-10-05-rename-work-skill.md(この PR の最後のコミットで archive に移す)
- Branch: refactor/rename-work-skill(base main 4a5d7071)
- Diff: plan・報告・insight を除くと 71 files、+240 / -169。うち Go が 5 files で +79 / -15、埋め込みの雛形(`templates/`)が 30 files で +69 / -69

## 何を変えたか

標準フローの実装ステップを担うスキルの名前を `work` から `implement` に変えた。Claude Code では `/implement`、Codex では `$implement` で呼ぶ。中身の手順は変えていない。名前を選んだ理由は、insights のフェーズ名(`internal/cli/insights.go` の `phaseOrder`)が既に `implement` で、委譲先のサブエージェントが `implementer` だったことにある。

旧名の互換スタブは置いていない。同じ説明文の自動起動スキルが 2 つ並ぶと、モデルがどちらを呼ぶか決まらなくなるためで、計画の段階でメンテナが決めた。

## 読む順

コミットの順に読むと、ディレクトリの移動と参照の書き換えを分けて見られる。

1. `ea470d8d` スキル本体の移動
   - 4 面(`.claude/skills`、`.agents/skills`、それぞれの `templates/base/` 側)を `git mv` で移した。git は 4 本とも rename(類似度 95%)として追跡している。
   - `name: implement` と、本文の自己参照 4 か所を直した。
   - スキルのパスを直書きしている `scripts/check-pipeline-sync.sh` の REFS と、`.claude/agents/implementer.md` のコメントも同じコミットで直した。どちらも root と template の両方。
2. `f7dc114a` 他スキル・rules・`.codex/` 文書の参照
   - 42 files。置換は `/work` → `/implement` と `$work` → `$implement` の 2 規則だけで、追加と削除の行数は全ファイルで一致する。
   - `.claude/rules/ralph/model-routing.md` は `check-sync.sh` の KNOWN_DIFFS に入っているので、root と template を別々に直した。
3. `71a3c2a8` AGENTS・README・現行 docs
   - フェーズ名も変えた(`3. Work` → `3. Implement`、`spec/plan/work` → `spec/plan/implement`)。
   - `AGENTS.md` 2 本と `.ralph/core/AGENTS.core.md` 2 本は、`check-sync.sh` が管理ブロックを突き合わせるので同時に直した。
   - `docs/specs/2026-08-01-org-runtime.md` は本文を書き換えず、「2026-10-05 改訂」の節と FR-11 の行の印を足した。2026-09-16 と 2026-10-04 の改訂と同じ書き方。
4. `6f1fec2c` Go
   - `internal/insights/backfill.go` の `detectFlow` に、`/implement` を語の境界で判定する `implementSkillRe` を足した。`agents/implementer.md` のようなパスは standard と判定しない。旧名 `/work` の判定は、改名前に書かれたレポートのために残した。
   - `TestDetectFlow` は 7 ケース。
   - `internal/cli/init_v2_test.go` のフィクスチャと、コメント 2 か所を直した。
5. `2292014b` self-review F-1 の修正
   - 「spec, plan, work, verify」のような列挙が残っていた(README 3 行、`.codex/README.md` 2 本、`docs/research/approach-comparison.md`)。最初の grep の形では拾えなかった。計画の AC2 の grep に列挙の形を足し、6 行を直した。

## 変えていないもの

- 履歴の成果物。archive の計画、過去のレポート、evidence、insight event、`docs/specs/2026-05-07-codex-cli-parity.md`、tech-debt の RESOLVED 行が該当する。
- `implementer` サブエージェントの名前、`.harness/state/standard-pipeline/` のパス、insights の `phase: implement`。もともと `implement` 系の名前だった。
- `.codex/config.toml` の `[profiles.work]`。スキル名ではなく、設定例の profile 名。
- `ralph upgrade` が削除の後に空ディレクトリを残す挙動。

## 下流への届き方

main(改名前)のビルドで `ralph init` したプロジェクトに、このブランチのビルドで `ralph upgrade --yes` をかけて確かめた。orchestrator、verifier、tester がそれぞれ実行し、3 回とも `created: 2, updated: 23, deleted: 2` だった。

| 所有権 | ファイル | upgrade での扱い |
|---|---|---|
| core | `.claude/skills/work/SKILL.md`、`.agents/skills/work/SKILL.md` | 削除される。`implement/SKILL.md` が 2 か所に作られる |
| core | 他スキル、rules、agents、scripts、`.codex/README.md` | 置き換わる |
| block | `AGENTS.md` | 管理ブロックが `spec/plan/implement` に更新される |
| seed | `.codex/AGENTS.override.md`、`docs/insights/README.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md` | 本文は変わらない。upgrade レポートの Advisories に diff が出る |

seed は利用者の持ち物なので、upgrade は本文を書き換えない。利用者が advisory の diff を取り込む必要がある。

## 確かめたこと

- 計画の AC2 の grep(許容リストを除く)が 0 行
- `check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、`run-static-verify.sh` が exit 0
- `run-test.sh`(shell 33 ファイル 1,523 件 PASS、FAIL 0)と `go test ./... -count=1`(8 パッケージ ok)
- `TestDetectFlow` 7/7。mutation 5 件(判定の削除、`strings.Contains` への置き換え、`\b` の削除、旧名判定の削除、常に空を返す)がすべて red
- 上の表の upgrade の結果と、新規の `ralph init` で `implement` だけができること

## 確かめていないこと

- upgrade 後に残る空の `.claude/skills/work/` と `.agents/skills/work/` を、Claude Code と Codex のスキル探索が無視するか。スキルは SKILL.md 単位で探されるので、無視されると考えている
- `/implement` が Claude Code と Codex の組み込みコマンドと衝突しないか
- 下流で `work` を改変している(fork)場合の upgrade
- `/implement` が実際のセッションで起動するか。スキルは main のチェックアウトから読まれるので、マージ後に確かめる
