# Testing rules

- Every meaningful change should update or strengthen verification.
- Prefer tests close to the code they validate when the language and framework support it.
- Treat tests as a reward signal for the agent. Missing tests reduce reliable automation.
- Add at least one edge case when changing logic, parsing, branching, or state transitions.
- Never make tests pass by weakening the intent without stating that change in the plan and review report.
- Keep test names specific enough that failures explain intent, not only mechanics.
