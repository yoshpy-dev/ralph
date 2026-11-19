# Test report: pack-remove

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-pack-remove.md`(issue #344)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `039e0d7f`(verify の V1 を直した commit。worktree は clean)、base は `main` の `a8c5e289`。`./scripts/run-test.sh` を worktree のルートから 1 回だけ打った。scope は wrapper の既定の changed で、選ばれた pack は golang(`==> Language packs selected: golang`)。local verifier は scope によらず `tests/*.sh` を全部回す。run のあと、AC の対応と変異の確かめのために、絞った `go test` を打った(下の「追加の run」)
- Evidence: `docs/evidence/test-2026-11-19-pack-remove.log`(5,619 行、run-test.sh の出力そのもの。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-19 21:11:24 〜 21:38:18 JST、1,614 s、rc 0、`==> All verifiers passed.` |
| local verifier(`tests/*.sh` 45 本) | 45 本 | 45 本 | 0(`FAIL` で始まる行は `FAIL: 0` の集計だけ) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`git 2.41 より古い本物の git` の場合(手元は git 2.49.0) | 約 10.5 分(21:11:24 〜 21:21:50 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` のパッケージもない | 約 16.5 分。下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.990 s |
| `internal/cli` | ok | 859.364 s |
| `internal/config` | ok | 11.627 s |
| `internal/gitroot` | ok | 2.365 s |
| `internal/insights` | ok | 2.730 s |
| `internal/org` | ok | 967.274 s |
| `internal/org/driver` | ok | 4.569 s |
| `internal/org/protocol` | ok | 1.811 s |
| `internal/scaffold` | ok | 2.727 s |
| `internal/toolchain` | ok | 25.608 s |
| `internal/upgrade` | ok | 4.179 s |

実行の条件:

- 負荷(`sysctl -n vm.loadavg` を 30 s ごと): 1 分値は 6.65 〜 19.82。Go の段のあいだ、ほかのセッションの `go test ./internal/org/ ./internal/cli/` と `go test ./internal/org/ -run 'Gate|Regate'` が並んで動いていた。こちらからは何も並べていない
- 空き容量: 155 〜 157 GiB。ログに `no space left` は 0 件
- `GOFLAGS` は足していない。golang の pack が `-timeout=30m` を入れる。`internal/org` の 967 s は上限の 54%、`internal/cli` の 859 s は 48%
- 起動: Bash ツールの `run_in_background` で `$TMPDIR` の runner を動かした。runner は pid、始まりと終わりの時刻、rc を meta file に書き、負荷と空き容量を 30 s ごとに記録する。`nohup` や `&` は使っていない。待つのは `caffeinate -w <pid>` を 175 s ずつ区切って繰り返した
- run-verify.sh は自分の evidence も `docs/evidence/verify-2026-11-19-121124.log` に書いた(同じく追跡外)

落ちたテストがないので、打ち直しはしていない。

### 追加の run(run-test.sh が終わったあと)

| Command | 結果 | 時間 |
| --- | --- | --- |
| `go test ./internal/scaffold/ -count=1 -coverprofile=...` | ok、coverage 78.7% | 0.813 s |
| `go test ./internal/cli/ -count=1 -timeout 30m -v -run 'Init\|Upgrade\|Eject\|Adopt\|Migrat\|Pack\|Doctor\|Scaffold' -coverprofile=...` | ok。top-level 280 PASS、0 FAIL、0 SKIP(subtest を含めると 298 PASS 行が別にある) | 55.6 s |
| 変異 4 つと使い捨てのテスト 2 本(`-overlay`、下の「変異の確かめ」) | 下の表 | 1 本あたり 1 〜 10 s |

overlay の写しは `$TMPDIR` に置いた。前後で `git status --porcelain` は同じで、worktree は変えていない。

## 計画のテストと AC の対応

全パッケージの run が通ったので、次のテストはすべて通っている。pack remove の新しいテストは 22 本(`pack_remove_test.go` 16 本、`pack_remove_paths_test.go` 6 本)、`Manifest.Write` のテストは 5 本(新しい 4 本と、名前だけ変えた 1 本)。

| AC / Test plan の項 | テスト | 結果 |
| --- | --- | --- |
| AC-1(golang を外すとファイル、項、`Meta.Packs`、空の `packs/` が消え、strict でも 5 つが pass、`Pack: golang` が出ない) | `TestRemovePack_UnmodifiedPack_DeletesFilesEntriesAndPacksDir`(`assertAllScaffoldChecksPass` は strict の false と true の両方で 5 つを見る、`doctor_scaffold_test.go:34`) | pass |
| AC-2(外したあと次の世代の upgrade でも戻らない) | `TestRemovePack_UpgradeAfterwardDoesNotReinstallThePack`(gen1 から gen2、`runUpgradeIOWithOptions`) | pass |
| AC-3(変更したファイルと eject したファイルは残り、理由が出て、項は外れ、strict が通り、続く upgrade も変えない) | `TestRemovePack_KeepsModifiedAndEjectedFiles`(gen2 への upgrade のあとも中身が同じ)、`TestRemovePack_NonCoreOwnersAndStaleDiskHash_AreKept`(seed、owner なし、DiskHash の違い) | pass |
| AC-4(入っていない pack はエラーで、何も変わらない) | `TestRemovePack_NotInstalled_RefusesWithoutWriting`(木と manifest の bytes を比べる)、`TestRemovePack_NotInstalled_NoPacksSaysNone` | pass |
| AC-5(legacy は番兵をそのまま返して何も書かない、manifest がなければエラー) | `TestRemovePack_LegacyManifest_FailsClosed`(`err.Error() == legacyLayoutFailClosedMsg`)、`TestRemovePack_NoManifest_Errors` | pass |
| AC-6(`--dry-run` は消すものと残すものを出し、何も変えない) | `TestRemovePack_DryRun_ChangesNothing`、`TestNewPackCmd_RemoveTakesOneLanguageAndDryRun`(cobra を通す) | pass |
| AC-7(typescript のファイルと項は変わらない) | `TestRemovePack_LeavesTheOtherPackUntouched`(項は値まで比べる) | pass |
| AC-8(手で消したあとでも項を外して終わり、strict が通る) | `TestRemovePack_FilesDeletedByHand_DropsTheEntries` | pass |
| AC-9(README と templates の recipe に remove が書かれている) | 文書の AC で、テストはない。`README.md:136`、`README.md:329-333`、`templates/base/docs/recipes/adding-a-language-pack.md:46-50` に書かれていることを grep で見た。文の中身は verify の担当(V1 は `039e0d7f` で直った) | テストなし |
| AC-10(rename が失敗しても元の manifest が読め、一時ファイルが残らない。成功したときの中身は前と同じ toml) | `TestManifestWrite_RenameFailureKeepsOldManifestAndRemovesTempFile`、`TestManifestWrite_FreshPathRoundTripsWithMode0644AndNoTempFile`(bytes を `toml.Marshal(m)` と比べ、mode 0644、ディレクトリには manifest だけ)、`TestManifestWrite_OverwriteReplacesExistingContent`、`TestManifestWrite_MissingParentDirFailsAndCreatesNothing` | pass |
| AC-11(ファイルを消したあと manifest の書き込みが失敗するとエラーで、元の manifest が残り、打ち直すと終わる) | `TestRemovePack_ManifestWriteFails_RerunFinishes`。書き込み全体を `writePackRemoveManifest` で差し替えて失敗させる。本物の `Manifest.Write` が途中で失敗しても元のファイルが残ることは AC-10 のテストが受け持つ | pass |
| Unit: 項ごとの分類(消す、残す、項だけ外す)、名前の形が不正な pack | 上の AC-1、3、8 のテスト、`TestRemovePack_MalformedName_RefusesBeforeReadingAnything`(6 つの名前を manifest のある dir とない dir で) | pass |
| Integration: upgrade で `DiskHash` が付いたあとの remove | `TestRemovePack_AfterUpgrade_ComparesWithTheRecordedDiskHash`(Hash を古くしても消える) | pass |
| Regression: pack add、init の help、manifest を書く init・upgrade・eject・adopt・migrate の既存のテスト | 全パッケージの run で通った。下の「Regression checks」に呼び手ごとの確かめを書いた。init の help の文を押さえるテストはもともとない | pass |
| Edge: 変更したファイル、eject した rule | `TestRemovePack_KeepsModifiedAndEjectedFiles` | pass |
| Edge: symlink の pack のファイル | `TestRemovePack_SymlinkAtAPackPath_KeptAndTargetUntouched` | pass |
| Edge: pack のディレクトリに利用者が足したファイル | `TestRemovePack_UserFileInPackDir_KeepsTheDirectory` | pass |
| Edge: retired の pack | `TestRemovePack_RetiredPack_DeletesByRecordedHash` | pass |
| Edge: 2 つの pack のうち 1 つだけ外す、手で消したあと | AC-7、AC-8 のテスト | pass |
| 計画にない足し: ディレクトリの pack のパス、symlink の pack のディレクトリと祖先、pack の外へ出る manifest の key | `TestRemovePack_DirectoryAtAPackPath_KeptEvenWhenEmpty`、`TestRemovePack_SymlinkedPackDir_RefusesAndDeletesNothingOutside`、`TestRemovePack_SymlinkedPacksAncestor_RemovesNoDirectoryOutside`(`packs`、`packs/languages`)、`TestRemovePack_ManifestKeyEscapingThePack_Refuses` | pass |
| Scope の 1 か所: migration が残した旧い rule の場所 `.claude/rules/<lang>.md` の項 | テストがない。`isPackRemoveKey` からこの場所を外す変異 M1 が残った | gap(G1) |

## 変異の確かめ

worktree のソースは変えず、`$TMPDIR` の写しを `go test -overlay` で当てた。scaffold の変異はパッケージ全体、cli の変異は `-run 'Pack'`(pack remove の 22 本を含む 59 本)で回した。`isPackRemoveKey` と `packRemoveKeepReason` を呼ぶのは `removePack` だけで、`removePack` を呼ぶテストはすべて名前に Pack を含む。

| ID | 変異 | 落ちたテスト | 結果 |
| --- | --- | --- | --- |
| M1 | `isPackRemoveKey`(`pack_remove.go:160`)から `\|\| p == legacyPackRuleRelPath(lang)` を消す | なし(59 本すべて pass) | survived(G1) |
| M2 | `Manifest.Write` の失敗の後始末 `_ = os.Remove(tmpPath)`(`manifest.go:143`)を消す | `TestManifestWrite_RenameFailureKeepsOldManifestAndRemovesTempFile`(`directory holds [.manifest.toml.326860479.tmp manifest.toml]`) | killed |
| M3 | `tmp.Chmod(0o644)`(`manifest.go:131`)を `err = nil` に替える | `TestManifestWrite_FreshPathRoundTripsWithMode0644AndNoTempFile`(`mode = 600, want 644`) | killed |
| M4 | `packRemoveKeepReason` が `DiskHash` を読まず `Hash` だけと比べる(`pack_remove.go:247-250`) | `TestRemovePack_AfterUpgrade_ComparesWithTheRecordedDiskHash`、`TestRemovePack_NonCoreOwnersAndStaleDiskHash_AreKept` | killed |

M1 を見分けるテストを使い捨てで書き、overlay で足した(`TestProbe_RemovePack_LegacyRulePathEntryIsRemoved`)。`initV2Project(gen1)` のあと、`.claude/rules/golang.md` を書いて owner=core、hash 一致の項を manifest に足し、`ralph pack remove golang` で 4 つが消え、そのファイルが消え、manifest に golang の項が残らず、strict の 5 つが pass することを確かめる。HEAD では pass、M1 では 3 つの assert(`4 deleted` の集計、ファイルの削除、manifest の項)で落ちた。M1 のままでも strict の 5 つは pass したので、この穴は doctor では見つからない。

`Manifest.Write` の文書にある 2 つの挙動も、使い捨てのテストで HEAD のふるまいを確かめた。どちらも pass した。

- manifest の場所が symlink なら、Write のあとは通常のファイルに置き換わり、symlink の先は変わらない(`TestProbe_SymlinkAtPathIsReplacedNotFollowed`)
- mode 0600 の既存の manifest は Write のあと 0644 になる(`TestProbe_ExistingMode0600BecomesMode0644`)

## Coverage

- Statement: `internal/scaffold` 78.7%(`go test ./internal/scaffold/ -count=1`)。`Manifest.Write` 95.2% で、通らないのは前からある `toml.Marshal` の失敗の return(`manifest.go:122`)だけ。`internal/cli` は上の filter つきの run で 52.2%(filter をかけたので、パッケージの値とは比べられない)
- Branch: 測る道具がない
- Function(`pack_remove.go`、filter つきの cli の run): `newPackRemoveCmd`、`validatePackRemoveName`、`packNotInstalledError`、`isPackRemoveKey`、`packRemoveKeys`、`applyPackRemoveDeletes`、`isRealDir`、`printPackRemoveDryRun`、`printPackRemoveSummary` が 100%。`removePack` 93.3%、`removeEmptyPackDirs` 96.2%、`packRemoveKeepReason` 93.3%、`planPackRemove` 89.5%、`validatePackRemovePath` 88.9%、`packDirAncestorProblem` 88.9%、`removeDirIfEmpty` 85.7%、`packRemoveNonRegularReason` 75.0%、`packRemoveDoneFiles` 75.0%、`filesWord` 66.7%、`packRemoveDisplayPath` 0.0%
- Notes: 通らないブロックは次のとおり。
  - `pack_remove.go:120`: 計画の検査を通ったあと `upgrade.ApplyOps` が失敗したときのエラー(「.ralph/manifest.toml was not changed」)。symlink の親はどのテストでも計画の段(`validatePackRemovePath`)で止まるので、ここには来ない(G2)
  - `pack_remove.go:283`: manifest の書き込みの失敗で、残したファイルもあるときの文(G3)
  - `pack_remove.go:384-393`: 空のディレクトリを消せなかったときの警告と、その表示に使う `packRemoveDisplayPath`(G4)
  - `pack_remove.go:191`、`:195`、`:245`(Lstat と ReadFile の I/O エラー)、`:214`(`CleanLocalRelPath` の失敗)、`:264`(symlink でもディレクトリでもない特殊ファイル)、`:290`(`filesWord` の単数)、`:309`(祖先を Lstat できない)、`:367`(pack のディレクトリがディレクトリでない)、`:95`(`filepath.Abs` の失敗)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `Manifest.Write` のすべての呼び手(init `init.go:473`、pack add `pack.go:115`、upgrade `upgrade_v2.go:704`、eject `eject.go:102`、adopt `adopt.go:174`、migrate `migrate.go:958`、pack remove `pack_remove.go:56`) | pass | 全パッケージの run が ok。filter つきの cli の run の coverage profile で、7 か所の呼び出しの行がどれも 1 回以上通っていることを確かめた。名前ごとの top-level のテストの数は Init 44、Upgrade 39、Eject 7、Adopt 14、Migrat 49、Pack 59(pack remove の 22 を含む)、Doctor 54、Scaffold 48(重なりあり)で、すべて pass |
| 書き込みの成功のあと `.ralph/` に一時ファイルが残らないこと | pass | `TestManifestWrite_FreshPathRoundTripsWithMode0644AndNoTempFile` と `..._OverwriteReplacesExistingContent` がディレクトリに manifest だけがあることを見る。木全体の hash を前後で比べる `snapshotTreeHashesExcluding` を使う既存のテスト(`upgrade_v2_test.go`、`migrate_test.go`、`eject_adopt_test.go`)も通った。呼び手の成功のあとに `.ralph/` の中身を数える既存のテストはない(`ReadDir` で `.ralph` を読むテストを grep して 0 件) |
| 旧い `TestManifestWriteCreatesDir` の意図 | pass | `TestManifestWrite_NestedPathRoundTrips` へ名前を変えただけで、本体は base と同じ(base の名前はディレクトリを作ると読めたが、テストは自分で `MkdirAll` していた)。弱めてはいない |
| legacy の manifest の断り(init、pack add、eject、adopt の番兵) | pass | 既存の legacy のテストが全パッケージの run で通った。pack remove は `TestRemovePack_LegacyManifest_FailsClosed` |
| `ralph init --help` の文 | pass(壊れていない) | `internal/cli` の全テストが通った。ただし init の `Long` の「added later with ralph pack add」の文を押さえるテストはなく(grep で 0 件)、remove を足した 1 文もテストでは見ていない |

## Test gaps

- G1(MEDIUM): migration が残した旧い rule の場所 `.claude/rules/<lang>.md` の項を外すテストがない。計画の Scope に名前の出ている対象で、`isPackRemoveKey` からこの場所を外す変異 M1 が残った。M1 のままだと、migration を経た repo では、そのファイルと項が残り、`Meta.Packs` から pack が消えたあとも manifest に項が残る(AC-1 の「それらの manifest の項がなくなる」に反する)。doctor --strict は pass するので doctor では気づけない。閉じるテストは上の `TestProbe_RemovePack_LegacyRulePathEntryIsRemoved` をそのまま `pack_remove_test.go` に入れる形で、HEAD で通り M1 で落ちることは確かめてある。PR の前に足すことを勧める
- G2(LOW): 計画の検査を通ったあと `ApplyOps` が失敗する経路(`pack_remove.go:120`)のテストがない。エラーの文と「manifest は変えていない」の主張を押さえるものがない。pack のディレクトリを 0555 にすれば、`os.Remove` が EACCES で失敗するので届く見込み(未確認)
- G3(LOW): manifest の書き込みが失敗し、残したファイルもあるときの文(`packRemoveDoneFiles` の kept の分)のテストがない
- G4(LOW): 空になったディレクトリを消せなかったときの警告(`removeDirIfEmpty` の失敗、`packRemoveDisplayPath`)のテストがない
- G5(LOW): `Manifest.Write` の文書にある 2 つの挙動(symlink を通常のファイルに置き換える、既存のファイルの mode を 0644 にする)を押さえるテストがない。使い捨てのテストで HEAD が文書どおりに動くことは確かめた。release ノートの候補がこの挙動を書くので、テストに入れておくと文と挙動のずれを防げる
- G6(INFO): `os.WriteFile` は新しく作るファイルの mode に umask をかけるが、`Chmod(0o644)` は umask の影響を受けない。umask 077 の利用者では、前は 0600 で作られた manifest が 0644 になる。Go の仕様から読んだことで、実測はしていない。計画の「挙動が変わるのは symlink と mode」の範囲の話なので、release ノートの候補に一言足すかは A の判断
- G7(INFO): `Manifest.Write` の途中の段(`tmp.Write`、`Chmod`、`Sync`、`Close`)の失敗を起こすテストはない。後始末は rename の失敗と同じ分岐を通る(M2 で押さえた)
- G8(INFO): AC-2 は fixture の gen1 から gen2 への upgrade で確かめた。本物の binary の次の版での確かめはない(verify report にも記録済み)

## Verdict

- Pass: 56/56(Go の 11 パッケージと shell の 45 本。`==> All verifiers passed.`、rc 0)。pack remove の 22 本と `Manifest.Write` の 5 本を含む
- Fail: 0
- Blocked: なし
