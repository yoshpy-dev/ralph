# Self-review report: reviewer-runs-gate

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-26-reviewer-runs-gate.md`(D1〜D8、AC1〜AC8、Progress checklist の逸れ 2 件)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD`(HEAD 6182cb2e、origin/main 633cd7c8、35 ファイル、+1742/-551、9 コミット)。コード(`permissions.go`、`spawn.go`、コメントと help の文言)、雛形 2 つ(`leader.md`、`reviewer.md`)、テスト、`/org` と `/director` skill の 4 写し、recipe、README、quality-gates、仕様、根拠の表、tech-debt、`git-commit-strategy.md`。診断の対象は diff の品質だけで、テストと静的検査は走らせていない。

## Evidence reviewed

- コードの差分 8 ファイルを全文読んだ。`codexWritableRootArgs` の判定(`permissions.go:195`)、Spawn の呼び出し(`spawn.go:1050-1051`)と、その `events` が manifest のロックの下の読み直し(`spawn.go:777`、`:958`)であること、`ActiveFeature` の定義(`reserve.go:495`)を確かめた。
- `leader.md` の差分と、手順 5・9・10・12・13・14、ミッション 5・6・7、運用規律(`:497`)を通して読み、`reviewer.md` を全文読んだ。leader の TASK の本文(`:245-260`)と reviewer の「gate の TASK」の節(`reviewer.md:44-109`)で、TASK_ID `gate-<n>`、2 つのコマンド、`.harness/state/gate-<org_id>.out`、3 つの返し方(`GATE: pass` の RESULT、`GATE: fail` と `GATE: unrunnable` の BLOCKED)が同じ形かを突き合わせた。合っている。例のメッセージ 2 つは `protocol.Parse` と `Validate` を通すテストが付いている(`prompts_test.go` の `TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`)。
- `watch.go` の ALERT のヘッダ(`SEAT`、`CONDITION`、`condStall = "stall"`、`:1127`)が、雛形の「`SEAT: reviewer`、`CONDITION: stall`」と合うこと、`checkDeadman` の `activity && !subjectIsLeader`(`:1587`)がコメント(c)の説明と合うことを読んだ。
- `/org` skill の「org の終わり方」1〜7(`:943-1045`)を leader の手順 9〜14 と順に照らした。順序と分岐は合う(skill の 4 が手順 12、5 が 13、6 が 14)。`/org` と `/director` の 4 写しは本文が一致(`.claude/` と `templates/base/.claude/`、`.agents/` と `templates/base/.agents/` は `cmp` で同一、`.claude/` と `.agents/` は先頭 5 行を除いて 0 行の差)。recipe と `git-commit-strategy.md` の root と template は `cmp` で同一。quality-gates は、前から違う 54・57・58 行目と 82 行目の「(see docs/tech-debt)」の有無を除いて、今回足した文が両側で同じ。
- 根拠の表の `[x]` と `[ ]` の行を数え直した(25 と 6)。文に書かれた数と合う。表と仕様が挙げるテスト名 5 つ(`TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`、`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`、`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`、`TestCodexWritableRootArgs_FeatureOrgReviewerGetsAddDir`、`TestOrgSpawn_Codex_WorkspaceWrite_FeatureOrgReviewerGetsAddDir`)は diff の中に実在する。
- 「leader が gate を打つ」を言う文が残っていないかを、`git ls-files` の全ファイルに `leader`・`gate` の正規表現を当てて掃引した(計画・報告・旧 evidence の写しを除く)。残りは、仕様の 7b と 8a-3b の注(FR-7 の「2026-10-26 に決めたこと」が「その時点の決定として残す」と書いている)と、`docs/evidence/verify-2026-10-13-org-gate-help.txt`(当時の help の写し)だけ。「`--add-dir` は leader だけ」も同じ掃引で、仕様の 1 段目の行(`:258`)が今回の変更を足されていることを見た。
- 秘密、debug 用の出力、`TODO`、コメントアウトしたコード、未使用の変数は diff に見つからなかった。
- 確かめていない: テスト、静的検査、`check-sync.sh`・`check-skill-sync.sh` の実行(`/verify` と `/test` の範囲)、実機(herdr、codex の sandbox)での動き。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (F1) | security | 機能の org の codex reviewer は `--add-dir <台帳>` で台帳のディレクトリ全体に書ける。台帳の下には、承認の記録(`plan_approved`)と gate の記録(`gate_passed`。merge-order と merge が根拠にする)が入る manifest、`inbox.jsonl`、座席の prompt ファイル(`prompts/`)、`director.jsonl`、`model-receipts.jsonl` がある。sandbox はこれを止めず、歯止めは `reviewer.md` の 1 文(「`ralph org gate` のほかの `ralph org` の動詞は打たず、台帳のファイルを直接書きません」)と、gate の TASK のコマンドを 2 つと照らす規則だけ。gate は repo の `run-test.sh` を reviewer の子として走らせるので、implementer が書いたテストのコードも、台帳に書ける sandbox の中で動く(codex の leader が gate を打っていた間も同じで、claude の座席には sandbox がなく前から同じ)。範囲は機能の org の reviewer に限られ、plan の D5 と R1 が受け入れているが、`permissions.go` のコメントと recipe は広げた理由だけを書き、残る危険を書かない。同じコメントの旧版は、implementer と reviewer に足さない理由を「manifest、receipts、他座席の prompt を書き換えられるから」と書いていて、その危険は今回の reviewer にそのまま当てはまる | `internal/org/permissions.go:156-181`(特に `:171-181`)、`:195`。`internal/org/approval.go:32`、`internal/org/approve.go:346-357`(`plan_approved` を manifest に書く)。`internal/org/inbox.go:221`。`internal/org/director_start.go:1143`(`prompts/`)。`internal/org/prompts/reviewer.md:12-15`。`docs/recipes/codex-seat-permissions.md:89-103`(`templates/base/` の写しも同じ)。`docs/tech-debt/README.md:216`(座席が台帳に書けることは既存の行が認めている)。計画の D5・R1 | コードは直さなくてよい(決定は D5 で、範囲はテストが固定している)。`permissions.go` のコメントと recipe に 1 文足す: 「reviewer は leader と同じく、承認と gate の記録を台帳に直接書ける。止めるのは雛形の文だけ」。tech-debt の 216 行目に、codex の reviewer も同じになったことを 1 文足すか、行を 1 つ足す。PR の本文の初めに、2026-10-07 にユーザーが選んだ「`--add-dir` は leader だけ」を変えたことを書く(D5 が「報告の初めに書く」としている) |
| MEDIUM (F2) | maintainability | 手順 12 は reviewer が生きていることを前提にするが、leader の一般則「座席は作業が終わるたびに / 使い終わったら都度 `ralph org stop`」(ミッション 6、運用規律)は、機能の org の節で読み替えられていない。reviewer の RESULT のあとに止めた leader が手順 12 の TASK を断られたときの手は書かれておらず、reviewer の spawn し直しは手順 14 の HEAD 不一致の枝にだけある。旧版は gate を leader が打ったので、reviewer が生きている必要があるのは gate が落ちた枝だけだった。新版は通る枝(いつもの道)でも要る | `internal/org/prompts/leader.md:42`、`:497`(一般則)、`:245`(手順 12)、`:293`(手順 13)、`:306-309`(spawn し直しは手順 14 だけ)。`/org` skill `:977-1040` も同じ | 手順 12 の初めに 1 文足す: 「reviewer は手順 13 まで止めない。止まっていれば、手順 14 の spawn し直しと同じ flag で立ててから送る」。skill の 4 の頭にも同じ文(4 写し) |
| LOW (F3) | maintainability | 手順 12 の頼み直しに回数の上限がなく、閉じていない枝がある。(a) `HEAD:` が合わない、または `.out` に `GATE: pass` の行がないとき「新しい TASK_ID で頼み直す」(`:270-272`)と、unrunnable の環境の問題で直して頼み直す枝(`:281-285`)は、1 回が数十分の gate を、同じ理由で何度でも頼める。(b) `reviewer.md` は、本文のコマンドが 2 つと違うとき QUESTION で聞くと決める(`:57-61`)が、leader の手順 12 に、その QUESTION への次の手(本文を直して `gate-<n+1>` を送り直す、など)がない。一般の裁定(ミッション 5)に任されている | `internal/org/prompts/leader.md:270-272`、`:281-285`。`internal/org/prompts/reviewer.md:57-61`。`leader.md:32` | 「同じ理由で 2 回続けたら `ralph org escalate` で BLOCKED を上げる」と、QUESTION の枝を 1 行足す。テストの pin は文を固定するので、足したら pin も足す |
| LOW (F4) | maintainability | unrunnable の枝は「gate の断り(作業ツリーが汚れている、…)なら、stderr に書かれた次の手をあなたが打つ」(`:273-276`)。汚れの断りの次の手は「commit or remove」(`gate.go:504`)で、手順 9 の「ほかの変更が残っていれば、自分ではコミットしない。implementer の変更なら…それ以外は escalate」(`:204-209`)と食い違う。この文は旧版の手順 12 から動かしたもので、新しいのは、leader と reviewer が同じ worktree を共有し、断りの原因が reviewer の置いたファイルでありうる点 | `internal/org/prompts/leader.md:273-276`、`:204-209`。`internal/org/gate.go:504`。`/org` skill `:1001-1004` | 汚れの断りだけ「手順 9 の分け方(誰の変更か)に従う」と 1 句足す |
| LOW (F5) | maintainability | 手順 14 の spawn し直しの flag の列挙に `--role reviewer` がない。`--role` は必須で(`internal/cli/org.go:771-774`)、手順 5 の列挙にもない(前から)。leader の雛形は `--role` をどこにも書かない。codex の `--add-dir` は role が文字列 `reviewer` と一致することに頼る(`permissions.go:195`)ので、列挙を「手順 5 と同じ flag」と言い切る新しい文は、足りない列挙で不足を固定する | `internal/org/prompts/leader.md:306-308`、`:157-175`(手順 5)。`internal/cli/org.go:771`、`:819`。`/org` skill `:1022-1030` | 列挙に `--role reviewer` を足す。手順 5 にも `--id` の隣に `--role` を書く |
| LOW (F6) | naming | `codexWritableRootArgs` が role を文字列 `"reviewer"` で比べる。同じ形の述語に、名前つきの定数 `implementerRole`(`approval.go:63-66`、コメント付き。`spawn.go:1525`、`verbs.go:550` が使う)がある。非テストのコードで `"reviewer"` を判定に使うのはここだけで、`grep` で役割の述語を探すとき、片方だけ定数になる | `internal/org/permissions.go:195`。`internal/org/approval.go:63-66` | 同じ場所に `reviewerRole = "reviewer"` を足して使う。1 か所だけなので、このままでも害は小さい |
| LOW (F7) | readability | 追加した節の折り返しに、文の途中で切れた短い行が出た(幅 24、22、48。ほかの行は 71〜77)。skill の 2 か所(`director/SKILL.md:938`「gate を打たせる。同じ」、`org/SKILL.md:1049`「に直しを頼むため。その」。後者は前からあった短い行「ため report は…」の直前)と、recipe の `:96-97`(「choice of 2026-10-07 (the ralph repository's plan」「`2026-10-26-reviewer-runs-gate`). Seats of every」)。skill は 4 写し、recipe は 2 写しに同じものがある。`/org` skill の reviewer の項(`:1284-1313`)は 1 つの箇条が 30 行になり、同じ内容が `reviewer.md` の「gate の TASK」の節にあるので、skill はその節への指しだけで足りる | `.claude/skills/director/SKILL.md:937-939`。`.claude/skills/org/SKILL.md:1048-1050`、`:1284-1313`。`docs/recipes/codex-seat-permissions.md:96-97` | 該当の段落を組み直す(語は変えない)。reviewer の項は、返し方の 3 つと待ち方の要点だけ残して、詳細は雛形の節を指す |
| LOW (F8) | maintainability | scaffold 先に届く recipe に、決定の履歴の句「This widens the leader-only choice of 2026-10-07 (the ralph repository's plan `2026-10-26-reviewer-runs-gate`)」が入った。下流の利用者には「leader-only choice」が何を指すか分からず、同じ履歴は仕様と計画にある。recipe は「何を足すか」だけを書く文書で、前例の「the ralph repository's `docs/evidence/…`」は参照先であって履歴ではない | `docs/recipes/codex-seat-permissions.md:95-97`、`templates/base/docs/recipes/codex-seat-permissions.md:95-97` | 句を削る(F7 の組み直しと同じ段落) |
| INFO (N1) | readability | `reviewer.md` の「gate の TASK」の節は、先に「本文のコマンドをそのまま打ってください」(`:48-49`)と書き、あとの段落で例外(2 つと違えば QUESTION、`:57-61`)を書く。上から読む LLM は、最初の文で止まりうる。`<台帳>` の値だけは「2 つと同じか」を照らせない(自由な path)ので、シェルに解釈される文字を含む path を打たない、の 1 句があるとよい。TASK は leader から来るものだけが指示なので害は小さい | `internal/org/prompts/reviewer.md:46-61` | 例外の段落を先にする。順序を pin するテストはない |
| INFO (N2) | readability | `watch.go` のコメント (c) が 36 行の 1 つの塊で、4 つの話(gate の記録の出所、regate の記録、どの座席が台帳に書けるか、基準値の維持)を持つ。中身は合っている(`checkDeadman` の条件を読んで確かめた)が、次に手を入れる人は、どの文が今回の理由かを探すことになる | `internal/org/watch.go:1345-1380` | 今回の理由(基準値の維持、副作用)を 1 段落に分けるとよい。急がない |

## Positive notes

- 動きの変更がコードではほぼ 1 行(`codexWritableRootArgs` の述語と、Spawn の 1 行)に収まり、残りは文言とテスト。`featureOrg` はロックの下で読んだ `events` から決めていて、新しいロックも読み直しも増やしていない。dry-run の道と `spawnPrecheckErr` は触っていない。
- 範囲の外が表で固定されている。機能の org の implementer、結びつきのない org の reviewer、大文字の `Reviewer`、guarded の codex、claude、台帳が cwd の下にある場合、director が `nil` になる行がある(`permissions_test.go`)。spawn の argv のテストは `--add-dir` の位置(permission の flag のすぐあと、`--model` の前)まで見る(`spawn_test.go`)。
- leader の雛形と reviewer の雛形で、gate のコマンドの文字列を 1 つの定数(`leaderGateCommand`)で突き合わせている。leader の雛形に leader 自身が `ralph org gate` を打つ文が残っていないことは、文の単位に切って検査するテストで固定し、そのテストが旧版の文を落とすことも確かめている(`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`)。
- 手順 9 の書き出しを TASK_ID で分けたこと(gate の RESULT で手順 9 に戻るとコミットが増えて HEAD がずれる)と、手順 12 の pass の枝で座席を止める前に `wip:` を確かめること、手順 14 の HEAD 不一致で reviewer を spawn し直すことは、計画の R2 と R8 の穴を、雛形、skill、テストの 3 か所で同じ文で塞いでいる。
- 見張りの数え方を変えなかった(D6)ので、コードと保留中の ALERT の基準値は更新をまたいでずれない。コメントはその理由を書いている。
- 記録の強さがそろっている。spec、evidence、tech-debt はいずれも「固定した」と書き、「確かめた」と言っていない。実機で動かしていないことを FR-7 の `[ ]` に反映し、数(25 と 6)が表と合う。
- `docs/plans/active/2026-10-26-reviewer-runs-gate.md` を指す参照が仕様(`:39`、`:124`、`:126`)と根拠の表(`:79`)にあり、`/pr` の archive で書き換わるのは tech-debt だけだが、同じ形の前例があり、前の報告も取り上げていないので、ここでも取り上げない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Codex の reviewer が gate を前面で打つあいだ、`ralph org send` が reviewer に届かない(leader からの STOP、gate の TASK への DECISION)。tech-debt にあるのは Codex の leader が待つ行(`:330`)だけで、reviewer の行がない | gate は数十分かかるので、そのあいだ leader は reviewer に何も送れない。reviewer の停滞の ALERT が出ても、leader は `ralph org read` で見るだけになる | 計画の Progress checklist が `/sync-docs` の判断に回している | 実機で Codex の reviewer の gate を動かしたとき、または send の idle の待ちを変えるとき | `internal/org/prompts/reviewer.md:63-67`、`docs/tech-debt/README.md:330`、計画 R4・D7 |
| 機能の org の codex reviewer が台帳に直接書ける(F1)。承認と gate の記録を座席が書き換えうることの、codex の reviewer への広がり | F1 のとおり。歯止めは雛形の文だけ | 計画の D5 と R1 が受け入れた。行を足す先は `docs/tech-debt/README.md:216` の隣 | codex の座席に台帳の一部だけを書かせる仕組み(`--add-dir` を台帳の下の `gates/` と manifest に絞れるか)が使えるようになったとき | 計画の D5・R1、`docs/tech-debt/README.md:216` |

_(行は `docs/tech-debt/README.md` に足していない。この報告だけを書く指示のため。1 つ目は `/sync-docs` の判断に回すと計画が書いている。)_

## Recommendation

- Merge: OK(CRITICAL 0、HIGH 0、MEDIUM 2、LOW 6、INFO 2)。動きの変更は小さく、範囲の外はテストで固定されている。leader と reviewer の雛形、skill、README、help は、TASK_ID、2 つのコマンド、`.out` の経路、3 つの返し方で矛盾しない。MEDIUM の 2 件は雛形または文書の 1 文の足しで直り、F1 は決定済みの範囲を変えない。F2 は、いつもの道で reviewer が止まっていたときに手順が閉じない点なので、直してから merge するのを勧める。
- Follow-ups:
  - 直すなら、F1〜F5 は `leader.md`、skill、コメント、recipe に当たり、この task の記録(`docs/reports/` の task の報告と insight event)ではない。`.claude/rules/ralph/post-implementation-pipeline.md` の記録だけの修正の例外は使えないので、pipeline を `/self-review` から回し直す(上限の 2 回のうちの 2 回目)。F2・F3・F4・F5 は `leader.md` の手順 12 と 14 の近くに集まるので、1 コミットにまとめ、`prompts_test.go` の pin を足す。F6〜F8 と N1・N2 は急がない。
  - `/verify` で、`check-skill-sync.sh` と `check-sync.sh` が通ること(`cmp` は一致を確認済み)、`gate` を打つ人を書く文で「leader が打つ」が残っていないこと(掃引済み。仕様の 7b と 8a-3b の注は意図して残してある)を見る。
  - PR の本文の初めに、codex の `--add-dir` を leader だけにする 2026-10-07 の決定を変えたことと、reviewer が打つ gate は実機で動かしていないこと(FR-7 を `[ ]` にした理由)を書く。
  - insight event は、次の 1 行を、この報告のコミットとは別に打つ(この報告では他のファイルを書かない指示のため、打っていない。`insights-append.sh` は UTC の日付でファイルを選ぶので、追記先の名前は実行した日に従う):
    `./scripts/insights-append.sh --slug reviewer-runs-gate --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 2 --low 6 --cycle auto --source skill`
