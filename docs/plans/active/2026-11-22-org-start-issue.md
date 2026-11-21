# org-start-issue

- Status: Approved
- Approved: 2026-11-22 sha256:cbeb26942ff9
- Owner: Claude Code
- Date: 2026-11-22
- Related request: issue #365(issue に起票した機能の org の PR が、その issue を閉じない)
- Related issue: 365
- Type: feat
- Branch: feat/org-start-issue

## Objective

issue に起票した機能(本文に `ralph-split:` の印の行がある issue)の org を立てるとき、その issue の番号を leader に渡し、leader が PR の本文に `Closes #<番号>` を書くようにする。PR が merge されると issue が閉じ、待つ側の issue の「頼られる issue」の行が merge 済みの機能を open と読まなくなる。

## Scope

- `ralph org start --plan <分割計画> --feature <slug> --issue <番号>`: `--issue` を足す。`--plan` と一緒のときだけ受け、値は 1 から始まる 10 桁までの 10 進数(`#` は付けない)。それ以外は何も作らずに断る
- 結びつき(`FeatureBinding`)に `Issue` を足し、予約の記録の Details の最後に `issue=<番号>` を書く(#363 の `shared=` のあと。#363 を merge してから実装する)。読めない値(数でない、2 回)は「issue なし」と読み、結びつきは完全なままにする(`shared=` と同じ扱い)
- 立て直し: `ralph org reseat` は記録の `Issue` を leader の start に渡す。disband していない org の `start --plan` のやり直しは、`--issue` が記録と同じとき(どちらもないときを含む)だけ通し、違えば記録と今の値を名指して断る(`sameFeature` と `featureDigestRefusal` と同じ所)
- leader のタスク(`featureLeaderTask`): `Issue` があるときだけ `- issue: #<番号>(PR の本文に Closes #<番号> を書く)` の行を足す。ないときのタスクは 1 バイトも変わらない。issue の行は分割計画の項目ではないので、機能ごとの digest(`featureDigest`)には入れない。そのことを `featureLeaderTask` のコメント(分割計画の項目を足したら digest にも足す、という所)に書く
- leader の雛形(`internal/org/prompts/leader.md` の手順 14): タスクに `- issue:` の行があれば、PR の本文の普通の文(コードブロック、HTML のコメント、引用の外)に `Closes #<番号>` を 1 行書く。行がなければ今の決まり(`/pr` の雛形と、計画の `Related issue:` の扱い)のままで、新しく禁じることはない。ただし、雛形の `Closes #__ISSUE__` の置き場の文字を本文に残さない(閉じる issue がなければその行を消す。`/pr` の決まりと同じ)と一言添える
- `ralph org status --org-id <org>`: 結びつきに `Issue` があれば `feature:` の行の次に `issue: #<番号>` の行を出す。`--json` の feature に `issue`(番号、ないときは出さない)
- `ralph org merge-order --json`: 各ブランチの `binding` に、そのブランチの最後の結びつきの `issue`(番号、ないときは出さない)を出す。org が disband したあとも、director が入れ替わったあとも、台帳からブランチの issue を引ける。テキストの出力は変えない
- director skill(4 写し): 「merge を待つ issue」の 3 で、issue の機能の org を立てるときに `--issue <その issue の番号>` を付ける。「機能ごとの出力先」の 3 の start の説明に `--issue` を書く。起票した機能の PR が merge されたと人に報告するとき(「merge と auto_merge」と「GitHub で merge した PR の後始末」)は、`ralph org merge-order --all --json` でそのブランチの `binding.issue` を読み、あればその issue が閉じたかを `gh issue view --repo github.com/<owner>/<repo> <番号> --json state --jq .state` で確かめ、`OPEN` なら閉じずに人に伝える(issue を閉じるのは人が頼んだときだけ、の決まりは変えない)。引き継ぎでも、merged のブランチの `binding.issue` が OPEN のままなら人に伝える
- org skill(4 写し): start の flag の説明と、立て直しで `--issue` が記録と違うと断られること
- README、AGENTS.md(必要なら)、仕様 `docs/specs/2026-10-07-org-multi-org-director.md`(FR-2 に 2026-11-22 の項目)と `docs/specs/2026-11-12-org-reseat.md`(FR-6 の leader の start が issue を引き継ぐこと)

## Non-goals

- `ralph org start` が gh で issue を探すこと(印の行の分割計画の id と slug から探す案)。start に gh と issue の本文の信頼の確かめが入る。印の行と書いた人の確かめは今も director がしている
- `ralph org start <task>`(分割計画なし)の org の issue
- 1 つの PR で 2 つ以上の issue を閉じること
- 起票していない機能の PR の本文(今のまま)
- `ralph status` の org ごとの行に issue を出すこと

## Assumptions

- 機能の PR は origin の repository にでき、既定のブランチに merge される。`ralph org merge` は origin の repository の PR だけを読み(`ReadMergeOrigin`)、director は origin の repository に issue を起票する。そのため `Closes #<番号>` は起票した issue を指す
- GitHub は、既定のブランチに merge された PR の本文の普通の文にある `Closes #<番号>` で issue を閉じる(コードブロック、HTML のコメント、引用の中では閉じない)
- #363(`shared=`)が先に merge される。実装はその main を取り込んでから始める

## Affected areas

- `internal/org/reserve.go`(`FeatureBinding.Issue`、Details の `issue=`、`sameFeature`、`featureDigestRefusal` か新しい拒否の句、`String`)とテスト
- `internal/org/feature.go`(`StartFeatureParams.Issue`、検証、`startFeatureLeaderParams`、`featureLeaderTask`)とテスト
- `internal/org/reseat.go`(leader の start に記録の `Issue` を渡す)とテスト
- `internal/org/mergeorder.go`(`MergeOrderBinding.Issue`)、`internal/cli/org_merge_order.go`(JSON)とテスト
- `internal/cli/org.go`(start の `--issue`、`checkOrgStartPlanInput`、status の行と JSON)とテスト
- `internal/org/prompts/leader.md` とそのテスト
- director と org の skill(4 写しずつ)、README、AGENTS.md、2 つの仕様

## Visual review

- ページ: `.harness/state/plan-visual/org-start-issue.html`。かんたんな図 1 枚(`overview`)と、詳しい図 2 枚(`detail-overview`、`detail-flow`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 3 枚の図を `shot` で撮って読んだ。はみ出しや重なりはなかった

## Design decisions

Critical forks: None

- 番号は director が `--issue` で明示して渡す。start が gh で印の行を探す案は、start に gh と信頼の確かめを持ち込むので採らない。director は着手を決めた時点で issue の番号を知っている
- 番号は結びつきに記録する。`ralph org reseat` は leader のタスクを記録から作り直すので、記録しないと立て直した leader が番号を失う
- やり直しの start は `--issue` が記録と同じときだけ通す。違う値を黙って通すと、記録とタスクが食い違い、あとの reseat で番号が変わる
- PR の本文は `Closes #<番号>`(同じ repository の短い形)。`<owner>/<repo>#<番号>` の形は start に origin の読み取りを持ち込むうえ、PR が origin 以外にできる場合は `ralph org merge` がすでに断るので要らない

## Acceptance criteria

- [ ] AC1: `ralph org start --plan ... --feature ... --issue 12` で立てた org の leader のタスクに `- issue: #12(PR の本文に Closes #12 を書く)` の行が入り、予約の記録の Details の最後に `issue=12` が入る
- [ ] AC2: `--issue` なしで立てた org のタスクと Details は、この変更の前と同じ(バイト単位)
- [ ] AC3: `--issue` は、`--plan` なし、空、`0`、先頭の 0、`#12`、数でない値、11 桁以上を、何も作らずに断る
- [ ] AC4: `ralph org reseat` が立て直した leader のタスクに、記録の issue の行が入る
- [ ] AC5: disband していない org の `start --plan` のやり直しは、`--issue` が記録と同じ(どちらもなしを含む)なら通り、違えば記録と今の値を名指して断る
- [ ] AC6: 壊れた `issue=` の値(数でない、2 回)の記録は issue なしと読み、結びつきは完全なまま。古い binary の書いた記録(`issue=` なし)も issue なし
- [ ] AC7: `ralph org status --org-id` は issue のある org に `issue: #<番号>` の行を出し、`--json` の feature に `issue` を出す。issue のない org の出力は変わらない
- [ ] AC8: leader の雛形の手順 14 が、`- issue:` の行があれば PR の本文の普通の文に `Closes #<番号>` を書くことを言い、行がないときの PR の本文の決まりを変えない(テストで固定。行がないときに閉じるキーワードを禁じる文がないことも確かめる)
- [ ] AC10: `ralph org merge-order --all --json` が、`--issue` で立てた org のブランチの `binding.issue` を、その org の disband のあとも出す。同じ org_id を別の機能で立て直したときは、ブランチごとの最後の結びつきの issue を出す。`--issue` なしのブランチには `issue` がない
- [ ] AC9: director skill が、issue の機能の org を `--issue` 付きで立てること、merge のあと issue が閉じたかを確かめ OPEN なら人に伝えることを言う。org skill が `--issue` を説明する。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る

## Implementation outline

1. S1 結びつき、start、merge-order の結びつき(AC1〜AC6、AC10 の org 側) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Binding|SameFeature|Reservation|StartFeature|FeatureLeaderTask|Reseat|Issue|MergeOrder'`
2. S2 CLI の flag、status、merge-order の JSON(AC3、AC7、AC10 の CLI 側) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgStart|OrgStatus|OrgMergeOrder|Help'`
3. S3 leader の雛形と skill と文書(AC8、AC9) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'RenderRolePrompt_Leader|Skill|DocsMatch' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Agents|Readme|Docs|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC9。Details の `issue=` が `shared=` のあとに来ること、古い binary の読み(知らない鍵から先を note と読む)で予約のパスとほかの結びつきが変わらないこと
- Documentation drift to check: director と org の skill(4 写し)、README、AGENTS.md、2 つの仕様、`ralph org start --help`
- Evidence to capture: start の `--issue` の拒否の文、status の行、leader のタスクの行(テストの出力で)

## Test plan

- Unit tests: `--issue` の検証(AC3 の各値)、Details の書き出しと読み(正しい値、数でない、2 回、`issue=` なし)、`sameFeature`(issue が同じ、違う、片方なし)、`featureLeaderTask` の行(あり、なしでバイト同じ)
- Integration tests: `StartFeature` で立てた org の記録とタスク、同じ start のやり直し(同じ issue は通る、違えば断る)、`ralph org reseat` の leader のタスク、CLI の `ralph org start --plan --issue`(fake の herdr)と `ralph org status` の行と JSON
- Regression tests: `--issue` なしの start、reseat、status の出力が今と同じ。#363 の `shared=` のある記録に `issue=` が続いても両方読める
- Edge cases: `--issue` と `shared=` が両方ある記録、`issue=` が `shared=` より前にある記録(手で書いた記録)、leader の雛形のテストで「行がないときの決まりは変えない」を固定、start → disband → merge-order の `binding.issue`、org_id の使い回し
- Evidence to capture: `go test` の出力、`./scripts/run-test.sh` の rc

## Risks and mitigations

- PR が origin 以外の repository にできると、`Closes #<番号>` がその repository の別の issue を指す: 機能の org の PR は origin にできる前提で、`ralph org merge` は origin の外の PR を断る。director は merge のあと issue が閉じたかを確かめ、閉じていなければ人に伝える
- leader が `Closes` をコードブロックや引用の中に書く、または書き忘れる: 雛形に普通の文に書くと明記し、director の merge のあとの確かめで拾う
- #363 との衝突(`reserve.go` の Details、skill の同じ節): #363 の merge のあとに main を取り込んでから実装する
- 古い binary が自己終了の補償で結びつきを書き戻すと `issue=` が落ちる: issue なしで読むだけで、そのあとの reseat の leader は Closes を書かない。director の merge のあとの確かめで拾う

## Rollout or rollback notes

- 新しい flag で、使わなければ挙動は変わらない。revert すると `--issue` が消え、記録の `issue=` は古い binary と同じく note として読まれる
- release ノート: `ralph org start --plan` に `--issue <番号>`。立てた org の leader が PR の本文に `Closes #<番号>` を書く。director skill は issue に起票した機能をこれで立てる。新しい skill は新しい binary が要る

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 2 件、どちらも計画に入れた。[MEDIUM] 1: org が disband したあとや director が入れ替わったあとに、merge 済みの機能の issue の番号を引く道がない(`ralph org status` は動いている結びつきしか出さず、merge-order の結びつきは issue を持たない) → `ralph org merge-order --json` の `binding.issue` を足し、director の merge のあとの確かめと引き継ぎはそれを読む(AC10)。[MEDIUM] 2: `--issue` がないときに閉じるキーワードを禁じると、`/pr` の `Related issue:` の決まりとぶつかる → 行がないときの決まりは変えない(AC8 を直した)。git の状態は助言の前後で同じ
- [x] Plan approved — 2026-11-22、系列の決まり(計画は推す案で自動承認)で承認した。人の目での承認はしていない。consult(consult-plan-issue)は「進めてよい」。補足の 2 点(`--issue` のない org で雛形の `Closes #__ISSUE__` を残さない一言、issue の行を digest に入れないことのコメント)を Scope に入れた
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
