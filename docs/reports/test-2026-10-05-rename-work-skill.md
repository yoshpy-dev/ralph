# Test report: rename-work-skill

- Date: 2026-10-05
- Plan: docs/plans/active/2026-10-05-rename-work-skill.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: branch refactor/rename-work-skill の HEAD 4b050be6 と base main 4a5d7071 の差分(75 ファイル)。behavioral test だけを実行した(static analysis は /verify で済んでいる)。plan の AC7 と AC4、Test plan の regression と edge case を確かめた。Test plan の integration(AC5 / AC6)も、このブランチと main のバイナリを作り直して再実行した
- Evidence: `docs/evidence/test-2026-10-05-rename-work-skill.log`(`docs/evidence/*.log` は gitignore 対象なので commit しない)。`run-test.sh` 自身が書いたログは `docs/evidence/verify-2026-10-05-050008.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(scope は既定の changed) | shell 33 ファイル(1,523 件)、Go 8 パッケージ | すべて | 0 | 1(下の注) | 235 s、rc 0 |
| `go test ./... -count=1 -coverprofile=...` | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 48 s、rc 0 |
| `go test ./... -count=1 -json`(件数の集計用) | top-level 1,022 件、subtest 480 件 | 1,020 + 480 | 0 | 2 | rc 0 |
| AC4: `go test ./internal/insights/ -run TestDetectFlow -v -count=1` | 7 subtest | 7 | 0 | 0 | 0.77 s、rc 0 |
| `detectFlow` の mutation(私が独立に作った 5 件) | 5 | 5(すべて red) | 0 | 0 | - |
| AC5 の再実行(このブランチのバイナリで `ralph init --yes`) | 1 | 1 | 0 | 0 | rc 0 |
| AC6 の再実行(main のバイナリで init、このブランチのバイナリで `ralph upgrade --yes`) | 2(未コミットと、コミット済みの fixture) | 2 | 0 | 0 | どちらも rc 0 |

- `run-test.sh` の scope は changed を指定したが、`scripts/check-pipeline-sync.sh` が分類できないため full に切り替わった(`Language scope: full fallback (unclassified:scripts/check-pipeline-sync.sh)`)。shell の 33 ファイルは `tests/test-*.sh` の全件で、ディスク上のファイル一覧と実行したファイルの一覧を `comm` で比べて差はなかった。
- shell の skip 1 件は `tests/test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」のケース。手元の git が 2.49.0 なので飛ばされ、stub の古い git で同じ経路を見ている。この変更とは関係がない。
- `run-test.sh` の中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。`templates/base/` は `internal/scaffold` に go:embed されるので、`-count=1` で全パッケージを流し直した。
- Go の skip 2 件は `internal/scaffold` の `TestBaseFS_WithMockFS` と `TestAvailablePacks_WithMockFS`。`go test` では `EmbeddedFS` が初期化されないので skip する(`internal/scaffold/embed_test.go:20,35`)。main でも同じで、この変更とは関係がない。

### shell の suite ごとの件数

| Suite | Passed / Total |
| --- | --- |
| `test-agent-models.sh` | 47 / 47 |
| `test-agent-phase-boundaries.sh` | 44 / 44 |
| `test-branch-name.sh` | 26 / 26 |
| `test-check-mojibake.sh` | 15 / 15 |
| `test-check-skill-sync.sh` | 13 / 13 |
| `test-check-template.sh` | 54 / 54 |
| `test-codex-exec-invocation.sh` | 144 / 144 |
| `test-detect-changed-languages.sh` | 83 / 83 |
| `test-detect-languages-terraform.sh` | 8 / 8 |
| `test-ensure-pr-ready.sh` | 7 / 7 |
| `test-ensure-pr-title-prefix.sh` | 13 / 13 |
| `test-gc-artifacts.sh` | 11 / 11 |
| `test-hook-wiring.sh` | 68 / 68 |
| `test-insights-append.sh` | 39 / 39 |
| `test-language-pack-monorepo-roots.sh` | 29 / 29 |
| `test-no-loop-references.sh` | 1 / 1 |
| `test-post-edit-verify.sh` | 24 / 24 |
| `test-pre-bash-guard.sh` | 4 / 4 |
| `test-ralph-config.sh` | 19 / 19 |
| `test-ralph-dispatch.sh` | 33 / 33 |
| `test-ralph-worktree.sh` | 143 / 143 |
| `test-run-verify-branch-secret-scan.sh` | 32 / 32 |
| `test-run-verify-scope.sh` | 19 / 19 |
| `test-secret-scan-branch.sh` | 246 / 246(skip 1) |
| `test-secret-scan.sh` | 123 / 123 |
| `test-self-review-scope.sh` | 64 / 64 |
| `test-sync-skills.sh` | 22 / 22 |
| `test-template-purity.sh` | 10 / 10 |
| `test-terraform-gitignore.sh` | 47 / 47 |
| `test-terraform-pack-verify.sh` | 36 / 36 |
| `test-terraform-rule-frontmatter.sh` | 11 / 11 |
| `test-verify-mode-split.sh` | 59 / 59 |
| `test-xreview-helpers.sh` | 29 / 29 |

合計 1,523 件。2026-10-04 の subagent-model-defaults の full run と同じ件数で、改名で増減した shell のテストはない。

## AC4: `detectFlow` の単体テストと mutation

`TestDetectFlow` の 7 subtest はすべて PASS した。plan が名前を挙げた 2 件の結果は次のとおり。

- `legacy_/work_name_still_standard`(`/work` を含む旧レポート → `"standard"`): PASS
- `implementer_path_is_not_the_skill`(先頭 20 行に `.claude/agents/implementer.md` しかない → `""`): PASS

ほかの 5 件(`/implement` を含む行、行末の `/implement`、`/implementer` という座席名、`ralph-pipeline` → loop、マーカーなし → 空)も PASS。

テストが実装の誤りを見分けられるかを、implementer の確認とは別に 5 件の mutation で確かめた。`internal/insights/backfill.go` を perl で 1 か所ずつ書き換えて `TestDetectFlow` を流し、毎回 `git checkout -- internal/insights/backfill.go` で戻した。最後に `git status --porcelain` が空であることを確かめた。

| Mutation | 落ちた subtest |
| --- | --- |
| M1: `\|\| implementSkillRe.MatchString(line)` を消す | `/implement` の 2 件 |
| M2: 正規表現を `strings.Contains(line, "/implement")` に替える | `implementer` の 2 件(パスと座席名) |
| M3: 旧名の `\|\| strings.Contains(line, "/work")` を消す | `legacy_/work_name_still_standard` |
| M4: 正規表現の `\b` を消す(`/implement`) | `implementer` の 2 件 |
| M5: `return "standard"` を `return ""` にする | `/implement` の 2 件と旧名の 1 件 |

5 件とも red になった。テーブルの各ケースは、新しい名前の判定、語の境界、旧名の互換をそれぞれ別に見張っている。

## Edge case: 無関係な語の過剰置換

plan のコマンドをそのまま実行した。

```
git diff main...HEAD -U0 | grep -E '^\+' | grep -niE 'implementtree|implementflow|implementspace'
```

出力は 1 行あった。`docs/reports/self-review-2026-10-05-rename-work-skill.md` の「過剰置換の有無」の行で、壊れた語がないことを書く文の中に、探した語そのものを引用している。壊れた語ではない。このレポートもコマンドを引用しているので、コミット後に同じコマンドを流すと当たりが 1 行増える。

対象を絞って 2 通り確かめた。

- `docs/reports` と `docs/plans` を pathspec で除いた同じ grep は 0 行(rc 1)
- `git diff --word-diff=porcelain` で、削除側の語に `work` を含む置換を全部数えた(`docs/reports`、`docs/plans`、`docs/insights/events` を除く)。出てきたのは 14 種類で、どれもスキル名の文脈だった。多い順に `/work` → `/implement`(100)、`work` → `implement`(11)、`$work` → `$implement`(9)、`.claude/skills/work/SKILL.md` → `.claude/skills/implement/SKILL.md`(8)、`Work` → `Implement`(5)、`spec/plan/work` → `spec/plan/implement`(5)。bare の `work` 11 件は、frontmatter の `name: work`、`3. Work (auto ...)`、`spec, plan, work, verify` のような列挙、`init_v2_test.go` の fixture で、英語の動詞や名詞を置き換えたものはない。`worktree`、`workflow`、`framework` のような複合語を書き換えた箇所もなかった

追加行の `implements` 4 件は、どれも削除行にもとから `implements` があった行(`README.md`、`docs/recipes/codex-setup.md` の root と template)で、置換で生まれた語ではない。

## Integration: AC5 / AC6 の再実行

AC5 と AC6 は /implement と /verify で確かめ済みだが、Test plan の integration にあるので、バイナリを作り直して再実行した。`go build -ldflags "-X main.Version=0.0.0-rename-test" ./cmd/ralph`(このブランチ)と、`git archive 4a5d7071` を scratchpad に展開してビルドした main のバイナリ(`0.0.0-main-4a5d707`)を使った。`HOME`、`GIT_CONFIG_GLOBAL`、`GIT_CONFIG_SYSTEM` は分離した。

- AC5: 空ディレクトリで `ralph init --yes`(rc 0)。`.claude/skills/implement/SKILL.md` と `.agents/skills/implement/SKILL.md` の 2 行目が `name: implement`。`work/` は 2 か所ともない。雛形全体に旧名の grep(`/work`、`$work`、`skills/work`、`name: work`、`spec/plan/work`)をかけて 0 件
- AC6: main のバイナリで `ralph init --yes`、このブランチのバイナリで `ralph upgrade --yes`(rc 0、`created: 2, updated: 23, deleted: 2`)
  - core: upgrade レポートの Deleted に `.agents/skills/work/SKILL.md` と `.claude/skills/work/SKILL.md`、Created に `implement/SKILL.md` が 2 か所
  - seed: Advisories 4 件(`.codex/AGENTS.override.md`、`docs/insights/README.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md`)。diff の中で `implement` を足す行と旧名を消す行は、`docs/insights/README.md` が 1 行ずつ、ほかの 3 本が 2 行ずつ。4 本の sha256 は upgrade の前後で一致した
  - block: `AGENTS.md` の管理ブロックが `spec/plan/implement/review`、`3. Implement (auto — ...)`、`spec/plan/implement flows` になった
  - 空の `.claude/skills/work/` が残った(plan の Non-goals に書かれた既知の挙動)
- 最初の AC6 の fixture では、雛形の commit-msg hook が `init` というメッセージを Conventional Commits でないとして止め、未コミットのまま upgrade した。`chore: init scaffold` でコミットした fixture でも流し直し、結果は同じだった(rc 0、`created: 2, updated: 23, deleted: 2`)。コミット済みの fixture の upgrade 後の `git status` では、変更が改名の範囲のファイル(他スキル 14 本、rules 5 本、`implementer.md`、`.codex/README.md`、`AGENTS.core.md`、`AGENTS.md`、`check-pipeline-sync.sh`、manifest)に限られ、seed の 4 本は変わっていない

## Coverage

- Statement: `go test ./... -count=1 -coverprofile` の合計は 87.0%。パッケージ別は `internal/cli` 84.7%、`internal/config` 92.3%、`internal/insights` 86.1%、`internal/org` 90.8%、`internal/org/driver` 92.0%、`internal/org/protocol` 97.9%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%。2026-10-04(org-drop-qa-seat)の記録とすべて同じ値
- `detectFlow`: 92.3%。カバーされていないのは `os.Open` が失敗したときの `return ""`(`internal/insights/backfill.go:219-221`)だけで、main にもある既存の分岐。今回足した `/implement` の判定の行は通っている
- Branch / Function: Go の標準ツールでは測っていない。shell はケースの範囲で見ている(計測ツールなし)
- Notes: `internal/insights` の値が変わらないのは、`detectFlow` の本体がもとから backfill のテストで通っていたためとみられる。`TestDetectFlow` の効果は coverage ではなく、上の mutation 5 件で確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 改名前の旧レポート(`/work` を含む)を `ralph insights backfill` が standard と判定する | 維持 | `legacy_/work_name_still_standard` が PASS、M3 で red |
| `agents/implementer.md` のパスや `/implementer` を新しいスキル名として数えない | 維持 | `implementer` の 2 件が PASS、M2 と M4 で red |
| `ralph init` の所有者の割り当て(skill は core) | 維持 | `TestExecuteInit_V2_FreshInit_LayoutAndOwners` が PASS(fixture は `skills/implement/SKILL.md`) |
| `model-routing.md` の implementer の固定文と agent frontmatter の対応(root / template) | 維持 | `tests/test-agent-models.sh` 47 / 47。改名で書き換えた「Standard flow delegation (/implement)」の節も含めて通った |
| 雛形が scaffold-green のまま(`check-template.sh` の required_files など) | 維持 | `tests/test-check-template.sh` 54 / 54 |
| skill ミラーの生成とずれの検査 | 維持 | `tests/test-sync-skills.sh` 22 / 22、`tests/test-check-skill-sync.sh` 13 / 13 |
| 撤去した Ralph Loop への参照が戻らない、雛形に meta-repo 専用の内容が混ざらない | 維持 | `tests/test-no-loop-references.sh` PASS、`tests/test-template-purity.sh` 10 / 10 |
| 下流の upgrade で `work` が消えて `implement` ができ、seed は変わらない | 維持 | AC6 の再実行(上の節) |

## Test gaps

- AC2 の旧名の grep は plan の手順にあるだけで、テストにも CI にもなっていない。`go test` の init のテストは in-memory の fixture を使い、実物の雛形を読む 2 件は `go test` では skip される。テンプレートに `skills/work/` や `/work` が戻ってきても、今のテストでは落ちない。`check-pipeline-sync.sh` の REFS は `implement/SKILL.md` がなくなれば気づくが、旧名が増えたことには気づかない。`tests/test-no-loop-references.sh` と同じ形の回帰テスト(履歴ディレクトリを除いて旧名を探す)を follow-up として提案する。マージを止める理由にはしない
- `detectFlow` が読むのは先頭 20 行だけだが、21 行目以降の `/implement` を無視することはテストしていない。`os.Open` の失敗の分岐もテストがない。どちらも main からある既存の空白
- 同じ行に `loop` と `/implement` があると loop と判定される(loop の判定が先にある)。既存の順序で、新しいテーブルには入っていない
- plan の Known gaps F-3(`refactor/implement-foo` や `/implement.md` も standard と判定される)はテストしていない。plan で受け入れ済み
- 下流で `work/SKILL.md` を改変している(fork)場合の upgrade(drift として報告し、`work` と `implement` が並ぶ)は、この cycle では流していない。plan の Assumptions にある挙動で、既存の fork の仕様
- upgrade 後に残る空の `.claude/skills/work/` を Claude Code と Codex のスキル探索が無視するかは確かめていない(plan の Non-goals で未確認とされている)
- `/implement` と `$implement` が実際のセッションで起動するかは、マージ前には確かめられない。スキルと agent 定義は main のチェックアウトから読まれる
- plan の edge case のコマンドはレポートの中の引用にも当たる(上の節)。次に同じ確認をするときは `':!docs/reports' ':!docs/plans'` を付けた形を使うと誤検出がない

## Verdict

- Pass: AC7(`./scripts/run-test.sh` rc 0、shell 1,523 件と Go 8 パッケージ。`go test ./... -count=1` rc 0、top-level 1,020 件と subtest 480 件が PASS、失敗 0)。AC4(`TestDetectFlow` 7 / 7、mutation 5 / 5 が red)。regression、edge case(報告書の引用 1 行を除いて 0 件)、AC5 / AC6 の再実行
- Fail: なし
- Blocked: なし

判定は pass。/pr に進んでよい。
