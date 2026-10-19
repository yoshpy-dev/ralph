# org-director-wait-bound

- Status: Approved
- Approved: 2026-10-19 sha256:505a695e5548
- Owner: Claude Code
- Date: 2026-10-19
- Related request: 9 段の導入順の 8 段目の後半の 2 本目の残り(8b-2c)。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-10(「Claude Code は Monitor で待ち」)と、tech-debt の行「director の `ralph org wait --inbox --director <id>` が、打ったセッションの終わりで止まらない」(8b-2b の B)
- Related issue: N/A
- Type: fix
- Branch: fix/org-director-wait-bound

## Objective

Claude Code の director が受信箱を待つ wait を、時間を区切って打ち直す形にする。いまの `/director` skill は `--timeout-ms 0`(時間切れのない wait)を Monitor で見張らせ、signal での終わりを「失敗、打ち直さない」とする。Monitor は `timeout_ms`(既定 5 分、最大 30 分)で処理に SIGTERM を送るので、Monitor を選んだ director は 30 分以内に受信箱を待つのをやめ、人に失敗を伝えて止まる。これを直す。

あわせて 8b-2b の B に、Go を変えずに上限を付ける。計画の前の実験(`docs/evidence/director-wait-monitor-2026-10-19.md`)では、Monitor で始めた処理は既定の `/exit`、「Move to background and exit」、pane を閉じたとき、Monitor の期限のどれでも SIGTERM で止まり、残るのは claude の SIGKILL だけだった。区切った wait は、残っても区切りの時間のうちにひとりでに終わる。

## Scope

- `/director` skill の「wait で待つ」の Claude Code の箇条と、終了コードの扱い(4 つの写し: `.claude/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、その `.agents` の 2 つ)
  - Claude Code は `ralph org wait --inbox --director d1 --timeout-ms 1500000 2>&1` を Monitor で見張り、Monitor の `timeout_ms` に `1800000` を渡す。Bash の `run_in_background` は代わりの手として残し、Move to background と claude の SIGKILL のあとも残ること、`note:` の行は wait が終わるまで届かないことを書く
  - 時間切れ(終了コード 1 で、stderr が `no open inbox item arrived within` を含む)は、Codex と同じく打ち直す
  - Monitor が期限切れを知らせたあとの signal での終わりは、Monitor が止めたので打ち直す。それ以外の signal での終わりは、いままでどおり失敗
- skill の文を固定する Go のテスト(`internal/org/director_skill_sync_test.go`)
- 記録: 実験の evidence(この計画と一緒にコミットする)、本物の ralph を Monitor の下で打つ実機の確認の evidence、tech-debt の B の行の書き換え、仕様の FR-10 への 8b-2c の注

## Non-goals

- wait が持ち主のセッションの終わりを見つけて止まる仕組み。印(`director-wait.json`)に親や祖父の pid を足して確かめる案は採らない。実験では wait の親は `zsh -c` のシェルで、claude が SIGKILL されてもシェルは pid 1 に付け替わるだけで、wait の ppid は変わらない。wait と claude のあいだの段数はシェルの包み方(fork か exec か)で変わる。codex の wait の親は app-server の daemon で、この見方では何も分からない
- 印に wait の区切り(`timeout_ms` や終わる予定の時刻)を足すこと。新しい wait が印を引き継ぐと、残った wait は印に出なくなるので、読む場面がほとんどない
- Go のコード(`ralph org wait` の既定、`--timeout-ms 0` を断ること)と `--help` の文。`--timeout-ms 0` は `--director` のない wait と人の打つ wait で使われる
- codex の wait の形(`--timeout-ms 300000` を繰り返す)。codex は実験の対象にしていない
- headless の director の役割の雛形(`internal/org/prompts/director.md`)。42〜44 行は「skill のとおり `--timeout-ms` を足して打つ」と書き、数字を持たないので、そのままで skill の変更に従う
- Monitor の期限そのものを変えること、Claude Code の外の仕組みで wait を見張ること

## Assumptions

- Claude Code 2.1.291 の Monitor の `timeout_ms` の上限は 1800000(30 分)で、期限で処理に SIGTERM を送り、セッションに「Monitor expired」の知らせを出す(実験の m4。上限はツールの説明による)
- `ralph org wait --inbox --director` は SIGTERM を受けると印に `end=signal` を書き、同じ signal で終わる(8b-2b の実機の確認の (f))
- `--timeout-ms` で区切った `--director` の wait は、時間切れで印に `end=timeout` を書き、終了コード 1 と時間切れの文で返る(8b-2b の実機の確認の (d))
- skill の 4 つの写しは `scripts/check-skill-sync.sh` が同じであることを確かめ、`.agents` の写しは `scripts/sync-skills.sh` で作る

## Affected areas

- `.claude/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md`(「受信箱の処理」の「wait で待つ」と、その下の終了コードの扱い)
- `internal/org/director_skill_sync_test.go`(テストを 1 本足す)
- `docs/evidence/director-wait-monitor-2026-10-19.md`(実験、計画と一緒にコミット)、`docs/evidence/director-wait-bound-livefire-2026-10-19.md`(新しい実機の確認)
- `docs/tech-debt/README.md`(B の行)
- `docs/specs/2026-10-07-org-multi-org-director.md`(FR-10 の注)

## Visual review

None(図解ページは作らなかった。書き込みの途中で止まったため。変更は skill の 4 つの写しの 1 箇所、Go のテスト 1 本、記録の文で、待ち方の流れは Design decisions の D1〜D4 に書いた)。lint と self-check は、ページがないので打っていない

## Design decisions

- D1 Claude Code の wait は Monitor で、時間を区切る: `ralph org wait --inbox --director d1 --timeout-ms 1500000 2>&1` を Monitor で見張り、Monitor の `timeout_ms` は `1800000` にする。仕様の FR-10 が Monitor を名指し、実験では Monitor の処理のほうが Bash の `run_in_background` より残る道が少ない(Move to background で止まる)。wait の区切り 25 分を Monitor の上限 30 分より 5 分短くし、wait の時間切れが Monitor の期限より先に来るようにする。consult(consult-plan-monitor)の判定による
- D2 Bash の `run_in_background` は代わりの手として残す: 同じコマンドで打てるが、Move to background と claude の SIGKILL のあとも残る(区切りの 25 分のうちに終わる)こと、`note:` の行と `wait --inbox --director: ` の行は wait が終わったときにまとめて届くことを書く
- D3 打ち直す終わりと失敗を分ける: 時間切れ(終了コード 1 と時間切れの文)は打ち直す。Monitor が期限切れを知らせたあとの signal での終わり(director が `timeout_ms` を渡し忘れて既定の 5 分で止められた場合など)は、Monitor が止めたので打ち直し、次から `timeout_ms` に 1800000 を渡す。それ以外の signal での終わり、panic、時間切れの文のない終了コード 1 は、いままでどおり失敗で打ち直さない
- D4 B は区切りで上限を付けるだけにする: 残った wait(Monitor では claude の SIGKILL、Bash では加えて Move to background)は、新しい wait が印を引き継いだあとも最長 25 分待ち続け、件が来れば返して終わる。返した件は ack されないので open のまま残り、次の wait が返す。tech-debt の B の行は閉じず、この残り方と根拠に書き換える
- D5 テストで数字を固定する: skill の 4 つの写しそれぞれで、Claude Code の箇条にある wait の `--timeout-ms` の値 N と Monitor の `timeout_ms` の値 M を読み、`0 < N < M <= 1800000` を確かめる。どの写しにも `--timeout-ms 0` が残っていないことも確かめる。`director_skill_sync_test.go` の既存のテストと同じ書き方(repo root を `runtime.Caller` で求める、写しがなければ skip)にする
- D6 実機の確認: 分離した headless の herdr server の pane で claude(haiku)を動かし、分離した台帳(`--state-dir`)とこのブランチで作った binary で、(a) 区切った wait が Monitor の下で時間切れで返る、(b) `--timeout-ms 0` の wait が Monitor の期限で SIGTERM を受けて `end=signal` になる(直す前の不具合の再現)、(c) 既定の `/exit` で wait が SIGTERM で止まる、(d) claude の SIGKILL のあとも残った区切りの wait が、区切りの時間で `end=timeout` を書いて終わる、を確かめる。さらに、直した skill の「wait で待つ」と終了コードの扱いに従う claude(sonnet)のセッションで、人の入力を足さずに、(e) 時間切れの wait を 2 回以上打ち直し、そのあと escalate した件を返して ack し、また待つ、(f) Monitor の `timeout_ms` を wait の区切りより短く渡させ、Monitor の期限切れのあとに打ち直す、(g) 対照として、外から wait に SIGTERM を送ると失敗として打ち直さずに人に伝える、を確かめる。(e)〜(g) では Monitor の task id、印の `wait_id`、終わり方を記録する(Codex の計画助言 1)。区切りは確認のために短くし(wait は 20〜30 秒、Monitor は 60 秒など)、短くした数字はセッションへの指示で skill の数字に替えて渡す。人の herdr と端末には触らず、signal は確認で起動したプロセスにだけ送る
- Critical forks: なし(D1 は仕様の FR-10 と consult の判定、D4 は consult の判定で決まった)

## Acceptance criteria

- [ ] AC1: skill の 4 つの写しの「wait で待つ」の Claude Code の箇条が、`ralph org wait --inbox --director d1 --timeout-ms 1500000 2>&1` を Monitor で見張り、Monitor の `timeout_ms` に `1800000` を渡すと書く。`--timeout-ms 0` はどの写しにも残らない
- [ ] AC2: 同じ箇条が、Bash の `run_in_background` を代わりの手として書き、Move to background と claude の SIGKILL のあとも wait が残ること(25 分のうちに終わる)と、`note:` の行が wait の終わりまで届かないことを書く
- [ ] AC3: skill の終了コードの扱いが、時間切れ(終了コード 1 と `no open inbox item arrived within`)を Claude Code でも打ち直すと書き、Monitor の期限切れの知らせのあとの signal での終わりを打ち直す終わりとして、それ以外の signal での終わりと分けて書く
- [ ] AC4: `internal/org/director_skill_sync_test.go` に足したテストが、4 つの写しで `0 < N < M <= 1800000` と `--timeout-ms 0` がないことを確かめ、skill を前の形(`--timeout-ms 0`)に戻すと落ちる
- [ ] AC5: `./scripts/check-skill-sync.sh` が通り、4 つの写しが同じ
- [ ] AC6: 実機の確認 (a)〜(g) の結果が `docs/evidence/director-wait-bound-livefire-2026-10-19.md` にある。(e) では人の入力なしに時間切れの打ち直しが 2 回以上続き、そのあとの件が返って ack されている。(f) では Monitor の期限切れのあとに wait が打ち直されている。(g) では外からの SIGTERM のあとに wait が打ち直されていない
- [ ] AC7: tech-debt の B の行が、Monitor と Bash それぞれで残る場合、残った wait の上限(25 分)、残った wait が返した件の行き先、親の pid で見分けない理由を書き、閉じていない
- [ ] AC8: 仕様の FR-10 に 8b-2c で決めたことの注がある
- [ ] AC9: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が通る

## Implementation outline

1. 計画と実験の evidence をコミットする
2. S1(implementer): skill の 2 つの元の写しを直し、`scripts/sync-skills.sh` で `.agents` の 2 つを作り直し、テストを足す。1 コミット
3. S2(orchestrator が自分で打つ): 実機の確認 (a)〜(g) を打ち、evidence を書く。pane の claude との対話が要るので、handoff に向かない
4. S3(implementer): tech-debt の B の行と仕様の FR-10 の注を直す。1 コミット
5. run-verify、そのあと self-review → verify → test → sync-docs → cross-review → pr

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、shellcheck、`check-skill-sync.sh` を含む)
- Spec compliance criteria to confirm: FR-10 の「Claude Code は Monitor で待ち」と skill の箇条が合う。codex の箇条と headless の雛形が変わっていない
- Documentation drift to check: README の director の段落(「waits on the inbox with `ralph org wait --inbox --director <id>`」)と `docs/recipes/director-wake.md` が `--timeout-ms 0` を書いていないこと。`internal/org/prompts/director.md` が数字を持たないこと
- Evidence to capture: verify report、`check-skill-sync.sh` の出力

## Test plan

- Unit tests: 新しいテスト(D5)。`go test ./internal/org/ -run TestDirectorSkill -count=1`、`GOTOOLCHAIN=go1.25.8` でも
- Integration tests: なし(skill の文と Go のテストだけ)。実機の確認は S2 で、(e)〜(g) が skill に従うセッションの打ち直しの分かれ目を確かめる
- Regression tests: `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase`(時間切れの文を skill が名指す)と `TestDirectorSkill_PinsTheAutoMergeSteps` が通り続ける。`./scripts/run-test.sh`
- Edge cases: 写しの 1 つだけ数字が違う、N が M 以上、M が 1800000 を超える、Monitor の数字がない、`--timeout-ms 0` が残る、のそれぞれでテストが落ちること(mutant で確かめる)
- Evidence to capture: test report、mutant の結果

## Risks and mitigations

- R1 director が Monitor の `timeout_ms` を渡し忘れ、既定の 5 分で止められる: D3 で、Monitor の期限切れの知らせのあとの signal での終わりは打ち直すと書く。止まるのではなく、5 分ごとに打ち直すだけになる
- R2 25 分ごとに director の手番が 1 回増える: 費用は 1 時間に 2〜3 回の短い手番。codex の 5 分ごとより少ない
- R3 区切りを付けても残った wait は最長 25 分残り、件を返して ack しないまま終わる: 件は open のまま次の wait が返すので失われない。tech-debt の行に書く
- R4 テストが skill の文の書き方に強く結びつく: 数字を読む正規表現は Claude Code の箇条の 1 行(`--director d1 --timeout-ms <数字>`)と `timeout_ms` に `<数字>` の形だけを見る。失敗の文に直し方を書く
- R5 文を直しても、それを読んだセッションが待ち直さない(終了の知らせの受け取り方が想定と違う): 実機の確認 (e)〜(g) で、skill に従うセッションの振る舞いを見る。待ち直さなければ skill の文を直してから先に進む
- R6 実機の確認で人の herdr や claude に触れる: 8b-2b と同じく XDG の 3 変数で herdr を分け、確認で起動したプロセスにだけ signal を送り、終わったら分離した場所を消す

## Rollout or rollback notes

- skill の文だけの変更なので、戻すときはこの PR を revert する。binary の変更はない
- 配布: 新しい skill は 8b-2b と同じ `--director` を使うので、必要な binary は 8b-2b のもの。release のノートに「Claude Code の director の wait は 25 分で区切り、Monitor の期限で止まらない」と書く(PR の本文と memory に書く)

## Open questions

- Bash の `run_in_background` の時間の上限: 計画の前の確認では 12 分 19 秒走って止められなかった(evidence の Control)。それより長い上限があるかは確かめていないが、25 分の区切りの wait が先に終わるので、この計画の形には関わらない

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- 2026-10-19: Codex の計画助言(gpt-6-astra、xhigh、rc 0)は MEDIUM 1 件。実機の確認が wait 1 回ごとの終わりしか見ず、skill に従うセッションが人の入力なしで待ち直すことを確かめていない。D6・AC6・Test plan・R5 に (e)〜(g) を足して直した。ユーザーの永続の指示により、計画は自動承認した(digest 505a695e5548)。図解ページは作っていない(Visual review の節)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
