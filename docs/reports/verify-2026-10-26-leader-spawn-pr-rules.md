# Verify report: leader-spawn-pr-rules

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-26-leader-spawn-pr-rules.md`(AC1〜AC6、D1〜D6。パイプライン cycle 1/2)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 619cf5f8...HEAD`(HEAD 0309acf8、14 ファイル、+416/-59)。コミットは 70cc1d77(計画)、efddd301(`feature.go`・`leader.md`・テスト 2 本)、c654d596(`/org` skill の 4 写し)、7d51236d(記録)、f9fe07ea(計画の印)、43dc55e4(self-review の report)、0309acf8(self-review の F1・F2 の直し)。テストは打っていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-26-leader-spawn-pr-rules.log`(git が無視する場所)。`run-static-verify.sh` の出力、`go vet` の 2 版の出力、下の 2 つの probe の出力を入れた。`run-static-verify.sh` 自身も `docs/evidence/verify-2026-10-26-062258.log` を書いた

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1(RC2) | 満たす | `featureLeaderTask` が `- 予約したパス: <パス>(座席の spawn には --scope <引数> を付ける)` を書き、引数は `shellQuote(strings.Join(f.Reserve, ", "))`(`internal/org/feature.go:412`、`shellQuote` は `:448`)。`- 台帳:` の行と同じ形。leader の雛形の手順 5(`internal/org/prompts/leader.md:157-163`)は implementer と reviewer の両方にその引数を書き換えずに付けると書き、手順 3(`:141-143`)は承認前の reviewer も `--cwd .`・`--model`・`--scope` を手順 5 と同じにすると書く。ミッションの 2(`:24-27`)は `--scope` が `autonomous` で必須だと書き、手順 5 を指す |
| AC1(spawn の検査を満たす形か) | 満たす(コードを読んだ。実機は未確認) | 検査は `autonomousScopeGateErr`(`internal/org/spawn.go:1693`)で、`mode == autonomous && p.Scope == "" && len(p.Reserve) == 0 && !p.AllowUnscoped` のときだけ断る。値の形は見ない。`--scope` は cobra の `StringVar`(`internal/cli/org.go:824`)で `SpawnParams.Scope` にそのまま入る(`:810`)。予約が空になる経路はない。`parseSplitReserve`(`internal/org/split.go:562`)は空の値を断り、`- Reserve:` の行がない機能は `:612` で断られ、`normalizeReservePath` は空のパスを断る。全体の予約は `.`(`reserve.go:39`)で、引数は `'.'` になる。`shellQuote` を python に写して `/bin/sh -c 'set -- <引数>; printf ...'` に読ませた probe では、10 組(下の Observational checks)がすべて `$#` 1 で元の値と一致した。値が `-` で始まる予約でも、pflag は `--flag arg` の形で次の引数をそのまま値にする(`spf13/pflag` の `flag.go:1018-1021`)。implementer と reviewer の座席は `--reserve` を受けない(エラー文「--reserve on the leader seat only」)ので、`--scope` だけが検査を満たす手段である |
| AC1b | 満たす(テストの中身を読んだ。テストの実行は `/test`) | `TestFeatureLeaderTask_ScopeIsOneShellWord`(`internal/org/feature_test.go:593`)が 7 組の予約(1 つ、2 つ、`$`、バッククォート、単一引用符、二重引用符、`$HOME`・`'`・`"`・バッククォート・`\`・`*` を混ぜた 5 つ)を `/bin/sh -c "set -- <arg>; printf '%s\|%s' ..."` に読ませ、`"1\|"+want` と比べる。fixture は先に `normalizeReservePath` がそのまま通すことを `t.Fatalf` で確かめる |
| AC2(RC5) | 満たす | 手順 14(`leader.md:285-289`)が `/pr` skill の手順(`.claude/skills/pr/SKILL.md`、Codex では `.agents/skills/pr/SKILL.md`)の題と本文の決まり(言語、題の prefix など)と `template.md` に合わせ、`/pr` skill そのものは実行しないと書く。名指された 8 ファイル(root と `templates/base/` の `.claude` と `.agents` の `pr/SKILL.md` と `pr/template.md`)はすべて実在する。4 つの `SKILL.md` の手順 5(`:31`)に「PR title and body must be written in Japanese」と、`./scripts/branch-name.sh title-prefix` による題の prefix の決まりがある。`template.md` の 2 写しに言語の決まりはない(`日本語`・`Japanese` の当たりは 0)。これは計画の前提どおりで、雛形が `SKILL.md` の手順を名指す理由になっている。`scripts/branch-name.sh` は root と `templates/base/` の両方にある |
| AC3 | 満たす | `/org` skill の leader のサイクルの手順 2(`.claude/skills/org/SKILL.md:1306-1313`)と「org の終わり方」の手順 6(`:1002-1010`)が leader の雛形と同じ向き。`cmp` で `.claude` と `templates/base/.claude`、`.agents` と `templates/base/.agents` が一致し、`.claude` と `.agents` は frontmatter の後ろで一致する。`check-skill-sync.sh` は「14 skill(s) in lock-step」 |
| AC4 | 満たす(テキストの検査を写して確かめた。`go test` は `/test`) | `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の新しい pin 6 つ(ミッションの `missionScope`、手順 5 の 3 句、手順 3 の 1 句、手順 14 の `/pr` の 1 句)を、`markdownSection`・`markdownItem`・`containsPhrase` ごと python に写し、base 619cf5f8 と HEAD の `leader.md` に掛けた。6 つとも HEAD で通り、base で落ちる。残した pin(`--id` の文、「`/pr` skill そのものは実行しない」)は両方で通る。文を 1 つ戻すと対応する pin が落ちることになる。`featureLeaderTask` の行は `TestFeatureLeaderTask` の完全一致と、`prompts_test.go:556-559` の `t.Fatalf` の 2 か所が固定する |
| AC5 | 満たす | tech-debt の RC2(`docs/tech-debt/README.md:365`)と RC5(`:369`)は、Debt と Trigger の欄が取り消し線で、ブランチ名の `<!-- RESOLVED 2026-10-26 in fix/leader-spawn-pr-rules ... -->` の注(`:364`、`:368`)を持つ。形は前例(`:325-326` の #251)と同じ。どちらの行も 5 欄。仕様の FR-4(`docs/specs/2026-10-07-org-multi-org-director.md:80`)と受け入れ条件 5(`:173`)は `[ ]` のままで、「印(2026-10-26)」(`:90`、`:176`)がある。`[ ]` は base も HEAD も 5、`[x]` は 27。根拠の表の RC2(`docs/evidence/org-spec-evidence-2026-10-21.md:53`)と RC5(`:56`)の「直す PR」は `fix/leader-spawn-pr-rules` で、tech-debt の行の欄は `~~..~~`(RESOLVED の行)の形。表の判定の数(`:105`、26 と 5)は変わらない。Status の段落(`:37`)の「開いたまま残る 7 つ(RC1、RC3、RC4、RC6、RC9、RC10、RC11)」は、根拠の表で「直す PR」が「未定」の 7 行と一致する |
| AC6 | 一部(静的な部分だけ) | `./scripts/run-static-verify.sh` は rc 0。`go vet ./internal/org/` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で rc 0(テストのファイルもコンパイルされる)。`./scripts/run-verify.sh` のテストの部分と、Go のテストを 2 版で通すことは `/test` で確かめる |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed、HEAD 0309acf8) | rc 0、`All verifiers passed.` | `Language packs selected: golang`。shellcheck、hook の `sh -n` 20 個、guard の awk の構文、`jq -e`(settings.json 2 つ)、Codex の hook の 3 検査、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照が OK。Go は `gofmt: ok` と golangci-lint の `0 issues.`。`secret-scan-branch: scanned 619cf5f8..0309acf8 against origin/main: clean` |
| `go vet ./internal/org/`(go1.26.0) | rc 0 | |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/org/` | rc 0 | go.mod は `go 1.25.8`。新しいテストの `strings.SplitSeq`(1.24 から)と `strings.CutPrefix` は使える |
| `git diff --check 619cf5f8...HEAD` | 出力なし、rc 0 | |
| 表の欄の数(`(?<!\\) \| ` で分割) | 変わった行は base と同じ数 | tech-debt の 365 行と 369 行は 5 欄。根拠の表の 52・53・56 行は 6 欄、76・93 行は 4 欄で、base と同じ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| leader の雛形と `/org` skill の `spawn` の行(`.claude/skills/org/SKILL.md:208`) | 一致 | 行は `--scope` を「担当範囲の説明。autonomous では必須」と書き、雛形のミッションの 2 と手順 5 も同じことを言う。「permission 作法」の節(`:1409-1411`)の「`--scope` は担当範囲を短く書く(例: `"internal/org/**"`)」は説明の文字列の例で、予約を `, ` でつないだ値もその範囲に入る |
| leader の雛形と implementer・reviewer の雛形の scope | 一致 | `implementer.md:4` と `reviewer.md:4` は `{{SCOPE}}` を表示し、implementer は scope の中のパスだけを `git add` する(`implementer.md:21`)。reviewer は report と gate の生成物を scope の外でも書いてよい(`reviewer.md:10-12`、`:87-88`)。雛形の手順 5 の理由の文と合う。通しの確かめの manifest では、leader が自分で足した `--scope` のうち 3 つが予約と同じパスだった(implementer 2 席、backfill-na の reviewer)。残る 1 つは hooks-matcher の reviewer の `review hooks-matcher; writes docs/reports/ only` で、D2 の判断と合う |
| 「機能ごとの org」の導入(`leader.md:61-70`) | 一致 | 予約の外に書くもの(手順 1・4・6 の計画、9 の reviewer のレポート、10 の report、12 の戻し)は変わらず、`--scope` を予約にしても導入と食い違わない |
| `/org` skill の `start --plan` の説明(`.claude/skills/org/SKILL.md:651-670`) | 一致(情報: 足してもよい) | タスクの行の並び(「予約」を含む)は今も正しい。`- 台帳:` の行には段落 1 つ分の説明があるが、`- 予約したパス:` の行が新しく持つ `--scope` の引数の説明はこの節になく、leader のサイクルの手順 2(`:1306-1310`)にだけある。古い形を書いている箇所ではない |
| AGENTS.md の repo map、README | 一致 | AGENTS.md は `feature.go` を「worktree + bound leader with the recorded `[org.approval].mode`」と書き、タスクの行には触れない。README の機能の org の段落(`README.md:252`)は「leader が `gh pr create` で PR を開く」とだけ書き、言語や spawn の flag には触れない。どちらも前の形を書いていない |
| ほかの docs | 一致 | `予約したパス` の当たりは、雛形、テスト、`/org` skill、記録(仕様、根拠の表、tech-debt)、過去の report だけ。`/director` skill と `director.md` には `--scope` も PR の言語もない。古い文言「`/pr` skill の雛形に合わせてよいが」は、計画が引用しているものを除いてもう残っていない |
| tech-debt RC4(`docs/tech-debt/README.md:367`) | 引き継ぎの注がない(LOW、V1) | Trigger の「The next change ... to the feature-org steps of `internal/org/prompts/leader.md`」が、この PR の手順 3・5・14 の変更で当たった。RC4 は計画の Non-goals でユーザーが release のあとに回したので直す必要はないが、行は base と同じで「carried over by fix/leader-spawn-pr-rules」の注がない |
| tech-debt のファイルの大きさの行(`:357`) | 引き継ぎの注がない(LOW、V2) | Trigger の「The next change to `Send` or to the leader template tests」が `TestRenderRolePrompt_Leader_FeatureOrgProcedure` の変更で当たった。`internal/org/prompts_test.go` は 2165 行から 2189 行(+24、閾値の +300 より下)。分ける作業は別の PR という行の判断のままでよいが、注がない |
| tech-debt org-feature-worktree の小さな所見(`:197`、(g)) | 前からの古い記述(LOW、V3。この PR が原因ではない) | (g) N2 は「導入が予約の外への書き込みを leader の手順 1・4・5 にしか許さず、reviewer のレポートが抜けている」と書き、Trigger は「the next time the feature-org section of `leader.md` is edited」。この PR で Trigger が当たった。導入はすでに「9 の reviewer のレポート」を挙げており(f1b0352e から)、(g) は直っている |
| 計画の Progress checklist | 遅れている(情報) | self-review の report(43dc55e4)はあるが「Review artifact created」が `[ ]`。AC6 の `[ ]` は `/test` を待つもので正しい |

## Observational checks

- `shellQuote` の写し(`'` + `'` を `'\''` に替えた値 + `'`)を `python3 -I` から `/bin/sh -c "set -- <arg>; printf '%s|%s' \"$#\" \"$1\""` に渡した。予約は `.`、`a/`、`docs/auth/, docs/x.md`、`app/routes/$slug.tsx`、``a/`id`/``、`it's/x.md`、`say"hi"/`、5 つを混ぜたもの、`-x/, docs/日本語.md`、`a/$(id)/` の 10 組。10 組とも `1|<元の値>` になり、展開も分割も起きなかった
- `--scope` を読むコードは `{{SCOPE}}` の置換(`internal/org/prompts.go:112`)と `spawned` の Details の `scope=%s`(`spawn.go:1711`)だけで、`scope=` を分けて読む処理は Go のソースにない。`, ` を含む値でも読む側は壊れない
- 仕様の Status(`:37`)と 2 つの「印(2026-10-26)」は、self-review の F1 の直し(0309acf8)のあと、テストが固定したもの(行の形、`/bin/sh` が 1 語に読むこと、雛形の文)と、実機で確かめていないこと(leader がそのとおりに打つこと、PR を日本語で書くこと)を分けて書いている。tech-debt の注の「not run live」と強さがそろった

## Coverage gaps

- 実機: 新しい binary の leader が手順 5 のとおりに `--scope` を付けて spawn し、それが検査を通ること、PR の題と本文を日本語で書くことは、どこでも確かめていない(計画の R3 と Non-goals)。FR-4 と受け入れ条件 5 の `[ ]` は次の実行で付け直す
- Go のテストの実行(`TestFeatureLeaderTask*`、`TestRenderRolePrompt_Leader_FeatureOrgProcedure`)と 2 版での `go test` は `/test` の範囲。AC4 の「戻すと落ちる」は、テキストの検査を写した probe で確かめたもので、Go のテストそのものは打っていない
- codex の leader が `.agents/skills/pr/SKILL.md` を読むことは確かめていない(ファイルがあることだけ確かめた)
- 足すとよい決定論の検査: なし。行の形、引用、雛形の文は、今あるテストと今回足したテストで固定されている

## Verdict

- Verified: AC1、AC1b(テストの中身と probe)、AC2、AC3、AC4(テキストの検査を写した probe)、AC5。静的解析はすべて通った(`run-static-verify.sh` rc 0、`go vet` の 2 版 rc 0)。手順 5 の `--scope` は空でない 1 つの引数として spawn に届き、`autonomousScopeGateErr` を満たす
- Partially verified: AC6(静的な部分だけ。テストは `/test`)
- Not verified: 実機での leader の振る舞い(spawn と PR の言語)
- Findings: LOW 3 件(V1 RC4 の行、V2 ファイルの大きさの行、V3 (g) の古い記述。どれも tech-debt の注で、`/sync-docs` で直せる)と情報 2 件(`/org` skill の `start --plan` の説明、計画の Progress checklist)。CRITICAL・HIGH・MEDIUM はない
- 判定: pass
