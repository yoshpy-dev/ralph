# Self-review report: local-config-org

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-local-config-org.md`(issue #276 の 1 本目 A1、S1〜S5、D1〜D13、AC1〜AC13)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD bea2993e、70 ファイル、+2519/-968)。中心は S1 `internal/gitroot`、S2 `internal/config/local.go` と `Load` の重ね合わせ、S3 の `.gitignore` と gate の env と TestMain、S4a の `driver.AgmsgHomeOverride`、S4b の `RALPH_ORG_AGMSG_HOME` の撤去(`org_watch.go`、`org_director_headless.go`、`director_cmd.go`、`doctor.go`)。S5 の文書は、コードの挙動を言い切っている箇所だけ突き合わせた。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は L1〜L4(すべて LOW)。

## Evidence reviewed

- `internal/gitroot/gitroot.go` の全体と、移す前の `internal/org/statedir.go` の関数との対応。`MainWorktree`・`parseMainWorktreeRecord`・`Toplevel`・`SamePath` の本体は `-C dir` を足した以外は同じ。`statedir_test.go` から消えた `TestParseMainWorktreeRecord` の 7 ケースは `gitroot_test.go:218` に同じ中身で移っている。
- `internal/config/local.go` の全体と `config.go` の `Load`・`validateLoaded`。相対パス(`filepath.Abs` で cwd に解く)、symlink(`git -C` は symlink 越しで動き、返る root は git が解決した絶対パス)、git の外(`MainWorktree` と `Toplevel` がどちらも失敗し、config の dir の隣に落ちる)、bare repo の linked worktree(`MainWorktree` が false、`Toplevel` でその worktree の root)の 4 つを、コードと `local_test.go` の fixture(`TestLocalPath_*`、`TestLoad_LocalFileThroughASymlink`)で突き合わせた。
- エラーの握りつぶし: `readLocalConfig` は `fs.ErrNotExist` だけを「ファイルなし」とし、ほかの read error は `os.PathError` のままパスつきで返す。parse error は `fmt.Errorf("%s: %w", local.path, err)`、validation error は `(with <path> applied)` が付く。`LocalPath` が握りつぶすのは `filepath.Abs` の失敗と git の失敗で、どちらも意図したフォールバック(N2)。
- `driver.ResolveAgmsgHome` の順(`AgmsgHomeOverride` → `cfgValue` → 既定)と、呼び出し側 4 か所(`cli/org.go:354`、`cli/doctor.go:137`、`:170`、`org/prompts.go:80`)。`prompts.go:80` の `ResolveAgmsgHome("")` は `Org.AgmsgHome` が空のときだけで、CLI は `org.go:354` で解決した値を `Org.AgmsgHome` に入れている。`git grep AgmsgHomeOverride` の非テストの出現は `agmsg.go:148`(宣言)と `:158`(読み)だけ。
- `RALPH_ORG_AGMSG_HOME` の非テストの出現: `doctor.go:139-142`、`:839-862` と `agmsg.go:154` のコメントだけ。`scripts/`、`templates/`、`.claude/`、`.agents/` に生きた参照はない(`docs/evidence/` と `docs/reports/` の過去の記録を除く)。
- D7 と D13: `orgWatchChild`(`org_watch.go:405`)と `checkStartAgmsgHome`(`org_director_headless.go`)が同じ `relativeAgmsgHomeError` を使う。`orgWatchChild` は `launchAndAwaitOrgWatch` の `e.child()` で最初に呼ばれ、`orgWatchLaunch` と `watch.log` の前にある。`startDirector` の `checkStartAgmsgHome` は `resolveModelOrWarn` と `StartDirector` の前にある。拒否のあとに何も書かないことは、`org_watch_test.go` の `refused` 行(launch 0 件と `watch.log` なし)と `org_start_watch_test.go` の `checkNothingStarted` が見ている。
- gate: `gate_run.go` の `gateScriptEnv`。受け継いだ `RALPH_LOCAL_CONFIG` を名前の一致で落とし、末尾に `RALPH_VERIFY_SCOPE=full` と `RALPH_LOCAL_CONFIG=off` を足す。`Gate`(`gate.go:177`)と `Regate`(`regate.go:659`)の 2 か所が同じ関数を通る。`TestGateScriptEnv` は `=on`、`=`、`=off`、`=` なしの 4 通りの継承と、名前が前方一致・部分一致するだけの `RALPH_LOCAL_CONFIGX`、`X=RALPH_LOCAL_CONFIG=on` を固定している。
- doctor: `checkLegacyAgmsgHomeEnv` は空なら行なし、あれば warn 1 行で、値には使わない。`--strict` は他の check に影響しない。
- 三つの TestMain(`config`、`cli`、`org`)の env 固定と `setAgmsgHomeOverride`。`t.Cleanup` で戻す形。`t.Parallel` を使う test(`split_test.go:968`、`prompts_leader_plan_test.go`)は上書き口を書かず、読むだけ。`-race` は `scripts/`、CI の workflow、Makefile にない。
- root と `templates/base/` の写し: `.gitignore`、`scripts/ralph-config.sh`、director と org の skill、recipe 2 本が `cmp` / `diff` で一致。`templates/` に足された行に `internal/`・`cmd/ralph`・`config.Load` の参照はない。`git check-ignore -v ralph.local.toml config/ralph.local.toml` は両方とも `.gitignore:25` に当たる。
- `docs/tech-debt/README.md` を両方向で突き合わせた(148 行の config の解決の行に個人用のファイルを追記、170 行の `:100`→`:169`(`statedir.go:169` の `os.Stat` で合う)、270 行の `RALPH_ORG_AGMSG_HOME` の行を取り消し線と RESOLVED コメントで閉じる、273 行の (b) だけを取り消す、384 行に新しい行)。
- 確かめていない: テスト、静的検査(skill の規則どおり、走らせていない)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1 この PR が作った 1 行の転送関数 `resolvedOrClean` を、直さずに tech-debt の行にした。しかも行の理由が事実と違う。行は「呼び出し側は slice の外のファイルにある」と言うが、`feature.go` は S1(ab012b7d)が `samePath` → `gitroot.SamePath` で自分で書き換えたファイルで、同じファイルの 197 行目に `resolvedOrClean` の呼び出しがある。呼び出しは非テスト 3 行(`feature.go`、`permissions.go`、`split.go`、計 4 呼び出し)と `feature_test.go` の 5 か所で、機械的に置き換えられる。残すと 1 つの関数に 2 つの名前が付き、`gitroot` の関数をどう呼ぶかを読む人が 1 段余計にたどる。 | `internal/org/statedir.go:190-195`、`internal/org/feature.go:197`(S1 は同じ file の `:236` と `:576` を編集)、`internal/org/permissions.go:209`、`internal/org/split.go:183`、`docs/tech-debt/README.md:384`、`git log --format=%h origin/main..HEAD -- internal/org/feature.go` は ab012b7d だけ | 9 か所を `gitroot.ResolvedOrClean` に置き換えて転送関数と 384 行の行を消す。触るのは `feature.go`、`permissions.go`、`split.go`、`feature_test.go` で、挙動は変わらない。この PR で直さないなら、行の「Why deferred」から「files outside that slice」を外し、`feature.go` を触った事実に合う理由に書き直す。 |
| LOW | maintainability | L2 「空、絶対パス、`~/` 始まりのどれでもない値は相対」という判定が 2 つのパッケージに別々に書かれ、理由の説明も別々にある。`config.checkLocalAgmsgHome` は否定形(`home == "" \|\| IsAbs \|\| HasPrefix "~/"` なら通す)、`cli.agmsgHomeIsRelative` は肯定形で、今は補集合になっている。`cli` 側の doc comment は「`config.Load` は個人用のファイルの相対を断る」と他方の存在を指すが、2 つを結ぶ test はない。`driver.expandTilde` の `~/` も 3 つ目の書き方。D5 と D7 は同じ事情(読むプロセスの cwd がどこか分からない)から出た規則で、片方だけ変わる(例: `~user/` を許す)と、`--config` では通って個人用のファイルでは断られる値が黙って生まれる。 | `internal/config/local.go:101`、`internal/cli/org_watch.go:441-443`(doc comment は `:432-440`)、`internal/org/driver/agmsg.go:171-179` | `config` に `AgmsgHomeIsRelative(home string) bool` を 1 つ出し、`checkLocalAgmsgHome` と `agmsgHomeIsRelative` の両方がそれを呼ぶ。出さないなら、`local_test.go` の拒否・許可の一覧(`TestLoad_LocalAgmsgHome`)と `org_watch_test.go` の一覧を同じ値の表にして、2 つの判定が一致することを test で固定する。 |
| LOW | exception handling | L3 個人用のファイルの失敗が、変えていない doctor の枠を通り、ラベルと文が合わなくなる。`runDoctorFull` は `config.Load` の error を `Name: "ralph.toml"`、`parse error: %v — using defaults` で出す。個人用のファイルの error は Detail にパスが入るので原因は読めるが、名前は `ralph.toml` のまま。また `Load` は error のとき個人用の値を重ねた `cfg` を返す(`checkLocalAgmsgHome` は `Unmarshal` のあと)ので、相対の `agmsg_home` を断られた doctor は、`using defaults` と言いながら次の agmsg の check(`doctor.go:137`)で断られた相対の値を使う。ralph.toml の validation error でも前から同じ(`validateLoaded` の途中の `cfg`)だが、個人用のファイルが入ったことで、手で書く人が増える種類の error になった。 | `internal/cli/doctor.go:93-101` と `:137`(この diff では未変更)、`internal/config/config.go` の `Load`(`toml.Unmarshal(local.data, &cfg)` → `checkLocalAgmsgHome` → `return cfg, err`)、`internal/config/config.go:735`(`checkLocalAgmsgHome(local)` の呼び出し)と `internal/config/local.go:81-102` | doctor の warn の Name を、error の文にパスがあるとき `ralph.local.toml` にするか、Detail の頭に「ralph.toml または ralph.local.toml」と書く。`using defaults` を本当にするなら、`cfgErr` のとき `cfg = config.Default()` に替える(挙動が変わるので doctor の test の確認が要る)。どちらも直さないなら、`runDoctorOpts` に個人用のファイルが相対の `agmsg_home` を断る場合の test を 1 本足し、いまの出力を固定する。 |
| LOW | maintainability | L4 `AgmsgHomeOverride` が本番の経路に漏れないことは、いまは grep で言えるが、守っているのは doc comment の 1 文だけ。`driver` の export された package 変数なので、同じ module のどのコードからも代入でき、`go vet` も test も止めない。計画の AC6(「`internal/` と `cmd/` の production のコードで `RALPH_ORG_AGMSG_HOME` を読むのは doctor の warn だけ」)も同じで、行為は test で見ているが、「ソースにこの名前が戻る」ことを見る test がない。この repo は「繰り返す規則は test か hook に上げる」を方針にしている。 | `internal/org/driver/agmsg.go:139-148`、`git grep -n AgmsgHomeOverride -- ':!*_test.go'` が `agmsg.go:148` と `:158` の 2 行、`internal/config/skill_frontmatter_test.go` に repo を歩く test の前例がある | `driver` かどこかに、`internal/` と `cmd/` の `_test.go` 以外の `.go` を歩き、(a) `AgmsgHomeOverride` が `agmsg.go` 以外にないこと、(b) `"RALPH_ORG_AGMSG_HOME"` の文字列が `doctor.go` 以外にないこと、を確かめる test を 1 本足す。10 行ほどで、AC6 の後半も固定できる。 |

## Positive notes

- 移した `TestParseMainWorktreeRecord` の 7 ケースは、消えた側と一字違わない。`command()` は dir が空のとき `-C` を付けないので、git の stub が引数列全体で照合する `statedir_test.go` と、`rev-parse` の呼び出し数を数える `TestStatusCmd_MergeGuaranteeRunsNoGH` を壊さない。その理由が `command()` の doc comment に書かれ、2 つの test 名は実在する。
- `LocalPath` の fixture が、cwd をわざと git の外(または別の repo)にして `configPath` の dir から git を打つことを見ている(`TestLocalPath_InGitIsTheMainWorktreeRoot`、`_OutsideGitIsTheConfigFilesDirectory`)。cwd から打つ実装は落ちる。相対の `ralph.toml` は cwd を絶対にしてから git に渡す(`git -C` に相対 dir を渡すと cwd 基準になるため)。
- エラーがパスを名指す。読めない(directory、権限)・parse・validation の 3 種が `TestLoad_LocalFileErrorsNameItsPath` で ralph.toml あり・なしの両方を通り、権限の case は root のとき skip し、directory の case は uid に依らない。ralph.toml だけの validation error は文が変わらない(`TestLoad_RalphTomlValidationErrorDoesNotNameAnAbsentLocalFile`)。
- 二重の `Unmarshal` の重ね方を、表はキーごと・配列は置き換え・`orgPoolKeysPresent` は 2 つのどちらかにあれば「ある」、と 3 つに分けて test している。検査は重ねたあとの 1 回だけ。
- 拒否の側の test が、拒否の文(`relativeAgmsgHomeRefusal`)、launch 0 件、`watch.log` なし、台帳に何も書かれない、を一緒に見る。通る側(絶対、`~/`、`--config` なし、空の `--config`)も同じ表にあり、相対の `RALPH_ORG_AGMSG_HOME` を子が受け継いだまま変えられないことも固定している。
- gate の env の test は、継承の 4 形(`=on`、`=`、`=off`、`=` なし)と、前方一致・部分一致だけの別の変数を分けている。
- tech-debt の台帳は両方向で更新されている。取り消した 270 行は HTML コメントで閉じた理由を残し、273 行は (b) だけを取り消して (a) を開けたままにし、148 行の config の解決の行は個人用のファイルを足している。

## Notes(所見にしないもの)

- N1 `RALPH_LOCAL_CONFIG` という名前は、`RALPH_LOCAL_CONFIG=/path/to/file` のようにパスを渡せる変数に読める。実際はちょうど `off` だけに意味があり、ほかの値(パスを含む)は黙って個人用のファイルを読む。D4 で決めた形で、`ralph config get` の help にも `=off skips it` とあるので、直しは求めない。パスを渡す人が出たら `off` 以外の非空の値を警告する手がある。
- N2 `LocalPath` は git の失敗を 1 つにまとめて「git の外」として扱う。git が PATH にない、`safe.directory` で断られる、のときも config の dir の隣のファイルに落ちる。linked worktree でそうなると、主 worktree の root の個人用のファイルは黙って当たらない。台帳の解決(`ResolveOrgStateDir`)も同じ落ち方をするので、2 つの落ち方はそろっている。README が「linked worktree に置いた写しは読まれず、それを知らせる所はない」と書いているのと同じ種類のもの。
- N3 個人用のファイルの `agmsg_home = ""` は、`ralph.toml` の値ではなく内蔵の既定に戻る(`TestLoad_LocalAgmsgHome` の `{"", Default().Org.AgmsgHome}` が固定している)。既存の「空は既定に補う」規則に合うが、「キーごとに上書き」という説明からは外れる。README の `ralph.local.toml` の節に 1 文あるとよい(文書は /sync-docs の範囲)。
- N4 `orgWatchChild` の拒否は `ensureOrgWatch` の probe のあとに出る。stale な watch があると、拒否までに `orgWatchEnsureDeadline`(10 秒)待つ。doc comment は「launch するときだけ」と書いており、動いている watch が見つかれば相対の値でも断らない。意図どおり。
- N5 `config.Load` は 1 回に git を 2 回(bare の linked worktree では 3 回)打つ。呼び出しは `cli/config.go`、`cli/org.go` の 2 か所、`cli/doctor.go` で、ループの中にはない。

## Coverage gaps

- 実機の herdr と本物の agmsg での通しは打っていない(計画の範囲外)。
- git の `safe.directory` で断られる環境と、`GIT_DIR` を継承する git hook の中から `ralph` を打つ場合は、コードを読んだだけで動かしていない(N2)。

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 4 件)
- Follow-ups: L1 は 9 か所の置き換えで済み、この PR の理由づけが事実と違っているので、この PR の中で直すのがよい。L2 は述語を 1 か所にするか、2 つを結ぶ test を足す。L3 と L4 は直すか tech-debt に送るかを orchestrator が決める。
