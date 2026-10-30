package org

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// The `ralph org merge-order` verb (MergeOrder), stage 7c-1 of plan
// docs/plans/active/2026-10-14-org-merge-order.md (FR-8 of
// docs/specs/2026-10-07-org-multi-org-director.md). It lists the feature
// branches of the ledger in the order their PRs may be merged now. A branch is
// ready when its latest gate record (gate_record.go's gateRecord) is a
// gate_passed (or a gate_ci_passed, a regate's record of its PR's checks)
// whose head is the branch's tip on origin and whose base is the
// tip of origin's default branch now. The gate records are the only evidence
// it reads: a reviewer's or the leader's word does not count.
//
// The targets are branches (C3): the branch of every complete feature binding
// of the manifest, disbanded orgs included, and every branch a readable gate
// record names. origin's state comes from one `git ls-remote --symref origin
// HEAD 'refs/heads/*'` (C2, C7), each branch's PR from `gh pr list` when gh is
// there (C5). gh picks the repository it reads itself (`gh repo set-default`,
// else a remote by its name, upstream before origin), so a PR is used only
// when its url names origin's owner/repo, which one `git remote get-url
// origin` reads the first time gh returns a PR. The check has three gaps
// (mergeOriginRepo.check): an origin URL with no owner/repo (a local path,
// file://, a scheme it does not read) skips it, hosts are not compared, and
// it runs only on a PR gh returns, so another repository with no PR for the
// branch gives no note. All three commands run through MergeOrderExec (C6).
// Evidence that cannot be read never makes a branch ready (C8).
//
// The result also carries the merge guarantee of origin's default branch
// (stage 7c-2, docs/plans/active/2026-10-14-org-merge-guarantee.md): the
// cache `ralph doctor` writes in the ledger's state dir, read with
// ReadMergeGuarantee (mergeguarantee.go), never with gh.
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
	// MergeClassFailed: the branch's latest gate record is a gate_failed or
	// a gate_ci_failed.
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

// The values of MergeOrderEntry.PRLookup: what gh's PR lookup gave for the
// branch (lookupMergePRs).
const (
	// MergeOrderPRLookupFound: gh returned a PR for the branch that is used,
	// one from origin's repository or, when origin's URL names none (a local
	// path), the one gh returned as it is. PR is set.
	MergeOrderPRLookupFound = "found"
	// MergeOrderPRLookupNone: gh returned no PR for the branch, or only PRs
	// from forks, fewer than mergeOrderGHLimit.
	MergeOrderPRLookupNone = "none"
	// MergeOrderPRLookupFailed: gh failed for the branch (an exit, a timeout,
	// output that is not a PR list); a note says why.
	MergeOrderPRLookupFailed = "failed"
	// MergeOrderPRLookupLimitForks: gh returned mergeOrderGHLimit PRs for the
	// branch, all from forks, so the branch's own PR may be past the limit; a
	// note says so.
	MergeOrderPRLookupLimitForks = "limit_forks"
	// MergeOrderPRLookupSkipped: no answer of gh is used for any branch, and
	// a note says why: gh was not found, or gh read another repository than
	// origin's or a PR's repository could not be checked against origin's, so
	// every PR was left out. The branches gh was not asked about then are
	// skipped too.
	MergeOrderPRLookupSkipped = "skipped"
)

// mergeClassOrder is the order MergeOrder lists the classes in.
var mergeClassOrder = []string{
	MergeClassReady, MergeClassRebase, MergeClassWrongBase, MergeClassRegate, MergeClassUnreadable,
	MergeClassFailed, MergeClassNoRemote, MergeClassNoGate, MergeClassMerged, MergeClassClosed,
}

const (
	// defaultMergeOrderLsRemoteTimeout and defaultMergeOrderGHTimeout bound the
	// ls-remote, and each gh call and the get-url, when MergeOrderParams
	// leaves them zero.
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
	// mergeOrderGHLimit is the --limit MergeOrder passes to gh pr list, whose
	// default is 30. With --head alone gh reads the PRs of that head branch
	// newest first, 100 per page, up to the limit, so 30 newer fork PRs of the
	// same branch name would push the branch's own PR out of the result
	// before pickMergePR leaves the forks out. A branch normally has one PR,
	// one page, so the higher limit costs nothing more.
	mergeOrderGHLimit = 1000
	// mergeOrderNoPRs ends every note that says no PR is used: what is then
	// not told apart.
	mergeOrderNoPRs = "merged and closed PRs are not told apart (a branch deleted after its merge shows as no_remote) and PR " +
		"bases are not checked"
	// mergeOrderMaxNamed is how many branches a note about gh names
	// (mergeNamedBranches).
	mergeOrderMaxNamed = 5
	// mergeGateRunsWhileOrgRuns ends every Detail and note that says to run
	// ralph org gate.
	mergeGateRunsWhileOrgRuns = "(the gate runs only while the feature org runs)"
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
	// LsRemoteTimeout bounds the ls-remote; GHTimeout bounds each gh call and
	// the `git remote get-url origin` lookupMergePRs runs with them. Zero or
	// less means 30 s and 15 s.
	LsRemoteTimeout, GHTimeout time.Duration
	// Origin, when it names a repository (ReadMergeOrigin), pins gh's PR
	// lookup to it, as ralph org merge pins its own; ralph org cleanup passes
	// it. gh pr list then gets -R <Origin.Arg()>, a PR whose url names another
	// host or owner/repo than Origin's leaves every PR out, as a PR of another
	// repository does without it, and lookupMergePRs runs no `git remote
	// get-url origin`. The zero MergeOrigin leaves the repository to gh and
	// checks the owner/repo of its PRs against origin's URL (mergeOriginRepo).
	Origin MergeOrigin
	// Followups also reports the ledger's open follow-ups, after the entries
	// (MergeOrderResult.Followups, mergeorder_followup.go): ralph org
	// merge-order sets it, ralph org cleanup does not.
	Followups bool
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
	// MergeGuarantee is the merge guarantee of origin's default branch as the
	// ledger's cache tells it (ReadMergeGuarantee), unknown
	// (default_branch_changed) when the cache was checked for another branch
	// than DefaultBranch (ForDefaultBranch). With no target branch no
	// ls-remote runs, so it is the cache's alone.
	MergeGuarantee MergeGuarantee `json:"merge_guarantee"`
	// Entries are the branches shown, in merge order.
	Entries []MergeOrderEntry `json:"entries"`
	// Notes are what the reader should know about the whole list, one
	// sentence each.
	Notes []string `json:"notes"`
	// Followups are the ledger's open follow-ups in ID order, each with
	// whether gh reads its after-branch's PR as merged since it was recorded
	// (mergeOrderFollowups); empty without MergeOrderParams.Followups.
	Followups []MergeOrderFollowup `json:"followups"`
	// FollowupRecord is the follow-up record's path and damage, which the CLI
	// warns about on stderr; not in the JSON.
	FollowupRecord MergeOrderFollowupRecord `json:"-"`
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
	// PR is the branch's PR (pickMergePR), set only when PRLookup is
	// MergeOrderPRLookupFound.
	PR *MergeOrderPR `json:"pr,omitempty"`
	// PRLookup is what gh's PR lookup gave for the branch, one of
	// MergeOrderPRLookupFound, MergeOrderPRLookupNone,
	// MergeOrderPRLookupFailed, MergeOrderPRLookupLimitForks and
	// MergeOrderPRLookupSkipped: why PR is nil, when it is.
	PRLookup string `json:"pr_lookup"`
	// PRLookupWhy says why gh's answer for the branch was not used: with
	// PRLookup failed, gh's error, and with skipped, why no answer of gh was
	// used for any branch (gh was not found, or it returned a PR of another
	// repository than origin's); "" otherwise. It is not in the JSON, whose
	// notes say it: ralph org cleanup puts it in its lines.
	PRLookupWhy string `json:"-"`
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

// MergeOrderGate is a branch's latest gate record. Head, Base, BaseRef,
// Reason and Via are empty when it cannot be read.
type MergeOrderGate struct {
	Event    string `json:"event"` // gate_passed, gate_failed, gate_ci_passed or gate_ci_failed
	Readable bool   `json:"readable"`
	Head     string `json:"head"`
	Base     string `json:"base"`
	BaseRef  string `json:"base_ref"`
	Reason   string `json:"reason"` // GateReason*, "" on a pass
	// Via is "regate" for a record ralph org regate wrote ("regate_ci" for
	// one it wrote from the PR's checks), left out for one ralph org gate
	// wrote.
	Via string `json:"via,omitempty"`
	TS  string `json:"ts"`
}

// MergeOrderBinding names the feature binding a branch was started for.
type MergeOrderBinding struct {
	OrgID   string `json:"org_id"`
	Feature string `json:"feature"`
	Split   string `json:"split"`
	TS      string `json:"ts"`
	// BoundTo is the branch of OrgID's latest complete binding when that is
	// another branch (the org_id was started again for another feature
	// after this binding), whether the org runs now or not; "" when the
	// latest is this one. ralph org cleanup leaves such a branch out unless
	// --branch names it.
	BoundTo string `json:"bound_to,omitempty"`
}

// MergeOrder runs `ralph org merge-order`. It reads the manifest line by line
// (readMergeLedger) and collects the target branches with their latest
// binding and gate record (collectMergeTargets). It reads the merge guarantee
// from the cache in the manifest's state dir (ReadMergeGuarantee, with p.Dir,
// the same exec and o.Now), which runs no gh and, only when the cache is
// fresh, the local `git remote get-url origin`. With no target it returns at
// once, with a note, the guarantee, and no ls-remote or gh call. Otherwise it
// reads origin with one ls-remote in p.Dir (listOriginBranches), makes the
// guarantee unknown when the cache was checked for another default branch
// (ForDefaultBranch), asks gh for each branch's PR and keeps it only when it
// is from origin's repository (lookupMergePRs, which runs `git remote get-url
// origin` once, when gh first returns a PR, unless p.Origin pins the lookup
// to the repository it names), classifies each branch
// (classifyMergeTarget) and orders them (orderMergeEntries). With
// p.Followups, it then reports the open follow-ups, with or without a target
// (mergeOrderFollowups), which never makes it return an error.
//
// The error is for a manifest that cannot be read, an ls-remote that fails,
// times out or names no default branch, and ctx done: nothing is classified
// then. A gh that is missing, fails, reads another repository than origin's,
// or returns its limit of PRs all from forks only leaves PRs out, with a note,
// and each entry's PRLookup tells what gh gave for its branch.
func (o *Org) MergeOrder(ctx context.Context, p MergeOrderParams) (MergeOrderResult, error) {
	ledger, err := readMergeLedger(o.Manifest.Path())
	if err != nil {
		return MergeOrderResult{}, fmt.Errorf("org: merge-order: read manifest: %w", err)
	}
	targets, unattached := collectMergeTargets(ledger)
	ledgerNotes := mergeLedgerNotes(len(ledger.corruptAt), unattached)
	run := p.Exec
	if run == nil {
		run = realMergeOrderExec
	}
	guaranteeDir := filepath.Dir(o.Manifest.Path())
	guaranteeParams := MergeGuaranteeReadParams{Dir: p.Dir, Exec: run, Now: o.Now}
	res := MergeOrderResult{Entries: []MergeOrderEntry{}, Notes: []string{}, Followups: []MergeOrderFollowup{}}
	if len(targets) == 0 {
		res.MergeGuarantee = ReadMergeGuarantee(ctx, guaranteeDir, guaranteeParams)
		res.Notes = append(res.Notes, "no feature branches in the ledger: a branch is listed once ralph org start --plan "+
			"binds an org to it or ralph org gate records a gate on it")
		res.Notes = append(res.Notes, ledgerNotes...)
		o.mergeOrderFollowups(ctx, p, run, &res)
		return res, nil
	}
	res.ListedAt = o.now()
	remote, err := listOriginBranches(ctx, p, run)
	if err != nil {
		return MergeOrderResult{}, err
	}
	res.DefaultBranch, res.DefaultHead = remote.defaultBranch, remote.defaultHead
	res.MergeGuarantee = ReadMergeGuarantee(ctx, guaranteeDir, guaranteeParams).ForDefaultBranch(res.DefaultBranch)
	var ghNotes []string
	if res.GHAvailable, ghNotes, err = lookupMergePRs(ctx, p, run, targets); err != nil {
		return MergeOrderResult{}, err
	}
	entries, hidden := orderMergeEntries(targets, remote, p.All)
	res.Entries = entries
	res.Notes = append(append(res.Notes, ghNotes...), ledgerNotes...)
	res.Notes = append(res.Notes, mergeOrderListNotes(res, hidden, mergeRegateBranches(targets))...)
	o.mergeOrderFollowups(ctx, p, run, &res)
	return res, nil
}

// mergeRegateBranches is the set of targets' branches whose org is not
// running (mergeTarget.regates). A branch whose org runs another feature now
// is not in it, though its Details name ralph org regate too
// (mergeTarget.namesRegate).
func mergeRegateBranches(targets []*mergeTarget) map[string]bool {
	branches := make(map[string]bool)
	for _, t := range targets {
		if t.regates() {
			branches[t.branch] = true
		}
	}
	return branches
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
	// orgRunning: binding is set and its org is in RunningOrgs.
	orgRunning bool
	// orgRunsOn: orgRunning, and the org's latest complete binding is of
	// another branch (the org_id was started again for another feature):
	// that branch. "" otherwise.
	orgRunsOn string
	pr        *MergeOrderPR
	// prLookup is what gh's PR lookup gave for the branch
	// (MergeOrderPRLookup*), and prWhy why its answer was not used
	// (MergeOrderEntry.PRLookupWhy), both set by lookupMergePRs.
	prLookup, prWhy string
}

// regates reports whether t has a binding and its org is not running
// (disbanded), so ralph org gate refuses it and ralph org regate runs on it.
func (t *mergeTarget) regates() bool {
	return t.binding != nil && !t.orgRunning
}

// namesRegate reports whether the Details of t name ralph org regate instead
// of ralph org gate (mergeRegateNext): t.regates(), or t's org runs another
// feature now (orgRunsOn), so ralph org gate --org-id <org> would gate that
// feature's worktree, and ralph org regate runs on t once the org is
// disbanded.
func (t *mergeTarget) namesRegate() bool {
	return t.regates() || t.orgRunsOn != ""
}

// mergeGateRecord is one real gate event attached to a branch: record is
// gateRecord{} when it cannot be read.
type mergeGateRecord struct {
	pos      int
	ev       ManifestEvent
	record   gateRecord
	readable bool
	// orgBranchCount is how many branches one that cannot be read is
	// attached to: every branch its org had before it (collectMergeTargets).
	orgBranchCount int
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
// A gate record (a gate_passed or gate_failed, or a gate_ci_passed or
// gate_ci_failed, read as a pass and a failure the same way: plan
// docs/plans/active/2026-10-29-regate-ci-checks.md, D7) is readable when
// gateRecordFromDetails reads it and its event name agrees with its reason
// and its via (gateRecord.event); it attaches to its branch= branch, which it
// makes a target. One that is not attaches to every branch its org had a
// complete binding or a readable gate record for before it, so that none of
// them falls back to an older pass (C8): ralph org gate runs only on
// the branch of the org's latest binding, but ralph org regate records under
// the org of a branch's binding while the org_id may be bound to another
// branch later (plan docs/plans/active/2026-10-16-org-regate.md, D2), so the
// record may be any of them. With none it attaches nowhere and counts in
// unattached. A branch's gate is the latest record attached to it. A branch
// with a binding is orgRunning when its binding's org is in RunningOrgs of
// l's events, read once, and has orgRunsOn when that org's latest complete
// binding, read in the same walk, is of another branch. Its binding's
// BoundTo is that other branch too, whether the org runs or not.
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
	// org_id -> every branch it had a complete binding or a readable gate
	// record for so far, each once.
	orgBranches := make(map[string][]string)
	addOrgBranch := func(orgID, branch string) {
		if !slices.Contains(orgBranches[orgID], branch) {
			orgBranches[orgID] = append(orgBranches[orgID], branch)
		}
	}
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
			addOrgBranch(ev.OrgID, f.Branch)
		case EventGatePassed, EventGateFailed, EventGateCIPassed, EventGateCIFailed:
			g := &mergeGateRecord{pos: i, ev: ev}
			g.record, g.readable = gateRecordFromDetails(ev.Details)
			g.readable = g.readable && g.record.event() == ev.Event
			if g.readable {
				addOrgBranch(ev.OrgID, g.record.Branch)
				target(g.record.Branch).gate = g
				continue
			}
			g.record = gateRecord{}
			// orgBranches holds no "": a complete binding and a readable
			// record both name a branch.
			branches := orgBranches[ev.OrgID]
			if len(branches) == 0 {
				unattached++
				continue
			}
			g.orgBranchCount = len(branches)
			for _, branch := range branches {
				target(branch).gate = g
			}
		}
	}
	if n := len(l.corruptAt); n > 0 {
		lastCorrupt := l.corruptAt[n-1]
		for _, t := range targets {
			t.corruptAfterGate = t.gate != nil && t.gate.readable && lastCorrupt > t.gate.pos
		}
	}
	running := RunningOrgs(l.events)
	for _, t := range targets {
		t.orgRunning = t.binding != nil && slices.Contains(running, t.binding.OrgID)
		if t.orgRunning && boundBranch[t.binding.OrgID] != t.branch {
			t.orgRunsOn = boundBranch[t.binding.OrgID]
		}
		if t.binding != nil && boundBranch[t.binding.OrgID] != t.branch {
			t.binding.BoundTo = boundBranch[t.binding.OrgID]
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
			"record comes before one is unreadable until ralph org gate (while the branch's org runs) or ralph org regate "+
			"--branch <branch> (once that org is disbanded) records a new one", corrupt))
	}
	if unattached > 0 {
		notes = append(notes, fmt.Sprintf("%d gate record(s) cannot be read and their org had neither a feature binding nor a "+
			"readable gate record before them, so they are attached to no branch", unattached))
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
// target.pr (C5) when the PR is from origin's repository: with p.Origin, the
// one gh pr list reads with -R and the PR's url names (MergeOrigin.hasPR),
// else the one origin's URL names (mergeOriginRepo). target.prWhy says why
// gh's answer for the branch was not used (MergeOrderEntry.PRLookupWhy).
// It sets target.prLookup to what gh gave for the branch: every target starts
// as MergeOrderPRLookupSkipped, and each one gh answers for becomes found (its
// PR is used), none, failed or limit_forks. available is false when gh is not
// installed: it is not run again for the other branches, every target is
// skipped with a nil PR, and the note says what is not told apart. A gh that
// fails for a branch (an exit, a timeout, output that is not a PR list)
// leaves that branch's PR nil, with one note per reason naming the branches
// (mergeGHFailures). A branch gh returns mergeOrderGHLimit PRs for, all from
// forks, also keeps a nil PR, since its own PR may be past the limit, and one
// note names every such branch (mergeGHLimitNotes). When the check of a PR's
// repository fails (gh read another repository, the PR's url names none, or
// origin's URL cannot be read), every target's PR is left nil and every
// target is skipped, gh is not run again, and one note says why. The error is
// for ctx done, and no target is classified then.
func lookupMergePRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec, targets []*mergeTarget) (available bool, notes []string, _ error) {
	timeout := p.GHTimeout
	if timeout <= 0 {
		timeout = defaultMergeOrderGHTimeout
	}
	skipAll := func(why string) {
		for _, t := range targets {
			t.pr, t.prLookup, t.prWhy = nil, MergeOrderPRLookupSkipped, why
		}
	}
	skipAll("")
	var failed mergeGHFailures
	var limitFull []string // the branches gh returned mergeOrderGHLimit fork PRs for
	withNotes := func(more ...string) []string {
		return append(append(failed.notes(), mergeGHLimitNotes(limitFull)...), more...)
	}
	repoArg := p.Origin.Arg()
	var origin mergeOriginRepo
	for _, t := range targets {
		pr, limitFullOfForks, err := lookupMergePR(ctx, p.Dir, run, timeout, repoArg, t.branch)
		switch {
		case ctx.Err() != nil:
			return false, nil, mergeOrderInterrupted(ctx, "gh ran")
		case errors.Is(err, exec.ErrNotFound):
			skipAll("gh was not found")
			return false, withNotes("gh was not found, so no PR was read: " + mergeOrderNoPRs), nil
		case err != nil:
			failed.add(err.Error(), t.branch)
			t.prLookup, t.prWhy = MergeOrderPRLookupFailed, err.Error()
			continue
		case limitFullOfForks:
			limitFull = append(limitFull, t.branch)
			t.prLookup = MergeOrderPRLookupLimitForks
			continue
		case pr == nil:
			t.prLookup = MergeOrderPRLookupNone
			continue
		}
		if repoArg != "" {
			if why := p.Origin.foreignPRWhy(t.branch, pr); why != "" {
				skipAll(why)
				return true, withNotes(why + ": no PR is used, so " + mergeOrderNoPRs), nil
			}
		} else {
			stop, err := origin.check(ctx, p.Dir, run, timeout, t.branch, pr)
			if err != nil {
				return false, nil, err
			}
			if stop != "" {
				skipAll(stop)
				return true, withNotes(stop), nil
			}
		}
		t.pr, t.prLookup = pr, MergeOrderPRLookupFound
	}
	return true, withNotes(), nil
}

// mergeGHFailures are the branches gh failed for, grouped by the error's
// text in the order each first came.
type mergeGHFailures []mergeGHFailure

type mergeGHFailure struct {
	reason   string
	branches []string
}

func (f *mergeGHFailures) add(reason, branch string) {
	for i := range *f {
		if (*f)[i].reason == reason {
			(*f)[i].branches = append((*f)[i].branches, branch)
			return
		}
	}
	*f = append(*f, mergeGHFailure{reason: reason, branches: []string{branch}})
}

// notes is one note per reason: for one branch it names the branch, for
// several it gives their count and names the first mergeOrderMaxNamed.
func (f mergeGHFailures) notes() []string {
	var notes []string
	for _, g := range f {
		if len(g.branches) == 1 {
			notes = append(notes, fmt.Sprintf("gh pr list --head %s failed (%s): its PR is unknown, so merged, closed and the PR "+
				"base are not told apart for it", g.branches[0], g.reason))
			continue
		}
		notes = append(notes, fmt.Sprintf("gh pr list failed for %d branches (%s) with the same error (%s): their PRs are "+
			"unknown, so merged, closed and the PR base are not told apart for them", len(g.branches), mergeNamedBranches(g.branches), g.reason))
	}
	return notes
}

// mergeGHLimitNotes is the note about the branches gh returned
// mergeOrderGHLimit PRs for, all from forks: for one it names the branch, for
// several it gives their count and names the first mergeOrderMaxNamed. It is
// empty when there are none.
func mergeGHLimitNotes(branches []string) []string {
	switch len(branches) {
	case 0:
		return nil
	case 1:
		return []string{fmt.Sprintf("gh returned %d PRs for %s, its limit, all from forks, and the branch's own PR may be "+
			"past them: no PR is used for it, so %s", mergeOrderGHLimit, branches[0], mergeOrderNoPRs)}
	}
	return []string{fmt.Sprintf("gh returned %d PRs, its limit, for each of %d branches (%s), all from forks, and their own "+
		"PRs may be past them: no PR is used for them, so %s", mergeOrderGHLimit, len(branches), mergeNamedBranches(branches), mergeOrderNoPRs)}
}

// mergeNamedBranches names branches in a note: all of them when there are
// at most mergeOrderMaxNamed, else the first mergeOrderMaxNamed and how many
// more.
func mergeNamedBranches(branches []string) string {
	named := strings.Join(branches[:min(len(branches), mergeOrderMaxNamed)], ", ")
	if more := len(branches) - mergeOrderMaxNamed; more > 0 {
		named += fmt.Sprintf(" and %d more", more)
	}
	return named
}

// mergeRepo is a GitHub repository, owner/name.
type mergeRepo struct{ owner, name string }

func (r mergeRepo) String() string { return r.owner + "/" + r.name }

// same reports whether r and o are the same repository: GitHub compares
// owner and repository names without case.
func (r mergeRepo) same(o mergeRepo) bool {
	return strings.EqualFold(r.owner, o.owner) && strings.EqualFold(r.name, o.name)
}

// mergeOriginRepo is origin's repository while lookupMergePRs checks gh's
// PRs against it: read the first time check runs.
type mergeOriginRepo struct {
	read     bool
	readable bool // origin's URL names an owner and a repository
	repo     mergeRepo
}

// check returns "" when pr, the PR gh returned for branch, may be used: it
// is from origin's repository, or origin's URL names none (a local path,
// which no PR can be from), so there is nothing to check it against.
// Otherwise it returns the note that leaves the PRs out: gh read another
// repository, pr's url names none, or `git remote get-url origin` failed.
// When gh read another repository, the note names two fixes, since check
// cannot tell it from origin's repository renamed on GitHub (git still
// reaches it at origin's old URL through GitHub's redirect, and gh's PR urls
// carry the new name): point origin at the new name with git remote set-url
// origin for a rename, else run gh repo set-default. Hosts are not compared,
// since origin's may be an ssh alias. The error is for ctx done.
func (o *mergeOriginRepo) check(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, branch string, pr *MergeOrderPR) (stop string, _ error) {
	if !o.read {
		o.read = true
		repo, readable, err := readMergeOriginRepo(ctx, dir, run, timeout)
		switch {
		case ctx.Err() != nil:
			return "", mergeOrderInterrupted(ctx, "git remote get-url origin ran")
		case err != nil:
			return fmt.Sprintf("git remote get-url origin failed (%v), so the repository gh read cannot be checked against "+
				"origin's: no PR is used, so %s", err, mergeOrderNoPRs), nil
		}
		o.repo, o.readable = repo, readable
	}
	if !o.readable {
		return "", nil
	}
	prRepo, ok := mergeRepoFromPRURL(pr.URL)
	switch {
	case !ok:
		return fmt.Sprintf("gh returned PR #%d for %s with the url %q, which names no repository to check against origin's "+
			"%s: no PR is used, so %s", pr.Number, branch, pr.URL, o.repo, mergeOrderNoPRs), nil
	case !prRepo.same(o.repo):
		return fmt.Sprintf("gh reads %s, not origin's %s: no PR is used, so %s; if %s was renamed to %s, point origin at it "+
			"(git remote set-url origin <its URL>); otherwise run gh repo set-default %s in this repository",
			prRepo, o.repo, mergeOrderNoPRs, o.repo, prRepo, o.repo), nil
	}
	return "", nil
}

// readMergeOriginRepo runs `git remote get-url origin` in dir with
// GIT_TERMINAL_PROMPT=0 and timeout, and reads origin's repository from the
// URL on the first line it prints (mergeRepoFromOriginURL); readable is false
// when the URL names none. The error is for a command that failed or timed
// out.
func readMergeOriginRepo(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration) (repo mergeRepo, readable bool, _ error) {
	getCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := run(getCtx, dir, []string{"GIT_TERMINAL_PROMPT=0"}, "git", "remote", "get-url", "origin")
	switch {
	case err != nil && getCtx.Err() != nil && ctx.Err() == nil:
		return mergeRepo{}, false, fmt.Errorf("timed out after %s: %w", timeout, err)
	case err != nil:
		return mergeRepo{}, false, err
	}
	line, _, _ := strings.Cut(out, "\n")
	repo, readable = mergeRepoFromOriginURL(strings.TrimSpace(line))
	return repo, readable, nil
}

// mergeOriginURLSchemes are the URL schemes whose host and path
// mergeOriginFromURL reads.
var mergeOriginURLSchemes = []string{"https", "http", "ssh", "git"}

// mergeRepoFromOriginURL reads the owner and repository from origin's URL
// (mergeOriginFromURL).
func mergeRepoFromOriginURL(s string) (repo mergeRepo, ok bool) {
	_, repo, ok = mergeOriginFromURL(s)
	return repo, ok
}

// mergeOriginFromURL reads the host, and the owner and repository, from
// origin's URL, only from one with a host: `scheme://[user@]host[:port]/<path>`
// for the schemes in mergeOriginURLSchemes, or the scp-like
// `[user@]host:<path>` (a colon with no slash before it, as git reads it).
// host is the host as written, lowercased, without the user and the port. A
// trailing "/", then a trailing ".git", is cut from the path, whose last two
// components are the owner and the repository. ok is false for anything
// else: a local path (absolute, relative, or file://), another scheme, or a
// path with fewer than two components.
func mergeOriginFromURL(s string) (host string, repo mergeRepo, ok bool) {
	var path string
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Hostname() == "" || !slices.Contains(mergeOriginURLSchemes, u.Scheme) {
			return "", mergeRepo{}, false
		}
		host, path = u.Hostname(), u.Path
	} else {
		var scp bool
		host, path, scp = strings.Cut(s, ":")
		if !scp || strings.Contains(host, "/") {
			return "", mergeRepo{}, false
		}
		if at := strings.LastIndexByte(host, '@'); at >= 0 {
			host = host[at+1:]
		}
		if host == "" {
			return "", mergeRepo{}, false
		}
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git"), "/")
	if len(parts) < 2 {
		return "", mergeRepo{}, false
	}
	repo = mergeRepo{owner: parts[len(parts)-2], name: parts[len(parts)-1]}
	if repo.owner == "" || repo.name == "" {
		return "", mergeRepo{}, false
	}
	return strings.ToLower(host), repo, true
}

// mergeRepoFromPRURL reads the owner and repository from a PR's url
// (mergeHostRepoFromPRURL).
func mergeRepoFromPRURL(s string) (repo mergeRepo, ok bool) {
	_, repo, ok = mergeHostRepoFromPRURL(s)
	return repo, ok
}

// mergeHostRepoFromPRURL reads the host, lowercased and without the port,
// and the owner and repository from a PR's url,
// `https://<host>/<owner>/<repo>/pull/<number>`; ok is false for any other
// shape.
func mergeHostRepoFromPRURL(s string) (host string, repo mergeRepo, ok bool) {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return "", mergeRepo{}, false
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != "pull" {
		return "", mergeRepo{}, false
	}
	if n, err := strconv.Atoi(parts[3]); err != nil || n <= 0 {
		return "", mergeRepo{}, false
	}
	return strings.ToLower(u.Hostname()), mergeRepo{owner: parts[0], name: parts[1]}, true
}

// lookupMergePR runs `gh pr list --head <branch> --state all --limit
// <mergeOrderGHLimit> --json ...` in dir with GH_PROMPT_DISABLED=1 and
// timeout, and returns the PR pickMergePR picks from its output, nil when
// there is none. When repo ([HOST/]OWNER/REPO) is not empty, `-R <repo>` is
// added after the other arguments, so gh reads that repository instead of the
// one it picks itself; merge-order passes MergeOrderParams.Origin.Arg(), ""
// unless the caller pinned the lookup to origin. limitFullOfForks reports that
// there is none and gh returned mergeOrderGHLimit PRs, all from forks, so the
// branch's own PR may be past the limit.
func lookupMergePR(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, repo, branch string) (pr *MergeOrderPR, limitFullOfForks bool, _ error) {
	ghCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := []string{"pr", "list", "--head", branch, "--state", "all", "--limit", strconv.Itoa(mergeOrderGHLimit), "--json", mergeOrderGHFields}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	out, err := run(ghCtx, dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", args...)
	switch {
	case err != nil && ghCtx.Err() != nil && ctx.Err() == nil:
		return nil, false, fmt.Errorf("timed out after %s: %w", timeout, err)
	case err != nil:
		return nil, false, err
	}
	var prs []ghPullRequest
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, false, fmt.Errorf("its output is not a JSON list of PRs: %w", err)
	}
	pr = pickMergePR(prs)
	return pr, pr == nil && len(prs) >= mergeOrderGHLimit, nil
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
	e := MergeOrderEntry{
		Class: class, Branch: t.branch, RemoteHead: remote.heads[t.branch], Detail: detail, PR: t.pr, PRLookup: t.prLookup,
		PRLookupWhy: t.prWhy, Binding: t.binding,
	}
	if g := t.gate; g != nil {
		e.Gate = &MergeOrderGate{
			Event: g.ev.Event, Readable: g.readable, Head: g.record.Head, Base: g.record.Base,
			BaseRef: g.record.BaseRef, Reason: g.record.Reason, Via: g.record.Via, TS: g.ev.TS,
		}
	}
	return e
}

// classifyMergeTarget returns t's class and its Detail: the first of these
// that applies (the plan's Scope, C4, C7, C8): merged or closed (the PR's
// state), unreadable, no_gate, failed, no_remote, regate, wrong_base (only
// with a PR), rebase, ready. A Detail that tells the reader to gate the
// branch names ralph org gate (mergeGateCommand) and ends with
// mergeGateRunsWhileOrgRuns, or, for a branch whose org is not running or
// runs another feature now (mergeTarget.namesRegate), names ralph org regate
// (mergeRegateNext), which runs from anywhere (plan
// docs/plans/active/2026-10-16-org-regate.md, AC10).
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
	gate := mergeGateCommand(t)
	switch {
	case g != nil && !g.readable:
		why := fmt.Sprintf("the latest gate record (%s at %s) cannot be read", g.ev.Event, g.ev.TS)
		if g.orgBranchCount > 1 {
			why += fmt.Sprintf(" and may be a record of another branch of the same org, %s, which has had %d branches", g.ev.OrgID,
				g.orgBranchCount)
		}
		if t.namesRegate() {
			return MergeClassUnreadable, why + ": " + mergeRegateNext(t, "", " to record a new one")
		}
		return MergeClassUnreadable, fmt.Sprintf("%s: run %s again in the feature worktree %s", why, gate, mergeGateRunsWhileOrgRuns)
	case g != nil && t.corruptAfterGate:
		why := fmt.Sprintf("a corrupt manifest line follows the latest gate record (%s at %s) and may be a newer one", g.ev.Event, g.ev.TS)
		if t.namesRegate() {
			return MergeClassUnreadable, why + ": " + mergeRegateNext(t, "", " to record a new one")
		}
		return MergeClassUnreadable, fmt.Sprintf("%s: run %s again in the feature worktree %s", why, gate, mergeGateRunsWhileOrgRuns)
	case g == nil && t.namesRegate():
		return MergeClassNoGate, "no gate record: " + mergeRegateNext(t, "", " before the PR is merged")
	case g == nil:
		return MergeClassNoGate, fmt.Sprintf("no gate record: run %s in the feature worktree before the PR is merged %s",
			gate, mergeGateRunsWhileOrgRuns)
	case g.record.Reason != "" && t.namesRegate():
		return MergeClassFailed, mergeRegateFailedDetail(t, g.record, remote)
	case g.record.Reason != "":
		return MergeClassFailed, fmt.Sprintf("the gate failed (%s): fix it and run %s again %s", g.record.Reason, gate,
			mergeGateRunsWhileOrgRuns)
	}
	return classifyGatePassed(t, g.record, remote)
}

// mergeRegateCommand is the ralph org regate command a Detail tells the
// reader to run for t, a branch whose Details name it
// (mergeTarget.namesRegate).
func mergeRegateCommand(t *mergeTarget) string {
	return "ralph org regate --branch " + t.branch
}

// mergeRegateNext is how a Detail of t, a branch whose Details name ralph org
// regate (mergeTarget.namesRegate), tells the reader what to do: steps, then
// "run" and mergeRegateCommand, then purpose. When t's org runs another
// feature now (mergeTarget.orgRunsOn), it first names that org and its
// branch, and says to run regate once the org is disbanded, since ralph org
// regate refuses a branch whose org runs.
func mergeRegateNext(t *mergeTarget, steps, purpose string) string {
	if t.orgRunsOn == "" {
		return steps + "run " + mergeRegateCommand(t) + purpose
	}
	return fmt.Sprintf("org_id %s now runs another feature, on the branch %s: %sonce %s is disbanded, run %s%s",
		t.binding.OrgID, t.orgRunsOn, steps, t.binding.OrgID, mergeRegateCommand(t), purpose)
}

// mergeRegateFailedDetail is the Detail of t, a branch whose Details name
// ralph org regate (mergeTarget.namesRegate), when its latest gate record r
// is a gate_failed. When ralph org regate wrote r (via=regate), it says on
// which head the gate ran and names r's log: origin's tip of the branch when
// that is r's head (regate gated it as it was), else the merge commit regate
// made of r's base into the branch, which regate undid after static_failed
// or test_failed and kept in the feature worktree after head_moved or
// tree_changed. A branch pushed after a gate-only regate failed on origin's
// tip reads as the second too: merge-order reads no commits, so it cannot
// tell the two apart. When regate wrote r from the PR's checks
// (via=regate_ci, a gate_ci_failed), it says that CI failed on r's head, the
// head regate pushed to origin's branch (or found there), names r's log,
// which holds the checks regate read, and says there is nothing to reset,
// since that head is on origin.
func mergeRegateFailedDetail(t *mergeTarget, r gateRecord, remote mergeRemote) string {
	next := mergeRegateNext(t, "fix it in the feature worktree, push it, and ", "")
	switch {
	case r.Via == gateViaRegateCI:
		return fmt.Sprintf("CI failed (%s) on %s, the head ralph org regate pushed to origin's branch or found there; the checks it "+
			"read are in %s, and with the head on origin there is nothing to reset: %s again", r.Reason, shortMergeSHA(r.Head), r.Log,
			next)
	case r.Via != gateViaRegate:
		return fmt.Sprintf("the gate failed (%s): %s", r.Reason, next)
	case remote.heads[t.branch] == r.Head:
		return fmt.Sprintf("the gate ralph org regate ran on origin's tip %s of the branch failed (%s); its log is %s: %s",
			shortMergeSHA(r.Head), r.Reason, r.Log, next)
	}
	kept := "undid"
	if r.Reason == GateReasonHeadMoved || r.Reason == GateReasonTreeChanged {
		kept = "kept in the feature worktree"
	}
	return fmt.Sprintf("the gate ralph org regate ran failed (%s) on %s, the merge commit it made of %s into the branch, which it "+
		"%s; its log is %s: %s", r.Reason, shortMergeSHA(r.Head), strings.TrimPrefix(r.BaseRef, "refs/remotes/"), kept, r.Log, next)
}

// mergeGateCommand is the ralph org gate command a Detail tells the reader
// to run for t, with the --org-id ralph org gate requires: the org of t's
// latest binding when it has one, else the org of its latest gate record (a
// branch only a gate record names). With neither org (collectMergeTargets
// attaches no record without an org_id), it is the bare command.
func mergeGateCommand(t *mergeTarget) string {
	switch {
	case t.binding != nil:
		return "ralph org gate --org-id " + t.binding.OrgID
	case t.gate != nil && t.gate.ev.OrgID != "":
		return "ralph org gate --org-id " + t.gate.ev.OrgID
	}
	return "ralph org gate"
}

// classifyGatePassed is classifyMergeTarget for a branch whose latest gate
// record r is a readable gate_passed or gate_ci_passed, read the same way
// whatever r's via: no_remote, regate, wrong_base, rebase
// or ready. A base_ref that is neither refs/remotes/origin/<name> nor
// refs/heads/<name> matches no default branch, so it is rebase. For a branch
// whose Details name ralph org regate (mergeTarget.namesRegate), regate also
// says that the gate's head may not have been pushed (a regate whose push
// failed leaves that), and rebase says that ralph org regate merges origin's
// default branch in, which it does instead of a rebase.
func classifyGatePassed(t *mergeTarget, r gateRecord, remote mergeRemote) (class, detail string) {
	def := remote.defaultBranch
	tip, onOrigin := remote.heads[t.branch]
	gate := mergeGateCommand(t)
	switch {
	case !onOrigin:
		return MergeClassNoRemote, fmt.Sprintf("origin has no branch %s: it was not pushed, or was deleted after a merge", t.branch)
	case tip != r.Head && t.namesRegate():
		return MergeClassRegate, fmt.Sprintf("pushed after the gate, or the gate's head was not pushed: origin has %s, the gate "+
			"checked %s: %s", shortMergeSHA(tip), shortMergeSHA(r.Head), mergeRegateNext(t, "", ""))
	case tip != r.Head:
		return MergeClassRegate, fmt.Sprintf("pushed after the gate: origin has %s, the gate checked %s: run %s again %s",
			shortMergeSHA(tip), shortMergeSHA(r.Head), gate, mergeGateRunsWhileOrgRuns)
	case t.pr != nil && t.pr.BaseRefName != def:
		return MergeClassWrongBase, fmt.Sprintf("PR #%d merges into %s, not origin's default branch %s: change the PR's base to %s",
			t.pr.Number, t.pr.BaseRefName, def, def)
	}
	name, named := gateBaseBranchName(r.BaseRef)
	var why string
	switch {
	case !named || name != def:
		why = fmt.Sprintf("the gate checked against %s, but origin's default branch is now %s", r.BaseRef, def)
	case r.Base != remote.defaultHead:
		why = fmt.Sprintf("origin/%s moved: gate base %s, now %s", def, shortMergeSHA(r.Base), shortMergeSHA(remote.defaultHead))
	}
	switch {
	case why != "" && t.namesRegate():
		return MergeClassRebase, why + ": " + mergeRegateNext(t, "", ", which merges origin/"+def+" in and runs the gate again")
	case why != "":
		return MergeClassRebase, fmt.Sprintf("%s: rebase on origin/%s and run %s again %s", why, def, gate, mergeGateRunsWhileOrgRuns)
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
// to the others, which also names ralph org regate when one of the ready
// branches is in regated (mergeRegateBranches), and that the order is only as
// of the ls-remote.
func mergeOrderListNotes(res MergeOrderResult, hidden int, regated map[string]bool) []string {
	var notes []string
	if hidden > 0 {
		notes = append(notes, fmt.Sprintf("%d branch(es) whose PR is merged or closed are not shown (--all shows them)", hidden))
	}
	ready, readyRegated := 0, 0
	for _, e := range res.Entries {
		if e.Class == MergeClassReady {
			ready++
			if regated[e.Branch] {
				readyRegated++
			}
		}
	}
	if ready >= 2 {
		note := fmt.Sprintf("%d branches are ready, but merging one moves origin/%s: the other ready ones then become rebase, "+
			"so rebase them on origin/%s and run ralph org gate again %s", ready, res.DefaultBranch, res.DefaultBranch,
			mergeGateRunsWhileOrgRuns)
		if readyRegated > 0 {
			note += "; a branch whose org is disbanded is brought up to date with ralph org regate --branch <branch> instead"
		}
		notes = append(notes, note)
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
	return runMergeOrderCommand(ctx, false, dir, env, name, args...)
}

// realMergeDetachedExec is realMergeOrderExec with the command in a process
// group of its own, as the gate scripts (runGateScript) and regate's fetch
// and push (regateBoundedGit) run: a signal the terminal sends its foreground
// group (Ctrl-C, a hangup) reaches ralph and not the command, and ctx done
// kills the whole group, the command's children with it. Merge runs the gh
// calls it makes once gh pr merge has started through it
// (mergeCheck.runDetached). Release builds target darwin and linux only,
// where Setpgid and kill(-pgid) exist.
func realMergeDetachedExec(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	return runMergeOrderCommand(ctx, true, dir, env, name, args...)
}

// runMergeOrderCommand runs the command of realMergeOrderExec, and with
// ownGroup the one of realMergeDetachedExec.
func runMergeOrderCommand(ctx context.Context, ownGroup bool, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	if ownGroup {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			return nil
		}
	}
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
