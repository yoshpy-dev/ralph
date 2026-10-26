# leader-spawn-pr-rules

- Status: Approved
- Approved: 2026-10-26 sha256:7fb9a3d2e3cf
- Owner: Claude Code
- Date: 2026-10-26
- Related request: release の前に、通しの確かめ(PR #257)の所見のうち RC2 と RC5 だけを直す(ユーザーの選択「RC2 と RC5 だけ先に直す」、2026-10-26)
- Related issue: N/A
- Type: fix
- Branch: fix/leader-spawn-pr-rules

## Objective

通しの確かめ(`docs/evidence/org-recheck-2026-10-24.md`)で、既定の設定のまま雛形どおりに動くと起きる所見が 2 つ見つかった。

- RC2: leader の雛形の「機能ごとの org」の手順 5 が spawn に `--scope` を渡すと書いていない。既定の permission mode `autonomous` では、`ralph org spawn` は `--scope` も `--reserve` もない座席を断る。確かめでは、雛形どおりの spawn が 4 回とも断られ、leader が自分で `--scope` を足して打ち直した
- RC5: 機能の org の PR の題(#255、#256)と #255 の本文が英語だった。この repo の `/pr` skill は手順 5 で題と本文を日本語にすると決めているが、leader の雛形は「`/pr` skill の雛形(`template.md`)に合わせる」とだけ言い、言語の決まりは `template.md` にない

この計画では、leader の雛形と `/org` skill の同じ箇所を直し、雛形どおりに打てば spawn が断られず、PR の題と本文が `/pr` skill の決まりに沿うようにする。release の前に入れる。

## Scope

- S1(RC2): `internal/org/feature.go` と `internal/org/prompts/leader.md`
  - `featureLeaderTask` が `- 予約したパス:` の行の後ろに、座席の spawn に付ける引数を `shellQuote` で引用して書く(形は `- 予約したパス: a/, b/(座席の spawn には --scope 'a/, b/' を付ける)`。`- 台帳:` の行が `--state-dir` を引用して書くのと同じ形)。予約のパスの検査は `$`、バッククォート、引用符を通すので、leader が値を自分で引用すると shell が展開しうる(Codex の計画助言)
  - 「機能ごとの org」の手順 5: implementer と reviewer の spawn のどちらにも `--cwd .`・`--model` と、タスクの `- 予約したパス:` の行の括弧にある `--scope` の引数をそのまま(書き換えず)付ける。implementer の雛形は scope を `git add` の範囲に使い、reviewer の雛形は report と gate の生成物を scope の外でも書いてよいとしているので、どちらも予約にそろえる
  - 手順 3: 承認の前に reviewer を spawn するときも手順 5 と同じ flag を使う
  - 「ミッション」の 2: `--scope` で座席の担当範囲を渡す(既定の `autonomous` では必須で、ないと断られる)の 1 文
- S2(RC5): 同じ雛形の手順 14 の PR の文を、`/pr` skill があればその手順(`.claude/skills/pr/SKILL.md`、Codex では `.agents/skills/pr/SKILL.md`)のうち題と本文の決まり(言語、題の prefix など)と `template.md` に合わせ、`/pr` skill そのものは実行しない、にする
- S3: `/org` skill の同じ向きの 2 か所を直す
  - 機能ごとの org の座席の spawn の手順(`.claude/skills/org/SKILL.md` の「2. 座席を `spawn` する」、:1301 あたり)に、タスクの `- 予約したパス:` の行の括弧にある `--scope` の引数を書き換えずに付けることを足す
  - leader の PR の文(:1001 あたり「タイトルと本文は `/pr` skill の雛形に合わせてよいが」)を S2 と同じ向きにする
  - `scripts/sync-skills.sh` で `.agents/skills/org/`、`templates/base/.claude/skills/org/`、`templates/base/.agents/skills/org/` の写しをそろえる
- S4(テスト): `internal/org/prompts_test.go`
  - `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の既存の pin(手順 5 の `--id` の文、手順 14 の `gh pr create` の 3 句、ミッションの「implementer 1 席と reviewer 1 席を既定」)を残し、手順 5 の `--scope` と `- 予約したパス:`、手順 3 が手順 5 と同じ flag を使うこと、手順 14 が `/pr` skill の手順を名指すことを足す
  - ミッションの 2 の `--scope` の文を固定する
  - `internal/org/feature_test.go` の `featureLeaderTask` の期待値(:521 あたり)を新しい行の形にする。`$`、バッククォート、単一引用符、二重引用符を含む予約と、複数のパスの予約で、行の `--scope` の引数を `/bin/sh` に渡すと元の値(`, ` でつないだもの)と同じ 1 つの引数になることを確かめるテストを足す
- S5(記録)
  - `docs/tech-debt/README.md` の RC2 と RC5 の行を取り消し線で RESOLVED にし、ブランチ名 `fix/leader-spawn-pr-rules` の注を付ける
  - 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-4 と受け入れ条件 5 は `[ ]` のままにし、「直した。次の実行で付け直す」の印(2026-10-26)を足す。Status に RC2・RC5 を直したことを足す
  - 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` の RC2・RC5 の「直す PR」をブランチ名にする

## Non-goals

- RC1、RC3、RC4、RC6、RC9〜RC12 と `/org` skill のフォルダ信頼の項(ユーザーの選択で release のあと)
- 実機での確かめ(分離した herdr で機能の org を動かすこと)。FR-4 と受け入れ条件 5 の `[x]` は次の実行で付ける
- spawn の検査(`internal/org/spawn.go`)や `--scope` の意味を変えること
- `/pr` skill と `template.md` の書き換え(言語の決まりを `template.md` に移す案は採らない。Design decisions)
- `docs/evidence/org-spec-evidence-2026-10-21.md:105` と仕様 :29 の印の数(印の数は変わらない)
- release(ユーザーが切る)

## Assumptions

- `featureLeaderTask`(`internal/org/feature.go:403`)は `- 予約したパス: <パス>, <パス>` の行をタスクに書く。同じ関数の `- 台帳:` の行は `shellQuote` で引用した `--state-dir` を書いている
- 予約のパスの検査(`normalizeReservePath`、`internal/org/reserve.go`)が断るのは空、絶対パス、カンマ・空白・制御文字、`..` だけで、`$`、バッククォート、引用符は通る
- `--scope` は担当範囲の説明の文字列で、spawn は空でないことだけを見る(`autonomousScopeGateErr`、`internal/org/spawn.go`)。値の形は spawn の検査に関わらない
- `- 予約したパス:` の行を読むのは leader の雛形と `internal/org/feature_test.go` の期待値だけで、行を解析するコードはない(grep で確かめた)
- implementer の雛形(`internal/org/prompts/implementer.md`)は scope の中のパスだけを `git add` する。予約のパスを渡せば機能の変更をそのまま足せる
- reviewer の雛形(`internal/org/prompts/reviewer.md`)は `docs/reports/` の report と gate の生成物を scope の外でも書いてよいとしている
- 雛形は go:embed で binary に入る。新しい binary で spawn した leader から効く。scaffold した project の `/pr` skill も同じ手順 5(日本語の決まり)を持つ
- `/org` skill の 4 写しは `scripts/check-skill-sync.sh`(と CI)が突き合わせる

## Affected areas

- 変更: `internal/org/feature.go`(`featureLeaderTask` の 1 行)、`internal/org/feature_test.go`、`internal/org/prompts/leader.md`、`internal/org/prompts_test.go`
- 変更: `.claude/skills/org/SKILL.md` と写し 3 つ(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`)
- 変更: `docs/tech-debt/README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`
- 後工程の報告と insight event

## Visual review

- ページ: `.harness/state/plan-visual/leader-spawn-pr-rules.html`(やさしい図 1 枚 `overview`、折りたたみの中に `detail-overview` と `detail-spawn`)
- lint: `./scripts/plan-visual.sh lint` が終了コード 0
- 自分での確かめ: 全体、overview、detail-overview、detail-spawn の 4 枚を `shot` で撮って読み、線のラベルの位置、高さ、見出しの折り返しを直した。Codex の計画助言を入れたあとに描き直し、撮り直した

## Design decisions

Critical forks: None(ユーザーの選択で範囲が決まり、残りは下のとおり自分で選んだ)。

- D1(1 本の PR): RC2 と RC5 はどちらも leader の雛形と `/org` skill の同じ節を直し、同じテストで固定する。分けると同じファイルを 2 回触り、記録の更新も 2 回になる
- D2(`--scope` の値は予約): implementer も reviewer も `- 予約したパス:` の値をそのまま渡す。確かめでは hooks-matcher の reviewer が `review hooks-matcher; writes docs/reports/ only` を使ったが、reviewer の雛形が report の書き込みを scope の外でも許すので、説明を足す必要はない。値を 1 つの形に決めると、leader が自分で考える余地が減る
- D3(RC5 は雛形の文で直す): 言語の決まりを `template.md` に移す案は、`template.md` の 2 写しと scaffold の project の雛形を変え、project ごとに言語が違う場合にも当たる。leader の雛形に「`/pr` skill の手順の題と本文の決まりに従う」と書けば、project の `/pr` skill が決めた言語にそのまま従う
- D4(印は `[ ]` のまま): 印の規則(`docs/evidence/org-spec-evidence-2026-10-21.md:5`)は、外の道具(herdr、claude)に頼る項目の `[x]` に実機の記録を 1 つ求める。今回は実機で動かさないので、FR-4 と受け入れ条件 5 は `[ ]` のままにし、直したことと次の実行で付け直すことを印(2026-10-26)に書く。2026-10-22 の F4 の印は `docs/evidence/seat-spawn-close-2026-10-22.md` の実機の記録を引いていた(consult の指摘で読み直した)
- D5(「ミッション」の 2 にも書く): 分割計画を使わない org でも leader は同じ spawn を打ち、`autonomous` で断られる。雛形の 1 文で済むので足す
- D6(引用は ralph が作る): leader に「値を二重引用符で囲む」と書く案は、予約のパスにある `$` やバッククォートを shell が展開し、誤った担当範囲を渡したり、パスの中のコマンドを leader の権限で走らせたりしうる(Codex の計画助言、HIGH)。`featureLeaderTask` が `shellQuote` で引用した引数を書き、leader は書き換えずに写す。`- 台帳:` の行の `--state-dir` と同じ形で、leader の読み方もそろう。予約のパスの検査を厳しくする案は、`$` を含む実在のパス(`app/routes/$slug.tsx` など)を断ることになるので採らない

## Acceptance criteria

- [x] AC1(RC2): `featureLeaderTask` の `- 予約したパス:` の行が、`shellQuote` で引用した `--scope` の引数を括弧の中に持つ。leader の雛形の手順 5 が、implementer と reviewer の spawn のどちらにもその引数を書き換えずに付けると書く。手順 3 の承認前の reviewer の spawn も手順 5 と同じ flag を使うと書く。「ミッション」の 2 が `--scope` で担当範囲を渡すこと(`autonomous` では必須)を書く
- [x] AC1b: `$`、バッククォート、単一引用符、二重引用符を含む予約と複数のパスの予約で、行の `--scope` の引数を `/bin/sh` に読ませると、展開されずに元の値(パスを `, ` でつないだもの)と同じ 1 つの引数になる(テストで確かめる)
- [x] AC2(RC5): 手順 14 が、`/pr` skill があればその手順(`.claude/skills/pr/SKILL.md`、Codex では `.agents/skills/pr/SKILL.md`)の題と本文の決まり(言語を含む)と `template.md` に合わせ、`/pr` skill そのものは実行しないと書く
- [x] AC3: `/org` skill の機能ごとの org の spawn の手順と leader の PR の文が、AC1・AC2 と同じ向きになっている。4 写しが同じで `scripts/check-skill-sync.sh` が通る
- [x] AC4: `internal/org/prompts_test.go` が AC1 と AC2 の文を固定し、既存の pin を残している。文を元に戻すとテストが落ちる(手で 1 回ずつ戻して確かめる)
- [x] AC5: tech-debt の RC2・RC5 の行が取り消し線の RESOLVED で、ブランチ名を書いている。仕様の FR-4 と受け入れ条件 5 は `[ ]` のままで、印(2026-10-26)がある。根拠の表の RC2・RC5 の「直す PR」がブランチ名になっている
- [x] AC6: `./scripts/run-verify.sh` が通り、Go のテストが手元の版と `GOTOOLCHAIN=go1.25.8` の両方で通る

## Implementation outline

1. S1・S2・S4: `featureLeaderTask` の行、leader の雛形、feature_test.go と prompts_test.go(implementer、1 コミット)
2. S3: `/org` skill の 2 か所と sync-skills(implementer、1 コミット)
3. S5: 記録(implementer、1 コミット)
4. 後工程(self-review → verify → test → sync-docs → cross-review → pr)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(Go の vet と gofmt、shell)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC6。手順 5 の `--scope` が spawn の検査(`internal/org/spawn.go` の autonomous の検査)を満たす形であること。`/org` skill の `spawn` の行(`--scope` の説明)と食い違わないこと
- Documentation drift to check: leader の雛形と `/org` skill の手順の対応、仕様の印の規則と FR-4・受け入れ条件 5 の印、tech-debt と根拠の表の RC2・RC5
- Evidence to capture: 変えた文の行番号、check-skill-sync の出力、仕様の印の数(base と HEAD で同じ)

## Test plan

- Unit tests: `TestRenderRolePrompt_Leader_FeatureOrgProcedure` に AC1・AC2 の文を足す。ミッションの 2 の文も固定する。`featureLeaderTask` の新しい行の期待値と、AC1b の引用のテスト(`/bin/sh -c` に引数を読ませて `printf '%s\n'` で返し、元の値と比べる。外の道具は打たない)
- Integration tests: なし(spawn の動きは変わらない。変わるのは leader のタスクの 1 行と雛形の文)
- Regression tests: `go test -count=1 -timeout 30m ./...` を手元の版と `GOTOOLCHAIN=go1.25.8` で。`./scripts/run-test.sh`
- Edge cases: 予約が複数のパス(`, ` でつないだ値)、`$` を含むパス、バッククォート、単一引用符(`shellQuote` が `'\''` にする)、二重引用符。`featureLeaderTask` が `- 予約したパス:` の行か `--scope` の引数を書かなくなればテストが落ちること
- Evidence to capture: 文を戻したときに落ちるテストの名前(AC4)、Go のテストの 2 回の結果

## Risks and mitigations

- R1: 既存のテストが雛形の文を固定していて、文を変えると落ちる。既存の pin を残す形で足す。落ちたら pin の意図を読んでから直す
- R2: `/org` skill の写しの 1 つだけを直して drift になる。`scripts/sync-skills.sh` で写し、`scripts/check-skill-sync.sh` で確かめる
- R3: 直した雛形を実機で確かめないので、leader が実際に `--scope` を渡し日本語で PR を書くかは分からない。FR-4 と受け入れ条件 5 は `[ ]` のままにし、次の実行で確かめる(Known gaps に書く)
- R4: 予約の値は複数のパスを `, ` でつないだもので空白を含み、1 つのパスも `$`、バッククォート、引用符を持ちうる。leader が自分で引用すると shell が分けるか展開する。D6 のとおり ralph が `shellQuote` で引用した引数を書き、AC1b のテストで固定する
- R5: leader のタスクの `- 予約したパス:` の行の形が変わる。行を解析するコードはない(Assumptions)。古い binary が書いたタスクの leader は新しい雛形を読まない(雛形も binary に入る)ので、形の食い違いは起きない

## Rollout or rollback notes

- 雛形は binary に入るので、新しい binary で spawn した leader から効く。release のノートに「leader の雛形が座席の spawn に `--scope` を渡し、PR の題と本文を `/pr` skill の手順に合わせる」と書く(題が `fix:` なので changelog にも出る)
- 戻すときはこの PR を revert する

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult の計画の相談 consult-plan-rc2rc5 の 3 点と、Codex の計画助言 1 件(HIGH、予約のパスの引用)を承認の前に入れた)
- [x] Plan approved(2026-10-26。ユーザーの決まり「作成したプランは自動承認します」による自動承認)
- [x] Branch created
- [x] Implementation started
  - S1・S2・S4(efddd301): `featureLeaderTask` の行、leader の雛形のミッションの 2 と手順 3・5・14、`feature_test.go` と `prompts_test.go`。Go のテストは go1.26.0 と go1.25.8 の両方で通り、文を 1 つずつ戻すとテストが落ちた(AC4、`go test -overlay` で写しを差し替えた)
  - S3(c654d596): `/org` skill の leader のサイクルの手順 2 と「org の終わり方」の手順 6、4 写し。`check-skill-sync.sh` と `check-sync.sh` が通った。`go test ./internal/org/... ./internal/cli/...` の 1 回目は `TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing/SIGTERM` が load average 約 12 の下で 10 秒の待ちを切らして落ち、2 回目は通った(この slice は Go のコードを変えていない)
  - S5(7d51236d): tech-debt の RC2・RC5 の行、仕様の FR-4 と受け入れ条件 5 の印(2026-10-26)と Status、根拠の表。`run-verify.sh` は S2 と S5 のあとでどちらも通った
  - AC6 の Go のテストを 2 つの版で通すことは、test の段で確かめる(1e2cf34f の test の report: go1.26.0 で rc 0、`GOTOOLCHAIN=go1.25.8` で rc 0、`run-test.sh` も rc 0)
- [x] Review artifact created(43dc55e4、Merge: OK、LOW 2 件は 0309acf8 で直した)
- [x] Verification artifact created(2947c524、pass。tech-debt の Trigger の LOW 3 件 V1〜V3 は /sync-docs に回す)
- [x] Test artifact created(1e2cf34f、pass。mutant は 12 個すべて落ちた)
- [ ] PR created
