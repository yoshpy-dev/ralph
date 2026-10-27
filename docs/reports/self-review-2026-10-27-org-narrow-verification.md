# Self-review report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(D1〜D6、AC1〜AC6、Progress checklist の S1〜S5 と直し F1〜F10 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 2 の 3 回目の run。cap を 3 に上げた回し直し。1 回目 dd2ffa05 と 2 回目 bd21c5fd は付録 A にまとめてある)
- Cycle: 2(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げた回し直し。`/cross-review` の規則で `cycle-count.json` は 2 のまま)、run 3
- Scope: `git diff 622cdcfb...HEAD`(HEAD cb573185、31 ファイル、+2630/-112)。中心は cross-review 2 回目の ACTION_REQUIRED 2 件の直し、F9 3c64d65d(org の雛形 2 本、`/org` skill の 4 つの写し、`internal/org/prompts_verification_test.go`)と F10 662d3e47(plan の雛形の 6 つの写し、`tests/test-slice-verification.sh`、org の雛形の `go.work` の言い方)。F9 と F10 のあとの cb573185 は計画の Progress だけ。2 回目の run のあとに入った F8 28d83b96(2 回目の R2-1〜R2-4 の直し)は、どの self-review run も読んでいなかったので、同じ範囲で読んだ。残りの差分は 1 回目・2 回目に読んだ範囲。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: この run の所見は R3-1〜R3-3。付録 A の ID(1〜5、R2-1〜R2-5)は、テストのコメント(`internal/org/prompts_verification_predate_test.go:15`、`:56` の「cycle 2 self-review LOW 1」「LOW 2 and 3」)と計画の Progress の「LOW 4」が引いているので、1 行ずつの表にして残してある。

## Evidence reviewed

- `git show 3c64d65d` と `git show 662d3e47` の全体。トリアージ `docs/reports/cross-review-triage-org-narrow-verification.md` の 2 件(狭めた `go test` に `-timeout=30m` がない。入れ子の `go.mod` の module を root から打つと落ちる)が名指す面は、implementer の雛形(`:37-40` と例 `:118`)、leader の雛形(`:32-35` と例 `:494`)、skill の 4 つの写し(`:1330`)、`implementerGoSliceCheck` と例の pin。直しはその 4 面に加えて、トリアージの「逸れ」にある plan の雛形の 6 つの写しにも入っている。
- コマンドの一致。`git grep -n 'go vet \./'` で live な面(docs/evidence、archive、reports を除く)を洗った。Go のコマンドを `grep -o` で取り出し、dir の名前だけ正規化すると、implementer の雛形 2 か所(`:39`、例 `:127`)、leader の雛形 2 か所(`:34`、例 `:503`)、root の skill 1 か所(`:1330`)、plan の雛形のうち見た 2 ファイルの主コマンド 2 か所、テストの定数 1 か所の 8 件と、入れ子の例 2 件が、どれも `test -z "$(gofmt -l DIR)" && go vet ./DIR/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./DIR/` の 1 形。残りの写しは取り出さず、skill の 4 つは `cmp` で、plan の雛形の 6 つは `md5` で同一を確かめた。入れ子の例 `(cd service && … go test ./pkg/)` は 6 つの写しに 1 つずつある。
- timeout の理由と入れ子の module の規則。implementer の雛形(`:41-49`)と leader の雛形(`:36-44`)の該当の文を、空白を全部落として `cmp` した(792 文字ずつで一致)。内容の正しさは、使い捨ての module(root の `go.mod` と `service/go.mod`、scratchpad、あとで消した)で確かめた: root から `go vet ./service/pkg/` は `main module (example.com/root) does not contain package` で落ち、`(cd service && …)` は通る。`go env GOFLAGS` が空のとき `GOFLAGS="-timeout=30m "` になるが、`go test` は通る。呼び手の `GOFLAGS=-timeout=5m` と `go env -w GOFLAGS=-timeout=7m` は 30m のあとに来て、実効の timeout は 5m と 7m になった(何も設定しないと 30m)。root の `go.work` が入れ子の module を `use` していれば、root からも通る。「root に `go.work` がなければ」は、この結果と合う(`go.work` があれば必ず通る、とは言っていない)。言語パックの `run_tests`(`packs/languages/golang/verify.sh:108-116`)は `GOFLAGS="-timeout=30m${goflags:+ $goflags}"` で、文が引く「先頭に置く」と一致する。
- plan の雛形の HTML コメント(`docs/plans/templates/feature-plan.md:37-40`)。`<!--` と `-->` を除いて `--` を grep し、0 件。コメントの終わりが `--->` になる行もない。
- テストの読み方。`TestRenderRolePrompt_GoSliceCheckKeepsThePackTimeoutAndEntersANestedModule`(`prompts_verification_test.go:488`)は、implementer の項目(`作業はスライス単位で進める`、`implementer.md:32-69`)と leader のミッションの 3(`:28-73`)に切って、その項目の中でコマンドの定数、timeout の 2 文、入れ子の 1 文を照合する。`markdownItem` の切れ目(trim した行が `- ` か番号で始まる所)に、2 つの項目の途中でかかる行はない。言語パックの部分は `run_tests` の本体に切ってから `GOFLAGS="-timeout=30m` を探す。`TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck`(`:531`)は skill の implementer の箇条書きに限って、コマンドと入れ子の文を照合する。例の pin(`:183`、`:792`)は同じ定数から `<dir>` を `internal/foo` に替えた形を作る。文を消す変異を頭の中で当てた結果: コマンドから `GOFLAGS=…` を消す → 定数が合わず 3 本が落ちる、timeout の理由の 1 文を消す → 2 本のサブテスト(implementer、leader)が落ちる、`go.work` の括弧を消す → 入れ子の 1 文が合わず同じ 2 本が落ちる、言語パックの `-timeout=30m` を消す → pack のサブテストが落ちる、skill の入れ子の文を消す → skill のテストが落ちる。実行はしていない。
- `tests/test-slice-verification.sh:119-125`。`assert_joined_contains_exact` は改行を空白 1 つにしたファイルから固定文字列を探す(`grep -F`)。主コマンドと入れ子の例の全文を 6 つのファイルで照合するので、`GOFLAGS=…` か `(cd service && …)` を消せば落ちる。理由の文と `go.work` の言い方は照合しない(R3-2)。
- F8 28d83b96 の全体(雛形 2 本、skill の 4 つの写し、`prompts_verification_predate_test.go`)。2 回目の R2-1〜R2-4 を 1 行ずつ当てた。下の表。
- `git diff 622cdcfb...HEAD --check` と `git diff cb573185~3..cb573185 --check` は rc 0。`wc -l`: `prompts_verification_test.go` は F9 の前が 799 行、いま 890 行。`prompts_test.go` は 2670 行(main の版と同一)。
- 確かめていない: テストと静的検査の実行(記録では F9 で変異 11 個、F10 で 3 個、どれも狙ったテストで落ちる)、座席の実機。

## 前回の所見の扱い

cross-review 2 回目のトリアージ:

| トリアージの所見 | 扱い | 理由 |
| --- | --- | --- |
| 1 狭めた `go test` に timeout がなく 10 分で打ち切られる | 解消 | 上の「コマンドの一致」の全面で、コマンドが `GOFLAGS="-timeout=30m $(go env GOFLAGS)"` の形。理由の文が 2 つの雛形にある。呼び手の timeout があとに来て勝つことは probe で確かめた。名指しの面(雛形 2 本の本文と例、skill の 4 つの写し、定数と例の pin)に加えて plan の雛形の 6 つの写しも直っている |
| 2 入れ子の `go.mod` の module を root から打つと落ちる | 解消 | 雛形 2 本に「`(cd <module> && …)` で包む」と理由がある。skill の写しに同じ規則がある(理由の文なし。要約の程度)。plan の雛形の 6 つの写しに入れ子の例がある。root の `go.work` の言い方は probe と合う |

2 回目の run の R2-1〜R2-5(F8 28d83b96 の読み):

| 2 回目の所見 | 扱い | 理由 |
| --- | --- | --- |
| R2-1 `RESUME:` の行にだけスライスの名前がない | 解消 | `implementer.md:22-27` と `leader.md:54-58` が、複数のスライスの TASK では行ごとに名前で始め、その名前のスライスのコミットに入れると書く。skill の 4 つの写し(implementer の箇条書きと leader 運用の手順 3、`.claude/skills/org/SKILL.md:1464-1465`)に `RESUME: S1: <パス>...` の例がある。テストは `resumeLineOfOneOfSeveralSlices` を 3 面で照合する |
| R2-2 skill の leader 手順 3 に分岐の欠け | 解消 | `.claude/skills/org/SKILL.md:1457-1469` に、消えていないとき・前からあった変更の秘密情報のときの escalate と、狭めるときの文がある。テストが `phrasesInOrder` で順に照合する |
| R2-3 「前からあった変更」の説明が雛形で食い違う | 解消 | `leader.md:65-69` の括弧が「止まった TASK の始めに記録された…`RESUME:` の行になかった変更。それより前に止まった TASK が残して並べなかったパスも入る」で、`implementer.md:14-18` の定義と同じ集合。旧い言い方「止まった TASK より前の変更」を戻さない照合が入った |
| R2-4 「残した」が 2 通りに読める | 解消 | `leader.md:51` と `:59-62` が「止まった TASK がコミットせずに残した」と主語を書き、外すものも含めてすべて並べる、元に戻すパスを TASK の本文に書く、と分けた。折り返しも直っている |
| R2-5 テストの整理 4 点 | 台帳へ(sync-docs 6a5bed1e)。F9 のあとに R3-1 を足す | F9 は同じ文を 2 か所で固定する形を足していない。ファイルの行数だけが動いた |

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R3-1 | LOW | maintainability | `prompts_verification_test.go` が 890 行になった。F9 の前は 799 行で、cycle 2 の 1 回目の run は 800 行の目安に「余裕は 1 行」と記録し、新しいテストを別のファイルに置いた経緯がある(R2-5 の背景)。F9 は定数 2 つ・変数 1 つとテスト 2 本(約 90 行)をこのファイルに足した。中身(Go のスライスの検査の定数とその pin)は、ファイルの主題と合っていて、置き場所の誤りではない。ただ 800 行の目安を越え、R2-5 の行(置き場と書き方 4 点)にはこの点がない。`prompts_test.go` は 2670 行で、パッケージで目安を越えるのはこのファイルだけではない | `internal/org/prompts_verification_test.go`(890 行)。F9 の追加は `:10-33`(定数と変数)、`:478-551`(テスト 2 本)。`git show 3c64d65d~1:internal/org/prompts_verification_test.go \| wc -l` が 799 | 急がない。次にこのファイルを足すとき、F9 の定数と 2 本を `prompts_verification_goslice_test.go` のような名前に移す(他のテストは `implementerGoSliceCheck` を共有するので、定数は本体に残すか、移し先から本体のテストが使えることを確かめる)。台帳は R2-5 の行に (e) として足す |
| R3-2 | LOW | maintainability | plan の雛形の 6 つの写しでは、2 つのコマンドだけが pin されていて、timeout の理由の文と入れ子の module の理由(`go.work` の言い方を含む)は pin されていない。理由の文を消しても `tests/test-slice-verification.sh` は通る。org の雛形では同じ文が定数で照合される(`prompts_verification_test.go:23-32`)ので、2 つの面で pin の深さがそろっていない。コマンドは pin されているので、座席の動作は変わらない。理由の文が消えると、次に誰かが「`GOFLAGS=` は不要では」と整理しても、HTML コメントの側からは止まらない | `tests/test-slice-verification.sh:119-125`。`git grep -n 'go\.work\|take over ten minutes' -- tests` が 0 件。雛形の文は `docs/plans/templates/feature-plan.md:39` | 2 つの needle の前に、理由の文を 1 つ(たとえば `The -timeout=30m first in GOFLAGS is the Go language pack's default`)と、`run the command inside that module's directory with paths relative to it` を足す。全部を引かなくてよい。急がない |
| R3-3 | LOW | readability | F9・F10 の新しい文の言い方が 2 つ。(a) 「呼び手が `GOFLAGS` か `go env -w` で決めた timeout」(implementer の雛形 `:44`、leader の雛形 `:39`、テストの定数 `:25`)の「呼び手」は、言語パックの英語のコメントの "the caller" を写した語で、雛形の読み手(座席)には先行する語がない。座席は TASK を送ってきた leader と取りうる。直後の「`GOFLAGS` か `go env -w` で決めた」が環境の設定だと言っているので、読み違えても動作は変わらない(おそらく。未確認)。(b) F10 が書き換えたテストの doc が文として崩れている: 「is the sentence of the same two places that runs implementerGoSliceCheck inside a nested module's root, since without a root go.work Go does not enter, from the root, a module with its own go.mod below the repository root」。主語の "sentence" が "runs" の主語に読める | `internal/org/prompts/implementer.md:44`、`internal/org/prompts/leader.md:39`、`internal/org/prompts_verification_test.go:25`、`:28-31` | (a) 「環境の `GOFLAGS` か `go env -w` で決めた timeout」にする。定数と雛形 2 本の 3 か所(`prompts_verification_test.go:25` の句が雛形と同じ文字列でなければならない)。(b) 「goSliceCheckNestedModuleSentence is the sentence of the same two places that tells the seat to run implementerGoSliceCheck inside the module's root: without a root go.work, Go does not enter, from the root, a module with its own go.mod below it」のように、動詞を「tells」に |

## Positive notes

- 直しの形が、トリアージの文言の再現でなく、言語パックの既定そのものに合わせてある。`GOFLAGS="-timeout=30m $(go env GOFLAGS)"` は `run_tests` の `GOFLAGS="-timeout=30m${goflags:+ $goflags}"` と同じ順序(既定を先頭、呼び手の設定をあと)で、呼び手の設定が勝つことを probe で確かめた。空の `go env GOFLAGS` で末尾に空白が残るが `go test` は通るので、`${…:+ }` の形にする必要はない。
- 言語パックの `run_tests` が今も `-timeout=30m` で始まることを、テストが `run_tests` の本体に切って照合する(`prompts_verification_test.go:514-521`)。雛形が引く「言語パックの既定」が変わったら、雛形の文の側が古くなる前にテストが落ちる。
- 入れ子の module の規則は `cd` を括弧で囲んだ形で、cwd が座席の残りの作業に残らない。`<dir>` を `<module>` からの相対にする指定が、`gofmt -l` と `go vet`・`go test` の 3 か所の引数を一緒に動かす理由になっている。
- 同じ文を 2 つの雛形に書き、片方に寄せなかった。leader は implementer の雛形を読まず、promoted な leader は skill だけを読む(計画 S4 の記録)ので、各面に必要な分を持たせる設計に合う。実際に 2 つの雛形の文は一字違わない。
- F10 が、トリアージの名指しにない plan の雛形の 6 つの写しまで直した。leader の手順 1 は「雛形に欄があればそれに沿う」ので、雛形が裸のコマンドのままだと、F9 の直しが plan の `— verify:` の行で元に戻る。

## Cleared, do not raise again

- skill の implementer の箇条書きに timeout の理由と `go.work` の文がないこと。skill は雛形より短い要約で、コマンドと入れ子の規則を持つ(cycle 1 から続く要約の程度)。理由の文は、コマンドを書き換えない限り効かない。
- implementer と leader の雛形の同じ文が 2 回書かれ、テストの定数にも 1 回ある(3 か所)こと。座席は自分の雛形しか読まず、定数は照合の道具なので、1 か所に寄せる道がない。
- 項目が長いこと。implementer の「作業はスライス単位で進める」は 29 行から 38 行、leader のミッションの 3 は 37 行から 46 行。1 回目の run が懸念した 18 行の項目より長いが、足された 9 行は理由・入れ子の module・ほかの言語、の独立した 3 文で、`markdownItem` が入れ子の `- ` で切るのでサブ箇条にもできない。指摘にしない。
- subtest の名前に `/` を含む `t.Run("packs/languages/golang/verify.sh", …)`(`prompts_verification_test.go:514`)。同じファイルに `docs/plans/templates/feature-plan.md`(`:622`)の先例がある。
- テストの doc に cross-review の cycle 名を書く形(`prompts_verification_test.go:480`、`tests/test-slice-verification.sh:121`)。cycle の名前と slug を持つので、triage が cycle ごとに書き換えられても指す先を取り違えにくい。1 回目の run で Cleared にした形と同じ。
- 雛形が引く `packs/languages/golang/verify.sh` が下流の project にないこと。このパスは `ralph init` が `packs/languages/<lang>/` に描くパス(`internal/cli/init.go:182`)で、下流の project にも同じ場所にある。
- 付録 A の 2 つの run の Cleared は引き続き有効。

## Tech debt identified

この run は cap を 3 に上げた回し直しで、以降に直しが入るかは `/cross-review` の結果による。直さなかった LOW は繰り越しになる前提で書く。この run の commit はレポートと insight event に限るので、台帳には書いていない。R3-1 は新しい行にせず、既にある R2-5 の行(`docs/tech-debt/README.md` の "Test housekeeping left in the org implementer's self-check and resume tests")の (e) として足す案。R3-2 と R3-3 は動作を変えない文言と pin の深さで、同じ行にまとめる。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R2-5 の行に足す (e): `internal/org/prompts_verification_test.go` が F9 で 799 行から 890 行になり、800 行の目安を越えた(Go のスライスの検査の定数とテスト 2 本、約 90 行は別ファイルに移せる)。(f): plan の雛形の 6 つの写しは主コマンドと入れ子の例だけを pin し(`tests/test-slice-verification.sh:119-125`)、timeout の理由と入れ子の理由の文は pin しない。(g): 「呼び手」(`implementer.md:44`、`leader.md:39`、定数 `:25`)に先行する語がなく、F10 が書き換えたテストの doc(`:28-31`)の文が崩れている | 動作は変わらない。理由の文が消えても、コマンドの pin が座席の動作を守る。ファイルの行数は読みにくさだけ | 動作を変えない文言とテストの置き場所で、直すたびに雛形 2 本と定数を同時に動かす。全段の回し直しに見合わない | org の implementer・leader の雛形の Go の検査の文を次に直すとき。または `prompts_verification_test.go` にテストを足すとき | `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(R3-1〜R3-3)、R2-5 の行 |

_(この review では `docs/tech-debt/README.md` を書き換えていない。)_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3)
- Follow-ups: cross-review 2 回目の ACTION_REQUIRED 2 件は、トリアージが名指した面と、名指しにない plan の雛形の写しまで、同じ 1 形のコマンドで直っている。理由の文は 2 つの雛形で一字違わず、probe で正しい。F8 の R2-1〜R2-4 も解消している。残る LOW 3 件は pin の深さ・ファイルの行数・言い方で、動作を変えない。回し直さず、上の R2-5 の (e)〜(g) としてまとめて台帳へ足すのが小さい。直すなら優先は R3-3 (a)(雛形 2 本とテストの句 1 つ)、R3-2、R3-1 の順。
- inline の修正はない(F9・F10 は implementer の slice)。F8 は、2 回目の run のあとに self-review を通らず入った直しだったので、この run で読んだ。新しい所見はない。

## 付録 A: 過去の run のまとめ

全文は git の履歴にある(`git show dd2ffa05:docs/reports/self-review-2026-10-27-org-narrow-verification.md`、`git show bd21c5fd:docs/reports/self-review-2026-10-27-org-narrow-verification.md`)。ID はテストのコメントと計画が引く。行番号は当時の HEAD のもので、直しのあとの行とは合わない。

| run | コミット | 見た HEAD | 結果 |
| --- | --- | --- | --- |
| cycle 1 の 1 回目 | 06f60e10 | 前半の S1〜S5 の差分 | LOW 8、Merge yes(F1 0ea899f5、F2 f0870d6e で解消) |
| cycle 1 の 2 回目 | f2d075c8 | F1・F2 のあと | LOW 2、Merge yes(F3 8cda482e で解消) |
| cycle 2 の 1 回目(run 1) | dd2ffa05 | 50fc3faf(cross-review 1 回目の F4 ffafc9d2 のあと) | LOW 5、Merge yes |
| cycle 2 の 2 回目(run 2) | bd21c5fd | 2f884ae1(F5・F6 のあと) | LOW 5、Merge yes |

run 1 の所見(ID 1〜5。F4 ffafc9d2 の「stage 済みの前からあった変更を巻き込む片付け」の直しへの所見):

| ID | Area | 内容 | 扱い |
| --- | --- | --- | --- |
| 1 | naming | 「前からあった変更」の呼び方が項目の中で 3 通り(記録・座席・スライス) | F5 a7d879d9 で解消(定義は TASK ごとの記録で 1 度) |
| 2 | readability | 「例外は 2 つ」の係り先が 2 文前の規則になる | F5 で解消(コミットしない 3 つの場合を 1 文に並べた) |
| 3 | security | unstage のコマンドの形がなく、引数なしの `git reset` が index 全体を空にしうる | F5 で解消(`git restore --staged -- <パス>`、パスなしの `git reset` を禁止) |
| 4 | maintainability | 標準フローの implementer の定義と `/implement` の手順 7 に同じ形の穴(#261 の文、この計画の範囲の外) | 台帳の行へ(sync-docs 6a5bed1e、`docs/tech-debt/README.md` の "The standard flow's implementer can clean up a secret…") |
| 5 | maintainability | F4 のテストが複数スライスのファイルに入った | F5 で `prompts_verification_predate_test.go` に移して一部解消。残りが R2-5 |

run 2 の所見(ID R2-1〜R2-5。F5・F6 と inline 51cf59e2 への所見):

| ID | Area | 内容 | 扱い |
| --- | --- | --- | --- |
| R2-1 | maintainability | `RESUME:` の行だけスライスの名前がない(複数スライスの TASK で引き継ぎ先が決まらない) | F8 28d83b96 で解消 |
| R2-2 | maintainability | skill の leader 手順 3 に、値が消えていないときの escalate と、スライスを狭めるときの文がない | F8 で解消 |
| R2-3 | naming | `leader.md` の「前からあった変更」の説明が `implementer.md` の定義と違う | F8 で解消 |
| R2-4 | readability | 「残したパス」の「残した」が 2 通りに読める。折り返しが崩れた | F8 で解消 |
| R2-5 | maintainability | テストの整理 4 点(同じ文を 2 か所で pin、`predate` のファイル名に再開のテスト、順序の照合が 2 通り、テスト名の語) | 台帳の行へ(sync-docs 6a5bed1e、"Test housekeeping left in the org implementer's self-check and resume tests") |

run 1・run 2 の Cleared(本文は上の履歴): 見出しだけ挙げる。skill が雛形より短いこと、`leader.md` の `git diff HEAD` が値を画面に出しうること、型付きプロトコルの節が本文の行として挙げるのは `VERIFY:` だけで `RESUME:` は「ミッション」の 3 に書かれていること、51cf59e2 を inline で直したのが軽い編集の例外の範囲だったこと、テストの doc の cross-review の引用の形。
