# sync-docs report: reviewer-runs-gate

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-26-reviewer-runs-gate.md`(承認の digest `6dd89c5dd6d0` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run は計画の本文を触れず、Progress に sub-bullet を 1 つ足しただけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `633cd7c8`(origin/main)、branch は `feat/reviewer-runs-gate`。run 開始時の HEAD は `d3309d33`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-26-reviewer-runs-gate.md`(3 回、Merge OK。LOW の F11 と INFO の N2 が開いたまま)、`docs/reports/verify-2026-10-26-reviewer-runs-gate.md`(pass。LOW の V1 と情報の I1)、`docs/reports/test-2026-10-26-reviewer-runs-gate.md`(pass。Go のテストを go1.27.3 と go1.25.8 で、shell の 44 本を rc 0 で。mutant 10 個がすべて落ちた。Test gaps 5 件)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、この run の前): 40 ファイル、+2307/-582。walkthrough を書くかどうかは `/pr` が決める

## Summary

この PR が変えた振る舞いは 3 つある。機能の org の `ralph org gate` を打つ座席が leader から reviewer になった(leader は report と計画の archive のコミットと secret scan を済ませたあと、TASK_ID `gate-<n>` の TASK を reviewer に送る)。機能の org の codex の reviewer にも、台帳が cwd の外にあれば `--add-dir <台帳>` が付く(2026-10-07 の「leader だけ」を変えた)。見張りの数え方は変えず、コメント、help、regate の断りの文が新しい打ち手を言う。

この振る舞いについて、古い形のまま残った記述を AGENTS.md、CLAUDE.md、README、`.claude/rules/ralph/`、`docs/recipes/`、`docs/quality/`、`docs/architecture/`、`docs/specs/2026-08-01-org-runtime.md`、template の写しで探したが、直すものは見つからなかった(下の「Surfaces checked for drift」)。diff の中の文書(`/org` と `/director` skill の 4 写し、recipe、README、quality-gates、仕様、根拠の表、`git-commit-strategy.md`)は verify が同期を確かめている。

この run が直したのは `docs/tech-debt/README.md` と計画の Progress の 2 つ。tech-debt では次の 4 つを済ませた。

1. verify の V1 が挙げた 10 行に、このブランチの持ち越しの注と Related の計画を足した。
2. そのうち 1 行(org-gate cycle 2 の小さな所見の (e))は、この PR の S2 が直していたので取り消し線にした。
3. V1 の外で Trigger が当たっていた 3 行にも同じ注を足した。
4. 自己レビューの「Tech debt identified」の 2 件を新しい行にした。

払った行は (e) の 1 項目だけで、RESOLVED にした行はない。

## Changes made

行番号は verify の報告と同じ(この run の前)。331 行目に行を 1 つ入れたので、357 行目以降は今 +1 になっている。各行は 5 欄のまま。

| Row(名前と行) | Change |
|------|--------|
| org runtime FR-7 の quality gate は prompt-instructed(157。V1) | Debt の「最後の gate は `ralph org gate` が走らせて記録する」の文に、reviewer 座席が leader の TASK で打つこと(plan `2026-10-26-reviewer-runs-gate`)を足し、「reviewer 座席の checks はまだ template 駆動」を「レビューの初めの checks は」に直した。Trigger に持ち越しの注(`ralph org gate` の検査と記録の形は計画の Non-goals で、hook も verb も足さないので、最後の gate の実行は今も prompt 頼み。ralph が確かめるのは記録だけ)。Related に計画 |
| org-state-dir-common の test gap(171。V1) | Trigger に注。`codexWritableRootArgs` に `featureOrg` と `writesLedger` が入り、機能の org の reviewer の表のテストが付いたので当たった。(a) は `LegacyWorktreeStateDir` で、触れていない。(b) の `rel != ".."` の事例は、新しい表も持たない(state dir が cwd の下か外かだけで、直接の親ではない)。この句は機能の org の reviewer の `--add-dir` も決めるようになった。テストの追加はコードの変更で `/self-review` から戻るので払わなかった。Related に計画 |
| org-stop-all の文書の所見(177。V1) | Trigger に注。コマンド表の `gate` の行(2,036 → 2,080 文字)と `regate` の行(4,054 → 4,072 文字)に 1 句ずつ足したので当たった。(a) が名指す `stop` と `disband` の cell は変わらない。`gate` と `regate` の行は今その 2 つより長いので、再構成はこの 2 行も範囲に入れる、と書いた。Related に計画 |
| org-feature-worktree の小さな所見(197。V1) | Trigger に 2 つ目の注。`leader.md` の機能ごとの org の節(ミッションの 5〜7、節の頭、手順 5・9・12・14、受信箱の節、運用規律)を直したので、(g) と済んだ (f)・(j) の Trigger に当たったが、(g) は f1b0352e、(f)・(j) は b0fc4c59 で直っていて払うものがなかった。足したピンは `containsPhrase` で読む。(a)〜(e)、(h)、(i)、(k)、(l) は計画の Scope の外。Related に計画 |
| org-gate cycle 2 の小さな所見(234。V1) | (e) を `~~...~~` で消し、「Closed in feat/reviewer-runs-gate (commit 5e8f5b50)」を足した。S2 が `leaderActivityEventCount` のコメント (c) を reviewer の gate に合わせて全部組み直したので、短い 1 行は残っていない(最後の行だけが短い)。Impact の (e) にも `~~...~~ (closed)`。Trigger の注は、`gate.go` の冒頭のコメントを直しただけで、`gate.go`、`gate_run.go`、`lockfile.go` のコードは変えていないこと、`org_gate_test.go` の変更は help の pin(`TestOrgGateHelp`)で (i) の interrupt のテストではないこと、(a)〜(d) と (f)〜(i) は触れていないこと。Related に計画 |
| org-director-attach の test gap(257。V1) | Trigger に注。`/director` skill の受信箱の節(4 写し)の 2 か所(leader が gate を頼んでいる間の reviewer の停滞の ALERT の文と、前面の待ちの例から gate を外したこと)を直したので当たったが、wait の段落、`WaitInbox`、`inboxWaitTimeoutPhrase`、skill のテストには触れていない。(1)〜(3) は開いたまま。Related に計画 |
| ファイルの大きさの行(357。V1) | Debt の行数を `wc -l` で数え直した。`prompts_test.go` 2165 → 2670、`spawn.go` 2674 → 2680、`spawn_test.go` 5904 → 5991(`verbs.go` 2277 と `verbs_test.go` 5854 は同じ)。#253 の時点の数は括弧に残し、「The PR added」を「#253 added」に直した。Trigger に注(`prompts_test.go` は 2189 から 2670 で +481、+300 の閾値を越えた。分割は Scope の外。数え直したので +300 は新しい数から数える)。Related に計画 |
| wip: の行(359。V1) | Trigger に注。`gate.go` の冒頭のコメントを直したので当たったが、計画の Non-goals が gate の検査と記録の形を変えないこと、`wip:` の検査を gate に入れると `gate_passed` の意味が変わることを書いた。手順 12 と 14 が早い止めを持つ。Related に計画 |
| RC1(363。V1) | Trigger に注。`reviewer.md`(gate の TASK の節、ミッション、前文、レポート契約)を直したので当たったが、計画の Scope(S3)は agmsg の path を足さない。新しい節の RESULT と BLOCKED の例も形を示すだけなので、`gate-<n>` への返事も同じ欠けに頼る(私的な agmsg の copy では動かしておらず、prompt からの推測)。Related に計画 |
| RC4(367。V1) | Trigger に 2 つ目の注。`leader.md` の機能ごとの org の手順(ミッションの 5〜7、手順 5・9・12・14)を直したが、RC4 は計画の Scope の外で、手順 4 にも `/org` skill の分割計画の例にも触れていない。Related に計画 |
| template の quality-gates の CI の配線の行(118。V1 の外) | Trigger の「次のタッチ」が当たった(template の `quality-gates.md` の Quality pipeline gate の行の 1 文を直した)。「Must pass in CI」の節と template の verify チェーンには触れていないので、持ち越しの注を足した。行は日本語だが注は英語 |
| regate の大きさの行(286。V1 の外) | Trigger の「The next change to `regate_test.go`」が当たった(`TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns` の断りの文とコメントで 7 行、ファイルの長さは同じ)。分割は Scope の外として注を足した |
| recipe の `docs/evidence/` の引用の行(362。V1 の外) | Trigger の「The next edit of any of these five recipes」が当たった(`codex-seat-permissions.md` の 2 写し)。引用は足さず、2 つの既存の引用も直していないので、org recheck の注と同じ形で「left it as it was」と書いた。行は開いたまま |
| 新しい行: Codex の reviewer と `ralph org send`(331。self-review の Tech debt の 1 つ目) | Codex の leader の行(330)の次に足した。Codex には `run_in_background` がなく、reviewer の gate の TASK は前面で数十分かかるので、そのあいだ `Send` は何も打てずに時間切れになる。Claude Code の reviewer は影響を受けない。leader の雛形と `/director` skill の扱い(停滞の ALERT では stop も spawn し直しもせず `ralph org read` で見る)も書いた。Trigger は 330 の直しと同じ変更で、実機の最初の Codex の reviewer の gate でも当たる |
| 新しい行: 手順 14 から手順 12 に戻る道で implementer がいない(375 行目、表の最後。self-review の F11) | 手順 14 が reviewer だけを spawn し直して手順 12 に戻ると、`GATE: fail` か implementer の手が要る断りで implementer への TASK が手順 13 で止めた座席に当たる。implementer の spawn し直し(`--role implementer`、per-plan の承認、手順 10 の archive の revert)の手がテンプレートにも skill にもない。まれな道で、誤った push や記録は起きない。トライアルの F11(session-end の WIP の行)と取り違えないよう、表の最後に置き、本文に「not the trial's F11」と書いた |
| `docs/plans/active/2026-10-26-reviewer-runs-gate.md`(Progress) | 「Implementation started」の sub-bullet に、この run の判断を 1 つ足した(本文は触れていない。digest は同じ) |
| `docs/reports/sync-docs-2026-10-26-reviewer-runs-gate.md` | 新規(この report) |

新しい注と行はすべて英語(このファイルの言葉)で、`|` を含まない。計画のパスは `/pr` の `archive-plan.sh` が `docs/plans/archive/` に書き換える形(`docs/plans/active/<name>` のあとが空白、`(`、`|`)で書いた。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md`(repo map の `internal/org/` の gate の説明) | 変更なし | `ralph org gate` が rebase 済みの head で 2 つのスクリプトを走らせ、`gate_passed`/`gate_failed` を記録する、とだけ書き、打つ座席を書かない |
| `CLAUDE.md` | 変更なし | gate も `--add-dir` も書かない |
| `README.md`(252 行、264 行) | 変更なし(diff 内で同期済み) | 264 行は reviewer が `gate-<n>` の TASK で打つこと、leader が出力のファイルと `git rev-parse HEAD` を照らすことを書く。252 行は打つ座席を書かない |
| `.claude/rules/ralph/`(`git-commit-strategy.md` ほか)と `templates/base/.claude/rules/ralph/` | 同期 | `git-commit-strategy.md` の 56 行は「after its reviewer's gate passes and again before it pushes」で、雛形の手順 12 と 14 に合う(diff 内、template の写しも同じ)。ほかの規則は `org gate` も `add-dir` も書かない |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし | 18 行の (b) は「reviewer が最初に決定論ゲートを再実行する」と書く(2026-10-04 の改訂の決定)。レビューの初めの素の実行は D2 で残したので今も当たる。`ralph org gate` の打ち手は書かない |
| `docs/recipes/`(`codex-seat-permissions.md`、`isolated-herdr-server.md`、`director-wake.md`、`director-launch-cmd.md`、`codex-setup.md`)と template の写し | 同期 | `codex-seat-permissions.md` の「The org ledger and `--add-dir`」(81 行からの節)は diff で直り、2 写しは同一。`isolated-herdr-server.md` の 166 行の「`GOFLAGS=-timeout=30m` for the gate of a Go repository」は打つ座席を書かない。残りは gate も `add-dir` も書かない |
| `docs/quality/quality-gates.md`(82 行)と template の写し(81 行) | 同期 | Quality pipeline gate の行が reviewer の `GATE: fail`/`GATE: unrunnable` と leader の裁定を書く(diff 内)。root と template の差は前からの KNOWN_DIFF の「(see docs/tech-debt)」だけ |
| `docs/architecture/`、`docs/quality/definition-of-done.md`、`.codex/`、`.claude/agents/`、`internal/org/prompts/director.md`、`implementer.md` | 変更なし | `ralph org gate` を leader が打つと言う文はない(`git grep -i -E` で leader と gate が同じ文にある行を探した。`internal/org/watch.go` の「when the leader ran the gate itself」は新しいコメントが旧版を言う文) |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 と根拠の表 | 同期 | 仕様の `- [ ]` は 6、`- [x]` は 26(FR-12 は表にない)。表の判定は 25 と 6。verify が確かめた数と同じで、この run は数え直しだけをした |
| `README.md` の「It needs no running watch and no approved plan」 | 変更なし(下の「見送り」の 2) | `ralph org gate` という verb の話で今も正しい |
| tech-debt の Trigger の列 | 更新 | 上の Changes made。変えたファイルの名前を Trigger が名指す行を `git diff --name-only` の一覧と照らして探した。表にある 13 行を直した(V1 の 10 行と、その外の 3 行) |

## 見送り

1. 計画の Affected areas に `internal/org/approval.go`、`internal/org/regate_merge.go`、`internal/cli/org_regate.go`、`internal/cli/org_gate_test.go` がない(verify の I1)。承認済みの計画の本文なので digest が変わるため、本文は触れていない。Progress の sub-bullet に「`regate_merge.go` と `org_regate.go` の help とコメント、`org_gate_test.go` の help の pin も S2 の続きで直した」を足した(`approval.go` は逸れ 2 の続きにすでにある)。
2. README の「It needs no running watch and no approved plan」は `ralph org gate` という verb の話で正しいまま直さない。ただし leader が reviewer に送る `gate-<n>` の TASK は TASK の送信なので、見張りが動いていないと断られる(`.claude/rules/ralph/agent-messaging.md` と `watchgate.go` の規則。以前は leader が gate を直接打ったので見張りなしで打てた)。`ralph org start --plan` が見張りを起こすので、普通は起きない。断られたら leader は、一般の裁定(ミッションの 5)か `ralph org escalate` の BLOCKED で受けることになる(手順 12 の枝は gate の出力のエラーを扱い、send の断りは扱わない)。実機で確かめていない。
3. tech-debt の 257 行目の「each copy of the skill holds the phrase once」は、`/director` skill の各写しが `no open inbox item arrived within` を 2 回持つ(`.claude/skills/director/SKILL.md` の受信箱の節の中に 2 か所。origin/main でも同じ)ことと合わない。この PR の diff の外で、2 か所とも受信箱の節の中にあるので「ほかの場所に出ても緑のまま」という (2) の穴の説明は変わらない。直していない。
4. 実機の確かめ(reviewer が herdr の上で gate の TASK を受けて `ralph org gate` を打つこと、codex の reviewer が `--add-dir` で台帳に書けること)。計画の R6・D8 のとおり、FR-7 の `[ ]` が残る。

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の V1(tech-debt の 10 行) | 上のとおり。10 行すべてに持ち越しの注を足した。払ったのは 234 行目の (e) 1 項目で、この PR の S2 が先に直していた。行は RESOLVED にしていない |
| verify の I1(計画の印と影響範囲) | 印は PR の前に付けてある(AC1〜AC8 と artifact の 3 行が `[x]`)。残る `[ ]` は「PR created」だけで、`/pr` が付ける。影響範囲の抜けは上の「見送り」の 1 |
| self-review の Tech debt 1 つ目(Codex の reviewer の send) | 331 行目に足した |
| self-review の Tech debt 2 つ目(F11) | 375 行目に足した。F11 を直さずに進める場合の行として、self-review が文面を用意していた |
| self-review の INFO N2(`watch.go` のコメント (c) の長さ) | 行にしない。self-review も Tech debt に入れていない。ただし org-gate の (e)(同じコメントの短い行)は、この PR が段落を全部組み直したので直った |
| test の Test gaps 1〜5 | 文書には足さない。1 は計画の R6・D8 で FR-7 を `[ ]` にした理由、2〜5 はこの run の実行範囲の注で、tech-debt の行にする所見ではない |
| V1 の外の 3 行(118、286、362) | verify の掃引は名前の一致だけで、この 3 行は Trigger の文が「次のタッチ」「次の変更」「次の編集」で、この diff が当たっていた。足した注は 1 〜 2 文で、不要ならこの 3 行の注だけを外せる |

## 数の突き合わせ

| 数 | 値 | 同じ数を言う場所 |
|----|----|------------------|
| 仕様の `- [ ]` と `- [x]` | HEAD 6 と 26(base は 5 と 27) | 仕様の Status(「`[ ]` を 6 項目」)、根拠の表の判定の数(25 と 6。FR-12 は表にない)。verify が base と HEAD で数えた値と同じ |
| `prompts_test.go`、`spawn.go`、`spawn_test.go` の行数 | 2670、2680、5991(`wc -l`)。`verbs.go` 2277、`verbs_test.go` 5854 は変わらない | tech-debt のファイルの大きさの行の Debt と Trigger の注。HEAD `d3309d33` で数えた(コードとテストを変えた最後のコミットは `d3c8e337`で、それ以降は文書だけ) |
| 持ち越しの注を足した行 | 13 行(V1 の 10 行と、その外の 3 行)。新しい行 2 つ。取り消し線にした項目は (e) の 1 つ | `docs/tech-debt/README.md` の行数は 2 増えた(新しい行が 2)。変わった行は 15(元の行が 13、新しい行が 2) |
| 計画の digest | `6dd89c5dd6d0`(run の前後で同じ) | 計画の `- Approved:` の行 |
| leader の雛形の差分の範囲 | ミッションの 5〜7、節の頭、手順 5・9・12・14、受信箱の節、運用規律(`git diff -U0` の hunk の位置を雛形の見出しに対応させた。手順 4 は触れていない) | 197 行と 367 行の注 |

## Plan drift notes

- 計画の Progress は「PR created」を除いて `[x]`。この run は sub-bullet を 1 つ足しただけで、digest は承認のまま。Progress の逸れ 1・2 と「sync-docs で判断すること」は、上の Changes made と Decisions で閉じた。
- この run が足した tech-debt の注は、計画の Affected areas(「`docs/tech-debt/README.md` の Codex の leader の gate の行を直す」)より広い。持ち越しの注と新しい 2 行だけで、コードも雛形も触れていない。

## Left open

- 実機の確かめ(FR-7 の `[ ]`)。release のノートに「機能の org の gate は reviewer が leader の TASK で打つ。codex の reviewer にも機能の org では `--add-dir` が付く」と書く(計画の Rollout)。
- 開いたままの tech-debt: 自己レビューの F11(375 行目)、Codex の reviewer の send(331 行目)、V1 の 10 行と V1 の外の 3 行の持ち越し。
- 後工程: `/cross-review`、`/pr`(PR の本文の初めに、2026-10-07 の「`--add-dir` は leader だけ」を変えたことと、reviewer が打つ gate を実機で動かしていないことを書く。self-review の Follow-ups)。

## Verification

| Check | Result |
|-------|--------|
| `GOTOOLCHAIN=go1.25.8 ./scripts/run-static-verify.sh`(tech-debt の行と計画の Progress を直したあと、この report を足す前。16:00:02〜16:00:20 UTC) | rc 0、終わりの行は `==> All verifiers passed.`。`tech-debt README plan references` は `OK`(足した `docs/plans/active/2026-10-26-reviewer-runs-gate.md` は存在する)、`check-skill-sync.sh` は「14 skill(s) in lock-step」、Go は `gofmt: ok`、golangci-lint は `0 issues.`、`secret-scan-branch: scanned 633cd7c8..d3309d33 against origin/main: clean`。出力は scratchpad に書き、証拠は `docs/evidence/verify-2026-10-26-160003.log`(git が無視する場所)。`GOTOOLCHAIN=go1.25.8` は環境の都合(手元の go 1.27.3 は golangci-lint の typecheck を壊す)で、変更の問題ではない |
| `./scripts/check-sync.sh`(tech-debt と計画を直したあと) | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-26-reviewer-runs-gate.md` | `6dd89c5dd6d0`(承認の digest と同じ) |
| tech-debt の変わった 15 行(`git show HEAD:docs/tech-debt/README.md` との照らし合わせ) | 5 欄のまま(`(?<!\\)\|` で分けて 7 片)、各欄のバッククォートと `~~` は偶数個、行末は `|`。元の行で変わったのは 13 行で、新しい行は 2 つ。追加した文に `file:line` の参照はなく、U+FFFD は 0 個 |
| `git diff --check`(作業ツリー) | 出力なし、rc 0 |
| テスト | この run は Go のコードもスクリプトも触れていない(変えたのは tech-debt の文、計画の Progress、report)。Go のテストは `/test` が HEAD ad1a5c64 で 2 版を通した |

## Insight event

追記していない(リードの指示)。足すときは `./scripts/insights-append.sh --slug reviewer-runs-gate --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`。`docs/insights/events/2026-10-26-reviewer-runs-gate.jsonl` には self_review、verify、test の 3 行がすでにある。
