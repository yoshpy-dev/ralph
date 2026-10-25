# Verify report: org-recheck

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-24-org-recheck.md`(AC1〜AC9、D1〜D12。パイプライン cycle 1/2。`./scripts/plan-visual.sh digest` は承認の行と同じ `1e218f3d7907` を返した)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff origin/main...HEAD`(HEAD 7f004c11、base は origin/main 98d38391 で、189cd55d で取り込み済み。49 ファイル、+2637/-51)。記録と手順書だけの差分で、Go とシェルの変更はない。self-review の直し a688fcf0 と、その記録 7f004c11 を含む。テストは実行していない(`/test` の範囲)
- Evidence: `run-static-verify.sh` が自分で書いた `docs/evidence/verify-2026-10-25-171155.log`(git が無視する場所。Timestamp は UTC の 2026-10-25T17:11:55Z)。依頼どおり、コミットするのはこの report と insight event だけにした

## Spec compliance

evidence の `sh` ブロック 29 個は、worktree の root で 1 つずつ `sh -c` で打ち、出力を直上の表の本文、直下の `text` ブロック、またはその 2 つを続けたもの(Setup の models の表と、Numbers の shim の表の 2 個)と行ごとに比べた(scratchpad の `python3 -I` のスクリプト。リストの中で字下げされた囲みは字下げを外して比べた)。29 個とも一致した。終了コードが 1 だったのは F11 の command E の 1 個だけで、`grep -c '^wip:'` が 0 件のときの値であり、出力は `text` と一致する。比べ方の対照として、期待値を 2 か所(`6 autonomous` を 7、Cleanup の `detached d1` を d2)変えた写しでは、その 2 個だけが不一致になった。Merges の `git log --oneline ffb3e271..98d38391` の囲み(`sh` ブロックなし)も、今の履歴の 15 行と一致した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 環境 | 満たす。道具の版の一部は計画の Assumptions から | evidence の Setup(`:157-248`)に、binary の commit ffb3e271(PR #254 の merge commit で origin/main の祖先)、wrapper の 16 個の変数の名前と、token を含まない値、shim、agmsg の写し(`RALPH_ORG_AGMSG_HOME`、D2 の deviation)、models の表がある。手順書の節ごとの記録(`:250-334`)は、節ごとに「as written」か deviation とその理由を書く。herdr と Claude Code の版は実行中に読み直しておらず、evidence 自身がそう書く(`:172-174`、Not verified)。漏れの grep は Observational checks に書いた |
| AC2 2 つの org | 満たす | manifest の `scope_reserved` の 2 件が `split=recheck digest=4f955a6f4269` を持ち、重なりは 2773 s(46 min 13 s)。`gh pr view 255` と `256`(読むだけ)は `MERGED`、head は `fix/backfill-na` と `fix/hooks-matcher`、変更は 5 ファイル +322/-2 と 5 ファイル +326/-1 で、evidence の Two orgs(`:380-422`)と一致する。`git diff --name-only ffb3e271 98d38391` の 10 ファイルに `docs/tech-debt/README.md` はない |
| AC3 所見の直しの効き | 満たす(表はある。観察の一部は director の notes による) | Fix effects の表(`:426-439`)に、F3、F4、F5、F6、F13、F7、F11、F8〜G1、開いたまま残る 2 行、gate の所要、フォルダ信頼の 12 行があり、どれも期待、観察、読んだ場所、判定を持つ。写しで出し直せたもの: `failed to ALERT` 0 行、`watcher error` 0 行、`WATCHDOG ESCALATION` 8 行、`spawn_failed`・`spawn_close_failed` 0 件、`notified` 0 件、shim の 12 行はすべて題が "Claude Code" で ralph の題は 0 行、watchdog の receipt 13 件、`wip:` 0 件(15 コミット中)。計画との読み先の違い(F3 は `herdr agent list` でなく見張りの `agent_get`、F11 は cleanup 前のブランチでなく main の履歴、F7 の `send_failed` という event は ralph にない)は Trial deviations(`:1162-1184`)に書いてある。load average、send の終了コード、pane の読みは notes にだけある |
| AC4 merge | 満たす | 1 本目の merge の前に、両方の `gate_passed` が base ffb3e271 を持ち、両方が disband していた(manifest の 03:09:33Z、03:10:18Z、03:16:07Z、03:16:47Z)。`pr_merged` は #255 が `auto_merge=false`(merge commit 536271a2)、#256 が `auto_merge=true`(98d38391)。#256 の前に `via=regate` の `gate_passed`(14:18:21Z、head 77c631b2、base 536271a2)がある。3 つの merge commit は origin/main の祖先で、98d38391 の親は 536271a2 と 77c631b2。`gh` の mergedAt(14:03:08Z、14:22:29Z)は evidence と同じ。`merge-order` が `rebase` と出したことは notes にだけある |
| AC5 数値と前回との比較 | 満たす | 介入、承認待ち、受信箱の処理、見張りの escalation、費用、gate の表がそれぞれ `sh` ブロックで写しから出る(上の 29 個のうち)。前回との比較の表(`:1139-1160`)の試行側の値のうち、「first start to the last disband」2 h 51 min 55 s(10315 s)、今回の 52 min 53 s(3173 s)、試行の `spawn_failed` 4 件と `rejected` 0 件は、2 つの写しの manifest から出し直して一致した。ほかの試行側の値(7 回・1302 s、109 s と 423 s、213 s と 997 s、$12.81、1375 s・1664 s・792 s)は `docs/reports/verify-2026-10-20-org-trial.md` と `verify-2026-10-23-trial-docs-wrapup.md` に書かれた値と同じ |
| AC6 記録 | 満たす | tech-debt の 2 行は RESOLVED になった。126 行と 127 行の `checkCodexEffectiveConfig` の行は #256(1c31dcf9、77c631b2、98d38391)、146 行と 147 行の backfill の行は #255(5babe702、536271a2)。どちらも直前に `<!-- RESOLVED ... -->` の注を持つ。注が挙げる `hasNonStateHooksKey`(`internal/cli/doctor.go:406`)と `reviewerStatusRe`(`internal/insights/backfill.go:428`)、テスト 5 つ(`func <名前>(` がそれぞれ 1 件)、reviewer の report 2 つは HEAD にある。直さない所見は RC1〜RC6 と RC9〜RC11 の 9 行(363〜371 行)。開いたまま残る 2 行(345 行と 346 行)には今回の 0 回と 29 回、13 回が書き足された。仕様の Status(`:35`)と根拠の表の新しい節(`:44-65`)に結果と evidence への参照がある |
| AC7 手順書 | 満たす | 2 つの写しは `cmp` で一致した(579 行)。`check-sync.sh` は `templates/base` の全ファイルを root と比べ(`scripts/check-sync.sh:201`)、DRIFTED 0、ROOT_ONLY 0 だった。「The steps were not run end to end again」と「An end-to-end run of these steps as written」の 2 文は消え、冒頭(`:17-24`)と「What this recipe does not cover」(`:572-579`)が今回の実行を書く。D2 の渡し方は `:174-184`、RC7 の `/tmp/ralph-iso/run herdr` は `:386-389`、自分の端末での trust の失敗は `:393-397`、RC8 の語順は `:407-420`、`.git/info` は `:82`、機能の worktree の片付けは `:529` にある。見出しの一覧は base と同じ |
| AC8 片付け | 記録はある(実行は repo から確かめられない) | Cleanup の 8 段(`:1186-1224`)は AC8 の順に並ぶ。写しの痕は順と合う: `director.jsonl` の `detached` 14:23:22Z、`watch-alive.json` の `stopped` 14:23:34Z、`director-wait.json` の `end=watch` 14:23:36Z、`ralph-status-final.txt` の `org watch: not running (stopped at 2026-10-25T14:23:34Z)`。AC8 は watch の停止の直後に `pgrep -fl /tmp/ralph-iso` を求めるが、evidence はその時点の `pgrep` を「ralph の watch がない」と書き、`pgrep -fl /tmp/ralph-iso` の終了コード 1 は `rm -rf` のあと(8 段目)に置く。各段の終了コード、利用者の herdr と agmsg の更新時刻は notes にだけある |
| AC9 写し | 満たす | 写しの 39 ファイルはすべて追跡されている(`.log` の 3 つは `git add -f`)。`jq` は上のとおり 29 個とも表になる。漏れの grep は Observational checks |

仕様の印と根拠の表は次のとおり確かめた。

- 数: 仕様は `[x]` 27・`[ ]` 5 の 32 項目(base は 29・3)、根拠の表の「項目」の判定の列は `[x]` 26・`[ ]` 5 の 31 行。差の 1 は表の前文が外すと書く FR-12 で、FR-1〜FR-11、NFR 5 つ、受け入れ条件 15 個を項目ごとに並べると印の違いは 0。根拠の表 `:105` の「26 と 5」「2026-10-23 までは 28 と 3」も、仕様 `:35` の「2026-10-25 の時点で `[ ]` は 5 項目」も、この数と合う
- `[ ]` の 2 項目(FR-4 `:78`、受け入れ条件 5 `:170`)は、どちらも次の項目の前に「印(2026-10-25)」の注を持つ(`:87`、`:172`)。注の「implementer は断られた 6 秒後と 8 秒後」は、manifest の `rejected`(02:31:47Z、02:31:49Z)と pane-3・pane-4 の最初の event(02:31:53Z、02:31:57Z)から出る。leader の雛形に `--scope` も `--reserve` もないこと(`internal/org/prompts/leader.md` で 0 件、f0668004 の版でも 0 件)と、断りの文(`internal/org/spawn.go:1696`)も合う
- 印の規則(根拠の表 `:5`)との突き合わせ: RC1〜RC12 の「食い違う」の列で「はい」は RC2 だけで、その 2 項目は `[ ]` になっている。ほかの RC が挙げる `[x]` の項目(FR-5、FR-6、FR-7、FR-9、NFR 記録の置き場所)について、列の理由を項目の文と照らした。RC6 の send は 10 件とも `sent` があり、RC9 の判断は FR-9 の 8b-2b の注(`:134`、30 秒ごとの読み直しと 5 秒ごとの確かめ)と合い、RC10 は `reject`(`internal/org/spawn.go:2160`)が `honored=false` を書く意図どおりの動きである。規則に反する `[x]` は見つからなかった。判断が分かれうるとされた RC1(FR-7)と RC6(FR-5)の理由も、項目の外の回り道だったという説明で筋が通る

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed、HEAD 7f004c11) | rc 0、`All verifiers passed.` | `Language scope: changed (docs_only)`、language pack は選ばれず、Go は走っていない。shellcheck、hook の `sh -n`(20 個)、guard の awk の構文、`jq -e`(settings.json 2 つ)、Codex の hook の 3 検査、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照が OK。`secret-scan-branch.sh` は `98d38391..7f004c11` で clean |
| `./scripts/check-sync.sh`(単独) | rc 0 | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。手順書は非一致の一覧に出ない |
| `./scripts/check-skill-sync.sh`(単独) | rc 0 | 14 skill が一致 |
| `cmp docs/recipes/isolated-herdr-server.md templates/base/docs/recipes/isolated-herdr-server.md` | rc 0 | 579 行 |
| `git diff --check origin/main...HEAD` | 出力なし、rc 0 | 前回の試行の写しにあった行末の空白はない |
| tech-debt の表の欄の数(`(?<!\\)\|` で分割) | 変わった行はすべて 5 欄 | 変わった表の行は 127、147、320、345、346、362〜371。5 欄でないのは 22 行(6 欄)、32 行(4 欄)、281 行(9 欄)で、base でも同じ行(279 行が 2 行ずれた)が同じ数だった |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 手順書の 2 つの写し | はい | 上の `cmp` と `check-sync.sh` |
| 消えた文「not run end to end」の参照 | はい | HEAD でこの文を引くのは、計画の AC7(引用)と、tech-debt の RESOLVED の行 319・320 だけ。後者は元の文を残して「Added 2026-10-25」「added 2026-10-25」の書き足しを持つ(前例どおり)。仕様、根拠の表、`docs/recipes`、`templates/base/docs`、`/org` と `/director` の skill、README、`docs/quality` に、手順書を通していないとする文は残っていない |
| `/org` skill の前提の参照(`.claude/skills/org/SKILL.md:67` と 3 つの写し) | はい(INFO-2) | 手順書の見出しは base と同じで、skill が挙げる中身(XDG の変数、台帳を分ける repo、git と gh の設定、pane の中の `PATH`、shell の alias、始める前の確かめ、片付け)はどれも残っている |
| evidence の所見が引くコード | はい | `DirectorWaitMarkInterval`、`DirectorWaitWatchProbeInterval`、`realClaudeInvoke`、`PaneRead`、`HonoredRate`、`ResolveAgmsgHome`、`EventSent`、`(*Org).reject` が `internal/` にある。evidence、手順書、新しい tech-debt の行が backtick で引く repo の path 69 個は、gitignore の `.claude/worktrees/`、ブランチ名 `docs/org-recheck`、接頭辞 2 つ、scaffold 先の `docs/tech-debt/.gitkeep` を除いてすべてある |
| tech-debt の RESOLVED の行と merge | はい | 上の AC6。`git log --oneline origin/main` の先頭 3 行が #256 の merge、取り直しの merge 77c631b2、#255 の merge |
| 試行の evidence の Correction の段落(`docs/evidence/org-trial-2026-10-20.md:475`) | はい | 古い文を消さず、日付つきで足してある。18/20 の数は evidence の 29 個のブロックの 1 つ(`:1115`)で出し直せた |
| self-review の F1〜F6・N1〜N4 の直し | 1 か所を除いて、はい(LOW-1) | F1 の「どの org でも毎回」、F2 の command B、F6 の冒頭は、grep で残りがない。F3 の「watcher の 1 回ごとに通知」は、根拠の表の RC11 の行に残っている |
| 計画の Progress と AC の印 | INFO-3 | |

## 計画の archive のあとの参照

`scripts/archive-plan.sh` は `docs/tech-debt/README.md` だけを書き換える(`:65` の `readme=` と `:73` の awk)。scratchpad に tech-debt、計画、evidence 2 つ、仕様、self-review の report を写し、`sh scripts/archive-plan.sh 2026-10-24-org-recheck.md` を打った。「Updated 12 reference(s)」と出て、計画は `docs/plans/archive/` に移った。`docs/plans/active/2026-10-24-org-recheck.md` を指す参照は HEAD に 16 個あり、行をまたぐものと、path を付けないものはない。

| 場所 | archive のあと |
| --- | --- |
| `docs/tech-debt/README.md` の 127、147、362〜371 行(12 個。362 行は既存の行への書き足し) | `docs/plans/archive/` に書き換わる |
| `docs/evidence/org-recheck-2026-10-24.md:3`(冒頭の `Plan:`) | `docs/plans/active/` のまま残り、切れる |
| `docs/specs/2026-10-07-org-multi-org-director.md:35`(Status の段落) | 同じく切れる |
| `docs/evidence/org-spec-evidence-2026-10-21.md:46`(根拠の表の「通しの確かめ(2026-10-24)」の節) | 同じく切れる |
| `docs/reports/self-review-2026-10-26-org-recheck.md:4` | 同じく切れる(この report の冒頭も同じ) |

切れる 4 個は前例と同じ扱いになる。試行の evidence の `:3` は `docs/plans/active/2026-10-20-org-trial.md` を、根拠の表の `:3` は `docs/plans/active/2026-10-21-org-spec-complete.md` を指したままで、どちらの計画も今は `docs/plans/archive/` にある。読み手は同じファイル名を archive で引けばよい。直すなら `/pr` の archive の手順の外で、別に書き換えることになる。

## Observational checks

- 漏れの grep(差分の 49 ファイル): `/Users/` は 3 件で、計画の AC9 と Verify plan、self-review の report が grep の対象として書いた語だけ。`/home/`、`/var/folders`、OS のログイン名と会社の domain は self-review の report の 15 行目の grep の語の一覧だけ(前の self-review の report にも同じ書き方がある)。UUID の形、`gh[pousr]_`、`github_pat_`、`xox`、`eyJ`、`BEGIN ... KEY`、`Bearer`、pane id の形(`w<N>:p<M>`、`p_`、`w_`)は 0 件。`sk-` の 6 件は `task-worktree` などの語の一部
- U+FFFD は 49 ファイルとも 0 件
- 写しの 7 桁以上の 16 進は 29 種で、24 種は repo にある commit、残る 5 種は計画の digest(`4f955a6f4269`、`1d3a60a7cd8b`、`fc16043d3a2d`)と `content=` の sha256 の 2 つ
- 写しの JSON の id の値: `pane_id` は `pane-1`〜`pane-6` と `ws-1`・`ws-2`、`workspace_id` は `ws-N`、`tab_id` と `terminal_id` は `tab-N` と `term-N`、watch-status の `agent_session` の値は `session-1`〜`session-6`、`by=` と `user` は `user-1`。`alert_id` と `wait_id` は nanosecond の時刻と pid で、乱数ではない
- `gh pr view 255` と `gh pr view 256`(読むだけ)を打った。ほかに外へ出る操作はしていない

## Findings

- LOW-1(`docs/evidence/org-spec-evidence-2026-10-21.md:62`、根拠の表の RC11 の行の「食い違う」の欄): 「分離した環境の外では、こうした hook を持つ利用者に、watcher の 1 回ごとに通知が 1 つ出ることになる(確かめていない)」が残っている。同じ行の「内容」の欄は「watcher の 13 回のうち 12 回」と書き、evidence の RC11 は最初の 1 回に shim の行がないと書く。self-review の F3 は手順書、tech-debt の RC11 の行、evidence の RC11 の 3 か所をこの所見の言い方に直したが、この 1 か所は直していない。括弧で「確かめていない」と断っているが、隣の数と食い違う。直すなら「watcher の多くの回(今回は 13 回のうち 12 回)で通知が 1 つ出る」の形にそろえる。根拠の表は `docs/reports/` の task の記録ではないので、記録だけの修正の例外には当たらない
- INFO-1(`docs/specs/2026-10-07-org-multi-org-director.md:29`): 印の数を書く段落は「2026-10-23 の時点で、3 項目(FR-3、FR-10、ツール非依存)は `[ ]` のまま残る」のままで、今の数(5 項目)は 6 行下の新しい段落(`:35`)にある。どちらも日付つきなので誤りではないが、`:29` で読み終える人は 3 項目と読む。前の PR はこの文の日付と数を書き換えてきた
- INFO-2(`.claude/skills/org/SKILL.md:49-55` と 3 つの写し): `/org` skill のフォルダ信頼の項は、人が main のチェックアウトで claude を起動して信頼すると書く。今回、分離した環境の clone ではこの道(計画の D6)で画面が出ず記録も残らず(RC7、理由は分かっていない)、手順書だけが捨ての pane で答える道に直された。分離しない環境でも同じことが起きるかは確かめていない。skill は分離した環境では手順書を指すので食い違いではないが、理由が分かるまでの注意として記録する
- INFO-3(計画 `:155-158` と Acceptance criteria): self-review の report はあるが「Review artifact created」は `[ ]` のまま。AC1〜AC9 も 9 つとも `[ ]`。`/pr` までに直せばよい
- INFO-4(`docs/reports/self-review-2026-10-26-org-recheck.md:64`): Follow-ups は insight event の追記先を `docs/insights/events/2026-10-26-org-recheck.jsonl` と書くが、`insights-append.sh` は UTC の日付でファイルを選び、実際の追記先は `docs/insights/events/2026-10-25-org-recheck.jsonl` だった(1b4d2ae1)。このファイルは task の記録なので、直すなら記録だけの修正の例外で直せる

## Coverage gaps

- 実行そのもの(S1〜S3、S7)は repo から打ち直せない。notes にだけある事実(送信と各段の終了コード、load average、pane の画面、`merge-order` の出力、利用者の herdr と agmsg の更新時刻、`pgrep` と `lsof` の結果、1 回目の写しと最後の写しの `cmp`)は確かめていない
- 座席の session の記録(`~/.claude/projects/` の下)を開いていない。RC1 の「6 つのうち 5 つ」と費用の表の元の数は、写しの `seat-usage.tsv` までを確かめた
- 実行中の herdr と Claude Code の版(evidence も計画の Assumptions に頼ると書く)
- RC9 の読む順序、RC10 の `ralph insights` の数え方、RC11 の hook の同定は、evidence と同じくコードと notes からの推論のまま
- テストの実行(`./scripts/run-test.sh`、`GOTOOLCHAIN=go1.25.8` を含む)は `/test` の範囲
- 手順書の手順は打っていない。今回の実行の記録とコードに照らして確かめた

## Verdict

- Verdict: pass
- Verified: AC1、AC2、AC4〜AC7、AC9 を、evidence の 29 個のブロックの打ち直し(対照つき)、写しの manifest・inbox・receipts・shim のログ、git の履歴、`gh pr view 255` と `256`(読むだけ)、手順書の 2 つの写しの `cmp`、tech-debt の欄の数、仕様と根拠の表の印の項目ごとの突き合わせで確かめた。静的解析(`run-static-verify.sh`、`check-sync.sh`、`check-skill-sync.sh`、`git diff --check`)はすべて rc 0
- Partially verified: AC3 は表の 12 行と写しから出る数までを確かめ、load average、終了コード、pane の読みは notes による。AC8 は、順の記録と写しの痕(`detached`、`stopped`、`end=watch`、`ralph status` の停止の行)が合うことまでで、各段の実行は確かめられない
- Not verified: 上の Coverage gaps。archive のあと 4 個の参照が `docs/plans/active/` のまま切れることは、前例と同じ扱いとして記録した
- Findings: LOW 1 件(LOW-1、self-review の F3 の直しの残り)、INFO 4 件。LOW-1 は根拠の表の 1 文で、直すには cycle 2 の `/self-review` から回し直すか(上限 2 回のうち 2 回目)、PR の本文に既知の穴として書く
