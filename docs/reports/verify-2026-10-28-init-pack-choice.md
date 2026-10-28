# Verify report: init-pack-choice

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-28-init-pack-choice.md`(AC1〜AC8、issue #275 の受け入れ条件 5 つ)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(origin/main ae64da70、HEAD f9df1718、9 ファイル)。静的検査は `./scripts/run-static-verify.sh` の既定(changed)で、`tests/test-check-template.sh` が言語に分類されないため full に落ちて golang の verifier が走った。Go のテストと `tests/test-check-template.sh` は走らせていない(`/test` の役目)。
- Evidence: `docs/evidence/verify-2026-10-28-init-pack-choice.log`(gitignore 対象。run-verify 自身の記録は `docs/evidence/verify-2026-10-28-155456.log`)

## Spec compliance

計画の digest: `./scripts/plan-visual.sh digest` は `614133eca51f`(rc=0)で、`- Approved:` の行(計画 `:4`)の `sha256:614133eca51f` と一致する。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: skip がある。skip だけで、パックなし・`packs/languages/` と `<lang>.md` なし・`ralph pack add` の案内あり。skip が manifest に残らない | 満たす | pty の t1(`go.mod` と `package.json` を置いた dir、`--pack` なし)で選択肢の最後に `skip (no language pack)` が出た。skip だけを選んで Enter → rc 0、manifest に `packs` の行なし、`packs/` なし、`.claude/rules/ralph/` に言語の rule なし、終わりに `ralph pack add <language> to add a language pack later (available: dart, golang, python, rust, terraform, typescript)`。skip の値 `/skip`(`internal/cli/init.go:168`)は `packsFromSelection`(`:252-261`)が `executeInit` の前に落とす(`:160`)。この 1 行は単体テストの外にあり、t1 の manifest に `/skip` がないことで確かめた |
| AC2: skip+言語、何も選ばない、で確かめの関数がエラーを返し、次へ進まない | 満たす | `validatePackSelection`(`init.go:240-249`)を `newPackQuestion` が `Validate` に渡す(`:284`)。huh v2.0.3 の Next と Submit は、確かめがエラーなら `NextField` を返さない(`field_multiselect.go:464-470`)。pty の t1: 何も選ばずに Enter → 画面が変わらず同じ問いに残った。t2: golang と skip を選ぶと `choose skip alone, or language packs without skip` が出て、Enter で進まなかった。skip を外して Enter で golang だけが入った |
| AC3: 直下の手がかりのファイルで「(Recommended)」が付き、値は名前のまま | 満たす | `packMarkers`(`init.go:176-183`)を python3 -I で取り出し、計画の「手がかりのファイル」の表と 6 言語すべてで集合が一致した。shell 側との関係も計画の説明どおり: golang・typescript・rust・dart は `has_project_marker`(`scripts/detect-changed-languages.sh:316-352`)と一致、python は `has_project_marker` に `setup.cfg` を足したもの、terraform は `classify_language` の `:295` の行と一致。issue の表はどの言語でも `packMarkers` の部分集合。`TestRecommendedPacks_FindsEachMarker` の 19 行で、すべてのパターンに当たるファイル名が 1 つ以上ある。pty: t1 で `golang (Recommended)` と `typescript (Recommended)`、t2(`Cargo.toml`)で `rust (Recommended)`。t2 で golang を選んだ manifest は `packs = ['golang']` で、表示の文字は値に入らない |
| AC4: `--pack` を渡さなければ、最初は何も選ばれていない | 満たす | pty の t1 で 7 つの選択肢がすべて `•`(未選択)。t2(`--pack golang`、`--yes` なし)では golang だけが `✓` で始まった(D2)。初期の選択を問いに渡す行(`runInitInteractive` の `cfg.Packs`)も単体テストの外にあり、t2 で確かめた |
| AC5: `--yes` だけでパックが入らず、案内が出る | 満たす | 組み立てた binary で `ralph init --yes <dir>` → rc 0、manifest に `packs` の行なし、`packs/languages/` なし、言語の rule なし、案内の行あり(下の表) |
| AC6: `--yes --pack` で入れられる。繰り返しとカンマの両方。知らない名前と空の名前は `--yes` の有無によらず、何も書かずにエラー | 満たす | binary で `--pack golang` → `['golang']`、`golang,python` → `['golang', 'python']`、`--pack golang --pack golang` → `['golang']`、`--pack typescript --pack golang` → `['typescript', 'golang']`、`' golang , python '` → `['golang', 'python']`。エラーはすべて rc=1 で、対象の dir がないときは作られず、先に作った空の dir は空のまま: `nosuch`(`--yes` あり・なし)、`''`、`golang,`、`golang` と `''`、`skip`、`/skip`、空白だけ。名前の検査は `RunE` の先頭(`init.go:50-57`)で、両方の経路の前にある |
| AC7: `tests/test-check-template.sh` の G が全パックの `--pack` で通る | 満たす(G の実行は `/test`) | G(`tests/test-check-template.sh:587-648`)は `templates/packs/` の下から `_template` を除いて `dart,golang,python,rust,terraform,typescript` を作り(`:598-604`、空なら fail は `:605-608`)、`--pack "$packs"` で init し(`:616`)、パックごとに `packs/languages/<name>/` を確かめる(`:626-635`)。同じ一覧で組み立てた binary を動かすと rc 0、manifest は 6 つ、その scaffold で `CI=true sh scripts/check-template.sh` は rc 0、`FAIL:` は 0 行。`bash -n` は rc 0 |
| AC8: README の表、雛形の recipe、init の help と `--yes` の説明 | 満たす | `README.md:120`、`templates/base/docs/recipes/adding-a-language-pack.md:30-38`、help の `Long`(`init.go:35-44`)、`--yes`(`:83`)、`--pack`(`:88`)。どれも binary の振る舞いと合う(下の Documentation drift) |

issue #275 の受け入れ条件との対応: (1) skip があり何も入れずに終えられる → AC1。(2) 「(Recommended)」→ AC3。(3) 最初は何も選ばれず、`--yes` だけでは入らない → AC4 と AC5。(4) `--yes --pack golang` → AC6。(5) G が通る → AC7(G の実行は `/test`)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0、44.9 秒) | 2026-10-28T15:54:56Z。shellcheck、hook の `sh -n` 20 本、guard の awk 2 本、`settings.json` の jq 2 本、Codex の hook 系 3 本、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、KNOWN_DIFF 5)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照がすべて OK。golang: `gofmt: ok`、golangci-lint `0 issues.`、go vet と staticcheck は成功時に何も出さない。branch の secret scan は `ae64da70..f9df1718` で clean。時間切れはなく、1 回で通った |
| `git diff --check origin/main...HEAD` / `gofmt -l internal/cli` | pass | どちらも出力なし |
| `bash -n tests/test-check-template.sh` | pass(rc=0) | |
| `shellcheck tests/test-check-template.sh`(単独) | rc=1、前からある INFO 1 件 | SC2016(info)が `:784` に 1 件。origin/main の同じファイルでは `:753` に同じ 1 件があり、この PR の変更ではない。`run-static-verify.sh` は `tests/` を shellcheck しない |

## Documentation drift

`git grep -n -- "--yes"` と `git grep -n "ralph init"` を、`docs/reports`・`docs/evidence`・`docs/plans`・`docs/insights` を除いて見た。`--yes` がすべてのパックを入れる、または最初からすべて選ばれている、と書く現在形の文はなかった。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:120`(コマンドの表) | yes | 対話の問い、`--pack` の初期選択、(Recommended)、skip、`--yes` は `--pack` のパックだけ、繰り返しとカンマ、`ralph pack add`。binary と合う |
| `templates/base/docs/recipes/adding-a-language-pack.md:30-38` | yes(INFO D-1) | `--pack` の初期選択、(Recommended)、skip、`ralph init --yes --pack golang,python` の例。例をそのまま打つと `['golang', 'python']` が入った |
| `ralph init --help`(`init.go:35-44`、`:83`、`:88`) | yes(INFO D-1) | binary の出力を読んだ。`--pack` の説明が指す `ralph pack list` は存在し、6 パックを出す |
| `docs/recipes/adding-a-language-pack.md`(ralph のリポジトリ向けの版) | 対象外 | `scripts/check-sync.sh:107-111` の KNOWN_DIFF。`:24-26` は「選んだパックの rule だけを書く」で、新しい振る舞いとも合う |
| `README.md:192`(scaffold の木) | yes | `packs/languages/` は「opt-in」と書いてある |
| `internal/cli/doctor_hooks_test.go:49` | yes | `ralph init --yes` は hook の不具合の再現手順として出てくるだけで、パックに触れない |
| `docs/tech-debt/README.md:43` | 履歴 | RESOLVED の行で、当時の `ralph init` を書いている |
| 計画の受け入れ条件のチェックボックス | INFO | AC1〜AC8 は `[ ]` のまま。`/verify` は計画を直さない |

- D-1(INFO): help(`init.go:42-43`)と雛形の recipe(`:37-38`)は「知らない名前は何も書く前に失敗する」と書くが、空の名前(`--pack ''`、`--pack golang,`)も同じく失敗することは書いていない。README は名前の検査に触れていない。振る舞いはエラーで止まる側なので、利用者を誤らせる記述ではない。

## Observational checks

組み立てた binary(`go build -o <scratch>/ralph ./cmd/ralph`、HEAD f9df1718)を、使い捨ての dir に向けて打った。

| Command | rc | manifest の `packs` | 書かれたもの |
| --- | --- | --- | --- |
| `init --yes <dir>` | 0 | 行なし | `packs/` なし。案内 `ralph pack add <language> ... (available: dart, golang, python, rust, terraform, typescript)` あり |
| `init --yes --pack golang <dir>` | 0 | `['golang']` | `packs/languages/golang`、`golang.md`。案内なし |
| `init --yes --pack golang,python <dir>` | 0 | `['golang', 'python']` | 2 パックと 2 つの rule |
| `init --yes --pack nosuch <dir>` | 1 | なし | 何も書かない(dir なし → 作らない、空の dir → 空のまま)。`unknown language pack "nosuch" in --pack (available: ...)` |
| `init --pack nosuch <dir>`(`--yes` なし) | 1 | なし | 同上。フォームを出す前に止まる |
| `init --yes --pack '' <dir>` | 1 | なし | 何も書かない。`empty language pack name in --pack "" (available: ...)` |
| `init --yes --pack golang, <dir>` | 1 | なし | 何も書かない。`empty language pack name in --pack "golang," (...)` |
| `TERM=dumb init <dir> </dev/null` | 1 | なし | 何も書かない(dir なし・空の dir の両方)。出力は 148 byte の 1 行 `TERM=dumb and stdin is not a terminal, so ralph init cannot ask its questions; run ralph init --yes (with --pack <name> to install a language pack)` |
| `TERM=dumb init --yes --pack rust <dir>` | 0 | `['rust']` | `--yes` があれば `TERM=dumb` でも動く |

ほかに見たこと:

- 既存の project(上の `--yes` で作った dir)に `TERM=dumb init <dir> </dev/null` → rc 0 で upgrade の案内。既存の project の判定は `TERM=dumb` の止めより前にあり、`TestRunInitInteractive_ExistingProjectSkipsForm`(`internal/cli/cli_test.go:268`)が確かめる順と同じ。同じ dir に `--yes --pack nosuch` → rc 1 で名前のエラー(検査が `RunE` の先頭にあるため、upgrade の案内より先)。
- TERM が dumb 以外で端末のない stdin(`</dev/null`)は、前からの振る舞いのまま `interactive form: huh: bubbletea: error opening TTY` で rc 1、何も書かない。
- pty(python3 -I の `pty.fork`、100x40、`TERM=xterm-256color`)で対話のフォームを 3 回動かした。t1 と t2 は上の AC1〜AC4 の欄。t3(`--pack typescript --pack golang`、そのまま Enter 2 回)の manifest は `['golang', 'typescript']`(名前順)で、`--yes` の同じ flag の `['typescript', 'golang']`(flag の順)と違う。どの回も一覧の 7 行がすべて見え、skip も隠れなかった(`Height` の効果)。
- V-1(INFO): `--pack` なしで言語パックの問いに入ると、何も押す前から `* choose at least one language pack, or skip` が赤く出る(t1)。huh v2.0.3 の `MultiSelect.Focus()` が `updateValue()` を呼び、その中で確かめの関数を通すため(`field_multiselect.go:236-240`、`:540-549`)。次のキーで `m.err = nil` になり消える(`:371-372`)。AC2 はこの表示に触れておらず、案内として読めるので不合格にはしない。気になるなら PR の Notes for reviewers に 1 行書く程度でよい。
- origin/main(ae64da70)の binary を `git archive` から組み立てて比べた: `init --yes` は 6 パックすべて、`TERM=dumb init <dir> </dev/null` も 6 パックすべてを入れて rc 0、`--pack` は `unknown flag`。下の release ノートの「前」はこの観察に基づく。
- `init.go:139-142` のコメント(`TERM=dumb` で huh が accessible モードになり、EOF を確定と読み、空の選択を断るあいだ問いを出し続ける)は huh のソースと合う: `form.go:129-132` が `TERM=dumb` で accessible にし、`MultiSelect.RunAccessible` のループ(`field_multiselect.go:752-765`)は 0 以下の答えで確かめを通し、エラーなら `continue` する。終わらないことそのものは、この report では再現していない(implementer の報告では約 70 秒で 675 MB)。

## Release-note material(`/pr` 向け)

- `ralph init --yes` は言語パックを入れなくなった。前は 6 パックすべてを入れていた。前と同じにするには `ralph init --yes --pack dart,golang,python,rust,terraform,typescript`。名前が分からなければ `ralph pack list`。
- `TERM=dumb` で stdin が端末でないときの `ralph init`(`--yes` なし)は、何も書かずに rc 1 で止まる。前は 6 パックすべてを入れて rc 0 だった。`--yes`(と `--pack`)を付ければ動く。
- manifest の `packs` の並びは、`--yes` では `--pack` に書いた順、対話のフォームでは名前順になる(例: `--pack typescript --pack golang` は `--yes` で `['typescript', 'golang']`、対話で `['golang', 'typescript']`)。
- 対話のフォームは最初に何も選ばず、何も選ばずに Enter を押しても進まない。言語を選ぶか skip を選ぶ。
- 新しい flag `--pack`(繰り返しとカンマ区切り)。知らない名前と空の名前は何も書く前にエラーで止まる。

## Coverage gaps

- `go test ./internal/cli/`(`init_packs_test.go` の 17 関数と `cli_test.go` の既存の init のテスト)と `tests/test-check-template.sh` は走らせていない。`/test` が走らせる。AC7 は G と同じ一覧で binary と `check-template.sh` を動かして代えた。
- `runInitInteractive` の 3 行(初期の選択、推定の結果を問いに渡すこと、`packsFromSelection` の呼び出し)は、計画のとおり自動テストでは固定されていない。この report では pty の t1・t2 で 3 行とも確かめたが、回帰を捕まえる仕組みはない。
- pty は 1 つの端末の大きさ(100x40)と `xterm-256color` だけ。狭い端末、ほかの TERM、Windows は見ていない。
- `TERM=dumb` で stdin が端末のとき(accessible モードで人が数字を打つ)は見ていない。
- symlink の手がかりのファイル、読めない dir での推定は binary では打っていない(単体テストの範囲。実行は `/test`)。

## Verdict

- Verdict: pass
- Verified: AC1〜AC6 と AC8(binary の観察、pty の 3 回、手がかりの表の機械的な照合、文書の grep)、計画の digest、静的検査の全体(`run-static-verify.sh` rc=0)、issue #275 の受け入れ条件 1〜4
- Partially verified: AC7 と issue の受け入れ条件 5(G の組み立てを読み、同じ一覧で binary と `check-template.sh` を通した。G そのものの実行は `/test`)
- Not verified: Go のテストと `tests/test-check-template.sh` の実行結果、`TERM=dumb` で問いが終わらないことの再現、狭い端末での一覧の表示
