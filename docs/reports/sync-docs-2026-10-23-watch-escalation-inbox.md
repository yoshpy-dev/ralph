# sync-docs report: watch-escalation-inbox

- Date: 2026-10-23
- Plan: `docs/plans/active/2026-10-23-watch-escalation-inbox.md`(承認の digest `0bf5660a7229` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が計画で触れたのは Progress checklist の 3 行だけで、digest の外)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `e8e8d652`、branch は `fix/watch-escalation-inbox`。run 開始時の HEAD は `b09b04bb`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-23-watch-escalation-inbox.md`(LOW 9 件は d9b8cd9a で直った)、`docs/reports/verify-2026-10-23-watch-escalation-inbox.md`(pass。LOW 3 件と INFO 1 件は 8d1c53ce と 000a2352 で直った)、`docs/reports/test-2026-10-23-watch-escalation-inbox.md`(pass。穴 1〜3 は 08083792 で埋まった)
- 差分の大きさ(`git diff e8e8d652...HEAD --shortstat`、この run の前): 39 ファイル、+3200/-283。walkthrough を書くかどうかは `/pr` が決める

## Summary

`e8e8d652...b09b04bb` の差分は、横断の見張り(`ralph org watch --all`)の escalation(理由 `leader_is_anomaly_subject` と `deadman_timeout`)を、受信箱の TYPE `ALERT` の件として記録する。`[org.director].enabled` で、その見張りが `--once` でなく止める頼みも出ていなければ delivery `deadline` で記録だけにして director と期限に任せ、そうでなければ記録と同時に人へ送る。受信箱に書けないとき、と見出しに入れる状態ファイルの値が検査を通らないときは、今までの道(banner、通知、`escalations.jsonl` の 1 行)に落ち、行の欄 `osascript` が通知の結果を持つ。

文書の大半は、この branch の S3・S4 と verify の直し(000a2352)がすでに合わせていた。この run はコードの読み(`escalateAlert`、`inboxDelivery`、`sendInboxItemToHuman`、`WatchStatusFileName`、`WATCHDOG ESCALATION` の banner、help の 4 つ)と文書の言い分を突き合わせ直し、食い違いを 4 つ見つけて直した。

- AGENTS.md の repo map に、見張りの escalation が受信箱の ALERT の件で、`escalate` と同じ規則(`inboxDelivery`)で届くことの 1 句がなかった
- `docs/recipes/director-wake.md` に、見張りが自分で書く ALERT の件の wake が次のサイクルになることの説明がなかった(test report の「Test plan の外で気づいた点」)
- tech-debt のファイルの大きさの行が、08083792 のあとの行数(`watch_test.go` 3007)と 000a2352 のあとの行数(`watch.go` 1848)に合っておらず、`watch_all_test.go` を挙げていなかった
- 計画の Progress の「Review / Verification / Test artifact created」が `[ ]` のままだった

## Changes made

| File | Change |
|------|--------|
| `AGENTS.md` | repo map の `internal/org/` の行、横断の見張りの括弧に「`escalateAlert`: 期限切れの deadman と leader 自身の ALERT は、同じ受信箱の `ALERT` の件で、`escalate` の規則 `inboxDelivery`(`escalate.go`)で届く」の句を足した。ファイルの境界は変えず、名前は `watch.go` と `escalate.go` の関数だけ |
| `docs/recipes/director-wake.md`、`templates/base/docs/recipes/director-wake.md` | 「What push does」の節の終わりに 1 段落を足した。見張りの ALERT の件は pulse の中で書かれ、wake はその前に走るので、push の director には次のサイクルの 1 行で知らされる(最大 `[org.watchdog].interval_seconds`、既定 30 秒の遅れ)。ack の期限は記録の時刻から数える。2 つの写しは同じ Edit で、`cmp` で一致 |
| `docs/tech-debt/README.md` | ファイルの大きさの行(`internal/org/watch.go` が 1847 行、`watch_test.go` が 2870 行と書いた行)を、`watch.go` 1848、`watch_test.go` 3007 に直し、「after the self-review fix」を「after the test-gap tests」にした。`watch_all_test.go`(2614 行、base で 2200 行)が 2 つ目の節 `// --- plan 2026-10-23-watch-escalation-inbox: the escalations of a run` にテストを持つことも足した。直す手立ての欄は変えていない |
| `docs/plans/active/2026-10-23-watch-escalation-inbox.md` | Progress checklist の「Review artifact created」「Verification artifact created」「Test artifact created」を `[x]` にした。Progress は digest の外(`digest_body` が `## Progress checklist` を飛ばす)。AC1〜AC10 の印と「PR created」は触れていない |
| `docs/reports/sync-docs-2026-10-23-watch-escalation-inbox.md` | 新規(この report) |

足した文にホームのパス、token はない。`git diff --check` は rc 0、差分の U+FFFD は 0 件。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md`(repo map の `internal/org/` の行) | 直した | 上の表のとおり。既存の文(`escalate` が記録だけにするか人にも送るかの規則、見張りの pulse)は今も正しい。足したのは両者の関係の 1 句 |
| `templates/base/AGENTS.md`、`.ralph/core/AGENTS.core.md`、`docs/architecture/repo-map.md`、`docs/quality/definition-of-done.md`、`.codex/README.md` | 変更なし | 前 2 つに repo map がない。後 3 つの「watchdog」は「watchdog state」「watchdog alerts」の総称で、escalation の行き先を書かない |
| `README.md`(124 行の動詞表、254・256 行の節) | 変更なし | `escalate` の行、見張りの行(ALERT の件の記録)、254 行(見張りの ALERT の件の届け方、director の resolve、受信箱に書けないとき)、256 行(サイクルの順と期限)が、コードと skill に合う。256 行の「an item escalate or the watch only recorded reaches no one until a watch runs again」は ALERT の件にも当たる |
| `docs/quality/quality-gates.md` と template の写し | 変更なし | 「Watchdog pulse layer」の行は、2 つの写しで文字列が同じ(python で比べた)。写しは元から KNOWN_DIFF の別の行を持つ |
| `.claude/rules/ralph/agent-messaging.md` と template の写し | 変更なし | TYPE の表の ALERT の行は agmsg の ALERT(見張りから leader)で、今も正しい。足された段落が「見張りが書く ALERT の件」と、`escalate` が `ALERT` を断ること、`REASON`・`ALERT_ID`・`ALERT_TS` の見出しを書く。2 つの写しは `check-sync.sh` で一致 |
| `/org` skill と `/director` skill の 4 写しずつ | 変更なし | `cmp` で `.claude/skills/{org,director}/SKILL.md` と `templates/base/.claude/skills/` の写し、`.agents/skills/` と `templates/base/.agents/skills/` の写しが、どれもバイト一致。skill の言い分を、コードと突き合わせた: 状態ファイル名 `watch-status-<org_id>.json`(`WatchStatusFileName`)、stderr の頭 `WATCHDOG ESCALATION`(`watch.go` と `escalate.go`)、`alert_id` の形 `<org>/<seat>/<condition>@<n>`(`sendAlert` の `"%s@%d"`)、`ralph org report --out` と `ralph org read --seat` の flag(`internal/cli/org.go`)、`/director` skill の参照先の見出し「件の処理」(644 行)。`stop`・`disband` の行は hunk が当たっておらず、tech-debt の F-8 の行の cell の長さは変わらない |
| `internal/org/prompts/director.md`、`leader.md` | 変更なし | director.md の 1 行は skill の「件の処理」と合う。leader.md の「件を上げる」の節は leader 自身の escalate だけを書き、ALERT や deadman の語がない |
| help(`ralph org watch`、`ralph org inbox`、`inbox notify`、`escalate`) | 変更なし | watch の help(`org_watch.go:73-83`)は 2 つの道を書き分け、`TestOrgWatch_HelpDescribesTheAlertItemEscalation`(`org_watch_test.go:1879`)が 2 つの句を固める。inbox と notify の help は ALERT の件を足し、`TestOrgEscalateAndInboxHelp_DescribeTheDirectorSwitch` が固める。escalate の help は受け付ける TYPE を 3 つだけ挙げるので、`ALERT` を断ることと合う |
| コード内の doc(`escalateAlert`、`checkDeadman`、`watchPendingAlert`、`EscalateFunc`、`WatchAllParams.Escalate`・`StatusDir`、`escalationRecord`、`realEscalate`、`config.go` の `DeadmanMinutes`、`templates/base/ralph.toml` の `deadman_minutes`) | 変更なし | どれも今のコードに合う。verify が挙げた古い文の 4 か所(`config.go`、`ralph.toml` の雛形、`watch.go` の 1312 行、tech-debt の Impact)は 000a2352 で直っている |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 変更なし | FR-6 の注「試行の所見 F6 と F13 の直しで決めたこと」が、`inboxDelivery` の判定、重複(今の見張りの判定だけ)、受信箱に書けないときと値の検査の 4 つ目の条件、遅れの上限、director が無効なときと登録がないときの 2 回目の通知、座席や org を止めるのは人、をコードと計画に合って書く。印は `[x]` 28、`[ ]` 4(grep で数えた)。Status の一覧の PR は 34 本で、この PR のブランチ名を数えない。「Current state」の 48 行(「人に届くのは watchdog のデッドマンだけ」)は 2026-10-07 時点の問題の記述で、5 段目から古い。この PR の変更ではなく、背景の節なので触れていない |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし | 179 行の 2026-10-23 の注が FR-6 の注を指す。90 行の受け入れ条件(Lead が N 分無応答 → 人に通知)は、29〜31 行の 2026-10-07 の注(後継の仕様が優先)が覆う |
| `docs/evidence/org-spec-evidence-2026-10-21.md` | 変更なし | 項目の表は `[x]` 27、`[ ]` 4(python で数えた)で、79 行の数と合う。F6・F13 の行、FR-6・安全の不変条件の行、PR 一覧(34 本)が仕様と tech-debt に合う。`fix/split-preamble` は `docs/specs`・`docs/evidence`・`docs/tech-debt` にブランチ名としては残らない(残るのは archive の計画名と report 名) |
| `docs/tech-debt/README.md` | 直した(上の表) | F6・F13 の行は取り消し線と `RESOLVED` の注で閉じている。開いている行 3 本(「An ALERT item of the watch that nobody resolves」「`sendInboxItemToHuman` ... returns with no `notified` event」「ファイルの大きさ」)が挙げる関数・フィールド・理由の名前 25 個は、どれも `internal`・`scripts`・`tests`・`.claude` のどこかに実在する。この 3 本はテスト名を挙げない。「The watch's ALERTs to the leaders failed 10 times」の行の Impact は ALERT の件の言い方に直っている |
| `docs/recipes/` | 直した(`director-wake.md`) | `director-launch-cmd.md` ほかの recipe は見張りの escalation にも ALERT の件にも触れない。`director-wake.md` の wake の対象(`DirectorWakeItems` は open で `woken` がない件。TYPE を読まない)は ALERT の件も含み、本文の「the open inbox items」と合う。足りなかったのは時間の順だけ |
| CHANGELOG の類 | なし | repo に CHANGELOG はなく、release ノートは goreleaser が git log から作る(`.goreleaser.yml`、`/release` skill)。計画の Rollout が書く release ノートの文は、`/pr` の本文と release の仕事 |
| `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh` | rc 0、rc 0 | 下の Verification |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の LOW(古い「人へ escalate する」文 4 か所) | 4 か所とも直っている。`config.go` の doc、`templates/base/ralph.toml`、`watch.go` の 1312 行の comment は読んで確かめ、tech-debt の Impact は行を読んで確かめた |
| verify の LOW(計画と skill が d9b8cd9a の 4 つ目の条件を書いていない) | 直っている。`/org` skill、FR-6 の注、計画の Scope・D5・AC5 と逸脱の注が書く。digest は `0bf5660a7229`(承認と同じ) |
| verify の LOW(tech-debt の「both deliveries」) | 直っている。F6 の RESOLVED の注に「both deliveries」の句はもうない |
| verify の INFO(README の resolve の言い方) | 変更なし。README の「A director that resolves an `ALERT` item left to it (delivery `deadline`, not yet sent to the human)」は仕様の言い方と同じ条件を書く |
| verify の Coverage gap(`ralph org watch` の help を固めるテストがない) | 埋まっている。`TestOrgWatch_HelpDescribesTheAlertItemEscalation`(`internal/cli/org_watch_test.go`)が 2 つの句を固める |
| test の穴 1〜3 | 08083792 が埋めた。tech-debt に残る行はない |
| test の「Test plan の外」の 2 点 | 1 つ目(cobra の `--once` を通した ALERT の件のテストがない)は記録だけ。2 つ目(deadline の ALERT の件の wake は次のサイクル)は recipe に書いた |

## Plan drift notes

- 計画の Affected areas は `docs/recipes/director-wake.md`、`AGENTS.md` を挙げない。この run が足した。コードは変えていない
- 計画の AC1〜AC10 は `[ ]` のまま。印は digest に入らないが、AC10(`GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0、変えたパッケージの Go のテストが go1.25.8 でも通る)の最後の run-verify は 08083792 と 000a2352 のあとに打たれていない(verify の Verdict と test の「この報告のあとに埋めた穴」が書く)。orchestrator が打つので、この run は AC の印を付けていない

## Left open

- 計画の Rollout が書く release ノートの文(見張りの escalation は受信箱の ALERT の件になること、resolve されない件が最大 2 回届くこと、新しい skill には新しい binary が要ること)は、まだどこにも書かれていない。`/pr` の本文か release のときに書く
- 実機(本物の herdr、director のセッション、osascript)の確かめは、計画の Non-goals のとおりない。Go のテストだけ
- 仕様の「Current state」の 48 行が古い(上の表)。背景の節の書き直しは別の仕事

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.`(recipe の写しを同じ Edit で直す前は DRIFTED 1 で rc 1 だった) |
| `cmp`(`org`・`director` の skill の root と template の写し、`director-wake.md` の root と template) | すべて一致 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-23-watch-escalation-inbox.md` | `0bf5660a7229`(承認の digest と同じ) |
| `git diff --check`、差分の U+FFFD の数 | rc 0、0 件 |
| テスト | 打っていない(`/test` の範囲。この run が変えたのは文書と計画の Progress だけで、Go とスクリプトには触れていない) |

## Insight event

追記していない(依頼が orchestrator に任せた)。打つなら次の 1 本。

```
./scripts/insights-append.sh --slug watch-escalation-inbox --flow standard --phase sync_docs --verdict pass --cycle auto --source skill
```
