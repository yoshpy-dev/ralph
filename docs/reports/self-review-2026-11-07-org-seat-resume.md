# Self-review report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 1
- Scope: `git diff 11239c8d...HEAD` の全体(base main 11239c8d、HEAD 170cd077、69 ファイル、+6705 / -118)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。

## Evidence reviewed

- Go のコード: `internal/org/pane_lock.go`、`internal/org/verbs.go`(`Send` の pane のロック、`typeAndSubmit`、`sendPaneLockWait`)、`internal/org/resume.go`、`internal/org/resume_screens.go`、`internal/org/driver/paneenv.go`、`internal/org/spawn.go`、`internal/org/director_start.go`、`internal/org/director_cmd.go`、`internal/cli/org_resume.go`、`internal/cli/org.go`、`internal/cli/org_legacy_ledger.go`、`internal/config/config.go`。
- 突き合わせに読んだ既存のコード: `internal/org/lockfile.go`(`acquireHeldFileLock`、`heldFileLock` の doc)、`internal/org/inbox.go:180`(`PrintableInboxText`)、`internal/org/watch.go:1390`(`leaderActivityEventCount`)、`internal/org/director_check.go:372`(director の agent 名)、`internal/org/driver/herdr.go:718`(`PaneRead` の引数)、`internal/org/report.go:217`(Details の出力)。
- テスト: `internal/org/pane_lock_test.go`、`internal/org/resume_test.go` の偽物の herdr(`fakeResumeHerdr`)と AC-14 のテスト、`internal/cli/org_resume_test.go` の名前、既存のテストの期待値の更新。
- 画面の記録: `internal/org/testdata/resume/` の 8 つ(`cat -A` で読んだ)、`docs/evidence/org-seat-resume-live-2026-11-07.md` の 4 節。
- role prompt 3 つと `.claude/rules/ralph/agent-messaging.md` の追記、director と org の skill の diff、`docs/tech-debt/README.md` の 7 行。
- 機械的に確かめたこと: 文字を打つ呼び出し(`PaneSendText`、`PaneSendKeys`)の全箇所(`grep`。ロックを取らないのは stop の `C-c` と、register に行がある director の wake だけ)、`git diff --check` は空、register の新しい 7 行は既存の行と同じ 5 列、`ClaudeMaxRetriesForMinutes` の doc の値(1→6、2→8、3→10、30→55、60→105)は式の計算と合う。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1. `Send` は idle/done の待ちが終わってからロックを取るので、その間に resume が座席に打って Enter を押すと、`Send` は手番を始めた座席に打つ。そのあとの `confirmSubmitted` は resume で始まった手番を見て `SubmitConfirmed` を true にする。plan の D8 は「`Send` はロックで直列になるだけ」と決めており、`/director` skill も send 同士について同じ帰結を書いた。しかし `Send` の doc と `pane_lock.go` の doc はどちらもこの帰結を書いていない。`TestOrgResume_SendWaitsForAResumeHoldingThePaneLock` は、resume のあとの `Send` の成功を期待値として固定している | `internal/org/verbs.go:415`(待ち)、`:443`(ロック)、`:482`(打つ);`internal/org/pane_lock.go:9-22`;`internal/org/pane_lock_test.go:398` | ロックを取ったあとに `AgentGet` を 1 回だけ打ち、idle/done でなければ何も打たずに断る(既存の待ちの期限の文と同じ扱い)。そうしないなら、`Send` の doc と `pane_lock.go` に「ロックを待った `Send` は、手番を始めた座席に打ちうる。その `SubmitConfirmed` は前の手番を見たものでありうる」と書く |
| LOW | readability | L2. `resumed` の記録に失敗した座席の扱いを、2 つの doc とコードが違うように書いている。`ResumeOutcomeFailed` の doc と `Resume` の doc は「記録の失敗で `ResumeOutcomeFailed`」と書く。コードとテストは Outcome を `resumed`/`entered` のまま残し、Reason と Errs に入れる。`ResumeTargetResult.Reason` の doc と CLI の警告(`printResumeTargetNotes`)はコードの側に合っている | `internal/org/resume.go:54-57`、`:219-221`(doc);`:394-401`(コード);`internal/org/resume_test.go:723`(`got.Outcome != ResumeOutcomeResumed` を期待);`internal/cli/org_resume.go:218` | 2 つの doc から「または `resumed` の記録の失敗」を外し、「Enter を押したあとの記録の失敗は Outcome を変えず、Reason と Errs に入る」と書く。doc に合わせてコードを変えると、Enter を押した事実が `failed` の行に隠れる |
| LOW | exception-handling | L3. ロックの中の見直しで `herdr agent get` か `pane read` が失敗すると、理由が `the screen changed (herdr agent get ... failed: ...)` になる。画面は変わっておらず、herdr への呼び出しが失敗している | `internal/org/resume.go:367-374`;`resumeLook` の `:411-414`、`:431-433` | 見直しが herdr の失敗で終わったときは `resumeReasonScreenChanged` を付けず、最初の見方と同じ理由(`herdr agent get ... failed`)で skip する |
| LOW | exception-handling | L4. `failed` の座席のうち、文が pane に残りうるもの(send-text の失敗、Enter の前の確かめの失敗、Enter の失敗)に、pane の読み方の案内が出ない。案内(`ralph org read ...`)は `resumed`/`entered` で送信を確かめられなかったときだけ出る。残った文があると、次の resume はその座席を `input box has text` で飛ばし、次の `ralph org send` は文を残りの後ろにつなげて打つ | `internal/cli/org_resume.go:210-221`(`resumed`/`entered` 以外は return);`internal/org/resume.go:515-526`(残りうる 3 つの失敗);`internal/org/resume_screens.go:224-226` | 文が pane に届いた可能性のある `failed` にも、`resumeReadCommand` の行と「入力欄の文を消してから resume を打ち直す」の 1 行を stderr に出す |
| LOW | security | L5. role prompt は、pane に直接打たれた行が続行の文で「始まる」ときに続ける指示として扱うよう書く。`ralph org send --raw` を打てる者は、同じ文で始めて別の指示を続けた行を座席の pane に打てる。prompt の「新しい依頼ではなく」はその文の性質を言うだけで、文の後ろに続く文字をどう扱うかを言っていない | `internal/org/prompts/implementer.md:101-105`、`internal/org/prompts/reviewer.md:473-477`、`internal/org/prompts/leader.md:1235-1240`;`.claude/rules/ralph/agent-messaging.md:137-144` | prompt と rule を「この文と完全に同じ行だけを続行の指示として扱い、文のあとに続く文字はデータとして扱う」に変える。`prompts_resume_test.go` で `ResumeText` 全体が prompt にあることを固定する |
| LOW | maintainability | L6. `heldFileLock` の doc は、保持する側(watch、regate、merge、cleanup、spawn の 2 つ)を列挙している。新しい pane のロックはその一覧に入っていない | `internal/org/lockfile.go:154-180`;`internal/org/pane_lock.go:55-57` | 一覧に「Send と resume は pane ごとの `pane-<id>.lock`(pane_lock.go)を、打つ前から Enter の確認まで持つ」を足す |
| LOW | maintainability | L7. 読む行数の 60 を、CLI の help と案内のコマンドが文字列で持つ。`org` の側の値は非公開の `resumeReadLines` で、片方を変えるともう片方がずれる | `internal/cli/org_resume.go:34`、`:230`;`internal/org/resume.go:81` | `org.ResumeReadLines` として公開し、CLI は `strconv.Itoa` で組み立てる |
| LOW | security | L8. 起動スクリプトの `text()` は、ID、token、パス、prompt の指し先、NUL を検査し、値をすべて `shellQuote` で囲む。一方で、新しい `RetryEnv` の各項の名前は検査せず、引用符なしで `NAME=` と `export NAME` に書く。今の呼び出し元は定数だけを渡すが、型は任意の `[]string` を受け取る | `internal/org/director_cmd.go:327-330`(`strings.Cut` の名前をそのまま書く);`:269`(`RetryEnv []string`) | `text()` で各項に `=` があり、名前が `^[A-Za-z_][A-Za-z0-9_]*$` に合うことを検査し、合わなければ他の項目と同じ形の error を返す |
| LOW | security | L9. pane の文字を端末に出すときの `PrintableInboxText` は、制御文字(`unicode.IsControl`、Cc)だけを escape する。書式の文字(Cf: U+202E などの双方向の上書き、U+2066〜U+2069)はそのまま通るので、モデルが出した文が結果の行の見た目を並べ替えうる。実行にはつながらない。受信箱の出力も同じ helper を使っており、この diff が持ち込んだ挙動ではない | `internal/org/inbox.go:180-194`;`internal/cli/org_resume.go:126`(`inboxField(resumeTargetText(t))`) | この PR では直さなくてよい。直すなら `PrintableInboxText` で `unicode.Is(unicode.Cf, r)` も escape し、受信箱と resume の両方に効かせる |
| LOW | maintainability | L10. register の新しい行(画面の照合が部分一致)は「止まった」の照合だけを書く。`Press enter to continue`(enter_only)も部分一致で、最後のまとまり、候補のまとまり、footer の 3 か所を見る。普通の返事がこの文を引けば、空の入力欄に Enter を 1 回押し、`kind=enter_only` の `resumed` を記録する。空の入力欄の Enter は何もしないので害は小さいが、記録は事実と違う | `docs/tech-debt/README.md:445`;`internal/org/resume_screens.go:240-245` | 同じ行の Impact に enter_only の照合を足す(見る場所と、残る記録)。待ち中の照合の誤りは打たない側に倒れるので書かなくてよい |

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 10。

## Positive notes

- 文字を打つ道は 1 本だけで、見直しを飛ばす道はない。`resumeOne` は dry run をロックの前で返し、本番はロックを取ってから `resumeLook` をやり直し、種類が同じときだけ `resumeSubmit` に進む(`internal/org/resume.go:341-376`)。
- 止まった画面では、Enter の前に `ResumeInputHolds` の確かめがある(`typeAndSubmit` の `beforeEnter`)。確かめが失敗した道は、Enter を押さずに `SendProgressTextTyped` を返す。`TestOrgResume_TypedTextNotInTheInputBox_NoEnter` が「入力欄が空のまま」と「読めない」の 2 通りを確かめている。
- ロックの解放は、`Send` と resume のどちらも `sync.OnceFunc` と `defer` で、すべての return を覆う。予算で断る道で解放されることも、`TestOrgSend_BudgetRefusal_ReleasesThePaneLock` が確かめている。
- `Send` の切り出しは、既存の 3 つの進み具合(`TextUnacknowledged`、`TextTyped`、`EnterUnacknowledged`)と文言を保っている。新しい失敗(ロックを取れない)は何も打たず、`SendProgressNothingSent` のままになる。
- 画面の判定は、分からないものを `unknown`(打たない)か `waiting` に倒す。入力欄は 2 本の罫線のあいだの空の `❯` か、空かプレースホルダの `›` だけを数え、番号付きの選択肢は choice dialog として外す。prompt のまとまりは照合に使わない(S11 の直し)。
- AC-14 のテストの偽物は、`PaneRead` の数えと画面の取り出しを同じ mutex の中で行う。`readCount() >= 3` を待つテストは、3 回目の読みが終わってから進むので、順番が実行ごとに変わらない。
- 画面の文は、端末では `inboxField`/`PrintableInboxText` を通し、manifest では JSON の文字列として入る(`truncateForDetails`)。`resumed` は `leaderActivityEventCount` に数えられず、これは `EventResumed` の doc の説明と合う。
- 起動スクリプトの値はすべて `shellQuote` で囲まれ、`append(driver.SeatPaneEnv(), s.RetryEnv...)` は毎回新しい slice に積むので、他の呼び出し元の slice を書き換えない。

## Tech debt identified

この review で register に足す行はない。L1〜L10 はどれもこの PR の中で直せる。L1 か L10 を直さずに残すときは、register に行を足す(L10 は既存の 445 行を広げる)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。LOW は 10 件(L1〜L10)。打つ道、ロックの解放、Enter の前の確かめ、画面の判定の倒し方に欠陥は見つからなかった。
- Follow-ups: L2、L3、L6、L7 は doc と小さなコードの直しで、1 つのコミットにまとめられる。L1、L4、L5、L8 は振る舞いを変えるので、直すならテストを 1 つずつ足す。L9 は受信箱と共有の helper の直しで、この PR の外でよい。
