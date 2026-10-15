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
