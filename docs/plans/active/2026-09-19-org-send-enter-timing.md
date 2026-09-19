# org-send-enter-timing

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-19
- Related request: #155 の実機検証(`docs/evidence/codex-seat-permissions-2026-09-18.md` P2)で、idle の codex 座席に `ralph org send` で複数行の TASK を送ると、本文は入力欄に貼り付けられるが送信されず、約 3 秒後に `herdr pane send-keys <pane> Enter` を手で追加すると submit された。busy の座席では queue に入り自動送信された。ralph は `pane send-text` の直後に `send-keys Enter` を送っている(`internal/org/verbs.go` の `Send`)ので、TUI の貼り付け処理と Enter の timing の問題とみられる。claude 座席では未確認(issue #163)
- Related issue: 163
- Type: fix
- Branch: fix/org-send-enter-timing

## Objective

`ralph org send` が idle の座席に複数行のメッセージを送ったとき、追加の手操作なしで submit されるようにする。テキスト送信と Enter の間に短い待ちを入れ、Enter の後に座席が working(または blocked)に移ったことを herdr の状態で確認する。確認できなかった場合は記録と案内だけを行い、ralph からキー入力を追加で送ることはしない。codex と claude の座席で実機確認し、evidence に残す。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | 待ち | `internal/org/verbs.go`(`Send`) | `PaneSendText` の後、`PaneSendKeys("Enter")` の前に待つ。既定 750ms(定数 `defaultSendEnterDelay`)。`SendParams.EnterDelayMS` が正ならその値。テスト用に `Org.SendEnterDelay`(ゼロなら既定。`AgentStartRetryInterval` と同じ形)で上書きできる。待ちは ctx の期限を尊重する。**self-review による改訂**: 座席の idle / done の待ちが終わった時点で `--timeout-ms` の残りが待ち時間以下なら、本文を入力する前にエラーで終わる(`nothing was typed`)。入力した後、Enter を送る前に失敗した場合(待ちの途中の期限切れ、Enter の送信エラー)は `SendResult` に `TextTyped` と `PaneID` を載せ(`EnterPressed` は false)、CLI が「その pane を確認してから送り直す」よう stderr に案内する(送り直すと入力欄の残骸の後ろに 2 通目が入力されるため)。Enter を送った後の失敗(`sent` イベントの記録エラー)は `EnterPressed` が true で、CLI は「submit は済んでいるが記録できなかった。送り直さない」と別の案内を出す。submit 済みの座席に「未送信なので submit せよ」と案内すると、承認ダイアログに進んだ座席で人が盲目的な Enter を押すことになるため(再確認 NEW-1) |
| 2 | submit の確認(再送なし) | 同上 | Enter の後、`Herdr.AgentWait(ctx, name, ["working", "blocked"], confirmMS)`(既定 3,000ms、定数 `defaultSendSubmitConfirmTimeout`、`Org.SendSubmitConfirmTimeout` で上書き可)で座席が idle / done を離れるのを待つ。working も blocked も「メッセージは submit された」ことを意味する(blocked は承認ダイアログなどの入力待ち)。確認できなくても Enter は再送しない(Design decisions 参照)。herdr の状態は `idle / working / blocked / done / unknown`(herdr 0.7.5 の `agent wait --help` で確認)。pane の文字列は読まない |
| 3 | 記録 | 同上 | 確認できなければ `sent` イベントの Details の先頭に `submit_unconfirmed=true` を付ける(`raw=true` と同じ形式)。`SendResult` に `SubmitConfirmed bool` を足す。確認できなくても `Send` はエラーにしない(ごく短いターンは待ちが始まる前に working → done になり得るため、未確認は「送信されなかった」ことを意味しない) |
| 4 | CLI | `internal/cli/org.go`(`send`) | `--enter-delay-ms`(既定 0 = 組み込みの既定値 750ms。負の値は拒否)。submit を確認できなかった場合は stderr に注意を出す(exit code は 0 のまま): 座席の pane を `ralph org read` で確認し、本文が入力欄に残っている場合に限って `herdr pane send-keys <pane_id> Enter` を送る、という案内と pane_id |
| 5 | テスト | `internal/org/verbs_test.go`、`internal/org/spawn_test.go`(fake)、`internal/cli/org_test.go` | fake herdr の `AgentWait` を呼び出しごとに結果を指定できるよう拡張し、呼び出し順を固定する: idle/done の待ち → send-text → 待ち → Enter → working / blocked の待ち → `sent` イベント。確認あり / 未確認、**未確認でも `PaneSendKeys` が 1 回しか呼ばれないこと(Enter を再送しない)**、Details の接頭辞、待ち時間の指定、ctx の期限切れ、既存の失敗経路(send-text / Enter のエラー)が不変であること。CLI のフラグ検証と stderr の案内 |
| 6 | 実機確認 | `docs/evidence/org-send-enter-timing-2026-09-19.md`(新規) | codex 座席(偽 HOME の herdr、#155 と同じ手順)と claude 座席(実 HOME の herdr)で、idle の座席に複数行の TASK を `ralph org send` し、追加操作なしで submit されることを確認する。修正前のバイナリ(main)での再現も同じ手順で記録する。`sent` イベントの Details(`submit_unconfirmed` の有無)と herdr の状態遷移を残す。あわせて、承認ダイアログを表示中の codex 座席(guarded)について herdr が返す状態を記録する(blocked を submit の確認に含めた根拠) |
| 7 | 文書 | `docs/recipes/codex-seat-permissions.md` + template、`.claude/skills/org/SKILL.md` + 3 ミラー | recipe の「手で Enter を 1 回送る」手順を、「`ralph org send` が待ってから Enter を送る。未確認の注意が出たときだけ pane を確認して Enter を送る」に置き換える。skill の send の説明に `--enter-delay-ms`、Details の接頭辞、未確認時の対処(盲目的に Enter を送らない理由を含む)を追記 |

## Non-goals

- Enter の自動再送(Codex plan advisory HIGH、ユーザー決定で不採用。Design decisions 参照)
- pane の出力を読んで「入力欄に本文が残っているか」を判定すること(TUI の描画に依存して壊れやすい。herdr の状態で判定する)
- `herdr pane wait-output` による貼り付け完了の検出(貼り付け完了を示す決まった出力がない)
- `ralph.toml` への設定キーの追加(CLI フラグと定数で足りる。テンプレートと既定値テストの同期が増えるため見送り)
- spawn 時の初期プロンプトの渡し方(`AgentStart` の引数で渡しており、本件の経路ではない)

## Assumptions

- 不具合は貼り付け直後の Enter が TUI に「貼り付けの一部」として扱われることによる(#155 では約 3 秒後の Enter で submit された)。750ms の待ちで足りるかは実機で確かめ、足りなければ既定値を実測に合わせる
- herdr は codex / claude の座席について working への遷移を報告する(watchdog が同じ状態を使っている)。報告が遅い、または来ない場合でも `Send` は成功のまま、Details に `submit_unconfirmed=true` が残り、lead は案内に従って pane を確認する

## Affected areas

- `internal/org/verbs.go`、`internal/org/spawn.go`(`Org` のフィールド)、`internal/cli/org.go`
- `internal/org/verbs_test.go`、`internal/org/spawn_test.go`(fake herdr)、`internal/cli/org_test.go`
- `docs/evidence/org-send-enter-timing-2026-09-19.md`(新規)、recipe + template、skill 4 面

## Design decisions

- **実機確認の範囲(ユーザー決定 2026-09-19)**: codex と claude の両方で実施する。codex 座席は偽 HOME の herdr(shell alias の衝突の回避、`~/.codex/auth.json` の一時複製)、claude 座席は実 HOME の herdr(claude は重複フラグを受け付けるので alias があっても起動する)。終了後は座席の stop / disband、複製した auth の削除、herdr server の停止まで行い、ユーザーの dotfile と実 config は変更しない

既定として採った選択:

- **Enter の自動再送はしない(Codex plan advisory HIGH、ユーザー決定 2026-09-19)**: 当初の設計は「working を確認できなければ Enter を 1 回再送する」だったが、1 回目の Enter で submit に成功したのに working を観測し損ね、座席が承認ダイアログ(#155 evidence の「Yes, proceed」)に進んでいた場合、再送の Enter がそのダイアログを承認し得る。送信されないメッセージは取り返せるが、人の承認なしに権限の境界を越える操作は取り返せない。ralph は盲目的なキー入力を足さず、未確認を記録して lead に確認を促す
- submit の確認は herdr の状態(working / blocked)で行う。pane の文字列を読む案(issue の 2)は採らない(Non-goals)
- 確認できなくても `Send` は失敗にしない。誤って失敗にすると、短いターンの座席への送信が成功しているのにエラーになる
- 待ちの既定は 750ms。issue の目安(500ms〜1s)の中間。実機で足りなければ evidence に基づいて変える

## Acceptance criteria

- [ ] AC-1: 単体テストで fake herdr に対する呼び出し順(idle/done の待ち → send-text → 待ち → Enter → working / blocked の待ち)が固定される
- [ ] AC-2: submit を確認できなければ `sent` の Details に `submit_unconfirmed=true` が付き、`Send` は成功を返す。未確認の場合も Enter は 1 回しか送られない(テストで固定)
- [ ] AC-3: `ralph org send --enter-delay-ms` が使え、負の値は拒否される。未確認のとき stderr に pane の確認を促す案内(pane_id を含む)が出て exit code は 0
- [ ] AC-4: 既存の経路が不変(プロトコル検証の拒否、dry-run、座席なし、send-text / Enter のエラー)
- [ ] AC-5: 実機 evidence: codex と claude の idle 座席への複数行の `ralph org send` が追加操作なしで submit される。修正前のバイナリでの再現、`sent` イベントの Details、承認ダイアログ表示中に herdr が返す状態、後始末(座席の stop / disband、auth の削除、herdr server の停止、dotfile と実 config が不変)を記録
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
- Edge cases: (1) working を確認、(2) blocked を確認、(3) 未確認(Enter は 1 回だけ)、(4) 確認の待ちが ctx の残り時間より長い場合は残り時間に切り詰める、(5) 待ちの途中で ctx の期限切れ、(6) `--enter-delay-ms` に負の値、(7) dry-run では待ちも確認も行わない
- Evidence to capture: `go test -count=1 -v` の出力、`docs/reports/test-*.md`、実機 evidence

## Risks and mitigations

- herdr が working を報告しない / 遅い → `Send` は成功のまま。Details と stderr に未確認である旨が残り、lead が pane を確認する
- 待ちの既定値が環境によって足りない → 未確認の案内で気づける。実機で足りなければ既定値を上げ、`--enter-delay-ms` でも調整できる
- 送信が最大で 750ms + 3s 遅くなる → 確認の待ちは working を観測した時点で終わるので、通常は 750ms 強の増加にとどまる。`--timeout-ms` の期限は全体に掛かったまま
- 実機確認が環境を汚す → #155 と同じ後始末を行い、evidence に記録する([[codex-seat-livefire-constraints]] の手順)

## Rollout or rollback notes

`ralph org send` の挙動変更(待ちと submit の確認)と CLI フラグ 1 つ。下流へは次回 release でバイナリ経由、skill / recipe の文言は `ralph upgrade` / `ralph init` で配布。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

- 2026-09-19 plan: Codex plan advisory(codex-cli 0.154.0、`-m gpt-6-astra -c model_reasoning_effort=xhigh`、stdin を閉じて実行)が HIGH 1 件を報告: Enter の自動再送は、submit 済みで承認ダイアログに進んだ座席のダイアログを承認し得る。ユーザー判断(AskUserQuestion)で自動再送を不採用とし、Objective / Scope 2・3・4・5・6・7 / Non-goals / Assumptions / Design decisions / AC-1〜3・5 / Test plan / Risks を改訂した。issue #163 の「やること 2(Enter を 1 回だけ再送)」は採らない
- 2026-09-19 work: Slice A は implementer に委譲(c990363、6 ファイル)。逸脱なし。CLI テスト用の stub に、submit の確認の待ちだけを失敗させる環境変数を足した(既存の失敗注入は両方の待ちを失敗させるため)。orchestrator が HEAD 一致・porcelain 空・`Send` の差分を確認(Enter の送信は 1 箇所だけ)、ビルド・vet・対象テストを再実行して pass。implementer の `./scripts/run-verify.sh`、`TMPDIR=/tmp` でのテスト、race、履歴込みの range secret scan は green。implementer の指摘: `Send` の send-text / Enter のエラー経路に既存の単体テストがない(本件より前からの欠落)
- 2026-09-19 work: Slice B(実機確認、`docs/evidence/org-send-enter-timing-2026-09-19.md`、4c2d134)。codex 座席: 修正前は 5 回中 3 回で本文が入力欄に残り手動の Enter が必要、修正後(待ち 750ms)は 5 / 5 で submit、`sent` は 5 件とも submit 確認済み。既定値の変更は不要。claude 座席: 修正前でも 5 / 5 で submit(不具合は codex の TUI に固有)、修正後も 5 / 5 で退行なし。承認ダイアログ表示中の codex 座席について herdr は `blocked` を返し、ダイアログは「Yes, proceed」が選択済みで「Press enter to confirm」と表示される(Enter を再送しない根拠)。ダイアログは Escape で拒否し、対象ファイルは作られていない
- 2026-09-19 work: Slice B の環境面。ユーザー自身の herdr server が動いていたため触れず、専用 server を別 socket で起動した。herdr は `HOME` では設定・状態ディレクトリを切り替えないことが分かり、1 回目の専用 server はユーザーの `session.json` から workspace を復元し、ユーザーの `herdr-server.log` に 8 行追記した。SIGKILL で止め、`session.json` が不変(更新時刻・ハッシュ)であることを確認。以降は `XDG_CONFIG_HOME` / `XDG_STATE_HOME` / `HERDR_SOCKET_PATH` の 3 つで分離した。claude 座席は偽 HOME では keychain を参照できず「Not logged in」になるため、実 HOME + `env -i` の最小環境で起動した。残った変更は `~/.claude.json` のスクラッチディレクトリの信頼記録 1 件と上記のログ 8 行。後始末(座席の stop / disband、複製した auth の削除、専用 server の停止、`/tmp/r163` の削除)は evidence に記録
- 2026-09-19 work: Slice C(77371a8)。recipe(2 コピー)の手動 Enter の手順を新しい挙動の説明に置き換え、skill(4 面)の `send` の行に待ち・確認・未確認時の対処を追記、#155 の evidence P2 に追記 1 行。同期ゲート 3 本 pass
- 2026-09-19 self-review cycle 1(`docs/reports/self-review-2026-09-19-org-send-enter-timing.md`、8d7d1d2): MERGE(修正付き)、MEDIUM 3 / LOW 4。Enter が構造的に 1 回であること、再送を足すと落ちる回帰テスト、submit の確認の前提(herdr は timeout で非ゼロ終了)は確認済み。全件を同 cycle 内で修正: M1 待ち時間を最小 1ms に切り詰める理由のコメントが herdr の仕様と逆(0 は無期限待ち)、M2 `--timeout-ms` の残りが Enter 前の待ちより短いと本文を入力した後でエラーになり、送り直すと 2 通が連結され得る → 入力前に fail-closed、入力後の失敗は `TextTyped` / `PaneID` を返して CLI が案内、M3 send の `--timeout-ms` のヘルプが 3 つの段階を説明していない、L4 Enter の送信エラーにも「入力済み・未送信」を明記、L5 submit の確認は「座席が idle / done を離れた」ことしか観測していない旨を doc comment に明記、L6 余裕の小さいテストの調整、L7 既定値 750 を `DefaultSendEnterDelayMS` に一本化し、skill 4 面と recipe 2 コピーの記述が定数と一致することを確かめるテストを追加
- 2026-09-19 work: Slice D は implementer に委譲(7813c2a、5 ファイル)。逸脱 1 件: 「待ちの途中で ctx の期限が切れる」経路は、入力前の予算チェックを通過した後でしか到達せず、同期的な fake では再現できないため、多重防御として残しテストは付けていない(テストファイルに理由を記載)。orchestrator が HEAD 一致・porcelain 空・`Send` の流れを確認し、ビルド・vet・対象テスト・履歴込みの range secret scan(exit 0)を再実行。implementer の `-count=20`、`TMPDIR=/tmp`、race、`./scripts/run-verify.sh` は green。CLI のエラー接頭辞が `org: send: org: send:` と二重になるのは main に以前からある挙動で、本件の範囲外として残した
- 2026-09-19 self-review 再確認(同 report の Revalidation 節、693c1c6): 当初 7 件はすべて解消、verdict は MERGE。修正で生じた新所見 MEDIUM 1 / LOW 3 も同 cycle 内で直す。NEW-1(MEDIUM)`sent` イベントの記録エラーの経路は Enter を送った後なのに `TextTyped` だけで「未送信」の案内が出る(orchestrator の handoff の誤り。Scope 1 の記述も同じ誤りだったので本コミットで修正)→ `EnterPressed` を足して案内を分ける。NEW-2 `ctx.Deadline()` の ok を捨てている → ok を見て、残り時間の表示を 0 で下限に切る。NEW-3 「待ちの途中で期限が切れる経路はテストできない」は誤りで、fake の `PaneSendText` に遅延を注入すれば決定的に再現できる → テストを追加。NEW-4 既定値の同期テストが「`--enter-delay-ms` の隣」と言いながらファイル全体を見ている → その行の中で照合する
- 2026-09-19 work: Slice E は implementer に委譲(bcc1eb2、6 ファイル)。`SendResult.EnterPressed` を追加し、CLI の案内を「入力済み・未送信」(`TextTyped && !EnterPressed`)と「Enter 済み・`sent` の記録だけ失敗」(`EnterPressed`)に分けた。後者は submit・消去・Enter のいずれも指示しない。`ctx.Deadline()` の ok を確認し、残り時間の表示は 0 を下限にした。fake の `PaneSendText` に遅延を注入して「待ちの途中で期限が切れる」経路のテストを追加(`-count=50` で不安定さなし)。既定値の同期テストは `--enter-delay-ms` と同じ行で照合する。`sent` の記録失敗は manifest を読み取り専用にして再現(root では skip。既存の doctor テストと同じ手法)。orchestrator が HEAD 一致・porcelain 空・差分を確認し、記録失敗時のエラー文言が「message submitted」と観測以上のことを述べていたため「Enter was pressed」に直した(f7ea44d、inline の 1 行修正)。2 回目の再確認は行わない。implementer の指摘: `git commit -m "$(cat <<'EOF' …)"` の形が commit 前の guard に止められ、`git commit -F -` で回避した(`git-commit-strategy.md` が推奨する形そのものが通らない。本件の範囲外、PR に記載)

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
