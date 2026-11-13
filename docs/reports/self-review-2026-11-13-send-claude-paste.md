# Self-review report: send-claude-paste

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-send-claude-paste.md`(issue #319)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目
- Scope: `git diff bb416c48...HEAD`(HEAD `6d67ef75`)。12 ファイル、1107 行追加、13 行削除。Go は 5 ファイル(396 行追加、11 行削除)で、残りは文書、計画、証拠。コミットは 5 つ(`c4faa627` 計画と probe の記録、`331eebd3` 本体とテスト、`0d8f04e1` 文書、`1e3dad38` 実機の記録、`6d67ef75` 受け入れ条件の印)で、すべて見た。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の範囲)。テストは回していない

## Evidence reviewed

読んだもの。

- `internal/org/send_paste.go`(46 行、全部)と `internal/org/verbs.go` の差分の全部。`Send` の本体(`checkNoPasteMarkers` から `typeAndSubmit` まで)と、`typeAndSubmit`(`verbs.go:845`)、`confirmSubmitted`
- テストの差分: `send_paste_test.go`(323 行、全部)、`send_fallback_test.go` と `pane_lock_test.go` の期待の書き換え 3 か所
- `.claude/rules/ralph/agent-messaging.md` の新しい節と、`templates/base/` の写し(`cmp` で同じ)。`AGENTS.md` の差分(語単位)と、挿入位置の前後
- `docs/tech-debt/README.md` の #308 の行(4 つのセルの取り消し線)と、新しい行(区切りの `|` の数)
- `docs/evidence/org-send-paste-probe-2026-11-13.md` の A1 から B2、S1、打ち込みが切れたときの節。`docs/evidence/org-send-paste-live-2026-11-13.md` の先頭。issue #319 の本文(`gh issue view 319`)
- 比べた先: `internal/org/send_screen.go` と `resume_screens.go` の driver の見分け方(`driver == "claude"`)、`DirectorDriverClaude`(`director.go:124`)、`internal/cli/org.go` の `send` の `Long` と `--raw` の説明

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 打つ文と記録する文の分け方: `typeAndSubmit` に渡すのは `sendTextForDriver(seat.Driver, p.Text)` の 1 か所だけで、`sent` の Details(`truncateForDetails(p.Text)`)、dry-run の Details、`sendScreenSubmitChecks(screenPath, seat, p.Text)` は元の文のまま。diff のどの hunk も `p.Text` の読み手を印付きの文に変えていない
- `typeAndSubmit` を共有する `ralph org resume` は文を変えない(`ResumeText` は素のまま)。`unlock()` の位置、`PaneSendText` の失敗で Enter を押さない向き、`SendProgress*` の分岐は diff の外で、context の行が変わっていない
- 断りの位置: `protocol.Validate` のあと、`isTask` と見張りの確かめより前、`findSeat` より前。dry-run も同じ答えになる。`Send` の呼び手は本番で `internal/cli/org.go:1440` の 1 か所だけ(`grep`)なので、ralph が自分で組んだ文(ALERT、leader の TASK など)が新しい断りに当たる道はない
- `null` と空: `seat.Driver` は string なので nil の道はない。空の driver は素の文(テストに行がある)。`sendTextForDriver("claude", "")` は印だけの文を返すが、`--raw` でも空の `--text` が `Send` まで来るかは CLI の検査の側で、ここでは変わらない
- エラー: 新しい `errSendPasteMarker` は `org: send:` の接頭辞で既存の文言とそろい、何も打たない(`nothing was sent`)ことを言う。新しい `_ =` や握りつぶしはない
- 写しの一致: `agent-messaging.md` と `templates/base/` の写しは `cmp` で同じ。#308 の行の (c) は 4 つのセルと関連のセルの前後で取り消しがそろっている。新しい行は 5 セル(`|` が 6 本)で隣の行と同じ。証拠 2 件に `$HOME` の実パス(`/Users/...`)はない
- ミューテーションの手がかり: テストの期待は印を定数ではなくバイト列で書いている(`pastedForTest`)ので、定数を書き換えても期待が追従しない。`"Claude"` の大文字、`gemini`、空の driver の否定の行がある

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1. 頭が欠ける大きさを「約 800 バイトを超える文」と書いているが、観測はその形ではない。issue #319 の表では 646 バイトは全部届き、1117 バイトは先頭 1022 バイトが消え、1329 バイトと 1829 バイトは全部届いた。800 バイトは claude が塊を貼り付けに畳む大きさで、欠ける境ではない。#319 自身が「頭が消える条件」を「確かめていないこと」に挙げ、probe の A1 は 1117 バイトの 1 回だけ。読み手は 800 バイトを超えればいつも欠けると読み、800 バイト以下は安全だと読む | `send_paste.go:23-27`(`sendTextForDriver` の doc)、`verbs.go:285-287`(`Send` の doc)、`.claude/rules/ralph/agent-messaging.md:117-119` とその写し。`gh issue view 319` の表と「確かめていないこと」。probe の「環境と分離」の表 | 観測をそのまま書く: 「1117 バイトの文で先頭 1022 バイトが消えた(A1)。ほかの大きさでは消えず、条件は特定していない」。3 か所と写しの 1 か所で同じ文に直す |
| LOW | maintainability | L2. `--raw` の説明が、この diff の後の動きより広い。`--raw` でも印を含む文は断るのに、`send` の `Long` は「Pass --raw to bypass validation entirely」、フラグの説明は「bypass typed message protocol validation」のまま。`agent-messaging.md` の Size cap 節も「`--raw` bypasses validation entirely」で、17 行下の新しい節の「`--raw` and `--dry-run` included」と同じ文書の中でぶつかる。貼り付けで打つことも断りも `send --help` に出ない | `internal/cli/org.go:1363`、`:1528`(この diff は `internal/cli/org.go` に触れていない)、`.claude/rules/ralph/agent-messaging.md:91`、`:111-124` | `Long` に 1 文足す(claude の座席には貼り付けで打ち、印を含む文は `--raw` でも断る)。「entirely」は「protocol validation」に絞る。`/sync-docs` が拾う範囲でもあるので、そちらで直してもよい |
| LOW | maintainability | L3. 下流の project に配る写し(`templates/base/.claude/rules/ralph/agent-messaging.md`)に、下流の読み手が辿れない参照が 2 つある。`(issue #319)` は ralph の repo の issue で、番号だけでは repo が分からない。回復の手順の「Ctrl+C を pane で 1 回」は、座席の pane にキーを送る command を書いていない(probe は `herdr pane send-keys <pane> ctrl+c` を使った)。同じ文書の他の参照は package 名で、「ralph repository source, not part of a scaffolded project」と断っている | `agent-messaging.md:118-119`、`:125-134`(写しも同じ)。probe の「打ち込みが途中で切れたときの確かめ」の `herdr pane send-keys <pane> ctrl+c` | 2 つの写しとも `(issue #319)` を落とす。Ctrl+C の行に `herdr pane send-keys <pane> ctrl+c` を書く。2 つの写しは `cmp` で同じに保つ |
| LOW | security | L4. 断りは 7 ビットの `ESC [200~` と `ESC [201~` の部分文字列だけを見る。8 ビットの CSI(U+009B に `201~`)を Claude Code の入力の解析が終わりの印として読むかは、probe にも計画にもない。読むなら、`ralph org send` に渡す文が外の文章を含むとき、その文章で貼り付けを閉じ、残りを打鍵として座席に入れられる。読まないなら害はない。**未確認**で、根拠はこの repo の外 | `send_paste.go:41-46`(`strings.Contains` が 2 本)。`docs/evidence/org-send-paste-probe-2026-11-13.md` に U+009B の行はない(`grep`) | コードは変えず、#319 の Known gaps の行に (f) として「8 ビットの CSI は確かめていない」を足す。確かめるなら、入力欄に出るだけで Enter を押さない貼り付けで足りる(claude の手番を使わない) |
| LOW | readability | L5. `TestOrgSend_PasteMarker_RefusedWithNothingSent` の表の `want` が、ある行では完全な文言、別の行(`an invalid message with a marker`)では末尾が `: ` で終わる接頭辞になっている。区別は `HasPrefix` のあとの `if tc.want == sendPasteMarkerErr` で、定数と等しいかどうかに頼る。行を足す人は、どちらの意味の `want` か読み解く必要がある | `send_paste_test.go:119-136`(`want`)、`:152-157`(2 段の比較) | `want`(完全一致)と `wantPrefix` の 2 つのフィールドに分ける。または、protocol の検査が先であることを見る行を別のサブテストに出す |
| LOW | readability | L6. 8 引数の `typeAndSubmit` の呼び出しに入れ子の呼び出しが入り、素の `false`、`beforeEnter`、`confirm` と並んで、1 行が約 150 字になった。「`p.Text`(確かめに使う文)と、打つ文(`sendTextForDriver` の戻り)が別」という、この変更の中心が、直前のコメントにだけあり、呼び出しの行からは見えない | `verbs.go:612-615` | `typed := sendTextForDriver(seat.Driver, p.Text)` を 1 行に分けて、コメントをその行に付ける。任意 |

## Positive notes

- 変えた範囲が狭い。本体は新しい 46 行のファイルと、`Send` の 2 か所(断り 4 行、`typeAndSubmit` の引数 1 つ)で、`typeAndSubmit` の本体、pane のロック、画面の道の規則には触れていない。計画の Non-goals(codex、`send_screen.go`、`resume_screens.go`)と diff が一致する
- `sendTextForDriver` は副作用のない関数で、driver の名前の比較は `send_screen.go` と同じ `"claude"` の文字列(完全一致)。大文字の `Claude` を素の文に落とす行が、ゆるい一致への書き換えを止める
- 断りの順序を 3 つ同時に固定するテストがある: protocol の検査が先(`an invalid message with a marker` の行)、見張りの確かめより前(`gateCalled`)、座席の有無によらない(`no-such-seat` の行)。manifest の件数と、2 つの fake(`h.calls()` は resume 用の記録、`h.fakeHerdr.calls` は土台の fake)の両方を見ている
- AC7 のテストは herdr の道と画面の道の両方で、`PaneSendText` が失敗したときに Enter を押さないこと、印付きの文を 1 回だけ試したこと、pane のロックが空いたことを見る。既存のテストの期待の書き換えは claude の行だけで、codex の行は変えていない(`typedForTest(tc.driver, ...)`)
- 運用の注意(`[Pasted text #N +M lines]` が残りうること、Ctrl+C を空の入力欄に押すと `Press Ctrl-C again to exit` になること)を、証拠に基づいて文書に書いている。回復の手順が `ralph org read` を先に挙げる順序は、既存の CLI の通知(`orgReadCommandHint`)とそろっている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 8 ビットの CSI(U+009B)の終わりの印を Claude Code が読むか(L4) | 読むなら、断りの検査を越えて貼り付けを閉じる文が作れる | この run の commit は報告 1 件に限られ、register に書けない。確かめには claude の画面が要る | #319 の Known gaps の行に (f) を足す(L4 の推奨)。または Claude Code の upgrade のときの probe に加える | `docs/plans/active/2026-11-13-send-claude-paste.md`(D3) |

_(register には書いていない。この run は報告 1 件だけを commit する指示のため。L4 を (f) として足すかは作者が決める。L1 から L3、L5、L6 は register に載せる性質ではなく、直すか、そのまま PR に残すかの判断。)_

## Recommendation

- Merge: yes。CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6
- Follow-ups: LOW 6 件はどれも直さなくても動きは変わらない。L1 から L3 はコメントと文書の文言で、L3 は写し 2 つを同じに保つ。L4 は Known gaps の 1 行。L5、L6 は読みやすさ。直すと本体のコメント、`internal/cli/org.go` の文字列、出荷する写しに触れるので、「記録だけの修正」の例外には入らず、パイプライン全体の再走になる(上限 2 回)。ほかの理由で再走するなら一緒に直し、そうでなければ PR 本文に残して `/sync-docs` に L2 を任せるのが安い
- 未確認: 実機の動作(`/test` と AC5 の記録の側)、`go vet`、`gofmt`、golangci-lint(`/verify` の側)、L4 の 8 ビットの CSI
