# Walkthrough: drop-org-shell-defaults(shell の [org] 既定値の写しを消す)

- Date: 2026-10-28
- Plan: docs/plans/active/2026-10-28-drop-org-shell-defaults.md(/pr で archive に移す)
- Issue: #273
- PR: #281
- Branch: refactor/drop-org-shell-defaults(この walkthrough の前の HEAD d5d7e7f8。origin/main 4389396f から 16 files、+858/-318)
- 読む範囲: 実行コードの変更は 2 つの `ralph-config.sh`(byte 一致、行の削除だけ)。Go の変更はテスト `internal/config/defaults_sync_test.go` と `config_test.go`、それに `config.go` のコメント。文書は spec の 2 か所、tech-debt の 1 行、`model-routing.md`、`tests/test-ralph-config.sh` の冒頭の注記。残る +500 行前後は計画、後工程の report 5 本、insight event。
- 後工程は cycle 1 の 1 回で済んだ。cross-review の所見は 0 件。記録だけの修正の例外は使っていない。

```
何を変えるか → 照合のテスト(S1) → shell の写しの削除(S2) → 文書(S3) → self-review の直し → main の取り込み → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで: `[org]` の既定値を `Default()`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` の 3 か所に書き、`TestDefaultsLockStep` が 3 面の一致を見ていた。`ralph org` は shell の値を読まないので、shell の写しは直す手間だけを生んでいた。
- この PR から:
  - `scripts/ralph-config.sh` とテンプレートの写しから、`RALPH_ORG_*` の 18 変数と export の 3 行が消える。`RALPH_ORG_AGMSG_HOME` だけが残る(export しない)。
  - 照合は `Default()` 対テンプレートの 2 面で、19 キーの値とキーの有無を見る。`agmsg_home` だけは shell の値とも比べる。
- release のノート: changelog に出るのは `refactor:` の 76c7595d と、`fix:` の 7d59e278・1f9dd1b6。後ろの 2 件はテストと self-review の直しなので、ノートには PR 本文の 1 行(「`scripts/ralph-config.sh` から `RALPH_ORG_*` の既定値の写しを消した。`ralph org` はもともと読んでいない。`RALPH_ORG_AGMSG_HOME` は残る」)を使う。
- 戻し方: この PR を revert する。

## 2. 照合のテスト(S1、e948b85d)

- `internal/config/defaults_sync_test.go:112` `orgTemplateProbe`: テンプレートの `[org]` を pointer の field で受ける struct。go-toml/v2 はキーがない field を nil のまま残す。
- `:148` `orgTemplateDrift`: probe と `Default()` を 19 キーぶん比べ、ずれを 1 行ずつ返す。値の比較は `:179` `valueDrift`、list の比較は `:191` `listDrift`。nil の field は「missing from the template」として返す。
- `:202` `TestDefaultsLockStep`: `:207` の `org` がテンプレート対 `Default()`、`:226` の `shell_and_skills` が shell の `RALPH_ORG_AGMSG_HOME` 対 `Default()` と SKILL.md の fallback の照合。
- `:314` `TestOrgTemplateDrift_DetectsChangedAndMissingKeys`: 実際のテンプレートでずれが 0 行、値を変えたテンプレート、キーを消したテンプレート(`auto_merge`、`codex_verified`、`wake`)、`model_pool` の 2 つの入れ替えで、それぞれちょうど 1 行。
- 見ること: `Load` は `Default()` から読み始めるので、`Load` の結果を比べるだけでは、テンプレートから消えたキーに気づけない。probe struct はこの穴を塞ぐためのもの(計画の D1、consult の助言で `map[string]any` の二重読みから変えた)。

## 3. shell の写しの削除(S2、76c7595d)

- `scripts/ralph-config.sh:46-70`: `[org]` の節は `RALPH_ORG_AGMSG_HOME` の 1 行(`:56`)と、export しない理由のコメント(`:60`)だけになった。`:70` の export は標準フローの 3 変数のまま。
- `templates/base/scripts/ralph-config.sh` は byte 一致(`check-sync.sh` が見る)。
- コメントに issue の URL を書いていないのは、`check-template-purity.sh` がテンプレートの中の `yoshpy-dev` を禁じるため。

## 4. 文書(S3、c56fa63c)

- `docs/specs/2026-08-01-org-runtime.md:45` の運用の注記 (a): 3 面から 2 面とキーの有無の確認へ。`:74` の FR-2 に #273 の印。
- `docs/tech-debt/README.md:149`: 「three value surfaces」を「two value surfaces」に。
- `.claude/rules/ralph/model-routing.md`(root だけ。check-sync の KNOWN_DIFFS): `defaults_sync_test.go` の説明に、`[org]` の照合と、その対象外のキーを書いた(/sync-docs の 097c7f35 で対象を絞った)。
- `internal/config/config.go:30` のコメント: 既定値はテンプレートと lock-step で、`agmsg_home` は shell とも比べる。

## 5. self-review の直し(7d59e278)

- self-review の LOW 5 件のうち L1・L2・L3・L5 を直した。L3 で `agmsg_home` の 3 面の照合が「shell 対 `Default()`」と「テンプレート対 `Default()`」の 2 つのサブテストの組み合わせになった(計画の Progress checklist に Deviation として記録)。L2 で、テンプレートがないときも SKILL.md の照合が走る。
- L4(照合するキーの数 19 がコメントにしかない)は Known gaps。

## 6. main の取り込み(fe093132、1f9dd1b6)

- #280 を merge した。テキストの衝突はなかったが、#280 が `TestLoad_TemplateRalphToml`(`config_test.go:76`)に足した `[doctor]` の生の文字列の確認が、この branch で外した `data` を使っていて build が落ちた。
- 1f9dd1b6 で `config_test.go:101` にテンプレートを読む行を戻した。#280 の注記「The same holds for [doctor]…」が指していた段落はこの branch で消えたので、理由(`Load` は欠けたキーに `Default()` の値を残す)をその場に書き直した。

## 7. 確かめたことと確かめていないこと

- 確かめた:
  - /verify: `run-static-verify.sh` rc 0(golangci-lint 0 件、go vet と staticcheck は何も出さない)、`check-sync.sh`・`check-template-purity.sh`・`check-skill-sync.sh` が通る。
  - /test: `go test ./internal/config/` の対象 3 本と `tests/test-ralph-config.sh` 19/19。テンプレートの `max_seats` を変えると `org.max_seats` の行で、`auto_merge` の行を消すと `missing from the template` の行で落ちる。`run-test.sh` は打ち直しで rc 0。
  - merge のあと: `go vet ./internal/config/`、`go test ./internal/config/ -count=1`、`check-sync.sh`。
- 確かめていない:
  - merge のあとの全体の `run-test.sh` と `run-static-verify.sh`(別のセッションが同じ Mac で CI を回していたため、この PR の CI に任せた)。
  - 既定値が同じキーどうしの配線の取り違えと、比較の 1 行の削除は、コミットしたテストでは捕まらない。HEAD の配線は tester の使い捨ての 19 行の probe で確かめた。表のテストは #274 で足す。
