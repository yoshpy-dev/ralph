# Walkthrough: review-switches-skills(計画助言を相手のモデルにし、計画助言と cross-review を設定で切り替える)

- Date: 2026-10-29
- Plan: docs/plans/active/2026-10-29-review-switches-skills.md(/pr で archive に移す)
- Issue: #266
- PR: #285
- Branch: feat/review-switches-skills(この walkthrough の前の HEAD d6d5a4d0。origin/main 9837f744 から 64 files、+1297/-298)
- 読む範囲: 中心は skill の文(`/plan` の手順 11、`/cross-review` の手順 0〜9、`/pr`、`/implement` の 13.e。どれも 4 つの写し)。Go の変更は `internal/insights/backfill.go` の 1 つの判定と、テスト 2 本。残りは文書とコメント、計画と report
- 後工程は cycle 1 の 1 回。self-review は 2 回(1 回目の LOW を F1 で、2 回目の N1 をインラインで直した)。cross-review の指摘は 0。記録だけの修正の例外は使っていない

```
何を変えるか → Go の土台(S1) → /plan の手順 11(S2) → /cross-review・/pr・/implement(S3) → 文書(S4) → self-review の直し(F1、N1) → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで: `/plan` の計画助言は driver によらず codex を呼んだ。計画助言と cross-review は相手の CLI があれば必ず走り、なければ黙って飛んだ。
- この PR から:
  - 計画助言は相手のモデルが行う。driver は skill を動かしている CLI で、`RALPH_PRIMARY_CLI` が上書きする。
  - どちらの review も `[standard]` の設定(既定 off)で決まる。repo の root で `ralph config get` を打って読む。
  - 飛ばしたときは、計画、triage の report、PR の本文に理由が残る。
  - 計画助言で claude を呼ぶときは、読む道具だけを使い、MCP と project の hook を止め、前後の git の状態を比べる。
- release のノートは PR の本文。既定 off で動きが変わるので、ノートの 1 と 2 を先に読ませる。
- 戻し方: この PR を revert する。

## 2. Go の土台(S1、07cf95f8)

- `internal/insights/backfill.go:391-394` の field を `incomplete` から `noReview` に改め、`Reviewer status:` が `incomplete` か `skipped` で始まる cycle を verdict n/a にした(`:159`)。テストは skipped 単独と、cycle 1 が complete で cycle 2 が skipped の 2 本。
- `internal/config/root_ralph_toml_test.go:24` `TestRootRalphToml_KnownKeysAndReviewSwitchesOn`: ルートの `ralph.toml` をゼロ値の `Config` に `DisallowUnknownFields` で読む。`config.Load` は知らない表を無視するので使わない。知らないキーがあればキーと行を挙げて落ち、4 つの実施有無が true であることを確かめ、ファイルがなければ落ちる。

## 3. /plan の手順 11(S2、93dec2cc)

- `.claude/skills/plan/SKILL.md:73` の名前は「Other-model plan advisory (`[standard].plan_advisory`)」。
- `:74` の a: 設定の読み方。0 なら値を使う。1 で `unknown command "config"` のときと、127 のときは off として記録する。それ以外は止まる。
- `:79` の b: driver と reviewer を 1 回の Bash で決め、値を表示する。F1 で、`$DRIVER` を同じ呼び出しの中で定める形にした。
- `:84` の c: まず git の状態を保存する(HEAD、porcelain、`git diff --cached`、`git diff`、未追跡のファイルの中身のハッシュ)。そのあと codex を今の形で呼ぶか、claude を読むだけの形で呼ぶ。
- `:99` の d: 完了の契約と、後の git の状態の比較。
- `:100` 以降: 所見の扱い。結果は Progress の `Plan reviewed` に記録し、box も付ける(F1)。
- 試し用の repo での小さな実行: `--restricted` がないと、`-p` でも project の hook が走り、wip のコミットが入った。skill の形では、hook も git の変化もなかった。init の event は、道具が Glob・Grep・Read だけで MCP は 0。

## 4. /cross-review、/pr、/implement(S3、9e233825)

- `.claude/skills/cross-review/SKILL.md:30` の手順 0: `/plan` の 11.a と同じ読み方。番号は変えていない(insights の README などが手順の番号を名指すため)。
- 飛ばしたとき(off、古い ralph、ralph がない、相手の CLI がない(`:61`)、差分が空(`:63` の手順 4、F1)): 手順 1・2 だけを走らせ、手順 6(`:115`)で `Reviewer status: skipped (<reason>)` の report を書く。insight は `--verdict n/a`。
- `:49` の手順 2: driver の判別は `/plan` の 11.b と同じ。
- 手順 4: 両方の reviewer の道に、前後の git の状態の確かめ。claude の道には `RALPH_WIP_CHECKPOINT=off` も付ける。N1(e2d842cd)で、prompt の base を表示した値にした。
- `/pr`: Verification に計画助言と cross-review の行を書く。記録の語との対応表は F1 で足した。`/implement` の 13.e。triage の雛形の `skipped (<reason>)`。`.codex/AGENTS.override.md`。

## 5. 文書(S4、f88561b8、sync-docs 67410c6f)

- rules の 4 つ(と写し)、`README.md`、`AGENTS.md` と `.ralph/core/AGENTS.core.md`、`docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md`、`docs/recipes/codex-setup.md` で、「optional」を「設定で決まり、飛ばせば記録する」に直した。空の差分の条件は sync-docs で足した。
- コメント: `scripts/ralph-config.sh`、`scripts/xreview-helpers.sh`、`internal/org/driver/paneenv.go`、`tests/test-codex-exec-invocation.sh`、`.codex/config.toml`。
- `docs/tech-debt/README.md` の 31 行(`pick_reviewer` の呼び手は 2 つ)、154 行((c) の claude の道)、376 行(RC11)。
- `templates/base/ralph.toml` の `[standard]` のコメント(F1): 「in a later release」の文を消し、exit 127 も off とした。

## 6. 確かめたことと確かめていないこと

- 確かめたこと:
  - /verify: `run-static-verify.sh` は rc 0。PATH の ralph 6.0.0 は rc 1 で `unknown command "config"` を返す。build した binary は root で `true`、知らないキーで 2、型の誤りで 3、ralph がないと 127。
  - /test: `run-test.sh` は rc 0。変異は 7 個中 6 個が落ちた。driver を決める 1 行は、bash と zsh で 28 通りとも期待どおり。skipped の report は `count_triage_findings` で 0/0/0。HEAD の backfill は n/a と読んだ。
  - cross-review: 指摘 0。
- 確かめていないこと:
  - 両方の driver で、skill を通しで動かすこと。
  - Codex の sandbox の中から `claude -p` が通るか。
  - settings 経由の認証と `--restricted` の組み合わせ。
  - 変異 a3(base からある穴)。
  - この repo の PATH の ralph は古いので、merge のあと、この repo の skill は 2 つの review を「古い ralph」として飛ばす。
