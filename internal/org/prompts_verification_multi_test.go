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
