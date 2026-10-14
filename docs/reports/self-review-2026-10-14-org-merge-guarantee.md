# Self-review report: org-merge-guarantee

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-guarantee.md(承認済み、digest d2923a8bf359。7 段目の 7c-2)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1、`.harness/state/standard-pipeline/cycle-count.json` は 1)
- Scope: diff の品質だけ。`git diff a16d7c4f...HEAD`(HEAD 8d1c6693、9 コミット、23 ファイル、+3925/-89)。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の Design decisions C1〜C7 と Progress checklist にある決定(pr_checks_only の勧めが ruleset の merge queue だけを名指すずれ、ssh の別名・fork の origin・管理者の bypass・host の文字列比較・`mergeguarantee.go` が約 830 行・get-url が 2 回走る・codex model slug の検査が TestMain で固定されない、の tech-debt 6 行)は、欠陥として数えていない

## Evidence reviewed

- 読んだ差分(コード、全体): `internal/org/mergeguarantee.go`(834 行すべて)、`internal/cli/doctor_merge_guarantee.go`、`internal/cli/doctor.go`・`status.go`・`org_merge_order.go`・`main_test.go` の hunk、`internal/org/mergeorder.go` の hunk(`MergeOrder` の読み込み、`mergeOriginFromURL` への分け直し)
- 読んだ差分(文書): `/org` skill の 2 か所(動詞リファレンスの `merge-order` の行と「merge してよい順」の節。`.agents/` 側と `templates/base/` 側の 2 組は `cmp` で root と同一)、README の 3 か所、AGENTS.md の repo map の 1 句、仕様 FR-7 の 7c-2 の注、`docs/tech-debt/README.md`(追加 6 行と変更 1 行。追加した行の `|` で分けた列は 5 で、見出しと合う)、`docs/evidence/merge-guarantee-livefire-2026-10-14.md`(`$HOME` は `<scratch>` に置き換えてあり、`/Users/` も token もない)
- 読んだテスト: `internal/org/mergeguarantee_test.go` の 1〜140 行と、`TestMergeGuarantee_HostComesFromOriginsURL`(origin の URL の認証情報が JSON に残らないことの確認)、`internal/cli/doctor_merge_guarantee_test.go` の seam と TestMain の固定の確認(100〜152 行、335〜465 行)、`mergeorder_test.go` の追加と削除の hunk、既存テストから消えた行の一覧。残りは名前の一覧だけを読んだ
- 読んだ周辺のコード(定義と呼び出しを確かめるため): `realMergeOrderExec`(stderr を `%w: <stderr>` で err に付ける)、`readMergeOriginRepo`、`writeJSONFileAtomic`(temp ファイル、rename、失敗時の `os.Remove`)、`ResolveOrgStateDir`、`PrintableInboxText`・`inboxField`、doctor の結果の出力ループと終了コード(`warn` は終了コードに効かない)、`Org.now`・`ManifestStore.Path`
- 確かめたこと
  - `git diff --check` は空。追加した Go の行に TODO・FIXME・`println`・debug 出力・`nolint` はない。`t.Skip` は 1 件(root で書き込めてしまう読み取り専用ディレクトリのテスト)だけ
  - origin の URL は、どの Detail にも cache の field にも入らない。`mergeOriginFromURL` が返すのは host(user・password・port を除き小文字)と owner/repo だけで、URL を format に渡す行はない(`readOrigin`、`readMergeGuaranteeOrigin`)。`git remote get-url origin` が失敗したときの err も stderr の文だけ
  - gh を実行する場所は `mergeGuaranteeRun.gh`(550 行)の 1 か所で、4 回の呼び出しはどれも `ghArgs` を通るので `--hostname` と `GH_PROMPT_DISABLED=1` が付く。git は `getMergeOriginURL` の 1 か所で `GIT_TERMINAL_PROMPT=0` 付き。owner・repo・branch は `url.PathEscape` を通してから path に入る
  - `readMergeRules` は、1 つの配列、連結した配列、空の出力(`pages == 0` でエラー)、途中で切れた JSON(`io.ErrUnexpectedEOF` はエラー側)を分けている。decode の変数 `page` は繰り返しごとに宣言し直しているので、スライスが使い回されない
  - cache の書き込み: stateDir がなければ作らず `NoLedger`、書けなければ古い cache を消し、消せなければ `RemoveErr` を残す。`mergeGuaranteeResult` の `switch` の順(NoLedger、Written、RemoveErr、Removed、それ以外)で 4 つの結果が混ざらない。doctor が `warn` にするのは `RemoveErr` のときだけ
  - 出力のエスケープ: status と merge-order の text は `mergeGuaranteeLine` が `Text()` 全体を `inboxField` に通す。JSON は encoding/json。cache から来る `detail`・`host`・`repo`・`checked_at` はどちらの経路でも生では出ない
  - 古くなった doc コメント: `MergeOrder` の「no git call」は書き換え済み。`internal/org/`・`internal/cli/`・README・skill・仕様を「no git」「git を打たず」「network」で検索して、「台帳に対象がなければ git を打たない」と今も言う場所はなかった(skill、README、`org_merge_order.go` の help は「保証のキャッシュが新しいときのローカルの get-url だけ」と書く)
  - `mergeRepoFromOriginURL` を `mergeOriginFromURL` に分けた変更で、`!ok` のときの戻り値が「埋まった repo」から「ゼロ値」に変わった。呼び出し側(`readMergeOriginRepo` → `o.repo`)は `readable` が true のときしか `o.repo` を読まないので、挙動は変わらない
  - doc コメントの置き場: 新旧の Go ファイルで、コメントの最初の語が後ろの宣言名と合い、空行で `func` から離れている箇所を python で拾ったが 0 件(目視の補助で、網羅の証明ではない)
  - skill 4 面は `cmp` で同一。追加行に `internal/`・`cmd/`・段の名前・plan のパスはない。tech-debt の追加行にエスケープしていない `|` はない
- 確かめていないもの: テスト、`gofmt`、`go vet`、`-race` は走らせていない。実際の gh の出力(特に `%2F` を含む既定のブランチ名への各 endpoint の応答、gh の stderr に制御文字が入るか)は確かめていない。実機の出力は `docs/evidence/` のファイルを読んだだけで、打ち直してはいない

## Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 1、LOW 6。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | tech-debt の登録簿の 226 行目(「Nothing stops a feature org's leader from pushing and opening the PR without `ralph org gate` ...」)の Trigger 列が「Stage 7c-2 (`ralph doctor`'s protected-branch warning, `ralph status`'s guarantee display), or a report of a feature PR merged without a gate」のまま。この PR がその 7c-2 で、doctor の warn と status の保証の行を入れたのに、行は閉じも書き換えもされていない。同じ PR が 233 行目の 7c-1 の行(「Once one feature PR is merged ...」)は 7c-2 に合わせて書き換えているので、同じ見方が 226 行目に届いていない。読む人には「7c-2 のときに見直す」と読めるが、7c-2 は今 merge される | `docs/tech-debt/README.md:226`(Trigger 列)。`git diff a16d7c4f...HEAD -U0 -- docs/tech-debt/README.md` の hunk は `-233 +233` と `+240,6` の 2 つだけで、226 行目は動いていない | 行は閉じない(gate を通さない PR を止める仕組みは今もない)。Trigger 列から「Stage 7c-2 (...)」を外して残る条件(gate なしで merge された機能の PR の報告、または 8a 段目の director)だけにし、Impact 列に「doctor は `pr_checks_only` のとき warn し、status と merge-order は保証の種類を出すが、ralph の gate を通していない PR を止める仕組みではない」と 1 文足す |
| L1 | LOW | typo | `mergeOriginName` の doc の文が崩れている。「mergeOriginName is host/owner/repo, for a cache whose check could not read origin "no readable origin".」は、`"no readable origin"` が何の値か読み取れない(host も repo も空のとき、その文字列を返す、という意味) | `internal/org/mergeguarantee.go:789-790`(関数は 791-796) | 「mergeOriginName is host/owner/repo, or "no readable origin" for a cache whose check could not read origin (host and repo both empty).」のように書き直す |
| L2 | LOW | maintainability | 同じ読み取りが 2 か所にある。`readMergeOriginRepo` と `getMergeOriginURL` は、どちらも `git remote get-url origin` を `GIT_TERMINAL_PROMPT=0` 付きで打ち、出力の 1 行目を `TrimSpace` で取る。前者は URL を repo に読み、後者は host と repo に読む。「get-url が 2 回走る」の登録済みの行(b)は実行が 2 回になることを言っていて、読み取りの実装が 2 つあることは言っていない。また `ReadMergeGuarantee` の get-url の期限は固定の 10 秒(`defaultMergeGuaranteeGitTimeout`)で、`MergeOrderParams` に調節する欄がなく、doc の「Zero or less means 30 s and 15 s」にも出ない | `internal/org/mergeorder.go:684-697`(`readMergeOriginRepo`)と `internal/org/mergeguarantee.go:591-598`(`getMergeOriginURL`)。`MergeOrderParams` の doc は `mergeorder.go:138-141`。`readMergeGuaranteeOrigin` の期限は `mergeguarantee.go:767-770` | `readMergeOriginRepo` が `getMergeOriginURL` を呼んで URL を読むだけにする(期限の扱いは今の `getCtx` のまま)。行(b)を直すとき一緒に直し、`MergeOrderParams` の doc に 10 秒の get-url を 1 句足す |
| L3 | LOW | security | doctor が出す Detail の gh の stderr は、1 行にして 300 バイトで切るが、制御文字は残る。`mergeGuaranteeErrText` の `strings.Fields` が分けるのは空白類だけで、ESC などは通る。status と merge-order は同じ文を `inboxField` に通してエスケープするのに、doctor の出力ループ(`fmt.Printf(" — %s", r.Detail)`)は生のまま端末に出す。stderr は gh が出す文で、GitHub(または GHES)が返す `message` を含むので、到達は限られる。doctor の他の検査も err を生で出す(`codex unusable: %v` など)ので前例はある | `internal/org/mergeguarantee.go:600-612`、`internal/cli/doctor.go:207-208`、`doctor_merge_guarantee.go:90`。doc の「on one line」は空白の畳み込みだけを指す | `mergeGuaranteeErrText` の最後で `org.PrintableInboxText(s, false)` を通す(cache に入る Detail も同じ文なので、status 側の二重のエスケープは無害)。直さないなら、doc の「on one line」を「whitespace collapsed」に直す |
| L4 | LOW | readability | strict を読む 403 の分岐は、原因によらず「admin のアクセスがないと読めない」と言い切る。`status == 403` や `(HTTP 403)` は、rate limit や SAML SSO の未承認でも出る。末尾の括弧に gh の err 文が入るので原因は見えるが、前半の文は観測していない原因を主張する | `internal/org/mergeguarantee.go:460-463`(`body.Message == "Not Found" \|\| fmt.Sprint(body.Status) == "403" \|\| ...`)と 461-463 の文 | 403 のときは「admin のアクセスがない、または 403 になる別の理由(rate limit、SSO)で」と言う。404 の `Not Found` だけを、今の文のままにする |
| L5 | LOW | readability | status と merge-order の行は、branch と repository を 2 回言う。`Text()` が括弧に `<branch> of <host>/<repo>` を出し、そのあとに続く cache の Detail も `subject()`(「main, the default branch of origin's repository o/r, ...」)で同じことを言う。実機の証跡の行がそのとおりで、1 行が長くなる(`github_protection (main of github.com/github/docs, checked ...): main, the default branch of origin's repository github/docs, requires its 37 ...`)。doctor の Detail は 1 つだけで読むので重複しない | `internal/org/mergeguarantee.go:822-834`(`Text`)と `:471-474`・`:512-520`(`protected`、`subject`)。証跡は `docs/evidence/merge-guarantee-livefire-2026-10-14.md` の status の行 | `Text()` の括弧を `checked <checked_at>` だけにするか、Detail を `subject()` なしの短い形にして doctor 側で subject を前に足す。直すと文書の例の行が変わるので、費用は /sync-docs の 1 回 |
| L6 | LOW | maintainability | 「gh を打つ、command 名を作る、`stopped` を見る」の 3 行が 4 か所に同じ形である。`readRepository`、`readMergeRules`、`readMergeClassicStrict`、`ghJSON`(`ghJSON` は 2 つの step の分だけを包む)。`g.gh(...)` と `g.command(...)` は引数が同じで、順序の入れ替えや `command` の引数の取り違えを防ぐものがない。ファイルが 834 行で 800 行を超える原因の 1 つで、登録済みの行(a)の対処は「cache の半分を別ファイルへ」だけを言う | `internal/org/mergeguarantee.go:289-293`、`:347-352`、`:436-440`、`:560-565` | `call(paginate, path) (out, cmd string, err error, stopped string)` のような 1 つの helper に寄せ、`ghJSON` を `readRepository` にも使えるようにする。直さないなら行(a)に「3 行の繰り返しも 1 つに寄せる」と足す |

## Positive notes

- origin の URL を持ち回る値がなく、認証情報つきの URL からは host と owner/repo しか取らない。`TestMergeGuarantee_HostComesFromOriginsURL` が、`https://token:...@host:8443/...` の origin で JSON に `token` も `oauth` も残らないことと、`--hostname` が小文字の host になることを同時に固定している
- 4 回の gh と 1 回の git の呼び出しが、それぞれ 1 か所の関数を通る。後から `--hostname` を付け忘れる gh の呼び出しを足しにくい作りで、`GH_HOST` を効かせない(C7)の実装が 1 か所に集まっている
- `ReadMergeGuarantee` は、安い判定(ledger の有無、cache の有無、版、古さ)を先にして、git は cache が新しいときだけ打つ。この順が doc コメント、help、README、skill の 5 か所で同じ言い方で書かれている
- 失敗の分類が「何を読めなかったか」を Detail に持つ(gh が PATH にない、gh が失敗、JSON が読めない、期限切れ、割り込み)。期限切れと割り込みを `stopped` で分けているので、ctx が切れた gh の失敗が「gh の失敗」として出ない
- TestMain の固定が `doctorShellAliasEnv`・`doctorCodexSandboxEnv` と同じ形(package 変数の seam)で、固定が効いていることを `TestRunDoctor_TestMainPinRunsNoGitOrGh` が stub の git と gh で確かめている
- 既存テストの書き換えは、出力に行が 1 つ増えたことに合わせたものだけで、消えた assertion に対応する新しいテストがある(`NoLedgerPrintsTheNoBranchesNoteWithoutRunningGit`、`OrgWatchLineFollowsTheStateDirLine` の行数)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L2: `readMergeOriginRepo` と `getMergeOriginURL` の読み取りが 2 つ。`ReadMergeGuarantee` の get-url の 10 秒が `MergeOrderParams` に出ない | 片方だけ直すと、もう片方の env や期限が古いまま残る | 小さな差分で直せるので、cycle 1 では登録せず直す | 直さずに最後の cycle に届いたとき(1 行にまとめて登録する) | `docs/reports/self-review-2026-10-14-org-merge-guarantee.md`(L2) |
| L6: gh の呼び出しの 3 行が 4 か所にある | `mergeguarantee.go` が 800 行を超え続ける | 登録済みの行(a)の分割と一緒に直すのが自然 | 行(a)の分割をするとき | 同(L6)、`docs/tech-debt/README.md` の org-merge-guarantee の leftovers の行 |

_(登録簿には追記していない。cycle 1 の LOW は次の fix の回で直せる量なので、直さずに最後の cycle に届いたものだけを 1 つの行にまとめて追記する。M1 は新しい debt ではなく、既存の行 226 の書き換え。)_

## Recommendation

- Merge: yes(CRITICAL と HIGH はなし。M1 は登録簿の 1 セルの書き換えで、merge を止める量ではない)
- Follow-ups:
  - M1: 行 226 の Trigger 列と Impact 列を書き換える。登録簿の編集は「記録だけの修正」の例外に入らないので、次の fix の回に含める(パイプラインは全体を回し直す)
  - L1(コメントの文)と L3(エスケープを 1 行足すか、doc を直す)は、ほかの fix と同じ回にまとめると費用が増えない
  - L2、L4、L5、L6 は任意。L5 は文書の例の行が変わるので、直すなら /sync-docs の回に入れる
- Known gaps: 既定のブランチ名に `/` を含むときの各 endpoint(`rules/branches/<b>`、`branches/<b>`、`.../protection/required_status_checks`)が `%2F` を受け付けるかは、実機の証跡が `main` と `trunk` だけなので未確認(fake の exec のテストだけ)。gh の stderr に制御文字が入るかは未確認(L3 は doc と code の読みから)。`gofmt`・`go vet`・テストは /verify と /test で確かめる

## Cycle 2

- Date: 2026-10-14
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2。`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 回の最後)
- Scope: `git diff fb954403..HEAD`(HEAD 59abeaec、2 コミット、4 ファイル、+97/-24)。59abeaec(`internal/org/mergeguarantee.go`、`internal/org/mergeguarantee_test.go`、`docs/tech-debt/README.md` の 244 行目)と 8d6499c4(計画の進捗に 1 行)。全体の差分 `git diff a16d7c4f...HEAD` は、コード側の hunk(`mergeorder.go`、`status.go`、`org_merge_order.go`、`doctor.go`)と `doctor_merge_guarantee.go` を読み直した。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs が受け持つので行っていない。計画の進捗にある cycle 2 の決定(直すのは L3、L4、test の Gap 1。L2・L5・L6 と Gap 4 は登録簿の行へ。cycle 2 で新しく出た LOW は直さない)は、欠陥として数えていない

### 読んだものと確かめたこと

- 読んだ: `git diff fb954403..HEAD` の全体(Go 2 ファイル、登録簿の 244 行目、計画の 1 行)。周辺は `mergeguarantee.go` の定数(80-106)、`decide`(212-252)、`readRepository`(287-315)、`readMergeRules` から `readMergeClassicStrict` まで(342-475)、gh の helper と `mergeGuaranteeErrText`(545-626)、cache の保存と読み(628-848)。`inbox.go` の `PrintableInboxText`(121-135)、`internal/cli/org_inbox.go` の `inboxField`(437-442)、`doctor_merge_guarantee.go` 全体、`doctor.go` の出力ループ(196-212)、`mergeorder.go` の `mergeOriginFromURL`(719-749)。テストは `mergeguarantee_test.go` の分類の表とその loop(320-415)、`mgAPIError`(241-252)、長いエラーの test(539-553)、新しい 2 本(554-596)
- 確かめたこと(コードを読んで辿った。走らせていない)
  - `git diff --check fb954403..HEAD` は空。追加行に TODO・FIXME・debug 出力・秘密情報はない。`unicode/utf8` を `mergeguarantee.go` の import から外したあと、同ファイルに `utf8.` と `unicode.` を使う行は残っていない(grep 0 件)。test 側は `utf8.ValidString` が残るので import は要り、`unicode` は新しい test が使う
  - 切る順: `mergeGuaranteeErrText` を手で辿った。`strings.Fields` で空白類を 1 つの空白にし、1 文字ずつ `PrintableInboxText` に通して、`b.Len()+len(printable) > 300` になる文字の手前で止めて `...` を足す。escape は 1 文字分が丸ごと入るか入らないかなので、途中で切れない。`\x1b` が 75 個(ちょうど 300 バイト)は切らず、`a` に続く 100 個は 297 バイトで切る。新しい 2 本の期待と合う。fix の前の順(生で切ってから escape)に戻すと 400 バイトに伸びて 1 本目が赤になり、`>` を `>=` にすると 2 本目が赤になる。`é` を 400 個並べた既存の test も、299 バイトで切って 302 バイトになり、そのまま通る
  - 二重の escape: `PrintableInboxText` は制御文字だけを書き換え、`\` は触らない。escape 済みの文は `inboxField` を通っても変わらない。`ControlCharactersInAnErrorAreEscaped` の最後の assertion が、cache の往復のあとの `Text()` でそれを固定している(`inboxField` は `PrintableInboxText` に空文字の `-` を足しただけで、`Text()` は空にならない)
  - 呼び出し: `mergeGuaranteeErrText` を呼ぶ非 test の行は 12 か所(218、261、296、308、353、362、466、471、474、575、578、794)。gh と git の error text を Detail に入れる箇所は、すべてこの関数を通る
  - 403 と 404 の分け方: `switch` は、(1)`mergeClassicNotStrict` の message は strict false、(2)`Not Found` は admin の文、(3)body の status が 403 か stderr に `(HTTP 403)` は「admin は原因の 1 つ」の文、(4)それ以外は一般の unknown、の順。(1)〜(3)は条件が重ならないので、順序を入れ替えて答えの変わる入力は作れない。`status` は `any` で、gh の本文は文字列 `"403"`、数値の 403 も `fmt.Sprint` で `403` になる。404 で stdout が空のときは (4) になり、fix の前と同じ
  - 新しい行が赤になる mutation(読みで追った): (a)403 の case を `Not Found` の case に戻す、と `strict answers 403:` の 3 行が赤。(b)403 の条件から `body.Status` の項を外す、と `in the body only` が赤(その行の stderr は `exit status 1` だけ)。(c)`(HTTP 403)` の項を外す、と `on stderr only` が赤(stdout が空)。(d)`Not Found` の判定を status 404 にする、と `strict answers a 404 with another message` が赤。この行は、test 報告の S03・S03b(`unknown` を `pr_checks_only` にする変更)も止める
  - 登録簿の 244 行目: `|` は 6 個で 5 列。(d)〜(f)に書いた関数名は HEAD にあり、`file:line` の pointer はない。(a)の「about 850 lines」は HEAD の 848 行と合う。(d)(e)(f)の中身も HEAD で確かめた(`readMergeOriginRepo` と `getMergeOriginURL` が get-url の 1 行目を取る、`Text()` の括弧と `subject()` が branch と repo を重ねて言う、gh を打つ 3 行が 4 か所)
  - cycle 1 の M1 と L1 は、fix の回でなく sync-docs の 240b0ce8 で直っている(下の表)
- 確かめていないもの: テスト、`gofmt`、`go vet`、`-race`。GitHub が strict の endpoint で 403 を返す実例(live の証跡は cli/cli の 404 だけ)。C1 の制御文字(`\u009b` など)を端末が実際にどう扱うか。scp 形の origin の文字の検査がないことは `mergeOriginFromURL` と `url.Parse` の挙動を読んで言っていて、走らせていない

### 所見(cycle 2)

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。cycle 1 の ID(M1、L1〜L6)と取り違えないよう、ID に `C2-` を付ける。登録簿の 244 行目が cycle 1 の `L2`・`L5`・`L6` を名指しているので、cycle 1 の節の ID と文は変えていない。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | security | fix が escape したのは gh と git の error text だけで、doctor が生で出す Detail には別の出どころの文字が残る。(a)`mergeGuaranteeResult` の `s.WriteErr`・`s.RemoveErr`・`s.Path`(cache の書き込みと削除の OS のエラーと state dir の path。実装者が範囲外とした件)。(b)origin の URL から来る `g.repo`(owner/repo)と `g.apiHost`。`mergeOriginFromURL` は `://` の形なら `url.Parse` を通る(ASCII の制御文字は断るが C1 は通す)が、scp 形(`git@host:owner/repo`)には文字の検査がなく、`subject()` など 4 か所の Detail に生で入る。(c)gh の JSON の `default_branch` と `full_name` は、git の ref 名が制御文字を持てず、full_name の文字も限られるので、届かないと読んだ。fix の commit message と `mergeGuaranteeErrText` の doc は「doctor prints a Detail as it is」を escape の理由にするが、安全になったのは error text の部分だけ。3 つとも出どころがローカル(自分の `.git/config`、checkout の path、OS のエラー)で、cycle 1 の L3 より届く範囲が狭く、doctor は他の検査も error を生で出す(`codex unusable: %v` など)ので LOW | `internal/cli/doctor_merge_guarantee.go:78`、`:80`、`:83-84`、`:86`、`:88`(`s.Path`、`s.WriteErr`、`s.RemoveErr`)。`internal/cli/doctor.go:207-208`(Detail を `fmt.Printf` で生のまま出す)。`internal/org/mergeorder.go:719-749`(scp 形は `strings.Cut` と `LastIndexByte` だけ)、`:623`(`mergeRepo.String()`)。`internal/org/mergeguarantee.go:218`、`:295-296`、`:311`、`:523`(`g.repo`・`g.apiHost` を Detail に入れる行)。doctor の test に ESC を持つ `WriteErr` や origin の行はない(`doctor_merge_guarantee_test.go:305-309` の `writeErr` は通常の文) | doctor の境界で 1 回だけ escape する: `mergeGuaranteeResult` の最後で `r.Detail = org.PrintableInboxText(r.Detail, false)`(escape 済みの文は変わらず、二重にならない)。`mergeGuaranteeErrText` の escape は cache に入る Detail のために残る。直さないなら、fix の doc の「doctor prints a Detail as it is」を「escape するのは gh と git の error text だけ」に言い直す |
| C2-2 | LOW | readability | `readMergeClassicStrict` の doc は、admin のない人への GitHub の答えを 2 か所で言う。1 文目は「admin のない人には 404 `Not Found`」、3 文目は「GitHub answers 403 without admin access and for other causes, such as a rate limit or an SSO authorization the token lacks」。両立はしうる(token の種類による)が、どの token でそうなるかを書いていないので、読む人には食い違って見える。403 の側は観測もしていない。live の証跡は cli/cli の 404 だけで、strict の endpoint が 403 を返す実例も条件も repo にない。Detail の文は「can come from」と控えめだが、doc は事実の形で書かれている | `internal/org/mergeguarantee.go:430-437`(doc)。`docs/evidence/merge-guarantee-livefire-2026-10-14.md:60-64`(cli/cli は 404 `Not Found`)。`docs/plans/active/2026-10-14-org-merge-guarantee.md:64`(404 だけ「cli/cli の trunk で確かめた」) | 3 文目を、観測した 404 と観測していない 403 を分ける形に直す(例: 「a 403 has not been seen from this endpoint; GitHub also answers 403 for a rate limit or an SSO authorization the token lacks, which is why the Detail does not claim admin access」)。1 文だけの修正 |
| C2-3 | LOW | doc drift | 403 の意味を変えた fix が、同じ主張を書いた 2 つの文書に届いていない。仕様 FR-7 の 7c-2 の注は「その strict を読めない(admin でない人への 404 `Not Found`、403)」と書き、403 も admin のない人の答えに読める(実装者が範囲外とした件)。計画の AC2 も「admin でなく strict を読めない(404 `Not Found` か 403)」と並べる。コードの Detail は、403 では admin のないことを原因の 1 つとしか言わない。README、AGENTS.md、skill の 4 面、help、`internal/` の非 test の Go に、403 を admin と結ぶ文はない | `docs/specs/2026-10-07-org-multi-org-director.md:74`、`docs/plans/active/2026-10-14-org-merge-guarantee.md:93`(AC2)。`403` を含む非 test の行を README.md、AGENTS.md、`.claude/skills`、`.agents/skills`、`templates/base`、`internal/cli`、`internal/org` で grep した結果は `mergeguarantee.go` の 4 行(433、434、467、469)だけ。plan の 30、33、121 行は 403 を admin と結んでいない | この回の /sync-docs で、仕様の 1 句を「admin のない人への 404 `Not Found`、または 403(admin のないことは原因の 1 つ)」の形にする。cycle 2 の pipeline はまだ /sync-docs を通るので登録簿の行は要らない。計画の AC2 は /pr で archive されるので直さない。sync-docs が直さなかったときだけ、登録簿の 244 行目に足す |
| C2-4 | LOW | maintainability | 登録簿 244 行目に 2 つ。(a)「Why deferred」の「the fix cycle after the self-review (cycle 2 of the pipeline, capped at 2) fixed only its L3 and L4」は cap の数字に頼る。cross-review の指摘で cap を上げて回し直す運用で、`docs/tech-debt/README.md` の 119 行目は同じ形の文が cap の引き上げで偽になった記録、org-merge-order の cycle 3 と 4 も cap を 2 から 3、4 に上げた。cycle 2 の cross-review が指摘を出して cycle 3 で (d)〜(f) を直せば、この文は偽になる。(b)「Related」の `docs/reports/test-2026-10-14-org-merge-guarantee.md (Test gaps 4 and 6)` は番号で指す。test の報告は cycle 2 の /test が書き直し、Gap 1 が閉じたので番号が動く見込みが高い。また (g) の 3 つめの `%2F` は Gap 8 の前半で、「4 and 6」に入っていない | `docs/tech-debt/README.md:244`(Why deferred の列の 4 文目と Related の列)、`:119`(cap の引き上げで偽になった前例)。`docs/reports/test-2026-10-14-org-merge-guarantee.md:109`、`:111`、`:113`(Test gaps の 4、6、8) | (a)cap の数字を外し、「(d)(e)(f) は最後の pipeline の回の fix に入れなかった」と事実だけにする。(b)番号をやめて、中身(「empty context」「os.Stat のエラー」「%2F」)で指す。登録簿は文書なので /sync-docs が直せる |

#### C2-1 の根拠

- doctor の出力は `fmt.Printf(" — %s", r.Detail)` で、`mergeGuaranteeResult` の `r.Detail` は `c.Kind + ": " + c.Detail + "; " + saved`。`saved` の 4 つの枝のうち、`s.Path` と `%v` の `s.WriteErr`・`s.RemoveErr` は escape を通らない。この Detail は cache に入らない(書けなかった、または書いたあとの文)ので、status と merge-order の `inboxField` も通らない
- scp 形の読みは `strings.Cut(s, ":")` と `LastIndexByte(host, '@')` だけで、host と owner/repo の文字を見ない。`mergeRepo.String()` は `owner + "/" + name` で、`subject()` の `origin's repository %s` に入る。`gh api` に渡す path は `url.PathEscape` を通るので、gh の引数と cmd 名の path 部分は `%1B` になる。生で残るのは `subject()`、`decide` の gh が PATH にない文(218 行)、`readRepository` の「check that gh can reach %s」(295-296 行)、`returned no default_branch`(311 行)の文面と、`cmd` に入る `--hostname <host>`
- 実害は、自分の `.git/config` の origin に ESC を入れた人が自分の端末に出す程度で、cycle 1 の L3(GitHub か GHES が返す message)より狭い。LOW にした。直す行は 1 行で、出どころごとに escape を足すより境界で 1 回にするほうが、次に Detail へ足す文も覆う

### cycle 1 の所見を HEAD で見直した結果

| ID | 状態 | 確かめた場所 |
| --- | --- | --- |
| M1 | 直った(sync-docs の 240b0ce8) | `docs/tech-debt/README.md` 226 行目。Trigger は「A report of a feature PR merged without a gate」だけで、Impact に doctor の warn と status・merge-order の表示が入った。`git log -S` で 240b0ce8 |
| L1 | 直った(240b0ce8) | `mergeguarantee.go:803-804` |
| L2 | 登録簿へ | 244 行目の (d)。get-url の固定の 10 秒が `MergeOrderParams` に出ないことも入った |
| L3 | 直った。範囲は error text だけ | `mergeguarantee.go:608-626`。残りは C2-1 |
| L4 | 直った | 403 の分岐(467-471)。404 `Not Found` は cycle 1 の推奨どおり 1 つめのまま。仕様の 1 句は C2-3、doc の文は C2-2 |
| L5 | 登録簿へ | (e) |
| L6 | 登録簿へ | (f) |
| test 報告の Gap 1 | 直った | `Branch not found` の 404 の行と、403 が本文だけにある行 |
| Gap 4、Gap 6 | 登録簿へ | (g)。pointer の直しは C2-4 |
| Gap 8 の後半(制御文字の test がない) | 直った | `TestMergeGuarantee_ControlCharactersInAnErrorAreEscaped` |

### 実装者が範囲外とした 2 件の重さ

- 仕様 FR-7 の 7c-2 の注の「403」: LOW(C2-3)。文書 1 句で、この回の /sync-docs が直せるので登録簿には載せない
- `doctor_merge_guarantee.go` の cache のエラーと `s.Path` を escape しない件: LOW(C2-1 の (a))。出どころは OS のエラーと自分の checkout の path で、(b) の origin の URL と一緒に、境界の escape 1 行で閉じる

### 所見にしなかったもの

- escape は `unicode.IsControl`(Cc)だけで、bidi の制御や幅のない空白などの書式文字(Cf)は残る。`PrintableInboxText` が受信箱の全員に使う規則と同じで、この fix が足した差ではない
- 不正な UTF-8: fix の前は生のバイトのまま Detail に入った。今は 1 文字ずつ読むので U+FFFD になり、cache の JSON と揃う。test はないが害はない
- `Not Found` と 403 の 2 つの `Sprintf` は前半の文が同じ。2 つなので helper にするほどではない
- test の行の名前 `strict answers 403: in the body` は、本文と stderr の両方が 403 を言う。`in the body only` と並ぶと名前だけでは違いが分かりにくいが、入力と期待は正しい
- `mergeguarantee_test.go` は 1,334 行だが、`internal/org/` には 5,552 行の `verbs_test.go` があり、test の長さはこの repo で珍しくない。登録簿の (a) は非 test の file だけを言う

### Tech debt identified

この回は報告だけをコミットする指示なので、`docs/tech-debt/README.md` には行を足していない。cycle 2 は上限の最後で、直すと 3 回目の pipeline が要る。次の 2 つを 244 行目の (h)(i) として足し、C2-4 の 2 点で行の文を直すことを勧める(足すのと直すのは /sync-docs が行う)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (h) `ralph doctor` prints the Detail of its `Merge guarantee` check as it is, and only gh's and git's error text in it is escaped (`mergeGuaranteeErrText`). Raw text can still come from the cache write and remove errors and the state dir path (`mergeGuaranteeResult`), from origin's host and owner/repo (`mergeOriginFromURL` checks no character of the scp-like form), and, probably not reachably, from gh's `default_branch` and `full_name`. (i) the doc of `readMergeClassicStrict` says GitHub answers 403 without admin access, which no check observed (the live check saw a 404 `Not Found` for a token without admin access), next to the sentence that says such a token gets 404 | (h) an ESC in a local path, a remote URL or an OS error reaches the terminal raw in doctor's line, while status and merge-order escape the same text. (i) a reader cannot tell which of the two answers a given token gets | (h) the sources are local (the user's own `.git/config`, checkout path, OS errors) and doctor prints other errors raw; one `PrintableInboxText` at the end of `mergeGuaranteeResult` closes them all. (i) a one-sentence comment fix, left because it was the last pipeline run | The next edit to `mergeGuaranteeResult` or `readMergeClassicStrict`, or a report of a control character in doctor's line | `docs/reports/self-review-2026-10-14-org-merge-guarantee.md` (Cycle 2, C2-1, C2-2); `internal/cli/doctor_merge_guarantee.go` (`mergeGuaranteeResult`); `internal/org/mergeguarantee.go` (`mergeGuaranteeErrText`, `readMergeClassicStrict`); `internal/org/mergeorder.go` (`mergeOriginFromURL`) |

C2-3 は /sync-docs が仕様の 1 句を直せば行は要らない。C2-4 の(a)は、244 行目の Why deferred の列から cap の数字を外す直し、(b)は Related の列の「Test gaps 4 and 6」を中身の言い方に変える直しで、どちらも債務ではなく行の文の直し。

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。cycle 1 の L3・L4 と test の Gap 1 は閉じ、fix は新しい欠陥を入れていない。C2-1〜C2-4 は LOW で、最後の cycle なので直さない)
- Follow-ups:
  1. /sync-docs(この回): 仕様の 1 句(C2-3)、登録簿 244 行目の 2 点の直し(C2-4)、(h)(i) の追記(C2-1、C2-2)
  2. 3 回目を回す理由が別にあるなら、C2-1(`mergeGuaranteeResult` の 1 行と、ESC を持つ `WriteErr` の test 1 本)と C2-2(doc の 1 文)を先に直す
  3. cycle 2 の /verify と /test が書き直す報告に、cycle 1 の時点の行番号や Gap の番号を指す pointer が残っていないか見る。登録簿の 244 行目の Related は番号で指している(C2-4)
- Known gaps: GitHub が `protection/required_status_checks` で 403 を返す実例は repo にない(C2-2)。C1 の制御文字を端末がどう出すかは未確認。`gofmt`・`go vet`・テストは /verify と /test で確かめる
