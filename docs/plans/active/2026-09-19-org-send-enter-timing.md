# org-send-enter-timing

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-19
- Related request: #155 の実機検証(`docs/evidence/codex-seat-permissions-2026-09-18.md` P2)で、idle の codex 座席に `ralph org send` で複数行の TASK を送ると、本文は入力欄に貼り付けられるが送信されず、約 3 秒後に `herdr pane send-keys <pane> Enter` を手で追加すると submit された。busy の座席では queue に入り自動送信された。ralph は `pane send-text` の直後に `send-keys Enter` を送っている(`internal/org/verbs.go` の `Send`)ので、TUI の貼り付け処理と Enter の timing の問題とみられる。claude 座席では未確認(issue #163)
- Related issue: 163
- Type: fix
- Branch: fix/org-send-enter-timing

## Objective

`ralph org send` が idle の座席に複数行のメッセージを送ったとき、追加の手操作なしで submit されるようにする。テキスト送信と Enter の間に短い待ちを入れ、Enter の後に座席が working に移ったことを herdr の状態で確認し、確認できなければ Enter を 1 回だけ再送する。codex と claude の座席で実機確認し、evidence に残す。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | 待ち | `internal/org/verbs.go`(`Send`) | `PaneSendText` の後、`PaneSendKeys("Enter")` の前に待つ。既定 750ms(定数 `defaultSendEnterDelay`)。`SendParams.EnterDelayMS` が正ならその値。テスト用に `Org.SendEnterDelay`(ゼロなら既定。`AgentStartRetryInterval` と同じ形)で上書きできる。待ちは ctx の期限を尊重する |
| 2 | submit の確認と再送 | 同上 | Enter の後、`Herdr.AgentWait(ctx, name, ["working"], confirmMS)`(既定 3,000ms、定数 `defaultSendSubmitConfirmTimeout`、`Org.SendSubmitConfirmTimeout` で上書き可)で座席が working に移るのを待つ。確認できなければ(待ちがエラーを返したら)Enter を 1 回だけ再送し、もう一度同じ確認をする。再送は 1 回まで。herdr の状態は `idle / working / blocked / done / unknown`(herdr 0.7.5 の `agent wait --help` で確認)。pane の文字列は読まない |
| 3 | 記録 | 同上 | `sent` イベントの Details の先頭に、再送したら `enter_resent=true`、2 回目の確認でも working を観測できなければ `submit_unconfirmed=true` を付ける(`raw=true` と同じ形式)。`SendResult` に `EnterResent` / `SubmitConfirmed` を足す。確認できなくても `Send` はエラーにしない(ごく短いターンは待ちが始まる前に working → done になり得るため。空の入力欄への Enter は codex / claude とも何も起こさないので再送は無害) |
| 4 | CLI | `internal/cli/org.go`(`send`) | `--enter-delay-ms`(既定 0 = 組み込みの既定値 750ms。負の値は拒否)。submit を確認できなかった場合は stderr に 1 行の注意を出す(exit code は 0 のまま) |
| 5 | テスト | `internal/org/verbs_test.go`、`internal/org/spawn_test.go`(fake)、`internal/cli/org_test.go` | fake herdr の `AgentWait` を呼び出しごとに結果を指定できるよう拡張し、呼び出し順を固定する: idle/done の待ち → send-text → 待ち → Enter → working の待ち →(未確認なら)Enter の再送 → working の待ち → `sent` イベント。再送あり / なし、未確認、Details の接頭辞、待ち時間の指定、ctx の期限切れ、既存の失敗経路(send-text / Enter のエラー)が不変であること。CLI のフラグ検証 |
| 6 | 実機確認 | `docs/evidence/org-send-enter-timing-2026-09-19.md`(新規) | codex 座席(偽 HOME の herdr、#155 と同じ手順)と claude 座席(実 HOME の herdr)で、idle の座席に複数行の TASK を `ralph org send` し、追加操作なしで submit されることを確認する。修正前のバイナリ(main)での再現も同じ手順で記録する。`sent` イベントの Details(再送の有無)と herdr の状態遷移を残す |
| 7 | 文書 | `docs/recipes/codex-seat-permissions.md` + template、`.claude/skills/org/SKILL.md` + 3 ミラー | recipe の「手で Enter を 1 回送る」手順を削除し、`ralph org send` が待ちと再送を行う旨に置き換える。skill の send の説明に `--enter-delay-ms` と Details の接頭辞を追記 |

## Non-goals

- pane の出力を読んで「入力欄に本文が残っているか」を判定すること(TUI の描画に依存して壊れやすい。herdr の状態で判定する)
- `herdr pane wait-output` による貼り付け完了の検出(貼り付け完了を示す決まった出力がない)
- `ralph.toml` への設定キーの追加(CLI フラグと定数で足りる。テンプレートと既定値テストの同期が増えるため見送り)
- spawn 時の初期プロンプトの渡し方(`AgentStart` の引数で渡しており、本件の経路ではない)

## Assumptions

- 不具合は貼り付け直後の Enter が TUI に「貼り付けの一部」として扱われることによる(#155 では約 3 秒後の Enter で submit された)。750ms の待ちで足りるかは実機で確かめ、足りなければ既定値を実測に合わせる
- herdr は codex / claude の座席について working への遷移を報告する(watchdog が同じ状態を使っている)。報告が遅い、または来ない場合でも、再送は無害で `Send` は成功のまま、Details に `submit_unconfirmed=true` が残る
- 空の入力欄への Enter は codex / claude とも何も起こさない。busy の座席への Enter も何も起こさない(実機で確認する)

## Affected areas

- `internal/org/verbs.go`、`internal/org/spawn.go`(`Org` のフィールド)、`internal/cli/org.go`
- `internal/org/verbs_test.go`、`internal/org/spawn_test.go`(fake herdr)、`internal/cli/org_test.go`
- `docs/evidence/org-send-enter-timing-2026-09-19.md`(新規)、recipe + template、skill 4 面

## Design decisions

- **実機確認の範囲(ユーザー決定 2026-09-19)**: codex と claude の両方で実施する。codex 座席は偽 HOME の herdr(shell alias の衝突の回避、`~/.codex/auth.json` の一時複製)、claude 座席は実 HOME の herdr(claude は重複フラグを受け付けるので alias があっても起動する)。終了後は座席の stop / disband、複製した auth の削除、herdr server の停止まで行い、ユーザーの dotfile と実 config は変更しない

既定として採った選択:

- submit の確認は herdr の状態(working)で行う。pane の文字列を読む案(issue の 2)は採らない(Non-goals)
- 確認できなくても `Send` は失敗にしない。誤って失敗にすると、短いターンの座席への送信が成功しているのにエラーになる
- 待ちの既定は 750ms。issue の目安(500ms〜1s)の中間。実機で足りなければ evidence に基づいて変える

## Acceptance criteria

- [ ] AC-1: 単体テストで fake herdr に対する呼び出し順(idle/done の待ち → send-text → 待ち → Enter → working の待ち → 必要なら再送 → working の待ち)が固定される
- [ ] AC-2: 再送したら `sent` の Details に `enter_resent=true`、2 回とも working を観測できなければ `submit_unconfirmed=true` が付く。どちらの場合も `Send` は成功を返す。再送は 1 回まで
- [ ] AC-3: `ralph org send --enter-delay-ms` が使え、負の値は拒否される。未確認のとき stderr に注意が出て exit code は 0
- [ ] AC-4: 既存の経路が不変(プロトコル検証の拒否、dry-run、座席なし、send-text / Enter のエラー)
- [ ] AC-5: 実機 evidence: codex と claude の idle 座席への複数行の `ralph org send` が追加操作なしで submit される。修正前のバイナリでの再現、`sent` イベントの Details、後始末(座席の stop / disband、auth の削除、herdr server の停止、dotfile と実 config が不変)を記録
- [ ] AC-6: recipe(root / template 一致)と `/org` skill(4 面一致)が新しい挙動を述べ、手動の Enter の手順が残っていない。同期ゲート 3 本 pass
- [ ] AC-7: `gofmt` / `go vet` / golangci-lint clean、`go test ./internal/... -count=1` green、`TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1` ok、`./scripts/run-verify.sh` green、`./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/main)..HEAD"` が exit 0
- [ ] AC-8: PR 本文は `Closes #163`

## Implementation outline

1. Slice A(Go、implementer 委譲): Scope 1〜5 → 検証コマンド一式 → commit
2. Slice B(実機確認、inline): 修正前 / 修正後のバイナリで codex 座席 → claude 座席。結果を evidence に記録。既定の待ち時間が足りなければ Slice A に戻って調整
3. Slice C(docs、inline): Scope 7 → `sync-skills.sh` + cp → 同期ゲート 3 本
4. post-implementation pipeline → `/cross-review` → push 前の確認(run-verify、`TMPDIR=/tmp`、range secret scan)→ `/pr`(`Closes #163`)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、同期ゲート 3 本、range secret scan
- Spec compliance criteria to confirm: AC-1〜AC-8。Details の接頭辞と CLI の注意文が doc comment・skill・recipe と一致すること
- Documentation drift to check: recipe の手順、skill の send の説明、`docs/specs/2026-08-01-org-runtime.md` の send の記述、`docs/evidence/codex-seat-permissions-2026-09-18.md` P2(追記 1 行で新 evidence を指す)
- Evidence to capture: 単体テストの出力、実機 evidence ファイル

## Test plan

- Unit tests: Scope 5 の各ケース
- Integration tests: `ralph org send` の CLI テスト(フラグ、stderr の注意、exit code)
- Regression tests: `go test ./internal/... -count=1`、`TMPDIR=/tmp` での実行
- Edge cases: (1) 1 回目で working を確認 → 再送なし、(2) 1 回目は未確認で再送後に確認、(3) 2 回とも未確認、(4) 再送の Enter がエラー、(5) 待ちの途中で ctx の期限切れ、(6) `--enter-delay-ms` に負の値、(7) dry-run では待ちも確認も行わない
- Evidence to capture: `go test -count=1 -v` の出力、`docs/reports/test-*.md`、実機 evidence

## Risks and mitigations

- herdr が working を報告しない / 遅い → 再送は無害で `Send` は成功のまま。Details と stderr に未確認である旨が残る
- 再送の Enter が意図しない入力になる → 空の入力欄と busy の座席では何も起こさないことを実機で確認する。確認できなければ再送の条件を見直す
- 送信が最大で 750ms + 3s + 3s 遅くなる → 確認の待ちは working を観測した時点で終わるので、通常は 750ms 強の増加にとどまる。`--timeout-ms` の期限は全体に掛かったまま
- 実機確認が環境を汚す → #155 と同じ後始末を行い、evidence に記録する([[codex-seat-livefire-constraints]] の手順)

## Rollout or rollback notes

`ralph org send` の挙動変更(待ちと再送)と CLI フラグ 1 つ。下流へは次回 release でバイナリ経由、skill / recipe の文言は `ralph upgrade` / `ralph init` で配布。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
