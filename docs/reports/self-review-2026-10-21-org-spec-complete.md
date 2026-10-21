# Self-review report: org-spec-complete

- Date: 2026-10-21
- Plan: `docs/plans/active/2026-10-21-org-spec-complete.md`(パイプライン cycle 1。直しのあとの再レビュー)
- Reviewer: reviewer サブエージェント(`/self-review`、2 回目)
- 前の版: 7bf4bbb9 の report(M1、L1〜L5、I1〜I3)。原文は `git show 7bf4bbb9:docs/reports/self-review-2026-10-21-org-spec-complete.md`。計画の Progress が「M1・L1」と ID で指すので、この版は ID を引き継ぎ、状態の列を足した。新しい所見は N1〜N5
- Scope: `git diff 765bcc46...HEAD`(HEAD c56b70a4、5 ファイル、docs だけ)。直しの commit 9733eb31(仕様 +11/-10、根拠の表 +13/-12、tech-debt +1)と計画の c56b70a4(+2/-1)を重点に見た。diff の品質だけを見た。仕様適合、テストの実行、静的解析、文書のずれは `/verify`・`/test`・`/sync-docs` に任せた

## Evidence reviewed

- 直しの diff を全部読んだ: `git show 9733eb31`、`git diff 7bf4bbb9 HEAD`、`git show c56b70a4`。あわせて根拠の表 109 行、計画 132 行、仕様の Status と印の注 20 個を HEAD で読んだ
- 仕様の変更が範囲に収まるか(計画の AC7): base と HEAD の仕様を行の集合で比べた。base の行のうち HEAD に同じ形で残らないのは 33 行で、内訳は印の 1 文字の変更 19、Rollout の PR 番号 9、文を直した行 5。足した行は 70 で、Status 17、「印(2026-10-21)」の注 20、印の行 19、Rollout 9、文を直した行 5。範囲外の行は 0。直しの commit が触れた「文を直した行」は 7c-2 の注の 1 行だけで、語単位の差は「それと 」から「このほかに、」への 1 か所だった
- 印と根拠の表の判定: 仕様の 32 個の checkbox のうち FR-12 を除く 31 個を、表の判定の列とスクリプトで 1 つずつ照らした(31 件すべて一致)。仕様は `[x]` が 20(表の 19 と、前から `[x]` の FR-12)、`[ ]` が 12 で、表の「判定の数」(19 と 12)、Status の「12 項目」と合う。Status の所見の一覧(F3、F4、F7、F10、F13、D-F3、「leader への ALERT」)は、`[ ]` の 10 項目の注が挙げる所見の和集合と同じで、F6 は FR-6 の注に「関連する所見」として 1 回出るだけ
- 新しい tech-debt の行を、根拠と照らした。`docs/evidence/org-trial-2026-10-20/watch.log` の `failed to ALERT leader` は 10 行で、`scope_change` 7、`liveness` 2、`stall` 1、すべて `bash: timed out: context deadline exceeded`。`internal/org/watch.go` の `sendAlert` は `callWithTimeout` の下で `SendWatchdogAlert` を打ち、失敗を stderr に書いて続ける。`internal/org/verbs.go` の `defaultDriverCallTimeout` は `10 * time.Second`。`checkDeadman` は、座席が subject の alert を leader の活動で消し、期限か leader が subject のときに escalate する(行の Impact は座席が subject の話として合う)。行は 5 列、記録された所見の名前は表と仕様の 3 か所で行の先頭と一致、16 進の値・pane の id・`/Users/` なし
- F4 の推定の元を HEAD で読み直した: `internal/org/reserve.go` の `RunningOrgs`(active な座席、閉じていない workspace、予約のどれかがある org を数える)と `TotalActiveSeats`(`s.Active` だけを数える)、`internal/org/seat.go` の `activeEvents`(`spawn_failed` を含まない)。AC3 の注と、FR-3 の注の「`max_total_seats` の数にも入らないはず」は、この 3 つと合う
- 範囲外の確かめ(読むだけ): 最新のタグは v5.1.0(2026-08-24)で、Status の release の文は今日正しい。`./scripts/plan-visual.sh digest` は `98781cfa6662` で承認時と同じ(Progress の行は digest に入らない)。`git diff --check origin/main...HEAD` は出力なし。直しの追加行に `/Users/`・`ghp_`・`sk-`・`AKIA`・`xox`・10 桁以上の 16 進の並びは 0 件、U+FFFD は 0 件。テスト名は、ブランチの差に `.go` が 0 本なので cycle 1 の照合(65 個、101 か所)がそのまま成り立つ
- `/pr` の archive が計画の `docs/plans/active/` のリンクを書き換えるのは `docs/tech-debt/README.md` だけで(`.claude/skills/pr/SKILL.md` の 8、前例は 7e05e368)、新しい行の `docs/plans/active/2026-10-21-org-spec-complete.md` は archive の commit で直る。根拠の表 3 行目の同じパスは直らないが、前例の `docs/evidence/org-trial-2026-10-20.md` の 3 行目も `plans/active/` のままなので、所見にしない
- 確かめていない: テストの本文(名前と置き場所だけ)、`gh pr view` での PR の状態(表の記録を信じた)、試行の本文の F6・F7 以外の所見、natural-japanese の lint(hook が書き込み時に回す)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

### 前の版の所見の状態

| ID | Severity | 要点 | 状態 | 直しの確認 |
| --- | --- | --- | --- | --- |
| M1 | MEDIUM | F4 の推定が FR-2・受け入れ条件 2・NFR では `[ ]` の理由なのに、FR-3 と受け入れ条件 3 では印の注がないまま `[x]`。表の F6 の行と「確かめていないこと」も FR-3 をめぐって引き合っていた | 解決(方針 (b)) | FR-3 を `[ ]` にした(仕様 61 行)。注(66 行)は、leader への ALERT の失敗 10 回(`scope_change` が 7 回、原因は調べていない)と、F4 の座席が `max_total_seats` の数に入らないはず(推定、再現していない)を書く。表は 37 行に「leader への ALERT」の行を足し、23 行(F4)に `max_total_seats` と `max_orgs` の扱い、49 行(FR-3)、104・105 行(確かめていないこと)を直した。受け入れ条件 3 は `[x]` のままで、注(152 行)が、`RunningOrgs` は閉じていない workspace と予約でも org を数えるので F4 の org も `max_orgs` に入る、と理由を書く。コードと合う。数は上の Evidence のとおり一致 |
| L1 | LOW | FR-6 の注が F6 を `[ ]` の理由に並べるが、F6 は FR-6 のどの文とも食い違わない | 解決 | FR-6 の注(92 行)は F13 から始まり、F6 は「関連する所見。FR-6 の文とは食い違わない」。表 25 行の F6 は「いいえ。…」で始まり、FR-3 に関わる ALERT の失敗は 37 行の別の行に分けた。Status の一覧から F6 が消えた |
| L2 | LOW | Status の release の文に日付がない | 解決 | 仕様 24 行が「2026-10-21 の時点で最新のタグは v5.1.0(2026-08-24)で…」になった。`git tag` と合う |
| L3 | LOW | 「この系列」の指す先が仕様にない | 解決(N1 が残る) | 「この系列」は消えた(仕様 22 行)。ただし直しの文言「この PR のあと」の「この PR」も、仕様に番号がない(N1) |
| L4 | LOW | 表の「このブランチの run-verify も通った」が確かめられない | 解決 | 表 61 行から消え、`verify.local.sh` と CI の説明だけが残る |
| L5 | LOW | 日本語の 4 点((a) 表 5 行の「食い違う」の定義、(b)「(状態の説明)」、(c)「合成の根拠で」、(d) 7c-2 の注の「それと」) | 解決 | (a) 表 5 行が「項目の文が起きないと言っていることが試行で起きたか、…回り道が要ったことで」になった。読める(「起きたか」と「要ったこと」の述語が不ぞろいなだけ)。(b) 仕様 137 行が「(NFR の「状態の説明」)」。(c) 仕様 148 行が「別々のテストで確かめた」で、「合成の根拠」は表 63 行の注に残る。(d) 仕様 102 行が「このほかに、」 |
| I1 | INFO | AC3 の注が計画を完全なパスで指す。「`start --plan` の事前の検査は何も書かずに断る」の決定が仕様にない | 一部解決 | パスはファイル名だけになった(仕様 152 行)。決定が仕様にない点は残るが、AC7 と引き合うので系列の PR に回す、は cycle 1 のとおり。対応不要 |
| I2 | INFO | 「28 本」「25 本」の数え方 | 解決 | Status 5 行は「関係する PR は次の 28 本」、表 62 行は「関係する PR 28 本のうち…25 本」。28 は、仕様 1、9 段 24、試行の機能の PR 2、#244 の 1。25 は 24 と #244 |
| I3 | INFO | AC1 の `[x]` の基準 | 対応不要のまま | M1 の直しで基準を締めた対象は F4 の推定と ALERT で、AC1(3 つの org の表示)に関わらない |

### 直したあとの新しい所見

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (N1) | maintainability(時間で古くなる文) | Status の 2・3 段落が、今日の数と一覧を現在形で言い、更新の担当に入れていない。20 行の「12 項目は `[ ]` のまま残る。試行の所見(F3、F4、F7、F10、F13、D-F3、「leader への ALERT」)と食い違う項目と…」は、系列の最初の PR が 1 項目に `[x]` を付けた時点で誤りになる。22 行は、各 PR が更新するものを「項目の印と根拠の表」と書き、Status の数と一覧を含めない。同じ 22 行の「この PR のあとの別々の PR」は、この PR の番号が仕様のどこにもなく(Status の一覧は merge 済みの 28 本)、あとから読む人は PR を特定できない。L3 の直しの文言は私が勧めたもので、「この PR」の指す先を見ていなかった。L2 と同じ型(日付のない現在形) | 仕様 20・22 行。計画の Risks の「2 本目以降の計画の Scope に「仕様の印と根拠の表を更新する」を入れる」。前例の `docs/specs/2026-08-17-overlay-scaffold-v2.md` の `## Status` は数も一覧も持たない | 次のどちらか。(a) 20 行を「2026-10-21 の時点で 12 項目は…」にし、22 行の更新の対象に「Status の節の数と所見の一覧」を足す。(b) 数と一覧を消し、「`[ ]` の項目と理由は、項目の下の「印(2026-10-21)」と根拠の表にある」にする(更新が要らない)。どちらでも、22 行の「この PR のあとの別々の PR で入れる」は「2026-10-21 の時点で、試行の所見の直しは別々の PR で入れる予定で」にする |
| INFO (N2) | maintainability(計画の本文が古い) | 既知の残り 1。計画の本文 28 行は「食い違う」の旧い言い回し、36 行は見込みの `[ ]` に FR-3 がなく、FR-6 の理由に F6 を挙げたまま。承認の digest が本文にかかるため直していない。ずれは Progress の 129 行に、FR-3 の追加と F6 の読み直しとして記録してあり、「印の規則は変えていない」とも合う(表 5 行の言い回しの直しは意味を変えない)。36 行は「見込む」と書き、計画の AC3 は仕様と表の判定の一致を言うので、見込みが AC を縛らない。`/pr` が計画を archive したあとも本文は旧いままだが、Progress の行が差を言う | 計画 28・36・129 行。`./scripts/plan-visual.sh digest` は承認時と同じ | 対応不要。直すなら Progress の 129 行に「28 行の言い回しは表の 5 行で読みやすく直した」を 1 句足す程度(任意) |
| INFO (N3) | maintainability(版の記録) | 既知の残り 2。表 7 行の「確かめた版は…85ec1339(base との差は計画のファイル 1 つ)」は、85ec1339 の時点の事実で、今の差は 5 ファイルに広がった。この版に固定されているのはテスト名の照合で、`git diff --name-only 85ec1339 HEAD` が出すのは docs の 5 ファイルだけ(`.go` も `.sh` もない)。新しい行が読んだ `RunningOrgs`・`TotalActiveSeats`・`sendAlert` も、765bcc46 と HEAD で同じコード。記述は今も成り立つ | 表 7 行。`git diff --name-only 85ec1339 HEAD` | 対応不要 |
| INFO (N4) | readability(原因の候補) | 表 37 行の「候補は座席の環境(F3)か busy な leader(F7)」と、tech-debt の行の Trigger「F3 の改名か F7 の busy な leader かを確かめる」は推測で、「候補」「確かめる」と書くので言い過ぎではない。ただし F3・F7 に引かれる点が 2 つある。`SendWatchdogAlert` は `Agmsg.Send` を直接呼び、F7 の失敗が出た `ralph org send` の herdr の idle 待ち(`wait for seat leader idle/done: herdr: timed out`)を通らない。エラーも `bash: timed out` で、F3 の `agent_not_found` や F7 の herdr のものと違う。10 回の失敗は doctor-escape 7、report-escape 3 で、F7 の polling は report-escape の leader の話(試行の本文の F7) | 表 37 行。tech-debt の行の Trigger。`internal/org/watch.go` の `SendWatchdogAlert`。`watch.log` の 10 行。試行の本文の F7 | 対応不要。直すなら Trigger の「確かめる」を「agmsg の send 自体(`bash: timed out`)を見る。F3・F7 は推測」にする |
| INFO (N5) | readability(所見から項目への参照) | 表 23 行の F4 の「関係する項目」に受け入れ条件 3 がない。同じ行の「食い違う」の列は `max_orgs` が F4 の org を数えると書き、仕様の受け入れ条件 3 の注(152 行)と表 65 行も F4 を指すので、所見から項目へたどると受け入れ条件 3 が出ない | 表 23・65 行。仕様 152 行 | 任意。「関係する項目」に「受け入れ条件 3(食い違わない)」を足す |

## Positive notes

- 31 個の印は表の判定と 1 対 1 で一致し、数(仕様 20 と 12、表 19 と 12、Status 12)と所見の一覧は、注の中身から数え直しても合った。M1 の直しは、同じ推定を FR-2 と FR-3 の両方に当てて、当てない AC3 には理由(`RunningOrgs`)を書いており、cycle 1 で指した「同じ推定を一部の項目にだけ当てる」型は残っていない
- 仕様の変更は計画の範囲に収まっている。base の 33 行と足した 70 行を行単位で分類して範囲外は 0 で、直しの commit が既存の要件・設計の文に触れたのは 7c-2 の注の「それと 」1 語だけ
- 新しい tech-debt の行は、ログの数(10 = 7 + 2 + 1)、10 秒、`sendAlert` の best-effort、`checkDeadman` の挙動、すべてコードとログと合う。原因を「調べていない」と書き、FR-3 が `[ ]` である理由をこの PR の外の作業(系列の PR)に結びつけている。漏えいと U+FFFD はなく、`git diff --check` は出力なし
- 計画の digest は変わらず、Progress の 129 行が、FR-3 の追加、tech-debt の行が計画の Affected areas の外にあること、F6 の読み直しを分けて記録している

## Coverage gaps

- 表のテストが項目の文のどこまでを見るかは確かめていない。名前と置き場所の実在だけを見た(cycle 1 と同じ)
- `watch.log` の行に時刻がなく、10 回の失敗の subject は、隣の `skipping semantic trigger` の行と escalation の alert id から読んだ(座席が subject のものがあり、`deadman_timeout` の escalation が出たところまで)。全件の対応表は作っていない
- `gh pr view` は打っていない。PR の状態は表の記録を信じた
- 試行の本文の F6・F7 以外の所見は読んでいない。表の「食い違う」の判定のうち F3・F13・D-F3 は、表の文と仕様の注の一致だけを見た
- 日本語の lint は回していない(書き込み時の hook の担当)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(新しい行は足さない) | — | この PR が足した行 "The watch's ALERTs to the leaders failed"(commit 9733eb31)は、行の主張をログとコードで照らして合った(N4 は書き方の注意)。N1 は直しを勧める。この回は cap の最後ではない。直さずに最後の回まで残るなら、1 行にまとめて足す | — | — |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。cycle 1 の M1 と L1〜L5 は解決した。LOW は N1 の 1 件で、Status の数と一覧に日付がなく、更新の担当にも入っておらず、「この PR」の指す先も仕様にない点。INFO は、前の版の I1(一部解決)と I3、新しい N2〜N5。印の 31 件の一致、数、範囲の遵守、新しい tech-debt の行の正しさ、漏えいと U+FFFD には欠陥が見つからなかった)
- Follow-ups:
  - N1 の直しは仕様の 2 行(20・22 行)の数語で、コードは変わらない。`/verify` の grep の数え上げの前に直すと手戻りがない。直したあとに self-review を回し直すかは lead の判断に任せる(仕様の 2 行の直しは `post-implementation-pipeline.md` の「記録だけの修正」の外)
  - N2〜N5 と I1、I3 は直さなくてよい。N2 が言う本文のずれは、Progress の行が記録している。計画の Progress の「Review artifact created」の行は、この再レビューの結果(MEDIUM 0、LOW 1)に合わせて直すと、cycle 1 の数(MEDIUM 1、LOW 5)と読み違えられない
  - insight event(`self_review`)は足していない。直しのあとの状態で足すなら `./scripts/insights-append.sh --slug org-spec-complete --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 1 --cycle auto --source skill`。N1 を直したあとなら `--low 0`
