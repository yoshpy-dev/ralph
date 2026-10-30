# Self-review report: org-state-dir-local

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-org-state-dir-local.md`(issue #276 の 2 本目 A2、S1〜S6、D1〜D7、AC1〜AC11)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base c03647f9、HEAD a3c975cf、67 ファイル、+3108/-573)。途中に main の取り込み 3700470e があるので、計画の base e3a3fbf7 からではなく origin/main から見た。中心は `internal/config/local.go` の `LocalOnlyKeys`・`checkLocalOnlyKeys`・`LocalOrgStateDir`、`internal/org/statedir.go` の `ResolveOrgStateDir`、`internal/cli/org_legacy_ledger.go` の `resolveOrgLedger` と `checkRetiredOrgStateDirEnv`、`org_inbox.go` の escalate の NOT RECORDED の経路、`org_cleanup.go` と `internal/org/cleanup.go`(S6)、doctor の `retiredEnvs`・`checkRetiredEnv`・`checkMergeGuarantee`。文書と skill は、コードの挙動を言い切っている箇所だけ見た。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は L1〜L9(すべて LOW)。L9 は依頼で範囲外とされたもの。

## Evidence reviewed

- 台帳の決め方: `ResolveOrgStateDir`(`statedir.go:70-89`)は flag → `config.LocalOrgStateDir("")` → `gitroot.MainWorktree("")` → `gitroot.Toplevel("")` → cwd の順。個人用のファイルの段の error は下の段に落ちずにそのまま返る(D4)。`LocalOrgStateDir`(`local.go:176-206`)は `RALPH_LOCAL_CONFIG=off` で git を打たずに空を返し、読めない・TOML でない・相対の値はパスを名指す error にする。`~/` は `os.UserHomeDir` で展開し、絶対パスは `filepath.Clean` する。本番で `OrgConfig.StateDir` を読むコードはない(`git grep 'Org\.StateDir'` の非テストの出現は `local.go:193` の probe だけ)。`ralph config get` だけが reflect で読む。
- `checkLocalOnlyKeys`(`local.go:125-141`)は ralph.toml の文書だけを map に読み、`tomlDocHasKey` で dotted key をたどる。空の値も断る。`LocalOnlyKeys()` は呼ぶたびに新しい slice を返し、`TestLocalOnlyKeys` がそれを固定している。
- `resolveOrgLedger`(`org_legacy_ledger.go:69-81`)の 16 の呼び出し: `org.go` の `newOrgRuntime`・spawn・start・start --plan・send、`org_watch.go`、`org_inbox.go`(escalate と inbox の runtime)、`org_merge_order.go`、`org_merge.go`、`org_cleanup.go`、`org_director.go`、`org_wait_director.go`、`status.go`、`insights.go`。どれも runtime を組む前、manifest・herdr・agmsg に触る前に呼ばれる。`newOrgSpawnRuntimeAt` から `guardLegacyOrgStateDir` を外しても `cmd` は `orgInboxHintStateDir` で使われている。doctor だけは `ResolveOrgStateDir` を直接呼び、env の扱いは `checkRetiredEnv` の warn に任せている。
- `checkRetiredOrgStateDirEnv`(`org_legacy_ledger.go:101-126`): 値は `TrimSpace` してから `filepath.Abs`。source が flag・config のとき、または `gitroot.SamePath` で既定の台帳と同じとき注意だけ。違うとき、台帳を変える動詞は error、読むだけの動詞は注意。error は `errOrgLedgerUnresolved` で包まれ、escalate はそれを `errors.Is` で見分けて NOT RECORDED に回す。`guardLegacyOrgStateDir` の error は包まずに返す。
- escalate: `newOrgEscalateFallbackRuntime("")` は `DesktopNotify` だけを持つ `Org` を返す。`EscalateNotRecorded` から `Inbox == nil` の拒否を外し、`alertUnrecordedEscalation` は `Inbox` が nil なら "any inbox" と書き、`EscalationsPath` が空なら行を書かない。nil の `Inbox` を参照する箇所は残っていない。`TestEscalateNotRecorded_WithNoLedgerWritesNothing` が state dir に何も書かれないことを見ている。
- cleanup(S6): `org_cleanup.go:141` は `resolveOrgLedger` に寄せ、source を `org.FeatureRepoRoot` に渡す。`internal/org/cleanup.go` の `ledgerMoveStep` は `config.LocalConfigFileName` を名指すだけの変更。`TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger` は gh を 0 回、snapshot を不変、env の場所が作られないことを、`TestOrgCleanup_PersonalFileNamesTheLedger` は相対の値で何も変えず、絶対の値で動かした台帳だけに `worktree_cleaned` が付くことを見ている。
- doctor: `retiredEnvs` の表と `checkRetiredEnv(os.Getenv)`。`checkMergeGuarantee` は台帳を決められないとき fail の `Org ledger` 行を先に足し、merge guarantee の行は「not cached」で続ける。Detail は `org.PrintableInboxText` を通る。TestMain の resolver は新しい引数を捨てて、存在しない state dir を返す。
- TestMain: `cli` は `RALPH_ORG_AGMSG_HOME` と `RALPH_ORG_STATE_DIR` を unset、`org` も `RALPH_ORG_STATE_DIR` を unset、どちらも `RALPH_LOCAL_CONFIG=off` のまま。
- source の文字列: 本番で `"env"` を比べる箇所はなく、`"flag", "config"` の組は `statedir.go:125` と `org_legacy_ledger.go:111` の 2 か所。
- `docs/tech-debt/README.md` を両方向で突き合わせた。この diff は register に触れていない。148・170・192 行が `RALPH_ORG_STATE_DIR` を台帳を選ぶ手として書いている(L8)。
- 確かめていない: テストと静的検査(skill の規則どおり走らせていない)。git の呼び出し回数(L5)はコードを読んで数えただけで、計測していない。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1 env の段を外したあとも、段の並びを書いたコメントに `env` が残っている。`org_watch.go:109` は S2(daf8e512)で `flag/env/config/` に書き足したもので、S4 で env の段を外したときに直していない。`status.go:451` の `printStateDirLine` の doc(`flag/env/`)と `org_test.go:2915` も同じ並びを書いている。`orgLedgerAccess` の doc(`org_legacy_ledger.go:17-33`)は「access が決めるのは `guardLegacyOrgStateDir` の扱い」とし、`orgLedgerMutating` は「legacy の台帳に座席がある間は断られる」とだけ書く。この PR から access は `checkRetiredOrgStateDirEnv` の拒否も決めている。 | `internal/cli/org_watch.go:109`(`git blame` は daf8e512)、`internal/cli/status.go:451`、`internal/cli/org_test.go:2915`、`internal/cli/org_legacy_ledger.go:17-33` | 3 か所の並びを `flag/config/git-main-worktree/git-toplevel/cwd` にする。`orgLedgerAccess` の 2 つの doc に、env が別の台帳を指す間の拒否を足す。 |
| LOW | naming | L2 `checkRetiredOrgStateDirEnv` の拒否も `errOrgLedgerUnresolved` で包まれ、文の頭が「cannot resolve the org ledger:」になる。続く文は「the ledger ralph finds by default is <dir>」と台帳を決めたことを言うので、1 つの error の中で前後が食い違う。同じ「台帳を変える動詞を止める」拒否でも、`guardLegacyOrgStateDir` は「org: refusing to change the org ledger:」と書く。env を設定したまま更新した人は、最初にこの error を読む。 | `internal/cli/org_legacy_ledger.go:44`、`:74-76`。テストが固定している文: `internal/cli/org_cleanup_test.go` の `want`(`"ralph org cleanup: cannot resolve the org ledger: RALPH_ORG_STATE_DIR names "...`) | env の拒否は「refusing to change the org ledger」で包む。escalate の見分けには、sentinel を 2 つにして両方を `errors.Is` で見るか、名前を `errOrgLedgerNotUsable` のように両方の意味に合うものに変える。固定している文のテストも合わせて直す。 |
| LOW | maintainability | L3 「絶対パスか `~/` で始まる」の判定が 3 か所に、`~/` の展開が 2 か所にある。判定は `checkLocalAgmsgHome`(受け入れ形、`local.go:106`)、`LocalOrgStateDir` の switch(`local.go:194-205`)、`agmsgHomeIsRelative`(拒否形、`org_watch.go:442`)。展開は `driver.expandTilde`(`agmsg.go:171-180`、`UserHomeDir` が失敗すると値をそのまま返す)と `LocalOrgStateDir`(同じ失敗を error にする)で、失敗の扱いが違う。A1 の self-review L2 が 2 か所の時点で指摘し、PR の Known gaps に残した(register の行はない)。この PR が 3 つ目を足した。 | `internal/config/local.go:106`、`:194-205`、`internal/cli/org_watch.go:442`、`internal/org/driver/agmsg.go:171-180`、`docs/reports/sync-docs-2026-10-29-local-config-org.md:69` | `config` に判定と展開の helper を 1 つずつ置き、4 か所から呼ぶ。直さないなら、A1 の L2 と合わせて 1 行の register の行にする。2 回続いたので、PR の Known gaps だけでは次も見落とす。 |
| LOW | exception-handling | L4 Load の新しい error(`checkLocalOnlyKeys`)が、変えていない doctor の枠を通る。doctor は `config.Load` の error を `ralph.toml: warn — parse error: %v — using defaults` で出す(`doctor.go:109-114`)。`ralph.toml` に `[org].state_dir` があると、方針で断る error が「parse error」と呼ばれる。さらに `checkLocalOnlyKeys` は `toml.Unmarshal(data, &cfg)` のあと(`config.go:726-731`)にあるので、Load は ralph.toml の値を入れた `cfg` を返す。doctor はその `cfg` で後の検査(agmsg、envelope)を続けるので、「using defaults」は事実と違う。A1 の self-review L3 と同じ形で、そちらも Known gaps に残っている。 | `internal/cli/doctor.go:106-114`、`internal/config/config.go:726-731` | `checkLocalOnlyKeys` を `toml.Unmarshal(data, &cfg)` の前に移す。そうすれば error のときの `cfg` は `Default()` のままで、doctor の文と合う。doctor の枠(名前 `ralph.toml`、`parse error`)の直しは A1 の L3 と一緒に扱う。 |
| LOW | maintainability | L5 既定の段で台帳を決めると、`gitroot.MainWorktree` を 2 回呼ぶ。1 回目は `LocalOrgStateDir("")` の中の `config.LocalPath`、2 回目は `ResolveOrgStateDir` の 3 段目。どちらも git を 2 回打つので、個人用のファイルが `state_dir` を持たない普通の場合に、1 回の動詞で git が 2 回から 4 回に増える。`LocalOrgStateDir` の引数 `dir` と返り値 `path` を使うのはテストだけで、本番の呼び出しは `local, _, err` で `path` を捨てる。その一方で `checkRetiredOrgStateDirEnv` の拒否は `config.LocalPath("ralph.toml")` でパスを計算し直す(git がさらに 2 回)。この repo は「解決ごとに git を打つ」ことを理由に二重の解決を一度直している(`org.go:345-350` のコメント)。 | `internal/org/statedir.go:74`、`:81`、`internal/config/local.go:42-54`、`:180`、`internal/cli/org_legacy_ledger.go:120`、`internal/cli/org.go:345-350` | 主 worktree の root を 1 回だけ求め、`<root>/ralph.local.toml` を読む関数(例: `config.LocalOrgStateDirAt(root)`)に渡す。または `LocalOrgStateDir` が見つけた root を返し、3 段目がそれを使う。拒否の文のパスも `ResolveOrgStateDir` の結果から渡す。 |
| LOW | null-safety | L6 個人用のファイルの段は値に `mustAbs` を通さない。`ResolveOrgStateDir` の doc は「The returned dir is always an absolute path」(`statedir.go:66`)と約束しているが、`~/` の値は `os.UserHomeDir()` が絶対パスのときだけ絶対になる。`$HOME` が相対だと台帳も相対になり、子のプロセスや hint の `--state-dir` が別の cwd から解かれる。この不変条件が防いでいる cwd のずれがそのまま起きる。起こりにくい環境だが、直しは 1 行で済む。 | `internal/org/statedir.go:66`、`:79`、`internal/config/local.go:197-202` | `return mustAbs(local), "config", nil` にする。または `LocalOrgStateDir` で展開後の値が絶対でなければ error にする。 |
| LOW | naming | L7 個人用のファイルの段の source の名前 `config` は、`ralph status`(`state-dir: … (source: config)`)、`ralph status --json` の `state_dir_source`、watch の起動の行に出る。D1 で、この値は `--config` のファイルからは読まないと決めている。`--config` を渡した人には「source: config」が「自分の `--config` から」と読める。名前は計画の Scope が決めたもので、直しは任意。 | `internal/org/statedir.go:79`、`internal/cli/status.go:459`、`:545`、`internal/cli/org_watch.go:243` | 直すなら `local-config` か `personal-file` のように、`--config` と取り違えない名前にする。`statedir.go:125` と `org_legacy_ledger.go:111` の比較も合わせる。 |
| LOW | maintainability | L8 tech-debt の register が、この PR で変わった事実を追っていない。148 行は台帳の決め方を「`--state-dir`, else `RALPH_ORG_STATE_DIR`, else …」と書き、Trigger に「a further change to `ResolveOrgStateDir`」を挙げている。この PR はその条件を満たした。しかも 148 行の論点(config と台帳を別の規則で探す)は D1 で広がった。個人用のファイルは、Load では `--config` のディレクトリから、台帳では cwd から探す。170 行(V-2 の「`--state-dir` / `RALPH_ORG_STATE_DIR` way」)と 192 行も、env を台帳を選ぶ手として書いている。diff は register に触れていない。 | `docs/tech-debt/README.md:148`、`:170`、`:192` | 148 行に再判定の注記を足す。満たした Trigger、開いたままの部分、D1 で `ralph.local.toml` の探し方が Load と台帳で分かれたことの 3 点。170 行と 192 行の env の言及を `--state-dir` か `[org].state_dir` に直す。/sync-docs が引き取ってもよい。 |
| LOW(範囲外) | maintainability | L9 cleanup と merge の案内のコマンドは、台帳を `--state-dir` で決めたときも `--state-dir` を付けない(`cleanupAgain`、`internal/org/cleanup.go:274`)。#288 からの振る舞いで、main でも同じ。この PR で `[org].state_dir` が台帳を固定する手になったので、個人用のファイルで決めた人には素のコマンドが同じ台帳を指す。残るのは 1 回だけ `--state-dir` を渡した場合だけ。 | `internal/org/cleanup.go:274-276` | この PR では直さない。`orgInboxHintStateDir` と同じく、flag のときだけ `--state-dir` を付ける形が使える。 |

## Positive notes

- 16 か所の「決める → legacy の検査」を `resolveOrgLedger` に寄せたので、env の扱いと error の印を 1 か所で持てている。拒否はどの動詞でも runtime を組む前に出る。start・spawn・send・cleanup のテストが、台帳が増えないこと、見張りを起こさないこと、herdr と agmsg を呼ばないこと、env の場所を作らないことを一緒に見ている。
- 個人用のファイルが壊れているとき、下の段に黙って落ちない(D4)。error はパスを名指す。doctor だけは fail の 1 行にして残りを続け、キャッシュは書かない。その 3 点を `TestRunDoctorFull_PersonalFileThatCannotResolveTheLedgerIsAFailLine` が、ファイルなしの対照(キャッシュが書かれる)と並べて見ている。
- escalate は、台帳がないときの `Org` を nil 安全にした。`Inbox` が nil でも banner は「any inbox」と書き、`EscalationsPath` が空なら行を書かない。前の `Inbox == nil` の拒否を外しても、nil を参照する箇所は残っていない。
- `checkMergeGuarantee` を `[]checkResult` にし、status の決め方を `mergeGuaranteeStatus` に分けた。fail の経路と通常の経路で status がずれない。
- TestMain が `cli` と `org` の両方で `RALPH_ORG_STATE_DIR` を unset する。開発機の shell に残った値で、台帳を変える動詞のテストが断られることはない。
- `LocalOnlyKeys` を template のドリフトの probe と突き合わせるテスト(`TestTemplateDriftProbesHoldNoLocalOnlyKey`)がある。注釈にしか書けないキーを、将来 probe に足す誤りを止められる。

## Notes(所見にしないもの)

- N1 `checkRetiredEnv` は値をそのまま空と比べ、`checkRetiredOrgStateDirEnv` は `TrimSpace` してから比べる。空白だけの値では doctor が warn を出し、動詞は黙る。A1 の `checkLegacyAgmsgHomeEnv` も同じ比べ方で、実害はない。
- N2 `gitroot.SamePath` は、存在しないパスの symlink を解かない。既定の台帳がまだ作られていない repository で、env が symlink を通る形で同じ場所を指すと「違う場所」と判定され、台帳を変える動詞は断られる。env を前に使っていれば台帳はあるので、普通は起きない。起きても拒否の文が直し方を言う。
- N3 拒否の文の `[org].state_dir = %q` は Go の引用で、制御文字は `\x..` になる。TOML の basic string は `\x` を受け付けない。制御文字を含むパスのときだけの話なので、所見にしない。
- N4 `newOrgEscalateFallbackRuntime("")` は `wireOrgInbox` の `DesktopNotify` の設定を写している。`EscalateNotRecorded` が使うフィールドが増えたら、2 か所を直すことになる。doc comment が「wireOrgInbox が設定するとおりに」と書いているので、いまは足りている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

この review では register に行を足していない。cycle 1 なので、L1〜L8 は直す余地がある。L3 と L4 を直さずに merge するなら、A1 の L2・L3 と合わせて 1 行にまとめることを勧める。2 本続けて PR の Known gaps にだけ残ったので、register にないと 3 本目(計画の B)でも同じことが起きる。

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 9 件、うち 1 件は範囲外)
- Follow-ups: L1 と L6 はそれぞれ数行で直る。L2 は文を固定しているテストの書き換えを伴う。L3・L4 は直すか register に 1 行。L8 は /sync-docs で引き取れる。
