# Walkthrough: leader-plan-depth(機能の org の leader の計画を、決まりとコードの読み込み・分かれ目・相手のモデルの計画助言で深める)

- Date: 2026-10-29
- Plan: docs/plans/active/2026-10-29-leader-plan-depth.md(/pr で archive に移す)
- Issue: #267
- Branch: feat/leader-plan-depth(この walkthrough の前の HEAD 83bb3077。merge を除く 34 commits と、origin/main を取り込んだ merge commit d59966a9。`git diff origin/main...HEAD` で 33 files、+2874/-108)
- Base: 7522835c(main、「Merge pull request #285」= #266)。分岐元は 9837f744 で、途中で main を merge commit で取り込んだ
- 読む範囲: 動きを決めるのは `internal/org/prompts/leader.md`(go:embed の leader の雛形)と `.claude/skills/director/SKILL.md`。テストは `internal/org/prompts_leader_plan_test.go`(新規)と `internal/org/director_skill_sync_test.go`。Go の実行コードは変えておらず、`internal/org/driver/paneenv.go` の警告文と doc comment、`internal/config/config.go` の doc comment だけが Go の差分。残りは skill の写し、`/org` skill、設定と文書の説明、記録
- 後工程は cycle 1 の中で回した。self-review は 5 回(LOW を L1〜L10 まで直し、L11 は Known gaps)、verify と test は 2 回ずつ(main の取り込みと直しのあとに打ち直した)。cross-review の所見は 0 件。記録だけの修正の例外は使っていない

```
何を変えるか → 読み込みと分かれ目(S1) → director の聞き方(S3) → 「まだ読まない」の文(S4) → 計画助言(S2) → レビューと main の取り込みで足した直し → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで: 機能の org の leader は雛形に計画を書くだけで、決まりやコードを読む手順も、分かれ目を人に聞く手順も、計画助言もなかった。
- この PR から:
  - leader は計画を書く前に AGENTS.md(あれば CLAUDE.md)・`.claude/rules/`・計画が触るコードを読む。
  - 機能の内側の分かれ目(`/plan` の手順 8 の 3 条件)を、推奨と理由つきで計画の `## Design decisions` に書き、per-plan では承認の QUESTION に `FORK <番号>: <問い> / a <案>(推奨) / b <案>` の行で全部載せる。split では推奨を採って記録する。
  - director は `FORK` の行を承認と一緒に聞く(3 つまでは 1 回の AskUserQuestion、4 つ以上は 4 問ずつ先に聞いてから承認)。推奨と違う答えがあれば approve を打たず、直しの DECISION を送る。leader は計画を直し、決まった分かれ目を `(決定済み)` の行にした 2 回目の QUESTION を上げる。
  - `[org.review].plan_advisory = true` なら、leader は承認の QUESTION の前(split では implementer の spawn の前)に、相手のモデルに読むだけの計画助言を頼み、要約を QUESTION の `ADVISORY:` の行(split では Design decisions)に書く。
- issue の文との違い: 「1 回の AskUserQuestion で聞く」は分かれ目が 3 つまでのとき(D3。Codex の計画助言の MEDIUM で、4 つ目からを推奨で決める初めの案をやめた)。「答えを DECISION に入れる」は推奨と違う答えのときだけで、それは直しの DECISION になる(D1。承認の DECISION の本文は固定の `approved` で、承認のあとに計画を変えると digest が合わず implementer が断られるため)。
- 計画の本文の読み替え: 本文の Scope の S2、D5、AC4・AC5、Test plan、Edge cases には、直す前の動き(`unknown command` の 1 だけ off、`pick_reviewer "$RALPH_PRIMARY_CLI"`)が残る。承認の digest を変えないため本文は直さず、Progress checklist で読み替えている(終了コードの 127 も off、driver は leader 自身の CLI で `RALPH_PRIMARY_CLI` が `claude` か `codex` のときだけ代わる)。
- release のノート: sync-docs の report の「Release notes」の節。雛形は go:embed なので、新しい binary で立てた leader から効く。director の skill は新しい skill を読んだ director から効く。この repo の `ralph.toml` は `plan_advisory = true` なので、ここで立てる機能の org の leader は計画助言を頼むようになる。
- 戻し方: この PR を revert する。台帳と設定の形は変えていない。

## 2. 読み込みと分かれ目(S1、95ad1e11)

- `internal/org/prompts/leader.md:143` の手順 1: 計画を書く前の読み込み、分かれ目の 3 条件、機能の内側に限ること、`## Design decisions` の書き方、分かれ目がなければ `Critical forks: None`、split の記録。最後は「この節の終わりの「計画助言」のとおりに扱う」の 1 文。
- `:199` の手順 3: `FORK` の行(推奨はいつも a、理由は書かない、数で打ち切らない、2,000 字に収まらなければ言い直す)と例(`:221`)、2 往復目の `(決定済み)`。
- `:241` の手順 4: `approved` で始まる DECISION は、すべての分かれ目が推奨で決まった意味も持ち、承認のあとは `## Design decisions` を書き換えない。推奨と違う答えは直しの依頼として届く。
- 手順の番号は振り直していない。

## 3. director の聞き方(S3、837cb430)

- `.claude/skills/director/SKILL.md:759` の「計画の承認」: 手順 1 に「人が会話にいなければ AskUserQuestion を使わず notify だけ」と「`ADVISORY:` の要約も伝える」。新しい手順 2(`:773`)が `FORK` の聞き方と、推奨と違う答えのときの直しの DECISION(推奨を選んだものも含めて全部の答えを書く)。元の 2〜6 は 3〜7 に振り直した。
- `/org` skill の「機能の計画の承認」に 1 段落。4 つの写しは一致(check-skill-sync)。
- テスト: `TestDirectorSkill_PinsTheApprovalForkSteps`(`internal/org/director_skill_sync_test.go`)。

## 4. 「まだ読まない」の文(S4、6189ef9b)

- `templates/base/ralph.toml` の `[org.review]` のコメント、`internal/config/config.go` の `OrgReviewConfig` の doc comment、`/org` skill: 機能の org の leader が `plan_advisory` を読むようになった。reviewer 座席の `cross_review` は #269。

## 5. 計画助言(S2、ce5bdbd1 と 6099187d)

- `leader.md:491` の `### 計画助言`(手順 16 の後ろ、節の終わり。L6 で手順 1 から移した)。
- 設定: `ralph config get org.review.plan_advisory` の終了コードは #266 の `/plan` 11.a と同じ(0 の true/false、`unknown command "config"` の 1 と 127 は off と記録、ほかは BLOCKED)。
- driver: leader 自身の CLI。`RALPH_PRIMARY_CLI` が `claude` か `codex`(大文字小文字を問わない)ならその値が代わる(`/plan` 11.b と同じ `case`)。`pick_reviewer` で相手を選ぶ。
- codex: `/plan` の手順 11.c の打ち方に、MCP を止める引数を足す。`--disable plugins` で plugin の持ち込む server を外し(plugin の server は `-c ...enabled=false` では止まらず、codex 0.163.0 が設定を読めなくなるため)、残る server ごとに `-c mcp_servers.<名前>.enabled=false`(欄がない・null の server も止める対象)。止めたあとの一覧で有効な server が 0 のときだけ呼ぶ。
- claude: #266 と同じ読むだけの形(`--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk`、`RALPH_WIP_CHECKPOINT=off`)。
- どちらも 20 分の見張り、呼ぶ前後の git の状態(HEAD、index、作業ツリー、未追跡のファイルの中身)を `cmp` で比べ、変わっていたら BLOCKED。終わったとみなすのは rc 0 で出力が空でないときだけ。呼べなければ飛ばして記録する。所見の HIGH と MEDIUM は反映するか、しない理由を書く。
- テスト: `prompts_leader_plan_test.go`(テスト関数 17)。文の pin のほかに、stand-in の codex と claude を相手に雛形のコマンドを bash と zsh で実際に打つ(止まらない助言者、書き換える助言者、MCP が止まらない場合、古い codex)。zsh がない環境では bash だけで走る。

## 6. レビューと main の取り込みで足した直し

- d52b6175(self-review Run 1 の L1〜L6): MCP の確かめを `select(.enabled != false)` に、director に `ADVISORY:` を伝える手順、テストの打ち切りの禁止の範囲と config.go の照合の仕方、テストの helper のまとめ、pane の値の扱いと警告文、計画助言の小見出しへの移動。
- cf0e5836(Run 2 の L7): pane の値は herdr の server の環境から来ることもあり、そのときはその値が leader の CLI に勝つと書いた。
- d59966a9(main の取り込み): #266 を取り込み、`paneenv.go` の doc comment の衝突を main の文を土台に解いた(L8 の「unset or empty」もここで直した)。
- 92554f77(/test の穴): 止まらない助言者の case の上限を 120 秒にし、書き換えの case を足した。
- 1aaadb12(Run 4 の L9・L10): 127 を off に、driver の読み方を `/plan` 11.b と同じ `case` に。
- 77c568fc(/sync-docs): README、`/org` skill、recipe、spec の日付の行、`ralph-config.sh`、model-routing、`xreview-helpers.sh`、tech-debt の台帳。

## 7. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(HEAD 20fa8316)
  - `./scripts/run-test.sh` rc 0(HEAD ff8404cf、Go 9 パッケージ、shell 45 本)
  - 変異: 終了コード、driver の読み方、MCP の止め方、見張り、git の比べ、`FORK` の形、director の手順などの変異は、どれもテストが落とす。生き残るのは driver の `case` の `claude` の枝を外す変異だけ(self-review の L11)
  - テストは本物の codex と claude を呼ばず、`~/.codex` に触れない(カナリアと空の HOME で確かめた)
  - AC8: この machine の codex 0.163.0 で、MCP を止める雛形の行を打つと有効な server が 0 になる(server の数だけを見た。モデルは呼んでいない)
  - cross-review(codex、cycle 1/2)の所見は 0 件
- 確かめていないこと:
  - 実機の leader と director の通し。本物の codex と claude での計画助言。codex の座席の sandbox から `claude -p` が通るか
  - self-review の L11(run の表に `claude` の枝を落とす行がない。足す 1 行は test の報告が確かめてある)。止まらない claude の case がない。終了コードの扱いは文の pin だけ
  - 走らせた shell は macOS の bash 3.2 と zsh 5.9 だけ
  - /sync-docs と cross-review のあとの `run-static-verify.sh`(文書と記録だけ。CI の verify で確かめる)
