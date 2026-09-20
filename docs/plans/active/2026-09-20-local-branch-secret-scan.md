# local-branch-secret-scan

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-20
- Related request: CI の `verify` ジョブは `./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/<base>)..HEAD"` で branch の履歴全体(`git log -p` の追加行)を scan するが、ローカルには同じ scan を走らせる手順がない。PR #168 では、テストの偽の値と、それを許可するために足した `.gitallowed` の行自体が CI で初めて掛かり、push 後に 2 回失敗した。履歴を読む scan なので、push 後に気づくと fixture を直しても過去の commit で掛かり続ける(issue #169)
- Related issue: 169
- Type: feat
- Branch: feat/local-branch-secret-scan

## Objective

CI と同じ「branch の履歴込みの secret scan」を、push の前にローカルで必ず通るようにする。`/pr` の事前チェックと `./scripts/run-verify.sh` の両方から、1 つのスクリプトで実行する。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | branch scan | `scripts/secret-scan-branch.sh`(新規)+ `templates/base/scripts/` のコピー | base branch を解決し(`GITHUB_BASE_REF` があればそれ、なければ `scripts/xreview-helpers.sh` の `detect_base_branch`)、`origin/<base>`(なければローカルの `<base>`)との merge-base から `HEAD` までを `./scripts/secret-scan.sh --range` で scan する。exit 0 = 問題なし、または理由を 1 行出して skip。exit 1 = 検出。exit 2 = 使い方の誤り。skip するのは: git の作業ツリーでない、HEAD が base branch そのもの、base の ref がどちらもない、merge-base が取れない、range に commit がない、scanner がない。skip は失敗にしない(CI が最終の関門) |
| 2 | run-verify | `scripts/run-verify.sh` + `templates/base/scripts/` のコピー | `HARNESS_VERIFY_MODE` が `static` か `all` のとき、branch scan を 1 つの step として実行する。検出したら全体を失敗にする(`status=1`)。「言語の verifier が 1 つも走らなかった」の判定(`ran_any`)には数えない。緊急時の回避として `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` で飛ばせる(飛ばしたことを 1 行出す) |
| 3 | /pr | `.claude/skills/pr/SKILL.md` + `.agents/skills/pr/SKILL.md` + `templates/base/` の 2 面 | Pre-checks に「`./scripts/secret-scan-branch.sh` が exit 0」を追加 |
| 4 | 案内 | `scripts/secret-scan.sh` + template のコピー、`docs/quality/quality-gates.md` | 検出時の案内文に 1 行: `.gitallowed` に足す行は、その行自体が scanner のパターンに一致しない形で書く(例: キー名を `api_ke[y]` のように括弧式にする)。range の scan は `.gitallowed` を足した commit の追加行も読むため。quality-gates にローカルの branch scan を追記 |
| 5 | 必須ファイル | `scripts/check-template.sh` + template のコピー | 必須ファイルの一覧に `scripts/secret-scan-branch.sh` を追加 |
| 6 | テスト | `tests/test-secret-scan.sh`(または新規 `tests/test-secret-scan-branch.sh`)、必要なら scaffold のテスト | 下の Test plan |
| 7 | 文書 | `README.md` / `AGENTS.md` の scripts の説明(該当箇所があれば)、`docs/recipes/`(該当があれば) | 新しいスクリプトの 1 行 |

## Non-goals

- scanner のパターンや `.gitallowed` の仕組みそのものの変更
- pre-push の git hook の追加(`/pr` と `run-verify.sh` で足りる。hook はユーザーの `.git/hooks` に入れる別の導入手順が要る)
- CI の workflow の変更(CI は今の step のまま)
- 既存の履歴にある検出済みの文字列の整理

## Assumptions

- `scripts/xreview-helpers.sh` は template にも入っており、`detect_base_branch` をそのまま使える
- `scripts/` と `templates/base/scripts/` は `check-sync.sh` が同一であることを確認する(新規ファイルも両方に置く)
- `git log -p` の range scan は branch の大きさに比例するが、通常の task branch では 1 秒未満

## Affected areas

- `scripts/`(新規 1、変更 3)と `templates/base/scripts/` のコピー
- `/pr` skill の 4 面
- `tests/`、`docs/quality/quality-gates.md`
- scaffold されたプロジェクト: `ralph upgrade` 後、`run-verify.sh` が branch scan を実行するようになる(CI と同じ基準。回避は環境変数)

## Design decisions

- Critical forks: None。
- 1 つのスクリプトに寄せる。`/pr` と `run-verify.sh` と人が、同じコマンドで同じ結果を得る。base の解決や skip の条件を 2 箇所に書かない。
- skip は失敗にしない。ローカルの scan は「push 前に気づく」ための早期検出で、最終の関門は CI。fetch していない、base の ref がない、といった環境の事情で verify 全体を落とさない。代わりに skip の理由を必ず 1 行出す。
- `run-verify.sh` では既定で有効にする。無効が既定だと #168 と同じ見落としが起きる。scaffold 先でも CI と同じ基準なので、新しい種類の失敗は増えない。

## Acceptance criteria

- [ ] AC-1: feature branch の途中の commit に secret 風の文字列があり、後の commit で消してあっても、`./scripts/secret-scan-branch.sh` は exit 1 になる(履歴を読むことの確認)
- [ ] AC-2: 問題のない feature branch では exit 0。base branch 上、base の ref がない、merge-base が取れない、range が空、git の外、では理由を 1 行出して exit 0
- [ ] AC-3: `origin/<base>` があればそれを、なければローカルの `<base>` を使う。`GITHUB_BASE_REF` があればそれを base にする
- [ ] AC-4: `./scripts/run-verify.sh` は、`static` / `all` のとき branch scan を実行し、検出したら非 0 で終わる。`test` のときは実行しない。`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` で飛ばせ、その旨を出力する。docs だけの変更の判定や既存の出力は変わらない
- [ ] AC-5: `.gitallowed` に、scanner のパターンに自分自身が一致する行を足した commit を含む branch は exit 1 になり、案内文が書き方の注意を示す。括弧式で書いた行なら exit 0
- [ ] AC-6: `/pr` の Pre-checks に branch scan が入り、skill の 4 面が一致する(`check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` が通る)
- [ ] AC-7: `scripts/` と `templates/base/scripts/` の該当ファイルが同一で、`check-template.sh` の必須一覧に新しいスクリプトが入っている。scaffold / upgrade の Go テストが通る
- [ ] AC-8: `./scripts/run-verify.sh` と `./scripts/run-test.sh` が green。PR 本文に `Closes #169`

## Implementation outline

1. Slice A: `secret-scan-branch.sh` とテスト(AC-1〜AC-3、AC-5)。template へのコピーと必須一覧(AC-7)
2. Slice B: `run-verify.sh` への組み込みとテスト(AC-4)、scanner の案内文(AC-5 の後半)
3. Slice C: `/pr` の Pre-checks(4 面)と `quality-gates.md` などの文書(AC-6)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(shellcheck を含む)、`sh -n` / `bash -n`、`check-skill-sync.sh`、`check-sync.sh`、`check-template-purity.sh`、`check-template.sh`
- Spec compliance criteria to confirm: AC-1〜AC-8。Non-goals(scanner のパターン、hook、CI は不変)
- Documentation drift to check: `quality-gates.md`、`/pr` skill、README / AGENTS.md の scripts の説明
- Evidence to capture: verify / test のログ、AC-1 の実演(一時 repo)

## Test plan

- Unit tests: 一時 repo を作る shell テスト。検出あり(途中の commit、後で削除)、問題なし、base branch 上、origin なし(ローカルの base に fallback)、base がどちらもない、range が空、`GITHUB_BASE_REF`、git の外、`.gitallowed` の自己一致と括弧式
- Integration tests: `run-verify.sh` の 3 つの mode と環境変数での skip。検出時の exit code
- Regression tests: 既存の `tests/test-secret-scan.sh`、`run-verify.sh` を使う既存のテスト、scaffold / upgrade の Go テスト
- Edge cases: detached HEAD、base と同じ commit にいる feature branch(range が空)、`RALPH_XREVIEW_BASE` の上書き、空白を含むパス
- Evidence to capture: `./scripts/run-test.sh` の出力

## Risks and mitigations

| リスク | 影響 | 対策 |
|---|---|---|
| テストの fixture 自体が、この repo の CI の secret scan に掛かる | #168 の再発 | fixture の文字列はテストの実行時に組み立てる(ソースに secret 風の代入を書かない)。既存の `tests/test-secret-scan.sh` の書き方に合わせる。push 前に range scan を実行する |
| scaffold 先で `run-verify.sh` が新しく失敗する | 利用者の verify が赤くなる | CI と同じ基準なので新しい種類の失敗ではない。回避の環境変数と、skip の理由の出力を用意する |
| base の解決を誤り、巨大な range を scan して遅くなる | verify が遅い | merge-base が取れなければ skip。range は merge-base から HEAD まで |
| `run-verify.sh` の `ran_any` / docs-only の判定を壊す | 既存の出力が変わる | branch scan は `ran_any` に数えない。既存のテストで確認 |

## Rollout or rollback notes

- 追加だけの変更。revert で元に戻る。緊急時は `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1`

## Open questions

- なし

## Deviation notes

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 現状の配線を確認した(CI の step、hook 群、`run-verify.sh`、template のコピー、既存の range のテスト)
- [x] critical fork なし
- [x] AC は一時 repo を使う shell テストで決定的に確認できる
- [ ] Codex plan advisory
