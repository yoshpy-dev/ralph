# sync-docs report: busy-leader

- Date: 2026-10-23
- Plan: `docs/plans/active/2026-10-23-busy-leader.md`(承認の digest `f046c0eac2ff` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run は計画を触れていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `a58f7c64`、branch は `fix/busy-leader`。run 開始時の HEAD は `cc03ff0e`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-23-busy-leader.md`(Merge OK、LOW 1 件)、`docs/reports/verify-2026-10-23-busy-leader.md`(pass。LOW 2 件と INFO 2 件)、`docs/reports/test-2026-10-23-busy-leader.md`(pass。穴 1 件は PR の Known gaps に書く)
- 差分の大きさ(`git diff a58f7c64...HEAD --shortstat`、この run の前): 30 ファイル、+2184/-207。walkthrough を書くかどうかは `/pr` が決める

## Summary

`a58f7c64...cc03ff0e` の差分は、Claude Code の leader が座席を待つあいだ agmsg の受信箱を Monitor で見張って手番を終えるようにし(`internal/org/prompts/leader.md` の `## agmsg の受信箱の運用`、`{{AGMSG_SCRIPTS}}`)、`ralph org send` の idle/done の待ちが切れたときのエラーに理由と打ち直し方を足し(`sendWaitDeadlineHint`)、`/director` と `/org` の skill に送り方(25 分の send を `run_in_background` で、終了コード 0 のあとに `ralph org read` で pane を読む)を書いた。文書の大半は、この branch の S4・S5 がすでに合わせていた。

この run は verify が挙げた 3 件(LOW-1、LOW-2、INFO-1)と self-review の観察 1 件を直し、そのほかの面をコードと skill に突き合わせた。直した文は、DECISION の send が終了コード 0 で終われば resolve してよいという 4 か所(README の 2 か所と quality-gates の 2 つの写し)、古いテストのファイル名 1 か所、実機の記録の 1 文、tech-debt の RESOLVED の注のテスト名 1 つである。

## Changes made

| File | Change |
|------|--------|
| `README.md:254` | 「resolve it only once that send exited 0」を「exited 0 and `ralph org read --seat leader` shows the `DECISION` was submitted, not left in the leader's input box (the `/director` skill gives the order)」にし、失敗の例に「leaving the `DECISION` in the input box」を足した。続けて 3 文足した。send は leader の手番が終わる(herdr が `idle` か `done` と報告する)まで打たず、手番の途中の leader(座席を前面で待つ leader など)への send は `--timeout-ms`(既定 30 秒)まで待ってから何も打たずに失敗する。Claude Code の director は `--timeout-ms 1500000` を付けて `run_in_background` で送る。leader の雛形は、Claude Code の leader に Monitor で待って手番を終えるよう書き、Codex の leader は Monitor がないので待つあいだ届かない。足した値と文は `/director` skill の「答える順」の 2、`internal/cli/org.go` の `send` の help、`sendWaitDeadlineHint` と合わせた |
| `README.md:262` | approve が出す resolve の打ち方の条件を「once that send exited 0」から「once that send exited 0 and `ralph org read` shows the `DECISION` was submitted, not left in the leader's input box」にした。後ろの「so an approval whose `DECISION` was not sent stays under the inbox deadlines」はそのまま |
| `docs/quality/quality-gates.md:81`、`templates/base/docs/quality/quality-gates.md:80` | 「Feature plan approval」の行の同じ条件を、README:262 と同じ語で直した。2 つの写しは同じ文字列の Edit で、`check-sync.sh` が一致を確かめた |
| `docs/evidence/org-spec-evidence-2026-10-21.md:117` | 「(`internal/org/prompts_test.go` のテストだけ)」を「(`internal/org/prompts_leader_inbox_test.go` の `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn` だけ)」にした。`description` の文を固めるテストは S6 の 1183b08e でこのファイルに移っている |
| `docs/evidence/busy-leader-2026-10-23.md:181` | 「雛形は `description` に触れていない」に括弧を足した。この確かめの雛形(3d1946d6)の話で、そのあとの 02e27384 が雛形に `description` の文を足したこと、その雛形では実機を打ち直していないことを書く。確かめた事実の記録は変えていない |
| `docs/tech-debt/README.md:326` | F7 の RESOLVED の注の「Tests:」に `TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone`(`internal/org/prompts_leader_inbox_test.go`)を足した。注の「ends ... within about 10 s of the leader's claude being gone」の句を固めるテストである。名前は `func` の定義で 1 件だけ見つかる(`prompts_leader_inbox_test.go:322`) |
| `docs/insights/events/2026-10-23-busy-leader.jsonl` | `sync_docs` の event を 1 行足した(`--cycle auto` で cycle 1)。`./scripts/insights-append.sh` の rc 0 |
| `docs/reports/sync-docs-2026-10-23-busy-leader.md` | 新規(この report) |

足した文にホームのパス、token はない。`git diff --check` は rc 0、差分の U+FFFD は 0 件。skill は触れていない(skill の写しの同期は要らなかった)。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md`(124 行の動詞表、258 行の director の節) | 変更なし | 124 行の approve の説明の「once that send succeeded」は approve の出力(`internal/org/approve.go:314`、`internal/cli/org_approve.go:130`)の文言をそのまま写す。verify の INFO-2 と同じ判断で残した。258 行の「send-then-resolve order above」は、直した 254 行の文を指す。動詞表の `send` の行は idle/done の待ちを書かない(`org send --help` に任せる) |
| `AGENTS.md` の repo map(`internal/org/` の行) | 変更なし | 「role prompt templates (go:embed)」の 1 句は今も正しい。`{{AGMSG_SCRIPTS}}` やファイル名は、ファイルの境界が変わっていないので足さない。この変更が足したのは `prompts_leader_inbox_test.go` と `org_send_help_test.go` のテストだけで、新しい package や公開のファイルはない |
| `.claude/rules/ralph/agent-messaging.md` と template の写し | 変更なし | `cmp` で一致。send の idle/done の待ちも `--timeout-ms` も書かず、書くのは TYPE の表、メッセージの形、サイズの上限、TASK の検査、受信箱への escalation で、この変更に触れない |
| `docs/specs/2026-08-01-org-runtime.md`(FR-3) | 変更なし | 「追加指示は `herdr agent wait --until idle` を挟んだ `send` 動詞経由とする」は今も正しい(待ちは idle か done。`Send` の comment が、idle だけを待つと受け入れる座席に対しても時間切れになったと書く)。この変更は待ちを変えず、待ちが切れたときのエラーの文だけを変えた |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 変更なし | FR-5 の 2026-10-23 の注(92 行)と、ツール非依存の注(151 行)が Claude Code の leader だけが直ったこと、Codex の leader は届かないことを書く。verify の AC6 が印の数(`[x]` 29、`[ ]` 3)と PR の一覧(35)を確かめている。126 行(8a-1 で決めた順序)は当時の記録で、同じ仕様の FR-5 の注が新しい順序を書く |
| `docs/evidence/org-spec-evidence-2026-10-21.md` | 直した(上の表) | 117 行以外の F7 の記述(確かめたもの・確かめていないもの)は `docs/evidence/busy-leader-2026-10-23.md` の (a)〜(f) と合う |
| `docs/evidence/busy-leader-2026-10-23.md` | 直した(上の表) | 17 行目が「3d1946d6 の雛形で動かした」と書き、181 行目の括弧がそのあとの 02e27384 を書く |
| `docs/tech-debt/README.md` | 直した(上の表) | F7 の RESOLVED の注が挙げるテスト 6 つ(足した 1 つを含む)は、どれも挙げたファイルに `func` の定義が 1 件ずつある。開いている F7 の行(Codex の leader に限る)は変えていない。ファイルの大きさの行は verify が `wc -l` で確かめた値のままで、この run は Go のファイルを変えていない |
| `docs/recipes/`(`codex-seat-permissions.md` ほか) | 変更なし | `codex-seat-permissions.md` の `send` の段落(229〜245 行)は、Enter の待ち 750 ms と submit の確かめを書く。待ちが切れたエラーは「何も打たない失敗」の類に入り、記述と食い違わない。この recipe が probe に使うのは reviewer の座席で、leader の雛形の使命は変えていない(project memory の ripple の確認)。`director-wake.md` と `director-launch-cmd.md` は send の待ちに触れない |
| `docs/quality/quality-gates.md` と template の写し | 直した(上の表) | 「Feature plan approval」の行だけ。ほかの行に send の待ちや Monitor の語はない。KNOWN_DIFF の行は触れていない |
| `docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md`、`.codex/README.md`、`.ralph/core/AGENTS.core.md` | 変更なし | send の待ちも leader の雛形も書かない(grep で確かめた) |
| `/org`・`/director` skill の 4 写しずつ | 変更なし | S4 が直した。`./scripts/check-skill-sync.sh` は 14 skill で lock-step。director の「答える順」の 2〜3 を今の README、quality-gates の文と突き合わせた |
| `internal/org/prompts/leader.md`、`internal/cli/org.go` の `send` の help、`sendWaitDeadlineHint` | 変更なし | 3 つとも、既定 30 秒、手番が終わるまで届かないこと、何も打たないこと、`--timeout-ms` を長くして打ち直すことで合う(verify の Documentation drift の表と同じ) |
| CHANGELOG の類 | なし | repo に CHANGELOG はなく、release ノートは goreleaser が git log から作る。計画の Rollout が書く release ノートの文は `/pr` と release の仕事 |
| `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh` | rc 0、rc 0 | 下の Verification |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の LOW-1(README の 2 か所と quality-gates の 2 つの写しが、終了コード 0 だけで resolve してよいと書く) | 直した。条件は `/director` skill の「答える順」と同じ(終了コード 0、そのあと `ralph org read` で DECISION が送られた入力として出ていて入力欄に残っていないこと、そのあと resolve)。`Press up to edit queued messages` の待ち行列の扱いは skill に任せ、README と quality-gates には書かない(条件が長くなり、skill の文と二重になるため) |
| verify の LOW-2(根拠の表の古いファイル名) | 直した |
| verify の INFO-1(実機の記録の 1 文) | 直した。事実の記録は残し、括弧で時点を書いた |
| verify の INFO-2(古い順序を書く 2 つの文) | 残した。仕様 126 行は 8a-1 の記録で、`ralph org approve` の出力の「once that send succeeded」は届いたことまで含めて読める |
| self-review の観察(RESOLVED の注のテスト) | 直した |
| test の穴 1(mutant V4: `--timeout-ms` が 0 以下のときの hint の ms) | 文書には書かない。計画の Progress が「PR の Known gaps に書く」とした |
| self-review の LOW(F3) | 計画の Progress が「`/pr` で PR 本文に書く」とした。この run は触れない |

## Plan drift notes

- 計画の Progress は全部 `[x]` で、残るのは「PR created」だけ。この run が計画を触れていないので digest は承認のまま
- Affected areas が挙げない `README.md`、`docs/quality/quality-gates.md`(と template の写し)をこの run が直した。verify の LOW-1 が計画の範囲の外と書いたもので、コードは変えていない

## Left open

- Codex の leader は、座席を待つあいだ `ralph org send` が届かない(計画の Non-goals。tech-debt の F7 の行が残る)。README の 254 行はこれを書いた
- 実機の確かめは、Codex の leader、`guarded` の leader、opus と sonnet の leader、25 分の期限でのループの終わりと張り直し、`ralph org wait` と gate を `run_in_background` で走らせる形、02e27384 のあとの雛形を含まない(`docs/evidence/busy-leader-2026-10-23.md` の「確かめていないもの」)
- 計画の Rollout が書く release ノートの文は、まだどこにも書かれていない。`/pr` の本文か release のときに書く

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-23-busy-leader.md` | `f046c0eac2ff`(承認の digest と同じ) |
| `git diff --check`、差分のホームのパスを含む追加行の数、差分の U+FFFD の数 | rc 0、0 件、0 件 |
| テスト | 打っていない(`/test` の範囲。この run が変えたのは文書と insight event だけで、Go とスクリプトには触れていない) |

## Insight event

追記した。`./scripts/insights-append.sh --slug busy-leader --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` が rc 0 で、`docs/insights/events/2026-10-23-busy-leader.jsonl` の末尾の行(cycle 1)になった。
