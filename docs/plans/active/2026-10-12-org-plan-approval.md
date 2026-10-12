# org-plan-approval

- Status: Approved
- Approved: 2026-10-12 sha256:951cf09bd7bc
- Owner: Claude Code
- Date: 2026-10-12
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 7 段目(FR-7 承認と統合検証、FR-8 の merge してよい順)の前半。6 段目は PR #222 と #223(225dff27)で merge 済み。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-plan-approval

## Objective

機能の org で、人が承認していない計画のまま実装が始まらないようにする。`[org.approval].mode` が `per-plan`(既定)のとき、`ralph org start --plan` で立てた org では、人が承認した機能の計画がなければ implementer の spawn と implementer への TASK を断る。承認は人が `ralph org approve` で台帳に記録する。`split` では今までどおり、分割計画の承認だけで進む。

あわせて、4 段目から持ち越した検査を入れる。機能の org で leader の座席が動いていないとき、leader 以外の座席の spawn を断る。

7 段目の後半(7b、次の PR)には、PR の前の rebase とゲートの取り直し、検証した head と base の SHA、merge してよい順、`ralph doctor` の保護ブランチの警告、`ralph status` の保証の表示を残す。

## Scope

- 設定 `[org.approval].mode`
  - 値は `"per-plan"`(既定)か `"split"`。それ以外と空文字は `Load()` の検証エラー。節がなければ既定のまま
  - `internal/config/config.go` の `Default()`、`templates/base/ralph.toml`、`scripts/ralph-config.sh`(と `templates/base/scripts/ralph-config.sh`)の 3 面を揃え、`defaults_sync_test.go` で固定する(6a の `[org.director]` と同じ形)
  - 読むのは `ralph org start --plan` だけで、全 org の上限と同じ規則で読む(`--config` がなく台帳が main worktree のものなら、main worktree の `ralph.toml`。`withMainWorktreeOrgLimits`)。読んだ値は結びつき(`FeatureBinding`)に `approval=<mode>` のトークンとして記録し、検査はいつも記録した値を読む。spawn・send・approve は自分の設定の mode を読まない(Design decisions の A8)
  - 結びつきのトークン `approval=` は既存のトークンの後ろに置く(古い版は知らないキーで読むのをやめて note とみなすので、ほかのトークンは今までどおり読める)。トークンのない記録(この版より前の start)は split として扱う。トークンが 2 つある、値が `per-plan` でも `split` でもない記録は、承認を求める側に倒して断る
- 承認の記録 `ralph org approve <受信箱の件の ID>...`
  - 人(8a 段目からは、人の答えを受けた director)が打つ。leader は承認を受信箱の QUESTION で頼み、本文のヘッダに `PLAN: <worktree の中のパス> sha256:<digest>` を書く(図解ページのパスは `VISUAL:` に書くよう雛形で求めるが、approve は読まない)
  - approve は ID ごとに次を確かめる。件があり、resolved でなく、TYPE が QUESTION。本文を `protocol.Parse` で読み、`PLAN:` のヘッダが 1 行だけあり、値がパスと `sha256:<12 桁>` の 2 語。件の org が走っていて、台帳に完全な結びつき(`ActiveFeature` の `Complete()`)がある。パスは worktree の中の相対パスで(絶対パスと `..` を断る)、symlink を解いても結びつきの worktree の下にある普通のファイル。今のファイルの digest(`PlanDigest`)がヘッダの digest と一致する
  - 1 つでも通らなければ、どの件も記録せずに終了コード 1 で終わり、通らなかった件と理由を並べる(まとめての承認は、人がまとめて決めたものなので、一部だけを記録しない)
  - すべて通れば、manifest のロックの下で件ごとに `plan_approved` を 1 行足す(org 全体の event で seat_id は空。Details は `plan=<パス> digest=<12 桁> content=<64 桁> inbox=<ID>` に続けて打った側の情報 `by=<USER> herdr_pane=<set|unset> cwd=<引用したパス>`)。`content` は、そのときのファイルの全文のハッシュ(下の「全文のハッシュ」)。そのあと件を resolve する(note は `approved <パス> sha256:<digest>`)。resolve に失敗しても承認の記録は残り、終了コード 1 で `ralph org inbox resolve` の打ち方を出す
  - 出力は件ごとに `approved <ID> (org=<org> plan=<パス> sha256:<digest>)` と、leader に知らせる `ralph org send --to leader` の DECISION の打ち方。approve 自身は herdr も agmsg も呼ばない
  - 環境の `HERDR_PANE_ID` が台帳の active な座席の pane と同じなら、何も読まずに断る(座席の pane から打たれた approve)。迂回を防ぐ境界ではなく、座席がうっかり打つのを止める仕掛け(Design decisions の A2)
  - `--org-id` は断る(件の ID は全 org で共有する受信箱の ID で、org は件から決まる。inbox の動詞と同じ)。`--state-dir` と `--config` は受け付ける
  - 件の org の結びつきに記録した承認方式が `split` のときも記録はする(検査は読まない)。stderr に一言出す
- 承認の検査(org 層)
  - 対象は結びつきのある org(`start --plan` で立てた org)だけ。`ralph org start <task>` と昇格した leader の org は、mode を問わず検査しない
  - 判定の関数(たとえば `checkPlanApproved`)は、org の結びつき・台帳・計画のパス・digest から次をすべて確かめる。この org の今の命(最後の `disbanded` より後)に、そのパスと digest の `plan_approved` がある。今のファイルの digest がその digest と一致する。今のファイルの全文のハッシュが、その記録の `content` と一致する。ファイルの見出しの前(最初の `## ` より前)の最初の `- Status:` が `Approved` で、最初の `- Approved:` の `sha256:` の値がその digest と一致する(`/implement` の 4 手目と同じ読み方)
  - 全文のハッシュ(たとえば `PlanContentHash`、SHA-256 の 64 桁): `PlanDigest` と同じ行の読み方で、飛ばすのは見出しの前の最初の `- Status:` と最初の `- Approved:` の 2 行だけ、行頭の `- [x]` / `- [X]` は `- [ ]` と読む。`PlanDigest` が読まない `## Progress checklist` の節、`- Branch:` の行、見出しの後ろの `- Status:` / `- Approved:` の行も読む。approve のあとに leader が書いてよいのは、先頭の 2 行と checkbox の印だけになる(Design decisions の A9)。人に見せる digest(QUESTION、`- Approved:`、TASK の `PLAN:`)は `/plan` と同じ `PlanDigest` のまま
  - `Send`: TYPE が TASK で(`--raw` も 6b と同じく 1 行目の TYPE)、宛先の org の結びつきに記録した承認方式が `per-plan` で、宛先の座席の role が `implementer` のとき、TASK の `PLAN:` のヘッダ(1 行だけ)の パスと digest で判定する。見張りの検査のあと、dry-run の枝の前に置く。宛先の座席を台帳で引くのはこの場合だけで、座席がなければ本番と同じ「seat not found」で断る。断ったときは manifest に何も書かず、herdr も agmsg も呼ばない(6b の F1 と同じ)
  - `Spawn`: 結びつきのある org で、role が leader でない座席は、active な leader の座席がなければ断る(mode を問わない)。結びつきに記録した承認方式が `per-plan` で role が `implementer` の座席は、この org に判定の関数を通る承認が 1 つ以上なければ断る。どちらも台帳を読む検査なので、今の `spawnCapacityErr` のあと(ロックの下)に置き、dry-run も同じ順で予測する。拒否は今の容量と予約の拒否と同じく `reject()` で `rejected` と receipt を書く。`spawnPrecheckErr` にも同じ順で入れる(leader の spawn には当たらない)
- leader の雛形と start が渡すタスク
  - `featureLeaderTask` に `- 承認方式: per-plan` か `- 承認方式: split` の行を足す(結びつきに記録する値と同じ)
  - `internal/org/prompts/leader.md` の「機能ごとの org」を書き換える。per-plan のときは、計画を書いてコミットし、図解ページを `/plan` skill の `diagrams.md` と `visual-template.html` に沿って worktree の `.harness/state/plan-visual/<slug>.html` に書き、`scripts/plan-visual.sh digest` の値で `PLAN:` と `VISUAL:` のヘッダを付けた QUESTION を escalate し、返事を待つ。承認の DECISION を受けたら、計画の先頭に `- Status: Approved` と `- Approved: <日付> sha256:<digest>` を書き、digest が変わっていないことを確かめてコミットし、implementer を spawn する。承認のあとは、計画のほかの行を変えない(checkbox の印は付けてよい。進み具合のメモは計画に書かない)。implementer への TASK にはいつも `PLAN:` を付ける。直しを求める DECISION なら、直してコミットし、ページを描き直し、新しい QUESTION を上げる。reviewer は承認の前に spawn してよい(計画のレビューを頼める)。split のときは今の手順のまま
  - 承認の検査で断られたら、打ち直さず、`--raw` にせず、自分で approve を打たず、BLOCKED を escalate する
- 文書: `/org` skill の 4 面(approve の行、機能ごとの org の承認の節)、`.claude/rules/ralph/agent-messaging.md` と template の写し(per-plan の TASK の `PLAN:` ヘッダ)、README、仕様 FR-7 の「7 段目の前半で決めたこと」、AGENTS.md の repo map(承認の新しいファイル)、`docs/tech-debt/README.md`

## Non-goals

- 7b の項目(PR の前の rebase とゲートの取り直し、検証した head と base の SHA、merge してよい順、`ralph doctor` の保護ブランチ、`ralph status` の保証の表示)
- 分割計画の承認を台帳に記録すること、director が自分の書いた分割計画を承認するのを防ぐこと(director が入る 8a 段目で決める。今は 4 段目のまま、人が分割計画の 2 行を書く)
- approve が leader に DECISION を送ること(approve は herdr も agmsg も呼ばない。8a 段目の director か人が送る)
- 機能ごとに承認方式を変えること(仕様の Out of scope)
- 結びつきのない org(`start <task>`、昇格)に承認を求めること
- leader の図解ページを ralph が検査すること(ページがあるか、中身が計画と合うか)
- 承認の取り消し(`plan_revoked` など)。計画を変えれば digest が変わり、承認は効かなくなる

## Assumptions

- 機能の計画は leader が worktree の中に書き、コミットする。計画の雛形(`/plan` の template)は先頭に `- Status:` と `- Approved:` の行を持つ。雛形のないプロジェクトでも、leader は先頭にこの 2 行を書く(雛形で求める)
- `scripts/plan-visual.sh` は `ralph init` が出荷するので、leader の worktree にある。digest は Go の `PlanDigest` と同じ値になる(`TestPlanDigest_MatchesScript`)
- 受信箱の件の本文は message の全文(`InboxItem.Body`)なので、approve は `protocol.Parse` でヘッダを取り直せる
- `HERDR_PANE_ID` は herdr の pane の中で herdr が設定し、座席の記録の `pane_id` と同じ形(`Stop` が自分の pane を見分けるのに使っている)
- leader が自分で `env -u HERDR_PANE_ID ralph org approve` を打つこと、台帳のファイルを直接書くことは止められない(leader は台帳に書ける)。承認の記録に打った側の情報を残し、事後に追えるようにする

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`、`internal/config/defaults_sync_test.go`、`templates/base/ralph.toml`、`scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh`
- 新しい `internal/org/approval.go`(`PLAN:` のヘッダの読み方、計画のパスの解決、承認の記録の読み書き、判定の関数、`Org.Approve`)と `internal/org/approval_test.go`
- `internal/org/verbs.go`(`Send` の検査)、`internal/org/spawn.go`(`Spawn` の検査、dry-run、`spawnPrecheckErr`)、`internal/org/feature.go`(`featureLeaderTask` の承認方式の行、結びつきに承認方式を入れる)、`internal/org/reserve.go`(`FeatureBinding` の承認方式とトークン)、`internal/org/seat.go`(event の定数)、`internal/org/split.go`(全文のハッシュを `PlanDigest` の隣に置くなら)
- 新しい `internal/cli/org_approve.go` と `internal/cli/org_approve_test.go`、`internal/cli/org.go`(サブコマンドの登録、`withMainWorktreeOrgLimits` が承認方式も main worktree から読む)
- 雛形: `internal/org/prompts/leader.md`、`internal/org/prompts_test.go`
- 文書: `.claude/skills/org/SKILL.md` と写し 3 面、`.claude/rules/ralph/agent-messaging.md` と `templates/base/.claude/rules/ralph/agent-messaging.md`、README、仕様、AGENTS.md、`docs/tech-debt/README.md`

## Visual review

- Page: `.harness/state/plan-visual/org-plan-approval.html`(図 1 全体図、図 2 per-plan の承認の流れ、図 3 検査が効く範囲、図 4 判定の関数が照らし合わせる値)
- Self-check: 全体のページ、全体図、図 2、図 4 を `scripts/plan-visual.sh shot` で撮って読んだ。図 2 の ① の枠から等幅の文字がはみ出していたのでパスを短くし、図 4 の上の 4 つの枠を高くして撮り直した。どれもきれい
- Codex advisory: 指摘 2 件(HIGH 1: 承認方式を打つ側の ralph.toml から読むと検査が消える、MEDIUM 1: PlanDigest が読まない場所に指示を足せる)。ユーザーの指示どおり聞かずに A8・A9 で計画を直し、図 1・3・4 を描き直して撮り直した。どれもきれい

## Design decisions

critical forks は consult(consult-plan-stage7)に聞いて自分で選んだ(ユーザーの指示「計画の分かれ目も自分で選ぶ」)。

- A1 7 段目を 2 つの PR に分ける。7a は承認(この計画)、7b は統合検証。6 段目の前例どおりで、触る場所もほぼ別。7a のあと 7b までの間、PR の統合検証は今のまま(PR を作った時点の検査だけ)
- A2 承認は台帳に記録し、検査はファイルと台帳の両方を見る。計画を書くのは leader 自身なので、ファイルの 2 行だけで承認とみなすと、leader が 2 行を書けば自分で承認できる。`ralph org approve` が台帳に `plan_approved` を書き、検査は「台帳の digest = 今のファイルの digest = `- Approved:` の digest、`- Status:` が Approved」を求める。approve は座席の pane から打たれたら断るが、`HERDR_PANE_ID` は外せるし、leader は台帳に書けるので、これは境界ではない。仕様の Security considerations の「迂回できない」は字義どおりには満たせず、記録に打った側の情報を残して監査(ユーザーストーリー 5)に回す。この読みを仕様の注と PR の本文に書く。退けた案: ファイルの 2 行だけ(leader の自己承認を止められない)
- A3 approve は台帳にだけ書き、計画のファイルには書かない。ファイルの 2 行は、承認の DECISION を受けた leader が書いてコミットする(`/plan` と `/implement` と同じ形を保つ)。退けた案: approve がファイルに 2 行を書く(走っている座席の作業場所を外から書き換え、未コミットの変更を残し、leader が計画を編集しているとぶつかる)
- A4 まとめての承認は受信箱の件の ID で行う(`ralph org approve e3 e5`)。approve は依頼のヘッダの digest と今のファイルの digest の一致を確かめるので、人が見た依頼と承認する中身がずれない。1 つでも通らなければ何も記録しない。退けた案: `--org-id` と `--plan` と `--digest` を org ごとに打つ(まとめられず、人が digest を写す)
- A5 検査は結びつきのある org だけ、per-plan の implementer の spawn と implementer への TASK だけに効く。reviewer への TASK は承認の前でも通す(計画のレビューを頼める)。split では承認を読まない(仕様の受け入れ条件 108 行)
- A6 spawn の承認の拒否と、leader のいない org の座席の拒否は、`rejected` と receipt を書く(今の容量・予約の拒否と同じ検査の並び)。6b の見張りの拒否(何も書かない)と違うのは、この 2 つは台帳の中身で決まる方針の拒否で、立っている座席の打ち直しは今の冪等の戻り(`idempotentRespawn`)が先に返すので、`rejected` が active な座席の状態を上書きしないため。TASK の拒否は何も書かない(`Send` には `rejected` がない)
- A7 TASK の `PLAN:` のヘッダが 2 行あるときは断る(`protocol.Parse` は後の行で上書きするので、検査と implementer が読む行がずれうる)。値はパスと `sha256:<12 桁>` の 2 語に限り、空白を含むパスは使えない
- A8(Codex plan advisory の指摘 1 で直した): 承認方式は start の時点で結びつきに記録し、検査はその値を読む。設定を検査のたびに読むと、leader が打つ spawn と send は feature worktree の `./ralph.toml`(ブランチの写しで、leader が書き換えられる)を読むので、そこが `split` なら承認の検査が黙って消える。start は全 org の上限と同じ規則で読む(`--config` がなければ台帳の main worktree の `ralph.toml`)。設定を変えても、走っている org の方式は変わらない(新しい org から効く)。同じ機能の start の打ち直しで方式が違えば、結びつきが違うとして断る(disband して立て直す)。トークンのない古い記録は split として扱う(その org の leader の雛形には承認の手順がない)。退けた案: 検査のたびに main worktree の設定を読む(`--config` を付けた leader の send は自分の設定を読むことになり、規則が 2 つに分かれる)
- A9(Codex plan advisory の指摘 2 で直した): approve はファイルの全文のハッシュも記録し、検査は今のファイルと照らす。`PlanDigest` は `## Progress checklist` の節と `- Status:` / `- Approved:` / `- Branch:` の行をどこにあっても読まないので、承認のあとにそこへ実装の指示を足しても digest は変わらない。全文のハッシュが飛ばすのは先頭の 2 行と checkbox の印だけ。人に見せる digest は仕様の「`/plan` と同じ digest」のとおり `PlanDigest` のままにし、全文のハッシュは ralph の中だけで使う。QUESTION を上げてから approve を打つまでの間に飛ばされる場所が書き足されると、approve はその中身ごと記録する。approve の出力と skill に「approve はその時点のファイルを記録する」と書く。退けた案: 人に見せる digest を厳しい方に替える(仕様と `/plan` の digest から離れ、leader が `scripts/plan-visual.sh` で出せない)、飛ばす場所の中身を構文で縛る(`/plan` の雛形の Progress checklist はメモの行を持つ)

## Acceptance criteria

- [ ] AC1: `[org.approval].mode` は既定で `per-plan`、`split` も受け付け、それ以外の値と空文字は `Load()` のエラーになる。節がなければ既定のまま。`Default()`・`templates/base/ralph.toml`・`scripts/ralph-config.sh` の値が一致し、`defaults_sync_test.go` が固定する。設定を読むのは `start --plan` だけ(AC13)
- [ ] AC2: `ralph org approve <ID>` は、PLAN のヘッダの digest が今のファイルの digest と一致する QUESTION の件について、manifest に `plan_approved`(パス、digest、件の ID、打った側の情報)を 1 行書き、件を resolve し、`approved <ID> (...)` と leader への DECISION の打ち方を出して終了コード 0 で終わる。計画のファイルは書き換えない
- [ ] AC3: approve は、知らない ID、resolved の件、QUESTION でない件、`PLAN:` がない・2 行ある・形が違う件、結びつきのない・不完全な org の件、worktree の外のパス(絶対パス、`..`、worktree の外を指す symlink)、ない・普通のファイルでないパス、ヘッダの digest と今のファイルの digest が違う件を、理由を添えて断る。複数の ID の 1 つでも断られたら、どの件も記録せず resolve もせず終了コード 1
- [ ] AC4: 環境の `HERDR_PANE_ID` が台帳の active な座席の pane と同じとき、approve は何も書かずに終了コード 1 で断る。`HERDR_PANE_ID` がないとき、どの座席の pane とも違うときは通る
- [ ] AC5: per-plan で結びつきのある org の implementer への TASK は、`PLAN:` がない、承認の記録がない、承認のあとに計画が変わった(digest が違う)、ファイルの `- Status:` が Approved でない、`- Approved:` の digest が違う、のどれでも、何も記録せず herdr も呼ばずに終了コード 1 で断られる(`--raw` と `--dry-run` も)。すべて揃えば今までどおり送れる。承認し直したあとは、新しい digest の `PLAN:` で送れる
- [ ] AC6: 同じ org の reviewer への TASK、TASK 以外の TYPE、結びつきのない org の TASK、split の org の implementer への TASK は、承認がなくても今までどおり送れる(仕様の受け入れ条件 108 行)
- [ ] AC7: per-plan で結びつきのある org では、判定を通る承認がないと role `implementer` の spawn は `rejected` を書いて断られる(`--dry-run` も同じ答え)。承認があれば進む。split では承認なしで進む
- [ ] AC8: 結びつきのある org で active な leader の座席がないとき、leader 以外の spawn は mode を問わず `rejected` を書いて断られる。leader の spawn(`start --plan`)と、結びつきのない org の spawn は今までどおり
- [ ] AC9: 前の命の承認は効かない。org を disband して同じ org_id で立て直すと、前の `plan_approved` では TASK も spawn も通らない
- [ ] AC10: `start --plan` が leader に渡すタスクに `- 承認方式:` の行があり、leader の雛形は per-plan の手順(計画のコミット、図解ページ、`PLAN:` と `VISUAL:` の QUESTION、DECISION 待ち、2 行のコミット、implementer の spawn、TASK の `PLAN:`)と split の手順、承認の検査で断られたときに BLOCKED を escalate することを書き、`prompts_test.go` がその語句を固定する。今の固定の語句は崩れない
- [ ] AC11: org 層と CLI の既存のテストは、書き換えずに通る(承認の検査は結びつきのある per-plan の org だけに効くので、既存の機能の org のテストで implementer を spawn・TASK するものは、必要なら承認の記録を足す。書き換えた本数と理由を Progress checklist に書く)
- [ ] AC12: 文書(skill の 4 面、agent-messaging の 2 面、README、仕様の注、AGENTS.md、tech-debt)が新しい動きに合い、`scripts/check-sync.sh` と `scripts/check-skill-sync.sh` が通る
- [ ] AC13: `start --plan` は承認方式を結びつきの `approval=` に記録し、`--config` がなければ台帳の main worktree の `ralph.toml` の値を使う。per-plan で立てた org の未承認の TASK は、send を打つ側の `./ralph.toml`(feature worktree やサブディレクトリ)と `--config` が `split` でも断られる。split で立てた org は、打つ側の設定が `per-plan` でも承認なしで送れる。`approval=` のない古い記録の org は split として扱い、値の読めない記録(2 つある、知らない値)の org は承認を求める
- [ ] AC14: approve のあとに計画の `## Progress checklist` の節に行を足す、`- Branch:` の行を変える、見出しの後ろに `- Status:` の行を足す、のどれでも、implementer への TASK と spawn は断られる(`PlanDigest` は変わらない)。先頭の 2 行を書く、checkbox に印を付ける、だけなら通る

## Implementation outline

1. S1 設定と結びつき: `[org.approval].mode` と 3 面の lock-step、検証、`FeatureBinding` の承認方式とトークン `approval=`、`start --plan` が記録すること(`withMainWorktreeOrgLimits` が承認方式も main worktree から読む)、`featureLeaderTask` の行、テスト
2. S2 org 層の承認: `approval.go`(ヘッダの読み方、パスの解決、計画の見出しの読み方、全文のハッシュ、記録の読み書き、判定の関数、`Org.Approve` と tripwire)、`plan_approved` の定数、テスト
3. S3 org 層の検査: `Send` の TASK の検査、`Spawn` の 2 つの検査(dry-run と `spawnPrecheckErr` を含む)、テスト(既存のテストの直しを含む)
4. S4 CLI: `ralph org approve` の配線と出力、`--org-id` の拒否、設定の食い違う台帳の CLI のテスト(AC13)、CLI のテスト
5. S5 雛形と文書: leader の雛形と `prompts_test.go`、skill の 4 面、agent-messaging の 2 面、README、仕様の注、AGENTS.md、tech-debt

各スライスは implementer に渡し、1 スライス 1 コミット。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint)、`scripts/check-sync.sh`、`scripts/check-skill-sync.sh`
- Spec compliance criteria to confirm: FR-7 の 1〜4 項目(承認方式の設定、split、per-plan の spawn と TASK の拒否と `PLAN:` の照合と改訂のときの検査、承認の記録を director が代行しないこと)と受け入れ条件 107・108 行を AC と対応づける。5・6 項目と FR-8 は 7b に送ったと書く。Security considerations の「迂回できない」の読み(A2)を仕様の注に書く
- Documentation drift to check: skill の 4 面の `spawn`・`send`・`start` の行と機能ごとの org の節、agent-messaging の TASK の説明、README の org runtime の節、仕様 FR-7 の注、AGENTS.md の repo map
- Evidence to capture: 一時の台帳での実機確認(機能の org で、承認なしの implementer の spawn と TASK の拒否、approve のあとの成功、計画を変えたあとの拒否、座席の pane からの approve の拒否)を `docs/evidence/` に

## Test plan

- Unit tests: `PLAN:` のヘッダの読み方(1 行、2 行、形の違い、digest の形)、パスの解決(相対、絶対、`..`、外を指す symlink、ないファイル、ディレクトリ)、計画の見出しの読み方(Status と Approved が見出しの前にある・ない・後ろにある)、全文のハッシュ(飛ばす 2 行、checkbox の印、Progress checklist と `- Branch:` と見出しの後ろの `- Status:` を読むこと)、判定の関数(記録なし、digest 違い、全文のハッシュ違い、Status 違い、前の命の記録)、結びつきの `approval=` の読み書き(あり、なし、2 つ、知らない値、古い版が読めること)、`Org.Approve`(まとめての承認、1 つ断られたとき、resolve の失敗、tripwire)、設定の検証
- Integration tests: CLI の `ralph org approve`(出力、終了コード、`--org-id` の拒否、`--state-dir`)。CLI の send と spawn で、機能の org の承認なし・ありの往復。main worktree と feature worktree とサブディレクトリの `ralph.toml` の mode が食い違う台帳で、start が main の値を記録し、send がどこから打っても記録した値で判定すること(AC13)
- Regression tests: org 層と CLI の既存のテスト、`TestRenderRolePrompt_*`、`TestPlanDigest_MatchesScript`、`defaults_sync_test.go`
- Edge cases: `--raw` と `--dry-run` の TASK、宛先の座席がない TASK、結びつきの不完全な org、split の org の approve、承認のあとの計画の改訂と承認し直し、disband のあとの同じ org_id、reviewer への TASK、leader が止まった org の reviewer の spawn
- Evidence to capture: `go test ./...`、`-race`(org と cli)、mutation(検査の条件、tripwire、まとめての承認の all-or-nothing)、実機確認

## Risks and mitigations

- 既存の機能の org のテストが落ちる: 承認の検査は per-plan の結びつきのある org の implementer だけに効く。既存のテストで当たるものは、承認の記録を足すか split にする。本数と理由を記録する
- leader が承認を待ったまま止まり、人が気づかない: leader は QUESTION を escalate し、escalate はすぐ人へ送り、6a の ack と resolve の期限でも届く
- 人が approve のあと leader に知らせ忘れる: approve の出力に DECISION の打ち方を出す。知らせなくても leader は待つだけで、何も壊れない。8a 段目で director が送る
- 計画を直すたびに承認し直しになる: 仕様どおり(「計画の改訂のときも同じ検査をする」)。checkbox の tick は digest を変えない
- start のあとで設定の mode を変えても、走っている org には効かない: 方式は start の時点で結びつきに記録し、leader のタスクの `- 承認方式:` と検査は同じ値を使う(A8)。新しい方式は次の start から効く。README と skill に書く
- 古い版で立てた org(`approval=` のない記録)は承認なしで進む: その leader の雛形には承認の手順がないため split として扱う。release のノートに、更新の前に走っている機能の org を終わらせるか、更新のあとに立て直すと書く
- 承認のあとに leader が計画にメモを書くと止まる: 全文のハッシュが変わるので TASK と spawn が断られる(A9)。雛形で「承認のあとは先頭の 2 行と checkbox の印のほかは変えない」と書き、断られたら BLOCKED を上げる
- 打った側の情報(USER、cwd)が台帳に残る: 台帳は git の外で、利用者の手元にある。`ralph org report` の表にも出る(監査の目的どおり)
- tripwire の誤検知: 人が herdr の pane の中で approve を打ち、その pane の id が別の herdr server の座席の pane と同じとき断られる。エラーに、herdr の外の端末で打つよう書く

## Rollout or rollback notes

- release のノート: 既定の `per-plan` では、`start --plan` で立てた org の implementer の spawn と TASK に、人の承認(`ralph org approve`)が要る。今までの動き(承認を待たない)にするには `[org.approval] mode = "split"`。方式は start の時点で決まり、走っている org には設定の変更が効かない。新しいサブコマンド `ralph org approve`、manifest の新しい event `plan_approved`、結びつきの新しいトークン `approval=`。更新の前に立てた機能の org は承認なしで進む
- revert すると検査と approve がなくなり、4 段目の動き(leader が承認を待たずに進む)に戻る。台帳に残った `plan_approved` は古い版では読まれない(Roster は seat_id の空の event を飛ばす)。結びつきの `approval=` は古い版では note として読まれ、ほかのトークンは今までどおり読める

## Open questions

- なし(critical forks は Design decisions に記録する)

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

### Notes

- 2026-10-12: 7 段目を 7a(この計画)と 7b に分けること、承認を台帳に記録して計画のファイルと両方で検査すること、受信箱の件の ID でまとめて承認することは consult(consult-plan-stage7)の判定で決めた。approve がファイルを書く案は consult の指摘で外した(A3)
- 2026-10-12: Codex の計画助言の 2 件(HIGH・MEDIUM)を A8・A9 で直した。計画はユーザーの指示(「作成したプランは自動承認します」)で自動承認した
- 2026-10-13: 5 つのスライスを implementer に渡し、1 スライス 1 コミットで入れた。S1 f779cc56(設定と結びつき)、S2 c08a1a14(org 層の承認と判定)、S3 e3d34062(spawn と TASK の検査)、S4 5596561a(`ralph org approve`)、S5 36b9d3a3(leader の雛形と文書)
- 2026-10-13: 逸脱は 1 件。S1 の implementer が `internal/cli/org_feature_test.go` に `gofmt -w` を 1 回かけた(桁揃えだけで、中身は変わらない)。S2 以降は使わせていない
- 2026-10-13: AC11 の書き換えた既存のテスト。S1 は 11 本と共有の helper `testFeature`(結びつきに承認方式の欄が増え、`String()` と照合の期待が変わったため)。S3 は 3 本(`feature_test.go` の 2 本は per-plan の org で implementer に TASK を送るので承認の記録を足した。`spawn_feature_test.go` の 1 本は leader が動いている状態で seat-1 を spawn するよう直した)。S4 は 0 本。S5 は `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の固定の語句 2 つ(手順の番号が 1〜14 に繰り下がったため)
- 2026-10-13: self-review の 1 回目(報告は 3d73dbf4、MEDIUM 1 件と LOW 11 件)の指摘は、consult(consult-plan-selfreviewfix)の判定に従って verify の前に直した。54feb287 は approve の動詞を別のファイルに移しただけで、中身は変えていない(L4)。9fb3fafa は M1・L1〜L3・L5〜L11 の直しで、8ad54167 は seat.go のコメントの参照を approve.go に直した。cross-review より前の直しなので、cycle は 1 のまま self-review からやり直す
- 2026-10-13: 逸脱 1。新しいファイル `internal/org/approve.go` と `internal/org/approve_test.go` を作った。Affected areas(69 行目)と実装の概略の S2(115 行目)は `Org.Approve` を approval.go に置くと書いている。approval.go が 853 行になったので、approve の動詞(`ApproveParams` から `resolveApproved` まで)とそのテストを分けた(L4)。approval.go は 635 行、approve.go は 408 行になった
- 2026-10-13: 逸脱 2。承認の検査で断られたときの次の手を、断られた理由ごとに 1 つにした(M1、9fb3fafa)。Scope(45 行目)、Risks(145 行目)、AC10 は「BLOCKED を escalate する」と書いているが、理由の文と次の手が食い違っていたため直した。今は、どの拒否にも目印 `refused by the approval check` が付き、その後ろに次の手が 1 つ続く。次の手は次の 5 つ
  - 承認の記録がない場合: 承認の DECISION をまだ受けていなければ返事を待つ。受けていれば BLOCKED を上げる
  - 計画の先頭の 2 行が違う場合: 2 行を書いてコミットし、打ち直す
  - 承認のあとに計画が変わった場合: 変更を戻すか、Draft に戻して新しい QUESTION を上げる
  - TASK の `PLAN:` がないか、形が違う場合: `TASK_ID:` の次の行に足して送り直す
  - それ以外: BLOCKED を上げる
- 2026-10-13: 逸脱 2 の続き。拒否の文は「leader は自分で approve を打たない」で終わる。`--raw` を禁じる一文は TASK の拒否にだけ付く。雛形の手順 7 は「目印の後ろの次の手に従う」に書き直した。AC10 の「BLOCKED を escalate する」は、記録がない場合の後半と、それ以外の場合の次の手として読む。tech-debt の「Small leftovers of org-plan-approval」の行の (d)(spawn の拒否が `--raw` に触れる)は、この直しで閉じた
- 2026-10-13: 逸脱 3。`- Approved:` の行の形を厳しくした(L2、9fb3fafa)。最初の `##` の見出しより前にある最初の `- Approved:` が、実在する日付の `YYYY-MM-DD` と `sha256:<12 桁>` の 2 語でなければ、digest がないものとして扱う。docs/plans の 22 本では、21 本がこの形だった。残りの 1 本はテンプレートの `- Approved: TBD` である
- 2026-10-13: self-review の 2 回目(報告は 02019f48、MEDIUM 1 件と LOW 4 件、Merge: yes)の指摘は、consult(consult-plan-selfreviewfix)の判定に従って直した。どちらも cross-review より前なので、cycle は 1 のまま
  - 4b6d0d55 で N1・N2・N3・N5 を直した。N1 は、計画を承認し直したあとに古い digest か打ち間違えた digest の `PLAN:` で TASK を送ると、次の手が「変更を戻す」か「返事を待つ」になっていた件。今の計画が承認済みなら、`PLAN:` をその digest に直す拒否にした
  - 0326574e で、implementer が見つけた隙間を閉じた。承認し直したあと、先頭の 2 行がまだ古いうちに古い `PLAN:` で送った場合は、新しい digest で 2 行を書かせる拒否にした。2 行を書いて送り直すと `PLAN:` を直す拒否になり、新しい digest で送れば通る
- 2026-10-13: 逸脱 2 の続き。逸脱 2 で挙げた 5 つの次の手のうち、`PLAN:` の次の手は 2 つに分かれた。ないか形が違えば `TASK_ID:` の次の行に足す。今の承認済みの計画と違う digest なら、その digest に直す
- 2026-10-13: 逸脱 4。`internal/org/approval.go` の拒否の文から、雛形の手順番号(「step 6」)を外した(N2)。雛形の手順 7 は「原因を直す前に同じコマンドを打ち直さない」に書き換えた
- 2026-10-13: N4 はコードを変えていない。2 つの fixture(`approval_test.go` の `checkFixture` と `approval_enforce_test.go` の `enforceFixture`)の `record` と `mark` は重なったままにした。9fb3fafa の本文は「L8 を直した」とだけ書いているが、まとめたのは置き換えの helper(`replaceOnce`)だけである
- 2026-10-13: 3 回目の self-review では、止める条件を HIGH 以上にする(consult の判定)。MEDIUM 以下は tech-debt の行に書いて先へ進む
- 2026-10-13: sync-docs の cycle 1(報告は `docs/reports/sync-docs-2026-10-13-org-plan-approval.md`)。verify の D1(`/org` skill の `approve` の行の括弧、4 面)と D3(仕様 FR-7 の注)を文書だけで直した。D2(`--help` の文字列)は cycle 2 の直しで扱い、D4・D5 は記録だけ。AC と checklist の印は cycle 2 で付ける
