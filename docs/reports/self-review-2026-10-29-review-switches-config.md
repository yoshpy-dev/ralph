# Self-review report: review-switches-config

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-review-switches-config.md`(issue #265、D1〜D6、AC1〜AC6、Progress の S1〜S4)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff da9d1b3a...HEAD`(HEAD 79fb9d65、9 コミット、31 ファイル、+1565/-200)。中心は S1 eff0340d(`[standard]` と `[org.review]`、テンプレート、lock-step の test)、S2 8e51e1c9(`ralph config get`、終了コード 2 と 3)、S3 33d859bb(`TabCreate` の `extraEnv` と `RALPH_PRIMARY_CLI`)、S4 98e19052(ルートの `ralph.toml`、`KNOWN_DIFFS`、文書)。計画ファイルと Progress だけのコミットは読んだが所見の対象にしていない。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- 既定の判断(依頼どおり欠陥として扱わない): skill がまだ設定を読まないこと(#266、D4)、headless の director の pane に `RALPH_PRIMARY_CLI` を渡さないこと(D5)、終了コード 3 を `ralph upgrade` と分け合うこと(D6)、存在しない `--config` のパスが既定になること。
- ID: M1(MEDIUM)、L1〜L6(LOW)。cycle 2 以降はこの ID で状態を追う。

## Evidence reviewed

- `internal/cli/config.go` の全体(138 行)。`configValue` の reflect の歩きは、先頭の `reflect.ValueOf(cfg)` が struct、`configField` が toml タグで一致を探し、`tag != ""` と `tag != "-"` を見るので、タグのない(unexported を含む)field は飛ばされる。空のキー、末尾の `.`、表の名前、map(`org.roles`)、リスト(`org.driver_pool`)、値の下のキー(`standard.plan_advisory.x`)は、すべて `ErrConfigKey` に落ちる。`Config` には pointer、float、uint、`time.Duration` の field がなく(`grep` で確認)、足されれば `default` の枝で exit 2 になり、読めない値を黙って出すことはない。
- 終了コードの対応。`RunE` は `config.Load` を先に呼び、失敗を `%w: %w` で `ErrConfigLoad` と元のエラーの両方に包む。`ExitCode`(`exitcode.go`)は `ErrConfigKey` を先に、`ErrConfigLoad` を `ErrUpgradeDriftRemaining` と同じ枝で見る。`Load` のエラーの連鎖に `ErrConfigKey` は入らないので、取り違えはない。`Load` をキーの照合より前に呼ぶので、壊れた `ralph.toml` では知らないキーでも 3 になる(S2 の報告と一致)。
- build した binary(scratchpad の `ralph-probe`、`go build` のみ)で手で打った結果: `org.max_seats.x` は exit 2 と `org.max_seats is a int value`、`org.approval` は exit 2 と `it is a table`、`ralph.toml` のない空の cwd の `standard.cross_review` は exit 0 と `false`(stderr は空)、`ralph config gte x` は exit 1 と `unknown command "gte" for "ralph config"`、`[org] agmsg_home = """a\nb"""` の `org.agmsg_home` は 2 行を出す。
- `config_test.go`(243 行)の全体。「読めない」ケースは `writeRalphTomlBehindLockedDir`(`org_test.go:1111`)を使い、`os.Geteuid() == 0` の skip と、`t.Cleanup` での chmod の戻しがある。cleanup は TempDir の削除より後に登録されるので先に走る。「ディレクトリ」のケースは root でも 3 になる。
- `internal/config/config.go` の差分、`config_test.go` の 4 テスト、`defaults_sync_test.go` の差分全体。`checkEachKeyIsComparedOnBothSides` の切り出しで、`org` の行は元の 19 行がそのまま残り、`probe.Name()` で失敗の名前が出る。`shell_and_skills` の上限の照合は `mustShell` の既存の道具を使う。
- `internal/org/driver/herdr.go`、`paneenv.go` の差分と `TabCreate` の全体、`spawn.go:985-1000`。`env` は `PaneEnv` のあとに `extraEnv` の順で、`len(env) > 0` で `--env` の probe に入る。`permissionArgsForDriver`(`permissions.go:126`)は `claude` と `codex` の完全一致以外を拒むので、`PrimaryCLIEntry(p.Driver)` に入るのはその 2 値だけ。`TabCreate` の非 test の呼び出しは `spawn.go:998` と `director_start.go:1125` の 2 つで、fake は `fakeHerdr`、`directorHerdr`、`fakeWatchHerdr` の 3 つ(`grep` で全部)。
- `scripts/xreview-helpers.sh:57-77`(`pick_reviewer`)と `.claude/skills/cross-review/SKILL.md:34-45`(step 2)と `.claude/skills/plan/SKILL.md:73-82`(step 11)を、`PrimaryCLIEnv` の説明と突き合わせた。
- ルートの `ralph.toml` とテンプレートの `diff`: 違うのは 2 行目の注、30-31 行目、255-256 行目の 3 か所だけ。`/org` skill の 4 つの写しと recipe の 2 つの写しは `diff` で一致。
- `git grep` で、seat の pane の env を書いた箇所の残り: `/director` skill と `docs/recipes/director-launch-cmd.md` は director の pane の話で変更不要。`internal/cli/org.go:357` は L2。
- 確かめていない: テスト、静的検査、変異の実行(skill の規則と依頼どおり、走らせていない)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1 ルートの `ralph.toml` がテンプレートの全文の写しで、有効なキー 27 個がすべて明示されている(`model_pool`、`watcher_model`、`codex_verified`、`auto_merge` を含む)。`KNOWN_DIFFS` は path 単位の全面抑止なので、「違うのは 4 つの実施有無と先頭の 1 行だけ」は今は本当だが、それを保つ仕組みがない。`TestDefaultsLockStep` はテンプレートしか読まないため、次に `Default()` の値を変える PR(直前の #274 は 2 つ変えた)がテンプレートを直しても、ルートは古い値のまま `ralph org` と `ralph doctor` に読まれ、`check-sync.sh` も lock-step の test も通る。この repo は自分の org runtime で検証する repo なので、下流が受け取る既定とずれた設定で動く。tech-debt の表にも行がない。計画の R2 が見ているのは「今の値が既定と同じ」までで、将来のずれは見ていない。 | `ralph.toml`(`grep -c '^[a-z_]* = '` が 27)、`scripts/check-sync.sh:112-115` と `:135-141`(`is_known_diff` は path の完全一致)、`defaults_sync_test.go` が読むのは `templates/base/ralph.toml` だけ、`docs/tech-debt/README.md` に `ralph.toml` の `KNOWN_DIFFS` の行なし | いずれか。(a) ルートを `[standard]` と `[org.review]` の 2 つの表だけにして、ほかのキーを `Default()` に従わせる(`KNOWN_DIFFS` は要る)。(b) ルートを `Load` し、4 つの実施有無以外が `Default()` と等しいことを見る test を 1 つ足す。(c) 直さないなら、トリガーを「次に `Default()` かテンプレートのキーを変える PR」にして tech-debt に 1 行足す。#264 も同じ file を作るので、先に方針を決めておくと merge で迷わない。 |
| LOW | maintainability | L1 「座席の計画助言も `pick_reviewer` でもう一方の CLI を選べるようになる」という説明が、今の repo では成り立たない。`RALPH_PRIMARY_CLI` と `pick_reviewer` を読むのは `/cross-review` の step 2 だけで、`/plan` step 11 の計画助言は `codex` を直接呼び、driver を見ない。この説明は、実行時に stderr に出る警告文、`/org` skill の 4 つの写し、README、recipe の 2 つの写し、spec の注、`spawn.go` のコメント、test の定数に入っている。leader の計画助言(#267)が `pick_reviewer` を使うなら、それを足す版で書くほうが正しい。また警告文の `pick_reviewer falls back to claude` は、読み手には「reviewer が claude になる」と取れる。実際は driver を claude とみなし、reviewer は codex になる(取り違えるのは codex の座席だけ)。 | `.claude/skills/plan/SKILL.md:79`(`command codex -m … exec --sandbox read-only`、`pick_reviewer` なし)、`git grep pick_reviewer -- .claude/skills` は cross-review と org だけ、`scripts/xreview-helpers.sh:69-77`、`internal/org/driver/paneenv.go:64-65`、`spawn.go:994`、`.claude/skills/org/SKILL.md:31-33` と `:40-42`、`README.md:259`、`docs/recipes/isolated-herdr-server.md:243-245`、`herdr_test.go:428` | 警告文と文書を「cross-review」に絞るか、「計画助言も、`pick_reviewer` を使う版から」と条件を付ける。警告文は `a seat's plan advisory or ` を外すだけで済み、`org_runtime_env_test.go` の部分文字列(`may pick the wrong CLI as its other model`)は変わらない。`falls back to claude` は `takes claude as the seat's own CLI` のように書く。 |
| LOW | readability | L2 配線の場所のコメントが、この PR のあとで足りない。`internal/cli/org.go:357-360` は「The seat panes spawn creates carry driver.SeatPaneEnv: AGMSG_SELF_NAME=off … and RALPH_WIP_CHECKPOINT=off …」と、座席の pane の env を 2 つで言い切っている。S3 のあと、座席の pane は spawn が足す `RALPH_PRIMARY_CLI=<driver>` も持つ。変更した 3 か所(`herdr.go`、`paneenv.go`、`spawn.go`)のコメントは直っているが、実際の `NewHerdr` を呼ぶこの行は diff に入っていない。 | `internal/cli/org.go:357-361`、`internal/org/driver/paneenv.go:20-30`(こちらは「座席の pane にはもう 1 つある」と直っている) | 「`driver.SeatPaneEnv` は `NewHerdr` の固定の列で、座席の pane はこのほかに spawn が `TabCreate` の `extraEnv` で渡す `RALPH_PRIMARY_CLI=<driver>` を持つ」と 1 文足す。 |
| LOW | naming | L3 `max_cross_review_cycles` の数える単位が決まっていない。`config.go:568-571` とテンプレートの `[org.review]` は「cross-review の fix cycles の上限。標準フローが pipeline runs を上限するのと同じ」(既定 2)と書く。この repo の言葉では、標準フローの上限 2 は「全部で 2 回の run、最初の run と直しの再 run 1 回」(`post-implementation-pipeline.md`、cross-review の `CAP_REACHED = (cycle >= cap)`)。「fix cycles」は直しの回数 2 回とも読める。キー名は seed として下流の `ralph.toml` に残るので、単位を #269 の実装に任せたままだと、#269 と #276 の `max_pipeline_cycles` が別の数え方になりうる。 | `internal/config/config.go:568-571`、`templates/base/ralph.toml:246-248`、`.claude/skills/cross-review/SKILL.md:124`、`.claude/rules/ralph/post-implementation-pipeline.md` の「2 total runs」 | コメントとテンプレートに「2 は最初の run を含む。標準フローの `RALPH_STANDARD_MAX_PIPELINE_CYCLES` と同じ数え方」と書く。単位を #269 で決めるなら、「単位は #269 が決める」と書いて、「as ... caps」の言い切りを外す。 |
| LOW | readability | L4 `ralph config get` のメッセージと help に小さなずれが 3 つ。(a) `configKindName` の既定が `"a " + kind.String() + " value"` で、`org.max_seats.x` が `org.max_seats is a int value` になる(手で確認)。(b) 知らないキーの文が `not a key of one value in ralph.toml: ralph.toml has no such key` と、同じ語を 2 回言う。(c) help の `The value is one line on stdout` は、TOML の複数行文字列の値(`agmsg_home = """a\nb"""`)では 2 行出て成り立たない。skill が読むのは bool と整数なので実害はない。 | `internal/cli/config.go:54`、`:100`、`:136`。手で打った出力(Evidence の 3 つ目) | (a) の kind 名を `int` は `an integer`、`bool` は `a boolean` のように switch で返す。(b) は後ろの句を `no such key` に縮める。(c) は help を「文字列は書かれたとおり」とだけにするか、改行を含む値を exit 2 にする。 |
| LOW | readability | L5 test の作りに小さな脆さが 3 つ。(a) `TestConfigGet_HelpStatesTheExitCodeContract` が `Long` の改行位置まで含めて照合する(`"A key\nthe file does not set prints its built-in default"` と、5 字下げの続き行)。文を折り直すと、約束が変わっていないのに落ちる。(b) `TestConfigGet_NoRalphTomlPrintsTheDefaults` が、このPRと関係のない既定(`org.max_seats` の 5、`org.approval.mode` の `per-plan`)を直書きしている。`internal/config` の test がすでに固定している値なので、変えると 2 か所で落ちる。(c) 冒頭の「Each test chdirs into a temp directory」は、`TestConfigGet_HelpStatesTheExitCodeContract` と `TestExitCode_ConfigGet` には当たらない。 | `internal/cli/config_test.go:13`、`:34-36`、`:210-220` | (a) は `strings.Join(strings.Fields(stdout), " ")` に直してから照合する。(b) は `config.Default()` の値と比べる。(c) は「ほとんどの test は」に直す。 |
| LOW | maintainability | L6 まだ起きていないことへの参照が、merge のあとに古くなる。`scripts/check-sync.sh:113-114` の `#264 also writes this file` は、#264 が入ったあとは意味が変わる。`config.go:31` の `they start to in issue #266` と `:560` の `issues #267 and #269` も同じ(入ったあとは時制が合わない)。また `scripts/ralph-config.sh:3-4` の見出しは「[org] の既定を 1 つだけ lock-step のために持つ」と言うが、`RALPH_STANDARD_MAX_PIPELINE_CYCLES`(`:30`)にも `[org.review].max_cross_review_cycles` との照合ができた。シェル側にはその印がなく、変える人が知るのは test の失敗からになる(失敗の文面は両側を名指しするので、損は小さい)。計画の Non-goals が `ralph-config.sh` の変更を外しているため、これは意図した範囲。 | `scripts/check-sync.sh:112-115`、`internal/config/config.go:31`、`:560`、`scripts/ralph-config.sh:3-4` と `:30`、`defaults_sync_test.go:297-302` | `check-sync.sh` のコメントは「#264 の merge 後に、この行を直す」を避けて、理由(ルートは 4 つを on にする)だけにする。issue 番号の参照は残すなら「#266 が読む」のように現在形にしない。`ralph-config.sh` の見出しは、別の PR で 1 行足すか、触れないなら tech-debt に回す。 |

## Positive notes

- `configValue` が fail-closed に作られている。知らないキー、表、リスト、map、値の下のキーはすべて exit 2 で、値を読めない型が `Config` に足されても黙って空を出さない(`default` の枝)。
- `Load` をキーの照合より前に呼ぶ作りが、D6 の回帰(`standard.cross_review = true` と `[org].max_seats = 0` の同居は exit 3)を test で固定している。`"unknown command \"config\""` を含まないことも、cobra の 4 つの誤りのケースで確かめている。
- `TabCreate` の `extraEnv` は呼び出しごとの値で、`PaneEnv` を変えないことと、`--env` のない herdr で `extraEnv` ごと落ちて警告が `RALPH_PRIMARY_CLI` の条項を持つことが test で固定されている。director の pane に env が入らないことも `director_start_test.go` が見る。
- `checkEachKeyIsComparedOnBothSides` への切り出しで、`org` の行が元のまま残り、`[standard]` を同じ道具で見られる。行の key が probe の key と過不足なく一致するかも、`probe.Name()` つきで見る。
- 警告文の引数に、probe が一度だけ走って最初の呼び出しの `env` を名指しする、という制約が doc に書かれている(`paneEnvSupported`)。

## Notes(所見にしないもの)

- `ralph config get` は cwd の `./ralph.toml` を読む。ファイルのない cwd(例: repo のサブディレクトリ)では exit 0 と既定の `false` を、stderr に何も出さずに返す(手で確認)。D3 の決めどおりで、倒れる向きは off。#266 の skill は、repo の root で打つ前提を書くか、サブディレクトリでは root へ移ってから打つ必要がある。
- Go の panic も exit 2 で終わる。help の「2 は止まって報告」と同じ動きになるので、取り違えても害はない。
- 将来 `Config` に pointer や float の field が足されると、`config get` はその key を exit 2(`a ptr value`)にする。黙って出さない向きなので、足す人が気づける。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| M1 を直さずに merge する場合: ルートの `ralph.toml` が `KNOWN_DIFFS` の全面抑止の下で、テンプレートの全キーを明示したまま `Default()` から離れうる | 次の既定値の変更が、この repo の `ralph org` と `ralph doctor` に届かない | 計画 R2 が「今は同じ値」で受けた | 次に `Default()` かテンプレートのキーを変える PR、または #264 のルート `ralph.toml` | 計画 `2026-10-29-review-switches-config` R2、この報告の M1 |

_(行は docs/tech-debt/README.md に足していない。M1 を直すか、tech-debt に回すかは依頼元が決める。回すなら上の行をそのまま足す。)_

## Recommendation

- Merge: OK(CRITICAL と HIGH はなし。MEDIUM が 1 件)。M1 は直すか tech-debt に回すかの判断が要る。LOW の 6 件は、修正の cycle を回すなら一緒に直せる。
- Follow-ups: L1 は 11 か所に及ぶが、実行時の挙動には効かず、警告文の 1 句と文書の言い回しで済む。L3 は seed に残るキーの意味なので、#269 の前に決めておくと安い。/verify は `ralph.toml` のルートとテンプレートの差が 3 か所だけ(M1 の前提)であることと、`go run ./cmd/ralph config get standard.plan_advisory` が `true` を返すことを実測で見てほしい。
