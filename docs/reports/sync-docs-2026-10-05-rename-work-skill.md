# sync-docs report: rename-work-skill

## Cycle 1

- Date: 2026-10-05
- Plan: `docs/plans/active/2026-10-05-rename-work-skill.md`
- Pipeline cycle: 1 of 2。差分は main の `4a5d7071` から branch HEAD `a627110e`(refactor/rename-work-skill)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-05-rename-work-skill.md`(`a8c137ec`。merge 可、F-1 は `2292014b` で修正済み)、
  `docs/reports/verify-2026-10-05-rename-work-skill.md`(`4b050be6`。pass、LOW 1 件 V-1)、
  `docs/reports/test-2026-10-05-rename-work-skill.md`(`a627110e`。pass)

## Summary

改名の本体は verify が確認済みなので、ここでは AC2 の grep が見ない場所を中心に、文書と契約のずれを探した。AGENTS.md の Primary loop、`ralph-workflow.md` の skill 一覧、`post-implementation-pipeline.md` の「Where this order is referenced」の 6 項目、`definition-of-done.md`、`quality-gates.md`、`docs/architecture/repo-map.md`、`.codex/` の文書、README の Quick start と Operating loop は、`/implement` で一致している。`CLAUDE.md` の「Manual-trigger skills」の文は、旧名を含まず、`/release` だけを挙げる内容のまま正しい。

直したのは次の 3 点。

1. plan の V-1。upgrade 後に残る空ディレクトリは `.claude/skills/work/` だけでなく `.agents/skills/work/` もだった。Non-goals と Evidence の 2 か所に足した。
2. `docs/specs/2026-08-01-org-runtime.md` の 2026-10-05 改訂 (a)。`/work` が残る箇所を Summary と FR-11 としていたが、影響範囲の表(127 行目)の `.claude/skills/`(work / loop / cross-review / self-review ほか)にも `work` がある。AC2 の grep は `docs/specs` を除外するので見つからなかった。読み替えの対象にこの表を足した。
3. plan の進捗。AC7 と Review / Verification / Test の 3 項目にチェックを付け、Evidence と Deviation notes と Known gaps を足した。

tech-debt には 2 行を足した(後述)。

## Changes made

| File | Change |
|------|--------|
| `docs/plans/active/2026-10-05-rename-work-skill.md` | (1) Non-goals の空ディレクトリの bullet を `.claude/skills/work/` と `.agents/skills/work/` の両方にした。Evidence の AC6 の bullet も同じ。(2) `- [ ] AC7` を `- [x] AC7` に。進捗の「Review artifact created」「Verification artifact created」「Test artifact created」にチェックを付けた。「PR created」は付けていない。(3) Evidence に AC7 の bullet(test report の数値)と成果物の bullet を追加。Deviation notes に V-1 と sync-docs D-1 を追加(V-1 の行に、PR 本文の移行手順へ「空の 2 ディレクトリは手で消してよい」を足すことを書いた)。Known gaps に旧名の grep が手動である点を追加。Status 行(`In progress`)はそのまま |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-10-05 改訂の 2 行だけを直した。(a) の読み替えの対象に影響範囲の表の `.claude/skills/`(work / loop / ...)を足し、2 つ目の bullet に「Summary と影響範囲の表には印を付けない」を書いた。この branch が足した 5 行の内側の変更で、本文と FR-11 の印には触れていない |
| `docs/tech-debt/README.md` | Entries 表の末尾に 2 行を追加(後述)。既存の行は触っていない。この branch が main に対して変えた行は、150 行目の 1 行と、今回の追加 2 行だけ |
| `docs/reports/sync-docs-2026-10-05-rename-work-skill.md` | この report |

### 追加した tech-debt 行

| 行 | 内容 | 既存の行との関係 |
|----|------|------------------|
| 154 行目 | 旧名 `work` が live な文書・rules・skill・template に戻っても止める自動検査がない。plan の AC2 の `git grep -nP` は手で流す確認で、`check-pipeline-sync.sh` の REFS は `implement` の skill ファイルがなくなれば気づくが、旧名が増えても気づかない。案は `tests/test-no-loop-references.sh` と同じ形の `tests/test-*.sh`。許可リスト(`internal/insights/backfill.go` と test の旧名判定、この表の RESOLVED 行、履歴ディレクトリ、`tests/test-check-template.sh` のシェル変数)を先に決める必要がある | 重なる行はない |
| 155 行目 | `ralph init` した直後のプロジェクトで `scripts/check-pipeline-sync.sh` が `FAIL: Referenced file missing: README.md` で落ちる。REFS に `README.md` があり、scaffold が README を持たないため。README を足した後は、5 つのパイプライン段階を README が挙げているかも見る。main 4a5d7071 でも再現するので、改名より前からある | `quality-gates.md` の「Must pass in CI」が `check-pipeline-sync.sh` を CI の実行内容として引用しているが、template の workflow は呼んでいない、という既存の行(118 行目付近)は、CI に配線されていない点だけを扱っていて、スクリプト自体が scaffold で落ちる点は書いていない。そこで重複させず、新しい行から参照した。(b) は既存の行で全部は覆われていないので追加した |

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `AGENTS.md`(root)と `templates/base/AGENTS.md`、`.ralph/core/AGENTS.core.md`(2 面) | Primary loop の `spec/plan/implement/review`、`3. Implement`、`spec/plan/implement flows` が 4 面で一致。Repo map は skill 名を個別に挙げていないので変更の対象外。`check-sync.sh` のブロック比較は PASS |
| `.claude/rules/ralph/ralph-workflow.md`(+ template) | skill 一覧に `/implement`。本文の `/implement` の記述も一致。`CLAUDE.md` の「Manual-trigger skills」は `/release` だけを挙げ、`anti-bottleneck` を別扱いにする文のままで、旧名を含まない |
| `.claude/rules/ralph/post-implementation-pipeline.md`(+ template)の「Where this order is referenced」 | 6 項目のファイルはすべて存在し、`check-pipeline-sync.sh` の REFS と一致。「(Step 13)」は `implement/SKILL.md` の 13 番(パイプライン実行)に一致 |
| step 番号を引く箇所(`subagent-policy.md` の step 6、`git-commit-strategy.md` の step 6 / 7、`.codex/AGENTS.override.md` と `.codex/README.md` の step 6) | `implement/SKILL.md` の 6(slice の委譲)、7(commit gate)に一致。番号は改名で変わっていない |
| `.claude/rules/ralph/model-routing.md`(+ template) | 「Standard flow delegation (/implement)」の節と `/implement subagent tiers` の文が新名。KNOWN_DIFF のファイルで、root と template を別々に直してある(plan の Evidence どおり)。`tests/test-agent-models.sh` 47/47 が test report にある |
| `docs/quality/definition-of-done.md`(+ template) | 見出し「standard /implement flow」と org への案内が新名。パイプライン順は変わっていない |
| `docs/quality/quality-gates.md` | 「pipeline order consistency across 6 reference files」の 6 は REFS の数(6)に一致。skill 名は挙げていない |
| `docs/architecture/repo-map.md` | `.claude/skills/implement/` の行が新名。ほかの行に旧名なし |
| `README.md` | Quick start(`/spec → /plan → /implement`、`$spec → $plan → $implement`)、Operating loop(`/implement` のノードと 3 番の説明)、`Subagents in /implement post-impl` の行、ディレクトリ図の skill 名の例が一致 |
| `.codex/AGENTS.override.md`、`.codex/README.md`(+ template) | `$implement` / `/implement` に更新済み。`.codex/config.toml` の `[profiles.work]` は別の名前空間(後述) |
| `docs/insights/README.md`(+ template)、`scripts/insights-append.sh` | `phase` の enum はもとから `implement` で、skill 名と phase 名がそろった。README の「standard `/implement` flow」が新名 |
| `internal/insights/backfill.go` の `detectFlow` | `/implement`(語の境界つき)と旧名 `/work` の両方を standard と判定する。旧名の判定が残る理由はコメントにある(2026-10-05 以前の report 用) |
| `docs/specs/2026-08-01-org-runtime.md` | 改訂節は 2026-10-04 の節の後ろにある。FR-11 の行の 2026-10-05 の印は 1 つ(2026-10-04 の印の後ろ)。上記の直しを 1 点した |
| 旧名の「AC2 の式が見ない形」(裸の `work`、`"work"`、`` `work` ``、`RALPH_WORK`、`state/work`、大文字の `Work`) | tracked ファイルで履歴を除いて探した。当たりは英語の動詞・名詞(`work tree`、`Bulk mechanical work` など)、`implement/SKILL.md` の 1 行目の動詞 `Work from the active plan`(4 面)、シェル変数 `$RALPH_WORKTREE`、codex の `[profiles.work]`、履歴の spec と tech-debt の RESOLVED 行。skill 名として残っているものはない |
| ファイル名の `work` | `git ls-files` で `worktree` / `workflow` / `framework` を除くと、この task の plan・report・insight event だけ |
| `.claude/commands/`、CHANGELOG、`.github/` | `.claude/commands/` と CHANGELOG はない。`.github/` に旧名なし(AC2 の grep が全 tracked ファイルを見ている) |
| `docs/plans/active/` | この plan だけ(ほかに `/work` を含む active plan はない) |
| AC2 の grep の再実行 | tech-debt の追加 2 行と plan・spec の編集の後も出力は 0 行。追加した行は旧名の literal(`/work`、`skills/work` など)を避けて書いたので、AC2 の式に当たらない |
| AC2b の再確認 | main との差分のうち、tech-debt の既存行の変更は 150 行目の 1 行だけ。16〜17 行目の RESOLVED 行は main と同一 |

## Found but left

- `.codex/config.toml` と `templates/base/.codex/config.toml` の `[profiles.work]`(「Example implementation profile」)。skill 名ではなく、利用者の user-level config に写す例の profile 名で、同じ名前を `tests/test-ralph-worktree.sh` と `internal/cli/doctor_codex_writable_root_test.go` の fixture と、tech-debt の 146 行目も使う。「work」という phase 名との対応は残るが、改名すると root と template の `.codex/config.toml`(byte 一致が要る)、fixture、tech-debt の行が連鎖する。plan の Non-goals にもないので、この task では触っていない。名前をそろえたい場合は別 task にする。
- `docs/research/approach-comparison.md` の 87 行目の `plan / work / review / verify` → `plan / implement / review / verify`。この文書は先頭に「歴史的文書」の注記があり、`tests/test-no-loop-references.sh` も歴史文書として除外している。ただし注記の範囲は Ralph Loop への言及で、この行は skill 層の説明であり、plan の Deviation notes (F-1) が現行の文書として直すと決めている。判断が分かれうるので、残す理由をここに書く。戻すなら 1 語。
- `docs/specs/2026-05-07-codex-cli-parity.md:42` の `.claude/skills/work/`、`docs/tech-debt/README.md` の 16〜17 行目の `/work`、`docs/plans/archive/`、`docs/reports/`、`docs/evidence/`、`docs/insights/events/`。依頼どおり履歴として触っていない。
- `docs/tech-debt/README.md` には `docs/plans/active/` を指す行が他にも多く、archive 済みの plan を指すものが混じる。この task が触る行ではないので掃除していない。追加した 2 行は `/pr` の archive 後の場所(`docs/plans/archive/2026-10-05-rename-work-skill.md (archived by /pr)`)を前方参照で書いた。
- `ralph upgrade` が削除後に残す空ディレクトリ(`.claude/skills/work/`、`.agents/skills/work/`)を Claude Code と Codex のスキル探索が無視するかは、まだ確かめていない(plan の Non-goals と verify の Coverage gaps にある未確認)。
- `/implement` が Claude Code と Codex の組み込みコマンドと衝突しないことは、plan の前提(推測)のまま。セッションが main のチェックアウトから skill を読むので、マージ前には確かめられない。
- PR 本文の移行手順(`/pr` の担当): 下流の seed 4 本の advisory を取り込むこと、fork した `work` は手で `implement` に移すこと、upgrade 後の空ディレクトリ 2 つは手で消してよいこと。plan の Risks と Deviation notes に材料を置いた。この step では書いていない。
- insight event(`./scripts/insights-append.sh --phase sync_docs`)は追記していない。`/sync-docs` の SKILL.md にも `doc-maintainer.md` にも追記の指示がなく(self-review / verify / test / cross-review の skill にはある)、`docs/insights/events/2026-10-05-rename-work-skill.jsonl` には self_review / verify / test の 3 行だけがある。追記するなら `./scripts/insights-append.sh --slug rename-work-skill --flow standard --phase sync_docs --verdict complete --source skill --cycle 1 --medium 0 --low 0` を orchestrator が実行する。
- 生きた Claude Code / Codex のセッションでの `/implement`・`$implement` の起動は確かめていない。

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | exit 0。IDENTICAL 159、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5(verify と同じ数)。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | exit 0。`13 skill(s) in lock-step` |
| `./scripts/check-pipeline-sync.sh` | exit 0。canonical source valid、REFS の 6 本がすべて `all pipeline steps referenced`(1 本目は `.claude/skills/implement/SKILL.md`) |
| `./scripts/check-template-purity.sh` | exit 0。template に meta-repo 固有の参照なし(template 側は編集していないが、念のため) |
| AC2 の `git grep -nP`(plan の式、許容リスト適用後) | 出力 0 行。tech-debt の追加行・plan・spec の編集の後に再実行した |
| `git diff --check` | exit 0 |
| tech-debt 表の追加 2 行 | 5 列(Python で backtick の外の `\|` を数えた)。表が壊れていない |
| `./scripts/secret-scan.sh --file`(この report)と `git diff \| ./scripts/secret-scan.sh --diff`(作業ツリーの編集) | どちらも exit 0。`--staged` の scan は commit 時の hook が実行し、`./scripts/secret-scan-branch.sh --strict` は push 前に orchestrator が実行する |

`go test` と `run-test.sh` はこの step では流していない。変更は文書だけで、Go と shell には触れていない。直前の test report が pass(`a627110e`)。

## Diff size (for /pr)

`git diff 4a5d7071...HEAD`(この pass の編集を含まない HEAD `a627110e`)は 76 files changed、715 insertions、169 deletions。walkthrough を書くかどうかは `/pr` が決める。この step では書いていない。

## Files changed in this pass

- `docs/plans/active/2026-10-05-rename-work-skill.md`
- `docs/specs/2026-08-01-org-runtime.md`
- `docs/tech-debt/README.md`
- `docs/reports/sync-docs-2026-10-05-rename-work-skill.md`(この report)
