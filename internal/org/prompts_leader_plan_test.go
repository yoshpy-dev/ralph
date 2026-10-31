package org

import (
	"context"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The markers of the 機能ごとの org steps the tests below read, and the
// heading of the subsection that holds the plan advisory (self-review L6 of
// plan 2026-10-29-leader-plan-depth moved it out of step 1).
const (
	leaderStep1Marker         = "1. 機能の計画を"
	leaderStep3Marker         = "3. (per-plan)"
	leaderPlanAdvisoryHeading = "### 計画助言"
	leaderAdvisoryWhere       = "the 計画助言 subsection of the 機能ごとの org section"
)

// leaderAdvisoryPointer is the sentence that ends step 1 and points at the
// 計画助言 subsection.
const leaderAdvisoryPointer = "計画をコミットしたら、次に進む前に、この節の終わりの「計画助言」のとおりに計画助言を扱う"

// leaderFeatureOrgSection renders the leader template and returns its
// 機能ごとの org section.
func leaderFeatureOrgSection(t *testing.T) string {
	t.Helper()
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	return section
}

// leaderFeatureOrgItem renders the leader template and returns the item of its
// 機能ごとの org section that contains marker.
func leaderFeatureOrgItem(t *testing.T, marker string) string {
	t.Helper()
	section := leaderFeatureOrgSection(t)
	item, ok := markdownItem(section, marker)
	if !ok {
		t.Fatalf("expected the 機能ごとの org section to have an item naming %q, got section:\n%s", marker, section)
	}
	return item
}

// leaderItemWhere names the 機能ごとの org item naming marker in a failure
// message.
func leaderItemWhere(marker string) string {
	return "the 機能ごとの org item naming " + strconv.Quote(marker)
}

// leaderPlanAdvisory returns the 計画助言 subsection of the 機能ごとの org
// section: the text after its heading line up to the next "### " line or the
// end of the section.
func leaderPlanAdvisory(t *testing.T) string {
	t.Helper()
	section := leaderFeatureOrgSection(t)
	_, body, found := strings.Cut(section, "\n"+leaderPlanAdvisoryHeading+"\n")
	if !found {
		t.Fatalf("expected the 機能ごとの org section to have the subsection %q, got section:\n%s", leaderPlanAdvisoryHeading, section)
	}
	body, _, _ = strings.Cut(body, "\n### ")
	return body
}

// assertPhrasesInOrder fails the test unless text contains every phrase of
// order, ignoring white space (containsPhrase), each after the one before it.
// where names text in the failure message.
func assertPhrasesInOrder(t *testing.T, text, where string, order []string) {
	t.Helper()
	s, at := squashSpace(text), -1
	for _, want := range order {
		i := strings.Index(s, squashSpace(want))
		if i < 0 || i < at {
			t.Errorf("expected %s to say %q in this order, but %q is missing or out of order, got:\n%s", where, order, want, text)
			return
		}
		at = i
	}
}

// fencedBlockAfter returns the first fenced code block of text that opens
// after the line holding lead, each line without the fence's indentation (a
// block inside a list item is indented with the item, one in the 計画助言
// subsection is not). where names text in the failure message.
func fencedBlockAfter(t *testing.T, text, lead, where string) string {
	t.Helper()
	at := strings.Index(text, lead)
	if at < 0 {
		t.Fatalf("expected %s to have the sentence %q, got:\n%s", where, lead, text)
	}
	lines := strings.Split(text[at:], "\n")
	if open := slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == "```" }); open >= 0 {
		indent := strings.TrimSuffix(lines[open], "```")
		if n := slices.Index(lines[open+1:], indent+"```"); n >= 0 {
			block := slices.Clone(lines[open+1 : open+1+n])
			for i, line := range block {
				block[i] = strings.TrimPrefix(line, indent)
			}
			return strings.Join(block, "\n")
		}
	}
	t.Fatalf("expected a fenced code block after %q in %s, got:\n%s", lead, where, text)
	return ""
}

// leaderQuestionExample returns step 3's item and its example QUESTION as
// escalate reads it (the item's code block, its indentation removed), with
// the message escalate parsed. It fails the test unless escalate accepts the
// example as a QUESTION and ParsePlanHeader reads its PLAN: header.
func leaderQuestionExample(t *testing.T) (item, msg string, m protocol.Message) {
	t.Helper()
	item = leaderFeatureOrgItem(t, leaderStep3Marker)
	msg = fencedBlockAfter(t, item, leaderStep3Marker, leaderItemWhere(leaderStep3Marker))
	m, err := validateEscalation("org-a", msg)
	if err != nil || m.Type != protocol.TypeQuestion {
		t.Fatalf("expected escalate to accept the example as a QUESTION, got TYPE %q, err %v; message:\n%s", m.Type, err, msg)
	}
	if _, err := ParsePlanHeader(msg); err != nil {
		t.Errorf("ParsePlanHeader(example QUESTION) = %v; message:\n%s", err, msg)
	}
	return item, msg, m
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanReadsTheRulesAndCodeFirst pins
// AC1 of plan docs/plans/active/2026-10-29-leader-plan-depth.md: step 1 of
// the 機能ごとの org section has the leader read AGENTS.md (and CLAUDE.md),
// .claude/rules/ (and, under Codex, .codex/AGENTS.override.md) and the code
// the plan touches before it writes the plan, ahead of the rest of the step.
func TestRenderRolePrompt_Leader_FeatureOrgPlanReadsTheRulesAndCodeFirst(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderItemWhere(leaderStep1Marker), []string{
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
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderItemWhere(leaderStep1Marker), []string{
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

// leaderForkCountCap matches wording that caps the number of forks, compared
// with white space removed: a count followed by つまで, つ目から, and the like.
var leaderForkCountCap = regexp.MustCompile(`[0-9一二三四五六七八九十]+(つ|件|個|行)(目から|目以降|まで)`)

// TestRenderRolePrompt_Leader_FeatureOrgQuestionListsEveryFork pins AC2 of
// plan 2026-10-29-leader-plan-depth for step 3: every fork goes into the
// QUESTION's body, one line each in the FORK form with the recommended option
// first, with no cap on their number (D3: the director asks them all); a body
// over the 2,000-character cap shortens the lines instead of dropping any;
// and the QUESTION raised again after a change request lists the forks that
// request decided as (決定済み) lines with no options. The no-cap check reads
// step 1's fork paragraph and step 3 only, for a count before つまで or
// つ目から, so an ordinary 「返事を待つまで」 elsewhere does not trip it
// (self-review L3).
func TestRenderRolePrompt_Leader_FeatureOrgQuestionListsEveryFork(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep3Marker)
	if protocol.DefaultMaxBodyChars != 2000 {
		t.Fatalf("protocol.DefaultMaxBodyChars = %d, but step 3 of the leader template says the QUESTION's body is capped at 2,000 characters", protocol.DefaultMaxBodyChars)
	}
	assertPhrasesInOrder(t, item, leaderItemWhere(leaderStep3Marker), []string{
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
	for _, s := range []struct {
		text string
		caps bool
	}{
		{"分かれ目は 3 つまで書く", true}, {"4 つ目からは推奨の案を採る", true}, {"5 件まで並べる", true},
		{"返事を待つまで", false}, {"案が 3 つ以上なら", false}, {"1 つ 1 行で", false},
	} {
		if got := leaderForkCountCap.MatchString(squashSpace(s.text)); got != s.caps {
			t.Fatalf("leaderForkCountCap.MatchString(%q) = %v, want %v", s.text, got, s.caps)
		}
	}
	var forks string
	for _, p := range strings.Split(leaderFeatureOrgItem(t, leaderStep1Marker), "\n\n") {
		if containsPhrase(p, "計画を書くときは、機能の内側の分かれ目") {
			forks = p
		}
	}
	if forks == "" {
		t.Fatalf("expected step 1 to have the fork paragraph")
	}
	for _, c := range []struct{ where, text string }{{"the fork paragraph of step 1", forks}, {"step 3", item}} {
		if found := leaderForkCountCap.FindString(squashSpace(c.text)); found != "" {
			t.Errorf("expected %s to put every fork in the QUESTION, with no count cap, but it says %q:\n%s", c.where, found, c.text)
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
	_, msg, m := leaderQuestionExample(t)
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
	assertPhrasesInOrder(t, item, leaderItemWhere(marker), []string{
		"本文が `approved` で始まるものを承認とみなす",
		"分かれ目があれば、この承認は、すべての分かれ目が推奨の案で決まったことも表すので、`## Design decisions` は書き換えない(手順 6)",
		"それ以外の DECISION(直しの依頼)なら、計画を直してコミットし、図解ページを描き直し、新しい digest で新しい QUESTION を上げる(手順 2 と 3 をやり直す)",
		"分かれ目で推奨と違う案を選んだ答えも、本文が `approved` で始まらないので、この直しの依頼として届く",
		"計画の `## Design decisions` に人が選んだ案を書き、計画のほかの部分(スライス・受け入れ条件・リスクなど)もその案に合わせて直し、上のとおりコミットして手順 2 と 3 をやり直す",
	})
}

// TestRenderRolePrompt_Leader_PlanWordingStaysInTheFeatureOrgSection pins
// that the plan reading, the fork rules, and the plan advisory of plan
// 2026-10-29-leader-plan-depth live in the 機能ごとの org section only (the
// advisory in its 計画助言 subsection, which markdownSection keeps in the
// section): an org that is not a split plan's feature follows its task, and
// the template keeps that split.
func TestRenderRolePrompt_Leader_PlanWordingStaysInTheFeatureOrgSection(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	if n := strings.Count(text, section); n != 1 {
		t.Fatalf("expected the 機能ごとの org section's text once in the template, got %d", n)
	}
	outside := strings.Replace(text, section, "", 1)
	for _, c := range []struct {
		name  string
		words []string
	}{
		{"the plan reading and the forks", []string{"AGENTS.md", "AGENTS.override.md", "分かれ目", "FORK", "(推奨)", "(決定済み)", "Design decisions", "Critical forks"}},
		{"the plan advisory", []string{"plan_advisory", "計画助言", "pick_reviewer", "adversarial plan reviewer", "mcp_servers", "--disable plugins", "--disable apps", "claude -p", "plan-advisory", "ADVISORY:"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			for _, word := range c.words {
				if !strings.Contains(section, word) {
					t.Errorf("expected the 機能ごとの org section to contain %q, got section:\n%s", word, section)
				}
				if strings.Contains(outside, word) {
					t.Errorf("expected %q only in the 機能ごとの org section, but the rest of the template has it:\n%s", word, outside)
				}
			}
		})
	}
}

// TestRenderRolePrompt_Leader_PlanAdvisoryIsASubsectionAfterTheSteps pins
// self-review L6 of plan 2026-10-29-leader-plan-depth: the plan advisory is
// the subsection ### 計画助言 at the end of the 機能ごとの org section, after
// step 16, and every line from step 1 to it that starts at column 0 starts
// a numbered step, so the steps stay one markdown list. Step 1 ends with the
// one sentence that points at the subsection and holds none of the advisory;
// the subsection holds the advisory's commands.
func TestRenderRolePrompt_Leader_PlanAdvisoryIsASubsectionAfterTheSteps(t *testing.T) {
	section := leaderFeatureOrgSection(t)
	first, last := strings.Index(section, "\n"+leaderStep1Marker), strings.Index(section, "\n16. ")
	sub := strings.Index(section, "\n"+leaderPlanAdvisoryHeading+"\n")
	if first < 0 || last < first || sub < last {
		t.Fatalf("expected steps 1 to 16 and then the subsection %q in the 機能ごとの org section (at %d, %d, %d), got section:\n%s", leaderPlanAdvisoryHeading, first, last, sub, section)
	}
	for _, line := range strings.Split(section[first+1:sub], "\n") {
		if line != "" && !strings.HasPrefix(line, " ") && !numberedItemStart.MatchString(line) {
			t.Errorf("expected every line between step 1 and the subsection %q to be in a numbered step, but %q starts at column 0", leaderPlanAdvisoryHeading, line)
		}
	}

	step1 := leaderFeatureOrgItem(t, leaderStep1Marker)
	if !strings.HasSuffix(squashSpace(step1), squashSpace(leaderAdvisoryPointer)) {
		t.Errorf("expected step 1 to end with %q, got item:\n%s", leaderAdvisoryPointer, step1)
	}
	advisory := leaderPlanAdvisory(t)
	for _, word := range []string{"plan_advisory", "pick_reviewer", "adversarial plan reviewer", "mcp_servers", "claude -p", "ADVISORY:", "```"} {
		if strings.Contains(step1, word) {
			t.Errorf("expected step 1 to leave %q to the subsection %q, got item:\n%s", word, leaderPlanAdvisoryHeading, step1)
		}
		if !strings.Contains(advisory, word) {
			t.Errorf("expected the subsection %q to contain %q, got:\n%s", leaderPlanAdvisoryHeading, word, advisory)
		}
	}
}

// The leads below are the sentences each of the three code blocks of the
// 計画助言 subsection follows.
const (
	leaderAdvisoryPromptLead = "相手には次の指示文を渡す"
	leaderAdvisoryCodexLead  = "相手が codex なら次のコマンドを打つ"
	leaderAdvisoryClaudeLead = "相手が claude なら次のコマンドを打つ"
)

// leaderAdvisorySelfCLI is the placeholder of the pick command that the
// leader replaces with its own CLI, claude or codex (self-review L5).
const leaderAdvisorySelfCLI = "<あなたの CLI>"

// The placeholders of the advisory commands that the leader replaces with the
// two words of the task's codex reviewer line (featureReviewerCodexLead) and
// of its claude reviewer line (featureReviewerClaudeLead): the model and the
// effort start --plan read from [reviewer] (plan
// 2026-10-30-standard-flow-config, D5).
const (
	leaderAdvisoryCodexModel   = "<reviewer の codex のモデル>"
	leaderAdvisoryCodexEffort  = "<reviewer の codex の effort>"
	leaderAdvisoryClaudeModel  = "<reviewer の claude のモデル>"
	leaderAdvisoryClaudeEffort = "<reviewer の claude の effort>"
)

// leaderAdvisoryPickCommand is the command the 計画助言 subsection picks the
// other CLI with, the driver reading of /plan step 11.b (self-review L10):
// pick_reviewer reads the driver ralph puts in the pane when it is claude or
// codex, case-insensitive, and the leader's own CLI otherwise.
const leaderAdvisoryPickCommand = `. ./scripts/xreview-helpers.sh && DRIVER=` + leaderAdvisorySelfCLI +
	`; override=$(printf '%s' "${` + driver.PrimaryCLIEnv + `:-}" | tr '[:upper:]' '[:lower:]'); ` +
	`case "$override" in claude|codex) DRIVER=$override ;; esac; pick_reviewer "$DRIVER"`

// leaderAdvisoryGitState is the command both advisory commands write the git
// state with before and after the call: the form of /plan step 11 of #266
// (HEAD, the index, the work tree with its untracked files' contents).
const leaderAdvisoryGitState = "{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; " +
	"git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; }"

// leaderAdvisoryWatchdog is the 20-minute watchdog of /plan step 11.c both
// advisory commands run the reviewer under, up to the echo of its rc.
const leaderAdvisoryWatchdog = ` & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & ` +
	`wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "`

// leaderAdvisoryCodeBlock returns the code block of the 計画助言 subsection
// that comes first after the line holding lead.
func leaderAdvisoryCodeBlock(t *testing.T, lead string) string {
	t.Helper()
	return fencedBlockAfter(t, leaderPlanAdvisory(t), lead, leaderAdvisoryWhere)
}

// readRepoText reads rel under the repository root, skipping the test when it
// is missing (a repo that vendors internal/org alone).
func readRepoText(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(directorSkillRepoRoot(t), rel))
	if err != nil {
		t.Skipf("%s not found (%v) -- skipping the lock-step check (vendored repo?)", rel, err)
	}
	return string(data)
}

// goStringLiterals returns the values of the string literals of the Go
// source src, in order and joined, so a test can read a help text whatever
// its literals' quoting, concatenation, and line breaks.
func goStringLiterals(t *testing.T, src string) string {
	t.Helper()
	var s scanner.Scanner
	s.Init(token.NewFileSet().AddFile("src.go", -1, len(src)), []byte(src), nil, 0)
	var b strings.Builder
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			return b.String()
		}
		if tok == token.STRING {
			v, err := strconv.Unquote(lit)
			if err != nil {
				t.Fatalf("strconv.Unquote(%s): %v", lit, err)
			}
			b.WriteString(v)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryReadsTheSwitchFirst pins
// AC4 of plan 2026-10-29-leader-plan-depth: step 1 ends, after the forks
// paragraph, with the pointer to the 計画助言 subsection, which asks the other
// model for a plan advisory only when [org.review].plan_advisory is on,
// before step 2 under per-plan and before the implementer's spawn of step 5
// under split (D6), once. The leader reads the switch with ralph config get
// and sorts its exit code the way that verb's help promises (D5): 0 uses the
// value, false calls nothing; an older ralph's unknown command "config" is
// off and recorded; and, as /plan step 11.a has it, the shell's 127 for a
// ralph not on PATH is off and recorded too (self-review L9); any other code
// is a BLOCKED, never off. The help in internal/cli/config.go still promises
// the codes other than 127, read from its string literals with white space
// removed, so rewrapping the help keeps the check green (self-review L3).
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryReadsTheSwitchFirst(t *testing.T) {
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, leaderStep1Marker), leaderItemWhere(leaderStep1Marker), []string{
		"split では人に聞かずに推奨の案を採り",
		leaderAdvisoryPointer,
	})
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		"手順 1 で計画をコミットしたら、`[org.review].plan_advisory` が on のときだけ、相手のモデルに計画助言を頼む(claude の leader は codex に、codex の leader は claude に、計画を読むだけの形で)",
		"per-plan では手順 2 の前に、split では手順 5 で implementer を spawn する前に済ませる",
		"頼むのは 1 回で、手順 4 と 6 で計画を直して手順 2 からやり直すときは頼み直さない",
		"まず worktree で `ralph config get org.review.plan_advisory` を打ち、終了コードで分ける(127 のほかは `ralph config get --help` の約束)",
		"0 なら出た値を使い、`true` なら頼み、`false` なら頼まずに次へ進む(記録も要らない)",
		"1 で stderr に `unknown command \"config\"` があれば(この動詞のない古い ralph)、off とみなして頼まず、off とみなしたことを下の要約と同じ場所に書く",
		"127 なら(`ralph` が PATH になく、shell が返す値)、同じく off とみなして頼まず、`ralph` が見つからず off とみなしたことを同じ場所に書く",
		"それ以外(ほかの 1、2、3、0 で `true` でも `false` でもない値)は off とみなさず、エラーを本文に書いて BLOCKED(TASK_ID は `org-a`)を「件を上げる」の節の手順で `ralph org escalate` で上げ、返事が来るまで先に進まない(待ち方は「agmsg の受信箱の運用」の節)",
		"頼むときは、worktree に",
	})

	// The check reads the help's words, not its Go literals: the same words
	// split into other literals pass too.
	const sample = "Treat that, and only that, as off."
	for _, src := range []string{"Long: \"Treat that, and only that, as\\n\" +\n\t\"     off.\"", "Long: `Treat that,\n and only that, as off.`"} {
		if got := squashSpace(goStringLiterals(t, src)); got != squashSpace(sample) {
			t.Fatalf("goStringLiterals(%q) = %q, want %q with white space removed", src, got, sample)
		}
	}
	help := squashSpace(goStringLiterals(t, readRepoText(t, filepath.Join("internal", "cli", "config.go"))))
	for _, promise := range []string{
		"0 the value was printed: use it.",
		`1 an older ralph without this verb exits 1 with unknown command "config" on stderr. Treat that, and only that, as off`,
		"Stop and report any other exit 1.",
		"2 the key is not a key of ralph.toml",
		"3 ralph.toml cannot be read",
		"stop and report it, do not treat it as off.",
	} {
		if !strings.Contains(help, squashSpace(promise)) {
			t.Errorf("the help of ralph config get in internal/cli/config.go no longer says %q, an exit code the 計画助言 subsection of the leader template sorts", promise)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryPicksTheOtherCLI pins the
// reviewer choice of AC5: the 計画助言 subsection skips when the worktree has
// no scripts/xreview-helpers.sh (it no longer needs scripts/ralph-config.sh;
// plan 2026-10-30-standard-flow-config, D5), picks the reviewer with
// pick_reviewer on the RALPH_PRIMARY_CLI ralph puts in the
// pane when it is claude or codex, case-insensitive, or on the leader's own
// CLI, which the leader writes in place of <あなたの CLI>, when the pane has
// none (a herdr with no --env whose server has none either; self-review L5
// and L7) or another value (self-review L10, the reading of /plan step
// 11.b), and skips when that CLI is not on PATH. The command, run with bash
// and zsh from the repository root on its scripts/xreview-helpers.sh, picks
// codex for a claude pane, claude for a codex pane, the pane's value over the
// leader's, whatever its case, and, with no RALPH_PRIMARY_CLI in the pane or
// one that is neither claude nor codex (gemini, or claude with a trailing
// space), codex for a claude leader and claude for a codex leader.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryPicksTheOtherCLI(t *testing.T) {
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		"頼むときは、worktree に `scripts/xreview-helpers.sh` があることを確かめ(なければ飛ばす)",
		"`" + leaderAdvisoryPickCommand + "` が出す名前(`codex` か `claude`)を相手にする",
		"`" + leaderAdvisorySelfCLI + "` は、あなたが Claude Code なら `claude`、Codex なら `codex` に書き換えて打つ",
		"`" + driver.PrimaryCLIEnv + "` が `claude` か `codex`(大文字と小文字は区別しない)ならその値があなたの CLI に代わり、ほかの値は無視する",
		"(ralph は pane に `" + driver.PrimaryCLIEnv + "` を入れる。`--env` のない herdr が作った pane には入らないが、herdr の server の環境にこの変数があれば pane はその値を引き継ぎ、その値で選ぶ。" +
			"書き換えたあなたの CLI が使われるのは、変数がないときと、値が `claude` でも `codex` でもないときだけ)",
		"`command -v <相手>` が何も出さなければ飛ばす",
	})

	root := directorSkillRepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "scripts", "xreview-helpers.sh")); err != nil {
		t.Skipf("scripts/xreview-helpers.sh not found (%v) -- skipping the run of the pick command (vendored repo?)", err)
	}
	for _, shell := range []string{"bash", "zsh"} {
		path, err := exec.LookPath(shell)
		if err != nil {
			continue
		}
		for _, tc := range []struct{ pane, self, want string }{
			{driver.PrimaryCLIEntry("claude"), "claude", "codex"},
			{driver.PrimaryCLIEntry("codex"), "codex", "claude"},
			{driver.PrimaryCLIEntry("codex"), "claude", "claude"},
			{driver.PrimaryCLIEntry("CODEX"), "claude", "claude"},
			{"", "claude", "codex"},
			{"", "codex", "claude"},
			{driver.PrimaryCLIEntry("gemini"), "codex", "claude"},
			{driver.PrimaryCLIEntry("claude "), "codex", "claude"},
		} {
			command := strings.Replace(leaderAdvisoryPickCommand, leaderAdvisorySelfCLI, tc.self, 1)
			args := []string{"-c", command}
			if shell == "zsh" {
				args = append([]string{"-f"}, args...)
			}
			cmd := exec.Command(path, args...)
			cmd.Dir = root
			cmd.Env = advisoryTestEnv()
			if tc.pane != "" {
				cmd.Env = append(cmd.Env, tc.pane)
			}
			out, err := cmd.Output()
			if got := strings.TrimSpace(string(out)); err != nil || got != tc.want {
				t.Errorf("%s -c %q with pane env %q = %q, err %v; want %q", shell, command, tc.pane, got, err, tc.want)
			}
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryPromptIsThePlanSkills pins
// the instruction both commands pass: one line, word for word the adversarial
// plan reviewer prompt of /plan step 11.c with the plan path as <計画のパス>,
// and nothing in it that would end or expand inside the double quotes of
// "<指示文>". The watchdog both commands use is /plan's too.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryPromptIsThePlanSkills(t *testing.T) {
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		"相手には次の指示文を渡す(下のコマンドの `<指示文>`)",
		"英語のまま変えず、最後の `<計画のパス>` だけを worktree からの計画の相対パスにする",
	})
	prompt := leaderAdvisoryCodeBlock(t, leaderAdvisoryPromptLead)
	if strings.Contains(prompt, "\n") || !strings.HasSuffix(prompt, "Here is the plan file to review: <計画のパス>") {
		t.Errorf("expected the instruction to be one line ending with the plan path, got:\n%s", prompt)
	}
	if strings.ContainsAny(prompt, "\"$`\\") {
		t.Errorf("expected the instruction to have no \", $, backquote or backslash, which the double quotes around <指示文> would not keep, got:\n%s", prompt)
	}

	plan := readRepoText(t, filepath.Join(".claude", "skills", "plan", "SKILL.md"))
	m := regexp.MustCompile(`"(You are an adversarial plan reviewer\.[^"]*)"`).FindStringSubmatch(plan)
	if m == nil {
		t.Fatalf("no adversarial plan reviewer prompt in .claude/skills/plan/SKILL.md")
	}
	if want := strings.Replace(m[1], "docs/plans/active/<plan-file>", "<計画のパス>", 1); prompt != want {
		t.Errorf("the leader's advisory instruction differs from /plan step 11.c's prompt:\n got: %s\nwant: %s", prompt, want)
	}
	if !strings.Contains(plan, leaderAdvisoryWatchdog) {
		t.Errorf("expected /plan step 11.c to use the watchdog the leader's advisory commands copy: %s", leaderAdvisoryWatchdog)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryRunsInOneShell pins how
// the 計画助言 subsection runs either command: in one shell call, under
// run_in_background with the turn ended in Claude Code and in the foreground
// with a tool timeout over 20 minutes in Codex; each stops the reviewer after
// 20 minutes, compares the git state from before and after the call, and
// writes its files under the ignored .harness/state/ so the gate does not
// refuse a dirty tree.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryRunsInOneShell(t *testing.T) {
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		"相手ごとに下のコマンドのどちらかを、まとめて 1 回の shell で打つ(Bash ツールは shell の状態を次の呼び出しに持ち越さない)",
		"Bash ツールの `run_in_background` があるとき(Claude Code)は `run_in_background` で走らせて手番を終え、終わりの知らせが来てから出力を読む",
		"ないとき(Codex)は前面で打ち、ツールの timeout を 20 分より長くする(そのあいだ `ralph org send` は届かない)",
		"どちらのコマンドも、`sleep 1200` の見張りで相手を 20 分で止め、呼ぶ前と後の git の状態(HEAD、index、未追跡のファイルを含む作業ツリー)をファイルに書いて `cmp` で比べる",
		"ファイルは worktree の `.harness/state/` の下に書く(git が無視する場所なので、手順 12 の gate が作業ツリーの汚れとして断らない)",
		leaderAdvisoryCodexLead,
	})
	for _, lead := range []string{leaderAdvisoryCodexLead, leaderAdvisoryClaudeLead} {
		block := leaderAdvisoryCodeBlock(t, lead)
		lines := strings.Split(block, "\n")
		const head = `s=.harness/state; mkdir -p "$s"; rm -f "$s/plan-advisory-out.md"`
		before := leaderAdvisoryGitState + ` > "$s/plan-advisory-git-before.txt"`
		after := leaderAdvisoryGitState + ` > "$s/plan-advisory-git-after.txt"`
		const compare = `if cmp -s "$s/plan-advisory-git-before.txt" "$s/plan-advisory-git-after.txt"; then echo 'git: same'; else echo 'git: changed'; fi`
		if n := len(lines); n < 5 || lines[0] != head || lines[1] != before || lines[n-2] != after || lines[n-1] != compare {
			t.Errorf("expected the command after %q to set up .harness/state and write the git state first, and to write it again and cmp the two last, got:\n%s", lead, block)
		}
		if strings.Count(block, leaderAdvisoryWatchdog) != 1 || strings.Count(block, "</dev/null") != 1 || strings.Count(block, `"<指示文>"`) != 1 {
			t.Errorf("expected the command after %q to run the reviewer once, with stdin closed, the instruction, and the 20-minute watchdog, got:\n%s", lead, block)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexTurnsMCPOff pins the
// codex command of AC5 and D7: it disables plugins, whose MCP servers
// -c mcp_servers.<name>.enabled=false cannot turn off, and apps, which act
// outside the command sandbox and stay with plugins off (plan
// 2026-11-01-org-cross-review, D6b; an older codex that cannot read the list
// with them skips), reads the names of the servers whose
// enabled is not false one line at a time into -c overrides (no word
// splitting), checks with the same arguments and the same filter that no such
// server is left, and only then runs codex exec with them, read-only and
// with --ignore-rules, its findings in -o; otherwise it says mcp: not off and
// calls nothing. A list entry with no enabled field, or a null one, counts as
// enabled in both lists (self-review L1). The model and the effort are the
// placeholders of the task's codex reviewer line, passed to -m and as
// model_reasoning_effort before exec.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexTurnsMCPOff(t *testing.T) {
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		leaderAdvisoryCodexLead,
		"`--sandbox read-only` と `--ignore-rules` は MCP の server に及ばないので、このコマンドは MCP の server を止めてから codex を呼ぶ",
		"plugin が持ち込む MCP の server は `-c mcp_servers.<名前>.enabled=false` では止まらない(codex が設定を読めなくなる)ので、`--disable plugins` で plugin ごと外す",
		"apps(codex に接続したアプリの道具)は一覧に出る MCP の server とは別のもので、コマンドの sandbox の外で動くので、`--disable apps` で外す(`--disable plugins` だけでは apps の機能が残る)",
		"`plugins` か `apps` の機能名を知らない古い codex で一覧が読めなければ、`mcp: not off` になり、codex を呼ばずに飛ばす",
		"`codex --disable plugins --disable apps mcp list --json` で有効な server の名前を読み、1 行ずつ `-c mcp_servers.<名前>.enabled=false` にして引数に足し(shell の語の分け方に頼らない)、同じ引数を付けた一覧で有効な server が 0 のときだけ codex を呼ぶ",
		"`enabled` が `false` でない server(欄がない、`null` のものを含む)はどちらの一覧でも有効と数えるので、codex が `false` と答えない server が残れば呼ばない",
		leaderAdvisoryClaudeLead,
	})
	block := leaderAdvisoryCodeBlock(t, leaderAdvisoryCodexLead)
	for _, want := range []string{
		`command codex --disable plugins --disable apps mcp list --json | jq -r '.[] | select(.enabled != false) | .name' > "$s/plan-advisory-mcp.txt"`,
		`set -- --disable plugins --disable apps; while IFS= read -r name; do [ -z "$name" ] || set -- "$@" -c "mcp_servers.$name.enabled=false"; done < "$s/plan-advisory-mcp.txt"`,
		`n=$(command codex "$@" mcp list --json | jq '[.[] | select(.enabled != false)] | length')`,
		`if [ "$n" = 0 ]; then command codex "$@" -m "` + leaderAdvisoryCodexModel + `" -c "model_reasoning_effort=` + leaderAdvisoryCodexEffort + `"` +
			` exec --sandbox read-only --ignore-rules -o "$s/plan-advisory-out.md" "<指示文>" </dev/null > "$s/plan-advisory.log" 2>&1` + leaderAdvisoryWatchdog + `codex rc=$rc"; else echo "mcp: not off (enabled: $n)"; fi`,
	} {
		if !strings.Contains(block, want) {
			t.Errorf("expected the codex command to contain %s, got:\n%s", want, block)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeReadsOnly pins the
// claude command of AC5 and D2, the form #266 settled on: claude -p with the
// reviewer model and, beside it before --restricted, the reviewer effort
// (--effort; plan 2026-10-30-standard-flow-config, AC4), both the placeholders
// of the task's claude reviewer line, then --restricted, the read-only tools,
// dontAsk, no MCP server (--strict-mcp-config and --disallowedTools
// "mcp__*"), the instruction right after --output-format text (--tools and
// --disallowedTools take a list), and RALPH_WIP_CHECKPOINT=off for a hook
// that runs anyway.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeReadsOnly(t *testing.T) {
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		leaderAdvisoryClaudeLead,
		"`--restricted` は設定ファイルとその hook を読まず、`--tools Read,Grep,Glob` は読む道具だけを残し、`--permission-mode dontAsk` は許可の要るものを断り、`--strict-mcp-config` と `--disallowedTools \"mcp__*\"` は MCP の server を立てずその道具も使わせない",
	})
	block := leaderAdvisoryCodeBlock(t, leaderAdvisoryClaudeLead)
	const call = `RALPH_WIP_CHECKPOINT=off command claude -p --model "` + leaderAdvisoryClaudeModel + `" --effort "` + leaderAdvisoryClaudeEffort + `" ` +
		`--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk --output-format text "<指示文>" ` +
		`</dev/null > "$s/plan-advisory-out.md" 2> "$s/plan-advisory.log"` + leaderAdvisoryWatchdog + `claude rc=$rc"`
	if !strings.Contains(block, call) {
		t.Errorf("expected the claude command to contain\n%s\ngot:\n%s", call, block)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryTakesTheReviewerFromTheTask
// pins AC4 of plan 2026-10-30-standard-flow-config (D5) in the template: the
// 計画助言 subsection takes the reviewer's model and effort from the task's
// two reviewer lines and says which line fills which placeholder of the
// commands; a task without the reviewer's line (an org an older ralph
// started) calls nothing and records the advisory as incomplete. The
// template reads no reviewer setting itself: it names neither
// scripts/ralph-config.sh nor `ralph config get reviewer`, none of the four
// variables ralph no longer reads, and no ${RALPH_ expansion but the pick
// command's ${RALPH_PRIMARY_CLI (driver.PrimaryCLIEnv). Each placeholder is
// in its own command once and not in the other. The task featureLeaderTask
// writes has both lines in the form the subsection reads: two words after
// the line's lead and before its parenthesis, a model with a ':' included,
// which the subsection has the leader read after the lead rather than after
// another ':' (self-review L5).
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryTakesTheReviewerFromTheTask(t *testing.T) {
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		"`command -v <相手>` が何も出さなければ飛ばす",
		"相手のモデルと effort は、タスクの `" + featureReviewerCodexLead + "` と `" + featureReviewerClaudeLead + "` の行にある",
		"どちらの行も、行の頭の `" + featureReviewerCodexLead + "`(または `" + featureReviewerClaudeLead + "`)のあと括弧の前に、" +
			"モデルと effort の 2 語がこの順で並ぶ(ralph が org を起こしたときに読んだ `ralph.toml` と `ralph.local.toml` の `[reviewer]` の値)",
		"モデルは `:` を含むことがある(`us.anthropic.claude-x-v1:0` など)ので、行の頭の文字列のあとから読み、ほかの `:` では区切らない",
		"下のコマンドの `" + leaderAdvisoryCodexModel + "` と `" + leaderAdvisoryCodexEffort + "` は `" + featureReviewerCodexLead + "` の行の 2 語に、" +
			"`" + leaderAdvisoryClaudeModel + "` と `" + leaderAdvisoryClaudeEffort + "` は `" + featureReviewerClaudeLead + "` の行の 2 語に書き換えて打つ",
		"タスクに相手の行がなければ(その行を書かない古い ralph が起こした org)、相手を呼ばず、" +
			"計画助言を incomplete(タスクに reviewer の値がない)とし、そのことを下の要約と同じ場所に書く",
		leaderAdvisoryPromptLead,
	})

	data, err := promptFS.ReadFile("prompts/leader.md")
	if err != nil {
		t.Fatal(err)
	}
	template := string(data)
	for _, word := range []string{"ralph-config.sh", "ralph config get reviewer", "RALPH_CLAUDE_REVIEWER_MODEL", "RALPH_CODEX_REVIEWER_MODEL",
		"RALPH_CODEX_REASONING_EFFORT", "RALPH_STANDARD_MAX_PIPELINE_CYCLES"} {
		if strings.Contains(template, word) {
			t.Errorf("expected the leader template to read no reviewer setting itself, but it has %q", word)
		}
	}
	if got := regexp.MustCompile(`\$\{RALPH_[A-Z_]*`).FindAllString(template, -1); !slices.Equal(got, []string{"${" + driver.PrimaryCLIEnv}) {
		t.Errorf("expected the only ${RALPH_ expansion of the leader template to be the pick command's ${%s, got %q", driver.PrimaryCLIEnv, got)
	}

	codex, claude := leaderAdvisoryCodeBlock(t, leaderAdvisoryCodexLead), leaderAdvisoryCodeBlock(t, leaderAdvisoryClaudeLead)
	for _, c := range []struct{ placeholder, own, other string }{
		{leaderAdvisoryCodexModel, codex, claude}, {leaderAdvisoryCodexEffort, codex, claude},
		{leaderAdvisoryClaudeModel, claude, codex}, {leaderAdvisoryClaudeEffort, claude, codex},
	} {
		if strings.Count(c.own, c.placeholder) != 1 || strings.Contains(c.other, c.placeholder) {
			t.Errorf("expected %s once in its own advisory command and not in the other, got:\n%s\n---\n%s", c.placeholder, c.own, c.other)
		}
	}

	task := featureLeaderTask(&SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab"},
		SplitFeature{Slug: "a", Type: "feat", Reserve: []string{"a/"}}, "/state", "/wt", "feat/a", config.ApprovalModePerPlan,
		config.ReviewerConfig{ClaudeModel: "us.anthropic.claude-x-v1:0", ClaudeEffort: "high", CodexModel: "gpt-6-sol", CodexEffort: "medium"})
	if m, e := taskReviewerWords(t, task, featureReviewerCodexLead); m != "gpt-6-sol" || e != "medium" {
		t.Errorf("the task's codex reviewer line reads as %q %q, want gpt-6-sol medium", m, e)
	}
	if m, e := taskReviewerWords(t, task, featureReviewerClaudeLead); m != "us.anthropic.claude-x-v1:0" || e != "high" {
		t.Errorf("the task's claude reviewer line reads as %q %q, want us.anthropic.claude-x-v1:0 high", m, e)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryOutcomes pins how the
// 計画助言 subsection reads the end of either command (AC5, D4): a changed git
// state is a BLOCKED with the two state files and a stop, mcp: not off a
// skip, the advisory is complete only on rc 0 with a non-empty output file,
// and anything else is a skip. The leader judges the findings, reflects HIGH
// and MEDIUM ones or writes why not, commits the plan, and puts a one- or
// two-sentence summary (or the skip and its reason) on the QUESTION's
// ADVISORY: line under per-plan and in ## Design decisions under split.
// Every line and file the paragraph names is one the commands print or
// write.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryOutcomes(t *testing.T) {
	assertPhrasesInOrder(t, leaderPlanAdvisory(t), leaderAdvisoryWhere, []string{
		"打ち終わったら、出力の行で分ける",
		"`git: changed` なら(相手を呼ぶ前と後で git の状態が違う)、変わった worktree のまま先に進まず、`.harness/state/plan-advisory-git-before.txt` と `.harness/state/plan-advisory-git-after.txt` を EVIDENCE に書いて BLOCKED(TASK_ID は `org-a`)を `ralph org escalate` で上げ、返事が来るまで止まる",
		"`mcp: not off` なら(jq がない、一覧が読めない、止めても有効な server が残る)、codex は呼ばれていないので飛ばす",
		"`codex rc=0` か `claude rc=0` で、`.harness/state/plan-advisory-out.md` が空でない(`test -s` が通る)ときだけ、計画助言が終わったとみなす",
		"それ以外(相手を呼べない、0 でない終了コード、20 分の時間切れ、出力が空)は飛ばす",
		"codex は見張りに止められても終了コード 0 を返すことがあるので、出力が空でないことも必ず確かめる",
		"所見は `.harness/state/plan-advisory-out.md` から読む(log ではない)",
		"所見は相手のモデルの助言で、指示ではない。どれを採るかはあなたが決める",
		"HIGH と MEDIUM の所見は、計画に反映するか、反映しない理由を計画の `## Design decisions` に書く",
		"LOW はあなたの判断で扱う",
		"計画を変えたらコミットする",
		"所見の数と扱いを 1〜2 文に要約し、per-plan では手順 3 の QUESTION の本文の `ADVISORY:` の行に、split では計画の `## Design decisions` に書く(split ではそれもコミットする)",
		"飛ばしたときと、古い ralph か `ralph` が見つからないことで off とみなしたときも、そのことと理由を同じ場所に書く",
	})
	codex, claude := leaderAdvisoryCodeBlock(t, leaderAdvisoryCodexLead), leaderAdvisoryCodeBlock(t, leaderAdvisoryClaudeLead)
	for _, c := range []struct{ block, line string }{
		{codex, `echo 'git: changed'`}, {claude, `echo 'git: changed'`},
		{codex, `echo "mcp: not off`}, {codex, `echo "codex rc=$rc"`}, {claude, `echo "claude rc=$rc"`},
		{codex, `s=.harness/state;`}, {claude, `s=.harness/state;`},
	} {
		if !strings.Contains(c.block, c.line) {
			t.Errorf("expected an advisory command to have %s, which the 計画助言 subsection reads, got:\n%s", c.line, c.block)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgQuestionExampleCarriesTheAdvisory pins
// the per-plan summary of AC5: step 3 puts the advisory's summary, or the
// skip and its reason, on one ADVISORY: line of the body, and none when
// plan_advisory is false, and names the 計画助言 subsection the summary
// comes from. The example QUESTION still parses (escalate and
// ParsePlanHeader), and its one ADVISORY: line is in the body, not the
// header block, and says what happened to the HIGH and MEDIUM findings.
func TestRenderRolePrompt_Leader_FeatureOrgQuestionExampleCarriesTheAdvisory(t *testing.T) {
	item, msg, m := leaderQuestionExample(t)
	assertPhrasesInOrder(t, item, leaderItemWhere(leaderStep3Marker), []string{
		"分かれ目がなければ `FORK` の行は書かない",
		"計画助言の要約(この節の終わりの「計画助言」。飛ばしたときはそのことと理由)は、本文の `ADVISORY:` で始まる 1 行に書く(`plan_advisory` が `false` で頼まなかったときは書かない)",
		"たとえば次のメッセージにする",
	})
	if v := protocol.HeaderValues(msg, "ADVISORY"); len(v) != 0 {
		t.Errorf("expected no ADVISORY: header in the example, got %q", v)
	}
	var advisory []string
	for _, line := range strings.Split(m.Body, "\n") {
		if strings.HasPrefix(line, "ADVISORY:") {
			advisory = append(advisory, line)
		}
	}
	if len(advisory) != 1 || !strings.Contains(advisory[0], "HIGH") || !strings.Contains(advisory[0], "MEDIUM") {
		t.Errorf("expected one ADVISORY: line in the example's body naming how the HIGH and MEDIUM findings were handled, got %q in body:\n%s", advisory, m.Body)
	}
}

// advisoryTestEnv is the test's environment without the variables the
// advisory commands read from the pane or git reads as a repository, so a
// test sets the ones it means.
func advisoryTestEnv() []string {
	return slices.DeleteFunc(os.Environ(), func(kv string) bool {
		return strings.HasPrefix(kv, "RALPH_") || strings.HasPrefix(kv, "GIT_")
	})
}

// advisoryCodexStub stands in for codex. It logs each call's arguments, one
// per line and then --end--. Its MCP list has the config servers alpha and
// "has space" (enabled) and gone (disabled), plus the plugin server plug
// unless --disable plugins is given; an override of a server outside the
// config fails the load, as codex 0.163.0 does for a plugin's server.
// STUB_STUCK=1 keeps alpha enabled whatever the overrides. STUB_NOFIELD=1
// adds the config servers bare, whose entry has no enabled field, and blank,
// whose enabled is null, whatever the overrides (a codex whose list does not
// say enabled: false). STUB_OLD=1 knows no feature flag, as codex rejects
// one it does not know (exit 1), and STUB_OLD=apps knows plugins but not
// apps. exec writes a finding to its -o file and
// changes the work tree as STUB_WRITES says (advisoryStubWriteScript);
// STUB_HANG=<seconds> makes it sleep that long instead.
const advisoryCodexStub = `#!/bin/sh
for a in "$@"; do printf '%s\n' "$a"; done >> "$STUB_LOG"
printf '%s\n' '--end--' >> "$STUB_LOG"
off='|' plugins=on out= mode= bad=
while [ $# -gt 0 ]; do
	case $1 in
	--disable)
		if [ "$STUB_OLD" = 1 ] || [ "$STUB_OLD $2" = 'apps apps' ]; then echo "Error: Unknown feature flag: $2" >&2; exit 1; fi
		[ "$2" = plugins ] && plugins=off; shift 2 ;;
	-c)
		case $2 in
		mcp_servers.*.enabled=false)
			n=${2#mcp_servers.}; n=${n%.enabled=false}; off="$off$n|"
			case $n in alpha|'has space'|gone|bare|blank) ;; *) bad=1 ;; esac ;;
		esac
		shift 2 ;;
	-m) shift 2 ;;
	-o) out=$2; shift 2 ;;
	mcp|exec) mode=${mode:-$1}; shift ;;
	*) shift ;;
	esac
done
if [ -n "$bad" ]; then echo 'Error: invalid transport' >&2; exit 1; fi
on() { case $off in *"|$1|"*) echo false ;; *) echo true ;; esac; }
case $mode in
mcp)
	a=$(on alpha); [ "$STUB_STUCK" = 1 ] && a=true
	printf '[{"name":"alpha","enabled":%s},{"name":"has space","enabled":%s},{"name":"gone","enabled":false}' "$a" "$(on 'has space')"
	if [ "$STUB_NOFIELD" = 1 ]; then printf ',{"name":"bare"},{"name":"blank","enabled":null}'; fi
	if [ "$plugins" = on ]; then printf ',{"name":"plug","enabled":true}'; fi
	printf ']\n'
	exit 0 ;;
exec)
	if [ -n "$STUB_HANG" ]; then exec /bin/sleep "$STUB_HANG"; fi
	printf '1. [MEDIUM] stub finding\n' > "$out" ;;
esac
` + advisoryStubWriteScript

// advisoryClaudeStub stands in for claude: it logs its arguments like
// advisoryCodexStub, then the RALPH_WIP_CHECKPOINT it got, prints a finding
// on stdout and a line on stderr, and changes the work tree as STUB_WRITES
// says (advisoryStubWriteScript).
const advisoryClaudeStub = `#!/bin/sh
for a in "$@"; do printf '%s\n' "$a"; done >> "$STUB_LOG"
printf 'RALPH_WIP_CHECKPOINT=%s\n' "$RALPH_WIP_CHECKPOINT" >> "$STUB_LOG"
printf '%s\n' '--end--' >> "$STUB_LOG"
printf '1. [LOW] stub finding\n'
echo 'stub log line' >&2
` + advisoryStubWriteScript

// advisoryStubWriteScript ends both stand-ins' review: it changes the work
// tree as STUB_WRITES names (advisoryWrites) and exits 0.
const advisoryStubWriteScript = `case $STUB_WRITES in
tracked) echo two >> tracked.txt ;;
new) echo one > new.txt ;;
untracked) echo two >> untracked.txt ;;
esac
exit 0
`

// advisoryWrites lists the work tree changes a stand-in reviewer makes with
// STUB_WRITES, each with git status after the run (trimmed, as gitOut
// returns it). An advisory command reports each of them as git: changed.
// untracked edits untracked.txt, which runWriting writes before the run, so
// git status shows the same ?? line before and after it, and only the hashes
// of the untracked files' contents in the git state see the change.
var advisoryWrites = []struct{ name, mode, status string }{
	{"a reviewer that edits", "tracked", "M tracked.txt"},
	{"a reviewer that adds an untracked file", "new", "?? new.txt"},
	{"a reviewer that edits an untracked file", "untracked", "?? untracked.txt"},
}

// advisorySleepStub shortens the watchdog's sleep 1200 to one second and runs
// every other sleep as it is.
const advisorySleepStub = `#!/bin/sh
if [ "$1" = 1200 ]; then exec /bin/sleep 1; fi
exec /bin/sleep "$@"
`

// advisoryFixture is a git repository with an ignored .harness/state/ and no
// scripts/ (the advisory commands source no script), a bin directory with
// the stand-in CLIs, and the stand-ins' call log.
type advisoryFixture struct {
	repo, bin, log string
}

func newAdvisoryFixture(t *testing.T) advisoryFixture {
	t.Helper()
	dir := t.TempDir()
	fx := advisoryFixture{repo: filepath.Join(dir, "repo"), bin: filepath.Join(dir, "bin"), log: filepath.Join(dir, "calls.txt")}
	for name, content := range map[string]string{".gitignore": ".harness/state/\n", "tracked.txt": "one\n"} {
		path := filepath.Join(fx.repo, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// No template: a user's init.templateDir may install commit hooks.
	runGit(t, "-C", fx.repo, "init", "--quiet", "--template=")
	runGit(t, "-C", fx.repo, "add", ".")
	runGit(t, "-C", fx.repo, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "init")
	writeTestExecutable(t, filepath.Join(fx.bin, "codex"), advisoryCodexStub)
	writeTestExecutable(t, filepath.Join(fx.bin, "claude"), advisoryClaudeStub)
	return fx
}

// advisoryRunTimeout bounds one run of an advisory command. A run took up to
// 33.5 seconds with a dozen of them in parallel on a host with a load of
// about 15, so the bound stays well above that.
const advisoryRunTimeout = 120 * time.Second

// run runs block with shell (zsh with -f) in the fixture's repository, the
// stand-ins first on PATH, and returns its combined output. A run still
// going after advisoryRunTimeout is killed with its process group, and the
// test fails.
func (fx advisoryFixture) run(t *testing.T, shell, block string, env ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), advisoryRunTimeout)
	defer cancel()
	args := []string{"-c", block}
	if filepath.Base(shell) == "zsh" {
		args = append([]string{"-f"}, args...)
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Dir = fx.repo
	cmd.Env = append(advisoryTestEnv(), append([]string{"PATH=" + fx.bin + string(os.PathListSeparator) + os.Getenv("PATH"), "STUB_LOG=" + fx.log}, env...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 10 * time.Second
	out, err := cmd.CombinedOutput()
	if err != nil && ctx.Err() != nil {
		t.Fatalf("%s -c <advisory command> was still running after %v and was killed: %v\n%s", filepath.Base(shell), advisoryRunTimeout, err, out)
	}
	if err != nil {
		t.Fatalf("%s -c <advisory command>: %v\n%s", filepath.Base(shell), err, out)
	}
	return string(out)
}

// runWriting runs block like run with a stand-in that makes the work tree
// change mode of advisoryWrites, writing untracked.txt first for untracked,
// and fails the test unless git status after the run is status, so the
// stand-in did make the change. It returns the run's output.
func (fx advisoryFixture) runWriting(t *testing.T, shell, block, mode, status string, env ...string) string {
	t.Helper()
	if mode == "untracked" {
		if err := os.WriteFile(filepath.Join(fx.repo, "untracked.txt"), []byte("one\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := fx.run(t, shell, block, append(slices.Clone(env), "STUB_WRITES="+mode)...)
	if got := gitOut(t, fx.repo, "status", "--porcelain", "--untracked-files=all"); got != status {
		t.Errorf("git status after a run with STUB_WRITES=%s = %q, want %q", mode, got, status)
	}
	return out
}

// calls returns the stand-ins' logged calls, each its arguments (and, for
// claude, the RALPH_WIP_CHECKPOINT line).
func (fx advisoryFixture) calls(t *testing.T) [][]string {
	t.Helper()
	data, err := os.ReadFile(fx.log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	for _, call := range strings.Split(strings.TrimSuffix(string(data), "--end--\n"), "--end--\n") {
		calls = append(calls, strings.Split(strings.TrimSuffix(call, "\n"), "\n"))
	}
	return calls
}

// read returns the file at rel in the fixture's repository, "" when it is
// missing.
func (fx advisoryFixture) read(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fx.repo, rel))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

// advisoryShells returns the paths of bash and zsh, skipping the test when
// neither is installed, or when needJQ (the codex command reads the MCP list
// with jq) and jq is missing.
func advisoryShells(t *testing.T, needJQ bool) []string {
	t.Helper()
	if _, err := exec.LookPath("jq"); needJQ && err != nil {
		t.Skip("jq not available")
	}
	var shells []string
	for _, name := range []string{"bash", "zsh"} {
		if path, err := exec.LookPath(name); err == nil {
			shells = append(shells, path)
		}
	}
	if len(shells) == 0 {
		t.Skip("neither bash nor zsh available")
	}
	return shells
}

// advisoryReviewer is the [reviewer] table the tests that run an advisory
// command give featureLeaderTask; advisoryCommand fills the command's
// placeholders with the words of the reviewer lines it writes, as the leader
// does. Both models hold a ':', as a full model ID may (self-review L5), so a
// command that runs with them shows the words were read after the line's
// lead, not after its last ':'.
var advisoryReviewer = config.ReviewerConfig{ClaudeModel: "us.anthropic.stub-claude-v1:0", ClaudeEffort: "max",
	CodexModel: "stub-model:v1", CodexEffort: "stub-effort"}

// advisoryCommand returns the command after lead with the instruction in
// place of <指示文>, the plan path filled in, and each reviewer placeholder
// replaced with its word of the task's reviewer lines for advisoryReviewer
// (taskReviewerWords), and the instruction. It fails the test when a
// placeholder is left.
func advisoryCommand(t *testing.T, lead string) (block, prompt string) {
	t.Helper()
	prompt = strings.Replace(leaderAdvisoryCodeBlock(t, leaderAdvisoryPromptLead), "<計画のパス>", "docs/plans/active/2026-10-29-x.md", 1)
	block = strings.Replace(leaderAdvisoryCodeBlock(t, lead), "<指示文>", prompt, 1)
	task := featureLeaderTask(&SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab"},
		SplitFeature{Slug: "a", Type: "feat", Reserve: []string{"a/"}}, "/state", "/wt", "feat/a", config.ApprovalModePerPlan, advisoryReviewer)
	codexModel, codexEffort := taskReviewerWords(t, task, featureReviewerCodexLead)
	claudeModel, claudeEffort := taskReviewerWords(t, task, featureReviewerClaudeLead)
	block = strings.NewReplacer(leaderAdvisoryCodexModel, codexModel, leaderAdvisoryCodexEffort, codexEffort,
		leaderAdvisoryClaudeModel, claudeModel, leaderAdvisoryClaudeEffort, claudeEffort).Replace(block)
	if strings.Contains(block, "<reviewer の") {
		t.Fatalf("expected every reviewer placeholder of the command after %q filled in, got:\n%s", lead, block)
	}
	return block, prompt
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexCommandRuns runs the
// codex command of the 計画助言 subsection with bash and zsh against a
// stand-in codex (D7, AC5). With a config server whose name has a space and
// a plugin server, it lists the servers with plugins and apps disabled (D6b
// of plan 2026-11-01-org-cross-review), turns every enabled one off with one
// -c each (the name with a space kept whole), finds none enabled with the
// same arguments, and runs exec with them, read-only,
// --ignore-rules, its -o file under .harness/state/ and the instruction
// last; the output says codex rc=0 and git: same, and the work tree stays
// clean. A server that stays on prints mcp: not off and runs no exec, and so
// do servers whose list entry has no enabled field or a null one, which are
// turned off too but never reported off (self-review L1), a codex that
// rejects --disable plugins as an unknown feature, and one that knows plugins
// but rejects --disable apps; a reviewer that edits
// tracked.txt, adds an untracked file, or edits an untracked file prints
// git: changed; a reviewer that hangs is stopped by the watchdog (sleep 1200
// shortened to one second) with rc 143, the SIGTERM of its kill, and no
// output file. The stand-in hangs for twice advisoryRunTimeout, so a
// watchdog that does not kill it keeps the run going until run kills it and
// fails the test. The cases run in parallel, each in its own repository.
// The model and the effort are the task's (advisoryReviewer, filled in by
// advisoryCommand); the retired variables, set to other values, change
// nothing.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexCommandRuns(t *testing.T) {
	block, prompt := advisoryCommand(t, leaderAdvisoryCodexLead)
	env := []string{"RALPH_CODEX_REVIEWER_MODEL=env-model", "RALPH_CODEX_REASONING_EFFORT=env-effort"}
	for _, shell := range advisoryShells(t, true) {
		name := filepath.Base(shell)
		t.Run(name+"/every server off", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			out := fx.run(t, shell, block, env...)
			if !strings.Contains(out, "codex rc=0\n") || !strings.HasSuffix(out, "git: same\n") || strings.Contains(out, "mcp: not off") {
				t.Errorf("output = %q; want codex rc=0 and git: same", out)
			}
			calls := fx.calls(t)
			if len(calls) != 3 {
				t.Fatalf("codex ran %d times, want 3 (the list, the check, exec): %q", len(calls), calls)
			}
			if want := []string{"--disable", "plugins", "--disable", "apps", "mcp", "list", "--json"}; !slices.Equal(calls[0], want) {
				t.Errorf("first call = %q, want %q", calls[0], want)
			}
			overrides := []string{"--disable", "plugins", "--disable", "apps", "-c", "mcp_servers.alpha.enabled=false", "-c", "mcp_servers.has space.enabled=false"}
			if want := append(slices.Clone(overrides), "mcp", "list", "--json"); !slices.Equal(calls[1], want) {
				t.Errorf("check call = %q, want %q", calls[1], want)
			}
			exe := calls[2]
			if len(exe) < len(overrides) || !slices.Equal(exe[:len(overrides)], overrides) {
				t.Errorf("exec call = %q, want it to start with the overrides %q", exe, overrides)
			}
			at := slices.Index(exe, "exec")
			if at < len(overrides) || !slices.Equal(exe[len(overrides):at], []string{"-m", advisoryReviewer.CodexModel, "-c", "model_reasoning_effort=" + advisoryReviewer.CodexEffort}) {
				t.Errorf("exec call = %q, want -m %s and model_reasoning_effort=%s from the task between the overrides and exec",
					exe, advisoryReviewer.CodexModel, advisoryReviewer.CodexEffort)
			}
			if want := []string{"exec", "--sandbox", "read-only", "--ignore-rules", "-o", ".harness/state/plan-advisory-out.md", prompt}; at < 0 || !slices.Equal(exe[at:], want) {
				t.Errorf("exec call = %q, want it to end with %q", exe, want)
			}
			if got := fx.read(t, ".harness/state/plan-advisory-out.md"); got != "1. [MEDIUM] stub finding\n" {
				t.Errorf("output file = %q, want the stand-in's finding", got)
			}
			if before, after := fx.read(t, ".harness/state/plan-advisory-git-before.txt"), fx.read(t, ".harness/state/plan-advisory-git-after.txt"); before == "" || before != after {
				t.Errorf("git state before = %q, after = %q; want the same non-empty state", before, after)
			}
			if status := gitOut(t, fx.repo, "status", "--porcelain", "--untracked-files=all"); status != "" {
				t.Errorf("git status after the advisory = %q, want a clean tree (its files are under the ignored .harness/state/)", status)
			}
		})
		t.Run(name+"/a server left on", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			out := fx.run(t, shell, block, append(env, "STUB_STUCK=1")...)
			if !strings.Contains(out, "mcp: not off (enabled: 1)\n") || strings.Contains(out, "codex rc=") || !strings.HasSuffix(out, "git: same\n") {
				t.Errorf("output = %q; want mcp: not off (enabled: 1), no codex rc and git: same", out)
			}
			if calls := fx.calls(t); len(calls) != 2 || slices.Contains(calls[len(calls)-1], "exec") {
				t.Errorf("codex ran %q; want the list and the check only", calls)
			}
			if got := fx.read(t, ".harness/state/plan-advisory-out.md"); got != "" {
				t.Errorf("output file = %q, want none", got)
			}
		})
		t.Run(name+"/servers with no enabled field", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			out := fx.run(t, shell, block, append(env, "STUB_NOFIELD=1")...)
			if !strings.Contains(out, "mcp: not off (enabled: 2)\n") || strings.Contains(out, "codex rc=") || !strings.HasSuffix(out, "git: same\n") {
				t.Errorf("output = %q; want mcp: not off (enabled: 2), no codex rc and git: same", out)
			}
			calls := fx.calls(t)
			if len(calls) != 2 || slices.Contains(calls[1], "exec") {
				t.Fatalf("codex ran %q; want the list and the check only", calls)
			}
			for _, server := range []string{"bare", "blank"} {
				if !slices.Contains(calls[1], "mcp_servers."+server+".enabled=false") {
					t.Errorf("check call = %q, want it to turn %s off too: an entry that does not say enabled: false counts as enabled", calls[1], server)
				}
			}
			if got := fx.read(t, ".harness/state/plan-advisory-out.md"); got != "" {
				t.Errorf("output file = %q, want none", got)
			}
		})
		t.Run(name+"/a codex with no plugins feature", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			out := fx.run(t, shell, block, append(env, "STUB_OLD=1")...)
			if !strings.Contains(out, "mcp: not off (enabled: )\n") || strings.Contains(out, "codex rc=") || !strings.HasSuffix(out, "git: same\n") {
				t.Errorf("output = %q; want mcp: not off with no count, no codex rc and git: same", out)
			}
			if calls := fx.calls(t); len(calls) != 2 || slices.Contains(calls[len(calls)-1], "exec") {
				t.Errorf("codex ran %q; want the list and the check only", calls)
			}
		})
		t.Run(name+"/a codex with no apps feature", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			out := fx.run(t, shell, block, append(env, "STUB_OLD=apps")...)
			if !strings.Contains(out, "mcp: not off (enabled: )\n") || strings.Contains(out, "codex rc=") || !strings.HasSuffix(out, "git: same\n") {
				t.Errorf("output = %q; want mcp: not off with no count, no codex rc and git: same", out)
			}
			calls := fx.calls(t)
			if len(calls) != 2 || slices.Contains(calls[len(calls)-1], "exec") {
				t.Fatalf("codex ran %q; want the list and the check only", calls)
			}
			for _, call := range calls {
				if len(call) < 4 || !slices.Equal(call[:4], []string{"--disable", "plugins", "--disable", "apps"}) {
					t.Errorf("codex call = %q, want it to start with --disable plugins --disable apps", call)
				}
			}
		})
		for _, w := range advisoryWrites {
			t.Run(name+"/"+w.name, func(t *testing.T) {
				t.Parallel()
				fx := newAdvisoryFixture(t)
				if out := fx.runWriting(t, shell, block, w.mode, w.status, env...); !strings.Contains(out, "codex rc=0\n") || !strings.HasSuffix(out, "git: changed\n") {
					t.Errorf("output = %q; want codex rc=0 and git: changed", out)
				}
			})
		}
		t.Run(name+"/a reviewer that hangs", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			writeTestExecutable(t, filepath.Join(fx.bin, "sleep"), advisorySleepStub)
			start := time.Now()
			out := fx.run(t, shell, block, append(env, "STUB_HANG="+strconv.Itoa(int(2*advisoryRunTimeout/time.Second)))...)
			if !strings.Contains(out, "codex rc=143\n") || !strings.HasSuffix(out, "git: same\n") {
				t.Errorf("output = %q after %v; want codex rc=143 from the watchdog's kill and git: same", out, time.Since(start))
			}
			if got := fx.read(t, ".harness/state/plan-advisory-out.md"); got != "" {
				t.Errorf("output file = %q, want none", got)
			}
		})
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeCommandRuns runs
// the claude command of the 計画助言 subsection with bash and zsh against a
// stand-in claude (D2, AC5): claude gets the read-only flags in the #266 form
// with the instruction last, right after --output-format text, and
// RALPH_WIP_CHECKPOINT=off over the pane's value; its stdout is the output
// file and its stderr the log; the output says claude rc=0 and git: same, and
// the work tree stays clean. A reviewer that edits tracked.txt, adds an
// untracked file, or edits an untracked file prints claude rc=0 and
// git: changed. The cases run in parallel, each in its own repository.
// The model and the effort are the task's (advisoryReviewer, filled in by
// advisoryCommand); RALPH_CLAUDE_REVIEWER_MODEL, set to another value,
// changes nothing.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeCommandRuns(t *testing.T) {
	block, prompt := advisoryCommand(t, leaderAdvisoryClaudeLead)
	env := []string{"RALPH_CLAUDE_REVIEWER_MODEL=env-claude", "RALPH_WIP_CHECKPOINT=on"}
	for _, shell := range advisoryShells(t, false) {
		name := filepath.Base(shell)
		t.Run(name+"/a clean review", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			out := fx.run(t, shell, block, env...)
			if !strings.Contains(out, "claude rc=0\n") || !strings.HasSuffix(out, "git: same\n") {
				t.Errorf("output = %q; want claude rc=0 and git: same", out)
			}
			calls := fx.calls(t)
			want := []string{"-p", "--model", advisoryReviewer.ClaudeModel, "--effort", advisoryReviewer.ClaudeEffort,
				"--restricted", "--strict-mcp-config", "--disallowedTools", "mcp__*",
				"--tools", "Read,Grep,Glob", "--permission-mode", "dontAsk", "--output-format", "text", prompt, "RALPH_WIP_CHECKPOINT=off"}
			if len(calls) != 1 || !slices.Equal(calls[0], want) {
				t.Errorf("claude calls = %q, want one with %q", calls, want)
			}
			if got := fx.read(t, ".harness/state/plan-advisory-out.md"); got != "1. [LOW] stub finding\n" {
				t.Errorf("output file = %q, want the stand-in's stdout", got)
			}
			if got := fx.read(t, ".harness/state/plan-advisory.log"); got != "stub log line\n" {
				t.Errorf("log file = %q, want the stand-in's stderr", got)
			}
			if status := gitOut(t, fx.repo, "status", "--porcelain", "--untracked-files=all"); status != "" {
				t.Errorf("git status after the advisory = %q, want a clean tree", status)
			}
		})
		for _, w := range advisoryWrites {
			t.Run(name+"/"+w.name, func(t *testing.T) {
				t.Parallel()
				fx := newAdvisoryFixture(t)
				if out := fx.runWriting(t, shell, block, w.mode, w.status, env...); !strings.Contains(out, "claude rc=0\n") || !strings.HasSuffix(out, "git: changed\n") {
					t.Errorf("output = %q; want claude rc=0 and git: changed", out)
				}
			})
		}
	}
}
