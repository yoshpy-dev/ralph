# Self-review report: reseat-leader-progress

- Date: 2026-11-16(run 7: cycle-count 4、cap 5、cycle 4 の cross-review の ACTION_REQUIRED 1 件の直し `3a5922e3` のあと)。cycle 4 の run と run 5 と cycle 3 の run と cycle 2 の run は同じ日、cycle 1 は 2026-11-15
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)
- Branch: `feat/reseat-leader-progress`、base は `origin/main` の `df186374`(main は `48a8a661` まで進み、branch は 3 回マージ済み)
- Reviewer: reviewer サブエージェント(`/self-review`)。**Cycle: 4**(`cycle-count.json` は `cycle: 4`。run 7 の時点で `cap: 5` で、cycle 4 の cross-review の ACTION_REQUIRED を受けて cap を 4 から 5 に上げ、cycle は 4 のまま。cycle 4 の run までは `cap: 4` で、run 5 の cross-review の ACTION_REQUIRED を受けて cycle を 3 から 4 にした)。新しい run を先頭に置き、前の run は履歴にした。判定の行は、各節の見出しの下に、その節の run のものを置く
## Run 7 (cycle-count 4, cap 5): HEAD `4487d948`

- **Merge: yes**(run 7: CRITICAL、HIGH、MEDIUM はなし。LOW 2 件。cycle 4 の cross-review の finding(文が入力欄に残った座席に TASK を重ねて送る)は、座席の確認の項が入力欄を先に見て、何か残っていれば何も送らず BLOCKED を上げる形で閉じ、直しは通知を消す向きの欠陥を持ち込んでいない。LOW は、空の入力欄が出す薄い案内文を「残り」に数えない 1 句がないことと、新しいテストの最後の 2 つの確認が名乗るほど固定しないこと)

Scope: `git diff 69257dc5..HEAD` の 5 ファイル。コードと prompt に当たるのは `3a5922e3`(`internal/org/prompts/leader.md` の +10 −1、`internal/org/prompts_leader_reseat_test.go` の +44 −1)で、差分を全部読んだ。`48e31ba5`(cycle 4 の triage)と `4487d948`(計画の記録)と insight event は記録なので、triage の finding の写しと直しの説明が `3a5922e3` の中身と合うかだけ見た。周辺で読んだもの: `Send` の進み具合ごとの枝(`verbs.go:600-700`)、send の画面の確かめ(`send_screen.go:20-120`、`:160-200`、`:551-570`、`:691`)、resume の画面の規則(`resume_screens.go:60-80`、`:276-290`)、typed protocol の節の段落(`leader.md:1323-1336`)、director の prompt の同じ状況の 1 項(`director.md:39-44`)、既存の段落のテスト(`prompts_send_unsubmitted_test.go:36-67`)、`markdownSection` と `containsPhrase` と `assertPhraseOrder`。テストは回していない。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれ(`/verify`、`/test`、`/sync-docs` の担当)。再提起しなかったもの: 登録済みの項目(行 519〜526)と、前の run が出した LOW。

### 依頼された 4 点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| 新しい文は、項の残りと順に読めるか(旧文の頭の「入力待ちなら、」が「入力欄が空なら、」になった) | 読める | 「入力待ちなら、まず、…残っていないかを見ます」(`leader.md:182-185`)→「残っていれば、前の send が…1 つのメッセージになります」(`:185-187`)→「その座席には何も送らず、キーも押さず、…待ちます」(`:187-190`)→「入力欄が空なら、「座席ごとの最後の TASK」のその座席の行から…」(`:191`)→「その座席の行がなければ」。「入力欄が空なら」は文法上は 1 文目の条件だが、そのあとの送る文(QUESTION の確かめ、TASK の送り直し)は、残っていた枝が「待ちます」で閉じたあとに続く。順に読めば、残った文のあとに送る道はない。`assertPhraseOrder` の 5 つが順を固定する |
| BLOCKED の TASK_ID(org_id)は typed protocol の段落と食い違わないか | 食い違わない | 段落は「TASK_ID は、その send が送った TASK の TASK_ID です(TASK 以外の送信は org_id)」(`leader.md:1329-1330`)。立て直された leader は、前の leader の send が送った TASK の TASK_ID を知らない(`sent` が台帳に残らず、「座席ごとの最後の TASK」の行にも出ない。`verbs.go:630-650` の `SendProgressTextTyped` は `sent` を足さずに返す)ので、org_id が使える唯一の値。同じ項の 10 分の BLOCKED も org_id で(`:179-181`)、`agent-messaging.md` の「BLOCKED は org 全体の件なら org_id」とも合う。項は括弧で TASK_ID を言い切り、段落は「のとおり」で参照するだけなので、読み違える余地は小さい |
| 貼り付けの印は send の画面の確かめが知るものと同じか | 同じ | send は、空白を落とした入力欄の text に `\[Pastedtext#\d+\+(\d+)lines\]`(claude、`send_screen.go:183`)と `\[PastedContent(\d+)chars\]`(codex、`:194`)を当てる。項の `[Pasted text #N +M lines]` と `[Pasted Content <N> chars]` は、同じ印の数を文字にした書き方で、`send_screen.go:57`、`:71` のコメントの例(`[Pasted text #1 +17 lines]`、`[Pasted Content 2016 chars]`)と同じ形。反対側の「空の入力欄が出す案内文」は項にない(R7-1) |
| 新しいテストは名乗ったものを固定するか | 項の文言は固定する。最後の 2 つの確認は名乗るほどではない | `assertPhraseOrder` の 5 句は、2 句目から 4 句目が文の全文で、「何も送らず、キーも押さず」の削除、BLOCKED の TASK_ID の変更、「入力欄が空なら」の削除で落ちる(読んで確かめた)。段落への参照と 2 つの印の確認は R7-2 |

### Findings

この run の指摘は R7-1 から振る(C4-n、C5-n、C3-n、C2-n、R3-n と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R7-1 | LOW | prompt | 座席の確認の項は、入力待ちの座席の入力欄に「文字か貼り付けの印」が残っていないかを見ると言い、空の入力欄が出す薄い案内文を除かない。codex の空の composer は `› Ask Codex to do anything`(作業中も出る)を、claude の新しい session の入力欄は `Try "…"` を出し、plain text の読み取りでは入力した文字と見分けがつかない。ralph のコードは、この 2 つを空に数える。この項が対象にするのは、「座席の行がない」座席(新しく立った reviewer など)と、TASK のあいだで入力待ちの座席で、案内文が出る状態そのもの。leader が案内文を文字と読むと、空の座席に TASK を送らず BLOCKED を上げる。次の「入力欄が空なら」も「空」としか言わず、案内文を空に数える文はどこにもない。向きは表に出る側(BLOCKED が人に届く。座席は止まるが壊れない)で、LLM の leader が案内文と気づく見込みもあるので LOW。ただし、この直しの項が健全な道で動きうる唯一の指摘 | `leader.md:182-185`(「文字か貼り付けの印…が残っていないかを見ます」)、`:191`(「入力欄が空なら」)。`send_screen.go:34-36`(「Placeholders count as empty: codex's composer placeholders (codexComposerPlaceholders, shown while working too) and claude's `Try "…"` of a fresh session」)、`:169-171`(`^Try "[^"]+"$`)、`:691`。`resume_screens.go:276-284`(`codexComposerPlaceholders` の `Ask Codex to do anything`、`Ask a follow-up question`、doc に「Plain text cannot tell them from typed text」)。`Ask Codex` と `Try "` を `internal/org/prompts/*.md` で grep して、どちらも 0 件。`[org].driver_pool` に claude がなければ全座席が codex(`internal/config/role_models.go:72-86`) | 「…残っていないかを見ます」の直後に、「入力欄が空のときに出る薄い案内文(codex の `Ask Codex to do anything`、claude の新しい session の `Try "…"`)は、残ったものに数えません」の 1 句。テストの `assertPhraseOrder` の 2 句目の後ろにその句を足す |
| R7-2 | LOW | test | `TestRenderRolePrompt_Leader_ReseatRuleSeatCheckSendsNothingOverLeftoverInput` の最後の 2 つの確認が、名乗るほどのものを固定しない。(a) `// The placeholders are the ones send's screen check knows.` のループは、`"[Pasted text #"` と `"[Pasted Content "` の文字列の写しを、生の `strings.Contains(item, …)` で見る。send の `sendClaudePastePlaceholder` と `sendCodexPastePlaceholder` とは結ばれておらず、send の側の印が変わってもテストは緑のまま。2 つの印は、2 句目が全文で既に固定している(重複)。このファイルの他の確認(折り返しを無視する `containsPhrase`、`squashSpace`)と違って生の比較なので、template を `[Pasted Content` の後ろの空白で折り返し直すと、語が同じでも落ちる。(b) typed protocol の節を見る 3 句は、段落の中の同じ 3 句を `TestRenderRolePrompt_Leader_SendTextTypedNotSubmittedEscalates` が、段落 1 つに限って固定している(こちらは節全体なので弱い複製) | `prompts_leader_reseat_test.go:522-528`(a)、`:508-520`(b)。`prompts_send_unsubmitted_test.go:36-67`(段落が 1 つ、3 句と「画面をそのまま貼らないでください」「TASK_ID は、その send が送った TASK の TASK_ID です(TASK 以外の送信は org_id の `org-a`)」まで)。`send_screen.go:183`、`:194`。「文字列の写しで producer との足並みを名乗る」形は、この branch の `TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites`(`reseat_progress_test.go:731`、cycle 2 の LOW)と同じ。向きは、send が印を変えたとき prompt の側が気づかれずに古くなる(漏れる側)で、そのときは send の画面の確かめのテストも更新されるので LOW | (a) ループを、項の 2 つの印の数を例にして(`#N` を `#1`、`+M` を `+17`、`<N>` を `2016`)空白を落とし、`sendClaudePastePlaceholder.MatchString` と `sendCodexPastePlaceholder.MatchString` で照合する形にする。簡単に済ますなら、ループとコメントを消す(2 句目が語は固定している)。(b) 「節が段落を持つ」ことは既存のテストが見るので、ループを消してよい。直すなら R7-1 の 1 句と同じ commit で |

### 読んだが指摘にしないもの

- 「前の send が文を打ったまま Enter を押さずに終わっていて」は、原因を言い切る。入力欄に残るのは、人が打った文や、`SendProgressTextUnacknowledged`(文が着いたか分からない send)でも起きる。取る手(何も送らず、キーも押さず、BLOCKED)は原因によらず同じなので指摘にしない。
- `SendProgressEnterUnacknowledged`(Enter が着いたか分からない send)は `sent` を残さず、箱は空になる(送られている)。「座席の行がない」座席として TASK がもう 1 本送られる向き。文が入力欄に残る場合の外で、この直しの前からの形。
- 「返事が来るまでその座席を待ちます」のあと、返事を受けてどう続けるか(画面を読み直して「入力欄が空なら」へ戻る、など)は書いていない。隣の 10 分の BLOCKED(`:179-181`)と typed protocol の段落も同じで、返事の DECISION の本文が決める。
- 座席の確認の項は 89 行(`:162-250`)で、この直しで 10 行増えた。長さは前から続く形で、この直しの指摘にはしない。
- triage の「残っていれば、BLOCKED を上げる」を、director の prompt(`director.md:39-44`)の「leader への send が `(text typed but not submitted)` で終わったとき」と並べて読んだ。向きと止め方(何も送らない、キーも押さない、人に上げる)は同じで、director は DECISION の全文が箱にあれば Enter を押す例外を持つが、leader 側にその例外はなく、段落(`leader.md:1324-1336`)どおり押さない。食い違いなし。

### Positive notes

- 直しは「送らない、キーを押さない、BLOCKED を上げる」を 1 文で言い、段落への参照が、節の名前(「typed protocol」)と括弧つきの文言(`(text typed but not submitted)`)の 2 つで引ける。その文言を持つ段落は 1 つだけで、既存のテストが数を固定する(`prompts_send_unsubmitted_test.go:43`)。
- 新しい確認は、項が先に読んでいる画面を使い、追加の `ralph org read` を足さない。
- 旧文の頭を「入力欄が空なら、」に替えただけで、旧文の残りは変えていない。後ろの枝(`不明`、QUESTION の確かめ、TASK の種類ごとの決め方)は、文面も順も変わらない。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

### `/sync-docs` への引き継ぎ(指摘ではない)

- 登録簿(`docs/tech-debt/README.md`)の行 365・426・514・524 が引く `prompts_leader_reseat_test.go` の 706 行は、HEAD で 749 行になった(`wc -l`)。`leader.md` は 1,572 行から 1,582 行。800 行の目安まで、テストは 51 行。
- `docs/specs/2026-11-12-org-reseat.md` に、座席の確認が入力欄の残りを見ると言う文はなく(「入力待ち」「leftover」で grep して 0 件)、仕様の側で足す文はない。cross-review の triage に「fixed in `3a5922e3`」の記録がないのは、前の cycle と同じく次の triage で足す形。

## Tech debt identified

R7-1 と R7-2 を直さず進めるなら、次の 1 行を register に足す(`/sync-docs` か `/pr` の担当)。直す場合は `leader.md` と test を触るので、`/self-review` から回し直す(cap 5、cycle 4 なので、あと 1 回の run が残る)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R7-1(座席の確認の入力欄の確認が、空の入力欄の薄い案内文を「残り」から除いていない)、R7-2(新しいテストの印の確認が send の正規表現と結ばれておらず、2 句目と重複する) | LOW。R7-1 は、案内文を文字と読んだ leader が空の座席に BLOCKED を上げる形で、人に届き、座席は壊れない。R7-2 は send が印を変えたときの足並みだけ | cap 5 の最後の run になりうる回で、LOW 2 件のために prompt と test を直すと pipeline が `/self-review` から回り直す | 次に `leader.md` の座席の確認の項か、`send_screen.go` の印を触るとき | この報告書 |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 2 件)
- Follow-ups:
  - cycle 4 の cross-review の finding は、prompt とテストの両方で閉じている。新しい欠陥は通知を消す向きにも、黙って送る向きにもない。
  - 直すなら R7-1(prompt に 1 句、テストの句に 1 つ)。R7-2 は同じ commit で済む。挙動が変わるのは R7-1 だけで、R7-2 はテストの質。cap 5 の最後の run になりうるので、直す場合は `/self-review` から回し直す。リードの判断。
  - `/sync-docs` への引き継ぎ: 上の節。

## Cycle 4 (cap 4、最後の回): HEAD `db0b806c`(履歴)

- **Merge: yes**(cycle 4: CRITICAL、HIGH、MEDIUM はなし。LOW 4 件。run 5 の cross-review の 2 件は `4096ae88` で閉じ、直しは通知を消す向きの欠陥を持ち込んでいない。LOW は、git に渡す改訂の並びに `--` がないこと、古くなった `reseatProgressSupplied` の doc、doc コメントの折り返しの残りと呼び名、gate への QUESTION の送り直しに「上限」を言う 1 句の 4 つ)

Scope: `git diff 6bebbd66..HEAD` の 7 ファイル。コードと prompt に当たるのは `4096ae88`(`internal/org/reseat_progress.go`、`internal/org/prompts/leader.md`、`internal/org/prompts_leader_reseat_test.go`、`internal/org/reseat_progress_start_test.go`)で、差分を全部読んだ。`5528e51c`(triage)と `db0b806c`(計画の記録)は記録なので、2 件の finding の写しと直しの説明が `4096ae88` の中身と合うかだけ見た。周辺で読んだもの: `resolveGateBase`、`gateRefExists`、`gateGit`、`gateGitOutput`(`gate.go:624-760`)、`scripts/ralph-worktree.sh` の ensure(`git worktree add … -b <branch> "$base"`、`base` は `default_branch` が出す名前で、手元のブランチ)、`leader.md` の座席の確認と計画の 2 項目、手順 12 の gate の項(`:706-712`、`:758-762`)、「cross-review」の小節の送り直しの規則(`:1144-1160`、`:1264-1270`)。確認に使った probe は、scratchpad に作った使い捨ての repo の `git` だけで、テストは回していない。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれ(`/verify`、`/test`、`/sync-docs` の担当)。再提起しなかったもの: 登録済みの項目(行 519〜526)。

### 依頼された 5 点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| Go のエラー処理と slice の組み立て | 問題なし | `span`(`reseat_progress.go:573`)に `append` するのはループの中だけ(`:581`)。`log` と `rev-list` の引数は毎回 `append([]string{…}, span...)`(`:586`、`:591`)で新しい slice に写してから作るので、2 つの呼び出しが同じ裏の配列を共有しない。`gateRefExists` の失敗はその場で返す(`:578-580`)。時間切れは `readReseatProgressCommits` が前と同じ形で読む。Atoi の失敗文は `strings.Join(span, " ")`(`:597`)で、実際に打った並びと同じ |
| `--not` の範囲は正しいか | 3 つの形で正しい | probe(`HEAD --not refs/remotes/origin/main refs/heads/main` の `rev-list --count`)。(A) 手元の main が origin より後ろ、branch は手元の main から: 2(旧形も 2)。(B) regate が origin の main を branch にマージしたあと: 3(2 commit とマージ commit、origin 側は数えない)。(C) 手元の main が origin より 1 つ進み、そこから branch: 1(旧形は 2)。手元の ref がないときは足さない(`gateRefExists` が false)。origin の ref がないときは `base.ref` が手元の ref で、ループは origin の ref を調べて false を得るだけで、結果は同じ(git の呼び出しが 1 回増える) |
| `readReseatProgressCommits` の doc | 古い文と折り返しの残りがある | C4-2、C4-3 |
| `leader.md` の文は順に読めるか | 読める。1 点だけ言い過ぎ | 座席の確認の項は、QUESTION が見つかったあと「reviewer の『打たなかった』『誰も呼ばなかった』QUESTION なら送り直し、それ以外は DECISION」を 1 つの流れで言い、「QUESTION もなければ」の前で閉じる(`:220-228`)。`assertPhraseOrder` の並びと同じ。計画の項は、範囲の説明、手元の commit を除く理由、「branch の commit」の行も同じ、の順(`:250-256`)。言い過ぎは C4-4 |
| fixture は主張を証明しているか | 証明している | `progressStartRepoAheadOfOrigin` は origin の main に 1 commit、手元の main にもう 1 commit を置き、手元の main から branch を作り、`origin/main..main` が 1 であることを先に `Fatalf` で確かめる(`reseat_progress_start_test.go:69-71`)。旧形(`origin/main..HEAD`)なら、2 commit の branch は total 3、commit のない branch は total 1 になり、2 つの assertion とも落ちる(読んで確かめた。計画の記録は旧 `reseat_progress.go` を `-overlay` で当てて落ちたと書く)。`symbolic-ref` の行は、なくても `resolveGateBase` の候補の `main` が同じ ref を拾うので結果は変わらず、害もない |

### Findings

この run の指摘は C4-1 から振る(C5-n、C3-n、C2-n、R3-n と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C4-1 | LOW | robustness | `reseatProgressCommitsIn` が `git log` と `git rev-list` に渡す改訂の並び(`HEAD --not <ref> <ref>`)に `--` がない。worktree の根に `HEAD` という名前のファイルがあると、git は `ambiguous argument 'HEAD': both revision and filename` で終了コード 128 になり、節の「branch の commit」の行が `読めなかった(…)` になる。旧形(`<ref>..HEAD`)は範囲なのでファイル名と取り違えず、同じ repo で通った。向きは表に出る側(節は残り、commit の行だけ読めなかったと出る)で、根に `HEAD` のファイルがある repo は稀 | probe: 根に空の `HEAD` ファイルを commit した repo で、`git log --no-show-signature --format='%h %s' HEAD --not refs/remotes/origin/main refs/heads/main` は rc 128、末尾に `--` を足すと rc 0 で 2 件、旧形 `refs/remotes/origin/main..HEAD` は rc 0 で 2 件。`reseat_progress.go:573`(`span`)、`:586`、`:591`。`leader.md:252` の計画の項の `git log` は `-- docs/plans/active/` で終わっている | `span` の末尾に `"--"` を足す。rev-list の失敗文(`:597`)も実際の並びのままになる |
| C4-2 | LOW | comment | `reseatProgressSupplied` の doc は「Commits are the first lines of `git log --format='%h %s'` from the gate's base to HEAD」のまま。`4096ae88` で commit は、origin の既定のブランチにも手元の既定のブランチにもないものになり、「gate の base から HEAD まで」とは別の範囲になった。同じファイルの `readReseatProgressCommits` の doc は直したので、2 つの doc が違う範囲を言う。run 5 の cross-review が finding にした事実(手元の main が先にいる)を、この doc だけが言わない | `reseat_progress.go:116-118`。同じ語の文は他にない(`grep "gate's base"` と `grep "base to HEAD"` で 1 か所) | 「the commits readReseatProgressCommits reads: the ones neither origin's nor the local default branch has」の形に直す |
| C4-3 | LOW | comment | `readReseatProgressCommits` の doc の折り返しの残りと呼び名。(a) 文を足したあと、`// to. A git that`(17 桁)の短い行が段落の中に残り、前後は 71〜78 桁。(b) `(the cycle 3 run 5 cross-review)` は 2 つの数え方(cycle-count 3 と run 5)を混ぜる。同じ cross-review を、同じ直しのテストの 2 か所は「the run 5 cross-review」と書く。cycle-count は今 4 | `reseat_progress.go:545`(b)、`:547-548`(a)。`reseat_progress_start_test.go:79`、`prompts_leader_reseat_test.go:429`、`:469` | (a) 段落を折り直す。(b) 「the run 5 cross-review」にそろえる |
| C4-4 | LOW | prompt | 座席の確認の項は、reviewer の QUESTION を gate か cross-review の項へ回したあと「(送り直しの上限と BLOCKED も、その項のとおりです。…)」と言う。cross-review の項には上限がある(`-r3` から先は送り直さず BLOCKED、`:1268-1270`)。手順 12 の gate の項(`:758-762`)には上限がなく、あるのは「何が違ったのかわからなければ BLOCKED」だけで、手順 12 の「2 回続けて通らなければ」(`:708-711`)が数える理由は 3 つで、QUESTION は入らない。gate の TASK では、言った上限がどこにもない。向きは、leader が言った上限を gate の項に探して見つけられないか、自分で数を作る(2 回、など)ことで、通知は消えない | `leader.md:225-226`、`:708-711`、`:758-762`、`:1268-1270`。テストが句を固定している(`prompts_leader_reseat_test.go:448`) | 句を「(cross-review の TASK には送り直しの上限があり、その項のとおりです。どちらも、何が違ったのかわからなければ BLOCKED を上げます。…)」の形にして、テストの句も合わせる |

### 読んだが指摘にしないもの

- `leader.md` の同じ形の `origin/<既定のブランチ>..HEAD` が 3 か所残る(`:591` の sync-docs の前のコミット、`:718` と `:767` の `wip:` の確かめ)。手元の main の継承 commit を数える点は節と同じ。ただし `wip:` の件名の commit は feature branch 専用の hook が作るもの(main には作らない)で、sync-docs の件名は slug ごとに決まる。どちらも手元の main に載っていることは考えにくく、向きは余分な BLOCKED か余分な「凍らせる」ファイルで、この直しの 2 件目(節と計画の項)の範囲の外。次に手順 9・12・14 を触るときの材料。
- 「どちらの TASK かは、QUESTION の TASK_ID か、その座席の行の TASK_ID で見ます」(`leader.md:226-227`)は、行が `不明` で QUESTION にも TASK_ID がないと決め手がない。gate の TASK の TASK_ID は `gate-<n>` で 2 行目にあり、台帳の記録が切れるのは raw の TASK か長い ID のときだけ(cycle 2 の run 3 で `Parse` と `truncateForDetails` を読んで確かめた。この run では読み直していない)ので、届かない見込み。
- fixture の前半 12 行は `progressStartRepo` と同じ。1 つにまとめられるが、2 つの形の差(origin の ref と 2 つ目の commit)が読み取りやすいので、そのままでよい。
- ループが持つ「origin の ref と手元の ref の組」は、`resolveGateBase`(`gate.go:638`)の組と 2 か所目。形が 3 つ目に増えない限りずれない。

### Positive notes

- 範囲を `HEAD --not` で書き、`base.ref` を最初に置き、他の ref は存在を確かめてから足す。存在しない ref を渡して git が `bad revision` で落ちる道がない。
- `log` と `rev-list --count` が同じ `span` を使う。行と総数が別の範囲を数える余地がない。
- `leader.md` の計画の項の `git log` は、コードと同じ範囲を `--` つきで言い、「branch の commit」の行も同じと言う。テストがコマンドと除く理由の 2 句を固定する。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

### `/sync-docs` への引き継ぎ(指摘ではない)

- `docs/specs/2026-11-12-org-reseat.md:88`(FR-9 の 2 本目)は「branch の commit(origin の既定のブランチから HEAD まで…)」と書く。`4096ae88` で、origin と手元のどちらの既定のブランチにもない commit になった。`既定のブランチにない` と `default branch does not have` で grep した限り、README、`/org` と `/director` の skill、`ralph org reseat --help` に同じ範囲を言う文はなかった。
- 登録簿(`docs/tech-debt/README.md`)の行で `leader.md`(1,572 行)、`prompts_leader_reseat_test.go`(706 行)、`reseat_progress_start_test.go`(477 行)、`reseat_progress.go`(600 行)の大きさを引くものは、`4096ae88` で数が動いた。`reseatProgressCommitsIn` は 33 行。

## Tech debt identified

C4-1〜C4-4 を直さず進めるなら、次の 1 行を register に足す(`/sync-docs` か `/pr` の担当)。直す場合は Go か `leader.md` を触るので、cap をもう一度上げて `/self-review` から回し直す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C4-1(`reseatProgressCommitsIn` の改訂の並びに `--` がなく、根に `HEAD` のファイルがある repo では commit の行が読めなかったになる)、C4-2(`reseatProgressSupplied` の doc が前の範囲のまま)、C4-3(doc コメントの折り返しと呼び名)、C4-4(gate への QUESTION の送り直しに言い過ぎの「上限」) | LOW。C4-1 は根に `HEAD` のファイルがある repo の節で commit の行が出ないだけで、節の残りと通知は消えない。C4-2 から C4-4 は読む人の誤解で、通知は消えない | cap 4 の最後の run で、LOW 4 件のために Go と `leader.md` を直すと、cap を上げて pipeline が `/self-review` から回り直す | 次に `reseat_progress.go` の commit を読む関数か、`leader.md` の座席の確認の項を触るとき | この報告書 |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 4 件)
- Follow-ups:
  - run 5 の cross-review の 2 件は、コードと prompt の両方で閉じている。新しい欠陥は通知を消す向きになく、直さなくても閉じたままになる。
  - 直すなら C4-1 だけ(`span` に `"--"` の 1 要素。挙動が変わるのはこの 1 件で、ほかは読む人向けの文言)。ただし cap 4 の最後の回なので、直す場合は cap を上げて `/self-review` から回し直す。リードの判断。
  - `/sync-docs` への引き継ぎ: 上の節。

## Run 5 (cycle-count 3, cap 4): HEAD `235621b2`(履歴)

- **Merge: yes**(run 5: CRITICAL、HIGH、MEDIUM はなし。LOW 4 件。cycle 3 の cross-review の 2 件は直った。時刻の比べ方は agmsg と台帳の時刻の書き方に合う。LOW は、「あれば」の主語、QUESTION の確かめの足場、「命」の語、テストのコメントと重複の 4 つで、どれも通知を消さない)

Scope: `git diff cef100ae..HEAD` の 5 ファイル。コードに当たるのは `5ab60578`(`internal/org/prompts/leader.md`、`internal/org/prompts_leader_reseat_test.go`)で、全部読んだ。`780a7adf`(cycle 3 の triage)と `235621b2`(計画の記録)は記録なので、2 件の finding の写しと直しの説明が `5ab60578` の中身と合うかだけ見た。`leader.md` は座席の確認の項目の全文とミッションの 5 を読み、テストは新旧の 3 つと helper を読んだ。確認に使った読み物: `reseat_progress.go` の last-TASK の組み立て(`readReseatProgress`、`reseatProgressSentTask`)、`verbs.go` の `sent` を書く位置と `confirmSubmitted`、agmsg の `history.sh` と storage ドライバの時刻の書き方(`drivers/storage/jsonl.sh:32`、`sqlite.sh:21`)、`docs/tech-debt/README.md` の行 519〜526。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれ(`/verify`、`/test`、`/sync-docs` の担当)。再提起しなかったもの: 行 519〜526 に登録済みの項目(行 522 の `history.sh` の節の偽の 1 句を含む)。

### 依頼された 4 点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| 新しい文は順に読めるか | 読める。ただし 3 か所が文の位置頼み(C5-1、C5-2) | 不明の行の枝は「送る前に、下の QUESTION の確かめだけを、この行の時刻でします」(`leader.md:190`)で下の確かめに渡し、確かめは「上の `history.sh`」(`:211`)で、不明でない行の枝の中にあるコマンド(`:195-197`)を指す。読み順では、確かめが両方のあとにあるので通る |
| 時刻の比べ方は正しいか | 正しい | agmsg の `at` は storage ドライバが `date -u +%Y-%m-%dT%H:%M:%SZ` で書き(jsonl と sqlite)、`history.sh` が `[$ts]` で出す。`%f` で小数の秒を足す書き方は `sqlite-sync.sh` の sync 用の表だけで、メッセージには使われない。台帳の `ts` は `o.now()`(`verbs.go:671`)で UTC の RFC3339。`sent` は Enter と送信の確認(`confirmSubmitted`、`working` を見たら戻り、既定 3 秒で打ち切る)のあとに書くので、行の時刻は Enter より最大数秒あとになる。座席の最初の RESULT、BLOCKED、QUESTION は LLM の 1 ターンのあとなので、これより前には来ない(実測はしていない)。比べ方は「同じかそれよりあと」で、同じ秒の返事も拾う。同じ TASK_ID で送り直すと行の時刻が進み、古い返事が外れる。これは意図どおり。前提は同じホストの時計 |
| 新しいテストは主張を固定しているか | 固定している。例外は C5-4 の 2 点 | 時刻の条件、「それより前のもの」の文、QUESTION の確かめ、ミッション 5 の名指し、不明の枝の「確かめだけ」を、この順で `assertPhraseOrder` が固定する。節の last-TASK の行が UTC の時刻で終わることは `reseatProgressSection` の実際の出力で見ている。DECISION を送っても最後の TASK の行は動かない(`reseatProgressSentTask` は TYPE が TASK の `sent` だけを TASK と読む)ので、QUESTION に答えたあとで立て直しても行の時刻は進まず、返事を落とさない |
| 変えた文が古い文を偽にしたか | `leader.md` の中ではなし。外に 2 つ(C5-4 の 1 つ目と、下の `/sync-docs` への引き継ぎ) | 「返事を捨てることはしません」は、数える返事の話として残る。「履歴は読まず」は「履歴で返事を探さず」に変わり、テストが古い語の再登場を拒む。偽になったのは、`UnknownTaskID` のテストの doc コメントと、仕様の 3 回目の段落の (3) |

### Findings

この run の指摘は C5-1 から振る(C3-n、C2-n、R3-n と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C5-1 | LOW | readability | 時刻の条件の 2 文(「見るのは…だけです」「それより前のものは…ありません」)が、返事を探す文と「あれば、前の leader が扱ったかどうかにかかわらず、今届いたものとして扱います」のあいだに入った。直前までは「…があるかを見ます。あれば、…」で「あれば」の主語が隣にあったが、今は直前の主題が「それより前のもの」になり、文だけを読むと「それより前のものがあれば今届いたものとして扱う」とも読める。その読みは、閉じたはずの cross-review の finding 1(前の命の返事を今の TASK への返事と取る)を戻す。正しい読みは前後の流れから取れるので、外れる見込みは低い。向きは「余分な手」で、通知が消える向きではない | `leader.md:198-203`(「あれば」は `:202`)。テストが `それより前のものは…ありません` の次に `あれば、前の leader が扱った…` を置く順で固定している(`prompts_leader_reseat_test.go:433-434`) | 「あれば」を「条件に合う返事があれば」にする。または、「それより前のものは…」の文を「返事を捨てることはしません」の文のあとへ移す。テストの句も合わせて直す |
| C5-2 | LOW | maintainability | QUESTION の確かめの足場が、文の位置に頼っている。(a)「何かを送る前に、QUESTION を確かめます」は、返事がなかった枝に限るとは書かれておらず、直前の「返事がなければ、…入力待ちです」(`:210`)の位置だけが限る。返事があった枝(BLOCKED を受けて直しの TASK を送る、など)でこの文を適用すると、古い QUESTION に DECISION を送り「RESULT か BLOCKED を待つ」ことになり、届いた返事の手を飛ばす。(b) `history.sh` のコマンドは「TASK_ID が `不明` でない行があれば」の枝の中にだけ書かれている(`:195-197`)。不明の行の leader は「履歴で返事を探さず」(`:188`)と読んだあとで、確かめの「上の `history.sh`」(`:211`)に着く。(c)「QUESTION がなければ、」(`:191`)は implementer の 1 文にだけかかり、続く reviewer の 1 文は、文の上では外れる(`:190` の「送る前に…確かめだけを」が意味ではかけている) | `leader.md:190-191`、`:195-197`、`:210-217`。テストは「何かを送る前に、QUESTION を確かめます」と、返事なしの文のあとにあることを固定するが、枝の限りは固定しない | 確かめの頭を「返事がなく、何かを送る前に」とし、コマンド(`bash {{AGMSG_SCRIPTS}}/history.sh {{TEAM}} leader 50`)は確かめの側か枝の外に 1 回だけ書く。(c) は「QUESTION がなければ」を implementer と reviewer の両方の文頭にするか、2 文を 1 文にまとめる |
| C5-3 | LOW | naming | 同じ項目の中で、同じ概念を「この org の今の life」(`:184`)と「disband の前の命」(`:200`)の 2 語で書く。prompt を読むのは LLM で、「命」だけでは「命令」の略とも読め、`life` で探しても結べない。仕様は「命」を使う(`docs/specs/2026-11-12-org-reseat.md:25`)が、`leader.md` で「命」を使うのはこの 1 か所だけ | `leader.md:184`、`:200`。`prompts_leader_reseat_test.go:433` が `disband の前の命など` を固定している | 「disband の前の life など」にして、テストの句も合わせる |
| C5-4 | LOW | test | (a) `TestRenderRolePrompt_Leader_ReseatRuleSeatCheckUnknownTaskID` の doc コメントは「The item reads no history for it」のまま。この直しで、不明の行も QUESTION のために履歴を読む。同じ直しの新しいテストのコメントは「it now reads it for the QUESTION only」と書き、2 つが食い違う。(b) 新しいテストの末尾の 10 行は、`NeverDiscardsAReply` の末尾の 10 行と同じ検査(`- implementer: slice-1 ` の行が UTC の時刻で終わる)で、失敗メッセージの文だけが違う。その前のコメント「history.sh lists the leader's agmsg traffic only; …」の前半は、下の検査とつながらない(検査は節の行の形だけを見て、`history.sh` の出力は見ない) | `prompts_leader_reseat_test.go:473`(a)、`:403-412` と `:455-467`(b) | (a) コメントを「reads no history to look for a reply」にする。(b) 検査を helper に 1 つにして 2 つのテストから呼ぶか、新しい方を消してコメントの理由を `NeverDiscardsAReply` に寄せる |

### 読んだが指摘にしないもの

- QUESTION の確かめは、前の leader がもう DECISION で答えた QUESTION も見つけて、もう一度答える。leader が送った DECISION は agmsg の履歴に残らない(`Org.Send` は pane に打つだけで、登録済みの行 522 のとおり)ので、確かめでは見分けられない。台帳の `sent` には DECISION も残るが、節は TYPE が TASK の `sent` だけを出す(`reseat_progress.go` の `reseatProgressSentTask`)。向きは「知らせる側」(座席に DECISION がもう 1 通届く)で、cycle 3 の triage が finding 2 を知らせる側と分類した理由と同じ。直すなら Go(節に DECISION の行を足す)なので、この直しの範囲の外。
- 履歴の 50 件の窓に、QUESTION の確かめも従う(「上の `history.sh`」)。足りなければ数を増やす指示は返事の探索の文の中にあり、確かめがそれを引き継ぐかは「上の」の読みに頼る。C5-2 (b) で、コマンドを確かめの側に置けば同時に解ける。
- 古い語を拒むテストの `履歴は読まず` は、旧文の正確な語だけを拒む(新しい文が同じ意味を別の語で書けば通る)。ただし新しい文を正の固定が押さえているので、害はない。

### Positive notes

- 2 件の finding を、それぞれ 1 つの条件で直している(返事は行の時刻以後のものだけ、送る前に QUESTION を見る)。同じ秒の返事を落とさないよう「同じかそれよりあと」にしてあり、送り直しで行の時刻が進むことと合う。
- QUESTION の確かめは、不明の行の枝にも入れてある。不明の枝は履歴を読まない規則だったので、そこへ確かめだけを足す形は、規則の元の意図(TASK_ID を推し量らない)を壊さない。
- DECISION が last-TASK の行を動かさない点を、`reseatProgressSentTask` の TYPE の見分けが保っている。確かめで DECISION を送っても、次の立て直しが返事を落とす材料にならない。
- テストが、直しの 3 つの句を 1 つずつ消した prompt で落ちることを、直しの記録(計画の注)が書いている(再実行はしていない)。秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

### `/sync-docs` への引き継ぎ(指摘ではない)

`docs/specs/2026-11-12-org-reseat.md:19` の 3 回目の段落の (3) は「規則はその座席の履歴を読まず、TASK_ID を推し量らず…」と書き、不明の行の枝は履歴を読まないとしている。`5ab60578` で、その枝も QUESTION のために履歴を読む(返事は探さない)。座席の確認が数える返事を時刻で限ったこと、送る前の QUESTION の確かめを足したことも、仕様と `docs/tech-debt/README.md` の行 520 の (e) の範囲の文に入っているか確かめる。`5ab60578` は `docs/` の仕様を触っていない。

## Tech debt identified

C5-1〜C5-4 を直さず進めるなら、次の 1 行を register に足す(`/sync-docs` か `/pr` の担当)。直す場合は `leader.md` を触るので、cap をもう一度上げて `/self-review` から回し直す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C5-1(「あれば」の主語)、C5-2(QUESTION の確かめが返事なしの枝の限りとコマンドの場所を位置に頼る)、C5-3(「命」と「life」)、C5-4(テストの古い doc コメントと同じ検査の重複) | LOW。どれも読み違えが起きたときの向きは余分な手で、通知は消えない。テストの 2 点は読む人の誤解とメンテの手間だけ | cap 4 の run で、通知を消さない LOW 4 件のために `leader.md` を直すと、cap を上げて pipeline が `/self-review` から回り直す | 次に `leader.md` の座席の確認の項目か `prompts_leader_reseat_test.go` の座席の確認のテストを触るとき | この報告書 |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 4 件)
- Follow-ups:
  - C5-1 から C5-3 は `leader.md` の 3 か所の言い換えで、C5-4 はテストのコメントと helper の話。直すかどうかは、cap の扱いとあわせてリードの判断。通知が消える欠陥はなく、直さなくても cross-review の 2 件は閉じている。1 つだけ直すなら C5-1(閉じたはずの finding 1 の読み違いに近い)。
  - `/sync-docs` への引き継ぎ: 上の節。`leader.md` も Go も触らないので、pipeline の再実行は要らない。

## Cycle 3 (cap 3、最後の回): HEAD `b79f300e`

- **Merge: yes**(cycle 3: CRITICAL、HIGH、MEDIUM はなし。LOW 2 件。3 回目の cross-review の finding 1 は `04d3d986` で閉じ、cycle 2 の finding 2 を黙って落ちる向きで開き直してはいない。ただし、済んだ手の見分けの例のうち 1 つは節の行から確かめられない)

Scope: `04d3d986`(3 回目の cross-review の finding 1 の直し。`internal/org/prompts/leader.md` と `internal/org/prompts_leader_reseat_test.go`)。`b79f300e` は計画の記録だけ。2 ファイルの差分を全部読み、`leader.md` は座席の確認・計画・gate・受信箱・座席の 5 項目と、ミッション 5、手順 8・10・12、「agmsg の受信箱の運用」の節を読んだ。確認に使った読み物: `reseat_progress.go` の commit の行を作る `git log --format=%h %s`、agmsg の `history.sh`、cross-review の triage report(3 回目の finding 1・2)、`docs/tech-debt/README.md` の行 519〜525。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれ(`/verify`、`/test`、`/sync-docs` の担当)。再提起しなかったもの: 行 519〜525 に登録済みの項目と、3 回目の finding 2(WORTH_CONSIDERING)。

### 依頼された 3 点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| finding 1(後の TASK を理由に返事を捨てる)は閉じたか | 閉じた | 座席の項目は、返事が見つかれば「前の leader が扱ったかどうかにかかわらず、今届いたものとして扱います」と書き、「返事を捨てることはしません」と続ける。時刻を比べて捨てる枝(「返事よりあとの時刻の行が 1 つでもあれば」)は項目から消え、テストが両方の文の再登場を拒む。BLOCKED が別の座席への TASK で消える経路はない |
| cycle 2 の finding 2(扱った失敗を繰り返す)を黙って落ちる向きで開き直していないか | していない。ただし済んだ手の見分けに 1 つ弱い例がある(C3-1) | 返事は必ず扱うので、外れても向きは「余分な手」で、「通知が消える」にはならない。3 つの例のうち (a)「計画がもう `docs/plans/active/` に戻っていれば archive の戻しはしない」は、手順 10 の末尾と手順 12 の `GATE: fail` が戻す先(`docs/plans/active/`)の今の状態そのものを見るので、外れない(archive のコミットを作らない org では計画が最初から active にあり、戻す手もない)。(c)「その返事のあとに送った、同じ直しの TASK」は、implementer の最後の TASK の行の TASK_ID と時刻で見られる。(b)「「branch の commit」に直しの commit があれば」は、行に時刻も TASK もなく、返事の失敗に結びつけられない(C3-1) |
| 到達でき、座席の確認の残りと手順 8・12 と食い違わないか | 食い違いなし | 入力待ち・行が不明でない・返事あり、で到達する。項目の頭の gate の例外は残る(gate が通って head が同じなら、この確認の残りをしない)。返事の TASK_ID の頭で手順 12(`gate-`)、「cross-review」の小節(`xreview-`)、手順 8(それ以外)に分かれるのは前の run と同じ。archive の戻しの条件は手順 12 の `GATE: fail` の「archive のコミットを作っていれば `git revert`」と同じ状態を指す。「受信箱」の項目の「どの件も `ralph org escalate` で上げ直しません」は、返事に応じる手に escalate がある場合の「済んでいる手」の確かめと合う |
| テストの変更は妥当か(固定を弱めていないか) | 弱めていない。ただし例 (b)(c) は固定されない(C3-2) | 時刻を比べる枝を固定していた文は、規則から枝が消えたので、固定も消えた(置き換えで、足した文が同じ数以上ある)。足された固定は、返事を必ず扱う文、「返事を捨てることはしません」、「済んでいる手だけは繰り返しません」、例 (a)、「返事がなければ」の順。古い枝の文は否定の検査(2 句)で拒まれる。`containsPhrase(item, "あれば今届いたものとして扱います")` の否定の検査は、新しい文が同じ主語で「前の leader が扱ったかどうかにかかわらず」を足した形なので、残す意味が薄く、消えたのは妥当 |

### Findings

この cycle の指摘は C3-1 から振る(C2-n、R3-n と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C3-1 | LOW | maintainability | 座席の確認の「済んでいる手は繰り返さない」の 3 つの例のうち、「「branch の commit」に直しの commit があればその直しは頼まない」は、その commit がこの返事の失敗の直しかを、節の行から確かめられない。「branch の commit」の行は `git log --format=%h %s` の hash と件名だけで、時刻も TASK_ID も持たない(返事の時刻とも、座席ごとの最後の TASK の行とも結べない)。前の round の `fix:` commit が 1 つあれば、新しい `GATE: fail` の BLOCKED の直しを「済んでいる」と読めてしまい、leader は直しの TASK を送らずに手順 8 の往復へ戻る(gate の失敗なら archive の戻しだけして直しを飛ばす)。通知は消えない: 直さずに通った再レビューと次の gate が同じ失敗を上げ、そのときは leader 自身が送った TASK への返事なので、直しの TASK が出る。損は gate 1 回分の往復で、cycle 2 の finding 2 が嫌った「黙って落ちる」向きではないので LOW。他の 2 つの例は確かめられる: (a) は計画の置き場所という今の状態、(c) は節の implementer の行の TASK_ID と時刻 | `leader.md:202`(例 (b))、`:203-204`(例 (c))、`reseat_progress.go` の commit の行を作る `git log --format=%h %s`(`reseatProgressMaxCommits` 件、新しい順)、節の `branch の commit: N 件(新しい順に M 件)` の行 | 例 (b) を落として (a)(c) だけにする。残すなら、確かめる材料を「その返事のあとに implementer に送った TASK(最後の TASK の行の時刻)」に限り、commit の件名だけでは何の失敗の直しか分からないと 1 句足す |
| C3-2 | LOW | test | `04d3d986` のテストが固定するのは、座席の確認の「済んでいる手は繰り返さない」の文と例 (a) までで、例 (b) と (c) の文は項目から消してもテストが通る。古い `...SkipsRepliesAlreadyHandled` は、規則の枝を最後まで 1 文ずつ固定していた。`reseatTimeShape` の doc コメントは「座席の確認の項目が、`history.sh` の行の時刻と同じ書き方と言う」と書くが、その文は `04d3d986` が時刻の比べ方の説明ごと項目から落としたので、コメントが偽になった。テスト本体のコメント(「節の最後の TASK の行の時刻は、項目の例の確認が読む」)も、読む側の例 (c) が固定されていないので、検査(`found` のループ)がどの固定にも結びつかない。例 (c) は返事の時刻を読む前提だが、その時刻の出どころ(`history.sh` の行の `[` と `]` のあいだ)は項目から消え、「その返事のあとに送った」だけが残る | `prompts_leader_reseat_test.go:363-365`(`reseatTimeShape` の doc)、`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckNeverDiscardsAReply` の固定の列(`assertPhraseOrder` の 5 句、例 (b)(c) なし)と `found` のループ | 固定の列に例 (c)(C3-1 の直しで (b) を落とすなら (c) だけ)の文を足す。`reseatTimeShape` の doc は今の項目に合わせて直す。例 (c) が返事の時刻を読むなら、出どころを 1 句、項目に戻す |

### 読んだが指摘にしないもの

- 例が失敗の直しだけを挙げ、成功側の手(implementer の RESULT で reviewer にレビューの TASK を送る、reviewer の `GATE: pass` で手順 9 に進む)には例がない。leader だけを立て直すと、`(active)` で残った implementer と reviewer の最後の返事が両方見つかるので、よくある形で、一般の文(「応じる手の一つ一つ」)の判断に任される。外れたときの向きは余分な TASK で、triage が報告側の穴として tech-debt に回した重複と同じ(3 回目の triage の Triage context)。
- 「返事がなければ、その座席はその TASK に答えないまま入力待ちです。」のあとに、返事なしの枝の文が続く並びは前の run と同じで、返事ありの枝は「あれば…」の 1 文で終わる。到達しない文はない。
- `leader.md` の `送り直さない、のように見ます)。返事がなければ、…` の行は、前後の折り返しより長い。プロンプトの本文で、折り返しは挙動に効かない。

### Positive notes

- 3 回目の cross-review の finding 1 に対して、規則を 1 方向(返事は必ず扱う)に倒し、cycle 2 の finding 2 の懸念を「手ごとの確認」で別に受けた。両方の finding が互いの反対側にある点を、triage が名指しした扱い(報告側の穴は tech-debt)に合わせている。
- 例 (a) は、手順 10 の末尾と手順 12 の `GATE: fail` が同じ状態(`docs/plans/active/` に計画がある)で戻しの要不要を決めるので、判定が見て分かる。
- テストが、古い枝の 2 句を否定の検査で拒む。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

### `/sync-docs` への引き継ぎ(指摘ではない)

この直しが規則から外した「時刻を比べて返事を扱い直さない」枝を、まだ書いている箇所がある: `docs/specs/2026-11-12-org-reseat.md:19` の (2)(座席の確認は、返事の時刻より後の時刻の行が 1 つでもあれば返事を扱い直さない)、`docs/tech-debt/README.md` 行 520 の (e) の (1)(同じ秒の TASK)と (2)(「どの座席でも後の TASK が 1 つあれば扱った」の限界)、その影響の欄の (e)(扱っていない失敗が落ちる、扱った失敗が 2 回扱われる)の前半 2 つ。どれも消えた枝の話なので、`/sync-docs` が「`04d3d986` で枝を外し、返事は必ず扱い、済んだ手だけ飛ばす形に変わった」と直す。行 522 が `d85fc0ef` の枝を書き換えた事実を述べる箇所は、当時の記録なのでそのまま。

## Tech debt identified

cycle 3 は cap の最後の回で、`leader.md` を直すと cap をもう一度上げて `/self-review` から回し直す。C3-1 と C3-2 を直さず進めるなら、次の 1 行を register に足す(`/sync-docs` か `/pr` の担当)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C3-1(座席の確認の例 (b)「「branch の commit」に直しの commit があれば」が、時刻も TASK もない commit の行では返事の失敗に結びつかない)、C3-2(例 (b)(c) の文が固定されず、`reseatTimeShape` の doc が項目から消えた文を指す) | LOW。C3-1 は外れると gate か review 1 回分の往復が増えるだけで、次の失敗で直しの TASK が出る。C3-2 は文を消しても気づかないテストの穴と古いコメント | 最後の回で、どちらも通知を消さない LOW(今 `leader.md` を直すと cap を上げて pipeline が `/self-review` から回り直す) | 次に `leader.md` の座席の確認の項目か `prompts_leader_reseat_test.go` の固定の列を触るとき | この報告書 |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 2 件)
- Follow-ups:
  - C3-1 と C3-2 は同じ 1 文(例 (b))と 2 文の固定の話で、直すなら `leader.md` の 1 文を落とし、test の固定の列と 1 つのコメントを直す。cap の最後の回なので、cap を上げて pipeline を回し直すか、上の 1 行を register に書いて進むかは、リードの判断。通知が消える欠陥はなく、直さなくても 3 回目の finding 1 は閉じている。
  - `/sync-docs` への引き継ぎ: 上の節。Go も `leader.md` も触らないので、pipeline の再実行は要らない。

## Cycle 2, run 3 (cap 3): HEAD `d85fc0ef`(履歴)

この節の「Tech debt identified」と「Recommendation」は、cap 3 に上げたあとの cycle 2 の run だった当時のもの。R3-1 と R3-2 の扱いは、cycle 3 の cross-review の triage と register に引き継がれている。

Scope: `d85fc0ef`(cycle 2 の cross-review の ACTION_REQUIRED 3 件の直し。`internal/org/prompts/leader.md`、`internal/org/reseat_progress.go` と、その test 2 つ)。4 ファイルの差分を全部読み、`leader.md` は座席の確認・gate の 2 項目と手順 12・14 を全文で読んだ。確認に使った読み物: `protocol.Parse` と `truncateForDetails`、`Send` が `sent` の Details を作る箇所(本文が最後に付き、後ろに足す語がないこと)、`gh pr list --json` の項目名(`gh` に未知の項目名を渡して出る一覧に `headRefOid` がある)、agmsg の `history.sh`、cross-review の triage report、`docs/tech-debt/README.md` の行 519〜524。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれの詳細(`/verify`、`/test`、`/sync-docs` の担当)。

### cross-review の 3 件が閉じたか

| # | 内容 | 状態 | 根拠 |
| --- | --- | --- | --- |
| 1 | gate の項目が PR の state だけを見て、gate を通したコミットが push されていない OPEN の PR を完了と報告する | 閉じた | `gh pr list … --json number,state,url,headRefOid` で PR の head を読む(`headRefOid` は `gh` の項目名にある。`merge.go`・`mergeorder.go` も同じ項目を使う)。PR の集まりは、(A)head が gate の head と同じ OPEN か MERGED、(B)A がなく head が違う OPEN、(C)A も B もなく MERGED、(D)CLOSED だけ、(E)PR なし、のどれか 1 つに入り、A は手順 15 へ、B は `wip:` の確かめと push のあと再読して一致なら手順 15・不一致なら BLOCKED、C と D は BLOCKED、E は手順 14 へ進む。B の push は force なしで、手順 14 の `wip:` の確かめと push を指すだけで、コマンドは書き写していない(手順の順を読むテストが push の最初の出現を手順 14 に置くため。テストがそれを固定する)。head が先に進んだ PR への push は non-fast-forward で失敗し、手順 14 のとおり BLOCKED になる |
| 2 | 座席の確認が、履歴で見つけた返事をすべて「今届いたもの」として扱い、前の leader が扱った失敗を繰り返す | 閉じた | 返事の時刻と、節の最後の TASK の全行の時刻を比べ、あとの行が 1 つでもあれば扱い直さず、座席は空いたものとしてほかの座席へ進む。あとの行がなければ従来どおり今届いたもの。出口は両方にある。時刻の書き方の主張(history.sh の `[` `]` の中が UTC の `YYYY-MM-DDTHH:MM:SSZ`、節の行の時刻と同じ書き方)は、台帳の `ts` が UTC の RFC3339 であること、既定の sqlite の driver が秒までの `%SZ` を書くことと合う |
| 3 | `truncateForDetails` が切った TASK の TASK_ID が `-` や切れ目の印つきで出て、返事と結びつかない | 閉じた | `reseatProgressSentTask` は、本文が切れ目の印で終わり、かつ TASK_ID が空か印を含むときだけ「不明」にする。TASK_ID の行が 200 byte より後に始まる、跨ぐ、ちょうど終わる(`\n` の前で切れたか値が続くか見分けられないので不明側に倒す)の 3 形は不明、TASK_ID が切れ目より前に全部ある形は後ろの行が切れても残る。切れ目の印は `truncateForDetails` の末尾にだけ付き、`Send` は本文のあとに何も足さないので `HasSuffix` が当たる。規則側は不明の行で履歴を読まず、同じ TASK_ID の送り直しもせず、implementer と reviewer のそれぞれに出口がある |

### 直しが持ち込んだ欠陥の確認

見つからなかった。確認した型と結果:

- 手順に到達できない・矛盾する指示: gate の項目の 5 つの場合(上の表)は互いに素で、`PR がなければ手順 14` が残りを受ける。座席の項目の入力待ちは、行なし・不明の行・不明でない行(返事あり=あとの行なし/あり、返事なし)の順に分かれ、どれも次の動作で終わる。
- 規則が読む値を節が出しているか: 不明の行の形 `- <座席>: 不明 (記録が切れている、<時刻>)` は Go の `Sprintf` と test が同じ文字列で固定し(`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckUnknownTaskID` が節の出力から行の形を作って規則の文と照らす)、`不明` の定数を test が固定する。時刻は、不明でない行は括弧の中、不明の行は `、` のあとと規則が書き分ける。規則が節の行の時刻を比べる相手は、静的な節(タスクに入った文)で、leader がこれから送る TASK の行は含まない。
- コマンドの形: `gh pr list --head <ブランチ> --state all --json number,state,url,headRefOid` は有効。push は手順 14 の `git push -u origin <ブランチ>` を指す。
- 不明の判定の解析: `Parse` は TASK_ID の行を最後の値で上書きし、ヘッダの形でない行で止まる。切れ目が TASK_ID のキーの途中なら、その行は `:` がなく解析が止まって TASK_ID は空になり、印で終わるので不明。切れ目が `TASK_ID: foo\n` の直後なら 3 行目が印だけの行になり、TASK_ID は `foo` のまま残る(正しい)。`value()` を通る前の値で判定するので、400 rune の切り詰めは影響しない。
- 既存 test の弱め: 変わった期待は、直した文に合わせて言い換えたもの(「行があれば」→「TASK_ID が不明でない行があれば」など)と、署名の変更への追従だけで、検査の対象を外したものはない。古い読み(「あれば今届いたものとして扱います」、`--json number,state,url` の読み、「OPEN か MERGED の PR があれば PR は作りません」)が消えたことを見る否定の検査が足されている。
- 切れた TASK の test が本物の切り取りを通っている: `progressSentEvent` は `truncateForDetails` を呼んで Details を作る。

確認したが指摘にしないもの:

- 不明の行に至るのは、TASK_ID より前にヘッダがある TASK、TASK_ID が約 180 byte を超える TASK、TASK_ID のない raw の TASK に限る。`leader.md:1315` が `TASK_ID:` の次の行に `PLAN:` を置くと決めているので、標準の TASK は TASK_ID が先頭 2 行目で切れ目の前に入る。到達の少ない形に規則の文が増えているが、cross-review の指摘で、triage が「Real」とした範囲なので、この run では取り上げない。
- 不明の行の implementer の出口(「まだ済んでいない最初のスライス」)は、全スライスが済んでいるときの行き先を書かない。同じ項目の不明でない行の出口(「計画の残りのスライスから次の TASK を送り」)も同じで、直しが持ち込んだものではない。
- agmsg の driver が optional の `sqlite-sync` のときは、`history.sh` の時刻が `%fZ`(ミリ秒つき)になる。規則は時刻の大小を読むだけなので、比べ方は変わらない。
- push のあとの `gh pr list` の再読で、GitHub 側の更新が遅れると一致しないことがありうるが、その場合は BLOCKED になって人に渡り、誤って完了を報告する向きではない。

### Findings

この run の指摘は R3-1 から振る(C2-n と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R3-1 | LOW | docs-in-register | `d85fc0ef` が触った 3 つのファイルについて、register の行が書いた数と「最後の commit」が古くなった。行 524: `reseat_progress_test.go` は「869 行、`wc -l` は `b8a9bd38` で、最後に変えた commit」と書くが、今は 969 行で最後は `d85fc0ef`(同じ行の「`reseat_progress.go` 551」「`prompts_leader_reseat_test.go` 516」も 582 と 615)。行 519 の (a): `reseatProgressSection` は「103 行」と書くが 107 行。行 520 の (a)(b) の取り消し線つきの「RESOLVED」は、`ec10ba6f` の時点の読み(`--json number,state,url`、「OPEN か MERGED の PR があれば PR は作らない」)をそのまま書き、今の規則(PR の `headRefOid` と gate の head を比べる)と読み比べると食い違う。行 519・523・524 の「次に触るとき」の条件は、この直しがファイルと `reseatProgressSentTask` を変えたので、また満たされている(支払いはされていない) | `docs/tech-debt/README.md` の行 519、520、524(`wc -l` は `internal/org/reseat_progress_test.go` 969、`reseat_progress.go` 582、`prompts_leader_reseat_test.go` 615)、行 523 の条件(`reseatProgressSentTask`) | `/sync-docs` が数を直し、行 520 の (a)(b) に「`d85fc0ef` で PR の head を gate の head と比べる形に変わった」と 1 句足す。条件が満たされたことは、cycle 2 の run で行 519 に書いた型(支払いなしの訂正)に揃える |
| R3-2 | LOW | naming | `reseat_progress.go` の `reseatProgressUnknownTaskID` は、不明の行に出す語 `不明` の定数だが、同じ struct の bool は `TaskIDUnknown` で、語と旗の名前が語順だけ違う。定数の doc コメントは「the first word after the colon is the marker」と書くが、その 6 行上の `reseatProgressTruncatedMarker` も「the marker」で、同じ語が別のものを指す。test の `TestReseatProgress_TheTruncatedMarkerIsTheOneTruncateForDetailsWrites` は、名前が切れ目の印だけを言うのに、`不明` の定数と 2 語の行の形も固定する | `reseat_progress.go:72-91`(`TaskIDUnknown`、`reseatProgressTruncatedMarker`、`reseatProgressUnknownTaskID`)、`reseat_progress_test.go` の同 test | 定数を `reseatProgressUnknownTaskIDWord` にし、doc コメントは「the word」と書く。test は名前を 2 つに分けるか、名前に行の形を足す。挙動は変わらない |

### Positive notes

- `reseatProgressSentTask` の doc コメントが、切れ目で TASK_ID が残る条件と不明になる条件(切れ目の印、TASK_ID が空か印を含む)を書き、「TASK_ID が切れ目より前に全部ある形は、後ろの行が切れても残る」まで言う。テストの 7 つの形(後ろに切れ目、跨ぐ、ちょうど終わる、短い TASK_ID と長い本文、短い TASK_ID と長い PLAN の行、切れた raw、切れていない raw)が、doc の場合分けと 1 対 1 に並ぶ。
- 規則の文が、gate の項目の B で push のコマンドを書き写さず手順 14 を指すこと、その理由(順を読む test)が test の中に書いてあること。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

## Tech debt identified

この run は cap 3 の最後の回なので、直さずに進める LOW は、`/sync-docs` か `/pr` が register に書かないと残らない。次の 1 行を足す(R3-1 は文書の直しなので `/sync-docs` がその場で直せる)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R3-1(register の行 519・520・524 の数と「最後の commit」が `d85fc0ef` のあとの状態でない)、R3-2(`reseatProgressUnknownTaskID` と `TaskIDUnknown` の名前、定数の doc の「the marker」、test の名前) | LOW。R3-1 は register を読む人に古い数を見せるだけ。R3-2 は読み手がどちらが語でどちらが旗かを名前から取れない | 最後の回で、どれも挙動を壊さない LOW(今から Go を直すと cap をもう一度上げて `/self-review` から回し直す) | R3-1 は `/sync-docs` の run で直す。R3-2 は次に `reseat_progress.go` の `reseatProgressTask` か不明の行を触るとき | この報告書、`docs/reports/cross-review-triage-reseat-leader-progress.md` |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 2 件)
- Follow-ups:
  - R3-1 は `/sync-docs` の担当で、Go も `leader.md` も触らないので、この run の結果で pipeline の再実行は要らない。
  - R3-2 は名前の直しで、挙動は変わらない。直すなら定数の改名と test の名前だけで、cap を上げた pipeline がもう 1 回回る。直さずに上の 1 行を register に書いて進めてもよい。
  - `/sync-docs` への引き継ぎ(指摘ではない): この直しが変えた 2 点(gate の項目が PR の head を比べること、「座席ごとの最後の TASK」の行に `不明 (記録が切れている、<時刻>)` の形があること)を、節の中身を列挙する文書(`README.md:303`、`AGENTS.md:92`、`docs/specs/2026-11-12-org-reseat.md`、`.claude/skills/org/SKILL.md` と `director/SKILL.md` とその mirror)が行ごとの形まで書いているかを、cycle 2 の run が書いた引き継ぎと合わせて見る。`.claude/skills/org/SKILL.md:1836-1850` は節の中身を名前で並べるだけで、行の形は書かない。

## Cycle 2: HEAD `bd4dcb9b`(cap 2 の run。履歴)

この節の「Tech debt identified」と「Recommendation」は、cap 2 の最後の回だった当時のもの。C2-1 から C2-5 の扱いは、cross-review の triage と `d85fc0ef` のあとの register に引き継がれている。

Scope: cycle 1 の self-review のあとに入った `291ac942`(N4・N5 の直し)、`e483b52a`(lint)、`77a45022`(sync-docs)、`ec10ba6f`(cross-review の直し)、`bd4dcb9b`(計画の記録)。`ec10ba6f` は 4 ファイル全部を読んだ(`internal/org/reseat_progress.go` は全文、`leader.md` は座席の確認・計画・gate の 3 項目と手順 8・12・14、「計画助言」「cross-review」の小節、「agmsg の受信箱の運用」の節、`prompts_leader_reseat_test.go` と `reseat_progress_test.go` の差分)。`77a45022` は Go の変更(`internal/cli/org_reseat.go` と test)と、節の中身を列挙している文書の箇所を読んだ。確認に使った読み物: `verbs.go` の `Send`(`sent` の Details を作る 3 か所)、`send_fallback.go` の `details()`、`protocol.Parse`、`truncateForDetails`、`docs/tech-debt/README.md` の行 519〜521、cross-review の triage report。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれの詳細(`/verify`、`/test`、`/sync-docs` の担当。ただし `77a45022` が書いた箇所が最終の挙動を述べているかは、依頼により読んだ)。

読んで問題なしとしたもの:

- `sent` の Details の読み取り(`reseatProgressSentTask`)。Send が前に付ける語は `raw=true `、`via=screen herdr_state=<state> `、`submit_unconfirmed=true ` の順で、`truncateForDetails` が本文を 200 byte で切る。読みは `=` を含み改行を含まない先頭の語を落とし、`TYPE: <値>` で始まる本文を `protocol.Parse` に渡す。TYPE と TASK_ID は先頭 2 行なので 200 byte の切れ目より前にある(`TASK_ID:` の次の行に `PLAN:` を書く手順 5 の決まりとも合う)。raw で TYPE が先頭行にない文、TYPE が TASK でない文、dry run、leader 宛て、ほかの org は数えない。Send の `isTask`(`parseErr == nil && Type == TASK`)と同じ判定で、`--raw` の TASK も数える。
- 今の life の境目。Tasks は最後の本物の `disbanded` の次のイベントから(添字)、Results は `disbanded` の時刻より後に上がった件(時刻の比較。同じ秒は前の life 側)。同じ規則を Inbox も使い、時刻が読めない件は注つきで残る。
- 空の規則。`reseatProgressSection` は Tasks の行と RESULT の行を `add` で書くので、TASK か RESULT しかない org でも節ができる。ほかの org の RESULT と TASK、dry run の TASK、TASK でない typed message は節を作らない(`TestReseatProgress_AnOrgWithNoProgressHasNoSection` がこれらを固定する)。
- 安全。新しい行の値は `value()` か `reseatProgressText` を通り、制御文字と U+2028・U+2029 が置き換わる。行頭は `- ` かラベルで、囲みの外に出る行はない。QUESTION の `PLAN:` の値は `strings.Fields` で空白を 1 つにまとめる。
- 規則の新しい分岐が読む値は、すべて節が出す。計画の項目の QUESTION の `PLAN <パス> sha256:<digest>`、座席の項目の「座席ごとの最後の TASK」の行、gate の項目の「上げた RESULT」の `HEAD`(手順 14 の RESULT のヘッダ `HEAD:` は gate の出力の HEAD で、gate の行の head と同じ値)。PR の状態は `gh pr list --json number,state,url` が出す。
- 計画助言の判定は「計画助言」の小節と合う。per-plan は手順 2 の前に済み、要約は QUESTION の `ADVISORY:` の行に、split は `## Design decisions` に書く。`plan_advisory` が off のときは記録を残さないので、どちらの印もない場合に「小節のとおり自分で決める」は、off なら何もしない形になる。

### Findings

この cycle の指摘は C2-1 から振る(cycle 1 の M・L・N と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | consistency | 同じ `leader.md` の中で、leader が送った TASK が agmsg の履歴に入るかについて、2 か所が逆を言う。直しの座席の項目は「送った TASK は台帳に残り、agmsg の履歴には残りません」と書き、テストは座席の項目に「あなたが送った TASK も 50 件に入る」が戻らないことを見る。一方「agmsg の受信箱の運用」の節は今も「`history.sh` は既読の印を変えません。あなたが送った TASK も 50 件に入るので、足りなければ数を増やします」と書き、別のテストがその文を固定する。`Send` は herdr の `send-text` で pane に打つだけで agmsg には何も投げない(cross-review の triage が `verbs.go` で確認した事実で、`Send` の呼び出しにも agmsg はない)ので、直った側が正しく、残った側が誤り。cross-review の finding 1 は報告された箇所だけを直し、同じ誤りを持つ兄弟の文が残った | `leader.md:182-184`(新しい文)、`leader.md:1405-1406`(誤りが残る文)、`prompts_leader_inbox_test.go:107`(誤った文を固定)、`prompts_leader_reseat_test.go:325`(座席の項目だけで禁止) | 1406 の「あなたが送った TASK も 50 件に入るので、」を落として「足りなければ数を増やします」だけにし、`prompts_leader_inbox_test.go:107` の固定を直す。`prompts_leader_reseat_test.go:325` の禁止語を、この項目に限らず `leader.md` 全体にかける |
| C2-2 | LOW | maintainability | 入力待ちで RESULT がない分岐(「その TASK に答えないまま入力待ちです」)の 2 つの出口が、implementer のスライスの TASK だけを前提に書かれている。(a)「branch の commit の行がその TASK のスライスをもう済んだと示していれば、座席は空いた」は、reviewer のレビューの TASK には当たらない: スライスの commit はレビューの前にあるので、返事のないレビューの TASK が、commit があることを理由に「済んだ」と読めてしまい、レビューを飛ばして次のスライスへ進める。(b)「同じ TASK_ID でもう一度送ります(中身は計画のそのスライスから書きます)」は、`gate-<n>` の TASK(中身は手順 12 の gate のコマンド)と `xreview-…` の TASK(中身は「cross-review」の小節の形)には書き方の出どころが違う。同じ項目の 1 つ前の分岐は、TASK_ID の頭で手順 12・小節・手順 8 に分けており、この分岐だけ分けていない(cycle 1 の N5 と同じ型)。手順 8 と 12 は、通らなかった TASK を新しい TASK_ID で送り直すと書くので、「同じ TASK_ID」を選んだ理由(RESULT を TASK_ID で突き合わせるため)も項目に書かれていない。到達は、返事を返さないまま入力待ちになった reviewer(利用上限や API エラーのあとの再開など)に限られ、まれ | `leader.md:191-195`(分岐)、`:189-191`(前の分岐の TASK_ID 別の行き先)、手順 8 の `新しい TASK_ID で、CHECKS: の行を付けたレビューの TASK を送り直す`、手順 12 の `新しい TASK_ID で送り直す`、`prompts_leader_reseat_test.go` の `そうでなければ、その TASK を同じ TASK_ID でもう一度送ります` | 出口を TASK_ID の頭で分ける 1 文を足す(implementer のスライスの TASK は今の文のとおり、`gate-` は手順 12 の gate の TASK を同じ TASK_ID で、`xreview-` は小節の TASK を同じ TASK_ID で、reviewer のレビューの TASK は commit の行を根拠にせず同じ TASK_ID で)。「同じ TASK_ID」にする理由を 1 句添える |
| C2-3 | LOW | readability | 新しい「座席ごとの最後の TASK」の行は `- <座席>: <TASK_ID>(<時刻>)` で、規則が比べる値の TASK_ID に時刻の括弧が空白なしでつく。承認の行は、規則が digest を空白で区切った 1 語として比べるので `sha256:<digest> (承認 <時刻>)` に直した(cycle 1 の L3)。TASK_ID は座席の項目が RESULT の TASK_ID と突き合わせる値なのに、同じ型が残った。register の行 519 の (d)(gate の行の括弧)が「規則が比べるのは head と base だけで、どちらも空白で区切れるので、判断はこれに依らない」を理由にしており、この行にはその理由が当たらない | `reseat_progress.go:311`(`"- %s: %s(%s)"`)、`reseat_progress_test.go` の `"- implementer: slice-1(2026-11-15T01:20:00Z)"`(固定)、`docs/tech-debt/README.md` 行 519 の (d) | `"- %s: %s (%s)"` にして、テストが空白で切った語を見る(`TestReseatProgress_TheApprovalLineGivesTheDigestAsOneWord` と同じ形)。直さないなら、行 519 の (d) に「座席ごとの最後の TASK の行(`reseat_progress.go`)にも同じ括弧があり、こちらは規則が比べる値」を足し、理由の文を直す |
| C2-4 | LOW | test | `TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites` は、doc コメントが「`reseatProgressSentTask` が落とす語を、Send が本文の前に付ける語と合わせておく」と言うが、語は `"raw=true "`、`"via=screen herdr_state=idle "`、`"submit_unconfirmed=true "` の文字列の写しで、Send の側(`verbs.go:479` の `"raw=true "`、`:666` の `"submit_unconfirmed=true "`)とは結びついていない。両側に共有の定数がなく、Send が `=` のない語や空白を含む値の語を足すと、読みは語の途中で止まって `protocol.Parse` が失敗し、その TASK は節から黙って落ちる(座席の項目は「その座席の行がなければ TASK を受けていない」と読み、実際に届いている座席へ TASK を送り直す)。落ちる向きは前の挙動(行なし)と同じなので害は小さいが、doc コメントの約束は守られていない。同じ test の隣の `TestReseatProgress_TheLastTaskPerSeatComesFromTheSentEvents` は `(&sendScreenPath{…}).details()` を使い、画面の語についてだけ本物の出力に結びついている | `reseat_progress_test.go:731-746`、`verbs.go:479`、`:666`、`send_fallback.go:60-65` | Send の 2 つの語を定数(`sendDetailsRawPrefix`、`sendDetailsSubmitUnconfirmed`)にして、test と `reseatProgressSentTask` の doc がそれを使う。または test が `Org.Send` の fake herdr を通して `sent` を作り、その Details を読む |
| C2-5 | LOW | docs-in-code | `ralph org reseat --help` の節の説明は、中身を閉じた列挙で書く(「最新の plan approval、最後の gate の記録、open と acked の受信箱の件、branch の commit、この reseat が leader のあとに起動する座席、前の reseat」)。`ec10ba6f` のあとで節は 2 つの行(座席ごとの最後の TASK、今の life で上げた RESULT)と QUESTION の `PLAN:` を持つが、`--help` とその test の固定は 6 項目のままで、`/sync-docs` が `77a45022` で書いた文書の列挙も同じ 6 項目。`--help` は Go の文字列で test が固定するので、文書の直しとは別に、次の fix commit の対象になる | `internal/cli/org_reseat.go:104-109`、`internal/cli/org_reseat_test.go:873-875`。同じ列挙を持つ文書(`/sync-docs` への引き継ぎを参照) | 2 項目を足す(「the last TASK sent to each seat, the RESULT items it raised」)。文書は `/sync-docs` の cycle 2 で足す |

### cycle 1 の指摘と cross-review の確認

| ID | 内容 | 状態 | 根拠 |
| --- | --- | --- | --- |
| N4 | gate の例外が座席の確認の入力待ちの分岐に掛かっていない | 閉じた | 座席の項目の頭に gate の例外がある(`leader.md:165-168`: 「gate の項目に当たるとき…はこの確認の残りはせず、どの座席にも TASK を送らず」)。作業中の分岐の括弧も残る |
| N5 | 入力待ちの分岐の「手順 8 のとおり」が implementer と reviewer のレビューにしか合わない | 閉じた(RESULT を扱う側) | `leader.md:190-191`: `gate-` は手順 12、`xreview-` は「cross-review」の小節、それ以外は手順 8。TASK を出す側の同じ型は C2-2 |
| cross-review 1 | leader が送った TASK を agmsg の履歴から探している | 直った。同じ誤りを持つ兄弟の文が 1 か所残る(C2-1) | `sent` の Details から座席ごとの最後の TASK を読み、TASK_ID で RESULT・BLOCKED を突き合わせる。到達する形と境目は上の「問題なしとしたもの」 |
| cross-review 2 | PR を作ったあと RESULT の前に止まった leader が、手順 14 の RESULT を上げない | 直った | 「上げた RESULT」の行(今の life、全状態、新しい 5 件)と、gate の項目の RESULT の確認。RESULT の `HEAD` は gate の行の head と同じ値(手順 14 のヘッダ)。5 件の上限は、RESULT を上げるのが手順 14 だけなので、実際には当たらない |
| cross-review 3 | `--state all` が merge なしで閉じた PR も返す | 直った | gate の項目が `OPEN`、`MERGED`、`CLOSED` を分け、`CLOSED` だけなら BLOCKED。PR なし、OPEN か MERGED、CLOSED だけの 3 つで次の手順がそろっている |
| cross-review 4 | 承認なしの計画で、計画助言が済んだかが分からない | 直った | QUESTION の行の `PLAN`(パスと digest)で判定し、split は `## Design decisions` の要約で判定する。どちらの印もなければ「計画助言」の小節のとおりに自分で決める |

### Positive notes

- `reseatProgressSentTask` の doc コメントが、Send の前の語、200 byte の切れ目、残るのが TYPE と TASK_ID の先頭 2 行であること、raw の扱いを順に書いている。コードは doc のとおりに読める。
- `Tasks` を map で集めて座席の id で並べ直すので、節の行の順が実行ごとに変わらない。
- テストが、各列挙値(prefix、切れた長い本文、dry run、ほかの org、leader 宛て、別の TYPE が後から来る)を 1 件ずつ持ち、期待を行の列で比べる。RESULT の 5 件の上限は、5 件ちょうどと 8 件の 2 つを見る。
- 規則のテストは、新しい分岐の文の順と、古い文(「で PR があるかを見ます」など)が消えたことを固定している。手順 14 の RESULT の文を引いて、gate の項目が上げる RESULT の出どころを固定する点も良い。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

## Tech debt identified

cycle 2 は `max_pipeline_cycles` の 2 回目で、最後の回。C2-1 から C2-5 を直さず進めるなら、次の 1 行を register に足す(`/sync-docs` か `/pr` の担当)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-1(`leader.md:1406` の「送った TASK も 50 件に入る」が座席の項目と逆)、C2-2(RESULT なしの分岐の出口が implementer のスライス前提)、C2-3(座席ごとの最後の TASK の行の括弧)、C2-4(prefix の test が Send に結びつかない)、C2-5(`--help` の列挙が 2 項目足りない) | LOW。C2-1 は leader が誤った一文を読むだけで、座席の項目の指示は正しい。C2-2 は、返事のない reviewer の TASK という到達のまれな形。C2-3 は行を空白で切る読み手にだけ。C2-4 は落ちる向きが前の挙動と同じ。C2-5 は表示のみ | 最後の回で、どれも挙動を壊さない LOW(今から直すと cap を上げて pipeline が `/self-review` から回り直す) | 次に `leader.md` の座席の確認か「agmsg の受信箱の運用」の節、`reseat_progress.go` の Tasks の行、`sent` の Details を作る `Send` を触るとき | この報告書、`docs/reports/cross-review-triage-reseat-leader-progress.md` |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 5 件)
- Follow-ups:
  - C2-1 と C2-3 は 1 行ずつで、テストの固定を直すだけで済む。C2-1 は兄弟の誤りが同じ PR の前の直しと矛盾している点で、5 件のうち先に直す価値がある。C2-2 は 1 文。直せば pipeline は最後の回を超えるので、cap を上げて `/self-review` から回し直すか、上の 1 行を register に書いて進む。
  - **`/sync-docs` への引き継ぎ**(指摘ではない): `ec10ba6f` が節に足した 2 行(座席ごとの最後の TASK、上げた RESULT)と QUESTION の `PLAN` を、`77a45022` が書いた中身の列挙のうち次の箇所が持たない: `README.md:303`、`AGENTS.md:92`(`reseat_progress.go` の説明)、`docs/specs/2026-11-12-org-reseat.md:80`(FR-9 の列挙)、`docs/specs/2026-08-01-org-runtime.md:125`、`.claude/skills/org/SKILL.md` と `.claude/skills/director/SKILL.md`(とその `.agents/` と `templates/base/` の mirror 6 つ)。`77a45022` の時点では正しく、cross-review の直しがあとから列挙を古くした。
  - **register の行**(`/sync-docs`): 行 520 の (a)(`--state all` が閉じた PR を数える)、(b)(PR のあと RESULT の前に止まる)、(c)(計画助言が済んだか分からない)は `ec10ba6f` が直したので、行から外すか「`ec10ba6f` で直した」と書く。(d)(live の確認)は残る。行 519 の (a) の「86 行」は `reseatProgressSection` が 103 行になって数が違う(L5 の中身は今回の再提起ではなく、数だけの訂正)。行 519 の (d) は C2-3 の新しい箇所を足す。

## Cycle 1(履歴): 3 回の run、すべて直して閉じた

HEAD `8ac1487c`、`d4d1ca93`、`a2494fab` の 3 回。報告書の全文は `git show f57e37f4:docs/reports/self-review-2026-11-15-reseat-leader-progress.md`。

| Run | HEAD | 指摘 | 直した commit |
| --- | --- | --- | --- |
| 1 | `8ac1487c` | MEDIUM 1(M1: leader だけ、または implementer を動かしたままの立て直しが規則のどの項目にも当たらない)、LOW 7(L1: `git log` に `--no-show-signature` がない、L2: 「最初の N 件」の向き、L3: digest の後ろの括弧、L4: Go のコメントが `docs/plans/active/` を指す、L5: 関数の長さ・`inboxSummary` との二重・`NewReplacer`、L6: 計画の項目の行き先、L7: gate の項目が座席を待たない) | `71ddcb30`(L5 と L3 の gate の行は register 行 519 へ) |
| 2 | `d4d1ca93` | MEDIUM 1(N1: 座席の確認の「待つ」に出口がない)、LOW 2(N2: 項目の語の範囲、N3: コメントの折り返し) | `2a20a1f2` |
| 3 | `a2494fab` | MEDIUM 1(N4: 入力待ちの分岐に gate の例外が掛からない)、LOW 1(N5: 「手順 8 のとおり」が TASK_ID の頭で分かれない) | `291ac942` |

cycle 1 の cross-review(4 件の ACTION_REQUIRED)は `ec10ba6f` で直り、上の表で確かめた。
