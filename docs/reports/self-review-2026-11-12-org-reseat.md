# Self-review report: org-reseat

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(issue #312 の受け入れ条件 2、`ralph org reseat`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(cycle 2、`cycle-count.json` の `cycle` は 2)。1 回目の cross-review が出した ACTION_REQUIRED 3 件の直しのあと
- この report は、1 回目(cycle 1)の report(f8579b7c、パイプライン 1 回目の中の 3 回目の実行)を置き換える。理由: cross-review の 3 件を直した adc3e205 と e9461b79、sync-docs の a40c646f、origin/main(4220ea7c、#310)の merge 8fbd7118 を読み、直しが新しい穴を作っていないかと、残りの LOW の判定をまとめるため。1 回目の本文は `git show f8579b7c:docs/reports/self-review-2026-11-12-org-reseat.md` で読める
- Scope: `git diff 2bc28bdc HEAD`(コードは `internal/org/reseat.go`、`internal/cli/org_reseat.go` とそのテスト 2 本)、`git show a40c646f` の Go と README・AGENTS.md の差分、8fbd7118 の README.md と `docs/tech-debt/README.md` の衝突の解消。`PlanReseat`・`reseatTargets`・`reseatNoActiveSeatErr`・`reseatLifeSeats`・`reseatRun`(`stopAll`・`startAll`・`finish`)・`runOrgReseat`・`printOrgReseatResult`、#310 の `printCodexHookTrustNotice` とその呼び元も読んだ。テスト、静的解析、仕様適合、文書の drift は見ていない。リポジトリのコードは実行していない
- ID の付け方: 前回の `L2` から `L6` と `L10` は番号を変えずに続ける。今回の新しい指摘は `L11` から `L14`。行番号は 8fbd7118 のもので、動くので関数名を主にする
- 追記: L11 と L12 を直した 0468d96f の差分は、末尾の「cycle 2 の差分(0468d96f)」の節で読んだ。L11 は閉じ、L12 は L15 に置き換わり、L16 が新しい。この節より下の L11・L12 の行と Recommendation の件数は、その節で置き換わった
- 追記(cycle 3): cycle 2 の cross-review の ACTION_REQUIRED 1 件を直した def8d2a3 の差分は、末尾の「cycle 3 の差分(def8d2a3)」の節で読んだ。新しい指摘は L17 と L18 で、L6 と L13 は箇所が増えた。ブランチの今の指摘 12 件の一覧、Tech debt、Recommendation は、その節の「ブランチの今の指摘」「Tech debt の置き換え」「Recommendation(この差分)」が、上の同名の節と cycle 2 の差分の節を置き換える
- 追記(cycle 3、df8f674e): L17 と L18 を直した df8f674e の差分は、末尾の「cycle 3 の差分(df8f674e)」の節で読んだ。L17 と L18 は閉じ、新しい指摘は L19 の 1 件。ブランチの今の指摘 11 件の一覧、Tech debt、Recommendation は、その節の同名の見出しが、def8d2a3 の節の同名の見出しを置き換える。def8d2a3 の節の `/sync-docs` への渡し(register に (l) と (m) を足す)のうち、L17 と L18 の分はその節で取り下げた
- 追記(cycle 3、b142944d): verify の V19 を受けて `ralph org reseat` の help に start 時の検出を足した b142944d の差分は、末尾の「cycle 3 の差分(b142944d)」の節で読んだ。新しい help の文は、コードと照らして偽のものが無い。新しい指摘は L20 と L21 の 2 件(どちらも LOW)。L19 は残り、持ち越しの扱いは変わらない。ブランチの今の指摘 13 件の一覧、Tech debt、Recommendation は、その節の同名の見出しが、df8f674e の節の同名の見出しを置き換える。df8f674e の節の `/sync-docs` への渡しのうち、help の分はその節で取り下げた

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

## cycle 2 の差分(0468d96f)

- Scope: `git diff f754188f..HEAD` の 4 ファイル(`internal/org/reseat.go`、`internal/cli/org_reseat.go`、そのテスト 2 本)を全部読み、`reseatTargets`・`reseatNoActiveSeatErr`・`reseatLifeSeats`・`reseatCustomRoleErr`・`startAll`・`finish` を周りまで読んだ。残りの差分は上の節で読んだもので、意味が変わる所は無い。テストと静的解析は走らせていない。テストが呼ぶ補助関数(`retryParams`・`mustPlanReseat`・`seatIDs`・`mustReadEvents`・`stopSeat`)は grep で存在だけ確かめた
- ID: 新しい指摘は `L15` と `L16`

### L11 と L12 の判定

| ID | 判定 | 根拠 |
| --- | --- | --- |
| L11 | 直った | 下の「`startable()`」 |
| L12 | 「every」は直った。書き直した文に L15 が残る | `read` と `send` を外し、`reseat`・`stop`・`spawn`・`start` と書いたのは、L12 の推奨のとおり。ただしその推奨に私の誤りがあった(L15 (a)) |

### `startable()`

- 受け付ける座席: 旧 `accepted` の本体 `spawned || active() || touched` と同じ式で、差分に式の変更は無い。`reseatTargets` の `--seat` の判定と、断りに並べる `can` の組み立て、`reseatNoActiveSeatErr` の 3 か所が同じメソッドを呼ぶ。`accepted(` は残っていない(grep)。`startable` を使う他の名前も無い
- 断りが並べる座席: `startable()` の座席だけ。旧版から変わるのは 2 点で、L11 の (a)(`spawned` も `touched` も無い、手で止めた座席が `stopped` の一覧に入る)と (b)(`touched` だけの座席が一覧に出ない)に当たる。断りに来る時点で active な座席は無く(あれば既定の対象になる)、`startable()` の `active()` の項は断りの側では働かない。害は無い
- 次の実行で通るか: 断りの一覧の座席は `startable()` を満たすので、打ち直しの `--seat` は `reseatTargets` を通る。対象の選び方で断られる座席は無い。そのあとの検査が 1 つ残る(`reseatCustomRoleErr`、L16)
- テスト: 新しい `TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes` は、直す前のコードなら落ちる形。「touched」の枝は reviewer が旧 `spawned` の一覧に入らない(`spawned` が無い)ので文言も一覧も合わず、断りの一覧の 3 座席を `reseatTargets` に 1 つずつ通し、`PlanReseat` にも全部通す。「by hand」の枝は reviewer が旧 `stopped` に入る。2 つの述語に戻せば、どちらかの枝が落ちる。既存 2 本の期待文字列は、新しい文言に合わせて変わっただけで、確かめる中身は弱くなっていない。`newStuckReviewer` を closure からパッケージ関数に出したのは、2 つのテストで使うため。名前の衝突は無い

## 新しい指摘

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L15 | LOW | Documentation in code | 書き直した「`--config` を繰り返す」の文に、事実と合わない所が 2 つある。(a) help、`ReseatParams.HintConfigPath` の doc、help を固定する test は、`read` と `send` を「ralph.toml を読まない」と言う。`newOrgSendCmd` は `newOrgRuntimeAt`、`newOrgReadCmd` は `newOrgRuntime` を通り、どちらも `resolveOrgConfig` で ralph.toml を読んで検証する(読めない ralph.toml なら `org: load config` で落ちる)。help が「reads no ralph.toml」と書く verb(`director stop`・`director detach` など)は、この経路を通らない別の作りで、director の prompt の test も、`--config` を外すのは `stop` と `status` だけだと書く。DECISION の `send` が使う `[org]` の値は見つけていない(`Send` は `verbs.go` の中で `o.Config` を読まない。`deadman_minutes` を読む watch gate は TASK だけ)ので、動作の害は見つけていない。直すのは「読まない」を「その値で動作が変わらない」にすること。L12 の私の書き方(「`send` と `read` は ralph.toml を読まないので害は無い」)が出どころで、`newOrgSendCmd` を読まずに書いた。(b) 「reseat・stop・spawn・start のコマンドは、出力と断りの中で `--config` を繰り返す」は、共通の callee が書く stop に当たらない。`startAll` は `Spawn` の error を `%w` で `res.Err` に入れて出し、`Spawn` の `failStepWithNote`(pane を閉じられなかった枝)と `inFlightSeatErr` は `stopSeatCommand` で `ralph org stop ...` を `--config` なしで書く。`HintConfigPath` の doc は「stopSeatCommand, say」でこの例外を書くが、help は書かない。L12 が挙げた `stopSeatCommand` の残り | `internal/cli/org_reseat.go` の `orgReseatHelp` の最後から 2 つ目の段落、`internal/org/reseat.go` の `ReseatParams.HintConfigPath` の doc、`internal/cli/org_reseat_test.go` の `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`、`internal/cli/org.go` の `newOrgSendCmd`・`newOrgReadCmd`・`resolveOrgConfig`、`internal/org/spawn.go` の `failStepWithNote`・`inFlightSeatErr`・`stopSeatCommand`、`reseat.go` の `startAll` | help の 2 文を、「reseat が自分で書く reseat・stop・spawn・start のコマンドは `--config` を繰り返す(`spawn` の error が名指す `ralph org stop` は繰り返さない)。出力の `ralph org read` と `ralph org send` は `--state-dir` だけで、ralph.toml の値で動作は変わらない」の形にし、doc と test の文も合わせる。`send` と `read` に `--config` を足すなら、`ReseatDecisionCommand` の引数と `orgReadCommandHint` の呼び元が動くので、文の直しより大きい |
| L16 | LOW | Error text | 断りの新しい文言「the stopped seats reseat can start again are ...」「the seats reseat can start again are ...」は、custom role の座席(role prompt の雛形が無く、`--prompt` で立てた座席)も並べる。`startable()` は `reseatTargets` の述語で、`reseatCustomRoleErr` は対象が決まったあとの別の検査なので、断りが出す `--seat leader --seat <custom> ...` は対象の選び方を通り、次の実行で「its role ... has no role prompt template」に断られる。その断りは手で立てる手順と、`--seat` で他の座席だけ名指す手順を言うので、1 往復で済む。旧文言「its stopped seats are」は何も約束せず、新しい文言が約束を足した。L11 と同じ型(出した打ち方が断られる)が 1 段奥に残っている。コードを読んで導いたもので、実行して確かめてはいない | `reseatNoActiveSeatErr`、`reseatCustomRoleErr`、`PlanReseat` の手順 4 の doc comment(custom role の断りは対象が決まったあと) | 並べる座席から custom role を落とす(`RenderRolePrompt` の ok で見る。`reseatCustomRoleErr` と同じ判定)か、文言を「--seat が受け付ける座席」に戻す |

指摘にしなかったもの:

- `others` の「none of them stopped」は、名指した座席のどれも stopped でない、の意味で読める。手で止めた座席が別にあっても文は偽にならない。
- 手順 4 の doc comment、`reseatTargets` と `reseatNoActiveSeatErr` の doc、`touched` の doc は、新しい述語と合っている。`startable` の doc の「(D2, P7)」は、同じファイルの既存の plan 記号の引き方に合わせてある。L6 の `docs/plans/active/...` の注釈は増えていない。
- L13(triage を番号で引く test の comment)は、この差分で増えても減ってもいない。新しいテストの comment に番号の引用は無い。

### Tech debt の置き換え

cycle 2 の「Tech debt identified」の表の L11 の行は要らなくなった(直った)。L12・L13 の行の L12 は L15 に置き換わり、L16 が新しい。2 回目の `/sync-docs` が register へ足すとき、持ち越しは L2・L3(b)・L4・L5・L6・L10・L13・L14・L15・L16 の 10 件になる。L15 と L16 は文の直しで、`orgReseatHelp` か `reseatNoActiveSeatErr` の次の変更で払う。

### Recommendation(この差分)

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW は 10 件(持ち越し 8 件の L2・L3(b)・L4・L5・L6・L10・L13・L14 と、新しい L15・L16)。L11 と L12 は閉じた
- 直しの中身(`startable()` の共通化、断りの文言、テスト)は、述語が `reseatTargets` と同じ式で、断りの座席が対象の選び方で断られない点を読んで確かめた。新しい穴は L16 だけで、reseat が止めず記録せずに断る側に出る
- 私が cycle 2 の L12 で書いた「`send` と `read` は ralph.toml を読まない」は誤りだった。直しはその推奨のとおりに書かれ、help と test に入った。L15 (a) はその訂正
- Follow-ups: L15 と L16 は文の直しだけだが、Go のソースの文字列なので、直すならパイプラインは `/self-review` からやり直す。上限(2)に届いているので、直すなら上限を上げる

## cycle 3 の差分(def8d2a3)

- Scope: `git show def8d2a3` の 2 ファイル(`internal/org/reseat.go` の +43/-3 と、新しい `internal/org/reseat_start_test.go` の 191 行)を全部読み、`startAll`・`finish`・`reseatRetryCommand`・`ReseatResult` と `ReseatSeatOutcome` の doc を周りまで読んだ。判定の前提になる `Spawn` の idempotent の戻り(`spawn.go` の 2 か所と `idempotentRespawn`)、`Roster`、`spawned` event の組み立て、`StartFeature` の `Spawn` 呼び、`orgReseatLeftLine` と `printOrgReseatResult` も読んだ。0468d96f から HEAD までの Go の差は def8d2a3 の 2 ファイルだけ。同じ範囲の skill 8 ファイルと README の差(31f77c53)は sync-docs の文書で、`/sync-docs` の範囲なので読んでいない
- 実行したもの: `go test -count=1 -run 'TestReseat_ASeatAnotherCommand|TestReseat_TheLeaderAnotherCommand|TestReseatIdempotentMismatch' ./internal/org/` を 1 回(全部 PASS、0.575 s)と、2 ファイルへの `gofmt -l`(出力なし)。`internal/org` 全体のテストと静的解析は走らせていない
- ID: 新しい指摘は `L17` と `L18`。L6 と L13 は箇所が増えた。行番号は動くので、関数名を主にする

### cross-review(cycle 2)の ACTION_REQUIRED の判定

| ID | 判定 | 根拠 |
| --- | --- | --- |
| AR(他のコマンドが別の設定で立てた座席を、reseat が `started` と報告する) | 直った。穴は L18 の 1 つだけ残る(`--driver`・`--model` を渡さなかった reseat の打ち直し) | 下の「idempotent の検出」 |

### idempotent の検出(def8d2a3)

- 判定が走る条件: `spawn.Err == nil` かつ `Outcome == Idempotent` のときだけで、`spawned` の戻りと失敗の戻りは前のまま。`Spawn` が idempotent を返すのは 2 か所(lock の前の `existing.Event == EventSpawned` と、lock の中で roster を読み直した同じ条件)で、どちらも最新の state event が `spawned` の座席を返す。reseat が止めた座席の最新は `stopped`、`plan.Start` に入るが active でない座席の最新は `stopped` か `spawn_failed` などで、`spawned` ではない。つまり idempotent で戻るのは、reseat の検査か stop のあとに他のコマンドが立てた座席だけで、doc comment の説明は成り立つ。reseat 自身が同じ座席を 2 回立てる道は無い(`plan.Start` の `SeatID` は 1 回ずつ)
- 比べる値: `got` は `Roster` が最新の state event から作る。`spawned` event は `Role: p.Role, Driver: p.Driver, Model: p.Model`(命じた値。`spawn.go` の event の組み立て)を書き、`ReseatSeat` の `Role`・`Driver`・`Model` は台帳の記録か `--driver`・`--model` で、同じ語彙になる。`Model` が `""` の両側を等しいとすることは、`TestReseatIdempotentMismatch` の表が固定する。leader は `StartFeature(start).Spawn` が `o.Spawn(s.leader)` の戻りをそのまま持つので、`Seat` が入っている
- 失敗のあと: 不一致の座席は `start_failed` で `return` する。あとの座席は stop 済みのまま(`stopped`)で、`finish` の `Retry` に入る。前の座席は `started` で、leader が対象でなければ DECISION が名指す。`attempted` が true なので `reseated` は記録される。`Changed` は付けない(L18)。CLI は変えていない: `printOrgReseatResult` は `started` でない座席を `start failed:` で出し、その座席が走っていることは error(main が stderr に出す)にしか書かれない
- 同じ設定のとき: 前のとおり `started` に数え、DECISION に入る。他のコマンドが立てた座席を「reseat が立て直した」と DECISION に書くことになるが、変えていない振る舞いで、新しい test が固定する
- テスト: `start_failed` を期待する 2 本(`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`)は、直す前のコード(idempotent を `started` に数える)なら落ちる。同じ設定の 1 本(2 枝)は前のコードでも通り、「変えていない」ことの固定。表の 1 本は関数が新しい。`intrudeBeforeStart` は `TestReseat_ASeatChangedBeforeItsStop` の closure と同じ形(`done` で 1 回、`t.Cleanup` で nil に戻す)。`reseat_test.go` と `reseat_start_test.go` に `t.Parallel` は無く、フックの共有で競合しない。`beforeStart` の doc は `beforeStopCheck` の隣にある
- 周りの文: `ReseatSeatStartFailed` の doc、`Reseat` の手順 4 の doc、`startAll` の doc は 3 か所とも直されている。`ReseatResult.Err` と `Retry` の doc は、不一致の座席を「失敗した start」と「started でない座席」に含めて読めるので、偽にならない(`Changed` の doc は L18)

### 前の指摘のうち def8d2a3 が触れた所

| ID | 判定 | 根拠 |
| --- | --- | --- |
| L2・L3(b)・L14・L15・L16 | 残っている。触っていない | `org_reseat.go`、`feature.go`、`reseatNoActiveSeatErr`・`reseatCustomRoleErr` は def8d2a3 の差分に無い。新しい error は `ralph org` のコマンドを書かないので、L15 (b) の「共通の callee のコマンド」を増やさない |
| L4 | 残っている | 新しい関数は `reseatSeat` と `ReseatSeat` の組に触れない(`reseatSeat` は `reseat.go:755`、型は `reseat.go:90`) |
| L5 | 残っている | `PlanReseat` は 147 行のまま。`startAll` は 39 行になった |
| L6 | 残っている。箇所が増えた | `reseat_start_test.go` の先頭の comment が `docs/plans/active/2026-11-12-org-reseat.md` を引き(6 か所目)、`/pr` が archive へ動かす |
| L10 | 残っている。同じ型の部分失敗が 1 つ増えた | 「座席が他のコマンドの設定で走っている」ことと打ち直しは、読まれない stdout と stderr、`reseated` の `start_failed` にしか無い。help の 1 文で足りる点は前のまま |
| L13 | 残っている。箇所が増えた | `reseat_start_test.go` の先頭の comment が「cross-review cycle 2, finding 1」と triage を番号で引く(5 か所目)。triage は次の `/cross-review` が同じファイルを書き直すので、「cycle 2 の finding 1」は読めなくなる。コミット(def8d2a3)で引くか、番号を落とす。L6 と同じ回で払える |

## 新しい指摘(cycle 3)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L17 | LOW | Naming | `reseatIdempotentMismatch` は `error` を返すのに、名前は `if reseatIdempotentMismatch(...)` と書ける述語の形をしている。同じファイルの `error` を返す関数と method は `reseatChangedErr`、`reseatWorktreeRecordErr`、`reseatNoBindingErr`、`reseatNoActiveSeatErr`、`reseatCustomRoleErr` で、どれも `Err` で終わる。呼び元は `if err := ...; err != nil` と書くので取り違えは起きないが、doc comment の 1 行目「is the error of ...」が名前の足りない所を補っている。名前は `reseat.go` に 6 か所(定義、呼び 1 か所、doc comment 4 か所)、テストに 3 か所(呼びと `Fatalf` 2 か所)あり、テスト名にも入っている | `internal/org/reseat.go` の `reseatIdempotentMismatch`(定義、`startAll` の呼び、`ReseatSeatStartFailed` と `Reseat` の手順 4 と `startAll` の doc)、`internal/org/reseat_start_test.go` の `TestReseatIdempotentMismatch` | `reseatIdempotentMismatchErr` に改名する。テスト名も合わせる。9 か所と test 名 |
| L18 | LOW | Exception handling(printed retry) | 他のコマンドが別の設定で立てた座席の打ち直しが、`--driver`・`--model` を渡さなかった reseat では、走っている座席を同じ設定で立て直すだけになる。`reseatRetryCommand` は `started` でない座席をすべて入れ、`Changed` の座席だけ外す。不一致の座席は `start_failed` で `Changed` が空なので入る。(a) `--driver`・`--model` を渡した reseat では、打ち直しは利用者が頼んだ設定への移行で、意味がある(test が固定する)。(b) 渡さなかった reseat では、打ち直しの `--seat <id>` は、台帳の最新の `spawn_started` か `spawned` の設定(`ReseatSeat` の `Driver`・`Model` の doc)で立てる。その最新は他のコマンドが立てた座席のものなので、座席はもうその設定で走っている。打ち直しは、その座席(不一致が leader なら org の leader)を止めて同じ設定でもう一度立てるだけで、他のコマンドの座席が TASK を持ち始めていれば途中のものを捨てる。出力は「to start the seats left, run:」の下にこの 1 行を置き、座席が走っていることは error の文にしか無い。止める側の同じ状況(`Changed`、FR-5a)は、座席を打ち直しから外して扱っている。コードを読んで導いたもので、実行して確かめてはいない | `internal/org/reseat.go` の `reseatRetryCommand`、`startAll` の不一致の枝、`ReseatResult.Changed` の doc、`ReseatSeat` の `Driver`・`Model` の doc、`internal/cli/org_reseat.go` の `printOrgReseatResult`、`orgReseatLeftLine` | orchestrator の案(Known gap)に同意する。書く文は「打ち直しは動く」ではなく「`--driver`・`--model` なしの reseat では、打ち直しは走っている座席を同じ設定で立て直すだけ」。直すなら 2 行: 不一致の枝で `!seat.Overridden` のとき `r.res.Changed = seat.SeatID` を立て(`reseatRetryCommand` が外す。`orgReseatLeftLine` は `start_failed` の座席に行を出さないので、出力の他の行は変わらない)、`Changed` の doc に start 時の分を足す |

指摘にしなかったもの:

- `startAll` の 2 つの `spawn.Err == nil && ...` は、1 つにまとめられそうに見えるが、不一致の枝は自分の error 文を持ち、後ろの枝は「did not start again」で包むので、まとめると分岐が増える。今の形で読める。
- error 文は、1 つの値だけが違うときも role・driver・model の 3 つを両側に並べる。1 度読むだけの文で、違いは見比べて分かる。
- `reseatIdempotentMismatch` は `ReseatPlan` を値で受けて `ID` と `OrgID` だけを使う。コピーは軽く、呼び元が 1 つなので直さない。
- scope・cwd・permission mode は比べない。doc comment と error 文は role・driver・model だけを名指しており、書いてあることと中身は合っている。
- 「started no seat after it」を、不一致の座席のあとにも座席がある reseat で確かめる test は無い。test の範囲は `/test` が見る。
- 文書の hand-off(`/sync-docs`): help の「Right before it stops a seat, reseat reads the ledger again. When another command stopped or started that seat meanwhile, reseat leaves it as it is and ends there」、spec の FR-5a・FR-6、plan の P6 は、stop の前の確かめだけを言い、start のときの検出を言わない。`startAll` が他のコマンドの座席を `start_failed` にすることは、どの文書にも書かれていない。

### ブランチの今の指摘(12 件)

CRITICAL・HIGH・MEDIUM は無い。LOW は次の 12 件で、本文は上の同じ ID の行にある。

| ID | Area | 一言 |
| --- | --- | --- |
| L2 | Maintainability | dry run の herdr probe が gate の前半を写している |
| L3(b) | Error text | leader を対象にしない reseat の `[reviewer]` の断りが leader の task を言う |
| L4 | Naming | 関数 `reseatSeat` と型 `ReseatSeat` が頭文字の大小だけで違う |
| L5 | Readability | `PlanReseat` が 147 行 |
| L6 | Maintainability | archive へ動く plan を指す注釈が 6 か所(今回 1 か所増えた) |
| L10 | Exception handling | 出力を読む手が居ない実行の部分失敗の account が stdout に偏る |
| L13 | Maintainability | triage を番号で引く test の comment が 5 か所(今回 1 か所増えた) |
| L14 | Consistency (merge) | `reseat` が #310 の codex hook の通知を出さない |
| L15 | Documentation in code | `--config` の help の文に、`send`・`read` の読む ralph.toml と、共通の callee が書く stop の 2 つの例外の食い違い |
| L16 | Error text | 断りが custom role の座席も「reseat が立て直せる座席」に並べる |
| L17 | Naming | `error` を返す `reseatIdempotentMismatch` に `Err` の接尾辞が無い |
| L18 | Exception handling | `--driver`・`--model` なしの reseat で、不一致の座席の打ち直しが走っている座席を立て直すだけになる |

### Tech debt の置き換え

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L17: `reseatIdempotentMismatch` の名前 | 述語に読める名前が `error` を返す | 9 か所と test 名の改名で、Go のソースを変えるため、直すとパイプラインが `/self-review` からになる。LOW | `reseatIdempotentMismatch` か `startAll` の次の変更 | 本 report の L17 |
| L18: 不一致の座席の打ち直し(`--driver`・`--model` なし) | 打ち直しを走らせると、走っている座席を同じ設定で止めて立て直す。TASK を持ち始めていれば途中のものを捨てる | 他のコマンドの割り込みのときだけで、出力の error が座席が走っていることを言う。2 行の直しだが、Go のソースを変える | `reseatRetryCommand` か `startAll` の次の変更。PR 本文の Known gap にも書く | 本 report の L18 |
| L6・L13(箇所が増えた) | 前の report の通り | 前の通り | 前の通り | `docs/tech-debt/README.md` の「Eleven small findings of docs/spec-org-reseat」の行 |

`docs/tech-debt/README.md` の「Eleven small findings of docs/spec-org-reseat」の行は、cycle 2 の `/sync-docs`(31f77c53)が足したもので、L2 から L6、L10、L13 から L16 を持つ。この cycle の `/sync-docs` が足す直し:

- 見出しの件数を 13 にし、L17 と L18 を (l) と (m) に足す。
- (e) の「five comments」を 6 にする(`reseat_start_test.go` が増えた)。(h) の「four places」を 5 にする(「cross-review cycle 2, finding 1」)。
- (h) から (k) の前置き「after the pipeline reached its cap (`max_pipeline_cycles` 2)」は、cross-review の ACTION_REQUIRED の直しのために cap を 3 に上げたので、そのままでは偽になる。「cap を上げた回でも、文と名前の直しは 1 回の再実行に値しない LOW として払わなかった」の形に書き直す。
- ファイルの大きさの行の `reseat.go` は、def8d2a3 で 1,347 行(+40)になった。新しい `reseat_start_test.go` は 191 行で、800 行の下。

### Recommendation(この差分)

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW は 12 件(持ち越し 10 件の L2・L3(b)・L4・L5・L6・L10・L13・L14・L15・L16 と、新しい L17・L18)。L6 と L13 は箇所が 1 つずつ増えた
- 直しの中身は、idempotent かつ role・driver・model が違うときだけ失敗にし、`spawned` の戻りと同じ設定の idempotent は前のままで、reseat 自身が idempotent を引く道が無いことを、`Spawn` の 2 つの idempotent の条件と `plan.Start` の組み立てから読んで確かめた。新しい穴は L18 だけで、reseat が何も壊さず、error が座席の状態を言う側に出る
- orchestrator の判断(L18 を直さず Known gap にする)に同意する。違うのは理由の言い方。「打ち直しは動く」は、打ち直しが走っている座席(leader なら org の leader)を止めて立て直すことを隠す。`--driver`・`--model` を渡した reseat では打ち直しが利用者の頼んだ移行なので、gap は渡さなかった reseat だけ。直す場合の 2 行は L18 の行にある
- Follow-ups: L17 は改名、L18 は 2 行で、どちらも Go のソースなので、直すならパイプラインは `/self-review` からやり直す。直さないなら register へ(上の表)。L13 は次の `/cross-review` の前に番号を落とすと安い(その cross-review が triage を書き直すと、番号の引用は読めなくなる)。`cycle-count.json` は cap の引き上げで cycle 2、cap 3 のまま(skill の文面から。未確認)で、`--cycle auto` の insight event は 2 と書く

## cycle 3 の差分(df8f674e)

- Scope: `git show df8f674e` の 2 ファイル(`internal/org/reseat.go` が 38 行、`internal/org/reseat_start_test.go` が 68 行の変更)を全部読み、`startAll`・`stopAll`・`finish`・`reseatRetryCommand`・`ReseatResult` の `Err`・`Changed`・`Retry` の doc、`Reseat` の手順 4 の doc、`reseatIdempotentMismatchErr`、`checkReseatFlags`、`plan.Stop` の組み立てを周りまで読んだ。CLI は `orgReseatHelp`、`printOrgReseatResult`、`orgReseatLeftLine`(`internal/cli/org_reseat.go`)を読み、`ReseatResult.Changed` を読む所を grep した(`finish` と `printOrgReseatResult` の 2 か所だけ)。偽になる文が無いかを見るため、org skill の reseat の出力の段落(`.claude/skills/org/SKILL.md`)と spec の FR-5a も読んだ。`internal/cli` のテストは `left stopped:` と `start failed:` の行を固定するが、`left as it is:` の行(止める側の `Changed`)を固定するものは見つからなかった(grep。テストの範囲は `/test` が見る)
- 実行したもの: `go test -count=1 -run 'TestReseat_ASeatAnotherCommand|TestReseat_TheLeaderAnotherCommand|TestReseatIdempotentMismatchErr|TestReseat_ASeatChangedBeforeItsStop' ./internal/org/` を 1 回(`ok`、1.926 s)と、2 ファイルへの `gofmt -l`(出力なし)。`internal/org` 全体のテストと静的解析、`ralph org reseat` の実行は無い。出力の指摘(L19)はコードを読んで導いたもの
- ID: 新しい指摘は `L19`。行番号は動くので、関数名を主にする

### L17 と L18 の判定

| ID | 判定 | 根拠 |
| --- | --- | --- |
| L17 | 直った | `reseatIdempotentMismatch` は、定義、`startAll` の呼び、`ReseatSeatStartFailed`・`Reseat` の手順 4・`startAll` の doc、テストの呼びと `Fatalf` 2 か所、テスト名のすべてが `...Err` になった。`Err` の付かない名前は Go のソースに残っていない(grep。残るのはこの report の本文だけ)。`reseatChangedErr` などと並ぶ。`TestReseatIdempotentMismatchErr` が通る |
| L18 | 直った。出力に 1 行の穴が残る(L19) | 下の「`Changed` の設定」 |

### `Changed` の設定(df8f674e)

- 条件: 不一致の枝(`spawn.Err == nil`、idempotent、role・driver・model が違う)で `!seat.Overridden` のとき `r.res.Changed = seat.SeatID` を立て、`return` する。`Changed` を書くもう 1 か所は `stopAll` で、そこが false を返すと `startAll` は走らないので、`startAll` が前の値を上書きする道は無い
- 打ち直しから外れる座席: `reseatRetryCommand` が `s.SeatID != skip` で外す。不一致の座席より後ろの座席は `stopped` か `not_started` のままで入り、前の座席は `started` なので入らない。leader だけを対象にした reseat は `left` が空で、`Retry` は `""` になる(test が固定する)。`finish` の DECISION は `started` の座席から作るので、この直しで変わらない
- `--driver`・`--model` を渡した reseat: `Overridden` が true になるのは `reseatSeat` の 1 か所で、`checkReseatFlags` は `--driver`・`--model` を `--seat` がちょうど 1 つのときだけ通す。だから `left` は 1 座席で、`reseatRetryCommand` の `len(left) == 1 && left[0].Overridden` と同じ座席を指す。この座席は `Changed` に入らず、打ち直しは `--driver`・`--model` つきで、座席を止めて頼まれた設定で立てる(`TestReseat_ASeatAnotherCommandStartedOnOtherSettings` が固定する)
- `Changed` を読む所: `finish`(`Retry`)と `printOrgReseatResult`(`orgReseatLeftLine`)。`orgReseatLeftLine` の「left as it is」の枝は `outcome == NotStopped && seat == changed` で、`start_failed` の座席に行を出さない。start のときは stop が全部済んでいて(`stopAll` が true を返してから `startAll`)、`NotStopped` の座席は無いので、止める側の `Changed` と start 側の `Changed` が同じ reseat に立つ道は無い
- doc: `ReseatResult.Changed` の 2 つの場合の説明、`Retry` の「(all but Changed)」、`startAll` と `reseatIdempotentMismatchErr` の doc は、コードと合う。`Err` の「a changed seat (Changed)」は 2 つの場合のどちらにも当たる。`ReseatSeatNotStopped` の「(ReseatResult.Changed names it)」は止める側の話で、偽にならない。`Reseat` の doc の結果の段落「the reseat that starts the ones not started (Retry)」は `Changed` の座席が外れることを書かないが、止める側で前からそうで、この直しで増えた偽ではない
- テスト: leader の test は `Changed == leader`、`Retry == ""`、DECISION なしを固定し、`OnOtherSettings` の test は `Changed == ""` を足した。新しい非 leader の test(`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`)は、直す前のコードなら `res.Changed != implementerSeatID` で落ちる。打ち直しを実際に走らせ、止める座席が無いこと、reviewer だけが立つこと、他のコマンドが立てた implementer が sonnet のまま `reseat=` の note なしで残ることまで見る。使う補助(`implementerSeatID`・`implementerRole`・`seatIDs`・`reseatOutcomes`・`retryParams`・`mustReseat`・`eventOf`・`assertReseated`)は存在する(grep と、test が通ること)

### 前の指摘のうち df8f674e が触れた所

| ID | 判定 | 根拠 |
| --- | --- | --- |
| L2・L3(b)・L4・L14・L15・L16 | 残っている。触っていない | `org_reseat.go`、`feature.go`、`reseatSeat`・`ReseatSeat`、`reseatNoActiveSeatErr`・`reseatCustomRoleErr` は df8f674e の差分に無い |
| L5 | 残っている | `PlanReseat` は触っていない。`startAll` は 42 行(+3)で、50 行の下 |
| L6・L13 | 残っている。箇所は増えていない | 追加行に `docs/plans/active/...` の引用も triage の番号の引用も無い(追加行を grep。`FR-5a` は書き直した doc の中の plan 記号で、同じファイルの既存の引き方) |
| L10 | 残っている。L19 と同じ型 | 「座席が他のコマンドの設定で走っている」ことが、stdout では `start failed:` の行にしか出ない点は前のまま。L19 はそのうち、打ち直しから外した理由が stdout に無い点 |
| ファイルの大きさ | `reseat.go` は 1,361 行(+14)、`reseat_start_test.go` は 237 行 | register の行の数字は `/sync-docs` が直す |

## 新しい指摘(cycle 3、df8f674e)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L19 | LOW | Consistency (printed output) | `--driver`・`--model` を渡さなかった reseat で、他のコマンドが別の設定で立てた座席を `Changed` にして打ち直しから外したのに、その理由が stdout のどの行にも出ない。止める側の同じ状況は `orgReseatLeftLine` の `left as it is: <座席> (another command stopped or started it meanwhile)` の行を持つが、start 側の座席は outcome が `start_failed` で、`orgReseatLeftLine` は `""` を返す。stdout に出るのは `start failed: <座席> (<役割>, <driver> <モデル>)` の行だけで、括弧の中は reseat が立てようとした設定(`orgReseatStartSeat(seat, "")`)であり、座席が走っている設定ではない。続く `to start the seats left, run:` の打ち直しにその座席は無い(leader だけの reseat では、`Retry` が空でこの案内自体が出ない)。座席が他のコマンドの設定で走っていて、reseat が触らなかったことは、error の文(main が stderr に出す)にしか無い。df8f674e より前は、打ち直しにその座席が入っていたので stdout は筋が通っていた(その打ち直しが L18 の害を起こした)。外したことで、stdout は座席が消えた理由を言わなくなった。org skill の失敗の段落は `left as it is:` を失敗のあとの行の 1 つに挙げ、読み手は stdout と stderr の両方を読む前提で書かれている。コードを読んで導いたもので、実行して確かめてはいない | `internal/cli/org_reseat.go` の `orgReseatLeftLine` と `printOrgReseatResult`、`internal/org/reseat.go` の `startAll`(不一致の枝)、`reseatRetryCommand`、`.claude/skills/org/SKILL.md` の reseat の出力の段落 | `orgReseatLeftLine` に枝を足す: `outcome == org.ReseatSeatStartFailed && seat.SeatID == changed` のとき `"left as it is: " + seat.SeatID + " (another command started it during the reseat, so it keeps running as that command started it)"` の形の行。doc の「A seat whose stop or start was tried has its line already, and gets \"\"」を、この枝を例外として直す。`printOrgReseatResult` は `start failed:` の行も出し続けるので、同じ座席が 2 行になる。それが気になるなら `start failed:` の行を出さず、この行だけにする。CLI の test を 1 本足す。Go のソースを変えるので、直すならパイプラインは `/self-review` からになる。直さないなら register へ |

指摘にしなかったもの:

- `startAll` の `if !seat.Overridden` の所に理由の 1 行が無い。理由(`--driver`・`--model` なしの打ち直しは、台帳の最新の spawn の設定、つまり座席がもう走っている設定で立てる)は `reseatIdempotentMismatchErr` の doc にあり、`startAll` の doc と `ReseatResult.Changed` の doc は条件を書く。理由は引ける場所にあるので直さない。ただし、理由の文が「error を作る関数」の doc にあり、`Changed` を立てる `startAll` の doc に無い置き場所は、関数の責務と少しずれる。
- `!seat.Overridden`(`startAll`)と `len(left) == 1 && left[0].Overridden`(`reseatRetryCommand`)は、「打ち直しが `--driver`・`--model` を持つか」を 2 か所で判定する。`checkReseatFlags` が `--driver`・`--model` を `--seat` 1 つに限るので同じ座席を指す。この前提が緩めば食い違うが、今は成り立ち、`OnOtherSettings` の test が両側を通す。
- `Changed` という名前は、止める側(記録が変わっていた)と start 側(他のコマンドが立てた)の両方を「他のコマンドが変えた座席」で言えるので、改名は要らない。`reseated` event の ` changed=<seat>:<driver>/<model>` の語とは意味が違うが、前からの並びで、この直しで増えていない。
- 不一致の error の文は、`--driver`・`--model` なしの reseat のとき、座席を頼む設定に動かす打ち方(`--seat <座席> --driver ... --model ...`)を言わない。打ち直しの案内が無くなった分、その手が要るなら足す価値はあるが、df8f674e より前も言わなかった(前の打ち直しは今の設定で立て直すだけだった)ので、新しい穴ではない。
- 文書の hand-off(`/sync-docs`): help の「Right before it stops a seat, reseat reads the ledger again. When another command stopped or started that seat meanwhile, reseat leaves it as it is and ends there」、org skill の失敗の段落(「座席の記録が変わっていた」「`left as it is:`」「出たコマンドをそのまま打つ」)、spec の FR-5a・FR-6、plan の P6 は、止める前の確かめだけを言う。start のときの検出(他のコマンドが別の設定で立てた座席を `start_failed` にし、`--driver`・`--model` なしの reseat では打ち直しから外す)はどの文書にも無い。どの文も偽にはなっていない。def8d2a3 の節で渡した点に、打ち直しから外れる分が増えた。

### ブランチの今の指摘(11 件)

CRITICAL・HIGH・MEDIUM は無い。LOW は次の 11 件で、本文は上の同じ ID の行にある。

| ID | Area | 一言 |
| --- | --- | --- |
| L2 | Maintainability | dry run の herdr probe が gate の前半を写している |
| L3(b) | Error text | leader を対象にしない reseat の `[reviewer]` の断りが leader の task を言う |
| L4 | Naming | 関数 `reseatSeat` と型 `ReseatSeat` が頭文字の大小だけで違う |
| L5 | Readability | `PlanReseat` が 147 行 |
| L6 | Maintainability | archive へ動く plan を指す注釈が 6 か所 |
| L10 | Exception handling | 出力を読む手が居ない実行の部分失敗の account が stdout に偏る |
| L13 | Maintainability | triage を番号で引く test の comment が 5 か所 |
| L14 | Consistency (merge) | `reseat` が #310 の codex hook の通知を出さない |
| L15 | Documentation in code | `--config` の help の文に、`send`・`read` の読む ralph.toml と、共通の callee が書く stop の 2 つの例外の食い違い |
| L16 | Error text | 断りが custom role の座席も「reseat が立て直せる座席」に並べる |
| L19 | Consistency (printed output) | 他のコマンドが立てた座席を打ち直しから外した理由が stdout に出ない |

L17 と L18 は閉じた。

### Tech debt の置き換え

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L19: start 時の `Changed` の座席の stdout の行 | `--driver`・`--model` なしの reseat で、打ち直しから外した座席が stdout で消える。理由は stderr の error にある | 他のコマンドの割り込みのときだけで、error が座席の状態を言う。`orgReseatLeftLine` の枝 1 つと test 1 本だが、Go のソースを変えるので、直すとパイプラインが `/self-review` からになる。LOW | `orgReseatLeftLine` か `printOrgReseatResult` の次の変更 | 本 report の L19 |
| L6・L13 | 前の report の通り(箇所は増えていない) | 前の通り | 前の通り | `docs/tech-debt/README.md` の「Eleven small findings of docs/spec-org-reseat」の行 |

L17 と L18 は閉じたので、register の行は要らない。def8d2a3 の節の「Tech debt の置き換え」の表の L17・L18 の行は、この表が置き換える。

`docs/tech-debt/README.md` の「Eleven small findings of docs/spec-org-reseat」の行へ、この cycle の `/sync-docs` が足す直しは、def8d2a3 の節の 4 項目のうち次のとおり変わる:

- 見出しの件数を 13 にして L17 と L18 を (l) と (m) に足す、は取り下げる。L17 と L18 は閉じたので足さない。L19 を register に足すなら (l) として、見出しの件数は 12 になる。
- (e) の「five comments」を 6 に、(h) の「four places」を 5 にする。変わらない(`reseat_start_test.go` と「cross-review cycle 2, finding 1」の箇所は def8d2a3 で増えたもの)。
- (h) から (k) の前置きの「after the pipeline reached its cap」の書き直しも変わらない。
- ファイルの大きさの行の `reseat.go` は 1,361 行(+14、df8f674e のあと)。`reseat_start_test.go` は 237 行で、800 行の下。

### Recommendation(この差分)

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW は 11 件(持ち越し 10 件の L2・L3(b)・L4・L5・L6・L10・L13・L14・L15・L16 と、新しい L19)。L17 と L18 は閉じた
- 直しの中身は、`Changed` を立てる条件が `!seat.Overridden` で、`--driver`・`--model` を渡した reseat の打ち直しは前のまま(座席を止めて頼まれた設定で立てる)であること、`stopAll` が書く止める側の `Changed` と同じ reseat に立たないことを、コードと test から読んで確かめた。新しい穴は L19 だけで、reseat が何も壊さず、stdout の 1 行が足りない側に出る
- 改名は定義・呼び・doc・test のすべてに及んでいて、`Err` の付かない名前は残っていない
- Follow-ups: L19 は枝 1 つと test 1 本で、Go のソースなので、直すならパイプラインは `/self-review` からやり直す。cycle 3 なので、上限に届いているはず(未確認)で、直すなら上限を上げる。直さないなら register へ(上の表)。L13 は次の `/cross-review` の前に番号を落とすと安い(前の節のとおり)

## cycle 3 の差分(b142944d)

- Scope: `git show b142944d` の 2 ファイル(`internal/cli/org_reseat.go` の `orgReseatHelp` が +8/-2 行、`internal/cli/org_reseat_test.go` が +6 行)を全部読んだ。新しい help の各文を、`startAll`(不一致の枝と `!seat.Overridden`)、`reseatIdempotentMismatchErr`、`ReseatResult` の `Changed`・`Retry` の doc、`reseatRetryCommand`、`reseatSeat`(`Overridden` を立てる 1 か所)、`checkReseatFlags`、`stopAll`(`r.attempted` を立てるより前の `Changed`)、`finish`、`reseatLifeSeat.recorded`、CLI の `printOrgReseatResult` と `orgReseatLeftLine` に当てた。`--driver`・`--model` つきの打ち直しが座席を止めて頼まれた設定で立てる点は、`TestReseat_ASeatAnotherCommandStartedOnOtherSettings` が打ち直しを実際に走らせて固定するのを読んだ。周りの既存の文は、同じ 2 段落(「A stop that fails」から「until it has written its record」まで)と、`Reseat` の doc の手順 5 を読んだ。`bab7b33a` は verify の report だけで、読んでいない
- 実行したもの: `go test -count=1 -run 'OrgReseatHelp' ./internal/cli/` を 1 回(`ok`、0.441 s)と、`orgReseatHelp` の行の長さを awk で測ったこと。`ralph org reseat` の実行と `internal/org` のテストは無い。文の判定はコードと test を読んで導いたもの
- ID: 新しい指摘は `L20` と `L21`。行番号は動くので、関数名を主にする

### 新しい help の文の判定

| 文(要点) | 判定 | 根拠 |
| --- | --- | --- |
| 打ち直しのコマンドは「他のコマンドが止めた、または立てた座席」を外し、ほかに座席が残らなければ何も出ない(「(see below)」つき) | 本当 | `reseatRetryCommand` は `Outcome != started && SeatID != skip` の座席だけを数え、`skip` は `finish` が渡す `res.Changed`。`Changed` を立てるのは `stopAll` の止める前の確かめ(「止めた、または立てた」)と、`--driver`・`--model` なしの start 時の不一致(「立てた」)の 2 か所。`left` が空なら `""` で、`printOrgReseatResult` は `res.Retry != ""` のときだけ `to start the seats left, run:` を出すので、何も出ない。無条件の「leaves out」は、`--driver`・`--model` つきなら start 時の座席が外れない点と食い違って読めるが、「(see below)」がその段落を指し、そこで分けている |
| start する座席を他のコマンドが別の role・driver・model で立てていたら、reseat はそこで終わる | 本当 | `startAll` は idempotent の start が `reseatIdempotentMismatchErr` を返すと `ReseatSeatStartFailed` にして `return` する。後ろの座席は起動しない。「too」は直前の止める前の確かめの文に合う。同じ設定なら error は nil で reseat は続く(help は言わないが、偽の文は無い) |
| `--driver`・`--model` なしなら、その座席はそのままで、打ち直しのコマンドから外れる | 本当 | `!seat.Overridden` のとき `Changed` を立てる。座席は Spawn が idempotent で返しただけで触られない。`Overridden` は `reseatSeat` の 1 か所で、`--driver`・`--model` を渡した 1 座席だけに付く |
| `--driver`・`--model` つきなら、打ち直しのコマンドがその座席を止めて、頼まれた設定で立てる | 本当 | `Changed` は立たず、`left` はその 1 座席で、`reseatRetryCommand` の `len(left) == 1 && left[0].Overridden` が `--driver --model` を足す。座席は今 spawned なので打ち直しの `Plan.Stop` に入る。上の test が、打ち直しの `Stop` が implementer 1 つであること、codex gpt-x で立つこと、`reseat=` の note が付くことまで見る |

周りの既存の文:

| 文 | 判定 | 根拠 |
| --- | --- | --- |
| 「Right before it stops a seat, reseat reads the ledger again. When another command stopped or started that seat meanwhile, reseat leaves it as it is and ends there.」 | 本当 | `stopAll` の確かめ。`reseatChangedErr` の文と同じ話 |
| 「Every stop and start a reseat makes records reseat=<id>, and the reseat ends with one reseated event of the org, also when it ended early.」 | 偽になる場合が 1 つある(L21) | `finish` は `r.attempted` が true のときだけ `reseated` を書く。最初の座席の止める前の確かめで記録が変わっていた reseat は、`attempted` を立てる前に返るので、書かない。b142944d より前から、この文はそうだった |
| 「A stop that fails ends the reseat before any start, and a start that fails ends it there. The output then names the seats stopped, started, and left, and the reseat command that starts the rest, ... and reseat exits 1.」 | 本当 | 止める前の確かめで終わる reseat と start 時の不一致は、この文の「stop が失敗」「start が失敗」のどちらでもないが、同じ出力の形(`left ...:` の行か `start failed:`、打ち直しのコマンド、終了コード 1)になり、続く文が両方を説明する。start 時の不一致は outcome が `start_failed` なので、「start that fails」に入る |
| 「When the leader is not one of the seats and keeps running, the output ends with the `ralph org send` command of a DECISION ...」 | 本当 | `finish` の DECISION は `started` の座席から作り、b142944d の文はそこに触れない |

### test の pin

- 足した pin は 2 つ。1 つ目は打ち直しのコマンドが座席を外す文、2 つ目は止める前の確かめの既存の文に、start 時の 3 文を続けた塊。2 つ目の先頭の既存の文は、これまで pin が無かった。どちらも `strings.Fields` で空白を畳んだ help に対する `Contains` で、他の pin と同じ作り。help の折り返しを詰め直しても落ちない(L20 の直しが test を巻き込まない)
- pin は文面を固定するもので、座席が実際に外れるかは見ない。そちらは `internal/org` の test が見る(`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`、`...IsLeftOutOfTheRetry`、leader の test)。`printOrgReseatResult` は `res.Retry != ""` で分岐するだけなので、CLI に別の test を足す必要は薄い。pin の文は、どれも b142944d より前の help では落ちる(足した文が無かった)
- 2 つ目の pin の失敗の出力は help 全体なので、どの文が落ちたかは読み比べになる。ほかの複数文の pin と同じで、直さない

## 新しい指摘(cycle 3、b142944d)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L20 | LOW | Readability (help layout) | `orgReseatHelp` の 2 つ目の段落で、足した文を既存の行に差し込んだだけで、段落の残りを詰め直していない。`command stops it and starts it on them. Every stop` の行が 50 文字で、前後は 69 文字と 74 文字。次の語 `and` を足しても 54 文字なので、折り返しの幅(66 から 74 文字)に合わない短い行が `ralph org reseat --help` の段落の途中に出る。1 つ目の段落の足した行(`exits 1. That command leaves out ...` から `printed. When the leader ...`)は 66 から 71 文字で揃っている。表示の問題だけで、意味は変わらない | `internal/cli/org_reseat.go` の `orgReseatHelp`(「Two reseats of one ledger」の段落)。行の長さは `awk` で測った(上の行の前後は 69 と 74、ほかの行は 60 から 74) | 「Every stop」から段落の終わり(`fail; the output then names the reseat that starts the rest.`)までの 11 行ほどを 72 文字前後に詰め直す。pin は空白を畳むので test は変わらない。または足した文に 1 句を足して、その行を埋める |
| L21 | LOW | Documentation in code (help) | help の「the reseat ends with one reseated event of the org, also when it ended early」は、`Reseat` の doc の手順 5 にある条件(「when it stopped or started a seat (or tried to)」)を落としている。止める前の確かめで最初の座席の記録が変わっていた reseat は、何も止めず、立てず、`r.attempted` を立てる前に返るので、`reseated` を書かない(`finish` の `if r.attempted`)。`TestReseat_ASeatChangedBeforeItsStop` の「the first seat to stop was stopped meanwhile」が「no reseated」を固定している。最初の座席の確かめで manifest を読み直せなかった reseat(`stopAll` の最初の error)も同じ。この文は ecb84fa8 から変わっていないが、b142944d が足した「他のコマンドが止めた、または立てた座席で reseat は終わる」の 2 文のすぐ後ろに続くので、読み手は「止める前の確かめで終わった reseat も `reseated` を書く」と読む。V19 と同じ型(狭い場合にだけ偽になる help の文)で、偽になる場所だけが違う。コードを読んで導き、test の期待で裏を取ったもので、実行して確かめてはいない | `internal/cli/org_reseat.go` の `orgReseatHelp`、`internal/org/reseat.go` の `reseatRun.stopAll`(`r.attempted = true` より前の 2 つの return)と `finish`、`Reseat` の doc の手順 5、`internal/org/reseat_test.go` の `TestReseat_ASeatChangedBeforeItsStop`(「the first seat to stop was stopped meanwhile」) | 文を `Reseat` の doc に合わせる: 「..., and the reseat ends with one reseated event of the org once it has stopped or started a seat, also when it ended early」の形。pin は先頭の「and the reseat ends with one reseated event of the org」で止まるので、続きを足しても落ちない。Go のソースなので、直すならパイプラインは `/self-review` からやり直す。直さないなら register へ |

指摘にしなかったもの:

- 1 つ目の文の無条件の「leaves out」と、3 つ目・4 つ目の「`--driver`・`--model` つきなら外れない」は、並べると食い違って読めるが、「(see below)」が指す段落で分けているので、偽の文ではない。
- 「on another role, driver, or model」に、何と比べて「another」かの語が無い(error の文は「not the ... the reseat meant to start it on」と言う)。続く 2 文が、`--driver`・`--model` を渡したかどうかで分けて「them」を使うので、読み手は同じ参照先に行き着く。足すなら `than the reseat starts it on` の一句だが、直す価値は薄い。
- 役割だけが違う不一致(driver と model は同じ)で `--driver`・`--model` つきの打ち直しは、役割を台帳の最新の spawn(他のコマンドが立てた方、`reseatLifeSeat.recorded`)から取るので、同じ役割で立て直すだけになる。help は「them」を driver と model の意味でしか使っていないので偽ではない。同じ seat_id を別の役割で立てるには別のコマンドが要り、起きにくい。
- 「leaves out ... none is printed」の後ろに「その座席は stderr の error が名指す」とは書いていない。error の文は座席と立てた設定を言うので、help で足りないわけではない。

### 前の指摘のうち b142944d が触れた所

| ID | 判定 | 根拠 |
| --- | --- | --- |
| L19 | 残っている。help が start 時の検出を言うようになった分、読み手が `--help` で知る道はできた | 出力は変わらない: start 側の座席に `left as it is:` の行は出ず、`start failed:` の行と stderr の error だけが座席を言う。新しい help は「left as it is」という語を start 側にも使うが、出力にその語の行が出るのは止める側だけ。L19 の Recommendation(`orgReseatLeftLine` の枝)は前のまま有効で、それをやれば help の語と出力が合う |
| L2・L3(b)・L4・L5・L6・L10・L13・L14・L15・L16 | 残っている。触っていない | b142944d の差分は `orgReseatHelp` と、help の test 1 本だけ。追加行に `docs/plans/active/...` の引用も triage の番号の引用も無い(L6・L13 は増えない) |
| ファイルの大きさ | `org_reseat_test.go` は 858 行(+6)、`org_reseat.go` は 516 行、`reseat.go` は 1,361 行のまま | register の行は `org_reseat_test.go` を 852 行と書く。数字は `/sync-docs` が直す |

文書の hand-off(`/sync-docs`): df8f674e の節で渡した「start のときの検出はどの文書にも無い」のうち、help の分は b142944d で入った。残るのは org skill の reseat の出力の段落(`.claude/skills/org/SKILL.md` と `.agents`・`templates/base` の 4 組。「座席の記録が変わっていた」だけを言い、start 時の不一致も、座席が残らなければコマンドが出ないことも言わない。「出たコマンドをそのまま打つ」は、コマンドが出ない場合に当てはまらない)、spec の FR-5a・FR-6、plan の P6。どの文も偽にはなっていない。

### ブランチの今の指摘(13 件)

CRITICAL・HIGH・MEDIUM は無い。LOW は次の 13 件で、本文は L20・L21 が上の表、ほかは同じ ID の行にある。

| ID | Area | 一言 |
| --- | --- | --- |
| L2 | Maintainability | dry run の herdr probe が gate の前半を写している |
| L3(b) | Error text | leader を対象にしない reseat の `[reviewer]` の断りが leader の task を言う |
| L4 | Naming | 関数 `reseatSeat` と型 `ReseatSeat` が頭文字の大小だけで違う |
| L5 | Readability | `PlanReseat` が 147 行 |
| L6 | Maintainability | archive へ動く plan を指す注釈が 6 か所 |
| L10 | Exception handling | 出力を読む手が居ない実行の部分失敗の account が stdout に偏る |
| L13 | Maintainability | triage を番号で引く test の comment が 5 か所 |
| L14 | Consistency (merge) | `reseat` が #310 の codex hook の通知を出さない |
| L15 | Documentation in code | `--config` の help の文に、`send`・`read` の読む ralph.toml と、共通の callee が書く stop の 2 つの例外の食い違い |
| L16 | Error text | 断りが custom role の座席も「reseat が立て直せる座席」に並べる |
| L19 | Consistency (printed output) | 他のコマンドが立てた座席を打ち直しから外した理由が stdout に出ない |
| L20 | Readability (help layout) | help の段落の途中に 50 文字の短い行 |
| L21 | Documentation in code (help) | `reseated` を「途中で終わっても書く」とする文が、何も止めず立てずに終わる場合を含む |

### Tech debt の置き換え

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L20・L21: `orgReseatHelp` の短い行と `reseated` の文 | 表示の乱れが 1 行と、`reseated` が書かれない狭い場合を書かない help の文 1 つ。reseat の動作は変わらない | Go のソースを変えるので、直すとパイプラインが `/self-review` からになる。LOW | `orgReseatHelp` の次の変更(L19 を直すなら一緒に) | 本 report の L20・L21 |
| L19 | df8f674e の節の行のまま。help が start 時の検出を言うようになったので、Impact は「stdout の出力だけが言わない」に狭まる | 前の通り | 前の通り | 本 report の L19 |
| L6・L13 | 前の report の通り(箇所は増えていない) | 前の通り | 前の通り | `docs/tech-debt/README.md` の「Eleven small findings of docs/spec-org-reseat」の行 |

`docs/tech-debt/README.md` の「Eleven small findings of docs/spec-org-reseat」の行へ、この cycle の `/sync-docs` が足す直しは、df8f674e の節の項目のうち次のとおり変わる:

- L19 を (l) として足す案は前のまま。L20 と L21 を足すなら (m) として、見出しの件数は 13 になる(L19 だけなら 12)。
- ファイルの大きさの行は、`reseat.go` が 1,361 行、`org_reseat_test.go` が 858 行(852 から +6)。

### Recommendation(この差分)

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW は 13 件(持ち越し 11 件の L2・L3(b)・L4・L5・L6・L10・L13・L14・L15・L16・L19 と、新しい L20・L21)。b142944d が足した help の文は、コードと test に照らして偽の文が無い。V19 の偽(打ち直しが出ない場合と、start 時の検出)は直った
- 直しの中身は、help の 2 か所と、その pin 2 つだけで、コードは変えない。新しい 2 件は、足した文の折り返し(L20)と、足した文の直後にある既存の文の条件落ち(L21)で、どちらも読み手に誤解を招くだけで、reseat の動作を変えない
- Follow-ups: L20 と L21 は `orgReseatHelp` の文字列だけで、test は変わらないが、Go のソースなので、直すならパイプラインは `/self-review` からやり直す。L19 を直すなら一緒に直すのが安い。cycle 3 なので、上限に届いているはず(未確認)で、直すなら上限を上げる。直さないなら register へ(上の表)。L21 は、`ralph org reseat --help` を読んで `reseated` を探す人にだけ当たる狭い場合なので、直さない判断も立つ

## cycle 3 の差分(cross-review の修正 4 コミット、7ea546da..b568ca8c)

- Scope: `git diff 7ea546da..b568ca8c` の 6 ファイル(`internal/org/reseat.go`、`internal/org/reseat_disband_test.go`(新規 366 行)、`reseat_test.go`、`reseat_start_test.go`、`internal/cli/org_reseat.go`、`org_reseat_test.go`)を全部読んだ。4 コミットは ffd32109(`--config` を DECISION の send と read にも)、84346015(start の前の disband の検出)、c4f55aef(stop の前の検出)、b568ca8c(`orgReseatLeftLine` の行)。周りは `finish`・`reseatRetryCommand`・`reseatVerbCommand`/`VerbCommand`・`startAll`・`stopAll` の全体、`Disband` と `appendDisbanded`(`verbs.go`)、`lastDisbandedIndexes`・`currentOrgLives`・`RunningOrgs`(`reserve.go`)、`EventReseated` の doc(`seat.go`)、`newOrgSendCmd`・`newOrgReadCmd`・`newOrgRuntimeAt`・`orgReadCommandHint`(`internal/cli/org.go`)、`printOrgReseatResult`、triage report の見出しと ACTION_REQUIRED の表を読んだ
- 実行したもの: `awk` による行の長さの測定、`gofmt -l` を 6 ファイルへ 1 回(出力なし)、`git grep` による呼び元・箇所数の数え直し。**テストと静的解析は走らせていない。** error 文・出力・test の判定は、コードと test の期待を読んで導いたもの
- ID: 新しい指摘は `L22` から `L25`。行番号は動くので、関数名を主にする(ただし L22 の comment 行だけは行番号を添える)

### 経路ごとの error 文

`reseatDisbandedErr` と、stop 側・start 側の読み直し失敗の文を、reseat が終わる経路ごとに当てた。

| 経路 | 出る文(要点) | 判定 | 根拠 |
| --- | --- | --- | --- |
| 最初の stop の前に disband | `ended before the stop of seat "reviewer", and started no seat: the org was disbanded during the reseat (disbanded at <TS>); the feature org starts again with ralph org start --plan` | 本当 | `checkBeforeStop` が `reseatDisbandedIn` を座席の確かめより先に呼ぶ。何も止めず立てず、`attempted` が立たないので `reseated` も書かない。`...BeforeItsFirstStop` が固定する |
| stop の途中(2 つ目の stop の前) | 同じ文 | 本当 | reseat が先に止めた座席は stdout の `stopped:` と `left stopped:` の行が言い、error は触れない。`Changed` は立たない(同じ 1 回の読みで disband を先に見る)。`...BeforeItsSecondStop` が固定し、`reseated` が `disbanded` のあとに書かれることまで見る |
| 全部止めたあと、最初の start の前 | `ended before the start of seat "implementer", and started no seat after it: ...` | 本当 | `started` が空なので「started before the disband」の句は付かない。`...OneSeat...BeforeItsStart` が固定 |
| leader を立てたあと、次の start の前 | 上に `seat "leader" started before the disband, in the life it ended, so ralph org status shows whether it still runs` | 本当 | 「止まっている」とは言わず、動いているかは status に任せる。2 座席以上は複数形の文で、`TestReseatDisbandedErr` が 0・1・2 座席と時刻なしを固定する。窓(再読と `Spawn` の書き込みの間)で disband が先に入ると「before the disband」は正確でなくなるが、既知の窓で、文は断定を避けている |
| 読み直しの失敗(manifest が読めない) | `... read the manifest again: <err>` | 本当 | `Disbanded` は立てず、`Changed` も立てず、打ち直しを出す。org が disband されたとは言わない。stop 側・start 側の test がある |
| disband のあとに org が start し直された | 上と同じ「disbanded」の文 | 偽ではない | 最新の `disbanded` の添字が変わるので検出される。末尾の「starts again with」は立て直し済みの org には余計だが、起きにくく、害は無い |

「打ち直しも DECISION も、disband された org が断るものは出さない」: `finish` は `Disbanded` で `reseated` の記録のあと、`Retry` を作る前に返る。`Retry`・`Decision`・`DecisionRead` は `""` のままで、CLI は `res.Retry != ""`・`res.Decision != ""` で行を出し分けるので、行は出ない。disband を「座席の記録が変わった」と読む道は無い(`checkBeforeStop` は 1 回の読みで disband を先に見て、`found == false` の分岐は手で書き換えた manifest にだけ届く。`manifest.go` には書き換えの関数が無い)。これで届かないのは次の経路で、どれも既知の窓の仲間か、文が偽にならないもの。

- 検出のあと `Spawn`・`Stop` の書き込みまでの窓(tracked)
- stop が失敗した経路(`stop_failed`)で、失敗の理由が disband だったとき: 打ち直しが出て、disband のあとに断られる
- disband が途中(座席の `stopped` は書かれたが `disbanded` がまだ)のとき: reseat は「ほかのコマンドが止めた」(`Changed`)と読み、打ち直しを出す。その文は偽ではなく、打ち直しは disband が済んだあとに断られる

### `--config`(ffd32109)

- 順と quote: `ReseatDecisionCommand` と `reseatLeaderReadCommand` は、打ち直しと断りの文が使う `reseatVerbCommand` を通る。語の並びは verb と引数、`--config`(`ShellWord` で 1 語)、`--state-dir`、send だけ最後に `--text`(`shellQuote`)で、打ち直しと同じ。`TestReseatDecisionCommand_RepeatsTheConfigAndTheStateDir` が、どちらも無い・state dir だけ・config だけ・両方(空白入りの path の quote)の 4 つを、期待を別に組み立てて固定する。CLI の test は、出力の組み立てと、印字された read・send を無効な `ralph.toml` のあるディレクトリから実際に走らせるところまで見る(`--config` なしの read がそこで `load config` で落ちることも)
- 「`send` と `read` は ralph.toml を読んで検証する」(`HintConfigPath` の doc の新しい文): `newOrgSendCmd` は `newOrgRuntimeAt`、`newOrgReadCmd` は `newOrgRuntime` を通り、どちらも `resolveOrgConfig` で読む(`org: load config`)。本当。cycle 2 の L15 (a) の訂正がそのまま入った
- 旧 `orgReadCommandHint` との出力の差: org id を `ShellWord` に通すようになった(前は素)。org_id は `ValidateIdentifier` で縛られているので、出力は同じ。`--state-dir` は `hintStateDir != ""` のとき同じ位置に付く
- `printOrgReseatResult` の引数 `hintStateDir` を消した。呼び元は `runOrgReseat` の 1 か所(grep)。`ReseatDecisionCommand` の非 test の呼び元は `finish` だけで、test の呼びはすべて新しい 4 引数に直っている(grep)

### help の文

| 文(要点) | 判定 | 根拠 |
| --- | --- | --- |
| disband は次の stop か start の前に reseat を終わらせ、残りを立てるコマンドは出ず、feature org は `ralph org start --plan` で立て直す | 本当(既知の窓を除く) | `Retry` が `""` になること、`reseatNoBindingErr` が disband 後の reseat を断ることは上のとおり。「too」は直前の文(他のコマンドが止めた座席で終わる)に合う。pin は `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` の既存の塊に足された |
| reseat・stop・spawn・start のコマンドに、output の `ralph org read` と `ralph org send` も `--config` を繰り返す | 本当。例外は L15 (b) のまま | read・send は上のとおり。`reseat`・`stop`・`spawn`・`start` の列にある「断りの中の stop」は、共通の callee(`stopSeatCommand`)が書くものが `--config` を持たない。help は言わず、`HintConfigPath` の doc は言う |
| 「read no ralph.toml」の文が消えた | 本当 | test は文の不在も pin する(`strings.Contains(help, "read no ralph.toml")` が false) |

### `startAll`・`stopAll` の読みやすさ

`stopAll` は 23 行、`checkBeforeStop` は 20 行、`startAll` は 45 行、`endIfDisbanded` は 21 行で、どれも短い。`stopAll` の読み直しと検査が `checkBeforeStop` に移り、`startAll` は先頭の 1 行(`if r.endIfDisbanded(i) { return }`)が増えただけで、本筋は変わらない。ただし 2 つの helper は真偽の向きが逆で(L25)、`reseat.go` は 1,537 行(+176)になった。

### test の pin

- 固定しているもの: 4 経路の error 文の全文(`disbanded.TS` を実際の `Disband` が書いた event から取る)、`Seats[from:]` の `Spawn == nil`、`reseated` が `disbanded` のあとにあること、最初の stop の前では `reseated` を書かないこと、読み直しの失敗で `Disbanded`・`Changed` が立たないこと、過去の disband があっても今の命の reseat が通ること(誤検出の逆)、`orgReseatLeftLine` の表(disbanded の真偽 × outcome)。disband は実際の `Disband` で作り、`disbanded` だけを足すのではない
- 固定していないもの: `Decision`・`DecisionRead` が空であること(L23)

## 新しい指摘(cycle 3、4 コミット)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L22 | LOW | Readability (layout) | 足した文を既存の行に差し込んだだけで、周りを詰め直していない箇所が 2 つある。(a) `orgReseatHelp` の「A disband of the org ...」の文の末尾の行 `with \`ralph org start --plan\`. A seat it` が 45 文字で、前後は 73 と 78 文字。L20 と同じ段落で、同じ型(差し込んだ文の尻が短い行になる)。(b) `ReseatResult.Disbanded` の doc の最後の行が 108 文字(tab を含む)で、`reseat.go` の中で 100 文字を超える comment 行はこの 1 行だけ。ほかの doc は 70 から 80 文字で折り返している。表示と読みやすさの問題だけで、意味は変わらない | (a) `internal/cli/org_reseat.go` の `orgReseatHelp`(79 行目、`awk` で `\t"` と `\n" +` を外して測った)。(b) `internal/org/reseat.go` の 957 行目(`and the leader a DECISION would go to is gone. The feature org starts again with ralph org start --plan.`)と、`awk 'length($0)>100'` で comment 行が 1 つだけ出たこと | (a) は L20 の直しと同じ段落なので一緒に詰め直す。pin は `strings.Fields` で空白を畳むので test は変わらない。(b) は「The feature org starts again ...」を次の行へ送る。Go のソースなので、直すならパイプラインは `/self-review` からやり直す |
| L23 | LOW | Tests (vacuous assertion) | `assertDisbandedResult` は「retry、DECISION、leader の pane の read が無い」と言い、4 つの disband の test がこれを通すが、`Decision` と `DecisionRead` の空の確かめは、どの test でも直しを外しても通る。`finish` が DECISION を作るのは「leader が対象でなく、立った対象が 1 つ以上」のときだけ。test 1(implementer だけ)は立った座席が無く、test 2 は `plan.Start[0]` が leader なので `finish` の loop が最初に返り、test 3・4 は start に届かない。`finish` の `if r.res.Disbanded { return r.res }` を消しても、`Retry` は test 1・2・3・4 のどれでも空でなくなって落ちるが、`Decision`・`DecisionRead` が空でなくなる test は無い。lead の確かめにある「disbanded な org が断る DECISION を出さない」は、コードは正しいが、固定されていない。効く場面は `reseat --seat implementer --seat reviewer` で、reviewer の start の前に disband が入るとき(implementer が立ち、leader は対象でない)。コードを読んで導いたもので、直しを外す実験はしていない | `internal/org/reseat_disband_test.go` の `assertDisbandedResult`、`TestReseat_OneSeatOfAnOrgDisbandedBeforeItsStart`・`TestReseat_AnOrgDisbandedAfterTheLeaderStarted`・`...BeforeItsSecondStop`・`...BeforeItsFirstStop`、`internal/org/reseat.go` の `finish`(`seat.Leader` で返す loop と、`Disbanded` の早い return) | test を 1 本足す: `disbandBeforeStart(t, st, "reviewer")` と `st.reseatParams(implementerSeatID, "reviewer")` で、implementer が `started`、reviewer が `stopped`、`Decision`・`DecisionRead`・`Retry` がすべて空、を `assertDisbandedResult` で見る(`from` は 1)。または `assertDisbandedResult` の doc から「DECISION」を外し、確かめを持つ test の名前を書く。Go のテストなので、足すならパイプラインは `/self-review` からになる |
| L24 | LOW | Documentation in code | `Retry`・`Decision` が空になる場合を書く 4 つの doc が、`Disbanded` の例外を言わない。例外を言うのは `ReseatResult.Disbanded` の doc と `finish` の doc だけ。(a) `ReseatResult.Retry`: 「"" when there is none」。(b) `ReseatResult.Decision`: 「"" otherwise」(leader が対象でなく、立った対象がある、のあと)。(c) `Reseat` の doc の結果の段落: 「the reseat that starts the ones not started (Retry), and, when the leader was not one of the targets, the DECISION ... (Decision)」。(d) `printOrgReseatResult` の doc: 「the reseat that starts the seats left (res.Retry), when there are any」と、DECISION を「when the leader was not a target and a target started, also after a start that failed」と書く。(a)(b)(d)は今回の diff が同じ doc に `--config` の文や `res.DecisionRead` を足しながら例外を足さなかった所。偽になる場面は、leader が対象でない reseat で、立てた座席があり、次の start の前に disband が入るとき: (d) の doc は DECISION が出ると読め、コードは出さない。読み手は `Disbanded` の doc まで読まないと分からない。コードを読んで導いたもの | `internal/org/reseat.go` の `ReseatResult`(`Retry`・`Decision`・`Disbanded`)、`Reseat` の doc の結果の段落、`finish` の doc、`internal/cli/org_reseat.go` の `printOrgReseatResult` の doc | (a)(b)(d)の文に「; "" also when the org was disbanded during the reseat (Disbanded)」の句を足し、(c) に同じ 1 句を足す。または (c) の段落に 1 文だけ足して、(a)(b)(d)からそこを指す。Go のソースの comment なので、直すならパイプラインは `/self-review` からになる |
| L25 | LOW | Naming | 新しい helper の名前と真偽の向きが、すぐ隣の語とずれる。(a) `reseatDisbandedIn` と `reseatDisbandedNow` の戻り値 `changed` は「disband が計画のあとにあった」を言うが、`checkBeforeStop` の中ではそのすぐ下で `r.res.Changed = seat.SeatID`(別のコマンドが座席を変えた)を立てる。同じ関数で `changed`(小文字)と `res.Changed` が別の意味を持ち、`if disbanded, changed := reseatDisbandedIn(...); changed` の `disbanded` は event、`ReseatResult.Disbanded` と `orgReseatLeftLine` の引数 `disbanded` は bool。(b) `checkBeforeStop` は true が「続ける」、`endIfDisbanded` は true が「終えた」で、`startAll` と `stopAll` で呼ぶ向きが逆になる(`if !r.checkBeforeStop(seat) { return false }` と `if r.endIfDisbanded(i) { return }`)。しかも `endIfDisbanded` は manifest が読めないときも reseat を終えるので、名前が言う「disband のとき」より広い。どちらも読む人が doc を開けば分かる範囲で、動作は正しい | `internal/org/reseat.go` の `checkBeforeStop`・`endIfDisbanded`・`reseatDisbandedIn`・`reseatDisbandedNow` | (a) 戻り値を `since`(または `newer`)にして、event の変数を `latest` にする。(b) `endIfDisbanded` を `checkBeforeStart` にして `checkBeforeStop` と向きと名前をそろえる(true で続ける)。`startAll` の呼びは `if !r.checkBeforeStart(i) { return }` になる。Go のソースなので、直すならパイプラインは `/self-review` からになる |

指摘にしなかったもの:

- `orgReseatLeftLine` の `not stopped: <seat> (the org was disbanded meanwhile)` は、読み手に「止まっていない(動いている)」と読まれる余地がある。b568ca8c がねらった「left running」の偽は直っている。隣の `not started: <seat>` が同じ作り(reseat がしなかったことを言う)で、括弧が理由を言う。`left stopped by the disband: ...` のような言い方も立つが、好みの範囲で、pin は `TestOrgReseatLeftLine` の 1 行に集まっている。
- `reseated` を disband のあとに書く点: org レベルの event で、`currentOrgLives` の switch は `reseated` を読まず、`RunningOrgs`・`Roster`・`orgsToDisband` に数えられない(`seat.go` の doc と同じ)。次の命の `reseatLifeSeats` はその id を `recorded` に入れるが、次の命に同じ id の note を持つ `stopped` は無いので、何も起きない。
- `ReseatPlan.LastDisbandedIndex` の「無い」が `-1` で、Go の零値(0)と違う。`ReseatPlan` を組むのは `PlanReseat` の 1 か所と、`Reseat` を走らせない test の literal だけ(grep)。`Reseat` は自分の `PlanReseat` の plan しか使わないので、零値の罠は届かない。
- `reseatChangedErr` の `found == false` の分岐は、ほぼ届かない道を doc の注釈つきで残している。manifest が追記だけであることを書く doc は本当(`manifest.go` に書き換えの関数が無い)。
- `reseatLeaderReadCommand` が、`internal/cli` の `orgReadCommandHint`(「`--config` は付けない」と doc に書く)と別に、`ralph org read` を組む 2 つ目の関数になった。org 層から CLI の関数は呼べず、`orgReadCommandHint` の doc と `ApprovalDecisionCommand` の `--config` なしは、lead の言う既知の tech-debt。
- 終了メッセージの「started no seat after it」の「it」が、座席そのものを含むか曖昧に読める。前に「ended before the start of seat X」があるので、X も立っていないと読める。直す価値は薄い。

### 前の指摘のうち 4 コミットが触れた所

| ID | 判定 | 根拠 |
| --- | --- | --- |
| L15 | (a) は閉じた。(b) が残る | (a) 「`send` と `read` は ralph.toml を読まない」の文は、help、`HintConfigPath` の doc、help の test から消え、2 つとも `--config` を繰り返す。残る (b) は、help の「reseat・stop・spawn・start のコマンドが断りの中でも繰り返す」に、共通の callee の `ralph org stop` が当たらない点。`HintConfigPath` の doc は例外を書き、help は書かない |
| L6 | 残っている。箇所が 1 つ増えた | `reseat_disband_test.go` の冒頭の comment が `docs/plans/active/2026-11-12-org-reseat.md, P6` を引く |
| L13 | 残っている。箇所が 4 つ増えた | `cross-review cycle 3, finding 1/2` を引く comment が、`reseat_disband_test.go` の冒頭、`reseat_start_test.go`、`reseat_test.go`、`org_reseat_test.go` に 1 つずつ(diff で数えた。regexp で数えると、折り返しの箇所が漏れる)。triage report は cycle ごとに書き直され(今は cycle 3 の 2 件だけ)、番号の引用は次の `/cross-review` で別の指摘を指す。コミット(ffd32109、84346015)は消えない |
| L5 | 残っている | `PlanReseat` は 147 行のまま(`ReseatPlan{...}` の 1 行が長くなっただけ) |
| L2・L3(b)・L4・L10・L14・L16・L19・L20・L21 | 残っている。触っていない | 4 コミットの差分に、それぞれの箇所が無い。L19 の `start_failed` の座席の枝は、`orgReseatLeftLine` の同じ関数だが、b568ca8c が足したのは `NotStopped` の枝だけ |
| ファイルの大きさ | `reseat.go` 1,537 行(+176)、`org_reseat.go` 521、`org_reseat_test.go` 923(+65)、`reseat_start_test.go` 267(+30)、`reseat_disband_test.go` 366(新規) | 800 行の目安を超えるのは `reseat.go`・`reseat_test.go`(1,868)・`org_reseat_test.go` で、前の節からの数字の続き。register の行の数字は `/sync-docs` が直す |

文書の hand-off(`/sync-docs`): spec の FR-5a、plan の P6、org skill の reseat の失敗の段落(`.claude/skills/org/SKILL.md` と `.agents`・`templates/base` の 3 組)は、disband による終わり(打ち直しが出ないこと、`not stopped:` の行)を言わない。どの文も偽にはなっていない。help は新しい文を持つ。register の「Thirteen small findings of docs/spec-org-reseat」の行へ、この cycle の `/sync-docs` が足す直しは次のとおり。

- (j) の L15 から (a) の半分(`send`・`read` の例外)を外す。(b)(stop の例外)だけが残る。
- L22 から L25 を (n) として足し、見出しの件数を 14 にする。
- (e) の L6 の箇所を 1 つ、(h) の L13 の箇所を 4 つ増やす。数え方は diff の追加行(上の表)。
- ファイルの大きさの行を上の数字にする。

### ブランチの今の指摘(17 件)

CRITICAL・HIGH・MEDIUM は無い。LOW は次の 17 件。

| ID | Area | 一言 |
| --- | --- | --- |
| L2 | Maintainability | dry run の herdr probe が gate の前半を写している |
| L3(b) | Error text | leader を対象にしない reseat の `[reviewer]` の断りが leader の task を言う |
| L4 | Naming | 関数 `reseatSeat` と型 `ReseatSeat` が頭文字の大小だけで違う |
| L5 | Readability | `PlanReseat` が 147 行 |
| L6 | Maintainability | archive へ動く plan を指す注釈が 7 か所 |
| L10 | Exception handling | 出力を読む手が居ない実行の部分失敗の account が stdout に偏る |
| L13 | Maintainability | triage を番号で引く test の comment が 9 か所 |
| L14 | Consistency (merge) | `reseat` が #310 の codex hook の通知を出さない |
| L15 | Documentation in code | `--config` の help の文に、共通の callee が書く stop の例外が無い((a) は閉じた) |
| L16 | Error text | 断りが custom role の座席も「reseat が立て直せる座席」に並べる |
| L19 | Consistency (printed output) | 他のコマンドが立てた座席を打ち直しから外した理由が stdout に出ない |
| L20 | Readability (help layout) | help の段落の途中に 50 文字の短い行 |
| L21 | Documentation in code (help) | `reseated` を「途中で終わっても書く」とする文が、何も止めず立てずに終わる場合を含む |
| L22 | Readability (layout) | disband の文の尻が 45 文字の短い行と、108 文字の comment 行 |
| L23 | Tests (vacuous assertion) | `Decision`・`DecisionRead` が空であることを、どの test も直しを外して通る |
| L24 | Documentation in code | `Retry`・`Decision` が空になる場合を書く 4 つの doc が、disband の例外を言わない |
| L25 | Naming | `changed`・`disbanded` の語が `res.Changed`・`res.Disbanded` とずれ、2 つの check の向きが逆 |

### Tech debt の置き換え

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L22 から L25: disband の検出の周りの layout、test の穴、doc の例外、名前 | 読み手が disband の例外を doc から拾えず、`finish` の早い return の一部が test に固定されていない。reseat の動作は変わらない | Go のソースと test を変えるので、直すとパイプラインが `/self-review` からになる。パイプラインは cycle 3、cap 3(`cycle-count.json` を読んだ)で、直すなら上限を上げる必要がある。LOW | `finish`・`checkBeforeStop`・`endIfDisbanded` の次の変更(L23 の test は一緒に足すのが安い) | 本 report の L22 から L25 |
| L15 | (b) だけ残る。Impact は前の通り | 前の通り | 前の通り | 本 report の L15 の行 |
| L6・L13 | 箇所が増えた(L6 +1、L13 +4) | 前の通り | 次の `/cross-review` の前に番号を落とすと安い | `docs/tech-debt/README.md` の「Thirteen small findings of docs/spec-org-reseat」の行 |

### Recommendation(この差分)

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW は 17 件(持ち越し 13 件の L2・L3(b)・L4・L5・L6・L10・L13・L14・L15・L16・L19・L20・L21 と、新しい L22 から L25)。cross-review の ACTION_REQUIRED 2 件の直しは、コードの経路と error 文を読んだ限り、偽の文も、disbanded な org が断る打ち直し・DECISION を出す道も、窓を除いて見つからなかった。L15 の (a) は閉じた
- 直しの中身は、`--config` を DECISION の send と read にも付けること(`reseatVerbCommand` を通るので打ち直しと同じ順・同じ quote)、`Reseat` が各 stop と各 start の前に manifest を読み直して、計画のあとの `disbanded` で終わること(`Disbanded` を立て、`Changed` と混ぜない)、`finish` が `Disbanded` で打ち直し・DECISION を作らないこと、`not stopped:` の行。新しい指摘は、layout(L22)、test の穴(L23)、doc の例外の抜け(L24)、名前(L25)で、どれも reseat の動作を変えない
- Follow-ups: L22 から L25 はどれも Go のソースか test なので、直すならパイプラインは `/self-review` からやり直し、cap は 3 に届いているので上げる必要がある。直さないなら register へ(上の表)。直すなら L23 の test が最も効く(`finish` の早い return を外す変更を、いまは test が見逃す)。L13 の番号は、次の `/cross-review` が triage を書き直す前に落とすと安い

## cycle 3 の差分(f2ed6bdf、L23 の test)

- Scope: `git show f2ed6bdf` の 1 ファイル(`internal/org/reseat_disband_test.go`、+22 行、新しい test 1 本)を全部読んだ。周りは `finish`・`startAll`・`endIfDisbanded`・`reseatDisbandedErr`(`internal/org/reseat.go`)、`assertDisbandedResult`・`disbandBeforeStart`・`intrudeBeforeStart`、隣の disband の test 4 本、`TestReseatDisbandedErr`
- 実行したもの: `go test -count=1 -run 'Disband' ./internal/org/` を 1 回(`ok`、2.4 s)。さらに `go test -overlay`(写しは scratchpad にあり、作業木は変えていない)で、`finish` を変えた mutant に同じ `-run 'Disband'` を 2 回。(1) `if r.res.Disbanded { return r.res }` を `Retry` の代入の後ろへ動かした mutant では、`Retry` も漏れるので disband の test 5 本とも落ちる(新しい test を含む)。(2) その return を消し、`Retry` の代入だけを `if !r.res.Disbanded` で囲んだ mutant(`Decision`・`DecisionRead` だけが漏れる)では、落ちるのは新しい test 1 本だけで、`reseat_disband_test.go:174` が `decision "ralph org send --org-id auth-core --to leader ..." read "ralph org read --org-id auth-core --seat leader"` を出した。lead の報告(return を消すと 5 本、`Decision`・`DecisionRead` だけが漏れる形では新しい test だけ)と合う。return を消しただけの mutant は走らせていない
- ID: 新しい指摘は `L26`

### L23 の判定

閉じた。L23 の穴は「`Decision`・`DecisionRead` が空であることを、どの test も直しを外して通る」だった。上の mutant (2) がそれで、新しい test だけが落ちる。場面も L23 が書いた形のとおり(`disbandBeforeStart(t, st, "reviewer")`、`reseatParams(implementerSeatID, "reviewer")`、`assertDisbandedResult(..., 1)`)。`assertDisbandedResult` の doc が言う「retry、DECISION、leader の pane の read が無い」は、これで 4 つ全部をどれかの test が固定する。

### test の判定

- 名前: `Disband` と `Decision` を含むので、`-run 'Disband'` に掛かる。隣の test は場面の名前(`...AfterTheLeaderStarted`)で、この名前は起きないことを言う。`After a Disband Once ...` は語が続くが、読めば分かる範囲で、指摘にしない
- comment の文は偽ではない。「the one shape」: `finish` が DECISION を作るのは「leader が対象でなく、`started` が 1 つ以上」のときで、`Disbanded` で終わる経路のうち start の前に終わるものだけが `started` を持てる。leader が対象なら `finish` の loop が最初に返る。「a leader the disband ended」: `BeforeItsSecondStop` の doc が言うとおり、disband は leader も止める。「the other disband tests start no seat but the leader」: test 1 は implementer を立てず、test 2 は leader だけ、test 3・4 は start に届かない
- 期待値: outcome `implementer:started,reviewer:stopped`、error 文は `reseatDisbandedErr` の「1 座席」の形で、`TestReseatDisbandedErr` の「one started」と同じ語。`Seats[0]` が立ったことは outcome の文字列が固定し、`Seats[1:]` の `Spawn == nil` は `assertDisbandedResult` が見る
- 後始末: `intrudeBeforeStart` が `t.Cleanup` でフックを戻す。`t.Parallel` は無い。`"reviewer"` は literal だが、同じ file の他の test(9 行)と同じ
- 弱い所: `reseated` の中身(`seats=implementer:started,reviewer:stopped`)は見ない。test 2 が同じ `reseatedDetails` の経路で固定しているので、この test の目的(DECISION)から外れる分は足りなくてよい

### 新しい指摘(cycle 3、f2ed6bdf)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L26 | LOW | Maintainability (comment) | 新しい test の doc の最後に「(self-review L23: the other disband tests start no seat but the leader)」とあり、この branch の self-review report の finding 番号を引く。report の名前は無く、番号だけでは読み手が探せない。L6(archive へ動く plan の path)・L13(triage の番号)と同じ型の引用で、同じ file の冒頭の comment が既にその 2 つを持つ。comment は番号を外しても文が残る。L13 の triage と違い、引く先の report は cycle をまたいで積まれるので別の指摘を指す恐れは小さく、残るのは「何の番号か分からない」だけ | `internal/org/reseat_disband_test.go` の新しい test の doc の最後の 2 行(161 行目) | 「self-review L23: 」を落とし、`(the other disband tests start no seat but the leader)` にする。Go のソースなので、直すならパイプラインは `/self-review` からになる。L22・L24・L25 と一緒に直すのが安い |

### 前の指摘のうち f2ed6bdf が触れた所

| ID | 判定 | 根拠 |
| --- | --- | --- |
| L23 | 閉じた | 上の「L23 の判定」 |
| L6・L13 | 残っている。L13 と同じ型の引用が 1 つ増えた(L26) | 新しい test の comment |
| そのほか | 触っていない | 差分は test 1 本だけ |

文書の hand-off(`/sync-docs`): 前の節の「L22 から L25 を (n) として足す」は、L23 が閉じたので L22・L24・L25・L26 の 4 件にする(見出しの件数 14 は変わらない)。ファイルの大きさの行は `reseat_disband_test.go` が 366 から 388 行になる。

### ブランチの今の指摘(17 件)

CRITICAL・HIGH・MEDIUM は無い。LOW は 17 件で、前の節の表(L23 を除く 16 件)に L26 が加わる。

| ID | Area | 一言 |
| --- | --- | --- |
| L2 | Maintainability | dry run の herdr probe が gate の前半を写している |
| L3(b) | Error text | leader を対象にしない reseat の `[reviewer]` の断りが leader の task を言う |
| L4 | Naming | 関数 `reseatSeat` と型 `ReseatSeat` が頭文字の大小だけで違う |
| L5 | Readability | `PlanReseat` が 147 行 |
| L6 | Maintainability | archive へ動く plan を指す注釈が 7 か所 |
| L10 | Exception handling | 出力を読む手が居ない実行の部分失敗の account が stdout に偏る |
| L13 | Maintainability | triage を番号で引く test の comment が 9 か所 |
| L14 | Consistency (merge) | `reseat` が #310 の codex hook の通知を出さない |
| L15 | Documentation in code | `--config` の help の文に、共通の callee が書く stop の例外が無い((a) は閉じた) |
| L16 | Error text | 断りが custom role の座席も「reseat が立て直せる座席」に並べる |
| L19 | Consistency (printed output) | 他のコマンドが立てた座席を打ち直しから外した理由が stdout に出ない |
| L20 | Readability (help layout) | help の段落の途中に 50 文字の短い行 |
| L21 | Documentation in code (help) | `reseated` を「途中で終わっても書く」とする文が、何も止めず立てずに終わる場合を含む |
| L22 | Readability (layout) | disband の文の尻が 45 文字の短い行と、108 文字の comment 行 |
| L24 | Documentation in code | `Retry`・`Decision` が空になる場合を書く 4 つの doc が、disband の例外を言わない |
| L25 | Naming | `changed`・`disbanded` の語が `res.Changed`・`res.Disbanded` とずれ、2 つの check の向きが逆 |
| L26 | Maintainability (comment) | 新しい test の comment が self-review の finding 番号を引く |

### Recommendation(この差分)

- Merge: yes。CRITICAL・HIGH・MEDIUM は無い。LOW は 17 件(持ち越し 16 件の L2・L3(b)・L4・L5・L6・L10・L13・L14・L15・L16・L19・L20・L21・L22・L24・L25 と、新しい L26)。L23 は閉じた。新しい test は、`finish` の早い return から `Decision`・`DecisionRead` だけが漏れる mutant を、他の test が通すところで落とす
- Follow-ups: L26 は comment の 1 句で、L22・L24・L25 と同じく Go のソースなので、直すならパイプラインは `/self-review` からやり直し、cap は 3 に届いている(`cycle-count.json` を読んだ。cycle 3、cap 3)ので上げる必要がある。直さないなら register へ(L26 は (n) に足す)
