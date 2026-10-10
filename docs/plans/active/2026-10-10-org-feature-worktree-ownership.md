# org-feature-worktree-ownership

- Status: Approved
- Approved: 2026-10-10 sha256:feadcc05c21f
- Owner: Claude Code
- Date: 2026-10-10
- Related request: 4 段目(PR #216、merge 済み)の cross-review cycle 2 で残った WORTH_CONSIDERING 2 件(`docs/reports/cross-review-triage-org-feature-worktree.md`)を、merge の直後の別 PR で直す。ユーザーの決定は「PR を作り、直しは別 PR」(2026-10-10)
- Related issue: N/A
- Type: fix
- Branch: fix/org-feature-worktree-ownership

## Objective

`ralph org start --plan` が、別の分割計画のために作られた worktree を自分のものとして使わないようにする。

今は、機能の worktree の記録(`scripts/ralph-worktree.sh` の state の `canonical_ref`)が `split:<分割計画の id>#<slug>` で、id はファイル名から `.md` を除いたものになっている。そのため次の 2 つのずれが起きうる。

1. 同じ repo で別の台帳(`--state-dir`)に同じ名前の分割計画と同じ slug があると、disband のあとに残った worktree を、もう一方の start が使い回しの検査を通って引き継ぐ
2. 別々の分割計画から同じ org_id の start を同時に打つと、どちらも `ensure` の前の照合を通る。後の方の `ensure` は、先の方が作った記録をそのまま返す(スクリプトは `canonical_ref` を見ない)。そのため、台帳の結びつきは自分の計画を指し、worktree の記録は相手の計画を指す

## Scope

- `internal/org/feature.go`
  - `featureCanonicalRef` を `split:<分割計画の絶対パス>#<slug>` にする。パスは `ResolveSplitPlanPath` が symlink を解決した値(`SplitPlan.Path`)なので、どこから打っても同じ文字列になる
  - `StartFeature` で、`ensure` が worktree のパスを返したあと、`Spawn` の前に、記録をもう一度読んで照合する。記録が今回の分割計画のものでなければ拒否する。leader は立てず、台帳にも何も書かない
  - この拒否のエラー文は、`cleanup` も別の `--org-id` も案内しない。先に作った方の org がその worktree とブランチを使っている最中のことがあり、ブランチ名は `<type>/<slug>` で決まるので、org_id を変えても同じブランチに当たるため(Codex plan advisory の指摘)。「別の start が同じ worktree を別の分割計画のために作った」と書き、この分割計画の機能の slug(か Type)を変えて承認し直すか、先の org が終わって worktree が消えるのを待つよう案内する
  - 同じ理由で、ほかの 2 つの案内も合わせる。使い回しの拒否(`canonical_ref` が違う記録)は、slug を変えて承認し直す道を先に書き、`cleanup` は「その worktree が要らないとき」に限る。`ensure` の「ブランチがすでにある」の案内は、ほかの org や worktree がそのブランチを使っているかもしれないので、まず slug を変える道を書き、名前の変更や削除は「使っていないと確かめてから」に限る
- テスト(`internal/org/feature_test.go`、必要なら `internal/cli/org_feature_test.go`)
  - `canonical_ref` の形が変わるのに合わせて、既存のテストの期待を直す
  - 本物のスクリプトで、2 つの台帳に同じ id と slug の分割計画を置き、1 つ目の start と disband のあとの 2 つ目の start が拒否されること
  - fake の `ensure` が別の分割計画の記録を書いたとき、`Spawn` の前に拒否されること
  - 本物のスクリプトで、拒否のあとに案内どおり slug を変えて承認し直すと start が通り、先の org のブランチ・worktree・記録が変わらないこと
- 文書
  - `/org` skill(4 面)の、`start` が拒否するものの一覧(`canonical_ref` の説明と、同時の start の拒否)
  - `docs/tech-debt/README.md` の同時の start の行。食い違いの場合は拒否するようになったことと、残る部分(先読みと `ensure` の直列化はしていない)

## Non-goals

- 同じ機能の start の直列化(`ralph-worktree.sh` のロックや org_id ごとのロック)。tech-debt の同時の start の行に残す
- `scripts/ralph-worktree.sh ensure` に `canonical_ref` の照合を足すこと。スクリプトはほかの流れ(`/plan`、`/spec`)でも使うので、ralph の側で照合する
- PR #216 のコードで作った worktree の記録(古い形の `canonical_ref`)の移行。PR #216 はまだ release していないので、そうした記録は開発中のものだけ

## Assumptions

- `SplitPlan.Path` は `ResolveSplitPlanPath` が symlink を解決した絶対パスで、`LoadSplitPlan` も同じ値を持つ(PR #216 の S1)
- `ralph-worktree.sh ensure` は、同じ id・path・branch・kind の記録があれば、記録を書き換えずに既存の worktree を返す(`update_last_seen` だけ)。そのため、`ensure` のあとの照合と `Spawn` の間に記録が別の計画のものに変わることはない
- `--canonical-ref` は argv で渡るので、パスに空白があってもそのまま記録される

## Affected areas

- `internal/org/feature.go`、`internal/org/feature_test.go`(必要なら `internal/cli/org_feature_test.go`)
- `.claude/skills/org/SKILL.md` と 3 つの写し、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-feature-worktree-ownership.html`
- セルフチェック: 全体と全体図(`--fragment overview`)を `plan-visual.sh shot` で撮って確かめた。図 1 の `checkFeatureWorktreeReuse` の文字が箱の端に触れていたので、箱を広げて撮り直した。Codex plan advisory の 1 件を入れて、図 1 の拒否の案内(slug を変えて承認し直す、待つ)を直して撮り直した

## Design decisions

- `canonical_ref` には分割計画の絶対パスを入れる。台帳の場所だけを足す案もあるが、分割計画の置き場所は台帳の下の `splits/` に限っているので、パスで台帳と id の両方が決まる
- `ensure` のあとの照合は、既存の `checkFeatureWorktreeReuse` の照合の部分を使い、エラー文だけを分ける。照合の規則を 2 か所に書かないため
- 照合はスクリプトではなく ralph の側に置く。スクリプトの `ensure` はほかの流れでも使われ、`canonical_ref` の意味は流れごとに違う
- 拒否の案内は、先の org のブランチに触れない道(slug を変えて承認し直す、待つ)を先に書く(Codex plan advisory の指摘、ユーザーが「計画を直す」を選んだ)。ブランチ名は `<type>/<slug>` で決まり、org_id を変えても同じブランチに当たるため

Critical forks: None(どれも 1 スライスの中で戻せる)

## Acceptance criteria

- [x] AC1: `start --plan` が作る worktree の記録の `canonical_ref` は `split:<分割計画の絶対パス>#<slug>` になる。同じ分割計画(同じパス)の同じ機能は、disband のあと承認し直しても同じ worktree を使う
- [x] AC2: 別の台帳にある同じ名前(同じ id)の分割計画の、同じ slug の機能で start すると、残った worktree を使い回さずに拒否される(本物のスクリプトのテスト)
- [x] AC3: `ensure` が返したあとに記録が別の分割計画のものになっていると、start は `Spawn` の前に拒否する。leader は立たず、台帳にも何も書かない。エラー文は `cleanup` も別の `--org-id` も案内せず、slug(か Type)を変えて承認し直すか、先の org が終わるのを待つよう案内する
- [x] AC3b: 使い回しの拒否と `ensure` の「ブランチがすでにある」の案内は、slug を変えて承認し直す道を先に書き、`cleanup` やブランチの名前の変更・削除は、使っていないと確かめたときに限ると書く。本物のスクリプトで、拒否のあとに slug を変えて承認し直すと start が通り、先の org のブランチ・worktree・記録が変わらない
- [x] AC4: 既存の `StartFeature` と CLI のテストは、`canonical_ref` の形の変更に合わせた期待で通る
- [x] AC5: `/org` skill(4 面)の拒否の一覧と、tech-debt の同時の start の行が新しい挙動と合っている。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [x] AC6: `./scripts/run-verify.sh` が通る

## Implementation outline

1. S1(コード): `featureCanonicalRef` の変更、`ensure` のあとの照合と専用のエラー文、ほかの 2 つの案内の直し、テスト(AC1〜AC4、AC3b)
2. S2(文書): `/org` skill の 4 面と tech-debt の行(AC5)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC6。triage の cycle 2 の WORTH_CONSIDERING #1・#2 がどちらも閉じていること
- Documentation drift to check: `/org` skill の拒否の一覧、`ralph org start --help`(`canonical_ref` に触れていれば)
- Evidence to capture: verify のレポート

## Test plan

- Unit tests: `featureCanonicalRef` の形。`ensure` のあとの照合(記録が同じ、別の分割計画、読めない)
- Integration tests: 本物のスクリプトで、2 つの台帳に同じ id の分割計画を置いた場合(AC2)。fake の `ensure` が別の計画の記録を書いた場合に、`Spawn` が呼ばれず台帳が空のまま(AC3)。同じ分割計画の打ち直しと、disband のあとの承認し直し(AC1)
- Regression tests: `./scripts/run-test.sh`。PR #216 の `StartFeature`・`OrgStartPlan`・使い回しの検査のテスト
- Edge cases: 分割計画のパスに空白がある、symlink を通した `--plan`(解決後のパスで記録される)
- Evidence to capture: test のレポート

## Risks and mitigations

- PR #216 のコードで作った worktree の記録は古い形の `canonical_ref` なので、この変更のあとの start で使い回せずに拒否される: PR #216 は未 release で、開発中の記録だけ。エラー文の `cleanup` の案内で消せる
- 分割計画のファイルを移すと(パスが変わると)、同じ計画でも別のものとして拒否される: 置き場所は台帳の下の `splits/` に限っているので、移すのは台帳ごと移すときだけ。エラー文が `cleanup` か別の `--org-id` を案内する
- `ensure` のあとの照合でも、先読みから `ensure` までの競合は残る(後の方のエラーの案内が合わないことがある): tech-debt の行に残す

## Rollout or rollback notes

- バイナリの更新で効く。戻すときはこの PR を revert する。新しい形の記録が残っていると、古いバイナリの start はそれを使い回さずに拒否する。`cleanup` で消せる

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- 2026-10-10: S1(33271a68)。照合の規則は `featureWorktreeMismatch` にまとめ、使い回しの検査と、新しい `checkEnsuredFeatureWorktree`(`ensure` のあとの照合)の両方がこれを使う。計画からのずれ: 記録がぶつかる 2 つの案内(使い回しの拒否のうち `canonical_ref` が違う場合と、`ensure` のあとの拒否)は、AC3 の「slug(か Type)」ではなく slug の変更だけを案内する。Type だけ変えても org_id(既定は slug)は変わらず、記録の id `org-<org_id>` に同じように当たるため(`TestStartFeature_AfterDisband` のケースで確かめた)。「ブランチがすでにある」の案内は slug か Type。`ensure` のあとの照合は、記録が消えた・読めない・ほかの点が合わない場合も拒否し、打ち直すよう案内する。成功する start ごとに記録の読み出しと git の呼び出しが 1 回ずつ増える
- 2026-10-10: S2(c8863608)。`/org` skill の 4 面の `start --plan` の手順に 5(`ensure` のあとの照合)を足して番号を 1 つ繰り下げ、拒否の一覧を新しい案内に合わせた。tech-debt の同時の start の行に、別の分割計画からの同時の start は拒否するようになったことと、残る部分(直列化はしていない)を書いた
- 2026-10-10: self-review(cycle 1、7bd95ebd)の F-1(MEDIUM)・F-2・F-4・F-5 を 52800bae で直した。計画からのずれ: `--org-id` を明示した start(org_id が slug でない)では、記録の id `org-<org_id>` が slug で変わらないので、記録がぶつかる 2 つの案内は別の `--org-id` を勧め(`--org-id` を外して slug を org_id にする道も示す)、先の記録が同じブランチにあるときは slug も変えるよう添える。AC3 の「別の `--org-id` を案内しない」は、既定の org_id(slug)のときだけに当たる。F-2 で、待つ相手を「先の org の持ち主が、その PR の merge のあとに worktree を消す」と書いた。F-3(大文字小文字の違うパスは別の値になる。拒否の側に倒れる)は tech-debt に送る
- 2026-10-10: verify(9bd90993、pass)の V-3。上の Risks の 2 つ目(分割計画のファイルを移した場合)は「エラー文が `cleanup` か別の `--org-id` を案内する」と書くが、AC3b のあとの使い回しの拒否は、既定の org_id では slug の変更を先に案内し、`cleanup` は要らないときに限り、別の `--org-id` は出さない(`--org-id` を渡した start は別の `--org-id` を案内する)。Risks のこの文は AC3b より前の書き方で、Scope と AC3b が挙動を正しく書いているので、計画の本文は承認のとおりに残す
- 2026-10-10: sync-docs(cycle 1、この commit)。tech-debt の同時の start の行の注(self-review N1、verify V-1)を 52800bae のあとの案内に合わせ、F-3・N2〜N5・test report の残りの gap を 3 行にして足した。`/org` skill の 4 面は、org_id の条件の言い方を「slug と同じとき」にそろえ(V-4、N3)、「`--plan` をどう書いても同じ」に大文字小文字の例外を足した(V-2)。コードは触っていない
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
