# Cross-review triage report: org-herdr-single-tab

- Date: 2026-11-06
- Plan: docs/plans/active/2026-11-06-org-herdr-single-tab.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-06-org-herdr-single-tab.md(承認の digest 61ba0f08bb4c)
- Self-review report: docs/reports/self-review-2026-11-06-org-herdr-single-tab.md(Merge: yes。LOW L1〜L9)
- Verify report: docs/reports/verify-2026-11-06-org-herdr-single-tab.md(pass。LOW V1・V2、INFO I1〜I4)
- Implementation context summary: S1〜S7 で、driver のペインの操作、CLI のテストの偽の herdr、止める前の確かめをペインのラベルで行うこと、spawn が座席を 1 タブのペインとして作ること(配置のロック、ワークスペースの確かめ、分割の起点、作ったときのタブの id の記録)、headless の director をルートペインに置くこと、文書、分離した herdr での実機の確かめを入れた。test report(docs/reports/test-2026-11-06-org-herdr-single-tab.md)は pass。

codex(`codex exec review --base main`、`-c sandbox_mode=read-only`)の出力の要約は「The new pane lifecycle introduces ownership-check and interrupted-start recovery regressions」で、所見は下の 2 件だった。レビューの前後で git の状態は変わらなかった。

### この run のあとの扱い

ACTION_REQUIRED の 2 件は、ユーザーの以前の指示(cross-review の ACTION_REQUIRED 以上は聞かずに直す)に従って cycle 2 で直す。consult-advisor に範囲を聞き、同じ cycle で次の LOW も直すことにした。

- self-review L6: `TabCreate` が応答にタブの id が無くても成功する → 失敗にする。
- self-review L1、verify V2・I4: 古くなった Go のコメント(`HerdrClient.WorkspaceCreate` の doc、`internal/org/driver/paneenv.go`、verify が挙げた spawn の doc)。
- self-review L3: `--env` の無い herdr の警告文に、agmsg がペインの名前を変えうるので、止める前の確かめが通らなくなることを足す。
- test の所見: ラベルを付けるまで配置のロックを持つことを固定するテスト(tester の案)と、`workspace create` の応答にルートペインはあるがタブが無い場合の行。

次は直さず、tech-debt と PR の既知の穴に書く。

- self-review L5: 最初の座席で `WorkspaceCreate` から spawn_step までに `tab rename` 1 回の窓がある。
- self-review L7: `DirectorTabLabel` などの名前がペインのラベルを指す。
- self-review L8: 並行のテストは 100 ms の待ちに頼る。tester が、ロックを外す変異で赤になることを 70 回確かめた。期限を足すと遅い CI で偽の失敗が出うるので足さない(consult の判断)。
- verify V1: 仕様の非機能要件 2(1 回の spawn で増える herdr の呼び出しは、状態の確認 1 回とラベル付け 1〜2 回)と FR-B4(ワークスペースを使う前に確かめる)がぶつかる。2 席目からの spawn は `workspace get`・`pane list`・`pane split`・`pane rename` の 4 回になる。仕様どうしの矛盾なので、仕様の印を `[ ]` にして理由を書き、PR の既知の穴にも書いてユーザーの判断に残す。
- ACTION_REQUIRED 1 の直し方(下)により、座席 id と org_id が同じ、前の版の座席は、止める前の確かめを通らなくなる(手で閉じるか `--force`)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 座席 id と org_id が同じとき(`--org-id review --id review`)、新しい形のタブのラベルも座席 id と同じになる。herdr が id を振り直して名前の無いシェルに記録した id が当たると、前の版の座席としてタブのラベルで受け入れ、Stop が C-c と close を送る。 | 実際の穴で、self-review L2 と同じ。閉じすぎる向きの失敗なので直す。consult の勧めで、タブのラベルがワークスペースのラベル(org_id)と同じときはタブのラベルによる受け入れを使わない形にする(新しい版の座席はペインにラベルがあるので、穴はこの場合だけ)。`seat_id == org_id` の fixture をテストに足す。 | internal/org/verbs.go、internal/org/confirm_pane_test.go |
| 2 | [P2] director の start が `RecordPane` のあと `PaneRename` の前に死ぬと、登録に名前の無いルートペイン(タブ「1」)が残り、次の start の片付けも `director stop` も `confirmSeatPane` で断るので、立て直せない。 | 実際の穴で、普通に打ち直せなくなるので直す。start の順を `WorkspaceCreate` → `RecordWorkspace` → `PaneRename` → `RecordPane` → `TabRename` にする。rename の前に死ねば、登録はワークスペースだけになり、片付けはワークスペースをラベルで確かめて閉じる(ルートペインごと消える)。計画の D7 の「RecordWorkspace と RecordPane の順」は保つ。 | internal/org/director_start.go、internal/org/director_start_root_pane_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
