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
// these tests pin the director's rules. scripts/check-skill-sync.sh keeps
// the four copies of the skill identical but does not read Go.
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
//
// The last test pins the leader's rule for the codex hook trust note that
// `ralph org spawn` prints (plan D5); for it, update its fragments the same
// way or restore internal/org/prompts/leader.md.

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
// and end, and what <dir> is: the main worktree's root for the three ralph
// org verbs, the directory ralph status ran in for ralph status; the director
// tells the human once in the conversation with the line's three values,
// says that ralph upgrade rewrites tracked files so the human decides whether
// and how to run it, says in the same message that a ralph status line naming
// a linked worktree is about that worktree's copy and does no more, gives no
// example, and does not tell again for the same <dir> and versions in the
// same session (D1); it does not run ralph upgrade even when asked, for the
// reason of 実装しない (D2); after the human's upgrade it reads the copy of
// the skill at the <dir> of the line the human followed again by absolute
// path (asking the human which <dir> they ran ralph upgrade in first, when the
// lines it told about had different <dir> and the human did not say), follows
// that copy over a different one in its own working
// directory's worktree and says so, and a headless director also recommends
// the stop and start rebuild, whose stop it cannot run itself (D3); and the
// codex hook trust note, the verbs that print it, its end, the human told
// the line from ralph org start, and the leader's QUESTION carrying the
// spawn's line: the leader does not stop for it, so the director does not
// wait for the human's answer, tells the human the line and the leader's
// question (keep the seat or stop it), sends a DECISION saying it told the
// human, that the seat stays in use until the human answers, and that the
// answer comes in another DECISION, in the order of ### 答える順, and
// resolves the item, since the watch takes the leader as waiting and does
// not report its stall while the item is not resolved; when the org has
// already disbanded (no active seat in ralph org status), it sends no
// DECISION and resolves with a --note, as for a RESULT item; and the human's
// later answer goes to the leader as a DECISION (steps 1 and 2 of
// ### 答える順) while the org runs, and the human is told there is nobody to
// send it to once the org has disbanded (cross-review cycle 1, findings 1
// and 2); then no edit of the codex config and no approval of the hooks,
// which the human decides after reading them (D4).
var directorNoticeRules = []string{
	"ralph の動詞が stderr に出す次の 2 つの `note:` の行は、人に向けた知らせ",
	"どちらが出ても動詞はそのまま進むので、director も作業を止めない",
	"雛形の版の知らせ: `ralph status`、`ralph org start`、`ralph org director attach`、`ralph org director start` が、" +
		"雛形の `.ralph/manifest.toml` の版が CLI より古いときに出す",
	"`run 'ralph upgrade' in <dir>` で終わる",
	"`<dir>` は絶対パスで、`ralph org` の 3 つの動詞の行では main worktree のルート、" +
		"`ralph status` の行では status を打ったディレクトリ(linked worktree で打てば、その worktree)",
	"行を見たら、その会話(headless の director は pane か端末)で人に 1 度伝える",
	"伝える文には行の 3 つの値(雛形の版、CLI の版、`<dir>`)を入れ",
	"`ralph upgrade` は追跡ファイルを書き換えるので、流すかどうかと流し方は人が決めると添える",
	"`ralph status` の行の `<dir>` が linked worktree なら、古いのはその worktree の雛形の写しだということも同じ文で伝え、" +
		"それ以上のことはしない",
	"流し方の例は書かない",
	"同じセッションで、同じ `<dir>` と版の組の行をまた見ても、もう 1 度は伝えない",
	"`ralph upgrade` は打たない。人に頼まれても打たず、人が流すか、標準フロー(`/plan` → `/implement`)に任せるよう返す",
	"理由は「してはいけないこと」の「実装しない」と同じで、upgrade は repo の追跡ファイルを書き換える",
	"人が upgrade を済ませたと言ったら、人が従った行の `<dir>` の skill を絶対パスで読み直す",
	"(Claude Code は `<dir>/.claude/skills/director/SKILL.md`、Codex は `<dir>/.agents/skills/director/SKILL.md`)",
	"セッションが持っている skill の文は古い版のまま残るため",
	"人に伝えた行の `<dir>` が違っていて、人がどの `<dir>` を upgrade したかを言わなかったときは、" +
		"読み直す前に、どの `<dir>` で `ralph upgrade` を打ったかを人に尋ねる",
	"自分の作業ディレクトリが `<dir>` と別の worktree で、そこの skill が `<dir>` のものと違うときは、`<dir>` の方に従い、" +
		"自分の worktree の写しが古いままだと人に伝える",
	"headless の director は、加えて `ralph org director stop` と `ralph org director start` での立て直しを人に勧める",
	"立て直すと新しい prompt と skill で始まる。stop は director 自身では打てない",
	"codex の hook の承認の知らせ: `ralph org spawn`、`ralph org start`、`ralph org director start` が、" +
		"codex で動く座席か director の hook が承認されていないか切られているときに出す",
	"`...; run 'ralph doctor' for <the key(s)> and how to <直し方>` で終わる",
	"`ralph org start` が出した行を見たら、行をそのまま人に伝える",
	"その行は leader の QUESTION で届く",
	"leader はこの QUESTION で作業を止めないので、director も人の答えを待たない",
	"行と leader の問い(座席をこのまま使うか止めるか)を人に伝える",
	"続けて、人に伝えたこと、人が答えるまで座席はそのまま使うこと、答えは別の DECISION で届くことを書いた DECISION を" +
		"「答える順」のとおりに送り、件を resolve する",
	"件を resolve しないあいだ、見張りは leader が答えを待っているとみなし、leader が止まっても停滞を知らせないため",
	"その QUESTION を処理するときに org がもう disband していれば(`ralph org status --org-id <org>` に active な座席がない)、" +
		"DECISION は送らず、RESULT の件と同じく、`--note` に人に伝えたことを 1 行で書いて resolve する",
	"どちらのときも、あとで人が答えたら、org が走っていれば「答える順」の 1 と 2 のとおりに答えを DECISION で leader に送り、" +
		"disband していれば送る先がないことを人に伝える",
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
// ralph status, and the attach after it, whose line names another <dir> in a
// linked worktree) and step 3 of ### 機能ごとの出力先 (ralph org start) point
// to it; and ## してはいけないこと has the bullet that
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
			"stderr に雛形の版の `note:` の行が出たら(3 の attach も同じ形の行を出す。" +
				"linked worktree で動いていると、attach の行の `<dir>` は status の行と違う)、" + directorNoticePointer,
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
// skill at the <dir> of the line the human followed again by absolute path
// and recommend the rebuild with the stop of the prompt's 終わり and ralph
// org director start.
// It names no ralph org start: the prompt says ralph org only in commands
// that carry the director's flags or that it does not run
// (TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig).
var directorPromptNoticePhrases = []string{
	"`ralph status` か機能ごとの org の start が stderr に雛形の版か codex の hook の `note:` の行を出したら",
	"skill の「進捗の報告」の「" + strings.TrimPrefix(directorNoticeHeading, "### ") + "」のとおり人に伝える",
	"`ralph upgrade` は人に頼まれても打たず、codex の設定も書き換えない",
	"人が upgrade を済ませたと言ったら、人が従った行の `<dir>` の skill を絶対パスで読み直し",
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

// leaderHookNoteMarker is the start of the 運用規律 bullet of
// prompts/leader.md on the codex hook trust note `ralph org spawn` prints.
const leaderHookNoteMarker = "- `ralph org spawn` が stderr に、"

// leaderHookNotePhrases are the phrases of that bullet, in order (plan D5):
// the note, told by the end of the codex hook trust line, does not stop the
// spawn, and a spawn that exits 0 leaves the seat usable; the line comes
// before the seat is started, so a spawn that then fails with rejected: or
// spawn failed: raises no QUESTION and follows the failure rule the bullet
// names (the spawn failed: bullet right above; for rejected:, the watch
// bullet above or step 7 of 機能ごとの org, the approval check); the QUESTION
// is raised only for a spawn that exits 0 after printing the line; the leader
// edits no codex config and starts no codex to approve the hooks, which the
// human decides; it raises a QUESTION by the 件を上げる section's procedure,
// with the escalate file of the org and `ralph org escalate`, the line
// unchanged and the seat id in the body, and --state-dir as that section
// says, asking the human whether to keep using the seat or stop it after
// reading the hooks; the QUESTION does not stop work, the exception to the
// stop rule of 件を上げる, so the seat stays in use and gets its TASKs, and
// the human's DECISION says whether to stop it and is followed; the same
// seat's same line is not raised again before the reply, nor after the
// DECISION, and another seat's line gets a QUESTION of its own even before
// the reply, for the reason given.
var leaderHookNotePhrases = []string{
	"`ralph org spawn` が stderr に、`; run 'ralph doctor' for ` を含む `note:` の行(codex の hook の承認の知らせ)を出しても、spawn は止まりません",
	"spawn が終了コード 0 で終われば、座席はそのまま使えます",
	"行は座席を立てる前に出るので、行のあとで spawn が失敗することもあります",
	"stdout に `rejected:` か `spawn failed:` の行が出て spawn が失敗したときは、下の QUESTION は上げないでください",
	"`spawn failed:` のときは直上の項に従います",
	"`rejected:` のときは、横断の見張りが動いていないために断られたなら上の見張りの項に、" +
		"承認の検査で断られたなら「機能ごとの org」の節の手順 7 に従います",
	"QUESTION を上げるのは、その座席の spawn が終了コード 0 で終わり、行を出していたときです",
	"codex の設定(`$CODEX_HOME/config.toml`、既定は `~/.codex/config.toml`)を書き換えず、codex を起動して hook を承認することもしないでください",
	"承認するかは人が決めます",
	"代わりに QUESTION を「件を上げる」の節の手順で `.harness/state/escalate-org-a.txt` に書き",
	"(本文には、その行を変えずに書き、座席 id を添えます)",
	"`ralph org escalate` で上げてください",
	"(`--state-dir` の付け方もその節のとおりです)",
	"QUESTION では、その座席をこのまま使うか止めるかを、hook の中身を読んで決めるよう人に尋ねます",
	"この QUESTION では作業を止めません",
	"「件を上げる」の節の、上げたら返事が来るまで作業を止める決まりの例外です",
	"座席はそのまま使い、TASK もこれまでどおり送ってください",
	"座席を止めるかは人が決め、その答えは DECISION で届きます",
	"DECISION が届いたら、そのとおりにしてください",
	"返事が来る前に同じ座席の spawn が同じ行をまた出しても、もう 1 度は上げません",
	"DECISION が届いたあとに同じ座席の spawn が同じ行を出したときも上げません",
	"別の座席の spawn が行を出したら、返事の前でも、その座席とその行で QUESTION をもう 1 つ上げてください",
	"hook の状態は座席の作業ディレクトリと設定で違い、人は座席ごとの行を見て止める座席を選ぶためです",
}

// leaderHookNoteStopPointer is the stop rule of the 件を上げる section with
// the phrase that points at the 運用規律 bullet as the exception.
const leaderHookNoteStopPointer = "BLOCKED と QUESTION を上げたら、その件にかかわる作業は返事が来るまで止める" +
	"(`ralph org spawn` が出す codex の hook の `note:` の行を上げる QUESTION は例外で、作業を止めない。「運用規律」の節の項のとおり)"

// TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping pins AC5
// and the leader half of AC6 of the plan: the rendered leader prompt's
// 運用規律 section has one top-level bullet starting leaderHookNoteMarker,
// right after the spawn failed: bullet and its nested cases, with every
// phrase of leaderHookNotePhrases in order and the end of the codex hook
// trust line (directorNoticeCLIFragments) quoted on one line as it is, spaces
// included; and the stop rule of the 件を上げる section points at that bullet
// as its exception (leaderHookNoteStopPointer). See the comment at the top of
// this file for what to update.
func TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	discipline, ok := markdownSection(text, "## 運用規律")
	if !ok {
		t.Fatalf("rendered leader prompt has no 運用規律 section:\n%s", text)
	}
	bullet, ok := markdownItem(discipline, leaderHookNoteMarker)
	if !ok || !strings.HasPrefix(bullet, leaderHookNoteMarker) {
		t.Fatalf("運用規律 section has no top-level bullet starting %q:\n%s", leaderHookNoteMarker, discipline)
	}
	assertPhrasesInOrder(t, bullet, "the 運用規律 bullet on the codex hook trust note", leaderHookNotePhrases)
	hookNoteEnd := directorNoticeCLIFragments[1]
	if !strings.Contains(hookNoteEnd, "ralph doctor") {
		t.Fatalf("directorNoticeCLIFragments[1] = %q, want the end of the codex hook trust note", hookNoteEnd)
	}
	if !strings.Contains(bullet, "`"+hookNoteEnd+"`") {
		t.Errorf("the 運用規律 bullet does not quote %q, the end of the codex hook trust note, on one line as it is:\n%s", hookNoteEnd, bullet)
	}
	if n := strings.Count(discipline, leaderHookNoteMarker); n != 1 {
		t.Errorf("運用規律 section has %d bullets starting %q, want 1:\n%s", n, leaderHookNoteMarker, discipline)
	}

	// The bullet comes right after the spawn failed: bullet, whose nested
	// cases end where the next top-level bullet starts.
	const spawnFailedMarker = "stdout に `spawn failed:` の行が"
	start := strings.Index(discipline, spawnFailedMarker)
	if start < 0 {
		t.Fatalf("運用規律 section does not name %q:\n%s", spawnFailedMarker, discipline)
	}
	rest := discipline[start:]
	if next := strings.Index(rest, "\n- "); next < 0 || !strings.HasPrefix(rest[next+1:], leaderHookNoteMarker) {
		t.Errorf("expected the bullet starting %q right after the spawn failed: bullet, got section:\n%s", leaderHookNoteMarker, discipline)
	}

	escalate, ok := markdownSection(text, "## 件を上げる(`ralph org escalate`)")
	if !ok {
		t.Fatalf("rendered leader prompt has no 件を上げる section:\n%s", text)
	}
	if item, ok := markdownItem(escalate, "BLOCKED と QUESTION を上げたら"); !ok || !containsPhrase(item, leaderHookNoteStopPointer) {
		t.Errorf("expected the 件を上げる stop rule to say %q, got item:\n%s", leaderHookNoteStopPointer, item)
	}
}
