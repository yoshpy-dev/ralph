# Self-review report: org-merge-order

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、`.harness/state/standard-pipeline/cycle-count.json` は 1)
- Scope: diff の品質だけ。`git diff ac34ccf3...HEAD`(HEAD 06462bcd、6 コミット、16 ファイル、+3557/-66)。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の Design decisions C1〜C8 と Progress checklist の Notes にある決定(7c を 2 つの PR に分ける、単位はブランチ、`gate` を打ち直す手を今は用意せず文書に穴として書く、対象がなければ ls-remote を打たない、など)は、欠陥として数えていない

## Evidence reviewed

- 読んだ差分(コード): `internal/org/mergeorder.go`(681 行、全体)、`internal/cli/org_merge_order.go`(全体)、`internal/cli/org.go`・`org_legacy_ledger.go`・`internal/org/seat.go` の hunk
- 読んだ差分(文書): `/org` skill の hunk(`.agents/` 側と `templates/base/` 側の 2 組は `cmp` で root と同一)、README の 2 か所、AGENTS.md の repo map の行、仕様 FR-8 の注、`docs/tech-debt/README.md` の追加 3 行と変更 1 行(新しい行の `|` は 6 個で、列の数が合う)
- 読んだテスト: `internal/org/mergeorder_test.go` の 1〜730 行(fixture、分類の表、最新の記録、付け先、対象、順)と 1040〜1209 行(何も書かない、JSON の形、`realMergeOrderExec` の 2 本)、`internal/cli/org_merge_order_test.go` の 1〜425 行(fixture、fake gh、出力の読み戻し)と 900〜1012 行(`--state-dir`、書かない・ロックを取らない)。残りは名前の一覧だけを読んだ
- 読んだ周辺のコード(定義と呼び出しを確かめるため): `gate.go` の `gateRecord`・`gateRecordFromDetails`・`resolveGateBase`、`manifest.go` の `readJSONLines`・`ManifestStore.Read`、`reserve.go` の `FeatureBinding.Complete`・`reservationFromEvent`・イベントの絞り込み(434 行目)、`org.go` の `newOrgRuntimeAt`、`org_inbox.go` の `inboxField`、`watch.go` の `realGitStatus`、`gate_run.go` の WaitDelay の扱い
- 確かめたこと
  - `git diff --check` は空。追加した Go の行に TODO・FIXME・デバッグ出力・秘密情報はない
  - `collectMergeTargets` のイベントの絞り込み(`DryRun`・`OrgID`・`SeatID`)は `reserve.go:434` と同じ。`corruptAt` の添字(壊れた行の前にあるイベントの数)と `lastCorrupt > t.gate.pos` の向きは合っている(gate が添字 p、壊れた行が `corruptAt = n` なら、p ≤ n-1 のときだけ壊れた行が後)
  - `classifyMergeTarget` と `classifyGatePassed` の判定の順は、計画の Scope の順(merged・closed、unreadable、no_gate、failed、no_remote、regate、wrong_base、rebase、ready)と同じ
  - `parseLsRemote` は名前がちょうど `HEAD` の行だけを symref と読み、`refs/heads/HEAD` は枝として読む。SHA でない値の `refs/heads/` 行(symbolic ref)は飛ばす
  - 外から来る値(台帳の `Details`、枝の名前、`gh` の URL と `baseRefName`、ls-remote の出力)は、テキストの出力では `inboxField` を通り、JSON では encode される。枝の名前は `gh pr list --head <値>` の引数だけで使い、`--head` が次の引数を値として取るので、`-` で始まる名前もフラグにならない。ls-remote の引数は固定
  - doc コメントの置き場: 新しい 4 ファイルで、`func` の直前のコメントの最初の語が関数名と合わない箇所(extraction で doc が孤立する型)を awk で拾ったが、`Test` や `AC` で始まるものを除いて 0 件。非テストの 2 ファイルで doc のない `func` も 0 件。目視の補助で、網羅の証明ではない
- 確かめていないもの: テスト、`gofmt`、`go vet`、`golangci-lint`、`-race` は走らせていない。実際の `gh` の未認証時の出力と `gh pr list` の `--limit` の既定(30 件)は確かめていない。ssh の passphrase で ls-remote が止まる動きは、tech-debt の行に書かれた説明を読んだだけ

## Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 8。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | maintainability | `Detail` の次の手の言い方がそろっていない。`unreadable`(2 か所)・`regate`・`rebase`(2 か所)は「`ralph org gate` を打ち直す」のあとに「(the gate runs only while the feature org runs)」と足すが、`no_gate` と `failed` は足さない。`no_gate` は PR を出したあと(org は disband 済みで gate は断られる)にいちばん起きる。どの `Detail` も `ralph org gate` に必要な `--org-id` を書かず、`Binding` があるのに使っていない。同じ句が format 文字列に 6 回(5 つの `Detail` と、ready が複数のときの注記)、別々に書かれている | `mergeorder.go:568-577`(unreadable、no_gate、failed)、`:593-606`(regate、rebase)、`:644-646`(ready が複数のときの注記)。`Gate` は `--org-id` を空だと断る(`gate.go:246-247`) | 句を 1 つの定数にして、今ある 6 か所と足す 2 か所で使う。`no_gate` と `failed` にも同じ但し書きを足す(または、org が disband 済みかを台帳から読めるなら、その場合だけ足す)。`Binding` があるときは `ralph org gate --org-id <id>` と書く |
| L2 | LOW | maintainability | `gh` が入っていて全部の枝で同じ理由で失敗する場合(未認証が代表)、枝の数だけ同じ内容の注記が並び、各注記が `gh` の stderr を丸ごと持つ。JSON は `gh_available: true` で、全 entry の `pr` が省かれる。「`gh` が失敗して PR が分からない」と「PR がない」は、entry の側では区別できず、区別は自由文の注記にしかない(8a の director は機械で読む) | `mergeorder.go:449-470`(失敗の分岐が枝ごとに `notes` へ足す)、`:112-126`(`GHAvailable` の doc は「`gh` が見つかった」だけ)。`TestMergeOrder_GHFailureForOneBranchLeavesItsPROut` は 1 本だけの失敗を固める | 失敗した枝の数と最初の理由を 1 つの注記にまとめる。entry の側は、PR の lookup が失敗したことを示す欄(例: `pr_lookup_failed`)を足すか、`GHAvailable` を「`gh` が PR を返せた」に変えて doc も合わせる |
| L3 | LOW | maintainability | `readMergeLedger` は `ManifestStore.Read` の復号のクロージャ(`json.Unmarshal` して `ManifestEvent` に足す)を写している。読み方の道が 2 つになり、`Read` が復号のあとに何かを足したとき merge-order だけ外れる。また `corruptAt` は壊れた行ごとの位置を持つが、読むのは最後の 1 つと数だけ | `mergeorder.go:225-248`、`:331-336`、`manifest.go:104-111` | `ManifestReadResult` に壊れた行の位置(または最後の位置)を足して `Read` を共通にする。足さないなら、`mergeLedger` の `corruptAt` を `corruptCount` と `lastCorruptAt` の 2 つにして、「全部持つ」という doc と読み方を合わせる |
| L4 | LOW | maintainability | `realMergeOrderExec` は、`exec.CommandContext` + `WaitDelay` + 「`ErrWaitDelay` で ctx が生きていれば成功」の組を 3 つ目の写しとして足す。`WaitDelay` の定数も 4 つ目(`gitStatusWaitDelay` 1 秒、`watcherWaitDelay` 1 秒、`gateKillGrace`、今回の 2 秒)になる | `mergeorder.go:657-681`、`watch.go:356-376`、`gate_run.go:193-209`、`watcher.go:80,244` | 今は直さなくてよい。次に外のコマンドを足す PR で、`dir`・`env`・stderr を error に付ける共通の実行関数を 1 つ置き、`realGitStatus` と `realMergeOrderExec` をそこへ寄せる |
| L5 | LOW | naming | SHA を 12 文字に切る関数が 2 つあり、名前が似ていて(`shortMergeSHA`、`mergeOrderShortSHA`)実装が違う。org 側はバイトで切り、`12` を直に書く。CLI 側は rune で切り、定数 `mergeOrderSHALen` を持つ。値は `gitObjectPattern` を通った 16 進だけなので、rune の扱いは要らない。help の「(30 s)」は `defaultMergeOrderLsRemoteTimeout` の写しで、15 秒の `gh` の上限は help にない | `mergeorder.go:624-627`、`org_merge_order.go:123-125,171-178`、`org_merge_order.go:85` | CLI の `mergeOrderShortSHA` を消し、org 層の関数を 1 つ export して使う。help の秒数は定数から組み立てるか、秒数を書かない |
| L6 | LOW | naming | テストの名前の付け方が 2 ファイルでそろっていない。org 側は 2 文字の接頭辞(`moExec`、`moFixture`、`moPR`、`moNoteWith`、`newMOFixture`)で、grep では `mo` が多くの語に当たる。CLI 側は `mergeOrder*`。CLI のテストは package `cli` の最上位に `short` という名前の関数を置く | `mergeorder_test.go:46-298`、`org_merge_order_test.go:346` | `moXxx` を `mergeOrderXxx` にそろえる(機械的な置換)。`short` は `mergeOrderShort` など、grep できる名前にする |
| L7 | LOW | readability | skill の「見張りがなくても使える」の段落の折り返しが半端。追加した 2 行が表示幅で 95 と 86 桁で、同じ hunk の追加行(中央値 71 桁)より広い。4 面とも同じ | `.claude/skills/org/SKILL.md:1103-1104`(`.agents/`、`templates/base/` の 2 面は同じ行) | その段落を 71 桁前後で折り直す。`scripts/sync-skills.sh` で `.agents/` を作り直し、`templates/base/` の 2 面は root と `cmp` で確かめる |
| L8 | LOW | maintainability | コードのコメントが、計画の置き場所 `docs/plans/active/2026-10-14-org-merge-order.md` と、計画の中でしか意味の通らない記号(C2・C3・C5〜C8、AC1〜AC11)を指す。`/pr` の `archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` の links だけなので、4 つの Go ファイルの `active/` は archive のあとに古くなる。同じ形は既存の `approval.go:21`、`gate_test.go:24` にもあり、この repo では許されている | `mergeorder.go:18`、`mergeorder_test.go:24`、`org_merge_order.go:16`、`org_merge_order_test.go:22` | 直さなくてよい。直すなら、パスを `docs/plans/` 以下の slug だけにし(`2026-10-14-org-merge-order`)、記号の意味は読む側が検索で引けるよう、`mergeorder.go` の先頭コメントに C2・C3・C7・C8 を 1 行ずつ書く |

## Positive notes

- 外のコマンドを `MergeOrderExec` の 1 つの seam に通したので、`git` と `gh` の分岐が org 層のテストで fake に替わり、CLI のテストは本物の git と bare の origin で打てている。CLI のテストは `PATH` を一時の bin だけにして、開発者の `gh` に届かないようにしている
- 証拠が読めないときに前の `gate_passed` へ戻らない(C8)ことが、`unreadable` の 2 本の分岐、壊れた行の位置、`TestMergeOrder_LatestGateRecordWins` の 11 行の表で固められている。壊れた行が最後の記録より前なら影響しない、新しい読める記録が後にあれば消える、の両側も表にある
- 出力の escape が一箇所(`printMergeOrder` の `inboxField`)に集まり、`TestOrgMergeOrder_ControlCharactersInLedgerAndGHValuesAreEscaped` が ledger と `gh` の両方の値で確かめている
- `parseLsRemote` が、`HEAD` へのパターンの当たり方(`refs/heads/HEAD`)と symbolic ref の行を明示して飛ばしている。doc コメントがその理由を言っている
- 書かないこと(AC9)を、ロックを他者に握られた read-only の台帳で打ち、manifest のバイト、state dir の listing、work repo の refs が変わらないことで確かめている。root のときは skip する(`os.Geteuid` の分岐)ので、chmod の失敗注入が root で落ちる形ではない
- 変更の範囲が小さい。既存の `org.go`・`org_legacy_ledger.go`・`seat.go` の hunk は、新しい動詞を一覧に足す文言だけで、`org_legacy_ledger.go` の `approve`・`gate` の追加も、`org_approve.go:60` と `org_gate.go:82` が `orgLedgerMutating` を使うことと合っている

## Tech debt identified

この回は報告だけをコミットする指示なので、`docs/tech-debt/README.md` には行を足していない。LOW 8 件のうち、L2(PR の lookup の失敗を entry の側で区別できない)と L4(外のコマンドの実行関数の 4 つ目の写し)は、直さずに進めるなら 1 行にまとめて register に足すことを勧める。L1 は文言の直しで、cycle 1 のうちに直せる。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (足す場合) PR の lookup の失敗が entry の欄に出ない(L2)、外のコマンドの実行関数が 4 つ(L4) | director(8a)が JSON を機械で読むと、`gh` の失敗を PR なしと読み違える。実行関数は直すたびに 4 か所を見る | 文言と欄の足し方で、動きは変わらない。直すと cycle が増える | 8a の director が merge-order の JSON を読む PR、または外のコマンドを足す次の PR | `docs/reports/self-review-2026-10-14-org-merge-order.md`(L2、L4) |

## Recommendation

- Merge: yes(cycle 4 の時点、branch 全体。CRITICAL・HIGH・MEDIUM はない。cycle 4 の差分(`999e0394..HEAD`。gh が読む repository が origin のものと違うときの注記に改名の直し方を並べる直しと、結びつきのない org の読めない gate の記録をその org の前の読める記録の枝に付ける直し)の LOW は 4 件(下の「Cycle 4」の C4-1〜C4-4)で、読む人に当たるのは C4-1 だけ(結びつきのない org の枝の Detail が、`ralph org gate` の断る手を勧める)。cross-review cycle 3 の ACTION_REQUIRED #1(改名した repository で注記の `gh repo set-default` が効かない)は、注記が改名の直し方と `gh repo set-default` を並べて閉じた(改名した repository の PR は、origin の URL を直すまで使わない)。WORTH_CONSIDERING #2(gate の記録だけが名指すブランチで、読めない新しい記録のあとに古い pass で `ready` になる)は、結びつきのない org でもその org の前の読める記録の枝に付けて閉じた。C4-3 のとおり `docs/tech-debt/README.md:239` の上限の言い回しは cycle 4 で偽になり、C4-4 のとおり 237 行の triage への指し方は次の cross-review で古くなる。cycle 4 は最後なので、C4-1・C4-2 と、cycle 3 の C3-1〜C3-4、C2-4〜C2-6・L3・L5・L6 は直さず、239 行に載せる(下の「Cycle 4」の Tech debt identified)。cycle 1〜3 の判定も yes で、cycle 1 の所見の表と「Cycle 2」「Cycle 3」の節はその時点のまま残す)
- Follow-ups:
  1. L1 を直す(`mergeorder.go` の 1 ファイル、定数 1 つと `no_gate`・`failed` の文言)。読んだ範囲のテストは `Detail` を部分一致で見るので文言を足しても通るが、完全一致で固める箇所があれば合わせる(残りのテストは読んでいない)
  2. L2 と L4 は、直さないなら上の 1 行を register に足す(orchestrator)。L5〜L8 は機械的な直しで、直すなら L7 は skill の 4 面を `scripts/sync-skills.sh` と `cmp` で合わせる
  3. 直すとコードか skill の 4 面に触るので、`/verify` から回し直しになる(cycle 2 が要る)。cycle 1 のまま進めるなら、L1〜L8 は PR の本文に known gaps として書く

## Cycle 2

- Date: 2026-10-14
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2。`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 回の最後)
- Scope: `git diff cab2fd50..HEAD`(HEAD 7632bb53、3 コミット、12 ファイル、+890/-69)。15d4b947(gh の PR を origin のものだけ使う。`internal/org/mergeorder.go`、`internal/cli/org_merge_order.go` の help、テスト 3 ファイル)、ee302b06(skill 4 面、README、tech-debt の 2 行)、7632bb53(計画の進捗に 1 行)。cycle 1 の差分は読み直していない。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の進捗にある cycle 2 の決定(`--repo` を渡さない、`git remote get-url origin` を 1 回足す、host は比べない、ローカルのパスの origin は比べない)は、欠陥として数えていない

### 読んだものと確かめたこと

- 読んだ: `git diff cab2fd50..HEAD` の全体(Go 4 ファイル、skill の hunk、README の追加 1 文、tech-debt の追加 2 行、計画の 1 行)。`mergeorder.go` は `MergeOrder`(209-242)、`collectMergeTargets`(300-352)、`lookupMergePRs` から `mergeRepoFromPRURL` まで(460-677)、`lookupMergePR`・`pickMergePR`(679-714)、`classifyMergeTarget`・`mergeGateCommand`・`classifyGatePassed`(770-830)、`mergeOrderListNotes`(862 行付近)、`realMergeOrderExec`。周辺は `internal/cli/org_gate.go` の `requireOrgID`(79 行)、skill の 183・928-933・1104-1117 行
- `git diff --check cab2fd50..HEAD` は空。追加行に TODO・FIXME・デバッグ出力・秘密情報はない。tech-debt の新しい 2 行は `|` が 6 個ずつで 5 列。skill の 3 面(`.agents/`、`templates/base/` の 2 つ)は root と `cmp` で同一
- 確かめたこと(コードを読んで辿った。走らせていない)
  - gate を勧める文の出どころは `mergeGateCommand` と、ready が複数のときの注記(864 行)だけ。`mergeGateRunsWhileOrgRuns` の定義は 101 行の 1 つで、`Detail` 7 つと注記 1 つがそれを使う(777・780・783・786・813・822・825・865 行)。`gate runs only` の直書きは `mergeorder.go` に 0 件
  - gh の失敗をまとめるキーは `err.Error()`。`realMergeOrderExec` が足すのは stderr だけ、`lookupMergePR` が足すのは固定の文だけで、枝の名前は入らない。同じ失敗は 1 つにまとまる。注記の枝の名前・理由・`pr.URL` は、テキストでは `inboxField` を通り、`--json` では encode される。`%q` で URL を囲むので制御文字は注記の中でも escape される
  - origin の URL の読み(`mergeRepoFromOriginURL`)を、テストの表にない形も含めて手で辿った。scp 形、`ssh://…:22/`、末尾の `/` のあとの `.git`、`file://`(ホストなし)、`git+ssh://`(スキームが一覧になく、読めない扱い)、`./a:b/app.git`(コロンの前に `/`)、空文字は、どれも想定どおり読める・読めないに分かれる
  - ctx の扱い: `check` は外側の ctx が終わっていれば interrupted のエラー、`readMergeOriginRepo` の timeout は内側の `getCtx` だけが終わったときの別の分岐で、注記になる。stop の枝で全 target の `pr` を nil に戻すのは、最初の PR の URL が通ったあとで別の PR の URL の形が違うときにだけ効く防御で、通常は到達しない
  - 新しいテストの fake: `moExec` は `git remote get-url origin` を `lsBlock` より先に見るので、ls-remote を止めるテストと干渉しない。CLI のテストの `mergeOrderGitWrapper` は `$#` と 3 つの引数を全部比べるので、`git remote get-url origin` 以外は本物の git に渡る。`t.Setenv("MERGE_ORDER_ORIGIN_URL", "")` を毎回入れるので、前のテストの値は残らない
- 確かめていないもの: テスト、`gofmt`、`go vet`、`golangci-lint`、`-race`。本物の gh が fork の作業場所で upstream を引くこと(コメントとテストの前提)。`git remote get-url` が `insteadOf` を展開すること、ネットワークに触れないこと(git の doc の記述を知っているだけで、走らせてはいない)

### 所見(cycle 2)

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6。cycle 1 の ID(L1〜L8)と取り違えないよう、ID に `C2-` を付ける。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | maintainability | 結びつきがなく gate の記録だけが名指す枝の `Detail` は、`ralph org gate` だけを勧める。`ralph org gate` は `--org-id` なしだと `requireOrgID` で断るのに、その枝の gate の記録には org_id がある。結びつきのない target は gate の記録からしか生まれず、その記録は読めるので、`failed`・`regate`・`rebase`・壊れた行が後にある `unreadable` の 4 つで org_id を言えるのに言っていない。テストは「org がない」と書いて、`--org-id` を出さないことのほうを固定している | `mergeorder.go:793-798`(`t.binding == nil` で `"ralph org gate"`)、`:321`(`ev.OrgID == ""` の記録は読まない。通った記録の org_id は空でない)、`org_gate.go:79`。テストは `mergeorder_test.go:468`(「has no org to name」)と `:508-510`(`--org-id` が出ると赤) | `t.binding == nil` のときは `t.gate.ev.OrgID` を使う(結びつきがないとき `t.gate` は nil でない)。テストの期待は `--org-id org-x` を名指す形にする。結びつきと gate の記録の org が違うときは、今のまま最新の結びつきの org でよい |
| C2-2 | LOW | maintainability | `GHTimeout` の doc は「each gh call」を縛ると言うが、`lookupMergePRs` の `timeout` は `git remote get-url origin` にも渡り、失敗の文も gh の秒数で「timed out after 15s」と言う。新しいテストは `p.GHTimeout = 20 * time.Millisecond` で get-url を止めて、この流用を固定している。また get-url に付ける `GIT_TERMINAL_PROMPT=0` は ls-remote の写しで、`git remote get-url` は設定から URL を出すだけでネットワークにもプロンプトにも触れないので効かない。doc とテストの期待がそれを固定している | `mergeorder.go:120-122`(doc)、`:473-475`(`timeout` は `p.GHTimeout`)、`:605-608`(get-url に `timeout` と env)、`:600-601`(doc の「with GIT_TERMINAL_PROMPT=0」)。テストは `TestMergeOrder_PRIsUsedWhenItsURLNamesOriginsRepository` の `getURL[0].env` の比べと `TestMergeOrder_OriginURLThatCannotBeReadLeavesPRsOut` の `GHTimeout` | `GHTimeout` の doc を「each gh call and the `git remote get-url origin` read with them」にする(または専用の定数を持つ)。env は外し、doc とテストの期待も合わせる |
| C2-3 | LOW | readability | 読む人に見える 3 面と Go のファイルの先頭の説明は、gh の PR を「origin のものだけ使う」と言い切るが、コードは origin の URL から owner/repo を読めない場合(ローカルのパス、`file://`、`git+ssh://` など)は確かめずに使い、host も比べない。この例外は `mergeOriginRepo.check` の doc と tech-debt の行にしかない | help `org_merge_order.go:47-50`、skill `.claude/skills/org/SKILL.md:183` と `:928-933`、README の追加の 1 文、`mergeorder.go:32-37`。コードは `:585`(`if !o.readable`)で素通しにする。例外を書くのは `:565-571` の doc と `docs/tech-debt/README.md:237` | 直すなら help の 1 句だけ(「origin が owner/repo を持たない URL のときは確かめない」)。skill 4 面と README は、tech-debt の行が例外を持つので、直さず known gaps に書く手もある |
| C2-4 | LOW | readability | `lookupMergePRs` は `for _, t := range targets` の中で、また `for _, t := range targets` を書き、外側の `t` を隠す。`check` の戻り値 `stop string` は「空なら使ってよい、空でなければ注記の文」を 1 つの文字列で表し、名前は「止める」だが中身は注記で、呼び元はその文を `append` する | `mergeorder.go:479`(外側)、`:497-498`(内側)、`:492`(`stop, err := origin.check(...)`)、`:572`(戻り値の名前) | 内側を `clearMergePRs(targets)` のような小さい関数に出すか、変数名を変える。`check` の戻りは `note string` に改める(空なら使ってよい、と doc に書く) |
| C2-5 | LOW | readability | L7 の折り返しの直しが途中までで、短い行が 1 本残った。直した段落は 64〜73 桁で組まれているが、その直後の 1 行だけ 48 桁で、次の行(元の折り返し)は 70 桁。4 面とも同じ | `.claude/skills/org/SKILL.md:1112`(「通知は typed `ALERT` として各 org の leader に」48 桁)、`:1111`(71 桁)、`:1113`(70 桁)。`.agents/` と `templates/base/` の 2 面は同じ行 | 1112-1113 を同じ幅に組み直す。動きには関係しない。直すなら `scripts/sync-skills.sh` で `.agents/` を作り直し、`templates/base/` の 2 面は root と `cmp` で確かめる |
| C2-6 | LOW | readability | 新しいテストの読みにくさが 3 つ。(a) `t.Run(origin, ...)` の名前が URL そのもので `/` を含み、`go test -run` は名前の中の `/` も階層の区切りに数えるので、1 件だけを選びにくい。(b) `map[bool]string{true: repo.String(), false: ""}[ok]` という言い方が表の 2 か所にあり、`ok` が false でも `repo.String()` を評価する。(c) コメントが「the cross-review finding」「self-review L2」と、slug も cycle もなく報告の中の名前だけで指す。triage の報告は cycle ごとに書き換わる | (a)`mergeorder_test.go:911`、`:1023`、(b)`:1065`、`:1083`、(c)`mergeorder_test.go:943`、`:1241`、`org_merge_order_test.go:774` | (a)名前を `scp_like`、`https` のような短い名前にし、URL は期待の側の欄に持つ。(b)`if ok { got = repo.String() }` にする。(c)欠陥の中身を 1 文で書くか、`of org-merge-order` を添える(org-gate の cycle 2 の C2-1 と同じ形) |

#### C2-1 の根拠

- `collectMergeTargets` は `ev.DryRun || ev.OrgID == "" || ev.SeatID != ""` の事象を飛ばすので(`mergeorder.go:321`)、gate の事象が target に付くときは org_id が入っている。結びつきのない target は、読める gate の記録が `branch=` で名指したときにだけできる(読めない記録は、その org の最新の結びつきの枝に付くので、結びつきがある)。なので `t.binding == nil` なら `t.gate` は nil でなく、`t.gate.ev.OrgID` が使える
- 起きるのは、gate が記録した枝の名前が、結びつきの枝の名前と違うとき(worktree で枝を作り直した、など)だと読んだ。おそらくまれ。実例は未確認です。当たると、`ralph org gate` を打った人は `--org-id` が要るという断りを 1 度受ける。次の手は 1 つに決まるので LOW にした。cycle 1 の L1 の推奨(「`Binding` があるときは `--org-id` を書く」)は、結びつきがない場合を書いておらず、直しはその字面どおりに入っている

#### C2-3 の根拠

- 「origin のものだけ」という言い方は、gh が PR を返した、origin の URL に owner/repo がある、の 2 つが成り立つときの話。成り立たない 2 つめの場合(`file://` の origin に GitHub の `upstream` remote が付いている fork の作業場所など)は、gh は upstream の PR を返し、それが確かめずに使われる。tech-debt の行は「skips the check and uses gh's PRs as they are」と書いていて、実害の見積もりはそちらにある
- 4 つの skill の面を直すと `/verify` から回し直しになる(cycle の上限に当たる)ので、help の 1 句を除いて register の行に任せるのが妥当と読んだ

### cycle 1 の所見を HEAD で見直した結果

| ID | 状態 | 確かめた場所 |
| --- | --- | --- |
| L1 | 直った(1 件は C2-1) | `mergeGateRunsWhileOrgRuns` を 8 か所(`Detail` 7 つと注記 1 つ)が使う。`no_gate`・`failed` にも但し書きが付き、結びつきがあれば `--org-id` が付く。結びつきのない枝だけが C2-1 |
| L2 | 半分 | 同じ失敗を 1 つの注記にまとめた(`mergeGHFailures`、10 本・3 つの理由のテスト)。entry 側の印(`pr_lookup_failed` のような欄)は `docs/tech-debt/README.md:238` の行に載った |
| L3 | 未着手 | `readMergeLedger` は `Read` の復号を写したまま。register に行がない |
| L4 | 行に載った | `docs/tech-debt/README.md:238`(実行関数の 3 つ目の写し) |
| L5 | 未着手 | `shortMergeSHA`(`mergeorder.go:844`)と `mergeOrderShortSHA`(`org_merge_order.go:176`)。register に行がない |
| L6 | 未着手 | `mo*` の名前と、CLI のテストの `short`(`org_merge_order_test.go:360`)。register に行がない |
| L7 | 半分 | 折り返しは直ったが、1 行が短く残った(C2-5) |
| L8 | 触れていない | cycle 1 の推奨どおり、直さない |

### cross-review の ACTION_REQUIRED #1 は閉じているか

- 閉じた経路: gh が PR を返し、その URL の owner/repo が origin のものと違う(大文字小文字は無視する)、URL が owner/repo を持たない、`git remote get-url origin` が失敗する、の 3 つ。どれも全 target の PR を捨て、注記を 1 つ出し、gh をそれ以上打たない。違いの注記は `gh repo set-default <origin の owner/repo>` を言う。fake の 5 通りのテストと、本物の git と fake の gh の往復(`TestOrgMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut`)が、feat/a が merged として隠れないことと、origin の PR に戻すと隠れることの両側を固定している
- 残る経路は `docs/tech-debt/README.md:237` が 3 つ書いている。gh が別の repository を読み、そこにその枝名の PR がない(空の結果は origin に PR がないのと区別できず、注記も出ない)、host の違い、origin の URL が owner/repo を持たない。読んだ範囲で、この行が言っていない経路は見つからなかった(fork だけの PR が返る場合は `pr == nil` で確かめに入らないが、「PR がない」と同じ結果なので 1 つめに含まれる)

### 所見にしなかったもの

- cycle 2 で足した 3 つの定数のうち `mergeGateRunsWhileOrgRuns` だけが、同じ const 群の `mergeOrder*` の接頭辞を持たない。`grep mergeOrder` で拾えないが、`grep mergeGateRuns` で 1 つに当たり、定義も 1 か所なので、名前の揺れとして数えない
- `lookupMergePRs` が gh の失敗のまとめ、origin の確認、gh の有無の 3 つを 1 つの loop で持つ。約 40 行で、分岐は `switch` の 4 つ。コメントが各分岐の意味を言っているので、分けるほどではない
- `mergeGHFailures` は添字の付け方が値と pointer の受け手の混在(`add` は pointer、`notes` は値)。slice の型で `add` が追加するので pointer が要り、`notes` は読むだけ。混在の理由は読み取れる
- コードのコメントが計画の `active/` の置き場や記号を指す件は、cycle 1 の L8 と同じで、repo が許している形。再掲しない

### Tech debt identified

この回は報告だけをコミットする指示なので、`docs/tech-debt/README.md` には行を足していない。cycle 2 は上限の最後で、直すと 3 回目の pipeline が要る。C2-1〜C2-6 と、cycle 1 から残る L3・L5・L6 は、次の 1 行にまとめて足すことを勧める(orchestrator が足す)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small diff-quality leftovers in the org-merge-order PR after self-review cycle 2 (all LOW): (1) a Detail for a branch only a gate record names says `ralph org gate` without the `--org-id` that record carries (C2-1, `mergeGateCommand`); (2) `GHTimeout`'s doc says "each gh call" but it also bounds `git remote get-url origin`, which also gets a no-op `GIT_TERMINAL_PROMPT=0` (C2-2); (3) the help, skill and README say gh's PRs are used "only when" they are origin's, without the local-path exemption (C2-3); (4) a shadowed `t` and a `stop string` result in `lookupMergePRs` / `mergeOriginRepo.check` (C2-4); (5) one 48-column orphan line in the skill's watch paragraph, four copies (C2-5); (6) test subtests named by URL, a `map[bool]string` idiom, bare review labels in test comments (C2-6); from cycle 1: `readMergeLedger` copies `ManifestStore.Read`'s decode closure (L3), two SHA-shortening helpers and a help text that hardcodes "30 s" (L5), `mo*` test names and a package-level test helper `short` (L6) | (1) is the only one a reader can hit: for a branch whose binding is not in the ledger, the Detail tells them to run a command the CLI refuses once (`--org-id` is required). The rest change no behavior or output | Found at the pipeline cap (2/2); fixing needs a third run, and each item is a few lines | The next edit to `internal/org/mergeorder.go` or `internal/cli/org_merge_order.go`, stage 8a's director reading merge-order, or a report that the Detail misled someone | `docs/reports/self-review-2026-10-14-org-merge-order.md` (Cycle 2, C2-1 to C2-6; L3, L5, L6) |

### Recommendation

- Merge: 上の「Recommendation」の Merge の行のとおり(yes)。cycle 2 の差分に CRITICAL・HIGH・MEDIUM はなく、cross-review の指摘は gh が PR を返す経路で閉じ、残る経路は tech-debt に書かれている
- Follow-ups:
  1. 上の tech-debt の 1 行を足す(上限なので、C2-1〜C2-6 は直さない)。3 回目を回す理由が別にあるなら、C2-1(`mergeGateCommand` の 1 関数とテストの期待 1 本)と C2-2(doc と env)を先に直す
  2. cycle 2 の `/verify` と `/test` が書き直す報告に、cycle 1 の時点の `mergeorder.go` の行番号を指す pointer が残っていないか見る。`lookupMergePRs` から後ろは cycle 2 で 100 行以上動いた(例: `classifyMergeTarget` は 556 行から 763 行へ)。関数名で書くか、HEAD で引き直す
  3. insight event(`./scripts/insights-append.sh --phase self_review --cycle auto`)は、報告だけをコミットする指示なので、この回では足していない。件数は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6(cycle 2 の分)で、orchestrator が足す

## Cycle 3

- Date: 2026-10-14
- Reviewer: reviewer subagent (Claude)。パイプライン 3 回目(cycle 3。cross-review cycle 2 の ACTION_REQUIRED を直すために上限を 2 から 3 に上げた回で、最後。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)
- Scope: `git diff e463eb9d..HEAD`(HEAD c885fcab、3 コミット、9 ファイル、+245/-45)。984a163e(計画の進捗に 1 行)、d7599d82(`gh pr list` に `--limit 1000`、上限ぶんの fork の PR だけが返ったときの注記、`mergeNamedBranches` の切り出し、help と skill 4 面、テスト)、c885fcab(C2-1 の `mergeGateCommand`、C2-2 の `GHTimeout` の doc)。cycle 1・2 の差分は読み直していない。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の進捗にある cycle 3 の決定(`--limit 1000` を渡す、返った数が上限と同じで fork でない PR が 1 本もないときは PR を使わず注記する)は、欠陥として数えていない

### 読んだものと確かめたこと

- 読んだ: `git diff e463eb9d..HEAD` の全体(Go 4 ファイル、skill の hunk 1 行 × 4 面、計画の 1 行)。`mergeorder.go` は定数と `MergeOrderParams`、`MergeOrderEntry` の doc、`lookupMergePRs` から `mergeNamedBranches` まで、`mergeOriginRepo.check`・`readMergeOriginRepo`、`lookupMergePR`・`pickMergePR`、`mergeEntryFor` から `classifyGatePassed`、`mergeOrderListNotes`、`realMergeOrderExec`。テストは `TestMergeOrder_DetailsNameTheGateCommandWithTheOrg` と新しい `TestMergeOrder_GHLimitFullOfForksLeavesThePROut`、`moNoteWith`・`moBindTo`。周辺は `internal/cli/org_gate.go` の `requireOrgID`、help の全文、tech-debt の 233〜239 行
- `git diff e463eb9d..HEAD --check` は空。追加行に TODO・FIXME・デバッグ出力・秘密情報はない。skill 4 面(root、`.agents/`、`templates/base/` の 2 つ)は `md5` が同じ。引用した行番号は c885fcab で引き直した
- 確かめたこと(コードを読んで辿った。走らせていない)
  - 上限の判定 `pr == nil && len(prs) >= mergeOrderGHLimit`: `pickMergePR` が nil を返すのは、PR がないか、全部 fork のときだけ。なので「上限ぶん返り、fork でない PR が 1 本もない」と読める。fork でない PR が上限の中に 1 本でもあれば、番号が最大のものを採る。gh が新しい順に返すなら、上限より古い PR にそれより大きい番号はない(順序は gh の挙動の読みで、未確認)
  - `lookupMergePRs` の `switch` の順は、ctx 終了、gh なし、失敗、上限、PR なし。上限の枝は `err == nil` のときだけ届き、`origin.check` の前で `continue` する。PR を使わないので、repository の確認に入らない。`withNotes` は閉包で、`failed` と `limitFull` を呼ぶ時点の値で読む。return の 3 か所(gh なし、stop、最後)がすべて通り、注記の並びは gh の失敗、上限、終わりの注記
  - `mergeNamedBranches` の切り出しは元の 4 行と同じ(`min(len, 5)` と ` and %d more`)。`mergeGHFailures.notes` の文は変わらない。doc コメントの直後に関数があり、前の関数の doc を切り離していない(テスト側も同じ)
  - 新しいテストは 4 つの subtest(1 本が上限、複数の枝が上限で注記は 1 つ、fork でない PR が 1 本混じる、混じった PR の repository の確認)。境界は、上限ちょうど(注記あり)と 1 本少ない(注記なし)の対と、fork でない PR が 1 本混じる場合(注記なし、その PR を使う)で、`>=` を `>` に替える、`pr == nil &&` を外す、上限の枝を消す、のどれにも赤くなる並びに読めた。argv の `--limit 1000` は org 層の `TestMergeOrder_RunsGitAndGHWithTheirArgsDirAndEnv` と CLI の `mergeOrderGHCall` の 2 か所で固定している
  - `mergeGateCommand` の 3 つめの枝(gate の記録があり org_id が空)は、`collectMergeTargets` が org_id のない記録を付けないので、手で作った target でしか届かない。doc とテストがそれを書いて固定しているので数えない。結びつきと gate の記録の org が違う枝は、新しい `twice` が最新の結びつきの org を固定している
  - `realMergeOrderExec` は stdout を `bytes.Buffer` に上限なしで受けるので、1000 件(数百 KB の JSON)でも切れず、`its output is not a JSON list of PRs` にはならない
  - 新しいコメントが名指す識別子(`mergeOrderGHLimit`、`mergeNamedBranches`、`mergeGHLimitNotes`、`mergeGateCommand`、`readMergeOriginRepo`)は、どれも定義がある
- 確かめていないもの: テスト、`gofmt`、`go vet`、`golangci-lint`、`-race`。`gh pr list` が `--head` だけのときに、作成の新しい順に 1 ページ 100 件で辿ること(`mergeOrderGHLimit` のコメント、計画の cycle 3 の行、コミットメッセージの前提。gh のソースは読んでいない。`gh pr list --help` だけ走らせ、`--limit` の既定が 30 であることと、`--head` が `<owner>:<branch>` を受けないことを見た。API には触れていない)。1000 件を 15 秒(`GHTimeout`)で取り切れるか。取り切れなければ、その枝は timed out の注記になり、結果は PR なしと同じ

### 所見(cycle 3)

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5。cycle 1(L)・cycle 2(C2-)の ID と取り違えないよう、ID に `C3-` を付ける。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C3-1 | LOW | maintainability | C2-2 の直しは doc だけで、`git remote get-url origin` に付ける効かない `GIT_TERMINAL_PROMPT=0` が残った。get-url は設定から URL を出すだけでネットワークにもプロンプトにも触れない(cycle 2 と同じ読みで、走らせていない)ので、この env は ls-remote の写しで、`readMergeOriginRepo` の doc とテストの期待が固定している。c885fcab のメッセージは doc だけを直したと言っていて偽ではないが、C2-2 の推奨の後ろ半分(env を外し、doc とテストの期待も合わせる)を追う場所は、tech-debt の 239 行の (b) だけになった | `mergeorder.go:653`(get-url の `[]string{"GIT_TERMINAL_PROMPT=0"}`)、`:645-646`(doc の「with GIT_TERMINAL_PROMPT=0 and timeout」)、`mergeorder_test.go:951-952`(`getURL[0].env` の比べ)、`:917`(テストの doc)。直した場所は `mergeorder.go:84-86` と `MergeOrderParams` の `GHTimeout` の doc | env を外し、doc とテストの比べを合わせる(3 か所、数行)。直さないなら 239 行の (b) を env だけに狭める(C3-5) |
| C3-2 | LOW | maintainability | ready が 2 本以上のときの注記が「run ralph org gate again」と書き、`--org-id` を付けない。c885fcab の `mergeGateCommand` の doc は「ralph org gate requires」の `--org-id` と言い切るのに、gate を勧める文のうちこの 1 つだけが裸のまま残り、C2-1 と同じく CLI が 1 度断る。注記は複数の枝(複数の org)を指すので、1 つの org_id は書けないが、各行の org(ORG 列)を見るよう書ける。テストは裸の文を末尾の一致で固定している。cycle 2 の確認は gate を勧める文の出どころを数えただけで、この注記に `--org-id` がないことは数えなかった | `mergeorder.go:918-920`(`mergeOrderListNotes`)、`mergeorder_test.go:515`(`HasSuffix "run ralph org gate again"+moGateClause`)、`internal/cli/org_gate.go:79`(`requireOrgID`)。help と skill はこの文を引用していない(`gate again` の grep で、`ralph org gate again` と書くのは `mergeorder.go:919` とそのテストだけ) | 文を「run ralph org gate --org-id <its org> again」の形にして、`:515` の期待を合わせる(1 ファイルとテスト 1 か所) |
| C3-3 | LOW | maintainability | 上限の値 1000 が手書きで散らばる。help に 2 か所、skill に 2 か所(4 面)、CLI のテストの `mergeOrderGHCall`、org 層のテストの期待。定数 `mergeOrderGHLimit` は org パッケージの非公開で、CLI の help は参照できない。定数を変えると CLI のテストは argv の不一致で赤くなるが、help と skill は赤くならない。239 行の (g) が「30 s」の手書きで挙げた形と同じ | `internal/cli/org_merge_order.go:45`・`:47`、`.claude/skills/org/SKILL.md:183`(「`--limit 1000`」と「返った 1000 本」)、`internal/cli/org_merge_order_test.go:81`、`internal/org/mergeorder.go:103` | 直さず、239 行の (g) に 1000 を足す。直すなら skill の 4 面に触れるので `/verify` から回し直しになる |
| C3-4 | LOW | readability | 言葉と名前の小さな点が 4 つ。(a) 定数の説明「bound the ls-remote, and each gh call and the get-url, when …」は、2 つの定数と 3 つの対象の対応が並びでしか分からない。`MergeOrderParams` の doc(「LsRemoteTimeout bounds the ls-remote; GHTimeout bounds each gh call and the …」)のほうが明確。(b) `mergeGHLimitNotes` は複数形で、0 件か 1 件の slice を返し、doc は「the note」と単数。(c) `withNotes` は注記を足すのでなく、最後に返す注記の並び全部を作る。(d) `MergeOrderEntry.PR` の doc は PR が nil になる場合を並べるが、上限ぶんの fork の PR で捨てた場合を言わない(「found none from this repository」が一番近い) | `mergeorder.go:84-86`、`:564-568`(doc と関数)、`:494`、`:168-170` | (a) は `MergeOrderParams` の doc と同じ言い方に。(b)(c) は名前を `mergeGHLimitNote`(string を返す)と `allNotes` にするか、今のままにする。(d) は 1 句足す。どれも動きは変わらない |
| C3-5 | LOW | maintainability | `docs/tech-debt/README.md` の最後の merge-order の行(239 行)が、この cycle の直しで古くなった。(a) と (k) は直り、(b) は半分(C3-1)。見出しと Why deferred の「at the pipeline cap (2/2) … needs a third pipeline run」は、上限を 3 に上げて 3 回目が走ったので偽。Impact の「Only (a) reaches a reader」は、(a) が直って、読む人に当たるものが C3-2 に替わった。直す場所は下の「register の 239 行の古くなった所」 | `docs/tech-debt/README.md:239`。c885fcab の差分(`mergeGateCommand` とテスト)、`mergeorder_test.go:519-526`(`twice`) | `/sync-docs` が下の表のとおり直す(この回は直さない) |

#### C3-2 の根拠

- gate を勧める文は、`mergeGateCommand` を通る 7 つの `Detail` と、`mergeOrderListNotes` の注記の 1 つ。後者だけが `ralph org gate` を直に書く。`mergeorder.go` の中のほかの `ralph org gate` は、「gate が記録する」と言う説明の文(232 行と 378 行)で、命令ではない
- 起きるのは、ready が 2 本以上のとき。注記を読んで、そのまま `ralph org gate` を打つと、`--org-id` が要るという断りを 1 度受ける。次の手は決まるので LOW にした。実例は未確認です

### cycle 2 の所見を HEAD で見直した結果

| ID | 状態 | 確かめた場所 |
| --- | --- | --- |
| C2-1 | 直った(同じ形の 1 件が C3-2) | `mergeGateCommand` が最新の結びつき、なければ最新の gate の記録の org_id を使う。`Detail` 7 つの出どころはこの関数だけ。`twice` が「結びつきの org と gate の記録の org が違う」枝で最新の結びつきの org を固定する |
| C2-2 | 半分(C3-1) | `GHTimeout` の doc と定数の説明は直った。get-url の `GIT_TERMINAL_PROMPT=0` と、その doc、テストの期待は残る |
| C2-3 | 直った(cycle 2 の `/sync-docs`) | help と skill と `mergeorder.go` の先頭の説明に、3 つの例外が書いてある |
| C2-4 | 未着手 | `lookupMergePRs` の内側の `for _, t := range targets` が外の `t` を隠したまま。`check` の戻りは `stop string` のまま。239 行の (c) |
| C2-5 | 未着手 | この cycle の skill の hunk は 183 行だけで、watch の段落は触れていない。239 行の (d) |
| C2-6 | 未着手 | (a)(b) は cycle 2 のテストで、この cycle は触れていない。この cycle の新しいテストは、subtest の名前を言葉にし、`map[bool]string` を使わず、コメントに slug を書いた(「covers cross-review cycle 2 of org-merge-order」)ので、同じ形を足していない。239 行の (e) |
| L3・L5・L6 | 未着手 | 239 行の (f)(g)(h)。L5 は C3-3 で、手書きの数がもう 1 つ増える |

### cross-review cycle 2 の ACTION_REQUIRED #1 は閉じているか

- 閉じた。指摘は「`gh pr list` の既定の上限 30 を、同じ名前の枝の新しい fork の PR が埋めると、本物の PR が結果から落ちる」こと。直しは 2 段で、(1) `--limit 1000` を渡す(定数。argv はテスト 2 か所が固定)、(2) 上限ぶん返って全部 fork のときは PR を使わず、枝の名前を挙げた注記を 1 つ出す(複数の枝は 1 つにまとめる)
- 残る経路: 本物の PR より新しい fork の PR が 1000 本を超えると、(2) の注記になり、PR は使われない。別の PR を本物と取り違えて使うことはなく、結果は「PR なし」と同じで、merged と closed を見分けないことを注記が言う。そのとき `--json` の entry には印がなく、注記だけで知らせる。これは 238 行の行(entry 側に lookup の結果がない)と同じ形で、C3-5 で足す
- 上限の中に fork でない PR が 1 本でもあれば、それを使い、その PR の repository の確認も走る(テストが固定)

### register の 239 行の古くなった所

| 場所 | 今の状態 | 直し方 |
| --- | --- | --- |
| 先頭の「found by self-review cycle 2 and test cycle 2 at the pipeline cap (2/2) and left in place」 | 上限は 3 に上がり、cycle 3 が (a)、(b) の doc の半分、(k) を直した | cycle 3 で直したものを外した、と書く |
| (a) | c885fcab で直った。`TestMergeOrder_DetailsNameTheGateCommandWithTheOrg` は `--org-id org-x` を名指す形に変わった | (a) だけを `~~...~~ (RESOLVED c885fcab)` にするか外す |
| (b) | doc は直った。get-url の `GIT_TERMINAL_PROMPT=0` は残り、`TestMergeOrder_PRIsUsedWhenItsURLNamesOriginsRepository` が固定している(C3-1) | env だけに狭める |
| (k) | `twice` が、結びつきの org と gate の記録の org が違う枝で最新の結びつきの org を固定する。結びつきと gate の記録の入れ替えは赤くなる | 外す |
| Impact の「Only (a) reaches a reader」「(k) would name the wrong org」 | (a)(k) は直り、読む人に当たるのは C3-2 の注記 | C3-2 に替える |
| Why deferred の「Found at the pipeline cap (2/2): fixing any of them needs a third pipeline run」 | 3 回目が走った。残りは最後の cycle(3/3)で直さず、直すと 4 回目が要る | 「cycle 3 が最後で、直すと 4 回目が要る」に |
| Trigger の「a report that a Detail sent someone to a refused command」と、`mergeGateCommand` | Detail の経路は直り、注記が残る(C3-2) | 注記と `mergeOrderListNotes` に替える |
| 足すもの | C3-1 の env(上の (b))、C3-2、C3-3(238 行でなく (g) へ)、C3-4。238 行の「gh failed for this branch ... from gh found no PR」に、上限ぶんの fork の PR で捨てた枝を 1 つ足す。Related に「Cycle 3: C3-1 to C3-5」を足す | 下の Tech debt identified |

### 所見にしなかったもの

- `mergeGateCommand` の 3 つめの枝は、手で作った target でしか届かない。doc とテストの両方がそう書いて固定しているので、防御として数えない
- 新しいテストの doc の「covers cross-review cycle 2 of org-merge-order」は、欠陥の中身を同じ段落で言い、slug も添えている。triage の報告が cycle 3 で書き換わっても読める(C2-6 の (c) の推奨のとおり)
- `lookupMergePR` の戻りが 3 つ(`pr`・`limitFullOfForks`・`error`)になった。呼び元は 1 か所で、`switch` の中の順(エラー、上限、PR なし)が読める。構造体にするほどではない
- 上限の枝が repository の確認より前で `continue` すること。PR を使わないので、確認する相手がいない
- stop の枝(gh が別の repository を読んでいた)でも、それまでに溜まった上限の注記が残ること。gh の失敗の注記が同じ扱いで、元の `append(failed.notes(), stop)` の形を保っている
- コードのコメントが計画の `active/` の置き場や記号を指す件は、cycle 1 の L8 と同じで、repo が許している形。再掲しない

### Tech debt identified

この回は報告だけをコミットする指示なので、`docs/tech-debt/README.md` には行を足していない。cycle 3 は最後(上限 3/3 で、直すと 4 回目が要る)なので、C3-1〜C3-4 は直さない。新しい行は作らず、239 行を上の表のとおり直して、そこに次の項目を足すことを勧める(`/sync-docs` が行う)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Add to the org-merge-order leftovers row (line 239) and remove its (a) and (k), which cycle 3 fixed: (o) `mergeOrderListNotes`'s several-ready note tells the reader to run `ralph org gate again` with no `--org-id`, which `ralph org gate` requires (C3-2); (p) `readMergeOriginRepo` still passes a no-op `GIT_TERMINAL_PROMPT=0` to `git remote get-url origin`, in its doc and pinned by a test (what is left of (b), C3-1); (q) wording and naming: the timeout constants' comment, the plural `mergeGHLimitNotes` that returns at most one note, `withNotes`, and `MergeOrderEntry.PR`'s doc, which does not name the case where gh returned its limit of fork PRs (C3-4); and extend (g): the gh limit `1000` is written by hand in the help, the `/org` skill (four copies) and the CLI test while `mergeOrderGHLimit` is unexported (C3-3). Extend the row about `--json` and the lookup outcome (line 238): a branch whose PRs were dropped because gh returned its limit of fork PRs also leaves out `pr` with only a note | (o) is the only one a reader can hit: the note names a command the CLI refuses once, and the next try with `--org-id` from the ORG column works. (p) to (q) change no behavior or output | Cycle 3 is the last cycle (3/3): fixing any of them needs a fourth pipeline run, and each is a few lines | The next edit to `internal/org/mergeorder.go` or `internal/cli/org_merge_order.go`, stage 8a's director reading merge-order, or a report that the several-ready note misled someone | `docs/reports/self-review-2026-10-14-org-merge-order.md` (Cycle 3, C3-1 to C3-5) |

### Recommendation

- cycle 3 の判定: 上の「Recommendation」の Merge の行のとおり(yes)。cycle 3 の差分に CRITICAL・HIGH・MEDIUM はなく、cross-review cycle 2 の指摘は閉じた
- Follow-ups:
  1. `/sync-docs` が 239 行と 238 行を上の表のとおり直す。C3-1〜C3-4 は最後の cycle なので直さず、その行に載せる
  2. 4 回目を回す理由が別にあるなら、C3-2(文 1 つとテストの期待 1 か所)と C3-1(env、doc、テストの比べの 3 か所)を先に直す。どちらも skill には触れない
  3. insight event(`./scripts/insights-append.sh --phase self_review --cycle auto`)は、報告だけをコミットする指示なので、この回では足していない。件数は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5(cycle 3 の分)で、orchestrator が足す

## Cycle 4

- Date: 2026-10-14
- Reviewer: reviewer subagent (Claude)。パイプライン 4 回目(cycle 4。cross-review cycle 3 の ACTION_REQUIRED 1 件と WORTH_CONSIDERING 1 件を直すために上限を 3 から 4 に上げた回で、最後。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)
- Scope: `git diff 999e0394..HEAD`(HEAD 1052a12a、4 コミット、13 ファイル、+244/-58)。546e9f66(cross-review cycle 3 の triage と、その insight event の 1 行)、fd1f3f30(計画の進捗に 1 行)、813d6d2d(`mergeOriginRepo.check` の注記に `git remote set-url origin` を並べる。help、skill 4 面、README、tech-debt の 237 行、テスト)、1052a12a(`collectMergeTargets` に `gatedBranch` を足し、読めない記録を結びつきの枝、なければその org の前の読める記録の枝に付ける。注記の文、テスト)。cycle 1〜3 の差分は読み直していない。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の進捗にある cycle 4 の決定(改名と別の repository を見分けず、注記に 2 つの直し方を並べる。正規の名前を gh で引かない。読めない記録は、結びつき、その org の前の読める記録、の順に付け、どちらもなければ付けない。後半は計画の Scope と C8 を広げる逸脱)は、欠陥として数えていない

### 読んだものと確かめたこと

- 読んだ: `git diff 999e0394..HEAD` の Go 4 ファイル(`internal/org/mergeorder.go`、`mergeorder_test.go`、`internal/cli/org_merge_order.go`、`org_merge_order_test.go`)の全体、skill の hunk 2 か所(表の 183 行と「見張りがなくても使える」の段落)、README の 262 行、tech-debt の 237 行、計画の進捗の 1 行、triage の差分(上の section の全文)、insight event の 1 行。周辺は `collectMergeTargets` 全体(`mergeorder.go:325-378`)、`mergeLedgerNotes`、`mergeOriginRepo.check`、`classifyMergeTarget`・`mergeGateCommand`、`gateRecordFromDetails`(`gate.go:172-200`)、`FeatureBinding.Complete`(`reserve.go:108-110`)、`gateOrgLife`(`gate.go:333-364`)、テストの helper(`moGarbled`・`moBind`・`moRepoCheckFixture`)、`mergeOrderNoPRs` の定義、register の 232〜239 行
- 確かめたこと(コードを読んで辿った。走らせていない)
  - `git diff 999e0394..HEAD --check` は空。追加行に TODO・FIXME・デバッグ出力・秘密情報はない。HEAD は 1052a12a のまま、`git status --porcelain` は読み始めに空で、コミットの直前にもう一度確かめる
  - skill 4 面(root、`.agents/`、`templates/base/` の 2 面)は `cmp` で同一。追加した行の幅は East Asian Width で 66〜72 桁(段落の最後の行を除く)で、同じ段落の既存の行(68〜75)と合い、cycle 1 の L7 の形ではない。help の追加行は 67〜79 文字(タブと引用符を含む)で、1 行だけ 67 と短いが、次の語を足すと 72 桁になり既存の最長 71 を超えるので、折り直しは要らない
  - `collectMergeTargets` の「どちらの map も "" を持たない」というコメントは本当。結びつきは `f.Complete()` を通ったものだけが `boundBranch` に入り(`Complete` は `Branch != ""` を要る)、読める記録は `gateRecordFromDetails` が `r.Branch == ""` を断る。なので `cmp.Or` が「なし」を空文字で見分けてよい。`gatedBranch` は読める記録のときだけ更新し、読めない記録が付いた先では更新しない
  - 新しい 2 つのテストは、次の変更でそれぞれ赤くなる並びに読めた。`gatedBranch` の代入を外す(org-c の feat/c が ready に戻る)、「最新の読める記録」を「最初の」に替える(feat/e が unreadable、feat/e2 が ready になる)、`cmp.Or` の引数を入れ替える(「結びつきが先」「結びつきが後」のどちらの subtest も feat/x に付く)。注記の件数は org-d の 1 件だけを `HasPrefix` で固定する
  - 改名の注記は、org 層が全文一致(`TestMergeOrder_RepositoryNoteNamesTheFixForARenameAndForAnotherRepository` の 2 subtest)、CLI が 2 つの部分一致で固定する。`renamed to` の 2 つの repository を入れ替えると赤くなる。subtest の名前に `/` はない
  - `set-default` を勧める場所を `grep` で数えた(`docs/evidence/` と過去の報告は記録なので除く)。Go 非テストは `mergeorder.go` の先頭コメント(勧めない)・`check` の doc と本文・help、skill は 4 面の各 2 か所(表と段落)、README 1 か所、register 237 行。直していない場所はない
  - 注記の文と help・skill・README・register は、同じ向きで言っている(改名なら `git remote set-url origin`、そうでなければ `gh repo set-default <origin の owner/repo>`)
  - cycle 3 の所見のうちこの cycle が触ったのは C3-5 の 237 行だけで、C3-1〜C3-4 は未着手(下の表)
- 確かめていないもの: テスト、`gofmt`、`go vet`、`golangci-lint`、`-race` は走らせていない。改名した本物の repository での挙動(GitHub の転送で git が古い URL から届くこと、`gh pr list` の PR の URL が新しい名前になること)は確かめていない。triage と consult の前提を読んだだけで、/test が本物の repository で確かめるとよい

### 所見(cycle 4)

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。計画の設計決定 C1〜C8 や cycle 3 の `C3-n` と取り違えないよう、ID に `C4-` を付ける(計画の「C4」は ID の形が違う)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C4-1 | LOW | maintainability | cycle 4 で、結びつきのない org の読めない記録が、その org の前の読める記録の枝に付くようになった。その枝は `unreadable` になり、Detail は「run ralph org gate --org-id org-c again in the feature worktree (the gate runs only while the feature org runs)」と言う。この道に来るのは org が完全な結びつきを持たないときだけで、そのとき `ralph org gate` は `gateOrgLife` で断る(結びつきなしは「holds no split plan feature binding」、不完全なら「disband it and start it again」)。org が走っていても通らず、但し書きの「while the feature org runs」は disband 済みの穴だけを言う。新しいテストはこのコマンドの文言を固定する。gate の記録だけが名指す枝の `failed`・`regate`・`rebase` の Detail(`mergeGateCommand` の 2 つめの枝、C2-1 で足した)と、壊れた行のあとの `unreadable` の Detail も同じで、cycle 4 が足したのは、読めない記録そのものがこの枝を `unreadable` にする道。起きるのは結びつきの行を失った台帳か手で書いた台帳だけで、枝は `unreadable` のままになる(安全な向き) | `internal/org/mergeorder.go:839-840`(unreadable の Detail)、`:859-867`(`mergeGateCommand`)、`:113`(`mergeGateRunsWhileOrgRuns`)、`internal/org/gate.go:344-355`(`gateOrgLife` の断り)、`internal/org/mergeorder_test.go:700`(`run ralph org gate --org-id org-c again` を固定) | `t.binding == nil` の枝の Detail は、org が完全な結びつきを持たないので `ralph org gate` は断られる、と言う(次の手は台帳を直すか、検証のスクリプトを自分で走らせる)。最後の cycle なので直さず、239 行に載せる(下の Tech debt identified の (s)) |
| C4-2 | LOW | readability | 新しいテスト 2 つの doc が「covers cross-review cycle 3's finding」と、slug を付けずに triage の指摘を指す。cycle 3 のテストは「covers cross-review cycle 2 of org-merge-order」と slug を添えていて、cycle 3 の報告はそれを数えなかった。239 行の (e) は、slug のない「the cross-review finding」「self-review L2」の引用を残りとして挙げており、cycle 4 の 2 件はその形が増えた。欠陥の中身は同じ段落で言っているので、読めないわけではない | `internal/org/mergeorder_test.go:671`(`covers cross-review cycle 3's finding (C8): ...`)、`:1112`(`covers cross-review cycle 3's finding: after origin's repository is renamed ...`)。slug 付きの形は `:1450-1451` | 2 か所に「of org-merge-order」を足す(数語)。直さないなら (e) に 2 件足す |
| C4-3 | LOW | maintainability | `docs/tech-debt/README.md` の 239 行が、cycle 4 が走ったことで偽になった。(a) 先頭の「Cycle 3 was the last pipeline run (the cap was raised from 2 to 3 ...)」、(b) Why deferred の「Found at the last pipeline run (cycle 3 of 3, after the cap was raised from 2): fixing any of them needs a fourth pipeline run」は、上限を 4 に上げて 4 回目が走った。残りは cycle 4 でも直さず、直すと 5 回目が要る。(c) Related の報告の節が cycle 3 までで、cycle 4 の節がない。(d) C4-1 と C4-2 を足す場所がない(C3-1〜C3-4 は (b)・(o)・(g)・(p) に載っていて、cycle 4 は触っていない)。cycle 3 の C3-5 と同じ形 | `docs/tech-debt/README.md:239`(先頭のセルと Why deferred のセル、Related のセル)。cap 4/4 は計画の進捗の cycle 4 の項目 | `/sync-docs` が下の「register の古くなった所」の表のとおり直す(この回は直さない) |
| C4-4 | LOW | maintainability | 237 行の Related が triage を「(the cycle 3 section at the top, ACTION_REQUIRED #1)」と位置で指す。triage は `/cross-review` が毎回その場で書き換え、最新の cycle を先頭に置き、前の cycle の見出しに `(cycle N)` を付けて下へ移す(546e9f66 で cycle 2 の節がそうなった)。次の cross-review(cycle 4)のあと、cycle 3 は先頭でなくなり、「ACTION_REQUIRED #1」は cycle 4 の表の行も指しうる。指摘の中身は Debt のセルが「(cross-review cycle 3 of this plan)」と言葉で言っているので、指し先だけが古くなる | `docs/tech-debt/README.md:237`(Related のセル)。`docs/reports/cross-review-triage-org-merge-order.md` の見出し(15 行目 `## Triage context` が先頭の cycle 3、40 行目 `## Cycle 2`、70 行目 `## Cycle 1`)。`git show 546e9f66` の見出しの付け替え(`## ACTION_REQUIRED` が `### ACTION_REQUIRED(cycle 2)` に) | 位置を言わず、「cross-review cycle 3 of this plan, the renamed-repository finding」のように、cycle と中身で指す。cross-review の前に `/sync-docs` が直せる |

#### C4-1 の根拠

- 結びつきのない org で `ralph org gate --org-id org-c` を打つと、`gateOrgLife` が `ActiveFeature` を引けず、走っている org には「holds no split plan feature binding ... run ./scripts/run-static-verify.sh and ./scripts/run-test.sh yourself」、走っていない org には「is not running and holds no split plan feature binding」と返す(`gate.go:344-355`)。どちらも gate の記録は書かない。なので `unreadable` は、台帳を直さない限り消えない。merge-order の側では `ready` にならないので、誤って merge に進む向きではない
- cycle 4 の前から、gate の記録だけが名指す枝の `failed`・`regate`・`rebase` の Detail と、最後の読める記録のあとに壊れた行がある枝の `unreadable` の Detail は、同じ「ralph org gate --org-id <org> again」を言い、結びつきがなければ同じ断りになる。cycle 4 が増やしたのは、読めない記録そのものがその枝を `unreadable` にする道と、それを固定するテストで、行き止まりの種類は増えていない(`no_gate` は gate の記録のない枝の分類なので、この枝には来ない)
- 実例は未確認です。結びつきの行を失った台帳か手で書いた台帳が前提で、この repo の `ralph org start --plan` が書く台帳には出ないと読んだ

### cycle 3 の所見を HEAD で見直した結果

| ID | 状態 | 確かめた場所 |
| --- | --- | --- |
| C3-1 | 未着手 | get-url の `[]string{"GIT_TERMINAL_PROMPT=0"}` が `mergeorder.go:667` にあり、doc(`:660`)とテストの期待も残る。239 行の (b) |
| C3-2 | 未着手 | `mergeOrderListNotes` の「run ralph org gate again」は `--org-id` なしのまま(`mergeorder.go:933`、テストの `HasSuffix` は `mergeorder_test.go:515`)。239 行の (o) |
| C3-3 | 未着手 | 上限の `1000` は help(`org_merge_order.go:45`・`:47`)と skill の表の行(4 面)に手書きのまま。cycle 4 は同じ表の行の別の文を直したが `1000` は触っていない。239 行の (g) |
| C3-4 | 未着手 | 定数の説明、`mergeGHLimitNotes`(`:577`)、`withNotes`(`:503`)は変わらない。239 行の (p) |
| C3-5 | 半分 | 237 行の書き足しは cycle 4 の直しに合っている。239 行は古いまま(C4-3) |
| 行番号 | ずれた | cycle 3 の節が引く `mergeorder.go:653`(get-url の env)は `:667`、`:918-920`(`mergeOrderListNotes`)は `:933` に動いた。register は `file.go:NNN` を使わず関数名で引いているので影響しない(`grep` で 0 件) |

### cross-review cycle 3 の ACTION_REQUIRED #1 と WORTH_CONSIDERING #2 は閉じているか

- ACTION_REQUIRED #1(改名した repository で、注記の `gh repo set-default` が効かない)は、計画の決定どおりに閉じた。注記は「if acme/app was renamed to upstream-org/app, point origin at it (git remote set-url origin <its URL>); otherwise run gh repo set-default acme/app in this repository」と 2 つの直し方を並べる。Codex が勧めた「正規の repository を割り出してから捨てる」ことはせず、PR は引き続き使わない(分類は gh がないときと同じ)。改名した repository の PR は、origin の URL が新しい名前を指すまで使われず、その間は merged・closed を見分けず、別の base の PR は `wrong_base` にならない。237 行の Impact がこれを言っている。entry 側に印がなく注記だけで知らせる点は 238 行の既知の穴と同じ形
- WORTH_CONSIDERING #2(gate の記録だけが名指すブランチで、読めない新しい記録のあとに古い pass で `ready` になる)は、指摘の形で閉じた。読めない記録は、結びつきがあればその枝、なければその org の前の読める記録の最新の枝に付き、どちらもなければ付けず注記で数える。注記と doc とテストの言い方が合っている(`mergeorder.go:390-391`、`:312-324`、`mergeorder_test.go:704-705`)。残る経路は、その org に結びつきも前の読める記録もないときで、枝を決められないので付かず、注記の数だけになる(計画の C8 が元から決めた形)。その org の読めない記録が最初の記録なら、あとの読める記録が新しいので結果は変わらない
- 2 つの直しは別のファイルの別の関数で、互いの前提を変えない(注記の文と、記録の付け先)

### register の古くなった所

| 場所 | 今の状態 | 直し方 |
| --- | --- | --- |
| 239 行 先頭「Cycle 3 was the last pipeline run (the cap was raised from 2 to 3 for cross-review cycle 2's ACTION_REQUIRED)」 | 上限は 4 に上がり、cycle 4 が最後。cycle 4 は 239 行の項目を直していない(237 行の書き足しと記録の付け先だけ) | 「Cycle 4 was the last pipeline run (the cap was raised to 3 and then to 4 for cross-review cycles 2 and 3)」とし、cycle 4 で直した項目はないと書く |
| 239 行 Why deferred「Found at the last pipeline run (cycle 3 of 3 ...): fixing any of them needs a fourth pipeline run」 | 4 回目は走った。残りを cycle 4 でも直さなかったので、直すと 5 回目 | 「cycle 4 of 4 ... a fifth pipeline run」に |
| 239 行 Related の報告の節 | cycle 3 までで、cycle 4 の節がない | `self-review ... (section Cycle 4: C4-1 to C4-4)` を足す。`/verify`・`/test` が cycle 4 の節を足すなら、それも |
| 239 行 項目 | C4-1、C4-2 を載せる場所がない | 下の Tech debt identified のとおり (s) を足し、(e) に 2 件足す |
| 237 行 Related「(the cycle 3 section at the top, ACTION_REQUIRED #1)」 | 次の cross-review で位置が変わる(C4-4) | 「cross-review cycle 3 of this plan, the renamed-repository finding」に |
| 236 行・238 行 | 変わらない。236 行は未終端の最後の行の話、238 行の「gh の結果の欠け」の 3 つの形に、改名の場合が増えても entry 側に印がない点は同じ | そのまま |

### 所見にしなかったもの

- 計画の Scope(23 行)と C8(82 行)は「前に結びつきがなければどのブランチにも付けない」のまま。cycle 4 の逸脱は進捗の最後の項目に書いてあり、cycle 2・3 の逸脱も同じ置き方で、承認の digest を変えずに済む。AC4(91 行)は「打った org のその時点の結びつきのブランチに付く」で、今も正しい
- 結びつきと「前の読める記録」が別のブランチを名指すとき、結びつきを先にする決定。テストが両方の並びで固定している。実際の台帳では、gate は結びつきのブランチでしか打てないので両者は一致し、食い違うのは手書きの台帳だけ
- 読める形だが event の名前と reason が合わない記録(`readable` が false)は、自分の `branch=` を使わず、結びつき、前の読める記録の順で付く。cycle 4 の前からそうで、手で書かない限り起きない
- 注記の「renamed」は、owner が変わる移管を言わない。移管も GitHub が転送し、PR の URL は新しい owner になる。直し方は同じで、「renamed to X」と読んで足りるので、文は直さない
- `mergeorder.go` は 969 行(cycle 1 は 681 行)。この repo では `spawn.go`(2215)、`verbs.go`(2156)、`watch.go`(1524)がもっと大きく、ファイルの大きさを守る規則は repo にない。分けるなら gh の lookup と repository の確認の一式が自然な継ぎ目だが、報告が行番号を引いているので別の PR で
- `check` の `fmt.Sprintf` が `%s` を 6 個並べ、引数の順で埋める。テストが全文を固定するので、順の間違いは赤くなる
- `mergeOrderNoPRs` の定数の説明は「ends every note that says no PR is used」だが、`!prRepo.same` の枝は cycle 2 から定数のあとに文が続く。cycle 4 は続く文を長くしただけで、この差は前からある

### Tech debt identified

この回は報告だけをコミットする指示なので、`docs/tech-debt/README.md` には行を足していない。cycle 4 は最後(上限 4/4 で、直すと 5 回目が要る)なので、C4-1 と C4-2 は直さない。新しい行は作らず、239 行を上の「register の古くなった所」の表のとおり直して、そこに次の項目を足すことを勧める(`/sync-docs` が行う。237 行の Related は C4-4 のとおり)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Add to the org-merge-order leftovers row (line 239): (s) a branch that only gate records name (its org holds no complete feature binding) gets Details (`failed`, `regate`, `rebase`, and `unreadable` after a corrupt manifest line) that tell the reader to run `ralph org gate --org-id <org>` again, and `ralph org gate` refuses an org with no complete binding (`gateOrgLife` in `internal/org/gate.go`: "holds no split plan feature binding", "disband it and start it again"); cycle 4 added a record that cannot be read, attached to the org's latest readable gate record's branch, as another way to reach that `unreadable` Detail, `TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranch` pins the command, and `mergeGateRunsWhileOrgRuns` says only "while the feature org runs" (C4-1); and extend (e): the two cycle 4 tests cite "cross-review cycle 3's finding" with no slug (C4-2) | (s) is the only one a reader can hit, and only on a ledger that lost a binding line or was written by hand: the branch stays `unreadable` (it is never `ready`) and the next step the Detail names is refused. (e) changes no behavior | Cycle 4 is the last cycle (4/4): fixing either needs a fifth pipeline run, and each is a few lines | The next edit to `internal/org/mergeorder.go` (`classifyMergeTarget`, `mergeGateCommand`), stage 8a's director reading merge-order, or a report that a Detail sent someone to a refused `ralph org gate` | `docs/reports/self-review-2026-10-14-org-merge-order.md` (Cycle 4, C4-1 to C4-4) |

### Recommendation

- cycle 4 の判定: 上の「Recommendation」の Merge の行のとおり(yes)。cycle 4 の差分に CRITICAL・HIGH・MEDIUM はなく、cross-review cycle 3 の指摘 2 件は閉じた
- Follow-ups:
  1. `/sync-docs` が 239 行と 237 行を上の表のとおり直す。C4-1 と C4-2 は最後の cycle なので直さず、239 行に載せる。237 行の指し方は、次の cross-review が triage を書き換える前に直す
  2. 5 回目を回す理由が別にあるなら、C4-2(数語)と C4-1(Detail の文 1 つとテストの期待 1 か所)を先に直す。skill には触れない
  3. insight event(`./scripts/insights-append.sh --phase self_review`)は、報告だけをコミットする指示なので、この回では足していない。`cycle-count.json` は 2 のままで `--cycle auto` は 2 を入れるので、`--cycle 4` を明示する。件数は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4(cycle 4 の分)で、orchestrator が足す
  4. `/test` で、改名した本物の repository(origin が古い URL、`gh pr list` の PR の URL が新しい名前)で注記が出ること、`git remote set-url origin` で PR が使われることを確かめるとよい。今は fake の gh と git だけで固定している
