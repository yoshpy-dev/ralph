package org

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The heading of the reviewer template's cross-review TASK section (plan
// 2026-11-01-org-cross-review, issue #269; rewritten by plan
// 2026-11-22-org-xreview-own-model, issue #360, for the seat to review the
// branch on its own model) and the leads of the lines and code blocks the
// tests below read. reviewerXreviewResultLead also ends the section's
// numbered steps.
const (
	reviewerXreviewHeading     = "## cross-review の TASK"
	reviewerXreviewWhere       = "the reviewer's cross-review TASK section"
	reviewerXreviewDiffLead    = "2. 読む前の git の状態と差分を書く"
	reviewerXreviewAfterLead   = "5. report を書く前に、git の状態を比べる"
	reviewerXreviewReportLead  = "6. `REPORT:` のパスに、次の形で report を書く"
	reviewerXreviewResultLead  = "cross-review が通ったときの RESULT の例"
	reviewerXreviewBlockedLead = "ACTION_REQUIRED があるときの BLOCKED の例"
)

// reviewerXreviewNoOtherModel is the sentence of the section's opening that
// has the seat call no other model (AC1 of plan
// 2026-11-22-org-xreview-own-model), and reviewerXreviewOldLinesIgnored the
// one that has it read none of the lines an earlier leader wrote for that
// call (AC2), so a TASK from an earlier leader still works. They are the only
// places of the section that name those commands and lines.
const (
	reviewerXreviewNoOtherModel    = "`codex exec`、`claude -p`、そのほかの CLI で別のモデルを呼ばないでください"
	reviewerXreviewOldLinesIgnored = "本文に `IMPLEMENTER_DRIVER:`、`REVIEWER_CODEX:`、`REVIEWER_CLAUDE:` の行があっても読みません" +
		"(前の版の leader が書く行で、別のモデルを呼ばないこの手順には要りません)"
)

// reviewerXreviewGone are words of the section from before the seat reviewed
// on its own model: the other model's call (codex exec, claude -p, the
// helpers that picked and checked the other CLI, the MCP lists, the
// 20-minute watchdog and its background run, the rc and log: lines of the
// run, its -out.md file, and the skipped reasons of a missing CLI or MCP
// servers left on), the TASK lines that fed it, and the wording of "the other
// model". The section names none of them outside reviewerXreviewNoOtherModel
// and reviewerXreviewOldLinesIgnored.
var reviewerXreviewGone = []string{
	"codex exec", "claude -p", "pick_reviewer", "xreview-helpers.sh", "codex-check.sh", "mcp list", "sleep 1200",
	"run_in_background", "codex rc=", "claude rc=", "`log:`", "-out.md", "not available", "helpers: missing", "mcp: not off",
	"IMPLEMENTER_DRIVER", "REVIEWER_CODEX", "REVIEWER_CLAUDE", "相手", "20 分",
}

// reviewerXreviewBaseRule is step 1's condition on XREVIEW_BASE:'s value and
// reviewerXreviewBaseLead the start of the sentence its check command
// follows (fix F7 of plan 2026-11-01-org-cross-review, cross-review cycle 1's
// ACTION_REQUIRED #2): the reviewer asks with a QUESTION when the command
// prints base: invalid, and not when it prints neither line, a heredoc left
// open that the reviewer types again (fix F8, self-review cycle 2's C2-2).
// The command replaced F6's list of characters with git check-ref-format
// --branch, so a default branch git takes (release=stable, release(v2),
// dévelop) no longer blocks the org.
const (
	reviewerXreviewBaseRule = "`XREVIEW_BASE:` の値でこの手順の終わりのコマンドが `base: invalid` を出すか"
	reviewerXreviewBaseLead = "`XREVIEW_BASE:` の値は、worktree で次のコマンドを"
)

// reviewerXreviewBaseNames are default branches under origin/ that git takes
// as branch names and step 1's check passes: names with + and @ (fix F6),
// and with =, parentheses and a letter outside ASCII (fix F7). Step 2's
// command runs with each.
var reviewerXreviewBaseNames = []string{"origin/release+stable", "origin/a@b", "origin/release=stable", "origin/release(v2)", "origin/dévelop"}

// reviewerXreviewSection renders the reviewer template and returns its
// cross-review TASK section.
func reviewerXreviewSection(t *testing.T) string {
	t.Helper()
	return reviewerXreviewSectionOf(t, renderSeatPrompt(t, "reviewer"))
}

// reviewerXreviewSectionFor renders the reviewer template for the seat
// reviewer on driver and model and returns its cross-review TASK section.
func reviewerXreviewSectionFor(t *testing.T, driver, model string) string {
	t.Helper()
	vars := testRolePromptVars()
	vars.Role, vars.SeatID, vars.Driver, vars.Model = "reviewer", "reviewer", driver, model
	text, ok, err := RenderRolePrompt("reviewer", vars)
	if err != nil || !ok {
		t.Fatalf("RenderRolePrompt(reviewer) = (ok %v, err %v), want the built-in template", ok, err)
	}
	return reviewerXreviewSectionOf(t, text)
}

// reviewerXreviewSectionOf returns the cross-review TASK section of the
// rendered reviewer template text.
func reviewerXreviewSectionOf(t *testing.T, text string) string {
	t.Helper()
	section, found := markdownSection(text, reviewerXreviewHeading)
	if !found {
		t.Fatalf("expected the reviewer template to contain a %q section, got:\n%s", reviewerXreviewHeading, text)
	}
	return section
}

// reviewerXreviewStep returns step n of the section's numbered list: the text
// from the line starting "<n>. " up to the line starting "<n+1>. ", or, for
// the last step, up to the line of the RESULT example's lead. The steps hold
// indented bullets, tables and code blocks, at which markdownItem would stop.
func reviewerXreviewStep(t *testing.T, n int) string {
	t.Helper()
	section := reviewerXreviewSection(t)
	_, step, found := strings.Cut(section, "\n"+strconv.Itoa(n)+". ")
	if !found {
		t.Fatalf("expected %s to have a step %d., got section:\n%s", reviewerXreviewWhere, n, section)
	}
	step, _, _ = strings.Cut(step, "\n"+strconv.Itoa(n+1)+". ")
	step, _, _ = strings.Cut(step, "\n"+reviewerXreviewResultLead)
	return strconv.Itoa(n) + ". " + step
}

// reviewerXreviewStepWhere names step n in a failure message.
func reviewerXreviewStepWhere(n int) string {
	return "step " + strconv.Itoa(n) + " of " + reviewerXreviewWhere
}

// assertContainsPhrases fails the test for each phrase of wants that text
// does not contain, ignoring white space (containsPhrase). where names text in
// the failure message.
func assertContainsPhrases(t *testing.T, text, where string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !containsPhrase(text, want) {
			t.Errorf("expected %s to say %q, got:\n%s", where, want, text)
		}
	}
}

// TestRenderRolePrompt_Reviewer_XreviewTaskIsNotAReviewTask pins D5 of plan
// 2026-11-01-org-cross-review: the first item of the reviewer's mission
// leaves a TASK whose TASK_ID starts with xreview- to the 「cross-review の
// TASK」 section, as it leaves a gate- TASK to the 「gate の TASK」 section;
// that section comes after the gate TASK section and before the
// star-topology rule; its opening says what the TASK is (the feature org
// leader's, sent with [org.review].cross_review on, after the review and
// before sync-docs; no gate; the seat itself reads the branch's diff again,
// apart from the review TASK, and triages its findings into a report); and
// the mission's opening and the scope rule let the reviewer write the
// section's files under .harness/state/ outside its scope.
func TestRenderRolePrompt_Reviewer_XreviewTaskIsNotAReviewTask(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## ミッション' section, got:\n%s", text)
	}
	if item, ok := markdownItem(mission, "1. レビューの TASK"); !ok ||
		!containsPhrase(item, "(TASK_ID が `gate-` で始まる TASK と `xreview-` で始まる TASK は、レビューの TASK ではなく、それぞれ下の「gate の TASK」と「cross-review の TASK」の節のとおりに扱う)") {
		t.Errorf("expected the reviewer mission's first item to leave an xreview- TASK to the cross-review TASK section, got item:\n%s", item)
	}
	missionIntro, _, _ := strings.Cut(mission, "\n1. ")
	assertContainsPhrases(t, missionIntro, "the reviewer mission's opening", []string{
		"と、cross-review の TASK で書く `.harness/state/xreview-org-a-` で始まるファイルと、gate の TASK で `ralph org gate` が台帳に書く記録とログ",
	})
	gateAt, xreviewAt, starAt := strings.Index(text, "\n## gate の TASK\n"), strings.Index(text, "\n"+reviewerXreviewHeading+"\n"), strings.Index(text, "\n## スター型トポロジのルール\n")
	if gateAt < 0 || xreviewAt < gateAt || starAt < xreviewAt {
		t.Errorf("expected the %q section between the gate TASK section and the star-topology rule, got:\n%s", reviewerXreviewHeading, text)
	}

	intro, _, _ := strings.Cut(reviewerXreviewSection(t), "\n1. ")
	assertContainsPhrases(t, intro, "the opening of "+reviewerXreviewWhere, []string{
		"TASK_ID が `xreview-` で始まる TASK は、cross-review の TASK です",
		"機能ごとの org の leader が、`[org.review].cross_review` が on のとき、レビューのあと、sync-docs の前に送ります",
		"決定論ゲートは打たず、レビューの TASK とは別に、あなた自身がこの branch の差分を読み直して所見を出し、triage して report に書きます",
	})

	scope, _ := markdownSection(text, "## スコープ規律")
	assertContainsPhrases(t, scope, "the reviewer's スコープ規律 section", []string{
		"(レポート、ゲートの生成物、cross-review の TASK と gate の TASK で書くものは、ミッションに書いたとおり例外です)",
	})
}

// TestRenderRolePrompt_Reviewer_XreviewReviewsOnTheSeatsOwnModel pins AC1 of
// plan 2026-11-22-org-xreview-own-model (the user's decision of 2026-11-21:
// in the org runtime the reviewer seat reviews, on the model it was given,
// whether or not that is the implementer's model family): the section's
// opening says the seat's own model (the driver: and model: line at the top
// of the template) reviews, the same or the other family alike, and has the
// seat call no other model with codex exec, claude -p or another CLI. Those
// commands, and the lines an earlier leader wrote for that call, appear only
// in that sentence and in the one that has the seat read none of them; the
// rest of the section names none of reviewerXreviewGone, and no command of
// its steps runs codex or claude.
func TestRenderRolePrompt_Reviewer_XreviewReviewsOnTheSeatsOwnModel(t *testing.T) {
	section := reviewerXreviewSection(t)
	intro, _, _ := strings.Cut(section, "\n1. ")
	assertPhrasesInOrder(t, intro, "the opening of "+reviewerXreviewWhere, []string{
		"あなた自身がこの branch の差分を読み直して所見を出し、triage して report に書きます",
		"レビューするのはこの座席のモデル(冒頭の `driver:` と `model:` の行)で、implementer と同じ系列でも違う系列でも変わりません",
		reviewerXreviewNoOtherModel,
	})
	if !strings.Contains(renderSeatPrompt(t, "reviewer"), "\n- driver: ") {
		t.Errorf("expected the reviewer template to have the driver: and model: line the cross-review section points at")
	}

	rest := squashSpace(section)
	for _, sentence := range []string{reviewerXreviewNoOtherModel, reviewerXreviewOldLinesIgnored} {
		if n := strings.Count(rest, squashSpace(sentence)); n != 1 {
			t.Fatalf("expected %s to say %q once, got %d times in:\n%s", reviewerXreviewWhere, sentence, n, section)
		}
		rest = strings.Replace(rest, squashSpace(sentence), "", 1)
	}
	for _, word := range reviewerXreviewGone {
		if strings.Contains(rest, squashSpace(word)) {
			t.Errorf("expected %s to name %q only in the sentences that have the seat call no other model and read none of an earlier leader's lines, got section:\n%s", reviewerXreviewWhere, word, section)
		}
	}

	for _, c := range []struct {
		n    int
		lead string
	}{{1, reviewerXreviewBaseLead}, {2, reviewerXreviewDiffLead}, {5, reviewerXreviewAfterLead}} {
		block := fencedBlockAfter(t, reviewerXreviewStep(t, c.n), c.lead, reviewerXreviewStepWhere(c.n))
		for _, cli := range []string{"codex", "claude"} {
			if strings.Contains(block, cli) {
				t.Errorf("expected the command of %s to run no %s, got:\n%s", reviewerXreviewStepWhere(c.n), cli, block)
			}
		}
	}
}

// TestRenderRolePrompt_Reviewer_XreviewTaskCarriesItsLinesInTheBody pins AC2
// of plan 2026-11-22-org-xreview-own-model and D5 of plan
// 2026-11-01-org-cross-review: the section lists the TASK's three body lines
// (body lines like CHECKS:, not headers) with what each holds: XREVIEW_BASE:
// named apart from the RESULT header BASE:, CYCLE: copied to the report
// while the leader decides whether to send, and REPORT: under docs/reports/
// with the leader's number; the IMPLEMENTER_DRIVER:, REVIEWER_CODEX: and
// REVIEWER_CLAUDE: lines an earlier leader writes are not read; and it says
// which line fills which placeholder of the commands. Step 1 reads nothing
// and asks the leader with a QUESTION, writing nothing, when one of the three
// lines is missing, XREVIEW_BASE:'s check prints base: invalid
// (reviewerXreviewBaseRule), or REPORT: is not a review-xreview- path of
// this org with a date of digits (so no path leaves docs/reports/): lines
// the leader fills when it sends the TASK again, each going into a command
// as it is (verify cycle 1, V2, of the earlier plan).
func TestRenderRolePrompt_Reviewer_XreviewTaskCarriesItsLinesInTheBody(t *testing.T) {
	intro, _, _ := strings.Cut(reviewerXreviewSection(t), "\n1. ")
	where := "the opening of " + reviewerXreviewWhere
	assertPhrasesInOrder(t, intro, where, []string{
		"TASK の本文には次の行があります(`CHECKS:` と同じく、ヘッダではなく本文の行です)",
		reviewerXreviewOldLinesIgnored,
		"下の手順のコマンドは、`<k>` を `REPORT:` のパスの終わりの番号に、`<XREVIEW_BASE>` を `XREVIEW_BASE:` の行の値に書き換えて打ちます",
	})
	for _, c := range []struct{ marker, want string }{
		{"- `XREVIEW_BASE:", "`XREVIEW_BASE: origin/<既定のブランチ>`: 差分の base。RESULT のヘッダの `BASE:` とは別の行"},
		{"- `CYCLE:", "`CYCLE: <n>/<上限>`: この cross-review が何回目かと、その上限。report に写す(TASK を送るかどうかは leader が決める)"},
		{"- `REPORT:", "`REPORT: docs/reports/review-xreview-<YYYY-MM-DD>-org-a-c<k>.md`: report を書くパス。`<k>` は leader が決めた番号"},
	} {
		if item, ok := markdownItem(intro, c.marker); !ok || !containsPhrase(item, c.want) {
			t.Errorf("expected the item of %s naming %q to say %q, got item:\n%s", where, c.marker, c.want, item)
		}
	}
	if keys := reviewerXreviewTaskKeys(t); !slices.Equal(keys, []string{"XREVIEW_BASE", "CYCLE", "REPORT"}) {
		t.Errorf("expected %s to list the TASK lines XREVIEW_BASE:, CYCLE: and REPORT: only, got %q", where, keys)
	}

	step := reviewerXreviewStep(t, 1)
	assertPhrasesInOrder(t, step, reviewerXreviewStepWhere(1), []string{
		"`XREVIEW_BASE:`・`CYCLE:`・`REPORT:` のどれかの行がないか、" + reviewerXreviewBaseRule + "、" +
			"`REPORT:` のパスが `docs/reports/review-xreview-<YYYY-MM-DD>-org-a-c<k>.md`(`<YYYY-MM-DD>` は数字の日付、`<k>` は数字)の形でないときは、" +
			"差分を読まず、何も書かずに QUESTION で leader に聞く",
		"これらの値は下のコマンドの引数と report のパスにそのまま入るので、option として読まれる値、`docs/reports/` の外を指すパス、引用符や `$` のように shell が読む文字を持つ値は使わない",
		"本文には受けた TASK の TASK_ID と、欠けた行か形の違う行を書く(どれも leader が TASK を直して送り直せばそろう)",
	})
}

// TestRenderRolePrompt_Reviewer_XreviewWritesTheGitStateAndTheDiff pins step
// 2 (AC1 and AC2 of plan 2026-11-22-org-xreview-own-model): one shell writes
// the git state before the reading (HEAD, the index, the work tree with its
// untracked files' contents, in the form of leaderAdvisoryGitState) and the
// branch's diff since the base, with --end-of-options, to the cycle's files
// under the ignored .harness/state/, and prints diff: written or diff:
// failed. diff: failed skips 3 and 4 and goes through step 5's comparison of
// the git state first (the cross-review finding of the branch,
// docs/reports/cross-review-triage-org-xreview-own-model.md: without it a
// commit made while git diff failed would end in an incomplete RESULT the
// leader goes on from); with git: same it is an incomplete report and RESULT
// with the gist of git diff's stderr in the output of step 2's command, not
// step 5's (a missing base ref is the leader's fetch to fix; self-review C2-3
// of the plan), as the claude path of the earlier section had it; diff:
// written goes on to 3.
func TestRenderRolePrompt_Reviewer_XreviewWritesTheGitStateAndTheDiff(t *testing.T) {
	step := reviewerXreviewStep(t, 2)
	where := reviewerXreviewStepWhere(2)
	assertPhrasesInOrder(t, step, where, []string{
		reviewerXreviewDiffLead,
		"worktree で次のコマンドを、まとめて 1 回の shell で打つ(Bash ツールは shell の状態を次の呼び出しに持ち越さない)",
		"コマンドは、読む前の git の状態(HEAD、index、未追跡のファイルを含む作業ツリー)を `.harness/state/xreview-org-a-c<k>-git-before.txt` に、" +
			"base からのこの branch の差分を `.harness/state/xreview-org-a-c<k>.diff` に書く",
		"どちらも git が無視する `.harness/state/` の下に書く",
		"`git diff` の `--end-of-options` は、1 で形を確かめた `XREVIEW_BASE:` の値を、それでも option としては読ませないためのもの",
		"- `diff: failed` なら、差分を読めない。3 と 4 を飛ばして 5 に進み、git の状態を比べる" +
			"(3 で読み終えられなかったときと同じく、report を書く前に比べる)",
		"`git: changed` なら 5 のとおり BLOCKED を返す",
		"`git: same` なら、2 のコマンドの出力に出た `git diff` の stderr の要点(`unknown revision` など)を理由に入れて、6 で report を " +
			"`Reviewer status: incomplete (git diff failed: <要点>)` で書き、7 のとおり `XREVIEW: incomplete (git diff failed: <要点>)` の RESULT を返す" +
			"(base の ref がないなら、leader の fetch の問題だと leader が見分けられる)",
		"- `diff: written` なら 3 に進む",
	})
	block := fencedBlockAfter(t, step, reviewerXreviewDiffLead, where)
	want := []string{
		`p=".harness/state/xreview-org-a-c<k>"; mkdir -p .harness/state`,
		leaderAdvisoryGitState + ` > "$p-git-before.txt"`,
		`if git diff --end-of-options "<XREVIEW_BASE>...HEAD" > "$p.diff"; then echo 'diff: written'; else echo 'diff: failed'; fi`,
	}
	if got := strings.Split(block, "\n"); !slices.Equal(got, want) {
		t.Errorf("expected the command of %s to be the lines %q, got %q", where, want, got)
	}
}

// TestRenderRolePrompt_Reviewer_XreviewReadsTheDiffItself pins step 3 (AC1 of
// plan 2026-11-22-org-xreview-own-model): the seat reads the diff file first
// (every change of the branch since the base), then the files it touches; it
// reads the diff again rather than retracing the report of its review TASK,
// doubting the whole branch (the adversarial stance of the claude
// instruction of the earlier section); it looks for correctness, security,
// error handling, logic, test gaps and blind spots of the change; each
// material finding gets a severity of HIGH, MEDIUM or LOW, its file:line,
// what can go wrong, why, the likely impact and a concrete fix; one strong
// finding goes before several weak ones, and no finding is said plainly; it
// changes nothing in the worktree while it reads, and a child subagent it
// splits the findings to writes nothing there either, by the rule of the
// ## 座席内 fan-out section, which says where the child's files go and why
// (self-review L3 and C2-2 of the plan; the rule is pinned by
// TestRenderRolePrompt_Reviewer_FanOutKeepsTheChildrensFilesOutOfTheWorktree);
// and a reading it cannot finish skips the triage and is an incomplete report
// and RESULT.
func TestRenderRolePrompt_Reviewer_XreviewReadsTheDiffItself(t *testing.T) {
	assertPhrasesInOrder(t, reviewerXreviewStep(t, 3), reviewerXreviewStepWhere(3), []string{
		"3. 差分を読み、所見を出す。先に `.harness/state/xreview-org-a-c<k>.diff` を読み、必要に応じて差分が触るファイルを worktree で読む",
		"diff のファイルには、base からのこの branch の変更がすべて入っている",
		"レビューの TASK で書いた自分のレポートを読み返して同じ結論をなぞるのではなく、差分を読み直し、branch 全体をもう一度疑ってかかる目で見る" +
			"(証拠が出るまでは、変更が気づきにくく、直すのに高くつく形で壊れうるとみなす)",
		"見る観点は、正しさ、security、エラーの扱いの抜け、論理の誤り、テストの抜け、この変更に特有の見落とし",
		"- 実質のある所見だけを出す。所見ごとに、重大度(HIGH / MEDIUM / LOW)、file:line、何が起きうるか、なぜそのコードで起きるか、ありそうな影響、それを直す具体的な変更をまとめる",
		"- 弱い所見を数件出すより、強い所見を 1 件出す。問題が見つからなければ、所見なしとはっきり言う(所見は 0 件)",
		"- 読むあいだは worktree のファイルを書き換えず、コミットせず、git の状態を変えるコマンド(checkout、stash、reset など)を打たない",
		"- 所見を出すのを子サブエージェントに分けるときは、「座席内 fan-out」の節のとおり、子にも worktree に何も書かせない" +
			"(子の作業ファイルの置き場所とその理由は、その節にある)",
		"- 差分か変わったファイルを読み終えられなければ(ファイルが読めない、など)、出した所見も triage せずに 5 に進み、" +
			"6 で report を `Reviewer status: incomplete (<理由>)` で書いて、7 のとおり `XREVIEW: incomplete (<理由>)` の RESULT を返す",
	})
}

// TestRenderRolePrompt_Reviewer_XreviewTriagesOnTwoAxes pins the triage of
// step 4, the standard flow's (/cross-review step 5) in the seat, now on the
// findings of step 3: the reviewer triages with the diff, the plan and its
// own review report; each finding is judged on the two axes into
// ACTION_REQUIRED, WORTH_CONSIDERING or DISMISSED by the six rows of the
// table, upward when unsure, never dropped, and a dismissed one carries one
// of the five categories. The earlier section's reading of the other model's
// output (its -out.md file, codex's [P0] to [P3]) is gone.
func TestRenderRolePrompt_Reviewer_XreviewTriagesOnTwoAxes(t *testing.T) {
	step := reviewerXreviewStep(t, 4)
	where := reviewerXreviewStepWhere(4)
	assertContainsPhrases(t, step, where, []string{
		"3 で出した所見を 1 件ずつ triage する",
		"triage には、差分と、この org のレビューの TASK で読んだ計画と、自分のレポートを使う",
		"軸 1 は本当の問題か",
		"軸 2 はいま直す価値があるか",
		"迷ったら上に分ける(`DISMISSED` より `WORTH_CONSIDERING`、`WORTH_CONSIDERING` より `ACTION_REQUIRED`)。所見を黙って落とさない",
		"`DISMISSED` には、次の 5 つの分類のどれかと理由を付ける",
	})
	for _, row := range []string{
		"| はい | はい | `ACTION_REQUIRED` |",
		"| はい | 迷う | `WORTH_CONSIDERING` |",
		"| 迷う | はい | `WORTH_CONSIDERING` |",
		"| 迷う | 迷う | `WORTH_CONSIDERING` |",
		"| いいえ | — | `DISMISSED` |",
		"| — | いいえ(scope の外、もう直してある) | `DISMISSED` |",
	} {
		if !hasLine(step, row) {
			t.Errorf("expected %s to have the table row %s, got:\n%s", where, row, step)
		}
	}
	for _, category := range []string{"false-positive", "already-addressed", "style-preference", "out-of-scope", "context-aware-safe"} {
		if !containsPhrase(step, "`"+category+"`(") {
			t.Errorf("expected %s to name the DISMISSED category %s with its meaning, got:\n%s", where, category, step)
		}
	}
	for _, gone := range []string{"[P0]", "所見は相手のモデルの意見"} {
		if containsPhrase(step, gone) {
			t.Errorf("expected %s to read no other model's output, but it says %q:\n%s", where, gone, step)
		}
	}
}

// TestRenderRolePrompt_Reviewer_XreviewComparesTheGitStateBeforeTheReport
// pins step 5 (AC1 of plan 2026-11-22-org-xreview-own-model, the codex plan
// advisory's MEDIUM 1): after the triage and before the report, one shell
// writes the git state again in step 2's form and compares it with cmp,
// printing git: same or git: changed. git: changed (the state differs
// between before the reading and before the report: HEAD or the work tree
// moved after step 2 wrote the state, so findings, if the seat made any, are
// not on the branch as it is; said so that it holds on each of the three
// ways into step 5, from 4, from 3's reading it could not finish, and from
// 2's diff: failed, self-review C2-3 of the plan)
// is a BLOCKED with XREVIEW: git_changed (one word like action_required), the
// gist of the two state files' diff and the files as EVIDENCE, with no
// report, so not a run the leader counts, and nothing undone; git: same goes
// on to 6.
func TestRenderRolePrompt_Reviewer_XreviewComparesTheGitStateBeforeTheReport(t *testing.T) {
	step := reviewerXreviewStep(t, 5)
	where := reviewerXreviewStepWhere(5)
	assertPhrasesInOrder(t, step, where, []string{
		reviewerXreviewAfterLead,
		"コマンドは、今の git の状態を 2 と同じ形で `.harness/state/xreview-org-a-c<k>-git-after.txt` に書き、2 で書いた `-git-before.txt` と `cmp` で比べる",
		"- `git: changed` なら(読む前と report を書く前で git の状態が違う。2 で状態を書いたあとに HEAD か作業ツリーが動いたので、所見を出していたなら、それは今の branch を見ていない)、" +
			"変わった worktree のまま先に進まない。report を書かず、何も戻さずに BLOCKED を返す",
		"本文に `XREVIEW: git_changed` の行と、`-git-before.txt` と `-git-after.txt` の `diff` から読んだ変わったところの要点を書き、" +
			"EVIDENCE に 2 つのファイルのパスを書く(diff をそのまま貼らない)。report を書かないので、この回は leader の数える回に入らない",
		"- `git: same` なら 6 に進む",
	})
	block := fencedBlockAfter(t, step, reviewerXreviewAfterLead, where)
	want := []string{
		`p=".harness/state/xreview-org-a-c<k>"`,
		leaderAdvisoryGitState + ` > "$p-git-after.txt"`,
		`if cmp -s "$p-git-before.txt" "$p-git-after.txt"; then echo 'git: same'; else echo 'git: changed'; fi`,
	}
	if got := strings.Split(block, "\n"); !slices.Equal(got, want) {
		t.Errorf("expected the command of %s to be the lines %q, got %q", where, want, got)
	}
	if section := reviewerXreviewSection(t); strings.Contains(section, "XREVIEW: git changed") {
		t.Errorf("expected %s to spell the value git_changed in one word, like action_required, got:\n%s", reviewerXreviewWhere, section)
	}
}

// reviewerXreviewReportLines are the header lines of step 6's report for a
// reviewer seat with the id reviewer on codex with the model gpt-stub, in
// order: the standard flow's triage template's lines that a seat can fill,
// with Reviewer: naming the seat, its driver and its model (AC3 of plan
// 2026-11-22-org-xreview-own-model), HEAD:, the HEAD read, and Base: added,
// and the Diff: line of Triage context in place of the other model's output
// file.
var reviewerXreviewReportLines = []string{
	"# Cross-review triage report: org-a c<k>",
	"- Date: <YYYY-MM-DD>",
	"- Reviewer: reviewer 座席 reviewer(codex、gpt-stub)",
	"- Triager: reviewer 座席 reviewer",
	"- Reviewer status: <complete | incomplete (<理由>)>",
	"- HEAD: <読んだ HEAD の SHA>",
	"- Base: <XREVIEW_BASE の値>",
	"- Cycle: <CYCLE の値>",
	"- Total reviewer findings: <3 で出した所見の数>",
	"- After triage: ACTION_REQUIRED=<数>, WORTH_CONSIDERING=<数>, DISMISSED=<数>",
	"## Triage context",
	"- Diff: <XREVIEW_BASE の値>...<HEAD の SHA>",
	"- Review report: <レビューの TASK で書いた自分のレポートのパス>",
	"- Summary: <差分を読み終えたか。読み終えなかったなら、その理由>",
	"## ACTION_REQUIRED",
	"| # | Reviewer finding | Triage rationale | Affected file(s) |",
	"## WORTH_CONSIDERING",
	"| # | Reviewer finding | Triage rationale | Affected file(s) |",
	"## DISMISSED",
	"| # | Reviewer finding | Dismissal reason | Category |",
}

// TestRenderRolePrompt_Reviewer_XreviewWritesTheReportUncommitted pins AC3 of
// plan 2026-11-22-org-xreview-own-model and D3 of plan
// 2026-11-01-org-cross-review: step 6 writes the report at REPORT:'s path in
// the shape of its code block, whose lines are reviewerXreviewReportLines in
// order, each once; HEAD: is the HEAD read (the first line of
// -git-before.txt) and Diff: the base and that HEAD; Reviewer: is the seat
// with its driver and model, which the leader copies to the PR body's
// Cross-review: line, and a seat with no model has the CLI default there; a
// finding's file is relative to the worktree's root, since the report is
// committed; an incomplete run still has the zero counts and says why in
// Summary:; and the report is not committed (the leader commits it). The
// earlier report's Driver: line, its Reviewer output: line and the skipped
// status are gone.
func TestRenderRolePrompt_Reviewer_XreviewWritesTheReportUncommitted(t *testing.T) {
	step := reviewerXreviewStep(t, 6)
	where := reviewerXreviewStepWhere(6)
	assertPhrasesInOrder(t, step, where, []string{
		"`HEAD:` には読んだ HEAD(`-git-before.txt` の 1 行目)を書き、`## Triage context` の `Diff:` には `<XREVIEW_BASE の値>...<その HEAD の SHA>` を書く",
		"`Reviewer:` の行は下の形のまま書く(この座席の seat_id と driver と model で、leader はこの値を PR の本文の `Cross-review:` の行に写す)",
		"所見のファイルのパスは、worktree の root からの相対パス(`internal/foo/bar.go:42`)で表に書く(report はコミットされ、PR からも読まれるので、手元の絶対パスを残さない)",
		"incomplete のときも `Total reviewer findings: 0` と `After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0` の行を残し、`Summary:` に読み終えなかった理由を書く",
		"report はコミットしない(leader がコミットする)",
	})

	section := reviewerXreviewSectionFor(t, "codex", "gpt-stub")
	_, rest, _ := strings.Cut(section, "\n6. ")
	block := fencedBlockAfter(t, rest, "`REPORT:` のパスに、次の形で report を書く", where)
	var got []string
	for line := range strings.SplitSeq(block, "\n") {
		if slices.Contains(reviewerXreviewReportLines, line) {
			got = append(got, line)
		}
	}
	if !slices.Equal(got, reviewerXreviewReportLines) {
		t.Errorf("expected the report of %s to have the lines %q in this order, each once, got %q:\n%s", where, reviewerXreviewReportLines, got, block)
	}
	for _, gone := range []string{"- Driver:", "Reviewer output:", "skipped"} {
		if strings.Contains(block, gone) {
			t.Errorf("expected the report of %s to have no %q, got:\n%s", where, gone, block)
		}
	}

	defaultModel := reviewerXreviewSectionFor(t, "claude", "")
	if want := "\n   - Reviewer: reviewer 座席 reviewer(claude、" + CLIDefaultModelText + ")\n"; !strings.Contains(defaultModel, want) {
		t.Errorf("expected the report of %s for a seat with no model to name the CLI default, %q, got section:\n%s", where, want, defaultModel)
	}
}

// TestRenderRolePrompt_Reviewer_XreviewReportIsReadByTheTriageCounter runs
// count_triage_findings of scripts/xreview-helpers.sh, the standard flow's
// triage report parser, on step 6's report with three counts filled in, and
// gets the three counts back: the After triage: line keeps the shape that
// parser reads.
func TestRenderRolePrompt_Reviewer_XreviewReportIsReadByTheTriageCounter(t *testing.T) {
	readRepoText(t, "scripts/xreview-helpers.sh")
	helpers := filepath.Join(directorSkillRepoRoot(t), "scripts", "xreview-helpers.sh")
	block := fencedBlockAfter(t, reviewerXreviewStep(t, 6), reviewerXreviewReportLead, reviewerXreviewStepWhere(6))
	report := block
	for _, n := range []string{"2", "1", "3"} {
		report = strings.Replace(report, "<数>", n, 1)
	}
	path := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("sh", "-c", `. "$1"; for c in ACTION_REQUIRED WORTH_CONSIDERING DISMISSED; do count_triage_findings "$2" "$c"; done`, "sh", helpers, path).CombinedOutput()
	if err != nil || string(out) != "2\n1\n3\n" {
		t.Errorf("count_triage_findings on the filled-in report = %q, err %v; want 2, 1 and 3, the counts of its After triage: line:\n%s", out, err, report)
	}
}

// TestRenderRolePrompt_Reviewer_XreviewAnswersBlockedOnlyForActionRequired
// pins AC3 of plan 2026-11-22-org-xreview-own-model and D5 of plan
// 2026-11-01-org-cross-review: step 7 answers with the TASK's TASK_ID, a
// BLOCKED with XREVIEW: action_required when ACTION_REQUIRED is not 0 and a
// RESULT with XREVIEW: pass (the diff read, no ACTION_REQUIRED) or
// incomplete otherwise, each with REPORT:, both body lines like CHECKS:,
// leaving the XREVIEW: git_changed BLOCKED and the QUESTION about the TASK's
// lines to steps 5 and 1. The section returns no XREVIEW: skipped, which only
// a seat that called another model could give. Its two examples are valid
// messages for TASK_ID xreview-org-a-c1, the form the leader sends
// (xreview-<org_id>-c<k>), with those body lines and no such header.
func TestRenderRolePrompt_Reviewer_XreviewAnswersBlockedOnlyForActionRequired(t *testing.T) {
	step := reviewerXreviewStep(t, 7)
	assertPhrasesInOrder(t, step, reviewerXreviewStepWhere(7), []string{
		"TASK_ID は受けた TASK の TASK_ID にする",
		"`ACTION_REQUIRED` が 1 件以上なら BLOCKED を返し、本文に `XREVIEW: action_required` と `REPORT: <report のパス>` の行を書く",
		"それ以外は RESULT を返し、本文に `XREVIEW: pass`(差分を読み終え、`ACTION_REQUIRED` が 0 件)か `XREVIEW: incomplete (<理由>)` の行と、`REPORT: <report のパス>` の行を書く",
		"(git の状態が変わったときの `XREVIEW: git_changed` の BLOCKED と、1 の QUESTION は、それぞれ 5 と 1 のとおりに返す)",
		"`XREVIEW:` と `REPORT:` は、`CHECKS:` と同じく本文の行に書く",
	})

	section := reviewerXreviewSection(t)
	if strings.Contains(section, "XREVIEW: skipped") {
		t.Errorf("expected %s to return no XREVIEW: skipped, since the seat calls no other model, got:\n%s", reviewerXreviewWhere, section)
	}
	const report = "REPORT: docs/reports/review-xreview-<YYYY-MM-DD>-org-a-c1.md"
	for _, c := range []struct{ lead, typ, xreview string }{
		{reviewerXreviewResultLead, protocol.TypeResult, "XREVIEW: pass"},
		{reviewerXreviewBlockedLead, protocol.TypeBlocked, "XREVIEW: action_required"},
	} {
		msg, found := codeBlockAfterLine(section, c.lead)
		if !found {
			t.Errorf("expected %s to show a code block after a line starting with %q, got section:\n%s", reviewerXreviewWhere, c.lead, section)
			continue
		}
		m, err := protocol.Parse(msg)
		if err != nil || protocol.Validate(m, protocol.DefaultMaxBodyChars) != nil || m.Type != c.typ || m.TaskID != "xreview-org-a-c1" {
			t.Errorf("expected the example after %q to be a valid %s with TASK_ID xreview-org-a-c1, got %+v, err %v; message:\n%s", c.lead, c.typ, m, err, msg)
			continue
		}
		for _, key := range []string{"XREVIEW", "REPORT"} {
			if got := protocol.HeaderValues(msg, key); len(got) != 0 {
				t.Errorf("expected %s: in the body of the example after %q, not its header block, got header values %q; message:\n%s", key, c.lead, got, msg)
			}
		}
		for _, line := range []string{c.xreview, report} {
			if !hasLine(m.Body, line) {
				t.Errorf("expected the body of the example after %q to have the line %s, got body:\n%s", c.lead, line, m.Body)
			}
		}
	}
}

// TestRenderRolePrompt_Reviewer_XreviewReportContractNamesTheReport pins D4:
// the reviewer's report contract has the cross-review report written at
// REPORT:'s path, the one the section's REPORT: line names, which starts with
// reviewerReportPrefix (so scripts/check-sync.sh excludes it) and xreview-, not
// committed, and never under the standard flow's cross-review-triage- name,
// which the template names nowhere as a path.
func TestRenderRolePrompt_Reviewer_XreviewReportContractNamesTheReport(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	contract, _ := markdownSection(text, "## レポート契約")
	item, ok := markdownItem(contract, "- cross-review の TASK")
	if !ok {
		t.Fatalf("expected the reviewer's report contract to have an item for the cross-review TASK, got section:\n%s", contract)
	}
	const path = "docs/reports/review-xreview-<YYYY-MM-DD>-org-a-c<k>.md"
	assertContainsPhrases(t, item, "the report contract's cross-review item", []string{
		"cross-review の TASK(TASK_ID が `xreview-` で始まる)の report は、TASK の `REPORT:` の行のパス `" + path + "` に、「cross-review の TASK」の節の形で書き、コミットしないでください(leader がコミットします)",
		"標準フローの `/cross-review` の report の名前(`cross-review-triage-` で始まる名前)は、標準フローの report として読まれるので使わないでください",
	})
	if !strings.HasPrefix(path, reviewerReportPrefix+"xreview-") {
		t.Errorf("expected the cross-review report %q to start with %q, which scripts/check-sync.sh excludes", path, reviewerReportPrefix+"xreview-")
	}
	intro, _, _ := strings.Cut(reviewerXreviewSection(t), "\n1. ")
	if !strings.Contains(intro, "`REPORT: "+path+"`") {
		t.Errorf("expected the REPORT: line of %s to name %s, as the report contract does, got:\n%s", reviewerXreviewWhere, path, intro)
	}
	if strings.Contains(text, defaultReportDir+"/cross-review-triage-") {
		t.Errorf("expected the reviewer template to name no report under %s/cross-review-triage-, got:\n%s", defaultReportDir, text)
	}
}

// The fixture of the tests below runs the section's commands with bash and
// zsh, as the leader's plan advisory tests do (prompts_leader_plan_test.go).

// reviewerXreviewCallStub stands in for codex and claude: it logs its name
// and each argument, one per line, then --end--, and exits 0. The section's
// commands call neither, so a test that finds a call in the log fails.
const reviewerXreviewCallStub = `#!/bin/sh
{ basename "$0"; for a in "$@"; do printf '%s\n' "$a"; done; echo --end--; } >> "$STUB_LOG"
exit 0
`

// reviewerXreviewDiffFile is the diff file step 2 writes for the TASK of
// reviewerXreviewTaskBody, relative to the repository.
const reviewerXreviewDiffFile = ".harness/state/xreview-org-a-c1.diff"

// reviewerXreviewFixture is an advisoryFixture whose repository has a branch
// one commit (feature.txt) ahead of origin/main, and whose bin holds
// reviewerXreviewCallStub as codex and as claude, first on PATH, so that a
// call of either is logged and none on the host runs.
type reviewerXreviewFixture struct {
	advisoryFixture
}

func newReviewerXreviewFixture(t *testing.T) reviewerXreviewFixture {
	t.Helper()
	dir := t.TempDir()
	fx := reviewerXreviewFixture{
		advisoryFixture: advisoryFixture{repo: filepath.Join(dir, "repo"), bin: filepath.Join(dir, "bin"), log: filepath.Join(dir, "calls.txt")},
	}
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
	fx.commit(t, "base")
	runGit(t, "-C", fx.repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	if err := os.WriteFile(filepath.Join(fx.repo, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", fx.repo, "add", "feature.txt")
	fx.commit(t, "feature")
	writeTestExecutable(t, filepath.Join(fx.bin, "codex"), reviewerXreviewCallStub)
	writeTestExecutable(t, filepath.Join(fx.bin, "claude"), reviewerXreviewCallStub)
	return fx
}

// commit commits what is staged in the fixture's repository with message.
func (fx reviewerXreviewFixture) commit(t *testing.T, message string) {
	t.Helper()
	runGit(t, "-C", fx.repo, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", message)
}

// reviewerXreviewTaskBody is the body of a cross-review TASK as the leader
// writes it, with base as XREVIEW_BASE:.
func reviewerXreviewTaskBody(base string) string {
	return strings.Join([]string{
		"XREVIEW_BASE: " + base,
		"CYCLE: 1/2",
		"REPORT: docs/reports/review-xreview-2026-11-01-org-a-c1.md",
	}, "\n") + "\n"
}

// reviewerXreviewEarlierTaskBody is the body of the same TASK as an earlier
// leader wrote it, for a seat that called the other model: the lines of
// reviewerXreviewTaskBody with the implementer's driver and the two reviewer
// lines around them.
func reviewerXreviewEarlierTaskBody(base string) string {
	return "IMPLEMENTER_DRIVER: claude\n" + reviewerXreviewTaskBody(base) +
		"REVIEWER_CODEX: stub-model:v1 stub-effort\nREVIEWER_CLAUDE: us.anthropic.stub-claude-v1:0 max\n"
}

// reviewerXreviewLine returns the value of body's line that starts with lead
// and a space: the rest of the line, read after the lead.
func reviewerXreviewLine(t *testing.T, body, lead string) string {
	t.Helper()
	for line := range strings.SplitSeq(body, "\n") {
		if v, ok := strings.CutPrefix(line, lead+" "); ok {
			return v
		}
	}
	t.Fatalf("expected the TASK body to have a line starting with %q, got:\n%s", lead, body)
	return ""
}

// reviewerXreviewPlaceholder matches a placeholder left in a filled-in
// command: a < that a word follows, not a redirection.
var reviewerXreviewPlaceholder = regexp.MustCompile(`<[^\s/"$<>][^<>\n]*>`)

// reviewerXreviewReportNumber reads <k> off the end of REPORT:'s path.
var reviewerXreviewReportNumber = regexp.MustCompile(`-c(\d+)\.md$`)

// reviewerXreviewCommand returns the command of step n that follows lead,
// with each placeholder replaced as the section says, from the TASK body:
// <k> with the number REPORT:'s path ends with and <XREVIEW_BASE> with its
// line's value. It fails the test when a placeholder is left.
func reviewerXreviewCommand(t *testing.T, n int, lead, body string) string {
	t.Helper()
	k := reviewerXreviewReportNumber.FindStringSubmatch(reviewerXreviewLine(t, body, "REPORT:"))
	if k == nil {
		t.Fatalf("expected a REPORT: path ending -c<k>.md, got:\n%s", body)
	}
	block := fencedBlockAfter(t, reviewerXreviewStep(t, n), lead, reviewerXreviewStepWhere(n))
	filled := strings.NewReplacer("<k>", k[1], "<XREVIEW_BASE>", reviewerXreviewLine(t, body, "XREVIEW_BASE:")).Replace(block)
	if left := reviewerXreviewPlaceholder.FindAllString(filled, -1); len(left) > 0 {
		t.Fatalf("expected every placeholder filled in, %q left in:\n%s", left, filled)
	}
	return filled
}

// checkBase runs step 1's command that checks XREVIEW_BASE:'s value, filled
// in with value, in the fixture's repository with shell and env and returns
// what it prints, trimmed.
func (fx reviewerXreviewFixture) checkBase(t *testing.T, shell, value string, env ...string) string {
	t.Helper()
	block := fencedBlockAfter(t, reviewerXreviewStep(t, 1), reviewerXreviewBaseLead, reviewerXreviewStepWhere(1))
	filled := strings.ReplaceAll(block, "<XREVIEW_BASE>", value)
	if left := reviewerXreviewPlaceholder.FindAllString(filled, -1); filled == block || len(left) > 0 {
		t.Fatalf("expected step 1's check to hold <XREVIEW_BASE> and no other placeholder, %q left in:\n%s", left, filled)
	}
	return strings.TrimSpace(fx.run(t, shell, filled, env...))
}

// TestRenderRolePrompt_Reviewer_XreviewCommandsWriteTheStateAndTheDiff runs
// step 2's and step 5's commands with bash and zsh in a repository whose
// branch is one commit ahead of origin/main, codex and claude stand-ins first
// on PATH (AC1 and AC2 of plan 2026-11-22-org-xreview-own-model): step 2
// prints diff: written, writes the branch's diff since origin/main and the
// git state starting with HEAD; step 5 prints git: same and writes the same
// state; neither calls codex or claude, and the work tree stays clean (the
// files are under the ignored .harness/state/). The commands filled in from
// the TASK an earlier leader wrote, with IMPLEMENTER_DRIVER: and the two
// reviewer lines, are the same as from the three lines, so that TASK works
// too.
func TestRenderRolePrompt_Reviewer_XreviewCommandsWriteTheStateAndTheDiff(t *testing.T) {
	body := reviewerXreviewTaskBody("origin/main")
	diffCmd := reviewerXreviewCommand(t, 2, reviewerXreviewDiffLead, body)
	afterCmd := reviewerXreviewCommand(t, 5, reviewerXreviewAfterLead, body)
	earlier := reviewerXreviewEarlierTaskBody("origin/main")
	if reviewerXreviewCommand(t, 2, reviewerXreviewDiffLead, earlier) != diffCmd || reviewerXreviewCommand(t, 5, reviewerXreviewAfterLead, earlier) != afterCmd {
		t.Errorf("expected the commands filled in from an earlier leader's TASK to be the same as from the three lines")
	}
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newReviewerXreviewFixture(t)
			if out := fx.run(t, shell, diffCmd); strings.TrimSpace(out) != "diff: written" {
				t.Errorf("step 2's output = %q, want diff: written", out)
			}
			if diff := fx.read(t, reviewerXreviewDiffFile); !strings.Contains(diff, "+++ b/feature.txt") || !strings.Contains(diff, "\n+feature\n") {
				t.Errorf("diff file = %q, want the branch's diff since origin/main (feature.txt)", diff)
			}
			if out := fx.run(t, shell, afterCmd); strings.TrimSpace(out) != "git: same" {
				t.Errorf("step 5's output = %q, want git: same", out)
			}
			before, after := fx.read(t, ".harness/state/xreview-org-a-c1-git-before.txt"), fx.read(t, ".harness/state/xreview-org-a-c1-git-after.txt")
			if head := gitOut(t, fx.repo, "rev-parse", "HEAD"); !strings.HasPrefix(before, head+"\n") || before != after {
				t.Errorf("git state before = %q, after = %q; want the same state, starting with HEAD %s", before, after, head)
			}
			if calls := fx.calls(t); len(calls) != 0 {
				t.Errorf("calls = %q, want no codex or claude", calls)
			}
			if status := gitOut(t, fx.repo, "status", "--porcelain", "--untracked-files=all"); status != "" {
				t.Errorf("git status after the commands = %q, want a clean tree (their files are under the ignored .harness/state/)", status)
			}
		})
	}
}

// reviewerXreviewChanges are the changes of the repository between step 2
// and step 5 that step 5 reports as git: changed, each with git status after
// it (trimmed, as gitOut returns it): an edit of a tracked file, a new
// untracked file, an edit of an untracked file there before step 2 (git
// status shows the same ?? line before and after it, and only the hashes of
// the untracked files' contents in the git state see the change), and a new
// commit, which moves HEAD and leaves the tree clean.
var reviewerXreviewChanges = []struct {
	name, status string
	before       func(t *testing.T, fx reviewerXreviewFixture)
	change       func(t *testing.T, fx reviewerXreviewFixture)
}{
	{"an edit of a tracked file", "M tracked.txt", nil, func(t *testing.T, fx reviewerXreviewFixture) {
		reviewerXreviewWrite(t, fx, "tracked.txt", "one\ntwo\n")
	}},
	{"a new untracked file", "?? new.txt", nil, func(t *testing.T, fx reviewerXreviewFixture) {
		reviewerXreviewWrite(t, fx, "new.txt", "one\n")
	}},
	{"an edit of an untracked file", "?? untracked.txt", func(t *testing.T, fx reviewerXreviewFixture) {
		reviewerXreviewWrite(t, fx, "untracked.txt", "one\n")
	}, func(t *testing.T, fx reviewerXreviewFixture) {
		reviewerXreviewWrite(t, fx, "untracked.txt", "two\n")
	}},
	{"a new commit", "", nil, reviewerXreviewCommitAnEdit},
}

// reviewerXreviewAfterAFailedDiff are the runs of step 5 after step 2 printed
// diff: failed for a base git diff cannot read, which sends the seat past 3
// and 4 to step 5's comparison (the cross-review finding of the branch,
// docs/reports/cross-review-triage-org-xreview-own-model.md; test gap 1 of
// cycle 2 in docs/reports/test-2026-11-22-org-xreview-own-model.md). The
// first run changes nothing in between and wants git: same, the way to the
// incomplete report: it holds only while step 2 keeps the state it wrote
// after the failure and step 5 compares that state without reading the
// empty diff file (the order of step 2's lines is pinned by the command-line
// match tests, not here). The second run commits in between and
// wants git: changed, the way to the BLOCKED: it holds only while step 5
// compares the state at all after diff: failed. Neither run leaves a file git
// sees in the work tree.
var reviewerXreviewAfterAFailedDiff = []struct {
	name, step5 string
	change      func(t *testing.T, fx reviewerXreviewFixture)
}{
	{"step 5 after diff: failed", "git: same", nil},
	{"step 5 after diff: failed and a new commit", "git: changed", reviewerXreviewCommitAnEdit},
}

// reviewerXreviewWrite writes content to rel in the fixture's repository.
func reviewerXreviewWrite(t *testing.T, fx reviewerXreviewFixture, rel, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(fx.repo, rel), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// reviewerXreviewCommitAnEdit commits an edit of tracked.txt in the fixture's
// repository, which moves HEAD and leaves the tree clean.
func reviewerXreviewCommitAnEdit(t *testing.T, fx reviewerXreviewFixture) {
	t.Helper()
	reviewerXreviewWrite(t, fx, "tracked.txt", "one\ntwo\n")
	runGit(t, "-C", fx.repo, "add", "tracked.txt")
	fx.commit(t, "a commit between step 2 and step 5")
}

// TestRenderRolePrompt_Reviewer_XreviewCommandOutcomes runs step 2's and step
// 5's commands with bash and zsh for the outcomes the steps sort: a base git
// diff cannot read prints git diff's stderr (unknown revision), which step 2
// puts in the incomplete reason, and diff: failed; a base that would read as
// an option (--output=<file>, which step 1 refuses first) prints diff: failed
// and writes no file (verify cycle 1, V2, of plan
// 2026-11-01-org-cross-review); step 5 run after step 2's diff: failed for a
// base git diff cannot read prints what reviewerXreviewAfterAFailedDiff says
// and leaves the work tree clean; and each change of reviewerXreviewChanges
// made between step 2 and step 5 makes step 5 print git: changed (AC1 of plan
// 2026-11-22-org-xreview-own-model: the state is compared before the reading
// and before the report). No case calls codex or claude.
func TestRenderRolePrompt_Reviewer_XreviewCommandOutcomes(t *testing.T) {
	diffCmd := reviewerXreviewCommand(t, 2, reviewerXreviewDiffLead, reviewerXreviewTaskBody("origin/main"))
	afterCmd := reviewerXreviewCommand(t, 5, reviewerXreviewAfterLead, reviewerXreviewTaskBody("origin/main"))
	goneBody := reviewerXreviewTaskBody("origin/gone")
	goneDiffCmd := reviewerXreviewCommand(t, 2, reviewerXreviewDiffLead, goneBody)
	goneAfterCmd := reviewerXreviewCommand(t, 5, reviewerXreviewAfterLead, goneBody)
	for _, shell := range advisoryShells(t, false) {
		name := filepath.Base(shell)
		t.Run(name+"/a base git diff cannot read", func(t *testing.T) {
			t.Parallel()
			fx := newReviewerXreviewFixture(t)
			// LC_ALL=C keeps git's message in English, whatever the host's locale.
			if out := fx.run(t, shell, goneDiffCmd, "LC_ALL=C"); !strings.Contains(out, "unknown revision") || !hasLine(out, "diff: failed") {
				t.Errorf("output = %q; want git diff's stderr (unknown revision) and diff: failed", out)
			}
			if calls := fx.calls(t); len(calls) != 0 {
				t.Errorf("calls = %q; want none", calls)
			}
		})
		t.Run(name+"/a base that reads as an option", func(t *testing.T) {
			t.Parallel()
			fx := newReviewerXreviewFixture(t)
			written := t.TempDir()
			block := reviewerXreviewCommand(t, 2, reviewerXreviewDiffLead, reviewerXreviewTaskBody("--output="+filepath.Join(written, "diff")))
			if out := fx.run(t, shell, block); !hasLine(out, "diff: failed") {
				t.Errorf("output = %q; want diff: failed", out)
			}
			if entries, err := os.ReadDir(written); err != nil || len(entries) != 0 {
				t.Errorf("expected git diff to write no file for an XREVIEW_BASE: of --output=<file>, got %v, err %v", entries, err)
			}
		})
		for _, c := range reviewerXreviewAfterAFailedDiff {
			t.Run(name+"/"+c.name, func(t *testing.T) {
				t.Parallel()
				fx := newReviewerXreviewFixture(t)
				if out := fx.run(t, shell, goneDiffCmd); !hasLine(out, "diff: failed") {
					t.Fatalf("step 2's output = %q, want diff: failed", out)
				}
				if c.change != nil {
					c.change(t, fx)
				}
				if out := fx.run(t, shell, goneAfterCmd); strings.TrimSpace(out) != c.step5 {
					t.Errorf("step 5's output = %q, want %s", out, c.step5)
				}
				if status := gitOut(t, fx.repo, "status", "--porcelain", "--untracked-files=all"); status != "" {
					t.Errorf("git status after steps 2 and 5 = %q, want a clean tree (their files are under the ignored .harness/state/)", status)
				}
				if calls := fx.calls(t); len(calls) != 0 {
					t.Errorf("calls = %q; want none", calls)
				}
			})
		}
		for _, c := range reviewerXreviewChanges {
			t.Run(name+"/"+c.name, func(t *testing.T) {
				t.Parallel()
				fx := newReviewerXreviewFixture(t)
				if c.before != nil {
					c.before(t, fx)
				}
				if out := fx.run(t, shell, diffCmd); strings.TrimSpace(out) != "diff: written" {
					t.Fatalf("step 2's output = %q, want diff: written", out)
				}
				c.change(t, fx)
				if status := gitOut(t, fx.repo, "status", "--porcelain", "--untracked-files=all"); status != c.status {
					t.Fatalf("git status after the change = %q, want %q", status, c.status)
				}
				if out := fx.run(t, shell, afterCmd); strings.TrimSpace(out) != "git: changed" {
					t.Errorf("step 5's output after %s = %q, want git: changed", c.name, out)
				}
				if calls := fx.calls(t); len(calls) != 0 {
					t.Errorf("calls = %q; want none", calls)
				}
			})
		}
	}
}

// TestRenderRolePrompt_Reviewer_XreviewBaseCheckRunsGitCheckRefFormat pins fix F7
// (cross-review cycle 1's ACTION_REQUIRED #2, after F6, the second live
// probe's codex P2): step 1 asks with a QUESTION when its command prints
// base: invalid for XREVIEW_BASE:'s value. The command reads the value from a
// heredoc whose delimiter is quoted, so the shell expands nothing in it, and
// passes it when it starts with origin/ and git check-ref-format --branch
// takes the name after it, which has none of what the command refuses before
// it calls git. Run with bash and zsh, it passes origin/main, a path, and the
// names of reviewerXreviewBaseNames; it refuses a value without origin/ or a
// name after it, names git refuses (a.lock, ~, ^, HEAD, a backslash), a name
// that starts like an option, one with .. or @{ (origin/@{-1} too, which git
// check-ref-format --branch reads as the branch checked out before and
// passes), white space or a control character, and names git takes with a
// quote, $ or a backquote (step 2 puts the value in double quotes, where a
// shell reads $ and the backquote); a value with a command substitution runs
// nothing. Typed with the list's indentation, the heredoc does not close and
// the command prints neither base: ok nor base: invalid, which step 1 tells
// the reviewer to type again without the leading spaces, not to ask about
// with a QUESTION (fix F8, self-review cycle 2's C2-2). Step 1 no longer
// asks when the command does not print base: ok, the condition before F8,
// so a reviewerXreviewBaseRule changed back with the template still fails
// (test gap 3 of the test cycle 2 report).
func TestRenderRolePrompt_Reviewer_XreviewBaseCheckRunsGitCheckRefFormat(t *testing.T) {
	assertPhrasesInOrder(t, reviewerXreviewStep(t, 1), reviewerXreviewStepWhere(1), []string{
		reviewerXreviewBaseRule,
		"差分を読まず、何も書かずに QUESTION で leader に聞く",
		reviewerXreviewBaseLead + "、まとめて 1 回の shell で打って確かめる",
		"`<XREVIEW_BASE>` の行を、値だけの 1 行に書き換える",
		"どの行も行の頭に空白を付けずに打つ(空白があると heredoc が閉じず、値の頭にも空白が入って `base: ok` が出ない)",
		"値は区切りを引用符で囲んだ heredoc で読むので、shell は値の中の文字を展開しない(値に `'` があってもコマンドは崩れない)",
		"コマンドは、値が `origin/` で始まり、そのあとのブランチ名が空でなく、`-` で始まらず(option として読まれる)、" +
			"`..` と `@{`(revision の式になる。`git check-ref-format --branch` は `@{-1}` を前のブランチの名前に読み替えて通す)、" +
			"空白と制御文字、引用符と `$`・`` ` ``・`\\`(下のコマンドは値を二重引用符で囲んで渡す)を持たず、" +
			"`git check-ref-format --branch` が通すときだけ `base: ok` を出す",
		"git がブランチ名に許す `release+stable`・`release=stable`・`release(v2)`・`dévelop` も通る",
		"コマンドが `base: ok` も `base: invalid` も出さなければ(heredoc が閉じなかった。たいていは行の頭に空白を付けて打ったため)、値はまだ確かめられていない",
		"行の頭の空白を外してコマンドを打ち直し、このときは QUESTION を送らない",
	})
	if step, old := reviewerXreviewStep(t, 1), "`base: ok` を出さないか"; containsPhrase(step, old) {
		t.Errorf("expected %s to ask with a QUESTION only when the check prints base: invalid, but it still has %q (the condition before fix F8, "+
			"which also asks about a heredoc left open that the step types again), got step:\n%s", reviewerXreviewStepWhere(1), old, step)
	}
	passed := append([]string{"origin/main", "origin/feat/x.y_z-1"}, reviewerXreviewBaseNames...)
	refused := []string{
		"origin/a@{1}", "origin/@{-1}", "origin/a..b", "origin/-x", "origin/-",
		"origin/a b", "origin/a\tb", "origin/a\x01b", "origin/a.lock", "origin/HEAD",
		"origin/", "feature/x", "main", "--output=x",
		"origin/a\"b", "origin/a'b", "origin/a$b", "origin/a`b", "origin/a\\b", "origin/a~1", "origin/a^",
		"origin/a$(touch${IFS}pwned)", "origin/a`touch${IFS}pwned`",
	}
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newReviewerXreviewFixture(t)
			// A branch checked out before, which @{-1} names.
			runGit(t, "-C", fx.repo, "checkout", "--quiet", "-b", "other")
			runGit(t, "-C", fx.repo, "checkout", "--quiet", "-")
			if out, err := exec.Command("git", "-C", fx.repo, "check-ref-format", "--branch", "@{-1}").CombinedOutput(); err != nil {
				t.Fatalf("expected git check-ref-format --branch to pass @{-1} after a checkout of another branch, which step 1's check must refuse before it calls git: %v\n%s", err, out)
			}
			for _, value := range passed {
				if got := fx.checkBase(t, shell, value); got != "base: ok" {
					t.Errorf("step 1's check of XREVIEW_BASE: %q printed %q, want base: ok", value, got)
				}
			}
			for _, value := range refused {
				if got := fx.checkBase(t, shell, value); got != "base: invalid" {
					t.Errorf("step 1's check of XREVIEW_BASE: %q printed %q, want base: invalid", value, got)
				}
			}
			if _, err := os.Stat(filepath.Join(fx.repo, "pwned")); !os.IsNotExist(err) {
				t.Errorf("expected step 1's check to run no command substitution in the value, got pwned in the repository (err %v)", err)
			}
			block := fencedBlockAfter(t, reviewerXreviewStep(t, 1), reviewerXreviewBaseLead, reviewerXreviewStepWhere(1))
			indented := "   " + strings.ReplaceAll(strings.ReplaceAll(block, "<XREVIEW_BASE>", "origin/main"), "\n", "\n   ")
			if out := fx.run(t, shell, indented); strings.Contains(out, "base: ok") || strings.Contains(out, "base: invalid") {
				t.Errorf("expected step 1's check typed with the list's indentation, which leaves the heredoc open, to print neither base: ok nor base: invalid, got %q", out)
			}
		})
	}
}

// TestRenderRolePrompt_Reviewer_XreviewBaseCheckRefusesBeforeItCallsGit runs
// step 1's check of XREVIEW_BASE: with bash and zsh and, first on PATH, a git
// that takes every name (test gap 2 of the test cycle 2 report): the
// command's case is then all that refuses, so a value without origin/ or a
// name after it, a name that starts with -, and names with .., @{, white
// space or a control character, a quote, $, a backquote or a backslash still
// print base: invalid, though git check-ref-format --branch refuses most of
// them too and would hide a case that no longer does. origin/main prints
// base: ok, and so do origin/a.lock and origin/HEAD, which the real git
// refuses, so the stand-in is the git the command called.
func TestRenderRolePrompt_Reviewer_XreviewBaseCheckRefusesBeforeItCallsGit(t *testing.T) {
	passed := []string{"origin/main", "origin/a.lock", "origin/HEAD"}
	refused := []string{
		"origin/", "feature/x", "main", "--output=x", "origin/-x", "origin/-", "origin/a..b",
		"origin/a@{1}", "origin/@{-1}", "origin/a b", "origin/a\tb", "origin/a\x01b",
		"origin/a\"b", "origin/a'b", "origin/a$b", "origin/a`b", "origin/a\\b",
	}
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newReviewerXreviewFixture(t)
			stub := t.TempDir()
			writeTestExecutable(t, filepath.Join(stub, "git"), "#!/bin/sh\nexit 0\n")
			path := "PATH=" + stub + string(os.PathListSeparator) + fx.bin + string(os.PathListSeparator) + os.Getenv("PATH")
			for _, value := range passed {
				if got := fx.checkBase(t, shell, value, path); got != "base: ok" {
					t.Errorf("step 1's check of XREVIEW_BASE: %q with a git that takes every name printed %q, want base: ok", value, got)
				}
			}
			for _, value := range refused {
				if got := fx.checkBase(t, shell, value, path); got != "base: invalid" {
					t.Errorf("step 1's check of XREVIEW_BASE: %q with a git that takes every name printed %q, want base: invalid from the check's case", value, got)
				}
			}
		})
	}
}

// TestRenderRolePrompt_Reviewer_XreviewBaseNamesWorkInTheCommands runs step
// 2's command with bash and zsh with each name of reviewerXreviewBaseNames as
// XREVIEW_BASE: (fixes F6 and F7), with refs of those names under origin/:
// it writes the branch's diff from them with git diff --end-of-options and
// prints diff: written.
func TestRenderRolePrompt_Reviewer_XreviewBaseNamesWorkInTheCommands(t *testing.T) {
	for _, shell := range advisoryShells(t, false) {
		name := filepath.Base(shell)
		for _, base := range reviewerXreviewBaseNames {
			t.Run(name+"/"+base, func(t *testing.T) {
				t.Parallel()
				fx := newReviewerXreviewFixture(t)
				runGit(t, "-C", fx.repo, "update-ref", "refs/remotes/"+base, "refs/remotes/origin/main")
				block := reviewerXreviewCommand(t, 2, reviewerXreviewDiffLead, reviewerXreviewTaskBody(base))
				if out := fx.run(t, shell, block); strings.TrimSpace(out) != "diff: written" {
					t.Errorf("output = %q; want diff: written", out)
				}
				if diff := fx.read(t, reviewerXreviewDiffFile); !strings.Contains(diff, "+++ b/feature.txt") {
					t.Errorf("diff file = %q, want the branch's diff since %s (feature.txt)", diff, base)
				}
			})
		}
	}
}

// reviewerFanOutSameModelLead starts the rule of the reviewer template's
// ## 座席内 fan-out that keeps the review's judgment on the seat's own model.
const reviewerFanOutSameModelLead = "- レビューの判断"

// TestRenderRolePrompt_Reviewer_FanOutKeepsTheReviewOnTheSeatsModel pins the
// user's decision of 2026-11-21 behind plan 2026-11-22-org-xreview-own-model
// (slice S3) for the reviewer template's ## 座席内 fan-out: in the org runtime
// the model the reviewer seat was given reviews, so a child subagent that
// takes part of the review's judgment (an aspect, a file, the cross-review
// TASK's findings and triage) runs on the seat's own model (the driver: and
// model: line at the top), with no cheaper model named on Claude Code's Task
// tool, no agent definition fixed to another model, and on Codex only custom
// agents of the seat's model. A cheaper child may only run the review TASK's
// gate commands, the third example. The rule is a bullet of the section
// before its examples, and of the examples only the gate's names a cheaper
// model, while the ones that split the review by aspect or by file do not.
func TestRenderRolePrompt_Reviewer_FanOutKeepsTheReviewOnTheSeatsModel(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	fanOut, found := markdownSection(text, "## 座席内 fan-out")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## 座席内 fan-out' section, got:\n%s", text)
	}
	rules, examples, found := strings.Cut(fanOut, "\n例:\n")
	if !found {
		t.Fatalf("expected the reviewer's fan-out section to list its examples after a line 例:, got section:\n%s", fanOut)
	}
	rule, ok := markdownItem(rules, reviewerFanOutSameModelLead)
	if !ok {
		t.Fatalf("expected the reviewer's fan-out section to have a rule starting %q before its examples, got section:\n%s", reviewerFanOutSameModelLead, fanOut)
	}
	assertPhrasesInOrder(t, rule, "the reviewer's fan-out rule on the review's model", []string{
		"レビューの判断(観点ごとやファイルごとのレビュー、cross-review の TASK で出す所見と triage)を子サブエージェントに分けるときは、" +
			"子をこの座席と同じモデル(冒頭の `driver:` と `model:` の行)で動かしてください",
		"Claude Code の `Task` ツールで子に安いモデルを指定したり、別のモデルが決めてあるエージェント定義を使ったりせず、" +
			"Codex でもこの座席と同じモデルのカスタムエージェントだけを使ってください",
		"レビューを担うのはこの座席のモデルです",
		"安いモデルの子に任せてよいのは、レビューの TASK のゲートのコマンドを打つことだけです(下の例の 3 つ目)",
	})
	if !strings.Contains(text, "\n- driver: ") {
		t.Errorf("expected the reviewer template to have the driver: and model: line the fan-out rule points at")
	}

	var items []string
	for _, line := range strings.Split(examples, "\n") {
		switch {
		case strings.HasPrefix(line, "- "):
			items = append(items, line)
		case len(items) > 0 && strings.HasPrefix(line, "  "):
			items[len(items)-1] += "\n" + line
		}
	}
	if len(items) < 3 || !containsPhrase(items[2], "レビューの TASK のゲートの実行") {
		t.Fatalf("expected the third example of the reviewer's fan-out section, which the rule names, to be the review TASK's gate run, got examples:\n%s", examples)
	}
	for i, item := range items {
		if containsPhrase(item, "安いモデル") && i != 2 {
			t.Errorf("expected only the gate's example to hand work to a cheaper model, got example %d:\n%s", i+1, item)
		}
	}
	for _, split := range []string{"レビュー観点ごとに子サブエージェントへ並行レビューさせ", "ファイルまたはモジュール単位で子サブエージェントに分担させ"} {
		if !containsPhrase(examples, split) {
			t.Errorf("expected the reviewer's fan-out examples to keep splitting the review (%q), which the rule keeps on the seat's model, got examples:\n%s", split, examples)
		}
	}
}

// reviewerFanOutNoWriteLead starts the rule of the reviewer template's
// ## 座席内 fan-out that keeps a child subagent's files out of the worktree
// between steps 2 and 5 of the cross-review TASK.
const reviewerFanOutNoWriteLead = "- cross-review の TASK の 2 から 5 のあいだ"

// TestRenderRolePrompt_Reviewer_FanOutKeepsTheChildrensFilesOutOfTheWorktree
// pins the rule of the reviewer template's ## 座席内 fan-out (self-review L3
// and C2-2 of plan 2026-11-22-org-xreview-own-model): a child subagent the
// seat uses between steps 2 and 5 of the cross-review TASK, for the findings
// of 3 or the triage of 4, writes nothing in the worktree and keeps its work
// files in a temporary directory outside it such as $TMPDIR. Each place in
// the worktree has its reason: a file git does not ignore makes step 5's
// comparison answer git: changed, and the ignored .harness/state/ is where
// the seat writes the git state files of 2 and 5 and the diff file, which a
// child's files could be mixed up with. The rule is a bullet of the section
// before its examples, so it holds for the triage as for the findings, and
// step 3 points at it without repeating where the files go.
func TestRenderRolePrompt_Reviewer_FanOutKeepsTheChildrensFilesOutOfTheWorktree(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	fanOut, found := markdownSection(text, "## 座席内 fan-out")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## 座席内 fan-out' section, got:\n%s", text)
	}
	rules, _, found := strings.Cut(fanOut, "\n例:\n")
	if !found {
		t.Fatalf("expected the reviewer's fan-out section to list its examples after a line 例:, got section:\n%s", fanOut)
	}
	rule, ok := markdownItem(rules, reviewerFanOutNoWriteLead)
	if !ok {
		t.Fatalf("expected the reviewer's fan-out section to have a rule starting %q before its examples, got section:\n%s", reviewerFanOutNoWriteLead, fanOut)
	}
	assertPhrasesInOrder(t, rule, "the reviewer's fan-out rule on a child's files", []string{
		"cross-review の TASK の 2 から 5 のあいだ(3 で所見を出す、4 で triage する)に子サブエージェントを使うときは、子に worktree の中に何も書かせず、" +
			"子の作業ファイルは worktree の外の一時ディレクトリ(`$TMPDIR` など)に置かせてください",
		"git が無視しないファイルが worktree にできると、5 の比べで `git: changed` になります",
		"git が無視する `.harness/state/` にも置かせないでください",
		"そこはこの座席が 2 と 5 で git の状態のファイル(`-git-before.txt` と `-git-after.txt`)と差分のファイルを書く所で、" +
			"子のファイルがそれらとまぎれうるからです",
	})
	if step := reviewerXreviewStep(t, 3); containsPhrase(step, "$TMPDIR") || containsPhrase(step, "まぎれ") {
		t.Errorf("expected %s to point at the fan-out rule rather than repeat where a child's files go, got:\n%s", reviewerXreviewStepWhere(3), step)
	}
}
