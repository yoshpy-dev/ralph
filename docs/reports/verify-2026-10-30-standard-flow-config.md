# Verify report: standard-flow-config

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-standard-flow-config.md`(issue #276 の B1、AC1〜AC8、D1〜D8、Progress の S1〜S7)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main 0351800d、HEAD f1685e87、52 ファイル、+2946/−741)。spec の準拠と静的解析と文書のずれだけを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。probe は HEAD から build した binary(`<scratchpad>/v276b1/ralph`)と一時の git repo で打った。`ralph org start --plan` は herdr と agmsg が要るので打っていない(lead の指示)。
- Evidence: `docs/evidence/verify-2026-10-30-standard-flow-config.md`(コミットする。静的解析の要約、probe の出力、grep)。全出力は `docs/evidence/verify-2026-10-30-standard-flow-config.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-verify 自身のログは `docs/evidence/verify-2026-10-30-111611.log`(同じく対象外)。
- load average: static の直前 8.40(`uptime`)、go build の直前 9.73 と 7.97。30 を超えたことはなく、待たずに打った。timeout にも当たっていない。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 既定値(2、fable、xhigh、gpt-6-astra、xhigh)、`ralph.toml` の値と `ralph.local.toml` の上書き、検証のエラーが exit 3 でキーを名指す | 満たす | `internal/config/config.go:669`・`:672-675`(`Default()`)、`:746`(`reviewerClaudeEfforts`)、`:752`(`reviewerWordPattern`)、`:1033-1036`(上限の確かめと `ValidateReviewer`)、`:1044-1055`。probe(evidence の 2 節): ファイルがないとき 5 つの既定値、`ralph.toml` の 5 つの値、`ralph.local.toml` がキーごとに 3 つを上書き(linked worktree からも同じ値、`RALPH_LOCAL_CONFIG=off` で外れる)。上限 0 と −1、claude_effort の `minimal`・`XHIGH`・空、4 つの文字列の空、`$`、空白、単一引用符、バッククォート、`;`、末尾の改行は、どれも exit 3 でキー(`[standard].max_pipeline_cycles`、`[reviewer].<key>`)と値を名指した。`ralph.local.toml` 側の誤りも exit 3 で、`(with <path>/ralph.local.toml applied)` が付く。`us.anthropic.claude-opus-5-5-v1:0` と `arn:…:inference-profile/…@2026+beta` は exit 0 でそのまま出た。テストは `TestDefault_PipelineCyclesAndReviewer`、`TestLoad_ReviewerValidation`、`TestLoad_LocalFileOverridesPipelineCyclesAndReviewer`、`TestConfigGet_InvalidPipelineCapOrReviewerValueExits3` ほか(存在を確かめた。実行は /test) |
| AC2 skill は 5 つの値を設定から読み、4 つの環境変数は効かない。skill・スクリプト・雛形に `${RALPH_…:-…}` の読み出しが残らない | 満たす | plan 11.c(`.claude/skills/plan/SKILL.md:88`・`:95`)、cross-review 1.d・4(`.claude/skills/cross-review/SKILL.md:44-47`・`:72`・`:78`)は `ralph_setting` で読む。4 つの写しは check-skill-sync と check-sync で一致。probe(evidence の 3 節): 4 つの変数を 9/haiku/gpt-x/minimal に設定しても、偽の ralph の値・既定値・HEAD の binary の値はどれも変わらなかった(sh、bash、dash、`zsh -f`)。`ralph config get` 自身も変わらない(2 節の 3d)。計画の S6 の否定の grep の当たりは `tests/test-ralph-config.sh:225` の 1 件だけで、source のあとに変数が設定されていないことを確かめるテストの行。素の `$VAR` の形は 0 件 |
| AC3 `ralph_setting` の分け方(0 は値、127 と unknown command は黙って既定値、2・3・ほかは stderr に 1 行で既定値、表にないキーは空で失敗)、表は `Default()` と一致 | 満たす | `scripts/ralph-config.sh:44-94`(template と `cmp` で一致)。probe(evidence の 3 節、偽の ralph 9 通り × 4 つの shell): exit 0 は値、PATH にない(127)と `unknown command "config"` の exit 1 は黙って既定値。ほかの exit 1、2、3(stderr あり・なし)、0 で空、シグナル(143)は、stderr に `ralph-config.sh: ralph config get <key> exited <rc> (<ralph の stderr の 1 行目>); using the default <値>` を 1 行出して既定値を出し、0 を返した。表にないキーは空の出力、stderr に 1 行、戻り値 2。`set -eu` の呼び出し側も既定値で続いた。PATH の 6.0.0 では黙って既定値。表と `Default()` の照合は `internal/config/defaults_sync_test.go:371-397`(コードを読んで確かめた。実行は /test)。scaffold の probe(evidence の 6 節)でも、同梱の helper が既定値と注釈を外した値を読んだ |
| AC4 claude の reviewer に `--effort`(/plan、/cross-review、leader)。leader は `featureLeaderTask` の 2 行を使い、雛形に `ralph-config.sh` の source も `ralph config get reviewer` もない。`claude -p --effort` の確かめが `docs/evidence/` にある | 満たす(leader の側はコードとテストの存在まで) | /plan `:96` と cross-review `:78` の claude の呼び出しに `--effort "$effort"`。`featureLeaderTask` は `- 台帳:` の次に 2 行を書く(`internal/org/feature.go:486-489`)。空のフィールドは `featureReviewer`(`:387-402`)が既定値で埋め、`StartFeature` は watch gate の前に `config.ValidateReviewer` で断る(`:227-229`)。`ralph org start --plan` は `resolveConfig` の `cfg.Reviewer` を渡す(`internal/cli/org.go:638-644`、`:1144-1151`)。leader.md は `:527-539` で 2 行の読み方を書き、codex の `-m` と `model_reasoning_effort`(`:578`)、claude の `--model` と `--effort`(`:592`)に置き場所を使う。leader.md の否定の grep(`ralph-config`、`ralph config get reviewer`、4 つの変数名)は 0 件。テストは `TestFeatureLeaderTask_Reviewer`、`TestStartFeature_LeaderTaskCarriesTheReviewer`、`TestStartFeature_RefusesReviewerValuesThatSplitTheShell`、`TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings`(ralph.toml と ralph.local.toml の 3 通り)、`TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryTakesTheReviewerFromTheTask`(`:` を含むモデル名で 2 語を取り出す)。evidence は `docs/evidence/claude-effort-probe-2026-10-30.md`(tracked) |
| AC5 有効な上限は設定と正の整数の `cap` の大きいほう。上げるときは `cap: cycle + 1` を書き `cycle` は増やさない。implement 3 は `cap` を保ち、`/pr` で消える | 満たす | cross-review 1.d(`:44-47`)、8 の `CAP_REACHED = (cycle >= cap)`(`:142`)、9 の Option 1(`:184-189`)。implement 3.c(`.claude/skills/implement/SKILL.md:29`)と 13.e(`:59`)。`/pr` は `.claude/skills/pr/SKILL.md:60` で `rm -f`。probe(evidence の 4 節、skill から grep で取り出したコマンドをそのまま打った): 読む jq は `cap` 3 で `3`、なし・`"3"`・2.5・0・−1・null・true・壊れた JSON・ファイルなしで空。書く jq は `{"cycle": 2}` を `"cap":3` にし、`cycle` は 2 のまま。`cycle` が文字列、壊れた JSON、jq が PATH にないときは元のファイルが残り、tmp も残らず、表示に新しい `cap` は出ない(9 の「出なければ止まる」に当たる)。書いたあと読むと `3`。4 つの写しにそれぞれ 1 回ずつある |
| AC6 `ralph doctor` が 4 つの変数(空でない値)に代わりのキーを名指す warn を出し、未設定・空では何も出さない | 満たす | `internal/cli/doctor.go:895-900`、`:903-909`(`retiredSettingEffect`)。probe(evidence の 5 節、4 回の実行の diff): 未設定と空の出力は同じ。4 つを設定すると 4 行が増え、それぞれ `[standard].max_pipeline_cycles`、`[reviewer].claude_model`、`[reviewer].codex_model`、`[reviewer].codex_effort` を名指した。1 つだけなら 1 行。rc はどれも 0 で、✗ の行はない |
| AC7 文書を新しい置き場所と既定値に合わせ、4 つの変数に触れるのは「もう読まない」の説明だけ | 満たす(/sync-docs に回す残りは下の表) | model-routing.md(sync note `:69-89`、full ID の pin と codex の項 `:96-108`、effort `:121-130`、「Where the values live」`:169-188`)と template の写し。post-implementation-pipeline.md `:116-118`、subagent-policy.md `:101-104`、definition-of-done.md `:34`、codex-setup.md `:92-109`(それぞれの写しは check-sync で一致)、template の ralph.toml の注釈(`:37-72`、`[org.review]` の注釈 `:294`)。4 つの変数名の `git grep`(記録を除く)は、rules と helper の「no longer reads / change nothing」の説明、doctor とテストの表、tech-debt の過去の記録(`:134`・`:162`・`:215`)だけ |
| AC8 build、gofmt、vet、golangci-lint、check-sync、check-skill-sync、run-verify | 一部(静的の部分は満たす) | `./scripts/run-static-verify.sh` rc 0(scope は `full fallback (unclassified:scripts/ralph-config.sh)` で Go 全体が走った)、`go build ./...` rc 0。run-verify の既定モード(テストを含む)は /test の範囲で、ここでは打っていない |

計画の Verify plan の「`standardTemplateDrift` の直し方」も確かめた。`standardTemplateDrift`(`internal/config/defaults_sync_test.go:201-211`)は 2 つの switch だけを見て、注釈にした 5 つのキーは `commentedTemplateDrift`(`:236-250`)が「template に set されていない、注釈がちょうど 1 回、値が `Default()` と同じ」で見る。scaffold の probe でも、`ralph init` の ralph.toml は 5 つのキーを注釈だけで持ち、注釈を外すと helper がその値を読んだ。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | changed の既定でも `scripts/ralph-config.sh` が分類されず、Go は全体で走った。gofmt `ok`、go vet と staticcheck は無出力(Skipping の行なし)、golangci-lint `0 issues.`。local verifier(shellcheck `--severity=warning`、`sh -n`、Codex の hook の検査)、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。secret scan は `0351800d..f1685e87` で clean。打ち直しはしていない |
| `go build ./...`、`go build -o <scratchpad>/ralph ./cmd/ralph` | pass(rc 0) | go1.27.3。前後で `git status --porcelain` は空 |
| `shellcheck <file>`(既定の severity) | info だけ | `scripts/ralph-config.sh`(と写し)と `tests/test-ralph-config.sh` は rc 0。`tests/test-codex-exec-invocation.sh` は SC2016(info)5 件(origin/main は 4 件。単一引用符の中の `$` を literal として探す行)、`tests/test-insights-append.sh` は SC2329 1 件(origin/main と同じ)。どちらも `--severity=warning` で rc 0 |
| `./scripts/plan-visual.sh digest <plan>` | `d8e03ef62de9` | `- Approved:` の digest と同じ |
| `git diff --check origin/main...HEAD`、変えたファイルの U+FFFD | どちらも空 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/ralph/model-routing.md`(root と template、KNOWN_DIFF) | yes | sync note、full ID の pin、codex の slug、effort、「Where the values live」が `[reviewer]`・`[standard].max_pipeline_cycles`・helper・leader の 2 行で書かれている。「doctor が見るのは `[org].model_pool` の slug で、その既定も同じ slug」は `config.go:684` で確かめた |
| post-implementation-pipeline.md、subagent-policy.md、definition-of-done.md、codex-setup.md(と写し) | yes | export の文はなくなり、上限は設定と `cap` の大きいほう |
| `templates/base/ralph.toml` | yes | 注釈は helper の振る舞い(黙る 2 つの場合、ほかは stderr に 1 行)と Load の検証(1 以上、5 つの effort、許す文字、空はエラー)に合う |
| `/org` skill の 4 つの写し | yes | `:689-692` と `:711-720` にタスクの 2 行と読む元 |
| `docs/tech-debt/README.md:161` の行 (b) | /sync-docs に回す(lead の指示) | Why deferred が「changing it means changing what `/cross-review` step 9 writes」と書く。step 9 は `cap` を書くようになったが、`cycle` は今も増やさないので (b) の事実は残る。1 文の追記で足りる |
| `ralph org start --help` | /sync-docs に回す(lead の指示) | persistent flag `--config` の説明(org の verb ごとに読むキーを列挙)に、start が `[reviewer]` を `--config` か `./ralph.toml`(と `ralph.local.toml`)から読むことがない |
| `internal/config/config.go:883`・`:935`、`config_test.go:419` の `PipelineConfig` | /sync-docs に回す(lead の指示、前から) | 行番号は lead の 875・927 から 8 行ずれた。origin/main では `config.go:821`・`:873`、`config_test.go:418` |
| `ralph config get --help` の exit code の案内(`internal/cli/config.go:66-75`) | no(V1) | 下の Findings |
| README の `ralph config get` の行(`:135`)と `ralph.local.toml` の節(`:142-148`) | yes(任意で足せる) | どちらも例を挙げるだけで、新しいキーに触れていないが誤りはない。`ralph.local.toml` の節は計画で B2 が扱う |
| 計画の Rollout の「revert した ralph は `[reviewer]` を知らないキーとして無視する(推測、未確認)」 | 確かめた(I3) | 下の Observational checks |

## Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | doc drift | `ralph config get --help`(`internal/cli/config.go:66-75`)は「Exit codes, for the skills and seats that read a setting with it」として、exit 2 と 3 を「stop and report it, do not treat it as off」と案内する。この PR で skill が `ralph config get` を呼ぶ道は `ralph_setting` になり、exit 2・3 では stderr に 1 行出して既定値で続ける(D2、`/plan` 11.c は「the launch goes on with the default; pass that line on to the user」)。「知らせる」は満たすが「止まる」は満たさず、help と skill の手順が違うことを言う。計画の D2 は help の約束を「知らせる」の側で受けると書いており、挙動は計画どおり | /sync-docs で help に 1 文足す(`scripts/ralph-config.sh` の `ralph_setting` が読む `[standard].max_pipeline_cycles` と `[reviewer]` は、2・3 で 1 行知らせて既定値で続ける)。または help の対象を review の switch に絞る書き方にする。merge は止めない |
| I1 | INFO | skill の jq | 1.d の読む jq は、整数の値でも書き方によって `3.0` や `1E+1` を出す(evidence の 4 節)。skill の 9 だけが `cap` を書き、`.cycle + 1` は `3` の形になるので、今の手順では起きない | なし |
| I2 | INFO | skill の jq | 9 の書く jq は `cycle` のない cycle-count.json に `"cap":1` を書く。1.c が `cycle` を必ず初期化するので、今の手順では届かない。表示に `cap` は出るが上限は上がらない | なし |
| I3 | INFO | rollback | 計画の Rollout は、revert した ralph が `[reviewer]` を無視するかを「推測、未確認」としている。PATH の ralph 6.0.0 の doctor は、5 つのキーを有効な行で書いた ralph.toml で rc 0、`All checks passed.`、ralph.toml の warn も parse error も出さなかった(evidence の 6 節)。直前の release では確かめられた | /sync-docs か PR 本文で「6.0.0 で確かめた」と書ける(任意) |
| I4 | INFO | shellcheck | `tests/test-codex-exec-invocation.sh` の SC2016(info)が 4 件から 5 件になった。単一引用符の中の `$` を literal として grep する行で、意図どおり。repo の gate(`--severity=warning`)は rc 0 | なし |

## Observational checks

- `ralph config get`、`ralph_setting`、cross-review の jq、`ralph doctor` を、HEAD の binary と一時の git repo で打った(evidence の 2〜5 節)。
- `/plan` 11.c の 1 行目(`. ./scripts/ralph-config.sh; model=$(ralph_setting …); effort=$(ralph_setting …)`)を、Bash tool の shell(zsh 5.9、ユーザーの設定を読んだまま)でそのまま打った。PATH の 6.0.0 では既定値を黙って返し、HEAD の binary を前に置くと fable、xhigh、上限 2 を返した。この repo のルートの ralph.toml は `[reviewer]` を書かず、主 worktree の root に `ralph.local.toml` はない。
- `ralph init --yes`(HEAD)の scaffold で、同梱の helper が既定値を読み、注釈を外した 5 つの値も読んだ。同じ ralph.toml を PATH の 6.0.0 の doctor が警告なしで読んだ(I3)。6.0.0 の doctor は scratch の repo で打ち、その repo の `.harness/state/org` にはキャッシュを書かなかった(「not cached: there is no org ledger yet」)。HEAD の doctor は `--state-dir <scratchpad>/ac6state` に `merge-guarantee.json` を書いた。どちらも scratchpad の中。
- 一時ディレクトリの後片付けを TMPDIR で数えようとしたが、macOS の `mktemp -d` は TMPDIR を見ないので、その数は使っていない。後片付けは `scripts/ralph-config.sh:76`(どの return より前の `rm -rf`)と `tests/test-ralph-config.sh` の偽の mktemp の照合(存在を確かめた)による。
- herdr、agmsg、codex、claude は打っていない。

## Coverage gaps

- Go のテスト、`tests/test-*.sh` の結果(/test の担当)。AC3 の表と `Default()` の照合、AC4 の leader の側は、コードとテストの存在まで。
- `ralph org start --plan` を本物の herdr と agmsg で起こし、leader が TASK の 2 行で codex と claude を呼ぶ通し。
- reviewer を実際に呼ぶ道(codex の `-m`・`-c`、claude の `--effort`)を `[reviewer]` の値で動かすこと。`claude -p --effort` は `docs/evidence/claude-effort-probe-2026-10-30.md` が haiku でだけ確かめている(fable と medium・high は未確認と、その evidence 自身が書いている)。
- Codex の driver で skill を動かす道(`.agents/skills/` の写しは文字列として一致することまで)。

## Verdict

pass

- Verified: AC1、AC2、AC3、AC5、AC6、AC7 は、コードの行と、HEAD の binary・偽の ralph・skill から取り出した jq の probe、grep で確かめた。AC4 は skill の呼び出しの形、`featureLeaderTask` と leader.md の行、テストの存在、evidence のファイルで確かめた。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、secret scan)はすべて通り、計画の digest は承認時と同じ。
- Partially verified: AC4 の leader の側(herdr と agmsg がいるので probe していない)。AC8 は静的の部分だけ。
- Not verified: Go と shell のテストの結果(/test)、本物の reviewer の呼び出し、Codex の driver での skill。
- 所見: LOW 1 件(V1、`ralph config get --help` の exit 2・3 の案内と `ralph_setting` の振る舞い)、INFO 4 件。どれも merge を止めない。
- /sync-docs に回すもの: `docs/tech-debt/README.md:161` の (b)、`ralph org start --help` の `[reviewer]` の読む元、`config.go:883`・`:935` と `config_test.go:419` の `PipelineConfig`(前から)、V1 の `ralph config get --help` の 1 文。任意: I3 の確かめを計画の Rollout か PR 本文に書く。B2 の範囲の 5 つの変数は、この PR では env で読むままで正しい。
- Insight event: `docs/insights/events/2026-10-30-standard-flow-config.jsonl` に verify の行を 1 行足した(`--verdict pass --cycle auto --low 1`)。
