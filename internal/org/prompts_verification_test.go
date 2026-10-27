package org

import (
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// implementerGoSliceCheck is the narrowest check the implementer template
// names for a Go slice with no VERIFY: line, <dir> standing for the touched
// package's directory. The templates' prose writes it in backticks, and
// their examples run it on internal/foo.
const implementerGoSliceCheck = `test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && go test ./<dir>/`

// implementerSelfCheckItems are the five items of the implementer's
// mechanical self-check, in the template's order. They mirror the standard
// flow's list (unnecessary changes, typos and copy-paste errors, debug
// leftovers, secrets, files outside the slice).
var implementerSelfCheckItems = []string{"不要な変更", "誤字とコピペの誤り", "デバッグの残り", "秘密情報", "scope の外のファイル"}

// TestRenderRolePrompt_Implementer_MissionRunsTheSliceCommand pins the
// verification half of AC2 of plan 2026-10-27-org-narrow-verification: after
// each slice the implementer runs the command on the VERIFY: line of the
// leader's TASK body; with no VERIFY: line it picks the narrowest check that
// covers the slice's files (for Go, the touched package's gofmt, vet and test,
// with test -z around gofmt -l), says RALPH_VERIFY_SCOPE=changed does not
// narrow, and keeps ./scripts/run-verify.sh for when nothing narrower covers
// the files; it writes the command in the RESULT and stages only after the
// check passes. ./scripts/run-verify.sh is named nowhere else in the
// template, so the old default (指示がなければ `./scripts/run-verify.sh`)
// cannot come back elsewhere.
func TestRenderRolePrompt_Implementer_MissionRunsTheSliceCommand(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "作業はスライス単位で進める"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the implementer mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	for _, want := range []string{
		"各スライスの実装後、leader の TASK の本文の `VERIFY:` の行のコマンドを実行する",
		"TASK に `VERIFY:` の行がなければ、そのスライスが触ったファイルを覆う、いちばん狭い検査を選ぶ",
		"Go なら、触ったパッケージのディレクトリを `<dir>` として `" + implementerGoSliceCheck + "` を打つ",
		"(`gofmt -l` はファイルを挙げても終了コード 0 なので、`test -z` で包む)",
		"ほかの言語なら、言語パックのパッケージごとの検査にする",
		"`RALPH_VERIFY_SCOPE=changed` では絞れない(走らせる言語パックを選ぶだけで、Go では `go test ./...` を全部走らせる)",
		"`./scripts/run-verify.sh` は、それより狭い検査ではスライスのファイルを覆えないときだけ使う",
		"打ったコマンドは RESULT に書く",
		"検証が通ってから、scope 内のパスだけを `git add <path>...`",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the implementer mission item %q to say %q, got item:\n%s", marker, want, item)
		}
	}
	if containsPhrase(text, "指示がなければ `./scripts/run-verify.sh`") {
		t.Errorf("expected the implementer template to drop ./scripts/run-verify.sh as the default for a TASK with no command, got:\n%s", text)
	}
	const fullCheck = "./scripts/run-verify.sh"
	if inText, inItem := strings.Count(text, fullCheck), strings.Count(item, fullCheck); inText != inItem {
		t.Errorf("expected the implementer template to name %s only in the mission's last-resort sentence, found %d mentions, %d of them in that item:\n%s", fullCheck, inText, inItem, text)
	}
}

// TestRenderRolePrompt_Implementer_MechanicalSelfCheckBeforeEachCommit pins
// the self-check half of AC2 of plan 2026-10-27-org-narrow-verification
// (D2): between the staging bullet and the no-commit-on-failure bullet, the
// mission has the implementer read the staged diff before each commit and
// check the five items in order, the fifth against git diff --cached
// --name-only; fix within scope, verify and stage again; leave what cannot be
// fixed within scope and report it as found and not fixed with the reason;
// and for a secret it cannot remove within scope, not commit, take it out of
// the index and the worktree, and return a BLOCKED naming the file and line,
// never the value. The staging bullet before it has the self-check done after
// staging and before the commit, so the mission reads verify, stage,
// self-check, commit.
func TestRenderRolePrompt_Implementer_MechanicalSelfCheckBeforeEachCommit(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "コミットの前に毎回、stage した差分"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the implementer mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	for _, want := range []string{
		"コミットの前に毎回、stage した差分(`git diff --cached`)を読んで、機械的な確かめを行う",
		"項目は 5 つで、この順に見る: " + strings.Join(implementerSelfCheckItems, "、") + "(`git diff --cached --name-only` に出る、scope にないパス)",
		"見つけたものは scope の中で直し、検証のコマンドを打ち直してから stage し直す",
		"scope の中で直せない項目はそのまま残し、RESULT に found and not fixed として理由とともに書く",
		"例外は 2 つで、どちらもコミットしない(コミットには stage してあるものがすべて入る)",
		"1 つ目は秘密情報で、このスライスの変更にある秘密情報を scope の中で消せなければ、このスライスのパスを unstage し、作業ツリーからも出し(値を消す。その値を持つのがこのスライスで作った未追跡のファイルなら、ファイルを消す)、BLOCKED を leader に返す",
		"BLOCKED には秘密情報のファイルと行を書き、値は書かない",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the implementer self-check item to say %q, got item:\n%s", want, item)
		}
	}
	s := squashSpace(mission)
	stageAt := strings.Index(s, squashSpace("1 スライス = 1 コミットとする"))
	checkAt := strings.Index(s, squashSpace(marker))
	failAt := strings.Index(s, squashSpace("検証が失敗した状態のままコミットしない"))
	if stageAt < 0 || checkAt < stageAt || failAt < checkAt {
		t.Errorf("expected the self-check item between the staging bullet and the no-commit-on-failure bullet, got section:\n%s", mission)
	}

	// The staging bullet, which comes first, sends the reader to the
	// self-check after staging and before the commit.
	const sliceMarker = "作業はスライス単位で進める"
	slice, _ := markdownItem(mission, sliceMarker)
	order := []string{"でステージし", "次の項目の機械的な確かめを行ってから", "Conventional Commits(`<type>: <description>`)でコミットする"}
	sliceText := squashSpace(slice)
	at := -1
	for _, step := range order {
		i := strings.Index(sliceText, squashSpace(step))
		if i < 0 || i < at {
			t.Errorf("expected the mission item %q to stage, do the self-check, then commit, saying %q in this order, but %q is missing or out of order, got item:\n%s", sliceMarker, order, step, slice)
			break
		}
		at = i
	}
}

// TestRenderRolePrompt_Implementer_ResultCarriesVerifyAndSelfCheck pins the
// RESULT half of AC2 of plan 2026-10-27-org-narrow-verification: the example
// RESULT is a valid message whose body (not its header block) has a VERIFY:
// line with the mission's Go check run on internal/foo and its result, and a
// SELF-CHECK: line with the five items in order, each clean; and the
// report contract asks for the command, where it came from and its result,
// and for each item's clean, fixed or found and not fixed.
func TestRenderRolePrompt_Implementer_ResultCarriesVerifyAndSelfCheck(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	protocolSection, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## typed protocol' section, got:\n%s", text)
	}
	const lead = "RESULT の例"
	msg, found := codeBlockAfterLine(protocolSection, lead)
	if !found {
		t.Fatalf("expected the implementer's typed protocol section to show a code block after a line starting with %q, got section:\n%s", lead, protocolSection)
	}
	m, err := protocol.Parse(msg)
	if err != nil || protocol.Validate(m, protocol.DefaultMaxBodyChars) != nil || m.Type != protocol.TypeResult || m.TaskID != "t-7" {
		t.Fatalf("expected the example to be a valid RESULT with TASK_ID t-7, got %+v, err %v; message:\n%s", m, err, msg)
	}
	bodyLine := func(key string) string {
		t.Helper()
		if got := protocol.HeaderValues(msg, key); len(got) != 0 {
			t.Errorf("expected %s: in the example RESULT's body, not its header block, got header values %q; message:\n%s", key, got, msg)
		}
		for line := range strings.SplitSeq(m.Body, "\n") {
			if value, ok := strings.CutPrefix(line, key+": "); ok {
				return value
			}
		}
		t.Errorf("expected the example RESULT's body to have a %s: line, got body:\n%s", key, m.Body)
		return ""
	}

	goCheck := strings.ReplaceAll(implementerGoSliceCheck, "<dir>", "internal/foo")
	if verify := bodyLine("VERIFY"); verify != goCheck+" → pass" {
		t.Errorf("expected the example's VERIFY: line to be the mission's Go check on internal/foo and its result, %q, got %q", goCheck+" → pass", verify)
	}
	selfCheck := bodyLine("SELF-CHECK")
	prevAt := -1 // where the previous item was found in selfCheck
	for _, name := range implementerSelfCheckItems {
		i := strings.Index(selfCheck, name+" clean")
		if i < 0 || i < prevAt {
			t.Errorf("expected the example's SELF-CHECK: line to give %q clean, in the mission's order %q, got %q", name, implementerSelfCheckItems, selfCheck)
			break
		}
		prevAt = i
	}

	contract, found := markdownSection(text, "## レポート契約")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## レポート契約' section, got:\n%s", text)
	}
	const marker = "変更したファイル一覧"
	item, ok := markdownItem(contract, marker)
	if !ok {
		t.Fatalf("expected the implementer's report contract to have an item starting %q, got section:\n%s", marker, contract)
	}
	for _, want := range []string{
		"変更したファイル一覧・決定事項や逸脱・検証結果・機械的な確かめの結果・コミット境界の証跡",
		"検証結果には、打った検証のコマンドとその結果を書き、そのコマンドが leader の `VERIFY:` の行のものか、自分で選んだものかも書きます",
		"機械的な確かめの結果には、5 つの項目それぞれについて clean、fixed(直したもの)、found and not fixed(残したものと、scope の中で直せない理由)のどれかを書きます",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the implementer's report contract to say %q, got item:\n%s", want, item)
		}
	}
}

// reviewerStaticChecksLine is the CHECKS: body line of a review RESULT or
// BLOCKED in a feature org, whose leader names only the static script.
const reviewerStaticChecksLine = "CHECKS: ./scripts/run-static-verify.sh"

// reviewerDiffReviewItems are the ten items of the reviewer's diff review, in
// the template's order (plan 2026-10-27-org-narrow-verification, D3).
var reviewerDiffReviewItems = []string{"命名", "読みやすさ", "不要な変更", "誤字とコピペの誤り", "null の扱い", "デバッグの残り", "秘密情報", "例外処理", "security", "保守性"}

// TestRenderRolePrompt_Reviewer_ReviewTaskRunsExactlyTheNamedCommands pins
// the reviewer half of AC3 of plan 2026-10-27-org-narrow-verification: the
// mission's first item has the reviewer run exactly the commands a review
// TASK names, adding and dropping none; says the TASK names them on a CHECKS:
// line of its body, the line step 8 of the leader template writes
// (reviewerStaticChecksLine), and only there (a TASK without one gets the
// default even if it names a command); says a feature org's leader names
// only the static script and that org's full test runs once, before the PR,
// in the gate TASK through ralph org gate; and keeps both scripts as the
// default for a TASK that names no command. That default is the safety
// property of D6: a leader that forgets to name commands, or an older leader
// template that never names them, still gets the tests run (later, never
// skipped). The item is checked on its own, because the mission's fifth item
// names both scripts too, in its CHECKS: example, and would mask a default
// cut back to the static script alone. The fan-out example has a sub-agent
// run the same commands.
func TestRenderRolePrompt_Reviewer_ReviewTaskRunsExactlyTheNamedCommands(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "1. レビューの TASK"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the reviewer mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	for _, want := range []string{
		"leader の TASK にコマンドの指示があればそれを、なければ `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` を実行する",
		"コマンドの指示は、TASK の本文の `CHECKS:` の行にある(例: `" + reviewerStaticChecksLine + "`)",
		"指示とみなすのはこの行だけで、`CHECKS:` の行のない TASK は、本文のほかの所にコマンドの名前があっても、指示のない TASK として扱い、両方のスクリプトを打つ",
		"指示があるときは、指示されたコマンドだけを打ち、足しも削りもしない",
		"機能ごとの org(`ralph org start --plan` で立てた org)の leader は、レビューの TASK に `./scripts/run-static-verify.sh` だけを指示する",
		"その org のフルの test は、PR の前に下の「gate の TASK」で `ralph org gate` が 1 回走らせる",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the reviewer mission item %q to say %q, got item:\n%s", marker, want, item)
		}
	}
	if containsPhrase(text, "`CHECKS:` の行がなくても") {
		t.Errorf("expected the reviewer template to take a review TASK's commands only from its CHECKS: line, not from the prose of a TASK without one, got:\n%s", text)
	}

	fanOut, found := markdownSection(text, "## 座席内 fan-out")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## 座席内 fan-out' section, got:\n%s", text)
	}
	if item, ok := markdownItem(fanOut, "ゲートの実行"); !ok ||
		!containsPhrase(item, "レビューの TASK のゲートの実行(TASK が指示したコマンド、指示がなければ `run-static-verify.sh` と `run-test.sh`)を安いモデルの子サブエージェントに任せ、子にも同じコマンドだけを打たせる") {
		t.Errorf("expected the fan-out example to hand a sub-agent the commands the review TASK names, or both scripts when it names none, and only those, got item:\n%s", item)
	}
}

// TestRenderRolePrompt_Reviewer_GateMeansTheCommandsOnTheChecksLine pins D5
// of plan 2026-10-27-org-narrow-verification: the mission's fifth item says
// the GATE: of a review RESULT or BLOCKED is the result of the first item's
// commands, which go on a CHECKS: line of the body in the order run (the
// static script alone, or both scripts when the TASK named none); every
// RESULT and BLOCKED of items 2 to 4 carries the line, a GATE: unrunnable
// BLOCKED with the commands it tried, the one it could not run named in
// item 3's reason; so in a feature org GATE: pass means the static checks
// passed and the full test's result is the gate TASK's GATE:; the typed
// protocol section keeps CHECKS: out of the header block; and the report
// contract asks for the line in a review RESULT and BLOCKED, the commands
// tried for GATE: unrunnable.
func TestRenderRolePrompt_Reviewer_GateMeansTheCommandsOnTheChecksLine(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "5. 2〜4 で返す RESULT と BLOCKED"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the reviewer mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	for _, want := range []string{
		"2〜4 で返す RESULT と BLOCKED の `GATE:` は、1 のコマンドの結果を指す",
		"1 のコマンドは、本文の `CHECKS:` の行に打った順に書く",
		"(例: `" + reviewerStaticChecksLine + "`。指示がなかったときは `CHECKS: ./scripts/run-static-verify.sh、./scripts/run-test.sh`)",
		"`CHECKS:` の行は、2〜4 で返すどの RESULT と BLOCKED にも付ける",
		"`GATE: unrunnable` の BLOCKED では、打とうとしたコマンドを書き、どれを打てなかったかは 3 のとおり本文の理由に書く",
		"機能ごとの org のレビューでは静的検査だけを打つので、`GATE: pass` は静的検査が通ったことを指す(フルの test の結果は、gate の TASK で返す `GATE:` が指す)",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the reviewer mission item %q to say %q, got item:\n%s", marker, want, item)
		}
	}

	protocolSection, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## typed protocol' section, got:\n%s", text)
	}
	if !containsPhrase(protocolSection, "`CHECKS:` はヘッダではなく、本文(空行のあと)の行に書く") {
		t.Errorf("expected the reviewer's typed protocol section to put CHECKS: in the body, not the header block, got section:\n%s", protocolSection)
	}

	contract, found := markdownSection(text, "## レポート契約")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## レポート契約' section, got:\n%s", text)
	}
	if item, ok := markdownItem(contract, "leader へは RESULT"); !ok ||
		!containsPhrase(item, "レビューの RESULT と BLOCKED の本文には、ゲートで打ったコマンドを `CHECKS:` の行で書いてください(`GATE: unrunnable` では、打とうとしたコマンドを書きます)") {
		t.Errorf("expected the reviewer's report contract to ask for the CHECKS: line in a review RESULT and BLOCKED, with the commands tried for GATE: unrunnable, got item:\n%s", item)
	}
}

// TestRenderRolePrompt_Reviewer_DiffReviewCoversTenItemsAndTheCriteria pins
// D3 of plan 2026-10-27-org-narrow-verification: inside the mission's fourth
// item (the review after a passing gate), one bullet lists exactly the ten
// items of reviewerDiffReviewItems, in that order, and another checks the
// spec's or plan's acceptance criteria; the old four-point list is gone.
func TestRenderRolePrompt_Reviewer_DiffReviewCoversTenItemsAndTheCriteria(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## ミッション' section, got:\n%s", text)
	}
	_, review, found := strings.Cut(mission, "\n4. ゲートが通ったら差分レビューに進む\n")
	if !found {
		t.Fatalf("expected the reviewer mission to have an item 4. ゲートが通ったら差分レビューに進む, got section:\n%s", mission)
	}
	review, _, _ = strings.Cut(review, "\n5. ")

	const itemsMarker = "- 差分を次の"
	itemsWant := "- 差分を次の 10 項目で確かめる: " + strings.Join(reviewerDiffReviewItems, "、")
	if item, ok := markdownItem(review, itemsMarker); !ok || squashSpace(item) != squashSpace(itemsWant) {
		t.Errorf("expected the mission's item 4 to have the bullet %q and nothing more, got item:\n%s", itemsWant, item)
	}
	const criteria = "spec / plan の受け入れ基準に照らして充足しているか確認する"
	if item, ok := markdownItem(review, "- spec / plan の"); !ok || !containsPhrase(item, criteria) {
		t.Errorf("expected the mission's item 4 to check the acceptance criteria (%q), got item 4:\n%s", criteria, review)
	}
	if containsPhrase(text, "差分品質(可読性・命名・責務分離・エラー処理)") {
		t.Errorf("expected the reviewer template to drop the old four-point diff review, got:\n%s", text)
	}
}

// TestRenderRolePrompt_Reviewer_ReviewExamplesCarryTheChecksLine pins the
// examples of D5 of plan 2026-10-27-org-narrow-verification: the review
// RESULT and the BLOCKED for a failing gate in the typed protocol section are
// valid messages for TASK_ID t-42 that keep their headers (GATE: pass, HEAD:
// and BASE: for the RESULT, GATE: fail for the BLOCKED) and an EVIDENCE:
// line, and carry reviewerStaticChecksLine in the body, not the header block.
// The BLOCKED's failure is a static check's from run-static-verify.sh,
// matching that line; it no longer blames run-test.sh, which a feature org's
// review does not run, and still says the diff review was not done.
func TestRenderRolePrompt_Reviewer_ReviewExamplesCarryTheChecksLine(t *testing.T) {
	text := renderSeatPrompt(t, "reviewer")
	protocolSection, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## typed protocol' section, got:\n%s", text)
	}
	const blockedLead = "ゲートが落ちたときの BLOCKED の例"
	var blockedBody string // the BLOCKED example's body, once it is a valid BLOCKED
	for _, c := range []struct {
		lead    string
		typ     string
		headers map[string]string
	}{
		{"RESULT の例", protocol.TypeResult, map[string]string{"GATE": "pass", "HEAD": "", "BASE": ""}},
		{blockedLead, protocol.TypeBlocked, map[string]string{"GATE": "fail"}},
	} {
		msg, found := codeBlockAfterLine(protocolSection, c.lead)
		if !found {
			t.Errorf("expected the reviewer's typed protocol section to show a code block after a line starting with %q, got section:\n%s", c.lead, protocolSection)
			continue
		}
		m, err := protocol.Parse(msg)
		if err != nil || protocol.Validate(m, protocol.DefaultMaxBodyChars) != nil || m.Type != c.typ || m.TaskID != "t-42" {
			t.Errorf("expected the example after %q to be a valid %s with TASK_ID t-42, got %+v, err %v; message:\n%s", c.lead, c.typ, m, err, msg)
			continue
		}
		for key, want := range c.headers {
			if got := protocol.HeaderValues(msg, key); len(got) != 1 || (want != "" && got[0] != want) {
				t.Errorf("expected the example after %q to carry one %s: header (%q), got %q; message:\n%s", c.lead, key, want, got, msg)
			}
		}
		if got := protocol.HeaderValues(msg, "CHECKS"); len(got) != 0 {
			t.Errorf("expected CHECKS: in the body of the example after %q, not its header block, got header values %q; message:\n%s", c.lead, got, msg)
		}
		if !hasLine(m.Body, reviewerStaticChecksLine) {
			t.Errorf("expected the body of the example after %q to have the line %s, got body:\n%s", c.lead, reviewerStaticChecksLine, m.Body)
		}
		if !slices.ContainsFunc(strings.Split(m.Body, "\n"), func(line string) bool { return strings.HasPrefix(line, "EVIDENCE: ") }) {
			t.Errorf("expected the body of the example after %q to keep an EVIDENCE: line, got body:\n%s", c.lead, m.Body)
		}
		if c.lead == blockedLead {
			blockedBody = m.Body
		}
	}

	if blockedBody != "" {
		if strings.Contains(blockedBody, "run-test.sh") {
			t.Errorf("expected the example after %q not to blame run-test.sh, which a feature org's review does not run, got body:\n%s", blockedLead, blockedBody)
		}
		for _, want := range []string{"SUMMARY: run-static-verify.sh の", "差分レビューは行っていない"} {
			if !containsPhrase(blockedBody, want) {
				t.Errorf("expected the body of the example after %q to say %q, got body:\n%s", blockedLead, want, blockedBody)
			}
		}
	}
}

// TestRenderRolePrompt_Leader_MissionPutsTheSliceCommandOnAVerifyLine pins
// the mission half of AC1 of plan 2026-10-27-org-narrow-verification (S3,
// D4): the mission's item 3 has the leader of every org, a feature org or
// not, write the slice's verification command on a VERIFY: line of an
// implementer TASK's body, not in its header block; the command is the
// narrowest check that covers the slice's files, for Go the string the
// implementer template picks for a TASK with no VERIFY: line
// (implementerGoSliceCheck); and an org that is not a feature org names no
// command in a review TASK, leaving the reviewer's default of both scripts
// (D6). The implementer template reads that line, and the reviewer template
// keeps that default, so the three templates agree.
func TestRenderRolePrompt_Leader_MissionPutsTheSliceCommandOnAVerifyLine(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the leader template to contain a '## ミッション' section, got:\n%s", text)
	}
	const marker = "3. `ralph org send`"
	item, ok := markdownItem(mission, marker)
	if !ok {
		t.Fatalf("expected the leader mission to have an item starting %q, got section:\n%s", marker, mission)
	}
	for _, want := range []string{
		"どの org でも(機能ごとの org でなくても)、implementer への TASK には、そのスライスの検証コマンドを本文の `VERIFY:` の行で書く(ヘッダにはしない。「typed protocol」の節)",
		"コマンドは、スライスが触ったファイルを覆う、いちばん狭い検査にする",
		"Go なら、触ったパッケージのディレクトリを `<dir>` として `" + implementerGoSliceCheck + "` にする(implementer の雛形が、`VERIFY:` の行のない TASK で選ぶものと同じ)",
		"ほかの言語なら、言語パックのパッケージごとの検査にする",
		"機能ごとの org でなければ、reviewer へのレビューの TASK には打つコマンドを書かず、reviewer の既定(`./scripts/run-static-verify.sh` と `./scripts/run-test.sh` の両方)に任せる(機能ごとの org は「機能ごとの org」の節の手順 8)",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the leader mission item %q to say %q, got item:\n%s", marker, want, item)
		}
	}

	implementer := renderSeatPrompt(t, "implementer")
	implementerMission, _ := markdownSection(implementer, "## ミッション")
	if implementerItem, ok := markdownItem(implementerMission, "作業はスライス単位で進める"); !ok ||
		!containsPhrase(implementerItem, "leader の TASK の本文の `VERIFY:` の行のコマンドを実行する") ||
		!containsPhrase(implementerItem, "`"+implementerGoSliceCheck+"`") {
		t.Errorf("expected the implementer template to run the VERIFY: line of the leader's TASK body and to pick %s with none, got item:\n%s", implementerGoSliceCheck, implementerItem)
	}
	reviewer := renderSeatPrompt(t, "reviewer")
	reviewerMission, _ := markdownSection(reviewer, "## ミッション")
	if reviewerItem, ok := markdownItem(reviewerMission, "1. レビューの TASK"); !ok ||
		!containsPhrase(reviewerItem, "なければ `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` を実行する") {
		t.Errorf("expected the reviewer template to run both scripts for a review TASK that names no command, got item:\n%s", reviewerItem)
	}
}

// verifyLineOfOneOfSeveralSlices is the example the leader and implementer
// templates give of one slice's VERIFY: line in a TASK that asks for several
// slices.
const verifyLineOfOneOfSeveralSlices = "`VERIFY: S1: <コマンド>`"

// TestRenderRolePrompt_SeveralSlicesInOneTaskGetAVerifyLineEach pins the
// several-slice form of the VERIFY: line of plan
// 2026-10-27-org-narrow-verification (D4): the leader's mission item 3 has a
// TASK that asks for several slices carry one VERIFY: line per slice, each
// starting with the slice's name (verifyLineOfOneOfSeveralSlices), and keeps
// VERIFY: <コマンド> for a single slice; step 5 of the 機能ごとの org section
// copies each slice's plan command to its own line with the slice's name; and
// the implementer's mission reads the same form and, after each slice, runs
// the command on that slice's line.
func TestRenderRolePrompt_SeveralSlicesInOneTaskGetAVerifyLineEach(t *testing.T) {
	leader := renderSeatPrompt(t, "leader")
	implementer := renderSeatPrompt(t, "implementer")
	sections := map[string]string{}
	for _, s := range []struct{ key, text, header string }{
		{"leader", leader, "## ミッション"},
		{"leader", leader, "## 機能ごとの org"},
		{"implementer", implementer, "## ミッション"},
	} {
		body, found := markdownSection(s.text, s.header)
		if !found {
			t.Fatalf("expected the %s template to contain a %q section, got:\n%s", s.key, s.header, s.text)
		}
		sections[s.key+" "+s.header] = body
	}
	for _, c := range []struct{ section, marker, want string }{
		{"leader ## ミッション", "3. `ralph org send`", "1 つの TASK で複数のスライスを頼むときは、`VERIFY:` の行をスライスごとに 1 行ずつ書き、各行をスライスの名前で始める(例: " + verifyLineOfOneOfSeveralSlices + ")"},
		{"leader ## ミッション", "3. `ralph org send`", "スライスが 1 つなら `VERIFY: <コマンド>` の形のままでよい"},
		{"leader ## 機能ごとの org", "5. implementer と reviewer を", "複数のスライスを頼むときは、スライスごとの行に、スライスの名前を付けて写す"},
		{"implementer ## ミッション", "作業はスライス単位で進める", "1 つの TASK が複数のスライスを頼むときは、`VERIFY:` の行がスライスごとに 1 行ずつあり、各行はスライスの名前で始まる(例: " + verifyLineOfOneOfSeveralSlices + ")"},
		{"implementer ## ミッション", "作業はスライス単位で進める", "そのときは、各スライスの実装後に、そのスライスの名前の行のコマンドを打つ"},
	} {
		if item, ok := markdownItem(sections[c.section], c.marker); !ok || !containsPhrase(item, c.want) {
			t.Errorf("expected the item %q of the %s section to say %q, got item:\n%s", c.marker, c.section, c.want, item)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgPlanCarriesTheSliceCommands pins the
// feature org half of AC1 of plan 2026-10-27-org-narrow-verification (S3):
// step 1 of the 機能ごとの org section has every slice of the feature plan's
// ## Implementation outline end with its verification command, in the form
// the repo's plan template (docs/plans/templates/feature-plan.md) gives its
// outline item, the narrowest check of the mission's item 3, and says
// RALPH_VERIFY_SCOPE=changed does not narrow it (in Go it still runs
// go test ./...); step 5 copies the slice's command, unchanged, to the
// VERIFY: line of the implementer TASK's body.
func TestRenderRolePrompt_Leader_FeatureOrgPlanCarriesTheSliceCommands(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	const verifyForm = "— verify: `<コマンド>`"
	for _, c := range []struct{ marker, want string }{
		{"1. 機能の計画を", "計画の `## Implementation outline` では、スライスごとの項目の終わりに、そのスライスの検証コマンドを「" + verifyForm + "」の形で書く(雛形にこの欄があればそれに沿う)"},
		{"1. 機能の計画を", "コマンドは「ミッション」の 3 のとおり、スライスが触ったファイルを覆う、いちばん狭い検査にする(Go なら触ったパッケージの gofmt・vet・test)"},
		{"1. 機能の計画を", "`RALPH_VERIFY_SCOPE=changed` は絞り込みに使わない(走らせる言語パックを選ぶだけで、Go では `go test ./...` を全部走らせる)"},
		{"5. implementer と reviewer を", "implementer への TASK では、頼むスライスの計画の `— verify:` のコマンドを、書き換えずに本文の `VERIFY:` の行に写す(「ミッション」の 3)"},
	} {
		if item, ok := markdownItem(section, c.marker); !ok || !containsPhrase(item, c.want) {
			t.Errorf("expected the 機能ごとの org item naming %q to contain %q, got item:\n%s", c.marker, c.want, item)
		}
	}

	t.Run("docs/plans/templates/feature-plan.md", func(t *testing.T) {
		const rel = "docs/plans/templates/feature-plan.md"
		body, ok, err := readRepoFile(reportNamesRepoRoot(t), rel)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if !ok {
			t.Skipf("%s not found -- skipping the check of its outline item", rel)
		}
		want := "1. <slice> " + strings.Replace(verifyForm, "<コマンド>", "<command>", 1)
		if outline, found := markdownSection(body, "## Implementation outline"); !found || !hasLine(outline, want) {
			t.Errorf("expected %s's ## Implementation outline to have the item %q, whose form step 1 of the leader template names, got section:\n%s", rel, want, outline)
		}
	})
}

// TestRenderRolePrompt_Leader_FeatureOrgReviewTaskNamesOnlyTheStaticScript
// pins the leader half of AC3 of plan 2026-10-27-org-narrow-verification (S3,
// D5, D6): step 8 of the 機能ごとの org section has the leader write one
// command, the static script, on a CHECKS: line in the body of a review TASK
// (a TASK_ID that does not start with gate-), the line the reviewer returns
// (reviewerStaticChecksLine), since the full test runs once in step 12's gate
// TASK; says a review's GATE: pass then means the static checks passed; and
// has the leader read the reviewer's CHECKS: line to see which commands a
// GATE: is the result of, ./scripts/run-test.sh on it meaning the reviewer
// ran its default. Step 12 still has the gate run both scripts (its whole
// wording is pinned by
// TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate), and the
// reviewer template says a feature org's leader names the static script alone
// and returns it on the CHECKS: line step 8 reads.
func TestRenderRolePrompt_Leader_FeatureOrgReviewTaskNamesOnlyTheStaticScript(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	static := strings.TrimPrefix(reviewerStaticChecksLine, "CHECKS: ")
	const marker = "8. TASK・RESULT・レビューの往復"
	item, ok := markdownItem(section, marker)
	if !ok {
		t.Fatalf("expected the 機能ごとの org section to have an item starting %q, got section:\n%s", marker, section)
	}
	for _, want := range []string{
		"この org では、reviewer へのレビューの TASK(TASK_ID が `gate-` で始まらない TASK)の本文に、打つコマンドを `" + reviewerStaticChecksLine + "` の 1 行で書く(`VERIFY:` と同じく本文の行で、ヘッダにはしない)",
		"フルの test は手順 12 の gate の TASK で 1 回走るので、レビューでは打たせない",
		"レビューの `GATE: pass` は静的検査が通ったことだけを指す(テストの結果は、手順 12 の gate の TASK の `GATE:` で見る)",
		"reviewer は打ったコマンドを RESULT と BLOCKED の本文の `CHECKS:` の行で返すので、その行で、`GATE:` がどのコマンドの結果かを確かめる",
		"`./scripts/run-test.sh` も並んでいれば、reviewer は既定の 2 つを打った",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the 機能ごとの org item %q to say %q, got item:\n%s", marker, want, item)
		}
	}
	if step12, ok := markdownItem(section, step12Marker); !ok ||
		!containsPhrase(step12, "gate は worktree と rebase を確かめてから `"+static+"` と `./scripts/run-test.sh` を走らせ") {
		t.Errorf("expected step 12's gate to run both scripts, the full test step 8 leaves to it, got item:\n%s", step12)
	}

	reviewer := renderSeatPrompt(t, "reviewer")
	reviewerMission, _ := markdownSection(reviewer, "## ミッション")
	if reviewerItem, ok := markdownItem(reviewerMission, "1. レビューの TASK"); !ok ||
		!containsPhrase(reviewerItem, "機能ごとの org(`ralph org start --plan` で立てた org)の leader は、レビューの TASK に `"+static+"` だけを指示する") {
		t.Errorf("expected the reviewer template to say a feature org's leader names %s alone in a review TASK, got item:\n%s", static, reviewerItem)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgResendsAReviewWithoutTheStaticCheck
// pins the leader's branch of D5 of plan 2026-10-27-org-narrow-verification:
// in step 8 of the 機能ごとの org section, a review RESULT with no CHECKS:
// line, or whose CHECKS: line lacks the static script, has not confirmed the
// static checks, so the leader does not go on to step 9 and sends the review
// TASK again, with a new TASK_ID and the CHECKS: line. The resend is bounded
// the way step 12 bounds the gate TASK: when two review TASKs in a row come
// back without a CHECKS: line naming the static script, the leader sends no
// third and raises a BLOCKED with ralph org escalate, giving the reason.
func TestRenderRolePrompt_Leader_FeatureOrgResendsAReviewWithoutTheStaticCheck(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	const marker = "8. TASK・RESULT・レビューの往復"
	item, ok := markdownItem(section, marker)
	if !ok {
		t.Fatalf("expected the 機能ごとの org section to have an item starting %q, got section:\n%s", marker, section)
	}
	static := strings.TrimPrefix(reviewerStaticChecksLine, "CHECKS: ")
	for _, want := range []string{
		"レビューの RESULT に `CHECKS:` の行がないか、その行に `" + static + "` がなければ、静的検査が通ったことは確かめられていないので、手順 9 に進まず、新しい TASK_ID で、`CHECKS:` の行を付けたレビューの TASK を送り直す",
		"この理由でレビューの TASK が 2 回続けて通らなければ(送り直した TASK の RESULT にも、`" + static + "` のある `CHECKS:` の行がなければ)、3 回目の TASK は送らず、その理由を本文に書いて `ralph org escalate` で BLOCKED を上げる",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the 機能ごとの org item %q to say %q, got item:\n%s", marker, want, item)
		}
	}
}

// leaderNamesStaticScriptAlone reports the sentences of text
// (templateSentences) that name the static script without
// ./scripts/run-test.sh, or say 静的検査だけ: the instruction step 8 of the
// 機能ごとの org section gives for a feature org's review TASK.
func leaderNamesStaticScriptAlone(text string) []string {
	static := squashSpace(strings.TrimPrefix(reviewerStaticChecksLine, "CHECKS: "))
	var bad []string
	for _, s := range templateSentences(text) {
		if (strings.Contains(s, static) && !strings.Contains(s, "./scripts/run-test.sh")) || strings.Contains(s, "静的検査だけ") {
			bad = append(bad, s)
		}
	}
	return bad
}

// TestRenderRolePrompt_Leader_OnlyAFeatureOrgNamesTheStaticScriptAlone pins
// D6 of plan 2026-10-27-org-narrow-verification for an org that is not a
// feature org: outside the 機能ごとの org section, every sentence of the
// leader template that names the static script names ./scripts/run-test.sh
// with it, and none says 静的検査だけ, so no such org asks for a review that
// drops the test. The mission's item 3, which leaves the reviewer's default
// of both scripts, is such a sentence, so the check is not vacuous, and the
// check flags step 8's instruction, so it does not pass on any text.
func TestRenderRolePrompt_Leader_OnlyAFeatureOrgNamesTheStaticScriptAlone(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	outside := strings.Replace(text, section, "", 1)
	if bad := leaderNamesStaticScriptAlone(outside); len(bad) > 0 {
		t.Errorf("expected no sentence outside the 機能ごとの org section to name the static script without ./scripts/run-test.sh, got:\n%s", strings.Join(bad, "\n"))
	}
	static := squashSpace(strings.TrimPrefix(reviewerStaticChecksLine, "CHECKS: "))
	if !slices.ContainsFunc(templateSentences(outside), func(s string) bool { return strings.Contains(s, static) }) {
		t.Errorf("expected a sentence outside the 機能ごとの org section (the mission's item 3) to name the static script with ./scripts/run-test.sh, got:\n%s", outside)
	}
	if step8, ok := markdownItem(section, "8. TASK・RESULT・レビューの往復"); !ok || len(leaderNamesStaticScriptAlone(step8)) == 0 {
		t.Errorf("expected the check to flag step 8's instruction to name the static script alone, got item:\n%s", step8)
	}
}

// TestRenderRolePrompt_Leader_ImplementerTaskExampleCarriesTheVerifyLine pins
// the example of AC1 of plan 2026-10-27-org-narrow-verification (D4): the
// typed protocol section puts VERIFY: in the body, not the header block, and
// shows an implementer TASK example that is a valid TASK whose body (not its
// header block) has a SUMMARY: line and a VERIFY: line with the mission's Go
// check run on internal/foo, a $(...) the shell would expand in a
// double-quoted argument (TestRenderRolePrompt_SeatsSendMessagesFromAFile).
// The review TASK example stays a valid TASK that names no command: the review
// TASK of an org that is not a feature org (D6).
func TestRenderRolePrompt_Leader_ImplementerTaskExampleCarriesTheVerifyLine(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	protocolSection, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("expected the leader template to contain a '## typed protocol' section, got:\n%s", text)
	}
	if !containsPhrase(protocolSection, "`VERIFY:` はヘッダではなく、本文(空行のあと)の行に書きます") {
		t.Errorf("expected the leader's typed protocol section to put VERIFY: in the body, not the header block, got section:\n%s", protocolSection)
	}

	const lead = "implementer への TASK の例"
	msg, found := codeBlockAfterLine(protocolSection, lead)
	if !found {
		t.Fatalf("expected the leader's typed protocol section to show a code block after a line starting with %q, got section:\n%s", lead, protocolSection)
	}
	m, err := protocol.Parse(msg)
	if err != nil || protocol.Validate(m, protocol.DefaultMaxBodyChars) != nil || m.Type != protocol.TypeTask || m.TaskID != "t-2" {
		t.Fatalf("expected the example after %q to be a valid TASK with TASK_ID t-2, got %+v, err %v; message:\n%s", lead, m, err, msg)
	}
	if got := protocol.HeaderValues(msg, "VERIFY"); len(got) != 0 {
		t.Errorf("expected VERIFY: in the body of the example after %q, not its header block, got header values %q; message:\n%s", lead, got, msg)
	}
	goCheck := strings.ReplaceAll(implementerGoSliceCheck, "<dir>", "internal/foo")
	if !hasLine(m.Body, "VERIFY: "+goCheck) {
		t.Errorf("expected the body of the example after %q to have the line %q, got body:\n%s", lead, "VERIFY: "+goCheck, m.Body)
	}
	if !strings.HasPrefix(m.Body, "SUMMARY: ") {
		t.Errorf("expected the body of the example after %q to start with a SUMMARY: line, got body:\n%s", lead, m.Body)
	}

	const reviewLead = "TASK の例("
	review, found := codeBlockAfterLine(protocolSection, reviewLead)
	if !found {
		t.Fatalf("expected the leader's typed protocol section to show a code block after a line starting with %q, got section:\n%s", reviewLead, protocolSection)
	}
	rm, err := protocol.Parse(review)
	if err != nil || protocol.Validate(rm, protocol.DefaultMaxBodyChars) != nil || rm.Type != protocol.TypeTask {
		t.Errorf("expected the example after %q to be a valid TASK, got %+v, err %v; message:\n%s", reviewLead, rm, err, review)
	}
	for _, banned := range []string{"./scripts/", "VERIFY:", "CHECKS:"} {
		if strings.Contains(review, banned) {
			t.Errorf("expected the review TASK example after %q to name no command (%q), as an org that is not a feature org writes it, got message:\n%s", reviewLead, banned, review)
		}
	}
}

// TestRenderRolePrompt_SeatsSendMessagesFromAFile pins the sending rule of S3
// of plan 2026-10-27-org-narrow-verification (consult's finding on S1): a
// VERIFY: line holds a $(...), which the shell expands inside a double-quoted
// argument, so the leader's typed protocol section has every message it sends
// with ralph org send --text written to a file first and passed as
// --text "$(cat <file>)", the form and the file rules of the 件を上げる
// section's escalate, and the implementer's typed protocol section has its
// RESULT and BLOCKED written to a file under .harness/state/, outside its
// scope if need be, and passed as "$(cat <file>)".
func TestRenderRolePrompt_SeatsSendMessagesFromAFile(t *testing.T) {
	const fromFile = "`--text \"$(cat <ファイル>)\"`"
	leader := renderSeatPrompt(t, "leader")
	protocolSection, found := markdownSection(leader, "## typed protocol")
	if !found {
		t.Fatalf("expected the leader template to contain a '## typed protocol' section, got:\n%s", leader)
	}
	for _, want := range []string{
		"`ralph org send --text` で送るメッセージ(TASK・DECISION・STOP)も、「件を上げる」の節の escalate のメッセージと同じく、ファイルに書き、" + fromFile + " で渡してください",
		"メッセージを `--text \"...\"` の二重引用符の中に直接書くと、implementer への TASK の `VERIFY:` の行にある `$(gofmt -l …)` のような `$(...)` を shell が展開してしまいます",
		"ファイルの置き場所と書き方(cwd の `.harness/state/` の下、書く前に `mkdir -p .harness/state`、Write ツールか `<<'EOF'` の heredoc)は「件を上げる」の節と同じです",
	} {
		if !containsPhrase(protocolSection, want) {
			t.Errorf("expected the leader's typed protocol section to say %q, got section:\n%s", want, protocolSection)
		}
	}
	escalate, _ := markdownSection(leader, "## 件を上げる(`ralph org escalate`)")
	for _, want := range []string{
		"メッセージはファイルに書き、" + fromFile + " で渡してください",
		"ファイルは cwd の `.harness/state/` の下に置きます",
		"書く前に `mkdir -p .harness/state` を打ちます",
		"`cat > .harness/state/escalate-org-a.txt <<'EOF'` の heredoc で書きます",
	} {
		if !containsPhrase(escalate, want) {
			t.Errorf("expected the 件を上げる section, whose file rules the typed protocol section points at, to say %q, got section:\n%s", want, escalate)
		}
	}

	implementer := renderSeatPrompt(t, "implementer")
	implementerProtocol, found := markdownSection(implementer, "## typed protocol")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## typed protocol' section, got:\n%s", implementer)
	}
	for _, want := range []string{
		"RESULT と BLOCKED のメッセージは、cwd の `.harness/state/` の下のファイルに書き(なければ `mkdir -p .harness/state` で作る。git が無視する場所なので、scope の外でも書いてよい)、送るコマンドの本文の引数には `\"$(cat <ファイル>)\"` を渡してください",
		"(二重引用符の引数に直接書くと、`VERIFY:` の行の `$(gofmt -l …)` のような `$(...)` を shell が展開してしまいます)",
	} {
		if !containsPhrase(implementerProtocol, want) {
			t.Errorf("expected the implementer's typed protocol section to say %q, got section:\n%s", want, implementerProtocol)
		}
	}
}

// TestRenderRolePrompt_Implementer_ScopeRuleExceptsTheMessageFiles pins the
// other side of the implementer's sending rule
// (TestRenderRolePrompt_SeatsSendMessagesFromAFile): the スコープ規律
// section's rule against writing outside the scope names the RESULT and
// BLOCKED message files, which the typed protocol section has it write under
// .harness/state/, as its exception, the way the reviewer's rule names its
// own.
func TestRenderRolePrompt_Implementer_ScopeRuleExceptsTheMessageFiles(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	scopeRule, found := markdownSection(text, "## スコープ規律")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## スコープ規律' section, got:\n%s", text)
	}
	const marker = "の範囲外のファイルは読むだけに留め"
	item, ok := markdownItem(scopeRule, marker)
	if !ok {
		t.Fatalf("expected the implementer's スコープ規律 section to have an item with %q, got section:\n%s", marker, scopeRule)
	}
	const want = "書き込みは行わないでください(RESULT と BLOCKED のメッセージのファイルは、「typed protocol」の節に書いたとおり例外です。置き場所の `.harness/state/` は git が無視します)"
	if !containsPhrase(item, want) {
		t.Errorf("expected the implementer's scope rule to say %q, got item:\n%s", want, item)
	}
}
