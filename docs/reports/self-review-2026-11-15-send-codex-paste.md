# Self-review report: send-codex-paste

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-15-send-codex-paste.md`(issue #326)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目
- Scope: `git diff origin/main...HEAD`(HEAD `bafce96c`、base は origin/main の `3c75e2e0`。取り込みのマージ `1f3146f5` は変更に入れていない)。25 ファイル、1095 行追加、88 行削除。Go の本体は 3 ファイル(`send_paste.go`、`send_screen.go`、`verbs.go` の doc comment)で、残りはテスト(新規 1、既存 6)、fixture 5、文書(規則 2、skill 4、`AGENTS.md`)、証拠 2、計画。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いの検査は `/verify` と `/test` と `/sync-docs` の側

## Evidence reviewed

読んだもの。

- `internal/org/send_paste.go` の差分の全部、`internal/org/send_screen.go` の差分の全部と `sendInputHolds`、`sendCodexPasteHolds`、`sendCodexTypedInputText`、`sendInputHasPastePlaceholder` の本体、`internal/org/verbs.go` の `Send` の doc comment の差分
- `internal/org/send_codex_paste_test.go`(389 行、全部)と、既存 6 テストファイルの差分(`send_paste_test.go`、`send_screen_test.go`、`send_fallback_test.go`、`send_fallback_marker_test.go`、`resume_screens_test.go`、`usage_limit_screen_test.go`)
- fixture のうち `codex-input-pasted-placeholder.txt` と `codex-submitted-pasted.txt` の中身
- `.claude/rules/ralph/agent-messaging.md` の差分、`/org` skill の `send` の行の追加部分、`AGENTS.md` の追加部分。3 組(規則、`.claude` の skill、`.agents` の skill)と `templates/base/` の写しは `cmp` で同じ
- `docs/evidence/org-send-codex-paste-probe-2026-11-15.md` と `docs/evidence/org-send-codex-paste-live-2026-11-15.md` の全部。`docs/evidence/org-send-paste-probe-2026-11-13.md` の X1 の行(5340 バイト、codex-cli 0.166.0)
- `docs/tech-debt/README.md` の #319 の行(503 行目)と、`send_screen.go` に触れる行
- 絞ったテスト `go test ./internal/org/ -run 'Paste|SendInputHolds' -count=1` は `ok`(3.273s)。ほかのテストと静的検査は走らせていない

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 打つ文と記録する文の分け方: `typeAndSubmit` に渡すのは `sendTextForDriver(seat.Driver, p.Text)` の 1 か所で、`sent` の Details、dry-run の Details、`sendScreenSubmitChecks(screenPath, seat, p.Text)` は元の文のまま。diff はこの分け方を変えていない。`sendInputHolds` の `typed` が印なしの文であることは、規則 4 の字数の比べ方(`utf8.RuneCountInString(typed)`)の前提で、テスト(`p4 typed with the markers counted` の行)が、印つきの文を渡すと false になることを固定している
- `sendCodexPasteHolds` の境界: `FindStringSubmatchIndex` が 1 つ目の一致だけを返しても、`loc[0] != 0 || loc[1] != len(got)` で先頭から末尾まで 1 つだけを要求するので、2 つの置き換えと前後の余計な文字は通らない。`loc[2]:loc[3]` は正規表現が 1 つの群を持ち、一致したあとなので範囲を外れない。`strconv.Atoi` の失敗(桁あふれ)は `err == nil` で false。先頭の 0 つきの数(`02016`)は通るが、codex が出す形ではなく、害もない
- 失敗の向き: 規則 4 を通らない形(別の数、2 つ、前後に文字、別の形の置き換え)は、今までどおり Enter を押さずに `text typed but not submitted` で止まる。Enter のあとの確かめは `sendInputHasPastePlaceholder` が数を問わずに置き換えを「まだ入力欄」とみなすので、送られなかった貼り付けを「送られた」と読む道はない
- `sendTextForDriver` の driver の比べ方は完全一致(`"claude"`、`"codex"`)。大文字の `Codex`、空、`gemini` は素の文(テストに行がある)。`ResumeText` は `typeAndSubmit` を直接呼ぶので素の文のまま(diff の外)
- 印の断り(`checkNoPasteMarkers`)は diff の外で、driver によらない。codex に貼り付けで打つことで、新しい断りの道は増えない
- 呼び手: `sendInputHasPastePlaceholder` の呼び手は `SendScreenSubmitted` の 1 か所だけ(`grep`)。`internal/` の非テストのコードに、claude だけが貼り付けで打たれると書く文言は `send_fallback.go:291` の 1 つを除いて残っていない(L2)
- 証拠と fixture: `/Users/` や `hiroki` の実パスはない(`grep`)。fixture の `Hook failed` の行は、証拠の環境(信頼しない選択)の写しで、そのまま
- テストの前提の固定: `TestSendCodexPasteProbeMessages_AsTyped` が、4 つの文のバイト数、字数、UTF-16 の単位の数、改行の数と SHA-256 を証拠の表と突き合わせる。fixture の元の文が実機で打った文と同じであることを、コードで確かめている

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1. コードのコメント 4 か所が、確かめた範囲より広いことを書いている。(a) 「1698 バイト以上は 1 つの置き換えになり、それより短い文は文として出る」と 3 か所(`send_screen.go` の冒頭の注、正規表現の注、`sendInputHasPastePlaceholder` の doc)にあるが、観測は 651 バイト(文が出た)と 1698、4925 バイト(置き換え)の 3 点で、証拠自身が「畳まれ始める大きさ(651 バイトと 1698 バイトのあいだ)」を確かめていないことに挙げている。651 と 1698 のあいだの文はどちらにもなりうる。また p3 は 1698 バイトで 1698 字、p4 は 4925 バイトで 2016 字なので、境が字数かバイト数かも観測からは分からない。(b) `send_paste.go` の「49 から 4925 バイトの文はどれも送られた」は、5 つの大きさを試した結果を範囲として書いている。(c) 正規表現の注の「字数(Unicode の符号位置)」は、絵文字 1 つ(1 符号位置で UTF-16 の 2 単位)を含む 1 回の観測で、UTF-16 の単位との区別だけがついている。符号位置と書記素(結合文字、ZWJ の絵文字)の区別は、証拠も計画の Open questions も確かめていないとしている。規則 4 は数が合わなければ通さない向きなので動きは安全側だが、読み手は閾値と数え方が分かっていると読む | `send_screen.go:70-71`、`:185`、`:188`、`:553`。`send_paste.go:31-32`。`docs/evidence/org-send-codex-paste-probe-2026-11-15.md:15-22`(651 バイト、1698 バイト、字数と書記素は確かめていない)、`:73-76`(確かめていないこと)。同じ型の前例は #319 の「約 800 バイト」(`self-review-2026-11-13-send-claude-paste.md` の L1) | コメントを観測の言い方に直す。例: 「651 バイトの文は入力欄に文として出て、1698 バイトと 4925 バイトの文は 1 つの置き換えになった(その間の大きさは確かめていない)」「数は、2016 字・UTF-16 の 2017 単位の文で 2016 だった(書記素との区別は確かめていない)」。文書側(`agent-messaging.md`)は閾値の数字を書いておらず、そのままでよい |
| LOW | maintainability | L2. `sendScreenSubmitChecks` の doc は、Enter の前の確かめが通る形を「文の頭、背の高い文の終わり、claude の貼り付けの置き換え」と並べ、codex の置き換えを書いていない。同じ列挙を持つ `verbs.go` の `Send` の doc は更新されているので、3 つの写しのうち 1 つが前の版のまま残った | `internal/org/send_fallback.go:289-292`(`or claude's paste placeholders for it`)。更新済みは `verbs.go:387-390` と `send_screen.go:263-293` | `send_fallback.go` の列挙に「or codex's paste placeholder for it」を足す。コメントだけの変更 |
| LOW | maintainability | L3. 計画が Affected areas に挙げた `docs/tech-debt/README.md` が diff にない。#319 の Known gaps の行の (e) 「印の断りは codex も含めて全 driver にかかるが、貼り付けを受けるのは claude の座席だけ」は、この diff で codex も貼り付けを受けるようになり、前提が成り立たない。(e) の Trigger 「印を受け取る必要のある codex の座席」も、断りが貼り付けを守る向きに変わったので意味が変わる。また計画の Risks と証拠の「確かめていないこと」にある次の残りが、register のどこにもない: 貼り付けが途中で切れたときの codex の入力欄、2000 字の上限を超える `--raw` の文と 10 KB 超、前の置き換えが残る入力欄への貼り付け、畳まれ始める大きさ(L1)、字数と書記素の数、codex の別の版、画面の道を ralph で通しで走らせていないこと(live の証拠の「確かめていないこと」)。8 ビットの終わりの印(#319 の (f))も codex に広がる | `docs/tech-debt/README.md:503`((e)、(f))。`docs/plans/active/2026-11-15-send-codex-paste.md` の Affected areas(「この PR の Known gaps、#319 の行の (e)」)、Risks、Open questions。`git diff origin/main...HEAD --stat` に `docs/tech-debt/` はない | `/sync-docs` か `/pr` の前に、(e) を取り消し線で閉じ(印の断りが codex の貼り付けを守る形になった)、#326 の Known gaps の行を 1 行足す。(f) の「Claude Code」を codex にも広げるか、新しい行に写す。文書の食い違いの検査は `/sync-docs` の側だが、falsified な行を残したまま PR に進むと、register を読む人が誤る |
| LOW | readability | L4. `Send` の doc の折り返しが 1 行だけ短い。追加した文のあとに前からあった行が続くので、`// the runes of Text); when` で行が切れ、次の行が `// it does not, or the pane cannot be read, ...` と長い | `internal/org/verbs.go:388-392` | 段落を詰め直す。コメントだけの変更 |
| LOW | naming | L5. テストの名前が 2 つ、周りと合わない。(a) `sendCodexPlaceholderAsciiLine` は Go の頭字語の綴り(`ASCII`)でない(この package の他の識別子に `Ascii` はない)。(b) fixture の名前が非対称: 日本語の p4 の画面が無印の `codex-input-pasted-placeholder.txt`、ASCII の p3 の画面が `-ascii` つきで、無印のほうが何の画面か名前から分からない(定数のコメントを読まないと分からない)。fixture は `resume_screens_test.go` など 3 つの表に名前で入っているので、あとで直す費用は小さくない | `send_codex_paste_test.go:183-186`、`:130-131`。`resume_screens_test.go`、`send_screen_test.go`、`usage_limit_screen_test.go` の表 | 定数を `sendCodexPlaceholderASCIILine` にする。fixture の名前の直しは任意。直すなら 3 つの表の行も一緒に |
| LOW | maintainability | L6. `internal/org/send_screen.go` が 735 行から 785 行になり、800 行の目安の 15 行下になった。次にこのファイルへ足す変更は目安を越える。#322 の screen-rule の変更(`sendTypedMarkerLineBox`)に続く 2 回目の追加で、規則 1 から 4 とそれぞれの正規表現、codex と claude の分岐が 1 ファイルに集まっている。register にこのファイルの行はない | `wc -l internal/org/send_screen.go` が 785(base は 735)。`docs/tech-debt/README.md` の `send_screen.go` を挙げる行は #308、#322 の Known gaps のみで、大きさの行はない | 今は直さない。L3 の Known gaps の行に「`send_screen.go` が 800 行に近い。次の規則の追加の前に、claude の貼り付けの規則と codex の規則(`sendClaudePastesHold`、`sendCodexPasteHolds` とその正規表現)を `send_screen_paste.go` に出す」を 1 項目として足す |
| LOW | maintainability | L7. 計画の経路 `docs/plans/active/2026-11-15-send-codex-paste.md` が、テストのコメントと実機の証拠に書かれている。`/pr` が計画を `docs/plans/archive/` に移すと、この 2 つの経路は切れる(`archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` だけ)。本体のコードのコメントは `plan 2026-11-15-send-codex-paste` と書き、経路を持たない。この repo には両方の形があり(#319 の `send_paste_test.go` の冒頭も `active/` のまま)、動作には影響しない | `internal/org/send_codex_paste_test.go:16`、`docs/evidence/org-send-codex-paste-live-2026-11-15.md:3` | 本体のコメントと同じく、スラッグだけ(`plan 2026-11-15-send-codex-paste`)にする。または `/pr` の移動のあとに直す。任意 |

## Positive notes

- 変えた本体が狭い。`sendTextForDriver` の条件 1 つ、`sendInputHolds` の codex の分岐 1 行、新しい関数 `sendCodexPasteHolds`(10 行)と正規表現 1 つ、`sendInputHasPastePlaceholder` の `switch` で、`typeAndSubmit`、pane のロック、`ResumeText`、印の断りには触れていない。計画の Non-goals(待ちの時間、resume の文、ほかの driver)と diff が一致する
- 規則 4 は「通すのは入力欄が置き換え 1 つだけで、数が打った文の字数と等しいときだけ」という狭い許可で、通らない形はどれも既存の止まり方(Enter を押さずに `text typed but not submitted`)に落ちる。Enter のあとの確かめは数を問わず置き換えを「残っている」とみなすので、「通す側は数を見る、止める側は数を見ない」という向きの非対称が、失敗を安全側に寄せている
- テストの行が、変異の型に沿って並んでいる。数が 1 つ多い、1 つ少ない、UTF-16 の単位数、バイト数、印を数えた数、置き換えが 2 つ(空白あり、なし)、前後の文字、次の行の文字、claude の形の置き換え、別の文の画面、が 1 つずつ行になっている。実機の画面を直接読むので、fixture が作り物でない
- `TestSendCodexPasteProbeMessages_AsTyped` が、テスト内で組み立てた文(日本語の p4 は繰り返しから作る)を、証拠の SHA-256 とバイト数、字数、UTF-16 の単位数、改行数に固定している。文を作り替えると fixture の画面と数が食い違うことを、先にここで知らせる
- 証拠の書き方が誠実で、実機の確かめ(herdr の道)と、通しで走らせていないこと(画面の道)を分けて書いている。コードのコメントのうち `sendInputHolds` の規則 4 の doc(「証拠の 1698 バイトと 4925 バイトの貼り付け」「そこにあった短い貼り付けは入力欄に文を出した」の趣旨)は、観測した点だけを挙げていて、L1 の直し方の見本になる
- 文書は規則(2 つの写し)、`AGENTS.md`、skill(4 つの写し)で同じ事実(codex にも貼り付け、置き換えの形、N は字数でバイトでも UTF-16 でもない)を言い、`cmp` で写しが一致する

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| #319 の Known gaps の行の (e) が falsified で、#326 の Known gaps の行がない(L3)。確かめていないこと: 貼り付けが途中で切れたときの codex の入力欄、2000 字の上限を超える `--raw` の文と 10 KB 超、前の置き換えが残る入力欄への貼り付け、畳まれ始める大きさ、字数と書記素の数、codex の別の版、画面の道の通しの実行 | register を読む人が「codex は貼り付けを受けない」と誤る。確かめていない各項目は、codex の別の版や特殊な文(結合文字、ZWJ の絵文字)で画面の道が `text typed but not submitted` で止まる、または頭が欠ける形で現れうる。止まる向きは安全側 | この run の commit は報告 1 件と insight event に限られ、register に書けない | `/sync-docs` か `/pr` の前に (e) を閉じて #326 の行を足す。各項目の Trigger は、codex の upgrade のときに probe(p0 から p4)をやり直す、字数と書記素の違う文で N を確かめる、2000 字を超える `--raw` の send が止まる報告が出る、など | `docs/plans/active/2026-11-15-send-codex-paste.md`(Risks、Open questions)、`docs/evidence/org-send-codex-paste-probe-2026-11-15.md`(確かめていないこと)、`docs/evidence/org-send-codex-paste-live-2026-11-15.md`(確かめていないこと) |
| `internal/org/send_screen.go` が 785 行で 800 行の目安に近い(L6) | 次の規則の追加で目安を越え、規則 1 から 4 と 2 つの driver の分岐が 1 ファイルで読みにくくなる | 今の変更は 50 行の追加で、分割は別の変更 | `send_screen.go` を次に変えるとき、貼り付けの規則(`sendClaudePastesHold`、`sendCodexPasteHolds`、2 つの正規表現)を `send_screen_paste.go` に出す | `internal/org/send_screen.go`(`sendInputHolds`)、`docs/plans/active/2026-11-15-send-codex-paste.md`(D2) |

_(register には書いていない。この run は報告 1 件と insight event だけを commit する指示のため。L3 の (e) の取り消しと #326 の行は `/sync-docs` か作者が足す。L6 は L3 の行の 1 項目にするのが安い。L1、L2、L4、L5、L7 は register に載せる性質ではなく、直すか、そのまま PR に残すかの判断。)_

## Recommendation

- Merge: yes。CRITICAL 0、HIGH 0、MEDIUM 0、LOW 7
- Follow-ups: LOW 7 件はどれも直さなくても動きは変わらない。L1、L2、L4、L7 はコメントと文言、L5 は名前、L3 と L6 は register の行。L1 と L2 と L4 は Go のコメントだけの変更だが、`internal/org/` のコードに触れるので「記録だけの修正」の例外には入らず、直すならパイプライン全体の再走になる(上限 2 回)。L3 は `/sync-docs` の側で足せる。ほかの理由で再走するなら L1、L2、L4 を一緒に直し、そうでなければ PR 本文に残して `/sync-docs` に L3 を任せるのが安い
- 未確認: 実機の動作と、結合文字を含む文での N(`/test` と L1(c) の側)、画面の道の ralph を通した実行(証拠自身が未確認としている)、`go vet`、`gofmt`、golangci-lint(`/verify` の側)。この run で走らせたのは `go test ./internal/org/ -run 'Paste|SendInputHolds' -count=1` だけ(`ok`)
