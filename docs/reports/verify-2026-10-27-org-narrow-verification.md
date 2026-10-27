# Verify report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(AC1〜AC6、D1〜D6、Verify plan、Progress checklist の S1〜S5 と直し F1〜F3)。issue #262 の受け入れ条件 4 つ
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1 / 2)
- Scope: `git diff 622cdcfb...HEAD`(HEAD 063f2ceb、18 ファイル、+1602/-100、15 コミット)。Go のコードの変更はテストのファイル 2 つだけで、ほかは役割雛形 3 本(go:embed)、`/org` skill の 4 つの写し、文書。self-review の 2 回目の LOW 2 件は、直し F3(8cda482e)に照らして HEAD で読んだ。テストは打っていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-27-org-narrow-verification.log`(gitignore。`run-static-verify.sh` の全出力、digest の計算、写しの `cmp`、gate の節の base と HEAD の `cmp`、drift の grep、`gofmt` と `go vet`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 leader の TASK の例に `VERIFY:` の行があり、機能ごとの org の手順 1 にスライスごとの検証コマンドと `RALPH_VERIFY_SCOPE=changed` の注意がある | Met | TASK の例は `internal/org/prompts/leader.md:463-474`(本文の `VERIFY: test -z "$(gofmt -l internal/foo)" && …`)。`VERIFY:` を本文の行にする決まりは `:449-451`(D4)。手順 1 は `:119-125`(「— verify: `<コマンド>`」、いちばん狭い検査、`RALPH_VERIFY_SCOPE=changed` は Go では `go test ./...` を全部走らせるので絞り込みに使わない)。ミッションの 3(`:28-43`)はどの org にも `VERIFY:` の行を求める。手順 5(`:199-202`)は計画の `— verify:` を書き換えずに写す |
| AC2 implementer の RESULT の契約に 5 項目の結果と検証コマンドの結果があり、既定が `run-verify.sh` ではなく、いちばん狭い検査 | Met | 既定は `internal/org/prompts/implementer.md:23-31`(`VERIFY:` の行がなければ触ったパッケージの `test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && go test ./<dir>/`、`run-verify.sh` は狭い検査で覆えないときだけ)。5 項目の確かめは `:48-57`(stage した差分、この順、範囲内で消せない秘密情報はコミットせず BLOCKED、値は書かない)。契約は `:133-145`、例は `:93-96`(`VERIFY: … → pass`、`SELF-CHECK:` の 5 項目)。stage → 確かめ → コミットの順は `:33-35` |
| AC3 機能ごとの org のレビューの TASK は `run-test.sh` を打たない。指示があればそれだけ、なければ両方。gate の TASK は今のまま | Met | leader の手順 8(`leader.md:226-245`)がレビューの TASK の本文に `CHECKS: ./scripts/run-static-verify.sh` の 1 行だけを書く。reviewer のミッションの 1(`internal/org/prompts/reviewer.md:17-29`)は、指示を本文の `CHECKS:` の行だけから取り、指示があれば足しも削りもせず、行がなければ両方を打つ(D6)。機能ごとでない org の leader はコマンドを書かない(`leader.md:40-43`)。gate の TASK: reviewer の「## gate の TASK」の節(67 行)と leader の手順 9〜12(120 行)は、base 622cdcfb と HEAD で `cmp` が一致した。`*.go` のうちテストでないファイルの変更は 0 件で、`ralph org gate` と `ralph org regate` のコードは変わっていない |
| AC4 `/org` skill の「役割」と「座席内 fan-out」が S1〜S3 と食い違わず、4 つの写しが一致する | Met | 「役割」の implementer は `.claude/skills/org/SKILL.md:1324-1340`、reviewer は `:1341-1363`。fan-out の例は `:1312-1316`。「org の終わり方」の冒頭の段落 `:951-965` と「Leader 運用 2 経路」の手順 3(`:1446-1452`)も雛形と同じ語。`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` は `cmp` で同一 |
| AC5 quality-gates・recipe・仕様の reviewer の記述が S2 と食い違わない | Met(S5 の範囲) | `docs/quality/quality-gates.md:82` と `templates/base/docs/quality/quality-gates.md:81` は、同じ文で「レビューの TASK が名指すコマンド、既定は両方、機能ごとの org は静的検査だけで、フルの test は gate で 1 回」(2 つの差は KNOWN_DIFF の「(see docs/tech-debt)」だけ)。recipe は `docs/recipes/codex-seat-permissions.md:220-223`、写しと `cmp` で同一。仕様は `docs/specs/2026-08-01-org-runtime.md:33-39` の「2026-10-27 改訂」(a)〜(d) と FR-7 の印。S5 が名指さない `docs/specs/2026-10-07-org-multi-org-director.md` に古い文が残る(下の Documentation drift の 1 行目) |
| AC6 雛形のテストと skill の同期の検査が通る | Partially(静的の半分) | `check-skill-sync.sh`(14 skill が一致)と `check-sync.sh`(DRIFTED 0)は通った。`go vet ./internal/org/` はテストのファイルを含めて rc 0 で、2 つの新しいテストのファイルはコンパイルできる。テストの実行(`TestRenderRolePrompt_*`、`TestOrgSkill_*`、`*_sync_test.go`)は `/test` に残す。足したテストは AC の文が名指す `prompts_test.go` ではなく `prompts_verification_test.go`(799 行、16 本)と `prompts_verification_multi_test.go`(90 行、2 本)にある(F2 と F3。Progress に記録済み)。`prompts_test.go` は base と同一 |

issue #262 の受け入れ条件:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 1. leader がスライスごとの検証コマンドを計画と TASK に書く | Met | AC1 と同じ |
| 2. implementer の RESULT に機械的な確かめと絞った検証の結果がある | Met | AC2 と同じ |
| 3. reviewer のレビューの TASK が `run-test.sh` を打たない | Met(機能ごとの org) | AC3 と同じ。D6 のとおり、指示のないレビュー(機能ごとでない org、古い leader)では両方を打つ |
| 4. 雛形のテストと同期の検査が通る | Partially | AC6 と同じ。テストは `/test` |

雛形 3 つの突き合わせ(Verify plan の spec compliance):

- leader が書く `VERIFY:` の行(`leader.md:28-39`、`:199-202`、例 `:473`)を、implementer は TASK の本文から読む(`implementer.md:19-23`)。複数のスライスの形 `VERIFY: S1: <コマンド>` は leader に 1 回、implementer に 2 回(受け取りと RESULT)、skill に 3 回、同じ文字列で出る。Go のいちばん狭い検査の文字列は、leader、implementer、skill に 1 回ずつ同じ形で出る
- 機能ごとの org のレビューの TASK の `CHECKS: ./scripts/run-static-verify.sh`(`leader.md:229`)を、reviewer は本文の行として読み(`reviewer.md:21-24`、`:148-149`)、同じ形で RESULT と BLOCKED に返す(`:53-59`、例 `:160`、`:173`、契約 `:220-222`)。leader は返った行に `./scripts/run-static-verify.sh` があるかを見て、なければ 2 回まで送り直し、3 回目は送らず escalate する(`leader.md:236-243`)。`./scripts/run-test.sh` も並んでいれば既定を打っただけとして扱う(`:244-245`)
- `GATE:` の意味: reviewer のミッションの 5(`reviewer.md:53-62`)と leader の手順 8(`leader.md:232-233`)が、機能ごとの org のレビューの `GATE: pass` は静的検査だけを指し、test の結果は gate の TASK の `GATE:` で見る、と同じことを言う

self-review の 2 回目の LOW 2 件(F3 8cda482e で直した分):

- LOW 1(複数のスライスの RESULT の形): 解消。`implementer.md:139-145` が `COMMIT:`・`VERIFY:`・`SELF-CHECK:` の行をスライスごとに名前付きで書くと決め、例(`:85-87`)がその節を指す。skill は `SKILL.md:1338-1340`。テスト `TestRenderRolePrompt_Implementer_ResultOfSeveralSlicesNamesEachSlice`(`prompts_verification_multi_test.go:20`)が足された
- LOW 2(`CHECKS:` の行のない TASK の散文を指示とみなす文): 解消。`reviewer.md:21-24` は「指示とみなすのはこの行だけ」で、行がなければ両方を打つ。消えた文(「を書かない leader もある」)は、雛形、skill の 4 つの写しのどこにも残っていない(残るのは self-review の report だけ)。skill は `SKILL.md:1344-1346`。テストは `prompts_verification_test.go:248` でこの文が戻らないことを確かめる

計画の digest: `./scripts/plan-visual.sh digest` が `97abe345a188` を返し、`- Approved: 2026-10-27 sha256:97abe345a188` と一致する。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | changed scope、言語は golang だけ。golang の pack は gofmt ok、`go vet` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし。local verifier の shellcheck、hook の `sh -n`、guard の awk の構文、settings の `jq`、Codex の hook の検査 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照はすべて OK。secret scan は 622cdcfb..063f2ceb で clean |
| `./scripts/check-skill-sync.sh` | rc 0 | 14 skill が一致 |
| `./scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `test -z "$(gofmt -l internal/org)"` と `go vet ./internal/org/` | rc 0 | テストのファイル 2 つを含む |
| `git diff --check 622cdcfb...HEAD` | rc 0 | 出力なし |
| 変更したファイルの U+FFFD | 0 件 | `grep -l` の rc 1 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/specs/2026-10-07-org-multi-org-director.md` FR-7 | No(LOW、`/sync-docs` へ) | `:124` の「2026-10-26 に決めたこと」が「テストは今までどおり review と gate の 2 回走り、2 回とも reviewer が走らせる」と言い、`:121` の 7b の注も「テストは review と gate の 2 回走る」と言う。このブランチのあと、機能ごとの org ではレビューが静的検査だけになり、test は gate の 1 回になる。この仕様は古い決定を書き換えず、日付の付いた「決めたこと」を足す(2026-10-26 の reviewer-runs-gate は FR-7 の項目と Status の段落 `:39` を足した)。計画の S5 はこの仕様を名指していないので AC5 の範囲の外だが、Verify plan の `docs/` の grep の範囲には入る。FR-7 はすでに `[ ]` なので、印の変更は要らない |
| `docs/tech-debt/README.md:157`(FR-7 の gate が雛形の指示による、の行) | No(既知、`/sync-docs` へ) | 「The reviewer seat re-runs `run-static-verify.sh` / `run-test.sh`」が残る。計画の Progress(`:182`)が、trigger に当たるが払わずに繰り越す注記を足す作業として sync-docs に渡している |
| `templates/base/ralph.toml:99` | Yes(情報) | 「the reviewer re-runs run-static-verify.sh / run-test.sh, so a guarded claude reviewer stops at the permission prompt」。機能ごとの org の reviewer もレビューで `run-static-verify.sh` を、gate の TASK で `ralph org gate`(中で `run-test.sh`)を打つので、guarded で許可待ちになるという注意は今も正しい |
| `docs/tech-debt/README.md:224-225`(tests twice の行) | Yes | RESOLVED の注記と打ち消し線。gate が静的検査をもう 1 回打つこと、指示のないレビューは両方を打つこと(D6)も書いてある。`:233` の reviewer の文は「(and, outside a feature org, its `run-test.sh`)」に直った。tech-debt の計画参照の検査は OK |
| `.claude/rules/ralph/git-commit-strategy.md:27`(Org runtime seats commit the same way as the standard flow above) | Yes | #261 の verify が「org の座席にはまだ当てはまらない(#262)」とした文。org の implementer がスライスのコマンド(TASK の `VERIFY:`、なければいちばん狭い検査)を打つようになったので、手順 2 と合う |
| README の `ralph org gate` の段落(`README.md:264`)と `/org` skill の gate の節(`SKILL.md:1113-1136` の「gate の記録」) | Yes | gate は今のまま 2 つのスクリプトを走らせる。「After the reviewer passes its review」はレビューの中身を言わない |
| implementer の既定を `run-verify.sh` とする文 | Yes | 雛形、skill、`docs/`、`.claude/rules/` の `run-verify.sh` で座席に触れるのは、狭い検査で覆えないときだけ使うと書いた 3 か所(`implementer.md:30`、`SKILL.md:1331`、仕様の改訂 (a))だけ |
| reviewer がレビューで必ず `run-test.sh` を打つと読める文 | Yes(上の 2 行を除く) | 記録(`docs/reports`、`docs/plans/archive`、`docs/evidence`、`docs/insights`)の外で `run-test.sh` を含むファイルを全部見た。reviewer の文は上の仕様と台帳の 2 行のほか、既定の両方を言うもの(D6)か gate のものだけ |
| 計画の記録 | 情報 | AC6 と S1 は `internal/org/prompts_test.go` を名指すが、テストは新しいファイル 2 つにある(F2・F3 で Progress に記録済み)。Progress の `:183` に、`:177` の `[x] Review artifact created` と重なる `[ ] Review artifact created` の行が残る。digest は一致しており、この 2 点は承認の検査に関係しない |

## Observational checks

- 版が混ざったときの道(雛形を読んだだけで、動かしていない): 新しい leader と古い reviewer の機能ごとの org では、古い reviewer も「指示があればそれ」なので静的検査だけを打つ(D6 の分析のとおり)。ただし古い reviewer は `CHECKS:` の行を返さないので、新しい leader は手順 8 のとおり 2 回送り直し、3 回目は送らずに escalate で BLOCKED を上げる。test を走らせずに先へ進むことはないので、安全側に止まる。D6 はこの組み合わせを F1 の送り直しの規則より前に書いた。leader は reviewer をその時点の PATH の binary で立てるので、reviewer が leader より古くなるのは binary を戻したときだけで、通常の手順ではこの道に入らない
- 座席の実機の起動は確かめていない(計画の Test plan の Integration tests と同じ)

## Coverage gaps

- テストの実行(雛形のテスト、`*_sync_test.go`、`go test ./internal/org/` の全体)は `/test` に残す。計画の Test plan の変異(reviewer のミッションに `run-test.sh` を戻すと落ちる、など)も `/test` の範囲
- 雛形の文に座席がそのとおり従うことは、文の確かめでしか見ていない。leader が `VERIFY:` と `CHECKS:` を書くこと、reviewer が `CHECKS:` を返すことの実機の記録はない

## Verdict

- Verified: AC1〜AC5(AC5 は S5 の範囲)、issue の受け入れ条件 1〜3、雛形 3 つの突き合わせ、gate の TASK の節と `ralph org gate`/`regate` が変わっていないこと、self-review の LOW 2 件の解消、計画の digest、`run-static-verify.sh`・`check-sync.sh`・`check-skill-sync.sh` の通過
- Partially verified: AC6 と issue の受け入れ条件 4(同期の検査と `go vet` は通った。テストは `/test`)
- Not verified: 座席の実機の振る舞い、版が混ざったときの道
- Result: pass。`/sync-docs` への渡し: `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 に「2026-10-27 に決めたこと」を足す(LOW)、`docs/tech-debt/README.md:157` の繰り越しの注記(計画の Progress の渡し)
