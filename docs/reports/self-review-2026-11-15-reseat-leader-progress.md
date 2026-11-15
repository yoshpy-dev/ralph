# Self-review report: reseat-leader-progress

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)
- Branch: `feat/reseat-leader-progress`、HEAD は `8ac1487c`、base は `origin/main` の `df186374`
- Reviewer: reviewer サブエージェント(`/self-review`)。**Cycle: 1**(`cycle-count.json` は `cycle: 1`)
- Scope: `git diff origin/main...HEAD` のうち Go の本体(`internal/org/reseat_progress.go`、`feature.go`、`reseat.go`)と `internal/org/prompts/leader.md` の規則を全部読んだ。テスト 3 ファイルは関数の一覧、ヘルパー、時間切れと受信箱の失敗のテストだけ読んだ。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれ(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

- `git diff origin/main...HEAD --stat`(9 ファイル、+1918/-6)。`git fetch origin` のあと。
- `reseat_progress.go` を全部。`readReseatProgress`(127-178)、`reseatProgressSection`(198-283)、`reseatProgressText`(321-328)、`readReseatProgressCommits`(370-381)、`reseatProgressCommitsIn`(386-405)。
- 読んだ先: `fenceFor`(`feature.go:603`)、`PrintableInboxText`(`inbox.go:208`)、`resolveGateBase` と `gateGit` と `gateGitOutput`(`gate.go:624`、`:702`、`:730`)、`lastDisbandedIndexes`(`reserve.go:458`)、`planApprovalFromDetails`(`approval.go:343`)、`gateRecord.event`(`gate_record.go:148`)、`default_branch`(`scripts/ralph-common.sh:33`)、`wireOrgInbox`(`internal/cli/org.go:464`)、`inboxSummary`(`internal/cli/org_inbox.go:518`)、`regateIsOwnMerge`(`regate_merge.go:195-203`)、`scopeCommittedPaths` の doc(`watch_scope.go:274`)。
- `StartFeature` の差分(`feature.go:289-295`)と `startAll` の差分(`reseat.go:1252`、`reseatSeatsAfterLeader`)。
- `leader.md:150-199` の規則と、それが指す手順 4、5、6、9、10、12、13、14、15 の本文、「ミッション」の 6。
- `ralph org status` の `(active)`(`report.go:187`)と `--org-id` の存在。
- tech-debt register の行 514(`reseat.go` と `feature.go` の長さ)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1. `leader.md` の規則は「今回の立て直しで立つ座席」だけを扱い、立て直さなかった座席の状態を何も言わない。leader だけを立て直す reseat(既定の対象は active な座席だけで、「ミッション」の 6 は座席を作業のたびに止めるので、leader だけになりやすい)や、leader と reviewer だけを立て直して implementer を動かしたままにする reseat では、節は `今回の立て直し: ...座席: なし(leader だけ)` か `reviewer` だけを言う。座席の項目は「今回の立て直し」の行に起動する予定の座席があるときだけ働き、ほかの 3 項目にも当たらないと「下の手順のとおり進めます」(`leader.md:157-158`)になる。新しい leader は手順 5 で座席を spawn し(動いていれば idempotent)、手順 8 で TASK を送るので、すでに TASK を持つ implementer に同じ TASK を重ねて送るか、コミット済みのスライスを頼み直す。計画の残りのスライスを「branch の commit」の行から決める文は座席の項目の中だけにある(`leader.md:190-193`) | `leader.md:150-199`、`reseat_progress_start_test.go:300-302`(「implementer, which this reseat leaves running, is not named」と自分で書いている)、`reseat_test.go:872`(leader だけの reseat が作れる)、`leader.md:95-98`(座席は作業のたびに止める) | 規則に、手順 5 より前に `ralph org status --org-id {{ORG_ID}} --state-dir <台帳>` で座席の状態を見る一文を足す。`(active)` の座席は TASK を持っているので、spawn も TASK も重ねず `ralph org read` で読んで待つ。止まっている座席に頼むスライスは、コミットの行で済んだものを除いて決める。役割の prompt のテストに、この文を 1 つ足して固定する。節に今の座席の一覧は入れなくてよい(読み取りが増える) |
| LOW | exception-handling | L1. `git log --format='%h %s'` に `--no-show-signature` がない。`log.showSignature=true` で署名付きの commit があると、署名の確認の出力が commit の行の前に stdout へ出て、`reseatProgressCommitLines` が空行だけを除くため、`gpg: ...` の行が commit として節に載り、20 件の上限も壊れる。同じ repo の `regateIsOwnMerge` は同じ理由で `--no-show-signature` を付けている(コメントに理由あり)。`watch_scope.go` の `git log` も付けている | `reseat_progress.go:392`、`regate_merge.go:195-203`、`watch_scope.go:274` | `"log", "--no-show-signature", "--format=%h %s", ...` にする。`rev-list --count` は影響を受けない |
| LOW | naming | L2. 節の見出しが `branch の commit: 31 件(最初の 20 件)` だが、20 件は `git log` の新しい順の先頭、つまり新しい 20 件である。「最初の」は古い順に読める。leader はこの行から済んだスライスを決めるので、31 件のうち古い 11 件が見えないことが伝わらない。受信箱の「ほか N 件」は逆に、ID の昇順で後ろ(新しい件)を落とす | `reseat_progress.go:253`、`:392`(`--max-count` は新しい順)、`reseat_progress_test.go:422`(「最初の 20 件」を固定)、`reseat_progress_start_test.go:25`(newest first と自分で書いている) | `(新しい順に 20 件)` にする。受信箱も同じ向きにするか、「ほか N 件(古い順で 20 件の後ろ)」と書く |
| LOW | readability | L3. 承認の行が `計画の承認: <パス> sha256:<digest>(<時刻>)` で、digest のすぐ後ろに `(` が付く。`leader.md:162-163` の規則は「行の `sha256:` のあとの digest」と `plan-visual.sh digest` の出力を比べるので、空白で切ると `0123456789ab(2026-11-15T01:00:00Z)` になって違うと読める。読み違えると安全側(承認なしの扱い)に倒れるが、規則の最初の分岐が働かない。gate の行も `branch <名前>(<時刻>)` と値の後ろに括弧が付く | `reseat_progress.go:208`、`:219`、`reseat_progress_test.go:313`(行の形を固定) | 括弧の前に空白を入れる(`sha256:<digest> (<時刻>)`)。または時刻を `at <時刻>` の形にする。値と注記を分けるほうがよい |
| LOW | maintainability | L4. Go のコメントが `docs/plans/active/2026-11-15-reseat-leader-progress.md` を指す(3 ファイル)。`/pr` が計画を archive へ移すので、指す先が消える。仕様のほうは `docs/plans/archive/...` を指していて食い違う。過去の reseat のレビューの L6 と同じ形 | `reseat_progress.go:15`、`reseat_progress_start_test.go:15`、`reseat_progress_test.go:11`。`prompts_leader_reseat_test.go:11` は slug だけで、これは問題ない | slug だけ(`plan 2026-11-15-reseat-leader-progress`)にするか、archive のパスにする |
| LOW | maintainability | L5. 3 点。(a) `reseatProgressSection` は 86 行(198-283)で、6 項目の整形と `progress` の旗の管理を 1 つの関数に持つ。`readReseatProgress` も 52 行。リポジトリの目安は 50 行。gate の行、受信箱の行、commit の行をそれぞれの関数に出せる。(b) 「本文の最初の空でない行」を取る処理が `inboxSummary`(`internal/cli/org_inbox.go:518`)と二重になった。違うのは長さと省略の記号だけ。`org` 側の関数に寄せて cli から呼べる。`reseatProgressFirstBodyLine` の doc コメントは `ralph org inbox` と同じ要約だと書いている。(c) `reseatProgressText` は呼ぶたびに `strings.NewReplacer` を作り、`"\\"+"u2028"` と定数を連結している(`` ` ` `` と書ける)。呼び出しは 1 回の節で数十回までなので害はない | `reseat_progress.go:198-283`、`:296-312`、`:321-328` | 次にこのファイルを触るときに直せばよい。(b) は両方の呼び手が 1 つの関数を読むようにしておくと、受信箱の要約の規則を片方だけ変える事故を防げる |
| LOW | readability | L6. 計画の項目の最初の分岐は「digest が同じなら書き直さない。先頭の 2 行が手順 4 のとおりでなければ書き換えてコミットし、手順 5 から進める」で、**すでに 2 行が正しい**ときの次の手順を言わない(`手順 5 から` は書き換える側の文の中にだけある)。ほかの分岐は進む先を言っている。読み手は、書き直さない、の次に手順 1 へ戻るか、手順 5 へ進むかを選ぶことになる | `leader.md:160-168` | 「どちらの場合も手順 5 から進めます」と、2 つの場合を並べたあとに書く。役割の prompt のテストの `ReseatRulePlanItem` に「手順 5」の位置を足す |
| LOW | maintainability | L7. gate の項目は「動いている implementer と reviewer を手順 13 のとおり止める」とだけ言い、座席の項目にある「`(active)` になるまで待つ」を持たない。reseat が leader のあとに座席を立てている最中に leader がこの分岐に入ると、まだ立っていない座席は止める対象に見えず、あとから立った座席が TASK を持たないまま disband まで残る(害は小さい: TASK が無いので作業はしない)。M1 の `ralph org status` の確認を座席の項目の前に置けば、この分岐も同じ確認を通る | `leader.md:176-185` と `:190-197`、`reseat.go:1285-1297`(座席は leader の後) | M1 の直しの一文を、gate の項目の前(4 項目の前)に置く |

## Positive notes

- 節の安全は、依頼で挙げられた点を読んで確かめた。
  - 座席や人が書いた文は、行の頭に ralph のラベルが付くので、行を偽れない(`- <id> <type> <task> <state>: <1 行目>`、`- <hash> <件名>`)。`leader.md` も「頭の...で見分けます」と書いている。改行と制御文字は `PrintableInboxText` と U+2028・U+2029 の置き換えで 1 行になる。長さは置き換えの後で数える。本文は `fenceFor` の囲みの中にあり、件名にバッククォートが続いても囲みが長くなる(`fenceFor` は連続の最長を見る)。
  - 時刻の比較は `time.Parse(RFC3339)` で、小数秒も offset も読める。読めない件は載せて理由を付け、`disbanded` の時刻が読めないときは org の全件がそうなる。同じ秒の件を外すことはコメントに書いてある。
  - git は 1 つの `context.WithTimeout`(10 秒)で `symbolic-ref`、`show-ref`、`rev-parse`、`log`、`rev-list` を通して縛る。エラーは 1 行の理由になり、`StartFeature` は止まらない。テストの stub は `exec sleep` で、pipe を握る子が残らない。時間切れの文はタイムアウトの値を書く。
  - 空の規則。`startProgressSection` が `""` を返すと `s.leader.Task` は触られず、`featureLeaderTask` の完全一致のテストの形のままである。読み取りの失敗だけでは節を作らない(`progress` の旗)。
  - 台帳の読み方は兄弟と同じ。`lastDisbandedIndexes`、`planApprovalFromDetails`、`gateRecordFromDetails` と `r.event() == ev.Event`(CI の記録と名前の食い違いを読まない)。S1 の作業メモは `currentPlanApprovals` との一致をテストで確かめたと書いている。
  - 座席の並び。`reseatSeatsAfterLeader` は `plan.Start` から leader を除き、起動順のまま渡す。leader が対象でない reseat は `StartFeature` を呼ばず、節も作らない。
  - 配線。`o.Inbox` は本番で `wireOrgInbox` が埋める(テストだけの配線ではない)。`ralph org status --org-id` と `(active)` の表示は存在する(`report.go:187`)。
  - `resolveGateBase` が見つけない(`!found`)ことは、feature の worktree では起きない。worktree を作る `ralph-worktree.sh` の `default_branch` が同じ順で解決し、解決できなければ worktree が作られないため。「base が無ければ 0 件、失敗なし」の判断は妥当。
  - 手順番号の参照(2、3、4、5、9、10、12、13、14、15)は本文と合う。手順 14 が控えとして使う `HEAD:` と `BASE:` を節の gate の行から取る指示は、`.harness/state/gate-<org>.out` が worktree と一緒に残ることとも矛盾しない。
  - 座席の項目の「自分で spawn しない」と「10 分で BLOCKED」は、BLOCKED の `TASK_ID`(org_id)と `ralph org escalate` の形が手順 7 と同じ。
- 新しい処理を `feature.go`(873 行)にも `reseat.go`(1,553 行)にも足さず、`reseat_progress.go`(405 行)に出している。`StartFeature` の変更は 6 行で、`featureLeaderTask` は触っていない。
- 重複した呼び出しの経路(leader が active のままの `start --plan` の再実行)でも節を作って捨てるが、害は git の数回と受信箱の 1 回の読みだけで、10 秒の上限がある。指摘にはしない。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L1 から L7 のうち、この PR で直さないもの | 上の表のとおり(LOW) | LOW で、挙動を変えない | cross-review か次の `leader.md` の変更 | この報告書 |

_(register の追記は `/sync-docs` の担当。この回は追記していない。下の引き継ぎを参照。)_

## Recommendation

- Merge: yes(CRITICAL と HIGH はなし。MEDIUM 1 件、LOW 7 件)
- Follow-ups:
  - M1(と L7)は `leader.md` の規則に 1 文足してテストで固定する。直すなら cycle 2 で `/self-review` から回す。直さず進めるなら、`/pr` の既知の穴に「leader だけの reseat では座席の TASK の状態を規則が言わない」と書く。
  - L1 は 1 引数の変更で、理由の先例が repo にある。M1 と一緒に直すのが安い。
  - **`/sync-docs` への引き継ぎ**(指摘ではない):
    - tech-debt の行 514 の trigger「次の `reseat.go` の変更」にこの差分が当たる(`reseat.go` 1,537 → 1,553、`feature.go` 848 → 873)。分割は払われていない。行に今回の数を足す。
    - S3 の implementer が範囲外として挙げた 3 点(`gh pr list --state all` が merge されず閉じた PR も「ある」と読むこと、PR を作ったあと RESULT の前に止まった leader は手順 15 に進み RESULT が上がらないこと、承認なしのコミット済みの計画から手順 2 に進むとき計画助言が済んでいたか分からないこと)は、この差分には register の行がない。
