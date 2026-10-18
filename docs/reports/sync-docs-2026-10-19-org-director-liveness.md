# sync-docs report: org-director-liveness

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-18-org-director-liveness.md`(8b-2a。承認の digest `d9e4345287da` はこの run の前後で同じ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `88e37b1d`、branch は `feat/org-director-liveness`。run 開始時の HEAD は `1cc4ce27`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-19-org-director-liveness.md`(Merge: yes)、`docs/reports/verify-2026-10-19-org-director-liveness.md`(pass)、`docs/reports/test-2026-10-19-org-director-liveness.md`(pass。テストだけの追補 7ef22637 の記録あり)。実機の確認は `docs/evidence/director-liveness-livefire-2026-10-18.md`
- この run が変えたのは、tech-debt の 3 か所、`internal/cli/org_director.go` の help の 1 文とその pin(`internal/cli/org_director_test.go`)、この report。計画、先行 3 report、insight event には触っていない

## Summary

`origin/main...1cc4ce27` の差分(55 file)のうち文書にあたる面を、実装と、この worktree でビルドした binary の help に突き合わせた。突き合わせた面は、README の動詞の表と `ralph status` の行と watch、director の 2 つの段落、AGENTS.md の `internal/org/` の行、仕様 FR-9 と FR-10 の注、recipe 2 つとその写し、`/director` と `/org` の skill の 8 面、help(`org director`、`attach`、`status`、`start`、`stop`、`launched`、`org watch`、`ralph status`)、tech-debt の 9 行、`docs/architecture/repo-map.md`、`docs/quality/`、`templates/base/ralph.toml` と `scripts/ralph-config.sh`、`internal/org/prompts/director.md`。

ずれは 3 点だった。直した。

1. `ralph org director --help`(グループの help)が、「登録はセッションが動いていることを表さない」を、昇格したセッションと headless の director を分けずに書いていた。`director status` と `ralph status` の help、README、skill、仕様は分けて書いていて、グループの help だけが前の文のままだった。計画の Verify plan が探すのは「registration does not tell whether its session is still running」の文言で、グループの help は「Nor does it mean the session is still running」と別の言い方なので、その探し方では当たらなかった。
2. tech-debt の 203 行(watch の停止が cycle の終わりを待つ行)は、8b-1 の起こす手順の呼び出しまでを数えていた。この段で cycle に足した確かめの herdr と ps の呼び出しがなかった。計画の Risks は「203 行の遅れと同じ扱い」と書くが、203 行は変わっていなかった。
3. tech-debt の file size の行(今の 305 行)が、この段で 800 行を超えたテスト `internal/org/driver/herdr_test.go`(711 から 852)を挙げていなかった。1 の直しで `internal/cli/org_director.go` が 793 から 794 になったので、その数も同じ行で直した。

## Changes made

| File | Change |
|------|--------|
| `internal/cli/org_director.go`(グループの `Long` の末尾) | 「Nor does it mean the session is still running」を、昇格したセッションは動いていると示さず、headless の director は横断の見張りが各 cycle で確かめる(status を見る)、という文にした。ack の期限の文は残した。1 行増えて 794 行 |
| `internal/cli/org_director_test.go`(`TestOrgDirectorHelp_DescribesTheRegistration`) | pin を新しい文に合わせた。「Nor does it mean a promoted session is still running」と「the cross-org watch checks a headless director each cycle (see status)」の 2 つ |
| `docs/tech-debt/README.md`(203 行) | 「Since stage 8b-2a」の文を 1 つ足した。確かめ(`CheckDirector`)が受信箱の期限のあと、起こす手順の前に cycle を持ち、herdr の director では `agent get` と、agent がなければ pane の label の確認 3 回、`starting` と launch_cmd の director では ps の読みが、それぞれ `DriverCallTimeout` を持つので、停止が最大 40 秒(herdr)か 10 秒(ps)待ちうること、`director.jsonl` に変更が記録されると確かめを 1 回やり直すので 2 倍になりうること。Related の欄に計画(`docs/plans/active/2026-10-18-org-director-liveness.md`、Risks and mitigations)を足した |
| `docs/tech-debt/README.md`(305 行、file size の行) | テストの一覧に `internal/org/driver/herdr_test.go`(852、stage の前は 711。herdr の `agent_not_found` のテストが入った)を足し、`internal/cli/org_director.go` を 793 から 794(800 まで 6 行)に直した |
| `docs/reports/sync-docs-2026-10-19-org-director-liveness.md` | 新規 |

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `AGENTS.md`(`internal/org/` の行) | 合っている | 足された「the director check」の句は `director_check.go`(`CheckDirector`、`DirectorAliveFileName`)、`director_start_cleanup.go`(`cleanUpEndedDirector`)、`watch.go` の cycle の順(deadlines、check、wake、pulse、mark)に合う |
| `README.md`(動詞の表の `ralph org ...` の行、`ralph status` の行) | 合っている | watch の「checks whether the current headless director is still alive (`director-alive.json`)」と、start の「cleans up a current headless director that has ended」は help に合う。`director:` の行の `checked by the watch at <time>: alive (<how>)`、`ended (<how>)`、`unknown (<how>: <detail>)`、`not checked yet`、片付けの案内(`enabled = false` では stop だけ)、`director.check` の 4 つの key は、`status` の help と `org_director_check.go` に合う |
| `README.md`(watch、director の 2 つの段落) | 合っている | cycle の順、how の例(`start_running`、`start_ended`、`no_starter`、`agent_found`、`agent_ended`、`pane_closed`、`process_ended`、`pid_reused`、`changed`)は `DirectorCheckHow*` の定数に全部ある。start の片付けの文(`cleaned up d1, which had ended (agent_ended): ...`)は実機の evidence の出力と同じ形。attach の断りの文は `attach` の help(binary の出力)に合う。「A promoted session's registration does not mean the session is still running」は `director status` の help に合う |
| 仕様 FR-9 の注(89 行)、FR-10 の注(96 行) | 合っている | 注が挙げる how の語(`pane_unconfirmed`、`agent_get_failed` を含む)、`MarkStoppedAtVersion`、`CheckDirector`、evidence のパスは実在する。D2 からの逸脱(check-in の ps をロックの外に出した)が書いてあり、`director_start.go` の `MarkCmdLaunched` の実装に合う。FR-10 の注は 8b-2 を 2 本に分けたことを足している |
| recipe `director-launch-cmd.md`、`director-wake.md` と `templates/base/` の写し | 合っている | `cmp` は 2 組とも終了コード 0。start の片付けの 3 つの場合、check-in を断る条件(start のプロセスの gone、zombie、起動時刻の違い。ps が読めないときと starter の記録がないときは受け付ける)、wake の ended を送らない文は、実装と help に合う |
| `/director` skill(4 面) | 合っている | `cmp` で 4 面とも同一バイト。「確かめを読む」の節の how の分類(alive 3、ended 4、unknown 9。計 16 で `DirectorCheckHow*` の定数と同じ数)は定数に合い、「立てる」の 5、「終える」の 4、「見張りが送る 1 行」の ended の扱いの相互参照の見出しは実在する |
| `/org` skill(4 面) | 合っている | `cmp` で 4 面とも同一バイト。`watch` の行の cycle の順と、`director` の行の attach の断りの追記、start の starter、片付け、check-in の断りが help と実装に合う |
| help(binary を scratch にビルドして読んだもの) | 1 点直した | `org director status`、`org director attach`、`org watch`、`org director stop` は README と skill に合う。`org director`(グループ)は上の 1 の通りで直した。`org director start`(Long の片付けの段落)、`ralph status`(Long の `director:` の説明)、`org director launched` は、diff と README・skill の突き合わせで読み、binary の出力では読んでいない |
| tech-debt(203、209、247、249、254、256、303、304、305 行) | 2 行を直した | 9 行とも欄は 5 で、U+FFFD はない。名指す関数(`CheckDirector`、`checkDirectorAgent`、`cleanUpEndedDirector`、`directorHerdrCleanupNotice`、`MarkStoppedAtVersion`、`directorCheckInStarterEnded`、`confirmDirectorResources`、`watchHerdrProbe`、`UnrecordedWorkspaceLabel`)は書かれた file に実在する。file:line の参照はない。file size の行の数(1203、977、857、804、782)は `wc -l` と `origin/main` の値に合う。256 行の取り消し線と Still open の書き分けは、plan の S5 の記録(255 行が 256 行目になった)に合う。203 行と 305 行を直した(上の Changes made) |
| tech-debt 271、272 行(launch_cmd の start time の行) | 変更不要 | 新しい確かめは起動時刻の違いを `unknown`(`pid_reused`)と読み、自動の片付けは gone と zombie と herdr の not_found だけで打つ。272 行が書く害(stop と start の `abortCmd` が生きている director を終わったと読む)は、確かめの側では起きない。256 行がこの 2 行を指している |
| `docs/architecture/repo-map.md` | 変更不要 | 新しい file は `internal/` の下だけで、repo-map は `internal/org/` の file を挙げない。recipe の一覧に足す recipe はない。`.claude/skills/director/` の行(48 行)は確かめを書かない粒度 |
| `docs/quality/quality-gates.md`(79、80 行)、`definition-of-done.md` | 変更不要 | 79 行の「Watchdog pulse layer」は、失敗すれば人に届く経路(ALERT と期限)を書く行で、cycle の全部を並べる行ではない。確かめは門ではなく、ended でも件は期限で人に届く。8b-1 の sync-docs と同じ判断。80 行の watch gate が断る動詞は、この段で増えていない |
| `templates/base/ralph.toml`、`scripts/ralph-config.sh` | 変更不要 | 確かめは設定を読まない(`[org.director].enabled` も見ない)ので、キーも環境変数も増えていない。launch_cmd のコメントの「you close the terminal yourself after `ralph org director stop`」は start の片付けのあとにも当てはまるが、誤りではない |
| `internal/org/prompts/director.md` | 変更不要 | 確かめ、片付け、starter の言及がない。参照先の skill の節は実在する |
| 絶対パスの `$HOME`、U+FFFD、`git diff --check` | なし | 追加行に `/Users/` はない。U+FFFD もない。`git diff --check` は終了コード 0 |

## Plan drift notes

計画の本文(Progress checklist より上)は承認の digest の下にあるので、書き換えていない。Progress checklist も orchestrator に任せ、触っていない。次を orchestrator が checklist に書く材料として残す。

- 計画の Verify plan の「Documentation drift to check」は、文言の探し方が「registration does not tell whether its session is still running」だけで、グループの help の言い回し(「Nor does it mean the session is still running」)に当たらなかった。この run で直した
- 計画の Risks は「tech-debt 203 行の遅れと同じ扱い」と書くが、203 行を直す作業が S5 の一覧になかった。この run で直した
- tech-debt の行は、この段で次のように動いた。203(この run で文を足した)、247、249、254、209 は文の書き換え。255 行は解決の注の行が入って 256 行目になった。新しい行は 303(codex の daemon)、304(別の herdr server)、305(file size)。計画の本文が書く 255、247、209、254 行の番号は、承認時の番号である

## Left open

- 実機で打っていない面: launch_cmd の director の確かめ(`process_ended`、`pid_reused`、遅れた check-in の断り)、Claude Code の director、push の wake が ended の director を飛ばすこと(D5)、2 つの start の重なり、unknown の各結果(`agent get` の失敗、label の違う pane、ps の失敗)。evidence の「What this run does not cover」が同じ一覧を書く。文書はこれらを実装とテストから書いていて、実機の出力には合わせていない
- Linux では確かめていない(`ps -o lstart=` の読みと時計の変更は tech-debt 272 行が書く)
- `ralph org director start --help`、`ralph status --help`、`ralph org director launched --help` は binary の出力で読んでいない(上の表)
- `ralph org director stop --help` の「When start's cleanup (after a step of start failed) or another stop records the director stopped while stop closes it」は、start の失敗後の片付けだけを挙げる。この段で start の片付けは、終わった director を片付ける場合も持つ。stop は登録を読み直して stopped を見つけるので、振る舞いは同じで、文は誤りではない。言い足すかは次に stop の help を触るときに決める

## Verification

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | 終了コード 0。`PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。skill は変えていないので `sync-skills.sh` は走らせていない |
| `cmp` の recipe 2 つ(root と template)、`director` と `org` の skill(`.claude` と、`.agents`・`templates/base/.claude`・`templates/base/.agents` の 3 つ。2 skill で 6 組) | どれも同一バイト |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-18-org-director-liveness.md` | `d9e4345287da`(計画の `- Approved:` の行と同じ) |
| `gofmt -l internal/cli`、`go vet ./internal/cli` | 出力なし |
| `go test ./internal/cli -run TestOrgDirectorHelp_DescribesTheRegistration -count=1 -v` | PASS |
| `go test ./internal/cli -count=1` | ok(339 秒、終了コード 0)。実行後の `git status --porcelain` は、変えた 3 file だけ |
| `git diff --check` | 出力なし、終了コード 0 |
| 編集した tech-debt の 203、305 行の欄の数 | どちらも 5。U+FFFD なし |
| docs が名指す path の実在 | この branch で足した参照に欠けはない(`docs/tech-debt/README.md` の古い行の欠けた参照は、この branch の前からある) |

`./scripts/verify.local.sh` と `./scripts/run-static-verify.sh` は走らせていない。変えたコードは help の 1 文と、その pin の 2 つの文字列で、`go test ./internal/cli` の全体が通っている。ただし、self-review、verify、test の 3 report は、この 1 文を変える前の HEAD(`1cc4ce27`)の結果である。この変更を cycle の再実行の対象にするかは orchestrator が決める。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 上の Left open の実機の面、Linux、`start --help` と `ralph status --help` の binary の出力

## Insight event

この run では追加していない(orchestrator が追加する)。
