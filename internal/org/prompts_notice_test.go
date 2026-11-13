package org

// The tests in this file pin what the /director skill and the headless
// director's role prompt say about the two human-facing note: lines ralph's
// verbs print on stderr (issue #318; plan
// docs/plans/active/2026-11-13-director-notice-handling.md): the scaffold
// version note (scaffoldVersionNotice in internal/cli, issue #313), which
// ends with `run 'ralph upgrade' in <dir>`, and the codex hook trust note
// (codexHookTrustNotice in internal/cli, issue #310), which ends with
// `; run 'ralph doctor' for ...`. Neither verb refuses anything, so the
// skill's text is all that keeps a director from running ralph upgrade or
// approving the hooks itself, and from passing the line over without telling
// the human. internal/org cannot import internal/cli, so the test that ties
// the skill to the CLI's own wording belongs to internal/cli (plan AC7);
// these tests pin the director's rules. scripts/check-skill-sync.sh keeps the four copies of
// the skill identical but does not read Go.
//
// If a test here fails after a deliberate change to a rule, put the rule's
// new words, short and verbatim, in its fragment in the skill's (or the
// prompt's) order; otherwise restore the text in
// .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies (root, then with
// CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills), or restore
// internal/org/prompts/director.md.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// directorNoticeHeading is the /director skill's subsection on the two
// note: lines, under ## 進捗の報告 (plan D1-D4).
const directorNoticeHeading = "### 人に向けた知らせの行"

// directorNoticeCLIFragments are the parts of the two CLI lines the
// subsection quotes as they are, spaces included: the end of the scaffold
// version note and the end of the codex hook trust note. internal/cli's test
// checks them against the lines the CLI builds.
var directorNoticeCLIFragments = []string{
	"run 'ralph upgrade' in ",
	"; run 'ralph doctor' for ",
}

// directorNoticeRules are fragments of the subsection ### 人に向けた知らせの行,
// in order: the two lines are notes for the human and the verb and the work
// go on; the scaffold version note, the four verbs that print it, its shape
// and end, and what <dir> is; the director tells the human once in the
// conversation with the line's three values, says that ralph upgrade rewrites
// tracked files so the human decides whether and how to run it, gives no
// example, and does not tell again for the same <dir> and versions in the
// same session (D1); it does not run ralph upgrade even when asked, for the
// reason of 実装しない (D2); after the human's upgrade it reads <dir>'s copy
// of the skill again by absolute path, follows <dir>'s copy over a stale one
// in its own linked worktree and says so, and a headless director also
// recommends the stop and start rebuild, whose stop it cannot run itself
// (D3); and the codex hook trust note, the verbs that print it, its end, the
// human told the line from ralph org start, a leader's QUESTION carrying the
// spawn's line handled as any QUESTION, and no edit of the codex config and
// no approval of the hooks, which the human decides after reading them (D4).
var directorNoticeRules = []string{
	"ralph の動詞が stderr に出す次の 2 つの `note:` の行は、人に向けた知らせ",
	"どちらが出ても動詞はそのまま進むので、director も作業を止めない",
	"雛形の版の知らせ: `ralph status`、`ralph org start`、`ralph org director attach`、`ralph org director start` が、" +
		"雛形の `.ralph/manifest.toml` の版が CLI より古いときに出す",
	"`run 'ralph upgrade' in <dir>` で終わる",
	"`<dir>` は main worktree のルートの絶対パス",
	"行を見たら、その会話(headless の director は pane か端末)で人に 1 度伝える",
	"伝える文には行の 3 つの値(雛形の版、CLI の版、`<dir>`)を入れ",
	"`ralph upgrade` は追跡ファイルを書き換えるので、流すかどうかと流し方は人が決めると添える",
	"流し方の例は書かない",
	"同じセッションで、同じ `<dir>` と版の組の行をまた見ても、もう 1 度は伝えない",
	"`ralph upgrade` は打たない。人に頼まれても打たず、人が流すか、標準フロー(`/plan` → `/implement`)に任せるよう返す",
	"理由は「してはいけないこと」の「実装しない」と同じで、upgrade は repo の追跡ファイルを書き換える",
	"人が upgrade を済ませたと言ったら、`<dir>` の skill を絶対パスで読み直す",
	"(Claude Code は `<dir>/.claude/skills/director/SKILL.md`、Codex は `<dir>/.agents/skills/director/SKILL.md`)",
	"セッションが持っている skill の文は古い版のまま残るため",
	"linked worktree で動いていて、自分の作業ディレクトリの skill が `<dir>` のものと違うときは、`<dir>` の方に従い、" +
		"自分の worktree の写しが古いままだと人に伝える",
	"headless の director は、加えて `ralph org director stop` と `ralph org director start` での立て直しを人に勧める",
	"立て直すと新しい prompt と skill で始まる。stop は director 自身では打てない",
	"codex の hook の承認の知らせ: `ralph org spawn`、`ralph org start`、`ralph org director start` が、" +
		"codex で動く座席か director の hook が承認されていないか切られているときに出す",
	"`...; run 'ralph doctor' for <the key(s)> and how to <直し方>` で終わる",
	"`ralph org start` が出した行を見たら、行をそのまま人に伝える",
	"その行は leader の QUESTION で届く。ほかの QUESTION と同じく扱う",
	"codex の設定(`$CODEX_HOME/config.toml`、既定は `~/.codex/config.toml`)を書き換えない",
	"codex を起動して hook を承認することもしない",
	"承認するかは、hook の中身を読んで人が決める",
}

// directorNoticePointer is the sentence that sends a director from where a
// note: line first appears to the subsection.
const directorNoticePointer = "「進捗の報告」の「人に向けた知らせの行」に従う"

// TestDirectorSkill_SaysWhatToDoWithTheHumanNotes pins AC1, AC2, AC3, and
// the skill half of AC6 of the plan: each copy of the /director skill has the
// subsection directorNoticeHeading inside ## 進捗の報告, before
// ### merge と auto_merge, with every fragment of directorNoticeRules in
// order and the two CLI fragments as they are (directorNoticeCLIFragments),
// and with no fenced block (D1 gives no example of how to run ralph upgrade,
// since its run in a linked worktree was not checked); step 2 of ## 昇格 (the
// ralph status, and the attach after it) and step 3 of ### 機能ごとの出力先
// (ralph org start) point to it; and ## してはいけないこと has the bullet that
// forbids ralph upgrade and the codex config edit. Fragments are compared
// with spaces and line breaks removed (withoutSpaces), except the CLI
// fragments.
func TestDirectorSkill_SaysWhatToDoWithTheHumanNotes(t *testing.T) {
	checkSkillHeadingOrder(t, directorSkillCopies, []string{"## 進捗の報告", directorNoticeHeading, "### merge と auto_merge"})
	checkDirectorSkillPlaces(t, "human notes", []skillPlace{
		{"the subsection " + directorNoticeHeading, func(text string) string {
			return directorSkillSubsection(text, directorNoticeHeading)
		}, directorNoticeRules},
		{"step 2 of the section ## 昇格", func(text string) string {
			return directorSkillNumberedStep(directorSkillSection(text, "## 昇格"), 2)
		}, []string{
			"`ralph status` を打ち、`org watch:` の行を読む",
			"stderr に雛形の版の `note:` の行が出たら(3 の attach も同じ行を出す)、" + directorNoticePointer,
		}},
		{"step 3 of the subsection " + directorSplitOutputHeading, directorSplitStep(directorSplitOutputHeading, 3), []string{
			"ralph org start --plan .harness/state/org/splits/<id>.md --feature <slug>",
			"start が stderr に雛形の版か codex の hook の `note:` の行を出したら、" + directorNoticePointer,
		}},
		{"the ralph upgrade bullet of ## してはいけないこと", func(text string) string {
			return directorSkillBullet(directorSkillSection(text, "## してはいけないこと"), "- `ralph upgrade` を打たない。")
		}, []string{
			"`ralph upgrade` を打たない。雛形の版の `note:` の行が名指していても、人に頼まれても打たない",
			"codex の設定を書き換えず、codex を起動して hook を承認しない",
			"(「進捗の報告」の「人に向けた知らせの行」)",
		}},
	})

	root := directorSkillRepoRoot(t)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's human notes check (vendored repo?)", rel, err)
		}
		text := string(data)
		if !strings.Contains(directorSkillSection(text, "## 進捗の報告"), "\n"+directorNoticeHeading+"\n") {
			t.Errorf("%s: the section ## 進捗の報告 has no subsection %s -- see the comment at the top of this file", rel, directorNoticeHeading)
		}
		sub := directorSkillSubsection(text, directorNoticeHeading)
		for _, fragment := range directorNoticeCLIFragments {
			if !strings.Contains(sub, fragment) {
				t.Errorf("%s: the subsection %s does not quote %q, part of a CLI note line, on one line as it is -- see the comment at the top of this file", rel, directorNoticeHeading, fragment)
			}
		}
		if strings.Contains(sub, "```") {
			t.Errorf("%s: the subsection %s holds a fenced block; it gives no example of how to run ralph upgrade (plan D1) -- see the comment at the top of this file", rel, directorNoticeHeading)
		}
	}
}

// directorPromptNoticeMarker is the start of the ミッション bullet of
// prompts/director.md on the two note: lines.
const directorPromptNoticeMarker = "- `ralph status` か機能ごとの org の start が"

// directorPromptNoticePhrases are the phrases of that bullet, in order: when
// ralph status or a feature org's start prints one of the two lines, tell the
// human by the skill's subsection; do not run ralph upgrade, even when asked,
// and do not edit the codex config; after the human's upgrade, read the
// skill at the line's <dir> again by absolute path and recommend the rebuild
// with the stop of the prompt's 終わり and ralph org director start. It names
// no ralph org start: the prompt says ralph org only in commands that carry
// the director's flags or that it does not run
// (TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig).
var directorPromptNoticePhrases = []string{
	"`ralph status` か機能ごとの org の start が stderr に雛形の版か codex の hook の `note:` の行を出したら",
	"skill の「進捗の報告」の「" + strings.TrimPrefix(directorNoticeHeading, "### ") + "」のとおり人に伝える",
	"`ralph upgrade` は人に頼まれても打たず、codex の設定も書き換えない",
	"人が upgrade を済ませたと言ったら、行の `<dir>` の skill を絶対パスで読み直し",
	"「終わり」の stop と `ralph org director start` での立て直しを人に勧める",
	"(立て直すと新しい prompt と skill で始まる)",
}

// TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection pins AC4 and the
// director half of AC6 of the plan: for a director in herdr and for one in
// the terminal launch_cmd opened, the ミッション section has one bullet with
// every phrase of directorPromptNoticePhrases in order, naming the skill's
// subsection by its heading (directorNoticeHeading), and the 終わり section
// it points to holds the stop the human runs. See the comment at the top of
// this file for what to update.
func TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection(t *testing.T) {
	for _, launch := range []string{DirectorLaunchHerdr, DirectorLaunchCmd} {
		t.Run(launch, func(t *testing.T) {
			vars := testDirectorPromptVars()
			vars.Launch = launch
			text := mustRenderDirectorPrompt(t, vars)
			mission, ok := markdownSection(text, "## ミッション")
			if !ok {
				t.Fatalf("rendered director prompt has no ミッション section:\n%s", text)
			}
			bullet, ok := markdownItem(mission, directorPromptNoticeMarker)
			if !ok {
				t.Fatalf("ミッション section has no bullet starting %q:\n%s", directorPromptNoticeMarker, mission)
			}
			assertPhrasesInOrder(t, bullet, "the ミッション bullet on the note: lines", directorPromptNoticePhrases)
			if n := strings.Count(mission, directorPromptNoticeMarker); n != 1 {
				t.Errorf("ミッション section has %d bullets starting %q, want 1:\n%s", n, directorPromptNoticeMarker, mission)
			}
			end, ok := markdownSection(text, "## 終わり")
			if !ok {
				t.Fatalf("rendered director prompt has no 終わり section:\n%s", text)
			}
			if stop := "ralph org director stop --id d3 --state-dir " + testDirectorStateDirArg; !containsPhrase(end, stop) {
				t.Errorf("終わり section, which the bullet points to for the stop, does not hold %q:\n%s", stop, end)
			}
		})
	}
}
