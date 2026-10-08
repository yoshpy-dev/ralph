# org-watch-stop-failed

- Status: Approved
- Approved: 2026-10-08 sha256:53d656a5e19f
- Owner: Claude Code
- Date: 2026-10-08
- Related request: PR #208(org-stop-all)の cross-review cycle 2 で残った ACTION_REQUIRED #1(`docs/reports/cross-review-triage-org-stop-all.md`)。ユーザーが「PR を作り、直しは別 PR」を選んだ(2026-10-08)
- Related issue: N/A
- Type: fix
- Branch: fix/org-watch-stop-failed

## Objective

`ralph org watch` の deadman が、閉じられなかった stop(`stop_failed`)も leader の活動として数えるようにする。PR #208 で、閉じられなかった `stop` は `stopped` の代わりに `stop_failed` を書くようになった。ところが `leaderActivityEventCount`(`internal/org/watch.go`)は `stop_failed` を数えない。herdr が応答しない間に leader が座席を止めようとしても活動に数えられず、deadman が leader の無活動として人に上げる。#208 の前は C-c が失敗しても `stopped` を書いていたので、数えられていた(後退)。

## Scope

- `leaderActivityEventCount`: 数えるイベントの種類(spawned、spawn_started、stopped、disbanded、rejected)に `stop_failed` を足す。ほかの種類と同じく、詳細に `reason=watchdog_` を持つものは数えない
- 同じ関数の doc comment の (b) の一覧と説明に `stop_failed` を足す。あわせて、`stop_failed` に `reason=watchdog_` のような互換のための除外が要らない理由を書く(下の前提の 3 つ目)
- watch のテスト(`internal/org/watch_test.go`): 警告が出たあと、次の評価までに leader の `stop_failed` があれば警告が消えること。別の org の `stop_failed` では消えないこと
- tech-debt(`docs/tech-debt/README.md` の `leaderActivityEventCount` の行): 解決済みにする(既存の RESOLVED の書き方に合わせる)

## Non-goals

- `org_workspace_closed` など、ほかのイベントを数えること。disband の成功は `disbanded` で数えられ、workspace の close に失敗したときも座席の `stopped` か `stop_failed` が残る
- #208 の cross-review の WORTH_CONSIDERING 2 件(tech-debt の 169 行)
- deadman の判定の仕組み(基準値と数え直しの差で見る)の変更

## Assumptions

- `stop_failed` を書くのは `ralph org stop` と `ralph org disband`(と `--all`)だけで、どれも leader か人が打つ動詞から呼ばれる。watchdog のコードは `Stop` を呼ばない(#152 以降。doc comment の (b) に書いてある)。このため `stop_failed` は、`stopped` と同じく leader の活動の証拠になる
- 数えた値は基準値との差としてだけ使われる(`sendAlert` が基準を記録し、`checkDeadman` が数え直す)。数える種類を増やしても、警告を出す前からあるイベントは差し引きで消える
- 保存済みの警告の基準(`ManifestLen`)は、警告を出したバイナリが数えた値になる。`stop_failed` を書くのに数えないバイナリ(#208 を含み、この PR を含まないもの)が基準を保存すると、この PR のバイナリの数え直しが過去の `stop_failed` の分だけ大きくなる。そうなると、新しい活動がないのに警告が消える(Codex plan advisory の指摘)。ただ、#208 を含む release はまだない(2026-10-08 時点の最新は v5.1.0、`git tag --contains da4dccb0` は空)。v5.1.0 は `stop_failed` を書かないので、v5.1.0 が保存した基準と、新しいバイナリの数え直しはずれない。このため、この PR を #208 と同じ release に入れれば、このずれは起きない
- 人が herdr の外から `stop --all` を打った場合も leader の活動に数える。これは今の `stopped` と同じ扱い

## Affected areas

- `internal/org/watch.go`(`leaderActivityEventCount` と、その doc comment)
- `internal/org/watch_test.go`
- `docs/tech-debt/README.md`(`leaderActivityEventCount` の行)

## Visual review

- ページ: `.harness/state/plan-visual/org-watch-stop-failed.html`(図 1 herdr が応答しない間の deadman、この PR の前と後)
- セルフチェック: 全体と全体図(`--fragment overview`)を `plan-visual.sh shot` で撮って確かめた。文字のはみ出しと重なりはなかった。Codex plan advisory の指摘を受けて図 1 の注記に release の条件を足し、撮り直した

## Design decisions

- 数える種類に `stop_failed` を足すだけにする。org のすべてのイベントを数える案もあるが、関数の doc comment は数える種類を意図して絞っている(watchdog 自身の書き込みや seat 由来の記録で警告が消えないようにするため)。広げる理由がない
- 更新をまたぐずれ(Codex plan advisory の指摘)は、`ManifestLen` とは別に `stop_failed` の基準を保存する形では扱わない。#208 は未配布で、この PR を同じ release に入れればずれは起きない。そのため、前提と rollout の注意に書いて止める(ユーザーが「軽く計画を直す」を選んだ。2026-10-08)

Critical forks: None

## Acceptance criteria

- [x] AC1: `leaderActivityEventCount` は、その org の `stop_failed` を 1 件として数える。詳細に `reason=watchdog_` を持つ `stop_failed` は数えない(ほかの種類と同じ)
- [x] AC2: deadman の警告が出たあと、次の評価までにその org で `stop_failed` が書かれると、警告は消え、人に上げない(`evaluateCycle` を 2 回回すテストで確かめる)
- [x] AC3: 別の org の `stop_failed` では、その警告は消えない
- [x] AC4: doc comment の (b) の一覧が `stop_failed` を含み、互換のための除外が要らない理由(#208 と同じ release に入る)を書いている。tech-debt の該当行が解決済みになっている
- [x] AC5: 今ある watch のテストがすべて通る

## Implementation outline

1. S1: `leaderActivityEventCount` に `EventStopFailed` を足し、doc comment を直し、watch のテストを足す。tech-debt の行を解決済みにする(AC1〜AC5)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC5。`docs/reports/cross-review-triage-org-stop-all.md` の cycle 2 の ACTION_REQUIRED #1 が直っていること
- Documentation drift to check: `git grep -n 'spawned, spawn_started, stopped' -- internal docs .claude` で、数える種類の一覧が古いまま残っていないか
- Evidence to capture: verify のレポート

## Test plan

- Unit tests: `leaderActivityEventCount` が `stop_failed` を数えること、`reason=watchdog_` つきは数えないこと
- Integration tests: `evaluateCycle` を使う deadman のテスト。警告のあとに `stop_failed` を書くと警告が消え、escalation が書かれない。別の org の `stop_failed` では消えない
- Regression tests: `./scripts/run-test.sh`。今の watch のテスト(`TestWatch_Deadman_*`)が通ること
- Edge cases: 警告を出す前からある `stop_failed` は基準値に入り、差し引きで消えること(既存の差分の仕組みのまま)
- Evidence to capture: test のレポート

## Risks and mitigations

- leader 以外が打った stop でも警告が消える: 今の `stopped` と同じ扱いで、新しい穴ではない。doc comment に書く
- herdr が止まったまま leader が stop を打ち続けると、deadman は人に上げない: leader は動いているので正しい。herdr が応答しないことは、stop の終了コード 1 と stderr で leader に伝わる
- #208 だけを含む release を切ると、更新をまたいで保存済みの警告が黙って消えうる: この PR を #208 と同じ release に入れる。PR 本文と rollout の注意に書く。main から自分でビルドしたバイナリを #208 とこの PR の間に使い、警告を保存した場合だけが残る(この repo の手元の `ralph` は Homebrew の v5.1.0)

## Rollout or rollback notes

- バイナリの更新で効く。この PR は #208 と同じ release に入れる(#208 だけの release を切らない)
- 戻すときはこの PR を revert する。台帳の形は変えない。revert したバイナリが、この PR のバイナリの保存した基準で数え直すと、数えが小さくなる。その場合は警告が消えにくくなる側(人に上げる側)に倒れる

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
- 2026-10-08: S1(2765f002)。テストは、偽の herdr の `PaneGet` を失敗させた本物の `Stop` で `stop_failed` を作る(`fakeWatchHerdr.PaneGetErr` を足した)。mutation で、この直しを戻すと新しいテストが落ちることを確かめた。implementer が前提の 3 つ目の穴を 1 つ見つけた。更新の途中で v5.1.0 の `ralph org watch` が動き続けていると、新しい `ralph org stop` が書いた `stop_failed` を除いた基準で警告を保存しうる。その警告が残ったまま watch を新しいバイナリで立て直すと、数え直しが 1 大きくなって警告が消える。doc comment にこの窓を書き、PR 本文の既知の穴にも書く。更新の前に watch を止めておけば起きない
