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
  - 実行属性・SKILL.md・agent の frontmatter の 3 つのループ(`for x in $(find ...)`、shellcheck SC2044)を、`find` の結果を一時ファイルに落として `while IFS= read -r` で読む形にする(subshell にしないので `fail` が `status` に伝わる。空白を含むパスでも壊れない)。一時ファイルは EXIT trap で消す。
  - `required_files` から meta-repo にしかない 3 項目(`README.md`、`docs/research/approach-comparison.md`、`docs/roadmap/harness-maturity-model.md`)を外す(25 項目になる)。理由を 1 行のコメントで書く。
- `tests/test-check-template.sh`: `GOLDEN_ENTRIES` を 25 項目に更新。hook の参照の検査の red / green のケース(存在する hook を引数付きで参照 → FAIL なし、存在しない hook を参照 → exit 1 で名前を出す)。fresh scaffold のケース(`go` があれば `go run ./cmd/ralph init --yes <tmp>` して `CI=true sh scripts/check-template.sh` が FAIL なしで exit 0、なければ SKIP)。root のケース(`CI=true` で FAIL 行なし)。
- `scripts/verify.local.sh`: shellcheck の対象の一覧に `scripts/check-template.sh` を加え、SC2044 のような警告が再び入ったら run-verify が止まるようにする(template に同じ一覧があれば同じく)。
- `docs/tech-debt/README.md` の #189 の行を削除。

## Non-goals

- `check-template.sh` の他の検査の判定の変更(実行属性、SKILL.md、agent の frontmatter はループの書き方だけを直し、判定は変えない。git hook の導入の検査は触らない)。
- hook のコマンドにパスの空白やクォートがある形の対応(settings の `command` は ralph が生成する固定の形)。
- 外した 3 項目を別の場所で検査すること(meta-repo の README とこの 2 つの docs は、検査の対象にする意味が薄い)。
- `scripts/bootstrap.sh` の変更。

## Assumptions

- settings の hook のコマンドは `"./.claude/hooks/<file> <args...>"` の形で、パスに空白を含まない。
- fresh scaffold のケースは `go` がある環境(CI と開発機)で走る。`go run` のビルドに数十秒かかりうる。

## Affected areas

- `scripts/check-template.sh`、`templates/base/scripts/check-template.sh`
- `scripts/verify.local.sh`(template に同じファイルがあれば同じく)
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
- [ ] AC-7: 3 つのループの失敗が exit code に伝わる: 実行属性のない `.sh`、SKILL.md のない skill、`tools:` のない agent の fixture で、それぞれ exit 1 と該当の FAIL 行になる。空白を含むパスのスクリプトも 1 つのパスとして扱われる。mutation: いずれかのループを `find ... | while read` の subshell にすると該当ケースが落ちる。
- [ ] AC-6: `docs/tech-debt/README.md` の #189 の行が削除されている。`shellcheck -S warning scripts/check-template.sh tests/test-check-template.sh` が警告なし、`scripts/verify.local.sh` の shellcheck の対象に `scripts/check-template.sh` が入っている、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`、`./scripts/check-sync.sh` が green。

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

## Deviation notes

- 2026-09-30 plan: Codex plan advisory(gpt-6-astra、xhigh、watchdog の 1 行、`codex rc=0`、`-o` 1074 バイト)は MEDIUM 1: 既存の 3 つのループが shellcheck SC2044 で `-S warning` が exit 1 になり、AC-6 が scope のままでは通らない(`check-template.sh` は `verify.local.sh` の shellcheck の対象外なので run-verify は止まらない)。ユーザー決定: 3 つのループも直す。ループを一時ファイル経由の `while read` にし、失敗が exit code に伝わることをテストで確かめ(AC-7)、`check-template.sh` を `verify.local.sh` の shellcheck の対象に加える
- 2026-09-30 work: Slice A は implementer(sonnet)に委譲(d66be51、5 ファイル、+257 / -31、push 済み)。一時ディレクトリを EXIT trap で消す形で、4 つの検査(実行属性、SKILL.md、agent の frontmatter、hook の参照)を `find` / `grep` の結果を一時ファイルに落として `while IFS= read -r ... < file` で読む形にした(subshell にしないので `fail` が `status` に伝わる)。hook のパスは `${hook_cmd%% *}`。`required_files` は 25 項目(meta-repo にしかない 3 項目を外し、理由をコメント)。`verify.local.sh` の shellcheck の対象に `scripts/check-template.sh` を追加。テストはケース D(root で FAIL なし)、E(引数付きの既存 hook は通る、存在しない hook は exit 1 でパスを出す)、F(3 つのループの失敗が exit 1 に伝わる、空白を含むパス)、G(`go run ./cmd/ralph init --yes` の fresh scaffold で FAIL なし)を追加、37 / 0。逸脱: `find` の後に `2>/dev/null || true` を付けた(探索先のディレクトリがないときに `set -e` で止まらないため。以前の `$(find ...)` と同じ許容)。red: subshell に戻す → E の存在しない hook が exit 0、パス全体を使う → E の既存 hook が FAIL、ループを `| while` に戻す → F が exit 0、README.md を戻す → A と G が落ちる。`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。orchestrator も 37 / 0、root の rc 0、shellcheck、cmp を確認
- 2026-09-30 self-review(cycle 1、375d6fa): CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 8、merge 可。MEDIUM: 配布されるコメントに "meta-repo-only" と "(issue #189)" が残る(scaffold 先ではその project の issue を指す)。LOW: `find` の `2>/dev/null || true` が読めないサブツリーを黙って飛ばす、`grep | tr` の終了コードが grep を隠し settings.json が読めないと hook を 1 つも検査しない、dispatcher がないと同じ FAIL が 7 行、コメントとテストの見出しの不正確さ、`embed_test.go` のコメントの古い例、go がないときの G を PASS と数える、F4 の grep 判定が効いていない、SIGTERM で一時ディレクトリが残る(dash)。全件を in-cycle で修正
- 2026-10-01 work: Slice B は implementer に委譲(8fd5113、4 ファイル、+215 / -49、push 済み。途中で API の接続エラーで止まり、再開した)。配布されるコメントは issue 番号と meta-repo の語を外した。`find` には存在する探索先だけを渡し、非 0 なら「could not list ...」で FAIL。settings.json は grep の rc を見て 2 以上なら FAIL、パスは `cut -d " " -f 1 | sort -u` で重複を除く。INT / TERM / HUP の trap を追加。テストは SKIP を PASS と別に数え、F4 は `^FAIL:` がないことで判定、(a) 読めないサブツリー、(b) 読めない settings.json、(c) 複数の event から参照される存在しない dispatcher で FAIL が 1 行、を追加。40 / 0 / skip 0。red: `2>/dev/null || true` に戻す → (a)、settings の検査を外す → (b)、`sort -u` を外す → (c) が落ちる。dash でも root と fixture で同じ挙動。既知の gap: シグナルの trap には専用のテストがない(タイミングに依存して不安定になるため)。orchestrator も 40 / 0、root の rc 0、cmp、purity を確認
## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 2 つの不具合を root と fresh scaffold で確認した(#183 の probe)
- [x] critical fork なし
- [x] Codex plan advisory(MEDIUM 1、対応案で plan を更新)
- [x] AC は fixture と fresh scaffold のテストで確認できる
