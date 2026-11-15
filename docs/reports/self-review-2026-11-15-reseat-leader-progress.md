# Self-review report: reseat-leader-progress

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)
- Branch: `feat/reseat-leader-progress`、base は `origin/main` の `df186374`
- Reviewer: reviewer サブエージェント(`/self-review`)。**Cycle: 1**(`cycle-count.json` は `cycle: 1`)。この報告書は 1 回目の run(HEAD `8ac1487c`)と、その指摘を直した `71ddcb30` のあとの 2 回目の run(HEAD `d4d1ca93`)を持つ
- **Merge: yes**(2 回目の run: CRITICAL と HIGH はなし。MEDIUM 1 件、LOW 2 件。1 回目の M1 は 2 回目で形を変えて残った。下の N1)

## Run 2(cycle 1、2 回目): 直しのあと

Scope: `71ddcb30` の差分(`leader.md`、`reseat_progress.go`、テスト 3 ファイル)と `d4d1ca93`(計画の記録)を全部読んだ。ブランチ全体(`git diff origin/main...HEAD`、`git fetch origin` のあと)は 1 回目に読んでいない `feature.go`・`reseat.go`・仕様の差分と、`leader.md` の手順 3、5、8〜15 と「agmsg の受信箱の運用」の節を読み直した。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれ(`/verify`、`/test`、`/sync-docs` の担当)

### Findings

この run の指摘は N1 から振る(1 回目の M1、L1〜L7 と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | MEDIUM | maintainability | 「座席の確認」の「待つ」に出口がない。項目の前提は「`(active)` の座席で、今回の立て直しの行にないものは、前の leader の TASK をまだ進めている」だが、機能の org では成り立たない。implementer と reviewer は手順 13 まで止めない(手順 12 の頭)ので、TASK と TASK のあいだも `(active)` のまま入力待ちで残る。手順 3 の reviewer は、承認の前に TASK なしで立ちうる。leader だけを立て直す場面(別のアカウントへの移行、利用上限)で、前の leader が RESULT を既読にしたあとや、手順 9〜11 の自分の作業中に止まっていると、新しい leader は届かない RESULT を待つ。節は RESULT を載せない。項目は座席の画面も agmsg の履歴も指さず、時間の上限もない(座席の項目と gate の項目は 10 分で BLOCKED)。per-plan で承認待ちのあいだに reviewer が `(active)` のとき、手順 5 の implementer の spawn まで止まる読み方もできる(項目は「その座席の RESULT か BLOCKED が来るまで待つ」と書き、reviewer に TASK はない)。1 回目の M1 の推奨は「`ralph org read` で読んで待つ」だったが、直しは読む部分を落とした。テストは待つ文を固定している | `leader.md:161-172`(項目)、`:568-573`(手順 12: 座席を手順 13 まで止めない)、`:313-314`(手順 3: 承認の前の reviewer)、`:1352-1353`(`history.sh` は既読の印を変えず、直近 50 件に送った TASK も入る)、`internal/cli/org.go:1976-1980`(`ralph org status --live` は active な座席ごとに herdr の状態と画面を出す)、`prompts_leader_reseat_test.go:269` | 項目を 2 段にする。(1) `(active)` の各座席を `ralph org status --org-id {{ORG_ID}} --state-dir <台帳> --live` の HERDR の欄(または `ralph org read`)で見る。作業中なら今の文のとおり待つ。(2) 入力待ちなら、`bash {{AGMSG_SCRIPTS}}/history.sh {{TEAM}} leader 50` で、その座席宛てに送った最後の TASK のあとに、その座席の RESULT か BLOCKED があるかを見る。あれば今届いたものとして手順 8 以降の裁定から扱い、なければ TASK を持たない座席として、頼むスライスを「計画の残りと branch の commit の行」から決めて TASK を送ってよい。待つ文には座席の項目と同じ 10 分の上限と BLOCKED を付ける。テストは 2 段それぞれの文を固定する |
| N2 | LOW | naming | 「座席の確認」が使う語の範囲が、文字どおりだと広い。(a) 述語「`(active)` の座席で、「今回の立て直し」の行にないもの」は、立て直された leader 自身の座席(`leader`、`(active)`)を含む。節の「今回の立て直し」の行は leader のあとに起動する座席だけを書く。(b) 「手順 5 でその役割をもう一度 spawn せず」は手順 5 だけを挙げる。手順 3 も reviewer を spawn するので(承認の前に立ててよい)、計画の項目が per-plan で手順 2・3 へ送る場合に、`(active)` の reviewer をもう一度立てうる。(c) 計画の項目の括弧「(gate の項目と座席の項目が、そのあとの進み方を決めます)」は、いつも行う「座席の確認」を挙げない。(d) gate の項目の「同じ BLOCKED」は、あとの座席の項目にだけある手順(本文に座席の名前、TASK_ID は org_id、`ralph org escalate`)を指す前方参照になった | `leader.md:163-165`、`:165-168`、`:180-181`、`:195-196`、`reseat_progress.go:269-277`(座席は `ReseatSeats` だけ。leader は入らない)、`leader.md:313-319`(手順 3 の spawn) | 述語は「leader 以外の `(active)` の座席で」にする。spawn の文は「手順 3 と 5」にする。計画の項目の括弧に座席の確認を足す。gate の項目は BLOCKED の書き方を座席の項目へのパスで示す(「座席の項目の BLOCKED」)。N1 を直すときに同じ段落で直せる |
| N3 | LOW | readability | `71ddcb30` が文を足したコメントの折り返しが崩れている。`readReseatProgressCommits` の doc は 1 行が 125 字(ほかの行は 80 字前後)。`reseatProgressSection` の doc は「and the block is」で 1 行を切り、次の行が「fenced against the rest.」だけになっている | `reseat_progress.go:371`、`:198-199` | 2 つの段落を折り返し直す。gofmt は直さない |

### 1 回目の指摘の確認(HEAD `d4d1ca93`)

| ID | 1 回目の内容 | 状態 | 根拠 |
| --- | --- | --- | --- |
| M1 | leader だけ、または implementer を動かしたままの立て直しが規則のどの項目にも当たらず、TASK を重ねうる | 二重の TASK と二重の spawn は止まった。待つ側に N1 が出た | `leader.md:161-172` が先頭の項目で、いつも行う。`leader.md:156-159` の導入も合う |
| L1 | `git log` に `--no-show-signature` がない | 直った | `reseat_progress.go:398`。偽の gpg のテストは、先に素の `git log` が署名の行を出すことを確かめるので、旗がなくても通る形にならない。`isolateGitEnv` で利用者の gpg の設定は入らない(`progressStartRepo`) |
| L2 | 「最初の 20 件」が新しい順の先頭を指す、受信箱の「ほか N 件」が新しい件を落とす | 直った | 見出しは `新しい順に N 件`、`古い順に N 件`、`ほか N 件(新しいもの)`。受信箱の並びは ID の数値の昇順(`inbox.go:730`)なので、「古い順」は合う |
| L3 | digest の後ろに `(` が付く。gate の行も同じ | 承認の行は直った。gate の行は残した | 承認の行は `sha256:<digest> (承認 <時刻>)`(`reseat_progress.go:213`)で、テストが空白で切った語を見る。gate の行の `branch <名前>(<時刻>)` と `via <名前>(<時刻>)` は括弧が付いたまま(`:220-226`)。規則が比べるのは `head` と `base` で、どちらも空白で区切れる |
| L4 | Go のコメントが `docs/plans/active/...` を指す | 直った | 3 ファイルとも `docs/plans/archive/...`。`/pr` が archive へ移すまでは、指す先は active にある。仕様の文と同じ形になった |
| L5 | `reseatProgressSection` の長さ、`inboxSummary` との二重、`NewReplacer` を毎回作る | 残した(計画の記録では tech-debt へ) | register に行はまだない(下の引き継ぎ) |
| L6 | 計画の項目の最初の分岐が、すでに 2 行が正しいときの行き先を言わない | 直った | `leader.md:179-181`「どちらの場合も手順 5 から進めます」。括弧の中身は N2(c) |
| L7 | gate の項目が、まだ立っていない座席を待たない | 直った | `leader.md:193-197`。待つ先は座席の項目と同じ `(active)` で、10 分の上限も同じ |

### Positive notes

- 1 回目の安全の確認(行頭のラベル、`PrintableInboxText` と U+2028・U+2029、`fenceFor` の囲み、10 秒の context、空の節はタスクに触れない)は、`71ddcb30` の変更の範囲では変わっていない。
- 偽の gpg のテストは、署名の行が出ることを前提として確かめてから本体を見る。`-S` の署名は `gpg.program` の偽で作るので、利用者の鍵に依らない。
- 受信箱の「古い順」の主張は、並びの実装(数値の昇順)で裏が取れる。文字列の昇順なら `e10` が `e2` より前に来るが、そうなっていない。
- 「座席の確認」を先頭の項目にし、「いつも行う」と導入で言ったので、1 回目の M1 の「どの項目にも当たらず手順 5〜8 に落ちる」形は消えた。テストは項目の順(5 項目の並び)と、座席の項目と同じ `ralph org status` を使うことを固定している。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。`feature.go`(+27)と `reseat.go`(+16)の差分は 1 回目のとおり小さい。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 1 回目の L5(3 点)と、L3 の残り(gate の行の `branch <名前>(<時刻>)` と `via <名前>(<時刻>)`) | LOW。挙動は変わらない。gate の行の括弧は、規則が比べない値にだけ付いている | 計画の記録(Progress checklist の `71ddcb30` の行)に「tech-debt へ」とある | 次に `reseat_progress.go` か `leader.md` の規則を触るとき | この報告書、計画の Progress checklist |

_(register の追記は `/sync-docs` の担当。`git diff origin/main...HEAD -- docs/tech-debt/README.md` は空で、今は計画の記録にしか残っていない。)_

## Recommendation

- Merge: yes(CRITICAL と HIGH はなし。MEDIUM 1 件、LOW 2 件)
- Follow-ups:
  - N1 は `leader.md` の 1 項目と、そのテストの直しで済む。`/verify` の前に直すほうが安い(直せば `/self-review` から回し直す)。直さず進めるなら、`/pr` の既知の穴に「leader だけの立て直しで、座席が入力待ちのまま `(active)` のとき、新しい leader が RESULT を待ち続けうる」と書く。N2 は N1 と同じ段落なので、一緒に直せる。
  - N3 は 2 つのコメントの折り返しだけ。
  - **`/sync-docs` への引き継ぎ**(指摘ではない):
    - 1 回目の引き継ぎはそのまま残る。tech-debt の行 514 の trigger(`reseat.go` と `feature.go` の長さ)に今回の数を足す。S3 の implementer が範囲外として挙げた 3 点(`gh pr list --state all` が merge されず閉じた PR も「ある」と読むこと、PR を作ったあと RESULT の前に止まった leader は手順 15 に進み RESULT が上がらないこと、承認なしのコミット済みの計画から手順 2 に進むとき計画助言が済んでいたか分からないこと)の行も register にない。
    - 上の Tech debt の行(L5 と gate の行の括弧)を register に足す。
    - 計画の D4 と AC4 は「最初の 20 件」と書き、出荷した見出しは「新しい順に 20 件」。計画の記録は「`git log` の先頭 20 行のことで実装と合う」としている。archive に移る計画と見出しの文が違うので、`/verify` が読むときの注として残す。

## Run 1(cycle 1、1 回目): 履歴

HEAD `8ac1487c`。Merge: yes。CRITICAL 0、HIGH 0、MEDIUM 1、LOW 7。報告書の全文は `64c7f828` にある。指摘は M1(leader だけ、または implementer を動かしたままの立て直しが規則のどの項目にも当たらない)、L1(`git log` に `--no-show-signature` がない)、L2(「最初の N 件」の向き)、L3(digest の後ろの括弧)、L4(Go のコメントが `docs/plans/active/` を指す)、L5(関数の長さ、`inboxSummary` との二重、`NewReplacer`)、L6(計画の項目の行き先)、L7(gate の項目が座席を待たない)。直しは `71ddcb30`、2 回目の run の表のとおり。1 回目の `/sync-docs` への引き継ぎは上の Recommendation に移した。
