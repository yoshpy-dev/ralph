# Architecture rules

- Prefer grep-able names over clever names.
- Keep module boundaries explicit and easy to infer from file paths.
- Favor feature-oriented structure over deep horizontal layering when possible.
- Make public contracts obvious: types, schemas, interfaces, CLI flags, API shapes.
- If a rule matters enough to repeat in reviews, consider promoting it into a test, linter, hook, or CI check.
- Avoid opaque helper layers that hide control flow from search-based code reading.
- Small, boring, well-named abstractions are usually better for agents than magical ones.
