# sync-docs report: init-pack-choice

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-28-init-pack-choice.md`(承認の digest `614133eca51f` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは AC1〜AC8 のチェックボックスと `## Progress checklist` の 3 項目だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `ae64da70`、branch は `feat/init-pack-choice`。run 開始時の HEAD は `ea7dce08`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-28-init-pack-choice.md`(7eca2cb1 から f9df1718 まで 4 回、最後の Run 4 は新しい所見なしで Merge yes)、`docs/reports/verify-2026-10-28-init-pack-choice.md`(63ccc8c8、pass、情報 D-1・V-1)、`docs/reports/test-2026-10-28-init-pack-choice.md`(2a4294c4、pass。ea7dce08 で Failure analysis と Verdict の 2 か所の文言を直した)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、この run の前): 11 ファイル、+1479/-40。walkthrough を書くかどうかは `/pr` が決める

## Summary

この branch は issue #275 の対応で、`ralph init` の言語パックの選び方を変えた。対話のフォームは最初に何も選ばず(`--pack` の名前があればそれを選んだ状態で始める)、直下に手がかりのファイルがあるパックに「(Recommended)」を付け、最後に skip を置く。`--yes` は `--pack` が名前を挙げたパックだけを入れる。知らない名前と空の名前は、何も書く前にエラーで止まる。`TERM=dumb` で stdin が端末でないときは、止まらずにエラーで終わる。パックが 0 個のときは、終わりの表示に `ralph pack add <language>` を勧める。

現在形の文書を洗った結果、新しい振る舞いと食い違う文は見つからなかった。`README.md:120`、雛形の recipe の init の段落、`ralph init --help` は、この branch の S4 が書いた時点で新しい振る舞いに合っている。直したのは、verify の情報 D-1 だけである。雛形の recipe が「知らない名前は失敗する」と書き、空の名前に触れていなかったので、「知らない名前か空の名前」に直した。

## Changes made

| File | Change |
|------|--------|
| `templates/base/docs/recipes/adding-a-language-pack.md:37-38` | 「An unknown name fails before anything is written.」を「An unknown or empty name fails before anything is written.」に直し、折り返しを合わせた。`--pack ''` と `--pack golang,` は `--yes` の有無によらず、何も書く前に失敗する(verify の「Observational checks」で確かめ済み)(D-1) |
| `docs/plans/active/2026-10-28-init-pack-choice.md` | AC1〜AC8 を `[x]` にした(verify が AC1〜AC6・AC8 を満たすとし、AC7 は test の G の PASS で満たした)。`## Progress checklist` の「Review artifact created」「Verification artifact created」「Test artifact created」を `[x]` にし、report の SHA と verdict、D-1・V-1 の扱いを書いた。ほかの節は触れていない(digest は同じ) |
| `docs/reports/sync-docs-2026-10-28-init-pack-choice.md` | この report |
| `docs/insights/events/2026-10-28-init-pack-choice.jsonl` | `sync_docs` の行を `insights-append.sh` で足した |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md:120`(コマンドの表の `ralph init` の行) | 変更なし(同期済み) | 対話の問い(`--pack` の名前を最初に選ぶ、なければ何も選ばない、(Recommended)、skip)、`--yes` は `--pack` のパックだけ(繰り返しとカンマ)、`ralph pack add` の案内が、binary の振る舞いと合う |
| `README.md:34`、`:112`、`:278-285`(Language packs の節) | 変更なし | 「opt-in」と書くだけで、`--yes` やフォームの初期値に触れない。`ralph pack add golang` の例は有効なまま |
| `README.md:58`、`:64`(Quick start の `ralph init my-project`) | 変更なし | 引数なしの対話の形で、新しいフォームでも通る |
| `README.md:152`(`ralph upgrade` の `--yes`) | 変更なし | `upgrade` の確認を飛ばす flag で、この branch の `init --yes` とは別 |
| `templates/base/docs/recipes/adding-a-language-pack.md:30-38`(init の段落) | 直した(D-1) | `--pack` の初期選択、(Recommended)、skip、`ralph init --yes --pack golang,python` の例は合う。空の名前だけ書いていなかった |
| `templates/base/docs/recipes/adding-a-language-pack.md:27` | 変更なし | `ralph upgrade` の `--yes`/`--dry-run` の話 |
| `docs/recipes/adding-a-language-pack.md`(contributor 版) | 変更なし(矛盾なし) | `scripts/check-sync.sh` の KNOWN_DIFF。`:24-26` は「選んだパックの rule だけを書く」で新しい振る舞いと合う。`:35` の `--yes` は `upgrade` の話。init の選び方は書いていない(計画の Verify plan のとおり対象外) |
| `ralph init --help`(`internal/cli/init.go:33-44`、`:83`、`:88`) | 変更なし | 依頼で Go は触らない。help は「unknown name fails」と書き、空の名前には触れない(D-1 の help 側。下の Info) |
| `AGENTS.md`、`CLAUDE.md`、`.ralph/core/AGENTS.core.md`、`templates/base/AGENTS.md`、`templates/base/CLAUDE.md` | 変更なし | `ralph init` は scaffold の入口として挙げるだけで、言語パックの既定に触れない |
| `docs/quality/`、`docs/architecture/repo-map.md` | 変更なし | `quality-gates.md:41` は changed 範囲の検証が入れたパックだけを動かす話。`repo-map.md` は init の既定に触れない |
| `.claude/skills/`、`.agents/skills/`、`templates/base/.claude/skills/` | 変更なし | `ralph init`・`--yes`・「language pack」の当たりは `org` skill の `ralph.toml` の seed-once の話、`audit-harness`・`sync-docs` の一般的な点検項目、`plan/template.md` の `RALPH_VERIFY_SCOPE` の注意で、init の既定は前提にしていない。skill は触れていないので `sync-skills.sh` は走らせていない |
| `.claude/rules/ralph/`、`templates/base/.claude/rules/ralph/` | 変更なし | `agent-messaging.md` の `ralph init` は「scaffold に含まれない」という注記だけ |
| `scripts/`、`.github/`、`tests/`(現在形のコード側の呼び出し) | 変更なし | `ralph init` を呼ぶのは `tests/test-check-template.sh` の G(`--yes --pack "$packs"`、S2 で直し済み)だけ。そのほかは scaffold の説明のコメント |
| `docs/specs/` | 変更なし | init の言語パックの選び方を書いた spec はない。`2026-04-16-ralph-cli-tool.md:296` の「pack 列挙の失敗」は `upgrade` の話 |
| `./scripts/check-sync.sh` | 通る | 下の Verification |

## Info(履歴の記録として残したもの)

`git grep -n "ralph init --yes"` が、過去の run の記録に多数当たる。どれも当時の binary で作った scaffold を述べた記録なので、書き換えない。この branch のあと同じ手順を打つと、言語パックのない scaffold ができる。

- `docs/reports/test-2026-09-29-check-template-required-files-test.md:103`: 「`ralph init --yes` ... 6 language packs」。当時は 6 パックが入った。
- `docs/plans/archive/` の 6 本(`2026-08-03-org-runtime-retire-loop.md:141`、`2026-09-29-check-template-required-files-test.md:21`、`2026-09-30-check-template-scaffold-green.md:19`・`:63`、`2026-10-02-codex-config-model-gpt-5-6-sol.md`、`2026-10-04-subagent-model-defaults.md:64`、`2026-10-05-rename-work-skill.md:157-158`): `ralph init --yes` で fresh scaffold を作った記録。
- `docs/reports/` の 19 本(verify・test・self-review・walkthrough、この branch の report を除く)と `docs/evidence/` の 7 本(`codex-hooks-livefire-*`、`codex-hooks-multi-event-fixture-2026-08-24.md`、`org-feature-worktree-live-2026-10-09.md`、`org-plan-approval-live-2026-10-13*`、`org-retire-loop-smoke-2026-08-03.txt` など): scaffold を作る手順として `ralph init --yes` を書く。これらの手順を後で再現するときは、パックが要るなら `--pack` を足す。
- `docs/tech-debt/README.md:43`: RESOLVED の行で、当時の `ralph init` の不具合を書く。
- help の D-1 の残り: `internal/cli/init.go:42-43` は「unknown name fails」と書き、空の名前に触れていない。依頼で Go は触らないので残した。振る舞いはエラーで止まる側で、利用者を誤らせる書き方ではない(verify の判断と同じ)。

## Plan drift notes

- 計画の Scope・Design decisions・Acceptance criteria の本文は digest の対象で、この run は触れていない。AC1〜AC8 は `[x]` になった。残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける。
- 実装中に計画の外へ出た点は、Progress の S1〜S3 と self-review の行に記録済み(空白を取る `TrimSpace`、skip の値 `/skip`、問いの `Height`、`TERM=dumb` の guard、help の文言の直し)。この run で新しく見つかった差はない。
- S2 の検証コマンドが macOS の `/bin/sh` で通らない点(`./tests/test-check-template.sh` で打つ)も Progress に記録済み。

## Follow-ups(この PR では足していない)

- 計画が「既知の穴」とした、`runInitInteractive` のフォームの先(初期の選択、推定の結果を問いに渡すこと、`packsFromSelection` の呼び出し)は、mutant p・q・r・s が `go test` を生き残り、pty でだけ落ちる。guard の条件(mutant g・h)も同じ。`docs/tech-debt/README.md` には行を足していない。足すかどうかは `/pr` の前に lead が決める。足すなら Trigger は「`runInitInteractive` に質問を足す(#277)とき」が自然である。
- `TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` は負荷が高いと `--timeout-ms 3000` を使い切る。test report の Failure analysis に書いたとおり、`docs/tech-debt/` にこのテストの行はない。この branch の変更に由来しないので、別の作業で行を足すか期限を直す。

## Release notes(`/pr` の本文に置く)

verify の「Release-note material」と test report の観察に基づく。

1. `ralph init --yes` は言語パックを入れなくなった。前は 6 パック(dart、golang、python、rust、terraform、typescript)すべてを入れていた。前と同じ結果にするには `ralph init --yes --pack dart,golang,python,rust,terraform,typescript`。名前は `ralph pack list` で分かる。
2. 新しい flag `--pack <name>`(繰り返しても、カンマで区切ってもよい)。知らない名前と空の名前(`--pack ''`、`--pack golang,`)は、`--yes` の有無によらず、何も書く前にエラーで止まる。
3. 対話のフォームの言語パックの問いは、最初に何も選ばない(`--pack` を渡すとその名前を選んだ状態で始まる)。対象のディレクトリの直下に `go.mod` などの手がかりのファイルがある言語には「(Recommended)」が付き、最後の選択肢 skip は言語パックを入れずに終える。何も選ばずに Enter を押しても、skip と言語を一緒に選んでも、問いは次へ進まない。
4. `TERM=dumb` で stdin が端末でないときの `ralph init`(`--yes` なし)は、前は 6 パックすべてを入れて終了コード 0 で終わった。今は何も書かずにエラーで止まり、`--yes`(と `--pack`)を勧める。
5. manifest の `packs` の並びは、`--yes` では `--pack` に書いた順、対話のフォームでは名前順になる(例: `--pack typescript --pack golang` は `--yes` で `['typescript', 'golang']`、対話で `['golang', 'typescript']`)。
6. 言語パックを 1 つも入れなかったときは、終わりの表示に `ralph pack add <language>` と使える名前の一覧が出る。

PR の Notes for reviewers に入れる点: verify の V-1(`--pack` なしで問いに入ると、何も押す前から `choose at least one language pack, or skip` が赤く出る。huh v2.0.3 の `MultiSelect.Focus()` が確かめの関数を通すため。次のキーで消える)、上の Follow-ups の 1 つ目(単体テストの外にある `runInitInteractive` の行は pty で確かめた)、`.goreleaser.yml` の changelog が `^docs:`・`^test:`・`^chore:` を除くので、自動の changelog に載るのは `feat:` と `fix:` の commit だけである(題が内容を言わない commit があれば release ノートの文を手で足す)。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(TEMPLATE_ONLY 11、KNOWN_DIFF 5)。recipe の template 側は KNOWN_DIFF なので、この編集は byte 比較の対象外 |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-28-init-pack-choice.md` | `614133eca51f`(`- Approved:` 行の値と同じ。Progress と AC の編集の前後で同じ) |
| 現在形の文書の `git grep` | `--yes`、`ralph init`、`language pack`、`ralph pack add` を、README、AGENTS、CLAUDE、`.ralph/core`、`docs/recipes`、`docs/quality`、`docs/architecture`、`templates/base/docs`、`.claude/skills`、`.agents/skills`、`templates/base/.claude/skills`、`.claude/rules`、`templates/base/.claude/rules` で打った。`--yes` がすべてのパックを入れる、最初からすべて選ばれている、と書く文はなかった |
| `git diff --check` | rc 0、出力なし(commit の前、追跡ファイルの変更に対して) |
| `go test`、`go build`、golangci-lint、`run-*.sh` | 打っていない。依頼で止められた。この run が変えたのは雛形の recipe の 1 文、計画の Progress と AC のチェックボックス、この report、insight event で、コードの動きは変わらない |
| insight event | `./scripts/insights-append.sh --slug init-pack-choice --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記した(rc 0、`docs/insights/events/2026-10-28-init-pack-choice.jsonl` の末尾の行が `phase":"sync_docs"`、`cycle":1`、`verdict":"pass"`) |
