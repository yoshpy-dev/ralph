# 分割計画の digest を機能ごとにも持ち、ほかの機能の節の変更で start を断らない(#312 の受け入れ条件 3)

- Status: Approved
- Approved: 2026-11-12 sha256:5a95d18e25ff
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/312(下流の運用で、分割計画のほかの機能の節を直したために digest が変わり、止めた org を `ralph org start --plan` で立て直せなかった。disband すると計画の承認の記録も消えた)
- Related issue: 312
- Type: fix
- Branch: fix/split-feature-digest

## Objective

分割計画から立てた org を、分割計画のほかの機能の節を直したあとでも、disband せずに `ralph org start --plan` で立て直せるようにする。そのために、分割計画の digest を機能ごとにも持つ。機能ごとの digest は、前文(最初の `## ` から `## Features` の前まで)と、その機能の `### <slug>` の節から作る。issue #312 の受け入れ条件 3 をこの PR で満たす。受け入れ条件 1(`usage_limited`)はレーン A の #312a、2(座席の立て直し)は #312c に回す。

## Scope

- `internal/org/split.go`: 機能ごとの digest を計算する(`SplitFeature` に持たせる)。前文と、その機能の slug・Type・Reserve・Depends on・本文から作り、ほかの機能の節、ヘッダの自由な文、`## Features` の最初の `### ` より前の行、`## Features` より後の `## ` の節は入れない
- `internal/org/reserve.go`: `FeatureBinding` に機能ごとの digest の欄を足す。`scope_reserved` の Details に、`approval=` の後ろの新しい token `feature_digest=<12 桁>` として書き、読み戻す。`sameFeature` は、両方の binding がこの欄を持つときはそれで比べ、どちらかが持たないとき(古い記録)は今のまま全体の digest で比べる。`validateFeatureBinding` は新しい欄を検査する。`String()` と、同じ機能で digest だけが違うときの拒否の文に、何が変わったかを書く
- `internal/org/feature.go`: `ralph org start --plan` が binding に機能ごとの digest を入れる
- `internal/cli/org.go`: `ralph org status --json` の `feature` に `feature_digest` を足す(古い記録では出さない)。`start` と `status` の help の文
- `.claude/skills/org/SKILL.md` と 3 つの写し(`.agents/skills/org/`、`templates/base/.claude/skills/org/`、`templates/base/.agents/skills/org/`): 「start が拒否するもの」と、打ち直しの段落、`status` の `--json` のキー
- テスト: `split_test.go`、`reserve_test.go`、`feature_test.go`、`internal/cli` の status のテスト

## Non-goals

- 分割計画の再承認をなくすこと。`- Approved:` の行の digest(全体の digest)と `CheckApproved` は今のまま。ほかの機能の節を直したら、分割計画は承認し直す
- gate.go と regate.go の `sameFeature` の呼び出し側の変更(レーン C の #309 が触る)。呼び出し側は変えず、`sameFeature` の意味の変化だけが及ぶ(D4)
- 古い記録(`feature_digest=` の無い `scope_reserved`)を書き換えて新しい token を足すこと。古い記録の org は今のまま全体の digest で比べるので、ほかの機能の節を直すと断られる(disband して立て直すと、新しい記録になる)
- 座席の立て直しの動詞 `ralph org reseat`、立て直した leader への経過の受け渡し(#312c)
- 利用上限で止まった座席の `usage_limited`(#312a)
- 機能ごとの digest を `scripts/plan-visual.sh` で出すこと。ralph の中だけで計算し、人が書く場所は無い

## Assumptions

- 機能の org の leader が受け取るのは、分割計画の前文、その機能の slug・Type(ブランチ)・Reserve・Depends on・本文と、分割計画の id・パス・全体の digest だけである(`internal/org/feature.go` の `featureLeaderTask`)。ヘッダの自由な文、ほかの機能の節、`## Features` の前置き、`## Features` の後の節は、どの座席にも渡らない(`internal/org/split.go` の先頭の注釈)
- 分割計画は、全体の digest が飛ばす行(`- Status:`・`- Approved:` の 2 行を除く)を読み込みで拒否する(`rejectDigestSkippedLine`)。そのため、全体の digest が同じなら、ファイルの中身は 2 行を除いて同じで、機能ごとの digest も同じになる。機能ごとの digest での比較は、全体の digest での比較より必ず緩い(同じ全体の digest は同じ機能ごとの digest を含む)
- `sameFeature` が真なら、`reservationDecision` は何も記録せずに通す(`record=false`)。これは今の「同じ start の打ち直し」と同じ道で、台帳の binding は最初の start のまま残る
- 古い binary は、binding の token を、知らない鍵の最初の token で読み終え、残りを note として扱う(`reserve.go` の `scopeReservedPathsKey` の注釈)。新しい token を `approval=` の後ろに置けば、古い binary も 5 つの欄を今までどおり読む
- 承認の記録(`plan_approved`)は、org の今の命(最後の `disbanded` より後)にあれば効き、`scope_reserved` の位置には結びつかない(`approval.go` の `currentPlanApprovals`)。disband しなければ、立て直したあとも承認は効く

## Affected areas

- `internal/org/split.go`、`split_test.go`
- `internal/org/reserve.go`、`reserve_test.go`
- `internal/org/feature.go`、`feature_test.go`(ほかに `FeatureBinding{` を手で組むテスト: `spawn_feature_test.go`、`regate_test.go`、`approval_test.go`、`prompts_test.go` など。`validateFeatureBinding` が新しい欄を求めるなら、Spawn に渡す binding を組むテストに欄を足す)
- `internal/cli/org.go`(`orgFeatureJSON`、`start` と `status` の help)と、その status のテスト
- `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`
- 呼び出し側は変えないが意味が変わる: `internal/org/gate.go` の `recordGate`(688 行目)、`internal/org/regate.go` の `regateBranchChanged`(978 行目)

## Visual review

- Page: `.harness/state/plan-visual/split-feature-digest.html`(やさしい図 2 枚: overview、easy-refuse。折りたたみの中に detail-overview、detail-digest、detail-token、detail-compare)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: 全体、overview、ほかの 5 枚をそれぞれ `shot` で撮って読んだ。文字のはみ出し、線と文字の重なり、矢印の外れはなかった

## Design decisions

Critical forks: 下の D1〜D3 は分かれ目だったが、consult(consult-plan-featuredigest、判定「進めてよい」)に聞き、推す案を自律で選んだ(ユーザーの 2026-11-12 の指示「critical forks は /consult で推す案を自分で選ぶ」に従う)。D4〜D6 はコードから決められるので自律で選んだ。D7 は codex の計画助言の指摘への答えで、同じ consult に聞いて自律で選んだ。

- D1 機能ごとの digest は、parse したものから作る。前文(`SplitPlan.Preamble`、前後の空行を落としたもの)、slug、Type(省略時の `feat` を入れたもの)、Reserve(`NormalizeReservePaths` の結果をカンマでつないだもの)、Depends on(カンマでつないだもの)、本文(`SplitFeature.Body`)を、欄の名前と長さを付けて決まった順に SHA-256 にかけ、先頭 12 桁を取る。ファイルの生の行を hash する案は採らない。生の行では節の後ろの空行も入り、最後の機能の後ろに新しい機能を足すだけで最後の機能の digest が変わって、要らない拒否が出るため。parse したものは `featureLeaderTask` が座席に渡す欄と一対一になる。代わりに、`featureLeaderTask` に欄を足したら digest にも足すことを両方の注釈に書き、各欄の変更が digest を変えることをテストで縛る
- D2 `sameFeature` が真で全体の digest だけが違う start は、今の「同じ start の打ち直し」と同じく何も記録しない(`reservationDecision` は `record=false`)。台帳の binding の全体の digest は最初の start のものが残り、leader のタスクの「承認の digest」の行(今の全体の digest)と食い違うが、それを help と skill に書く。新しい `scope_reserved` を足す案は採らない。merge-order(`collectMergeTargets` が `scope_reserved` ごとに `bindingPos` を動かす)と regate(`regateBranchChanged` の `now.pos != was.pos`)の位置の判定が動くため。binding の全体の digest を読むのは `ralph org status` と `String()` だけで、gate・regate・merge-order・approval は比べない(consult が確かめた)
- D3 `validateFeatureBinding` は、新しい欄 `FeatureDigest` を必須にし、12 桁の小文字の 16 進数であることを求める。start が入れ忘れると Spawn が断る。任意にすると、入れ忘れが黙って緩和を無効にするため。手で binding を組んで Spawn に渡すテストには欄を足す
- D4 `Complete()` は今の 5 欄(split・feature・digest・branch・worktree)のままにし、新しい欄を入れない。入れると、token の無い古い記録の org が「不完全」になり、gate(`gate.go` の `gateOrgLife`)、merge-order(`collectMergeTargets`)、approval、`ralph org status` がすべて拒否か不完全の表示に倒れるため
- D5 `sameFeature` は構造体の `==` をやめ、欄ごとに比べる。両方が `FeatureDigest` を持つときは split・feature・branch・worktree・承認方式と `FeatureDigest` を比べ、全体の digest は比べない。どちらかが持たないときは、今までどおり 6 欄(全体の digest を含む)を比べる。片方だけが持つ組は 6 欄で比べるので、古い記録に対しては全体の digest が同じでなければ断る。不完全な binding は今までどおり何とも等しくない。gate.go と regate.go の呼び出し側は変えない
- D7 依存先の機能の節は、依存する側の機能ごとの digest に入れない(codex の計画助言の指摘 1 への答え。consult に聞き、自律で選んだ)。指摘は「B に依存する A は、B の節が契約を変えても立て直せ、古い前提で承認した計画が残る」というもの。入れない理由: leader のタスクに入る依存の情報は slug だけで(`featureLeaderTask` の「- 依存する機能:」の行)、`prompts/leader.md` は依存に触れない。A が使う B の契約は B のブランチと PR で、分割計画の B の節ではない。走っている A は今でも B の節の変更で止まらず、digest を見るのは立て直しの start だけである。依存先を推移的に入れると、上流の文言直し 1 つで下流の org がすべて disband を迫られ、#312 の痛みを依存する側に持ち込む。ユーザーの指示と issue の案も「前文と自分の節」と範囲を決めている。代わりに、機能をまたぐ契約は前文に書くよう org skill の分割計画の節に 1 文足す(前文は全 leader に渡り、digest に入るので、変えれば全機能の立て直しを断る)。残る穴(依存先の節にだけ書いた前提は依存する側に届かない)は PR の Known gaps に書く
- D6 新しい token の鍵は `feature_digest` とし、`approval=` の後ろに書く(`approval=` が無い binding では `branch=` の後ろ)。token が 2 つある記録と値が空の記録は、欄を空として読み、全体の digest の比較に戻す。全体の digest の比較は機能ごとの digest の比較より強いので、壊れた token は断る側に倒れる

## Acceptance criteria

- [ ] AC1 機能ごとの digest は、前文と、その機能の slug・Type(省略時の `feat` を入れたもの)・Reserve(正規化したもの)・Depends on・本文から作る 12 桁の 16 進数である。ほかの機能の節、ヘッダの自由な文(題を含む)、`## Features` の最初の `### ` より前の行、`## Features` より後の `## ` の節、節の前後の空行、行末の CR を変えても変わらない。前文、その機能の本文・Type・Reserve・Depends on のどれかを変えると変わる
- [ ] AC2 `ralph org start --plan` は、leader の `scope_reserved` の Details に `feature_digest=<機能ごとの digest>` を `approval=` の後ろに書く。読み戻すと binding の新しい欄に入る。token が 2 つある記録、値が空の記録は欄を空として読む(全体の digest での比較に戻る)
- [ ] AC3 `sameFeature` は、両方の binding が機能ごとの digest を持つとき、split・feature・branch・worktree・承認方式とその digest が等しければ真を返し、全体の digest は比べない。どちらかが持たないとき(片方だけが持つ組を含む)は、今までどおり 6 つの欄すべてを比べる。壊れた記録(不完全な binding)は今までどおり何とも等しくない。`Complete()` は今の 5 欄のままで、token の無い記録の binding も完全と読む(D4)
- [ ] AC4 disband せずに leader を止めた機能の org を、ほかの機能の節だけを直して承認し直した分割計画で `ralph org start --plan` すると、断られずに leader が立ち、`scope_reserved` は増えない。その org の今の命の `plan_approved` はそのまま効き、per-plan の org で implementer の spawn が通る(issue #312 の受け入れ条件 3)。直したほかの機能が、この機能の Depends on に挙がった機能(依存先)でも同じく通る(D7)
- [ ] AC5 同じ org で、前文かその機能の節を直して承認し直した分割計画の start は、今までどおり断られ、何も記録しない。拒否の文は、その機能の節か前文が org を立てたあとに変わったこと、前と今の機能ごとの digest、disband するか別の `--org-id` を使うことを言う
- [ ] AC6 古い記録(`feature_digest=` の無い `scope_reserved`)の org は、ほかの機能の節だけを直した分割計画の start も今までどおり断る。拒否の文は、その記録が機能ごとの digest を持たない(古い ralph が立てた)ので分割計画のどの変更でも断ること、disband して立て直せば新しい記録になることを言う
- [ ] AC7 分割計画を直して承認し直さずに打った start は、今までどおり `CheckApproved` で断られる
- [ ] AC8 古い binary の読み方: `feature_digest=` の token を足した Details から、古い binary と同じ読み方(知らない鍵で読み終える)で 5 つの欄と承認方式が読めることをテストで確かめる
- [ ] AC9 `ralph org status --json` の `feature` に `feature_digest` が出る。古い記録の org では出ない。`start` と `status` の `--help` が新しい規則を言う
- [ ] AC10 gate.go と regate.go の既存のテストが変更なしで通る(呼び出し側は変えない)
- [ ] AC11 org skill の 4 つの写しが新しい規則を言い、`./scripts/check-skill-sync.sh` が通る。分割計画の節は、機能をまたぐ契約(ある機能が別の機能に約束すること)を前文に書くよう言う(D7)

## Implementation outline

1. S1 機能ごとの digest(`split.go`): `SplitFeature` に `Digest` を足し、`finishSplitPlan` で前文と機能の欄から計算する。テストは AC1 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'TestLoadSplitPlan|TestSplitPlan|TestFeatureDigest|TestPlanDigest' ./internal/org/`
2. S2 binding の欄と比較(`reserve.go`、`feature.go`): `FeatureBinding.FeatureDigest`、token の読み書き、`sameFeature`、`validateFeatureBinding`、`String()`、拒否の文、start が欄を入れる。テストは AC2〜AC8 と AC10 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'Reserv|SameFeature|FeatureBinding|StartFeature|SpawnFeature|Spawn_Feature|SpawnPrecheck|Approval|Gate|Regate' ./internal/org/`
3. S3 CLI と文書(`internal/cli/org.go`、org skill の 4 つの写し): status の JSON、help、skill の文。テストは AC9 と AC11 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -run 'Status|Feature|StartPlan|Help' ./internal/cli/ && ./scripts/check-skill-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、skill の同期、check-sync)
- Spec compliance criteria to confirm: AC1〜AC11。特に、機能ごとの digest が全体の digest より緩い比較にしかならないこと(Assumptions の 2 つ目)、古い記録が今のまま全体の digest で比べられること
- Documentation drift to check: org skill の 4 つの写し、`ralph org start --help`・`ralph org status --help`、`AGENTS.md` の `internal/org/` の行(split.go の説明。触るかは sync-docs で決める。衝突しやすい場所)、`docs/specs/2026-10-07-org-multi-org-director.md` の FR-4 の決定の注記
- Evidence to capture: `scope_reserved` の Details の実例(テストの台帳)、拒否の文の実例

## Test plan

- Unit tests: 機能ごとの digest の表(変えて変わる箇所・変わらない箇所、AC1)。token の書き出しと読み戻し、2 つある・空の token(AC2)。`sameFeature` の表(両方が持つ・片方だけ・どちらも持たない・不完全、AC3)。古い binary の読み方の再現(AC8)
- Integration tests: `StartFeature` を偽の herdr と worktree で回す。止めて立て直す(AC4、承認の記録が効くことを含む)、自分の節と前文の変更での拒否(AC5)、古い記録の拒否(AC6)、承認し直さない start の拒否(AC7)。`spawnPrecheckErr` と Spawn の判定が一致すること(既存の `TestSpawnPrecheckErr_MatchesSpawn` に場合を足す)
- Regression tests: gate・regate・merge-order・approval の既存のテスト(AC10)。`TestStartFeature_OtherBindingRefusedWithoutWorktree`(自分の節の変更は今も断られる)。`feature_digest=` の無い古い記録の binding が `Complete()` 真のまま、gate の life の読み取りと merge-order の対象に入ること(D4)
- Edge cases: 前文の無い分割計画、本文の空の機能、最後の機能の後ろに機能を足す(節の後ろの空行が変わる)、CRLF の分割計画、Depends on の並びを変える(変わる側に倒す)
- Evidence to capture: `go test` の出力、status の JSON の例

## Risks and mitigations

- 機能ごとの digest の作り方が足りず、座席に渡る中身が変わったのに start が通る(知らせが消える向きの穴)。→ 作り方を `featureLeaderTask` が座席に渡す欄と一対一にし、テストで各欄の変更が digest を変えることを確かめる。将来 `featureLeaderTask` に欄を足したら digest にも足す、と注釈に書く
- 台帳の binding が最初の start の全体の digest のままになり、leader のタスクの「承認の digest」と食い違う。→ 意図した形として help と skill に書く(binding の全体の digest は最初の start のもの)。新しい記録を足すと merge-order と regate の位置の判定が動くので、足さない
- `sameFeature` の意味の変化が gate と regate に及ぶ。→ 両方とも、同じ台帳の同じ記録から読んだ 2 つの binding を比べる用途で、`reservationDecision` が同じ機能の再記録を作らないため、実際の比較の結果は変わらない。既存のテストで確かめる(AC10)。レーン C(ralph-d1 経由)に意味の変化を知らせる
- 依存先の機能の節にだけ書いた前提(契約)を変えても、依存する側の立て直しは断られない(D7、codex の計画助言の指摘 1)。→ 機能をまたぐ契約は前文に書くよう skill で言う。PR の Known gaps に書く
- 古い binary の補償(spawn の自己終了の失敗の補償)が binding を書き戻すと、新しい token が落ちる。→ 落ちても全体の digest での比較に戻るだけで、断る側に倒れる。Known gaps に書く
- `validateFeatureBinding` に新しい欄を必須にする(D3)ので、手で binding を組んで Spawn に渡すテストが壊れる。→ 機械的に欄を足す。start が必ず入れることもテストで確かめる
- `sameFeature` の構造体の `==` のまま欄を足すと、新旧の混ざった組がいつも偽になり、片方だけが持つ組の扱いが黙って変わる。→ D5 のとおり欄ごとの比較に書き直し、表のテストで組ごとに確かめる

## Rollout or rollback notes

- 新しい binary が立てた org の記録には `feature_digest=` が入る。古い binary に戻しても、その token は note として読み飛ばされ、全体の digest で比べる今の規則に戻る
- 走っている古い記録の org は、新しい binary でも全体の digest で比べる。ほかの機能の節を直して立て直すには、今までどおり disband が要る。release ノートに書く項目: 「`start --plan` は、その機能の節と前文が変わっていなければ、分割計画のほかの節の変更では断らない(この版より前に立てた org は対象外)」

## Open questions

- なし(分かれ目は Design decisions に書く)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 1 件([MEDIUM] 依存先の機能の節の契約が変わっても、依存する側の機能ごとの digest が変わらず、古い前提で承認した計画のまま立て直せる)。consult に聞き、依存先を digest に入れる案は採らず、D7・AC4・AC11・Risks に答えを書いた(機能をまたぐ契約は前文に書くよう skill で言い、残る穴は Known gaps に書く)。図解ページの「見ておくこと」に 1 行足し、lint と撮り直しで確かめた
- [x] Plan approved — 2026-11-12、ユーザーの方針(計画は自動承認)により承認ゲートを聞かずに通した
- [x] Branch created
- [x] Implementation started — 2026-11-12、base 9ac72c50(#307b の merge のあと)。cycle 1
  - S1 e3eeb643(implementer): `SplitFeature.Digest` と `featureDigest`。検証コマンド ok、変異 5 つがテストで落ちることを確かめた。self-check 5 項目 clean
  - S2 80545503 + 06683a35(implementer): binding の `FeatureDigest`、`feature_digest=` の token、`sameFeature` の欄ごとの比較(`b` にも Complete を求め、比較を対称にした)、`validateFeatureBinding`、`featureDigestRefusal`。範囲に足したファイル: `spawn_feature_test.go`(承認し直した binding の機能ごとの digest も変え、自分の節の変更で断る確かめを保つ)、`verbs_test.go`(補償が token ごと書き戻す期待値)、`internal/cli/org_feature_test.go` と `internal/cli/org_approve_test.go`(`approval=` を最後の token と決め打ちしていた 2 本。2 本目のコミット)。D4 の回帰は `reserve_test.go` の `TestFeatureBinding_OlderRecordStillBound`。検証コマンド ok(628s)、cli の 2 本も ok。self-check: 範囲外のファイルは上の承認済みのものだけ
  - S3 5a749c9b + 34d74a8d(implementer): status `--json` の `feature_digest`、start と status の help、org skill の 4 つの写し(shasum 一致)。2 本目は start の help に「承認方式も同じ」を足した(consult の指摘)。検証コマンド ok、check-skill-sync と check-sync も ok。self-check 5 項目 clean
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
