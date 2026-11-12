package cli

import (
	"fmt"
	"io"
)

// printCodexHookTrustNotice writes one line to w when a codex agent working in
// dir would run without some of the hooks of the .codex/hooks.json codex
// reads there (plan 2026-11-12-codex-hook-trust-check, D5, D6, and AC6; issue
// #310). `ralph org spawn`, both forms of `ralph org start`, and `ralph org
// director start` call it once, right after they resolve the driver, with the
// driver they resolved: verb names the command ("org spawn"), seat the seat
// (or "director"), and dir the directory the agent will run in. It does
// nothing unless driverName is "codex". The line calls the agent "codex
// seat <seat>", or "codex director" for codexHookTrustDirectorVerb, since the
// director is not an org seat (codexHookTrustAgent).
//
// It reads codex's user config from the path doctorCodexSandboxEnv resolves,
// the one `ralph doctor` reads (internal/cli's TestMain pins it to a path that
// does not exist, so no test reads the developer's ~/.codex), and prints only
// when codexHookTrustFor classified every hook and some are not approved or
// are turned off; a missing config counts, since then none is approved.
// Nothing is printed when the path cannot be resolved, when the hooks.json is
// missing, unreadable, or invalid, or when the config cannot be read or
// decoded: `ralph doctor` reports those. The line names paths and counts,
// never the config's content, and the command goes on as before.
func printCodexHookTrustNotice(w io.Writer, driverName, verb, seat, dir string) {
	if driverName != "codex" {
		return
	}
	env, err := doctorCodexSandboxEnv()
	if err != nil {
		return
	}
	report := codexHookTrustFor(dir, env)
	if notice := codexHookTrustNotice(report, codexConfigDisplayPath(env.Home, env.ConfigPath), verb, seat); notice != "" {
		_, _ = fmt.Fprintln(w, notice)
	}
}

// codexHookTrustNotice words printCodexHookTrustNotice's line for report, or
// returns "" when there is nothing to say. cfgDisplay is codex's user config
// as the line names it (codexConfigDisplayPath). The line gives the counts
// the doctor check gives and sends the reader to `ralph doctor` for the keys
// and the way to approve or turn the hooks back on, so it stays one line.
func codexHookTrustNotice(report codexHookTrustReport, cfgDisplay, verb, seat string) string {
	if !report.Classified() {
		return ""
	}
	m, n, k := len(report.Slots), len(report.NotApproved), len(report.Off)
	if n+k == 0 {
		return ""
	}
	var head string
	switch {
	case report.ConfigState == codexHookSourceMissing:
		head = fmt.Sprintf("there is no codex user config at %s to approve the %s in %s", cfgDisplay, codexHookNoun(m), report.HooksPath)
	case n > 0 && k > 0:
		head = fmt.Sprintf("%d of %s in %s %s not approved in %s, %d turned off (enabled = false)",
			n, codexHookNoun(m), report.HooksPath, codexHookIsAre(n), cfgDisplay, k)
	case n > 0:
		head = fmt.Sprintf("%d of %s in %s %s not approved in %s", n, codexHookNoun(m), report.HooksPath, codexHookIsAre(n), cfgDisplay)
	default:
		head = fmt.Sprintf("%d of %s in %s %s turned off (enabled = false) in %s", k, codexHookNoun(m), report.HooksPath, codexHookIsAre(k), cfgDisplay)
	}
	skip, key, them := "they do not run", "the keys", "them"
	if n+k == 1 {
		skip, key, them = "it does not run", "the key", "it"
	}
	var fix string
	switch {
	case n > 0 && k > 0:
		fix = "trust or re-enable " + them
	case n > 0:
		fix = "trust " + them
	default:
		fix = "re-enable " + them
	}
	who, where := codexHookTrustAgent(verb, seat)
	return fmt.Sprintf("note: %s: %s: %s, so %s in %s; run 'ralph doctor' for %s and how to %s", verb, who, head, skip, where, key, fix)
}

// codexHookTrustDirectorVerb is the verb `ralph org director start` passes
// to printCodexHookTrustNotice.
const codexHookTrustDirectorVerb = "org director start"

// codexHookTrustAgent names the agent codexHookTrustNotice's line is about
// and where its hooks do not run: "codex director" and "the director" for
// codexHookTrustDirectorVerb, and "codex seat <seat>" and "the seat" for
// every other verb (spawn and both forms of start). It goes by the verb, not
// the seat id, because `ralph org spawn` may name a seat "director".
func codexHookTrustAgent(verb, seat string) (who, where string) {
	if verb == codexHookTrustDirectorVerb {
		return "codex director", "the director"
	}
	return "codex seat " + seat, "the seat"
}

// orgCodexHookTrustNoticeHelp is the paragraph `ralph org spawn --help`,
// `ralph org start --help`, and `ralph org director start --help` end with,
// about the note printCodexHookTrustNotice writes (issue #310; plan
// 2026-11-12-codex-hook-trust-check, AC8).
const orgCodexHookTrustNoticeHelp = "When the driver of the agent the command starts resolves to codex, the\n" +
	"command also reads the .codex/hooks.json codex reads in the directory the\n" +
	"agent runs in (the main checkout's, for a linked worktree) and looks up\n" +
	"each hook's approval in [hooks.state] of codex's user config\n" +
	"($CODEX_HOME/config.toml, else ~/.codex/config.toml). codex does not run\n" +
	"a hook that is not approved or is turned off (enabled = false), so when\n" +
	"one is, or that config does not exist, one note on stderr gives the\n" +
	"counts; the command goes on either way. The Codex hook trust check of\n" +
	"ralph doctor lists the keys and how to approve the hooks. There is no\n" +
	"note for a claude agent, or when the hooks.json is missing or invalid or\n" +
	"the config cannot be read or decoded (ralph doctor reports those)."
