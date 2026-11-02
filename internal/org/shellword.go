package org

import (
	"regexp"
	"strings"
)

// The shell words of the commands ralph prints: ShellWord, the one function
// that quotes a word only when needed, shellQuote, which always quotes, and
// VerbCommand, which builds a ralph org command on ShellWord.

// shellPlainWord matches a word that no shell reads specially anywhere in
// it, which ShellWord leaves unquoted.
var shellPlainWord = regexp.MustCompile(`^[A-Za-z0-9_./:@+,-]+$`)

// shellQuote returns s as one POSIX shell word: s in single quotes, where
// each single quote of s closes the quotes, is written escaped, and opens
// them again.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ShellWord is s as one shell word in a printed command: as it is when
// shellPlainWord matches it, else in single quotes (shellQuote). It is the one
// function that quotes a word only when needed, and internal/org and
// internal/cli use it for the commands they print: a Manual command, a command
// a refusal names to run again, and the hints of the CLI's own messages.
// shellQuote, which always quotes, is still used directly by the sentences of
// escalate, approve, watchgate, and others, which print a value in quotes even
// when ShellWord would leave it bare.
func ShellWord(s string) string {
	if shellPlainWord.MatchString(s) {
		return s
	}
	return shellQuote(s)
}

// VerbCommand is `ralph org <verbAndArgs>`, followed by --state-dir
// <hintStateDir> when hintStateDir is not "", the dir one shell word
// (ShellWord); verbAndArgs is written as it is. It builds every ralph org
// command that ralph org merge and ralph org cleanup print with
// MergeParams.HintStateDir or CleanupParams.HintStateDir, which the CLI sets
// only when it was given --state-dir: the ones to run again
// (orgRetryCommand), and those of the verbs that read or change the same
// ledger, such as ralph org merge-order --all, ralph org inbox, and ralph org
// status --org-id <org_id>. A ledger [org].state_dir names gets no
// --state-dir, since the bare command reads the same file.
//
// It does not escape: the caller passes the whole sentence that holds the
// command through PrintableInboxText (the CLI uses orgVerbHint, which escapes
// the dir).
func VerbCommand(verbAndArgs, hintStateDir string) string {
	command := "ralph org " + verbAndArgs
	if hintStateDir != "" {
		command += " --state-dir " + ShellWord(hintStateDir)
	}
	return command
}
