# regate-ci-checks

- Status: Approved
- Approved: 2026-10-30 sha256:994880bfa86d
- Owner: Claude Code
- Date: 2026-10-29
- Related request: org: regate の手元の test を CI の結果に置き換えられるようにする(https://github.com/yoshpy-dev/ralph/issues/264)
- Related issue: 264
- Type: feat
- Branch: feat/regate-ci-checks

## Objective

`ralph org regate --branch <ブランチ>` の検査の仕方を設定で選べるようにする。既定の `local` は今と同じく gate のスクリプトを手元で走らせる。`ci` では、取り込み(origin の既定のブランチの merge commit)と push だけを行い、push した head の PR の checks の結果で gate の記録を書く。regate のたびに CI と同じフルの test を手元でもう 1 回待たなくて済むようにする。

## Scope

- 設定 `[org.gate].regate_checks = "local" | "ci"`(既定 `local`)。`templates/base/ralph.toml` に既定とコメント、この repo のルートの `ralph.toml` に `ci`
- `ci` の regate: 事前の確かめ、merge、push、PR の checks の読み取り、gate の記録
- `ralph org regate --wait-checks <時間>`(`ralph org merge --wait-checks` と同じ上限と検証)
- `ci` で書いた記録を merge-order と merge が読めること、失敗の記録の文言
- 文書: `/director` skill の「merge のあとの取り直し」、`/org` skill の regate の行と手順、README、仕様 FR-7 の注、AGENTS.md の地図、`ralph org regate --help`、それぞれの写しと固定しているテスト

## Non-goals

- PR の前の gate(reviewer 座席が打つ `ralph org gate`)を CI に変えること。2026-10-27 にユーザーに示し、選ばれなかった
- CI が落ちたときに、push した merge commit を戻すこと(force push はしない)
- 必須の checks だけを見る読み方(`gh pr checks --required`)。merge と同じ判定を使う
- `ralph doctor` の新しい警告(`ci` なのに保護ブランチが checks を強制していない、など)
- `ralph org merge` の待ち方や判定の変更
- push されていないコミットを regate が代わりに push すること(今と同じく断る)

## Assumptions

- 実装は #263 の PR(`internal/org/mergeorigin.go` の `ReadMergeOrigin` を足し、merge の `readOrigin` をそれに載せた)が merge されてから始め、最初に origin/main をこのブランチに merge する。`ci` の regate は、merge と同じ読み方で origin の host と owner/repo を読み、gh を `-R` で固定する
- regate を打つとき、機能のブランチには既定のブランチへの open な PR がある(leader が PR を作ってから disband する、`/org` skill の「org の終わり方」)
- このリポジトリの CI(`.github/workflows/verify.yml`)は PR への push で `./scripts/run-verify.sh` を走らせる
- `gh pr checks` は PR の今の head の checks を返す。push の直後は PR の head が古いままのことがある(GitHub の更新の遅れ。未確認)ので、PR の `headRefOid` が push した SHA と一致するまでは結果を使わない

## Affected areas

- 設定: `internal/config/config.go`(`OrgGateConfig`、既定、`Load` の検証)、`internal/config/config_test.go`、`internal/config/defaults_sync_test.go`(`orgTemplateProbe` と `orgTemplateDrift`)、`internal/config/root_ralph_toml_test.go`、`templates/base/ralph.toml`、`ralph.toml`、`scripts/check-sync.sh`(`KNOWN_DIFFS` の `ralph.toml` のコメント)
- regate: `internal/org/regate.go`(`RegateParams`、`Regate` の流れ、gate だけを取り直す道)、`internal/org/regate_merge.go`(push と記録の順、push の失敗、`regateResetHint` と PushOnly の via の分かれ)、`internal/org/regate_git.go`、新しい `internal/org/regate_ci.go`(事前の確かめ、checks の読み取りと待ち、記録)
- 記録と読み手: `internal/org/gate_record.go`(reason `ci_failed`、via `regate_ci`)、`internal/org/seat.go`(event `gate_ci_passed`・`gate_ci_failed` と注釈)、`internal/org/watch.go`(gate の event を数える箇所の注釈。数え方は変えない)、`internal/org/mergeorder.go`(`mergeRegateFailedDetail` の文言)、`internal/cli/org_gate.go`(reason ごとの表示)
- checks の読み取りの共用: `internal/org/merge.go`(`readChecks`、`viewPR`)、`internal/org/merge_wait.go`(待ちの上限、間隔、検証の `MergeWaitChecksRefusal`)
- CLI: `internal/cli/org_regate.go`(`--wait-checks`、設定の読み、出力の行、help)
- 文書: `.claude/skills/director/SKILL.md` と `.claude/skills/org/SKILL.md` とそれぞれの写し 3 つ、`README.md`、`AGENTS.md`、`docs/specs/2026-10-07-org-multi-org-director.md`(FR-7)
- テスト: `internal/org/regate_test.go`、`internal/org/regate_merge_test.go`、新しい `internal/org/regate_ci_test.go`、`internal/org/mergeorder_test.go`、`internal/org/gate_test.go`、`internal/cli/org_regate_test.go`、`internal/org/director_skill_sync_test.go`、`internal/org/org_skill_send_sync_test.go`

## Visual review

- Page: `.harness/state/plan-visual/regate-ci-checks.html`(やさしい図 `overview`・`easy-result`、折りたたみの中に `detail-overview`・`detail-sequence`・`detail-checks`・`detail-wait`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: full、overview(`--scale 2`)、ほかの図を 1 枚ずつ撮って読んだ。直したのは、h1 の折り返し、`detail-overview` の矢印が面の下に隠れていたこと、`detail-sequence` の丸数字の字形と行のはみ出し、`detail-wait` のつながらない箱。計画の直し(新しい event、D3 の 3 つの分かれ、D4 の 3 行)のあとに `detail-overview`・`detail-sequence`・`detail-checks` を撮り直して読んだ

## Design decisions

### D1 設定

`[org.gate].regate_checks` に `local`(既定)か `ci` を書く。ほかの値は `config.Load` が断る(`[org.approval].mode` と同じ形)。`ralph config get org.gate.regate_checks` で読める。regate の CLI は今と同じく、打った場所の `ralph.toml` か `--config` を読む(`newOrgRuntime`)。この repo のルートの `ralph.toml` には `[org.gate]` の表だけを足す(template との差分だけを書く約束)。

### D2 `ci` の regate の順

1. 事前の確かめ(merge の前): origin の URL が GitHub の repository を名指す(`ReadMergeOrigin`)、gh が PATH にある、ブランチの PR が merge と同じ読み方(`findPR`: `-R <origin>` に固定し、open で base が既定のブランチ)で見つかる。merge と同じ読み方にするのは、merge-order と merge が読む PR と食い違わないため。どれかが通らなければ、何もせずに断る(exit 1)
2. 今と同じ fetch、HEAD と origin の確かめ、merge commit の作成、identity と in-the-way の確かめ
3. gate のスクリプトは走らせず、push する(force なし)
4. push した head の PR の checks を読み、D4 のとおりに記録する

今の `local` は merge → スクリプト → 記録 → push の順で、記録があるので push だけの打ち直し(PushOnly)ができる。`ci` は記録が push のあとになるので、push の失敗は D3 で扱う。

### D3 push の失敗

`ci` で push が失敗したら、origin のブランチの先頭を読み返し、3 つに分ける。

- 読み返した先頭が push の前と同じ(merge した HEAD の親、origin の元の先頭): push は届いていない。merge commit を今の undo で外し(何も公開されず、何も記録しない)、理由と「直してから regate を打ち直す」を出す
- 読み返した先頭が push した SHA: push は届いたが応答が失われた。checks に進む
- それ以外(先頭が push した SHA の子孫になっている、別のコミットになっている、読み返せない): 公開されたかどうかを ralph が言い切れないので、HEAD を残し、何も記録せずに断る(exit 1)。理由には読み返した先頭(または読み返せなかったこと)を出し、次の手は `git fetch origin` と `git log` で origin と HEAD を確かめてから打ち直すこと。merge commit を外すのは、公開されていないと確かめられたときだけにする

### D4 checks の読み方と記録

merge の `readChecks` と同じ判定を使う(pass と skipping 以外が 1 つでもあれば通らない、pass が 1 つ以上要る、0 件、読めない)。先に `gh pr view --json headRefOid` で PR の head が push した SHA かを確かめ、違ううちは pending と同じに扱う。

- すべて通った: 新しい event `gate_ci_passed`、`via=regate_ci`、head は push した SHA、base と base_ref は取り込んだ既定のブランチの先頭
- 1 つでも落ちて、pending がない: 新しい event `gate_ci_failed`、reason `ci_failed`、`via=regate_ci`。merge commit は origin に残る(force しない)。次の手は「ブランチで直して push し、regate を打ち直す」
- 記録の Details の形は今の gate の記録と同じ(`gateRecord.details`)で、event の名前だけを分ける。今の binary は event の名前で gate の記録を選ぶ(`mergeorder.go:458`、`watch.go:1401`)ので、この PR を revert して古い binary に戻すと、CI の記録は読み飛ばされる。古い binary の regate が `regate_ci` の passed を普通の passed と読み、PushOnly で押し直すことはない(plan advisory の指摘 1)
- pending、0 件、PR の head がまだ古い: 記録しない。`CHECKS: pending (<理由>)` の行を出し、断りや失敗(exit 1)と区別できる専用の終了コード 6 で終わる(`internal/cli/exitcode.go` の一覧に足す)。次の手は、しばらくして regate を打ち直す(D6 の道で続く)か `--wait-checks` を付けること。CI は 1 回に 10 分を越えるので、`--wait-checks 0` では push の直後にほぼ必ずこの形になる。`/director` skill の「exit 1 は止めて人に伝え、頼まれずに打ち直さない」に当たらないよう、終了コードを分ける
- checks を読めない(gh の失敗、時間切れ): 記録しない。exit 1 で理由を出す(merge の Unread と同じ扱い)
- 落ちた check と pending の check が両方あるとき: pending と同じに扱う(merge と同じく、すべて終わるまで結果を出さない)
- すべて skipping で pass が 1 つもないとき: 記録しない。exit 1 で「通った check が 1 つもない」と出す。コードの失敗ではないので `ci_failed` の記録にはしない
- `--wait-checks` の上限まで待っても終わらないとき: 記録しない。pending と同じく `CHECKS: pending` の行と終了コード 6
- `log=` は gate の log のファイルで、中身は読んだ checks の JSON(`createGateLog`)

### D5 待ち

`ralph org regate --wait-checks <時間>` は merge と同じ検証(`MergeWaitChecksRefusal`、上限 6 時間)と間隔(30 秒)で、checks が終わるまで待つ。既定 0 は 1 回だけ読む(merge の既定と同じ)。待つあいだは merge と同じく `regate.lock` を外し、取り直してから記録の前の確かめ(結びつき、org が止まっている、HEAD と origin の先頭が push した SHA)をやり直す。CI を何十分も待つあいだ、ほかのブランチの merge と regate を止めないため。

### D6 gate だけを取り直す道

HEAD が origin の先頭で、既定のブランチの先頭を含み、その head の passed の記録がないとき、今の `local` はスクリプトだけを走らせる(push なし)。`ci` ではここで PR の checks を読み、D4 のとおりに記録する。pending で終わったあとの打ち直しと、CI が落ちて人が直しを push したあとの打ち直しは、この道を通る。

### D7 読み手

gate の記録を選ぶ箇所(merge-order の収集 `mergeorder.go:458`、regate が結びつきの最新の記録を読むところ)に `gate_ci_passed` と `gate_ci_failed` を足し、passed と failed として今の記録と同じに読む。merge-order の `ready` の判定(`classifyGatePassed`)と merge は via を見ないので、そこから先は変えない。via を見る 3 か所を直す: merge-order の失敗の文言(`mergeRegateFailedDetail`、`ci_failed` 用の次の手)、regate の `regateResetHint`(`regate_ci` は対象外のまま。push 済みの merge commit の reset を勧めない)、PushOnly の分かれ(`regate_merge.go:76-91`。今は via が `regate` でない passed の記録を、origin の先頭が HEAD の祖先なら push だけの打ち直しにする。`regate_ci` の passed は push が届いたあとに書くので、そのあと HEAD が origin の先頭でないのは、だれかが origin を戻したか force push したときで、押し直すとその人の作業を戻す。そこで `regate_ci` の passed は PushOnly に入れず、理由を出して断る)。reason `ci_failed` は `isGateReason` と `ralph org gate` の表示に足す(足さないと記録が unreadable になる)。merge は merge の直前に checks を自分で読み直すので、記録のあとに checks が変わっても merge はそれを見る。

### Critical forks

3 つあり、2026-10-27 のユーザーの方針(分かれ目は consult に聞いて自分で選び理由を書く)で決めた。

1. `ci` で gh がない、PR がない、origin が GitHub でないとき: **断る**(D2 の 1)。手元の gate に戻す案は、設定の意図を黙って変え、`ci` を選んだ repo で手元のフルの test が突然走るので採らない。merge も同じ場面で断る
2. 待ちの既定: **merge と同じ `--wait-checks`、既定 0(1 回だけ読む)**(D5)。regate に独自の既定の待ちを持たせる案は、merge と食い違い、CLI の時間切れ(Bash の上限など)にも当たりやすい。pending のあとは打ち直せば D6 の道で続く
3. 待つあいだの `regate.lock`: **merge と同じく外し、取り直してから確かめ直す**(D5)。持ち続ける案は、CI のあいだ同じ台帳のすべての merge・regate・cleanup を止める

## Acceptance criteria

- [ ] AC1 既定(`local`)では、今の regate と同じ動きをする(今の regate のテストが変えずに通る)
- [ ] AC2 `ci` では、regate は gate のスクリプトを走らせず、merge と push のあと、push した head の PR の checks の結果で記録を書く(passed は event `gate_ci_passed`、`via=regate_ci`、head は push した SHA、base は取り込んだ既定のブランチの先頭)。今の event 名(`gate_passed`・`gate_failed`)だけを選ぶ読み手は、この記録を読まない
- [ ] AC3 `ci` で書いた passed の記録で、merge-order が `ready` を出し、`ralph org merge` が merge できる
- [ ] AC4 CI が落ちたときは `gate_ci_failed`(reason `ci_failed`、`via=regate_ci`)の記録になり、merge-order は `ready` を出さず、次の手を出す
- [ ] AC5 checks が終わっていない(pending、0 件、PR の head がまだ古い)ときは何も記録せず、`CHECKS: pending` の行と終了コード 6 で次の手を出す。checks を読めないときは何も記録せず exit 1。打ち直すと D6 の道で checks を読み直し、`--wait-checks` では上限まで `regate.lock` を外して待つ
- [ ] AC10 `ci` の passed の記録があり HEAD が origin の先頭でないとき、regate は push だけの打ち直しをせずに断る
- [ ] AC6 `ci` で origin が GitHub の repository を名指さない、gh が PATH にない、既定のブランチへの open な PR がないときは、merge の前に何もせずに断る
- [ ] AC7 `ci` で push が失敗したとき、読み返した origin の先頭が push の前と同じなら merge commit を外して何も記録しない。push した SHA なら checks に進む。それ以外(子孫、別のコミット、読み返せない)なら HEAD を残し、何も記録せずに断って確かめ方を出す
- [ ] AC8 この repo のルートの `ralph.toml` が `[org.gate].regate_checks = "ci"` を持ち、template は `local` とコメントを持つ。`ralph config get org.gate.regate_checks` が値を返し、ほかの値は `config.Load` が断る。設定のずれの検査がこのキーを見る
- [ ] AC9 `/director` skill の「merge のあとの取り直し」、`/org` skill の regate の行と手順、README、仕様 FR-7 の注、AGENTS.md、`ralph org regate --help` が `ci` の動きを書き、写しの検査と文を固定するテストが通る

## Implementation outline

1. S1 設定: `OrgGateConfig` と `regate_checks`、既定、`Load` の検証、template、ルートの `ralph.toml`、ずれの検査、`check-sync.sh` のコメント。#263 と重ならない(`internal/config` を触らない)ので、#263 の merge を待たずに始める — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ && ./scripts/check-sync.sh`
2. S0 main の取り込み: #263 の PR の merge のあと、origin/main をこのブランチに merge する(`ReadMergeOrigin` を使うため。#263 は `regate.go`・`regate_git.go`・`merge.go`・`mergeorder.go`・`seat.go`・`org_regate.go`・skill を触る)。S2 からはこのあとに始める — verify: `go build ./... && go vet ./internal/org/ ./internal/cli/ ./internal/config/`
3. S2 記録と読み手: event `gate_ci_passed`・`gate_ci_failed`(`seat.go`)と、gate の記録を選ぶ箇所への追加、reason `ci_failed`、via `regate_ci`、merge-order の失敗の文言、`regateResetHint` と PushOnly の分かれ、`ralph org gate` の reason の表示 — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'GateRecord|MergeOrder|Regate' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'OrgGate|OrgMergeOrder'`
4. S3 checks の読み取りの共用: merge の `findPR`(PR の読み方)、`readChecks`、PR の head の読み取りを、regate からも呼べる形にする(merge の動きは変えない) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Merge'`
5. S4 `ci` の regate: 事前の確かめ、push と記録の順、push の失敗の undo、checks から記録、gate だけを取り直す道(D2・D3・D4・D6) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Regate'`
6. S5 待ち: `RegateParams` の待ちの時間、`regate.lock` を外して待ち、取り直して確かめ直す(D5) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Regate|MergeWait'`
7. S6 CLI: `--wait-checks`、設定から `RegateParams` への受け渡し、出力の行(`CHECKS:` の行)、終了コード 6(`internal/cli/exitcode.go`)、help — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'OrgRegate|Help'`
8. S7 文書: skill 2 つとその写し、README、仕様、AGENTS.md、固定しているテスト。`/director` skill の「merge のあとの取り直し」は、`ci` では `--wait-checks` を付けて打つこと、終了コード 6 は断りではなく打ち直してよいこと、「スクリプトが走るので数十分」の文を `ci` の場合に合わせて書き直す — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill|Prompt' && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、vet、staticcheck、golangci-lint)、`check-skill-sync.sh`、`check-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC10。特に AC1(既定で今と同じ)、AC2 の記録の head と base、AC5 の何も記録しない場合、AC6 の merge の前に断ること
- Documentation drift to check: regate を説明する文書(skill 2 つとその写し、README、仕様 FR-7、AGENTS.md の地図、help)と、PR の前の gate の説明が変わっていないこと
- Evidence to capture: verify の report、static verify の出力

## Test plan

- Unit tests: 設定の既定と検証(`local`・`ci`・ほかの値)、`ralph config get`、記録の `ci_failed` と `regate_ci` の書き読み、merge-order の失敗の文言
- Integration tests: regate の fixture(ローカルの bare origin)に、`git remote get-url origin` と gh を偽にした Exec を足して `ci` を打つ。passed、failed、pending、0 件、読めない、PR の head が古い、PR がない、gh がない、origin が GitHub でない、push の失敗(origin が SHA を持つ・持たない)、`--wait-checks` で pending のあと passed(偽の時計)、待つあいだに `regate.lock` が外れていること、取り直しの確かめで結びつきが変わったとき
- Regression tests: 今の regate のテストが `local` で変えずに通る。merge と merge-order のテスト
- Rollback の確かめ(plan advisory の指摘 1): 今の event 名だけを選ぶ読み方(この PR の前の `mergeorder.go:458` と同じ形)が `gate_ci_*` を読まないこと。`gate_ci_passed` があり、origin が意図して古いコミットに戻されたとき、新しい regate は push せずに断ること(AC10)
- push の失敗の 3 つの分かれ(plan advisory の指摘 2): 読み返した先頭が push の前と同じ(merge を外す)、push した SHA(checks に進む)、push した SHA の子孫(HEAD を残して断る)、読み返せない(HEAD を残して断る)
- Edge cases: 待つあいだに既定のブランチが進む(記録の base が古くなり merge-order が `rebase` と言う)、同じブランチの regate が 2 つ(取り直しのあとに passed の記録があれば UpToDate)、割り込み(push のあと記録の前)
- Evidence to capture: test の report、変異(checks の判定を緩める、PR の head の確かめを外す、`ci` でスクリプトを走らせる、push の失敗の undo を外す)が落ちること

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| push の直後、PR の head が古いまま古い head の pass を読む | PR の `headRefOid` が push した SHA のときだけ checks を使う(D4) |
| CI を待つあいだ `regate.lock` を持ち続け、ほかの merge と regate が止まる | merge と同じく待つあいだは外す(D5) |
| CI が落ちても merge commit は origin に残る | force しないので戻さない。`ci_failed` の記録と次の手で直しを促し、merge-order は ready を出さない |
| `via` と reason を足し忘れ、記録が unreadable になるか、via を見る分かれが弱い側に入る | S2 で `isGateReason` と via を見る 3 か所を直し、テストで固定する |
| 新しい event を読み手に足し忘れ、CI の記録が merge-order や regate から見えない | S2 で gate の記録を選ぶ箇所(`mergeorder.go` の収集、regate の結びつきの読み)に足し、merge-order・merge・regate のテストで `gate_ci_*` を読ませる。watch の活動の数え方(`watch.go:1401`)は regate が disband 済みの org にしか書かないので足さない |
| revert して古い binary に戻したあと、古い regate が CI の passed を普通の passed と読み、意図して戻された origin のコミットを押し直す | CI の記録を新しい event 名で書き、古い binary には見えないようにする(D4) |
| push の応答が失われ、そのあと origin が進んでいるのに、公開済みの merge を外す | 外すのは読み返した先頭が push の前と同じときだけ。それ以外は HEAD を残して断る(D3) |
| CI のない repo で `ci` を選ぶと、checks が 0 件のまま通らない | 既定は `local`。0 件は記録せずに次の手を出す。template のコメントに書く |
| #263 と `merge.go`・`mergeorder.go`・skill が重なる | #263 の merge のあとに実装を始め、先に main を取り込む(outline の 0) |
| regate の記録の pass で merge を急ぎ、CI の結果が古い | merge は直前に checks を読み直し、保護ブランチの強制も見る(今のまま) |

## Rollout or rollback notes

- 既定は `local` なので、設定を変えない repo では動きが変わらない。この repo だけが `ci` になる
- 戻すときはこの PR を revert する。台帳に増えるのは `gate_ci_passed` と `gate_ci_failed` の行だけで、古い binary はこの event を知らないので読み飛ばす。merge-order はそのブランチの前の記録で分類し(多くは `regate`)、古い regate は手元のスクリプトで取り直す。`regate_ci` の passed を読んで PushOnly に入ることはない
- release ノートに 1 行: `[org.gate].regate_checks = "ci"` で regate が手元の test の代わりに PR の CI の結果で記録を書けるようになったこと

## Open questions

- push の直後に PR の head と checks が更新されるまでの時間(GitHub の遅れ。未確認)。D4 は PR の head が push した SHA になるまで pending と読むので、遅れは待ちの時間に入るだけで、古い結果を記録することはない

## Progress checklist

- [x] Plan reviewed(consult の判定は「直してから進める」。直したのは、PushOnly で `regate_ci` を断ると書くこと(D7、AC10)、pending を `CHECKS: pending` の行と終了コード 6 で出すこと(D4、S6、S7)、事前の確かめを merge の `findPR` と同じ読み方にすること(D2、S3)、S1 を #263 の merge を待たずに始めること(outline)。plan advisory: ran with codex。HIGH 1 件(revert して古い binary に戻すと、`regate_ci` の passed を普通の passed と読み、意図して戻された origin のコミットを PushOnly で押し直しうる)は、CI の記録を新しい event `gate_ci_passed`・`gate_ci_failed` で書き、古い binary に見えないようにして反映した(D4、D7、AC2、AC4、リスクの表、Rollback、Test plan)。MEDIUM 1 件(push の応答が失われ origin が進んでいるときに、公開済みの merge を外す)は、D3 を読み返した先頭で 3 つに分け、外すのは push の前と同じときだけにして反映した(D3、AC7、リスクの表、Test plan))
- [x] Plan approved(自動承認。2026-10-11 のユーザー指示「作成したプランは自動承認します」と、2026-10-27 の「最後まで止まらずに」による。2026-10-29 のユーザーの問い「#263を待たないといけないの？」を受け、#263 のパイプラインと並べて計画した。S1 は #263 を待たずに始め、S2 からは #263 の merge のあとに main を取り込んでから始める)
- [x] Branch created
- [x] Implementation started(スライスは implementer に委譲)
  - S1 3a55f6d1: `OrgGateConfig`(toml `gate`)と `RegateChecksLocal`・`RegateChecksCI`、`Default()` は local、`Load` は local と ci 以外(空も)を断る。template の末尾に `[org.gate]` とコメント、ルートの `ralph.toml` に `regate_checks = "ci"`(ヘッダのコメントも直した)、ずれの検査と `check-sync.sh` のコメント、`ralph config get` のテストに 2 行。検証: handoff のコマンドが通過(`internal/config`、`internal/cli -run Config` 12 秒、check-sync PASS)、検証・既定・ずれの行・template・ルートの値をそれぞれ外した版で新しいテストが落ちる。scaffold・upgrade・`cmd/ralph` のテストと check-template-purity も通過し、変えるテストはなかった。機械的な確かめは 5 項目とも clean
  - sync-docs に渡すこと: `.claude/rules/ralph/model-routing.md:151` の、defaults_sync_test が見るキーの一覧に `[org.gate]` を足す
  - S2 からは #263 の PR の merge のあと、main を取り込んでから始める(outline の 2)
  - S0 137ec668: #263(PR #288、c03647f9)の merge のあと origin/main を取り込んだ。衝突は `internal/config/config.go` の 1 か所で、#287 が検証を `validateLoaded`(error だけを返す)に切り出していたので、`[org.gate]` の検証をその形に合わせた。build・vet・check-sync・check-skill-sync と `internal/config`・`internal/gitroot`・`internal/cli -run Config` が通過
  - S2 9becd883: `EventGateCIPassed`・`EventGateCIFailed`、`gateViaRegateCI`、`GateReasonCIFailed`。`gateRecord.event()` は via で分け、merge-order の「`event()` と event の名前が合うときだけ読む」がそのまま組を守る(CI の Details を `gate_passed` で書いた記録は unreadable)。merge-order の CI の失敗の文(head を名指し、reset はない)、PushOnly で `regate_ci` の passed を断る文(AC10)、`regateResetHint` は前から `regate_ci` を外していたので注釈とテストだけ。watch は数えない(注釈とテスト)。`ralph org gate` の CLI は自分の結果しか出さないので変えない。検証: handoff のコマンドが通過(internal/org 525 秒、internal/cli 137 秒)、golangci-lint と staticcheck が 0 件、変更を戻した版で新しいテストが落ちる。機械的な確かめは 5 項目とも clean
  - 後のスライスに渡すこと(S2 の報告から): `internal/cli/org_gate.go` の `printGateFailDetail` は regate の CLI と共用で `ci_failed` で何も出さない(S6)。`org_merge_order.go` と `org_regate.go` の help が `gate_passed`/`gate_failed` だけを書く(S6・S7)。`regate.go` の `Regate` の概要と `RegateOutcomeUpToDate` の注釈(S4)。merge が `gate_ci_passed` で merge できることのテスト(AC3、S4 に入れる)
  - S3 ab1b0d43: `internal/org/prchecks.go` に `runGH`(前の `ghWith` の中身)、`findOriginPR`(`-R` に固定した PR の読み方。problem は Usable・Interrupted・GHFailed・LimitForks・None・Foreign・NotOpen・WrongBase)、`readPRHead`、`readPRChecks`(total、passed、pending、failed を gh の順に持つ。読めないときの理由と stdout も持つ。checks がない文は 0 件として読む)。merge の `findPR` と `readChecks` はこれを使い、文言と `mergeRefusalKind` は前のまま(merge の既存のテストの期待は 1 つも変えていない)。検証: `internal/org -run 'Merge|PRChecks|PRFind|PRHead'` 183 秒、`internal/cli -run OrgMerge` 50 秒、golangci-lint 0 件、helper を壊した版 6 つで新しいテストが落ちる。機械的な確かめは 5 項目とも clean
  - 計画とのずれ(本文は変えない): D4 の「落ちた check と pending が両方あるとき: pending と同じに扱う(merge と同じく…)」の括弧の中は正確でない。merge は落ちた check が 1 つでもあれば pending があっても NotPassed で止める。regate は計画どおり pending と同じに扱って待つ(`readPRChecks` が pending と failed を分けて返す)
  - S4 は 2 つに分ける。S4a は事前の確かめ、checks から記録への判定、gate だけを取り直す道(D6)、merge が CI の記録で merge できるテスト。S4b は merge と push の道(push を記録の前に、D3 の 3 つの分かれ)
  - S4a 2ab52083: `RegateParams` に `Checks`・`Exec`・`LookPath`・`GHTimeout`(local は Exec を一度も呼ばない)。ci の事前の確かめ `regateCIPreCheck`(`ReadMergeOrigin` → gh → `findOriginPR`)は fetch の直後、すべての道の前で 1 回(consult の判定で可。UpToDate と push だけの打ち直しでも gh と open な PR が要る)。判定 `regateChecks`(PR の head が試した head か → checks → passed / ci_failed / pending / すべて skipping / 読めない)。gate だけの道(D6)は ci でスクリプトの代わりに判定を打つ。記録は `recordRegate` の同じ道(確かめ直しの文は ci なら「the PR's checks were read」)、log は `gh pr checks` の JSON だけ。merge が `gate_ci_passed` で merge できるテスト(AC3)。検証: internal/org 502 秒、internal/cli 161 秒、golangci-lint・staticcheck 0 件、判定を壊した版 6 つで新しいテストが落ちる。機械的な確かめは 5 項目とも clean
  - S4b 688d4d6a: `RegateOutcomeChecksPending`(pending は記録しないので Gated と Merged の約束を守る)。ci の merge の道はスクリプトを走らせず、記録の前に push する。push の失敗は `git ls-remote origin refs/heads/<b>`(`regateReadOriginTip`)の読み返しで 3 つに分け、外すのは元の先頭のままのときだけ。push か読み返しの最中の割り込みは HEAD を残す。push のあとの断りと error は「merge commit は origin に残る」と言う。CLI の `printRegateLines` は `case Gated, Merged:` にして、知らない outcome で誤った `GATE: fail` を出さない(S6 で正式な表示に置き換える)。検証: internal/org 639 秒、internal/cli 156 秒、golangci-lint・staticcheck 0 件、変異 9 つがすべて落ちる。機械的な確かめは 5 項目とも clean
  - S6 に渡すこと(S4a・S4b の報告から): `printRegateResult` に `CHECKS: pending (<理由>)` と終了コード 6、`ci_failed` の落ちた check の表示(`FailedChecks`)、Merged と `ci_failed` の組で出る偽の「nothing was pushed」と空の NotUndone の行、LogPath のある断りに付く「the scripts' output is in」の ci 用の言い換え、知らない outcome のテスト
  - 13d19f3d: 完了した S4b の implementer が、作業中に送った返事で再開して足した直し(知らない outcome の CLI の行を固定するテストと、`Regate` の要約の注釈「ci の事前の確かめはすべての道の前で走るので、UpToDate と push だけの打ち直しでも gh と open な PR が要る」)。amend はさせず、202c2f85 の上の新しいコミットにした。S5 と同じ `regate.go` を触っていたので、S5 は自分の編集を退避して待ち、このコミットのあとに戻した
  - S5 33be080c: `RegateParams.WaitChecks`(範囲は `MergeWaitChecksRefusal` と同じ 0〜6 時間、local で 0 でない値は断る)。pending のあと、`regate.lock` を外して 30 秒待ち、取り直してから結びつきと止まった org、worktree の HEAD、origin の先頭(`regateReadOriginTip`)、ほかの regate の pass(あれば UpToDate)を確かめ直して判定を打ち直す。上限に届けば ChecksPending(理由に「待ちが尽きた」)。待ちの始めと終わりを Stderr に 1 行ずつ。割り込みは何も記録せず、merge commit は origin に残る。`recordRegate` の確かめ直しを `regateBranchChanged` と `regateChangedRefusal` に切り出して共用。検証: `internal/org -run 'Regate|MergeWait|Merge_'` 476 秒、golangci-lint・staticcheck 0 件、壊した版 14 個で新しいテストが落ちる。機械的な確かめは 5 項目とも clean
  - S6 に渡すこと(S5 の報告から): `--wait-checks` を `RegateParams.WaitChecks` に渡し、台帳を読む前に検証する(merge と同じ)。UpToDate と Pushed の組の表示。上限まで待った ChecksPending の行と終了コード 6。`internal/org/lockfile.go:140` の注釈「ralph org regate one from its first check to its end」は、ci の待ちで lock を外すので古い
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
