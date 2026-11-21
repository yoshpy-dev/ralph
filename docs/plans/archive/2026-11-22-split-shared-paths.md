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

- [x] AC1: ヘッダに `- Shared: pnpm-lock.yaml, tsconfig.json` を書いた分割計画を読むと、`SplitPlan.Shared` に正規化した 2 つのパスが入る。`- Shared:` が 2 行、`.`、絶対パス、`..`、機能の `- Reserve:` と重なるパスのどれかなら、理由を言って断る
- [x] AC2: `- Shared:` のない分割計画の、計画の digest と機能の digest は今と同じ(既存の計画の承認と、止めた機能の org の立て直しが今のまま通る)
- [x] AC3: `ralph org start --plan` が記録する予約の Details の最後に `shared=<パス>,...` が入り、`reservationFromEvent` が `Shared` に読み戻す。`shared=` のない古い記録は `Shared` が空
- [x] AC4: 共有のパスだけを変えた座席に、見張りは scope_change の ALERT を上げない(受信箱に入らない)。共有でも予約の内でもないパスの変更は今のとおり ALERT になる
- [x] AC5: 2 つの機能の org がどちらも共有のパスを変えても、予約の重なりの判定は今のまま(共有のパスどうしで断らない、予約どうしの重なりは今のとおり断る)
- [x] AC7: 走っている org の予約のパスと重なる共有のパスを持つ機能の org の start は断られ、走っている org の共有のパスと重なる予約のパスを持つ org の start も断られる(start の順の両方、ファイルとディレクトリの重なりをテストする)
- [x] AC8: 共有のパスのある計画の機能の leader のタスクに `- 共有のパス:` の行があり、座席の spawn の `--scope` の案内が予約のパスと共有のパスを合わせたものになる。leader の prompt が共有のパスも変えてよいと書く。`ralph org reseat` と `start --plan` の立て直しでも同じタスクになる(テストで確かめる)
- [x] AC6: director skill が `- Shared:` の書き方を、org skill が共有のパスの衝突の直し方を書く(各 4 写し)。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る

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
- [x] Implementation started
  - S1(35458c51、implementer): 確認コマンドが exit 0。`SplitPlan.Shared`。ヘッダの `- Shared:` を読み、空の値、空の要素、`.`(`./` も)、予約の規則で断るパス、2 行、どれかの機能の `- Reserve:` との重なり(機能と両方のパスと行を名指す)を断る。囲みの中の `- Shared:` は欄でない。`featureDigest(preamble, shared, f)` は共有のパスがあるときだけ `shared <len>` を足し、ないときは今と同じ値(HEAD 1a478b6b で計った digest をテストに固定)。変異 5 種でテストが落ちる。S3 への注意: `featureDigest` の doc は「座席に渡るものと同じ」と言うので、S3 で leader のタスクに `- 共有のパス:` を足すまで食い違う。機能の節や前文の中の `- Shared:` は本文として扱い、断らない(S4 の skill で書き方を案内する)
  - S2(2bb50cd4、implementer): 確認コマンドが exit 0。Details の最後の `shared=`、受け付けで共有と別の org の予約の重なりを両方向で断る(共有どうしは断らない)、`start --plan` が計画の共有のパスを記録する、同じ計画での立て直しは通り共有のパスが変われば断る。変異 11 種でテストが落ちる。ずれ: (1) 共有のパスは計画の書く `Reservation.Shared` ではなく `FeatureBinding.Shared`(正規化したパスをカンマで結んだ文字列)と `SharedPaths()` に持たせた(記録の予約は範囲外の `spawn.go` の `SpawnParams.reservation()` が組み立てるので `Reservation` に足しても `start --plan` の記録に届かない、`FeatureBinding` は比較に使われ `[]string` を持てない)。共有のパスは機能の org にだけ付く、(2) 壊れた `shared=` の値は「共有のパスなし」と読む(repo 全体と読むと見張りが何も知らせなくなる。代わりにそのパスは受け付けで守られない)。S3 への注意: 見張りは `ActiveFeature(events, orgID).SharedPaths()`、leader のタスクは `plan.Shared` を使う
  - S3(89429da9、implementer): 確認コマンドが exit 0。見張りは予約のパスと `SharedPaths()` を合わせて「外」を判定する(ALERT の文は変えない)。leader のタスクに `- 共有のパス:` の行を足し、`- 予約したパス:` の行の括弧の `--scope` を予約と共有を合わせたものにした(共有のパスのない計画のタスクは 1 バイトも変わらない)。leader.md の「機能ごとの org」に共有のパスの段落。implementer.md は `{{SCOPE}}` だけで区切るので変えていない(前提をテストで固定)。変異 5 種でテストが落ちる。範囲外の発見: `ralph org reseat` が立て直す implementer と reviewer の `--scope`(`internal/org/reseat.go:471`、555 行目と `reseatCustomRoleErr` の 933 行目も)は今も予約のパスだけで、AC8 の立て直しの `--scope` を満たしていない → S3b で直す
  - S3b(c9e99c63、implementer): 確認コマンドが exit 0。`PlanReseat` の `scope` を予約のパスと `binding.SharedPaths()` を合わせたものにした(leader 以外の座席の spawn、dry run、`reseatCustomRoleErr` が出す spawn のコマンドの 3 か所がそろう)。共有のパスのない org は今と同じ文字列。変異 5 種でテストが落ちる。範囲外の気づき: reseat が立て直す座席の `--scope` に共有のパスが入ることは、まだ文書にない(S4 で reseat の仕様か skill に書く)
  - S4(eb43b1b7、implementer): 確認コマンドと `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` が exit 0。director skill に「### 共有のパス」(書き方、断る形、承認と digest、別の org の予約との重なり、ほかの節の追記)、org skill に「### 共有のパスの直し方」(あとから merge する側が直す、生成されるファイルは既定のブランチの版に戻して作り直す、手で書くファイルは両側を残す、gate から進め直す)と reseat の `--scope`、各 4 写し。README、AGENTS.md、2 つの仕様。`internal/org/skill_shared_paths_test.go` で skill の文と例の読みを固定。範囲外の気づき: (1) `ralph org start --help`(`internal/cli/org.go` 1094〜1100 行付近)の立て直しの条件が共有のパスの変化に触れていない、(2) `ralph status` と `ralph org status` は共有のパスを出さない(director は分割計画のファイルを読む)
  - self-review の LOW の直し(143deb8f、implementer): L1〜L8。L3 で、ヘッダより後(前文の節、機能の節、`## Features` の最初の `###` より前、`## Features` より後の節)のフェンスの外で行頭に書いた `- Shared:` を読み込みで断るようにした。上の S1 の記録の「本文として扱い、断らない」はこれで改めた(字下げした行と、文の途中で Shared に触れる行は今までどおり本文)。仕様 `2026-10-07-org-multi-org-director.md` の FR-4 の同じ日の項目もこれに合わせた(6168f606。/verify で「FR-2」の書き誤りを直した)。L8 で `- Reserve:` の拒否の文の頭も `reserve path` から `path` になった
- [x] Review artifact created — `docs/reports/self-review-2026-11-22-split-shared-paths.md`(Merge: yes、LOW 8 件は 143deb8f で直した。cycle 2 の Run 2: Merge: yes、MEDIUM 1(R2-1: 手順 1 の pnpm の例と再適用の命令が pnpm の動きと合わない)、LOW 2)
- [x] Verification artifact created — `docs/reports/verify-2026-11-22-split-shared-paths.md`。Run 1 は pass。cycle 2 の Run 2(HEAD e5dbabe8)も pass。cross-review の ACTION_REQUIRED 1 は 2c0c28e6 で、self-review Run 2 の R2-1 と R2-2 は e5dbabe8 で直った(当て直しの命令をやめ、書き留めた版が戻っていれば `BLOCKED` で止まる)。R2-3 は直していない。下の「`/pr` の前」の取り込みは 809a887e で済み、そのあと check-sync と check-skill-sync は通った(`./scripts/run-test.sh` は cycle 2 の `/test` が回す)
- [x] Test artifact created — `docs/reports/test-2026-11-22-split-shared-paths.md`。Run 1(HEAD 77a6961a)は pass。cycle 2 の Run 2(HEAD c17f1b9e、main の取り込みのあと)も pass: `./scripts/run-test.sh` が rc 0(1,350 s)、落ちたテストはなく、cycle 2 で足した skill の固定は 4 写しへの変異 4 個でどれも落ちた
- [x] Sync-docs artifact created — `docs/reports/sync-docs-2026-11-22-split-shared-paths.md`(文書の直しは 5cdc7150)。verify の V6・V7(org skill のコマンド表の `spawn` の行と分割計画の例の座席の決まり、各 4 写し)と V8(tech-debt の Trigger の引き継ぎの注)を直した。注は 17 行(verify が挙げた 12 行と leader.md の段落の行、sweep で見つけた 88・202・519・521)、新しい行は 3 つ(`ralph status` が共有のパスを出さない、`reserve.go` が 822 行、この PR の known gaps)。Go の comment 7 か所は `docs/plans/active/...` の path で計画を引いていて、`/pr` が計画を archive へ動かすと切れるので、slug の引用に直した(comment だけ、動きは変えていない)
  - release ノートの候補: ヘッダより後(前文の節、機能の節、`## Features` の最初の `###` より前、`## Features` より後の節)のフェンスの外の行頭に `- Shared:` を書いた既存の分割計画と、ヘッダの自由な文として `- Shared:` で始まる行を持つ既存の分割計画は、この版から読み込みで断られる(前は本文か自由な文として読まれた)。鍵が #363 で初めて決まったので影響は小さいと読む。上の Rollout は revert の向きだけを書いている
  - cycle 2 の sync-docs(`docs/reports/sync-docs-2026-11-22-split-shared-paths.md` の `## Run 2 (cycle 2)`、文書の直しは 35f7aaa4): self-review Run 2 の R2-3(ヘッダに `- Shared:` があるときの、ヘッダより後の行の拒否の文)と verify Run 2 の INFO(手順 1 が `BLOCKED` で止まるとき、途中の merge か rebase を残すか `--abort` するか)を、tech-debt の新しい 1 行(554 行目)にした。R2-1 と R2-2 は e5dbabe8 で直ったので、self-review が用意した行は足していない。仕様 FR-4 の 2026-11-22 の項目の要約に、書き留める、確かめる、戻っていれば `BLOCKED` で止まる、を足した。#366 を取り込んだあとの `leader.md`(1,668 行)と `feature.go`(899 行)を、その数を挙げる tech-debt の 5 行に注した。2 周目の直しが新しく当てた Trigger の行はない
  - `/pr` の前: main は 2 回取り込み済み(809a887e で PR #368 の #362、3fba16b4 で PR #369 の #366。衝突は両側の追記を残して直した)。`check-sync.sh` と `check-skill-sync.sh` は通る。cycle 2 の `/test`(`./scripts/run-test.sh` rc 0)は 809a887e のあと、3fba16b4 の前(HEAD c17f1b9e)に回したので、#366 を入れた木の `./scripts/run-test.sh` は未実行。3fba16b4 の上で、`go build`、`go vet`、`gofmt`、`internal/org` の Skill・DocsMatch・SharedPaths・SharedLine・Leader・HumanDecision のテストと、`internal/cli` の Agents・Readme・Docs・Skill・Help・Reserve・Send のテストは通った(sync-docs が回した)。`/pr` の前に `./scripts/run-test.sh` を一度回すかは lead の判断。PR 本文には、cross-review の ACTION_REQUIRED 1 の直しが triage の案(同じ更新のコマンドで当て直す)と違い、当て直しをやめて戻った版は `BLOCKED` で止まる形にしたことを書く(self-review Run 2 の R2-1)
- [ ] PR created
