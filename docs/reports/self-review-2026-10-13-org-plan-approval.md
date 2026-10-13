# Self-review report: org-plan-approval

## Latest run: cycle 2 の 2 回目(HEAD edee787f)

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc。edee787f は計画に触れていない)
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2)の 2 回目の実行(run 2)。cycle 2 の 1 回目(HEAD 48333f62、報告は 655d47cf)の LOW C2-1〜C2-5 を直した edee787f(1 コミット、6 ファイル、+30/-24)を見た。cycle 2 の 1 回目の結果は、コメントと tech-debt の行が「self-review cycle 2 C2-5」の形でその ID を指すので、末尾の付録の先頭にそのまま残した
- Scope: diff の品質だけ。`git diff 655d47cf..edee787f` を対象にした。ブランチ全体の diff は読み直していない(この直しはコメント、doc、tech-debt の 1 行だけで、全体は cycle 2 の 1 回目が読んでいる)。テスト・静的解析・仕様適合・文書のずれの検査は /test・/verify・/sync-docs が受け持つので、ここでは行っていない
- 止める条件: `.harness/state/standard-pipeline/cycle-count.json` は cycle 2 で、`RALPH_STANDARD_MAX_PIPELINE_CYCLES` は既定の 2 のまま(`scripts/ralph-config.sh:30`)なので、この回のあとに回し直しはない。止めて直すのは CRITICAL と HIGH だけで、MEDIUM 以下は tech-debt の行にして先へ進む。行の案は「Tech debt identified」に書いた
- 番号の付け方: C2-1〜C2-5 は cycle 2 の 1 回目の ID のまま使う(付録にある)。今回足した指摘は C2-6 から

### 前の指摘の扱い(cycle 2 の 1 回目の C2-1〜C2-5)

| ID | 状態 | 根拠 |
| --- | --- | --- |
| C2-1 | edee787f で直った | 7 か所すべてに cycle が付いた。`verbs.go:274`(`cross-review cycle 1 #1`)、`approval.go:477` と `approval_test.go:943`(`cross-review cycle 1 #2`)、`approval_enforce_test.go:310`(`cross-review cycle 1 #1`)、`:517`(`R03 of /test cycle 1`)、`:540`(`R04 and R05 of /test cycle 1`)、`approve_test.go:298`(`R11 of /test cycle 1`)。`git grep -n -E 'cross-review #[0-9]\|test report of org-plan-approval\|\bR(03\|04\|05\|11)\b' -- internal cmd .claude/skills .claude/rules templates docs/quality scripts tests` は 0 件。書き方は同じ系列の前の PR(`internal/cli/org_watch.go:194` の `cross-review cycle 1 #1`、`internal/cli/org_watch_test.go:444` の `/test cycle 1 of org-watch-gate`)と同じ。指し先も合う。今の triage の報告(`Cycle: 1/2`)の ACTION_REQUIRED は #1(`Send` の確かめ直し)と #2(承認 A、B のあと A へ戻す)で、/test の報告(cycle 1)の mutation の表に R03、R04、R05、R11 がある |
| C2-2 | 直っていない。edee787f の範囲外 | `templates/base/docs/quality/quality-gates.md:80` は今も `... of a plan whose current version is approved; escalate a BLOCKED ...` で、root の 81 行目は `... of a plan whose current version is its latest approval and passes the check, never to an earlier approved version; ...`。edee787f の 6 ファイルに template の quality-gates はない。/sync-docs が直す前提は cycle 2 の 1 回目と同じ |
| C2-3 | edee787f で直った。同じ言い回しが 1 か所残る(C2-6) | (1) `Send` の doc(`verbs.go:312-313`)は「see the budget check after the idle/done wait and the second approval check below」になった。コードの順は、待ち(`:386`)、承認の確かめ直し(`:390-397`)、予算の検査(`:414`)、`PaneSendText`(`:428`)で、doc と合う。(2) `checkTaskPlanApproved` の doc(`approval.go:464-468`)は、PlanHeader の枝を「the leader has not written the lines naming the current digest yet (the plan was approved again, or approved once and the TASK names a mistyped digest)」と書く。「as when」を戻す案とは違う書き方だが、枝の条件(今の digest が最新の承認で、先頭の 2 行がまだ今の digest を名指さない)をそのまま言い、再承認と打ち間違いの両方を含む |
| C2-4 | edee787f で直った(doc)。名前は広いまま | doc の最後に「The fixture's leader runs, so the test sees the approval half of the feature org's checks only.」が入った(`approval_enforce_test.go:541-542`)。fixture の leader が動くことは、`newEnforceFixture` が `spawnFeatureLeaderIn` を呼ぶ(`:39`)ことで確かめた。名前の `FeatureOrgChecks` と doc の冒頭の「(its leader, then its approval)」は残るので、読み始めは 2 つとも固定すると読めるが、最後の 1 文で範囲を断っていて、読んで誤らない |
| C2-5 | edee787f で N4 の行に 1 文足された。書かれた事実は合う。列は足りない(C2-8) | Progress のメモを足す編集 `replaceOnce("- [ ] Implementation started\n", ...skip the tests...)` は `git grep -n 'skip the tests' -- internal` で 6 行(`approval_enforce_test.go:198`、`:328`、`:816`、`approval_test.go:598`、`:824`、`:1137`)。`git blame` で cycle 2 の 3563129d が足したのは `:328` と `:1137` の 2 つで、行の「two of them added by cycle 2」と合う。`replaceOnce("Add the login form.", "Add the signup form.")` の新しい 1 つは `approval_enforce_test.go:325`(3563129d)。行の差分は、直前の句点と 1 文だけで、エスケープしていない `\|` は 6 本のまま(列は 5 つ) |
| N4、N8 | そのまま残る | edee787f は `approve.go`、`approvalNextStep` に触れていない。N4 の行は C2-5 の分が足されたので 1 行のまま |

### Evidence reviewed

- コミットの並び: edee787f の親は 655d47cf で、1 本の列。HEAD は読み始めも書く前も edee787f、`git status --porcelain` は空
- 読んだ差分: 6 ファイルの全 hunk。周囲として `approval.go:440-515`(`checkTaskPlanApproved` の doc と本体)、`verbs.go:255-330`(`Send` の doc)、`:355-432`(待ち、確かめ直し、予算の検査、`PaneSendText`)、`approval_enforce_test.go:296-330`、`:505-560`、`approval_test.go:930-962`、README の N4 の行。参照先として、triage の報告(ACTION_REQUIRED の 2 行)、/test の報告の mutation の表(R03〜R05、R11)、template と root の quality-gates の該当行、`scripts/ralph-config.sh:30`
- 機械的に確かめたこと
  - 動きが変わっていないこと: 5 つの Go ファイル(`approval.go`、`approval_enforce_test.go`、`approval_test.go`、`approve_test.go`、`verbs.go`)について、`//` で始まる行を除いた 655d47cf と edee787f の版が `cmp` で同じ。diff の追加行と削除行に、`//` で始まらない Go の行は 0 行
  - 追加した Go の行は 29 行で、どれもコメント。タブを 4 桁と数えて最長は 79 桁(`approval.go:467`)。`git diff --check` は空で、追加行に非 ASCII、TODO、FIXME、デバッグ出力はない
  - 新しいコメントが挙げる識別子 `approvalRefusalPlanHeader`、`SendProgressNothingSent`、`PaneSendText` は、どれも非テストの Go に定義か使い道がある
  - README の差分は N4 の行 1 本。足した文を除くと元の行と同じ(直前の句点は足している)
  - 7 か所の cycle の付け方は、C2-1 の grep が 0 件で、org-watch-gate の書き方と同じ
- 確かめていないもの: テスト、`gofmt`、静的解析は走らせていない。折り返したコメントが `gofmt` のあとも変わらないかは、doc comment の書式に当たる行(見出し、リスト、字下げ)がないことを読んで確かめただけ

### 直しが入れた誤りの確認(依頼された点)

1. コードの動き: 変わっていない(上の `cmp`)。コメントの語句だけが変わった
2. コメントが指す先: 7 か所の `cross-review cycle 1 #N` と `R0N of /test cycle 1` は、triage と test の報告の今の中身と合う。cycle 2 の `/cross-review` が triage の報告を書き直すと、cycle 1 の #1、#2 はファイルから消えうるが、org-watch-gate も同じ形で指しているので、この系列の約束事として問題にしない。`approval.go` の新しい文が挙げる枝(今の digest が最新の承認で、先頭の 2 行が未記入)は、コード(`approval.go:501-516`)の条件と合い、再承認と打ち間違いの 2 つの入力のどちらも、その枝に入る
3. 行の折り返し: 80 桁には収まるが、2 か所で折り返しが途中で止まり、短い行が残った(C2-7)
4. コードとの食い違い: 新しい食い違いはない。C2-3 の (1) で直した言い回しが、同じ関数の本体のコメントに 1 か所残る(C2-6)
5. tech-debt の行: 書かれた事実は合う。足した範囲が Debt の列にとどまる(C2-8)

### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 4 件(C2-2 は cycle 2 の 1 回目から続く 1 件、今回の新しい指摘は C2-6〜C2-8 の 3 件)。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-2(cycle 2 の 1 回目から続く。直っていない): `templates/base/docs/quality/quality-gates.md` の 80 行目の「Feature plan approval」の行が、`TASK` の `PLAN:` の digest を直す場合の条件を `of a plan whose current version is approved` のまま言い、root の 81 行目は `whose current version is its latest approval and passes the check, never to an earlier approved version` に変わっている。`ralph init` はこの template を配るので、downstream の project には直す前の案内の規則が出る。`scripts/check-sync.sh` は `docs/quality/quality-gates.md` を `KNOWN_DIFFS` に入れていてファイルごと照合から外れ、機械は気づかない | `templates/base/docs/quality/quality-gates.md:80`、`docs/quality/quality-gates.md:81`、`scripts/check-sync.sh:98-101`、`git diff --name-only 655d47cf..edee787f`(template の quality-gates を含まない) | /sync-docs が template の 80 行目に root の 81 行目と同じ文を足す。直されずに残るなら、下の行の (c) に回す |
| LOW | readability | C2-6: C2-3 の (1) の直しは `Send` の doc の 1 か所(`verbs.go:312-313`)だけで、同じ「予算の検査は待ちのすぐ後」の言い回しが本体のコメントに残る。`:374-376` は「Resolved before the idle/done wait ... so the fail-closed budget check right after that wait can compare it against the ctx time actually left」と言うが、TASK の本番の道では、待ち(`:386`)と予算の検査(`:414`)の間に承認の確かめ直し(`:390-397`)が入った。確かめ直しは予算を使わず、予算の検査が測る `time.Until(deadline)` にその時間が入るので、値の意味は変わらない。古いのは「すぐ後」だけで、TASK でない送信では今もすぐ後 | `internal/org/verbs.go:375`(`git grep -n 'budget check right after' -- internal` で 1 件)、`:386`、`:390-397`、`:414` | `:375-376` を「the fail-closed budget check after that wait (and, for a TASK, the second approval check)」のように直すか、「right」を落とす |
| LOW | readability | C2-7: edee787f が折り返した doc の 2 か所で、折り返しが途中で止まり、短い行が残った。`approval_test.go:944-946` は `// set back to the first approval's version, (f) a TASK naming the latest` の次が `// approval gets`(16 桁)で、そのまた次の行の先頭の語 `the` が入る幅がある。`verbs.go:277-278` も `// SendProgressNothingSent. The watch gate is`(45 桁)のあとに `// not run again: it is a check at the time of the call (plan` が続き、`not` が入る。ほかの hunk は行末が前後の行と揃っている(`approval.go:484` の `// returned as well.` は文の終わり)ので、この 2 か所だけが揃っていない。80 桁には収まり、コードは変わらない | `internal/org/approval_test.go:945`、`internal/org/verbs.go:277`、`git diff -U0 655d47cf..edee787f -- '*.go'` の追加行の桁数(16 と 45) | 次にその 2 つの doc を触るとき、続く行を詰め直す。この HEAD では直さなくてよい |
| LOW | maintainability | C2-8: C2-5 の追記は N4 の行の Debt の列だけに入り、同じ行の Impact、Why deferred、Trigger、Related の列は N4 の元の範囲のままである。Impact は「承認の記録の仕方や 2 行の書き方を変えると 2 つのファイルで直す」、Trigger は「どちらかの fixture の承認の helper を次に変えるとき」で、Progress のメモを足す編集のリテラル 6 か所は、計画の形を変えても fixture の helper を触らなければ出番がない。Why deferred の「N4 は意図して残した(計画の Progress checklist、2026-10-13)」は cycle 1 の決定で、cycle 2 が足した 3 か所(`:328`、`:1137`、`:325`)には当たらない。Related は報告を `N4 in its run-2 appendix` としか指さず、C2-5 のある場所(cycle 2 の 1 回目の付録)がない | `docs/tech-debt/README.md:219`、`git diff 655d47cf..edee787f -- docs/tech-debt/README.md`(変わったのは Debt の列の 1 文だけ) | Impact に「or a change to the Progress checklist line the tests edit has to be made at six sites」、Trigger に「or the next change to the plan format those literals edit」、Related に「(C2-5 in its cycle 2 run-1 appendix)」を足す |

### Positive notes

- 5 つの Go ファイルは、コメントを除いた行が前の版と同じで、動きを変えずに指し先と doc だけを直している。cycle 2 の 1 回目の C2-1 の 7 か所は、grep が 0 件になるまで揃った
- `approval.go` の PlanHeader の枝の説明が、例(「as when the plan was approved again」)から、枝の条件そのもの(最後に承認された digest で、先頭の 2 行が未記入)に変わった。再承認と打ち間違いの 2 つの入力を括弧で挙げ、コードの条件と合う
- C2-4 は、名前を変えずに、doc の最後で、leader は動いているのでテストが見るのは承認の側だけ、と範囲を断った。mutant を取りこぼさない(`spawnFeatureSeatErr` は 2 つの検査を 1 つの関数に持つ)ことは cycle 2 の 1 回目が確かめている
- N4 の行の追記は事実が合い(6 か所、cycle 2 が足したのは 2 か所、別のリテラルが 1 か所)、`blame` と `grep` で数え直して同じになる。表の列の数も崩れていない

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small leftovers of the org-plan-approval cycle-2 comment fixes (commit edee787f); none changes what is refused or allowed. (a) `Send` (`internal/org/verbs.go`) still comments that its `enterDelay` is resolved before the wait "so the fail-closed budget check right after that wait can compare it", although a TASK's second approval check now sits between the wait and the budget check (the doc comment above was fixed). (b) Two docs reflowed by edee787f stop early and leave a short line: `TestCheckTaskPlanApproved_StaleOrMistypedDigest` in `internal/org/approval_test.go` (`approval gets`) and `Send`'s doc in `internal/org/verbs.go` (after `The watch gate is`). (c) `templates/base/docs/quality/quality-gates.md` (the Feature plan approval row) still says a TASK's `PLAN:` digest moves to the current one for a plan "whose current version is approved", where the root file says its latest approval that passes the check; `scripts/check-sync.sh` lists the file in `KNOWN_DIFFS`, so nothing flags the difference | (a) A reader of `Send` takes the budget check to follow the wait directly. (b) Appearance only. (c) A project scaffolded by `ralph init` gets the earlier account of when a TASK's `PLAN:` is moved to the current digest | Found by self-review cycle 2 run 2 of org-plan-approval (C2-6, C2-7, and C2-2 carried from run 1); at the cycle cap a comment-only fix still needs a full pipeline re-run, and none of the three changes behavior | (a) and (b) The next change to `Send` or to the tests of `checkTaskPlanApproved`. (c) /sync-docs of org-plan-approval; if it is not fixed there, the next change to either quality-gates file | `docs/reports/self-review-2026-10-13-org-plan-approval.md` (C2-2, C2-6, C2-7); `internal/org/verbs.go`; `internal/org/approval_test.go`; `templates/base/docs/quality/quality-gates.md`; `docs/quality/quality-gates.md`; commits 3563129d and edee787f |

_(この 1 行は、依頼の指示(この報告と insight のほかは書かない)に従い、`docs/tech-debt/README.md` に足していない。(c) は /sync-docs が直すなら行にしない。C2-8 は新しい行ではなく、N4 の行(README の 219 行)の Impact、Trigger、Related に、上の指摘の「Recommendation」の句を足す直しである。列の順は README の表と同じで、エスケープしていない `|` は 6 本である。)_

### Coverage gaps

- 動きが変わっていないことは、コメントを除いた行の比較で確かめた。ビルド、`gofmt`、テストは走らせていない。/verify と /test が確かめる
- `checkTaskPlanApproved` の doc は、PlanHeader の枝に入る入力として「承認は 1 回で、TASK が打ち間違えた digest を名指す」を挙げるようになった。テスト (e) の doc とその入力は再承認だけで、この入力の組(承認 1 回、先頭の 2 行が未記入、打ち間違えた digest)を回すケースは読んだ範囲にない。枝は (e) が固定する同じもので、doc の書き方が先に進んだだけなので、指摘にはしない
- cycle 2 の `/cross-review` が triage の報告を書き直したあとの「cross-review cycle 1 #N」の指し先は、その報告の書き方に依る。cycle 1 の中身を残す場所はこの repo の約束事で決まる(org-watch-gate も同じ形)ので、ここでは見ていない

### Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなく、LOW は 4 件。cycle 2 の 1 回目の C2-1、C2-3、C2-4 は直り、C2-5 は N4 の行に書かれた。コードの動きは変わっていない。残る LOW は C2-2(/sync-docs の前提)と、今回の C2-6〜C2-8 で、どれもコメント、README の行、template の 1 文に限られ、挙動を変えない)
- Follow-ups:
  - C2-2: /sync-docs で template の quality-gates の 80 行目に root の 81 行目と同じ文を足す
  - C2-8: N4 の行(README の 219 行)の Impact、Trigger、Related に句を足す。これは tech-debt の README の直しで、コードに触れない
  - C2-6、C2-7: 上限が 2 のままなら、この HEAD では直さず、上の「Tech debt identified」の行を `docs/tech-debt/README.md` に足す(orchestrator か /sync-docs)。コメントを直すなら、/self-review から回し直しになる
  - N8 と N4 の README の行は残す
  - insight event: `docs/insights/events/2026-10-13-org-plan-approval.jsonl` に足した(`self_review`、cycle 2、verdict pass、critical 0、high 0、medium 0、low 4)。UTC の日付が 10-13 に変わったので、cycle 2 の 1 回目の event(`2026-10-12-org-plan-approval.jsonl`)とは別のファイルになった

## 付録: cycle 2 の 1 回目の結果(HEAD 48333f62 の時点)

cycle 2 の 1 回目の結果の本文を、見出しを `## 付録` に改め、小見出しの名前に「(cycle 2 の 1 回目)」を足し、`Merge:` の行の名前を「cycle 2 の 1 回目の判定」に変えたほかは、そのまま残す。小見出しは、ほかの付録と同じく `## 付録` の下の `###` に置いた。C2-1、C2-3、C2-4 は edee787f で直り、C2-5 は tech-debt の N4 の行(README の 219 行)に 1 文足された。C2-2 は直っていない。本文の「Tech debt identified」の行は README に足していない(直した分は要らなくなり、残りは最新の節の行にまとめた)。本文の中の「末尾の付録 3 つ」は、この付録のあとに続く 3 つ(cycle 1 の 3 回目、2 回目、最初の結果)を指す。


- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc。cycle 2 の直しは Progress checklist に 1 行足しただけで、本文は変えていない)
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2)。cross-review の ACTION_REQUIRED 2 件(#1、#2)、/test の mutation の生き残り(R03〜R05、R11)、/verify の D2 を直した 4 コミット(3563129d、2d238702、5e809ac6、48333f62)を見た。cycle 1 の 3 回分(3 回目、2 回目、最初の結果)は末尾の付録 3 つにそのまま残した(コードのコメントとテストの doc が「self-review N7 of org-plan-approval」などの形でその ID を指すため)
- Scope: diff の品質だけ。直しの範囲 `git diff b1f934c9..48333f62`(4 コミット、13 ファイル、+417/-73)を重点に見た。ブランチ全体 `git diff 225dff27...HEAD` は、直しが前の部分と食い違わないかを確かめる範囲で読んだ。テスト・静的解析・仕様適合・文書のずれの検査は /test・/verify・/sync-docs が受け持つので、ここでは行っていない
- 止める条件: `RALPH_STANDARD_MAX_PIPELINE_CYCLES` が既定の 2 のままなら(`scripts/ralph-config.sh:30`)、この回が最後の run になる。止めて直すのは CRITICAL と HIGH だけで、MEDIUM 以下は tech-debt の行にして先へ進む。LOW の行の案は「Tech debt identified」に書いた
- 番号の付け方: cycle 1 の M1、L1〜L11、N1〜N8 はそのまま使う(付録にある)。今回足した指摘は C2-1〜C2-5。`/cross-review` の報告は cycle ごとに書き直されるので、cross-review の指摘は「cross-review cycle 1 の #1」のように cycle をつけて書く

### 前の指摘の扱い(cycle 1 の 3 回目の N6〜N8 と cross-review の 2 件)(cycle 2 の 1 回目)

| ID | 状態 | 根拠 |
| --- | --- | --- |
| N6 | 3563129d で直った | 今の digest の拒否が PlanHeader のときに加えて PlanChanged のときも、`checkTaskPlanApproved` はその拒否を返す(`internal/org/approval.go:509-511`)。承認し直したあとに Progress のメモなど digest が読まない場所が変わった計画へ、古い digest か打ち間違えた digest の TASK を送ると、今の digest の「承認のあとに計画が変わった」が返り、次の手はその変更を戻すことだけになる。戻して送り直せば TaskHeader になり、通る。テスト (h) の 2 ケース(`approval_test.go:1136-1170`)が、拒否を今の digest の `checkPlanApproved` の error と文字列ごと比べ、TASK 自身の拒否の語句(`approved digest <TASK の digest>`、`has no approval recorded`)がないことも見る。tech-debt の N6 の行は消えた |
| N7 | 3563129d で直った | 振り替えの条件に「今の digest がそのパスの最新の承認である」が入った(`approval.go:499-502`、`latestApprovedDigest` は `:570-580`)。TASK が最新の承認を名指すとき、`currentPlanRef` は TASK の digest と違うものしか返さないので、この条件は偽になり、TASK 自身の拒否が残る(`:497-498` のコメント)。ファイルが前の版に戻った場合も、今の digest は最新の承認でないので同じになる。(f) と (g)(`approval_test.go:1122-1134`)がこの 2 つを固定する。雛形の手順 5(最後に承認された digest)は変えておらず、拒否と雛形が同じ digest を言うようになった。tech-debt の N7 の行は消えた |
| N8 | 直っていない。tech-debt の行が残る(`docs/tech-debt/README.md:218`) | cycle 2 は `internal/org/approve.go` にも `approvalNextStep` にも触れていない(`git diff --stat b1f934c9..48333f62 -- internal/org/approve.go` は空)。行の「改訂の前の digest の TASK が新しい digest の PlanHeader の拒否になる」は、今の digest が最新の承認である場合の枝(`approval.go:509-511`)として残るので、3563129d のあとも成り立つ |
| N4 | 意図して残した(README の 219 行に行がある) | 同じ種類の重なりが cycle 2 のテストで 3 つ増えた(C2-5) |
| cross-review cycle 1 の #1 | 3563129d の (a) で直った | `Send` は `AgentWait`(`internal/org/verbs.go:384`)のあと、`PaneSendText`(`:426`)の前に `checkSendApproval` を呼び直す(`:391-395`)。テストは `TestOrgSend_ApprovalCheck_AgainAfterTheWait`(`approval_enforce_test.go:319`)。doc の指し先に小さい残りがある(C2-3) |
| cross-review cycle 1 の #2 | 3563129d の (b) で直った | 承認 A、B の順のあとファイルを A に戻し、TASK が B を名指す入力は、N7 と同じ分岐で TASK 自身の PlanChanged が残る。(f) が固定する |
| /test の R03〜R05、R11 と /verify の D2 | 2d238702(テスト 3 本)と 5e809ac6(help)で手当てした | mutant が本当に落ちるかと help の文書としての判定は /test と /verify の仕事で、ここでは書き方とコードとの合い方だけを見た(下の点 3 と 4) |

### Evidence reviewed(cycle 2 の 1 回目)

- コミットの並び: `git log --format='%h parent=%p' b1f934c9..HEAD` は 3563129d(親 b1f934c9)、2d238702、5e809ac6、48333f62 の 1 本の列で、どの報告より前に書かれた直しも混ざっていない。HEAD は読み始めも書く前も 48333f62、作業ツリーはきれいだった
- 読んだコード: `internal/org/verbs.go` の `Send` の全体(`:234-487`)、`checkSendApproval`、`sendApprovalErr`、`seatRoleFromEvents`。`internal/org/approval.go` の `:380-740`(`approvalRefusalKind` から `planNotApprovedError` まで)。`internal/org/spawn.go` の `Spawn` の dry-run の枝、`spawnPrecheckErr`、`checkCapacityAndStart`、`spawnCapacityErr`、`spawnFeatureSeatErr`、`reject`。`internal/org/approve.go` の `approveCandidates` の note の検査。`internal/cli/org.go` の spawn と send の help
- 読んだテスト: 新規と変更のすべてを、fixture の helper まで。`planEditingHerdr`、`fakeHerdr.AgentWait`(`internal/org/spawn_test.go:239`)、`enforceFixture`(`record`、`mark`)、`checkFixture`(`reapprove`、`markApproved`)、`assertTaskRefused`、`sendVariants`、`TestCheckTaskPlanApproved_StaleOrMistypedDigest` の (f)(g)(h)
- 読んだ文書: 4 つの `/org` SKILL.md の差分、`docs/quality/quality-gates.md` の 81 行目、`templates/base/docs/quality/quality-gates.md` の 80 行目、`docs/tech-debt/README.md` と計画の差分、`internal/org/prompts/leader.md` の手順 5〜7(変更なし)、`.claude/rules/ralph/agent-messaging.md` の承認の段落(変更なし)、アーカイブした `docs/plans/archive/2026-10-12-org-watch-gate.md` の 71、99、108 行目
- 機械的に確かめたこと
  - spawn の側、leader の雛形、start は変わっていない: `git diff --stat b1f934c9..48333f62 -- internal/org/approve.go internal/org/spawn.go internal/org/reserve.go internal/org/feature.go internal/org/prompts/` は空
  - `checkPlanApproved` の書き換えは、`latest` の求め方を `latestApprovedDigest` に移しただけである。`matching` の条件と `since` の文字列は前と同じで、拒否の文は変わっていない(diff を読んだ)
  - 4 つの `/org` SKILL.md(`.claude/`、`.agents/`、`templates/base/` の 2 つ)は md5 が同じ
  - 足した Go のコメントの行は、タブを 4 桁と数えてどれも 80 桁以内。追加行に TODO、デバッグ出力、secret らしい文字列、U+FFFD はない
  - 足したコメントにある CamelCase の識別子 24 個は、どれもコメント以外の行に定義か使い道がある(`git grep -w`)。新しい `func` と `type` の直上の doc は、その直上にある
  - cycle をつけずに報告の番号を指すコメントは 7 か所(`git grep -n 'cross-review #\|test report of org-plan-approval'`)
  - tech-debt の README は、消えた 2 行(N6、N7)以外の差分がない
- 確かめていないもの: テスト、静的解析、実機の確認はしていない。「どの変更を戻すとどのテストが落ちるか」は、コードとテストを読んで組み立てたもので、mutant を動かしてはいない

### 依頼された点の結論(cycle 2 の 1 回目)

1. `Send` の確かめ直し(3563129d の (a))
   - 置き場所: `AgentWait`(`verbs.go:384`)のあと、`--timeout-ms` の残りの検査(`:412-424`)と `PaneSendText`(`:426`)の前(`:391-395`)。残りの検査より前に置いたのは正しい。あちらは `time.Until(deadline)` で測るので、承認の検査が manifest と計画のファイルを読む時間も、その見積もりに入る
   - 拒否のとき: `SendResult{Err: err}` を返し、`Progress` は零値の `SendProgressNothingSent`、`PaneID` は空のままになる。`PaneSendText`、`PaneSendKeys`、`sent` の `appendEvent` はどれもこの後ろにあるので、何も打たず何も記録しない。テストは `assertTaskRefused`(Progress と PaneID)、`fx.h.calls` に `pane_send_text` と `pane_send_keys` がないこと、manifest と receipts のバイト列が前と同じことを見る
   - エラーの形: 早い検査と同じ `checkSendApproval` から `sendApprovalErr` を通るので、`taskNotSentError` の中の `planNotApprovedError` で、manifest を読めないときの文(`org: send: read manifest: ...`)も同じになる。テストは早い検査のテストと同じ `assertTaskRefused` を使う
   - 見張りの関門: doc(`:276-278`)が「もう一度は走らせない。時点の検査である(plan 2026-10-12-org-watch-gate)」と書く。その計画が「検査は時点の判定」と決めている(archive の 71、99、108 行目)ので、理由と出典がある
   - fake: `planEditingHerdr`(`approval_enforce_test.go:295`)は、埋め込んだ `fakeHerdr.AgentWait` が戻ったあとに 1 回だけ `onWait` を呼ぶ。`Send` は `AgentWait` を最初の検査のあとに呼ぶので、`onWait` が計画のファイルを書くのは「最初の検査のあと、2 回目の前」になり、待つ間の変更を再現している。`waited` の確認は、最初の検査で断られて待たずに終わった場合の空振りを塞ぐ。ケースは、digest が変わる編集、digest が読まない Progress のメモ、編集なし、checkbox の印の 4 つを、型付きと `--raw` で回す(8 本)。2 回目の検査を消すと断られるはずの 4 本が落ち、`PaneSendText` の後ろへ動かすと `calls` の確認で落ちる(読んで組み立てたもの)。dry-run は待たないので対象外で、早い検査のテストが受け持つ
   - 小さい残り: `Send` の doc の `--timeout-ms` の説明(`:310-313`)の「idle/done の待ちのすぐ後」が、今は承認の確かめ直しを指す(C2-3)
2. `checkTaskPlanApproved` の新しい規則(3563129d の (b))
   - 規則: 最初の検査が NoApproval か PlanChanged で断り、ファイルの今の digest が TASK のものと違い、かつそのパスの最新の承認と等しいときだけ、今の digest で `checkPlanApproved` を回す(`approval.go:499-514`)。通れば TaskHeader、断りが PlanHeader か PlanChanged ならその断り、ほかは TASK 自身の断りを返す。今の digest の PlanChanged を返す枝が N6 を直し、「最新の承認」の条件が N7 と cross-review #2 を直す
   - `latestApprovedDigest` は、前に `checkPlanApproved` の中で回していた `latest`(`a.Plan == ref.Path` の最後の digest)と同じ求め方である。「its latest approval is of sha256:…」の文は同じ条件で出る。承認の読み込みは `approvals` に 1 回にまとめた
   - (f)(g)(h): 直す前のコードで落ちる。(f) は今の digest(最初の承認の版)が `checkPlanApproved` を通るので、直す前は TaskHeader になり、kind の確認で落ちる。(g) も同じ理由で、NoApproval を期待する。(h) の 2 ケースは、直す前は TASK 自身の拒否(古い digest の PlanChanged か NoApproval)が返り、`r.Error() != changedErr.Error()` で落ちる。cross-review #2 は (f)、N7 は (f) と (g)、N6 は (h) が固定する
   - `/org` skill と quality-gates: SKILL.md の 4 項目は、TaskHeader の枝、PlanHeader の枝、PlanChanged の枝、TASK 自身の拒否が残る場合に 1 つずつ対応する。PlanChanged の項目の「その変更だけを戻す」は、次の手(変更を戻すか、Draft に戻して頼み直す)の前半だけを言うが、従っても誤らない。quality-gates の「whose current version is its latest approval and passes the check, never to an earlier approved version」は TaskHeader の条件と合う。`checkTaskPlanApproved` の doc の 2 段目(`approval.go:475-484`)も、TASK が最新の承認を名指す場合と、ファイルが前の承認の版に戻った場合の両方を書き、コードと合う
3. 2d238702 のテスト 3 本
   - `TestOrgSend_WatchGateBeforeApprovalCheck`(`approval_enforce_test.go:519`): 名前と doc は計画の Scope(40 行目の「見張りの検査のあと、dry-run の枝の前」)のとおり。計画に 2 行は書いたが承認は記録しない台帳で、見張りの関門だけが断る状態にし、型付き、`--raw`、dry-run、`--raw` の dry-run の 4 通りで `ErrNoRunningWatch` で断られ `ErrPlanNotApproved` でないことを見る。2 つの検査の順を入れ替えると 4 通りとも落ちる
   - `TestOrgSpawn_CapacityCheckBeforeFeatureOrgChecks`(`:542`): Scope の 41 行目(「今の `spawnCapacityErr` のあと」)の順を、dry-run と本番で固定する。`max_seats` を 1 にして leader に席を埋めさせ、承認のない implementer の spawn が `max_seats 1 reached` で断られることを見る。ただし名前と doc は「leader の検査、そのあと承認の検査」が容量の検査のあとと言うのに、fixture の leader は動いているので、leader の検査はどの順でも通り、観測できるのは承認の半分だけである。`spawnFeatureSeatErr` は 2 つの検査を 1 つの関数に持つので、呼び出しの順を入れ替える mutant は承認の側で落ち、実害はない(C2-4)
   - `TestOrgApprove_RefusesAnItemItCannotResolve`(`approve_test.go:299`): 計画のパスを 490 字にして note(`approved <パス> sha256:<12 桁>`)が 500 字を超えることを先に確かめ、`Refused` が 1 件、`Approved` が 0 件、manifest と受信箱のバイト列が前と同じことを見る。AC3 の「1 件でも断られたら何も記録しない」のうち、resolve できない件の端を言い当てている。複数の ID の場合は `TestOrgApprove_AllOrNothing` が受け持つので、1 件で足りる
   - 3 本とも、doc が指す `test report of org-plan-approval` の R03 などに cycle がない(C2-1)
4. 5e809ac6 の help: spawn の 3 文と send の文は、どれもコードと合う
   - spawn、leader のいない org で leader 以外を断る: `spawnFeatureSeatErr`(`spawn.go:1284-1291`)。承認方式は問わない
   - spawn、方式が per-plan で role が implementer の座席を、`ralph org approve` で記録した承認が計画のファイルに合うまで断る: `spawn.go:1292-1295` と `checkAnyPlanApproved`(どれか 1 つの承認が検査を通ればよい)
   - spawn、どちらの拒否も dry-run を含めて `rejected` で記録する: dry-run は `o.reject`(`spawn.go:546-548`)が `DryRun: true` の `rejected` と receipt を書き、本番は `checkCapacityAndStart`(`:1227-1229`)が書く
   - send、per-plan で宛先の role が implementer の TASK を、`--raw` と `--dry-run` を含め、何も記録せず断る: `Send` の `isTask` の枝(`verbs.go:325-335`)が dry-run の枝の前にある。「ちょうど 1 行の `PLAN:`」は `HeaderValues` の 1 行の規則。文の順(見つけたこと、目印、次の手)は `planNotApprovedError.Error` と同じ
5. `templates/base/docs/quality/quality-gates.md:80`: 確かめた。root の 81 行目だけが「latest approval ... never to an earlier approved version」に変わり、template の 80 行目は直す前の文のままである。`scripts/check-sync.sh` は `docs/quality/quality-gates.md` を `KNOWN_DIFFS`(98〜101 行目)に入れているので、ファイルごと照合から外れ、この差は機械では見つからない。cycle 1 の 3 回目の時点では 2 つの行は同じだった。`ralph init` が配る面なので、/sync-docs が直す前提で LOW(C2-2)にした

### Findings(cycle 2 の 1 回目)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 5 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1: 7 か所のコメントとテストの doc が、`/cross-review` の報告の番号(`cross-review #1`、`#2`)と `/test` の報告の mutation の番号(`R03`、`R04`、`R05`、`R11`)を、cycle をつけずに指す。triage の報告は slug ごとに 1 ファイル(`cross-review-triage-org-plan-approval.md`)で、見出しに今の cycle を書き(今は `Cycle: 1/2`、skill の例は `Cycle: 2/2 (cap reached)`)、cycle 2 の run が書き直す。`/test` の報告は `test-<date>-<slug>.md` で、同じ日の run なら同じ名前になる(未確認)。書き直されると、`#1`、`#2`、`R03` は別の指摘を指しうる。同じ系列の前の PR は `mutation survivor E8 of /test cycle 1 of org-watch-gate` と cycle をつけている。今回の self-review の ID は、cycle 1 の 3 回分を付録に残したので壊れない | `internal/org/verbs.go:274`、`internal/org/approval.go:476`、`internal/org/approval_test.go:943`、`internal/org/approval_enforce_test.go:310`、`:517`、`:540`、`internal/org/approve_test.go:298`。前の PR の書き方は `internal/cli/org_watch_test.go:444`。`.claude/skills/cross-review/SKILL.md:111` | 7 か所に cycle を足す(`cross-review cycle 1 #1 of org-plan-approval`、`R03 of /test cycle 1 of org-plan-approval`)。Go のコメントだけの直しでも、上限に届いたあとは全工程の回し直しになるので、直さないなら tech-debt の 1 行にまとめる |
| LOW | maintainability | C2-2: `templates/base/docs/quality/quality-gates.md` の 80 行目に、直す前の文が残る。root の 81 行目は 3563129d で `whose current version is its latest approval and passes the check, never to an earlier approved version` に書き直されたが、template の同じ行は `whose current version is approved` のままである。`ralph init` はこの template を配るので、downstream の project には、直す前の案内の規則が出る。`KNOWN_DIFFS` はこのファイルをまるごと照合から外すので、機械は気づかない。cycle 1 の 3 回目の時点では 2 つの行は同じ文だった。同じ commit で、template 側の `/org` skill 2 面は更新している | `templates/base/docs/quality/quality-gates.md:80`、`docs/quality/quality-gates.md:81`、`scripts/check-sync.sh:98-101`、`git diff --stat b1f934c9..48333f62` に template の quality-gates がない | /sync-docs で template の 80 行目に root の 81 行目と同じ文を足し、2 つの行を同じ文に戻す |
| LOW | readability | C2-3: 3563129d が触った 2 つの doc の文が、コードより古いか狭い。(1) `Send` の doc は `--timeout-ms` の予算の検査を「idle/done の待ちのすぐ後」(`verbs.go:310-313`)と指すが、そこには今、承認の確かめ直しがあり、予算の検査はその次になった。(2) `checkTaskPlanApproved` の doc は、今の digest が PlanHeader で断られる場合を「the plan was approved again and the leader has not written the lines since」と言う。コードは、1 回だけ承認した計画で、leader が 2 行をまだ書いていないときに打ち間違えた digest の TASK を送った場合も同じ枝に入る(今の digest が最新の承認で、2 行が Draft と TBD のまま)。前の版は「as when the plan was approved again」と例として書いていた。(2) は例が条件に変わった形で、テスト (e) も再承認の入力だけを使う | `internal/org/verbs.go:310-313`、`:391-395`、`internal/org/approval.go:464-467`、`internal/org/approval_test.go:1054-1090`(e)(e2) | (1) は「承認の確かめ直しのあと、`PaneSendText` の前の予算の検査」と書く。(2) は「as when」を戻す |
| LOW | naming | C2-4: `TestOrgSpawn_CapacityCheckBeforeFeatureOrgChecks` の名前と doc は、機能の org の検査(leader の検査、そのあとに承認の検査)が容量の検査のあとに来ると言う。fixture は leader が動いているので(`newEnforceFixture` が leader を spawn する)、leader の検査は順に関係なく通り、テストが観測できるのは承認の半分だけである。`spawnFeatureSeatErr` は leader の検査と承認の検査を 1 つの関数に持つので、呼び出しの順を入れ替える mutant(R04、R05)は承認の側で落ち、実害のある穴ではない。範囲を広く言う名前と doc だけが残る | `internal/org/approval_enforce_test.go:535-554`、`internal/org/spawn.go:1282-1298`(`spawnFeatureSeatErr`)、`:1257-1268`(`spawnCapacityErr`) | 名前と doc を承認の検査に絞る(`TestOrgSpawn_CapacityCheckBeforeApprovalCheck`)。leader の検査の順も言うなら、leader が止まっていて org 全体の上限に届いている台帳のケースを足す |
| LOW | maintainability | C2-5: N4 の重なりが cycle 2 のテストで増えた。Progress のメモを足す編集 `replaceOnce("- [ ] Implementation started\n", "- [ ] Implementation started\n- skip the tests\n")` が 2 つのテストファイルに 6 か所あり、cycle 2 が 2 か所(`approval_test.go:1136`、`approval_enforce_test.go:328`)、`replaceOnce("Add the login form.", "Add the signup form.")` を 1 か所(`approval_enforce_test.go:325`)足した。N4 の行(README の 219 行)は承認の記録と 2 行を書く helper の重なりだけを挙げ、この編集のリテラルは含まない | `git diff b1f934c9..48333f62 -- internal/org/approval_test.go internal/org/approval_enforce_test.go` の追加行、`git grep -c 'skip the tests'`(両ファイルとも 3) | `approval_enforce_test.go` の `replaceOnce` の隣に、Progress のメモを足す編集の変数を 1 つ置いて使い回す。直さないなら N4 の行にこの 1 句を足す |

### Positive notes(cycle 2 の 1 回目)

- 2 回目の確かめ直しは、早い検査と同じ `checkSendApproval` を呼ぶので、エラーの形が 2 つの位置でずれない。残りの検査より前に置いたので、承認の検査が読む時間も予算の見積もりに入る
- `latestApprovedDigest` を、拒否の文の「its latest approval is of sha256:…」と振り替えの条件の 2 か所が使う。「最新」の意味が 2 か所で別々に決まらない
- (f)(g)(h) は、拒否の kind と、今の digest の `checkPlanApproved` の error の文字列を突き合わせる。直す前のコードでは kind の段で落ちるので、assert が空振りしない
- `TestOrgSend_ApprovalCheck_AgainAfterTheWait` は、digest が変わる編集と digest が読まない編集の両方を断り、編集なしと checkbox の印を通す。2 回目の検査が何でも断る実装になっても落ちる。`waited` の確認が、最初の検査で断られた場合の空振りを塞ぐ
- 3 本のテストは、それぞれ名指しの mutant に結びつき、`TestOrgApprove_RefusesAnItemItCannotResolve` は fixture が note の上限を超えることを先に確かめる。fixture が足りなくて通ってしまう道がない
- 4 つの `/org` SKILL.md と quality-gates の root の行を、コードの直しと同じ commit で書き換えた。leader の雛形は、手順 5 が最初から「最後に承認された digest」なので、変える必要がなかった

### Tech debt identified(cycle 2 の 1 回目)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small leftovers of the org-plan-approval cycle-2 fixes; none changes what is refused or allowed. (a) Seven comments cite `cross-review #1`, `cross-review #2` and the `/test` mutation survivors `R03`, `R04`, `R05`, `R11` of org-plan-approval without a cycle (`Send`'s doc in `internal/org/verbs.go`, `checkTaskPlanApproved`'s doc in `internal/org/approval.go`, and tests in `approval_test.go`, `approval_enforce_test.go` (three) and `approve_test.go`); the triage report and the test report are one file per slug that a later cycle rewrites, so the numbers can come to name other findings. (b) `Send`'s doc still places the `--timeout-ms` budget check "right after the idle/done wait", where the second approval check now is, and `checkTaskPlanApproved`'s doc says the plan-header step applies when "the plan was approved again", although it also applies to a plan approved once whose two header lines the leader has not written. (c) `TestOrgSpawn_CapacityCheckBeforeFeatureOrgChecks` says the feature org's leader check and approval check follow the capacity check, but its leader runs, so it observes only the approval half (a swap of the two calls is still caught, since both checks sit in one function). (d) The Progress-note plan edit is spelled as a literal at six sites of the two approval test files, three of the copies added by cycle 2 | (a) A reader after merge may open the wrong finding. (b) A reader of the two docs gets a narrower or older account than the code. (c) A reader takes the test for a pin of the leader check's position too. (d) A change to the plan format has to be made at each site | Found by self-review cycle 2 of org-plan-approval (C2-1, C2-3, C2-4, C2-5). At the cycle cap, a comment-only fix still needs a full pipeline re-run, and none of the five changes behavior | The next change to `Send`, `checkTaskPlanApproved`, or the org-plan-approval tests: add `cycle 1` to the seven citations, fix the two doc phrases, narrow the spawn test's name or add the leader case, and share one Progress-note edit next to `replaceOnce` (the N4 row has the same trigger) | `docs/reports/self-review-2026-10-13-org-plan-approval.md` (C2-1, C2-3, C2-4, C2-5); `internal/org/verbs.go`; `internal/org/approval.go`; `internal/org/approval_test.go`; `internal/org/approval_enforce_test.go`; `internal/org/approve_test.go`; commits 3563129d and 2d238702 |

_(この 1 行は、依頼の指示(この報告と insight のほかは書かない)に従い、`docs/tech-debt/README.md` に足していない。C2-2 は /sync-docs が直す前提なので行にしていない。直されずに残るなら、同じ行に足す。列の順は README の表と同じで、エスケープしていない `|` は 6 本である。)_

### Coverage gaps(cycle 2 の 1 回目)

- 2 回目の確かめ直しが戻る mutant(検査を消す、`PaneSendText` の後ろへ動かす)と、(f)(g)(h) が直す前に赤いこと、3 本のテストが名指しの mutant で落ちることは、読んで組み立てたもので、動かしていない。/test が確かめる
- (f)(g)(h) は `checkTaskPlanApproved` の段だけを通る。Send を通る古い digest のテスト(`TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest`)は (a)〜(e2) の入力までで、(f)(g)(h) の入力は含まない。Send の包み(`sendApprovalErr`)は cycle 2 で変えていないので、指摘にはしない
- 2 回目の検査から `PaneSendText` までの窓は残る(manifest と計画のファイルを読み、予算を測るだけの時間)。計画の A2 のとおり検査は境界ではなく tripwire なので、指摘にしない
- 新しい help の段落を読むテストはない(verify の D2 が、help は固定されていないと書いた)。検査を変えたときに help が黙って古くなる。文書の判定は /verify と /sync-docs の仕事である
- spawn には、`Send` の確かめ直しに当たるものがない。spawn は計画を名指さず、承認の検査はロックの中の 1 回だけである

### Recommendation(cycle 2 の 1 回目)

- cycle 2 の 1 回目の判定: yes(CRITICAL、HIGH、MEDIUM はなく、LOW は C2-1〜C2-5 の 5 件。cycle 2 の直しは cross-review の 2 件と N6、N7 を直し、cycle 1 の部分とは食い違わない。残る LOW はどれも挙動を変えない)
- Follow-ups:
  - C2-2: /sync-docs で template の quality-gates の 80 行目に root の 81 行目と同じ文を足す
  - C2-1、C2-3〜C2-5: 上限が 2 のままなら直さず、上の「Tech debt identified」の 1 行を `docs/tech-debt/README.md` に足す(orchestrator か /sync-docs)。上限を上げて回し直すなら、5 件とも一緒に直せる。C2-1 は cycle 2 の `/cross-review` が報告を書き直す前に直すほど、指し先が壊れにくい
  - N8 と N4 の README の行は残す。N4 の行には C2-5 の 1 句を足してもよい
  - insight event: `docs/insights/events/2026-10-12-org-plan-approval.jsonl` の最後の行に足した(`self_review`、cycle 2、verdict pass、critical 0、high 0、medium 0、low 5)。UTC の日付がまだ 10-12 だったので、cycle 1 の event と同じファイルに入った

## 付録: cycle 1 の 3 回目の結果(HEAD ee680eff の時点)

3 回目の結果の本文を、見出しを 1 段下げて名前に「(3 回目の結果)」を足し、`Merge:` の行の名前を「3 回目の結果の判定」に変えたほかは、そのまま残す。N6 と N7 は cycle 2 の 3563129d で直ったので、本文の「tech-debt の 4 行」のうち N6 と N7 の行は、今の README にない。本文の中の「付録」は、この報告の最後の 2 つの節を指す。

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)の 3 回目の実行(run 3)。2 回目の結果(02019f48、HEAD 1b95bb84 の時点)の指摘を直した 4b6d0d55・0326574e・ee680eff のあとに見直した。cross-review の前の直しなので cycle は 1 のまま。2 回目と最初の結果は末尾の 2 つの付録にそのまま残した(コードのコメントとテストの doc が「self-review N1 of org-plan-approval」「self-review M1 of org-plan-approval」の形でその ID を指すため)
- Scope: diff の品質だけ。ブランチ全体 `git diff 225dff27...HEAD`(HEAD ee680eff)を対象にし、直しの範囲 `git diff 02019f48..HEAD`(3 コミット、15 ファイル、+556/-52)を重点に見た。テスト・静的解析・仕様適合・文書のずれの検査は /test・/verify・/sync-docs が受け持つので、ここでは行っていない
- 止める条件: orchestrator が consult の判定に従って決めたとおり、この回で止めて直すのは CRITICAL と HIGH だけである。MEDIUM と LOW には、「Tech debt identified」に貼り付けられる行の案を書いた
- 番号の付け方: 最初の結果の M1・L1〜L11 と、2 回目の結果の N1〜N5 はそのまま使う。今回足した指摘は N6〜N8

### 2 回目の結果の指摘の扱い(HEAD ee680eff)(3 回目の結果)

| ID | 状態 | 根拠 |
| --- | --- | --- |
| N1 | 4b6d0d55 と 0326574e で直った(残りは N6 と N7) | `checkTaskPlanApproved`(`internal/org/approval.go:470-497`)は、TASK の digest が「記録なし」か「計画の変更」で断られ、ファイルの今の digest が別の値のとき(`currentPlanRef`、`:504-518`)、今の digest で `checkPlanApproved` をもう一度回す。通れば TaskHeader の種類で今の digest を名指し(`:488-491`)、先頭の 2 行で断られればその拒否を返す(`:492-493`)。承認し直したあとの古い digest と打ち間違いのどちらも、「変更を戻す」「返事を待つ」の手に行かなくなった。雛形の手順 5 は「最後に承認された digest。計画の `- Approved:` の行の digest」になった(`internal/org/prompts/leader.md:137-139`)。最後の枝(`:494-495`)と、基準をファイルの digest に置いたことに残りがある |
| N2 | 4b6d0d55 で直った | 雛形の手順 7 と 4 つの `/org` SKILL.md は「原因を直す前に同じコマンドを打ち直さず」になった(`leader.md:154-155`、`.claude/skills/org/SKILL.md:690`)。PlanChanged の次の手から「step 6 of its procedure」が消え、`TestPlanNotApprovedError_OneNextStepPerKind` が全部の次の手に `(?i)\bsteps? [0-9]` がないことを確かめる。`internal/org` の非テストの Go の文字列に、手順の番号はほかにない |
| N3 | 4b6d0d55 で文は直った(残りは N8) | ほかの誰かが resolve した件について、承認の DECISION をまだ送らず、先に `ralph org inbox show <ID>`(state-dir の案内つき)で resolve のメモを読むよう言う(`internal/org/approve.go:404-409`)。`inbox show` は実在し(`internal/cli/org_inbox.go:150`、引数は 1 つ)、`--state-dir` は `ralph org` の persistent flag で(`internal/cli/org.go:45`)、件の note と各 event の note を出す(`org_inbox.go:382`、`:398`)。ただし承認は DECISION がなくても効いているので、「まだ送らない」が leader を止めるのは、leader が DECISION を待っている間だけである(N8) |
| N4 | 意図して残した。計画のメモにはあり、tech-debt にはない | 計画の Progress checklist(ee680eff)が、2 つの fixture の `record` と `mark` を重ねたままにしたことと、9fb3fafa の本文の書き方を記録している。この直しで同じ重なりが 1 組増えた。`checkFixture.reapprove`(`internal/org/approval_test.go:503`)と、`TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest` の `approveRevisionAs`(`internal/org/approval_enforce_test.go:314`)は、最初の承認を記録し、改訂の承認を追記し、計画を書く、を同じ形で行う。`/pr` で計画は archive されるので、tech-debt の行の案を下に書いた |
| N5 | 4b6d0d55 で直った | `featurePlanHeaderLines` の doc(`approval.go:194-200`)は、行の探し方は /implement と同じで、`- Approved:` の値は `approvedLineDigest` がより厳しく読み、語を足した行は /implement を通りこの検査で断られる、と書く。/implement の 4 手目(`.claude/skills/implement/SKILL.md:34`)と合う |

### Evidence reviewed(3 回目の結果)

- コミットの並び: `git log --format='%h parent=%p' 02019f48..HEAD` は 4b6d0d55(親 02019f48)、0326574e、ee680eff の 1 本の列で、2 回目の報告より前に書かれた直しはない
- 直しの範囲で読んだもの: `internal/org/approval.go` の `featurePlanHeaderLines` の doc、`approvalRefusalKind` と `approvalCheckError`、`checkTaskPlanApproved`、`currentPlanRef`、`approvalNextStep`、`planNotApprovedError`。変わっていないことを確かめるために `checkPlanApproved`、`checkApprovedPlanFile`、`checkApprovedPlanHeader`、`checkAnyPlanApproved`、`PlanContentHash` の doc も読んだ。ほかに `internal/org/approve.go` の `resolveApproved`、`internal/cli/org_approve.go` のコメントと `printApprovedPlans`、`internal/cli/org_inbox.go` の `show`、`internal/org/prompts/leader.md` の手順 3〜7 と `implementer.md` のミッション、4 つの `/org` SKILL.md の承認の節、`docs/quality/quality-gates.md` の 2 つの面、計画の Progress checklist の新しいメモ
- テストは、書き方と、どこを外すと落ちるかを読んだ: `TestPlanNotApprovedError_OneNextStepPerKind` の新しい 2 ケースと 2 つの形の数え方、`TestCheckTaskPlanApproved_StaleOrMistypedDigest` の (a)〜(e2)、`TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest`、`TestOrgSend_ApprovalCheck_PassesAndApprovedAgain` の書き換え、`TestOrgApprove_ResolvedBySomeoneElse`、`prompts_test.go` の 2 行
- 機械的に確かめたこと
  - spawn の側: `git diff --stat 02019f48..HEAD -- internal/org/spawn.go internal/org/verbs.go internal/org/reserve.go` は空。`approval.go` の差分に `checkPlanApproved`、`checkApprovedPlanFile`、`checkApprovedPlanHeader`、`checkAnyPlanApproved` の hunk はない。`spawnFeatureSeatErr`(`internal/org/spawn.go:1293`)は今も `checkAnyPlanApproved` を呼ぶ
  - 0326574e で消えた名前 `currentApprovedRef` は、Go と md のどこにも残っていない(`docs/reports/` を除く)
  - 直しの範囲で足した Go のコメントにある CamelCase の識別子 17 個は、どれもコメント以外の行に定義か使い道がある
  - 足した Go のコメントの行は、タブを 4 桁と数えてどれも 80 桁以内。追加行にデバッグ出力、TODO、secret らしい文字列はない
  - 4 つの `/org` SKILL.md は md5 が同じ。`docs/quality/quality-gates.md` と `templates/base/` の写しは、新しい行が同じで、ほかの違いはこの PR より前からある 3 か所(root 側の 54 行目、57〜58 行目、82 行目)だけ
  - ee680eff の 7 行は `## Progress checklist`(158 行目から終わりまで)の中にあり、digest が読まない節である
  - 作業ツリーは読み始めのとききれいで、HEAD は ee680eff だった
- 確かめていないもの: テスト、静的解析、実機の確認はしていない。N6〜N8 はコードを読んで組み立てた入力で、動かしてはいない

### 依頼された点の結論(3 回目の結果)

1. `checkTaskPlanApproved` と `currentPlanRef` の分岐の順と種類: 最初の検査が NoApproval か PlanChanged で断ったときだけ先へ進む(`approval.go:479-481`)。Other(結びつきが不完全など)はここで返るので、`currentPlanRef` が `f.Worktree` を読むとき f は必ず完全で、その前提は `currentPlanRef` の doc(`:502-503`)にある。TASK の digest の PlanHeader もここで返る。全文のハッシュだけが違う PlanChanged は digest が同じなので、`currentPlanRef` が false を返して元の拒否が残る。最初の検査が失敗したあとの枝はどれも error を返すので、通る道は増えていない。種類は、今の digest が通れば TaskHeader(今の digest と path を持つ)、先頭の 2 行で断られれば今の digest の PlanHeader、それ以外は TASK の digest の拒否になる。残りは、最後の枝に今の digest の PlanChanged が入ること(N6)と、基準が最後の承認ではなくファイルの digest であること(N7)
2. leader が従う拒否の並びが止まらなくなるか、承認済みの計画を捨てさせるか: 止まらなくなる並びは見つからなかった。PlanHeader(今の digest)の手は先頭の 2 行を、TaskHeader の手は TASK のヘッダを変えるので、どちらも次の入力が変わり、2 手で通る。承認済みの計画を捨てさせる並びは 2 つ残る。(a) 承認し直したあとに、計画の digest が読まない部分(Progress のメモなど)が変わると、「新しい digest で 2 行を書け」のすぐあとに「変更を戻せ」が来る(N6)。(b) ファイルが前に承認された版に戻っていて、TASK が最後の承認の digest を名指すと、拒否は前の版の 2 行を書かせ、ヘッダを前の版の digest に直させる(N7)。どちらも行き着くのはこの命の中で人が承認した版で、承認のない計画が通ることはない
3. spawn の側の検査: 変わっていない(Evidence)。`checkAnyPlanApproved` は最後の承認の拒否の種類と digest を受け継ぐが、`plan` は受け継がない。`plan` を入れるのは `checkTaskPlanApproved` だけなので、spawn の拒否に「`PLAN:` を直す」形の手は出ない
4. 新しいテスト: (e) は、TASK の拒否が今の digest での `checkPlanApproved` の error と文字列ごと同じであることを、2 行が Draft と TBD のままの場合と前の digest のままの場合の両方で確かめる。PlanHeader の枝を消すと最後の枝に落ちて古い digest の PlanChanged になるので、(e) は落ちる。次の手が古い digest の 2 行を書かせないこと(`"sha256:"+fx.ref.Digest+" before"`)も見る。(e2) は、2 行を書いたあとに TaskHeader になり、名指された digest で通ることまで確かめる。Send の側は 6 ケースと (e2) を typed・raw・dry-run の 3 通りで回し、断られた前後で台帳と driver の写しが同じであることと、最後の送信の `dry_run` が合うことを見る。N6 と N7 の入力を使うケースはない(どれも読んだだけで、動かしていない)
5. `ErrInboxResolved` の文: 名指すコマンドは実在し、`--state-dir` も付き、resolve のメモが表示される(上の表の N3)。件が 2 つ以上のとき「the note it was resolved with」が単数のままだが、読み違える文ではないので指摘にしない。残りは N8
6. 文書の面: `docs/quality/quality-gates.md` の 2 つの面は、`PLAN:` のヘッダの手を「ないか形が違えば `TASK_ID:` の次に付ける」と「古いか打ち間違えた digest なら今の digest に直す」の 2 つに分けて書き、コードと合う。0326574e の、先に 2 行を書かせる拒否は、同じ行の「計画の先頭の 2 行を書いてコミットする」に含まれる。4 つの SKILL.md の新しい 2 項目もコードと合う。2 つ目の項目(`SKILL.md:682-686`)は場合を「承認し直す前の digest」に絞って書くが、コードは打ち間違えた digest で 2 行がまだのときも同じ拒否を返す。例が狭いだけで、従っても誤らない。どの面も「古いか打ち間違えた digest」という書き方で、TASK の digest のほうが新しい N7 の入力は書いていない
7. leader.md の手順 5 と 7: 手順 5 の「最後に承認された digest。計画の `- Approved:` の行の digest」は、手順 4 で 2 行を書いてから手順 5 に進む順と合う。手順 7 の「原因を直す前に同じコマンドを打ち直さず」は、種類ごとの次の手(PlanHeader は 2 行を直してから tries again、TaskHeader はヘッダを変えたコマンド)と食い違わない。`prompts_test.go` が両方の語句を固定している。N7 の入力でだけ、手順 5 は最後の承認の digest を、拒否は前の版の digest を名指し、2 つが別の digest を言う

### Findings(3 回目の結果)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 3 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | N6: N1 の直しの最後の枝は、今の digest に承認の記録があり、その承認のあとに digest が読まない部分だけが変わった場合も、TASK の digest の拒否を返す。digest は `## Progress checklist` の節、`- Branch:` の行、先頭の 2 行より後の `- Status:` と `- Approved:` を読まないが、全文のハッシュは読む。承認し直した計画 Z に Progress のメモが足されたあと、古い digest Y で TASK を送ると、まず 2 行が古いので「`sha256:Z` で 2 行を書いてコミットし、送り直す」(0326574e の枝)が来る。書いて送り直すと、今度は Z の検査が全文のハッシュで落ちて最後の枝に入り、Y の PlanChanged「承認のあとに計画が変わった(承認の digest Y、今の digest Z)。変更を戻すか、頼み直す」が来る。Z の承認を書かせた拒否のすぐあとに、Y から Z への改訂を戻させる手が来る。戻せば人が最後に承認した版を捨てて Y で進み、頼み直せば記録のある Z の QUESTION がもう一度人に届く。打ち間違えた digest で同じ状態なら NoApproval の「承認の DECISION を受けていれば記録のない DECISION なので BLOCKED」になり、原因の説明が誤る。doc の「The TASK then goes through in two steps」もこの入力では成り立たない。古いヘッダと、雛形が座席に禁じさせる計画の書き換えの 2 つが重なったときだけで、どの手に従っても行き着くのは承認のある版なので LOW とした | `internal/org/approval.go:494-495`(最後の枝)、`:464-467`(doc)、`:567-580`(digest、2 行、全文のハッシュの順)、`PlanContentHash` の doc(`:271-275`)、`internal/org/prompts/leader.md:144-148`(手順 6: 座席への TASK にも計画を書き換えないよう書く)、`.claude/rules/ralph/ralph-workflow.md:51`(「checkpoint progress in the active plan」。座席のセッションも読む) | 最後の枝の前に、今の digest の拒否が PlanChanged ならそれを返す枝を足す(今の digest に記録がなければ NoApproval になるので、PlanChanged は記録があることを意味する)。その手は Progress のメモだけを戻させ、送り直すと TaskHeader になる。`TestCheckTaskPlanApproved_StaleOrMistypedDigest` に、承認し直したあとに Progress のメモを足したケースを足す |
| LOW | maintainability | N7: N1 の直しが振り替え先にするのは、計画のファイルが今持つ digest で、そのパスの最後の承認ではない。ファイルが前に承認された版 Y に戻っていて(承認し直したあとの revert など)、TASK が最後の承認 Z を名指すと、最初の検査は Z の PlanChanged になり、今の digest は Y になる。2 行が Z のままなら「`sha256:Y` で 2 行を書け」、書けば「ヘッダを `PLAN: <path> sha256:Y` に直して送れ」が来て、leader は人が置き換えた版で進む。直す前は Z の PlanChanged「変更を戻すか、頼み直す」が返り、戻せば Z に戻った。doc は、この枝に来るのはヘッダが「old or mistyped digest」を名指すときで、直しの目的は「the plan a human approved last」を捨てさせないことだと書くが、この入力ではどちらも成り立たない。雛形の手順 5 は TASK のヘッダに最後に承認された digest を書かせるので、手順 5 と拒否が別の digest を名指す。今の命のどの承認でも通るのは計画の決め方どおり(`checkAnyPlanApproved` も同じ)で、承認のない版は通らず、ファイルが前の版に戻されたときだけの話なので LOW とした | `internal/org/approval.go:451-457`(doc)、`:482-486`(`currentPlanRef` と 2 回目の検査)、`:504-518`(`currentPlanRef`)、`checkPlanApproved` の `latest`(`:535-544`)、`internal/org/prompts/leader.md:137-139`(手順 5)、`docs/quality/quality-gates.md:81` と `.claude/skills/org/SKILL.md:679-681`(「古いか打ち間違えた digest」) | TASK の digest がそのパスの最後の承認であるときは、今の digest へ振り替えずに元の拒否を返す(`checkPlanApproved` がすでに `latest` を求めている)。コードを変えないなら、doc と SKILL.md・quality-gates の書き方を「ファイルが今持つ承認済みの版へ振り替える(前の版に戻っていればその版)」にする |
| LOW | exception-handling | N8: N3 の直しの文は、ほかの誰かが resolve した件について「承認の DECISION をまだ leader に送らない」と言うが、承認は DECISION がなくても効いている。記録は残り、今の命の中でそれを取り消す動詞はない。承認の検査は記録と計画の 2 行を読み、DECISION は読まない。PlanHeader の次の手は、NoApproval の手と違って承認の DECISION を受けたかを条件にせず、「2 行を書いてコミットし、送り直す」と言う。そのため、先に resolve した人が直しを頼んでいて、人の次の DECISION より前に leader が TASK を送ると、leader は計画に承認の 2 行を自分で書くよう言われ、そのまま進める。0326574e の枝で、改訂の前の digest の TASK も同じ拒否になる。2 人が同じ件を扱い、さらに leader が雛形に反して DECISION の前に TASK を送ったときだけなので LOW とした | `internal/org/approve.go:404-409`(文)、`internal/org/approval.go:665-667`(NoApproval の手の条件)、`:668-674`(PlanHeader の手)、`:492-493`(0326574e の枝)、`internal/org/prompts/leader.md:120-125`(手順 3 と 4: 返事を待ち、DECISION を受けてから 2 行を書く) | PlanHeader の手に NoApproval と同じ条件を付ける(承認の DECISION をまだ受けていなければ QUESTION の返事を待つ)。コードを変えないなら、approve の文を「承認はもう効いている。メモが直しの依頼なら、承認の DECISION ではなく直しを頼む DECISION を送る」にする |

### Positive notes(3 回目の結果)

- 最初の検査が失敗したあとの枝はどれも error を返し、`checkPlanApproved` と `checkAnyPlanApproved` には手を入れていない。振り替えで通る道は増えず、spawn の拒否の種類と並びも変わらない
- (e) は理由の文を、今の digest での `checkPlanApproved` の error と文字列ごと比べるので、0326574e の枝を消すと落ちる。(e2) は 2 手で通るところまでを、検査と Send の両方で、typed・raw・dry-run の 3 通りで確かめ、断られた前後で台帳と driver の写しが変わらないことも見る
- 次の手の文に手順の番号を書かないことを、コメントの規則だけでなく、全部の種類のケースにかかる正規表現のテストで固定した
- `TestOrgApprove_ResolvedBySomeoneElse` は、一時ディレクトリの名前に別の件の ID が入りうることを踏まえ、`inbox show` のコマンドを取り除いてから ID がないことを確かめる。テストが偶然落ちる道を先に塞いでいる
- 0326574e は `currentApprovedRef` を `currentPlanRef` に改め、この関数は今の digest を読むだけになり、検査は呼び出し元が回す。古い名前はどこにも残っていない

### Tech debt identified(3 回目の結果)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| The stale-`PLAN:` routing of org-plan-approval returns the TASK's own refusal when the plan's current digest is approved but the plan changed after that approval outside its digest. `checkTaskPlanApproved` (`internal/org/approval.go`) moves a TASK whose `PLAN:` names an old or mistyped digest to the file's current digest only when `checkPlanApproved` of that digest passes or fails on the two header lines; when it fails as a changed plan (a `## Progress checklist` note, a `- Branch:` line, or another line the digest skips, written after the approval), its last branch returns the refusal of the TASK's digest. After a re-approval that is "changed after its approval: approved digest <old>, current digest <new>" with the step to undo the change, right after the refusal that had the leader write the new digest's header lines; for a mistyped digest it is the no-approval refusal, whose step says an approving DECISION came without its record. The doc's "The TASK then goes through in two steps" does not hold for this input | A leader that follows the step goes back to the plan version approved before the revision, asks again for an approval the ledger already holds, or escalates a BLOCKED with a wrong cause. Needs a stale or mistyped `PLAN:` plus a plan edit that leader template step 6 has the leader tell its seats not to make; every outcome is a version a human approved | Found by self-review run 3 of org-plan-approval (N6); that run fixes only CRITICAL and HIGH findings | The next change to `checkTaskPlanApproved`: when `checkPlanApproved` of the current digest fails with `approvalRefusalPlanChanged` (which means the current digest has a record), return that refusal, whose step undoes only the edit, and add a re-approved plan with a Progress note to `TestCheckTaskPlanApproved_StaleOrMistypedDigest` | `docs/reports/self-review-2026-10-13-org-plan-approval.md` (N6); `internal/org/approval.go` (`checkTaskPlanApproved`, `checkApprovedPlanFile`, `PlanContentHash`); `internal/org/prompts/leader.md` (step 6); commits 4b6d0d55 and 0326574e |
| The stale-`PLAN:` routing of org-plan-approval follows the plan file's digest, not the plan's latest approval. `checkTaskPlanApproved` (`internal/org/approval.go`) moves the TASK to the digest the file has now (`currentPlanRef`) whenever that digest passes or fails only on the header lines. When the file holds an earlier approved version again (reverted after a re-approval) and the TASK names the latest approval, the refusals have the leader write the earlier digest's header lines and then change the TASK's `PLAN:` to that digest; before this routing the refusal had it undo the change, which restored the latest approval. The doc says the header then names "an old or mistyped digest" and that the routing keeps the leader from undoing "the plan a human approved last", leader template step 5 has the header carry the last approved digest, and the `/org` skill and `docs/quality/quality-gates.md` describe only old or mistyped digests; none of these holds for this input | A leader implements a version the human replaced, while the template and the refusal name different digests. Any approval of the org's current life passes by design (`checkAnyPlanApproved` too), so no unapproved version runs; needs the plan file put back to an earlier approved version | Found by self-review run 3 of org-plan-approval (N7); that run fixes only CRITICAL and HIGH findings | The next change to `checkTaskPlanApproved`: keep the TASK's own refusal when its digest is the latest approval of the path (the `latest` that `checkPlanApproved` already finds), or say in its doc, the `/org` skill, and the quality-gates row that the TASK is moved to whatever approved version the file holds | `docs/reports/self-review-2026-10-13-org-plan-approval.md` (N7); `internal/org/approval.go` (`checkTaskPlanApproved`, `currentPlanRef`, `checkPlanApproved`); `internal/org/prompts/leader.md` (step 5); commits 4b6d0d55 and 0326574e |
| When `ralph org approve` finds that someone else resolved an item while it ran, its error says not to send the leader the approval DECISION yet and to read the note first, but the approval is in effect without the DECISION. `resolveApproved` (`internal/org/approve.go`) keeps the record, and nothing withdraws it in the org's current life; the approval check reads the record and the plan's two header lines, never the DECISION, and the plan-header refusal tells the leader to write the lines and try again whether or not an approving DECISION came (`approvalNextStep`), while the no-approval refusal makes that a condition. A TASK naming the pre-revision digest gets the same plan-header refusal for the new digest (commit 0326574e) | If the other resolver asked for changes and the leader sends a TASK before the human's next DECISION, the refusal has the leader mark the plan approved itself, and the TASK goes ahead. Needs two people on one item and a leader that sends a TASK before the DECISION, which leader template steps 3 and 4 forbid | Found by self-review run 3 of org-plan-approval (N8, the rest of N3 and L3); that run fixes only CRITICAL and HIGH findings | The next change to `approvalNextStep` or `resolveApproved`: give the plan-header step the no-approval step's condition (with no approving DECISION yet, wait for the answer to the QUESTION), or have approve's error say that the approval is in effect and that a DECISION asking for the changes, not the approval DECISION, is what holds the leader back | `docs/reports/self-review-2026-10-13-org-plan-approval.md` (N8; N3 and L3 in its appendices); `internal/org/approve.go` (`resolveApproved`); `internal/org/approval.go` (`approvalNextStep`); `internal/org/prompts/leader.md` (steps 3 and 4); commits 4b6d0d55 and 0326574e |
| The org-plan-approval tests keep their approval helpers twice. `checkFixture` (`internal/org/approval_test.go`: `record`, `markApproved`, `reapprove`) and `enforceFixture` with `TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest` (`internal/org/approval_enforce_test.go`: `record`, `mark`, the `approveRevisionAs` closure) record an approval, record a re-approval, and write the plan's two header lines the same way; commit 9fb3fafa merged only the string-replace helper (`replaceOnce`), commit 4b6d0d55 added the re-approval pair, and neither helper's doc says why there are two | A change to how an approval is recorded or the header lines are written has to be made in both files | Self-review N4 of org-plan-approval was left as is on purpose (the plan's Progress checklist, 2026-10-13); it changes no behavior | The next change to either fixture's approval helpers: one helper per action that takes the manifest and the worktree, or a line in both docs on why the fixtures differ (one has fake drivers) | `docs/reports/self-review-2026-10-13-org-plan-approval.md` (N4 in its run-2 appendix); `internal/org/approval_test.go`; `internal/org/approval_enforce_test.go`; commits 9fb3fafa and 4b6d0d55 |

_(この 4 行は、依頼の指示(この報告のファイル以外は書かない)に従い、`docs/tech-debt/README.md` に足していない。列の順は README の表と同じで、どの行もエスケープしていない `|` は 6 本である。4 行目は 2 回目の結果の N4 で、計画のメモにしか書かれておらず、計画は `/pr` で archive されるので足した。)_

### Recommendation(3 回目の結果)

- 3 回目の結果の判定: yes(CRITICAL と HIGH はなく、この回の止める条件に当たる指摘はない。LOW は N6〜N8 の 3 件で、どれも上の行を tech-debt に足して先へ進められる)
- Follow-ups:
  - N6〜N8 と N4: 上の 4 行を `docs/tech-debt/README.md` に足す(orchestrator か /sync-docs)
  - N6 は枝を 1 つ足せば直るので、cross-review の結果でコードに戻る回があれば一緒に直せる
  - insight event: 同じ cycle の 3 回目なので足していない。phase と cycle で集計するので、足すと self_review の cycle 1 が 3 回数えられる。最初の結果の event は 6d3369a6 にある

## 付録: cycle 1 の 2 回目の結果(02019f48、HEAD 1b95bb84 の時点)

2 回目の結果の本文を、見出しを 1 段下げて名前に「(2 回目の結果)」を足し、`Merge:` の行の名前を「2 回目の結果の判定」に変えたほかは、そのまま残す。本文の中の「付録」は、この報告の最後の節を指す。

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)の 2 回目の実行(run 2)。最初の結果(3d73dbf4、HEAD ef503bbd の時点)の指摘を直した 54feb287..1b95bb84 のあとに見直した。cross-review の前の直しなので cycle は 1 のまま。最初の結果は末尾の「付録」にそのまま残した(コードのコメントとテストの doc が「self-review M1 of org-plan-approval」の形でその ID を指すため)
- Scope: diff の品質だけ。ブランチ全体 `git diff 225dff27...HEAD`(HEAD 1b95bb84)を対象にし、直しの範囲 `git diff 6d3369a6..HEAD`(4 コミット、22 ファイル、+1510/-985)を重点に見た。テスト・静的解析・仕様適合・文書のずれの検査は /test・/verify・/sync-docs が受け持つので、ここでは行っていない
- 番号の付け方: 最初の結果の M1、L1〜L11 はそのまま使う。今回足した指摘は N1〜N5

### 最初の結果の指摘の扱い(HEAD 1b95bb84)(2 回目の結果)

| ID | 状態 | 根拠 |
| --- | --- | --- |
| M1 | 9fb3fafa で直った(残りは N1) | 拒否の文は、見つけた状態、目印 `refused by the approval check`、理由の種類ごとの次の手 1 つ、leader がしないこと、の順になった(`planNotApprovedError.Error`、`internal/org/approval.go:624-631`)。理由の部分から次の手の文は消えた。`TestPlanNotApprovedError_OneNextStepPerKind` が 5 種類すべてについて、理由に次の手の語がないことと、次の手がその種類の文だけであることを確かめる。雛形の手順 7 は「目印の後ろの次の手に従う」に変わった。ただし、TASK の `PLAN:` の digest が古いときの種類の決め方が原因に合わない(N1) |
| L1 | 9fb3fafa で tech-debt に移した | 勧めたとおりコードは変えず、`docs/tech-debt/README.md` の 213 行目(tripwire の行)の Debt item に、検査が role の名前 `implementer` と TASK の `PLAN:` だけを読むことと、custom role の座席・reviewer への TASK・本文で別の作業を頼む TASK の 3 つの道を足した |
| L2 | 9fb3fafa で直った | `approvedLineDigest`(`internal/org/approval.go:244-257`)は、`- Approved:` の値が実在する日付 `YYYY-MM-DD` と `sha256:<12 桁の小文字の hex>` の 2 語のときだけ digest を返す。`/plan` の SKILL.md(99 行目)と leader の雛形の手順 4(127 行目)が書く形はどちらもこの形で、docs/plans の既存の計画の `- Approved:` 行もすべて 2 語だった。`TestApprovedLineDigest` の 15 ケースと `TestCheckPlanApproved` の 2 ケースがある |
| L3 | 半分直った(残りは N3) | 先にほかの誰かが resolve した件は、`ErrInboxResolved` を見分けて「承認は記録してある、resolve するものは残っていない」と言い、失敗する `inbox resolve` のコマンドを出さない(`resolveApproved`、`internal/org/approve.go:382-408`)。approve と、直しを頼むつもりの resolve が競合したときに承認が記録される半分は直っておらず、tech-debt にもない |
| L4 | 54feb287 で直った | approve の動詞を `internal/org/approve.go`(408 行)に、そのテストを `approve_test.go` に移した。削除した行と追加した行を取り出して比べると、違いは package 行、import、ファイル頭のコメントだけで、コードは同じだった(Evidence を参照)。approval.go は 635 行になった |
| L5 | 9fb3fafa で直った | `ApprovedPlan.Resolved` と `ApproveResult` の doc に、CLI は `Approved` と `Err` だけを読み、`Resolved` と `Refused` はテストと件ごとの結果がほしい呼び出し元のための欄だと書いた |
| L6 | 9fb3fafa で直った | CLI は `os.Getwd()` を呼ばず、`ApproveParams.Cwd` を空のまま渡す(`internal/cli/org_approve.go:64-66`)。作業ディレクトリは `Approve` が読む |
| L7 | 9fb3fafa で直った | `spawn.go:555-565`、`reserve.go:526-538`、`prompts_test.go` の該当の段落を折り返し直した |
| L8 | 半分直った(残りは N4) | `TestCheckPlanApproved` の中の `replace` は消え、`replaceOnce` を使う。`checkFixture` の `record`・`markApproved` と `enforceFixture` の `record`・`mark` の重なりは残り、残した理由は計画のメモにもない |
| L9 | 9fb3fafa で直った | `templates/base/ralph.toml:127-129` は、`[org.approval]` がなければ既定の per-plan になり `start --plan` の org が `ralph org approve` を待つことと、元の動きに戻すには `mode = "split"` にすることを書く |
| L10 | 9fb3fafa で直った | `taskNotSentError` の欄の名前は `refusal` になった。`taskNotSentError{gate` と `planNotApprovedNext` を grep すると 0 件(`watchgate_positions_test.go` の `gate.gate` は別の型) |
| L11 | 9fb3fafa で直った | 雛形の前置きは「手順 1・4・6 の機能の計画」になり、`prompts_test.go` の固定の語句も合わせてある |

### Evidence reviewed(2 回目の結果)

- 直しの範囲で読んだもの: `internal/org/approval.go`(新しい `approvalRefusalKind`、`approvalCheckError`、`checkTaskPlanApproved`、`checkApprovedPlanHeader`、`approvedLineDigest`、`approvalNextStep`、`planNotApprovedError`)、`internal/org/approve.go`(`ApproveParams` から `resolveApproved` まで)、`internal/cli/org_approve.go`、`internal/org/verbs.go`(`Send` の中の検査の位置、`sendApprovalErr`、`taskNotSentError`)、`internal/org/spawn.go`(`spawnFeatureSeatErr` とその 3 つの呼び出し元)、`internal/org/prompts/leader.md` の「機能ごとの org」節、4 つの `/org` SKILL.md の承認の節、`docs/quality/quality-gates.md` の新しい行、`docs/tech-debt/README.md` の 213 行目と 217 行目、`templates/base/ralph.toml` の `[org.approval]`、1b95bb84 で計画の Progress checklist に足したメモ
- テストの差分は書き方だけを見た: `approval_test.go`(`TestApprovedLineDigest`、`TestPlanNotApprovedError_OneNextStepPerKind`)、`approve_test.go`(`TestOrgApprove_ResolvedBySomeoneElse`)、`approval_enforce_test.go`、`prompts_test.go`
- 機械的に確かめたこと
  - 54feb287 が移しただけであること: `git show 54feb287 -- internal/org/approval.go` の `-` 行(375 行)と、`-- internal/org/approve.go` の `+` 行(392 行)を取り出して `diff` した。違いは追加の側の package 行、import の 11 行、ファイル頭のコメント 3 行と空行だけだった。テストも同じ方法で比べ、違いは package 行、import、2 つのファイルの頭のコメントだけだった。コミットの本文は新しいファイルのコメントが 2 つとだけ書くが、`approval_test.go` の頭のコメント 4 行も書き換わっている。中身の違いではないので指摘にしない
  - 拒否を作る場所: `planNotApprovedError` を作るのは `spawnFeatureSeatErr`(`internal/org/spawn.go:1294`)と `sendApprovalErr`(`internal/org/verbs.go:503`)の 2 か所だけである。`spawnFeatureSeatErr` は本番のロックの中(`spawn.go:546`)、dry-run(`:1192`)、`spawnPrecheckErr`(`:1227`)の 3 か所から同じ events で呼ばれる。Send の検査は dry-run の枝より前にある(`verbs.go:316-329`)。種類は `checkPlanApproved` の各 return と `checkApprovedPlanHeader` で決まり、`checkAnyPlanApproved` は最後の承認の種類と digest を受け継ぐ。種類を持たない error は Other(BLOCKED)に倒れる
  - 直しの範囲で足した Go のコメントにある CamelCase の識別子 77 個は、どれもコメント以外の行に定義か使い道がある(`git grep -w`)
  - 4 つの `/org` SKILL.md は md5 が同じ。`docs/quality/quality-gates.md` と `templates/base/` の写しは、新しい行(81 行目と 80 行目)が同じで、ほかの違いはこの PR より前からある 54 行目と 57 行目の 2 か所だけ。2 つの `agent-messaging.md` は `cmp` で同じ
  - tech-debt の 213 行目と 217 行目は、エスケープしていない `|` がどちらも 6 本で、列は崩れていない。217 行目は (d) を Debt item、Impact、Trigger のどこからも消してある。この PR で足した行なので、取り消し線ではなく消すのでよい
  - 足したコメントの行はどれも 80 桁以内(awk が 81 と数えた 1 行は多バイト文字のため)
  - HEAD は読み始めと書く前のどちらも 1b95bb84 で、作業ツリーはきれいだった
- 確かめていないもの: テスト、静的解析、実機の確認はしていない。N1 と N3 はコードを読んで組み立てたもので、動かしてはいない

### 依頼された点の結論(2 回目の結果)

1. 種類ごとの次の手は、検査が見つけたことに合っているか: 5 種類のうち 4 つは合っている。PlanHeader(先頭の 2 行)は、記録があり、digest も全文のハッシュも合うときにしか出ない。2 行はどちらのハッシュにも入らないので、書き直して打ち直せば通る。TaskHeader は `ParsePlanHeader` の誤りだけで、ヘッダを直して送り直せば次の段へ進む。Other は結びつきの不完全、worktree の外を指す symlink、ファイルがないか読めないときで、leader は自分で直せないので BLOCKED でよい。合わないのは、TASK の `PLAN:` が古い digest か打ち間違えた digest を名指すときである。この場合は NoApproval か PlanChanged に振り分けられ、どちらの次の手も実際の原因(ヘッダの digest)に向かない(N1)
2. 手順 7 に従う leader が止まらなくなるか: N1 の (a)、承認し直した計画に古い digest の TASK を送る場合がそうなる。次の手の「新しい QUESTION を上げる」に従っても TASK のヘッダは変わらないので、TASK を送るたびに同じ拒否が繰り返され、そのたびに人に QUESTION が届く。もう一方の「変更を戻す」に従うと、人が最後に承認した計画が古い版に戻り、古い承認(同じ命の中なので数えられる)で implementer が進む。ほかに、NoApproval の「QUESTION の返事を待つ」は、QUESTION を上げていない leader にとっては終わりがない。これは手順 3 を飛ばしたときだけで、deadman の escalation が人に知らせるので、指摘にはしない
3. 5 種類の振り分け: Evidence のとおり、6 つの場所(`checkTaskPlanApproved`、`checkAnyPlanApproved`、Send、Spawn の本番、dry-run、`spawnPrecheckErr`)はどれも `sendApprovalErr` か `spawnFeatureSeatErr` を通るので、同じ入力には同じ種類がつく。spawn には `PLAN:` がないので TaskHeader は出ず、`--raw` の一文も付かない(`planNotApprovedNeverRaw` は `task` のときだけ付く)
4. `- Approved:` を厳しく読むことで、`/plan` か雛形が書く行を断るか: 断らない。どちらも `<YYYY-MM-DD> sha256:<digest>` を書き、`scripts/plan-visual.sh digest` は `sha256sum` の小文字の hex を出す。断るのは `TBD`、語を足した行、存在しない日付、大文字の hex で、どれも雛形の手順では書かない。ただ `/implement` は同じ行の `sha256:` の値だけを見るので、`/implement` が承認済みとみなす計画の一部を org の検査は断る。このずれを `featurePlanHeaderLines` の doc が言っていない(N5)
5. L3 の `ErrInboxResolved` の文: 失敗するコマンドを出さなくなった点は直った。ただし、先に resolve した人が直しを頼むつもりだった場合に、その人のメモを読む手を示さない(N3)
6. 54feb287: Evidence のとおり、移しただけである
7. 文書の面: 4 つの SKILL.md、quality-gates.md の 2 つの面、tech-debt の 213 行目と 217 行目、`templates/base/ralph.toml` は、コードの 5 種類と次の手、`--raw` の一文が TASK だけに付くこと、`- Approved:` の形と合っている。合わないのは、雛形と SKILL.md の「同じコマンドをそのまま打ち直さず」と、PlanHeader の次の手の「打ち直す」の組み合わせである(N2)

### Findings(2 回目の結果)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | readability | N1: TASK の `PLAN:` が、今のファイルのものでない digest を名指すと、拒否の種類は検査が最初につまずいた段で決まり、次の手が原因に向かない。(a) 手順 6 で計画を直して新しい digest Z の承認を受けたあと、前の digest Y のヘッダで TASK を送ると、Y の記録があるので PlanChanged になり、次の手は「変更を戻すか、Draft に戻して新しい QUESTION を上げる」になる。Z の記録が同じ命にあることを、検査は読まない。戻せば人が最後に承認した計画を捨てて Y で進み、頼み直せば承認済みの計画の QUESTION が人に届く。どちらでもヘッダは Y のままなので、次の TASK も同じように断られる。(b) digest を打ち間違えると NoApproval になる。理由の文は「(its latest approval is of sha256:Z)」と書くのに、DECISION を受けた leader への次の手は「その DECISION は記録なしに来たので BLOCKED を上げる」で、記録はあるので原因の説明が誤っている。雛形の手順 5 の「手順 4 で承認された digest」も、承認し直したあとにどちらの digest を使うかを言わない。どちらに転んでも承認のない計画では進めないので、安全の問題ではない | `internal/org/approval.go:465-483`(`matching` は TASK の digest の記録だけを集め、`latest` は理由の文にしか使わない)、`:498-501`(PlanChanged)、`:592-594` と `:601-603`(次の手)、`internal/org/prompts/leader.md:138`(手順 5)、`:148-150`(手順 6) | `checkPlanApproved` で、TASK の digest に記録がないとき、またはファイルがその digest から変わっているときに、同じパスに今のファイルの digest の記録があるかを見る。あれば TaskHeader の種類にし、次の手で「`PLAN: <path> sha256:<今の digest>` に直して送り直す」と digest を名指す(`approvalCheckError.digest` が使える)。コードを変えないなら、NoApproval と PlanChanged の次の手の先頭に「まず TASK の `PLAN:` が最後に承認された digest かを確かめる」を足し、手順 5 を「最後に承認された digest」にする。どちらの場合も、`TestPlanNotApprovedError_OneNextStepPerKind` に、承認し直した計画と古いヘッダの組を足す |
| LOW | readability | N2: 打ち直しを禁じる文と、打ち直しを求める次の手が、同じ拒否について並ぶ。雛形の手順 7 と SKILL.md は「どの場合も、同じコマンドをそのまま打ち直さず」と書き、PlanHeader の次の手は「2 行を書いてコミットし、tries again」と言う。spawn では打ち直すのは引数も同じ `ralph org spawn` なので、「そのまま」の読み方によっては禁じられた操作になる。ralph が出す拒否の文の終わり(`planNotApprovedNever`)は打ち直しに触れないので、この禁止は雛形と SKILL.md にしかない。また PlanChanged の次の手は、雛形の手順の番号を「step 6 of its procedure」と Go の文字列に書いている。雛形の番号が変わってもテストは気づかない(この PR でも番号は 1〜14 に繰り下がった) | `internal/org/prompts/leader.md:154`、`.claude/skills/org/SKILL.md:673-674` と `:682`(写しの 3 つも同じ)、`internal/org/approval.go:599-600`(tries again)、`:603`(step 6) | 雛形と SKILL.md の文を「次の手をする前に、同じコマンドを打ち直さず」にする。「step 6 of its procedure」は、手順の中身(Draft に戻して新しい QUESTION)がすでに同じ文にあるので番号を消すか、`prompts_test.go` で Go の文字列の番号と雛形の手順の頭を突き合わせる |
| LOW | exception-handling | N3: L3 の残り。approve が受信箱を読んだあと、別の人が直しを頼むつもりで件を resolve すると、承認はそのまま記録され、approve は「承認は記録してある、resolve するものは残っていない」と言って終了コード 1 を返す。CLI は同じ件の DECISION のコマンドも出すので、approve した人がそれを打つと、leader には同じ `PLAN:` の承認の DECISION と直しの DECISION が両方届く。文は、先に resolve した人のメモを読む手(`ralph org inbox show <ID>`)を示さない。この競合は tech-debt にもない。テストは競合する resolve のメモに「handled by hand」を使うが、その場合の扱いは確かめていない。2 人が同じ件を同時に扱うときだけの話である | `internal/org/approve.go:399-401`、`internal/cli/org_approve.go:104-106`(承認した件ごとに DECISION のコマンドを出す)、`internal/org/approve_test.go:432` と `:447` | 文に「`ralph org inbox show <ID>` で先に resolve した人のメモを読んでから、下の DECISION を送る」を足す。直さないなら tech-debt に 1 行足す |
| LOW | maintainability | N4: L8 の残り。`checkFixture` の `record`・`markApproved` と `enforceFixture` の `record`・`mark` は、中身がほぼ同じまま残った。9fb3fafa の本文と計画のメモは、L8 を直したとだけ書く | `internal/org/approval_test.go:483-497`、`internal/org/approval_enforce_test.go:44-61`、9fb3fafa の本文の「L8: TestCheckPlanApproved uses replaceOnce instead of its own helper」 | 2 つの fixture の土台が違う(fake の driver があるかどうか)ので分けておくなら、その理由を helper の doc に書く。まとめるなら、承認の記録と 2 行を書く helper を、manifest と worktree を引数に取る関数 1 つずつにする |
| LOW | readability | N5: `featurePlanHeaderLines` の doc は、2 行を「/implement の承認の検査が計画を読むのと同じように」読むと書く。L2 の直しで、org の検査は `- Approved:` の値を 2 語に限った。一方 `/implement` は同じ行の `sha256:` の値だけを見るので、`- Approved: 2026-10-12 sha256:<digest> (再承認)` のような行を `/implement` は通し、org の検査は断る。行の探し方は同じで、値の読み方が違うことを doc が言っていない | `internal/org/approval.go:191-195`、`.claude/skills/implement/SKILL.md:34`、`approvedLineDigest`(`internal/org/approval.go:237-257`) | doc を「行の探し方は /implement と同じ(最初の `## ` より前の最初の行)で、`- Approved:` の値は approvedLineDigest がより厳しく読む」とする |

### Positive notes(2 回目の結果)

- 種類を error の型(`approvalCheckError`)で運び、理由の文と次の手を別の場所で作るようにしたので、理由の文に次の手が混ざらないことを、テストが種類ごとに確かめられる。種類を持たない error は zero value の Other(BLOCKED)に倒れる
- `TestPlanNotApprovedError_OneNextStepPerKind` は、どの次の手の文がどの種類に出るかを全部の組み合わせで確かめ、最後に 5 種類すべてにケースがあることを数える。次の手を 1 つ足してケースを足し忘れると落ちる
- `TestOrgApprove_ResolvedBySomeoneElse` は `Org.Now` を差し替え、受信箱を読んだあと resolve の前に別の resolve を割り込ませる。割り込みが起きなければ `t.Fatal` で落ちるので、テストが空振りしない
- 54feb287 は移すだけのコミットで、直しは次の 9fb3fafa に分けてある。そのため、移しただけであることを `diff` 1 回で確かめられた
- `--raw` の一文を TASK の拒否だけに付け、spawn の拒否から外した。tech-debt の 217 行目の (d) はこれで閉じ、行からも消えている

### Tech debt identified(2 回目の結果)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| N1(直さないときだけ): TASK の `PLAN:` が今のファイルのものでない digest を名指すと、承認の検査は NoApproval か PlanChanged で断り、次の手が「変更を戻す」「新しい QUESTION を上げる」「BLOCKED を上げる」のどれかになる。同じパスに今の digest の記録があっても読まない | 計画を承認し直したあとに古いヘッダで TASK を送る leader が、承認済みの計画を古い版に戻すか、TASK を送るたびに同じ QUESTION を人に上げる | この cycle で直すかは orchestrator が決める | 次に `checkPlanApproved` か `approvalNextStep` を変えるとき、または leader が承認済みの計画の QUESTION を上げ直した例が出たとき | この報告の N1、`internal/org/approval.go`(`checkPlanApproved`、`approvalNextStep`) |
| N2〜N5(直さないときだけ): 打ち直しの文の食い違いと Go の文字列の手順の番号(N2)、approve と直しを頼む resolve の競合で先に resolve した人のメモを示さないこと(N3)、テストの fixture の helper の重なり(N4)、`featurePlanHeaderLines` の doc の /implement との比べ方(N5) | N3 は、2 人が同じ件を同時に扱うと leader に食い違う DECISION が届く。ほかは読み手が迷うだけで、挙動は変わらない | どれも LOW で、挙動を変えるのは N3 だけ | 次にそれぞれのファイル(`internal/org/prompts/leader.md` と `/org` SKILL.md、`internal/org/approve.go`、`internal/org/approval_test.go` と `approval_enforce_test.go`、`internal/org/approval.go`)を変えるとき | この報告の N2〜N5 |

_(この 2 行は、依頼の指示(この報告のファイル以外は書かない)に従い、`docs/tech-debt/README.md` に足していない。直さずに残すなら orchestrator が足す。)_

### Recommendation(2 回目の結果)

- 2 回目の結果の判定: yes(CRITICAL と HIGH はない。MEDIUM は N1 の 1 件で、PR の前に直すことを勧めるが、止める理由にはしない。LOW は N2〜N5 の 4 件)
- Follow-ups:
  - N1: `checkPlanApproved` で同じパスの今の digest の記録を見て、TASK のヘッダを直す次の手を出す。直さないなら、次の手と雛形の手順 5 の文を直し、上の 1 行目を tech-debt に足す
  - N2〜N5: この cycle でまとめて直せる。直さないなら上の 2 行目を tech-debt に足す
  - /verify への申し送り: 計画の AC10 の読み方と、Scope・Risks の「BLOCKED を escalate する」の書き方は、1b95bb84 のメモ(逸脱 2)が扱っている
  - insight event: 同じ cycle の 2 回目なので足していない。phase と cycle で集計するので、足すと self_review の cycle 1 が 2 回数えられる。最初の結果の event は 6d3369a6 にある

## 付録: cycle 1 の最初の結果(3d73dbf4、HEAD ef503bbd の時点)

最初の結果の本文を、見出しを 1 段下げ、`Merge:` の行の名前を「最初の結果の判定」に変えたほかは、そのまま残す。

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: `git diff 225dff27...HEAD`(HEAD ef503bbd)。コミットは f779cc56(設定と結びつきの `approval=`)、c08a1a14(`internal/org/approval.go`)、e3d34062(Send と Spawn の拒否)、5596561a(`ralph org approve`)、36b9d3a3(leader の雛形と文書)、ef503bbd(計画のメモ)。見たのは差分の質だけで、仕様との突き合わせ、テストの網羅、文書のずれは `/verify` と `/test` に任せる

### Evidence reviewed(最初の結果)

- 全文を読んだコード: `internal/org/approval.go`(853 行)、`internal/cli/org_approve.go`
- 差分を読み、周りのコードも読んだもの: `internal/org/verbs.go`(`Send`、`sendApprovalErr`、`seatRoleFromEvents`、`seatFromEvents`、`taskNotSentError`)、`internal/org/spawn.go`(`Spawn` の doc と dry-run の枝、`checkCapacityAndStart`、`spawnPrecheckErr`、`spawnFeatureSeatErr`、`idempotentRespawn`、`reject`、`checkSpawnInput`)、`internal/org/reserve.go`(`FeatureBinding`、`RequiresApproval`、`featureBindingFromTokens`、`sameFeature`、`validateFeatureBinding`、`ActiveFeature`、`lastDisbandedIndexes`)、`internal/org/feature.go`、`internal/org/protocol/protocol.go`(`Parse` と新しい `HeaderValues`)、`internal/org/split.go`(`PlanDigest`、`planLines`、`checkedBoxIndex`、`approvedDigestFrom`)、`internal/org/manifest.go`(`Roster`)、`internal/org/inbox.go`(`PrintableInboxText`)、`internal/cli/org.go`(`withMainWorktreeOrgLimits`)、`internal/cli/org_inbox.go`(`newOrgInboxRuntime`)、`internal/config/config.go`、`scripts/ralph-config.sh`
- 文書の差分で読んだもの: `internal/org/prompts/leader.md`、`templates/base/ralph.toml`、`docs/tech-debt/README.md` の新しい 5 行、`docs/quality/quality-gates.md`、`.claude/rules/ralph/agent-messaging.md`、`/org` skill の追加行のうち承認に触れる行
- テストの差分は、helper の重複と書き方だけを見た(`internal/org/approval_test.go`、`internal/org/approval_enforce_test.go`、`internal/cli/org_approve_test.go`、`internal/org/prompts_test.go`、`internal/cli/org_feature_test.go`)
- 機械的な確認:
  - 追加行にデバッグ出力・TODO・secret らしい文字列はない(`git diff ... | grep '^+'` で `TODO|FIXME|fmt.Print|println|t.Log|api_key|token=|password` を探して 0 件)
  - 新しい `func` の直上の doc はどれも、その関数の直上にある
  - `docs/tech-debt/README.md` の新しい 5 行は、エスケープしていない `|` がどれも 6 本で、列は崩れていない
  - `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` は `cmp` で一致する。`RALPH_ORG_APPROVAL_MODE` は export の行に入っていない
  - `approval.go` の関数で 50 行を超えるものはない(いちばん長いのは `Approve` の 43 行)。`org_approve.go` の `newOrgApproveCmd` は 60 行だが、その大半は `Long` の文字列である
  - S1 で implementer が `gofmt -w` をかけた `internal/cli/org_feature_test.go` の差分は追加の 95 行だけで、既存の行の書式だけの変更は入っていない
- 確かめていないもの: テスト、静的解析、実機での確認はしていない。下の指摘のうち、競合(L3)と迂回(L1、L2)は、コードを読んで組み立てたもので、動かしてはいない

### 迂回と誤判定の確認(依頼された観点)(最初の結果)

1. パスの解決と symlink: `normalizePlanPath` は空、絶対パス、制御文字、`..` の要素、`.` を断り、`filepath.Clean` と `ToSlash` で揃える。`resolvePlanFile` は worktree とファイルの両方を `EvalSymlinks` で解き、`filepath.Rel` が `..` で始まれば断り、普通のファイルでなければ断る。worktree の中から外を指す symlink は断られる。ファイルを解いたあと `os.ReadFile` までの間に差し替える TOCTOU は残るが、A2 のとおり境界ではないので指摘にしない
2. ヘッダの読み方: `HeaderValues` は `Parse` と同じ規則(1 行目が `TYPE:`、空行か `KEY: value` でない行で終わる、キーと値を trim する)で読む。`PLAN:` が 2 行あれば断るので、検査と座席が読む行はずれない。キーの大文字小文字(`Plan:`)と本文に書いた別の計画は検査の外で、L1 に含めた
3. 前の命の承認: `currentPlanApprovals` は最後の実の `disbanded` より後の、org 全体の(seat_id が空の)実の `plan_approved` だけを読む。読めない Details の記録は数えない。disband の補償で承認が数えられなくなる件は、閉じる側に倒れており、tech-debt の行にもある
4. 読めない方式の印: `approvalModeUnreadable` は `per-plan` でも `split` でもないので、`RequiresApproval` は承認を求める。新しい結びつきには `validateFeatureBinding` が使わせない。補償で書き戻しても同じ値が読み戻される。`approval=` のない記録だけが split として通り、これは A8 で決めた扱いである。古い版の補償が結びつきを書き直すと `approval=` が落ちる(古い版はこのトークンを note として読み、書き直すときに note を落とす)が、版を混ぜたときだけの話で、release のノートが扱う範囲と読んだ
5. split と古い記録の枝: `sendApprovalErr` と `spawnFeatureSeatErr` は、記録した方式だけを読み、`o.Config` を読まない。leader の spawn だけを外す条件は `SeatID == leader && Role == leader` の両方で、片方だけでは外れない
6. `Org.Approve` の書き込み: ロックを 1 回取り、その中で manifest を読み直し、全部の件を確かめてから追記する。全文のハッシュは、digest を確かめたのと同じ読み込みから取る。1 件でも断られれば何も追記しない。追記の途中で I/O が失敗したときは、記録した件と残りの件と打ち直すコマンドを返し、`ApproveResult` の doc もそう書いている。resolve はロックの外で、失敗しても記録は残る。受信箱をロックの外で読むことは L3 に書いた
7. 座席の role の引き方: `seatRoleFromEvents` は role を持つ最後の状態の event を読むので、`spawn_step` で止まった座席も implementer として検査される。`rejected` が role を上書きしても、`Roster` の最後の状態の event に pane がなくなるので、`Send` の本番の枝は pane のない座席として断る。この順で検査を外す道はない

### Findings(最初の結果)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | readability | M1: 承認の検査の拒否の文で、理由の部分と最後の決まり文句が、leader に別々の次の手を言う。理由の部分は「新しい QUESTION で承認を頼み直す」「先頭の 2 行を書いてコミットする」「変更を戻すか、頼み直す」と言い、続く `planNotApprovedNext` は「打ち直さず、`--raw` にせず、approve を打たず、BLOCKED を escalate する」と言う。記録はあるが leader が 2 行を書き忘れた場合、leader は自分で直せるのに、同じ文の後ろ半分が打ち直しを禁じる。leader は LLM で、エラーの文を字義どおりに読む。leader の雛形も、手順 6 は計画を変えたら新しい QUESTION、手順 7 は断られたらいつも BLOCKED と書いており、どちらに従うかが決まらない。どちらに転んでも承認なしでは通らないので、安全の問題ではない | `internal/org/approval.go:381`(記録がないとき)、`:400`(digest が変わったとき)、`:403-418`(2 行がないか違うとき。`write := ...` の文)、`:425`(全文のハッシュが違うとき)、`:465`(`planNotApprovedNext`)、`internal/org/prompts/leader.md` の手順 4・6・7 | 次の手を書く場所を 1 つにする。理由の部分は状態だけを言い(何が足りないか)、次の手は理由ごとに変わる末尾に移す。たとえば 2 行がないときは「2 行を書いてコミットし、送り直す」、digest か全文のハッシュが違うときは「手順 6 のとおり新しい QUESTION を上げる」、記録がないときは「返事を待つか BLOCKED を上げる」。leader の雛形の手順 7 も同じ分け方に合わせる。tech-debt の (d)(spawn の拒否が `--raw` に触れる)も同じ直しで閉じられる |
| LOW | security | L1: 検査は role の名前 `implementer` だけを見るので、別の名前の role で実装させると承認を通らずに済む。`checkSpawnInput` は任意の role を受け、`--prompt` を付ければ雛形のない custom role として起動する(`spawn.go:1375` の doc がそう書く)。その座席への TASK も `seatRoleFromEvents` が implementer を返さないので検査されない。ほかに、reviewer への TASK(A5 で通す)で実装を頼むことも、`PLAN:` には承認済みの計画を書いて本文で別の作業を頼むこともできる。A2 のとおり境界ではないが、tech-debt の tripwire の行は、迂回の道として `env -u HERDR_PANE_ID` と台帳への直接の書き込みしか挙げていない | `internal/org/spawn.go:1293`、`internal/org/verbs.go:498`、`internal/org/spawn.go:1371-1376`、`docs/tech-debt/README.md` の新しい行のうち `approveTripwire` の行 | コードは変えず、tech-debt の tripwire の行の Debt item に「検査は role の名前と `PLAN:` のヘッダを見るだけで、座席がする作業は見ない」と、上の 3 つの道を足す |
| LOW | security | L2: A9 の残り。全文のハッシュは、見出しの前の最初の `- Approved:` の行をまるごと飛ばし、検査はその行の最初の `sha256:` の値だけを読む。そのため承認のあとに、その 1 行の日付の位置や digest の後ろに文を書き足しても、検査は通る(例: `- Approved: 2026-10-13 sha256:<digest> あわせて X も実装する`)。`- Status:` は値が `Approved` と完全に一致する必要があるので、こちらは閉じている | `internal/org/approval.go:229`(`approvedDigestFrom` に渡すだけ)、`internal/org/split.go:463-473`、`PlanContentHash`(`approval.go:250-264`) | `readFeaturePlanHeader` で、`- Approved:` の値が `<日付> sha256:<digest>` の 2 語だけであることを求める。直さないなら、skill の承認の節か tech-debt に、この 1 行が残ることを書く |
| LOW | exception-handling | L3: `Approve` は受信箱をロックの外で 1 回読むだけで、ロックの中で件の状態を読み直さない。同じ件に 2 人が同時に approve を打つと、どちらも記録し(`plan_approved` が 2 行になる)、遅いほうの resolve は失敗する。そのとき出す案内 `ralph org inbox resolve <ID> --note ...` は、件がもう resolved なので、打っても失敗する。approve と、人が直しを頼むつもりで打った `ralph org inbox resolve` が競合したときも、承認が記録される。どちらも窓は短い | `internal/org/approval.go:568`(受信箱を読む)、`:603-621`(`recordUnderLock` は manifest だけを読み直す)、`:836-853`(`resolveApproved` の案内) | ロックの中で追記の直前に受信箱を読み直して状態を確かめるか、resolve の失敗が `ErrInboxResolved` のときは「ほかの誰かが resolve した。承認は記録してある」と言い分け、打っても失敗するコマンドを案内しない |
| LOW | maintainability | L4: `approval.go` が 853 行で、ファイルの目安の 800 行を超える。中身は 3 つに分かれている: `PLAN:` のヘッダとパスの読み方とハッシュ、検査(`checkPlanApproved` など)、approve の動詞(`ApproveParams` から `resolveApproved` まで、373 行) | `wc -l internal/org/approval.go` | approve の動詞とその helper を `internal/org/approve.go` に分ける。どちらも 600 行を切る |
| LOW | maintainability | L5: `ApproveResult.Refused` と `ApprovedPlan.Resolved` を読むのはテストだけである。CLI は `Approved` と `Err` しか読まない | `internal/cli/org_approve.go:72-78`(`printApprovedPlans` に渡すのは `res.Approved`、返すのは `res.Err`)、`grep -rn 'Refused\|Resolved' internal/cli/org_approve.go` が 0 件 | CLI の出力に使う(たとえば resolve できなかった件に印を付ける)か、テストのための欄だと doc に書く |
| LOW | unnecessary-change | L6: 作業ディレクトリの取り方が 2 か所にある。CLI は `os.Getwd()` が失敗すると `""` を渡し、`Approve` は `""` を受けると `os.Getwd()` をもう一度呼ぶ。同じ呼び出しを 2 回試すだけで、2 回目が成功する場面はほぼない | `internal/cli/org_approve.go:65-71`、`internal/org/approval.go:574-581` | どちらか一方にする。CLI は `ApproveParams.Cwd` を空のまま渡し、`Approve` に任せれば、CLI 側の 4 行と注記が消える |
| LOW | readability | L7: コメントの折り返し直しが途中で止まっている。`spawn.go` は `appendEvent" window` のあとで行が短く切れ、次の行に `docs/tech-debt/README.md` だけがある(切れ方は base にもあったが、この差分で 3 行目が増えた)。`reserve.go` は `before. When orgID holds none, a want with` で短く切れる。`prompts_test.go` には、ほかより長い行が 1 行ある | `internal/org/spawn.go:557-560`、`internal/org/reserve.go:531`、`internal/org/prompts_test.go:467` | 段落ごとに折り返し直す |
| LOW | maintainability | L8: 同じ package のテストに、同じ働きの helper が重なっている。「置き換える文字列がなければ panic する」置き換えが `replaceOnce` とテストの中の `replace` の 2 か所にあり、`checkFixture` の `record`・`markApproved` と `enforceFixture` の `record`・`mark` もほぼ同じ中身である | `internal/org/approval_enforce_test.go:44-74`、`internal/org/approval_test.go:451-475` | `replaceOnce` と、承認の記録と 2 行を書く helper を 1 つずつにまとめる |
| LOW | readability | L9: `templates/base/ralph.toml` の `[org.approval]` のコメントが「Absent [org.approval] is fully compatible」と書く。`[org.director]` の節から写した決まり文句だが、この節は、ないと既定の `per-plan` になり、既存のプロジェクトの `start --plan` の org は `ralph org approve` を待つようになる。同じコメントの 2 行上は、`split` が「as before this setting existed」だと書いている。`ralph init` で配る面にある | `templates/base/ralph.toml:127-128` | 「Absent [org.approval] loads as the default per-plan; set mode = "split" to keep the behavior from before this setting」のように、既定で動きが変わることを書く |
| LOW | naming | L10: `taskNotSentError` の欄の名前が `gate` のままで、見張りの拒否のほかに承認の検査の拒否も入るようになった。doc は直してあるが、欄の名前だけが古い | `internal/org/verbs.go:226` | `refusal` などに改める(同じ package の中だけの変更) |
| LOW | readability | L11: leader の雛形の前置きは、予約の外に書くものとして「手順 1 と 4 の機能の計画」を挙げるが、手順 6(計画を `- Status: Draft` に戻して直す)も計画を書く | `internal/org/prompts/leader.md:62-64`、同じファイルの手順 6 | 「手順 1・4・6 の機能の計画」とする。`prompts_test.go` の固定の語句も合わせる |

### Positive notes(最初の結果)

- 承認の記録は 1 回のロックの中で、読み直し、全部の件の検査、追記の順に行う。全文のハッシュは、digest を確かめたのと同じ読み込みから取るので、記録した中身と確かめた中身がずれない
- 読めない方式、不完全な結びつき、方式のない新しい結びつき(`featureApprovalMode` が `""` を `per-plan` にする)は、どれも承認を求める側に倒れる。`approval=` のない古い記録だけを split として通し、その理由が `RequiresApproval` の doc にある
- `HeaderValues` は `Parse` の規則をそのまま写し、`PLAN:` が 2 行あるときを数えられるようにした。A7 の理由(後の行で上書きされる)は doc に書いてある
- `Send` の検査は dry-run の枝の前、`Spawn` の検査は本番と dry-run と `spawnPrecheckErr` の 3 か所で同じ順に置かれ、`Spawn` の doc の手順 7 と本文のコメントが一致している
- `seatRoleFromEvents` は、`spawn_step` が role を持たないために `Roster` の role が空になる場合を拾っており、その理由が doc にある
- 新しい tech-debt の 5 行は、どれも Trigger が具体的で、列も崩れていない

### Tech debt identified(最初の結果)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L1: 承認の検査は role の名前 `implementer` と TASK の `PLAN:` のヘッダを見るだけで、座席がする作業は見ない。custom role(`--role <名前> --prompt ...`)の座席、reviewer への TASK、承認済みの `PLAN:` を付けて本文で別の作業を頼む TASK は、承認なしで実装を進められる | leader が誤って、または注入された指示で、人が見ていない作業を始められる。A2 で境界ではないとした範囲の中の話 | A2 で境界を作らないと決めており、tripwire の行と同じ扱い。行を足すだけで、コードは変えない | 既存の tripwire の行と同じ(8a 段目、または本物の境界が要るとき) | この報告の L1、`internal/org/spawn.go:1293`、`internal/org/verbs.go:498` |
| L2: 計画の見出しの前の最初の `- Approved:` の行は、両方のハッシュから外れ、検査は最初の `sha256:` の値しか読まないので、承認のあとにこの 1 行へ文を書き足せる | 計画の中で 1 行だけ、承認のあとに指示を足せる場所が残る | 直すなら `readFeaturePlanHeader` の小さな変更で済む。この cycle で直すかは orchestrator が決める | 次に `readFeaturePlanHeader` か `PlanContentHash` を変えるとき | この報告の L2 |

_(この 2 行は、依頼の指示(この報告のファイル以外は書かない)に従い、`docs/tech-debt/README.md` に足していない。直さずに残すなら、orchestrator が足す。M1 は tech-debt の (d) の行と合わせて直すか、その行に足す。)_

### Recommendation(最初の結果)

- 最初の結果の判定: yes(CRITICAL と HIGH はない。M1 は PR の前に直すことを勧めるが、止める理由にはしない)
- Follow-ups:
  - M1: 拒否の文の次の手を理由ごとに 1 つにし、leader の雛形の手順 7 を合わせる。直さないなら tech-debt の (d) の行に足す
  - L1・L2: 直さないなら上の 2 行を `docs/tech-debt/README.md` に足す
  - L3〜L11: 次にこのファイルを触るときにまとめて直せる
  - insight event: 依頼の指示でこの報告のほかには何も書いていないので、`./scripts/insights-append.sh --slug org-plan-approval --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 1 --low 11 --cycle auto --source skill || true` は打っていない
