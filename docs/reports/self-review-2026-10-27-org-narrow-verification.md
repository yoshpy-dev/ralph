# Self-review report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(D1〜D6、AC1〜AC6、Progress checklist の S1〜S5 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 / 2)
- Scope: `git diff 622cdcfb...HEAD`(HEAD d164a8ae、15 ファイル、+1193/-88、8 コミット)。役割雛形 3 本(`internal/org/prompts/implementer.md`、`reviewer.md`、`leader.md`)と、それを固定する `internal/org/prompts_test.go` の +657 行を中心に、`/org` skill の 4 つの写し、quality-gates の 2 つ、recipe の 2 つ、仕様、tech-debt の台帳を、命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。

## Evidence reviewed

- 3 つの雛形の差分を通読し、変更後の全文も読んだ(`implementer.md` 135 行、`reviewer.md` 217 行、`leader.md` の手順 1・5・8、ミッションの 3、typed protocol の節)。雛形どうしの突き合わせは次の 4 本で行った。
  - leader が写す `VERIFY:` の行(`leader.md:28-40`、`:196-197`、`:450-459`)を implementer が読む(`implementer.md:19-29`)。
  - leader が機能ごとの org で静的検査だけを指示する(`leader.md:222-232`)と、reviewer が指示のコマンドだけを打つ(`reviewer.md:17-25`)。
  - reviewer が `CHECKS:` で返す(`reviewer.md:49-55`、`:209-215`)と、leader がそれを読む(`leader.md:229-232`)。
  - implementer の file 渡しの決め(`implementer.md:74-79`)と、leader の file 渡しの決め(`leader.md:427-436`、「件を上げる」の節 `:479-483`)。
- テストは `git diff` の全文(新しい 13 本と補助の定数・関数)を読み、対象の雛形の文と 1 本ずつ突き合わせた。`markdownItem` が各項目を切る位置(次の `- ` か番号の行)に、待つ文が入っていることを雛形の行で確かめた。補助の `markdownSection`・`markdownItem`・`containsPhrase`・`codeBlockAfterLine`・`hasLine`・`templateSentences` は既存のもので、この PR は変えていない。
- 写しの一致: `.claude/skills/org/SKILL.md` は `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` と `cmp` で一致した。`docs/recipes/codex-seat-permissions.md` は `templates/base/` と `cmp` で一致した。`docs/quality/quality-gates.md` の「Quality pipeline gate」の行は、2 列目が root と `templates/base/` で同一で、3 列目は既知の差(`(see docs/tech-debt)` の有無)だけ。
- `git diff 622cdcfb...HEAD --check` は空。
- 台帳: 変えた 3 行は、変更前と同じく `|` が 6 本で、列の数は崩れていない。取り消し線の行の書き方(取り消し線、`(RESOLVED …)`、HTML コメントの `Row preserved for traceability.`)は、台帳の 16〜33 行目の前例と同じ。
- 出荷される面(雛形 3 本、skill 4 つ、recipe 2 つ)の追加行に、変更履歴の言い方(「今までは」「旧」「もう〜しない」「計画 D6」など)がないことを grep で確かめた。
- `RALPH_VERIFY_SCOPE=changed` が Go で `go test ./...` を全部走らせるという主張は、`packs/languages/golang/verify.sh:115` で確かめた。
- 確かめていない: テストと静的検査の実行(実装側の記録では雛形のテスト 50 本が通過)、各テストの変異の再現。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | implementer の「スコープ規律」は範囲外への書き込みを例外なしで禁じるが、typed protocol の節は RESULT と BLOCKED のメッセージのファイルを `.harness/state/` に「scope の外でも書いてよい」と許す。reviewer の雛形は、書いてよいものをスコープ規律の側にも「ミッションに書いたとおり例外です」と置いている。implementer だけ、同じ種類の例外が一方の節にしかない | `implementer.md:74-79`(許す側)、`:97-98`(禁じる側)。reviewer の前例: `reviewer.md:174-176`。テスト `TestRenderRolePrompt_SeatsSendMessagesFromAFile` は許す側の文だけを固定する | スコープ規律の 1 行目に「メッセージのファイル(`.harness/state/`)は例外」と足す。許す側の文と同じ言葉なら、テストの固定はそのまま使える |
| LOW | readability | leader の手順 8 は、reviewer が返す `CHECKS:` の行で「`GATE:` がどのコマンドの結果かを確かめる」と言うが、確かめた結果が違うときの動きがない。`CHECKS:` の行がない(古い binary で立てた reviewer)、行に `run-static-verify.sh` がない、のどちらでも、`GATE: pass` を静的検査が通ったものとして手順 9 へ進んでよいのか決まらない。書いてあるのは「`run-test.sh` も並んでいれば、そのまま扱ってよい」の 1 分岐だけ。reviewer 側も、`GATE: unrunnable` の BLOCKED で「ゲートで打ったコマンド」を何と書くか(打てなかったコマンドか、空か)を言わない。後ろに gate の TASK があるので、取りこぼしても遅く見つかるだけで、結果は残る | `leader.md:222-232`、`reviewer.md:49-55`、`:213-215`。テスト `…FeatureOrgReviewTaskNamesOnlyTheStaticScript`(`prompts_test.go:865`)は「確かめる」の文を固定するが、その先の分岐は固定する対象がない | 手順 8 に 1 文足す。たとえば「`CHECKS:` の行がないか、`run-static-verify.sh` がなければ、`GATE:` は確かめられていないので、同じレビューの TASK を新しい TASK_ID で送り直す。gate の TASK が最後に静的検査も走らせる」。reviewer 側は、`unrunnable` のときは打とうとしたコマンドを書くと一言足す |
| LOW | maintainability | leader から reviewer への「このコマンドだけ打て」に、決まった形がない。implementer への指示は `VERIFY:` の本文の行、例、テストが 3 点そろうが、機能ごとの org のレビューの TASK は「本文に、打つコマンドとして `…` の 1 つだけを書く」と散文で言うだけで、どの行に書くかも、例もない。reviewer 側は「TASK にコマンドの指示があれば」を読み取る。TASK の本文に別の文脈でコマンド名が出たとき(たとえば前回の失敗を述べる文)に、指示と読み違える余地がある。読み違えは、片方しか打たない側にも、既定の両方に戻る側にも振れる | `leader.md:224-226`(散文)、`reviewer.md:17-21`(読み取る側)。leader の TASK の例 `:440-447` はコマンドを持たない(機能ごとでない org の形)ため、機能ごとの org のレビューの TASK の例がどこにもない。テストは散文を固定するだけ(`prompts_test.go:865`) | 指示も `CHECKS: ./scripts/run-static-verify.sh` の本文の行にして、reviewer が同じ行を RESULT に写す形にそろえる。leader は送った行と返った行を見比べればよく、手順 8 の確かめも 1 文で済む。形を変えない場合でも、機能ごとの org のレビューの TASK の例を 1 つ足す |
| LOW | readability | implementer の `VERIFY:` は、1 つの TASK が複数のスライスを頼む場合を決めていない。implementer の側は「各スライスの実装後、TASK の `VERIFY:` の行のコマンドを実行する」と「1 スライス = 1 コミット」で、単数の行を前提にする。fan-out の例には「大きな TASK を複数の独立したスライスに分け」とあり、TASK が複数スライスを含むことは想定されている。leader の側は「頼むスライスの計画の `— verify:` のコマンド」(単数)を写す。スライスごとにコマンドが違うとき、1 行に並べるのか、スライスごとの行なのか、どの行がどのスライスのものかが分からない | `implementer.md:19-20`、`:43`、`:119-120`。`leader.md:196-197`、`:450-459`(例は 1 スライス) | leader 側に「1 つの TASK で頼むスライスは 1 つにする」か、「複数なら `VERIFY: 1: …` のようにスライスごとに行を分ける」かのどちらかを書く。2,000 字の上限があるので、前者のほうが収まりがよい |
| LOW | readability | implementer のミッションで、コミットする文が機械的な確かめの項目より前にある。最初の項目(`:19-43`)は「検証が通ってから … ステージし … コミットする」で終わり、次の項目(`:44-53`)が「コミットの前に毎回、stage した差分を読んで…」と始まる。上から順に実行する読み方では、コミットの文を読んだあとに「その前に」を読む。標準フローの implementer の定義は「ステージ → 確かめ → コミット」の順に並べている。この PR のテストは項目の並び(ステージの項目、確かめ、検証が失敗した状態のままコミットしない)を固定している | `implementer.md:29-31`、`:43`、`:44`。標準フロー側: `.claude/agents/implementer.md:33`、`:42` | 最初の項目の「ステージし、…でコミットする」に、「(コミットの前に、次の項目の機械的な確かめを行う)」と指す文を足す。項目の並びは変えないので、順序のテストはそのまま通る |
| LOW | maintainability | `/org` skill の `send` の行の例は `--text "$(cat task.txt)"` のままで、ファイルの置き場所を言わない。この PR は leader の雛形に「TASK・DECISION・STOP も `.harness/state/` のファイルから渡す」を足した(`VERIFY:` の行の `$(gofmt -l …)` を shell に展開させないため)。skill では同じ決めが「受信箱」の節の escalate にしか届いていない。S4 の記録は「昇格したセッションの leader は雛形を読まず、この skill だけを読む」ので skill に `VERIFY:` の文を足したと言う。その leader は `$(cat <ファイル>)` の形は例から知るが、`task.txt` を worktree の直下に作ってもよいと読める。機能の org の worktree に未追跡のファイルが残ると、`ralph org gate` は作業ツリーが汚れているとして断る(reviewer の BLOCKED の例 `reviewer.md:120` が同じ断りを使う) | `.claude/skills/org/SKILL.md:209`(`send` の行の例)、`:334-341`(escalate だけのファイルの決め)。雛形側: `leader.md:427-436`。4 つの写しは同一 | `send` の行の例を `.harness/state/task-X.txt` の形にし、「受信箱」の節のファイルの決めを TASK の送信にも当てはまると 1 文で言う(4 つの写しは `scripts/sync-skills.sh` と templates/base で同じ手当て) |
| LOW | maintainability | `internal/org/prompts_test.go` が 2,670 行から 3,327 行になった(+657 行)。files < 800 行の目安(`~/.claude/rules/code-review.md`)を、この PR の前から 3 倍超えている。同じパッケージには、leader の受信箱のテストを分けた前例の `prompts_leader_inbox_test.go`(378 行)がある。足した 13 本と 5 つの補助(`implementerGoSliceCheck`、`implementerSelfCheckItems`、`reviewerStaticChecksLine`、`reviewerDiffReviewItems`、`leaderNamesStaticScriptAlone`)は「検証コマンドの固定」で 1 つのまとまりになり、既存の補助(`markdownItem` など)を使うだけで、他のテストからは呼ばれない | `internal/org/prompts_test.go`(`wc -l` 3327)、新しいテストは `:233-402` と `:563-1049` の 2 か所に入っている | 次にこのファイルを触るとき、足した分を `prompts_verify_test.go` に移す。同じパッケージなので補助はそのまま使える |
| LOW | readability | テストの補助に読みにくい点が 3 つある。(a) `implementerGoSliceCheck` は雛形の文字列を照合するために markdown のバッククォートごと持つ。そのため、RESULT と TASK の例を照合する 2 か所が同じ「バッククォートを外し、`<dir>` を `internal/foo` にする」を書き写している。(b) `…ReviewExamplesCarryTheChecksLine` の `bodies` は RESULT と BLOCKED の両方の本文をためるが、読むのは BLOCKED の 1 件だけ。(c) `selfCheck, at := bodyLine("SELF-CHECK"), -1` は型の違う 2 つを 1 行で初期化し、`at` の意味(直前の項目の位置)が行からは読めない | (a) `prompts_test.go:236`、`:369`、`:974`。(b) `:712`、`:745`、`:748`。(c) `:373` | (a) バッククォートなしの定数と、`<dir>` を埋める小さな関数 `goSliceCheckFor(dir)` に分け、雛形の照合には `"`" + … + "`"` を使う。(b) `blockedBody` の 1 変数にする。(c) `at := -1` を別の行にし、名前を `prevAt` にする。どれも直さなくても誤りではない |

## Positive notes

- 3 つの雛形が互いに指す箇所は、突き合わせた 4 本ともそろっている。`VERIFY:` の文字列(`test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && go test ./<dir>/`)は leader のミッションの 3、implementer のミッション、RESULT の例、TASK の例で同一で、`implementerGoSliceCheck` 1 つから 3 本のテスト(implementer の例、leader のミッション、leader の TASK の例)が照合する。
- D6 の安全側の性質(コマンドの指示がなければ両方を打つ)を、reviewer の項目 1 だけを切り出して固定している(`…ReviewTaskRunsExactlyTheNamedCommands`)。項目 5 にも両方のスクリプトが出るので、項目全体を見ると既定が静的検査だけに戻っても通ってしまう。テストの doc がそのことを理由に書いている。
- `…OnlyAFeatureOrgNamesTheStaticScriptAlone` は、検出器が空振りしていないことを自分で示す。ミッションの 3 の文が「静的検査と両方」の側で拾われること、手順 8 の指示が「静的検査だけ」の側で拾われることを、同じテストの中で確かめる。
- 例の RESULT・TASK・BLOCKED は、文字列の照合に加えて `protocol.Parse` と `protocol.Validate` を通し、`VERIFY:`・`CHECKS:` がヘッダではなく本文にあることを `protocol.HeaderValues` で確かめる。雛形の例が実際に送れるメッセージであることを固定している。
- 失敗の文言は、期待する言葉と、実際に取り出した項目の全文を両方出す。既存のテストと同じ形。
- `VERIFY:` の行の `$(gofmt -l …)` を shell に展開させない問題を、S1 の時点で見つけて S3 で送り方の決めにした(implementer と leader の両方)。D4 の「ヘッダにしない」とも合っている。
- tech-debt の台帳は、解決した行を取り消し線にし、解決の理由と「静的検査は 2 回走る」ことの両方を HTML コメントに残した。残る行(codex の sandbox、スクリプトの共有)の文も、新しい分担に合わせて直してある。

## Cleared, do not raise again

- テストの doc が `docs/plans/active/<plan>.md` のパスではなく計画の slug(`2026-10-27-org-narrow-verification`)で書いているので、`/pr` の archive で指す先が消えることはない。台帳の行が `docs/plans/active/…` を指すのは、`archive-plan.sh` が台帳だけを書き換えるので、既知の扱い。
- implementer が「scope の中で直せない項目」を残してコミットする形は、標準フローの implementer の定義と同じ(`.claude/agents/implementer.md:40`)。org の雛形だけの穴ではない。
- 台帳の「org runtime FR-7 quality gate is prompt-instructed, not hook-enforced.」の行に古い文が残るのは、計画の Progress が sync-docs への渡しとして記録済み。
- `packs/languages/golang/verify.sh:115` の `go test ./...` は `RALPH_VERIFY_SCOPE` に関係なく走る。雛形の主張は合っている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

台帳への追加はない。上の LOW 8 件はどれも cycle 2 の直しで済む範囲で、繰り越すのは cycle 2 でも直さなかった分になる(cycle 2 が最後の run なので、その時点で残った LOW は 1 本の行にまとめて台帳へ足す)。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 8)
- Follow-ups: 所見はどれも雛形・skill・テストの文言や置き場所の直しで、動作を変えない。直す場合、雛形と skill は `/self-review` から回し直す。優先は、3 件目(leader から reviewer への指示に決まった形を持たせる。直せば 2 件目の分岐も 1 文で済む)と 1 件目(スコープ規律の 1 行)。6 件目は skill の 4 つの写しに同じ手当てが要る。`internal/org/prompts_test.go` への手入れは 7・8 件目と一緒にするのがよい。
