package org

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The tests in this file pin what the /director and /org skills say about a
// split plan's shared paths (plan 2026-11-22-split-shared-paths, AC6): how
// the director writes the header's `- Shared:` line and what it changes
// (### 共有のパス of the /director skill), and how a conflict on a shared
// path between feature orgs running side by side is resolved
// (### 共有のパスの直し方 of the /org skill, which the leader template names
// by those words), with the /org skill's format, reservation, and reseat
// sentences. The examples of the line in both skills must also parse as
// split.go reads them.
//
// If one of these tests fails after a deliberate change to a rule, put the
// rule's new words, short and verbatim, in its fragment in the skill's
// order; otherwise restore the text in .claude/skills/<director or
// org>/SKILL.md and templates/base/.claude/skills/<director or
// org>/SKILL.md and run scripts/sync-skills.sh for the two .agents copies,
// as for the first test in director_skill_sync_test.go.

// The subsections these tests read: the /director skill's, after
// ### 起票と記録 and before ## 受信箱の処理, and the /org skill's, after
// ### org の終わり方 and before ### 座席の立て直し.
const (
	directorSharedPathsHeading = "### 共有のパス"
	orgSharedPathsFixHeading   = "### 共有のパスの直し方"
)

// orgSkillBetween returns the part of text from the line from up to the line
// to, both whole lines, or "" when text has no such pair in that order. It
// reads a subsection whose code block holds lines that start with "#", where
// directorSkillSubsection would stop.
func orgSkillBetween(text, from, to string) string {
	_, rest, ok := strings.Cut(text, "\n"+from+"\n")
	if !ok {
		return ""
	}
	section, _, ok := strings.Cut(rest, "\n"+to+"\n")
	if !ok {
		return ""
	}
	return section
}

// TestDirectorSkill_SharedPathsSaysWhereToWriteTheLineAndWhatItChanges pins
// the /director skill's ### 共有のパス: the line goes in the header, once,
// with the reservation rules, and is refused in the preamble or a feature
// section (an example goes in a code block); only files every feature has to
// change go there, never `.`, and never a path a feature reserves; the watch
// raises no scope_change for them, the seats' --scope (reseat's included)
// holds them, the reservation overlap rule stays, and a start whose shared
// paths overlap another running org's reservation, or whose reservation
// overlaps a running feature org's shared paths, is refused whichever
// started first; a change to the line
// after the approval changes every digest, so the plan is approved again and
// a stopped feature org is refused at a restart; and the fix of a conflict is
// the /org skill's. The question before the split plan and step 1 of
// ### 機能ごとの出力先 name the line too, and step 1 says that the plan file's
// line may differ from what a running org recorded at its start, and that
// start's refusal then decides.
func TestDirectorSkill_SharedPathsSaysWhereToWriteTheLineAndWhatItChanges(t *testing.T) {
	checkSkillHeadingOrder(t, directorSkillCopies, []string{directorSplitFilingHeading, directorSharedPathsHeading, "## 受信箱の処理"})
	checkDirectorSkillPlaces(t, "shared paths", []skillPlace{
		{"the subsection " + directorSharedPathsHeading, directorSplitSubsection(directorSharedPathsHeading), []string{
			"分割計画の先頭に `- Shared:` の行を書くと、そのパスはどの機能の org も変えてよい共有のパスになる",
			"- Shared: pnpm-lock.yaml, tsconfig.json",
			"書く場所: 先頭(最初の `## ` の見出しより前)に 1 行だけ",
			"前文の節か機能の節など、先頭より後の行頭に書いた `- Shared:` は、読み込みが断る",
			"本文に例として引くなら、コードブロックに入れる(コードブロックの中のものは何も共有しない)",
			"書くパス: どの機能も変えざるをえないファイルだけにする",
			"`.`(repo 全体)は書けない",
			"どの機能の `- Reserve:` とも重ねない",
			"見張りは、機能の org の共有のパスへの変更に scope_change を立てない",
			"座席の spawn に付ける `--scope` に予約のパスと共有のパスを合わせて入れる(`ralph org reseat` が立て直す座席の `--scope` も同じ)",
			"予約どうしの重なりの判定は変わらず",
			"共有のパスが走っている別の org の予約と重なる start と、予約が走っている機能の org の共有のパスと重なる start は、" +
				"どちらが先に立ったかによらず断られる",
			"`- Shared:` の行を変える(足す、消すも)と、分割計画の digest もすべての機能の digest も変わる。もう一度承認を聞く",
			"共有のパスが変わった機能として、立て直しの start にも reseat にも断られる",
			"直し方は `/org` skill の「共有のパスの直し方」",
		}},
		{"the subsection " + directorSplitQuestionsHeading, directorSplitSubsection(directorSplitQuestionsHeading), []string{
			"どの機能も変えてよい共有のパス(分割計画の先頭の `- Shared:` の行。下の「共有のパス」)",
		}},
		{"step 1 of the subsection " + directorSplitOutputHeading, directorSplitStep(directorSplitOutputHeading, 1), []string{
			"`- Shared:` の行があれば、その共有のパスも同じ規則で各 org の予約と比べ",
			"走っている機能の org の共有のパスは `reserved:` の行に出ないので",
			"承認し直して `- Shared:` を変えたあとは、その org が start のときに記録した値と違うことがある",
			"ここで数えた重なりと start の答えが違えば、start の断りに従う",
		}},
	})
}

// TestOrgSkill_SharedPathsFixSaysTheLaterMergeRegeneratesGeneratedFiles pins
// the /org skill's ### 共有のパスの直し方, which the leader template names
// (prompts/leader.md, 機能ごとの org): the side that merges later takes the
// default branch the usual way (a rebase through the implementer in a running
// org, a merge in the feature worktree for a disbanded org's branch),
// regenerates a generated file from the default branch's version instead of
// merging it by hand, keeps both sides of a hand-written shared file,
// commits, and runs the review and the gate again (a regate for a disbanded
// org's branch). It also pins the /org skill's sentences on the line itself
// (a `- Shared:` line after the header refused, in ### 分割計画 and in the
// refusals of ### 承認), the reservation, the rebase step that points at the
// fix, and reseat's --scope.
func TestOrgSkill_SharedPathsFixSaysTheLaterMergeRegeneratesGeneratedFiles(t *testing.T) {
	if leader := renderSeatPrompt(t, "leader"); !strings.Contains(leader, "共有のパスの直し方") {
		t.Fatalf("expected the leader template to name the /org skill's 共有のパスの直し方, got:\n%s", leader)
	}
	checkSkillHeadingOrder(t, orgSkillCopies, []string{"### org の終わり方", orgSharedPathsFixHeading, "### 座席の立て直し"})
	checkOrgSkillPlaces(t, "shared paths", []skillPlace{
		{"the subsection " + orgSharedPathsFixHeading, directorSplitSubsection(orgSharedPathsFixHeading), []string{
			"直すのは、あとから merge する側で、既定のブランチの取り込み方はこれまでと変えない",
			"leader が implementer に rebase と衝突の解消を TASK で頼む(「org の終わり方」の 1)",
			"機能の worktree で `git merge origin/<既定のブランチ>` を打って直す",
			"生成されるファイル(lockfile、`go.sum` など)は手で merge しない",
			// The cross-review finding of the branch (docs/reports/cross-review-triage-split-shared-paths.md,
			// ACTION_REQUIRED 1): a version the feature raised in the lockfile only would go back silently
			// on the regeneration, so the step has it noted first and applied again after.
			"この機能が生成されるファイルだけで上げた版を控える",
			"`git diff \"$(git merge-base origin/<既定のブランチ> <先頭>)\" <先頭> -- <パス>`",
			"生成されるファイルは既定のブランチの版に戻し(`git checkout origin/<既定のブランチ> -- <パス>`)、生成するコマンドで作り直す",
			"`pnpm-lock.yaml` なら `pnpm install`、`go.sum` なら `go mod tidy`",
			"控えた版は黙って戻る(テストは通ったままになる)",
			"同じ更新のコマンド(`pnpm update <パッケージ>@<版>` など)を打ち直し、作り直したファイルにその版があることを確かめる",
			"控える版を読み切れないとき、コマンドが動かない(ネットワークのない sandbox など)ときも手で merge せず",
			"手で書く共有のファイル(root の `tsconfig.json` の `references`、`go.mod` の `require` など)は、両側の変更を残して merge する",
			"「org の終わり方」の 1 から gate までを進め直す",
			"disband 済みの org のブランチでは、push してから regate を打ち直す",
		}},
		{"the subsection ### 分割計画", func(text string) string { return orgSkillBetween(text, "### 分割計画", "### 承認") }, []string{
			"先頭には、任意で `- Shared:` の行を 1 つ置ける",
			"`.`(repo 全体)、2 行目の `- Shared:`、どれかの機能の `- Reserve:` と重なるパスは、読み込みで拒否する",
			"`- Shared:` の行として読むのは先頭のものだけ",
			"先頭より後の行頭に書いた `- Shared:` は、読み込みで拒否する",
		}},
		{"the subsection ### 承認", func(text string) string { return orgSkillBetween(text, "### 承認", "### `ralph org start --plan`") }, []string{
			"先頭より後の行頭の(コードブロックの外の)`- Shared:`、のどれでも読み込みは拒否する",
		}},
		{"the subsection ### 担当範囲の予約(`--reserve`)", directorSplitSubsection("### 担当範囲の予約(`--reserve`)"), []string{
			"分割計画の共有のパス(「分割計画」の `- Shared:`)は予約のパスに数えない",
			"どちらが先に立ったかによらず拒否され",
		}},
		{"step 1 of the subsection ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 1) }, []string{
			"衝突したのがタスクの `- 共有のパス:` の行のパスなら、TASK の本文に、下の「共有のパスの直し方」に従うよう書く",
		}},
		{"step 4 of the subsection ### 座席の立て直し", directorSplitStep("### 座席の立て直し", 4), []string{
			"leader 以外の座席の `--scope` は org の予約のパスで、結びつきに分割計画の共有のパスがあれば、そのあとに共有のパスが続く",
		}},
	})
}

// TestSkills_SharedLineExamplesParse reads the example of the header's
// `- Shared:` line in the /director skill's ### 共有のパス with
// parseSplitShared, and the /org skill's example split plan (the markdown
// code block of ### 分割計画) with parseSplitPlan, so that neither skill
// shows a line split.go refuses.
func TestSkills_SharedLineExamplesParse(t *testing.T) {
	root := directorSkillRepoRoot(t)
	read := func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the shared line example check (vendored repo?)", rel, err)
		}
		return string(data)
	}

	for _, rel := range directorSkillCopies {
		section := directorSkillSubsection(read(rel), directorSharedPathsHeading)
		var line string
		for l := range strings.SplitSeq(section, "\n") {
			if strings.HasPrefix(l, splitSharedPrefix) {
				line = l
				break
			}
		}
		if line == "" {
			t.Errorf("%s: %s has no line starting with %q", rel, directorSharedPathsHeading, splitSharedPrefix)
			continue
		}
		got, err := parseSplitShared(strings.TrimPrefix(line, splitSharedPrefix))
		if want := []string{"pnpm-lock.yaml", "tsconfig.json"}; err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: parseSplitShared(%q) = %q, %v, want %q, nil", rel, line, got, err, want)
		}
	}

	for _, rel := range orgSkillCopies {
		section := orgSkillBetween(read(rel), "### 分割計画", "### 承認")
		_, block, ok := strings.Cut(section, "```markdown\n")
		if ok {
			block, _, ok = strings.Cut(block, "\n```\n")
		}
		if !ok {
			t.Errorf("%s: ### 分割計画 has no closed ```markdown code block", rel)
			continue
		}
		plan, err := parseSplitPlan([]byte(block))
		if err != nil {
			t.Errorf("%s: the example split plan of ### 分割計画 does not parse: %v", rel, err)
			continue
		}
		if want := []string{"go.mod", "go.sum"}; !slices.Equal(plan.Shared, want) {
			t.Errorf("%s: the example split plan's shared paths = %q, want %q", rel, plan.Shared, want)
		}
	}
}
