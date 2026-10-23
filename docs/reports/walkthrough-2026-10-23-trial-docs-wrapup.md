# Walkthrough: trial-docs-wrapup(試行の所見を直す系列の最後、文書のまとめ)

- Date: 2026-10-23
- Plan: docs/plans/active/2026-10-23-trial-docs-wrapup.md(/pr で archive に移す)
- PR: ブランチ `docs/trial-docs-wrapup` の PR。系列の最後の PR なので、記録にもこの walkthrough にも番号は書き足さない(merge のあとは `gh pr list --state merged --head docs/trial-docs-wrapup` で引ける)
- Branch: docs/trial-docs-wrapup(この walkthrough の前の HEAD 454df239。19 commits、24 files、+1935/-88)
- Base: 3c1cd14b(origin/main、「Merge pull request #253」)
- 読む範囲: 新しい recipe が 2 写しで 486 行ずつ。`/org` と `/director` の skill が 4 写しずつ(org は +40/-13、director は +7/-3)。Go は leader の雛形 `internal/org/prompts/leader.md`(+3/-2)、その固定の文 `internal/org/prompts_test.go`(+1/-1)、新しいテスト `internal/org/report_names_sync_test.go`(+114)。記録は仕様(+6/-5)、根拠の表(+16/-11)、tech-debt(+8/-4)、repo-map(+1/-1)。残りは計画と、後工程の報告 5 本(cross-review は triage)と insight event
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → recipe(S1) → skill と雛形とテスト(S2) → 記録(S3) → 後工程と、その中の直し → 設計の判断 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行の所見を直す系列の 8 本目で、最後の PR。7 本目は #253(F7)
- 直すもの:
  - F1: 分離した herdr server のために `XDG_CONFIG_HOME` を移すと、pane の中の git と gh が `~/.config` の下の設定を失った
  - F2: pane の login の bash で macOS の `path_helper` が `/usr/bin` を前に出し、claude の Bash tool が試行の shim でなく本物の `osascript` を見つけた
  - folder trust: ralph は新しい repo の folder trust を前もって済ませない。試行では director が trust の画面にキーを送って答えた
  - `/org` skill の RESULT の例が reviewer の雛形(GATE・HEAD・BASE のヘッダ、`review-` で始まる report)と食い違っていた
  - skill と leader の雛形が、gate と regate は「数分」で終わると書いていた。試行の gate は 1375〜1990 秒かかった
- 直し方:
  - 新しい recipe `docs/recipes/isolated-herdr-server.md`(英語、templates/base に同じもの)が、分離のしかたを 1 か所に書く(F1・F2 を含む)
  - folder trust はコードで確かめず、人が repo ごとに 1 回答える手順を `/org` skill の前提に書いた(D1)。`/director` skill は、trust と bypass の画面にキーを送らないと言う
  - RESULT の例を雛形に合わせ、Go のテストで固定した(D5)
  - 「数分」を「repo のテストの量とマシンの負荷によっては数十分かかる」にした。試行の数値は根拠の表にだけ書いた(D2)
- 記録: 「fix/busy-leader」を「#253」に直し、PR の一覧を 36 本にした。F1・F2・folder trust の行を「この PR で直した」にし、tech-debt の 2 行を RESOLVED にした。試行の記録にだけあった watcher の時間切れ 6 回を、所見の表と tech-debt の新しい行にした。仕様の Status と根拠の表に、系列が終わったことと、開いたまま残る 3 行(leader への ALERT、Codex の leader、watcher の時間切れ)を書いた
- 動きは変わらない。leader の雛形の文は go:embed で binary に入るので、新しい binary で spawn した leader から効く
- 戻し方: この PR を revert する

## 2. 読む順(commit ごと)

1. be61a339 計画
   - Scope、Non-goals、Design decisions(D1〜D5)、AC1〜AC6 を読む。承認の digest は 932bafc59d5c で、verify と sync-docs が `./scripts/plan-visual.sh digest` で取り直した値と同じ。承認のあと、本文は変えていない(Progress と AC の checkbox だけが動く)
   - Codex の計画助言(HIGH 1、MEDIUM 1)は、台帳と見張りを分けること、片付けで見張りを止めることとして Scope の recipe の項目と AC1 に入った
   - 見ること: Non-goals。start で trust を確かめるコード、watcher の時間切れの直し、Codex の leader への send、leader への ALERT の原因、tech-debt の古い RESOLVED の注のブランチ名、recipe を通しで実機で打つこと、AGENTS.md の repo map の書き直し
2. e9884d9b、493e4ce8、e07bf30b S1 recipe(AC1)
   - `docs/recipes/isolated-herdr-server.md` の節の順: 「Why isolate, and what has to move」(`:33`)、「A repository with its own ledger」(`:64`)、「The environment」(`:103`。wrapper `/tmp/ralph-iso/run` と、F1 `:174`、F2 `:195`、shell の alias `:224`)、「Optional: a private agmsg and a notification shim」(`:242`)、「Start the server」(`:283`)、「Checks before the first start」(`:295`。server の確かめ、folder trust と捨ての pane の probe `:313`、`ralph doctor` と `ralph status` `:359`)、「Run」(`:377`)、「Clean up」(`:397`)、「Running again」(`:462`)、「What this recipe does not cover」(`:473`)
   - 分離するのは `XDG_CONFIG_HOME`・`XDG_STATE_HOME`・`HERDR_SOCKET_PATH` の 3 つ。herdr は `HOME` で設定と状態の置き場所を変えず、socket のパスには長さの上限があるので、短い `/tmp/ralph-iso` の下に置く。`env -i` で環境を空にし、wrapper の変数だけを渡す。macOS の claude は keychain を本当の `HOME` の下で読むので、`HOME` は変えない
   - 台帳も分ける。ほかの org が使う repo の linked worktree では台帳が共有され、`ralph org watch --all --detach` と `ralph org start` は、その台帳で動いている見張りを環境を比べずに使う。最初の start の前に `ralph status` が `org watch: not running` を言うのを確かめる
   - 片付けは、org を閉じ、見張りを止めて止まったことを確かめてから `herdr server stop` を打ち、専用の場所を消す。止める頼みが 30 秒で終わらなかったら、見張りが止まるまで消さない
   - 493e4ce8 と e07bf30b は、implementer への返事が行き違って足された直しで、受け入れた(amend はない)
   - 見ること: 利用者に固有のパス(`mise`、`/tmp/ot`、利用者の名前)が 0 件であること。templates の写しとバイト一致すること(`cmp`、`scripts/check-sync.sh`)
3. be8acda8 S2 skill と leader の雛形とテスト(AC1 の前提の 1 行、AC2〜AC4)
   - `/org` skill(`.claude/skills/org/SKILL.md`)の前提: bypass の項目(`:43`)の主語を「人が」にそろえた。「フォルダ信頼(repo ごとに 1 回)」(`:49-65`)は、人が最初の org の前に main のチェックアウトで claude を一度起動して trust すること、その下の `.claude/worktrees/` の機能の worktree は聞かれないこと(試行で確かめた)、repo の外の `--cwd` は別に要ること(推論で、確かめていないと書く)、codex は確かめていないので `codex trust .` を指すこと、画面で止まった座席は herdr が blocked と報告し send が届かないこと、`ralph org read` で読んで人が pane で答えることを書く。「分離した herdr server で動かすとき」(`:66`)が recipe を指す
   - `/org` skill の `gate` の行(`:221`)と「merge してよい順」の regate(`:1166`)、`/director` skill の「merge のあとの取り直し」の 1(`:908`)、leader の雛形の手順 12(`internal/org/prompts/leader.md:230`)の所要時間の文を「repo のテストの量とマシンの負荷によっては数十分かかる」にした。固定の文は `internal/org/prompts_test.go:830`
   - `/director` skill の「してはいけないこと」(`:952`)に、座席の pane のフォルダ信頼の画面と bypass の承諾の画面にキーを送って答えない(`herdr pane send-keys` も `ralph org send` も使わない)、人に伝えて人が答える、の 1 項目(`:962`)を足した
   - `/org` skill の RESULT の例(`:1332-1343`): `GATE: pass`・`HEAD:`・`BASE:` のヘッダを足し、EVIDENCE を `docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md` にした。SUMMARY は eed46c2c で雛形の severity 別件数の形になった
   - テスト(`internal/org/report_names_sync_test.go`): `codeBlockAfterLine`(`:196`)が、導入の行のあとの最初の囲みを読む。`TestCodeBlockAfterLine_TakesTheFirstBlockAfterTheLead`(`:220`、5 件)。`TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`(`:269`)が、root の skill の `## typed protocol` の節で「RESULT の例」(`orgSkillResultExampleLead`、`:243`)の行のあとの囲みを `Parse` と `Validate` に通し、`GATE` が `pass` の 1 つ、`HEAD` と `BASE` が 1 つずつ、本文に `EVIDENCE: ` + `reviewerReportPrefix` + `<YYYY-MM-DD>-<org_id>-<seat_id>.md` の行があることを見る。`reviewerReportPrefix` は既存の `TestReportNames_DocsMatchConstants` が雛形に結んでいる
   - 4 写しは `scripts/sync-skills.sh` を root と templates/base で 2 回打って揃えた(`scripts/check-skill-sync.sh` は 14 skill)
   - 見ること: テストが読むのは root の写しだけで、ほかの 3 写しは check-skill-sync が見ること。skill の 3 か所の「数十分」はテストで固定していないこと(下の「確かめていないこと」)
4. 63e629a3、f310e994 S3 記録(AC5)
   - 仕様(`docs/specs/2026-10-07-org-multi-org-director.md`): 一覧を 36 本にし(`:5`)、「試行の所見 F7 の直し: #253」と、この PR をブランチ名で書く行(`:26-27`)を足した。Status の節(`:31`)は、系列が予定の順に進んでこの PR で終わったこと、開いたまま残る 3 行を書く。FR-5(`:93`)と NFR ツール非依存(`:152`)の注の「fix/busy-leader」を #253 にした。項目の印は変えていない(FR-10 は `[ ]` のまま。f310e994 が Status の FR-10 の理由を根拠の表の言い方にそろえた)
   - 根拠の表(`docs/evidence/org-spec-evidence-2026-10-21.md`): 前置き(`:16`)、F1・F2(`:20-21`)と folder trust(`:35`)を「`docs/trial-docs-wrapup` で直した」に、F7(`:26`)を #253 に、watcher の時間切れの行(`:38`)を足し、gate と regate の所要時間と load average の段落(`:42`)を足した。NFR 入れ方(`:65`)は 36 本と 33 本、段ごとの PR に #253 の行とこの PR の行(`:108-109`)、確かめていないことに 2 項目(`:121-122`)
   - tech-debt(`docs/tech-debt/README.md`): 分離した herdr の行(`:318`)と folder trust の行(`:340`)を取り消し線と `<!-- RESOLVED ... -->` の注の形にした。watcher の時間切れの行(`:344`)は、原因を確かめていないことと、watcher も `watcherInvokeTimeout` も変えないことを書く。F7 の注(`:327`)とファイルの大きさの行(`:355`)の「fix/busy-leader」を #253 にした
   - 見ること: `grep -rn 'fix/busy-leader' docs/specs docs/evidence docs/tech-debt` が 0 件(base では 3・6・2 件)。この PR は 36 本に数えないこと(D3)
5. 後工程と、その中の直し
   - 58ca93cc self-review(Merge yes、LOW 8・INFO 3)。5ac0af89 で直した: F1(Clean up のコマンドをすべて wrapper から打つ)、F2(`ralph status` を repo から打ってから `cd ~` し、`pgrep` と `lsof +D` をそこから打つ)、F3(server の環境の確かめを `ps eww` の形に)、F4(`mkdir -m 700 /tmp/ralph-iso` を `-p` なしで打ち、あれば使い回さない)、F5(`herdr --help` の確かめを観測した出力の形に)、F6(昇格した director は `ralph` と `gh` を wrapper で打ち、herdr のコマンドは人のもの)、F7(テストの doc に、雛形は読まず `reviewerReportPrefix` で結ぶと書く)、F8(watcher の時間切れの行の言い回し)、I1(sandbox ではない段落)、I2(`env -i` で `SSH_AUTH_SOCK` と `GH_TOKEN` が落ちる)、I3(`find` が出した `session.json` を消す)。16cc451b の再レビューの N1(30 秒の分岐の `ralph status` を repo から打つ理由)、N2(環境の確かめを `cut -d= -f1` で名前だけに)、N3(`--help` の観測に日付)は f15b696c で直し、edf04ef7 の再レビューは新しい LOW 以上なし。N4(template の recipe が scaffold に届かない `docs/evidence/` を出典にする)は sync-docs が tech-debt の行(`:360`)にした
   - c6d98b0a verify(pass、INFO 4)。eed46c2c で直した: INFO-1(`ps` が環境を読めないと名前が 1 つも出ない、と書く)、INFO-2(RESULT の例の SUMMARY を件数の形に)、INFO-3(repo の外の `--cwd` の trust は推論で確かめていないと、skill と recipe と根拠の表に書く)、INFO-4(計画の印)。c300f1a9 の再確認で pass のまま
   - 6053ab00 test(pass)。`./scripts/run-test.sh` が c300f1a9 で rc 0(942 秒)、`go test ./internal/org/... ./internal/cli/... -count=1 -timeout 30m` が go1.26.0 と `GOTOOLCHAIN=go1.25.8` で rc 0。mutant 8 本はすべて赤(M1: EVIDENCE を `self-review-foo.md` に、M4: `GATE:` を消す、M5: 導入の行の名前を変える、M3 と M3b: leader の雛形の文を戻す、ほか)。c8f17b07 が AC6 に印を付けた
   - 957720d3 sync-docs。repo-map の recipe の行(`docs/architecture/repo-map.md:37`)と tech-debt の N4 の行を足した。`ralph org gate --help` の「They can take minutes」(`internal/cli/org_gate.go:55`)はコードなので残した
   - 454df239 cross-review の triage(cycle 1/2、指摘 0)。`codex exec review --base main` を read-only で打ち、指摘なしで終わった。codex は read-only の sandbox が Go の build ディレクトリを書けないので、Go のテストは走らせていない
   - insight event は `docs/insights/events/2026-10-23-trial-docs-wrapup.jsonl` の 5 行(self_review、verify、test、sync_docs、cross_review。どれも cycle 1)

## 3. 設計の判断(計画の D1〜D5)

- D1 folder trust は文書だけにした。start で確かめるには claude の `~/.claude.json` を読むことになり、その形は claude が決めて公開していない。`ralph doctor` も codex の trust を Go から読んでいない
- D2 所要時間は一般の言い方にした。skill と leader の雛形は `ralph init` で downstream に配られ、テストの量は repo ごとに違う
- D3 この PR はブランチ名で書き、番号を書き足さない。仕様の一覧は「すべてが merge 済みであることを `gh pr view` で確かめた」と書くので、merge の前の PR を数に入れると偽になる。番号を書くコミットをあとから足す案は、一覧の本数の文とぶつかる
- D4 recipe は templates にも置いた。scaffold に配る `/org` skill が recipe を指すので、置かないと downstream でリンクが切れる
- D5 RESULT の例をテストで固定した。#244 の sync-docs が残した食い違い(例は `self-review-foo.md`、雛形は `self-review-` で始まる名前を使わない)が、また開かないようにする

## 4. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 recipe | 2 写しが `cmp` で一致し、`scripts/check-sync.sh` が rc 0。Scope の項目は verify の表で 1 つずつ確かめた。利用者に固有のパスは 0 件。`/org` skill の前提 `:66` が指す |
| AC2 folder trust | `/org` skill の前提 `:49-65`、bypass `:43`、`/director` skill `:962`。`scripts/check-skill-sync.sh` が rc 0 |
| AC3 RESULT の例 | `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate` が通り、mutant M1・M4・M5 ほかが赤 |
| AC4 所要時間 | AC4 の grep が gate と regate について 0 件。`TestRenderRolePrompt_Leader_FeatureOrgGate` が新しい文で通り、M3・M3b が赤 |
| AC5 記録 | `fix/busy-leader` が 3 つのディレクトリで 0 件。36 本と `gh pr view 253` の読み、F1・F2・folder trust、watcher の時間切れ、系列の終わりは上の 2 の 4 |
| AC6 | `./scripts/run-verify.sh` が f310e994 で rc 0。Go のテストは両方の版で通った(test の報告) |

## 5. 確かめていないこと

- recipe を通しで実機で打っていない。手順は試行(2026-10-20)と 2026-10-22・23 の実機の確かめから写した。確かめた版は herdr 0.7.5、Claude Code 2.1.291、agmsg 1.5.1 で、すべて macOS
- repo の外の `--cwd` の trust は、試行で見た規則からの推論で、確かめていない。codex の trust の画面も確かめていない
- skill の 3 か所の「数十分」を固定するテストはない(AC4 は grep で見る)。`ralph org gate --help` の「They can take minutes」は残した
- verifier と orchestrator が、分離していない環境で `herdr --version` を打った(版を読むだけ)。verifier が、10 分以内に herdr のファイルが変わっていないことを確かめた
- 開いたまま残る行: leader への ALERT(原因が分からず、FR-3 は `[ ]`)、Codex の leader(NFR ツール非依存は `[ ]`)、watcher の時間切れ、FR-10(codex の director の pull を実機で走らせていない)、N4(template の recipe の出典)
