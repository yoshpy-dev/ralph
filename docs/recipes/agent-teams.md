# Agent teams

Agent teams are a later-stage optimization.

Use them when:
- parallel research adds real value
- cross-layer work can be split cleanly
- competing debugging hypotheses need parallel testing
- reviewers benefit from independent contexts

Avoid them when:
- work is mostly sequential
- files overlap heavily
- the coordination cost exceeds the benefit

Suggested enablement flow:
1. Get the single-session loop working first
2. Use subagents next
3. Add agent teams only for tasks that truly benefit from direct worker-to-worker communication

Example environment setting:

```json
{
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  }
}
```

Keep this experimental until it proves its value in your repo.
