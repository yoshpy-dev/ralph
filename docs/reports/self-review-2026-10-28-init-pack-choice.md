# Self-review report: init-pack-choice

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-init-pack-choice.md`(D1〜D5、AC1〜AC8、Progress の S1〜S4 と S3 の追加 3 点)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge base ae64da70、HEAD 20c13af6、7 ファイル、+1035/-27)。コードは `internal/cli/init.go`(+215)、`internal/cli/init_packs_test.go`(新規 627 行)、`internal/cli/cli_test.go`(呼び出し 1 か所)、`tests/test-check-template.sh`(case G)、`README.md`(表の 1 行)、`templates/base/docs/recipes/adding-a-language-pack.md`(1 段落)。計画ファイルは読んだが所見の対象にしていない。命名、読みやすさ、null / 空の扱い、例外処理、security、保守性、テストの質(主張を固定しているか、TTY なしで止まらないか)の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。テスト、静的検査、spec 適合は実行も評価もしていない。
- ID: 所見は L1〜L5。

## Evidence reviewed

- `internal/cli/init.go` の diff の全体と、`runInitInteractive`(`:116-162`)、`executeInit` の終わり(`:420-440`)の周辺。`RunE` の先頭(`:47-58`)で `AvailablePacks` と `parsePackFlag` を呼ぶのが、`filepath.Abs` と `runInitInteractive` の既存 project 判定より前であることを確かめた。`absDir, err :=` は `err` が宣言済みでも `absDir` が新しいので通る。
- `parsePackFlag`(`:90-106`)の分岐。空の名前(空の値、末尾・先頭のカンマ、空白だけ)、許可リストにない名前(`skip` と `_template` を含む。`AvailablePacks` は `_template` を除く、`internal/scaffold/embed.go:41-44`)、重なり(最初の順を残す)のどれも、パスに使われる前にエラーか除外になる。pack の名前がファイルパスに届く前に許可リストを通るので、パス区切りやコマンドの注入の経路はない。
- `recommendedPacks` / `isPackMarker`(`:189-216`)。`ReadDir` の失敗は「推定なし」に倒す(ラベルだけの用途で、書き込み時の失敗は `executeInit` が出すという理由がコメントにある)。`Match` のエラーを捨てる箇所は、`packMarkers` の pattern が固定であることがコメントに書かれている。`Stat` は symlink をたどり、dangling は false。
- `packMarkers`(`:169-180`)と `scripts/detect-changed-languages.sh:316-352` の `has_project_marker`、`:276-298` の `classify_language` の突き合わせ。typescript 6 種、python 6 種(`setup.cfg` は classify_language 由来)、rust / golang / dart 各 1、terraform 4 種で、計画の表と一致する。
- huh v2.0.3 のソース(`~/go/pkg/mod/charm.land/huh/v2@v2.0.3`)。`form.go:129-132` が `TERM=dumb` で accessible にする条件と同じ条件を、`init.go` のガードが書いている。`field_multiselect.go:739-777` の `RunAccessible` は `validate` がエラーのとき `continue` で同じ問いに戻る。`Height`(`:206`)は設定時に `updateViewportSize`(`:495-515`)を呼び、そのときの title と description の高さを引くので、`newPackQuestion` が `Title`・`Description`・`Options` のあとに `Height` を置いている順序は正しい。EOF を「確定」と読む挙動は計画 S3 の測定(約 70 秒で 675 MB)に拠り、`PromptString` は読み直していない。
- `go.mod:7` の `github.com/charmbracelet/x/term` は最初から direct で、`internal/cli/upgrade.go:14` も使っている。`go.mod` / `go.sum` に差分はない。`StringArrayVar` は `org.go:825` の `--reserve` と同じ idiom。
- `internal/cli/init_packs_test.go` の全体。`TestInitCmd_BadPackNameStopsBeforeAnyWrite`(`:183-245`)は、フォームが端末を待つ前に、既存 v2 project への実行で `nil` が返ることを先に確かめ、検査が欠けた退行がハングでなく失敗として出るように並べてある。`TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm`(`:559-588`)は stdin の書き込み側を開いたまま保つので、検査が欠けても huh は読み取りで止まり、出力を溢れさせない。待ちは 10 秒で上限がある。`heldStdinWriters` は退行時だけ増える。
- `tests/test-check-template.sh` の case G。`templates/packs/*/` を `_template` 抜きで集め、空なら fail、`init` のあとで各 `packs/languages/<name>/` を確かめてから check-template を通す。失敗経路ごとに `rm -rf "$fresh"; trap - EXIT; return` がそろっている。
- 確かめていない: テストの実行、`pty` での挙動(計画の S3 に実装側の記録がある)、静的検査。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | help text | `--pack` を `--yes` なしで渡すと、フォームがその名前を選んだ状態で始まる(D2)。help と表はこれを書かず、むしろ反対を言う。`Long` の「The language-pack question starts with nothing selected.」は `--pack` の有無を言わない。`--pack` の説明「language pack to install」は、フォームでは取り消せる初期値の指定になるので不正確。README の行と recipe の段落は `--yes` と組みでだけ `--pack` を書く。 | `init.go:35`(`Long`)、`:80`(`--pack` の説明)、`README.md:120`、`templates/base/docs/recipes/adding-a-language-pack.md` の追加段落 | `Long` の 1 文目を「starts with nothing selected, or with the --pack names selected」の形にし、`--pack` の説明を「language pack to install (preselected in the interactive form)」のようにする。README と recipe にも 1 句足す。文書のずれの判定は `/sync-docs` だが、文言は `init.go` の中にあるのでここでも挙げた |
| LOW | readability | `Height(1 + strings.Count(packQuestionDescription, "\n") + 1 + len(options))` の足し算は、どの項が何の行かを式から読めない(1 = title、`Count + 1` = description の行数、`len(options)` = 選択肢)。直前のコメントは「なぜ要るか」は書くが、各項は書かない。description を 1 行増やしたり title が 2 行になったりしたときに、直す場所が式の中に埋もれる。`TestNewPackQuestion_ShowsEveryOptionWithItsLabel` が結果を固定するので壊れたら気づくが、読む人には式の意味が先に要る。 | `init.go:271-276`、`field_multiselect.go:495-515`(`Height` は title と description の高さを引いたものが一覧の高さ) | 名前をつけた局所変数にする。例: `titleLines := 1`、`descriptionLines := strings.Count(packQuestionDescription, "\n") + 1`、`Height(titleLines + descriptionLines + len(options))` |
| LOW | maintainability | `scaffold.AvailablePacks()` が 1 回の init で最大 3 回呼ばれる(`RunE` の `:49`、`runInitInteractive` の `:128`、`executeInit` の hint の `:434`)。CLI から来る限り、`RunE` が成功した時点で 2 つ目の呼び出しは失敗しないので、`runInitInteractive` の `listing packs: %w` の分岐は直接呼ぶテストでしか動かず、同じ文言が 2 か所にある。hint 側は `err == nil && len(avail) > 0` で一覧を黙って落とす(コメントで理由は書いてある)が、一覧を持っている呼び出し元から渡せば、この握りつぶしも要らない。 | `init.go:49-52` と `:128-131`(同じ `listing packs` の文言)、`:433-437`(hint) | `runInitInteractive` に `RunE` が読んだ `availPacks` を渡す(`cli_test.go:285` の呼び出しは 1 引数増える)。hint の一覧は `initConfig` に持たせるか、`executeInit` の引数で受ける。範囲が広がるなら、今のままで受け入れ可。動作は変わらない |
| LOW | test quality | `TestSkipPackValue_IsNoPackName` の後半(テストの MapFS とリポジトリの templates の両方で、`AvailablePacks()` が `skipPackValue` を含まないことの確認)は、前半(`skipPackValue` が `/` を含む)から導かれる。`AvailablePacks` は `fs.ReadDir` の名前を返し、その名前に `/` は入らないので、前半が通るなら後半は落ちようがない。前半が落ちるときは後半も落ちるので、後半は独立した情報を足さない。そのわりに、`scaffold.EmbeddedFS` を 2 回差し替える準備(`setupTestEmbedFS` と `useRepositoryTemplates`)を抱える。 | `init_packs_test.go:259-279` | 後半を消して、`/` を含むことの確認と「理由」(pack 名はディレクトリ名なので `/` を持てない)のコメントだけにする。残すなら、計画 S3 の「AvailablePacks に含まれないことを確かめる」に応える意図がコメントで分かるようにする |
| LOW | maintainability | `isPackMarker` のコメントの「as test -f sees it」は、shell の突き合わせ先と 2 pattern でずれる。`tsconfig.*.json` と `requirements-*.txt` は shell では `has_name_in_dir`(`find -maxdepth 1 -type f -name`)で、`find -type f` は symlink をたどらない。Go は全 pattern で `os.Stat` を使い、symlink の `tsconfig.build.json` も印にする。影響は選択肢のラベルだけで小さいが、`packMarkers` のコメントが「`has_project_marker` に従う」と書き、計画のリスク欄が shell との一覧のずれを挙げているので、ずれの種類を正しく書いておきたい。テストの symlink の行は `go.mod`(完全一致の名前)だけで、glob の pattern では固定されていない。 | `init.go:204-216`、`scripts/detect-changed-languages.sh:327`、`:334`(`has_name_in_dir`)と `:322-326` ほかの `[ -f ... ]` | コメントを「symlink はたどる(shell の `[ -f ]` と同じ。`tsconfig.*.json` と `requirements-*.txt` だけは shell が `find -type f` でたどらない)」にするのが最小。挙動を shell に合わせるなら glob の pattern だけ `Lstat` に変える必要があるが、ラベルだけの用途なので不要と考える |

## Positive notes

- `--pack` の名前は、フォームを出す前、何かを書く前に `RunE` で許可リストと突き合わせる。`executeInit` が知らない pack を警告して manifest に残す性質(警告は `init.go:360`)を、入口で止めている。`StringSliceVar` が空の値を捨てる理由をコメントで示して `StringArrayVar` を選んでおり、`org.go` の既存の idiom とも合う。
- skip の値を `/skip` にして、pack の名前が持てない文字を使う、という不変条件を定数のコメントとテストの両方に置いた。`packsFromSelection` が必ず取り除くので、skip が manifest に残る経路は閉じている。
- `validatePackSelection` のエラー文が、次にすることを言っている(「choose at least one language pack, or skip」)。同じ関数を `Validate` と単体テストの両方で使い、`Blur()` / `Error()` で配線まで確かめている。
- `TERM=dumb` のガードは、huh の判定と同じ条件を使い、その理由(accessible モードで EOF を確定と読み、D1 の検査が無限に問う)をコメントに書いている。エラー文は `--yes` と `--pack` を勧める。テストは stdin を開いたまま保ち、検査が欠けた退行でも出力があふれず、10 秒の上限で失敗として終わる。
- `recommendedPacks` は、ディレクトリがまだない・読めない場合を「推定なし」に倒し、その理由と、書き込み時の失敗は `executeInit` が出すことをコメントにしている。マーカーの表は、リポジトリの全 pack に行があることを `TestPackMarkers_NameEveryRepositoryPack` が固定し、pack を足したときの直し先をエラー文が言う。
- `tests/test-check-template.sh` の G は、pack の一覧を `templates/packs/` から作るので、pack を足しても書き換えが要らない。`--yes` だけでは何も入らなくなったことで check-template が pack の中身を見なくなる穴を、`packs/languages/<name>/` の存在確認で塞いでいる。

## Coverage gaps

- `runInitInteractive` の 3 行(`Packs: packs` の初期の選択、`recommendedPacks(...)` を問いに渡すこと、`cfg.Packs = packsFromSelection(cfg.Packs)`)は `form.Run()` の先で TTY が要り、単体テストでは変異が生き残る。計画の S3 が既知の穴として挙げ、pty での確認で代えている。特に `packsFromSelection` の呼び出しが外れても単体テストは通り、`/skip` が manifest に残る(AC1 の後半)。`/test` の判断。
- `runInitInteractive` に `--pack` の一覧を渡して問いに選ばせるところと、その結果を `executeInit` に渡すところは、フォームの外の関数として切り出せる余地がある(例: フォームを組む関数と、フォーム後の `cfg` を直す関数)。やるかどうかは計画の範囲外の判断。

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5)
- Follow-ups: 修正を必須とする所見はない。(1) L1 は文言だけの直しで小さく、`--pack` の効き方が help から読めるようになるので、直すのを勧める。(2) L2・L4・L5 はコメントか名前の直しで、動作は変わらない。(3) L3 は引数の形を変えるので、範囲が広がるなら受け入れてよい。直した場合も受け入れた場合も、コードの変更を含むので `/verify` へ進む(record-only の例外には当たらない)。cycle 1 のため、残した所見の台帳への登録は今回は不要。
