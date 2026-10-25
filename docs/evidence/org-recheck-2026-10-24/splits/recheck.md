# release の前の通しの確かめ: tech-debt の 2 行の直し

- Status: Approved
- Approved: 2026-10-25 sha256:4f955a6f4269

## 座席の決まり

- 座席はどれも claude で動かす。implementer は `--model opus`、reviewer は `--model sonnet` で spawn する
- 予約(`- Reserve:`)の外のファイルは編集しない
- `docs/tech-debt/README.md` は編集しない。直した行の RESOLVED は、この確かめの記録の PR が書く
- テストは本物の herdr、agmsg、gh、osascript、network を叩かない

## Features

### backfill-na

- Type: fix
- Reserve: internal/insights/backfill.go, internal/insights/backfill_test.go
- Depends on: none

`docs/tech-debt/README.md` の行「`ralph insights`' report backfill (`internal/insights/backfill.go`'s `parseCrossReviewAllCycles` / `crossReviewVerdict`) derives a cross-review event's verdict purely from the `After triage: …` line」を直す。

今は、cross-review の triage report の `Reviewer status:` の見出しを backfill が読まない。reviewer が終わらなかった report(`Reviewer status: incomplete (<理由>)`)も、`After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0` の行があるので、`crossReviewVerdict` が指摘なしとして `pass` を返す。

- [ ] `Reviewer status:` が `incomplete` で始まる cycle の verdict は `pass` ではなく `n/a` になる(読めない report と同じ扱い)
- [ ] `Reviewer status: complete` で 0/0/0 の cycle は、今までどおり `pass`
- [ ] 2 つの cycle を持つ report で、後ろの cycle だけが incomplete なら、その cycle だけが `n/a` になる
- [ ] 上の 3 つをテストで確かめる(ファイルは `t.TempDir()` に書く)

### hooks-matcher

- Type: fix
- Reserve: internal/cli/doctor.go, internal/cli/doctor_hooks_test.go
- Depends on: none

`docs/tech-debt/README.md` の行「`checkCodexEffectiveConfig`(doctor)の warn 精度に既知の限界 2 件」の (1) と (2) を直す。

- [ ] (2) `validateCodexHooksJSON` が、matcher group の `matcher` が文字列でないとき(数、配列、object など)に finding を出す。`matcher` がないときと文字列のときは出さない
- [ ] (1) config.toml の併存の警告(`hooks` のキーがあると削除を案内する)を、`[hooks]` の子のキーが `state` だけのとき(Codex 自身の trust の記録 `[hooks.state...]`)は出さない。ほかの子のキーがあれば今までどおり出す
- [ ] どちらも、出すときと出さないときの両方をテストで確かめる
