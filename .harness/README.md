# .harness

Runtime-only harness state lives here.

Suggested usage:
- `.harness/state/` for transient markers, summaries, and counters
- `.harness/logs/` for local logs
- keep versioned plans, reports, and long-lived knowledge in `docs/`

Canonical truth should remain in version-controlled docs and code, not in transient state.
