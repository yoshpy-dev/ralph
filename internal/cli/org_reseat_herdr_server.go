package cli

import (
	"context"
	"fmt"
	"io"
	"os"
)

// The lines `ralph org reseat` prints on stderr about the herdr server
// (spec docs/specs/2026-11-20-org-reseat-restart-herdr.md, FR-2 and FR-6):
// before it runs, the TUI note of --restart-herdr and the CLAUDE_CONFIG_DIR
// of the server that runs, with the notes that follow it
// (printOrgReseatHerdrServerNotes); after a real reseat that started the
// server again, that CLAUDE_CONFIG_DIR read once more
// (printOrgReseatRestartedHerdrServerNotes).

// orgReseatTUINote is the line a reseat with --restart-herdr prints on
// stderr before it stops anything (spec FR-2): a herdr TUI is not a pane, so
// the pane check does not refuse it, but it closes with the server. The line
// comes before the checks that may refuse the reseat, so it says that the
// server stops only once they pass, and stays true for a refused run.
const orgReseatTUINote = "note: once its checks pass, --restart-herdr stops the herdr server, which also closes any herdr " +
	"TUI attached to it; run 'herdr' again once the reseat has ended to see the panes"

// printOrgReseatHerdrServerNotes writes on stderr what a reseat says about
// the herdr server before it runs, in a dry run too (spec FR-6, FR-2): with
// restartHerdr, orgReseatTUINote; then the CLAUDE_CONFIG_DIR of the server
// that runs, read with orgHerdrServerClaudeConfigDirReader
// (orgReseatHerdrServerLine). Without restartHerdr, when the value was read
// and is not the one wanted (orgReseatWantedClaudeConfigDir), one note says
// that the seats start with the server's environment
// (orgReseatClaudeConfigDirMismatchNote); when it could not be read,
// --claude-config-dir was given, and the server's status says it runs, one
// note says that the flag does not reach a server that runs. A read that
// fails is never a refusal. A nil reader reads nothing and prints neither the
// line nor the notes after it: the tests' TestMain sets it so, and a test of
// the lines sets a fake.
func printOrgReseatHerdrServerNotes(stderr io.Writer, restartHerdr bool, env orgHerdrServerEnvOverride) {
	if restartHerdr {
		_, _ = fmt.Fprintln(stderr, orgReseatTUINote)
	}
	read := orgHerdrServerClaudeConfigDirReader
	if read == nil {
		return
	}
	got := read(context.Background())
	_, _ = fmt.Fprintln(stderr, orgReseatHerdrServerLine(got))
	if restartHerdr {
		return
	}
	value, set := orgReseatWantedClaudeConfigDir(env)
	switch got.state {
	case orgHerdrServerEnvSet, orgHerdrServerEnvUnset:
		if orgReseatHasWantedClaudeConfigDir(got, value, set) {
			return
		}
		_, _ = fmt.Fprintln(stderr, orgReseatClaudeConfigDirMismatchNote(got, value, set, env.setClaudeConfigDir))
	default:
		if !env.setClaudeConfigDir {
			return
		}
		if st, err := readOrgHerdrServerStatusWithin(orgHerdrStatusTimeout); err == nil && st.Running {
			_, _ = fmt.Fprintln(stderr, orgReseatClaudeConfigDirRunningNote)
		}
	}
}

// orgReseatHerdrServerLine is the line of what got read of the herdr
// server's CLAUDE_CONFIG_DIR: `herdr server: CLAUDE_CONFIG_DIR=<value>`,
// `herdr server: CLAUDE_CONFIG_DIR=unset`, or
// `herdr server: CLAUDE_CONFIG_DIR cannot be read: <reason>`.
func orgReseatHerdrServerLine(got orgHerdrServerClaudeConfigDir) string {
	switch got.state {
	case orgHerdrServerEnvSet:
		return "herdr server: " + orgClaudeConfigDirName + "=" + got.value
	case orgHerdrServerEnvUnset:
		return "herdr server: " + orgClaudeConfigDirName + "=unset"
	}
	return "herdr server: " + orgClaudeConfigDirName + " cannot be read: " + got.reason
}

// orgReseatWantedClaudeConfigDir is the CLAUDE_CONFIG_DIR the seats are to
// use, and whether there is one: env's when it sets it (none for ""), else
// this command's own.
func orgReseatWantedClaudeConfigDir(env orgHerdrServerEnvOverride) (value string, set bool) {
	if env.setClaudeConfigDir {
		return env.claudeConfigDir, env.claudeConfigDir != ""
	}
	return os.LookupEnv(orgClaudeConfigDirName)
}

// orgReseatHasWantedClaudeConfigDir says whether got, a CLAUDE_CONFIG_DIR
// that was read (orgHerdrServerEnvSet or orgHerdrServerEnvUnset), is the
// wanted one (value, set; orgReseatWantedClaudeConfigDir): both unset, or
// both set to the same value.
func orgReseatHasWantedClaudeConfigDir(got orgHerdrServerClaudeConfigDir, value string, set bool) bool {
	return (got.state == orgHerdrServerEnvSet) == set && (!set || got.value == value)
}

// orgReseatClaudeConfigDirText is how the notes name a CLAUDE_CONFIG_DIR:
// `CLAUDE_CONFIG_DIR=<value>` when set, else `no CLAUDE_CONFIG_DIR`.
func orgReseatClaudeConfigDirText(value string, set bool) string {
	if !set {
		return "no " + orgClaudeConfigDirName
	}
	return orgClaudeConfigDirName + "=" + value
}

// orgReseatClaudeConfigDirMismatchNote is the note of a reseat without
// --restart-herdr whose herdr server runs with another CLAUDE_CONFIG_DIR than
// the wanted one (value, set): the seats start with the server's. given says
// the wanted one is --claude-config-dir's, which applies only to a server
// reseat starts. Each side reads `CLAUDE_CONFIG_DIR=<value>` or
// `no CLAUDE_CONFIG_DIR` (orgReseatClaudeConfigDirText), and both fit the
// sentence after "run with".
func orgReseatClaudeConfigDirMismatchNote(got orgHerdrServerClaudeConfigDir, value string, set, given bool) string {
	server := orgReseatClaudeConfigDirText(got.value, got.state == orgHerdrServerEnvSet)
	if given {
		return fmt.Sprintf("note: the herdr server runs with %s, and the seats start with its environment, so they will not "+
			"run with %s, which --claude-config-dir asks for: it applies only when ralph starts a stopped herdr server; add "+
			"--restart-herdr to start the server again with it", server, orgReseatClaudeConfigDirText(value, set))
	}
	return fmt.Sprintf("note: the herdr server runs with %s, and the seats start with its environment, so they will not run "+
		"with %s from this command's environment; add --restart-herdr to start the server again with this command's "+
		"environment", server, orgReseatClaudeConfigDirText(value, set))
}

// orgReseatClaudeConfigDirRunningNote is the note of a reseat given
// --claude-config-dir without --restart-herdr while a herdr server runs
// whose CLAUDE_CONFIG_DIR could not be read.
const orgReseatClaudeConfigDirRunningNote = "note: the herdr server runs, and --claude-config-dir applies only when ralph " +
	"starts a stopped herdr server, so the seats start with the server's environment as it is; add --restart-herdr to " +
	"start the server again with it"

// printOrgReseatRestartedHerdrServerNotes writes on stderr what a real
// reseat says about the herdr server it started again (restarted,
// ReseatResult.HerdrRestarted), once its result is printed, so after
// `started: herdr server` and the seats' lines: the CLAUDE_CONFIG_DIR of the
// server that runs now, read again with orgHerdrServerClaudeConfigDirReader,
// in the line printed before the reseat (orgReseatHerdrServerLine). When the
// value was read and is not the wanted one (orgReseatWantedClaudeConfigDir,
// compared with orgReseatHasWantedClaudeConfigDir, as before the reseat),
// orgReseatRestartedClaudeConfigDirMismatchNote follows: the gate starts a
// stopped server with the wanted value, so a server with another value is
// most likely one that another process started between the stop and the
// gate, and the seats started with its environment. Without restarted, and
// with a nil reader (the tests' TestMain), it prints nothing. A read that
// fails is never an error: its line says why the value cannot be read.
func printOrgReseatRestartedHerdrServerNotes(stderr io.Writer, restarted bool, env orgHerdrServerEnvOverride) {
	read := orgHerdrServerClaudeConfigDirReader
	if !restarted || read == nil {
		return
	}
	got := read(context.Background())
	_, _ = fmt.Fprintln(stderr, orgReseatHerdrServerLine(got))
	if got.state == orgHerdrServerEnvUnreadable {
		return
	}
	value, set := orgReseatWantedClaudeConfigDir(env)
	if orgReseatHasWantedClaudeConfigDir(got, value, set) {
		return
	}
	_, _ = fmt.Fprintln(stderr, orgReseatRestartedClaudeConfigDirMismatchNote(got, value, set, env.setClaudeConfigDir))
}

// orgReseatRestartedClaudeConfigDirMismatchNote is the note of a reseat that
// started the herdr server again and then read another CLAUDE_CONFIG_DIR of
// it (got) than the wanted one (value, set; given as for
// orgReseatClaudeConfigDirMismatchNote).
func orgReseatRestartedClaudeConfigDirMismatchNote(got orgHerdrServerClaudeConfigDir, value string, set, given bool) string {
	wanted := "this command's environment has " + orgReseatClaudeConfigDirText(value, set)
	if given {
		wanted = "--claude-config-dir asks for " + orgReseatClaudeConfigDirText(value, set)
	}
	return fmt.Sprintf("note: after the restart the herdr server runs with %s, but %s: another process may have started the "+
		"server after reseat stopped it and before reseat could start it, and any seat this reseat started runs with the "+
		"server's environment",
		orgReseatClaudeConfigDirText(got.value, got.state == orgHerdrServerEnvSet), wanted)
}
