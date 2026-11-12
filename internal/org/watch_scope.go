package org

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// The scope-change condition of a seat in an org that holds a reservation
// (plan 2026-11-12-watch-alert-dedupe, D5 and D6). An org without one keeps
// the raw `git status --porcelain` comparison of evaluateSeat.

// GitRunFunc runs git with args in cwd and returns its standard output; a
// git that exits non-zero, or that cannot run, is an error. It is the git of
// the scope-change condition of an org that holds a reservation
// (evaluateReservedScope). Injectable like GitStatusFunc; the real one is
// realGitRun.
type GitRunFunc func(cwd string, args ...string) (string, error)

// realGitRun is GitRunFunc's real implementation: `git --no-optional-locks
// <args>` in cwd, bounded like realGitStatus (runGitBounded). The option
// keeps `git status` from taking the index lock for its optional refresh,
// so a seat's own git command in the same worktree is not refused while the
// watch reads it, as the gate reads a worktree (gateGit).
func (o *Org) realGitRun(cwd string, args ...string) (string, error) {
	return o.runGitBounded(cwd, append([]string{"--no-optional-locks"}, args...)...)
}

// runGitBounded runs git with args in cwd and returns its standard output,
// killed when driverCallTimeout passes like each herdr and agmsg call of
// the pulse (callWithTimeout), so a git that does not return cannot hold the
// cycle. An exit 0 whose pipe a child still held is a success, the way
// driver.ExecRunner reads it. realGitStatus and realGitRun share it.
func (o *Org) runGitBounded(cwd string, args ...string) (string, error) {
	var out []byte
	err := o.callWithTimeout(func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = cwd
		cmd.WaitDelay = gitStatusWaitDelay
		var runErr error
		out, runErr = cmd.Output()
		if errors.Is(runErr, exec.ErrWaitDelay) && ctx.Err() == nil {
			runErr = nil
		}
		return runErr
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parsePorcelainZ parses the output of `git status --porcelain=v1 -z
// --untracked-files=all`: entries separated by NUL, each `XY <path>`. When
// X or Y is R (renamed) or C (copied), the field after the entry is the
// original path, and it is returned as an entry of its own with the same
// XY, so both sides of a rename are there. Entries come back in order, each
// `XY <path>`; empty fields (the one after the last NUL) and fields too
// short to be an entry are skipped. The paths are relative to the
// repository root whatever the directory git ran in, and -z writes them
// as they are, with no quoting.
func parsePorcelainZ(out string) []string {
	fields := strings.Split(out, "\x00")
	var entries []string
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if len(field) < 4 || field[2] != ' ' {
			continue
		}
		entries = append(entries, field)
		if x, y := field[0], field[1]; x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			if i+1 < len(fields) && fields[i+1] != "" {
				i++
				entries = append(entries, field[:3]+fields[i])
			}
		}
	}
	return entries
}

// splitNUL returns the non-empty NUL-separated fields of out, the output of
// a git command run with -z.
func splitNUL(out string) []string {
	var fields []string
	for field := range strings.SplitSeq(out, "\x00") {
		if field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

// scopePathInside reports whether path p, relative to the repository root,
// is inside the reservation reserved (normalized paths, NormalizeReservePaths):
// some reserved path covers it. That is reservePathsOverlap with the
// reserved path no longer than p, which for a file p is the overlap rule
// itself (`.`, a directory p is under, or p itself) and for a directory p
// (`?? sub/`, an untracked repository inside the worktree) leaves out a
// reserved path inside p, which covers only part of it.
func scopePathInside(reserved []string, p string) bool {
	for _, r := range reserved {
		if reservePathsOverlap(r, p) && len(r) <= len(p) {
			return true
		}
	}
	return false
}

// ownRecordCandidate reports whether path p can be a record the org's
// branch wrote (plan 2026-11-12-watch-alert-dedupe, D5): a file under
// docs/plans/ or docs/reports/, where the role prompts have the seats write
// the feature plan and the reports outside the reservation, and not under
// docs/plans/templates/ or docs/reports/templates/, which hold the inputs
// the skills read. Whether it is one is up to the merge base
// (scopeRecordsAtBase): only a candidate that did not exist there is.
func ownRecordCandidate(p string) bool {
	if strings.HasSuffix(p, "/") {
		return false
	}
	for _, dir := range []string{"docs/plans/", "docs/reports/"} {
		if strings.HasPrefix(p, dir) && !strings.HasPrefix(p, dir+"templates/") {
			return true
		}
	}
	return false
}

// printableScopePath returns p with every rune that would end a line or
// reach a terminal as a control character (unicode.IsControl, and the line
// breaks of isInboxLineBreak) written as its Go escape (\n, \x1b, \u2028),
// and every byte that is not valid UTF-8 as \xNN, so a path is always one
// printable line of the ALERT body and of the recurred note.
func printableScopePath(p string) string {
	var b strings.Builder
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRuneInString(p[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&b, `\x%02x`, p[i])
		case unicode.IsControl(r) || isInboxLineBreak(r):
			quoted := strconv.QuoteRune(r)
			b.WriteString(quoted[1 : len(quoted)-1])
		default:
			b.WriteString(p[i : i+size])
		}
		i += size
	}
	return b.String()
}

// scopeNotePrefix starts the note of a scope_change recurrence
// (scopeOutsideNote).
const scopeNotePrefix = "outside reservation: "

// scopeNoteMore is the end of a scope_change recurrence note that left out
// the last n paths.
func scopeNoteMore(n int) string {
	return fmt.Sprintf("… (+%d more)", n)
}

// scopeOutsideNote returns the note of the recurred event a scope_change
// records on its open inbox item (raiseOrClear's recurNote): `outside
// reservation: <p1>, <p2>, ...` with each path printable
// (printableScopePath), so it is one line with no control character, as
// AppendRecurred requires. When the paths do not all fit in
// InboxNoteMaxRunes characters, it keeps as many as fit from the start and
// ends with `… (+N more)` for the N it left out. No paths is no note.
func scopeOutsideNote(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(scopeNotePrefix)
	runes := utf8.RuneCountInString(scopeNotePrefix)
	for i, p := range paths {
		item := printableScopePath(p)
		if i > 0 {
			item = ", " + item
		}
		n := utf8.RuneCountInString(item)
		room := InboxNoteMaxRunes - runes - n
		if rest := len(paths) - i - 1; rest > 0 {
			room -= utf8.RuneCountInString(", " + scopeNoteMore(rest))
		}
		if room < 0 {
			// The previous path was kept only with room after it for this
			// end, which counts this path and the ones after it, so it fits.
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(scopeNoteMore(len(paths) - i))
			return b.String()
		}
		b.WriteString(item)
		runes += n
	}
	return b.String()
}

// scopeEntry is one change outside the reservation: path is its path, and
// committed is set when a commit since the previous cycle changed it rather
// than a porcelain entry naming it.
type scopeEntry struct {
	path      string
	committed bool
}

// scopeOutsideEntries returns the paths of the entries of porcelain
// (parsePorcelainZ) and the committed paths that are not inside reserved
// (scopePathInside), porcelain first.
func scopeOutsideEntries(reserved, porcelain, committed []string) []scopeEntry {
	var outside []scopeEntry
	for _, entry := range porcelain {
		if p := entry[3:]; !scopePathInside(reserved, p) {
			outside = append(outside, scopeEntry{path: p})
		}
	}
	for _, p := range committed {
		if !scopePathInside(reserved, p) {
			outside = append(outside, scopeEntry{path: p, committed: true})
		}
	}
	return outside
}

// isGitObjectName reports whether s is a full object name: 40 (SHA-1) or 64
// (SHA-256) lowercase hex digits. A HEAD read back from the status file is
// passed to git only when it is one, so it can never be read as an option.
func isGitObjectName(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool { return (r < '0' || r > '9') && (r < 'a' || r > 'f') }) < 0
}

// scopeDefaultBranchRef returns the ref of origin's default branch in
// worktree's repository: the ref refs/remotes/origin/HEAD points to (`git
// symbolic-ref --quiet`), else the first of refs/remotes/origin/main and
// refs/remotes/origin/master, each only when it names a commit (`git
// rev-parse --verify --quiet <ref>^{commit}`). origin/HEAD can point to a
// ref that is gone (a default branch renamed upstream and its old
// remote-tracking branch pruned), and every `git log ... --not <ref>` with
// it would fail, so its target is checked too. The full ref name is used,
// so a local branch called origin/main cannot take its place. ok is false
// when there is none.
func scopeDefaultBranchRef(git GitRunFunc, worktree string) (ref string, ok bool) {
	namesCommit := func(ref string) bool {
		_, err := git(worktree, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
		return err == nil
	}
	if out, err := git(worktree, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil {
		if ref := strings.TrimSpace(out); strings.HasPrefix(ref, "refs/remotes/origin/") && ref != "refs/remotes/origin/HEAD" && namesCommit(ref) {
			return ref, true
		}
	}
	for _, ref := range []string{"refs/remotes/origin/main", "refs/remotes/origin/master"} {
		if namesCommit(ref) {
			return ref, true
		}
	}
	return "", false
}

// scopeCommittedPaths returns the paths changed by the commits from prev to
// head that are not on origin's default branch, the ref defaultRef resolves
// (scopeDefaultBranchRef): `git log --no-show-signature --cc --no-renames
// --no-relative --name-only -z --format= <prev>..<head> --not <ref> --`, both
// sides of a move included. The leader rebases the branch onto origin's
// default branch before its PR (plan 2026-11-12-watch-alert-dedupe, D6 and
// AC15), which moves HEAD past upstream's commits, and those are not the
// org's writes. A merge commit, such as one that merges origin's default
// branch, lists by --cc (the dense combined diff) only the paths whose merged
// result differs from every parent: the merge's own edits, such as a conflict
// resolved outside the reservation, while upstream's changes it only brings
// in stay out. With no ref it falls back to `git diff --no-renames --no-relative
// --name-only -z <prev> <head>`, which lists upstream's changes too, so the
// condition fails toward alerting. listed is false, with no paths, when that
// git fails (a git killed at the deadline, or a previous HEAD git no longer
// has). A path may come back more than once. --no-show-signature keeps a
// user's log.showSignature from mixing gpg lines into the names.
// --no-relative keeps a user's diff.relative from making the names relative
// to the seat's cwd, which would drop the siblings of that directory.
func scopeCommittedPaths(git GitRunFunc, worktree, prev, head string, defaultRef func() (string, bool)) (paths []string, listed bool) {
	args := []string{"diff", "--no-renames", "--no-relative", "--name-only", "-z", prev, head}
	if ref, ok := defaultRef(); ok {
		args = []string{"log", "--no-show-signature", "--cc", "--no-renames", "--no-relative", "--name-only", "-z", "--format=", prev + ".." + head, "--not", ref, "--"}
	}
	out, err := git(worktree, args...)
	if err != nil {
		return nil, false
	}
	return splitNUL(out), true
}

// scopeHeadKnown reports whether git has prev, a full object name, as a
// commit in worktree's repository: `git rev-parse --verify --quiet
// <prev>^{commit}`. A git that fails for any other reason reports false
// too.
func scopeHeadKnown(git GitRunFunc, worktree, prev string) bool {
	_, err := git(worktree, "rev-parse", "--verify", "--quiet", prev+"^{commit}")
	return err == nil
}

// scopeRecordsAtBase returns which of candidates existed at the merge base
// of worktree's HEAD and ref, origin's default branch
// (scopeDefaultBranchRef): `git merge-base HEAD <ref>`, then `git ls-tree
// --full-tree --name-only -z <base> -- <candidates>`, which matches each
// path literally. ok is false when any step fails, so the caller drops
// nothing (plan 2026-11-12-watch-alert-dedupe, AC14).
func scopeRecordsAtBase(git GitRunFunc, worktree, ref string, candidates []string) (existing map[string]bool, ok bool) {
	out, err := git(worktree, "merge-base", "HEAD", ref)
	if err != nil {
		return nil, false
	}
	base := strings.TrimSpace(out)
	if !isGitObjectName(base) {
		return nil, false
	}
	out, err = git(worktree, append([]string{"ls-tree", "--full-tree", "--name-only", "-z", base, "--"}, candidates...)...)
	if err != nil {
		return nil, false
	}
	existing = map[string]bool{}
	for _, p := range splitNUL(out) {
		existing[p] = true
	}
	return existing, true
}

// dropOwnRecords returns outside without the records the org's branch wrote
// (plan 2026-11-12-watch-alert-dedupe, D5): an entry whose path is an
// ownRecordCandidate that did not exist at the merge base
// (scopeRecordsAtBase) of HEAD and origin's default branch, the ref
// defaultRef resolves. A path that existed there (a template, another
// feature's plan, an earlier report) stays. With no candidate it runs no
// git; when there is no ref or the base cannot be read it drops nothing.
func dropOwnRecords(git GitRunFunc, worktree string, defaultRef func() (string, bool), outside []scopeEntry) []scopeEntry {
	var candidates []string
	for _, e := range outside {
		if ownRecordCandidate(e.path) && !slices.Contains(candidates, e.path) {
			candidates = append(candidates, e.path)
		}
	}
	if len(candidates) == 0 {
		return outside
	}
	ref, ok := defaultRef()
	if !ok {
		return outside
	}
	existing, ok := scopeRecordsAtBase(git, worktree, ref, candidates)
	if !ok {
		return outside
	}
	return slices.DeleteFunc(outside, func(e scopeEntry) bool {
		return ownRecordCandidate(e.path) && !existing[e.path]
	})
}

// evaluateReservedScope is the scope-change condition (e) of seat seatID,
// whose worktree is worktree, in org orgID that holds the reservation
// reserved (plan 2026-11-12-watch-alert-dedupe, D5 and D6). It collects the
// paths that changed:
//
//   - the entries of `git status --porcelain=v1 -z --untracked-files=all`
//     (parsePorcelainZ: both sides of a rename, each file of a new
//     directory);
//   - when the seat's snapshot holds the HEAD of the previous cycle and HEAD
//     moved since, the paths changed by the commits from it to the current
//     HEAD that are not on origin's default branch (scopeCommittedPaths:
//     `git log --cc --no-renames --name-only -z --format= <prev>..<cur>
//     --not <ref>`, both sides of a move included, and of a merge commit
//     only its own edits), so a rebase onto origin's default branch, or a
//     merge of it, does not report upstream's changes. With no default
//     branch it reads `git diff --no-renames --name-only -z <prev> <cur>`
//     instead. When that git fails, the snapshot keeps the previous HEAD,
//     so the next cycle lists the same commits again and the condition
//     fails toward alerting. Only when git no longer has the previous HEAD
//     (scopeHeadKnown: a rebase and a gc, or a status file of another
//     repository), which no later cycle could list from, does HEAD move on
//     with no commit paths.
//
// It keeps the paths outside reserved (scopeOutsideEntries), drops the
// records the org's branch wrote (dropOwnRecords), and keeps the rest,
// sorted, as the seat's baseline (ScopeOutsidePaths). origin's default
// branch (scopeDefaultBranchRef) is resolved at most once per call, and
// only when the commit paths or dropOwnRecords need it. The condition is
// active when the snapshot holds a previous baseline (ScopeOutsidePathsSeen)
// and a path outside is not in it, or a commit since the previous cycle
// changed a path outside; a path that leaves the set (a change put back) or
// whose porcelain status alone changes (a `git add`) does not raise it. A
// commit raises it even for a path already reported while it was
// uncommitted: the commit makes the outside change part of the branch the PR
// will carry, which the director should see. While the item of the earlier
// ALERT is open, that is a recurrence on that item (one `recurred` note), not
// a new item. The first cycle only records the baseline. The ALERT names the
// paths outside the reservation, one per line (truncateScopeOutput), and the
// note of a recurrence names them too (scopeOutsideNote).
//
// A porcelain that cannot be read leaves the cycle out, with the snapshot
// as it was, as an org without a reservation does. A HEAD that cannot be
// read (a repository with no commit yet) gives no commit paths and keeps
// the previous HEAD, so the next HEAD that is read is compared with it.
func (w *watchRun) evaluateReservedScope(status *watchStatusFile, orgID, seatID, worktree string, reserved []string, snap *watchSeatSnapshot, now time.Time) {
	git := w.gitRun
	if git == nil {
		git = w.org.realGitRun
	}
	porcelain, err := git(worktree, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return
	}
	head := ""
	if out, err := git(worktree, "rev-parse", "--verify", "-q", "HEAD"); err == nil && isGitObjectName(strings.TrimSpace(out)) {
		head = strings.TrimSpace(out)
	}
	defaultRef := sync.OnceValues(func() (string, bool) { return scopeDefaultBranchRef(git, worktree) })
	var committed []string
	keepHead := false
	if prev := snap.ScopeHead; isGitObjectName(prev) && head != "" && head != prev {
		var listed bool
		committed, listed = scopeCommittedPaths(git, worktree, prev, head, defaultRef)
		keepHead = !listed && scopeHeadKnown(git, worktree, prev)
	}

	outside := dropOwnRecords(git, worktree, defaultRef, scopeOutsideEntries(reserved, parsePorcelainZ(porcelain), committed))
	var paths []string
	committedOutside := false
	for _, e := range outside {
		paths = append(paths, e.path)
		committedOutside = committedOutside || e.committed
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)

	newOutside := slices.ContainsFunc(paths, func(p string) bool { return !slices.Contains(snap.ScopeOutsidePaths, p) })
	active := snap.ScopeOutsidePathsSeen && (newOutside || committedOutside)
	var message, note string
	if active {
		lines := make([]string, len(paths))
		for i, p := range paths {
			lines[i] = printableScopePath(p)
		}
		message = fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nseat %s changed paths outside the org's reservation:\n%s",
			orgID, seatID, condScopeChange, seatID, truncateScopeOutput(strings.Join(lines, "\n"), scopeChangeBodyBudget))
		note = scopeOutsideNote(paths)
	}
	w.raiseOrClear(status, orgID, seatID, condScopeChange, active, now, message, true, note)

	snap.ScopeOutsidePaths = paths
	snap.ScopeOutsidePathsSeen = true
	if head != "" && !keepHead {
		snap.ScopeHead = head
	}
}
