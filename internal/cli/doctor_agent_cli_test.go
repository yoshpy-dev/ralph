package cli

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestAgentCLIResults_EveryCombination pins how agentCLIResults decides the
// claude and codex lines for every combination of the two probes and the two
// [doctor].require_*_cli flags (issue #272): a usable CLI passes; one that is
// not usable fails when it is required, warns when it is not and the other
// one is usable, and fails with the "at least one" wording when neither is.
func TestAgentCLIResults_EveryCombination(t *testing.T) {
	const (
		claudeVersion = "2.1.0 (Claude Code)"
		codexVersion  = "codex-cli 0.154.0"
		claudeErrText = "claude not found in PATH: stub lookup error"
		codexErrText  = "codex --version failed: exit status 1"
	)
	claudeUsable := agentCLIProbe{Version: claudeVersion}
	claudeMissing := agentCLIProbe{Err: errors.New(claudeErrText)}
	codexUsable := agentCLIProbe{Version: codexVersion}
	codexMissing := agentCLIProbe{Err: errors.New(codexErrText)}

	// Detail phrases each kind of line must carry. A pass line's detail is
	// the version; the others carry the probe error text.
	const (
		notRequired = "unusable (not required): "
		atLeastOne  = "neither claude nor codex is usable, and at least one of them is required"
	)
	warnClaude := []string{"claude " + notRequired + claudeErrText, "codex is usable, so ralph runs", "plan advisory", "cross-review that call claude will not run"}
	warnCodex := []string{"codex " + notRequired + codexErrText, "claude is usable, so ralph runs", "plan advisory", "cross-review that call codex will not run"}
	requiredClaude := []string{"claude unusable: " + claudeErrText}
	requiredCodex := []string{"codex unusable: " + codexErrText}
	neitherClaude := []string{"claude unusable: " + claudeErrText, atLeastOne}
	neitherCodex := []string{"codex unusable: " + codexErrText, atLeastOne}

	tests := []struct {
		name                        string
		claude, codex               agentCLIProbe
		requireClaude, requireCodex bool
		wantClaudeStatus            string
		wantClaudeDetail            []string
		wantClaudeNotDetail         []string
		wantCodexStatus             string
		wantCodexDetail             []string
		wantCodexNotDetail          []string
	}{
		// Both usable: both pass whatever the flags say.
		{name: "both usable, nothing required", claude: claudeUsable, codex: codexUsable,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},
		{name: "both usable, claude required", claude: claudeUsable, codex: codexUsable, requireClaude: true,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},
		{name: "both usable, codex required", claude: claudeUsable, codex: codexUsable, requireCodex: true,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},
		{name: "both usable, both required", claude: claudeUsable, codex: codexUsable, requireClaude: true, requireCodex: true,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},

		// Only claude usable: codex warns unless it is required.
		{name: "claude only, nothing required", claude: claudeUsable, codex: codexMissing,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "warn", wantCodexDetail: warnCodex, wantCodexNotDetail: []string{atLeastOne}},
		{name: "claude only, claude required", claude: claudeUsable, codex: codexMissing, requireClaude: true,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "warn", wantCodexDetail: warnCodex, wantCodexNotDetail: []string{atLeastOne}},
		{name: "claude only, codex required", claude: claudeUsable, codex: codexMissing, requireCodex: true,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "fail", wantCodexDetail: requiredCodex, wantCodexNotDetail: []string{notRequired, atLeastOne, "plan advisory"}},
		{name: "claude only, both required", claude: claudeUsable, codex: codexMissing, requireClaude: true, requireCodex: true,
			wantClaudeStatus: "pass", wantClaudeDetail: []string{claudeVersion},
			wantCodexStatus: "fail", wantCodexDetail: requiredCodex, wantCodexNotDetail: []string{notRequired, atLeastOne, "plan advisory"}},

		// Only codex usable: claude warns unless it is required.
		{name: "codex only, nothing required", claude: claudeMissing, codex: codexUsable,
			wantClaudeStatus: "warn", wantClaudeDetail: warnClaude, wantClaudeNotDetail: []string{atLeastOne},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},
		{name: "codex only, claude required", claude: claudeMissing, codex: codexUsable, requireClaude: true,
			wantClaudeStatus: "fail", wantClaudeDetail: requiredClaude, wantClaudeNotDetail: []string{notRequired, atLeastOne, "plan advisory"},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},
		{name: "codex only, codex required", claude: claudeMissing, codex: codexUsable, requireCodex: true,
			wantClaudeStatus: "warn", wantClaudeDetail: warnClaude, wantClaudeNotDetail: []string{atLeastOne},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},
		{name: "codex only, both required", claude: claudeMissing, codex: codexUsable, requireClaude: true, requireCodex: true,
			wantClaudeStatus: "fail", wantClaudeDetail: requiredClaude, wantClaudeNotDetail: []string{notRequired, atLeastOne, "plan advisory"},
			wantCodexStatus: "pass", wantCodexDetail: []string{codexVersion}},

		// Neither usable: both fail. A required side keeps the required
		// wording; a side that is not required says at least one is needed.
		{name: "neither, nothing required", claude: claudeMissing, codex: codexMissing,
			wantClaudeStatus: "fail", wantClaudeDetail: neitherClaude, wantClaudeNotDetail: []string{notRequired, "plan advisory"},
			wantCodexStatus: "fail", wantCodexDetail: neitherCodex, wantCodexNotDetail: []string{notRequired, "plan advisory"}},
		{name: "neither, claude required", claude: claudeMissing, codex: codexMissing, requireClaude: true,
			wantClaudeStatus: "fail", wantClaudeDetail: requiredClaude, wantClaudeNotDetail: []string{notRequired, atLeastOne, "plan advisory"},
			wantCodexStatus: "fail", wantCodexDetail: neitherCodex, wantCodexNotDetail: []string{notRequired, "plan advisory"}},
		{name: "neither, codex required", claude: claudeMissing, codex: codexMissing, requireCodex: true,
			wantClaudeStatus: "fail", wantClaudeDetail: neitherClaude, wantClaudeNotDetail: []string{notRequired, "plan advisory"},
			wantCodexStatus: "fail", wantCodexDetail: requiredCodex, wantCodexNotDetail: []string{notRequired, atLeastOne, "plan advisory"}},
		{name: "neither, both required", claude: claudeMissing, codex: codexMissing, requireClaude: true, requireCodex: true,
			wantClaudeStatus: "fail", wantClaudeDetail: requiredClaude, wantClaudeNotDetail: []string{notRequired, atLeastOne, "plan advisory"},
			wantCodexStatus: "fail", wantCodexDetail: requiredCodex, wantCodexNotDetail: []string{notRequired, atLeastOne, "plan advisory"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doctor := config.DoctorConfig{RequireClaudeCLI: tt.requireClaude, RequireCodexCLI: tt.requireCodex}
			claude, codex := agentCLIResults(doctor, tt.claude, tt.codex)
			assertAgentCLIResult(t, claude, "Claude Code CLI", tt.wantClaudeStatus, tt.wantClaudeDetail, tt.wantClaudeNotDetail)
			assertAgentCLIResult(t, codex, "Codex", tt.wantCodexStatus, tt.wantCodexDetail, tt.wantCodexNotDetail)
		})
	}
}

func assertAgentCLIResult(t *testing.T, r checkResult, wantName, wantStatus string, wantDetail, wantNotDetail []string) {
	t.Helper()
	if r.Name != wantName {
		t.Errorf("name = %q, want %q", r.Name, wantName)
	}
	if r.Status != wantStatus {
		t.Errorf("%s status = %q, want %q (detail %q)", wantName, r.Status, wantStatus, r.Detail)
	}
	for _, want := range wantDetail {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("%s detail = %q, want it to contain %q", wantName, r.Detail, want)
		}
	}
	for _, notWant := range wantNotDetail {
		if strings.Contains(r.Detail, notWant) {
			t.Errorf("%s detail = %q, want it not to contain %q", wantName, r.Detail, notWant)
		}
	}
	if strings.Contains(r.Detail, "\n") {
		t.Errorf("%s detail = %q, want a single line", wantName, r.Detail)
	}
}

// templateRalphToml returns the repository's templates/base/ralph.toml, the
// file `ralph init` writes. The test embed FS ships a smaller ralph.toml, so
// tests that depend on the shipped [doctor] defaults read this one instead.
func templateRalphToml(t *testing.T) []byte {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// thisFile is internal/cli/doctor_agent_cli_test.go → repo root is ../../
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "..", "templates", "base", "ralph.toml"))
	if err != nil {
		t.Fatalf("read templates/base/ralph.toml: %v", err)
	}
	return data
}

// doctorLine returns the status and detail of the doctor report line named
// name ("  <icon> <name>: <status> — <detail>"), and false when there is none.
func doctorLine(out, name string) (status, detail string, ok bool) {
	for line := range strings.SplitSeq(out, "\n") {
		_, rest, found := strings.Cut(strings.TrimSpace(line), " ")
		if !found {
			continue
		}
		after, isLine := strings.CutPrefix(rest, name+": ")
		if !isLine {
			continue
		}
		status, detail, _ = strings.Cut(after, " — ")
		return status, detail, true
	}
	return "", "", false
}

// TestRunDoctorOpts_AgentCLIs runs doctor on an initialized project with
// stub binaries on PATH and pins the claude and codex lines and doctor's
// error (exit code 1 when non-nil) for AC1-AC3 of issue #272. Every case but
// the two that set require_*_cli uses the repository's templates/base/ralph.toml,
// so a change to its [doctor] defaults shows up here. The both-usable case is
// the baseline: it shows that no other check fails in this project, so a
// failure in the other cases comes from the agent lines.
func TestRunDoctorOpts_AgentCLIs(t *testing.T) {
	tests := []struct {
		name             string
		ralphToml        []byte // nil: the repository's templates/base/ralph.toml
		bins             []string
		brokenBins       []string // on PATH, but `--version` exits 1
		wantErr          bool
		wantClaude       string
		wantClaudeDetail []string
		wantCodex        string
		wantCodexDetail  []string
		wantFailLines    int
	}{
		{name: "both usable", bins: []string{"claude", "codex"},
			wantClaude: "pass", wantCodex: "pass"},
		{name: "claude only", bins: []string{"claude"},
			wantClaude: "pass",
			wantCodex:  "warn", wantCodexDetail: []string{"codex unusable (not required): codex not found in PATH", "claude is usable, so ralph runs", "plan advisory and cross-review that call codex will not run"}},
		{name: "codex only", bins: []string{"codex"},
			wantClaude: "warn", wantClaudeDetail: []string{"claude unusable (not required): claude not found in PATH", "codex is usable, so ralph runs", "plan advisory and cross-review that call claude will not run"},
			wantCodex: "pass"},
		{name: "neither", wantErr: true, wantFailLines: 2,
			wantClaude: "fail", wantClaudeDetail: []string{"claude not found in PATH", "neither claude nor codex is usable, and at least one of them is required"},
			wantCodex: "fail", wantCodexDetail: []string{"codex not found in PATH", "neither claude nor codex is usable, and at least one of them is required"}},
		{name: "codex required and absent", ralphToml: []byte("[doctor]\nrequire_codex_cli = true\n"), bins: []string{"claude"},
			wantErr: true, wantFailLines: 1,
			wantClaude: "pass",
			wantCodex:  "fail", wantCodexDetail: []string{"codex unusable: codex not found in PATH"}},
		{name: "claude required and absent", ralphToml: []byte("[doctor]\nrequire_claude_cli = true\n"), bins: []string{"codex"},
			wantErr: true, wantFailLines: 1,
			wantClaude: "fail", wantClaudeDetail: []string{"claude unusable: claude not found in PATH"},
			wantCodex: "pass"},
		{name: "broken claude shim", bins: []string{"codex"}, brokenBins: []string{"claude"},
			wantClaude: "warn", wantClaudeDetail: []string{"claude unusable (not required): claude --version failed", "codex is usable, so ralph runs"},
			wantCodex: "pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestEmbedFS(t)
			Version = "0.1.0-test"

			dir := t.TempDir()
			if err := executeInit(dir, initConfig{ProjectName: "test", Packs: []string{"golang"}}, false); err != nil {
				t.Fatalf("init: %v", err)
			}
			ralphToml := tt.ralphToml
			if ralphToml == nil {
				ralphToml = templateRalphToml(t)
			}
			if err := os.WriteFile(filepath.Join(dir, "ralph.toml"), ralphToml, 0o644); err != nil {
				t.Fatal(err)
			}

			binDir := filepath.Join(dir, "bin")
			if err := os.MkdirAll(binDir, 0o755); err != nil {
				t.Fatal(err)
			}
			for _, bin := range append([]string{"go"}, tt.bins...) {
				writeStubBin(t, binDir, bin, "")
			}
			for _, bin := range tt.brokenBins {
				writeFailingStubBin(t, binDir, bin, bin+": entry script missing")
			}
			t.Setenv("PATH", binDir)
			t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(dir, "no-such-agmsg-home"))

			var out string
			err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) })
			if (err != nil) != tt.wantErr {
				t.Fatalf("runDoctorOpts error = %v, want error %v\noutput:\n%s", err, tt.wantErr, out)
			}

			for _, line := range []struct {
				name, wantStatus string
				wantDetail       []string
			}{
				{"Claude Code CLI", tt.wantClaude, tt.wantClaudeDetail},
				{"Codex", tt.wantCodex, tt.wantCodexDetail},
			} {
				status, detail, ok := doctorLine(out, line.name)
				if !ok {
					t.Fatalf("no %q line in output:\n%s", line.name, out)
				}
				if status != line.wantStatus {
					t.Errorf("%s status = %q, want %q (detail %q)", line.name, status, line.wantStatus, detail)
				}
				for _, want := range line.wantDetail {
					if !strings.Contains(detail, want) {
						t.Errorf("%s detail = %q, want it to contain %q", line.name, detail, want)
					}
				}
			}
			if got := strings.Count(out, "✗ "); got != tt.wantFailLines {
				t.Errorf("fail lines = %d, want %d (only the agent lines may fail)\noutput:\n%s", got, tt.wantFailLines, out)
			}
		})
	}
}
