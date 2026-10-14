# org-merge-guarantee

- Status: Approved
- Approved: 2026-10-14 sha256:d2923a8bf359
- Owner: Claude Code
- Date: 2026-10-14
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 7 段目の最後(7c-2)。FR-7 の最後の 2 つ(`ralph doctor` は GitHub の保護ブランチに「最新でないと merge できない」設定がなければ warn する。`ralph status` は merge の時点の保証の種類(GitHub の保護か、PR 作成時の検査だけか)を表示する)。7c-1(`ralph org merge-order`)は PR #227(a16d7c4f)で merge 済み。分け方は 7c-1 の計画(`docs/plans/archive/2026-10-14-org-merge-order.md`)の C1。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-merge-guarantee

## Objective

merge の時点で何が PR の検証を保証しているかを、人と director が見られるようにする。`ralph doctor` が origin の GitHub の repository の既定のブランチの保護を読み、「最新でないと merge できない」設定(または merge queue)と必須の検査がなければ warn する。読んだ結果を台帳の下のキャッシュに書き、`ralph status` と `ralph org merge-order` の出力がそれを読んで、保証の種類(GitHub の保護か、PR 作成時の検査だけか、分からないか)を出す。status と merge-order はネットワークに出ない。

## Scope

- 保証の種類の判定(org の層、新しいファイル `internal/org/mergeguarantee.go`)
  - 種類は 3 つ
    - `github_protection`: GitHub が、merge の時点で最新の base を含む head での必須の検査を強制する
    - `pr_checks_only`: 強制しない。保証は PR を作った時点の gate と CI だけ
    - `unknown`: 読めない、または決められない
  - repository は origin のもの: `git remote get-url origin` の host と owner/repo(owner/repo は 7c-1 の `mergeRepoFromOriginURL`)。読むのは origin の repository の既定のブランチの保護で、origin が fork なら fork の保護になる(PR の統合先の保護ではない。7c-1 と同じ割り切りで、detail と文書に書く)
  - 読む host(C7): どの gh にも `--hostname <host>` を渡す(`GH_HOST` を効かせない)。host は origin の URL に書かれたままのもの(port と user を除く)で、`ssh.github.com` だけを `github.com` に読み替える。host を名指さない URL(ローカルのパスなど)は `unknown`。ssh の別名(`~/.ssh/config` の Host)は解かない。gh がその host に認証していなければ gh が失敗して `unknown` になり、detail は「ssh の別名なら doctor は解けない」と言う(tech-debt)
  - gh が PATH になければ(`exec.LookPath`)gh を打たず、すぐ `unknown`(C1)
  - 打つ gh(どれも `GH_PROMPT_DISABLED=1` と `--hostname`。全体で 1 つの期限 20 秒(C1)。順に打ち、決まった時点で止める)
    1. `gh api repos/<owner>/<repo>`: `default_branch` と `full_name`
    2. `gh api --paginate repos/<owner>/<repo>/rules/branches/<既定>`: 有効な ruleset の規則(evaluate と disabled の ruleset は返らない)。出力は 1 つの配列か、ページごとの配列を連結したもの(gh の版による)で、どちらも読む
    3. `gh api repos/<owner>/<repo>/branches/<既定>`: classic の保護の有無(`protection.enabled`)と必須の検査(`protection.required_status_checks` の `enforcement_level` が `off` でなく、`contexts` か `checks` が 1 件以上)
    4. classic の保護に必須の検査があり、2 で決まっていないときだけ `gh api repos/<owner>/<repo>/branches/<既定>/protection/required_status_checks`: `strict`。404 で message が `Not Found`(admin の権限がない)、403 なら strict は分からない
  - 決め方(C4)
    - `github_protection`: ruleset の `required_status_checks` が `strict_required_status_checks_policy: true` でその規則の必須の検査が 1 件以上、または classic の `strict: true` で classic の必須の検査が 1 件以上、または ruleset の `merge_queue` があり、ruleset か classic に必須の検査が 1 件以上
    - `unknown`: classic の保護に必須の検査があり、strict を読めない(4 の 404 `Not Found`・403)。gh・origin・JSON の失敗、期限切れ
    - `pr_checks_only`: それ以外(保護がない、必須の検査がない、strict が false、strict や merge queue はあるが必須の検査が 0 件)。必須の検査が 0 件の strict は、「最新にはなるが検査は走らない」と detail に書く
  - 決まった種類に、理由の 1 文(どの設定を見たか)、既定のブランチ、origin の owner/repo、gh が返した `full_name` を付ける
  - 外のコマンドは 7c-1 の `MergeOrderExec` と同じ形の seam で打つ(テストは偽物)
- キャッシュ(C2): `<state dir>/merge-guarantee.json`(`version`、`kind`、`detail`、`host`(origin の URL の host)、`repo`(origin の owner/repo)、`api_repo`(`full_name`)、`default_branch`、`checked_at`(UTC RFC3339))。doctor が「最後に見た結果」を書く。一時ファイルと rename で書く
- キャッシュを読む(status と merge-order が使う 1 つの関数): 次のどれかなら種類を `unknown` にし、理由と、読めれば最後に見た種類と `checked_at` を添える
  - キャッシュがない、形が読めない、版が違う
  - `checked_at` から 24 時間(定数 `MergeGuaranteeMaxAge`)を過ぎた(C5)
  - 今の `git remote get-url origin`(ローカル。ネットワークに出ない)の host と owner/repo が、キャッシュの `host` と `repo` と大文字小文字を無視して違う、または origin を読めない(打った場所が repo の外など)(C7)
- `ralph doctor`(`internal/cli/doctor.go`、新しいファイル `internal/cli/doctor_merge_guarantee.go`): 検査 `Merge guarantee` を足す
  - `github_protection` は `pass`、`pr_checks_only` は `warn`(何を設定すればよいか: Require branches to be up to date before merging と必須の検査、または merge queue)、`unknown` は `info`(理由)(C3)
  - 台帳のディレクトリ(`org.ResolveOrgStateDir("", false)` の解決先)が既にあるときだけ、読んだ結果を `unknown` でもいつも書く(C2、C5)。書けなければ古いキャッシュを消し、detail にそう書く。消せもしなければ検査を `warn` にし、古いキャッシュが残ると言う
  - doctor.go の「subprocess を増やすのは `--probe-models` だけ」の注釈を書き換える
- `ralph status`: `org watch:` の行のあとに `merge guarantee:` の 1 行を出す(org があってもなくても)。`--json` に `merge_guarantee` を足す。台帳のディレクトリがなければ `unknown (no org ledger yet)`、あってキャッシュがなければ `unknown (not checked: run ralph doctor)`、読めなければ `unknown (cache unreadable: run ralph doctor)`、古ければ `unknown (stale: last <kind> at <checked_at>; run ralph doctor)`、origin が違えば `unknown (doctor checked <host>/<repo>; origin is now <host>/<repo>)`(C2、C5、C7)
- `ralph org merge-order`: テキストの先頭の `origin/<既定>` の行のあとに `merge guarantee:` の行を出し、`--json` の `MergeOrderResult` に `merge_guarantee` を足す。キャッシュは自分の state dir の解決先から、同じ関数で読む。加えて、キャッシュの `default_branch` が ls-remote で読んだ今の既定のブランチと違えば `unknown`(doctor が見たブランチと今のブランチを言う)
- 保証の行には `checked_at` を出す
- 文書: README(doctor の表の行と org の節)、`/org` skill の 4 面(「merge してよい順」の節の保証の段落)、仕様 FR-7 の「7c-2 で決めたこと」の注、AGENTS.md の repo map(新しいファイル)、`docs/tech-debt/README.md`

## Non-goals

- 8a 段目: director が保証を報告すること、`auto_merge`
- doctor が保護を設定・変更すること(読むだけ)
- GitHub 以外のホスト(GitLab など)の保護。ssh の別名(`~/.ssh/config` の Host)を解くこと(別名の origin は `unknown`。tech-debt)
- 管理者や ruleset の bypass で保護を飛ばせること(保護があれば `github_protection` とする。tech-debt に書く)
- PR の統合先が既定のブランチでないときの、その統合先の保護(merge-order の `wrong_base` が先に言う)
- status や merge-order が gh を打つこと

## Assumptions

- `GET /repos/{o}/{r}/rules/branches/{b}` は、読める人なら誰でも、そのブランチに効く有効な規則を、repository と organization の ruleset から返す。evaluate と disabled の ruleset の規則は返さない(GitHub の REST の文書)。`strict_required_status_checks_policy` は「必須の検査が 1 件以上ないと効かない」(同じ文書)。2026-10-14 に github/docs の main で strict(必須の検査 37 件)と `merge_queue` を、yoshpy-dev/ralph の main で strict のない ruleset を確かめた。この endpoint はページを分ける(`per_page=2` で `Link: rel="next"`)。gh 2.102.0 の `--paginate` は 1 つの配列にまとめて出した
- `GET /repos/{o}/{r}/branches/{b}` の `protection` は読める人なら誰でも読め、classic の保護の `enabled` と必須の検査(`enforcement_level`、`contexts`、`checks`)を返すが、`strict` は返さない(cli/cli の trunk、golang/go の master で確かめた)。ruleset だけのブランチは `protected: true`、`protection.enabled: false`(github/docs の main)
- classic の `GET .../branches/{b}/protection` 以下は admin の権限が要る。admin でない人には、保護があっても 404 `Not Found` を返す(cli/cli の trunk で確かめた)。保護のないブランチは 404 `Branch not protected`(yoshpy-dev/ralph の main)
- 改名した repository の古い名前の API は新しい名前に転送され、`full_name` は新しい名前を返す(vuejs/vue-next → vuejs/core で確かめた)
- `gh api --hostname github.com repos/yoshpy-dev/ralph` は `full_name` を返し、ssh の別名の host(`--hostname github.com.emu`)は `error connecting to github.com.emu` で失敗する(2026-10-14、gh 2.102.0)。この repo の origin は `ssh://git@github.com/yoshpy-dev/ralph.git`

## Affected areas

- 新しいファイル: `internal/org/mergeguarantee.go`(判定、キャッシュの読み書き)、`internal/org/mergeguarantee_test.go`、`internal/cli/doctor_merge_guarantee.go`(doctor の検査と seam)、`internal/cli/doctor_merge_guarantee_test.go`
- 変えるファイル: `internal/cli/doctor.go`(検査を足す、注釈)、`internal/cli/status.go`(行と JSON)、`internal/cli/org_merge_order.go`(行)、`internal/org/mergeorder.go`(`MergeOrderResult` の field と読み込み)、`internal/cli/main_test.go`(TestMain で seam を固める)、`internal/cli/status_test.go`、`internal/cli/org_merge_order_test.go`、README、`/org` skill の 4 面、仕様の FR-7 の注、AGENTS.md、`docs/tech-debt/README.md`

## Visual review

- Page: `.harness/state/plan-visual/org-merge-guarantee.html`
- Self-check: ページ全体と全体図(`--fragment overview`、1150×580)を撮って読んだ。図は 3 つ(全体図、保証の種類の決め方、出力に増える行)。1 回目は全体図の doctor の箱で副題が枠に触れ、「書く」の注記の背景がグループの枠に重なったので、副題を短くし、注記をグループの外に下げた。撮り直して読み、文字の重なり、はみ出し、線が文字を横切る箇所、的を外した矢印はない。Codex の助言(C5・C7)を入れたあと、全体図(キャッシュの field、`--hostname`、読む側の照合の注)、図 2 の①、図 3(unknown の理由を 5 つに)を描き直し、ページ全体と全体図(1150×600)を撮り直して読んだ。同じ問題はない

## Design decisions

- C1 doctor はこの検査をいつも打つ(仕様の「doctor は warn する」の字義どおり)。gh が PATH になければ gh を打たずに `unknown` にし、gh の 4 回の呼び出しに全体で 1 つの期限(20 秒)をかける。退けた案: `--probe-github` のようなフラグのときだけ打つ(既定の doctor は速いが、status の保証の行が既定で `unknown` のままになる)、台帳があるときだけ打つ
- C2 キャッシュは `<state dir>/merge-guarantee.json` に、台帳のディレクトリが既にあるときだけ書く。org を使っていない repo に doctor が `.harness/state/org/` を作らない。status は、台帳のディレクトリがないとき(doctor を打っても書かれない)と、あってキャッシュがないときを言い分ける。退けた案: いつも書く(ディレクトリを作る)、キャッシュを持たず status と merge-order が毎回 gh を打つ(7c-1 の C1 と違い、status がネットワークに出る)
- C3 doctor の状態は、`github_protection` が `pass`、`pr_checks_only` が `warn`、`unknown` が `info`。退けた案: `unknown` も `warn`(gh のない人、オフラインの人、classic の保護のある他人の repo の admin でない人に、いつも warn が出る)
- C4 `github_protection` は、最新にする設定(ruleset の strict、classic の strict)か merge queue に加えて、必須の検査が 1 件以上あるときだけ。0 件なら `pr_checks_only`。GitHub の文書が strict は「必須の検査が 1 件以上ないと効かない」と言い、この計画の `github_protection` の定義(最新の base を含む head での検査を強制する)にも合う。classic の strict を読めないときは `unknown` にし、推測で `github_protection` にしない
- C5 キャッシュは「最後に見た結果」で、`checked_at` から 24 時間(定数)を過ぎたら読む側が `unknown` にし、最後に見た種類と時刻を添える。doctor は読んだ結果を `unknown` でもいつも書き、書けなければ古いキャッシュを消す。こうすると、古い `github_protection` が残る経路は「書けず、消せもしない」の 1 つだけで、そのとき doctor は `warn` を出す。期限は設定にしない(doctor は手で打ち直せる)。Codex の計画助言 #2(HIGH)。最初の案(古さで種類を変えない)を退けた
- C6 owner/repo は `git remote get-url origin` から(7c-1 と同じ)、既定のブランチは `gh api repos/<o>/<r>` の `default_branch` から読む。doctor の state dir は `org.ResolveOrgStateDir("", false)` で、merge-order は自分の `--state-dir` の解決先から読む(既定では同じ規則で同じ場所)。退けた案: 既定のブランチを `git ls-remote --symref` で読む(git のネットワークも使う)
- C7 保護を読んだ対象(host と owner/repo)をキャッシュに残し、読む側は今の origin の host と owner/repo と比べて、違えば `unknown` にする。取得では、origin の URL に書かれた host を `gh api --hostname` に渡し(`ssh.github.com` だけ `github.com` に読み替え)、`GH_HOST` を効かせない。ssh の別名は解かず、gh が失敗して `unknown` になる。Codex の計画助言 #1(HIGH)。退けた案: `ssh -G` で別名を解く(外部コマンドと解析が増え、`ssh.github.com` の読み替えはそれでも要る。consult の判定)、host を渡さない(7c-1 と同じだが、GHES の origin で github.com の同名の repository を読みうる)

Critical forks: C1〜C7 は consult(consult-plan-guarantee)の判定で決めた(ユーザーの指示「計画の分かれ目も自分で選ぶ」)。C4 は私の案(必須の検査 0 件も `github_protection`)を consult が退け、GitHub の文書で確かめた。C5 と C7 は Codex の計画助言の HIGH 2 件を、ユーザーの指示どおりに取り込み、直し方を consult と詰めた。

## Acceptance criteria

- [ ] AC1 ruleset の `required_status_checks` が strict で必須の検査が 1 件以上のとき、または ruleset に `merge_queue` があり必須の検査が 1 件以上あるとき、doctor の `Merge guarantee` は `pass` で、種類は `github_protection`
- [ ] AC2 classic の保護の必須の検査(1 件以上)が `strict: true` のとき `github_protection`。classic の保護に必須の検査があり、admin でなく strict を読めない(404 `Not Found` か 403)とき `unknown`(`info`)
- [ ] AC3 「最新でないと merge できない」設定か merge queue がない、または必須の検査が 0 件のとき(保護がない、必須の検査がない、strict が false、strict や merge queue だけ)、doctor は `warn` で、何を設定すればよいかを言う。種類は `pr_checks_only`。必須の検査が 0 件の strict は、最新にはなるが検査は走らないと言う
- [ ] AC4 origin がない、origin の URL が host か owner/repo を名指さない、gh が PATH にない(gh を打たない)、gh が失敗する(ssh の別名の host を含む)、全体の期限を過ぎる、JSON が読めないとき、doctor は `info` で理由を言い、種類は `unknown`。どれでも doctor の終了コードは変わらない。どの gh にも `--hostname <origin の host>` を渡す(`ssh.github.com` は `github.com`)
- [ ] AC5 doctor は、台帳のディレクトリがあるときだけ `merge-guarantee.json` を、読んだ結果が `unknown` でも書く。ないときは作らない。書けなければ古いキャッシュを消し、detail にそう書く。消せもしなければ検査は `warn`
- [ ] AC6 `ralph status` はキャッシュの種類・既定のブランチ・`checked_at` を `merge guarantee:` の 1 行で出し(org があってもなくても)、`--json` に `merge_guarantee` を出す。台帳のディレクトリがない、キャッシュがない、読めない、24 時間を過ぎた、今の origin の host か owner/repo がキャッシュと違う、origin を読めないときは `unknown` と、それぞれの理由を言う。gh を打たず、git もネットワークに出ない
- [ ] AC7 `ralph org merge-order` は、テキストの先頭の既定のブランチの行のあとに保証の行を、`--json` に `merge_guarantee` を出す。AC6 と同じ条件のほか、キャッシュの既定のブランチが今の既定のブランチと違えば `unknown`
- [ ] AC10 保護のある repository で `github_protection` を書いたあと、origin を同じ既定のブランチの名前を持つ別の repository に向け直すと、status と merge-order は `unknown` を出す。古い `github_protection` のキャッシュがあるまま doctor の書き込みが失敗すると、古いキャッシュは消える。時計を 24 時間進めると `unknown` になる
- [ ] AC8 テストは本物の gh を打たない(TestMain で seam を固め、テストが偽物の呼ばれ方を確かめる)。テストは repo の台帳にキャッシュを書かない
- [ ] AC9 README、`/org` skill の 4 面、仕様 FR-7 の注、AGENTS.md、tech-debt が保証の判定と表示を説明する。skill の 4 面は同じ中身

## Implementation outline

1. S1 org の層(`internal/org/mergeguarantee.go`): 判定(seam で git と gh を打つ)、キャッシュの読み書き。テストは偽の seam で分岐ごとに固める
2. S2 doctor(`internal/cli/doctor_merge_guarantee.go`、`doctor.go`、`main_test.go`): 検査の状態と detail、キャッシュを書く条件、seam を TestMain で固める
3. S3 status と merge-order: キャッシュを読む行と JSON
4. S4 文書

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`gofmt -l`、`go vet ./...`、`GOTOOLCHAIN=go1.25.8 go build ./... && go vet ./...`、`scripts/check-skill-sync.sh`、`scripts/check-sync.sh`
- Spec compliance criteria to confirm: FR-7 の「doctor は保護ブランチに最新でないと merge できない設定がなければ warn」を AC1〜AC4、「status は merge の時点の保証の種類を表示」を AC6 と対応づける。7c-1 の C1(doctor が台帳の下にキャッシュを書き、status と merge-order が読む)と合っていること
- Documentation drift to check: README の doctor の行、`/org` skill の 4 面が同じこと、AGENTS.md の repo map
- Evidence to capture: HEAD の binary で doctor を本物の gh で打った出力(この repo の main は strict のない ruleset なので `warn`、github/docs を origin にした一時の clone で `pass`、classic の保護がある他人の repo(cli/cli)を origin にした一時の clone で `unknown`)、status と merge-order の保証の行

## Test plan

- Unit tests(`internal/org/mergeguarantee_test.go`、偽の seam)
  - ruleset: strict で必須の検査あり、strict で必須の検査 0 件、strict が false、merge_queue と必須の検査、merge_queue だけ、規則なし
  - classic: strict true / false、必須の検査なし(`enforcement_level` が `off`)、strict を読めない(404 `Not Found`、403)、保護なし(`protection.enabled: false`)
  - ruleset と classic の組み合わせ(merge_queue と classic の必須の検査)
  - 失敗: origin なし、host か owner/repo のない URL、gh が PATH にない(gh を打たない)、gh の失敗(ssh の別名の host)、全体の期限切れ、JSON が読めない、`default_branch` が空
  - host: https・ssh・scp 形の URL から host を読む(port と user を除く)、`ssh.github.com` を `github.com` に読み替える、どの gh にも `--hostname` が付く
  - paginate: 1 つの配列、連結された複数の配列(`[...][...]`)の両方
  - 改名(`full_name` が origin の owner/repo と違う)
  - キャッシュ: 書いて読む、壊れたファイル、版の違い、一時ファイルが残らない、24 時間の境目(偽の時計)、今の origin の host・owner/repo が違う(大文字小文字は無視)、origin を読めない、書けないときに古いファイルを消す、消せないとき
- Integration tests: doctor(`runDoctorFull` に偽の seam と一時の台帳、台帳のディレクトリのある・なし)、status(台帳なし、キャッシュあり・なし・壊れた・古い、org あり・なし、origin が違う、repo の外で打つ、`--json`)、merge-order(本物の git と bare の origin、キャッシュの既定のブランチが合う・違う、origin を別の repository に向け直す、`--json`)
- Regression tests: `go test ./internal/org/... ./internal/cli/...` 全体を go1.25.8 と go1.26 で。既存の doctor・status・merge-order のテストが出力の変化で壊れないこと
- Edge cases: 既定のブランチの名前に `/`(gh api のパスで escape する)、classic と ruleset の両方、`gh api` の 404 の本文の message が想定外(`unknown`)
- Evidence to capture: `docs/evidence/test-2026-10-14-org-merge-guarantee-*.txt`

## Risks and mitigations

- doctor が遅くなる: gh がなければ打たない。打つときは全体で 20 秒の期限。オフラインでは最初の gh で切れて `info`
- 保護を誤って `github_protection` と言う: 読めないものは `unknown` にし、推測で `github_protection` にしない。gh には origin の host を `--hostname` で渡し、読む側は今の origin の host と owner/repo と比べる(C7)
- テストが本物の gh を打つ、repo の台帳に書く: TestMain で seam を固める(AC8)
- キャッシュが古くなる: 24 時間で `unknown` にし、doctor は `unknown` でも書き、書けなければ消す(C5)
- ssh の別名の origin(`github.com.emu` など)では gh が失敗して `unknown` になる: detail で言い、tech-debt に書く。別名から実名に origin を書き換えても `unknown` になるが、doctor を打ち直せば直る(文書に書く)
- repo の外(`--state-dir` だけ)で打った status は origin を読めず `unknown`: 理由を言う
- fork の origin では fork の保護を読む: detail と文書に「origin の repository の既定のブランチ」と書く

## Rollout or rollback notes

- doctor に検査が 1 つ増え、台帳に 1 ファイル増える。status と merge-order の出力に 1 行と JSON の key が 1 つ増える。ロールバックはこの PR の revert
- release のノート: doctor が gh で保護ブランチを読む(ネットワーク、最大 20 秒)、`merge-guarantee.json`、status と merge-order の `merge guarantee:` の行

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult-plan-guarantee、Codex の計画助言の HIGH 2 件を C5・C7 として取り込んだ)
- [x] Plan approved(ユーザーの「作成したプランは自動承認します」(2026-10-11)により自動承認)
- [x] Branch created
- [x] Implementation started
  - S1 f6bbd49c(`internal/org/mergeguarantee.go`: 判定、キャッシュの読み書き。implementer の変異 13 件はすべてテストで落ちた)
  - S2 e3f3403b(doctor の Check 11c `Merge guarantee`、seam `doctorMergeGuaranteeEnv` を TestMain で固定。台帳のない repo では書かないことを実機の doctor で確認)
  - S3 の決めごと: merge-order は分岐がなく ls-remote を打たないときも保証の行を出す(既定のブランチの行がないので先頭に出す)。キャッシュがなければ git は打たない
  - S3 0e3813dc(status と merge-order の `merge guarantee:` の行と `merge_guarantee`。MergeOrder は 24 時間の判定に org の時計 `o.Now` を使う。変異 10 件はすべてテストで落ちた)
  - S4 d7da3441(README、`/org` skill の 4 面、仕様 FR-7 の 7c-2 の注、AGENTS.md、tech-debt 6 行)
  - 計画からのずれ: Scope と AC3 の pr_checks_only の勧め「Require branches ... と必須の検査、または merge queue」は、判定が merge queue を ruleset の `merge_queue` の規則からしか読まないので、「merge queue は ruleset に足す」と言うようにした(89b981d8 で detail と help、c56c1f11 で skill と tech-debt。classic の保護の merge queue を読まないことは tech-debt に書いた)
  - 実機の証跡: `docs/evidence/merge-guarantee-livefire-2026-10-14.md`(この repo は warn、github/docs は pass、cli/cli と ssh の別名は unknown、台帳がなければ書かない)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
