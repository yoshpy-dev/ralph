package org

// TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase ties the /director
// skill's inbox section to WaitInbox's timeout error. The skill tells a
// timeout of a bounded `ralph org wait --inbox` from a failure by finding
// inboxWaitTimeoutPhrase in stderr, and reruns the wait only on a match, so
// a change to one text without the other makes the director stop at every
// timeout. scripts/check-skill-sync.sh keeps the four copies of the skill
// identical but does not read Go, so nothing else catches that drift. It
// follows send_defaults_sync_test.go's repo-root-via-runtime.Caller pattern.
//
// If this test fails, either restore inboxWaitTimeoutPhrase in escalate.go,
// or write the new phrase, in backquotes, in the inbox section
// (受信箱の処理) of .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies (root, then with
// CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills).

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// directorSkillRepoRoot resolves the repository root from this source file's
// location: internal/org/ -> ../../ (as sendDefaultsRepoRoot does).
func directorSkillRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

func TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase(t *testing.T) {
	f := newEscalateFixture(t)
	f.o.InboxPollInterval = 10 * time.Millisecond
	out := runWaitInbox(t, f.o, 20*time.Millisecond, 5*time.Second)
	if out.err == nil || !strings.Contains(out.err.Error(), inboxWaitTimeoutPhrase) {
		t.Fatalf("WaitInbox timeout error = %v; want it to contain inboxWaitTimeoutPhrase %q", out.err, inboxWaitTimeoutPhrase)
	}

	root := directorSkillRepoRoot(t)
	want := "`" + inboxWaitTimeoutPhrase + "`"
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			// A repo that vendors internal/org without the skills tree, as
			// in TestSendDefaults_DocsMatchConstant.
			t.Skipf("%s not found (%v) -- skipping the skill/error lock-step check (vendored repo?)", rel, err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("%s does not contain %s, the phrase of WaitInbox's timeout error -- see this test's doc comment for what to update", rel, want)
		}
	}
}

// directorSkillCopies are the four copies of the /director skill, relative
// to the repository root.
var directorSkillCopies = []string{
	filepath.Join(".claude", "skills", "director", "SKILL.md"),
	filepath.Join(".agents", "skills", "director", "SKILL.md"),
	filepath.Join("templates", "base", ".claude", "skills", "director", "SKILL.md"),
	filepath.Join("templates", "base", ".agents", "skills", "director", "SKILL.md"),
}

// directorSkillAutoMergeSteps are fragments of the /director skill's merge
// steps (its section merge と auto_merge), in the order the skill has them:
// with auto_merge = false the director merges and regates only when the
// human asks; with auto_merge = true it merges the first ready branch with
// ralph org merge, run with --wait-checks 60m in the background and read once
// it ended (plan D13: the verb waits for checks still pending), runs
// merge-order again and regates one branch, and merges that branch once it
// is ready again; with either value it never runs gh pr merge itself, and it
// stops a branch at a refusal or a failure of the verb and does not run it
// again unasked. The last two say that --wait-checks also waits for the
// branch's org to be disbanded after its RESULT (D5), and that the director
// does not disband an org that still runs once the time is up.
var directorSkillAutoMergeSteps = []string{
	"merge も regate も人に頼まれたときだけ打つ",
	"先頭の `ready` を `ralph org merge --branch <ブランチ>` で 1 本 merge する",
	"`--wait-checks 60m` を付けて background で打ち、終わってから終了コードと出力を読む",
	"merge-order を打ち直し、理由の欄が `ralph org regate --branch <ブランチ>` を名指すブランチの 1 本に regate を打つ",
	"そのブランチが `ready` に戻っていれば、2 と同じく merge して 3 に戻る",
	"`gh pr merge` を直接打たない。merge は `ralph org merge` でだけ打つ",
	"そのブランチはそこで止め、stderr の文(理由と次の手)をそのまま人に伝える。人に頼まれないまま打ち直さない",
	"動詞はこの断りでも、同じく `regate.lock` を外し、30 秒ごとに台帳だけを読み直して(git と gh は打たない)org の disband を待つ",
	"director は disband を打たずに、その断りを人に伝える",
}

// directorSkillOldMergeStep is a fragment of the step the skill had before
// [org.director].auto_merge, which merged with gh pr merge directly.
const directorSkillOldMergeStep = "`gh pr merge` で 1 本 merge し"

// withoutSpaces drops every space and line break from s, so a fragment of
// the skill matches however its paragraph is wrapped.
func withoutSpaces(s string) string { return strings.Join(strings.Fields(s), "") }

// TestDirectorSkill_PinsTheAutoMergeSteps pins the /director skill's merge
// steps, which meet AC11 of plan docs/plans/archive/2026-10-17-org-auto-merge.md
// and FR-8's acceptance criterion (with auto_merge = false the director does
// not merge): ralph org merge does not read [org.director].auto_merge to
// refuse (plan D1), so the skill's text is all that keeps a director from
// merging unasked, and the merge of auto_merge = true carries
// --wait-checks 60m, without which it stops at checks still pending. Each
// copy must contain every fragment of
// directorSkillAutoMergeSteps, each after the one before it, compared with
// spaces and line breaks removed (withoutSpaces), and must not contain
// directorSkillOldMergeStep. scripts/check-skill-sync.sh keeps the copies
// identical but does not read Go.
//
// If this test fails after a deliberate change to the steps, put the new
// fragment in directorSkillAutoMergeSteps in the skill's order; otherwise
// restore the text in the section merge と auto_merge of
// .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for the test above.
func TestDirectorSkill_PinsTheAutoMergeSteps(t *testing.T) {
	root := directorSkillRepoRoot(t)
	old := withoutSpaces(directorSkillOldMergeStep)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's merge-step check (vendored repo?)", rel, err)
		}
		text := withoutSpaces(string(data))
		at := 0
		for _, step := range directorSkillAutoMergeSteps {
			want := withoutSpaces(step)
			i := strings.Index(text[at:], want)
			if i < 0 {
				t.Errorf("%s does not contain %q after the fragments before it in directorSkillAutoMergeSteps -- see this test's doc comment for what to update", rel, step)
				continue
			}
			at += i + len(want)
		}
		if strings.Contains(text, old) {
			t.Errorf("%s still contains %q, the step that merged with gh pr merge directly -- see this test's doc comment", rel, directorSkillOldMergeStep)
		}
	}
}

// directorSkillMonitorMaxTimeoutMs is the Monitor tool's maximum timeout_ms
// (30 minutes) in Claude Code 2.1.291, the version of the experiment in
// docs/evidence/director-wait-monitor-2026-10-19.md. The Monitor stops a task
// at its timeout_ms with SIGTERM.
const directorSkillMonitorMaxTimeoutMs = 1800000

// directorSkillMonitorDefaultTimeoutMs is the Monitor tool's default
// timeout_ms (5 minutes), the one it uses when none is passed, in the same
// Claude Code version and evidence.
const directorSkillMonitorDefaultTimeoutMs = 300000

// directorSkillMinuteMs is one minute in milliseconds, the unit of the
// skill's "<K> 分".
const directorSkillMinuteMs = int(time.Minute / time.Millisecond)

var (
	// directorSkillWaitTimeoutRe reads N, the --timeout-ms of the Claude Code
	// director's inbox wait.
	directorSkillWaitTimeoutRe = regexp.MustCompile(`--director d1 --timeout-ms ([0-9]+)`)
	// directorSkillMonitorTimeoutRe reads M, the timeout_ms the skill tells a
	// Claude Code director to pass to the Monitor tool.
	directorSkillMonitorTimeoutRe = regexp.MustCompile("`timeout_ms` に `([0-9]+)`")
	// directorSkillUnboundedWaitRe finds --timeout-ms 0, a wait with no
	// timeout, which the Monitor stops with SIGTERM at its own timeout_ms.
	directorSkillUnboundedWaitRe = regexp.MustCompile(`--timeout-ms 0([^0-9]|$)`)
	// directorSkillMinutesRe reads K of each "<K> 分", a number of minutes.
	directorSkillMinutesRe = regexp.MustCompile(`([0-9]+) ?分`)
)

// withSingleSpaces makes every run of spaces and line breaks in s one space,
// so a fragment of the skill that keeps a space between a flag and its value
// matches however its paragraph is wrapped.
func withSingleSpaces(s string) string { return strings.Join(strings.Fields(s), " ") }

// directorSkillWaitSubsection returns the skill's subsection ### wait で待つ:
// its heading line and the lines after it, up to the next line that starts
// with "#" or the end of the text. It returns "" when the text has no such
// heading.
func directorSkillWaitSubsection(text string) string {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		if start >= 0 {
			return strings.Join(lines[start:i], "\n")
		}
		if strings.TrimSpace(line) == "### wait で待つ" {
			start = i
		}
	}
	if start < 0 {
		return ""
	}
	return strings.Join(lines[start:], "\n")
}

// directorSkillClaudeCodeWaitBullet returns the Claude Code bullet of the
// subsection section, from directorSkillWaitSubsection: the line starting
// "- Claude Code:" and the lines after it, up to the next line that is not
// blank and does not start with a space (the next bullet at the same indent,
// or a paragraph) or the end of the subsection. ok is false when the
// subsection has no such bullet.
func directorSkillClaudeCodeWaitBullet(section string) (bullet string, ok bool) {
	lines := strings.Split(section, "\n")
	start := -1
	for i, line := range lines {
		if start >= 0 {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, " ") {
				return strings.Join(lines[start:i], "\n"), true
			}
			continue
		}
		if strings.HasPrefix(line, "- Claude Code:") {
			start = i
		}
	}
	if start < 0 {
		return "", false
	}
	return strings.Join(lines[start:], "\n"), true
}

// directorSkillOneNumber reads the number of re's single match in the
// Claude Code bullet b. what names the match in the problem it returns, ""
// when there is none.
func directorSkillOneNumber(b string, re *regexp.Regexp, what string) (int, string) {
	matches := re.FindAllStringSubmatch(b, -1)
	if len(matches) != 1 {
		return 0, fmt.Sprintf("the Claude Code bullet of ### wait で待つ has %d matches of %s, want exactly 1", len(matches), what)
	}
	n, err := strconv.Atoi(matches[0][1])
	if err != nil {
		return 0, fmt.Sprintf("the Claude Code bullet's %s does not fit in an int: %v", what, err)
	}
	return n, ""
}

// directorSkillSubsectionNumberProblems checks the numbers of the whole
// subsection s, compared with withSingleSpaces, against the Claude Code
// bullet's N and M: every `timeout_ms` に `X` has X == M; N is a whole number
// of minutes; and every "<K> 分" has K equal to N, to the Monitor's default
// (directorSkillMonitorDefaultTimeoutMs), or to its maximum
// (directorSkillMonitorMaxTimeoutMs), in minutes. The check of X is skipped
// when M could not be read (hasM false), the checks of N and K when N could
// not be read (hasN false), and the check of K also when N is not a whole
// number of minutes. It returns one line per problem.
func directorSkillSubsectionNumberProblems(s string, n int, hasN bool, m int, hasM bool) []string {
	var problems []string
	if hasM {
		for _, match := range directorSkillMonitorTimeoutRe.FindAllStringSubmatch(s, -1) {
			if match[1] != strconv.Itoa(m) {
				problems = append(problems, fmt.Sprintf("the subsection ### wait で待つ has `timeout_ms` に `%s`, but the Claude Code bullet's M=%d: every `timeout_ms` に `<M>` there must be the same", match[1], m))
			}
		}
	}
	if !hasN {
		return problems
	}
	if n%directorSkillMinuteMs != 0 {
		return append(problems, fmt.Sprintf("the wait's --timeout-ms N=%d must be a whole number of minutes (a multiple of %d), as the subsection names it in \"<K> 分\"", n, directorSkillMinuteMs))
	}
	nMin := n / directorSkillMinuteMs
	defaultMin := directorSkillMonitorDefaultTimeoutMs / directorSkillMinuteMs
	maxMin := directorSkillMonitorMaxTimeoutMs / directorSkillMinuteMs
	want := []string{strconv.Itoa(nMin), strconv.Itoa(defaultMin), strconv.Itoa(maxMin)}
	for _, match := range directorSkillMinutesRe.FindAllStringSubmatch(s, -1) {
		if !slices.Contains(want, match[1]) {
			problems = append(problems, fmt.Sprintf("the subsection ### wait で待つ says %s 分, want N/%d=%d (the wait's --timeout-ms N=%d), the Monitor's default %d, or its maximum %d", match[1], directorSkillMinuteMs, nMin, n, defaultMin, maxMin))
		}
	}
	return problems
}

// directorSkillWaitBoundProblems checks that a /director skill text bounds
// the Claude Code director's inbox wait below the Monitor's timeout and
// keeps the other numbers of its subsection ### wait で待つ in step. The
// Claude Code bullet of that subsection, compared with withSingleSpaces,
// holds exactly one --director d1 --timeout-ms N and exactly one
// `timeout_ms` に `M`, with 0 < N < M <= directorSkillMonitorMaxTimeoutMs;
// the whole subsection passes directorSkillSubsectionNumberProblems (every
// `timeout_ms` に `X` is M, N is a whole number of minutes, and every
// "<K> 分" names N, the Monitor's default, or its maximum); and the whole
// text holds no --timeout-ms 0. It returns one line per problem, nil when
// there is none.
func directorSkillWaitBoundProblems(text string) []string {
	var problems []string
	section := directorSkillWaitSubsection(text)
	if bullet, ok := directorSkillClaudeCodeWaitBullet(section); !ok {
		problems = append(problems, `no "- Claude Code:" bullet in the subsection "### wait で待つ"`)
	} else {
		b := withSingleSpaces(bullet)
		n, nProblem := directorSkillOneNumber(b, directorSkillWaitTimeoutRe, "the wait's `--director d1 --timeout-ms <N>`")
		m, mProblem := directorSkillOneNumber(b, directorSkillMonitorTimeoutRe, "the Monitor's \"`timeout_ms` に `<M>`\"")
		for _, p := range []string{nProblem, mProblem} {
			if p != "" {
				problems = append(problems, p)
			}
		}
		if nProblem == "" && n <= 0 {
			problems = append(problems, fmt.Sprintf("the wait's --timeout-ms N=%d must be greater than 0: the wait must time out on its own", n))
		}
		if nProblem == "" && mProblem == "" && n >= m {
			problems = append(problems, fmt.Sprintf("the wait's --timeout-ms N=%d must be less than the Monitor's timeout_ms M=%d, so the wait times out before the Monitor stops it with SIGTERM", n, m))
		}
		if mProblem == "" && m > directorSkillMonitorMaxTimeoutMs {
			problems = append(problems, fmt.Sprintf("the Monitor's timeout_ms M=%d is over the Monitor tool's maximum %d", m, directorSkillMonitorMaxTimeoutMs))
		}
		problems = append(problems, directorSkillSubsectionNumberProblems(withSingleSpaces(section), n, nProblem == "", m, mProblem == "")...)
	}
	if directorSkillUnboundedWaitRe.MatchString(withSingleSpaces(text)) {
		problems = append(problems, "the text contains --timeout-ms 0, a wait with no timeout that the Monitor stops with SIGTERM")
	}
	return problems
}

// TestDirectorSkill_BoundsTheClaudeCodeWaitBelowTheMonitorCap pins the
// numbers of the Claude Code director's inbox wait in the /director skill
// (plan 2026-10-19-org-director-wait-bound.md, D1 and D5).
// The skill tells a Claude Code director to run
// ralph org wait --inbox --director d1 --timeout-ms N under the Monitor tool
// with the Monitor's timeout_ms M. The Monitor stops a task at its
// timeout_ms with SIGTERM, and the skill counts an end by a signal as a
// failure not to run again unless the Monitor said it expired, so the wait
// must time out first: 0 < N < M <= directorSkillMonitorMaxTimeoutMs. The
// subsection ### wait で待つ repeats M in the paragraph on the Monitor's
// expiry and names N in minutes ("25 分"), so every `timeout_ms` に `X` there
// must be M, N must be a whole number of minutes, and every "<K> 分" there
// must be N, the Monitor's default, or its maximum, in minutes. No copy may
// keep --timeout-ms 0, the unbounded wait the skill had before.
// scripts/check-skill-sync.sh keeps the four copies identical but does not
// read the numbers.
//
// If this test fails after a deliberate change to the numbers, keep one
// --director d1 --timeout-ms <N> and one `timeout_ms` に `<M>` in the Claude
// Code bullet of ### wait で待つ, write the same M wherever that subsection
// has `timeout_ms` に, and change each "<K> 分" there that names the old N
// (or change directorSkillWaitBoundProblems with the text); otherwise restore
// the numbers in .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for the first test in
// this file. If a newer Claude Code changes the Monitor's maximum or default,
// change directorSkillMonitorMaxTimeoutMs or
// directorSkillMonitorDefaultTimeoutMs with evidence of the new value.
func TestDirectorSkill_BoundsTheClaudeCodeWaitBelowTheMonitorCap(t *testing.T) {
	root := directorSkillRepoRoot(t)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's wait-bound check (vendored repo?)", rel, err)
		}
		for _, p := range directorSkillWaitBoundProblems(string(data)) {
			t.Errorf("%s: %s -- see this test's doc comment for what to update", rel, p)
		}
	}
}

// TestDirectorSkillWaitBoundProblems runs directorSkillWaitBoundProblems on
// small texts shaped like the skill's subsection ### wait で待つ. Each has
// the Codex bullet after the Claude Code one and, after the list, the
// Monitor's `timeout_ms` に `1800000` of the paragraph on the Monitor's
// expiry: neither may be read as the Claude Code bullet's. A case whose
// bullet has an M other than 1800000 therefore also gets the problem that
// the paragraph's `timeout_ms` is not M.
func TestDirectorSkillWaitBoundProblems(t *testing.T) {
	skill := func(claudeCode string) string {
		return "## 受信箱の処理\n\n### wait で待つ\n\nwait には `--director <自分の id>` を付ける。\n\n" +
			claudeCode +
			"- Codex: `ralph org wait --inbox --director d1 --timeout-ms 300000` のように\n" +
			"  時間を区切った wait を繰り返す。\n\n" +
			"wait を打ち直し、それからは Monitor の `timeout_ms` に `1800000` を渡す。\n\n" +
			"### 見張りが送る 1 行\n"
	}
	bullet := func(n, m string) string {
		return "- Claude Code: `ralph org wait --inbox --director d1 --timeout-ms " + n + " 2>&1`\n" +
			"  を Monitor で見張り、Monitor の `timeout_ms` に `" + m + "` を渡す。\n"
	}
	// oldBullet is the bullet the skill had before the plan, verbatim.
	oldBullet := "- Claude Code: `ralph org wait --inbox --director d1 --timeout-ms 0 2>&1` を\n" +
		"  Monitor で見張るか、Bash の `run_in_background` で走らせる。`2>&1` で、\n" +
		"  stderr の文も同じ出力に出る。終了コード 0 なら open の件を出して返った\n" +
		"  ので、件を処理し、また待つ。件は 1 件 1 行で、ID、org、TYPE、本文の\n" +
		"  1 行目をタブで区切る。stderr の `note:` の行と、`wait --inbox --director: `\n" +
		"  で始まる行が混ざることもあるが、件ではない(扱いは下の箇条)。\n" +
		"  `--timeout-ms 0` には時間切れがないので、終了コード 1 はいつも失敗で、\n" +
		"  打ち直さない。扱いは下の段落のとおり。\n"
	for _, tc := range []struct {
		name string
		text string
		want []string // a fragment of each problem, in order; nil for none
	}{
		{
			name: "the bounded wait below the Monitor's timeout_ms",
			text: skill(bullet("1500000", "1800000")),
		},
		{
			name: "the old bullet: --timeout-ms 0 and no Monitor timeout_ms",
			text: skill(oldBullet),
			want: []string{"has 0 matches of the Monitor's", "N=0 must be greater than 0", "contains --timeout-ms 0"},
		},
		{
			name: "N equal to M",
			text: skill(bullet("1800000", "1800000")),
			want: []string{"N=1800000 must be less than the Monitor's timeout_ms M=1800000"},
		},
		{
			name: "N greater than M",
			text: skill(bullet("1800000", "1500000")),
			want: []string{
				"N=1800000 must be less than the Monitor's timeout_ms M=1500000",
				"has `timeout_ms` に `1800000`, but the Claude Code bullet's M=1500000",
			},
		},
		{
			name: "M over the Monitor's maximum",
			text: skill(bullet("1500000", "1800001")),
			want: []string{
				"M=1800001 is over the Monitor tool's maximum 1800000",
				"has `timeout_ms` に `1800000`, but the Claude Code bullet's M=1800001",
			},
		},
		{
			name: "the paragraph's timeout_ms differs from the bullet's M",
			text: skill(bullet("1500000", "1700000")),
			want: []string{"has `timeout_ms` に `1800000`, but the Claude Code bullet's M=1700000"},
		},
		{
			name: "two timeout_ms values in the bullet",
			text: skill(bullet("1500000", "1800000") +
				"  Monitor の `timeout_ms` に `1800000` より長い時間は渡せない。\n"),
			want: []string{"has 2 matches of the Monitor's"},
		},
		{
			name: "the minutes the bullet names: N's 25, the Monitor's default 5 and maximum 30",
			text: skill("- Claude Code: `ralph org wait --inbox --director d1 --timeout-ms 1500000 2>&1`\n" +
				"  を Monitor で見張り、Monitor の `timeout_ms` に `1800000` を渡す。Monitor は\n" +
				"  `timeout_ms`(既定 5 分、最大 30 分)が過ぎた処理に SIGTERM を送って止める\n" +
				"  ので、wait が自分の区切りの 25 分で先に終わるようにする。\n"),
		},
		{
			name: "a stale 25 分 after the list once N is 20 minutes",
			text: skill(bullet("1200000", "1800000") +
				"\n時間を区切った wait を 25 分や 300 秒ごとに繰り返す。\n\n"),
			want: []string{"says 25 分, want N/60000=20"},
		},
		{
			name: "N not a whole number of minutes",
			text: skill(bullet("1500001", "1800000")),
			want: []string{"N=1500001 must be a whole number of minutes"},
		},
		{
			name: "minutes and timeout_ms outside the subsection",
			text: "### 件の処理\n\n10 分ごとに Monitor の `timeout_ms` に `600000` を渡す。\n\n" +
				skill(bullet("1200000", "1800000")) +
				"\n25 分ごとに Monitor の `timeout_ms` に `900000` を渡す。\n",
		},
		{
			name: "M missing from the bullet, though the paragraph after the list has one",
			text: skill("- Claude Code: `ralph org wait --inbox --director d1 --timeout-ms 1500000 2>&1`\n" +
				"  を Monitor で見張る。\n"),
			want: []string{"has 0 matches of the Monitor's"},
		},
		{
			name: "N missing from the bullet, though the Codex bullet has one",
			text: skill("- Claude Code: `ralph org wait --inbox --director d1 2>&1` を Monitor で\n" +
				"  見張り、Monitor の `timeout_ms` に `1800000` を渡す。\n"),
			want: []string{"has 0 matches of the wait's `--director d1 --timeout-ms <N>`"},
		},
		{
			name: "two --timeout-ms values in the bullet",
			text: skill(bullet("1500000", "1800000") +
				"  Bash なら `ralph org wait --inbox --director d1 --timeout-ms 600000` でもよい。\n"),
			want: []string{"has 2 matches of the wait's"},
		},
		{
			name: "--timeout-ms 0 elsewhere in the text, with a correct bullet",
			text: skill(bullet("1500000", "1800000")) + "\n人の wait は `ralph org wait --inbox --timeout-ms 0` で待つ。\n",
			want: []string{"contains --timeout-ms 0"},
		},
		{
			name: "--timeout-ms 0 at the very end of the text",
			text: skill(bullet("1500000", "1800000")) + "\n人の wait は `ralph org wait --inbox --timeout-ms 0",
			want: []string{"contains --timeout-ms 0"},
		},
		{
			name: "--timeout-ms 0 wrapped over two lines elsewhere",
			text: skill(bullet("1500000", "1800000")) + "\n人の wait は `ralph org wait --inbox --timeout-ms\n0` で待つ。\n",
			want: []string{"contains --timeout-ms 0"},
		},
		{
			name: "the bullet wrapped over lines",
			text: skill("- Claude Code: `ralph org wait --inbox --director d1 --timeout-ms\n" +
				"  1500000 2>&1` を Monitor で見張り、Monitor の `timeout_ms`\n" +
				"  に `1800000` を渡す。\n"),
		},
		{
			name: "the bullet at the end of the text",
			text: "### wait で待つ\n\n" + bullet("1500000", "1800000"),
		},
		{
			name: "no Claude Code bullet",
			text: skill(""),
			want: []string{`no "- Claude Code:" bullet`},
		},
		{
			name: "a Claude Code bullet only in another subsection",
			text: "### 件の処理\n\n" + bullet("1500000", "1800000") + "\n" + skill(""),
			want: []string{`no "- Claude Code:" bullet`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := directorSkillWaitBoundProblems(tc.text)
			if len(got) != len(tc.want) {
				t.Fatalf("problems = %q, want %d matching %q", got, len(tc.want), tc.want)
			}
			for i, frag := range tc.want {
				if !strings.Contains(got[i], frag) {
					t.Errorf("problem %d = %q, want it to contain %q", i, got[i], frag)
				}
			}
		})
	}
}
