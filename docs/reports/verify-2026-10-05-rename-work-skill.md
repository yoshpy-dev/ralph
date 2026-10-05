# Verify report: rename-work-skill

- Date: 2026-10-05
- Plan: docs/plans/active/2026-10-05-rename-work-skill.md
- Verifier: verifier subagent (Claude)、cycle 1
- Scope: 仕様への適合(AC1〜AC7、AC2b / AC2c を含む)、静的解析、文書のずれ。対象は `git diff main...HEAD`(base main 4a5d7071、HEAD 2292014b、74 ファイル)。plan の Evidence 節は信用せず、AC1〜AC3 と AC5 / AC6 はこの検証で再実行した。テスト(`go test` / `./scripts/run-test.sh`)は /test の担当なので実行していない
- Evidence: `docs/evidence/verify-2026-10-05-rename-work-skill.log`(`docs/evidence/*.log` は gitignore 対象なので、手元にだけ残る)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: skill 4 面が `name: implement` で存在し、`work/` が残っていない | Met | 4 面とも `implement/SKILL.md` があり、frontmatter の `name:` は `implement`。4 面の blob はすべて `bfd9f9f3`。4 面とも `work/` ディレクトリはなく、`git ls-files` にも `skills/work/` のパスはない。`git diff --name-status` で 4 本とも `R095`(rename として追跡されている) |
| AC2: 旧名の参照が残っていない(plan の拡張版コマンド) | Met | plan に書かれたコマンドをそのまま実行し、出力は 0 行。許容リストで落とした行は `docs/tech-debt/README.md:16-17`(RESOLVED 行)と `internal/insights/backfill.go:215,230`、`backfill_test.go:321-322` だけで、想定どおり。大文字小文字を無視した版の当たりは `plan/worktree` の 2 行(`Plan/work` が `-i` で `plan/worktree` に当たる誤検出)だけ。self-review F-1 の補足の式も 0 行 |
| AC2b: tech-debt の差分が 150 行目だけ | Met | `git diff -U0` のハンクは `@@ -150 +150 @@` の 1 つ。語単位の差分は `` `/work` `` → `` `/implement` `` だけ。16〜17 行目は main と同一 |
| AC2c: org-runtime spec の差分が改訂節と FR-11 の印だけ | Met | ハンクは `@@ -23,0 +24,5 @@`(「### 2026-10-05 改訂(refactor/rename-work-skill)」節、空行を含む 5 行)と `@@ -66 +71 @@`(FR-11)の 2 つ。FR-11 の新しい行は旧行で始まり、末尾に `**(2026-10-05 改訂: `/work` は `/implement` に改名)**` が付いただけ。追加した 5 行と FR-11 の末尾を除くと、残りの 184 行は main と一致する(python で行ごとに比較) |
| AC3: sync 系 3 本と `run-verify.sh` が exit 0 | Met(`run-verify.sh` は静的解析の部分だけ) | `check-skill-sync.sh`、`check-sync.sh`(DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh` は単独で実行して rc 0。`run-verify.sh` は `./scripts/run-static-verify.sh`(= `HARNESS_VERIFY_MODE=static` の `run-verify.sh`)で rc 0。既定の `all` モードは `go test` と hook のテストも回すので、テストの部分は /test に回した |
| AC4: `detectFlow` の判定と単体テスト | コード上は満たす(テストは未実行) | `internal/insights/backfill.go:210` の `/implement\b` と `:230` の判定を読んだ。`TestDetectFlow` に plan が求めるケース(`/implement` → standard、`/work` → standard、`agents/implementer.md` だけ → 空、`ralph-pipeline` → loop、マーカーなし → 空)と、行末の `/implement`、`/implementer` の 2 ケースがある。実行結果は /test で確かめる |
| AC5: このブランチのバイナリで `ralph init` すると `implement` ができる | Met(再実行) | `go build` したこのブランチのバイナリ(`0.0.0-verify2292`)で、`git init` しただけの空ディレクトリに `ralph init --yes`(rc 0)。`.claude/skills/implement/SKILL.md` と `.agents/skills/implement/SKILL.md` が `name: implement` ででき、`work` ディレクトリはどこにもない。雛形全体に AC2 の式を `git grep --no-index -P` でかけて 0 行。`AGENTS.md` の管理ブロックは `spec/plan/implement` と `3. Implement` |
| AC6: main のバイナリで init したプロジェクトをこのブランチで upgrade | Met(再実行) | main 4a5d7071 を `git archive` で一時領域に展開してビルドした(このワークツリーでは main をチェックアウトしていない。`git worktree add` も使っていない)。main ビルドで `ralph init --yes` してコミットしたあと、このブランチのビルドで `ralph upgrade --yes`(rc 0、`created: 2, updated: 23, deleted: 2`)。core: Deleted に `work/SKILL.md` が 2 面、Created に `implement/SKILL.md` が 2 面。seed: Advisories 4 件(`.codex/AGENTS.override.md`、`docs/insights/README.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md`)で、各 diff に `implement` への変更がある。4 本とも upgrade 前後で `shasum` が一致(本文は変わらない)。upgrade 後の旧名の残りは 7 行で、すべてこの 4 本の seed にあり、advisory の diff と一致する。block: `AGENTS.md` の管理ブロックが `spec/plan/implement` と `3. Implement` に更新された |
| AC7: `go test ./...` と `./scripts/run-test.sh` が通る | Not verified | /test の担当。この検証では実行していない |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | 指定の範囲は changed だったが、`scripts/check-pipeline-sync.sh` が分類されないため full に切り替わった(`full fallback (unclassified:scripts/check-pipeline-sync.sh)`)。結果として全体の静的ゲートと同じ範囲を見ている。local verifier: shellcheck OK、`sh -n` を hook 20 本、`jq -e` を settings.json 2 本、Codex の hook 単一ソース / provenance の各ガード、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`。golang: `gofmt: ok`、`go vet` は出力なし(成功)、golangci-lint `0 issues.`、staticcheck は出力なし(成功)。branch secret scan: `scanned 4a5d7071..2292014b against origin/main: clean` |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 単独実行 |
| `./scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 159、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-pipeline-sync.sh` | pass(rc 0) | REFS の 1 行目が `.claude/skills/implement/SKILL.md`。root と template の 2 本は同一(`cmp`) |
| `go build ./cmd/ralph`(このブランチと main 4a5d7071) | pass | AC5 / AC6 用のバイナリ。どちらも一時領域に出力した |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/ralph/post-implementation-pipeline.md` の「Where this order is referenced」 | Yes | root の 6 項目はすべて存在する。template 側では `templates/base/README.md` がないが、main にもない既存の状態(下の「既存の問題」を参照) |
| 現行文書の `.claude/skills/work/` への参照 | Yes | 履歴文書を除いた tracked ファイルで、当たりは `docs/specs/2026-05-07-codex-cli-parity.md:42` だけ。plan の Non-goals が履歴として残すと決めたファイル |
| AC2 の式が見ない形の旧名 | Yes | 追加で探した形: `` `work` ``、`"work"`、`Skill(work`、`work skill`、`work step`、`work phase`、`work flows`、`, work,`、日本語の「work スキル / フェーズ / ステップ」。当たりはすべて英語の動詞・名詞、シェル変数(`$workdir` など)、codex の profile 名(`profile "work"`)、履歴の spec で、直す対象はない |
| root と template の対 | Yes | template 側で変わった 30 ファイルは、すべて root 側でも変わっている。template だけの変更はない |
| `docs/specs/2026-08-01-org-runtime.md` | Yes | 改訂節は 2026-09-16 / 2026-10-04 の節の後ろに日付順で並ぶ。FR-11 の印の文言は「改訂:」で、2026-10-04 の印(「改訂で変更:」など)と少し形が違う。読み違える余地はないので、指摘にはしない |
| self-review の指摘への対応 | Yes | F-1: 6 行とも直っている(補足の式で 0 行)。F-2: `internal/cli/cli_test.go:347-351` は 77 文字以下に折り直された。F-4: `detectFlow` の doc コメントは空の `//` 行で段落が分かれた(旧名を残す理由を `:230` の直前に移す案と日付の扱いは採っていない。任意の提案なので問題にしない)。F-5: Evidence 節から scratchpad のパスが消えた。F-3 は plan の Known gaps にある。self-review が受け入れる場合に勧めたコメントかテストでの固定はしていない |
| plan の Non-goals(31 行目)と Evidence(159 行目) | No(V-1) | upgrade 後に残る空ディレクトリを `.claude/skills/work/` だけと書いているが、再実行では `.agents/skills/work/` も空のまま残った(下の V-1) |

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V-1 | LOW | plan は upgrade 後に残る空ディレクトリを `.claude/skills/work/` だけと書いている。実際には `.agents/skills/work/` も空で残る。Codex 側の skill 探索が SKILL.md のないディレクトリを無視するかは、Claude Code 側と同じく未確認 | AC6 の再実行後、`ls -la .claude/skills/work .agents/skills/work` がどちらも空のディレクトリを返した。plan の 31 行目は `.claude/skills/work/` だけを挙げ、159 行目は「空の `.claude/skills/work/` が残った」と書いている | /sync-docs で plan の 2 か所に `.agents/skills/work/` を足す。PR 本文の移行手順に「upgrade 後、空の `.claude/skills/work/` と `.agents/skills/work/` は手で消してよい」と書く |

### 既存の問題(この差分の範囲外)

`ralph init` した直後のプロジェクトで `./scripts/check-pipeline-sync.sh` を実行すると、`FAIL: Referenced file missing: README.md` で rc 1 になる。雛形に `README.md` がないためで、main 4a5d7071 のビルドで init した場合も同じ結果になった(この差分で起きた問題ではない)。`docs/tech-debt/README.md:118` は、template の CI がこのスクリプトを呼んでいない点を記録している。下流で失敗が表に出ないのは、そのためだと考えられる。未確認です。

## Observational checks

- AC5: 一時領域の空のリポジトリに、このブランチのバイナリで `ralph init --yes`。skill のディレクトリ、frontmatter、旧名の残り(0 行)、`AGENTS.md` の管理ブロックを確認した
- AC6: main ビルドで init してコミット → このブランチのビルドで `ralph upgrade --yes`。upgrade レポートの Summary は Deleted 2 / Created 2 / Updated 23 / Unresolved drift 0 / Advisories 4。Updated には core の `.codex/README.md` と `.ralph/core/AGENTS.core.md` が入っている。upgrade 後の `git status` は、core の M / D と `implement/SKILL.md` 2 本の新規、`AGENTS.md`(block)と `.ralph/manifest.toml` の更新、upgrade レポートの新規だけで、seed の 4 本は変わっていない
- 一時領域(scratchpad)のビルドと scaffold は、検証のあとに消した。`git worktree add` は使っていない

## Coverage gaps

- AC7 と、AC4 の `TestDetectFlow` の実行は /test の担当で、この検証では見ていない
- `run-verify.sh` の既定(`all` モード)のうち、テストの部分は実行していない。静的解析の部分だけを `run-static-verify.sh` で見た
- `/implement` が Claude Code や Codex の組み込みコマンドと衝突しないことは、plan の前提(推測)のままで、確かめていない
- SKILL.md のない空の `work/` ディレクトリを Claude Code と Codex が無視するかは、確かめていない(V-1)
- 下流で `work/SKILL.md` を改変している(fork)場合の upgrade は試していない。plan の Assumptions は既存の fork の仕様どおり両方が並ぶとしている
- 旧名の検出は正規表現の列挙に頼っている。AC2 の式と追加の式で 0 件だが、列挙していない言い回しが残っている可能性は消せない

## Verdict

- Verified: AC1、AC2、AC2b、AC2c、AC3(sync 系 3 本と、`run-verify.sh` の静的解析の部分)、AC5、AC6(どちらもこの検証で再実行)。静的解析は全体の範囲で pass
- Partially verified: AC4(コードとテストの有無は確認。実行は /test)
- Not verified: AC7(/test の担当)
- 判定: **pass**。指摘は LOW 1 件(V-1、plan の記述漏れ)で、マージを止めるものではない
