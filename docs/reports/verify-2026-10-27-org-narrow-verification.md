# Verify report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(AC1〜AC6、D1〜D6、Verify plan、Progress checklist の S1〜S5、直し F1〜F10 と 246c7432)。issue #262 の受け入れ条件 4 つ
- Verifier: verifier サブエージェント(`/verify`、cycle 2 の回し直し(cap 3)。cross-review 2 回目の ACTION_REQUIRED 2 件を直したあとの回し直しで、`cycle-count.json` は skill の決まりで 2 のまま)
- Scope: `git diff 622cdcfb...HEAD`(HEAD 83602c22、31 ファイル、+2590/-112)。前回の verify(7458318d)のあとに入ったのは、直し F9 3c64d65d(org の雛形 2 つ、`/org` skill の 4 つの写し、`internal/org/prompts_verification_test.go`)、F10 662d3e47(plan の雛形の 6 つの写し、`tests/test-slice-verification.sh`、org の雛形 2 つの `go.work` の言い方)、inline の 246c7432(R3-3、org の雛形 2 つとテストの定数)、sync-docs 6a5bed1e(仕様 2 つ、tech-debt、計画の Progress)と記録のコミット。`git diff 7458318d..HEAD` は 23 ファイル、+350/-234。テストでない `*.go` の変更は今回も 0 件。テストは打っていない(`/test` の範囲)。観察の probe では `go vet`・`gofmt`・`go env` だけを scratchpad の使い捨ての module で打ち、`go test` は打っていない
- Evidence: `docs/evidence/verify-2026-10-27-org-narrow-verification.log`(gitignore。`run-static-verify.sh` の全出力、`check-sync.sh` と `check-skill-sync.sh`、digest、写しの `cmp`、コマンドの形の数え上げ、入れ子の module と `GOFLAGS` の probe、guard の判定、Go のソースの該当箇所、probe のスクリプト)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 leader の TASK の例に `VERIFY:` の行があり、機能ごとの org の手順 1 にスライスごとの検証コマンドと `RALPH_VERIFY_SCOPE=changed` の注意がある | Met | コマンドが変わったので読み直した。implementer への TASK の例は `internal/org/prompts/leader.md:493-504` で、`:503` の `VERIFY:` の行は新しい形(`… && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/foo/`)。手順 1 は `:145-155`(「— verify: `<コマンド>`」は `:150`、`RALPH_VERIFY_SCOPE=changed` の注意は `:153-155`)で、7458318d から変わっていない。手順 1 が指す「ミッション」の 3(`:28-48`)は、コマンド(`:34`)、timeout の理由(`:36-40`)、入れ子の module の `(cd <module> && …)`(`:40-45`)を書く |
| AC2 implementer の RESULT の契約に 5 項目の結果と検証コマンドの結果があり、既定が `run-verify.sh` ではなく、いちばん狭い検査 | Met | 既定は `internal/org/prompts/implementer.md:32-55`。`VERIFY:` の行がなければ、触ったパッケージに `implementerGoSliceCheck` と同じ形のコマンドを打つ(`:39`)。timeout の理由は `:41-45`、入れ子の module は `:45-50`。`run-verify.sh` は、狭い検査でスライスのファイルを覆えないときだけ使う(`:52-54`)。RESULT の例は `:127-128`(`VERIFY: … → pass` と `SELF-CHECK:` の 5 項目)、契約は `:165-178`。5 項目の名前と順は変わっていない |
| AC3 機能ごとの org のレビューの TASK は `run-test.sh` を打たない。指示があればそれだけ、なければ両方。gate の TASK は今のまま | Met | `internal/org/prompts/reviewer.md` は 7458318d から変わっていない(`git diff --quiet` が rc 0)。`leader.md` の 7458318d からの hunk は `:34`、`:36-45`、`:503` の 3 つだけで、手順 8 と gate の手順 9〜16 には触れていない。テストでない `*.go`、`packs/`、`templates/packs/`、`internal/cli/`、`scripts/` の差分は 0 件 |
| AC4 `/org` skill の「役割」と「座席内 fan-out」が S1〜S3 と食い違わず、4 つの写しが一致する | Met | 「役割」の implementer の箇条(`.claude/skills/org/SKILL.md:1324-1333`)は、新しいコマンド(`:1330`)と入れ子の module の `(cd <module> && …)`(`:1331-1333`)を書く。雛形の文の短い版で、食い違いはない。3 つの写しは root と `cmp` で同一。「Leader 運用 2 経路」の手順 3 には Go の形がない(下の LOW 1) |
| AC5 quality-gates・recipe・仕様の reviewer の記述が S2 と食い違わない | Met(S5 の範囲) | reviewer の記述は 7458318d から変わっていない。sync-docs 6a5bed1e が仕様 2 つに足したのは `RESUME:` の行と片付けの境界で、reviewer の文には触れていない |
| AC6 雛形のテストと skill の同期の検査が通る | Partially(静的の半分) | `check-skill-sync.sh`(14 skill が一致)と `check-sync.sh`(DRIFTED 0)は通った。`go vet ./internal/org/` はテストのファイルを含めて rc 0 で、F9 と 246c7432 が直したテストはコンパイルできる。`tests/test-slice-verification.sh` は `sh -n`・`bash -n`・`shellcheck` が rc 0。テストの実行は `/test` に残す。テストを AC の文が名指す `prompts_test.go` ではなく `prompts_verification*_test.go` に置いたことは、F2・F3・F5 で Progress に記録済み |

issue #262 の受け入れ条件:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 1. leader がスライスごとの検証コマンドを計画と TASK に書く | Met | AC1 と同じ |
| 2. implementer の RESULT に機械的な確かめと絞った検証の結果がある | Met | AC2 と同じ |
| 3. reviewer のレビューの TASK が `run-test.sh` を打たない | Met(機能ごとの org) | AC3 と同じ。D6 のとおり、指示のないレビューでは両方を打つ |
| 4. 雛形のテストと同期の検査が通る | Partially | AC6 と同じ。テストは `/test` |

cross-review 2 回目の指摘(triage の ACTION_REQUIRED #1 と #2)の解消:

- 解消した。どちらも、狭めた Go の検査が、前の `run-verify.sh` にあった言語パックの性質を落としたという指摘である。#1 は `-timeout=30m` の既定、#2 は module ごとの root に入ることを指す
- コマンドの形がそろっていること(証拠のログの「command forms」):
  - 対象は雛形 2 つ(`internal/org/prompts/`)、`/org` skill の 4 つの写し、plan の雛形の 6 つの写しにある Go の検査で、全部で 20 個(implementer 2、leader 2、skill 4、plan の雛形 12。plan の雛形は主の例と入れ子の例が 1 行に 2 つ)
  - `internal/foo` と `pkg` を `<dir>` に置き換えると、20 個すべてが `test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./<dir>/` の 1 つの形になった
  - `go vet ./<dir>/ && go test` の裸の形は、雛形、skill、agent 定義、plan の雛形、quality-gates、recipe、仕様のどこにも残っていない(`git grep` の rc 1)
- 写しの一致: `/org` skill の 4 つは `cmp` で同一。plan の雛形は、`feature-plan.md` の 2 つ、`plan/template.md` の 4 つがそれぞれ `cmp` で同一で、6 つの 39 行目の sha256 も同じ
- 雛形 2 つの理由の文(timeout の 2 文と入れ子の 1 文)は、改行を詰めて取り出すと sha256 が同じ。テストの定数 `goSliceCheckTimeoutSentences`(`internal/org/prompts_verification_test.go:24`)と `goSliceCheckNestedModuleSentence`(`:33`)が両方の雛形のミッションの項目を照合する(`TestRenderRolePrompt_GoSliceCheckKeepsThePackTimeoutAndEntersANestedModule`、`:489`)。RESULT と TASK の例は `:184` と `:793` が定数から作った形で照合する。skill の箇条は `TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck`(`:532`)、plan の雛形の 2 つの例は `tests/test-slice-verification.sh:122-125` が照合する

timeout と入れ子の module の文を、言語パックと Go に照らして読んだ:

| 雛形の文 | 照らしたもの | 読み |
| --- | --- | --- |
| `-timeout=30m` は言語パックの既定(`run_tests`)で、10 分より長いパッケージが `go test` の既定の 10 分で打ち切られないようにする | `packs/languages/golang/verify.sh:108-116`。`templates/packs/golang/verify.sh` と `cmp` で同一 | 正しい。`run_tests` は `GOFLAGS="-timeout=30m${goflags:+ $goflags}" go test ./...` を打つ(`:115`) |
| `GOFLAGS` の先頭に置くので、環境の `GOFLAGS` か `go env -w` で timeout が決めてあれば、そちらがあとに並んで効く | Go 1.27.3 の `cmd/go/internal/base/goflags.go`(`quoted.Split` で分け、`flags.Set` を順に呼ぶ)と `go help environment` の GOFLAGS の項。`GOENV` を scratchpad のファイルに向けた probe | 正しい。展開した値は、何もないとき `-timeout=30m `、環境の `GOFLAGS=-timeout=45m` で `-timeout=30m -timeout=45m`、`go env -w GOFLAGS=-timeout=50m` で `-timeout=30m -timeout=50m`。両方あれば環境が勝つ(`-timeout=30m -timeout=45m`)。これは Go の決まりと同じ |
| 入れ子の module のパッケージは `(cd <module> && …)` で包む(root に `go.work` がなければ、Go は root から入れ子の module に入らない) | `find_project_roots`(`verify.sh:20-35`、すべての `go.mod` のディレクトリ)と、root ごとに `cd` する本体(`:133-138`)。scratchpad の 3 つの配置で `go vet` を打った | 正しい。root に `go.mod` がない配置 A では root からの `go vet ./service/pkg/` が rc 1(「go.mod file not found in current directory or any parent directory」)。root と入れ子の 2 つの module の配置 B でも rc 1(「main module (example.com/root) does not contain package example.com/root/service/pkg」)。`(cd service && …)` はどちらも rc 0。B に root の `go.work` を足した配置 C では root からも rc 0 で、`(cd service && …)` も rc 0 のまま |

言語パックとの小さな違い(Info、直さなくてよい):

- 雛形の形は、`go env GOFLAGS` が空のとき値の末尾に空白が 1 つ残る。`quoted.Split` が空白を読み飛ばすので害はない。`GOFLAGS="-timeout=30m " go vet` と雛形の形の `go vet` はどちらも rc 0 で、`GOFLAGS=-bogusflag go vet` は rc 1 なので、`go vet` は実際に `GOFLAGS` を読んでいる
- 言語パックは `go env GOFLAGS` が失敗したときに `$GOFLAGS` へ戻る。雛形にはこの戻りがない。ただ `go env` は `GOFLAGS` の読み取りの誤りを隠して値を出す(`goflags.go` の `hideErrors`。`GOFLAGS=-bogusflag go env GOFLAGS` は rc 0)ので、戻りが要る場面はおそらくない。未確認です

self-review の cycle 2 の 3 回目(ea7a6a5b、LOW 3)を HEAD で読んだ:

- R3-3: (a) は 246c7432 で解消した。「呼び手」は雛形 2 つとテストの定数から消え、`git grep '呼び手'` は記録の外で 0 件。(b) テストの doc(`prompts_verification_test.go:29-32`)の崩れは直り、文として読める。残るのは "the sentence, in the same two places, that runs implementerGoSliceCheck" の言い方だけで、self-review が勧めた "tells" にはなっていない(Info)
- R3-1(`prompts_verification_test.go` が 891 行)と R3-2(plan の雛形の理由の文を shell の pin が照合しない)は HEAD のまま。計画(`:209`)のとおり、sync-docs が tech-debt の R2-5 の行に足す。`docs/tech-debt/README.md` にはまだない(下の Documentation drift)

計画の digest: `./scripts/plan-visual.sh digest` が `97abe345a188` を返し、`- Approved: 2026-10-27 sha256:97abe345a188`(`:4`)と一致する。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | 言語の範囲は、分類できない `tests/test-slice-verification.sh` があるので full に倒れ、選ばれた pack は golang だけ。golang の pack は `gofmt: ok`、`go vet` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし。local verifier の shellcheck、hook の `sh -n`(root と template の 10 本ずつ)、guard の awk の構文、settings の `jq`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照はすべて通った。secret scan は 622cdcfb..83602c22 で clean。31.8 秒。wrapper 自身のログは `docs/evidence/verify-2026-10-27-224822.log` |
| `./scripts/check-skill-sync.sh` | rc 0 | 14 skill が一致 |
| `./scripts/check-sync.sh` | rc 0、PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `test -z "$(gofmt -l internal/org)"` と `go vet ./internal/org/` | rc 0 | テストのファイルを含む。go1.27.3(go.mod と同じ) |
| `sh -n`・`bash -n`・`shellcheck tests/test-slice-verification.sh` | rc 0 | F10 が足した 2 つの assert を含む |
| `git diff --check 622cdcfb...HEAD` | rc 0 | 出力なし |
| 変更したファイルの U+FFFD | 0 件 | |
| guard の判定(`pre_bash_guard.sh`、root と `templates/base/`、default と bypassPermissions) | 新しい形・入れ子の形・前の形はすべて判定なし | 陽性の対照(`--no-verify` 付きのコミット。スクリプトの中で組み立てた)は 4 つとも deny。座席が雛形のコマンドを打っても guard には止められない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/specs/2026-08-01-org-runtime.md:35` の「2026-10-27 改訂」(a)、`docs/specs/2026-10-07-org-multi-org-director.md:127` の FR-7「2026-10-27 に決めたこと」 | Yes(Info) | (a) は「Go なら触ったパッケージの gofmt・vet・test」、FR-7 は「触ったところを覆ういちばん狭い検査」と書く。新しいコマンドも触ったパッケージの gofmt・vet・test で、2 つの文は今も正しい。仕様はコマンドの形を書かないので、偽になった文はない。timeout と入れ子の module はコマンドの中身で、テストが雛形の文として照合している。仕様に足さなくてよい。足すなら (a) の括弧に「test は言語パックと同じ `-timeout=30m` を `GOFLAGS` の先頭に置き、入れ子の module ではその root で打つ」の 1 句で、任意 |
| `docs/quality/quality-gates.md:82`、`templates/base/docs/quality/quality-gates.md:81` | Yes | 「the slice's verification command from the leader's TASK and a mechanical self-check on the staged diff」はコマンドの形を書かないので、今も正しい。7458318d から変わっていない |
| `docs/recipes/codex-seat-permissions.md` と写し | Yes | 7458318d から変わっていない。`cmp` で同一(check-sync の IDENTICAL に入る) |
| `docs/tech-debt/README.md` の R2-5 の行(`:381`、"Test housekeeping left in the org implementer's self-check and resume tests") | 未記録(既知、`/sync-docs` へ) | 計画(`:209`)が sync-docs に渡した R3-1 と R3-2 は、この回し直しの sync-docs がまだ走っていないので HEAD にない。self-review の台帳の案(報告の `:76`)は (e) R3-1、(f) R3-2、(g) R3-3 の 3 つを並べるが、(g) の (a) は 246c7432 で直り、(b) の文の崩れも直った。台帳に足すのは (e) と (f) だけでよい。(b) の残り(上の Info)を書くかは任意 |
| `.claude/skills/org/SKILL.md:1456-1461`(「Leader 運用 2 経路」の手順 3)と 3 つの写し | 一部(LOW 1) | 下の所見 |
| 計画の記録 | 情報 | Progress の F9・F10・246c7432 の記録は、各コミットの `git show --stat` と合う。`:202` は consult の probe の結果を「cannot find main module」で落ちると書くが、手元の go1.27.3 の文言は配置 A で「go.mod file not found in current directory or any parent directory」、配置 B で「main module … does not contain package …」だった。落ちるという結論は同じ。plan の雛形の英語の文は "a timeout the caller set" のままで、org の雛形の R3-3 の言い換え(「環境の `GOFLAGS` か `go env -w` で」)と語が違う。言語パックのコメントの語で、指すものは同じ。残る `[ ]` は `PR created` だけで、digest は一致 |

所見 LOW 1: `/org` skill の route (A) の leader の手順 3 に、Go のコマンドの形がない。

- 場所: `.claude/skills/org/SKILL.md:1456-1461`。`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行も同じ
- 中身: 手順 3 は、implementer への TASK の `VERIFY:` の行に「そのスライスの検証コマンド(スライスが触ったファイルを覆う、いちばん狭い検査)」を書くよう言う。Go の形(`GOFLAGS="-timeout=30m …"` と `(cd <module> && …)`)も、その形を書いた「役割」の implementer の箇条(`:1328-1333`)への参照もない
- 影響: (A) 現セッション昇格の leader は雛形を読まず、この skill だけを読む(計画 `:175` の S4 の記録)。分割計画にも結びつかないので、plan の雛形の欄からコマンドを写すこともない。この leader が Go のスライスに裸の `go test ./<dir>/` を書くと、implementer はそれをそのまま打つ。そのため triage #1 の「leader が書くコマンドにも timeout を残す」と #2 の module の root が、この経路では文で守られない。(B) headless の leader は雛形のミッションの 3(`leader.md:28-48`)に形があり、機能ごとの org は plan の雛形の欄を写すので、どちらも当たらない。当たるのは (A) の経路で、Go のパッケージのテストが 10 分を越えるか、入れ子の module にあるときだけ
- 手当ての案: 手順 3 の括弧に「Go なら「役割」の implementer の箇条の形」の 1 句を足す(4 つの写しは `scripts/sync-skills.sh`)。skill の文を変えるので、記録だけの修正の例外には当たらない。この回し直しで直さないなら、sync-docs で tech-debt に 1 行記録する

## Observational checks

- 版が混ざったときの道は、cycle 1 の分析のままで変わらない。直しは implementer と leader の雛形、skill、plan の雛形のコマンドの文字だけを変え、reviewer の雛形と `CHECKS:` の扱い、gate の手順には触れていない
- 座席が古い binary の雛形(裸の `go test`)で動いている org では、10 分を越えるパッケージのスライスの検証が打ち切られうる。新しい binary で spawn した座席から直る。release のノートの扱いは計画の Rollout のまま
- 標準フローの `/implement` 手順 6 と `/plan` の手順は「the narrowest check」とだけ書き、Go の形は plan の雛形の HTML コメントにしかない。#261 の作りで、この計画の範囲の外(Info)
- 座席の実機の起動は確かめていない(計画の Test plan の Integration tests と同じ)

## Coverage gaps

- テストの実行(雛形のテスト、`*_sync_test.go`、`tests/test-slice-verification.sh`、`go test ./internal/org/` の全体)と、変異が落ちることの確かめは `/test` に残す。F9 の 2 本と F10 の 12 個の assert は、コンパイルと構文までしか見ていない
- `-timeout=30m` が `go test` の残り時間として効くことは、Go のソースと `go env` の展開で読み、`go test` では確かめていない。計画 `:202` の consult の probe(残り時間 30m、呼び手の 45m と 50m が勝つ)を根拠に残す
- 雛形の文に座席がそのとおり従うこと(leader が新しい形を書くこと、implementer が入れ子の module で `cd` すること)の実機の記録はない

## Verdict

- Verified: AC1〜AC5(AC5 は S5 の範囲)、issue の受け入れ条件 1〜3、cross-review 2 回目の指摘 2 件の解消(20 個のコマンドが 1 つの形、裸の形の残りが 0 件、写しの一致、理由の文の一致)、timeout と入れ子の module の文が言語パックと Go に合うこと、self-review の R3-3 の解消((b) の言い方を除く)、gate の TASK と reviewer の雛形が変わっていないこと、guard が雛形のコマンドを止めないこと、計画の digest、`run-static-verify.sh`・`check-sync.sh`・`check-skill-sync.sh` の通過
- Partially verified: AC6 と issue の受け入れ条件 4(同期の検査、`go vet`、shell の構文は通った。テストは `/test`)
- Not verified: 座席の実機の振る舞い、`go test` の上での timeout の効き方、版が混ざったときの道
- Result: pass。所見は LOW 1(`/org` skill の route (A) の leader の手順 3 に Go の形がない。直すか tech-debt に記録するかは lead が決める)と、既知の渡し 1(tech-debt の R2-5 の行に (e) R3-1 と (f) R3-2。(g) は足さない)。Info は、仕様に timeout を足さなくてよいこと、雛形と言語パックの小さな違い 2 つ、R3-3 (b) の言い方、計画 `:202` の文言、plan の雛形の "the caller"
