package org

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The words the /org skill tests below find the sync-docs paragraphs at the
// end of item 1 of ### org の終わり方 by, and the first line of item 2, which
// stays where it was (plan 2026-10-30-leader-sync-docs, S2). The follow-up
// heading, the skip line, and the DOC_FOLLOWUP key are the leader template's
// (prompts_leader_syncdocs_test.go).
const (
	orgSkillSyncDocsHandOff   = "rebase が済んだら(要らなかったときも)、leader は sync-docs を doc-maintainer のサブエージェントに任せ、文書は自分では直さない"
	orgSkillSyncDocsCheck     = "doc-maintainer が終わったら、leader は `git status --porcelain` で変わったパスを読む"
	orgSkillEndStep2FirstLine = "2. `ralph org report --org-id <org_id>` を打ち(report は worktree の"
)

// orgSkillPlace is one place in the /org skill and the fragments it must
// have there, in order.
type orgSkillPlace struct {
	name      string
	section   func(text string) string
	fragments []string
}

// orgSkillEndOfOrgStep returns item n of the numbered list of the /org
// skill's subsection ### org の終わり方 (directorSkillNumberedStep).
func orgSkillEndOfOrgStep(text string, n int) string {
	return directorSkillNumberedStep(directorSkillSubsection(text, "### org の終わり方"), n)
}

// orgSkillEndOfOrgBullet returns the bullet of item 4 of ### org の終わり方
// whose first line holds marker (markdownItem), "" when there is none.
func orgSkillEndOfOrgBullet(text, marker string) string {
	item, _ := markdownItem(orgSkillEndOfOrgStep(text, 4), marker)
	return item
}

// checkOrgSkillPlaces fails t for each fragment of places that a copy of the
// /org skill (orgSkillCopies) does not have where it says, after the fragment
// before it, compared with spaces and line breaks removed
// (skillOrderedFragmentProblems). It skips t when a copy is missing (a repo
// that vendors internal/org without the skills tree).
func checkOrgSkillPlaces(t *testing.T, what string, places []orgSkillPlace) {
	t.Helper()
	root := directorSkillRepoRoot(t)
	for _, rel := range orgSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's %s check (vendored repo?)", rel, err, what)
		}
		text := string(data)
		for _, p := range places {
			for _, problem := range skillOrderedFragmentProblems(p.section(text), p.name, p.fragments) {
				t.Errorf("%s: %s -- see this test's doc comment for what to update", rel, problem)
			}
		}
	}
}

// TestOrgSkill_FanOutLetsTheFeatureLeaderHandSyncDocsToTheDocMaintainer pins
// AC3 of plan 2026-10-30-leader-sync-docs for ### 座席内 fan-out: the seats'
// fan-out stays as it was, and the leader may also hand work to its driver's
// subagent, sync-docs to the doc-maintainer in a feature org (Task on Claude
// Code, the custom agent on Codex), under the same rules: not a seat, not in
// the manifest, not counted in max_seats, and it sends nothing; the leader
// commits.
//
// If this test, or another in this file, fails after a deliberate change to
// a rule, put the rule's new words, short and verbatim, in its fragment in
// the skill's order; otherwise restore the text in .claude/skills/org/SKILL.md
// and templates/base/.claude/skills/org/SKILL.md and run scripts/sync-skills.sh
// for the two .agents copies, as for the first test in
// org_skill_send_sync_test.go.
func TestOrgSkill_FanOutLetsTheFeatureLeaderHandSyncDocsToTheDocMaintainer(t *testing.T) {
	checkOrgSkillPlaces(t, "fan-out", []orgSkillPlace{
		{"the subsection ### 座席内 fan-out", func(text string) string { return directorSkillSubsection(text, "### 座席内 fan-out") }, []string{
			"各座席(implementer / reviewer)は自分のドライバのサブエージェント(Claude Code: `Task`、Codex: `.codex/agents/`)へ作業を分割してよい",
			"子サブエージェントは座席の内部実装であり、org の座席ではない(manifest に現れず、`max_seats` に数えず、`leader` や他座席へ送信しない)",
			"例: implementer はフロント/バック/インフラ or モジュール単位で実装を分担する",
			"leader も自分のドライバのサブエージェントに作業を任せてよい",
			"機能ごとの org では、leader は sync-docs を doc-maintainer の子サブエージェントに任せる" +
				"(Claude Code: `Task(subagent_type=\"doc-maintainer\")`、Codex: `.codex/agents/doc-maintainer.toml`。「org の終わり方」の 1)",
			"この子にも上の決まりが当てはまる",
			"org の座席でないので manifest に現れず、`max_seats` に数えず、`ralph org` のコマンドを打たないので誰にも送信しない",
			"文書を直して report を書くだけで、コミットは leader がする",
		}},
	})
}

// TestOrgSkill_EndOfOrgStep1HandsSyncDocsToTheDocMaintainerAfterTheRebase
// pins AC3 of plan 2026-10-30-leader-sync-docs (D1, D2b, D5) for ### org
// の終わり方, whose items 1-7 follow the leader template's steps 9-16: item 1
// ends, after the fetch and the rebase, with sync-docs handed to the
// doc-maintainer subagent, skipped with SYNC_DOCS: skipped kept for item 6
// when there is no definition or the call fails, and the hand-off names the
// plan, the base, the reserved paths and how to read them, the edits inside
// the reservation only, the all-or-none rule for a set of copies, the copy
// checks, the report and its follow-up section, and what the doc-maintainer
// must not do. The sync-docs paragraphs come before item 2, whose first line
// is unchanged, so the items are not renumbered.
func TestOrgSkill_EndOfOrgStep1HandsSyncDocsToTheDocMaintainerAfterTheRebase(t *testing.T) {
	checkOrgSkillPlaces(t, "sync-docs hand-off", []orgSkillPlace{
		{"item 1 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 1) }, []string{
			"`git fetch origin <既定のブランチ>` を打ち",
			"`git rebase origin/<既定のブランチ>` を打つ",
			"fetch が失敗したら(ネットワークのない sandbox など)、打ち直さずに `BLOCKED` を escalate する",
			orgSkillSyncDocsHandOff,
			"(leader のコンテキストを使わないため。「座席内 fan-out」)",
			"Claude Code では `Task(subagent_type=\"doc-maintainer\")` で、Codex では custom agent の `.codex/agents/doc-maintainer.toml` で呼ぶ",
			"その定義が worktree にない(Claude Code では `.claude/agents/doc-maintainer.md`、Codex では `.codex/agents/doc-maintainer.toml`)か、" +
				"呼び出しが失敗したら(Codex で custom agent を呼べない、など)、sync-docs を飛ばし、" + syncDocsSkipLine + " の行を 6 のために控えて 2 に進む",
			"doc-maintainer への指示には、機能の計画のパス、比べる base の `origin/<既定のブランチ>`、予約したパス(タスクの `- 予約したパス:` の行)と" +
				"その読み方(ralph の予約と同じく、ファイルはそのパス、`/` で終わるディレクトリはその下、`.` は repo 全体)を書き、次のことを頼む",
			"直すのは予約の中のファイルだけ",
			"repo の検査が中身をそろえる写しの組(skill の写しなど)は、組のすべてが予約の中にあるときだけ直し、1 つでも予約の外にあれば組のどれも直さずに follow-up に書く",
			"直したあと、写しの検査(`scripts/check-skill-sync.sh`・`scripts/check-sync.sh`)があれば打ち、落ちる自分の変更を戻して follow-up に回す",
			"report を `docs/reports/sync-docs-<YYYY-MM-DD>-<機能の slug>.md` に書き",
			"予約の外で見つけた文書のずれを、その `" + syncDocsFollowupHeading + "` の節に 1 件 1 行(パスと、何が古いか)で書く(なければ `なし` の 1 行)",
			"insight event を付けず、コミットせず、機能の計画を変えず、`ralph org` のコマンドを打たない",
			orgSkillSyncDocsCheck,
		}},
		{"the subsection ### org の終わり方", func(text string) string { return directorSkillSubsection(text, "### org の終わり方") }, []string{
			"1. reviewer のレポート(reviewer が RESULT や BLOCKED に書いた、`docs/reports/` の下のファイル)をコミットし",
			orgSkillSyncDocsHandOff,
			orgSkillSyncDocsCheck,
			orgSkillEndStep2FirstLine,
			"3. `scripts/secret-scan-branch.sh` があれば `--strict` を付けて打ち",
			"4. reviewer に gate を頼む TASK を送る",
		}},
		{"the first line of item 2 of ### org の終わり方", func(text string) string {
			first, _, _ := strings.Cut(orgSkillEndOfOrgStep(text, 2), "\n")
			return first
		}, []string{orgSkillEndStep2FirstLine}},
	})
}

// TestOrgSkill_EndOfOrgStep1UndoesEveryChangeOrNoneAndCommitsOnce pins AC8 of
// plan 2026-10-30-leader-sync-docs (D2) for item 1 of ### org の終わり方:
// after the doc-maintainer, the leader reads git status --porcelain, undoes
// every sync-docs change but the report when a path outside the reservation
// or the feature plan changed, never only the outside half (a set of copies
// half changed fails the gate's copy check, which the implementer cannot fix
// outside the reservation), keeps the undone changes as follow-ups for item
// 6, and commits what is left with the report as one commit, the report alone
// when nothing else changed, and nothing when nothing changed. The plan is
// undone because of the approval check that ### 機能の計画の承認 describes.
func TestOrgSkill_EndOfOrgStep1UndoesEveryChangeOrNoneAndCommitsOnce(t *testing.T) {
	checkOrgSkillPlaces(t, "sync-docs undo", []orgSkillPlace{
		{"item 1 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 1) }, []string{
			orgSkillSyncDocsCheck,
			"(ここまでで何も出なくなっているので、出たものはすべて sync-docs の変更)",
			"report のほかに、予約の外のパスか機能の計画が 1 つでも変わっていれば、report のほかの sync-docs の変更をすべて戻し" +
				"(追跡しているファイルは `git restore <パス>`、新しいファイルは消す)、戻した変更を 1 件ずつ follow-up として 6 のために控える",
			"予約の外の分だけを戻すと、写しの組の片方だけが変わったまま残って gate の写しの検査が落ち、予約の外を直せない implementer も直せないので、" +
				"一部だけを戻すことはしない",
			"計画を戻すのは、per-plan では承認のあとに計画が変わると implementer への TASK が断られるため(「機能の計画の承認」)",
			"どちらもなければ変更はそのまま残す",
			"残った変更と report を 1 つのコミットにし、`git status --porcelain` が何も出さないことを確かめる",
			"report のほかに変更がなければ report だけでコミットし、report も前の回のコミットと同じで何も出なければ、コミットせずに 2 に進む",
		}},
		{"the subsection ### 機能の計画の承認", func(text string) string { return directorSkillSubsection(text, "### 機能の計画の承認") }, []string{
			"今のファイルの digest(`scripts/plan-visual.sh digest` と同じ値)がその digest と一致する",
		}},
	})
}

// TestOrgSkill_EndOfOrgSyncDocsRunsAgainOnARedo pins AC6 of plan
// 2026-10-30-leader-sync-docs for ### org の終わり方: a redo of the gate (a
// GATE: fail, or a gate refusal whose fix needs the implementer) passes item
// 1 again, so sync-docs runs again on the new head and the follow-ups and
// SYNC_DOCS: line kept for item 6 are the last run's; item 4's GATE: fail
// bullet takes a failure in files the sync-docs commit changed the same way.
func TestOrgSkill_EndOfOrgSyncDocsRunsAgainOnARedo(t *testing.T) {
	const gateFail, gateUnrunnable = "- `GATE: fail` の `BLOCKED`:", "- `GATE: unrunnable` の `BLOCKED`:"
	checkOrgSkillPlaces(t, "sync-docs redo", []orgSkillPlace{
		{"item 1 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 1) }, []string{
			orgSkillSyncDocsCheck,
			"gate をやり直すとき(4 の `GATE: fail` や、implementer の手が要る gate の断りのあと)は、implementer と reviewer の往復のあとに 1 を通り直すので、" +
				"sync-docs も新しい HEAD でもう一度走る",
			"控えた follow-up と `SYNC_DOCS:` の行は、そのたびに最後の回のものに置き換える",
		}},
		{"the GATE: fail bullet of item 4 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgBullet(text, gateFail) }, []string{
			"`REASON:` と `LOG:` を本文に書いて implementer に直しを TASK で頼み、reviewer が通したら 1 からやり直す",
			"1 の sync-docs のコミットが変えたファイルで落ちたとき(写しの検査など)も同じで、implementer に予約の中で直しを頼む",
			"1 からやり直すと、その終わりで sync-docs がもう一度走る",
		}},
		// The refusal that needs the implementer is the GATE: unrunnable
		// bullet's, which also comes back to item 1.
		{"the GATE: unrunnable bullet of item 4 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgBullet(text, gateUnrunnable) }, []string{
			"次の手に implementer の手が要るとき",
			"implementer に頼み、reviewer が通したら 1 からやり直す",
		}},
	})
}

// TestOrgSkill_EndOfOrgStep6CarriesTheDocFollowups pins AC4 of plan
// 2026-10-30-leader-sync-docs (D3, D5) for item 6 of ### org の終わり方: the
// PR body has the follow-up section the sync-docs report has, with the last
// report's items and the changes item 1 undid (なし when none, the skip
// reason when skipped), and the RESULT's body carries the same items as one
// DOC_FOLLOWUP: line each, DOC_FOLLOWUP: なし when none, one line with the
// count pointing at the PR body when they do not fit the body cap, and the
// SYNC_DOCS: skipped line instead when sync-docs was skipped.
func TestOrgSkill_EndOfOrgStep6CarriesTheDocFollowups(t *testing.T) {
	checkOrgSkillPlaces(t, "doc follow-up", []orgSkillPlace{
		{"item 6 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 6) }, []string{
			"`/pr` skill は実行しない",
			"PR の本文には `" + syncDocsFollowupHeading + "` の節を置き、1 で最後に走らせた sync-docs の report の同じ名前の節の項と、" +
				"1 で戻して控えた変更を、1 件 1 行(パスと、何が古いか)で書く",
			"(どちらもなければ `なし`。sync-docs を飛ばしたときは、控えた " + syncDocsSkipLine + " の理由)",
			"PR ができたら、4 で控えた `HEAD:` と `BASE:` の 2 行をヘッダに写し、PR の URL を EVIDENCE に書いて `RESULT` を escalate する(TASK_ID は org_id)",
			"RESULT の本文には、PR の本文の follow-up の節と同じ項を、1 件 1 行の `" + docFollowupKey + ": <パス>: <ずれ>` で書く" +
				"(なければ `" + docFollowupKey + ": なし` の 1 行)",
			"本文が 2,000 字に収まらなければ、その行の代わりに、件数と PR の本文の節を指す 1 行" +
				"(`" + docFollowupKey + ": 14 件(PR の本文の「予約外の文書の follow-up」の節)` など)を書く",
			"sync-docs を飛ばしたときは、`" + docFollowupKey + ":` の行の代わりに、控えた " + syncDocsSkipLine + " の行を書く",
		}},
	})
	if protocol.DefaultMaxBodyChars != 2000 {
		t.Errorf("protocol.DefaultMaxBodyChars = %d, but item 6 of the /org skill's ### org の終わり方 says the RESULT's body is capped at 2,000 characters", protocol.DefaultMaxBodyChars)
	}
}

// TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate pins that items
// 1 and 6 of the /org skill's ### org の終わり方 use the names the leader
// template's steps 9 and 14 use (plan 2026-10-30-leader-sync-docs, S2): the
// doc-maintainer's call and definitions, the report's path and follow-up
// section, the skip line, and the DOC_FOLLOWUP: lines. A leader runs the
// template and a director or human reads the skill, so a name changed in one
// and not the other would have them look for different files and lines.
func TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate(t *testing.T) {
	step9, step14 := leaderFeatureOrgItem(t, step9Marker), leaderFeatureOrgItem(t, leaderStep14Marker)
	root := directorSkillRepoRoot(t)
	for _, c := range []struct {
		name, template string
		skillStep      int
		names          []string
	}{
		{"step 9 / item 1", step9, 1, []string{
			"`Task(subagent_type=\"doc-maintainer\")`",
			"`.codex/agents/doc-maintainer.toml`",
			"`.claude/agents/doc-maintainer.md`",
			"`docs/reports/sync-docs-<YYYY-MM-DD>-<機能の slug>.md`",
			"`" + syncDocsFollowupHeading + "`",
			syncDocsSkipLine,
		}},
		{"step 14 / item 6", step14, 6, []string{
			"`" + syncDocsFollowupHeading + "`",
			syncDocsSkipLine,
			"`" + docFollowupKey + ": <パス>: <ずれ>`",
			"`" + docFollowupKey + ": なし`",
		}},
	} {
		for _, name := range c.names {
			if !containsPhrase(c.template, name) {
				t.Errorf("expected the leader template's %s to name %s, got item:\n%s", c.name, name, c.template)
			}
		}
		for _, rel := range orgSkillCopies {
			data, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Skipf("%s not found (%v) -- skipping the skill's sync-docs names check (vendored repo?)", rel, err)
			}
			step := orgSkillEndOfOrgStep(string(data), c.skillStep)
			for _, name := range c.names {
				if !containsPhrase(step, name) {
					t.Errorf("%s: expected item %d of ### org の終わり方 to name %s, as the leader template's %s does, got item:\n%s", rel, c.skillStep, name, c.name, step)
				}
			}
		}
	}
}
