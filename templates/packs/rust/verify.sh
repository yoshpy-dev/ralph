#!/usr/bin/env sh
set -eu

if [ ! -f Cargo.toml ]; then
  echo "Skipping Rust verifier: Cargo.toml not found."
  exit 0
fi

if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo is required for Rust verification."
  exit 1
fi

cargo fmt --all --check
cargo clippy --all-targets --all-features -- -D warnings
cargo test --all-features
