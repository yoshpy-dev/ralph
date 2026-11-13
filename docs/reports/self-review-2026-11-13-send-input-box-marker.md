# Self-review report: send-input-box-marker

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-send-input-box-marker.md`(issue #322、Design decisions D1〜D3)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目
- Scope: `git diff bbe2807d...HEAD`(HEAD `89a59858`)。29 ファイル、1477 行追加、10 行削除。コミットは 6 つ(`72eba9f1` 計画と証拠、`ae2e336e` 計画の承認、`a953ff11` 本体と fixture とテスト、`538eb5ce` 計画の進捗、`98b243bb` prompt と skill、`89a59858` 受け入れ条件の印)。Go の本体は `internal/org/send_screen.go`(+98 行、-10 行は doc comment の組み替えを含む)の 1 ファイルだけで、残りはテスト、fixture 11、prompt 2、skill の写し 8、証拠、計画。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは `/verify`、`/test`、`/sync-docs` の範囲なので評価していない。テストは走らせていない(読んだだけ)

## Evidence reviewed

読んだもの。

- `internal/org/send_screen.go` の差分の全部と、その前後(`SendScreenSubmitted`、`sendInputHolds`、`sendTypedInputBox`、`sendTypedMarkerLineBox`、`sendBoxShowsTheEnd` から `sendScreenBoxCheck` まで)
- 呼ばれる側: `resume_screens.go` の `resumeInputBox`、`findResumeInputBox`、`resumeMarkerLineBox`、`resumeInputBoxShape`、`codexFooterStart`、`isResumeRule`、`resumeMarker`
- テストの差分の全部: `send_screen_test.go`(+329 行)、`send_fallback_test.go`(+43)、`resume_screens_test.go`(+51)、`usage_limit_screen_test.go`(+14)、`prompts_send_unsubmitted_test.go`(111 行、全部)。fixture は `claude-input-typed-marker-line.txt` と `codex-input-typed-marker-line-scrolled.txt` を実際に開き(行頭の空白、U+00A0)、残りは名前と、テストが期待する頭の行
- prompt と skill の差分: `leader.md`、`director.md`、`/org` skill の `send` の行、`/director` skill の「答える順」の手順 2。4 つの写しの一致は `cmp`(`/org` と `/director` について、`.claude/skills/` と `templates/base/.claude/skills/`、`.agents/skills/` と `templates/base/.agents/skills/`)
- 文面の裏取り: `verbs.go:637-648` のエラーの文(`(text typed but not submitted)` が 2 か所)と、`internal/cli/org.go:1466-1478` の `SendProgressTextTyped` の note
- `docs/tech-debt/README.md` の #308 の行(473、482〜484)と、`wc -l` による行数(`send_screen_test.go` は `bbe2807d` で 831、今 1160。`send_fallback_test.go` は 835 から 878)
- fixture と証拠(`docs/evidence/org-send-marker-probe-2026-11-13.md`)に `/Users/` や個人のメールアドレスがないこと(`grep`)
- 前例の形式: `docs/reports/self-review-2026-11-13-send-claude-paste.md`

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 添字の安全: `sendTypedMarkerLineBox` の `lines[last.marker]`、頭を探す `lines[i]`(`i >= 0`)、`resumeInputBoxShape` の `lines[idx-1]`(`idx == 0` を先に見る)、`sendCodexFooterInInput` の `lines[end-1]` と `lines[start]`(`start < end`)。頭の行 M は必ず L より上(`i` は `last.marker-1` から始まる)で、M == L にならない
- 前に通った画面の答えが変わらないこと: 新しい道は `sendTypedInputBox` の先頭で呼ばれるが、条件 1(L の印の前に空白がある)で、頭が 0 桁目の実機の画面と、0 桁目の番号付きの頭の合成 fixture(`*-input-typed-numbered-steps.txt`)はすぐ抜ける。`findResumeInputBox`、`ClassifyResumeScreen`、`ResumeInputHolds`、`ClassifySendScreen` のコードは diff に入っていない
- 条件 4 の claude の `box.inputEnd <= last.marker` は到達できる。`isResumeRule` は空白を `TrimSpace` するので、2 字下げの罫線は条件 3(空行か 2 字下げ)を通り抜け、この行で止まる。テストに「二つの空白の付いた下の罫線」の行がある。codex の footer も 2 字下げで条件 3 を通るので、`last.marker >= footer` が止める(「footer の下」の行がある)
- 失敗側の向き: 新しい道が false を返すと、今までの `findResumeInputBox` の道に落ちるだけで、Enter を押す側には倒れない。claude の `resumeInputBoxShape` が失敗しても、box が `resumeInputBox{}` に戻るので `inputEnd(0) <= last.marker` でも止まる(`reason != ""` の行を外しても結果が変わらない、と計画の進捗にあるのはこのため)
- テストの空振り: 拒否の表の編集は `sendMarkerEdit` が「元の文字列がちょうど 1 回」を要求するので、fixture が変わって編集が当たらなくなれば `t.Fatalf` で落ちる。fixture の一覧表は `testdata/resume/` を走査して、名前が合うのに表にない fixture を `t.Errorf` で挙げる
- 写し: skill の 4 つの写しは `cmp` で一致。leader.md と director.md は go:embed の 1 か所で写しがない。prompt のテストは `markdownSection`、`markdownItem`、`containsPhrase` を使い、空白と改行に左右されない
- 秘密と個人情報: fixture 11 と証拠に実パスも個人のアドレスもない(`/private/tmp/ml308/work` は一時ディレクトリ)

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1. #308 の register の行が「次にどちらかのファイルを変えるときは、先に動かす」と書いたその変更が、2 ファイルを大きくした。`send_screen_test.go` は 831 行(800 行の目安を超えて登録済み)から 1160 行へ +329、`send_fallback_test.go` は 835 から 878 へ +43。行の Trigger は「次の変更。`send_screen_test.go` の番号付きの行のテストを別のファイルに移し、`send_fallback_test.go` の時間のテストを別に移す」。#319 は同じ trigger に当たったとき、新しいテストを新しい `send_paste_test.go` に置いて行を carry over にした。この diff は新しいテスト 5 本と定数、表を既存の `send_screen_test.go` の末尾に足した。register の行は diff に入っておらず、831 のまま。ファイルが長いほど、新旧のテストの境が読めなくなる(`sendMarkerEdit` や `sendTypedMarkerFixtures` が、番号付きの行のテストと同じファイルの別の場所にある) | `wc -l`(`bbe2807d`: 831、HEAD: 1160)。`docs/tech-debt/README.md:484` の Trigger と 2026-11-13 の carry-over の文。`send_screen_test.go:860` 以降が新しい部分(`sendMarkerCodexShortMsg` から `TestSendInputHolds_MarkerLineRefuses` まで) | 新しい部分は既存の行を 1 行も動かさない純粋な追加なので、`send_screen_marker_test.go` のような新しいファイルへの移しが安い(`sendScreenFixtureWants` の 11 行と `TestSendScreenSubmitted` の 2 行は表の中なので残してよい)。移さないなら、行 484 に今回の行数(1160 と 878)と carry-over の文を足し、PR の本文で言う。移すと pipeline が `/self-review` からやり直しになる(cap 2)ので、作者が選ぶ |
| LOW | maintainability | L1. leader の prompt と、ralph が出す note が、同じ失敗について逆のことを言う。`internal/cli/org.go:1474-1477` の note は「Check it with 'ralph org read …' and clear or submit it there before sending again」で、leader の prompt は「座席の pane にキーも送りません」。`/org` skill の `send` の行は、新しい括弧の中で「leader は … 座席の pane にキーを送らない」と言った数文あとに、既存の文「次に何をすべきかは note の指示に従う」が残る。leader が prompt どおりに動けば note の「submit it there」には従わないが、どちらが勝つかは書いていない。director の側は Enter 1 回が許されるので、note と矛盾しない | `leader.md` の追加の段落、`.claude/skills/org/SKILL.md` の `send` の行(括弧と「note の指示に従う」)、`internal/cli/org.go:1474-1477` | leader の段落か `/org` の括弧に「この失敗では note の「clear or submit it there」に従わない(leader は escalate)」と 1 文足す。note の文面を変える手もあるが、`internal/cli` のテストが固定しているかもしれず、この slice の範囲を出る |
| LOW | maintainability | L2. 同じ規則が近い場所で 2 回書かれ、片方は閉包になった。「その行の文が打った文の一部か」は、`sendTypedInputBox` の番号付きの道では `text := box.first; if codex { TrimSuffix }; strings.Contains(want, resumeDropSpace(text))`(330-340 行)で、新しい関数では閉包 `partOfWant`(373-378 行)。片方の規則(たとえば矢印を外す条件)を変えると、もう片方が追従しない。さらに `partOfWant` は L と M の両方に使われるが、codex の矢印は頭の行(M)にしか付かないので、L に対する「矢印を末尾から外す」(doc の条件 2 の括弧)は意味がない。footer の始まりを決める 5 行(413-418 行)も、`sendCodexHasFooter` の中の `sendCodexFooterInInput` と同じ呼びを、もう一度打つ形になっている | `send_screen.go:330-340`、`373-378`、`411-420` | 後続の変更があってからで足りる。やるなら `sendTypedInputBox` の番号付きの道も同じ小さな関数(`sendTextInWant(driver, want, text)`)を呼ぶようにし、footer は `sendCodexFooterStart(lines, box) (int, bool)` にして `sendCodexHasFooter` と新しい関数が共有する |
| LOW | readability | L3. `sendTypedMarkerLineBox` は 51 行で、3 つの段(頭の行を探す、頭の行の文を確かめる、箱の中に L があるかを確かめる)が 1 つの関数に並ぶ。頭を探す `for i := last.marker - 1; i >= 0 && head < 0; i--` は、`switch` の中で `break` が `switch` を抜けるため、ループ条件の `head < 0` を終了の印にしている。読み手はこの癖に気づくまで、ループが止まる理由を探す。末尾の `switch driver` は default がなく、知らない driver では箱をそのまま `true` で返す(`sendInputHolds` の表の確かめで、今は到達しないが、同じ `sendInputHolds` の `switch` は `return false` で終わる) | `send_screen.go:372-422`(特に 385-394、404-420、421) | 頭を探す部分を `sendMarkerLineHead(lines, marker, from) int`(見つからなければ -1、`return` で抜ける)に出すと、ループ条件の癖が消え、頭の探し方だけを単体で試せる。`switch` の末尾に `default: return resumeInputBox{}, false` を足すか、`return box, true` の前に driver の確かめを置く |
| LOW | null-safety | L4. 条件 2(L の文が打った文の一部)は、印のあとの文だけを比べるので、文が空の行(`  ❯` だけ)では `strings.Contains(want, "")` が真になり、何も確かめない。番号付きの道が同じ規則を使えるのは、番号付きの行の文が空ではないから。条件 3 と 4(0 桁目の頭の行、箱の形)が後ろにあるので、画面を誤って入力欄とみるまでは行かない。ただ、条件 2 が弁別する力は文が短いほど薄く、空では 0 になる。この関数の doc comment は条件 2 を「文が一部」と書いていて、空の行のときに何も言っていないことは読み取れない | `send_screen.go:373-381`(`partOfWant(last.first)`)。`last.first` は `resumeMarkerLineBox` が `TrimSpace` で作る。空の L を試すテストは diff にない(未確認: 走らせていない) | 空の L を拒否するなら `last.first == ""` を条件 2 に足す(印だけの行を含む文は今までどおり止まる側に倒れる)。許すなら doc の条件 2 に「文が空の行は条件 3 と 4 に任せる」と書く。どちらでも、1 行のテストを足す |
| LOW | maintainability | L5. コードの comment に、`/pr` が archive する前の計画のパス `docs/plans/active/2026-11-13-send-input-box-marker.md` が 6 か所ある(`send_screen.go:351`、`send_screen_test.go:987` と `1076`、`send_fallback_test.go:482`、`prompts_send_unsubmitted_test.go:28` と `66`)。`./scripts/archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` のリンクだけ(`/pr` の Step 8)なので、archive のあとは存在しないパスになる。同じ形の前例は repo に多い(`internal/cli/org_approve.go:13` など)が、`internal/org/director.go:24` や `feature.go:19` は `docs/plans/archive/` で書いている | `grep -n 'docs/plans/active/2026-11-13-send-input-box-marker.md' internal/org/*.go`(6 行) | comment は計画の ID(`D1`、`AC4b`)と issue 番号 `#322` で足りる。パスを残すなら `docs/plans/archive/2026-11-13-send-input-box-marker.md` と書く |
| LOW | readability | L6. `markdownParagraphWith` は、段落を空行で割って印を含むものを数える汎用の helper だが、この機能のテストファイルに置かれ、同じ割り方(`strings.Split(…, "\n\n")`)が `prompts_leader_plan_test.go:225` にも手で書かれている。近くの `markdownSection`、`markdownItem`、`containsPhrase` は `prompts_test.go` にあり、それぞれ自分の単体テストがあるが、これには無い | `prompts_send_unsubmitted_test.go:17-25`、`prompts_test.go:27`、`80`、`145`、`prompts_leader_plan_test.go:225` | `prompts_test.go` の `markdownItem` の隣に移し、`TestMarkdownItem_…` と同じ形の小さな単体テストを付ける。急がない |

## Positive notes

- 変えた Go のコードが狭い。本体は新しい 1 関数と、`sendTypedInputBox` の先頭の 3 行で、resume と打つ前と Enter のあとの確かめが共有する `findResumeInputBox` には触れていない(plan の D1)。新しい道は失敗しても今までの道に落ちるだけで、Enter を押す側には倒れない
- D2 の 5 つの条件が、doc comment の番号、コードの分岐、拒否の表の行の 3 つで対応している(条件 1 は「0 桁目の頭」の行、2 は「打った文でない L」の 2 行、3 は「0 桁目の行がない」と「間に 0 桁目の行がある」、4 は罫線の下と 2 字下げの罫線の下と footer の下と罫線なしと footer なし、5 は番号付きの頭が打った文でない 2 行)。テストの表は 1 つの条件だけを外した画面で、他の条件が原因で拒否されているのではないことを `sendMarkerEdit` の編集で示している
- fixture の名前を走査する表の網羅の検査(`sendTypedMarkerName`)で、あとから足した fixture が表から漏れない。定数の文(`sendMarkerCodexShortMsg` など)が証拠のバイト数と改行数に合うことを `TestSendMarkerProbeMessages_AsTyped` が固定していて、画面と打った文の食い違いに気づける
- 結合テストは `via=screen` だけでなく、`SubmitConfirmed`、`submit_unconfirmed` の不在、herdr の呼びの並びとその回数(`keys pane-1 Enter` が 1 回)、打った文、pane のロックが空いたことまで見る
- prompt のテストは、leader は節の中の段落 1 つに、director は箇条 1 つに閉じて語を固定し、director の箇条が `pane` と `ralph org` を含まないことを、launch_cmd 形の prompt の既存のテストに合わせて確かめている
- 手順の文面が、実際のエラーの文(`verbs.go:639`、`645`)と、note に出る `ralph org read` の呼び(`internal/cli/org.go:1447`)と合っている。director の Enter の許可は、`/director` skill の既存の許可(2036 行)の範囲に収めている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `send_screen_test.go`(1160 行)と `send_fallback_test.go`(878 行)の大きさ(M1)。register の #308 の行(`docs/tech-debt/README.md:484`)の数字(831 と 835)は古くなる | LOW。長いテストを 2 つ読むことになる。振る舞いは正しい | この run の commit は報告と insight event に限られ、register を書けない。新しい部分を別ファイルに移すかは作者が選ぶ(移すと pipeline が最初からになる) | M1 の推奨: 新しい部分を新しいファイルに移す。移さないなら、行 484 の数と carry-over の文を直す | `docs/plans/active/2026-11-13-send-input-box-marker.md`(Scope)、`docs/tech-debt/README.md:484` |

_(register には書いていない。この run は報告と insight event だけを commit する指示のため。上の行の追記か、M1 を直すかは作者が決める。L1 から L6 は register に載せる性質ではなく、直すか、PR にそのまま残すかの判断。)_

## Recommendation

- Merge: yes(CRITICAL と HIGH はない。MEDIUM 1 件、LOW 6 件。M1 は、新しい部分を別ファイルに移すか、register の行 484 に数と carry-over を足すかのどちらかを、PR の前に済ませるのが望ましい)
- Follow-ups:
  - M1: 新しいテストを `send_screen_marker_test.go` に移す(純粋な追加なので差分が小さい)。移さないなら行 484 を更新する。どちらでも `sendCodexFooterInInput` などの本体は動かさない
  - L1: leader の段落か `/org` の括弧に「note の clear or submit it there には従わない」の 1 文。prompt の文を変えるので、`prompts_send_unsubmitted_test.go` の固定の語も合わせる
  - L2 から L4: 本体の小さな整理。L4 の空の L の扱いは、直すなら 1 行のテストと一緒に
  - L5、L6: comment のパスと helper の置き場。急がない
  - 計画の Risks が「Known gaps に残す」とした、頭が 0 桁目にあることへの依存(ほかの版の CLI が入力欄を字下げすると、選び直しが働かず今までどおり Enter を押さない)は、`docs/tech-debt/README.md` にまだ行がない。`/sync-docs` か PR の前に #308 の行の続きとして足す
  - 計画の進捗が挙げる、変異で生き残った `reason != ""` の行は、上の「指摘なしで確かめたこと」のとおり、次の `inputEnd <= last.marker` が同じ結果にするためで、害はない。直すなら、行を外すか、残す理由を comment に 1 行書く
