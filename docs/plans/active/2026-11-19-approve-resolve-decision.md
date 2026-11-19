# approve-resolve-decision

- Status: Approved
- Approved: 2026-11-20 sha256:37ad5d8a83d5
- Owner: Claude Code
- Date: 2026-11-19
- Related request: 2026-11-18〜19 に、per-plan の 3 つの機能の org で、director が `ralph org approve` のあと承認の DECISION(`ralph org send`)を打たずに `ralph org inbox resolve` だけを打ち、leader に承認が届かなかった。1 つの leader は 33 分止まった。issue #340〜#344 の系列にユーザーが足した 1 件で、レーン D が受け持つ(2026-11-19)
- Related issue: #345
- Type: fix
- Branch: fix/approve-resolve-decision

## Objective

承認を記録した計画の件を、leader に DECISION を送らないまま閉じられないようにする。閉じられなければ、件は開いたまま `ralph org watch --all` の ack と resolve の期限にかかり、人に届く。leader の待ち方も 1 つにそろえる。

## Scope

- `ralph org inbox resolve <id>` の検査(`internal/org/` に新しい関数、`internal/cli/org_inbox.go`)
  - 件が `PLAN:` のヘッダのある QUESTION(`ParsePlanHeader`)で、その org の今の命に、この件の `plan_approved` の記録がある(`currentPlanApprovals` の `InboxID` がこの件)とき、その記録より後に、同じ org の leader(`LeaderIdentity`)への `sent` で、dry run でなく、本文が承認の DECISION のものがなければ、何も書かずに断る。照らし方: `sent` の Details は本文を先頭 200 バイトに切るので、計画のパスが長いと承認と直しの依頼、違う digest が同じ記録になる(Codex の計画助言 [HIGH] が 160 バイトと 180 バイトのパスで示した)。そこで `ralph org send` が `sent` に本文全体(paste の印を除いた、書かれたままの文)の sha256 を新しい欄 `text_sha256` として記録し、resolve は `sha256(ApprovalDecisionText(ref))` と一致する `sent` だけを数える。この欄のない古い `sent`(この版より前の ralph が書いたもの)は数えない(`--without-decision` で閉じる)
- `ManifestEvent` に `TextSHA256 string \`json:"text_sha256,omitempty"\`` を足し(`internal/org/manifest.go`)、`Send` の 2 つの `sent` の書き込み(`internal/org/verbs.go`)で埋める。dry run の `sent` には書かなくてよい(数えない)。古い ralph はこの欄を読み飛ばす
- resolve の検査と inbox への書き込みは、approve が承認を記録するのと同じ manifest の lock(`withManifestLock`)の中で行う。lock の順は manifest → inbox とし、inbox の lock を持ったまま manifest の lock を取る経路がないことを確かめる(Codex の計画助言 [MEDIUM]: 別々の lock だと、resolve の検査のあと approve が記録し、resolve が先の結果で閉じる)。approve が先に記録すれば resolve は断り、resolve が先に閉じれば approve は今の `checkResolvedMeanwhile` で「ほかの誰かが resolve した」と言う
  - `submit_unconfirmed=true` の `sent` も送ったと数える。そのときは resolve が stderr に、leader の pane で DECISION が送られたかを `ralph org read` で確かめるよう 1 行出す
  - 断る文に、approve が出したのと同じ `ralph org send` の打ち方(`ApprovalDecisionCommand`、`--state-dir` は `InboxHintStateDir`)と、わざと送らずに閉じる `--without-decision` を出す
  - `--without-decision` を付けると、検査を飛ばして resolve する。stderr に、この ralph が記録した承認の DECISION が leader に見つからなかったので、leader に伝わったかを確かめるよう 1 行出す(古い ralph で送った DECISION は数えないので、「届いていない」とは言わない)。resolve の note は approve が出した `approved <path> sha256:<digest>` のままにする(起こされた leader がその note を承認として読む)
  - `PLAN:` のある QUESTION でない件は、manifest の lock を取らずに今までどおり resolve する(self-review の L1)
  - それ以外の件(`PLAN:` のない件、承認を記録していない計画の件 = 直しの依頼や取り下げ)は今のまま。resolve 済みと知らない件のエラーも今のまま
- `ralph org approve` の help と出力の文(`internal/cli/org_approve.go`)と、`internal/org/approve.go` のコメント: 「DECISION を送らなかった承認でも leader が黙って待ち続けることはない」を、resolve が断るので成り立つ形に書き直す
- leader の role prompt(`internal/org/prompts/leader.md` の手順 3 と 4、受信箱の項): 承認の QUESTION を上げたあとは DECISION を待つ、を保つ。待つ間に起こされたら(stall の ALERT、Monitor の張り直し、`ralph org resume` の文、ほかのメッセージ)、`ralph org inbox show <件の id>` で件を読む。resolve 済みで note が `approved <同じ PLAN>`(`ApprovalResolveCommand` の note)なら、承認として手順 4 に進む。件がまだ open か acked なら待ち続け、台帳(`manifest.jsonl`)を直接読んで進まない。台帳に承認がなければ、implementer の spawn と TASK は ralph が断るので、手順 7 のとおり止まる
- `/director` skill の 4 写し(「計画の承認」の 3 と 4): resolve は DECISION を送ってからでないと断られること、`--without-decision` は数えられる DECISION なしでわざと閉じるとき(別の方法で知らせたとき、古い ralph で送ったとき)だけ使うこと
- 文書: README の `ralph org` の行の approve の文、org skill の「機能の計画の承認」に resolve の断りがあれば同じ文
- テスト

## Non-goals

- `ralph org approve` が DECISION を送る(`--send`)こと。approve は記録だけのまま(D7)
- watch に「承認のあと DECISION が来ない」の ALERT を足すこと(Design decisions の 3)
- 計画の承認以外の件(cross-review の上限、codex の hook の QUESTION、BLOCKED)の resolve
- 次の release の注記(`/release` で書く。候補は A に送る)

## Assumptions

- `sent` の記録は OrgID、SeatID(宛先)、DryRun、Details(`raw=true`、`via=screen ...`、`submit_unconfirmed=true` の印のあとに本文の先頭 200 バイト)を持つ(`internal/org/verbs.go:484`、`:671`、`truncateForDetails`)。Details だけでは長い本文を区別できないので、`text_sha256` を足す
- approve は manifest の lock の中で承認を記録する(`internal/org/approve.go:164` の `withManifestLock`)。inbox は自分の lock(`InboxStore.withLock`)を持つ
- `currentPlanApprovals`(`internal/org/approval.go:376`)は org の今の命(最後の disband のあと)の承認を返す。disband の前の承認は数えない
- inbox の runtime(`newOrgInboxRuntime`)は `*org.Org` で、manifest も読める(`rt.Approve` が使っている)
- per-plan の org では、implementer の spawn と TASK を ralph が台帳の承認で確かめる(`checkTaskPlanApproved`)。leader が承認を誤って読んでも、承認のない計画で implementer は動かない
- 待っている leader は起こされる: watch は leader 自身の stall の ALERT も leader に送ってから受信箱に上げる(`internal/org/watch.go` の `sendAlert`、`SendWatchdogAlert`)。Claude Code の leader の Monitor は 25 分で終わって張り直す(`internal/org/prompts/leader.md` の「agmsg の受信箱の運用」)。issue の entry-ios の leader も stall の ALERT のあとに動いた(確かめ済み)

## Affected areas

- `internal/org/`(新しい検査の関数、`manifest.go` の `TextSHA256`、`verbs.go` の `sent` の書き込み、`approve.go` のコメント)、`internal/cli/org_inbox.go`、`internal/cli/org_approve.go`
- `internal/org/prompts/leader.md`
- `.claude/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md`(後ろの 2 つは `scripts/sync-skills.sh` で作る)
- `README.md`(`ralph org` の行)、場合により `.claude/skills/org/SKILL.md` の 4 写し
- テスト: `internal/org/` と `internal/cli/` の新しいテスト、prompt と skill の文を固めるテスト
- 共有ファイル: README.md、director skill の 4 写し(B の #342 が「merge と auto_merge」の節を触る。節は別)、org skill の 4 写し(触るなら)

## Visual review

- ページ: `.harness/state/plan-visual/approve-resolve-decision.html`(やさしい図 2 枚: overview、easy-leader。折りたたみに詳細図 2 枚: detail-overview、detail-resolve)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: 全体と 4 枚の図を `shot` で撮って読んだ。詳細 1 の左の箱の幅を 1 度直した。ほかに重なりやはみ出しはない

## Design decisions

1. 断る場所は `ralph org inbox resolve` にする(自律で選んだ。A の推す案)。approve に `--send` を足す案は、approve が記録だけという D7 の形を崩し、herdr の pane の中から打つと断られる approve が send まで抱えることになる。resolve は安全網(件が開いていれば期限が人に届く)を外す唯一の操作なので、そこで止めるのが小さい
2. leader は DECISION を待つ形を保ち、起こされたときに件の resolve の note で承認を読めるようにする(自律で選んだ)。leader が台帳を直接読む読み出しの動詞はないので、すでに leader が使う `ralph org inbox show` で読む。note を偽っても、implementer の spawn と TASK は台帳の承認で確かめられるので、承認のない計画で implementer は動かない。台帳の承認を副作用なしに読む手段として `ralph org spawn --dry-run` もある(承認の検査まで同じ順で走り、herdr を触らない)が、断りの文から承認の有無を読み分ける必要があり、note の方が読みやすいので採らない。件が open か acked のまま台帳を直接読んで進む形(issue の dev-redeploy-m4 と entry-ios)は、leader ごとのばらつきの元なので prompt で止める
3. watch の ALERT は足さない(自律で選んだ)。1 で resolve が断るので、DECISION を送らない承認の件は開いたままになり、今の ack と resolve の期限が人に届ける。`--without-decision` で閉じたときは、人がわざと別の方法で知らせると決めたときなので、ALERT で重ねて知らせない

4. 送ったかの証拠は、`sent` に足す本文全体の sha256 にする(自律で選んだ。Codex の計画助言 [HIGH] による)。Details の照らし合わせは、本文を 200 バイトで切るので長いパスで区別がつかない。DECISION だけに構造化した欄(決定の種類と PLAN)を足す案もあるが、本文の hash は TYPE によらず同じ書き方で済み、送った文そのものを確かめられる。古い `sent` を数えないので、版をまたいだ承認の件は `--without-decision` で閉じる
5. resolve の検査と書き込みを manifest の lock の中に置く(自律で選んだ。Codex の計画助言 [MEDIUM] による)。approve の記録と resolve が同じ lock で順に並ぶので、どちらが先でも DECISION のない承認の件は閉じない

Critical forks: None(上の 1〜3 は /consult の consult-plan-approveresolve に聞き、判定「直してから進める」の指摘(照らし方を本文全体に、leader が起こされる経路の事実、`submit_unconfirmed` の扱い)を入れて自律で選んだ。4 と 5 は Codex の計画助言を受けて足した。判断の材料は repo の中にある)

## Acceptance criteria

- [ ] AC-1: 承認を記録した計画の件を、leader への承認の DECISION の `sent` がないまま `ralph org inbox resolve` で閉じると、終了コード 1 で断られ、件は開いたまま(open か acked)。エラーに `ApprovalDecisionCommand` の打ち方と `--without-decision` が出る
- [ ] AC-2: 承認の DECISION を leader に送ったあと(`sent` が承認の記録より後にある)なら、resolve は今までどおり通る
- [ ] AC-3: `--without-decision` を付けると、DECISION がなくても resolve が通り、stderr に、この ralph が記録した承認の DECISION が leader に見つからなかったので、leader に伝わったかを確かめるよう 1 行出る
- [ ] AC-4: 数えない `sent`: dry run、承認の記録より前、leader 以外の座席、別の org、別の `PLAN:` の DECISION、同じ `PLAN:` でも本文が `approved` でない DECISION(直しの依頼)、DECISION でない TYPE、`text_sha256` のない古い `sent`。これらしかなければ断る。計画のパスが 160 バイトと 180 バイトのとき、直しの依頼と違う digest の DECISION で断られる。`submit_unconfirmed=true` の承認の DECISION は数え、stderr に `ralph org read` で確かめる 1 行が出る
- [ ] AC-9: `ralph org send` の `sent`(dry run でないもの)が、本文全体の sha256 を `text_sha256` に持つ
- [ ] AC-10: resolve の検査と書き込みは manifest の lock の中で行う。resolve が検査を終えたところで止めて approve を走らせるテストで、approve は resolve が終わるまで記録できず、終わったあと「ほかの誰かが resolve した」と言う。逆の順(approve が先)では resolve が断る
- [ ] AC-5: 今のままの件: `PLAN:` のない件、承認を記録していない計画の件(直しの依頼のあとの resolve)、前の命(disband の前)の承認しかない件。どれも検査なしで resolve できる
- [ ] AC-6: leader の role prompt が、承認を待つ間に起こされたら `ralph org inbox show` で件を読み、resolve 済みで note が `approved <同じ PLAN>` なら承認として進み、件が open か acked なら待ち続けて台帳を直接読んで進まない、と書く。prompt のテストが文を固める
- [ ] AC-7: `/director` skill の 4 写しの「計画の承認」と `ralph org approve --help` と `ralph org inbox resolve --help` が、resolve の断りと `--without-decision` を書く。`scripts/check-skill-sync.sh` が通る
- [ ] AC-8: 送らずに閉じる場合(断られる、flag で通る)と送ってから閉じる場合(通る)を、`internal/org` と CLI の両方のテストで確かめる

## Implementation outline

1. `sent` の `text_sha256`、resolve の検査と `--without-decision`、manifest の lock、approve と resolve の help(`internal/org/manifest.go`、`internal/org/verbs.go`、`internal/org/approve_resolve.go`(新規、仮名)とテスト、`internal/cli/org_inbox.go`、`internal/cli/org_approve.go`、`internal/org/approve.go` のコメント、CLI のテスト) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Approv|Inbox|Resolve|Send|Manifest' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Approv|Inbox|Resolve|Send'`
2. leader の role prompt と director skill(`internal/org/prompts/leader.md`、director skill の 4 写し、prompt のテスト) — verify: `./scripts/sync-skills.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Prompt|RenderRolePrompt'`
3. 文書(`README.md`、org skill の写し(要れば)) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC-1〜AC-8 を issue #345 の受け入れ条件 3 つと照らす
- Documentation drift to check: `ralph org approve --help`、`ralph org inbox resolve --help`、director skill の 4 写し、leader の prompt、README、`docs/specs/2026-10-07-org-multi-org-director.md` の D7 の記述
- Evidence to capture: verify の報告

## Test plan

- Unit tests(`internal/org`): 断る(sent なし)、通る(承認の後の DECISION)、flag で通る、AC-4 の数えない `sent` 8 種と `submit_unconfirmed`、AC-5 の今のままの件 3 種、160 バイトと 180 バイトの計画のパスで、承認は通り、直しの依頼と違う digest は断られる 4 件、`sent` の `text_sha256`(AC-9)、lock の順のテスト(AC-10、検査のあとで止める差し込み口を使う)
- Integration tests(`internal/cli`): `ralph org inbox resolve` の終了コード、エラーの文(send の打ち方と flag)、flag の stderr、`--state-dir` を付けたときの打ち方
- Regression tests: 今の inbox と approve のテスト、prompt と skill の文を固めるテスト
- Edge cases: acked の件、件の org が disband 済み、同じ件に承認が 2 つ(打ち直し)
- Evidence to capture: test の報告

## Risks and mitigations

- 人が直しを求めた件(承認なし)まで断る: 承認の記録がある件だけを見る(AC-5)
- DECISION を送ったのに断る: `sent` の `text_sha256`(本文全体の sha256)を承認の DECISION の sha256 と照らすので、approve が出したとおりに送れば数えられる。DECISION を手で書き換えて送ったときと、この版より前の ralph で送ったときは断られるので、エラーに approve の打ち方と `--without-decision` を出す
- leader が note を誤って読み、承認のない計画で進む: implementer の spawn と TASK は台帳の承認で確かめられる
- 共有ファイル(README、director skill の 4 写し)の衝突: merge の直前に最新の main を取り込み、A に一言送る
- lock の順の取り違えで止まる(deadlock): resolve は manifest → inbox の順に取る。inbox の lock を持ったまま manifest の lock を取る経路がないことを slice 1 で grep して確かめ、報告に書く
- manifest の欄を足すことで古い ralph が壊れる: `omitempty` の欄で、Go の JSON は知らない欄を読み飛ばす。manifest の行を丸ごと比べるテストがあれば、slice 1 で直す
- 版をまたいだ承認の件(古い ralph で DECISION を送り、新しい ralph で resolve)は断られる: エラーに `--without-decision` を出し、release ノートの候補に書く

## Rollout or rollback notes

- resolve の挙動が変わる(承認を記録した計画の件は、DECISION を送るまで閉じられない)。release ノートの候補にし、A に送る。戻すときはこの PR を revert する

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh、rc=0、git の状態は前後で同じ)。所見 2 件。[HIGH] `sent` の Details は本文を 200 バイトで切るので、長いパスでは承認と直しの依頼、違う digest が同じ記録になる → `sent` に本文全体の `text_sha256` を足し、それで照らす形に直した(Scope、Design decisions 4、AC-4、AC-9、Test plan)。[MEDIUM] approve は manifest の lock、resolve は inbox の lock で、間に approve が記録すると DECISION なしで閉じる → resolve の検査と書き込みを manifest の lock の中に置く形に直した(Scope、Design decisions 5、AC-10、Risks)。どちらもユーザーの決まり(計画助言の修正提案は聞かずに直す)で反映し、図解ページの詳細 1 と 2 を描き直した
- [x] Plan approved — ユーザーの決まり(計画は推す案で自動承認)により 2026-11-19 に承認。2026-11-20 に、self-review の直し(42524d6e)に合わせて計画本文を直し(Scope の stderr の文と L1、director skill の項、AC-3、Risks の照らし方)、図解ページの詳細 2 の 1 か所を描き直して(lint exit 0、shot を読んだ)、同じ決まりで承認し直した
- [x] Branch created
- [x] Implementation started
  - slice 1(implementer、47257bf2): `sent` の `text_sha256`、`ResolveInboxItem`(manifest の lock の中で検査と書き込み)、resolve の `--without-decision`、approve と resolve の help。verify(gofmt、vet、`go test ./internal/org/ -run 'Approv|Inbox|Resolve|Send|Manifest'`、`go test ./internal/cli/ -run 'Approv|Inbox|Resolve|Send'`)が pass。self-check はテストのコメント 1 か所を fixed、ほかは clean。lock の順は、inbox の lock を持ったまま manifest の lock を取る経路がないことを grep で確かめた。既存の `org_approve_test.go` の 3 本を新しい検査に合わせた(dry-run の send だけで resolve していたものなど)
  - main の取り込み(f660f745): #347 の org skill の変更を先に入れた。build と vet が pass
  - slice 2(implementer、872775b6): leader の prompt の手順 3・4 と立て直しの受信箱の項、director skill の 4 写しの「計画の承認」の 3・4。verify(sync-skills、check-skill-sync、check-sync、`go test ./internal/org/ -run 'Prompt|RenderRolePrompt'`)と、director skill のテストを含む `-run 'Skill|Director|Leader|Template|Embed'` が pass。self-check 5 項目 clean
  - slice 3(implementer、86f72048): README の 2 文、org skill の 4 写しの 3 か所、仕様の 8a-1 に日付つきの追記。verify(sync-skills、check-skill-sync、check-sync、check-template-purity、`go test ./internal/org/ -run 'Skill|OrgSkill'`)が pass。self-check 5 項目 clean
  - self-review の 1 回目(3df4ad1a: MEDIUM 1、LOW 9)の直し(implementer、42524d6e: M1、L1〜L3、L5〜L8)と計画の L9(a908b2ec、承認し直し)。2 回目(d35730a7: LOW 4)の N1〜N3 の直し(implementer、ae5b028f、skill の追従 f8bcd876)。L4(規則の写し)は follow-up に残した
  - verify の 1 回目は ST1005(断りの文の末尾の句点)で fail → f3c8e4b2 で直した。main 32274500 を取り込み(a8e0e72f、org skill の表の approve と gate の行の衝突を両側から取って解いた)、2 回目の verify は pass
  - test の G1〜G4(implementer、4c4edcd7): lock の中の書き込み、別の件の承認、resolve 済みの件、最初の読みで見つからない ID のテスト。変異で落ちることを確かめた。通しの run-test.sh は e42ee962 の前の b7a9552e で打った(4c4edcd7 はテストを足しただけで、絞った `go test -race` で pass)
- [x] Review artifact created — docs/reports/self-review-2026-11-19-approve-resolve-decision.md(2 回目: LOW 4、Merge yes)
- [x] Verification artifact created — docs/reports/verify-2026-11-19-approve-resolve-decision.md(2 回目 pass)
- [x] Test artifact created — docs/reports/test-2026-11-19-approve-resolve-decision.md(pass、run-test.sh rc 0)
- [ ] PR created
