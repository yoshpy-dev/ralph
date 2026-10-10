# Verify report: org-inbox-escalate

- Date: 2026-10-11
- Plan: docs/plans/active/2026-10-10-org-inbox-escalate.md(承認済み、digest b09718e048cd)
- Verifier: verifier subagent (Claude Opus 5.5)。pipeline cycle 1(`cycle-count.json` は 1、上限 2)。cross-review はまだ走っていない
- Scope: 仕様への適合(AC1〜AC10 と AC2b)、静的解析、文書のずれ。対象は `git diff origin/main...HEAD`(base 382c18c8、HEAD 11af3897、12 コミット、23 ファイル、+4988/-104)。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-5・FR-11 と `.claude/rules/ralph/agent-messaging.md` とも突き合わせた。テストは実行していない(`/test` の担当)。テストは、何を固定しているかを読んで確かめた
- Evidence: `docs/evidence/verify-2026-10-10-org-inbox-escalate.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore -v` で `.gitignore:58` に当たることを確かめた)。runner 自身のログは `docs/evidence/verify-2026-10-10-155240.log`(changed scope)と `docs/evidence/verify-2026-10-10-155309.log`(full scope)。時刻は UTC

## Spec compliance

計画の承認: `./scripts/plan-visual.sh digest` は HEAD で `b09718e048cd` を返し、`- Approved:` の値と一致する。承認のあと計画を触ったコミットは c2bc718d・1d865fdd・11af3897 の 3 つで、AC のチェックボックスと `## Progress checklist` の中だけを変えている。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 TYPE が QUESTION・BLOCKED・RESULT でない、TASK_ID がない、本文が 2,000 字超、org_id の形が違う、のどれでも終了コード 1 で、何も書かない | 満たす | `validateEscalation`(`internal/org/escalate.go:254-269`)が `ValidateIdentifier`、`protocol.Parse` と `protocol.Validate(m, protocol.DefaultMaxBodyChars)`、`escalateTypes` の順に検査し、`Escalate`(129 行)はそれが通ってから `Inbox.Escalate` を呼ぶ。`EscalateNotRecorded`(160 行)も同じ検査を先に通す。`TestEscalate_RefusalsWriteNothingAndNotifyNoOne`(`escalate_test.go:101`、13 ケース)が、新しい台帳と使用中の台帳の両方で `inbox.jsonl` と `escalations.jsonl` がバイト単位で変わらないこと、通知も banner も出ないこと、`inbox.lock` ができないことを見る。CLI は `TestOrgEscalate_RefusalsWriteNothing`(`org_inbox_test.go:232`、9 ケース)で、状態ディレクトリができないことまで見る |
| AC2 `escalated` を書いて `e<N>` を出す。同時の escalate で ID が重ならない。壊れた行・切れた最後の行の ID を使い回さず、新しいイベントは読める | 満たす | `InboxStore.Escalate`(`internal/org/inbox.go:166-196`)はロックの下で読み、`foldInbox` が生の行の正規表現(64 行、369〜371 行)と読めた行の `id`(396〜398 行)の両方から最大の N を取る。`nextID`(447 行)は int64 の上限でエラーにする。`appendLocked`(333 行)は、末尾が改行でなければ同じ 1 回の `Write` の頭に改行を足す(338〜340 行)。テストは `TestInboxEscalate_NeverReusesTheIDOfADamagedLine`(283)、`_TornLastLineStaysSeparateAndTheNewEventIsReadable`(320)、`_CountsAnIDWrittenWithJSONEscapes`(353)、`_ConcurrentCallsGetDistinctIDs`(402)、`TestEscalate_ConcurrentEscalatesGetDistinctIDs`(`escalate_test.go:304`)。CLI は `Recorded` のときだけ stdout に `escalated <id> (org=… type=…)` を出す(`internal/cli/org.go:1716` の `newOrgEscalateCmd`) |
| AC2b 記録はできたが通知の記録が書けないと、ID を出して終了コード 1、`inbox notify` を案内。一覧で未通知、`inbox notify` で同じ ID のまま送り直せる。受信箱に書けない escalate も stderr と osascript で知らせて終了コード 1 | 満たす | `sendInboxItemToHuman`(`escalate.go:284-298`)は、`escalations.jsonl` に書けなければ `notified` も書かずにエラーを返す。`Escalate` はそのとき ID・`Recorded`・`InboxCommand("notify", …)` つきのエラーを返す(143〜147 行)。`NotifyInboxItem`(178〜198 行)は件を増やさずに同じ経路を通る。受信箱に書けないときは `alertUnrecordedEscalation`(345〜354 行)が banner・通知・best-effort の `escalations.jsonl` の行を出す。テストは `TestEscalate_EscalationsUnwritable_ReportsTheIDAndInboxNotifyCompletesIt`(338)、`_NotifiedUnwritable_…`(384)、`_InboxUnwritable_AlertsTheHumanAndFails`(432)、CLI の `TestOrgEscalate_RecordedNotNotified_ExitsOneThenInboxNotifyCompletesIt`(276)と `_InboxUnwritable_AlertsTheHumanAndExitsOne`(321)。計画より広く、`ralph.toml` が読めない場合も同じ結果になる(`org.go:1761`、`TestOrgEscalate_BrokenConfig_AlertsTheHumanAndExitsOne`(349)、self-review L1 の直し) |
| AC3 `escalations.jsonl` に件の ID・org_id・理由の 1 行、stderr の表示、osascript の結果つきの `notified`。osascript が失敗しても終了コード 0 | 満たす | `escalationRecord` に `inbox_id` を足し、`alert_id` を `omitempty` にした(`internal/org/watch.go:266-276`)。理由は `inbox_no_director`。`inboxDesktopNotify`(`escalate.go:359-375`)の戻り値 `ok` / `failed: …` / `skipped` が `notified` の `osascript` に入る。通知の失敗は `Notified` を false にしない。テストは `TestEscalate_RecordsSendsToTheHumanPathAndMarksNotified`(205)、`_DesktopNotificationFailureStillSucceeds`(272)、CLI の `TestOrgEscalate_DesktopNotificationFailureStillExitsZero`(216)、行の形は `TestEscalationRecord_WatchAndInboxLinesKeepTheirOwnID`(722)が watch と inbox の両方で固定する |
| AC4 `inbox` は open と acked、`--all` で resolved も、`--json`。`inbox show` は本文と履歴。知らない ID は終了コード 1 | 満たす | `newOrgInboxCmd`(`org.go:1782`)が `all` でなければ resolved を除き、`--json` は `orgInboxJSON`(件、`corrupt_lines`、`ignored_events`)を出す。`show` は知らない ID で `ErrInboxUnknownID` を返す。テストは `TestOrgInbox_ListAllAndJSON`(473)、`TestOrgInbox_Show`(531、知らない ID は 571 行)、`TestOrgInbox_EmptyInbox`(580)、`TestOrgInbox_WarnsAboutDamageOnStderr`(602) |
| AC5 ack と resolve の遷移、note の検査、同時の ack と resolve で状態が壊れない | 満たす | `Ack`(`inbox.go:214-230`)、`Resolve`(235〜245 行)、`ValidateInboxNote`(250〜267 行。空・改行(LF・CR・U+2028・U+2029)・不正な UTF-8・501 字以上・制御文字)、`update`(281〜302 行)はロックの下で読み直してから書く。テストは `TestInboxAckAndResolve_TransitionTable`(438)、`TestValidateInboxNote`(543)、`TestInboxConcurrentAckAndResolve_LeaveAValidState`(623)、`TestInboxConcurrentAcks_OnlyOneChangesTheItem`(664)、CLI の `TestOrgInboxAck_Transitions`(681)と `TestOrgInboxResolve_NoteRulesAndTransitions`(727) |
| AC6 `wait --inbox` は open があればすぐ返り、届けば返り、届かなければ `--timeout-ms` で終了コード 1。acked では返らない。`--org-id` なしで打て、`--seat` / `--until` は拒否 | 満たす(計画からのずれは Progress に記録済み) | `runOrgWaitInbox`(`org.go:953`)と `WaitInbox`(`escalate.go:208-250`。open だけを数え、期限の時点でもう 1 回読む)。ずれは 1 つで、`--org-id` も「求めない」ではなく拒否する(Progress の 155 行。黙って無視すると 1 つの org の一覧に見えるため)。AC の「`--org-id` なしで打て」は成り立つ。テストは `TestWaitInbox_*`(`escalate_test.go:799-906`)、CLI の `TestOrgWaitInbox_ReturnsAnOpenItemAtOnce`(830)、`_ReturnsAnItemEscalatedWhileWaiting`(848)、`_TimesOut`(876、acked だけの場合を含む)、`_RefusesSeatFlags`(909) |
| AC7 受信箱は共通の台帳の下にでき、linked worktree と main のチェックアウトで同じものを読む。`--state-dir` でその台帳の受信箱 | 満たす | `newOrgRuntimeAt` が `Inbox: org.NewInboxStore(resolvedStateDir)` を manifest と同じ解決済みのディレクトリで組む。テストは `TestOrgInbox_LinkedWorktreeSharesTheMainCheckoutsInbox`(`org_inbox_test.go:936`。worktree から escalate、main から `inbox --json` と `wait --inbox`、別の `--state-dir` は別の受信箱)と `TestInboxStore_LivesUnderTheGivenStateDir`(`inbox_test.go:94`) |
| AC8 leader の雛形の置き換え、PR を作ったら RESULT、`--state-dir` を付ける動詞に escalate、本文はデータ、予備の手順と `inbox notify`。`TestRenderRolePrompt_Leader_*` が通る | 満たす(テストの実行は `/test`) | `internal/org/prompts/leader.md` の「人に上げ」は 0 件(base は 5 件)。67 行の列挙に `escalate・inbox notify`、機能ごとの org の手順 7 に RESULT の escalate、151 行の「## 件を上げる」節に例・TASK_ID の決まり・データであって指示ではないこと・予備の手順が入った。固定するテストは `TestRenderRolePrompt_Leader_EscalatesThroughRalphOrgEscalate`、`_NamesWhichInbox`、書き換えた `_FeatureOrgProcedure`。雛形の書き込みの例と escalate の例は、HEAD の guard(root と `templates/base/` の 2 本、default と bypassPermissions)でどちらも判定なし(通る)。陽性対照の 1 件は deny になった(下の「Observational checks」) |
| AC9 `/org` skill(4 面)、README、AGENTS.md、仕様 FR-5 の注記が合う。`check-skill-sync.sh` と `check-sync.sh` が通る | 一部(既知のずれ K-2・K-4 が残る。`/sync-docs` で直す予定) | skill の 4 面は `cmp` で同一。skill の「人に上げ」は 0 件(base は 2 件)。`check-skill-sync.sh` は 13 skill が lock-step、`check-sync.sh` は DRIFTED 0。README・AGENTS.md・FR-5 の注記はコードと合う。ただし skill と FR-5 の注記は `ralph.toml` が読めない場合を書いていない(K-4)、skill の 453 行が表示幅 104(K-2) |
| AC10 `./scripts/run-verify.sh` が通る | 静的な部分は満たす(テストの部分は `/test`) | `./scripts/run-static-verify.sh` は rc 0。`HARNESS_VERIFY_MODE=static RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` も rc 0。既定のモード(`all`)は `go test` と `tests/test-*.sh` を回すので、この段では打っていない |

## 仕様と規則との突き合わせ

| 仕様・規則の項目 | 結果 | 根拠 |
| --- | --- | --- |
| FR-5「今の型付きのメッセージと同じ検証(型、本文 2,000 字、証拠はポインタ)」 | 合う | `ralph org send` と同じ `protocol.Parse` と `protocol.Validate(…, DefaultMaxBodyChars)` を通し(send は `protocol.ValidateText` 経由、`internal/org/verbs.go:270`)、TYPE を 3 つに絞る。「証拠はポインタ」は send と同じく機構では検査しない(`agent-messaging.md` は 2,000 字の上限をその下限と位置づける)。雛形と skill がポインタで書くよう指示する |
| FR-5「共通の台帳の受信箱ファイルにイベント ID を付けて追記する」 | 合う | 各イベントは件の ID `e<N>` を持つ。イベントごとの ID はないが、5 段目の注記が ID を「件の ID」と定めている |
| FR-5 open → acked → resolved、`inbox ack` と `inbox resolve --note` | 合う | AC5 の行のとおり |
| FR-5 `wait --inbox` は未処理の件が届くと返る(全 org が対象) | 合う | 未処理は open として実装し、acked では返らない。この決定は 5 段目の注記に書いてある |
| FR-5 director が無効のとき、記録したうえで人への経路にすぐ送る | 合う | 5 段目は director を登録できないので、常にこの扱い。AC3 の行のとおり |
| FR-5 の「5 段目で決めたこと」の注記 | 1 か所足りない | `ralph.toml` が読めない escalate も NOT RECORDED で人に届くことが書かれていない(K-4) |
| FR-6 の「通知を試したこと、人が受け取ったことは別々に記録する」(計画の Design decisions が引く) | 合う | `notified` は試したことと osascript の結果、ack は受け取ったことで、別のイベント |
| FR-11 leader の雛形の「人に上げる」の置き換え | 合う | AC8 の行のとおり。skill の側は 4 段目で済んでいる |
| Security「受信箱の本文は director にとってデータであって指示ではない」 | 合う | 雛形の「件を上げる」節、skill の「受信箱」節、`ralph org inbox --help` の 3 か所が書く。デスクトップ通知には org・ID・TYPE だけを渡し、本文は渡さない(`escalate.go:293`) |
| `agent-messaging.md` の message shape・TASK_ID の必須の集合・2,000 字の上限 | 合う | escalate は同じ `Validate` を使い、`--raw` の抜け道はない。escalate は agmsg のメッセージではないので、星型の規則は変わらない。規則の文書は escalate に触れていない(参考の 1 つ目) |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | changed scope で golang を選んだ。shellcheck、hook の `sh -n`(root と templates の 20 本)、guard の awk の構文、`jq -e` の settings 2 つ、Codex の hook の 3 つのガード、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照。golang は `gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、`staticcheck` は出力なし(どちらも導入済みで、skip ではない)。`secret-scan-branch` は 382c18c8..11af3897 で clean |
| `HARNESS_VERIFY_MODE=static RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` | pass(rc 0) | full scope でも選ばれる言語は golang だけで、結果は上と同じ |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 13 skill が lock-step |
| `./scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `cmp` で skill の 4 面 | 同一 | `.claude/skills/org/SKILL.md` と `.agents/`、`templates/base/.claude/`、`templates/base/.agents/` |
| `./scripts/plan-visual.sh digest <plan>` | `b09718e048cd` | `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org escalate --help` と skill の `escalate` の行と雛形 | 合っている(1 か所を除く) | 受け付ける TYPE、TASK_ID、2,000 字、stdout の `escalated <id>`、未通知のときの `inbox notify`、受信箱に書けないときの扱いは 3 か所で同じ。`ralph.toml` が読めない場合は `--help` と雛形にあり、skill にない(K-4) |
| `ralph org inbox` / `show` / `ack` / `resolve` / `notify` の `--help` と skill の `inbox` の行 | 合っている | 列の名前、`--all`・`--json`、遷移と終了コード、note の規則(1 行、500 字、制御文字なし)、`--org-id` の拒否 |
| `ralph org wait --help` と skill の `wait` の行 | 合っている | 出力の 4 欄(ID、org、TYPE、本文の 1 行目)、acked で返らないこと、併用の拒否 |
| 旧い台帳の検査の一覧(`internal/cli/org_legacy_ledger.go` のコメント、skill の「前提」節) | 合っている | escalate・ack・resolve・notify は `orgLedgerMutating`、list・show・`wait --inbox` は `orgLedgerReadOnly` で、コードの引数と同じ |
| `--org-id` の persistent flag の説明(`org.go:46`) | 合っている | 拒否する動詞の一覧が `rejectFlagsWithAll`・`newOrgInboxRuntime`・`runOrgWaitInbox` と合う |
| README の org の節と表、AGENTS.md の `internal/org/` の行 | 合っている | |
| 仕様 FR-5 の注記 | 1 か所足りない | K-4 |
| `docs/tech-debt/README.md` | ずれている | K-1、K-3 |
| `.claude/skills/org/SKILL.md:453`(4 面) | 見た目だけのずれ | K-2 |
| 計画の Progress checklist | 少しずれている | V-1 |

## Findings

既知のずれ(self-review から `/sync-docs` に回したもの。依頼どおり verdict には数えない):

| ID | 元の ID と severity | Finding | Evidence |
| --- | --- | --- | --- |
| K-1 | self-review M2(MEDIUM) | tech-debt 台帳の「Small findings of org-feature-worktree」の行の (f) と (j) が開いたまま。S4 の `containsPhrase` への書き換えで直っているが、Debt・Impact・Trigger の (f)・(j) と Why deferred の (j) が閉じていない | `docs/tech-debt/README.md:193`、`internal/org/prompts_test.go` の `squashSpace` / `containsPhrase` |
| K-2 | self-review L7(LOW) | skill の「機能ごとの org」の段落の最後の行が、折り直されずに表示幅 104 のまま(前後の行は 70〜75) | `.claude/skills/org/SKILL.md:453`(4 面とも同じ) |
| K-3 | self-review L4・L5・L8・N2・N3(LOW) | tech-debt 台帳にこの計画の行がない(`grep -c org-inbox-escalate docs/tech-debt/README.md` は 0)。L8 (c) の `watch.go:452` は今も `filepath.Join(p.StatusDir, EscalationsRelName)` | `docs/tech-debt/README.md`、`internal/org/watch.go:452` |
| K-4 | self-review L1 の直しに伴う追記(LOW) | skill の `escalate` の行と「受信箱」節、仕様 FR-5 の注記は、人に NOT RECORDED で届く条件を「受信箱に書けなかったとき」だけで挙げる。コードは `ralph.toml` が読めないときも同じにし、`--help` と雛形はそう書いている | `.claude/skills/org/SKILL.md:178` と `:305`、`docs/specs/2026-10-07-org-multi-org-director.md:55`、`internal/cli/org.go:1761` |

この段で見つけたもの:

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| V-1 | LOW | doc drift(計画の記録) | 計画の `## Progress checklist` の `- [ ] Review artifact created` が、self-review の報告が 6d803bed と 39cec1b0 でコミットされたあとも未チェック。AC10 の `[x]` は c2bc718d で付き、`/test` より前にテストの部分まで済んだように読める | 計画の 105〜107 行と 159 行 | `/verify` は計画を書き換えない。`/pr` の前に計画を触る機会に、`Review artifact created` と `Verification artifact created` を付ける。AC10 の `[x]` は `/test` が通れば正しくなる |

参考(指摘にはしない):

- `.claude/rules/ralph/agent-messaging.md` は、`ralph org send` のほかに `ralph org escalate` も同じ検証を通すこと(TYPE は 3 つ、`--raw` なし)を書いていない。escalate の `--help` は message の形の参照先としてこの規則を挙げており、規則の記述そのものは正しい。計画の文書の範囲にも入っていない。director の規則を書く 8a 段目で足すかを決めればよい
- `realEscalate`(watch の通知)は `osascriptNotify` に切り出され、`Run()` から `CombinedOutput()` に変わった。watch の通知が失敗したときのエラー文に osascript の出力が付くだけで、成否の扱いは変わらない
- `escalationRecord.AlertID` を `omitempty` にしたが、watch の唯一の組み立て(`watch.go:1059`)は常に alert ID を入れる。`escalations.jsonl` を読む非テストのコードはなく(grep で書き手の 3 か所だけ)、計画の前提 3 は成り立つ
- `newOrgRuntimeAt` が失敗するのは config の読み込みだけなので、`--help` の「ralph.toml does not load」は実際の条件と合う

## Observational checks

- 雛形の 2 つの例(`mkdir -p .harness/state` から `EOF` までの書き込みと、`ralph org escalate --state-dir '/tmp/ledger a' … --text "$(cat …)"`、つないだもの)を `leader.md` から取り出して `{{ORG_ID}}` を埋め、HEAD の `pre_bash_guard.sh`(root と `templates/base/`)に `jq -nc` で組んだ payload を default と bypassPermissions で渡した。12 通りとも rc 0 で判定なし。陽性対照(コーパスにある deny の形)は 2 本とも `deny`。self-review が「確かめていない」とした `--text "$(cat ...)"` が guard を通ることは、これで確かめた
- 枝のバイナリを scratchpad に `go build` して、7 つの動詞の `--help` を取った(evidence の末尾)。escalate そのものは、本物の osascript が動くので打っていない
- `escalations.jsonl` の読み手を grep した。非テストのコードは `watch.go:1059`、`escalate.go:287`、`escalate.go:350` の書き手だけ

## Coverage gaps

- テストは実行していない。AC1〜AC8 と AC2b は、テストが何を固定しているかを読んで確かめた。通るかどうかは `/test` が確かめる
- 本物の osascript(darwin)の経路はどのテストも通らない(`DesktopNotify` は stub)。`osascriptNotify` の `%q` での組み立てが AppleScript の文字列として正しいかは、watch の既存の経路と同じ形であることだけを確かめた
- 別々のプロセスからの同時の escalate はテストにない。goroutine のテストはそれぞれが lock ファイルを開き直すので flock の衝突を通るが、プロセスをまたぐ場合は manifest のロックと同じ仕組みであることからの推測。未確認です
- 実機の leader の座席が雛形のとおりに escalate を打つ確認(herdr の上)はしていない。計画も求めていない

増やすと確信が最も上がる確認: `/test` で、変更を 1 つずつ戻す 2 つの mutant を回す。(1) `inbox.go:338-340` の改行を足す 3 行を外すと `TestInboxEscalate_TornLastLineStaysSeparateAndTheNewEventIsReadable` が落ちること。(2) `inbox.go:369-371` の生の行の走査を外すと `TestInboxEscalate_NeverReusesTheIDOfADamagedLine` が落ちること。どちらも落ちれば、Codex plan advisory の指摘 1 の直しがテストで固定されていると言える。

## Verdict

pass

- Verified: AC1〜AC8 と AC2b はコードとテストの読み合わせで満たす(AC6 の計画からのずれは Progress に記録済み)。仕様 FR-5・FR-11・Security と `agent-messaging.md` に合う(FR-5 の注記の K-4 を除く)。`./scripts/run-static-verify.sh` と full scope の静的な `run-verify.sh` は rc 0。`check-skill-sync.sh` と `check-sync.sh` は通る。skill の 4 面は同一。雛形の 2 つの例は HEAD の guard を通る。計画の承認 digest は一致する
- Partially verified: AC9(K-2 と K-4 が残る。`/sync-docs` で直す予定)、AC10(静的な部分だけ。テストは `/test`)
- Not verified: テストの実行、本物の osascript、プロセスをまたぐ同時の escalate、実機の leader の座席
