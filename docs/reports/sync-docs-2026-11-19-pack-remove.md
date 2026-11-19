# sync-docs report: pack-remove

- Date: 2026-11-19
- Plan: `docs/plans/active/2026-11-19-pack-remove.md`(承認の digest `88ea9bb6774f`。`./scripts/plan-visual.sh digest` で計算し直した値と今も一致する)。issue #344
- Branch: `feat/pack-remove`。base は origin/main `a8c5e289`、run 開始時の HEAD は `8dc63dcf`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は 2 回(1 回目 `f14a6e89`、2 回目 `16e79b18`。指摘 0、Merge yes)。verify は `a12d256e` で partial pass(V1 は `039e0d7f` で直し済み)。test は `731ce7d9` で pass(G1 は `4426eb3f` で埋め済み)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff a8c5e289...HEAD`、verify の V1・V2・I1 から I5、test の G1 から G8。tech-debt の全行の Trigger を自分で走査した(下の「Trigger の走査」)
- 差分の大きさ(`/pr` が walkthrough を決める材料。この commit が足す前の HEAD `8dc63dcf` の値): `git diff --shortstat a8c5e289...HEAD` は 16 ファイル、+2,220/-14。内訳は docs(計画、report 3 本、insights)が 5 ファイル +492/-0、テストの Go が 3 ファイル +1,198/-1、残りの 8 ファイル(`pack_remove.go` 435 行、`manifest.go`、`doctor.go`、`init.go`、`pack.go`、`upgrade.go`、README、template の recipe)が +530/-13。この commit は docs だけを足す
- 制約: Go のコードは触っていない。`./scripts/run-test.sh` は回していない(依頼どおり)。`scripts/check-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、`check-pipeline-sync.sh` は回した(下の「確認」)

## Summary

verify の V2(LOW)を直した。tech-debt の「800 行を超えた 5 つのファイル」の行(415 行目)に、`doctor.go` の comment だけの変更を持ち越す note を足した。verify の V1 は `039e0d7f` で直っていて、README、recipe、help、実行時の出力の 4 か所を読み直して今の挙動(残した通常のファイルは上書き、symlink はその先に書く、ディレクトリでは失敗)と合っていることを確かめた。

`ralph pack add` が pack のパスの symlink を通して書く穴を、tech-debt に新しい行として足した(527 行目)。verify が作った binary で見つけたもので、この計画の Non-goals(`pack add` の挙動は変えない)の外にある。

I1 から I5 は、どれも文書を直さない。理由は下の表に書いた。計画の AC の箱は、verify と test の report が満たしたとする 11 件を `[x]` にした(digest は変わらない)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 415 行目の Trigger のセルの末尾に 2026-11-19 の持ち越しの note、Related のセルの末尾に計画を足した。527 行目に新しい行を 1 つ足した。ほかの行は変わっていない。下の「V2」「新しい行」 |
| `docs/plans/active/2026-11-19-pack-remove.md` | Acceptance criteria の AC-1 から AC-11 を `[x]` にした。Progress の「Implementation started」の下に sync-docs の 1 行を足した。計画の本文は変えていない(digest は `88ea9bb6774f` のまま) |
| `docs/insights/events/2026-11-19-pack-remove.jsonl` | `sync_docs` の event を 1 行足した(`--cycle auto` で cycle 1) |
| `docs/reports/sync-docs-2026-11-19-pack-remove.md` | この report |

## verify が挙げた drift

### V2(LOW): tech-debt 415 行目の Trigger

直した。「The next change to any of the five」の 5 つのうち、`internal/cli/doctor.go` だけを `4e98b216` が変えた(`checkInstalledPacks` の中の `Meta.Packs` を読む所の comment に `ralph pack remove` を足した)。行数は origin/main `a8c5e289` で 1648、`4e98b216` で 1649(`git show <sha>:internal/cli/doctor.go | wc -l`)。この branch で `doctor.go` を変えた commit は `4e98b216` だけで、HEAD も 1649。残る 4 つ(`internal/config/config.go` 1190、`tests/test-detect-changed-languages.sh` 848、`tests/test-plan-visual.sh` 1012、`tests/test-secret-scan-branch.sh` 2217)は base と HEAD で同じ行数で、触っていない。

note は、ほかの branch と同じ形(`2026-11-13: feat/gate-tool-versions (#311) hit this trigger with …, but did not pay it: … Carried over to the next change that meets this trigger`)にした。払わなかった理由は 2 つ書いた。comment だけの変更であること、move-only の分割はそれと混ざること。`4e98b216` は計画の Scope に `doctor.go` がなく、2 回目の self-review の Follow-ups(`doctor.go:773` の「Both」)を inline で直したものなので、そのことも書いた。Related に `docs/plans/active/2026-11-19-pack-remove.md`(`archive-plan.sh` が `/pr` で `archive/` に書き換える。先には書かない)を足した。

### V1(MEDIUM): 直し済み、読み直した

`039e0d7f` が、残したものを `ralph pack add` が「上書きする」と一括で言っていた 4 か所(`pack_remove.go` の help と実行時の出力、README、template の recipe)を、種類ごとの挙動に直している。この run で `README.md:333`、`templates/base/docs/recipes/adding-a-language-pack.md:60-67`、`pack_remove.go:36-38` と `:430` を読み、どれも「残したファイルは上書きする、残した symlink はその先に書く、残したディレクトリでは失敗する」と同じ文になっていることを確かめた。挙動は `renderMappedFile` と `scaffold.RenderFS` を読んだ内容と合う。文を押さえるテストは `TestRemovePack_KeepsModifiedAndEjectedFiles`(help と出力の文)。挙動そのものを押さえるテストはなく、下の新しい行がそれを持つ。

## 新しい行(527 行目)

`ralph pack add` は `renderPackInto` に overwrite を `true` で渡す(`addPack`)。pack の中のファイルを書く `scaffold.RenderFS` と rule を書く `renderMappedFile` は、どちらも `Lstat` で在るかを見て上書きと数え、`os.WriteFile` を呼ぶ。パスが repo の中かどうかの検査(`isInsideDir`、`RenderFS` の prefix の検査)は文字列の比べで、リンクを解かない。だから pack のパスの symlink は、`ralph pack remove` が残したものも、add がその先に書く(repo の外のファイルでもよい)。ディレクトリでは add が `is a directory` で止まる。verify が作った binary で両方を見た(verify report の V1 の確かめ方)。

- 重さは LOW とした。repo の外に書くのは、利用者が作った symlink(または `ralph pack remove` が残したもの)に、同じ言語の `ralph pack add` を打ったときだけで、書く中身は pack 自身のファイル。もう一方は error で止まるだけ
- 払わなかった理由: 計画の Non-goals が `ralph pack add` の挙動を変えないとしている。この PR は `ralph pack remove` が言うことだけを直した(`039e0d7f`)。直すと `ralph init` も呼ぶ `scaffold.RenderFS` と `renderMappedFile` が変わる
- Trigger: `ralph pack add`、`renderPackInto`、`renderMappedFile`、`scaffold.RenderFS` の overwrite の枝のどれかの次の変更、または repo の外のファイルが `pack add` で上書きされたという報告。直し方の候補(symlink は断る、または `Manifest.Write` のようにリンク自体を置き換える。ディレクトリは何も書く前に断る)と、それを押さえるテスト 2 本、`pack remove` の 4 か所の文を一緒に変えることを書いた
- `scaffold.RenderFS` の comment が、symlink をたどるのを `--force` の「explicit user consent」と言う点にも触れた。`pack add` は flag なしで overwrite を渡すので、その前提が当てはまらない

既存の行に同じ内容はない(`renderMappedFile`、`RenderFS`、「write through」、`pack add` で tech-debt 全行を探して確かめた。43 行目は `~~` で閉じた別の件)。

## I1 から I5 の判断

| ID | 判断 | 理由 |
|----|------|------|
| I1(117 行目の Trigger。template の recipe に節が足された) | 当たっていないと読む。note は足さない | Trigger は「template-content の restructure」。この PR は template の recipe に「Removing a pack」を 1 節足しただけで、既存の節を組み替えていない。それでも `KNOWN_DIFFS` を読み直した。root の recipe は ralph の repo の貢献者向け(pack を作る手順)、template の recipe は `packs/languages/` などを持たない下流向けで、足した節は下流向けにしか要らない。差は縮まらず広がったので、`KNOWN_DIFF` は今も要る。計画の Scope にも同じ判断がある。「当たる」と読み直すなら、note は 1 文(落としてよい) |
| I2(計画の AC の箱) | AC-1 から AC-11 を `[x]` にした | verify は AC-1 から AC-11 のどれも満たさないとしておらず、test の report が全部のテストの pass と対応を示している。AC-9 は verify が V1 の文を付けて Met としたもので、`039e0d7f` で直ったことを上で読み直した。「Review artifact created」は計画で既に `[x]`。箱は digest に入らない(`plan-visual.sh digest` が `- [x]` を `- [ ]` として数える。計算し直して `88ea9bb6774f` のまま) |
| I3(`doctor.go:1239` と `eject.go:112` の comment が v2 の壁を持つ動詞を挙げ、pack remove を挙げない) | 直さない | 例を挙げた文で、今も誤りではない(verify と self-review が同じ判断)。`doctor.go` を足して編集すると、415 行目に足した note の行数(1649)と「the only commit」が崩れ、post-implementation の run のあとに Go を触ることになる。この PR で直した `upgrade.go` の doc comment が 5 つの動詞(init、pack add、pack remove、eject、adopt)を全部挙げているので、読み手はそこから辿れる |
| I4(残す理由に owner が seed か空の項もある) | 直さない | 実行時の出力が理由を名指す(`packRemoveKeepReason`: `no recorded owner (only owner=core files are deleted)`、`owner=seed (only owner=core files are deleted)`)。help と文書の「changed、ejected、通常のファイルでない」は、init と add が作る項(owner=core)で ralph 自身が起こす場合を言い尽くしている。seed や空になるのは manifest を手で書き換えたときだけ |
| I5(古い仕様 `docs/specs/2026-04-16-ralph-cli-tool.md:295` が、release で消えた pack は upgrade が `Meta.Packs` から外すと書く) | 直さない | その節は v1 の upgrade engine(`ActionConflict`、`Managed=false` など)の仕様で、overlay-scaffold-v2 が置き換えた。日付つきの歴史的な仕様で、sync-docs の skill はこれを直す対象にしていない。overlay-scaffold-v2 の仕様には retired pack の扱いの文がなく、今の挙動は `upgrade_v2.go` の `preserve` の comment と、この計画の Design decisions にある。この diff より前からのずれで、新しい行も足さない |

## test report の穴

| 穴 | 扱い |
|---|---|
| G1(MEDIUM) | `4426eb3f` で閉じている(旧い rule の場所の項のテスト 2 本) |
| G5(LOW、`Manifest.Write` が symlink を置き換える、mode を 0644 にする) | symlink を置き換える方は `4426eb3f` のテストが押さえた。mode の方は `TestManifestWrite_FreshPathRoundTripsWithMode0644AndNoTempFile` が新しいファイルで 0644 を押さえるが、0600 の既存のファイルを 0644 にする挙動のテストは見つからない。新しい行は足さない |
| G2、G3、G4(LOW。`ApplyOps` の失敗の経路、manifest の書き込みの失敗で残したファイルもあるときの文、空のディレクトリを消せなかったときの警告) | tech-debt の行には足していない(依頼が足す行を 1 つにしていた)。test report の Test gaps に残っている。足すなら 1 行にまとめられる |
| G6、G7、G8(INFO) | 記録だけ。G6(umask 077 の利用者の manifest が 0644 になる)は release ノートの候補を持つ A が判断する。G7、G8 は test report のとおり |

## Trigger の走査

tech-debt の全行から、Trigger が全部 `~~` の行を除き、Trigger のセルと Debt のセルを、この PR が変えた物の語(`manifest.go`、`Manifest.Write`、`pack.go`、`init.go`、`upgrade.go`、`doctor.go`、`language_pack`、`renderPackInto`、`renderMappedFile`、`RenderFS`、`README.md`、`adding-a-language-pack`、`pack add`、`Meta.Packs`、`check-sync`、`KNOWN_DIFF`、`manifest.toml`、`recipes/`、`errLegacyLayoutFailClosed` など)で探し、候補の Trigger を読んだ。この PR が満たした Trigger は 415 行目だけだった。

| 行(先頭の語) | 判断 | 理由 |
|---|---|---|
| 415「Five files went past the 800-line guide」 | 当たった。持ち越した | 上の V2 |
| 117「`scripts/check-sync.sh`'s `KNOWN_DIFFS`」 | 当たっていない | 上の I1 |
| 371「The recipes under `templates/base/docs/recipes/`」 | 当たっていない | 「these five recipes」は `isolated-herdr-server.md`、`codex-seat-permissions.md`、`codex-setup.md`、`director-wake.md`、`director-launch-cmd.md`。`adding-a-language-pack.md` は入っていない。足した節は `docs/evidence/` を引かない(`check-template-purity.sh` も通る) |
| 99「5 unfixed cycle-1 LOW findings from overlay-scaffold-v2」 | 当たっていない | Trigger は列挙された call site(`classifyFork`、`WriteUpgradeReport`、`writeFileV2`、`writeMigrationFile` など)の次の変更。`upgrade.go` は doc comment だけで、どれも触っていない |
| 108「Two small … findings for overlay-scaffold-v2 Phase 2」 | 当たっていない | Trigger は `ralph-dispatch.sh` と `scripts/check-sync.sh` の変更。どちらも触っていない |
| 110「Two cycle-3 self-review findings」 | 当たっていない | 生きている方の Trigger は `internal/cli/cli_test.go` の `TestAddPack_LegacyManifest_StaysOwnerless` の近く。触っていない |
| 429「No Go test runs `runInitInteractive`」 | 当たっていない | Trigger は `runInitInteractive` に問いか group を足す変更。`init.go` は `Long` の 1 文だけ |
| 466「`checkManifestVersion`」 | 当たっていない | `doctor.go` の変更は `checkInstalledPacks` の comment だけ |
| 401、513 の `README.md` の項 | 当たっていない | 401 は `[org].state_dir` の段落、513 は `.codex/README.md`。この PR の README の変更は Commands の表と Language packs の節 |
| 23、44、119、144、188、200、431 | 当たっていない | `filePerm()`、`tfsec` の腕、歴史的 report の行番号、`doctor_codex_writable_root_test.go`、org の件。Trigger のどれにも当たる編集がない |
| 245、249、362(`manifest.go`) | 当たっていない | org の ledger の `internal/org/manifest.go` の話で、`internal/scaffold/manifest.go` ではない |

## 他に確かめた文書

| 文書 | 結果 |
|---|---|
| `AGENTS.md` の repo map | 変えない。`internal/cli/` の行は subcommand の一覧で `pack` を挙げ、サブコマンドは並べていない。`internal/scaffold/` の行は manifest の書き方(一時ファイルと rename)に触れていない。pack remove を名指す所はない |
| `CLAUDE.md`、`.claude/rules/ralph/` | 変えない。`ralph pack`、`Manifest.Write` を挙げる所は 0 件(`git grep` で確かめた) |
| `README.md` | Commands の表(129 行目の `ralph init`、135 行目の `ralph pack add`、136 行目の `ralph pack remove`)と Language packs の節(329 行目から 333 行目)が書かれている。コードと合う |
| `templates/base/docs/recipes/adding-a-language-pack.md` | 「Removing a pack」の節がコードと合う。`check-template-purity.sh` が通る |
| `docs/recipes/adding-a-language-pack.md`(root) | 変えない。貢献者向けで、計画の Scope が `KNOWN_DIFF` と決めている |
| `docs/quality/`、`docs/architecture/repo-map.md` | 変えない。新しいファイルも script も gate もない。repo-map の recipe の行は「language packs」を挙げ、remove で意味が変わらない |
| `docs/specs/` | 変えない。`Manifest.Write` の書き方を言う文は 0 件。`ralph pack remove` を足す仕様はなく、overlay-scaffold-v2 の仕様の「`ralph pack` の UX 変更」は、その仕様の Out of scope の 1 項 |
| skill、hook、`.claude/settings.json`、language pack | 変更なし。`check-skill-sync.sh` が 14 skill で通る |
| `internal/upgrade/notes/<版>.md` | 計画の Non-goals(`/release` の段で書く)。候補は計画の Rollout にあり、A に送る |

## 直さなかったこと

- I1 の note、I3 の 2 つの comment、I4 の文書、I5 の古い仕様(上の表)
- test report の G2、G3、G4 の tech-debt の行(足す行を 1 つにする依頼だった)
- 過去の日付つきの note、report の中の古い数字

## 確認

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`
- `./scripts/check-template-purity.sh`: `PASS: no meta-repo-specific references found in templates.`
- `./scripts/check-pipeline-sync.sh`: README と AGENTS.md の pipeline の手順がそろっている
- tech-debt の編集: HEAD と突き合わせ、変わった行は 415 と、足した 527 だけ。どちらの行も 5 つのセル(` | ` で割る)、先頭が `| `、末尾が ` |`。U+FFFD は 0 件。計画の参照 `docs/plans/active/2026-11-19-pack-remove.md` と `docs/reports/verify-2026-11-19-pack-remove.md` は在る
- 計画の digest: `88ea9bb6774f`(AC を `[x]` にしたあとも同じ)

## Insight event

`./scripts/insights-append.sh --slug pack-remove --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-19-pack-remove.jsonl` に 1 行足した(rc 0、cycle 1、ts `2026-11-19T12:57:49Z`)。先の phase の 4 行は既にあった。
