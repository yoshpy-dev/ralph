# Self-review report: reviewer-runs-gate

- Date: 2026-10-26(1 回目と 2 回目は同じ日)
- Plan: `docs/plans/active/2026-10-26-reviewer-runs-gate.md`(D1〜D8、AC1〜AC8、Progress checklist の逸れ 2 件)
- Reviewer: reviewer サブエージェント(`/self-review`)
- 1 回目の範囲: `git diff origin/main...HEAD`(HEAD 6182cb2e、origin/main 633cd7c8、35 ファイル、+1742/-551、9 コミット)。コード(`permissions.go`、`spawn.go`、コメントと help の文言)、雛形 2 つ(`leader.md`、`reviewer.md`)、テスト、`/org` と `/director` skill の 4 写し、recipe、README、quality-gates、仕様、根拠の表、tech-debt、`git-commit-strategy.md`。診断の対象は diff の品質だけで、テストと静的検査は走らせていない。
- 2 回目の範囲: 同じ diff の HEAD 67f3609d(37 ファイル、+2008/-582、13 コミット)。1 回目の報告のコミット a183673e のあとの 3 コミットが直しで、286f3d28(leader と reviewer の雛形とピン: F2、F3、F4、F5、N1)、64f36f3f(Go: F6、F1 のコメント)、67f3609d(skill、recipe、tech-debt と写し: F1、F7、F8)。N2 は意図して直していない。診断の対象は同じく diff の品質だけで、テストと静的検査は走らせていない。
- 番号: 1 回目の F1〜F8、N1・N2 はそのまま残す。テストのコメントが「self-review F2 and F5」「F3a」「F3b」「N1」「F4」でこの報告の番号を指しているため、同じファイルを書き換えても指す先が変わらないようにしている。2 回目に足す指摘は F9 から続ける(F9、F10、N3)。「1 回目の表」の Evidence 欄の行番号は HEAD 6182cb2e のもので、直しの 3 コミットで動いている。2 回目の表の行番号は HEAD 67f3609d を `sed -n` で引き直したもの。

## Evidence reviewed

### 2 回目(HEAD 67f3609d)

- 直しの 3 コミットを全文読んだ(`git show 286f3d28 64f36f3f 67f3609d`)。HEAD の `leader.md` のミッション 5・6、手順 3・5・9〜14、運用規律(`:42-44`、`:249-334`、`:523-524`)と、`reviewer.md` の「gate の TASK」の節(`:44-109`)、`/org` skill の「org の終わり方」(`.claude/skills/org/SKILL.md:943-1075`)と reviewer の項(`:1304-1335`)を通して読んだ。
- 手順 12 の新しい文を、依頼された観点ごとに読んだ。結果は次のとおり。
  - reviewer を手順 13 まで止めないこと: 手順 12 の頭(`leader.md:249-251`)に書かれ、ミッション 6(`:42-44`)と運用規律(`:523-524`)が互いに指す。曖昧さはない。
  - 動いていなければ spawn し直す: `ralph org status` の STATE の欄の `(active)` は実際の出力(`internal/cli/org.go:1796`)で、コマンドは `--org-id` と `--state-dir` を取る(`:1653`)。spawn し直しの flag は手順 14 の列挙(`leader.md:331-334`)を指し、そこに `--role reviewer` がある。手順 3 の早い reviewer の spawn も「手順 5 と同じ flag」(`:147-149`)で、手順 5 に `--role` が入ったので同じ。曖昧さはない。
  - 同じ理由で頼み直すのは 2 回まで: 数える単位が文から決まらない(F9)。
  - QUESTION の枝: 本文を 2 つのコマンドだけに直して新しい TASK_ID で 1 回送り直し、何が違ったのかわからなければ BLOCKED を上げる(`:313-317`)。送り直しの次に同じ QUESTION が来ても、後半の「わからなければ」で閉じる。QUESTION は TASK_ID を必須にしないので、leader は内容で見分ける。足りている。
  - 汚れた作業ツリーの枝: 手順 9 が 3 つに分ける(reviewer のレポートは leader がコミット、implementer の変更は implementer に TASK、それ以外は BLOCKED)うち、手順 12 の枝は implementer の変更と「それ以外」を書き、「手順 9 と同じく分ける」(`:289`)で全体を手順 9 に結ぶ。reviewer のレポートの分は書かれていないが、gate の TASK で reviewer は worktree を書き換えない(`reviewer.md:68-70`)ので起きにくい(N3)。implementer の枝は手順 10 の終わりの計画の戻しを先に打ち(`:290`)、手順 8 の往復に戻る。手順 9 と同じ形。
- `/org` skill の項 4 と 6 を、`leader.md` の手順 12 と 14 に文ごとに照らした(`SKILL.md:977-1032`、`:1050-1053`)。止めない、`status` で `(active)` を見る、spawn し直し、頼み直しの上限の文(`:998-1000`)、汚れの断りの分け方、QUESTION の枝、`--role reviewer` を含む flag の列挙が、同じ内容で並ぶ。skill は文が短く、escalate の本文に何を書くか(残ったファイルなど)を一部略しているが、食い違いではない。
- 写しを `cmp` で確かめた。`.claude/skills/{org,director}/SKILL.md` と `templates/base/.claude/skills/...`、`.agents/skills/...` と `templates/base/.agents/skills/...` は同一。`.claude/` と `.agents/` は先頭 5 行を除いて 0 行の差。recipe の root と `templates/base/` は同一。
- `reviewerRole` の定数: 定義は `approval.go:70` の 1 か所、使うのは `permissions.go:199` の 1 か所。非テストの Go で役割を `"reviewer"` の文字列で決める所はもうない(`spawn.go:2022` はコメント、`prompts.go:315` は撤去した役割の後継の表)。テストに同名の識別子はない。
- コードの差分 8 ファイル(`org_gate.go`、`org_regate.go`、`approval.go`、`gate.go`、`regate.go`、`regate_merge.go`、`spawn.go`、`permissions.go`)を読み直した。README、quality-gates、`git-commit-strategy.md` の変更文と、tech-debt の変更行(両方向。下の「肯定的な点」)も読み直した。新しい問題は F9、F10、N3 だけ。
- 確かめていない: テスト、静的検査、`check-sync.sh`・`check-skill-sync.sh` の実行(`/verify` と `/test` の範囲)、実機(herdr、codex の sandbox)での動き。pin のテストは文の部分一致を読んだだけで、走らせていない。

### 1 回目(HEAD 6182cb2e)

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

### 1 回目の指摘と、HEAD 67f3609d での状態

行は 1 回目のまま(Evidence 欄の行番号は 6182cb2e のもの)。最後の列が 2 回目の判定。

| Severity | Area | Finding | Evidence | Recommendation | Status(2 回目) |
| --- | --- | --- | --- | --- | --- |
| MEDIUM (F1) | security | 機能の org の codex reviewer は `--add-dir <台帳>` で台帳のディレクトリ全体に書ける。台帳の下には、承認の記録(`plan_approved`)と gate の記録(`gate_passed`。merge-order と merge が根拠にする)が入る manifest、`inbox.jsonl`、座席の prompt ファイル(`prompts/`)、`director.jsonl`、`model-receipts.jsonl` がある。sandbox はこれを止めず、歯止めは `reviewer.md` の 1 文(「`ralph org gate` のほかの `ralph org` の動詞は打たず、台帳のファイルを直接書きません」)と、gate の TASK のコマンドを 2 つと照らす規則だけ。gate は repo の `run-test.sh` を reviewer の子として走らせるので、implementer が書いたテストのコードも、台帳に書ける sandbox の中で動く(codex の leader が gate を打っていた間も同じで、claude の座席には sandbox がなく前から同じ)。範囲は機能の org の reviewer に限られ、plan の D5 と R1 が受け入れているが、`permissions.go` のコメントと recipe は広げた理由だけを書き、残る危険を書かない。同じコメントの旧版は、implementer と reviewer に足さない理由を「manifest、receipts、他座席の prompt を書き換えられるから」と書いていて、その危険は今回の reviewer にそのまま当てはまる | `internal/org/permissions.go:156-181`(特に `:171-181`)、`:195`。`internal/org/approval.go:32`、`internal/org/approve.go:346-357`(`plan_approved` を manifest に書く)。`internal/org/inbox.go:221`。`internal/org/director_start.go:1143`(`prompts/`)。`internal/org/prompts/reviewer.md:12-15`。`docs/recipes/codex-seat-permissions.md:89-103`(`templates/base/` の写しも同じ)。`docs/tech-debt/README.md:216`(座席が台帳に書けることは既存の行が認めている)。計画の D5・R1 | コードは直さなくてよい(決定は D5 で、範囲はテストが固定している)。`permissions.go` のコメントと recipe に 1 文足す: 「reviewer は leader と同じく、承認と gate の記録を台帳に直接書ける。止めるのは雛形の文だけ」。tech-debt の 216 行目に、codex の reviewer も同じになったことを 1 文足すか、行を 1 つ足す。PR の本文の初めに、2026-10-07 にユーザーが選んだ「`--add-dir` は leader だけ」を変えたことを書く(D5 が「報告の初めに書く」としている) | 解消。`permissions.go:171-181` のコメントが、台帳全体(`plan_approved` と `gate_passed` の記録、inbox、prompt ファイル)に書けること、gate が repo のテストのコードを同じ sandbox の中で走らせること、止めるのは雛形だけであることを書く。recipe(`:94-97`、`templates/base/` の写しも同じ)と tech-debt の 216 行目(承認の行に、codex の reviewer も同じことができる、と 1 文)にも入った。PR の本文の初めに決定の変更を書くのは `/pr` に残る(下の Follow-ups) |
| MEDIUM (F2) | maintainability | 手順 12 は reviewer が生きていることを前提にするが、leader の一般則「座席は作業が終わるたびに / 使い終わったら都度 `ralph org stop`」(ミッション 6、運用規律)は、機能の org の節で読み替えられていない。reviewer の RESULT のあとに止めた leader が手順 12 の TASK を断られたときの手は書かれておらず、reviewer の spawn し直しは手順 14 の HEAD 不一致の枝にだけある。旧版は gate を leader が打ったので、reviewer が生きている必要があるのは gate が落ちた枝だけだった。新版は通る枝(いつもの道)でも要る | `internal/org/prompts/leader.md:42`、`:497`(一般則)、`:245`(手順 12)、`:293`(手順 13)、`:306-309`(spawn し直しは手順 14 だけ)。`/org` skill `:977-1040` も同じ | 手順 12 の初めに 1 文足す: 「reviewer は手順 13 まで止めない。止まっていれば、手順 14 の spawn し直しと同じ flag で立ててから送る」。skill の 4 の頭にも同じ文(4 写し) | 解消。手順 12 の頭(`leader.md:249-254`)に、止めないことと、`ralph org status` で `(active)` がなければ手順 14 と同じ flag で spawn し直してから送ることが入った。ミッション 6(`:42-44`)と運用規律(`:523-524`)に例外への指しがあり、skill の項 4(`SKILL.md:977-984`)と 4 写しも同じ。止めてしまった leader も手順 12 で拾われる。implementer の側が残る(F10) |
| LOW (F3) | maintainability | 手順 12 の頼み直しに回数の上限がなく、閉じていない枝がある。(a) `HEAD:` が合わない、または `.out` に `GATE: pass` の行がないとき「新しい TASK_ID で頼み直す」(`:270-272`)と、unrunnable の環境の問題で直して頼み直す枝(`:281-285`)は、1 回が数十分の gate を、同じ理由で何度でも頼める。(b) `reviewer.md` は、本文のコマンドが 2 つと違うとき QUESTION で聞くと決める(`:57-61`)が、leader の手順 12 に、その QUESTION への次の手(本文を直して `gate-<n+1>` を送り直す、など)がない。一般の裁定(ミッション 5)に任されている | `internal/org/prompts/leader.md:270-272`、`:281-285`。`internal/org/prompts/reviewer.md:57-61`。`leader.md:32` | 「同じ理由で 2 回続けたら `ralph org escalate` で BLOCKED を上げる」と、QUESTION の枝を 1 行足す。テストの pin は文を固定するので、足したら pin も足す | 解消(上限の数え方の曖昧さは F9)。(a) 上限の文が手順 12 の結果の前(`leader.md:270-274`)に入り、skill の項 4(`SKILL.md:998-1000`)にも写った。(b) QUESTION の枝(`leader.md:313-317`、skill `:1028-1032`)が入った。pin は `prompts_test.go:1088`、`:1097` |
| LOW (F4) | maintainability | unrunnable の枝は「gate の断り(作業ツリーが汚れている、…)なら、stderr に書かれた次の手をあなたが打つ」(`:273-276`)。汚れの断りの次の手は「commit or remove」(`gate.go:504`)で、手順 9 の「ほかの変更が残っていれば、自分ではコミットしない。implementer の変更なら…それ以外は escalate」(`:204-209`)と食い違う。この文は旧版の手順 12 から動かしたもので、新しいのは、leader と reviewer が同じ worktree を共有し、断りの原因が reviewer の置いたファイルでありうる点 | `internal/org/prompts/leader.md:273-276`、`:204-209`。`internal/org/gate.go:504`。`/org` skill `:1001-1004` | 汚れの断りだけ「手順 9 の分け方(誰の変更か)に従う」と 1 句足す | 解消。汚れの断りだけ、stderr の次の手を自分では打たず、手順 9 と同じく分ける(`leader.md:288-295`、skill `:1011-1017`)。ほかの断りは従来どおり。gate が汚れの断りに「commit or remove them」を出すことは、テストが固定している(`prompts_test.go` の `gateDirtyRefusal` の確認)。reviewer のレポートの分の書き方は N3 |
| LOW (F5) | maintainability | 手順 14 の spawn し直しの flag の列挙に `--role reviewer` がない。`--role` は必須で(`internal/cli/org.go:771-774`)、手順 5 の列挙にもない(前から)。leader の雛形は `--role` をどこにも書かない。codex の `--add-dir` は role が文字列 `reviewer` と一致することに頼る(`permissions.go:195`)ので、列挙を「手順 5 と同じ flag」と言い切る新しい文は、足りない列挙で不足を固定する | `internal/org/prompts/leader.md:306-308`、`:157-175`(手順 5)。`internal/cli/org.go:771`、`:819`。`/org` skill `:1022-1030` | 列挙に `--role reviewer` を足す。手順 5 にも `--id` の隣に `--role` を書く | 解消。手順 5(`leader.md:162-163`)に `--role` を `--id` の隣に足し、手順 14(`:331`)と skill の項 6(`SKILL.md:1051`)の列挙に `--role reviewer` を足した。手順 3 の reviewer の spawn は「手順 5 と同じ flag」なので、そちらにも届く。pin は `prompts_test.go:1103`、`:1131` |
| LOW (F6) | naming | `codexWritableRootArgs` が role を文字列 `"reviewer"` で比べる。同じ形の述語に、名前つきの定数 `implementerRole`(`approval.go:63-66`、コメント付き。`spawn.go:1525`、`verbs.go:550` が使う)がある。非テストのコードで `"reviewer"` を判定に使うのはここだけで、`grep` で役割の述語を探すとき、片方だけ定数になる | `internal/org/permissions.go:195`。`internal/org/approval.go:63-66` | 同じ場所に `reviewerRole = "reviewer"` を足して使う。1 か所だけなので、このままでも害は小さい | 解消。`reviewerRole` を `implementerRole` の隣に足し(`approval.go:67-70`)、`permissions.go:199` が使う。コメントも `reviewerRole` に直った(`:157-158`) |
| LOW (F7) | readability | 追加した節の折り返しに、文の途中で切れた短い行が出た(幅 24、22、48。ほかの行は 71〜77)。skill の 2 か所(`director/SKILL.md:938`「gate を打たせる。同じ」、`org/SKILL.md:1049`「に直しを頼むため。その」。後者は前からあった短い行「ため report は…」の直前)と、recipe の `:96-97`(「choice of 2026-10-07 (the ralph repository's plan」「`2026-10-26-reviewer-runs-gate`). Seats of every」)。skill は 4 写し、recipe は 2 写しに同じものがある。`/org` skill の reviewer の項(`:1284-1313`)は 1 つの箇条が 30 行になり、同じ内容が `reviewer.md` の「gate の TASK」の節にあるので、skill はその節への指しだけで足りる | `.claude/skills/director/SKILL.md:937-939`。`.claude/skills/org/SKILL.md:1048-1050`、`:1284-1313`。`docs/recipes/codex-seat-permissions.md:96-97` | 該当の段落を組み直す(語は変えない)。reviewer の項は、返し方の 3 つと待ち方の要点だけ残して、詳細は雛形の節を指す | 解消(1 点だけ別の形)。短い行の 3 か所は組み直され、4 写しと recipe の 2 写しで同じ。reviewer の項は、指しにせず 5 つの小箇条に分けた(`SKILL.md:1314-1335`)。読みやすくなったが、`reviewer.md` との重複は残る。LOW のまま持ち越すほどではなく、下の Tech debt の行にも入れない |
| LOW (F8) | maintainability | scaffold 先に届く recipe に、決定の履歴の句「This widens the leader-only choice of 2026-10-07 (the ralph repository's plan `2026-10-26-reviewer-runs-gate`)」が入った。下流の利用者には「leader-only choice」が何を指すか分からず、同じ履歴は仕様と計画にある。recipe は「何を足すか」だけを書く文書で、前例の「the ralph repository's `docs/evidence/…`」は参照先であって履歴ではない | `docs/recipes/codex-seat-permissions.md:95-97`、`templates/base/docs/recipes/codex-seat-permissions.md:95-97` | 句を削る(F7 の組み直しと同じ段落) | 解消。句は消え、F1 の文に置き換わった(`docs/recipes/codex-seat-permissions.md:95-97`、`templates/base/` の写しも同一) |
| INFO (N1) | readability | `reviewer.md` の「gate の TASK」の節は、先に「本文のコマンドをそのまま打ってください」(`:48-49`)と書き、あとの段落で例外(2 つと違えば QUESTION、`:57-61`)を書く。上から読む LLM は、最初の文で止まりうる。`<台帳>` の値だけは「2 つと同じか」を照らせない(自由な path)ので、シェルに解釈される文字を含む path を打たない、の 1 句があるとよい。TASK は leader から来るものだけが指示なので害は小さい | `internal/org/prompts/reviewer.md:46-61` | 例外の段落を先にする。順序を pin するテストはない | 解消(主旨のみ)。最初の文が「本文のコマンドが次の 2 つと同じときだけ、そのまま打ってください」になった(`reviewer.md:48-49`)。`<台帳>` の path にシェルの文字が入る場合の 1 句は足していない。INFO のまま |
| INFO (N2) | readability | `watch.go` のコメント (c) が 36 行の 1 つの塊で、4 つの話(gate の記録の出所、regate の記録、どの座席が台帳に書けるか、基準値の維持)を持つ。中身は合っている(`checkDeadman` の条件を読んで確かめた)が、次に手を入れる人は、どの文が今回の理由かを探すことになる | `internal/org/watch.go:1345-1380` | 今回の理由(基準値の維持、副作用)を 1 段落に分けるとよい。急がない | 直していない(意図して)。「急がない」とした INFO で、中身は合っている。次に `watch.go` のコメント (c) に手を入れるときに分ける |

### 2 回目の新しい指摘

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (F9) | readability | 手順 12 の頼み直しの上限は、数える単位が文から決まらない。「頼み直しが同じ理由(…)で 2 回続けて通らなければ、3 回目は頼まず」は、2 つに読める。(a)「頼み直し」を数える: 最初の TASK が落ち、1 回目の頼み直しも 2 回目も同じ理由で落ちたら、3 回目の頼み直しは送らない(同じ理由の gate は最大 3 回走る)。(b) gate の TASK を数える: 最初と 1 回目の頼み直しが同じ理由で落ちたら、3 回目の TASK は送らない(最大 2 回)。主語が「頼み直し」で、「3 回目」は TASK の回数のようにも読める。差は gate 1 回分(数十分)で、どちらに読んでも BLOCKED は上がるので害は小さい。同じ文が雛形、skill の 4 写し、pin にあり、いまの pin は文を丸ごと固定している | `internal/org/prompts/leader.md:270-274`。`.claude/skills/org/SKILL.md:998-1000`(`templates/base/` と `.agents/` の写しも同じ)。`internal/org/prompts_test.go:1088` | 単位を書く。たとえば「同じ理由で gate の TASK が 2 回続けて通らなければ(最初の TASK と 1 回目の頼み直し)、3 回目の TASK は送らず」。直すなら雛形、skill の 4 写し、pin の 3 か所 |
| LOW (F10) | maintainability | ミッション 6 と運用規律の例外は reviewer だけを名指しする(「機能ごとの org の reviewer だけは、…手順 13 まで止めない」)が、手順 13 は implementer と reviewer を一緒に止め(`leader.md:318`)、gate が落ちたとき(`GATE: fail`)も、汚れの断りの implementer の枝も、rebase が衝突したときも、手順 12 は implementer に TASK を送る。skill は「座席を gate のあとまで止めないのは、gate を reviewer が打ち、gate が落ちたときに implementer に直しを頼むため」(`SKILL.md:1068-1070`)と、座席を両方とも指して書く。雛形の一般則(作業が終わるたびに止める)を字どおりに読む leader は、implementer を RESULT のあとに止め、`GATE: fail` の TASK を断られる。そのとき implementer を立て直す手は、雛形のどこにもない。この食い違いは前からあった(旧版は reviewer も同じ扱いだった)が、reviewer にだけ例外を書いたので、implementer は止めてよいという読みが強まった | `internal/org/prompts/leader.md:42-44`、`:318`、`:523-524`。`.claude/skills/org/SKILL.md:1068-1070`。例外の pin は `internal/org/prompts_test.go:1140` | 例外を「機能ごとの org の implementer と reviewer は手順 13 まで止めない」に広げる(ミッション 6、運用規律、手順 12 の頭の 3 か所、skill の項 4、pin)。広げない場合は、`GATE: fail` の前に implementer が止まっていたときの spawn し直し(`--role implementer`、per-plan では承認が要る)を手順 12 に書く |
| INFO (N3) | readability | 手順 12 の汚れの断りの枝は、「手順 9 と同じく分ける」と言いながら、書き下すのは implementer の変更と「それ以外」の 2 つだけ。手順 9 の 3 つ目(reviewer のレポートは leader がコミットする)は書かれず、「reviewer のほかのファイル」の「ほか」が間接に示すだけ。gate の TASK で reviewer は worktree のファイルを書き換えない(`reviewer.md:68-70`)ので、reviewer のレポートが汚れの原因になる場面は、レビューの TASK のあとに gate をやり直す場合に限る。害は小さい | `internal/org/prompts/leader.md:288-295`、`:204-209`(手順 9)。`internal/org/prompts/reviewer.md:68-70` | 直さなくてよい。直すなら「reviewer のレポートなら手順 9 のとおりコミットして、新しい TASK_ID で頼み直す」と 1 句足す |

## Positive notes

1 回目:

- 動きの変更がコードではほぼ 1 行(`codexWritableRootArgs` の述語と、Spawn の 1 行)に収まり、残りは文言とテスト。`featureOrg` はロックの下で読んだ `events` から決めていて、新しいロックも読み直しも増やしていない。dry-run の道と `spawnPrecheckErr` は触っていない。
- 範囲の外が表で固定されている。機能の org の implementer、結びつきのない org の reviewer、大文字の `Reviewer`、guarded の codex、claude、台帳が cwd の下にある場合、director が `nil` になる行がある(`permissions_test.go`)。spawn の argv のテストは `--add-dir` の位置(permission の flag のすぐあと、`--model` の前)まで見る(`spawn_test.go`)。
- leader の雛形と reviewer の雛形で、gate のコマンドの文字列を 1 つの定数(`leaderGateCommand`)で突き合わせている。leader の雛形に leader 自身が `ralph org gate` を打つ文が残っていないことは、文の単位に切って検査するテストで固定し、そのテストが旧版の文を落とすことも確かめている(`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`)。
- 手順 9 の書き出しを TASK_ID で分けたこと(gate の RESULT で手順 9 に戻るとコミットが増えて HEAD がずれる)と、手順 12 の pass の枝で座席を止める前に `wip:` を確かめること、手順 14 の HEAD 不一致で reviewer を spawn し直すことは、計画の R2 と R8 の穴を、雛形、skill、テストの 3 か所で同じ文で塞いでいる。
- 見張りの数え方を変えなかった(D6)ので、コードと保留中の ALERT の基準値は更新をまたいでずれない。コメントはその理由を書いている。
- 記録の強さがそろっている。spec、evidence、tech-debt はいずれも「固定した」と書き、「確かめた」と言っていない。実機で動かしていないことを FR-7 の `[ ]` に反映し、数(25 と 6)が表と合う。
- `docs/plans/active/2026-10-26-reviewer-runs-gate.md` を指す参照が仕様(`:39`、`:124`、`:126`)と根拠の表(`:79`)にあり、`/pr` の archive で書き換わるのは tech-debt だけだが、同じ形の前例があり、前の報告も取り上げていないので、ここでも取り上げない。

2 回目:

- 直しは 3 コミットに分かれ、1 回目の指摘ごとに範囲が閉じている。雛形とピン(286f3d28)、Go(64f36f3f)、skill・recipe・tech-debt と写し(67f3609d)で、1 つのコミットが別の層の直しを抱えない。雛形に足した文はどれも `prompts_test.go` に pin が付く(読んだだけで、走らせていない)。
- 写しが直しのあとも一致している(`cmp` で確認、上の Evidence)。skill の 4 写しと recipe の 2 写しを同じコミットで直した。
- tech-debt は両方向で合っている。F1 の文は承認の行(`:216`)に入り、「Codex の leader が待つ」行は、leader の待ちから gate を外した(reviewer が打つ gate は leader の待ちではないため)。`wip:` の行は「手順 12 と 14」に直った。開いたままのはずの行が閉じられたことも、閉じたはずの行が開いたままのこともない。
- 手順 3 の早い reviewer の spawn が「手順 5 と同じ flag」を指すので、F5 の `--role` は手順 3 にも届く。
- F6 の定数は、コメントの `role == reviewerRole` の書き方まで直っている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Codex の reviewer が gate を前面で打つあいだ、`ralph org send` が reviewer に届かない(leader からの STOP、gate の TASK への DECISION)。tech-debt にあるのは Codex の leader が待つ行(`:330`)だけで、reviewer の行がない(2 回目でも未記録) | gate は数十分かかるので、そのあいだ leader は reviewer に何も送れない。reviewer の停滞の ALERT が出ても、leader は `ralph org read` で見るだけになる | 計画の Progress checklist が `/sync-docs` の判断に回している | 実機で Codex の reviewer の gate を動かしたとき、または send の idle の待ちを変えるとき | `internal/org/prompts/reviewer.md:63-67`、`docs/tech-debt/README.md:330`、計画 R4・D7 |
| 手順 12 とその写し(skill の項 4)の 2 点(2 回目の F9 と F10)。(a) 頼み直しの上限の数える単位が決まらず、同じ理由の gate が 2 回とも 3 回とも読める。(b) 「作業が終わるたびに座席を止める」の例外が reviewer だけで、implementer は手順 13 まで要るのに、止めてよいと読め、`GATE: fail` の前に止まっていたときの立て直しがない(前からある食い違いで、skill の「座席を gate のあとまで止めない」と雛形が違う) | (a) 余計な gate が 1 回(数十分)走りうるだけで、BLOCKED は上がる。(b) 一般則を字どおりに読む leader の機能の org で、`GATE: fail` の TASK が implementer に届かず、leader は BLOCKED を上げるしかない | 既定の上限 2 回のうち 2 回目(最後)の self-review で見つかった LOW。直すと雛形、skill の 4 写し、pin に当たり、パイプラインを `/self-review` から回し直す 3 回目になる | 手順 12〜14 か、ミッション 6・運用規律の止める決まりに次に手を入れるとき。または、機能の org の実機で、gate が同じ理由で 2 回続けて落ちた、または implementer が `GATE: fail` の前に止まっていた最初の報告 | この報告の F9、F10。`internal/org/prompts/leader.md:42-44`、`:270-274`、`:318`、`:523-524`。`.claude/skills/org/SKILL.md:998-1000`、`:1068-1070` |

_(行は `docs/tech-debt/README.md` に足していない。この報告だけを書く指示のため。1 つ目は `/sync-docs` の判断に回すと計画が書いている。2 つ目は、上限をこのまま 2 回で終える場合に `/sync-docs` が足す行の文面。1 回目の表の 2 つ目の行(codex の reviewer が台帳に書ける)は、F1 の直しで tech-debt の `:216` に入ったので、ここからは外した。)_

## Recommendation

- Merge: OK(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2、INFO 2。いずれも HEAD 67f3609d で開いているもの)。1 回目の MEDIUM 2、LOW 6、INFO 1(N1)は解消、INFO 1(N2)は意図して直していない。2 回目の新しい指摘は LOW 2(F9、F10)と INFO 1(N3)で、手順 12 の文の意味(F9)と、止める決まりの例外の幅(F10)に当たり、どちらも害は小さい。leader と reviewer の雛形、skill、README、help は、TASK_ID、2 つのコマンド、`.out` の経路、3 つの返し方、spawn し直しの flag で矛盾しない。
- 手順 12 の読みやすさの答え: reviewer を手順 13 まで止めないこと、動いていなければ手順 14 の flag で spawn し直すこと、QUESTION の枝、汚れた作業ツリーの分け方は曖昧さがない。頼み直しの上限は数える単位が決まらない(F9)。skill の項 4 と 6 は、雛形の手順 12 と 14 と同じことを言っている。
- Follow-ups:
  - F9 と F10 はテキストだけの直しで、雛形、skill の 4 写し、pin に当たる。この回は既定の上限 2 回のうちの 2 回目なので、直すとパイプラインを `/self-review` から回し直す 3 回目になる。勧めは、直さずに上の Tech debt の 2 つ目の行を `/sync-docs` に足してもらうこと。上限を上げて回し直すなら、F9 と F10 を 1 コミットにまとめる(F10 は例外の幅を広げる側を勧める)。
  - `/verify` で、`check-skill-sync.sh` と `check-sync.sh` が通ること(`cmp` は一致を確認済み)、`gate` を打つ人を書く文で「leader が打つ」が残っていないこと(掃引済み。仕様の 7b と 8a-3b の注は意図して残してある)を見る。
  - PR の本文の初めに、codex の `--add-dir` を leader だけにする 2026-10-07 の決定を変えたことと、reviewer が打つ gate は実機で動かしていないこと(FR-7 を `[ ]` にした理由)を書く。
  - insight event は、次の 1 行を、この報告のコミットとは別に打つ(この報告では他のファイルを書かない指示のため、打っていない。`insights-append.sh` は UTC の日付でファイルを選ぶので、追記先の名前は実行した日に従う)。2 回目の分の件数は、開いた指摘の数で、`--cycle auto` は `cycle-count.json` から 2 回目になる:
    `./scripts/insights-append.sh --slug reviewer-runs-gate --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 2 --cycle auto --source skill`
