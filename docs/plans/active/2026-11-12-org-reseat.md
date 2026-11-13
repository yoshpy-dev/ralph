# 機能の org の座席を disband せずに立て直す `ralph org reseat`(#312c の 1 本目)

- Status: Approved
- Approved: 2026-11-13 sha256:3058f6f66b5b
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/312(利用上限で止まった座席を別のアカウントで立て直す手順と動詞が無い)。仕様は `docs/specs/2026-11-12-org-reseat.md`
- Related issue: 312
- Type: feat
- Branch: docs/spec-org-reseat

## Objective

`start --plan` で立てた機能の org の座席を、disband せずに止めて今の環境で立て直す動詞 `ralph org reseat` を足す。計画の承認、予約、機能との結びつきは org の今の命に残るので引き継がれる。仕様の D1〜D6、FR-1〜FR-8、FR-11、AC-1〜AC-12 を満たし、issue #312 の受け入れ条件 2 を満たす。経過の節と `leader.md` の規則(仕様の D7、FR-9、FR-10、AC-13、AC-14)は 2 本目の PR に回す。

## Scope

- `internal/org/reseat.go`(新しいファイル): 対象の座席を決める、座席の pane から打った reseat を断る、止める前の検査、止める、立てる、記録する
- `internal/org/feature.go`: `StartFeature` の検査の部分(分割計画の読み込みと承認、worktree の記録の検査、spawn の事前の判定)を、reseat が止める前に呼べる関数に切り出す。`StartFeature` の動きは変えない
- `internal/org/verbs.go` と `internal/org/spawn.go`: `StopParams` と `SpawnParams` に、Details の末尾に足す note を渡す欄を足す(`reseat=<id>`)
- `internal/org/seat.go`: org レベルの event `reseated`
- `internal/cli/org_reseat.go`(新しいファイル)と `internal/cli/org.go`(サブコマンドの登録): フラグ、設定と台帳の読み、見張りと herdr server の検査、出力、help
- 文書: org skill の 4 つの写し(動詞の表、reseat の節、アカウントの切り替えの手順)、director skill の 4 つの写し(人に頼まれたときの打ち方)、`docs/recipes/isolated-herdr-server.md` に分離した herdr での立て直しの 1 段落(要るなら)
- 実機の確かめ(AC-12): 分離した herdr で 1 回、`docs/evidence/org-reseat-live-2026-11-12.md` に記録

## Non-goals

- 経過の節と `leader.md` の規則(2 本目の PR)
- pane ごとに `CLAUDE_CONFIG_DIR` などを渡すこと(仕様の D6)
- 結びつきの無い org の立て直し(仕様の D4)
- 見張りが `usage_limited` の件を見て自動で reseat すること、画面の判定(`internal/org/resume_screens.go`)の変更
- 分割計画の再承認の規則の変更

## Assumptions

- `Stop` は座席レベルの event だけを書き、org の今の命を終えない。今の命の `plan_approved` と予約は disband まで効く(`internal/org/approval.go` の `currentPlanApprovals`、`internal/org/reserve.go` の `currentOrgLives`)
- `Stop` は pane が無い座席を「already closed」として `stopped` にする(`internal/org/verbs.go` の `stopSeatPane`)。herdr server を立て直したあとの座席も止められる
- `stopped` の座席は Spawn の新しい spawn の道を通る。ほかの座席の pane が残っていなければ新しい workspace を作り、最初の座席が root の pane になる(`internal/org/spawn.go` の `resolveWorkspace`、`splitParent`)
- `StartFeature` は、#312b(PR #320、d2e5c6e9)のあと、前文とその機能の節と承認方式が同じなら、分割計画のほかの節の変更では断らない
- `spawnPrecheckErr` は events だけから spawn の判定を先にかけられる(`TestSpawnPrecheckErr_MatchesSpawn` が Spawn と一致を確かめている)
- 台帳の読み手は state でない event を読み飛ばす(`internal/org/manifest.go`)。org レベルの `reseated` は古い binary の判定を変えない

## Affected areas

- `internal/org/reseat.go`、`internal/org/reseat_test.go`(新しい)
- `internal/org/feature.go`、`internal/org/feature_test.go`(検査の切り出し)
- `internal/org/verbs.go`、`internal/org/spawn.go`、`internal/org/seat.go`
- `internal/cli/org_reseat.go`、`internal/cli/org_reseat_test.go`(新しい)、`internal/cli/org.go`
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と、それぞれの 3 つの写し
- `docs/recipes/isolated-herdr-server.md`(要るなら)、`AGENTS.md` の `internal/org/` の行(sync-docs で)
- `docs/evidence/org-reseat-live-2026-11-12.md`(新しい)

## Visual review

- Page: `.harness/state/plan-visual/org-reseat.html`(やさしい図 2 枚: overview、easy-check。折りたたみの中に detail-overview、detail-sequence、detail-targets)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: 全体、overview、ほかの 4 枚をそれぞれ `shot` で撮って読んだ。文字のはみ出し、線と文字の重なり、矢印の外れはなかった

## Design decisions

Critical forks: None。仕様(`docs/specs/2026-11-12-org-reseat.md` の D1〜D8)が分かれ目を決めている。次の実装の選び方は、コードから決められるので自律で選んだ(人の方針「推奨案を自動承認」に従う)。

- P1 止める前の検査は、台帳の events に、止める座席の `stopped` を足した写しを作り、立てる順に座席ごとに spawn の事前の判定(`spawnPrecheckErr`)をかける。1 席を検査するたびに、その座席の `spawned` を写しに足し、次の座席の上限の数え方を実行時と同じにする。leader は、`StartFeature` から切り出した検査(分割計画の読み込みと承認、機能、結びつきの判定)も通す。切り出す関数は manifest を読まず、events を引数に取り、`StartFeature` は実際に読んだ events を渡す。worktree の記録は start より厳しく、記録が有り、結びつきの Worktree と Branch に一致することを求め、無ければ止める前に断る(reseat は新しい worktree を作る場面ではなく、記録が無いと止めたあとの ensure が既にある branch で落ちるため。consult の指摘)。見張りの検査、herdr server の検査、regate のロックの検査(`regateSpawnErr`)も、止める前に 1 回かける(`--dry-run` では見張りと server を起動しない、P8)。理由: Spawn と StartFeature が後でかける判定と同じ関数を、止めたあとの台帳の形で先にかけられる。判定を二重に書かない。残る失敗(herdr と agmsg の失敗、同じ座席の spawn のロックの競合)は FR-6 の出力で扱う
- P6 reseat は、検査の前から記録を書き終えるまで、台帳ごとのロック `reseat.lock`(台帳のディレクトリ、`regate.lock` と同じ作り)を待たずに取る。取れなければ、持っている reseat の org を言って断る。止める座席ごとに、止める直前に台帳を読み直し、その座席の最新の state event が検査のときと同じ(同じ位置)であることを確かめる。違えば(leader がその座席を立て直した、人が stop した、など)そこで止め、止めた座席と残った座席、打ち直し方を出す。理由: 計画助言の指摘 1(2 つの reseat が同じ座席を読み、片方が立て直したあとに、もう片方の古い停止がその座席を `stopped` と記録しうる)。Stop は座席の spawn のロックを取らない(tech-debt の 361 行目)ので、reseat どうしの重なりはロックで、ほかの動き手との重なりは直前の読み直しで防ぐ
- P7 `--seat` は、今の命で一度 `spawned` になった座席なら、最新の state が active、`stopped`、`spawn_failed`、`rejected` のどれでも受け付ける。今の命で今 active な座席と、reseat が止めたか立てようとした座席(今の命の event の Details が `reseat=<id>` で終わる座席)も受け付ける。今の命で一度も `spawned` にならず、active でもなく、reseat が触れてもいない座席(例: 手で止めた座席)は断る。「今 active な座席がない」の断りは、`--seat` が受け付ける座席だけを挙げる(2026-11-13 の cross-review の直しで足した)。理由: 計画助言の指摘 2(起動に失敗した座席は `spawn_failed` か `rejected` になり、FR-6 が出す打ち直しの `--seat` を断ってしまう)。失敗した起動は意図した停止ではないので、D2 の「leader が意図して止めた座席を蘇らせない」は既定の対象(active だけ)で守る
- P8 `--dry-run` は何も起動しない。見張りと herdr server は、動いているかを読むだけにし、動いていなければ「本当の reseat が起動する」と出す。理由: 計画助言の指摘 3(dry-run が今の環境で server を起動すると、アカウントを切り替える前の環境の server が残る)。本当の reseat では、止める前に今までどおり起動する
- P9 途中で切れた reseat(self-review の M1 を受けて足した、実装中のずれ): CLI は本当の reseat のあいだ SIGINT・SIGTERM・SIGHUP を受け止め、止めて立てる処理を最後まで続ける(受けたら stderr に 1 行)。読み手の居ない pipe への書き込みで終わらないよう、SIGPIPE も受け止める(2 回目の self-review の M2)。終了コードは reseat の結果のまま。kill やクラッシュで切れたときに備え、既定の対象(`--seat` なし)に、今の命で最新の state が `stopped` で、Details に `reseat=<id>` があり、その id の `reseated` が無い座席(終わっていない reseat が止めた座席)を足す。同じコマンドを同じ `--seat` のまま打ち直せば立て直しが終わる(`--seat` なしなら、切れた reseat の座席に加えて active な座席もすべて立て直す)。理由: 止めた座席が記録も案内もなく残り、打ち直しでも拾われない穴は、今回の変更が持ち込んだ知らせが消える向きの穴なので直す
- P10 役割の prompt の雛形が無い独自の役割の座席(`--prompt` で立てた座席)は、台帳から指示を作り直せないので、対象にしたら断る(self-review の L1)。断りの文は、その座席を止めて `--prompt` を付けて立て直すよう言う
- P2 reseat の id は、時刻と 4 桁の 16 進数(`<UTC の YYYYMMDDThhmmss>-<4 hex>`)。理由: 同じ台帳で一意で、人が読んで時刻が分かる
- P3 `StopParams` と `SpawnParams` に `Note string` を足し、Details の末尾に ` reseat=<id>` として足す。既存の Details の形は変えない。`StartFeatureParams` にも同じ欄を足し、leader の spawn に渡す
- P4 org レベルの `reseated` は、止める処理を 1 つでも始めたら必ず書く(全部が立ったときも、途中で失敗したときも)。Details は `id=<id> seats=<座席>:<結果>,...` と、driver か model を変えた座席の `changed=<座席>:<driver>/<model>`。理由: 途中で失敗したときも、何が起きたかを台帳から読めるようにする
- P5 driver と model: 座席ごとに、台帳の最新の `spawn_started` か `spawned` の Driver と Model を使う。`--seat` が 1 つで `--driver`・`--model` があればそれを使う(`ResolveRoleModel` と同じ検証)。Model が `""` の記録は CLI の既定で立てた座席なので、`""` のまま渡す

## Acceptance criteria

仕様の受け入れ条件のうち、1 本目の分。

- [ ] AC-1 機能の org の leader・implementer・reviewer が active で、per-plan の計画の承認が今の命にあるとき、`ralph org reseat --org-id <id>` で 3 席が止まり、leader、implementer、reviewer の順に立つ。`scope_reserved` は増えず、implementer の spawn と implementer への TASK は承認の検査を通る(issue #312 の受け入れ条件 2)
- [ ] AC-2 全座席を立て直したとき、leader が最初に立ち、新しい workspace の最初の pane になる
- [ ] AC-3 `--seat implementer --driver codex --model <m>` では implementer だけが codex で立ち、leader は動き続ける。出力の最後に、leader へ送る DECISION の `ralph org send` の行が出る
- [ ] AC-4 分割計画が承認のあと変わった、自分の節か前文を直した、分割計画のファイルが無い、見張りが動いていない、機能の worktree の記録が無いか結びつきと合わない、のどれかのとき、どの座席も止めず、何も記録せずに終了コード 1 で理由を言う
- [ ] AC-5 結びつきの無い org の reseat は断られ、stop と spawn の打ち方が出る
- [ ] AC-6 active な座席が無く `--seat` も無いとき、断って今の命の `stopped` の座席と `--seat` を付けた打ち方を出す。`--seat` を付けて打ち直すと立つ
- [ ] AC-7 その org の座席の pane から打った reseat は断られる
- [ ] AC-8 座席の pane が無いとき(herdr server を立て直したあと)も、座席が立ち直る
- [ ] AC-9 `--dry-run` は何も書かず、止める座席、立てる順、各座席の driver と model を出す
- [ ] AC-10 `Stop` が `stop_failed` になった座席があれば、座席を立てず、止めた座席と残った座席と打ち直し方を出して終了コード 1
- [ ] AC-11 `--driver` だけ、または `--seat` を 2 つ付けて `--model` を渡すと断られる
- [ ] AC-12 分離した herdr の実機で、AC-1 と AC-8 を 1 回ずつ確かめ、`docs/evidence/org-reseat-live-2026-11-12.md` に記録する
- [ ] AC-17 ある reseat が `reseat.lock` を持っているあいだ、別の reseat は何もせず断られる。検査のあと、止める前に座席の記録が変わったとき(ほかの動き手が立て直した、止めた)は、その座席を止めずに止まり、状態と打ち直し方を出す(決まった順で起きるテストで確かめる)
- [ ] AC-18 起動の途中で失敗させたあと(leader の起動の失敗、非 leader の起動の失敗)、出力に出た打ち直しのコマンドをそのまま打つと座席が立ち直る(`spawn_failed`・`rejected` の座席も `--seat` で受け付ける)
- [ ] AC-19 `--dry-run` は、止まっている herdr server と見張りを起動しない。台帳にも何も書かない
- [ ] AC-20 終わっていない reseat が止めた座席(`reseat=<id>` を持つ `stopped` があり、その id の `reseated` が無い)は、`--seat` なしの reseat の対象になり、打ち直しで立つ。終わった reseat の座席や、人が止めた座席は対象にならない。本当の reseat の CLI は SIGINT・SIGTERM・SIGHUP と、読み手の居ない stderr への書き込み(SIGPIPE)で途中で終わらない
- [ ] AC-21 役割の prompt の雛形が無い座席を対象にすると、何も止めずに断られ、手で立て直す方法が出る
- [ ] AC-15 止めた座席の `stopped` と立てた座席の `spawned` に `reseat=<id>` があり、org レベルの `reseated` が 1 件ある。途中で失敗したときも `reseated` が座席ごとの結果を持つ
- [ ] AC-16 org skill と director skill の 4 つの写しと `ralph org reseat --help` が、動詞と、別のアカウントへ移す手順(herdr server を止めると全 org の pane と herdr の中の headless の director も消えること)を言う

## Implementation outline

1. S1 検査の切り出しと立て直しの段取り(`feature.go`、新しい `reseat.go` の前半): `StartFeature` の検査を関数に切り出し、`(o *Org) PlanReseat` が対象の座席、順序、各座席の driver と model、検査の結果を返す(止めない、書かない。dry-run では何も起動しない)。AC-4〜AC-7、AC-9、AC-11、AC-19 と、P7 の `--seat` の受け付けのテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'Reseat|StartFeature|SpawnPrecheck' ./internal/org/`
2. S2 立て直しの実行(`reseat.go` の後半、`verbs.go`・`spawn.go` の note、`seat.go` の event): `(o *Org) Reseat` が `reseat.lock` を持って止めて立てて記録する(止める直前に座席の記録を読み直す)。AC-1〜AC-3、AC-8、AC-10、AC-15、AC-17、AC-18 のテスト(偽の herdr) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'Reseat|StartFeature|Stop|Spawn_Feature|SpawnFeature' ./internal/org/`
3. S3 CLI(`internal/cli/org_reseat.go`、`org.go`): フラグ、`runOrgStartPlan` と同じ設定と台帳の読み、見張りと herdr server の検査、出力と DECISION の行、help。CLI のテスト — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'Reseat|OrgHelp|OrgCommand' ./internal/cli/`
4. S4 文書(org skill と director skill の 4 つの写し、要るなら recipe): AC-16 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`
5. S5 実機の確かめ(AC-12): 分離した herdr(`docs/recipes/isolated-herdr-server.md`)で、小さな分割計画の機能の org を立て、reseat と、herdr server を立て直したあとの reseat を 1 回ずつ打ち、台帳と pane を記録する。ユーザーやほかのセッションの herdr には触れない — verify: `test -s docs/evidence/org-reseat-live-2026-11-12.md`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: 仕様の FR-1〜FR-8 と FR-11、上の AC。特に FR-4(止める前の検査で、後の start と spawn が断る条件をすべて先に確かめる)と FR-5・FR-6(途中の失敗で状態と打ち直し方を出す)
- Documentation drift to check: org skill と director skill の 4 つの写し、`ralph org reseat --help`、`AGENTS.md` の `internal/org/` の行、`docs/specs/2026-11-12-org-reseat.md` の FR の印
- Evidence to capture: `--dry-run` の出力、台帳の `stopped`・`spawned`・`reseated` の実例、実機の記録

## Test plan

- Unit tests: 対象の座席の決め方(D2: 既定、`--seat`、active が無いとき、今の命に無い座席)、フラグの組み合わせ(AC-11)、座席の pane からの断り(AC-7)、検査の失敗(AC-4、AC-5。worktree の記録が無い org と、regate のロックが掛かった org は止めずに断る)、`--dry-run`(AC-9)、reseat の id と note の形
- Integration tests: 偽の herdr と偽の worktree で `Reseat` を回す。全座席(AC-1、AC-2、承認の検査が通ること)、1 座席と driver の変更(AC-3)、pane が無い座席(AC-8)、`stop_failed`(AC-10)、起動の途中の失敗、記録(AC-15)。CLI の出力と終了コード
- Regression tests: `StartFeature` の既存のテスト(検査の切り出しで動きが変わらないこと)、`TestSpawnPrecheckErr_MatchesSpawn`、Stop と Spawn の既存のテスト(note が空のとき Details が今と同じ)
- Edge cases: leader だけの org、reviewer がまだいない org、同じ座席を 2 回 `--seat` で名指し、`--seat` に leader と非 leader の両方、分割計画のファイルが消えた org、承認方式が split の org
- Evidence to capture: `go test` の出力、実機の記録

## Risks and mitigations

- 止めたあとに立てる処理が失敗し、座席が足りない org が残る → 止める前の検査で、後で断られる条件を先に確かめる(P1)。それでも失敗したら、立った座席と立たなかった座席、打ち直し方を出し、`reseated` に結果を残す(P4)
- 検査と実行のあいだに、ほかの spawn か start か reseat が台帳を変える(同じ org の leader が非 leader を立てる、2 つ目の reseat など)→ reseat どうしは `reseat.lock` で重ならない(P6)。止める直前に座席の記録を読み直し、変わっていれば止めずに止まる(P6)。Spawn は自分のロックの下でもう一度判定するので、二重に立つことはない。断られたら FR-6 の打ち直しの案内に乗る(P7 で失敗した座席も受け付ける)。読み直しと Stop の書き込みのあいだの窓は残る(Stop が座席のロックを取らない、前からの型)。Known gaps に書く
- `StartFeature` の検査の切り出しで、start の動きが変わる → 切り出しは関数の移動だけにし、既存の StartFeature のテストと `TestSpawnPrecheckErr_MatchesSpawn` で確かめる
- `StopParams`・`SpawnParams` に note を足すことで、既存の Details の形が変わる → note が空なら何も足さない。既存のテストで確かめる
- 実機の確かめで claude の座席が利用上限に当たる、またはほかのセッションの herdr に触れる → 分離した herdr でだけ打つ。小さな機能にし、確かめ終えたら disband する。上限で確かめられなければ、未確認として記録する

## Rollout or rollback notes

- 新しい動詞と event を足すだけで、既存の動詞の動きは変えない。古い binary は `reseated` と `reseat=` の note を読み飛ばす
- release ノートに書く項目: `ralph org reseat`、別のアカウントへ移す手順(herdr server を新しい環境で立て直してから reseat)

## Open questions

- 全座席が agmsg の team から Leave したとき、team が消えるか(仕様の Open questions)。S5 の実機で確かめる

## Progress checklist

- [x] Plan reviewed — 先に consult(consult-plan-reseat)に 2 回聞き、spec の D5・D3・D2 と、plan の P1(worktree の記録の存在と一致、regate のロック、写しに spawned を順に足す)を直した。plan advisory: ran with codex。指摘 3 件をすべて直した。(1) [HIGH] 2 つの reseat が重なると、立て直した座席を古い停止が `stopped` と記録しうる → P6(`reseat.lock` と止める直前の読み直し)、spec の FR-5a、AC-17。(2) [HIGH] 出す打ち直しのコマンドが起動に失敗した座席を断る → P7(今の命で一度立った座席は失敗のあとも受け付ける)、spec の FR-2、AC-18。(3) [MEDIUM] `--dry-run` が herdr server を起動しうる → P8(dry-run は読むだけ)、spec の FR-8、AC-19。図解ページの「見ておくこと」と詳細 2・3 を直し、lint と撮り直しで確かめた
- [x] Plan approved — 2026-11-12、ユーザーの方針(推奨案を自動承認)により承認ゲートを聞かずに通した
- [x] Branch created — docs/spec-org-reseat(spec から同じ worktree)。origin/main(#312b、d2e5c6e9)を 26f3ef44 で取り込み済み
- [x] Implementation started — 2026-11-12、base 40aebad9。cycle 1
  - S1 d7389e50(implementer): `PlanReseat`、`StartFeature` の検査を `startFeatureInput` と `checkLedger` の 2 つに切り出した(入力の検査が見張りと server の gate より前に来る今の順を保つため)。leader が対象外のときは結びつきの判定、regate のロック、worktree の記録だけを見る。`--seat` は今の命で spawned がある座席か今 active な座席。dry-run の読み取りは `ReseatParams` の `WatchProbe`・`HerdrServerProbe`。拒否は `reseatRefusal` で包む。検証コマンド ok、変異 7 つが落ちることを確かめた
  - S2 6a40276e(implementer): `Reseat`、`reseat.lock`(待たずに 1 回)、止める直前の読み直し(`LatestIndex` と `Latest`)、`Note`(Stop・Spawn・StartFeature)、`reseated`(止めるか立てるかを 1 つでも試したとき)、打ち直しと DECISION のコマンド。検証コマンド ok、変異 12 個。途中でディスクが満杯になり、go-build の 1 日以上使われていない項目を消した(ralph-d1 に連絡済み)
  - S3 ecb84fa8 + dd204370(implementer): `ralph org reseat` の CLI と help。立てる順を今の命の最初の spawn_started の順に直した。2 本目のコミットで、フラグの検査を `reseat.lock` の前に移した(consult の指摘)。検証コマンド ok
  - S4 51d0129d(implementer): org skill(動詞の表、「座席の立て直し」の小節)と director skill(「座席を立て直す」の節、「してはいけないこと」の 1 項)の 4 つの写し。check-skill-sync と check-sync ok
  - S5(分離した herdr、別の担当): `docs/evidence/org-reseat-live-2026-11-12.md`。binary は dd204370。AC-1 は leader と reviewer の 2 席で合格(止める順、立てる順、leader が新しい workspace の root の pane、`scope_reserved` が増えない、`reseat=<id>` と `reseated`)。implementer と承認の検査は実機では見ていない(費用を抑えるため。テストで確かめる)。AC-8 は合格(server の立て直しで戻った pane は中身が無く、Stop は `ctrl_c=failed ... pane_not_found ... pane=closed` で閉じた)。仕様の開いた問い: 全座席が Leave しても agmsg の team と履歴は消えない。ほかの herdr server(pid 34407)は同じ pid のまま
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
- 実装中のずれ(2026-11-12): /self-review(6aa5188f、MEDIUM 1・LOW 6)の M1 と L1 を受けて、Design decisions に P9(途中で切れた reseat の座席を既定の対象に足す、CLI は中断の signal を受け止める)と P10(独自の役割の座席を断る)、AC-20・AC-21 を足し、spec の D2 に注を足した。図解ページの詳細 3 を直した。承認を digest 1f1ecc9ed258 で記録し直した(人の方針「推奨案を自動承認」)。直しは S6 として implementer に渡し、パイプラインを /self-review からやり直す
  - S6 93395521(implementer): CLI の signal の受け止め、切れた reseat の座席を既定の対象に、独自の役割の座席を断る、`FeatureRepoRoot` の断りの文。検証コマンド ok、変異 11 本。origin/main(#309、0936aaea)を f0c53cde で取り込み(衝突なし)
  - /self-review の 2 回目(f597b3dd): MEDIUM 1(M2 SIGPIPE)、LOW 8。M2・L8・L9 を S7 で直した。S7 e5b2cb85(implementer): SIGPIPE の受け止め、打ち直しの案内を「同じ --seat のまま」に、独自の役割の断りの spawn の行に --scope と記録の driver・model。検証コマンド ok
  - inline 772da303: S7 の報告の範囲外 1(golangci-lint の errcheck が reseat_test.go の helper の戻り値で 15 件)を直した。helper を戻り値なしにし、戻り値を読む 4 か所は `checkReseatRefused` を呼ぶ。検証: `go test -run Reseat ./internal/org/` ok、`golangci-lint run ./internal/org/ ./internal/cli/` 0 issues。self-check: 不要な変更 clean、誤字 fixed(Edit で落とした空白を gofmt で見つけて直した)、デバッグの残り clean、秘密 clean、範囲外のファイル clean
  - P9・AC-20 と spec の D2 の注を、SIGPIPE と「同じ --seat のまま打ち直す」に合わせて直した。承認を記録し直した
  - /self-review の 3 回目(f8579b7c の report): blocking 0、LOW L2〜L6・L10。`docs/reports/self-review-2026-11-12-org-reseat.md`
  - /verify(af927216): pass。drift V1〜V8(LOW)、INFO I1〜I6。`docs/reports/verify-2026-11-12-org-reseat.md`
  - /test(69908490): pass。run-test.sh(shell 45 本、Go 10 パッケージ)、`-race`、変異 10 個のうち 5 個が生き残り(M4、M6、M9、M68、C1)、Test gaps 8 件。`docs/reports/test-2026-11-12-org-reseat.md`
  - /sync-docs: V1〜V8 と INFO I2 を直した(skill の 4 つの写し、README、AGENTS.md、quality-gates の 2 つの写し、仕様の印と Open questions、`docs/specs/2026-08-01-org-runtime.md` の改訂の項、help の使い方の文、tech-debt の持ち越しの注 12 行と新しい 3 行)。仕様の AC-12 は、2 席の実機の限りを書いたうえで `[x]`。`docs/reports/sync-docs-2026-11-12-org-reseat.md`
  - 2 回目のパイプライン(2026-11-13): cross-review(triage 2bc28bdc)の ACTION_REQUIRED 3 件を直した(adc3e205、e9461b79、0468d96f)。origin/main の #310 と #308 を 8fbd7118 と 9a295a84 で取り込み。P7 の `--seat` の受け付ける座席と、DECISION を部分失敗のあとも出すこと、打ち直しの command に絶対の `--config` を付けることを spec と P7 に反映した(sync-docs の 2 回目)
  - /self-review の 2 回目(f754188f、c50d79d6): blocking 0、LOW(L2、L3(b)、L4、L5、L6、L10、L13〜L16 が open、L11 は閉じ、L12 は L15 に置き換え)。`docs/reports/self-review-2026-11-12-org-reseat.md`
  - /verify の 2 回目(80f58f86): pass。drift V9〜V16(LOW)、INFO I7〜I10。`docs/reports/verify-2026-11-12-org-reseat.md`
  - /test の 2 回目(2d37fa32): pass。変異 11 本をすべて殺した。org と cli のパッケージは負荷のもとで 40 分の上限に 1 度当たり、同じ上限での再実行で pass。`docs/reports/test-2026-11-12-org-reseat.md`
  - /sync-docs の 2 回目: V9〜V13 と V16 を直し、V14・V15 は tech-debt に持ち越した(help の文は Go なので、直すとパイプラインを /self-review からやり直す)。P7 を変えたので計画の digest が変わる(3058f6f66b5b から)。承認の記録し直しはオーケストレーターが行う。`docs/reports/sync-docs-2026-11-12-org-reseat.md`
