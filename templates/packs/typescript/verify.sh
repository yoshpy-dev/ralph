#!/usr/bin/env sh
set -eu

if [ ! -f package.json ]; then
  echo "Skipping TypeScript verifier: package.json not found."
  exit 0
fi

has_script() {
  grep -q ""$1"[[:space:]]*:" package.json
}

pm="npm"
if [ -f pnpm-lock.yaml ] && command -v pnpm >/dev/null 2>&1; then
  pm="pnpm"
elif [ -f yarn.lock ] && command -v yarn >/dev/null 2>&1; then
  pm="yarn"
elif ! command -v npm >/dev/null 2>&1; then
  echo "No supported package manager found for TypeScript verification."
  exit 1
fi

run_script() {
  script="$1"
  case "$pm" in
    npm)
      npm run "$script" --if-present
      ;;
    pnpm)
      if has_script "$script"; then
        pnpm run "$script"
      else
        echo "Skipping $script: script not defined."
      fi
      ;;
    yarn)
      if has_script "$script"; then
        yarn "$script"
      else
        echo "Skipping $script: script not defined."
      fi
      ;;
  esac
}

run_script lint
run_script typecheck
run_script test
