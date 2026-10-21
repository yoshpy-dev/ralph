# seat-agmsg-self-name

- Status: Approved
- Approved: 2026-10-22 sha256:dc40c452f092
- Owner: Claude Code
- Date: 2026-10-22
- Related request: 試行の所見を直す系列の 4 本目の前半「座席の環境 4a」(ユーザーの依頼 2026-10-21「2. 修正可能なものは修正してください」)。consult(consult-plan-seatenv)の判定で 4 本目を 4a と 4b に分けた
- Related issue: N/A
- Type: fix
- Branch: fix/seat-agmsg-self-name

## Objective

agmsg 1.5.1 が herdr の agent と pane の名前を書き換えて、ralph が座席を見失う所見 F3 を直す。ralph が打つ agmsg の呼び出しと、ralph が作る座席の pane の両方で、agmsg の書き換えを止める(`AGMSG_SELF_NAME=off`)。あわせて、見張りから leader への ALERT が 10 回時間切れになった所見(「leader への ALERT」)の原因が、この書き換えかどうかを分離した herdr で測る。前の PR #247 の記録の書き換え(V4)も入れる。

## Scope

- ralph が打つ agmsg のスクリプト(`internal/org/driver/agmsg.go` の Send・Join・TeamMembers・History・Leave・Whoami・DeliverySet)を、どれも `AGMSG_SELF_NAME=off` の環境で打つ。打つ側の環境に別の値があっても off が勝つ
- ralph が作る座席の pane(`ralph org spawn` と `ralph org start` の `herdr tab create`)と、herdr で立てる headless の director の pane に、`herdr tab create --env AGMSG_SELF_NAME=off` を付ける。座席が自分で打つ agmsg(`inbox.sh` など)でも名前が変わらないようにするため
- `--env` のない herdr では `--env` を付けずに pane を作り、stderr に警告を 1 行出す。確かめ(`herdr tab create --help`)は 1 プロセスで 1 回
- `[org.director].launch_cmd` の起動スクリプトが、エージェントを exec する前に `AGMSG_SELF_NAME=off` を export する
- 分離した herdr server での計測と確かめ(下の S1)。結果を `docs/evidence/agmsg-self-name-2026-10-22.md` に書く
- 記録: tech-debt の F3 の行と「leader への ALERT」の行、根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md`、仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の印の注と Status、V4(#247 への書き換えと PR 一覧)

## Non-goals

- 4b の範囲: F4(spawn の補償で pane を閉じる、announce が落ちたが動いた座席の記録)と F11(session end の WIP コミット)
- agmsg 自身を直すこと、agmsg の版を上げ下げすること
- 試行のときに書き換わった名前を、今の環境で戻すこと
- herdr の最低の版を決めること、doctor に herdr の版の検査を足すこと
- 系列の残り(5 分割計画の前文、6 見張りの escalation、7 busy な leader、8 文書のまとめ)
- release を切ること

## Assumptions

- agmsg 1.5.1 の `agmsg_self_name_on_action`(`lib/self-name.sh`)、`agmsg_self_rename_on_action`(`lib/self-rename.sh`)、`join.sh` の `agmsg_terminal_name_self_safe`(`lib/terminal-registry.sh`)は、どれも `AGMSG_SELF_NAME=off` で何もしない(スクリプトを読んで確かめた。agmsg 自身の `spawn.sh` も join をこの形で打つ)
- 試行では 03:39Z から座席の環境に `AGMSG_SELF_NAME=off` を足して、そのあとの座席は DECISION と TASK を受け取れた(`docs/evidence/org-trial-2026-10-20.md` の D-F3)。off で agmsg の配送が壊れないことの根拠はこれだけで、S1 でもう一度見る
- herdr 0.7.5 の `herdr tab create` と `herdr workspace create` の help には `--env <KEY=VALUE>`(「Set an environment variable for the launched process」)がある。`herdr agent start` にはない。`--env` が環境を足すのか置き換えるのかは S1 で確かめる
- 本番で agmsg と herdr を打つ Runner は `driver.ExecRunner` だけ(`internal/cli/org.go:349`、`internal/cli/org_director_headless.go:787`)

## Affected areas

- `internal/org/driver/driver.go`(環境を足して打つ口)
- `internal/org/driver/agmsg.go`(すべての呼び出しに off)
- `internal/org/driver/herdr.go`(tab create の `--env`、`--env` の確かめと警告)
- `internal/cli/org.go`、`internal/cli/org_director_headless.go`(driver の組み立て)
- `internal/org/director_cmd.go`(起動スクリプトの export)
- 各テスト(`internal/org/driver/*_test.go`、`internal/cli/*_test.go`、`internal/org/director_cmd` のテスト)
- `docs/evidence/agmsg-self-name-2026-10-22.md`(新規)
- `docs/tech-debt/README.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、`docs/specs/2026-10-07-org-multi-org-director.md`

## Visual review

- Page: `.harness/state/plan-visual/seat-agmsg-self-name.html`(今の形。かんたんな図 2 枚: `overview`、`easy-alert`。折りたたみの中: `detail-overview`、`detail-spawn`、`detail-probe`)
- Lint: `./scripts/plan-visual.sh lint` が rc 0
- Self-check: 全体と 5 枚の図を `shot` で撮って読んだ。`detail-spawn` の 2 本目の矢印の文字が herdr の線にかかっていたので 2 行に分け、撮り直して読んだ。計画の Codex 助言を受けて S1 の説明と「気をつけること」を足したあと、lint と全体・`easy-alert` の撮り直しをした。どの PNG にも文字の重なりと切れはない

## Design decisions

- D1 環境を足す口: `driver.Runner` に手を入れず、任意の interface(仮名 `EnvRunner`、`RunEnv(ctx, env []string, name string, args ...string)`)を足し、`ExecRunner` が実装する。`Agmsg.run` は Runner がこれを実装していれば `AGMSG_SELF_NAME=off` を付けて打ち、していなければ今の `Run` で打つ。argv は変えないので、今の stub テストの前置の assert はそのまま通る。本番の組み立てが実装する Runner を使っていることは、CLI 側のテストで固める(consult の指摘: argv に `env` を前置する案より小さい)
- D2 pane の環境: `HerdrClient` interface の `TabCreate` の引数は変えない。`driver.Herdr` に、pane に足す環境の欄と、`--env` の確かめの結果を持つ欄を足し、本番の組み立てで `AGMSG_SELF_NAME=off` を渡す。欄が空なら今までどおり `--env` を付けず、確かめもしない(テストの fake と、今の `Herdr{R: ...}` の書き方はそのまま)
- D3 古い herdr: `herdr tab create --help` の出力に `--env` がなければ、付けずに作って stderr に 1 行の警告を出す。help が失敗したときも付けない(付けて tab create が落ちるより、作って警告するほうを採る)。確かめは同じ `driver.Herdr` の値で 1 回だけ打つ(consult の指摘: spawn ごとに打つと 10 秒の予算の中の herdr の呼び出しが 1 つ増える)
- D4 計測: 試行の 10 回の時間切れを再現できなくても、計測の数字と、ralph が off を付けたことを記録する。ALERT の行は、原因が書き換えだと示せたときだけ閉じる(consult の判定)
- Critical forks: None(上の 4 つは consult と相談して決めた。どれも 1 スライスの中で戻せる)

## Acceptance criteria

- [x] AC1: ralph が打つ agmsg のすべての呼び出し(Send・Join・TeamMembers・History・Leave・Whoami・DeliverySet)が、`AGMSG_SELF_NAME=off` を含む環境で打たれる。打つ側の環境に `AGMSG_SELF_NAME=on` があっても、子の環境の値は off になる
- [x] AC2: 本番の組み立て(`ralph org` の verbs が使う Org)の agmsg の Runner は、環境を足して打てる。テストがこれを固める
- [x] AC3: `--env` のある herdr では、座席の pane と herdr の headless の director の pane の `herdr tab create` に `--env AGMSG_SELF_NAME=off` が付く。`--env` の確かめは同じ driver の値で 1 回だけ打たれる
- [x] AC4: `--env` のない herdr(help に `--env` がない、または help が失敗する)では、`--env` を付けずに pane を作り、spawn と start は今までどおり成功し、stderr に警告が 1 行出る(何が起きうるかを書く)
- [x] AC5: launch_cmd の起動スクリプトが、check-in とエージェントの exec の前に `AGMSG_SELF_NAME=off` を export する
- [x] AC6: 分離した herdr server での計測と確かめを `docs/evidence/agmsg-self-name-2026-10-22.md` に書く。(a) `tab create --env FOO=bar` の値が pane の shell とその子に届き、PATH と HOME が残るか。(b) pane の中で打った agmsg が、off なら pane の名前を変えず、on なら変えるか(対照)。(c) 見張りの環境(HERDR_PANE_ID なし)と座席の pane の環境で、send.sh の時間を on と off で測った数字、同時に打った場合の数字。(d) 受け取る側に届くか: 条件ごとに新しい agmsg の状態(team と DB を作り直し、on の条件で書かれた placement の記録を使い回さない)で、off で送った ALERT と同じ形のメッセージを、受け取る側の `inbox.sh`(off)が読めること。(e) 本物の座席で名前が残るか: pane を `--env AGMSG_SELF_NAME=off` 付きで作り、`herdr agent start` で claude(haiku)を起動し、その座席が自分の Bash tool から `inbox.sh` を打ったあとも、`herdr agent get <名前>` でその名前が引けること。`--env` なしの pane(on)を対照にする。(f) ALERT の行を閉じるか開いたままにするかの判断と理由。ユーザーの herdr server、`~/.config/herdr`、Orca に触れていないこと
- [x] AC7: 記録を合わせる。tech-debt の F3 の行は、AC6 の (e) が通ったときだけ RESOLVED にする(通らなければ、残る部分を書いて開いたままにする)。「leader への ALERT」の行を AC6 の結果で書き換える。根拠の表の F3 と「leader への ALERT」の行、仕様の FR-5 と FR-3 の印の注を合わせる。V4: 根拠の表の F9・F10・G1 の行と、両方の文書の FR-7 の注にある「fix/test-stability」を「#247」に、両方の文書の PR 一覧に #247 を足し(30 本)、仕様の Status に「試行の所見 F9・F10・G1 の直し: #247」を足す
- [x] AC8: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0。変えたパッケージの Go のテストが `GOTOOLCHAIN=go1.25.8` でも通る

## Implementation outline

1. S1(計測、orchestrator が打つ): 分離した herdr server(`/tmp` の短い directory。unix socket の path の長さの上限のため。XDG の 3 変数と `HERDR_SOCKET_PATH`)と agmsg の写し(scratch の下、`scripts`・`plugins`・`types`・`SKILL.md`・`VERSION` だけで、ユーザーの `db`・`teams`・`run`・`agents` は写さない)で AC6 の (a)〜(e) を確かめる。(a)〜(d) は偽の HOME で打つ。(e) の claude は認証のために本当の HOME で起動し(試行と同じ)、`--model haiku` と Bash だけの許可で 1 つのコマンドを頼む。`--env` が環境を置き換えるなら D2 を見直す。結果を evidence に書いてコミットする
2. S2(implementer): D1。`EnvRunner` と `ExecRunner.RunEnv`、`Agmsg.run` の off。テスト(AC1、AC2)
3. S3(implementer): D2・D3。`driver.Herdr` の pane の環境と確かめ、本番の組み立て。テスト(AC3、AC4)
4. S4(implementer): 起動スクリプトの export。テスト(AC5)
5. S5(implementer、sonnet): 記録(AC7)。S1 の結果で ALERT の行の書き方を決める

## Verify plan

- Static analysis checks: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh`(gofmt、go vet、check-sync、check-skill-sync ほか)
- Spec compliance criteria to confirm: AC1〜AC8。仕様の印の数(`[x]` 21 / `[ ]` 11、根拠の表は 20 / 11)は、この PR で変わるならその理由を書く
- Documentation drift to check: `docs/specs/2026-08-01-org-runtime.md` と `.claude/rules/ralph/agent-messaging.md` と `/org`・`/director` の skill に、agmsg の環境や座席の pane の作り方を書いた箇所があれば合わせる。`CLAUDE_ENV_FILE` の回り道(D-F3)を手順として書いた箇所がないか
- Evidence to capture: run-verify の rc と時間、evidence の文書

## Test plan

- Unit tests: 7 つの agmsg の呼び出しが env を記録する fake で `AGMSG_SELF_NAME=off` を受け取る。`ExecRunner.RunEnv` が本物の `sh` の子に値を渡し、親の `AGMSG_SELF_NAME=on`(`t.Setenv`)より勝つ。`Herdr.TabCreate` が `--env` を付ける場合、付けない場合(help に `--env` がない、help が失敗、欄が空)、確かめが 1 回だけ打たれる、警告が 1 回だけ出る
- Integration tests: `internal/cli` の本番の組み立て(`newOrgRuntimeAt` など)で、agmsg の Runner が環境を足せること、Herdr に `AGMSG_SELF_NAME=off` が渡ること。spawn と director start のテストが fake の herdr のまま通ること
- Regression tests: 今の driver と spawn と director のテストが argv を変えずに通る
- Edge cases: 親の環境に `AGMSG_SELF_NAME=on`、help の出力に `--env` が別の文脈で出る(例: `--envelope`)ときに誤って付けない、help が時間切れ
- Evidence to capture: `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` を go1.26 と go1.25.8 で

## Risks and mitigations

- `--env` が pane の環境を足すのではなく置き換えると、座席の PATH や HOME が消えて agent が起動しない → S1 の (a) で PATH と HOME が残るかを先に見る。置き換えなら D2 を見直す
- off で agmsg が placement の記録を書かなくなり、送ったメッセージが受け取る側に届かなくなるかもしれない。見張りの ALERT は `Agmsg.Send` で保存するだけで、ralph は leader を起こさない。leader は次に受信箱を読むときに知る(計画の Codex 助言 HIGH)→ S1 の (d) で、新しい agmsg の状態で off で送った件を受け取る側の `inbox.sh` が読めることを確かめる。試行の 03:39Z 以降の座席で動いた記録も合わせて見る
- pane の名前と shell の子の環境だけを見て、壊れた本体である herdr の agent の名前を確かめないまま F3 を閉じてしまう(計画の Codex 助言 MEDIUM)→ S1 の (e) で、本物の座席が自分の Bash tool から打ったあとも `herdr agent get` で名前が引けることを確かめ、通らなければ F3 は開いたまま
- (e) の claude は本当の HOME で動くので、ユーザーの `~/.claude` の設定と hook を読む → 分離した herdr の中で、scratch の cwd、`--model haiku`、Bash だけの許可で 1 つのコマンドを頼むだけにし、終わったら pane と server を閉じる
- 確かめが一時的に失敗すると、その spawn の pane に off が付かない → 警告を出し、ralph 自身の呼び出しの off は herdr の版によらず効く
- S1 の計測でユーザーの herdr server や Orca に触れる → 計測のコマンドはすべて `env -i` に scratch の `HERDR_SOCKET_PATH` と偽の HOME を渡して打ち、ORCA の変数を渡さない。始める前に socket の path を読み直す

## Rollout or rollback notes

- 下流では、次の release の binary で効く。scaffold のファイルは変えない
- release のノートに書くこと: ralph が打つ agmsg は名前を書き換えない(`AGMSG_SELF_NAME=off`)、`--env` のある herdr では座席の pane にも付く、ない herdr では警告が出る
- 戻すときはこの PR を revert する

## Open questions

- S1 で ALERT の時間切れを再現できるか(できなければ行は開いたまま)
- `--env` の値が Claude Code の hook まで届くか(4b の F11 の前提)。S1 の (e) で Bash tool まで届くかは分かる。hook は 4b で確かめる

## Progress checklist

- [x] Plan reviewed(consult-plan-seatenv: 4 本目を 4a と 4b に分け、案 i の計測と案 x の help の確かめで進めてよい)
- [x] Plan approved(ユーザーの指示「作成したプランは自動承認します」による。Codex の計画助言 2 件(HIGH: 受け取る側に届くかを見ていない、MEDIUM: 本物の座席の agent の名前を見ていない)は、AC6 の (d) と (e)、AC7、S1、Risks に反映した)
- [x] Branch created
- [x] Implementation started
  - S1(計測): `docs/evidence/agmsg-self-name-2026-10-22.md`。(a) `--env` は足すだけで D2 はそのまま。(b)〜(d) は期待どおり。(e) は off で名前が残り、対照の on で `agent_not_found` になったので、F3 は S2〜S4 のあとに RESOLVED にできる。(f) 見張りの環境の上乗せは約 65 ms なので ALERT の行は開いたまま。pane の環境は約 1〜2 s 遅く、F4 に効いていた可能性を 4b に渡す
  - S2 b0a607d9、S4 a34612fd(spawn.go の HerdrClient のコメントの配線の文も直した)、S3 f16423da(CLI テストの herdr の stub が `tab create --help` に答えるよう、`internal/cli/org_test.go` と `internal/cli/org_director_headless_test.go` を範囲に足した)
  - AC7 の範囲を足した(consult-plan-seatenv の判定 (i)): F3 を直すと D-F3 の回り道 3 つが要らなくなるので、F8・F10 の前例のとおり D-F3 を「いいえ(今は)」にし、D-F3 だけが理由だった FR-9 を根拠の表と仕様の両方で `[x]` にする。印の数は仕様が 21/11 から 22/10、根拠の表が 20/11 から 21/10 になる。FR-9 の注には AC3・AC4 のテスト名と evidence の (e) と「残る食い違いはない」を書く
  - 後の PR に残すこと: この PR の番号への書き換え(F3・D-F3・FR-5・FR-9 の注と表の「fix/seat-agmsg-self-name」、仕様の Status と PR 一覧)。tech-debt の F4 の行の「announce の時間切れの原因」は、evidence の (c)・(f)(pane の環境で on が約 1〜2 s 遅い)で 4b が書き換える。F4 の行は F3 の行を "the agmsg self-rename of the row above" と指しており、F3 の行を RESOLVED にしたので、4b で合わせる
  - S5 08d091f5(tech-debt の F3 を RESOLVED、ALERT の行に計測の数字、根拠の表と仕様の F3・D-F3・FR-3・FR-5・FR-9 と #247)。印の数は仕様 22/10、根拠の表 21/10
  - AC8: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は 08d091f5 で rc 0(760 s)。go1.25.8 のテストは各 slice の implementer が driver・org・cli で通した
  - sync-docs に渡すもの: `docs/recipes/director-launch-cmd.md` の 131 行と 248 行(と templates/base の写し)、`/director` skill の起動スクリプトの export の説明(と写し)に `AGMSG_SELF_NAME=off`、`AgmsgSelfNameOff` の doc コメントに起動スクリプト。未確認のこと: 0.7.5 より古い herdr が `tab create --help` を無視するなら、確かめのときに tab を 1 つ作る
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
