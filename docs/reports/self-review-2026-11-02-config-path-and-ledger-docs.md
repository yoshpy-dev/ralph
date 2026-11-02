# Self-review report: config-path-and-ledger-docs

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-config-path-and-ledger-docs.md`(tech-debt の 421 行の (a)〜(d)、396 行の (c)、397 行の残り、S1〜S4、D1〜D3、AC1〜AC5)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1、Run 1)
- Scope: `git diff origin/main...HEAD`(base 350f22ea、HEAD ea8b4d16)。コードは S1 40cedc9e(`resolveConfig` がパスを付ける)と a6be3f5e(`ralph status` の warning)、S2 f185f22d(テストだけ)、S3 3311cf16(`config.go` の doc 2 か所)、S4 d257824e と b16f4bd0(README の 1 語、`/org` skill の 4 写し、条件の文のテスト)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。
- ID: 所見は L1〜L5。CRITICAL、HIGH、MEDIUM はなく、LOW が 5 件。

## Evidence reviewed

- `resolveConfig` と `resolveOrgConfig` の呼び出し元を非テストの Go で全部 grep した。`resolveOrgConfig` の呼び出しは 4 つ(`newOrgRuntimeAt` の `internal/cli/org.go:378`、`startDirector` の `org_director_headless.go:303`、`newOrgDirectorRuntime` の `org_director.go:395`、`statusOrgConfig` の `status.go:309`)で、`resolveConfig` を直接呼ぶのは `resolveOrgStartConfig`(`org.go:688`)だけ。`org: load config: ` で始まる文は `org.go:380`、`org.go:690`、`org_director.go:396`、`org_director_headless.go:305` の 4 つで、計画の数と合う。`load config` と `load ralph.toml` で grep して、ほかにこの文を組む本番コードはない。
- 4 つの文は、どれも `fmt.Errorf("org: load config: %w", err)` で包むだけで、自分では設定ファイルを名指さない。`status.go` の warning だけが `load ralph.toml: %v` と自分で名指していて、a6be3f5e が `load %v` に直した。`status.go` の warning の末尾「because ralph.toml could not be read」は残るが、`statusOrgConfig` は引数なしの `resolveOrgConfig("")` を呼ぶので、読むのは常に `./ralph.toml` で、文と合う。
- `errors.Is` / `errors.As` の鎖は、`resolveConfig` の `fmt.Errorf("%s: %w", path, err)` が `%w` を 1 つだけ使い、呼び出し元の `%w` がもう 1 層足すので、`config.Load` が返した error まで届く。`config.Load` の error は `fs.ErrNotExist` 以外では型を使わないので、型の判定が外れる所もない。`loadConfigFile` を差し替えるテスト(`countConfigReads`)の戻り値も同じ道を通る。
- `config.Load`(`internal/config/config.go:888`)の error の形を一通り読んだ。パスを名指すのは、`os.ReadFile` の error(`open <path>: ...`)、`checkLocalOnlyKeys`(`internal/config/local.go:151`、`path` を先頭に置く)、個人用ファイルの 3 つ(`config.go:923`、`:927` は `local.path` を先頭に、`:938` は末尾に `(with <local.path> applied)`)。TOML の構文 error と検証 error は名指さない。
- 計画の `org_start_plan_config_test.go`(547 行)の diff、`org_retired_state_dir_condition_docs_test.go`(新規 52 行)、`status_test.go` の 1 条件、`config.go` の doc 2 か所、README の 1 語、skill 4 写しの diff を全部読んだ。`.agents` の 2 写しは `.claude` の写しと、足した行が一字違わない。
- skill の英語の code span が本当に doctor の warn と `ralph org escalate --help` の言葉かを、`internal/cli/org_legacy_ledger.go:75`、`doctor.go:905`、`org_inbox.go:88` の使い方と、`TestRetiredOrgStateDirRefusalCondition`(`org_retired_state_dir_condition_test.go:118-122`)が両方に `Contains` で当てることで確かめた。skill の「この条件を英語でこう書く」は合っている。
- `config.go` の `StateDir` の doc の「`org.ResolveOrgStateDir` が `LocalOrgStateDirAt` で、main worktree の root の個人用ファイルを読む」を `internal/org/statedir.go:41-43` と `internal/config/local.go:205-221` に当てた。`LocalOrgStateDir` は `LocalOrgStateDirAt` を呼び、どちらも同じ値の検査をする。合っている。`Load` の doc は、`return Default(), err`(`config.go:910`)と合う。
- テストの `runtime.Caller(0)` で repo の root を求める形は、`internal/config/defaults_sync_test.go:67`、`internal/org/director_skill_sync_test.go:38`、`internal/cli/org_merge_test.go:175` など 10 か所以上にあり、この repo の慣習。`scripts`、`.github`、`Makefile` に `-trimpath` はない。
- 新しいテストのうち `TestOrgStartPlan_OutsideLedger_RecordsTheApprovalModeOfTheFileItRead` は、4 ケースの期待を、読んだファイルの方式と main の方式が必ず逆になるように作ってあるので、既定値(per-plan)を返す変異も、main の値を返す変異も、どちらかのケースで落ちる、と読み取れた。`TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml` の `max_total_seats` の対照 2 つは、同じ表の拒否の行(`max_total_seats = 1`)と同じ main の値を使うので、対照が空振りしない。実行はしていない(変異の結果は計画の Progress の記録を信頼した)。
- `commit.gpgsign=false` で grep すると、同じ add と commit の組が出る他のテストファイルは 6 つ(`config_test.go`、`org_approve_test.go`、`org_feature_test.go`、`org_legacy_ledger_test.go`、`org_feature_reviewer_test.go`、`org_gate_test.go`)。計画の Progress は「5 つ」と書くが、メッセージが `ralph.toml` のものは 3 つ(`org_approve_test.go`、`org_feature_test.go`、`org_feature_reviewer_test.go`)。数え方が違うので、/sync-docs が 421 行に書くときに数え直す。
- diff 全体に U+FFFD は 0 件。TODO、debug の出力、secret、範囲外のファイルはない。`docs/tech-debt/README.md` はこの PR で触っていない(計画が /sync-docs の担当と記録している)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1: 定数 `retiredOrgStateDirRefusalCondition` の doc が、この定数と同じ言葉を使う文を 2 つ(doctor の warn と `ralph org escalate` の help)しか挙げない。この PR は README の段落と `/org` skill の 4 写しも同じ定数の言葉にし、テストで結んだが、定数の側の doc は変わっていない(`org_legacy_ledger.go` は diff に入っていない)。定数を直す人が最初に読むのは定義の doc で、そこには README と skill が出てこない。直せば新しいテストが落ちて気づくが、落ちる前に分かる場所がない。 | `internal/cli/org_legacy_ledger.go:66-74`(「`ralph doctor`'s warn ... and the help of `ralph org escalate` ... say it in these words」)。結ぶ側は `internal/cli/org_retired_state_dir_condition_docs_test.go:12-19`。 | doc に「README の `[org].state_dir` の段落と `/org` skill の 4 写し(code span)も同じ言葉で、`TestRetiredOrgStateDirRefusalConditionIsWordedAsTheConstantInDocs` が結ぶ」と 1 文足す。 |
| LOW | naming | L2: 新しいテストの doc が tech-debt の表を「the 397 row of docs/tech-debt/README.md」と行番号で指す。表は行を足したり、/sync-docs が 396・397・421 行を書き換えたりして動く。同じ package の他のテストは、行の頭の言葉で指す(「the doctor-test row of docs/tech-debt/README.md」)。 | `internal/cli/org_retired_state_dir_condition_docs_test.go:15`。慣習は `internal/cli/cli_test.go:417` と `internal/cli/main_test.go:44`。 | 「the row about the refusal condition of RALPH_ORG_STATE_DIR」のように、行の内容で指す。 |
| LOW | readability | L3: `resolveConfig` の doc は「error は読んだファイルを名指し、`org: load config: ` の文はどれも、どのファイルが読めなかったかを言う」と書き、パスが 2 回出る例を 2 つ(ファイルの読み込み、個人用のファイルだけが持てるキー)挙げる。実際にはもう 1 つ形がある。個人用のファイルの error(`ralph.local.toml` の読み込み、構文、`agmsg_home`)は個人用ファイルのパスを先頭に置くので、`org: load config: ralph.toml: /path/ralph.local.toml: toml: ...` になる。先に名指される `ralph.toml` は問題なく読めたファイルで、読めなかったのは 2 番目に出るファイル。「どのファイルが読めなかったかを言う」は、この形では最初の名前について合わない。3 つの形のどれも、テストで形を固定していない(`org_start_plan_config_test.go` の新しいテストは、構文 error の `toml: ` だけを見る)。計画の Risks は「テストの 1 つで形を固定する」と書く。 | `internal/cli/org.go:645-664`。個人用ファイルの error は `internal/config/config.go:923`、`:927`、`:938` と `internal/config/local.go:76-85`(`readLocalFile` の error は `os.ReadFile` が自分のパスを持つ)。 | doc に「個人用のファイルの error では、そのパスが `ralph.toml` のあとに続く」の 1 句を足し、「どのファイルが読めなかったかを言う」を「読んだファイルを先頭に置く」に直す。形の固定は `resolveConfig` に 3 つの入力(読めないファイル、`[org].state_dir` を書いた `ralph.toml`、構文を誤った `ralph.local.toml`)を渡す表のテスト 1 本でできる(AC は /verify が見る)。 |
| LOW | maintainability | L4: S1 の代価が 1 つの文の読みやすさに出る。`ralph.toml` に `[org].state_dir` を書いた人が見る migration の error は、S1 の前は `org: load config: ralph.toml: [org].state_dir can be set only in ralph.local.toml ...` で、パスが 1 回、直すべきことまで読めた。S1 のあとは `org: load config: ralph.toml: ralph.toml: [org].state_dir can be set only in ...` と同じパスが 2 回続く。README は「an error that names its path and says to write the key in ralph.local.toml」と、この文を `ralph.toml` の読み込みの error の例に挙げている。D1 と計画の Assumptions が受け入れた代価で、`ralph config get` も同じ形で出す。ただし、受け入れたことが PR の Known gaps か tech-debt に残っていないと、あとで見た人はバグに見る。`resolveConfig` の中で直すなら `err.Error()` の接頭辞を文字列で比べることになり、勧めない。`Load` の側で直せば、計画の D1 が挙げた 3 か所の文が連れて変わる。 | `internal/config/local.go:151`(`path` を先頭に置く文)、`internal/cli/org.go:655-664`、README の `[org].state_dir` の段落(148 行)、計画の D1 と Assumptions の 2 つ目。`status.go` の warning は S1 の前も `load ralph.toml: ralph.toml: ...` と 2 回出していた(a6be3f5e は 2 回目を足さないように直したので、この形は変わっていない)。 | PR の Known gaps と /sync-docs の 421 行の書き換えに「パスが 2 回出る error が 2 種類ある(ファイルの読み込みと個人用だけのキー)。`config.Load` が自分で名指しているため。`Load` の側で直すなら 3 か所の文とテストが連れて変わる」と書く。 |
| LOW | readability | L5: 新しい docs テストの表が、3 つの位置引数と、同じ「code span で囲む」を 2 回手で書く形になっている。`dropBackTick bool` と `want string` が、「README は散文として backtick を外して比べる」と「skill は code span のまま比べる」の 2 つの比べ方を、片方は真偽値、片方は文字列の組み立てで表す。失敗の文の「(backticks removed = false)」は、skill の行では何を探したかを言わない。 | `internal/cli/org_retired_state_dir_condition_docs_test.go:28-36`(表)、`:46-48`(失敗の文)。 | フィールドに `form`("prose" か "code span")を持たせるか、表の `want` に `"`" + c + "`"` を 1 度だけ書く関数を使い、失敗の文は `%q` の `want` の前に「README の散文」「skill の code span」と書く。 |

### 依頼された点の判断

1. `resolveConfig` と `resolveOrgConfig` の呼び出し元の文の読み方: 4 つの `org: load config: ` の文と `resolveOrgStartConfig` の最初の文(同じ接頭辞)は、どれも `<接頭辞><path>: <cause>` の形になった。自分で設定ファイルを名指していたのは `status.go` の warning だけで、a6be3f5e が直した。残る 2 回出る形は 2 種類だけ(読み込みの error と、個人用だけのキーの error)で、どちらも `config.Load` の側が名指すためで、L3 と L4 に書いた。`resolveOrgStartConfig` の 2 つ目の文(main worktree の `ralph.toml`)は S1 の前から絶対パスを自分で名指し、`loadMainWorktreeConfig` が包まない error を返すので、2 回出る形は増えていない。
2. `errors.Is` / `errors.As` の鎖: `%w` が 1 つで、呼び出し元の `%w` が 1 層足すだけなので、鎖は切れない。`resolveConfig` が `cfg` を一緒に返す形は元のまま(`loadConfigFile` の戻りをそのまま返す)で、`resolveOrgConfig` は error のとき空の `OrgConfig` を返す。`ExitCode` も鎖を通る。
3. 新しいテストが言うことを打っているか: 打っている。`TestOrgStartPlan_UnreadableConfigStartsNothing` の root のケースは `org: load config: ralph.toml: ` を期待するので、S1 の前のコード(`org: load config: toml: ...`)では落ちる。`TestOrgSpawnStartAndDirectorStart_NameTheConfigFileTheyCouldNotLoad` は 3 つの動詞 × 2 つの渡し方で、期待に実際のパスを使う。`TestOrgStartPlan_OutsideLedger_RecordsTheApprovalModeOfTheFileItRead` と上限の対照は上の Evidence のとおり、期待が空振りしない形。`TestOrgStartPlan_RefusedLedgerStartsNothing` は git の外、`GIT_CEILING_DIRECTORIES` を cwd の親にして、リポジトリがどこにもない状態を作り、`errors.Is` と接頭辞と「設定を読んでいない」を打つ。S2 の変異 5 つは Progress の記録を信頼し、再実行はしていない。
4. README と skill の言い方: README は 1 語の直しで、定数の文と同じ。skill は日本語の文を残し、英語の定数の文を code span で足した。b16f4bd0 が「Go の定数と同じ」(scaffold した project にはない)を、doctor の warn と `escalate --help`(どちらも scaffold した project の `ralph` にある)に直したので、指す先は読者の手元にある。skill の写しは 4 つ(`.claude`、`.agents`、templates の 2 つ)で、足した行は一字違わない。skill には 4000 字を超える行がすでにあり(`.claude/skills/org/SKILL.md` の 247〜252 行)、足した行(124 行)の長さも、その並びでは目立たない。
5. README と SKILL.md を読むテスト: `runtime.Caller(0)` で repo の root を求める形は慣習どおり。README は backtick を全部外してから `Contains` で比べるので、段落の外に同じ文があっても通る(所見にするほどではない)。skill は code span が 1 行に収まっていることに頼る(テストの doc が書いている)。skill の写しを整形して折り返すと落ちるが、`sync-skills.sh` は折り返さない。`.agents` と templates の写しを読まない理由は、テストの doc が `check-skill-sync.sh` と `check-sync.sh` に任せると書いている。

## Positive notes

- 変更が `resolveConfig` の 1 か所に集まり、4 つの `org: load config: ` の文が同時に直った。文の接頭辞は変わらず、`HasPrefix` で固定する既存のテスト(`org_wait_director_test.go:274`)はそのまま通る。
- `status.go` の二重の名指しを、implementer が範囲の外で見つけ、orchestrator が `status_test.go` の既存の条件に「`ralph.toml: ralph.toml` を含まない」を足すだけで固定した。出力は S1 の前と同じに戻っている。
- `config.go` の doc の直しは、直す箇所が計画の指す 2 つに収まり、`StateDir` の doc は `statedir.go` の言葉(「main worktree の root」)と合う。
- skill の「Go の定数と同じ」を scaffold した project で読める文に直した b16f4bd0 は、templates に出荷する文書の読者を考えた直し。
- 秘密、debug の残り、範囲外のファイル、U+FFFD は見つからなかった。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(L1〜L3、L5 は PR の中で直せる LOW。L4 は D1 が受け入れた代価で、PR の Known gaps と 421 行の書き換えに書く) | - | - | - | - |

_(`docs/tech-debt/README.md` の 396・397・421 行の書き換えは、計画が /sync-docs の担当と記録済み。421 行に「同じ add と commit の組が他のテストファイルにある」を書くときは、ファイルの数を数え直す。上の Evidence に数え方の違いを書いた。)_

## Coverage gaps

- Go のテスト、`go vet`、静的解析、`check-skill-sync.sh`、`check-sync.sh` は打っていない(/verify と /test の担当)。S2 の変異の結果と各スライスの verify の rc は、計画の Progress の記録を信頼した。
- `org: load config: ` の 4 つ目の文(`org_director.go:396`、`director attach` と `director status`)は、パスを名指すことを打つテストがない。新しいテストは `spawn`、`start <task>`、`director start` で、`wait --inbox --director` の既存のテストは接頭辞だけを見る。4 つの文は同じ `resolveConfig` を通るので、直接のリスクは小さい。AC1 の範囲は /verify が見る。
- 個人用のファイルの error の形(L3)と、権限で読めないファイルの形は、どのテストも固定していない。

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5。どれも merge を止めない)
- Follow-ups:
  - L1 は doc の 1 文、L2 はテストの doc の 1 語。直すなら /verify の前に 1 コミットで直せる。出力もテストの結果も変わらない。
  - L3 は doc の 1 句と、形を固定する表のテスト 1 本(任意)。L5 はテストの書き方で任意。
  - L4 は直さず、PR の Known gaps と tech-debt の 421 行の書き換えに書く。
