# Walkthrough: init-pack-choice(ralph init の言語パックに skip と Recommended、`--pack`)

- Date: 2026-10-29
- Plan: docs/plans/active/2026-10-28-init-pack-choice.md(/pr で archive に移す)
- Issue: #275
- Branch: feat/init-pack-choice(この walkthrough の前の HEAD 7a743fb9。merge を除く 25 commits と、origin/main を取り込んだ merge commit 20c13af6。`git diff origin/main...HEAD` で 13 files、+1607/-40)
- Base: ae64da70(main、「Merge pull request #281」)。分岐元は 4389396f で、途中で main を merge commit で取り込んだ
- 読む範囲: 実行コードは `internal/cli/init.go` だけ。テストは `internal/cli/init_packs_test.go`(新規)と `internal/cli/cli_test.go`(呼び出し 1 か所)、shell は `tests/test-check-template.sh` の G。文書は `README.md` の 1 行と `templates/base/docs/recipes/adding-a-language-pack.md` の 1 段落。記録は計画、後工程の report 6 本、insight event
- 後工程は cycle 1 の中で回した。self-review は 4 回(LOW 5 件と LOW 2 件を直した。Run 2 は所見なしだったが、テストの網が狭くなった点を指摘したので設定の組み立てを直した)。cross-review の所見は 0 件。記録だけの修正の例外は使っていない

```
何を変えるか → --pack と --yes(S1) → 試験 G(S2) → 対話の問い(S3) → 文書(S4) → レビューで足した直し → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで: `ralph init` の対話のフォームは、6 つの言語パックを最初から全部選んだ状態で聞いた。`--yes` では全部を入れた。
- この PR から:
  - 対話の問いは、何も選ばない状態から始まる(`--pack` を渡したときは、その名前を選んだ状態)。対象のディレクトリの直下に手がかりのファイルがある言語に「(Recommended)」が付き、最後に skip(言語パックを入れない)がある。skip とほかを一緒に選んだとき、何も選ばなかったときは、同じ問いに残る。
  - `--yes` は、`--pack` が名前を挙げない限り言語パックを入れない。`--pack` は繰り返しても、カンマで区切ってもよい。知らない名前と空の名前は、`--yes` の有無によらず、何も書く前に止まる。
  - 言語パックが 0 個のとき、終わりの表示に `ralph pack add <language>` と使える名前を出す。
  - `TERM=dumb` で stdin が端末でないときは、フォームの前にエラーで止まり、`--yes` を勧める。
- release のノートに書くこと(sync-docs report の「Release notes」に文がある): `--yes` だけでは言語パックが入らなくなった(前と同じにするには `--yes --pack dart,golang,python,rust,terraform,typescript`)。`TERM=dumb` で端末のない stdin では、前は全パックを入れて exit 0 だったが、今はエラーで止まる。manifest の `packs` の並びは、`--yes` では flag の順、対話では名前の順。changelog に出るのは `feat:` 2 件(baa66655、d1c756de)、`fix:` 2 件(f495bb23、80a2eb28)、`refactor:` 1 件(dbc5afaf)。`fix:` と `refactor:` は同じ PR の中の直しなので、ノートでは新しい動きの一部として書く。
- 戻し方: この PR を revert する。manifest の形は変えていない。

## 2. `--pack` と `--yes`(S1、baa66655)

- `internal/cli/init.go:22` の `newInitCmd`: `--pack` は `StringArrayVar` で受ける。`StringSliceVar` は CSV として読むので、空の値(`--pack ''`)を黙って捨て、検査で拾えないため(計画の D5、Codex の計画助言の MEDIUM)。
- `:98` の `parsePackFlag`: カンマで分け、前後の空白を取り、重なりを最初に出た順で 1 つにまとめる。空の名前と知らない名前はエラーで、文に使える名前の一覧を入れる。
- `RunE` はこの検査を、対話と `--yes` のどちらの経路よりも前、ディレクトリを作る前に行う。`executeInit` は知らないパックを警告だけで続け、manifest に名前を残すので、その手前で止める必要がある。
- `--yes` の flag の説明を「--pack が名前を挙げない限り言語パックを入れない」に変えた。
- `executeInit`(`:314` から)は、パックが 0 個のとき Next steps に `ralph pack add <language>` の案内を出す。
- テスト: `TestParsePackFlag_ReturnsThePacksToInstall`(`init_packs_test.go:40`)、`..._RefusesEmptyAndUnknownNames`(`:67`)、`TestInitCmd_YesAloneInstallsNoPackAndPrintsTheHint`(`:129`)、`TestInitCmd_YesWithPackInstallsExactlyThosePacks`(`:149`)、`TestInitCmd_BadPackNameStopsBeforeAnyWrite`(`:183`)。最後のものは、検査が抜けたときにフォームで止まらず失敗で終わるよう、既存の v2 project に対する実行を先に置いている。

## 3. 試験 G(S2、b1b244de)

- `tests/test-check-template.sh` の `run_case_g`: `templates/packs/` の下のパック(`_template` を除く)をカンマでつないだ 1 つの `--pack` で渡し、init のあと各 `packs/languages/<name>/` があることを確かめてから check-template を通す。一覧が空なら G を fail にする。
- 見ること: 計画の S2 の検証コマンド `sh tests/test-check-template.sh` は、macOS の `/bin/sh`(POSIX モードの bash 3.2)では前からある `diff <(...)` で構文エラーになる。shebang の bash で動く `./tests/test-check-template.sh` で確かめた(計画の Progress に記録)。

## 4. 対話の問い(S3、d1c756de)

- `:168` の `skipPackValue = "/skip"`: パックの名前はディレクトリ名なので `/` を含まない。値が名前とぶつからない。
- `:176` の `packMarkers` と `:190` の `recommendedPacks`、`:211` の `isPackMarker`: 計画の「手がかりのファイル」の表を 1 か所に持つ。issue の表と `scripts/detect-changed-languages.sh` の `has_project_marker` の和で、terraform は `classify_language` の行を使う。直下の通常ファイルだけを見て、symlink はたどる。対象のディレクトリがないときは推定しない。
- `:226` の `packOptions`: 名前順の選択肢。推定した言語の表示に「(Recommended)」を添え、値は名前のまま。初期の一覧(`--pack`)にある名前だけを選んだ状態にし、最後に skip を置く。
- `:240` の `validatePackSelection`: skip だけ、または言語だけなら通す。skip と言語を一緒に選んだとき、何も選ばなかったときはエラー。
- `:252` の `packsFromSelection`: フォームの選択から skip の値を取り除いて、入れるパックの一覧にする。skip の値が `executeInit` に届くと、manifest に残るため。
- `:270` の `newPackQuestion`: 問いの組み立て。`Height`(`:283`)を付けた。huh v2.0.3 は高さがないと一覧の高さを「選択肢の行数 − タイトルと説明の行数」にするので、末尾の選択肢が隠れる(変更前の HEAD でも typescript が見えていなかった)。
- `:143` の guard: `TERM=dumb` で stdin が端末でないときは、フォームの前に何も書かずにエラーで止まる。huh は `TERM=dumb` で accessible モードになり、EOF を確定として読むので、確かめの関数がエラーを返す間ずっと問いを出し続ける(`TERM=dumb ralph init dir < /dev/null` が約 70 秒で 675 MB を出して終わらなかった)。
- テスト: `TestSkipPackValue_IsNoPackName`(`:262`)、`TestPackMarkers_NameEveryRepositoryPack`(`:268`)、`TestRecommendedPacks_FindsEachMarker`(`:283`)、`..._LooksOnlyAtFilesDirectlyInTheTarget`(`:321`)、`TestPackOptions_LabelsSelectionAndSkip`(`:382`)、`TestValidatePackSelection`(`:426`)、`TestPacksFromSelection_DropsSkip`(`:453`)、`TestNewPackQuestion_ValidatesTheSelectionOnBlur`(`:481`、TTY なしで `Blur()` と `Error()` から配線を確かめる)、`..._ShowsEveryOptionWithItsLabel`(`:514`)、`TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm`(`:546`、guard が外れても 10 秒の上限で失敗に終わる)、`TestExecuteInit_SkipAloneInstallsNoPackAndPrintsTheHint`(`:578`)。

## 5. 文書(S4、411a3499)

- `init.go` の `Long` の help、`README.md:120` のコマンドの表の行、`templates/base/docs/recipes/adding-a-language-pack.md` の init の段落。sync-docs(4403a317)で、recipe の「知らない名前」を「知らない名前、または空の名前」にした。
- `docs/recipes/adding-a-language-pack.md` は ralph のリポジトリ向けの版で、check-sync の KNOWN_DIFF。init の選び方を書いていないので変えていない。

## 6. レビューで足した直し

- f495bb23(self-review Run 1 の LOW 5 件): help と文書に、`--yes` なしで `--pack` を渡すと選んだ状態で始まることを書いた。一覧の高さの式の各項に名前を付けた。`AvailablePacks()` を 1 回の init で 3 回読んでいたのを 1 回にした。冗長なテストの半分を外した。コメントの symlink の記述を直した。
- dbc5afaf(self-review Run 2 の指摘): f495bb23 で対話の経路の `AvailablePacks` の代入がフォームの先に移り、単体テストの外に出た。`RunE` で `initConfig`(`:116`、ProjectName、Packs、AvailablePacks)を 1 回だけ組んで、`runInitInteractive`(`:129`)と `runInitNonInteractive`(`:290`)の両方に渡す形にした。`--yes` の経路のテストが共有の代入を押さえる。
- 80a2eb28(self-review Run 3 の LOW 2 件): help の「those packs」の指す先がずれていたのを直し、`AvailablePacks` 欄のコメントに問いの選択肢にも使うことを書いた。

## 7. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(HEAD f9df1718)
  - `./scripts/run-test.sh`: 1 回目は rc 1。このブランチが触っていない `TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` が、負荷(load average 19.5)で 3000 ms の期限を超えた。単独で 20 回打つと HEAD も origin/main も 20/20。2 回目の全件は rc 0。時間切れと同じ扱いの 1 回の打ち直しとして記録した
  - `./tests/test-check-template.sh` 60/60、G PASS
  - mutant: 指定した 6 種と追加の 8 種は落ちる。生き残る 6 種(フォームの先の 3 行、`RunE` から `runInitInteractive` への呼び出し、guard の条件 2 つ)のうち 5 種は、組み立てた binary を pty で動かすと落ちる
  - verifier と tester が組み立てた binary を pty で動かし、(Recommended)、何も選ばれていない状態、空と skip+言語で同じ問いに残ること、skip だけでパックが入らず案内が出ること、`--pack golang` が選ばれた状態で始まることを確かめた
  - cross-review(codex、cycle 1/2)の所見は 0 件
- 確かめていないこと:
  - フォームの先の数行は `go test` で実行されない(TTY が要る)。pty でだけ確かめた
  - help は空の名前も失敗することに触れていない(recipe は触れている)
  - `--pack` なしで問いに入ると、何も押す前から確かめの文言が出る。huh が focus の時点で確かめの関数を呼ぶためで、次のキーで消える
  - Linux(CI)での実行、狭い端末での表示
  - sync-docs(4403a317)以降の `run-static-verify.sh`。文書と記録だけの変更で、CI の verify で確かめる
