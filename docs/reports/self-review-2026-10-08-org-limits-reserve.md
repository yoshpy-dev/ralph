# Self-review report: org-limits-reserve

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Branch: feat/org-limits-reserve(HEAD b5a2ecc6、base 51855166)
- Reviewer: reviewer subagent (Claude)、パイプライン 1 回目(cycle 1)
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、typo、null 安全、デバッグ用コード、秘密情報、例外処理、安全性、保守性、コメントとヘルプ文の正確さ)。対象は `git diff 51855166...HEAD` の 28 ファイル(+2880/-74)。コードは 620458d7(設定)、11fc2261 と 8fe95acd(org の層)、78e46f36(CLI)、文書は fd3e3b47、残りは計画。重点は、`spawn.go` のロックの中の判定の順序と `rejected` の書き分け、idempotent の経路が既存の座席の状態を変えないこと、`reserve.go` のパスの規則と重なりと「走っている org」の導出、`verbs.go` の補償(`reserveAgain`)、CLI の全体の上限の読み元、doc comment・help・`/org` skill とコードの一致。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査、formatter、`check-sync.sh` 類は実行していない

## Evidence reviewed

- `git diff 51855166...HEAD` の非テストのコードを全行読んだ(`reserve.go`、`spawn.go`、`envelope.go`、`seat.go`、`verbs.go`、`verbs_all.go`、`statedir.go`、`cli/org.go`、`config.go`、`ralph-config.sh`、`templates/base/ralph.toml`)。テストは `verbs_test.go`、`verbs_all_test.go`、`envelope_test.go`、`statedir_test.go`、`config_test.go`、`watch_test.go` の差分と、`spawn_test.go` の `TestOrgSpawn_Reserve_ExistingLeader` を読み、残りは宣言の一覧と先頭を見た。計画は全行読んだ
- 判定の順序: `spawnCapacityErr`(`spawn.go:1063`)は max_seats、`ValidateOrgWideCapacity`(max_orgs、max_total_seats の順)、`reservationDecision` の順で、本番の経路(`checkCapacityAndStart`、`:1032`)と dry-run(`:500`)が同じ関数を呼ぶ。`rejected` を書く経路は `o.reject` を通る拒否(max_seats、max_orgs、max_total_seats、予約の判定)、書かない経路は入力検査(`:430-441`、書かないと step 0 の doc に明記)と `idempotentRespawn` の拒否(`:1085-1099`、書かない理由が doc にある)で、書き分けは doc comment と合う。`rejected` は状態のイベント、`scope_reserved` は座席 id の空な非状態イベントなので、idempotent の経路が追記する `scope_reserved` は Roster の座席の状態を動かさない(`seat.go:44-50` と `stateEvents`)
- 予約の記録の位置: `checkCapacityAndStart` はロックの中で `scope_reserved`、次に `spawn_started` を追記する。`Spawn` の locked closure は Phase 1 と Phase 2 の両方でここを通り、stale な座席の Phase 2 でも判定をやり直す。2 回目の `idempotentRespawn`(`:760`)は Phase 2 の新しい `events` を渡している
- `reserve.go`: `normalizeReservePath` を `.`、`./`、`a//`、`a/.`、`./.`、`a/..`、`...`、絶対パス、空白だけ、制御文字で読み合わせた(`path.Clean` と `HasSuffix` の組み合わせで、正規化は冪等)。`reservePathsOverlap` は区切りの単位で、ファイルとディレクトリは「ファイルがディレクトリの下にあるか」だけを見る。`currentOrgLives` は最後の `disbanded` 以前を `i <= d` で飛ばし、補償で `disbanded` の後に書いた `org_workspace_created` と `scope_reserved` は数える。`reservationBeforeLastDisband` の `events[:d]` は、`d` より前のもう 1 つ前の `disbanded` から数え直すので、その生涯の最後の予約を返す
- 補償: `CloseDeferredSelfWorkspace`(`verbs.go:1111`)の `reserveAgain` は、今予約があれば何もせず、`disbanded` の前の予約があれば `paths=<paths> restored: <why>` で書く。`reservedPathsFromDetails` は最初の空白で切るので、`restored:` の注記は読み戻しに影響しない。`force` では呼ばれない(`:1100` の `return` が先)
- CLI の読み元: `withMainWorktreeOrgLimits`(`cli/org.go:148`)は `configPath != ""` と `source != "git-main-worktree"` で何も変えず、main の `ralph.toml` が読めなければエラーで止める。`MainWorktreeRoot` は `.harness/state/org` の 3 要素を `filepath.Dir` で戻し、`Join` で元の dir と一致するかを確かめる。`rt.Spawn` の呼び出し元は `cli/org.go:436` と `:558` の 2 つだけで、どちらも `newOrgSpawnRuntime` を通る
- `/org` skill: `.claude/skills/org/SKILL.md` の差分を全行読み、コードと突き合わせた。4 面は `diff -q` で `.claude/skills/org/SKILL.md` と `.agents/skills/org/SKILL.md`、`templates/base/` の 2 面が同じ内容
- 機械的な確認: `git diff --check` は空、追加行に末尾の空白、U+FFFD、`fmt.Print`、`println`、`TODO`、`t.Skip`、`time.Sleep` はない。`scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` は同じ差分。`RALPH_ORG_MAX_ORGS` / `RALPH_ORG_MAX_TOTAL_SEATS` を読む Go 側のコードはなく、`RALPH_ORG_MAX_SEATS` と同じく同期テストのための値

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | scaffold が配る `templates/base/ralph.toml` の説明と、`config.go` の doc comment が「the director is not a seat」と書く。director は `docs/specs/` と計画にしかなく、scaffold した project にも、このリポジトリの出荷物(README、skill、rules、コード)にも存在しない語で、読み手は何のことか分からない。計画の Non-goals も「director はまだなく」と書いていて、まだない機能について「数えない」と断る文になっている。director が入る 8 段目で数え方が変わると、この一文が先に古くなる | `templates/base/ralph.toml:46`、`internal/config/config.go:53`、`:58`。出荷物での `director` の語(単語境界)の grep は、この 3 か所だけ | 3 か所とも「The director is not a seat and is not counted」の文を消す(`max_orgs` は走っている org、`max_total_seats` は動いている座席、で完結している)。director を数える・数えないは、director を入れる段の計画で書く。scaffold の `ralph.toml` なので、`/verify` の前に直せば cycle は増えない |
| F-2 LOW | readability | `spawn.go` の中で `reserve` が 2 つの型を指す。入力検査(`:437`)では `NormalizeReservePaths` の結果の `[]string`、dry-run の分岐(`:500`)、`checkCapacityAndStart`(`:1032`)、`spawnCapacityErr` の戻り値(`:1063`)、`dryRunSpawn` の引数(`:1680`)では「予約を記録するか」の `bool`。隣に `p.Reserve`(`[]string`)もある。`scopeReservedEvent("", p.OrgID, p.Reserve, "", true)`(`:1686`)と `scopeReservedEvent(o.now(), orgID, paths, "restored: "+why, false)`(`verbs.go:1233`)は、文字列 2 つと `bool` を位置で渡すので、呼び出し側だけでは ts と note、dry-run の向きが読めない | `spawn.go:437`、`:500`、`:1032`、`:1063`、`:1680`、`:1686`、`verbs.go:1233`、`reserve.go:151` | `bool` 側を `recordReservation` にする。dry-run の呼び出し(`ts` が空)は、`dryRunSpawn` が `steps[i].TS` を上書きするので、`scopeReservedEvent` の引数から `ts` を外して呼び出し側で足す形でもよい |
| F-3 LOW | readability | autonomous の scope ゲートのエラー文が「requires --scope or --reserve」で、`--reserve` を使えない座席(leader 以外)にも勧める。その座席が従うと、次は「only the leader seat reserves paths ... (spawn it without reserve paths)」で拒否され、2 回目の拒否が 1 回目の案内と食い違う。`autonomousScopeGateErr` の条件は `p.SeatID` を見ない(`Reserve` を渡せるのが leader だけなので、条件としては合っている) | `spawn.go:1189`、`:430-441`、`:1177-1193` | 文を seat によって変えるか、「requires --scope (the leader seat may pass --reserve instead)」のように leader 限定を文に入れる。`autonomousScopeGateErr` は `p` を持っているので、`p.SeatID == LeaderIdentity` で分けられる |
| F-4 LOW | maintainability | 予約だけを渡した autonomous の leader は、scope ゲートを通るのに、役割プロンプトの `{{SCOPE}}` は `p.Scope` だけで描かれ、既定の「未指定(読み取り中心で、リポジトリ規約に従うこと)」になる。ゲートの目的(autonomous の座席が担当範囲を持つこと)を `--reserve` で満たしたことにしながら、座席に範囲が渡らない。計画の進捗の(c)として実装者が見つけて送る予定の項目と同じ。直す場合は `Scope` が空のとき `strings.Join(p.Reserve, ", ")` を渡す 1 行で、予約と `--scope` が一致する | `spawn.go:809`、`:1715`(`Scope: p.Scope`)、`prompts.go:21-26`、`:82`、`internal/org/prompts/leader.md:4`、計画の進捗 156 行 | 今直すか tech-debt に送るかを決める。送るなら、`/org` skill と README の「`--reserve` は `--scope` のゲートを満たす」の隣に、座席のプロンプトには範囲が出ないことを 1 文足す |
| F-5 LOW | exception-handling | `reserveAgain` に渡す `rr.Events` は、`CloseDeferredSelfWorkspace` の冒頭(`:1087` 付近)で読んだ snapshot で、最大 3 回 10 秒の herdr の呼び出しの前のもの。窓の間に同じ org_id を `ralph org start --reserve` で立て直した場合、snapshot は「予約なし」を示し、`reserveAgain` は古い予約を新しい予約の後ろに追記する。`currentOrgLives` は最後の `scope_reserved` を採るので、新しい予約が古いものに入れ替わる。重なりの検査もしない。計画のリスクに書いた窓(ほかの org が枠を取る)とは別の、同じ org_id の組み合わせ。`reactivateSeat` も同じ snapshot を使う既存の形なので、新しく持ち込んだというより同じ形を踏襲している | `verbs.go:1111`(呼び出し)、`:1215-1237`(`reserveAgain`)、`reserve.go:218`(最後が勝つ) | 追記の直前に `o.Manifest.Read()` で読み直し、そのとき `ActiveReservation` が nil の場合だけ書く。直さないなら `reserveAgain` の doc の窓の説明に、同じ org_id の立て直しも含めて書く |
| F-6 LOW | maintainability | 同じ導出が 2 つある。`currentOrgLives` の workspace の畳み込み(`DeleteFunc` してから `Created` なら追記)は `openOrgWorkspaces` と同じ処理を、最後の `disbanded` より後に限って書き直したもの。`orgLife.workspaces` は `len(...) > 0` の判定にしか使われず、「oldest first」の順序も誰も読まない。2 つが食い違うと、`disband --all` の対象(`orgsToDisband` が使う `openOrgWorkspaces`)と max_orgs の「走っている」が別々に動く。いまは `PaneID == ""`、dry-run、org 単位の 3 条件が同じで合っている | `reserve.go:159-164`、`:206-214`、`:241`、`spawn.go:1833-1850` | `currentOrgLives` が org ごとに `events[d+1:]` を `openOrgWorkspaces` へ渡す形にするか、少なくとも `openOrgWorkspaces` の doc に「`currentOrgLives` と同じ規則」と足す。`workspaces` は `[]string` のまま持つ理由がなければ `open bool` でよい |
| F-7 LOW | maintainability | `newOrgSpawnRuntime` は `newOrgRuntime` の 2 手順(`ResolveOrgStateDir`、`guardLegacyOrgStateDir`)をそのまま写している。`newOrgRuntime` に手順が増える(別の guard や警告)と、spawn と start だけが取り残される。写した側の doc は「`newOrgRuntime` と同じ」としか言わず、写しであることが読み手に見えにくい | `cli/org.go:108-114`、`:121-134` | 解決と guard を 1 つの関数(`resolveOrgLedger(cmd, stateDir, access) (dir, source string, err error)` など)に切り出して両方が呼ぶか、`newOrgRuntime` に限度の読み元を差し替える引数を足す |
| F-8 LOW | readability | 読めない `scope_reserved` を「repo 全体」として扱う選択は fail-closed で、doc comment(`reserve.go:183-186`)にも書いてある。ただし読み戻した結果は本物の `.` と区別できない。`ralph org status` は `reserved: .` と出し、`reservationDecision` の拒否は「already reserves ., so a different reservation ... is refused」と言い、`reserveAgain` はそれを `paths=.` の明示の記録として書き直す。利用者から見ると、自分が予約していない repo 全体の予約が現れ、原因の台帳の記録の破損は出ない | `reserve.go:216-219`、`:256-262`、`:289-292`、`verbs.go:1226-1233`、`cli/org.go:1109` | 影響は壊れた記録がある org の中だけなので、直さなくてもよい。直すなら、`ActiveReservation` が「読めなかった」を返し、`status` と拒否の文に「(unreadable scope_reserved record, treated as the whole repo)」を足す |
| F-9 LOW | readability | ディレクトリを末尾の `/` なしで書く(`--reserve internal/auth`)と、規則どおりファイルの予約になり、`internal/auth/token.go` や `internal/auth/` の予約と重ならない。拒否もされず、予約が付いたように見えて何も守らない。規則は計画の AC6 のとおりで、help と skill にも書いてあるが、書き間違えたときに何も知らせない。CLI は `--cwd` を持っているので、パスが存在するディレクトリなら気づける | `reserve.go:90-102`(`default: return a == b`)、`cli/org.go:372-378`(`orgReserveFlagUsage`) | CLI で、`filepath.Join(cwd, path)` が既存のディレクトリで末尾の `/` がないとき、stderr に「`internal/auth` is a directory; write `internal/auth/` to reserve its contents」を 1 行出す(予約の判定は変えない)。出さないなら tech-debt に残す |
| F-10 LOW | maintainability | 補償は workspace の経路(`CloseDeferredSelfWorkspace`)にしか予約を戻さない。`CloseDeferredSelfPane` は `reactivateSeat` だけで、`disbanded` で解けた予約を戻さない(計画の進捗の(a)と同じ)。その経路の org は、座席が動いている扱いに戻って走っている一方、範囲は誰にも守られない。AC15 は workspace の経路に限っているので計画どおりだが、同じ種類の欠けが片方に残る | `verbs.go:1010-1034`(`CloseDeferredSelfPane`)、`:1111`(`CloseDeferredSelfWorkspace` の `reserveAgain`)、計画の進捗 156 行 | 計画どおり tech-debt に送る。`reserveAgain` の doc に、pane の経路は戻さないことを 1 文足すと、次に読む人が経路の非対称を見落とさない |
| F-11 LOW | readability | help 文の 2 点。(1) `--config` の persistent flag の説明に、spawn と start だけに関わる文が入り(「without --config, spawn and start read ...」)、`--config` 自身の説明の中で「without --config」と言うので読みにくい。`stop` や `status` の help にも出る。(2) `orgWideLimitsHelp` は「Before starting a seat, spawn checks ...」で始まり、`ralph org start --help` にもそのまま出るので、start の読み手には動詞の `spawn` が何を指すか曖昧 | `cli/org.go:46`、`:361-370`、`:395`、`:531` | (1)は `--config` の説明を元に戻し、限度の読み元は `spawn` と `start` の Long にだけ書く(`orgWideLimitsHelp` に既にある)。(2)は「Before starting a seat, `ralph org spawn` and `ralph org start` check ...」にする |

## Positive notes

- 判定を `spawnCapacityErr` の 1 関数にまとめ、ロックの中の本番の経路と、ロックなしの dry-run の経路が同じ順序で同じ関数を呼ぶ。dry-run が本番の拒否を予測するテスト(`TestOrgSpawn_OrgWideLimits_DryRunPredictsTheSameRejection`)もある
- idempotent の経路の拒否が `rejected` を書かない理由(状態のイベントなので、動いている leader が inactive に見える)を `idempotentRespawn` の doc に書き、`assertSeatUnchanged` と件数の比較(events、receipts、herdr、agmsg)で固定している。計画の指摘 3 を、コードとテストの両方で守っている
- `reserve.go` のパスの規則は 1 か所(`normalizeReservePath`)で、書き込み側と読み戻し側(`reservedPathsFromDetails`)が同じ関数を通る。`paths=` の形がカンマと空白を拒否する規則と一致していて、`restored:` の注記を後ろに付けても読み戻しが壊れない
- 上限が 0 以下の hand-built config を「無制限」ではなく拒否にして、`max_seats` と同じ fail-closed に揃え、`ValidateOrgWideCapacity` の doc とテストに書いている
- CLI の `--reserve` を `StringArray` にして(`StringSlice` ではなく)、カンマを含む 1 値を分割せず入力検査で拒否する。テスト(`TestOrgReserveFlag_ValueIsOnePath`)が選択を固定している
- 全体の上限を main worktree の `ralph.toml` から読むのは spawn と start だけにして、ほかの動詞が main の設定の読み込みエラーで止まらないようにしている。`MainWorktreeRoot` は git を再実行せず、解決済みの dir から戻して検算する
- 3 面(Go の既定、`templates/base/ralph.toml`、`scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh`)と同期テストが同じ commit で揃っている

## Coverage gaps

- テスト、mutation、`check-sync.sh`、`check-skill-sync.sh` は実行していない(指示による)。`/org` skill の 4 面が同じ内容であることは `diff -q` だけで見た
- 本物の herdr で、補償の窓(`disbanded` から補償までの数十秒)を作った実測はしていない。窓の挙動は `verbs_test.go` の fake で固定されたものを読んだだけ
- 同時実行のテスト(`TestOrgSpawn_ConcurrentSpawns_*`、`ConcurrentReservations_*`)は名前と構成を見ただけで、flock の下で本当に直列化されているかは再現していない
- macOS の標準の大文字小文字を区別しないファイルシステムでは、`Internal/` と `internal/` は別のパスとして比べられ、重ならない。パスの正規化は大文字小文字を畳まない。予約は助言であり(範囲外への書き込みは止めない)、計画に書かれていない扱いなので、finding にはせず、`/verify` と `/sync-docs` が判断する材料として残す

## Tech debt identified

新しい行は `/sync-docs` で足す。計画の進捗が挙げた 4 件のうち、(a)は F-10、(c)は F-4 と同じ。(b)(予約のパスの `*` はそのままファイル名として扱う)は skill に書いてあり、コードの挙動は `normalizeReservePath` が `*` を弾かないことと合う。(d)(start の `--scope` の help の表示崩れ)は既存。このほか、F-5、F-8、F-9 は直さない場合に tech-debt の候補。

`/sync-docs` に渡すもの(文書のずれは見ていないので、位置だけ):

- `docs/quality/quality-gates.md:77` は envelope validation を `max_seats` だけで書いている
- `internal/org/envelope_summary.go:36`(leader のプロンプトの `{{ENVELOPE}}`)と `internal/cli/doctor.go:788`(doctor の `[org]` の要約)は `max_seats` だけを出し、`max_orgs` と `max_total_seats` は出さない。出す・出さないは判断が要る
- `.claude/skills/org/SKILL.md:201` の「`--config` を渡したときと、台帳を flag か env で決めたとき、git の外では」は、`ResolveOrgStateDir` の 4 段目(`git-toplevel`、bare repository の linked worktree)も、main の `ralph.toml` を読まない場合に入ることを書いていない(`internal/org/statedir.go:38-44` と `withMainWorktreeOrgLimits` は `source != "git-main-worktree"` のすべてで読まない)
- 同じ SKILL の「どちらも台帳のロックの下で判定するので、同時に打った `spawn` でも超えない」は、自分の workspace の close が失敗した補償の窓では超える(計画のリスクとテスト `..._RiskWindowClearedByRetry` が固定している)ことを書いていない

## Recommendation

- Merge: 可。CRITICAL、HIGH はない。MEDIUM 1 件(F-1)は、scaffold が配る `ralph.toml` と doc comment が、まだ存在しない director に触れている点。`/verify` の前に直せば、コードは変わらないので cycle は増えない。LOW の 10 件のうち、F-2、F-3、F-11 は同じ編集で直せる
- Follow-ups: F-1 を直す。F-3 と F-11 は文言の直し。F-4 は今直すか tech-debt に送るかを決める(1 行で直せる)。F-5 は読み直しを足すか、doc に窓を足す。F-6、F-7、F-8、F-9、F-10 は受け入れて tech-debt の候補にしてよい
