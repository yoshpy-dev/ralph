package org

import (
	"regexp"
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
// GitHub reads it; a task with no such line keeps the rule of before (the
// /pr skill's, with the plan's `Related issue:`); and either way the /pr
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
	// issue's number.
	task := featureLeaderTask(&SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab"},
		SplitFeature{Slug: "a", Type: "feat", Reserve: []string{"a/"}}, "/state", "/wt", "feat/a", config.ApprovalModePerPlan, "12", config.ReviewerConfig{})
	if !strings.Contains(task, "\n"+featureIssueLead+" #12(PR の本文に Closes #12 を書く)\n") {
		t.Fatalf("featureLeaderTask no longer writes the %q line the leader template points to; got:\n%s", featureIssueLead, task)
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
