# rename-work-skill

- Status: In progress
- Owner: Claude Code
- Date: 2026-10-05
- Related request: work スキルの名称を implement スキル(もしくは短縮形)に変更したい(2026-10-05 ユーザー依頼)
- Related issue: N/A
- Type: refactor
- Branch: refactor/rename-work-skill

## Objective

標準フローの実装ステップを担う `work` スキルを `implement` に改名する。Claude Code では `/implement`、Codex では `$implement` で呼ぶ。現行のドキュメント・ルール・スキル・スクリプト・Go コードにある参照をすべて新しい名前にそろえ、`ralph init` と `ralph upgrade` で下流プロジェクトにも新しい名前が届くようにする。

## Scope

- スキル本体の改名(4 か所): `.claude/skills/work/` → `.claude/skills/implement/`、`.agents/skills/work/` → `.agents/skills/implement/`、`templates/base/` 側の同じ 2 か所。frontmatter の `name:` も `implement` にする
- スキル本体の中の自己参照(`/work`)
- 他スキル・rules・agents・`.codex/` ドキュメント・`AGENTS.md`・`.ralph/core/AGENTS.core.md`・`README.md`・`docs/` の現行文書にある参照。root と `templates/base/` の両側
- `scripts/check-pipeline-sync.sh` の REFS(root と template)
- Go: `internal/cli/init_v2_test.go` のフィクスチャ、`internal/insights/backfill.go` の `detectFlow`(新旧両方の名前を認識させる)、`internal/cli/doctor.go` と `internal/cli/cli_test.go` のコメント
- 散文中のフェーズ名: `spec/plan/work` → `spec/plan/implement`、`3. Work (auto — ...)` → `3. Implement (auto — ...)` など
- `docs/specs/2026-08-01-org-runtime.md` に「2026-10-05 改訂」節を足す。この spec は日付付きの改訂節と本文の改訂の印で保守されている(2026-09-16 / 2026-10-04 改訂)ので、同じ流儀で `/work` → `/implement` を記録する。本文は書き換えず、FR-11 の行に改訂の印を付ける(consult 指摘)

## Non-goals

- 履歴文書の書き換え: `docs/plans/archive/`、`docs/reports/`、`docs/evidence/`、`docs/insights/events/`、`docs/specs/2026-05-07-codex-cli-parity.md`(改訂節の慣行がない)。当時の名前のまま残す
- `docs/tech-debt/README.md` の RESOLVED 行(16〜17 行目)。履歴として残す。現行の行(150 行目)だけ直す
- 旧名 `work` の互換スタブ(エイリアス)は置かない(Design decisions 参照)
- 下流の seed ファイル(`.codex/AGENTS.override.md`、`docs/**`)を upgrade で自動的に書き換えること。seed は利用者の持ち物で、upgrade はテンプレートの変化を advisory diff で 1 回知らせるだけという既存の仕様に従う(AC6 で advisory に載ることを確かめる)
- `ralph upgrade` が削除後に空ディレクトリ(`.claude/skills/work/`)を残す挙動の修正。`ApplyOps` は `os.Remove` だけを呼ぶ(`internal/upgrade/replaceplan.go:540`)。git は空ディレクトリを追跡せず、`check-skill-sync.sh` は SKILL.md のあるディレクトリだけを数えるので検査には影響しない。Claude Code / Codex のスキル探索が SKILL.md のないディレクトリを無視する点は推測で、未確認
- `implementer` サブエージェント・`.harness/state/standard-pipeline/` のパス・insights の `phase: implement` の名前変更(すでに `implement` 系でそろっている)
- リリース(`/release` は手動。この PR では版を切らない)

## Assumptions

- `ralph upgrade` は「テンプレートから消えた core ファイルで、manifest のハッシュと一致するもの」を削除し、テンプレートに新しく現れたファイルを作る(`internal/upgrade/replaceplan.go` の `classifyCore` で確認済み)。よって未改変の下流プロジェクトでは `work` が消え、`implement` が入る
- 下流で `work/SKILL.md` を改変している(fork 扱い)場合、`ralph upgrade` は削除せず drift として報告する。この場合は `work`(fork)と `implement`(core)が並ぶ。これは fork の既存の仕様どおり
- 下流の seed ファイルは、テンプレート側のハッシュが前回記録と変わると advisory に載る(`internal/upgrade/replaceplan.go` の `classifySeed`、417 / 465 行目)。所有権は `internal/cli/init.go` の `ownerForScaffoldPath` で決まり、`.codex/AGENTS.override.md` と `docs/**` が seed、`AGENTS.md` が block(管理ブロックは `.ralph/core/AGENTS.core.md` から更新)、それ以外のスキル・rules・agents・scripts・`.codex/README.md` は core
- `/implement` は Claude Code の組み込みコマンドとも Codex の組み込みコマンドとも衝突しない。推測で、外部仕様は未確認。プラグインの `prp-implement` は名前空間付きなので衝突しない
- `check-skill-sync.sh` は「frontmatter の `name:` がディレクトリ名と一致すること」を検査する。ディレクトリ名と `name:` を同じスライスで変える

## Affected areas

root 側(38 ファイル):
- `.claude/skills/work/SKILL.md`(移動)、`.agents/skills/work/SKILL.md`(移動)
- `.claude/skills/{audit-harness,cross-review,org,plan,pr,self-review,sync-docs}/SKILL.md` と `.agents/skills/` の同名ミラー
- `.claude/agents/implementer.md`(コメントのパス)
- `.claude/rules/ralph/{git-commit-strategy,model-routing,post-implementation-pipeline,ralph-workflow,subagent-policy}.md`
- `.codex/AGENTS.override.md`、`.codex/README.md`
- `.ralph/core/AGENTS.core.md`、`AGENTS.md`、`README.md`
- `docs/architecture/repo-map.md`、`docs/insights/README.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md`、`docs/roadmap/harness-maturity-model.md`、`docs/tech-debt/README.md`(150 行目のみ)
- `scripts/check-pipeline-sync.sh`
- `internal/cli/init_v2_test.go`、`internal/insights/backfill.go`(+ テスト追加)、`internal/cli/doctor.go`(コメント)、`internal/cli/cli_test.go`(コメント)

templates 側(30 ファイル): `templates/base/` 配下の上記と同じ構成(`.agents/skills`、`.claude/skills`、`.claude/agents/implementer.md`、`.claude/rules/ralph/` 5 本、`.codex/` 2 本、`.ralph/core/AGENTS.core.md`、`AGENTS.md`、`docs/insights/README.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md`、`scripts/check-pipeline-sync.sh`)

## Design decisions

- **スキル名: `implement`(ユーザー確定、2026-10-05)**。候補は `implement` と `impl`。`implement` を選んだ理由: insights のフェーズ名(`internal/cli/insights.go` の `phaseOrder`)が既に `implement`、委譲先のサブエージェントが `implementer` で、名前がそろう。`architecture.md` の「grep しやすい名前」にも合う。`impl` は Rust の `impl` や旧 org spec の `impl` 座席と grep で混ざる
- **互換スタブは置かない(ユーザー確定、2026-10-05、Codex 計画レビュー後)**。旧名 `work` を残すと、同じ説明文の自動起動スキルが 2 つ並び、モデルがどちらを呼ぶか不定になる。明示呼び出し専用のスタブも検討したが、スキルが 1 本増え、後で消す作業が残る。下流の seed 文書に残る旧名は upgrade の advisory diff と PR 本文の移行手順で知らせる。改名は upgrade レポート(Deleted / Created / Advisory)で利用者に伝わる
- **`detectFlow` は新旧両方を認識する**。`ralph insights backfill` は過去のレポートも読むので、`/work` の判定を消すと既存レポートの flow が空になる。`/implement` を足し、`/work` は旧レポート用に今の `strings.Contains` のまま残す。`/implement` は語の境界で区切って判定し、`agents/implementer.md` のようなパスには当てない(consult 指摘)
- Critical forks: スキル名と互換スタブの 2 点(どちらも解決済み)

## Acceptance criteria

- [x] AC1: `.claude/skills/implement/SKILL.md`、`.agents/skills/implement/SKILL.md`、`templates/base/.claude/skills/implement/SKILL.md`、`templates/base/.agents/skills/implement/SKILL.md` が存在し、frontmatter が `name: implement`。4 か所のどこにも `work/` ディレクトリが残っていない
- [x] AC2: 現行ファイルに旧名の参照が残っていない。次のコマンドの出力が空になる(許容リストの 2 種類を `grep -v` で除いたあと):
  `git grep -nP '/work\b|\$work\b|skills/work|"skills", "work"|name: work|spec/plan/work|Plan/work|\*\*Work\*\*|^[0-9]+\. Work |plan,? (and )?work\b|plan / work\b' -- . ':!docs/plans/archive' ':!docs/reports' ':!docs/evidence' ':!docs/specs' ':!docs/insights/events' ':!docs/plans/active/2026-10-05-rename-work-skill.md' ':!tests/test-check-template.sh' | grep -vE '^internal/insights/backfill(_test)?\.go:' | grep -v 'RESOLVED 2026-08-03 in refactor/org-runtime-retire-loop'`
  - 許容リスト: (1) `internal/insights/backfill.go` と `backfill_test.go` の旧名判定(旧レポート用に意図して残す)、(2) `docs/tech-debt/README.md` の RESOLVED 行 2 行(16〜17 行目、履歴)
  - `-P` を使う。macOS の `git grep -E` は `\b` を解釈しない。`tests/test-check-template.sh` の `$work` はシェル変数なので除外する。着手前の時点で 68 ファイルが該当
- [x] AC2b: 許容した履歴行が変わっていない。`git diff main...HEAD -- docs/tech-debt/README.md` の変更行が 150 行目の implementer.toml の行だけで、RESOLVED 行(16〜17 行目)に差分がない
- [x] AC2c: `docs/specs/2026-08-01-org-runtime.md` に「### 2026-10-05 改訂(refactor/rename-work-skill)」節があり、FR-11 の行に改訂の印がある。それ以外の本文の行は変わっていない(`git diff main...HEAD` で確認)
- [x] AC3: `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`./scripts/check-pipeline-sync.sh`、`./scripts/run-verify.sh` がすべて exit 0
- [x] AC4: `detectFlow` が `/implement` を含むレポートと `/work` を含む旧レポートの両方で `"standard"` を返し、先頭 20 行に `agents/implementer.md` しか含まないレポートでは `"standard"` を返さない。単体テストで確認する
- [x] AC5: このブランチのバイナリで `ralph init` した空ディレクトリに `.claude/skills/implement/SKILL.md` と `.agents/skills/implement/SKILL.md` ができ、`work` スキルはできない
- [x] AC6: main(改名前)のバイナリで `ralph init` したプロジェクトに、このブランチのバイナリで `ralph upgrade` をかけると、次のすべてが成り立つ
  - core: `.claude/skills/work/SKILL.md` と `.agents/skills/work/SKILL.md` が削除され、`implement/SKILL.md`(2 か所)が作られる。upgrade レポートの Deleted / Created に載る
  - seed: upgrade レポートの advisory に `.codex/AGENTS.override.md`、`docs/quality/definition-of-done.md`、`docs/insights/README.md`、`docs/recipes/codex-setup.md` が載り、各 diff に `implement` への変更が含まれる。これらのファイルの本文は upgrade で変わらない(seed の仕様)
  - block: `AGENTS.md` の管理ブロックが `spec/plan/implement` に更新される
- [ ] AC7: `go test ./...` と `./scripts/run-test.sh` が通る

## Implementation outline

1. **Slice A — スキル本体の改名**: 4 面とも `git mv` で `work/` を `implement/` に移す(`sync-skills.sh` は孤児ミラーを `rm -rf` するので、移動には使わない。履歴が切れる)。`name: implement` にし、本体の自己参照(`/work` → `/implement`)を直す。同じスライスで `scripts/check-pipeline-sync.sh`(root と template)の REFS と、`.claude/agents/implementer.md`(root と template)のコメントのパスを直す。`./scripts/run-verify.sh`(`check-sync` / `check-pipeline-sync` / `check-skill-sync` を含む)が通る状態でコミットする
2. **Slice B — 参照の更新(スキル・rules・Codex 文書)**: 他スキル 7 本(root / template × Claude / Codex の 4 面)、rules 5 本、`.codex/` 2 本の `/work`・`$work` を直す。ディレクトリが既にある `.agents/skills/` の本文は `scripts/sync-skills.sh` で再生成してよい
3. **Slice C — 参照の更新(AGENTS・README・docs)**: `AGENTS.md`、`.ralph/core/AGENTS.core.md`、`README.md`、`docs/` の現行文書(root と template)を直す。散文中のフェーズ名(`spec/plan/work`、`3. Work`、`**Work**`、`Plan/work`)も含める。`check-sync.sh` は root と template の `AGENTS.md` をどちらも `templates/base/.ralph/core/AGENTS.core.md` とブロック比較するので、`AGENTS.md` 2 本と `AGENTS.core.md` 2 本は同じスライスで変える。`docs/specs/2026-08-01-org-runtime.md` の改訂節もこのスライスに入れる
4. **Slice D — Go**: `init_v2_test.go` のフィクスチャを `implement` にする。`detectFlow` に語の境界付きの `/implement` を足し、テーブルテスト(`/implement` → standard、`/work` → standard、`agents/implementer.md` だけ → standard にならない、`ralph-pipeline` → loop、どれも含まない → 空)を追加する。`doctor.go` と `cli_test.go` のコメントを直す
5. 全体確認: AC2 / AC2b / AC2c の grep と diff、AC5 / AC6 の init / upgrade 実機確認を行い、証拠を plan に記録する

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt / go vet / shellcheck 等)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`./scripts/check-pipeline-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC7(AC2b / AC2c を含む)
- Documentation drift to check: AC2 の grep が許容リスト以外 0 件。`post-implementation-pipeline.md` の「Where this order is referenced」一覧が新しいパスを指している
- Evidence to capture: AC2 の grep 出力、AC2b / AC2c の diff、各 check スクリプトの出力末尾、AC5 / AC6 のコマンドと出力

## Test plan

- Unit tests: `detectFlow` のテーブルテスト(Slice D の 5 ケース)
- Integration tests: AC5(`ralph init` を一時ディレクトリで実行)、AC6(main ビルドで init → このブランチのビルドで upgrade。core の Deleted / Created、seed の advisory、block の更新を見る)
- Regression tests: `go test ./...`、`./scripts/run-test.sh`(既存の `tests/test-*.sh` を含む)
- Edge cases: `/work` が `/workspace` などの別語に含まれる箇所を誤って置換しない(`\b` で区切り、差分を目視確認)。`docs/tech-debt/README.md` の RESOLVED 行は変えない(AC2b)。`detectFlow` が `/implementer` に当たらない(AC4)
- Evidence to capture: テスト出力、AC6 の upgrade レポートの Deleted / Created / Advisory 節

## Risks and mitigations

- **置換漏れ**: 4 面(root / template × Claude / Codex)のどこかに旧名が残る → AC2 の grep を決定的なゲートにする。`check-skill-sync.sh` と `check-sync.sh` が面の間の差を検出する
- **過剰置換**: 履歴文書や無関係な `work` を書き換える → 置換対象のファイルを Affected areas に限定し、pathspec で履歴ディレクトリを除外する。差分を目視確認する
- **下流の利用者が `/work` を打って見つからない** → PR 本文に改名を明記する。upgrade レポートに Deleted / Created が出る。互換スタブは後から足せる
- **下流の seed 文書が旧名を指し続ける(Codex 計画レビュー HIGH)**: `.codex/AGENTS.override.md` と `docs/**` は seed なので upgrade では本文が変わらず、Codex が読む指示やセットアップ手順に `$work` / `/work` が残る → upgrade レポートの advisory に 4 本が載ることを AC6 で確かめる。PR 本文に移行手順(「advisory の diff を取り込む。少なくとも `.codex/AGENTS.override.md` の `$work` と `/work` step 6 は `implement` に直す」)を書く
- **下流で `work` を fork している場合に 2 つ並ぶ** → fork の既存仕様どおり。PR 本文に「fork している場合は手で `implement` に移す」と書く
- **このセッション自体が旧スキル名で動いている**: main のチェックアウトから skill が読まれるため、マージまでは `/work` で起動する。マージ後の次のタスクから `/implement` になる

## Rollout or rollback notes

- Rollout: PR マージ後、次の `/release` で下流に届く。破壊的な改名なので、リリースノートで知らせるのが望ましい(版の判断は `/release` 時にメンテナが行う)
- Rollback: PR を revert すれば root と template が元に戻る。下流で既に upgrade 済みなら、revert 後のリリースで再度 upgrade すると `implement` が消え `work` が戻る(同じ core 削除・作成の仕組み)

## Open questions

- なし

## Deviation notes

- 2026-10-05 self-review F-1(MEDIUM): 当初の AC2 の grep は、`spec, plan, work, verify`、`Spec, plan, and work flows`、`plan / work` のような列挙を拾えなかった。`README.md` 29 / 111 / 170 行目、`.codex/README.md` 48 行目(template 側も同じ)、`docs/research/approach-comparison.md` 87 行目に旧名が残っていた。`.codex/README.md` は core 扱いなので下流にも届く。AC2 に `plan,? (and )?work\b|plan / work\b` を足し、修正スライスで 6 行を直す。`docs/research/approach-comparison.md` は Affected areas になかったが、改訂の続いている現行の文書なので対象に加える
- self-review F-2 / F-4(LOW): コメントの折り返しと段落を直す。F-5(LOW): Evidence 節にあった scratchpad のパスを消した。PR 後には残らない一時領域のため

## Known gaps

- self-review F-3(LOW): `detectFlow` の `/implement\b` は、`refactor/implement-foo` や `/implement.md` にも当たる。`\b` が `-` と `.` の前でも成り立つためで、そういう文字列が先頭 20 行にあるレポートは standard と判定される。Ralph Loop が撤去された今は standard と判定されることがほぼ正しいので、直さずに残す

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Slice A: スキル本体の改名(ea470d8d)
- [x] Slice B: スキル・rules・Codex 文書の参照(f7dc114a)
- [x] Slice C: AGENTS・README・docs の参照(71a3c2a8)
- [x] Slice D: Go(6f1fec2c)
- [x] AC5 / AC6 の実機確認
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Evidence

- 4 スライスとも implementer に委譲した。各スライスの報告で、ハンドオフに書いた検証コマンドの exit 0 と、コミット後に `git status --porcelain` が空であることを確かめた。オーケストレーター側では HEAD の SHA の一致と差分を照合した
- Slice B: `.claude/rules/ralph/model-routing.md` は `check-sync.sh` の KNOWN_DIFFS に入っている。このため root と template を別々に編集し、`cp` していない。差分は `/work` → `/implement` の 1 行ずつ
- AC2: 最終の grep(許容リスト適用後)は 0 行
- AC2b / AC2c: `git diff --word-diff` で確認した。tech-debt の変更は 150 行目の `/work` → `/implement` だけ。spec の変更は 2026-10-05 改訂節の追加と、FR-11 の行末に付けた改訂の印だけ
- AC4: `TestDetectFlow` の 7 ケースがすべて PASS。implementer がミューテーション確認も行った。`/implement` の判定を外すと 2 ケースが落ち、`strings.Contains` に戻すと implementer 系の 2 ケースが落ちる
- AC5: このブランチのビルド(`-X main.Version=0.0.0-rename6f1f`)で空ディレクトリに `ralph init --yes` を実行した。`implement/SKILL.md` が 2 か所に `name: implement` でできた。`work/` はできず、雛形全体を `git grep` しても旧名は 0 件
- AC6: main 4a5d7071 のビルドで `ralph init --yes` したプロジェクトに、このブランチのビルドで `ralph upgrade --yes` を実行した(rc 0、`created: 2, updated: 23, deleted: 2`)
  - core: Deleted に `work/SKILL.md` が 2 か所、Created に `implement/SKILL.md` が 2 か所載った
  - seed: Advisories(4 件)に `.codex/AGENTS.override.md`、`docs/insights/README.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md` が載った。どの diff にも `implement` への変更が入っており、ディスク上の本文は変わっていない
  - block: `AGENTS.md` の管理ブロックが `spec/plan/implement` と `3. Implement` に更新された
  - 空の `.claude/skills/work/` が残った(Non-goals に書いた既知の挙動)

## Readiness checklist

- [x] 影響範囲を grep で洗い出した(root 38 / template 30 ファイル、計 68)
- [x] スキル名をユーザーと確定した(`implement`)
- [x] upgrade の削除・作成の挙動をソースで確認した
- [x] 決定的なゲート(AC2 の grep、sync 系スクリプト)を決めた
- [x] consult-advisor(plan)の判定「直してから進める」を反映した: org-runtime spec の改訂節、4 面とも `git mv`、`detectFlow` の 3 ケース
- [x] Codex 計画レビューの 2 件を反映した: seed の advisory 確認(AC6)と移行手順、AC2 の許容リストと AC2b
