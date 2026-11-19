# Walkthrough: `ralph pack remove` で入れた言語 pack を外し、manifest の書き込みを原子的にする(pack-remove)

- 計画: `docs/plans/archive/2026-11-19-pack-remove.md`(PR の作成のあとに archive へ移す)
- issue: #344。PR: #346
- 差分: `git diff origin/main...HEAD`(この walkthrough を足す前で 19 ファイル、+2,389 / −15、base `a8c5e289`)。Go の本体は 4 ファイルで約 +470(新しい `pack_remove.go` が 434 行)、Go のテストは 3 ファイルで約 +1,170、README と recipe と tech-debt は約 +50、残りは report・計画・insight
- パイプライン: 1 cycle。cross-review(codex)の指摘は 0 件

## 1. `removePack`(`internal/cli/pack_remove.go:89`。読む価値が一番高い)

- 順番は、名前の検査(`:140`、manifest を読む前)→ manifest を読み、v2 か(legacy は番兵 `errLegacyLayoutFailClosed` を包まずに返す)、`Meta.Packs` に pack があるか(`:147`)→ 全部の項を分ける(`planPackRemove`、`:178`)→ dry run ならここで出して終わる → 消す(`applyPackRemoveDeletes`、`:270`、`upgrade.ApplyOps` の `OpDelete`)→ 空のディレクトリを片付ける(`:329`)→ manifest の項と `Meta.Packs` を外して書く(`writePackRemoveManifest`、`:55`)
- 分けるのは書く前に 1 度だけ(`:178`)。項ごとに、整えたパスが pack の 3 か所の中にあるか(`validatePackRemovePath`、`:211`。`packs/languages/golang/../../AGENTS.md` のような key を断る)、親の連なりが本物のディレクトリか(`upgrade.ValidateRealParentChain`)を確かめる。1 つでもだめなら「nothing was changed」の文で何も書かずに返す
- 消すか残すかは `packRemoveKeepReason`(`:230`)。通常のファイルでない、owner が core でない(fork、seed、記録なし)、hash が記録(`DiskHash`、空なら `Hash`。upgrade の `classifyPath` と同じ読み方)と違う、のどれかなら残す。disk にない項は外すだけ
- 消してから manifest を書く順は upgrade の commit barrier と同じ。書き込みが失敗しても manifest はまだ pack を入れたままなので、打ち直すと消したファイルが「already missing」になって終わる(`TestRemovePack_ManifestWriteFails_RerunFinishes`)

## 2. 空のディレクトリの片付け(`:302`、`:329`)

- `packDirAncestorProblem`(`:302`)が先に `packs` と `packs/languages` を Lstat する。どちらかが symlink なら何も消さずに 1 行出す。`os.Lstat` はパスの最後の要素しか symlink と見ないので、`packs` が外への symlink だと `packs/languages/<lang>` が本物に見え、外の空のディレクトリを消してしまうため(self-review の L1)
- `packs/languages/<lang>` の下を `filepath.WalkDir` で集め、深い順に、空のものだけを消す。残すと決めたパスがディレクトリなら、空でも消さない。最後に `packs/languages` と `packs` を空なら消す

## 3. `scaffold.Manifest.Write`(`internal/scaffold/manifest.go:118`)

- 同じディレクトリに `.manifest.toml.*.tmp` を作り、書く、`Chmod(0o644)`、`Sync`、`Close`、`renameManifestFile`(`:109`、テストで差し替える `os.Rename`)の順。どこかで失敗したら一時ファイルを消し、元のファイルは変えない
- 呼び手(init、pack add、upgrade、eject、adopt、migrate、pack remove)は変えていない。書く TOML も同じ。変わるのは、manifest が symlink なら通常のファイルに置き換わること(`TestManifestWrite_SymlinkAtPathIsReplacedNotFollowed`、`manifest_test.go:592`)と、mode が必ず 0644 になること
- Codex の計画助言の [HIGH] による。remove だけに `WriteAtomic` を足す案は、upgrade の commit barrier に同じ穴が残るので採らなかった(計画の Design decisions)

## 4. 出力と文書

- dry run(`printPackRemoveDryRun`、`:399`)は delete / keep / untrack の行、片付けるディレクトリの 1 行、manifest の 1 行、`Nothing was changed.` を出す
- 終わりの出力(`printPackRemoveSummary`、`:423`)は件数の 1 行のあと、残したものがあれば理由つきの一覧と、`ralph pack add` を打ち直す前に消すか動かすように、という 1 文を出す。add は残したファイルを上書きし、残した symlink はリンク先に書き、残したディレクトリでは止まる(verify の V1。add の挙動は tech-debt の新しい行)
- README の Commands の表と Language packs の節、`templates/base/docs/recipes/adding-a-language-pack.md` の「Removing a pack」、`ralph init --help` の 1 文、`doctor.go` と `upgrade.go` のコメント

## 5. テスト

- `internal/cli/pack_remove_test.go`(622 行): AC-1(外したあと doctor の scaffold の 5 検査が strict で pass、`Pack: golang` が出ない)、AC-2(次の世代の upgrade で戻らない)、AC-3(変更と eject を残す)、upgrade のあと(`DiskHash` が入った項)、入っていない pack、不正な名前、legacy、manifest なし、dry run、ほかの pack、手で消したあと、書き込みの失敗と打ち直し、retired の pack、cobra の配線
- `internal/cli/pack_remove_paths_test.go`(320 行ほど): symlink とディレクトリの葉、利用者が足したファイル、symlink の pack ディレクトリ、symlink の `packs` と `packs/languages`、pack の外を指す key、旧い rule の場所(`.claude/rules/<lang>.md`)の変更なしと変更あり
- `internal/scaffold/manifest_test.go`: 新しいパス、上書き、rename の失敗、symlink の置き換え、親ディレクトリなし
- 変異: implementer が 11 個(slice 2)と 2 個(slice 1)、tester が 4 個を当てた。tester の M1(旧い rule の場所の判定を外す)だけが残り、4426eb3f のテスト 2 本で落ちるようにした。`run-test.sh` は 039e0d7f で rc 0(56/56)。4426eb3f のテストは絞った `go test` で pass

## 見どころ

- 変えたファイルの manifest の項も外す。残すと `classifyCore` が template-absent かつ hash 違いで `plan.Drift` に入れ(`internal/upgrade/replaceplan.go:369`)、`doctor --strict` の core file hashes が落ちる。外したファイルは利用者のもので、doctor も upgrade も触らない
- 消す判断は manifest と disk だけで行い、埋め込みの template を読まない。だから今の binary にない pack も外せる(upgrade はそうした pack を preserve し続けるので、外す手段はこのコマンドだけ)
- 開いている穴: (1) `ralph pack add` が pack のパスの symlink を通して書く(tech-debt の新しい行。add の挙動は計画の Non-goals)。(2) `Manifest.Write` が途中で kill されたときに一時ファイルが残りうる。(3) 下流の本物の repo では打っていない(scratch の repo と作った binary で確かめた)
