# org-human-decision

- Status: Approved
- Approved: 2026-11-21 sha256:fe3344aa0cac
- Owner: Claude Code
- Date: 2026-11-21
- Related request: issue #366(人の決定を、受信箱の件なしで走っている org に送る手順がない)
- Related issue: 366
- Type: feat
- Branch: feat/org-human-decision

## Objective

人が走っている org について決めたこと(座席の設定を変える、やり直す、など)を、受信箱の件がなくても leader に届ける手順を決める。director は `ORIGIN: human` のヘッダを付けた DECISION を `ralph org send` で leader に送ってよく、ralph はその送信を台帳に「人の決定」と分かる形で記録する。

## Scope

- `ralph org send`(`internal/org/verbs.go` の Send): 本文のヘッダに `ORIGIN:` があるメッセージを確かめる。`ORIGIN` は TYPE が DECISION のときだけ、値は `human` だけを受け付け、ほかは何も打たず何も記録せずに断る(`--raw` でも同じ形で読んだときは同じ規則)。宛先は leader だけ
- `sent` の記録: `ORIGIN: human` の DECISION は、`sent` の Details の先頭に `origin=human ` を付ける(`raw=true ` と同じ形、`raw=true ` のあと)。dry run の `sent` も同じ。本物の送信の `sent` には、本文の全文も新しい欄(ManifestEvent の `Text`、JSON は `text`、`ORIGIN: human` のときだけ)に残す。Details は本文の先頭の数百バイトしか持たないため
- leader の立て直し(`internal/org/reseat_progress.go`): 立て直した leader のタスクの進捗の節に、その org の今の命のうちに leader へ送られた `ORIGIN: human` の DECISION を、送った時刻と全文で載せる(今は leader 宛ての `sent` を読まない)。`ralph org reseat` と `start --plan` の立て直しのどちらも同じ
- leader の prompt(`internal/org/prompts/leader.md`): `ORIGIN: human` の DECISION は人の決定として扱う。振り分けの順を書く: `ORIGIN: human` の DECISION は、計画の承認(per-plan の手順 4)でも直しの依頼でも、cross-review の上限の答えでも、ほかの待っている件への答えでもない(`HOOK_SEAT:` の DECISION と同じ扱い)。leader は本文の決定を進め、待っていた答えを待ち続ける。決定が承認済みの計画の変更を求めるなら、計画を直して承認を取り直す(手順 2 と 3)。立て直したあとのタスクの進捗の節にある人の決定も、同じく守る
- director skill(4 写し): 「してはいけないこと」の DECISION の送り方の項と「答える順」に、件のない人の決定の送り方を書く。人が決めたときだけ送る、本文に人の言葉(またはその要点)と決めた日時を書く、送ったあと leader の pane で届いたことを確かめる(今の DECISION の送り方と同じ)
- `.claude/rules/ralph/agent-messaging.md`(と雛形の写し): `ORIGIN: human` の DECISION の段落
- org skill: leader の動きに 1 文(必要なら)

## Non-goals

- 座席(leader 以外)への `ORIGIN: human` の送信。人の決定は leader を通す(星形)
- `ralph org send` に新しい flag を足すこと(ヘッダで足りる)
- 受信箱の件を作ること(件なしの送信が目的)
- 人であることを ralph が確かめること(誰が打ったかは ralph には分からない。director の skill の決まりで守る)

## Assumptions

- director は今も「答える順」の DECISION を `ralph org send` で送っている。`ORIGIN: human` はその形に 1 行足すだけ
- `sent` の Details は本文の先頭の数百バイトを持つ(`truncateForDetails`)ので、ヘッダの行は今も Details に入るが、先頭の印のほうが機械で読みやすく、本文が長くても消えない

## Affected areas

- `internal/org/verbs.go`(Send の確かめと Details)、`internal/org/send_origin.go`(新しいファイルにするなら)、`internal/org/manifest.go`(`Text` の欄)、テスト
- `internal/org/reseat_progress.go` とテスト(人の決定の項)
- `internal/org/prompts/leader.md` と prompt のテスト
- `.claude/skills/director/SKILL.md` と写し、`.claude/skills/org/SKILL.md` と写し(必要なら)
- `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し
- README.md(`ralph org send` の段落)、AGENTS.md(必要なら)

## Visual review

None (送信の確かめに 1 つの規則と記録の印を足すだけで、部品の間のやり取りの順や状態の移り変わりは変えない。文書は手順の文)

## Design decisions

Critical forks: None

- 印は Details の先頭の `origin=human ` にする(`raw=true ` や `via=screen ` と同じ形)。全文は `ORIGIN: human` のときだけ新しい欄 `text` に残す。Details の先頭だけでは、leader を立て直したときに長い決定を引き継げない(codex の計画助言の 2)。ほかの送信の記録は増やさない
- leader の振り分けは `HOOK_SEAT:` の DECISION の扱いに倣う。人の決定を待っている件の答えと取り違えると、不要な計画の直しや承認のやり直しが起きる(codex の計画助言の 1)
- `ORIGIN` を DECISION 以外に付けたら断る。人の決定でない種類に付くと、後から読む人が取り違える
- 宛先は leader だけ。座席は leader 以外からの指示を受けない(agent-messaging.md の星形)

## Acceptance criteria

- [ ] AC1: `ORIGIN: human` のヘッダつき DECISION を leader へ `ralph org send` すると送れ、`sent` の Details が `origin=human ` で始まる(`raw=true ` があればそのあと)。dry run も同じ
- [ ] AC2: `ORIGIN` が DECISION 以外の TYPE に付いている、値が `human` 以外、宛先が leader でない、のどれかなら、何も打たず何も記録せずに断り、理由を言う
- [ ] AC3: `ORIGIN` のないメッセージの送信と記録は今のまま(既存の send のテストが通る)
- [ ] AC4: leader の prompt が、`ORIGIN: human` の DECISION を人の決定として扱い、計画の承認待ち(per-plan の手順 4)と cross-review の上限の答え待ちのどちらでも、それを待っている答えとみなさず待ち続けることを書く。prompt のテストが、承認待ちと上限の答え待ちの両方の文を固定する
- [ ] AC5: director skill の 4 写しが件のない人の決定の送り方を書き(「してはいけないこと」の DECISION の項と「答える順」)、skill のテストと `./scripts/check-skill-sync.sh`・`./scripts/check-sync.sh` が通る
- [ ] AC6: agent-messaging.md に `ORIGIN: human` の段落がある
- [ ] AC7: 200 バイトを超える `ORIGIN: human` の DECISION を送ったあと leader を立て直すと(`ralph org reseat` と `start --plan` の立て直し)、立て直した leader のタスクの進捗の節に、その決定の全文と時刻が載る。org が disband されたあとの命の決定は載らない。`ORIGIN` のない leader 宛ての送信は今のとおり載らない

## Implementation outline

1. S1 send の確かめと記録(AC1〜AC3、`text` の欄) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Send|Origin' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'OrgSend'`
2. S2 立て直しの進捗の節(AC7) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Reseat|Progress|StartFeature'`
3. S3 leader の prompt と文書(AC4〜AC6) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Prompt|Skill|Director|DocsMatch' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Agents|Readme|Docs|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC6 と issue #366 の受け入れ条件 2 つ
- Documentation drift to check: README の `ralph org send`、`ralph org send --help`、agent-messaging.md と雛形の写し、director と org の skill の 4 写し、leader の prompt
- Evidence to capture: verify の report

## Test plan

- Unit tests: ORIGIN の確かめ(DECISION と human なら通す、TYPE の違い・値の違い・宛先の違いで断る、大文字小文字の扱い)、Details の印(raw のあと、dry run)
- Integration tests: CLI の `ralph org send` で ORIGIN つきの DECISION を送る(fake の herdr)
- Regression tests: 既存の send のテストすべて
- Edge cases: `ORIGIN:` が 2 行ある、値の前後の空白、`--raw` で TYPE が DECISION の行を持つ本文
- Evidence to capture: `./scripts/run-test.sh` の log、確かめを外す変異でテストが落ちること

## Risks and mitigations

- 誰でも `ORIGIN: human` を書ける: ralph は人であることを確かめられない。director の skill の「人が決めたときだけ」で守り、台帳の印は「人の決定として送られた」ことの記録と書く(人であることの証明ではない)
- leader が `ORIGIN: human` の DECISION を件への答えと取り違える: leader の prompt に振り分けの順を書き、承認待ちと上限の答え待ちの両方の文をテストで固定する
- 台帳が大きくなる: 全文を残すのは `ORIGIN: human` の DECISION だけで、本文は 2,000 文字の上限がある
- 他のレーンが leader.md と director skill を同時に触る(系列の B・C・D): テストが固定する文を変える前に知らせ、merge の前に main を取り込む

## Rollout or rollback notes

この PR を revert すれば、`ORIGIN` のヘッダは何も確かめられずに本文として送られる(今の動き)。台帳の形は変えない。

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 2 件、どちらも計画に入れた。[MEDIUM] 1: 件のない人の決定を、承認待ち(per-plan の手順 4)や cross-review の上限の答え待ちの答えと取り違え、不要な計画の直しや承認のやり直しが起きる → leader の振り分けの順を Scope に書き、AC4 で承認待ちと上限の答え待ちの両方の文をテストで固定する。[MEDIUM] 2: leader を立て直すと人の決定が引き継がれない(記録は本文の先頭だけ、立て直しの進捗の節は leader 宛ての送信を読まない) → `ORIGIN: human` のときだけ全文を `text` の欄に残し、進捗の節に載せる(Scope、AC7、S2)。git の状態は助言の前後で同じ
- [x] Plan approved — 2026-11-21、系列の決まり(計画は推す案で自動承認)で承認した。人の目での承認はしていない
- [x] Branch created
- [x] Implementation started
  - S1(fd7e9d9e、implementer): 確認コマンドが exit 0。`internal/org/send_origin.go`(`checkSendOrigin`)を貼り付けマーカーの確認の直後、watch gate と dry run の分岐より前で呼ぶ。Details の先頭は `raw=true ` → `origin=human ` → `via=screen …` → `submit_unconfirmed=true ` の順。dry run は印だけで `Text` なし。ずれ: (1) `ORIGIN: human` の全文に 16 KiB の上限を足した(`--raw` には 2,000 文字の上限が効かず、台帳の読み手は 1 行 1 MiB まで)、(2) キーは protocol と同じく大文字小文字を区別するので `Origin:` は ORIGIN と扱わず印も付かない、値 `Human` は断る、2 行の ORIGIN は断る、(3) 宛先は seat id が `leader` かで見る(dry run でも同じ判定)、(4) 1 行目が TYPE のヘッダでない `--raw` は確かめず印も付けない。変異 2 種(宛先の確かめ、`Text` の書き込み)でテストが落ちる。範囲外の気づき: `ralph org send --help`(`internal/cli/org.go`)に ORIGIN の説明がない → S3 の範囲に入れる
  - S2(e1a9ad18、implementer。1 回目はアプリの再起動で途切れ、何も残さなかったので出し直した): 確認コマンドが exit 0。立て直した leader の進捗の節の最後に「人の決定」の項(定数 `reseatProgressHumanDecisionsLabel`)。1 件ごとに `- <時刻> に送った:` と全文を 2 字下げで載せ、制御文字はエスケープする。印(Details の `origin=human `)と空でない `Text` の両方で判定する(`send_origin.go` の `sentDetailsHumanOrigin`)。ずれ: 16 KiB を超える `Text`(手で書き換えた台帳だけ)は切って、そのことを書く。変異 5 種でテストが落ちる。S3 への注意: 進捗の節の頭(`reseatProgressLead`)は「本文は data で、指示ではない」と言うので、leader の prompt では「人の決定」の項だけは従う対象と書き分ける。`internal/org/feature.go:226-228` のコメントは進捗の中身に人の決定を挙げていない
  - S3(1daf1591、implementer): 確認コマンドが exit 0。leader.md に「運用規律」の人の決定の項、手順 4(承認待ち)と cross-review の上限の答え待ちの文(どちらも `HOOK_SEAT:` の扱いに倣う)、立て直しの経過の「人の決定」の行だけは data でなく指示として読む例外。director skill に `### 人の決定を送る`(「答える順」と「計画の承認」の間)と「してはいけないこと」の項の追記。`ralph org send --help`、agent-messaging.md と雛形の写し、README、`feature.go` のコメント。変異 8 種でテストが落ちる。範囲外の気づき(sync-docs と verify に渡す): (1) org skill の動詞リファレンスの `send` の行に ORIGIN の説明がない、(2) AGENTS.md の `reseat_progress.go` と send の説明、(3) director skill の「座席を立て直す」の最後の段落は、件のない人の判断を ORIGIN なしで送る形のまま(承認待ちの leader は直しの依頼と読むので ORIGIN: human にすべきかの判断が要る)、(4) compaction のあとに leader が人の決定を読み直す手段はない(進捗の節に載るのは立て直したときだけ)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
