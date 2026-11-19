package org

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"
)

// The merge queue calls of merge_queue.go (spec
// docs/specs/2026-11-19-org-merge-queue.md, FR-4 to FR-9 and FR-5b; plan
// 2026-11-20-org-merge-queue-verb, D3 to D5, AC13 and AC16) with mqExec, a
// fake exec that records each call and answers with one fixed output. No test
// runs gh or reaches a network.

// The values the tests read and enqueue with: a PR node ID, the PR's head
// and the merge commit of a merged PR.
const (
	mqID    = "PR_kwDOAbc123"
	mqHead  = "0123456789abcdef0123456789abcdef01234567"
	mqMerge = "89abcdef0123456789abcdef0123456789abcdef"
)

// mqCall is one call of mqExec.run.
type mqCall struct {
	ctx  context.Context
	dir  string
	env  []string
	name string
	args []string
}

// mqExec is a MergeOrderExec that records each call and returns out and err.
type mqExec struct {
	calls []mqCall
	out   string
	err   error
}

func (e *mqExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	e.calls = append(e.calls, mqCall{ctx: ctx, dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args)})
	return e.out, e.err
}

// mqCtxKey marks the ctx a test passes, to show that a runner hands gh the
// caller's ctx and not one of its own.
type mqCtxKey struct{}

// mqPRFields is the pullRequest of a mergeQueuePRQuery answer for an open PR
// that is not in the merge queue and has no queue events; tests change its
// fields.
func mqPRFields() map[string]any {
	return map[string]any{"id": mqID, "state": "OPEN", "headRefOid": mqHead, "baseRefName": "main", "mergeCommit": nil,
		"isInMergeQueue": false, "mergeQueueEntry": nil, "timelineItems": mqNodes()}
}

// mqPRAnswer is a mergeQueuePRQuery answer whose pullRequest is pr.
func mqPRAnswer(pr map[string]any) string {
	return mgJSON(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": pr}}})
}

// mqNodes is a timelineItems value with nodes.
func mqNodes(nodes ...any) map[string]any {
	return map[string]any{"nodes": append([]any{}, nodes...)}
}

// mqEntry is a mergeQueueEntry value.
func mqEntry(state string, position int, enqueuedAt string) map[string]any {
	return map[string]any{"state": state, "position": position, "enqueuedAt": enqueuedAt}
}

// mqRemoved is a RemovedFromMergeQueueEvent node; a nil reason is null, and a
// nil actor makes actor null.
func mqRemoved(at string, reason, actor any) map[string]any {
	n := map[string]any{"__typename": "RemovedFromMergeQueueEvent", "createdAt": at, "reason": reason, "actor": nil}
	if actor != nil {
		n["actor"] = map[string]any{"login": actor}
	}
	return n
}

// mqEnqueueAnswer is an enqueuePullRequest answer whose mergeQueueEntry is
// entry (nil makes it null).
func mqEnqueueAnswer(entry any) string {
	return mgJSON(map[string]any{"data": map[string]any{"enqueuePullRequest": map[string]any{"mergeQueueEntry": entry}}})
}

// mqTime is s parsed as RFC 3339.
func mqTime(t *testing.T, s string) time.Time {
	t.Helper()
	at, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

// mqTimeText is t in UTC, for the comparisons of mqPRText.
func mqTimeText(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// mqPRText is pr as one line, with its times in UTC, so a table can compare
// a whole mergeQueuePR with a string.
func mqPRText(pr mergeQueuePR) string {
	var b strings.Builder
	fmt.Fprintf(&b, "id=%s state=%s head=%s base=%s merge=%s inQueue=%t", pr.ID, pr.State, pr.HeadRefOID, pr.BaseRefName,
		pr.MergeCommit, pr.InQueue)
	if pr.Entry != nil {
		fmt.Fprintf(&b, " entry=%s/%d/%s", pr.Entry.State, pr.Entry.Position, mqTimeText(pr.Entry.EnqueuedAt))
	}
	for _, r := range pr.Removed {
		fmt.Fprintf(&b, " removed=%s/%q/%q", mqTimeText(r.At), r.Reason, r.Actor)
	}
	return b.String()
}

// mqAssertNoControl fails the test when s holds a control character.
func mqAssertNoControl(t *testing.T, what, s string) {
	t.Helper()
	if i := strings.IndexFunc(s, unicode.IsControl); i >= 0 {
		t.Errorf("%s holds a control character at byte %d: %q", what, i, s)
	}
}

// mqAssertOneGHCall fails the test unless e ran exactly one gh call, in /repo
// with exactly GH_PROMPT_DISABLED=1, with the arguments want and the caller's
// ctx (no deadline of the runner's own), and returns that call.
func mqAssertOneGHCall(t *testing.T, e *mqExec, want []string) mqCall {
	t.Helper()
	if len(e.calls) != 1 {
		t.Fatalf("calls = %d, want 1: %+v", len(e.calls), e.calls)
	}
	c := e.calls[0]
	if c.name != "gh" || c.dir != "/repo" || !slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) || !slices.Equal(c.args, want) {
		t.Errorf("call = %s %q in %q with %q, want gh %q in /repo with [GH_PROMPT_DISABLED=1]", c.name, c.args, c.dir, c.env, want)
	}
	if c.ctx.Value(mqCtxKey{}) != "caller" {
		t.Errorf("the gh call did not get the caller's ctx")
	}
	if _, ok := c.ctx.Deadline(); ok {
		t.Errorf("the gh call got a deadline the caller did not set")
	}
	return c
}

// mqFieldFlags is each variable of a gh api call and the flag before it
// (-f or -F), the query left out.
func mqFieldFlags(t *testing.T, args []string) map[string]string {
	t.Helper()
	flags := map[string]string{}
	for i := 0; i < len(args); i++ {
		if args[i] != "-f" && args[i] != "-F" {
			continue
		}
		if i+1 >= len(args) {
			t.Fatalf("%s is the last argument: %q", args[i], args)
		}
		name, _, ok := strings.Cut(args[i+1], "=")
		if !ok {
			t.Fatalf("%s %q has no =", args[i], args[i+1])
		}
		if name != "query" {
			flags[name] = args[i]
		}
		i++
	}
	return flags
}

// TestMergeQueue_ReadRunsOneGraphQLCallWithTheNumberAsAnInt covers AC13 for
// the read: one gh api graphql call with --hostname <host> and
// GH_PROMPT_DISABLED=1, the fixed query, owner and name with -f and the
// number with -F (the query declares $number: Int!, and gh sends a -f value
// as a string), in the caller's ctx.
func TestMergeQueue_ReadRunsOneGraphQLCallWithTheNumberAsAnInt(t *testing.T) {
	tests := []struct {
		host, owner, name string
		number            int
	}{
		{"github.com", "acme", "app", 12},
		{"ghe.example.com", "Org-1", "repo.name", math.MaxInt32},
		{"github.com", "acme", "app", 1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s/%s#%d", tt.host, tt.owner, tt.name, tt.number), func(t *testing.T) {
			e := &mqExec{out: mqPRAnswer(mqPRFields())}
			ctx := context.WithValue(context.Background(), mqCtxKey{}, "caller")
			if _, err := readMergeQueuePR(ctx, e.run, "/repo", tt.host, mergeRepo{tt.owner, tt.name}, tt.number); err != nil {
				t.Fatal(err)
			}
			want := []string{"api", "--hostname", tt.host, "-f", "query=" + mergeQueuePRQuery, "-f", "owner=" + tt.owner,
				"-f", "name=" + tt.name, "-F", "number=" + strconv.Itoa(tt.number), "graphql"}
			c := mqAssertOneGHCall(t, e, want)
			wantFlags := map[string]string{"owner": "-f", "name": "-f", "number": "-F"}
			if got := mqFieldFlags(t, c.args); !maps.Equal(got, wantFlags) {
				t.Errorf("variable flags = %v, want %v", got, wantFlags)
			}
		})
	}
	if !strings.Contains(mergeQueuePRQuery, "$number: Int!") || !strings.Contains(mergeQueuePRQuery, "pullRequest(number: $number)") {
		t.Errorf("mergeQueuePRQuery does not declare and use $number as an Int!: %s", mergeQueuePRQuery)
	}
}

// TestMergeQueue_EnqueueRunsOneGraphQLCallWithStringVariables covers FR-4 and
// AC13 for the mutation: one gh api graphql call with --hostname <host> and
// GH_PROMPT_DISABLED=1, the fixed mutation, and the node ID and the head
// with -f, as the strings ID! and GitObjectID! take, in the caller's ctx.
func TestMergeQueue_EnqueueRunsOneGraphQLCallWithStringVariables(t *testing.T) {
	head64 := strings.Repeat("ab", 32)
	tests := []struct{ host, id, head string }{
		{"github.com", mqID, mqHead},
		{"ghe.example.com", "MDExOlB1bGxSZXF1ZXN0MTIzNA==", head64},
	}
	for _, tt := range tests {
		t.Run(tt.host+"/"+tt.id, func(t *testing.T) {
			e := &mqExec{out: mqEnqueueAnswer(mqEntry("QUEUED", 1, "2026-11-20T10:00:00Z"))}
			ctx := context.WithValue(context.Background(), mqCtxKey{}, "caller")
			if _, err := enqueueMergeQueuePR(ctx, e.run, "/repo", tt.host, tt.id, tt.head); err != nil {
				t.Fatal(err)
			}
			want := []string{"api", "--hostname", tt.host, "-f", "query=" + mergeQueueEnqueueMutation, "-f", "pullRequestId=" + tt.id,
				"-f", "expectedHeadOid=" + tt.head, "graphql"}
			c := mqAssertOneGHCall(t, e, want)
			wantFlags := map[string]string{"pullRequestId": "-f", "expectedHeadOid": "-f"}
			if got := mqFieldFlags(t, c.args); !maps.Equal(got, wantFlags) {
				t.Errorf("variable flags = %v, want %v", got, wantFlags)
			}
		})
	}
}

// TestMergeQueue_QueriesAreFixedAndDeclareTheirVariables pins the two fixed
// GraphQL strings: their brackets balance, each declares its variables with
// the types GitHub's schema gives them and uses each, and the read asks for
// every field the queue path decides by, and for no AddedToMergeQueueEvent,
// which nothing reads.
func TestMergeQueue_QueriesAreFixedAndDeclareTheirVariables(t *testing.T) {
	tests := []struct {
		name, text, start string
		vars              map[string]string // variable -> declared type
		fields            []string
		absent            []string
	}{
		{
			name: "read", text: mergeQueuePRQuery, start: "query(",
			vars: map[string]string{"owner": "String!", "name": "String!", "number": "Int!"},
			fields: []string{"repository(owner: $owner, name: $name)", "pullRequest(number: $number)", " id ", " state ",
				" headRefOid ", " baseRefName ", "mergeCommit { oid }", " isInMergeQueue ", "mergeQueueEntry { state position enqueuedAt }",
				"timelineItems(last: 50, itemTypes: [REMOVED_FROM_MERGE_QUEUE_EVENT])", "__typename",
				"... on RemovedFromMergeQueueEvent { createdAt reason actor { login } }"},
			absent: []string{"ADDED_TO_MERGE_QUEUE_EVENT", "AddedToMergeQueueEvent"},
		},
		{
			name: "enqueue", text: mergeQueueEnqueueMutation, start: "mutation(",
			vars: map[string]string{"pullRequestId": "ID!", "expectedHeadOid": "GitObjectID!"},
			fields: []string{"enqueuePullRequest(input: {pullRequestId: $pullRequestId, expectedHeadOid: $expectedHeadOid})",
				"mergeQueueEntry { state position enqueuedAt }"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.HasPrefix(tt.text, tt.start) {
				t.Errorf("%q does not start with %q", tt.text, tt.start)
			}
			for _, pair := range []string{"{}", "()", "[]"} {
				depth := 0
				for _, r := range tt.text {
					switch r {
					case rune(pair[0]):
						depth++
					case rune(pair[1]):
						depth--
					}
					if depth < 0 {
						t.Fatalf("%c closes before it opens in %q", pair[1], tt.text)
					}
				}
				if depth != 0 {
					t.Errorf("%s does not balance (depth %d) in %q", pair, depth, tt.text)
				}
			}
			head, body, _ := strings.Cut(tt.text, "{")
			if got := strings.Count(head, "$"); got != len(tt.vars) {
				t.Errorf("%q declares %d variables, want %d", head, got, len(tt.vars))
			}
			for v, typ := range tt.vars {
				if !strings.Contains(head, "$"+v+": "+typ) {
					t.Errorf("%q does not declare $%s: %s", head, v, typ)
				}
				if !strings.Contains(body, "$"+v) {
					t.Errorf("the body does not use $%s: %q", v, body)
				}
			}
			for _, f := range tt.fields {
				if !strings.Contains(tt.text, f) {
					t.Errorf("%q does not hold %q", tt.text, f)
				}
			}
			for _, a := range tt.absent {
				if strings.Contains(tt.text, a) {
					t.Errorf("%q holds %q", tt.text, a)
				}
			}
		})
	}
}

// TestMergeQueue_ChecksRefuseBadValuesBeforeAnyGH covers AC13's checks: a PR
// number that is not 1 to 2147483647, an empty host, owner or name, a node ID
// that is not 1 to 128 characters of A-Z a-z 0-9 _ - =, and a head that is
// not 40 or 64 lowercase hex digits are refused with an error that wraps
// errMergeQueueInput, holds no control character, and runs no gh. A leading -
// is let through, since the ID goes to gh inside pullRequestId=<id>.
func TestMergeQueue_ChecksRefuseBadValuesBeforeAnyGH(t *testing.T) {
	type call func(e *mqExec) error
	read := func(host, owner, name string, number int) call {
		return func(e *mqExec) error {
			_, err := readMergeQueuePR(context.Background(), e.run, "/repo", host, mergeRepo{owner, name}, number)
			return err
		}
	}
	enqueue := func(host, id, head string) call {
		return func(e *mqExec) error {
			_, err := enqueueMergeQueuePR(context.Background(), e.run, "/repo", host, id, head)
			return err
		}
	}
	bad := []struct {
		name string
		call call
		want string
	}{
		{"number 0", read("github.com", "acme", "app", 0), "the PR number 0 is not 1 to 2147483647"},
		{"number -1", read("github.com", "acme", "app", -1), "the PR number -1 is not 1 to 2147483647"},
		{"number past Int", read("github.com", "acme", "app", math.MaxInt32+1), "the PR number 2147483648 is not 1 to 2147483647"},
		{"read with no host", read("", "acme", "app", 12), "the GitHub host is empty"},
		{"no owner", read("github.com", "", "app", 12), `the repository "/app" has an empty owner or name`},
		{"no name", read("github.com", "acme", "", 12), `the repository "acme/" has an empty owner or name`},
		{"enqueue with no host", enqueue("", mqID, mqHead), "the GitHub host is empty"},
		{"empty node ID", enqueue("github.com", "", mqHead), `the PR node ID "" is not 1 to 128 characters`},
		{"node ID of 129", enqueue("github.com", strings.Repeat("A", 129), mqHead), "is not 1 to 128 characters"},
		{"node ID with a space", enqueue("github.com", "PR_a b", mqHead), `the PR node ID "PR_a b" is not`},
		{"node ID with ;", enqueue("github.com", "PR_a;b", mqHead), `the PR node ID "PR_a;b" is not`},
		{"node ID with a newline", enqueue("github.com", "PR_a\nb", mqHead), `the PR node ID "PR_a\nb" is not`},
		{"node ID with ESC", enqueue("github.com", "PR_a\x1b[31m", mqHead), `the PR node ID "PR_a\x1b[31m" is not`},
		{"node ID with /", enqueue("github.com", "PR/a", mqHead), `the PR node ID "PR/a" is not`},
		{"node ID with +", enqueue("github.com", "PR+a", mqHead), `the PR node ID "PR+a" is not`},
		{"node ID with braces", enqueue("github.com", "{owner}", mqHead), `the PR node ID "{owner}" is not`},
		{"node ID with @", enqueue("github.com", "@file", mqHead), `the PR node ID "@file" is not`},
		{"node ID not ASCII", enqueue("github.com", "PR_\xc3\xa9", mqHead), "the PR node ID"},
		{"empty head", enqueue("github.com", mqID, ""), `the head "" is not a commit SHA`},
		{"short head", enqueue("github.com", mqID, "0123456"), `the head "0123456" is not a commit SHA`},
		{"head of 39", enqueue("github.com", mqID, mqHead[:39]), "is not a commit SHA"},
		{"head of 41", enqueue("github.com", mqID, mqHead+"0"), "is not a commit SHA"},
		{"head of 63", enqueue("github.com", mqID, strings.Repeat("a", 63)), "is not a commit SHA"},
		{"uppercase head", enqueue("github.com", mqID, strings.ToUpper(mqHead)), "is not a commit SHA"},
		{"head not hex", enqueue("github.com", mqID, strings.Repeat("g", 40)), "is not a commit SHA"},
		{"head with a newline", enqueue("github.com", mqID, mqHead[:39]+"\n"), `\n" is not a commit SHA`},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			e := &mqExec{out: "{}"}
			err := tt.call(e)
			if err == nil {
				t.Fatal("no error")
			}
			if !errors.Is(err, errMergeQueueInput) {
				t.Errorf("error does not wrap errMergeQueueInput: %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to hold %q", err, tt.want)
			}
			mqAssertNoControl(t, "the error", err.Error())
			if len(e.calls) != 0 {
				t.Errorf("gh ran: %+v", e.calls)
			}
		})
	}
	good := []struct {
		name string
		call call
	}{
		{"current node ID", enqueue("github.com", mqID, mqHead)},
		{"legacy node ID", enqueue("github.com", "MDExOlB1bGxSZXF1ZXN0MTIzNA==", mqHead)},
		{"node ID with a leading -", enqueue("github.com", "-PR_kwDO", mqHead)},
		{"node ID of 128", enqueue("github.com", strings.Repeat("A", 128), mqHead)},
		{"head of 64", enqueue("github.com", mqID, strings.Repeat("0f", 32))},
	}
	for _, tt := range good {
		t.Run(tt.name, func(t *testing.T) {
			e := &mqExec{out: mqEnqueueAnswer(mqEntry("QUEUED", 1, "2026-11-20T10:00:00Z"))}
			if err := tt.call(e); err != nil {
				t.Fatal(err)
			}
			if len(e.calls) != 1 {
				t.Errorf("calls = %d, want 1", len(e.calls))
			}
		})
	}
}

// TestMergeQueue_ParsesThePRState covers the reads FR-5, FR-6, FR-8 and FR-9
// decide by: a PR not in the queue, a queued PR, a merged PR (mergeCommit
// set, and "" when it is not a SHA), a PR removed for failed checks with an
// actor, a removal with no reason and no actor, an empty or null errors, and
// times with fractions and an offset.
func TestMergeQueue_ParsesThePRState(t *testing.T) {
	base := "id=" + mqID + " state=%s head=" + mqHead + " base=main merge=%s inQueue=%t"
	tests := []struct {
		name string
		out  func() string
		want string
	}{
		{"open, not in the queue", func() string { return mqPRAnswer(mqPRFields()) }, fmt.Sprintf(base, "OPEN", "", false)},
		{"queued", func() string {
			pr := mqPRFields()
			pr["isInMergeQueue"] = true
			pr["mergeQueueEntry"] = mqEntry("AWAITING_CHECKS", 2, "2026-11-20T10:00:00Z")
			return mqPRAnswer(pr)
		}, fmt.Sprintf(base, "OPEN", "", true) + " entry=AWAITING_CHECKS/2/2026-11-20T10:00:00Z"},
		{"merged", func() string {
			pr := mqPRFields()
			pr["state"] = "MERGED"
			pr["mergeCommit"] = map[string]any{"oid": mqMerge}
			pr["timelineItems"] = mqNodes(mqRemoved("2026-11-20T10:20:00Z", "merged", "github-merge-queue"))
			return mqPRAnswer(pr)
		}, fmt.Sprintf(base, "MERGED", mqMerge, false) + ` removed=2026-11-20T10:20:00Z/"merged"/"github-merge-queue"`},
		{"merged with a mergeCommit that is not a SHA", func() string {
			pr := mqPRFields()
			pr["state"] = "MERGED"
			pr["mergeCommit"] = map[string]any{"oid": "not-a-sha"}
			return mqPRAnswer(pr)
		}, fmt.Sprintf(base, "MERGED", "", false)},
		{"removed for failed checks", func() string {
			pr := mqPRFields()
			pr["timelineItems"] = mqNodes(mqRemoved("2026-11-20T10:05:00Z", "failed_checks", "octocat"))
			return mqPRAnswer(pr)
		}, fmt.Sprintf(base, "OPEN", "", false) + ` removed=2026-11-20T10:05:00Z/"failed_checks"/"octocat"`},
		{"removed with no reason and no actor", func() string {
			pr := mqPRFields()
			pr["timelineItems"] = mqNodes(mqRemoved("2026-11-20T10:05:00Z", nil, nil))
			return mqPRAnswer(pr)
		}, fmt.Sprintf(base, "OPEN", "", false) + ` removed=2026-11-20T10:05:00Z/""/""`},
		{"errors an empty list", func() string {
			return mgJSON(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": mqPRFields()}},
				"errors": []any{}})
		}, fmt.Sprintf(base, "OPEN", "", false)},
		{"errors null", func() string {
			return mgJSON(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": mqPRFields()}},
				"errors": nil})
		}, fmt.Sprintf(base, "OPEN", "", false)},
		{"times with fractions and an offset", func() string {
			pr := mqPRFields()
			pr["isInMergeQueue"] = true
			pr["mergeQueueEntry"] = mqEntry("QUEUED", 1, "2026-11-20T19:00:00.5+09:00")
			pr["timelineItems"] = mqNodes(mqRemoved("2026-11-20T18:00:00-01:00", "manual", "octocat"))
			return mqPRAnswer(pr)
		}, fmt.Sprintf(base, "OPEN", "", true) + ` entry=QUEUED/1/2026-11-20T10:00:00.5Z removed=2026-11-20T19:00:00Z/"manual"/"octocat"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := parseMergeQueuePR(tt.out())
			if err != nil {
				t.Fatal(err)
			}
			if got := mqPRText(pr); got != tt.want {
				t.Errorf("pr =\n  %s\nwant\n  %s", got, tt.want)
			}
		})
	}
}

// TestMergeQueue_ParseRefusesAnswersItCannotRead covers the answers the read
// cannot decide by, each an error: output that is not JSON, errors set (with
// data or not, its control characters escaped), no data, repository or
// pullRequest, a non-null field missing, no timeline, a null or unexpected
// timeline node, and a time that is not RFC 3339.
func TestMergeQueue_ParseRefusesAnswersItCannotRead(t *testing.T) {
	without := func(field string) func() string {
		return func() string {
			pr := mqPRFields()
			delete(pr, field)
			return mqPRAnswer(pr)
		}
	}
	with := func(field string, value any) func() string {
		return func() string {
			pr := mqPRFields()
			pr[field] = value
			return mqPRAnswer(pr)
		}
	}
	fixed := func(s string) func() string { return func() string { return s } }
	tests := []struct {
		name string
		out  func() string
		want string
	}{
		{"not JSON", fixed("not json"), "printed output that is not GraphQL JSON ("},
		{"empty output", fixed(""), "printed output that is not GraphQL JSON ("},
		{"errors and no data", fixed(`{"data":null,"errors":[{"message":"Could not resolve to a PullRequest with the number of 12."}]}`),
			`answered with errors ([{"message":"Could not resolve to a PullRequest with the number of 12."}])`},
		{"errors beside data", func() string {
			return mgJSON(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": mqPRFields()}},
				"errors": []any{map[string]any{"message": "partial"}}})
		}, `answered with errors ([{"message":"partial"}])`},
		{"errors with control characters", func() string {
			return mgJSON(map[string]any{"errors": []any{map[string]any{"message": "bad\x1b[31m\rthing"}}})
		}, `answered with errors ([{"message":"bad\u001b[31m\rthing"}])`},
		{"errors not a list", fixed(`{"errors":{"message":"x"}}`), `answered with errors ({"message":"x"})`},
		{"data null", fixed(`{"data":null}`), "returned no data"},
		{"no data", fixed(`{}`), "returned no data"},
		{"repository null", fixed(`{"data":{"repository":null}}`), "returned no repository"},
		{"pullRequest null", fixed(`{"data":{"repository":{"pullRequest":null}}}`), "returned no repository.pullRequest"},
		{"no id", without("id"), "returned a pullRequest with no id"},
		{"empty id", with("id", ""), "returned a pullRequest with no id"},
		{"no state", without("state"), "returned a pullRequest with no state"},
		{"no headRefOid", without("headRefOid"), "returned a pullRequest with no headRefOid"},
		{"no baseRefName", without("baseRefName"), "returned a pullRequest with no baseRefName"},
		{"no isInMergeQueue", without("isInMergeQueue"), "returned a pullRequest with no isInMergeQueue"},
		{"isInMergeQueue null", with("isInMergeQueue", nil), "returned a pullRequest with no isInMergeQueue"},
		{"no timelineItems", without("timelineItems"), "returned a pullRequest with no timelineItems.nodes"},
		{"timelineItems null", with("timelineItems", nil), "returned a pullRequest with no timelineItems.nodes"},
		{"nodes null", with("timelineItems", map[string]any{"nodes": nil}), "returned a pullRequest with no timelineItems.nodes"},
		{"a null node", with("timelineItems", mqNodes(mqRemoved("2026-11-20T10:00:00Z", "manual", "octocat"), nil)),
			"returned a timeline item that is null"},
		{"an unexpected node", with("timelineItems", mqNodes(map[string]any{"__typename": "MergedEvent", "createdAt": "2026-11-20T10:00:00Z"})),
			`returned a timeline item of the type "MergedEvent", which the query did not ask for`},
		{"a node with no type", with("timelineItems", mqNodes(map[string]any{"createdAt": "2026-11-20T10:00:00Z"})),
			`returned a timeline item of the type "", which the query did not ask for`},
		{"entry time not RFC 3339", with("mergeQueueEntry", mqEntry("QUEUED", 1, "2026-11-20 10:00:00")),
			"returned a time that is not RFC 3339 in mergeQueueEntry.enqueuedAt ("},
		{"entry time empty", with("mergeQueueEntry", map[string]any{"state": "QUEUED", "position": 1}),
			"returned a time that is not RFC 3339 in mergeQueueEntry.enqueuedAt ("},
		{"entry position not an integer", with("mergeQueueEntry", map[string]any{"state": "QUEUED", "position": 1.5,
			"enqueuedAt": "2026-11-20T10:00:00Z"}), "printed output that is not GraphQL JSON ("},
		{"an add event, which the query does not ask for", with("timelineItems", mqNodes(map[string]any{
			"__typename": "AddedToMergeQueueEvent", "createdAt": "2026-11-20T10:00:00Z",
		})), `returned a timeline item of the type "AddedToMergeQueueEvent", which the query did not ask for`},
		{"removal time not RFC 3339", with("timelineItems", mqNodes(mqRemoved("2026-11-20T10:05:00", "failed_checks", "octocat"))),
			"returned a time that is not RFC 3339 in RemovedFromMergeQueueEvent.createdAt ("},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := parseMergeQueuePR(tt.out())
			if err == nil {
				t.Fatalf("no error, pr = %s", mqPRText(pr))
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to hold %q", err, tt.want)
			}
			mqAssertNoControl(t, "the error", err.Error())
		})
	}
}

// TestMergeQueue_ParsesTheEnqueueAnswer covers FR-4 and D5: the entry
// enqueuePullRequest made is read, and an answer with errors (beside an entry
// too), with no data, no enqueuePullRequest or no entry, or with a time that
// is not RFC 3339 is an error, since whether the PR went into the queue is
// then unknown.
func TestMergeQueue_ParsesTheEnqueueAnswer(t *testing.T) {
	entry, err := parseMergeQueueEnqueue(mqEnqueueAnswer(mqEntry("QUEUED", 3, "2026-11-20T10:00:00Z")))
	if err != nil {
		t.Fatal(err)
	}
	if entry.State != "QUEUED" || entry.Position != 3 || !entry.EnqueuedAt.Equal(mqTime(t, "2026-11-20T10:00:00Z")) {
		t.Errorf("entry = %+v, want QUEUED at 3 enqueued at 2026-11-20T10:00:00Z", entry)
	}
	tests := []struct{ name, out, want string }{
		{"errors", `{"data":{"enqueuePullRequest":null},"errors":[{"message":"Pull request is not mergeable"}]}`,
			`answered with errors ([{"message":"Pull request is not mergeable"}])`},
		{"errors beside an entry", mgJSON(map[string]any{
			"data":   map[string]any{"enqueuePullRequest": map[string]any{"mergeQueueEntry": mqEntry("QUEUED", 1, "2026-11-20T10:00:00Z")}},
			"errors": []any{map[string]any{"message": "partial"}},
		}), `answered with errors ([{"message":"partial"}])`},
		{"data null", `{"data":null}`, "returned no enqueuePullRequest.mergeQueueEntry"},
		{"enqueuePullRequest null", `{"data":{"enqueuePullRequest":null}}`, "returned no enqueuePullRequest.mergeQueueEntry"},
		{"entry null", mqEnqueueAnswer(nil), "returned no enqueuePullRequest.mergeQueueEntry"},
		{"entry missing", `{"data":{"enqueuePullRequest":{}}}`, "returned no enqueuePullRequest.mergeQueueEntry"},
		{"time not RFC 3339", mqEnqueueAnswer(mqEntry("QUEUED", 1, "now")),
			"returned a time that is not RFC 3339 in enqueuePullRequest.mergeQueueEntry.enqueuedAt ("},
		{"not JSON", "<html>", "printed output that is not GraphQL JSON ("},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := parseMergeQueueEnqueue(tt.out)
			if err == nil {
				t.Fatalf("no error, entry = %+v", entry)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to hold %q", err, tt.want)
			}
		})
	}
}

// TestMergeQueue_RunnersNameTheCommandWhenGHFailsOrAnswersBadly covers the
// runners' errors after gh ran: a gh that fails is a *mergeQueueGHError that
// names the command, escapes gh's stderr onto one line, does not wrap
// errMergeQueueInput, and unwraps to the exec's error; an answer the parse
// cannot read names the command before the parse's words.
func TestMergeQueue_RunnersNameTheCommandWhenGHFailsOrAnswersBadly(t *testing.T) {
	readCmd := "gh api --hostname github.com graphql (the merge queue state of acme/app#12)"
	enqueueCmd := "gh api --hostname github.com graphql (enqueuePullRequest of " + mqID + " at " + mqHead + ")"
	read := func(e *mqExec) error {
		_, err := readMergeQueuePR(context.Background(), e.run, "/repo", "github.com", mergeRepo{"acme", "app"}, 12)
		return err
	}
	enqueue := func(e *mqExec) error {
		_, err := enqueueMergeQueuePR(context.Background(), e.run, "/repo", "github.com", mqID, mqHead)
		return err
	}
	ghErr := fmt.Errorf("HTTP 502: \x1b[31mbad gateway\x1b[0m\nsecond line: %w", exec.ErrNotFound)
	for _, tt := range []struct {
		name string
		run  func(*mqExec) error
		cmd  string
	}{{"read", read, readCmd}, {"enqueue", enqueue, enqueueCmd}} {
		t.Run(tt.name+" gh fails", func(t *testing.T) {
			e := &mqExec{err: ghErr}
			err := tt.run(e)
			want := tt.cmd + ` failed (HTTP 502: \x1b[31mbad gateway\x1b[0m second line: executable file not found in $PATH)`
			if err == nil || err.Error() != want {
				t.Fatalf("error = %v, want %q", err, want)
			}
			var ghFailed *mergeQueueGHError
			if !errors.As(err, &ghFailed) || !errors.Is(err, exec.ErrNotFound) {
				t.Errorf("error is not a *mergeQueueGHError that unwraps to exec.ErrNotFound: %#v", err)
			}
			if errors.Is(err, errMergeQueueInput) {
				t.Errorf("an error after gh ran wraps errMergeQueueInput: %v", err)
			}
			if len(e.calls) != 1 {
				t.Errorf("calls = %d, want 1", len(e.calls))
			}
		})
	}
	for _, tt := range []struct {
		name string
		run  func(*mqExec) error
		out  string
		want string
	}{
		{"read answers with no data", read, `{"data":null}`, readCmd + " returned no data"},
		{"read answers with errors", read, `{"errors":[{"message":"x"}]}`, readCmd + ` answered with errors ([{"message":"x"}])`},
		{"enqueue answers with no entry", enqueue, mqEnqueueAnswer(nil), enqueueCmd + " returned no enqueuePullRequest.mergeQueueEntry"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := &mqExec{out: tt.out}
			err := tt.run(e)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
			if errors.Is(err, errMergeQueueInput) {
				t.Errorf("an error after gh ran wraps errMergeQueueInput: %v", err)
			}
		})
	}
}

// TestMergeQueue_LatestRemovalAfterCountsTheSameSecond covers the removal
// FR-9 reports (D4: after the entry's enqueuedAt): of several removals, in
// any order, the latest at or after the time; one at the time itself counts,
// since GitHub's times are whole seconds and a removal in the second of the
// enqueue is of that entry, and one a second before it does not. It reads the
// removals from a parsed answer.
func TestMergeQueue_LatestRemovalAfterCountsTheSameSecond(t *testing.T) {
	pr := mqPRFields()
	pr["timelineItems"] = mqNodes(
		mqRemoved("2026-11-20T10:01:00Z", "manual", "alice"),
		mqRemoved("2026-11-20T10:03:00Z", "failed_checks", "github-merge-queue"),
		mqRemoved("2026-11-20T10:02:30Z", "manual", "bob"),
	)
	parsed, err := parseMergeQueuePR(mqPRAnswer(pr))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		after  string
		want   string // the actor of the removal, "" for none
		wantOK bool
	}{
		{"2026-11-20T09:00:00Z", "github-merge-queue", true},
		{"2026-11-20T10:01:00Z", "github-merge-queue", true},
		{"2026-11-20T10:02:59Z", "github-merge-queue", true},
		{"2026-11-20T10:03:00Z", "github-merge-queue", true},
		{"2026-11-20T10:03:01Z", "", false},
		{"2026-11-20T11:00:00Z", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.after, func(t *testing.T) {
			r, ok := latestMergeQueueRemovalAfter(parsed.Removed, mqTime(t, tt.after))
			if ok != tt.wantOK || r.Actor != tt.want {
				t.Errorf("latest after %s = %+v, %t; want actor %q, %t", tt.after, r, ok, tt.want, tt.wantOK)
			}
		})
	}
	if _, ok := latestMergeQueueRemovalAfter(nil, time.Time{}); ok {
		t.Errorf("no removals gave one")
	}
}

// TestMergeQueue_RemovalAfterGateAllowsForClockSkew covers AC16 (FR-5b, D4):
// a removal counts as after the gate when it is strictly after the gate's
// time minus mergeQueueClockSkew (5 minutes), so one before, at, or after the
// gate inside that bound counts (the merge refuses without --requeue), and
// one at the bound or further before does not. A zero gate time counts
// every removal, and of several, the latest that counts is returned.
func TestMergeQueue_RemovalAfterGateAllowsForClockSkew(t *testing.T) {
	if mergeQueueClockSkew != 5*time.Minute {
		t.Fatalf("mergeQueueClockSkew = %s, want 5m0s", mergeQueueClockSkew)
	}
	gate := mqTime(t, "2026-11-20T10:00:00Z")
	tests := []struct {
		name   string
		offset time.Duration // removal time minus gate time
		want   bool
	}{
		{"an hour before", -time.Hour, false},
		{"just outside the skew", -5*time.Minute - time.Second, false},
		{"at the skew", -5 * time.Minute, false},
		{"just inside the skew", -5*time.Minute + time.Second, true},
		{"a second before", -time.Second, true},
		{"at the gate", 0, true},
		{"a second after", time.Second, true},
		{"an hour after", time.Hour, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := gate.Add(tt.offset)
			if got := mergeQueueRemovedAfterGate(at, gate); got != tt.want {
				t.Errorf("mergeQueueRemovedAfterGate(gate%+v) = %t, want %t", tt.offset, got, tt.want)
			}
			r, ok := mergeQueueRemovalAfterGate([]mergeQueueRemoval{{At: at, Reason: "failed_checks"}}, gate)
			if ok != tt.want || ok && !r.At.Equal(at) {
				t.Errorf("mergeQueueRemovalAfterGate(gate%+v) = %+v, %t; want %t", tt.offset, r, ok, tt.want)
			}
		})
	}
	if !mergeQueueRemovedAfterGate(gate.Add(-24*time.Hour), time.Time{}) {
		t.Errorf("a zero gate time did not count a removal")
	}
	removals := []mergeQueueRemoval{
		{At: gate.Add(-time.Hour), Actor: "old"},
		{At: gate.Add(2 * time.Minute), Actor: "latest"},
		{At: gate.Add(-time.Minute), Actor: "inside"},
	}
	if r, ok := mergeQueueRemovalAfterGate(removals, gate); !ok || r.Actor != "latest" {
		t.Errorf("of several = %+v, %t; want the latest", r, ok)
	}
	if r, ok := mergeQueueRemovalAfterGate(removals[:1], gate); ok {
		t.Errorf("a removal an hour before the gate counted: %+v", r)
	}
}

// TestMergeQueue_RemovalTextEscapesControlCharacters covers the text of a
// removal: its time in UTC, the actor and the reason, each control character
// of the last two written as its Go escape, so none reaches a terminal raw,
// and words for a null actor or reason. A reason read from an answer is
// escaped the same way.
func TestMergeQueue_RemovalTextEscapesControlCharacters(t *testing.T) {
	at := mqTime(t, "2026-11-20T19:00:00+09:00")
	tests := []struct {
		name string
		r    mergeQueueRemoval
		want string
	}{
		{"plain", mergeQueueRemoval{At: at, Reason: "failed_checks", Actor: "octocat"},
			"removed from the merge queue at 2026-11-20T10:00:00Z by octocat (reason: failed_checks)"},
		{"no reason and no actor", mergeQueueRemoval{At: at},
			"removed from the merge queue at 2026-11-20T10:00:00Z by an actor GitHub does not name (reason: none given)"},
		{"control characters", mergeQueueRemoval{At: at, Reason: "failed\x1b[31m\r\nchecks\x07", Actor: "bot\x1b]0;x\x07\t"},
			`removed from the merge queue at 2026-11-20T10:00:00Z by bot\x1b]0;x\a\t (reason: failed\x1b[31m\r\nchecks\a)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeQueueRemovalText(tt.r)
			if got != tt.want {
				t.Errorf("text =\n  %s\nwant\n  %s", got, tt.want)
			}
			mqAssertNoControl(t, "the text", got)
		})
	}
	pr := mqPRFields()
	pr["timelineItems"] = mqNodes(mqRemoved("2026-11-20T10:05:00Z", "manual\x1b[2J\xc2\x85", "eve\xc2\x9b"))
	parsed, err := parseMergeQueuePR(mqPRAnswer(pr))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Removed) != 1 || !strings.Contains(parsed.Removed[0].Reason, "\x1b[2J") {
		t.Fatalf("removals = %+v, want one whose reason holds ESC as it is", parsed.Removed)
	}
	text := mergeQueueRemovalText(parsed.Removed[0])
	mqAssertNoControl(t, "the text of a parsed removal", text)
	if !strings.Contains(text, `(reason: manual\x1b[2J`) {
		t.Errorf("text = %q, want the reason's ESC escaped", text)
	}
}
