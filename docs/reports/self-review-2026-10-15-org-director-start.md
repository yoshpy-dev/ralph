# Self-review report: org-director-start

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest d59311f57d8e。8 段目の 8a-2)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: diff の品質だけ。`git diff 9cbf7080...HEAD`(HEAD 731cba34、11 コミット、35 ファイル、+6848/-301)。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs の担当なので行っていない。計画の D1〜D11(先に登録する D3、token の名前 D9、`stop --force` の D11 など)と、計画に書いた既知の穴(tech-debt の新しい 5 行)は、欠陥として数えない。

## Evidence reviewed

- 読んだ差分(コード、全体): `internal/org/director_start.go`(640 行)、`director_stop.go`(468 行)、`director.go` の diff すべて、`internal/cli/org_director_headless.go`(355 行)、`org_director.go` の diff すべて、`org.go`・`status.go`・`org_legacy_ledger.go` の diff すべて、`spawn.go`・`permissions.go`・`prompts.go`・`envelope.go`・`verbs_all.go` の diff すべて、`prompts/director.md`
- 読んだ差分(文書): `/director` skill(363 行すべて)、`templates/base/ralph.toml` の 2 か所、`AGENTS.md` の 1 行(word diff)、README の 3 か所、仕様 FR-9 の追記 1 行、`docs/tech-debt/README.md` の追加 5 行(`|` で分けた列は全行 5)
- 読んだテスト: `internal/cli/org_director_test.go` の diff すべて(削除行は、attach の `--driver` の test を attach と start の 2 つの動詞に広げた書き換えで、弱めていない)、`org_director_headless_test.go` の helper(1〜138 行)と `RefusedWithNothingWritten`・`WatchGate`、`director_stop_test.go` の `ForceWhenAGetTimesOut`。新しい test 名の一覧は 3 ファイルとも確認。残りの本文は読んでいない
- 読んだ周辺のコード: `confirmSeatPane`・`confirmOrgWorkspace`・`isNotFound`(`verbs.go`)、`herdrAgentName`・`writePromptFile`・`PromptFilePointer`・Spawn の `agentArgs` の組み立て(`spawn.go`)、`identifierPattern`(`identifier.go`)、`CommandWithStateDir`・`shellQuote`、`driver.Herdr.WorkspaceCreate`(エラー時に出力を返す癖)、`newOrgRuntimeAt` の runner の作り方、`orgInboxHintStateDir`、secret scan のパターン(`scripts/secret-scan.sh`)
- 確かめたこと
  - `git diff --check` は空。追加行に TODO・FIXME・debug 出力(`fmt.Print*`、`log.*`、`t.Log`)・panic・秘密らしい文字列はない(test の `"token":"0123456789ab"` は secret scan の generic パターン `access_token` などに当たらない)。生成物の混入もない
  - skill は 4 面が `md5` で同一(6509f6c7…)
  - **herdr に打つ破壊的な呼び出し**: `grep -rn 'PaneClose(\|PaneSendKeys(\|WorkspaceClose('` で、この diff が足したのは `director_stop.go` の 370・375・400 行の 3 か所だけ。3 か所とも `closeDirectorPane`・`closeDirectorWorkspace` の中で、呼び出し元は `closeConfirmedDirectorResources` だけ。そこへ入る道は 2 つで、`StopDirector`(137 行、`if closed.err == nil` の中)と `closeDirectorResources`(289 行、`if c.err != nil { return c }` のあと)。`--force` の道は `closed.err != nil` のまま `recordDirectorStop` へ進み、close を打つ分岐を持たない。`confirmDirectorResources` は pane と workspace の両方の label を確かめ、どちらかが確認できなければ、どちらにも何も送らない。確認できない id には C-c も close も届かない
  - label の文字列: `confirmSeatPane(wsLabel, tabLabel, paneID)` は pane の tab の label と workspace の label を文字列で比べるだけで、manifest を読まない。`confirmOrgWorkspace` も同じ。org の workspace・座席の tab の label に `_` は入らない(`identifierPattern`)ので、`ralph_director_<token>`・`director_<token>` と重ならない
  - 打つコマンドの quoting: `--state-dir` は `CommandWithStateDir` → `shellQuote` を通る。herdr の id は `directorShellWord`(英数字と `_.:-` 以外は `shellQuote`)。`{{STATE_DIR}}` は雛形の描画で `shellQuote`。`id` は `directorIDNumber` で `d<N>` に限り、token は `^[0-9a-f]{12}$` でしか畳み込みに入らない。端末への出力は `inboxField` を通る
  - 雛形: `strings.NewReplacer` は置換した文字を読み直さず、描画後の `{{` の残りを最後に断る。`{{CWD}}` は引用なしだが、雛形のどのコマンドにも入っていない
  - 並行性: `AttachHeadless`・`RecordWorkspace`・`RecordPane`・`MarkLaunched`・`markStopped` は `director.lock` の下で読み直してから決め、herdr を打つ間はロックを持たない。`markStopped` は現在の登録なら `detached` と `stopped` を 1 回の書き込みにし、`detached` を先にする理由がコメントにある。start の `abort` は、herdr が返した id を記録の成否にかかわらず閉じる(`r.reg.WorkspaceID` は返った時点で入る)。`WorkspaceCreate`・`TabCreate` がエラー時に出力を返す癖は、`= ""` に戻して扱っている
  - Spawn との引数の順: `permArgs`、`codexWritableRootArgs`、`--model`、雛形の 1 文、の順で Spawn(`spawn.go:898-923`)と同じ
  - 畳み込み: 新しい field のない 8a-1 の行はそのまま読める。headless の `attached` の `pane_id` は読まない(`registrationOf`)。`workspace_created`・`pane_created` は最初の 1 つを残す。`result()` は登録のコピーを返す
  - 8a-1 の test の書き換え 2 つ(`TestOrgDirector_AttachRefusesABadDriverBeforeReadingAnything`、`TestOrgDirectorHelp_DescribesTheRegistration`)は、元の確認を残したまま足している。`root` で動く環境の skip(`Geteuid`)も残っている。20 ms の `DriverCallTimeout` を使う test は、時間切れが起きる側を見ていて、勝ち負けの競合ではない
  - register の 5 行は `|` で分けて全行 5 列。Trigger 列に 8a・director・headless を含む行を全部抜き出して照らした(M1)
- 確かめていないもの: テスト、`gofmt`、`go vet`、go1.25.8 でのビルド。本物の herdr での動き(`_` を含む 27 字の label が返ること、実機の agent start。27 字は 15 + 12 で、初版の「22 字」は /sync-docs で訂正した)。`ralph org director start|stop` を binary で動かした出力。`director_start_test.go`・`director_test.go`・`director_stop_test.go` の本文の大半

## Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 1、LOW 6。

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1: 登録簿の行のうち、この PR の段(8a-2、headless の director)を Trigger や本文で名指ししたものが、そのまま残っている。192 行は「trigger は headless の director(`ralph org director start`、stage 8a-2)とその自動化に移る」と書いたまま、8a-2 が入ったのに判断の 1 文がない。245 行は「`pane_id` field は、stage 8a-2 の headless director が書くまで空」と書いているが、headless の `attached` は `pane_id` を持たず、pane は `pane_created` から来る(この PR が `DirectorEvent` の doc を直したのに、行は直っていない)ので、この文は偽になった。189・212・234〜241 行の「stage 8a の director」は、8a-1 の director への判断だけで、headless の director への 1 文がない | `docs/tech-debt/README.md:192`・`:245`・`:189`・`:212`・`:234-241`。`internal/org/director.go` の `registrationOf`(`reg.State, reg.PaneID = DirectorStateStarting, ""`)と `DirectorEvent` の doc。8a-1 の self-review の M2 も同じ型 | 245 行の文を「pane は `pane_created` が書く。attached の `pane_id` は promoted session の分だけ」の趣旨に直す。192・189・212・234〜241 には、8a-1 が兄弟の行に書いたのと同じ形の判断を 1 文ずつ足す(headless の director も同じ `/director` skill に従う session なので 8a-1 の判断が変わらないのか、変わるのか、行ごとに言う)。登録簿は records-only の例外の外なので、cycle 1 の fix か /sync-docs の回で入れる |
| LOW | exception-handling | L1: `director start` の help・skill・コメントは「start は、director が attached の間は何も書かず herdr も打たずに断る」と言うが、見張りの確かめ(`checkWatchGate`)が attached の確かめ(`AttachHeadless`)より前にある。見張りが動いていないときに attached の登録があると(attach は見張りなしでも通り、警告するだけ)、start は見張りを起動して `watch.log`・`watch.lock`・印を書き、stderr に `started the cross-org watch` と出してから、attached で断る。test は attached の場合に見張りの gate を差し替えて動いている状態にしてあり、この道を通らない | `internal/org/director_start.go:445-452`(`checkDirectorStart` の 514 行で gate、449 行で `AttachHeadless`)。`org_director_headless.go:61-66`(help)。`.claude/skills/director/SKILL.md:126`。`org.go:378-392`(「input checks のあと、書く前に gate」「入力で断られた start は見張りを起動しない」)。`org_director_headless_test.go:235-282`(`setEscalateWatch` で attached の case だけ見張りを受け付け状態に) | 動きは `ralph org start`(Spawn の idempotent・上限の確かめが gate のあと)と同じで害は小さい。help・skill・`org.go` のコメントを「attached の確かめの前に、見張りがなければ起動することがある」に直すか、`checkDirectorStart` で `store.Read()` を 1 回見て、current があれば gate の前に断る(AttachHeadless は引き続きロックの下で決める) |
| LOW | maintainability | L2: headless の登録を `detach --id`・`detach --force`・`attach --replace` で外すと、成功の出力は `detached d1` と `attached d2 ..., replacing d1 ...` だけで、d1 の pane と workspace が開いたままであることも、`stop` で閉じることも言わない。それを言うのは `attach` が断られたときの文(`attachedError`)と `status` の `director:` の行だけ。人の手で外す道(8a-2 の D4 で「detach は記録だけ」を保った道)の出力に 1 行がない | `internal/cli/org_director.go:193-197`(`detached %s` の 2 つの Fprintf)、`:130-137`(attach の成功の出力)。`ended`・`replaced` は `Launch` を持つので `Headless()` で分けられる。`detach --help` も headless に触れていない | `ended.Headless()`・`replaced.Headless()` のとき `its herdr pane and workspace stay open: ralph org director stop --id d1` を 1 行足す。`detach --help` に 1 文 |
| LOW | maintainability | L3: `orgDirectorLine` と `orgDirectorOpenHeadlessNote` が出す `ralph org director stop --id dN` に、この呼び出しの `--state-dir` が付かない。同じ PR の他の打ち方(`attachedError`・`startAttachedError`・`stop --all` の note・`DirectorCleanupError.StopCommand`)は `CommandWithStateDir` で付ける。`ralph status --state-dir X` と `director status --state-dir X` で、既定の台帳に効く打ち方が出る(8a-1 の「hint が既定を仮定する」型) | `internal/cli/org_director.go:492`・`:529`・`:531`。`orgDirectorLine(r orgDirectorReport)` は report だけを受け取る。`internal/cli/org.go` の `printStopAllDirectorNote`(同じ打ち方に `org.CommandWithStateDir`) | `orgDirectorLine` と note に hint を引数で渡し、`CommandWithStateDir` を通す。status の test の文字列は `--state-dir` 付きに変わる |
| LOW | readability | L4: director の stop が seat 用の確認関数(`confirmSeatPane(orgID, seatID, ...)`、`confirmOrgWorkspace(orgID, ...)`)に workspace の label と tab の label を渡して再利用していて、label の不一致のとき利用者に出る文に `seat` と `org_id` の語が混ざる(`pane "p1" is not seat "director_<token>"'s: ...`、`workspace "ws-1" is not org_id "ralph_director_<token>"'s: herdr labels it "org-a"`)。関数の doc(「seatID's pane in orgID's workspace、spawn が付けた label」)と引数名は、呼び出しが 1 種類だった前提のまま。test は外側の文と `labels it "org-a"` だけを固定していて、内側の語は固定していない | `internal/org/verbs.go:798-808`・`:809`・`:837-845`・`:850-855`・`:867-868`。`internal/org/director_stop.go:312`・`:320`(`fmt.Errorf("pane %q is not confirmed as %s's ...: %w", ..., err)` が内側の文を包む)。`director_stop_test.go:113` | 引数名を `workspaceLabel`・`tabLabel` にして doc を「label」に直し、内側の文から seat・org_id の語を外す。直さないなら、doc に「`confirmDirectorResources` も director の label で呼ぶ」と 1 行足す |
| LOW | naming | L5: package をまたぐためだけの pass-through が 2 つできた。`ModelAllowedForRole` は未公開の `modelAllowedForRole` を返すだけ。`DirectorRoleInputErr` は `directorRoleSpawnErr` を返すだけで、名前も `...InputErr` と `...SpawnErr` で揃っていない(同じ検査が 2 つの名前を持ち、test は未公開の側を叩く)。既存の `RetiredRoleInputErr` は 1 つの公開関数で、この形が前例 | `internal/org/envelope.go:149-153`、`internal/org/spawn.go:1428-1441`、`internal/cli/org.go:690`、`internal/cli/org_director_headless.go:178`。呼び出しは `modelAllowedForRole` が 2 か所(`envelope.go:59`、`envelope_summary.go:77`) | 元の関数を公開名に変える(`directorRoleSpawnErr` → `DirectorRoleInputErr`、`modelAllowedForRole` → `ModelAllowedForRole`)。呼び出し側の置換だけで済み、wrapper が要らない |
| LOW | readability | L6: コメントの手入れが残っている。(a) 折り返しが片方だけ直った長い行(他の行は 72〜80 桁)が 4 か所: `director.go:311`(115 桁)・`director.go:521`(88 桁)・`permissions.go:158`(88 桁)・`org.go:357`(86 桁)。(b) `directorRoleSpawnErr` の doc が `Like` で行を切る(`spawn.go:1425`)。(c) `notRestoredError` の doc が「attach or detach」のまま、start・stop も通る(`org_director.go:345`) | 各行は左の path:行。(a) は編集箇所の途中だけ詰め直した形 | 4 行を詰め直し、(b)(c) を直す。1 コミットで足りる |

## Positive notes

- 破壊的な 3 つの呼び出し(C-c、pane close、workspace close)が、label の確認を通った道にしかない。`--force` は確認できないものに何も打たず、記録だけを書く。確認の失敗、時間切れ、label の違い、id の振り直しを、test が別々に固定している
- 先に登録して herdr を打つ間はロックを持たない形が、コードと doc コメントで一致している。start の後始末が、herdr の返した id を記録の成否に関係なく閉じる(`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade`)
- 畳み込みが、新しいイベントを知らない古い binary でも壊れない形で書かれ、`detached` を `stopped` より先に書く理由(読み手が間の状態を見ても止められる側に倒れる)がコメントにある
- 新しい出力はすべて `inboxField`、JSON は encoder、描画は `shellQuote` を通る。hand edit した `director.jsonl` から端末と shell に値が漏れる道は見つからなかった
- 8a-1 の test の書き換えが「文の更新」と test のコメントに明記され、確認は残っている。main worktree の `ralph.toml` で読む規則は、`withMainWorktreeDirectorStartConfig` に 4 つの key だけを移す形で、他の key(leader が書ける設定)は動かさない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(登録簿には追記していない。M1 は新しい debt ではなく既存の行の書き換えで、L1〜L6 は小さな差分で直せる。cycle 1 の fix で直さず最後の cycle に届いた LOW だけを、1 つの行にまとめて追記する。)_

## Coverage gaps

- 実機の動き: herdr 0.7.5 が `ralph_director_<token>`(27 字、`_` を含む。初版の「22 字」は /sync-docs で訂正した)の label をそのまま返すこと、claude と codex の agent start が通ること、`ralph org director start|stop` の出力を見ていない。計画の Evidence to capture と AC13 のとおり /test の担当
- L1 の道(見張りなし、director が attached、start)を通る test がない
- `internal/org/director_test.go`(+958 行)ほか 3 つの新しい test ファイルの本文の大半は読んでいない。test 名の一覧と、拒否・時間切れ・label の違いの test だけを読んだ

## Recommendation

- Merge: yes(CRITICAL・HIGH はなし。MEDIUM 1 件は登録簿の文の直しで、コードの欠陥ではない。LOW 6 件は help の文言、hint、命名、コメントで、herdr に打つ処理と登録の畳み込みに欠陥は見つからなかった)
- Follow-ups:
  1. M1: 登録簿の 245 行の文を直し、192・189・212・234〜241 に 8a-2 の 1 文を足す。cycle 1 の fix か /sync-docs の回で入れる(登録簿は records-only の例外の外)
  2. L1: help・skill・`org.go` のコメントの文を直すか、`checkDirectorStart` で gate の前に current を見る。どちらでも skill の 4 面を同時に直す
  3. L2・L3: 出力と hint に `--state-dir` と pane が開いたままの 1 行を足す。status の test の期待文字列が変わる
  4. L4〜L6: 小さな差分なので、fix の回にまとめて入れるとよい。直さずに最後の cycle に届いたら、1 つの行にまとめて登録簿に足す
  5. Go のコメントの `docs/plans/active/2026-10-15-org-director-start.md` は、`/pr` の archive で古くなる。8a-1 のときと同じ形で、指摘としては数えない
- Known gaps: テスト、`gofmt`、`go vet`、go1.25.8 でのビルドは確かめていない。herdr の実機の動きと CLI の実際の出力は見ていない。新しい test の本文の大半は読んでいない

## Cycle 2

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest f65d7c59746c。cycle 1 の報告の時点は d59311f57d8e で、その後、verify の字数の訂正と cross-review の `--config` の 1 文で 2 回承認し直した)
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2)。`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定 2 のうち最後の回
- Scope: diff の品質だけ。cross-review cycle 1 の ACTION_REQUIRED 1 件の直し 3243c499(11 ファイル、+276/-45)と、/test の穴 D4・E1・E12/E13・P2 の test と登録簿の行の縮め 4ef33195(3 ファイル、+134/-4)を、全体の `git diff origin/main...HEAD`(HEAD 4ef33195)の中で読んだ。この節の ID には `C2-` を付け、cycle 1 の M1・L1〜L6 と取り違えないようにする。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs の担当なので行っていない

### Evidence reviewed

- 読んだ差分: 3243c499 と 4ef33195 の全 hunk(コード、test、skill の 4 面、`director.md`、登録簿の 1 行)。plan の 5a61a3cc と triage の 54cab8a2 は、直しの出どころとして読んだ
- 読んだ周辺: `RenderDirectorPrompt` と `directorPromptNoConfigText`、`director.md` の全体、`checkDirectorStart`・`StartDirector` の doc と本体、`startDirector`・`stopDirector`・`startAttachedError`、`newOrgDirectorRuntime` の doc、`withMainWorktreeDirectorEnabled` と `withMainWorktreeDirectorStartConfig` の doc、`resolveOrgConfig`、`--config` の persistent flag(`org.go:52`)、`ralph status` の flag、`fakeHerdr` の `WorkspaceCreate`・`TabCreate`、`dirStartFixture`、`TestStartDirector_RefusesBeforeWritingOrCallingHerdr` の表と実行部、`TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml` の全体、`director_skill_sync_test.go` の冒頭、`/director` skill の 95〜190 行、plan の `--config` を含む行と Progress checklist、cross-review の triage report、test report の Test gaps、evidence の A12〜A14 の行、8a-1 の self-review の cycle 2 の節(形式の確認)
- 確かめたこと
  - cycle 1 の指摘 7 件(M1、L1〜L6)は、HEAD ですべて閉じている。M1: 登録簿の 189・192・212・245 行に 8a-2 の判断があり、245 行は headless の `attached` に `pane_id` がないことに直っている。L1: `checkDirectorStart` が gate の前に `Store.Read` している。L2: detach の出力に pane と workspace が開いたままの行がある。L3: `orgDirectorLine(r, hintStateDir)` が hint を受け取る。L4: `confirmSeatPane` の引数が `workspaceLabel`・`tabLabel`。L5: `directorRoleSpawnErr` はなく、公開名の `DirectorRoleInputErr` が 1 つ(`spawn.go:1430`)。L6: `origin/main...HEAD` の追加コメント行に 84 桁を超えるものはなく、「attach or detach」と行末の `Like` も残っていない。cycle 1 から持ち越した LOW はないので、最後の回でも登録簿に足す分は cycle 2 の指摘だけになる
  - `--config` の道: CLI が `configPath` を `StartDirector` へ渡し(`org_director_headless.go:196`)、`checkDirectorStart` が gate のあとで絶対化し(`director_start.go:554-558`。state dir と cwd の絶対化と同じ位置)、`StartDirector` は `checkDirectorStart` のあとの `p` を `directorStartRun` にコピーするので、`writePrompt`(`:616`)は絶対化した値を渡す。`RenderDirectorPrompt` は絶対でない値を断る(`prompts.go:160-162`)。`{{STATE_DIR}}`・`{{ORG_FLAGS}}`・`{{CONFIG}}` の置換は `strings.NewReplacer` の 1 回で、置換後の文字は読み直されず、値に `{{` があれば最後の検査が断る
  - 戻すと赤くなるか(読んで判断した。実行はしていない): CLI の `ConfigPath: configPath` を消すと、`TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml` の `--config` の case が赤になる。`checkDirectorStart` の絶対化を消すと、`RenderDirectorPrompt` が相対を断るので、org の test と CLI の test の両方が赤になる。CLI の case は linked worktree で start を打ち、director は main root で動くので、絶対化の基準を `Cwd` に変えても赤になる(期待値は worktree 側の `ralph.toml`)。`IsAbs` の断りを消すと `a relative config path` の case が赤になる。雛形の 3 つの実行コマンドと `ralph status` の文のどれかを戻すと、`directorPromptCommands` の突き合わせ(`ralph org` の言及の個数がコード内の個数と等しいこと、実行する 3 つの末尾が flags であること)が赤になる。期待値の quoting は `shellQuote` を呼ばず `'/cfg dir/it'\''s ralph.toml'` と書いてあり、宣言の写しではない。TempDir のパスは `--config` を含む subtest 名を持つので、空白つきの ` --config` で探しているのは正しい
  - 4 つの穴の test: D4 は `wantGate` が false の行で、最初の `Store.Read` の失敗を無視する変異では gate が呼ばれて `gateCalls > 0` が赤になる。E1 は `AttachHeadless` の失敗のあとも進める変異で `workspace_create` が呼ばれ、herdr の呼び出しが 0 でなくなって赤になる。E12・E13 は、失敗した作成が出力を返す wrapper が `o.Herdr` に入っている(`newDirStartFixture` の `o.Herdr = f.h`)ので、出力を id として残す変異では `workspace_get`・`pane_get` の引数に出力が現れて赤になる。P2 は期待文が `startAttachedError` の format と一字一句同じで、`--state-dir` の有無を 2 行で固定している。`directorFileState` は directory を `<a directory>` と返すようになり、D4 の行の before と after の比較が成り立つ
  - 登録簿の 256 行は `|` で分けて 5 列。A12〜A14 の説明(A12: 失敗した workspace の get を閉じ済みとみなす、A13: tab を herdr が知らない pane を閉じ済みとみなす、A14: workspace を知らない pane を確認済みとみなす)と test 名は、evidence の 867〜869 行・949〜954 行と合う。D4・E1・E12/E13・P2 の 4 件は、この行から外れて test になった
  - `ralph status` に `--config` はない(`status.go:105-107` の flag は `--state-dir`・`--org-id`・`--json`)ので、skill の「`ralph status` には `--config` がない」は合う。`director stop`・`detach` は config を読まない(`newOrgDirectorRuntime` の `loadConfig` が false)ので、`--config` を足しても終了を止める道は増えない
  - skill の 4 面は `cmp` で同一(`.claude` を基準に、`.agents` と `templates/base` の 3 面)。`git diff --check e683ee47..HEAD` は空。追加行に TODO・debug 出力・`/Users/`・秘密らしい文字列・U+FFFD はない
  - 順序: `git log --format='%h parent=%p'` で、直しの 3243c499 の親は plan の 5a61a3cc、その親が triage の 54cab8a2、その親が AC の tick の e683ee47。cycle 1 の verify・test・sync-docs の報告は、どれも直しの前の tree を述べている
  - 指摘にしなかったもの: `filepath.Abs` は `..` を字面で畳むので、cwd が symlink 経由で `--config ../x` のとき、start が読むファイルと雛形のパスが分かれうる。同じ関数が state dir と cwd に使う形と同じで、repo の他の `absPath` の使い方とも揃っているので数えない。`resolve the --config path` の失敗の枝に test がないのは、隣の `resolve the state dir`・`resolve the cwd` も同じ(`absPath` を差し替えるのは `spawn_test.go` だけ)
- 確かめていないもの: テスト、`gofmt`、`go vet`、go1.25.8 でのビルド(新しい test が通ることは、読んだだけで実行していない)。本物の herdr で claude と codex の director が `--config` つきの雛形のとおりに動くこと。cwd が symlink 経由のときの `--config` の動き

### Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-L1 | LOW | maintainability | 雛形の stop の行は `--state-dir` のあとに `--config` を持つが、`director stop` は config を読まない。stop は `loadConfig` が false で作られ、help も「stop reads no ralph.toml」と書く。start が stdout に出す stop の行は `--config` なしで、直したコミットの `startDirector` の doc も「the stop command, which needs no --config」と書くので、同じ start から人に渡る stop が 2 通りになる。あわせて、「every ralph org command in the director's prompt carries --config」という言い方(help、`startDirector` の doc、`RenderDirectorPrompt` の doc)は、雛形の `ralph org` の言及 7 つのうち、実行する 3 つ(status が 2 か所、stop)にだけ当たる。残る 4 つ(`ralph org director start`、`ralph org watch --all --detach`、`ralph org director detach`、語としての `ralph org`)は打たない言及で、test の `notRun` がそれを除いている。害はない(stop は `--config` を無視し、読めない config が stop を止めることもない) | `internal/org/prompts/director.md:33`・`:50`。`internal/cli/org_director_headless.go:59`(help)・`:133`・`:140`・`:161`・`:164`・`:207`(start が出す stop の行)。`internal/org/prompts.go:139`。`internal/cli/org_director.go:287-293`(detach と stop は config を読まない)。`internal/org/prompts_test.go:1613`(stop にも flags を要求) | 雛形の stop の行を、`ralph status` と同じ `--state-dir {{STATE_DIR}}` だけにする。`prompts_test.go` の突き合わせは stop の末尾だけ `--state-dir` にする。残すなら、`director.md:33`・help・2 つの doc の「どれも」「every」を「director が打つ」に直し、`startDirector` の doc に、雛形の stop の行が `--config` を持つ理由を足す |
| C2-L2 | LOW | naming | `{{CONFIG}}` は値ではなく見出しの文を入れる。config があれば `'<path>'(start の --config)`、なければ散文の `directorPromptNoConfigText` になる。隣の `{{STATE_DIR}}` は、コマンドの中でそのまま動く shell の 1 語で、doc もそう書く。次に誰かが雛形のコマンドに `--config {{CONFIG}}` と書くと、`--config '/p'(start の --config)` か散文が出る。描画後の `{{` の残りの検査は通るので、見つからない。今の使い方(見出しの 1 行だけ)は合っていて、doc も「the header's name of the config」と書いている | `internal/org/prompts.go:134`・`:137-138`・`:141`・`:168-172`。`internal/org/prompts/director.md:4`・`:5` | 値でないと名前でわかる形(`{{CONFIG_NOTE}}` など)に改名する。変えるのは `director.md` と `prompts.go` の 2 か所で、test は描画後の文字列を見ているので変わらない |
| C2-L3 | LOW | readability | skill の「始める」の 1 は、director が最初に打つ `ralph org director status --state-dir <台帳>` を `--config` なしの字面のまま残し、`--config` の条件は 3 にだけ足した。3 の「雛形に設定のパスが書かれていれば」を見分ける目印は、雛形の `- 設定:` の行が config のときだけ持つ `(start の --config)` で、config なしのときの行は `main worktree の ralph.toml(…)` と、path に見える語を含む散文になる。skill はその目印にも `設定:` の行にも触れない。director が実際に打つのは雛形の字面(`director.md:13`)なので、害は小さい。「立てる」の節と「start の設定」の箇条には、`--config` が雛形に入る話がなく、help にだけある。skill の文と雛形の見出しの結びつきを固定する test はない(inbox の待ちの文言には `director_skill_sync_test.go` がある) | `.claude/skills/director/SKILL.md:149`・`:156-160`・`:101-145`。`internal/org/prompts.go:134`・`:172`。`internal/cli/org_director_headless.go:59-61`。`internal/org/director_skill_sync_test.go:40` | 1 を「`ralph org director status --state-dir <台帳>`(雛形に `--config` があればそれも付ける)」にし、3 の条件は「雛形の `- 設定:` の行が `(start の --config)` で終わっていれば」と目印を名指す。skill の 4 面を `scripts/sync-skills.sh` で揃える |
| C2-L4 | LOW | maintainability | Go のコメントが、書き換わる成果物の番号を指している。「cross-review cycle 1 finding 1」が 2 か所(`director_start_test.go:281` は slug もない)、「/test gap N of org-director-start, mutant X」が 4 か所。triage report は `/cross-review` が cycle ごとに同じ path へ書き換える(ヘッダは `Cycle: 1/2`)ので、cycle 2 の cross-review が終わると「finding 1」は別の指摘を指しうる。test report の gap の番号と evidence の mutant の ID も、cycle 2 の `/test` が作り直したときに同じとは限らない。8a-1 の self-review の cycle 2 の C2-L3 が同じ型で、直しは起きることを言葉で書く形だった。ここのコメントはどれも、欠陥を言葉でも書いているので、番号は足しているだけ。plan の 46 行にも同じ指し方がある | `internal/org/director_start_test.go:281`・`:450`・`:610`・`:651`。`internal/org/prompts_test.go:1573`。`internal/cli/org_director_headless_test.go:375`。`docs/plans/active/2026-10-15-org-director-start.md:46`。`docs/reports/cross-review-triage-org-director-start.md`(`Cycle: 1/2`) | 番号を外して言葉だけにするか、安定した AC11 に結ぶ(`prompts_test.go:1573` は AC11 も書いている)。`/cross-review` が走る前に直す。走ったあとでは、コメントの変更でも pipeline をもう 1 回回すことになる |
| C2-L5 | LOW | records | plan の Progress checklist は、cycle 2 の直しを「cycle 2 で直し、/test のテストの穴 D4・E1・E12/E13・P2 のテストも足す」と、これからする形で書いたままで、実際の commit(3243c499 と 4ef33195)を記録していない。S1〜S5 と cycle 1 の直し(S4 da818b4f)は commit つきで書いてある。`/pr` が plan を archive するので、archive には予定の文が残る。また AC11 は e683ee47 で tick されたあとに `--config` の 1 文が足され(5a61a3cc)、それを満たす 3243c499 より前から `[x]` のまま | `docs/plans/active/2026-10-15-org-director-start.md:183-187`・`:126`・`:46`。`git log --format='%h parent=%p'`(e683ee47、54cab8a2、5a61a3cc、3243c499 の順) | Progress の行を commit つきの過去形に直す。Progress は承認の digest に入らない行なので、digest は変わらない。plan は records-only の例外の対象外なので、/verify の前に直す。/verify は AC11 を HEAD で読み直す |

### Positive notes

- cycle 1 の 7 件が、持ち越しも半端な直しもなく閉じている。cycle 2 の直しは、cross-review の「雛形に入れ、読む `ralph org` の動詞に付ける」の字面ではなく、start の cwd を基準にした絶対化(director は別の directory で動く)、相対を断る検査、quoting(空白と単引用符)まで届いている
- CLI の test の case が、start を linked worktree で打ち、director を main root で動かす形になっていて、絶対化の基準を取り違える変異(`Cwd` 基準)を赤にする。org の test、prompt の test、CLI の test が、それぞれ別の層の変異を止めている
- `directorPromptCommands` の突き合わせは、`ralph org` の言及の個数がコード内の個数と等しいことを要求するので、雛形にバッククォートなしの `ralph org ...` が足されても、flags のない言及が増えても気づける。実行しない 4 つの言及は `notRun` に明示されている
- 4 つの穴の test は、それぞれ変異の入口を読んで赤になる形で、足した test が既存の表の行を弱めていない。`failedCreateOutput` は、本物の `driver.Herdr` が失敗でも stdout を返す癖を fake に足す形で、既存の seat の test の挙動(空文字列)は変えない
- 登録簿の行は、直った 4 件を外し、残る A12〜A14 だけを、今は穴でない理由(seat の test が止める)と Trigger(director stop が共有の関数を使わなくなったとき)つきで残している。行は 5 列のまま

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `director start` の雛形・skill・コメントの小さな残り(cycle 2 の self-review C2-L1〜C2-L5)。雛形の stop の行が stop の読まない `--config` を持つことと「every ralph org command」の言い方。`{{CONFIG}}` が値でなく見出しの文であること。skill の「始める」の 1 の字面と 3 の条件の目印。コメントの「cross-review cycle 1 finding 1」「/test gap N」の指す先。plan の Progress が cycle 2 の commit を持たないこと | どれも挙動を変えない。director が打つ `ralph org` のコマンドに `--config` が付く動きは、org・prompt・CLI の 3 層の test が固定している | cap 2 の最後の cycle で、文言・コメント・名前の変更に pipeline をもう 1 回回す価値がない | 次に `director.md` か `RenderDirectorPrompt` の placeholder を変えるとき(C2-L1・C2-L2)。次に `/director` skill の headless の節を変えるとき(C2-L3)。次にそのコメントのある test に触れるとき(C2-L4) | `docs/reports/self-review-2026-10-15-org-director-start.md`(Cycle 2 の C2-L1〜C2-L5) |

_(登録簿には追記していない。上の行は、C2-L1〜C2-L4 を直さずに進めるときに 1 行で足す形。C2-L5 は plan の記録で、`/pr` の前に直せるので行に入れない。)_

### Coverage gaps

- skill の `--config` の文(「雛形に設定のパスが書かれていれば」)と雛形の `- 設定:` の行を結ぶ test はない。inbox の待ちの文言には `director_skill_sync_test.go` があり、同じ形が取れる
- `resolve the --config path`(`absPath` の失敗)の枝に test がない。隣の state dir と cwd の枝も同じ
- 本物の herdr で claude と codex の director が、`--config` つきの雛形を読んで最初の `director status` を打つところは見ていない(test report の Test gaps 8 と同じ穴)
- cwd が symlink 経由で `--config` に `..` があるときの、start が読むファイルと雛形のパスの一致

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。LOW 5 件は雛形の stop の行と言い回し、placeholder の名前、skill の字面、コメントの番号、plan の記録で、`--config` を director の `ralph org` のコマンドへ運ぶ動きと、4 つの穴の test に欠陥は見つからなかった)
- Follow-ups:
  1. C2-L5: plan の Progress の 1 行を、/verify の前に直す(plan は records-only の例外の外)。C2-L4 のコメントは、`/cross-review` が triage を書き換える前に直す
  2. C2-L1〜C2-L3: 直すなら `director.md`・`prompts.go`・`prompts_test.go` と skill の 4 面を 1 つのコミットで。skill は `scripts/sync-skills.sh` で揃える
  3. 直さずに進むなら、上の Tech debt の行を登録簿に足す。cap 2 の最後の回で、plan は `/pr` が archive するので、登録簿の行が唯一の受け皿になる
  4. /verify への引き継ぎ: AC11 の tick は、AC11 の文が変わる(5a61a3cc)前、それを満たす直し(3243c499)が入る前のもの。cycle 1 の verify・test・sync-docs の報告と、README の director の段落(258 行)、`/director` skill の「立てる」の節、`/org` skill の director の行は、直しの前の tree を述べている。読み直しは各 phase の担当
- Known gaps: テスト、`gofmt`、`go vet`、go1.25.8 でのビルドは確かめていない。新しい test が通ることと、「戻すと赤くなる」は、読んで判断したもので実行していない。本物の herdr での `--config` つきの director の動きは見ていない

## Cycle 3

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest 43b4eae64cf2。cycle 2 の報告の時点は f65d7c59746c で、cross-review cycle 2 のあと D12・D13・AC15・S8 を足して承認し直した)
- Reviewer: reviewer subagent (Claude)。パイプライン 3 回目(cycle 3)。`RALPH_STANDARD_MAX_PIPELINE_CYCLES` を 3 に上げた回で、この PR の最後の回。`cycle-count.json` は 2 のままなので、insight event は `--cycle 3` を明示して足した
- Scope: diff の品質だけ。cross-review cycle 2 の ACTION_REQUIRED 2 件の直し。`git diff d2971e1a..HEAD`(HEAD 749ea59e、3 コミット、18 ファイル、+478/-153)を読んだ。4b721b18(plan に D12・D13・AC15・S8)、caf1d025(D12 と D13 のコードと文書、17 ファイル、+462/-145)、749ea59e(plan の記録)。全体の差分は、直しが前のコードに触れる所だけを読んだ。この節の ID には `C3-` を付け、cycle 1 の M1・L1〜L6 と cycle 2 の `C2-` と取り違えないようにする。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs の担当なので行っていない

### Evidence reviewed

- 読んだ差分: caf1d025 の全 hunk(`director_start.go`、`permissions.go`、`org_watch.go`、`org_director_headless.go`、org 層と CLI の test、skill の 4 面、recipe の 2 面、README・仕様・`templates/base/ralph.toml` の変更行、登録簿の追加 3 行と書き換え 1 行)と、plan の 2 コミット分
- 読んだ周辺: `checkDirectorStart` と `StartDirector` の doc と本体、`permissionArgsForDriver` の codex の枝、`codexWritableRootArgs` の全体、`withMainWorktreeDirectorStartConfig` と `loadMainWorktreeOrgConfig`、`orgWatchChildAgmsgHome` の全体、`driver.ResolveAgmsgHome` と `expandTilde`、`codexSeatModesPossible` と `codexModelPermittedForRole`、`driver.Herdr` の `TabCreate`・`AgentStart`、`scripts/ralph-worktree.sh` の `git worktree add`、`/org` skill の permission の項、cycle 2 の triage report の冒頭と ACTION_REQUIRED の表、cycle 2 の verify・test の報告の AC6 と permission の行
- 確かめたこと
  - cycle 2 の `C2-L1`〜`C2-L5` は HEAD でも閉じたまま(`{{CONFIG_LINE}}` への改名、`director.md` の stop の行、Go のコメントに `cross-review cycle N finding` と `/test gap` は残っていない)。この cycle の直しは、これらを戻していない
  - D12: `mode == PermissionModeAutonomous || mode == PermissionModeEdits` の断りは `permissionArgsForDriver` の前にあり(`director_start.go:541`)、`codex_verified = false` のときも新しい理由が先に出る。断りは `checkDirectorStart` の中なので、herdr も `director.jsonl` も触る前。`codexWritableRootArgs` の呼び出しは `spawn.go:898` の 1 か所だけになり、director の枝は消えた。`permissionArgsForDriver` の codex の autonomous と edits の枝は Spawn が今も使うので、死んだコードにならない。test が否定で引く `requires [org.permissions].codex_verified=true` の文字列は `permissions.go:127`・`:132` にあり、空の assertion ではない
  - D13: `agmsgHomeIsRelative` は `home != "" && !filepath.IsAbs(home) && !strings.HasPrefix(home, "~/")`。`orgWatchChildAgmsgHome` の元の条件 `home == "" || IsAbs || HasPrefix` の否定と同じで、抜き出しで挙動は変わらない。bare の `~` は `expandTilde`(`agmsg.go:137-140`)が展開しないので、「相対に数える」の doc は合う。`checkStartAgmsgHome` は `cmd.Flags().Changed("config")` を、兄弟の `orgWatchChildAgmsgHome` と同じ形で使う
  - 登録簿の新しい行 259・260 の主張: `TabCreate`・`AgentStart` の引数に環境変数はない(`herdr.go:153`・`:186`)、Spawn も同じ `AgentStart` を打つ(`spawn.go:1784`)、`git worktree add` は `scripts/ralph-worktree.sh:315`。2 行とも `|` で分けて 5 列
  - 戻すと赤くなるか(読んで判断した。実行はしていない): CLI の相対 agmsg_home の test は、実際の watch gate と必ず失敗する launcher を使うので、`checkStartAgmsgHome` を gate の後ろへ動かすと別のエラーになり赤くなる。`--config` なしの test は、`Changed("config")` の条件を外すと main worktree の相対の値が断られて赤くなる。`TestCodexWritableRootArgs` の director の 6 行は、director の枝を戻すと autonomous と edits の 2 行が赤くなる。org 層の断りの test は、`errors.Is(err, ErrDirectorCodexWorkspaceWrite)` と文の 7 つの断片と herdr の呼び出しが 0 であることを見る
  - 順序: `git log --format='%h parent=%p'` で、4b721b18(plan)→ caf1d025(コード)→ 749ea59e(記録)の一直線。AC15 は 4b721b18 では `[ ]`、749ea59e で `[x]`。cycle 2 の verify・test・sync-docs・cross-review の報告は、どれも caf1d025 の前の tree を述べている
  - skill の 4 面、recipe の 2 面は `cmp` で同一。`git diff --check d2971e1a..HEAD` は空。追加行に TODO・debug 出力・`/Users/`・U+FFFD はない
  - 指摘にしなかったもの: `ErrDirectorCodexWorkspaceWrite` の文が長い(1 行に 4 つの節)ことは、同じ文が help・skill・template・recipe にあっても、この repo の他の断りと同じ作り。`warning: director start: ...` の前置きは `org_director.go:143` などと揃う。警告が、そのあと `StartDirector` に断られる start でも出ることは、環境変数が director に届かない事実の通知で、害はない。`withMainWorktreeDirectorStartConfig` の doc が `codex_verified` を取る key に数えたままなのは、`Permissions` の構造体ごと写すので偽ではない
- 確かめていないもの: テスト、`gofmt`、`go vet`、go1.25.8 でのビルド(新しい test が通ることは、読んだだけで実行していない)。guarded の codex director が、人の codex の設定(`danger-full-access`)のもとで `ralph org start --plan` まで通ること(verify の evidence は、guarded の codex director が起動して承認なしで動くところまで)。herdr の pane が start の `RALPH_ORG_AGMSG_HOME` を受け取らないことの直接の確認(evidence が `CLAUDE_CONFIG_DIR` を変えるために専用の herdr server を作り直しているので、server の環境が pane に出る話とは合う)

### Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C3-L1 | LOW | maintainability | 追加した登録簿の 2 行が、cross-review の triage report を位置で指す。codex の director の行(259 行)は Related が `(## ACTION_REQUIRED, row 1)`、Why が「cycle 2, finding 1」。agmsg_home の行(260 行)は Related が `(## ACTION_REQUIRED, row 2)`。triage は `/cross-review` が cycle ごとに同じ path へ書き換える(ヘッダは今 `Cycle: 2/2 (cap reached)`)ので、cycle 3 の `/cross-review` のあと row 1・row 2 は別の指摘を指しうる。plan の Progress(196 行)は `d2971e1a` と SHA で指しており、こちらは安定している。Go のコメントと test が引く D12・D13・AC6・AC15 は plan の ID で安定している。8a-1 の C2-L3 と、この PR の C2-L4 と同じ型で、cycle 2 は番号を言葉に直して閉じた | `docs/tech-debt/README.md:259`・`:260`(Related と Why の cell)。`docs/reports/cross-review-triage-org-director-start.md` のヘッダの `Cycle: 2/2`。`docs/plans/active/2026-10-15-org-director-start.md:196`(SHA で指す)・`:111`・`:112`(「cross-review cycle 2 の指摘 1・2」と位置で書く。Progress の SHA があるので辿れる) | 2 行の Related を「`docs/reports/cross-review-triage-org-director-start.md` の `d2971e1a` 時点の row 1(`git show d2971e1a:<path>`)」にするか、plan の D12・D13 だけにし、Why の「cycle 2, finding 1」は消す。docs の編集なので、`/cross-review` が triage を書き換える前に、cycle 3 の `/sync-docs` で直せる |
| C3-L2 | LOW | readability | D13 の断りは、director と見張りが違う場所から解くと言い切るが、検査は 2 つの directory を比べない。違いが出ないのは 2 つの場合で、start を main worktree のルートから打つとき(director の cwd と start の cwd が同じ)と、台帳に main worktree がないとき(どちらも打った場所)。この 2 つでも start は断り、エラー文は「the director runs in the ledger's main worktree root … while the watch resolves it from this directory, so the two would use different agmsg installs」と言う。`(when the ledger has one)` の但し書きがあっても、後半の言い切りは偽になる。skill は同じ言い切りを書いたすぐ下の項で「台帳が main worktree のものでなければ、start を打った場所で動く」と言うので、skill の 2 つの項が食い違う。D13 は「相対なら断る」を単純な規則として選んだ(plan 112 行)ので、規則は欠陥ではなく、文が規則より強い | `internal/cli/org_director_headless.go:248-251`(検査とエラー文)・`:77-80`(help)・`:234-246`(doc)。`.claude/skills/director/SKILL.md:151-156`(言い切り)と `:157-158`(director の cwd)。`README.md:258` の同じ 1 文。test が断る行は相対の 3 つ(`agmsg/home`・`./agmsg`・`~`)で、main root から打つ case も、台帳に main worktree がない case も持たない | 文を「別の場所から解きうる(start を main worktree のルート以外から打つ、または …)ので、どこから打っても同じに断る」の形にするか、断る条件に 2 つの cwd の比較を入れる。前者なら、`checkStartAgmsgHome` の doc に「規則は start を打つ場所によらない」と 1 文足せば、unconditional が意図だと分かる |
| C3-L3 | LOW | readability | D12 の文が、もとは続いていた 2 つの文のあいだに入っている。(a) help: 新しい codex の文(`:54-62`、`warning.` の直後から)が、model の文と `Without --config, those settings and …`(`:63`)のあいだにあるので、`those settings` が直前の `~/.codex/config.toml` と読める。同じ commit の `templates/base/ralph.toml` は、同じ位置で `Without it` を `Without a director entry` に直した(108 行)のに、help は直していない。(b) recipe: director の文(55-59 行)が、「`db` ディレクトリ自体を挙げる。広い root は効かない。`.git` などは read-only」と「`AGMSG_STORAGE_PATH` があるなら、その directory を挙げる」のあいだに入り、行末が `If` で切れる。この bullet の主題は agmsg の db の writable root。director が `--add-dir` を持たない話は、同じ recipe の `## The org ledger and --add-dir`(82 行)が「Only the leader gets it」と言う所に自然に入る。2 面(root と `templates/base`)で同じ。あわせて、挿入のあとに折り返しが直っていない comment が 2 つ(`startDirector` の doc の 1 行 31 桁、`DirectorStartParams` の doc の 1 行 56 桁)。どれも読みにくいだけで、偽の文はない | `internal/cli/org_director_headless.go:54-63`・`:176`。`templates/base/ralph.toml:108`。`docs/recipes/codex-seat-permissions.md:55-61`(と `templates/base/docs/recipes/codex-seat-permissions.md`)・`:82-100`。`internal/org/director_start.go:377` | help は `Without --config` の前に `Settings:` のような言葉を足すか、codex の段落を `Without --config …` の段落のあとへ動かす。recipe は 55-59 行を `## The org ledger and --add-dir` の「Only the leader gets it」の段落へ動かし、`If AGMSG_STORAGE_PATH …` を `recursively.` の直後へ戻す。recipe は docs なので `/sync-docs` で直せる |
| C3-L4 | LOW | maintainability | 「autonomous と edits は codex を `--sandbox workspace-write` で動かす」という mode の集合が、4 か所に別々の式で書かれている(`director_start.go:541`、`permissions.go:174`、`permissions.go:123`・`:128` の switch、`doctor_codex_writable_root.go:339`)。このうち `codexSeatModesPossible` は `[org.permissions.roles]` を座席の role と同じに数える(352-356 行)ので、`director = "edits"` の 1 つの role だけが workspace-write の mode のとき(`codex_verified = true`、既定 guarded、codex の model が pool にある)、agmsg の store を覆う writable root がなければ、`ralph doctor` は「codex の座席が workspace-write で動く」として writable root を warn するが、その director は start に断られ、ほかに workspace-write の座席はない。狭い入力で、害は doctor の偽の warn 1 つ。plan の Progress(197 行)は、この点を /sync-docs に渡すと書くが、/sync-docs は文書だけを直すので、コードの点は受け皿がない | `internal/cli/doctor_codex_writable_root.go:336-357`(`apply(mode)` が `Permissions.Roles` の全 role に、`:339` が集合)。`internal/org/director_start.go:541`。`internal/org/permissions.go:174`・`:123-133`。`docs/plans/active/2026-10-15-org-director-start.md:197`。guarded の director と `sandbox_mode = "workspace-write"` の組は、doctor の理由 (b) が正しく拾う | `codexSeatModesPossible` で role `org.DirectorRole` の mode を、codex の director が通る guarded だけ数える。または登録簿の行にして、集合を 1 つの関数(例 `runsUnderWorkspaceWrite(mode)`)にまとめる Trigger をつける。`/org` skill の director の行と permission の項(1156-1166 行)が D12 に触れない点は文書なので、/sync-docs の担当 |

### Positive notes

- D12 の断りは、`codex_verified` の断りより前に置かれ、同じ入力でも新しい理由が先に出る。`errors.Is` の sentinel と文の 7 つの断片、herdr の呼び出しが 0 で `director.jsonl` がないことを、org 層と CLI の test の両方が見る。`codex_verified = true` と `false` の行が対になっていて、true で通る道を戻すと true の行が、`codex_verified` の断りが先に出る順へ戻すと否定の assertion が赤くなる(読んで判断した)
- director の `--add-dir` の枝は、コード・doc・test・仕様・README の全部から消えている。`codexWritableRootArgs` の呼び出し元は Spawn の 1 か所で、`TestCodexWritableRootArgs` の director の行は nil に反転して、枝を戻すと赤くなる。`watch.go:1212` の「leader 以外の codex の座席は --add-dir を持たない」と、仕様の「leader の座席にだけ --add-dir」(182 行)が、直しで再び本当になった
- `agmsgHomeIsRelative` の抜き出しは、見張りの子が使う条件と断りの条件を 1 つの関数にしていて、片方だけが変わることがない。CLI の test は、実際の watch gate・失敗する launcher・`stdout == "" && stderr == ""`・台帳の directory が空であることまで見るので、「何も書かない、herdr を打たない、見張りを起動しない、model の fallback の警告も出さない」の 4 つが 1 つの test に入っている。`--config` なしで相対の値が断られないことも、別の test が止める
- 新しい文書の主張が、コードで確かめられる形で書かれている。登録簿の行 260 は `TabCreate`・`AgentStart`・`Spawn` の関数名で指し、行番号を使っていない。guarded の codex director が人の設定に従うことの限界(`workspace-write` だと同じ壁に当たる)は、help・skill・template・登録簿に同じ言葉で書かれ、start が検出しないことまで言っている
- 登録簿の既存の行(251 行)は、D12 に合わせて「codex の director は `codex_verified` が要る」が「guarded だけ」に書き換えられ、新しい行(259 行)を指している。D12 が古くした行はほかに見つからなかった(`agmsg_home` と `--add-dir` と director を含む行を grep した)

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small leftovers of the org-director-start PR's cross-review cycle-2 fix (self-review cycle 3, C3-L2 to C3-L4). The D13 refusal of a relative `[org].agmsg_home` in a `--config` is worded (start's error, `--help`, the `/director` skill, the README) as if the director and the watch always resolve it from different directories, but `checkStartAgmsgHome` never compares them, so start also refuses when it runs in the ledger's main worktree root or the ledger has none, and the skill's next bullet says the director then runs where start was typed. The D12 inserts land between sentences that belonged together: the `--help` text of `director start` (`Without --config, those settings` follows the codex block) and the `db` writable-root bullet of `docs/recipes/codex-seat-permissions.md` (the director sentence splits it, with a dangling `If`; both faces). The set of modes that run a codex seat under `--sandbox workspace-write` (autonomous, edits) is spelled in four places, and `codexSeatModesPossible` in `internal/cli/doctor_codex_writable_root.go` counts role director like a seat role, although start refuses it for codex | No behavior of a start that works changes. A relative `agmsg_home` with `--config` is refused in the two cases where nothing would diverge. `ralph doctor` warns about an agmsg writable root that no codex seat needs only when `codex_verified = true`, a codex model is in the pool, no writable root covers the store, the default mode is guarded, and role director alone is set to edits or autonomous | Cap 3 was raised for this run and cycle 3 is its last pipeline run, so each of these wording and placement edits would need another full run; none changes what start does | The next edit of `checkStartAgmsgHome` or of the `director start` help (C3-L2, C3-L3); the next edit of the codex seat-permission recipe (C3-L3); the next change to `codexSeatModesPossible` or to the modes that run under workspace-write, then give them one function (C3-L4) | `docs/plans/active/2026-10-15-org-director-start.md` (D12, D13); `docs/reports/self-review-2026-10-15-org-director-start.md` (`## Cycle 3`: C3-L2 to C3-L4; the report is appended to, not rewritten, so the IDs stay) |

_(登録簿には追記していない。上の行は、C3-L2〜C3-L4 を直さずに進めるときに /sync-docs が 1 行で足す形。C3-L1 は登録簿の 2 行の pointer なので、行には入れず、/sync-docs が 2 行の Related を直す。recipe の並び替え(C3-L3 の b)も文書なので、/sync-docs で直せるなら行の文から外す。)_

### Coverage gaps

- 断りの規則が start を打つ場所によらないことを固定する test がない。main worktree のルートから打つ case と、台帳に main worktree がない case で、相対の agmsg_home が断られること(C3-L2 で文を「どこから打っても断る」にするなら、この 2 つが規則の根拠になる)
- guarded の codex director が、人の codex の設定が書き込みと network を許す環境で、`ralph org start --plan` まで通ること。verify の evidence は guarded の director が起動して動くところまでで、D12 の「道は 2 つ」の 1 つ目の実機の確認ではない
- herdr の pane が start の `RALPH_ORG_AGMSG_HOME` を受け取らないことの直接の確認。D13 の警告と登録簿の行 260 の前提で、server の環境が pane に出る話(evidence の `CLAUDE_CONFIG_DIR` のための server の作り直し)とは合うが、この変数での確認ではない
- `ralph doctor` の `codexSeatModesPossible` に、role director だけが workspace-write の mode の case がない(C3-L4)

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。LOW 4 件は、登録簿の pointer、断りの文の言い切り、挿入した文の位置、mode の集合の重複と doctor の偽の warn で、D12 の codex の断りと D13 の相対 agmsg_home の断りの動きに欠陥は見つからなかった)
- Follow-ups:
  1. C3-L1 と C3-L3 の recipe の並び替えは文書なので、cycle 3 の `/sync-docs` が、`/cross-review` が triage を書き換える前に直す。登録簿の 259・260 行の Related を SHA か plan の ID にする
  2. C3-L2・C3-L3 の help・C3-L4 は Go に触れるので、直すなら pipeline をもう 1 回回すことになる。cap 3 の最後の回なので、直さずに進むなら、上の Tech debt の行を `/sync-docs` が登録簿に足す(C3-L1 と recipe の分は行から外す)
  3. /verify への引き継ぎ: AC6 の tick は、4b721b18 が文を変えたときも `[x]` のままで、満たすのは caf1d025。HEAD で読み直す。AC15 の順序は正しい。cycle 2 の verify(AC6 の行は `codex_verified = false` の断りと director の `codexWritableRootArgs` を根拠にしている)・test(permission の行が director の `--add-dir` を挙げる)の報告は、caf1d025 の前の tree を述べているので、cycle 3 の各 phase が読み直す
  4. plan の Progress(197 行)が /sync-docs に渡した 2 点のうち、`/org` skill の director の行と permission の項は文書の直し。`ralph doctor` の点は C3-L4 の行で受ける
- Known gaps: テスト、`gofmt`、`go vet`、go1.25.8 でのビルドは確かめていない。新しい test が通ることと、「戻すと赤くなる」は、読んで判断したもので実行していない。本物の herdr での guarded の codex director の動きと、pane の環境変数は見ていない

## Cycle 4

- Date: 2026-10-16
- Plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest ace92dd0aaa7。cross-review cycle 3 のあと D14・D15・AC16・S10 を足し、AC9 に 1 文足して承認し直した)
- Reviewer: reviewer subagent (Claude)。パイプライン 4 回目(cycle 4)。`RALPH_STANDARD_MAX_PIPELINE_CYCLES` を 4 に上げた回で、この PR の最後の回。`cycle-count.json` は 2 のままなので、insight event は `--cycle 4` を明示して足した
- Scope: diff の品質だけ。cross-review cycle 3 の ACTION_REQUIRED 2 件の直し。`git diff 6eef89e8..HEAD`(HEAD 94cc1b7f、3 コミット、13 ファイル、+704/-36)を読んだ。8a0356de(plan に D14・D15・AC16・S10)、b888d637(コードと skill の 4 面)、94cc1b7f(plan の記録)。全体の差分は、直しが前のコードに触れる所(start の `abort`、`--force`、CLI の stop のエラー)だけ読んだ。この節の ID には `C4-` を付け、前の cycle の ID と取り違えないようにする。テストの実行、静的解析、仕様適合、文書のずれの検査は /test・/verify・/sync-docs の担当なので行っていない

### Evidence reviewed

- 読んだ差分: b888d637 の全 hunk(`director_start.go`、`director_stop.go`、`org_director_headless.go`、org 層と CLI の test 4 ファイル、`skill_frontmatter_test.go` の全体、skill の 4 面)と、plan の 2 コミット
- 読んだ周辺: `markStopped` の全体、`recordHerdrID`(`director_start.go:187-226`)、start の `abort`(`:714-735`)、`confirmDirectorResources`・`closeConfirmedDirectorResources`・`closeDirectorPane`・`closeDirectorWorkspace`・`newDirectorCleanupError`、`StopDirector` の全体、`openHeadlessRegistration`、CLI の `stopError`・`printOrgDirectorStop`・`orgDirectorStopNotes`、書き手の名前を並べた `director.go` の doc 3 か所
- 確かめたこと
  - D14 の根拠: `recordHerdrID` は記録済みの id を断る(`director_start.go:218-220`)ので、id は空から値にしか変わらず、`RecordWorkspace` と `RecordPane` の書き込み口はこの 1 つ(`:176`・`:182`)。「等しければ記録されていない」は成り立つ。`launched` を比べないのも、start は pane を記録してからでないと `launched` を書かないので、id が動かない `launched` だけの変化は stop が閉じるものに影響しない
  - 比べない側: start の `abort` は `MarkStopped`(`director_start.go:718`)のままで、`ErrDirectorStopped` を成功に扱う。`MarkForceStopped` も比べない。`abort` が `MarkStoppedIfUnchanged(r.reg)` だったら、記録に失敗した id を `r.reg` だけが持つので比較が食い違い、stop を書けない。`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade` に足したコメントの「比べない」はこの理由で成り立つ
  - `recordDirectorStop` の分岐: forced の枝は `MarkForceStopped` なので `ErrDirectorChangedDuringStop` を返さない。changed の枝は `ErrDirectorNoOpenHeadless` を巻かず(org 層の test が `!errors.Is` で見る)、`HerdrCommands` は閉じ終えているので空になり、CLI の `--force` の案内(`len(cleanup.HerdrCommands) > 0` が条件)は出ない。CLI の test は force の true と false の両方で同じ文を固定している
  - 戻すと赤くなるか(読んで判断した。実行はしていない): `markStopped` の比較を外すと、store の test の拒否 3 行と、`TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` の 3 行が赤くなる。`recordDirectorStop` の非 forced の枝を `MarkStopped` に戻すと `TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` が赤く、forced の枝を `MarkStoppedIfUnchanged` に替えると `TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile`(読みのあとに pane を記録する)が赤くなる。store の test は「launched だけが読みのあとに来たら記録される」行も持つ
  - D15 の主張を Ruby の `YAML.safe_load`(Psych)で再現した。`a: x: y` 型は `mapping values are not allowed`(列は test の期待どおり、`:\t` と末尾の `:` も同じ)、`a: use C #2` は `{"a"=>"use C"}` で後ろが落ち、直したあとの description は読める。test の限界(C4-L3)も同じ probe で確かめた
  - skill の 4 面は `cmp` で同一(`.claude` と template の `.claude`、`.agents` と template の `.agents`。`.agents` と `.claude` は frontmatter が同じ)。tracked の `SKILL.md` は 54 個で、`.claude/skills` と `.agents/skills` が 14 個ずつ、template の 2 つが 13 個ずつ。test の `skillRoots` の 4 つに収まり、コメントの「54」は合う
  - `git diff --check 6eef89e8..HEAD` は空。追加行に TODO・debug 出力・`/Users/`・U+FFFD はない。Go のコメントが引くのは plan の ID(D14・D15・AC16)で、triage の行や指摘の番号を引く所はない。長い追加コメント 2 本(CLI test の 1 行 96 桁、frontmatter test の 1 行 87 桁)は識別子を折れないだけなので指摘にしない
  - 順序: `git log --format='%h parent=%p'` で 6eef89e8 → 8a0356de → b888d637 → 94cc1b7f の一直線。AC16 は 8a0356de と b888d637 で `[ ]`、94cc1b7f で `[x]`(正しい)。AC9 は 8a0356de が最後の 1 文を足したときも `[x]` のままで、その文を満たすのは b888d637(下の引き継ぎ)
- 指摘にしなかったもの: store の helper `recordForTest`(`director_test.go:1725`)が、同じファイルの `mustAttachHeadless`・`mustReadDirector` の `must*` の名前に揃っていないことと、文字列の step を取ること。`recordPaneAndLaunched`(`director_stop_test.go:783`)と役割が近いが、id の固定と `Errorf` と `Fatalf` の違いに理由があり、同じファイルに `MarkLaunched` を直に打つ所がほかに複数あるので、helper の乱立とまでは言えない。`t.Skipf` の入口(`.claude/skills` がなければ skip)は、`defaults_sync_test.go` の前例と同じで、ほかの 3 つの root は欠けるとエラーになる。`workspaceCloseHook` が実際の close のあとに走ること(doc は「answers の前」)は、close が終わってから start の記録を差し込む形で足りている
- 確かめていないもの: テスト、`gofmt`、`go vet`、go1.25.8 でのビルド。新しい test が通ること。実機の herdr で stop と start が重なったときの動き(test は偽の herdr の hook で差し込む)

### Findings

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C4-L1 | LOW | maintainability | 書き手の名前を並べた doc が、新しい公開の書き手 `MarkStoppedIfUnchanged` を入れずに残っている。`ErrDirectorWriteNotRestored` の doc は「every DirectorStore writer (Attach, Detach, AttachHeadless, RecordWorkspace, RecordPane, MarkLaunched, MarkStopped, and MarkForceStopped)」と言い切るが、`MarkStoppedIfUnchanged` も同じ `markStopped` を通り、同じ書き込みの失敗を返す。`ErrDirectorNoOpenHeadless` の doc(「returned by RecordWorkspace, RecordPane, MarkStopped, and MarkForceStopped」)、`DirectorWho` の doc(「later events (RecordWorkspace, RecordPane, MarkLaunched, MarkStopped, MarkForceStopped)」、`MarkStoppedIfUnchanged` も `who` を取る)も同じ。`openHeadlessRegistration` の doc は「read without the lock (MarkStopped decides again under it)」のままだが、stop が決め直すのは今は `MarkStoppedIfUnchanged` と `MarkForceStopped` で、`MarkStopped` は start の `abort` だけが呼ぶ。同じ commit が `MarkStopped`・`MarkForceStopped`・`StopDirector`・`recordDirectorStop` の doc は直しているので、直し残しの 4 か所が目立つ | `internal/org/director.go:95-99`・`:115-118`、`internal/org/director_start.go:39-42`、`internal/org/director_stop.go:203-207`。`MarkStoppedIfUnchanged` は `director_start.go:316`、呼び出しは `director_stop.go:182` の 1 か所 | 4 か所に `MarkStoppedIfUnchanged` を足す(`openHeadlessRegistration` は「`MarkStoppedIfUnchanged` and `MarkForceStopped` decide again under it」)。名前の列挙をやめ、「every DirectorStore writer」の括弧を外す形でもよい。Go のコメントだけの直しだが、Go に触れるので pipeline を回し直す。最後の cycle なので、直さないなら下の登録簿の 1 行に入れる |
| C4-L2 | LOW | readability | stop が「start が記録を足した」と断るときの出力が、同じ 3 つの note を 2 回出し、文が長い。CLI の `stopError` は `*org.DirectorCleanupError` の枝で先に `stop d1: ctrl_c=skipped, pane=none recorded, workspace=closed` を stderr に書く(note は `inboxField` で escape する)。org 層のエラーの文も `this stop closed only what it had read (ctrl_c=..., pane=..., workspace=...)` と同じ 3 つを、escape せずに埋める。ほかの stop の失敗は note を stderr にしか出さない(`closed.err` の文は note を含まない)。`ctrl_c=failed: <herdr のエラー文>` は `closed.err == nil` でも note に入るので、この文だけ herdr の文字列が escape なしで端末に出る(close の失敗の文は herdr の文字列を escape なしで持つ前例があるので、害は小さい)。文の前半は `org: director: stop d1: record d1 stopped: org: director: stop d1: a start in progress recorded ...` と前置きが 2 回続き(既存の `record %s stopped: %w` の枝と同じ作り)、`stop read` が 2 回出る。CLI の test は stderr の行とエラーの文の両方を固定しているので、重複は意図した形だが、読む人にはどちらも同じ情報 | `internal/org/director_stop.go:189-191`、`internal/cli/org_director_headless.go:353`・`:412-414`、`internal/cli/org_director_headless_test.go`(新しい test の `want` と `stderr` の assertion)、`internal/org/director_stop_test.go` の `changedDuringStopText` | エラーの文から note の括弧を外し、「this stop closed only what it had read」だけにする(note は stderr の行が出す)。または stderr の行を出さず、エラーの文だけにする。どちらでも `changedDuringStopText` と CLI の `want` の 2 か所の期待文字列が変わる |
| C4-L3 | LOW | readability | `skill_frontmatter_test.go` の説明が、それを要る場所に付いていない。(a) 30 行の説明が `package` と `import` のあいだに浮いていて、`TestSkillFrontmatter_PlainValuesReadAsTheySay`(`:136`)自体には doc がない。前例の `defaults_sync_test.go` も説明を `import` の前に置くが、関数には 1 行の doc(`// TestDefaultsLockStep is the main tripwire.`)を持つ。(b) 名前は「plain な値は書いたとおりに読める」と約束するが、検査は 2 つの形(「コロンと空白」「空白と `#`」)だけで、説明の「Other YAML errors pass it」が限界を言う。Psych で試すと、値が backtick・`[`・`*` で始まる行(`a: \`x\` y`、`a: [x y`、`a: *x`)は YAML として失敗するが、この test は通す。D15 は「今回の型だけを止める」と決めているので欠陥ではなく、失敗の文(`:179`)も限界を言わない。値が backtick で始まる description は書きうる | `internal/config/skill_frontmatter_test.go:3-28`(説明)、`:136`(関数)、`:16`(限界の 1 文)、`:179`(失敗の文) | 関数に 1 行の doc を付けて、説明の要点(2 つの形だけ、限界、直し方)をそこへ移す。限界は、失敗の文ではなく関数の doc に 1 文足せば足りる。範囲を広げるなら、plain scalar の先頭に置けない文字(backtick・`[`・`{`・`*`・`&`・`!`・`%`・`@`)も形に足す。これは D15 を変えるので、別の PR か登録簿の 1 行 |
| C4-L4 | LOW | maintainability | plan の Progress が、これから /sync-docs が書き換える文書を行番号で指す。S10 の行(`:210`)は「`/director` skill の「終える」5 番、README 258 行、仕様 85 行が、stop のこの断りを書いていない」と言い、S9 の行(`:208`)も仕様 85 行を引く。README の 258 行は今は director の段落だが、/sync-docs がその段落に stop の断りを足すと、行は動く。plan は /pr で archive され、Progress が最後に読まれる記録になる。D12〜D15 は cross-review の指摘を「cycle 3 の指摘 2」のように番号で言うが、triage は `/cross-review` が同じ path へ書き換える(cycle 4 のあと、番号は別の指摘を指す)。D の文には何の指摘かが同じ文の中にあるので害は小さいが、Progress の行は SHA(`6eef89e8`・`b888d637`)で指しており、そちらが安定している | `docs/plans/active/2026-10-15-org-director-start.md:208`・`:210`(行番号)、`:46`・`:115-118`(「指摘 N」) | Progress の 2 行は、行番号を「README の director の段落」「仕様 FR-9 の注」のように見出しか段落で言い換える(Progress の行は digest の外なので、承認し直しは要らない)。D12〜D15 の「指摘 N」は digest に入る文なので、直すなら承認し直しになる。直さずに archive するのでよい |

### Positive notes

- D14 の比較は、判定を `director.lock` の下の `markStopped` に 1 つだけ置き、stop の呼び出し(`MarkStoppedIfUnchanged`)と start の呼び出し(`MarkStopped`)と forced の呼び出し(`MarkForceStopped`)が、同じ本体に `read` の有無だけで分かれる。比べる対象(workspace と pane の id)、比べない対象(`launched`)、比べない呼び出し側(start の後始末、forced)が、それぞれ理由つきで doc に書かれ、store の test と stop の test が両側を固定している。start の後始末が比べないことは、`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade` のコメントに足されたので、将来 `MarkStopped` を `MarkStoppedIfUnchanged` に統一する人が理由を見つけられる
- エラーの型の扱いが既存の形に揃っている。新しい sentinel は `errors.Is` で `ErrDirectorNoOpenHeadless` と別に判別でき、CLI は新しい分岐を足さず、`*org.DirectorCleanupError` の枝で、`--force` の案内の条件(`HerdrCommands` が空)だけで出し分けている
- test が競合を作る仕方が、sleep や goroutine を使わず、偽の herdr の `WorkspaceClose` の中と `Getenv` の中に start の記録を差し込む形で、順序が決まっている。「stop がもう一度打たれると記録される」まで、同じ test が見る
- D15 は、直した description そのもの(155 桁目の `ways: the`)を test の row に残し、各 row を Psych で確かめたと書き、限界(入れ子と block の行は見ない)を doc に言っている。検査の入口が `skillRoots` を歩く形で、`git` を使わず CI でも同じに動く

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(登録簿には追記していない。この cycle は `RALPH_STANDARD_MAX_PIPELINE_CYCLES` の最後の回なので、C4-L1〜C4-L3 を直さずに進めるなら、/sync-docs が次の 1 行を足す。C4-L4 は plan の Progress の文なので行に入れない。)_

- 足す行の案: 「org-director-start の cross-review cycle 3 の直し(plan の D14・D15、commit b888d637)の Go の小さな直し残し。(a) `ErrDirectorNoOpenHeadless`・`ErrDirectorWriteNotRestored`・`DirectorWho`・`openHeadlessRegistration` の doc が書き手の名前を並べ、新しい `MarkStoppedIfUnchanged` を入れていない。(b) stop が start の記録を理由に断るときの出力が、同じ 3 つの note を stderr の行とエラーの文の両方に出す。(c) `skill_frontmatter_test.go` の説明が関数の外に浮いていて、検査しない YAML の形(値の先頭の backtick・`[`・`*` など)を失敗の文が言わない」。Trigger は「`internal/org/director.go` の書き手の doc を触るとき」「`ralph org director stop` の出力を変えるとき」「SKILL.md に新しい YAML の形の失敗が出たとき」。行は 5 列、`|` を文中に使わない。Related は plan の ID と SHA で書き、報告の行や番号では指さない

### Coverage gaps

- stop が start と重なる道を CLI から通す test がない。CLI の test は、org 層が返すエラーを `stopError` に渡す形で、重なりそのものは org 層の test が偽の herdr の hook で作る(test のコメントがそう書いている)
- start の `abort` が `MarkStopped` を使い続けることを、新しい専用の test は持たない。`ARecordThatFailsStillClosesWhatHerdrMade` が間接に見る(読んで判断した。`MarkStoppedIfUnchanged` に替えると赤くなるはず)
- `ctrl_c=failed` の note を持つ stop が、start の記録に重なって断られる道の出力(C4-L2)を固定する test はない

### Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。LOW 4 件は、新しい書き手を入れ忘れた doc の列挙、断りの出力の重複、test の説明の置き場所、plan の Progress の指し方で、stop が動いている director を stopped と記録する経路と、skill の frontmatter が YAML として読めない問題は、コードと test の両方で閉じている)
- Follow-ups:
  1. C4-L1〜C4-L3 は Go に触れるので、直すなら pipeline をもう 1 回回すことになる。cap 4 の最後の回なので、直さずに進むなら、上の登録簿の 1 行を /sync-docs が足す。この 3 件は Related を plan の ID(D14・D15)と commit(`b888d637`)で書き、報告の行や ID の番号では指さない
  2. C4-L4 は plan の Progress の言い換えで、digest の外。/sync-docs が README と仕様を直すのと同じ回に、行番号を見出しの言い方に直す
  3. /verify への引き継ぎ: AC9 の `[x]` は、8a0356de が最後の 1 文を足したときも付いたままで、その文を満たすのは b888d637(AC6 のときと同じ形)。HEAD で読み直す。AC16 の順序は正しい。cycle 3 の verify・test の報告は b888d637 の前の tree を述べているので、cycle 4 の各 phase が読み直す
  4. plan の Progress(S10 の行)が /sync-docs に渡した 1 点(`/director` skill の「終える」の項、README、仕様が stop のこの断りを書いていない)は文書の直しで、stop の `--help` は b888d637 で書き終えている
- Known gaps: テスト、`gofmt`、`go vet`、go1.25.8 でのビルドは確かめていない。「戻すと赤くなる」は読んで判断したもので実行していない。Ruby の `YAML.safe_load` で YAML の主張を確かめたほかは、コマンドを実行していない。実機の herdr での stop と start の重なりは見ていない
