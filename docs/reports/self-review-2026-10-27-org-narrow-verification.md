# Self-review report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(D1〜D6、AC1〜AC6、Progress checklist の S1〜S5 と直し F1・F2 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 / 2 の 2 回目の run。1 回目は 06f60e10 の LOW 8 件)
- Scope: `git diff 622cdcfb...HEAD`(HEAD 7abc0bed、17 ファイル、+1481/-99、12 コミット)。中心は 1 回目のあとの直し 2 本、F1 0ea899f5(1〜5 件目)と F2 f0870d6e(6〜8 件目)。役割雛形 3 本(`internal/org/prompts/implementer.md`、`reviewer.md`、`leader.md`)、`/org` skill の 4 つの写し、新しいテストファイル `internal/org/prompts_verification_test.go`(796 行、テスト 16 本)を、命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。1 回目に読んだ残り(quality-gates、recipe、仕様、台帳)は、直しの 2 本が触れていないので読み直していない。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。

## Evidence reviewed

- `git diff 06f60e10..HEAD` のうち雛形 3 本と skill の差分を全文読み、変更後の `implementer.md`(ミッション、typed protocol、スコープ規律、レポート契約)、`reviewer.md` のミッションの 1〜5、`leader.md` の手順 8 と typed protocol の節を、前後を含めて読んだ。
- 1 回目の 8 件を、直した箇所ごとに雛形の行で確かめた(下の「1 回目の 8 件の扱い」)。
- 新しいテストファイルは全文を読み、対象の雛形の文と突き合わせた。`step12Marker`・`squashSpace`・`templateSentences`・`renderSeatPrompt`・`markdownSection`・`markdownItem`・`containsPhrase` は `prompts_test.go`、`reportNamesRepoRoot`・`readRepoFile`・`codeBlockAfterLine`・`hasLine` は `report_names_sync_test.go` に定義があり、移したあとも参照先が残っている(定義の grep)。`markdownItem` の照合は「marker を含む最初の行」から次の `- ` か番号の行までなので、スコープ規律の項目(`- scope: {{SCOPE}} の範囲外…`)の途中の語句を marker にした新しいテストも、項目の続きの行まで取れる。
- `git diff 622cdcfb...HEAD --stat -- internal/org/prompts_test.go` は空で、`prompts_test.go` は base と同一(2,670 行)。足したテストはすべて新しいファイルにある。
- 写しの一致: `.claude/skills/org/SKILL.md` は `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` と `cmp` で一致。
- `git diff 06f60e10..HEAD --check` は空。出荷される面(雛形 3 本、skill)の追加行に、変更履歴の言い方(「今までは」「旧」「計画 D6」など)はない(grep)。`.harness/state/` は root と `templates/base/` の両方の `.gitignore` 47 行目にある。`ralph org gate` が未追跡のファイルで断ることは `internal/org/gate_test.go:482-486` のケースで確かめた。
- 確かめていない: テストと静的検査の実行(実装側の記録では雛形のテスト 53 本が移す前後で同じ)、変異の再現。

## 1 回目の 8 件の扱い

| # | 1 回目の所見 | 扱い | 理由 |
| --- | --- | --- | --- |
| 1 | implementer のスコープ規律にメッセージのファイルの例外がない | 解消 | `implementer.md:99-103` の 1 行目に「RESULT と BLOCKED のメッセージのファイルは、「typed protocol」の節に書いたとおり例外です。置き場所の `.harness/state/` は git が無視します」が入った。許す側(`:77-79`)と同じ言葉で、reviewer の前例(`reviewer.md` のスコープ規律)と同じ形。固定するテスト `…ScopeRuleExceptsTheMessageFiles` も足された |
| 2 | leader の手順 8 に、`CHECKS:` が確かめられなかったときの動きがない | 解消 | `leader.md:236-243` に、RESULT の `CHECKS:` がないか `run-static-verify.sh` がなければ手順 9 に進まず新しい TASK_ID で送り直す、2 回続けて通らなければ 3 回目は送らず `ralph org escalate` で BLOCKED を上げる、が入った。上限の文は手順 12 の gate の TASK(`leader.md:318-320`)と同じ形。reviewer 側は `reviewer.md:58-60` と report 契約に「`GATE: unrunnable` では、打とうとしたコマンドを書く」が入った。送り直しの規則は RESULT だけを対象にし、BLOCKED の `fail` と `unrunnable` は手順 5 の既存の扱いに任せている。静的検査が通ったという主張が出るのは RESULT だけなので、これでよい |
| 3 | leader から reviewer への「このコマンドだけ打て」に決まった形がない | 解消(残りが 1 点。下の所見 2) | 指示を TASK の本文の `CHECKS: ./scripts/run-static-verify.sh` の 1 行にし(`leader.md:229-230`、reviewer が読む側は `reviewer.md:21-25`)、reviewer が同じ行を RESULT と BLOCKED に写す。leader は送った行と返った行を見比べるだけでよく、2 件目の分岐も 1 文で済んでいる。形を決める書式が文字列としてあるので、機能ごとの org のレビューの TASK の例を別に置かなくてもよい。ただし reviewer 側に、`CHECKS:` の行がなくても散文の指示を指示として扱う文が残った |
| 4 | 1 つの TASK が複数のスライスを頼む場合が決まっていない | 解消(RESULT 側が残る。下の所見 1) | `VERIFY:` の行をスライスごとに 1 行、各行をスライスの名前で始める(`VERIFY: S1: <コマンド>`)、スライスが 1 つなら `VERIFY: <コマンド>` のまま、が leader のミッションの 3(`leader.md:37-39`)、手順 5(`:201-202`)、implementer のミッション(`implementer.md:20-24`)、skill の 3 か所で同じ言い方でそろった。1 回目の 2 つの案のうち、2,000 字の上限に収まりにくい案を選んだが、Go のコマンドは 1 行 80 字ほどなので収まる |
| 5 | implementer のミッションで、コミットする文が機械的な確かめの項目より前にある | 解消 | `implementer.md:31-33` が「ステージし、次の項目の機械的な確かめを行ってから、Conventional Commits … でコミットする」になった。項目の並びは変えていない。テスト `…MechanicalSelfCheckBeforeEachCommit` が、ステージ、確かめ、コミットの順を同じ項目の中で固定する |
| 6 | `/org` skill の `send` の例が `task.txt` を直下に置くと読める | 解消 | `send` の行の例が `--text "$(cat .harness/state/task.txt)"` になり、行の終わりに「本文のファイルは `.harness/state/` の下に置く(git が無視する場所。ほかの場所に未追跡のファイルを残すと、機能の org の `gate` が作業ツリーが汚れているとして断る)」が入った。4 つの写しは同一 |
| 7 | `prompts_test.go` が 3,327 行 | 解消 | `prompts_test.go` は base と同一(2,670 行)に戻り、足した分は `prompts_verification_test.go`(796 行)にある。このブランチが 800 行の目安を越えさせてはいない。新しいファイルは目安の 4 行下なので、次にテストを足すなら 2 つに分ける |
| 8 | テストの補助の読みにくさ 3 点 | 解消 | (a) `implementerGoSliceCheck` はバッククォートなしの定数になり、バッククォートで包む照合は `"`" + … + "`"` の形。`<dir>` を埋める `ReplaceAll` の 1 行が 2 か所(`:166`、`:698`)に残るが、書き写しではなく 1 行の呼び出し。(b) `blockedBody` の 1 変数になった。(c) `prevAt := -1 // where the previous item was found in selfCheck` に分かれた |

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | 複数のスライスを頼む TASK の `VERIFY:` の形(スライスごとの行)は、受け取る側と送る側で決まったが、返す側の RESULT の形が決まっていない。implementer の RESULT の例は `COMMIT:`・`VERIFY:`・`SELF-CHECK:` が 1 行ずつで、1 スライス = 1 コミット(`:47`)なので、k スライスなら SHA も検証の結果も機械的な確かめも k 組できる。レポート契約は「打った検証のコマンドとその結果」「コミット SHA」と単数で言う。leader は RESULT の `VERIFY:` の行を機械的には読まないので結果は残るが、どのスライスの SHA か、どのスライスの検証かが RESULT から読めない。なお 2,000 字の上限は、3 スライスなら `VERIFY:` と `SELF-CHECK:` で 700 字ほどで収まる | `implementer.md:20-24`(複数スライスの受け取り)、`:88-98`(RESULT の例、`COMMIT:` は `:91`)、`:129-137`(レポート契約)。leader 側は `leader.md:37-39`、`:201-202` | レポート契約に 1 文足す。たとえば「複数のスライスを頼まれたときは、`COMMIT:`・`VERIFY:`・`SELF-CHECK:` の行をスライスの名前を付けてスライスごとに書く(`VERIFY: S1: <コマンド> → pass`)」。例は 1 スライスのままでよい。固定するテストは、`…SeveralSlicesInOneTaskGetAVerifyLineEach` の対象に report 契約の文を 1 つ足す |
| LOW | maintainability | reviewer のミッションの 1 に、`CHECKS:` の行がなくても「TASK が打つコマンドをはっきり指示していれば、それも指示として扱う(`CHECKS:` の行を書かない leader もある)」が残った。1 回目の 3 件目が挙げた、TASK の本文にコマンド名が別の文脈で出たときに指示と読み違える余地は、この文が開けたままにしている。「はっきり」の判断は reviewer に任され、機能ごとの org では送り直しの規則が拾うが、そうでない org には拾う側がない。この雛形の leader は機能ごとでない org のレビューの TASK にコマンドを書かない(`leader.md:40-43`)ので、`CHECKS:` を書かない leader は、この雛形で出荷される範囲には居ない。括弧の中の「書かない leader」が誰かも、文からは分からない。方向は、指示がなければ両方を打つという D6 の安全側とは逆(絞る側)に倒れる | `reviewer.md:21-25`。skill の写しは `.claude/skills/org/SKILL.md:1342-1343`。テストは `prompts_verification_test.go:239` で文を固定している | この文を外す。`CHECKS:` の行がなければ既定の 2 つを打つことになり、遅くなるだけで抜けない。残す場合は、括弧の中を、どの leader の話かが読める言い方(送り元が `CHECKS:` の形を知らない場合)にする。外すなら skill の写し 4 つと、テストの 1 つの期待を一緒に直す |

## Positive notes

- 直しは 8 件すべて、1 回目の「推奨」の形のとおりか、それより小さい手で済んでいる。雛形、skill の 4 つの写し、テストの固定が同じ言葉でそろい、skill の `cmp` も一致した。
- 送り直しの上限(`leader.md:240-243`)は、手順 12 の gate の TASK の上限と同じ文型で、「この理由で」「送り直した TASK の RESULT にも」と数える対象を限っている。BLOCKED の経路(手順 5 の `fail` と `unrunnable`)を巻き込まないので、静的検査が落ちた reviewer が送り直しの対象にならない。
- `TestRenderRolePrompt_Implementer_ScopeRuleExceptsTheMessageFiles` は、1 件目の穴(片方の節にしか例外がない)を、スコープ規律の側から固定する。許す側の文は既存のテストが固定しているので、2 つの節が対で固定された。
- 移したテストファイルは、`prompts_test.go` を base に戻したので、main への差分に既存テストの削除や並べ替えが混ざらない。補助が `prompts_test.go` と `report_names_sync_test.go` の既存のものだけで、同じパッケージ内の重複定義もない。
- テスト `…FeatureOrgResendsAReviewWithoutTheStaticCheck` の doc は、上限の数え方(「2 回続けて」「3 回目は送らない」)を手順 12 の文と並べて書いていて、後で数字を変えるときに突き合わせる先が分かる。

## Cleared, do not raise again

- テストの doc が `docs/plans/active/<plan>.md` のパスではなく計画の slug で書いているので、`/pr` の archive で指す先が消えることはない。台帳の行が `docs/plans/active/…` を指すのは、`archive-plan.sh` が台帳だけを書き換えるので、既知の扱い。
- implementer が「scope の中で直せない項目」を残してコミットする形は、標準フローの implementer の定義と同じ(`.claude/agents/implementer.md:40`)。org の雛形だけの穴ではない。
- 台帳の「org runtime FR-7 quality gate is prompt-instructed, not hook-enforced.」の行に古い文が残るのは、計画の Progress が sync-docs への渡しとして記録済み。
- `packs/languages/golang/verify.sh:115` の `go test ./...` は `RALPH_VERIFY_SCOPE` に関係なく走る。雛形の主張は合っている。
- 1 回目の 8 件(上の表)。解消した理由を確かめたので、同じ指摘を出さない。
- `…Implementer_MechanicalSelfCheckBeforeEachCommit` の `at := -1` と、別のテストの `prevAt := -1` の名前の差は、どちらも直前の位置を持つ変数で、読み違えない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

台帳への追加はない。上の LOW 2 件はどちらも雛形・skill・テストの文言の直しで済む範囲で、cycle 2 が最後の run になる。cycle 2 でも直さなかった分は、そのときに 1 本の行にまとめて台帳へ足す。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2)
- Follow-ups: 1 回目の 8 件はすべて解消した。残る 2 件は動作を変えない文言の直しで、直す場合は雛形と skill の 4 つの写しとテストを一緒に直し、`/self-review` から回し直す。優先は 2 件目(reviewer の散文の指示を受け入れる文。外せば D6 の安全側に戻る)。1 件目はレポート契約の 1 文で済む。
