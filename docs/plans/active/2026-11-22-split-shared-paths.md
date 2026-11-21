# split-shared-paths

- Status: Approved
- Approved: 2026-11-22 sha256:9046d5c8297e
- Owner: Claude Code
- Date: 2026-11-22
- Related request: issue #363(分割計画で、どの機能も変えてよい共有のパス(lockfile など)を宣言できない)
- Related issue: 363
- Type: feat
- Branch: feat/split-shared-paths

## Objective

分割計画で、どの機能の org も変えてよい共有のパス(`pnpm-lock.yaml`、root の `tsconfig.json` など)を宣言できるようにする。見張りの scope_change は、宣言した共有のパスへの変更を予約の外の変更と数えない。機能の予約の重なりの判定は変えない。共有のパスが並んで走る org の間で衝突したときの直し方を skill に書く。

## Scope

- 分割計画の形(`internal/org/split.go`): ヘッダ(`- Status:` と `- Approved:` のある最初の部分、最初の `## ` より前)に `- Shared: <パス>, <パス>` の行を 1 つ書ける。パスは予約と同じ規則(`NormalizeReservePaths`: repo の root からの相対、`..` なし、空白・`,`・制御文字なし)で読む。`.`(repo 全体)は断る(全部を共有にすると予約の意味がなくなる)。2 行あれば断る。`SplitPlan.Shared` に入れる
- 共有のパスと機能の予約: どの機能の `- Reserve:` とも重なる共有のパスは断る(同じパスが予約の内でも共有でもあると、どちらの規則が効くか読み手に分からない)
- 機能の digest(`featureDigest`): 計画に `- Shared:` があるときだけ、その値を機能の digest に入れる(共有のパスが変わればその機能の org が触ってよいものが変わる)。`- Shared:` のない計画の digest は今と同じ
- 予約の記録(`internal/org/reserve.go`、`internal/org/feature.go`): `ralph org start --plan` が機能の予約を記録するとき、計画の共有のパスを EventScopeReserved の Details の最後のトークン `shared=<パス>,<パス>` に書く(古い binary は知らない鍵から先を note として読むので、最後に置く)。`Reservation` に `Shared []string` を足す。予約どうしの重なりの判定(`reserveOverlaps` の Paths と Paths)は今のまま
- 予約の受け付け(`internal/org/reserve.go` の manifest の lock の中の判定): 新しい予約の `Shared` が走っている別の org の `Paths` と重なるとき、また新しい予約の `Paths` が走っている別の org の `Shared` と重なるときは、どちらも断る(別の計画や `--reserve` の org が持つパスを共有として変え、見張りも知らせない、という食い違いを防ぐ。codex の計画助言の 2)。`Shared` どうしの重なりは断らない
- 座席が変えてよい範囲(`internal/org/feature.go` の `featureLeaderTask`、`internal/org/prompts/leader.md`、必要なら implementer の prompt): leader のタスクに `- 共有のパス:` の行を足し、座席の spawn の `--scope` は予約のパスと共有のパスを合わせたものにする。leader の prompt の「機能のコードと文書の変更は `- 予約したパス:` の中に収める」に、共有のパスも変えてよいことを足す。立て直し(`ralph org reseat`、`start --plan` の立て直し)でも同じタスクと `--scope` になる(codex の計画助言の 1)
- 見張り(`internal/org/watch.go`、`internal/org/watch_scope.go`): scope_change の「予約の外」を判定するとき、予約のパスに加えて共有のパスの内も「外ではない」と数える。共有のパスだけの変更では scope_change の ALERT が受信箱に入らない
- skill: director skill(分割計画の書き方に `- Shared:` の行、どんなパスを書くか、予約と重ねないこと)、org skill(共有のパスが並んで走る org の間で衝突したときの直し方: 後から merge する側が既定のブランチを取り込み、lockfile を作り直す(`pnpm install` など)、生成物は手で merge しない)。各 4 写し
- README(分割計画の説明があれば)、仕様(分割計画の形を書く仕様に日付つきの注)

## Non-goals

- `ralph org start --reserve`(分割計画なし)の共有のパス
- 共有のパスの衝突を ralph が自動で直すこと(merge・regate・gate は今のまま。直し方は skill に書く)
- 共有のパスを機能ごとに変えること(計画で 1 つ)

## Assumptions

- 機能の org の予約は `ralph org start --plan` が記録し、見張りは `ActiveReservation`(`watch.go`)で読む。共有のパスも同じ記録から読める
- 今の分割計画のヘッダの自由な文は「どこにも行かない」(`split.go` の doc)。`- Shared:` の行は、ヘッダの中で読む初めての欄になる。digest はヘッダの行(`- Status:` と `- Approved:` 以外)を含むので、承認のあとに共有のパスを書き換えれば digest が変わる

## Affected areas

- `internal/org/split.go` とテスト(`- Shared:` の読みと確かめ、`featureDigest`)
- `internal/org/reserve.go` とテスト(`Reservation.Shared`、Details の `shared=` の書きと読み)
- `internal/org/feature.go` とテスト(`start --plan` が共有のパスを予約に渡す、`featureLeaderTask` の `- 共有のパス:` と `--scope`)
- `internal/org/prompts/leader.md`(と必要なら `implementer.md`)と prompt のテスト
- `internal/org/watch.go`、`internal/org/watch_scope.go` とテスト(scope_change の判定)
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と各写し、README.md、分割計画の形を書く仕様
- 共有ファイルの注意: `feature.go` はレーン D の #365 も触る。director skill は他のレーンも触る

## Visual review

- ページ: `.harness/state/plan-visual/split-shared-paths.html`。かんたんな図 1 枚(`overview`)と、詳しい図 2 枚(`detail-overview`、`detail-flow`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 3 枚の図を `shot` で撮って読み、直すところはなかった

## Design decisions

Critical forks: None

- 宣言の場所はヘッダの `- Shared:` の行にする。前文の `## ` の節にすると、leader に渡る自由な文と機械で読む欄が混ざる。ヘッダは今どこにも行かないので、欄を足しても他の動きを変えない
- 共有のパスは予約どうしの重なりの判定に入れない(issue の受け入れ条件 2)。並んで走る org がどれも共有のパスを変えてよいことが目的。ただし、共有のパスと別の org の予約のパスの重なりは断る(片方が持つと決めたパスを、もう片方が共有として黙って変えないため)
- 座席が変えてよい範囲は「予約のパス ∪ 共有のパス」、ほかの org と分け合わない範囲は「予約のパス」だけ。2 つを分けて持つ
- 共有のパスと機能の予約の重なりは断る。どちらの規則が効くかを読み手が迷わないため
- `.` は共有にできない。全部を共有にすると scope_change が何も見なくなる

## Acceptance criteria

- [ ] AC1: ヘッダに `- Shared: pnpm-lock.yaml, tsconfig.json` を書いた分割計画を読むと、`SplitPlan.Shared` に正規化した 2 つのパスが入る。`- Shared:` が 2 行、`.`、絶対パス、`..`、機能の `- Reserve:` と重なるパスのどれかなら、理由を言って断る
- [ ] AC2: `- Shared:` のない分割計画の、計画の digest と機能の digest は今と同じ(既存の計画の承認と、止めた機能の org の立て直しが今のまま通る)
- [ ] AC3: `ralph org start --plan` が記録する予約の Details の最後に `shared=<パス>,...` が入り、`reservationFromEvent` が `Shared` に読み戻す。`shared=` のない古い記録は `Shared` が空
- [ ] AC4: 共有のパスだけを変えた座席に、見張りは scope_change の ALERT を上げない(受信箱に入らない)。共有でも予約の内でもないパスの変更は今のとおり ALERT になる
- [ ] AC5: 2 つの機能の org がどちらも共有のパスを変えても、予約の重なりの判定は今のまま(共有のパスどうしで断らない、予約どうしの重なりは今のとおり断る)
- [ ] AC7: 走っている org の予約のパスと重なる共有のパスを持つ機能の org の start は断られ、走っている org の共有のパスと重なる予約のパスを持つ org の start も断られる(start の順の両方、ファイルとディレクトリの重なりをテストする)
- [ ] AC8: 共有のパスのある計画の機能の leader のタスクに `- 共有のパス:` の行があり、座席の spawn の `--scope` の案内が予約のパスと共有のパスを合わせたものになる。leader の prompt が共有のパスも変えてよいと書く。`ralph org reseat` と `start --plan` の立て直しでも同じタスクになる(テストで確かめる)
- [ ] AC6: director skill が `- Shared:` の書き方を、org skill が共有のパスの衝突の直し方を書く(各 4 写し)。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る

## Implementation outline

1. S1 分割計画の `- Shared:` と digest(AC1、AC2) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Split|Digest'`
2. S2 予約の記録と受け付け(AC3、AC5、AC7) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Reserve|Reservation|Feature'`
3. S3 見張りと、leader のタスクと prompt(AC4、AC8) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Scope|Watch|Feature|Prompt|Reseat' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgStart'`
4. S4 skill と文書(AC6) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill|Director|DocsMatch|Split' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Agents|Readme|Docs|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC6 と issue #363 の受け入れ条件 3 つ
- Documentation drift to check: `split.go` の形の doc、分割計画の形を書く仕様と skill、`ralph org start --help`(分割計画の形を書いていれば)、AGENTS.md の split.go と watch_scope.go の説明
- Evidence to capture: verify の report

## Test plan

- Unit tests: `- Shared:` の読み(1 つ、複数、空白、正規化、断る形)、digest(Shared なしは今と同じ値、あれば機能の digest が変わる)、Details の書きと読み(`shared=` あり・なし、古い記録、note つき)、scope_change(共有だけ、共有と外、予約の内と共有、ディレクトリの共有)
- Integration tests: `ralph org start --plan` で共有のパスが予約の記録に入り、見張りの 1 周で共有のパスの変更に ALERT が立たない
- Regression tests: 既存の split、reserve、feature、watch のテスト
- Edge cases: 共有のパスがディレクトリ(`packages/shared/`)、共有のパスの下のファイルの rename、予約が `.` の機能(共有と重なるので断る)
- Evidence to capture: `./scripts/run-test.sh` の log、共有の除外を外す変異で AC4 のテストが落ちること

## Risks and mitigations

- digest が変わって、承認済みの計画や止めた機能の org が立て直せなくなる: `- Shared:` があるときだけ機能の digest に入れ、ないときは今と同じ値になることをテストで固定する
- 古い binary が `shared=` を読めない: Details の最後に置くので、古い binary は note として読み、予約のパスと binding は今のとおり読める
- 共有のパスを広く書きすぎると scope_change が見なくなる: `.` を断り、skill に「lockfile や root の設定のように、どの機能も変えざるをえないファイルだけ」と書く
- 他のレーン(D の #365)が `feature.go` を触る: merge の前に main を取り込む

## Rollout or rollback notes

この PR を revert すれば、`- Shared:` の行は分割計画のヘッダの自由な文に戻り(どこにも行かない)、記録の `shared=` は note として読まれる。共有のパスの変更は今のとおり scope_change になる。`- Shared:` のある計画の機能の digest は revert のあと変わるので、止めた機能の org を立て直すときに計画の変更として扱われる(release ノートに書く)。

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 2 件、どちらも計画に入れた。[HIGH] 1: 共有のパスを宣言しても、座席に渡るのは予約のパスだけで、leader の prompt は予約の外の変更を禁じるので、座席は共有のパスを変えられない → 座席が変えてよい範囲を「予約のパス ∪ 共有のパス」にし、leader のタスクの `- 共有のパス:` の行、`--scope`、leader の prompt、立て直しでも同じになることを Scope と AC8 に足した。[MEDIUM] 2: 共有のパスが別の org の予約のパスと重なっても断られない → manifest の lock の中で、共有と別の org の予約の重なりを両方向で断る(共有どうしは断らない)。AC7 を足した。S2 が大きくなったので S2(記録と受け付け)と S3(見張りと leader のタスク)に分け、文書を S4 にした。図解の印も合わせた。git の状態は助言の前後で同じ
- [x] Plan approved — 2026-11-22、系列の決まり(計画は推す案で自動承認)で承認した。人の目での承認はしていない
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
