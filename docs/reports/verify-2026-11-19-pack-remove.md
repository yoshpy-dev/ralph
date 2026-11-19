# Verify report: pack-remove

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-pack-remove.md`。承認の digest `88ea9bb6774f` は `./scripts/plan-visual.sh digest` で計算し直した値と一致する。issue #344
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff a8c5e289...HEAD`(base は `main` の `a8c5e289`、HEAD `4e98b216`、14 ファイル、+1833 −14)。self-review の 2 回目のあとに入った `4e98b216`(`internal/cli/doctor.go` の comment 1 か所)も含む。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-pack-remove.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない。静的検査の全出力と、作った binary で打った観察の記録を入れた)

## Spec compliance

「Met」は、コードを読み、対応するテストのソースを読み、作った binary で観察できるものは観察した、という意味。テストが通ることは `/test` で確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: golang を外すと `Meta.Packs`、`packs/languages/golang/` の下、rule、manifest の項が消え、`packs/` が空なら消え、`checkScaffoldIntegrity` の 5 つが strict でも pass し、`Pack: golang` の行が出ない | Met | 本体は `internal/cli/pack_remove.go:88-131`(計画、`ApplyOps` での削除、空のディレクトリの片付け、項と `Meta.Packs` の削除、manifest の書き込み)。テストは `TestRemovePack_UnmodifiedPack_DeletesFilesEntriesAndPacksDir`(`pack_remove_test.go:87`。`assertAllScaffoldChecksPass` は strict の false と true の両方で 5 つを確かめる、`doctor_scaffold_test.go:34`)。観察: golang と typescript を入れた scratch の repo で `ralph pack remove golang` が rc=0、`packs = ['typescript']`、manifest に golang は 0 件、rule は消え、そのあとの `ralph doctor --strict` が rc=0 で 5 つとも pass、`Pack:` の行は typescript だけ |
| AC-2: 外したあと次の世代の `ralph upgrade` でも golang は戻らない | Met(テストのソースとコード) | upgrade は `oldManifest.Meta.Packs` の pack だけを desired に入れる(`internal/cli/upgrade_v2.go:400`)。テストは `TestRemovePack_UpgradeAfterwardDoesNotReinstallThePack`(`pack_remove_test.go:125`、gen1 から gen2)。観察は同じ版の upgrade だけで、`Upgrade no-op: dev` になり golang は戻らなかった。次の世代での観察はしていない |
| AC-3: 変更したファイルと eject したファイルは残り、名前と理由が出て、項は外れ、doctor --strict が通り、続く upgrade も変えない | Met | 分け方は `packRemoveKeepReason`(`pack_remove.go:229`。通常のファイルでない、owner が core でない、hash が記録と違う、のどれかで残す)。テストは `TestRemovePack_KeepsModifiedAndEjectedFiles`(`pack_remove_test.go:146`、gen2 への upgrade のあとも中身が同じ)。観察: verify.sh を変え、rule を symlink に、README.md をディレクトリにした typescript で、3 つとも理由つきで残り(`modified since ralph wrote it`、`not a regular file (symlink)`、`not a regular file (directory)`)、doctor --strict が rc=0 |
| AC-4: 入っていない pack はエラーで、disk も manifest も変わらない | Met | `pack_remove.go:105-107` と `packNotInstalledError`(`:146`)。テストは `TestRemovePack_NotInstalled_RefusesWithoutWriting`(`:277`、木と manifest の bytes を比べる)と `TestRemovePack_NotInstalled_NoPacksSaysNone`(`:299`)。観察: 外したあとの `ralph pack remove golang` が rc=1 で `pack "golang" is not installed in this project (installed packs: typescript)` |
| AC-5: legacy では `errLegacyLayoutFailClosed` を返して何も書かない。manifest がなければエラー | Met | `pack_remove.go:103` が番兵を包まずに返す。manifest がないときは `reading manifest: %w`(`:100`)。テストは `TestRemovePack_LegacyManifest_FailsClosed`(`:337`、`err.Error() == legacyLayoutFailClosedMsg` で包んでいないことまで確かめる)と `TestRemovePack_NoManifest_Errors`(`:368`) |
| AC-6: `--dry-run` は消すものと残すものを出し、何も変えない | Met | `pack_remove.go:113-116` が削除の前に返す。テストは `TestRemovePack_DryRun_ChangesNothing`(`:381`)と、cobra の配線を通す `TestNewPackCmd_RemoveTakesOneLanguageAndDryRun`(`:584`)。観察: dry run の前後で manifest の sha256 が一致し、ファイルも残った |
| AC-7: ほかの pack(typescript)のファイルと項は変わらない | Met | 対象は `isPackRemoveKey` の 3 か所だけ(`pack_remove.go:158`)。テストは `TestRemovePack_LeavesTheOtherPackUntouched`(`:419`、項は値まで比べる)。観察: golang を外したあとも `packs/languages/typescript` と `typescript.md` が残り、doctor の `Pack: typescript` は pass |
| AC-8: 手で消したあとの repo でも項を外して終わり、doctor --strict が通る | Met(テストのソースとコード) | disk にない項は `plan.missing` に入り、項だけ外す(`pack_remove.go:184-187`)。テストは `TestRemovePack_FilesDeletedByHand_DropsTheEntries`(`:485`) |
| AC-9: README の Commands の表と Language packs の節、templates の recipe に remove が書かれている | Met(文の中身に V1 あり) | `README.md:136`(Commands の表)、`README.md:326-333`(Language packs の節)、`templates/base/docs/recipes/adding-a-language-pack.md:40-69`(「Removing a pack」)。書かれてはいるが、残したものを `ralph pack add` が上書きするという 1 文が symlink とディレクトリには合わない(V1) |
| AC-10: `Manifest.Write` の rename が失敗しても元の manifest が読め、一時ファイルが残らない。成功したときの中身は前と同じ toml | Met(テストのソースとコード) | `internal/scaffold/manifest.go:107-146`(同じディレクトリに `CreateTemp`、`Chmod(0o644)`、`Sync`、`Close`、`renameManifestFile`、失敗したら `os.Remove(tmpPath)`)。テストは `TestManifestWrite_RenameFailureKeepsOldManifestAndRemovesTempFile`、`TestManifestWrite_FreshPathRoundTripsWithMode0644AndNoTempFile`(bytes を `toml.Marshal(m)` と比べる)、`TestManifestWrite_OverwriteReplacesExistingContent`、`TestManifestWrite_MissingParentDirFailsAndCreatesNothing`(`manifest_test.go`) |
| AC-11: ファイルを消したあと manifest の書き込みが失敗するとエラーになり、元の manifest が残り、打ち直すと終わる | Met(テストのソースとコード) | 書き込みは `writePackRemoveManifest` を通る(`pack_remove.go:54-56`、`:127`)。テストは `TestRemovePack_ManifestWriteFails_RerunFinishes`(`:511`。失敗の文、元の manifest の bytes、打ち直しの `3 already missing`、そのあとの strict の 5 つ) |

### issue #344 の受け入れ条件 3 つ

| issue #344 の条件 | 対応する AC | Status |
| --- | --- | --- |
| `ralph pack remove golang` で manifest の packs と golang のファイルが消え、`ralph doctor --strict` が通る | AC-1、AC-2 | Met。作った binary で、`pack remove golang` のあとの `ralph doctor --strict` が rc=0 だった |
| 変更されたファイルは消さず、名前を出す | AC-3 | Met。binary で、変更した verify.sh が理由つきで残った |
| 入っていない pack はエラー | AC-4 | Met。binary で rc=1 |

### 計画の Scope との照合

- 名前が 1 つのパスの要素でないとき、manifest を読む前に断る: `validatePackRemoveName`(`pack_remove.go:139`)を `removePack` の最初に呼ぶ。binary でも `ralph pack remove ../x` が rc=1
- 削除は `upgrade.ApplyOps` の `OpDelete` に任せ(`:269`)、計画の段で各 key の親の連なりを `upgrade.ValidateRealParentChain` で確かめる(`:219`)
- 計画からの足し(manifest の key を整えたあと pack の 3 か所から外れるものは何も書かずにエラー)は、計画の Progress checklist の slice 2 に書かれている。テストは `TestRemovePack_ManifestKeyEscapingThePack_Refuses`(`pack_remove_paths_test.go:227`)
- `pack.go` は `AddCommand(newPackRemoveCmd())` の 1 行だけで、`addPack` の doc は変わっていない
- `origin/main` は base と同じ `a8c5e289` のまま(`git fetch` のあとで確かめた)。`git merge-tree --write-tree --name-only origin/main HEAD` は rc=0 で衝突はない

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc=0、`All verifiers passed.` | 2026-11-19T11:54:56Z、HEAD `4e98b216` で取った。言語は golang。中身は下の行 |
| gofmt / `go vet ./...` / `golangci-lint run ./...` / `staticcheck ./...` | pass | `gofmt: ok`、golangci-lint は `0 issues.`。go vet と staticcheck は何も出さずに終わった。`Skipping` の行はないので 4 つとも走った |
| shellcheck、hook の `sh -n`、guard の awk、`jq -e` | OK | この diff はシェルを変えていない。wrapper が毎回回すもの |
| `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6。`docs/recipes/adding-a-language-pack.md` は KNOWN_DIFF(計画の Scope のとおり、root の貢献者向けの recipe は変えていない) |
| `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照 | OK | template の recipe に meta-repo の名前は入っていない |
| branch の secret scan | clean | `scanned a8c5e289..4e98b216 against origin/main: clean` |
| `./scripts/plan-visual.sh digest` | `88ea9bb6774f` | 計画の `- Approved:` の digest と一致 |
| `git diff --check a8c5e289...HEAD` | rc=0 | 行末の空白などはない |
| 変わった 14 ファイルの U+FFFD | 0 件 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph pack --help` | Yes | `Available Commands` に `remove      Remove a language pack from the project` が出る(作った binary で確かめた) |
| `ralph pack remove --help` | 一部ずれ(V1) | 削除と片付けと manifest の扱い、`--dry-run`、legacy、打ち直しの説明はコードと合う。「ralph pack add <language> would overwrite them.」(`pack_remove.go:36`)は、直前の文が挙げる symlink とディレクトリには当たらない |
| `ralph init --help` | Yes | 「A pack can be added later with ralph pack add <language> and removed with ralph pack remove <language>.」(`init.go:38-40`、`Long` の一部) |
| `README.md` の Commands の表 | Yes | `README.md:136` に `ralph pack remove <lang> [--dry-run]` の行。`ralph init` の行の末尾にも remove を足した(`:129`)。中身はコードと合う |
| `README.md` の Language packs の節 | 一部ずれ(V1) | `:326-333`。「A file you changed, ejected, or replaced with a symlink or directory is kept and listed; ralph stops tracking it, `ralph pack add golang` overwrites it」の後半が symlink とディレクトリに合わない。run-verify.sh についての文は `templates/base/scripts/run-verify.sh:171-173`(実行可能な `packs/languages/<lang>/verify.sh` を打つ)と合う |
| `templates/base/docs/recipes/adding-a-language-pack.md` | 一部ずれ(V1) | 「Removing a pack」の節(`:40-69`)。`:60-63` が symlink とディレクトリを挙げた直後に「`ralph pack add <language>` overwrites them.」と書く |
| `pack remove` の実行時の出力 | 一部ずれ(V1) | 残したものがあるときの締めの文「note that ralph pack add <lang> overwrites them」(`pack_remove.go:429`)は、残したものの種類によらず出る |
| `AGENTS.md` の repo map | Yes | `internal/cli/` の一覧は `pack` を挙げるだけで、サブコマンドは並べていない。`internal/scaffold/` の説明は manifest の書き方に触れていない |
| doc comment(`doctor.go:772-775`、`upgrade.go:87-110`) | Yes | `Meta.Packs` を保つものと番兵を返すものに pack remove が入った。番兵を返す所は `init.go`、`pack.go`、`pack_remove.go:103`、`eject.go` の `requireV2ManifestForOwnership` の 4 か所で、comment の列挙と合う |
| `docs/tech-debt/README.md` | ずれ(V2) | 415 行の Trigger「The next change to any of the five.」の 5 つに `internal/cli/doctor.go` が入っていて、`4e98b216` が comment を変えた(1648 → 1649 行)。持ち越しの日付つきの注記がない |
| 計画の受け入れ条件のチェック欄 | 遅れ(I2) | AC-1〜AC-11 は `[ ]` のまま。Progress checklist の「Review artifact created」も、self-review の report(`f14a6e89`、`16e79b18`)があるのに `[ ]` |

### V1 の確かめ方(作った binary での観察)

scratch の repo で、typescript の rule を repo の外のファイルへの symlink に、`packs/languages/typescript/README.md` をディレクトリに置き換え、`ralph pack remove typescript` で 2 つが理由つきで残ったあとに `ralph pack add typescript` を打った。

- ディレクトリが残っていると、`ralph pack add typescript` は rc=1 で `rendering pack typescript: open .../packs/languages/typescript/README.md: is a directory` を出して止まった。上書きはされない
- ディレクトリを消して打ち直すと rc=0 になり、symlink はそのまま残って、その先の repo の外のファイルの中身が typescript の rule に置き換わった。`scaffold.RenderFS` の comment(`internal/scaffold/render.go:82-85`)にある、上書きのときは symlink をたどるという挙動のとおり
- 変更したファイル(verify.sh)は上書きされた。ここは文のとおり

`ralph pack add` の挙動を変えないことは計画の Non-goals に入っていて、それは守られている。計画の Non-goals は「残ったファイルは add し直すと上書きされる」とファイルについて書いているが、文書と出力はその主語を symlink とディレクトリまで広げた。symlink のときは repo の外のファイルが上書きされうるので、利用者が読み違えると困る。

## Observational checks

作った binary(`go build ./cmd/ralph`、HEAD `4e98b216`)を `$TMPDIR` の scratch の git repo で打った。記録は evidence の log の末尾にある。

- `ralph init --yes --pack golang,typescript` → `ralph doctor --strict` rc=0
- `ralph pack remove golang --dry-run`: `3 to delete, 0 to keep, 0 already missing`、manifest の sha256 は変わらない
- `ralph pack remove golang`: `✓ Pack golang removed (3 deleted, 0 kept, 0 already missing)`、`ralph doctor --strict` rc=0
- `ralph upgrade --yes`: 同じ版なので `Upgrade no-op`。golang は戻らない
- `ralph pack remove golang`(2 回目)rc=1、`ralph pack remove ../x` rc=1
- 変更、symlink、ディレクトリを混ぜた typescript の remove → 3 つ残り、doctor --strict rc=0。そのあとの `pack add` の挙動は V1 に書いた

## Coverage gaps

- テストはどれも走らせていない。AC-2、AC-8、AC-10、AC-11 はテストのソースとコードを読んだだけで、通ることは `/test` で確かめる
- AC-2 の観察は同じ版の upgrade(no-op)だけ。次の世代の template での upgrade は、テストの gen1 → gen2 に頼る
- 下流の本物の repo(ralph 8.2.1 で init したもの)では打っていない。scratch の repo と dev の binary での観察だけ
- `Manifest.Write` の途中で process が kill された場合(`.manifest.toml.*.tmp` が残る)は確かめていない。self-review が「読んだが指摘にしないもの」に挙げている
- 決まった手順の検査で足りないもの: 残したものに対する `ralph pack add` の挙動を固めるテストがない。V1 の文を直すなら、kept のディレクトリで add が止まることと、kept の symlink を add がたどることを 1 つずつ確かめるテストを足すと、文と挙動がまたずれたときに気づける

## Findings

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V1 | MEDIUM | 「残したものは `ralph pack add` が上書きする」という文が、残すものとして挙げた symlink とディレクトリに合わない。ディレクトリでは add がエラーで止まり、symlink では add がその先(repo の外でもよい)を上書きする。4 か所: `pack_remove.go:36`(help)、`:429`(実行時の出力)、`README.md:333`、`templates/base/docs/recipes/adding-a-language-pack.md:63` | 文を変更したファイルと eject したファイルに絞り、ディレクトリでは add が止まること、symlink では add がその先に書くことを書く。help と出力は Go の文字列で、`TestRemovePack_KeepsModifiedAndEjectedFiles` が `ralph pack add golang overwrites them` を確かめているので、テストも合わせて変える。`/pr` の前に直す |
| V2 | LOW | tech-debt 415 行の Trigger を `internal/cli/doctor.go` の変更(`4e98b216`、1648 → 1649 行、comment のみ)が満たしたが、持ち越しの注記がない | `/sync-docs` で、ほかの branch と同じ形の日付つきの注記を足す(comment だけの変更で、移すだけの分割は Scope の外、ほかの 4 つは触れていない) |
| I1 | INFO | tech-debt 117 行の Trigger(`docs/recipes/adding-a-language-pack.md` の template の中身の組み替え)は、template の recipe に節が 1 つ足されたので、当たると読むこともできる。root の recipe は貢献者向けのまま変えておらず、KNOWN_DIFF はまだ要る(計画の Scope にその判断がある) | 当たると読むなら、KNOWN_DIFF は残すという注記を `/sync-docs` で足す。記録だけでもよい |
| I2 | INFO | 計画の AC-1〜AC-11 のチェック欄と「Review artifact created」が `[ ]` のまま | `/sync-docs` か `/pr` で更新する |
| I3 | INFO | `doctor.go:1239` と `eject.go:112` の comment は v2 の壁を持つ動詞を eject、adopt、pack add と挙げ、pack remove を挙げない。例を挙げた文で、今も誤りではない。self-review も同じ判断 | 記録だけ |
| I4 | INFO | help と文書は残す理由として「変更、eject、通常のファイルでない」を挙げるが、owner が seed か空の項も残る(`pack_remove.go:229-243`、`TestRemovePack_NonCoreOwnersAndStaleDiskHash_AreKept`)。init と add は pack のパスを core にするので、普通は起きない | 記録だけ |
| I5 | INFO | 古い仕様 `docs/specs/2026-04-16-ralph-cli-tool.md:295` は、release で消えた pack は upgrade が `Meta.Packs` から外すと書く。今の v2 の upgrade は残し続け(`upgrade_v2.go:409-415` の `preserve`)、計画の Design decisions はそれを前提にしている。この diff より前からのずれ | 記録だけ |

## Insight event

`./scripts/insights-append.sh --slug pack-remove --flow standard --phase verify --verdict pass --cycle auto --medium 1 --low 1 --source skill` で 1 行足した。verdict の値に partial pass はないので pass と記録した。

## Verdict

- Verified: 静的検査(`run-static-verify.sh` rc=0。gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、check-template-purity、secret scan を含み、HEAD `4e98b216` で取った)、計画の digest、`git diff --check`、U+FFFD 0 件、`origin/main` との試しの merge。AC-1〜AC-11 の本体とテストの対応(file:line とテスト名)。作った binary での AC-1、AC-3、AC-4、AC-6、AC-7 と issue #344 の 3 条件の観察。`ralph pack --help`、`ralph pack remove --help`、`ralph init --help` の文
- Partially verified: AC-2(同じ版の upgrade の観察とコード。次の世代はテストのソースだけ)、AC-8、AC-10、AC-11(コードとテストのソース。通るかは `/test`)、AC-9(書かれてはいるが V1 の文が残る)
- Not verified: テストの実行、下流の本物の repo での実行、`Manifest.Write` の途中での kill
- Verdict: **partial pass**。満たさない AC はなく、静的検査はすべて通った。V1(MEDIUM)は help、実行時の出力、README、recipe の 4 か所の文が symlink とディレクトリについて挙動と合わないもので、`/pr` の前に直す。Go の文字列とテストを変えるので、`/sync-docs` だけでは閉じない。V2(LOW)は `/sync-docs` で直す。I1〜I5 は記録だけ
