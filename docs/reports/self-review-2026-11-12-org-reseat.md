# Self-review report: org-reseat

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(issue #312 の受け入れ条件 2、`ralph org reseat`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(cycle 2、`cycle-count.json` の `cycle` は 2)。1 回目の cross-review が出した ACTION_REQUIRED 3 件の直しのあと
- この report は、1 回目(cycle 1)の report(f8579b7c、パイプライン 1 回目の中の 3 回目の実行)を置き換える。理由: cross-review の 3 件を直した adc3e205 と e9461b79、sync-docs の a40c646f、origin/main(4220ea7c、#310)の merge 8fbd7118 を読み、直しが新しい穴を作っていないかと、残りの LOW の判定をまとめるため。1 回目の本文は `git show f8579b7c:docs/reports/self-review-2026-11-12-org-reseat.md` で読める
- Scope: `git diff 2bc28bdc HEAD`(コードは `internal/org/reseat.go`、`internal/cli/org_reseat.go` とそのテスト 2 本)、`git show a40c646f` の Go と README・AGENTS.md の差分、8fbd7118 の README.md と `docs/tech-debt/README.md` の衝突の解消。`PlanReseat`・`reseatTargets`・`reseatNoActiveSeatErr`・`reseatLifeSeats`・`reseatRun`(`stopAll`・`startAll`・`finish`)・`runOrgReseat`・`printOrgReseatResult`、#310 の `printCodexHookTrustNotice` とその呼び元も読んだ。テスト、静的解析、仕様適合、文書の drift は見ていない。リポジトリのコードは実行していない
- ID の付け方: 前回の `L2` から `L6` と `L10` は番号を変えずに続ける。今回の新しい指摘は `L11` から `L14`。行番号は 8fbd7118 のもので、動くので関数名を主にする

## cross-review の 3 件と前回の指摘の判定

| ID | 判定 | 根拠 |
| --- | --- | --- |
| cross-review 1(`--seat` が `spawned` の無い座席を断る) | 直った | `reseatLifeSeats` が、今の命の座席の event のうち `reseat=<id>` の note で終わるものを持つ座席に `touched` を立て、`reseatTargets` の `accepted` が `spawned \|\| active() \|\| touched` を受け付ける。下の「`touched`」 |
| cross-review 2(部分失敗で DECISION が消える) | 直った | `finish` が、leader が対象に無ければ、`Outcome == started` の座席だけで DECISION を作る。下の「DECISION」 |
| cross-review 3(`--config` が打ち直しから落ちる) | 直った | `reseatVerbCommand` が `--config` と `--state-dir` を足し、CLI が `--config` を絶対パスにして `HintConfigPath` へ渡す。下の「`--config`」。ただし help の「every」は言い過ぎ(L12) |
| L10 | 残っている | `finish` の `Retry` と `Decision` は stdout にしか出ない。直しは、出力が読まれる場合の道を増やしただけで、読まれない場合は変えていない |
| L2 | 残っている | `orgReseatHerdrServerProbe`(`org_reseat.go:340`、行は動いた)と `ensureOrgHerdrServer`。cross-review の直しは触っていない |
| L3 | (b) だけ残っている | `feature.go:413` の「the leader's task copies the [reviewer] values」。`feature.go` は 2bc28bdc から変わっていない |
| L4 | 残っている | 関数 `reseatSeat`(`reseat.go:738`)と型 `ReseatSeat`(`reseat.go:86`) |
| L5 | 残っている | `PlanReseat` は 147 行のまま。直しは引数を 2 か所足しただけ(`refuse(` の行の中) |
| L6 | 残っている | `docs/plans/active/2026-11-12-org-reseat.md` を指す注釈は同じ 5 か所(`reseat.go:22`、`seat.go:190`、`org_reseat.go:20`、`reseat_test.go:19`、`org_reseat_test.go:26`)。L13 が同じ型の追加 |

## Evidence reviewed

### `touched`(adc3e205)

- 立つ条件: `reseatLifeSeats` の中で、今の命(最後の disband より後)・`DryRun` でない・同じ org・`SeatID` が空でない event のうち、`reseatNoteID(ev.Details) != ""`(Details の最後の語が `reseat=` で始まる)ものがあれば、その座席に立つ。state event かどうかは問わない。座席の一覧(`seats`)は state event で作るので、state event が 1 件も無い座席は `touched` を持っても一覧に出ない
- 受け付けすぎるか: `Note` を書くのは `Stop`(`stopped` と `stop_failed`)と `Spawn`(`spawn_started` と `spawned`。`spawn_failed` には付かない)と、それに `Note` を渡す `StartFeature` だけで、`Reseat` しかそれに `reseat=<id>` を渡さない。CLI の flag は note を持たない。つまり `touched` の座席は、今の命で reseat が止めたか立てようとした座席で、その座席は計画のときに `accepted` だった。`touched` が足す先は「`spawned` が無く、今は active でもなく、reseat が触った座席」で、cross-review の 1 件目の座席だけになる。手で止めた座席(note 無し、`spawned` 無し)は断られたままで、test の「stopped by hand」が固定している
- 打ち直しが断られないか: 打ち直しの座席は、前の reseat の対象のうち立っていない座席。対象は計画のときに `spawned`・active・`touched` のどれかで、`spawned` と `touched` は前の reseat のあとも消えない。active だけで受けた座席(`spawn_started` で止まった座席)は、reseat が止めれば note を持つので `touched` になり、止める前に reseat が終われば active のまま。どちらでも打ち直しの `--seat` は通る
- 戻りの経路: 既定の対象(`--seat` なし)は `touched` を読まない。`reseatSeatNow` は `latest` と `latestIdx` だけを比べる。他の読み手は無い(`reseatLifeSeats` の呼び元は `PlanReseat` と `reseatSeatNow`)

### DECISION(adc3e205)

- `finish`: 先頭から `plan.Start` を見て、leader が 1 人でもいれば `return`(DECISION なし)。leader がいなければ `Outcome == ReseatSeatStarted` の座席を集め、1 つ以上あれば DECISION を作る。leader が先頭に並ぶことに頼らず、並び順が変わっても結果は同じ
- 名指しされる座席は `started` だけ。`stopAll` の失敗では `startAll` が走らず `started` は空になり、DECISION は出ない。`startAll` は失敗した座席で止まるので、あとの座席は `not_started` のまま入らない。名指しから漏れる座席は無い(立った座席は全部 `ReseatSeatStarted`)
- leader が対象で後の座席が失敗する場合は、DECISION は出ない(test の `res.Decision != ""` の確かめが、leader を含む失敗 3 か所にある)。その打ち直しは leader を含まないので、立った座席の DECISION を出す。つまり、立った座席は必ずどちらかの DECISION に入る
- `SpawnOutcomeIdempotent` を `started` に数えるのは前からのままで、他のコマンドが先に立てた座席も DECISION に入る。この直しで変わっていない

### `--config`(adc3e205 と e9461b79)

- 引用: 出すのは `reseatVerbCommand` の 1 か所だけで、`ShellWord` を通す。`reseat.go` で `ralph org` のコマンドを組む `VerbCommand(` は、この関数の中と DECISION の `send`(`ReseatDecisionCommand`)の 2 つだけ。残りの 8 か所(`reseatNoBindingErr` の 2、`reseatNoActiveSeatErr` の 3、`reseatCustomRoleErr` の 2、`reseatRetryCommand` の 1)は全部 `reseatVerbCommand` を通る
- 絶対パス: `runOrgReseat` が `filepath.Abs(p.configPath)` を `HintConfigPath` に入れ、org 層は受け取ったまま引用する。絶対パスは `/` で始まるので、`--` で始まる語になる心配も無い。`--config` は `ralph org` の PersistentFlag(`org.go:66`)なので、`stop`・`spawn`・`start`・`reseat` のどれにも付けられる
- 順序: `... --prompt <its prompt> --config X --state-dir Y` の形で、`<its prompt>` の置き場所の後ろに付く。引数の置き換えは `--state-dir` と同じ形
- テスト: CLI の test が、相対の `--config` を渡し、`ralph.toml` の承認方式を変えておき、`--config` なしの reseat が断られること、打ち直しを別のディレクトリから打てば通ることを固定する。直しを外せば落ちる形

### merge の衝突の解消(8fbd7118)

- `README.md` の `ralph org` の動詞の行: 共通の祖先(14531858)、1 回目の親(e9461b79)、origin/main(4220ea7c)から行を取り出し、文と `; ` で割って突き合わせた。祖先から両側で足された文は、merge 後に全部ある。どちらの親にも無い文は無い(5834 + 328 + 353 = 6515 バイト)。ファイル全体でも、merge と各親の差のうち、もう片方の側の変更と食い違うのはこの行だけ
- `docs/tech-debt/README.md` の「Test gaps left by org-feature-worktree」の行: #310 の carry-over の 2 件(11-12 と cycle 2)と、reseat の carry-over の 1 件(11-13)が時系列順に並び、「Related plan/report」の 2 つの文書(`docs/plans/archive/2026-11-12-codex-hook-trust-check.md` と `docs/plans/active/2026-11-12-org-reseat.md`)も両方ある。列は 5 のまま。reseat の注の `feature.go` の行数(792 → 848)は今も `wc -l` と同じ

### a40c646f

- Go の変更は `--seat` と `--config` の usage 文字列と、`org_legacy_ledger.go` のコメントに `reseat` を足しただけ。README の 3 行と AGENTS.md の 1 行は、`reseat` を足したほかの変更が無く、行末の空白も無い。skill の 4 組(`.claude` と `.agents` の `org` と `director`)は `templates/base` と 4 組とも同じ内容。`quality-gates.md` の root と template の差は、14531858 でも同じ位置にある既知の差

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L11 | LOW | Maintainability | `--seat` が受け付ける座席の定義が 2 つあり、直しのあとで食い違う。`reseatTargets` の `accepted`(`spawned \|\| active() \|\| touched`)と、`reseatNoActiveSeatErr` が自分で作る 2 つの一覧(`stopped` は最後の state event が `stopped` の座席すべて、`spawned` は `s.spawned` の座席)。(a) `stopped` の一覧は、`spawned` が無く reseat が触っていない座席(`spawn_started` で止まった座席を `ralph org stop` で手で止めたもの。`inFlightSeatErr` が勧める片付け方)も含む。active な座席が 1 つも無い org で、その座席が一覧に入ると、断りが「`reseat --seat leader --seat implementer`」と出し、その打ち直しが「has not been spawned since the org's last disband」で断られる。cross-review の 1 件目と同じ型(出した打ち方が通らない)で、何も止めず何も書かずに断られ、断りが通る座席を並べるので、1 往復で済む。(b) 逆に `touched` で受かる座席(`spawn_failed` で終わった座席)は、どの一覧にも入らない。`spawned` の座席が 1 つも無いと、断りは「no seat has been spawned since then: start its leader with ralph org start --plan ...」と言う。`reseatNoActiveSeatErr` の doc comment は「with none stopped, the seats that --seat accepts」と言うが、`touched` を足す前の定義の話になった。コードを読んで導いたもので、実行して確かめてはいない | `internal/org/reseat.go:600`(`accepted`)、`reseat.go:635-660`(`reseatNoActiveSeatErr` の `stopped` と `spawned` の switch)、`internal/org/reseat_test.go` の `TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded` の「stopped by hand」(断りを固定するのは `--seat` の道で、断りの一覧は見ていない) | `accepted` を `(s *reseatLifeSeat) startable() bool` のようなメソッドにし、`reseatTargets` と `reseatNoActiveSeatErr` の 2 つの一覧(`stopped` は `startable()` も満たす座席だけ、もう一方は `startable()` の座席すべて)が同じ関数を呼ぶ。6 行ほど |
| L12 | LOW | Documentation in code | 「`--config` を繰り返す」の範囲が、help と doc comment では広すぎる。help は「Every ralph org command reseat prints, in its output or in a refusal, repeats the --config that was given」、`ReseatParams.HintConfigPath` の doc は「Every ralph org command the reseat prints for its user to run, the retry … and those its refusals name」と言う。出力の最後の DECISION の `ralph org send`(`ReseatDecisionCommand`)と、その前の `ralph org read`(`orgReadCommandHint`)は `--state-dir` だけを繰り返し、`--config` を持たない。reseat が呼ぶ共通の関数の断りの中の `ralph org disband --org-id ...`(`reserve.go:609-624`)、`stopSeatCommand`(`spawn.go:2863` ほか)も持たない。`send` と `read` は ralph.toml を読まないので、動作に害は無い。害になるのは、この文を信じて「全部に付いている」と読む人か agent と、help の文をそのまま固定する test(`org_reseat_test.go:836`) | `internal/cli/org_reseat.go:95`、`internal/org/reseat.go:77`、`internal/cli/org_reseat_test.go:836`、`reseat.go` の `VerbCommand(` の呼び元(`reseatVerbCommand` の中と `ReseatDecisionCommand` の 2 つ) | 範囲を書き直す: 「`reseat`・`start`・`stop`・`spawn` のコマンドは、与えた `--config` を絶対パスにして繰り返す。`read` と `send` は繰り返さない(ralph.toml を読まない)」。test の文も合わせる |
| L13 | LOW | Maintainability | 直しのコミットが足したテストの doc comment が、cross-review の triage を番号で引く。「cross-review cycle 1, finding 1」「finding 2」「finding 3」「second and third findings of the cross-review (cycle 1)」の 4 か所。triage `docs/reports/cross-review-triage-org-reseat.md` は `/cross-review` が cycle ごとに同じファイルを書き直し、この cycle の cross-review が走ると「cycle 1 の finding 2」は読めなくなる(`#` の番号は、書き直した表のものになる)。どの comment も、本文で何を直したかを言っているので、困るのは引用だけ。L6(archive へ動く plan を指す注釈)と同じ型 | `internal/org/reseat_test.go:1601`、`1658-1663`、`internal/cli/org_reseat_test.go:277` | 番号の引用を落とすか、コミット(adc3e205、e9461b79)で引く。L6 の 5 か所と同じ回で直す |
| L14 | LOW | Consistency (merge) | origin/main の #310 は、codex の agent が hook の承認なしで動くことを知らせる 1 行(`printCodexHookTrustNotice`)を、`ralph org spawn`・`org start` の 2 つの形・`org director start` の 4 つの呼び元に足した(`org.go:967`、`1202`、`1332`、`org_director_headless.go:366`、関数の doc がその 4 つを数える)。`ralph org reseat` はこの branch にしかなかった verb で、codex の leader と座席を台帳の driver で立て直すが、この呼びを持たない(`runOrgReseat` に `printCodexHookTrustNotice` は無い。#310 の plan にも `reseat` の語は無い)。merge は文字の上では衝突しないので、8fbd7118 で気づく場所が無かった。reseat の主な使い道(usage limit のあとに別の account へ移す)は、codex の user config(hook の承認の置き場)が変わりうる場面と重なる。通知だけで、動作は変わらず、`ralph doctor` も同じ内容を報告するので LOW にする。reseat は計画を `rt.Reseat` の中で作るので、start のように「driver を解決した直後」に CLI から 1 行を出せず、直し方は 1 行の追加より大きい | `internal/cli/org.go:1332`(`runOrgStartPlan` の呼び)、`internal/cli/codex_hook_trust_notice.go:8-17`(呼び元の列挙)、`internal/cli/org_reseat.go` の `runOrgReseat`、`printOrgReseatResult`、`printOrgReseatDryRun` | `printOrgReseatDryRun` と `printOrgReseatResult` が、`plan.Start` の codex の座席ごとに 1 行を出す(計画が driver と cwd を持つ)。それが大きければ、register に #310 の兄弟として足し、reseat の次の変更で払う |
| L2 | LOW | Maintainability | 前回のまま。dry run の herdr probe が gate の前半を写している | `orgReseatHerdrServerProbe`、`ensureOrgHerdrServer` | 「status を読んで running かを返す」関数を 1 つ出す |
| L3 | LOW | Error text | (b) だけ残っている。leader を対象にしない reseat で `[reviewer]` の値が不正だと、「the leader's task copies the [reviewer] values」と出る | `feature.go:413`、`PlanReseat` の `startFeatureInput` の呼び | `refuse` の文に「leader は起こさないが、予約の比較に leader の起動の検査を通す」の 1 句を足す |
| L4 | LOW | Naming | 前回のまま。関数 `reseatSeat` と型 `ReseatSeat` が頭文字の大小だけで違う | `reseat.go:738`、`reseat.go:86` | 関数を `newReseatSeat` にする |
| L5 | LOW | Readability | 前回のまま。`PlanReseat` が 147 行で、`plan.Start[0]` が leader であることに 2 か所が頼る | `PlanReseat` | `leaderIdx` を返す形にする。leader 向けの組み立てと非 leader の `Spawn` の組み立てを別の関数にする |
| L6 | LOW | Maintainability | 前回のまま。注釈が archive へ動く plan を指す | 上の判定表の L6 の 5 か所 | plan の slug だけにするか、動かない `docs/specs/2026-11-12-org-reseat.md` を指す |
| L10 | LOW | Exception handling | 前回のまま。出力を読む手が居ない実行で reseat が一部失敗すると、残った座席と `Retry`、今回の直しで増えた部分失敗の DECISION は、読まれない stdout にしか無い | `finish`、`reseatLifeSeats` の `recorded`、`org_reseat.go` の help | help に「出力を失った reseat のあとは `ralph org status --org-id <id>` で止まった座席を見て `--seat` で名指しする」の 1 文 |

指摘にしなかったもの:

- `filepath.Abs` の失敗の断り(`runOrgReseat`)は、`--config` が相対で作業ディレクトリが消えている実行でしか起きず、そこまでは何も止めていない。引数を足した関数(`reseatNoActiveSeatErr`・`reseatCustomRoleErr`・`reseatRetryCommand`)は、文字列が隣り合うので取り違えてもコンパイルが通るが、テストが `--config` と `--state-dir` の位置を固定している。
- `finish` の `for` の中の `return` は、leader の有無を先に調べる形のほうが読みやすいが、結果は同じ。
- sync-docs(a40c646f)のあとに直しが入ったので、README の reseat の段落(`--seat` は「active or not」、「When it started only seats other than the leader」)、`/org` skill の reseat の行(「`--seat` は止めた座席と起動に失敗した座席も受け付ける」)、spec の D5・FR-7、plan の P7 は、`touched`・部分失敗の DECISION・`--config` の引き継ぎを言わない。文書の drift は `/verify` と `/sync-docs` の範囲で、このあとの 2 回目の `/sync-docs` が拾う。

## Positive notes

- `touched` は、`Note` を書く側(`Stop`・`Spawn`・`StartFeature`)が `Reseat` だけなので、受け付ける座席が「reseat が触った座席」に閉じる。手で止めた座席は断られたままで、そのことを test の「stopped by hand」が固定する。
- DECISION の直しは、`started` を `Outcome` から作り、leader が対象なら出さない。leader を含む失敗 3 か所の既存の確かめ(`res.Decision != ""`)がそのまま残り、直しで外れていない。
- `--config` は出す場所が `reseatVerbCommand` の 1 つで、`ShellWord` と絶対パス(`--` で始まらない)の両方が効く。CLI の test が、承認方式の違う `ralph.toml` を作業ディレクトリに置いて、別のディレクトリからの打ち直しまで通す。
- 直しのテストは、直しを外せば落ちる形で書かれている(cross-review 1 は打ち直しが断られる、2 は `Decision` が空、3 は文字列が合わない)。
- merge の衝突の解消は、両側の追加を落とさず、列を数え直しても崩れていない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L11: `--seat` が受け付ける座席の定義が `accepted` と `reseatNoActiveSeatErr` の 2 つ | active な座席が無い org で、断りの打ち方が通らないことがある(1 往復)。`touched` の座席が断りの一覧に出ない | パイプラインの上限(2)に届く。コードを変える直しは `/self-review` からの 2 回の再実行になる | `reseatTargets` か `reseatNoActiveSeatErr` の次の変更 | 本 report の L11 |
| L12・L13: help・doc comment の「every」の範囲、triage を番号で引く test の comment | 読み手が `--config` の範囲を誤る。cycle 2 の `/cross-review` のあとで引用が読めなくなる | 同上 | help か `HintConfigPath` の次の変更、L6 を払う回 | 本 report の L12、L13 |
| L14: `reseat` が #310 の hook の通知を出さない | 承認されない hook のまま codex の座席が立て直されても、reseat の出力には出ない(`ralph doctor` には出る) | 同上。直し方が 1 行の追加より大きい | `runOrgReseat`・`printOrgReseatResult` の次の変更、または codex の座席の reseat の最初の報告 | 本 report の L14、`docs/reports/self-review-2026-11-12-codex-hook-trust-check.md` |
| L2・L3(b)・L4・L5・L6・L10 | 前の report の通り | 前の通り | 前の通り | `docs/tech-debt/README.md` の「Seven small findings of docs/spec-org-reseat」の行 |

_(上の表は `docs/tech-debt/README.md` へまだ足していない。この実行はパイプラインの上限(`max_pipeline_cycles` 2)の 2 回目で、直さない LOW は持ち越しになる。2 回目の `/sync-docs` が、L11 から L14 を「Seven small findings of docs/spec-org-reseat」の行へ足すか、新しい 1 行にまとめるのがよい。同じ行の (e)(L6)と (f)(L10)の記述は、L13 と L10 の部分失敗の DECISION を含めて読み直す。)_

## Recommendation

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW 10 件(新しい L11 から L14 と、持ち越しの L2 から L6、L10)は、どれも先へ進むのを止めない
- cross-review の 3 件は、`touched` が受け付けすぎない、DECISION が立っていない座席を名指さず立った座席を落とさない、`--config` が `reseatVerbCommand` を通る 8 か所すべてで引用され絶対パスで出る、の 3 点を読んで確かめた。新しい穴は、断りの一覧と `accepted` の食い違い(L11)と、help の「every」(L12)の 2 つで、どちらも reseat が止めず記録せずに断る側に出る
- merge の衝突の解消は、README の動詞の行と tech-debt の行の両方で、両側の追加が残っている。L14 は衝突の外の意味の食い違い
- Follow-ups: L11 は `accepted` の関数化で 6 行ほど。L12 と L13 は文の直しだけ。L14 は設計が要る。コードを変える直しなら、パイプラインは `/self-review` からやり直す(`post-implementation-pipeline.md`)。上限に届いているので、直すなら上限を上げる
