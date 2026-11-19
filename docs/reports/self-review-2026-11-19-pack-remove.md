# Self-review report: pack-remove

- Date: 2026-11-19(cycle 1 の 2 回目の self-review。`cycle-count.json` は `cycle: 1` のまま)
- Plan: `docs/plans/active/2026-11-19-pack-remove.md`
- Branch: `feat/pack-remove`、HEAD `34f91f36`、base は `main` の `a8c5e289`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff a8c5e289...HEAD` の 13 ファイル(+1831 −12)の diff の質だけ。1 回目の report(`f14a6e89`)のあとに入った直しの commit `42a87923`(L1〜L6 の直し)と、テストを分けた commit `34f91f36` を中心に、ブランチ全体をもう一度読んだ。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify` と `/test` の担当)

## Evidence reviewed

- コード: `internal/cli/pack_remove.go` の全文(434 行)。`42a87923` で足された `packRemoveDoneFiles`、`filesWord`、`packDirAncestorProblem` と、変わった `removePack`、`removeEmptyPackDirs`、`printPackRemoveDryRun`、`Long`。`internal/scaffold/manifest.go` の `Write`、`internal/cli/init.go`、`pack.go`、`upgrade.go` の差分は 1 回目から `upgrade.go` の comment のほかに変わっていないことを確かめた
- 呼び出し先: `upgrade.ValidateRealParentChain` の文(`internal/upgrade/replaceplan.go:631`、rel を含まないので `validatePackRemovePath` の `%s: %w` と重ならない)、`packRelDir` / `packNamespacePrefix` / `packPrefixFor`(`internal/cli/language_pack.go:88-101`)
- 番兵の返し手: `git grep` で `errLegacyLayoutFailClosed` を返す所は `init.go:342`、`pack.go:77`、`pack_remove.go:103`、`eject.go:124`(`requireV2ManifestForOwnership`、`eject.go:57` と `adopt.go:95` が呼ぶ)の 4 か所。直した doc comment の列挙と一致する
- テストの分割(`34f91f36`): 消した行と足した行を `diff` で比べ、違いは新しいファイルの先頭 19 行(package、ファイルの説明、import)と末尾の空行 1 つだけだった。テスト本体は 1 文字も変わっていない。ファイルの説明に挙げた 5 つの場合は、移した 6 つのテストの中身と合う
- テスト(読んだだけで、走らせていない): `42a87923` で足した `TestRemovePack_SymlinkedPacksAncestor_RemovesNoDirectoryOutside`(`pack_remove_paths_test.go:176`)、変えた `TestRemovePack_DryRun_ChangesNothing`(`pack_remove_test.go:381`)、`TestRemovePack_ManifestWriteFails_RerunFinishes`(`:511`)、`TestRemovePack_SymlinkedPackDir_RefusesAndDeletesNothingOutside`(`pack_remove_paths_test.go:126`)、`TestNewPackCmd_RemoveTakesOneLanguageAndDryRun`(`pack_remove_test.go:584`)
- 古い文の残り: `git grep` で `were removed, but`(pack remove の古い文)、`become empty`、`TestManifestWriteCreatesDir`、`refusing to write through` を探した。当たったのは過去の evidence と org の別の文だけで、この PR の文の残りはない
- tech-debt 台帳: pack remove を名指す行はない。この diff で足す行、閉じる行もない

## 1 回目の指摘(L1〜L6)の直り方

| ID | 1 回目の指摘 | 直し(`42a87923`) | 判定 |
| --- | --- | --- | --- |
| L1 | `removeEmptyPackDirs` が `packs` と `packs/languages` の symlink を確かめず、外の空のディレクトリを消しうる | `packDirAncestorProblem` を足し、`removeEmptyPackDirs` は歩き出す前にそれを呼ぶ。`packs` か `packs/languages` があって本物のディレクトリでないとき、または確かめられないときは何も消さず、理由を 1 行出す。dry run も同じ関数の答えを出す。doc comment に祖先の検査と、Lstat が最後の要素しか見ない理由を書いた。テストは `packs` と `packs/languages` の 2 通りで、外の空のディレクトリが残り、rule は消え、manifest から pack が外れることを確かめる | 直っている |
| L2 | 計画の段の断りが「write」の文のままで、何も変えていないと言わない | `removePack` が `planPackRemove` の err を `pack %s: nothing was changed: %w` で包む。`Long` に「A parent directory of a pack file that is a symlink stops the command before anything changes.」を足した。中の `refusing to write through` は残るが、前に「nothing was changed」と key が付くので読み違えない。テストは dry run と本番の両方でこの前置きを確かめる | 直っている |
| L3 | manifest の書き込みの失敗の文が、残したファイルがあっても「the files of pack X were removed」と言う | `packRemoveDoneFiles` が「deleted N unchanged files」と、残したものがあれば「and kept M changed, ejected, or other files」を作り、文は「pack X: …, but writing .ralph/manifest.toml failed, so it was not updated: …」になった。テストは文の頭を丸ごと確かめる | 直っている |
| L4 | dry run が消えるディレクトリを出さず、`Long` が「become empty」と言い、締めの文が残すものがなくても「kept files」と言う | dry run にディレクトリの 1 行(祖先に問題があるときは「It would remove no directories: …」)を足した。`Long`、README、recipe の「become empty」を「are empty afterwards (and packs/languages/ and packs/ when nothing else is in them)」にした。「so ralph no longer tracks the kept files」は `len(plan.kept) > 0` のときだけ付く。テストは両方の場合を確かめる | 直っている |
| L5 | `legacyLayoutFailClosedMsg` と `errLegacyLayoutFailClosed` の doc が返し手を init と pack add だけと書く | 2 つの doc に pack remove、eject、adopt を足した。上の grep の 4 か所と一致する | 直っている |
| L6 | `TestManifestWriteCreatesDir` の名前が、自分で `MkdirAll` する中身と、`Write` の doc に合わない | `TestManifestWrite_NestedPathRoundTrips` に改めた。中の `t.Fatalf("Write to nested path: …")` とも合う | 直っている |

## Findings

指摘はない。CRITICAL、HIGH、MEDIUM、LOW のどれもない。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| — | — | — | — | — |

直しが持ち込んだものとして、次を確かめた。

- `packDirAncestorProblem` の分岐: `packs` がないときは `packs/languages` を見ずに "" を返し、`packs` が本物で `packs/languages` がないときも "" を返す。どちらの場合も、そのあとの `isRealDir` の検査だけで足りる(祖先が本物なので、最後の要素の Lstat で symlink を見分けられる)。`packs` が外への symlink で、その先に `languages` がない形も、`packs` の段で止まる
- 祖先に問題があるときの戻り値 "": 「`packs/languages/<lang>` がなくなった」ときの "" と同じ値になるが、doc comment は両方を書いており、呼び手は "" のとき「left in place」の行を出さないだけで、代わりに `⚠ Removed no directories` の行が出ている。取り違えは起きない
- `nothing was changed` で包む範囲: `planPackRemove` が返す err はどれも、ファイルを 1 つも消す前(dry run なら何もしない段)に出るので、包んだ文は常に正しい。`ApplyOps` の段の失敗は前と同じく「.ralph/manifest.toml was not changed」とだけ言い、途中まで消えた場合と矛盾しない
- `errors.Is` は `%w` で包んだあとも中の err に届く

## 読んだが指摘にしないもの

- 祖先の検査の文は `packs is not a real directory (mode Lrwxr-xr-x)` のように mode をそのまま出す。同じファイルの `packRemoveNonRegularReason` は symlink を言葉で言う。「not a real directory」で意味は通るので上げない
- 祖先に問題があるときと、空のディレクトリを消せなかったときの `⚠` の行は、`✓ Pack … removed` の行より前に出る。`removeDirIfEmpty` の `⚠` の行が 1 回目から同じ順なので、それに合わせた形
- dry run の「its %d entries」は 1 件でも複数形のまま。`filesWord` を足した文と揃っていないが、1 回目からの文で、読み違えはない
- 番兵の doc の外にも、ralph の動詞を並べる comment が 3 つある。`internal/cli/doctor.go:773` は Meta.Packs を保つものを「Both `ralph init` … and `ralph pack add`」と書き、pack remove も Meta.Packs を書くようになった今は 3 つめが抜けている。`doctor.go:1238` と `eject.go:112` も pack remove を挙げない。どれもこの diff の外のファイルで、「Meta.Packs が入れた pack を表す」という主張そのものは pack remove が pack を外すので今も正しい。直すなら `doctor.go:773` の「Both」を「`ralph init`, `ralph pack add`, and `ralph pack remove`」にする
- 1 回目の「読んだが指摘にしないもの」(hash の読みと削除のあいだの窓、`Meta.Packs` に core の rule の名前を入れる改ざん、`Chmod(0o644)` と umask、kill されたときの `.manifest.toml.*.tmp`、`Write` の err で path が 2 度出ること、`case scaffold.OwnerCore:` の空の case、記録の hash が空のとき)は、直しで変わっていない。同じ理由で上げない

## Positive notes

- L1 の直しは、Lstat が最後の要素しか symlink と見ない理由を、検査する関数の doc comment に書いている。次に片付けの対象を増やす人が、なぜ祖先を確かめるのかを定義の場所で読める
- dry run と本番が同じ `packDirAncestorProblem` を呼ぶので、dry run の「It would remove no directories」と本番の `⚠ Removed no directories` が食い違わない。1 回目に良い点として挙げた「dry run が約束したことを本番が断らない」形を、ディレクトリの片付けにも広げている
- 新しいテストは、外の空のディレクトリが残ることを中身で確かめ、`packs` と `packs/languages` の 2 通りを sub-test で回す。外の場所の計算(`strings.TrimPrefix`)には何を指すかの comment がある
- テストの分割は中身を変えずに移しただけで、`pack_remove_test.go` は 622 行、`pack_remove_paths_test.go` は 247 行になり、1 回目に書いた 800 行の手前で分かれた

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| — | — | — | — | — |

台帳に足した行はない。1 回目の L1〜L6 はすべてこの PR の中で直り、先送りにしたものはない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM、LOW のどれもなし)
- Follow-ups:
  - 「読んだが指摘にしないもの」の `doctor.go:773` の「Both」は、diff の外の comment の直しなので、この PR で直すかどうかは実装側の判断に任せる
  - 1 回目に書いた、pack が書く場所が `renderPackInto`、upgrade の `preserve`、`isPackRemoveKey` の 3 つに別々に書かれている件は変わっていない。pack が新しい場所に書くようになったら 3 つを揃えて直す
