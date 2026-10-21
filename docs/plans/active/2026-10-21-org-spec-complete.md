# org-spec-complete

- Status: Approved
- Approved: 2026-10-21 sha256:98781cfa6662
- Owner: Claude Code
- Date: 2026-10-21
- Related request: 機能ごとの org の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)を、9 段の完了が読み取れる形に直す。2026-10-21 のユーザーの返事「1. 直してください」
- Related issue: N/A
- Type: docs
- Branch: docs/org-spec-complete

## Objective

9 段はすべて merge 済み(最後は PR #244)だが、仕様は Rollout の 1 段目が「進行中」のままで、FR-1〜FR-11、NFR 5 つ、受け入れ条件 15 個の checkbox がすべて `[ ]` になっている。読んだ人が、どの段がどの PR で入り、どの項目が確かめられ、何が残っているかを仕様だけで分かるようにする。

## Scope

- 仕様の先頭(`#` の見出しの直後)に `## Status` の節を足す。書くこと: 9 段がすべて merge 済みであること、段ごとの PR 番号、試行の所見の直しの系列がこの PR から続くこと、release(タグ)はまだ切っておらず人が切ること。前例は `docs/specs/2026-08-17-overlay-scaffold-v2.md` の `## Status`
- Rollout の 1 行目の「進行中」を消し、9 つの段の行に PR 番号を添える
- FR-1〜FR-11、NFR 5 つ、受け入れ条件 15 個の checkbox を、下の「印の規則」で付ける。`[ ]` のまま残す項目には理由の注を 1 文で足す
- 「checkbox は付けない」と書いた FR の注の文(FR-7 の 7c-2 と 8a-3b の注、FR-8 の 8a-3a の注、FR-10 の 8b-1 の注、FR-12 の注)を、今の印の状態に合う文に直す
- FR-10 の 8b-1 の注が受け入れ条件を「121 行」「120 行」と行番号で指している 4 か所を、条件の文言で指す形に直す(今の行番号は AC9 と AC8 を指しており、ずれている)
- 印の根拠の表を `docs/evidence/org-spec-evidence-2026-10-21.md` に置く(項目ごとに根拠のテスト名・証拠のファイル・仕様の注の行、判定、`[ ]` の理由)

### 印の規則

- `[x]`: 項目が述べる動きが merge 済みのコードにあり、テストで確かめてある。項目が外の道具(herdr、codex、claude)に頼るなら、その道具で動かした実機の記録も 1 つある。そのうえで、試行の所見(`docs/evidence/org-trial-2026-10-20.md` の Findings と Trial deviations、`docs/tech-debt/README.md` の試行の行)のどれとも食い違わない
- 食い違うとは、項目の文が起きないと言うことを試行で見た、または項目の動きを起こすのに回り道が要ったこと。食い違う項目は、ほかの根拠がそろっていても `[ ]` のまま残し、食い違う所見の番号と tech-debt の行を指す
- 仕様の注が読み替えた項目(受け入れ条件の「見張りを kill した」と「push を含むが wake_cmd も送り先もない」)と、部品ごとのテストで確かめた項目は、読み替えた形で満たし、所見と食い違わなければ `[x]` にし、その注かテストを指す
- それ以外は `[ ]` のまま残し、足りないものを 1 文で書いて Open questions の該当の問いを指す
- 「9 段がすべて merge 済み」は Status の節で言い、項目の印とは分ける。印が `[ ]` の項目が残っても、段が終わったことは変わらない
- この系列の 2 本目以降の PR は、直した所見に結びつく項目の印と根拠の表を同じ PR で更新する

根拠の表(読み取りの調べ、2026-10-21)と試行の所見から見込む `[ ]` は次のとおり。

- 所見と食い違う: FR-2 と受け入れ条件「全 LLM のセッションを止めた」と NFR「安全の不変条件」(F4: agmsg_announce で失敗と記録された implementer が動き続けて TASK を処理した。`stop --all` は動いている座席だけを止めるので、その座席は止まらず、`ralph status` もその座席を動いていないと出す。この PR では止まらないことは再現していない。コードからの推定)、FR-5(F3: agmsg が leader の herdr の agent を改名し、director の DECISION が `agent_not_found` で届かなかった。F7: agmsg を polling する leader に `ralph org send` が 4 回待ち切れ、DECISION が 49 分後に届いた)、FR-6(F13: 見張りの escalation は通知の結果を記録しない。F6: 見張りの escalation は director が登録されていても受信箱を通らず人に直接届いた)、FR-7(F10: `ralph org regate` は `GOFLAGS` がなく `go test` の既定の 10 分で 2 つのパッケージが打ち切られ、`GOFLAGS=-timeout=30m` を付けて打ち直した)、FR-9(D-F3: director が F3 の回り道に `herdr agent rename` と `herdr pane rename` を打った。director に許すのは ralph と gh の操作だけ)
- 実機が足りない: FR-10(codex の director が timeout 付きの wait を繰り返す pull を実機で動かしていない。Open questions の 1 つ目)、NFR「ツール非依存」(機能の org の leader・implementer・reviewer を codex で動かした実機がない。試行はすべて claude。Open questions の 2 つ目)
- 注を付けて `[x]`: 受け入れ条件の「3 つの org」(3 org を 1 本で動かすテストはなく、台帳の解決と全 org の表示を別のテストで確かめた)、「`max_orgs` 個」(`start <task>` と spawn は `rejected` を記録し、`start --plan` の事前の検査は何も書かずに断る。4 段目の決定)、「見張りを kill した」(FR-6 の後半の注の読み替え)、「`auto_merge = false`」(merge を控えるのは skill と雛形の指示で、試行では director が merge してよい順を報告し、人に頼まれてから `ralph org merge` を打った)、「push を含むが」(FR-10 の 8b-1 の注の読み替え)、「`director start --driver codex`」(codex の director は `guarded` でしか動かない。FR-9 の 8a-2 の注)
- 所見のうち項目と食い違わないもの(F1・F2・F5・F9・F11・F12・F14、folder trust の行、G1)は、根拠の表に「関係する項目」と「食い違わない理由」を書き、仕様には書かない

## Non-goals

- 仕様の要件・設計・受け入れ条件の中身を変えること(印と注と Status と Rollout の PR 番号だけを変える)
- `[ ]` の項目を満たすための実機の確認やコードの変更(FR-10 と NFR の codex の実機は Open questions に残す)
- ほかの仕様(`docs/specs/2026-08-01-org-runtime.md` の 26 個の `[ ]` など)の印
- 試行の所見(F1〜F14、G1 など)の直し(この系列の 2 本目以降)
- release を切ること(ユーザーが切る)

## Assumptions

- 段ごとの PR 番号は memory の記録と `gh pr view` で確かめられる: 仕様 #205、1 段目 #207、2 段目 #208 #209、3 段目 #212、4 段目 #216 #218、5 段目 #220、6 段目 #222 #223、7 段目 #224 #225 #227 #228、8a 段目 #229 #231 #232 #234 #235、8b 段目 #236 #237 #238 #239、9 段目 #240 #243(試行の中で機能の PR #241 #242 を merge)、F8 の直し #244
- 根拠の表は 2026-10-21 の読み取りの調べの結果で、テストの名前と行は main 765bcc46 のもの
- 仕様の行番号を指す文書はほかにない(`git grep 'org-multi-org-director.md:[0-9]'` は reports・evidence・archive の外で 0 件)。Status の節を足して行がずれても、壊れる参照はない
- テストは仕様の checkbox を読まない(コードのコメントは FR の番号だけを指す)

## Affected areas

- `docs/specs/2026-10-07-org-multi-org-director.md`
- `docs/evidence/org-spec-evidence-2026-10-21.md`(新規)
- この計画

## Visual review

None (docs-only change: 仕様の印と注と Status の節を変えるだけで、部品のやりとり・状態・形式・境界のどれも変えない)

## Design decisions

- 印の規則は「Scope」の「印の規則」のとおり。「一部」の項目を全部 `[x]` にすると、codex の実機の穴が仕様から読めなくなる。全部 `[ ]` にすると、読み替えで満たした項目まで残りに見える。読み替えと部品ごとの確認は注で指して `[x]`、実機が足りない項目を `[ ]` にした
- 試行の所見と食い違う項目は `[ ]` に残す(Codex の計画助言 1、HIGH)。テストが通っていても、試行で項目の文が起きないと言うことが起きた項目に印を付けると、確かめていないことを確かめたと書くことになる。完了は Status の節で段の単位で言い、項目の印は系列の PR が所見を直すたびに付ける
- 根拠の表は `docs/evidence/` に置く。`docs/reports/` は名前の接頭辞ごとに check-sync の除外があり、新しい接頭辞は ROOT_ONLY で落ちる。`docs/evidence/` は丸ごと除外されている
- Critical forks: None(上の 2 つは規則と置き場所の選択で、戻すのに 1 スライスもかからない)

## Acceptance criteria

- [ ] AC1: 仕様に `## Status` の節があり、9 段がすべて merge 済みであること、段ごとの PR 番号(Assumptions の一覧)、`[ ]` のまま残る項目があってその理由(試行の所見と codex の実機)を根拠の表が持つこと、release は人が切ることを書いている。PR 番号は `gh pr view <n> --json state` で MERGED を確かめた番号だけ
- [ ] AC2: Rollout に「進行中」がなく(`grep -c 進行中` が 0)、9 つの段の行それぞれに PR 番号がある
- [ ] AC3: FR-1〜FR-11、NFR 5 つ、受け入れ条件 15 個の印が根拠の表の判定と一致する。`[ ]` の項目には、足りないものと Open questions の問いを指す注がある
- [ ] AC4: 仕様の中に「checkbox は付けない」「checkbox は、ほかの FR と同じく付けない」の類の文が残らない(`grep -n 'checkbox' <spec>` の各行が今の印と食い違わない)
- [ ] AC5: FR-10 の 8b-1 の注に「121 行」「120 行」が残らず、受け入れ条件を文言で指す
- [ ] AC6: 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` が 31 項目(FR 11、NFR 5、受け入れ条件 15)を持ち、各行に根拠(テストのパスと関数名、証拠のファイル、仕様の注のどれか 1 つ以上)と判定がある。表のテスト名は `git grep -n 'func <名前>('` で 1 件ずつ見つかる
- [ ] AC6b: 根拠の表に試行の所見の表がある。F1〜F14、D-F3、folder trust の行、G1 のそれぞれについて、関係する項目、食い違うかどうか、直す PR(この系列のどれか。決まっていなければ「未定」)を書く。食い違うとした所見に結びつく項目は、仕様で `[ ]` のままで、その所見と tech-debt の行を指す
- [ ] AC7: 仕様の要件・設計の文(印・注・Status・Rollout の PR 番号以外)は変わらない。`git diff` の変わった行が上の範囲に収まる
- [ ] AC8: `./scripts/run-verify.sh` が exit 0(check-sync と check-skill-sync を含む)

## Implementation outline

1. スライス 1(1 コミット): 根拠の表を書く。先に試行の所見の表(AC6b)を書き、次に項目の表を書く。読み取りの調べの結果を元に、テスト名を `git grep` で 1 件ずつ確かめ、判定と `[ ]` の理由を書く。PR 番号を `gh pr view` で確かめて表の末尾に載せる
2. スライス 2(1 コミット): 仕様を直す。Status の節、Rollout の PR 番号、checkbox の印と `[ ]` の注、「checkbox は付けない」の文、97 行の行番号の参照
3. 両方のスライスで `./scripts/run-verify.sh` を回す

## Verify plan

- Static analysis checks: `./scripts/run-verify.sh`(check-sync、check-skill-sync、markdown の検査を含む)
- Spec compliance criteria to confirm: AC1〜AC7 を grep と `git diff` で確かめる。印の数を `grep -c '^- \[x\]'` と `grep -c '^- \[ \]'` で数え、表の判定の数と照らす
- Documentation drift to check: 仕様の行番号を指すほかの文書がないこと(`git grep 'org-multi-org-director.md:[0-9]'`)。tech-debt の行が仕様の FR の印を前提にしていないこと
- Evidence to capture: verify の報告に grep と数え上げの結果

## Test plan

- Unit tests: なし(docs だけの変更)
- Integration tests: `./scripts/run-test.sh` が通ること(仕様を読むテストがないことの確認を兼ねる)
- Regression tests: `go test ./...` の結果が main と同じ
- Edge cases: 根拠の表のテスト名がすべて今のコードにある(改名で消えた名前を載せない)。PR 番号がすべて MERGED
- Evidence to capture: test の報告に run-test の結果、テスト名の照合の結果

## Risks and mitigations

- 根拠を過大に書く(テストが条件の一部しか見ていないのに満たしたと書く): 表の各行に、テストが確かめることを短く書き、self-review と verify で照らす
- 試行の所見と食い違う項目に印を付けてしまう: 所見の表(AC6b)を項目の表より先に書き、項目の印は所見の表から決める
- 系列の後の PR が印を付け忘れ、直したのに `[ ]` が残る: 根拠の表の所見の行に直す PR を書き、2 本目以降の計画の Scope に「仕様の印と根拠の表を更新する」を入れる
- 注の言い換えで要件の意味を変える: 印と注と Status と PR 番号の外の行を変えない(AC7)
- 長い FR の注の 1 文だけを直す編集で、まわりの文を壊す: Edit で該当の文だけを置き換え、`git diff --word-diff` で確かめる

## Rollout or rollback notes

docs だけなので、戻すときはこの PR を revert する。

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(Codex の計画助言 1 件、HIGH: 試行の所見と食い違う項目を `[ ]` に残す規則を足した)
- [x] Plan approved(ユーザーの常設の指示により自動承認、2026-10-21)
- [x] Branch created
- [x] Implementation started(S1 根拠の表 1a5bc2ee、S2 仕様の印と Status bc9fe9e0。S1 の implementer は run-verify のあとコミットの前に止まったので、表を確かめてから手でコミットした)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
