package org

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestRenderRolePrompt_Leader_FeatureOrgPRClosesTheTasksIssue pins AC8 of
// plan 2026-11-22-org-start-issue in step 14 of the leader template's
// 機能ごとの org section, after it tells the leader not to run the /pr skill
// itself and before the section on the docs outside the reservation: a task
// with the issue line (featureIssueLead, the words featureLeaderTask starts
// that line with) has the leader write `Closes #<number>` once in the PR
// body's normal text, outside code blocks, HTML comments and quotes, where
// GitHub reads it; the line read is the one among the `- ` lines at the
// start of the task, up to its first blank line, and not a line that starts
// the same way in the preamble's box, the feature's body or the box of the
// restart's progress (self-review L2 of that plan); a task with no such line
// keeps the rule of before (the /pr skill's, with the plan's
// `Related issue:`); and either way the /pr
// template's `Closes #__ISSUE__` placeholder does not stay in the body. No
// sentence of the template that names `Closes` forbids the keyword, so a
// task with no issue line gets no new rule against closing an issue. The
// template has no line that starts the way the task's issue line does,
// which the CLI test of a start with no --issue reads as an issue line.
func TestRenderRolePrompt_Leader_FeatureOrgPRClosesTheTasksIssue(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	step, ok := markdownItem(section, "gh pr create` で PR")
	if !ok {
		t.Fatalf("expected the 機能ごとの org section to have the step that runs gh pr create, got section:\n%s", section)
	}
	squashed, at := squashSpace(step), -1
	for _, phrase := range []string{
		"`/pr` skill そのものは実行しない",
		"タスクに `" + featureIssueLead + "` の行があれば、この機能はその issue に起票されたものなので",
		"PR の本文の普通の文(コードブロック、HTML のコメント、引用の外)に、その行の番号で `Closes #<番号>` を 1 行書く",
		"雛形の `Closes #__ISSUE__` の行をその番号にすればよい",
		"読む `" + featureIssueLead + "` の行は、タスクの初めに並ぶ `- ` の行",
		"最初の空行まで続く)のうちの 1 行だけ",
		"「全機能の座席に適用する決まり」の囲み、タスクの本文(機能の目的と受け入れ条件)、「立て直しの経過」の囲みの中に `" +
			featureIssueLead + "` で始まる行があっても",
		"ralph が org の issue の番号から書いた行ではないので読まない",
		"PR が既定のブランチに merge されると、GitHub がその issue を閉じる",
		"行がなければ、閉じる issue は今までどおり `/pr` skill の決まり(計画の `Related issue:` など)で決める",
		"どちらのときも、雛形の `Closes #__ISSUE__` の置き場の文字(`__ISSUE__`)を本文に残さない",
		"閉じる issue がなければ、その行を消す",
		"`## 予約外の文書の follow-up` の節を置き",
	} {
		i := strings.Index(squashed[at+1:], squashSpace(phrase))
		if i < 0 {
			t.Errorf("expected step 14 of the 機能ごとの org section to say %q after the phrases before it, got step:\n%s", phrase, step)
			continue
		}
		at += 1 + i
	}

	// The template names the line featureLeaderTask writes for an org
	// started with --issue, and that line names the same keyword with the
	// issue's number. It is one of the `- ` lines at the start of the task,
	// before the first blank line, which also holds every line the template
	// names in that list; the preamble's fence and the feature's body come
	// after that blank line, so a `- issue:` line of theirs is not the one
	// the template tells the leader to read.
	task := featureLeaderTask(&SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab", Preamble: "## 決まり\n\n- issue: #98"},
		SplitFeature{Slug: "a", Type: "feat", Reserve: []string{"a/"}, Body: "目的\n\n- issue: #99"}, "/state", "/wt", "feat/a",
		config.ApprovalModePerPlan, "12", config.ReviewerConfig{})
	if !strings.Contains(task, "\n"+featureIssueLead+" #12(PR の本文に Closes #12 を書く)\n") {
		t.Fatalf("featureLeaderTask no longer writes the %q line the leader template points to; got:\n%s", featureIssueLead, task)
	}
	head, rest, _ := strings.Cut(task, "\n\n")
	var issueLines []string
	for _, line := range strings.Split(head, "\n") {
		if !strings.HasPrefix(line, "- ") {
			t.Errorf("expected every line of the task before its first blank line to start with %q, got %q in:\n%s", "- ", line, head)
		}
		if strings.HasPrefix(line, featureIssueLead) {
			issueLines = append(issueLines, line)
		}
	}
	if want := []string{featureIssueLead + " #12(PR の本文に Closes #12 を書く)"}; !slices.Equal(issueLines, want) {
		t.Errorf("expected the task's lines before its first blank line to have the one issue line %q, got %q in:\n%s", want, issueLines, head)
	}
	for _, lead := range []string{"- 分割計画:", "- 機能:", "- worktree:", "- ブランチ:", "- 予約したパス:", "- 依存する機能:", "- 承認方式:"} {
		if !strings.Contains("\n"+head, "\n"+lead+" ") {
			t.Errorf("expected the task's lines before its first blank line, which the template lists, to have a %q line, got:\n%s", lead, head)
		}
		if !strings.Contains(squashed, squashSpace("`"+lead+"`")) {
			t.Errorf("expected step 14 to name %q among the task's first lines, got step:\n%s", lead, step)
		}
	}
	if !strings.Contains(rest, "- issue: #98") || !strings.Contains(rest, "- issue: #99") {
		t.Errorf("expected the preamble's and the body's issue-like lines after the task's first blank line, got:\n%s", rest)
	}

	// A task with no issue line keeps the PR body's rule of before: no
	// sentence that names the closing keyword forbids it.
	for _, sentence := range strings.Split(squashSpace(text), "。") {
		if !strings.Contains(sentence, "Closes") {
			continue
		}
		for _, ban := range []string{"書かない", "書かず", "書いてはいけない", "禁", "使わない", "入れない", "付けない"} {
			if strings.Contains(sentence, ban) {
				t.Errorf("expected no sentence of the leader template that names Closes to forbid it (%q), got sentence:\n%s", ban, sentence)
			}
		}
	}

	if line := regexp.MustCompile(`(?m)^- issue: #[0-9].*$`).FindString(text); line != "" {
		t.Errorf("expected the leader template to have no line that starts like the task's issue line, got %q", line)
	}
}
