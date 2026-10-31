# Walkthrough: org-followups(merge を待つ issue を台帳に記録し、着手できるかを `ralph status` と `ralph org merge-order` に出す)

- Date: 2026-11-01
- Plan: docs/plans/active/2026-10-30-org-followups.md(/pr で archive に移す)
- Issue: #270
- PR: #294
- Branch: feat/org-followups(この walkthrough の前の HEAD は edf7d5b5。merge を除く 59 commits と、origin/main(#293)を取り込んだ merge commit edf7d5b5)
- Base: a4122131(origin/main、#293 の merge)。branch を切った元は 1cc84ffa
- 読む範囲: `git diff origin/main...HEAD` は 45 files、+7466/-135。Go は 22 files(+5334/-82)で、本番コードは 11 files(新しいファイルは `internal/org/followup.go` 555 行、`internal/org/mergeorder_followup.go` 371 行、`internal/cli/org_followup.go` 281 行、`internal/cli/status_followup.go` 189 行)。残りはテストと、skill の 4 つの写し、README、AGENTS.md、仕様、quality-gates、tech-debt、pipeline の報告
- 後工程: パイプラインは 2 回。cross-review の cycle 1 が ACTION_REQUIRED を 1 件出し(follow-up だけの台帳で `gh_available` が false)、F5 で直して回し直した。cycle 2 は所見 0

```
何を変えるか → 読む順 → 台帳の記録 → 動詞 → merge-order → status → どの merge を数えるか → 文書 → テスト → 直しの経過 → 確かめたことと確かめていないこと
```

## 1. この PR が何か

director と人が「この issue は機能 B の PR が merge されてから聞く」と決めても、これまでその約束は director の会話の中にしか残らなかった。この PR は、その約束を台帳の追記のファイル `followups.jsonl` に記録し、待っているブランチが merge されたら `ralph status` と `ralph org merge-order` が「着手できる」と出すようにする。記録は台帳にあるので、director の入れ替えや会話の要約のあとも残る。

`ralph status` は gh を打たない約束を help とテストで固定しているので、台帳の `pr_merged` と `worktree_cleaned` で判定する(判定の道 `recorded`)。`ralph org merge-order` は gh で PR を引いて判定する(判定の道 `gh`)。merge の保証の表示と同じ分担で、表と JSON の SOURCE の列でどちらの道かを示す(計画の D2)。

あわせて `ralph status` は、走っている org ごとに予約のパスを `reserved:` の行と JSON の `reservation` に出す。座席がなく予約だけがある org も 1 つの組として出す。

## 2. 読む順

1. `internal/org/followup.go`: 記録の形、lock、畳み込み、検証、`FollowupRecordedMerge`
2. `internal/cli/org_followup.go`: 3 つの動詞
3. `internal/org/mergeorder_followup.go` と `internal/org/mergeorder.go` の差分: merge-order の節
4. `internal/cli/status_followup.go` と `internal/cli/status.go` の差分: status の節と予約
5. skill(`.claude/skills/org/SKILL.md` の「merge を待つ issue」、`.claude/skills/director/SKILL.md` の手順)
6. テスト

## 3. 台帳の記録(`internal/org/followup.go`)

- 作りは受信箱の store(`internal/org/inbox.go`)と同じ。自分の lock ファイル `followups.lock` を `withFileLock` で取り、lock の中で読み直してから 1 行を追記する(`Add` 147 行、`Done` 181 行、`appendLocked` 317 行)。最後の行が切れていれば前に改行を足す。
- event は `added` と `done` の 2 つ。状態は open → done で、2 度目の `done` は何も変えずに成功する(受信箱の ack と同じ)。
- ID は `f<N>` で、壊れた行に読める番号があればそれも数えて最大の次を振る(`foldFollowups` 352 行、`seeN` 374 行)。
- 畳み込みは純粋な関数で、壊れた行と、知らない event や検証に落ちる `added`(手で書いた `-` 始まりのブランチなど)を数えて捨てる。CLI は数を stderr に警告し、JSON の `corrupt_lines` と `ignored_events` に出す。
- 検証(`ValidateFollowupAdd` 219 行): `--issue` は正の整数、`--after-branch` は空でなく、空白と制御文字を含まず、`-` で始まらず、有効な UTF-8。`--note` は受信箱の note と同じ規則(1 行、500 字まで、制御文字なし)。

## 4. 動詞(`internal/cli/org_followup.go`)

- `ralph org followup add --issue <N> --after-branch <B> [--note <文>]` は ID を出す。`list` は open だけ、`--all` で done も(そのとき `DONE_AT` と `DONE_NOTE` の列を足す)、`--json` で機械向け。`done <id> [--note]` は知らない ID なら終了コード 1。
- 3 つとも `--org-id` を断る。記録は台帳全体のもの(受信箱と同じ)。
- `--note` の検査は台帳を解決する前に走る(self-review L5)。
- 見張りがなくても打てる。sync-docs で `docs/quality/quality-gates.md` と `/org` skill の、見張りなしで打てる動詞の一覧に足した。

## 5. merge-order の節(`internal/org/mergeorder_followup.go`)

- `MergeOrderParams.Followups` を merge-order だけが立てる。`ralph org cleanup` が同じ関数を呼ぶときは立てないので、cleanup の gh の呼び出しは変わらない。
- open の follow-up の after-branch ごとに 1 回、`gh pr list --head <B> --state all` を `mergedAt` 付きで打つ(`lookupFollowupPRs` 240 行)。gh の呼び出しは `ghBranchPRs` にまとめ、merge と cleanup が使う `lookupMergePR` と共有した(self-review L1)。origin の repository に固定し、fork の PR は除く。
- 判定(`followupMergeState` 285 行): 記録より後に merged の PR があれば `ready`、open の PR があれば `waiting`、merge されずに閉じたものだけなら `closed`、gh がないか引けない・中断・`mergedAt` が読めない・上限 1,000 件・`added_at` が読めないときは `unknown`。`closed` と `unknown` は DETAIL に理由を書く。
- 表は並べる対象の表のあとに `follow-ups:` の節で、列は `ID ISSUE AFTER_BRANCH STATE SOURCE PR DETAIL`。JSON の `followups` は `{items, corrupt_lines, ignored_events, error}` の object で、`ralph status` と外側の形が同じ。
- follow-up の検索が gh を見つけたら `gh_available` を true にする(F5、cross-review cycle 1)。並べる対象の検索が出した true を消すことはない。
- 並べる対象、ほかの note、終了コードは変えていない。

## 6. status の節と予約(`internal/cli/status_followup.go`)

- `readStatusFollowups`(126 行)は `followups.jsonl` と manifest の event だけを読み、gh を打たない。テスト `TestStatusCmd_FollowupsRunNoGH` が、PATH に呼び出しを記録する gh だけを置いて確かめる。
- `statusFollowup`(153 行)は `org.FollowupRecordedMerge` で `ready (recorded)` か `waiting` を出す。`waiting` の DETAIL は「gh で確かめるには `ralph org merge-order`」と添える。`added_at` が読めない件は `unknown`。
- 予約(`withStatusReservations` 74 行): 走っている org の予約を組に足してから空の判定をするので、予約だけの台帳も "no org runtime state found" にならない。
- follow-up は台帳全体のものなので `--org-id` では絞らない。記録が読めないときは status を失敗させず、stderr に警告し、JSON の `error` に理由を入れる。

## 7. どの merge を数えるか(計画 D5、self-review M1)

- ブランチ名は使い回されうるので、follow-up を記録した時刻より後の merge だけを数える。merge-order は gh の `mergedAt` と比べ、status は event の時刻と比べる。
- `worktree_cleaned` の時刻は片付けを打った時刻で、merge の時刻ではない。そのため「merge(`pr_merged`)→ 片付けが断られる → follow-up を記録 → 片付けが通る」で、status だけが早く ready を出していた。`FollowupRecordedMerge`(470 行)は、記録より前の同じブランチの `pr_merged` と head が同じ `worktree_cleaned` を数えない。head が違えばブランチの使い回しなので数える。
- 残る形は、GitHub で記録より前に merge され(`pr_merged` がない)、記録のあとで片付けられたブランチ。status では見分けられないので、help と skill に「ready と waiting で割れたら merge-order の答えを使い、merge-order が unknown なら、その理由を添えて status の答えを使う」と書いた。片付けが PR の `mergedAt` を記録する直しは既存の記録の形を変えるので tech-debt に送った。
- 記録より前に merge 済みのブランチは待つ相手がいないので、`add` の help と `/director` skill に「follow-up にせず、すぐ人に聞く」と書いた。

## 8. 文書

- `/org` skill: 動詞の表に 3 行、`merge-order` と `status` の行に follow-up の節と予約の行、新しい節「merge を待つ issue」(記録、2 つの判定の道、割れたときの規則、`followups.lock`)。#268 の「予約外の文書の follow-up」とは別物だと書いた。
- `/director` skill: 人と issue を後に回すと決めたら記録し、merge を報告するときに着手できる件を人に伝え、人が決めたら `done` にする手順。記録は台帳にあるので director が替わっても残ると書いた。
- README、AGENTS.md の `internal/org/` の行、仕様 FR-9 の「2026-10-30 に決めたこと(issue #270)」の注、`docs/quality/quality-gates.md` とその template の写し。
- tech-debt: M1 の残りの形、`status.go` と `mergeorder.go` の大きさ、テストの穴 1〜10、この branch が Trigger に当たった行への持ち越しの注。

## 9. テスト

- `internal/org/followup_test.go` と `followup_recorded_merge_test.go`: store の追加・完了・読み出し、ID の採番、切れた最後の行、2 度目の done、知らない event、検証、同時の add、`FollowupRecordedMerge` の時刻と head の判定(event の順を入れ替えても答えが同じ)。`pr_merged` は本物の `recordMerge` を通して作る。
- `internal/org/mergeorder_followup_test.go`: 偽の gh で merged・open・closed・gh なし・失敗・上限・中断、`gh_available` の OR。
- `internal/cli/org_followup_test.go`、`org_merge_order_followup_test.go`、`status_followup_test.go`: 表、`--all`、`--json`、終了コード、`--org-id` の拒否、gh を打たないこと、予約だけの台帳、JSON の形が空と中身ありで同じこと。
- `internal/org/skill_followup_test.go`: skill の 4 つの写しの文を固定する。
- help の動詞の一覧を固定していた既存のテスト 3 本(`org_approve_test.go`、`org_director_test.go`、`org_regate_test.go`)と、`TestMergeOrder_ResultJSONShape` の key を合わせた。

## 10. 直しの経過

- self-review cycle 1 の run 1(MEDIUM 1・LOW 9): M1 を F1(コード)と F2(文書)で直した。F1 では gh の呼び出しの共有(L1)、判定と DETAIL の共有(L2)、JSON の形(L3)、event の書式の約束(L4)、テストの分割(L9)も直し、F2 では文言(L5)、`unknown` の理由(L6)、`list --all` の列(L7)を直した。
- run 2(LOW 4): F3 で、割れたときの規則を ready と waiting に絞り(R2-1)、`merged_at` の意味を書き(R2-2)、`mergeorder.go` の大きさを tech-debt に送り(R2-3)、テストの JSON 名の重複を外した(R2-4)。
- verify cycle 1(pass): 計画の D5 を R2-1 の形に直して承認し直し(I1)、merge-order の表に SOURCE の列を足した(F4、I2)。
- cross-review cycle 1(ACTION_REQUIRED 1): F5 で `gh_available` を直し、全体を回し直した。
- self-review cycle 2(LOW 2): F6 でコメントと help の折り返しを直し、台帳の数と注は sync-docs に回した。

## 11. 確かめたことと確かめていないこと

- 確かめたこと: verify は cycle 1・2 とも pass で、HEAD の binary を一時の台帳で打った(AC1〜AC8、`gh_available` の 13 通り、見張りなしの動詞)。test は cycle 2 で `./scripts/run-test.sh` rc 0(1,578 秒)、`-race -count=3` で 306 件 PASS、変異は ready の判定・lock・status からの gh・M1 の除外・`gh_available` の OR で落ちた。origin/main を merge したあとに `./scripts/run-static-verify.sh` rc 0 と、絞った go test が ok。
- 確かめていないこと: 実物の gh での merge-order の follow-up(テストは偽の gh)。merge のあとの `./scripts/run-test.sh` 全体(CI で見る)。テストの穴 1〜10(tech-debt の 1 行)。
