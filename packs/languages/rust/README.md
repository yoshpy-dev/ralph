# Rust pack

Default verification order:
- cargo fmt --check
- cargo clippy --all-targets --all-features -- -D warnings
- cargo test --all-features

Customize this pack if your workspace needs:
- package-level selection
- feature-flag subsets
- integration tests or nextest
- rustfmt or clippy config overrides
