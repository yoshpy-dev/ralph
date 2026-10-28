# Walkthrough: codex-verified-auto-merge-on(codex_verified と auto_merge の既定を true にする)

- Date: 2026-10-29
- Plan: docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md(/pr で archive に移す)
- Issue: #274
- PR: #283
- Branch: feat/codex-verified-auto-merge-on(この walkthrough の前の HEAD d9359707。origin/main 09dc0260 から 42 files、+1499/-344。計画、report、insight event を除くと 35 files、+758/-344)
- Base: ae64da70(main、「Merge pull request #281」)。途中で #282 の merge 09dc0260 を aa4ec8d3 で取り込んだ(衝突なし)
- 読む範囲: 実行コードは `internal/config/config.go` の既定値 2 行、`internal/cli/status.go` の 2 つの fallback、`internal/cli/org.go` の `resolveOrgConfig`。残りのコードはテストとコメント。文書は skill の 4 つの写し、recipe の 2 つの写し、README、spec、tech-debt の 3 行
- 後工程は 2 cycle。cycle 1 の cross-review の ACTION_REQUIRED(F2)で、/self-review から全段を回し直した。記録だけの修正の例外は使っていない

```
何を変えるか → 照合の表のテスト(S1) → 既定を true に(S2) → Go のコメント(S3) → 文書(S4) → 読めない ralph.toml の 2 つの道(F1、F2) → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで: `codex_verified` と `auto_merge` は既定 false。codex の座席は guarded だけ、director は merge を人に任せる。
- この PR から:
  - キーを書かない `ralph.toml` と `ralph.toml` のない project では、どちらも true。`ralph init` も true を書く。
  - false を書けば前の動き。既存の project の `ralph.toml` は seed なので変わらないが、行のない設定は true になる(v4.0.0 より前の `ralph init` は 2 行とも、v4.0.0〜v5.x は `auto_merge` を書いていない。tag の template で確かめた)。
  - `ralph.toml` があるのに読めないとき、`ralph org` の動詞はエラーにし、`ralph status` は auto_merge を false と出す。
- release のノートは PR の本文の「Release notes」の枠。changelog に載るのは `feat:` の cb8d258e と `fix:` の 958d9bff・aea6b494 だけで、文書の更新は `docs:` なので載らない。
- 戻し方: コードはこの PR を revert する。新しい `ralph init` が書いた true は revert では戻らないので、2 行を false にし、codex の座席を spawn し直す(PR の本文の「元に戻すには」)。

## 2. 照合の表のテスト(S1、29fd179e)

- `internal/config/defaults_sync_test.go:404` `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`: 照合する全キーの表。行ごとに (i) `Default()` のコピーの 1 field、(ii) テンプレートの文字列のそのキーの値、(iii) テンプレートの文字列のそのキーの行の削除を当て、そのキーを名指しする 1 行だけが返ることを確かめる。bool の行のテンプレート側の値は `!Default()` から作るので、S2 のあとも書き換えずに通る。
- `:501` `tomlKeys`: `orgTemplateProbe` の toml のタグを reflect でたどり、点つきの名前を作る。表のキーの集合と比べ、数をコメントに頼らない(#281 の Known gaps (a))。
- `:526` `editTemplateKey`: 表の見出しから次の見出しまでの中で、`key =` の行を 1 つだけ書き換えるか消す。複数行の配列(`model_pool`)も扱い、1 つでなければ error にする。
- 見ること: #273 のテストは、既定値が同じキーの取り違え(`max_orgs` と `deadman_minutes` の 10、この PR のあとは `codex_verified` と `auto_merge` の true など)を見逃していた(#281 の Known gaps (b))。/test の変異で、それを落とすのはこのテストだけだと確かめた。

## 3. 既定を true に(S2、cb8d258e と ce6e972d)

- `internal/config/config.go:566` と `:576`: `CodexVerified: true`、`AutoMerge: true`。field のコメントも直した。
- `templates/base/ralph.toml:97` と `:163`: 2 行を true に。直前のコメントは「true が既定、false にすると codex は guarded に限られる / director は merge を人に任せる」。agmsg の DB が writable root にないと RESULT を送れない、という注意は残した。
- `internal/config/config_test.go:123` `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(ralph.toml なし、[org] なし、キーなし)と `:160` `TestLoad_TemplateWithBothKeysFalse`(テンプレートの 2 行を false にすると 2 つとも false、ほかの値はテンプレートのまま)。
- `internal/cli` のテスト 9 本は計画の D2 で直した。false の動きのテストは false を明示して残した(`writeCodexOrgConfig` の `codex_verified = false`、status と director status の `auto_merge = false` の場合など)。既定を読むテストは期待を true にした(共有の文言の定数、ralph.toml のない wake・wait・check-line・merge のテスト)。main worktree の ralph.toml が優先されることを確かめるテストは、「worktree が false と書く → true」に組み替えた。
- 見ること: `internal/org` のテストは手組みの `testOrgConfig()` を使い、`Default()` の 2 つの値を読まないので壊れない。F1 で `CodexVerified: false` を明示した。

## 4. Go のコメント(S3、920a4dcb)

- `internal/org/permissions.go`: fail-closed は `codex_verified` が false のときだけ。既定が true なのは #274 の人の決定で、2026-09-18 の 1 台での検証は flag の対応の証拠(F1 の L1 で分けて書いた)。
- `internal/cli/doctor_shell_alias.go`: 既定の project では codex の座席が edits と autonomous でも起動する。doctor の detail の文は F1 の L3 で「only while … codex_verified is true, the default」にした。
- `internal/cli/org.go`: `withMainWorktreeDirectorEnabled` の「(true and false)」を「(both true)」に。a91f6694 で、`ralph status` の fallback の説明も直した。
- 実行時のエラー文(`permissions.go` の「requires … codex_verified=true; only guarded is allowed until then …」)は変えていない(PR の Notes の D-4)。

## 5. 文書(S4、f99ac82d、sync-docs e3363a99・d30e5940)

- `/org` skill(`.claude/skills/org/SKILL.md` と写し 3 つ): codex の座席の段落、director の行、recipe の手順の段落。
- `/director` skill: 「merge と auto_merge」の節で「(既定)」を `auto_merge = true` に移しただけ。「2 つの値が違えば false として扱う」規則と、`TestDirectorSkill_PinsTheAutoMergeSteps` が固定する文は変えていない。
- `docs/recipes/codex-seat-permissions.md` と写し: 題を「opt out if it fails」に。既定は true、手順で確かめて通らなければ false。手順の中身は変えていない。
- `docs/recipes/codex-setup.md` と写し、`README.md:124`。
- `docs/specs/2026-10-07-org-multi-org-director.md`: FR-8 に「2026-10-28 に決めたこと」、受け入れ条件と Security に印。旧文は履歴として残した。
- `docs/tech-debt/README.md`: 136(doctor の文の引用と「already set」)、144(b)、148(`resolveOrgConfig` を変えた跡)。

## 6. 読めない ralph.toml の 2 つの道(F1 958d9bff、F2 aea6b494)

- F1(self-review の M1): `internal/cli/status.go:262` `statusOrgConfig` と `:294` `statusDirectorConfig` は、`ralph.toml` が読めないとき `Default()` で director の行を出していた。既定が true になると、false と書いた file が壊れていても「聞かずに merge する」と出る。2 つの fallback で `AutoMerge` だけ false にし、warning が「not the default, because … could not be read」と理由を言う。
- F2(cross-review cycle 1): `internal/cli/org.go:613` `resolveOrgConfig` は `./ralph.toml` を `os.Stat` で見て、どのエラーでも「ない」として既定を返していた。権限のないディレクトリへの symlink の `ralph.toml` に false と書いてあっても true になり、F1 の fallback にも届かなかった。いまは `config.Load` に任せ、`fs.ErrNotExist` だけを既定にする。
- テスト: `internal/cli/org_test.go:1133` `TestResolveOrgConfig_UnreadableRalphTomlIsAnErrorNotTheDefaults` と、`status_test.go` の fallback のテストの「読めない」の軸(./ralph.toml と main worktree の 2 か所、テキストと `--json`)。root では skip する。
- 見ること: 行き先のない symlink は、今までどおり「ない」(既定)。`--config` で名指したファイルがないときも既定(PR より前からの契約、PR の Notes の D-7)。

## 7. 確かめたことと確かめていないこと

- 確かめた:
  - /verify cycle 2: `run-static-verify.sh` rc 0(golangci-lint 0 件)、check-sync、check-skill-sync、check-template-purity。AC1〜AC3、AC5〜AC7。
  - /test cycle 2: `run-test.sh` rc 0(Go 9 パッケージ、shell 45 本)。`go test -overlay` の変異は cycle 1 で 17 個、cycle 2 で 7 個、すべて落ちた。
  - cross-review cycle 2: 指摘 0。
- 確かめていない:
  - `-race` と Linux(Linux はこの PR の CI で見る)。
  - root で走るときの読めない ralph.toml のテスト(skip する)。
  - 本物の codex の座席の spawn と、director の本物の merge。この repo にはルートの `ralph.toml` がないので、この PR のあとは、この repo の `ralph org` も新しい既定で動く。
