# sync-docs report: seat-spawn-close

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-spawn-close.md`(承認の digest `550d25183df2` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。計画には触っていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `5850f89b`、branch は `fix/seat-spawn-close`。run 開始時の HEAD は `628369b9`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-22-seat-spawn-close.md`(Merge 可、LOW 10。L1・L2・L5〜L9 は直り、L3・L4・L10 は残した)、`docs/reports/verify-2026-10-22-seat-spawn-close.md`(pass。LOW V1〜V3、INFO V4〜V7。V1〜V6 は 0a032b37 と 3b07848d で直り、V7 は 628369b9 で直った)、`docs/reports/test-2026-10-22-seat-spawn-close.md`(pass。G1・G2 は 0a032b37 で直り、G3〜G6 は INFO で残した)
- この run が変えたのは、`docs/quality/quality-gates.md` とその template の写し(各 1 行)、この report、insight event。コード、先行 3 report、計画、skill、仕様、tech-debt、evidence の文書には触っていない
- 差分の大きさ(`git diff 5850f89b...HEAD --shortstat`): 28 ファイル、+2901/-268。そのうち Go は 14 ファイル、+2219/-214(テストを含む)。walkthrough を書くかどうかは `/pr` が決める

## Summary

`5850f89b...628369b9` の差分は、次の 5 つを変える。pane を作ったあとに失敗した spawn は、その pane に C-c を送ってから `herdr pane close` を打つ(1 回ごとに 10 秒の期限)。閉じられなければ non-state の event `spawn_close_failed` を書き、座席を active のまま残す。座席ごとの spawn のロック(`spawn-<org_id>_<seat_id>.lock`)が、同じ座席の 2 つ目の spawn を何も書かずに断る。途中で止まった座席の打ち直しが、エンベロープ・permission・`--scope` の検査で断られるときは、`rejected` も receipt も書かない。stale の補償は、pane の label を確かめてから閉じる。あわせて `--timeout-ms` の help が spawn 全体の 1 つの期限を言い、期限切れのエラーが値と `--timeout-ms` を上げる案内を持ち、復旧の案内が台帳を `--state-dir` で指したときはそれを繰り返す。

この branch は、`/org` skill の 4 つの写し、org runtime の仕様(FR-1、FR-9)、機能ごとの org の仕様、根拠の表、tech-debt をすでに直している。この run はコードの該当箇所を読み返してそれらが合うことを確かめ、残りの文書を掃いた。直したのは `docs/quality/quality-gates.md` の「Envelope validation」の行 1 つだけで、「Spawn rejected, recorded in manifest」が、途中で止まった座席の打ち直しに当てはまらなくなったためである。

## Changes made

| File(写しを含む) | Change |
|------|--------|
| `docs/quality/quality-gates.md`(77 行)と `templates/base/docs/quality/quality-gates.md`(76 行。この 1 行は両方で同じ文) | 「Envelope validation」の On failure の欄を「Spawn rejected, recorded in manifest」から、「…recorded in manifest; a retry of a seat whose earlier spawn is unfinished (last state event `spawn_started` or `spawn_step`) that fails the pool, role, permission, or `--scope` check is refused with nothing recorded, so the pane on record stays for `ralph org stop`」に直した。根拠は `Spawn` の Phase 1 で、`ValidateSpawnEnvelope`、`permissionArgsForDriver`、`autonomousScopeGateErr` の 3 つが `rejectUnlessInFlight` を通ること(`internal/org/spawn.go` の 818〜876 行付近)と、仕様 FR-1 に同じ例外を書いたこと。`max_seats` などの上限は stale の分岐より後の `checkCapacityAndStart` で見るので、例外に入れていない |
| `docs/reports/sync-docs-2026-10-22-seat-spawn-close.md` | 新規 |
| `docs/insights/events/2026-10-22-seat-spawn-close.jsonl` | sync_docs の event を 1 行足した(cycle 1) |

足した文にホームのパス、pane の id、token はない。`git diff --check` は rc 0。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md` の repo map(`internal/org/` の行) | 変更なし | 行は `seat saga manifest (flock-serialized), receipts, …` と書き、`spawn_failed` や `stop_failed` のような event の名前、spawn の補償、座席ごとのロックを書かない。ロックの名前を書くのは `watch.lock`、`director.lock`、`regate.lock` など、動詞をまたいで効く台帳のロックで、座席ごとのロックはそれと並べるほどの記述ではない。AGENTS.md は地図に保つ。`templates/base/AGENTS.md` と `.ralph/core/AGENTS.core.md` は `internal/org` のファイルを名指さない |
| `.claude/rules/ralph/agent-messaging.md`(と template の写し) | 変更なし | message の型と形の規約で、spawn の結果は書かない。`HELLO` の行は「seat announces itself to leader at spawn time」だけで、今も合う。ほかの rules(`git-commit-strategy.md`、`model-routing.md`、`subagent-policy.md` ほか)に spawn の失敗の扱いを書く文はない。`model-routing.md` の receipt の節は spawn の成功時の receipt の話で、今回の変更に触れない |
| `internal/org/prompts/leader.md`(go:embed。`templates/base/` に写しはない) | 変更なし。下の Left open | leader.md に spawn の失敗の扱いを書いた箇所はない。spawn に触れるのは、座席の既定の 2 席、watch の gate の拒否、承認の検査の拒否、leader の座席が動いていないときの拒否で、どれも今回の変更で内容が変わらない。`spawn_close_failed` と `HELLO が届いたのに spawn が失敗した場合` を leader に指示する文はない。leader の手引きとして指される `/org` skill の spawn の行には新しい文が入っている |
| `README.md` | 変更なし | `spawn` を書く行(112、124、232〜268 行)は、動詞の一覧、watch の gate、上限、機能ごとの org の話で、spawn が失敗したときの pane の扱い、`--timeout-ms`、ロックを書かない |
| `docs/recipes/codex-seat-permissions.md`(と template の写し) | 変更なし | 22 行の「`ralph org spawn` then reports `spawn_failed` after the `agent_start` timeout」は、pane を閉じられた場合に今も正しい(pane を閉じられなければ `spawn_close_failed` になるが、alias の誤りでは herdr は応答しているので閉じられるはず。実機では確かめていない)。pane を読んで失敗を調べる手順は、成功した spawn の pane の話(178 行)で、失敗した spawn の pane を前提にしていない。`--timeout-ms` を書く 238〜249 行は `ralph org send` と `wait` の話 |
| `docs/recipes/director-launch-cmd.md`(と写し) | 変更なし | 125、167、272〜278 行の `--timeout-ms` は `ralph org director start` の check-in の待ちで、この PR が help を直した spawn と start の flag ではない |
| `docs/quality/quality-gates.md`(ほかの行)と `docs/quality/definition-of-done.md` | 上の 1 行以外は変更なし | 「Cross-org watch gate」の行は、watch gate が断る spawn が何も書かないことを書いていて、新しい断りと矛盾しない。「Feature plan approval」の行の「a spawn is refused with a `rejected` event and a receipt, as the envelope checks are」は承認の検査の話で、承認の検査(`spawnFeatureSeatErr`)は `checkCapacityAndStart` の中で、途中の座席の補償のあとに走り、`reject()` を通る。今も正しい |
| `docs/architecture/repo-map.md`、`.ralph/`、`ralph.toml` と `templates/base/ralph.toml` | 変更なし | spawn の補償、期限、ロックを書いた文はない。`ralph.toml` の timeout の語は watch の liveness の印の話 |
| `/org` skill の 4 つの写し | 変更なし | `md5` が 1 種類。spawn の行と start の行と「無期限待機は避ける」の項の新しい文を、`Spawn` のコードと突き合わせた。10 秒の期限、spawn の期限を使わないこと、`spawn_close_failed`、ロックの断り、label を確かめる stale の補償、label が違うと触らないこと、途中の座席への検査拒否が何も書かないこと、すべて合う。stop と disband の行は変えていない(F-8 の tech-debt の行が欄の長さを引く) |
| `docs/specs/2026-08-01-org-runtime.md`(FR-1、FR-9) | 変更なし | FR-1 の例外の文は `rejectUnlessInFlight` の対象(エンベロープ、permission、`--scope`)と `--dry-run` は今までどおり記録することを書き、コードと合う。FR-9 の補償の文は、閉じられたときだけ `spawn_failed`、閉じられなければ non-state の `spawn_close_failed`、座席ごとのロック、label の確かめを書き、コードと合う |
| `docs/specs/2026-10-07-org-multi-org-director.md` と `docs/evidence/org-spec-evidence-2026-10-21.md` | 変更なし | 仕様の `[x]` は 26、`[ ]` は 6(数え直した)。PR の一覧は 31 本(`#` の番号を数えた)で、先頭の文の「31 本」と合う。根拠の表の 25/6 は verify が突き合わせ済み。Status の「試行の所見 F4 の直し」の行はブランチ名を書く(下の Left open) |
| `docs/tech-debt/README.md` | 変更なし | F4 の行、F3 の行(`#248`)、org-stop-all の行(173 行、(b) の取り消し)、新しい 3 行(338〜340 行)、大きさの行(341 行)を読んだ。大きさの行の `spawn.go` 2665 行、`spawn_test.go` 5859 行、`spawn_lock_test.go` 198 行は `wc -l` と同じ。計画のパスは `docs/plans/active/...` で、`verify.local.sh` の検査に合う |
| CHANGELOG と release のノート | 変更なし | repo に CHANGELOG はない。release のノートは goreleaser が git log から作る(`.goreleaser.yml` の `changelog.filters`、`/release` skill の手順 3)。計画の Rollout の「release のノートに書くこと」は、commit message と PR の本文に書く |
| `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh` | rc 0、rc 0 | 下の Verification |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の V3(LOW)仕様 FR-1 の拒否の記録 | 直っている(3b07848d)。今回、同じ文を `quality-gates.md` の「Envelope validation」の行にも足した |
| verify の V1・V2・V4〜V6 | 直っている(0a032b37、3b07848d)。この run で読み返して、記録の文がコードと合うことを確かめた |
| verify の V7(INFO)計画の checkbox | 628369b9 で直っている。計画は読むだけで、触っていない |
| self-review の L3・L4・L10 | 残したと self-review の追記が書いている。文書の話ではない |
| test の G3〜G6(INFO) | 文書の話ではない。tech-debt の行は立てていない(test report が直さないと決めている) |

## Plan drift notes

- 計画の AC1〜AC10 と Progress の checkbox は 628369b9 で `[x]` になっている。Progress に sync-docs の項はない(`[ ] PR created` が最後)。計画は依頼どおり触らなかったので、`/pr` か orchestrator が足す(digest の範囲の外)
- 計画の Rollout の「release のノートに書くこと」(spawn が失敗すると pane を閉じる、閉じられなければ `spawn_close_failed` で `ralph org stop`、`--timeout-ms` は spawn 全体の期限)は、PR の本文と commit message に載せる。CHANGELOG がないので、文書側の置き場所はない
- 計画の Risks の「leader の雛形は spawn の結果を見て次の手を決める」は、leader.md の今の本文では「spawn が失敗したら何をするか」を書いていないので、雛形からは読み取れない。実際には leader は spawn の終了コードとエラーの文(`spawn_close_failed` では `herdr pane close <id>` と `ralph org stop --org-id <o> --seat <s>` を含む)を直接読む。下の Left open

## Left open

- leader の雛形(`internal/org/prompts/leader.md`)に、spawn の失敗と `spawn_close_failed` の扱いを書いた箇所がない。この変更が既存の文を誤りにしたわけではなく、前からある欠けである。直すと leader の振る舞いが変わり、`TestRenderRolePrompt_Leader_FeatureOrgProcedure`(`internal/org/prompts_test.go`)が文の位置を固めているので、sync-docs の範囲では書かなかった。エラーの文が次の手を示すので、leader が従えるはずだが、実際の leader で確かめてはいない。足すなら、運用規律の節に「spawn が終了コード 1 で返り、stdout の `spawn failed:` かエラーが `spawn_close_failed` に触れていたら、エラーの `ralph org stop` を打ってから打ち直す。打ち直す前に同じ座席の spawn が動いていないかを `ralph org status` で見る」程度の 1 項目で、`/org` skill の spawn の行を指す形になる
- HELLO が届いたあとで spawn が時間切れになると、leader の受信箱に、閉じられた座席の HELLO が残る(evidence の (a) で確かめた。計画の Non-goals で消さないと決めた)。台帳の `spawn_failed` が結果であることを leader に書いた文書はない。上の項目と同じ場所に書くのがよい
- 「試行の所見 F4 の直し」の行は、仕様の Status、tech-debt、根拠の表が `fix/seat-spawn-close` とブランチ名で書いている。PR の番号は PR を作るまで分からない。直前の系列(#248)と同じく、次の PR が番号に置き換える
- 本物の herdr と agmsg での、閉じられなかった座席の `ralph org stop` と `stop --all` による片付けは、fake の herdr のテストだけが確かめている(verify の V1。文書の書き方は実機の範囲に直してある)
- 計画の Rollout の release のノートの内容(上の Plan drift notes)は、PR の本文の仕事

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/check-sync.sh`(編集のあと) | rc 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。`quality-gates.md` は KNOWN_DIFF の 1 つで、この run の行は両方で同じ文(`diff` の差は 54、57〜58、82 行のみで、この行は出ない) |
| `./scripts/check-skill-sync.sh` | rc 0。`14 skill(s) in lock-step`。`/org` skill の 4 つの写しは `md5` が 1 種類(skill には触っていない) |
| `git diff --check` | rc 0 |
| `./scripts/plan-visual.sh digest`(計画) | `550d25183df2`(承認の digest と同じ) |
| 仕様の `[x]`/`[ ]` と PR の数 | 26/6、31 本(数え直した) |
| tech-debt の大きさの行 | `wc -l` と一致(`spawn.go` 2665、`spawn_test.go` 5859、`spawn_lock_test.go` 198) |
| 足した行の走査(ホームのパス、`hiroki`、U+FFFD、token) | 漏れなし |

Go のテストと `run-verify.sh` は走らせていない(コードを変えていない。直した 1 行は `docs/quality/quality-gates.md` とその写しで、それを読む Go のテストはない。`scripts/` で名指すのは、`check-sync.sh` の KNOWN_DIFF の一覧と `verify.local.sh` の冒頭のコメントだけ)。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: leader の雛形に欠けた spawn の失敗の扱いが実際の leader の動きに与える影響、閉じられなかった座席の実機での片付け、Go のテスト全体(`/test` が済ませた)

## Insight event

`./scripts/insights-append.sh --slug seat-spawn-close --flow standard --phase sync_docs --verdict pass --source skill --cycle 1` で `docs/insights/events/2026-10-22-seat-spawn-close.jsonl` に 1 行足した(cycle 1)。verify と test の event と同じファイルで、日付またぎの食い違いはない。
