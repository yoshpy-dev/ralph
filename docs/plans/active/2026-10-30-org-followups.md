# org-followups

- Status: Draft
- Approved: TBD
- Owner: Claude Code
- Date: 2026-10-30
- Related request: org: 台帳に follow-up を記録し、先行する PR の merge のあとに着手できる件を出す(https://github.com/yoshpy-dev/ralph/issues/270)
- Related issue: 270
- Type: feat
- Branch: feat/org-followups

## Objective

director が「先行する機能の PR の merge を待ってから人に聞く issue」を台帳に記録できるようにする。記録は追加・一覧・完了ができ、director の入れ替えや会話の要約のあとも残る。先行するブランチの PR が merge されたら、`ralph status` と `ralph org merge-order` がその follow-up を「着手できる」と出す。あわせて `ralph status` が、走っている org の予約のパスを表と `--json` で出す。どれも LLM を使わずに状態を説明できる。

## Scope

- 台帳の記録: `followups.jsonl`(追記だけ)と `followups.lock`。`internal/org/followup.go` の store(追加、完了、読み出しと畳み込み、壊れた行と無視した event の数え)
- 動詞: `ralph org followup add --issue <番号> --after-branch <ブランチ> [--note <文>]`、`ralph org followup list [--all] [--json]`、`ralph org followup done <id> [--note <文>]`
- `ralph org merge-order`: open の follow-up の先行ブランチの PR を、今の PR の引き方(gh、origin に固定)で引き、表のあとに follow-up の節(`ready` / `waiting` / `unknown`)を出す。`--json` にも入れる
- `ralph status`: 走っている org ごとに予約のパスを出す(表と `--json`、座席がなく予約だけがある org も)。台帳の follow-up を出し、台帳の記録(`pr_merged` か `worktree_cleaned`)で先行ブランチの merge がわかるものを「着手できる」とする(D2)
- 文書: `/org` skill(動詞の行、follow-up の使い方)、`/director` skill(issue を待たせるときに記録し、merge の報告のときに着手できる件を人に伝え、決まったら完了にする)とそれぞれの写し、README、AGENTS.md の地図、仕様の注、`ralph org` の help

## Non-goals

- 着手できる follow-up を、見張り(`ralph org watch --all`)が受信箱の件にして人に送ること(#271 で director の要件として扱う)
- follow-up から org を立てること、issue の中身を GitHub から読むこと(issue の番号を記録するだけ)
- `ralph status` で gh を走らせること(D2)
- squash merge や rebase merge を git で見分けること
- 既存の記録(manifest、inbox、director)の形の変更

## Assumptions

- 先行するブランチは、機能の org のブランチ(`ralph org merge-order` が並べるもの)が主だが、どのブランチ名でも記録できる
- `ralph org merge` は merge したときに manifest に `pr_merged`(Details に `branch=`)を、`ralph org cleanup` は GitHub で merge された PR の片付けのあとに `worktree_cleaned`(同じく `branch=`)を書く。どちらも gh で merged を確かめたあとの記録
- 台帳の lock は記録ごとに別のファイルで、ほかの lock と同時に持たない(`internal/org/inbox.go` と同じ規則)

## Affected areas

- 新しい `internal/org/followup.go` とテスト
- 新しい `internal/cli/org_followup.go` とテスト、`internal/cli/org.go`(`followup` の動詞の登録、`ralph org` の help、`--org-id` を断る動詞の一覧)
- `internal/org/mergeorder.go` と `internal/cli/org_merge_order.go`(follow-up の節と JSON)、テスト
- `internal/cli/status.go`(予約と follow-up の表示と JSON)、テスト
- `.claude/skills/org/SKILL.md`・`.claude/skills/director/SKILL.md` とそれぞれの写し、README.md、AGENTS.md、`docs/specs/2026-10-07-org-multi-org-director.md`

## Visual review

## Design decisions

- D1 記録の形: `followups.jsonl` に event を 1 行ずつ追記する(`added` と `done`)。`internal/org/inbox.go` の store と同じ作り(自分の lock ファイル `followups.lock`、lock の中で読み直してから追記、ID は `f<N>` で壊れた行も含めた最大の番号の次、壊れた最後の行の前に改行を足す、畳み込みは純粋な関数で、壊れた行と無視した event を数える)。状態は open → done の 2 つ。done は 2 度目でも何も変えずに 0 で終わる(inbox の ack と同じ)
- D2 `ralph status` の merged の判定: `ralph status` は gh もネットワークも使わない(help に書いてあり、テストが固定している)ので、この規則を守る。台帳の manifest に先行ブランチの `pr_merged` か `worktree_cleaned` があれば「着手できる」(`ready (recorded)`)、なければ「待ち」(`waiting`)と出し、gh で確かめるには `ralph org merge-order` を打つ、と 1 行で添える。`ralph org merge-order` は今と同じく gh で PR を引き、merged なら `ready`、まだ open なら `waiting`、merge されずに closed なら `closed`(理由つき)、gh がないか引けなければ `unknown`(理由つき)と出す。issue は「status と merge-order は merged の判定を gh で」と書くが、status は gh なしを help とテストで固定しており、merge の保証の表示でもすでに「doctor が gh で確かめ、status は記録を読む」分担を取っている。status が読む `pr_merged` と `worktree_cleaned` は、どちらも gh で merged を確かめたあとの記録なので、status の「着手できる」も gh 由来の事実に基づく。差は status 自身が gh を叩くかどうかだけで、表と JSON にどの道で判定したか(status は `recorded`、merge-order は `gh`)を出して #271 が区別できるようにする(consult)
- D5 どの merge を数えるか(codex の計画助言 MEDIUM): ブランチ名は使い回されうるので、follow-up を記録した時刻より後の merge だけを数える。status は台帳の `pr_merged` か `worktree_cleaned` の event の時刻が follow-up の `added` の時刻以後のもの、merge-order は gh で PR の `mergedAt` も読み、`added` 以後に merged のものを `ready` にする。前に merge された同じ名前のブランチの記録や PR では着手できるにしない。記録より前にもう merge されたブランチの follow-up は待つ相手がいないので、`add` の help と `/director` skill に「merge 済みなら follow-up にせず、すぐ人に聞く」と書く。PR の番号で結びつける案は、PR ができる前に記録する場合に「まだ結びついていない」状態と結びつける手順が要るので採らない
- D3 動詞の名前と引数: `ralph org followup add|list|done`。`--issue` は正の整数、`--after-branch` は空でなく、空白と制御文字を含まず、`-` で始まらない名前(照合は台帳の `branch=` と `gh pr list --head` への文字列の一致だけなので、`git check-ref-format` の規則を Go で持たない。consult)、`--note` は受信箱の note と同じ検証(1 行、500 字まで、制御文字なし)。`list` は既定で open だけ、`--all` で done も、`--json` で機械向け。`--org-id` は断る(inbox と同じ)
- Critical forks: D2(`ralph status` で gh を叩くか、台帳の記録で判定するか)。ユーザーの「計画の分かれ目は consult に聞いて自分で選ぶ」の決めに従い、consult と相談して台帳の記録で判定する案を選んだ(根拠は D2)
- D4 `ralph status` の予約: `org.RunningOrgs` と `org.ActiveReservation` で、走っている org ごとに `reserved: <パス>, …` を出す。座席がなく予約だけがある org も 1 つの組として出す。JSON は `statusOrgJSON` に `reservation`(配列、空なら空の配列)を足し、top level に `followups` を足す(空のときと同じ形。今のテストが固定する)

## Acceptance criteria

- [ ] AC1 `ralph org followup add --issue <N> --after-branch <B> [--note <文>]` が follow-up を台帳に記録して ID を出し、`list` が open の件を(`--all` で done も)表と `--json` で出し、`done <id>` が完了にする。記録は `followups.jsonl` に残り、プロセスや director が替わっても読める
- [ ] AC2 記録は自分の lock(`followups.lock`)の中で読み直してから追記し、壊れた行や知らない event は数えて警告し、ID は壊れた行を含めて重ならない
- [ ] AC3 `ralph org merge-order` が、open の follow-up の先行ブランチの PR を gh で引き、follow-up の記録より後に merged なら `ready`、まだ open なら `waiting`、merge されずに closed なら `closed`(理由つき)、gh がないか引けなければ `unknown`(理由つき)として、表のあとの節と `--json`(判定の道 `gh`)に出す。終了コードは今と同じ
- [ ] AC4 `ralph status` が open の follow-up を出し、台帳に follow-up の記録より後の先行ブランチの `pr_merged` か `worktree_cleaned` があれば「着手できる」と出す。表と JSON に判定の道(`recorded`)を出す。gh は走らせない(今の gh なしのテストが通る)
- [ ] AC8 同じ名前のブランチの前の merge(記録より前の `pr_merged`、前に merged の PR)では「着手できる」にならない(D5)
- [ ] AC5 `ralph status` が、走っている org の予約のパスを表と `--json` に出す(予約だけの org も)。空のときと中身のあるときの JSON の形は同じ
- [ ] AC6 `/org` skill と `/director` skill(と写し)、README、AGENTS.md、仕様の注、`ralph org` の help が follow-up の動詞と使い方を書き、写しの検査と文を固定するテストが通る
- [ ] AC7 LLM を使わずに follow-up の状態を説明できる: `followup list`、`ralph status`、`ralph org merge-order` の出力だけで、どの issue がどのブランチ待ちで、着手できるか・できないか・わからないか(理由)がわかる

## Implementation outline

1. S1 台帳の記録: `internal/org/followup.go`(store、event、畳み込み、検証)とテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Followup'`
2. S2 動詞: `internal/cli/org_followup.go`(add・list・done)、`internal/cli/org.go` の登録と help、テスト — verify: `test -z "$(gofmt -l internal/cli internal/org)" && go vet ./internal/cli/ ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgFollowup|OrgHelp|OrgInbox'`
3. S3 merge-order: follow-up の節と JSON(`internal/org/mergeorder.go`、`internal/cli/org_merge_order.go`)、テスト — verify: `test -z "$(gofmt -l internal/cli internal/org)" && go vet ./internal/cli/ ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'MergeOrder|Followup' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgMergeOrder'`
4. S4 status: 予約と follow-up の表示と JSON(`internal/cli/status.go`)、テスト — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Status'`
5. S5 文書: `/org`・`/director` skill と写し、README、AGENTS.md、仕様の注、文を固定するテスト — verify: `./scripts/check-template-purity.sh && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill' && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`./scripts/check-template-purity.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC8。特に AC4 の gh を走らせないこと、AC2 の lock と ID、AC5 の JSON の形
- Documentation drift to check: `ralph org` の help と skill の動詞の一覧、`ralph status` の help(gh を使わないこと、表と JSON の新しい項目)、仕様の安全の不変条件の注
- Evidence to capture: verify の report、static verify の出力

## Test plan

- Unit tests: store の追加・完了・読み出し、ID の採番(壊れた行を含む)、壊れた最後の行、二度目の done、知らない event、検証(issue の番号、ブランチ名、note)
- Integration tests: CLI の add・list・done(表、`--all`、`--json`、終了コード、`--org-id` を断る)、merge-order の follow-up の節(偽の gh で merged・open・gh なし・gh の失敗)、status の予約と follow-up(台帳の `pr_merged`・`worktree_cleaned` で ready、gh を走らせないこと)
- Regression tests: 今の merge-order と status と inbox のテストが変えずに通る。status の JSON の空と中身ありの形の一致
- Edge cases: 同じブランチを待つ follow-up が 2 つ、先行ブランチの PR が closed(merged ではない: `closed` と理由)、follow-up の追加より前に merge 済みのブランチ(着手できるにならない)、同じブランチの古い merged の PR と新しい open の PR(着手できるにならない)、lock の取り合い
- Evidence to capture: test の report、変異(ready の判定を外す、lock を外す、status で gh を呼ぶ)が落ちること

## Risks and mitigations

- `ralph status` の「着手できる」と `ralph org merge-order` の `ready` が食い違う(GitHub で merge したが cleanup をまだ打っていない): status は「台帳の記録で」と書き、gh で確かめるには merge-order を打つ、と 1 行で添える(D2)
- merge-order の gh の呼び出しが follow-up の分だけ増える: 先行ブランチが今の並べる対象と同じなら同じ結果を使い、違うものだけを引く
- `status.go` は 901 行、`mergeorder.go` は 1291 行で 800 行の目安を越えている: 新しい部分は別のファイル(`internal/cli/status_followup.go`、`internal/org/mergeorder_followup.go`)に置く
- 台帳の新しいファイルを、古い binary は読まない: 追加だけで、既存の記録は変えない

## Rollout or rollback notes

- 新しい動詞と表示の追加だけで、既存の記録の形は変えない。戻すときはこの PR を revert し、`followups.jsonl` は残っても読まれないだけ
- release のノート: `ralph org followup` を足した。`ralph status` が予約と follow-up を、`ralph org merge-order` が着手できる follow-up を出す

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult の判定は「進めてよい」。D2 は台帳の記録で判定する案で可とし、直したのは、D2 に issue の文との差と merge の保証の表示と同じ分担の根拠を書き、表と JSON に判定の道(`recorded` / `gh`)を出すこと、D3 のブランチ名の検証を空・空白・制御文字・`-` 始まりの拒否に縮めること、AC3 に merge されずに closed の PR を理由つきで出すこと。plan advisory: ran with codex。MEDIUM 1 件(ブランチ名の使い回しで、前の merge の記録や PR が新しい follow-up を着手できるにしてしまう、番号のいちばん大きい PR を選ぶので待つ相手が途中で替わる)は、follow-up の記録より後の merge だけを数える規則(D5、AC8)で反映した。PR の番号で結びつける案は採らない理由を D5 に書いた)
- [ ] Plan approved
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
