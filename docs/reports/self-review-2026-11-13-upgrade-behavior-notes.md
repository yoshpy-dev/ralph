# Self-review report: upgrade-behavior-notes

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-upgrade-behavior-notes.md`(issue #317)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `2a3e2c27`、origin/main `4220ea7c` を `f55823a2` で取り込み済み)。12 ファイル、1420 行追加、5 行削除(計画ファイルを含む)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)
- Addendum: HEAD `ae1724a8`(L1 から L5 を直した commit、親は `e91aa2b3`)を `git show ae1724a8` で見た。6 ファイル、59 行追加、53 行削除。パイプラインは cycle 1 のまま。下の本文と表の L1 から L5 は `2a3e2c27` の時点の名前(`FreeUpgradeReportRelPath`、`semverWithV`)で書いたままにし、直したあとの確認は「Addendum」の節にまとめた

## Evidence reviewed

読んだもの。

- `internal/upgrade/behavior_notes.go`(187 行、全部)と `internal/upgrade/notes/README.md`
- `internal/upgrade/report.go` の差分の全部と、既存の `WriteUpgradeReport` と `ValidateRealParentChain`(`replaceplan.go:612`)。新しい `FreeUpgradeReportRelPath` との重なりを見るため
- `internal/cli/upgrade_v2.go` の差分の全部と、その前後(`ApplyOps`、no-op の判定、manifest の書き戻し)。`fromVersion` の取り方は `upgrade.go:154-157` の 2 つの呼び手まで追った
- `.claude/skills/release/SKILL.md` の全文(allowed-tools は `Bash, Read, Grep, AskUserQuestion`)と差分。`.agents/skills/release/SKILL.md` は frontmatter 以外が同じことだけ確かめた
- README の `ralph upgrade` の節の差分
- テストの差分: `behavior_notes_test.go` の前半と `TestValidateBehaviorNotes_EmbeddedNotesPass`、`report_test.go` の追加の全部、`upgrade_v2_test.go` の追加(最初の 4 つと `SameDayRerun` の全部)、`migrate_test.go` の差分
- 比べた先: `internal/cli/scaffold_version_notice.go` の `semverWithV`、`migrate.go:968` と `:1736` の移行の報告の書き手、`docs/tech-debt/README.md`(この diff は register に触れない。関係する既存の行は無い)
- repo の追跡ファイルで `templates/packs/` と `templates/base/.ralph/core/` があること(skill の「言語 pack」と「managed block の元」が `templates/` の差分に出ることの確かめ)

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 報告のファイル名の安全性: 名前は `reportNameSanitizeRe` で版と日付を削った文字列と `int` の `n` だけから作る。`FreeUpgradeReportRelPath` は名前を 1 つも見る前に `ValidateRealParentChain` を通し、symlink の `docs/` や `docs/reports/` の向こうを Lstat しない。書く側の `WriteUpgradeReport` が同じ検査と leaf の Lstat をもう一度する。symlink の `docs/reports` の拒否は `TestFreeUpgradeReportRelPath_SymlinkedReportsDir_Refused` が、ディレクトリ・ぶら下がった symlink・生きた symlink を「使用中」と数えることは `_AnyEntryCountsAsTaken` が固定している
- 版の読み方: `behaviorNoteVersion` は `semver.Canonical("v"+version) == "v"+version` を要るので、`8.2`、`8.2.0+abc`、先頭の `v`、空は名前として落ちる。`BehaviorNotesBetween` は from が to より新しい、同じ、どちらかが `dev` か空のときに早く返る。範囲は `from < v <= to` で、prerelease の to が同じ番号の release の注記を含めないことも表のテストで見た
- 埋め込み: `//go:embed notes/*.md` は README があるので空のパターンにならない。`fs.Sub(embeddedBehaviorNotes, "notes")` は妥当な path に対して失敗しないので `panic` の道は実際には通らない。形のテスト(`TestValidateBehaviorNotes_EmbeddedNotesPass`)は差し替え可能な `BehaviorNotesFS` ではなく `embeddedBehaviorNotesDir()` を直接読むので、他のテストの差し替えの残りに影響されない
- 差し替え口: `BehaviorNotesFS` を書き換えるテストは `t.Cleanup` で戻し、両 package とも `t.Parallel` を使わない(grep で 0 件)
- 本番の呼び手: `runUpgradeV2` が `BehaviorNotesBetween(fromVersion, Version)` を呼ぶのは dry-run の返りと no-op の返りの後ろ 1 か所で、移行のあとの chain も同じ関数を通る。`problems` は `notes` に足され、報告の Notes 節に出る(テストあり)。stdout の 1 行は報告を書いた run だけが出す
- 失敗の位置: 名前の決定は `ApplyOps` などの書き込みのあとに来る。移行前から `WriteUpgradeReport` の `ValidateRealParentChain` が同じ位置で失敗しうるので、位置そのものは新しい退行ではない。新しく足された失敗(100 個すべて使用中、Lstat が not-exist 以外)も manifest の壁の前で止まり、エラー文の「move some of them out ... and re-run」で再実行すれば収束する(失敗した run は manifest を進めない)
- `/release` の 4b: `allowed-tools` の `Bash`(`git diff --stat`、`git ls-tree`)、`AskUserQuestion`、`Read` だけで手順が終わる。Edit・Write を要する手順は無く、「注記が要るときは終えて PR で入れてからやり直す」と書いてある。`:(exclude)templates/base/docs/` の pathspec は正の path(`templates/`)と並んでいるので有効。`templates/packs/` と `templates/base/.ralph/core/` は追跡されているので、言語 pack と managed block の元の変更は diff に出る

読んでいない・実行していないもの。テスト、`go build`、`go vet`、linter、`check-skill-sync.sh` は走らせていない(このレビューの対象外。`/verify` と `/test` が見る)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | [解決済み ae1724a8] L1: `UpgradeReportData` の doc と `RenderUpgradeReport` の doc は「どの slice も順不同でよい、描画は path で並べる」「節の中は sorted-path 順」と言うが、新しい `BehaviorNotes` は並べ替えずに渡された順で描く(oldest first を保つのは呼び手の `BehaviorNotesBetween`)。フィールド自身の comment はこの例外を書いているが、それと食い違う 2 つの上位の doc は直していない。順序を呼び手に預ける契約が、「どの slice も」という一文で読めなくなっている | `internal/upgrade/report.go:15-16`(「All slices may be given in any order — rendering sorts by path」)、`:61-63`(「sections with content list entries in sorted-path order」)。例外を書いている所は `:28-34`。描画は `:94` 以降の `renderBehaviorNotesSection` で、`for _, n := range notes` のまま | 2 つの doc に 1 句ずつ足す(「`BehaviorNotes` を除く。これは渡された順に描く」)。描画側で semver 順に並べ直す手もあるが、`upgrade` の中でも `report.go` が semver に依存する形になるので、doc の修正だけで足りる |
| LOW | maintainability | [解決済み ae1724a8] L2: `UpgradeReportRelPath` は本番の呼び手が 0 になった。`runUpgradeV2` は `FreeUpgradeReportRelPath` を呼び、その中が最初の名前を `upgradeReportRelPathN(version, date, 1)` で自分で作る(wrapper を通らない)。wrapper の doc は「It is the first name FreeUpgradeReportRelPath tries」と言うが、値が同じだけで、つながりは呼び出しにも型にも無い。今の使い道は、upgrade の報告の名前を作る 11 前後のテストが期待値を作る oracle だけ。あわせて `Free…` は動詞の「解放する」にも読め、grep で `FreeUpgradeReportRelPath` を探す人が何をするか名前から分からない | `internal/upgrade/report.go:246`(wrapper)、`:287`(`first := upgradeReportRelPathN(version, date, 1)`)。非テストの呼び出しは `internal/cli/upgrade_v2.go:218` の `FreeUpgradeReportRelPath` だけ(`grep -rn "UpgradeReportRelPath(" --include='*.go' internal` から `_test.go` と `report.go` 自身を除いた結果) | `:287` を `UpgradeReportRelPath(version, date)` の呼び出しにして wrapper を本番の経路に入れる(1 行)。名前は `NextFreeUpgradeReportRelPath` か `UnusedUpgradeReportRelPath` のほうが、新しい名前を選ぶ関数だと読める。どちらも直さなくても動作は変わらない |
| LOW | maintainability | [解決済み ae1724a8] L3: `semverWithV` は `internal/cli/scaffold_version_notice.go:37` の写しで、comment が「It is a copy of the helper of the same name」と認めている。2 つの写しをそろえておく仕掛けは無く(テストも、共通の定義も無い)、片方だけ振る舞いが変わる(たとえば大文字の `V`)と、`ralph org` の知らせと upgrade の報告が同じ版を別に読む。`internal/cli` は `upgrade` を既に import している(`upgrade_v2.go` が `upgrade.` を使う)ので、向きの問題で写したわけではない | `internal/upgrade/behavior_notes.go:181`(写し)と `internal/cli/scaffold_version_notice.go:37`(元)。後者の呼び手は同 `:27` の 1 か所 | `upgrade` 側を `SemverWithV` として公開し、`cli` の写しを消して呼ぶ(呼び手 1 か所)。写しを残すなら、2 つの comment がお互いを名指して「ここを直すときはもう一方も」と書く。直さなくても今の動作は変わらない |
| LOW | readability | [解決済み ae1724a8] L4: README の `ralph upgrade` の節の 1 項目が 639 字から 1411 字になった。1 つの箇条に、報告の出る条件、`Version:` の行、`## Behavior changes` の節、節が出ない 2 つの条件、stdout の 1 行、同じ日の 2 回目の名前の規則が入っている。節の挙動が増えるたびに同じ箇条へ足す形で、あとから 1 つの規則だけを探して直すのが難しい | `README.md:164`(`awk 'NR==164{print length($0)}'` で 1411。`origin/main` の同じ行は 639) | 報告の節を「報告の中身」「stdout」「同じ日の 2 回目」の 3 つの子の箇条に分ける。文の中身は変えなくてよい |
| LOW | maintainability | [解決済み ae1724a8、同じ test に残る literal は N2] L5: `TestFreeUpgradeReportRelPath_GivesUpAfterTheLimit` の最初の 3 行は `upgradeReportNameLimit` を同じ値の literal `100` と比べるだけで、定数を自分の写しに固定している。限度を変えるとこの行が先に落ち、挙動の誤りを見つけたことにならない。残りのテストは定数を使っているので、この 3 行を消しても上限の検査は残る | `internal/upgrade/report_test.go:447-449`(`if upgradeReportNameLimit != 100 { t.Errorf(...) }`) | 3 行を消す。「100 個目まで調べる」ことを固定したいなら、`n <= upgradeReportNameLimit` の境界(limit 個が使用中で失敗、limit-1 個で成功)を見るテストにする(いまの「57 番だけ空ける」テストは境界を見ていない) |
| LOW | maintainability | N1(addendum で追加): S2 の Progress の行は `upgrade.FreeUpgradeReportRelPath` と書いているが、`ae1724a8` のあと `.go` の tracked file にこの名前は無い。計画に `NextUpgradeReportRelPath`、`SemverWithV`、`ae1724a8` を書いた行も無い(3 つで 0 件)。計画は `/pr` で archive されるので、この行が関数名の恒久の記録になり、archive から旧名を引いた人が存在しない関数に行き着く。S2 の行自体は `f68ab3c7` の時点では正しい | `docs/plans/active/2026-11-13-upgrade-behavior-notes.md:129`。`git grep -n FreeUpgradeReportRelPath -- '*.go'` は 0 件。`grep -c -e ae1724a8 -e NextUpgradeReportRelPath -e SemverWithV` が計画で 0 | self-review の項を埋めるときの Progress に `ae1724a8` の 1 行を足す(「L1 から L5 を直した。S2 の `FreeUpgradeReportRelPath` は `NextUpgradeReportRelPath` に改名、`semverWithV` は `upgrade.SemverWithV` に公開」)。S2 の行は書き換えず、改名の行で旧名とのつながりを残す。コードの変更は要らない |
| LOW | maintainability | N2(addendum で追加): L5 の直しは `100` を `upgradeReportNameLimit` に固定していた 3 行を消したが、同じ test の中に literal が残っている。error の確認は `"-100"` を、新しい doc comment は「`-99` と limit の 100」「`-100`」を、数値で書く。限度を変えると、挙動が合っていてもこの行が「error must name the first name and the last suffix tried」で落ち、定数から数を作るほかの行と読み方が割れる。`"-100"` の行は `2a3e2c27` からあり、最初の run で L5 と同じ型として拾い切れなかった | `internal/upgrade/report_test.go:464`(`strings.Contains(err.Error(), "-100")`)、`:444-446`(comment)。`git show 2a3e2c27:internal/upgrade/report_test.go` の `:459` に同じ literal。`fmt` は `report_test.go` の import に無い | `"-100"` を `fmt.Sprintf("-%d", upgradeReportNameLimit)`(`fmt` の import を足す)か `"-" + strconv.Itoa(upgradeReportNameLimit)` にし、comment は数を書かず「limit-1 個まで」「最後の名前」とする。直さなくても今の動作は変わらず、限度を変える日に 1 行が落ちるだけ |

重大度の数(`2a3e2c27` の最初の run): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5(L1 から L5)。

重大度の数(addendum 後、`ae1724a8` 時点): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2 件が未解決(N1、N2)。L1 から L5 の 5 件は解決済み。

## Addendum: `ae1724a8` の確認

読んだもの。`git show ae1724a8` の全部、`report.go` の `UpgradeReportData` から `renderNotesSection`(`:13-111`、`:210-219`)と名前の関数(`:236-308`)、`report_test.go:443-484`、`scaffold_version_notice.go` の import と本体、`README.md:160-167`、計画の Progress(`:126-132`)。テストと linter は走らせていない。

| 指摘 | 結果 | 確かめたこと |
| --- | --- | --- |
| L1 | 解決 | `UpgradeReportData` と `RenderUpgradeReport` の doc を「path・drift・advisory の slice は path で並べる、`BehaviorNotes` と `Notes` は渡された順」に直した。描画と一致する。`sortedCopy` が `report.go:142` と `:153`(Applied の 3 つと path 一覧の 2 つ)、`sort.Slice` が `:165`(drift)と `:188`(advisory)にあり、`renderNotesSection` と `renderBehaviorNotesSection` は `range` のまま。最初の指摘は `BehaviorNotes` だけだったが、元の doc は `Notes` についても誤っていた。直しは `Notes` も分けている |
| L2 | 解決 | `first := UpgradeReportRelPath(version, date)`(`report.go:293`)で wrapper が本番の経路に入った。名前は `NextUpgradeReportRelPath` で、`.go` に旧名は残らない(`git grep FreeUpgradeReportRelPath -- '*.go'` が 0 件)。loop は `n := 1` から `upgradeReportRelPathN` で作り、`first` と同じ値なので挙動は変わらない。wrapper の doc、`upgradeReportNameLimit` の doc、テスト名 5 つと test 内の文言も新しい名前になっている。旧名の残りは計画の 1 行だけ(N1) |
| L3 | 解決 | `SemverWithV` を `internal/upgrade` に公開し(`behavior_notes.go:183`)、`internal/cli` の写しを消した。呼び手は 2 か所(`behavior_notes.go:67`、`scaffold_version_notice.go:28`)で、doc が名指す 2 つと HEAD で一致する。本体は消した写しと同じ(`TrimSpace`、先頭 `v` の補い)なので、2 つの版の読み方は変わらない。`internal/cli` は `upgrade` を元から import している(`upgrade_v2.go`)ので循環は無く、`scaffold_version_notice.go` の `strings`(`:51-52`)と `semver`(`:29` ほか)の import は使われたまま。`semverWithV` を参照するテストと `.go` は無い |
| L4 | 解決 | `README.md:164-167` が親 1 と子 3 の箇条になった。親は 639 字(`origin/main` の同じ行と同じ長さ)、子は 377、154、262 字。文は「Next comes a」が「After the header comes a」に変わった以外同じ。2 字下げの箇条は README でこの 3 行だけで、CommonMark では親の内容の位置に揃う |
| L5 | 解決(literal の残りは N2) | 3 行の定数固定を消し、境界を両側から見る流れにした。1 から limit-1 までを使用中にすると最後の名前(`-100`)が返り、それも使用中にすると error。実装は `for n := 1; n <= upgradeReportNameLimit`(`report.go:297`)で、`<` に変えると最初の確認が、`<= limit+1` に変えると 2 つ目の確認が落ちる(読みで確かめた。走らせていない)。後ろの `len(entries) != upgradeReportNameLimit`(99 + 1)と「57 番を空ける」確認は新しい流れでも成り立つ |

挙動が変わっていないことの確かめ。

- 変更は、関数 1 つとテスト 5 つの改名、wrapper を通す 1 行(`report.go:293`)、`SemverWithV` の公開と写しの削除、doc と README の文だけ。`NextUpgradeReportRelPath` の本体は `first` の作り方以外 diff に出ない。`runUpgradeV2` の呼び出しは名前だけが変わった(`upgrade_v2.go:218`)
- 旧名の取り残しを `git grep` で探した結果は、計画の Progress の 1 行(N1)と、この報告自身の本文(`2a3e2c27` 時点の名前を意図して残した)だけ
- この commit は `docs/tech-debt/README.md` に触れない。register の行を閉じも開きもしない

## Positive notes

- 選び方(`BehaviorNotesBetween`)、形の確かめ(`ValidateBehaviorNotes`)、1 つの注記の読み(`readBehaviorNote`)、名前の判定(`behaviorNoteVersion`)が小さな関数に分かれ、実行時と CI の形のテストが同じ `readBehaviorNote` を通る。実行時は読めない注記を飛ばして 1 行残し、テストはそれを落とす、という D5 の二重の扱いが 1 つの関数から出ている
- 名前の判定が「`semver.Canonical` と一致するもの」の 1 条件で、`8.2`、`+build`、先頭の `v` を同時に落とす。判定の理由(`8.2` を `golang.org/x/mod/semver` は受け付ける)が doc comment に書いてある
- `FreeUpgradeReportRelPath` は上書きしない道を Lstat で作り、「ディレクトリ、symlink、ぶら下がった symlink も使用中」「親の chain を先に検査」「100 個で止める」を doc とテストの両方で固定している。同じ日の 3 回目までの実走(`SameDayRerun_KeepsEarlierReports`)が、drift のエラー文と stdout の `report:` の行が実際に書いた名前を出すことまで確かめる
- `renderBehaviorNotesSection` は注記が無いとき何も書かず、`NoBehaviorNotes_ByteIdentical` が節を足す前の出力とのバイト一致を、nil と空 slice の両方で固定している。`OnlyAddTheSection` は「注記の有る報告から節を除くと注記の無い報告と同じ」を直接の等式にしている
- `/release` の 4b は、skill が書けないこと(Edit・Write が無い)と、そのため注記が要るときは「終えて PR で入れてからやり直す」を最初の段落で言い、やり直しの step 3 に注記の PR が見えることまで書く。Completion gate と step 9 の報告の両方に 4b の結果が入っている
- (addendum)L5 の直しは、定数を literal に固定する代わりに境界の両側(limit-1 個で成功、limit 個で失敗)を見るテストに替えた。off-by-one の 2 方向がそれぞれ別の確認で落ちる
- (addendum)L3 は共通の定義を 1 つにしたうえで、doc に呼び手 2 つを書いた。HEAD で数えて一致し、呼び手が増えたときに探す出発点になる

観察として残すもの(指摘にはしない)。

- `BehaviorNotesFS` は production が書かない公開の package 変数で、doc が「nothing else sets it」と言う。`scaffold.EmbeddedFS` と同じく package をまたぐテスト(`internal/cli`)が差し替える口で、他に形が無いので指摘にしない。もう 1 つの代案(非公開の変数に test 用の setter)は、結局 setter が公開になる
- `FreeUpgradeReportRelPath` の Lstat と、そのあとの `os.WriteFile` の間は check-then-act で、`O_EXCL` ではない。この diff の前の `WriteUpgradeReport` も leaf の Lstat のあとに `WriteFile` していて、同じ隙間だった。2 つの `ralph upgrade` が同じ repo で同じ日に同時に走ることは想定外なので、新しい退行にはしない。「上書きしない」を厳密にするなら、`WriteUpgradeReport` の書き込みを `O_CREATE|O_EXCL` にする(ただし移行の報告の書き手 `migrate.go:968` も同じ関数を使うので、そちらの挙動も変わる)
- AC9 のテストは `first` の名前を実走の前に `time.Now().UTC()` で 1 回作り、3 回の run がそれぞれ `time.Now().UTC()`(`upgrade.go:157`)で日付を決める。UTC の 0 時を 3 回の run のあいだに跨ぐと名前がずれて落ちる。同じ作りは既存の 11 個のテストにもあり(1 回の run につき 1 回の窓)、このテストは 3 回の窓を持つ。`runUpgradeIOWithOptions` に時刻の差し込み口が無いので、直すなら口を足す変更になる。数 ms の窓なので指摘にしない
- 計画の Progress が「範囲外の気づき」に挙げた移行の報告 `ralph-migration-<date>.md`(`migrate.go:1736`)の上書きは、移行が manifest を v3 に書き直してから報告を書く(`migrate.go:958-968`)ので、同じ日に再び移行が走るのは manifest を旧版へ戻したときだけである。register の行にするほどの影響は見つからなかった
- `/release` の step 4 の既存の字下げのずれは、計画が範囲外とした。4b を挟んだことで「`4b.` は番号つきの項目ではなく段落」になり、`5.` から新しい list が始まるが、CommonMark の描画では step の番号は読める
- (addendum)`SemverWithV` は版の読み方の共通部品だが、`behavior_notes.go` に置かれている。`internal/cli` の `scaffoldVersionOlder` はこの file を知らずに呼ぶので、探すときは `SemverWithV` の名前で引くことになる。名前が grep できるので指摘にはしない。doc の呼び手の列挙は非公開の `scaffoldVersionOlder` を名指すので、その関数を改名したときは doc の側も直す必要がある

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (なし) | - | - | - | - |

_(行は足していないので `docs/tech-debt/` への追記もしない。L1 から L5 は `ae1724a8` で直った。残る N1 と N2 は直さずに進める場合に限り、`/pr` の側で Known gaps に書くか register に足すかを決める。N1 は計画の Progress に 1 行足せば閉じる。この diff は register のどの行も閉じていないし、開いてもいない。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、未解決の LOW 2。addendum 後。`2a3e2c27` の最初の run は LOW 5 で、L1 から L5 は `ae1724a8` で解決済み)
- Follow-ups: 残る N1 と N2 はどちらも直さなくても動作は変わらない。N1 は計画の Progress に `ae1724a8` の 1 行を足す文書の直しで、コードの変更は無い。N2 はテスト 1 行と comment の直しで、直すならコードの変更なので、パイプラインの規則どおり `/self-review` から回し直す(cycle 2 になる)。直さず進める場合は、N2 を PR の Known gaps に書く
- insight event は足していない(依頼どおり)。`ae1724a8` の確認は同じ cycle 1 の addendum として扱う
