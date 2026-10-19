# Live check: the bounded director wait under the Monitor tool (stage 8b-2c)

Plan: `docs/plans/active/2026-10-19-org-director-wait-bound.md` (D6, AC6).
Run on 2026-10-19 with the `ralph` binary built from the task branch at
5aea657f (S1: the `/director` skill's bounded Claude Code wait and its test;
no Go code outside tests changed, so the binary behaves as main's).

Values left out: the herdr pane id and claude's session ids. `$HOME` is `~`.
The Monitor's task ids of (e) to (g) were not recorded, though plan D6
named them; the mark's `wait_id` (Mark log) ties each wait to its end.

## Setup

- macOS, Claude Code 2.1.291, model sonnet (the session showed effort low),
  `--allowedTools Bash Monitor ToolSearch Read` (the user's settings put the
  session in bypass permissions mode)
- An interactive claude in a pane of an isolated, headless herdr server
  (XDG_CONFIG_HOME, XDG_STATE_HOME, and HERDR_SOCKET_PATH under
  `/private/tmp/xbw`; the real HOME for claude's auth). No herdr server of
  the user was touched
- An isolated ledger, `--state-dir /tmp/xbw/state`, run from `/tmp/xbw`,
  which has no `ralph.toml`, so every setting is the default. From a shell
  outside the pane: `ralph org director attach --driver claude` (d1,
  `USER=tester`, `HERDR_PANE_ID` unset) and `ralph org watch --all --detach`,
  so `ralph org escalate` only records an item for the director
- `osascript` on `PATH` was a shim that logs and exits 0. Its log stayed
  empty: nothing was sent to a human
- A loop in the outer shell read `director-wait.json` once a second and
  logged each change (below)
- Signals went only to processes this check started: the waits in the pane
  and the claude process of (d)

## Results

| Check | What was done | What happened |
|-------|---------------|---------------|
| (b) | the old form: the session ran `ralph org wait --inbox --director d1 --timeout-ms 0 2>&1` under Monitor with `timeout_ms` 15000 | the Monitor expired after 15 s ("Monitor expired after 15s with no …"); the wait got SIGTERM; the mark: `end` signal, `detail` "terminated signal received" (started 12:10:29Z, ended 12:10:44Z); `ralph org director status` printed `wait: … ended at 2026-10-19T12:10:44Z (signal)` |
| (c) | a wait with `--timeout-ms 600000` under Monitor (`timeout_ms` 900000), then `/exit` with its default "Exit and stop tasks" | the exit dialog listed "monitor · c wait"; the wait and claude were gone within 1 s; the mark: `end` signal (12:11:43Z) |
| (d) | a wait with `--timeout-ms 40000` under Monitor (`timeout_ms` 60000), then `kill -KILL` of the claude process | the wait kept running and its `zsh -c` shell was reparented to pid 1; the wait ended on its own 40 s after it started, at 12:12:54Z, with the mark `end` timeout, `detail` "no open inbox item within 40000 ms"; the shell ended with it |
| (a), (e), (f) | a new session read the test instructions (below), which give the skill's section with the test numbers: the wait `--timeout-ms 20000`, the Monitor `timeout_ms` 60000, and, for its first wait only, `timeout_ms` 10000 | (f) the first wait's Monitor expired after 10 s and the wait ended by a signal at 12:13:51Z; the session ran the wait again 3 s later. (a)(e) each later wait ended on its own 20 s after it started (`end` timeout), and the session ran it again within 2 to 3 s, three times in a row (12:14:16Z, 12:14:38Z, 12:15:01Z), with no human input. The item e1, escalated at 12:15:12Z, ended the running wait with `end` items at 12:15:13Z; the session ran `ralph org inbox ack e1` (acked at 12:15:15Z) and waited again at 12:15:18Z. That wait timed out at 12:15:38Z and the session ran it again at 12:15:40Z; after that wait's timeout (12:16:00Z) the session stopped on its own (Notes) |
| (g) | in a second session with the same instructions, after two timeouts it ran again, `kill -TERM` of its running wait 6 s after it started | the mark: `end` signal (12:18:11Z); the session wrote that the wait ended with 143 (SIGTERM), that no "Monitor expired" notice came before it and there was no timeout text, that the section counts this as a failure, so it does not run the wait again; that d1 stays registered since it did not detach; and that it waits for the human. No new wait started in the 45 s that followed |

Notes on the runs:

- After the fifth timeout of the first session (the wait that ended at
  12:16:00Z), that session stopped on its own and wrote that the steps it
  had been given had all happened, so it stopped there. The instructions
  called it a test director and listed the steps, which it read as a check
  with an end. The second session got one more line ("This role has no end
  of its own: keep waiting … until a human writes stop") and did not stop
  on its own before (g). The real skill does not frame the director as a
  test; whether a director running the real `/director` skill keeps waiting
  for hours was not checked.
- The first attempt at (g), in the first session, missed: the wait had just
  ended by its timeout when the outer shell read the mark, so no signal was
  sent.
- In (d), the orphaned wait wrote its time-out error to a stderr socket whose
  other end was gone; it still wrote its mark and ended.

## Mark log

Each line: the time the outer loop saw the change, then the mark's `wait_id`
(the wait's pid and its start time in Unix nanoseconds), `started_at`,
`end`, `ended_at`, and `detail`.

```text
12:13:19 5522-1792411934562320000 2026-10-19T12:12:14Z timeout 2026-10-19T12:12:54Z no open inbox item within 40000 ms
12:13:42 12311-1792412021821652000 2026-10-19T12:13:41Z
12:13:52 12311-1792412021821652000 2026-10-19T12:13:41Z signal 2026-10-19T12:13:51Z terminated signal received
12:13:54 13178-1792412034285952000 2026-10-19T12:13:54Z
12:14:15 13178-1792412034285952000 2026-10-19T12:13:54Z timeout 2026-10-19T12:14:14Z no open inbox item within 20000 ms
12:14:17 14527-1792412056689627000 2026-10-19T12:14:16Z
12:14:37 14527-1792412056689627000 2026-10-19T12:14:16Z timeout 2026-10-19T12:14:36Z no open inbox item within 20000 ms
12:14:39 15932-1792412078703563000 2026-10-19T12:14:38Z
12:14:59 15932-1792412078703563000 2026-10-19T12:14:38Z timeout 2026-10-19T12:14:58Z no open inbox item within 20000 ms
12:15:02 17516-1792412101010085000 2026-10-19T12:15:01Z
12:15:13 17516-1792412101010085000 2026-10-19T12:15:01Z items 2026-10-19T12:15:13Z 1 open inbox item
12:15:18 18886-1792412118120571000 2026-10-19T12:15:18Z
12:15:39 18886-1792412118120571000 2026-10-19T12:15:18Z timeout 2026-10-19T12:15:38Z no open inbox item within 20000 ms
12:15:41 20772-1792412140462751000 2026-10-19T12:15:40Z
12:16:01 20772-1792412140462751000 2026-10-19T12:15:40Z timeout 2026-10-19T12:16:00Z no open inbox item within 20000 ms
12:17:20 27050-1792412239951014000 2026-10-19T12:17:19Z
12:17:40 27050-1792412239951014000 2026-10-19T12:17:19Z timeout 2026-10-19T12:17:39Z no open inbox item within 20000 ms
12:17:43 28266-1792412262723751000 2026-10-19T12:17:42Z
12:18:03 28266-1792412262723751000 2026-10-19T12:17:42Z timeout 2026-10-19T12:18:02Z no open inbox item within 20000 ms
12:18:06 29863-1792412285316981000 2026-10-19T12:18:05Z
12:18:11 29863-1792412285316981000 2026-10-19T12:18:05Z signal 2026-10-19T12:18:11Z terminated signal received
```

The lines of (b) and (c) came before the loop started; they are in the
table above. The first line is (d)'s wait. The lines from 12:13:42 to
12:16:01 are the first session of (a)(e)(f), and those from 12:17:20 the
second session, ending with (g).

## Test instructions

The sessions of (a), (e), (f), and (g) read a file that started with these
lines (the second session's two added lines are marked), followed by the
section `### wait で待つ` and the exit-code handling after it, copied from
`.claude/skills/director/SKILL.md` at 5aea657f:

```text
# Test director instructions

You are acting as a test director in an isolated test ledger. Your registration id is d1.
Follow the section below (copied from the /director skill: "wait で待つ" and the exit-code handling) for waiting on the inbox, with these test substitutions:

- Use the binary /tmp/xbw/ralph instead of ralph, and add --state-dir /tmp/xbw/state to every ralph command.
- The wait's bound is --timeout-ms 20000 instead of 1500000.
- The Monitor's timeout_ms is 60000 instead of 1800000, everywhere the section says 1800000.
- For the very first wait only, pass the Monitor a timeout_ms of 10000 (this simulates forgetting the right value). Every later wait follows the section with the test numbers above.
- When a wait returns open items (exit code 0), handle each item only by running /tmp/xbw/ralph org inbox ack <ID> --state-dir /tmp/xbw/state for each ID it printed, then wait again. Do not run any other ralph command.
- "Tell the human" means: write one short line in this chat.
- After each wait ends, write one short line: how it ended and what you do next.
- Never run anything other than the wait (through the Monitor tool) and the ack command.
- (second session) This role has no end of its own: keep waiting, as the section says, until a human writes stop in this chat. Do not stop because the steps above have each happened once.
- (second session) The first-wait timeout_ms of 10000 applied to an earlier session; in this session every wait uses 60000.
```

The prompt of each session was: "Read /tmp/xbw/proj/director-test.md with
the Read tool and act as it says, starting now. Keep going on your own: no
human will type anything in this session."

## Not checked here

- The real numbers (a 25-minute wait under a 30-minute Monitor); the check
  used 20 s and 60 s
- A director that runs the whole `/director` skill (attach, the watch, the
  item handling with send and resolve) for hours
- codex, and a headless director started with `ralph org director start`
- Bash `run_in_background` as the wait's runner
- The skill's text after 5aea657f: the sessions read the section as of
  5aea657f. After the check, dc98d5ec narrowed the sentence on how a
  Monitor wait stops and tied the signal exception to that wait's own
  Monitor, and the commit that added this line made the exception
  decidable when the director sees the signal end before the Monitor's
  notice. These later texts were not run again. For (b) and (f) the table
  records that the Monitor expired and the wait ended by a signal, not
  whether the session saw the notice before the end; a signal end seen
  before the Monitor's notice was not checked

Left on the machine: claude's own records of these sessions
(`~/.claude/projects/-private-tmp-xbw-proj/`, the trust entry for
`/private/tmp/xbw/proj`). `/private/tmp/xbw` was removed after the run.
