# Self-review report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(D1〜D6、AC1〜AC6、Progress checklist の S1〜S5 と直し F1〜F4 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 2 / 2。cross-review の ACTION_REQUIRED 1 件を F4 ffafc9d2 で直したあとの再 run。cycle 1 の 2 回目は LOW 2 件)
- Cycle: 2 / 2(最後の run。ここで直さなかった LOW は繰り越しになる)
- Scope: `git diff 622cdcfb...HEAD`(HEAD 50fc3faf、23 ファイル、+1987/-101、22 コミット)。中心は F4 ffafc9d2(7 ファイル、+102/-11): `internal/org/prompts/implementer.md` の機械的な確かめの項目、`/org` skill の implementer の箇条書き(4 つの写し)、テスト 3 か所(`prompts_verification_multi_test.go` に 2 本、`prompts_verification_test.go` の期待 2 行)。残りの差分は cycle 1 に読んだ範囲で、F4 が触っていない(`git show ffafc9d2 --stat` に `leader.md`、`reviewer.md`、docs、台帳はない)。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。

## Evidence reviewed

- `git show ffafc9d2` の全差分と、トリアージ `docs/reports/cross-review-triage-org-narrow-verification.md` の指摘 #1(stage 済みの scope 外の資格情報ファイルを、秘密情報の片付けが unstage して消しうる)。直し後の `implementer.md` 全文(ミッションの 15-18 行目の「記録」、48-65 行目の確かめの項目、86-91 行目、109-115 行目、139-155 行目)。
- 指摘 #1 への答えを 3 点に分けて雛形の文で確かめた。(1) 直すのはこのスライスの変更だけ(`implementer.md:53-54`)。(2) 記録した `git status --porcelain` に出ていた変更は stage 済みも含めて書き換えも削除も unstage もしない(`:54-56`)。ファイルを消すのは「このスライスで作った未追跡のファイル」だけ(`:59-60`)。(3) 前からあった変更の秘密情報と、前から stage されていた scope 外のパスは、触らずコミットせずに BLOCKED(`:61-65`)。コミットしない理由は「コミットには stage してあるものがすべて入る」(`:56-57`)で、`git commit` が index 全体を取る挙動と合う。
- skill の写し: 4 つが `cmp` で一致。変更行は `.claude/skills/org/SKILL.md:1336-1338`。雛形の言葉より短く、unstage と値を消す手順は書かず、「そのスライスの変更だけ」「前からあった変更の秘密情報と前から stage されていたパスには触らずコミットせずに BLOCKED」を言う。cycle 1 から同じ要約の程度。
- テスト: `squashSpace` は `strings.Fields` を空文字でつなぐので(`prompts_test.go:139`)、雛形の折り返しは照合に影響しない。`SelfCheckLeavesChangesThatPredateTheSlice` の 9 つの句を雛形の出現順と突き合わせ、順序が一致することを確かめた。古い言い方の「例外は秘密情報だけ」「新しい未追跡のファイル」の不在も照合する。どちらの古い句も F4 の前の雛形にはあったので、F4 を戻せばこのテストは落ちる。skill 側のテストは `want` の 1 文が F4 で足された文で、同じく戻せば落ちる。`markdownItem` は項目の境界を「trim した行が `- ` か番号で始まる」で切る(`prompts_test.go:93-98`)。
- 引用の形: テストの doc の「cross-review cycle 1 #1 of plan 2026-10-27-org-narrow-verification」は、cycle と番号と slug を持つ形で、`prompts_test.go:939` の「cross-review cycle 3 #1 of org-gate」と同じ。
- `git diff ffafc9d2^ ffafc9d2 --check` は rc 0。`wc -l`: `prompts_verification_test.go` 799 行、`prompts_verification_multi_test.go` 165 行。
- 隣の面: `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`(`templates/base/` 側と `cmp` で一致)、`.claude/skills/implement/SKILL.md:42-43`。`docs/tech-debt/README.md` に `unstage` はない(grep)。
- 確かめていない: テストと静的検査の実行、変異の再現(実装側の記録では変異 7 つがそれぞれ落ちる)、座席の実機。

## 前回までの指摘の扱い

cycle 1 の 1 回目の 8 件と、2 回目の LOW 2 件。F4 は `implementer.md` の 53-65 行目、skill の implementer の箇条書き、テストの文字列しか触っていないので、残りは前回の確認がそのまま当たる。下は F4 の周辺で読み直したもの。

| 前回の所見 | 扱い | 理由 |
| --- | --- | --- |
| 1 回目 #1 スコープ規律にメッセージのファイルの例外がない | 解消のまま | `implementer.md:111-113` に例外の文がある。F4 は触っていない |
| 1 回目 #2、#3 leader の手順 8 と reviewer の `CHECKS:` | 解消のまま | `leader.md` と `reviewer.md` は F4 の差分にない。2 回目 LOW 2(散文の指示を受け入れる文)も、`reviewer.md:21-24` が「指示とみなすのはこの行だけ」「`CHECKS:` の行のない TASK は…両方のスクリプトを打つ」で、解消している |
| 1 回目 #4、2 回目 LOW 1 複数スライスの `VERIFY:` と RESULT | 解消のまま | `implementer.md:148-153` のレポート契約がスライスごとの名前付きの `COMMIT:`・`VERIFY:`・`SELF-CHECK:` の形を持つ。F4 は 53-65 行目を書き換えたが、確かめの項目の前後の構成(スライスの名前、1 スライス = 1 コミット)に触れていない |
| 1 回目 #5 コミットの文が確かめより前 | 解消のまま | `:33-35` の「次の項目の機械的な確かめを行ってから…コミットする」が残る。新しい「どちらもコミットしない」の文はこの順と矛盾しない |
| 1 回目 #6 `send` の例の置き場所 | 解消のまま | F4 は skill の `send` の行に触れていない |
| 1 回目 #7 テストファイルの行数 | 解消のまま。余裕は 1 行 | `prompts_test.go` は main と同一。`prompts_verification_test.go` は 799 行で 800 行の目安を越えていない。F4 の新しいテストは `…_multi_test.go` に入った(下の所見 5) |
| 1 回目 #8 テストの補助 3 点 | 解消のまま | F4 が足した 2 本は補助を共有定数からでなく直書きの句で持つが、`implementerSelfCheckItems` などの既存の補助は変わらない |

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| 1 | LOW | naming | 「前からあった変更」を指す言葉が項目の中で 3 通りある。集合の定義は「編集を始める前に記録した `git status --porcelain` に出ていた変更」(記録)。それを指すのが「座席が始める前からあった変更」「座席が始める前から stage されていた」(座席)と、leader に返す BLOCKED に書く「スライスより前からあった変更」(スライス)。この座席は常駐する(`:8`)ので、「座席が始める前」を座席の起動前と読むと、前の TASK が残した stage 済みのパスは前からあった変更に入らず、「このスライスの変更」として消せる側に倒れる。おそらくこの読み違いは起きにくい(記録の文が先にある)。未確認です。ただ BLOCKED に書く語が座席の語と違うので、leader が読む語と座席が持つ語の対応が文からは取れない | `implementer.md:54-55`(座席)、`:61-62`(スライス)、`:63-64`(座席)、`:8`(常駐)。skill は `.claude/skills/org/SKILL.md:1336-1337`(座席)。テストが「座席が始める前」を 3 か所で固定する: `prompts_verification_multi_test.go:120`、`:126`(「座席が始める前から stage されていた」)、`:161`(skill 側)。同じファイルの doc に "predates the seat" | 言い方を 1 つにする。記録で定義した語に合わせ、たとえば「この TASK の編集を始める前に記録した…に出ていた変更」を定義の文だけに置き、あとは「記録にあった変更」と呼ぶ。BLOCKED に書く語も同じにする。直す場所は雛形 3 か所、skill の 4 つの写し、テストの句 3 つ |
| 2 | LOW | readability | 「例外は 2 つで、どちらもコミットしない」(`:56`)の「例外」の係り先が、直前の「前からあった変更は触らない」(`:54-56`)ではなく、2 文前の「scope の中で直せない項目はそのまま残し、RESULT に found and not fixed として書く」(`:52-53`)になる。読み手は直前の文に係ると取りやすく、そう取ると 1 つ目(このスライスの秘密情報を unstage して消す)は直前の規則の例外になっていない。コミットしない場合は実際には 3 つ(自分の秘密情報、前からあった変更の秘密情報、前から stage されていた scope 外のパス)あるのに、2 つ目の「前からあった変更の秘密情報」が 1 つ目の中の節(`:61-62`)に入って、数が合わない。項目は 18 行の 1 文の連なりで、座席が実行中に引いて読むには、場合分けが読み取りにくい | `implementer.md:52-65`。項目の境界の規則 `prompts_test.go:93-98` | 「例外は 2 つで」を「次のときはコミットしない」に替え、3 つの場合を同じ深さの文で並べる。サブ箇条にはしない(`markdownItem` が入れ子の `- ` で項目を切るので、テストの照合が項目の途中までになる)。テストの句 1 つ(`prompts_verification_multi_test.go:121`、`prompts_verification_test.go:97`)を一緒に直す |
| 3 | LOW | security | 「unstage」の仕方が書かれていない。コミットの側は `git add <path>...` を指定し `-A`・`-u`・`.` を禁じる(`:33`)のに、unstage の側は「このスライスのパスを unstage し」(`:58-59`)だけで、コマンドの形がない。引数なしの `git reset` や `git restore --staged .` は index 全体を空にするので、この直しが「触らない」と決めた前から stage されていたパスまで unstage する。内容は残るので失うのは index の状態だけで、重さは低い。ただトリアージの #1 が避けたい種類の作用で、座席が省略した形を選びやすい箇所でもある | `implementer.md:58-59`、`:65`。雛形 3 本に `git restore --staged` と `git reset` はない(grep) | 「パスを指定して unstage する(`git restore --staged -- <path>...`)。引数なしの `git reset` と `git restore --staged .` は使わない」を 1 文足す。skill は要約なので足さなくてよい。テストに 1 句 |
| 4 | LOW | maintainability | 同じ「stage した差分の秘密情報が消せなければ unstage して止まる」が、この計画の外の面に同じ形で残る。トリアージの #1 は、前から stage されていたパスを巻き込む穴だった。標準フローの implementer 定義(`.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`)は「unstage it and stop」で、前からあった変更を除く文がない。`/implement` の手順 7(`.claude/skills/implement/SKILL.md:42`)は「take a secret out of the worktree (edit the value out, or delete a new untracked file that holds it)」で、止まったスライスが作ったファイルに限る文がない。標準フローの作業ツリーは clean な base から作るので、共有の作業ツリーで起きる org より当たりにくい。計画の F4 の記録(`docs/plans/active/2026-10-27-org-narrow-verification.md:188`)は `/implement` の手順 7 だけを台帳に回すと書き、座席の定義 2 本(と `templates/base/` の写し)を名指していない。台帳に行はまだない(`docs/tech-debt/README.md` に `unstage` はない) | `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`、`.claude/skills/implement/SKILL.md:42-43`。写しは `.agents/skills/implement/SKILL.md`、`templates/base/.claude/agents/implementer.md`、`templates/base/.claude/skills/implement/SKILL.md`、`templates/base/.agents/skills/implement/SKILL.md`、`templates/base/.codex/agents/implementer.toml`。#261 で merge 済みの文で、この計画の範囲の外 | この計画では直さない(範囲の外)。`/sync-docs` が足す台帳の行は、`/implement` の手順 7 だけでなく、上の 3 種類の面(skill、Claude の agent 定義、Codex の agent 定義)と、それぞれの写しを名指す。「stage 済みの前からあった変更を除く文を足す」を trigger にする |
| 5 | LOW | maintainability | F4 のテスト 2 本が `prompts_verification_multi_test.go` に入ったが、このファイルは複数スライスの形(RESULT の名前付きの行、`CHECKS:` の行だけを読む reviewer)のためのもので、F4 のテスト(前からあった変更に触れないこと)は複数スライスと関係がない。同じ項目(機械的な確かめ)を固定するテストが 2 つのファイルに分かれた: `MechanicalSelfCheckBeforeEachCommit`(`prompts_verification_test.go:81`)と `SelfCheckLeavesChangesThatPredateTheSlice`(`…_multi_test.go:107`)。入れた理由は本体のファイルが 799 行で 800 行の目安に 1 行しか余裕がないことと読める(推測)が、ファイル名からは中身が引けない。`.claude/rules/ralph/architecture.md` の module の境界をパスから推せるようにする方針に合わない | `prompts_verification_multi_test.go:107`、`:153`、`prompts_verification_test.go:81`(799 行) | ファイル名を中身に合わせる(たとえば `prompts_verification_implementer_test.go` に implementer の確かめ 3 本を寄せ、本体から出す)か、先頭にファイルの中身を 1 行書く。急がない。次にテストを足すときに分ける |

## Positive notes

- 直しの規則を、新しい検出の仕組みでなく、雛形がすでに求めていた「編集を始める前の `git status --porcelain` の記録」で決めている。座席が新しく覚える手順がなく、削除を許す範囲も「このスライスで作った未追跡のファイル」に絞られた(`:59-60`)。
- 「コミットには stage してあるものがすべて入る」(`:56-57`)の 1 文が、コミットしない 2 つの場合の理由になっている。座席が `git commit` に pathspec を付けて避ける道を選ばず、BLOCKED にする筋が文から読める。
- 2 本目のテスト(skill 側)は、F4 で足した 1 文を `## 役割` の implementer の箇条書きに限って照合する。隣の箇条書きの一致で通ることがない。
- 1 本目のテストは、9 つの句の順序と、古い言い方の不在の両方を固定する。F4 の前の雛形にあった句を使うので、戻せば落ちる(F4 の前後の差分で確かめた)。
- 1 回目の 8 件と 2 回目の 2 件は、F4 を挟んでも解消のまま。F4 の差分が小さく(雛形 1 項目、skill の 1 文、テスト)、他の面に触っていない。

## Cleared, do not raise again

- テストの doc の「cross-review cycle 1 #1 of plan …」は、triage が cycle ごとに書き換えられても、cycle の名前を持つので指す先を取り違えにくい。既存の `prompts_test.go:939` と同じ形。
- 計画の S1(`docs/plans/active/2026-10-27-org-narrow-verification.md:25`)が F4 の前の規則のまま残るのは、F1〜F3 の直しも S の行を書き換えず Progress に記録した形(`:178-181`)と同じ。F4 の記録は `:186-189` にある。
- 仕様 `docs/specs/2026-08-01-org-runtime.md` の 2026-10-27 改訂 (a)(`:35`)は、秘密情報の片付けの規則を持たず、5 項目の列挙だけなので、F4 で直す文がない。
- skill が雛形より短く、unstage と値を消す手順を書かないのは、cycle 1 から続く要約の程度。
- 前回までの 10 件(上の表)。

## Tech debt identified

この run が cycle 2 / 2 の最後なので、直さなかった LOW は繰り越しになる。上の LOW 5 件のうち、このあと直されない分は、1 本の行にまとめて台帳へ足す。この run の commit はレポートと insight event に限るので、台帳には書いていない。次の行を `/sync-docs` か orchestrator が足す。所見 4 はこの計画の範囲の外なので、直す直さないにかかわらず足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| org の implementer の確かめの文言の残り(所見 1: 「前からあった変更」の言い方が 3 通り、所見 2: 「例外は 2 つ」の係り先、所見 3: unstage のコマンドの形がない、所見 5: F4 のテストが multi のファイルに入った)と、標準フローの implementer の同じ形の穴(所見 4: `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:54-56`、`.claude/skills/implement/SKILL.md:42-43` と各写し) | 文言の読み違いと省略した unstage で、前から stage されていた別の作業の index を空にしうる。内容は失われない。標準フローは clean な base の作業ツリーなので、起きにくい | pipeline の cap 2 / 2 に当たり、再 run すると 5 件の文言直しのために全段を回し直すことになる。動作を変えない文言の直しで、座席は F4 の規則に従う | org の implementer 雛形か、標準フローの implementer 定義を次に直すとき。または前から stage されていたパスを巻き込んだ報告が出たとき | `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(cycle 2)、`docs/reports/cross-review-triage-org-narrow-verification.md`(cycle 1 の #1) |

_(この review では `docs/tech-debt/README.md` を書き換えていない。上の行は `/sync-docs` か orchestrator が足す。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5)
- Follow-ups: cross-review の ACTION_REQUIRED 1 件(前から stage されていた変更を巻き込む片付け)は、雛形と skill の 4 つの写しとテストで直っている。cycle 1 の 10 件は解消のまま。残る LOW 5 件は文言とテストの置き場所で、動作を変えない。cap に届いているので回し直さず、上の 1 行にまとめて台帳へ足す。回し直すなら、優先は所見 2(「例外」の係り先。直す場所が雛形の 1 項目とテストの句 1 つで小さい)、所見 3(unstage の形。1 文)、所見 1(言い方の統一)の順。所見 4 は直さず、台帳に標準フローの面を全部名指して足す。
