# reviewer-runs-gate

- Status: Approved
- Approved: 2026-10-26 sha256:6dd89c5dd6d0
- Owner: Claude Code
- Date: 2026-10-26
- Related request: 機能ごとの org で、静的検査とテストを leader ではなく reviewer が担当するようにする(ユーザーの依頼「静的検査とテストもreviewerが担当するように変更してください」、2026-10-26)
- Related issue: N/A
- Type: feat
- Branch: feat/reviewer-runs-gate

## Objective

機能ごとの org(`ralph org start --plan` で立てた org)では、PR の前の最後の検証を leader が打っている。leader の雛形(`internal/org/prompts/leader.md`)の「機能ごとの org」の手順 12 で、leader が `ralph org gate` を打ち、gate が `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` を走らせて、結果を台帳に `gate_passed` か `gate_failed` として記録する。reviewer はレビューの TASK の初めに 2 つのスクリプトを自分で走らせるが、こちらは台帳に記録しない。

この計画では、記録する gate も reviewer に打たせる。leader は report と計画の archive のコミット、secret scan までを済ませてから、reviewer に gate だけを頼む TASK を送り、その結果を受けて push と PR に進む。leader は `ralph org gate` も 2 つのスクリプトも打たなくなる。

## Scope

- S1(codex の reviewer の台帳への書き込み): `internal/org/permissions.go` と `internal/org/spawn.go`
  - `codexWritableRootArgs` が `--add-dir <state dir>` を足す座席を、今の「leader」から「leader と、機能の org(manifest に結びつきがある org、`ActiveFeature`)の reviewer」に広げる。条件のほかの部分(codex で、permission mode が autonomous か edits、台帳が cwd の下にない)は変えない
  - Spawn は manifest のロックの下で読んだ events(`spawn.go` の `spawnFeatureSeatErr` と同じもの)から、機能の org かどうかを決めて渡す。新しいロックは取らない
  - 機能の org の implementer、結びつきのない org の reviewer、未知の役割、director には今までどおり足さない
  - コメント(`permissions.go` の `codexWritableRootArgs` と `CodexModeRunsUnderWorkspaceWrite`、`spawn.go` の呼び出しの前)を新しい範囲に合わせる
- S2(見張りのコメントと動詞の文言): `internal/org/watch.go`、`internal/org/regate.go`、`internal/cli/org_gate.go`、`internal/org/gate.go`
  - `leaderActivityEventCount` の数え方は変えない(D6)。(c)(`gate_passed` と `gate_failed` を leader の動きとして数える)のコメントを、「機能の org の gate は reviewer が leader の TASK で打つので、記録は leader が頼んだ仕事が進んだ印として数える」に書き直す
  - `regate.go` の断りの文「while it runs, its leader runs ralph org gate --org-id X」を、走っている間は reviewer が leader に頼まれて gate を打つ、という文にする
  - `ralph org gate` の help(`org_gate.go` の Long)と `gate.go` の冒頭のコメントで、gate を打つのを「機能の org の reviewer 座席が leader の TASK で打つ(人が打ってもよい)」にする
- S3(雛形): `internal/org/prompts/leader.md` と `internal/org/prompts/reviewer.md`
  - leader の手順 12: 自分で gate を打つ代わりに、reviewer に gate の TASK を送る。TASK_ID は `gate-<n>`(この org で gate を頼んだ回数、1 から)にし、本文に打つコマンド(`mkdir -p .harness/state` と `ralph org gate --org-id {{ORG_ID}} --state-dir <台帳> > .harness/state/gate-{{ORG_ID}}.out 2>&1`)を書く。結果が来るまで worktree を変えず(コミット・rebase・ファイルの書き換えをしない)、implementer に TASK を送らない。待ち方は「agmsg の受信箱の運用」の節のとおり。この間に見張りから reviewer の停滞の ALERT が来ても、reviewer を stop も spawn し直しもしない(`ralph org read` で gate が走っていることを確かめる)
  - 手順 12 の結果の扱い
    - `GATE: pass` の RESULT: `.harness/state/gate-{{ORG_ID}}.out` を自分で読み、`GATE: pass` の行があり、その `HEAD:` が RESULT の `HEAD:` と `git rev-parse HEAD` に同じことを確かめ、`HEAD:` と `BASE:` を控える。続けて、座席を止める前に、今の手順 14 にある `git log --format='%h %s' origin/<既定のブランチ>..HEAD` の `wip:` の確かめを済ませる(`wip:` のコミットがあれば今と同じく BLOCKED を escalate)。HEAD が合わなければ、座席を止めずに gate を頼み直す
    - `GATE: unrunnable` の BLOCKED: 本文の理由(gate の stderr)を読む。gate の断り(作業ツリーが汚れている、rebase していない、など)なら、その次の手を leader が打ってから頼み直す。implementer の手が要るとき(rebase の衝突など)は今の手順 12 と同じく archive のコミットを revert して implementer に頼む。reviewer の環境(台帳に書けない、権限)なら「ミッション」の 5 のとおり。次の手のないエラーなら escalate
    - `GATE: fail` の BLOCKED: 今と同じく archive のコミットを revert し、本文の `REASON:` と LOG のパスを implementer への TASK に書き、手順 8 の往復に戻る
  - 手順 9 の書き出しを「レビューの TASK(TASK_ID が `gate-` で始まらない TASK)への `GATE: pass` の RESULT」にする。gate の TASK の RESULT で手順 9 に戻ってコミットを足すと、gate の HEAD と push する HEAD がずれるため
  - 手順 14: push の前に、HEAD を「手順 12 で `.harness/state/gate-{{ORG_ID}}.out` から控えた `HEAD:`」と照らし、`wip:` の確かめをもう一度打つ。座席は手順 13 で止めてあるので、HEAD が違えば(leader 自身がコミットを足したときに限られる)、手順 5 と同じ flag(`--id reviewer`、`--cwd .`、`--driver`、`--model`、タスクの `- 予約したパス:` の行の `--scope`)で reviewer を spawn し直し、手順 12 から(gate の TASK を新しい TASK_ID で)やり直し、通ったら手順 13 で止め直してから手順 14 に戻る。spawn が断られたら、エラーを本文に書いて BLOCKED を escalate する(Codex の計画助言)。RESULT の escalate の例の `HEAD:`・`BASE:` の出どころの文も直す
  - 「ミッション」の 5 に、gate の TASK への BLOCKED は手順 12 のとおりに扱う、を足す。「ミッション」の 7 の「`ralph org gate` の前に打ち」を、reviewer に gate を頼む前に、にする。「機能ごとの org」の前文の「12 の計画の戻し」の参照は残す
  - 「agmsg の受信箱の運用」の節: Claude Code の項の例から `ralph org gate` を外し(`ralph org wait` だけにする)、Codex の項の「座席を待つあいだ(前面のループ、`ralph org wait`、gate)」から gate を外す
  - reviewer の雛形: 「gate の TASK」の節を足す。TASK_ID が `gate-` で始まる TASK では、差分レビューもスクリプトの素の実行もせず、TASK に書かれたコマンドをそのまま打つ。Claude Code(Bash ツールの `run_in_background` があるとき)は `run_in_background` で走らせて手番を終え、終わりの知らせのあとにファイルを読む。Codex は前面で打ち、ツールの timeout を gate にかかる時間より長くする。worktree のファイルを書き換えず、コミットもしない。結果は次のとおり返す
    - ファイルに `GATE: pass`: RESULT。ヘッダに `GATE: pass` と、ファイルの `HEAD:`・`BASE:` を写す。EVIDENCE にファイルのパスと `LOG:` のパス
    - `GATE: fail`: BLOCKED。ヘッダに `GATE: fail`、本文に `REASON:` と `LOG:` のパスと、落ちたスクリプトの出力の要点(ポインタ原則のとおり、ログを貼らない)
    - `GATE:` の行がない(gate の断り、台帳や git のエラー): BLOCKED。ヘッダに `GATE: unrunnable`、本文に stderr の理由と次の手(短く)。次の手(rebase、コミット)は reviewer が打たず、leader に任せる
    - ツールに切られたとき: 前の gate のスクリプトが終わるのを待ってから 1 回だけ打ち直す。2 回目も切られたら `GATE: unrunnable`
  - reviewer の雛形のミッションの前文: 書いてよいものに「gate の TASK で `ralph org gate` が台帳に書く記録とログ、`.harness/state/gate-<org_id>.out`」を足す。`ralph org gate` のほかの `ralph org` の動詞は打たず、台帳のファイルを直接書かない、を書く。ミッションの 4 の「ralph はこの 2 行を読まない(…leader が PR の前に打つ `ralph org gate` が台帳に残す)」の括弧を、gate の TASK で打つ `ralph org gate` が残す、にする。「レポート契約」に gate の TASK の RESULT と BLOCKED の形を足す
- S4(テスト): S1〜S3 の変更をテストで固定する
  - `internal/org/permissions_test.go` の `codexWritableRootArgs` の表に、機能の org の reviewer(codex、autonomous と edits)で `--add-dir` が付く行、結びつきのない org の reviewer と機能の org の implementer で付かない行を足す(今の「reviewer codex autonomous/edits で nil」の行は、結びつきのない org の行として残す)
  - spawn の argv のテスト: 機能の org の codex の reviewer の spawn の argv に `--add-dir` が permission の flag のすぐあと、`--model` の前に入り、implementer の argv には入らない
  - `internal/org/gate_test.go` の `leaderActivityEventCount` のテスト(gate の記録を 1 つずつ数える)は変えずに残す(D6)
  - `internal/org/regate_test.go`(:594)で断りの文を固定しているテストを新しい文にする
  - `internal/org/prompts_test.go` と `internal/org/prompts_leader_inbox_test.go`: leader の手順 9・12・14、ミッション 5・7、受信箱の節の pin を新しい文にする。leader の手順 12 が reviewer に `gate-<n>` の TASK を送ること、待つあいだ worktree を変えないこと、`.out` と `git rev-parse HEAD` を照らすこと、pass の枝で座席を止める前に HEAD と `wip:` を確かめることを固定する。手順 14 の HEAD の不一致で、手順 5 の flag で reviewer を spawn し直して手順 12 からやり直す文を固定する。leader の雛形に leader 自身が `ralph org gate` を打つ文(`ralph org gate --org-id … を打つ`)が残っていないことを確かめる。reviewer の「gate の TASK」の節の文(TASK_ID の見分け、コマンドをそのまま打つ、run_in_background と前面、3 つの返し方、打ち直しは 1 回、ほかの動詞を打たない)を固定し、例の RESULT と BLOCKED が `protocol.Parse` と `Validate` を通ることを確かめる
- S5(skill と文書)
  - `/org` skill(`.claude/skills/org/SKILL.md`): 動詞の表の `gate` の行(打つのは機能の org の reviewer 座席で、leader が TASK で頼む。人が打ってもよい)、「org の終わり方」の 4〜6(4 は reviewer に頼む形、pass のあと止める前の HEAD と `wip:` の確かめ、6 の不一致での reviewer の spawn し直し)、「gate の記録」の「見張りは leader の動きとして数える」に理由(leader が頼んだ仕事の記録)を足す、役割の reviewer の説明(gate の TASK を受ける)、leader のサイクルの 7、Codex の leader の待ちの文、codex の `--add-dir` の段落(機能の org の reviewer にも足す)
  - `/director` skill(`.claude/skills/director/SKILL.md`): 前面で待つ leader の例から gate を外す。受信箱の ALERT の件のうち、leader が gate を頼んでいる間の reviewer の停滞の件は gate が長いと起きうる、`ralph org read` で gate が走っていれば待つ、の 1 文
  - `scripts/sync-skills.sh` で両 skill の写し(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)をそろえる
  - `docs/recipes/codex-seat-permissions.md` と `templates/base/docs/recipes/codex-seat-permissions.md` の「The org ledger and `--add-dir`」節
  - `README.md` の `ralph org gate` の段落(:264)、`docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md` の Quality pipeline gate の行
- S6(記録)
  - 仕様 `docs/specs/2026-10-07-org-multi-org-director.md`: FR-7 を `[ ]` にし、項目の下に印(2026-10-26)を足す(gate を reviewer に移した。外の道具で動かした記録は leader が打った gate のものなので、次の実行で付け直す)。FR-7 の注に「2026-10-26 に決めたこと」を足す(gate を打つ座席、codex の reviewer の `--add-dir`、見張りは gate の記録を今までどおり数えること、テストの回数は 2 回のまま)。7b の注は当時の決定として残す。Research findings の「1 段目で、leader の座席にだけ `--add-dir` を足す」の行に、今回の変更を足す。Status に今回の変更と `[ ]` の数(6 項目)を足す
  - 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md`: FR-7 の行の判定を `[ ]` にして理由を書き、判定の数の文(`[x]` 25、`[ ]` 6)を直す
  - `docs/tech-debt/README.md`: codex の leader が gate のスクリプトと `git fetch` を sandbox の中で打てるかの行(「Whether a codex leader can run the gate's scripts」)を、gate のスクリプトは codex の reviewer が走らせる、に直す

## Non-goals

- `ralph org gate` の検査と記録の形を変えること(打つ人の制約も足さない。人が打つ道は残す)
- レビューの TASK での素のスクリプトの実行をやめること(D2)。テストは今までどおり review と gate の 2 回走り、2 回とも reviewer が走らせる
- `ralph org regate`、`merge-order`、`merge` の動き(どれも誰が打った gate の記録かを見ないので変わらない)
- 機能の org でない org(`ralph org start <task>`)の流れ。gate は機能の org だけを受け付ける
- 座席の停滞の判定(`internal/org/watch.go` の stall)を gate の間だけ止める仕掛け
- 実機での確かめ(分離した herdr で機能の org を動かすこと)。FR-7 の `[x]` は次の実行で付ける
- release(ユーザーが切る)

## Assumptions

- `ralph org gate`(`internal/org/gate.go` の `Gate`)は打つ人を確かめない。確かめるのは org の結びつき、cwd が結びつきの worktree とブランチにあること、作業ツリーがきれいなこと、HEAD が既定のブランチの base ref の先頭を含むこと、スクリプトがあること。記録の `by=` は `USER`、`herdr_pane=` は `HERDR_PANE_ID` の有無で、reviewer の pane から打っても leader の pane と同じ形になる
- gate は台帳の下の `gates/<org>-<時刻>.log` と manifest(ロックの下)に書く。claude の座席は sandbox がないので、autonomous(bypassPermissions)なら書ける。codex の座席は workspace-write の sandbox で、`--add-dir` がないと cwd の外の台帳に書けない(`docs/evidence/codex-add-dir-2026-10-07.md`)
- reviewer は leader と同じ機能の worktree で動く(leader の雛形の手順 5 の `--cwd .`)。`.harness/state/` は ignore されるので、`.harness/state/gate-<org>.out` は gate の作業ツリーの検査に数えられない
- merge-order(`internal/org/mergeorder.go`)は、ブランチごとに最新の gate の記録を読み、head を origin のブランチの先頭と照らす。誰が打ったかは読まない。座席の RESULT の `HEAD:` と `BASE:` は ralph が読まない
- 座席の停滞は、herdr の `agent get` の出力が前の pulse と同じで、座席の最後の記録が `[org.watchdog].stall_minutes`(既定 15 分)より古いときに出る(`watch.go` の `evaluateSeat`)。leader が `[org].deadman_minutes`(既定 10 分)のうちに動かなければ、受信箱の ALERT の件になる
- Spawn は manifest のロックの下で events を読み(`spawn.go` の `spawnFeatureSeatErr` が `ActiveFeature(events, p.OrgID)` を使う)、そのあとで agent の引数を組む
- 雛形は go:embed で binary に入る。新しい binary で spawn した座席から効く。古い binary で立てた leader は自分で gate を打ち続ける(新しい reviewer は `gate-` の TASK が来たときだけ gate を打つので食い違わない)
- `/org` と `/director` skill の 4 写しは `scripts/check-skill-sync.sh`、recipe と quality-gates の 2 写しは `scripts/check-sync.sh` が突き合わせる

## Affected areas

- 変更(コード): `internal/org/permissions.go`、`internal/org/spawn.go`、`internal/org/watch.go`、`internal/org/regate.go`、`internal/org/gate.go`(コメント)、`internal/cli/org_gate.go`(help)
- 変更(雛形): `internal/org/prompts/leader.md`、`internal/org/prompts/reviewer.md`
- 変更(テスト): `internal/org/permissions_test.go`、`internal/org/spawn_test.go`(:3758 の、linked worktree の codex の leader に `--add-dir` が付くテストのそば)、`internal/org/prompts_test.go`、`internal/org/prompts_leader_inbox_test.go`、regate の断りの文を固定している `internal/org/regate_test.go`(:594)
- 変更(skill と文書): `/org` と `/director` skill の各 4 写し、`docs/recipes/codex-seat-permissions.md` の 2 写し、`README.md`、`docs/quality/quality-gates.md` の 2 写し
- 変更(記録): `docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、`docs/tech-debt/README.md`
- 後工程の報告と insight event

## Visual review

- ページ: `.harness/state/plan-visual/reviewer-runs-gate.html`(やさしい図 2 枚 `overview` と `easy-ledger`、折りたたみの中に `detail-overview`、`detail-sequence`、`detail-adddir`)
- lint: `./scripts/plan-visual.sh lint` が終了コード 0
- 自分での確かめ: 全体、overview、easy-ledger、detail-overview、detail-sequence、detail-adddir の 6 枚を `shot` で撮って読んだ。重なりと切れはなく、「承認の前に」の 1 文の言い方だけを直した
- Codex の所見を入れたあと(D6 を見張りの数え方を変えない形にし、手順 12 の止める前の確かめと手順 14 の spawn し直しを足した)、変わった全体、overview、detail-overview、detail-sequence を撮り直して読んだ。detail-sequence の手順 13〜16 の箱の文がはみ出していたので、箱を広げて文を縮めた。lint は終了コード 0

## Design decisions

Critical forks: なし(ユーザーの指示で、計画の分かれ目は自分で選ぶ。下のとおり選んだ)。

- D1(gate を打つのは reviewer、頼むのは leader): gate は push する head で取る必要があり、その head は reviewer のレポート、org の report、計画の archive のコミットのあとにしか決まらない。そこで leader が手順 9〜11 を今のとおり済ませてから、reviewer に gate だけの TASK を送る。reviewer にレビューと gate を 1 つの TASK でさせる案は、gate のあとにレビューのレポートのコミットが入って head がずれるので採らない
- D2(レビューの TASK の素の実行は残す): 壊れたコードをレビューしないために、レビューの初めのスクリプトの実行は残す。テストは今と同じく 2 回走る(review と gate)。2 回とも reviewer が走らせるので、依頼の「静的検査とテストを reviewer が担当する」を満たす。1 回にする案は、gate が落ちるまで壊れたコードのレビューが続くので採らない
- D3(gate の TASK は TASK_ID で見分ける): reviewer のレビューの RESULT と gate の RESULT は、どちらも `GATE:`・`HEAD:`・`BASE:` のヘッダを持つ。leader が gate の RESULT を手順 9 の合図と読むと、コミットを足して HEAD がずれる。gate の TASK の TASK_ID を `gate-<n>` に決め、leader の手順 9 と 12、reviewer の雛形の両方がこの形で見分ける(consult の指摘)
- D4(leader は `.out` と HEAD を照らす): reviewer の RESULT は座席の申告なので、leader は reviewer の gate が共有の worktree に書いた `.harness/state/gate-{{ORG_ID}}.out`(中身は ralph の出力)を自分で読み、`GATE: pass` と `HEAD:` を `git rev-parse HEAD` と照らす。merge の根拠は今までどおり台帳の `gate_passed` で、merge-order と merge が PR の head と照らすので、座席が申告を違えても merge には進まない
- D5(codex の reviewer にも `--add-dir`): 2026-10-07 にユーザーは「codex の `--add-dir` は leader だけ」を選んだ(`docs/plans/archive/2026-10-07-org-state-dir-common.md` の Progress checklist)。当時の理由は「座席のうち台帳を書くのは leader だけ」だった。今回の依頼で reviewer が gate を打つと台帳に書く必要ができるので、その決定を変える。範囲は機能の org の reviewer だけにし、結びつきのない org の reviewer と、どの org の implementer にも足さない。claude の reviewer はもともと書ける(sandbox がない)ので、codex の reviewer をそれにそろえる形になる。変えたことは報告の初めに書く
- D6(見張りは gate の記録を今までどおり leader の動きと数える): 最初の案は、gate の記録を数から外すことだった。Codex の計画助言の指摘で取りやめた。保留中の ALERT は、そのときの数を基準値(`watch.go` の `watchPendingAlert.ManifestLen`)として状態ファイルに残し、見張りの再起動をまたいで `checkDeadman` が今の数と比べる。数え方を変えると、更新をまたいだ ALERT では基準値が gate の記録の数だけ高く残り、動いている leader を escalate しうる。戻したときは逆に、本物の ALERT を消しうる。数え方を変えなければこの問題は起きない。gate は leader の TASK(`sent`)を起点に reviewer が打つので、記録は leader が頼んだ仕事が進んだ印として数える。副作用として、reviewer の gate の記録が、leader 以外の座席についての保留中の ALERT を消しうる。leader 自身が止まったときは subject が leader の ALERT になり、動きの数によらずすぐ件になるので(`checkDeadman` は `activity && !subjectIsLeader` のときだけ消す)、止まった leader を見逃す経路にはならない。基準値を新旧 2 つ持つ案は、見張りの仕組みの変更が依頼の範囲より大きいので採らない(consult の判断)
- D7(座席の停滞の ALERT は雛形で扱う): Codex の reviewer が前面で gate を走らせる数十分、pane の出力が変わらず、見張りが停滞の ALERT を leader に送りうる。今は Codex の leader 自身が同じ形で止まって見えた。停滞の判定を変えず、leader の雛形に「gate の TASK を出している間の reviewer の停滞の ALERT では stop も spawn もしない」を書き、`/director` skill にその件の読み方を 1 文足す
- D8(FR-7 は `[ ]`): 印の規則(`docs/evidence/org-spec-evidence-2026-10-21.md` の冒頭)は、外の道具に頼る項目の `[x]` に実機の記録を 1 つ求める。FR-7 の実機の記録は leader が打った gate のもので、reviewer が打つ gate の記録はない。2026-10-26 の RC2 の直しで FR-4 と受け入れ条件 5 を `[ ]` のまま残した前例に合わせ、FR-7 を `[ ]` にして、次の実行で付け直すと印に書く(consult の指摘)

## Acceptance criteria

- [x] AC1(S1): codex で permission mode が autonomous か edits の座席のうち、leader と、機能の org の reviewer の argv に、台帳が cwd の下にないとき `--add-dir <state dir>` が permission の flag のすぐあと、`--model` の前に入る。機能の org の implementer、結びつきのない org の reviewer、未知の役割、director の argv には入らない(表のテストと spawn の argv のテストで確かめる)
- [x] AC2(S2): `leaderActivityEventCount` の数え方は変わらず(`gate_passed` と `gate_failed` を 1 つずつ数える)、(c) のコメントが D6 の理由を言う。regate の断りの文、`ralph org gate` の help、`gate.go` の冒頭のコメントが、機能の org の gate を reviewer が leader の TASK で打つと言う
- [x] AC3(S3、leader): leader の雛形の手順 12 が、reviewer に TASK_ID `gate-<n>` の TASK を送ってコマンドを本文に書くこと、結果が来るまで worktree を変えず implementer に TASK を送らないこと、reviewer の停滞の ALERT で stop も spawn もしないこと、3 つの結果(`GATE: pass` の RESULT、`GATE: unrunnable` と `GATE: fail` の BLOCKED)の扱いを書く。`GATE: pass` では `.harness/state/gate-{{ORG_ID}}.out` の `GATE: pass` と `HEAD:` を RESULT と `git rev-parse HEAD` に照らし、座席を止める前に `wip:` の確かめを済ませる。手順 9 はレビューの TASK(TASK_ID が `gate-` で始まらない)への RESULT で始まる。手順 14 は手順 12 で控えた `HEAD:` と照らし、違えば手順 5 と同じ flag で reviewer を spawn し直して手順 12 からやり直す。leader 自身が `ralph org gate` を打つ文が雛形にない
- [x] AC4(S3、reviewer): reviewer の雛形の「gate の TASK」の節が、TASK_ID の見分け、差分レビューと素の実行をしないこと、TASK のコマンドをそのまま打つこと、Claude Code と Codex の待ち方、3 つの返し方、打ち直しは 1 回、`ralph org gate` のほかの動詞を打たず台帳を直接書かないことを書く。前文の書いてよいものに台帳の gate の記録とログと `.out` が入っている。例の RESULT と BLOCKED が typed protocol の検査を通る
- [x] AC5(S4): AC1〜AC4 の文と動きがテストで固定され、既存の pin は意図を残して新しい文になっている。変えた文を 1 つずつ元に戻すと、対応するテストが落ちる(手で確かめる)
- [x] AC6(S5): `/org` と `/director` skill、recipe、README、quality-gates が AC1〜AC4 と食い違わない。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る
- [x] AC7(S6): 仕様の FR-7 が `[ ]` で、印(2026-10-26)と注「2026-10-26 に決めたこと」がある。Status と根拠の表の判定の数が `[x]` 25、`[ ]` 6 になっている。tech-debt の codex の gate の行が reviewer を指す
- [ ] AC8: `./scripts/run-verify.sh` が通り、Go のテストが手元の版と `GOTOOLCHAIN=go1.25.8` の両方で通る

## Implementation outline

1. S1 と S4 のうち S1 のテスト(permissions.go、spawn.go、表と argv のテスト。implementer、1 コミット)
2. S2 と S4 のうち S2 のテスト(watch.go のコメント、regate.go、org_gate.go、gate.go と regate_test.go。implementer、1 コミット)
3. S3 と S4 のうち雛形のテスト(leader.md、reviewer.md、prompts_test.go、prompts_leader_inbox_test.go。implementer、1 コミット)
4. S5(skill 2 つと sync-skills、recipe、README、quality-gates。implementer、1 コミット)
5. S6(仕様、根拠の表、tech-debt。implementer、1 コミット)
6. 後工程(self-review → verify → test → sync-docs → cross-review → pr)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(Go の vet と gofmt、shell)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC8。leader と reviewer の雛形が手順の番号と TASK_ID の形で食い違わないこと(leader の手順 12 の TASK と reviewer の「gate の TASK」の節)。`/org` skill の「org の終わり方」が leader の雛形の手順 9〜16 と同じ順になっていること
- Documentation drift to check: `ralph org gate` を打つ人を書いた文(雛形、skill、help、README、quality-gates、仕様、recipe、コメント)のうち、leader が打つと言う文が残っていないこと(`git grep` で `leader` と `org gate` が同じ文にある行を読む)。codex の `--add-dir` を「leader だけ」と言う文が残っていないこと
- Evidence to capture: 変えた文の行番号、`git grep` の結果、check-skill-sync と check-sync の出力、仕様の印の数(base と HEAD)

## Test plan

- Unit tests: `codexWritableRootArgs` の表(機能の org の reviewer の行と、付かない行)、`leaderActivityEventCount` の既存のテストが変わらず通ること、雛形の pin(leader の手順 9・12・14、ミッション 5・7、受信箱の節、reviewer の「gate の TASK」の節と例のメッセージの Parse と Validate)
- Integration tests: spawn の argv(偽の herdr で、機能の org の codex の reviewer と implementer、結びつきのない org の codex の reviewer を spawn し、argv の `--add-dir` の有無と位置を見る。外の道具は打たない)。regate の断りの文(偽の git と台帳で)
- Regression tests: `go test -count=1 -timeout 30m ./...` を手元の版と `GOTOOLCHAIN=go1.25.8` で。`./scripts/run-test.sh`
- Edge cases: 台帳が cwd の下にある機能の org の reviewer(`--add-dir` なし)、guarded の codex の reviewer(なし)、claude の reviewer(なし)、disband 済みで結びつきのない org の reviewer(なし)。leader の雛形に `gate-` の TASK_ID の決まり、pass の枝の止める前の確かめ、手順 14 の spawn し直しのどれかがなくなるとテストが落ちること
- Evidence to capture: 文を戻したときに落ちるテストの名前(AC5)、Go のテストの 2 回の結果

## Risks and mitigations

- R1: codex の reviewer が台帳に書けるようになり、manifest や座席の prompt を書き換えうる(2026-10-07 に避けた形)。範囲を機能の org の reviewer に限り、雛形で「`ralph org gate` のほかの動詞を打たず、台帳のファイルを直接書かない」とする。claude の reviewer は今も書けるので、程度はそろう。merge の根拠は台帳の `gate_passed` だが、座席が台帳に直接書けることは仕様が承認の注ですでに認めている(Security considerations の読み替え)
- R2: leader が gate の RESULT をレビューの RESULT と取り違え、手順 9 に戻ってコミットを足す。D3 の TASK_ID で分け、雛形のテストで固定する。取り違えても、手順 14 の HEAD の照らし合わせで push の前に止まり、merge-order も `regate` と出す
- R3: reviewer の gate のあいだに leader か implementer が worktree を変え、gate が `head_moved` か `tree_changed` で落ちる。leader の手順 12 で worktree を変えず implementer に TASK を送らないと書く。落ちても記録は `gate_failed` で、leader は頼み直せる
- R4: Codex の reviewer の長い gate で停滞の ALERT が出て、leader が動かないと deadman で受信箱の件になる。D7 のとおり雛形と `/director` skill に扱いを書く。件が人に届くのは今の Codex の leader の gate と同じ程度
- R5: 既存のテストが leader の手順 12 の文を細かく固定している(`prompts_test.go` の手順 9〜14、`prompts_leader_inbox_test.go` の gate の待ち)。pin の意図(待ち方、ファイルに書く、打ち直しの前に待つ)を reviewer の節の pin に移し、意図を落とさない
- R6: 実機で reviewer が gate を打つのを確かめないので、雛形どおりに動くかは分からない。FR-7 を `[ ]` にし(D8)、Known gaps に書く
- R7: reviewer の gate の記録が、leader 以外の座席についての保留中の ALERT を消しうる(D6 の副作用)。止まった leader は subject が leader の ALERT としてすぐ件になるので見逃さない。数え方を変えないので、更新と戻しで保留中の ALERT の基準値がずれることはない
- R8: 座席を止めたあとで HEAD が動く(leader 自身がコミットを足したとき)と、gate を頼む reviewer がいない。手順 12 の pass の枝で止める前に HEAD と `wip:` を確かめ、手順 14 の不一致では reviewer を手順 5 の flag で spawn し直す(Codex の計画助言)。実機では確かめない(Known gaps)

## Rollout or rollback notes

- 雛形と spawn の引数は binary に入るので、新しい binary で spawn した座席から効く。release のノートに「機能の org の gate は reviewer が leader の TASK で打つ。codex の reviewer にも機能の org では `--add-dir` が付く」と書く(題が `feat:` なので changelog にも出る)
- 戻すときはこの PR を revert する

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult で TASK_ID の分け方、停滞の ALERT の扱い、FR-7 を `[ ]` に戻すことを確かめた。Codex の計画助言の MEDIUM 2 件を入れた: 座席を止めたあとの HEAD の不一致で reviewer を spawn し直す道、見張りの数え方を変える D6 の取り下げ)
- [x] Plan approved(ユーザーの常設の指示「作成したプランは自動承認します」による)
- [x] Branch created
- [x] Implementation started(S1 e67152da、S2 5e8f5b50、S3 ce85125a、S4 8902cbf0 と 480b8781、S5 097a8c66 と 12e4b9d5。S4 と S5 は計画の S5 と S6 にあたる)
  - 計画からの逸れ 1: reviewer の雛形の「gate の TASK」は、本文のコマンドを「そのまま打つ」だけでなく、決まった 2 つと違えば打たずに QUESTION で leader に聞く(8902cbf0、`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand` で固定)。gate の TASK で台帳に書いてよいのは `ralph org gate` が書く分だけなので、consult の勧めで足した。`/org` skill の reviewer の役割にも同じ文を書いた
  - 計画からの逸れ 2: 計画の影響範囲になかった `.claude/rules/ralph/git-commit-strategy.md` と template の写しの 1 文を直した(12e4b9d5)。leader が `wip:` を確かめるのは「gate を打ち直す前と push の前」ではなく、「reviewer の gate が通ったあとと push の前」(雛形の手順 12 と 14)になったため
  - sync-docs で判断すること: Codex の reviewer が gate を前面で打つあいだは `ralph org send` が reviewer に届かない(reviewer.md の「gate の TASK」)。tech-debt には Codex の leader の同じ仕組みの行しかない
  - self-review の直し: 1 回目の F1〜F8 と N1 を 286f3d28、64f36f3f、67f3609d で、2 回目の F9・F10・N3 を d3c8e337 で直した(機能の org の implementer と reviewer を手順 13 まで止めない、頼み直しは同じ理由で gate の TASK 2 回まで、reviewer の QUESTION の枝、作業ツリーの汚れの断りは手順 9 と同じく分ける、spawn に `--role`、`reviewerRole` の定数)。直しで、影響範囲の表にない `internal/org/approval.go`(`reviewerRole`)にも触れた。3 回目の LOW の F11(手順 14 で reviewer だけを立て直した gate が落ちたとき、implementer を立て直す手がない)は直さず、`/sync-docs` で tech-debt の行にする。INFO の N2(`watch.go` のコメント (c) の段落分け)は直さない
- [x] Review artifact created(`docs/reports/self-review-2026-10-26-reviewer-runs-gate.md`、3 回、Merge OK)
- [x] Verification artifact created(`docs/reports/verify-2026-10-26-reviewer-runs-gate.md`、pass。LOW の V1 は `/sync-docs` で tech-debt の持ち越しの注を足す)
- [ ] Test artifact created
- [ ] PR created
