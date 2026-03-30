---
paths:
  - "**/*.ts"
  - "**/*.tsx"
  - "**/*.js"
  - "**/*.jsx"
---
# TypeScript and JavaScript rules

- Prefer explicit exports and stable file naming over default-export-heavy layouts.
- Keep runtime validation close to external boundaries.
- Use the package manager already present in the repo; do not switch package managers casually.
- If scripts exist, use `lint`, `typecheck`, and `test` through the package manager before claiming completion.
- Keep generated types, schemas, and API contracts synchronized.
