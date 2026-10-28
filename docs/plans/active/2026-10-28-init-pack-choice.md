# init-pack-choice

- Status: Approved
- Approved: 2026-10-28 sha256:614133eca51f
- Owner: Claude Code
- Date: 2026-10-28
- Related request: ralph init: 言語パックに skip を足し、推定した言語に Recommended を添える
- Related issue: #275
- Type: feat
- Branch: feat/init-pack-choice

## Objective

`ralph init` の言語パックの選び方を変える。対話のフォームでは、最初は何も選ばず、skip(言語パックを入れない)を選べるようにし、対象のディレクトリのルートにあるファイルから推定した言語に「(Recommended)」を添える。`--yes` では言語パックを入れず、入れたいときは `--pack <名前>` で指定する。

## Scope

- S1 `--pack` の flag と `--yes` の既定(`internal/cli/init.go`)
  - `--pack <名前>` を足す。繰り返しても、カンマで区切ってもよい。flag は `StringArrayVar` で受け、カンマで分けるのは自分で行う。`StringSliceVar` は CSV として読むので、空の値(`--pack ''`)を黙って捨て、検査で拾えないため(pflag の `stringSliceValue.Set` は `readAsCSV("")` の空の結果を足すだけ)。重なった名前は 1 つにまとめる
  - 名前の検査は `RunE` で、`--yes` の有無によらず、フォームを出す前と何かを書く前に行う。知らない名前と空の名前(`--pack ''`、`--pack golang,` の末尾の空)はエラーで止める(終了コードは 0 以外)。エラーの文には使える名前の一覧を入れる。`executeInit` は知らないパックを警告して続け、manifest にはその名前を残す(`init.go:187-201`)ので、その手前で止める
  - `--yes` だけのときは言語パックを入れない。`--yes --pack golang` なら golang だけを入れる
  - `--yes` の flag の説明(`init.go:50` の「use defaults」)を、言語パックを入れないことが分かる文に変える
  - 言語パックを 1 つも入れなかったときは、終わりの表示に「あとから `ralph pack add <language>` で入れられる」と、使える名前の一覧を出す
- S2 `tests/test-check-template.sh` の G
  - `go run ./cmd/ralph init --yes` に、`templates/packs/` の下のすべてのパック(`_template` を除く)を `--pack` で渡し、今と同じ内容の scaffold を作る
- S3 対話のフォーム(`internal/cli/init.go`)
  - 言語パックの選択肢は、最初は何も選ばない。`cfg.Packs` の初期値を、今のすべてのパック(`init.go:80`)から `--pack` の名前(なければ空)に変える
  - 選択肢の最後に skip(言語パックを入れない)を足す
  - 対象のディレクトリのルートに手がかりのファイルがある言語は、選択肢の表示に「(Recommended)」を添える。値(パックの名前)は変えない
  - skip とほかの言語を一緒に選んだときと、何も選ばなかったときは、選択を確かめる関数(huh の `Validate`)がエラーを返し、フォームが同じ問いにとどまる(もう一度聞く)
  - フォームの選択を `executeInit` に渡す前に、入れるパックの一覧に直す。skip だけの選択は空の一覧にする。skip の値がパックの名前として `executeInit` に届くと、パックの書き出しは警告だけで続き、manifest に skip が残る(`init.go:187-201`)ため
  - 推定、選択肢の組み立て、選択の確かめ、選択から入れるパックへの直し、言語パックの問い(`*huh.MultiSelect`)の組み立ては、フォームの外の関数にしてテストする
- S4 文書
  - `README.md` のコマンドの表の `ralph init` の行
  - `templates/base/docs/recipes/adding-a-language-pack.md` の「init のときに選べる」の段落に `--pack` を足す
  - `init` の help(`Long`)に、`--yes` は言語パックを入れないことと `--pack` を書く

### 手がかりのファイル

issue の表、`scripts/detect-changed-languages.sh` の `has_project_marker`(316-352 行)、terraform は手がかりを持たない `has_project_marker` に代えて `classify_language`(295 行)の和を取る。探すのは対象のディレクトリの直下だけ。

| パック | 手がかりのファイル | 出どころ |
|---|---|---|
| golang | `go.mod` | 表、has_project_marker |
| typescript | `package.json`、`package-lock.json`、`pnpm-lock.yaml`、`yarn.lock`、`tsconfig.json`、`tsconfig.*.json` | 表、has_project_marker |
| python | `pyproject.toml`、`requirements.txt`、`requirements-*.txt`、`setup.py`、`setup.cfg`、`tox.ini` | 表、has_project_marker(`setup.cfg` は表と classify_language) |
| rust | `Cargo.toml` | 表、has_project_marker |
| dart | `pubspec.yaml` | 表、has_project_marker |
| terraform | `*.tf`、`*.tofu`、`*.tftest.hcl`、`.terraform.lock.hcl` | 表、classify_language |

## Non-goals

- ralph.toml の質問をフォームに足すこと(#277)
- `ralph pack add` の振る舞い
- 手がかりのファイルをルートより下の階層まで探すこと。issue はルートにあるファイルと決めている
- 既存の project(`.ralph/manifest.toml` がある)での init の振る舞い。今までどおり、フォームを出す前に upgrade の案内で終わる
- 言語の推定を `scripts/detect-changed-languages.sh` と同じ実装にまとめること(shell と Go で二重になる。手がかりのファイルの一覧だけをそろえる)
- ソースのファイル(`*.go`、`*.py` など)やツールの設定(`vite.config.*` など)からの推定。プロジェクトの印になるファイルだけを見る

## Assumptions

- 言語パックが 0 個の scaffold は、今も扱える。`manifest.Meta.Packs` は `omitempty` で、`ralph doctor` の `checkInstalledPacks` は空のとき「none installed」で pass にする(`internal/cli/doctor.go:697-699`)。`tests/test-check-template.sh` の H1 は、`packs/` のない project が check-template を通ることを確かめている
- `ralph init --yes` を呼ぶのは、リポジトリの中では `tests/test-check-template.sh` の G だけ(`git grep` で確かめた。`.github/` と `scripts/` に呼び出しはない。report・evidence・archive の計画を除く)
- huh v2.0.3 の `MultiSelect` には `Validate(func([]T) error)` がある。Submit と Next のキーでは確かめの関数がエラーを返すと次へ進まない(`field_multiselect.go:464-470`)。`Blur()` も選択を読み直して確かめの関数を通し、結果を `Error()` で返す(`:243-247`、`:540-548`)。この 2 つを使えば、TTY なしで問いの組み立てと確かめの配線をテストできる

## Affected areas

- `internal/cli/init.go`(flag、`RunE` の名前の検査、`runInitNonInteractive`、`runInitInteractive`、終わりの表示)
- `internal/cli/init_packs_test.go`(新規。推定、選択肢、選択の確かめ、問いの配線、`--pack` と `--yes` の経路)
- `tests/test-check-template.sh`(G)
- `README.md`、`templates/base/docs/recipes/adding-a-language-pack.md`

## Visual review

- Page: `.harness/state/plan-visual/init-pack-choice.html`(易しい図 2 つ: `overview`、`easy-yes`。fold の中に `detail-overview`、`detail-flow`、`detail-question`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: ページ全体と 5 つの図を `shot` で撮って読んだ。易しい図 2 の副文が枠に近かったので短くし、撮り直して崩れがないことを確かめた

## Design decisions

- D1(F1、何も選ばずに送ったとき): 確かめの関数がエラーを返し、同じ問いにとどめる。skip とほかを一緒に選んだときと同じ仕組み(`Validate`)で、誤って Enter を押してパックなしで進むことを防ぐ。skip を選ぶ意味も残る。consult(consult-plan-initpack)の判定も同じ
- D2(F2、`--pack` を `--yes` なしで渡したとき): 対話のフォームで、その名前を選んだ状態から始める。flag は既定値を与えるだけにする。フォームの形が条件で変わらないので、#277 で質問が増えても崩れない。「最初は何も選ばない」は、flag を渡さないときの話として扱う
- D3(F3、手がかりのファイル): issue の表と `has_project_marker` の和を取り、terraform は `classify_language` の行を使う(上の表)。issue が「判定に合わせる」と書いているため。表はテストにも置く
- D4(F4、skip の位置): 選択肢の最後に置く。パックの並び(`AvailablePacks` の名前順)を変えない
- D5(Codex の計画助言 MEDIUM 2 件を反映): `--pack` は `StringArrayVar` で受けてカンマを自分で分け、空の要素をエラーにする。フォームの選択は、skip を空の一覧に直してから `executeInit` に渡す
- Critical forks: D1〜D4。どれも consult に聞いて、推奨案を選んだ(ユーザーの指示)

## Acceptance criteria

- [ ] AC1: 対話のフォームの言語パックの選択肢に skip がある。skip だけを選ぶと、言語パックを入れずに init が終わる(`manifest.Meta.Packs` が空、`packs/languages/` と `.claude/rules/ralph/<lang>.md` を作らない、終わりに `ralph pack add` の案内が出る)。skip がパックの名前として manifest に残らない
- [ ] AC2: skip とほかの言語を一緒に選んだとき、何も選ばなかったとき、確かめの関数がエラーを返し、言語パックの問いは次へ進まない
- [ ] AC3: 対象のディレクトリの直下に、上の表の手がかりのファイルがある言語の選択肢に「(Recommended)」が付く。値はパックの名前のまま
- [ ] AC4: 対話のフォームでは、`--pack` を渡さなければ、最初は何も選ばれていない
- [ ] AC5: `--yes` だけでは言語パックが入らず、終わりの表示に `ralph pack add` の案内が出る
- [ ] AC6: `--yes --pack golang` のように、flag で言語パックを入れられる。繰り返しとカンマ区切りの両方を受け付ける。知らない名前と空の名前は、`--yes` の有無によらず、何も書かずにエラーで止まる
- [ ] AC7: `tests/test-check-template.sh` の G が、`--pack` ですべてのパックを入れた scaffold で通る
- [ ] AC8: README のコマンドの表、雛形の recipe、init の help と `--yes` の説明が新しい振る舞いを書く

## Implementation outline

1. S1: `--pack` の flag と名前の検査、`--yes` の既定と説明、パックを入れないときの案内(AC5・AC6) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Init|Pack'`
2. S2: `tests/test-check-template.sh` の G をすべてのパックの `--pack` に(AC7) — verify: `sh tests/test-check-template.sh`
3. S3: 対話のフォームの推定、選択肢、選択の確かめ、問いの組み立て(AC1〜AC4) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Init|Pack'`
4. S4: README、雛形の recipe、init の help(AC8) — verify: `grep -n "ralph init" README.md && grep -n -- "--pack" templates/base/docs/recipes/adding-a-language-pack.md && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、check-sync、check-skill-sync を含む)
- Spec compliance criteria to confirm: AC1〜AC8。issue #275 の受け入れ条件 5 つとの対応
- Documentation drift to check: `git grep -n -- "--yes"` と `git grep -n "ralph init"` で、`--yes` がすべてのパックを入れると書く文書が残っていないこと。`docs/recipes/adding-a-language-pack.md`(ralph のリポジトリ向けの版)は init の選び方を書いていないので対象外
- Evidence to capture: `run-static-verify.sh` の出力。組み立てた binary で `ralph init --yes`、`--yes --pack golang`、`--yes --pack nosuch`、`--pack nosuch`(`--yes` なし)を打った出力と終了コード

## Test plan

- Unit tests:
  - 推定: 上の表の手がかりのファイルごとに、その言語だけが推定される(表をテストに置く)。ルートにファイルがないとき、手がかりが下の階層にだけあるとき、対象のディレクトリがまだないときは何も推定しない
  - 選択肢: `--pack` がなければどれも選ばれていない。推定した言語の表示に「(Recommended)」が付き、値はパックの名前のまま。最後に skip がある
  - 選択の確かめ: skip だけ → nil、言語だけ → nil、skip と言語 → エラー、何も選ばない → エラー
  - 問いの配線: 言語パックの問いを組み立て、選んだ状態を作って `Blur()` を呼び、`Error()` が上の確かめと同じ結果になる
  - 選択から入れるパックへの直し: skip だけ → 空、言語だけ → その名前のまま
  - `--pack` の名前: 重なりを 1 つに、カンマで分ける、知らない名前と空の名前はエラーで、使える名前を挙げる
- Integration tests: `newInitCmd` に `--yes`、`--yes --pack golang`、`--yes --pack golang,python`、`--yes --pack golang --pack golang`、`--yes --pack nosuch`、`--pack nosuch`(`--yes` なし)、`--yes --pack ''`、`--yes --pack golang --pack ''`、`--yes --pack '' --pack golang`、`--yes --pack golang,` を渡す。manifest の `Packs`、`packs/languages/` と `.claude/rules/ralph/<lang>.md` の有無、終わりの表示を確かめる。知らない名前と空の名前ではエラー(終了コードは 0 以外)になり、対象のディレクトリに何も書かれないこと(`--yes` なしでもフォームを出す前に止まること)を確かめる。skip だけの選択を直した結果で `executeInit` を通し、manifest の `Packs` が空、パックのファイルなし、`ralph pack add` の案内ありを確かめる
- Regression tests: `tests/test-check-template.sh` の G(すべてのパック)と H1(パックなし)。既存の `TestExecuteInit_V2_*` と `TestRunInitInteractive*`(`cli_test.go:263` の、既存の project で upgrade の案内に入る確かめ)が通り続ける
- Edge cases: `--pack` だけ(`--yes` なし)は、その名前を選んだ状態の選択肢になる(D2)。`--pack skip` は知らない名前としてエラー。空の名前は単独でも、ほかの名前の前後に混ざっても、カンマの末尾でもエラー
- Evidence to capture: `go test` の出力、`run-test.sh` の結果

## Risks and mitigations

- `ralph init --yes` を script で使い、すべてのパックが入る前提の利用者がいる: release のノートに「`--yes` は言語パックを入れない。今までと同じにするには `--pack` で名前を渡す」と書く。PR の Notes for reviewers にも入れる
- huh のフォーム全体はテストで動かせない: 推定、選択肢、選択の確かめ、問いの組み立てをフォームの外の関数にし、問いは `Blur()` と `Error()` で配線まで確かめる。`form.Run()` を呼ぶ部分は短く保つ
- 手がかりのファイルの一覧が shell の検出とずれる: 上の表をテストに置き、`has_project_marker` と `classify_language` の行をコメントで指す
- `--pack` の検査を `executeInit` の中に置くと、知らない名前が警告だけで通ってしまう: `RunE` の先頭で検査し、テストで何も書かれないことを確かめる

## Rollout or rollback notes

- 戻すときはこの PR を revert する。manifest の形は変えないので、この版で作った scaffold は前の版でも読める
- release のノートに `--yes` の振る舞いの変化と `--pack` を書く

## Open questions

- なし(分かれ目は Design decisions に書いた)

## Progress checklist

- [x] Plan reviewed(consult-plan-initpack: 判定「直してから進める」。F1〜F4 は案のとおり。手がかりのファイルの出どころを言語ごとの表にし、`--pack` の検査を `RunE` で両方の経路の前に置き、`--yes` の説明と `cfg.Packs` の初期値を直した。Codex の計画助言: MEDIUM 2 件(skip を空の一覧に直す処理、空の `--pack` を `StringSliceVar` が捨てる)を聞かずに反映した(D5))
- [x] Plan approved(自動承認(ユーザーの指示)。2026-10-28、digest sha256:614133eca51f)
- [x] Branch created
- [x] Implementation started
  - 別のセッション(stoic-app)の local CI の間(2026-10-28 21:43 から約 90 分)は Go のテストを打たず、計画と図解だけを進めた。S1 は予備のタイマーが切れたあとに始めた
  - S1(baa66655、implementer): `--pack`(`StringArrayVar`、カンマを自分で分ける、重なりを 1 つに、空と知らない名前はエラー)、`RunE` の先頭での検査、`--yes` だけではパックなし、0 個のときの `ralph pack add` の案内、`--yes` の説明。verify rc=0(8s)。`go test -overlay` の mutant 5 種(検査が `--yes` の経路だけ、`StringSliceVar`、`--yes` で全部入る、案内なし、重なりを残す)がすべて落ちる。組み立てた binary で `--yes`、`--yes --pack golang`、知らない名前・空の名前(`--yes` の有無とも)を打って確かめた。self-check 5 項目 clean
    - 計画になかった追加: 名前の前後の空白を取る(`TrimSpace`)。`--pack "golang, typescript"` を受け付け、空白だけの名前は空としてエラーにする
    - S3 までの中間の状態: 対話の経路では `--pack` を検査するが、フォームにはまだ渡さない(S3 で選んだ状態にする)
  - S2(b1b244de、implementer): G は `templates/packs/` の下のパック(`_template` を除く)をカンマでつないだ 1 つの `--pack` で渡し、init のあと各 `packs/languages/<name>/` があることを確かめてから check-template を通す。一覧が空なら G を fail にする。verify は `./tests/test-check-template.sh` で rc=0(60 passed、G は PASS)。mutant 2 つ(`--pack` を外す、一覧の場所を変える)で G が fail する。self-check 5 項目 clean
    - 計画の検証コマンドの直し: 計画の S2 の `sh tests/test-check-template.sh` は macOS の `/bin/sh`(POSIX モードの bash 3.2)では、前からある `diff <(...)` で構文エラーになり rc=2(変更前の HEAD でも同じ)。shebang の bash で動く `./tests/test-check-template.sh` で確かめた。以後この形で打つ
  - S3(d1c756de、implementer): `runInitInteractive` に `--pack` の一覧を渡し、初期の選択にした(なければ空)。手がかりの表 `packMarkers` と推定 `recommendedPacks`(直下の通常ファイルだけ、symlink はたどる)、選択肢 `packOptions`(名前順、(Recommended) は表示だけ、最後に skip)、確かめ `validatePackSelection`、直し `packsFromSelection`(skip の値は必ず取り除く)、問い `newPackQuestion`。verify rc=0(11s)、`golangci-lint run ./internal/cli/` 0 issues。`go test -overlay` の mutant 12 種が落ちる。組み立てた binary を pty で動かし、(Recommended)、空の確定と skip+言語の確定で同じ問いに残ること、skip だけでパックなしと案内、`--pack golang` が選ばれた状態を確かめた。self-check: 不要な変更 fixed(テストの冗長な比較)、誤字とコピペ fixed(出力の検査の文字列を絞った)、ほか clean
    - 計画になかった追加 1: skip の値は `/skip`(パックの名前はディレクトリ名なので `/` を含まない)。テストは値に `/` が入ることを確かめる(self-review L4 の直しで、AvailablePacks に含まれないことの確かめは `/` から導けるので外した)
    - 計画になかった追加 2: 問いに `Height` を付けた。huh v2.0.3 は `Height` がないと一覧の高さを「選択肢の行数 − タイトルと説明の行数」にするので、末尾の選択肢が隠れる(pty で、変更前の HEAD は typescript が、`Height` なしの新しい版は typescript と skip が見えなかった)
    - 計画になかった追加 3: `TERM=dumb` で stdin が端末でないときは、フォームの前に何も書かずにエラーで止める。huh は `TERM=dumb` で accessible モードになり、EOF を確定として読むので、D1 の確かめがエラーを返す間ずっと問いを出し続ける(`TERM=dumb ralph init dir < /dev/null` が約 70 秒で 675 MB を出して終わらなかった。変更前は全パックを入れて exit 0)。エラーの文は `--yes`(と `--pack`)を勧める。consult(consult-plan-initpack)は異論なし。release のノートに書く
    - 自動テストで捕まえられない穴: `runInitInteractive` の中の 3 行(初期の選択、推定の結果を問いに渡すこと、`packsFromSelection` の呼び出し)は `form.Run()` の先で TTY が要るので、mutant が生き残る。pty での確認で代えた
    - 対話の経路では、huh が選択を選択肢の並び(名前順)で書き戻すので、`--yes` なしの `--pack typescript --pack golang` の manifest の順は名前順になる(`--yes` の経路は flag の順)。受け入れ条件には関わらない
  - S4(411a3499、implementer): init の `Long` の help(問い、(Recommended)、skip、`ralph pack add`、`--yes` と `--pack`、知らない名前、`TERM=dumb` で端末がないときは `--yes`)、README のコマンドの表の行、雛形の recipe の段落(`ralph init --yes --pack golang,python` の例)。verify: grep 2 つ rc=0、`./scripts/check-sync.sh` rc=0(DRIFTED 0)、gofmt・vet・`go test ./internal/cli/ -run 'Init|Pack'` rc=0。`go run ./cmd/ralph init --help` の出力を確かめた。self-check 5 項目 clean
  - main の取り込み(20c13af6): #281 を merge commit で取り込んだ。衝突なし、`go vet ./internal/cli/` ok
  - self-review Run 1(7eca2cb1): LOW 5 件(L1 `--pack` を `--yes` なしで渡したときの説明がない、L2 一覧の高さの式、L3 `AvailablePacks` を 1 回の init で 3 回読む、L4 テストの冗長な半分、L5 コメントの symlink の記述)。5 件とも f495bb23(implementer)で直した。L3 は `initConfig` に `AvailablePacks` を持たせ、`RunE` で 1 回だけ読んで両方の経路に渡す形にし、`runInitInteractive` の 2 回目の読み出しと案内の側のエラーの握りつぶしを消した。案内の一覧が空のときのテストを足した。`runInitInteractive` が `cfg.AvailablePacks` を入れ忘れてもフォームの先なので単体テストでは捕まらず、pty で skip だけを選んで案内に一覧が出ることを確かめた。verify rc=0(12s)、check-sync rc=0。self-check 5 項目 clean
  - self-review Run 2(a89381e4): 所見なし、Merge yes。ただし L3 の直しで、対話の経路の `AvailablePacks: availPacks` の 1 行がフォームの先になり、単体テストの外に出たと指摘した(前は `executeInit` が一覧を読み直していた)。consult(consult-plan-initpack)の案のとおり、`RunE` で `initConfig`(ProjectName、Packs、AvailablePacks)を 1 回だけ組んで両方の経路に渡す形にした(dbc5afaf、implementer)。`--yes` の経路の `RunE` のテストが共有の代入を押さえ、`go test -overlay` でその代入を消すと `TestInitCmd_YesAloneInstallsNoPackAndPrintsTheHint` が落ちる。help の「the packs --pack names」の 2 回目を「those packs」にした。verify rc=0(14s)、`golangci-lint run ./internal/cli/` 0 issues。self-check 5 項目 clean
  - self-review Run 3(e697e86c): LOW 2 件(L6 help の「those packs」の指す先が 3 文前の別の段落、L7 `AvailablePacks` 欄のコメントが問いの選択肢に使うことを書いていない)。init.go の文言だけの直しなので inline で直した(「Trivial single-file edits」の例外)。help は README と同じ「--pack が名前を挙げない限り言語パックは入らない」の形にした。verify(gofmt、vet、`go test ./internal/cli/ -run 'Init|Pack'`)rc=0、help の行は最長 79 桁。self-check: 不要な変更 clean、誤字とコピペ clean、デバッグの残り clean、秘密情報 clean、範囲外のファイル clean(stage は init.go だけ)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
