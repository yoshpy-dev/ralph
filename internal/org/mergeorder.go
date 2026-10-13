package org

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// The `ralph org merge-order` verb (MergeOrder), stage 7c-1 of plan
// docs/plans/active/2026-10-14-org-merge-order.md (FR-8 of
// docs/specs/2026-10-07-org-multi-org-director.md). It lists the feature
// branches of the ledger in the order their PRs may be merged now. A branch is
// ready when its latest gate record (gate.go's gateRecord) is a gate_passed
// whose head is the branch's tip on origin and whose base is the tip of
// origin's default branch now. The gate records are the only evidence it
// reads: a reviewer's or the leader's word does not count.
//
// The targets are branches (C3): the branch of every complete feature binding
// of the manifest, disbanded orgs included, and every branch a readable gate
// record names. origin's state comes from one `git ls-remote --symref origin
// HEAD 'refs/heads/*'` (C2, C7), each branch's PR from `gh pr list` when gh is
// there (C5), and both run through MergeOrderExec (C6). Evidence that cannot
// be read never makes a branch ready (C8).
//
// MergeOrder writes nothing, to the ledger or anywhere else, takes no lock,
// and never fetches. Neither the watch gate (watchgate.go) nor the approval
// check (approval.go) applies to it, since it starts no work.

// The classes of a MergeOrderEntry. MergeOrder lists them in this order
// (mergeClassOrder) and gives a branch the first that applies in another one
// (classifyMergeTarget).
const (
	// MergeClassReady: origin's branch tip is the head the gate passed, and
	// origin's default branch is the one the gate checked, still at its base.
	MergeClassReady = "ready"
	// MergeClassRebase: origin's default branch moved since the gate, or is
	// another branch than the one the gate checked.
	MergeClassRebase = "rebase"
	// MergeClassWrongBase: the branch's PR merges into another branch than
	// origin's default one.
	MergeClassWrongBase = "wrong_base"
	// MergeClassRegate: origin's branch tip is not the head the gate checked
	// (pushed after the gate).
	MergeClassRegate = "regate"
	// MergeClassUnreadable: the branch's latest gate record cannot be read, or
	// a corrupt manifest line follows it and may be a newer one.
	MergeClassUnreadable = "unreadable"
	// MergeClassFailed: the branch's latest gate record is a gate_failed.
	MergeClassFailed = "failed"
	// MergeClassNoRemote: origin has no such branch.
	MergeClassNoRemote = "no_remote"
	// MergeClassNoGate: the branch has no gate record.
	MergeClassNoGate = "no_gate"
	// MergeClassMerged and MergeClassClosed: the branch's PR is merged, or was
	// closed without a merge (gh). Shown only with MergeOrderParams.All.
	MergeClassMerged = "merged"
	MergeClassClosed = "closed"
)

// mergeClassOrder is the order MergeOrder lists the classes in.
var mergeClassOrder = []string{
	MergeClassReady, MergeClassRebase, MergeClassWrongBase, MergeClassRegate, MergeClassUnreadable,
	MergeClassFailed, MergeClassNoRemote, MergeClassNoGate, MergeClassMerged, MergeClassClosed,
}

const (
	// defaultMergeOrderLsRemoteTimeout and defaultMergeOrderGHTimeout bound the
	// ls-remote and each gh call when MergeOrderParams leaves them zero.
	defaultMergeOrderLsRemoteTimeout = 30 * time.Second
	defaultMergeOrderGHTimeout       = 15 * time.Second
	// mergeOrderWaitDelay is realMergeOrderExec's exec.Cmd.WaitDelay: how
	// long after the kill, or after the command exited, it waits for the
	// output pipes, which a child (ssh under git) can hold.
	mergeOrderWaitDelay = 2 * time.Second
	// mergeOrderLsRemoteCommand names the ls-remote in messages.
	mergeOrderLsRemoteCommand = "git ls-remote --symref origin HEAD 'refs/heads/*'"
	// mergeOrderGHFields are the PR fields MergeOrder asks gh for.
	mergeOrderGHFields = "number,state,url,headRefOid,baseRefName,isCrossRepository"
)

// MergeOrderExec runs the command name with args in dir, with env (NAME=value
// entries) added to the environment, and returns its stdout. The error is for
// a command that could not start (errors.Is exec.ErrNotFound when it is not
// installed), exited non-zero, or was stopped by ctx; it holds the command's
// stderr. MergeOrder runs git and gh through it; tests pass a fake.
type MergeOrderExec func(ctx context.Context, dir string, env []string, name string, args ...string) (stdout string, err error)

// MergeOrderParams describes one `ralph org merge-order` call.
type MergeOrderParams struct {
	// Dir is the repository directory git and gh run in; empty means the
	// current directory.
	Dir string
	// All also lists the branches whose PR is merged or closed.
	All bool
	// Exec runs git and gh; nil means realMergeOrderExec.
	Exec MergeOrderExec
	// LsRemoteTimeout bounds the ls-remote, GHTimeout each gh call; zero or
	// less means 30 s and 15 s.
	LsRemoteTimeout, GHTimeout time.Duration
}

// MergeOrderResult is what MergeOrder found. The CLI prints it, or marshals
// it as it is for --json.
type MergeOrderResult struct {
	// DefaultBranch is the branch origin's HEAD names, DefaultHead its tip.
	DefaultBranch string `json:"default_branch"`
	DefaultHead   string `json:"default_head"`
	// ListedAt is when the ls-remote started, UTC RFC3339.
	ListedAt string `json:"listed_at"`
	// GHAvailable reports that gh was found. False when it was not, and when
	// there was no branch to ask it about.
	GHAvailable bool `json:"gh_available"`
	// Entries are the branches shown, in merge order.
	Entries []MergeOrderEntry `json:"entries"`
	// Notes are what the reader should know about the whole list, one
	// sentence each.
	Notes []string `json:"notes"`
}

// MergeOrderEntry is one branch of the merge order.
type MergeOrderEntry struct {
	// Rank is the branch's 1-based position in Entries.
	Rank  int    `json:"rank"`
	Class string `json:"class"` // MergeClass*
	// Branch is the feature branch; RemoteHead its tip on origin, "" when
	// origin has no such branch.
	Branch     string `json:"branch"`
	RemoteHead string `json:"remote_head"`
	// Detail says, in one sentence, why the branch has its class and what to
	// do next.
	Detail string `json:"detail"`
	// PR is the branch's PR (pickMergePR), nil when gh was not there,
	// failed, or found none from this repository.
	PR *MergeOrderPR `json:"pr,omitempty"`
	// Gate is the branch's latest gate record, nil when it has none.
	Gate *MergeOrderGate `json:"gate,omitempty"`
	// Binding is the latest complete feature binding of the branch, nil
	// when only a gate record names it.
	Binding *MergeOrderBinding `json:"binding,omitempty"`
}

// MergeOrderPR is the PR gh reports for a branch.
type MergeOrderPR struct {
	Number      int    `json:"number"`
	State       string `json:"state"` // OPEN, MERGED or CLOSED
	URL         string `json:"url"`
	HeadRefOid  string `json:"head_ref_oid"`
	BaseRefName string `json:"base_ref_name"`
}

// MergeOrderGate is a branch's latest gate record. Head, Base, BaseRef and
// Reason are empty when it cannot be read.
type MergeOrderGate struct {
	Event    string `json:"event"` // gate_passed or gate_failed
	Readable bool   `json:"readable"`
	Head     string `json:"head"`
	Base     string `json:"base"`
	BaseRef  string `json:"base_ref"`
	Reason   string `json:"reason"` // GateReason*, "" on gate_passed
	TS       string `json:"ts"`
}

// MergeOrderBinding names the feature binding a branch was started for.
type MergeOrderBinding struct {
	OrgID   string `json:"org_id"`
	Feature string `json:"feature"`
	Split   string `json:"split"`
	TS      string `json:"ts"`
}

// MergeOrder runs `ralph org merge-order`. It reads the manifest line by line
// (readMergeLedger) and collects the target branches with their latest
// binding and gate record (collectMergeTargets). With no target it returns
// at once, with a note and no git call. Otherwise it reads origin with one
// ls-remote in p.Dir (listOriginBranches), asks gh for each branch's PR
// (lookupMergePRs), classifies each branch (classifyMergeTarget) and orders
// them (orderMergeEntries).
//
// The error is for a manifest that cannot be read, an ls-remote that fails,
// times out or names no default branch, and ctx done: nothing is classified
// then. A gh that is missing or fails only leaves PRs out, with a note.
func (o *Org) MergeOrder(ctx context.Context, p MergeOrderParams) (MergeOrderResult, error) {
	ledger, err := readMergeLedger(o.Manifest.Path())
	if err != nil {
		return MergeOrderResult{}, fmt.Errorf("org: merge-order: read manifest: %w", err)
	}
	targets, unattached := collectMergeTargets(ledger)
	ledgerNotes := mergeLedgerNotes(len(ledger.corruptAt), unattached)
	res := MergeOrderResult{Entries: []MergeOrderEntry{}, Notes: []string{}}
	if len(targets) == 0 {
		res.Notes = append(res.Notes, "no feature branches in the ledger: a branch is listed once ralph org start --plan "+
			"binds an org to it or ralph org gate records a gate on it")
		res.Notes = append(res.Notes, ledgerNotes...)
		return res, nil
	}
	run := p.Exec
	if run == nil {
		run = realMergeOrderExec
	}
	res.ListedAt = o.now()
	remote, err := listOriginBranches(ctx, p, run)
	if err != nil {
		return MergeOrderResult{}, err
	}
	res.DefaultBranch, res.DefaultHead = remote.defaultBranch, remote.defaultHead
	var ghNotes []string
	if res.GHAvailable, ghNotes, err = lookupMergePRs(ctx, p, run, targets); err != nil {
		return MergeOrderResult{}, err
	}
	entries, hidden := orderMergeEntries(targets, remote, p.All)
	res.Entries = entries
	res.Notes = append(append(res.Notes, ghNotes...), ledgerNotes...)
	res.Notes = append(res.Notes, mergeOrderListNotes(res, hidden)...)
	return res, nil
}

// mergeLedger is the manifest as merge-order reads it: the events that parse,
// in file order, and for each line that does not, the number of events
// before it (so a line at corruptAt[k] comes after events[corruptAt[k]-1]).
type mergeLedger struct {
	events    []ManifestEvent
	corruptAt []int
}

// readMergeLedger reads the manifest at path with readJSONLines, keeping the
// position of each corrupt line, which ManifestStore.Read only counts (C8). A
// missing manifest reads as empty.
func readMergeLedger(path string) (mergeLedger, error) {
	var l mergeLedger
	_, err := readJSONLines(path, func(line []byte) error {
		var ev ManifestEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			l.corruptAt = append(l.corruptAt, len(l.events))
			return err
		}
		l.events = append(l.events, ev)
		return nil
	})
	return l, err
}

// mergeTarget is one target branch while MergeOrder works on it.
type mergeTarget struct {
	branch     string
	binding    *MergeOrderBinding // latest complete binding, nil for none
	bindingPos int                // its index in the events
	gate       *mergeGateRecord   // latest gate record attached, nil for none
	// corruptAfterGate: gate is readable and a corrupt line follows it.
	corruptAfterGate bool
	pr               *MergeOrderPR
}

// mergeGateRecord is one real gate event attached to a branch: record is
// gateRecord{} when it cannot be read.
type mergeGateRecord struct {
	pos      int
	ev       ManifestEvent
	record   gateRecord
	readable bool
}

// pos is t's place within its class: the index of its latest gate record, or
// of its latest binding when it has no gate record.
func (t *mergeTarget) pos() int {
	if t.gate != nil {
		return t.gate.pos
	}
	return t.bindingPos
}

// collectMergeTargets walks l's real org-level events in order and returns
// the target branches in the order they first appear (C3, C8). A complete
// binding (a scope_reserved read the way reserve.go reads it) makes its
// branch a target and becomes the branch's binding and its org's latest one.
// A gate record is readable when gateRecordFromDetails reads it and its event
// name agrees with its reason; it attaches to its branch= branch, which it
// makes a target. One that is not attaches to the branch of its org's latest
// complete binding before it, since the gate runs only on that branch; with
// no such binding it attaches nowhere and counts in unattached. A branch's
// gate is the latest record attached to it.
func collectMergeTargets(l mergeLedger) (targets []*mergeTarget, unattached int) {
	byBranch := make(map[string]*mergeTarget)
	target := func(branch string) *mergeTarget {
		t, ok := byBranch[branch]
		if !ok {
			t = &mergeTarget{branch: branch}
			byBranch[branch] = t
			targets = append(targets, t)
		}
		return t
	}
	boundBranch := make(map[string]string) // org_id -> its latest complete binding's branch so far
	for i, ev := range l.events {
		if ev.DryRun || ev.OrgID == "" || ev.SeatID != "" {
			continue
		}
		switch ev.Event {
		case EventScopeReserved:
			f := reservationFromEvent(ev).Feature
			if f == nil || !f.Complete() {
				continue
			}
			t := target(f.Branch)
			t.binding = &MergeOrderBinding{OrgID: ev.OrgID, Feature: f.Feature, Split: f.Split, TS: ev.TS}
			t.bindingPos = i
			boundBranch[ev.OrgID] = f.Branch
		case EventGatePassed, EventGateFailed:
			g := &mergeGateRecord{pos: i, ev: ev}
			g.record, g.readable = gateRecordFromDetails(ev.Details)
			g.readable = g.readable && g.record.event() == ev.Event
			branch := g.record.Branch
			if !g.readable {
				g.record = gateRecord{}
				var bound bool
				if branch, bound = boundBranch[ev.OrgID]; !bound {
					unattached++
					continue
				}
			}
			target(branch).gate = g
		}
	}
	if n := len(l.corruptAt); n > 0 {
		lastCorrupt := l.corruptAt[n-1]
		for _, t := range targets {
			t.corruptAfterGate = t.gate != nil && t.gate.readable && lastCorrupt > t.gate.pos
		}
	}
	return targets, unattached
}

// mergeLedgerNotes are the notes about the manifest itself: its corrupt
// lines and the unreadable gate records attached to no branch.
func mergeLedgerNotes(corrupt, unattached int) []string {
	var notes []string
	if corrupt > 0 {
		notes = append(notes, fmt.Sprintf("the manifest has %d corrupt line(s) that cannot be read: a branch whose latest gate "+
			"record comes before one is unreadable until ralph org gate records a new one", corrupt))
	}
	if unattached > 0 {
		notes = append(notes, fmt.Sprintf("%d gate record(s) cannot be read and their org had no feature binding before them, "+
			"so they are attached to no branch", unattached))
	}
	return notes
}

// mergeRemote is origin's state as the ls-remote listed it.
type mergeRemote struct {
	defaultBranch string            // the branch origin's HEAD names
	defaultHead   string            // its tip
	heads         map[string]string // branch name -> tip, every refs/heads/ ref
}

// listOriginBranches runs the ls-remote in p.Dir, with GIT_TERMINAL_PROMPT=0
// and p.LsRemoteTimeout, and reads origin's default branch from its HEAD
// symref (C7). The error says why origin's state is unknown: the command
// failed or timed out (with its stderr), ctx is done, origin's HEAD symref is
// not in the output, it names no branch, or it names one the output does not
// list.
func listOriginBranches(ctx context.Context, p MergeOrderParams, run MergeOrderExec) (mergeRemote, error) {
	timeout := p.LsRemoteTimeout
	if timeout <= 0 {
		timeout = defaultMergeOrderLsRemoteTimeout
	}
	lsCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := run(lsCtx, p.Dir, []string{"GIT_TERMINAL_PROMPT=0"}, "git", "ls-remote", "--symref", "origin", "HEAD", "refs/heads/*")
	where := cmp.Or(p.Dir, ".")
	switch {
	case ctx.Err() != nil:
		return mergeRemote{}, mergeOrderInterrupted(ctx, "the ls-remote ran")
	case err != nil && lsCtx.Err() != nil:
		return mergeRemote{}, fmt.Errorf("org: merge-order: %s in %s timed out after %s (%w): origin's state is unknown, so nothing "+
			"was classified; check the network and origin's credentials and run it again", mergeOrderLsRemoteCommand, where, timeout, err)
	case err != nil:
		return mergeRemote{}, fmt.Errorf("org: merge-order: %s in %s failed (%w): origin's state is unknown, so nothing was "+
			"classified; merge-order reads origin, so run it in a repository whose origin answers", mergeOrderLsRemoteCommand, where, err)
	}
	r := mergeRemote{heads: make(map[string]string)}
	symref := parseLsRemote(out, r.heads)
	branch, isBranch := strings.CutPrefix(symref, "refs/heads/")
	switch {
	case symref == "":
		return mergeRemote{}, fmt.Errorf("org: merge-order: %s in %s printed no HEAD symref, so origin's default branch is unknown "+
			"and nothing was classified: point origin's HEAD at the default branch and run it again", mergeOrderLsRemoteCommand, where)
	case !isBranch || branch == "":
		return mergeRemote{}, fmt.Errorf("org: merge-order: origin's HEAD names %s, which is not a branch, so origin's default branch "+
			"is unknown and nothing was classified: point origin's HEAD at the default branch and run it again", symref)
	}
	tip, listed := r.heads[branch]
	if !listed {
		return mergeRemote{}, fmt.Errorf("org: merge-order: origin's HEAD names %s, which %s did not list, so its tip is unknown "+
			"and nothing was classified: point origin's HEAD at an existing branch and run it again", symref, mergeOrderLsRemoteCommand)
	}
	r.defaultBranch, r.defaultHead = branch, tip
	return r, nil
}

// parseLsRemote reads the ls-remote's output lines (`<value>\t<ref>`): the
// tip of each refs/heads/<name> into heads, and returns the target of the
// first `ref: <target>\tHEAD` line, "" when there is none. Only a ref named
// exactly HEAD is origin's HEAD: the pattern HEAD also matches refs such as
// refs/heads/HEAD, which is read as a branch called HEAD. A refs/heads/ line
// whose value is not a commit SHA (the symref line of a symbolic branch) is
// skipped.
func parseLsRemote(out string, heads map[string]string) (symref string) {
	for line := range strings.SplitSeq(out, "\n") {
		value, ref, ok := strings.Cut(strings.TrimSuffix(line, "\r"), "\t")
		if !ok {
			continue
		}
		if ref == "HEAD" {
			if target, isSymref := strings.CutPrefix(value, "ref: "); isSymref && symref == "" {
				symref = target
			}
			continue
		}
		if branch, isBranch := strings.CutPrefix(ref, "refs/heads/"); isBranch && branch != "" && gitObjectPattern.MatchString(value) {
			heads[branch] = value
		}
	}
	return symref
}

// ghPullRequest is one element of `gh pr list --json` output.
type ghPullRequest struct {
	Number            int    `json:"number"`
	State             string `json:"state"`
	URL               string `json:"url"`
	HeadRefOid        string `json:"headRefOid"`
	BaseRefName       string `json:"baseRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
}

// lookupMergePRs asks gh for each target's PR, in target order, and sets
// target.pr (C5). available is false when gh is not installed: it is not run
// again for the other branches, and the note says what is not told apart. A
// gh that fails for one branch (an exit, a timeout, output that is not a PR
// list) leaves that branch's PR nil with a note naming it. The error is for
// ctx done.
func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec, targets []*mergeTarget) (available bool, notes []string, _ error) {
	timeout := p.GHTimeout
	if timeout <= 0 {
		timeout = defaultMergeOrderGHTimeout
	}
	for _, t := range targets {
		pr, err := lookupMergePR(ctx, p.Dir, run, timeout, t.branch)
		switch {
		case ctx.Err() != nil:
			return false, nil, mergeOrderInterrupted(ctx, "gh ran")
		case errors.Is(err, exec.ErrNotFound):
			return false, append(notes, "gh was not found, so no PR was read: merged and closed PRs are not told apart (a branch "+
				"deleted after its merge shows as no_remote) and PR bases are not checked"), nil
		case err != nil:
			notes = append(notes, fmt.Sprintf("gh pr list --head %s failed (%v): its PR is unknown, so merged, closed and the PR "+
				"base are not told apart for it", t.branch, err))
		default:
			t.pr = pr
		}
	}
	return true, notes, nil
}

// lookupMergePR runs `gh pr list --head <branch> --state all --json ...` in
// dir with GH_PROMPT_DISABLED=1 and timeout, and returns the PR
// pickMergePR picks from its output, nil when there is none.
func lookupMergePR(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, branch string) (*MergeOrderPR, error) {
	ghCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := run(ghCtx, dir, []string{"GH_PROMPT_DISABLED=1"}, "gh",
		"pr", "list", "--head", branch, "--state", "all", "--json", mergeOrderGHFields)
	switch {
	case err != nil && ghCtx.Err() != nil && ctx.Err() == nil:
		return nil, fmt.Errorf("timed out after %s: %w", timeout, err)
	case err != nil:
		return nil, err
	}
	var prs []ghPullRequest
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, fmt.Errorf("its output is not a JSON list of PRs: %w", err)
	}
	return pickMergePR(prs), nil
}

// pickMergePR returns the PR with the highest number among prs that are not
// from a fork (isCrossRepository false), so a fork's branch of the same name
// is never read as the feature's PR; nil when there is none.
func pickMergePR(prs []ghPullRequest) *MergeOrderPR {
	var best *ghPullRequest
	for i := range prs {
		if pr := &prs[i]; !pr.IsCrossRepository && (best == nil || pr.Number > best.Number) {
			best = pr
		}
	}
	if best == nil {
		return nil
	}
	return &MergeOrderPR{Number: best.Number, State: best.State, URL: best.URL, HeadRefOid: best.HeadRefOid, BaseRefName: best.BaseRefName}
}

// orderMergeEntries classifies every target against remote and returns the
// entries in mergeClassOrder, each class by pos (oldest first), ranked from
// 1. Without all, merged and closed entries are left out and counted in
// hidden.
func orderMergeEntries(targets []*mergeTarget, remote mergeRemote, all bool) (entries []MergeOrderEntry, hidden int) {
	type ordered struct {
		entry MergeOrderEntry
		class int
		pos   int
	}
	var list []ordered
	for _, t := range targets {
		e := mergeEntryFor(t, remote)
		if !all && (e.Class == MergeClassMerged || e.Class == MergeClassClosed) {
			hidden++
			continue
		}
		list = append(list, ordered{entry: e, class: slices.Index(mergeClassOrder, e.Class), pos: t.pos()})
	}
	slices.SortStableFunc(list, func(a, b ordered) int {
		return cmp.Or(cmp.Compare(a.class, b.class), cmp.Compare(a.pos, b.pos))
	})
	entries = make([]MergeOrderEntry, 0, len(list))
	for i, o := range list {
		o.entry.Rank = i + 1
		entries = append(entries, o.entry)
	}
	return entries, hidden
}

// mergeEntryFor is t's entry, Rank unset.
func mergeEntryFor(t *mergeTarget, remote mergeRemote) MergeOrderEntry {
	class, detail := classifyMergeTarget(t, remote)
	e := MergeOrderEntry{Class: class, Branch: t.branch, RemoteHead: remote.heads[t.branch], Detail: detail, PR: t.pr, Binding: t.binding}
	if g := t.gate; g != nil {
		e.Gate = &MergeOrderGate{
			Event: g.ev.Event, Readable: g.readable, Head: g.record.Head, Base: g.record.Base,
			BaseRef: g.record.BaseRef, Reason: g.record.Reason, TS: g.ev.TS,
		}
	}
	return e
}

// classifyMergeTarget returns t's class and its Detail: the first of these
// that applies (the plan's Scope, C4, C7, C8): merged or closed (the PR's
// state), unreadable, no_gate, failed, no_remote, regate, wrong_base (only
// with a PR), rebase, ready.
func classifyMergeTarget(t *mergeTarget, remote mergeRemote) (class, detail string) {
	if t.pr != nil {
		switch strings.ToUpper(t.pr.State) {
		case "MERGED":
			return MergeClassMerged, fmt.Sprintf("PR #%d is merged", t.pr.Number)
		case "CLOSED":
			return MergeClassClosed, fmt.Sprintf("PR #%d was closed without a merge", t.pr.Number)
		}
	}
	g := t.gate
	switch {
	case g != nil && !g.readable:
		return MergeClassUnreadable, fmt.Sprintf("the latest gate record (%s at %s) cannot be read: run ralph org gate again "+
			"in the feature worktree (the gate runs only while the feature org runs)", g.ev.Event, g.ev.TS)
	case g != nil && t.corruptAfterGate:
		return MergeClassUnreadable, fmt.Sprintf("a corrupt manifest line follows the latest gate record (%s at %s) and may "+
			"be a newer one: run ralph org gate again in the feature worktree (the gate runs only while the feature org runs)",
			g.ev.Event, g.ev.TS)
	case g == nil:
		return MergeClassNoGate, "no gate record: run ralph org gate in the feature worktree before the PR is merged"
	case g.record.Reason != "":
		return MergeClassFailed, fmt.Sprintf("the gate failed (%s): fix it and run ralph org gate again", g.record.Reason)
	}
	return classifyGatePassed(t, g.record, remote)
}

// classifyGatePassed is classifyMergeTarget for a branch whose latest gate
// record r is a readable gate_passed: no_remote, regate, wrong_base, rebase
// or ready. A base_ref that is neither refs/remotes/origin/<name> nor
// refs/heads/<name> matches no default branch, so it is rebase.
func classifyGatePassed(t *mergeTarget, r gateRecord, remote mergeRemote) (class, detail string) {
	def := remote.defaultBranch
	tip, onOrigin := remote.heads[t.branch]
	switch {
	case !onOrigin:
		return MergeClassNoRemote, fmt.Sprintf("origin has no branch %s: it was not pushed, or was deleted after a merge", t.branch)
	case tip != r.Head:
		return MergeClassRegate, fmt.Sprintf("pushed after the gate: origin has %s, the gate checked %s: run ralph org gate again "+
			"(the gate runs only while the feature org runs)", shortMergeSHA(tip), shortMergeSHA(r.Head))
	case t.pr != nil && t.pr.BaseRefName != def:
		return MergeClassWrongBase, fmt.Sprintf("PR #%d merges into %s, not origin's default branch %s: change the PR's base to %s",
			t.pr.Number, t.pr.BaseRefName, def, def)
	}
	name, named := gateBaseBranchName(r.BaseRef)
	switch {
	case !named || name != def:
		return MergeClassRebase, fmt.Sprintf("the gate checked against %s, but origin's default branch is now %s: rebase on "+
			"origin/%s and run ralph org gate again (the gate runs only while the feature org runs)", r.BaseRef, def, def)
	case r.Base != remote.defaultHead:
		return MergeClassRebase, fmt.Sprintf("origin/%s moved: gate base %s, now %s: rebase on origin/%s and run ralph org gate again "+
			"(the gate runs only while the feature org runs)", def, shortMergeSHA(r.Base), shortMergeSHA(remote.defaultHead), def)
	}
	return MergeClassReady, fmt.Sprintf("the gate passed on %s, origin's tip of the branch, with base %s, origin/%s's tip",
		shortMergeSHA(r.Head), shortMergeSHA(r.Base), def)
}

// gateBaseBranchName returns the branch a gate record's base_ref names:
// refs/remotes/origin/<name> or refs/heads/<name> (resolveGateBase). ok is
// false for any other ref.
func gateBaseBranchName(baseRef string) (name string, ok bool) {
	for _, prefix := range []string{"refs/remotes/origin/", "refs/heads/"} {
		if name, ok := strings.CutPrefix(baseRef, prefix); ok && name != "" {
			return name, true
		}
	}
	return "", false
}

// shortMergeSHA is sha's first 12 characters, the length merge-order shows.
func shortMergeSHA(sha string) string {
	return sha[:min(len(sha), 12)]
}

// mergeOrderListNotes are the notes about the list res holds: the merged
// or closed branches left out (hidden), what merging one ready branch does
// to the others, and that the order is only as of the ls-remote.
func mergeOrderListNotes(res MergeOrderResult, hidden int) []string {
	var notes []string
	if hidden > 0 {
		notes = append(notes, fmt.Sprintf("%d branch(es) whose PR is merged or closed are not shown (--all shows them)", hidden))
	}
	ready := 0
	for _, e := range res.Entries {
		if e.Class == MergeClassReady {
			ready++
		}
	}
	if ready >= 2 {
		notes = append(notes, fmt.Sprintf("%d branches are ready, but merging one moves origin/%s: the other ready ones then "+
			"become rebase (rebase them on origin/%s and run ralph org gate again; the gate runs only while the feature org "+
			"runs)", ready, res.DefaultBranch, res.DefaultBranch))
	}
	return append(notes, fmt.Sprintf("the order is as of the ls-remote at %s and is not a guarantee: origin/%s can move "+
		"before a merge", res.ListedAt, res.DefaultBranch))
}

// mergeOrderInterrupted is MergeOrder's error when ctx is done while step.
func mergeOrderInterrupted(ctx context.Context, step string) error {
	return fmt.Errorf("org: merge-order: interrupted (%v) while %s: nothing was classified", context.Cause(ctx), step)
}

// realMergeOrderExec is MergeOrderExec's real implementation:
// exec.CommandContext in dir with os.Environ() and env, stdout returned and
// stderr, trimmed, added to the error. WaitDelay closes the pipes a child
// still holds after the kill (or after an exit), so a timeout cannot be held
// up by them; an exit 0 whose pipe a child held is a success, the way
// realGitStatus reads it.
func realMergeOrderExec(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.WaitDelay = mergeOrderWaitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil {
		err = nil
	}
	if err != nil {
		if s := strings.TrimSpace(stderr.String()); s != "" {
			return stdout.String(), fmt.Errorf("%w: %s", err, s)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}
