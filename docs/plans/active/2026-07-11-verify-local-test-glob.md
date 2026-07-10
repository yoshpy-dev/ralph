# verify-local-test-glob

- Status: Approved
- Owner: Claude Code
- Date: 2026-07-11
- Related request: PR #113 の tester レポートが検出したテストゲート列挙漏れの修正
- Related issue: N/A
- Type: fix
- Branch: fix/verify-local-test-glob

## Objective

`scripts/verify.local.sh` の手動列挙方式により `tests/` の28本中5本
（test-ralph-config / test-ralph-signals / test-ralph-status /
test-xreview-gate-regression / test-xreview-prompt-render、計75+ケース）が
どのモードでも実行されない問題を、glob ループ化で構造的に解消する。

## Scope

1. `run_hook_tests()`: 25個の if 連鎖を `for f in tests/test-*.sh` の
   glob ループに置換（実行可能ファイルのみ、従来と同じ `run` ラッパー経由）
2. `run_static_checks()` の shellcheck 対象列挙: `tests/test-*.sh` 個別列挙を
   glob に置換（同じ列挙ドリフトの別インスタンス）
3. shellcheck 新規・既存対象の警告解消（glob 化で露出した7ファイル）:
   - `tests/test-ralph-config.sh`: file-level `# shellcheck disable=SC1090`
     （動的 `$CONFIG` sourcing がテストの本質）
   - `tests/test-ralph-status.sh`: 未使用の `RALPH=` 行を削除、
     `SC1090,SC2034` の file-level disable（helpers 動的 source + その関数が
     消費する変数への偽陽性）
   - `tests/test-xreview-gate-regression.sh` / `test-xreview-prompt-render.sh`:
     `cd "$REPO_ROOT" || exit 1`
   - （実装中の追加）`tests/test-ralph-cli-driver.sh`: 同じく `cd ... || exit 1`、
     `tests/test-secret-scan.sh`: `CDPATH= ` → `CDPATH='' `（SC1007）。
     この2本は run 列挙には載っていたが shellcheck 列挙からは漏れていた

## Non-goals

- テスト自体の内容変更（合格状態を維持したまま。5本とも main 上で単体合格を確認済み）
- Go テスト側の選択方式（`go test ./...` は既に全量実行）
- 将来の意図的除外リスト機構（必要になったときに追加）

## Design decisions

- 列挙5本の追加ではなく glob ループ化を採用: 今後テストを追加した際の
  列挙漏れを構造的に防ぐ（PR #113 レビューでユーザーと合意済みの方針）
- shellcheck 側も同時に glob 化: 同一ファイル内の同型ドリフトを残すと
  「テストは走るが構文検査されない」非対称が生まれるため

Critical forks: None

## Acceptance criteria

- [ ] `run_hook_tests` が glob ループで、`grep -c 'if \[ -x tests/' scripts/verify.local.sh` が 0
- [ ] `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` 相当の実行で
      tests/test-*.sh 全28本が実行される（出力に test-ralph-config.sh 等5本が現れる）
- [ ] `shellcheck --severity=warning` が tests/test-*.sh 全本で warning 0
- [ ] `./scripts/run-test.sh` exit 0（全テスト合格）
- [ ] `./scripts/run-verify.sh` exit 0

## Verify / Test plan

- `./scripts/run-verify.sh` / `./scripts/run-test.sh` のフル実行
- test モード出力に5本が新規に現れることの diff 確認
- shellcheck 単体実行で warning 0 の証跡

## Risks and mitigations

- glob は英数字順で実行されるため従来の列挙順と変わる → 各テストは独立
  （5本の単体合格を事前確認済み）。順序依存が発覚したらテスト側の欠陥として別対応
- 将来追加されるテストが自動的にゲート対象になる → それが目的（意図的除外が
  必要になったら除外リストを導入）

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
