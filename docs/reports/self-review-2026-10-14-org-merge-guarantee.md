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
