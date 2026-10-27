# Verify report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(AC1〜AC6、D1〜D6、Verify plan、Progress checklist の S1〜S5、直し F1〜F8)。issue #262 の受け入れ条件 4 つ
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2 / 2。cross-review の ACTION_REQUIRED を直したあとの回し直し)
- Scope: `git diff 622cdcfb...HEAD`(HEAD 844655c1、24 ファイル、+2440/-105)。cycle 1 の verify(584f8103)のあとに入ったのは、直し F4 ffafc9d2・F5 a7d879d9・F6 854be9db と 51cf59e2・F8 28d83b96、sync-docs 02afccd2、記録のコミット。`git diff 584f8103..HEAD` は 16 ファイル、+838/-88。Go のコードの変更は、今回もテストのファイル 3 つだけ。cycle 2 の self-review(bd21c5fd)は F8 の前の HEAD 2f884ae1 を読んだもので、計画の Progress(`:197`)は F8 の diff の確認を verify と cross-review に任せている。そのため F8 の diff は全部読んだ。テストは打っていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-27-org-narrow-verification.log`(gitignore。`run-static-verify.sh` の全出力、`check-sync.sh` と `check-skill-sync.sh`、digest、写しの `cmp`、gate の節の base と HEAD の `cmp`、語の数え上げ、古い言い方の grep、`gofmt` と `go vet`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 leader の TASK の例に `VERIFY:` の行があり、機能ごとの org の手順 1 にスライスごとの検証コマンドと `RALPH_VERIFY_SCOPE=changed` の注意がある | Met | implementer への TASK の例は `internal/org/prompts/leader.md:484-495`(本文の `VERIFY: test -z "$(gofmt -l internal/foo)" && …` は `:494`)。`VERIFY:` を本文の行にする決まりは `:470-472`(D4)。手順 1 は `:140-146`(「— verify: `<コマンド>`」、いちばん狭い検査、`RALPH_VERIFY_SCOPE=changed` は Go では `go test ./...` を全部走らせるので絞り込みに使わない)。ミッションの 3(`:28-39`)は、どの org でも `VERIFY:` の行を書くよう求める。cycle 1 から変わっていない |
| AC2 implementer の RESULT の契約に 5 項目の結果と検証コマンドの結果があり、既定が `run-verify.sh` ではなく、いちばん狭い検査 | Met | 既定は `internal/org/prompts/implementer.md:32-45`(`VERIFY:` の行がなければ、触ったパッケージの `test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && go test ./<dir>/`。`run-verify.sh` は狭い検査でスライスのファイルを覆えないときだけ)。5 項目の確かめは `:61-80`(stage した差分を読み、この順に見る)。契約は `:156-170`、例は `:112-122`(`VERIFY: … → pass` と `SELF-CHECK:` の 5 項目)。F4・F5 は `:61-80` の秘密情報の扱いを書き換えたが、5 項目の名前と順は変わっていない |
| AC3 機能ごとの org のレビューの TASK は `run-test.sh` を打たない。指示があればそれだけ、なければ両方。gate の TASK は今のまま | Met | leader の手順 8(`leader.md:247-266`)が、レビューの TASK の本文に `CHECKS: ./scripts/run-static-verify.sh` の 1 行だけを書く。reviewer のミッションの 1(`internal/org/prompts/reviewer.md:17-29`)は、指示を本文の `CHECKS:` の行だけから取り、指示があれば足しも削りもせず、行がなければ両方を打つ(D6)。機能ごとでない org の leader はコマンドを書かない(`leader.md:61-64`)。`reviewer.md` は 584f8103 から変わっていない。gate の TASK: reviewer の「## gate の TASK」の節(67 行)、leader の手順 9〜12(120 行)と 9〜16(170 行)、skill の reviewer の gate の TASK の箇条(41 行)は、base 622cdcfb と HEAD で `cmp` が一致した。skill の「org の終わり方」の手順から「merge してよい順」までも一致した(続く「座席内 fan-out」の例は S4 で変えた分)。テストでない `*.go` の変更は 0 件で、`internal/org/gate*.go`、`regate*.go`、`internal/cli/` の差分は 0 行 |
| AC4 `/org` skill の「役割」と「座席内 fan-out」が S1〜S3 と食い違わず、4 つの写しが一致する | Met | 「役割」の implementer は `.claude/skills/org/SKILL.md:1324-1347`、reviewer は `:1348-1370`、fan-out の例は `:1312-1316`。cycle 2 で足した文(前からあった変更の扱い `:1336-1339`、`RESUME:` の引き継ぎ `:1339-1343`、leader 運用の手順 3 の再開 `:1458-1466`)は雛形と同じ語で書かれている(下の「4 つの面の突き合わせ」)。`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` は root と `cmp` で同一 |
| AC5 quality-gates・recipe・仕様の reviewer の記述が S2 と食い違わない | Met(S5 の範囲) | `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md`、`docs/recipes/codex-seat-permissions.md` と写し、`docs/specs/2026-08-01-org-runtime.md` は 584f8103 から変わっていない(`git diff --quiet` が rc 0)。cycle 1 で LOW とした `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 には、sync-docs 02afccd2 が「2026-10-27 に決めたこと」と「印(2026-10-27)」を足した(`:127-128`)。中身はレビューの `CHECKS:` の行、gate の 1 回の test、D6 の既定で、雛形と合う。印が挙げるテスト 5 本はどれも `internal/org/prompts_verification_test.go` にある |
| AC6 雛形のテストと skill の同期の検査が通る | Partially(静的の半分) | `check-skill-sync.sh`(14 skill が一致)と `check-sync.sh`(DRIFTED 0)は通った。`go vet ./internal/org/` はテストのファイルを含めて rc 0 で、テストのファイル 3 つ(`prompts_verification_test.go`、`…_multi_test.go`、`…_predate_test.go`)はコンパイルできる。テストの実行は `/test` に残す。足したテストは AC の文が名指す `prompts_test.go` ではなく、この 3 つのファイルにある(F2・F3・F5 で Progress に記録済み)。`prompts_test.go` は base と同一 |

issue #262 の受け入れ条件:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 1. leader がスライスごとの検証コマンドを計画と TASK に書く | Met | AC1 と同じ |
| 2. implementer の RESULT に機械的な確かめと絞った検証の結果がある | Met | AC2 と同じ |
| 3. reviewer のレビューの TASK が `run-test.sh` を打たない | Met(機能ごとの org) | AC3 と同じ。D6 のとおり、指示のないレビュー(機能ごとでない org、古い leader)では両方を打つ |
| 4. 雛形のテストと同期の検査が通る | Partially | AC6 と同じ。テストは `/test` |

cross-review の指摘(triage の ACTION_REQUIRED #1)の解消:

- 解消した。指摘は、implementer の秘密情報の片付けが、座席が始める前から stage されていた scope の外の変更まで unstage し、中身を消すかファイルを消しうる、というもの。HEAD の `implementer.md` では次のとおりになっている
  - 「前からあった変更」を TASK ごとの記録で 1 度だけ定義する(`:15-18`。stage 済みのものも含む)
  - 直すのはこのスライスの変更だけで、前からあった変更は stage 済みのものも含めて、書き換えも削除も unstage もしない(`:66-68`)
  - unstage はパスを付けた `git restore --staged -- <このスライスのパス>...` で行い、パスのない `git reset` は打たない(`:68-70`)
  - コミットしないのは 3 つの場合で、どれも BLOCKED を返す。(1) このスライスの変更にある秘密情報を scope の中で消せない、(2) 前からあった変更に秘密情報がある、(3) stage 済みの前からあった変更に scope の外のパスがある(`:70-74`)
  - ファイルを消すのは (1) で、そのファイルが「このスライスで作った未追跡のファイル」のときだけ。(2) と (3) では何も触らない(`:74-77`)
- triage が挙げた例(scope の外に前から stage された認証情報のファイル)は (2) か (3) に当たり、座席はそれに触らずに BLOCKED で止まる
- 古い言い方(「例外は秘密情報だけ」「新しい未追跡のファイルなら」「秘密情報を scope の中で消せなければ」「座席が始める前」「既存の変更」)は、雛形、skill の 4 つの写し、`docs/specs/`、`docs/recipes/`、`docs/quality/` のどこにも残っていない。テストの中に、それが戻らないことを確かめる照合として残るだけ(`prompts_verification_predate_test.go:47`、`:108`、`:139`)
- skill の implementer の箇条(`SKILL.md:1336-1339`)も同じ境界を書く。前からあった変更の秘密情報と、その中で stage 済みの scope の外のパスには触らず、コミットせずに BLOCKED を返す

4 つの面(雛形 3 つと `/org` skill)の突き合わせ(空白を詰めて数えた。証拠のログの「phrase counts」):

| 語 | implementer | leader | reviewer | skill | 読み |
| --- | --- | --- | --- | --- | --- |
| `VERIFY: S1: <コマンド>`(複数スライス) | 2 | 1 | 0 | 3 | leader が書き(`leader.md:38`)、implementer が受け取りと RESULT の形で読む(`implementer.md:35`、`:168`)。skill は手順 3、「役割」、「org の終わり方」に 1 回ずつ書く。reviewer は扱わない |
| `VERIFY: <コマンド>`(1 スライス) | 0 | 1 | 0 | 0 | leader だけが形を書き(`:39`)、ほかは例(`implementer.md:118`、`leader.md:494`)と散文で示す。食い違いはない |
| `CHECKS: ./scripts/run-static-verify.sh` | 0 | 1 | 5 | 3 | leader の手順 8(`:250`)が書き、reviewer が本文の行として読み(`reviewer.md:21-24`)、同じ形で返す(`:54-57`、例 `:160`、`:173`)。skill は「org の終わり方」の冒頭、「役割」の reviewer、RESULT の例(`:1505`)。cycle 1 から変わっていない |
| `RESUME: S1: <パス>...`(複数スライス) | 1 | 1 | 0 | 1 | F8 で 3 つの面にそろった(`implementer.md:24`、`leader.md:47`、`SKILL.md:1462`)。implementer はその名前のスライスのコミットに入れる(`:24-26`)。skill の「役割」は散文で同じことを書く(`:1341-1343`) |
| `RESUME: <パス>...`(1 スライス) | 0 | 1 | 0 | 0 | leader が形を書き(`:47-48`)、例は `:44`。implementer と skill は「`RESUME:` の行に挙がったパス」と散文で書く |
| 「前からあった変更」 | 11 | 2 | 0 | 3 | 定義は implementer の 1 か所(`:15-18`)。leader の説明(`:56-59`。止まった TASK の始めの記録に出ていて、その TASK の `RESUME:` の行になかった変更)はこの定義と同じ集合を指す。skill の説明(`:1336-1338`。TASK ごとに、最初の編集の前の記録に出ていた変更)も同じ |
| コミットしない 3 つの場合 | 3 つを並べる | — | — | 3 つを 1 文で書く | implementer は `:70-74`、skill は `:1335-1339`(「scope の中で消せない秘密情報」「前からあった変更の秘密情報」「その中で stage 済みの scope の外のパス」)。leader は秘密情報の 2 つについて、再開か escalate かを決める(`leader.md:53-60`) |

self-review の cycle 2 の 2 回目(bd21c5fd、HEAD 2f884ae1 を読んだ)の所見を、F8 28d83b96 に照らして HEAD で読んだ:

- R2-1(複数スライスの `RESUME:` に名前がない): 解消。上の表の行のとおり、3 つの面が `RESUME: S1: <パス>...` を書く。テストは `resumeLineOfOneOfSeveralSlices`(`prompts_verification_predate_test.go:149-152`)で 3 つの面を照合する
- R2-2(skill の leader 手順 3 に escalate と狭めるときの文がない): 解消。`SKILL.md:1462-1466` に、狭めるときも残したパスをすべて並べて元に戻すパスを書く文と、値が消えていないとき、または前からあった変更の秘密情報のときに escalate する文がある
- R2-3(leader の「前からあった変更」の説明が implementer の定義と違う): 解消。`leader.md:56-59` の括弧は implementer の定義と同じ集合を言う。古い括弧(「止まった TASK より前の変更」)は消え、テストがそれの戻りを照合する
- R2-4(「残したパス」が 2 通りに読める、折り返し): leader の側は解消。`leader.md:50-53` は「止まった TASK がコミットせずに残したパスは、狭めたスライスから外すものも含めてすべて」と書き、折り返しも直った。R2-4 が添えた `implementer.md` の短い行(F6 の挿入による)は `:160`(「パスを書きます。検証結果には、打った」)に残る。Markdown の段落としては同じ文に読めるので、意味は変わらない(Info)
- R2-5(テストの整理)と、1 回目の所見 4(標準フローの implementer の片付け): 計画の決定(`:192`、`:197`)のとおり、sync-docs で tech-debt に記録する。HEAD の `docs/tech-debt/README.md` にはまだない。self-review の報告の「案 B」の行は R2-1〜R2-5 を並べているが、R2-1〜R2-4 は HEAD で解消しているので、台帳に足すのは R2-5 だけでよい(下の Documentation drift)

計画の digest: `./scripts/plan-visual.sh digest` が `97abe345a188` を返し、`- Approved: 2026-10-27 sha256:97abe345a188` と一致する。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | changed scope、言語は golang だけ。golang の pack は `gofmt: ok`、`go vet` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし。local verifier の shellcheck、hook の `sh -n`(root と template の 10 本ずつ)、guard の awk の構文、settings の `jq`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照はすべて OK。secret scan は 622cdcfb..844655c1 で clean。wrapper 自身のログは `docs/evidence/verify-2026-10-27-195522.log` |
| `./scripts/check-skill-sync.sh` | rc 0 | 14 skill が一致 |
| `./scripts/check-sync.sh` | rc 0、PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `test -z "$(gofmt -l internal/org)"` と `go vet ./internal/org/` | rc 0 | テストのファイル 3 つを含む。go1.27.3(go.mod と同じ) |
| `git diff --check 622cdcfb...HEAD` | rc 0 | 出力なし |
| 変更したファイルの U+FFFD | 0 件 | `grep -l` の rc 1 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/specs/2026-08-01-org-runtime.md` の「2026-10-27 改訂」(a)、`docs/specs/2026-10-07-org-multi-org-director.md` の FR-7「2026-10-27 に決めたこと」 | 一部(LOW、`/sync-docs` へ) | 2 つの仕様は、本文の行 `VERIFY:` と `CHECKS:` を契約として書いている。cycle 2 の直しで増えた 2 つは書いていない。1 つは leader から implementer への本文の行 `RESUME:`(止まったスライスの再開。複数スライスでは名前付き)、もう 1 つは implementer の片付けの境界(このスライスの変更だけを直し、前からあった変更には触らず、コミットしない 3 つの場合は BLOCKED)である。triage は影響するファイルに改訂 (a) を挙げたが、F4〜F8 は仕様を触っていない。改訂 (a) は前から秘密情報の片付けを書いていなかったので、偽になった文はない。そのため食い違いではなく欠けとして LOW にした。足すなら改訂 (a) に 1〜2 文(`RESUME:` の行と片付けの境界)。FR-7 の印は `[ ]` のままでよい |
| `docs/tech-debt/README.md`(R2-5 と、1 回目の所見 4 の行) | 未記録(既知、`/sync-docs` へ) | 計画の Progress(`:192`、`:197`)が sync-docs に渡した 2 行は、cycle 2 の sync-docs がまだ走っていないので HEAD にない。self-review の報告の「案 A」(標準フローの implementer の片付け)はそのまま使える。「案 B」は R2-1〜R2-5 を並べるが、R2-1〜R2-4 は F8 で解消したので、R2-5(同じ文を 2 か所で固定、`predate` のファイルに再開のテスト、順序の照合が 2 通り、テスト名の語)に絞る |
| `docs/tech-debt/README.md:157`(FR-7 の gate が雛形の指示による、の行) | Yes | cycle 1 で既知とした古い文(「The reviewer seat re-runs `run-static-verify.sh` / `run-test.sh`」)は、sync-docs 02afccd2 が「runs the commands its review TASK names」に直し、2026-10-27 の繰り越しの注記を足した。列の数は 5 のまま。cycle 2 の直しは座席が打つコマンドを変えていないので、この注記は今も正しい |
| `docs/quality/quality-gates.md:82`、`templates/base/docs/quality/quality-gates.md:81`、`docs/recipes/codex-seat-permissions.md:220-223` と写し | Yes | 584f8103 から変わっていない。quality-gates の「impl exit checks (the slice's verification command from the leader's TASK and a mechanical self-check on the staged diff)」は、秘密情報の片付けを書いていないので、F4〜F8 と食い違わない。recipe の写しは `cmp` で同一 |
| `.claude/rules/ralph/agent-messaging.md` | Yes | ralph が検証するヘッダの契約だけを書く rule で、`VERIFY:`・`CHECKS:` と同じく、本文の行 `RESUME:` も書かない(cycle 1 の sync-docs の判断と同じ)。`RESUME:` は D4 のとおり本文の行で(`leader.md:44-45`)、ralph は読まない |
| `RESUME:` の行が出る所 | Yes(上の仕様の行を除く) | 記録(`docs/reports`、`docs/plans/archive`、`docs/insights`、`docs/evidence`)の外では、雛形 2 つ、skill の 4 つの写し、計画、テストだけ |
| README の `ralph org gate` の段落と `/org` skill の gate の節 | Yes | gate の節は base と同一(AC3 の行)。cycle 2 の直しは gate に触れていない |
| 計画の記録 | 情報 | Progress の F4〜F8 の記録は HEAD の差分と合う(コミットの SHA、触ったファイル)。cycle 1 で情報とした `[ ] Review artifact created` の重なりは消えている。残る `[ ]` は `PR created` だけ。digest は一致 |

## Observational checks

- 版が混ざったときの道は cycle 1 の分析のままで、cycle 2 の直しで変わらない。直しは implementer と leader の雛形だけを変え、reviewer の雛形と `CHECKS:` の扱いには触れていない
- 片付けの境界を、雛形の文で読んで確かめた(動かしていない)
  - scope の外の前からあった変更は、始めに触らずに残す(`implementer.md:29-30`)
  - それが stage 済みなら、コミットの時点で 3 つ目の場合として BLOCKED になる
  - scope と重なる前からあった変更は、実装を始める前に BLOCKED になる(`:30-31`)
  - このため `RESUME:` に並ぶ scope の中のパスは、止まった TASK(またはそれより前に止まって並べられた TASK)が残したものに限られる。ほかの作業の変更をこのスライスの変更として引き継ぐ道は、雛形の上ではない
- 3 つ目の場合(前から stage されていた scope の外のパス)で止まったあと、leader の再開の段落(`leader.md:40-41`)は「止まった原因を取り除いて」と書くが、その stage を誰が外すかは書いていない。leader はコードを直さないので、多くは escalate に倒れる。安全側に止まる道で、AC の範囲の外なので所見にしていない
- 座席の実機の起動は確かめていない(計画の Test plan の Integration tests と同じ)

## Coverage gaps

- テストの実行(雛形のテスト、`*_sync_test.go`、`go test ./internal/org/` の全体)と、変異が落ちることの確かめは `/test` に残す。cycle 2 で足した `prompts_verification_predate_test.go` の 6 本はコンパイルまでしか見ていない
- 雛形の文に座席がそのとおり従うことは、文の確かめでしか見ていない。leader が `VERIFY:`・`CHECKS:`・`RESUME:` を書くこと、implementer が前からあった変更に触らずに BLOCKED で止まることの実機の記録はない

## Verdict

- Verified: AC1〜AC5(AC5 は S5 の範囲)、issue の受け入れ条件 1〜3、cross-review の指摘の解消、4 つの面の `VERIFY:`・`CHECKS:`・`RESUME:`(1 スライスと複数スライスの形)・「前からあった変更」・コミットしない 3 つの場合の一致、skill の 4 つの写しの一致、gate の TASK の節と `ralph org gate`/`regate` が変わっていないこと、self-review の R2-1〜R2-4 の解消(R2-4 は `implementer.md:160` の短い行を除く)、計画の digest、`run-static-verify.sh`・`check-sync.sh`・`check-skill-sync.sh` の通過
- Partially verified: AC6 と issue の受け入れ条件 4(同期の検査と `go vet` は通った。テストは `/test`)
- Not verified: 座席の実機の振る舞い、版が混ざったときの道
- Result: pass。所見は LOW 1(仕様 2 つに `RESUME:` の行と片付けの境界がない)と、既知の渡し 1(tech-debt に R2-5 と 1 回目の所見 4 の 2 行。案 B は R2-5 に絞る)。どちらも cycle 2 の `/sync-docs` に渡す。Info 1(`implementer.md:160` の短い行)
