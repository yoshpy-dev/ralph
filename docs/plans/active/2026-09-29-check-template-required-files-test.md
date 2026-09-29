# check-template-required-files-test

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-29
- Related request: `scripts/check-template.sh` の `required_files` の一覧には回帰テストがない。#169 の `/test`(mutation 6f)で、一覧から `scripts/xreview-helpers.sh` を落としても既存のテストはどれも落ちなかった。`internal/scaffold/embed_test.go` の `TestTemplateBaseScriptsExist` は Go 側の別の一覧で、shell 側とは連動していない(`docs/tech-debt/README.md` に記録済み)。issue #183
- Related issue: 183
- Type: test
- Branch: test/check-template-required-files-test

## Objective

`check-template.sh` の `required_files` から項目を 1 つ落とすとテストが落ちる状態にし、Go 側の一覧(`TestTemplateBaseScriptsExist` の `required`)と shell 側の一覧の不一致をテストで検出する。

## 調査で確認したこと(2026-09-29、main 4794808)

- `scripts/check-template.sh`(template と byte 一致、`check-sync.sh` の同期対象)の `required_files` は 18 項目: `README.md`、`AGENTS.md`、`CLAUDE.md`、`.claude/settings.json`、docs 2 件、`scripts/` 配下 12 件。実行はカレントディレクトリに対して行い、呼び出し元は `scripts/bootstrap.sh` と meta-repo の `.github/workflows/check-template.yml`。
- Go 側の `required`(`internal/scaffold/embed_test.go:62-85`)は `templates/base/scripts/` に存在すべき 22 本のスクリプト名。shell 側の 12 本はすべて含まれ、Go 側だけにあるのは 10 本(`run-static-verify.sh`、`run-test.sh`、`detect-changed-languages.sh`、`detect-languages.sh`、`new-feature-plan.sh`、`codex-check.sh`、`ralph-config.sh`、`ralph-worktree.sh`、`check-template.sh`、`check-skill-sync.sh`)。いずれも skill や `run-verify.sh` から呼ばれる core のスクリプトで、`ralph init` / `upgrade` が必ず配る。
- `templates/base/scripts/` には他に 6 本(`check-coverage.sh`、`check-pipeline-sync.sh`、`gc-artifacts.sh`、`insights-append.sh`、`ralph-common.sh`、`sync-skills.sh`)があり、どちらの一覧にもない。
- `tests/test-*.sh` は `scripts/verify.local.sh` が全件実行し、shellcheck も同じ glob で掛かる。ただし runner(`run_hook_tests`、213 行)は実行権限のないファイルを黙ってスキップし、`bootstrap.sh` の権限付与にも `tests/` は含まれない(Codex advisory)。`tests/test-check-template.sh` はまだない。
- fresh scaffold(`go run ./cmd/ralph init --yes <tmpdir>`)には 30 本のスクリプトが配られ、Go 側だけにあった 10 本もすべて存在する。一方 `check-template.sh` 自体は main の時点で fresh scaffold を通らない: `README.md`、`docs/research/approach-comparison.md`、`docs/roadmap/harness-maturity-model.md` は template に配られず(`.gitkeep` だけ)、settings の hook 参照の検査は引数込みのコマンド文字列を拾って誤検出し、`| while` の subshell で `status=1` が捨てられる(root の CI は 7 行の FAIL を出しつつ exit 0)。これらは #189 に起票した(#183 の範囲外)。

## Scope

- `tests/test-check-template.sh`(新規): (A) `check-template.sh` の `required_files` ブロックを読み、テスト内の期待一覧(golden)と一致することを確認する。(B) 必須ファイルをすべて持つ fixture ディレクトリで `check-template.sh` が pass し、golden の各項目を 1 つずつ消すと `Missing required file: <item>` で exit 1 になることを確認する(golden が駆動するので、script 側の一覧から項目が落ちても検出する)。
- `internal/scaffold/embed_test.go`: `templates/base/scripts/check-template.sh` の `required_files` ブロックを読み、`scripts/` 配下の項目(prefix を除いた名前の集合)が Go 側の `required` と等しいことを確認するテストを追加する。どちらの一覧から項目を落としても落ちる。
- 一覧の整合: shell 側の `required_files` に Go 側だけにあった 10 本を追加する(root と template の両方、byte 一致)。これにより「scaffold された project に必須の scripts」と「template に必ず入る scripts」が同じ集合になる。
- `docs/tech-debt/README.md` の該当行を削除。`docs/quality/quality-gates.md` と `docs/architecture/repo-map.md` の `check-template.sh` の記述を確認し、必要なら新しいテストを 1 文で加える。

## Non-goals

- どちらの一覧にもない 6 本(`check-coverage.sh` など)を必須にするかどうかの判断。今回は両方の一覧に「入っているもの」を一致させるだけにとどめる(Open questions に記録)。
- `check-template.sh` の他の検査(実行属性、SKILL.md、agent の frontmatter、settings の hook 参照、git hook の導入)のテストと修正。fixture は pass する最小構成にするだけ。settings の hook 参照の検査の誤検出と fail-open、`required_files` の meta-repo 専用 3 項目(fresh scaffold で必ず red)は #189 で扱う。
- Go 側のテストが `check-template.sh` を実際に scaffold へ実行すること(sh への依存を Go テストに持ち込まない)。fresh scaffold 全体が `check-template.sh` を通ることは #189 の対象(main の時点で通らない)。

## Assumptions

- `required_files` の書式(`required_files="` の次の行から `"` の行まで、1 行 1 項目)は維持される。テストはこの書式を前提に読む(書式が変わればテストが落ちて気づける)。
- root の `scripts/check-template.sh` と `templates/base/scripts/check-template.sh` は `check-sync.sh` が同一に保つ。Go テストは template 側を読む(配布物が対象)。shell テストは root 側を読む。
- 10 本を必須に加えても、`ralph init` の scaffold と meta-repo の root にはすべて存在する(2026-09-29 に fresh scaffold で確認済み。実装時にも記録する)。

## Affected areas

- `tests/test-check-template.sh`(新規)
- `internal/scaffold/embed_test.go`(テスト追加)
- `scripts/check-template.sh`、`templates/base/scripts/check-template.sh`(一覧に 10 本追加)
- `docs/tech-debt/README.md`(行の削除)、`docs/quality/quality-gates.md` / `docs/architecture/repo-map.md`(必要なら 1 文)

## Design decisions

- source of truth は shell 側の `required_files`(project の中で実際に走る検査)。Go 側の `required` は明示的な一覧として残し、導出はしない(grep で読める contract を保つ)。両者は集合として等しいことをテストで守る。
- shell テストは golden(テスト内の期待一覧)で駆動する。script 側の一覧を読んで自分自身と比べるだけでは「落とした項目」を検出できないため。golden の更新は一覧の変更と同じ PR で行う。
- Go テストは `templates/base/scripts/check-template.sh` をテキストとして読み、`scripts/` prefix の項目を抜き出して比べる。`sh` は起動しない。
- Critical forks: None(一覧の整合は Go 側 10 本の追加で行う。逆方向(Go 側を 12 本に減らす)は配布物の検査を弱めるので採らない)

## Acceptance criteria

- [ ] AC-1: `tests/test-check-template.sh` があり、`scripts/check-template.sh` の `required_files` から任意の 1 項目を落とすと、golden の比較(A)と fixture の削除ループ(B)の両方が落ちる(scratch のコピーで全項目について mutation を回し、report に残す)。
- [ ] AC-2: `internal/scaffold/embed_test.go` に、template 側 `check-template.sh` の `scripts/` 項目の集合と Go 側 `required` が等しいことを確かめるテストがあり、どちらか一方から 1 項目を落とすと落ちる(両方向の mutation を report に残す)。不一致の出力は「どちらにだけある名前」を列挙する。
- [ ] AC-3: `required_files` に Go 側だけにあった 10 本を追加し、root と template が byte 一致。meta-repo の root で `CI=true ./scripts/check-template.sh` の exit code が今までどおり 0 で、`Missing required file` の行が出ない。`go run ./cmd/ralph init --yes <tmpdir>` で作った fresh scaffold に 10 本すべてが存在し、そこでの `check-template.sh` の `Missing required file` の行が #189 の 3 項目(`README.md`、docs 2 件)だけである(scaffold 全体の pass は #189 の対象)。
- [ ] AC-4: fixture の検査(B)は `check-template.sh` の他の検査に引っかからない最小構成で pass する(`.claude/settings.json` は `{}`、`.claude/hooks` / `.claude/skills` / `.claude/agents` / `packs` / `scripts` は空ディレクトリ、`.git` なし)。
- [ ] AC-5: `bash -n` / `shellcheck -S warning tests/test-check-template.sh` 警告なし、`go test ./internal/scaffold/...` と `TMPDIR=/tmp go test ./internal/scaffold/...` green、`./scripts/run-verify.sh` green。
- [ ] AC-6: `docs/tech-debt/README.md` の該当行が削除され、#189 の内容(hook 参照検査の誤検出と fail-open、meta-repo 専用の 3 項目)が tech-debt の 1 行として #189 を指して記録され、`docs/quality/quality-gates.md` / `docs/architecture/repo-map.md` の記述が最終の挙動と一致する。
- [ ] AC-7: `tests/test-check-template.sh` は Git mode `100755` でコミットされる(`git ls-files -s tests/test-check-template.sh` で確認)。scratch のコピーでこのテストをわざと落とした(golden に存在しない項目を足すなど)状態で `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` が非ゼロで終わり、失敗の出力にこのテストの名前が出る(runner が実行権限のないファイルを黙ってスキップするため)。

## Implementation outline

1. Slice A(implementer、sonnet): 一覧の整合(両コピー)、`tests/test-check-template.sh`(A / B、`chmod +x`)、Go の等価テスト、tech-debt の行の削除と #189 の行の追加。1 コミット。red の証拠: scratch のコピーで shell 側の各項目を 1 つずつ落として A / B が落ちること、Go 側から 1 項目落として等価テストが落ちること、shell 側から `scripts/` の 1 項目を落として等価テストが落ちること、テストをわざと落としたコピーで `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` が非ゼロになること。fresh scaffold に 10 本が存在することと `check-template.sh` の出力の記録。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR。

## Verify plan

- Static analysis checks: `shellcheck -S warning tests/test-check-template.sh`、`bash -n`、`gofmt -l internal/scaffold`、`go vet ./internal/scaffold/...`、`./scripts/run-static-verify.sh`、`./scripts/check-sync.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-6 を該当行と実行結果で確認。golden の内容が script の一覧と一致すること。
- Documentation drift to check: `docs/tech-debt/README.md`、`docs/quality/quality-gates.md`、`docs/architecture/repo-map.md`、`scripts/check-template.sh` のコメント。
- Evidence to capture: `docs/evidence/verify-*.log`、verify report の AC 表。

## Test plan

- Unit tests: `bash tests/test-check-template.sh`、`go test ./internal/scaffold/... -run 'TestTemplateBase'`。
- Integration tests: `./scripts/run-verify.sh`、fresh scaffold で `./scripts/check-template.sh`。
- Regression tests: mutation(shell 側全項目、Go 側 1 項目、shell 側 scripts 1 項目)。
- Edge cases: `required_files` ブロックの前後の空行、`scripts/` 以外の項目(README.md など)は Go 側の比較から除外されること、template 側と root 側の一覧が違う状態(check-sync が止めることの確認)、テストの実行権限がない状態を runner が黙って飛ばすこと(AC-7 の理由)。
- Evidence to capture: test report(件数、mutation 表、fresh scaffold の結果)。

## Risks and mitigations

- 10 本を必須にしたことで downstream の `check-template.sh` が厳しくなる: いずれも `ralph init` / `upgrade` が配る core のスクリプトで、欠けていれば skill が動かない。fresh scaffold で pass することを確認する。
- golden の二重管理: 一覧の変更と golden の更新が同じ PR に入ることをテストが強制する(落ちる)。
- Go テストが shell の書式に依存する: 書式が変わればテストが「項目 0 件」で落ち、原因が分かる文言にする。

## Rollout or rollback notes

テストと一覧の変更のみ。問題があれば 1 コミットを revert する。

## Open questions

- どちらの一覧にもない 6 本(`check-coverage.sh`、`check-pipeline-sync.sh`、`gc-artifacts.sh`、`insights-append.sh`、`ralph-common.sh`、`sync-skills.sh`)を必須にするか。`ralph-common.sh` は他のスクリプトから `source` されるので候補。今回は判断せず、後続の判断材料として記録する。

## Deviation notes

- 2026-09-29 plan: fresh scaffold の probe で `check-template.sh` が main の時点で通らないことを発見(meta-repo 専用の必須 3 項目、hook 参照検査の誤検出と fail-open)。#183 の範囲外として #189 に起票し、AC-3 を「10 本の存在と root の exit 0」に修正。Codex plan advisory(gpt-6-astra、xhigh)は MEDIUM 1: 新規テストが実行権限なしだと `verify.local.sh` が黙ってスキップし CI で走らない → AC-7(`100755` と、わざと落としたコピーで `verify.local.sh` が非ゼロ)を追加。ユーザー決定: 対応案で plan を更新
- 2026-09-29 work: Slice A は implementer(sonnet)に委譲(26c247f、5 ファイル、+330 / -29、push 済み)。`required_files` は 28 項目(非スクリプト 6 + `scripts/` 22、Go 側と同じ順)、root と template は byte 一致。`tests/test-check-template.sh`(mode 100755、32 assertion): A = golden との順序込みの一致(空ブロックは FAIL)、B = fixture で pass + 28 項目を 1 つずつ外して `Missing required file` を検出、C = repo root に全項目が存在。Go 側は `required` を package 変数 `requiredTemplateScripts` にし、`TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles` が template 側の `scripts/` 項目の集合と等しいことを確認(不一致は「only in check-template.sh / only in the Go list」で列挙、0 件なら書式変更として Fatal)。tech-debt は該当行を削除し #189 の行を追加。red: shell 側 28 項目すべてで A / B が落ちる、Go 側と template 側からそれぞれ 1 項目落として等価テストが落ちる、golden をわざと壊したコピーで `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` が exit 1(chmod -x にすると黙って exit 0 = AC-7 の根拠)。fresh scaffold には 22 本すべてあり、`check-template.sh` の `Missing required file` は #189 の 3 項目だけ。orchestrator も 32 / 0、`go test`、cmp、shellcheck、mode を確認。逸脱: golden の非スクリプト項目は 4 ではなく 6(handoff の数え間違い、plan の 18 + 10 = 28 と一致)
## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 両方の一覧と差分をコードで確認した
- [x] critical fork なし
- [x] Codex plan advisory(MEDIUM 1、対応案で plan を更新)
- [x] AC は決定的なテストで確認できる
