package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"
)

// The GitHub merge queue calls of the queue path of `ralph org merge` (spec
// docs/specs/2026-11-19-org-merge-queue.md, FR-4 to FR-9 and FR-5b; plan
// 2026-11-20-org-merge-queue-verb, D3 to D5): readMergeQueuePR, the one
// GraphQL read of a PR's merge queue state, and enqueueMergeQueuePR, the
// enqueuePullRequest mutation. Each query is a fixed string, and its
// variables go to gh api as fields, never into the query text. The checks of
// the values, the argv, the parse of the answers, and the helpers about the
// PR's removals from the queue are pure functions, so tests pin each with no
// gh; the two runners only put them together around one gh call.

// mergeQueuePRQuery is the GraphQL query readMergeQueuePR reads a PR's merge
// queue state with: the PR's node ID (D3 of the plan: the ID
// enqueuePullRequest takes comes from the same answer as the state), its
// state, head, base, merge commit, whether it is in the merge queue and its
// entry there, and the last 50 events of its timeline that added it to the
// queue or removed it from it. $number is an Int!, so mergeQueuePRArgs passes
// it with -F; gh sends a -f value as a string, which GraphQL refuses for an
// Int!.
const mergeQueuePRQuery = "query($owner: String!, $name: String!, $number: Int!) " +
	"{ repository(owner: $owner, name: $name) { pullRequest(number: $number) { " +
	"id state headRefOid baseRefName mergeCommit { oid } isInMergeQueue mergeQueueEntry { state position enqueuedAt } " +
	"timelineItems(last: 50, itemTypes: [ADDED_TO_MERGE_QUEUE_EVENT, REMOVED_FROM_MERGE_QUEUE_EVENT]) { nodes { __typename " +
	"... on AddedToMergeQueueEvent { createdAt } " +
	"... on RemovedFromMergeQueueEvent { createdAt reason actor { login } } } } } } }"

// mergeQueueEnqueueMutation is the GraphQL mutation enqueueMergeQueuePR adds
// a PR to the merge queue with (FR-4): it passes the PR's node ID and the
// head the queue may merge, and gets back the entry GitHub made. Both
// variables are strings (ID! and GitObjectID!), so mergeQueueEnqueueArgs
// passes them with -f.
const mergeQueueEnqueueMutation = "mutation($pullRequestId: ID!, $expectedHeadOid: GitObjectID!) " +
	"{ enqueuePullRequest(input: {pullRequestId: $pullRequestId, expectedHeadOid: $expectedHeadOid}) " +
	"{ mergeQueueEntry { state position enqueuedAt } } }"

const (
	// mergeQueueClockSkew is how far apart FR-5b allows ralph's clock (the
	// gate record's ts) and GitHub's (a removal's createdAt) to be (D4 of the
	// plan): a removal less than this before the gate counts as after it,
	// so a skewed clock refuses a re-enqueue rather than lets one through
	// without --requeue.
	mergeQueueClockSkew = 5 * time.Minute
	// mergeQueueMaxNodeID is the longest PR node ID mergeQueueEnqueueCheck
	// lets through. GitHub's node IDs are some 20 to 40 characters long.
	mergeQueueMaxNodeID = 128
)

// mergeQueueNodeIDPattern is the form of a PR node ID mergeQueueEnqueueCheck
// lets through: 1 to mergeQueueMaxNodeID characters of A-Z a-z 0-9 _ - =,
// the characters of GitHub's node IDs, both the current (PR_kwDO...) and the
// legacy base64 ones (MDExOlB1bGxSZXF1ZXN0...). A leading - is let through:
// the ID goes to gh inside the one argument pullRequestId=<id>, so gh never
// reads it as a flag.
var mergeQueueNodeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_=-]{1,` + strconv.Itoa(mergeQueueMaxNodeID) + `}$`)

// errMergeQueueInput marks an error of readMergeQueuePR or
// enqueueMergeQueuePR from a value that failed its check, before any gh ran:
// errors.Is(err, errMergeQueueInput) tells the caller that nothing reached
// GitHub, so the PR was not enqueued.
var errMergeQueueInput = errors.New("not passed to gh")

// mergeQueuePR is a PR's merge queue state as readMergeQueuePR read it. ID,
// State, HeadRefOID, BaseRefName and the removals' Reason and Actor are
// GitHub's text as it is: escape any of them before it reaches a terminal
// (mergeQueueRemovalText does for a removal).
type mergeQueuePR struct {
	ID          string // the PR's node ID, which enqueuePullRequest takes
	State       string // OPEN, CLOSED or MERGED
	HeadRefOID  string
	BaseRefName string
	// MergeCommit is mergeCommit.oid when it is a commit SHA
	// (gitObjectPattern), else "": mergeCommit is null for a PR that is not
	// merged. This is what mergeReadBack.mergeCommit does too.
	MergeCommit string
	InQueue     bool             // isInMergeQueue
	Entry       *mergeQueueEntry // nil when mergeQueueEntry is null
	// Added holds the createdAt of each AddedToMergeQueueEvent, and Removed
	// each RemovedFromMergeQueueEvent, of the last 50 such events, in the
	// answer's order.
	Added   []time.Time
	Removed []mergeQueueRemoval
}

// mergeQueueEntry is a PR's entry in the merge queue: a PR's
// mergeQueueEntry, or the one enqueuePullRequest returned.
type mergeQueueEntry struct {
	State      string // GitHub's MergeQueueEntryState, which ralph decides nothing by
	Position   int
	EnqueuedAt time.Time
}

// mergeQueueRemoval is a RemovedFromMergeQueueEvent of a PR's timeline.
type mergeQueueRemoval struct {
	At     time.Time // createdAt
	Reason string    // GitHub's free text, "" when null
	Actor  string    // actor.login, "" when actor is null
}

// mergeQueueEntryJSON is a mergeQueueEntry as GraphQL answers it.
type mergeQueueEntryJSON struct {
	State      string `json:"state"`
	Position   int    `json:"position"`
	EnqueuedAt string `json:"enqueuedAt"`
}

// mergeQueueTimelineJSON is a node of the PR's timelineItems: an
// AddedToMergeQueueEvent or a RemovedFromMergeQueueEvent.
type mergeQueueTimelineJSON struct {
	Typename  string  `json:"__typename"`
	CreatedAt string  `json:"createdAt"`
	Reason    *string `json:"reason"`
	Actor     *struct {
		Login string `json:"login"`
	} `json:"actor"`
}

// mergeQueuePRJSON is repository.pullRequest of mergeQueuePRQuery's answer.
// The pointers tell a field that is missing or null from one that is empty.
type mergeQueuePRJSON struct {
	ID             string `json:"id"`
	State          string `json:"state"`
	HeadRefOID     string `json:"headRefOid"`
	BaseRefName    string `json:"baseRefName"`
	IsInMergeQueue *bool  `json:"isInMergeQueue"`
	MergeCommit    *struct {
		OID string `json:"oid"`
	} `json:"mergeCommit"`
	MergeQueueEntry *mergeQueueEntryJSON `json:"mergeQueueEntry"`
	TimelineItems   *struct {
		Nodes *[]*mergeQueueTimelineJSON `json:"nodes"`
	} `json:"timelineItems"`
}

// mergeQueuePRCheck checks the values of a readMergeQueuePR call before any
// argv is built: the host, owner and name are not empty, and the PR number is
// one GraphQL's Int can hold, 1 to 2147483647. The error wraps
// errMergeQueueInput.
func mergeQueuePRCheck(host string, repo mergeRepo, number int) error {
	switch {
	case host == "":
		return fmt.Errorf("the GitHub host is empty: %w", errMergeQueueInput)
	case repo.owner == "" || repo.name == "":
		return fmt.Errorf("the repository %s has an empty owner or name: %w", mergeQueueShown(repo.String()), errMergeQueueInput)
	case number <= 0 || number > math.MaxInt32:
		return fmt.Errorf("the PR number %d is not 1 to %d: %w", number, math.MaxInt32, errMergeQueueInput)
	}
	return nil
}

// mergeQueueEnqueueCheck checks the values of an enqueueMergeQueuePR call
// before any argv is built: the host is not empty, prID has the form of a
// GitHub node ID (mergeQueueNodeIDPattern), and head is a commit SHA, 40 or
// 64 lowercase hex digits (gitObjectPattern). The error wraps
// errMergeQueueInput.
func mergeQueueEnqueueCheck(host, prID, head string) error {
	switch {
	case host == "":
		return fmt.Errorf("the GitHub host is empty: %w", errMergeQueueInput)
	case !mergeQueueNodeIDPattern.MatchString(prID):
		return fmt.Errorf("the PR node ID %s is not 1 to %d characters of A-Z a-z 0-9 _ - =: %w", mergeQueueShown(prID),
			mergeQueueMaxNodeID, errMergeQueueInput)
	case !gitObjectPattern.MatchString(head):
		return fmt.Errorf("the head %s is not a commit SHA of 40 or 64 lowercase hex digits: %w", mergeQueueShown(head),
			errMergeQueueInput)
	}
	return nil
}

// mergeQueueShown is s quoted for an error of a check (strconv.Quote, which
// writes each control character as its Go escape), cut to its first 64 bytes
// with "..." after the quotes.
func mergeQueueShown(s string) string {
	const limit = 64
	if len(s) > limit {
		return strconv.Quote(s[:limit]) + "..."
	}
	return strconv.Quote(s)
}

// mergeQueuePRArgs are the arguments of readMergeQueuePR's gh api graphql
// call, the endpoint last as in mergeQueueArgs. owner and name go with -f,
// which gh sends as strings; the number goes with -F, which gh sends as a
// JSON number, because mergeQueuePRQuery declares $number: Int!. A number
// mergeQueuePRCheck let through is digits only, so -F cannot read it as a
// file (@) or a placeholder ({owner}).
func mergeQueuePRArgs(host string, repo mergeRepo, number int) []string {
	return []string{"api", "--hostname", host, "-f", "query=" + mergeQueuePRQuery, "-f", "owner=" + repo.owner,
		"-f", "name=" + repo.name, "-F", "number=" + strconv.Itoa(number), "graphql"}
}

// mergeQueueEnqueueArgs are the arguments of enqueueMergeQueuePR's gh api
// graphql call. Both variables are strings and go with -f, which gh sends as
// they are.
func mergeQueueEnqueueArgs(host, prID, head string) []string {
	return []string{"api", "--hostname", host, "-f", "query=" + mergeQueueEnqueueMutation, "-f", "pullRequestId=" + prID,
		"-f", "expectedHeadOid=" + head, "graphql"}
}

// mergeQueueGraphQLErrors is the error of a GraphQL answer's errors member:
// nil when it is missing, null or an empty list, else an error that quotes it
// through mergeGuaranteeErrText. GraphQL may answer with data and errors
// both, and an answer with errors is not read, whatever data it has.
func mergeQueueGraphQLErrors(raw json.RawMessage) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err == nil && len(list) == 0 {
		return nil
	}
	return fmt.Errorf("answered with errors (%s)", mergeGuaranteeErrText(errors.New(string(raw))))
}

// parseMergeQueueTime parses value, the GraphQL DateTime of the field what,
// as RFC 3339.
func parseMergeQueueTime(what, value string) (time.Time, error) {
	at, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("returned a time that is not RFC 3339 in %s (%s)", what, mergeGuaranteeErrText(err))
	}
	return at, nil
}

// entry is j with enqueuedAt parsed; an enqueuedAt that is not an RFC 3339
// time is an error.
func (j mergeQueueEntryJSON) entry(what string) (mergeQueueEntry, error) {
	at, err := parseMergeQueueTime(what+".enqueuedAt", j.EnqueuedAt)
	if err != nil {
		return mergeQueueEntry{}, err
	}
	return mergeQueueEntry{State: j.State, Position: j.Position, EnqueuedAt: at}, nil
}

// parseMergeQueuePR reads the answer of mergeQueuePRQuery, out. It is an
// error when out is not JSON, when errors is set (mergeQueueGraphQLErrors),
// when data, repository or pullRequest is missing or null, and when the PR
// lacks a field mergeQueuePRJSON.pr requires. Each error's text starts with
// a verb, for readMergeQueuePR to put after the command.
func parseMergeQueuePR(out string) (mergeQueuePR, error) {
	var body struct {
		Data *struct {
			Repository *struct {
				PullRequest *mergeQueuePRJSON `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		return mergeQueuePR{}, fmt.Errorf("printed output that is not GraphQL JSON (%s)", mergeGuaranteeErrText(err))
	}
	if err := mergeQueueGraphQLErrors(body.Errors); err != nil {
		return mergeQueuePR{}, err
	}
	switch {
	case body.Data == nil:
		return mergeQueuePR{}, errors.New("returned no data")
	case body.Data.Repository == nil:
		return mergeQueuePR{}, errors.New("returned no repository")
	case body.Data.Repository.PullRequest == nil:
		return mergeQueuePR{}, errors.New("returned no repository.pullRequest")
	}
	return body.Data.Repository.PullRequest.pr()
}

// pr is j as a mergeQueuePR. The fields GitHub's schema makes non-null (id,
// state, headRefOid, baseRefName, isInMergeQueue, timelineItems and its
// nodes) must be there and not empty; a timeline node that is null or of a
// type the query did not ask for, and a time that is not RFC 3339, are
// errors too. Without the timeline, a removal FR-5b refuses on could go
// unseen.
func (j *mergeQueuePRJSON) pr() (mergeQueuePR, error) {
	for _, f := range []struct{ name, value string }{
		{"id", j.ID}, {"state", j.State}, {"headRefOid", j.HeadRefOID}, {"baseRefName", j.BaseRefName},
	} {
		if f.value == "" {
			return mergeQueuePR{}, fmt.Errorf("returned a pullRequest with no %s", f.name)
		}
	}
	if j.IsInMergeQueue == nil {
		return mergeQueuePR{}, errors.New("returned a pullRequest with no isInMergeQueue")
	}
	if j.TimelineItems == nil || j.TimelineItems.Nodes == nil {
		return mergeQueuePR{}, errors.New("returned a pullRequest with no timelineItems.nodes")
	}
	pr := mergeQueuePR{ID: j.ID, State: j.State, HeadRefOID: j.HeadRefOID, BaseRefName: j.BaseRefName, InQueue: *j.IsInMergeQueue}
	if j.MergeCommit != nil && gitObjectPattern.MatchString(j.MergeCommit.OID) {
		pr.MergeCommit = j.MergeCommit.OID
	}
	if j.MergeQueueEntry != nil {
		entry, err := j.MergeQueueEntry.entry("mergeQueueEntry")
		if err != nil {
			return mergeQueuePR{}, err
		}
		pr.Entry = &entry
	}
	var err error
	if pr.Added, pr.Removed, err = mergeQueueTimeline(*j.TimelineItems.Nodes); err != nil {
		return mergeQueuePR{}, err
	}
	return pr, nil
}

// mergeQueueTimeline is the times of the AddedToMergeQueueEvent nodes and
// the RemovedFromMergeQueueEvent nodes of a PR's timeline, in the nodes'
// order. A null node, a node of any other __typename, and a createdAt that is
// not an RFC 3339 time are errors.
func mergeQueueTimeline(nodes []*mergeQueueTimelineJSON) ([]time.Time, []mergeQueueRemoval, error) {
	var added []time.Time
	var removed []mergeQueueRemoval
	for _, n := range nodes {
		if n == nil {
			return nil, nil, errors.New("returned a timeline item that is null")
		}
		switch n.Typename {
		case "AddedToMergeQueueEvent":
			at, err := parseMergeQueueTime("AddedToMergeQueueEvent.createdAt", n.CreatedAt)
			if err != nil {
				return nil, nil, err
			}
			added = append(added, at)
		case "RemovedFromMergeQueueEvent":
			at, err := parseMergeQueueTime("RemovedFromMergeQueueEvent.createdAt", n.CreatedAt)
			if err != nil {
				return nil, nil, err
			}
			r := mergeQueueRemoval{At: at}
			if n.Reason != nil {
				r.Reason = *n.Reason
			}
			if n.Actor != nil {
				r.Actor = n.Actor.Login
			}
			removed = append(removed, r)
		default:
			return nil, nil, fmt.Errorf("returned a timeline item of the type %s, which the query did not ask for",
				mergeQueueShown(n.Typename))
		}
	}
	return added, removed, nil
}

// parseMergeQueueEnqueue reads the answer of mergeQueueEnqueueMutation, out:
// the entry enqueuePullRequest made. It is an error when out is not JSON,
// when errors is set, when data, enqueuePullRequest or its mergeQueueEntry is
// missing or null, and when enqueuedAt is not an RFC 3339 time. With no entry
// it is unknown whether the PR went into the queue (D5 of the plan), so the
// caller records nothing.
func parseMergeQueueEnqueue(out string) (mergeQueueEntry, error) {
	var body struct {
		Data *struct {
			EnqueuePullRequest *struct {
				MergeQueueEntry *mergeQueueEntryJSON `json:"mergeQueueEntry"`
			} `json:"enqueuePullRequest"`
		} `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &body); err != nil {
		return mergeQueueEntry{}, fmt.Errorf("printed output that is not GraphQL JSON (%s)", mergeGuaranteeErrText(err))
	}
	if err := mergeQueueGraphQLErrors(body.Errors); err != nil {
		return mergeQueueEntry{}, err
	}
	if body.Data == nil || body.Data.EnqueuePullRequest == nil || body.Data.EnqueuePullRequest.MergeQueueEntry == nil {
		return mergeQueueEntry{}, errors.New("returned no enqueuePullRequest.mergeQueueEntry")
	}
	return body.Data.EnqueuePullRequest.MergeQueueEntry.entry("enqueuePullRequest.mergeQueueEntry")
}

// mergeQueueGHError is the error of a gh call of readMergeQueuePR or
// enqueueMergeQueuePR that failed: its text names the command and quotes the
// failure through mergeGuaranteeErrText, so a control character of gh's
// stderr does not reach the terminal raw, and Unwrap gives the failure as
// the exec returned it (errors.Is(err, exec.ErrNotFound) for a gh that is
// not installed).
type mergeQueueGHError struct {
	cmd string
	err error
}

func (e *mergeQueueGHError) Error() string {
	return fmt.Sprintf("%s failed (%s)", e.cmd, mergeGuaranteeErrText(e.err))
}

func (e *mergeQueueGHError) Unwrap() error { return e.err }

// readMergeQueuePR reads the merge queue state of the PR number of repo on
// host with one `gh api --hostname <host> graphql` call through run in dir,
// with GH_PROMPT_DISABLED=1 (FR-5, FR-6, FR-9; D3 of the plan: the PR's node
// ID comes from the same answer). It checks the values first
// (mergeQueuePRCheck: an error that wraps errMergeQueueInput, and no gh
// runs). A gh that fails is a *mergeQueueGHError, and an answer
// parseMergeQueuePR cannot read is an error that names the command. It sets no
// timeout of its own: ctx bounds the call, and when ctx is done the error is
// gh's, so the caller checks ctx.Err().
func readMergeQueuePR(ctx context.Context, run MergeOrderExec, dir, host string, repo mergeRepo, number int) (mergeQueuePR, error) {
	if err := mergeQueuePRCheck(host, repo, number); err != nil {
		return mergeQueuePR{}, err
	}
	cmd := fmt.Sprintf("gh api --hostname %s graphql (the merge queue state of %s#%d)", host, repo, number)
	out, err := run(ctx, dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", mergeQueuePRArgs(host, repo, number)...)
	if err != nil {
		return mergeQueuePR{}, &mergeQueueGHError{cmd: cmd, err: err}
	}
	pr, err := parseMergeQueuePR(out)
	if err != nil {
		return mergeQueuePR{}, fmt.Errorf("%s %w", cmd, err)
	}
	return pr, nil
}

// enqueueMergeQueuePR adds the PR whose node ID is prID to the merge queue at
// head, with the enqueuePullRequest mutation in one `gh api --hostname <host>
// graphql` call through run in dir, with GH_PROMPT_DISABLED=1 (FR-4), and
// returns the entry GitHub made. It checks the values first
// (mergeQueueEnqueueCheck: an error that wraps errMergeQueueInput, and no gh
// runs, so the PR was not enqueued). Any other error comes after gh ran, so
// whether the PR went into the queue is unknown (D5 of the plan): a gh that
// fails is a *mergeQueueGHError, and an answer parseMergeQueueEnqueue cannot
// read, an entry missing included, is an error that names the command. Like
// readMergeQueuePR it sets no timeout of its own.
func enqueueMergeQueuePR(ctx context.Context, run MergeOrderExec, dir, host, prID, head string) (mergeQueueEntry, error) {
	if err := mergeQueueEnqueueCheck(host, prID, head); err != nil {
		return mergeQueueEntry{}, err
	}
	cmd := fmt.Sprintf("gh api --hostname %s graphql (enqueuePullRequest of %s at %s)", host, prID, head)
	out, err := run(ctx, dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", mergeQueueEnqueueArgs(host, prID, head)...)
	if err != nil {
		return mergeQueueEntry{}, &mergeQueueGHError{cmd: cmd, err: err}
	}
	entry, err := parseMergeQueueEnqueue(out)
	if err != nil {
		return mergeQueueEntry{}, fmt.Errorf("%s %w", cmd, err)
	}
	return entry, nil
}

// latestMergeQueueRemovalAfter returns the latest of removals whose time is
// strictly after after, and whether there is one: the removal FR-9 reports
// once ralph enqueued the PR at after (D4 of the plan: the entry's
// enqueuedAt, GitHub's clock like the removal's).
func latestMergeQueueRemovalAfter(removals []mergeQueueRemoval, after time.Time) (latest mergeQueueRemoval, ok bool) {
	for _, r := range removals {
		if r.At.After(after) && (!ok || r.At.After(latest.At)) {
			latest, ok = r, true
		}
	}
	return latest, ok
}

// mergeQueueRemovedAfterGate reports whether a removal at removedAt (GitHub's
// clock) counts as after the gate record of gateAt (ralph's clock) for FR-5b:
// removedAt is strictly after gateAt minus mergeQueueClockSkew (D4 of the
// plan). A removal up to that much before the gate counts as after it, so a
// clock skew refuses a re-enqueue without --requeue rather than lets one
// through. A zero gateAt counts every removal.
func mergeQueueRemovedAfterGate(removedAt, gateAt time.Time) bool {
	return removedAt.After(gateAt.Add(-mergeQueueClockSkew))
}

// mergeQueueRemovalAfterGate returns the latest of removals that counts as
// after the gate record of gateAt (mergeQueueRemovedAfterGate), and whether
// there is one: the removal FR-5b refuses a re-enqueue without --requeue on.
func mergeQueueRemovalAfterGate(removals []mergeQueueRemoval, gateAt time.Time) (latest mergeQueueRemoval, ok bool) {
	for _, r := range removals {
		if mergeQueueRemovedAfterGate(r.At, gateAt) && (!ok || r.At.After(latest.At)) {
			latest, ok = r, true
		}
	}
	return latest, ok
}

// mergeQueueRemovalText is r for ralph's output: when GitHub removed the PR
// from the merge queue (in UTC), by whom, and why. The reason and the login
// come from GitHub as they are, so each control character in them is written
// as its Go escape (PrintableInboxText), newlines too, which keeps the text
// on one line.
func mergeQueueRemovalText(r mergeQueueRemoval) string {
	by := "an actor GitHub does not name"
	if r.Actor != "" {
		by = PrintableInboxText(r.Actor, false)
	}
	reason := "none given"
	if r.Reason != "" {
		reason = PrintableInboxText(r.Reason, false)
	}
	return fmt.Sprintf("removed from the merge queue at %s by %s (reason: %s)", r.At.UTC().Format(time.RFC3339), by, reason)
}
