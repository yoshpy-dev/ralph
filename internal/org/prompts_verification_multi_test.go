package org

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// TestRenderRolePrompt_Implementer_ResultOfSeveralSlicesNamesEachSlice pins
// the RESULT side of the several-slice form of plan
// 2026-10-27-org-narrow-verification (D4): when one TASK asked for several
// slices, the implementer's report contract has the RESULT carry one COMMIT:,
// one VERIFY: and one SELF-CHECK: line per slice, each starting with the
// slice's name the way the TASK's VERIFY: lines do
// (verifyLineOfOneOfSeveralSlices), and keeps the unnamed lines for a single
// slice. The typed protocol section says its example is the RESULT of a
// single slice and points at the contract for several, and the example keeps
// its unnamed COMMIT: line.
func TestRenderRolePrompt_Implementer_ResultOfSeveralSlicesNamesEachSlice(t *testing.T) {
	text := renderSeatPrompt(t, "implementer")
	contract, found := markdownSection(text, "## レポート契約")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## レポート契約' section, got:\n%s", text)
	}
	const marker = "変更したファイル一覧"
	item, ok := markdownItem(contract, marker)
	if !ok {
		t.Fatalf("expected the implementer's report contract to have an item starting %q, got section:\n%s", marker, contract)
	}
	verifyOfSlice := strings.TrimSuffix(verifyLineOfOneOfSeveralSlices, "`") + " → pass`"
	selfCheckOfSlice := "`SELF-CHECK: S1: " + implementerSelfCheckItems[0] + " clean / …`"
	for _, want := range []string{
		"1 つの TASK が複数のスライスを頼んだときは、`COMMIT:`・`VERIFY:`・`SELF-CHECK:` の行をスライスごとに 1 行ずつ書き、各行を TASK の `VERIFY:` の行と同じくスライスの名前で始めます(例: `COMMIT: S1: <commit-sha>`、" + verifyOfSlice + "、" + selfCheckOfSlice + ")",
		"スライスが 1 つなら、「typed protocol」の節の例のとおり、名前を付けずに書きます",
	} {
		if !containsPhrase(item, want) {
			t.Errorf("expected the implementer's report contract to say %q, got item:\n%s", want, item)
		}
	}

	protocolSection, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("expected the implementer template to contain a '## typed protocol' section, got:\n%s", text)
	}
	const lead = "RESULT の例(スライスが 1 つの TASK への RESULT。複数のスライスのときの行の形は「レポート契約」の節。"
	if !containsPhrase(protocolSection, lead) {
		t.Errorf("expected the implementer's typed protocol section to introduce its RESULT example with %q, got section:\n%s", lead, protocolSection)
	}
	msg, found := codeBlockAfterLine(protocolSection, "RESULT の例")
	if !found {
		t.Fatalf("expected the implementer's typed protocol section to show a code block after a line starting with %q, got section:\n%s", "RESULT の例", protocolSection)
	}
	m, err := protocol.Parse(msg)
	if err != nil {
		t.Fatalf("expected the example RESULT to parse, got %v; message:\n%s", err, msg)
	}
	if !hasLine(m.Body, "COMMIT: <commit-sha>") {
		t.Errorf("expected the single-slice example RESULT's body to keep the unnamed line COMMIT: <commit-sha>, got body:\n%s", m.Body)
	}
}

// TestOrgSkill_RoleBulletsNameEachSliceAndReadOnlyTheChecksLine pins AC4 of
// plan 2026-10-27-org-narrow-verification for two rules of the seat
// templates: the ## 役割 section's implementer bullet has a RESULT for several
// slices start its COMMIT:, VERIFY: and SELF-CHECK: lines with the slice's
// name, as the implementer's report contract does, and its reviewer bullet
// takes a review TASK's commands only from the CHECKS: line, both scripts
// running without one, as item 1 of the reviewer's mission does.
// scripts/check-skill-sync.sh keeps the four copies identical, so this test
// reads the root copy only.
func TestOrgSkill_RoleBulletsNameEachSliceAndReadOnlyTheChecksLine(t *testing.T) {
	const rel = ".claude/skills/org/SKILL.md"
	text := readRepoFileOrSkip(t, rel)
	roles, found := markdownSection(text, "## 役割")
	if !found {
		t.Fatalf("%s: expected a '## 役割' section, found none", rel)
	}
	for _, c := range []struct{ marker, want string }{
		{"- **implementer**:", "RESULT には commit SHA と証拠のポインタのほかに、打った検証のコマンドとその結果、5 項目それぞれの確かめの結果を書く。複数のスライスを頼む TASK では、`COMMIT:`・`VERIFY:`・`SELF-CHECK:` の行をスライスごとに書き、各行をスライスの名前で始める(例: `COMMIT: S1: <sha>`)"},
		{"- **reviewer**:", "指示は TASK の本文の `CHECKS:` の行で来て、指示とみなすのはその行だけ(`CHECKS:` の行がなければ、本文にコマンドの名前があっても両方を打つ)"},
	} {
		if item, ok := markdownItem(roles, c.marker); !ok || !containsPhrase(item, c.want) {
			t.Errorf("%s: expected the ## 役割 bullet %q to say %q, got item:\n%s", rel, c.marker, c.want, item)
		}
	}
	if containsPhrase(roles, "その行がなくても") {
		t.Errorf("%s: expected the reviewer bullet to take a review TASK's commands only from its CHECKS: line, not from the prose of a TASK without one, got section:\n%s", rel, roles)
	}
}

// TestRenderRolePrompt_Implementer_SelfCheckLeavesChangesThatPredateTheSlice
// pins the fix for cross-review cycle 1 #1 of plan
// 2026-10-27-org-narrow-verification: the implementer's mechanical self-check
// fixes only the slice's own changes. A change the git status --porcelain
// recorded before the first edit shows (a staged one too) predates the seat,
// so the seat neither edits, deletes nor unstages it. Neither exception
// commits, since a commit takes everything staged. A secret in the slice's
// own changes leaves the index and the worktree (a file is deleted only when
// the slice created it); a secret in a change that predates the slice stays
// as it is, and the BLOCKED says the change predates the slice; the BLOCKED
// line after both names the file and line, never the value. A path outside
// the scope staged before the seat started stays staged and is named in a
// BLOCKED, while one the slice staged by mistake is unstaged. The old wording,
// which cleaned up any secret in the staged diff and deleted any new
// untracked file, must not come back.
func TestRenderRolePrompt_Implementer_SelfCheckLeavesChangesThatPredateTheSlice(t *testing.T) {
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
	steps := []string{
		"直すのはこのスライスの変更だけにする",
		"編集を始める前に記録した `git status --porcelain` に出ていた変更(stage 済みのものも含む)は、座席が始める前からあった変更なので、書き換えも削除も unstage もしない",
		"例外は 2 つで、どちらもコミットしない(コミットには stage してあるものがすべて入る)",
		"このスライスの変更にある秘密情報を scope の中で消せなければ、このスライスのパスを unstage し",
		"その値を持つのがこのスライスで作った未追跡のファイルなら、ファイルを消す",
		"前からあった変更にある秘密情報は、書き換えも削除も unstage もせず、スライスより前からあった変更だと書いた BLOCKED を返す",
		"BLOCKED には秘密情報のファイルと行を書き、値は書かない",
		"2 つ目は、座席が始める前から stage されていた scope の外のパスで、触らずに、そのパスを書いた BLOCKED を返す",
		"(このスライスが誤って stage したパスなら、unstage すれば直る)",
	}
	s := squashSpace(item)
	at := -1
	for _, step := range steps {
		i := strings.Index(s, squashSpace(step))
		if i < 0 || i < at {
			t.Errorf("expected the implementer self-check item to say %q in the order %q, but it is missing or out of order, got item:\n%s", step, steps, item)
			break
		}
		at = i
	}
	for _, old := range []string{"例外は秘密情報だけ", "新しい未追跡のファイル"} {
		if containsPhrase(item, old) {
			t.Errorf("expected the implementer self-check item to drop the old wording %q, which cleaned up changes that predate the slice, got item:\n%s", old, item)
		}
	}
}

// TestOrgSkill_ImplementerBulletLeavesChangesThatPredateTheSlice pins the
// /org skill side of the same fix: the ## 役割 section's implementer bullet
// says the seat fixes only the slice's own changes and, for a secret in a
// change that predates the seat or a path staged before it, touches nothing,
// does not commit and returns a BLOCKED, as the implementer's self-check
// item does. Like TestOrgSkill_RoleBulletsNameEachSliceAndReadOnlyTheChecksLine,
// it reads the root copy only.
func TestOrgSkill_ImplementerBulletLeavesChangesThatPredateTheSlice(t *testing.T) {
	const rel = ".claude/skills/org/SKILL.md"
	text := readRepoFileOrSkip(t, rel)
	roles, found := markdownSection(text, "## 役割")
	if !found {
		t.Fatalf("%s: expected a '## 役割' section, found none", rel)
	}
	const marker = "- **implementer**:"
	const want = "直すのはそのスライスの変更だけで、座席が始める前からあった変更の秘密情報や、前から stage されていたパスには触らず、コミットせずに BLOCKED を返す"
	if item, ok := markdownItem(roles, marker); !ok || !containsPhrase(item, want) {
		t.Errorf("%s: expected the ## 役割 bullet %q to say %q, got item:\n%s", rel, marker, want, item)
	}
}
