# Self-review report: split-shared-paths

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-split-shared-paths.md`(issue #363)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(`cycle-count.json` は cycle 2)。Run 1 は下にそのまま残す。今の判定はこの節の Recommendation の行
- Scope: Run 1 のあと(`4a4be9de..HEAD`)にこのブランチ自身が足した 9 コミット(HEAD `2c0c28e6`)のうち、diff の質を見るもの。コードは `143deb8f`(Run 1 の LOW 8 件の直し)と `9535c7a7`(コメント、`--reserve` の help、recipe の文言)。文書とテストは `6168f606`(spec と plan を拒否に合わせた)、`5cdc7150`(sync-docs の本体: tech-debt の行、コメントの plan の指し方、skill の 1 文)、`2c0c28e6`(cross-review の ACTION_REQUIRED の直し: org skill の「共有のパスの直し方」の 1 の 4 写しと、`internal/org/skill_shared_paths_test.go` の固定)。pipeline の記録だけの `77a6961a`、`84bdf572`、`c1d3628b`、`48f86f25` と、main の取り込み `809a887e` が連れてきた #362 のコミットは見ていない。仕様への適合、テストの網羅、文書の食い違いの網羅は見ていない。テストと静的解析は走らせていない。手順の主張を確かめるために、repo の外の使い捨ての場所(scratchpad)で git と pnpm を打った。repo の中では何も実行していない

### Run 1 の LOW 8 件の行き先

番号は Run 1 の Findings の表の行の順(C1-LOW n)。

| C1 | 内容 | 結果 | 確かめたこと |
| --- | --- | --- | --- |
| LOW 1 | `sameFeature` の doc の前提 | 直った | `reserve.go` の doc が「feature digest に入るヘッダの行は `- Shared:` だけで、全体の digest にも入る」になり、次の文と矛盾しない |
| LOW 2 | タスクの共有のパスの行の文言 | 直った | `feature.go` の行が「上の `- 予約したパス:` の行の --scope に、これらのパスもすでに入っている」。`featureLeaderTask` の doc と、固定していた 3 つのテスト(`feature_shared_task_test.go`、`feature_shared_test.go`、`prompts_leader_shared_test.go`)が同じコミットで変わっている |
| LOW 3 | ヘッダの外の `- Shared:` が黙って本文になる | 直った(拒否) | 下の「`- Shared:` の位置の拒否」 |
| LOW 4 | `start --help` の立て直しの条件 | 直った | `internal/cli/org.go` の 2 か所(条件の文と feature digest の説明)に共有のパスが入った |
| LOW 5 | `reservationDecision` の長さと 3 つ目の文 | 直った | 57 行が 42 行、新しい `reservationOverlapErr` は 20 行。検査の順(予約と予約、共有と予約、予約と共有)と文は同じで、3 つ目だけ「the path split plan %q shares」になった。`.Split` は `SharedPaths()` が空でないときだけ読むので、`Feature` が nil でも落ちない |
| LOW 6 | 損なわれた `shared=` の痕跡と reseat の `--scope` | 直った(doc の 1 文) | `featureBindingFromTokens` の doc。`reseat.go:470-476` のコメントと言っていることが同じ |
| LOW 7 | director skill の手順 1 | 直った | 「ファイルの値は今のもので、記録と違うことがある。start の断りに従う」。テストが 2 文を固定している |
| LOW 8 | `featureLeaderTask` の組み立てと `parseSplitShared` の切り方 | 直った | (a) 末尾を別の文で `append`。(b) `splitPathsErr` を `parseSplitReserve` と `parseSplitShared` が共有する。`- Reserve:` の拒否の文の頭が `reserve path` から `path` になり、`split_test.go` の 3 行が固定している。CLI の `--reserve` は `NormalizeReservePaths` を直接呼ぶので `reserve path` のまま(`internal/cli/org_reserve_test.go:100`)。Run 1 が挙げた「空白とカンマ」の文の `a reservation cannot hold`(`reserve.go:309`)は残るが、`shared=` の token も予約の event の field なので偽ではない |

半分だけ直った行と、台帳に載らないまま開いている行はない。

### `- Shared:` の位置の拒否(`143deb8f`)

- 置き場所は `split.go:388-392`。`header.read` が先頭の `- Shared:` を読んで `continue` したあとで、`!inFeatures` の分岐(前文に溜める)と機能の節の読みより前にある。前文、`## Features` の最初の `###` より前、機能の節、`## Features` より後の節は、どれもここを通ってから行き先に入る
- `fenced` が真の行は拒否しない。フェンスを読む `fence.read` が先に走るので、閉じる行までコードブロックの中として通る。行頭で比べる(`strings.HasPrefix`)ので、ヘッダの読み(`splitHeader.read`)と `isStatusOrApprovedLine` と幅がそろっている。字下げした行と、文の途中で `Shared:` に触れる行は本文のまま(テストが固定している)
- 文は行番号、理由(共有のパスはヘッダだけから読むので、ここでは何も共有しない)、直し方(ヘッダに移す、またはコードブロックに入れる)を言う。`- Status:` の拒否の文と同じ形
- 拒否のテストは 7 ケース(前文、機能、ヘッダにもある場合、空の値、`## Features` の直後、`## Features` より後の節、閉じたフェンスのあと)で、`misplaced` の定数 1 つに固定している。skill の director 側と org 側(「分割計画」と「承認」の拒否の列挙)、spec の FR-2、plan の S1 の記録が同じ向きに直っている。「本文のまま」「読まれず」と言う古い文は、このブランチの文書とコードに残っていない(grep)
- 拒否で困る点は 1 つだけ。R2-3

### 手順(`2c0c28e6`)の確かめ

| 手順の主張 | 確かめ方 | 結果 |
| --- | --- | --- |
| merge の途中の `HEAD` が取り込む前の先頭で、`git diff "$(git merge-base origin/<既定> HEAD)" HEAD -- <パス>` がこのブランチの変更だけを出す | 使い捨ての repo で lockfile 相当のファイルを衝突させた | 合っている。`HEAD` は衝突の途中も元の先頭のまま |
| rebase の衝突の途中の `ORIG_HEAD` | 3 コミットのうち 2 つが同じファイルで衝突する rebase | 合っている。最初の停止でも、`--continue` で 2 つ目の衝突に進んだあとでも、rebase が終わったあとでも、`ORIG_HEAD` は元の先頭。`git merge-base main ORIG_HEAD` は分岐点で、`git diff` はブランチ全体の変更を出す |
| 作り直しで lockfile だけの版が黙って戻る | `"ms": "^2.0.0"`、lockfile が 2.0.0 の状態で `pnpm update ms --no-save`(lockfile が 2.1.3、package.json は同じ)。main の lockfile に戻して `pnpm install` | 合っている。2.0.0 に戻る。手順の仕組みの主張は正しい |
| `pnpm update <パッケージ>@<版>` で版が戻る | 同じ状態から | lockfile は 2.1.3 に戻る。ただし R2-1 |
| 例の「`package.json` の範囲のまま `pnpm update <パッケージ>` で上げた版」 | `pnpm update ms` | package.json も `"^2.1.3"` に書き換わる。R2-1 |
| 間接の依存 | `micromatch` の下の `braces` で `pnpm update braces@3.0.3` | pnpm が断る。R2-1 |
| `go.sum` は版を `go.mod` が決めるので控えるものがない | 読みだけ(実行していない) | 異論なし。`go.sum` は checksum の列で、`go mod tidy` は行を足し引きする。未確認 |

pnpm は手元の 12.9.1(mise の latest)。ほかの版、npm と yarn の lockfile は確かめていない。

## Findings (Run 2)

番号は R2-n。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R2-1 | MEDIUM | maintainability | 「控えた版を戻す」手順の例と再適用の命令が、名指す pnpm の動きと合わない。手順の組み立て(版を控える、作り直す、同じ版を入れ直して確かめる)は正しく、命令が動かないときは末尾の BLOCKED に落ちるので安全側だが、書いてあるとおりにすると (a) 例の「`package.json` の範囲のまま `pnpm update <パッケージ>` で上げた版」は、lockfile だけの変更ではない。`pnpm update ms`(`"ms": "^2.0.0"`)は package.json を `"^2.1.3"` に書き換える。lockfile だけで上がるのは `--no-save` のときと間接の依存のとき (b) 再適用の `pnpm update <パッケージ>@<版>` も package.json の範囲を書き換える。手順 2 で merge したばかりの手書きの共有のファイルを、衝突の直しのコミットの中で、機能がしていない変更として動かす。`--no-save` を付けたときだけ package.json が動かない (c) 間接の依存は、lockfile だけで上がる版のいちばん多い出どころだが、pnpm が版つきの update を断る。手順が言う「同じ更新のコマンドを打ち直す」は成り立たず、作り直しのあとの確かめで戻っていると分かっても、手順は入れ直し方を言わない | `.claude/skills/org/SKILL.md:1726-1727`(例)、`:1737-1738`(再適用)、`:1739-1742`(`go.sum` と BLOCKED)。4 写し(`.agents/skills/org/`、`templates/base/` の 2 つ)は `cmp` で同一。pnpm 12.9.1 の実測: `pnpm update ms` は package.json を `^2.1.3` にし、`pnpm update ms@2.1.3` も同じ、`--no-save` を付けると package.json は `^2.0.0` のまま lockfile だけ 2.1.3。`pnpm update braces@3.0.3`(間接)は `ERR_PNPM_UPDATE_VERSION_ON_INDIRECT_DEP`(「"braces" (requested "3.0.3") is not a direct dependency, so the requested version cannot be recorded」、版を固定するなら pnpm-workspace.yaml の `overrides` に足すよう案内)。テストは再適用の文を固定している(`internal/org/skill_shared_paths_test.go:132`)。例の文は固定していない | 例を実際に lockfile だけで上がる場合に直す(`--no-save` で上げた版、間接の依存を上げた版)。再適用は直接の依存なら `pnpm update <パッケージ>@<版> --no-save` と書き、間接の依存は「pnpm は版つきの update を断る。版の固定は手書きのファイル(`overrides`)の変更なので、手で足さず BLOCKED で leader に返す」と書く。4 写しと `skill_shared_paths_test.go:132` の固定を合わせる。skill とテストに触れるので記録だけの修正ではなく、`/verify` 以降の再実行が要る |
| R2-2 | LOW | readability | 手順 1 が 1 つの番号項目に 20 行(`:1723-1742`)で、マージ元の直し、版を控える、見つけ方、作り直し、再適用と確かめ、`go.sum` の例外、BLOCKED の 6 つを詰める。語も 2 つ紛らわしい。「控える」は書き留めるの意味だが、この repo の spec は `merge を控える` を「しない」の意味で使う(`docs/specs/2026-10-07-org-multi-org-director.md:216`)。`<先頭>`(取り込む前のブランチの先頭)は、同じ skill の「先頭には、任意で `- Shared:` の行を 1 つ置ける」の先頭(分割計画のヘッダ)と同じ語で、grep で 3 つの意味が混ざる | `.claude/skills/org/SKILL.md:1725`、`:1728-1730`、分割計画の節の `- Shared:` の説明 | 項目を 1 の中の小さな手順(控える、作り直す、入れ直す)に分け、「書き留める」と言い、placeholder を `<取り込み前の先頭>` のようにする。固定している文(`この機能が生成されるファイルだけで上げた版を控える`、`git diff` の行)も合わせて直す。R2-1 と同じ行なので、一緒に直すのがよい |
| R2-3 | LOW | exception-handling | ヘッダにすでに `- Shared:` の行があるとき、ヘッダの外の行の拒否の文は「ヘッダに移す」と言う。そのとおり別の行としてヘッダに移すと、次の読み込みは「a second - Shared: line」で断る。1 往復ぶん遠回りになるだけで、どちらの文も次の手を言う。テストが「ヘッダにもある場合」を含めて文を 1 つの定数に固定している | `internal/org/split.go:388-392`、`internal/org/split_shared_test.go` の `misplaced` と「a Shared line in a feature, with one in the header」 | 変更は任意。直すなら「ヘッダの行があるときはその行に足す」と文に 1 句足し、定数 `misplaced` を合わせる |

## Positive notes (Run 2)

- Run 1 の 8 件は 1 コミット(`143deb8f`)で全部直り、コード、help、skill 4 写し、spec、plan、テストの固定が同じ向きに動いた。直したあとの文書に古い言い方が残っていない(grep)
- `reservationOverlapErr` は、決定の関数から断りの組み立てだけを出し、検査の順と文を変えていない。doc が順と、共有どうしを検査しないことを書いている
- `5cdc7150` の register への追記は、既存の行に「2026-11-22 に当たった」の注記を足すだけの形で、行の途中の語を書き換えていない(語単位の diff で確認)。新しい 3 行(status の表示、`reserve.go` が 822 行、既知の穴)は HEAD で成り立つ(`wc -l internal/org/reserve.go` は 822)
- 手順の git の主張(`HEAD`、`ORIG_HEAD`、`merge-base`)はそのまま使える。`ORIG_HEAD` が rebase の途中ずっと元の先頭であることは、手順の命令に残してよい

## Tech debt identified (Run 2)

R2-1 と R2-2 を今回の周で直さないなら、cycle の上限に達した最後の周なので、直すつもりの所見も繰り越しとして台帳に 1 行要る。`docs/tech-debt/README.md` は、ほかのレーンの PR と末尾が衝突しやすいので、この周では足していない。`/pr` の時点で足す行の案:

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| The step that keeps a lockfile-only version (`### 共有のパスの直し方`, step 1 of the `/org` skill, 4 copies; feat/split-shared-paths, #363) names `pnpm update <pkg>` as a change that leaves `package.json` alone and `pnpm update <pkg>@<version>` as the re-apply. With pnpm 12.9.1 both rewrite the range unless `--no-save`, and a version on an indirect dependency is refused (`ERR_PNPM_UPDATE_VERSION_ON_INDIRECT_DEP`). The step is also one 20-line list item whose `<先頭>` and 「控える」 read like other words of the skill and the spec | A leader that follows it edits `package.json` in a conflict-fix commit, or ends in BLOCKED for an indirect dependency, the commonest lockfile-only change; the BLOCKED sentence keeps it safe | Not fixed in cycle 2 of 2 | The next edit to this step, or a report of a regeneration that dropped a version the feature raised | `docs/reports/self-review-2026-11-22-split-shared-paths.md` (Run 2, R2-1, R2-2) |

## Recommendation (Run 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 2)
- Follow-ups:
  - R2-1 を直してから `/pr` に進むのがよい。手順の組み立ては正しく、直す文は 1 項目の中の 3 か所(例、再適用、間接の依存)と、4 写しと、テストの固定の 1 行で済む。直さないなら、上の台帳の行を `/pr` で足す
  - R2-2 は R2-1 と同じ行を触るので一緒に直すのが安い。R2-3 は直さなくてよい
  - どれもコードではなく skill とテストの文言の直しだが、skill の本体とテストに触れるので記録だけの修正ではなく、直したら `/verify` 以降の再実行が要る。この周の中で直す分は cycle の上限に数えない
  - `143deb8f` の拒否、`reservationOverlapErr`、`splitPathsErr` のコードに直すものは見つからなかった

## Run 1 (cycle 1)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-split-shared-paths.md`(issue #363)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `477594d1`、base `origin/main` 9cd63d60)。コードの変更は S1 `35458c51`(`split.go`)、S2 `2bb50cd4`(`reserve.go`、`feature.go`)、S3 `89429da9`(`watch.go`、`watch_scope.go`、`feature.go`、`prompts/leader.md`)、S3b `c9e99c63`(`reseat.go`)、文書は S4 `eb43b1b7`(skill の 4 写し、README、AGENTS.md、2 つの仕様、skill のテスト)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いの網羅は見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テストは 1 本も走らせていない

## Evidence reviewed

- `git diff origin/main...HEAD` のコード部分の全部(`split.go`、`reserve.go`、`feature.go`、`watch.go`、`watch_scope.go`、`reseat.go`、`prompts/leader.md`)と、skill の director 側の diff、org 側の変更行(1 行が長いので `-U0` と切り詰めで読んだ)
- 周辺の全文読み: `parseSplitPlan` の読みの本体(`split.go:343-415`)、`splitFeatureDraft.read`(`split.go:599-625`)、`NormalizeReservePaths` 一式(`reserve.go:274-360`)、`featureBindingFromTokens`(`reserve.go:459-492`)、`reservationDecision` の全体(`reserve.go:686-742`)、`PlanReseat`(`reseat.go:440-560`)、`idempotentRespawnDecision` と `spawnCapacityErr`(`spawn.go:1599-1708`)
- 洗い出し: `FeatureBinding{` の組み立て(非テストは `feature.go:513` の 1 か所と `featureBindingFromTokens` だけ)、`.Shared`・`SharedPaths()`・`featureDigest(` の非テストの参照、`ActiveReservation(` と `activeReservation(` の読み手(`verbs.go:1961` と `:1987`、`status_followup.go:84`、`cli/org.go:2100` は表示か復元で、共有のパスは要らない)、`FeatureBinding` を値で比べる箇所(無い。`sameFeature` と個々の field の比較だけ)
- 新しいテストの読み(質の目視): `feature_shared_task_test.go` 全体、`reserve_shared_test.go` の `TestReservationDecision_Shared`、`prompts_leader_shared_test.go`、`skill_shared_paths_test.go` の前半、`watch_scope_shared_test.go` の fixture、`split_test.go` と `reserve_test.go` と `watch_scope_test.go` と `reseat_test.go` の既存テストの変更行

### 頼まれた観点ごとの確認

| 観点 | 確認したこと | 結果 |
| --- | --- | --- |
| digest と Details の後方互換 | `featureDigest`(`split.go:727-745`)は `shared` が空のとき field を足さないので、`- Shared:` のない計画のバイト列は変わらない。`TestFeatureDigest_Encoding` が空の `[]string{}` と nil を含めて固定している。Details は `shared=` を最後のトークンに置き(`reserve.go:378`)、古い binary は最初の未知の鍵から先を note として読む。`featureBindingFromTokens` の鍵の表に足しただけなので、`shared=` のない記録は `Shared` が `""` で、`Complete()` にも `RequiresApproval()` にも関わらない | 問題なし。ただし `- Shared:` で始まる自由文をすでにヘッダに書いた計画は、新しい binary で読み込みを拒否される(空白を含む文は「空白を含むパス」で断られる)。鍵が #363 で初めて決まったものなので現実の影響は小さい |
| 損なわれた `shared=` の読み | 下の「損なわれた `shared=` の評価」 | 向きは妥当。細かい点は LOW 6 |
| 受け付けと並行 | 共有のパスの 2 方向の判定は `reservationDecision` の中で、呼び出しは `checkCapacityAndStart` の manifest lock の下(`spawn.go:1599-1618`、idempotent 側は `:1661-1708`)。dry-run と `spawnPrecheckErr` は lock なしの読みで、予測だけ。reseat の `reservationDecision(sim, ...)`(`reseat.go:539`)も計画の段階の予測で、本物の spawn が lock の下で決める | 問題なし。`want.Feature` と `held.Feature` が nil でも `SharedPaths()` が nil を返すので、nil 参照にならない |
| パスの正規化の一貫 | split(`parseSplitShared`)、`sharedFromToken`(記録の読みと `validateFeatureBinding`)、見張り、reseat のどれも `NormalizeReservePaths`(並べ替えと重複除去を含む)を通り、`strings.Join(",")` と `Split(",")` で往復する。`.` は split で断り、読みでは `""` に落とす。重なりの判定は split の `finishSplitPlan` と `reservationDecision` が同じ `reserveOverlaps` を使う。leader のタスクの `--scope`(`feature.go:588`、予約、共有の順)と reseat の `--scope`(`reseat.go:476`、同じ順)は並びも同じ | 問題なし |
| leader のタスクの shell の引用 | `shellQuote(scope)` は予約と共有を合わせた 1 つの語を作る。`$`、バッククォート、引用符、バックスラッシュ、glob を含む 4 つの組を `/bin/sh` で 1 語に読むテスト(`feature_shared_task_test.go:73-117`)がある。`- 共有のパス:` の行の生のパスは、予約の行と同じく表示用で、leader が写すのは `--scope` の引数のほう | 問題なし。文言は LOW 2 |
| エラー文 | 3 つの断り(計画の読み込み、共有と予約、予約と共有)は、どちらの側が持つパスか、重なりの組、直し方か待つ先を言う。`featureDigestRefusal` に共有のパスの前後の値を出す枝が足された | 言い回しの点は LOW 5 |
| skill が言うこととコード | director の「共有のパス」の各文(置き場所、`.` と重なりの拒否、digest が全部変わる、立て直しと reseat の拒否、`--scope`、受け付けの両順)と、org の「共有のパスの直し方」の手順を、コードと突き合わせた。ヘッダ以外の `- Shared:` が本文になること、共有どうしが重なってよいこと、reseat の `--scope`、「分割計画の承認」の 4 と「機能ごとの出力先」の 1・3 の番号の指し先は合っている | 偽の主張は見つからなかった。director の手順 1 が計画のファイルを読ませる点だけ LOW 7 |

### 損なわれた `shared=` の評価

`shared=` が読めない記録(値が空、鍵が 2 回、規則に反するパス、`.` を含む)を「共有のパスなし」と読む選択を、`.`(repo 全体)と読む案と比べた。

- 見張り: 「なし」と読むと、共有のはずのパスの変更に scope_change が立つ。うるさい側に倒れる。「repo 全体」と読むと、その org の変更を何も知らせなくなり、静かに緩む側に倒れる。予約のパスが読めないときに repo 全体と読むのは、受け付けで重なりを塞ぐ向きが目的で、見張りは同じ読みの副作用で黙る(既存)。共有のパスにまで同じ副作用を持ち込まなかった判断は妥当
- 受け付け: 「なし」と読むと、その org の共有のパスが別の org の予約から守られなくなる(`reserve.go:452-458` の doc が代償として書いている)。この側は静かだが、守られないパスを変えれば見張りが知らせるので、無音にはならない
- 損なわれた記録を誰が作れるか: manifest は ralph だけが書く。座席が台帳に書けることは台帳全体の前提で、この PR が広げるものではない
- 一部のパスだけ壊れた値は、全部を捨てて「なし」にする(`sharedFromToken` は 1 つでも断られたら `""`)。うるさい側に倒れるだけで、足りないものは出ない

評価は、向きは妥当で変更不要。残る点は LOW 6(痕跡が残らないことと、reseat の `--scope` が計画でなく記録から作られること)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `sameFeature` の doc が「ヘッダの行は feature digest に入らない」を前提にしているが、S2 の変更で `- Shared:` の値が入るようになり、前提が偽になった。結論(全体の digest が同じなら feature digest も同じ)は今も正しいが、同じ段落の次の文(S2 が足した「共有のパスはどちらの digest にも入る」)と矛盾して読める | `internal/org/reserve.go:206-207`(`and no header line goes into a feature digest`)。直後の `reserve.go:209-213` が S2 の追加。`split.go:57-58` の「ヘッダの自由な文は行き先がない」は `- Shared:` を自由な文に数えないので成り立つ | 「`- Shared:` の行の値のほかに、ヘッダの行は feature digest に入らず、その行は全体の digest にも入るので」の形にする |
| LOW | readability | leader のタスクの共有のパスの行の括弧「座席の spawn の --scope には予約したパスと合わせて渡す」は、leader に自分で合わせさせる指示に読める。直前の `- 予約したパス:` の行の `--scope` はすでに予約と共有の合算で、`leader.md:132-133` は「両方が入っています」と言い、手順 3 と 5 は引数を書き換えずに付けると言う。leader が `--scope '<予約>, <共有>'` を組み立て直しても動くが、RC2 の直しが避けた「leader が引数を作る」形に戻る | `internal/org/feature.go:598-599`。同じ文が `feature_shared_task_test.go:22` と `:125`、`prompts_leader_shared_test.go` の期待に固定されている | 「座席の spawn の --scope は、上の行の括弧のとおり予約したパスと合わせてある」のように、できている事実を言う文にする。固定している 3 か所を合わせて直す |
| LOW | maintainability | ヘッダの外に書いた `- Shared:` は黙って読まれず、前文なら本文として leader に渡る。機能の節に書くのは `- Reserve:` と並べやすい間違いで、機能ごとに共有を変えるという誤解(計画の Non-goals が機能ごとを外している)をそのまま通す。同じ種類の「digest が見ない行」は `- Status:` と `- Approved:` がヘッダの外に出ると断られる(`isStatusOrApprovedLine`)ので、扱いがそろっていない。結果は scope_change の ALERT が出る(静かに緩まない)ので重くないが、書いた人は何も共有していないと気づく手がかりが ALERT だけ | `internal/org/split.go:599-607`(`splitFeatureDraft.read` は未知の行を本文にする)、`split.go:378-382`(ヘッダの外の `isStatusOrApprovedLine` の断り)。決定は `split_shared_test.go:88-95` が固定、skill の director 側が注意を書く | 決定済みなので変更必須ではない。直すなら、フェンスの外の `- Shared:` 行を機能の節(と前文)で断る。引用したい本文はフェンスに入れる、と断りの文に書く。直さないなら、director skill の「前文の節か機能の節に書いた…」の文を残す現状でよい |
| LOW | maintainability | `ralph org start --help` の立て直しの条件が、共有のパスの変化を数えていない。今は「計画の前文と機能の節が変わっていなければ」と言うが、`- Shared:` の行が変わっても断られる(`featureDigestRefusal` の新しい枝)。この PR が作った条件なので、この PR の中で 1 句足すのが筋で、後回しにすると `/verify` の文書の食い違いの検査と `/sync-docs` が同じ指摘を出す。断りの文は共有のパスを名指すので、影響は小さい | `internal/cli/org.go:1094-1100`(`as long as the plan's preamble and the feature's own section did not change`)。条件は `reserve.go:761-765` | 「前文、機能の節、ヘッダの `- Shared:` の行が変わっていなければ」と 1 句足し、feature digest の説明(`org.go:1097-1100`)にも「と共有のパス」を足す。`internal/cli` に help のテストがあれば合わせる |
| LOW | readability | `reservationDecision` が 43 行から 57 行(`reserve.go:686-742`)になり、50 行の目安を超えた。ループの中の 3 つの断りが長い書式文字列を並べ、最初の 1 つ(予約どうし)と後の 2 つ(共有)で書き方も違う。3 つ目の文の「then the one split plan %q shares」は「その分割計画が共有するパス」と読めるまで引っかかる | `internal/org/reserve.go:722-741`、とくに `:736` | 3 つの断りを `reservationOverlapErr(other, want, held)` のような 1 つの関数に出し、ループは「どの組が重なるか」だけにする。3 つ目の文は「then the path split plan %q shares」にする(テストが文を固定しているので、`reserve_shared_test.go` の `sharedThere` も合わせる) |
| LOW | null-safety | 読めない `shared=` は「なし」と読まれ、「記録に無い」と区別がつかない。`feature_digest=` は `String()` が `no feature digest` と言い、`approval=` は `approvalModeUnreadable` を持つが、共有のパスにはどちらも無い。また reseat の非 leader の座席の `--scope` は記録の `binding.SharedPaths()` から作られ(`reseat.go:476`)、leader のタスクは計画の `plan.Shared` から作られる(`feature.go:588`)。通常は feature digest が同じなので一致するが、記録が損なわれたときは、feature digest が同じで再起動が通り、leader のタスクは共有のパスを言い、座席の `--scope` はそれを持たない。損なわれた台帳の中だけの話 | `internal/org/reserve.go:491`、`:452-458`(doc の「代償」)、`reseat.go:476`、`feature.go:588` | 変更なしでよい。`featureBindingFromTokens` の doc の「代償」の段落に「reseat の `--scope` も共有のパスを持たなくなる」と 1 文足せば、読み手が気づける |
| LOW | maintainability | director skill の手順 1 は、走っている機能の org の共有のパスを `ralph org status` が出さないので、`feature:` の行が名指す分割計画の台帳の `splits/<split>.md` の `- Shared:` の行を読ませる。org が持つ値は start のときの記録で、計画のファイルを承認し直して変えたあと(止めていない org は立て直しの検査を受けない)は食い違うことがある。start の断りが決め手なので、食い違いの影響は director の数え直しと説明の誤りまで。計画が言う未対応の「status は共有のパスを出さない」の裏返し | `.claude/skills/director/SKILL.md` の「機能ごとの出力先」の 1(手順 1 の追加部分)。状態の表示は `internal/cli/org.go:2095-2145`(`orgFeatureJSON` と `printStatusTable`)。`orgFeatureJSON` は `Digest` と `FeatureDigest` を出す | status の表示は、この PR に入れない(下の「範囲外の 2 件の扱い」)。skill の文には「ファイルの値は今のもので、org が start のときに記録した値と違うことがある。start の断りが決め手」と 1 句足す |
| LOW | readability | (a) `featureLeaderTask` が 1 つのスライス・リテラルを `lines` と末尾の 9 個の要素に分け、共有の行を条件で足したあと `strings.Join(append(lines, ...9 個...), "\n")` で戻す。行の順を読むのに 2 か所を行き来する。(b) `parseSplitShared` は `NormalizeReservePaths` の断りの文から `"org: reserve "` を前置きごと切る。兄弟の `parseSplitReserve` は `"org: "` を切るので、2 つで切り方が違い、`normalizeReservePath` の文面が変わると共有のほうだけ `org: reserve` が残る。「空白とカンマを含む」の文には共有の文脈で `a reservation cannot hold` が残る | (a) `internal/org/feature.go:590-612`、(b) `split.go:539` と `:641`、`reserve.go:306-312` | (a) 共有の行を入れる位置に `slices.Insert` を使うか、末尾を `tail` に名前を付けて `slices.Concat(lines, tail)` にする。(b) 切り方をそろえる(どちらも `"org: "` を切る)か、文面の変更でテストが落ちることを `TestLoadSplitPlan_SharedRejects` が固定していることを doc に書く |

## Positive notes

- `featureDigest` は「共有のパスがあるときだけ field を足す」1 点に絞った。`digestField` を使った書き直しで、既存の field の順とバイト列は変わらず、`TestFeatureDigest_Encoding` が文字列のレベルで固定している。AC2(既存計画の digest が同じ)の根拠が計算の形で見える
- 共有のパスを `FeatureBinding.Shared`(カンマでつないだ文字列)にして、`FeatureBinding` が `==` で比べられる性質を保った。`Reservation.Shared []string` にしなかった理由(`SpawnParams.reservation()` が範囲外で組み立てる)と、記録の read/write の対称性が `reserve.go` の doc に書いてある
- 受け付けの 2 方向の判定を manifest lock の中の 1 関数に置き、共有どうしだけを通す規則をテストの表(ファイルとディレクトリ、同じ prefix の別ディレクトリ、全ペアの名指し)で固定した。重なる計画の側が `finishSplitPlan` と受け付けで同じ `reserveOverlaps` を使うので、判定の規則が 1 つ
- 見張りは「外」を決める入力に共有のパスを足しただけで、`scopePathInside` の規則と ALERT の文を変えていない。`ActiveReservation` を `activeReservation` に替えた 1 行で、予約の長さの判定(`len(r.Paths) > 0`)は同じ
- 共有のパスのない計画の leader のタスクが 1 バイトも変わらないことを、共有なしの表(nil と空)でテストが固定している。leader の prompt のテストは、タスクの行の語(`featureSharedLead`)を prompt の文と突き合わせる
- 範囲外の発見(reseat の `--scope` が予約だけだった)を S3 で見つけて S3b に分け、同じ PR の中で閉じた。reseat の 3 か所(spawn、dry run、`reseatCustomRoleErr` のコマンド)が同じ `scope` 変数を共有している

### 範囲外の 2 件の扱い(plan の Progress の S4 の記録)

- `ralph org start --help` の立て直しの条件(LOW 4): **この PR に入れるべき**。この PR が条件を足したので、1 句で足りる
- `ralph status` と `ralph org status` が共有のパスを出さない: **この PR に入れなくてよい**。`orgFeatureJSON` の field と table の行という出力の形が増え、テストの組も増える。director は計画のファイルから読めて、start の断りが決め手になる。ただし skill の手順 1 の文に「ファイルの値と記録の値が違うことがある」の 1 句は足したい(LOW 7)。この PR で入れないなら、台帳に行を足す

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `ralph status` と `ralph org status`(table と `--json` の `feature` の object)が、機能の org の共有のパス(`FeatureBinding.Shared`)を出さない。director は台帳の `splits/<split>.md` を読んで代わりにするが、その値は start のときの記録と違うことがある | 走っている org の共有のパスを知る手段が計画のファイルだけで、承認し直したあとは台帳の値と食い違う。`reserved:` の行に共有が出ないので、予約の重なりを数える人が見落としうる | 出力の形(`--json` の field)が増え、この PR の範囲(計画の Non-goals と Progress S4 の記録)の外 | `ralph status` の出力を次に触る変更、または共有のパスの食い違いで director が重なる org を数え間違えた報告 | `docs/plans/active/2026-11-22-split-shared-paths.md` の Progress S4、この report の LOW 7 |

_`docs/tech-debt/README.md` には足していない。ほかのレーンの PR も同じ表の末尾に足すので、並行の merge で衝突しやすい。`/pr` の時点で、この PR が status の表示を入れないと決まっていれば、上の行を足す。_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 8)
- Follow-ups:
  - 直すなら安い順に LOW 4(`--help` の 1 句、この PR に入れるべきもの)、LOW 1(`sameFeature` の doc の前提)、LOW 2(タスクの行の文言と固定している 3 か所)、LOW 7(skill の 1 句)
  - LOW 3 は決定済みの動きで、直すかどうかは人の判断。LOW 5・LOW 8 は読みやすさだけで、直さなくても動きは変わらない。LOW 6 は doc の 1 文で足りる
  - どれを直しても、コードか skill に触れるので記録だけの修正ではなく、`/verify` 以降の全体の再実行が要る
  - 計画の Verify plan にある文書の食い違い(AGENTS.md の `split.go` と `watch_scope.go` の説明、分割計画の形を書く仕様)は `/verify` と `/sync-docs` の担当で、この review では網羅していない
