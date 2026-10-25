# hooks-matcher: checkCodexEffectiveConfig の既知の限界 2 件を直す

- Status: Approved
- Approved: 2026-10-25 sha256:fc16043d3a2d
- Owner: Claude Code (org leader hooks-matcher)
- Date: 2026-10-25
- Related request: 分割計画 recheck(digest 4f955a6f4269)の機能 hooks-matcher
- Related issue: docs/tech-debt/README.md の行「`checkCodexEffectiveConfig`(doctor)の warn 精度に既知の限界 2 件」
- Type: fix
- Branch: fix/hooks-matcher

## 全機能の座席に適用する決まり

```
## 座席の決まり

- 座席はどれも claude で動かす。implementer は `--model opus`、reviewer は `--model sonnet` で spawn する
- 予約(`- Reserve:`)の外のファイルは編集しない
- `docs/tech-debt/README.md` は編集しない。直した行の RESOLVED は、この確かめの記録の PR が書く
- テストは本物の herdr、agmsg、gh、osascript、network を叩かない
```

## Objective

`ralph doctor` の Codex hooks 検査の誤案内と見逃しを 1 件ずつなくす。

## Scope

- (2) `validateCodexHooksJSON` が、matcher group の `matcher` が文字列でないとき(数、配列、object、bool、null など)に finding を出す
- (1) `checkCodexEffectiveConfig` の config.toml 併存警告を、`[hooks]` の子キーが `state` だけのときは出さない

## Non-goals

- 未知のトップレベルキーの拒否(公式スキーマの拡張に弱くなるため採らない)
- `docs/tech-debt/README.md` の更新(確かめの記録の PR が書く)
- 予約の外のファイルの編集

## Assumptions

- `matcher` は省略可能で、あるときは文字列(正規表現)である
- Codex の trust の記録は `[hooks.state...]` の子テーブルとして書かれる

## Affected areas

- `internal/cli/doctor.go`
- `internal/cli/doctor_hooks_test.go`

## Visual review

`.harness/state/plan-visual/hooks-matcher.html`(lint と shot の結果は承認の QUESTION に書く)

## Design decisions

Critical forks: None

## Acceptance criteria

- [ ] (2) `validateCodexHooksJSON` が、matcher group の `matcher` が文字列でないとき(数、配列、object など)に finding を出す。`matcher` がないときと文字列のときは出さない
- [ ] (1) config.toml の併存の警告(`hooks` のキーがあると削除を案内する)を、`[hooks]` の子のキーが `state` だけのとき(Codex 自身の trust の記録 `[hooks.state...]`)は出さない。ほかの子のキーがあれば今までどおり出す
- [ ] どちらも、出すときと出さないときの両方をテストで確かめる

## Implementation outline

1. matcher group のループで `group["matcher"]` があり `string` でなければ `hooks.json %s[%d].matcher must be a string` の finding を足す
2. `raw["hooks"]` の map から `state` を除いた子キーが 1 つ以上あるときだけ併存警告を出す
3. テーブル駆動テストを `doctor_hooks_test.go` に足す

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: 受け入れ条件 3 件
- Documentation drift to check: doctor の warn 文言を引く文書がないか grep
- Evidence to capture: reviewer のレポート(docs/reports/)

## Test plan

- Unit tests: `go test ./internal/cli/ -run 'Codex'`
- Integration tests: なし
- Regression tests: 既存の doctor の hooks テストが通る
- Edge cases: matcher が null・空文字列・配列。`[hooks]` が空の表、`state` と他キーの混在
- Evidence to capture: `./scripts/run-test.sh` の結果

## Risks and mitigations

- `matcher: null` を書く既存の設定が warn になる: 公式スキーマでは文字列なので warn が正しい。warn レベルで --strict ゲートは変わらない

## Rollout or rollback notes

doctor の warn の出方だけが変わる。戻すときは PR を revert する。

## Open questions

なし

## Progress checklist

- [ ] Plan reviewed
- [ ] Plan approved
- [ ] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
