---
paths:
  - "**/*.py"
---
# Python rules

- Prefer clear module boundaries and explicit imports.
- Keep side effects at edges; keep pure logic easy to unit test.
- Use project-local tooling when available: `ruff`, `pytest`, `mypy`, task runners, or scripts.
- Avoid hiding environment-dependent behavior; surface it in config and tests.
- If a command is required to validate work, capture it in the plan and verify report.
