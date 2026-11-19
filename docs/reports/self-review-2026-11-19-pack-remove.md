# Self-review report: pack-remove

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-pack-remove.md`
- Branch: `feat/pack-remove`、HEAD `fac9a1cf`、base は `main` の `a8c5e289`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff a8c5e289...HEAD` の 9 ファイル(+1598 −4)の diff の質だけ。コードは slice 1 `c593c00f`(`internal/scaffold/manifest.go`)と slice 2 `39bc44b4`(`internal/cli/pack_remove.go`、`pack.go`、`init.go`)。文書は `33cdace4` と `8cbe2ea2`。計画書の commit は Progress が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify` と `/test` の担当)

## Evidence reviewed

- コード: `internal/cli/pack_remove.go` の全文(372 行)、`internal/scaffold/manifest.go` の `Write`、`internal/cli/pack.go` の `addPack`(比較のため)
- 呼び出し先: `upgrade.ApplyOps` と `ValidateRealParentChain`(`internal/upgrade/replaceplan.go:509-660`)、`scaffold.CleanLocalRelPath`(`internal/scaffold/paths.go`)、`packPrefixFor` / `packRuleRelPath` / `legacyPackRuleRelPath` / `packRelDir`(`internal/cli/language_pack.go:88-120`)、upgrade の `classifyPath` の recorded hash の読み方(`replaceplan.go:297-300`)、`errLegacyLayoutFailClosed`(`internal/cli/upgrade.go:82-96`)、atomic write の手本 `writeJSONFileAtomic`(`internal/org/watchalive.go:306`)
- manifest の書き手: `git grep` で `Manifest.Write` の呼び手 7 か所(adopt、init、pack add、eject、upgrade_v2、migrate、pack remove)を数えた。`manifest.toml` を `os.WriteFile` などで直接書く所はない
- テスト(読んだだけで、走らせていない): `internal/cli/pack_remove_test.go`(786 行)、`internal/scaffold/manifest_test.go` の新しい 4 つと既存の `TestManifestWriteCreatesDir`。`internal/cli` に `t.Parallel()` はなく、package 変数の差し替え(`writePackRemoveManifest`、`renameManifestFile`)は `t.Cleanup` で戻している
- probe(`$TMPDIR` の使い捨てのディレクトリ、macOS APFS): `packs` を外のディレクトリへの symlink にすると、`stat -f %HT`(lstat)は `packs` を Symbolic Link、`packs/languages` と `packs/languages/golang` を Directory と答え、`rmdir packs/languages/golang/empty` は外のディレクトリを消した(L1 の根拠)
- tech-debt 台帳: `internal/cli/pack.go`、`init.go`、`internal/scaffold/manifest.go` を名指す行を grep した。429 行(`runInitInteractive` の form のテスト)の trigger は「質問か group を足す変更」で、この diff は `Long` の 1 文だけなので当たらない。この PR で閉じる行、無効になる行はない

## 依頼された観点の確認

| 観点 | 結果 | 根拠 |
| --- | --- | --- |
| pack 名による traversal | 防げている | `validatePackRemoveName` が manifest を読む前に空、`.`、`..`、`/` と `\` を含む名前を断る(`pack_remove.go:136-141`)。そのうえで `Meta.Packs` に名前があることを求める |
| manifest の key による traversal | 防げている | `packRemoveKeys` は生の key で pack の 3 か所に当たるものを集め、`validatePackRemovePath` が `CleanLocalRelPath` で整えたあと、もう一度 3 か所の中にあるかを確かめる(`:207-220`)。`packs/languages/golang/../../AGENTS.md` は整えると `AGENTS.md` になり、何も消す前に断られる。`packPrefixFor` は末尾に `/` を付けるので、`go` と `golang` の前方一致も起きない |
| 葉の symlink | 消さない | `os.Lstat` で通常のファイルでないものは残す(`:181`、`:227-229`)。`ApplyOps` も葉を Lstat し直す(`replaceplan.go:522-534`) |
| 親の連なりの symlink | 消すものについては防げている。空のディレクトリの片付けに穴が 1 つ(L1) | 消す候補の key すべてで `ValidateRealParentChain` を先に通し(`:216`)、1 つでも symlink の親があれば何も消さずに断る。片付けの `removeEmptyPackDirs` は、`packs/languages/<lang>/` の下に key が 1 つもないとき、祖先の symlink を確かめずに辿る |
| 途中で止まったときの順序 | 正しい | ファイルを消す、空のディレクトリを片付ける、manifest を書く、の順(`:115-126`)。`ApplyOps` が途中で止まれば manifest は書かず、打ち直しで消えたものは `missing` として項だけ外す。manifest の書き込みは一時ファイルと rename になり、失敗しても元の manifest が残る(`manifest.go` の `Write`)。打ち直しで終わる |
| 利用者のファイルを消す経路 | 見つからない | 消すのは owner=core の項で、通常のファイルで、中身の hash が `DiskHash`(空なら `Hash`)と一致するものだけ(`:226-251`)。中身が ralph の書いたものと同じなので、失うものはない。manifest にないファイルは対象に入らない。消えるディレクトリは中身が空のものだけ(L4 で dry run に出ないことを指摘) |

## Findings

指摘はすべて LOW。CRITICAL、HIGH、MEDIUM はない。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | security | L1: `removeEmptyPackDirs` は `packs` と `packs/languages` が symlink かを確かめない。`isRealDir` の `os.Lstat` は最後の要素しか symlink と見ないので、`packs` が外のディレクトリへの symlink なら `packs/languages` も `packs/languages/<lang>` も「本物のディレクトリ」に見え、`WalkDir` は外のディレクトリを辿って空のディレクトリを消し、外の `languages` も空なら消す。消すファイルの経路では、`packs/languages/<lang>/` の下の key すべてが `ValidateRealParentChain` を通るので、この形は断られる。届くのは、`Meta.Packs` に pack があり、その下に key が 1 つもない manifest(手で直した manifest、または rule の項だけが残った manifest)のときだけ。消えるのは空のディレクトリだけで、ファイルは消えない。doc comment は「when each is a real directory」「a symlinked packs/languages/<lang> is not walked at all」と書き、symlink を辿らない意図を示している | `internal/cli/pack_remove.go:274-317`(`isRealDir` は `:319-322`)。probe: `packs` → 外、で `stat -f %HT packs/languages` が Directory、`rmdir packs/languages/golang/empty` が外を消した | 歩き出す前に `upgrade.ValidateRealParentChain(absDir, packPrefixFor(lang)+"x")` を呼ぶか、`packs` と `packs/languages` を順に `isRealDir` で確かめ、だめなら片付けを飛ばして理由を 1 行出す。doc comment に「祖先も本物のディレクトリであること」を足す |
| LOW | exception-handling | L2: 計画の段で断るときの文が、消すコマンドの文になっていない。親の連なりに symlink があると、`ValidateRealParentChain` の文がそのまま出る: `packs/languages/golang/README.md: refusing to write through non-directory path component /…/packs/languages/golang (mode L…)`。消すコマンドなのに「write」と言い、何も変えていないことも、次に何をすればよいかも言わない。`ApplyOps` の段の失敗(`:116`)は「.ralph/manifest.toml was not changed」と言うので、2 つの断り方の文が揃っていない。また `Long` と README は「a symlink … is kept」と書くが、親のディレクトリ(`packs/languages/<lang>`、`.claude/rules/ralph`)が symlink のときは残すのではなく、コマンド全体が止まる | `internal/cli/pack_remove.go:106-109`、`:202-220`。文は `internal/upgrade/replaceplan.go:631`。`Long` は `pack_remove.go:32-35` | `planPackRemove` の err を `removePack` で包み、`pack %s: nothing was changed: %w` のように何も変えていないことを言う。`Long` に「a parent directory that is a symlink stops the command」を 1 文足す |
| LOW | readability | L3: manifest の書き込みに失敗したときの文が「the files of pack %s were removed」と言う。残したファイル(変更したもの、eject したもの)があるときや、消したものが 0 のときも同じ文になる。この経路では `printPackRemoveSummary` を呼ばないので、どれを残したかも出ない。変更した `verify.sh` を残した利用者は、この文を読んで消えたと受け取りうる。打ち直すと Kept の一覧が出るので、害は一時の誤解にとどまる | `internal/cli/pack_remove.go:124-126` | 「the unchanged files of pack %s were deleted (changed and ejected files were kept)」のように、消したものだけを言う。数(`len(plan.deletes)`)を添えてもよい |
| LOW | readability | L4: dry run は消えるディレクトリを出さない。本番は `packs/languages/<lang>/` の下の空のディレクトリ、`packs/languages/`、`packs/` を消しうるが、dry run の出力は delete / keep / untrack の行と manifest の 1 文だけ。本番の summary も「left in place」の理由だけで、消したディレクトリは出ない。また `Long` は「the directories that become empty」と言うが、もとから空だった利用者のディレクトリ(テストの `sub/empty`)も消える。dry run の締めの文は、残すファイルが 0 件でも「so ralph no longer tracks the kept files」と言う | `internal/cli/pack_remove.go:345-358`(dry run)、`:274-317`(片付け)、`:25-30`(`Long`)。テストの comment `pack_remove_test.go` の `TestRemovePack_UserFileInPackDir_KeepsTheDirectory` | dry run に「and the directories under packs/languages/<lang>/ that are empty afterwards (packs/ too when nothing else is in it)」の 1 行を足すか、`Long` の「become empty」を「are empty afterwards」にする。締めの文の「kept files」は `len(plan.kept) > 0` のときだけ付ける |
| LOW | maintainability | L5: `legacyLayoutFailClosedMsg` と `errLegacyLayoutFailClosed` の doc comment が、返す側を `ralph init` と `ralph pack add`(init.go、pack.go)だけと書く。この PR で `ralph pack remove`(`pack_remove.go:100`)が 3 つめの返し手になった(`eject.go:124` も前から返しているが、doc にない)。文の中身(「no files were changed」)は remove にも合う | `internal/cli/upgrade.go:82-96` | 2 つの doc comment に pack remove(と eject)を足すか、「every v2-only verb outside ralph upgrade (init, pack add, pack remove, eject)」のようにまとめる |
| LOW | naming | L6: 既存のテスト `TestManifestWriteCreatesDir` は、名前に反して自分で `os.MkdirAll` してから `Write` を呼ぶ。この PR は `Write` の doc に「The directory of path must exist; Write does not create it」と書き、そのすぐ下に逆のことを確かめる `TestManifestWrite_MissingParentDirFailsAndCreatesNothing` を足したので、隣り合う 2 つの名前が矛盾して読める。名前は PR の前からのもの | `internal/scaffold/manifest_test.go:379-398`、`:589` | `TestManifestWrite_NestedPathRoundTrips` などに改める |

## 読んだが指摘にしないもの

- 計画の段の hash の読みと `ApplyOps` の削除のあいだにファイルが変えられると、変えたものが消える。窓はミリ秒で、1 人で打つ CLI なので上げない
- 手で直した `Meta.Packs` に core の rule の名前(例 `architecture`)を入れると、`ralph pack remove architecture` は hash の合う `.claude/rules/ralph/architecture.md` を消して項を外す。名前の検査が想定する「改ざんされた manifest」と同じ経路だが、利用者が自分で入れたもので、`ralph upgrade` が core のファイルとして作り直すので上げない
- `Manifest.Write` は `tmp.Chmod(0o644)` で umask を無視する。前の `os.WriteFile(path, data, 0644)` は新しいファイルで umask に従った(umask 077 なら 0600)。計画は挙動の変化を「symlink」と「既存の mode が 0644 でないとき」の 2 つとするが、umask もその 1 つ。manifest は git に入るファイルで、git は実行ビットしか持たないので害はない。`writeJSONFileAtomic` と同じ形
- `Write` の途中で process が殺されると、`.ralph/.manifest.toml.*.tmp` が残り、git status に untracked として出る。まれで、元の manifest は無事なので上げない
- `Write` の err は `write <path>: open <tmp>: …` のように path が 2 度出る。読めるので上げない
- `packRemoveKeepReason` の `case scaffold.OwnerCore:` は空の case で、switch を抜けて hash の比較に進む。Go では読める形
- 記録の hash が `DiskHash` も `Hash` も空のとき、理由は「modified since ralph wrote it」になる。v2 の項は必ず `Hash` を持つので上げない
- `initV2Project` と `upgradeToGen2` は `scaffold.EmbeddedFS` と `Version` を戻さない。既存のテストの書き方に合わせたもので、`internal/cli` は並列に走らない

## Positive notes

- 名前の検査を manifest の読み込みより前に置き、key を整えたあとにもう一度 pack の 3 か所に入るかを確かめている。`Meta.Packs` の改ざんと key の改ざんの両方を、何も書く前に断る
- 消す判断を manifest と disk だけで行い、template を読まない。そのため、今の binary にない pack(retired)も外せる
- 削除を `upgrade.ApplyOps` に任せ、自前で `os.Remove` を書いていない。dry run も同じ検査(`CleanLocalRelPath`、`ValidateRealParentChain`、葉の Lstat)を先に通すので、dry run が約束した削除を本番が断ることはない
- `Manifest.Write` を呼び手 7 か所に効く形で atomic にし、失敗の各段で一時ファイルを消している。手本(`writeJSONFileAtomic`)と同じ形に `Sync` を足した
- テストは、symlink の pack のディレクトリ、外へ出る key、retired の pack、upgrade のあとの `DiskHash`、manifest の書き込みの失敗と打ち直しを、ファイルの中身と manifest のバイト列で確かめている。失敗の注入は package 変数で、root で走る CI でも効く(`chmod` を使っていない)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| — | — | — | — | — |

台帳に足した行はない。L1〜L6 はどれも数行で直せる指摘で、先送りにすると決めたものではない。`/pr` までに直さないものがあれば、その段で台帳に足す。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 6 件)
- Follow-ups:
  - L1 と L2 は、消すコマンドの安全と断りの文の話なので、直すなら先にこの 2 つ。L1 は 1 行の検査で閉じる
  - pack が書く場所(`packs/languages/<lang>/` と rule の 2 か所、legacy の rule を入れて 3 か所)が、`renderPackInto`(書く側)、upgrade の `preserve`(`upgrade_v2.go:413`、前方一致)、`isPackRemoveKey`(`pack_remove.go:155`、rule は完全一致)の 3 つに別々に書かれている。pack が新しい場所に書くようになったら 3 つを揃えて直す必要がある。今の差は害がないので指摘にはしない
  - `internal/cli/pack_remove_test.go` は 786 行。次にテストを足すと 800 行を超えるので、そのときはファイルを分ける
