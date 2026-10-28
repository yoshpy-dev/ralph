package org

import (
	"context"
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

	"github.com/yoshpy-dev/ralph/internal/org/driver"
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

// The plan advisory of plan 2026-10-29-leader-plan-depth (AC4, AC5, S2) is
// the last part of step 1 of the 機能ごとの org section. The leads below are
// the sentences each of its three code blocks follows.
const (
	leaderStep1Marker        = "1. 機能の計画を"
	leaderAdvisoryPromptLead = "相手には次の指示文を渡す"
	leaderAdvisoryCodexLead  = "相手が codex なら次のコマンドを打つ"
	leaderAdvisoryClaudeLead = "相手が claude なら次のコマンドを打つ"
)

// leaderAdvisoryPickCommand is the command step 1 picks the other CLI with:
// pick_reviewer reads the driver ralph puts in the pane.
const leaderAdvisoryPickCommand = `. ./scripts/xreview-helpers.sh && pick_reviewer "$` + driver.PrimaryCLIEnv + `"`

// leaderAdvisoryGitState is the command both advisory commands write the git
// state with before and after the call: the form of /plan step 11 of #266
// (HEAD, the index, the work tree with its untracked files' contents).
const leaderAdvisoryGitState = "{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; " +
	"git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; }"

// leaderAdvisoryWatchdog is the 20-minute watchdog of /plan step 11.c both
// advisory commands run the reviewer under, up to the echo of its rc.
const leaderAdvisoryWatchdog = ` & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & ` +
	`wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "`

// leaderStep1CodeBlock returns the code block of step 1 of the 機能ごとの org
// section that comes first after the line holding lead, with the item's
// indentation removed.
func leaderStep1CodeBlock(t *testing.T, lead string) string {
	t.Helper()
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	at := strings.Index(item, lead)
	if at < 0 {
		t.Fatalf("expected step 1 to have the sentence %q, got item:\n%s", lead, item)
	}
	const fence = "\n   ```\n"
	_, rest, opened := strings.Cut(item[at:], fence)
	block, _, closed := strings.Cut(rest, fence)
	if !opened || !closed {
		t.Fatalf("expected an indented code block after %q in step 1, got item:\n%s", lead, item)
	}
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, "   ")
	}
	return strings.Join(lines, "\n")
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

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryReadsTheSwitchFirst pins
// AC4 of plan 2026-10-29-leader-plan-depth: after the forks paragraph, step 1
// asks the other model for a plan advisory only when
// [org.review].plan_advisory is on, before step 2 under per-plan and before
// the implementer's spawn of step 5 under split (D6), once. The leader reads
// the switch with ralph config get and sorts its exit code the way that
// verb's help promises (D5): 0 uses the value, false calls nothing; an older
// ralph's unknown command "config" is off and recorded; any other code is a
// BLOCKED, never off. The help in internal/cli/config.go still promises those
// codes.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryReadsTheSwitchFirst(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderStep1Marker, []string{
		"split では人に聞かずに推奨の案を採り",
		"計画をコミットしたら、`[org.review].plan_advisory` が on のときだけ、相手のモデルに計画助言を頼む(claude の leader は codex に、codex の leader は claude に、計画を読むだけの形で)",
		"per-plan では手順 2 の前に、split では手順 5 で implementer を spawn する前に済ませる",
		"頼むのは 1 回で、手順 4 と 6 で計画を直して手順 2 からやり直すときは頼み直さない",
		"まず worktree で `ralph config get org.review.plan_advisory` を打ち、終了コードで分ける(`ralph config get --help` の約束)",
		"0 なら出た値を使い、`true` なら頼み、`false` なら頼まずに次へ進む(記録も要らない)",
		"1 で stderr に `unknown command \"config\"` があれば(この動詞のない古い ralph)、off とみなして頼まず、off とみなしたことを下の要約と同じ場所に書く",
		"それ以外(ほかの 1、2、3、0 で `true` でも `false` でもない値)は off とみなさず、エラーを本文に書いて BLOCKED(TASK_ID は `org-a`)を「件を上げる」の節の手順で `ralph org escalate` で上げ、返事が来るまで先に進まない(待ち方は「agmsg の受信箱の運用」の節)",
		"頼むときは、worktree に",
	})

	help := readRepoText(t, filepath.Join("internal", "cli", "config.go"))
	for _, promise := range []string{
		`"  0  the value was printed: use it.\n"`,
		`unknown command \"config\" on stderr. Treat that, and only that, as\n`,
		`any other exit 1.`,
		`"  2  the key is not a key of ralph.toml`,
		`"  3  ralph.toml cannot be read`,
	} {
		if !strings.Contains(help, promise) {
			t.Errorf("internal/cli/config.go no longer has %s in the help of ralph config get, whose exit codes step 1 of the leader template sorts", promise)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryPicksTheOtherCLI pins the
// reviewer choice of AC5: step 1 skips when the worktree has no
// scripts/xreview-helpers.sh or scripts/ralph-config.sh, picks the reviewer
// with pick_reviewer on the RALPH_PRIMARY_CLI ralph puts in the pane, and
// skips when that CLI is not on PATH. The command, run with bash and zsh from
// the repository root on its scripts/xreview-helpers.sh, picks codex for a
// claude pane, claude for a codex pane, and codex when the pane has no
// RALPH_PRIMARY_CLI (the edge case of the plan's test plan).
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryPicksTheOtherCLI(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderStep1Marker, []string{
		"頼むときは、worktree に `scripts/xreview-helpers.sh` と `scripts/ralph-config.sh` があることを確かめ(どちらかがなければ飛ばす)",
		"`" + leaderAdvisoryPickCommand + "` が出す名前(`codex` か `claude`)を相手にする(ralph は pane に `" + driver.PrimaryCLIEnv + "` を入れる)",
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
		for _, tc := range []struct{ pane, want string }{
			{driver.PrimaryCLIEntry("claude"), "codex"},
			{driver.PrimaryCLIEntry("codex"), "claude"},
			{"", "codex"},
		} {
			args := []string{"-c", leaderAdvisoryPickCommand}
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
				t.Errorf("%s -c %q with pane env %q = %q, err %v; want %q", shell, leaderAdvisoryPickCommand, tc.pane, got, err, tc.want)
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
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderStep1Marker, []string{
		"相手には次の指示文を渡す(下のコマンドの `<指示文>`)",
		"英語のまま変えず、最後の `<計画のパス>` だけを worktree からの計画の相対パスにする",
	})
	prompt := leaderStep1CodeBlock(t, leaderAdvisoryPromptLead)
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
// step 1 runs either command: in one shell call, under run_in_background with
// the turn ended in Claude Code and in the foreground with a tool timeout
// over 20 minutes in Codex; each stops the reviewer after 20 minutes,
// compares the git state from before and after the call, and writes its files
// under the ignored .harness/state/ so the gate does not refuse a dirty tree.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryRunsInOneShell(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderStep1Marker, []string{
		"相手ごとに下のコマンドのどちらかを、まとめて 1 回の shell で打つ(Bash ツールは shell の状態を次の呼び出しに持ち越さない)",
		"Bash ツールの `run_in_background` があるとき(Claude Code)は `run_in_background` で走らせて手番を終え、終わりの知らせが来てから出力を読む",
		"ないとき(Codex)は前面で打ち、ツールの timeout を 20 分より長くする(そのあいだ `ralph org send` は届かない)",
		"どちらのコマンドも、`sleep 1200` の見張りで相手を 20 分で止め、呼ぶ前と後の git の状態(HEAD、index、未追跡のファイルを含む作業ツリー)をファイルに書いて `cmp` で比べる",
		"ファイルは worktree の `.harness/state/` の下に書く(git が無視する場所なので、手順 12 の gate が作業ツリーの汚れとして断らない)",
		leaderAdvisoryCodexLead,
	})
	for _, lead := range []string{leaderAdvisoryCodexLead, leaderAdvisoryClaudeLead} {
		block := leaderStep1CodeBlock(t, lead)
		lines := strings.Split(block, "\n")
		const head = `. ./scripts/ralph-config.sh; s=.harness/state; mkdir -p "$s"; rm -f "$s/plan-advisory-out.md"`
		before := leaderAdvisoryGitState + ` > "$s/plan-advisory-git-before.txt"`
		after := leaderAdvisoryGitState + ` > "$s/plan-advisory-git-after.txt"`
		const compare = `if cmp -s "$s/plan-advisory-git-before.txt" "$s/plan-advisory-git-after.txt"; then echo 'git: same'; else echo 'git: changed'; fi`
		if n := len(lines); n < 5 || lines[0] != head || lines[1] != before || lines[n-2] != after || lines[n-1] != compare {
			t.Errorf("expected the command after %q to source ralph-config.sh and write the git state first, and to write it again and cmp the two last, got:\n%s", lead, block)
		}
		if strings.Count(block, leaderAdvisoryWatchdog) != 1 || strings.Count(block, "</dev/null") != 1 || strings.Count(block, `"<指示文>"`) != 1 {
			t.Errorf("expected the command after %q to run the reviewer once, with stdin closed, the instruction, and the 20-minute watchdog, got:\n%s", lead, block)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexTurnsMCPOff pins the
// codex command of AC5 and D7: it disables plugins, whose MCP servers
// -c mcp_servers.<name>.enabled=false cannot turn off (an older codex that
// cannot read the list with it skips), reads the names of the
// enabled servers one line at a time into -c overrides (no word splitting),
// checks with the same arguments that no server is enabled, and only then
// runs codex exec with them, read-only and with --ignore-rules, its findings
// in -o; otherwise it says mcp: not off and calls nothing. The model and
// effort fall back to the defaults of scripts/ralph-config.sh.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexTurnsMCPOff(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderStep1Marker, []string{
		leaderAdvisoryCodexLead,
		"`--sandbox read-only` と `--ignore-rules` は MCP の server に及ばないので、このコマンドは MCP の server を止めてから codex を呼ぶ",
		"plugin が持ち込む MCP の server は `-c mcp_servers.<名前>.enabled=false` では止まらない(codex が設定を読めなくなる)ので、`--disable plugins` で plugin ごと外す(`plugins` の機能名を知らない古い codex で一覧が読めなければ、`mcp: not off` になり、codex を呼ばずに飛ばす)",
		"`codex --disable plugins mcp list --json` で有効な server の名前を読み、1 行ずつ `-c mcp_servers.<名前>.enabled=false` にして引数に足し(shell の語の分け方に頼らない)、同じ引数を付けた一覧で有効な server が 0 のときだけ codex を呼ぶ",
		leaderAdvisoryClaudeLead,
	})
	block := leaderStep1CodeBlock(t, leaderAdvisoryCodexLead)
	for _, want := range []string{
		`command codex --disable plugins mcp list --json | jq -r '.[] | select(.enabled) | .name' > "$s/plan-advisory-mcp.txt"`,
		`set -- --disable plugins; while IFS= read -r name; do [ -z "$name" ] || set -- "$@" -c "mcp_servers.$name.enabled=false"; done < "$s/plan-advisory-mcp.txt"`,
		`n=$(command codex "$@" mcp list --json | jq '[.[] | select(.enabled)] | length')`,
		`if [ "$n" = 0 ]; then command codex "$@" -m "${RALPH_CODEX_REVIEWER_MODEL:-`,
		`-c "model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-`,
		` exec --sandbox read-only --ignore-rules -o "$s/plan-advisory-out.md" "<指示文>" </dev/null > "$s/plan-advisory.log" 2>&1` + leaderAdvisoryWatchdog + `codex rc=$rc"; else echo "mcp: not off (enabled: $n)"; fi`,
	} {
		if !strings.Contains(block, want) {
			t.Errorf("expected the codex command to contain %s, got:\n%s", want, block)
		}
	}
	// The fallbacks are ralph-config.sh's defaults, which the command sources.
	config := readRepoText(t, filepath.Join("scripts", "ralph-config.sh"))
	for _, name := range []string{"RALPH_CODEX_REVIEWER_MODEL", "RALPH_CODEX_REASONING_EFFORT"} {
		m := regexp.MustCompile(name + `="\$\{` + name + `:-([^}]*)\}"`).FindStringSubmatch(config)
		if m == nil || !strings.Contains(block, "${"+name+":-"+m[1]+"}") {
			t.Errorf("expected the codex command's fallback for %s to be scripts/ralph-config.sh's default (%v), got:\n%s", name, m, block)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeReadsOnly pins the
// claude command of AC5 and D2, the form #266 settled on: claude -p with the
// reviewer model, --restricted, the read-only tools, dontAsk, no MCP server
// (--strict-mcp-config and --disallowedTools "mcp__*"), the instruction right
// after --output-format text (--tools and --disallowedTools take a list), and
// RALPH_WIP_CHECKPOINT=off for a hook that runs anyway.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeReadsOnly(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderStep1Marker, []string{
		leaderAdvisoryClaudeLead,
		"`--restricted` は設定ファイルとその hook を読まず、`--tools Read,Grep,Glob` は読む道具だけを残し、`--permission-mode dontAsk` は許可の要るものを断り、`--strict-mcp-config` と `--disallowedTools \"mcp__*\"` は MCP の server を立てずその道具も使わせない",
	})
	block := leaderStep1CodeBlock(t, leaderAdvisoryClaudeLead)
	const call = `RALPH_WIP_CHECKPOINT=off command claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --restricted --strict-mcp-config ` +
		`--disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk --output-format text "<指示文>" ` +
		`</dev/null > "$s/plan-advisory-out.md" 2> "$s/plan-advisory.log"` + leaderAdvisoryWatchdog + `claude rc=$rc"`
	if !strings.Contains(block, call) {
		t.Errorf("expected the claude command to contain\n%s\ngot:\n%s", call, block)
	}
	config := readRepoText(t, filepath.Join("scripts", "ralph-config.sh"))
	if !strings.Contains(config, `RALPH_CLAUDE_REVIEWER_MODEL="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"`) {
		t.Errorf("expected scripts/ralph-config.sh's default for RALPH_CLAUDE_REVIEWER_MODEL to be the claude command's fallback opus")
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryOutcomes pins how step 1
// reads the end of either command (AC5, D4): a changed git state is a BLOCKED
// with the two state files and a stop, mcp: not off a skip, the advisory is
// complete only on rc 0 with a non-empty output file, and anything else is a
// skip. The leader judges the findings, reflects HIGH and MEDIUM ones or
// writes why not, commits the plan, and puts a one- or two-sentence summary
// (or the skip and its reason) on the QUESTION's ADVISORY: line under
// per-plan and in ## Design decisions under split. Every line and file the
// paragraph names is one the commands print or write.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryOutcomes(t *testing.T) {
	item := leaderFeatureOrgItem(t, leaderStep1Marker)
	assertPhrasesInOrder(t, item, leaderStep1Marker, []string{
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
		"飛ばしたときと、古い ralph で off とみなしたときも、そのことと理由を同じ場所に書く",
	})
	codex, claude := leaderStep1CodeBlock(t, leaderAdvisoryCodexLead), leaderStep1CodeBlock(t, leaderAdvisoryClaudeLead)
	for _, c := range []struct{ block, line string }{
		{codex, `echo 'git: changed'`}, {claude, `echo 'git: changed'`},
		{codex, `echo "mcp: not off`}, {codex, `echo "codex rc=$rc"`}, {claude, `echo "claude rc=$rc"`},
		{codex, `s=.harness/state;`}, {claude, `s=.harness/state;`},
	} {
		if !strings.Contains(c.block, c.line) {
			t.Errorf("expected an advisory command to have %s, which step 1 reads, got:\n%s", c.line, c.block)
		}
	}
}

// leaderQuestionExample returns the example QUESTION of step 3 as escalate
// reads it: the code block of the item, with its indentation removed.
func leaderQuestionExample(t *testing.T) (item, msg string) {
	t.Helper()
	item = leaderFeatureOrgItem(t, "3. (per-plan)")
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
	return item, strings.Join(lines, "\n")
}

// TestRenderRolePrompt_Leader_FeatureOrgQuestionExampleCarriesTheAdvisory pins
// the per-plan summary of AC5: step 3 puts the advisory's summary, or the
// skip and its reason, on one ADVISORY: line of the body, and none when
// plan_advisory is false. The example QUESTION still parses (escalate and
// ParsePlanHeader), and its one ADVISORY: line is in the body, not the header
// block, and says what happened to the HIGH and MEDIUM findings.
func TestRenderRolePrompt_Leader_FeatureOrgQuestionExampleCarriesTheAdvisory(t *testing.T) {
	const marker = "3. (per-plan)"
	item, msg := leaderQuestionExample(t)
	assertPhrasesInOrder(t, item, marker, []string{
		"分かれ目がなければ `FORK` の行は書かない",
		"手順 1 の計画助言の要約(飛ばしたときはそのことと理由)は、本文の `ADVISORY:` で始まる 1 行に書く(`plan_advisory` が `false` で頼まなかったときは書かない)",
		"たとえば次のメッセージにする",
	})
	m, err := validateEscalation("org-a", msg)
	if err != nil || m.Type != protocol.TypeQuestion {
		t.Fatalf("expected escalate to accept the example as a QUESTION, got TYPE %q, err %v; message:\n%s", m.Type, err, msg)
	}
	if _, err := ParsePlanHeader(msg); err != nil {
		t.Errorf("ParsePlanHeader(example QUESTION) = %v; message:\n%s", err, msg)
	}
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

// TestRenderRolePrompt_Leader_PlanAdvisoryStaysInTheFeatureOrgSection pins
// that the plan advisory is part of the 機能ごとの org section only: an org
// that is not a split plan's feature follows its task.
func TestRenderRolePrompt_Leader_PlanAdvisoryStaysInTheFeatureOrgSection(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	outside := strings.Replace(text, section, "", 1)
	for _, word := range []string{"plan_advisory", "計画助言", "pick_reviewer", "adversarial plan reviewer", "mcp_servers", "--disable plugins", "claude -p", "plan-advisory", "ADVISORY:"} {
		if !strings.Contains(section, word) {
			t.Errorf("expected the 機能ごとの org section to contain %q, got section:\n%s", word, section)
		}
		if strings.Contains(outside, word) {
			t.Errorf("expected %q only in the 機能ごとの org section, but the rest of the template has it:\n%s", word, outside)
		}
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
// STUB_STUCK=1 keeps alpha enabled whatever the overrides. STUB_OLD=1 knows
// no feature flag, as codex rejects one it does not know (exit 1). exec
// writes a finding to its -o file; STUB_WRITES=1 also changes tracked.txt,
// and STUB_HANG=1 makes it hang instead.
const advisoryCodexStub = `#!/bin/sh
for a in "$@"; do printf '%s\n' "$a"; done >> "$STUB_LOG"
printf '%s\n' '--end--' >> "$STUB_LOG"
off='|' plugins=on out= mode= bad=
while [ $# -gt 0 ]; do
	case $1 in
	--disable)
		if [ "$STUB_OLD" = 1 ]; then echo "Error: Unknown feature flag: $2" >&2; exit 1; fi
		[ "$2" = plugins ] && plugins=off; shift 2 ;;
	-c)
		case $2 in
		mcp_servers.*.enabled=false)
			n=${2#mcp_servers.}; n=${n%.enabled=false}; off="$off$n|"
			case $n in alpha|'has space'|gone) ;; *) bad=1 ;; esac ;;
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
	if [ "$plugins" = on ]; then printf ',{"name":"plug","enabled":true}'; fi
	printf ']\n' ;;
exec)
	if [ "$STUB_HANG" = 1 ]; then exec /bin/sleep 30; fi
	printf '1. [MEDIUM] stub finding\n' > "$out"
	if [ "$STUB_WRITES" = 1 ]; then echo two >> tracked.txt; fi ;;
esac
exit 0
`

// advisoryClaudeStub stands in for claude: it logs its arguments like
// advisoryCodexStub, then the RALPH_WIP_CHECKPOINT it got, and prints a
// finding on stdout and a line on stderr.
const advisoryClaudeStub = `#!/bin/sh
for a in "$@"; do printf '%s\n' "$a"; done >> "$STUB_LOG"
printf 'RALPH_WIP_CHECKPOINT=%s\n' "$RALPH_WIP_CHECKPOINT" >> "$STUB_LOG"
printf '%s\n' '--end--' >> "$STUB_LOG"
printf '1. [LOW] stub finding\n'
echo 'stub log line' >&2
`

// advisorySleepStub shortens the watchdog's sleep 1200 to one second and runs
// every other sleep as it is.
const advisorySleepStub = `#!/bin/sh
if [ "$1" = 1200 ]; then exec /bin/sleep 1; fi
exec /bin/sleep "$@"
`

// advisoryFixture is a git repository with the shipped scripts/ralph-config.sh
// and an ignored .harness/state/, a bin directory with the stand-in CLIs, and
// the stand-ins' call log.
type advisoryFixture struct {
	repo, bin, log string
}

func newAdvisoryFixture(t *testing.T) advisoryFixture {
	t.Helper()
	config := readRepoText(t, filepath.Join("scripts", "ralph-config.sh"))
	dir := t.TempDir()
	fx := advisoryFixture{repo: filepath.Join(dir, "repo"), bin: filepath.Join(dir, "bin"), log: filepath.Join(dir, "calls.txt")}
	for name, content := range map[string]string{".gitignore": ".harness/state/\n", "tracked.txt": "one\n", "scripts/ralph-config.sh": config} {
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

// run runs block with shell (zsh with -f) in the fixture's repository, the
// stand-ins first on PATH, and returns its combined output. A run still
// going after 60 seconds is killed with its process group.
func (fx advisoryFixture) run(t *testing.T, shell, block string, env ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	if err != nil {
		t.Fatalf("%s -c <advisory command>: %v\n%s", filepath.Base(shell), err, out)
	}
	return string(out)
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

// advisoryCommand returns the command after lead with the instruction in
// place of <指示文>, the plan path filled in, and the instruction.
func advisoryCommand(t *testing.T, lead string) (block, prompt string) {
	t.Helper()
	prompt = strings.Replace(leaderStep1CodeBlock(t, leaderAdvisoryPromptLead), "<計画のパス>", "docs/plans/active/2026-10-29-x.md", 1)
	return strings.Replace(leaderStep1CodeBlock(t, lead), "<指示文>", prompt, 1), prompt
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexCommandRuns runs the
// codex command of step 1 with bash and zsh against a stand-in codex (D7,
// AC5). With a config server whose name has a space and a plugin server, it
// lists the servers with plugins disabled, turns every enabled one off with
// one -c each (the name with a space kept whole), finds none enabled with
// the same arguments, and runs exec with them, read-only, --ignore-rules,
// its -o file under .harness/state/ and the instruction last; the output
// says codex rc=0 and git: same, and the work tree stays clean. A server
// that stays on prints mcp: not off and runs no exec, and so does a codex
// that rejects --disable plugins as an unknown feature; a reviewer that edits
// tracked.txt prints git: changed; a reviewer that hangs is stopped by the
// watchdog (sleep 1200 shortened to one second) with a non-zero rc and no
// output file. The cases run in parallel, each in its own repository.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexCommandRuns(t *testing.T) {
	block, prompt := advisoryCommand(t, leaderAdvisoryCodexLead)
	env := []string{"RALPH_CODEX_REVIEWER_MODEL=stub-model", "RALPH_CODEX_REASONING_EFFORT=stub-effort"}
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
			if want := []string{"--disable", "plugins", "mcp", "list", "--json"}; !slices.Equal(calls[0], want) {
				t.Errorf("first call = %q, want %q", calls[0], want)
			}
			overrides := []string{"--disable", "plugins", "-c", "mcp_servers.alpha.enabled=false", "-c", "mcp_servers.has space.enabled=false"}
			if want := append(slices.Clone(overrides), "mcp", "list", "--json"); !slices.Equal(calls[1], want) {
				t.Errorf("check call = %q, want %q", calls[1], want)
			}
			exe := calls[2]
			if len(exe) < len(overrides) || !slices.Equal(exe[:len(overrides)], overrides) {
				t.Errorf("exec call = %q, want it to start with the overrides %q", exe, overrides)
			}
			at := slices.Index(exe, "exec")
			if at < 0 || !slices.Contains(exe[:at], "stub-model") || !slices.Contains(exe[:at], "model_reasoning_effort=stub-effort") {
				t.Errorf("exec call = %q, want -m stub-model and the effort before exec", exe)
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
		t.Run(name+"/a reviewer that edits", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			if out := fx.run(t, shell, block, append(env, "STUB_WRITES=1")...); !strings.Contains(out, "codex rc=0\n") || !strings.HasSuffix(out, "git: changed\n") {
				t.Errorf("output = %q; want codex rc=0 and git: changed", out)
			}
		})
		t.Run(name+"/a reviewer that hangs", func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			writeTestExecutable(t, filepath.Join(fx.bin, "sleep"), advisorySleepStub)
			start := time.Now()
			out := fx.run(t, shell, block, append(env, "STUB_HANG=1")...)
			if strings.Contains(out, "codex rc=0") || !strings.Contains(out, "codex rc=") || !strings.HasSuffix(out, "git: same\n") || time.Since(start) > 20*time.Second {
				t.Errorf("output = %q after %v; want a non-zero codex rc from the watchdog, soon, and git: same", out, time.Since(start))
			}
			if got := fx.read(t, ".harness/state/plan-advisory-out.md"); got != "" {
				t.Errorf("output file = %q, want none", got)
			}
		})
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeCommandRuns runs
// the claude command of step 1 with bash and zsh against a stand-in claude
// (D2, AC5): claude gets the read-only flags in the #266 form with the
// instruction last, right after --output-format text, and
// RALPH_WIP_CHECKPOINT=off over the pane's value; its stdout is the output
// file and its stderr the log; the output says claude rc=0 and git: same, and
// the work tree stays clean.
func TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeCommandRuns(t *testing.T) {
	block, prompt := advisoryCommand(t, leaderAdvisoryClaudeLead)
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newAdvisoryFixture(t)
			out := fx.run(t, shell, block, "RALPH_CLAUDE_REVIEWER_MODEL=stub-claude", "RALPH_WIP_CHECKPOINT=on")
			if !strings.Contains(out, "claude rc=0\n") || !strings.HasSuffix(out, "git: same\n") {
				t.Errorf("output = %q; want claude rc=0 and git: same", out)
			}
			calls := fx.calls(t)
			want := []string{"-p", "--model", "stub-claude", "--restricted", "--strict-mcp-config", "--disallowedTools", "mcp__*",
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
	}
}
