# sync-docs report: org-recheck

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-24-org-recheck.md`(承認の digest `1e218f3d7907` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run は計画を触れていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `98d38391`(origin/main)、branch は `docs/org-recheck`。run 開始時の HEAD は `acdac00a`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-26-org-recheck.md`(Merge yes。LOW 6 件と INFO 4 件は a688fcf0 で直った)、`docs/reports/verify-2026-10-26-org-recheck.md`(pass。LOW 1 件は直り、INFO 4 件が残った)、`docs/reports/test-2026-10-26-org-recheck.md`(pass。Test gaps 4 件。文書を読む Go のテストがないこと、手順書を頭から打つテストがないことなど)
- 差分の大きさ(`git diff 98d38391...HEAD --shortstat`、この run の前): 52 ファイル、+2824/-51。walkthrough を書くかどうかは `/pr` が決める

## Summary

`98d38391...acdac00a` の差分は、release の前の通しの確かめ(分離した herdr で director と機能の org 2 つを動かした実行)の記録で、コードの変更はない。evidence と台帳の写し、手順書 `docs/recipes/isolated-herdr-server.md` と `templates/base/` の同じ写し、tech-debt の RC1〜RC6・RC9〜RC11 の 9 行と RESOLVED 2 行、仕様の Status と印、根拠の表の通しの節、試行の evidence の Correction の段落からなる。

この run が変えたのは、self-review report の Follow-ups の 1 か所(verify の INFO-4)で、足したものは insight event とこの report である。AGENTS.md、README.md、`.claude/rules/ralph/`、`.claude/skills/`、`docs/recipes/`、`docs/quality/`、`docs/architecture/repo-map.md` を所見 RC1〜RC12 に照らして突き合わせたが、この記録が誤りと示した文は、この差分の外にはなかった。所見が skill、雛形、コードの直しを求める箇所は、すべて tech-debt の行が持っており、この PR の範囲外として残した(下の「範囲外、残す」)。

## Changes made

| File | Change |
|------|--------|
| `docs/reports/self-review-2026-10-26-org-recheck.md`(Recommendation の Follow-ups の 2 つ目) | insight event の追記先を `2026-10-26-org-recheck.jsonl` と書いた箇所に、「sync-docs で訂正」の注を足した。`insights-append.sh` は UTC の日付でファイルを選ぶので、self-review の event(2026-10-25T16:50:53Z)は `2026-10-25-org-recheck.jsonl` に入った(1b4d2ae1)。verify の INFO-4 と同じ指摘。指摘の重大度、`Merge:` の行、Findings の表は変えていない |
| `docs/insights/events/2026-10-26-org-recheck.jsonl` | `sync_docs` の event を 1 行足した(`--cycle auto` で cycle 1、`./scripts/insights-append.sh` の rc 0。同じ UTC の日付なので test の event と同じファイルに入った) |
| `docs/reports/sync-docs-2026-10-26-org-recheck.md` | 新規(この report) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md:112`(`docs/recipes/` の行)、`docs/architecture/repo-map.md:37` | 変更なし | 手順書は repo-map が 1 つずつ挙げる recipe のうち「running the org runtime in an isolated headless herdr server」としてすでに載っている。AGENTS.md の行は例を 3 つ挙げる地図の行で、載せ直さない |
| `README.md`(124 行の動詞の表、254〜268 行の org の段落) | 変更なし | この記録は新しい動詞を足さない。`ralph org merge` と `regate`、`merge-order` の説明は、実行で `rebase`、`via=regate` の `gate_passed`、`pr_merged` が出たことと食い違わない。通しの確かめが済んだ、または済んでいないと言う文は README にない |
| `.claude/rules/ralph/agent-messaging.md` | 変更なし | RC1 は、implementer と reviewer の雛形が RESULT を返す手段を座席に教えないことで、規則の「座席は `TO: leader` だけに宛てる」「型付きのメッセージ」とは食い違わない。直しは雛形(コード)の側で、tech-debt の RC1 の行が持つ |
| `.claude/rules/ralph/model-routing.md`(Org runtime model receipts) | 変更なし | 「各 spawn が receipt を 1 行足す」は、断られた spawn が `honored=false` の receipt を残す RC10 と食い違わない(org runtime の仕様 FR-1 も、拒否を receipts に残すと言う)。数え方の問題は tech-debt の RC10 の行 |
| `.claude/rules/ralph/git-commit-strategy.md`(`RALPH_WIP_CHECKPOINT=off`) | 変更なし | F11 の実行(push した 15 commit に `wip:` がない)と合う。herdr の `tab create` に `--env` がない場合の警告の文は、今回の実行では確かめていない(herdr の版は実行中に読み直しておらず、計画の Assumptions の 0.7.5 は `--env` を持つ) |
| `.claude/skills/org/SKILL.md` の前提の節(22 行以降) | 変更なし(範囲外のずれあり) | 67 行の分離した herdr server の指し先は、手順書の見出しがそのままで、挙げる中身(XDG の変数、台帳を分ける repo、git と gh の設定、pane の中の `PATH`、shell の alias、始める前の確かめ、片付け)は残っている。54 行の「機能の worktree では画面が出ない」は今回も合った。49〜65 行のフォルダ信頼の手順は RC7 とずれる(下の 1) |
| `.claude/skills/director/SKILL.md` | 変更なし(範囲外のずれあり) | 536〜539 行(wait が登録の変更から 30 秒ほどで終了コード 5 を返す)は RC9、728〜731 行と 771〜772 行(send のあとに pane で DECISION を確かめる)は RC6 が指す(下の 2、3) |
| `docs/recipes/isolated-herdr-server.md` と `templates/base/` の写し | 同期 | `cmp` が一致(579 行)。冒頭の「通しで打っていない」とする 2 文は消え、「What this recipe does not cover」(562 行以降)は今回の実行が打たなかったことを書く。Clean up の順(detach のあとに `watch --all --stop`)は RC9 の状況を作りうるが、手順書は終了コードを書かないので誤りではない(下の 7) |
| 他の recipe(`docs/recipes/director-wake.md`、`director-launch-cmd.md`、`codex-seat-permissions.md`、`codex-setup.md`) | 変更なし | 「私用の agmsg の写しで座席のメッセージが分かれる」「seat の通知は座席の hook から」とする文は、どれにもない。`director-wake.md` の「Claude Code の director は打っていない」は push の起こし口(`wake_cmd`)のことで、今回は pull(既定)の director だった |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし | 受け入れ条件の「拒否は receipts に `honored=false` 相当で残る」(87 行。FR-1)は RC10 の動きと合う。後の段の動詞は挙げない |
| `docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/org-spec-evidence-2026-10-21.md` | 変更なし | `[ ]` の項目数は仕様の Status(35 行)、根拠の表の通しの節の数(105 行)、項目の表で一致する(27 と 5、判定の列は 26 と 5)。29 行の「3 項目(2026-10-23 の時点)」は日付つきの文で、直さないと決めてある(verify の INFO-1) |
| `docs/evidence/org-trial-2026-10-20.md` | 変更なし | 試行の「Claude Code の通知は座席のもの」は Correction の段落(471〜487 行)が古い文を残したまま訂正している。ほかの文書で同じ主張を繰り返す所はない(`"Claude Code"` と通知を `grep`) |
| `docs/tech-debt/README.md` の RC 行と RESOLVED 行 | 同期 | 9 行は `grep -n '^| RC'` で RC1〜RC6・RC9〜RC11 のみ(363〜371 行)、行の数字は evidence と合う(下の「数の突き合わせ」)。`file:line` で出どころを指す行はない(`grep` で 0 件) |
| `docs/quality/quality-gates.md`、`definition-of-done.md` と template の写し | 変更なし | Envelope validation の行(77 行)の「Spawn rejected, recorded in manifest」は RC2 の 4 回の `rejected` と合う。evidence の `$HOME` を `~` にする決まり(`definition-of-done.md:58`)は、差分 52 ファイルの `/Users/<名前>` の `grep` が 0 件 |

## 範囲外、残す

どれも skill、雛形、コード、または別の PR の記録で、依頼どおり直していない。tech-debt の行が持つものは行を名指す。

1. `.claude/skills/org/SKILL.md:49-65`(フォルダ信頼。`.agents/skills/org/`、`templates/base/.claude/skills/org/`、`templates/base/.agents/skills/org/` の 3 つの写しも同じ 49 行目から): 人が main のチェックアウトで claude を一度起動して信頼すると書く。分離した環境の clone ではこの道で画面が出ず記録も残らず(RC7、理由は分からない)、手順書だけが捨ての pane で答える道に直った。分離しない環境でも同じかは確かめていない。skill は分離した環境では手順書を指すので食い違いではない。verify の INFO-2。tech-debt の行はない(RC7 は手順書で直したので、証拠が足りるまで skill は変えない)
2. `.claude/skills/director/SKILL.md:536-539`(RC9): wait は登録が変わってから 30 秒ほどで終了コード 5 を返すと書き、その間に見張りが止まると 4 になることは書かない。tech-debt の `RC9 (org recheck 2026-10-24)` の行が持つ
3. `.claude/skills/director/SKILL.md:728-731` と `:771-772`(RC6): send のあとに leader の pane で DECISION が送られた入力として出ていることを確かめる手順は、leader がすぐ作業を始めて入力が画面の外へ流れると打てない。tech-debt の `RC6 (org recheck 2026-10-24)` の行が持つ
4. `.claude/skills/org/SKILL.md:529`(RC4): 分割計画の例の「予約の外のファイルは編集しない」は、機能の計画と report を予約の外に書かせる leader の雛形の手順と、文字どおりには食い違う。tech-debt の `RC4 (org recheck 2026-10-24)` の行が持つ
5. `.claude/skills/pr/SKILL.md:31` と `internal/org/prompts/leader.md`(RC5): PR の言語の規則は skill の手順にあり、雛形が指す `template.md` にない(雛形だけに従う leader は落とす、というのは evidence の推論)。tech-debt の `RC5 (org recheck 2026-10-24)` の行が持つ
6. `internal/org/prompts/leader.md`(`--scope` を書かない手順 5、RC2)、`implementer.md`・`reviewer.md`(RESULT を返す手段がない、RC1)、見張りの `scope_change`(RC3)、`reject` の receipt(RC10)、watcher の `claude -p` が利用者の hook を読むこと(RC11): コードと雛形の直しで、tech-debt の各 RC の行が持つ。`.claude/skills/org/SKILL.md:305` の「範囲の外への変更は scope_change ALERT で知らせる」は誤りではない。機能の org では予約の中の編集でも出ることを、RC3 の行が持つ
7. `docs/recipes/isolated-herdr-server.md` の Clean up(detach のあとに見張りを止める順): promoted の director の wait が 5 でなく 4 で終わりうること(RC9)は手順書に書いていない。誤った文はなく、tech-debt の RC9 の行が持つので、手順書は変えていない。手順書は scaffold 先にも届くので、2 つの写しを同じに保つ必要が出る編集は、RC9 を直す PR で足りる

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の LOW-1(根拠の表の RC11 の行の「1 回ごとに通知」) | director が verify のあと inline で直した(「watcher の多くの回(今回は 13 回のうち 12 回)」)。この run で `grep` して、同じ言い切り(`each watcher run`、「1 回ごとに通知」)が手順書、evidence、tech-debt、仕様、根拠の表に残っていないことを確かめた |
| verify の INFO-1(仕様 29 行の「3 項目(2026-10-23 の時点)」) | 直さない(依頼の決定)。今の数は 35 行の段落にある |
| verify の INFO-2(`/org` skill のフォルダ信頼) | 上の 1。範囲外 |
| verify の INFO-3(計画の Progress と AC の印) | director が直し済み。計画は AC1〜AC9 と Review・Verification・Test の 3 つの artifact が `[x]` で、残る `[ ]` は PR created だけ(この run は計画を触れていない) |
| verify の INFO-4(self-review の Follow-ups が書く event の追記先) | 直した(上の Changes made)。ほかの task の report にも同じ形の予測の文が残りうるが、この run は触れていない |
| test の Test gaps 1〜4 | 文書には書かない。1(文書を読む Go のテストがない)と 2(手順書を打つテストがない)は、evidence の実機の実行と /verify の 29 ブロックの打ち直しが根拠で、計画の Test plan が「新しいテストはない」と書く。3(`tests/test-check-sync.sh` は scratch の木で規則を見る)は `check-sync.sh` が本物の 2 つの写しを比べる。4(coverage は計っていない)は、コードの変更がないので差分で動かない |

## 数の突き合わせ

tech-debt、evidence、仕様、根拠の表で同じ事実を言う数を、台帳の写し(`docs/evidence/org-recheck-2026-10-24/`)から数え直した。食い違いは見つからなかった。

| 数 | 写しからの値 | 同じ数を言う場所 |
|----|--------------|------------------|
| 見張りの escalation | `watch.log` の `WATCHDOG ESCALATION` が 8 行、受信箱の ALERT の件が e3〜e10 の 8 件 | evidence の F6 の行、tech-debt の RC3 の行(12 件のうち 8 件)、根拠の表の RC3 |
| 見張りから leader への ALERT の失敗 | `failed to ALERT` が 0 行。送った ALERT は 29(12 と 17) | tech-debt の「The watch's ALERTs to the leaders failed」の行、evidence の開いたまま残る行、根拠の表の末尾の段落 |
| watcher | `role=watchdog` の receipt が 13、`watcher error` が 0 行、`watcher already in flight` が 11 行 | tech-debt の「The on-demand watcher」の行、evidence の RC11 |
| 通知の shim | 12 行、すべて題が `Claude Code` | 手順書(326 行)、tech-debt の RC11、根拠の表の RC11、evidence の RC11 |
| 受信箱の件 | 36 行(件ごとに 3 行)で 12 件 | evidence の RC3、tech-debt の RC3 |
| 手順書が引く evidence | `templates/base/docs/recipes/` の 5 つの recipe で、異なりのファイルが 6・2・1・1・1(合わせて 11)。手順書は `org-recheck-2026-10-24.md` を 7 回引く | tech-debt の「The recipes under `templates/base/docs/recipes/`」の行(「cited four times, and seven after the fixes」) |
| 仕様の `[ ]` | 5 項目(FR-3、FR-4、FR-10、ツール非依存、受け入れ条件 5) | 仕様の Status(35 行)、根拠の表の 105 行、項目ごとの印 |

## Plan drift notes

- 計画の Progress は「PR created」を除いて `[x]`。この run が計画を触れていないので digest は承認のまま(`1e218f3d7907`)
- 計画の Affected areas が挙げないものが、この PR にいくつか入っている。`docs/evidence/org-trial-2026-10-20.md` の Correction の段落(Progress の S6 の項が書く)、`docs/tech-debt/README.md` の isolated-herdr-server の RESOLVED の行(319〜320 行)への「Added 2026-10-25」の書き足し、同じ 362 行の recipe の evidence 引用の行への書き足し(self-review の F4 を直した結果で、Progress には書いていない)。どれも Non-goals には入らず、記録を最新にする足しで、tech-debt の行と RESOLVED の行の範囲の中
- 計画の AC8 は watch の停止の直後に `pgrep -fl /tmp/ralph-iso` を求めるが、evidence は 8 段目(`rm -rf` のあと)にしか記録がない(verify の AC8 の行)。計画は変えていない

## Left open

- 開いたまま残る行は、仕様と根拠の表の「leader への ALERT」、Codex の leader、watcher の時間切れ、FR-10、N4 と、この PR が足した RC1〜RC6・RC9〜RC11 の 9 行。RC2(leader の雛形に `--scope` がない)は、雛形のとおりに spawn を打つ機能の org で毎回起きるので、release の前に直すかどうかはユーザーが判断する(FR-4 と受け入れ条件 5 を `[ ]` にした理由)。RC1(座席が私用の agmsg を使わない)は、既定と違う agmsg の home のときだけ起きる(おそらく。確かめていない)
- 通しで確かめていないこと(evidence の Not verified): 停止と再開、codex の座席と director、headless の director、leader が Monitor で待ったか(F7)、due な通知の記録(F13)、Go の verifier の既定の timeout(F10)、実行中の herdr と Claude Code の版、利用者の hook の中身(RC11)

## Verification

| Check | Result |
|-------|--------|
| `cmp docs/recipes/isolated-herdr-server.md templates/base/docs/recipes/isolated-herdr-server.md` | rc 0(同じバイト列) |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-pipeline-sync.sh`、`./scripts/check-template-purity.sh` | どちらも ok(pipeline の順は `definition-of-done.md`、`README.md`、`AGENTS.md` に揃っている。template に meta-repo 固有の参照なし) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-24-org-recheck.md` | `1e218f3d7907`(承認の digest と同じ) |
| `git diff --check 98d38391...HEAD` | 出力なし、rc 0 |
| 差分 52 ファイルの `/Users/<名前>`、pane id の形、UUID、GitHub の token の形の `grep`、U+FFFD の数 | 0 件。token の接頭辞の語が verify report の 71 行目に 1 件あるが、verify が grep に使った語の一覧で、token の値ではない |
| テスト | 打っていない(`/test` の範囲。この run が変えたのは report と insight event だけで、コードとスクリプトには触れていない) |

## Insight event

追記した。`./scripts/insights-append.sh --slug org-recheck --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` が rc 0 で、`docs/insights/events/2026-10-26-org-recheck.jsonl` の末尾の行(cycle 1)になった。UTC の日付が 2026-10-26 だったので、test の event と同じファイルに入り、`??` の新しいファイルは増えなかった。
