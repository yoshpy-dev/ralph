package org

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// leaderFeatureOrgItem renders the leader template and returns the item of its
// 機能ごとの org section that contains marker.
func leaderFeatureOrgItem(t *testing.T, marker string) string {
	t.Helper()
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	item, ok := markdownItem(section, marker)
	if !ok {
		t.Fatalf("expected the 機能ごとの org section to have an item naming %q, got section:\n%s", marker, section)
	}
	return item
}

// assertPhrasesInOrder fails the test unless item contains every phrase of
// order, ignoring white space (containsPhrase), each after the one before it.
func assertPhrasesInOrder(t *testing.T, item, marker string, order []string) {
	t.Helper()
	s, at := squashSpace(item), -1
	for _, want := range order {
		i := strings.Index(s, squashSpace(want))
		if i < 0 || i < at {
			t.Errorf("expected the 機能ごとの org item naming %q to say %q in this order, but %q is missing or out of order, got item:\n%s", marker, order, want, item)
			return
		}
		at = i
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanReadsTheRulesAndCodeFirst pins
// AC1 of plan docs/plans/active/2026-10-29-leader-plan-depth.md: step 1 of
// the 機能ごとの org section has the leader read AGENTS.md (and CLAUDE.md),
// .claude/rules/ (and, under Codex, .codex/AGENTS.override.md) and the code
// the plan touches before it writes the plan, ahead of the rest of the step.
func TestRenderRolePrompt_Leader_FeatureOrgPlanReadsTheRulesAndCodeFirst(t *testing.T) {
	const marker = "1. 機能の計画を"
	item := leaderFeatureOrgItem(t, marker)
	assertPhrasesInOrder(t, item, marker, []string{
		"機能の計画を worktree の `docs/plans/active/` に書き、コミットする",
		"書く前に、worktree の AGENTS.md(あれば CLAUDE.md も)と `.claude/rules/` の下の rule(Codex では、あれば `.codex/AGENTS.override.md` も)を読み、計画が触るコードを読む",
		"計画のスライス・検証のコマンド・してはいけないことは、読んだ決まりと今のコードに合わせる",
		"プロジェクトに計画の雛形",
		"計画を書くときは、機能の内側の分かれ目",
	})
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanListsTheForksInsideTheFeature
// pins AC2 of plan 2026-10-29-leader-plan-depth for step 1: the leader looks
// for forks inside the feature only, by the three conditions of the standard
// flow's /plan (all three), leaves forks across features to the split plan,
// writes 2 to 4 options with a recommendation and its reason in
// ## Design decisions and the rest of the plan with the recommended option,
// writes `Critical forks: None` when there is none, asks per-plan in the
// QUESTION of step 3, and under split takes the recommendation without
// asking and records that it did.
func TestRenderRolePrompt_Leader_FeatureOrgPlanListsTheForksInsideTheFeature(t *testing.T) {
	const marker = "1. 機能の計画を"
	item := leaderFeatureOrgItem(t, marker)
	assertPhrasesInOrder(t, item, marker, []string{
		"機能の内側の分かれ目(この機能の中だけで決まる、どの案で作るかの選択)を探す",
		"分かれ目は、次の 3 つをすべて満たす選択に限る(標準フローの `/plan` が人に聞く分かれ目と同じ)",
		"案によって、リスク・費用・戻しやすさが大きく違う",
		"コード・rules・文書・妥当な既定値のどれでも決まらない",
		"実装の途中で変えると、およそ 1 スライスを作り直すより高くつく",
		"機能をまたぐ分かれ目(ほかの機能の org と合わせて決める選択)は、分割計画で決めるものなので扱わず、手順 3 の `FORK` の行にも書かない",
		"分かれ目ごとに、計画の `## Design decisions` の節(雛形になければ作る)に 2〜4 の案と推奨の案とその理由を書き、計画のほかの部分は推奨の案で書く",
		"分かれ目がなければ、その節に `Critical forks: None` と書く",
		"per-plan では、分かれ目を手順 3 の QUESTION で人に聞く",
		"split では人に聞かずに推奨の案を採り、分かれ目ごとに、承認方式が split なので推奨の案を採ったことを `## Design decisions` に書く",
	})
}

// TestRenderRolePrompt_Leader_FeatureOrgQuestionListsEveryFork pins AC2 of
// plan 2026-10-29-leader-plan-depth for step 3: every fork goes into the
// QUESTION's body, one line each in the FORK form with the recommended option
// first, with no cap on their number (D3: the director asks them all); a body
// over the 2,000-character cap shortens the lines instead of dropping any;
// and the QUESTION raised again after a change request lists the forks that
// request decided as (決定済み) lines with no options.
func TestRenderRolePrompt_Leader_FeatureOrgQuestionListsEveryFork(t *testing.T) {
	const marker = "3. (per-plan)"
	item := leaderFeatureOrgItem(t, marker)
	if protocol.DefaultMaxBodyChars != 2000 {
		t.Fatalf("protocol.DefaultMaxBodyChars = %d, but step 3 of the leader template says the QUESTION's body is capped at 2,000 characters", protocol.DefaultMaxBodyChars)
	}
	assertPhrasesInOrder(t, item, marker, []string{
		"手順 1 の分かれ目は、空行のあとの本文に、1 つ 1 行で `FORK <番号>: <問い> / a <案>(推奨) / b <案>` の形ですべて書く",
		"推奨の案を a にし、案が 3 つ以上なら `/ c <案>` と続ける。番号は `## Design decisions` の順に 1 から振る",
		"数が多くても打ち切らない",
		"`FORK` の行に理由は書かず、計画の `## Design decisions` に任せる",
		"本文が 2,000 字に収まらなければ、各行の問いと案を短く言い直し、行は削らない",
		"分かれ目がなければ `FORK` の行は書かない",
		"推奨と違う案を選んだ直しの依頼(手順 4)で計画を直し、QUESTION を上げ直すときは、その依頼で決まった分かれ目を `FORK <番号>: <問い> → <決まった案>(決定済み)` の 1 行にし、案を並べない(director は `(決定済み)` の行を人に聞かない)",
		"直しで新しく出た分かれ目は、上の形で案を並べる",
	})
	// D3: no cap on the number of forks, so no wording that stops at some
	// count or takes the recommendation from some fork on.
	text := renderSeatPrompt(t, "leader")
	for _, banned := range []string{"つ目から", "つまで"} {
		if strings.Contains(squashSpace(text), banned) {
			t.Errorf("expected the leader template to put every fork in the QUESTION, with no count cap, but it says %q:\n%s", banned, text)
		}
	}
}

// forkLine is one first-round FORK line of the approval QUESTION's body: the
// number, the question, the recommended option a, then option b and up to two
// more.
var forkLine = regexp.MustCompile(`^FORK (\d+): \S[^/]* / a \S[^/]*\(推奨\)( / [b-d] \S[^/]*){1,3}$`)

// TestRenderRolePrompt_Leader_FeatureOrgQuestionExampleCarriesForks pins the
// example QUESTION of step 3 (AC2 of plan 2026-10-29-leader-plan-depth):
// escalate still accepts it as a QUESTION and ParsePlanHeader reads its
// PLAN: header, its FORK lines are in the body (after the blank line), not
// in the header block, numbered from 1, each in the FORK form with only
// option a marked (推奨), and none of them is a (決定済み) line.
func TestRenderRolePrompt_Leader_FeatureOrgQuestionExampleCarriesForks(t *testing.T) {
	item := leaderFeatureOrgItem(t, "3. (per-plan)")
	const fence = "\n   ```\n"
	_, rest, ok := strings.Cut(item, fence)
	block, _, closed := strings.Cut(rest, fence)
	if !ok || !closed {
		t.Fatalf("expected step 3 to show the example QUESTION in an indented code block, got item:\n%s", item)
	}
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, "   ")
	}
	msg := strings.Join(lines, "\n")

	m, err := validateEscalation("org-a", msg)
	if err != nil || m.Type != protocol.TypeQuestion {
		t.Fatalf("expected escalate to accept the example as a QUESTION, got TYPE %q, err %v; message:\n%s", m.Type, err, msg)
	}
	if _, err := ParsePlanHeader(msg); err != nil {
		t.Errorf("ParsePlanHeader(example QUESTION) = %v; message:\n%s", err, msg)
	}
	// "FORK 1" has a space, so it is never a header key: a FORK line in the
	// header block would end the block and start the body itself. The
	// blank line has to end the block before the first FORK line.
	header, body, ok := strings.Cut(msg, "\n\n")
	if !ok || strings.Contains(header, "FORK") || m.Body != body {
		t.Errorf("expected the example's header block to end at the blank line, with every FORK line in the body after it; header block:\n%s\nparsed body:\n%s", header, m.Body)
	}

	var forks []string
	for _, line := range strings.Split(m.Body, "\n") {
		if strings.HasPrefix(line, "FORK") {
			forks = append(forks, line)
		}
	}
	if len(forks) == 0 {
		t.Fatalf("expected the example QUESTION's body to have a FORK line, got body:\n%s", m.Body)
	}
	for i, line := range forks {
		sub := forkLine.FindStringSubmatch(line)
		if sub == nil {
			t.Errorf("expected FORK line %d of the example to have the form `FORK <番号>: <問い> / a <案>(推奨) / b <案>`, got %q", i+1, line)
			continue
		}
		if sub[1] != strconv.Itoa(i+1) {
			t.Errorf("expected the example's FORK lines to be numbered from 1, got %q as line %d", line, i+1)
		}
		if n := strings.Count(line, "(推奨)"); n != 1 {
			t.Errorf("expected only option a of %q to be marked (推奨), got %d marks", line, n)
		}
		if strings.Contains(line, "(決定済み)") {
			t.Errorf("expected the first-round example to have no (決定済み) line, got %q", line)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgNonRecommendedForkIsAChangeRequest
// pins D1 of plan 2026-10-29-leader-plan-depth in step 4: a DECISION whose
// body starts with approved stays the approval and, with forks, means every
// fork took its recommended option, so ## Design decisions is not rewritten
// (the approved digest would no longer match); an answer that picks another
// option arrives as a change request, and the leader writes the chosen
// option into ## Design decisions and the parts it changes, commits, and
// redoes steps 2 and 3.
func TestRenderRolePrompt_Leader_FeatureOrgNonRecommendedForkIsAChangeRequest(t *testing.T) {
	const marker = "4. (per-plan)"
	item := leaderFeatureOrgItem(t, marker)
	assertPhrasesInOrder(t, item, marker, []string{
		"本文が `approved` で始まるものを承認とみなす",
		"分かれ目があれば、この承認は、すべての分かれ目が推奨の案で決まったことも表すので、`## Design decisions` は書き換えない(手順 6)",
		"それ以外の DECISION(直しの依頼)なら、計画を直してコミットし、図解ページを描き直し、新しい digest で新しい QUESTION を上げる(手順 2 と 3 をやり直す)",
		"分かれ目で推奨と違う案を選んだ答えも、本文が `approved` で始まらないので、この直しの依頼として届く",
		"計画の `## Design decisions` に人が選んだ案を書き、計画のほかの部分(スライス・受け入れ条件・リスクなど)もその案に合わせて直し、上のとおりコミットして手順 2 と 3 をやり直す",
	})
}

// TestRenderRolePrompt_Leader_ForkWordingStaysInTheFeatureOrgSection pins
// that the plan reading and the fork rules of plan
// 2026-10-29-leader-plan-depth live in the 機能ごとの org section only: an
// org that is not a split plan's feature follows its task, and the template
// keeps that split.
func TestRenderRolePrompt_Leader_ForkWordingStaysInTheFeatureOrgSection(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	if n := strings.Count(text, section); n != 1 {
		t.Fatalf("expected the 機能ごとの org section's text once in the template, got %d", n)
	}
	outside := strings.Replace(text, section, "", 1)
	for _, word := range []string{"AGENTS.md", "AGENTS.override.md", "分かれ目", "FORK", "(推奨)", "(決定済み)", "Design decisions", "Critical forks"} {
		if !strings.Contains(section, word) {
			t.Errorf("expected the 機能ごとの org section to contain %q, got section:\n%s", word, section)
		}
		if strings.Contains(outside, word) {
			t.Errorf("expected %q only in the 機能ごとの org section, but the rest of the template has it:\n%s", word, outside)
		}
	}
}
