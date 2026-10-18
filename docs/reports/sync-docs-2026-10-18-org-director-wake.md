# sync-docs report: org-director-wake

- Date: 2026-10-18
- Plan: `docs/plans/active/2026-10-18-org-director-wake.md`(8b-1。承認の digest `e26fe4769759` はこの run の前後で同じ)
- Pipeline cycle: 1。base は `9e92d928`、branch は `feat/org-director-wake`。run 開始時の HEAD は `bb2e2203`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-18-org-director-wake.md`、`docs/reports/verify-2026-10-18-org-director-wake.md`(pass。再確認の節あり)、`docs/reports/test-2026-10-18-org-director-wake.md`(pass。テストの穴の直しの節あり)。evidence は `docs/evidence/director-wake-livefire-2026-10-18.md`
- この run が変えたのは `docs/architecture/repo-map.md` の 1 行、計画の Progress checklist の 1 項目、この report、insight event の jsonl 1 行だけ。コードもテストも skill も help も触っていない

## Summary

`9e92d928...bb2e2203` の差分(57 file)のうち文書にあたる面を、実装と、この worktree でビルドした binary の help と出力に突き合わせた。突き合わせた面は、AGENTS.md の `internal/org/` の行、README の動詞の表と watch の段落と director の段落と wake の段落、仕様 FR-10 の注、recipe の 2 面(`director-wake.md`、`director-launch-cmd.md`)、`/director` と `/org` の skill の 8 面、`internal/org/prompts/director.md`、`templates/base/ralph.toml` のコメント、`scripts/ralph-config.sh` の 2 面、help(`attach`、`start`、`watch`、`status`、`inbox show`、`escalate`、`--config`)、tech-debt の 7 行、`docs/architecture/repo-map.md`、`docs/quality/quality-gates.md`。

ずれは 1 点だった。`docs/architecture/repo-map.md` の recipe の一覧に、新しい `director-wake.md` がなかった。足した。計画の S4 の記録は「recipe の索引はなく」と書くが、repo-map.md に一覧があった(前の段の sync-docs が launch_cmd の recipe を足した行)ので、この記録は正しくない。計画の Progress checklist に、この点を書いた。

旧い理由の文(`-` で始まる送り先の理由が wake_cmd を名指す形)は、書き換えない記録にだけ残っている。`git grep` で探すと、残るのは `docs/evidence/verify-2026-10-18-org-director-wake.md` の 1 行(前の run の出力の引用)と、`docs/reports/test-2026-10-18-org-director-wake.md` の d2a740d3 の説明(旧い文と新しい文を並べて書く)だけで、README、skill、recipe、help、tech-debt、仕様、雛形にはない。実装(`ValidateDirectorWakeTarget`)と、binary の attach の断り(`--wake-target=-x`)は新しい文「the program it is passed to」を出す。

## Changes made

| File | Change |
|------|--------|
| `docs/architecture/repo-map.md`(37 行) | `docs/recipes/` の一覧の末尾に「waking the director by push with `wake_cmd`」を足した |
| `docs/plans/active/2026-10-18-org-director-wake.md`(Progress checklist の「Implementation started」の子項目) | sync-docs の項目を 1 つ足した。上のずれ 1 点と、S4 の「recipe の索引はなく」が正しくないこと、`AGENTS.md` の recipe の行を足さない理由を書く。本文は触っていない。`./scripts/plan-visual.sh digest` は変更の前後とも `e26fe4769759` |
| `docs/reports/sync-docs-2026-10-18-org-director-wake.md` | 新規 |

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `AGENTS.md`(90 行の `internal/org/` の行) | 合っている | 足された句「the push wake (`director_wake.go`: `ralph org watch --all` wakes a director registered with push through `[org.director].wake_cmd` or input to its herdr pane, and records `woken` in the inbox)」は `director_wake.go` の `WakeDirector`、`directorWakeSender`、`InboxStore.AppendWoken` に合う |
| `AGENTS.md`(112 行の recipe の行) | 変えない | 「(Codex setup, language packs, worktrees)」は例示で、前の段の launch_cmd の recipe も seat-permissions も挙げていない。AGENTS.md は短い map にする決まりなので、足さない |
| `README.md`(124 行の動詞の表) | 合っている | attach の `--wake pull\|push\|pull,push`(既定は `[org.director].wake`)、watch の「wakes a director registered with push for new inbox items」は help に合う |
| `README.md`(256、258、260 行) | 合っている | 260 行の wake の段落の数値と文言を実装に突き合わせた。起こす 1 行の文(`DirectorWakeText`)、20 件の上限と `plus N more`、750 ms(`DefaultSendEnterDelayMS`)、5 分(`directorWakeRetry`)、送り先の形の断り(空、制御文字、256 文字、`-` 始まり)、`woken director_id=d1 via=herdr`(`inbox show` の events の形)、`watch.log` が `--detach` の出力先であること。attach の成功行と `status` の出力は、隔離した scratch の台帳で attach を打ち、`wake=pull,push, wake_target=<値>`、`wake:` と `wake_target:` の行、`director:` の行の `wake=pull,push to <送り先>` が README と skill の書き方に合うことを確かめた |
| `README.md`(254 行の escalate の段落) | 変えない | 「for the director to read」は push でも誤りではない。起こし方は 256 行以降が書く |
| 仕様 FR-10 の注(`docs/specs/2026-10-07-org-multi-org-director.md:95`) | 合っている | 名指すテスト 3 本(`TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem`、`TestOrgDirectorAttach_RefusesAWakeItCannotSendWithNothingWritten`、`TestOrgDirectorStart_ALaunchCmdThatPushesNeedsAWakeCmd`)は `internal/` に実在する。86 行の「生存の印は 8b 段目の起こし口で考える」は 8a-1 の時点の注で、95 行が 8b-2 に分けたと書くので、そのままにする |
| recipe `docs/recipes/director-wake.md` と `templates/base/` の写し | 合っている | `cmp` は終了コード 0(同一バイト)。見張りのサイクルの順、`{target}` の表、attach と start の断り、`wake_cmd` の走り方(10 秒、末尾 512 バイト、設定のディレクトリ、相対パスの解き方)、herdr の既定の 3 回の label 確認と send-text と 750 ms と Enter、失敗の行の形(`watchdog: director d1 not woken for e3 e5: ...`、`... is not woken: ... (said once)`)、`ralph org watch --stop` の存在を実装に突き合わせた。recipe の Orca と tmux と promoted session の herdr の例は実機で打っていないと、recipe 自身が書く |
| recipe `docs/recipes/director-launch-cmd.md` と `templates/base/` の写し | 合っている | `cmp` は終了コード 0。start の断り(push で wake_cmd が空)と、wake の案内の 2 か所は start の help に合う |
| `/director` skill(4 面) | 合っている | `cmp` で 4 面とも同一バイト。attach の例と断りの 4 つ、start の記述、「受信箱の処理」の wake ごとの扱い、見張りが送る 1 行、`woken` の表示、失敗時の 5 分、`ralph status` の `wake=push to <送り先>` の書き方が、実装と binary の出力に合う。「まだないもの」の節は消えていて、「push はまだない」の記述は repo にない |
| `/org` skill(4 面) | 合っている | `cmp` で 4 面とも同一バイト。`watch`、`inbox`、`director` の行と受信箱の節の追記が help と実装に合う |
| `internal/org/prompts/director.md` | 合っている | 参照先の節「受信箱の処理」は skill に実在する。headless の herdr と launch_cmd の 2 通りの案内は `ResolvedWakeTarget` と送り方の分け方に合う |
| `templates/base/ralph.toml`、`scripts/ralph-config.sh` と template の写し | 合っている | `wake = ["pull"]` と `RALPH_ORG_DIRECTOR_WAKE="pull"` は同じ値で、`defaults_sync_test.go` に `org.director.wake` の行がある。`wake_cmd` の既定は空で、環境変数はない(ralph-config.sh のコメントがそう書く)。コメントの `{target}` を shell の文字列に入れない注意は、recipe の Security と `OrgDirectorConfig.WakeCmd` の doc に合う。repo の root に `ralph.toml` はない |
| help | 合っている | `attach --help`(`--wake`、`--wake-target`、断りの 5 つ、成功行の例)、`start --help`(launch_cmd の断り、herdr では wake_cmd が要らない)、`watch --help`(起こす手順、wake_cmd を起動時に読むこと)、`director status --help`、`inbox show --help`(events に woken)、`escalate --help`(push の言及)、`--config` の help(attach、start、`watch --all` が wake と wake_cmd を読むこと)を、binary の出力で読んだ |
| tech-debt(203、209、239、247、249、255、301 行) | 合っている | 7 行とも欄は 5 で、U+FFFD はない。239 行の `defaultDirectorCmdWaitDelay`、301 行の `shellQuote`(`internal/org/feature.go`)、`ValidateDirectorWakeTarget`、`WakeDirector` は実在する。203 行の「about 50 s」は herdr の呼び出し 5 回(label の確認 3 回、send-text、send-keys)かける `DriverCallTimeout` の 10 秒で、実装に合う。209、247、255 行の「8b-2 に残す」は計画の Non-goals に合う。249 行は push で登録が送り先を決めるようになったことと、歯止め(件の ID だけの 1 行、ack の期限)を書く |
| `docs/architecture/repo-map.md` | 直した | recipe の一覧に director-wake を足した。`.claude/skills/director/` の行(48 行)は push の扱いを書かない粒度で、変更不要 |
| `docs/quality/quality-gates.md`(79、80 行) | 変更不要 | 79 行の「Watchdog pulse layer」の行は、watch が各サイクルで何をするかの全部を列挙する行ではなく、deadline と ALERT という門(失敗すれば人に届く経路)を書く行。起こす手順は門ではない(起こさなくても、起こしても、件は期限で人に届く)ので足さない。80 行の「Cross-org watch gate」が挙げる、watch がなければ断る動詞は、この段で増えていない |
| `.claude/rules/ralph/agent-messaging.md` | 変更不要 | 受信箱の event の一覧を持たず、escalate の検証だけを書く。`woken` はこの検証に関わらない |
| 絶対パスの `$HOME` と herdr の pane id、wake の送り先の値 | なし | 今回足した行にも report にも、これらの値は書いていない |

## Plan drift notes

計画の本文(Progress checklist より上)は承認の digest の下にあるので、書き換えていない。気づいたずれは次の 1 つ。

- S4 の記録(Progress checklist の「Implementation started」の下)の「recipe の索引はなく、check-sync.sh は root と template を名前を挙げずに比べるので、どちらも直していない」のうち、「recipe の索引はなく」は、`docs/architecture/repo-map.md` に recipe の一覧があったので正しくない。check-sync.sh が名前を挙げずに比べる点は合っている。一覧は今回直し、sync-docs の項目に書いた。Progress checklist の記録の中なので、digest には影響しない

## Left open

- recipe の Orca、tmux、herdr の中の promoted session の `wake_cmd` の例は、実機で打っていない(recipe と計画の Assumptions が書く)。Orca の `orca terminal send` の引数は `--help` の読みで、この run では打っていない
- Claude Code の director が作業中に打ち込まれた文を次の入力として待たせるかは、確かめていない(skill と recipe が「実機で確かめていない」と書く)
- 人が打ち込んでいる pane への push の影響は、確かめていない(recipe の「When to stay on pull」が書く)
- codex が pull の設定で `wait --inbox` を自分から始めるかは、確かめていない(recipe の「What the agents do with the line」が書く)
- `docs/recipes/director-wake.md:179` の 1 行が、ほかの行より長い折り返し(cosmetic)。直すと 2 面の `cmp` を取り直す必要があるが、内容は変わらないので、直していない
- この run は Go のテストを走らせていない。コードもテストも触っておらず、test report(`6a99b1a8`)と、その後のテストだけの直し(`e50bed12`)と `d2a740d3` の run-verify の rc 0 は、計画の Progress checklist が記録する

## Verification

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | 終了コード 0。`PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `cmp docs/recipes/director-wake.md templates/base/docs/recipes/director-wake.md` | 終了コード 0 |
| `cmp` の `director-launch-cmd.md`(root と template)、`director` と `org` の skill の 4 面(`.claude` と、`.agents`・`templates/base/.claude`・`templates/base/.agents` の 3 つ。2 skill で 6 組) | どれも終了コード 0(同一バイト) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-18-org-director-wake.md` | `e26fe4769759`(計画の `- Approved:` の行と同じ) |
| `git diff --check` | 出力なし、終了コード 0 |
| 編集した 2 file と tech-debt の 7 行の U+FFFD | なし |
| `git grep` の旧い理由の文 | 残るのは evidence 1 行と test report の説明だけ(上の Summary) |
| 隔離した scratch の台帳での `attach`、`status`、`attach --wake-target=-x` | 成功行、`wake:` と `wake_target:` の行、`director:` の行、`-` 始まりの断りの文が、文書の書き方と新しい理由の文に合った。台帳は scratch にあり、repo の台帳には書いていない。binary は scratch にビルドし、repo には置いていない |

`./scripts/verify.local.sh` と `./scripts/run-static-verify.sh` は走らせていない。変えたのは `repo-map.md` の 1 行、計画の checklist の 1 項目、この report で、tech-debt の plan の参照と secret scan の対象に新しい文字列は入らない(token、`$HOME`、pane id、送り先の値を書いていない)。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Orca、tmux、herdr の promoted session の `wake_cmd` の例の実機の動き、Claude Code の director の push の扱い、人が打ち込んでいる pane への push の影響、この run での Go のテストの全体

## Insight event

`./scripts/insights-append.sh --slug org-director-wake --flow standard --phase sync_docs --verdict pass --cycle 1 --source skill --critical 0 --high 0 --medium 0 --low 0` を打ち(終了コード 0)、`docs/insights/events/2026-10-18-org-director-wake.jsonl` の 4 行目として足した。ts は `2026-10-18T11:53:44Z`。commit は orchestrator が打つ。
