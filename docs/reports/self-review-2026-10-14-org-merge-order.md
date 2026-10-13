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

- Merge: yes(CRITICAL・HIGH・MEDIUM はなく、LOW は 8 件。計画が決めた穴(`rebase`・`regate`・`unreadable` を `ready` に戻す手がない)は、skill・README・仕様 FR-8 の注・tech-debt に書かれていて、`Detail` にも但し書きがある。L1 だけは、gate を勧める 7 つの `Detail` のうち 5 つにしか但し書きがない)
- Follow-ups:
  1. L1 を直す(`mergeorder.go` の 1 ファイル、定数 1 つと `no_gate`・`failed` の文言)。読んだ範囲のテストは `Detail` を部分一致で見るので文言を足しても通るが、完全一致で固める箇所があれば合わせる(残りのテストは読んでいない)
  2. L2 と L4 は、直さないなら上の 1 行を register に足す(orchestrator)。L5〜L8 は機械的な直しで、直すなら L7 は skill の 4 面を `scripts/sync-skills.sh` と `cmp` で合わせる
  3. 直すとコードか skill の 4 面に触るので、`/verify` から回し直しになる(cycle 2 が要る)。cycle 1 のまま進めるなら、L1〜L8 は PR の本文に known gaps として書く
