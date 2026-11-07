# Self-review report: org-seat-resume

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの cycle 2 の再実行(cycle 2 after R4)。cycle 2 の R3 のあとの self-review(`0720102d`)の R3-1、R3-2 (a)、C2-3 を受けた直し R4 `6ee7c1dd` を見たもの。上限 2 の最後の回で、cycle は増やしていない
- Scope: R4 `6ee7c1dd` の 4 ファイル(`internal/org/resume_screens.go` の `resumeCodexFooterLine` とその doc、画面の型の手順 2、`internal/org/resume_screens_test.go` の edge 2 つ、`internal/org/verbs.go` の `Send` の doc、`internal/org/pane_lock.go` の doc)。diff 全体(`git diff 11239c8d...HEAD`、91 ファイル、+8004 / -191)は見直しておらず、前回までの指摘の今の状態だけを HEAD で確かめた。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。

## Evidence reviewed

- R4 の差分(`git show 6ee7c1dd`、4 ファイル、+37 / -13)。`codexFooterStart`(`internal/org/resume_screens.go:391-414`)は R4 で変わっておらず、正規表現だけが変わったことを確かめた。
- 正規表現は `^\? for shortcuts|^.+ · Context \d+% (?:used|left)$|^(?:.+ · )?\d+% context left(?: · \? for shortcuts)?$`。Go の `$` は `m` フラグなしでは文字列の終わりにだけ合い、`codexFooterStart` は行を `strings.TrimSpace` してから渡すので、2 つ目と 3 つ目の選択肢は行の頭から終わりまでに掛かる。1 つ目の選択肢には `$` がない。
- 合う行と合わない行を、`python3 -I` の `re.search`(Go の `MatchString` と同じく文字列のどこからでも探す。`re.ASCII` で Go の `\d` に合わせた)で机上の確かめの代わりに試した。テストでも lint でもなく、使った構文(`^`、`$`、選択、非捕獲の括弧、`\d`)は 1 行の文字列については RE2 と同じ意味になる。結果:
  - `internal/org/testdata/resume/codex-*.txt` の 6 つのうち footer を持つ 4 つ(`codex-usage-limit.txt`、`codex-input-draft-after-empty-line.txt`、`codex-input-typed.txt`、`codex-network-waiting.txt`)の footer の 8 行はすべて合う。`codex-input-draft-no-footer.txt` と `codex-hooks-review.txt`(`enter confirm · esc skip`)の行はどれも合わない。
  - 証拠 `docs/evidence/org-seat-resume-probe-2026-11-07.md` の 153 行(`GPT-6-Astra · xhigh · Context 0% used`)と 154 行(`? for shortcuts                           ⚠ 11 · f2`)は合う。ほかの行は合わない。正規表現の `·` は U+00B7 で、証拠の行と同じ文字である。
  - 合う: `90% context left`、`100% context left · ? for shortcuts`、`fix it · 3% context left`、`a · Context 5% used`、`? for shortcuts please`。合わない: `please keep Context 5% used in mind`、`see 3% context left`、`Context 5% used`(前に欄がない)、`GPT-6-Astra · xhigh · Context 0% used · ~/repo`、`100% context left · ? for shortcuts · ⚠ 11`、`90% context left  ? for shortcuts`。
- 古い文の残り: `was not idle/done after the lock`、`after the lock of its pane was taken` は `internal/org/*.go` に 0 件。
- 証拠の 5 節(codex のバイナリの文字列)を `grep -n -i 'context\|shortcut\|left\|footer'` で見た。`left` は 38 行(claude の続行の文)だけで、`Context <n>% left` も `<n>% context left` も載っていない。
- 記録: 計画の進み具合(`docs/plans/active/2026-11-07-org-seat-resume.md:152`。R3 のあとの self-review と R4 の注はまだない)、`docs/tech-debt/README.md:444-455`(R4 は変えていない)、`/director` skill の 4 つの写しの 1043〜1044 行。

## Cycle 2 と R3 のあとの指摘の今の状態

| ID | 今の状態 | 根拠 |
| --- | --- | --- |
| C2-1 | 開いたまま。`/sync-docs` で扱う(計画の進み具合の注) | `docs/tech-debt/README.md:454`、`:444-447`;`.claude/skills/director/SKILL.md:1043-1044`(ほかの 3 つの写しも同じ行) |
| C2-2 | 直った(R3) | `internal/org/verbs.go:480-485`、`:346-357`、`:784`(`capMSToContext`) |
| C2-3 | 直った(R4)。`pane_lock.go` の doc は、`Send` がロックを待ったあと idle/done を待ち直すこと、手番が続くあいだは打たず、手番が終わったあとは打ちうること、前の持ち主の submit が確かめられないままロックが外れると herdr がまだ idle/done と報告し待ち直しが通ることを書く。勧めの後半(`Send` の doc にも同じ 1 文を足す)は入っていないが、`Send` の doc は「打たない」とは言っておらず、誤った主張はない | `internal/org/pane_lock.go:17-24`;`internal/org/verbs.go:298-312` |
| C2-4 | 直った(R3) | `internal/org/resume_screens.go:391-414` |
| C2-5 | 開いたまま。`/sync-docs` で扱う | 計画の D8 の R2 の改訂(`:72`)に `⎿` の取りこぼしの 1 文がない |
| C2-6 | 開いたまま。`/sync-docs` で扱う | 下の「Cycle 1 の指摘の今の状態」の L3〜L9 |
| R3-1 | 直った(R4)。前のほうの段落は「its error is described at the end of this comment with the error of the first wait」と後ろの段落を指し、エラーの文を書くのは後ろの段落の 1 か所になった | `internal/org/verbs.go:311-312`、`:346-357` |
| R3-2 | (a) 直った(R4)。状態行の 2 つの選択肢は行の頭から終わりまでに掛かり、`Context 5% used` を含むだけの下書きの行と `3% context left` で終わるだけの下書きの行は入力になる(edge 2 つ)。(b) 開いたまま、R4 で 1 つ増えた。(c) 開いたまま、R4 で当たる形が 1 つ増えた。どちらも下の Findings の R3-2 | `internal/org/resume_screens.go:246-261`;`internal/org/resume_screens_test.go:428-439` |

## Cycle 1 の指摘の今の状態

R4 はこれらのファイルを変えていない。前回から変わらない。

| ID | 今の状態 | 根拠 |
| --- | --- | --- |
| L1 | 直った(R1)。register の行は開いたまま(C2-1) | `internal/org/verbs.go:459-485`;`docs/tech-debt/README.md:454` |
| L2 | 直った(`a4f60ff9`) | `internal/org/resume.go:54-58` |
| L3 | 開いたまま、register にない | `internal/org/resume.go:376-381` |
| L4 | 開いたまま、register にない | `internal/cli/org_resume.go:210-212` |
| L5 | 開いたまま、register にない | `internal/org/prompts/implementer.md:101-105` |
| L6 | 開いたまま、register にない | `internal/org/lockfile.go:154-180` |
| L7 | 開いたまま、register にない | `internal/cli/org_resume.go:34`、`:230`;`internal/org/resume.go:83` |
| L8 | 開いたまま、register にない | `internal/org/director_cmd.go:327` |
| L9 | 開いたまま(この PR より前の helper)、register にない | `internal/org/inbox.go:180-194` |
| L10 | register の新しい行に移った | `docs/tech-debt/README.md:448` |

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1(前回のまま)。R1 と R2 のあとの register と skill が、直した挙動を古いまま書いている。(a) cycle 1 で足した `Send` の行(L1)は、R1 がその「Trigger to pay down」のとおりに直したのに開いたままで、「`Send` は手番を始めた座席に打つ」と言い続ける。(b) `/director` skill の 4 つの写しは「2 本目は 1 本目で手番を始めた leader に打つ」と書く。R1 のあとは、2 本目はロックの中で待ち直す。(c) 止まった文の行の解消の注は、返事の先頭が一覧の文で始まるときにまだ止まったと判定することを書いていない。(d) 解消にした 2 行の Related の欄は triage の `(ACTION_REQUIRED #3)`、`(ACTION_REQUIRED #4)` を cycle なしで指す | `docs/tech-debt/README.md:454`、`:444-447`;`.claude/skills/director/SKILL.md:1043-1044`(ほかの 3 つの写しも同じ行);`internal/org/resume_screens.go:551`(`findResumeStop`) | (a) 454 行を `<!-- RESOLVED ... -->` と取り消し線で閉じ、R1 `0aa8e274` と R3 `3714f6d5` を書く。R4 で `pane_lock.go` の doc が残る 2 つの場合を書くようになったので、解消の注はその doc を指せばよい。(b) skill の文を「2 本目はロックを取ったあと idle/done を待ち直すので、1 本目で始まった手番が続くあいだは打たない」に直す。(c) 解消の注に残る場合を足す。(d) Related を「cycle 1 の ACTION_REQUIRED #3(commit dd008093)」の形にする。どれも `/sync-docs` の回で済む |
| LOW | maintainability | C2-5(前回のまま)。R2 は edge「only the first ⎿ line under a prompt starts a block, the next ones join it」の期待値を Stopped から Unknown に変えた。prompt の下の最初の `⎿` の行がほかの行のとき、その下の上限の行は続ける対象にならない。打たない側に倒れるが、計画の D8 の R2 の改訂と、register の利用上限の行は、この取りこぼしを書いていない(`.claude/rules/ralph/testing.md` は計画と review の報告に書くよう求める) | `internal/org/resume_screens_test.go:300`;計画の D8 の 2 つ目の改訂(`:72`);`docs/tech-debt/README.md:451` | D8 の R2 の改訂に「prompt の下の最初の `⎿` の行でない上限の行は照合しない(打たない側)」の 1 文を足し、利用上限の行の Impact にも同じことを書く。この報告にはここで書いた |
| LOW | maintainability | C2-6(前回のまま)。cycle 1 の L3〜L9 の 7 件は、HEAD でも直っておらず、register のどの行にもない。上限 2 の最後の回で、`/pr` が計画を archive すると、ほかにこれらを持つものがなくなる | 上の「Cycle 1 の指摘の今の状態」の L3〜L9 | 7 件を 1 行にまとめて register に足す(下の Tech debt identified の 1 行目) |
| LOW | null-safety | R3-2(前回の (b) と (c) が残る。(a) は R4 で直った)。(b) コメントは `Context <n>% left` と `<n>% context left` を「codex-cli 0.165.0 binary strings」にあるとするが、証拠の 5 節はこの 2 つを載せていない。R4 はさらに 3 つ目の選択肢の後ろに ` · ? for shortcuts` を許す形を足した。これも証拠になく、R3 の edge「codex footer with the context left forms is a footer」の合成の行 `100% context left · ? for shortcuts` が R4 の後も footer であるために要る(この後ろがなければ、R4 の 3 つ目の選択肢はこの行に合わない)。証拠にない形が 3 つ、合成の fixture に合わせて正規表現に入っている。(c) codex が footer の形を変えると、codex の座席の画面はどれも「input box has text」で unknown になる。R4 の `$` で、当たる形が 1 つ増えた。状態行の数字の後ろに別の欄が続く行(`GPT-6-Astra · xhigh · Context 0% used · ~/repo`)は、R3 の正規表現には合い、R4 の正規表現には合わない。打たない側に倒れるが、理由の文は入力欄を指し、footer の形が原因であることは画面の結果からは分からない | `internal/org/resume_screens.go:246-251`(由来のコメント)、`:261`(正規表現)、`:130`(`resumeReasonInputHasText`);`internal/org/resume_screens_test.go:411-415`;`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 5 節;上の Evidence reviewed の試した行 | (b) 3 つの形を次の probe で証拠の 5 節に足すか、コメントに「`% left` の 2 つの形と ` · ? for shortcuts` が後ろに付く形は証拠に載せていない。最後の形は合成の edge のためにある」と書く。(c) `resumeReasonInputHasText` の doc か register の行に「codex の footer の形が変わると(状態行の数字の後ろに欄が足されるときを含む)、この理由で全座席が飛ばされる。そのときは `resumeCodexFooterLine` を直す」を書く。どちらも下の Tech debt identified の 2 行目に入れた |
| LOW | readability | R4-1(新規)。R4 が足した正規表現の説明が、2 か所で正規表現と食い違う。(a) 画面の型の手順 2 は「resumeCodexFooterLine, which matches the whole line」、正規表現のコメントは「Each of the three alternatives matches a whole line」と書くが、1 つ目の選択肢 `^\? for shortcuts` には `$` がなく、同じコメントの項目 1 が「what follows it on the line is not checked」と書く。1 つのコメントの中で先頭の文と項目 1 が食い違う。このため、`? for shortcuts` で始まる下書きの行(`? for shortcuts の一覧を出して` のような行)は今も footer の行と読まれる。下書きが footer と読まれる道は、これと、行が数字だけ(`90% context left`)か ` · ` の後ろに数字で終わる行に限られ、どれも人が打つ下書きとしてはまれである。(b) 手順 2 は 3 つ目の選択肢を「a line that ends in `<n>% context left`」と書く。R4 が足した edge「codex draft that ends in the context left figure is input」の行 `see 3% context left` はこの説明に当たるのに footer ではなく、R3 の edge の行 `100% context left · ? for shortcuts` は数字で終わらないのに footer である。正しい説明は正規表現のコメントの項目 1〜3 で、手順 2 と先頭の文がそれと違う | `internal/org/resume_screens.go:48`、`:50-51`(手順 2);`:253-260`(正規表現のコメント);`:261`;`internal/org/resume_screens_test.go:413`、`:436-438` | doc だけの 3 行の直しで済む。手順 2 の「a line that ends in `<n>% context left`」を「a line that is `<n>% context left`, alone or as the last field after ` · `, optionally followed by ` · ? for shortcuts`」に、「which matches the whole line」と「Each of the three alternatives matches a whole line」を「anchored at the start of the line, and the two context alternatives also at its end」に直す。1 つ目の選択肢も行の終わりまで縛るなら、証拠の `⚠ 11 · f2` を通すために `^\? for shortcuts(?:\s{2,}\S.*)?$` のように 2 つ以上の空白のあとに限る形にし、edge を 1 つ足す。直さないなら下の Tech debt identified の 2 行目に入れた |

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5(前回から残る C2-1、C2-5、C2-6、R3-2 の (b) と (c)、R4 で生じた R4-1)。C2-3、R3-1、R3-2 (a) は R4 で直った。

## Positive notes

- R4 の正規表現は、3 つの選択肢のすべてが行頭に掛かり、状態行の 2 つは行の終わりにも掛かる。証拠の footer の 2 行と、fixture の footer の 8 行はすべて合い、footer のない画面と選択画面の行はどれも合わない。R4 で新しく footer と読まれるようになった行はなく、受け入れる範囲は狭くなる向きにだけ変わった。打つ側に倒れる新しい道はない。
- R4 の edge 2 つは、状態行の 2 つの形のそれぞれについて、数字を含むだけの下書きが入力になることを固定する。R3 の edge「press ? for shortcuts」とあわせて、3 つの選択肢のどれか 1 つを R3 の錨のない形に戻すと、その選択肢の edge の下書きが footer と読まれて落ちる(1 つずつ戻して `re.search` で確かめた)。ただし状態行の選択肢の `$` だけを外すか、前の欄を任意(`(?:.+ · )?`)にするだけなら、edge の行の読まれ方はどれも変わらない(状態行の選択肢は 2 つの錨の両方で守られている)。
- C2-3 の直しは、前回の勧めの 2 つの場合(手番が終わったあとは打ちうること、submit が確かめられないまま外れたロック)をどちらも書いた。「refuses instead」の言い過ぎは消えた。
- R3-1 の直しで、ロックの中の待ちのエラーの文を書くのは `Send` の doc の最後の段落の 1 か所になった。古い文の literal は `internal/org/*.go` に残っていない。
- R4 はコードの道を変えていない。変わったのは正規表現と doc comment だけで、ロックの取り方、解放、何も打たない失敗の道は R3 のときのままである。

## Known gaps

- 正規表現は Python の `re` で試した。Go の RE2 では走らせていない(テストの実行は `/test` の範囲)。使った構文は 1 行については同じ意味だが、確かめたのは机上の比べ合わせまでである。
- codex-cli 0.165.0 が footer の状態行の欄を設定で変えられるかは確かめていない。変えられるなら R3-2 (c) は設定だけで起きる。おそらく起きうるが、未確認です。
- 正規表現が実際の codex の footer に合うかは、証拠 4 節の 1 画面でしか確かめていない(前回のまま)。
- `TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing` が stub の都合で「failed; nothing was typed」の文を期待する点は、前回のとおりテストの忠実さの問題で `/test` の範囲とした。

## Tech debt identified

この review では register を直接編集していない。コミットを報告と insight の event だけにする指示のためである。次の 2 行を `docs/tech-debt/README.md` に足すことを勧める。C2-1 は既存の行の直しなので、ここには書かない。前回の 2 行目は C2-3、R3-1、R3-2 を持っていたが、C2-3、R3-1、R3-2 (a) は R4 で直ったので、R3-2 の (b) と (c)、R4-1 に入れ替えた。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| org-seat-resume の cycle 1 の self-review の LOW のうち、直さずに merge するもの(L3〜L9)。(L3) resume のロックの中の見直しで herdr の呼び出しが失敗すると、理由が `the screen changed (...)` になる(`resumeOne`、`internal/org/resume.go`)。(L4) 文が pane に残りうる `failed`(send-text の失敗、Enter の前の確かめの失敗、Enter の失敗)に、pane の読み方の案内が出ない(`printResumeTargetNotes`、`internal/cli/org_resume.go`)。(L5) role prompt 3 つと `agent-messaging.md` は、続行の文で「始まる」行を続行の指示とし、後ろに続く文字の扱いを書いていない。(L6) `heldFileLock` の doc(`internal/org/lockfile.go`)の保持する側の一覧に pane のロックがない。(L7) 読む行数の 60 を CLI の help と案内が文字列で持ち、`org` の側は非公開の `resumeReadLines`。(L8) 起動スクリプトの `text()`(`internal/org/director_cmd.go`)は `RetryEnv` の名前を検査せず、引用符なしで書く。(L9) `PrintableInboxText`(`internal/org/inbox.go`)は Cc だけを escape し、Cf(双方向の上書き)を通す | LOW。どれも打つ道の安全には関わらない。L5 は `ralph org send --raw` を打てる者が続行の文に指示をつなげうる点、L4 は残った文を人が見落としうる点が大きい | 上限 2 の最後の回で、cross-review cycle 1 の ACTION_REQUIRED 4 件と cycle 2 の self-review の指摘を優先した | `resume.go` の見直し、`org_resume.go` の結果の行、role prompt の続行の段落、`lockfile.go` の doc、`director_cmd.go` の `text()`、`PrintableInboxText` のどれかを次に変えるとき。直し方は cycle 1 の self-review の L3〜L9 の Recommendation のとおり | `docs/reports/self-review-2026-11-07-org-seat-resume.md` の git 履歴の cycle 1 版(commit 65eea701、L3〜L9);plan `2026-11-07-org-seat-resume` |
| codex の footer の判定(`resumeCodexFooterLine`、`internal/org/resume_screens.go`。org-seat-resume の cycle 2 の self-review の R3-2 の (b) と (c)、R4-1)。(R3-2 b) `Context <n>% left`、`<n>% context left`、後ろに ` · ? for shortcuts` が付く形の 3 つは証拠に載っておらず、最後の形は合成の edge に合わせてある。(R3-2 c) codex が footer の形を変えると(状態行の数字の後ろに欄が足されるときを含む)、codex の全座席が「input box has text」で飛ばされ、原因が footer の形であることは理由の文から分からない。(R4-1) 画面の型の手順 2 と正規表現のコメントの先頭の文は「行全体に合う」「`<n>% context left` で終わる行」と書くが、1 つ目の選択肢は行頭だけに掛かり、3 つ目は数字で終わるだけの行には合わない | LOW。打つ道に関わるのは、`? for shortcuts` で始まる下書きの行が footer と読まれる場合だけで、まれである。(c) は打たない側に倒れる | 上限 2 の最後の回で、直すと `/self-review` からの回し直しが要る | `resumeCodexFooterLine` か画面の型の doc を次に変えるとき、codex の release が footer を変えたとき、または codex の footer の probe を取り直すとき。直し方は R3-2 と R4-1 の Recommendation のとおり | `docs/reports/self-review-2026-11-07-org-seat-resume.md`(cycle 2 after R4 の版、R3-2、R4-1。R3-2 のもとの文は commit 0720102d の版) |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。LOW は 5 件(C2-1、C2-5、C2-6、R3-2 の (b) と (c)、R4-1)。R4 は C2-3、R3-1、R3-2 (a) を直し、正規表現の変更は footer と読む行を狭める向きにだけ効く。証拠の footer はすべて今も合い、新しく footer と読まれる行はない。
- Follow-ups: C2-1、C2-5、C2-6 は記録と skill の文の直しで、`/sync-docs` の回で扱う予定(計画の進み具合の注)。R4-1 は Go の doc comment の 3 行の直しで、`/sync-docs` は前にも Go のコメントを直している(`a42b9845`、`a4f60ff9`)ので、そこで直すか、上の 2 行目に入れる。R3-2 の (b) と (c) は 2 行目に入れる。計画の進み具合には R3 のあとの self-review と R4 の注がまだない。注を書くときは、前回 C2-3 が両方の一覧から落ちたので、`/sync-docs` に回すもの(C2-1、C2-5、C2-6、R4-1)と register に入れるもの(C2-6 の L3〜L9、R3-2 (b)(c)、直さないときの R4-1)を ID で並べる。
