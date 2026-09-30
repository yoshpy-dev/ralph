# check-template-scaffold-green

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-30
- Related request: `scripts/check-template.sh` の 2 つの不具合(#183 の fresh scaffold の probe で発見)。(1) settings の hook の参照の検査が、コマンド文字列全体(`./.claude/hooks/ralph-dispatch.sh PreToolUse` のように引数込み)をパスとして `[ -f ]` にかけるので、存在する hook を 7 行の FAIL として誤検出し、しかも `| while` の subshell の中の `fail` が `status=1` を親に伝えないので exit 0 になる(fail-open)。(2) `required_files` の `README.md`、`docs/research/approach-comparison.md`、`docs/roadmap/harness-maturity-model.md` は template に配られないので、scaffold された project の PR CI(`templates/base/.github/workflows/verify.yml` の "Check template structure")が最初の PR から exit 1 になる。issue #189
- Related issue: 189
- Type: fix
- Branch: fix/check-template-scaffold-green

## Objective

meta-repo の root でも fresh scaffold でも `CI=true ./scripts/check-template.sh` が FAIL 行を出さずに exit 0 になり、settings が本当に存在しない hook を参照したときは exit 1 で止まるようにする。

## 調査で確認したこと(2026-09-30、main d754bcd)

- `scripts/check-template.sh`(template と byte 一致)の hook の参照の検査(79〜85 行付近): `grep -o '"\./.claude/hooks/[^"]*"' .claude/settings.json | tr -d '"' | while IFS= read -r hook_path; do [ -f "$hook_path" ] || fail ...; done`。settings の `command` は root も template も `./.claude/hooks/ralph-dispatch.sh <Event>` の 7 種類で、引数を含む。
- root での `CI=true ./scripts/check-template.sh`: 7 行の `FAIL: Settings file .claude/settings.json references missing hook: ./.claude/hooks/ralph-dispatch.sh <Event>` を出しつつ `Template structure looks good.` で exit 0。
- fresh scaffold(2026-09-29、`go run ./cmd/ralph init --yes <tmp>`)での `CI=true sh scripts/check-template.sh`: 上の 7 行に加えて、`Missing required file` が README.md と docs の 2 件で、exit 1。それ以外の検査(実行属性、SKILL.md、agent の frontmatter)は通る。
- scaffold 側の呼び出し元は `templates/base/.github/workflows/verify.yml` の "Check template structure"(`on: pull_request`)。GitHub Actions は `CI=true` を設定するので、git hook の導入の検査は飛ばされる。
- `required_files` の一覧は #183 で `tests/test-check-template.sh` の `GOLDEN_ENTRIES` と `internal/scaffold/embed_test.go` の `requiredTemplateScripts`(`scripts/` の項目だけ)に守られている。一覧を変えるときは 4 箇所を同時に変える(check-template.sh の 2 コピー、`GOLDEN_ENTRIES`、`requiredTemplateScripts`。今回は `scripts/` の項目を変えないので Go 側は変わらない)。
- `docs/tech-debt/README.md` に #189 の行がある(#183 で追加)。

## Scope

- `scripts/check-template.sh`(+ template、byte 一致):
  - hook の参照の検査: コマンド文字列の最初の語(空白の前まで)をパスとして取り出して存在を確かめる。subshell を使わない(`for` とファイル経由、または here-document)ので、`fail` が `status` に反映される。
  - `required_files` から meta-repo にしかない 3 項目(`README.md`、`docs/research/approach-comparison.md`、`docs/roadmap/harness-maturity-model.md`)を外す(25 項目になる)。理由を 1 行のコメントで書く。
- `tests/test-check-template.sh`: `GOLDEN_ENTRIES` を 25 項目に更新。hook の参照の検査の red / green のケース(存在する hook を引数付きで参照 → FAIL なし、存在しない hook を参照 → exit 1 で名前を出す)。fresh scaffold のケース(`go` があれば `go run ./cmd/ralph init --yes <tmp>` して `CI=true sh scripts/check-template.sh` が FAIL なしで exit 0、なければ SKIP)。root のケース(`CI=true` で FAIL 行なし)。
- `docs/tech-debt/README.md` の #189 の行を削除。

## Non-goals

- `check-template.sh` の他の検査(実行属性、SKILL.md、agent の frontmatter、git hook の導入)の変更。
- hook のコマンドにパスの空白やクォートがある形の対応(settings の `command` は ralph が生成する固定の形)。
- 外した 3 項目を別の場所で検査すること(meta-repo の README とこの 2 つの docs は、検査の対象にする意味が薄い)。
- `scripts/bootstrap.sh` の変更。

## Assumptions

- settings の hook のコマンドは `"./.claude/hooks/<file> <args...>"` の形で、パスに空白を含まない。
- fresh scaffold のケースは `go` がある環境(CI と開発機)で走る。`go run` のビルドに数十秒かかりうる。

## Affected areas

- `scripts/check-template.sh`、`templates/base/scripts/check-template.sh`
- `tests/test-check-template.sh`
- `docs/tech-debt/README.md`

## Design decisions

- meta-repo にしかない 3 項目は外す(条件付きの別の一覧にはしない)。README.md は配布すると利用者の README を上書きするので配れず、docs の 2 件も template に入れる意味がない。meta-repo 側での検査の価値も小さい。戻すのは一覧に 3 行足すだけ。
- hook のパスは、コマンド文字列の最初の空白までとする(`${cmd%% *}`)。
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: meta-repo の root で `CI=true ./scripts/check-template.sh` が FAIL 行を出さずに exit 0 で `Template structure looks good.` を出す。
- [ ] AC-2: settings が存在しない hook(例 `./.claude/hooks/missing.sh SessionStart`)を参照すると exit 1 で、FAIL 行にそのパス(引数なし)が出る。存在する hook を引数付きで参照する場合は FAIL を出さない。どちらも `tests/test-check-template.sh` の fixture で確認する。
- [ ] AC-3: `go run ./cmd/ralph init --yes <tmp>` で作った fresh scaffold で `CI=true sh scripts/check-template.sh` が FAIL 行なしで exit 0(テストで確認、`go` がなければ SKIP と明示)。
- [ ] AC-4: `required_files` が 25 項目で、`GOLDEN_ENTRIES` と一致(ケース A)、Go の `TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles` が green、root と template の `check-template.sh` が byte 一致。
- [ ] AC-5: mutation: hook の検査を `| while` の subshell に戻す → 存在しない hook のケースが落ちる、最初の語ではなく全体をパスにする → 引数付きのケースが落ちる。
- [ ] AC-6: `docs/tech-debt/README.md` の #189 の行が削除されている。`shellcheck -S warning`、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`、`./scripts/check-sync.sh` が green。

## Implementation outline

1. Slice A(implementer、sonnet): 上の Scope を 1 コミットで。red の証拠は AC-5。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR(`Closes #189`)。

## Verify plan

- Static analysis checks: `sh -n` と `shellcheck -S warning scripts/check-template.sh tests/test-check-template.sh`、`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`、`./scripts/check-sync.sh`、`go vet ./internal/scaffold/...`。
- Spec compliance criteria to confirm: AC-1〜AC-6 を該当行と実行結果で確認。
- Documentation drift to check: `docs/tech-debt/README.md`、`docs/quality/quality-gates.md`、`docs/architecture/repo-map.md`、#183 の記録(archive 済み、変えない)。
- Evidence to capture: `docs/evidence/verify-*.log`、verify report の AC 表。

## Test plan

- Unit tests: `bash tests/test-check-template.sh`、`go test ./internal/scaffold/... -count=1`。
- Integration tests: fresh scaffold のケース、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`。
- Regression tests: AC-5 の mutation。
- Edge cases: settings に hooks が 1 つもない、settings が存在しない、同じ hook を複数の event で参照する、`CI` を設定しない local 実行(git hook の検査は従来どおり)。
- Evidence to capture: test report(件数、mutation 表、fresh scaffold の結果)。

## Risks and mitigations

- fail-open を直すと、これまで exit 0 だった環境が exit 1 になりうる: 本当に存在しない hook を参照している場合だけで、それは検知すべき状態。root と fresh scaffold が green になることを AC で確認する。
- fresh scaffold のケースが遅い: `go run` は 1 回だけにし、`go` がなければ SKIP。

## Rollout or rollback notes

検査の修正と一覧の縮小だけ。問題があれば 1 コミットを revert する。

## Open questions

なし。

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 2 つの不具合を root と fresh scaffold で確認した(#183 の probe)
- [x] critical fork なし
- [ ] Codex plan advisory
- [x] AC は fixture と fresh scaffold のテストで確認できる
