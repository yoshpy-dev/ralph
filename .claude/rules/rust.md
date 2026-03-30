---
paths:
  - "**/*.rs"
  - "Cargo.toml"
---
# Rust rules

- Keep ownership and error boundaries explicit.
- Prefer narrow traits and well-named domain modules over large utility blobs.
- Use `cargo fmt --check`, `cargo clippy`, and `cargo test` before completion when the project supports them.
- Make async, IO, and domain logic boundaries obvious in the module layout.
- If architecture rules matter, encode them in compile-time checks, tests, or linting where possible.
