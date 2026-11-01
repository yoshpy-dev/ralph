# sync-docs report: director-split-questions

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-director-split-questions.md`(issue #271)
- Branch: `feat/director-split-questions`。run 開始時の HEAD は `144fa87b`、base は origin/main `4fef6b8c`
- Pipeline cycle: 1(`/test` の report の記録どおり。cycle 2 にはなっていない)
- 先行 report: self-review(Run 1〜3、Merge: yes)、verify(pass。LOW の V1〜V3 と INFO の I1〜I4)、test(pass。S8 のあとの打ち直しは `run-test.sh` なし)
- 制約: Go のファイルと skill(4 つの写し × 2)は変えていない。plan は Progress だけを直し、digest は前後とも `c3e06a2ab308`

## Summary

verify が回した V1〜V3 と、INFO の I1・I3 を直した。直したのは spec の FR-9 に dated の段落 1 つ、tech-debt の 8 行(新しい行 1 つを含む)、README の 1 文、`docs/recipes/director-wake.md` とその template の写し、plan の Progress。skill の文は今のコードと合っているので変えていない。

tech-debt の Trigger は、依頼の 4 行(V2 の 2 行と V3 の 2 行)のほかに、自分で開いた行を掃いて 3 行を見つけた(266 行の (b)、362 行、368 行)。368 行は、recipe に足す証跡の引用が行の Trigger そのものに当たったので、`director-wake.md` の分は払った。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | V1。FR-9 の「2026-10-30 に決めたこと(issue #270)」の段落(書き換えていない)のすぐ下に、「2026-11-01 に決めたこと(issue #271)」の段落を足した。#270 の段落が #271 に送った「見張りが ready の follow-up を受信箱の件にして人に送ること」は #271 でもしていないこと、director が merge と進捗の報告のたびに `follow-ups:` を読んで着手を聞く形にしたこと、そのうえで分割計画の前の質問、承認と出力先の選択肢、起票と印の行と follow-up、merge のあとの着手の問い、起票した機能を文書の follow-up の集め始めで数えないこと、push の director は番号つきの選択肢で聞くこと(証跡の結果つき)を、(1)〜(7)で書いた |
| `docs/tech-debt/README.md`(status の ready の行、412 行) | V2。Trigger の「#271 (the watch sending ready follow-ups to the inbox)」を「A change that has the watch send ready follow-ups to the inbox」に直し、#271 がそれをしなかった理由を括弧で 1 つ。ほかの欄は Related に verify の V2 を足しただけ |
| 同(test gaps の行、415 行) | V2。(7) の Trigger を同じ形に直した。Related に verify の V2 |
| 同(`/director` skill の受信箱の処理の行、259 行) | V3 (a)。Trigger の末尾の「Carried over」の前に、2026-11-01 の注。`### 計画の承認` の `FORK` の問いと push の項を直したので発火したこと、待ちの段落・`WaitInbox`・`inboxWaitTimeoutPhrase`・その固定のテストには触れていないこと、(1)〜(3) は開いたままであること。`directorSkillSection` が (2) の「受信箱の節を切り出す」に使えることも 1 文。Related に plan |
| 同(merge cleanup で 800 行を超えた 4 つのテストファイルの行、386 行) | V3 (b)。1171 行(origin/main)から 1223 行になったこと(+52)、新しい固定は新しいファイルに置いたので、この行が頼む「cleanup のテストを出す」移動は move-only で Scope の外だったこと。Related に plan |
| 同(`verbs.go` などの大きいファイルの行、362 行) | 自分で見つけた。Trigger の「these files」に `director_skill_sync_test.go` が入っていて、前の branch も 1 本ごとに注を足している。1171 から 1223(+52、+300 の下)と、新しい固定が `skill_split_questions_test.go` にあること。Related に plan |
| 同(266 行) | 自分で見つけた。(b) の Trigger の最後の句「the next test added to `internal/org/director_skill_sync_test.go`」に、`TestDirectorSkillSection` が当たる。払わなかった理由(始める step 3 と `- 設定:` の印は Scope の外、(b) は開いたまま)。Related に plan |
| 同(368 行、recipe が `docs/evidence/` を引く行) | 自分で見つけた。下の「判断を残したもの」。Trigger の「The row stays open」の前に注、Related に plan |
| 同(新しい 417 行) | I1。`skill_split_questions_test.go` が 832 行(`wc -l`、commit `8bc6ae84`。verify が測った 814 行に S8 の表の 18 行)で 800 行の目安を超えたこと、同じ組の `skill_doc_followup_test.go`(891 行。base も 890 行で超えていた)と `director_skill_sync_test.go`(別の行が扱う)、払う契機(このファイルに次のテストを足すとき、または節の読み取り helper を変えるとき)と払い方 |
| `README.md`(director の wake の段落) | I3。「Leave a session a human types into on pull」の文のあとに、同じ Enter が Claude Code の pane で開いた `AskUserQuestion` に答えてしまうこと(カーソルの下の選択肢か、自由記述の欄なら 1 行が答え)と、push の wake の director は番号つきの選択肢を文で書いて聞くこと、証跡の path を 1 文で足した |
| `docs/recipes/director-wake.md` と `templates/base/docs/recipes/director-wake.md`(同じ内容で `cmp` が一致) | README より細かく書いた。「When to stay on pull」に段落 1 つ(開いた問いを閉じること、push の director の聞き方、pull だけなら使えること)、「What the agents do with the line」に実機の確かめ(T3〜T5)の結果を段落 1 つと 3 つの箇条、「What was verified」に、手で再現しただけで見張りは動かしていないことと、確かめていないこと。`docs/evidence/` の引用 3 つ(古い 2 つと新しい 1 つ)は「recorded in the ralph repository's」の形にした |
| `docs/plans/active/2026-11-01-director-split-questions.md` | Progress の「Verification artifact created」と「Test artifact created」に `[x]` と報告の path、S8 の行、この pass の行。AC の checkbox と本文は変えていない。digest は `c3e06a2ab308` のまま |
| `docs/reports/sync-docs-2026-11-01-director-split-questions.md` | この report |
| `docs/insights/events/2026-10-31-director-split-questions.jsonl` | `sync_docs` の行(cycle 1、pass、findings 0)。`insights-append.sh --cycle auto` で足した。UTC の日付はまだ 2026-10-31 で、既存のファイルに入った |

## 依頼の項目ごとの結果

| 項目 | 結果 |
|------|------|
| V1(spec の FR-9) | 足した。dated の段落は書き換えず、#270 の段落の「#271 に送った」を、新しい段落の冒頭で「#271 でもしなかった」と受けた |
| V2(412 行と 415 行の Trigger) | Trigger の句だけを振る舞いで書き直した。「既存の行の中身の書き換えは Trigger の句だけ」に従い、Debt・Impact・Why deferred は触れていない。Related に verify の V2 を足した(欄の追記で、書き換えではない) |
| V3(259 行と 386 行) | どちらも払わず、持ち越しの注を足した(理由は上の表と下の「判断を残したもの」) |
| INFO: 800 行超 | 新しい行を足した(417 行)。`director_skill_sync_test.go` は既存の 2 行(362、386)の注で扱った |
| INFO: README の push の段落 | 1 文を足した。recipe にも細かい説明を足した |
| release のノート | 下に書く |
| 古い形で書いた箇所の探索 | 下の「Surfaces checked」。直す必要があったのは README の push の段落と recipe だけで、どちらも「古い形」ではなく、証跡で分かったことの追記 |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `/director` skill の 4 つの写しと `/org` skill の 4 つの写し | 合っている | 変えていない。`check-skill-sync.sh` 14 本が lock-step。director の「人への聞き方」「分割計画の前の質問」「分割計画の承認」「機能ごとの出力先」「起票と記録」「merge を待つ issue」「予約外の文書の follow-up」と、`/org` の「merge を待つ issue」の 1 文は、spec に足した段落と合っている |
| `AGENTS.md` の地図 | 変更なし | `internal/org/` の長い行は分割計画の手順にも聞き方にも触れず、director の登録・check・wait・wake の実装を書く。動かしていない |
| `.claude/rules/ralph/` | 変更なし | `AskUserQuestion` が出るのは `subagent-policy.md` の `/spec` と `/plan` の節だけで、director の話ではない。`agent-messaging.md` の受信箱の段落は、見張りが件を送らないままなので今も正しい |
| `docs/quality/quality-gates.md` とその template | 変更なし | 「Split plan approval」の行は `ralph org start --plan` の digest の検査で、director の聞き方は書かない |
| `internal/org/prompts/director.md` と CLI の help | 変更なし | `git grep` で `AskUserQuestion`・`番号つき`・`numbered` を探し、無関係な行(doctor、merge-order の help)しか出なかった。雛形は skill に従うと書く(plan の Non-goals) |
| `docs/specs/2026-08-01-org-runtime.md`、`.codex/README.md`、`.codex/AGENTS.override.md` | 変更なし | 前者は `AskUserQuestion` の話がない。後者 2 つは Codex が番号つきの選択肢で答えると書いていて、skill の「Codex は wake によらず番号つき」と合っている |
| `docs/recipes/director-launch-cmd.md` | 変更なし | `launch_cmd` の端末に push の wake を打つ話はなく、push の説明は `director-wake.md` にある |
| spec の分割計画の記述(`:63`、`:143`) | 合っている | 「人と話して分割計画を書き、承認を取る」は新しい手順と食い違わない |
| 800 行の目安 | 新しい行を足した | 変えた `*.go` を base `4fef6b8c` と HEAD で数えた: `skill_split_questions_test.go` 0 から 832、`director_skill_sync_test.go` 1171 から 1223、`skill_doc_followup_test.go` 890 から 891、`skill_followup_test.go` 287 から 290 |

### tech-debt の Trigger の掃き(開いた行を `sweep.py` で掃いた)

変えたファイルと関数の名前(`/director` と `/org` の skill、`director_skill_sync_test.go`、`skill_doc_followup_test.go`、`skill_followup_test.go`、`skill_split_questions_test.go`、`AskUserQuestion`、`#271`、README、recipe、spec、`docs/evidence`)で、取り消し線のない Trigger を拾った。

当たった行: 259(依頼)、386(依頼)、362、266 の (b)、368、412 と 415(#271 の名指し)。

外した候補とその理由:

- 118: quality-gates を編集していない。
- 177 の (a): `/org` skill の command table が契機。変えたのは「merge を待つ issue」の節の 1 文で、table には触れていない。
- 246: Trigger の本体は `mergeorder.go` と `org_merge_order.go` で、どちらも触れていない。`skill_followup_test.go` が出てくるのは前の branch の持ち越しの注の中だけ。今回の変更は「merge を待つ issue」の 3 の固定(すべて done にする文)の更新で、merge-order の gh の `--limit` の値は書かない。
- 352: 契機は `InboxDeadlineDue`・`SweepInboxDeadlines` と `/org` skill の受信箱の節。どれも触れていない。
- 373: 契機は `/org` skill の分割計画の例と leader の雛形。触れていない。
- 376: 契機は `/director` skill の send と確かめの手順(「答える順」)で、触れていない。`### 計画の承認` の 4 の send の文は変えていない。
- 377: 契機は wait の終了コードの段落で、触れていない。
- 174、192、196、241〜249: 「stage 8a's director」を契機に持つ行。この変更は見張りも merge-order の読み方も変えていないので、契機に入らないと読んだ。
- 313、334、335、393: `wake_cmd` の `{target}`、Codex の leader と reviewer の到達、`director_cmd_test.go` と `director_wake_test.go`。どれも触れていない(recipe の追記は `{target}` と `wake_cmd` の仕様を変えない)。
- 67、138、145、227、324: `docs/evidence/` の検査や取り込み方を契機にする行で、証跡を 1 つ足したことは契機に入らない。

## 判断を残したもの

- 259、386、362、266 は払わなかった。払うには `director_skill_sync_test.go` から cleanup のテストを出す(386、362)、受信箱の節の固定を直す(259)、`- 設定:` の印の固定を足す(266)のどれかが要る。どれも plan の Scope の外で、move-only の分割は skill の文の review と混ざる。
- 266 の (b) は、前の branch が `director_skill_sync_test.go` にテストを足したときに注を足していない。Trigger の最後の句を字面どおりに読み、今回の分だけ注を足した。前の分を遡って足してはいない。
- 368 は `director-wake.md` の分だけ払った。Trigger は「recipe を編集したら、`docs/evidence/` の引用を ralph の repository の記録として書く」で、この変更は recipe に引用を 1 つ足したため。足した 1 つだけを直すと同じ recipe の中に 2 つの形が混ざるので、古い 2 つも同じ形にした(両方の写し)。ほかの 4 つの recipe は触れていない。行は開いたまま。
- recipe の末尾の「確かめたこと」は、証跡の範囲より広く書かないようにした。見張りが打つ wake も `ralph org director start` も使わず、手で再現したこと、Codex の director と `wake_cmd`、ほかの版と pane の大きさは確かめていないことを書いた。
- spec の段落に `natural-japanese` の lint を回し、warn が 1 つ出た(日本語と英数字のあいだの半角空白)。この spec の既存の段落も同じ書き方なので残した。
- verify の I4(印の行の正規表現が `after: none,feat/a` に当たる、note の 500 字を超える長い `after:` の手順がない)は、記録だけで直す必要がないとされていて、文書は触れていない。
- 計画の AC の checkbox は付けていない。verify は AC1〜AC7 を満たし、AC8 は静的の部分、test が `go test ./internal/org/` を通したので、付けられる状態にはある。付けるのは lead か /pr。

## 気づいたこと(直していない)

- `docs/insights/events/2026-10-31-director-split-questions.jsonl` に `self_review` の行がない。self-review の report にも `insights-append.sh` の記述がなく、verify と test の行だけが入っている。self-review は Run 1(MEDIUM 1、LOW 8)、Run 2(LOW 2)、Run 3(LOW 3)で、数は plan の Progress にある。足すかどうかは lead が決める。

## release のノートの文

director が分割計画を書く前に、推奨つきの選択肢で人と要件を詰め、分割計画の承認と機能ごとの出力先(issue に起票か、今すぐ着手か)を選択肢で聞くようになった。走っている org と予約が重なる機能は、issue に起票して印の行(`ralph-split: <id>/<slug> after: ...`)と follow-up に記録し、先行する PR の merge のあとに、着手してよいかを聞く。登録の wake に push がある director は、見張りの 1 行と Enter が開いた `AskUserQuestion` に答えてしまうので、番号つきの選択肢を文で書いて聞く(Codex の director は wake によらず番号つき)。新しい skill を読んだ director から効く。ralph の binary は変えていない。

## Verification

| コマンド | 結果 |
|---------|------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-01-director-split-questions.md` | `c3e06a2ab308`(編集の前後とも。`- Approved:` の digest と同じ) |
| `git diff --numstat`(commit の前) | README 1/1、plan 4/2、recipe 36/3 と template の写し 36/3、spec 1/0、tech-debt 8/7(7 行を変え 1 行を足した) |
| tech-debt の確認(`tdcheck.py`、`git show HEAD:` と比べた) | 変わったのは 259・266・362・368・386・412・415 の 7 行と、足した 417 行。どれも 5 欄(区切りで 5 片)、欄ごとの backtick と `~~` は偶数、`file:line` の参照は 0。15 の Edit の `old_string` は、事前にファイル全体で 1 回ずつ数えた。2 つ(362 行と 386 行の Related の末尾)は最初の案が 2 回と 3 回当たり、手前の語を足して 1 回にした |
| `./scripts/run-static-verify.sh` | rc 0(gofmt ok、`0 issues.`、tech-debt の plan 参照 `OK`、branch の secret scan clean)。log は `docs/evidence/verify-2026-10-31-230848.log`(gitignore の対象) |
| `./scripts/check-skill-sync.sh` | 14 本が lock-step |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-template-purity.sh` | rc 0 |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill' -count=1` | ok |
| `cmp docs/recipes/director-wake.md templates/base/docs/recipes/director-wake.md` | 同じ |
| `git diff --check`、変えたファイルの U+FFFD | どちらも 0 |

skill と Go を変えていないので、`sync-skills.sh` は打っていない。`run-test.sh` は打っていない(`go test` の Skill だけ)。

## Cycle 2

- Date: 2026-11-01
- Pipeline cycle: 2(先行 report はすべて pass)
- 変えたもの: `/org` skill と写し 3 つ、spec、tech-debt、plan の Progress、insight event。`/director` skill と Go は変えていない。plan の digest は前後とも `c3e06a2ab308`

### 直したもの

| 項目 | 内容 |
|------|------|
| C2-V1 | `/org` skill の「merge を待つ issue」の「PR がもう merge されたブランチは follow-up にしない」の項に、例外の 1 文を足した。director が follow-up のない起票の issue に人の答えを残すときは、merge 済みのブランチに `followup add` を打ち、すぐ `followup done` にする(待つための件ではなく答えの記録。手順は `/director` skill の同じ節の 3)。`skill_followup_test.go` の断片は変えていない。写しは `sync-skills.sh` を 2 回打ち、`cmp` で同じ |
| C2-V2 | tech-debt の持ち越しの注(`verbs.go` などの行)の `skill_split_questions_test.go` を、832 行から `wc -l` の 886 行にした |
| test の穴 1 | tech-debt の表の末尾に新しい行を足した。`--limit` の値が helper と断片に別々にあること、L1+L1T が残ったこと、払う契機、定数 1 つで組む直し方。ほかの行は動かしていない |
| spec | FR-9 の #271 の段落の (5) に、引き継ぎで merge 済みの issue も聞くこと、答えを add と done で残すこと、重なる org の数え直し、件数の行を足した。「`/org` skill の 1 文」は 2 か所に直した |
| 記録 | insight event に `self_review` cycle 2(pass、medium 1、low 5)と `sync_docs` cycle 2 の行。plan の Progress に cycle 2 の 3 行 |

### 直さなかったもの

- `/org` skill の command table の `followup add` の行も「PR がもう merge されたブランチは follow-up にせず」と書く。テストが固定する文で、表の編集は tech-debt の `:177` (a)(Trigger は表の次の編集)に当たる。行は「merge を待つ issue」の節を指すので、例外はそこで読める。表を次に触るときに添える
- `ralph org followup add` の help も同じ言い方(C2-I3)。Go は Non-goals
- README、recipe、rule、`AGENTS.md`、director の雛形は S9〜S11 の文と食い違わない(`git grep` で確かめた)。C2-I2 は任意で直していない
- tech-debt の Trigger を掃いた。新しい行のほかは発火しない(`:417` はテストも節の読み取りも変えず、`:177` (a) は表に触れていない)

### release のノートの文

変わる。cycle 1 の文の 2 文目(「…着手してよいかを聞く。」)のあとに、次の 1 文を足す。

director が入れ替わっても同じ issue を聞き直さないよう、待つ PR がすべて merge 済みの issue も引き継ぎで聞き、人の答えは台帳に残す。

### Verification

`check-skill-sync.sh` 14 本、`check-sync.sh` PASS、`check-template-purity.sh` rc 0、`run-static-verify.sh` rc 0(tech-debt の計画参照 OK、secret scan clean)、`go test ./internal/org/ -run Skill -count=1` ok、`plan-visual.sh digest` が `c3e06a2ab308`。tech-debt の変えた行は 5 欄。U+FFFD は 0。

## Cycle 3

- Date: 2026-11-01
- Pipeline cycle: 3(cross-review cycle 2 で上限を 3 に上げたあとの 3 回目の run。先行 report の self-review・verify・test はすべて pass)
- 測った HEAD: `b5701eca`(base は origin/main `4fef6b8c`)。この run の差分 `git diff 6f62affb..HEAD` は、`/director` skill の 4 つの写し(各 +59)、`skill_split_questions_test.go`(+165)、tech-debt の 2 行、計画と report と insight event
- 変えたもの: spec、tech-debt、plan の Progress、insight event、この report。`/director` skill、`/org` skill、Go は変えていない。plan の digest は前後とも `c3e06a2ab308`

### 直したもの

| 項目 | 内容 |
|------|------|
| C3-I1(spec) | FR-9 の #271 の段落に 3 つ足した。(4) に、頼られる機能を先に起票したら、その issue の番号を本文に `頼られる issue: #<番号>` の 1 行で行頭から書くこと(2 つ以上なら 1 つ 1 行、`- Depends on:` は slug の並びのまま)。(5) に、着手の問いの推奨は「着手する」だが、予約が今走っている org と重なるとき、または頼られる issue のどれかがまだ open のときは「後に回す」を先頭に置き、open の頼られる issue を全部説明に書くこと(番号は行全体が `^頼られる issue: #[0-9]+$` に当たる行の数字だけ、状態は `gh issue view <番号> --json state` で読む)。同じく (5) に、引き継ぎが数える follow-up は今の印の行の `after:` のブランチの件だけで、done と open が混ざる issue は `DONE_NOTE` を答えとして残りを done にすること。既存の文は書き換えず、句と文を足した(spec の差分は 1 行で、段落の中への 3 か所の挿入) |
| テストの穴 2(tech-debt) | `--limit` の行(`skill_split_questions_test.go` の「定数が 2 か所」の行)に、E7 を足した。表にタブの行がないこと、`\s` に替えても通ること、skill の文は D4 の行の確かめで 1 つだけでは変えられないこと、足すなら表に 1 行(試していない)、Related に `Cycle 3, Test gaps 2`。Debt の文では「同じ種類の 2 つ目の穴」と書いた |
| 時間切れの flake(tech-debt) | 表の末尾に新しい行。`TestRunWatcher_TimeoutIndependentOfSmallInterval` の待ち(5 秒、stub は 1 秒寝る)が、負荷 9〜10(`internal/org` 1,193.7 秒)で `claude: timed out` になったこと、単独 20/20、`-count=1` の打ち直しは通ったこと。Trigger は次に落ちたとき、または `watcher_test.go` を変えるとき(待ちを 15 秒に広げる案)。Related は test report の Cycle 3。本番の watcher の時間切れの行(stage 9 trial)とは別のものと書き分けた |
| insight event | 2026-10-31 の jsonl の末尾に 2 行。`self_review` cycle 3(pass、medium 1、low 5。self-review の `## Cycle 3` の数)と `sync_docs` cycle 3。`insights-append.sh ... --cycle 3 --events-dir <scratch>` で作った行を写した。これで C3-I4(self-review の行がない)は解消。verify と test の cycle 3 の行は、それぞれの run がすでに入れていた。`self_review` の `ts` は今の時刻で、実際の run より後(cycle 2 と同じ入り方) |
| plan | Progress に verify cycle 3、test cycle 3、この sync-docs の 3 行。本文は変えていない |

### 食い違いを探したもの

- S12・S13 の語(`頼られる`、`DONE_NOTE`、`AFTER_BRANCH`、`followup done`、`follow-up が 1 件もな`)を、reports・plans・evidence・skill の写しの外で `git grep` した。当たったのは `/director` skill、`/org` skill の `followup list` と `followup done` の行(`AFTER_BRANCH`・`DONE_NOTE` の列名と制御文字の説明はコードと合う)、README の `followup` の説明(`followup list --all` が done の時刻と note も出すこと、`followup done <id>` は人に聞いたあとに打つこと。S12・S13 の混ざった件の片付けと合う)、spec の FR-9、tech-debt だけ。`docs/recipes/director-wake.md`、`.claude/rules/ralph/`、`AGENTS.md` に当たりはなく、S12・S13 の文と食い違わない
- `/org` skill の `- Depends on:` の定義は slug の並びのままで、S13 の「番号は別の行に書く」と合う
- 気づいたが直していない: `/org` skill の「merge を待つ issue」の例外の 1 文は「follow-up のない起票の issue」と書く。S13 の 3 は「今の `after:` のブランチの follow-up が 1 件もないとき」で、書き換えの前の待ちで done の件が残る issue も当たる。この 1 文は手順を `/director` skill の 3 に委ねていて、`skill_followup_test.go` は固定していない。直すなら「今の `after:` のブランチの follow-up がない」に替える 1 句で、`/org` skill の 4 つの写しを作り直すことになる(skill の文の変更で、パイプラインの 3 回目の run のあとに入る)。lead が決める

### tech-debt の Trigger の掃き方

open の行の Trigger(4 列目)を python で掃いた(`~~` だけの Trigger は除いた)。この run が触った面(`/director` skill の「起票と記録」と「merge を待つ issue」、`skill_split_questions_test.go`、`followup done`、`- Depends on:`)と、自分で直す面(spec、tech-debt、plan、insight event)の語で当てた。

- 発火しない: 受信箱の処理の行(S12・S13 の変更は「分割計画と機能ごとの org」と「merge を待つ issue」で、`## 受信箱の処理` の外)、`director_skill_sync_test.go` を名指す 3 つの行(このテストは cycle 3 で変わっていない)、`skill_split_questions_test.go` の行(`func Test` の数は 14 本のまま、節を読む関数も変わっていない。行数は 886 から 995 だが、行は数を書かず「over 800 lines」と書く)、`--limit` の行(`--limit` は変わっておらず、S12・S13 はテスト関数を足していない。E7 の句はこの sync-docs が足したもので、Trigger の発火ではない)、recipe の行(recipe は変わっていない)
- 見たが当たらない: `watcherInvokeTimeout` の変更を Trigger に持つ stage 9 trial の行。flake の行が指す変更は、テストが自分で上書きする 5 秒で、本番の既定(60 秒)ではない
- cycle 1 の持ち越しの注(`verbs.go` の行と 800 行の行など): cycle 3 は注の対象の `director_skill_sync_test.go` を変えていないので、新しい文は足していない

### release のノートの文

変わる。通しの最終の文は次のとおり(cycle 1 の文に cycle 2・3 の文を足した)。

director が分割計画を書く前に、推奨つきの選択肢で人と要件を詰め、分割計画の承認と機能ごとの出力先(issue に起票か、今すぐ着手か)を選択肢で聞くようになった。走っている org と予約が重なる機能は、issue に起票して印の行(`ralph-split: <id>/<slug> after: ...`)と follow-up に記録し、先行する PR の merge のあとに、着手してよいかを聞く。起票した機能に頼る機能の issue には、本文に `頼られる issue: #<番号>` の行を書き、頼られる issue が open のあいだは着手の問いで「後に回す」を推奨する。director が入れ替わっても同じ issue を聞き直さないよう、待つ PR がすべて merge 済みの issue も引き継ぎで聞き、人の答えは台帳に残す。引き継ぎが数える follow-up は今の印の行の `after:` のブランチの件だけで、done と open が混ざった issue は、done の件の note を答えとして残りを done にする。登録の wake に push がある director は、見張りの 1 行と Enter が開いた `AskUserQuestion` に答えてしまうので、番号つきの選択肢を文で書いて聞く(Codex の director は wake によらず番号つき)。新しい skill を読んだ director から効く。ralph の binary は変えていない。

### Verification

| コマンド | 結果 |
|---------|------|
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308`(編集の前後とも。`- Approved:` の digest と同じ) |
| `git diff --numstat`(commit の前) | insight event 2/0、plan 3/0、spec 1/1、tech-debt 2/1(1 行を変え 1 行を足した) |
| tech-debt の確認(`tdcheck.py`、`git show HEAD:` と比べた) | 変わったのは 418 行と、足した 419 行だけ。どちらも 5 欄(区切りで 5 片)、欄ごとの backtick と `~~` は偶数、`file:line` の参照は 0。5 つの Edit の `old_string` は、事前にファイル全体で 1 回ずつ数えた |
| insight event | 14 行すべて JSON として読める |
| `./scripts/check-skill-sync.sh` | 14 本が lock-step |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `./scripts/check-template-purity.sh` | rc 0 |
| `./scripts/run-static-verify.sh` | rc 0(gofmt ok、`0 issues.`、tech-debt の plan 参照 OK、branch の secret scan は `4fef6b8c..b5701eca` で clean) |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill' -count=1` | ok(0.921 秒) |
| `git diff --check`、差分の U+FFFD | どちらも 0 |

skill と Go を変えていないので、`sync-skills.sh` は打っていない。`run-test.sh` も打っていない。

## Cycle 4

- Date: 2026-11-01
- Pipeline cycle: 4(`cycle-count.json` は `cycle` 3、`cap` 3 で、上限の最後の run。先行 report の self-review・verify・test はすべて pass)
- 測った HEAD: `960c88c3`(base は origin/main `4fef6b8c`)。この run の差分 `git diff 3edfa501..960c88c3` (S14 09c9f34d と S15 ac5a0578 ほか)は、`/director` skill の 4 つの写し(各 +31/−11)、`skill_split_questions_test.go`(+60/−18)、計画と report と insight event
- 変えたもの: `/org` skill と写し 3 つ(各 +1 行)、spec、plan の Progress、insight event、この report。`/director` skill と Go は変えていない。plan の digest は前後とも `c3e06a2ab308`

### 直したもの

| 項目 | 内容 |
|------|------|
| C4-I1(spec) | FR-9 の #271 の段落に、S14・S15 の振る舞いを句で足した。(4) の先頭に、起票の前に `ralph status` を打ち直して `reserved:` を機能の `- Reserve:` と比べ、重なる org を全部数え直すこと(start が断った最初の 1 つで決めない)。(5) の `ready` の括弧の中に、open の follow-up の STATE が `waiting` でも `ralph org merge-order --all` で `merged` のブランチは、記録の直前に merge 済みとして待ち終えたと数えること、add と done のあいだで止まった件はこれで拾い、もう一度聞いてから done にすること。既存の文は書き換えていない(spec の差分は 1 行で、段落の中への 2 か所の挿入) |
| C4-I2(`/org` skill) | 「merge を待つ issue」の、PR が merge 済みのブランチを follow-up にしない項の例外(`followup add` のすぐ後に `followup done`)の最後に、1 文を足した。「add と done のあいだで止まった件は、director が同じ節の 2 で拾う。」`skill_followup_test.go` が固定するのは例外の前の文(`PR がもう merge されたブランチは follow-up にしない`)と command table の行で、足した文はその外。写しは `sync-skills.sh` を root と `templates/base/` の 2 回打ち、4 つとも `cmp` で同じ |
| C4-I3(insight event) | 2026-10-31 の jsonl の末尾に 2 行。`self_review` cycle 4(pass、low 5、self-review の `## Cycle 4` の数)と `sync_docs` cycle 4。`insights-append.sh ... --cycle 4 --events-dir <scratch>` で作った行を写した。verify と test の cycle 4 の行は、それぞれの run が入れていた。2 行の `ts` は同じ(`04:39:58Z`)で、`self_review` の `ts` は test の行より後(cycle 2・3 と同じ入り方) |
| plan | Progress に self-review・verify・test の 4 回目の run とこの sync-docs の 4 行。本文は変えていない |

### 食い違いを探したもの

- S14・S15 だけの語(`数え直`、`記録の直前`)を、`docs/reports`・`docs/plans`・`docs/evidence` の外で `git grep` した。当たったのは `/director` skill の 4 つの写し、spec の FR-9、`skill_split_questions_test.go` だけで、README、`docs/recipes/`、`.claude/rules/ralph/`、`AGENTS.md`、`/org` skill に当たりはない(`/org` skill は、この run で足した 1 文が `followup` の手順を `/director` skill に委ねる形で、S14・S15 の語を持たない)。`ralph org merge-order --all` の語は別の機能の文とテストにも広く出るので、S14・S15 の語としては掃いていない。README の `followup` の説明(`followup done` は人に聞いたあとに打つ)は S15 の「答えのあと done」と合う(verify の cycle 4 も確かめた)
- `/org` skill の command table の `followup add` の行(「PR がもう merge されたブランチは follow-up にせず」)は、C2-V1 のときと同じく変えていない。例外は「merge を待つ issue」の節で読める
- 気づいたが直していない: spec の (5) は、skill の言い方に合わせて「記録の直前に merge 済み」と書いた。self-review の C4-2 は、この言い方が止まった件(merge は何日前でもよい)に合わないと指す。skill の言い方を変えるときは、spec の (5) も同じ語に直す。C4-1(名前の使い回しで古い merge を待ち終えたと数える道)も、skill と同様に spec の (5) の規則に含まれる。skill の側を Known gaps に書くので、spec の文は変えていない
- 気づいたが直していない: spec の (3) は、start が断ったあとの聞き直しに数え直しを書いていない。(4) に足した句(start が断った最初の 1 つで決めない)が同じ数え直しを指すので、食い違いにはしていない

### tech-debt の Trigger の掃き方

open の行の Trigger(4 列目)を python で掃いた(`~~` で始まる Trigger は除いた)。この run が触った面(`/org` skill の「merge を待つ issue」の項、spec の FR-9、plan、insight event)と、S14・S15 が触った面(`/director` skill の「起票と記録」「merge を待つ issue」、`skill_split_questions_test.go`)の語で当てた。`skill`・`表`・`table`・`答えの記録`・`merge を待つ`・`FR-9`・`jsonl`・`events/` を含む Trigger を全部読んだ。

- 発火しない: `/org` skill を名指す行のうち、`:157`(org の quality pipeline の変更)、`:352`(inbox の resolve deadline と `/org` skill の inbox の節)、`:373`(`/org` skill の split-plan の例)、`:177` (a)(command table の次の編集)は、この run が触れた項(「merge を待つ issue」の例外の文)ではない。`:417` と `:418`(`skill_split_questions_test.go` へのテストの追加、節を読む関数の変更、`--limit`)は、S14・S15 でも `func Test` が 14 本のままで読む関数も変わらず、この run はテストを触っていない。`:376`・`:377`(`/director` skill の send の確かめ、wait の終了コード)は、S14・S15 が触った節の外。`:412`・`:415`(follow-up の `ready` の決め方、`FollowupStore.Done`)は、コードを変えていない
- cycle 1 の持ち越しの注(`verbs.go` の行と 800 行の行など): この run は注の対象の `director_skill_sync_test.go` を変えていないので、新しい文は足していない
- 注を足した行: なし

### release のノートの文

変わる。通しの最終の文は次のとおり(cycle 3 の文に、S14・S15 の分を足した)。

director が分割計画を書く前に、推奨つきの選択肢で人と要件を詰め、分割計画の承認と機能ごとの出力先(issue に起票か、今すぐ着手か)を選択肢で聞くようになった。走っている org と予約が重なる機能は、起票の前に `ralph status` を打ち直して重なる org を全部数え直したうえで、issue に起票して印の行(`ralph-split: <id>/<slug> after: ...`)と follow-up に記録し、先行する PR の merge のあとに、着手してよいかを聞く。起票した機能に頼る機能の issue には、本文に `頼られる issue: #<番号>` の行を書き、頼られる issue が open のあいだは着手の問いで「後に回す」を推奨する。director が入れ替わっても同じ issue を聞き直さないよう、待つ PR がすべて merge 済みの issue も引き継ぎで聞き、人の答えは台帳に残す。引き継ぎが数える follow-up は今の印の行の `after:` のブランチの件だけで、done と open が混ざった issue は、done の件の note を答えとして残りを done にする。open の follow-up の STATE が `waiting` でも merge-order で `merged` のブランチは待ち終えたものに数え、答えの記録が add と done のあいだで止まった issue も聞き直す。登録の wake に push がある director は、見張りの 1 行と Enter が開いた `AskUserQuestion` に答えてしまうので、番号つきの選択肢を文で書いて聞く(Codex の director は wake によらず番号つき)。新しい skill を読んだ director から効く。ralph の binary は変えていない。

### Verification

| コマンド | 結果 |
|---------|------|
| `./scripts/plan-visual.sh digest <plan>` | `c3e06a2ab308`(編集の前後とも。`- Approved:` の digest と同じ) |
| `git diff --numstat`(commit の前、この report を除く) | `/org` skill の 4 つの写し 各 1/0、spec 1/1、plan 4/0、insight event 2/0 |
| `sync-skills.sh`(root)と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills sync-skills.sh` | 14 本と 13 本を mirror。`/org` skill の 4 つの写しは `cmp` で同じ |
| insight event | 19 行すべてが JSON として読める。`self_review` と `sync_docs` の cycle 4 の行は `insights-append.sh` が作ったものと同じ |
| `./scripts/check-skill-sync.sh` | 14 本が lock-step |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0) |
| `./scripts/check-template-purity.sh` | rc 0 |
| `./scripts/run-static-verify.sh` | rc 0(gofmt ok、`0 issues.`、tech-debt の plan 参照 OK、branch の secret scan は `4fef6b8c..960c88c3` で clean)。log は scratch に置き、コミットしない |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill' -count=1` | ok(0.759 秒) |
| `git diff --check`、差分の U+FFFD | どちらも 0 |

`run-test.sh` は打っていない(`go test` の Skill だけ)。Go は変えていない。
