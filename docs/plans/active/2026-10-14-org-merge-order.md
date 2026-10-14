# org-merge-order

- Status: Approved
- Approved: 2026-10-14 sha256:c0b1e9230999
- Owner: Claude Code
- Date: 2026-10-14
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 7 段目の残り(7c)のうち、FR-8 の「merge してよい順」(検証した base が今の main と一致する PR)。7b(`ralph org gate`)は PR #225(b20e70df)、CI の Go を go.mod に合わせる PR #226(ac34ccf3)は merge 済み。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-merge-order

## Objective

機能の org が出した PR のうち、どれを今 merge してよいかを、台帳の gate の記録とリモートの今の状態から決めて出す動詞 `ralph org merge-order` を足す。merge してよいのは、最後の gate が通っていて、gate が検証した head がリモートのブランチの先頭と同じで、検証した base がリモートの既定のブランチの今の先頭と同じ PR である。根拠は台帳の `gate_passed` だけにし、reviewer や leader の申告は読まない。

7c の残り(`ralph doctor` の保護ブランチの warn、`ralph status` の merge の時点の保証の表示)は 7c-2(次の PR)に回す(Design decisions の C1)。

## Scope

- 新しい動詞 `ralph org merge-order`(`--state-dir`、`--json`、`--all`)
  - 読むだけの動詞で、manifest にも台帳のほかのファイルにも何も書かない。ロックは取らない。6b の見張りの関門と 7a の承認の検査はかけない。台帳の読み方の守り(`guardLegacyOrgStateDir` の読み取り)は `ralph status` と同じ
  - 単位はブランチ(C3)。対象は、台帳の real(dry-run でない)な `scope_reserved` のうち完全な結びつきを持つもの全部(disband 済みの org の分も含む)のブランチと、`gate_passed` / `gate_failed` の記録のブランチ。ブランチごとに、結びつきの最新のもの(org_id・feature・split)を補足として出す
  - ブランチごとの gate の記録(C8): org 全体の real な `gate_passed` / `gate_failed` を manifest の順に読む。読める記録(`gateRecordFromDetails` が ok で、event の名前と reason の有無が合う)は `branch=` のブランチに付ける。読めない記録は、その org_id の、その記録より前の最新の結びつきのブランチに付ける(gate は結びつきのブランチでしか打てないので、打ったブランチはそれに決まる)。前に結びつきがなければどのブランチにも付けず、注記で数を言う。ブランチの最新の記録が読めないものなら、そのブランチは `unreadable`
  - manifest の壊れた行(JSON として読めない行): merge-order は manifest を自分で行ごとに読み、壊れた行の位置を知る。ブランチの最後に読めた gate の記録より後に壊れた行があれば、その行が新しい gate の記録かもしれないので、そのブランチは `unreadable` にする(C8)。壊れた行の数は注記で言う
  - リモートの今の状態: 打った場所の repo で `git ls-remote --symref origin HEAD 'refs/heads/*'` を 1 回打つ(C2、C7)。`HEAD` の symref(`ref: refs/heads/<名前>\tHEAD`)から origin の今の既定のブランチを、`refs/heads/*` から各ブランチと既定のブランチの先頭を読む。打てない(origin がない、ネットワーク、認証、30 秒の timeout)とき、または `HEAD` の symref が出ない(既定のブランチが分からない)ときは、推測せずに理由を言って終了コード 1 で終わる
  - PR の状態(任意): `gh` があれば、ブランチごとに `gh pr list --head <ブランチ> --state all --json number,state,url,headRefOid,baseRefName,isCrossRepository` を打ち、`isCrossRepository` が false の PR のうち番号が最大のものを採る。`gh` がない、または打てない(未認証、ネットワーク、15 秒の timeout)ときは PR の欄を空にし、merged と closed と PR の統合先を見分けないと注記する。`gh` の失敗では終了しない
  - 分類(C4、C7、C8)。上から順に最初に当てはまるもの
    - `merged` / `closed`: `gh` で分かった PR の state が MERGED / CLOSED。既定では出さず、`--all` で出す
    - `unreadable`: ブランチの最新の gate の記録が読めない、または最後に読めた記録より後に manifest の壊れた行がある。前の記録が通っていても `ready` にしない
    - `no_gate`: そのブランチの gate の記録がない
    - `failed`: 最新の gate の記録が `gate_failed`(reason を出す)
    - `no_remote`: リモートにそのブランチがない(push していない、または merge のあとに消された。`gh` がなければ見分けない)
    - `regate`: リモートのブランチの先頭が、gate が検証した head と違う(gate のあとに push した)
    - `wrong_base`: `gh` で分かった PR の統合先(`baseRefName`)が origin の今の既定のブランチと違う(PR の統合先を直す)
    - `rebase`: head は同じだが、gate の `base_ref` の名前が origin の今の既定のブランチと違う(既定のブランチを変えた)、または既定のブランチの今の先頭が gate の base と違う(gate のあとに main が進んだ)
    - `ready`: head が同じで、gate の base_ref の名前と base が origin の今の既定のブランチとその先頭に合い、PR が分かるならその統合先も同じ
  - 出す順: `ready`(gate の記録の時刻が古い順)、`rebase`、`wrong_base`、`regate`、`unreadable`、`failed`、`no_remote`、`no_gate`、(`--all` のとき)`merged`、`closed`。同じ分類の中は gate の記録の時刻、gate のないものは結びつきの記録の順
  - 出力(テキスト): 先頭に origin の今の既定のブランチとその先頭(`origin/main <12 桁>`、ls-remote で読んだ時刻)、1 行に 1 ブランチ(順位・分類・ブランチ・PR の番号と URL・gate の head と base の 12 桁・org_id と feature)。最後に注記を出す: `ready` を 1 本 merge すると main が進み、ほかの `ready` は `rebase` になる(rebase と gate の打ち直しが要る)こと、`gh` がなければ merged と closed を見分けないこと
  - `--json`: 既定のブランチ、ls-remote の時刻、ブランチごとの分類・PR・gate の記録(event、head、base、base_ref、reason、ts)・リモートの先頭・結びつきの補足、注記の配列
  - 終了コード: 分類によらず 0。manifest が読めない、ls-remote が打てないときは 1
  - 外のコマンド(`git ls-remote` と `gh`)は、打つ場所(dir)を受け取る小さな seam を通して打つ(テストは偽物に替える)。`git` には `GIT_TERMINAL_PROMPT=0` を渡す
- 文書: `/org` skill の 4 面(動詞の表に `merge-order`、機能ごとの org の締めのあとに人が使うこと)、README の org の節、仕様 FR-8 の「7c-1 で決めたこと」の注、AGENTS.md の repo map(新しいファイル)、`docs/tech-debt/README.md`

## Non-goals

- 7c-2: `ralph doctor` の保護ブランチの warn、`ralph status` の merge の時点の保証の表示、merge-order の出力に保証の種類を出すこと
- 8a 段目: director が merge してよい順を報告すること、`[org.director].auto_merge` と merge すること
- merge-order が fetch・rebase・gate の打ち直し・merge をすること
- gate を打たずに出された PR、結びつきのないブランチの PR を順に入れること(gate の記録がなければ `no_gate`、結びつきも gate の記録もないブランチは対象にしない)
- reviewer の RESULT の `HEAD:`・`BASE:` を読むこと(7b の B2)
- 古い対象を期限で外すこと(`--since` など)。`gh` がない環境では merge 済みのブランチが `no_remote` として残る(tech-debt に書く)

## Assumptions

- 機能の org は、leader の雛形の手順 12〜16(gate → 座席の stop → push → `gh pr create` → disband)で終わる。PR が出たときには org は disband されているので、`ActiveFeature`(最後の disbanded より後の命だけを見る)では引けない。merge-order は disband 済みの命の結びつきも読む
- 同じ org_id は別の機能で立て直されうる。単位をブランチにすれば、前の機能の PR も落ちない(C3)
- gate の記録の `branch=` は結びつきのブランチと同じ(gate は結びつきのブランチでしか打てない)
- 機能のブランチは origin に push される(雛形の手順 14)。GitHub の「merge のあとにブランチを消す」設定があると、merge 済みのブランチはリモートから消える(この repo は消える設定)
- `git ls-remote --symref origin HEAD 'refs/heads/*'` の出力は、`ref: refs/heads/<名前>\tHEAD`(symref)、`<SHA>\tHEAD`、`<SHA>\trefs/heads/<名前>` の行。パターンは ref の末尾に合うので、`HEAD` は `refs/heads/HEAD` のような名前にも合いうる。読むのは名前がちょうど `HEAD` の行と `refs/heads/` の行だけにする。GitHub(protocol v2)は `HEAD` の symref を返す

## Affected areas

- 新しいファイル: `internal/org/mergeorder.go`(対象の集め方、分類、順、seam)、`internal/org/mergeorder_test.go`、`internal/cli/org_merge_order.go`(CLI)、`internal/cli/org_merge_order_test.go`
- 変えるファイル: `internal/cli/org.go`(サブコマンドの登録)、`/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面)、`README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 の注)、`AGENTS.md`、`docs/tech-debt/README.md`
- 読むだけ: `internal/org/gate.go`(`gateRecordFromDetails`、`EventGatePassed`・`EventGateFailed`)、`internal/org/reserve.go`(`reservationFromEvent`、`FeatureBinding`)

## Visual review

- Page: `.harness/state/plan-visual/org-merge-order.html`
- Self-check: ページ全体と全体図(`--fragment overview`、1150×560)を撮って読んだ。文字の重なり、はみ出し、線が文字を横切る箇所、的を外した矢印はない。図は 4 つ(全体図、分類の決め方、1 本 merge したあとの順、単位はブランチ)。Codex の助言(C7・C8)を入れたあと、全体図と図 2(`--fragment detail-1`、1150×620)を描き直し、ページ全体・全体図・図 2 を撮り直して読んだ。同じ問題はない

## Design decisions

- C1 7c を 2 つの PR に分ける。7c-1(この計画)は `ralph org merge-order`、7c-2 は doctor の保護ブランチの warn、doctor が台帳の下に書く保証のキャッシュ、`ralph status` の保証の行、merge-order の出力の先頭の保証の行。理由は 7b の B1 と同じで、どちらも外のコマンド(`git ls-remote`・`gh`)の seam が要り、1 本だと上限 2 の cross-review で詰まったときに戻しにくい。consult(consult-plan-stage7c)の判定。退けた案: 1 本にまとめる(FR-7 の残りと FR-8 の順が一度に閉じるが、大きい)
- C2 「今の main」と各ブランチの先頭は `git ls-remote` で origin から読む。仕様の「検証した base が今の main と一致する」の字義に合い、gate は fetch しない(`internal/org/gate.go:30`)ので、ローカルの `origin/<既定>` で比べると古い main で `ready` と出しうる。打てなければ推測せずに終わる。退けた案: ローカルの ref(人が先に fetch する決まりに頼る)
- C3 単位はブランチ(PR の単位)。結びつき(disband 済みを含む全 `scope_reserved`)はブランチごとの補足にする。org の最後の結びつきを単位にすると、同じ org_id を別の機能で立て直したときに前の PR が順から消える(`currentOrgLives` は最後の disbanded より後しか見ない)。consult の判定
- C4 リモートにブランチがないものは `regate` に入れず、`no_remote` として分ける。merge のあとに消されたブランチと push していないブランチは、`gh` がなければ見分けられない(squash merge では先頭が main の祖先にならず、ls-remote では祖先も分からない)ので、`gh` があるときだけ merged と closed に振り分ける。consult の判定
- C5 `merged` / `closed` は `gh` の PR の state だけで決める(7b の consult の助言)。同じブランチの PR が複数あるときは、同じ repo の PR(`isCrossRepository` が false)のうち番号が最大のものを採る。fork の同じ名前のブランチの PR を拾わないため
- C6 外のコマンドは dir を受け取る seam で打つ。`driver.Runner`(`internal/org/driver/driver.go:26`)は打つ場所を持たないので、`gh` が打った場所の repo を読むように、dir を渡せる形にする(git は `-C` でも足りるが、2 つを同じ seam にする)

- C7 「今の main」は origin の `HEAD` の symref が指すブランチとし、gate の記録の `base_ref` の名前だけで決めない。既定のブランチを変えたあとに古いブランチが残っていると、名前で引けば古い base との一致で `ready` と出しうる。gate の base_ref の名前が今の既定と違えば `rebase`、`gh` で分かる PR の統合先が違えば `wrong_base` にする。既定のブランチが分からなければ推測せずに終わる。Codex の計画助言 #1(HIGH)
- C8 読めない証拠があれば `ready` にしない。読めない gate の記録を数えずに飛ばすと、通ったあとに落ちた記録が壊れていたとき、前の `gate_passed` に戻って `ready` と出る。読めない記録は、打った org の、その時点の結びつきのブランチに付ける(gate は結びつきのブランチでしか打てない)。`ManifestStore.Read` は JSON として読めない行を数えて飛ばすだけで位置を返さないので、merge-order は行ごとに読み、最後に読めた gate の記録より後に壊れた行があるブランチを `unreadable` にする。退けた案: 壊れた行が 1 つでもあれば全体を終了コード 1 にする(古い壊れた行が残る台帳では、ずっと使えなくなる)。Codex の計画助言 #2(HIGH)

Critical forks: C1〜C4 は consult の判定で決めた(ユーザーの指示「計画の分かれ目も自分で選ぶ」)。C7 と C8 は Codex の計画助言をユーザーの指示どおりに取り込んだ。

## Acceptance criteria

- [x] AC1 gate の記録が `gate_passed` で、gate の head がリモートのブランチの先頭と同じ、gate の base がリモートの既定のブランチの先頭と同じブランチは `ready` になる。gate のあとに main が進んだブランチは `rebase`、gate のあとに push したブランチは `regate` になる
- [x] AC2 最新の gate の記録が `gate_failed` のブランチは `failed`(reason つき)、gate の記録がない結びつきのブランチは `no_gate`、リモートにないブランチは `no_remote` になる。同じブランチで通ったあとに落ちたら、最新の記録(落ちたほう)を読む
- [x] AC3 disband 済みの org の結びつきのブランチも対象になる。同じ org_id を別の機能で立て直しても、前のブランチが対象から消えない
- [x] AC4 dry-run の記録と完全でない結びつきは数えない。`gate_passed` のあとの最新の gate の記録が読めない(`gateRecordFromDetails` が ok でない、event の名前と reason の有無が合わない)ブランチ、`gate_passed` のあとに manifest の壊れた行があるブランチは、前の `gate_passed` に戻らず `unreadable` になる。読めない記録は、打った org のその時点の結びつきのブランチに付く
- [x] AC5 `gh` があれば、PR の state が MERGED / CLOSED のブランチは既定で出さず、`--all` で `merged` / `closed` として出す。PR が複数あれば、同じ repo の番号が最大の PR を採る。`gh` がない、または打てないときは PR の欄を空にし、merged と closed と PR の統合先を見分けないことを注記し、終了コード 0 で終わる
- [x] AC6 出す順は `ready`(gate の時刻が古い順)、`rebase`、`wrong_base`、`regate`、`unreadable`、`failed`、`no_remote`、`no_gate`、(`--all`)`merged`、`closed`。テキストの出力は、`ready` を 1 本 merge するとほかの `ready` が `rebase` になることを注記する
- [x] AC7 `git ls-remote` が打てない(origin がない、失敗、timeout)とき、origin の `HEAD` の symref が出ないときは、何も分類せず、理由を言って終了コード 1。manifest が読めないときも終了コード 1
- [x] AC11 origin の既定のブランチを変えた(`HEAD` が `trunk` を指し、古い `main` も残る)とき、`main` で gate を通したブランチは、`main` の先頭と base が同じでも `rebase` になる。`gh` で分かる PR の統合先が今の既定のブランチと違うブランチは `wrong_base` になる
- [x] AC8 `--json` は分類・PR・gate の記録・リモートの先頭・結びつきの補足・注記を出し、テキストと同じ順に並べる
- [x] AC9 merge-order は台帳に何も書かない(manifest の行数とバイト、台帳のディレクトリの中身が打つ前と同じ)。ロックを取らず、見張りがなくても打てる
- [x] AC10 `/org` skill の 4 面、README、仕様 FR-8 の注、AGENTS.md、tech-debt が merge-order を説明する。skill の 4 面は同じ中身(`scripts/check-skill-sync.sh`、`scripts/check-sync.sh`)

## Implementation outline

1. org の層(`internal/org/mergeorder.go`): 対象のブランチの集め方(結びつきと gate の記録)、ls-remote の出力の読み方、分類と順、`gh` の PR の選び方。外のコマンドは seam を通す。テストは偽の seam で、分類ごと・順・PR の選び方・失敗の扱いを固める
2. CLI(`internal/cli/org_merge_order.go`): `--state-dir`・`--json`・`--all`、テキストと JSON の出力、終了コード。`internal/cli/org.go` に登録する。テストは本物の git(bare の origin を一時ディレクトリに作り、ls-remote はローカルのパスを読む)と偽の `gh`
3. 文書: `/org` skill の 4 面、README、仕様 FR-8 の注、AGENTS.md、tech-debt

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`gofmt -l`、`go vet ./...`、`scripts/check-skill-sync.sh`、`scripts/check-sync.sh`。Go のテストは go.mod の版(`GOTOOLCHAIN=go1.25.8`)でも build できること
- Spec compliance criteria to confirm: FR-8 の「merge してよい順(検証した base が今の main と一致する PR)」を AC1〜AC9 と対応づける。FR-7 の doctor と status は 7c-2、FR-8 の auto_merge と director の報告は 8a に送ったと書く。7b の B2(ralph は reviewer の `HEAD:`・`BASE:` を読まない)と合っていること
- Documentation drift to check: `/org` skill の 4 面が同じこと、AGENTS.md の repo map が新しいファイルを挙げること、README と仕様の注が動詞の挙動と合うこと
- Evidence to capture: 一時の repo(bare の origin、機能のブランチ 2〜3 本、台帳に結びつきと gate の記録)で HEAD の binary を打った出力(`ready`・`rebase`・`regate`・`no_remote`、`--json`、ls-remote が打てないときの終了コード 1)を `docs/evidence/` に残す

## Test plan

- Unit tests(`internal/org/mergeorder_test.go`、偽の seam)
  - 分類ごとに 1 本以上: ready、rebase(main が進んだ、既定のブランチを変えた)、wrong_base、regate、unreadable、failed、no_remote、no_gate、merged、closed
  - 最新の記録を読む: 通ったあとに落ちたら failed、落ちたあとに通ったら ready、通ったあとの記録が読めなければ unreadable、通ったあとに壊れた行があれば unreadable、壊れた行が最後の gate の記録より前なら影響しない
  - 読めない記録の付け先: 打った org のその時点の結びつき(同じ org_id で前後に別の機能があっても、記録より前の最新の結びつき)、前に結びつきがない記録はどこにも付かず注記だけ
  - ls-remote の symref: `HEAD` の symref がない出力はエラー、`refs/heads/HEAD` のような名前を既定と読み違えない
  - 対象: disband 済みの org のブランチ、同じ org_id で 2 つの機能、同じブランチに 2 つの結びつき(補足は最新)、dry-run・読めない記録・完全でない結びつきを数えない
  - 順: ready の中が gate の時刻の順、分類の順、`--all` の有無
  - PR の選び方: 複数の PR から番号が最大のもの、`isCrossRepository` が true の PR を外す、`gh` がない、`gh` が失敗する、JSON が読めない
  - ls-remote: 出力の読み方(ブランチ名の `/`、空の出力)、失敗と timeout はエラー
- Integration tests(`internal/cli/org_merge_order_test.go`): 本物の git と bare の origin、偽の `gh`(PATH の先頭に置く sh、または seam)で、テキストと `--json`、終了コード、`--all`
- Regression tests: `go test ./internal/org/... ./internal/cli/...` 全体。gate の記録の読み方(`gateRecordFromDetails`)を変えないこと
- Edge cases: 結びつきはあるが一度もリモートに push していない、gate の `base_ref` が `refs/heads/<名前>`(origin のない時点の gate)、既定のブランチの名前が main 以外、既定のブランチを変えて古いブランチが残る、PR の統合先が既定でない、ブランチ名に `/` が 2 つ以上、manifest が空、台帳がない、manifest の最後の行が改行のない書きかけ
- Evidence to capture: `docs/evidence/test-2026-10-14-org-merge-order-*.txt`(go test の出力、mutation の結果)

## Risks and mitigations

- ls-remote が認証の入力で止まる(ssh の passphrase など): `GIT_TERMINAL_PROMPT=0` と 30 秒の timeout で切り、理由を言って終了コード 1。ssh の askpass は止めきれない(tech-debt に書く)
- `gh` がない環境では merge 済みのブランチが `no_remote` で残り続ける: 注記で言い、`--since` は今回入れない(tech-debt)
- `gh pr list --head` はブランチの名前だけで引く: `isCrossRepository` で fork の PR を外す
- 対象が多いと `gh` を何度も打つ(ブランチの数だけ): 15 秒の timeout をブランチごとにかけ、1 本の失敗で全体を止めない。ブランチの数は max_orgs(既定 10)の org が作るぶんで、数十本の見込み
- 壊れた行や読めない記録のあと、ブランチは gate を打ち直すまで `unreadable` に残る: 意図した動き(C8)。新しい読める記録がその行より後にあれば分類はそれで決まる、とテストで確かめ、出力の行に打ち直せば消えると書く
- ls-remote の時刻と merge の時刻の間に main が進む: merge-order は「打った時点」の順で、保証はしない。保証は GitHub の保護(7c-2 の doctor の warn)に任せると注記する

## Rollout or rollback notes

- 新しい読むだけの動詞で、台帳の形も既存の動詞も変えない。ロールバックはこの PR の revert
- release のノート: 新しいサブコマンド `ralph org merge-order`(`--json`、`--all`)。`git ls-remote` でリモートを読み、`gh` があれば merge 済みと閉じた PR を外す

## Open questions

- なし(7c-2 の doctor の API の field 名は、計画の時点で `gh api` で確かめた: ruleset の `rules/branches/main` は `type` と `parameters` の配列を返し、classic の `branches/main/protection` は保護のないブランチで 404 `Branch not protected`)

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
  - S1 org の層: d27892f9(`internal/org/mergeorder.go` と test)
  - S2 CLI: 65a0d404(`internal/cli/org_merge_order.go` と test、`internal/cli/org.go` に登録)
  - S3 文書: 251b9234(`/org` skill の 4 面、README、仕様 FR-8 の注、AGENTS.md、tech-debt、`seat.go` の package doc、`org_legacy_ledger.go` の comment)。追いの 4277a295 で、ready が 2 本以上のときの注記の文言、tech-debt の gate なしの PR の行、skill の見張りなしで使える動詞の一覧を合わせた
  - 実装中に分かった穴: PR を出したあとの機能の org は disband 済みで、`ralph org gate` は走っている org でしか打てないので、`rebase`・`regate`・`unreadable` のブランチを `ready` に戻す決まった手がない(`start --plan` を打ち直すと worktree は使い回すが、新しい leader は計画から始める)。consult(consult-plan-regategap)の判定 A: この PR では埋めず、Details に「the gate runs only while the feature org runs」と足し、skill・README・仕様 FR-8 の注・tech-debt に穴と人の手の道(rebase して push、CI と GitHub の保護に頼る)と持ち主(8a、director)を書く。立て直しは勧めない。計画の非目標(rebase・gate をしない)の範囲で、本文は変えない
  - 決めたこと: 対象のブランチがなければ ls-remote を打たず、注記だけで終了コード 0
  - cycle 2(cross-review の ACTION_REQUIRED 1 件、gh が origin でない repository を読みうる): 15d4b947 と ee302b06。consult(consult-plan-ghrepo)の判定どおり、`--repo` は渡さず、gh が PR を返したら `git remote get-url origin` を 1 回だけ打ち、PR の url の owner/repo と origin の owner/repo を大文字小文字を無視して比べる。合わなければ PR をすべて捨てて注記 1 行(`gh repo set-default <origin の owner/repo>`)を出し、gh をそれ以上打たない。host は比べず、host のない origin(ローカルのパス)では比べない。計画の「git は ls-remote を 1 回」に、gh を使うときだけ `git remote get-url origin` が 1 回加わる逸脱。同じ cycle で、self-review の L1(gate は走っている org でしか打てない旨の定数)、L2(同じ gh の失敗の注記をまとめる)、L7(skill の折り返し)と、test の穴の P1〜P5 を埋めた
  - cycle 3(cross-review cycle 2 の ACTION_REQUIRED 1 件、gh の `pr list` の既定の上限 30 を fork の PR が埋めると本物の PR が落ちる。上限 2 に届いたので 3 に上げた。`cycle-count.json` は 2 のまま): consult(consult-plan-ghlimit)の判定どおり、`gh pr list` に `--limit 1000` を渡す(定数。gh は `--head` 単独では作成の新しい順に 1 ページ 100 件で辿るので、通常は費用が変わらない)。返った数が上限と同じで fork でない PR が 1 本もないときは、そのブランチの PR を使わずに注記する。別のコミットで self-review の C2-1(gate の記録だけが名指すブランチの Detail が `--org-id` を付けない)と C2-2(`GHTimeout` の doc が get-url にも使うことを言わない)を直す
  - cycle 4(cross-review cycle 3 の ACTION_REQUIRED 1 件と WORTH_CONSIDERING 1 件。上限を 4 に上げた。`cycle-count.json` は 2 のまま): consult(consult-plan-cycle3)の判定どおり 2 件とも直す。(1) gh が読む repository と origin のものが違うときの注記は、改名なら origin をそれに向ける(`git remote set-url origin`)、そうでなければ `gh repo set-default` を打つ、の両方を並べる。分類は今どおり PR なし。正規の名前を gh で引くことはしない。(2) 読めない gate の記録は、その org の前の完全な結びつきの枝に付け、結びつきがなければその org が前に書いた読める gate の記録の枝(最新のもの)に付け、どちらもなければ今どおりどのブランチにも付けない。計画の Scope と C8 の「前に結びつきがなければどのブランチにも付けない」を広げる逸脱で、gate の記録だけが名指すブランチが、読めない新しい記録のあとに古い pass で `ready` になるのを防ぐ
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
